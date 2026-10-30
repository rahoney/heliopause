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
#include <initializer_list>
#include <string>
#include <map>
#include <memory>
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
constexpr size_t kMaxGoResolverRecordsPerConnection = 200000;
constexpr size_t kMaxGoBuildRecordsPerConnection = 250000;
constexpr size_t kMaxTrackedProcessGroups = 64;
constexpr size_t kMaxPendingDirectExecAdmissions = 64;
constexpr size_t kDirectExecAdmissionNonceChars = 64;
constexpr size_t kControlSessionGenerationChars = 64;
constexpr size_t kMaxControlRecordBytes = 4096;
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
constexpr char kNodeHandoffMode[] = "--handoff-node";
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
constexpr char kProfilePythonExecution[] = "python-execution";
constexpr char kPythonExecutionPath[] = "/usr/local/bin/python3.14";
constexpr char kPythonExecutionProgram[] = "import importlib\nimport sys\nexpected = ['/usr/local/lib/python314.zip', '/usr/local/lib/python3.14', '/usr/local/lib/python3.14/lib-dynload']\nif not (sys.flags.isolated and sys.flags.no_site and sys.flags.dont_write_bytecode) or 'site' in sys.modules or sys.path != expected:\n    raise RuntimeError('console interpreter startup differs from pinned isolated runtime')\nmodule, attr, command = sys.argv[1:4]\nsys.path.append('/haa-site')\nsys.argv = [command, *sys.argv[4:]]\ntarget = importlib.import_module(module)\nfor part in attr.split('.'):\n    target = getattr(target, part)\nraise SystemExit(target())\n";
constexpr char kProfileNodeExecution[] = "node-execution";
constexpr char kProfilePyPI[] = "pypi-wheel";
constexpr char kProfileUVResolver[] = "uv-pip-resolver";
constexpr char kUVToolPath[] = "/haa-runtime/uv";
constexpr char kProfilePNPMResolver[] = "pnpm-resolver";
constexpr char kPNPMToolPath[] = "/haa-runtime/pnpm";
constexpr char kProfileBunResolver[] = "bun-resolver";
constexpr char kProfileBunProjectResolver[] = "bun-project-resolver";
constexpr char kBunToolPath[] = "/haa-runtime/bun";
constexpr char kProfilePyTorchCPU[] = "pypi-wheel-pytorch-cpu";
constexpr char kProfilePyTorchCU126[] = "pypi-wheel-pytorch-cu126";
constexpr char kProfilePyTorchCU130[] = "pypi-wheel-pytorch-cu130";
constexpr char kProfilePyTorchCU132[] = "pypi-wheel-pytorch-cu132";
constexpr char kProfileGitHub[] = "github-elf";
constexpr char kProfilePipxRuntime[] = "github-pipx-runtime";
constexpr char kProfilePipxNativeResolver[] = "pipx-native-resolver";
constexpr char kPipxNativeProgramPrefix[] = "import importlib.machinery, sys\nversion = '";
constexpr char kPipxNativeProgramSuffix[] = "'\nminor = version.split('.')[:2]\nstdlib = '/usr/local/lib/python' + '.'.join(minor)\nexpected = ['/usr/local/lib/python' + ''.join(minor) + '.zip', stdlib, stdlib + '/lib-dynload']\nif sys.version.split()[0] != version or not (sys.flags.isolated and sys.flags.no_site and sys.flags.dont_write_bytecode) or 'site' in sys.modules or sys.path != expected:\n    raise RuntimeError('pipx native interpreter differs from pinned isolated startup')\nroot = '/haa-pipx/pipx-runtime.zip'\nspec = importlib.machinery.PathFinder.find_spec('pipx', [root])\nif spec is None or spec.origin != root + '/pipx/__init__.py' or list(spec.submodule_search_locations or []) != [root + '/pipx']:\n    raise RuntimeError('pipx native package is outside original ZIP')\nentry = importlib.machinery.PathFinder.find_spec('pipx.__main__', spec.submodule_search_locations)\nif entry is None or entry.origin != root + '/pipx/__main__.py':\n    raise RuntimeError('pipx native entry is outside original ZIP')\nsys.path.append(root)\nsys.argv = ['pipx', *sys.argv[1:]]\nif len(sys.argv) > 2 and sys.argv[1] == 'upgrade':\n    import subprocess\n    subprocess.run(['/haa-runtime/uv', 'venv', '--python', '/usr/local/bin/python', '--allow-existing', '--quiet', '/haa-site/pipx/venvs/' + sys.argv[2]], check=True)\nfrom pipx.__main__ import cli\nraise SystemExit(cli())\n";
constexpr char kPipxNativeMetadataQuery[] = "\nimport json\nimport os\nimport platform\nimport sys\n\nimpl_ver = sys.implementation.version\nimplementation_version = \"{0.major}.{0.minor}.{0.micro}\".format(impl_ver)\nif impl_ver.releaselevel != \"final\":\n    implementation_version = \"{}{}{}\".format(\n        implementation_version,\n        impl_ver.releaselevel[0],\n        impl_ver.serial,\n    )\n\nsys_path = sys.path\ntry:\n    sys_path.remove(\"\")\nexcept ValueError:\n    pass\n\nprint(\n    json.dumps(\n        {\n            \"sys_path\": sys_path,\n            \"python_version\": \"{0.major}.{0.minor}.{0.micro}\".format(sys.version_info),\n            \"environment\": {\n                \"implementation_name\": sys.implementation.name,\n                \"implementation_version\": implementation_version,\n                \"os_name\": os.name,\n                \"platform_machine\": platform.machine(),\n                \"platform_release\": platform.release(),\n                \"platform_system\": platform.system(),\n                \"platform_version\": platform.version(),\n                \"python_full_version\": platform.python_version(),\n                \"platform_python_implementation\": platform.python_implementation(),\n                \"python_version\": \".\".join(platform.python_version_tuple()[:2]),\n                \"sys_platform\": sys.platform,\n            },\n        }\n    )\n)\n";
constexpr char kPipxNativePipArguments[] = "--pip-args=--no-build --no-sources --no-config --no-cache --no-python-downloads --no-managed-python --link-mode=copy --index-url=https://pypi.org/simple/";

constexpr char kProfileTerraformProvider[] = "terraform-provider";
constexpr char kProfileGoResolver[] = "go-module-resolver";
constexpr char kProfileGoBuild[] = "go-module-build";
constexpr char kProfileCargoResolver[] = "cargo-resolver";
constexpr char kProfileCargoBuild[] = "cargo-build";
constexpr char kCargoBinary[] = "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/bin/cargo";
constexpr char kRustcBinary[] = "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/bin/rustc";

bool IsPythonProfile(const char* profile) {
  return profile != nullptr &&
      (strcmp(profile, kProfilePyPI) == 0 ||
       strcmp(profile, kProfilePyTorchCPU) == 0 ||
       strcmp(profile, kProfilePyTorchCU126) == 0 ||
       strcmp(profile, kProfilePyTorchCU130) == 0 ||
       strcmp(profile, kProfilePyTorchCU132) == 0);
}
// Wheel preparation keeps its original profile. The ZIP only shares the
// immutable interpreter ABI and the irreversible Python ARTIFACT handoff.
bool IsPipxRuntimeProfile(const char* profile) {
  return profile != nullptr && strcmp(profile, kProfilePipxRuntime) == 0;
}

bool IsPipxNativeProfile(const char* profile) {
  return profile != nullptr && strcmp(profile, kProfilePipxNativeResolver) == 0;
}

bool IsPythonRuntimeProfile(const char* profile) {
  return IsPythonProfile(profile) || IsPipxRuntimeProfile(profile) || IsPipxNativeProfile(profile) ||
      (profile != nullptr && strcmp(profile,kProfilePythonExecution) == 0);
}
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
          const Attribution* attribution = nullptr, uint64_t count = 0, const char* fault_site = nullptr, uint64_t image_locator = 0, const std::string& fault_open = "", const std::string& fault_budget = "", const std::string& fault_raw = "") {
  if (!ValidContainerID(container_id)) return false;
  std::string message = "{\"container_id\":\"" + container_id + "\",\"kind\":\"" + kind + "\"";
  if (reason != nullptr) message += ",\"reason\":\"" + std::string(reason) + "\"";
  if (fault_site != nullptr) message += ",\"fault_site\":\"" + std::string(fault_site) + "\"";
  if (image_locator != 0) message += ",\"fault_image_locator\":" + std::to_string(image_locator);
  if (!fault_open.empty()) message += ",\"fault_open\":" + fault_open;
  if (!fault_budget.empty()) message += ",\"fault_budget\":" + fault_budget;
  if (!fault_raw.empty()) message += ",\"fault_raw\":" + fault_raw;
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
  kUV,
  kPNPM,
  kBun,
  kNode,
  kNpm,
  kArtifact,
  kSleep,
  kMkdir,
  kCat,
  kChmod,
  kUname,
  kGo,
  kCargo,
  kCargoTar,
  kCargoRustc,
  kProjectBuildBoundary,
  kProjectBuildSetpriv,
  kGoBuildTool,
  kGoBuildCgo,
  kGoBuildLink,
  kGoBuildGcc,
  kGoBuildCc1,
  kGoBuildAssembler,
  kGoBuildCollect2,
  kGoBuildNativeLinker,
  kCargoLldLauncher,
  kCargoRustLld,
  kCargoBuildProgram,
};

enum class SocketClassification {
  kLocal,
  kSpecialKernelLocal,
  kNetwork,
  kUnknown,
};

// Diagnostic identities never participate in classification.
enum class DiagnosticImage { kUnknown, kBoundary, kSetpriv, kShell, kEnv, kNpmCLI, kNode, kCargo, kTar, kRustc, kGcc, kCollect2, kLldLauncher, kRustLld };
DiagnosticImage DiagnosticImageForPath(const std::string& path) {
  if (path == "/usr/bin/x86_64-linux-gnu-gcc-12") return DiagnosticImage::kGcc;
  if (path == "/usr/lib/gcc/x86_64-linux-gnu/12/collect2") return DiagnosticImage::kCollect2;
  if (path == "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/bin/gcc-ld/ld.lld") return DiagnosticImage::kLldLauncher;
  if (path == "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/bin/rust-lld") return DiagnosticImage::kRustLld;
  if (path == kCargoBinary) return DiagnosticImage::kCargo;
  if (path == kRustcBinary) return DiagnosticImage::kRustc;
  if (path == "/usr/bin/tar") return DiagnosticImage::kTar;
  if (path == "/haa-runtime/haa-boundary") return DiagnosticImage::kBoundary;
  if (path == "/usr/bin/setpriv") return DiagnosticImage::kSetpriv;
  if (path == "/bin/sh" || path == "/usr/bin/dash") return DiagnosticImage::kShell;
  if (path == "/usr/bin/env") return DiagnosticImage::kEnv;
  if (path == "/usr/local/lib/node_modules/npm/bin/npm-cli.js" || path == "/usr/local/bin/npm") return DiagnosticImage::kNpmCLI;
  if (path == "/usr/local/bin/node") return DiagnosticImage::kNode;
  return DiagnosticImage::kUnknown;
}
const char* DiagnosticImageName(DiagnosticImage image) {
  switch (image) {
    case DiagnosticImage::kBoundary: return "BOUNDARY";
    case DiagnosticImage::kSetpriv: return "SETPRIV";
    case DiagnosticImage::kShell: return "SHELL";
    case DiagnosticImage::kEnv: return "ENV";
    case DiagnosticImage::kNpmCLI: return "NPM_CLI";
    case DiagnosticImage::kNode: return "NODE";
    case DiagnosticImage::kCargo: return "CARGO";
    case DiagnosticImage::kTar: return "TAR";
    case DiagnosticImage::kRustc: return "RUSTC";
    case DiagnosticImage::kGcc: return "GCC";
    case DiagnosticImage::kCollect2: return "COLLECT2";
    case DiagnosticImage::kLldLauncher: return "LLD_LAUNCHER";
    case DiagnosticImage::kRustLld: return "RUST_LLD";
    default: return "UNKNOWN";
  }
}

enum class CommandPhase {
  kUnknown,
  kNpmVersion,
  kManifestWrite,
  kLockGeneration,
  kLockRead,
  kReadinessCheck,
  kOther,
};

const char* CommandPhaseName(CommandPhase phase) {
  switch (phase) {
    case CommandPhase::kNpmVersion: return "NPM_VERSION";
    case CommandPhase::kManifestWrite: return "MANIFEST_WRITE";
    case CommandPhase::kLockGeneration: return "LOCK_GENERATION";
    case CommandPhase::kLockRead: return "LOCK_READ";
    case CommandPhase::kReadinessCheck: return "READINESS_CHECK";
    case CommandPhase::kOther: return "OTHER";
    default: return "UNKNOWN";
  }
}

enum class FaultSite {
  kNone,
  kRecvTrunc,
  kRecvShort,
  kProfileLookup,
  kEventLimit,
  kHeaderSize,
  kDroppedCount,
  kContainerStart,
  kSentryClone,
  kSentryExitNotifyParent,
  kSentryExec,
  kExecSyscall,
  kOpen,
  kOpenResult,
  kOpenResultEnvelope,
  kOpenResultCorrelation,
  kOpenResultFailureFormat,
  kOpenResultSuccessFormat,
  kOpenResultAnchor,
  kOpenResultShadow,
  kOpenResultClassification,
  kOpenResultClassificationProcessName,
  kOpenResultClassificationProc,
  kOpenResultClassificationSys,
  kOpenResultClassificationImage,
  kOpenResultClassificationOther,
  kTopologySnapshot,
  kTopologyMutation,
  kConnect,
  kSocket,
  kRaw,
  kFdTrack,
  kUnknownMessage,
  kRecvError,
  kUnsealedTopology,
  kPendingSockets,
  kPendingOpens,
  kWorkspaceSend,
};

const char* FaultSiteName(FaultSite site) {
  switch (site) {
    case FaultSite::kRecvTrunc: return "RECV_TRUNC";
    case FaultSite::kRecvShort: return "RECV_SHORT";
    case FaultSite::kProfileLookup: return "PROFILE_LOOKUP";
    case FaultSite::kEventLimit: return "EVENT_LIMIT";
    case FaultSite::kHeaderSize: return "HEADER_SIZE";
    case FaultSite::kDroppedCount: return "DROPPED_COUNT";
    case FaultSite::kContainerStart: return "CONTAINER_START";
    case FaultSite::kSentryClone: return "SENTRY_CLONE";
    case FaultSite::kSentryExitNotifyParent: return "SENTRY_EXIT_NOTIFY_PARENT";
    case FaultSite::kSentryExec: return "SENTRY_EXEC";
    case FaultSite::kExecSyscall: return "EXEC_SYSCALL";
    case FaultSite::kOpen: return "OPEN";
    case FaultSite::kOpenResult: return "OPEN_RESULT";
    case FaultSite::kOpenResultEnvelope: return "OPEN_RESULT_ENVELOPE";
    case FaultSite::kOpenResultCorrelation: return "OPEN_RESULT_CORRELATION";
    case FaultSite::kOpenResultFailureFormat: return "OPEN_RESULT_FAILURE_FORMAT";
    case FaultSite::kOpenResultSuccessFormat: return "OPEN_RESULT_SUCCESS_FORMAT";
    case FaultSite::kOpenResultAnchor: return "OPEN_RESULT_ANCHOR";
    case FaultSite::kOpenResultShadow: return "OPEN_RESULT_SHADOW";
    case FaultSite::kOpenResultClassification: return "OPEN_RESULT_CLASSIFICATION";
    case FaultSite::kOpenResultClassificationProcessName: return "OPEN_RESULT_CLASSIFICATION_PROCESS_NAME";
    case FaultSite::kOpenResultClassificationProc: return "OPEN_RESULT_CLASSIFICATION_PROC";
    case FaultSite::kOpenResultClassificationSys: return "OPEN_RESULT_CLASSIFICATION_SYS";
    case FaultSite::kOpenResultClassificationImage: return "OPEN_RESULT_CLASSIFICATION_IMAGE";
    case FaultSite::kOpenResultClassificationOther: return "OPEN_RESULT_CLASSIFICATION_OTHER";
    case FaultSite::kTopologySnapshot: return "TOPOLOGY_SNAPSHOT";
    case FaultSite::kTopologyMutation: return "TOPOLOGY_MUTATION";
    case FaultSite::kConnect: return "CONNECT";
    case FaultSite::kSocket: return "SOCKET";
    case FaultSite::kRaw: return "RAW";
    case FaultSite::kFdTrack: return "FD_TRACK";
    case FaultSite::kUnknownMessage: return "UNKNOWN_MESSAGE";
    case FaultSite::kRecvError: return "RECV_ERROR";
    case FaultSite::kUnsealedTopology: return "UNSEALED_TOPOLOGY";
    case FaultSite::kPendingSockets: return "PENDING_SOCKETS";
    case FaultSite::kPendingOpens: return "PENDING_OPENS";
    case FaultSite::kWorkspaceSend: return "WORKSPACE_SEND";
    case FaultSite::kNone:
    default: return "NONE";
  }
}

