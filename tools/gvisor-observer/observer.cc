// HAA's gVisor observer is compiled inside the exact pinned gVisor source tree.
// It consumes the upstream remote sink protocol and emits only bounded kinds to
// HAA's trusted datagram boundary; it never logs trace payloads.
#include <arpa/inet.h>
#include <err.h>
#include <netinet/in.h>
#include <poll.h>
#include <signal.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include <cstdint>
#include <cerrno>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <cstddef>
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
constexpr size_t kMaxPyTorchCPURecordsPerConnection = 500000;
constexpr size_t kMaxPyTorchCU126RecordsPerConnection = 100000;
constexpr size_t kMaxTrackedProcessGroups = 64;
constexpr int kProfileRegistrationWaitMilliseconds = 2000;
constexpr char kProfileNPM[] = "npm-lifecycle";
constexpr char kProfilePyPI[] = "pypi-wheel";
constexpr char kProfilePyTorchCPU[] = "pypi-wheel-pytorch-cpu";
constexpr char kProfilePyTorchCU126[] = "pypi-wheel-pytorch-cu126";
constexpr char kProfileGitHub[] = "github-elf";
#pragma pack(push, 1)
struct Header { uint16_t header_size; uint16_t message_type; uint32_t dropped_count; };
#pragma pack(pop)

bool ValidContainerID(const std::string& container_id) {
  if (container_id.size() < 12 || container_id.size() > 64) return false;
  for (const char character : container_id) if ((character < 'a' || character > 'f') && (character < '0' || character > '9')) return false;
  return true;
}

