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
#include <set>
#include <utility>
#include <vector>

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
constexpr uint64_t kMaxNormalizedObservationCount = 10000;
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
constexpr uint64_t kOpenAccessMode = 00000003;
constexpr uint64_t kOpenWriteOnly = 00000001;
constexpr uint64_t kOpenReadWrite = 00000002;
constexpr uint64_t kOpenCreate = 00000100;
constexpr uint64_t kOpenTruncate = 00001000;
constexpr uint64_t kOpenAppend = 00002000;
constexpr uint64_t kOpenLargefile = 00100000;
constexpr uint64_t kCloneThread = 0x00010000;
constexpr char kBoundaryHelperPath[] = "/haa-runtime/haa-boundary";
constexpr char kSetprivPath[] = "/usr/bin/setpriv";
constexpr char kLaunchMode[] = "--launch";
constexpr char kPythonHandoffMode[] = "--handoff-python";
constexpr char kELFHandoffMode[] = "--handoff-elf";
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
constexpr size_t kMaxTopologyMounts = 64;
constexpr size_t kMaxTopologyMountpointBytes = 512;
constexpr size_t kMaxTopologyFilesystemTypeBytes = 32;
constexpr size_t kMaxTopologySnapshotBytes = 64 << 10;
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

bool Send(int output, const std::string& container_id, const char* kind, const char* reason = nullptr,
          const Attribution* attribution = nullptr, uint64_t count = 0) {
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
  if (count != 0) message += ",\"count\":" + std::to_string(count);
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
  enum class Role { kUnknown, kControl, kArtifact };
  enum class Provenance { kUnknown, kOCIRoot, kDirectExecRoot, kCloneChild };
  enum class OCIBootstrapStage { kNotOCI, kAwaitingBootstrapShell, kAwaitingDemotion, kAwaitingSleep, kComplete };
  struct GroupState {
    int64_t start_time_ns;
    Role role;
    Provenance provenance;
    bool root_eligible;
    bool root_consumed;
    // This is deliberately distinct from role/provenance. Only the actual
    // target of one accepted direct-control launch may use the narrow network
    // exception, and the bit is cleared before every later image transition.
    bool trusted_control_network_active;
    bool demotion_pending;
    bool launch_target_pending;
    bool handoff_target_pending;
    // This is set only after the exact fixed npm CLI image and argv execute
    // in a bounded control clone child. It permits one exact interpreter
    // image transition; it never grants or restores trust.
    bool npm_node_transition_pending;
    ProcessClass handoff_target_class;
    OCIBootstrapStage oci_bootstrap_stage;
  };
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
  std::map<int32_t, GroupState> groups;
  struct PendingOpen {
    int32_t thread_group_id;
    int64_t thread_group_start_time_ns;
    uint64_t sysno;
    uint32_t flags;
    bool early_finding_emitted;
  };
  std::map<std::pair<int32_t, int64_t>, PendingOpen> pending_opens;
};

struct NormalizedCounts {
  uint64_t workspace_access = 0;
  uint64_t runtime_root_access = 0;
  size_t immediate_records = 0;
};

struct ExpectedMount {
  std::string mountpoint;
  std::string mount_class;
  std::string parent;
  std::string filesystem_type;
  bool read_only;
  bool noexec;
  bool nosuid;
  bool nodev;
};

struct MountAnchor {
  uint64_t mount_id;
  std::string mountpoint;
  std::string mount_class;
};

struct TopologyState {
  std::vector<ExpectedMount> expected;
  std::map<uint64_t, MountAnchor> anchors;
  uint64_t namespace_id = 0;
  bool snapshot_seen = false;
  bool sealed = false;
};

struct ProfileRegistration {
  std::string profile;
  std::vector<ExpectedMount> expected;
};

enum class BoundaryMode { kNone, kLaunch, kHandoff, kPythonHandoff, kELFHandoff };

bool SameGroup(const ProcessState::GroupState& group, const gvisor::common::ContextData& context);
bool ValidProcessIdentity(const gvisor::common::ContextData& context);
bool ValidateContextContainer(const gvisor::common::ContextData& context,
                              std::string* container_id, const char** reason);
bool IsNormalizedAbsolutePath(const std::string& path);
bool IsAtOrBelowMountpoint(const std::string& path, const std::string& mountpoint);

bool SameGroup(const ProcessState::GroupState& group, const gvisor::common::ContextData& context) {
  return group.start_time_ns == context.thread_group_start_time_ns();
}

BoundaryMode BoundaryInvocation(const gvisor::sentry::ExecveInfo& message) {
  if (message.execfn() != kBoundaryHelperPath && message.binary_path() != kBoundaryHelperPath) return BoundaryMode::kNone;
  int mode_index = -1;
  const int bounded_argc = message.argv_size() < 8 ? message.argv_size() : 8;
  for (int index = 0; index + 1 < bounded_argc; ++index) {
    if (message.argv(index) == kBoundaryHelperPath) {
      mode_index = index + 1;
      break;
    }
  }
  if (mode_index < 0) return BoundaryMode::kNone;
  const std::string& mode = message.argv(mode_index);
  if (mode == kLaunchMode) return BoundaryMode::kLaunch;
  if (mode == kPythonHandoffMode) return BoundaryMode::kPythonHandoff;
  if (mode == kELFHandoffMode) return BoundaryMode::kELFHandoff;
  // npm invokes script-shell as <path> -c <script>; the fixed helper path is
  // the only trust-removal marker, while the script remains opaque.
  if (mode == "-c") return BoundaryMode::kHandoff;
  return BoundaryMode::kNone;
}

bool IsExactSetprivDemotion(const gvisor::sentry::ExecveInfo& message) {
  if (message.binary_path() != kSetprivPath || message.execfn() != kSetprivPath ||
      message.argv_size() < 10) return false;
  static constexpr const char* kRequired[] = {
      kSetprivPath, "--reuid=1000", "--regid=1000", "--clear-groups",
      "--inh-caps=-all", "--ambient-caps=-all", "--bounding-set=-all",
      "--no-new-privs", "--"};
  for (size_t index = 0; index < sizeof(kRequired) / sizeof(kRequired[0]); ++index) {
    if (message.argv(static_cast<int>(index)) != kRequired[index]) return false;
  }
  return true;
}

bool IsExactOCIBootstrapDemotion(const gvisor::sentry::ExecveInfo& message) {
  return IsExactSetprivDemotion(message) && message.argv_size() == 11 &&
      message.argv(9) == "/bin/sleep" && message.argv(10) == "infinity";
}

// This is the byte-for-byte identity produced by boundaryContainerCommand().
// A deliberate Go-side bootstrap change must fail closed here until reviewed.
constexpr char kOCIBootstrapCommand[] = R"HAA(set -eu; tmp=/haa-runtime/.haa-boundary.tmp; printf '%s' '#!/bin/sh
set -eu
demote() { exec /usr/bin/setpriv --reuid=1000 --regid=1000 --clear-groups --inh-caps=-all --ambient-caps=-all --bounding-set=-all --no-new-privs -- "$@"; }
already_demoted() {
  uid= gid= groups= cap_inh= cap_prm= cap_eff= cap_bnd= cap_amb=
  while IFS=: read -r key value; do
    set -- $value
    case "$key" in
      Uid) [ "$#" -eq 4 ] && [ "$1" = 1000 ] && [ "$2" = 1000 ] && [ "$3" = 1000 ] && [ "$4" = 1000 ] && uid=1 ;;
      Gid) [ "$#" -eq 4 ] && [ "$1" = 1000 ] && [ "$2" = 1000 ] && [ "$3" = 1000 ] && [ "$4" = 1000 ] && gid=1 ;;
      Groups) [ "$#" -eq 0 ] && groups=1 ;;
      CapInh) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_inh=1 ;;
      CapPrm) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_prm=1 ;;
      CapEff) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_eff=1 ;;
      CapBnd) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_bnd=1 ;;
      CapAmb) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_amb=1 ;;
    esac
  done < /proc/self/status
  [ "$uid:$gid:$groups:$cap_inh:$cap_prm:$cap_eff:$cap_bnd:$cap_amb" = 1:1:1:1:1:1:1:1 ]
}
case "${1-}" in
  --origin-launch) shift; exec /haa-runtime/haa-boundary --launch "$@" ;;
  --origin-handoff-python) shift; exec /haa-runtime/haa-boundary --handoff-python "$@" ;;
  --origin-handoff-elf) shift; exec /haa-runtime/haa-boundary --handoff-elf "$@" ;;
  --launch|--handoff-python|--handoff-elf) shift; demote "$@" ;;
  -c) shift; already_demoted; exec /bin/sh -c "$@" ;;
  *) exit 125 ;;
esac
' > "$tmp"; chown 0:0 "$tmp"; chmod 0555 "$tmp"; mv "$tmp" /haa-runtime/haa-boundary; exec /usr/bin/setpriv --reuid=1000 --regid=1000 --clear-groups --inh-caps=-all --ambient-caps=-all --bounding-set=-all --no-new-privs -- /bin/sleep infinity)HAA";

bool IsExactOCIBootstrapShellIdentity(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == "/usr/bin/dash" && message.execfn() == "/bin/sh" &&
      message.argv_size() == 3 && message.argv(0) == "/bin/sh" &&
      message.argv(1) == "-ceu" && message.argv(2) == kOCIBootstrapCommand;
}

bool IsExactOCIBootstrapContainerStart(const gvisor::container::Start& message) {
  return message.args_size() == 3 && message.args(0) == "/bin/sh" &&
      message.args(1) == "-ceu" && message.args(2) == kOCIBootstrapCommand;
}

bool IsExactOCIBootstrapShell(const gvisor::sentry::ExecveInfo& message,
                              const ProcessState::GroupState& group,
                              const ProcessState& state) {
  const auto& context = message.context_data();
  return group.role == ProcessState::Role::kControl &&
      group.provenance == ProcessState::Provenance::kOCIRoot &&
      group.oci_bootstrap_stage == ProcessState::OCIBootstrapStage::kAwaitingBootstrapShell &&
      group.start_time_ns == context.thread_group_start_time_ns() &&
      state.bootstrap_active && !group.trusted_control_network_active &&
      !context.is_exec_session() && context.parent_thread_group_id() == 0 &&
      IsExactOCIBootstrapShellIdentity(message);
}