struct ProcessState {
  enum class Role { kUnknown, kControl, kArtifact };
  enum class Provenance { kUnknown, kOCIRoot, kDirectExecRoot, kCloneChild };
  enum class PipxNativeImage { kNone, kRoot, kUV, kUVQuery, kMetadataQuery };
  enum class OCIBootstrapStage { kNotOCI, kAwaitingBootstrapShell, kAwaitingDemotion, kAwaitingSleep, kComplete };
  struct UnexpectedExecDiagnostic {
    bool present = false;
    DiagnosticImage previous_image = DiagnosticImage::kUnknown;
    DiagnosticImage current_image = DiagnosticImage::kUnknown;
    CommandPhase phase = CommandPhase::kUnknown;
    int32_t process_id = 0;
    int32_t parent_id = 0;
    ProcessClass previous_class = ProcessClass::kUnknown;
    ProcessClass current_class = ProcessClass::kUnknown;
    const char* classification_reason = "NONE";
    Role role = Role::kUnknown;
    Provenance provenance = Provenance::kUnknown;
    const char* parent_relation = "NONE";
    bool root_eligible = false;
    bool root_consumed = false;
    bool trusted_control_network_active = false;
    bool demotion_pending = false;
    bool launch_target_pending = false;
    bool handoff_target_pending = false;
    bool npm_node_transition_pending = false;
    bool lock_generation_npm_node_transition_pending = false;
    bool npm_version_node_transition_pending = false;
    bool npm_version_node_transition_consumed = false;
  };
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
    // This is a distinct one-shot authority for the fixed GenerateLockfile
    // boundary command. It is armed only by that command's exact clone-child
    // npm launcher and consumed by its exact Node interpreter transition.
    bool lock_generation_npm_node_transition_pending;
    // This is set only after the exact direct-exec npm --version launcher is
    // accepted for the exact boundary command. It authorizes one exact Node
    // interpreter image transition and is cleared as it is consumed.
    bool npm_version_node_transition_pending;
    // This is set only while that exact interpreter image remains current.
    // It records the completed transition for the one NPM_VERSION runtime-read
    // exception below; a later image transition clears it and never regains it.
    bool npm_version_node_transition_consumed;
    ProcessClass handoff_target_class;
    OCIBootstrapStage oci_bootstrap_stage;
    CommandPhase command_phase = CommandPhase::kUnknown;
    DiagnosticImage diagnostic_image = DiagnosticImage::kUnknown;
    // Sentry clone provenance establishes this bounded parent relation. It is
    // used by exact kernel-owned runtime transitions.
    int32_t clone_creator_group_id = 0;
    int64_t clone_creator_group_start_time_ns = 0;
    // Kernel image-load evidence, retained across thread-name changes and
    // fork, replaced on every exec. This grants no CONTROL or network trust.
    ProcessClass current_image_class = ProcessClass::kUnknown;
    // FNV of the last kernel-resolved exec image is diagnostic only.
    uint64_t executable_locator = 0;
    // Current exact runtime query only; neither role nor inheritable authority.
    bool runtime_cache_query = false;
    std::string bun_run_package;
    bool bun_run_install_candidate = false;
    bool bun_run_install_active = false;
    bool bun_offline_install_active = false;
    bool diagnostic_image_pinned = false;
    bool diagnostic_rustc_version = false;
    bool diagnostic_rustc_metadata = false;
    bool diagnostic_native_cc = false;
    bool diagnostic_lld_same_group = false;
    bool diagnostic_build_target_image = false;
    uint32_t diagnostic_rustc_argc = 0;
    uint64_t diagnostic_rustc_argv_locator = 0;
    // Existing SocketPair telemetry reads guest return-buffer bytes. These
    // fields are diagnostics only and must never populate the FD authority.
    bool diagnostic_socketpair_seen = false;
    int32_t diagnostic_socketpair_first = -1;
    int32_t diagnostic_socketpair_second = -1;
    int32_t diagnostic_socketpair_domain = 0;
    uint32_t diagnostic_socketpair_type = 0;
    bool cargo_rustc_query_candidate = false;
    bool cargo_rustc_query_active = false;
    // One first exec of an admitted UV driver's kernel clone. The query is
    // demoted to ARTIFACT and never joins expected/control network groups.
    bool uv_python_query_candidate = false;
    bool uv_python_query_active = false;
    // Query descendants retain this restriction, never the active admission.
    bool uv_python_query_restricted = false;
    std::string uv_python_query_environment;
    std::string uv_python_query_helper_root;
    bool cargo_build_rustc_active = false;
    bool cargo_lld_launcher_candidate = false;
    bool cargo_lld_launcher_active = false;
    bool cargo_rust_lld_active = false;
    bool cargo_build_program_candidate = false;
    bool cargo_build_program_active = false;
    bool cargo_build_program_query_candidate = false;
    bool cargo_build_program_query_active = false;
    bool go_build_tool_candidate = false;
    bool go_build_tool_active = false;
    bool go_build_gcc_candidate = false;
    bool go_build_gcc_child_candidate = false;
    bool go_build_native_linker_candidate = false;
    // One first exec of a kernel clone of the admitted PNPM driver. Its
    // version-only Node query is ARTIFACT, never an expected CONTROL group.
    bool pnpm_node_query_candidate = false;
    bool pnpm_node_query_active = false;
    // Exact target from one admitted control boundary, inherited only through
    // kernel-owned CONTROL clone lineage. It is not artifact/source authority.
    std::string pnpm_target_lifecycle{};
    bool pnpm_target_npm_pending = false;
    bool pnpm_target_npm_consumed = false;
    // Native pipx remains ARTIFACT. These facts authorize neither CONTROL nor
    // networking and are never inherited as active admission by a clone.
    PipxNativeImage pipx_native_image = PipxNativeImage::kNone;
    bool pipx_native_candidate = false;
    bool pipx_native_query_restricted = false;
    std::string pipx_native_operation, pipx_native_project, pipx_native_requirement;
    std::string pipx_native_environment;
    // Exact current Node handoff; private argv never enter diagnostics.
    std::vector<std::string> node_execution_target;
    bool node_execution_active = false;
    bool node_execution_demoting = false;
    std::vector<std::string> python_execution_target;
    bool python_execution_active = false;
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
  struct PendingSocketPair {
    int32_t thread_group_id;
    int64_t thread_group_start_time_ns;
    uint64_t sysno;
    int32_t domain;
    int32_t type;
    int32_t protocol;
  };
  std::map<std::pair<int32_t, int64_t>, PendingSocketPair> pending_socketpairs;
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
  UnexpectedExecDiagnostic first_unexpected_exec;
  FaultSite terminal_fault_site = FaultSite::kNone;
  uint64_t fault_image_locator = 0;
  std::string fault_open_diagnostic;
  std::string first_outside_open_diagnostic;
  std::string fault_raw_diagnostic;
  // Diagnostic only: these bounded scalars are never descriptor authority.
  struct FDMutationDiagnostic {
    int32_t group_id;
    int64_t group_start;
    int32_t fd;
    const char* operation;
    int32_t family;
  };
  std::vector<FDMutationDiagnostic> fd_mutation_diagnostics;
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

bool IsAtOrBelowMountpoint(const std::string& path, const std::string& mountpoint);

// Kernel-resolved path + sealed, host-attested read-only OCI namespace, not a
// pathname claim. A nested mount (even read-only) cannot substitute this file.
bool IsPinnedReadOnlyRootPath(const TopologyState* topology, const std::string& path) {
  if (topology == nullptr || !topology->sealed || !topology->snapshot_seen ||
      topology->namespace_id == 0) return false;
  bool readonly_root = false;
  for (const auto& mount : topology->expected) {
    if (mount.mountpoint == "/" && mount.mount_class == "oci-root" && mount.read_only)
      readonly_root = true;
  }
  bool actual_root = false;
  for (const auto& entry : topology->anchors) {
    const auto& anchor = entry.second;
    if (anchor.mountpoint == "/" && anchor.mount_class == "oci-root") actual_root = true;
    else if (IsAtOrBelowMountpoint(path, anchor.mountpoint)) return false;
  }
  return readonly_root && actual_root;
}

// The controller supplies and rehashes the exact tool on a root-owned private tmpfs.
// A path on an OCI layer or a nested/shadow mount is never this tool.
bool IsPrivateNativeToolImage(const TopologyState* topology, const std::string& path,
                              const char* expected) {
  if (path != expected || topology == nullptr || !topology->sealed ||
      !topology->snapshot_seen || topology->namespace_id == 0) return false;
  size_t expected_helpers = 0;
  for (const auto& mount : topology->expected) {
    if (mount.mountpoint != "/haa-runtime") continue;
    if (mount.mount_class != "helper" || mount.parent != "/" ||
        mount.filesystem_type != "9p" || mount.read_only || mount.noexec || mount.nosuid || mount.nodev)
      return false;
    expected_helpers++;
  }
  size_t actual_helpers = 0;
  for (const auto& entry : topology->anchors) {
    const auto& mount = entry.second;
    if (mount.mountpoint == "/haa-runtime" && mount.mount_class == "helper") actual_helpers++;
    else if (mount.mountpoint != "/" && IsAtOrBelowMountpoint(path, mount.mountpoint)) return false;
  }
  return expected_helpers == 1 && actual_helpers == 1;
}

bool IsUVToolImage(const TopologyState* topology, const std::string& path) {
  return IsPrivateNativeToolImage(topology, path, kUVToolPath);
}

bool IsPNPMToolImage(const TopologyState* topology, const std::string& path) {
  return IsPrivateNativeToolImage(topology, path, kPNPMToolPath);
}

bool IsBunToolImage(const TopologyState* topology, const std::string& path) {
  return IsPrivateNativeToolImage(topology, path, kBunToolPath);
}

// These two native diagnostics are the only Bun commands admitted at this
// stage. Host control chooses the arguments; guest names confer no authority.
bool IsExactBunQuery(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kBunToolPath && message.execfn() == kBunToolPath &&
      message.argv_size() == 2 && message.argv(0) == kBunToolPath &&
      (message.argv(1) == "--version" || message.argv(1) == "--revision");
}

bool IsBunPublicSelector(const std::string& value, bool allow_constraint) {
  if (value.empty() || value.size() > 344) return false;
  std::string name = value;
  const auto at = value.rfind('@');
  if (at != std::string::npos && at != 0) {
    if (!allow_constraint || at + 1 == value.size() || value.size() - at - 1 > 128) return false;
    name = value.substr(0, at);
    for (size_t i = at + 1; i < value.size(); ++i) {
      const char c = value[i];
      if (!((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') ||
            (c >= 'a' && c <= 'z') || std::string(".*~^<>=|+_ -").find(c) != std::string::npos)) return false;
    }
  }
  if (name.empty() || name.size() > 214) return false;
  auto segment = [](const std::string& part) {
    if (part.empty() || !((part[0] >= 'a' && part[0] <= 'z') || (part[0] >= '0' && part[0] <= '9'))) return false;
    for (const char c : part) if (!((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-')) return false;
    return true;
  };
  if (name[0] != '@') return segment(name);
  const auto slash = name.find('/');
  return slash != std::string::npos && segment(name.substr(1, slash-1)) && segment(name.substr(slash+1));
}

// Fixed selection and script-free offline preparation shapes. The host typed
// input owns selector/data semantics; neither preparation shape accepts argv.
// Guest arguments cannot change registry/cache/linker, execute scripts or load env.
bool IsExactBunProjectLockInstall(const gvisor::sentry::ExecveInfo& message) {
  static constexpr const char* expected[] = {kBunToolPath,
      "--cwd=/tmp/haa-bun-input/project", "install", "--lockfile-only", "--ignore-scripts",
      "--no-env-file", "--registry=https://registry.npmjs.org/",
      "--cache-dir=/tmp/haa-bun-input/cache", "--backend=copyfile", "--linker=hoisted",
      "--save-text-lockfile", "--no-progress", "--no-summary"};
  constexpr int fixed_count = sizeof(expected)/sizeof(expected[0]);
  if (message.binary_path() != kBunToolPath || message.execfn() != kBunToolPath ||
      message.argv_size() < fixed_count || message.argv_size() > fixed_count + 2) return false;
  const bool cache_seed = message.argv(1) == "--cwd=/tmp/haa-bun-input/seed";
  const bool offline_install = !cache_seed && message.argv(3) == "--offline";
  for (int i=0; i<fixed_count; ++i) {
    if (i == 2 || (i == 1 && cache_seed) || (i == 3 && offline_install)) continue;
    if (message.argv(i) != expected[i]) return false;
  }
  const auto& intent = message.argv(2);
  if (cache_seed || offline_install) return intent == "install" && message.argv_size() == fixed_count+1 &&
      message.argv(fixed_count) == (cache_seed ? "--offline" : "--frozen-lockfile");
  if (intent == "install") return message.argv_size() == fixed_count ||
      (message.argv_size() == fixed_count+1 && message.argv(fixed_count) == "--frozen-lockfile");
  if (intent != "add" && intent != "update") return false;
  if (message.argv_size() == fixed_count) return intent == "update";
  return message.argv_size() == fixed_count+2 && message.argv(fixed_count) == "--" &&
      IsBunPublicSelector(message.argv(fixed_count+1), intent == "add");
}

// Script-free native selection requests only a missing command in the sealed
// helper mount. The typed caller and independent source binding own the package
// semantics; this closed shape cannot pass a program or caller argv.
bool IsExactBunRunSelection(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kBunToolPath && message.execfn() == kBunToolPath &&
      message.argv_size() == 7 && message.argv(0) == kBunToolPath &&
      message.argv(1) == "--cwd=/tmp/haa-bun-input" && message.argv(2) == "x" &&
      message.argv(3) == "--ignore-scripts" && message.argv(4) == "--registry=https://registry.npmjs.org/" &&
      message.argv(5).rfind("--package=", 0) == 0 && IsBunPublicSelector(message.argv(5).substr(10), true) &&
      message.argv(6) == "/haa-runtime/helox-bunx-inert-query";
}

struct ProfileRegistration {
  std::string profile;
  std::vector<ExpectedMount> expected;
  std::string session_generation;
	int control_fd = -1;
  enum class AdmissionState { kPending, kConsumed };
  struct PendingAdmission {
		std::string session_generation;
    std::string mode;
    std::string nonce;
    AdmissionState state = AdmissionState::kPending;
    int32_t root_group_id = 0;
    int64_t root_start_time_ns = 0;
    bool target_loaded = false;
    bool terminal_seen = false;
    uint32_t terminal_wait_status = 0;
  };
  std::vector<PendingAdmission> pending_admissions;
};

struct ControlPeer {
  int fd = -1;
  bool request_seen = false;
  bool registered = false;
  bool terminal = false;
  std::string container_id;
  std::string session_generation;
};

enum class BoundaryMode { kNone, kLaunch, kHandoff, kPythonHandoff, kELFHandoff, kNodeHandoff };

bool SameGroup(const ProcessState::GroupState& group, const gvisor::common::ContextData& context);
bool ValidProcessIdentity(const gvisor::common::ContextData& context);
bool ValidateContextContainer(const gvisor::common::ContextData& context,
                              std::string* container_id, const char** reason);
bool IsNormalizedAbsolutePath(const std::string& path);
bool IsAtOrBelowMountpoint(const std::string& path, const std::string& mountpoint);

bool SameGroup(const ProcessState::GroupState& group, const gvisor::common::ContextData& context) {
  return group.start_time_ns == context.thread_group_start_time_ns();
}

bool HasExactCloneCreator(const ProcessState::GroupState& group, const ProcessState& state) {
  if (group.clone_creator_group_id <= 0 || group.clone_creator_group_start_time_ns <= 0) return false;
  const auto creator = state.groups.find(group.clone_creator_group_id);
  return creator != state.groups.end() &&
      creator->second.start_time_ns == group.clone_creator_group_start_time_ns;
}

// Only the kernel clone of a currently admitted bun x selector can run its
// fixed script-free cold installer. Neither marker is copied to descendants.
const ProcessState::GroupState* BunRunInstallCreator(const gvisor::common::ContextData& context,
    const ProcessState::GroupState& group, const ProcessState& state) {
  if (!context.is_exec_session() || group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild || group.root_eligible || !group.root_consumed ||
      group.trusted_control_network_active || group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      context.parent_thread_group_id() != group.clone_creator_group_id || !HasExactCloneCreator(group,state)) return nullptr;
  const auto& parent = state.groups.find(group.clone_creator_group_id)->second;
  const auto expected = state.expected_groups.find(group.clone_creator_group_id);
  return parent.role == ProcessState::Role::kControl && parent.provenance == ProcessState::Provenance::kDirectExecRoot &&
      !parent.root_eligible && parent.root_consumed && parent.trusted_control_network_active &&
      !parent.demotion_pending && !parent.launch_target_pending && !parent.handoff_target_pending &&
      parent.current_image_class == ProcessClass::kBun && !parent.bun_run_package.empty() &&
      expected != state.expected_groups.end() && expected->second.start_time_ns == parent.start_time_ns &&
      expected->second.process_class == ProcessClass::kBun ? &parent : nullptr;
}

bool IsBunRunInstallGroup(const gvisor::common::ContextData& context, const ProcessState& state) {
  const auto group = state.groups.find(context.thread_group_id());
  const auto expected = state.expected_groups.find(context.thread_group_id());
  return group != state.groups.end() && SameGroup(group->second,context) && group->second.bun_run_install_active &&
      group->second.current_image_class == ProcessClass::kBun && expected != state.expected_groups.end() &&
      expected->second.start_time_ns == group->second.start_time_ns && expected->second.process_class == ProcessClass::kBun &&
      BunRunInstallCreator(context,group->second,state) != nullptr;
}

bool IsExactBunRunInstall(const gvisor::sentry::ExecveInfo& message, const ProcessState::GroupState& parent) {
  const auto version = parent.bun_run_package.rfind('@');
  const bool bare = version == std::string::npos || version == 0;
  const int force = bare ? 5 : 4;
  return message.binary_path() == kBunToolPath && message.execfn() == kBunToolPath &&
      message.argv_size() == force + 3 && message.argv(0) == kBunToolPath && message.argv(1) == "add" &&
      message.argv(2) == parent.bun_run_package && message.argv(3) == "--no-summary" &&
      (!bare || message.argv(4) == "--no-cache") && message.argv(force) == "--force" &&
      message.argv(force+1) == "--ignore-scripts" && message.argv(force+2) == "--registry=https://registry.npmjs.org/";
}

bool HasExactNativeQueryCreator(const gvisor::common::ContextData& context,
                            const ProcessState::GroupState& group,
                            const ProcessState& state, ProcessClass source_class) {
  if (source_class != ProcessClass::kUV && source_class != ProcessClass::kPNPM) return false;
  if (group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      !context.is_exec_session() || group.root_eligible || !group.root_consumed ||
      group.trusted_control_network_active || group.demotion_pending ||
      group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state) ||
      context.parent_thread_group_id() != group.clone_creator_group_id) return false;
  const auto& parent = state.groups.find(group.clone_creator_group_id)->second;
  const auto expected = state.expected_groups.find(group.clone_creator_group_id);
  return parent.role == ProcessState::Role::kControl &&
      parent.provenance == ProcessState::Provenance::kDirectExecRoot &&
      parent.current_image_class == source_class && !parent.root_eligible &&
      parent.root_consumed && parent.trusted_control_network_active &&
      !parent.demotion_pending && !parent.launch_target_pending && !parent.handoff_target_pending &&
      expected != state.expected_groups.end() && expected->second.start_time_ns == parent.start_time_ns &&
      expected->second.process_class == source_class;
}

bool HasExactUVQueryCreator(const gvisor::common::ContextData& context,
                            const ProcessState::GroupState& group,
                            const ProcessState& state) {
  return HasExactNativeQueryCreator(context, group, state, ProcessClass::kUV);
}

bool HasExactPNPMQueryCreator(const gvisor::common::ContextData& context,
                              const ProcessState::GroupState& group,
                              const ProcessState& state) {
  return HasExactNativeQueryCreator(context, group, state, ProcessClass::kPNPM);
}

// This grammar is the pinned UV's launcher, not arbitrary Python code. The
// module producer is the authenticated UV in a private, no-build namespace;
// neither these random names nor a diagnostic hash authenticate code on their own.
constexpr char kUVExportedCacheRoot[] = "/haa-site/uv-run-cache";

bool HasUVExportedCacheTopology(const TopologyState* topology, const std::string& path) {
  if (topology == nullptr || !topology->sealed || !topology->snapshot_seen || topology->namespace_id == 0 ||
      !IsAtOrBelowMountpoint(path, kUVExportedCacheRoot)) return false;
  size_t site_expected = 0, site_actual = 0, tmp_expected = 0, tmp_actual = 0;
  for (const auto& mount : topology->expected) {
    if (mount.mountpoint == "/haa-site") {
      if (mount.mount_class != "workspace" || mount.parent != "/" || mount.filesystem_type != "9p" || mount.read_only) return false;
      site_expected++;
    }
    if (mount.mountpoint == "/tmp") {
      if (mount.mount_class != "workspace" || mount.parent != "/" || mount.filesystem_type != "tmpfs" ||
          mount.read_only || !mount.noexec || !mount.nosuid) return false;
      tmp_expected++;
    }
  }
  for (const auto& entry : topology->anchors) {
    const auto& mount = entry.second;
    if (mount.mountpoint == "/haa-site") {
      if (mount.mount_class != "workspace") return false;
      site_actual++;
    } else if (mount.mountpoint == "/tmp") {
      if (mount.mount_class != "workspace") return false;
      tmp_actual++;
    } else if (mount.mountpoint != "/" &&
        (IsAtOrBelowMountpoint(path, mount.mountpoint) || IsAtOrBelowMountpoint(mount.mountpoint, kUVExportedCacheRoot))) return false;
  }
  return site_expected == 1 && site_actual == 1 && tmp_expected == 1 && tmp_actual == 1;
}

std::string UVToolQueryEnvironment(const std::string& execfn, const TopologyState* topology) {
  if (topology == nullptr) return "";
  const std::string exported_prefix = std::string(kUVExportedCacheRoot) + "/archive-v0/";
  if (execfn.compare(0, exported_prefix.size(), exported_prefix) == 0) {
    constexpr size_t kIDBytes = 16;
    if (execfn.size() != exported_prefix.size() + kIDBytes + 11 ||
        execfn.compare(exported_prefix.size() + kIDBytes, 11, "/bin/python") != 0) return "";
    for (size_t i = exported_prefix.size(); i < exported_prefix.size() + kIDBytes; ++i) {
      const char c = execfn[i];
      if (!((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-')) return "";
    }
    const std::string environment = execfn.substr(0, exported_prefix.size() + kIDBytes);
    return HasUVExportedCacheTopology(topology, environment) ? environment : "";
  }
  if (execfn.compare(0, 5, "/tmp/") == 0) {
    // Fresh UV0.13.4 Cache::persist uses a tempfile root and secure ID16.
    // These names confer no identity without the pinned image/creator below.
    if (execfn.size() != 54 || execfn.compare(0, 9, "/tmp/.tmp") != 0 ||
        execfn.compare(15, 12, "/archive-v0/") != 0 ||
        execfn.compare(43, 11, "/bin/python") != 0) return "";
    auto alnum = [](char c) { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'); };
    for (size_t i = 9; i < 15; ++i) if (!alnum(execfn[i])) return "";
    for (size_t i = 27; i < 43; ++i) if (!alnum(execfn[i]) && execfn[i] != '_' && execfn[i] != '-') return "";
    const std::string environment = execfn.substr(0, 43);
    size_t expected = 0, actual = 0;
    for (const auto& mount : topology->expected) {
      if (mount.mountpoint != "/tmp") continue;
      if (mount.mount_class != "workspace" || mount.parent != "/" ||
          mount.filesystem_type != "tmpfs" || mount.read_only || !mount.noexec || !mount.nosuid) return "";
      expected++;
    }
    for (const auto& entry : topology->anchors) {
      const auto& mount = entry.second;
      if (mount.mountpoint == "/tmp" && mount.mount_class == "workspace") actual++;
      else if (mount.mountpoint != "/" &&
          (IsAtOrBelowMountpoint(execfn, mount.mountpoint) ||
           IsAtOrBelowMountpoint(mount.mountpoint, environment))) return "";
    }
    return expected == 1 && actual == 1 ? environment : "";
  }
  const std::string prefix = "/haa-site/tools/", suffix = "/bin/python";
  if (topology == nullptr || execfn.size() <= prefix.size() + suffix.size() ||
      execfn.compare(0, prefix.size(), prefix) != 0 ||
      execfn.compare(execfn.size() - suffix.size(), suffix.size(), suffix) != 0) return "";
  const std::string project = execfn.substr(prefix.size(), execfn.size() - prefix.size() - suffix.size());
  auto alnum = [](char c) { return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'); };
  if (project.size() > 214 || !alnum(project.front()) || !alnum(project.back())) return "";
  for (char c : project) if (!alnum(c) && c != '-') return "";
  size_t expected = 0, actual = 0;
  for (const auto& mount : topology->expected) {
    if (mount.mountpoint != "/haa-site") continue;
    if (mount.mount_class != "workspace" || mount.parent != "/" ||
        mount.filesystem_type != "9p" || mount.read_only) return "";
    expected++;
  }
  for (const auto& entry : topology->anchors) {
    const auto& mount = entry.second;
    if (mount.mountpoint == "/haa-site" && mount.mount_class == "workspace") actual++;
    else if (mount.mountpoint != "/" && IsAtOrBelowMountpoint(execfn, mount.mountpoint)) return "";
  }
  return expected == 1 && actual == 1 ? prefix + project : "";
}

bool IsExactUVPythonQueryProgram(const gvisor::sentry::ExecveInfo& message,
                                   const TopologyState* topology, std::string* helper_root = nullptr) {
  if (topology == nullptr || message.argv_size() != 5) return false;
  const std::string prefix = "import sys; sys.path = [__import__(\"os\").fsdecode(b\"";
  const std::string suffix = "\")] + sys.path; from python.get_interpreter_info import main; main()";
  const std::string& code = message.argv(4);
  const std::string exported_prefix = std::string(kUVExportedCacheRoot) + "/.tmp";
  const size_t exported_bytes = exported_prefix.size() + 6;
  if (code.size() == prefix.size() + exported_bytes + suffix.size() &&
      code.compare(0, prefix.size(), prefix) == 0 &&
      code.compare(prefix.size() + exported_bytes, suffix.size(), suffix) == 0) {
    const std::string root = code.substr(prefix.size(), exported_bytes);
    if (root.compare(0, exported_prefix.size(), exported_prefix) != 0) return false;
    for (size_t i = exported_prefix.size(); i < root.size(); ++i) {
      const char c = root[i];
      if (!((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'))) return false;
    }
    if (!HasUVExportedCacheTopology(topology, root)) return false;
    if (helper_root != nullptr) *helper_root = root;
    return true;
  }
  constexpr size_t kRootBytes = 26;  // /tmp/.tmpXXXXXX/.tmpXXXXXX
  if (code.size() != prefix.size() + kRootBytes + suffix.size() ||
      code.compare(0, prefix.size(), prefix) != 0 ||
      code.compare(prefix.size() + kRootBytes, suffix.size(), suffix) != 0) return false;
  const std::string root = code.substr(prefix.size(), kRootBytes);
  if (root.compare(0, 9, "/tmp/.tmp") != 0 || root.compare(15, 5, "/.tmp") != 0) return false;
  for (size_t i : {size_t{9}, size_t{20}}) {
    for (size_t j = i; j < i + 6; ++j) {
      const char c = root[j];
      if (!((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'))) return false;
    }
  }
  size_t expected_tmp = 0, actual_tmp = 0;
  for (const auto& mount : topology->expected) {
    if (mount.mountpoint != "/tmp") continue;
    if (mount.mount_class != "workspace" || mount.parent != "/" || mount.filesystem_type != "tmpfs" ||
        mount.read_only || !mount.noexec || !mount.nosuid) return false;
    expected_tmp++;
  }
  for (const auto& entry : topology->anchors) {
    const auto& mount = entry.second;
    if (mount.mountpoint == "/tmp" && mount.mount_class == "workspace") actual_tmp++;
    else if (mount.mountpoint != "/" && IsAtOrBelowMountpoint(root, mount.mountpoint)) return false;
  }
  const bool valid = expected_tmp == 1 && actual_tmp == 1;
  if (valid && helper_root != nullptr) *helper_root = root;
  return valid;
}

bool IsExactUVPythonQuery(const gvisor::sentry::ExecveInfo& message,
                          const TopologyState* topology, std::string* helper_root = nullptr) {
  const std::string environment = UVToolQueryEnvironment(message.execfn(), topology);
  const bool tool_alias = !environment.empty();
  if (message.binary_path() != "/usr/local/bin/python3.14" ||
      (!tool_alias && message.execfn() != "/usr/local/bin/python" && message.execfn() != message.binary_path()) ||
      message.argv_size() != 5 || message.argv(0) != message.execfn() ||
      message.argv(1) != "-I" || message.argv(2) != "-B" || message.argv(3) != "-c" ||
      !IsPinnedReadOnlyRootPath(topology, message.binary_path()) ||
      (!tool_alias && !IsPinnedReadOnlyRootPath(topology, message.execfn())) || !IsUVToolImage(topology, kUVToolPath)) return false;
  std::string root;
  if (!IsExactUVPythonQueryProgram(message, topology, &root)) return false;
  const bool exported_helper = IsAtOrBelowMountpoint(root, kUVExportedCacheRoot);
  const bool exported_environment = IsAtOrBelowMountpoint(environment, kUVExportedCacheRoot);
  if ((exported_environment && !exported_helper) ||
      (exported_helper && !environment.empty() && !exported_environment)) return false;
  if (environment.compare(0, 5, "/tmp/") == 0) {
    for (const auto& entry : topology->anchors) {
      const auto& mount = entry.second;
      if (mount.mountpoint != "/" && mount.mountpoint != "/tmp" &&
          (IsAtOrBelowMountpoint(root, mount.mountpoint) ||
           IsAtOrBelowMountpoint(mount.mountpoint, root))) return false;
    }
  }
  if (helper_root != nullptr) *helper_root = root;
  return true;
}

bool IsPipxNativeTopology(const TopologyState* topology) {
  if (topology == nullptr || !topology->sealed || !topology->snapshot_seen || topology->namespace_id == 0 || topology->expected.size() != 6) return false;
  const std::vector<ExpectedMount> required{
      {"/", "oci-root", "/", "", true, false, false, false},
      {"/tmp", "workspace", "/", "tmpfs", false, true, true, false},
      {"/haa-runtime", "helper", "/", "9p", false, false, false, false},
      {"/haa-pipx", "workspace", "/", "9p", true, false, false, false},
      {"/haa-wheels", "workspace", "/", "9p", true, false, false, false},
      {"/haa-site", "workspace", "/", "9p", false, false, false, false}};
  for (const auto& target : required) {
    size_t expected = 0, actual = 0;
    for (const auto& mount : topology->expected) {
      if (mount.mountpoint != target.mountpoint) continue;
      if (mount.mount_class != target.mount_class || mount.parent != target.parent ||
          mount.filesystem_type != target.filesystem_type || mount.read_only != target.read_only ||
          mount.noexec != target.noexec || mount.nosuid != target.nosuid || mount.nodev != target.nodev) return false;
      expected++;
    }
    for (const auto& entry : topology->anchors) {
      const auto& mount = entry.second;
      if (mount.mountpoint == target.mountpoint) {
        if (mount.mount_class != target.mount_class) return false;
        actual++;
      } else if (target.mountpoint != "/" && IsAtOrBelowMountpoint(mount.mountpoint, target.mountpoint)) return false;
    }
    if (expected != 1 || actual != 1) return false;
  }
  return true;
}

bool IsPipxNativeProject(const std::string& project) {
  if (project.empty() || project.size() > 214 || project.front() == '-' || project.back() == '-') return false;
  char previous = 0;
  for (char c : project) {
    if (!((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') || (c == '-' && previous == '-')) return false;
    previous = c;
  }
  return true;
}

bool IsPipxNativeEnvironment(const std::string& environment, const ProcessState::GroupState& parent) {
  if (!IsPipxNativeProject(parent.pipx_native_project)) return false;
  if (parent.pipx_native_operation != "run") {
    return (parent.pipx_native_operation == "install" || parent.pipx_native_operation == "upgrade") &&
        environment == "/haa-site/pipx/venvs/" + parent.pipx_native_project;
  }
  // Pinned pipx uses the first 15 lowercase SHA256 digits for its temporary
  // environment. This shape is only an ABI bound; live creator/image/role
  // checks still authenticate the query, never the native cache name.
  const std::string prefix = "/haa-site/pipx/.cache/";
  if (environment.size() != prefix.size() + 15 || environment.compare(0,prefix.size(),prefix) != 0) return false;
  for (size_t i = prefix.size(); i < environment.size(); ++i) {
    const char c = environment[i];
    if (!((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))) return false;
  }
  return true;
}

std::string PipxNativePythonEnvironment(const std::string& executable, const ProcessState::GroupState& parent) {
  const std::string suffix = "/bin/python";
  if (executable.size() <= suffix.size() || executable.compare(executable.size()-suffix.size(),suffix.size(),suffix) != 0) return "";
  const std::string environment = executable.substr(0,executable.size()-suffix.size());
  return IsPipxNativeEnvironment(environment,parent) ? environment : "";
}

bool IsPipxNativeProgram(const std::string& code) {
  const std::string prefix = kPipxNativeProgramPrefix, suffix = kPipxNativeProgramSuffix;
  if (code.size() <= prefix.size() + suffix.size() || code.size() > prefix.size() + suffix.size() + 20 ||
      code.compare(0, prefix.size(), prefix) != 0 || code.compare(code.size() - suffix.size(), suffix.size(), suffix) != 0) return false;
  const std::string version = code.substr(prefix.size(), code.size() - prefix.size() - suffix.size());
  size_t dots = 0, digits = 0;
  for (char c : version) {
    if (c == '.') { if (digits == 0 || ++dots > 2) return false; digits = 0; }
    else if (c >= '0' && c <= '9') { if (++digits > 6) return false; }
    else return false;
  }
  return dots == 2 && digits > 0;
}

bool ParsePipxNativeRoot(const gvisor::sentry::ExecveInfo& message, const TopologyState* topology,
                         ProcessState::GroupState* group) {
  if (group == nullptr || !IsPipxNativeTopology(topology) ||
      message.binary_path() != "/usr/local/bin/python3.14" || message.execfn() != "/usr/local/bin/python" ||
      !IsPinnedReadOnlyRootPath(topology, message.binary_path()) || !IsPinnedReadOnlyRootPath(topology, message.execfn()) ||
      !IsUVToolImage(topology, kUVToolPath) || message.argv_size() < 7 ||
      message.argv(0) != message.execfn() || message.argv(1) != "-I" || message.argv(2) != "-S" ||
      message.argv(3) != "-B" || message.argv(4) != "-c" || !IsPipxNativeProgram(message.argv(5))) return false;
  const std::string operation = message.argv(6);
  std::string project, requirement;
  if (operation == "--version") {
    if (message.argv_size() != 7) return false;
  } else {
    if (operation == "run") {
      if (message.argv_size() != 17 || message.argv(7) != "--backend" || message.argv(8) != "uv" ||
          message.argv(9) != "--python" || message.argv(10) != "/usr/local/bin/python" ||
          message.argv(11) != "--no-cache" || message.argv(12) != "--no-path-check" ||
          message.argv(13) != "--spec" || message.argv(15) != kPipxNativePipArguments ||
          message.argv(16) != "helox-pipx-inert-query") return false;
      requirement = message.argv(14);
    } else {
      if ((operation != "install" && operation != "upgrade") || message.argv_size() != (operation == "install" ? 13 : 11) ||
        message.argv(8) != "--backend" || message.argv(9) != "uv" ||
        message.argv(message.argv_size()-1) != kPipxNativePipArguments) return false;
      if (operation == "install" && (message.argv(10) != "--python" || message.argv(11) != "/usr/local/bin/python")) return false;
      requirement = message.argv(7);
    }
    if (requirement.empty() || requirement.size() > 4096) return false;
    size_t end = 0;
    while (end < requirement.size()) {
      const char c = requirement[end];
      if (!((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-')) break;
      end++;
    }
    project = requirement.substr(0,end);
    if (!IsPipxNativeProject(project) || (operation == "upgrade" && requirement != project)) return false;
    if (operation == "run" && (project == "helox-pipx-inert-query" || requirement.size() <= end+2 ||
        requirement.compare(end,2,"==") != 0 || requirement.find_first_of(",[]*",end) != std::string::npos)) return false;
    if (end < requirement.size() && requirement[end] != '<' && requirement[end] != '>' && requirement[end] != '=' &&
        requirement[end] != '!' && requirement[end] != '~' && requirement[end] != '[') return false;
    for (unsigned char c : requirement) if (c <= 32 || c >= 127 || c == '\\' || c == '\'' || c == '"') return false;
  }
  group->pipx_native_image = ProcessState::PipxNativeImage::kRoot;
  group->pipx_native_operation = operation;
  group->pipx_native_project = project;
  group->pipx_native_requirement = requirement;
  return true;
}

bool IsPipxNativeActiveGroup(const ProcessState::GroupState& group) {
  return group.role == ProcessState::Role::kArtifact && !group.root_eligible && group.root_consumed &&
      !group.trusted_control_network_active && !group.demotion_pending && !group.launch_target_pending && !group.handoff_target_pending;
}

const ProcessState::GroupState* PipxNativeCreator(const gvisor::common::ContextData& context,
    const ProcessState::GroupState& group, const ProcessState& state, ProcessState::PipxNativeImage kind) {
  if (!IsPipxNativeActiveGroup(group) || group.provenance != ProcessState::Provenance::kCloneChild ||
      !context.is_exec_session() || context.parent_thread_group_id() != group.clone_creator_group_id ||
      state.expected_groups.find(context.thread_group_id()) != state.expected_groups.end() || !HasExactCloneCreator(group,state)) return nullptr;
  const auto& parent = state.groups.find(group.clone_creator_group_id)->second;
  if (!IsPipxNativeActiveGroup(parent) || parent.pipx_native_image != kind ||
      state.expected_groups.find(group.clone_creator_group_id) != state.expected_groups.end() ||
      (parent.pipx_native_operation != "install" && parent.pipx_native_operation != "upgrade" && parent.pipx_native_operation != "run") ||
      !IsPipxNativeProject(parent.pipx_native_project)) return nullptr;
  if (kind == ProcessState::PipxNativeImage::kRoot) {
    return parent.provenance == ProcessState::Provenance::kDirectExecRoot && parent.current_image_class == ProcessClass::kPython ? &parent : nullptr;
  }
  if (kind != ProcessState::PipxNativeImage::kUV || parent.provenance != ProcessState::Provenance::kCloneChild ||
      parent.current_image_class != ProcessClass::kUV || !HasExactCloneCreator(parent,state)) return nullptr;
  const auto& root = state.groups.find(parent.clone_creator_group_id)->second;
  return IsPipxNativeActiveGroup(root) && root.provenance == ProcessState::Provenance::kDirectExecRoot &&
      root.current_image_class == ProcessClass::kPython && root.pipx_native_image == ProcessState::PipxNativeImage::kRoot &&
      root.pipx_native_operation == parent.pipx_native_operation && root.pipx_native_project == parent.pipx_native_project &&
      root.pipx_native_requirement == parent.pipx_native_requirement &&
      state.expected_groups.find(parent.clone_creator_group_id) == state.expected_groups.end() ? &parent : nullptr;
}

// The pinned pipx/packaging producer sorts and deduplicates conjunctions.
// Compare only the same literal atoms bound to the live controller root. This
// never interprets a new version expression or admits a different project.
bool IsSamePipxNativeRequirement(const std::string& expected, const std::string& actual,
                                 const std::string& project) {
  if (expected == actual) return true;
  if (!IsPipxNativeProject(project) || expected.size() > 4096 || actual.size() > 4096) return false;
  auto atoms = [&](const std::string& text, std::set<std::string>* result) {
    if (text.size() <= project.size() || text.compare(0,project.size(),project) != 0) return false;
    size_t start = project.size();
    for (;;) {
      const size_t comma = text.find(',',start);
      const size_t end = comma == std::string::npos ? text.size() : comma;
      if (start == end) return false;
      result->insert(text.substr(start,end-start));
      if (comma == std::string::npos) return true;
      start = comma+1;
    }
  };
  std::set<std::string> left,right;
  return atoms(expected,&left) && atoms(actual,&right) && left == right;
}

bool IsExactPipxNativeUVChild(const gvisor::sentry::ExecveInfo& message, const TopologyState* topology,
                            const ProcessState::GroupState& parent) {
  if (!IsPipxNativeTopology(topology) || !IsUVToolImage(topology,message.binary_path()) ||
      message.binary_path() != kUVToolPath || message.execfn() != kUVToolPath ||
      message.argv_size() < 2 || message.argv(0) != kUVToolPath) return false;
  if (message.argv_size() == 2 && message.argv(1) == "--version") return true;
  if (message.argv_size() == 7 && message.argv(1) == "venv" &&
      message.argv(2) == "--python" && message.argv(3) == "/usr/local/bin/python" &&
      message.argv(4) == "--allow-existing" && message.argv(5) == "--quiet" && IsPipxNativeEnvironment(message.argv(6),parent)) return true;
  const bool upgrade = parent.pipx_native_operation == "upgrade";
  if (message.argv_size() != (upgrade ? 16 : 15) || message.argv(1) != "pip" || message.argv(2) != "install" ||
      message.argv(3) != "--python" || PipxNativePythonEnvironment(message.argv(4),parent).empty() || message.argv(5) != "--quiet") return false;
  int at = 6;
  if (upgrade && message.argv(at++) != "--upgrade") return false;
  for (const char* value : {"--no-build", "--no-sources", "--no-config", "--no-cache", "--no-python-downloads",
      "--no-managed-python", "--link-mode=copy", "--index-url=https://pypi.org/simple/"}) if (message.argv(at++) != value) return false;
  return IsSamePipxNativeRequirement(parent.pipx_native_requirement,message.argv(at),parent.pipx_native_project);
}

bool IsExactPipxNativePythonQuery(const gvisor::sentry::ExecveInfo& message, const TopologyState* topology,
    const ProcessState::GroupState& parent, bool metadata, std::string* environment) {
  if (environment == nullptr || !IsPipxNativeTopology(topology) || message.binary_path() != "/usr/local/bin/python3.14" ||
      !IsPinnedReadOnlyRootPath(topology,message.binary_path())) return false;
  const std::string env = PipxNativePythonEnvironment(message.execfn(),parent);
  const bool alias = !env.empty();
  if ((!alias && (metadata || (message.execfn() != "/usr/local/bin/python" && message.execfn() != message.binary_path()) ||
      !IsPinnedReadOnlyRootPath(topology,message.execfn())))) return false;
  if (message.argv_size() != (metadata ? 3 : 5) || message.argv(0) != message.execfn()) return false;
  if (metadata) {
    if (message.argv(1) != "-c" || message.argv(2) != kPipxNativeMetadataQuery) return false;
  } else if (message.argv(1) != "-I" || message.argv(2) != "-B" || message.argv(3) != "-c" ||
             !IsExactUVPythonQueryProgram(message,topology)) return false;
  *environment = alias ? env : "";
  return true;
}


bool IsExactPNPMNodeQuery(const gvisor::sentry::ExecveInfo& message,
                          const TopologyState* topology) {
  return message.binary_path() == "/usr/local/bin/node" &&
      message.execfn() == message.binary_path() && message.argv_size() == 2 &&
      (message.argv(0) == "node" || message.argv(0) == message.binary_path()) &&
      message.argv(1) == "--version" &&
      IsPinnedReadOnlyRootPath(topology, message.binary_path()) &&
      IsPNPMToolImage(topology, kPNPMToolPath);
}

bool HasExactCargoResolverCreator(const gvisor::common::ContextData& context,
                                  const ProcessState::GroupState& group,
                                  const ProcessState& state) {
  if (group.role != ProcessState::Role::kControl ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state) ||
      context.parent_thread_group_id() != group.clone_creator_group_id) return false;
  const auto& parent = state.groups.find(group.clone_creator_group_id)->second;
  const auto expected = state.expected_groups.find(group.clone_creator_group_id);
  return parent.role == ProcessState::Role::kControl &&
      parent.provenance == ProcessState::Provenance::kDirectExecRoot &&
      parent.current_image_class == ProcessClass::kCargo && !parent.root_eligible &&
      parent.root_consumed && parent.trusted_control_network_active &&
      !parent.demotion_pending && !parent.launch_target_pending && !parent.handoff_target_pending &&
      expected != state.expected_groups.end() && expected->second.start_time_ns == parent.start_time_ns &&
      expected->second.process_class == ProcessClass::kCargo;
}

// Exact info-only SDK query observed from the admitted Cargo driver. Compile
// options, artifact names, and print variants that trigger compilation are excluded.
bool IsExactCargoRustcMetadataQuery(const gvisor::sentry::ExecveInfo& message) {
  if (message.binary_path() != kRustcBinary || message.execfn() != kRustcBinary ||
      message.argv_size() < 22 || message.argv_size() > 24 || message.argv(0) != kRustcBinary) return false;
  std::vector<std::string> expected{kRustcBinary, "-", "--crate-name", "___", "--print=file-names"};
  if (message.argv(5) == "--target") {
    expected.push_back("--target"); expected.push_back("x86_64-unknown-linux-gnu");
  }
  for (const char* kind : {"bin", "rlib", "dylib", "cdylib", "staticlib", "proc-macro"}) {
    expected.push_back("--crate-type"); expected.push_back(kind);
  }
  for (const char* print : {"--print=sysroot", "--print=split-debuginfo", "--print=crate-name", "--print=cfg", "-Wwarnings"}) expected.push_back(print);
  if (message.argv_size() != static_cast<int>(expected.size())) return false;
  for (size_t i = 0; i < expected.size(); ++i) if (message.argv(static_cast<int>(i)) != expected[i]) return false;
  return true;
}

bool IsCargoBuildDriverGroup(const ProcessState::GroupState& group) {
  return group.role == ProcessState::Role::kArtifact &&
      group.provenance == ProcessState::Provenance::kDirectExecRoot &&
      group.current_image_class == ProcessClass::kCargo && !group.root_eligible &&
      group.root_consumed && !group.trusted_control_network_active &&
      !group.demotion_pending && !group.launch_target_pending && !group.handoff_target_pending;
}

bool HasExactCargoBuildDriverCreator(const gvisor::common::ContextData& context,
                                    const ProcessState::GroupState& group,
                                    const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state) ||
      context.parent_thread_group_id() != group.clone_creator_group_id) return false;
  return IsCargoBuildDriverGroup(state.groups.find(group.clone_creator_group_id)->second);
}

bool IsCargoBuildProgramProducerGroup(const ProcessState::GroupState& group, const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact || group.provenance != ProcessState::Provenance::kCloneChild ||
      group.current_image_class != ProcessClass::kCargoBuildProgram || !group.cargo_build_program_active ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending || !HasExactCloneCreator(group,state)) return false;
  return IsCargoBuildDriverGroup(state.groups.find(group.clone_creator_group_id)->second);
}

bool HasExactCargoBuildProgramCreator(const gvisor::common::ContextData& context,
                                     const ProcessState::GroupState& group, const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact || group.provenance != ProcessState::Provenance::kCloneChild ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group,state) || context.parent_thread_group_id() != group.clone_creator_group_id) return false;
  return IsCargoBuildProgramProducerGroup(state.groups.find(group.clone_creator_group_id)->second,state);
}

bool IsCargoBuildCompilerProducerGroup(const ProcessState::GroupState& group, const ProcessState& state) {
  if(group.role!=ProcessState::Role::kArtifact || group.provenance!=ProcessState::Provenance::kCloneChild ||
     group.current_image_class!=ProcessClass::kCargoRustc || !group.cargo_build_rustc_active ||
     group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
     group.demotion_pending || group.launch_target_pending || group.handoff_target_pending || !HasExactCloneCreator(group,state))return false;
  return IsCargoBuildDriverGroup(state.groups.find(group.clone_creator_group_id)->second);
}

bool HasExactCargoBuildCompilerCreator(const gvisor::common::ContextData& context,
                                      const ProcessState::GroupState& group,const ProcessState& state) {
  if(group.role!=ProcessState::Role::kArtifact || group.provenance!=ProcessState::Provenance::kCloneChild ||
     group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
     group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
     !HasExactCloneCreator(group,state) || context.parent_thread_group_id()!=group.clone_creator_group_id)return false;
  return IsCargoBuildCompilerProducerGroup(state.groups.find(group.clone_creator_group_id)->second,state);
}

bool IsCargoBuildGccProducerGroup(const ProcessState::GroupState& group,const ProcessState& state) {
  if(group.role!=ProcessState::Role::kArtifact || group.provenance!=ProcessState::Provenance::kCloneChild ||
     group.current_image_class!=ProcessClass::kGoBuildGcc || !group.go_build_tool_active ||
     group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
     group.demotion_pending || group.launch_target_pending || group.handoff_target_pending || !HasExactCloneCreator(group,state))return false;
  return IsCargoBuildCompilerProducerGroup(state.groups.find(group.clone_creator_group_id)->second,state);
}

bool HasExactCargoBuildGccChildCreator(const gvisor::common::ContextData& context,
                                     const ProcessState::GroupState& group,const ProcessState& state) {
  if(group.role!=ProcessState::Role::kArtifact || group.provenance!=ProcessState::Provenance::kCloneChild ||
     group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
     group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
     !HasExactCloneCreator(group,state) || context.parent_thread_group_id()!=group.clone_creator_group_id)return false;
  return IsCargoBuildGccProducerGroup(state.groups.find(group.clone_creator_group_id)->second,state);
}

bool IsCargoBuildCollect2ProducerGroup(const ProcessState::GroupState& group,const ProcessState& state) {
  if(group.role!=ProcessState::Role::kArtifact || group.provenance!=ProcessState::Provenance::kCloneChild ||
     group.current_image_class!=ProcessClass::kGoBuildCollect2 || !group.go_build_tool_active ||
     group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
     group.demotion_pending || group.launch_target_pending || group.handoff_target_pending || !HasExactCloneCreator(group,state))return false;
  return IsCargoBuildGccProducerGroup(state.groups.find(group.clone_creator_group_id)->second,state);
}

bool HasExactCargoBuildCollect2ChildCreator(const gvisor::common::ContextData& context,
                                          const ProcessState::GroupState& group,const ProcessState& state) {
  if(group.role!=ProcessState::Role::kArtifact || group.provenance!=ProcessState::Provenance::kCloneChild ||
     group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
     group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
     !HasExactCloneCreator(group,state) || context.parent_thread_group_id()!=group.clone_creator_group_id)return false;
  return IsCargoBuildCollect2ProducerGroup(state.groups.find(group.clone_creator_group_id)->second,state);
}

// Used only to report the actual launcher creator; it grants no permission.
bool HasExactCargoLldLauncherCreator(const gvisor::common::ContextData& context,
                                    const ProcessState::GroupState& group,const ProcessState& state) {
  if(group.role!=ProcessState::Role::kArtifact || group.provenance!=ProcessState::Provenance::kCloneChild ||
     group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
     group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
     !HasExactCloneCreator(group,state) || context.parent_thread_group_id()!=group.clone_creator_group_id)return false;
  const auto& creator=state.groups.find(group.clone_creator_group_id)->second;
  if(creator.current_image_class!=ProcessClass::kCargoLldLauncher || !creator.cargo_lld_launcher_active ||
     creator.role!=ProcessState::Role::kArtifact || creator.provenance!=ProcessState::Provenance::kCloneChild ||
     creator.root_eligible || !creator.root_consumed || creator.trusted_control_network_active ||
     creator.demotion_pending || creator.launch_target_pending || creator.handoff_target_pending || !HasExactCloneCreator(creator,state))return false;
  return IsCargoBuildCollect2ProducerGroup(state.groups.find(creator.clone_creator_group_id)->second,state);
}

bool IsExactCargoNativeCCInvocation(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path()=="/usr/bin/x86_64-linux-gnu-gcc-12" &&
      message.execfn()=="/usr/bin/cc" && message.argv_size()>0 &&
      (message.argv(0)=="cc" || message.argv(0)=="/usr/bin/cc");
}

bool IsGoBuildDriverGroup(const ProcessState::GroupState& group) {
  return group.role == ProcessState::Role::kArtifact &&
      group.provenance == ProcessState::Provenance::kDirectExecRoot &&
      group.current_image_class == ProcessClass::kGo && !group.root_eligible &&
      group.root_consumed && !group.trusted_control_network_active &&
      !group.demotion_pending && !group.launch_target_pending && !group.handoff_target_pending;
}

bool HasExactGoBuildDriverCreator(const gvisor::common::ContextData& context,
                                 const ProcessState::GroupState& group,
                                 const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      group.root_eligible || !group.root_consumed ||
      group.trusted_control_network_active || group.demotion_pending ||
      group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state) ||
      context.parent_thread_group_id() != group.clone_creator_group_id) return false;
  const auto& parent = state.groups.find(group.clone_creator_group_id)->second;
  return IsGoBuildDriverGroup(parent);
}

// Only Cgo and the Go SDK linker produce external GCC children. Compile and
// asm retain their SDK reads without this subprocess authority.
bool IsGoBuildCompilerProducerGroup(const ProcessState::GroupState& group,
                             const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      (group.current_image_class != ProcessClass::kGoBuildCgo &&
       group.current_image_class != ProcessClass::kGoBuildLink) || !group.go_build_tool_active ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state)) return false;
  return IsGoBuildDriverGroup(state.groups.find(group.clone_creator_group_id)->second);
}

bool HasExactGoBuildGccCreator(const gvisor::common::ContextData& context,
                             const ProcessState::GroupState& group,
                             const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state) ||
      context.parent_thread_group_id() != group.clone_creator_group_id) return false;
  const auto& parent = state.groups.find(group.clone_creator_group_id)->second;
  return IsGoBuildDriverGroup(parent) || IsGoBuildCompilerProducerGroup(parent, state);
}

bool IsGoBuildGccProducerGroup(const ProcessState::GroupState& group,
                             const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      group.current_image_class != ProcessClass::kGoBuildGcc || !group.go_build_tool_active ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state)) return false;
  const auto& parent = state.groups.find(group.clone_creator_group_id)->second;
  return IsGoBuildDriverGroup(parent) || IsGoBuildCompilerProducerGroup(parent, state);
}

bool HasExactGoBuildGccChildCreator(const gvisor::common::ContextData& context,
                             const ProcessState::GroupState& group,
                             const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state) ||
      context.parent_thread_group_id() != group.clone_creator_group_id) return false;
  return IsGoBuildGccProducerGroup(state.groups.find(group.clone_creator_group_id)->second, state);
}

bool IsGoBuildCollect2ProducerGroup(const ProcessState::GroupState& group,
                                  const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      group.current_image_class != ProcessClass::kGoBuildCollect2 || !group.go_build_tool_active ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state)) return false;
  return IsGoBuildGccProducerGroup(state.groups.find(group.clone_creator_group_id)->second, state);
}

bool HasExactGoBuildNativeLinkerCreator(const gvisor::common::ContextData& context,
                                      const ProcessState::GroupState& group,
                                      const ProcessState& state) {
  if (group.role != ProcessState::Role::kArtifact ||
      group.provenance != ProcessState::Provenance::kCloneChild ||
      group.root_eligible || !group.root_consumed || group.trusted_control_network_active ||
      group.demotion_pending || group.launch_target_pending || group.handoff_target_pending ||
      !HasExactCloneCreator(group, state) ||
      context.parent_thread_group_id() != group.clone_creator_group_id) return false;
  return IsGoBuildCollect2ProducerGroup(state.groups.find(group.clone_creator_group_id)->second, state);
}

// Shape alone is diagnostic only. Admission also requires the sealed runtime
// namespace and the exact bounded environment below.
bool IsExactLdconfigCacheQuery(const gvisor::sentry::ExecveInfo& message) {
  return (message.binary_path() == "/sbin/ldconfig" || message.binary_path() == "/usr/sbin/ldconfig") &&
      message.argv_size() == 2 &&
      (message.argv(0) == "/sbin/ldconfig" || message.argv(0) == "/usr/sbin/ldconfig") &&
      message.argv(1) == "-p";
}

bool IsSupportedLdconfigCacheQuery(const gvisor::sentry::ExecveInfo& message,
                                  const TopologyState* topology) {
  if (!IsExactLdconfigCacheQuery(message) ||
      (message.execfn() != "/sbin/ldconfig" && message.execfn() != "/usr/sbin/ldconfig") ||
      !IsPinnedReadOnlyRootPath(topology, message.binary_path()) ||
      !IsPinnedReadOnlyRootPath(topology, "/etc/ld.so.cache") || message.env_size() != 2) return false;
  // The pinned ctypes.util supplies only these two variables. In particular,
  // no artifact-controlled loader, locale path, or executable search path is admitted.
  return (message.env(0) == "LC_ALL=C" && message.env(1) == "LANG=C") ||
         (message.env(1) == "LC_ALL=C" && message.env(0) == "LANG=C");
}

BoundaryMode BoundaryInvocation(const gvisor::sentry::ExecveInfo& message) {
  if (message.execfn() != kBoundaryHelperPath && message.binary_path() != kBoundaryHelperPath) return BoundaryMode::kNone;
  if (message.argv_size() < 2 || message.argv(0) != kBoundaryHelperPath) return BoundaryMode::kNone;
  const std::string& mode = message.argv(1);
  if (mode == kLaunchMode) return BoundaryMode::kLaunch;
  if (mode == kPythonHandoffMode) return BoundaryMode::kPythonHandoff;
  if (mode == kELFHandoffMode) return BoundaryMode::kELFHandoff;
  if (mode == kNodeHandoffMode) return BoundaryMode::kNodeHandoff;
  // npm invokes script-shell as <path> -c <script>; the fixed helper path is
  // the only trust-removal marker, while the script remains opaque.
  if (mode == "-c") return BoundaryMode::kHandoff;
  return BoundaryMode::kNone;
}

const char* BoundaryModeName(BoundaryMode mode) {
  switch (mode) {
    case BoundaryMode::kLaunch: return kLaunchMode;
    case BoundaryMode::kPythonHandoff: return kPythonHandoffMode;
    case BoundaryMode::kELFHandoff: return kELFHandoffMode;
    case BoundaryMode::kNodeHandoff: return kNodeHandoffMode;
    default: return nullptr;
  }
}

bool ValidAdmissionNonce(const std::string& nonce) {
  if (nonce.size() != kDirectExecAdmissionNonceChars) return false;
  for (const char character : nonce) {
    if ((character < '0' || character > '9') && (character < 'a' || character > 'f')) return false;
  }
  return true;
}

bool ValidSessionGeneration(const std::string& generation) {
  if (generation.size() != kControlSessionGenerationChars) return false;
  for (const char character : generation) {
    if ((character < '0' || character > '9') && (character < 'a' || character > 'f')) return false;
  }
  return true;
}

bool ExtractBoundaryAdmissionNonce(const gvisor::sentry::ExecveInfo& message,
                                  BoundaryMode mode, std::string* nonce) {
  const char* expected_mode = BoundaryModeName(mode);
  if (expected_mode == nullptr || nonce == nullptr || message.argv_size() < 3 ||
      message.argv(0) != kBoundaryHelperPath || message.argv(1) != expected_mode) return false;
  *nonce = message.argv(2);
  return ValidAdmissionNonce(*nonce);
}

// Exact controller-owned readiness tuple; diagnostic command shapes grant no authority.
constexpr char kBoundaryReadinessScript[] = R"HAA(init_demoted() {
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
  done <<HAA_STATUS
$(grep -E '^(Uid|Gid|Groups|CapInh|CapPrm|CapEff|CapBnd|CapAmb):' /proc/1/status)
HAA_STATUS
  [ "$uid:$gid:$groups:$cap_inh:$cap_prm:$cap_eff:$cap_bnd:$cap_amb" = 1:1:1:1:1:1:1:1 ]
}
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
  done <<HAA_STATUS
$(grep -E '^(Uid|Gid|Groups|CapInh|CapPrm|CapEff|CapBnd|CapAmb):' /proc/self/status)
HAA_STATUS
  [ "$uid:$gid:$groups:$cap_inh:$cap_prm:$cap_eff:$cap_bnd:$cap_amb" = 1:1:1:1:1:1:1:1 ]
}
init_demoted
already_demoted)HAA";

bool IsExactPipxReadiness(const gvisor::sentry::ExecveInfo& message, bool boundary) {
  const int offset = boundary ? 3 : 0;
  const char* binary = boundary ? kBoundaryHelperPath : "/usr/bin/dash";
  const char* invoked = boundary ? kBoundaryHelperPath : "/bin/sh";
  if (message.binary_path() != binary || message.execfn() != invoked ||
      message.argv_size() != offset + 3) return false;
  if (boundary && (message.argv(0) != kBoundaryHelperPath ||
      message.argv(1) != kLaunchMode || !ValidAdmissionNonce(message.argv(2)))) return false;
  return message.argv(offset) == "/bin/sh" && message.argv(offset + 1) == "-ceu" &&
      message.argv(offset + 2) == kBoundaryReadinessScript;
}

bool ConsumeDirectExecAdmission(ProfileRegistration* registration, BoundaryMode mode,
                                const std::string& nonce) {
  const char* expected_mode = BoundaryModeName(mode);
  if (registration == nullptr || expected_mode == nullptr || !ValidAdmissionNonce(nonce)) return false;
  for (auto pending = registration->pending_admissions.begin(); pending != registration->pending_admissions.end(); ++pending) {
    if (pending->session_generation == registration->session_generation &&
        pending->mode == expected_mode && pending->nonce == nonce &&
        pending->state == ProfileRegistration::AdmissionState::kPending) {
      pending->state = ProfileRegistration::AdmissionState::kConsumed;
      return true;
    }
  }
  return false;
}