bool Send(int output, const std::string& container_id, const char* kind, const char* reason = nullptr) {
  if (!ValidContainerID(container_id)) return false;
  std::string message = "{\"container_id\":\"" + container_id + "\",\"kind\":\"" + kind + "\"";
  if (reason != nullptr) message += ",\"reason\":\"" + std::string(reason) + "\"";
  message += "}";
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

enum class ProcessClass {
  kUnknown,
  kShell,
  kPython,
  kPip,
  kNode,
  kNpm,
  kArtifact,
  kSleep,
  kMkdir,
  kCat,
  kChmod,
};

struct ProcessState {
  bool bootstrap_active = true;
  bool bootstrap_group_set = false;
  int32_t bootstrap_group_id = 0;
  int64_t bootstrap_group_start_time_ns = 0;
  struct ExpectedGroup {
    int64_t start_time_ns;
    ProcessClass process_class;
  };
  std::map<int32_t, ExpectedGroup> expected_groups;
};

ProcessClass ProcessClassForPath(const std::string& path, const char* profile) {
  if (path == "/bin/sh" || path == "sh" || path == "/usr/bin/dash") return ProcessClass::kShell;
  if (path == "/usr/bin/sleep" || path == "/bin/sleep" || path == "sleep") return ProcessClass::kSleep;
  if (path == "/usr/bin/mkdir" || path == "/bin/mkdir" || path == "mkdir") return ProcessClass::kMkdir;
  if (path == "/usr/bin/cat" || path == "/bin/cat" || path == "cat") return ProcessClass::kCat;
  if (path == "/usr/bin/chmod" || path == "/bin/chmod" || path == "chmod") return ProcessClass::kChmod;
  if (strcmp(profile, kProfileNPM) == 0) {
    if (path == "/usr/local/bin/node" || path == "node") return ProcessClass::kNode;
    if (path == "/usr/local/bin/npm" || path == "npm") return ProcessClass::kNpm;
  }
  if (strcmp(profile, kProfilePyPI) == 0 || strcmp(profile, kProfilePyTorchCPU) == 0 || strcmp(profile, kProfilePyTorchCU126) == 0) {
    if (path == "/usr/local/bin/python" || path == "/usr/local/bin/python3" || path == "/usr/local/bin/python3.14" || path == "python") return ProcessClass::kPython;
    if (path == "/usr/local/bin/pip" || path == "pip") return ProcessClass::kPip;
  }
  if (strcmp(profile, kProfileGitHub) == 0 && path == "/work/artifact") return ProcessClass::kArtifact;
  return ProcessClass::kUnknown;
}

bool IsProfileExpectedClass(ProcessClass process_class, const char* profile) {
  if (process_class == ProcessClass::kShell) return true;
  if (strcmp(profile, kProfileNPM) == 0) return process_class == ProcessClass::kNode || process_class == ProcessClass::kNpm;
  if (strcmp(profile, kProfilePyPI) == 0 || strcmp(profile, kProfilePyTorchCPU) == 0 || strcmp(profile, kProfilePyTorchCU126) == 0) {
    return process_class == ProcessClass::kPython || process_class == ProcessClass::kPip;
  }
  return process_class == ProcessClass::kArtifact;
}

bool IsBootstrapRootClass(ProcessClass process_class) {
  return process_class == ProcessClass::kShell || process_class == ProcessClass::kSleep;
}

bool IsBootstrapChildClass(ProcessClass process_class) {
  return process_class == ProcessClass::kSleep || process_class == ProcessClass::kMkdir ||
      process_class == ProcessClass::kCat || process_class == ProcessClass::kChmod;
}

bool IsBootstrapHandoffClass(ProcessClass process_class, const char* profile) {
  if (strcmp(profile, kProfileNPM) == 0) return process_class == ProcessClass::kNode || process_class == ProcessClass::kNpm;
  return strcmp(profile, kProfileGitHub) == 0 && process_class == ProcessClass::kArtifact;
}

bool ValidProcessIdentity(const gvisor::common::ContextData& context) {
  return context.thread_group_id() > 0 && context.thread_group_start_time_ns() > 0;
}

bool IsBootstrapRoot(const gvisor::common::ContextData& context, const ProcessState& state) {
  return !context.is_exec_session() && context.parent_thread_group_id() == 0 &&
      (!state.bootstrap_group_set || context.thread_group_id() == state.bootstrap_group_id) &&
      (!state.bootstrap_group_set || context.thread_group_start_time_ns() == state.bootstrap_group_start_time_ns);
}

bool IsBootstrapChild(const gvisor::common::ContextData& context, const ProcessState& state) {
  return state.bootstrap_group_set && !context.is_exec_session() &&
      context.parent_thread_group_id() == state.bootstrap_group_id;
}

bool IsTrustedShellChild(const gvisor::common::ContextData& context, ProcessClass process_class, const ProcessState& state) {
  if (!IsBootstrapChildClass(process_class)) return false;
  auto parent = state.expected_groups.find(context.parent_thread_group_id());
  return parent != state.expected_groups.end() && parent->second.process_class == ProcessClass::kShell;
}

bool TrackExpectedProcessGroup(const gvisor::common::ContextData& context, ProcessClass process_class, ProcessState* state) {
  if (!ValidProcessIdentity(context) || state == nullptr) return false;
  auto existing = state->expected_groups.find(context.thread_group_id());
  if (existing != state->expected_groups.end()) {
    return existing->second.start_time_ns == context.thread_group_start_time_ns() && existing->second.process_class == process_class;
  }
  if (state->expected_groups.size() >= kMaxTrackedProcessGroups) return false;
  state->expected_groups[context.thread_group_id()] = ProcessState::ExpectedGroup{context.thread_group_start_time_ns(), process_class};
  return true;
}

bool IsExpectedProcess(const std::string& path, const gvisor::common::ContextData& context, const char* profile, ProcessState* state) {
  if (state == nullptr || !ValidProcessIdentity(context)) return false;
  const ProcessClass process_class = ProcessClassForPath(path, profile);
  auto tracked = state->expected_groups.find(context.thread_group_id());
  if (tracked != state->expected_groups.end() && tracked->second.start_time_ns != context.thread_group_start_time_ns()) return false;

  if (tracked != state->expected_groups.end()) {
    if (tracked->second.process_class == process_class) return true;
    if (state->bootstrap_active && context.thread_group_id() == state->bootstrap_group_id &&
        IsBootstrapRootClass(tracked->second.process_class) && IsBootstrapRootClass(process_class)) {
      tracked->second.process_class = process_class;
      return true;
    }
    if (state->bootstrap_active && context.thread_group_id() == state->bootstrap_group_id &&
        IsBootstrapHandoffClass(process_class, profile)) {
      tracked->second.process_class = process_class;
      state->bootstrap_active = false;
      return true;
    }
    return false;
  }

  if (state->bootstrap_active && !context.is_exec_session() && IsBootstrapRoot(context, *state) && IsBootstrapRootClass(process_class)) {
    if (!state->bootstrap_group_set) {
      state->bootstrap_group_set = true;
      state->bootstrap_group_id = context.thread_group_id();
      state->bootstrap_group_start_time_ns = context.thread_group_start_time_ns();
    }
    return TrackExpectedProcessGroup(context, process_class, state);
  }
  if (state->bootstrap_active && IsBootstrapChild(context, *state) && IsBootstrapChildClass(process_class)) {
    return TrackExpectedProcessGroup(context, process_class, state);
  }
  if (IsTrustedShellChild(context, process_class, *state)) {
    return TrackExpectedProcessGroup(context, process_class, state);
  }
  if (state->bootstrap_active && IsBootstrapChild(context, *state) && IsBootstrapHandoffClass(process_class, profile)) {
    state->bootstrap_active = false;
    return TrackExpectedProcessGroup(context, process_class, state);
  }
  if (context.is_exec_session() && context.parent_thread_group_id() == 0 && IsProfileExpectedClass(process_class, profile)) {
    state->bootstrap_active = false;
    return TrackExpectedProcessGroup(context, process_class, state);
  }
  return false;
}

enum class SocketClassification {
  kLocal,
  kNetwork,
  kUnknown,
};

SocketClassification ClassifySocketFamily(int family) {
  if (family == AF_UNIX) return SocketClassification::kLocal;
  if (family == AF_INET || family == AF_INET6) return SocketClassification::kNetwork;
  return SocketClassification::kUnknown;
}

bool ReadSocketFamily(const std::string& address, int* family) {
  if (family == nullptr || address.size() < sizeof(sa_family_t)) return false;
  sa_family_t parsed = 0;
  memcpy(&parsed, address.data(), sizeof(parsed));
  *family = static_cast<int>(parsed);
  return true;
}

bool ValidSocketAddressLength(int family, size_t length) {
  switch (family) {
    case AF_UNIX: return length >= offsetof(sockaddr_un, sun_path) + 1 && length <= sizeof(sockaddr_un);
    case AF_INET: return length >= sizeof(sockaddr_in);
    case AF_INET6: return length >= sizeof(sockaddr_in6);
    default: return false;
  }
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
  if (!ValidContainerID(id) || (profile != kProfileNPM && profile != kProfilePyPI && profile != kProfilePyTorchCPU && profile != kProfilePyTorchCU126 && profile != kProfileGitHub)) return false;
  if (profiles->find(id) != profiles->end()) return false;
  (*profiles)[id] = profile;
  return true;
}

size_t MaximumRecords(const char* profile) {
  if (profile == nullptr) return kMaxNormalizedRecordsPerConnection;
  if (strcmp(profile, kProfilePyTorchCPU) == 0) return kMaxPyTorchCPURecordsPerConnection;
  if (strcmp(profile, kProfilePyTorchCU126) == 0) return kMaxPyTorchCU126RecordsPerConnection;
  return kMaxNormalizedRecordsPerConnection;
}

bool DrainProfiles(int control, std::map<std::string, std::string>* profiles) {
  char control_message[512];
  ssize_t control_size;
  while ((control_size = recv(control, control_message, sizeof(control_message), MSG_DONTWAIT)) > 0) {
    if (!ParseControlRecord(control_message, static_cast<size_t>(control_size), profiles)) return false;
  }
  return control_size == 0 || (control_size < 0 && (errno == EAGAIN || errno == EWOULDBLOCK));
}

const char* AwaitProfile(int control, const std::string& container_id, std::map<std::string, std::string>* profiles) {
  for (int waited = 0; waited < kProfileRegistrationWaitMilliseconds; waited += 50) {
    if (!DrainProfiles(control, profiles)) return nullptr;
    auto profile = profiles->find(container_id);
    if (profile != profiles->end()) return profile->second.c_str();
    pollfd descriptor{control, POLLIN, 0};
    const int result = poll(&descriptor, 1, 50);
    if (result < 0) {
      if (errno == EINTR) continue;
      return nullptr;
    }
    if (result == 0) continue;
    if ((descriptor.revents & POLLIN) == 0) return nullptr;
  }
  return nullptr;
}

template <typename Message>
bool ParseAndSend(const char* payload, size_t payload_size, int output, const char* kind, std::string* container_id, const char* profile, const char** reason) {
  (void)profile;
  Message message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate)) { *reason = "STREAM_FAULT"; return false; }
  if (container_id->empty()) {
    *container_id = candidate;
  } else if (*container_id != candidate) {
    *reason = "CONTAINER_MISMATCH"; return false;
  }
  return Send(output, *container_id, kind);
}