bool IsExactOCIBootstrapSleep(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == "/usr/bin/sleep" && message.execfn() == "/bin/sleep" &&
      message.argv_size() == 2 && message.argv(0) == "/bin/sleep" &&
      message.argv(1) == "infinity";
}

// These are the exact pinned npm launcher and Node interpreter argv tuples
// emitted by the fixed HAA npm lifecycle command. They are intentionally
// duplicated here so a Go-side command drift fails closed at observation.
constexpr char kNpmCLIPath[] = "/usr/local/lib/node_modules/npm/bin/npm-cli.js";
constexpr char kNpmPath[] = "/usr/local/bin/npm";
constexpr char kNodePath[] = "/usr/local/bin/node";

bool IsExactNpmLifecycleArguments(const gvisor::sentry::ExecveInfo& message,
                                  int first_argument) {
  static constexpr const char* kArguments[] = {
      "install", "--ignore-scripts=false", "--no-audit", "--no-fund",
      "--offline", "--no-update-notifier", "/tmp/artifact.tgz"};
  if (message.argv_size() != first_argument + static_cast<int>(sizeof(kArguments) / sizeof(kArguments[0]))) return false;
  for (size_t index = 0; index < sizeof(kArguments) / sizeof(kArguments[0]); ++index) {
    if (message.argv(first_argument + static_cast<int>(index)) != kArguments[index]) return false;
  }
  return true;
}

bool IsExactNpmCLILauncher(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kNpmCLIPath && message.execfn() == kNpmPath &&
      message.argv_size() == 8 && message.argv(0) == kNpmPath &&
      IsExactNpmLifecycleArguments(message, 1);
}

bool IsExactNpmNodeInterpreter(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kNodePath && message.execfn() == kNodePath &&
      message.argv_size() == 9 && message.argv(0) == "node" &&
      message.argv(1) == kNpmPath && IsExactNpmLifecycleArguments(message, 2);
}

bool MayArmExactNpmNodeTransition(const gvisor::sentry::ExecveInfo& message,
                                  const char* profile,
                                  const ProcessState::GroupState& group) {
  return profile != nullptr && strcmp(profile, kProfileNPM) == 0 &&
      group.role == ProcessState::Role::kControl &&
      group.provenance == ProcessState::Provenance::kCloneChild &&
      !group.root_eligible && group.root_consumed &&
      !group.trusted_control_network_active && !group.demotion_pending &&
      !group.launch_target_pending && !group.handoff_target_pending &&
      IsExactNpmCLILauncher(message);
}

bool IsExactNpmNodeTransition(const gvisor::sentry::ExecveInfo& message,
                              const char* profile,
                              int32_t group_id,
                              const ProcessState::GroupState& group,
                              const ProcessState::ExpectedGroup& expected) {
  const auto& context = message.context_data();
  return profile != nullptr && strcmp(profile, kProfileNPM) == 0 &&
      context.thread_group_id() == group_id &&
      group.role == ProcessState::Role::kControl &&
      group.provenance == ProcessState::Provenance::kCloneChild &&
      !group.root_eligible && group.root_consumed &&
      !group.trusted_control_network_active && !group.demotion_pending &&
      !group.launch_target_pending && !group.handoff_target_pending &&
      group.npm_node_transition_pending &&
      expected.start_time_ns == context.thread_group_start_time_ns() &&
      expected.process_class == ProcessClass::kNpm && IsExactNpmNodeInterpreter(message);
}

bool IsNewGroup(const ProcessState& state, const gvisor::common::ContextData& context) {
  return state.groups.find(context.thread_group_id()) == state.groups.end();
}

bool RegisterGroup(ProcessState* state, const gvisor::common::ContextData& context,
                   ProcessState::Role role, ProcessState::Provenance provenance,
                   bool root_eligible, bool root_consumed) {
  if (state == nullptr || !ValidProcessIdentity(context)) return false;
  if (state->groups.size() >= kMaxTrackedProcessGroups) return false;
  if (!IsNewGroup(*state, context)) return false;
  state->groups.emplace(context.thread_group_id(), ProcessState::GroupState{
      context.thread_group_start_time_ns(), role, provenance, root_eligible,
      root_consumed, false, false, false, false, false, ProcessClass::kUnknown,
      ProcessState::OCIBootstrapStage::kNotOCI});
  return true;
}

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