// Only the consumed, host-owned admission can bind a root. Neither an argv
// marker nor a descendant exit can supply its external terminal outcome.
bool BindDirectExecAdmissionRoot(ProfileRegistration* registration, BoundaryMode mode,
                                const std::string& nonce, const gvisor::common::ContextData& context) {
  const char* expected_mode = BoundaryModeName(mode);
  if (registration == nullptr || expected_mode == nullptr || !ValidProcessIdentity(context) ||
      !context.is_exec_session() || context.parent_thread_group_id() != 0) return false;
  for (auto& pending : registration->pending_admissions) {
    if (pending.session_generation == registration->session_generation && pending.mode == expected_mode &&
        pending.nonce == nonce && pending.state == ProfileRegistration::AdmissionState::kConsumed &&
        pending.root_group_id == 0 && pending.root_start_time_ns == 0) {
      pending.root_group_id = context.thread_group_id();
      pending.root_start_time_ns = context.thread_group_start_time_ns();
      return true;
    }
  }
  return false;
}

// Called only after the existing profile accepted the direct root target's
// SENTRY_EXEC image. Attempt telemetry and descendants never supply this fact.
bool RetainDirectExecTarget(ProfileRegistration* registration,
                            const gvisor::common::ContextData& context, BoundaryMode mode) {
  const char* expected_mode = BoundaryModeName(mode);
  if (registration == nullptr || expected_mode == nullptr || !ValidProcessIdentity(context)) return false;
  for (auto& pending : registration->pending_admissions) {
    if (pending.session_generation != registration->session_generation ||
        pending.mode != expected_mode ||
        pending.state != ProfileRegistration::AdmissionState::kConsumed ||
        pending.root_group_id != context.thread_group_id() ||
        pending.root_start_time_ns != context.thread_group_start_time_ns()) continue;
    if (pending.target_loaded || pending.terminal_seen) return false;
    pending.target_loaded = true;
    return true;
  }
  // Existing callers may release their slot before the target event arrives.
  return true;
}

bool ValidTerminalWaitStatus(int32_t status) {
  if (status < 0 || status > 65535) return false;
  const uint32_t low = static_cast<uint32_t>(status) & 255;
  if (low == 0) return true;
  const uint32_t signal = low & 127;
  return (status & ~255) == 0 && signal >= 1 && signal <= 64;
}

bool RetainDirectExecTerminal(ProfileRegistration* registration,
                              const gvisor::common::ContextData& context, int32_t status) {
  // Ordinary callers may already have completed and released their slot.
  // Their process reclamation does not create new completion authority.
  if (registration == nullptr) return true;
  for (auto& pending : registration->pending_admissions) {
    if (pending.session_generation != registration->session_generation ||
        pending.state != ProfileRegistration::AdmissionState::kConsumed ||
        pending.root_group_id != context.thread_group_id() ||
        pending.root_start_time_ns != context.thread_group_start_time_ns()) continue;
    if (pending.terminal_seen || !ValidTerminalWaitStatus(status)) return false;
    pending.terminal_seen = true;
    pending.terminal_wait_status = static_cast<uint32_t>(status);
    return true;
  }
  return true;
}

// These are the exact pinned npm launcher and Node interpreter argv tuples
// emitted by the fixed HAA npm lifecycle command. They are intentionally
// duplicated here so a Go-side command drift fails closed at observation.
constexpr char kNpmCLIPath[] = "/usr/local/lib/node_modules/npm/bin/npm-cli.js";
constexpr char kNpmPath[] = "/usr/local/bin/npm";
constexpr char kNodePath[] = "/usr/local/bin/node";
constexpr char kLockGenerationCommand[] =
    "cd /tmp/haa-resolver; HOME=/tmp npm_config_cache=/tmp/cache npm install "
    "--package-lock-only --ignore-scripts --no-audit --no-fund "
    "--registry=https://registry.npmjs.org/ --userconfig=/tmp/haa-user.npmrc "
    "--globalconfig=/tmp/haa-global.npmrc";
constexpr char kLockUpdateCommand[] =
    "cd /tmp/haa-resolver; HOME=/tmp npm_config_cache=/tmp/cache npm update "
    "--package-lock-only --ignore-scripts --no-audit --no-fund --save=false "
    "--registry=https://registry.npmjs.org/ --userconfig=/tmp/haa-user.npmrc "
    "--globalconfig=/tmp/haa-global.npmrc";

CommandPhase ClassifyCommandShape(const gvisor::sentry::ExecveInfo& message) {
  for (int index = 0; index < message.argv_size(); ++index) {
    const std::string& arg = message.argv(index);
    if (arg.find("package-lock-only") != std::string::npos ||
        (arg.find("npm install") != std::string::npos && arg.find("--offline") == std::string::npos)) {
      return CommandPhase::kLockGeneration;
    }
    if (arg.find("package.json") != std::string::npos ||
        arg.find("mkdir -p /tmp/haa-resolver") != std::string::npos) {
      return CommandPhase::kManifestWrite;
    }
    if (arg.find("package-lock.json") != std::string::npos) {
      return CommandPhase::kLockRead;
    }
    if (arg.find("CapInh") != std::string::npos ||
        arg.find("/proc/1/status") != std::string::npos ||
        arg.find("CapAmb") != std::string::npos) {
      return CommandPhase::kReadinessCheck;
    }
  }
  bool has_npm = false;
  bool has_version = false;
  for (int index = 0; index < message.argv_size(); ++index) {
    const std::string& arg = message.argv(index);
    if (arg == "npm" || arg == kNpmPath || arg == kNpmCLIPath) {
      has_npm = true;
    }
    if (arg == "--version" || arg == "-v") {
      has_version = true;
    }
  }
  if (has_npm && has_version) {
    return CommandPhase::kNpmVersion;
  }
  return CommandPhase::kOther;
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
admission() { [ "${#1}" -eq 64 ] || exit 125; case "$1" in *[!0123456789abcdef]*|'\"'\"''\"'\"') exit 125 ;; esac; }
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
  --origin-launch) shift; admission "${1-}"; token="$1"; shift; exec /haa-runtime/haa-boundary --launch "$token" "$@" ;;
  --origin-handoff-python) shift; admission "${1-}"; token="$1"; shift; exec /haa-runtime/haa-boundary --handoff-python "$token" "$@" ;;
  --origin-handoff-elf) shift; admission "${1-}"; token="$1"; shift; exec /haa-runtime/haa-boundary --handoff-elf "$token" "$@" ;;
  --origin-handoff-node) shift; admission "${1-}"; token="$1"; shift; exec /haa-runtime/haa-boundary --handoff-node "$token" "$@" ;;
  --launch|--handoff-python|--handoff-elf|--handoff-node) shift; admission "${1-}"; shift; demote "$@" ;;
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

// The exact Go producer command. Other npm entry points keep their original
// argument predicates and cannot inherit this target from pathname/comm text.
constexpr char kPNPMTargetLifecycleCommand[] = R"HAA(mkdir -p /tmp/.npm; HOME=/tmp npm_config_cache=/tmp/.npm npm_config_script_shell=/haa-runtime/haa-boundary npm --prefix /tmp/haa-pnpm-input/project rebuild --ignore-scripts=false --bin-links=false --foreground-scripts --offline --no-update-notifier --workspaces=false --no-audit --no-fund --dangerously-allow-all-scripts "$1")HAA";
constexpr char kPNPMProjectMount[] = "/tmp/haa-pnpm-input";
constexpr char kBunTargetProjectMount[] = "/tmp/haa-bun-input";
constexpr char kBunTargetLifecycleCommand[] = R"HAA(mkdir -p /tmp/.npm; HOME=/tmp npm_config_cache=/tmp/.npm npm_config_script_shell=/haa-runtime/haa-boundary npm --prefix /tmp/haa-bun-input/project rebuild --ignore-scripts=false --bin-links=false --foreground-scripts --offline --no-update-notifier --workspaces=false --no-audit --no-fund --dangerously-allow-all-scripts "$1")HAA";
constexpr char kBunRuntimeTargetLifecycleCommand[] = R"HAA(mkdir -p /tmp/.npm; PATH=/haa-runtime:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin HOME=/tmp npm_config_cache=/tmp/.npm npm_config_script_shell=/haa-runtime/haa-boundary npm --prefix /tmp/haa-bun-input/project rebuild --ignore-scripts=false --bin-links=false --foreground-scripts --offline --no-update-notifier --workspaces=false --no-audit --no-fund --dangerously-allow-all-scripts "$1")HAA";

bool ValidPNPMTargetLifecyclePath(const std::string& path) {
  const std::string prefix = std::string(kPNPMProjectMount) + "/project/node_modules/.pnpm/";
  if (path.size() > kMaxTopologyMountpointBytes || !HasPrefix(path, prefix.c_str()) || !IsNormalizedAbsolutePath(path)) return false;
  const std::string tail = path.substr(prefix.size());
  const size_t split = tail.find("/node_modules/");
  if (split == std::string::npos) return false;
  const std::string slot = tail.substr(0, split);
  const std::string name = tail.substr(split + strlen("/node_modules/"));
  bool hashed_slot = slot.size() == 120 && slot[87] == '_';
  if (hashed_slot) {
    for (size_t i = 88; i < slot.size(); ++i) {
      const char c = slot[i];
      if (!((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))) { hashed_slot = false; break; }
    }
  }
  if (slot.size() < 2 || slot.size() > 120 || (slot.find('@', 1) == std::string::npos && !hashed_slot) || slot.find('/') != std::string::npos || name.empty()) return false;
  for (unsigned char c : tail) {
    if ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
        c == '@' || c == '+' || c == '_' || c == '.' || c == '-' || c == '/') continue;
    return false;
  }
  const size_t slash = name.find('/');
  if (slash == std::string::npos) return name[0] != '@' && name != "." && name != "..";
  return name[0] == '@' && slash > 1 && slash + 1 < name.size() &&
      name.find('/', slash + 1) == std::string::npos && name.substr(slash + 1) != "." && name.substr(slash + 1) != "..";
}

bool ValidBunTargetLifecyclePath(const std::string& path) {
  const std::string prefix = std::string(kBunTargetProjectMount) + "/project/node_modules/";
  if (path.size() > kMaxTopologyMountpointBytes || !HasPrefix(path, prefix.c_str()) || !IsNormalizedAbsolutePath(path)) return false;
  auto segment = [](const std::string& value) {
    if (value.empty() || value == "." || value == "..") return false;
    for (unsigned char c : value) if (!((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
        (c >= '0' && c <= '9') || c == '_' || c == '.' || c == '-')) return false;
    return true;
  };
  std::string tail = path.substr(prefix.size());
  while (!tail.empty()) {
    size_t slash = tail.find('/');
    std::string name = tail.substr(0, slash);
    if (name.empty()) return false;
    if (name[0] == '@') {
      if (!segment(name.substr(1)) || slash == std::string::npos) return false;
      tail = tail.substr(slash + 1); slash = tail.find('/'); name = tail.substr(0, slash);
    }
    if (!segment(name)) return false;
    if (slash == std::string::npos) return true;
    tail = tail.substr(slash + 1);
    constexpr char nested[] = "node_modules/";
    if (!HasPrefix(tail, nested)) return false;
    tail = tail.substr(sizeof(nested) - 1);
  }
  return false;
}

bool HasExactNPMTargetLifecycleMount(const TopologyState* topology, const std::string& target, bool bun) {
  const char* project_mount = bun ? kBunTargetProjectMount : kPNPMProjectMount;

  if (topology == nullptr || !topology->sealed || !topology->snapshot_seen || topology->namespace_id == 0 ||
      !(bun ? ValidBunTargetLifecyclePath(target) : ValidPNPMTargetLifecyclePath(target)) || !IsPinnedReadOnlyRootPath(topology, kNpmCLIPath) ||
      !IsPinnedReadOnlyRootPath(topology, kNodePath)) return false;
  size_t expected = 0, actual = 0;
  for (const auto& mount : topology->expected) {
    if (mount.mountpoint == project_mount) {
      if (mount.mount_class != "workspace" || mount.parent != "/tmp" || mount.filesystem_type != "9p" ||
          mount.read_only || mount.noexec || mount.nosuid || mount.nodev) return false;
      ++expected;
    } else if (mount.mountpoint != "/" && mount.mountpoint != "/tmp" && IsAtOrBelowMountpoint(target, mount.mountpoint)) return false;
  }
  for (const auto& entry : topology->anchors) {
    const auto& mount = entry.second;
    if (mount.mountpoint == project_mount) {
      if (mount.mount_class != "workspace") return false;
      ++actual;
    } else if (mount.mountpoint != "/" && mount.mountpoint != "/tmp" && IsAtOrBelowMountpoint(target, mount.mountpoint)) return false;
  }
  return expected == 1 && actual == 1;
}

bool IsExactPNPMTargetLifecycleBoundary(const gvisor::sentry::ExecveInfo& message,
                                       const char* profile, const TopologyState* topology) {
  return profile != nullptr && strcmp(profile, kProfileNPM) == 0 &&
      message.binary_path() == kBoundaryHelperPath && message.execfn() == kBoundaryHelperPath &&
      message.argv_size() == 8 && message.argv(0) == kBoundaryHelperPath &&
      message.argv(1) == kLaunchMode && ValidAdmissionNonce(message.argv(2)) &&
      message.argv(3) == "/bin/sh" && message.argv(4) == "-ceu" &&
      message.argv(5) == kPNPMTargetLifecycleCommand && message.argv(6) == "helox-pnpm-target" &&
      HasExactNPMTargetLifecycleMount(topology, message.argv(7), false);
}

// The native runtime and fixed hoisted project are separately Host-attested.
// This identifies only an image/ABI; it confers no CONTROL or network role.
bool IsBunArtifactRuntimeImage(const TopologyState* topology, const std::string& path) {
  if (!IsBunToolImage(topology, path) || !IsPinnedReadOnlyRootPath(topology, kNodePath) ||
      !IsPinnedReadOnlyRootPath(topology, kNpmCLIPath)) return false;
  size_t expected = 0, actual = 0;
  for (const auto& mount : topology->expected) {
    if (mount.mountpoint != kBunTargetProjectMount) continue;
    if (mount.mount_class != "workspace" || mount.parent != "/tmp" || mount.filesystem_type != "9p" ||
        mount.read_only || mount.noexec || mount.nosuid || mount.nodev) return false;
    ++expected;
  }
  for (const auto& entry : topology->anchors) {
    if (entry.second.mountpoint != kBunTargetProjectMount) continue;
    if (entry.second.mount_class != "workspace") return false;
    ++actual;
  }
  return expected == 1 && actual == 1;
}

bool IsExactBunTargetLifecycleBoundary(const gvisor::sentry::ExecveInfo& message,
                                       const char* profile, const TopologyState* topology) {
  return profile != nullptr && strcmp(profile, kProfileNPM) == 0 &&
      message.binary_path() == kBoundaryHelperPath && message.execfn() == kBoundaryHelperPath &&
      message.argv_size() == 8 && message.argv(0) == kBoundaryHelperPath &&
      message.argv(1) == kLaunchMode && ValidAdmissionNonce(message.argv(2)) &&
      message.argv(3) == "/bin/sh" && message.argv(4) == "-ceu" &&
      (message.argv(5) == kBunTargetLifecycleCommand ||
       (message.argv(5) == kBunRuntimeTargetLifecycleCommand && IsBunArtifactRuntimeImage(topology, kBunToolPath))) &&
      message.argv(6) == "helox-bun-target" &&
      HasExactNPMTargetLifecycleMount(topology, message.argv(7), true);
}

bool IsExactPNPMTargetLifecycleArguments(const gvisor::sentry::ExecveInfo& message,
                                        int first, const std::string& target) {
  static constexpr const char* args[] = {"--prefix", "/tmp/haa-pnpm-input/project", "rebuild",
      "--ignore-scripts=false", "--bin-links=false", "--foreground-scripts", "--offline",
      "--no-update-notifier", "--workspaces=false", "--no-audit", "--no-fund", "--dangerously-allow-all-scripts"};
  const int count = static_cast<int>(sizeof(args) / sizeof(args[0]));
  const bool bun = ValidBunTargetLifecyclePath(target);
  if ((!bun && !ValidPNPMTargetLifecyclePath(target)) || message.argv_size() != first + count + 1 || message.argv(first + count) != target) return false;
  for (int i = 0; i < count; ++i) {
    const char* expected = bun && i == 1 ? "/tmp/haa-bun-input/project" : args[i];
    if (message.argv(first + i) != expected) return false;
  }
  return true;
}

bool HasExactPNPMTargetLifecycleCreator(const gvisor::common::ContextData& context,
                                       const ProcessState::GroupState& group, const ProcessState& state) {
  if (group.role != ProcessState::Role::kControl || group.provenance != ProcessState::Provenance::kCloneChild ||
      group.pnpm_target_lifecycle.empty() || group.root_eligible || !group.root_consumed ||
      group.trusted_control_network_active || group.demotion_pending || group.launch_target_pending ||
      group.handoff_target_pending || !HasExactCloneCreator(group, state) || context.parent_thread_group_id() != group.clone_creator_group_id) return false;
  const auto creator = state.groups.find(group.clone_creator_group_id);
  return creator != state.groups.end() && creator->second.role == ProcessState::Role::kControl &&
      creator->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
      creator->second.current_image_class == ProcessClass::kShell &&
      creator->second.pnpm_target_lifecycle == group.pnpm_target_lifecycle;
}

bool MayArmExactPNPMTargetNpmTransition(const gvisor::sentry::ExecveInfo& message,
                                       const char* profile, const ProcessState::GroupState& group,
                                       const ProcessState& state) {
  return profile != nullptr && strcmp(profile, kProfileNPM) == 0 && !group.pnpm_target_npm_pending &&
      !group.pnpm_target_npm_consumed && HasExactPNPMTargetLifecycleCreator(message.context_data(), group, state) &&
      message.binary_path() == kNpmCLIPath && message.execfn() == kNpmPath && message.argv_size() > 0 &&
      message.argv(0) == kNpmPath && IsExactPNPMTargetLifecycleArguments(message, 1, group.pnpm_target_lifecycle);
}

bool IsExactPNPMTargetNpmNodeTransition(const gvisor::sentry::ExecveInfo& message,
                                       const char* profile, int32_t group_id,
                                       const ProcessState::GroupState& group, const ProcessState::ExpectedGroup& expected,
                                       const ProcessState& state) {
  const auto& context = message.context_data();
  return profile != nullptr && strcmp(profile, kProfileNPM) == 0 && context.thread_group_id() == group_id &&
      expected.start_time_ns == context.thread_group_start_time_ns() && expected.process_class == ProcessClass::kNpm &&
      group.pnpm_target_npm_pending && !group.pnpm_target_npm_consumed &&
      HasExactPNPMTargetLifecycleCreator(context, group, state) &&
      message.binary_path() == kNodePath && message.execfn() == kNodePath && message.argv_size() > 1 &&
      message.argv(0) == "node" && message.argv(1) == kNpmPath &&
      IsExactPNPMTargetLifecycleArguments(message, 2, group.pnpm_target_lifecycle);
}

bool IsExactNpmVersionNodeInterpreter(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kNodePath && message.execfn() == kNodePath &&
      message.argv_size() == 3 && message.argv(0) == "node" &&
      message.argv(1) == kNpmPath && message.argv(2) == "--version";
}

bool IsExactResolverNpmVersionBoundary(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kBoundaryHelperPath && message.execfn() == kBoundaryHelperPath &&
      message.argv_size() == 5 && message.argv(0) == kBoundaryHelperPath &&
      message.argv(1) == kLaunchMode && ValidAdmissionNonce(message.argv(2)) &&
      message.argv(3) == "npm" && message.argv(4) == "--version";
}

bool IsExactNpmVersionLauncher(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kNpmCLIPath && message.execfn() == kNpmPath &&
      message.argv_size() == 2 && message.argv(0) == kNpmPath &&
      message.argv(1) == "--version";
}

// This is the complete boundary invocation emitted only by
// NPMResolver's install/update selection. The command string is trusted controller
// source, not artifact input; any resolver-side drift fails closed here.
bool IsExactResolverLockGenerationBoundary(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kBoundaryHelperPath && message.execfn() == kBoundaryHelperPath &&
      message.argv_size() == 6 && message.argv(0) == kBoundaryHelperPath &&
      message.argv(1) == kLaunchMode && ValidAdmissionNonce(message.argv(2)) &&
      message.argv(3) == "/bin/sh" && message.argv(4) == "-ceu" &&
      (message.argv(5) == kLockGenerationCommand || message.argv(5) == kLockUpdateCommand);
}

bool IsExactLockGenerationArguments(const gvisor::sentry::ExecveInfo& message,
                                    int first_argument) {
  static constexpr const char* kArguments[] = {
      "install", "--package-lock-only", "--ignore-scripts", "--no-audit", "--no-fund",
      "--registry=https://registry.npmjs.org/", "--userconfig=/tmp/haa-user.npmrc",
      "--globalconfig=/tmp/haa-global.npmrc",
  };
  static constexpr const char* kUpdateArguments[] = {
      "update", "--package-lock-only", "--ignore-scripts", "--no-audit", "--no-fund",
      "--save=false", "--registry=https://registry.npmjs.org/", "--userconfig=/tmp/haa-user.npmrc",
      "--globalconfig=/tmp/haa-global.npmrc",
  };
  if (message.argv_size() <= first_argument) return false;
  const bool update = message.argv(first_argument) == "update";
  const auto* arguments = update ? kUpdateArguments : kArguments;
  const size_t count = update ? sizeof(kUpdateArguments) / sizeof(kUpdateArguments[0]) :
      sizeof(kArguments) / sizeof(kArguments[0]);
  if (message.argv_size() != first_argument + static_cast<int>(count)) return false;
  for (size_t index = 0; index < count; ++index) {
    if (message.argv(first_argument + static_cast<int>(index)) != arguments[index]) return false;
  }
  return true;
}

bool IsExactLockGenerationNpmLauncher(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kNpmCLIPath && message.execfn() == kNpmPath &&
      message.argv_size() >= 1 && message.argv(0) == kNpmPath &&
      IsExactLockGenerationArguments(message, 1);
}

bool IsExactLockGenerationNpmNodeInterpreter(const gvisor::sentry::ExecveInfo& message) {
  return message.binary_path() == kNodePath && message.execfn() == kNodePath &&
      message.argv_size() >= 2 && message.argv(0) == "node" && message.argv(1) == kNpmPath &&
      IsExactLockGenerationArguments(message, 2);
}

bool MayArmExactLockGenerationNpmNodeTransition(
    const gvisor::sentry::ExecveInfo& message, const char* profile,
    const ProcessState::GroupState& group, const ProcessState& state) {
  const auto& context = message.context_data();
  return profile != nullptr && strcmp(profile, kProfileNPM) == 0 &&
      group.command_phase == CommandPhase::kLockGeneration &&
      group.role == ProcessState::Role::kControl &&
      group.provenance == ProcessState::Provenance::kCloneChild &&
      group.clone_creator_group_id > 0 &&
      HasExactCloneCreator(group, state) &&
      context.parent_thread_group_id() == group.clone_creator_group_id &&
      !group.root_eligible && group.root_consumed &&
      !group.trusted_control_network_active && !group.demotion_pending &&
      !group.launch_target_pending && !group.handoff_target_pending &&
      IsExactLockGenerationNpmLauncher(message);
}

bool IsExactResolverLockGenerationNpmNodeTransition(
    const gvisor::sentry::ExecveInfo& message, const char* profile, int32_t group_id,
    const ProcessState::GroupState& group, const ProcessState::ExpectedGroup& expected,
    const ProcessState& state) {
  const auto& context = message.context_data();
  return profile != nullptr && strcmp(profile, kProfileNPM) == 0 &&
      context.thread_group_id() == group_id &&
      group.command_phase == CommandPhase::kLockGeneration &&
      group.role == ProcessState::Role::kControl &&
      group.provenance == ProcessState::Provenance::kCloneChild &&
      group.clone_creator_group_id > 0 &&
      HasExactCloneCreator(group, state) &&
      context.parent_thread_group_id() == group.clone_creator_group_id &&
      !group.root_eligible && group.root_consumed &&
      !group.trusted_control_network_active && !group.demotion_pending &&
      !group.launch_target_pending && !group.handoff_target_pending &&
      group.lock_generation_npm_node_transition_pending &&
      expected.start_time_ns == context.thread_group_start_time_ns() &&
      expected.process_class == ProcessClass::kNpm &&
      IsExactLockGenerationNpmNodeInterpreter(message);
}

bool IsExactResolverNpmVersionNodeTransition(const gvisor::sentry::ExecveInfo& message,
                                            const char* profile,
                                            int32_t group_id,
                                            const ProcessState::GroupState& group,
                                            const ProcessState::ExpectedGroup& expected) {
  const auto& context = message.context_data();
  return profile != nullptr && strcmp(profile, kProfileNPM) == 0 &&
      context.thread_group_id() == group_id &&
      context.is_exec_session() && context.parent_thread_group_id() == 0 &&
      group.role == ProcessState::Role::kControl &&
      group.provenance == ProcessState::Provenance::kDirectExecRoot &&
      !group.root_eligible && group.root_consumed &&
      group.trusted_control_network_active && !group.demotion_pending &&
      !group.launch_target_pending && !group.handoff_target_pending &&
      group.command_phase == CommandPhase::kNpmVersion &&
      group.npm_version_node_transition_pending &&
      expected.start_time_ns == context.thread_group_start_time_ns() &&
      expected.process_class == ProcessClass::kNpm &&
      IsExactNpmVersionNodeInterpreter(message);
}

bool IsNewGroup(const ProcessState& state, const gvisor::common::ContextData& context) {
  return state.groups.find(context.thread_group_id()) == state.groups.end();
}

bool RegisterGroup(ProcessState* state, const gvisor::common::ContextData& context,
                   ProcessState::Role role, ProcessState::Provenance provenance,
                   bool root_eligible, bool root_consumed,
                   CommandPhase command_phase = CommandPhase::kUnknown) {
  if (state == nullptr || !ValidProcessIdentity(context)) return false;
  if (state->groups.size() >= kMaxTrackedProcessGroups) return false;
  if (!IsNewGroup(*state, context)) return false;
  state->groups.emplace(context.thread_group_id(), ProcessState::GroupState{
      context.thread_group_start_time_ns(), role, provenance, root_eligible,
      root_consumed, false, false, false, false, false, false, false, false, ProcessClass::kUnknown,
      ProcessState::OCIBootstrapStage::kNotOCI, command_phase});
  return true;
}

const char* ProcessClassName(ProcessClass process_class) {
  switch (process_class) {
    case ProcessClass::kShell: return "SHELL";
    case ProcessClass::kPython: return "PYTHON";
    case ProcessClass::kPip: return "PIP";
    case ProcessClass::kUV: return "OTHER";
    case ProcessClass::kPNPM: return "OTHER";
    case ProcessClass::kBun: return "OTHER";
    case ProcessClass::kNode: return "NODE";
    case ProcessClass::kNpm: return "NPM";
    case ProcessClass::kGo: return "GO";
    case ProcessClass::kCargo: return "OTHER";
    case ProcessClass::kCargoTar: return "OTHER";
    case ProcessClass::kCargoRustc: return "OTHER";
    case ProcessClass::kArtifact: return "ARTIFACT";
    case ProcessClass::kSleep: return "SLEEP";
    case ProcessClass::kMkdir: return "MKDIR";
    case ProcessClass::kCat: return "CAT";
    case ProcessClass::kChmod: return "CHMOD";
    case ProcessClass::kUname: return "OTHER";  // Still an unexpected artifact exec.
    case ProcessClass::kProjectBuildBoundary:
    case ProcessClass::kProjectBuildSetpriv:
    case ProcessClass::kGoBuildTool:
    case ProcessClass::kGoBuildCgo:
    case ProcessClass::kGoBuildLink:
    case ProcessClass::kGoBuildGcc:
    case ProcessClass::kGoBuildCc1:
    case ProcessClass::kGoBuildAssembler:
    case ProcessClass::kGoBuildCollect2:
    case ProcessClass::kGoBuildNativeLinker:
    case ProcessClass::kCargoBuildProgram:
    case ProcessClass::kCargoRustLld:
    case ProcessClass::kCargoLldLauncher:
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
    case ProcessClass::kGo:
    case ProcessClass::kArtifact:
    case ProcessClass::kUnknown:
      return ProcessClassName(process_class);
    case ProcessClass::kCargo:
    case ProcessClass::kUV:
    case ProcessClass::kPNPM:
    case ProcessClass::kBun:
    case ProcessClass::kCargoTar:
    case ProcessClass::kCargoRustc: return "OTHER";
    case ProcessClass::kSleep:
    case ProcessClass::kMkdir:
    case ProcessClass::kCat:
    case ProcessClass::kChmod:
    case ProcessClass::kUname:
    case ProcessClass::kProjectBuildBoundary:
    case ProcessClass::kProjectBuildSetpriv:
    case ProcessClass::kGoBuildTool:
    case ProcessClass::kGoBuildCgo:
    case ProcessClass::kGoBuildLink:
    case ProcessClass::kGoBuildGcc:
    case ProcessClass::kGoBuildCc1:
    case ProcessClass::kGoBuildAssembler:
    case ProcessClass::kGoBuildCollect2:
    case ProcessClass::kGoBuildNativeLinker:
    case ProcessClass::kCargoBuildProgram:
    case ProcessClass::kCargoRustLld:
    case ProcessClass::kCargoLldLauncher:
      return "OTHER";
  }
  return "OTHER";
}

const char* RoleName(ProcessState::Role role) {
  switch (role) {
    case ProcessState::Role::kControl: return "CONTROL";
    case ProcessState::Role::kArtifact: return "ARTIFACT";
    default: return "UNKNOWN";
  }
}

const char* ProvenanceName(ProcessState::Provenance provenance) {
  switch (provenance) {
    case ProcessState::Provenance::kOCIRoot: return "OCI_ROOT";
    case ProcessState::Provenance::kDirectExecRoot: return "DIRECT_EXEC_ROOT";
    case ProcessState::Provenance::kCloneChild: return "CLONE_CHILD";
    default: return "UNKNOWN";
  }
}

ProcessState::UnexpectedExecDiagnostic CaptureExecDiagnostic(
                          const ProcessState::GroupState& group,
                          const gvisor::common::ContextData& context,
                          ProcessClass prev_class,
                          ProcessClass curr_class,
                          DiagnosticImage current_image) {
  ProcessState::UnexpectedExecDiagnostic snapshot;
  auto* diag = &snapshot;
  diag->previous_image = group.diagnostic_image;
  diag->current_image = current_image;
  diag->phase = group.command_phase;
  diag->process_id = context.thread_group_id();
  diag->parent_id = context.parent_thread_group_id();
  diag->previous_class = prev_class;
  diag->current_class = curr_class;
  diag->role = group.role;
  diag->provenance = group.provenance;
  diag->root_eligible = group.root_eligible;
  diag->root_consumed = group.root_consumed;
  diag->trusted_control_network_active = group.trusted_control_network_active;
  diag->demotion_pending = group.demotion_pending;
  diag->launch_target_pending = group.launch_target_pending;
  diag->handoff_target_pending = group.handoff_target_pending;
  diag->npm_node_transition_pending = group.npm_node_transition_pending;

  return snapshot;
}

void RecordUnexpectedExec(ProcessState::UnexpectedExecDiagnostic* diag,
                          const ProcessState::UnexpectedExecDiagnostic& snapshot,
                          const char* reason, const char* parent_relation) {
  if (diag == nullptr || diag->present) return;
  *diag = snapshot;
  diag->present = true;
  diag->classification_reason = reason;
  diag->parent_relation = parent_relation;
}

ProcessClass ProcessClassForPath(const std::string& path, const char* profile) {
  if (path == "/bin/sh" || path == "sh" || path == "/usr/bin/dash") return ProcessClass::kShell;
  if (path == "/usr/bin/sleep" || path == "/bin/sleep" || path == "sleep") return ProcessClass::kSleep;
  if (path == "/usr/bin/mkdir" || path == "/bin/mkdir" || path == "mkdir") return ProcessClass::kMkdir;
  if (path == "/usr/bin/cat" || path == "/bin/cat" || path == "cat") return ProcessClass::kCat;
  if (path == "/usr/bin/chmod" || path == "/bin/chmod" || path == "chmod") return ProcessClass::kChmod;
  if (profile == nullptr) return ProcessClass::kUnknown;
  if ((strcmp(profile, kProfileUVResolver) == 0 || IsPipxNativeProfile(profile)) && path == kUVToolPath) return ProcessClass::kUV;
  if (strcmp(profile, kProfilePNPMResolver) == 0 && path == kPNPMToolPath) return ProcessClass::kPNPM;
  if ((strcmp(profile, kProfileBunResolver) == 0 || strcmp(profile, kProfileBunProjectResolver) == 0) && path == kBunToolPath) return ProcessClass::kBun;
  if (strcmp(profile, kProfileGoBuild) == 0 || strcmp(profile, kProfileCargoBuild) == 0) {
    if (path == "/haa-runtime/haa-boundary") return ProcessClass::kProjectBuildBoundary;
    if (path == "/usr/bin/setpriv") return ProcessClass::kProjectBuildSetpriv;
  }
  if (strcmp(profile, kProfileCargoBuild) == 0 && path == "/usr/bin/x86_64-linux-gnu-gcc-12") return ProcessClass::kGoBuildGcc;
  if (strcmp(profile, kProfileCargoBuild) == 0 && path == "/usr/lib/gcc/x86_64-linux-gnu/12/collect2") return ProcessClass::kGoBuildCollect2;
  if (strcmp(profile, kProfileCargoBuild) == 0 && path == "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/bin/rust-lld") return ProcessClass::kCargoRustLld;
  if (strcmp(profile, kProfileCargoBuild) == 0 && path == "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/bin/gcc-ld/ld.lld") return ProcessClass::kCargoLldLauncher;
  if (strcmp(profile, kProfileGoBuild) == 0) {
    if (path == "/usr/bin/x86_64-linux-gnu-gcc-12") return ProcessClass::kGoBuildGcc;
    if (path == "/usr/lib/gcc/x86_64-linux-gnu/12/cc1") return ProcessClass::kGoBuildCc1;
    if (path == "/usr/bin/x86_64-linux-gnu-as") return ProcessClass::kGoBuildAssembler;
    if (path == "/usr/lib/gcc/x86_64-linux-gnu/12/collect2") return ProcessClass::kGoBuildCollect2;
    if (path == "/usr/bin/x86_64-linux-gnu-ld.bfd") return ProcessClass::kGoBuildNativeLinker;
    if (path == "/usr/local/go/pkg/tool/linux_amd64/compile" ||
        path == "/usr/local/go/pkg/tool/linux_amd64/asm") return ProcessClass::kGoBuildTool;
    if (path == "/usr/local/go/pkg/tool/linux_amd64/link") return ProcessClass::kGoBuildLink;
    if (path == "/usr/local/go/pkg/tool/linux_amd64/cgo") return ProcessClass::kGoBuildCgo;
  }
  if (strcmp(profile,kProfilePythonExecution) == 0 && path == kPythonExecutionPath) return ProcessClass::kPython;
  if (strcmp(profile, kProfileNodeExecution) == 0 && path == "/usr/local/bin/node") return ProcessClass::kNode;
  if (strcmp(profile, kProfileNPM) == 0) {
    if (path == kBunToolPath) return ProcessClass::kBun;
    if (path == "/usr/local/bin/node" || path == "node") return ProcessClass::kNode;
    if (path == "/usr/local/bin/npm" || path == "npm") return ProcessClass::kNpm;
    if (path == "/usr/local/lib/node_modules/npm/bin/npm-cli.js") return ProcessClass::kNpm;
  }
  if (IsPipxRuntimeProfile(profile) || IsPipxNativeProfile(profile)) {
    if (path == "/usr/local/bin/python" || path == "/usr/local/bin/python3" || path == "/usr/local/bin/python3.14" || path == "python") return ProcessClass::kPython;
  }
  if (IsPythonProfile(profile)) {
    if (path == "/usr/bin/uname") return ProcessClass::kUname;
    if (path == "/usr/local/bin/python" || path == "/usr/local/bin/python3" || path == "/usr/local/bin/python3.14" || path == "python") return ProcessClass::kPython;
    if (path == "/usr/local/bin/pip" || path == "pip") return ProcessClass::kPip;
  }
  if ((strcmp(profile, kProfileGitHub) == 0 || strcmp(profile, kProfileTerraformProvider) == 0) && path == "/work/artifact") return ProcessClass::kArtifact;
  if ((strcmp(profile, kProfileGoResolver) == 0 || strcmp(profile, kProfileGoBuild) == 0) && path == "/usr/local/go/bin/go") return ProcessClass::kGo;
  if ((strcmp(profile, kProfileCargoResolver) == 0 || strcmp(profile, kProfileCargoBuild) == 0) && path == kCargoBinary) return ProcessClass::kCargo;
  if (strcmp(profile, kProfileCargoResolver) == 0 && path == "/usr/bin/tar") return ProcessClass::kCargoTar;
  if ((strcmp(profile, kProfileCargoResolver) == 0 || strcmp(profile, kProfileCargoBuild) == 0) && path == kRustcBinary) return ProcessClass::kCargoRustc;
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
  return (strcmp(profile, kProfileGitHub) == 0 || strcmp(profile, kProfileTerraformProvider) == 0) && process_class == ProcessClass::kArtifact;
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

const char* ResolverNetworkKind(const gvisor::common::ContextData& context, const ProcessState& state, const char* profile) {
  if (IsTrustedControlNetwork(context,state)) return "trusted-control-network";
  if (profile != nullptr && strcmp(profile,kProfileBunProjectResolver)==0 && IsBunRunInstallGroup(context,state))
    return "trusted-bun-install-network";
  return "network-attempt";
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
  if (strcmp(profile, kProfileGitHub) == 0 || strcmp(profile, kProfileTerraformProvider) == 0) return HasPrefix(path, "/work/");
  if (IsPipxNativeProfile(profile)) {
    return path == "/tmp" || HasPrefix(path,"/tmp/") || path == "/haa-site" || HasPrefix(path,"/haa-site/") ||
        path == "/haa-pipx" || path == "/haa-pipx/pipx-runtime.zip" || path == "/haa-wheels" || HasPrefix(path,"/haa-wheels/");
  }
  if (IsPipxRuntimeProfile(profile)) {
    return path == "/tmp" || HasPrefix(path, "/tmp/") ||
        path == "/haa-pipx" || path == "/haa-pipx/pipx-runtime.zip";
  }
  if ((profile != nullptr && strcmp(profile,kProfilePythonExecution) == 0) || IsPythonProfile(profile) || strcmp(profile, kProfileUVResolver) == 0) {
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

// Cargo configuration is controller input, even when its parent is a build
// workspace. Deny artifact writes on entry (including failed attempts) and on
// resolved results. The actual home/config also lives in the read-only input
// mount, so rename/unlink cannot substitute an unobserved authority.
bool IsCargoBuildConfigurationWrite(const gvisor::common::ContextData& context,
                                    const ProcessState& state, const std::string& path,
                                    uint64_t flags, const char* profile) {
  if (profile == nullptr || strcmp(profile, kProfileCargoBuild) != 0 || !IsWriteCapableOpen(flags)) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || group->role != ProcessState::Role::kArtifact) return false;
  return path == "/tmp/haa-cargo-input/cargo-home/config" ||
      path == "/tmp/haa-cargo-input/cargo-home/config.toml" ||
      IsAtOrBelowMountpoint(path, "/tmp/.cargo");
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
      path == "/haa-runtime/.haa-boundary.tmp" && (flags & ~kOpenLargefile) == 577;
}

ProcessClass FilesystemProcessClass(const gvisor::common::ContextData& context,
                                    const ProcessState& state) {
  const auto* group = FindFilesystemGroup(context, state);
  if (group != nullptr && group->role == ProcessState::Role::kArtifact) {
    return group->current_image_class;
  }
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

// Diagnostic only: exact, fixed subjects avoid retaining a guest path/name.
const char* FaultOpenSubject(const gvisor::common::ContextData& context,
                             const std::string& path, const MountAnchor& anchor) {
  const std::string self = "/proc/" + std::to_string(context.thread_group_id());
  if (path == "/dev/null") return "DEV_NULL";
  if (path == self + "/auxv") return "PROC_SELF_AUXV";
  if (path == self + "/maps") return "PROC_SELF_MAPS";
  if (path == self + "/statm") return "PROC_SELF_STATM";
  if (path == self + "/cgroup") return "PROC_SELF_CGROUP";
  if (path == self + "/comm") return "PROC_SELF_COMM";
  if (context.thread_id() > 0 && path == "/proc/" + std::to_string(context.thread_id()) + "/comm") return "PROC_THREAD_COMM";
  if (context.parent_thread_group_id() > 0 && path == "/proc/" + std::to_string(context.parent_thread_group_id()) + "/comm") return "PROC_PARENT_COMM";
  if (path == self + "/mountinfo") return "PROC_SELF_MOUNTINFO";
  if (path == "/proc/sys/vm/overcommit_memory") return "VM_OVERCOMMIT_MEMORY";
  if (path == "/sys/kernel/mm/transparent_hugepage/hpage_pmd_size") return "THP_PAGE_SIZE";
  if (path == "/sys/fs/cgroup/cpu/cpu.cfs_quota_us" ||
      path == "/sys/fs/cgroup/cpu/cpu.cfs_period_us") return "CGROUP_CPU_QUOTA";
  return anchor.mount_class == "oci-root" ? "OCI_IMAGE" : "OTHER";
}

std::string FaultOpenDiagnostic(const gvisor::syscall::Open& message,
                                const ProcessState& state, const MountAnchor& anchor) {
  const auto* group = FindFilesystemGroup(message.context_data(), state);
  uint64_t locator = 14695981039346656037ULL;
  for (unsigned char byte : message.pathname()) {
    locator ^= byte;
    locator *= 1099511628211ULL;
  }
  uint64_t mount_locator = 14695981039346656037ULL;
  for (unsigned char byte : anchor.mountpoint) {
    mount_locator ^= byte;
    mount_locator *= 1099511628211ULL;
  }
  return std::string("{\"image\":\"") + ProcessClassName(FilesystemProcessClass(message.context_data(), state)) +
      "\",\"kernel_image\":\"" + DiagnosticImageName(group == nullptr ? DiagnosticImage::kUnknown : group->diagnostic_image) +
      "\",\"role\":\"" + RoleName(group == nullptr ? ProcessState::Role::kUnknown : group->role) +
      "\",\"provenance\":\"" + ProvenanceName(group == nullptr ? ProcessState::Provenance::kUnknown : group->provenance) +
      "\",\"subject\":\"" + FaultOpenSubject(message.context_data(), message.pathname(), anchor) +
      "\",\"mount\":\"" + anchor.mount_class + "\",\"flags\":" + std::to_string(message.flags()) +
      ",\"path_locator\":" + std::to_string(locator == 0 ? 1 : locator) +
      ",\"mountpoint_locator\":" + std::to_string(mount_locator == 0 ? 1 : mount_locator) +
      ",\"executable_pinned\":" + (group != nullptr && group->diagnostic_image_pinned ? "true" : "false") +
      ",\"go_driver_creator\":" + (group != nullptr && HasExactGoBuildDriverCreator(message.context_data(), *group, state) ? "true" : "false") +
      ",\"cargo_driver_creator\":" + (group != nullptr && (HasExactCargoResolverCreator(message.context_data(), *group, state) || HasExactCargoBuildDriverCreator(message.context_data(), *group, state)) ? "true" : "false") +
      (group != nullptr && HasExactCargoBuildProgramCreator(message.context_data(), *group, state) ? ",\"cargo_program_creator\":true" : "") +
      ",\"rustc_version_query\":" + (group != nullptr && group->diagnostic_rustc_version ? "true" : "false") +
      ",\"rustc_metadata_query\":" + (group != nullptr && group->diagnostic_rustc_metadata ? "true" : "false") +
      ",\"rustc_argc\":" + std::to_string(group == nullptr ? 0 : group->diagnostic_rustc_argc) +
      ",\"rustc_argv_locator\":" + std::to_string(group == nullptr ? 0 : group->diagnostic_rustc_argv_locator) +
      ",\"executable_locator\":" + std::to_string(group == nullptr ? 0 : group->executable_locator) +
      (group != nullptr && (group->diagnostic_image == DiagnosticImage::kGcc || group->diagnostic_image == DiagnosticImage::kCollect2 || group->diagnostic_image == DiagnosticImage::kLldLauncher || group->diagnostic_image == DiagnosticImage::kRustLld)
          ? std::string(",\"cargo_compiler_creator\":") + ((HasExactCargoBuildCompilerCreator(message.context_data(), *group, state) || HasExactCargoBuildGccChildCreator(message.context_data(), *group, state) || HasExactCargoBuildCollect2ChildCreator(message.context_data(), *group, state) || HasExactCargoLldLauncherCreator(message.context_data(), *group, state)) ? "true" : "false") +
            ",\"native_cc_invocation\":" + (group->diagnostic_native_cc ? "true" : "false") : "") +
      (group != nullptr && group->diagnostic_build_target_image ? std::string(",\"build_target_image\":true") : "") +
      (group != nullptr && group->diagnostic_image == DiagnosticImage::kRustLld ? std::string(",\"lld_same_group\":") + (group->diagnostic_lld_same_group ? "true" : "false") : "") + "}";
}

// Preserve the first native finding without names, contents or mutable comm.
std::string FirstOutsideOpenDiagnostic(const gvisor::syscall::Open& message,
    const ProcessState& state,const MountAnchor& anchor) {
 const auto* group=FindFilesystemGroup(message.context_data(),state);
 uint64_t locator=14695981039346656037ULL;
 for(unsigned char byte:message.pathname()){locator^=byte;locator*=1099511628211ULL;}
 return std::string("{\"image\":\"")+ProcessClassName(FilesystemProcessClass(message.context_data(),state))+
  "\",\"role\":\""+RoleName(group==nullptr?ProcessState::Role::kUnknown:group->role)+
  "\",\"provenance\":\""+ProvenanceName(group==nullptr?ProcessState::Provenance::kUnknown:group->provenance)+
  "\",\"subject\":\""+FaultOpenSubject(message.context_data(),message.pathname(),anchor)+
  "\",\"mount\":\""+anchor.mount_class+"\",\"flags\":"+std::to_string(message.flags())+
  ",\"path_locator\":"+std::to_string(locator==0?1:locator)+"}";
}

// Existing Send stays unchanged. Only a native classification fault can carry
// this compact context; reserve 256 bytes for its fixed envelope header.
std::string FaultOpenWithFirstOutside(const ProcessState& state,const char* reason,const std::string& budget) {
 const std::string& open=state.fault_open_diagnostic;
 const std::string& first=state.first_outside_open_diagnostic;
 if(open.empty()||first.empty()||reason==nullptr||strcmp(reason,"STREAM_FAULT")!=0||!budget.empty()||!state.fault_raw_diagnostic.empty())return open;
 const std::string field=",\"first_outside\":"+first;
 if(open.back()!='}'||open.size()+field.size()>768)return open;
 return open.substr(0,open.size()-1)+field+"}";
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

// Kernel relation to the exact host-attested private executable build volume.
// Its payload stays untrusted; serialized diagnostics never grant permission.
bool IsCargoBuildTargetImage(const TopologyState* topology,const std::string& path) {
  constexpr const char* target="/tmp/haa-cargo-target";
  std::string normalized;
  if(topology==nullptr || !topology->sealed || !topology->snapshot_seen || topology->namespace_id==0 ||
     !NormalizeAbsolutePath(path,&normalized) || normalized!=path || path==target ||
     !IsAtOrBelowMountpoint(path,target) || !IsPinnedReadOnlyRootPath(topology,kCargoBinary))return false;
  size_t declarations=0,anchors=0;
  for(const auto& mount:topology->expected)if(mount.mountpoint==target) {
    if(mount.mount_class!="workspace" || mount.parent!="/tmp" || mount.filesystem_type!="9p" || mount.read_only || mount.noexec)return false;
    ++declarations;
  }
  for(const auto& entry:topology->anchors) {
    const auto& mount=entry.second;
    if(mount.mountpoint==target) {
      if(mount.mount_class!="workspace")return false;
      ++anchors;
    } else if(IsAtOrBelowMountpoint(mount.mountpoint,target) && IsAtOrBelowMountpoint(path,mount.mountpoint))return false;
  }
  return declarations==1 && anchors==1;
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
      "/lib/x86_64-linux-gnu/libdl.so.2",
      "/lib/x86_64-linux-gnu/libffi.so.8",
      "/lib/x86_64-linux-gnu/libgcc_s.so.1",
      "/lib/x86_64-linux-gnu/libgdbm.so.6",
      "/lib/x86_64-linux-gnu/liblzma.so.5",
      "/lib/x86_64-linux-gnu/libm.so.6",
      "/lib/x86_64-linux-gnu/libncursesw.so.6",
      "/lib/x86_64-linux-gnu/libpanelw.so.6",
      "/lib/x86_64-linux-gnu/libpthread.so.0",
      "/lib/x86_64-linux-gnu/libreadline.so.8",
      "/lib/x86_64-linux-gnu/librt.so.1",
      "/lib/x86_64-linux-gnu/libsqlite3.so.0",
      "/lib/x86_64-linux-gnu/libssl.so.3",
      "/lib/x86_64-linux-gnu/libstdc++.so.6",
      "/lib/x86_64-linux-gnu/libtinfo.so.6",
      // glibc compatibility SONAME supplied by the pinned image, like
      // libdl/libpthread. This is not an artifact/vendor exception.
      "/lib/x86_64-linux-gnu/libutil.so.1",
      "/lib/x86_64-linux-gnu/libuuid.so.1",
      "/lib/x86_64-linux-gnu/libz.so.1",
      "/lib/x86_64-linux-gnu/libzstd.so.1",
  };
  for (const char* library : kLibraries) {
    if (MatchesLibraryNameOrVersion(p, library)) return true;
  }
  return false;
}

// Raw ELF programs stay ARTIFACT. Their own sandbox CPU metadata and the
// locked OCI image's native ABI inputs confer no control/network authority.
bool IsGitHubArtifactRuntimeRead(const gvisor::common::ContextData& context,
                                const ProcessState& state,
                                const std::string& path, uint64_t flags,
                                const char* profile, const MountAnchor* anchor) {
  constexpr uint64_t kReadOnlyFlags = kOpenLargefile | 02000000;  // O_CLOEXEC
  if (profile == nullptr || strcmp(profile, kProfileGitHub) != 0 ||
      anchor == nullptr || (flags & ~kReadOnlyFlags) != 0 ||
      !IsAtOrBelowMountpoint(path, anchor->mountpoint)) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || group->role != ProcessState::Role::kArtifact ||
      group->provenance != ProcessState::Provenance::kDirectExecRoot ||
      group->current_image_class != ProcessClass::kArtifact ||
      group->root_eligible || !group->root_consumed ||
      group->trusted_control_network_active || group->demotion_pending ||
      group->launch_target_pending || group->handoff_target_pending) return false;
  if (anchor->mount_class == "system") {
    const std::string self = "/proc/" + std::to_string(context.thread_group_id());
    if (anchor->mountpoint == "/proc") {
      if (flags == 557056 &&
          (path == self + "/maps" || path == "/proc/sys/vm/overcommit_memory")) return true;
      return path == self + "/cgroup" || path == self + "/mountinfo" ||
          path == "/proc/sys/kernel/osrelease";
    }
    return anchor->mountpoint == "/sys/fs/cgroup/cpu" &&
        (path == "/sys/fs/cgroup/cpu/cpu.cfs_quota_us" ||
         path == "/sys/fs/cgroup/cpu/cpu.cfs_period_us");
  }
  if (anchor->mount_class != "oci-root" || anchor->mountpoint != "/") return false;
  std::string library = path;
  if (HasPrefix(library, "/usr/lib/")) library = "/lib/" + library.substr(9);
  for (const char* name : {"libm.so.6", "libstdc++.so.6", "libgcc_s.so.1", "librt.so.1", "libpthread.so.0", "libdl.so.2"}) {
    if (MatchesLibraryNameOrVersion(library,
        (std::string("/lib/x86_64-linux-gnu/") + name).c_str())) return true;
  }
  return false;
}

bool IsExactPinnedPythonLoaderCandidate(const std::string& path) {
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
  for (const char* candidate : kLoaderCandidates) {
    if (path == candidate) return true;
  }
  return false;
}

bool IsPinnedRuntimeRootRead(const gvisor::common::ContextData& context,
                             const ProcessState& state,
                             const std::string& path, uint64_t flags,
                             const char* profile) {
  if (profile == nullptr || strcmp(profile,kProfilePythonExecution) == 0) return false;
  std::string normalized;
  if (!NormalizeAbsolutePath(path, &normalized)) return false;
  if (IsWriteCapableOpen(flags) && normalized != "/dev/null") return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr ||
      (group->provenance != ProcessState::Provenance::kDirectExecRoot &&
       group->provenance != ProcessState::Provenance::kCloneChild)) return false;
  if (strcmp(profile, kProfileGitHub) == 0) {
    return normalized == "/etc/ld.so.cache" ||
        IsExactLibc6(normalized);
  }
  if (!IsPythonRuntimeProfile(profile)) return false;
  if ((IsPipxRuntimeProfile(profile) || IsPipxNativeProfile(profile)) && (normalized == "/usr/local/bin/pip" ||
      normalized == "/etc/pip.conf" || normalized == "/usr/local/lib/python3.14/site-packages" ||
      HasPrefix(normalized, "/usr/local/lib/python3.14/site-packages/"))) return false;
  if (group->role == ProcessState::Role::kArtifact) {
    // comm is mutable by ordinary Python/native code. Artifact reads depend
    // on the observed current executable, never on a claimed thread name.
    const auto image = FilesystemProcessClass(context, state);
    // uname is used by the pinned Python build runtime. Its immutable loader
    // inputs are readable without treating its execution as expected or granting
    // Python's broader runtime surface. No write or role/network privilege.
    if (image == ProcessClass::kUname) {
      return normalized == "/etc/ld.so.cache" || IsExactLibc6(normalized);
    }
    if (image != ProcessClass::kPython) return false;
  } else if (context.process_name() != "python" && context.process_name() != "python3.14" &&
             context.process_name() != "uname" && context.process_name() != "sh") {
    return false;
  }
  const bool exact_loader_candidate = IsExactPinnedPythonLoaderCandidate(path);
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
  const std::string proc_pid = "/proc/" + pid_str;
  const bool exact_self_proc =
      normalized == "/proc/self/environ" || normalized == "/proc/self/maps" ||
      normalized == "/proc/self/status" || normalized == "/proc/self/mounts" ||
      normalized == proc_pid + "/environ" || normalized == proc_pid + "/maps" ||
      normalized == proc_pid + "/status" || normalized == proc_pid + "/mounts";
  if (exact_self_proc || normalized == "/proc/cpuinfo" ||
      normalized == "/proc/mounts" ||
      normalized == "/proc/sys/vm/overcommit_memory" ||
      normalized == "/proc/sys/kernel/random/boot_id" ||
      normalized == "/dev/null" || normalized == "/dev/urandom" ||
      normalized == "/etc/nsswitch.conf" || normalized == "/etc/passwd" || normalized == "/etc/group" ||
      normalized == "/etc/resolv.conf" || normalized == "/etc/hosts" || normalized == "/etc/host.conf" || normalized == "/etc/gai.conf" ||
      normalized == "/etc/services" || normalized == "/etc/protocols" ||
      normalized == "/etc/pip.conf" ||
      normalized == "/etc/os-release" || normalized == "/usr/lib/os-release" ||
      normalized == "/etc/debian_version" || normalized == "/etc/issue" ||
      normalized == "/etc/lsb-release" ||
      normalized == "/etc/ssl" || HasPrefix(normalized, "/etc/ssl/") ||
      normalized == "/usr/lib/ssl" || HasPrefix(normalized, "/usr/lib/ssl/") ||
      normalized == "/etc/ca-certificates" || HasPrefix(normalized, "/etc/ca-certificates/") ||
      normalized == "/usr/share/ca-certificates" || HasPrefix(normalized, "/usr/share/ca-certificates/") ||
      normalized == "/sys/devices/system/cpu/possible") return true;
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

// Host-attested readonly clean input, independently bound to current source
// by the existing complete DATA reader. No path string grants CONTROL.
bool IsNodeExecutionInputPath(const TopologyState* topology, const std::string& path) {
  constexpr const char* input = "/tmp/haa-node-execution";
  std::string normalized;
  if (path.size() > 4096 || !NormalizeAbsolutePath(path,&normalized) || normalized != path || !HasPrefix(path, "/tmp/haa-node-execution/project/node_modules/") ||
      !IsPinnedReadOnlyRootPath(topology, kNodePath)) return false;
  size_t declared = 0, anchors = 0;
  for (const auto& mount : topology->expected) if (mount.mountpoint == input) {
    if (mount.mount_class != "workspace" || mount.parent != "/tmp" || mount.filesystem_type != "9p" ||
        !mount.read_only || mount.noexec || mount.nosuid || mount.nodev) return false;
    ++declared;
  }
  for (const auto& entry : topology->anchors) {
    const auto& mount = entry.second;
    if (mount.mountpoint == input) { if (mount.mount_class != "workspace") return false; ++anchors; }
    else if (IsAtOrBelowMountpoint(mount.mountpoint,input) && IsAtOrBelowMountpoint(path,mount.mountpoint)) return false;
  }
  return declared == 1 && anchors == 1;
}

bool BindNodeExecutionTarget(const gvisor::sentry::ExecveInfo& message, const TopologyState* topology,
                             ProcessState::GroupState* group) {
  if (group == nullptr || !group->node_execution_target.empty() || message.argv_size() < 6 || message.argv_size() > 134 ||
      message.argv(3) != kNodePath || message.argv(4) != "--" || !IsNodeExecutionInputPath(topology,message.argv(5))) return false;
  size_t bytes = 0;
  for (int i=6;i<message.argv_size();++i) { bytes += message.argv(i).size(); if (bytes > 32768 || message.argv(i).find('\0') != std::string::npos) return false; }
  for (int i=3;i<message.argv_size();++i) group->node_execution_target.push_back(message.argv(i));
  return true;
}

bool SameCapturedExecutionTarget(const gvisor::sentry::ExecveInfo& message, const std::vector<std::string>& target, int offset) {
  if (target.empty() || message.argv_size() != offset+static_cast<int>(target.size())) return false;
  for (size_t i=0;i<target.size();++i) if (message.argv(offset+static_cast<int>(i)) != target[i]) return false;
  return true;
}
bool SameNodeExecutionTarget(const gvisor::sentry::ExecveInfo& message, const ProcessState::GroupState& group, int offset) {
  return SameCapturedExecutionTarget(message,group.node_execution_target,offset);
}

bool IsPinnedNodeExecutionRead(const gvisor::common::ContextData& context, const ProcessState& state,
                                const std::string& path, uint64_t flags, const char* profile,
                                const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile,kProfileNodeExecution) != 0 || anchor == nullptr || IsWriteCapableOpen(flags)) return false;
  const auto* group = FindFilesystemGroup(context,state);
  if (group == nullptr || group->role != ProcessState::Role::kArtifact || group->provenance != ProcessState::Provenance::kDirectExecRoot ||
      group->root_eligible || !group->root_consumed || group->trusted_control_network_active || group->demotion_pending ||
      group->launch_target_pending || state.expected_groups.find(context.thread_group_id()) != state.expected_groups.end() ||
      !context.is_exec_session() || context.parent_thread_group_id() != 0) return false;
  const bool node = group->node_execution_active && group->current_image_class == ProcessClass::kNode && !group->handoff_target_pending;
  const bool demoting = group->node_execution_demoting && group->handoff_target_pending;
  if (!node && !demoting) return false;
  if (anchor->mountpoint == "/" && anchor->mount_class == "oci-root") {
    if (demoting) return IsExactPinnedLibraryRead(path) || path == "/etc/nsswitch.conf" || path == "/etc/passwd" || path == "/etc/group";
    return flags == 557056 && (IsExactPinnedLibraryRead(path) || path == kNodePath || path == "/etc/ssl/openssl.cnf" || path == "/usr/share/zoneinfo/Etc/UTC");
  }
  if (node && anchor->mountpoint == "/sys/fs/cgroup/memory" && anchor->mount_class == "system" && flags == 557056)
    return path == "/sys/fs/cgroup/memory/memory.soft_limit_in_bytes" || path == "/sys/fs/cgroup/memory/memory.limit_in_bytes";
  if (node && anchor->mountpoint == "/dev" && anchor->mount_class == "system" && flags == 557056)
    return path == "/dev/null";
  if (node && anchor->mountpoint == "/sys" && anchor->mount_class == "system" && flags == 557056)
    return path == "/sys/devices/system/cpu/online";
  if (anchor->mountpoint != "/proc" || anchor->mount_class != "system") return false;
  const std::string self = "/proc/"+std::to_string(context.thread_group_id());
  return (node && flags == 557056 && (path == "/proc/self/maps" || path == self+"/maps" || path == self+"/cgroup" ||
      path == "/proc/meminfo" || path == "/proc/version_signature" || path == "/proc/sys/vm/overcommit_memory")) ||
      (demoting && (path == "/proc/self/status" || path == self+"/status" || path == "/proc/sys/kernel/cap_last_cap"));
}

// The selected console owns no CONTROL or ambient Python/Pipx authority.
bool IsPythonExecutionTopology(const TopologyState* topology) {
  if (!IsPinnedReadOnlyRootPath(topology,kPythonExecutionPath)) return false;
  size_t declared=0,anchors=0;
  for (const auto& mount:topology->expected) if (mount.mountpoint=="/haa-site") {
    if (mount.mount_class!="workspace" || mount.parent!="/" || mount.filesystem_type!="9p" ||
        !mount.read_only || mount.noexec || mount.nosuid || mount.nodev) return false;
    ++declared;
  }
  for (const auto& entry:topology->anchors) {
    const auto& mount=entry.second;
    if (mount.mountpoint=="/haa-site") { if(mount.mount_class!="workspace")return false; ++anchors; }
    else if (IsAtOrBelowMountpoint(mount.mountpoint,"/haa-site")) return false;
  }
  return declared==1 && anchors==1;
}
bool IsConsoleQualifiedName(const std::string& name) {
  if(name.empty() || name.size()>512)return false;
  bool first=true;
  for(unsigned char c:name) {
    if(c=='.') { if(first)return false; first=true; continue; }
    if(!((c>='a' && c<='z') || (c>='A' && c<='Z') || c=='_' || (!first && c>='0' && c<='9')))return false;
    first=false;
  }
  return !first;
}
bool BindPythonExecutionTarget(const gvisor::sentry::ExecveInfo& message,const TopologyState* topology,ProcessState::GroupState* group) {
  if(group==nullptr || !group->python_execution_target.empty() || !IsPythonExecutionTopology(topology) ||
      message.argv_size()<12 || message.argv_size()>140 || message.argv(3)!=kPythonExecutionPath ||
      message.argv(4)!="-I" || message.argv(5)!="-S" || message.argv(6)!="-B" || message.argv(7)!="-c" ||
      message.argv(8)!=kPythonExecutionProgram || !IsConsoleQualifiedName(message.argv(9)) || !IsConsoleQualifiedName(message.argv(10)) ||
      message.argv(11).empty() || message.argv(11).size()>128)return false;
  for(unsigned char c:message.argv(11))if(c<=32 || c>=127 || c=='/' || c=='\\')return false;
  size_t bytes=0;
  for(int i=12;i<message.argv_size();++i) { bytes+=message.argv(i).size(); if(bytes>32768 || message.argv(i).find('\0')!=std::string::npos)return false; }
  for(int i=3;i<message.argv_size();++i)group->python_execution_target.push_back(message.argv(i));
  return true;
}
bool IsActivePythonExecutionGroup(const gvisor::common::ContextData& context,const ProcessState::GroupState* group) {
  return group!=nullptr && group->role==ProcessState::Role::kArtifact && group->provenance==ProcessState::Provenance::kDirectExecRoot &&
      !group->root_eligible && group->root_consumed && !group->trusted_control_network_active && !group->demotion_pending &&
      !group->launch_target_pending && !group->handoff_target_pending && group->python_execution_active &&
      group->current_image_class==ProcessClass::kPython && context.is_exec_session() && context.parent_thread_group_id()==0;
}
bool IsPinnedPythonExecutionRead(const gvisor::common::ContextData& context,const ProcessState& state,
    const std::string& path,uint64_t flags,const char* profile,const MountAnchor* anchor) {
  if(profile==nullptr || strcmp(profile,kProfilePythonExecution)!=0 || anchor==nullptr || IsWriteCapableOpen(flags))return false;
  const auto* group=FindFilesystemGroup(context,state);
  if(!IsActivePythonExecutionGroup(context,group))return false;
  std::string normalized;
  if(!NormalizeAbsolutePath(path,&normalized) || (path!=normalized && !IsExactPinnedPythonLoaderCandidate(path)))return false;
  if(anchor->mountpoint=="/" && anchor->mount_class=="oci-root") {
    if(normalized=="/usr/local/lib/python3.14/site-packages" || HasPrefix(normalized,"/usr/local/lib/python3.14/site-packages/"))return false;
    return IsExactPinnedPythonLoaderCandidate(path) || IsExactPinnedPythonSystemLibrary(normalized) ||
        IsExactPinnedLibraryRead(normalized) || normalized==kPythonExecutionPath || normalized=="/usr/local/lib/libpython3.14.so.1.0" ||
        normalized=="/usr/local/lib/python314.zip" || normalized=="/usr/local/lib/python3.14" || HasPrefix(normalized,"/usr/local/lib/python3.14/") ||
        normalized=="/usr/share/zoneinfo/Etc/UTC" || normalized=="/etc/localtime";
  }
  if(anchor->mount_class!="system")return false;
  if(anchor->mountpoint=="/proc" && path=="/proc/sys/vm/overcommit_memory")return flags==32768;
  return (anchor->mountpoint=="/dev" && (path=="/dev/null" || path=="/dev/urandom")) ||
      (anchor->mountpoint=="/proc" && (path=="/proc/self/maps" || path=="/proc/"+std::to_string(context.thread_group_id())+"/maps"));
}

// The exact version-only Node child remains ARTIFACT. Reuse only the
// canonical immutable native-loader ABI, not npm scripts/configuration or
// another process's authority. The first actual read was ld.so.cache/557056.
bool IsPinnedPNPMNodeQueryRead(const gvisor::common::ContextData& context,
                              const ProcessState& state, const std::string& path,
                              uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfilePNPMResolver) != 0 ||
      anchor == nullptr || anchor->mountpoint != "/" || anchor->mount_class != "oci-root" ||
      flags != 557056) return false;
  const auto* group = FindFilesystemGroup(context, state);
  return group != nullptr && group->pnpm_node_query_active &&
      group->current_image_class == ProcessClass::kNode &&
      HasExactPNPMQueryCreator(context, *group, state) && IsExactPinnedLibraryRead(path);
}

