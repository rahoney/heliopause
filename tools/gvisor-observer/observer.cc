// HAA's gVisor observer is compiled inside the exact pinned gVisor source tree.
// It consumes the upstream remote sink protocol and emits only bounded kinds to
// HAA's trusted datagram boundary; it never logs trace payloads.
#include <arpa/inet.h>
#include <err.h>
#include <signal.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>
#include <map>

#include "pkg/sentry/seccheck/points/common.pb.h"
#include "pkg/sentry/seccheck/points/container.pb.h"
#include "pkg/sentry/seccheck/points/sentry.pb.h"
#include "pkg/sentry/seccheck/points/syscall.pb.h"

namespace {
const char* gControlPath = nullptr;

void CleanupControlSocket(int) {
  if (gControlPath != nullptr) unlink(gControlPath);
  _exit(0);
}
#ifndef HAA_GVISOR_COMMIT
#error "HAA_GVISOR_COMMIT must be set by the pinned observer build"
#endif
constexpr uint32_t kProtocolVersion = 1;
constexpr size_t kMaxEventSize = 1024 * 1024;
constexpr size_t kMaxNormalizedRecordsPerConnection = 10000;
constexpr char kProfileNPM[] = "npm-lifecycle";
constexpr char kProfilePyPI[] = "pypi-wheel";
constexpr char kProfileGitHub[] = "github-elf";
#pragma pack(push, 1)
struct Header { uint16_t header_size; uint16_t message_type; uint32_t dropped_count; };
#pragma pack(pop)

bool ValidContainerID(const std::string& container_id) {
  if (container_id.size() < 12 || container_id.size() > 64) return false;
  for (const char character : container_id) if ((character < 'a' || character > 'f') && (character < '0' || character > '9')) return false;
  return true;
}

bool Send(int output, const std::string& container_id, const char* kind) {
  if (!ValidContainerID(container_id)) return false;
  const std::string message = "{\"container_id\":\"" + container_id + "\",\"kind\":\"" + kind + "\"}";
  return send(output, message.data(), message.size(), 0) == static_cast<ssize_t>(message.size());
}

bool HasPrefix(const std::string& value, const char* prefix) {
  return value.compare(0, strlen(prefix), prefix) == 0;
}

bool IsHoneytoken(const std::string& path) {
  return path == "/tmp/.haa-honeytoken" || path == "/work/.haa-honeytoken";
}

bool IsClearlyOutsideWorkspace(const std::string& path) {
  return HasPrefix(path, "/root/") || HasPrefix(path, "/home/") ||
      HasPrefix(path, "/run/secrets/") || path == "/root" || path == "/home";
}

bool IsExpectedProcess(const std::string& path, const char* profile) {
  if (strcmp(profile, kProfileNPM) == 0) {
    return path == "/bin/sh" || path == "/usr/local/bin/node" || path == "/usr/local/bin/npm";
  }
  if (strcmp(profile, kProfilePyPI) == 0) {
    return path == "/bin/sh" || path == "/usr/local/bin/python" || path == "/usr/local/bin/pip";
  }
  return path == "/bin/sh" || path == "/work/artifact";
}

const char* ProfileKind(const std::string& path, const char* profile) {
  if (IsHoneytoken(path)) return "honeytoken-access";
  if (IsClearlyOutsideWorkspace(path)) return "filesystem-outside-workspace";
  if (HasPrefix(path, "/tmp/") || HasPrefix(path, "/work/")) return "filesystem-workspace-access";
  return "filesystem-open";
}

bool ParseControlRecord(const char* payload, size_t size, std::map<std::string, std::string>* profiles) {
  std::string body(payload, size);
  const std::string id_key = "\"container_id\":\"";
  const std::string profile_key = "\"profile\":\"";
  const size_t id_start = body.find(id_key);
  const size_t profile_start = body.find(profile_key);
  if (id_start == std::string::npos || profile_start == std::string::npos) return false;
  const size_t id_end = body.find('"', id_start + id_key.size());
  const size_t profile_end = body.find('"', profile_start + profile_key.size());
  if (id_end == std::string::npos || profile_end == std::string::npos) return false;
  const std::string id = body.substr(id_start + id_key.size(), id_end - id_start - id_key.size());
  const std::string profile = body.substr(profile_start + profile_key.size(), profile_end - profile_start - profile_key.size());
  if (!ValidContainerID(id) || (profile != kProfileNPM && profile != kProfilePyPI && profile != kProfileGitHub)) return false;
  if (profiles->find(id) != profiles->end()) return false;
  (*profiles)[id] = profile;
  return true;
}

template <typename Message>
bool ParseAndSend(const char* payload, size_t payload_size, int output, const char* kind, std::string* container_id, const char* profile) {
  (void)profile;
  Message message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate)) return false;
  if (container_id->empty()) {
    *container_id = candidate;
  } else if (*container_id != candidate) {
    return false;
  }
  return Send(output, *container_id, kind);
}