// The helper envelope's network schema intentionally has a smaller process
// class vocabulary than process-exec diagnostics. Keep control utilities as
// bounded OTHER metadata rather than widening the Go decoder.
const char* NetworkProcessClassName(ProcessClass process_class) {
  switch (process_class) {
    case ProcessClass::kShell:
    case ProcessClass::kPython:
    case ProcessClass::kPip:
    case ProcessClass::kNode:
    case ProcessClass::kNpm:
    case ProcessClass::kArtifact:
    case ProcessClass::kUnknown:
      return ProcessClassName(process_class);
    case ProcessClass::kSleep:
    case ProcessClass::kMkdir:
    case ProcessClass::kCat:
    case ProcessClass::kChmod:
      return "OTHER";
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
    if (path == "/usr/local/lib/node_modules/npm/bin/npm-cli.js") return ProcessClass::kNpm;
  }
  if (strcmp(profile, kProfilePyPI) == 0 || strcmp(profile, kProfilePyTorchCPU) == 0 || strcmp(profile, kProfilePyTorchCU126) == 0) {
    if (path == "/usr/local/bin/python" || path == "/usr/local/bin/python3" || path == "/usr/local/bin/python3.14" || path == "python") return ProcessClass::kPython;
    if (path == "/usr/local/bin/pip" || path == "pip") return ProcessClass::kPip;
  }
  if (strcmp(profile, kProfileGitHub) == 0 && path == "/work/artifact") return ProcessClass::kArtifact;
  return ProcessClass::kUnknown;
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
  auto group = state->groups.find(context.thread_group_id());
  if (group == state->groups.end() || !SameGroup(group->second, context)) {
    result.reason = "PROCESS_PROVENANCE_UNKNOWN";
    result.parent_relation = "UNTRACKED_PARENT";
    return result;
  }
  if (group->second.role == ProcessState::Role::kArtifact) {
    result.reason = "ARTIFACT_ROLE";
    result.parent_relation = "ARTIFACT_GROUP";
    return result;
  }
  if (group->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
      group->second.root_consumed && state->expected_groups.find(context.thread_group_id()) == state->expected_groups.end()) {
    result.expected = true;
    result.parent_relation = "DIRECT_EXEC_ROOT";
    result.reason = "OTHER";
    TrackExpectedProcessGroup(context, process_class, state);
    return result;
  }
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
  if (group->second.role == ProcessState::Role::kControl) {
    result.parent_relation = group->second.provenance == ProcessState::Provenance::kCloneChild ? "CONTROL_CHILD" : "CONTROL_ROOT";
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
  auto group = state.groups.find(context.thread_group_id());
  if (group == state.groups.end() || !SameGroup(group->second, context)) return "UNKNOWN";
  if (group->second.role == ProcessState::Role::kArtifact) return "ARTIFACT_GROUP";
  if (group->second.provenance == ProcessState::Provenance::kDirectExecRoot) {
    return "DIRECT_EXEC_SESSION";
  }
  if (group->second.role == ProcessState::Role::kControl) return "CONTROL_GROUP";
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

bool IsTrustedControlNetwork(const gvisor::common::ContextData& context,
                             const ProcessState& state) {
  if (!ValidProcessIdentity(context)) return false;
  auto group = state.groups.find(context.thread_group_id());
  return group != state.groups.end() && SameGroup(group->second, context) &&
      group->second.role == ProcessState::Role::kControl &&
      group->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
      group->second.trusted_control_network_active;
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

enum class FilesystemClass { kWorkspace, kOutside, kHoneytoken, kRuntimeRoot, kHelperOnly, kUnknown };

bool IsWorkspacePath(const std::string& path, const char* profile) {
  if (profile == nullptr) return false;
  if (strcmp(profile, kProfileGitHub) == 0) return HasPrefix(path, "/work/");
  if (strcmp(profile, kProfilePyPI) == 0) {
    return path == "/tmp" || HasPrefix(path, "/tmp/") ||
        path == "/haa-site" || HasPrefix(path, "/haa-site/");
  }
  return HasPrefix(path, "/tmp/");
}

bool IsWriteCapableOpen(uint64_t flags) {
  const uint64_t access = flags & kOpenAccessMode;
  return access == kOpenWriteOnly || access == kOpenReadWrite ||
      (flags & (kOpenCreate | kOpenTruncate | kOpenAppend)) != 0;
}

// The pinned openat profile carries authoritative group/start identity. It may
// only reference an already-validated group; it never establishes process
// identity, role, or provenance.
const ProcessState::GroupState* FindFilesystemGroup(const gvisor::common::ContextData& context,
                                                     const ProcessState& state) {
  auto group = state.groups.find(context.thread_group_id());
  if (group == state.groups.end() ||
      group->second.start_time_ns != context.thread_group_start_time_ns() ||
      group->second.role == ProcessState::Role::kUnknown ||
      group->second.provenance == ProcessState::Provenance::kUnknown) {
    return nullptr;
  }
  return &group->second;
}

bool IsFilesystemControlGroup(const gvisor::common::ContextData& context, const ProcessState& state) {
  const auto* group = FindFilesystemGroup(context, state);
  return group != nullptr && group->role == ProcessState::Role::kControl;
}

bool IsExactBootstrapHelperWrite(const gvisor::common::ContextData& context,
                                 const ProcessState& state, const std::string& path,
                                 uint64_t flags) {
  const auto* group = FindFilesystemGroup(context, state);
  return group != nullptr && group->role == ProcessState::Role::kControl &&
      group->provenance == ProcessState::Provenance::kOCIRoot &&
      group->oci_bootstrap_stage == ProcessState::OCIBootstrapStage::kAwaitingDemotion &&
      state.bootstrap_active && !group->root_eligible && group->root_consumed &&
      !group->trusted_control_network_active &&
      path == "/haa-runtime/.haa-boundary.tmp" && flags == 577;
}

ProcessClass FilesystemProcessClass(const gvisor::common::ContextData& context,
                                    const ProcessState& state) {
  auto expected = state.expected_groups.find(context.thread_group_id());
  if (expected == state.expected_groups.end() ||
      expected->second.start_time_ns != context.thread_group_start_time_ns()) {
    return ProcessClass::kUnknown;
  }
  return expected->second.process_class;
}

bool MatchesLibraryNameOrVersion(const std::string& path, const char* library) {
  const size_t len = strlen(library);
  if (path == library) return true;
  if (path.size() > len && path.compare(0, len, library) == 0 && path[len] == '.') {
    for (size_t i = len + 1; i < path.size(); ++i) {
      if (!isdigit(path[i]) && path[i] != '.') return false;
    }
    return true;
  }
  return false;
}

bool IsExactLibc6(const std::string& path) {
  std::string p = path;
  if (HasPrefix(p, "/usr/lib/")) {
    p = "/lib/" + p.substr(9);
  }
  return MatchesLibraryNameOrVersion(p, "/lib/x86_64-linux-gnu/libc.so.6");
}

bool IsExactLibcapNg0(const std::string& path) {
  std::string p = path;
  if (HasPrefix(p, "/usr/lib/")) {
    p = "/lib/" + p.substr(9);
  }
  return MatchesLibraryNameOrVersion(p, "/lib/x86_64-linux-gnu/libcap-ng.so.0");
}

bool IsExactPinnedLibraryRead(const std::string& path) {
  std::string p = path;
  if (HasPrefix(p, "/usr/lib/")) {
    p = "/lib/" + p.substr(9);
  } else if (HasPrefix(p, "/usr/lib64/")) {
    p = "/lib64/" + p.substr(11);
  }
  static constexpr const char* kLibraries[] = {
      "/etc/ld.so.cache",
      "/lib64/ld-linux-x86-64.so.2",
      "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2",
      "/lib/x86_64-linux-gnu/libacl.so.1",
      "/lib/x86_64-linux-gnu/libattr.so.1",
      "/lib/x86_64-linux-gnu/libc.so.6",
      "/lib/x86_64-linux-gnu/libcap-ng.so.0",
      "/lib/x86_64-linux-gnu/libdl.so.2",
      "/lib/x86_64-linux-gnu/libgcc_s.so.1",
      "/lib/x86_64-linux-gnu/libm.so.6",
      "/lib/x86_64-linux-gnu/libpcre2-8.so.0",
      "/lib/x86_64-linux-gnu/libpthread.so.0",
      "/lib/x86_64-linux-gnu/libselinux.so.1",
      "/lib/x86_64-linux-gnu/libstdc++.so.6",
  };
  for (const char* library : kLibraries) {
    if (MatchesLibraryNameOrVersion(p, library)) return true;
  }
  return false;
}

// The seccheck pathname is lexical.  Runtime images can legitimately invoke
// Python through /usr/local/bin/../lib, so compare a strictly normalized
// absolute path.  This grants no prefix: only the exact resulting immutable
// image pathname is accepted.
bool NormalizeAbsolutePath(const std::string& path, std::string* normalized) {
  if (normalized == nullptr || path.empty() || path.front() != '/' ||
      path.find('\0') != std::string::npos) return false;
  std::vector<std::string> parts;
  size_t begin = 1;
  while (begin <= path.size()) {
    const size_t end = path.find('/', begin);
    const size_t length = (end == std::string::npos ? path.size() : end) - begin;
    const std::string part = path.substr(begin, length);
    if (part.empty() || part == ".") {
      // Empty components and explicit current-directory components are not
      // part of a deterministic pinned runtime pathname.
      return false;
    }
    if (part == "..") {
      if (parts.empty()) return false;
      parts.pop_back();
    } else {
      parts.push_back(part);
    }
    if (end == std::string::npos) break;
    begin = end + 1;
  }
  if (parts.empty()) return false;
  *normalized = "/";
  for (size_t index = 0; index < parts.size(); ++index) {
    if (index != 0) *normalized += "/";
    *normalized += parts[index];
  }
  return true;
}

bool IsExactPinnedPythonSystemLibrary(const std::string& path) {
  std::string p = path;
  if (HasPrefix(p, "/usr/lib/")) {
    p = "/lib/" + p.substr(9);
  } else if (HasPrefix(p, "/usr/lib64/")) {
    p = "/lib64/" + p.substr(11);
  }
  static constexpr const char* kLibraries[] = {
      "/etc/ld.so.cache",
      "/lib64/ld-linux-x86-64.so.2",
      "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2",
      "/lib/x86_64-linux-gnu/libbz2.so.1.0",
      "/lib/x86_64-linux-gnu/libc.so.6",
      "/lib/x86_64-linux-gnu/libcrypto.so.3",
      "/lib/x86_64-linux-gnu/libdb-5.3.so",
      "/lib/x86_64-linux-gnu/libffi.so.8",
      "/lib/x86_64-linux-gnu/libgdbm.so.6",
      "/lib/x86_64-linux-gnu/liblzma.so.5",
      "/lib/x86_64-linux-gnu/libm.so.6",
      "/lib/x86_64-linux-gnu/libncursesw.so.6",
      "/lib/x86_64-linux-gnu/libpanelw.so.6",
      "/lib/x86_64-linux-gnu/libreadline.so.8",
      "/lib/x86_64-linux-gnu/libsqlite3.so.0",
      "/lib/x86_64-linux-gnu/libssl.so.3",
      "/lib/x86_64-linux-gnu/libtinfo.so.6",
      "/lib/x86_64-linux-gnu/libuuid.so.1",
      "/lib/x86_64-linux-gnu/libz.so.1",
      "/lib/x86_64-linux-gnu/libzstd.so.1",
  };
  for (const char* library : kLibraries) {
    if (MatchesLibraryNameOrVersion(p, library)) return true;
  }
  return false;
}

bool IsPinnedRuntimeRootRead(const gvisor::common::ContextData& context,
                             const ProcessState& state,
                             const std::string& path, uint64_t flags,
                             const char* profile) {
  if (profile == nullptr || IsWriteCapableOpen(flags)) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr ||
      (group->provenance != ProcessState::Provenance::kDirectExecRoot &&
       group->provenance != ProcessState::Provenance::kCloneChild)) return false;
  std::string normalized;
  if (!NormalizeAbsolutePath(path, &normalized)) return false;
  if (strcmp(profile, kProfileGitHub) == 0) {
    return normalized == "/etc/ld.so.cache" ||
        IsExactLibc6(normalized);
  }
  if (strcmp(profile, kProfilePyPI) != 0) return false;
  if (context.process_name() != "python" && context.process_name() != "python3.14" &&
      context.process_name() != "uname" && context.process_name() != "sh") return false;
  static constexpr const char* kLoaderCandidates[] = {
      "/usr/local/bin/../lib/glibc-hwcaps/x86-64-v3/libpython3.14.so.1.0",
      "/usr/local/bin/../lib/glibc-hwcaps/x86-64-v2/libpython3.14.so.1.0",
      "/usr/local/bin/../lib/tls/x86_64/libpython3.14.so.1.0",
      "/usr/local/bin/../lib/tls/libpython3.14.so.1.0",
      "/usr/local/bin/../lib/x86_64/libpython3.14.so.1.0",
      "/usr/local/bin/../lib/libpython3.14.so.1.0",
      "/usr/local/bin/../lib/libc.so.6",
      "/usr/local/bin/../lib/libpython3.14.so.1.0._pth",
  };
  bool exact_loader_candidate = false;
  for (const char* candidate : kLoaderCandidates) {
    if (path == candidate) exact_loader_candidate = true;
  }
  if (path != normalized && !exact_loader_candidate) return false;
  if (exact_loader_candidate) return true;
  if (normalized == "/usr/local/lib/glibc-hwcaps/x86-64-v3/libpython3.14.so.1.0" ||
      normalized == "/usr/local/lib/libpython3.14.so.1.0" ||
      normalized == "/usr/local/lib/python314.zip" ||
      normalized == "/usr/local/bin/python" ||
      normalized == "/usr/local/bin/python3.14" ||
      normalized == "/usr/local/bin/pip" ||
      normalized == "/proc/sys/vm/overcommit_memory" ||
      normalized == "/etc" ||
      normalized == "/etc/localtime" ||
      normalized == "/usr/share/zoneinfo" || HasPrefix(normalized, "/usr/share/zoneinfo/") ||
      normalized == "/usr/lib/locale" || HasPrefix(normalized, "/usr/lib/locale/") ||
      normalized == "/usr/share/locale" || HasPrefix(normalized, "/usr/share/locale/") ||
      normalized == "/usr/lib/x86_64-linux-gnu/gconv" || HasPrefix(normalized, "/usr/lib/x86_64-linux-gnu/gconv/") ||
      normalized == "/usr/lib/gconv" || HasPrefix(normalized, "/usr/lib/gconv/") ||
      IsExactPinnedPythonSystemLibrary(normalized)) return true;
  const std::string pid_str = std::to_string(context.thread_group_id());
  if (normalized == "/proc/self/maps" || normalized == "/proc/" + pid_str + "/maps" ||
      normalized == "/proc/self/status" || normalized == "/proc/" + pid_str + "/status" ||
      normalized == "/proc/self/cgroup" || normalized == "/proc/" + pid_str + "/cgroup" ||
      normalized == "/proc/mounts" || normalized == "/proc/self/mounts" || normalized == "/proc/" + pid_str + "/mounts" ||
      normalized == "/proc/stat" || normalized == "/proc/self/stat" || normalized == "/proc/" + pid_str + "/stat" ||
      normalized == "/proc/version" || normalized == "/proc/version_signature" ||
      normalized == "/proc/meminfo" || normalized == "/proc/cpuinfo" ||
      normalized == "/proc/filesystems" ||
      normalized == "/dev/null" || normalized == "/dev/urandom" ||
      normalized == "/etc/nsswitch.conf" || normalized == "/etc/passwd" || normalized == "/etc/group" ||
      normalized == "/etc/pip.conf" ||
      normalized == "/etc/os-release" || normalized == "/usr/lib/os-release" ||
      normalized == "/etc/debian_version" || normalized == "/etc/issue" ||
      normalized == "/etc/lsb-release" ||
      normalized == "/etc/ssl" || HasPrefix(normalized, "/etc/ssl/") ||
      normalized == "/usr/lib/ssl" || HasPrefix(normalized, "/usr/lib/ssl/") ||
      normalized == "/etc/ca-certificates" || HasPrefix(normalized, "/etc/ca-certificates/") ||
      normalized == "/usr/share/ca-certificates" || HasPrefix(normalized, "/usr/share/ca-certificates/") ||
      normalized == "/sys/devices/system/cpu/online") return true;
  constexpr char kStdlibRoot[] = "/usr/local/lib/python3.14/";
  constexpr char kSiteRoot[] = "/usr/local/lib/python3.14/site-packages";
  if (normalized == kSiteRoot || normalized == "/usr/local/lib/python3.14") return true;
  if (HasPrefix(normalized, "/usr/local/lib/python3.14/site-packages/")) {
    // Only pip shipped by the exact image is runtime tooling. Artifact wheels
    // are installed into /haa-site or a /tmp venv and never enter this root.
    return normalized == "/usr/local/lib/python3.14/site-packages/pip" ||
        HasPrefix(normalized, "/usr/local/lib/python3.14/site-packages/pip/") ||
        normalized == "/usr/local/lib/python3.14/site-packages/pip-26.2.1.dist-info" ||
        HasPrefix(normalized, "/usr/local/lib/python3.14/site-packages/pip-26.2.1.dist-info/") ||
        normalized == "/usr/local/lib/python3.14/site-packages/README.txt";
  }
  return HasPrefix(normalized, kStdlibRoot);
}

bool IsCanonicalBootstrapProfile(const char* profile) {
  return profile != nullptr &&
      (strcmp(profile, kProfileNPM) == 0 || strcmp(profile, kProfilePyPI) == 0 ||
       strcmp(profile, kProfilePyTorchCPU) == 0 ||
       strcmp(profile, kProfilePyTorchCU126) == 0 ||
       strcmp(profile, kProfileGitHub) == 0);
}

bool IsDirectExecLoaderProfile(const char* profile) {
  return IsCanonicalBootstrapProfile(profile);
}

bool IsPinnedNpmRuntimeRead(const gvisor::common::ContextData& context,
                            const ProcessState& state, const std::string& path,
                            uint64_t flags, const char* profile) {
  if (!IsCanonicalBootstrapProfile(profile) || IsWriteCapableOpen(flags) ||
      !IsFilesystemControlGroup(context, state)) return false;
  const auto* group = FindFilesystemGroup(context, state);
  // Immutable dynamic-loader reads are common runtime mechanics for every
  // supported profile. They are accepted only for an already validated
  // CONTROL group and never create or modify process trust.
  if (IsExactPinnedLibraryRead(path)) return true;
  if (HasPrefix(path, "/usr/lib/locale/") || HasPrefix(path, "/usr/share/locale/") ||
      HasPrefix(path, "/usr/lib/x86_64-linux-gnu/gconv/") || HasPrefix(path, "/usr/lib/gconv/")) return true;
  if (context.process_name() == "setpriv" &&
      (path == "/proc/sys/kernel/cap_last_cap" ||
       path == "/etc/nsswitch.conf" || path == "/etc/passwd" ||
       path == "/etc/group")) return true;
  if (group->provenance == ProcessState::Provenance::kDirectExecRoot &&
      group->demotion_pending && !group->root_eligible &&
      group->root_consumed && !group->trusted_control_network_active &&
      context.process_name() == "haa-boundary" &&
      (path == "/etc/ld.so.cache" ||
       IsExactLibc6(path) ||
       path == "/haa-runtime/haa-boundary")) return true;
  auto direct_parent = state.groups.find(context.parent_thread_group_id());
  const bool exact_direct_control_child =
      group->role == ProcessState::Role::kControl &&
      group->provenance == ProcessState::Provenance::kCloneChild &&
      !group->root_eligible && group->root_consumed &&
      !group->trusted_control_network_active &&
      direct_parent != state.groups.end() &&
      direct_parent->second.role == ProcessState::Role::kControl &&
      direct_parent->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
      !direct_parent->second.root_eligible && direct_parent->second.root_consumed;
  if (exact_direct_control_child && context.process_name() == "id" &&
      path == "/proc/filesystems") return true;
  if (exact_direct_control_child && context.process_name() == "grep" &&
      (path == "/proc/1/status" || path == "/proc/self/status" ||
       path == "/proc/self/maps" ||
       path == "/proc/" + std::to_string(context.thread_group_id()) + "/status" ||
       path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps")) return true;
  const bool existing_common_profile = strcmp(profile, kProfileNPM) == 0 ||
      strcmp(profile, kProfileGitHub) == 0;
  const bool exact_oci_bootstrap_group = state.bootstrap_active &&
      state.bootstrap_group_set &&
      ((group->provenance == ProcessState::Provenance::kOCIRoot &&
        context.thread_group_id() == state.bootstrap_group_id) ||
       (group->provenance == ProcessState::Provenance::kCloneChild &&
        context.parent_thread_group_id() == state.bootstrap_group_id));
  if (!existing_common_profile && !exact_oci_bootstrap_group) return false;
  const ProcessClass process_class = FilesystemProcessClass(context, state);
  if (path == "/haa-runtime/haa-boundary" || path == "/haa-runtime" ||
      path == "/dev/null") return true;
  if (group->provenance == ProcessState::Provenance::kOCIRoot && state.bootstrap_active &&
      path == "/usr/local/bin/docker-entrypoint.sh") return true;
  if (group->provenance == ProcessState::Provenance::kCloneChild &&
      context.process_name() == "chown" &&
      (path == "/etc/nsswitch.conf" || path == "/etc/passwd" ||
       path == "/etc/group")) return true;
  if (group->provenance == ProcessState::Provenance::kCloneChild &&
      (context.process_name() == "chown" || context.process_name() == "chmod" ||
       context.process_name() == "mv" || context.process_name() == "mkdir" ||
       context.process_name() == "id") && path == "/proc/filesystems") return true;
  if (group->provenance == ProcessState::Provenance::kCloneChild &&
       context.process_name() == "grep" &&
      (path == "/proc/1/status" || path == "/proc/self/status" ||
       path == "/proc/self/maps" ||
       path == "/proc/" + std::to_string(context.thread_group_id()) + "/status" ||
       path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps")) return true;
  if (strcmp(profile, kProfileNPM) != 0) return false;
  if (process_class == ProcessClass::kNpm || process_class == ProcessClass::kNode) {
    if (HasPrefix(path, "/usr/local/lib/node_modules/npm/") ||
        path == "/usr/local/bin/node" || path == "/usr/local/bin/npm" ||
        path == "/usr/local/etc/npmrc" || path == "/etc/ssl/openssl.cnf") return true;
  }
  if ((process_class == ProcessClass::kNpm || process_class == ProcessClass::kNode) &&
      (path == "/etc/localtime" || HasPrefix(path, "/usr/share/zoneinfo/") ||
       path == "/etc/nsswitch.conf" ||
       path == "/etc/resolv.conf" || path == "/etc/netsvc.conf" ||
       path == "/etc/svc.conf" || path == "/usr/bin/ldd")) {
    return true;
  }
  if (process_class != ProcessClass::kNode ||
      group->provenance != ProcessState::Provenance::kCloneChild) return false;
  const std::string cgroup = "/sys/fs/cgroup/memory/" + context.container_id();
  return path == "/proc/version_signature" || path == "/proc/meminfo" ||
      path == "/proc/self/cgroup" || path == "/proc/self/maps" ||
      path == "/proc/" + std::to_string(context.thread_group_id()) + "/cgroup" ||
      path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps" ||
      path == "/proc/sys/vm/overcommit_memory" ||
      path == cgroup + "/memory.soft_limit_in_bytes" ||
      path == cgroup + "/memory.limit_in_bytes" ||
      path == "/sys/fs/cgroup/memory/memory.soft_limit_in_bytes" ||
      path == "/sys/fs/cgroup/memory/memory.limit_in_bytes" ||
      path == "/sys/devices/system/cpu/online";
}

bool IsExactHAAELFHandoffDemotionRead(
    const gvisor::common::ContextData& context, const ProcessState& state,
    const std::string& path, uint64_t flags, const char* profile) {
  if (profile == nullptr || (strcmp(profile, kProfileGitHub) != 0 && strcmp(profile, kProfilePyPI) != 0) ||
      IsWriteCapableOpen(flags)) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || group->role != ProcessState::Role::kArtifact ||
      group->provenance != ProcessState::Provenance::kDirectExecRoot ||
      group->root_eligible || !group->root_consumed ||
      group->trusted_control_network_active || !group->handoff_target_pending) {
    return false;
  }
  if (group->demotion_pending) {
    return context.process_name() == "haa-boundary" &&
        (path == "/etc/ld.so.cache" ||
         IsExactLibc6(path) ||
         path == "/haa-runtime/haa-boundary");
  }
  if (context.process_name() != "setpriv") return false;
  return path == "/etc/ld.so.cache" ||
      IsExactLibcapNg0(path) ||
      IsExactLibc6(path) ||
      path == "/proc/sys/kernel/cap_last_cap" ||
      path == "/etc/nsswitch.conf" || path == "/etc/passwd" ||
      path == "/etc/group" ||
      path == "/proc/" + std::to_string(context.thread_group_id()) + "/status";
}

FilesystemClass ClassifyFilesystemOpen(const gvisor::syscall::Open& message,
                                       const ProcessState& state, const char* profile) {
  const std::string& path = message.pathname();
  // This precedence is deliberately before every runtime/helper exception:
  // a decoy access is always actionable, irrespective of role or image.
  if (IsHoneytoken(path)) return FilesystemClass::kHoneytoken;
  if (FindFilesystemGroup(message.context_data(), state) == nullptr) {
    const auto& context = message.context_data();
    const bool exact_pre_sentry_loader = IsDirectExecLoaderProfile(profile) &&
        context.thread_group_id() > 0 && context.thread_group_start_time_ns() > 0 &&
        context.parent_thread_group_id() == 0 && context.is_exec_session() &&
        (context.process_name() == "haa-boundary" || context.process_name() == "sh" || context.process_name() == "dash") && !IsWriteCapableOpen(message.flags()) &&
        (path == "/etc/ld.so.cache" || IsExactLibc6(path) ||
         path == "/haa-runtime/haa-boundary");
    return exact_pre_sentry_loader ? FilesystemClass::kHelperOnly : FilesystemClass::kUnknown;
  }
  const auto* tracked = FindFilesystemGroup(message.context_data(), state);
  if (IsExactHAAELFHandoffDemotionRead(message.context_data(), state, path,
                                       message.flags(), profile)) {
    return FilesystemClass::kHelperOnly;
  }
  const bool exact_self_status = tracked->role == ProcessState::Role::kControl &&
      message.context_data().process_name() == "setpriv" && !IsWriteCapableOpen(message.flags()) &&
      path == "/proc/" + std::to_string(message.context_data().thread_group_id()) + "/status";
  if (exact_self_status) return FilesystemClass::kHelperOnly;
  const bool exact_handoff_validation = tracked->role == ProcessState::Role::kArtifact &&
      (tracked->provenance == ProcessState::Provenance::kCloneChild ||
       tracked->provenance == ProcessState::Provenance::kDirectExecRoot) &&
      !tracked->root_eligible && tracked->root_consumed &&
      !tracked->trusted_control_network_active &&
      message.context_data().process_name() == "haa-boundary" &&
      !IsWriteCapableOpen(message.flags()) &&
      (path == "/haa-runtime/haa-boundary" || path == "/proc/self/status" ||
       path == "/etc/ld.so.cache" || IsExactLibc6(path) ||
       path == "/proc/" + std::to_string(message.context_data().thread_group_id()) + "/status");
  if (exact_handoff_validation) return FilesystemClass::kHelperOnly;
  const bool exact_artifact_shell_loader = tracked->role == ProcessState::Role::kArtifact &&
      tracked->provenance == ProcessState::Provenance::kCloneChild &&
      !tracked->root_eligible && tracked->root_consumed &&
      !tracked->trusted_control_network_active &&
      message.context_data().process_name() == "sh" &&
      !IsWriteCapableOpen(message.flags()) &&
      (path == "/etc/ld.so.cache" || IsExactLibc6(path));
  if (exact_artifact_shell_loader) return FilesystemClass::kHelperOnly;
  if (IsWorkspacePath(path, profile)) return FilesystemClass::kWorkspace;
  if (IsClearlyOutsideWorkspace(path) ||
      (IsWriteCapableOpen(message.flags()) &&
       !IsExactBootstrapHelperWrite(message.context_data(), state, path, message.flags()))) {
    return FilesystemClass::kOutside;
  }
  if (IsExactBootstrapHelperWrite(message.context_data(), state, path, message.flags()) ||
      IsPinnedRuntimeRootRead(message.context_data(), state, path,
                              message.flags(), profile) ||
      IsPinnedNpmRuntimeRead(message.context_data(), state, path, message.flags(), profile)) {
    return profile != nullptr && strcmp(profile, kProfilePyPI) == 0 &&
            IsPinnedRuntimeRootRead(message.context_data(), state, path,
                                    message.flags(), profile)
        ? FilesystemClass::kRuntimeRoot : FilesystemClass::kHelperOnly;
  }
  return FilesystemClass::kUnknown;
}

bool IsNormalizedAbsolutePath(const std::string& path) {
  if (path.empty() || path.size() > kMaxTopologyMountpointBytes || path[0] != '/') return false;
  if (path == "/") return true;
  if (path.back() == '/' || path.find("//") != std::string::npos) return false;
  size_t begin = 1;
  while (begin < path.size()) {
    const size_t end = path.find('/', begin);
    const std::string component = path.substr(begin, end == std::string::npos ? std::string::npos : end - begin);
    if (component.empty() || component == "." || component == "..") return false;
    begin = end == std::string::npos ? path.size() : end + 1;
  }
  return true;
}

bool ParseExpectedTopology(const std::string& encoded, std::vector<ExpectedMount>* expected) {
  if (expected == nullptr || encoded.empty() || encoded.size() > 4096) return false;
  size_t start = 0;
  while (start < encoded.size()) {
    const size_t end = encoded.find(';', start);
    const std::string entry = encoded.substr(start, end == std::string::npos ? std::string::npos : end - start);
    std::vector<std::string> fields;
    size_t field_start = 0;
    while (field_start <= entry.size()) {
      const size_t field_end = entry.find('|', field_start);
      fields.push_back(entry.substr(field_start, field_end == std::string::npos ? std::string::npos : field_end - field_start));
      if (field_end == std::string::npos) break;
      field_start = field_end + 1;
    }
    if (fields.size() != 8 || !IsNormalizedAbsolutePath(fields[0]) || !IsNormalizedAbsolutePath(fields[2]) ||
        fields[1].empty() || fields[1].size() > 32 || fields[3].size() > kMaxTopologyFilesystemTypeBytes ||
        (fields[4] != "0" && fields[4] != "1") || (fields[5] != "0" && fields[5] != "1") ||
        (fields[6] != "0" && fields[6] != "1") || (fields[7] != "0" && fields[7] != "1")) return false;
    expected->push_back(ExpectedMount{fields[0], fields[1], fields[2], fields[3], fields[4] == "1", fields[5] == "1", fields[6] == "1", fields[7] == "1"});
    if (expected->size() > kMaxTopologyMounts) return false;
    if (end == std::string::npos) break;
    start = end + 1;
  }
  return !expected->empty();
}

bool ParseControlRecord(const char* payload, size_t size, std::map<std::string, ProfileRegistration>* profiles) {
  std::string body(payload, size);
  const std::string id_key = "\"container_id\":\"";
  const std::string profile_key = "\"profile\":\"";
  const std::string topology_key = "\"expected_topology\":\"";
  const size_t id_start = body.find(id_key);
  const size_t profile_start = body.find(profile_key);
  const size_t topology_start = body.find(topology_key);
  if (id_start == std::string::npos || profile_start == std::string::npos || topology_start == std::string::npos) return false;
  const size_t id_end = body.find('"', id_start + id_key.size());
  const size_t profile_end = body.find('"', profile_start + profile_key.size());
  const size_t topology_end = body.find('"', topology_start + topology_key.size());
  if (id_end == std::string::npos || profile_end == std::string::npos || topology_end == std::string::npos) return false;
  const std::string id = body.substr(id_start + id_key.size(), id_end - id_start - id_key.size());
  const std::string profile = body.substr(profile_start + profile_key.size(), profile_end - profile_start - profile_key.size());
  const std::string topology = body.substr(topology_start + topology_key.size(), topology_end - topology_start - topology_key.size());
  if (!ValidContainerID(id) || (profile != kProfileNPM && profile != kProfilePyPI && profile != kProfilePyTorchCPU && profile != kProfilePyTorchCU126 && profile != kProfileGitHub)) return false;
  if (profiles->find(id) != profiles->end()) return false;
  std::vector<ExpectedMount> expected;
  if (!ParseExpectedTopology(topology, &expected)) return false;
  (*profiles)[id] = ProfileRegistration{profile, std::move(expected)};
  return true;
}

bool ParseContainerStart(const char* payload, size_t payload_size, int output,
                         std::string* container_id, ProcessState* state,
                         const char** reason) {
  if (state == nullptr) return false;
  gvisor::container::Start message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  if (!ValidateContextContainer(message.context_data(), container_id, reason) ||
      !ValidProcessIdentity(message.context_data()) ||
      message.context_data().is_exec_session() ||
      message.context_data().parent_thread_group_id() != 0) {
    *reason = "CONTAINER_ROOT_INVALID";
    return false;
  }
  if (!RegisterGroup(state, message.context_data(), ProcessState::Role::kControl,
                     ProcessState::Provenance::kOCIRoot, false, true)) {
    *reason = "CONTAINER_ROOT_DUPLICATE";
    return false;
  }
  state->groups.find(message.context_data().thread_group_id())->second.oci_bootstrap_stage =
      IsExactOCIBootstrapContainerStart(message)
          ? ProcessState::OCIBootstrapStage::kAwaitingDemotion
          : ProcessState::OCIBootstrapStage::kAwaitingBootstrapShell;
  state->bootstrap_group_set = true;
  state->bootstrap_group_id = message.context_data().thread_group_id();
  state->bootstrap_group_start_time_ns = message.context_data().thread_group_start_time_ns();
  return Send(output, *container_id, "container-start");
}

bool ParseSentryClone(const char* payload, size_t payload_size,
                      int output, std::string* container_id, ProcessState* state,
                      const char** reason) {
  (void)output;
  if (state == nullptr) return false;
  gvisor::sentry::CloneInfo message;
  if (!message.ParseFromArray(payload, payload_size) ||
      !ValidateContextContainer(message.context_data(), container_id, reason) ||
      !ValidProcessIdentity(message.context_data()) ||
      message.created_thread_group_id() <= 0 ||
      message.created_thread_start_time_ns() <= 0) {
    *reason = "CLONE_PROVENANCE_INVALID";
    return false;
  }
  const int32_t creator_group = message.context_data().thread_group_id();
  const int32_t child_group = message.created_thread_group_id();
  auto creator = state->groups.find(creator_group);
  if (creator == state->groups.end() || !SameGroup(creator->second, message.context_data())) {
    *reason = "CLONE_PROVENANCE_INVALID";
    return false;
  }
  if ((message.flags() & kCloneThread) != 0) {
    if (child_group != creator_group) *reason = "CLONE_PROVENANCE_INVALID";
    return child_group == creator_group;
  }
  if (state->groups.find(child_group) != state->groups.end() ||
      state->groups.size() >= kMaxTrackedProcessGroups) {
    *reason = state->groups.size() >= kMaxTrackedProcessGroups ? "PROCESS_STATE_LIMIT" : "CLONE_PROVENANCE_INVALID";
    return false;
  }
  state->groups.emplace(child_group, ProcessState::GroupState{
      message.created_thread_start_time_ns(), creator->second.role,
      ProcessState::Provenance::kCloneChild, false, true, false, false, false, false,
      false, ProcessClass::kUnknown, ProcessState::OCIBootstrapStage::kNotOCI});
  return true;
}

size_t MaximumRecords(const char* profile) {
  if (profile == nullptr) return kMaxNormalizedRecordsPerConnection;
  if (strcmp(profile, kProfilePyTorchCPU) == 0) return kMaxPyTorchCPURecordsPerConnection;
  if (strcmp(profile, kProfilePyTorchCU126) == 0) return kMaxPyTorchCU126RecordsPerConnection;
  return kMaxNormalizedRecordsPerConnection;
}

bool DrainProfiles(int control, std::map<std::string, ProfileRegistration>* profiles) {
  char control_message[4096];
  ssize_t control_size;
  while ((control_size = recv(control, control_message, sizeof(control_message), MSG_DONTWAIT)) > 0) {
    if (!ParseControlRecord(control_message, static_cast<size_t>(control_size), profiles)) return false;
  }
  return control_size == 0 || (control_size < 0 && (errno == EAGAIN || errno == EWOULDBLOCK));
}

const ProfileRegistration* AwaitProfile(int control, const std::string& container_id, std::map<std::string, ProfileRegistration>* profiles) {
  for (int waited = 0; waited < kProfileRegistrationWaitMilliseconds; waited += 50) {
    if (!DrainProfiles(control, profiles)) return nullptr;
    auto profile = profiles->find(container_id);
    if (profile != profiles->end()) return &profile->second;
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
  const int32_t group_id = message.context_data().thread_group_id();
  const BoundaryMode boundary_mode = BoundaryInvocation(message);
  auto group = candidate.groups.find(group_id);
  bool new_direct_root = false;
  if (group != candidate.groups.end() && !SameGroup(group->second, message.context_data())) {
    *reason = "PROCESS_IDENTITY_REUSED";
    return false;
  }
  if (group == candidate.groups.end()) {
    if (!message.context_data().is_exec_session() || message.context_data().parent_thread_group_id() != 0 ||
        boundary_mode == BoundaryMode::kNone ||
        candidate.groups.size() >= kMaxTrackedProcessGroups ||
        !RegisterGroup(&candidate, message.context_data(), ProcessState::Role::kControl,
                       ProcessState::Provenance::kDirectExecRoot, true, false)) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN";
      return false;
    }
    group = candidate.groups.find(group_id);
    new_direct_root = true;
  }
  if (group->second.provenance == ProcessState::Provenance::kOCIRoot) {
    if (group->second.role != ProcessState::Role::kControl ||
        !candidate.bootstrap_active || group->second.trusted_control_network_active) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN";
      return false;
    }
    if (group->second.oci_bootstrap_stage == ProcessState::OCIBootstrapStage::kAwaitingBootstrapShell) {
      if (!IsExactOCIBootstrapShell(message, group->second, candidate)) {
        *reason = "PROCESS_PROVENANCE_UNKNOWN";
        return false;
      }
      group->second.oci_bootstrap_stage = ProcessState::OCIBootstrapStage::kAwaitingDemotion;
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output, *container_id, "process-exec-expected");
    }
    if (group->second.oci_bootstrap_stage == ProcessState::OCIBootstrapStage::kAwaitingDemotion) {
      if (!IsExactOCIBootstrapDemotion(message)) {
        *reason = "PROCESS_PROVENANCE_UNKNOWN";
        return false;
      }
      group->second.oci_bootstrap_stage = ProcessState::OCIBootstrapStage::kAwaitingSleep;
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output, *container_id, "process-exec-expected");
    }
    if (group->second.oci_bootstrap_stage == ProcessState::OCIBootstrapStage::kAwaitingSleep) {
      if (!IsExactOCIBootstrapSleep(message)) {
        *reason = "PROCESS_PROVENANCE_UNKNOWN";
        return false;
      }
      group->second.oci_bootstrap_stage = ProcessState::OCIBootstrapStage::kComplete;
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output, *container_id, "process-exec-expected");
    }
    if (group->second.oci_bootstrap_stage == ProcessState::OCIBootstrapStage::kComplete &&
        (IsExactOCIBootstrapDemotion(message) || IsExactOCIBootstrapSleep(message))) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN";
      return false;
    }
  }
  if (IsExactOCIBootstrapShellIdentity(message)) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN";
    return false;
  }
  if (group->second.role == ProcessState::Role::kControl &&
      boundary_mode == BoundaryMode::kLaunch) {
    if (group->second.provenance != ProcessState::Provenance::kDirectExecRoot ||
        !group->second.root_eligible || group->second.root_consumed || group->second.launch_target_pending) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN";
      return false;
    }
    group->second.root_eligible = false;
    group->second.root_consumed = true;
    group->second.demotion_pending = true;
    ApplyExecCloexec(&candidate, group_id);
    process_state->groups = candidate.groups;
    process_state->fd_states = candidate.fd_states;
    return Send(output, *container_id, "process-exec-expected");
  }
  if (boundary_mode == BoundaryMode::kHandoff || boundary_mode == BoundaryMode::kPythonHandoff ||
      boundary_mode == BoundaryMode::kELFHandoff) {
    if (group->second.role == ProcessState::Role::kControl) {
      if (group->second.provenance == ProcessState::Provenance::kDirectExecRoot) {
        if (new_direct_root) {
          // A Docker exec may enter the verified handoff directly. It creates
          // one root provenance record but never activates CONTROL target
          // trust, then consumes eligibility as it irreversibly demotes.
          group->second.root_eligible = false;
          group->second.root_consumed = true;
        } else if (!group->second.root_consumed || group->second.launch_target_pending ||
                   group->second.demotion_pending) {
          *reason = "PROCESS_PROVENANCE_UNKNOWN";
          return false;
        }
      }
      group->second.role = ProcessState::Role::kArtifact;
      group->second.trusted_control_network_active = false;
      group->second.demotion_pending = boundary_mode != BoundaryMode::kHandoff;
      group->second.handoff_target_pending = boundary_mode != BoundaryMode::kHandoff;
      group->second.handoff_target_class = boundary_mode == BoundaryMode::kPythonHandoff
          ? ProcessClass::kPython : ProcessClass::kArtifact;
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output, *container_id, "process-exec-expected");
    }
  }
  if (group->second.demotion_pending) {
    if (!IsExactSetprivDemotion(message) || IsExactOCIBootstrapDemotion(message)) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN";
      return false;
    }
    group->second.demotion_pending = false;
    if (group->second.role == ProcessState::Role::kControl) {
      group->second.launch_target_pending = true;
    }
    ApplyExecCloexec(&candidate, group_id);
    process_state->groups = candidate.groups;
    process_state->fd_states = candidate.fd_states;
    return Send(output, *container_id, "process-exec-expected");
  }
  if (IsExactSetprivDemotion(message)) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN";
    return false;
  }
  if (group->second.role == ProcessState::Role::kControl &&
      group->second.launch_target_pending) {
    const ProcessClass process_class = ProcessClassForPath(message.binary_path(), profile);
    const TrackResult tracked_result = TrackExpectedProcessGroup(message.context_data(), process_class, &candidate);
    if (tracked_result != TrackResult::kTracked) {
      *reason = TrackFailureReason(tracked_result);
      return false;
    }
    group->second.launch_target_pending = false;
    group->second.trusted_control_network_active = true;
    ApplyExecCloexec(&candidate, group_id);
    process_state->groups = candidate.groups;
    process_state->expected_groups = candidate.expected_groups;
    process_state->fd_states = candidate.fd_states;
    return Send(output, *container_id, "process-exec-expected");
  }
  if (group->second.role == ProcessState::Role::kArtifact) {
    const ProcessClass process_class = ProcessClassForPath(message.binary_path(), profile);
    if (group->second.handoff_target_pending && process_class == group->second.handoff_target_class) {
      group->second.handoff_target_pending = false;
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output, *container_id, "process-exec-expected");
    }
    ApplyExecCloexec(&candidate, group_id);
    const Attribution attribution{"SENTRY_EXEC", nullptr, nullptr,
                                  ProcessClassName(process_class), "ARTIFACT_ROLE", "ARTIFACT_GROUP"};
    process_state->groups = candidate.groups;
    process_state->fd_states = candidate.fd_states;
    return Send(output, *container_id, "process-exec-unexpected", nullptr, &attribution);
  }
  auto tracked_group = candidate.expected_groups.find(group_id);
  if (tracked_group != candidate.expected_groups.end() &&
      IsExactNpmNodeTransition(message, profile, group_id, group->second, tracked_group->second)) {
    tracked_group->second.process_class = ProcessClass::kNode;
    group->second.npm_node_transition_pending = false;
    ApplyExecCloexec(&candidate, group_id);
    process_state->expected_groups = candidate.expected_groups;
    process_state->groups = candidate.groups;
    process_state->fd_states = candidate.fd_states;
    return Send(output, *container_id, "process-exec-expected");
  }
  // A later successful image transition in the direct root cannot retain the
  // narrow trusted-control network exception.
  if (group->second.provenance == ProcessState::Provenance::kDirectExecRoot) {
    group->second.trusted_control_network_active = false;
  }
  const ProcessClassification classification = IsExpectedProcess(message.binary_path(), message.context_data(), profile, &candidate);
  process_state->bootstrap_active = candidate.bootstrap_active;
  process_state->bootstrap_group_set = candidate.bootstrap_group_set;
  process_state->bootstrap_group_id = candidate.bootstrap_group_id;
  process_state->bootstrap_group_start_time_ns = candidate.bootstrap_group_start_time_ns;
  process_state->expected_groups = std::move(candidate.expected_groups);
  process_state->launch_roots = std::move(candidate.launch_roots);
  process_state->groups = std::move(candidate.groups);
  const int64_t group_start_time_ns = message.context_data().thread_group_start_time_ns();
  if (strcmp(classification.parent_relation, "DIRECT_EXEC_ROOT") == 0 && classification.expected) {
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
  auto updated_group = process_state->groups.find(group_id);
  if (classification.expected && updated_group != process_state->groups.end() &&
      MayArmExactNpmNodeTransition(message, profile, updated_group->second)) {
    updated_group->second.npm_node_transition_pending = true;
  }
  ApplyExecCloexec(process_state, group_id);
  if (classification.expected) return Send(output, *container_id, "process-exec-expected");
  const Attribution attribution{"SENTRY_EXEC", nullptr, nullptr,
                                ProcessClassName(classification.process_class), classification.reason,
                                classification.parent_relation};
  return Send(output, *container_id, "process-exec-unexpected", nullptr, &attribution);
}