// The query consumes only the immutable interpreter ABI/stdlib. It cannot
// import installed artifact packages or confer CONTROL/network authority.
bool IsPinnedUVPythonQueryRead(const gvisor::common::ContextData& context,
                               const ProcessState& state, const std::string& path,
                               uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfileUVResolver) != 0 ||
      anchor == nullptr || IsWriteCapableOpen(flags)) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || !group->uv_python_query_active ||
      group->current_image_class != ProcessClass::kPython ||
      !HasExactUVQueryCreator(context, *group, state)) return false;
  if (anchor->mount_class == "system") {
    if (anchor->mountpoint != "/proc") return false;
    return (flags == kOpenLargefile && path == "/proc/sys/vm/overcommit_memory") ||
        (flags == 557056 && path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps");
  }
  if (anchor->mountpoint != "/" || anchor->mount_class != "oci-root") return false;
  std::string normalized;
  if (!NormalizeAbsolutePath(path, &normalized)) return false;
  const bool loader_candidate = IsExactPinnedPythonLoaderCandidate(path);
  if (path != normalized && !loader_candidate) return false;
  if (normalized == "/usr/share/zoneinfo/Etc/UTC") return flags == 557056;
  if (loader_candidate || normalized == "/etc/ld.so.cache" ||
      IsExactPinnedPythonSystemLibrary(normalized) ||
      normalized == "/usr/local/lib/libpython3.14.so.1.0" ||
      normalized == "/usr/local/lib/python314.zip" ||
      normalized == "/usr/local/bin/python" || normalized == "/usr/local/bin/python3.14" ||
      normalized == "/usr/local/lib/python3.14" ||
      normalized == "/usr/local/lib/python3.14/site-packages") return true;
  if (HasPrefix(normalized, "/usr/local/lib/python3.14/site-packages/")) return false;
  return HasPrefix(normalized, "/usr/local/lib/python3.14/");
}

// Observed Bun1.4.4 startup reads for the admitted query or lock-only root.
// Project-only data grants do not apply to the version/revision query profile.
bool IsBunImageStartupRead(const gvisor::common::ContextData& context, const std::string& path,
                           uint64_t flags, const MountAnchor* anchor) {
  // Fixed GNU loader inputs and exact self-startup reads already measured for
  // this same native image. No resolver files, general proc/sys tree or writes.
  if (anchor->mount_class == "oci-root" && anchor->mountpoint == "/" && path == "/" && flags == 622592) return true;
  if (anchor->mount_class == "oci-root" && anchor->mountpoint == "/" && flags == 557056 &&
      (IsExactPinnedLibraryRead(path) || path == "/usr/share/zoneinfo/Etc/UTC")) return true;
  if (anchor->mount_class == "system" && anchor->mountpoint == "/proc") {
    if (path == "/proc/sys/vm/overcommit_memory" || path == "/proc/sys/vm/mmap_min_addr") return flags == 32768;
    if (path == "/proc/" + std::to_string(context.thread_group_id()) + "/statm") return flags == 32768;
    return flags == 557056 && (path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps" ||
        path == "/proc/" + std::to_string(context.thread_group_id()) + "/cgroup");
  }
  if (anchor->mount_class == "system" && anchor->mountpoint == "/sys/fs/cgroup/memory" && flags == 557056) {
    return path == "/sys/fs/cgroup/memory/memory.soft_limit_in_bytes" ||
        path == "/sys/fs/cgroup/memory/memory.limit_in_bytes";
  }
  return anchor->mount_class == "system" && anchor->mountpoint == "/sys" &&
      path == "/sys/devices/system/cpu/online" && flags == 557056;
}

bool IsPinnedBunArtifactRuntimeRead(const gvisor::common::ContextData& context,
                                  const ProcessState& state, const std::string& path,
                                  uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfileNPM) != 0 || anchor == nullptr) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || group->role != ProcessState::Role::kArtifact ||
      group->provenance != ProcessState::Provenance::kCloneChild ||
      group->current_image_class != ProcessClass::kBun || FilesystemProcessClass(context, state) != ProcessClass::kBun ||
      group->root_eligible || !group->root_consumed || group->trusted_control_network_active ||
      group->demotion_pending || group->launch_target_pending || group->handoff_target_pending ||
      context.parent_thread_group_id() != group->clone_creator_group_id || !HasExactCloneCreator(*group, state) ||
      state.expected_groups.find(context.thread_group_id()) != state.expected_groups.end()) return false;
  return IsBunImageStartupRead(context,path,flags,anchor);
}

bool IsPinnedBunRuntimeRead(const gvisor::common::ContextData& context,
                          const ProcessState& state, const std::string& path,
                          uint64_t flags, const char* profile, const MountAnchor* anchor) {
  const bool native_install = profile != nullptr && strcmp(profile,kProfileBunProjectResolver)==0 && IsBunRunInstallGroup(context,state);
  if (profile == nullptr || (strcmp(profile, kProfileBunResolver) != 0 && strcmp(profile, kProfileBunProjectResolver) != 0) ||
      anchor == nullptr ||
      !context.is_exec_session() || (!native_install && context.parent_thread_group_id() != 0)) return false;
  if (native_install && IsBunImageStartupRead(context,path,flags,anchor)) return true;
  const bool overcommit = anchor->mount_class == "system" && anchor->mountpoint == "/proc" && path == "/proc/sys/vm/overcommit_memory" && flags == 32768;
  const bool own_maps = anchor->mount_class == "system" && anchor->mountpoint == "/proc" && path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps" && flags == 557056;
  const bool online_cpus = strcmp(profile, kProfileBunProjectResolver) == 0 && anchor->mount_class == "system" && anchor->mountpoint == "/sys" && path == "/sys/devices/system/cpu/online" && flags == 557056;
  const bool own_cgroup = strcmp(profile, kProfileBunProjectResolver) == 0 && anchor->mount_class == "system" && anchor->mountpoint == "/proc" && path == "/proc/" + std::to_string(context.thread_group_id()) + "/cgroup" && flags == 557056;
  const bool utc_zone = strcmp(profile, kProfileBunProjectResolver) == 0 && anchor->mount_class == "oci-root" && anchor->mountpoint == "/" && path == "/usr/share/zoneinfo/Etc/UTC" && flags == 557056;
  const bool host_configuration = strcmp(profile, kProfileBunProjectResolver) == 0 && anchor->mount_class == "oci-root" && anchor->mountpoint == "/" && (path == "/etc/host.conf" || path == "/etc/nsswitch.conf" || path == "/etc/gai.conf") && flags == 557056;
  const bool resolver_configuration = strcmp(profile, kProfileBunProjectResolver) == 0 && anchor->mount_class == "system" && (anchor->mountpoint == "/etc/resolv.conf" || anchor->mountpoint == "/etc/hosts") && path == anchor->mountpoint && flags == 557056;
  const bool root_directory = strcmp(profile, kProfileBunProjectResolver) == 0 && anchor->mount_class == "oci-root" && anchor->mountpoint == "/" && path == "/" && flags == 622592;
  if (!overcommit && !own_maps && !online_cpus && !own_cgroup && !utc_zone && !host_configuration && !resolver_configuration && !root_directory) return false;
  if (native_install) return FilesystemProcessClass(context,state) == ProcessClass::kBun;
  const auto* group = FindFilesystemGroup(context, state);
  return group != nullptr && group->role == ProcessState::Role::kControl &&
      group->provenance == ProcessState::Provenance::kDirectExecRoot &&
      group->current_image_class == ProcessClass::kBun &&
      FilesystemProcessClass(context, state) == ProcessClass::kBun &&
      !group->root_eligible && group->root_consumed && group->trusted_control_network_active &&
      !group->demotion_pending && !group->launch_target_pending && !group->handoff_target_pending;
}

// Observed GNU/Rust startup inputs of the pinned PNPM ELF. This is not a
// shared library grant for artifacts, descendants, UV or other profiles.
bool IsPinnedPNPMRuntimeRead(const gvisor::common::ContextData& context,
                            const ProcessState& state, const std::string& path,
                            uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfilePNPMResolver) != 0 ||
      anchor == nullptr || flags != 557056 ||
      !context.is_exec_session() || context.parent_thread_group_id() != 0) return false;
  const bool library = anchor->mountpoint == "/" && anchor->mount_class == "oci-root" &&
      (path == "/usr/lib/x86_64-linux-gnu/librt.so.1" || path == "/etc/nsswitch.conf" ||
       path == "/etc/host.conf" || path == "/etc/gai.conf");
  const bool resolver_configuration = anchor->mount_class == "system" &&
      (anchor->mountpoint == "/etc/resolv.conf" || anchor->mountpoint == "/etc/hosts") &&
      path == anchor->mountpoint;
  const bool proc_metadata = anchor->mountpoint == "/proc" && anchor->mount_class == "system" &&
      (path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps" ||
       path == "/proc/sys/vm/overcommit_memory" || path == "/proc/sys/kernel/osrelease" ||
       path == "/proc/" + std::to_string(context.thread_group_id()) + "/cgroup" ||
       path == "/proc/" + std::to_string(context.thread_group_id()) + "/mountinfo");
  const bool cpu_limits = anchor->mountpoint == "/sys/fs/cgroup/cpu" && anchor->mount_class == "system" &&
      (path == "/sys/fs/cgroup/cpu/cpu.cfs_quota_us" || path == "/sys/fs/cgroup/cpu/cpu.cfs_period_us");
  const bool online_cpus = anchor->mountpoint == "/sys" && anchor->mount_class == "system" &&
      path == "/sys/devices/system/cpu/online";
  const bool native_file = anchor->mountpoint == "/haa-runtime" && anchor->mount_class == "helper" &&
      path == kPNPMToolPath;
  const bool device_read = anchor->mountpoint == "/dev" && anchor->mount_class == "system" &&
      (path == "/dev/urandom" || path == "/dev/null");
  if (!library && !resolver_configuration && !proc_metadata && !cpu_limits && !online_cpus && !native_file && !device_read) return false;
  const auto* group = FindFilesystemGroup(context, state);
  return group != nullptr && group->role == ProcessState::Role::kControl &&
      group->provenance == ProcessState::Provenance::kDirectExecRoot &&
      group->current_image_class == ProcessClass::kPNPM &&
      FilesystemProcessClass(context, state) == ProcessClass::kPNPM &&
      !group->root_eligible && group->root_consumed && group->trusted_control_network_active &&
      !group->demotion_pending && !group->launch_target_pending && !group->handoff_target_pending;
}

const ProcessState::GroupState* PipxNativeQueryParent(const gvisor::common::ContextData& context,
    const ProcessState& state, const ProcessState::GroupState& group) {
  if (!group.pipx_native_query_restricted || group.current_image_class != ProcessClass::kPython) return nullptr;
  const auto kind = group.pipx_native_image;
  if (kind != ProcessState::PipxNativeImage::kUVQuery && kind != ProcessState::PipxNativeImage::kMetadataQuery) return nullptr;
  const auto* parent = PipxNativeCreator(context,group,state,kind == ProcessState::PipxNativeImage::kUVQuery
      ? ProcessState::PipxNativeImage::kUV : ProcessState::PipxNativeImage::kRoot);
  if (parent == nullptr || group.pipx_native_project != parent->pipx_native_project ||
      group.pipx_native_operation != parent->pipx_native_operation || group.pipx_native_requirement != parent->pipx_native_requirement) return nullptr;
  if ((!group.pipx_native_environment.empty() && !IsPipxNativeEnvironment(group.pipx_native_environment,*parent)) ||
      (kind == ProcessState::PipxNativeImage::kMetadataQuery && group.pipx_native_environment.empty())) return nullptr;
  return parent;
}

