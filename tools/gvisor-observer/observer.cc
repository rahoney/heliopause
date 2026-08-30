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
#include <climits>
#include <cerrno>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <cstddef>
#include <string>
#include <map>
#include <utility>

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
constexpr size_t kMaxTrackedFileDescriptorsPerGroup = 4096;
constexpr uint64_t kSyscallExecve = 59;
constexpr uint64_t kSyscallExecveat = 322;
constexpr uint64_t kSyscallSendtoX86 = 44;
constexpr uint64_t kSyscallSendmsgX86 = 46;
constexpr uint64_t kSyscallSendmmsgX86 = 307;
constexpr uint64_t kSyscallSendtoArm64 = 206;
constexpr uint64_t kSyscallSendmsgArm64 = 211;
constexpr uint64_t kSyscallSendmmsgArm64 = 269;
constexpr uint64_t kSyscallCloseRange = 436;
constexpr int kFcntlDupFD = 0;
constexpr int kFcntlDupFDCloexec = 1030;
constexpr int kFcntlSetFD = 2;
constexpr int kFD_CLOEXEC = 1;
constexpr uint64_t kCloneThread = 0x00010000;
// These sizes match the pinned gVisor ABI structures used by the socket
// providers. They bound only address-shape validation; the observer never
// retains address bytes.
constexpr size_t kSockAddrNetlinkSize = 12;
constexpr size_t kSockAddrPacketSize = 20;
// Linux UAPI values consumed from pinned gVisor protobufs. Keep these explicit
// so observer tests do not inherit the build Host's socket-family surface.
constexpr int kLinuxAFNetlink = 16;
constexpr int kLinuxAFPacket = 17;
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

struct Attribution {
  const char* event_source = nullptr;
  const char* family = nullptr;
  const char* process_relation = nullptr;
  const char* process_class = nullptr;
  const char* classification_reason = nullptr;
  const char* parent_relation = nullptr;
};