bool ParseOpenAndSend(const char* payload, size_t payload_size, int output, std::string* container_id,
                      ProcessState* state, NormalizedCounts* counts, const char** reason) {
  if (state == nullptr || counts == nullptr) return false;
  gvisor::syscall::Open message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  if (!ValidateContextContainer(message.context_data(), container_id, reason) ||
      message.context_data().thread_id() <= 0 || message.context_data().thread_start_time_ns() <= 0) {
    *reason = "STREAM_FAULT"; return false;
  }
  const auto group = state->groups.find(message.context_data().thread_group_id());
  if (group != state->groups.end()) {
    if (!SameGroup(group->second, message.context_data())) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN"; return false;
    }
  } else {
    if (!message.context_data().is_exec_session() || message.context_data().parent_thread_group_id() != 0) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN"; return false;
    }
  }
  const auto key = std::make_pair(message.context_data().thread_id(), message.context_data().thread_start_time_ns());
  if (state->pending_opens.find(key) != state->pending_opens.end() || state->pending_opens.size() >= kMaxTrackedFileDescriptorsPerGroup) {
    *reason = "STREAM_FAULT"; return false;
  }
  bool emitted = false;
  // Only pre-resolution facts are actionable here. Relative pathname text is
  // deliberately never used as a workspace/runtime target identity.
  if (IsHoneytoken(message.pathname())) {
    if (!Send(output, *container_id, "honeytoken-access")) return false;
    ++counts->immediate_records;
    emitted = true;
  } else if (IsWriteCapableOpen(message.flags())) {
    if (!Send(output, *container_id, "filesystem-outside-workspace")) return false;
    ++counts->immediate_records;
    emitted = true;
  }
  state->pending_opens.emplace(key, ProcessState::PendingOpen{message.context_data().thread_group_id(),
      message.context_data().thread_group_start_time_ns(), message.sysno(), message.flags(), emitted});
  return true;
}