// Reuse immutable ELF/Python ABI facts only. The native UV child never receives
// the existing direct CONTROL driver's proc, resolver or network permissions.
bool IsPipxNativeABIRead(const gvisor::common::ContextData& context, const ProcessState& state,
    const std::string& path, uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (!IsPipxNativeProfile(profile) || anchor == nullptr || IsWriteCapableOpen(flags)) return false;
  const auto* group = FindFilesystemGroup(context,state);
  if (group == nullptr) return false;
  const auto* root = PipxNativeCreator(context,*group,state,ProcessState::PipxNativeImage::kRoot);
  const auto* uv = PipxNativeCreator(context,*group,state,ProcessState::PipxNativeImage::kUV);
  if (flags == 557056 && group->pipx_native_candidate) {
    if (root != nullptr && path == kUVToolPath && anchor->mount_class == "helper" && anchor->mountpoint == "/haa-runtime") return true;
    if (uv != nullptr && (path == "/usr/local/bin/python" || path == "/usr/local/bin/python3.14") &&
        anchor->mountpoint == "/" && anchor->mount_class == "oci-root") return true;
  }
  // Actual pinned native UV startup reads this guest VM setting. This is
  // neither the CONTROL driver's proc grant nor permission for a query.
  if (root != nullptr && group->pipx_native_image == ProcessState::PipxNativeImage::kUV &&
      group->current_image_class == ProcessClass::kUV && flags == 557056 &&
      anchor->mountpoint == "/proc" && anchor->mount_class == "system" &&
      (path == "/proc/sys/vm/overcommit_memory" ||
       path == "/proc/"+std::to_string(context.thread_group_id())+"/cgroup" ||
       path == "/proc/"+std::to_string(context.thread_group_id())+"/mountinfo" ||
       path == "/proc/"+std::to_string(context.parent_thread_group_id())+"/comm")) return true;
  if (root != nullptr && group->pipx_native_image == ProcessState::PipxNativeImage::kUV &&
      group->current_image_class == ProcessClass::kUV && flags == 557056 &&
      anchor->mountpoint == "/sys/fs/cgroup/cpu" && anchor->mount_class == "system" &&
      (path == "/sys/fs/cgroup/cpu/cpu.cfs_quota_us" ||
       path == "/sys/fs/cgroup/cpu/cpu.cfs_period_us")) return true;
  if (root != nullptr && group->pipx_native_image == ProcessState::PipxNativeImage::kUV &&
      group->current_image_class == ProcessClass::kUV && flags == 557056 &&
      anchor->mountpoint == "/dev" && anchor->mount_class == "system" && path == "/dev/null") return true;
  if (root != nullptr && group->pipx_native_image == ProcessState::PipxNativeImage::kUV &&
      group->current_image_class == ProcessClass::kUV && flags == 557056 &&
      anchor->mountpoint == "/" && anchor->mount_class == "oci-root" &&
      (IsExactPinnedLibraryRead(path) || path == "/usr/local/bin/python3.14" || path == "/usr/lib/os-release")) return true;
  if (PipxNativeQueryParent(context,state,*group) == nullptr || group->pipx_native_image != ProcessState::PipxNativeImage::kUVQuery) return false;
  if (anchor->mountpoint == "/proc" && anchor->mount_class == "system") {
    return flags == 557056 && path == "/proc/"+std::to_string(context.thread_group_id())+"/maps";
  }
  return group->pipx_native_environment.empty() && anchor->mountpoint == "/" && anchor->mount_class == "oci-root" &&
      path == "/usr/local/lib/python3.14/site-packages" && (flags == 557056 || flags == 622592 || flags == 624640);
}


bool IsPinnedUVRuntimeRead(const gvisor::common::ContextData& context,
                          const ProcessState& state, const std::string& path,
                          uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfileUVResolver) != 0 ||
      anchor == nullptr || flags != 557056 ||
      !context.is_exec_session() || context.parent_thread_group_id() != 0) return false;
  const std::string self = "/proc/" + std::to_string(context.thread_group_id());
  const bool proc_read = anchor->mount_class == "system" && anchor->mountpoint == "/proc" &&
      (path == "/proc/sys/vm/overcommit_memory" || path == self + "/cgroup" || path == self + "/mountinfo");
  const bool quota_read = anchor->mount_class == "system" && anchor->mountpoint == "/sys/fs/cgroup/cpu" &&
      (path == "/sys/fs/cgroup/cpu/cpu.cfs_quota_us" || path == "/sys/fs/cgroup/cpu/cpu.cfs_period_us");
  const bool null_read = anchor->mount_class == "system" && anchor->mountpoint == "/dev" && path == "/dev/null";
  const bool interpreter_read = anchor->mount_class == "oci-root" && anchor->mountpoint == "/" &&
      path == "/usr/local/bin/python3.14";
  const bool distribution_read = anchor->mount_class == "oci-root" && anchor->mountpoint == "/" &&
      path == "/usr/lib/os-release";
  // Native tool discovery enumerates the fixed runtime PATH directories.
  // These are exact immutable image directories, not prefix/file or execution
  // grants. OpenResult supplies the resolved name (/bin and /sbin resolve into
  // /usr); descendants, artifacts and other profiles keep their own rules.
  const bool search_directory_read = anchor->mount_class == "oci-root" && anchor->mountpoint == "/" &&
      (path == "/usr/local/sbin" || path == "/usr/local/bin" ||
       path == "/usr/sbin" || path == "/usr/bin");
  const bool hosts_read = anchor->mount_class == "system" && anchor->mountpoint == "/etc/hosts" &&
      path == "/etc/hosts";
  if (!proc_read && !quota_read && !null_read && !interpreter_read && !distribution_read && !search_directory_read && !hosts_read) return false;
  const auto* group = FindFilesystemGroup(context, state);
  return group != nullptr && group->role == ProcessState::Role::kControl &&
      group->provenance == ProcessState::Provenance::kDirectExecRoot &&
      group->current_image_class == ProcessClass::kUV &&
      FilesystemProcessClass(context, state) == ProcessClass::kUV &&
      !group->root_eligible && group->root_consumed && group->trusted_control_network_active &&
      !group->demotion_pending && !group->launch_target_pending && !group->handoff_target_pending;
}

// Cargo's fixed ELF needs the glibc realtime compatibility ABI, and Rust
// initializes its stack guard from its own memory map. These are resolver-only
// reads by the kernel-classified Cargo driver and its exact info-only SDK queries;
// neither filenames, mutable comm nor diagnostics establish that authority.
bool IsPinnedCargoResolverRead(const gvisor::common::ContextData& context,
                              const ProcessState& state, const std::string& path,
                              uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfileCargoResolver) != 0 ||
      anchor == nullptr || !IsAtOrBelowMountpoint(path, anchor->mountpoint) ||
      IsWriteCapableOpen(flags)) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group != nullptr && group->current_image_class == ProcessClass::kCargoRustc &&
      FilesystemProcessClass(context, state) == ProcessClass::kCargoRustc &&
      group->cargo_rustc_query_active && HasExactCargoResolverCreator(context, *group, state)) {
    if (anchor->mount_class == "system" && anchor->mountpoint == "/sys/fs/cgroup/cpu") {
      return path == "/sys/fs/cgroup/cpu/cpu.cfs_quota_us" ||
          path == "/sys/fs/cgroup/cpu/cpu.cfs_period_us";
    }
    if (anchor->mount_class == "system" && anchor->mountpoint == "/proc") {
      return path == "/proc/sys/vm/overcommit_memory" ||
          path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps" ||
          path == "/proc/" + std::to_string(context.thread_group_id()) + "/statm" ||
          path == "/proc/" + std::to_string(context.thread_group_id()) + "/cgroup" ||
          path == "/proc/" + std::to_string(context.thread_group_id()) + "/mountinfo";
    }
    // Exact ABI inputs of the locked Rustc/driver/LLVM images. An artifact
    // manifest, runtime diagnostic or another SDK subtree cannot add inputs.
    return anchor->mount_class == "oci-root" && anchor->mountpoint == "/" &&
        (path == "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so" ||
         path == "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/libLLVM.so.23.1-rust-1.99.0-stable" ||
         path == "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/lib" ||
         path == "/usr/lib/x86_64-linux-gnu/librt.so.1" || path == "/lib/x86_64-linux-gnu/librt.so.1" ||
         path == "/usr/lib/x86_64-linux-gnu/libz.so.1.2.13" || path == "/lib/x86_64-linux-gnu/libz.so.1.2.13");
  }
  if (!context.is_exec_session() || context.parent_thread_group_id() != 0) return false;
  if (group == nullptr || group->role != ProcessState::Role::kControl ||
      group->provenance != ProcessState::Provenance::kDirectExecRoot ||
      group->current_image_class != ProcessClass::kCargo ||
      FilesystemProcessClass(context, state) != ProcessClass::kCargo ||
      group->root_eligible || !group->root_consumed || !group->trusted_control_network_active ||
      group->demotion_pending || group->launch_target_pending || group->handoff_target_pending) return false;
  if (anchor->mount_class == "oci-root" && anchor->mountpoint == "/") {
    return path == "/usr/lib/x86_64-linux-gnu/librt.so.1" ||
        path == "/lib/x86_64-linux-gnu/librt.so.1" ||
        path == "/etc/host.conf" ||
        path == "/etc/gai.conf" ||
        path == "/etc/nsswitch.conf" ||
        path == "/usr/share/zoneinfo/Etc/UTC" ||
        path == "/etc/ssl/certs/ca-certificates.crt";
  }
  if (anchor->mount_class == "system" && anchor->mountpoint == "/dev") {
    return (path == "/dev/urandom" && (flags == kOpenLargefile || flags == 557056)) ||
        (path == "/dev/null" && flags == 557056);
  }
  if (anchor->mount_class == "system" &&
      (anchor->mountpoint == "/etc/resolv.conf" || anchor->mountpoint == "/etc/hosts")) {
    return path == anchor->mountpoint;
  }
  return anchor->mount_class == "system" && anchor->mountpoint == "/proc" &&
      (path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps" ||
       (path == "/proc/sys/vm/overcommit_memory" && flags == 557056));
}

// ABI dependency closure of the locked Cargo executable, independently read
// from its ELF without starting the OCI image. Only sealed root data callers
// may use this set; it does not classify arbitrary ARTIFACT libraries.
bool IsExactCargoRuntimeABIRead(const std::string& path) {
  std::string canonical = path;
  if (HasPrefix(canonical, "/usr/lib/")) canonical = "/lib/" + canonical.substr(9);
  for (const char* name : {"libdl.so.2", "libgcc_s.so.1", "librt.so.1", "libpthread.so.0", "libm.so.6", "libc.so.6", "ld-linux-x86-64.so.2"}) {
    if (MatchesLibraryNameOrVersion(canonical, (std::string("/lib/x86_64-linux-gnu/") + name).c_str())) return true;
  }
  return false;
}

// Same locked Cargo runtime, now executing untrusted build inputs. The driver
// remains ARTIFACT and has no CONTROL/network authority. Only its immutable
// startup data and own runtime metadata have a bounded read classification.
bool IsPinnedCargoBuildDriverRead(const gvisor::common::ContextData& context,
                                 const ProcessState& state, const std::string& path,
                                 uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfileCargoBuild) != 0 || anchor == nullptr ||
      !IsAtOrBelowMountpoint(path, anchor->mountpoint) || IsWriteCapableOpen(flags) ||
      !context.is_exec_session() || context.parent_thread_group_id() != 0) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || !IsCargoBuildDriverGroup(*group)) return false;
  if (anchor->mount_class == "oci-root" && anchor->mountpoint == "/") {
    return path == "/etc/ld.so.cache" || IsExactCargoRuntimeABIRead(path) ||
        path == "/usr/share/zoneinfo/Etc/UTC" ||
        (path == "/etc/ssl/certs/ca-certificates.crt" && flags == kOpenLargefile);
  }
  if (anchor->mount_class != "system") return false;
  if (anchor->mountpoint == "/dev") {
    return (path == "/dev/urandom" && (flags == kOpenLargefile || flags == 557056)) ||
        (path == "/dev/null" && flags == 557056);
  }
  return anchor->mountpoint == "/proc" &&
      (path == "/proc/" + std::to_string(context.thread_group_id()) + "/maps" ||
       (path == "/proc/sys/vm/overcommit_memory" && flags == 557056));
}

// Canonical target standard libraries from the locked Rust SDK. Each compiler
// or linker consumer must separately prove its sealed image and exact lineage.
bool IsExactPinnedRustTargetLibraryInput(const std::string& path) {
  constexpr const char* sdk="/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/lib";
  return path==sdk || HasPrefix(path,(std::string(sdk)+"/").c_str());
}

// Locked Rustc consumes untrusted project/macro inputs as ARTIFACT. Its
// one-shot kernel clone must still have the exact Cargo build creator. Fixed
// SDK bytes are input data, never CONTROL, network or completion authority.
bool IsPinnedCargoBuildCompilerRead(const gvisor::common::ContextData& context,
                                   const ProcessState& state, const std::string& path,
                                   uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile,kProfileCargoBuild)!=0 || anchor==nullptr ||
      IsWriteCapableOpen(flags) || !IsAtOrBelowMountpoint(path,anchor->mountpoint)) return false;
  const auto* group=FindFilesystemGroup(context,state);
  if(group==nullptr || group->current_image_class!=ProcessClass::kCargoRustc ||
     !group->cargo_build_rustc_active || !HasExactCargoBuildDriverCreator(context,*group,state)) return false;
  if(anchor->mount_class=="oci-root" && anchor->mountpoint=="/") {
    return path=="/etc/ld.so.cache" || IsExactCargoRuntimeABIRead(path) ||
        path=="/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so" ||
        path=="/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/libLLVM.so.23.1-rust-1.99.0-stable" ||
        IsExactPinnedRustTargetLibraryInput(path) ||
        path=="/usr/lib/x86_64-linux-gnu/libz.so.1.2.13" || path=="/lib/x86_64-linux-gnu/libz.so.1.2.13";
  }
  if(anchor->mount_class!="system")return false;
  const std::string self="/proc/"+std::to_string(context.thread_group_id());
  if(anchor->mountpoint=="/proc")return path==self+"/maps" || path==self+"/statm" || path==self+"/cgroup" || path==self+"/mountinfo" || (path=="/proc/sys/vm/overcommit_memory" && flags==557056);
  return anchor->mountpoint=="/sys/fs/cgroup/cpu" && (path=="/sys/fs/cgroup/cpu/cpu.cfs_quota_us" || path=="/sys/fs/cgroup/cpu/cpu.cfs_period_us");
}

bool IsExactCargoLldLauncherInvocation(const gvisor::sentry::ExecveInfo& message) {
  constexpr const char* binary="/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/bin/gcc-ld/ld.lld";
  std::string invoked,argv0;
  return message.binary_path()==binary && NormalizeAbsolutePath(message.execfn(),&invoked) && invoked==binary &&
      message.argv_size()>0 && (message.argv(0)=="ld.lld" || (NormalizeAbsolutePath(message.argv(0),&argv0) && argv0==binary));
}

// A build program may inspect the fixed compiler version. This grants no
// compile/tool producer role and no CONTROL/network authority.
bool IsPinnedCargoBuildProgramQueryRead(const gvisor::common::ContextData& context,
                                      const ProcessState& state, const std::string& path,
                                      uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile,kProfileCargoBuild) != 0 || anchor == nullptr ||
      IsWriteCapableOpen(flags) || !IsAtOrBelowMountpoint(path,anchor->mountpoint)) return false;
  const auto* group = FindFilesystemGroup(context,state);
  if (group == nullptr || group->current_image_class != ProcessClass::kCargoRustc ||
      !group->cargo_build_program_query_active || !HasExactCargoBuildProgramCreator(context,*group,state)) return false;
  if (anchor->mount_class == "oci-root" && anchor->mountpoint == "/") {
    return path == "/etc/ld.so.cache" || IsExactCargoRuntimeABIRead(path) ||
        path == "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so" ||
        path == "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/libLLVM.so.23.1-rust-1.99.0-stable" ||
        path == "/usr/lib/x86_64-linux-gnu/libz.so.1.2.13" || path == "/lib/x86_64-linux-gnu/libz.so.1.2.13";
  }
  const std::string self = "/proc/"+std::to_string(context.thread_group_id());
  return anchor->mount_class == "system" && anchor->mountpoint == "/proc" && flags == 557056 &&
      (path == self+"/maps" || path == self+"/statm" || path == "/proc/sys/vm/overcommit_memory");
}

bool IsPinnedCargoLldLauncherRead(const gvisor::common::ContextData& context,
                                const ProcessState& state,const std::string& path,
                                uint64_t flags,const char* profile,const MountAnchor* anchor) {
  if(profile==nullptr || strcmp(profile,kProfileCargoBuild)!=0 || anchor==nullptr ||
     IsWriteCapableOpen(flags) || !IsAtOrBelowMountpoint(path,anchor->mountpoint))return false;
  const auto* group=FindFilesystemGroup(context,state);
  if(group==nullptr || group->current_image_class!=ProcessClass::kCargoLldLauncher || !group->cargo_lld_launcher_active ||
     !HasExactCargoBuildCollect2ChildCreator(context,*group,state))return false;
  if(anchor->mount_class=="oci-root" && anchor->mountpoint=="/") {
    if(path=="/etc/ld.so.cache")return true;
    std::string canonical=path;
    if(HasPrefix(canonical,"/usr/lib/"))canonical="/lib/"+canonical.substr(9);
    // Independently read ELF dependency closure of the locked launcher.
    for(const char* name:{"libgcc_s.so.1","libpthread.so.0","libc.so.6","ld-linux-x86-64.so.2"})
      if(MatchesLibraryNameOrVersion(canonical,(std::string("/lib/x86_64-linux-gnu/")+name).c_str()))return true;
    return false;
  }
  return anchor->mount_class=="system" && anchor->mountpoint=="/proc" &&
      (path=="/proc/"+std::to_string(context.thread_group_id())+"/maps" || (path=="/proc/sys/vm/overcommit_memory" && flags==557056));
}

// Fixed C ABI link-input roles shared by the locked Debian toolchains.
// Each consumer separately proves its kernel image, one-shot state and creator.
bool IsExactCABILinkInput(const std::string& path) {
  std::string canonical=path;
  if(HasPrefix(canonical,"/usr/lib/"))canonical="/lib/"+canonical.substr(9);
    for (const char* input : {
        "/lib/gcc/x86_64-linux-gnu/12/crtbegin.o", "/lib/gcc/x86_64-linux-gnu/12/crtbeginS.o",
        "/lib/gcc/x86_64-linux-gnu/12/crtbeginT.o", "/lib/gcc/x86_64-linux-gnu/12/crtend.o",
        "/lib/gcc/x86_64-linux-gnu/12/crtendS.o", "/lib/gcc/x86_64-linux-gnu/12/libgcc.a",
        "/lib/gcc/x86_64-linux-gnu/12/libgcc_eh.a", "/lib/gcc/x86_64-linux-gnu/12/libgcc_s.so",
        "/lib/x86_64-linux-gnu/Scrt1.o", "/lib/x86_64-linux-gnu/crt1.o", "/lib/x86_64-linux-gnu/rcrt1.o",
        "/lib/x86_64-linux-gnu/crti.o", "/lib/x86_64-linux-gnu/crtn.o", "/lib/x86_64-linux-gnu/libc.so",
        "/lib/x86_64-linux-gnu/libc_nonshared.a", "/lib/x86_64-linux-gnu/libpthread.a",
        "/lib/x86_64-linux-gnu/libpthread_nonshared.a", "/lib/x86_64-linux-gnu/libdl.a",
        "/lib/x86_64-linux-gnu/libgcc_s.so.1", "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2"}) {
      if (canonical == input) return true;
    }
  return false;
}

// Same-group one-shot locked native linker; LLVM's independently inspected
// transitive ABI is data only. Arbitrary SDK paths and plugins remain denied.
bool IsPinnedCargoRustLldRead(const gvisor::common::ContextData& context,
                            const ProcessState& state,const std::string& path,
                            uint64_t flags,const char* profile,const MountAnchor* anchor) {
  if(profile==nullptr || strcmp(profile,kProfileCargoBuild)!=0 || anchor==nullptr ||
     IsWriteCapableOpen(flags) || !IsAtOrBelowMountpoint(path,anchor->mountpoint))return false;
  const auto* group=FindFilesystemGroup(context,state);
  if(group==nullptr || group->current_image_class!=ProcessClass::kCargoRustLld || !group->cargo_rust_lld_active ||
     !HasExactCargoBuildCollect2ChildCreator(context,*group,state))return false;
  if(anchor->mount_class=="oci-root" && anchor->mountpoint=="/") {
    std::string canonical=path;
    if(HasPrefix(canonical,"/usr/lib/"))canonical="/lib/"+canonical.substr(9);
    // Fixed Rust standard-library C ABI input: merged glibc archive stubs
    // and the independently read libm script's two exact targets. This is
    // link data only; it changes no existing Python/runtime libutil rule.
    return path=="/etc/ld.so.cache" || IsExactCargoRuntimeABIRead(path) || IsExactCABILinkInput(path) || IsExactPinnedRustTargetLibraryInput(path) ||
        path=="/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/libLLVM.so.23.1-rust-1.99.0-stable" ||
        canonical=="/lib/x86_64-linux-gnu/libz.so.1.2.13" || canonical=="/lib/x86_64-linux-gnu/libutil.a" ||
        canonical=="/lib/x86_64-linux-gnu/librt.a" || canonical=="/lib/x86_64-linux-gnu/libm.so" ||
        canonical=="/lib/x86_64-linux-gnu/libmvec.so.1";
  }
  if(anchor->mount_class=="system" && anchor->mountpoint=="/dev")
    return path=="/dev/urandom" && flags==kOpenLargefile;
  return anchor->mount_class=="system" && anchor->mountpoint=="/proc" &&
      (path=="/proc/"+std::to_string(context.thread_group_id())+"/maps" || (path=="/proc/sys/vm/overcommit_memory" && flags==557056));
}

// Startup data only for the one-shot untrusted Cargo-owned build program.
// Its arbitrary filesystem/network/child activity is still observed as ARTIFACT.
bool IsPinnedCargoBuildProgramRead(const gvisor::common::ContextData& context,
                                 const ProcessState& state,const std::string& path,
                                 uint64_t flags,const char* profile,const MountAnchor* anchor) {
  if(profile==nullptr || strcmp(profile,kProfileCargoBuild)!=0 || anchor==nullptr ||
     IsWriteCapableOpen(flags) || !IsAtOrBelowMountpoint(path,anchor->mountpoint))return false;
  const auto* group=FindFilesystemGroup(context,state);
  if(group==nullptr || group->current_image_class!=ProcessClass::kCargoBuildProgram || !group->cargo_build_program_active ||
     !HasExactCargoBuildDriverCreator(context,*group,state))return false;
  if(anchor->mount_class=="system" && anchor->mountpoint=="/dev")
    return path=="/dev/null" && flags==557056;
  if(anchor->mount_class=="system" && anchor->mountpoint=="/proc")
    return path=="/proc/"+std::to_string(context.thread_group_id())+"/maps" && flags==557056;
  if(anchor->mount_class!="oci-root" || anchor->mountpoint!="/")return false;
  std::string canonical=path;
  if(HasPrefix(canonical,"/usr/lib/"))canonical="/lib/"+canonical.substr(9);
  return path=="/etc/ld.so.cache" || IsExactLibc6(path) ||
      MatchesLibraryNameOrVersion(canonical,"/lib/x86_64-linux-gnu/libgcc_s.so.1") ||
      canonical=="/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2";
}

bool IsPinnedGoResolverRead(const gvisor::common::ContextData& context,
                            const ProcessState& state, const std::string& path,
                            uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfileGoResolver) != 0 ||
      anchor == nullptr ||
      !IsAtOrBelowMountpoint(path, anchor->mountpoint) || IsWriteCapableOpen(flags)) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || group->role != ProcessState::Role::kControl ||
      group->root_eligible || !group->root_consumed || group->demotion_pending ||
      group->launch_target_pending || group->handoff_target_pending) return false;
  const auto image = FilesystemProcessClass(context, state);
  if (image == ProcessClass::kMkdir &&
      group->provenance == ProcessState::Provenance::kCloneChild &&
      !group->trusted_control_network_active && HasExactCloneCreator(*group, state) &&
      context.parent_thread_group_id() == group->clone_creator_group_id &&
      !IsWriteCapableOpen(flags) && anchor->mount_class == "system" &&
      anchor->mountpoint == "/proc" && path == "/proc/filesystems") {
    const auto& parent = state.groups.find(group->clone_creator_group_id)->second;
    return parent.role == ProcessState::Role::kControl &&
        parent.provenance == ProcessState::Provenance::kDirectExecRoot &&
        !parent.root_eligible && parent.root_consumed;
  }
  if (image != ProcessClass::kGo ||
      group->provenance != ProcessState::Provenance::kDirectExecRoot ||
      !group->trusted_control_network_active) return false;
  const std::string self = "/proc/" + std::to_string(context.thread_group_id());
  if (anchor->mount_class == "oci-root" && anchor->mountpoint == "/") {
    // X.509 reads the immutable Debian bundle and its certificate directory;
    // resolved symlinks reach the same image's packaged certificate store.
    return path == "/usr/local/go/go.env" || path == "/etc/nsswitch.conf" ||
        path == "/usr/share/zoneinfo/Etc/UTC" ||
        path == "/usr/local/go/src" || HasPrefix(path, "/usr/local/go/src/") ||
        path == "/etc/ssl/certs" || HasPrefix(path, "/etc/ssl/certs/") ||
        HasPrefix(path, "/usr/share/ca-certificates/");
  }
  if (anchor->mount_class != "system") return false;
  return ((path == "/etc/resolv.conf" || path == "/etc/hosts") && anchor->mountpoint == path) ||
      (anchor->mountpoint == "/proc" &&
          (path == self + "/cgroup" || path == self + "/mountinfo")) ||
      path == "/sys/fs/cgroup/cpu/cpu.cfs_quota_us" ||
      path == "/sys/fs/cgroup/cpu/cpu.cfs_period_us";
}

// Fixed HAA project copying uses the locked mkdir image in a CONTROL
// clone of the admitted shell. This metadata read grants no artifact, network,
// other proc subject, or runtime-tool authority and does not use mutable comm.
bool IsPinnedProjectConfigurationRead(
    const gvisor::common::ContextData& context, const ProcessState& state,
    const std::string& path, uint64_t flags, const char* profile,
    const MountAnchor* anchor) {
  if (profile == nullptr ||
      (strcmp(profile, kProfileGoBuild) != 0 && strcmp(profile, kProfileCargoResolver) != 0 &&
       strcmp(profile, kProfileCargoBuild) != 0) ||
      path != "/proc/filesystems" || IsWriteCapableOpen(flags) || anchor == nullptr ||
      anchor->mount_class != "system" || anchor->mountpoint != "/proc") return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || group->role != ProcessState::Role::kControl ||
      group->provenance != ProcessState::Provenance::kCloneChild ||
      (group->current_image_class != ProcessClass::kMkdir &&
       !(strcmp(profile, kProfileCargoResolver) == 0 && group->current_image_class == ProcessClass::kCargoTar)) ||
      group->root_eligible || !group->root_consumed ||
      group->trusted_control_network_active || group->demotion_pending ||
      group->launch_target_pending || group->handoff_target_pending ||
      !HasExactCloneCreator(*group, state) ||
      context.parent_thread_group_id() != group->clone_creator_group_id) return false;
  const auto& parent = state.groups.find(group->clone_creator_group_id)->second;
  return parent.role == ProcessState::Role::kControl &&
      parent.current_image_class == ProcessClass::kShell &&
      parent.provenance == ProcessState::Provenance::kDirectExecRoot &&
      !parent.root_eligible && parent.root_consumed && !parent.demotion_pending &&
      !parent.launch_target_pending && !parent.handoff_target_pending;
}

bool IsCanonicalBootstrapProfile(const char* profile) {
  return profile != nullptr &&
      (strcmp(profile,kProfilePythonExecution) == 0 || strcmp(profile, kProfileNodeExecution) == 0 || strcmp(profile, kProfileBunProjectResolver) == 0 || strcmp(profile, kProfileBunResolver) == 0 || strcmp(profile, kProfilePNPMResolver) == 0 || strcmp(profile, kProfileUVResolver) == 0 || strcmp(profile, kProfileNPM) == 0 || strcmp(profile, kProfilePyPI) == 0 ||
       strcmp(profile, kProfilePyTorchCPU) == 0 ||
       strcmp(profile, kProfilePyTorchCU126) == 0 ||
       strcmp(profile, kProfilePyTorchCU130) == 0 ||
       strcmp(profile, kProfilePyTorchCU132) == 0 ||
       strcmp(profile, kProfileGitHub) == 0 || strcmp(profile, kProfilePipxRuntime) == 0 || strcmp(profile, kProfilePipxNativeResolver) == 0 || strcmp(profile, kProfileTerraformProvider) == 0 || strcmp(profile, kProfileGoResolver) == 0 ||
       strcmp(profile, kProfileGoBuild) == 0 || strcmp(profile, kProfileCargoResolver) == 0 ||
       strcmp(profile, kProfileCargoBuild) == 0);
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
      HasExactCloneCreator(*group, state) &&
      direct_parent != state.groups.end() &&
      direct_parent->second.start_time_ns == group->clone_creator_group_start_time_ns &&
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
  // glibc resolver initialization reads these exact immutable OCI-root files
  // during the fixed GenerateLockfile Node transition. Keep it narrower than
  // the pre-existing common name-service identities: this is not a generic
  // NODE, CONTROL, clone-child, or OCI-root filesystem allowance.
  if (process_class == ProcessClass::kNode &&
      group->provenance == ProcessState::Provenance::kCloneChild &&
      group->command_phase == CommandPhase::kLockGeneration &&
      (path == "/etc/host.conf" || path == "/etc/gai.conf")) {
    return true;
  }
  const auto expected = state.expected_groups.find(context.thread_group_id());
  const bool exact_npm_version_node =
      strcmp(profile, kProfileNPM) == 0 &&
      process_class == ProcessClass::kNode &&
      context.is_exec_session() && context.parent_thread_group_id() == 0 &&
      group->role == ProcessState::Role::kControl &&
      group->provenance == ProcessState::Provenance::kDirectExecRoot &&
      !group->root_eligible && group->root_consumed &&
      group->trusted_control_network_active && !group->demotion_pending &&
      !group->launch_target_pending && !group->handoff_target_pending &&
      group->command_phase == CommandPhase::kNpmVersion &&
      !group->npm_version_node_transition_pending &&
      group->npm_version_node_transition_consumed &&
      expected != state.expected_groups.end() &&
      expected->second.start_time_ns == context.thread_group_start_time_ns() &&
      expected->second.process_class == ProcessClass::kNode;
  if (process_class != ProcessClass::kNode ||
      (group->provenance != ProcessState::Provenance::kCloneChild &&
       !exact_npm_version_node)) return false;
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

// The linker and assembler use the runtime null device as an output sink.
// A matched /dev mount, exact owned tool and observed flag shape do not
// grant general device or filesystem writes.
bool IsPinnedGoBuildNullDeviceOpen(
    const gvisor::common::ContextData& context, const ProcessState& state,
    const std::string& path, uint64_t flags, const char* profile,
    const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfileGoBuild) != 0 || anchor == nullptr ||
      anchor->mount_class != "system" || anchor->mountpoint != "/dev" || path != "/dev/null" ||
      (flags & ~kOpenLargefile) != (kOpenReadWrite | kOpenCreate | kOpenTruncate)) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || !group->go_build_tool_active) return false;
  return (group->current_image_class == ProcessClass::kGoBuildNativeLinker &&
          HasExactGoBuildNativeLinkerCreator(context, *group, state)) ||
      (group->current_image_class == ProcessClass::kGoBuildAssembler &&
       HasExactGoBuildGccChildCreator(context, *group, state));
}

// Both fixed build runtimes contain the same GCC loader ABI. Each caller
// keeps its own exact SDK creator contract; this grants no generic library,
// compiler descendant, CONTROL, network or mutable-name authority.
bool IsPinnedProjectBuildGccRead(const gvisor::common::ContextData& context,
                                const ProcessState& state, const std::string& path,
                                uint64_t flags, const char* profile, const MountAnchor* anchor) {
  if(profile==nullptr || anchor==nullptr || anchor->mount_class!="oci-root" ||
     anchor->mountpoint!="/" || IsWriteCapableOpen(flags))return false;
  const auto* group=FindFilesystemGroup(context,state);
  if(group==nullptr || group->current_image_class!=ProcessClass::kGoBuildGcc || !group->go_build_tool_active)return false;
  const bool owned=(strcmp(profile,kProfileGoBuild)==0 && HasExactGoBuildGccCreator(context,*group,state)) ||
      (strcmp(profile,kProfileCargoBuild)==0 && HasExactCargoBuildCompilerCreator(context,*group,state));
  return owned && (path=="/etc/ld.so.cache" || IsExactLibc6(path));
}

bool IsPinnedProjectBuildCollect2Read(const gvisor::common::ContextData& context,
                                    const ProcessState& state,const std::string& path,
                                    uint64_t flags,const char* profile,const MountAnchor* anchor) {
  if(profile==nullptr || anchor==nullptr || anchor->mount_class!="oci-root" ||
     anchor->mountpoint!="/" || IsWriteCapableOpen(flags))return false;
  const auto* group=FindFilesystemGroup(context,state);
  if(group==nullptr || group->current_image_class!=ProcessClass::kGoBuildCollect2 || !group->go_build_tool_active)return false;
  const bool owned=(strcmp(profile,kProfileGoBuild)==0 && HasExactGoBuildGccChildCreator(context,*group,state)) ||
      (strcmp(profile,kProfileCargoBuild)==0 && HasExactCargoBuildGccChildCreator(context,*group,state));
  std::string canonical=path;if(HasPrefix(canonical,"/usr/lib/"))canonical="/lib/"+canonical.substr(9);
  return owned && (path=="/etc/ld.so.cache" || IsExactLibc6(path) || canonical=="/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2");
}

// The locked build driver and its exact SDK tools read immutable SDK inputs,
// fixed runtime data and their own cgroup metadata. These are not completion or
// resource authority. Keep this
// distinct from the CONTROL/network-enabled public module resolver.
bool IsPinnedGoBuildRuntimeRead(
    const gvisor::common::ContextData& context, const ProcessState& state,
    const std::string& path, uint64_t flags, const char* profile,
    const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfileGoBuild) != 0 ||
      IsWriteCapableOpen(flags) || anchor == nullptr) return false;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || group->role != ProcessState::Role::kArtifact ||
      group->root_eligible || !group->root_consumed ||
      group->trusted_control_network_active || group->demotion_pending ||
      group->launch_target_pending || group->handoff_target_pending) return false;
  const bool driver = IsGoBuildDriverGroup(*group);
  const bool tool = (group->current_image_class == ProcessClass::kGoBuildTool ||
      group->current_image_class == ProcessClass::kGoBuildCgo ||
      group->current_image_class == ProcessClass::kGoBuildLink) &&
      group->go_build_tool_active && HasExactGoBuildDriverCreator(context, *group, state);
  const bool native_gcc = group->current_image_class == ProcessClass::kGoBuildGcc &&
      group->go_build_tool_active && HasExactGoBuildGccCreator(context, *group, state);
  if (native_gcc) {
    // The compiler stays ARTIFACT. Its fixed loader data is not an SDK,
    // arbitrary image-library or system-metadata grant.
    return IsPinnedProjectBuildGccRead(context,state,path,flags,profile,anchor);
  }
  const bool native_linker = group->current_image_class == ProcessClass::kGoBuildNativeLinker &&
      group->go_build_tool_active && HasExactGoBuildNativeLinkerCreator(context, *group, state);
  if (native_linker) {
    if (anchor->mount_class != "oci-root" || anchor->mountpoint != "/") return false;
    if (path == "/etc/ld.so.cache" || path == "/usr/lib/gcc/x86_64-linux-gnu/12/liblto_plugin.so") return true;
    std::string library_path = path;
    if (HasPrefix(library_path, "/usr/lib/")) library_path = "/lib/" + library_path.substr(9);
    if (IsExactCABILinkInput(path)) return true;
    for (const char* name : {"libbfd-2.40-system.so", "libctf.so.0", "libjansson.so.4", "libz.so.1",
         "libzstd.so.1", "libsframe.so.0", "libc.so.6", "libresolv.so.2"}) {
      if (MatchesLibraryNameOrVersion(library_path, (std::string("/lib/x86_64-linux-gnu/") + name).c_str())) return true;
    }
    return false;
  }
  const bool native_collect2 = group->current_image_class == ProcessClass::kGoBuildCollect2 &&
      group->go_build_tool_active && HasExactGoBuildGccChildCreator(context, *group, state);
  if (native_collect2) {
    return IsPinnedProjectBuildCollect2Read(context,state,path,flags,profile,anchor);
  }
  const bool native_assembler = group->current_image_class == ProcessClass::kGoBuildAssembler &&
      group->go_build_tool_active && HasExactGoBuildGccChildCreator(context, *group, state);
  if (native_assembler) {
    if (anchor->mount_class != "oci-root" || anchor->mountpoint != "/") return false;
    if (path == "/etc/ld.so.cache") return true;
    std::string library_path = path;
    if (HasPrefix(library_path, "/usr/lib/")) library_path = "/lib/" + library_path.substr(9);
    for (const char* name : {"libbfd-2.40-system.so", "libz.so.1", "libzstd.so.1", "libsframe.so.0", "libc.so.6"}) {
      if (MatchesLibraryNameOrVersion(library_path, (std::string("/lib/x86_64-linux-gnu/") + name).c_str())) return true;
    }
    return false;
  }
  const bool native_cc1 = group->current_image_class == ProcessClass::kGoBuildCc1 &&
      group->go_build_tool_active && HasExactGoBuildGccChildCreator(context, *group, state);
  if (native_cc1) {
    if (anchor->mount_class != "oci-root" || anchor->mountpoint != "/") return false;
    if (path == "/etc/ld.so.cache" || path == "/usr/include" || HasPrefix(path, "/usr/include/") ||
        path == "/usr/lib/gcc/x86_64-linux-gnu/12/include" ||
        HasPrefix(path, "/usr/lib/gcc/x86_64-linux-gnu/12/include/")) return true;
    // CC1 preprocesses/compiles the fixed SDK's Cgo runtime inputs. This
    // immutable C ABI subtree is distinct from arbitrary Go SDK source.
    if (path == "/usr/local/go/src/runtime/cgo" ||
        HasPrefix(path, "/usr/local/go/src/runtime/cgo/")) return true;
    std::string library_path = path;
    if (HasPrefix(library_path, "/usr/lib/")) library_path = "/lib/" + library_path.substr(9);
    // Fixed ABI roles from the locked cc1 ELF. Neither artifact output nor a
    // mutable metadata file can add a loader permission at runtime.
    for (const char* name : {"libisl.so.23", "libmpc.so.3", "libmpfr.so.6", "libgmp.so.10",
         "libz.so.1", "libzstd.so.1", "libm.so.6", "libc.so.6", "ld-linux-x86-64.so.2"}) {
      if (MatchesLibraryNameOrVersion(library_path, (std::string("/lib/x86_64-linux-gnu/") + name).c_str())) return true;
    }
    return false;
  }
  if (!driver && !tool) return false;
  if (anchor->mount_class == "oci-root" && anchor->mountpoint == "/") {
    return path == "/usr/local/go/go.env" || path == "/usr/local/go/src" ||
        HasPrefix(path, "/usr/local/go/src/") || path == "/usr/local/go/pkg/include" ||
        HasPrefix(path, "/usr/local/go/pkg/include/") || path == "/usr/share/zoneinfo/Etc/UTC";
  }
  if (anchor->mount_class != "system") return false;
  if (anchor->mountpoint == "/dev") return path == "/dev/null";
  const std::string self = "/proc/" + std::to_string(context.thread_group_id());
  if (anchor->mountpoint == "/proc") {
    return path == self + "/cgroup" || path == self + "/mountinfo";
  }
  return anchor->mountpoint == "/sys/fs/cgroup/cpu" &&
      (path == "/sys/fs/cgroup/cpu/cpu.cfs_quota_us" ||
       path == "/sys/fs/cgroup/cpu/cpu.cfs_period_us");
}