bool Send(int output, const std::string& container_id, const char* kind, const char* reason = nullptr, const Attribution* attribution = nullptr) {
  if (!ValidContainerID(container_id)) return false;
  std::string message = "{\"container_id\":\"" + container_id + "\",\"kind\":\"" + kind + "\"";
  if (reason != nullptr) message += ",\"reason\":\"" + std::string(reason) + "\"";
  if (attribution != nullptr) {
    if (attribution->event_source != nullptr) message += ",\"event_source\":\"" + std::string(attribution->event_source) + "\"";
    if (attribution->family != nullptr) message += ",\"family\":\"" + std::string(attribution->family) + "\"";
    if (attribution->process_relation != nullptr) message += ",\"process_relation\":\"" + std::string(attribution->process_relation) + "\"";
    if (attribution->process_class != nullptr) message += ",\"process_class\":\"" + std::string(attribution->process_class) + "\"";
    if (attribution->classification_reason != nullptr) message += ",\"classification_reason\":\"" + std::string(attribution->classification_reason) + "\"";
    if (attribution->parent_relation != nullptr) message += ",\"parent_relation\":\"" + std::string(attribution->parent_relation) + "\"";
  }
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

enum class SocketClassification {
  kLocal,
  kSpecialKernelLocal,
  kNetwork,
  kUnknown,
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
  struct FDEntry {
    SocketClassification family;
    int raw_family;
    bool cloexec;
  };
  std::map<int32_t, std::map<int32_t, FDEntry>> fd_states;
  std::map<int32_t, uint32_t> pending_sockets;
  bool launch_root_set = false;
  bool launch_root_active = false;
  int32_t launch_root_group_id = 0;
  int64_t launch_root_group_start_time_ns = 0;
  std::map<int32_t, int64_t> launch_roots;
};

const char* ProcessClassName(ProcessClass process_class) {
  switch (process_class) {
    case ProcessClass::kShell: return "SHELL";
    case ProcessClass::kPython: return "PYTHON";
    case ProcessClass::kPip: return "PIP";
    case ProcessClass::kNode: return "NODE";
    case ProcessClass::kNpm: return "NPM";
    case ProcessClass::kArtifact: return "ARTIFACT";
    case ProcessClass::kSleep: return "SLEEP";
    case ProcessClass::kMkdir: return "MKDIR";
    case ProcessClass::kCat: return "CAT";
    case ProcessClass::kChmod: return "CHMOD";
    case ProcessClass::kUnknown: return "OTHER";
  }
  return "OTHER";
}

ProcessClass ProcessClassForPath(const std::string& path, const char* profile) {
  if (path == "/bin/sh" || path == "sh" || path == "/usr/bin/dash") return ProcessClass::kShell;
  if (path == "/usr/bin/sleep" || path == "/bin/sleep" || path == "sleep") return ProcessClass::kSleep;
  if (path == "/usr/bin/mkdir" || path == "/bin/mkdir" || path == "mkdir") return ProcessClass::kMkdir;
  if (path == "/usr/bin/cat" || path == "/bin/cat" || path == "cat") return ProcessClass::kCat;
  if (path == "/usr/bin/chmod" || path == "/bin/chmod" || path == "chmod") return ProcessClass::kChmod;
  if (profile == nullptr) return ProcessClass::kUnknown;
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
  if (!state.bootstrap_active || !IsBootstrapChildClass(process_class)) return false;
  auto parent = state.expected_groups.find(context.parent_thread_group_id());
  return parent != state.expected_groups.end() && parent->second.process_class == ProcessClass::kShell;
}

enum class TrackResult { kTracked, kIdentityInvalid, kStartTimeMismatch, kClassMismatch, kLimit };

TrackResult TrackExpectedProcessGroup(const gvisor::common::ContextData& context, ProcessClass process_class, ProcessState* state) {
  if (!ValidProcessIdentity(context) || state == nullptr) return TrackResult::kIdentityInvalid;
  auto existing = state->expected_groups.find(context.thread_group_id());
  if (existing != state->expected_groups.end()) {
    if (existing->second.start_time_ns != context.thread_group_start_time_ns()) return TrackResult::kStartTimeMismatch;
    return existing->second.process_class == process_class ? TrackResult::kTracked : TrackResult::kClassMismatch;
  }
  if (state->expected_groups.size() >= kMaxTrackedProcessGroups) return TrackResult::kLimit;
  state->expected_groups[context.thread_group_id()] = ProcessState::ExpectedGroup{context.thread_group_start_time_ns(), process_class};
  return TrackResult::kTracked;
}

struct ProcessClassification {
  bool expected = false;
  ProcessClass process_class = ProcessClass::kUnknown;
  const char* reason = "OTHER";
  const char* parent_relation = "UNKNOWN";
};

const char* TrackFailureReason(TrackResult result) {
  switch (result) {
    case TrackResult::kIdentityInvalid: return "INVALID_PROCESS_IDENTITY";
    case TrackResult::kStartTimeMismatch: return "START_TIME_MISMATCH";
    case TrackResult::kClassMismatch: return "CLASS_MISMATCH";
    case TrackResult::kLimit: return "TRACKING_LIMIT";
    case TrackResult::kTracked: return "OTHER";
  }
  return "OTHER";
}

ProcessClassification IsExpectedProcess(const std::string& path, const gvisor::common::ContextData& context, const char* profile, ProcessState* state) {
  ProcessClassification result;
  result.process_class = ProcessClassForPath(path, profile);
  if (state == nullptr || !ValidProcessIdentity(context)) {
    result.reason = "INVALID_PROCESS_IDENTITY";
    return result;
  }
  const ProcessClass process_class = ProcessClassForPath(path, profile);
  result.process_class = process_class;
  auto tracked = state->expected_groups.find(context.thread_group_id());
  if (tracked != state->expected_groups.end() && tracked->second.start_time_ns != context.thread_group_start_time_ns()) {
    result.reason = "START_TIME_MISMATCH";
    result.parent_relation = "TRACKED_GROUP";
    return result;
  }

  if (tracked != state->expected_groups.end()) {
    result.parent_relation = "TRACKED_GROUP";
    // A successful transition already recorded for this group is a re-exec,
    // not another trusted launch. Trust is consumed once per direct session.
    if (tracked->second.process_class == process_class) {
      result.reason = "BOOTSTRAP_ENDED";
      return result;
    }
    if (state->bootstrap_active && context.thread_group_id() == state->bootstrap_group_id &&
        IsBootstrapRootClass(tracked->second.process_class) && IsBootstrapRootClass(process_class)) {
      tracked->second.process_class = process_class;
      result.expected = true; return result;
    }
    if (state->bootstrap_active && context.thread_group_id() == state->bootstrap_group_id &&
        IsBootstrapHandoffClass(process_class, profile)) {
      tracked->second.process_class = process_class;
      state->bootstrap_active = false;
      result.expected = true; return result;
    }
    result.reason = "CLASS_MISMATCH";
    return result;
  }

  if (state->bootstrap_active && !context.is_exec_session() && IsBootstrapRoot(context, *state) && IsBootstrapRootClass(process_class)) {
    result.parent_relation = "BOOTSTRAP_ROOT";
    if (!state->bootstrap_group_set) {
      state->bootstrap_group_set = true;
      state->bootstrap_group_id = context.thread_group_id();
      state->bootstrap_group_start_time_ns = context.thread_group_start_time_ns();
    }
    const TrackResult tracked_result = TrackExpectedProcessGroup(context, process_class, state);
    result.expected = tracked_result == TrackResult::kTracked;
    result.reason = TrackFailureReason(tracked_result);
    return result;
  }
  if (state->bootstrap_active && IsBootstrapChild(context, *state) && IsBootstrapChildClass(process_class)) {
    result.parent_relation = "BOOTSTRAP_CHILD";
    const TrackResult tracked_result = TrackExpectedProcessGroup(context, process_class, state);
    result.expected = tracked_result == TrackResult::kTracked;
    result.reason = TrackFailureReason(tracked_result);
    return result;
  }
  if (IsTrustedShellChild(context, process_class, *state)) {
    result.parent_relation = "TRACKED_PARENT";
    const TrackResult tracked_result = TrackExpectedProcessGroup(context, process_class, state);
    result.expected = tracked_result == TrackResult::kTracked;
    result.reason = TrackFailureReason(tracked_result);
    return result;
  }
  if (state->bootstrap_active && IsBootstrapChild(context, *state) && IsBootstrapHandoffClass(process_class, profile)) {
    result.parent_relation = "BOOTSTRAP_CHILD";
    state->bootstrap_active = false;
    const TrackResult tracked_result = TrackExpectedProcessGroup(context, process_class, state);
    result.expected = tracked_result == TrackResult::kTracked;
    result.reason = TrackFailureReason(tracked_result);
    return result;
  }
  if (context.is_exec_session() && context.parent_thread_group_id() == 0 && IsProfileExpectedClass(process_class, profile)) {
    result.parent_relation = "DIRECT_EXEC_SESSION";
    state->bootstrap_active = false;
    const TrackResult tracked_result = TrackExpectedProcessGroup(context, process_class, state);
    result.expected = tracked_result == TrackResult::kTracked;
    result.reason = TrackFailureReason(tracked_result);
    return result;
  }
  if (process_class == ProcessClass::kUnknown) result.reason = "UNKNOWN_CLASS";
  else if (context.is_exec_session() && context.parent_thread_group_id() == 0) result.reason = "DIRECT_EXEC_NOT_ALLOWED";
  else if (!state->bootstrap_active) result.reason = "BOOTSTRAP_ENDED";
  else result.reason = "UNMODELED_PARENT";
  result.parent_relation = context.parent_thread_group_id() == 0 ? "ROOT" : "UNTRACKED_PARENT";
  return result;
}

const char* NetworkProcessRelation(const gvisor::common::ContextData& context, const ProcessState& state) {
  if (!ValidProcessIdentity(context)) return "UNKNOWN";
  auto launch_root = state.launch_roots.find(context.thread_group_id());
  if (launch_root != state.launch_roots.end() && launch_root->second == context.thread_group_start_time_ns()) {
    return "DIRECT_EXEC_SESSION";
  }
  auto tracked = state.expected_groups.find(context.thread_group_id());
  if (tracked != state.expected_groups.end()) {
    if (tracked->second.start_time_ns == context.thread_group_start_time_ns()) return "TRACKED_EXPECTED_GROUP";
    return "TRACKED_UNEXPECTED_GROUP";
  }
  if (state.bootstrap_active && IsBootstrapRoot(context, state)) return "BOOTSTRAP_ROOT";
  if (state.bootstrap_active && IsBootstrapChild(context, state)) return "BOOTSTRAP_CHILD";
  return "UNKNOWN";
}

SocketClassification ClassifySocketFamily(int family) {
  if (family == AF_UNIX) return SocketClassification::kLocal;
  if (family == kLinuxAFNetlink) return SocketClassification::kSpecialKernelLocal;
  if (family == AF_INET || family == AF_INET6) return SocketClassification::kNetwork;
  if (family == kLinuxAFPacket) return SocketClassification::kNetwork;
  return SocketClassification::kUnknown;
}

const char* SocketUnknownFamilyReason(int family) {
  switch (family) {
    case AF_UNSPEC: return "SOCKET_AF_UNSPEC";
    case kLinuxAFNetlink: return "SOCKET_AF_NETLINK";
    case kLinuxAFPacket: return "SOCKET_AF_PACKET";
    default: return "SOCKET_OTHER_FAMILY";
  }
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
    case AF_UNSPEC: return length >= sizeof(sa_family_t);
    case AF_UNIX: return length >= sizeof(sa_family_t) && length <= sizeof(sockaddr_un);
    case AF_INET: return length >= sizeof(sockaddr_in);
    case AF_INET6: return length >= sizeof(sockaddr_in6);
    case kLinuxAFNetlink: return length >= kSockAddrNetlinkSize;
    case kLinuxAFPacket: return length >= kSockAddrPacketSize;
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

bool ValidateContextContainer(const gvisor::common::ContextData& context, std::string* container_id, const char** reason) {
  const std::string& candidate = context.container_id();
  if (!ValidContainerID(candidate)) { *reason = "STREAM_FAULT"; return false; }
  if (!container_id->empty() && *container_id != candidate) { *reason = "CONTAINER_MISMATCH"; return false; }
  if (container_id->empty()) *container_id = candidate;
  return true;
}

void ApplyExecCloexec(ProcessState* state, int32_t group_id) {
  auto table = state->fd_states.find(group_id);
  if (table == state->fd_states.end()) return;
  for (auto fd = table->second.begin(); fd != table->second.end();) {
    if (fd->second.cloexec) fd = table->second.erase(fd);
    else ++fd;
  }
}

template <typename Message>
bool ParseExecSyscallTelemetry(const char* payload, size_t payload_size, std::string* container_id, const char** reason) {
  Message message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  if (!ValidateContextContainer(message.context_data(), container_id, reason) || !ValidProcessIdentity(message.context_data())) {
    if (*reason == nullptr) *reason = "EXEC_CORRELATION_INVALID";
    return false;
  }
  const uint64_t syscall_number = message.sysno();
  if (syscall_number != kSyscallExecve && syscall_number != kSyscallExecveat) {
    *reason = "EXEC_CORRELATION_INVALID";
    return false;
  }
  // In pinned gVisor syscall EXIT precedes the exec continuation. ENTER and
  // EXIT are bounded attempt telemetry only; sentry/execve is the image-load
  // boundary that classifies an executable transition.
  return true;
}

bool ParseSentryProcessAndClassify(const char* payload, size_t payload_size, int output, std::string* container_id,
                                   const char* profile, ProcessState* process_state, const char** reason) {
  if (profile == nullptr || process_state == nullptr) return false;
  gvisor::sentry::ExecveInfo message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  if (!ValidateContextContainer(message.context_data(), container_id, reason) || !ValidProcessIdentity(message.context_data())) {
    if (*reason == nullptr) *reason = "EXEC_CORRELATION_INVALID";
    return false;
  }
  ProcessState candidate = *process_state;
  candidate.fd_states.clear();
  const ProcessClassification classification = IsExpectedProcess(message.binary_path(), message.context_data(), profile, &candidate);
  process_state->bootstrap_active = candidate.bootstrap_active;
  process_state->bootstrap_group_set = candidate.bootstrap_group_set;
  process_state->bootstrap_group_id = candidate.bootstrap_group_id;
  process_state->bootstrap_group_start_time_ns = candidate.bootstrap_group_start_time_ns;
  process_state->expected_groups = std::move(candidate.expected_groups);
  process_state->launch_roots = std::move(candidate.launch_roots);
  const int32_t group_id = message.context_data().thread_group_id();
  const int64_t group_start_time_ns = message.context_data().thread_group_start_time_ns();
  if (strcmp(classification.parent_relation, "DIRECT_EXEC_SESSION") == 0 && classification.expected) {
    process_state->launch_root_set = true;
    process_state->launch_root_active = true;
    process_state->launch_root_group_id = group_id;
    process_state->launch_root_group_start_time_ns = group_start_time_ns;
    process_state->launch_roots[group_id] = group_start_time_ns;
  } else if (process_state->launch_root_set && process_state->launch_root_group_id == group_id &&
             process_state->launch_root_group_start_time_ns == group_start_time_ns) {
    process_state->launch_root_active = false;
    process_state->launch_roots.erase(group_id);
  }
  ApplyExecCloexec(process_state, group_id);
  if (classification.expected) return Send(output, *container_id, "process-exec-expected");
  const Attribution attribution{"SENTRY_EXEC", nullptr, nullptr,
                                ProcessClassName(classification.process_class), classification.reason,
                                classification.parent_relation};
  return Send(output, *container_id, "process-exec-unexpected", nullptr, &attribution);
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
bool ParseSocketAndTrack(const char* payload, size_t payload_size, int output, std::string* container_id,
                         const char* profile, ProcessState* process_state, const char** reason) {
  Message message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  if (!ValidateContextContainer(message.context_data(), container_id, reason) || !ValidProcessIdentity(message.context_data())) {
    if (*reason == nullptr) *reason = "FD_STATE_UNKNOWN";
    return false;
  }
  const ProcessClass process_class = ProcessClassForPath(message.context_data().process_name(), profile);
  const char* relation = NetworkProcessRelation(message.context_data(), *process_state);
  const SocketClassification family_class = ClassifySocketFamily(message.domain());
  if (family_class == SocketClassification::kUnknown) { *reason = SocketUnknownFamilyReason(message.domain()); return false; }
  if (!message.has_exit()) {
    if (process_state->pending_sockets[message.context_data().thread_group_id()] >= kMaxTrackedFileDescriptorsPerGroup) {
      *reason = "FD_STATE_LIMIT";
      return false;
    }
    ++process_state->pending_sockets[message.context_data().thread_group_id()];
    if (family_class == SocketClassification::kNetwork && message.domain() == kLinuxAFPacket) {
      const Attribution attribution{"SOCKET", "PACKET", relation, ProcessClassName(process_class), nullptr, nullptr};
      return Send(output, *container_id, "network-attempt", nullptr, &attribution);
    }
    return true;
  }
  auto pending = process_state->pending_sockets.find(message.context_data().thread_group_id());
  if (pending == process_state->pending_sockets.end() || pending->second == 0) { *reason = "FD_STATE_UNKNOWN"; return false; }
  if (--pending->second == 0) process_state->pending_sockets.erase(pending);
  if (message.exit().errorno() != 0 || message.exit().result() < 0) return true;
  const int64_t fd = message.exit().result();
  if (fd < 0 || fd > INT_MAX) { *reason = "FD_STATE_UNKNOWN"; return false; }
  auto table = process_state->fd_states.find(message.context_data().thread_group_id());
  if (table == process_state->fd_states.end()) {
    if (process_state->fd_states.size() >= kMaxTrackedProcessGroups) { *reason = "FD_STATE_LIMIT"; return false; }
    table = process_state->fd_states.emplace(message.context_data().thread_group_id(), std::map<int32_t, ProcessState::FDEntry>{}).first;
  }
  if (table->second.size() >= kMaxTrackedFileDescriptorsPerGroup && table->second.find(static_cast<int32_t>(fd)) == table->second.end()) {
    *reason = "FD_STATE_LIMIT";
    return false;
  }
  table->second[static_cast<int32_t>(fd)] = ProcessState::FDEntry{family_class, message.domain(), (message.type() & 02000000) != 0};
  return true;
}

const char* FamilyName(SocketClassification classification, int raw_family) {
  if (classification != SocketClassification::kNetwork) return nullptr;
  if (raw_family == AF_INET) return "INET";
  if (raw_family == AF_INET6) return "INET6";
  if (raw_family == kLinuxAFPacket) return "PACKET";
  return nullptr;
}

bool ParseCloseAndTrack(const char* payload, size_t payload_size, std::string* container_id, ProcessState* state, const char** reason) {
  gvisor::syscall::Close message;
  if (!message.ParseFromArray(payload, payload_size) || !ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  if (!message.has_exit()) return true;
  if (message.exit().errorno() != 0 || message.exit().result() < 0) return true;
  auto table = state->fd_states.find(message.context_data().thread_group_id());
  if (table != state->fd_states.end()) table->second.erase(static_cast<int32_t>(message.fd()));
  return true;
}

bool ParseDupAndTrack(const char* payload, size_t payload_size, std::string* container_id, ProcessState* state, const char** reason) {
  gvisor::syscall::Dup message;
  if (!message.ParseFromArray(payload, payload_size) || !ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  if (!message.has_exit()) return true;
  if (message.exit().errorno() != 0 || message.exit().result() < 0) return true;
  const int32_t old_fd = message.old_fd();
  const int64_t result_fd = message.exit().result();
  if (result_fd < 0 || result_fd > INT_MAX) { *reason = "FD_STATE_UNKNOWN"; return false; }
  auto source = state->fd_states.find(message.context_data().thread_group_id());
  if (source == state->fd_states.end()) return true;
  auto source_fd = source->second.find(old_fd);
  if (source_fd == source->second.end()) return true;
  if (source->second.size() >= kMaxTrackedFileDescriptorsPerGroup && source->second.find(static_cast<int32_t>(result_fd)) == source->second.end()) {
    *reason = "FD_STATE_LIMIT";
    return false;
  }
  source->second[static_cast<int32_t>(result_fd)] = source_fd->second;
  source->second[static_cast<int32_t>(result_fd)].cloexec = (message.flags() & 02000000) != 0;
  return true;
}

bool ParseFcntlAndTrack(const char* payload, size_t payload_size, std::string* container_id, ProcessState* state, const char** reason) {
  gvisor::syscall::Fcntl message;
  if (!message.ParseFromArray(payload, payload_size) || !ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  if (!message.has_exit()) return true;
  if (message.exit().errorno() != 0 || message.exit().result() < 0) return true;
  auto table = state->fd_states.find(message.context_data().thread_group_id());
  if (table == state->fd_states.end()) return true;
  auto source = table->second.find(message.fd());
  if (message.cmd() == kFcntlSetFD && source != table->second.end()) {
    source->second.cloexec = (message.args() & kFD_CLOEXEC) != 0;
    return true;
  }
  if (message.cmd() != kFcntlDupFD && message.cmd() != kFcntlDupFDCloexec) return true;
  if (source == table->second.end()) return true;
  const int64_t result_fd = message.exit().result();
  if (result_fd < 0 || result_fd > INT_MAX || (table->second.size() >= kMaxTrackedFileDescriptorsPerGroup &&
      table->second.find(static_cast<int32_t>(result_fd)) == table->second.end())) {
    *reason = "FD_STATE_LIMIT";
    return false;
  }
  table->second[static_cast<int32_t>(result_fd)] = source->second;
  table->second[static_cast<int32_t>(result_fd)].cloexec = message.cmd() == kFcntlDupFDCloexec;
  return true;
}

bool ParseCloneAndTrack(const char* payload, size_t payload_size, std::string* container_id, ProcessState* state, const char** reason) {
  gvisor::syscall::Clone message;
  if (!message.ParseFromArray(payload, payload_size) || !ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  if (!message.has_exit() || message.exit().errorno() != 0 || message.exit().result() <= 0 || (message.flags() & kCloneThread) != 0) return true;
  const int32_t child_group = static_cast<int32_t>(message.exit().result());
  if (state->fd_states.find(message.context_data().thread_group_id()) == state->fd_states.end()) return true;
  if (state->fd_states.size() >= kMaxTrackedProcessGroups) { *reason = "FD_STATE_LIMIT"; return false; }
  state->fd_states[child_group] = state->fd_states[message.context_data().thread_group_id()];
  return true;
}

bool ParseForkAndTrack(const char* payload, size_t payload_size, std::string* container_id, ProcessState* state, const char** reason) {
  gvisor::syscall::Fork message;
  if (!message.ParseFromArray(payload, payload_size) || !ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  if (!message.has_exit() || message.exit().errorno() != 0 || message.exit().result() <= 0) return true;
  auto source = state->fd_states.find(message.context_data().thread_group_id());
  if (source == state->fd_states.end()) return true;
  if (state->fd_states.size() >= kMaxTrackedProcessGroups) { *reason = "FD_STATE_LIMIT"; return false; }
  state->fd_states[static_cast<int32_t>(message.exit().result())] = source->second;
  return true;
}

bool ParseRawAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile,
                     ProcessState* state, const char** reason) {
  gvisor::syscall::Syscall message;
  if (!message.ParseFromArray(payload, payload_size) || !ValidateContextContainer(message.context_data(), container_id, reason) || !ValidProcessIdentity(message.context_data())) {
    *reason = "RAW_SYSCALL_INVALID";
    return false;
  }
  const uint64_t sysno = message.sysno();
  if (sysno == kSyscallCloseRange) {
    state->fd_states.erase(message.context_data().thread_group_id());
    return true;
  }
  const char* source = nullptr;
  if (sysno == kSyscallSendtoX86 || sysno == kSyscallSendtoArm64) source = "SENDTO";
  else if (sysno == kSyscallSendmsgX86 || sysno == kSyscallSendmsgArm64) source = "SENDMSG";
  else if (sysno == kSyscallSendmmsgX86 || sysno == kSyscallSendmmsgArm64) source = "SENDMMSG";
  else { *reason = "RAW_SYSCALL_INVALID"; return false; }
  if (message.arg1() > INT_MAX) { *reason = "FD_STATE_UNKNOWN"; return false; }
  auto table = state->fd_states.find(message.context_data().thread_group_id());
  if (table == state->fd_states.end()) { *reason = "FD_STATE_UNKNOWN"; return false; }
  auto fd = table->second.find(static_cast<int32_t>(message.arg1()));
  if (fd == table->second.end()) { *reason = "FD_STATE_UNKNOWN"; return false; }
  if (fd->second.family == SocketClassification::kLocal || fd->second.family == SocketClassification::kSpecialKernelLocal) return true;
  if (fd->second.family != SocketClassification::kNetwork) { *reason = "FD_STATE_UNKNOWN"; return false; }
  const char* family = FamilyName(fd->second.family, fd->second.raw_family);
  if (family == nullptr) { *reason = "FD_STATE_UNKNOWN"; return false; }
  const char* relation = NetworkProcessRelation(message.context_data(), *state);
  const ProcessClass process_class = ProcessClassForPath(message.context_data().process_name(), profile);
  const Attribution attribution{source, family, relation, ProcessClassName(process_class), nullptr, nullptr};
  const bool trusted = relation != nullptr && strcmp(relation, "DIRECT_EXEC_SESSION") == 0;
  return Send(output, *container_id, trusted ? "trusted-control-network" : "network-attempt", nullptr, &attribution);
}

bool ParseConnectAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile, const ProcessState& process_state, const char** reason) {
  gvisor::syscall::Connect message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  if (!ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  const ProcessClass process_class = ProcessClassForPath(message.context_data().process_name(), profile);
  const char* relation = NetworkProcessRelation(message.context_data(), process_state);
  int family = 0;
  if (!ReadSocketFamily(message.address(), &family)) { *reason = "CONNECT_ADDRESS_TOO_SHORT"; return false; }
  const SocketClassification classification = ClassifySocketFamily(family);
  switch (family) {
    case AF_UNSPEC:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_UNSPEC"; return false; }
      return true;
    case AF_UNIX:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_UNIX_INVALID_LENGTH"; return false; }
      return true;
    case AF_INET:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_INET_INVALID_LENGTH"; return false; }
      { const Attribution attribution{"CONNECT", "INET", relation, ProcessClassName(process_class), nullptr, nullptr};
        return Send(output, *container_id, relation != nullptr && strcmp(relation, "DIRECT_EXEC_SESSION") == 0 ? "trusted-control-network" : "network-attempt", nullptr, &attribution); }
    case AF_INET6:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_INET6_INVALID_LENGTH"; return false; }
      { const Attribution attribution{"CONNECT", "INET6", relation, ProcessClassName(process_class), nullptr, nullptr};
        return Send(output, *container_id, relation != nullptr && strcmp(relation, "DIRECT_EXEC_SESSION") == 0 ? "trusted-control-network" : "network-attempt", nullptr, &attribution); }
    case kLinuxAFNetlink:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_NETLINK_INVALID_LENGTH"; return false; }
      return true;
    case kLinuxAFPacket:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_PACKET_INVALID_LENGTH"; return false; }
      { const Attribution attribution{"CONNECT", "PACKET", relation, ProcessClassName(process_class), nullptr, nullptr}; return Send(output, *container_id, "network-attempt", nullptr, &attribution); }
    default:
      (void)classification;
      *reason = "CONNECT_UNKNOWN_FAMILY";
      return false;
  }
}
bool Handle(const Header& header, const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile, ProcessState* process_state, const char** reason) {
  if (header.dropped_count != 0) { *reason = "STREAM_FAULT"; return false; }
  switch (static_cast<gvisor::common::MessageType>(header.message_type)) {
    case gvisor::common::MESSAGE_CONTAINER_START: return ParseAndSend<gvisor::container::Start>(payload, payload_size, output, "container-start", container_id, profile, reason);
    case gvisor::common::MESSAGE_SENTRY_CLONE: return ParseAndSend<gvisor::sentry::CloneInfo>(payload, payload_size, output, "process-clone", container_id, profile, reason);
    case gvisor::common::MESSAGE_SENTRY_EXEC: return ParseSentryProcessAndClassify(payload, payload_size, output, container_id, profile, process_state, reason);
    // pathname, argv and envv are parsed by protobuf but are deliberately never
    // copied to the HAA envelope. M11-003 supplies the trusted profile required
    // to classify this bounded process fact as expected or unexpected.
    case gvisor::common::MESSAGE_SYSCALL_EXECVE: return ParseExecSyscallTelemetry<gvisor::syscall::Execve>(payload, payload_size, container_id, reason);
    case gvisor::common::MESSAGE_SYSCALL_OPEN: return ParseOpenAndSend(payload, payload_size, output, container_id, profile, reason);
    case gvisor::common::MESSAGE_SYSCALL_CONNECT: return ParseConnectAndSend(payload, payload_size, output, container_id, profile, *process_state, reason);
    case gvisor::common::MESSAGE_SYSCALL_SOCKET: return ParseSocketAndTrack<gvisor::syscall::Socket>(payload, payload_size, output, container_id, profile, process_state, reason);
    case gvisor::common::MESSAGE_SYSCALL_RAW: return ParseRawAndSend(payload, payload_size, output, container_id, profile, process_state, reason);
    case gvisor::common::MESSAGE_SYSCALL_CLOSE: return ParseCloseAndTrack(payload, payload_size, container_id, process_state, reason);
    case gvisor::common::MESSAGE_SYSCALL_DUP: return ParseDupAndTrack(payload, payload_size, container_id, process_state, reason);
    case gvisor::common::MESSAGE_SYSCALL_FCNTL: return ParseFcntlAndTrack(payload, payload_size, container_id, process_state, reason);
    case gvisor::common::MESSAGE_SYSCALL_CLONE: return ParseCloneAndTrack(payload, payload_size, container_id, process_state, reason);
    case gvisor::common::MESSAGE_SYSCALL_FORK: return ParseForkAndTrack(payload, payload_size, container_id, process_state, reason);
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
    if (!fault && !process_state.pending_sockets.empty()) {
      fault = true;
      fault_reason = "EXEC_CORRELATION_MISSING";
    }
    if (!container_id.empty()) {
      Send(output, container_id, fault ? "stream-fault" : "stream-end", fault_reason);
      profiles.erase(container_id);
    }
    close(client);
  }
}