bool ParseOpenResultAndSend(const char* payload, size_t payload_size, int output, std::string* container_id,
                            const char* profile, ProcessState* state, NormalizedCounts* counts,
                            const TopologyState& topology, const char** reason) {
  if (profile == nullptr || state == nullptr || counts == nullptr || !topology.sealed) {
    return false;
  }
  gvisor::syscall::OpenResult result;
  if (!result.ParseFromArray(payload, payload_size) || !ValidateContextContainer(result.context_data(), container_id, reason) ||
      result.context_data().thread_id() <= 0 || result.context_data().thread_start_time_ns() <= 0) {
    *reason = "STREAM_FAULT"; return false;
  }
  const auto group = state->groups.find(result.context_data().thread_group_id());
  if (group != state->groups.end()) {
    if (!SameGroup(group->second, result.context_data())) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN"; return false;
    }
  } else {
    if (!result.context_data().is_exec_session() || result.context_data().parent_thread_group_id() != 0) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN"; return false;
    }
  }
  const auto key = std::make_pair(result.context_data().thread_id(), result.context_data().thread_start_time_ns());
  const auto pending = state->pending_opens.find(key);
  if (pending == state->pending_opens.end() || pending->second.thread_group_id != result.context_data().thread_group_id() ||
      pending->second.thread_group_start_time_ns != result.context_data().thread_group_start_time_ns() ||
      pending->second.sysno != result.sysno() ||
      (pending->second.flags | kOpenLargefile) != (result.flags() | kOpenLargefile)) {
    *reason = "STREAM_FAULT"; return false;
  }
  const bool early = pending->second.early_finding_emitted;
  state->pending_opens.erase(pending);
  if (!result.success()) {
    if (result.errorno() == 0 || !result.resolved_pathname().empty() || result.mount_id() != 0) {
      *reason = "STREAM_FAULT"; return false;
    }
    return true;
  }
  if (result.errorno() != 0 || result.mount_id() == 0 || !IsNormalizedAbsolutePath(result.resolved_pathname())) {
    *reason = "STREAM_FAULT"; return false;
  }
  const auto anchor = topology.anchors.find(result.mount_id());
  if (anchor == topology.anchors.end() || !IsAtOrBelowMountpoint(result.resolved_pathname(), anchor->second.mountpoint)) {
    *reason = "STREAM_FAULT"; return false;
  }
  for (const auto& candidate : topology.anchors) {
    if (candidate.first != anchor->first && candidate.second.mountpoint.size() > anchor->second.mountpoint.size() &&
        IsAtOrBelowMountpoint(result.resolved_pathname(), candidate.second.mountpoint)) {
      *reason = "STREAM_FAULT"; return false;
    }
  }
  if (early) return true;
  if (IsHoneytoken(result.resolved_pathname())) {
    if (!Send(output, *container_id, "honeytoken-access")) return false;
    ++counts->immediate_records; return true;
  }
  if (anchor->second.mount_class == "workspace") {
    if (IsWriteCapableOpen(result.flags())) {
      *reason = "STREAM_FAULT"; return false;
    }
    if (counts->workspace_access < kMaxNormalizedObservationCount) ++counts->workspace_access;
    return true;
  }
  gvisor::syscall::Open final_open;
  *final_open.mutable_context_data() = result.context_data();
  final_open.set_pathname(result.resolved_pathname()); final_open.set_flags(result.flags()); final_open.set_sysno(result.sysno());
  switch (ClassifyFilesystemOpen(final_open, *state, profile)) {
    case FilesystemClass::kRuntimeRoot: if (counts->runtime_root_access < kMaxNormalizedObservationCount) ++counts->runtime_root_access; return true;
    case FilesystemClass::kHelperOnly: return true;
    case FilesystemClass::kOutside: if (!Send(output, *container_id, "filesystem-outside-workspace")) return false; ++counts->immediate_records; return true;
    default:
      *reason = "STREAM_FAULT"; return false;
  }
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
  auto group_state = process_state->groups.find(message.context_data().thread_group_id());
  if (group_state == process_state->groups.end() || !SameGroup(group_state->second, message.context_data())) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN";
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
      const Attribution attribution{"SOCKET", "PACKET", relation, NetworkProcessClassName(process_class), nullptr, nullptr};
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
  auto group_state = state->groups.find(message.context_data().thread_group_id());
  if (group_state == state->groups.end() || !SameGroup(group_state->second, message.context_data())) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN";
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
  const Attribution attribution{source, family, relation, NetworkProcessClassName(process_class), nullptr, nullptr};
  const bool trusted = IsTrustedControlNetwork(message.context_data(), *state);
  return Send(output, *container_id, trusted ? "trusted-control-network" : "network-attempt", nullptr, &attribution);
}