template <typename Message>
bool ParseProcessAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile, ProcessState* process_state, const char** reason) {
  if (profile == nullptr) return false;
  Message message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate)) { *reason = "STREAM_FAULT"; return false; }
  if (!container_id->empty() && *container_id != candidate) { *reason = "CONTAINER_MISMATCH"; return false; }
  if (container_id->empty()) *container_id = candidate;
  const std::string path = message.pathname();
  return Send(output, *container_id, IsExpectedProcess(path, message.context_data(), profile, process_state) ? "process-exec-expected" : "process-exec-unexpected");
}

bool ParseSentryProcessAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile, ProcessState* process_state, const char** reason) {
  if (profile == nullptr) return false;
  gvisor::sentry::ExecveInfo message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate)) { *reason = "STREAM_FAULT"; return false; }
  if (!container_id->empty() && *container_id != candidate) { *reason = "CONTAINER_MISMATCH"; return false; }
  if (container_id->empty()) *container_id = candidate;
  return Send(output, *container_id, IsExpectedProcess(message.binary_path(), message.context_data(), profile, process_state) ? "process-exec-expected" : "process-exec-unexpected");
}

bool ParseOpenAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile, const char** reason) {
  if (profile == nullptr) return false;
  gvisor::syscall::Open message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate)) { *reason = "STREAM_FAULT"; return false; }
  if (!container_id->empty() && *container_id != candidate) { *reason = "CONTAINER_MISMATCH"; return false; }
  if (container_id->empty()) *container_id = candidate;
  return Send(output, *container_id, ProfileKind(message.pathname(), profile));
}