// This new build profile cannot obtain helper reads from comm. The kernel
// helper/demoter image and the one-shot, exact direct handoff must agree. Both
// stages remain ARTIFACT, and the allowance disappears at the exact project driver exec.
bool IsPinnedProjectBuildHandoffRead(
    const gvisor::common::ContextData& context, const ProcessState& state,
    const std::string& path, uint64_t flags, const char* profile,
    const MountAnchor* anchor) {
  if (profile == nullptr ||
      (strcmp(profile, kProfileGoBuild) != 0 && strcmp(profile, kProfileCargoBuild) != 0) ||
      IsWriteCapableOpen(flags) || anchor == nullptr) return false;
  const auto target = strcmp(profile, kProfileCargoBuild) == 0 ? ProcessClass::kCargo : ProcessClass::kGo;
  const auto* group = FindFilesystemGroup(context, state);
  if (group == nullptr || group->role != ProcessState::Role::kArtifact ||
      group->provenance != ProcessState::Provenance::kDirectExecRoot ||
      group->root_eligible || !group->root_consumed ||
      group->trusted_control_network_active || !group->handoff_target_pending ||
      group->handoff_target_class != target) return false;
  const bool helper = group->demotion_pending &&
      group->current_image_class == ProcessClass::kProjectBuildBoundary;
  const bool demoter = !group->demotion_pending &&
      group->current_image_class == ProcessClass::kProjectBuildSetpriv;
  if (!helper && !demoter) return false;
  if (helper && anchor->mount_class == "helper" &&
      anchor->mountpoint == "/haa-runtime" && path == "/haa-runtime/haa-boundary") return true;
  if (anchor->mount_class == "oci-root" && anchor->mountpoint == "/") {
    if (path == "/etc/ld.so.cache" || IsExactLibc6(path)) return true;
    return demoter && (IsExactLibcapNg0(path) || path == "/etc/nsswitch.conf" ||
        path == "/etc/passwd" || path == "/etc/group");
  }
  return demoter && anchor->mount_class == "system" && anchor->mountpoint == "/proc" &&
      (path == "/proc/sys/kernel/cap_last_cap" ||
       path == "/proc/" + std::to_string(context.thread_group_id()) + "/status");
}