template <typename Message>
bool ParseProcessAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile) {
  if (profile == nullptr) return false;
  Message message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate) || (!container_id->empty() && *container_id != candidate)) return false;
  if (container_id->empty()) *container_id = candidate;
  const std::string path = message.pathname();
  return Send(output, *container_id, IsExpectedProcess(path, profile) ? "process-exec-expected" : "process-exec-unexpected");
}

bool ParseSentryProcessAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile) {
  if (profile == nullptr) return false;
  gvisor::sentry::ExecveInfo message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate) || (!container_id->empty() && *container_id != candidate)) return false;
  if (container_id->empty()) *container_id = candidate;
  return Send(output, *container_id, IsExpectedProcess(message.binary_path(), profile) ? "process-exec-expected" : "process-exec-unexpected");
}

bool ParseOpenAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile) {
  if (profile == nullptr) return false;
  gvisor::syscall::Open message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate) || (!container_id->empty() && *container_id != candidate)) return false;
  if (container_id->empty()) *container_id = candidate;
  return Send(output, *container_id, ProfileKind(message.pathname(), profile));
}

bool Handle(const Header& header, const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile) {
  if (header.dropped_count != 0) return false;
  switch (static_cast<gvisor::common::MessageType>(header.message_type)) {
    case gvisor::common::MESSAGE_CONTAINER_START: return ParseAndSend<gvisor::container::Start>(payload, payload_size, output, "container-start", container_id, profile);
    case gvisor::common::MESSAGE_SENTRY_CLONE: return ParseAndSend<gvisor::sentry::CloneInfo>(payload, payload_size, output, "process-clone", container_id, profile);
    case gvisor::common::MESSAGE_SENTRY_EXEC: return ParseSentryProcessAndSend(payload, payload_size, output, container_id, profile);
    // pathname, argv and envv are parsed by protobuf but are deliberately never
    // copied to the HAA envelope. M11-003 supplies the trusted profile required
    // to classify this bounded process fact as expected or unexpected.
    case gvisor::common::MESSAGE_SYSCALL_EXECVE: return ParseProcessAndSend<gvisor::syscall::Execve>(payload, payload_size, output, container_id, profile);
    case gvisor::common::MESSAGE_SYSCALL_OPEN: return ParseOpenAndSend(payload, payload_size, output, container_id, profile);
    case gvisor::common::MESSAGE_SYSCALL_CONNECT: return ParseAndSend<gvisor::syscall::Connect>(payload, payload_size, output, "network-attempt", container_id, profile);
    case gvisor::common::MESSAGE_SYSCALL_SOCKET: return ParseAndSend<gvisor::syscall::Socket>(payload, payload_size, output, "network-attempt", container_id, profile);
    default: return false;
  }
}

int ConnectDatagram(const char* path) {
  int fd = socket(AF_UNIX, SOCK_DGRAM, 0);
  if (fd < 0) err(1, "socket output");
  sockaddr_un address{}; address.sun_family = AF_UNIX;
  if (strlen(path) >= sizeof(address.sun_path)) errx(1, "output endpoint too long");
  strcpy(address.sun_path, path);
  if (connect(fd, reinterpret_cast<sockaddr*>(&address), sizeof(address)) < 0) err(1, "connect output");
  return fd;
}
}