template <typename Message>
bool ParseSocketAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char** reason) {
  Message message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate)) { *reason = "STREAM_FAULT"; return false; }
  if (!container_id->empty() && *container_id != candidate) { *reason = "CONTAINER_MISMATCH"; return false; }
  if (container_id->empty()) *container_id = candidate;
  switch (ClassifySocketFamily(message.domain())) {
    case SocketClassification::kLocal: return true;
    case SocketClassification::kNetwork: return Send(output, *container_id, "network-attempt");
    case SocketClassification::kUnknown: *reason = "STREAM_FAULT"; return false;
  }
  *reason = "STREAM_FAULT";
  return false;
}

bool ParseConnectAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char** reason) {
  gvisor::syscall::Connect message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  const std::string& candidate = message.context_data().container_id();
  if (!ValidContainerID(candidate)) { *reason = "STREAM_FAULT"; return false; }
  if (!container_id->empty() && *container_id != candidate) { *reason = "CONTAINER_MISMATCH"; return false; }
  if (container_id->empty()) *container_id = candidate;
  int family = 0;
  if (!ReadSocketFamily(message.address(), &family)) { *reason = "STREAM_FAULT"; return false; }
  if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "STREAM_FAULT"; return false; }
  switch (ClassifySocketFamily(family)) {
    case SocketClassification::kLocal: return true;
    case SocketClassification::kNetwork: return Send(output, *container_id, "network-attempt");
    case SocketClassification::kUnknown: *reason = "STREAM_FAULT"; return false;
  }
  *reason = "STREAM_FAULT";
  return false;
}