bool IsExactHAAELFHandoffDemotionRead(
    const gvisor::common::ContextData& context, const ProcessState& state,
    const std::string& path, uint64_t flags, const char* profile) {
  if (profile == nullptr || (strcmp(profile, kProfileGitHub) != 0 && strcmp(profile, kProfileTerraformProvider) != 0 && !IsPythonRuntimeProfile(profile)) ||
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

// Docker supplies this file as a distinct runtime mount, not as pinned OCI
// content. Keep the mount fact at the authorization boundary: a pathname-only
// /etc/hosts rule would also accept a replaced rootfs identity.
bool IsExactDockerHostsMount(const MountAnchor* anchor) {
  return anchor != nullptr && anchor->mount_class == "system" &&
      anchor->mountpoint == "/etc/hosts";
}

bool IsExactDockerEtcHostsLockGenerationRead(
    const gvisor::common::ContextData& context, const ProcessState& state,
    const std::string& path, uint64_t flags, const char* profile,
    const MountAnchor* anchor) {
  if (profile == nullptr || strcmp(profile, kProfileNPM) != 0 ||
      path != "/etc/hosts" || IsWriteCapableOpen(flags) ||
      !IsExactDockerHostsMount(anchor)) {
    return false;
  }
  const auto* group = FindFilesystemGroup(context, state);
  const auto expected = state.expected_groups.find(context.thread_group_id());
  const auto creator = group == nullptr ? state.groups.end() :
      state.groups.find(group->clone_creator_group_id);
  return group != nullptr && expected != state.expected_groups.end() &&
      expected->second.start_time_ns == context.thread_group_start_time_ns() &&
      expected->second.process_class == ProcessClass::kNode &&
      group->role == ProcessState::Role::kControl &&
      group->provenance == ProcessState::Provenance::kCloneChild &&
      group->command_phase == CommandPhase::kLockGeneration &&
      !group->root_eligible && group->root_consumed &&
      // The clone must not inherit the direct-root network exception.
      !group->trusted_control_network_active && !group->demotion_pending &&
      !group->launch_target_pending && !group->handoff_target_pending &&
      group->clone_creator_group_id > 0 &&
      HasExactCloneCreator(*group, state) &&
      context.parent_thread_group_id() == group->clone_creator_group_id &&
      creator != state.groups.end() &&
      creator->second.start_time_ns == group->clone_creator_group_start_time_ns &&
      creator->second.role == ProcessState::Role::kControl;
}

FilesystemClass ClassifyFilesystemOpen(const gvisor::syscall::Open& message,
                                       const ProcessState& state, const char* profile,
                                       const MountAnchor* anchor = nullptr) {
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
  // Interpreter discovery is never installed-code admission. This precedes
  // the general workspace rule, also after clone/reexec; startup/package files,
  // other environments and writes remain actionable ARTIFACT accesses.
  // A fresh cached query's package environment shares TMP with UV's private
  // query modules. Only the exact parsed module root and inert environment
  // directories/config remain readable; no package/startup body or TMP write.
  const bool exported_query = IsAtOrBelowMountpoint(tracked->uv_python_query_helper_root, kUVExportedCacheRoot);
  std::string cached_query_path;
  if (profile != nullptr && strcmp(profile, kProfileUVResolver) == 0 &&
      tracked->uv_python_query_restricted &&
      (tracked->uv_python_query_environment.compare(0, 5, "/tmp/") == 0 || exported_query) &&
      ((anchor != nullptr && IsAtOrBelowMountpoint(anchor->mountpoint, "/tmp")) ||
       IsAtOrBelowMountpoint(path, "/tmp") ||
       (NormalizeAbsolutePath(path, &cached_query_path) && IsAtOrBelowMountpoint(cached_query_path, "/tmp")) ||
       (exported_query && ((anchor != nullptr && IsAtOrBelowMountpoint(anchor->mountpoint, "/haa-site")) ||
        IsAtOrBelowMountpoint(path, "/haa-site") ||
        (NormalizeAbsolutePath(path, &cached_query_path) && IsAtOrBelowMountpoint(cached_query_path, "/haa-site")))))) {
    const std::string& env = tracked->uv_python_query_environment;
    const std::string& helper = tracked->uv_python_query_helper_root;
    const bool canonical = NormalizeAbsolutePath(path, &cached_query_path) && path == cached_query_path;
    const bool module = !helper.empty() && IsAtOrBelowMountpoint(path, helper);
    const bool directory = !env.empty() && (path == env || path == env + "/lib" ||
        path == env + "/lib/python3.14" || path == env + "/lib/python3.14/site-packages");
    const bool config = !env.empty() && path == env + "/pyvenv.cfg";
    const bool readonly_flags = message.flags() == 557056 || message.flags() == 622592;
    // Pinned CPython's opendir uses O_NONBLOCK in addition to directory/RO.
    // Bind only the four directories observed in the exact native query ABI.
    const bool scanned_directory = path == helper || path == helper + "/python" ||
        path == helper + "/python/packaging" || path == env + "/lib/python3.14/site-packages";
    const bool allowed = tracked->uv_python_query_active &&
        tracked->current_image_class == ProcessClass::kPython &&
        HasExactUVQueryCreator(message.context_data(), *tracked, state) && canonical &&
        anchor != nullptr && anchor->mountpoint == (exported_query ? "/haa-site" : "/tmp") && anchor->mount_class == "workspace" &&
        !IsWriteCapableOpen(message.flags()) &&
        (((module || directory) && readonly_flags) ||
         (scanned_directory && message.flags() == 624640) ||
         (config && (message.flags() == 557056 || message.flags() == kOpenLargefile)));
    return allowed ? FilesystemClass::kHelperOnly : FilesystemClass::kOutside;
  }
  std::string query_path;
  if (profile != nullptr && strcmp(profile, kProfileUVResolver) == 0 &&
      tracked->uv_python_query_restricted &&
      ((anchor != nullptr && IsAtOrBelowMountpoint(anchor->mountpoint, "/haa-site")) ||
       IsAtOrBelowMountpoint(path, "/haa-site") ||
       (NormalizeAbsolutePath(path, &query_path) && IsAtOrBelowMountpoint(query_path, "/haa-site")))) {
    const std::string& env = tracked->uv_python_query_environment;
    const bool exact_directory = !env.empty() && (path == env || path == env + "/lib" ||
        path == env + "/lib/python3.14" || path == env + "/lib/python3.14/site-packages");
    const bool exact_config = !env.empty() && path == env + "/pyvenv.cfg";
    const bool allowed = tracked->uv_python_query_active &&
        HasExactUVQueryCreator(message.context_data(), *tracked, state) &&
        anchor != nullptr && anchor->mount_class == "workspace" && anchor->mountpoint == "/haa-site" &&
        !IsWriteCapableOpen(message.flags()) &&
        ((exact_config && message.flags() == 557056) ||
         (exact_directory && (message.flags() == 557056 || message.flags() == 622592)));
    return allowed ? FilesystemClass::kHelperOnly : FilesystemClass::kOutside;
  }
  // A query never inherits general installed/output or DATA-input access.
  // Retain this restriction after clone/reexec, even when active admission ends.
  if (IsPipxNativeProfile(profile) && tracked->pipx_native_query_restricted) {
    if (PipxNativeQueryParent(message.context_data(),state,*tracked) == nullptr || IsWriteCapableOpen(message.flags())) return FilesystemClass::kOutside;
    std::string normalized;
    const bool site = (anchor != nullptr && IsAtOrBelowMountpoint(anchor->mountpoint,"/haa-site")) ||
        IsAtOrBelowMountpoint(path,"/haa-site") || (NormalizeAbsolutePath(path,&normalized) && IsAtOrBelowMountpoint(normalized,"/haa-site"));
    const bool data = (anchor != nullptr && (IsAtOrBelowMountpoint(anchor->mountpoint,"/haa-pipx") || IsAtOrBelowMountpoint(anchor->mountpoint,"/haa-wheels"))) ||
        IsAtOrBelowMountpoint(path,"/haa-pipx") || IsAtOrBelowMountpoint(path,"/haa-wheels") ||
        (NormalizeAbsolutePath(path,&normalized) && (IsAtOrBelowMountpoint(normalized,"/haa-pipx") || IsAtOrBelowMountpoint(normalized,"/haa-wheels")));
    if (data) return FilesystemClass::kOutside;
    if (site) {
      const std::string& env = tracked->pipx_native_environment;
      const bool directory = !env.empty() && (path == env || path == env+"/lib" || path == env+"/lib/python3.14" || path == env+"/lib/python3.14/site-packages");
      const bool config = !env.empty() && path == env+"/pyvenv.cfg";
      const bool allowed = PipxNativeQueryParent(message.context_data(),state,*tracked) != nullptr && anchor != nullptr &&
          anchor->mountpoint == "/haa-site" && anchor->mount_class == "workspace" && !IsWriteCapableOpen(message.flags()) &&
          ((config && (message.flags() == 557056 || message.flags() == kOpenLargefile)) ||
           (directory && (message.flags() == 557056 || message.flags() == 622592 ||
            (message.flags() == 624640 && path == env+"/lib/python3.14/site-packages"))));
      return allowed ? FilesystemClass::kHelperOnly : FilesystemClass::kOutside;
    }
  }
  if (IsPipxNativeProfile(profile)) {
    std::string normalized;
    const bool normalized_ok = NormalizeAbsolutePath(path,&normalized);
    const bool data = (anchor != nullptr && (IsAtOrBelowMountpoint(anchor->mountpoint,"/haa-pipx") || IsAtOrBelowMountpoint(anchor->mountpoint,"/haa-wheels"))) ||
        IsAtOrBelowMountpoint(path,"/haa-pipx") || IsAtOrBelowMountpoint(path,"/haa-wheels") ||
        (normalized_ok && (IsAtOrBelowMountpoint(normalized,"/haa-pipx") || IsAtOrBelowMountpoint(normalized,"/haa-wheels")));
    if (data) {
      const bool root = IsPipxNativeActiveGroup(*tracked) && tracked->provenance == ProcessState::Provenance::kDirectExecRoot &&
          tracked->current_image_class == ProcessClass::kPython && tracked->pipx_native_image == ProcessState::PipxNativeImage::kRoot &&
          message.context_data().is_exec_session() && message.context_data().parent_thread_group_id() == 0 &&
          state.expected_groups.find(message.context_data().thread_group_id()) == state.expected_groups.end();
      const bool uv = tracked->pipx_native_image == ProcessState::PipxNativeImage::kUV && tracked->current_image_class == ProcessClass::kUV &&
          PipxNativeCreator(message.context_data(),*tracked,state,ProcessState::PipxNativeImage::kRoot) != nullptr;
      const bool zip = (path == "/haa-pipx" || path == "/haa-pipx/pipx-runtime.zip") && anchor != nullptr && anchor->mountpoint == "/haa-pipx";
      const bool wheel = (path == "/haa-wheels" || (HasPrefix(path,"/haa-wheels/") && path.find('/',12) == std::string::npos &&
          path.size() > 4 && path.compare(path.size()-4,4,".whl") == 0)) && anchor != nullptr && anchor->mountpoint == "/haa-wheels";
      const bool allowed = (root || uv) && normalized_ok && path == normalized && !IsWriteCapableOpen(message.flags()) &&
          anchor != nullptr && anchor->mount_class == "workspace" && (zip || wheel);
      return allowed ? FilesystemClass::kWorkspace : FilesystemClass::kOutside;
    }
    if (IsPipxNativeABIRead(message.context_data(),state,path,message.flags(),profile,anchor)) return FilesystemClass::kHelperOnly;
  }
  if (IsPinnedProjectBuildHandoffRead(message.context_data(), state, path,
                                  message.flags(), profile, anchor)) {
    return FilesystemClass::kHelperOnly;
  }
  if (IsExactHAAELFHandoffDemotionRead(message.context_data(), state, path,
                                       message.flags(), profile)) {
    return FilesystemClass::kHelperOnly;
  }
  const bool exact_self_status = tracked->role == ProcessState::Role::kControl &&
      message.context_data().process_name() == "setpriv" && !IsWriteCapableOpen(message.flags()) &&
      path == "/proc/" + std::to_string(message.context_data().thread_group_id()) + "/status";
  if (exact_self_status) return FilesystemClass::kHelperOnly;
  const bool exact_handoff_validation = (profile == nullptr || (strcmp(profile, kProfileGoBuild) != 0 && strcmp(profile, kProfileCargoBuild) != 0)) &&
      (profile == nullptr || (strcmp(profile,kProfileNodeExecution) != 0 && strcmp(profile,kProfilePythonExecution) != 0) || tracked->handoff_target_pending) &&
      tracked->role == ProcessState::Role::kArtifact &&
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
  const bool exact_artifact_shell_loader = (profile == nullptr || (strcmp(profile, kProfileGoBuild) != 0 && strcmp(profile, kProfileCargoBuild) != 0)) &&
      (profile == nullptr || (strcmp(profile,kProfileNodeExecution) != 0 && strcmp(profile,kProfilePythonExecution) != 0)) &&
      tracked->role == ProcessState::Role::kArtifact &&
      tracked->provenance == ProcessState::Provenance::kCloneChild &&
      !tracked->root_eligible && tracked->root_consumed &&
      !tracked->trusted_control_network_active &&
      message.context_data().process_name() == "sh" &&
      !IsWriteCapableOpen(message.flags()) &&
      (path == "/etc/ld.so.cache" || IsExactLibc6(path));
  if (exact_artifact_shell_loader) return FilesystemClass::kHelperOnly;
  if (IsPythonProfile(profile) && tracked->role == ProcessState::Role::kArtifact &&
      tracked->runtime_cache_query && anchor != nullptr &&
      anchor->mountpoint == "/" && anchor->mount_class == "oci-root" &&
      !IsWriteCapableOpen(message.flags()) && path == "/etc/ld.so.cache") {
    return FilesystemClass::kRuntimeRoot;
  }
  if (IsCargoBuildConfigurationWrite(message.context_data(), state, path, message.flags(), profile)) return FilesystemClass::kOutside;
  if (IsPipxRuntimeProfile(profile) &&
      (path == "/haa-pipx" || HasPrefix(path, "/haa-pipx/")) &&
      (!IsWorkspacePath(path, profile) || IsWriteCapableOpen(message.flags()))) return FilesystemClass::kOutside;
  if (profile != nullptr && strcmp(profile,kProfilePythonExecution)==0 && (path=="/haa-site" || HasPrefix(path,"/haa-site/"))) {
    return !IsWriteCapableOpen(message.flags()) && anchor != nullptr && anchor->mountpoint=="/haa-site" && anchor->mount_class=="workspace" &&
        IsActivePythonExecutionGroup(message.context_data(),tracked) ? FilesystemClass::kWorkspace : FilesystemClass::kOutside;
  }
  if (IsWorkspacePath(path, profile)) return FilesystemClass::kWorkspace;
  if (IsClearlyOutsideWorkspace(path) ||
      (IsWriteCapableOpen(message.flags()) && path != "/dev/null" &&
       !IsExactBootstrapHelperWrite(message.context_data(), state, path, message.flags()))) {
    return FilesystemClass::kOutside;
  }
  if (IsGitHubArtifactRuntimeRead(message.context_data(), state, path,
                                  message.flags(), profile, anchor) ||
      IsPinnedPNPMNodeQueryRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedNodeExecutionRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedPythonExecutionRead(message.context_data(), state, path, message.flags(), profile, anchor)) {
    return FilesystemClass::kRuntimeRoot;
  }
  if (IsExactBootstrapHelperWrite(message.context_data(), state, path, message.flags()) ||
      IsPinnedBunArtifactRuntimeRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedBunRuntimeRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedPNPMRuntimeRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedUVRuntimeRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedUVPythonQueryRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedCargoResolverRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedCargoBuildDriverRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedCargoBuildCompilerRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedCargoBuildProgramQueryRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedCargoLldLauncherRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedCargoRustLldRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedCargoBuildProgramRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedProjectBuildGccRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedProjectBuildCollect2Read(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedGoResolverRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedProjectConfigurationRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedGoBuildRuntimeRead(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsPinnedGoBuildNullDeviceOpen(message.context_data(), state, path, message.flags(), profile, anchor) ||
      IsExactDockerEtcHostsLockGenerationRead(message.context_data(), state, path,
                                              message.flags(), profile, anchor) ||
      IsPinnedRuntimeRootRead(message.context_data(), state, path,
                              message.flags(), profile) ||
      IsPinnedNpmRuntimeRead(message.context_data(), state, path, message.flags(), profile)) {
    return profile != nullptr && IsPythonRuntimeProfile(profile) &&
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

bool SendControlBody(int control, const std::string& body) {
  return control >= 0 && body.size() <= kMaxControlRecordBytes &&
      send(control, body.data(), body.size(), 0) == static_cast<ssize_t>(body.size());
}

bool SendAdmissionAck(int control,
                      const std::string& operation, const std::string& container_id,
                      const std::string& generation, const std::string& mode,
                      const std::string& nonce, const char* status) {
  if (status == nullptr || !ValidContainerID(container_id) || !ValidSessionGeneration(generation) ||
      !ValidAdmissionNonce(nonce)) return false;
  return SendControlBody(control, "{\"op\":\"ack\",\"ack_op\":\"" + operation +
      "\",\"status\":\"" + status + "\",\"container_id\":\"" + container_id +
      "\",\"session_generation\":\"" + generation + "\",\"mode\":\"" + mode +
      "\",\"nonce\":\"" + nonce + "\"}");
}

bool SendProfileAck(int control,
                    const std::string& container_id, const std::string& profile,
                    const std::string& topology, const std::string& generation, const char* status) {
  if (status == nullptr || !ValidContainerID(container_id) || !ValidSessionGeneration(generation)) return false;
  return SendControlBody(control, std::string("{\"op\":\"ack\",\"ack_op\":\"profile\",\"status\":\"") +
      status + "\",\"container_id\":\"" + container_id + "\",\"profile\":\"" + profile +
      "\",\"expected_topology\":\"" + topology + "\",\"session_generation\":\"" + generation + "\"}");
}

bool SendInvalidateAck(int control,
                       const std::string& container_id, const std::string& generation, const char* status) {
  if (status == nullptr || !ValidContainerID(container_id) || !ValidSessionGeneration(generation)) return false;
  return SendControlBody(control, "{\"op\":\"ack\",\"ack_op\":\"invalidate\",\"status\":\"" +
      std::string(status) + "\",\"container_id\":\"" + container_id + "\",\"session_generation\":\"" + generation + "\"}");
}

bool ParseStrictStringObject(const char* payload, size_t size, std::map<std::string, std::string>* fields) {
  if (payload == nullptr || fields == nullptr || size == 0 || size > kMaxControlRecordBytes) return false;
  size_t position = 0;
  auto whitespace = [&]() { while (position < size && (payload[position] == ' ' || payload[position] == '\n' || payload[position] == '\r' || payload[position] == '\t')) ++position; };
  auto string = [&](std::string* out) {
    if (out == nullptr || position >= size || payload[position++] != '"') return false;
    out->clear();
    while (position < size && payload[position] != '"') {
      const unsigned char character = static_cast<unsigned char>(payload[position++]);
      if (character < 0x20 || character == '\\') return false;
      out->push_back(static_cast<char>(character));
    }
    return position < size && payload[position++] == '"';
  };
  whitespace(); if (position >= size || payload[position++] != '{') return false; whitespace();
  if (position < size && payload[position] == '}') { ++position; whitespace(); return position == size; }
  for (;;) {
    std::string key, value; if (!string(&key)) return false; whitespace();
    if (position >= size || payload[position++] != ':') return false;
    whitespace();
    if (!string(&value) || !fields->emplace(key, value).second) return false;
    whitespace();
    if (position >= size) return false;
    if (payload[position] == '}') { ++position; whitespace(); return position == size; }
    if (payload[position++] != ',') return false;
    whitespace();
  }
}

bool ExactControlFields(const std::map<std::string, std::string>& fields,
                        std::initializer_list<const char*> required) {
  if (fields.size() != required.size()) return false;
  for (const char* key : required) if (fields.find(key) == fields.end()) return false;
  return true;
}

bool ParseControlRecord(const char* payload, size_t size, ControlPeer* peer,
                        std::map<std::string, ProfileRegistration>* profiles) {
  if (peer == nullptr || peer->fd < 0 || profiles == nullptr) return false;
  std::map<std::string, std::string> fields;
  if (!ParseStrictStringObject(payload, size, &fields)) return false;
  const auto op = fields.find("op");
  if (op == fields.end()) return false;
  if (op->second == "profile") {
    if (!ExactControlFields(fields, {"op", "container_id", "profile", "expected_topology", "session_generation"})) return false;
    const std::string& id = fields["container_id"]; const std::string& profile = fields["profile"];
    const std::string& topology = fields["expected_topology"]; const std::string& generation = fields["session_generation"];
    if (!ValidContainerID(id) || !ValidSessionGeneration(generation) ||
        (profile != kProfilePythonExecution && profile != kProfileNodeExecution && profile != kProfileBunProjectResolver && profile != kProfileBunResolver && profile != kProfilePNPMResolver && profile != kProfileUVResolver && profile != kProfileNPM && profile != kProfilePyPI && profile != kProfilePyTorchCPU &&
         profile != kProfilePyTorchCU126 && profile != kProfilePyTorchCU130 &&
         profile != kProfilePyTorchCU132 && profile != kProfileGitHub && profile != kProfilePipxRuntime && profile != kProfilePipxNativeResolver && profile != kProfileTerraformProvider && profile != kProfileGoResolver &&
         profile != kProfileGoBuild && profile != kProfileCargoResolver && profile != kProfileCargoBuild) ||
        peer->request_seen || peer->registered || peer->terminal || profiles->find(id) != profiles->end()) {
      return SendProfileAck(peer->fd, id, profile, topology, generation, "rejected");
    }
    std::vector<ExpectedMount> expected; if (!ParseExpectedTopology(topology, &expected)) return false;
    peer->request_seen = true;
    (*profiles)[id] = ProfileRegistration{profile, std::move(expected), generation, peer->fd, {}};
    peer->registered = true;
    peer->container_id = id;
    peer->session_generation = generation;
    return SendProfileAck(peer->fd, id, profile, topology, generation, "registered");
  }
  if (op->second == "invalidate") {
    if (!ExactControlFields(fields, {"op", "container_id", "session_generation"})) return false;
    const std::string& id = fields["container_id"]; const std::string& generation = fields["session_generation"];
    if (!ValidContainerID(id) || !ValidSessionGeneration(generation)) return false;
    peer->request_seen = true;
    auto profile = profiles->find(id); const char* status = "rejected";
    if (peer->registered && !peer->terminal && peer->container_id == id &&
        peer->session_generation == generation && profile != profiles->end() &&
        profile->second.session_generation == generation && profile->second.control_fd == peer->fd) {
      profiles->erase(profile);
      peer->terminal = true;
      status = "invalidated";
    }
    return SendInvalidateAck(peer->fd, id, generation, status);
  }
  if (op->second != "arm" && op->second != "cancel" && op->second != "status" && op->second != "complete" && op->second != "outcome" && op->second != "target") return false;
  if (!ExactControlFields(fields, {"op", "container_id", "session_generation", "mode", "nonce"})) return false;
  const std::string& id = fields["container_id"]; const std::string& generation = fields["session_generation"];
  const std::string& mode = fields["mode"]; const std::string& nonce = fields["nonce"];
  if (!ValidContainerID(id) || !ValidSessionGeneration(generation) ||
      (mode != kLaunchMode && mode != kPythonHandoffMode && mode != kELFHandoffMode && mode != kNodeHandoffMode) || !ValidAdmissionNonce(nonce)) return false;
  peer->request_seen = true;
  auto profile = profiles->find(id); const char* status = "rejected";
  std::string outcome_status;
  if (peer->registered && !peer->terminal && peer->container_id == id &&
      peer->session_generation == generation && profile != profiles->end() &&
      profile->second.session_generation == generation && profile->second.control_fd == peer->fd &&
      (!(IsPipxRuntimeProfile(profile->second.profile.c_str()) || IsPipxNativeProfile(profile->second.profile.c_str())) || mode == kPythonHandoffMode || mode == kLaunchMode) &&
      (mode != kNodeHandoffMode || profile->second.profile == kProfileNodeExecution) &&
      (profile->second.profile != kProfileNodeExecution || mode == kNodeHandoffMode || mode == kLaunchMode) &&
      (profile->second.profile != kProfilePythonExecution || mode == kPythonHandoffMode || mode == kLaunchMode)) {
    auto pending = profile->second.pending_admissions.end();
    for (auto entry = profile->second.pending_admissions.begin(); entry != profile->second.pending_admissions.end(); ++entry) {
      if (entry->session_generation == generation && entry->mode == mode && entry->nonce == nonce) { pending = entry; break; }
    }
    if (op->second == "arm") {
      if (pending == profile->second.pending_admissions.end() && profile->second.pending_admissions.size() < kMaxPendingDirectExecAdmissions) {
        profile->second.pending_admissions.push_back(ProfileRegistration::PendingAdmission{generation, mode, nonce}); status = "armed";
      }
    } else if (pending != profile->second.pending_admissions.end()) {
      if (op->second == "status") {
        status = pending->state == ProfileRegistration::AdmissionState::kPending ? "pending" : "consumed";
      } else if (op->second == "outcome") {
        outcome_status = pending->state == ProfileRegistration::AdmissionState::kPending ? "pending" :
            (pending->terminal_seen ? "exited:" + std::to_string(pending->terminal_wait_status) : "running");
        status = outcome_status.c_str();
      } else if (op->second == "target") {
        status = pending->state == ProfileRegistration::AdmissionState::kPending ? "pending" :
            (pending->target_loaded ? "loaded" : (pending->terminal_seen ? "missed" : "waiting"));
      } else if (op->second == "cancel") {
        if (pending->state == ProfileRegistration::AdmissionState::kPending) {
          profile->second.pending_admissions.erase(pending);
          status = "cancelled";
        } else {
          status = "consumed";
        }
      } else if (op->second == "complete" && pending->state == ProfileRegistration::AdmissionState::kConsumed) {
        profile->second.pending_admissions.erase(pending);
        status = "completed";
      }
    }
  }
  return SendAdmissionAck(peer->fd, op->second, id, generation, mode, nonce, status);
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
      false, false, false, false, ProcessClass::kUnknown, ProcessState::OCIBootstrapStage::kNotOCI,
      creator->second.command_phase});
  state->groups.find(child_group)->second.current_image_class = creator->second.current_image_class;
  state->groups.find(child_group)->second.uv_python_query_restricted = creator->second.uv_python_query_restricted;
  state->groups.find(child_group)->second.uv_python_query_environment = creator->second.uv_python_query_environment;
  state->groups.find(child_group)->second.uv_python_query_helper_root = creator->second.uv_python_query_helper_root;
  auto& native_child = state->groups.find(child_group)->second;
  native_child.bun_run_install_candidate = creator->second.role == ProcessState::Role::kControl &&
      creator->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
      creator->second.current_image_class == ProcessClass::kBun && !creator->second.bun_run_package.empty();
  if (native_child.bun_run_install_candidate) native_child.role=ProcessState::Role::kArtifact;
  native_child.pipx_native_query_restricted = creator->second.pipx_native_query_restricted;
  native_child.pipx_native_environment = creator->second.pipx_native_environment;
  native_child.pipx_native_candidate = IsPipxNativeActiveGroup(creator->second) &&
      (creator->second.pipx_native_operation == "install" || creator->second.pipx_native_operation == "upgrade" || creator->second.pipx_native_operation == "run") &&
      ((creator->second.pipx_native_image == ProcessState::PipxNativeImage::kRoot &&
        creator->second.provenance == ProcessState::Provenance::kDirectExecRoot && creator->second.current_image_class == ProcessClass::kPython) ||
       (creator->second.pipx_native_image == ProcessState::PipxNativeImage::kUV &&
        creator->second.provenance == ProcessState::Provenance::kCloneChild && creator->second.current_image_class == ProcessClass::kUV));
  if (creator->second.role == ProcessState::Role::kControl)
    state->groups.find(child_group)->second.pnpm_target_lifecycle = creator->second.pnpm_target_lifecycle;
  state->groups.find(child_group)->second.executable_locator = creator->second.executable_locator;
  state->groups.find(child_group)->second.diagnostic_image_pinned = creator->second.diagnostic_image_pinned;
  state->groups.find(child_group)->second.cargo_rustc_query_candidate =
      creator->second.current_image_class == ProcessClass::kCargo;
  state->groups.find(child_group)->second.uv_python_query_candidate =
      creator->second.role == ProcessState::Role::kControl &&
      creator->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
      creator->second.current_image_class == ProcessClass::kUV;
  state->groups.find(child_group)->second.pnpm_node_query_candidate =
      creator->second.role == ProcessState::Role::kControl &&
      creator->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
      creator->second.current_image_class == ProcessClass::kPNPM;
  // One kernel clone of the admitted build driver may execute one locked SDK
  // tool. Neither tool authority nor CONTROL/network privilege is inherited.
  state->groups.find(child_group)->second.go_build_tool_candidate = IsGoBuildDriverGroup(creator->second);
  state->groups.find(child_group)->second.go_build_gcc_candidate =
      IsGoBuildDriverGroup(creator->second) || IsGoBuildCompilerProducerGroup(creator->second, *state) ||
      IsCargoBuildCompilerProducerGroup(creator->second, *state);
  state->groups.find(child_group)->second.go_build_gcc_child_candidate =
      IsGoBuildGccProducerGroup(creator->second, *state) || IsCargoBuildGccProducerGroup(creator->second, *state);
  state->groups.find(child_group)->second.cargo_build_program_candidate = IsCargoBuildDriverGroup(creator->second);
  state->groups.find(child_group)->second.cargo_build_program_query_candidate = IsCargoBuildProgramProducerGroup(creator->second, *state);
  state->groups.find(child_group)->second.cargo_lld_launcher_candidate = IsCargoBuildCollect2ProducerGroup(creator->second, *state);
  state->groups.find(child_group)->second.go_build_native_linker_candidate =
      IsGoBuildCollect2ProducerGroup(creator->second, *state);
  state->groups.find(child_group)->second.clone_creator_group_id = creator_group;
  state->groups.find(child_group)->second.clone_creator_group_start_time_ns =
      creator->second.start_time_ns;
  return true;
}

bool ParseSentryExitNotifyParent(const char* payload, size_t payload_size,
                                 std::string* container_id, ProcessState* state,
                                 const char** reason, ProfileRegistration* registration = nullptr) {
  if (state == nullptr) return false;
  gvisor::sentry::ExitNotifyParentInfo message;
  if (!message.ParseFromArray(payload, payload_size) ||
      !ValidateContextContainer(message.context_data(), container_id, reason) ||
      !ValidProcessIdentity(message.context_data())) {
    *reason = "PROCESS_EXIT_INVALID";
    return false;
  }
  const auto& context = message.context_data();
  const int32_t group_id = context.thread_group_id();
  const int64_t start_time_ns = context.thread_group_start_time_ns();
  const auto group = state->groups.find(group_id);
  if (group == state->groups.end()) {
    *reason = "PROCESS_EXIT_UNKNOWN";
    return false;
  }
  if (!SameGroup(group->second, context)) {
    *reason = "PROCESS_EXIT_IDENTITY_REUSED";
    return false;
  }

  const auto expected = state->expected_groups.find(group_id);
  if (expected != state->expected_groups.end()) {
    if (expected->second.start_time_ns != start_time_ns) {
      *reason = "PROCESS_EXIT_IDENTITY_REUSED";
      return false;
    }
    state->expected_groups.erase(expected);
  }
  state->fd_states.erase(group_id);
  state->pending_sockets.erase(group_id);
  for (auto pending = state->pending_socketpairs.begin(); pending != state->pending_socketpairs.end();) {
    if (pending->second.thread_group_id == group_id &&
        pending->second.thread_group_start_time_ns == start_time_ns) {
      pending = state->pending_socketpairs.erase(pending);
    } else {
      ++pending;
    }
  }
  const auto launch = state->launch_roots.find(group_id);
  if (launch != state->launch_roots.end()) {
    if (launch->second != start_time_ns) {
      *reason = "PROCESS_EXIT_IDENTITY_REUSED";
      return false;
    }
    state->launch_roots.erase(launch);
  }
  for (auto pending = state->pending_opens.begin(); pending != state->pending_opens.end();) {
    if (pending->second.thread_group_id == group_id &&
        pending->second.thread_group_start_time_ns == start_time_ns) {
      pending = state->pending_opens.erase(pending);
    } else {
      ++pending;
    }
  }
  if (state->launch_root_set && state->launch_root_group_id == group_id &&
      state->launch_root_group_start_time_ns == start_time_ns) {
    state->launch_root_set = false;
    state->launch_root_active = false;
    state->launch_root_group_id = 0;
    state->launch_root_group_start_time_ns = 0;
  }
  if (state->bootstrap_group_set && state->bootstrap_group_id == group_id &&
      state->bootstrap_group_start_time_ns == start_time_ns) {
    state->bootstrap_active = false;
    state->bootstrap_group_set = false;
    state->bootstrap_group_id = 0;
    state->bootstrap_group_start_time_ns = 0;
  }
  if (group->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
      !RetainDirectExecTerminal(registration, context, message.exit_status())) {
    *reason = "PROCESS_EXIT_INVALID";
    return false;
  }
  state->groups.erase(group);
  return true;
}

size_t MaximumRecords(const char* profile) {
  if (profile == nullptr) return kMaxNormalizedRecordsPerConnection;
  if (strcmp(profile, kProfileGoResolver) == 0) return kMaxGoResolverRecordsPerConnection;
  if (strcmp(profile, kProfileGoBuild) == 0) return kMaxGoBuildRecordsPerConnection;
  if (strcmp(profile, kProfilePyTorchCPU) == 0) return kMaxPyTorchCPURecordsPerConnection;
  if (strcmp(profile, kProfilePyTorchCU126) == 0 || strcmp(profile, kProfilePyTorchCU130) == 0 ||
      strcmp(profile, kProfilePyTorchCU132) == 0) return kMaxPyTorchCU126RecordsPerConnection;
  return kMaxNormalizedRecordsPerConnection;
}

bool RemoveControlPeer(int fd, std::map<int, ControlPeer>* peers,
                       std::map<std::string, ProfileRegistration>* profiles) {
  if (peers == nullptr || profiles == nullptr) return false;
  auto peer = peers->find(fd);
  if (peer == peers->end()) return false;
  if (peer->second.registered) {
    auto profile = profiles->find(peer->second.container_id);
    if (profile != profiles->end() && profile->second.control_fd == fd &&
        profile->second.session_generation == peer->second.session_generation) {
      profiles->erase(profile);
    }
  }
  close(fd);
  peers->erase(peer);
  return true;
}

bool ServiceControlPeer(int fd, short revents, std::map<int, ControlPeer>* peers,
                        std::map<std::string, ProfileRegistration>* profiles) {
  if ((revents & (POLLERR | POLLHUP | POLLNVAL)) != 0) return RemoveControlPeer(fd, peers, profiles);
  if ((revents & POLLIN) == 0) return true;
  char control_message[kMaxControlRecordBytes];
  const ssize_t control_size = recv(fd, control_message, sizeof(control_message), MSG_TRUNC);
  if (control_size <= 0 || static_cast<size_t>(control_size) > sizeof(control_message)) {
    return RemoveControlPeer(fd, peers, profiles);
  }
  auto peer = peers->find(fd);
  return peer != peers->end() && ParseControlRecord(control_message, static_cast<size_t>(control_size),
                                                     &peer->second, profiles);
}

bool AcceptControlPeer(int listener, std::map<int, ControlPeer>* peers) {
  if (peers == nullptr) return false;
  const int accepted = accept(listener, nullptr, nullptr);
  if (accepted < 0) return false;
  peers->emplace(accepted, ControlPeer{accepted, false, false, false, "", ""});
  return true;
}

ProfileRegistration* AwaitProfile(int control, const std::string& container_id,
                                  std::map<int, ControlPeer>* peers,
                                  std::map<std::string, ProfileRegistration>* profiles) {
  if (control < 0 || peers == nullptr || profiles == nullptr) return nullptr;
  for (int waited = 0; waited < kProfileRegistrationWaitMilliseconds; waited += 50) {
    auto profile = profiles->find(container_id);
    if (profile != profiles->end()) return &profile->second;
    std::vector<pollfd> descriptors{{control, POLLIN, 0}};
    for (const auto& peer : *peers) descriptors.push_back(pollfd{peer.first, POLLIN, 0});
    const int result = poll(descriptors.data(), descriptors.size(), 50);
    if (result < 0) {
      if (errno == EINTR) continue;
      return nullptr;
    }
    if (result == 0) continue;
    if ((descriptors[0].revents & POLLIN) != 0 && !AcceptControlPeer(control, peers)) return nullptr;
    for (size_t index = 1; index < descriptors.size(); ++index) {
      if (descriptors[index].revents != 0 &&
          !ServiceControlPeer(descriptors[index].fd, descriptors[index].revents, peers, profiles)) return nullptr;
    }
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

void RecordFDMutationDiagnostic(ProcessState* state, int32_t group, int64_t start,
                                int32_t fd, const char* operation, int32_t family) {
  constexpr size_t maximum = 16;
  if (state->fd_mutation_diagnostics.size() == maximum)
    state->fd_mutation_diagnostics.erase(state->fd_mutation_diagnostics.begin());
  state->fd_mutation_diagnostics.push_back({group, start, fd, operation, family});
}

// Parse a separate diagnostic projection. No outcome, descriptor or accounting
// decision consumes this projection, including when a message is later rejected.
void CaptureFDMutationDiagnostic(const Header& header, const char* payload, size_t size,
                                 ProcessState* state) {
  auto family = [&](int32_t group, int32_t fd) {
    auto table = state->fd_states.find(group);
    if (table == state->fd_states.end()) return int32_t{-1};
    auto entry = table->second.find(fd);
    return entry == table->second.end() ? int32_t{-1} : entry->second.raw_family;
  };
  auto record = [&](const gvisor::common::ContextData& context, int32_t fd,
                    const char* operation, int32_t domain) {
    RecordFDMutationDiagnostic(state, context.thread_group_id(),
        context.thread_group_start_time_ns(), fd, operation, domain);
  };
  switch (static_cast<gvisor::common::MessageType>(header.message_type)) {
    case gvisor::common::MESSAGE_SYSCALL_SOCKET: {
      gvisor::syscall::Socket m;
      if (m.ParseFromArray(payload, size) && m.has_exit() && m.exit().errorno() == 0 &&
          m.exit().result() >= 0 && m.exit().result() <= INT_MAX)
        record(m.context_data(), static_cast<int32_t>(m.exit().result()), "SOCKET", m.domain());
      break;
    }
    case gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR_RESULT: {
      gvisor::syscall::SocketPairResult m;
      if (m.ParseFromArray(payload, size) && m.success() && m.errorno() == 0) {
        record(m.context_data(), m.socket1(), "SOCKETPAIR", m.domain());
        record(m.context_data(), m.socket2(), "SOCKETPAIR", m.domain());
      }
      break;
    }
    case gvisor::common::MESSAGE_SYSCALL_CLOSE: {
      gvisor::syscall::Close m;
      if (m.ParseFromArray(payload, size) && !m.has_exit())
        record(m.context_data(), m.fd(), "CLOSE", family(m.context_data().thread_group_id(), m.fd()));
      break;
    }
    case gvisor::common::MESSAGE_SYSCALL_DUP: {
      gvisor::syscall::Dup m;
      if (m.ParseFromArray(payload, size) && m.has_exit() && m.exit().errorno() == 0 &&
          m.exit().result() >= 0 && m.exit().result() <= INT_MAX)
        record(m.context_data(), static_cast<int32_t>(m.exit().result()), "DUP", family(m.context_data().thread_group_id(), m.old_fd()));
      break;
    }
    case gvisor::common::MESSAGE_SYSCALL_FCNTL: {
      gvisor::syscall::Fcntl m;
      if (m.ParseFromArray(payload, size) && m.has_exit() && m.exit().errorno() == 0 &&
          m.exit().result() >= 0 && m.exit().result() <= INT_MAX &&
          (m.cmd() == kFcntlDupFD || m.cmd() == kFcntlDupFDCloexec))
        record(m.context_data(), static_cast<int32_t>(m.exit().result()), "FCNTL", family(m.context_data().thread_group_id(), m.fd()));
      break;
    }
    case gvisor::common::MESSAGE_SYSCALL_CLONE:
    case gvisor::common::MESSAGE_SYSCALL_FORK: {
      int64_t child = -1;
      if (header.message_type == gvisor::common::MESSAGE_SYSCALL_CLONE) {
        gvisor::syscall::Clone m;
        if (m.ParseFromArray(payload, size) && m.has_exit() && m.exit().errorno() == 0 &&
            (m.flags() & kCloneThread) == 0) child = m.exit().result();
      } else {
        gvisor::syscall::Fork m;
        if (m.ParseFromArray(payload, size) && m.has_exit() && m.exit().errorno() == 0) child = m.exit().result();
      }
      if (child > 0 && child <= INT_MAX) {
        auto group = state->groups.find(static_cast<int32_t>(child));
        if (group != state->groups.end())
          RecordFDMutationDiagnostic(state, group->first, group->second.start_time_ns, -1, "CLONE", -1);
      }
      break;
    }
    case gvisor::common::MESSAGE_SENTRY_EXEC: {
      gvisor::sentry::ExecveInfo m;
      if (m.ParseFromArray(payload, size)) record(m.context_data(), -1, "EXEC", -1);
      break;
    }
    case gvisor::common::MESSAGE_SYSCALL_RAW: {
      gvisor::syscall::Syscall m;
      if (m.ParseFromArray(payload, size) && m.sysno() == kSyscallCloseRange)
        record(m.context_data(), -1, "CLOSE_RANGE", -1);
      break;
    }
    default: break;
  }
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
                                   const char* profile, ProfileRegistration* registration,
                                   ProcessState* process_state, const char** reason,
                                   const TopologyState* topology = nullptr) {
  if (profile == nullptr || registration == nullptr || process_state == nullptr) return false;
  gvisor::sentry::ExecveInfo message;
  if (!message.ParseFromArray(payload, payload_size)) return false;
  if (!ValidateContextContainer(message.context_data(), container_id, reason) || !ValidProcessIdentity(message.context_data())) {
    if (*reason == nullptr) *reason = "EXEC_CORRELATION_INVALID";
    return false;
  }
  const int32_t group_id = message.context_data().thread_group_id();
  const auto prior_group = process_state->groups.find(group_id);
  const auto prior_expected = process_state->expected_groups.find(group_id);
  const ProcessState::GroupState empty_group{};
  const auto snapshot = CaptureExecDiagnostic(
      prior_group == process_state->groups.end() ? empty_group : prior_group->second,
      message.context_data(),
      prior_expected == process_state->expected_groups.end() ? ProcessClass::kUnknown : prior_expected->second.process_class,
      ProcessClassForPath(message.binary_path(), profile), DiagnosticImageForPath(message.binary_path()));
  ProcessState candidate = *process_state;
  const BoundaryMode boundary_mode = BoundaryInvocation(message);
  auto group = candidate.groups.find(group_id);
  bool new_direct_root = false;
  if (group != candidate.groups.end() && !SameGroup(group->second, message.context_data())) {
    *reason = "PROCESS_IDENTITY_REUSED";
    return false;
  }
  if (group == candidate.groups.end()) {
    CommandPhase phase = ClassifyCommandShape(message);
    // NPM_VERSION is an authorization-relevant phase only for the complete
    // canonical helper invocation. Other command shapes are not authorization-relevant.
    if ((phase == CommandPhase::kNpmVersion && !IsExactResolverNpmVersionBoundary(message)) ||
        (phase == CommandPhase::kLockGeneration && !IsExactResolverLockGenerationBoundary(message))) {
      phase = CommandPhase::kOther;
    }
    std::string admission_nonce;
    if (!message.context_data().is_exec_session() || message.context_data().parent_thread_group_id() != 0 ||
        boundary_mode == BoundaryMode::kNone ||
        (boundary_mode == BoundaryMode::kNodeHandoff && strcmp(profile,kProfileNodeExecution) != 0) ||
        (strcmp(profile,kProfilePythonExecution)==0 && boundary_mode!=BoundaryMode::kPythonHandoff &&
         !(boundary_mode==BoundaryMode::kLaunch && IsExactPipxReadiness(message,true))) ||
        (strcmp(profile,kProfileNodeExecution) == 0 && boundary_mode != BoundaryMode::kNodeHandoff &&
         !(boundary_mode == BoundaryMode::kLaunch && IsExactPipxReadiness(message,true))) ||
        ((IsPipxRuntimeProfile(profile) || IsPipxNativeProfile(profile)) && boundary_mode != BoundaryMode::kPythonHandoff &&
         !(boundary_mode == BoundaryMode::kLaunch && IsExactPipxReadiness(message, true))) ||
        !ExtractBoundaryAdmissionNonce(message, boundary_mode, &admission_nonce) ||
        !ConsumeDirectExecAdmission(registration, boundary_mode, admission_nonce) ||
        candidate.groups.size() >= kMaxTrackedProcessGroups ||
        !RegisterGroup(&candidate, message.context_data(), ProcessState::Role::kControl,
                       ProcessState::Provenance::kDirectExecRoot, true, false, phase)) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN";
      return false;
    }
    group = candidate.groups.find(group_id);
    if (!BindDirectExecAdmissionRoot(registration, boundary_mode, admission_nonce, message.context_data())) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN";
      return false;
    }
    if (IsExactPNPMTargetLifecycleBoundary(message, profile, topology) || IsExactBunTargetLifecycleBoundary(message, profile, topology))
      group->second.pnpm_target_lifecycle = message.argv(7);
    if (boundary_mode == BoundaryMode::kNodeHandoff && !BindNodeExecutionTarget(message,topology,&group->second)) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN"; return false;
    }
    if(strcmp(profile,kProfilePythonExecution)==0 && boundary_mode==BoundaryMode::kPythonHandoff && !BindPythonExecutionTarget(message,topology,&group->second)) {
      *reason="PROCESS_PROVENANCE_UNKNOWN"; return false;
    }
    new_direct_root = true;
  }
  group->second.python_execution_active = false;
  group->second.node_execution_active = false;
  group->second.node_execution_demoting = false;
  const bool bun_install_candidate = group->second.bun_run_install_candidate;
  group->second.bun_run_install_candidate = false;
  group->second.bun_run_install_active = false;
  group->second.bun_offline_install_active = false;
  group->second.bun_run_package.clear();
  const bool pipx_native_candidate = group->second.pipx_native_candidate;
  group->second.pipx_native_candidate = false;
  group->second.pipx_native_image = ProcessState::PipxNativeImage::kNone;
  group->second.runtime_cache_query = false;
  const bool pnpm_query_candidate = group->second.pnpm_node_query_candidate;
  group->second.pnpm_node_query_candidate = false;
  group->second.pnpm_node_query_active = false;
  const bool uv_query_candidate = group->second.uv_python_query_candidate;
  group->second.uv_python_query_candidate = false;
  group->second.uv_python_query_active = false;
  const bool cargo_program_candidate=group->second.cargo_build_program_candidate;
  group->second.cargo_build_program_candidate=false;
  group->second.cargo_build_program_active=false;
  const bool cargo_program_query_candidate = group->second.cargo_build_program_query_candidate;
  group->second.cargo_build_program_query_candidate = false;
  group->second.cargo_build_program_query_active = false;
  const bool cargo_query_candidate = group->second.cargo_rustc_query_candidate;
  group->second.cargo_rustc_query_candidate = false;
  group->second.cargo_rustc_query_active = false;
  group->second.cargo_build_rustc_active = false;
  // Consume actual kernel image/active state once. Diagnostic fields never
  // participate in this transition or its later read classification.
  const bool cargo_lld_transition = group->second.current_image_class==ProcessClass::kCargoLldLauncher &&
      group->second.cargo_lld_launcher_active;
  group->second.cargo_rust_lld_active=false;
  group->second.diagnostic_lld_same_group =
      group->second.current_image_class==ProcessClass::kCargoLldLauncher &&
      group->second.cargo_lld_launcher_active &&
      DiagnosticImageForPath(message.binary_path())==DiagnosticImage::kRustLld;
  const bool cargo_lld_candidate=group->second.cargo_lld_launcher_candidate;
  group->second.cargo_lld_launcher_candidate=false;
  group->second.cargo_lld_launcher_active=false;
  const bool go_tool_candidate = group->second.go_build_tool_candidate;
  const bool go_gcc_candidate = group->second.go_build_gcc_candidate;
  const bool go_gcc_child_candidate = group->second.go_build_gcc_child_candidate;
  const bool go_native_linker_candidate = group->second.go_build_native_linker_candidate;
  group->second.go_build_native_linker_candidate = false;
  group->second.go_build_gcc_child_candidate = false;
  group->second.go_build_gcc_candidate = false;
  group->second.go_build_tool_candidate = false;
  group->second.go_build_tool_active = false;
  uint64_t executable_locator = 14695981039346656037ULL;
  for (unsigned char byte : message.binary_path()) {
    executable_locator ^= byte;
    executable_locator *= 1099511628211ULL;
  }
  group->second.executable_locator = executable_locator == 0 ? 1 : executable_locator;
  // Bounded diagnostics only. Permissions use the actual topology predicate,
  // never this cached flag or its serialized value.
  group->second.diagnostic_image_pinned = IsPinnedReadOnlyRootPath(topology, message.binary_path());
  group->second.diagnostic_build_target_image = strcmp(profile,kProfileCargoBuild)==0 && IsCargoBuildTargetImage(topology,message.binary_path());
  group->second.current_image_class = ProcessClassForPath(message.binary_path(), profile);
  const auto* bun_creator = bun_install_candidate ? BunRunInstallCreator(message.context_data(),group->second,candidate) : nullptr;
  const bool bun_install = strcmp(profile,kProfileBunProjectResolver)==0 && bun_creator != nullptr &&
      candidate.expected_groups.find(group_id)==candidate.expected_groups.end() && IsExactBunRunInstall(message,*bun_creator);
  if (bun_install_candidate && !bun_install) { *reason="PROCESS_PROVENANCE_UNKNOWN"; return false; }
  if (group->second.current_image_class == ProcessClass::kUV &&
      (!IsUVToolImage(topology, message.binary_path()) || message.execfn() != kUVToolPath ||
       message.argv_size() == 0 || message.argv(0) != kUVToolPath)) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN";
    return false;
  }
  if (group->second.current_image_class == ProcessClass::kPNPM &&
      (!IsPNPMToolImage(topology, message.binary_path()) || message.execfn() != kPNPMToolPath ||
       message.argv_size() == 0 || message.argv(0) != kPNPMToolPath)) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN";
    return false;
  }
  if (group->second.current_image_class == ProcessClass::kBun &&
      (!IsBunToolImage(topology, message.binary_path()) ||
       !((strcmp(profile, kProfileBunResolver) == 0 && IsExactBunQuery(message)) ||
         (strcmp(profile, kProfileBunProjectResolver) == 0 &&
          (IsExactBunProjectLockInstall(message) ||
           (IsExactBunRunSelection(message) && group->second.role == ProcessState::Role::kControl &&
            group->second.provenance == ProcessState::Provenance::kDirectExecRoot && group->second.launch_target_pending) || bun_install)) ||
         (strcmp(profile, kProfileNPM) == 0 && group->second.role == ProcessState::Role::kArtifact &&
          IsBunArtifactRuntimeImage(topology, message.binary_path()) && message.execfn() == kBunToolPath &&
          message.argv_size() > 0 && (message.argv(0) == kBunToolPath || message.argv(0) == "bun"))))) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN";
    return false;
  }
  if ((strcmp(profile, kProfileCargoResolver) == 0 || strcmp(profile, kProfileCargoBuild) == 0) &&
      (group->second.current_image_class == ProcessClass::kCargo ||
       group->second.current_image_class == ProcessClass::kCargoTar ||
       group->second.current_image_class == ProcessClass::kCargoRustc ||
       group->second.current_image_class == ProcessClass::kGoBuildGcc ||
       group->second.current_image_class == ProcessClass::kGoBuildCollect2 ||
       group->second.current_image_class == ProcessClass::kCargoLldLauncher ||
       group->second.current_image_class == ProcessClass::kCargoRustLld ||
       group->second.current_image_class == ProcessClass::kProjectBuildSetpriv ||
       group->second.current_image_class == ProcessClass::kMkdir ||
       group->second.current_image_class == ProcessClass::kShell) &&
      !IsPinnedReadOnlyRootPath(topology, message.binary_path())) {
    group->second.current_image_class = ProcessClass::kUnknown;
  }
  // Kernel-resolved utility image in the sealed immutable runtime, not comm,
  // argv[0], an artifact copy or a path shadow. This only classifies loader reads;
  // the ordinary ARTIFACT unexpected-exec event below is unchanged.
  if (group->second.current_image_class == ProcessClass::kUname &&
      (!IsPinnedReadOnlyRootPath(topology, message.binary_path()) ||
       !IsPinnedReadOnlyRootPath(topology, "/etc/ld.so.cache"))) {
    group->second.current_image_class = ProcessClass::kUnknown;
  }
  // The new build target remains ARTIFACT. A shadow of its locked SDK image
  // cannot obtain the GO classification or consume the one-shot handoff.
  if (strcmp(profile, kProfileGoBuild) == 0 &&
      (group->second.current_image_class == ProcessClass::kGo ||
       group->second.current_image_class == ProcessClass::kMkdir ||
       group->second.current_image_class == ProcessClass::kShell ||
       group->second.current_image_class == ProcessClass::kGoBuildTool ||
       group->second.current_image_class == ProcessClass::kGoBuildCgo ||
       group->second.current_image_class == ProcessClass::kGoBuildLink ||
       group->second.current_image_class == ProcessClass::kGoBuildGcc ||
       group->second.current_image_class == ProcessClass::kGoBuildCc1 ||
       group->second.current_image_class == ProcessClass::kGoBuildAssembler ||
       group->second.current_image_class == ProcessClass::kGoBuildCollect2 ||
       group->second.current_image_class == ProcessClass::kGoBuildNativeLinker ||
       group->second.current_image_class == ProcessClass::kProjectBuildSetpriv) &&
      !IsPinnedReadOnlyRootPath(topology, message.binary_path())) {
    group->second.current_image_class = ProcessClass::kUnknown;
  }
  if ((strcmp(profile, kProfileGoBuild) == 0 || strcmp(profile, kProfileCargoBuild) == 0) &&
      group->second.current_image_class == ProcessClass::kProjectBuildBoundary) {
    bool exact_helper = topology != nullptr && topology->sealed &&
        topology->snapshot_seen && topology->namespace_id != 0;
    size_t helper_mounts = 0;
    if (exact_helper) {
      for (const auto& entry : topology->anchors) {
        const auto& mount = entry.second;
        if (mount.mountpoint == "/haa-runtime" && mount.mount_class == "helper") helper_mounts++;
        else if (IsAtOrBelowMountpoint(message.binary_path(), mount.mountpoint) &&
                 mount.mountpoint != "/") exact_helper = false;
      }
    }
    if (!exact_helper || helper_mounts != 1) group->second.current_image_class = ProcessClass::kUnknown;
  }
  group->second.diagnostic_image = DiagnosticImageForPath(message.binary_path());
  group->second.diagnostic_native_cc = IsExactCargoNativeCCInvocation(message);
  group->second.diagnostic_rustc_version = message.binary_path() == kRustcBinary &&
      message.execfn() == kRustcBinary && message.argv_size() == 2 &&
      message.argv(0) == kRustcBinary && (message.argv(1) == "-vV" || message.argv(1) == "--version");
  group->second.diagnostic_rustc_metadata = IsExactCargoRustcMetadataQuery(message);
  group->second.diagnostic_rustc_argc = 0;
  group->second.diagnostic_rustc_argv_locator = 0;
  group->second.diagnostic_socketpair_seen = false;
  if (message.binary_path() == kRustcBinary) {
    group->second.diagnostic_rustc_argc = static_cast<uint32_t>(std::min(message.argv_size(), 33));
    if (message.argv_size() <= 32) {
      uint64_t hash = 14695981039346656037ULL;
      size_t bytes = 0;
      for (const auto& argument : message.argv()) {
        bytes += argument.size() + 1;
        if (bytes > 1024) break;
        for (const unsigned char character : argument) { hash ^= character; hash *= 1099511628211ULL; }
        hash ^= 0; hash *= 1099511628211ULL;
      }
      if (bytes <= 1024) group->second.diagnostic_rustc_argv_locator = hash == 0 ? 1 : hash;
    }
  }
  // One exact kernel clone of the admitted Cargo driver may query the locked
  // compiler version or fixed target metadata. Other compiler commands and later execs gain no reads;
  // the diagnostic flags above never participate in this permission check.
  group->second.cargo_rustc_query_active = strcmp(profile, kProfileCargoResolver) == 0 &&
      cargo_query_candidate && group->second.current_image_class == ProcessClass::kCargoRustc &&
      HasExactCargoResolverCreator(message.context_data(), group->second, candidate) &&
      message.binary_path() == kRustcBinary && message.execfn() == kRustcBinary &&
      ((message.argv_size() == 2 && message.argv(0) == kRustcBinary && message.argv(1) == "-vV") ||
       IsExactCargoRustcMetadataQuery(message));
  if (strcmp(profile, kProfilePNPMResolver) == 0 &&
      group->second.role == ProcessState::Role::kControl) {
    auto query_group = group->second;
    query_group.role = ProcessState::Role::kArtifact;
    const bool exact_creator = HasExactPNPMQueryCreator(message.context_data(), query_group, candidate);
    if (pnpm_query_candidate || exact_creator) {
      const bool query = pnpm_query_candidate && exact_creator &&
          candidate.expected_groups.find(group_id) == candidate.expected_groups.end() &&
          IsExactPNPMNodeQuery(message, topology);
      // Both a valid query and a rejected PNPM candidate lose CONTROL. A
      // rejected exec cannot regain it through the generic helper fallback.
      group->second.role = ProcessState::Role::kArtifact;
      group->second.trusted_control_network_active = false;
      group->second.pnpm_node_query_active = query;
      if (query) group->second.current_image_class = ProcessClass::kNode;
      candidate.expected_groups.erase(group_id);
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->expected_groups = candidate.expected_groups;
      process_state->fd_states = candidate.fd_states;
      if (query) return Send(output, *container_id, "process-exec-expected");
      const Attribution attribution{"SENTRY_EXEC", nullptr, nullptr,
          ProcessClassName(group->second.current_image_class), "ARTIFACT_ROLE", "ARTIFACT_GROUP"};
      RecordUnexpectedExec(&process_state->first_unexpected_exec, snapshot, "ARTIFACT_ROLE", "ARTIFACT_GROUP");
      return Send(output, *container_id, "process-exec-unexpected", nullptr, &attribution);
    }
  }
  std::string uv_query_helper_root;
  if (strcmp(profile, kProfileUVResolver) == 0 && uv_query_candidate &&
      group->second.role == ProcessState::Role::kControl &&
      candidate.expected_groups.find(group_id) == candidate.expected_groups.end() &&
      IsExactUVPythonQuery(message, topology, &uv_query_helper_root)) {
    auto query_group = group->second;
    query_group.role = ProcessState::Role::kArtifact;
    if (HasExactUVQueryCreator(message.context_data(), query_group, candidate)) {
      group->second.role = ProcessState::Role::kArtifact;
      group->second.current_image_class = ProcessClass::kPython;
      group->second.uv_python_query_active = true;
      group->second.uv_python_query_restricted = true;
      group->second.uv_python_query_environment = UVToolQueryEnvironment(message.execfn(), topology);
      group->second.uv_python_query_helper_root = uv_query_helper_root;
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output, *container_id, "process-exec-expected");
    }
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
      boundary_mode == BoundaryMode::kELFHandoff || boundary_mode == BoundaryMode::kNodeHandoff) {
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
      group->second.handoff_target_class = boundary_mode == BoundaryMode::kNodeHandoff ? ProcessClass::kNode : boundary_mode == BoundaryMode::kPythonHandoff
          ? ProcessClass::kPython : (strcmp(profile, kProfileGoBuild) == 0
              ? ProcessClass::kGo : (strcmp(profile, kProfileCargoBuild) == 0
                  ? ProcessClass::kCargo : ProcessClass::kArtifact));
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
    if (strcmp(profile,kProfileNodeExecution) == 0 && group->second.role == ProcessState::Role::kArtifact) {
      if (!IsPinnedReadOnlyRootPath(topology,kSetprivPath) || !SameNodeExecutionTarget(message,group->second,9)) {
        *reason = "PROCESS_PROVENANCE_UNKNOWN"; return false;
      }
      group->second.node_execution_demoting = true;
    }
    if(strcmp(profile,kProfilePythonExecution)==0 && group->second.role==ProcessState::Role::kArtifact) {
      if(!IsPinnedReadOnlyRootPath(topology,kSetprivPath) || !SameCapturedExecutionTarget(message,group->second.python_execution_target,9)) {
        *reason="PROCESS_PROVENANCE_UNKNOWN"; return false;
      }
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
    if ((strcmp(profile,kProfilePythonExecution)==0 || IsPipxRuntimeProfile(profile) || IsPipxNativeProfile(profile)) && !IsExactPipxReadiness(message, false)) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN";
      return false;
    }
    const ProcessClass process_class = ProcessClassForPath(message.binary_path(), profile);
    const TrackResult tracked_result = TrackExpectedProcessGroup(message.context_data(), process_class, &candidate);
    if (tracked_result != TrackResult::kTracked) {
      *reason = TrackFailureReason(tracked_result);
      return false;
    }
    if (group->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
        !RetainDirectExecTarget(registration, message.context_data(), BoundaryMode::kLaunch)) {
      *reason = "PROCESS_PROVENANCE_UNKNOWN";
      return false;
    }
    group->second.launch_target_pending = false;
    group->second.trusted_control_network_active = true;
    if (strcmp(profile,kProfileBunProjectResolver)==0 && IsExactBunRunSelection(message))
      group->second.bun_run_package = message.argv(5).substr(10);
    if (strcmp(profile,kProfileBunProjectResolver)==0 && IsExactBunProjectLockInstall(message) &&
        message.argv_size()==14 && message.argv(3)=="--offline")
      group->second.bun_offline_install_active = true;
    group->second.npm_version_node_transition_pending =
        group->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
        group->second.command_phase == CommandPhase::kNpmVersion &&
        process_class == ProcessClass::kNpm && IsExactNpmVersionLauncher(message);
    group->second.npm_version_node_transition_consumed = false;
    ApplyExecCloexec(&candidate, group_id);
    process_state->groups = candidate.groups;
    process_state->expected_groups = candidate.expected_groups;
    process_state->fd_states = candidate.fd_states;
    return Send(output, *container_id, "process-exec-expected");
  }
  if (bun_install) {
    const auto tracked = TrackExpectedProcessGroup(message.context_data(),ProcessClass::kBun,&candidate);
    if (tracked != TrackResult::kTracked) { *reason=TrackFailureReason(tracked); return false; }
    group->second.bun_run_install_active = true;
    ApplyExecCloexec(&candidate,group_id);
    process_state->groups=candidate.groups;
    process_state->expected_groups=candidate.expected_groups;
    process_state->fd_states=candidate.fd_states;
    return Send(output,*container_id,"process-exec-expected");
  }
  if (group->second.role == ProcessState::Role::kArtifact) {
    if (IsPipxNativeProfile(profile) && pipx_native_candidate) {
      const auto* root = PipxNativeCreator(message.context_data(),group->second,candidate,ProcessState::PipxNativeImage::kRoot);
      const auto* uv = PipxNativeCreator(message.context_data(),group->second,candidate,ProcessState::PipxNativeImage::kUV);
      const ProcessState::GroupState* parent = nullptr;
      std::string environment;
      if (root != nullptr && group->second.current_image_class == ProcessClass::kUV && IsExactPipxNativeUVChild(message,topology,*root)) {
        parent = root;
        group->second.pipx_native_image = ProcessState::PipxNativeImage::kUV;
      } else if (root != nullptr && group->second.current_image_class == ProcessClass::kPython &&
                 IsExactPipxNativePythonQuery(message,topology,*root,true,&environment)) {
        parent = root;
        group->second.pipx_native_image = ProcessState::PipxNativeImage::kMetadataQuery;
      } else if (uv != nullptr && group->second.current_image_class == ProcessClass::kPython &&
                 IsExactPipxNativePythonQuery(message,topology,*uv,false,&environment)) {
        parent = uv;
        group->second.pipx_native_image = ProcessState::PipxNativeImage::kUVQuery;
      }
      if (parent != nullptr) {
        group->second.pipx_native_operation = parent->pipx_native_operation;
        group->second.pipx_native_project = parent->pipx_native_project;
        group->second.pipx_native_requirement = parent->pipx_native_requirement;
        group->second.pipx_native_environment = environment;
        group->second.pipx_native_query_restricted = group->second.pipx_native_image != ProcessState::PipxNativeImage::kUV;
        ApplyExecCloexec(&candidate,group_id);
        process_state->groups = candidate.groups;
        process_state->fd_states = candidate.fd_states;
        return Send(output,*container_id,"process-exec-expected");
      }
    }
    group->second.pnpm_target_lifecycle.clear();
    group->second.pnpm_target_npm_pending = false;
    if (strcmp(profile,kProfileCargoBuild) == 0 && cargo_program_query_candidate &&
        group->second.current_image_class == ProcessClass::kCargoRustc &&
        message.binary_path() == kRustcBinary && message.execfn() == kRustcBinary &&
        message.argv_size() == 2 && message.argv(0) == kRustcBinary && message.argv(1) == "--version" &&
        HasExactCargoBuildProgramCreator(message.context_data(),group->second,candidate)) {
      group->second.cargo_build_program_query_active = true;
      ApplyExecCloexec(&candidate,group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output,*container_id,"process-exec-expected");
    }
    // Compiled project/build code stays untrusted ARTIFACT. The image must be
    // kernel-resolved in the exact host-attested executable target volume and
    // loaded once by its owned Cargo clone. Neither names nor diagnostics grant
    // this transition, and it grants no CONTROL/expected-group/network authority.
    if(strcmp(profile,kProfileCargoBuild)==0 && cargo_program_candidate &&
       IsCargoBuildTargetImage(topology,message.binary_path()) &&
       HasExactCargoBuildDriverCreator(message.context_data(),group->second,candidate)) {
      group->second.current_image_class=ProcessClass::kCargoBuildProgram;
      group->second.cargo_build_program_active=true;
      ApplyExecCloexec(&candidate,group_id);
      process_state->groups=candidate.groups;
      process_state->fd_states=candidate.fd_states;
      return Send(output,*container_id,"process-exec-expected");
    }

    // The locked launcher reexecs the locked linker in the same kernel group.
    // Untrusted linker arguments confer no CONTROL/network authority.
    if(strcmp(profile,kProfileCargoBuild)==0 && cargo_lld_transition &&
       group->second.current_image_class==ProcessClass::kCargoRustLld &&
       HasExactCargoBuildCollect2ChildCreator(message.context_data(),group->second,candidate)) {
      group->second.cargo_rust_lld_active=true;
      ApplyExecCloexec(&candidate,group_id);
      process_state->groups=candidate.groups;
      process_state->fd_states=candidate.fd_states;
      return Send(output,*container_id,"process-exec-expected");
    }

    if(strcmp(profile,kProfileCargoBuild)==0 && cargo_lld_candidate &&
       group->second.current_image_class==ProcessClass::kCargoLldLauncher && IsExactCargoLldLauncherInvocation(message) &&
       HasExactCargoBuildCollect2ChildCreator(message.context_data(),group->second,candidate)) {
      group->second.cargo_lld_launcher_active=true;
      ApplyExecCloexec(&candidate,group_id);
      process_state->groups=candidate.groups;
      process_state->fd_states=candidate.fd_states;
      return Send(output,*container_id,"process-exec-expected");
    }

    if(strcmp(profile,kProfileCargoBuild)==0 && go_gcc_candidate &&
       group->second.current_image_class==ProcessClass::kGoBuildGcc &&
       IsExactCargoNativeCCInvocation(message) && IsPinnedReadOnlyRootPath(topology,message.execfn()) &&
       HasExactCargoBuildCompilerCreator(message.context_data(),group->second,candidate)) {
      group->second.go_build_tool_active=true;
      ApplyExecCloexec(&candidate,group_id);
      process_state->groups=candidate.groups;
      process_state->fd_states=candidate.fd_states;
      return Send(output,*container_id,"process-exec-expected");
    }

    if (strcmp(profile,kProfileCargoBuild)==0 && cargo_query_candidate &&
        group->second.current_image_class==ProcessClass::kCargoRustc &&
        message.binary_path()==kRustcBinary && message.execfn()==kRustcBinary &&
        message.argv_size()>0 && message.argv(0)==kRustcBinary &&
        HasExactCargoBuildDriverCreator(message.context_data(),group->second,candidate)) {
      group->second.cargo_build_rustc_active=true;
      ApplyExecCloexec(&candidate,group_id);
      process_state->groups=candidate.groups;
      process_state->fd_states=candidate.fd_states;
      return Send(output,*container_id,"process-exec-expected");
    }

    const ProcessClass process_class = (strcmp(profile, kProfileGoBuild) == 0 || strcmp(profile, kProfileCargoBuild) == 0)
        ? group->second.current_image_class : ProcessClassForPath(message.binary_path(), profile);
    const bool exact_sdk_tool = (process_class == ProcessClass::kGoBuildTool ||
        process_class == ProcessClass::kGoBuildCgo ||
        process_class == ProcessClass::kGoBuildLink) &&
        message.execfn() == message.binary_path() && message.argv_size() > 0 &&
        message.argv(0) == message.binary_path();
    const bool exact_gcc = process_class == ProcessClass::kGoBuildGcc &&
        message.execfn() == "/usr/bin/gcc" && message.argv_size() > 0 &&
        (message.argv(0) == "gcc" || message.argv(0) == "/usr/bin/gcc") &&
        IsPinnedReadOnlyRootPath(topology, message.execfn());
    const bool exact_gcc_internal_tool = (process_class == ProcessClass::kGoBuildCc1 ||
        process_class == ProcessClass::kGoBuildCollect2) &&
        message.execfn() == message.binary_path() && message.argv_size() > 0 &&
        message.argv(0) == message.binary_path();
    const bool exact_assembler = process_class == ProcessClass::kGoBuildAssembler &&
        message.execfn() == "/usr/bin/as" && message.argv_size() > 0 &&
        (message.argv(0) == "as" || message.argv(0) == "/usr/bin/as") &&
        IsPinnedReadOnlyRootPath(topology, message.execfn());
    const bool exact_native_linker = process_class == ProcessClass::kGoBuildNativeLinker &&
        message.execfn() == "/usr/bin/ld" && message.argv_size() > 0 && message.argv(0) == "/usr/bin/ld" &&
        IsPinnedReadOnlyRootPath(topology, message.execfn());
    if(strcmp(profile,kProfileCargoBuild)==0 && go_gcc_child_candidate &&
       process_class==ProcessClass::kGoBuildCollect2 && exact_gcc_internal_tool &&
       HasExactCargoBuildGccChildCreator(message.context_data(),group->second,candidate)) {
      group->second.go_build_tool_active=true;
      ApplyExecCloexec(&candidate,group_id);
      process_state->groups=candidate.groups;
      process_state->fd_states=candidate.fd_states;
      return Send(output,*container_id,"process-exec-expected");
    }
    if (strcmp(profile, kProfileGoBuild) == 0 &&
        ((go_tool_candidate && exact_sdk_tool &&
          HasExactGoBuildDriverCreator(message.context_data(), group->second, candidate)) ||
         (go_gcc_candidate && exact_gcc &&
          HasExactGoBuildGccCreator(message.context_data(), group->second, candidate)) ||
         (go_gcc_child_candidate && (exact_gcc_internal_tool || exact_assembler) &&
          HasExactGoBuildGccChildCreator(message.context_data(), group->second, candidate)) ||
         (go_native_linker_candidate && exact_native_linker &&
          HasExactGoBuildNativeLinkerCreator(message.context_data(), group->second, candidate)))) {
      group->second.go_build_tool_active = true;
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output, *container_id, "process-exec-expected");
    }
    if (group->second.handoff_target_pending && process_class == group->second.handoff_target_class) {
      if (strcmp(profile,kProfileNodeExecution) == 0) {
        if (process_class != ProcessClass::kNode || message.binary_path() != kNodePath || message.execfn() != kNodePath ||
            !SameNodeExecutionTarget(message,group->second,0) || !IsNodeExecutionInputPath(topology,message.argv(2))) {
          *reason = "PROCESS_PROVENANCE_UNKNOWN"; return false;
        }
        group->second.node_execution_active = true;
        group->second.node_execution_target.clear();
      }
      if(strcmp(profile,kProfilePythonExecution)==0) {
        if(process_class!=ProcessClass::kPython || message.binary_path()!=kPythonExecutionPath || message.execfn()!=kPythonExecutionPath ||
            !IsPythonExecutionTopology(topology) || !SameCapturedExecutionTarget(message,group->second.python_execution_target,0)) {
          *reason="PROCESS_PROVENANCE_UNKNOWN"; return false;
        }
        group->second.python_execution_active=true;
        group->second.python_execution_target.clear();
      }
      if (IsPipxNativeProfile(profile) && !ParsePipxNativeRoot(message,topology,&group->second)) {
        *reason = "PROCESS_PROVENANCE_UNKNOWN";
        return false;
      }
      if (group->second.provenance == ProcessState::Provenance::kDirectExecRoot &&
          !RetainDirectExecTarget(registration, message.context_data(),
              group->second.handoff_target_class == ProcessClass::kNode ? BoundaryMode::kNodeHandoff :
                  (group->second.handoff_target_class == ProcessClass::kPython ?
                      BoundaryMode::kPythonHandoff : BoundaryMode::kELFHandoff))) {
        *reason = "PROCESS_PROVENANCE_UNKNOWN";
        return false;
      }
      group->second.handoff_target_pending = false;
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output, *container_id, "process-exec-expected");
    }
    if (IsPythonProfile(profile) && IsSupportedLdconfigCacheQuery(message, topology)) {
      // Expected operation remains ARTIFACT. Do not touch expected_groups,
      // root eligibility, network attribution, or launch admission.
      group->second.runtime_cache_query = true;
      ApplyExecCloexec(&candidate, group_id);
      process_state->groups = candidate.groups;
      process_state->fd_states = candidate.fd_states;
      return Send(output, *container_id, "process-exec-expected");
    }
    ApplyExecCloexec(&candidate, group_id);
    const char* artifact_reason = IsPythonProfile(profile) && IsExactLdconfigCacheQuery(message)
        ? "ARTIFACT_LDCONFIG_QUERY" : "ARTIFACT_ROLE";
    const Attribution attribution{"SENTRY_EXEC", nullptr, nullptr,
                                  ProcessClassName(process_class), artifact_reason, "ARTIFACT_GROUP"};
    process_state->groups = candidate.groups;
    process_state->fd_states = candidate.fd_states;
    RecordUnexpectedExec(&process_state->first_unexpected_exec, snapshot,
                         artifact_reason, "ARTIFACT_GROUP");
    return Send(output, *container_id, "process-exec-unexpected", nullptr, &attribution);
  }
  auto tracked_group = candidate.expected_groups.find(group_id);
  const bool exact_npm_node_transition =
      tracked_group != candidate.expected_groups.end() &&
      IsExactNpmNodeTransition(message, profile, group_id, group->second, tracked_group->second);
  const bool exact_npm_version_node_transition =
      tracked_group != candidate.expected_groups.end() &&
      IsExactResolverNpmVersionNodeTransition(message, profile, group_id, group->second,
                                              tracked_group->second);
  const bool exact_lock_generation_npm_node_transition =
      tracked_group != candidate.expected_groups.end() &&
      IsExactResolverLockGenerationNpmNodeTransition(message, profile, group_id, group->second,
                                                      tracked_group->second, candidate);
  const bool exact_pnpm_target_npm_node_transition =
      tracked_group != candidate.expected_groups.end() &&
      IsExactPNPMTargetNpmNodeTransition(message, profile, group_id, group->second, tracked_group->second, candidate);
  if (exact_npm_node_transition || exact_npm_version_node_transition ||
      exact_lock_generation_npm_node_transition || exact_pnpm_target_npm_node_transition) {
    tracked_group->second.process_class = ProcessClass::kNode;
    group->second.npm_node_transition_pending = false;
    group->second.pnpm_target_npm_pending = false;
    group->second.pnpm_target_npm_consumed = exact_pnpm_target_npm_node_transition;
    group->second.lock_generation_npm_node_transition_pending = false;
    group->second.npm_version_node_transition_pending = false;
    group->second.npm_version_node_transition_consumed = exact_npm_version_node_transition;
    ApplyExecCloexec(&candidate, group_id);
    process_state->expected_groups = candidate.expected_groups;
    process_state->groups = candidate.groups;
    process_state->fd_states = candidate.fd_states;
    return Send(output, *container_id, "process-exec-expected");
  }
  // A later successful image transition in the direct root cannot retain the
  // narrow trusted-control network exception or the NPM_VERSION runtime-read
  // exception.
  if (group->second.provenance == ProcessState::Provenance::kDirectExecRoot) {
    group->second.trusted_control_network_active = false;
    group->second.npm_version_node_transition_consumed = false;
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
  if (classification.expected && updated_group != process_state->groups.end() &&
      MayArmExactLockGenerationNpmNodeTransition(message, profile, updated_group->second,
                                                  *process_state)) {
    updated_group->second.lock_generation_npm_node_transition_pending = true;
  }
  if (classification.expected && updated_group != process_state->groups.end() &&
      MayArmExactPNPMTargetNpmTransition(message, profile, updated_group->second, *process_state)) {
    updated_group->second.pnpm_target_npm_pending = true;
  }
  ApplyExecCloexec(process_state, group_id);
  if (classification.expected) return Send(output, *container_id, "process-exec-expected");
  const Attribution attribution{"SENTRY_EXEC", nullptr, nullptr,
                                ProcessClassName(classification.process_class), classification.reason,
                                classification.parent_relation};
  RecordUnexpectedExec(&process_state->first_unexpected_exec, snapshot,
                       classification.reason, classification.parent_relation);
  return Send(output, *container_id, "process-exec-unexpected", nullptr, &attribution);
}