bool ParseConnectAndSend(const char* payload, size_t payload_size, int output, std::string* container_id, const char* profile, const ProcessState& process_state, const char** reason) {
  gvisor::syscall::Connect message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  if (!ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  auto group_state = process_state.groups.find(message.context_data().thread_group_id());
  if (group_state == process_state.groups.end() || !SameGroup(group_state->second, message.context_data())) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN";
    return false;
  }
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
      { const Attribution attribution{"CONNECT", "INET", relation, NetworkProcessClassName(process_class), nullptr, nullptr};
        return Send(output, *container_id, IsTrustedControlNetwork(message.context_data(), process_state) ? "trusted-control-network" : "network-attempt", nullptr, &attribution); }
    case AF_INET6:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_INET6_INVALID_LENGTH"; return false; }
      { const Attribution attribution{"CONNECT", "INET6", relation, NetworkProcessClassName(process_class), nullptr, nullptr};
        return Send(output, *container_id, IsTrustedControlNetwork(message.context_data(), process_state) ? "trusted-control-network" : "network-attempt", nullptr, &attribution); }
    case kLinuxAFNetlink:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_NETLINK_INVALID_LENGTH"; return false; }
      return true;
    case kLinuxAFPacket:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_PACKET_INVALID_LENGTH"; return false; }
      { const Attribution attribution{"CONNECT", "PACKET", relation, NetworkProcessClassName(process_class), nullptr, nullptr}; return Send(output, *container_id, "network-attempt", nullptr, &attribution); }
    default:
      (void)classification;
      *reason = "CONNECT_UNKNOWN_FAMILY";
      return false;
  }
}