int main(int argc, char** argv) {
  if (argc == 2 && strcmp(argv[1], "--identity") == 0) {
    printf("gvisor-commit=%s\n", HAA_GVISOR_COMMIT);
    return 0;
  }
  if (argc != 4 && argc != 5) errx(2, "usage: haa_gvisor_observer REMOTE_SEQPACKET_SOCKET HAA_OUTPUT_DGRAM_SOCKET HAA_CONTROL_DGRAM_SOCKET [--ready-fd=FD]");
  int ready_fd = -1;
  if (argc == 5) {
    if (strncmp(argv[4], "--ready-fd=", 11) != 0) errx(2, "invalid readiness option");
    char* end = nullptr;
    const long parsed_ready_fd = strtol(argv[4] + 11, &end, 10);
    if (end == argv[4] + 11 || *end != '\0' || parsed_ready_fd < 0 || parsed_ready_fd > INT32_MAX) errx(2, "invalid readiness file descriptor");
    ready_fd = static_cast<int>(parsed_ready_fd);
  }
  int listener = socket(AF_UNIX, SOCK_SEQPACKET, 0);
  if (listener < 0) err(1, "socket remote");
  sockaddr_un address{}; address.sun_family = AF_UNIX;
  if (strlen(argv[1]) >= sizeof(address.sun_path)) errx(1, "remote endpoint too long");
  strcpy(address.sun_path, argv[1]);
  if (bind(listener, reinterpret_cast<sockaddr*>(&address), sizeof(address)) < 0) err(1, "bind remote");
  if (listen(listener, 16) < 0) err(1, "listen remote");
  int control = socket(AF_UNIX, SOCK_DGRAM, 0);
  if (control < 0) err(1, "socket control");
  sockaddr_un control_address{}; control_address.sun_family = AF_UNIX;
  if (strlen(argv[3]) >= sizeof(control_address.sun_path)) errx(1, "control endpoint too long");
  strcpy(control_address.sun_path, argv[3]);
  gControlPath = argv[3];
  if (bind(control, reinterpret_cast<sockaddr*>(&control_address), sizeof(control_address)) < 0) err(1, "bind control");
  signal(SIGTERM, CleanupControlSocket);
  signal(SIGINT, CleanupControlSocket);
  std::map<std::string, std::string> profiles;
  const int output = ConnectDatagram(argv[2]);
  if (ready_fd >= 0) {
    const char ready = 'R';
    if (write(ready_fd, &ready, 1) != 1) err(1, "signal readiness");
    close(ready_fd);
  }
  for (;;) {
    char control_message[512];
    ssize_t control_size;
    while ((control_size = recv(control, control_message, sizeof(control_message), MSG_DONTWAIT)) > 0) {
      if (!ParseControlRecord(control_message, static_cast<size_t>(control_size), &profiles)) errx(1, "invalid observer profile registration");
    }
    int client = accept(listener, nullptr, nullptr); if (client < 0) err(1, "accept remote");
    char handshake[1024]; ssize_t size = recv(client, handshake, sizeof(handshake), 0);
    gvisor::common::Handshake incoming;
    if (size <= 0 || !incoming.ParseFromArray(handshake, size) || incoming.version() != kProtocolVersion) { close(client); continue; }
    gvisor::common::Handshake outgoing; outgoing.set_version(kProtocolVersion); std::string encoded; outgoing.SerializeToString(&encoded);
    if (send(client, encoded.data(), encoded.size(), 0) != static_cast<ssize_t>(encoded.size())) { close(client); continue; }
    std::string container_id; char event[kMaxEventSize]; bool fault = false;
    const char* profile = nullptr;
    size_t normalized_records = 0;
    while ((size = recv(client, event, sizeof(event), MSG_TRUNC)) > 0) {
      if (static_cast<size_t>(size) > sizeof(event)) { fault = true; break; }
      if (static_cast<size_t>(size) < sizeof(Header)) { fault = true; break; }
      Header header{}; memcpy(&header, event, sizeof(header));
      if (profile == nullptr && !container_id.empty()) {
        auto profile_it = profiles.find(container_id);
        if (profile_it == profiles.end()) { fault = true; break; }
        profile = profile_it->second.c_str();
      }
      if (normalized_records == kMaxNormalizedRecordsPerConnection || header.header_size < sizeof(Header) || header.header_size > static_cast<uint16_t>(size) || !Handle(header, event + header.header_size, size - header.header_size, output, &container_id, profile)) { fault = true; break; }
      ++normalized_records;
    }
    if (size < 0) fault = true;
    if (!container_id.empty()) {
      Send(output, container_id, fault ? "stream-fault" : "stream-end");
      profiles.erase(container_id);
    }
    close(client);
  }
}