// Unit parser callers without a trusted control registration intentionally
// receive no direct-root authority. This compatibility overload cannot arm an
// admission and therefore preserves fail-closed behavior.
bool ParseSentryProcessAndClassify(const char* payload, size_t payload_size, int output, std::string* container_id,
                                   const char* profile, ProcessState* process_state, const char** reason) {
  ProfileRegistration no_admissions{profile == nullptr ? "" : profile, {}, std::string(64, '0'), {}};
  return ParseSentryProcessAndClassify(payload, payload_size, output, container_id, profile,
                                       &no_admissions, process_state, reason);
}

bool ParseOpenAndSend(const char* payload, size_t payload_size, int output, std::string* container_id,
                      const char* profile, ProcessState* state, NormalizedCounts* counts, const char** reason) {
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
  } else if (IsCargoBuildConfigurationWrite(message.context_data(), *state, message.pathname(), message.flags(), profile) ||
             IsClearlyOutsideWorkspace(message.pathname()) ||
             (IsWriteCapableOpen(message.flags()) && message.pathname() != "/dev/null" &&
              !message.pathname().empty() && message.pathname()[0] == '/' &&
              !IsWorkspacePath(message.pathname(), profile) &&
              !IsExactBootstrapHelperWrite(message.context_data(), *state, message.pathname(), message.flags()) &&
              !(HasPrefix(message.pathname(), "/usr/local/lib/python3.14/") &&
                message.pathname().find("/__pycache__/") != std::string::npos))) {
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
                            const TopologyState& topology, const char** reason, FaultSite* fault_site = nullptr) {
  auto detail = [fault_site](FaultSite site) { if (fault_site != nullptr) *fault_site = site; };
  if (profile == nullptr || state == nullptr || counts == nullptr || !topology.sealed) {
    return false;
  }
  gvisor::syscall::OpenResult result;
  if (!result.ParseFromArray(payload, payload_size) || !ValidateContextContainer(result.context_data(), container_id, reason) ||
      result.context_data().thread_id() <= 0 || result.context_data().thread_start_time_ns() <= 0) {
    detail(FaultSite::kOpenResultEnvelope); *reason = "STREAM_FAULT"; return false;
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
  // Pinned gVisor emits a kernel OpenResult for openat2 but has no entry
  // point for it. Only an exact native installer may account a successful
  // O_PATH|O_CLOEXEC handle in its private cache or offline project tree.
  // This grants no file read/write; failed, foreign and other orphan results
  // still fail correlation. A mismatched existing entry is never bypassed.
  const auto expected = state->expected_groups.find(result.context_data().thread_group_id());
  const bool bun_offline = group != state->groups.end() && group->second.bun_offline_install_active &&
      result.context_data().is_exec_session() && result.context_data().parent_thread_group_id()==0 &&
      IsTrustedControlNetwork(result.context_data(),*state) && group->second.current_image_class==ProcessClass::kBun &&
      !group->second.root_eligible && group->second.root_consumed && !group->second.launch_target_pending &&
      !group->second.handoff_target_pending && !group->second.demotion_pending &&
      expected!=state->expected_groups.end() && expected->second.start_time_ns==group->second.start_time_ns &&
      expected->second.process_class==ProcessClass::kBun;
  const bool bun_path_handle = pending == state->pending_opens.end() && result.sysno() == 437 &&
      result.flags() == 2654208 && result.success() && strcmp(profile,kProfileBunProjectResolver)==0 &&
      ((IsBunRunInstallGroup(result.context_data(),*state) &&
        HasPrefix(result.resolved_pathname(),"/tmp/haa-bun-input/cache/.bunx-1000/")) ||
       (bun_offline && HasPrefix(result.resolved_pathname(),"/tmp/haa-bun-input/project/node_modules/")));
  if (!bun_path_handle && (pending == state->pending_opens.end() || pending->second.thread_group_id != result.context_data().thread_group_id() ||
      pending->second.thread_group_start_time_ns != result.context_data().thread_group_start_time_ns() ||
      pending->second.sysno != result.sysno() ||
      (pending->second.flags | kOpenLargefile) != (result.flags() | kOpenLargefile))) {
    detail(FaultSite::kOpenResultCorrelation); *reason = "STREAM_FAULT"; return false;
  }
  const bool early = !bun_path_handle && pending->second.early_finding_emitted;
  if (!bun_path_handle) state->pending_opens.erase(pending);
  if (!result.success()) {
    if (result.errorno() == 0 || !result.resolved_pathname().empty() || result.mount_id() != 0) {
      detail(FaultSite::kOpenResultFailureFormat); *reason = "STREAM_FAULT"; return false;
    }
    return true;
  }
  if (result.errorno() != 0 || result.mount_id() == 0 || !IsNormalizedAbsolutePath(result.resolved_pathname())) {
    detail(FaultSite::kOpenResultSuccessFormat); *reason = "STREAM_FAULT"; return false;
  }
  const auto anchor = topology.anchors.find(result.mount_id());
  if (anchor == topology.anchors.end() || !IsAtOrBelowMountpoint(result.resolved_pathname(), anchor->second.mountpoint)) {
    detail(FaultSite::kOpenResultAnchor); *reason = "STREAM_FAULT"; return false;
  }
  for (const auto& candidate : topology.anchors) {
    if (candidate.first != anchor->first && candidate.second.mountpoint.size() > anchor->second.mountpoint.size() &&
        IsAtOrBelowMountpoint(result.resolved_pathname(), candidate.second.mountpoint)) {
      detail(FaultSite::kOpenResultShadow); *reason = "STREAM_FAULT"; return false;
    }
  }
  if (early) return true;
  if (bun_path_handle) {
    if (anchor->second.mount_class != "workspace" || anchor->second.mountpoint != kBunTargetProjectMount) {
      detail(FaultSite::kOpenResultAnchor); *reason="STREAM_FAULT"; return false;
    }
    if (counts->workspace_access < kMaxNormalizedObservationCount) ++counts->workspace_access;
    return true;
  }
  if (IsHoneytoken(result.resolved_pathname())) {
    if (!Send(output, *container_id, "honeytoken-access")) return false;
    ++counts->immediate_records; return true;
  }
  if (IsCargoBuildConfigurationWrite(result.context_data(), *state, result.resolved_pathname(), result.flags(), profile)) {
    if (!Send(output, *container_id, "filesystem-outside-workspace")) return false;
    ++counts->immediate_records; return true;
  }
  // Workspace attribution does not override a query's narrower read/write
  // contract. Dispatch restricted queries and the native profile through the
  // same predicate used for non-workspace opens, after all resolved-path and
  // mount/correlation checks above. Other profiles retain their fast path.
  const bool restricted_workspace = strcmp(profile,kProfilePythonExecution)==0 || IsPipxNativeProfile(profile) ||
      (strcmp(profile, kProfileUVResolver) == 0 && group != state->groups.end() &&
       group->second.uv_python_query_restricted);
  if (anchor->second.mount_class == "workspace" && !restricted_workspace) {
    if (counts->workspace_access < kMaxNormalizedObservationCount) ++counts->workspace_access;
    return true;
  }
  gvisor::syscall::Open final_open;
  *final_open.mutable_context_data() = result.context_data();
  final_open.set_pathname(result.resolved_pathname()); final_open.set_flags(result.flags()); final_open.set_sysno(result.sysno());
  const FilesystemClass filesystem_class = ClassifyFilesystemOpen(
      final_open, *state, profile, &anchor->second);
  if (anchor->second.mount_class == "workspace" && filesystem_class == FilesystemClass::kWorkspace) {
    if (counts->workspace_access < kMaxNormalizedObservationCount) ++counts->workspace_access;
    return true;
  }
  if (filesystem_class == FilesystemClass::kOutside && IsPipxNativeProfile(profile) && state->first_outside_open_diagnostic.empty()) {
    state->first_outside_open_diagnostic = FirstOutsideOpenDiagnostic(final_open,*state,anchor->second);
  }
  switch (filesystem_class) {
    case FilesystemClass::kRuntimeRoot: if (counts->runtime_root_access < kMaxNormalizedObservationCount) ++counts->runtime_root_access; return true;
    case FilesystemClass::kHelperOnly: return true;
    case FilesystemClass::kOutside:
      if (!Send(output, *container_id, "filesystem-outside-workspace")) return false;
      ++counts->immediate_records;
      return true;
    default: {
      state->fault_open_diagnostic = FaultOpenDiagnostic(final_open, *state, anchor->second);
      // Diagnostics reuse the actual runtime predicate; they grant no access
      // and retain no arbitrary pathname or mutable process-name bytes.
      auto named_python = final_open.context_data();
      named_python.set_process_name("python");
      if (IsPythonProfile(profile) &&
          IsPinnedRuntimeRootRead(named_python, *state, final_open.pathname(), final_open.flags(), profile)) {
        detail(FaultSite::kOpenResultClassificationProcessName);
      } else if (HasPrefix(final_open.pathname(), "/proc/")) {
        detail(FaultSite::kOpenResultClassificationProc);
      } else if (HasPrefix(final_open.pathname(), "/sys/")) {
        detail(FaultSite::kOpenResultClassificationSys);
      } else if (anchor->second.mount_class == "oci-root") {
        detail(FaultSite::kOpenResultClassificationImage);
        // Noncryptographic diagnostic locator, never identity/admission
        // authority. Resolve only against the pinned image's data-only file
        // catalogue; collisions remain ambiguous. No pathname is exposed.
        uint64_t locator = 14695981039346656037ULL;
        for (unsigned char byte : final_open.pathname()) {
          locator ^= byte;
          locator *= 1099511628211ULL;
        }
        state->fault_image_locator = locator == 0 ? 1 : locator;
      } else {
        detail(FaultSite::kOpenResultClassificationOther);
      }
      *reason = "STREAM_FAULT"; return false;
    }
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
  // Revoke before the kernel releases the descriptor number. A close exit
  // belongs to an incompatible profile and must not erase a reused socket.
  // Failed closes leave authority revoked rather than restore uncertain state.
  if (message.has_exit()) { *reason = "STREAM_FAULT"; return false; }
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

// The ordinary SocketPair exit message reads a guest return buffer. Only the
// separately patched kernel result may establish descriptor classification.
bool ParseSocketPairEntry(const char* payload, size_t payload_size, int output,
                          std::string* container_id, ProcessState* state, const char** reason) {
  gvisor::syscall::SocketPair message;
  if (!message.ParseFromArray(payload, payload_size) ||
      !ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  if (message.has_exit()) return true;
  const auto& context = message.context_data();
  const auto group = state->groups.find(context.thread_group_id());
  if (!ValidProcessIdentity(context) || context.thread_id() <= 0 || context.thread_start_time_ns() <= 0 ||
      group == state->groups.end() || !SameGroup(group->second, context)) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN"; return false;
  }
  if (message.sysno() != 53 && message.sysno() != 199) { *reason = "FD_STATE_UNKNOWN"; return false; }
  const auto family = ClassifySocketFamily(message.domain());
  if (family == SocketClassification::kUnknown) { *reason = SocketUnknownFamilyReason(message.domain()); return false; }
  const auto key = std::make_pair(context.thread_id(), context.thread_start_time_ns());
  if (state->pending_socketpairs.find(key) != state->pending_socketpairs.end() ||
      state->pending_socketpairs.size() >= kMaxTrackedFileDescriptorsPerGroup) {
    *reason = "FD_STATE_LIMIT"; return false;
  }
  state->pending_socketpairs.emplace(key, ProcessState::PendingSocketPair{context.thread_group_id(),
      context.thread_group_start_time_ns(), message.sysno(), message.domain(), message.type(), message.protocol()});
  if (family == SocketClassification::kNetwork && message.domain() == kLinuxAFPacket) {
    const Attribution attribution{"SOCKET", "PACKET", NetworkProcessRelation(context, *state), "OTHER", nullptr, nullptr};
    return Send(output, *container_id, "network-attempt", nullptr, &attribution);
  }
  return true;
}

bool ParseSocketPairResult(const char* payload, size_t payload_size, std::string* container_id,
                           ProcessState* state, const char** reason) {
  gvisor::syscall::SocketPairResult message;
  if (!message.ParseFromArray(payload, payload_size) ||
      !ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  const auto& context = message.context_data();
  const auto group = state->groups.find(context.thread_group_id());
  if (!ValidProcessIdentity(context) || context.thread_id() <= 0 || context.thread_start_time_ns() <= 0 ||
      group == state->groups.end() || !SameGroup(group->second, context)) {
    *reason = "PROCESS_PROVENANCE_UNKNOWN"; return false;
  }
  const auto key = std::make_pair(context.thread_id(), context.thread_start_time_ns());
  const auto pending = state->pending_socketpairs.find(key);
  if (pending == state->pending_socketpairs.end() ||
      pending->second.thread_group_id != context.thread_group_id() ||
      pending->second.thread_group_start_time_ns != context.thread_group_start_time_ns() ||
      pending->second.sysno != message.sysno() || pending->second.domain != message.domain() ||
      pending->second.type != message.type() || pending->second.protocol != message.protocol()) {
    *reason = "FD_STATE_UNKNOWN"; return false;
  }
  if (!message.success()) {
    if (message.errorno() <= 0 || message.errorno() > 4095 || message.socket1() != -1 || message.socket2() != -1) {
      *reason = "FD_STATE_UNKNOWN"; return false;
    }
    state->pending_socketpairs.erase(pending);
    return true;
  }
  if (message.errorno() != 0 || message.socket1() < 0 || message.socket2() < 0 ||
      message.socket1() == message.socket2() || (message.type() & ~(0xf | SOCK_CLOEXEC | SOCK_NONBLOCK)) != 0) {
    *reason = "FD_STATE_UNKNOWN"; return false;
  }
  const auto family = ClassifySocketFamily(message.domain());
  if (family == SocketClassification::kUnknown) { *reason = SocketUnknownFamilyReason(message.domain()); return false; }
  auto table = state->fd_states.find(context.thread_group_id());
  if (table == state->fd_states.end()) {
    if (state->fd_states.size() >= kMaxTrackedProcessGroups) { *reason = "FD_STATE_LIMIT"; return false; }
    table = state->fd_states.emplace(context.thread_group_id(), std::map<int32_t, ProcessState::FDEntry>{}).first;
  }
  // NewFDs allocates two free descriptors. A collision signals stale tracking;
  // never relabel an existing network FD as local.
  if (table->second.find(message.socket1()) != table->second.end() ||
      table->second.find(message.socket2()) != table->second.end()) { *reason = "FD_STATE_UNKNOWN"; return false; }
  if (table->second.size() > kMaxTrackedFileDescriptorsPerGroup - 2) { *reason = "FD_STATE_LIMIT"; return false; }
  const ProcessState::FDEntry descriptor{family, message.domain(), (message.type() & SOCK_CLOEXEC) != 0};
  table->second.emplace(message.socket1(), descriptor);
  table->second.emplace(message.socket2(), descriptor);
  state->pending_socketpairs.erase(pending);
  return true;
}

// Descriptor diagnostics retain bounded scalars and fixed classifications.
// Guest-buffer correlation is labelled and never establishes authority.
std::string FaultRawDiagnostic(const gvisor::syscall::Syscall& message,
                               const ProcessState& state, const char* check) {
  const auto& group = state.groups.find(message.context_data().thread_group_id())->second;
  std::string diagnostic = std::string("{\"check\":\"") + check + "\",\"sysno\":" + std::to_string(message.sysno()) +
      ",\"fd\":" + std::to_string(message.arg1()) +
      ",\"kernel_image\":\"" + DiagnosticImageName(group.diagnostic_image) +
      "\",\"role\":\"" + RoleName(group.role) +
      "\",\"provenance\":\"" + ProvenanceName(group.provenance) +
      "\",\"executable_locator\":" + std::to_string(group.executable_locator);
  if (group.diagnostic_socketpair_seen) {
    const bool match = message.arg1() == static_cast<uint64_t>(group.diagnostic_socketpair_first) ||
        message.arg1() == static_cast<uint64_t>(group.diagnostic_socketpair_second);
    diagnostic += ",\"socketpair_observed\":true,\"socketpair_fd_match\":" + std::string(match ? "true" : "false") +
        ",\"socketpair_domain\":" + std::to_string(group.diagnostic_socketpair_domain) +
        ",\"socketpair_type\":" + std::to_string(group.diagnostic_socketpair_type) +
        ",\"socketpair_source\":\"GUEST_RETURN_BUFFER\"";
  }
  size_t retained = 0;
  for (auto it = state.fd_mutation_diagnostics.rbegin(); it != state.fd_mutation_diagnostics.rend(); ++it) {
    if (it->group_id != message.context_data().thread_group_id() ||
        it->group_start != message.context_data().thread_group_start_time_ns() ||
        (it->fd != -1 && static_cast<uint64_t>(it->fd) != message.arg1())) continue;
    const char* prefix = retained == 0 ? "fd_last" : "fd_previous";
    diagnostic += std::string(",\"") + prefix + "\":\"" + it->operation +
        "\",\"" + prefix + "_family\":" + std::to_string(it->family);
    if (++retained == 2) break;
  }
  return diagnostic + "}";
}

bool ParseSocketPairDiagnostic(const char* payload, size_t payload_size, std::string* container_id,
                               ProcessState* state, const char** reason) {
  gvisor::syscall::SocketPair message;
  if (!message.ParseFromArray(payload, payload_size) ||
      !ValidateContextContainer(message.context_data(), container_id, reason)) return false;
  if (!message.has_exit() || message.exit().errorno() != 0 || message.exit().result() != 0) return true;
  const auto found = state->groups.find(message.context_data().thread_group_id());
  if (found == state->groups.end() || !SameGroup(found->second, message.context_data())) return true;
  auto& group = found->second;
  group.diagnostic_socketpair_seen = true;
  group.diagnostic_socketpair_first = message.socket1();
  group.diagnostic_socketpair_second = message.socket2();
  group.diagnostic_socketpair_domain = message.domain();
  group.diagnostic_socketpair_type = static_cast<uint32_t>(message.type());
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
  auto unknown_fd = [&](const char* check) {
    *reason = "FD_STATE_UNKNOWN";
    state->fault_raw_diagnostic = FaultRawDiagnostic(message, *state, check);
    return false;
  };
  if (message.arg1() > INT_MAX) return unknown_fd("ARGUMENT");
  auto table = state->fd_states.find(message.context_data().thread_group_id());
  if (table == state->fd_states.end()) return unknown_fd("TABLE");
  auto fd = table->second.find(static_cast<int32_t>(message.arg1()));
  if (fd == table->second.end()) return unknown_fd("DESCRIPTOR");
  if (fd->second.family == SocketClassification::kLocal || fd->second.family == SocketClassification::kSpecialKernelLocal) return true;
  if (fd->second.family != SocketClassification::kNetwork) return unknown_fd("CLASSIFICATION");
  const char* family = FamilyName(fd->second.family, fd->second.raw_family);
  if (family == nullptr) return unknown_fd("FAMILY");
  const char* relation = NetworkProcessRelation(message.context_data(), *state);
  const ProcessClass process_class = ProcessClassForPath(message.context_data().process_name(), profile);
  const Attribution attribution{source, family, relation, NetworkProcessClassName(process_class), nullptr, nullptr};
  return Send(output, *container_id, ResolverNetworkKind(message.context_data(), *state, profile), nullptr, &attribution);
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
        return Send(output, *container_id, ResolverNetworkKind(message.context_data(), process_state, profile), nullptr, &attribution); }
    case AF_INET6:
      if (!ValidSocketAddressLength(family, message.address().size())) { *reason = "CONNECT_AF_INET6_INVALID_LENGTH"; return false; }
      { const Attribution attribution{"CONNECT", "INET6", relation, NetworkProcessClassName(process_class), nullptr, nullptr};
        return Send(output, *container_id, ResolverNetworkKind(message.context_data(), process_state, profile), nullptr, &attribution); }
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
    // Exact registered mounts already passed parent, filesystem and flag
    // validation. Check that identity before testing their writable ancestors;
    // an explicitly declared child is not an unregistered shadow mount.
    if (expected_ids.find(actual->mountpoint()) != expected_ids.end()) continue;
    for (const auto& expected : topology->expected) {
      if (expected.mountpoint != "/" && IsAtOrBelowMountpoint(actual->mountpoint(), expected.mountpoint)) {
        *reason = "TOPOLOGY_MISMATCH"; return false;
      }
    }
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
            const char* profile, ProfileRegistration* registration, ProcessState* process_state, NormalizedCounts* counts, TopologyState* topology,
            const char** reason, FaultSite* fault_site = nullptr) {
  auto set_fault_site = [fault_site](FaultSite site) {
    if (fault_site != nullptr) *fault_site = site;
  };
  if (header.dropped_count != 0) {
    *reason = "STREAM_FAULT";
    set_fault_site(FaultSite::kDroppedCount);
    return false;
  }
  CaptureFDMutationDiagnostic(header, payload, payload_size, process_state);
  switch (static_cast<gvisor::common::MessageType>(header.message_type)) {
    case gvisor::common::MESSAGE_CONTAINER_START:
      if (!ParseContainerStart(payload, payload_size, output, container_id, process_state, reason)) {
        set_fault_site(FaultSite::kContainerStart);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SENTRY_CLONE:
      if (!ParseSentryClone(payload, payload_size, output, container_id, process_state, reason)) {
        set_fault_site(FaultSite::kSentryClone);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SENTRY_EXIT_NOTIFY_PARENT:
      if (!ParseSentryExitNotifyParent(payload, payload_size, container_id, process_state, reason, registration)) {
        set_fault_site(FaultSite::kSentryExitNotifyParent);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SENTRY_EXEC:
      if (!ParseSentryProcessAndClassify(payload, payload_size, output, container_id, profile, registration, process_state, reason, topology)) {
        set_fault_site(FaultSite::kSentryExec);
        return false;
      }
      return true;
    // pathname, argv and envv are parsed by protobuf but are deliberately never
    // copied to the HAA envelope. M11-003 supplies the trusted profile required
    // to classify this bounded process fact as expected or unexpected.
    case gvisor::common::MESSAGE_SYSCALL_EXECVE:
      if (!ParseExecSyscallTelemetry<gvisor::syscall::Execve>(payload, payload_size, container_id, reason)) {
        set_fault_site(FaultSite::kExecSyscall);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_OPEN:
      if (!ParseOpenAndSend(payload, payload_size, output, container_id, profile, process_state, counts, reason)) {
        set_fault_site(FaultSite::kOpen);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT:
      set_fault_site(FaultSite::kOpenResult);
      if (!ParseOpenResultAndSend(payload, payload_size, output, container_id, profile, process_state, counts, *topology, reason, fault_site)) {
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_SNAPSHOT:
      if (!process_state->bootstrap_group_set) {
        *reason = "TOPOLOGY_INVALID";
        set_fault_site(FaultSite::kTopologySnapshot);
        return false;
      }
      if (!ParseTopologySnapshot(payload, payload_size, output, container_id, topology, reason)) {
        set_fault_site(FaultSite::kTopologySnapshot);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_MUTATION:
      if (!ParseTopologyMutation(payload, payload_size, container_id, *topology, reason)) {
        set_fault_site(FaultSite::kTopologyMutation);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_CONNECT:
      if (!ParseConnectAndSend(payload, payload_size, output, container_id, profile, *process_state, reason)) {
        set_fault_site(FaultSite::kConnect);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_SOCKET:
      if (!ParseSocketAndTrack<gvisor::syscall::Socket>(payload, payload_size, output, container_id, profile, process_state, reason)) {
        set_fault_site(FaultSite::kSocket);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_RAW:
      if (!ParseRawAndSend(payload, payload_size, output, container_id, profile, process_state, reason)) {
        set_fault_site(FaultSite::kRaw);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR:
      if (!ParseSocketPairEntry(payload, payload_size, output, container_id, process_state, reason) ||
          !ParseSocketPairDiagnostic(payload, payload_size, container_id, process_state, reason)) {
        set_fault_site(FaultSite::kFdTrack);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR_RESULT:
      if (!ParseSocketPairResult(payload, payload_size, container_id, process_state, reason)) {
        set_fault_site(FaultSite::kFdTrack);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_CLOSE:
      if (!ParseCloseAndTrack(payload, payload_size, container_id, process_state, reason)) {
        set_fault_site(FaultSite::kFdTrack);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_DUP:
      if (!ParseDupAndTrack(payload, payload_size, container_id, process_state, reason)) {
        set_fault_site(FaultSite::kFdTrack);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_FCNTL:
      if (!ParseFcntlAndTrack(payload, payload_size, container_id, process_state, reason)) {
        set_fault_site(FaultSite::kFdTrack);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_CLONE:
      if (!ParseCloneAndTrack(payload, payload_size, container_id, process_state, reason)) {
        set_fault_site(FaultSite::kFdTrack);
        return false;
      }
      return true;
    case gvisor::common::MESSAGE_SYSCALL_FORK:
      if (!ParseForkAndTrack(payload, payload_size, container_id, process_state, reason)) {
        set_fault_site(FaultSite::kFdTrack);
        return false;
      }
      return true;
    default:
      *reason = "UNKNOWN_EVENT_KIND";
      set_fault_site(FaultSite::kUnknownMessage);
      return false;
  }
}

bool Handle(const Header& header, const char* payload, size_t payload_size, int output, std::string* container_id,
            const char* profile, ProcessState* process_state, NormalizedCounts* counts, TopologyState* topology,
            const char** reason, FaultSite* fault_site = nullptr) {
  ProfileRegistration no_admissions{profile == nullptr ? "" : profile, {}, std::string(64, '0'), {}};
  return Handle(header, payload, payload_size, output, container_id, profile, &no_admissions,
                process_state, counts, topology, reason, fault_site);
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
  if (argc == 1 || (argc == 2 && strcmp(argv[1], "--help") == 0)) {
    puts("usage: haa_gvisor_observer REMOTE_SEQPACKET_SOCKET HAA_OUTPUT_DGRAM_SOCKET HAA_CONTROL_SEQPACKET_SOCKET [--ready-fd=FD]");
    return 0;
  }
  if (argc == 2 && strcmp(argv[1], "--identity") == 0) {
    printf("gvisor-commit=%s\n", HAA_GVISOR_COMMIT);
    return 0;
  }
  if (argc != 4 && argc != 5) errx(2, "usage: haa_gvisor_observer REMOTE_SEQPACKET_SOCKET HAA_OUTPUT_DGRAM_SOCKET HAA_CONTROL_SEQPACKET_SOCKET [--ready-fd=FD]");
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
  int control = socket(AF_UNIX, SOCK_SEQPACKET, 0);
  if (control < 0) err(1, "socket control");
  sockaddr_un control_address{}; control_address.sun_family = AF_UNIX;
  if (strlen(argv[3]) >= sizeof(control_address.sun_path)) errx(1, "control endpoint too long");
  strcpy(control_address.sun_path, argv[3]);
  gControlPath = argv[3];
  if (bind(control, reinterpret_cast<sockaddr*>(&control_address), sizeof(control_address)) < 0) err(1, "bind control");
  if (listen(control, 64) < 0) err(1, "listen control");
  signal(SIGTERM, CleanupControlSocket);
  signal(SIGINT, CleanupControlSocket);
  std::map<std::string, ProfileRegistration> profiles;
  std::map<int, ControlPeer> control_peers;
  struct RemoteStream {
    std::string container_id;
    ProcessState process_state;
    NormalizedCounts normalized_counts;
    TopologyState topology_state;
    const char* fault_reason = nullptr;
    std::string profile;
    std::string registration_generation;
    size_t normalized_records = 0;
    // Diagnostic counters only; the existing charge and limit stay unchanged.
    size_t close_records = 0, fcntl_records = 0, raw_records = 0, other_records = 0;
    bool fault = false;
  };
  constexpr size_t kMaxConcurrentRemoteStreams = 8;
  std::map<int, std::unique_ptr<RemoteStream>> remote_streams;
  const int output = ConnectDatagram(argv[2]);
  if (ready_fd >= 0) {
    const char ready = 'R';
    if (write(ready_fd, &ready, 1) != 1) err(1, "signal readiness");
    close(ready_fd);
  }
  auto finish_remote = [&](int client, RemoteStream& stream) {
    // A successfully handshaken stream that never acquired a container identity
    // poisons the shared helper: selective cleanup cannot be attributed.
    if (stream.container_id.empty()) errx(1, "accepted remote stream ended before attribution");
    ProfileRegistration* registration = nullptr;
    auto current = profiles.find(stream.container_id);
    if (current != profiles.end() &&
        (stream.registration_generation.empty() || current->second.session_generation == stream.registration_generation)) {
      registration = &current->second;
    }
    if (!stream.fault && registration == nullptr) {
      stream.fault = true;
      stream.fault_reason = "PROFILE_LOOKUP_FAILURE";
      stream.process_state.terminal_fault_site = FaultSite::kProfileLookup;
    }
    bool unresolved_admission = false;
    if (registration != nullptr) {
      for (const auto& admission : registration->pending_admissions) {
        if (admission.state == ProfileRegistration::AdmissionState::kPending) {
          unresolved_admission = true;
          break;
        }
      }
    }
    if (!stream.fault && (!stream.topology_state.sealed ||
        !stream.process_state.pending_sockets.empty() || !stream.process_state.pending_socketpairs.empty() ||
        !stream.process_state.pending_opens.empty() ||
        unresolved_admission)) {
      stream.fault = true;
      if (!stream.topology_state.sealed) {
        stream.fault_reason = "TOPOLOGY_NOT_READY";
        stream.process_state.terminal_fault_site = FaultSite::kUnsealedTopology;
      } else if (!stream.process_state.pending_sockets.empty() || !stream.process_state.pending_socketpairs.empty()) {
        stream.fault_reason = "FD_STATE_UNKNOWN";
        stream.process_state.terminal_fault_site = FaultSite::kPendingSockets;
      } else if (!stream.process_state.pending_opens.empty()) {
        stream.fault_reason = "STREAM_FAULT";
        stream.process_state.terminal_fault_site = FaultSite::kPendingOpens;
      } else {
        stream.fault_reason = "PROCESS_PROVENANCE_UNKNOWN";
        stream.process_state.terminal_fault_site = FaultSite::kSentryExec;
      }
    }
    if (!stream.fault && stream.normalized_counts.workspace_access != 0) {
      const char* profile = stream.profile.empty() ? nullptr : stream.profile.c_str();
      if (stream.normalized_records + stream.normalized_counts.immediate_records == MaximumRecords(profile)) {
        stream.fault = true;
        stream.fault_reason = "EVENT_LIMIT";
        stream.process_state.terminal_fault_site = FaultSite::kEventLimit;
      } else if (!Send(output, stream.container_id, "filesystem-workspace-access", nullptr, nullptr,
                       stream.normalized_counts.workspace_access)) {
        stream.fault = true;
        stream.fault_reason = "STREAM_FAULT";
        stream.process_state.terminal_fault_site = FaultSite::kWorkspaceSend;
      }
    }
    std::string fault_budget;
    if (stream.fault && stream.process_state.terminal_fault_site == FaultSite::kEventLimit) {
      const char* profile = stream.profile.empty() ? nullptr : stream.profile.c_str();
      fault_budget = "{\"charged\":" + std::to_string(stream.normalized_records + stream.normalized_counts.immediate_records) +
          ",\"limit\":" + std::to_string(MaximumRecords(profile)) +
          ",\"close\":" + std::to_string(stream.close_records) +
          ",\"fcntl\":" + std::to_string(stream.fcntl_records) +
          ",\"raw\":" + std::to_string(stream.raw_records) +
          ",\"other\":" + std::to_string(stream.other_records) +
          ",\"workspace\":" + std::to_string(stream.normalized_counts.workspace_access) + "}";
    }
    const std::string fault_open = stream.fault ? FaultOpenWithFirstOutside(stream.process_state,stream.fault_reason,fault_budget) : "";
    Send(output, stream.container_id, stream.fault ? "stream-fault" : "stream-end", stream.fault_reason, nullptr, 0,
         stream.fault ? FaultSiteName(stream.process_state.terminal_fault_site) : nullptr,
         stream.fault ? stream.process_state.fault_image_locator : 0,
         fault_open, fault_budget,
         stream.fault ? stream.process_state.fault_raw_diagnostic : "");
    if (registration != nullptr) profiles.erase(stream.container_id);
    close(client);
  };
  for (;;) {
    std::vector<pollfd> descriptors{{listener, POLLIN, 0}, {control, POLLIN, 0}};
    std::vector<int> peers, remotes;
    for (const auto& peer : control_peers) {
      peers.push_back(peer.first);
      descriptors.push_back(pollfd{peer.first, POLLIN, 0});
    }
    for (const auto& stream : remote_streams) {
      remotes.push_back(stream.first);
      descriptors.push_back(pollfd{stream.first, POLLIN, 0});
    }
    const int ready = poll(descriptors.data(), descriptors.size(), -1);
    if (ready < 0) {
      if (errno == EINTR) continue;
      err(1, "poll observer streams");
    }
    if ((descriptors[1].revents & POLLIN) != 0 && !AcceptControlPeer(control, &control_peers)) {
      errx(1, "accept control");
    }
    for (size_t index = 0; index < peers.size(); ++index) {
      const short events = descriptors[2 + index].revents;
      if (events != 0 && !ServiceControlPeer(peers[index], events, &control_peers, &profiles)) {
        errx(1, "invalid observer control connection");
      }
    }
    for (size_t index = 0; index < remotes.size(); ++index) {
      const int client = remotes[index];
      const short events = descriptors[2 + peers.size() + index].revents;
      if (events == 0) continue;
      RemoteStream& stream = *remote_streams.at(client);
      bool terminal = false;
      if ((events & POLLIN) != 0) {
        char event[kMaxEventSize];
        const ssize_t size = recv(client, event, sizeof(event), MSG_TRUNC);
        if (size <= 0) {
          if (size < 0) {
            stream.fault = true;
            stream.fault_reason = "STREAM_FAULT";
            stream.process_state.terminal_fault_site = FaultSite::kRecvError;
          }
          terminal = true;
        } else if (static_cast<size_t>(size) > sizeof(event) ||
                   static_cast<size_t>(size) < sizeof(Header)) {
          stream.fault = true;
          stream.fault_reason = "STREAM_FAULT";
          stream.process_state.terminal_fault_site = size > static_cast<ssize_t>(sizeof(event)) ?
              FaultSite::kRecvTrunc : FaultSite::kRecvShort;
          terminal = true;
        } else {
          ProfileRegistration* registration = nullptr;
          if (!stream.registration_generation.empty()) {
            auto current = profiles.find(stream.container_id);
            if (current == profiles.end() || current->second.session_generation != stream.registration_generation) {
              stream.fault = true;
              stream.fault_reason = "PROFILE_LOOKUP_FAILURE";
              stream.process_state.terminal_fault_site = FaultSite::kProfileLookup;
              terminal = true;
            } else {
              registration = &current->second;
            }
          }
          if (!terminal && stream.profile.empty() && !stream.container_id.empty()) {
            registration = AwaitProfile(control, stream.container_id, &control_peers, &profiles);
            if (registration == nullptr) {
              stream.fault = true;
              stream.fault_reason = "PROFILE_LOOKUP_FAILURE";
              stream.process_state.terminal_fault_site = FaultSite::kProfileLookup;
              terminal = true;
            } else {
              stream.profile = registration->profile;
              stream.registration_generation = registration->session_generation;
              stream.topology_state.expected = registration->expected;
            }
          }
          const char* profile = stream.profile.empty() ? nullptr : stream.profile.c_str();
          if (!terminal && stream.normalized_records + stream.normalized_counts.immediate_records == MaximumRecords(profile)) {
            stream.fault = true;
            stream.fault_reason = "EVENT_LIMIT";
            stream.process_state.terminal_fault_site = FaultSite::kEventLimit;
            terminal = true;
          }
          Header header{};
          memcpy(&header, event, sizeof(header));
          if (!terminal && (header.header_size < sizeof(Header) || header.header_size > static_cast<uint16_t>(size))) {
            stream.fault = true;
            stream.fault_reason = "STREAM_FAULT";
            stream.process_state.terminal_fault_site = FaultSite::kHeaderSize;
            terminal = true;
          }
          if (!terminal && !Handle(header, event + header.header_size, size - header.header_size, output,
                                   &stream.container_id, profile, registration, &stream.process_state,
                                   &stream.normalized_counts, &stream.topology_state, &stream.fault_reason,
                                   &stream.process_state.terminal_fault_site)) {
            stream.fault = true;
            if (stream.fault_reason == nullptr) stream.fault_reason = "STREAM_FAULT";
            terminal = true;
          }
          if (!terminal && header.message_type != gvisor::common::MESSAGE_SYSCALL_OPEN &&
              header.message_type != gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT) {
            ++stream.normalized_records;
            switch (static_cast<gvisor::common::MessageType>(header.message_type)) {
              case gvisor::common::MESSAGE_SYSCALL_CLOSE: ++stream.close_records; break;
              case gvisor::common::MESSAGE_SYSCALL_FCNTL: ++stream.fcntl_records; break;
              case gvisor::common::MESSAGE_SYSCALL_RAW: ++stream.raw_records; break;
              default: ++stream.other_records; break;
            }
          }
        }
      } else if ((events & (POLLERR | POLLHUP | POLLNVAL)) != 0) {
        terminal = true;
      }
      if (terminal) {
        finish_remote(client, stream);
        remote_streams.erase(client);
      }
    }
    if ((descriptors[0].revents & POLLIN) != 0) {
      const int client = accept(listener, nullptr, nullptr);
      if (client < 0) err(1, "accept remote");
      if (remote_streams.size() >= kMaxConcurrentRemoteStreams) {
        close(client);
        continue;
      }
      char handshake[1024];
      const ssize_t size = recv(client, handshake, sizeof(handshake), MSG_TRUNC);
      gvisor::common::Handshake incoming;
      if (size <= 0 || size > static_cast<ssize_t>(sizeof(handshake)) ||
          !incoming.ParseFromArray(handshake, size) || incoming.version() != kProtocolVersion) {
        close(client);
        continue;
      }
      gvisor::common::Handshake outgoing;
      outgoing.set_version(kProtocolVersion);
      std::string encoded;
      outgoing.SerializeToString(&encoded);
      if (send(client, encoded.data(), encoded.size(), 0) != static_cast<ssize_t>(encoded.size())) {
        close(client);
        continue;
      }
      remote_streams.emplace(client, std::make_unique<RemoteStream>());
    }
  }
}