bool IsAtOrBelowMountpoint(const std::string& path, const std::string& mountpoint) {
  if (mountpoint == "/") return IsNormalizedAbsolutePath(path);
  return path == mountpoint || (path.size() > mountpoint.size() && path.compare(0, mountpoint.size(), mountpoint) == 0 && path[mountpoint.size()] == '/');
}

bool ParseTopologySnapshot(const char* payload, size_t payload_size, int output,
                           std::string* container_id, TopologyState* topology,
                           const char** reason) {
  if (topology == nullptr || topology->snapshot_seen || topology->sealed || payload_size > kMaxTopologySnapshotBytes) {
    *reason = "TOPOLOGY_INVALID"; return false;
  }
  gvisor::sentry::MountTopologySnapshot message;
  if (!message.ParseFromArray(payload, payload_size) || message.ByteSizeLong() > kMaxTopologySnapshotBytes ||
      !ValidateContextContainer(message.context_data(), container_id, reason) ||
      !message.snapshot_complete() || message.mount_namespace_id() == 0 ||
      message.mounts_size() <= 0 || static_cast<size_t>(message.mounts_size()) > kMaxTopologyMounts) {
    *reason = "TOPOLOGY_INVALID"; return false;
  }
  std::map<uint64_t, const gvisor::sentry::MountTopologyEntry*> mounts;
  std::map<std::string, uint64_t> mountpoints;
  uint64_t root_id = 0;
  for (const auto& mount : message.mounts()) {
    if (mount.mount_id() == 0 ||
        mount.mountpoint().size() > kMaxTopologyMountpointBytes || mount.filesystem_type().size() > kMaxTopologyFilesystemTypeBytes ||
        !IsNormalizedAbsolutePath(mount.mountpoint()) || mounts.find(mount.mount_id()) != mounts.end() ||
        mountpoints.find(mount.mountpoint()) != mountpoints.end()) { *reason = "TOPOLOGY_INVALID"; return false; }
    mounts.emplace(mount.mount_id(), &mount);
    mountpoints.emplace(mount.mountpoint(), mount.mount_id());
    if (mount.mountpoint() == "/") {
      if (root_id != 0 || mount.parent_mount_id() == 0) { *reason = "TOPOLOGY_INVALID"; return false; }
      root_id = mount.mount_id();
    }
  }
  if (root_id == 0) { *reason = "TOPOLOGY_INVALID"; return false; }
  for (const auto& pair : mounts) {
    const auto* mount = pair.second;
    if (mount->mount_id() == root_id) continue;
    if (mounts.find(mount->parent_mount_id()) == mounts.end() || mount->parent_mount_id() == mount->mount_id()) { *reason = "TOPOLOGY_INVALID"; return false; }
    std::set<uint64_t> walked;
    uint64_t current = mount->mount_id();
    while (current != root_id) {
      if (!walked.insert(current).second || mounts.find(current) == mounts.end()) { *reason = "TOPOLOGY_INVALID"; return false; }
      current = mounts.find(current)->second->parent_mount_id();
    }
  }
  std::map<std::string, uint64_t> expected_ids;
  for (const auto& expected : topology->expected) {
    const auto found = mountpoints.find(expected.mountpoint);
    if (found == mountpoints.end() || expected_ids.find(expected.mountpoint) != expected_ids.end()) { *reason = "TOPOLOGY_MISMATCH"; return false; }
    const auto* actual = mounts.find(found->second)->second;
    const auto parent = mountpoints.find(expected.parent);
    if (parent == mountpoints.end() ||
        (expected.mountpoint != "/" && actual->parent_mount_id() != parent->second) ||
        (!expected.filesystem_type.empty() && actual->filesystem_type() != expected.filesystem_type) ||
        actual->read_only() != expected.read_only || actual->noexec() != expected.noexec ||
        actual->nosuid() != expected.nosuid || actual->nodev() != expected.nodev) { *reason = "TOPOLOGY_MISMATCH"; return false; }
    expected_ids.emplace(expected.mountpoint, actual->mount_id());
  }
  // System mounts are pinned-runtime topology, but no unregistered mount may
  // be nested beneath an HAA writable/control anchor after registration.
  for (const auto& pair : mounts) {
    const auto* actual = pair.second;
    bool registered = false;
    for (const auto& expected : topology->expected) {
      if (actual->mountpoint() == expected.mountpoint) { registered = true; break; }
      if (expected.mountpoint != "/" && IsAtOrBelowMountpoint(actual->mountpoint(), expected.mountpoint)) {
        *reason = "TOPOLOGY_MISMATCH"; return false;
      }
    }
    (void)registered;
  }
  topology->anchors.clear();
  for (const auto& pair : mounts) {
    const auto* actual = pair.second;
    std::string mclass = "system";
    for (const auto& exp : topology->expected) {
      if (exp.mountpoint == actual->mountpoint()) {
        mclass = exp.mount_class;
        break;
      }
    }
    topology->anchors.emplace(actual->mount_id(), MountAnchor{actual->mount_id(), actual->mountpoint(), mclass});
  }
  topology->namespace_id = message.mount_namespace_id();
  topology->snapshot_seen = true;
  topology->sealed = true;
  return Send(output, *container_id, "mount-anchors-ready");
}

