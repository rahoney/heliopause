// HAA's gVisor observer is compiled inside the exact pinned gVisor source tree.
// It consumes the upstream remote sink protocol and emits only bounded kinds to
// HAA's trusted datagram boundary; it never logs trace payloads.
#include <arpa/inet.h>
#include <err.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>

#include "pkg/sentry/seccheck/points/common.pb.h"
#include "pkg/sentry/seccheck/points/container.pb.h"
#include "pkg/sentry/seccheck/points/sentry.pb.h"
#include "pkg/sentry/seccheck/points/syscall.pb.h"

namespace {
#ifndef HAA_GVISOR_COMMIT
#error "HAA_GVISOR_COMMIT must be set by the pinned observer build"
#endif
constexpr uint32_t kProtocolVersion = 1;
constexpr size_t kMaxEventSize = 1024 * 1024;
#pragma pack(push, 1)
struct Header { uint16_t header_size; uint16_t message_type; uint32_t dropped_count; };
#pragma pack(pop)

bool Send(int output, const std::string& container_id, const char* kind) {
  if (container_id.size() < 12 || container_id.size() > 64) return false;
  for (const char character : container_id) if ((character < 'a' || character > 'f') && (character < '0' || character > '9')) return false;
  const std::string message = "{\"container_id\":\"" + container_id + "\",\"kind\":\"" + kind + "\"}";
  return send(output, message.data(), message.size(), 0) == static_cast<ssize_t>(message.size());
}

template <typename Message>
bool ParseAndSend(const char* payload, size_t payload_size, int output, const char* kind, std::string* container_id) {
  Message message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  *container_id = message.context_data().container_id();
  return Send(output, *container_id, kind);
}

bool Handle(const Header& header, const char* payload, size_t payload_size, int output, std::string* container_id) {
  if (header.dropped_count != 0) return false;
  switch (static_cast<gvisor::common::MessageType>(header.message_type)) {
    case gvisor::common::MESSAGE_CONTAINER_START: return ParseAndSend<gvisor::container::Start>(payload, payload_size, output, "container-start", container_id);
    case gvisor::common::MESSAGE_SENTRY_CLONE: return ParseAndSend<gvisor::sentry::CloneInfo>(payload, payload_size, output, "process-clone", container_id);
    case gvisor::common::MESSAGE_SENTRY_EXEC: return ParseAndSend<gvisor::sentry::ExecveInfo>(payload, payload_size, output, "process-exec", container_id);
    case gvisor::common::MESSAGE_SYSCALL_OPEN: return ParseAndSend<gvisor::syscall::Open>(payload, payload_size, output, "filesystem-open", container_id);
    case gvisor::common::MESSAGE_SYSCALL_CONNECT: return ParseAndSend<gvisor::syscall::Connect>(payload, payload_size, output, "network-attempt", container_id);
    case gvisor::common::MESSAGE_SYSCALL_SOCKET: return ParseAndSend<gvisor::syscall::Socket>(payload, payload_size, output, "network-attempt", container_id);
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
  if (argc != 3 && argc != 4) errx(2, "usage: haa_gvisor_observer REMOTE_SEQPACKET_SOCKET HAA_OUTPUT_DGRAM_SOCKET [--ready-fd=FD]");
  int ready_fd = -1;
  if (argc == 4) {
    if (strncmp(argv[3], "--ready-fd=", 11) != 0) errx(2, "invalid readiness option");
    char* end = nullptr;
    const long parsed_ready_fd = strtol(argv[3] + 11, &end, 10);
    if (end == argv[3] + 11 || *end != '\0' || parsed_ready_fd < 0 || parsed_ready_fd > INT32_MAX) errx(2, "invalid readiness file descriptor");
    ready_fd = static_cast<int>(parsed_ready_fd);
  }
  int listener = socket(AF_UNIX, SOCK_SEQPACKET, 0);
  if (listener < 0) err(1, "socket remote");
  sockaddr_un address{}; address.sun_family = AF_UNIX;
  if (strlen(argv[1]) >= sizeof(address.sun_path)) errx(1, "remote endpoint too long");
  strcpy(address.sun_path, argv[1]);
  if (bind(listener, reinterpret_cast<sockaddr*>(&address), sizeof(address)) < 0) err(1, "bind remote");
  if (listen(listener, 16) < 0) err(1, "listen remote");
  const int output = ConnectDatagram(argv[2]);
  if (ready_fd >= 0) {
    const char ready = 'R';
    if (write(ready_fd, &ready, 1) != 1) err(1, "signal readiness");
    close(ready_fd);
  }
  for (;;) {
    int client = accept(listener, nullptr, nullptr); if (client < 0) err(1, "accept remote");
    char handshake[1024]; ssize_t size = recv(client, handshake, sizeof(handshake), 0);
    gvisor::common::Handshake incoming;
    if (size <= 0 || !incoming.ParseFromArray(handshake, size) || incoming.version() != kProtocolVersion) { close(client); continue; }
    gvisor::common::Handshake outgoing; outgoing.set_version(kProtocolVersion); std::string encoded; outgoing.SerializeToString(&encoded);
    if (send(client, encoded.data(), encoded.size(), 0) != static_cast<ssize_t>(encoded.size())) { close(client); continue; }
    std::string container_id; char event[kMaxEventSize];
    while ((size = recv(client, event, sizeof(event), 0)) > 0) {
      if (static_cast<size_t>(size) < sizeof(Header)) { close(client); break; }
      Header header{}; memcpy(&header, event, sizeof(header));
      if (header.header_size < sizeof(Header) || header.header_size > static_cast<uint16_t>(size) || !Handle(header, event + header.header_size, size - header.header_size, output, &container_id)) { close(client); break; }
    }
    if (!container_id.empty()) Send(output, container_id, "stream-end");
    close(client);
  }
}