bool Handle(const Header& header, const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile, ProcessState* process_state, const char** reason) {
  if (header.dropped_count != 0) { *reason = "STREAM_FAULT"; return false; }
  switch (static_cast<gvisor::common::MessageType>(header.message_type)) {
    case gvisor::common::MESSAGE_CONTAINER_START: return ParseAndSend<gvisor::container::Start>(payload, payload_size, output, "container-start", container_id, profile, reason);
    case gvisor::common::MESSAGE_SENTRY_CLONE: return ParseAndSend<gvisor::sentry::CloneInfo>(payload, payload_size, output, "process-clone", container_id, profile, reason);
    case gvisor::common::MESSAGE_SENTRY_EXEC: return ParseSentryProcessAndSend(payload, payload_size, output, container_id, profile, process_state, reason);
    // pathname, argv and envv are parsed by protobuf but are deliberately never
    // copied to the HAA envelope. M11-003 supplies the trusted profile required
    // to classify this bounded process fact as expected or unexpected.
    case gvisor::common::MESSAGE_SYSCALL_EXECVE: return ParseProcessAndSend<gvisor::syscall::Execve>(payload, payload_size, output, container_id, profile, process_state, reason);
    case gvisor::common::MESSAGE_SYSCALL_OPEN: return ParseOpenAndSend(payload, payload_size, output, container_id, profile, reason);
    case gvisor::common::MESSAGE_SYSCALL_CONNECT: return ParseConnectAndSend(payload, payload_size, output, container_id, reason);
    case gvisor::common::MESSAGE_SYSCALL_SOCKET: return ParseSocketAndSend<gvisor::syscall::Socket>(payload, payload_size, output, container_id, reason);
    default: *reason = "UNKNOWN_EVENT_KIND"; return false;
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
    if (!DrainProfiles(control, &profiles)) errx(1, "invalid observer profile registration");
    int client = accept(listener, nullptr, nullptr); if (client < 0) err(1, "accept remote");
    char handshake[1024]; ssize_t size = recv(client, handshake, sizeof(handshake), 0);
    gvisor::common::Handshake incoming;
    if (size <= 0 || !incoming.ParseFromArray(handshake, size) || incoming.version() != kProtocolVersion) { close(client); continue; }
    gvisor::common::Handshake outgoing; outgoing.set_version(kProtocolVersion); std::string encoded; outgoing.SerializeToString(&encoded);
    if (send(client, encoded.data(), encoded.size(), 0) != static_cast<ssize_t>(encoded.size())) { close(client); continue; }
    std::string container_id; char event[kMaxEventSize]; bool fault = false;
    ProcessState process_state;
    const char* fault_reason = nullptr;
    const char* profile = nullptr;
    size_t normalized_records = 0;
    while ((size = recv(client, event, sizeof(event), MSG_TRUNC)) > 0) {
      if (static_cast<size_t>(size) > sizeof(event)) { fault = true; fault_reason = "STREAM_FAULT"; break; }
      if (static_cast<size_t>(size) < sizeof(Header)) { fault = true; fault_reason = "STREAM_FAULT"; break; }
      Header header{}; memcpy(&header, event, sizeof(header));
      if (profile == nullptr && !container_id.empty()) {
        profile = AwaitProfile(control, container_id, &profiles);
        if (profile == nullptr) { fault = true; fault_reason = "PROFILE_LOOKUP_FAILURE"; break; }
      }
      if (normalized_records == MaximumRecords(profile)) { fault = true; fault_reason = "EVENT_LIMIT"; break; }
      if (header.header_size < sizeof(Header) || header.header_size > static_cast<uint16_t>(size)) { fault = true; fault_reason = "STREAM_FAULT"; break; }
      if (!Handle(header, event + header.header_size, size - header.header_size, output, &container_id, profile, &process_state, &fault_reason)) { fault = true; if (fault_reason == nullptr) fault_reason = "STREAM_FAULT"; break; }
      ++normalized_records;
    }
    if (size < 0) { fault = true; fault_reason = "STREAM_FAULT"; }
    if (!container_id.empty()) {
      Send(output, container_id, fault ? "stream-fault" : "stream-end", fault_reason);
      profiles.erase(container_id);
    }
    close(client);
  }
}