bool ParseTopologyMutation(const char* payload, size_t payload_size, std::string* container_id,
                           const TopologyState& topology, const char** reason) {
  gvisor::sentry::MountTopologyMutation message;
  if (!message.ParseFromArray(payload, payload_size) || !ValidateContextContainer(message.context_data(), container_id, reason) ||
      !topology.sealed || message.mount_namespace_id() == 0 || message.mount_namespace_id() != topology.namespace_id) {
    *reason = topology.sealed ? "TOPOLOGY_MUTATION" : "TOPOLOGY_INVALID";
    return false;
  }
  *reason = "TOPOLOGY_MUTATION";
  return false;
}

bool Handle(const Header& header, const char* payload, size_t payload_size, int output, std::string* container_id,
            const char* profile, ProcessState* process_state, NormalizedCounts* counts, TopologyState* topology,
            const char** reason) {
  if (header.dropped_count != 0) { *reason = "STREAM_FAULT"; return false; }
  switch (static_cast<gvisor::common::MessageType>(header.message_type)) {
    case gvisor::common::MESSAGE_CONTAINER_START: return ParseContainerStart(payload, payload_size, output, container_id, process_state, reason);
    case gvisor::common::MESSAGE_SENTRY_CLONE: return ParseSentryClone(payload, payload_size, output, container_id, process_state, reason);
    case gvisor::common::MESSAGE_SENTRY_EXEC: return ParseSentryProcessAndClassify(payload, payload_size, output, container_id, profile, process_state, reason);
    // pathname, argv and envv are parsed by protobuf but are deliberately never
    // copied to the HAA envelope. M11-003 supplies the trusted profile required
    // to classify this bounded process fact as expected or unexpected.
    case gvisor::common::MESSAGE_SYSCALL_EXECVE: return ParseExecSyscallTelemetry<gvisor::syscall::Execve>(payload, payload_size, container_id, reason);
    case gvisor::common::MESSAGE_SYSCALL_OPEN: return ParseOpenAndSend(payload, payload_size, output, container_id, process_state, counts, reason);
    case gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT: return ParseOpenResultAndSend(payload, payload_size, output, container_id, profile, process_state, counts, *topology, reason);
    case gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_SNAPSHOT:
      if (!process_state->bootstrap_group_set) { *reason = "TOPOLOGY_INVALID"; return false; }
      return ParseTopologySnapshot(payload, payload_size, output, container_id, topology, reason);
    case gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_MUTATION: return ParseTopologyMutation(payload, payload_size, container_id, *topology, reason);
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
  std::map<std::string, ProfileRegistration> profiles;
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
    NormalizedCounts normalized_counts;
    TopologyState topology_state;
    const char* fault_reason = nullptr;
    const char* profile = nullptr;
    size_t normalized_records = 0;
    while ((size = recv(client, event, sizeof(event), MSG_TRUNC)) > 0) {
      if (static_cast<size_t>(size) > sizeof(event)) { fault = true; fault_reason = "STREAM_FAULT"; break; }
      if (static_cast<size_t>(size) < sizeof(Header)) { fault = true; fault_reason = "STREAM_FAULT"; break; }
      Header header{}; memcpy(&header, event, sizeof(header));
      if (profile == nullptr && !container_id.empty()) {
        const ProfileRegistration* registration = AwaitProfile(control, container_id, &profiles);
        if (registration == nullptr) { fault = true; fault_reason = "PROFILE_LOOKUP_FAILURE"; break; }
        profile = registration->profile.c_str();
        topology_state.expected = registration->expected;
      }
      if (normalized_records + normalized_counts.immediate_records == MaximumRecords(profile)) {
        fault = true; fault_reason = "EVENT_LIMIT"; break;
      }
      if (header.header_size < sizeof(Header) || header.header_size > static_cast<uint16_t>(size)) { fault = true; fault_reason = "STREAM_FAULT"; break; }
      if (!Handle(header, event + header.header_size, size - header.header_size, output, &container_id, profile,
                  &process_state, &normalized_counts, &topology_state, &fault_reason)) {
        fault = true;
        if (fault_reason == nullptr) fault_reason = "STREAM_FAULT";
        break;
      }
      // Filesystem runtime/workspace reads are semantically aggregated. They
      // must not consume the normalized transport-record budget one raw open
      // at a time. Actionable filesystem records are counted by the parser.
      if (header.message_type != gvisor::common::MESSAGE_SYSCALL_OPEN &&
          header.message_type != gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT) ++normalized_records;
    }
    if (size < 0) { fault = true; fault_reason = "STREAM_FAULT"; }
    if (!fault && (!topology_state.sealed || !process_state.pending_sockets.empty() || !process_state.pending_opens.empty())) {
      fault = true;
      // Socket enter events without their required exit leave FD-family state
      // unclassifiable. This is a network-state fault, not obsolete exec
      // correlation state.
      fault_reason = !topology_state.sealed ? "TOPOLOGY_NOT_READY" : (!process_state.pending_sockets.empty() ? "FD_STATE_UNKNOWN" : "STREAM_FAULT");
    }
    if (!fault && !container_id.empty() && normalized_counts.workspace_access != 0) {
      if (normalized_records + normalized_counts.immediate_records == MaximumRecords(profile)) {
        fault = true;
        fault_reason = "EVENT_LIMIT";
      } else if (!Send(output, container_id, "filesystem-workspace-access", nullptr, nullptr,
                       normalized_counts.workspace_access)) {
        fault = true;
        fault_reason = "STREAM_FAULT";
      }
    }
    if (!container_id.empty()) {
      Send(output, container_id, fault ? "stream-fault" : "stream-end", fault_reason);
      profiles.erase(container_id);
    }
    close(client);
  }
}
