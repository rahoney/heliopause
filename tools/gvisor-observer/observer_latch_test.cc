// Exercises the exact pinned remote-sink protobuf framing through the helper.
// It deliberately checks only normalized records and never retains payloads.
#define main observer_main
#include "tools/haa_gvisor_observer/observer.cc"
#undef main

#include <tuple>
#include <chrono>
#include <signal.h>
#include <fstream>
#include <iterator>
#include <sys/time.h>
#include <sys/wait.h>
#include <thread>
#include <type_traits>
#include <utility>

namespace {

constexpr char kFirstID[] = "0123456789abcdef";
constexpr char kSecondID[] = "fedcba9876543210";
constexpr char kThirdID[] = "abcdef0123456789";
constexpr char kFourthID[] = "9876543210fedcba";
constexpr char kFifthID[] = "0011223344556677";
constexpr char kSixthID[] = "7766554433221100";
constexpr char kSeventhID[] = "1122334455667788";
constexpr char kEighthID[] = "2233445566778899";
constexpr char kNinthID[] = "33445566778899aa";
constexpr char kTenthID[] = "445566778899aabb";
constexpr char kEleventhID[] = "5566778899aabbcc";
constexpr char kTwelfthID[] = "66778899aabbccdd";
constexpr char kThirteenthID[] = "778899aabbccddee";
constexpr char kFourteenthID[] = "8899aabbccddeeff";
constexpr char kFifteenthID[] = "99aabbccddeeff00";
constexpr char kSixteenthID[] = "aabbccddeeff0011";
constexpr char kSeventeenthID[] = "bbccddeeff001122";
constexpr char kEighteenthID[] = "ccddee0011223344";
constexpr char kNineteenthID[] = "ddeeff0011223344";
constexpr char kTwentiethID[] = "eeff001122334455";
constexpr char kTwentyFirstID[] = "ff00112233445566";
constexpr char kTwentySecondID[] = "0011223344556678";
constexpr char kTwentyThirdID[] = "1122334455667789";
constexpr char kTwentyFourthID[] = "223344556677889a";
constexpr char kTwentyFifthID[] = "33445566778899ab";
constexpr char kTwentySixthID[] = "445566778899aabc";
constexpr char kTwentySeventhID[] = "5566778899aabbcd";
constexpr char kTwentyEighthID[] = "66778899aabbccde";
constexpr char kTwentyNinthID[] = "778899aabbccdde0";
constexpr char kThirtiethID[] = "8899aabbccddeeff";
constexpr char kThirtyFirstID[] = "99aabbccddeeff01";
constexpr char kThirtySecondID[] = "a1b2c3d4e5f60718";
constexpr char kThirtyThirdID[] = "b2c3d4e5f6071829";
constexpr char kThirtyFourthID[] = "c3d4e5f60718293a";
constexpr char kThirtyFifthID[] = "d4e5f60718293a4b";
constexpr char kThirtySixthID[] = "e5f60718293a4b5c";
constexpr char kThirtySeventhID[] = "f60718293a4b5c6d";
constexpr char kThirtyEighthID[] = "0718293a4b5c6d7e";
constexpr char kThirtyNinthID[] = "18293a4b5c6d7e8f";
constexpr char kFortiethID[] = "293a4b5c6d7e8f90";
constexpr char kFortyFirstID[] = "3a4b5c6d7e8f90a1";
constexpr char kFortySecondID[] = "4b5c6d7e8f90a1b2";
constexpr char kFortyThirdID[] = "5c6d7e8f90a1b2c3";
constexpr char kFortyFourthID[] = "6d7e8f90a1b2c3d4";
constexpr char kFortyFifthID[] = "7e8f90a1b2c3d4e5";
constexpr char kFortySixthID[] = "8f90a1b2c3d4e5f6";
constexpr char kFortySeventhID[] = "90a1b2c3d4e5f607";
constexpr char kFortyEighthID[] = "a1b2c3d4e5f60708";
constexpr char kFortyNinthID[] = "b1b2c3d4e5f60701";
constexpr char kFiftiethID[] = "b1b2c3d4e5f60702";
constexpr char kFiftyFirstID[] = "b1b2c3d4e5f60703";
constexpr char kFiftySecondID[] = "b1b2c3d4e5f60704";
constexpr char kFiftyThirdID[] = "b1b2c3d4e5f60705";
constexpr char kFiftyFourthID[] = "b1b2c3d4e5f60706";
constexpr char kFiftyFifthID[] = "b1b2c3d4e5f60707";
constexpr char kFiftySixthID[] = "b1b2c3d4e5f60708";
constexpr char kFiftySeventhID[] = "b1b2c3d4e5f60709";
constexpr char kFiftyEighthID[] = "b1b2c3d4e5f6070a";
constexpr char kFiftyNinthID[] = "b1b2c3d4e5f6070b";
constexpr char kSixtiethID[] = "b1b2c3d4e5f6070c";
constexpr char kSixtyFirstID[] = "b1b2c3d4e5f6070d";
constexpr char kSixtySecondID[] = "b1b2c3d4e5f6070e";
constexpr char kSixtyThirdID[] = "b1b2c3d4e5f6070f";
constexpr char kSixtyFourthID[] = "b1b2c3d4e5f60710";
constexpr char kSixtyFifthID[] = "c1b2c3d4e5f60701";
constexpr char kSixtySixthID[] = "d1b2c3d4e5f60701";

bool SendAll(int fd, const std::string& value) {
  return send(fd, value.data(), value.size(), 0) == static_cast<ssize_t>(value.size());
}

bool ExpectRecordExact(int output, const char* container_id, const char* kind, const char* reason = nullptr, const char* wanted_site = nullptr);
bool ExpectRecord(int output, const char* container_id, const char* kind, const char* reason = nullptr);

int BindDatagram(const std::string& path) {
  int fd = socket(AF_UNIX, SOCK_DGRAM, 0);
  if (fd < 0) return -1;
  sockaddr_un address{}; address.sun_family = AF_UNIX;
  if (path.size() >= sizeof(address.sun_path)) return -1;
  strcpy(address.sun_path, path.c_str());
  if (bind(fd, reinterpret_cast<sockaddr*>(&address), sizeof(address)) < 0) return -1;
  timeval timeout{2, 0};
  return setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout)) == 0 ? fd : -1;
}

void DrainDatagram(int fd) {
  char buffer[2048];
  while (recv(fd, buffer, sizeof(buffer), MSG_DONTWAIT) > 0) {}
}

int ConnectRemote(const std::string& path) {
  int fd = socket(AF_UNIX, SOCK_SEQPACKET, 0);
  if (fd < 0) return -1;
  sockaddr_un address{}; address.sun_family = AF_UNIX;
  if (path.size() >= sizeof(address.sun_path)) return -1;
  strcpy(address.sun_path, path.c_str());
  return connect(fd, reinterpret_cast<sockaddr*>(&address), sizeof(address)) == 0 ? fd : -1;
}

static std::map<std::string, std::string> gRegisteredProfiles;
static std::string gAdmissionControl;
static std::set<std::string> gTestDirectRootAdmissions;
static std::map<std::string, int> gControlClients;
static std::map<std::string, std::string> gActiveTestGenerations;
static uint64_t gNextTestGeneration = 0;

std::string TestAdmissionNonce(int32_t group_id, int64_t start_time_ns) {
  std::string nonce(64, 'a');
  char suffix[17];
  snprintf(suffix, sizeof(suffix), "%08x%08x", static_cast<uint32_t>(group_id),
           static_cast<uint32_t>(start_time_ns));
  nonce.replace(48, 16, suffix);
  return nonce;
}

std::string TestSessionGeneration(const char* container_id) {
  if (container_id != nullptr) {
    const auto active = gActiveTestGenerations.find(container_id);
    if (active != gActiveTestGenerations.end()) return active->second;
  }
  std::string generation(64, 'b');
  if (container_id != nullptr) {
    const std::string id(container_id);
    generation.replace(64 - id.size(), id.size(), id);
  }
  return generation;
}

std::string FreshTestSessionGeneration() {
  std::string generation(64, 'b');
  char suffix[17];
  snprintf(suffix, sizeof(suffix), "%016llx", static_cast<unsigned long long>(++gNextTestGeneration));
  generation.replace(48, 16, suffix);
  return generation;
}

ProfileRegistration TestProfileWithAdmission(const char* profile, const char* mode,
                                             const std::string& nonce) {
  const std::string generation(64, 'b');
  ProfileRegistration registration{profile == nullptr ? "" : profile, {}, generation, -1, {}};
  registration.pending_admissions.push_back(ProfileRegistration::PendingAdmission{generation, mode, nonce});
  return registration;
}

int ConnectControl(const std::string& path) {
  const int fd = ConnectRemote(path);
  if (fd < 0) return -1;
  timeval timeout{2, 0};
  if (setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout)) != 0) { close(fd); return -1; }
  return fd;
}

bool ControlExchange(int fd, const std::string& request, const std::string& expected) {
  if (fd < 0 || !SendAll(fd, request)) return false;
  char reply[kMaxControlRecordBytes];
  const ssize_t size = recv(fd, reply, sizeof(reply), MSG_TRUNC);
  return size == static_cast<ssize_t>(expected.size()) && std::string(reply, static_cast<size_t>(size)) == expected;
}

bool ControlAdmissionOnFD(int fd, const char* container_id, const char* mode,
                          const std::string& generation, const std::string& nonce,
                          const char* operation, const char* status) {
  if (fd < 0 || !ValidContainerID(container_id) || !ValidAdmissionNonce(nonce) ||
      operation == nullptr || status == nullptr ||
      (strcmp(operation, "arm") != 0 && strcmp(operation, "cancel") != 0 &&
       strcmp(operation, "status") != 0 && strcmp(operation, "complete") != 0)) return false;
  const std::string request = std::string("{\"op\":\"") + operation + "\",\"container_id\":\"" + container_id +
      "\",\"session_generation\":\"" + generation + "\",\"mode\":\"" + mode + "\",\"nonce\":\"" + nonce + "\"}";
  const std::string expected = std::string("{\"op\":\"ack\",\"ack_op\":\"") + operation + "\",\"status\":\"" +
      status + "\",\"container_id\":\"" + container_id +
      "\",\"session_generation\":\"" + generation + "\",\"mode\":\"" + mode + "\",\"nonce\":\"" + nonce + "\"}";
  return ControlExchange(fd, request, expected);
}

bool ControlAdmissionWithGeneration(const std::string&, const char* container_id, const char* mode,
                                    const std::string& generation, const std::string& nonce, const char* operation) {
  const auto client = gControlClients.find(container_id == nullptr ? "" : container_id);
  const char* status = strcmp(operation, "arm") == 0 ? "armed" :
      strcmp(operation, "cancel") == 0 ? "cancelled" :
      strcmp(operation, "complete") == 0 ? "completed" : "";
  return client != gControlClients.end() && ControlAdmissionOnFD(client->second, container_id, mode, generation, nonce,
                                                                  operation, status);
}

bool ControlAdmission(const std::string& path, const char* container_id, const char* mode,
                      const std::string& nonce, const char* operation) {
  return ControlAdmissionWithGeneration(path, container_id, mode, TestSessionGeneration(container_id), nonce, operation);
}

bool ArmAdmission(const std::string& path, const char* container_id, const char* mode,
                  const std::string& nonce) {
  return ControlAdmission(path, container_id, mode, nonce, "arm");
}

bool CancelAdmission(const std::string& path, const char* container_id, const char* mode,
                     const std::string& nonce) {
  return ControlAdmission(path, container_id, mode, nonce, "cancel");
}

bool CompleteAdmission(const std::string& path, const char* container_id, const char* mode,
                       const std::string& nonce) {
  return ControlAdmission(path, container_id, mode, nonce, "complete");
}

bool InvalidateProfile(const std::string&, const char* container_id, const std::string& generation) {
  const auto client = gControlClients.find(container_id == nullptr ? "" : container_id);
  if (client == gControlClients.end() || !ValidContainerID(container_id) || !ValidSessionGeneration(generation)) return false;
  const std::string request = std::string("{\"op\":\"invalidate\",\"container_id\":\"") + container_id +
      "\",\"session_generation\":\"" + generation + "\"}";
  const std::string expected = std::string("{\"op\":\"ack\",\"ack_op\":\"invalidate\",\"status\":\"invalidated\",\"container_id\":\"") +
      container_id + "\",\"session_generation\":\"" + generation + "\"}";
  return ControlExchange(client->second, request, expected);
}

bool RegisterProfileWithGeneration(const std::string& path, const char* container_id, const char* profile,
                                   const std::string& generation) {
  gRegisteredProfiles[container_id] = profile;
  const std::string prefix = std::string(container_id) + ":";
  for (auto identity = gTestDirectRootAdmissions.begin(); identity != gTestDirectRootAdmissions.end();) {
    if (identity->compare(0, prefix.size(), prefix) == 0) identity = gTestDirectRootAdmissions.erase(identity);
    else ++identity;
  }
  auto prior = gControlClients.find(container_id);
  if (prior != gControlClients.end()) { close(prior->second); gControlClients.erase(prior); }
  const int fd = ConnectControl(path);
  if (fd < 0) return false;
  std::string topology = "/|oci-root|/||1|0|0|0;/tmp|workspace|/|tmpfs|0|1|1|0;/haa-runtime|helper|/|tmpfs|0|0|1|0";
  if (strcmp(profile, kProfilePyPI) == 0 || strcmp(profile, kProfilePyTorchCPU) == 0 ||
      strcmp(profile, kProfilePyTorchCU126) == 0 || strcmp(profile, kProfilePyTorchCU130) == 0 ||
      strcmp(profile, kProfilePyTorchCU132) == 0) {
    topology += ";/haa-site|workspace|/|tmpfs|0|0|1|0";
  } else if (IsPipxNativeProfile(profile)) {
    topology = "/|oci-root|/||1|0|0|0;/tmp|workspace|/|tmpfs|0|1|1|0;/haa-runtime|helper|/|9p|0|0|0|0;/haa-pipx|workspace|/|9p|1|0|0|0;/haa-wheels|workspace|/|9p|1|0|0|0;/haa-site|workspace|/|9p|0|0|0|0";
  } else if (IsPipxRuntimeProfile(profile)) {
    topology += ";/haa-pipx|workspace|/|9p|1|0|0|0";
  } else if (strcmp(profile, kProfileGitHub) == 0 || strcmp(profile, "terraform-provider") == 0) {
    topology += ";/work|workspace|/|tmpfs|0|0|1|0";
  }
  const std::string body = std::string("{\"op\":\"profile\",\"container_id\":\"") + container_id + "\",\"profile\":\"" + profile +
      "\",\"expected_topology\":\"" + topology + "\",\"session_generation\":\"" + generation + "\"}";
  const std::string expected = std::string("{\"op\":\"ack\",\"ack_op\":\"profile\",\"status\":\"registered\",\"container_id\":\"") + container_id +
      "\",\"profile\":\"" + profile + "\",\"expected_topology\":\"" + topology + "\",\"session_generation\":\"" + generation + "\"}";
  if (!ControlExchange(fd, body, expected)) { close(fd); return false; }
  gControlClients[container_id] = fd;
  gActiveTestGenerations[container_id] = generation;
  return true;
}

bool RegisterProfile(const std::string& path, const char* container_id, const char* profile) {
  return RegisterProfileWithGeneration(path, container_id, profile, FreshTestSessionGeneration());
}

gvisor::sentry::MountTopologySnapshot BuildCanonicalTopologySnapshot(const char* container_id, const char* profile) {
  gvisor::sentry::MountTopologySnapshot snapshot;
  auto* context = snapshot.mutable_context_data();
  context->set_container_id(container_id);
  context->set_thread_group_id(1);
  context->set_thread_group_start_time_ns(1);
  context->set_parent_thread_group_id(0);
  snapshot.set_mount_namespace_id(1);
  snapshot.set_snapshot_complete(true);
  auto add_mount = [&snapshot](uint64_t id, const char* mountpoint, const char* fs, bool read_only, bool noexec) {
    auto* mount = snapshot.add_mounts();
    mount->set_mount_id(id);
    mount->set_parent_mount_id(id == 1 ? 1 : 1);
    mount->set_mountpoint(mountpoint);
    mount->set_filesystem_type(fs);
    mount->set_read_only(read_only);
    mount->set_noexec(noexec);
    mount->set_nosuid(id != 1);
    mount->set_nodev(false);
  };
  add_mount(1, "/", "", true, false);
  add_mount(2, "/tmp", "tmpfs", false, true);
  add_mount(3, "/haa-runtime", "tmpfs", false, false);
  uint64_t next_id = 4;
  if (strcmp(profile, kProfilePyPI) == 0 || strcmp(profile, kProfilePyTorchCPU) == 0 ||
      strcmp(profile, kProfilePyTorchCU126) == 0 || strcmp(profile, kProfilePyTorchCU130) == 0 ||
      strcmp(profile, kProfilePyTorchCU132) == 0) {
    add_mount(next_id++, "/haa-site", "tmpfs", false, false);
  } else if (IsPipxRuntimeProfile(profile)) {
    add_mount(next_id++, "/haa-pipx", "9p", true, false);
  } else if (strcmp(profile, kProfileGitHub) == 0 || strcmp(profile, "terraform-provider") == 0) {
    add_mount(next_id++, "/work", "tmpfs", false, false);
  }
  return snapshot;
}

bool Handshake(int client) {
  gvisor::common::Handshake incoming;
  incoming.set_version(kProtocolVersion);
  std::string encoded;
  if (!incoming.SerializeToString(&encoded) || !SendAll(client, encoded)) return false;
  char reply[1024]; const ssize_t size = recv(client, reply, sizeof(reply), 0);
  gvisor::common::Handshake outgoing;
  return size > 0 && outgoing.ParseFromArray(reply, size) && outgoing.version() == kProtocolVersion;
}

template <typename Message, typename std::enable_if<!std::is_same<Message, gvisor::sentry::ExecveInfo>::value, int>::type = 0>
bool SendEvent(int client, gvisor::common::MessageType type, const Message& message, uint32_t dropped_count = 0) {
  std::string payload;
  if (!message.SerializeToString(&payload)) return false;
  Header header{static_cast<uint16_t>(sizeof(Header)), static_cast<uint16_t>(type), dropped_count};
  std::string packet(reinterpret_cast<const char*>(&header), sizeof(header));
  packet.append(payload);
  return SendAll(client, packet);
}

// The production launcher arms before it invokes Docker exec. The synthetic
// transport fixture mirrors that path for valid direct-root boundaries while
// retaining the generic serializer for explicit missing-admission negatives.
bool SendEvent(int client, gvisor::common::MessageType type, const gvisor::sentry::ExecveInfo& original,
               uint32_t dropped_count = 0) {
  gvisor::sentry::ExecveInfo message = original;
  const BoundaryMode mode = type == gvisor::common::MESSAGE_SENTRY_EXEC ? BoundaryInvocation(message) : BoundaryMode::kNone;
  if (message.context_data().is_exec_session() && message.context_data().parent_thread_group_id() == 0 &&
      (mode == BoundaryMode::kLaunch || mode == BoundaryMode::kPythonHandoff || mode == BoundaryMode::kELFHandoff)) {
    const std::string identity = message.context_data().container_id() + ":" +
        std::to_string(message.context_data().thread_group_id()) + ":" +
        std::to_string(message.context_data().thread_group_start_time_ns());
    if (!gTestDirectRootAdmissions.insert(identity).second) {
      std::string payload;
      if (!message.SerializeToString(&payload)) return false;
      Header header{static_cast<uint16_t>(sizeof(Header)), static_cast<uint16_t>(type), dropped_count};
      std::string packet(reinterpret_cast<const char*>(&header), sizeof(header));
      packet.append(payload);
      return SendAll(client, packet);
    }
    const char* mode_name = BoundaryModeName(mode);
    int mode_index = -1;
    for (int index = 0; index + 1 < message.argv_size(); ++index) {
      if (message.argv(index) == kBoundaryHelperPath && message.argv(index + 1) == mode_name) {
        mode_index = index;
        break;
      }
    }
    if (mode_index < 0) return false;
    if (mode_index + 2 >= message.argv_size() || !ValidAdmissionNonce(message.argv(mode_index + 2))) {
      const std::string nonce = TestAdmissionNonce(message.context_data().thread_group_id(),
                                                   message.context_data().thread_group_start_time_ns());
      if (!ArmAdmission(gAdmissionControl, message.context_data().container_id().c_str(), mode_name, nonce)) return false;
      std::vector<std::string> argv;
      for (int index = 0; index < message.argv_size(); ++index) argv.push_back(message.argv(index));
      message.clear_argv();
      for (int index = 0; index <= mode_index + 1; ++index) message.add_argv(argv[index]);
      message.add_argv(nonce);
      for (size_t index = static_cast<size_t>(mode_index + 2); index < argv.size(); ++index) message.add_argv(argv[index]);
    }
  }
  std::string payload;
  if (!message.SerializeToString(&payload)) return false;
  Header header{static_cast<uint16_t>(sizeof(Header)), static_cast<uint16_t>(type), dropped_count};
  std::string packet(reinterpret_cast<const char*>(&header), sizeof(header));
  packet.append(payload);
  return SendAll(client, packet);
}

bool SendEvent(int client, gvisor::common::MessageType type,
               const gvisor::container::Start& original, uint32_t dropped_count = 0,
               bool send_snapshot = true) {
  gvisor::container::Start message = original;
  auto* context = message.mutable_context_data();
  if (context->thread_group_id() == 0) context->set_thread_group_id(1);
  if (context->thread_group_start_time_ns() == 0) context->set_thread_group_start_time_ns(1);
  context->set_parent_thread_group_id(0);
  if (!SendEvent<gvisor::container::Start>(client, type, message, dropped_count)) return false;
  if (type != gvisor::common::MESSAGE_CONTAINER_START || !send_snapshot) return true;
  std::string profile = kProfilePyPI;
  auto it = gRegisteredProfiles.find(context->container_id());
  if (it != gRegisteredProfiles.end()) {
    profile = it->second;
  }
  gvisor::sentry::MountTopologySnapshot snapshot = BuildCanonicalTopologySnapshot(context->container_id().c_str(), profile.c_str());
  return SendEvent(client, gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_SNAPSHOT, snapshot);
}

bool SendEvent(int client, gvisor::common::MessageType type,
               const gvisor::syscall::Open& original, uint32_t dropped_count = 0) {
  gvisor::syscall::Open message = original;
  auto* context = message.mutable_context_data();
  if (context->thread_id() == 0) context->set_thread_id(context->thread_group_id() == 0 ? 1 : context->thread_group_id());
  if (context->thread_start_time_ns() == 0) context->set_thread_start_time_ns(context->thread_group_start_time_ns() == 0 ? 1 : context->thread_group_start_time_ns());
  if (!SendEvent<gvisor::syscall::Open>(client, type, message, dropped_count) || type != gvisor::common::MESSAGE_SYSCALL_OPEN) return false;
  gvisor::syscall::OpenResult result;
  *result.mutable_context_data() = message.context_data(); result.set_sysno(message.sysno()); result.set_flags(message.flags()); result.set_success(true);
  std::string path = message.pathname(); if (path.empty() || path[0] != '/') path = "/tmp/" + (path.empty() ? "relative" : path);
  result.set_resolved_pathname(path);
  uint64_t mount = 1;
  if (path == "/tmp" || path.compare(0, 5, "/tmp/") == 0) mount = 2;
  else if (path == "/haa-runtime" || path.compare(0, 13, "/haa-runtime/") == 0) mount = 3;
  else if (path == "/haa-site" || path.compare(0, 10, "/haa-site/") == 0) mount = 4;
  else if (path == "/work" || path.compare(0, 6, "/work/") == 0) mount = 4;
  result.set_mount_id(mount);
  return SendEvent(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, result);
}

bool SendSuccessfulExec(int client, const gvisor::syscall::Execve& enter,
                        const gvisor::sentry::ExecveInfo& sentry) {
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, enter)) return false;
  gvisor::syscall::Execve exit = enter;
  exit.mutable_exit()->set_result(0);
  exit.mutable_exit()->set_errorno(0);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, exit)) return false;
  return SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, sentry);
}

bool SendDirectLaunchExec(int output, int client, const gvisor::syscall::Execve& enter,
                          const gvisor::sentry::ExecveInfo& target) {
  gvisor::sentry::ExecveInfo launcher = target;
  launcher.set_binary_path(kBoundaryHelperPath);
  launcher.set_execfn(kBoundaryHelperPath);
  launcher.clear_argv();
  launcher.add_argv(kBoundaryHelperPath);
  launcher.add_argv(kLaunchMode);
  if (!SendSuccessfulExec(client, enter, launcher) ||
      !ExpectRecord(output, enter.context_data().container_id().c_str(), "process-exec-expected")) return false;
  gvisor::sentry::ExecveInfo demotion = target;
  demotion.set_binary_path(kSetprivPath);
  demotion.set_execfn(kSetprivPath);
  demotion.clear_argv();
  demotion.add_argv(kSetprivPath);
  demotion.add_argv("--reuid=1000");
  demotion.add_argv("--regid=1000");
  demotion.add_argv("--clear-groups");
  demotion.add_argv("--inh-caps=-all");
  demotion.add_argv("--ambient-caps=-all");
  demotion.add_argv("--bounding-set=-all");
  demotion.add_argv("--no-new-privs");
  demotion.add_argv("--");
  demotion.add_argv(target.binary_path());
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, demotion) ||
      !ExpectRecord(output, enter.context_data().container_id().c_str(), "process-exec-expected")) return false;
  return SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, target);
}

bool SendRealDirectExecTransition(int output, int client, const char* container_id, const char* mode,
                                  const std::string& nonce, int32_t thread_group_id, int64_t start_time_ns) {
  gvisor::common::ContextData context;
  context.set_container_id(container_id);
  context.set_thread_group_id(thread_group_id);
  context.set_thread_group_start_time_ns(start_time_ns);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("haa-boundary");
  gvisor::sentry::ExecveInfo boundary;
  *boundary.mutable_context_data() = context;
  boundary.set_binary_path(kBoundaryHelperPath);
  boundary.set_execfn(kBoundaryHelperPath);
  boundary.add_argv(kBoundaryHelperPath);
  boundary.add_argv(mode);
  boundary.add_argv(nonce);
  boundary.add_argv("/bin/true");
  std::string payload;
  if (!boundary.SerializeToString(&payload)) return false;
  Header header{static_cast<uint16_t>(sizeof(Header)), static_cast<uint16_t>(gvisor::common::MESSAGE_SENTRY_EXEC), 0};
  std::string packet(reinterpret_cast<const char*>(&header), sizeof(header));
  packet.append(payload);
  if (!SendAll(client, packet)) return false;
  return ExpectRecord(output, container_id, "process-exec-expected");
}

bool SendExactSetprivDemotion(int output, int client, const char* container_id,
                              const gvisor::sentry::ExecveInfo& target,
                              const char* expected_kind = "process-exec-expected",
                              const char* expected_reason = nullptr) {
  gvisor::sentry::ExecveInfo demotion = target;
  demotion.set_binary_path(kSetprivPath);
  demotion.set_execfn(kSetprivPath);
  demotion.clear_argv();
  demotion.add_argv(kSetprivPath);
  demotion.add_argv("--reuid=1000");
  demotion.add_argv("--regid=1000");
  demotion.add_argv("--clear-groups");
  demotion.add_argv("--inh-caps=-all");
  demotion.add_argv("--ambient-caps=-all");
  demotion.add_argv("--bounding-set=-all");
  demotion.add_argv("--no-new-privs");
  demotion.add_argv("--");
  demotion.add_argv(target.binary_path());
  return SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, demotion) &&
      ExpectRecord(output, container_id, expected_kind, expected_reason);
}

bool SendExactOCIBootstrapDemotion(int output, int client, const char* container_id,
                                   const gvisor::common::ContextData& context) {
  gvisor::sentry::ExecveInfo demotion;
  *demotion.mutable_context_data() = context;
  demotion.set_binary_path(kSetprivPath);
  demotion.set_execfn(kSetprivPath);
  demotion.add_argv(kSetprivPath);
  demotion.add_argv("--reuid=1000");
  demotion.add_argv("--regid=1000");
  demotion.add_argv("--clear-groups");
  demotion.add_argv("--inh-caps=-all");
  demotion.add_argv("--ambient-caps=-all");
  demotion.add_argv("--bounding-set=-all");
  demotion.add_argv("--no-new-privs");
  demotion.add_argv("--");
  demotion.add_argv("/bin/sleep");
  demotion.add_argv("infinity");
  return SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, demotion) &&
      ExpectRecord(output, container_id, "process-exec-expected");
}

gvisor::sentry::ExecveInfo ExactOCIBootstrapShell(
    const gvisor::common::ContextData& context) {
  gvisor::sentry::ExecveInfo shell;
  *shell.mutable_context_data() = context;
  shell.set_binary_path("/usr/bin/dash");
  shell.set_execfn("/bin/sh");
  shell.add_argv("/bin/sh");
  shell.add_argv("-ceu");
  shell.add_argv(kOCIBootstrapCommand);
  return shell;
}

bool SendProcessClone(int output, int client, const char* container_id,
                      int32_t parent_group, int64_t parent_start,
                      int32_t child_group, int64_t child_start) {
  gvisor::sentry::CloneInfo clone;
  clone.mutable_context_data()->set_container_id(container_id);
  clone.mutable_context_data()->set_thread_group_id(parent_group);
  clone.mutable_context_data()->set_thread_group_start_time_ns(parent_start);
  clone.mutable_context_data()->set_parent_thread_group_id(0);
  clone.mutable_context_data()->set_is_exec_session(true);
  clone.set_created_thread_group_id(child_group);
  clone.set_created_thread_start_time_ns(child_start);
  (void)output;
  return SendEvent(client, gvisor::common::MESSAGE_SENTRY_CLONE, clone);
}

bool ExpectRecordExact(int output, const char* container_id, const char* kind, const char* reason, const char* wanted_site) {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  if (size <= 0) return false;
  std::string expected = std::string("{\"container_id\":\"") + container_id + "\",\"kind\":\"" + kind + "\"";
  if (reason != nullptr) expected += ",\"reason\":\"" + std::string(reason) + "\"";
  if (strcmp(kind, "stream-fault") == 0) {
    // The diagnostic extension is mandatory on faults. Preserve exact envelope
    // checks and accept only the fixed enum vocabulary, never arbitrary payloads.
    const std::string actual(buffer, size);
    const char* sites[] = {"NONE", "RECV_TRUNC", "RECV_SHORT", "PROFILE_LOOKUP", "EVENT_LIMIT", "HEADER_SIZE", "DROPPED_COUNT", "CONTAINER_START", "SENTRY_CLONE", "SENTRY_EXIT_NOTIFY_PARENT", "SENTRY_EXEC", "EXEC_SYSCALL", "OPEN", "OPEN_RESULT", "OPEN_RESULT_ENVELOPE", "OPEN_RESULT_CORRELATION", "OPEN_RESULT_FAILURE_FORMAT", "OPEN_RESULT_SUCCESS_FORMAT", "OPEN_RESULT_ANCHOR", "OPEN_RESULT_SHADOW", "OPEN_RESULT_CLASSIFICATION", "OPEN_RESULT_CLASSIFICATION_PROCESS_NAME", "OPEN_RESULT_CLASSIFICATION_PROC", "OPEN_RESULT_CLASSIFICATION_SYS", "OPEN_RESULT_CLASSIFICATION_IMAGE", "OPEN_RESULT_CLASSIFICATION_OTHER", "TOPOLOGY_SNAPSHOT", "TOPOLOGY_MUTATION", "CONNECT", "SOCKET", "RAW", "FD_TRACK", "UNKNOWN_MESSAGE", "RECV_ERROR", "UNSEALED_TOPOLOGY", "PENDING_SOCKETS", "PENDING_OPENS", "WORKSPACE_SEND"};
    for (const char* site : sites) {
      if (wanted_site != nullptr && strcmp(site, wanted_site) != 0) continue;
      if (actual == expected + ",\"fault_site\":\"" + site + "\"}") return true;
      if (strcmp(site, "OPEN_RESULT_CLASSIFICATION_IMAGE") == 0) {
        const std::string prefix = expected + ",\"fault_site\":\"" + site + "\",\"fault_image_locator\":";
        if (actual.compare(0, prefix.size(), prefix) == 0 && actual.back() == '}') {
          const std::string decimal = actual.substr(prefix.size(), actual.size() - prefix.size() - 1);
          uint64_t value = 0;
          bool valid = !decimal.empty() && decimal.size() <= 20;
          for (char digit : decimal) {
            if (digit < '0' || digit > '9' || value > (UINT64_MAX - (digit - '0')) / 10) { valid = false; break; }
            value = value * 10 + (digit - '0');
          }
          if (valid && value != 0) return true;
        }
      }
    }
    return false;
  }
  expected += "}";
  return std::string(buffer, size) == expected;
}

bool ExpectRecord(int output, const char* container_id, const char* kind, const char* reason) {
  if (!ExpectRecordExact(output, container_id, kind, reason)) return false;
  if (strcmp(kind, "container-start") == 0) {
    return ExpectRecordExact(output, container_id, "mount-anchors-ready");
  }
  return true;
}

bool ExpectCountedRecord(int output, const char* container_id, const char* kind, uint64_t count) {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  if (size <= 0) return false;
  const std::string expected = std::string("{\"container_id\":\"") + container_id +
      "\",\"kind\":\"" + kind + "\",\"count\":" + std::to_string(count) + "}";
  return std::string(buffer, size) == expected;
}

bool ExpectNetworkRecord(int output, const char* container_id, const char* source, const char* family,
                         const char* relation = "UNKNOWN", const char* process_class = "OTHER") {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  if (size <= 0) return false;
  const std::string expected = std::string("{\"container_id\":\"") + container_id +
      "\",\"kind\":\"network-attempt\",\"event_source\":\"" + source +
      "\",\"family\":\"" + family + "\",\"process_relation\":\"" + relation +
      "\",\"process_class\":\"" + process_class + "\"}";
  return std::string(buffer, size) == expected;
}

bool ExpectTrustedNetworkRecord(int output, const char* container_id, const char* source, const char* family,
                               const char* relation = "DIRECT_EXEC_SESSION", const char* process_class = "PYTHON") {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  if (size <= 0) return false;
  const std::string expected = std::string("{\"container_id\":\"") + container_id +
      "\",\"kind\":\"trusted-control-network\",\"event_source\":\"" + source +
      "\",\"family\":\"" + family + "\",\"process_relation\":\"" + relation +
      "\",\"process_class\":\"" + process_class + "\"}";
  return std::string(buffer, size) == expected;
}

bool ExpectUnexpectedProcessRecord(int output, const char* container_id, const char* source,
                                   const char* process_class, const char* reason, const char* parent_relation) {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  if (size <= 0) return false;
  const std::string expected = std::string("{\"container_id\":\"") + container_id +
      "\",\"kind\":\"process-exec-unexpected\",\"event_source\":\"" + source +
      "\",\"process_class\":\"" + process_class + "\",\"classification_reason\":\"" + reason +
      "\",\"parent_relation\":\"" + parent_relation + "\"}";
  return std::string(buffer, size) == expected;
}

bool ExpectNoRecord(int output) {
  pollfd descriptor{output, POLLIN, 0};
  return poll(&descriptor, 1, 50) == 0;
}

bool HasPinnedPodInitProfile() {
  std::ifstream input("tools/haa_gvisor_observer/pod-init.json");
  if (!input.is_open()) return false;
  const std::string profile((std::istreambuf_iterator<char>(input)), std::istreambuf_iterator<char>());
  const std::string connect_context = "\"name\": \"syscall/connect/enter\", \"context_fields\": [\"container_id\", \"group_id\", \"thread_group_start_time\", \"parent_thread_group_id\", \"is_exec_session\", \"process_name\"]";
  const std::string socket_context = "\"name\": \"syscall/socket/enter\", \"context_fields\": [\"container_id\", \"group_id\", \"thread_group_start_time\", \"parent_thread_group_id\", \"is_exec_session\", \"process_name\"]";
  const std::string open_context = "\"name\": \"syscall/openat/enter\", \"context_fields\": [\"container_id\", \"group_id\", \"thread_group_start_time\", \"thread_id\", \"task_start_time\", \"parent_thread_group_id\", \"is_exec_session\", \"process_name\"]";
  const std::string result_context = "\"name\": \"syscall/open_result\", \"context_fields\": [\"container_id\", \"group_id\", \"thread_group_start_time\", \"thread_id\", \"task_start_time\", \"parent_thread_group_id\", \"is_exec_session\", \"process_name\"]";
  const std::string pair_entry_context = "\"name\": \"syscall/socketpair/enter\", \"context_fields\": [\"container_id\", \"group_id\", \"thread_group_start_time\", \"thread_id\", \"task_start_time\", \"parent_thread_group_id\", \"is_exec_session\", \"process_name\"]";
  const std::string pair_result_context = "\"name\": \"syscall/socketpair_result\", \"context_fields\": [\"container_id\", \"group_id\", \"thread_group_start_time\", \"thread_id\", \"task_start_time\", \"parent_thread_group_id\", \"is_exec_session\", \"process_name\"]";
  return profile.find("syscall/execve/enter") != std::string::npos &&
      profile.find("syscall/openat/enter") != std::string::npos &&
      profile.find("syscall/connect/enter") != std::string::npos &&
      profile.find("syscall/socket/enter") != std::string::npos &&
      profile.find("syscall/execve/exit") != std::string::npos &&
      profile.find("syscall/execveat/enter") != std::string::npos &&
      profile.find("syscall/execveat/exit") != std::string::npos &&
      profile.find("syscall/socket/exit") != std::string::npos &&
      profile.find("syscall/socketpair/enter") != std::string::npos &&
      profile.find("syscall/socketpair_result") != std::string::npos &&
      profile.find("syscall/socketpair/exit") != std::string::npos &&
      profile.find("syscall/close/enter") != std::string::npos &&
      profile.find("syscall/close/exit") == std::string::npos &&
      profile.find("syscall/dup/exit") != std::string::npos &&
      profile.find("syscall/dup2/exit") != std::string::npos &&
      profile.find("syscall/dup3/exit") != std::string::npos &&
      profile.find("syscall/fcntl/exit") != std::string::npos &&
      profile.find("syscall/clone/exit") != std::string::npos &&
      profile.find("syscall/fork/exit") != std::string::npos &&
      profile.find("syscall/sysno/44/enter") != std::string::npos &&
      profile.find("syscall/sysno/46/enter") != std::string::npos &&
      profile.find("syscall/sysno/307/enter") != std::string::npos &&
      profile.find("syscall/sysno/436/enter") != std::string::npos &&
      profile.find("thread_group_start_time") != std::string::npos &&
      profile.find("parent_thread_group_id") != std::string::npos &&
      profile.find("is_exec_session") != std::string::npos &&
      profile.find("\"group_id\"") != std::string::npos &&
      profile.find("\"process_name\"") != std::string::npos &&
      profile.find(connect_context) != std::string::npos && profile.find(socket_context) != std::string::npos &&
      profile.find(open_context) != std::string::npos && profile.find(result_context) != std::string::npos &&
      profile.find(pair_entry_context) != std::string::npos && profile.find(pair_result_context) != std::string::npos &&
      profile.find("sentry/mount_topology_snapshot") != std::string::npos &&
      profile.find("sentry/mount_topology_mutation") != std::string::npos &&
      profile.find("sentry/exit_notify_parent") != std::string::npos &&
      profile.find("ignore_missing") == std::string::npos;
}

bool HasBoundedProfileRecordLimits() {
  return MaximumRecords(kProfilePyTorchCPU) == 500000 &&
      MaximumRecords(kProfilePyTorchCU126) == 100000 &&
      MaximumRecords(kProfilePyTorchCU130) == 100000 &&
      MaximumRecords(kProfilePyTorchCU132) == 100000 &&
      MaximumRecords(kProfileGoResolver) == 200000 &&
      MaximumRecords(kProfileGoBuild) == 250000 &&
      MaximumRecords(kProfileCargoResolver) == 10000 &&
      MaximumRecords(kProfileNPM) == 10000 &&
      MaximumRecords(kProfileGitHub) == 10000 &&
      MaximumRecords(kProfilePyPI) == 10000 &&
      MaximumRecords("unknown-profile") == 10000 &&
      MaximumRecords(nullptr) == 10000;
}

bool VerifyProfileRecordBoundary(int output, const std::string& remote, const std::string& control, const char* id, const char* profile, size_t limit, bool overflow) {
  if (!RegisterProfile(control, id, profile)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(id);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, id, "container-start")) return false;
  gvisor::syscall::Close close_message;
  auto* context = close_message.mutable_context_data();
  context->set_container_id(id);
  context->set_thread_group_id(1);
  context->set_thread_group_start_time_ns(1);
  close_message.set_fd(7);
  // Start and sealed topology each charge one; retain the original accounting.
  const size_t count = overflow ? limit - 1 : limit - 3;
  for (size_t i = 0; i < count; ++i) {
    if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CLOSE, close_message)) { close(client); return false; }
  }
  shutdown(client, SHUT_WR);
  char data[2048];
  const ssize_t n = recv(output, data, sizeof(data), 0);
  close(client);
  if (n <= 0) return false;
  const std::string result(data, n);
  if (!overflow) {
    const std::string expected = std::string("{\"container_id\":\"") + id + "\",\"kind\":\"stream-end\"}";
    return result == expected;
  }
  const bool rejected = result.find("\"reason\":\"EVENT_LIMIT\"") != std::string::npos &&
      result.find("\"fault_site\":\"EVENT_LIMIT\"") != std::string::npos &&
      result.find("\"charged\":" + std::to_string(limit)) != std::string::npos &&
      result.find("\"limit\":" + std::to_string(limit)) != std::string::npos;
  fprintf(stderr, "profile=%s bounded terminal=%s\n", profile, result.c_str());
  return rejected;
}

bool VerifyPinnedAccessors(int output, const std::string& remote, const std::string& control) {
  if (!RegisterProfile(control, kFirstID, kProfileNPM)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kFirstID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kFirstID, "container-start")) return false;
  gvisor::syscall::Execve execve;
  execve.mutable_context_data()->set_container_id(kFirstID);
  execve.mutable_context_data()->set_thread_group_id(7);
  execve.mutable_context_data()->set_thread_group_start_time_ns(1);
  execve.mutable_context_data()->set_parent_thread_group_id(0);
  execve.mutable_context_data()->set_is_exec_session(true);
  execve.mutable_context_data()->set_process_name("trusted-test-process");
  execve.set_pathname("/usr/local/bin/node");
  execve.set_sysno(kSyscallExecve);
  execve.add_argv("raw-argv-must-not-leave-helper");
  gvisor::sentry::ExecveInfo execResolved;
  *execResolved.mutable_context_data() = execve.context_data();
  execResolved.set_binary_path("/usr/local/bin/node");
  if (!SendDirectLaunchExec(output, client, execve, execResolved) || !ExpectRecord(output, kFirstID, "process-exec-expected")) return false;
  gvisor::syscall::Open open;
  *open.mutable_context_data() = execve.context_data();
  open.set_pathname("/root/.ssh/authorized_keys");
  open.set_flags(1); open.set_mode(0600); open.set_sysno(257);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open) || !ExpectRecord(output, kFirstID, "filesystem-outside-workspace")) return false;
  open.set_pathname("/tmp/npm-cache-entry");
  open.set_flags(0); open.set_mode(0); open.set_sysno(257);
  for (int index = 0; index < 1153; ++index) {
    if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) return false;
  }
  gvisor::syscall::Socket socket;
  socket.mutable_context_data()->set_container_id(kFirstID);
  socket.mutable_context_data()->set_thread_group_id(7);
  socket.mutable_context_data()->set_thread_group_start_time_ns(1);
  socket.mutable_context_data()->set_parent_thread_group_id(0);
  socket.mutable_context_data()->set_is_exec_session(true);
  socket.mutable_context_data()->set_process_name("trusted-test-process");
  socket.set_domain(2); socket.set_type(1); socket.set_protocol(0);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket) || !ExpectNoRecord(output)) return false;
  socket.mutable_exit()->set_result(3);
  socket.mutable_exit()->set_errorno(0);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket)) return false;
  close(client);
  return ExpectCountedRecord(output, kFirstID, "filesystem-workspace-access", 1153) &&
      ExpectRecord(output, kFirstID, "stream-end");
}


bool VerifyCargoResolverRuntimeReadBoundary(const std::string& pathname = "/usr/lib/x86_64-linux-gnu/librt.so.1",
                                          const char* mountpoint = "/", const char* mount_class = "oci-root",
                                          uint64_t flags = 557056) {
  ProcessState accepted;
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID);
  context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("renamed-worker");
  if (!RegisterGroup(&accepted, context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  accepted.groups[99].trusted_control_network_active = true;
  accepted.groups[99].current_image_class = ProcessClass::kCargo;
  accepted.expected_groups[99] = ProcessState::ExpectedGroup{990, ProcessClass::kCargo};
  gvisor::syscall::Open open;
  *open.mutable_context_data() = context;
  open.set_pathname(pathname);
  open.set_flags(flags);
  const MountAnchor image{1, mountpoint, mount_class};
  auto readable = [&](const ProcessState& state, const MountAnchor& anchor) {
    return ClassifyFilesystemOpen(open, state, kProfileCargoResolver, &anchor) == FilesystemClass::kHelperOnly;
  };
  if (!readable(accepted, image)) return false;
  for (int mutation = 0; mutation < 12; ++mutation) {
    ProcessState rejected = accepted;
    auto& group = rejected.groups[99];
    switch (mutation) {
      case 0: group.role = ProcessState::Role::kArtifact; break;
      case 1: group.provenance = ProcessState::Provenance::kCloneChild; break;
      case 2: group.current_image_class = ProcessClass::kUnknown; break;
      case 3: group.root_eligible = true; break;
      case 4: group.root_consumed = false; break;
      case 5: group.trusted_control_network_active = false; break;
      case 6: group.demotion_pending = true; break;
      case 7: group.launch_target_pending = true; break;
      case 8: group.handoff_target_pending = true; break;
      case 9: group.start_time_ns++; break;
      case 10: rejected.expected_groups.clear(); break;
      case 11: rejected.expected_groups[99].process_class = ProcessClass::kGo; break;
    }
    if (readable(rejected, image)) return false;
  }
  if (readable(accepted, MountAnchor{2, "/usr/lib", "system"}) ||
      readable(accepted, MountAnchor{2, "/usr/lib", "oci-root"}) ||
      readable(accepted, MountAnchor{2, "/usr/share/zoneinfo", "oci-root"}) ||
      readable(accepted, MountAnchor{2, "/", "system"}) ||
      ClassifyFilesystemOpen(open, accepted, kProfileCargoResolver) == FilesystemClass::kHelperOnly) return false;
  for (const char* profile : {kProfileGoResolver, kProfileGoBuild, kProfileNPM, kProfilePyPI}) {
    // NPM already permits this CONTROL input. Preserve that existing consumer
    // rather than treating a Cargo fixture as a change to NPM's boundary.
    const bool existing_npm_null = pathname == "/dev/null" && strcmp(profile, kProfileNPM) == 0;
    if ((ClassifyFilesystemOpen(open, accepted, profile, &image) == FilesystemClass::kHelperOnly) != existing_npm_null) return false;
  }
  open.mutable_context_data()->set_parent_thread_group_id(98);
  if (readable(accepted, image)) return false;
  *open.mutable_context_data() = context;
  open.mutable_context_data()->set_is_exec_session(false);
  if (readable(accepted, image)) return false;
  *open.mutable_context_data() = context;
  for (uint64_t flags : {1, 2, 64, 512, 1024}) {
    open.set_flags(flags);
    if (readable(accepted, image)) return false;
  }
  if (pathname == "/proc/sys/vm/overcommit_memory") {
    for (uint64_t other_flags : {0, 32768}) { open.set_flags(other_flags); if (readable(accepted,image)) return false; }
    open.set_flags(flags);
    if (readable(accepted,MountAnchor{2,"/proc/sys","system"}) || readable(accepted,MountAnchor{2,"/proc/sys/vm","workspace"})) return false;
  }
  open.set_flags(flags);
  for (const char* path : {"/usr/lib/x86_64-linux-gnu/librt.so.2", "/usr/lib/evil.so", "/etc/passwd", "/proc/98/maps", "/proc/99/environ", "/proc/99/status", "/proc/sys/vm/drop_caches", "/proc/sys/vm/overcommit_ratio", "/proc/sys/vm/overcommit_memory/", "/dev/random", "/dev/zero", "/dev/tty", "/usr/share/zoneinfo/Europe/London", "/tmp/usr/share/zoneinfo/Etc/UTC", "/usr/share/zoneinfo/Etc/UTC.evil"}) {
    open.set_pathname(path);
    if (readable(accepted, image)) return false;
  }
  return true;
}

bool VerifyCargoRustcQueryBoundary(const std::string& pathname =
    "/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so",
    const char* mountpoint = "/", const char* mount_class = "oci-root",
    bool metadata_query = false, bool explicit_target = false) {
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  TopologyState topology;
  topology.expected.push_back(ExpectedMount{"/", "oci-root", "/", "", true, false, false, false});
  topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
  topology.namespace_id = 10; topology.snapshot_seen = true; topology.sealed = true;
  auto run = [&](const char* scenario) {
    ProcessState state;
    gvisor::common::ContextData parent;
    parent.set_container_id(kFirstID); parent.set_thread_group_id(99);
    parent.set_thread_group_start_time_ns(990); parent.set_parent_thread_group_id(0);
    parent.set_is_exec_session(true); parent.set_process_name("renamed-cargo");
    if (!RegisterGroup(&state, parent, ProcessState::Role::kControl,
                       ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
    state.groups[99].current_image_class = ProcessClass::kCargo;
    state.groups[99].trusted_control_network_active = true;
    state.expected_groups[99] = ProcessState::ExpectedGroup{990, ProcessClass::kCargo};
    gvisor::sentry::CloneInfo clone;
    *clone.mutable_context_data() = parent;
    clone.set_created_thread_group_id(100); clone.set_created_thread_start_time_ns(1000);
    std::string payload, id; const char* reason = nullptr;
    if (!clone.SerializeToString(&payload) || !ParseSentryClone(payload.data(), payload.size(),
          output[0], &id, &state, &reason)) return false;
    gvisor::sentry::ExecveInfo image;
    *image.mutable_context_data() = parent;
    image.mutable_context_data()->set_thread_group_id(100);
    image.mutable_context_data()->set_thread_group_start_time_ns(1000);
    image.mutable_context_data()->set_parent_thread_group_id(99);
    image.mutable_context_data()->set_is_exec_session(false);
    image.mutable_context_data()->set_process_name("renamed-rustc");
    image.set_binary_path(kRustcBinary); image.set_execfn(kRustcBinary);
    image.add_argv(kRustcBinary);
    if (metadata_query) {
      // Literal kernel-observed host query, independent of the production
      // validator. The only alternate target is the same locked GNU target.
      for (const char* argument : {"-", "--crate-name", "___", "--print=file-names"}) image.add_argv(argument);
      if (explicit_target) { image.add_argv("--target"); image.add_argv("x86_64-unknown-linux-gnu"); }
      for (const char* argument : {"--crate-type", "bin", "--crate-type", "rlib", "--crate-type", "dylib",
           "--crate-type", "cdylib", "--crate-type", "staticlib", "--crate-type", "proc-macro",
           "--print=sysroot", "--print=split-debuginfo", "--print=crate-name", "--print=cfg", "-Wwarnings"}) image.add_argv(argument);
    } else image.add_argv("-vV");
    auto mounts = topology;
    const TopologyState* selected = &mounts;
    if (strcmp(scenario, "copied-image") == 0) image.set_binary_path("/tmp/rustc");
    if (strcmp(scenario, "execfn") == 0) image.set_execfn("/tmp/rustc");
    if (strcmp(scenario, "argv0") == 0) image.set_argv(0, "rustc");
    if (strcmp(scenario, "compile") == 0) image.set_argv(1, "/tmp/build.rs");
    if (strcmp(scenario, "extra-arg") == 0) image.add_argv("--extern=evil");
    if (strcmp(scenario, "unsafe-print") == 0) image.set_argv(metadata_query ? 4 : 1, "--print=native-static-libs");
    if (strcmp(scenario, "link-print") == 0) image.set_argv(metadata_query ? 4 : 1, "--print=link-args");
    if (strcmp(scenario, "parent-class") == 0) state.groups[99].current_image_class = ProcessClass::kUnknown;
    if (strcmp(scenario, "parent-role") == 0) state.groups[99].role = ProcessState::Role::kArtifact;
    if (strcmp(scenario, "parent-start") == 0) state.groups[99].start_time_ns++;
    if (strcmp(scenario, "parent-pending") == 0) state.groups[99].handoff_target_pending = true;
    if (strcmp(scenario, "parent-context") == 0) image.mutable_context_data()->set_parent_thread_group_id(98);
    if (strcmp(scenario, "parent-expected") == 0) state.expected_groups.erase(99);
    if (strcmp(scenario, "parent-network") == 0) state.groups[99].trusted_control_network_active = false;
    if (strcmp(scenario, "parent-eligible") == 0) state.groups[99].root_eligible = true;
    if (strcmp(scenario, "parent-unconsumed") == 0) state.groups[99].root_consumed = false;
    if (strcmp(scenario, "parent-expected-class") == 0) state.expected_groups[99].process_class = ProcessClass::kUnknown;
    if (strcmp(scenario, "missing-clone") == 0) {
      state.groups.erase(100);
      if (!RegisterGroup(&state, image.context_data(), ProcessState::Role::kControl,
                         ProcessState::Provenance::kCloneChild, false, true)) return false;
      state.groups[100].clone_creator_group_id = 99;
      state.groups[100].clone_creator_group_start_time_ns = 990;
    }
    if (strcmp(scenario, "unsealed") == 0) mounts.sealed = false;
    if (strcmp(scenario, "writable-root") == 0) mounts.expected[0].read_only = false;
    if (strcmp(scenario, "shadow") == 0) mounts.anchors.emplace(2, MountAnchor{2, "/usr/local/rustup", "workspace"});
    if (strcmp(scenario, "no-topology") == 0) selected = nullptr;
    const char* profile = strcmp(scenario, "other-profile") == 0 ? kProfileGoResolver : kProfileCargoResolver;
    ProfileRegistration registration;
    if (!image.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(),
          output[0], &id, profile, &registration, &state, &reason, selected) ||
        !ExpectRecordExact(output[1], id.c_str(), "process-exec-expected")) return false;
    gvisor::syscall::Open open;
    *open.mutable_context_data() = image.context_data(); open.set_flags(557056);
    open.set_pathname(pathname);
    MountAnchor anchor{1, mountpoint, mount_class};
    auto readable = [&]() { return ClassifyFilesystemOpen(open, state, profile, &anchor) == FilesystemClass::kHelperOnly; };
    const bool normal = strcmp(scenario, "normal") == 0;
    // Neither diagnostic booleans nor mutable comm confer read authority.
    state.groups[100].diagnostic_rustc_version = true;
    state.groups[100].diagnostic_rustc_metadata = true;
    state.groups[100].diagnostic_image_pinned = true;
    if (readable() != normal || IsTrustedControlNetwork(image.context_data(), state)) return false;
    if (!normal) return true;
    for (const char* path : {"/etc/passwd", "/usr/lib/evil.so", "/usr/include/stdio.h",
         "/usr/local/rustup/toolchains/other/lib/librustc_driver-999a121de2f042be.so",
         "/proc/99/maps", "/proc/99/statm", "/proc/99/cgroup", "/proc/99/mountinfo", "/proc/100/stat", "/proc/100/environ", "/proc/sys/vm/drop_caches",
         "/sys/fs/cgroup/cpu/cpu.shares", "/sys/fs/cgroup/cpu/other/cpu.cfs_quota_us"}) {
      open.set_pathname(path); if (readable()) return false;
    }
    open.set_pathname(pathname);
    for (uint64_t flags : {1, 2, 64, 512, 1024}) { open.set_flags(flags); if (readable()) return false; }
    open.set_flags(557056);
    for (const MountAnchor& wrong : {MountAnchor{2, "/usr/local/rustup", "system"},
         MountAnchor{2, "/usr/local/rustup", "oci-root"}, MountAnchor{2, "/", "system"}}) {
      anchor = wrong; if (readable()) return false;
    }
    anchor = MountAnchor{1, mountpoint, mount_class};
    if (ClassifyFilesystemOpen(open, state, profile) == FilesystemClass::kHelperOnly) return false;
    for (int mutation = 0; mutation < 9; ++mutation) {
      const auto accepted = state;
      auto& child = state.groups[100];
      switch (mutation) {
        case 0: child.role = ProcessState::Role::kArtifact; break;
        case 1: child.provenance = ProcessState::Provenance::kDirectExecRoot; break;
        case 2: child.start_time_ns++; break;
        case 3: child.root_eligible = true; break;
        case 4: child.root_consumed = false; break;
        case 5: child.trusted_control_network_active = true; break;
        case 6: child.launch_target_pending = true; break;
        case 7: state.expected_groups.erase(100); break;
        case 8: state.expected_groups[100].process_class = ProcessClass::kGo; break;
      }
      if (readable()) return false;
      state = accepted;
    }
    // A second image transition cannot renew the one-shot query grant.
    if (!image.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(),
          output[0], &id, profile, &registration, &state, &reason, selected) ||
        !ExpectUnexpectedProcessRecord(output[1], id.c_str(), "SENTRY_EXEC", "OTHER", "BOOTSTRAP_ENDED", "TRACKED_GROUP") || readable()) return false;
    *clone.mutable_context_data() = image.context_data();
    clone.set_created_thread_group_id(101); clone.set_created_thread_start_time_ns(1010);
    if (!clone.SerializeToString(&payload) || !ParseSentryClone(payload.data(), payload.size(),
          output[0], &id, &state, &reason)) return false;
    auto descendant = image;
    descendant.mutable_context_data()->set_thread_group_id(101);
    descendant.mutable_context_data()->set_thread_group_start_time_ns(1010);
    descendant.mutable_context_data()->set_parent_thread_group_id(100);
    if (!descendant.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(),
          output[0], &id, profile, &registration, &state, &reason, selected) ||
        !ExpectRecordExact(output[1], id.c_str(), "process-exec-expected")) return false;
    *open.mutable_context_data() = descendant.context_data();
    if (readable() || IsTrustedControlNetwork(descendant.context_data(), state)) return false;
    return true;
  };
  bool ok = run("normal");
  for (const char* scenario : {"copied-image", "execfn", "argv0", "compile", "extra-arg", "unsafe-print", "link-print",
       "parent-class", "parent-role", "parent-start", "parent-pending", "parent-context",
       "parent-expected", "parent-network", "parent-eligible", "parent-unconsumed", "parent-expected-class",
       "missing-clone", "unsealed", "writable-root", "shadow", "no-topology", "other-profile"}) ok = run(scenario) && ok;
  close(output[0]); close(output[1]);
  return ok;
}

bool VerifyPipxRuntimeReadBoundary() {
  if (IsPythonProfile(kProfilePipxRuntime) ||
      ProcessClassForPath("/usr/local/bin/python", kProfilePipxRuntime) != ProcessClass::kPython ||
      ProcessClassForPath("/usr/local/bin/pip", kProfilePipxRuntime) != ProcessClass::kUnknown ||
      ProcessClassForPath("/usr/bin/uname", kProfilePipxRuntime) != ProcessClass::kUnknown ||
      ProcessClassForPath("/usr/local/bin/pip", kProfilePyPI) != ProcessClass::kPip ||
      IsWorkspacePath("/haa-site/module.py", kProfilePipxRuntime) ||
      IsWorkspacePath("/haa-pipx/foreign.zip", kProfilePipxRuntime)) return false;
  ProcessState state;
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID);
  context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("mutable-name");
  if (!RegisterGroup(&state, context, ProcessState::Role::kArtifact,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  state.groups[99].current_image_class = ProcessClass::kPython;
  const auto baseline = state.groups[99];
  gvisor::syscall::Open open;
  *open.mutable_context_data() = context;
  open.set_flags(557056);
  MountAnchor root{1, "/", "oci-root"};
  for (const char* path : {"/usr/local/lib/python3.14/json/__init__.py", "/usr/local/lib/libpython3.14.so.1.0"}) {
    open.set_pathname(path);
    if (ClassifyFilesystemOpen(open, state, kProfilePipxRuntime, &root) != FilesystemClass::kRuntimeRoot) return false;
  }
  for (const char* path : {"/usr/local/bin/pip", "/etc/pip.conf", "/usr/local/lib/python3.14/site-packages/pip/__init__.py", "/usr/local/lib/python3.14/site-packages/foreign.py", "/proc/98/maps", "/haa-site/module.py"}) {
    open.set_pathname(path);
    auto result = ClassifyFilesystemOpen(open, state, kProfilePipxRuntime, &root);
    if (result == FilesystemClass::kRuntimeRoot || result == FilesystemClass::kWorkspace) return false;
  }
  MountAnchor data{4, "/haa-pipx", "workspace"};
  open.set_pathname("/haa-pipx/pipx-runtime.zip");
  if (ClassifyFilesystemOpen(open, state, kProfilePipxRuntime, &data) != FilesystemClass::kWorkspace) return false;
  for (uint64_t flags : {kOpenWriteOnly,kOpenReadWrite,kOpenCreate,kOpenTruncate,kOpenAppend}) {
    open.set_flags(flags);
    if (ClassifyFilesystemOpen(open, state, kProfilePipxRuntime, &data) != FilesystemClass::kOutside) return false;
  }
  open.set_flags(557056);open.set_pathname("/haa-pipx/foreign.zip");
  if (ClassifyFilesystemOpen(open, state, kProfilePipxRuntime, &data) != FilesystemClass::kOutside) return false;
  open.set_pathname("/usr/local/lib/python3.14/json/__init__.py");
  state.groups[99].current_image_class=ProcessClass::kUnknown;
  open.mutable_context_data()->set_process_name("python");
  if (ClassifyFilesystemOpen(open,state,kProfilePipxRuntime,&root)==FilesystemClass::kRuntimeRoot) return false;
  const auto& after=state.groups[99];
  return after.role==baseline.role && after.provenance==baseline.provenance &&
      after.root_eligible==baseline.root_eligible && after.root_consumed==baseline.root_consumed &&
      after.trusted_control_network_active==baseline.trusted_control_network_active;
}

bool VerifyPipxRuntimeReadiness(const char* profile=kProfilePipxRuntime) {
  int output[2];
  if (socketpair(AF_UNIX, SOCK_SEQPACKET, 0, output) != 0) return false;
  bool success = true;
  for (int scenario = 0; scenario < 13; ++scenario) {
    gvisor::sentry::ExecveInfo image;
    auto* context = image.mutable_context_data();
    context->set_container_id(kFirstID); context->set_thread_group_id(99);
    context->set_thread_group_start_time_ns(990); context->set_parent_thread_group_id(0);
    context->set_is_exec_session(true); context->set_process_name("arbitrary-comm");
    image.set_binary_path(kBoundaryHelperPath); image.set_execfn(kBoundaryHelperPath);
    for (const auto& arg : std::vector<std::string>{kBoundaryHelperPath,kLaunchMode,std::string(64,'e'),"/bin/sh","-ceu",kBoundaryReadinessScript}) image.add_argv(arg);
    if (scenario >= 1 && scenario <= 6) image.set_argv(scenario - 1,"foreign");
    if (scenario == 7) image.add_argv("extra");
    if (scenario == 8) image.set_binary_path("/tmp/haa-boundary");
    if (scenario == 9) image.set_execfn("/tmp/haa-boundary");
    if (scenario == 10) image.mutable_context_data()->set_parent_thread_group_id(88);
    if (scenario == 11) image.mutable_context_data()->set_is_exec_session(false);
    if (scenario == 12) image.set_argv(5,std::string(kBoundaryReadinessScript)+"; /usr/local/bin/python -c evil");
    ProfileRegistration registration;
    registration.session_generation=std::string(64,'a');
    registration.pending_admissions.push_back(ProfileRegistration::PendingAdmission{registration.session_generation,kLaunchMode,std::string(64,'e')});
    ProcessState state; std::string payload,id;const char* reason=nullptr;
    if (!image.SerializeToString(&payload)) { success=false; break; }
    const bool accepted=ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason);
    if (accepted != (scenario == 0)) { success=false; break; }
    if (accepted) {
      if (!ExpectRecordExact(output[1],kFirstID,"process-exec-expected") ||
          state.groups[99].role != ProcessState::Role::kControl ||
          !state.groups[99].root_consumed || !state.groups[99].demotion_pending ||
          registration.pending_admissions[0].state != ProfileRegistration::AdmissionState::kConsumed) { success=false; break; }
      // Exercise the actual demotion and target transitions, with kernel-resolved
      // dash distinct from the controller's /bin/sh invocation name.
      auto demotion=image;
      demotion.set_binary_path(kSetprivPath);demotion.set_execfn(kSetprivPath);demotion.clear_argv();
      for(const char* arg:{kSetprivPath,"--reuid=1000","--regid=1000","--clear-groups","--inh-caps=-all","--ambient-caps=-all","--bounding-set=-all","--no-new-privs","--","/bin/sh","-ceu",kBoundaryReadinessScript})demotion.add_argv(arg);
      if(!demotion.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason)||!ExpectRecordExact(output[1],kFirstID,"process-exec-expected")){success=false;break;}
      auto target=image;target.set_binary_path("/usr/bin/dash");target.set_execfn("/bin/sh");target.clear_argv();
      for(const char* arg:{"/bin/sh","-ceu",kBoundaryReadinessScript})target.add_argv(arg);
      for(int mutation=0;mutation<5;++mutation){
        auto bad=target;auto unchanged=state;
        if(mutation==0)bad.set_binary_path("/bin/sh");
        if(mutation==1)bad.set_execfn("/usr/bin/dash");
        if(mutation==2)bad.set_argv(2,"python -c evil");
        if(mutation==3)bad.add_argv("extra");
        if(mutation==4)bad.set_argv(1,"-c");
        reason=nullptr;
        if(!bad.SerializeToString(&payload)||ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&unchanged,&reason)||!unchanged.groups[99].launch_target_pending){success=false;break;}
      }
      if(!success||!target.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason)||!ExpectRecordExact(output[1],kFirstID,"process-exec-expected")||state.groups[99].launch_target_pending){success=false;break;}
    } else if (!state.groups.empty() || registration.pending_admissions[0].state != ProfileRegistration::AdmissionState::kPending) { success=false; break; }
  }
  close(output[0]);close(output[1]);return success;
}

bool VerifyPipxRuntimeAdmission(const std::string& control) {
  constexpr char id[]="7a7a7a7a7a7a7a7a";
  if (!RegisterProfile(control,id,kProfilePipxRuntime)) return false;
  const auto client=gControlClients.find(id);
  if (client==gControlClients.end()) return false;
  const auto generation=TestSessionGeneration(id);
  for (const char* mode : {kELFHandoffMode}) {
    if (!ControlAdmissionOnFD(client->second,id,mode,generation,std::string(64,'b'),"arm","rejected")) return false;
  }
  if (!ArmAdmission(control,id,kLaunchMode,std::string(64,'d')) ||
      !CancelAdmission(control,id,kLaunchMode,std::string(64,'d')) ||
      !VerifyPipxRuntimeReadiness() ||
      !ArmAdmission(control,id,kPythonHandoffMode,std::string(64,'c')) ||
      !CancelAdmission(control,id,kPythonHandoffMode,std::string(64,'c')) ||
      !InvalidateProfile(control,id,generation)) return false;
  close(client->second);gControlClients.erase(client);
  return true;
}

bool VerifyGitHubArtifactRuntimeReadBoundary() {
  ProcessState state;
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID);
  context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("renamed-artifact");
  if (!RegisterGroup(&state, context, ProcessState::Role::kArtifact,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  state.groups[99].current_image_class = ProcessClass::kArtifact;
  const ProcessState baseline = state;
  gvisor::syscall::Open open;
  *open.mutable_context_data() = context;
  open.set_flags(557056);
  MountAnchor anchor{10, "/proc", "system"};
  auto allowed = [&](const char* profile = kProfileGitHub) {
    return ClassifyFilesystemOpen(open, state, profile, &anchor) == FilesystemClass::kRuntimeRoot;
  };
  for (const char* path : {"/proc/99/cgroup", "/proc/99/mountinfo", "/proc/sys/kernel/osrelease", "/proc/99/maps", "/proc/sys/vm/overcommit_memory"}) {
    open.set_pathname(path);
    if (!allowed()) return false;
  }
  for (const char* path : {"/proc/98/cgroup", "/proc/98/maps", "/proc/self/maps", "/proc/99/maps.evil", "/proc/99/mem", "/proc/99/environ",
                           "/proc/sys/kernel/hostname", "/proc/sys/vm/overcommit_memory.evil", "/proc/sys/vm/overcommit_memory/", "/proc/sys/vm/../vm/overcommit_memory", "/proc/sys/vm/overcommit_ratio", "/proc/sys/vm/drop_caches", "/proc/self/cgroup"}) {
    open.set_pathname(path);
    if (allowed()) return false;
  }
  for (const char* path : {"/proc/99/maps", "/proc/sys/vm/overcommit_memory"}) {
    open.set_pathname(path);
    for (uint64_t flags : {uint64_t{0}, kOpenLargefile, uint64_t{557056 | 0400000}, kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenTruncate, kOpenAppend}) {
      open.set_flags(flags); if (allowed()) return false;
    }
  }
  open.set_flags(557056);
  open.set_pathname("/proc/99/cgroup");
  for (uint64_t flags : {kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenTruncate, kOpenAppend}) {
    open.set_flags(flags);
    if (allowed()) return false;
  }
  open.set_flags(557056);
  for (const char* path : {"/proc/99/maps", "/proc/sys/vm/overcommit_memory", "/proc/99/cgroup"}) {
    open.set_pathname(path);
    for (const char* profile : {kProfilePNPMResolver,kProfileUVResolver,kProfileNPM, kProfilePyPI, kProfileGoResolver, kProfileGoBuild,
                                kProfileCargoBuild, kProfileTerraformProvider}) {
      if (allowed(profile)) return false;
    }
    anchor.mountpoint = "/proc/99";
    if (allowed()) return false;
    anchor.mountpoint = "/proc";
    anchor.mount_class = "workspace";
    if (allowed()) return false;
    anchor.mount_class = "system";
    open.mutable_context_data()->set_thread_group_start_time_ns(991);
    if (allowed()) return false;
    *open.mutable_context_data() = context;
    for (int scenario = 0; scenario != 9; ++scenario) {
      state = baseline;
      auto& group = state.groups[99];
      switch (scenario) {
        case 0: group.role = ProcessState::Role::kControl; break;
        case 1: group.current_image_class = ProcessClass::kShell; break;
        case 2: group.provenance = ProcessState::Provenance::kCloneChild; break;
        case 3: group.root_eligible = true; break;
        case 4: group.root_consumed = false; break;
        case 5: group.trusted_control_network_active = true; break;
        case 6: group.demotion_pending = true; break;
        case 7: group.launch_target_pending = true; break;
        case 8: group.handoff_target_pending = true; break;
      }
      if (allowed()) return false;
    }
    state = baseline;
  }
  anchor = MountAnchor{11, "/sys/fs/cgroup/cpu", "system"};
  for (const char* path : {"/sys/fs/cgroup/cpu/cpu.cfs_quota_us", "/sys/fs/cgroup/cpu/cpu.cfs_period_us"}) {
    open.set_pathname(path);
    if (!allowed()) return false;
  }
  open.set_pathname("/sys/fs/cgroup/cpu/cpu.shares");
  if (allowed()) return false;
  anchor = MountAnchor{1, "/", "oci-root"};
  for (const char* path : {"/usr/lib/x86_64-linux-gnu/libm.so.6",
                           "/usr/lib/x86_64-linux-gnu/libstdc++.so.6.0.30",
                           "/lib/x86_64-linux-gnu/libgcc_s.so.1", "/usr/lib/x86_64-linux-gnu/librt.so.1", "/usr/lib/x86_64-linux-gnu/libpthread.so.0", "/usr/lib/x86_64-linux-gnu/libdl.so.2"}) {
    open.set_pathname(path);
    if (!allowed()) return false;
    open.set_flags(kOpenReadWrite);
    if (allowed()) return false;
    open.set_flags(557056);
  }
  for (const char* path : {"/usr/lib/x86_64-linux-gnu/libpthread.so.0.evil", "/usr/lib/x86_64-linux-gnu/libdl.so.2.evil", "/usr/lib/x86_64-linux-gnu/librt.so.1.evil", "/usr/lib/x86_64-linux-gnu/libstdc++.so.6.evil",
                           "/usr/lib/x86_64-linux-gnu/libcrypto.so.3", "/etc/passwd",
                           "/work/libm.so.6", "/home/user/libm.so.6"}) {
    open.set_pathname(path);
    if (allowed()) return false;
  }
  open.set_pathname("/usr/lib/x86_64-linux-gnu/libm.so.6");
  anchor.mountpoint = "/usr/lib";
  if (allowed()) return false;
  anchor.mountpoint = "/";
  anchor.mount_class = "system";
  if (allowed()) return false;
  return state.groups[99].role == ProcessState::Role::kArtifact &&
      state.groups[99].current_image_class == ProcessClass::kArtifact &&
      !state.groups[99].trusted_control_network_active;
}

bool VerifyGoResolverRuntimeReadBoundary() {
  ProcessState state;
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID);
  context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("renamed-worker");
  if (!RegisterGroup(&state, context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  state.groups[99].trusted_control_network_active = true;
  state.groups[99].current_image_class = ProcessClass::kGo;
  state.expected_groups[99] = ProcessState::ExpectedGroup{990, ProcessClass::kGo};
  gvisor::syscall::Open open;
  *open.mutable_context_data() = context;
  open.set_pathname("/proc/99/cgroup");
  open.set_flags(557056);
  MountAnchor system{10, "/proc", "system"};
  auto classify = [&]() { return ClassifyFilesystemOpen(open, state, kProfileGoResolver, &system); };
  const auto before_diagnostic = classify();
  const std::string diagnostic = FaultOpenDiagnostic(open, state, system);
  if (classify() != before_diagnostic ||
      diagnostic.find("\"image\":\"GO\"") == std::string::npos ||
      diagnostic.find("\"subject\":\"PROC_SELF_CGROUP\"") == std::string::npos) return false;
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/proc/99/mountinfo");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/sys/fs/cgroup/cpu/cpu.cfs_quota_us");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  system.mountpoint = "/sys/fs/cgroup/cpu";
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/sys/fs/cgroup/cpu/cpu.cfs_period_us");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/sys/fs/cgroup/cpu/cpu.shares");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  system.mountpoint = "/proc";
  for (const char* path : {"/proc/98/cgroup", "/proc/99/maps", "/proc/99/environ", "/etc/passwd"}) {
    open.set_pathname(path);
    if (classify() == FilesystemClass::kHelperOnly) return false;
  }
  open.set_pathname("/usr/local/go/go.env");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  system = MountAnchor{1, "/", "oci-root"};
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/etc/nsswitch.conf");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  for (const char* path : {"/etc/ssl/certs", "/etc/ssl/certs/ca-certificates.crt",
                           "/usr/share/ca-certificates/mozilla/ISRG_Root_X1.crt",
                           "/usr/share/zoneinfo/Etc/UTC",
                           "/usr/local/go/src", "/usr/local/go/src/encoding",
                           "/usr/local/go/src/encoding/json/decode.go"}) {
    open.set_pathname(path);
    if (classify() != FilesystemClass::kHelperOnly) return false;
    open.set_flags(kOpenWriteOnly);
    if (classify() == FilesystemClass::kHelperOnly) return false;
    open.set_flags(557056);
  }
  open.set_pathname("/etc/ssl/private/secret.key");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/usr/local/go/go.env");
  system = MountAnchor{2, "/usr/local/go", "oci-root"};
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/dev/null");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  system = MountAnchor{3, "/dev", "system"};
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(557057);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(kOpenWriteOnly | kOpenTruncate);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(557056);
  open.set_pathname("/dev/random");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/etc/resolv.conf");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  system = MountAnchor{4, "/etc/resolv.conf", "system"};
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/etc/hosts");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  system = MountAnchor{5, "/etc/hosts", "system"};
  if (classify() != FilesystemClass::kHelperOnly) return false;
  system = MountAnchor{10, "/proc", "system"};
  open.set_pathname("/proc/99/cgroup");
  system.mountpoint = "/proc/99/cgroup";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  system.mountpoint = "/proc";
  open.set_flags(kOpenWriteOnly);
  if (classify() != FilesystemClass::kOutside) return false;
  open.set_flags(557056);
  for (const char* profile : {kProfileNPM, kProfilePyPI, kProfilePyTorchCPU,
                              kProfilePyTorchCU126, kProfilePyTorchCU130,
                              kProfilePyTorchCU132, kProfileGitHub}) {
    if (ClassifyFilesystemOpen(open, state, profile, &system) == FilesystemClass::kHelperOnly) return false;
  }
  auto& group = state.groups[99];
  group.role = ProcessState::Role::kArtifact;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.role = ProcessState::Role::kControl;
  group.provenance = ProcessState::Provenance::kCloneChild;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.provenance = ProcessState::Provenance::kDirectExecRoot;
  group.root_consumed = false;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.root_consumed = true;
  state.expected_groups[99].process_class = ProcessClass::kUnknown;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  state.expected_groups[99].process_class = ProcessClass::kGo;
  state.expected_groups[99].start_time_ns = 991;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  // Go's fixed control-file setup uses the pinned mkdir utility. Its SELinux
  // discovery is metadata; the clone receives no direct-root network grant.
  auto parent_context = context;
  parent_context.set_thread_group_id(110);
  parent_context.set_thread_group_start_time_ns(1100);
  if (!RegisterGroup(&state, parent_context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  auto child_context = context;
  child_context.set_thread_group_id(111);
  child_context.set_thread_group_start_time_ns(1110);
  child_context.set_parent_thread_group_id(110);
  child_context.set_process_name("renamed-mkdir");
  if (!RegisterGroup(&state, child_context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kCloneChild, false, true)) return false;
  auto& child = state.groups[111];
  child.clone_creator_group_id = 110;
  child.clone_creator_group_start_time_ns = 1100;
  state.expected_groups[111] = ProcessState::ExpectedGroup{1110, ProcessClass::kMkdir};
  *open.mutable_context_data() = child_context;
  open.set_pathname("/proc/filesystems");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  child.trusted_control_network_active = true;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  child.trusted_control_network_active = false;
  state.groups[110].start_time_ns = 1101;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  state.groups[110].start_time_ns = 1100;
  child.role = ProcessState::Role::kArtifact;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  child.role = ProcessState::Role::kControl;
  open.set_flags(kOpenWriteOnly);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(557056);
  state.expected_groups[111].process_class = ProcessClass::kUnknown;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  return ProcessClassForPath("/usr/local/go/bin/go", kProfileGoResolver) == ProcessClass::kGo &&
      ProcessClassForPath("go", kProfileGoResolver) == ProcessClass::kUnknown &&
      ProcessClassForPath("/tmp/go", kProfileGoResolver) == ProcessClass::kUnknown &&
      ProcessClassForPath("/usr/local/go/bin/go", kProfilePyPI) == ProcessClass::kUnknown;
}

bool VerifyCargoBuildArtifactSpoofDenied() {
 ProcessState state;gvisor::common::ContextData context;
 context.set_container_id(kFirstID);context.set_thread_group_id(99);context.set_thread_group_start_time_ns(990);context.set_parent_thread_group_id(0);context.set_is_exec_session(true);
 if(!RegisterGroup(&state,context,ProcessState::Role::kArtifact,ProcessState::Provenance::kDirectExecRoot,false,true))return false;
 state.groups[99].current_image_class=ProcessClass::kUnknown;
 gvisor::syscall::Open open;*open.mutable_context_data()=context;open.set_flags(557056);MountAnchor root{1,"/","oci-root"};
 for(const char* name:{"haa-boundary","sh","setpriv","cargo","rustc","other"}){
  open.mutable_context_data()->set_process_name(name);
  for(const char* path:{"/etc/ld.so.cache","/lib/x86_64-linux-gnu/libc.so.6","/proc/99/status","/etc/shadow"}){
   open.set_pathname(path);if(ClassifyFilesystemOpen(open,state,kProfileCargoBuild,&root)==FilesystemClass::kHelperOnly)return false;
  }
 }
 return true;
}

bool VerifyCargoBuildBootstrapReadBoundary() {
 ProcessState state;
 gvisor::common::ContextData context;
 context.set_container_id(kFirstID);context.set_thread_group_id(99);context.set_thread_group_start_time_ns(990);context.set_parent_thread_group_id(0);context.set_is_exec_session(false);context.set_process_name("untrusted-name");
 if(!RegisterGroup(&state,context,ProcessState::Role::kControl,ProcessState::Provenance::kOCIRoot,false,true))return false;
 state.bootstrap_active=true;state.bootstrap_group_set=true;state.bootstrap_group_id=99;state.bootstrap_group_start_time_ns=990;
 auto& group=state.groups[99];group.oci_bootstrap_stage=ProcessState::OCIBootstrapStage::kAwaitingBootstrapShell;
 state.expected_groups[99]=ProcessState::ExpectedGroup{990,ProcessClass::kArtifact};
 gvisor::syscall::Open open;*open.mutable_context_data()=context;open.set_pathname("/etc/ld.so.cache");open.set_flags(557056);MountAnchor root{1,"/","oci-root"};
 auto classify=[&](){return ClassifyFilesystemOpen(open,state,kProfileCargoBuild,&root);};
 if(classify()!=FilesystemClass::kHelperOnly)return false;
 for(const char* name:{"","cargo","sh","setpriv","spoofed"}){open.mutable_context_data()->set_process_name(name);if(classify()!=FilesystemClass::kHelperOnly)return false;}
 for(const char* path:{"/etc/shadow","/proc/99/environ","/usr/local/rustup/credentials"}){open.set_pathname(path);if(classify()==FilesystemClass::kHelperOnly)return false;}
 open.set_pathname("/etc/ld.so.cache");open.set_flags(kOpenWriteOnly);if(classify()==FilesystemClass::kHelperOnly)return false;open.set_flags(557056);
 group.role=ProcessState::Role::kArtifact;if(classify()==FilesystemClass::kHelperOnly)return false;group.role=ProcessState::Role::kControl;
 open.mutable_context_data()->set_thread_group_start_time_ns(991);if(classify()==FilesystemClass::kHelperOnly)return false;open.mutable_context_data()->set_thread_group_start_time_ns(990);
 const auto saved=group;state.groups.erase(99);if(classify()==FilesystemClass::kHelperOnly)return false;state.groups[99]=saved;
 return classify()==FilesystemClass::kHelperOnly && MaximumRecords(kProfileCargoBuild)==10000;
}

bool VerifyGoBuildConfigurationReadBoundary(const char* profile = kProfileGoBuild, bool tar = false) {
  ProcessState state;
  gvisor::common::ContextData parent;
  parent.set_container_id(kFirstID);
  parent.set_thread_group_id(99);
  parent.set_thread_group_start_time_ns(990);
  parent.set_parent_thread_group_id(0);
  parent.set_is_exec_session(true);
  parent.set_process_name("renamed-parent");
  if (!RegisterGroup(&state, parent, ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  state.groups[99].current_image_class = ProcessClass::kShell;
  auto child_context = parent;
  child_context.set_thread_group_id(100);
  child_context.set_thread_group_start_time_ns(1000);
  child_context.set_parent_thread_group_id(99);
  child_context.set_process_name("renamed-child");
  if (!RegisterGroup(&state, child_context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kCloneChild, false, true)) return false;
  auto& child = state.groups[100];
  child.clone_creator_group_id = 99;
  child.clone_creator_group_start_time_ns = 990;
  child.current_image_class = ProcessClassForPath(tar ? "/usr/bin/tar" : "/usr/bin/mkdir", profile);
  state.expected_groups[100] = ProcessState::ExpectedGroup{1000, child.current_image_class};
  gvisor::syscall::Open open;
  *open.mutable_context_data() = child_context;
  open.set_pathname("/proc/filesystems");
  open.set_flags(557056);
  MountAnchor anchor{10, "/proc", "system"};
  auto classify = [&]() { return ClassifyFilesystemOpen(open, state, profile, &anchor); };
  if (classify() != FilesystemClass::kHelperOnly) return false;
  for (const char* path : {"/proc/99/environ", "/proc/100/status", "/etc/shadow"}) {
    open.set_pathname(path);
    if (classify() == FilesystemClass::kHelperOnly) return false;
  }
  open.set_pathname("/proc/filesystems");
  open.set_flags(kOpenWriteOnly);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(557056);
  anchor.mountpoint = "/proc/other";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor.mountpoint = "/proc";
  anchor.mount_class = "oci-root";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor.mount_class = "system";
  child.role = ProcessState::Role::kArtifact;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  child.role = ProcessState::Role::kControl;
  child.trusted_control_network_active = true;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  child.trusted_control_network_active = false;
  child.current_image_class = ProcessClass::kUnknown;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  child.current_image_class = ProcessClassForPath(tar ? "/usr/bin/tar" : "/usr/bin/mkdir", profile);
  state.groups[99].start_time_ns++;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  state.groups[99].start_time_ns--;
  state.groups[99].role = ProcessState::Role::kArtifact;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  state.groups[99].role = ProcessState::Role::kControl;
  state.groups[99].current_image_class = ProcessClass::kGo;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  state.groups[99].current_image_class = ProcessClass::kShell;
  open.mutable_context_data()->set_thread_group_start_time_ns(1001);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.mutable_context_data()->set_thread_group_start_time_ns(1000);
  return classify() == FilesystemClass::kHelperOnly &&
      MaximumRecords(kProfileGoBuild) == 250000 &&
      MaximumRecords(kProfileGoResolver) == 200000;
}

bool VerifyGoBuildHandoffReadBoundary(const char* profile = kProfileGoBuild) {
  const auto target = strcmp(profile, kProfileCargoBuild) == 0 ? ProcessClass::kCargo : ProcessClass::kGo;
  ProcessState state;
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID);
  context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("renamed-handoff");
  if (!RegisterGroup(&state, context, ProcessState::Role::kArtifact,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  auto& group = state.groups[99];
  group.handoff_target_pending = true;
  group.handoff_target_class = target;
  group.current_image_class = ProcessClassForPath("/usr/bin/setpriv", profile);
  gvisor::syscall::Open open;
  *open.mutable_context_data() = context;
  open.set_pathname("/etc/ld.so.cache");
  open.set_flags(557056);
  MountAnchor anchor{1, "/", "oci-root"};
  auto classify = [&]() { return ClassifyFilesystemOpen(open, state, profile, &anchor); };
  if (classify() != FilesystemClass::kHelperOnly) return false;
  for (const char* path : {"/usr/lib/x86_64-linux-gnu/libc.so.6", "/usr/lib/x86_64-linux-gnu/libcap-ng.so.0.0.0", "/etc/nsswitch.conf", "/etc/passwd", "/etc/group"}) {
    open.set_pathname(path);
    if (classify() != FilesystemClass::kHelperOnly) return false;
  }
  for (const char* path : {"/etc/shadow", "/usr/lib/x86_64-linux-gnu/libprivate.so.0", "/etc/ld.so.cache.backup"}) {
    open.set_pathname(path);
    if (classify() == FilesystemClass::kHelperOnly) return false;
  }
  open.set_pathname("/etc/ld.so.cache");
  open.set_flags(kOpenWriteOnly);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(557056);
  group.handoff_target_pending = false;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.handoff_target_pending = true;
  group.trusted_control_network_active = true;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.trusted_control_network_active = false;
  group.root_eligible = true;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.root_eligible = false;
  group.current_image_class = ProcessClassForPath("/tmp/setpriv", profile);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.current_image_class = ProcessClassForPath("/usr/bin/setpriv", profile);
  group.provenance = ProcessState::Provenance::kCloneChild;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.provenance = ProcessState::Provenance::kDirectExecRoot;
  anchor.mountpoint = "/etc/ld.so.cache";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor = MountAnchor{10, "/proc", "system"};
  open.set_pathname("/proc/sys/kernel/cap_last_cap");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/proc/99/status");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/proc/98/status");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/proc/99/status");
  group.role = ProcessState::Role::kControl;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.role = ProcessState::Role::kArtifact;
  group.demotion_pending = true;
  group.current_image_class = ProcessClassForPath("/haa-runtime/haa-boundary", profile);
  anchor = MountAnchor{3, "/haa-runtime", "helper"};
  open.set_pathname("/haa-runtime/haa-boundary");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  anchor = MountAnchor{1, "/", "oci-root"};
  open.set_pathname("/etc/ld.so.cache");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  group.handoff_target_class = ProcessClass::kArtifact;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.handoff_target_class = target;
  open.mutable_context_data()->set_thread_group_start_time_ns(991);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.mutable_context_data()->set_thread_group_start_time_ns(990);
  group.root_consumed = false;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.root_consumed = true;
  group.current_image_class = ProcessClass::kUnknown;
  group.handoff_target_pending = false;
  open.mutable_context_data()->set_process_name("haa-boundary");
  return classify() != FilesystemClass::kHelperOnly;
}

bool VerifyCargoBuildRustcChildBoundary() {
  int output[2]; if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
  auto run = [&](const char* scenario) {
    ProcessState state;
    gvisor::common::ContextData parent;
    parent.set_container_id(kFirstID);parent.set_thread_group_id(99);parent.set_thread_group_start_time_ns(990);
    parent.set_parent_thread_group_id(0);parent.set_is_exec_session(true);parent.set_process_name("untrusted-parent-name");
    if(!RegisterGroup(&state,parent,ProcessState::Role::kArtifact,ProcessState::Provenance::kDirectExecRoot,false,true))return false;
    state.groups[99].current_image_class=ProcessClass::kCargo;
    gvisor::sentry::CloneInfo clone;*clone.mutable_context_data()=parent;
    clone.set_created_thread_group_id(100);clone.set_created_thread_start_time_ns(1000);
    std::string payload,id;const char* reason=nullptr;
    if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
    gvisor::sentry::ExecveInfo image;*image.mutable_context_data()=parent;
    image.mutable_context_data()->set_thread_group_id(100);image.mutable_context_data()->set_thread_group_start_time_ns(1000);
    image.mutable_context_data()->set_parent_thread_group_id(99);image.mutable_context_data()->set_is_exec_session(false);
    image.mutable_context_data()->set_process_name("untrusted-rustc-name");image.set_binary_path(kRustcBinary);image.set_execfn(kRustcBinary);
    image.add_argv(kRustcBinary);image.add_argv("-vV");
    if(strcmp(scenario,"compile")==0){image.set_argv(1,"/tmp/haa-cargo-input/project/src/main.rs");image.add_argv("--out-dir");image.add_argv("/tmp/haa-cargo-target");}
    TopologyState topology;topology.namespace_id=1;topology.snapshot_seen=true;topology.sealed=true;
    topology.expected.push_back(ExpectedMount{"/","oci-root","/","overlay",true,false,true,true});topology.anchors.emplace(1,MountAnchor{1,"/","oci-root"});
    const TopologyState* selected=&topology;
    if(strcmp(scenario,"copy")==0)image.set_binary_path("/tmp/rustc");
    if(strcmp(scenario,"execfn")==0)image.set_execfn("/tmp/rustc");
    if(strcmp(scenario,"argv0")==0)image.set_argv(0,"rustc");
    if(strcmp(scenario,"parent-class")==0)state.groups[99].current_image_class=ProcessClass::kUnknown;
    if(strcmp(scenario,"parent-role")==0)state.groups[99].role=ProcessState::Role::kControl;
    if(strcmp(scenario,"parent-start")==0)state.groups[99].start_time_ns++;
    if(strcmp(scenario,"parent-network")==0)state.groups[99].trusted_control_network_active=true;
    if(strcmp(scenario,"parent-pending")==0)state.groups[99].handoff_target_pending=true;
    if(strcmp(scenario,"parent-context")==0)image.mutable_context_data()->set_parent_thread_group_id(98);
    if(strcmp(scenario,"no-clone")==0)state.groups[100].cargo_rustc_query_candidate=false;
    if(strcmp(scenario,"unsealed")==0)topology.sealed=false;
    if(strcmp(scenario,"writable")==0)topology.expected[0].read_only=false;
    if(strcmp(scenario,"shadow")==0)topology.anchors.emplace(2,MountAnchor{2,"/usr/local/rustup","workspace"});
    if(strcmp(scenario,"no-topology")==0)selected=nullptr;
    const char* profile=strcmp(scenario,"other-profile")==0?kProfileCargoResolver:kProfileCargoBuild;
    ProfileRegistration registration;
    const bool accepted=strcmp(scenario,"normal")==0||strcmp(scenario,"compile")==0;
    if(!image.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,selected))return false;
    if(accepted){if(!ExpectRecordExact(output[1],id.c_str(),"process-exec-expected"))return false;}
    else if(!ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP"))return false;
    gvisor::syscall::Open open;*open.mutable_context_data()=image.context_data();open.set_flags(557056);
    open.set_pathname("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so");
    MountAnchor anchor{1,"/","oci-root"};
    auto readable=[&](){return ClassifyFilesystemOpen(open,state,profile,&anchor)==FilesystemClass::kHelperOnly;};
    if(readable()!=accepted||IsTrustedControlNetwork(image.context_data(),state)||state.groups[100].role!=ProcessState::Role::kArtifact)return false;
    if(!accepted)return true;
    for(const char* path:{"/etc/shadow","/usr/local/rustup/credentials","/proc/99/environ"}){open.set_pathname(path);if(readable())return false;}
    open.set_pathname("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so");
    open.set_flags(kOpenWriteOnly);if(readable())return false;open.set_flags(557056);
    anchor.mountpoint="/usr/local/rustup";if(readable())return false;anchor.mountpoint="/";
    if(!image.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology)||
       !ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP")||readable())return false;
    *clone.mutable_context_data()=image.context_data();clone.set_created_thread_group_id(101);clone.set_created_thread_start_time_ns(1010);
    if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
    auto descendant=image;descendant.mutable_context_data()->set_thread_group_id(101);descendant.mutable_context_data()->set_thread_group_start_time_ns(1010);descendant.mutable_context_data()->set_parent_thread_group_id(100);
    if(!descendant.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology)||
       !ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP"))return false;
    *open.mutable_context_data()=descendant.context_data();return !readable();
  };
  bool ok=true;for(const char* scenario:{"normal","compile","copy","execfn","argv0","parent-class","parent-role","parent-start","parent-network","parent-pending","parent-context","no-clone","unsealed","writable","shadow","no-topology","other-profile"})ok=run(scenario)&&ok;
  close(output[0]);close(output[1]);return ok;
}

bool VerifyCargoBuildProgramBoundary(bool abi=false,bool maps=false,bool null_device=false) {
  int output[2];if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
  auto run=[&](const char* scenario){
    ProcessState state;gvisor::common::ContextData root;
    root.set_container_id(kFirstID);root.set_thread_group_id(99);root.set_thread_group_start_time_ns(990);root.set_parent_thread_group_id(0);root.set_is_exec_session(true);root.set_process_name("arbitrary-root-name");
    if(!RegisterGroup(&state,root,ProcessState::Role::kArtifact,ProcessState::Provenance::kDirectExecRoot,false,true))return false;
    state.groups[99].current_image_class=ProcessClass::kCargo;
    gvisor::sentry::CloneInfo clone;*clone.mutable_context_data()=root;clone.set_created_thread_group_id(100);clone.set_created_thread_start_time_ns(1000);
    std::string payload,id;const char* reason=nullptr;
    if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
    gvisor::sentry::ExecveInfo image;*image.mutable_context_data()=root;image.mutable_context_data()->set_thread_group_id(100);image.mutable_context_data()->set_thread_group_start_time_ns(1000);image.mutable_context_data()->set_parent_thread_group_id(99);image.mutable_context_data()->set_is_exec_session(false);image.mutable_context_data()->set_process_name("sh");
    image.set_binary_path("/tmp/haa-cargo-target/debug/build/derived/build-script-build");image.set_execfn(image.binary_path());image.add_argv("arbitrary-argv0");
    TopologyState topology;topology.namespace_id=1;topology.snapshot_seen=true;topology.sealed=true;
    topology.expected.push_back(ExpectedMount{"/","oci-root","/","overlay",true,false,true,true});
    topology.expected.push_back(ExpectedMount{"/tmp/haa-cargo-target","workspace","/tmp","9p",false,false,false,false});
    topology.anchors.emplace(1,MountAnchor{1,"/","oci-root"});topology.anchors.emplace(2,MountAnchor{2,"/tmp/haa-cargo-target","workspace"});
    if(strcmp(scenario,"copy")==0)image.set_binary_path("/tmp/unowned/program");
    if(strcmp(scenario,"escape")==0)image.set_binary_path("/tmp/haa-cargo-target/../unowned/program");
    if(strcmp(scenario,"parent-class")==0)state.groups[99].current_image_class=ProcessClass::kUnknown;
    if(strcmp(scenario,"parent-role")==0)state.groups[99].role=ProcessState::Role::kControl;
    if(strcmp(scenario,"parent-start")==0)state.groups[99].start_time_ns++;
    if(strcmp(scenario,"parent-network")==0)state.groups[99].trusted_control_network_active=true;
    if(strcmp(scenario,"parent-pending")==0)state.groups[99].handoff_target_pending=true;
    if(strcmp(scenario,"parent-context")==0)image.mutable_context_data()->set_parent_thread_group_id(98);
    if(strcmp(scenario,"child-start")==0)image.mutable_context_data()->set_thread_group_start_time_ns(1001);
    if(strcmp(scenario,"unsealed")==0)topology.sealed=false;
    if(strcmp(scenario,"namespace")==0)topology.namespace_id=0;
    if(strcmp(scenario,"snapshot")==0)topology.snapshot_seen=false;
    if(strcmp(scenario,"root-writable")==0)topology.expected[0].read_only=false;
    if(strcmp(scenario,"target-readonly")==0)topology.expected[1].read_only=true;
    if(strcmp(scenario,"target-noexec")==0)topology.expected[1].noexec=true;
    if(strcmp(scenario,"target-fs")==0)topology.expected[1].filesystem_type="tmpfs";
    if(strcmp(scenario,"missing-declaration")==0)topology.expected.pop_back();
    if(strcmp(scenario,"missing-anchor")==0)topology.anchors.erase(2);
    if(strcmp(scenario,"wrong-anchor")==0)topology.anchors[2].mount_class="system";
    if(strcmp(scenario,"shadow")==0)topology.anchors.emplace(3,MountAnchor{3,"/tmp/haa-cargo-target/debug","workspace"});
    if(strcmp(scenario,"diagnostic-forged")==0){image.set_binary_path("/tmp/unowned/program");state.groups[100].diagnostic_build_target_image=true;}
    const char* profile=strcmp(scenario,"other-profile")==0?kProfileGoBuild:kProfileCargoBuild;
    const bool accepted=strcmp(scenario,"normal")==0;
    ProfileRegistration registration;
    if(!image.SerializeToString(&payload))return false;
    const bool parsed=ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology);
    if(strcmp(scenario,"child-start")==0)return !parsed && reason!=nullptr && strcmp(reason,"PROCESS_IDENTITY_REUSED")==0 && ExpectNoRecord(output[1]);
    if(!parsed)return false;
    if(accepted){if(!ExpectRecordExact(output[1],id.c_str(),"process-exec-expected"))return false;}
    else if(!ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP"))return false;
    gvisor::syscall::Open open;*open.mutable_context_data()=image.context_data();open.set_pathname("/etc/ld.so.cache");open.set_flags(557056);MountAnchor anchor{1,"/","oci-root"};
    auto readable=[&](){return ClassifyFilesystemOpen(open,state,profile,&anchor)==FilesystemClass::kHelperOnly;};
    if(readable()!=accepted || IsTrustedControlNetwork(image.context_data(),state) || state.groups[100].role!=ProcessState::Role::kArtifact)return false;
    if(!accepted)return true;
    if(abi)for(const char* path:{"/usr/lib/x86_64-linux-gnu/libgcc_s.so.1","/usr/lib/x86_64-linux-gnu/libc.so.6","/usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2"}){open.set_pathname(path);if(!readable())return false;}
    for(const char* path:{"/etc/shadow","/usr/local/rustup/credentials","/usr/lib/x86_64-linux-gnu/libdl.so.2","/usr/local/go/src/runtime"}){open.set_pathname(path);if(readable())return false;}
    if(maps){
      anchor={3,"/proc","system"};open.set_pathname("/proc/100/maps");if(!readable())return false;
      for(const char* path:{"/proc/99/maps","/proc/100/environ","/proc/100/mountinfo"}){open.set_pathname(path);if(readable())return false;}
      open.set_pathname("/proc/100/maps");for(uint64_t flags:{0,1,2,64,32768}){open.set_flags(flags);if(readable())return false;}open.set_flags(557056);
      anchor={3,"/","system"};if(readable())return false;
      anchor={1,"/","oci-root"};
    }
    if(null_device){
      anchor={3,"/dev","system"};open.set_pathname("/dev/null");if(!readable())return false;
      for(uint64_t flags:{0,1,2,64,32768}){open.set_flags(flags);if(readable())return false;}open.set_flags(557056);
      for(const char* path:{"/dev/urandom","/dev/random","/dev/zero"}){open.set_pathname(path);if(readable())return false;}
      open.set_pathname("/dev/null");anchor={3,"/","system"};if(readable())return false;anchor={1,"/","oci-root"};
    }
    open.set_pathname("/etc/ld.so.cache");open.set_flags(kOpenWriteOnly);if(readable())return false;open.set_flags(557056);
    *clone.mutable_context_data()=image.context_data();clone.set_created_thread_group_id(101);clone.set_created_thread_start_time_ns(1010);
    if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
    auto descendant=image;descendant.mutable_context_data()->set_thread_group_id(101);descendant.mutable_context_data()->set_thread_group_start_time_ns(1010);descendant.mutable_context_data()->set_parent_thread_group_id(100);
    if(!descendant.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology)||!ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP"))return false;
    *open.mutable_context_data()=descendant.context_data();if(readable())return false;*open.mutable_context_data()=image.context_data();
    if(!image.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology)||!ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP")||readable())return false;
    return true;
  };
  bool ok=true;for(const char* scenario:{"normal","copy","escape","parent-class","parent-role","parent-start","parent-network","parent-pending","parent-context","child-start","unsealed","namespace","snapshot","root-writable","target-readonly","target-noexec","target-fs","missing-declaration","missing-anchor","wrong-anchor","shadow","diagnostic-forged","other-profile"})ok=run(scenario)&&ok;
  close(output[0]);close(output[1]);return ok;
}

bool VerifyCargoBuildProgramRustcQueryBoundary() {
  int output[2];if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
  auto run=[&](const char* scenario){
    ProcessState state;gvisor::common::ContextData root;root.set_container_id(kFirstID);root.set_thread_group_id(99);root.set_thread_group_start_time_ns(990);root.set_is_exec_session(true);
    if(!RegisterGroup(&state,root,ProcessState::Role::kArtifact,ProcessState::Provenance::kDirectExecRoot,false,true))return false;
    state.groups[99].current_image_class=ProcessClass::kCargo;
    std::string payload,id;const char* reason=nullptr;ProfileRegistration registration;
    gvisor::sentry::CloneInfo clone;*clone.mutable_context_data()=root;clone.set_created_thread_group_id(100);clone.set_created_thread_start_time_ns(1000);
    if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
    gvisor::sentry::ExecveInfo program;*program.mutable_context_data()=root;program.mutable_context_data()->set_thread_group_id(100);program.mutable_context_data()->set_thread_group_start_time_ns(1000);program.mutable_context_data()->set_parent_thread_group_id(99);program.mutable_context_data()->set_is_exec_session(false);program.set_binary_path("/tmp/haa-cargo-target/debug/build/derived/build-script-build");program.set_execfn(program.binary_path());program.add_argv("untrusted-name");
    TopologyState topology;topology.namespace_id=1;topology.snapshot_seen=true;topology.sealed=true;topology.expected.push_back(ExpectedMount{"/","oci-root","/","overlay",true,false,true,true});topology.expected.push_back(ExpectedMount{"/tmp/haa-cargo-target","workspace","/tmp","9p",false,false,false,false});topology.anchors.emplace(1,MountAnchor{1,"/","oci-root"});topology.anchors.emplace(2,MountAnchor{2,"/tmp/haa-cargo-target","workspace"});
    if(!program.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,kProfileCargoBuild,&registration,&state,&reason,&topology)||!ExpectRecordExact(output[1],id.c_str(),"process-exec-expected"))return false;
    *clone.mutable_context_data()=program.context_data();clone.set_created_thread_group_id(101);clone.set_created_thread_start_time_ns(1010);
    if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
    gvisor::sentry::ExecveInfo query;*query.mutable_context_data()=program.context_data();query.mutable_context_data()->set_thread_group_id(101);query.mutable_context_data()->set_thread_group_start_time_ns(1010);query.mutable_context_data()->set_parent_thread_group_id(100);query.mutable_context_data()->set_process_name("cargo");query.set_binary_path(kRustcBinary);query.set_execfn(kRustcBinary);query.add_argv(kRustcBinary);query.add_argv("--version");
    if(strcmp(scenario,"compile")==0)query.set_argv(1,"artifact.rs");
    if(strcmp(scenario,"extra-arg")==0)query.add_argv("artifact.rs");
    if(strcmp(scenario,"version-shape")==0)query.set_argv(1,"-vV");
    if(strcmp(scenario,"copy")==0)query.set_binary_path("/tmp/copied-rustc");
    if(strcmp(scenario,"argv0")==0)query.set_argv(0,"rustc");
    if(strcmp(scenario,"execfn")==0)query.set_execfn("/tmp/copied-rustc");
    if(strcmp(scenario,"parent-class")==0)state.groups[100].current_image_class=ProcessClass::kUnknown;
    if(strcmp(scenario,"parent-role")==0)state.groups[100].role=ProcessState::Role::kControl;
    if(strcmp(scenario,"parent-active")==0)state.groups[100].cargo_build_program_active=false;
    if(strcmp(scenario,"parent-start")==0)state.groups[100].start_time_ns++;
    if(strcmp(scenario,"parent-network")==0)state.groups[100].trusted_control_network_active=true;
    if(strcmp(scenario,"ancestor-start")==0)state.groups[99].start_time_ns++;
    if(strcmp(scenario,"ancestor-role")==0)state.groups[99].role=ProcessState::Role::kControl;
    if(strcmp(scenario,"parent-context")==0)query.mutable_context_data()->set_parent_thread_group_id(99);
    if(strcmp(scenario,"unsealed")==0)topology.sealed=false;
    if(strcmp(scenario,"root-writable")==0)topology.expected[0].read_only=false;
    if(strcmp(scenario,"shadow")==0)topology.anchors.emplace(3,MountAnchor{3,"/usr/local/rustup","workspace"});
    if(strcmp(scenario,"diag-only")==0){query.set_argv(1,"artifact.rs");state.groups[101].diagnostic_rustc_version=true;}
    const char* profile=strcmp(scenario,"other-profile")==0?kProfileGoBuild:kProfileCargoBuild;
    const bool expected=strcmp(scenario,"normal")==0;
    if(!query.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology))return false;
    if(expected){if(!ExpectRecordExact(output[1],id.c_str(),"process-exec-expected"))return false;}
    else if(!ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP"))return false;
    gvisor::syscall::Open open;open.set_pathname("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so");open.set_flags(557056);*open.mutable_context_data()=query.context_data();MountAnchor anchor{1,"/","oci-root"};
    auto readable=[&](){return ClassifyFilesystemOpen(open,state,profile,&anchor)==FilesystemClass::kHelperOnly;};
    if(readable()!=expected || IsTrustedControlNetwork(query.context_data(),state) || state.groups[101].cargo_build_rustc_active)return false;
    if(!expected)return true;
    for(const char* path:{"/etc/ld.so.cache","/usr/lib/x86_64-linux-gnu/libgcc_s.so.1","/usr/lib/x86_64-linux-gnu/libz.so.1.2.13","/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/libLLVM.so.23.1-rust-1.99.0-stable"}){open.set_pathname(path);if(!readable())return false;}
    for(const char* path:{"/etc/shadow","/usr/local/rustup/credentials","/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/lib","/usr/lib/x86_64-linux-gnu/libutil.a"}){open.set_pathname(path);if(readable())return false;}
    open.set_pathname("/etc/ld.so.cache");open.set_flags(1);if(readable())return false;open.set_flags(557056);
    anchor={4,"/proc","system"};for(const char* path:{"/proc/101/maps","/proc/101/statm","/proc/sys/vm/overcommit_memory"}){open.set_pathname(path);if(!readable())return false;}
    for(const char* path:{"/proc/100/maps","/proc/101/environ","/proc/101/cgroup"}){open.set_pathname(path);if(readable())return false;}
    anchor={1,"/","oci-root"};open.set_pathname("/etc/ld.so.cache");
    *clone.mutable_context_data()=query.context_data();clone.set_created_thread_group_id(102);clone.set_created_thread_start_time_ns(1020);
    if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
    auto descendant=query;descendant.mutable_context_data()->set_thread_group_id(102);descendant.mutable_context_data()->set_thread_group_start_time_ns(1020);descendant.mutable_context_data()->set_parent_thread_group_id(101);
    if(!descendant.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology)||!ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP"))return false;
    *open.mutable_context_data()=descendant.context_data();if(readable())return false;*open.mutable_context_data()=query.context_data();
    if(!query.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology)||!ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP")||readable())return false;
    return true;
  };
  bool ok=true;for(const char* scenario:{"normal","compile","extra-arg","version-shape","copy","argv0","execfn","parent-class","parent-role","parent-active","parent-start","parent-network","ancestor-start","ancestor-role","parent-context","unsealed","root-writable","shadow","diag-only","other-profile"})ok=run(scenario)&&ok;
  close(output[0]);close(output[1]);return ok;
}

bool VerifyCargoBuildNativeCCChildBoundary(int native_stage = 0) {
  const bool collect2 = native_stage >= 1;
  const bool launcher = native_stage >= 2;
  const bool rust_lld = native_stage >= 3;
  const bool link_inputs = native_stage >= 4;
  const bool sdk_inputs = native_stage >= 5;
  const bool rust_c_abi = native_stage >= 6;
  const bool lld_entropy = native_stage >= 7;
  int output[2];if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
  auto run=[&](const char* scenario){
    ProcessState state;gvisor::common::ContextData root;
    root.set_container_id(kFirstID);root.set_thread_group_id(99);root.set_thread_group_start_time_ns(990);root.set_parent_thread_group_id(0);root.set_is_exec_session(true);root.set_process_name("ignored");
    if(!RegisterGroup(&state,root,ProcessState::Role::kArtifact,ProcessState::Provenance::kDirectExecRoot,false,true))return false;
    state.groups[99].current_image_class=ProcessClass::kCargo;
    gvisor::sentry::CloneInfo clone;*clone.mutable_context_data()=root;clone.set_created_thread_group_id(100);clone.set_created_thread_start_time_ns(1000);
    std::string payload,id;const char* reason=nullptr;
    if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
    gvisor::sentry::ExecveInfo sdk;*sdk.mutable_context_data()=root;sdk.mutable_context_data()->set_thread_group_id(100);sdk.mutable_context_data()->set_thread_group_start_time_ns(1000);sdk.mutable_context_data()->set_parent_thread_group_id(99);sdk.mutable_context_data()->set_is_exec_session(false);
    sdk.set_binary_path(kRustcBinary);sdk.set_execfn(kRustcBinary);sdk.add_argv(kRustcBinary);sdk.add_argv("-vV");
    TopologyState topology;topology.namespace_id=1;topology.snapshot_seen=true;topology.sealed=true;topology.expected.push_back(ExpectedMount{"/","oci-root","/","overlay",true,false,true,true});topology.anchors.emplace(1,MountAnchor{1,"/","oci-root"});
    ProfileRegistration registration;
    if(!sdk.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,kProfileCargoBuild,&registration,&state,&reason,&topology)||!ExpectRecordExact(output[1],id.c_str(),"process-exec-expected"))return false;
    *clone.mutable_context_data()=sdk.context_data();clone.set_created_thread_group_id(101);clone.set_created_thread_start_time_ns(1010);
    if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
    auto image=sdk;image.mutable_context_data()->set_thread_group_id(101);image.mutable_context_data()->set_thread_group_start_time_ns(1010);image.mutable_context_data()->set_parent_thread_group_id(100);
    image.set_binary_path("/usr/bin/x86_64-linux-gnu-gcc-12");image.set_execfn("/usr/bin/cc");image.clear_argv();image.add_argv("cc");image.add_argv("/tmp/input.o");image.add_argv("-o");image.add_argv("/tmp/haa-cargo-target/product");
    if(collect2){
      if(!image.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,kProfileCargoBuild,&registration,&state,&reason,&topology)||!ExpectRecordExact(output[1],id.c_str(),"process-exec-expected"))return false;
      *clone.mutable_context_data()=image.context_data();clone.set_created_thread_group_id(102);clone.set_created_thread_start_time_ns(1020);
      if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
      image.mutable_context_data()->set_thread_group_id(102);image.mutable_context_data()->set_thread_group_start_time_ns(1020);image.mutable_context_data()->set_parent_thread_group_id(101);
      image.set_binary_path("/usr/lib/gcc/x86_64-linux-gnu/12/collect2");image.set_execfn(image.binary_path());image.set_argv(0,image.binary_path());
    }
    if(launcher){
      if(!image.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,kProfileCargoBuild,&registration,&state,&reason,&topology)||!ExpectRecordExact(output[1],id.c_str(),"process-exec-expected"))return false;
      *clone.mutable_context_data()=image.context_data();clone.set_created_thread_group_id(103);clone.set_created_thread_start_time_ns(1030);
      if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
      image.mutable_context_data()->set_thread_group_id(103);image.mutable_context_data()->set_thread_group_start_time_ns(1030);image.mutable_context_data()->set_parent_thread_group_id(102);
      image.set_binary_path("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/bin/gcc-ld/ld.lld");image.set_execfn(image.binary_path());image.set_argv(0,image.binary_path());
    }
    if(rust_lld){
      if(!image.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,kProfileCargoBuild,&registration,&state,&reason,&topology)||!ExpectRecordExact(output[1],id.c_str(),"process-exec-expected"))return false;
      image.set_binary_path("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/bin/rust-lld");image.set_execfn(image.binary_path());image.set_argv(0,image.binary_path());
      if(strcmp(scenario,"launcher-active")==0)state.groups[103].cargo_lld_launcher_active=false;
      if(strcmp(scenario,"launcher-class")==0)state.groups[103].current_image_class=ProcessClass::kUnknown;
      if(strcmp(scenario,"diagnostic-forged")==0){state.groups[103].cargo_lld_launcher_active=false;state.groups[103].diagnostic_lld_same_group=true;}
      if(strcmp(scenario,"extra-clone")==0){
        *clone.mutable_context_data()=image.context_data();clone.set_created_thread_group_id(104);clone.set_created_thread_start_time_ns(1040);
        if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),output[0],&id,&state,&reason))return false;
        image.mutable_context_data()->set_thread_group_id(104);image.mutable_context_data()->set_thread_group_start_time_ns(1040);image.mutable_context_data()->set_parent_thread_group_id(103);
      }
    }
    const int32_t producer=launcher?102:(collect2?101:100);
    if(strcmp(scenario,"copy")==0)image.set_binary_path("/tmp/cc");
    if(strcmp(scenario,"execfn")==0)image.set_execfn("/tmp/cc");
    if(strcmp(scenario,"argv0")==0)image.set_argv(0,"other");
    if(strcmp(scenario,"parent-class")==0)state.groups[producer].current_image_class=ProcessClass::kUnknown;
    if(strcmp(scenario,"parent-active")==0){state.groups[producer].cargo_build_rustc_active=false;state.groups[producer].go_build_tool_active=false;};
    if(strcmp(scenario,"parent-role")==0)state.groups[producer].role=ProcessState::Role::kControl;
    if(strcmp(scenario,"parent-start")==0)state.groups[producer].start_time_ns++;
    if(strcmp(scenario,"grandparent-class")==0)state.groups[99].current_image_class=ProcessClass::kUnknown;
    if(strcmp(scenario,"grandparent-start")==0)state.groups[99].start_time_ns++;
    if(strcmp(scenario,"parent-network")==0)state.groups[producer].trusted_control_network_active=true;
    if(strcmp(scenario,"parent-pending")==0)state.groups[producer].handoff_target_pending=true;
    if(strcmp(scenario,"parent-context")==0)image.mutable_context_data()->set_parent_thread_group_id(98);
    if(strcmp(scenario,"unsealed")==0)topology.sealed=false;
    if(strcmp(scenario,"writable")==0)topology.expected[0].read_only=false;
    if(strcmp(scenario,"shadow")==0)topology.anchors.emplace(2,MountAnchor{2,launcher?"/usr/local/rustup":(collect2?"/usr/lib/gcc":"/usr/bin"),"workspace"});
    if(strcmp(scenario,"alias-shadow")==0)topology.anchors.emplace(2,MountAnchor{2,launcher?image.binary_path():(collect2?"/usr/lib/gcc/x86_64-linux-gnu/12/collect2":"/usr/bin/cc"),"workspace"});
    const char* profile=strcmp(scenario,"other-profile")==0?kProfileGoBuild:kProfileCargoBuild;
    const bool accepted=strcmp(scenario,"normal")==0;
    if(!image.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology))return false;
    if(accepted){if(!ExpectRecordExact(output[1],id.c_str(),"process-exec-expected"))return false;}
    else if(!ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP"))return false;
    gvisor::syscall::Open open;*open.mutable_context_data()=image.context_data();open.set_pathname("/etc/ld.so.cache");open.set_flags(557056);MountAnchor anchor{1,"/","oci-root"};
    auto readable=[&](){return ClassifyFilesystemOpen(open,state,profile,&anchor)==FilesystemClass::kHelperOnly;};
    if(readable()!=accepted||IsTrustedControlNetwork(image.context_data(),state)||state.groups[launcher?103:(collect2?102:101)].role!=ProcessState::Role::kArtifact)return false;
    if(!accepted)return true;
    if(launcher && !rust_lld){for(const char* path:{"/usr/lib/x86_64-linux-gnu/libdl.so.2","/usr/lib/x86_64-linux-gnu/librt.so.1","/usr/lib/x86_64-linux-gnu/libm.so.6"}){open.set_pathname(path);if(readable())return false;}}
    if(rust_lld){for(const char* path:{"/usr/lib/x86_64-linux-gnu/libpthread.so.0","/usr/lib/x86_64-linux-gnu/libm.so.6","/usr/lib/x86_64-linux-gnu/libgcc_s.so.1","/usr/lib/x86_64-linux-gnu/libc.so.6","/usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2","/usr/lib/x86_64-linux-gnu/libz.so.1.2.13","/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/libLLVM.so.23.1-rust-1.99.0-stable"}){open.set_pathname(path);if(!readable())return false;}}
    if(link_inputs){for(const char* path:{"/usr/lib/x86_64-linux-gnu/Scrt1.o","/usr/lib/x86_64-linux-gnu/crti.o","/usr/lib/x86_64-linux-gnu/crtn.o","/usr/lib/x86_64-linux-gnu/libc.so","/usr/lib/x86_64-linux-gnu/libc_nonshared.a","/usr/lib/gcc/x86_64-linux-gnu/12/crtbeginS.o","/usr/lib/gcc/x86_64-linux-gnu/12/crtendS.o","/usr/lib/gcc/x86_64-linux-gnu/12/libgcc.a","/usr/lib/gcc/x86_64-linux-gnu/12/libgcc_eh.a","/usr/lib/gcc/x86_64-linux-gnu/12/libgcc_s.so"}){open.set_pathname(path);if(!readable())return false;}}
    if(sdk_inputs){for(const char* path:{"/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/lib","/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/lib/libstd-44675f186029a4cd.rlib"}){open.set_pathname(path);if(!readable())return false;}}
    if(rust_c_abi){for(const char* path:{"/usr/lib/x86_64-linux-gnu/libutil.a","/usr/lib/x86_64-linux-gnu/librt.a","/usr/lib/x86_64-linux-gnu/libm.so","/usr/lib/x86_64-linux-gnu/libmvec.so.1"}){open.set_pathname(path);if(!readable())return false;}}
    for(const char* path:{"/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so","/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/aarch64-unknown-linux-gnu/lib/libstd.rlib","/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/bin/other"}){open.set_pathname(path);if(readable())return false;}
    for(const char* path:{"/etc/shadow","/usr/local/rustup/credentials","/usr/local/go/src/runtime"}){open.set_pathname(path);if(readable())return false;}
    if(lld_entropy){
      open.set_pathname("/dev/urandom");open.set_flags(32768);anchor={10,"/dev","system"};if(!readable())return false;
      for(uint64_t flags:{0,1,2,64,557056}){open.set_flags(flags);if(readable())return false;}
      open.set_flags(32768);open.set_pathname("/dev/random");if(readable())return false;
      open.set_pathname("/dev/urandom");anchor={10,"/","system"};if(readable())return false;
      anchor={1,"/","oci-root"};if(readable())return false;
    }
    open.set_pathname("/etc/ld.so.cache");open.set_flags(kOpenWriteOnly);if(readable())return false;open.set_flags(557056);
    if(!image.SerializeToString(&payload)||!ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,&topology)||!ExpectUnexpectedProcessRecord(output[1],id.c_str(),"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP")||readable())return false;
    return true;
  };
  bool ok=true;for(const char* scenario:{"normal","copy","execfn","argv0","parent-class","parent-active","parent-role","parent-start","grandparent-class","grandparent-start","parent-network","parent-pending","parent-context","unsealed","writable","shadow","alias-shadow","other-profile"}){if(rust_lld && (strcmp(scenario,"execfn")==0 || strcmp(scenario,"argv0")==0))continue;ok=run(scenario)&&ok;}
  if(rust_lld)for(const char* scenario:{"launcher-active","launcher-class","diagnostic-forged","extra-clone"})ok=run(scenario)&&ok;
  close(output[0]);close(output[1]);return ok;
}

bool VerifyCargoBuildDriverRuntimeRead() {
  ProcessState accepted;
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID); context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990); context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true); context.set_process_name("renamed-cargo");
  if (!RegisterGroup(&accepted, context, ProcessState::Role::kArtifact,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  accepted.groups[99].current_image_class = ProcessClass::kCargo;
  gvisor::syscall::Open open;
  *open.mutable_context_data() = context; open.set_pathname("/etc/ld.so.cache"); open.set_flags(557056);
  MountAnchor anchor{1, "/", "oci-root"};
  auto readable = [&](const ProcessState& state) { return ClassifyFilesystemOpen(open, state, kProfileCargoBuild, &anchor) == FilesystemClass::kHelperOnly; };
  for (const char* path : {"/etc/ld.so.cache", "/usr/lib/x86_64-linux-gnu/libc.so.6", "/usr/lib/x86_64-linux-gnu/librt.so.1", "/usr/lib/x86_64-linux-gnu/libdl.so.2", "/usr/lib/x86_64-linux-gnu/libgcc_s.so.1", "/usr/lib/x86_64-linux-gnu/libpthread.so.0", "/usr/lib/x86_64-linux-gnu/libm.so.6", "/usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2", "/usr/share/zoneinfo/Etc/UTC"}) {
    open.set_pathname(path); if (!readable(accepted)) return false;
  }
  open.set_pathname("/etc/ssl/certs/ca-certificates.crt"); open.set_flags(32768);
  if (!readable(accepted)) return false;
  for (uint64_t flags : {0, 557056, 1, 2, 64}) {open.set_flags(flags); if(readable(accepted)) return false;}
  open.set_flags(557056);
  open.set_pathname("/etc/ld.so.cache");
  for (int mutation=0; mutation<10; ++mutation) {
    ProcessState rejected=accepted; auto& group=rejected.groups[99];
    switch(mutation){
      case 0:group.role=ProcessState::Role::kControl;break;
      case 1:group.provenance=ProcessState::Provenance::kCloneChild;break;
      case 2:group.current_image_class=ProcessClass::kUnknown;break;
      case 3:group.root_eligible=true;break;
      case 4:group.root_consumed=false;break;
      case 5:group.trusted_control_network_active=true;break;
      case 6:group.demotion_pending=true;break;
      case 7:group.launch_target_pending=true;break;
      case 8:group.handoff_target_pending=true;break;
      case 9:group.start_time_ns++;break;
    }
    // Existing canonical CONTROL loader mechanics are preserved. This
    // negative checks ARTIFACT inheritance, not removal of CONTROL reads.
    if(readable(rejected) != (mutation == 0)) return false;
  }
  for(uint64_t flags:{1,2,64,512,1024}){open.set_flags(flags);if(readable(accepted))return false;}
  open.set_flags(557056);
  for(const char* path:{"/etc/shadow","/etc/passwd","/etc/ssl/certs/private-cert.pem","/usr/lib/evil.so","/usr/lib/x86_64-linux-gnu/libacl.so.1","/usr/share/zoneinfo/Europe/London"}){open.set_pathname(path);if(readable(accepted))return false;}
  open.set_pathname("/etc/ld.so.cache"); anchor.mountpoint="/etc";if(readable(accepted))return false;
  anchor={10,"/proc","system"};
  for(const char* path:{"/proc/99/maps","/proc/sys/vm/overcommit_memory"}){open.set_pathname(path);if(!readable(accepted))return false;}
  for(const char* path:{"/proc/98/maps","/proc/99/environ","/proc/sys/vm/drop_caches"}){open.set_pathname(path);if(readable(accepted))return false;}
  anchor={11,"/dev","system"};open.set_pathname("/dev/urandom");if(!readable(accepted))return false;
  open.set_pathname("/dev/null");if(!readable(accepted))return false;
  open.set_pathname("/dev/zero");if(readable(accepted))return false;
  anchor={1,"/","oci-root"};open.set_pathname("/etc/ld.so.cache");
  open.mutable_context_data()->set_parent_thread_group_id(98);if(readable(accepted))return false;
  open.mutable_context_data()->set_parent_thread_group_id(0);open.mutable_context_data()->set_is_exec_session(false);if(readable(accepted))return false;
  return MaximumRecords(kProfileCargoBuild)==10000;
}

bool VerifyCargoBuildHandoffPinnedTarget() {
  auto run = [](bool shadow) {
    ProcessState state;
    gvisor::sentry::ExecveInfo message;
    auto* context = message.mutable_context_data();
    context->set_container_id(kFirstID);
    context->set_thread_group_id(99);
    context->set_thread_group_start_time_ns(990);
    context->set_parent_thread_group_id(0);
    context->set_is_exec_session(true);
    context->set_process_name("untrusted-name");
    if (!RegisterGroup(&state, *context, ProcessState::Role::kArtifact,
                       ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
    state.groups[99].handoff_target_pending = true;
    state.groups[99].handoff_target_class = ProcessClass::kCargo;
    message.set_binary_path(kCargoBinary);
    message.set_execfn(kCargoBinary);
    message.add_argv(kCargoBinary);
    message.add_argv("metadata");
    TopologyState topology;
    topology.sealed = true;
    topology.snapshot_seen = true;
    topology.namespace_id = 1;
    MountAnchor root{1, "/", "oci-root"};
    topology.expected.push_back(ExpectedMount{"/", "oci-root", "/", "overlay", true, false, true, true});
    topology.anchors.emplace(1, root);
    if (shadow) topology.anchors.emplace(2, MountAnchor{2, "/usr/local/rustup", "workspace"});
    std::string payload, id;
    const char* reason = nullptr;
    int sockets[2];
    if (socketpair(AF_UNIX, SOCK_SEQPACKET, 0, sockets) != 0) return false;
    auto registration = TestProfileWithAdmission(kProfileCargoBuild, kELFHandoffMode, "1234567890abcdef");
    bool ok = message.SerializeToString(&payload) && ParseSentryProcessAndClassify(
        payload.data(), payload.size(), sockets[0], &id, kProfileCargoBuild, &registration, &state, &reason, &topology);
    std::array<char, 4096> body{};
    const ssize_t length = ok ? recv(sockets[1], body.data(), body.size(), 0) : -1;
    const std::string text = length > 0 ? std::string(body.data(), length) : "";
    ok = ok && text.find(shadow ? "process-exec-unexpected" : "process-exec-expected") != std::string::npos &&
        state.groups[99].handoff_target_pending == shadow &&
        state.groups[99].role == ProcessState::Role::kArtifact &&
        !state.groups[99].trusted_control_network_active;
    close(sockets[0]); close(sockets[1]);
    return ok;
  };
  return run(false) && run(true);
}

bool VerifyGoBuildRuntimeMetadataBoundary() {
  ProcessState state;
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID);
  context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("renamed-build");
  if (!RegisterGroup(&state, context, ProcessState::Role::kArtifact,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  auto& group = state.groups[99];
  group.current_image_class = ProcessClass::kGo;
  gvisor::syscall::Open open;
  *open.mutable_context_data() = context;
  open.set_pathname("/proc/99/cgroup");
  open.set_flags(557056);
  MountAnchor anchor{10, "/proc", "system"};
  auto classify = [&]() { return ClassifyFilesystemOpen(open, state, kProfileGoBuild, &anchor); };
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/proc/99/mountinfo");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  anchor.mountpoint = "/sys/fs/cgroup/cpu";
  for (const char* path : {"/sys/fs/cgroup/cpu/cpu.cfs_quota_us", "/sys/fs/cgroup/cpu/cpu.cfs_period_us"}) {
    open.set_pathname(path);
    if (classify() != FilesystemClass::kHelperOnly) return false;
  }
  for (const char* path : {"/sys/fs/cgroup/cpu/cpu.shares", "/sys/fs/cgroup/memory/memory.limit_in_bytes", "/sys/devices/system/cpu/online"}) {
    open.set_pathname(path);
    if (classify() == FilesystemClass::kHelperOnly) return false;
  }
  open.set_pathname("/sys/fs/cgroup/cpu/cpu.cfs_quota_us");
  group.role = ProcessState::Role::kControl;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.role = ProcessState::Role::kArtifact;
  group.provenance = ProcessState::Provenance::kCloneChild;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.provenance = ProcessState::Provenance::kDirectExecRoot;
  group.current_image_class = ProcessClass::kUnknown;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.current_image_class = ProcessClass::kGo;
  group.trusted_control_network_active = true;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.trusted_control_network_active = false;
  open.set_flags(kOpenWriteOnly);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(557056);
  anchor.mountpoint = "/sys/fs/cgroup";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor.mountpoint = "/sys/fs/cgroup/cpu";
  anchor.mount_class = "oci-root";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor.mount_class = "system";
  open.mutable_context_data()->set_thread_group_start_time_ns(991);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.mutable_context_data()->set_thread_group_start_time_ns(990);
  if (ClassifyFilesystemOpen(open, state, kProfileGoResolver, &anchor) == FilesystemClass::kHelperOnly) return false;
  anchor.mountpoint = "/";
  anchor.mount_class = "oci-root";
  open.set_pathname("/usr/local/go/go.env");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/usr/local/go/go.env.extra");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  for (const char* path : {"/usr/local/go/src", "/usr/local/go/src/runtime", "/usr/local/go/src/runtime/asm_amd64.s", "/usr/local/go/pkg/include", "/usr/local/go/pkg/include/textflag.h", "/usr/share/zoneinfo/Etc/UTC"}) {
    open.set_pathname(path);
    if (classify() != FilesystemClass::kHelperOnly) return false;
  }
  for (const char* path : {"/usr/local/go/src.extra", "/usr/local/go/pkg/include.extra", "/usr/local/go/pkg/tool/linux_amd64/compile", "/usr/local/go/test/eof.go", "/usr/share/zoneinfo/Etc/GMT"}) {
    open.set_pathname(path);
    if (classify() == FilesystemClass::kHelperOnly) return false;
  }
  open.set_pathname("/usr/local/go/go.env");
  open.set_flags(kOpenWriteOnly);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(557056);
  anchor.mountpoint = "/usr/local/go";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor.mountpoint = "/dev";
  anchor.mount_class = "system";
  open.set_pathname("/dev/null");
  if (classify() != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/dev/zero");
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/dev/null");
  open.set_flags(kOpenWriteOnly);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(557056);
  anchor.mountpoint = "/dev/null";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor.mountpoint = "/dev";
  anchor.mount_class = "oci-root";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor.mountpoint = "/proc";
  anchor.mount_class = "system";
  for (const char* path : {"/proc/98/cgroup", "/proc/99/maps", "/proc/99/environ", "/proc/99/status", "/etc/ld.so.cache"}) {
    open.set_pathname(path);
    if (classify() == FilesystemClass::kHelperOnly) return false;
  }
  open.set_pathname("/proc/99/cgroup");
  group.role = ProcessState::Role::kControl;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.role = ProcessState::Role::kArtifact;
  group.provenance = ProcessState::Provenance::kCloneChild;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.provenance = ProcessState::Provenance::kDirectExecRoot;
  group.handoff_target_pending = true;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.handoff_target_pending = false;
  group.current_image_class = ProcessClass::kUnknown;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.current_image_class = ProcessClass::kGo;
  group.trusted_control_network_active = true;
  if (classify() == FilesystemClass::kHelperOnly) return false;
  group.trusted_control_network_active = false;
  open.set_flags(kOpenWriteOnly);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  open.set_flags(557056);
  anchor.mountpoint = "/proc/99";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor.mountpoint = "/proc";
  anchor.mount_class = "oci-root";
  if (classify() == FilesystemClass::kHelperOnly) return false;
  anchor.mount_class = "system";
  open.mutable_context_data()->set_thread_group_start_time_ns(991);
  if (classify() == FilesystemClass::kHelperOnly) return false;
  return ClassifyFilesystemOpen(open, state, kProfileGoResolver, &anchor) != FilesystemClass::kHelperOnly;
}

bool VerifyGoBuildCreatorDiagnosticBoundary() {
  ProcessState state;
  gvisor::common::ContextData parent;
  parent.set_container_id(kFirstID);
  parent.set_thread_group_id(99);
  parent.set_thread_group_start_time_ns(990);
  parent.set_parent_thread_group_id(0);
  parent.set_is_exec_session(true);
  if (!RegisterGroup(&state, parent, ProcessState::Role::kArtifact,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  state.groups[99].current_image_class = ProcessClass::kGo;
  auto context = parent;
  context.set_thread_group_id(100);
  context.set_thread_group_start_time_ns(1000);
  context.set_parent_thread_group_id(99);
  context.set_is_exec_session(false);
  if (!RegisterGroup(&state, context, ProcessState::Role::kArtifact,
                     ProcessState::Provenance::kCloneChild, false, true)) return false;
  auto& child = state.groups[100];
  child.clone_creator_group_id = 99;
  child.clone_creator_group_start_time_ns = 990;
  gvisor::syscall::Open open;
  *open.mutable_context_data() = context;
  open.set_pathname("/proc/100/cgroup");
  open.set_flags(557056);
  MountAnchor anchor{10, "/proc", "system"};
  auto exact = [&]() {
    return FaultOpenDiagnostic(open, state, anchor).find("\"go_driver_creator\":true") != std::string::npos;
  };
  if (!exact()) return false;
  child.clone_creator_group_start_time_ns = 991;
  if (exact()) return false;
  child.clone_creator_group_start_time_ns = 990;
  state.groups[99].current_image_class = ProcessClass::kUnknown;
  if (exact()) return false;
  state.groups[99].current_image_class = ProcessClass::kGo;
  state.groups[99].role = ProcessState::Role::kControl;
  if (exact()) return false;
  state.groups[99].role = ProcessState::Role::kArtifact;
  state.groups[99].handoff_target_pending = true;
  if (exact()) return false;
  state.groups[99].handoff_target_pending = false;
  open.mutable_context_data()->set_parent_thread_group_id(98);
  if (exact()) return false;
  open.mutable_context_data()->set_parent_thread_group_id(99);
  child.provenance = ProcessState::Provenance::kDirectExecRoot;
  return !exact();
}

bool VerifyGoBuildToolChildBoundary() {
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  TopologyState topology;
  topology.expected.push_back(ExpectedMount{"/", "oci-root", "/", "", true, false, false, false});
  topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
  topology.namespace_id = 10; topology.snapshot_seen = true; topology.sealed = true;
  auto run = [&](const char* tool, const char* scenario) {
    ProcessState state;
    gvisor::common::ContextData parent;
    parent.set_container_id(kFirstID); parent.set_thread_group_id(99);
    parent.set_thread_group_start_time_ns(990); parent.set_parent_thread_group_id(0);
    parent.set_is_exec_session(true); parent.set_process_name("renamed-driver");
    if (!RegisterGroup(&state, parent, ProcessState::Role::kArtifact,
                       ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
    state.groups[99].current_image_class = ProcessClass::kGo;
    gvisor::sentry::CloneInfo clone;
    *clone.mutable_context_data() = parent;
    clone.set_created_thread_group_id(100); clone.set_created_thread_start_time_ns(1000);
    std::string payload, id; const char* reason = nullptr;
    if (!clone.SerializeToString(&payload) || !ParseSentryClone(payload.data(), payload.size(),
          output[0], &id, &state, &reason)) return false;
    gvisor::sentry::ExecveInfo image;
    *image.mutable_context_data() = parent;
    image.mutable_context_data()->set_thread_group_id(100);
    image.mutable_context_data()->set_thread_group_start_time_ns(1000);
    image.mutable_context_data()->set_parent_thread_group_id(99);
    image.mutable_context_data()->set_is_exec_session(false);
    image.mutable_context_data()->set_process_name("renamed-compiler");
    const bool native_gcc = strcmp(tool, "gcc") == 0;
    const std::string path = native_gcc ? "/usr/bin/x86_64-linux-gnu-gcc-12" :
        std::string("/usr/local/go/pkg/tool/linux_amd64/") + tool;
    image.set_binary_path(path);
    image.set_execfn(native_gcc ? "/usr/bin/gcc" : path);
    image.add_argv(native_gcc ? "gcc" : path);
    image.add_argv("-V=full");
    auto mounts = topology;
    const TopologyState* selected = &mounts;
    const bool expected = strcmp(scenario, "normal") == 0 || strcmp(scenario, "absolute-argv0") == 0;
    if (strcmp(scenario, "absolute-argv0") == 0) image.set_argv(0, "/usr/bin/gcc");
    if (strcmp(scenario, "alias-shadow") == 0) mounts.anchors.emplace(2, MountAnchor{2, "/usr/bin/gcc", "workspace"});
    if (strcmp(scenario, "copied-image") == 0) image.set_binary_path("/tmp/compile");
    if (strcmp(scenario, "execfn") == 0) image.set_execfn("/tmp/alias");
    if (strcmp(scenario, "argv0") == 0) image.set_argv(0, "compile");
    if (strcmp(scenario, "parent-class") == 0) state.groups[99].current_image_class = ProcessClass::kUnknown;
    if (strcmp(scenario, "parent-role") == 0) state.groups[99].role = ProcessState::Role::kControl;
    if (strcmp(scenario, "parent-start") == 0) state.groups[99].start_time_ns = 991;
    if (strcmp(scenario, "parent-pending") == 0) state.groups[99].handoff_target_pending = true;
    if (strcmp(scenario, "parent-context") == 0) image.mutable_context_data()->set_parent_thread_group_id(98);
    if (strcmp(scenario, "missing-clone") == 0) {
      state.groups.erase(100);
      if (!RegisterGroup(&state, image.context_data(), ProcessState::Role::kArtifact,
                         ProcessState::Provenance::kCloneChild, false, true)) return false;
      state.groups[100].clone_creator_group_id = 99;
      state.groups[100].clone_creator_group_start_time_ns = 990;
      state.groups[100].current_image_class = ProcessClass::kGo;
    }
    if (strcmp(scenario, "unsealed") == 0) mounts.sealed = false;
    if (strcmp(scenario, "writable-root") == 0) mounts.expected[0].read_only = false;
    if (strcmp(scenario, "shadow") == 0) mounts.anchors.emplace(2, MountAnchor{2, native_gcc ? "/usr/bin" : "/usr/local/go/pkg", "system"});
    if (strcmp(scenario, "no-topology") == 0) selected = nullptr;
    ProfileRegistration registration;
    if (!image.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(),
          output[0], &id, strcmp(scenario, "other-profile") == 0 ? kProfileGoResolver : kProfileGoBuild, &registration, &state, &reason, selected)) return false;
    if (expected ? !ExpectRecordExact(output[1], id.c_str(), "process-exec-expected") :
        !ExpectUnexpectedProcessRecord(output[1], id.c_str(), "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
    if (state.groups.at(100).role != ProcessState::Role::kArtifact || state.groups.at(100).root_eligible ||
        IsTrustedControlNetwork(image.context_data(), state) || state.expected_groups.count(100) != 0) return false;
    gvisor::syscall::Open open;
    *open.mutable_context_data() = image.context_data(); open.set_flags(557056);
    open.set_pathname(native_gcc ? "/etc/ld.so.cache" : "/proc/100/cgroup");
    MountAnchor proc = native_gcc ? MountAnchor{1, "/", "oci-root"} : MountAnchor{10, "/proc", "system"};
    auto readable = [&]() { return ClassifyFilesystemOpen(open, state, kProfileGoBuild, &proc) == FilesystemClass::kHelperOnly; };
    if (readable() != expected) return false;
    if (!expected) return true;
    const MountAnchor sdk{1, "/", "oci-root"};
    for (const char* path : {"/usr/local/go/src/runtime/runtime.go", "/usr/local/go/pkg/include/textflag.h"}) {
      open.set_pathname(path);
      if ((ClassifyFilesystemOpen(open, state, kProfileGoBuild, &sdk) == FilesystemClass::kHelperOnly) == native_gcc) return false;
      open.set_flags(kOpenWriteOnly);
      if (ClassifyFilesystemOpen(open, state, kProfileGoBuild, &sdk) == FilesystemClass::kHelperOnly) return false;
      open.set_flags(557056);
    }
    if (native_gcc) {
      for (const char* path : {"/lib/x86_64-linux-gnu/libc.so.6", "/usr/lib/x86_64-linux-gnu/libc.so.6"}) {
        open.set_pathname(path); if (!readable()) return false;
        open.set_flags(kOpenWriteOnly); if (readable()) return false;
        open.set_flags(557056);
      }
      for (const char* path : {"/etc/passwd", "/usr/lib/x86_64-linux-gnu/libm.so.6", "/usr/include/stdio.h", "/proc/100/cgroup"}) {
        open.set_pathname(path); if (readable()) return false;
      }
    }
    for (const char* path : {"/proc/99/cgroup", "/proc/100/environ", "/proc/100/maps"}) {
      open.set_pathname(path); if (readable()) return false;
    }
    open.set_pathname(native_gcc ? "/etc/ld.so.cache" : "/proc/100/cgroup"); open.set_flags(kOpenWriteOnly);
    if (readable()) return false;
    open.set_flags(557056);
    proc.mountpoint = "/proc/100"; if (readable()) return false;
    proc.mountpoint = native_gcc ? "/" : "/proc";
    open.mutable_context_data()->set_thread_group_start_time_ns(1001); if (readable()) return false;
    open.mutable_context_data()->set_thread_group_start_time_ns(1000);
    // A compiler descendant cannot inherit the driver's one-shot tool grant.
    *clone.mutable_context_data() = image.context_data();
    clone.set_created_thread_group_id(101); clone.set_created_thread_start_time_ns(1010);
    if (!clone.SerializeToString(&payload) || !ParseSentryClone(payload.data(), payload.size(),
          output[0], &id, &state, &reason)) return false;
    auto descendant = image;
    descendant.mutable_context_data()->set_thread_group_id(101);
    descendant.mutable_context_data()->set_thread_group_start_time_ns(1010);
    descendant.mutable_context_data()->set_parent_thread_group_id(100);
    if (!descendant.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(),
          output[0], &id, kProfileGoBuild, &registration, &state, &reason, &topology) ||
        !ExpectUnexpectedProcessRecord(output[1], id.c_str(), "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
    // A second exec also consumes no renewed grant, even for the same SDK tool.
    if (!image.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(),
          output[0], &id, kProfileGoBuild, &registration, &state, &reason, &topology) ||
        !ExpectUnexpectedProcessRecord(output[1], id.c_str(), "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
    return !readable() && state.first_unexpected_exec.present;
  };
  bool ok = true;
  for (const char* tool : {"compile", "asm", "link", "cgo"}) ok = ok && run(tool, "normal");
  for (const char* tool : {"compile", "asm", "link", "cgo"}) {
    for (const char* scenario : {"copied-image", "execfn", "argv0", "parent-class", "parent-role", "parent-start",
         "parent-pending", "parent-context", "missing-clone", "unsealed", "writable-root", "shadow", "no-topology"}) {
      ok = ok && run(tool, scenario);
    }
  }
  ok = ok && run("vet", "outside-build-tools");
  ok = ok && run("gcc", "normal") && run("gcc", "absolute-argv0");
  for (const char* scenario : {"copied-image", "execfn", "argv0", "parent-class", "parent-role", "parent-start",
       "parent-pending", "parent-context", "missing-clone", "unsealed", "writable-root", "shadow", "no-topology",
       "alias-shadow", "other-profile"}) ok = ok && run("gcc", scenario);
  close(output[0]); close(output[1]);
  return ok;
}

bool VerifyGoBuildCgoGccCreatorBoundary(bool sdk_link = false) {
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  TopologyState topology;
  topology.expected.push_back(ExpectedMount{"/", "oci-root", "/", "", true, false, false, false});
  topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
  topology.namespace_id = 10; topology.snapshot_seen = true; topology.sealed = true;
  auto run = [&](const char* scenario) {
    ProcessState state;
    gvisor::common::ContextData driver;
    driver.set_container_id(kFirstID); driver.set_thread_group_id(99);
    driver.set_thread_group_start_time_ns(990); driver.set_parent_thread_group_id(0);
    driver.set_is_exec_session(true); driver.set_process_name("renamed-driver");
    if (!RegisterGroup(&state, driver, ProcessState::Role::kArtifact,
                       ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
    state.groups[99].current_image_class = ProcessClass::kGo;
    std::string payload, id; const char* reason = nullptr;
    auto clone = [&](const gvisor::common::ContextData& creator, int child, int64_t start) {
      gvisor::sentry::CloneInfo event;
      *event.mutable_context_data() = creator;
      event.set_created_thread_group_id(child); event.set_created_thread_start_time_ns(start);
      return event.SerializeToString(&payload) && ParseSentryClone(payload.data(), payload.size(),
          output[0], &id, &state, &reason);
    };
    if (!clone(driver, 100, 1000)) return false;
    gvisor::sentry::ExecveInfo cgo;
    *cgo.mutable_context_data() = driver;
    cgo.mutable_context_data()->set_thread_group_id(100);
    cgo.mutable_context_data()->set_thread_group_start_time_ns(1000);
    cgo.mutable_context_data()->set_parent_thread_group_id(99);
    cgo.mutable_context_data()->set_is_exec_session(false);
    cgo.mutable_context_data()->set_process_name("renamed-cgo");
    cgo.set_binary_path(sdk_link ? "/usr/local/go/pkg/tool/linux_amd64/link" : "/usr/local/go/pkg/tool/linux_amd64/cgo");
    cgo.set_execfn(cgo.binary_path()); cgo.add_argv(cgo.binary_path());
    ProfileRegistration registration;
    auto exec = [&](const gvisor::sentry::ExecveInfo& event) {
      return event.SerializeToString(&payload) && ParseSentryProcessAndClassify(payload.data(), payload.size(),
          output[0], &id, kProfileGoBuild, &registration, &state, &reason, &topology);
    };
    if (!exec(cgo) || !ExpectRecordExact(output[1], id.c_str(), "process-exec-expected")) return false;
    if (!clone(cgo.context_data(), 101, 1010)) return false;
    gvisor::sentry::ExecveInfo gcc;
    *gcc.mutable_context_data() = cgo.context_data();
    gcc.mutable_context_data()->set_thread_group_id(101);
    gcc.mutable_context_data()->set_thread_group_start_time_ns(1010);
    gcc.mutable_context_data()->set_parent_thread_group_id(100);
    gcc.mutable_context_data()->set_process_name("renamed-gcc");
    gcc.set_binary_path("/usr/bin/x86_64-linux-gnu-gcc-12");
    gcc.set_execfn("/usr/bin/gcc"); gcc.add_argv("gcc"); gcc.add_argv("-E");
    const bool expected = strcmp(scenario, "normal") == 0;
    if (strcmp(scenario, "parent-image") == 0) state.groups[100].current_image_class =
        ProcessClassForPath("/usr/local/go/pkg/tool/linux_amd64/compile", kProfileGoBuild);
    if (strcmp(scenario, "parent-role") == 0) state.groups[100].role = ProcessState::Role::kControl;
    if (strcmp(scenario, "parent-start") == 0) state.groups[100].start_time_ns = 1001;
    if (strcmp(scenario, "parent-inactive") == 0) state.groups[100].go_build_tool_active = false;
    if (strcmp(scenario, "parent-pending") == 0) state.groups[100].handoff_target_pending = true;
    if (strcmp(scenario, "driver-image") == 0) state.groups[99].current_image_class = ProcessClass::kUnknown;
    if (strcmp(scenario, "driver-role") == 0) state.groups[99].role = ProcessState::Role::kControl;
    if (strcmp(scenario, "driver-start") == 0) state.groups[99].start_time_ns = 991;
    if (strcmp(scenario, "creator-context") == 0) gcc.mutable_context_data()->set_parent_thread_group_id(99);
    if (!exec(gcc)) return false;
    if (expected ? !ExpectRecordExact(output[1], id.c_str(), "process-exec-expected") :
        !ExpectUnexpectedProcessRecord(output[1], id.c_str(), "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
    if (state.groups[101].role != ProcessState::Role::kArtifact || state.groups[101].root_eligible ||
        state.expected_groups.count(101) != 0 || IsTrustedControlNetwork(gcc.context_data(), state)) return false;
    gvisor::syscall::Open open;
    *open.mutable_context_data() = gcc.context_data(); open.set_pathname("/etc/ld.so.cache"); open.set_flags(557056);
    const MountAnchor image{1, "/", "oci-root"};
    auto readable = [&]() { return ClassifyFilesystemOpen(open, state, kProfileGoBuild, &image) == FilesystemClass::kHelperOnly; };
    if (readable() != expected) return false;
    if (!expected) return true;
    // The cgo ancestry cannot renew a GCC exec grant or pass one to a GCC descendant.
    if (!clone(gcc.context_data(), 102, 1020)) return false;
    auto descendant = gcc;
    descendant.mutable_context_data()->set_thread_group_id(102);
    descendant.mutable_context_data()->set_thread_group_start_time_ns(1020);
    descendant.mutable_context_data()->set_parent_thread_group_id(101);
    if (!exec(descendant) || !ExpectUnexpectedProcessRecord(output[1], id.c_str(),
        "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
    if (!exec(gcc) || !ExpectUnexpectedProcessRecord(output[1], id.c_str(),
        "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
    return !readable();
  };
  bool ok = run("normal");
  for (const char* scenario : {"parent-image", "parent-role", "parent-start", "parent-inactive", "parent-pending",
       "driver-image", "driver-role", "driver-start", "creator-context"}) ok = ok && run(scenario);
  close(output[0]); close(output[1]);
  return ok;
}

enum class GoBuildGccChildFixture { kCc1, kAssembler, kCollect2, kNativeLinker, kNativeLinkerPlugin, kNativeLinkerNull, kNativeLinkerRuntime, kAssemblerNull, kCc1SDKCgo, kNativeLinkerSDKLink, kNativeLinkerResolver };

bool VerifyGoBuildGccChildBoundary(bool check_headers = false,
                                 GoBuildGccChildFixture tool = GoBuildGccChildFixture::kCc1) {
  const bool resolver_abi = tool == GoBuildGccChildFixture::kNativeLinkerResolver;
  const bool sdk_link = tool == GoBuildGccChildFixture::kNativeLinkerSDKLink;
  const bool sdk_cgo = tool == GoBuildGccChildFixture::kCc1SDKCgo;
  const bool assembler_null = tool == GoBuildGccChildFixture::kAssemblerNull;
  const bool assembler = tool == GoBuildGccChildFixture::kAssembler || assembler_null;
  const bool linker_plugin = tool == GoBuildGccChildFixture::kNativeLinkerPlugin;
  const bool linker_null = tool == GoBuildGccChildFixture::kNativeLinkerNull;
  const bool null_device = linker_null || assembler_null;
  const bool runtime_data = tool == GoBuildGccChildFixture::kNativeLinkerRuntime;
  const bool native_linker = tool == GoBuildGccChildFixture::kNativeLinker || linker_plugin || linker_null || runtime_data || sdk_link || resolver_abi;
  const bool collect2 = tool == GoBuildGccChildFixture::kCollect2 || native_linker;
  const int target_group = native_linker ? 103 : 102;
  const int64_t target_start = target_group * 10;
  const int parent_group = native_linker ? 102 : 101;
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  auto run = [&](const char* scenario) {
    TopologyState topology;
    topology.expected.push_back(ExpectedMount{"/", "oci-root", "/", "", true, false, false, false});
    topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
    topology.namespace_id = 10; topology.snapshot_seen = true; topology.sealed = true;
    ProcessState state;
    gvisor::common::ContextData driver;
    driver.set_container_id(kFirstID); driver.set_thread_group_id(99);
    driver.set_thread_group_start_time_ns(990); driver.set_parent_thread_group_id(0);
    driver.set_is_exec_session(true); driver.set_process_name("renamed-driver");
    if (!RegisterGroup(&state, driver, ProcessState::Role::kArtifact,
                       ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
    state.groups[99].current_image_class = ProcessClass::kGo;
    std::string payload, id; const char* reason = nullptr;
    auto clone = [&](const gvisor::common::ContextData& creator, int child, int64_t start) {
      gvisor::sentry::CloneInfo event;
      *event.mutable_context_data() = creator;
      event.set_created_thread_group_id(child); event.set_created_thread_start_time_ns(start);
      return event.SerializeToString(&payload) && ParseSentryClone(payload.data(), payload.size(),
          output[0], &id, &state, &reason);
    };
    if (!clone(driver, 100, 1000)) return false;
    gvisor::sentry::ExecveInfo cgo;
    *cgo.mutable_context_data() = driver;
    cgo.mutable_context_data()->set_thread_group_id(100);
    cgo.mutable_context_data()->set_thread_group_start_time_ns(1000);
    cgo.mutable_context_data()->set_parent_thread_group_id(99);
    cgo.mutable_context_data()->set_is_exec_session(false);
    cgo.mutable_context_data()->set_process_name("renamed-cgo");
    cgo.set_binary_path(sdk_link ? "/usr/local/go/pkg/tool/linux_amd64/link" : "/usr/local/go/pkg/tool/linux_amd64/cgo");
    cgo.set_execfn(cgo.binary_path()); cgo.add_argv(cgo.binary_path());
    ProfileRegistration registration;
    auto exec = [&](const gvisor::sentry::ExecveInfo& event) {
      return event.SerializeToString(&payload) && ParseSentryProcessAndClassify(payload.data(), payload.size(),
          output[0], &id, kProfileGoBuild, &registration, &state, &reason, &topology);
    };
    if (!exec(cgo) || !ExpectRecordExact(output[1], id.c_str(), "process-exec-expected")) return false;
    if (!clone(cgo.context_data(), 101, 1010)) return false;
    gvisor::sentry::ExecveInfo gcc;
    *gcc.mutable_context_data() = cgo.context_data();
    gcc.mutable_context_data()->set_thread_group_id(101);
    gcc.mutable_context_data()->set_thread_group_start_time_ns(1010);
    gcc.mutable_context_data()->set_parent_thread_group_id(100);
    gcc.mutable_context_data()->set_process_name("renamed-gcc");
    gcc.set_binary_path("/usr/bin/x86_64-linux-gnu-gcc-12");
    gcc.set_execfn("/usr/bin/gcc"); gcc.add_argv("gcc"); gcc.add_argv("-E");
    if (!exec(gcc) || !ExpectRecordExact(output[1], id.c_str(), "process-exec-expected")) return false;
    if (!clone(gcc.context_data(), 102, 1020)) return false;
    gvisor::sentry::ExecveInfo cc1;
    *cc1.mutable_context_data() = gcc.context_data();
    cc1.mutable_context_data()->set_thread_group_id(102);
    cc1.mutable_context_data()->set_thread_group_start_time_ns(1020);
    cc1.mutable_context_data()->set_parent_thread_group_id(101);
    cc1.mutable_context_data()->set_process_name("renamed-frontend");
    cc1.set_binary_path(assembler ? "/usr/bin/x86_64-linux-gnu-as" : collect2 ?
        "/usr/lib/gcc/x86_64-linux-gnu/12/collect2" : "/usr/lib/gcc/x86_64-linux-gnu/12/cc1");
    cc1.set_execfn(assembler ? "/usr/bin/as" : cc1.binary_path()); cc1.add_argv(assembler ? "as" : cc1.binary_path());
    if (assembler && strcmp(scenario, "canonical-argv0") == 0) cc1.set_argv(0, "/usr/bin/as");
    if (native_linker) {
      if (!exec(cc1) || !ExpectRecordExact(output[1], id.c_str(), "process-exec-expected")) return false;
      if (!clone(cc1.context_data(), target_group, target_start)) return false;
      cc1.mutable_context_data()->set_thread_group_id(target_group);
      cc1.mutable_context_data()->set_thread_group_start_time_ns(target_start);
      cc1.mutable_context_data()->set_parent_thread_group_id(parent_group);
      cc1.set_binary_path("/usr/bin/x86_64-linux-gnu-ld.bfd");
      cc1.set_execfn("/usr/bin/ld"); cc1.set_argv(0, "/usr/bin/ld");
    }
    const bool expected = strcmp(scenario, "normal") == 0 || strcmp(scenario, "canonical-argv0") == 0;
    if (strcmp(scenario, "parent-image") == 0) state.groups[parent_group].current_image_class = ProcessClass::kUnknown;
    if (strcmp(scenario, "parent-role") == 0) state.groups[parent_group].role = ProcessState::Role::kControl;
    if (strcmp(scenario, "parent-start") == 0) state.groups[parent_group].start_time_ns = parent_group * 10 + 1;
    if (strcmp(scenario, "parent-inactive") == 0) state.groups[parent_group].go_build_tool_active = false;
    if (strcmp(scenario, "parent-pending") == 0) state.groups[parent_group].handoff_target_pending = true;
    if (strcmp(scenario, "gcc-image") == 0) state.groups[101].current_image_class = ProcessClass::kUnknown;
    if (strcmp(scenario, "gcc-role") == 0) state.groups[101].role = ProcessState::Role::kControl;
    if (strcmp(scenario, "gcc-start") == 0) state.groups[101].start_time_ns = 1011;
    if (strcmp(scenario, "gcc-inactive") == 0) state.groups[101].go_build_tool_active = false;
    if (strcmp(scenario, "gcc-pending") == 0) state.groups[101].handoff_target_pending = true;
    if (strcmp(scenario, "driver-image") == 0) state.groups[99].current_image_class = ProcessClass::kUnknown;
    if (strcmp(scenario, "driver-role") == 0) state.groups[99].role = ProcessState::Role::kControl;
    if (strcmp(scenario, "driver-start") == 0) state.groups[99].start_time_ns = 991;
    if (strcmp(scenario, "creator-context") == 0) cc1.mutable_context_data()->set_parent_thread_group_id(100);
    if (strcmp(scenario, "cgo-image") == 0) state.groups[100].current_image_class =
        ProcessClassForPath("/usr/local/go/pkg/tool/linux_amd64/compile", kProfileGoBuild);
    if (strcmp(scenario, "cgo-inactive") == 0) state.groups[100].go_build_tool_active = false;
    if (strcmp(scenario, "copy") == 0) cc1.set_binary_path("/tmp/cc1");
    if (strcmp(scenario, "argv0") == 0) cc1.set_argv(0, "cc1");
    if (strcmp(scenario, "execfn") == 0) cc1.set_execfn("/tmp/cc1");
    if (strcmp(scenario, "shadow") == 0) topology.anchors.emplace(2, MountAnchor{2, (assembler || native_linker) ? "/usr/bin" : "/usr/lib/gcc", "workspace"});
    if (strcmp(scenario, "unsealed") == 0) topology.sealed = false;
    if (strcmp(scenario, "writable-root") == 0) topology.expected[0].read_only = false;
    if (!exec(cc1)) return false;
    if (expected ? !ExpectRecordExact(output[1], id.c_str(), "process-exec-expected") :
        !ExpectUnexpectedProcessRecord(output[1], id.c_str(), "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
    if (state.groups[target_group].role != ProcessState::Role::kArtifact || state.groups[target_group].root_eligible ||
        state.expected_groups.count(target_group) != 0 || IsTrustedControlNetwork(cc1.context_data(), state)) return false;
    gvisor::syscall::Open open;
    *open.mutable_context_data() = cc1.context_data(); open.set_pathname("/etc/ld.so.cache"); open.set_flags(557056);
    MountAnchor image{1, "/", "oci-root"};
    auto readable = [&]() { return ClassifyFilesystemOpen(open, state, kProfileGoBuild, &image) == FilesystemClass::kHelperOnly; };
    if (readable() != expected) return false;
    if (sdk_cgo) {
      open.set_pathname("/usr/local/go/src/runtime/cgo/gcc_amd64.S"); open.set_flags(33024);
      if (readable() != expected) return false;
      open.set_pathname("/etc/ld.so.cache"); open.set_flags(557056);
    }
    if (!expected) return true;
    // Independent data-only DT_NEEDED inspection of locked ELF images supplies these ABI fixtures.
    const std::vector<const char*> loader_paths = native_linker ? std::vector<const char*>{
        "/usr/lib/x86_64-linux-gnu/libbfd-2.40-system.so", "/usr/lib/x86_64-linux-gnu/libctf.so.0",
        "/usr/lib/x86_64-linux-gnu/libjansson.so.4", "/usr/lib/x86_64-linux-gnu/libz.so.1",
        "/usr/lib/x86_64-linux-gnu/libzstd.so.1", "/usr/lib/x86_64-linux-gnu/libsframe.so.0",
        "/usr/lib/x86_64-linux-gnu/libc.so.6"} : collect2 ? std::vector<const char*>{
        "/usr/lib/x86_64-linux-gnu/libc.so.6", "/usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2"} : assembler ? std::vector<const char*>{
        "/usr/lib/x86_64-linux-gnu/libbfd-2.40-system.so", "/usr/lib/x86_64-linux-gnu/libsframe.so.0",
        "/usr/lib/x86_64-linux-gnu/libz.so.1", "/usr/lib/x86_64-linux-gnu/libzstd.so.1",
        "/usr/lib/x86_64-linux-gnu/libc.so.6"} : std::vector<const char*>{
        "/usr/lib/x86_64-linux-gnu/libisl.so.23", "/usr/lib/x86_64-linux-gnu/libmpc.so.3",
        "/usr/lib/x86_64-linux-gnu/libmpfr.so.6", "/usr/lib/x86_64-linux-gnu/libgmp.so.10",
        "/usr/lib/x86_64-linux-gnu/libz.so.1", "/usr/lib/x86_64-linux-gnu/libzstd.so.1",
        "/usr/lib/x86_64-linux-gnu/libm.so.6", "/usr/lib/x86_64-linux-gnu/libc.so.6",
        "/usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2"};
    for (const char* path : loader_paths) {
      open.set_pathname(path); if (!readable()) return false;
      open.set_flags(kOpenWriteOnly); if (readable()) return false;
      open.set_flags(557056);
    }
    if (runtime_data) {
      // Independently copied fixed OCI ELF objects, archives and linker scripts.
      for (const char* path : {
          "/usr/lib/gcc/x86_64-linux-gnu/12/crtbegin.o", "/usr/lib/gcc/x86_64-linux-gnu/12/crtbeginS.o",
          "/usr/lib/gcc/x86_64-linux-gnu/12/crtbeginT.o", "/usr/lib/gcc/x86_64-linux-gnu/12/crtend.o",
          "/usr/lib/gcc/x86_64-linux-gnu/12/crtendS.o", "/usr/lib/gcc/x86_64-linux-gnu/12/libgcc.a",
          "/usr/lib/gcc/x86_64-linux-gnu/12/libgcc_eh.a", "/usr/lib/gcc/x86_64-linux-gnu/12/libgcc_s.so",
          "/usr/lib/x86_64-linux-gnu/Scrt1.o", "/usr/lib/x86_64-linux-gnu/crt1.o",
          "/usr/lib/x86_64-linux-gnu/rcrt1.o", "/usr/lib/x86_64-linux-gnu/crti.o", "/usr/lib/x86_64-linux-gnu/crtn.o",
          "/usr/lib/x86_64-linux-gnu/libc.so", "/usr/lib/x86_64-linux-gnu/libc_nonshared.a",
          "/usr/lib/x86_64-linux-gnu/libpthread.a", "/usr/lib/x86_64-linux-gnu/libpthread_nonshared.a",
          "/usr/lib/x86_64-linux-gnu/libdl.a", "/usr/lib/x86_64-linux-gnu/libgcc_s.so.1",
          "/usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2"}) {
        open.set_pathname(path); if (!readable()) return false;
        open.set_flags(kOpenWriteOnly); if (readable()) return false;
        open.set_flags(557056);
      }
    }
    if (resolver_abi) {
      open.set_pathname("/usr/lib/x86_64-linux-gnu/libresolv.so.2"); open.set_flags(32768);
      if (!readable()) return false;
      open.set_flags(kOpenWriteOnly); if (readable()) return false;
      open.set_flags(557056);
    }
    if (!native_linker) {
      open.set_pathname("/usr/lib/x86_64-linux-gnu/libresolv.so.2"); if (readable()) return false;
    }
    if (!native_linker) {
      open.set_pathname("/usr/lib/x86_64-linux-gnu/Scrt1.o"); if (readable()) return false;
    }
    if (null_device) {
      MountAnchor device{2, "/dev", "system"};
      open.set_pathname("/dev/null"); open.set_flags(kOpenReadWrite | kOpenCreate | kOpenTruncate | kOpenLargefile);
      auto device_sink = [&]() { return ClassifyFilesystemOpen(open, state, kProfileGoBuild, &device) == FilesystemClass::kHelperOnly; };
      if (!device_sink()) return false;
      open.set_flags(kOpenReadWrite | kOpenCreate | kOpenTruncate); if (!device_sink()) return false;
      for (uint64_t flags : {kOpenWriteOnly, kOpenReadWrite, kOpenReadWrite | kOpenAppend,
          kOpenReadWrite | kOpenCreate | kOpenTruncate | kOpenAppend}) {
        open.set_flags(flags); if (device_sink()) return false;
      }
      open.set_flags(kOpenReadWrite | kOpenCreate | kOpenTruncate | kOpenLargefile);
      open.set_pathname("/dev/zero"); if (device_sink()) return false;
      open.set_pathname("/dev/null");
      device.mount_class = "workspace"; if (device_sink()) return false;
      device.mount_class = "system"; device.mountpoint = "/"; if (device_sink()) return false;
      device.mountpoint = "/dev";
      if (ClassifyFilesystemOpen(open, state, kProfileGoResolver, &device) == FilesystemClass::kHelperOnly ||
          ClassifyFilesystemOpen(open, state, kProfileGoBuild) == FilesystemClass::kHelperOnly) return false;
      open.mutable_context_data()->set_thread_group_start_time_ns(target_start + 1); if (device_sink()) return false;
      open.mutable_context_data()->set_thread_group_start_time_ns(target_start);
      state.groups[parent_group].go_build_tool_active = false; if (device_sink()) return false;
      state.groups[parent_group].go_build_tool_active = true;
      open.set_flags(557056);
    }
    if (linker_plugin) {
      open.set_pathname("/usr/lib/gcc/x86_64-linux-gnu/12/liblto_plugin.so"); if (!readable()) return false;
      open.set_flags(kOpenWriteOnly); if (readable()) return false;
      open.set_flags(557056);
    }
    if (!native_linker) {
      open.set_pathname("/usr/lib/gcc/x86_64-linux-gnu/12/liblto_plugin.so"); if (readable()) return false;
    }
    if (assembler || collect2) {
      for (const char* path : {"/usr/include/stdc-predef.h", "/usr/lib/gcc/x86_64-linux-gnu/12/include/stddef.h",
          "/usr/lib/x86_64-linux-gnu/libisl.so.23"}) {
        open.set_pathname(path); if (readable()) return false;
      }
    }
    if (sdk_cgo) {
      for (const char* path : {"/usr/local/go/src/runtime/cgo", "/usr/local/go/src/runtime/cgo/gcc_amd64.S",
          "/usr/local/go/src/runtime/cgo/gcc_libinit.c", "/usr/local/go/src/runtime/cgo/abi_amd64.h"}) {
        open.set_pathname(path); open.set_flags(33024); if (!readable()) return false;
        open.set_flags(kOpenWriteOnly); if (readable()) return false;
      }
      open.set_pathname("/usr/local/go/src/runtime/cgo/gcc_amd64.S"); open.set_flags(33024);
      image.mount_class = "workspace"; if (readable()) return false;
      image.mount_class = "oci-root"; image.mountpoint = "/usr/local/go/src/runtime/cgo"; if (readable()) return false;
      image.mountpoint = "/";
      if (ClassifyFilesystemOpen(open, state, kProfileGoResolver, &image) == FilesystemClass::kHelperOnly ||
          ClassifyFilesystemOpen(open, state, kProfileGoBuild) == FilesystemClass::kHelperOnly) return false;
      state.groups[parent_group].go_build_tool_active = false; if (readable()) return false;
      state.groups[parent_group].go_build_tool_active = true;
      open.set_flags(557056);
    }
    if (assembler || collect2) {
      open.set_pathname("/usr/local/go/src/runtime/cgo/gcc_amd64.S"); if (readable()) return false;
    }
    if (check_headers) {
      for (const char* path : {"/usr/include", "/usr/include/stdc-predef.h", "/usr/include/x86_64-linux-gnu/bits/types.h",
          "/usr/lib/gcc/x86_64-linux-gnu/12/include", "/usr/lib/gcc/x86_64-linux-gnu/12/include/stddef.h"}) {
        open.set_pathname(path); if (!readable()) return false;
        open.set_flags(kOpenWriteOnly); if (readable()) return false;
        open.set_flags(557056);
      }
    }
    for (const char* path : {"/etc/passwd", "/usr/lib/x86_64-linux-gnu/libpython3.14.so.1.0",
        "/usr/local/go/src/runtime/runtime.go", "/usr/local/go/src/runtime/cgo-other/gcc_amd64.S",
        "/usr/local/go/pkg/include/textflag.h", "/usr/include-other/stdio.h",
        "/usr/lib/gcc/x86_64-linux-gnu/12/include-other/stddef.h", "/usr/lib/gcc/x86_64-linux-gnu/12/specs", "/usr/lib/gcc/x86_64-linux-gnu/12/libcc1.so", "/usr/lib/x86_64-linux-gnu/arbitrary.o", "/usr/lib/gcc/x86_64-linux-gnu/12/crtbeginS.o.other", "/usr/lib/x86_64-linux-gnu/libunknown.a", "/usr/lib/x86_64-linux-gnu/libresolv-other.so.2", "/proc/102/cgroup"}) {
      open.set_pathname(path); if (readable()) return false;
    }
    open.set_pathname("/etc/ld.so.cache");
    image.mountpoint = "/usr/lib"; if (readable()) return false;
    image.mountpoint = "/";
    open.mutable_context_data()->set_thread_group_start_time_ns(target_start + 1); if (readable()) return false;
    open.mutable_context_data()->set_thread_group_start_time_ns(target_start);
    if (!clone(cc1.context_data(), target_group + 1, target_start + 10)) return false;
    auto descendant = cc1;
    descendant.mutable_context_data()->set_thread_group_id(target_group + 1);
    descendant.mutable_context_data()->set_thread_group_start_time_ns(target_start + 10);
    descendant.mutable_context_data()->set_parent_thread_group_id(target_group);
    if (!exec(descendant) || !ExpectUnexpectedProcessRecord(output[1], id.c_str(),
        "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
    if (!exec(cc1) || !ExpectUnexpectedProcessRecord(output[1], id.c_str(),
        "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
    return !readable();
  };
  bool ok = run("normal");
  for (const char* scenario : {"canonical-argv0", "parent-image", "parent-role", "parent-start", "parent-inactive", "parent-pending",
       "driver-image", "driver-role", "driver-start", "creator-context", "cgo-image", "cgo-inactive",
       "copy", "argv0", "execfn", "shadow", "unsealed", "writable-root"}) ok = ok && run(scenario);
  if (native_linker) {
    for (const char* scenario : {"gcc-image", "gcc-role", "gcc-start", "gcc-inactive", "gcc-pending"}) ok = ok && run(scenario);
  }
  close(output[0]); close(output[1]);
  return ok;
}

bool VerifyFilesystemClassification() {
  ProcessState state;
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID);
  context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990);
  context.set_parent_thread_group_id(98);
  if (!RegisterGroup(&state, context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kCloneChild, false, true)) return false;
  gvisor::syscall::Open open;
  *open.mutable_context_data() = context;
  state.expected_groups[context.thread_group_id()] =
      ProcessState::ExpectedGroup{context.thread_group_start_time_ns(), ProcessClass::kNpm};
  open.set_flags(0);
  open.set_pathname("/usr/local/lib/node_modules/npm/lib/npm.js");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  open.mutable_context_data()->set_process_name("npm install /tm");
  for (const char* path : {"/etc/localtime", "/etc/nsswitch.conf", "/etc/resolv.conf",
                           "/etc/netsvc.conf", "/etc/svc.conf", "/usr/bin/ldd"}) {
    open.set_pathname(path);
    if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  }
  open.mutable_context_data()->set_process_name("npm");
  open.set_pathname("/etc/localtime");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  state.expected_groups[context.thread_group_id()].process_class = ProcessClass::kNode;
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  state.expected_groups[context.thread_group_id()].process_class = ProcessClass::kNpm;
  open.set_pathname("/etc/localtime");
  if (ClassifyFilesystemOpen(open, state, kProfilePyPI) != FilesystemClass::kUnknown) return false;
  open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kOutside) return false;
  open.set_flags(0);
  state.expected_groups[context.thread_group_id()].process_class = ProcessClass::kUnknown;
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  state.expected_groups[context.thread_group_id()].process_class = ProcessClass::kNpm;
  open.mutable_context_data()->set_process_name("");
  open.set_pathname("/usr/local/bin/docker-entrypoint.sh");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kOutside) return false;
  open.set_flags(0);
  open.set_pathname("/usr/local/bin/docker-entrypoint-other.sh");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  open.set_pathname("/usr/local/bin/docker-entrypoint.sh");
  if (ClassifyFilesystemOpen(open, state, kProfilePyPI) != FilesystemClass::kUnknown) return false;
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  open.mutable_context_data()->set_process_name("chown");
  state.expected_groups[context.thread_group_id()].process_class = ProcessClass::kUnknown;
  for (const char* path : {"/etc/nsswitch.conf", "/etc/passwd", "/etc/group"}) {
    open.set_pathname(path);
    if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  }
  open.set_pathname("/etc/shadow");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  open.set_pathname("/etc/nsswitch.conf");
  if (ClassifyFilesystemOpen(open, state, kProfilePyPI) != FilesystemClass::kUnknown) return false;
  open.mutable_context_data()->set_process_name("chmod");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  open.mutable_context_data()->set_process_name("chown");
  open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kOutside) return false;
  open.set_flags(0);
  open.mutable_context_data()->set_process_name("");
  state.expected_groups[context.thread_group_id()].process_class = ProcessClass::kNpm;
  const auto before = state.groups.find(context.thread_group_id())->second;
  const size_t group_count = state.groups.size();
  if (state.groups.size() != group_count ||
      state.groups.find(context.thread_group_id())->second.role != before.role ||
      state.groups.find(context.thread_group_id())->second.provenance != before.provenance ||
      state.groups.find(context.thread_group_id())->second.root_eligible != before.root_eligible ||
      state.groups.find(context.thread_group_id())->second.root_consumed != before.root_consumed ||
      state.groups.find(context.thread_group_id())->second.trusted_control_network_active != before.trusted_control_network_active) return false;

  auto missing_group = open;
  missing_group.mutable_context_data()->set_thread_group_id(100);
  if (ClassifyFilesystemOpen(missing_group, state, kProfileNPM) != FilesystemClass::kUnknown ||
      state.groups.size() != group_count) return false;
  gvisor::common::ContextData unknown_context = context;
  unknown_context.set_thread_group_id(101);
  unknown_context.set_thread_group_start_time_ns(1010);
  if (!RegisterGroup(&state, unknown_context, ProcessState::Role::kUnknown,
                     ProcessState::Provenance::kUnknown, false, true)) return false;
  auto unknown_group = open;
  unknown_group.mutable_context_data()->set_thread_group_id(101);
  if (ClassifyFilesystemOpen(unknown_group, state, kProfileNPM) != FilesystemClass::kUnknown) return false;

  ProcessState oci_state;
  gvisor::common::ContextData oci_context = context;
  oci_context.set_thread_group_id(102);
  oci_context.set_thread_group_start_time_ns(1020);
  oci_context.set_parent_thread_group_id(0);
  if (!RegisterGroup(&oci_state, oci_context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kOCIRoot, false, true)) return false;
  oci_state.bootstrap_group_set = true;
  oci_state.bootstrap_active = true;
  oci_state.bootstrap_group_id = oci_context.thread_group_id();
  oci_state.bootstrap_group_start_time_ns = oci_context.thread_group_start_time_ns();
  oci_state.groups.find(oci_context.thread_group_id())->second.oci_bootstrap_stage =
      ProcessState::OCIBootstrapStage::kAwaitingBootstrapShell;
  auto oci_open = open;
  *oci_open.mutable_context_data() = oci_context;
  oci_open.set_pathname("/etc/ld.so.cache");
  if (ClassifyFilesystemOpen(oci_open, oci_state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  if (ClassifyFilesystemOpen(oci_open, oci_state, kProfileGitHub) != FilesystemClass::kHelperOnly) return false;
  if (ClassifyFilesystemOpen(oci_open, oci_state, kProfilePyPI) != FilesystemClass::kHelperOnly) return false;
  oci_open.set_pathname("/usr/local/bin/docker-entrypoint.sh");
  if (ClassifyFilesystemOpen(oci_open, oci_state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  if (ClassifyFilesystemOpen(oci_open, oci_state, kProfileGitHub) != FilesystemClass::kHelperOnly) return false;

  ProcessState elf_handoff_state;
  gvisor::common::ContextData elf_context = context;
  elf_context.set_thread_group_id(108);
  elf_context.set_thread_group_start_time_ns(1080);
  elf_context.set_parent_thread_group_id(0);
  elf_context.set_is_exec_session(true);
  elf_context.set_process_name("haa-boundary");
  if (!RegisterGroup(&elf_handoff_state, elf_context, ProcessState::Role::kArtifact,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  auto& elf_group = elf_handoff_state.groups.find(108)->second;
  elf_group.demotion_pending = true;
  elf_group.handoff_target_pending = true;
  gvisor::syscall::Open elf_open;
  *elf_open.mutable_context_data() = elf_context;
  elf_open.set_flags(524288);
  for (const char* path : {"/etc/ld.so.cache", "/lib/x86_64-linux-gnu/libc.so.6",
                           "/haa-runtime/haa-boundary"}) {
    elf_open.set_pathname(path);
    if (ClassifyFilesystemOpen(elf_open, elf_handoff_state, kProfileGitHub) !=
        FilesystemClass::kHelperOnly) return false;
  }
  elf_group.demotion_pending = false;
  elf_open.mutable_context_data()->set_process_name("setpriv");
  for (const char* path : {"/etc/ld.so.cache", "/lib/x86_64-linux-gnu/libcap-ng.so.0",
                           "/lib/x86_64-linux-gnu/libc.so.6", "/proc/sys/kernel/cap_last_cap",
                           "/etc/nsswitch.conf", "/etc/passwd", "/etc/group",
                           "/proc/108/status"}) {
    elf_open.set_pathname(path);
    if (ClassifyFilesystemOpen(elf_open, elf_handoff_state, kProfileGitHub) !=
        FilesystemClass::kHelperOnly) return false;
  }
  const auto elf_before = elf_group;
  elf_open.set_pathname("/etc/shadow");
  if (ClassifyFilesystemOpen(elf_open, elf_handoff_state, kProfileGitHub) != FilesystemClass::kUnknown) return false;
  elf_open.set_pathname("/etc/ld.so.cache");
  elf_open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(elf_open, elf_handoff_state, kProfileGitHub) != FilesystemClass::kOutside) return false;
  elf_open.set_flags(0);
  elf_group.handoff_target_pending = false;
  // The exact immutable runtime file stays telemetry-only after the handoff;
  // this is a pathname classification, not a temporary loader privilege.
  if (ClassifyFilesystemOpen(elf_open, elf_handoff_state, kProfileGitHub) != FilesystemClass::kHelperOnly) return false;
  elf_group.handoff_target_pending = true;
  elf_group.provenance = ProcessState::Provenance::kCloneChild;
  if (ClassifyFilesystemOpen(elf_open, elf_handoff_state, kProfileGitHub) != FilesystemClass::kHelperOnly) return false;
  elf_group = elf_before;
  auto wrong_elf_start = elf_open;
  wrong_elf_start.mutable_context_data()->set_thread_group_start_time_ns(1081);
  if (ClassifyFilesystemOpen(wrong_elf_start, elf_handoff_state, kProfileGitHub) != FilesystemClass::kUnknown) return false;
  if (elf_group.role != elf_before.role || elf_group.provenance != elf_before.provenance ||
      elf_group.root_eligible != elf_before.root_eligible ||
      elf_group.root_consumed != elf_before.root_consumed ||
      elf_group.trusted_control_network_active != elf_before.trusted_control_network_active) return false;

  // PyPI has a separately pinned Python image. Runtime-root classification is
  // filesystem-only and must neither depend on nor change CONTROL/ARTIFACT.
  ProcessState python_runtime_state;
  gvisor::common::ContextData python_runtime_context = context;
  python_runtime_context.set_thread_group_id(111);
  python_runtime_context.set_thread_group_start_time_ns(1110);
  python_runtime_context.set_process_name("python3.14");
  if (!RegisterGroup(&python_runtime_state, python_runtime_context,
                     ProcessState::Role::kControl,
                     ProcessState::Provenance::kCloneChild, false, true)) return false;
  python_runtime_state.groups.find(111)->second.current_image_class = ProcessClass::kPython;
  const auto python_before = python_runtime_state.groups.find(111)->second;
  gvisor::syscall::Open python_runtime_open;
  *python_runtime_open.mutable_context_data() = python_runtime_context;
  python_runtime_open.set_flags(0);
  python_runtime_open.set_pathname("/usr/local/bin/../lib/glibc-hwcaps/x86-64-v3/libpython3.14.so.1.0");
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) != FilesystemClass::kRuntimeRoot) return false;
  python_runtime_open.set_pathname("/usr/local/bin/../lib/glibc-hwcaps/x86-64-v2/libpython3.14.so.1.0");
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) != FilesystemClass::kRuntimeRoot) return false;
  for (const char* path : {
           "/usr/local/lib/python3.14/importlib/__init__.py",
           "/usr/local/lib/python3.14/__pycache__/os.cpython-314.pyc",
           "/usr/local/lib/python3.14/lib-dynload/_ssl.cpython-314-x86_64-linux-gnu.so",
           "/usr/local/lib/python3.14/site-packages/pip/__init__.py",
           "/usr/local/lib/python3.14/site-packages/pip-26.2.1.dist-info/METADATA",
           "/lib/x86_64-linux-gnu/libssl.so.3"}) {
    python_runtime_open.set_pathname(path);
    if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) !=
        FilesystemClass::kRuntimeRoot) return false;
  }
  // The same immutable read remains filesystem runtime material after role
  // demotion; classification does not restore CONTROL.
  python_runtime_state.groups.find(111)->second.role = ProcessState::Role::kArtifact;
  python_runtime_open.set_pathname("/usr/local/lib/python3.14/os.py");
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) != FilesystemClass::kRuntimeRoot) return false;
  for (const char* profile : {kProfilePyTorchCPU, kProfilePyTorchCU126,
                             kProfilePyTorchCU130, kProfilePyTorchCU132}) {
    python_runtime_open.mutable_context_data()->set_process_name("worker");
    python_runtime_open.set_pathname("/proc/111/maps");
    if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, profile) != FilesystemClass::kRuntimeRoot) return false;
    for (const char* library : {"/lib/x86_64-linux-gnu/libutil.so.1",
                                "/usr/lib/x86_64-linux-gnu/libutil.so.1"}) {
      python_runtime_open.set_pathname(library);
      if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, profile) != FilesystemClass::kRuntimeRoot) return false;
    }
    for (const char* unknown : {"/usr/lib/x86_64-linux-gnu/libutil.so.2",
                                "/usr/lib/x86_64-linux-gnu/libutil.so.1evil"}) {
      python_runtime_open.set_pathname(unknown);
      if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, profile) == FilesystemClass::kRuntimeRoot) return false;
    }
    python_runtime_open.set_pathname("/usr/lib/x86_64-linux-gnu/libutil.so.1");
    // A claimed Python name cannot supply a missing/different executable.
    python_runtime_open.mutable_context_data()->set_process_name("python");
    for (ProcessClass image : {ProcessClass::kUnknown, ProcessClass::kShell, ProcessClass::kArtifact}) {
      python_runtime_state.groups.find(111)->second.current_image_class = image;
      if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, profile) == FilesystemClass::kRuntimeRoot) return false;
    }
    python_runtime_state.groups.find(111)->second.current_image_class = ProcessClass::kPython;
    auto replaced = python_runtime_open;
    replaced.mutable_context_data()->set_thread_group_start_time_ns(1111);
    if (ClassifyFilesystemOpen(replaced, python_runtime_state, profile) == FilesystemClass::kRuntimeRoot) return false;
    python_runtime_open.set_flags(kOpenWriteOnly);
    if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, profile) != FilesystemClass::kOutside) return false;
    python_runtime_open.set_flags(0);
  }
  python_runtime_open.mutable_context_data()->set_process_name("python3.14");

  python_runtime_open.set_pathname("/usr/local/lib/glibc-hwcaps/x86-64-v2/libpython3.14.so.1.0");
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) != FilesystemClass::kUnknown) return false;
  python_runtime_open.set_pathname("/usr/local/lib/glibc-hwcaps/x86-64-v3/libpython3.14.so.1.1");
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) != FilesystemClass::kUnknown) return false;
  python_runtime_open.set_pathname("/usr/local/lib/glibc-hwcaps/x86-64-v3/libpython3.14.so.1.0");
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfileGitHub) != FilesystemClass::kUnknown) return false;
  for (const char* path : {
           "/usr/local/lib/python3.15/os.py",
           "/usr/local/lib/python3.14/site-packages/artifact_owned.py",
           "/usr/local/lib/python3.14/site-packages/setuptools/__init__.py",
           "/usr/local/lib/glibc-hwcaps/x86-64-v3/not-libpython.so",
           "/usr/local/lib/../lib/python3.14/os.py",
           "/usr/local//lib/python3.14/os.py",
           "/../../usr/local/lib/python3.14/os.py"}) {
    python_runtime_open.set_pathname(path);
    if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) !=
        FilesystemClass::kUnknown) return false;
  }
  python_runtime_open.set_pathname("/tmp/haa-buildenv/lib/python3.14/site-packages/backend.py");
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) != FilesystemClass::kWorkspace) return false;
  python_runtime_open.set_pathname("/haa-site/example/__init__.py");
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) != FilesystemClass::kWorkspace) return false;
  python_runtime_open.set_pathname("/usr/local/lib/python3.14/os.py");
  python_runtime_open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) != FilesystemClass::kOutside) return false;
  python_runtime_open.set_flags(0);
  python_runtime_open.set_pathname("/tmp/.haa-honeytoken");
  if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) != FilesystemClass::kHoneytoken) return false;
  for (const char* path : {
            "/proc/cpuinfo", "/proc/mounts", "/proc/self/environ",
            "/proc/self/maps", "/proc/self/status", "/proc/self/mounts",
            "/proc/111/environ", "/proc/111/maps", "/proc/111/status", "/proc/111/mounts",
            "/proc/sys/vm/overcommit_memory", "/proc/sys/kernel/random/boot_id", "/sys/devices/system/cpu/possible",
            "/dev/null", "/dev/urandom"}) {
    python_runtime_open.set_pathname(path);
    if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) !=
        FilesystemClass::kRuntimeRoot) return false;
  }
  for (const char* path : {
           "/proc/self/fd/1", "/proc/self/fdinfo/1", "/proc/self/root/etc/passwd",
           "/proc/self/task/111/status", "/proc/self/ns/mnt", "/proc/1/status",
           "/proc/112/maps", "/proc/112/mounts", "/sys/devices/system/cpu/cpu0/online",
           "/sys/devices/system/node/node0/meminfo"}) {
    python_runtime_open.set_pathname(path);
    if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) ==
        FilesystemClass::kRuntimeRoot) return false;
  }
  for (const char* path : {"/tmp/libstdc++.so.6", "/haa-site/libgomp.so"}) {
    python_runtime_open.set_pathname(path);
    if (ClassifyFilesystemOpen(python_runtime_open, python_runtime_state, kProfilePyPI) ==
        FilesystemClass::kRuntimeRoot) return false;
  }
  const auto& python_after = python_runtime_state.groups.find(111)->second;
  if (python_after.role != ProcessState::Role::kArtifact ||
      python_after.provenance != python_before.provenance ||
      python_after.root_eligible != python_before.root_eligible ||
      python_after.root_consumed != python_before.root_consumed ||
      python_after.trusted_control_network_active != python_before.trusted_control_network_active) return false;

  ProcessState artifact_state;
  gvisor::common::ContextData artifact_context = context;
  artifact_context.set_thread_group_id(103);
  artifact_context.set_thread_group_start_time_ns(1030);
  if (!RegisterGroup(&artifact_state, artifact_context, ProcessState::Role::kArtifact,
                     ProcessState::Provenance::kCloneChild, false, true)) return false;
  auto artifact_open = open;
  *artifact_open.mutable_context_data() = artifact_context;
  artifact_open.set_pathname("/root/artifact-outside");
  if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kOutside) return false;
  artifact_open.set_pathname("/usr/local/bin/docker-entrypoint.sh");
  if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  artifact_open.mutable_context_data()->set_process_name("chown");
  artifact_open.set_pathname("/etc/nsswitch.conf");
  if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  artifact_open.mutable_context_data()->set_process_name("haa-boundary");
  for (const char* path : {"/haa-runtime/haa-boundary", "/proc/self/status", "/etc/ld.so.cache",
                           "/lib/x86_64-linux-gnu/libc.so.6", "/proc/103/status"}) {
    artifact_open.set_pathname(path);
    if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  }
  artifact_open.set_pathname("/lib/x86_64-linux-gnu/libm.so.6");
  if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  artifact_open.set_pathname("/proc/102/status");
  if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  artifact_open.set_pathname("/proc/103/status");
  artifact_open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kOutside) return false;
  artifact_open.set_flags(0);
  artifact_open.mutable_context_data()->set_process_name("sh");
  artifact_open.set_pathname("/etc/ld.so.cache");
  if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  artifact_open.set_pathname("/lib/x86_64-linux-gnu/libc.so.6");
  if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  artifact_open.set_pathname("/lib/x86_64-linux-gnu/libm.so.6");
  if (ClassifyFilesystemOpen(artifact_open, artifact_state, kProfileNPM) != FilesystemClass::kUnknown) return false;

  auto wrong_start_open = open;
  wrong_start_open.mutable_context_data()->set_thread_group_start_time_ns(991);
  if (ClassifyFilesystemOpen(wrong_start_open, state, kProfileNPM) != FilesystemClass::kUnknown) return false;

  ProcessState pre_sentry_state;
  gvisor::syscall::Open pre_sentry;
  pre_sentry.mutable_context_data()->set_container_id(kFirstID);
  pre_sentry.mutable_context_data()->set_thread_group_id(105);
  pre_sentry.mutable_context_data()->set_thread_group_start_time_ns(1050);
  pre_sentry.mutable_context_data()->set_parent_thread_group_id(0);
  pre_sentry.mutable_context_data()->set_is_exec_session(true);
  pre_sentry.mutable_context_data()->set_process_name("haa-boundary");
  pre_sentry.set_flags(524288);
  pre_sentry.set_pathname("/etc/ld.so.cache");
  if (ClassifyFilesystemOpen(pre_sentry, pre_sentry_state, kProfileNPM) != FilesystemClass::kHelperOnly ||
      !pre_sentry_state.groups.empty()) return false;
  if (ClassifyFilesystemOpen(pre_sentry, pre_sentry_state, kProfileGitHub) != FilesystemClass::kHelperOnly ||
      !pre_sentry_state.groups.empty()) return false;
  for (const char* supported_profile : {kProfileNPM, kProfilePyPI,
                                        kProfilePyTorchCPU,
                                        kProfilePyTorchCU126,
                                        kProfileGitHub}) {
    if (ClassifyFilesystemOpen(pre_sentry, pre_sentry_state, supported_profile) !=
            FilesystemClass::kHelperOnly ||
        !pre_sentry_state.groups.empty()) return false;
  }
  pre_sentry.set_pathname("/etc/passwd");
  if (ClassifyFilesystemOpen(pre_sentry, pre_sentry_state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  pre_sentry.set_pathname("/etc/ld.so.cache");
  pre_sentry.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(pre_sentry, pre_sentry_state, kProfileNPM) != FilesystemClass::kUnknown) return false;

  ProcessState direct_demotion_state;
  auto direct_demotion_context = pre_sentry.context_data();
  if (!RegisterGroup(&direct_demotion_state, direct_demotion_context,
                     ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  auto& direct_demotion_group =
      direct_demotion_state.groups.find(direct_demotion_context.thread_group_id())->second;
  direct_demotion_group.demotion_pending = true;
  auto direct_demotion_open = pre_sentry;
  direct_demotion_open.set_flags(0);
  for (const char* path : {"/etc/ld.so.cache", "/lib/x86_64-linux-gnu/libc.so.6",
                           "/haa-runtime/haa-boundary"}) {
    direct_demotion_open.set_pathname(path);
    for (const char* supported_profile : {kProfileNPM, kProfilePyPI,
                                          kProfilePyTorchCPU,
                                          kProfilePyTorchCU126,
                                          kProfileGitHub}) {
      if (ClassifyFilesystemOpen(direct_demotion_open, direct_demotion_state,
                                 supported_profile) != FilesystemClass::kHelperOnly) return false;
    }
  }
  direct_demotion_group.demotion_pending = false;
  direct_demotion_open.set_pathname("/haa-runtime/haa-boundary");
  if (ClassifyFilesystemOpen(direct_demotion_open, direct_demotion_state,
                             kProfilePyPI) != FilesystemClass::kUnknown) return false;
  direct_demotion_group.demotion_pending = true;
  direct_demotion_open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(direct_demotion_open, direct_demotion_state,
                             kProfilePyPI) != FilesystemClass::kOutside) return false;

  ProcessState direct_child_state;
  gvisor::common::ContextData direct_parent_context = context;
  direct_parent_context.set_thread_group_id(109);
  direct_parent_context.set_thread_group_start_time_ns(1090);
  direct_parent_context.set_parent_thread_group_id(0);
  direct_parent_context.set_is_exec_session(true);
  if (!RegisterGroup(&direct_child_state, direct_parent_context,
                     ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  gvisor::common::ContextData direct_child_context = context;
  direct_child_context.set_thread_group_id(110);
  direct_child_context.set_thread_group_start_time_ns(1100);
  direct_child_context.set_parent_thread_group_id(109);
  direct_child_context.set_is_exec_session(true);
  direct_child_context.set_process_name("id");
  if (!RegisterGroup(&direct_child_state, direct_child_context,
                     ProcessState::Role::kControl,
                     ProcessState::Provenance::kCloneChild, false, true)) return false;
  // Mirror SENTRY_CLONE provenance: filesystem admission for this special
  // control child requires the creator's complete execution identity.
  auto& direct_child_group = direct_child_state.groups.find(110)->second;
  direct_child_group.clone_creator_group_id = 109;
  direct_child_group.clone_creator_group_start_time_ns = 1090;
  gvisor::syscall::Open direct_child_open;
  *direct_child_open.mutable_context_data() = direct_child_context;
  direct_child_open.set_flags(524288);
  direct_child_open.set_pathname("/proc/filesystems");
  for (const char* supported_profile : {kProfileNPM, kProfilePyPI,
                                        kProfilePyTorchCPU,
                                        kProfilePyTorchCU126,
                                        kProfileGitHub}) {
    if (ClassifyFilesystemOpen(direct_child_open, direct_child_state,
                               supported_profile) != FilesystemClass::kHelperOnly) return false;
  }
  direct_child_open.set_pathname("/proc/mounts");
  if (ClassifyFilesystemOpen(direct_child_open, direct_child_state,
                             kProfilePyPI) != FilesystemClass::kUnknown) return false;
  direct_child_open.set_pathname("/proc/filesystems");
  direct_child_open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(direct_child_open, direct_child_state,
                             kProfilePyPI) != FilesystemClass::kOutside) return false;

  auto& oci_group = oci_state.groups.find(oci_context.thread_group_id())->second;
  oci_group.oci_bootstrap_stage = ProcessState::OCIBootstrapStage::kAwaitingDemotion;
  oci_open.set_pathname("/haa-runtime/.haa-boundary.tmp");
  oci_open.set_flags(577);
  if (ClassifyFilesystemOpen(oci_open, oci_state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  oci_open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(oci_open, oci_state, kProfileNPM) != FilesystemClass::kOutside) return false;

  ProcessState setpriv_state;
  gvisor::common::ContextData setpriv_context = context;
  setpriv_context.set_thread_group_id(106);
  setpriv_context.set_thread_group_start_time_ns(1060);
  setpriv_context.set_process_name("setpriv");
  if (!RegisterGroup(&setpriv_state, setpriv_context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  gvisor::syscall::Open self_status;
  *self_status.mutable_context_data() = setpriv_context;
  self_status.set_flags(524288);
  self_status.set_pathname("/proc/106/status");
  if (ClassifyFilesystemOpen(self_status, setpriv_state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  self_status.set_pathname("/proc/107/status");
  if (ClassifyFilesystemOpen(self_status, setpriv_state, kProfileNPM) != FilesystemClass::kUnknown) return false;

  state.expected_groups[context.thread_group_id()].process_class = ProcessClass::kNode;
  open.mutable_context_data()->set_process_name("node");
  for (const char* path : {"/proc/version_signature", "/proc/meminfo", "/proc/self/cgroup",
                           "/proc/self/maps", "/proc/sys/vm/overcommit_memory",
                           "/sys/devices/system/cpu/online"}) {
    open.set_pathname(path);
    open.set_flags(524288);
    if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  }
  open.set_pathname(std::string("/sys/fs/cgroup/memory/") + kFirstID + "/memory.limit_in_bytes");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kHelperOnly) return false;
  open.set_pathname("/proc/cpuinfo");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  open.set_pathname("/proc/version_signature");
  open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kOutside) return false;

  // The weaker filesystem lookup must not leak into Sentry or clone identity.
  gvisor::common::ContextData wrong_start = context;
  wrong_start.set_thread_group_start_time_ns(991);
  ProcessClassification sentry = IsExpectedProcess("/usr/local/bin/node", wrong_start, kProfileNPM, &state);
  if (sentry.expected || strcmp(sentry.reason, "PROCESS_PROVENANCE_UNKNOWN") != 0) return false;
  gvisor::sentry::CloneInfo clone;
  *clone.mutable_context_data() = wrong_start;
  clone.set_created_thread_group_id(104);
  clone.set_created_thread_start_time_ns(1040);
  std::string encoded;
  if (!clone.SerializeToString(&encoded)) return false;
  std::string container_id = kFirstID;
  const char* reason = nullptr;
  if (ParseSentryClone(encoded.data(), encoded.size(), -1, &container_id, &state, &reason) ||
      reason == nullptr || strcmp(reason, "CLONE_PROVENANCE_INVALID") != 0) return false;

  open.set_flags(0);
  open.set_pathname("/opt/unclassified");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kUnknown) return false;
  open.set_pathname("/root/.ssh/id_rsa");
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kOutside) return false;
  open.set_pathname("/usr/local/lib/node_modules/npm/lib/npm.js");
  open.set_flags(kOpenWriteOnly);
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kOutside) return false;
  open.set_pathname("/tmp/.haa-honeytoken");
  open.set_flags(0);
  if (ClassifyFilesystemOpen(open, state, kProfileNPM) != FilesystemClass::kHoneytoken) return false;
  return !state.groups.find(context.thread_group_id())->second.root_eligible &&
      state.groups.find(context.thread_group_id())->second.root_consumed &&
      !state.groups.find(context.thread_group_id())->second.trusted_control_network_active;
}

std::string SocketAddress(sa_family_t family, size_t length) {
  std::string address(length, '\0');
  memcpy(&address[0], &family, sizeof(family));
  return address;
}

std::string PacketSocketAddress() {
  std::string address = SocketAddress(kLinuxAFPacket, 20);
  // Pinned gVisor's AF_PACKET parser requires an Ethernet hardware address.
  address[11] = 6;
  return address;
}

bool SendSocketCreate(int output, int client, gvisor::syscall::Socket socket, int fd, bool expect_record,
                      const char* container_id, const char* family) {
  socket.clear_exit();
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket)) return false;
  if (expect_record) {
    if (!ExpectNetworkRecord(output, container_id, "SOCKET", family, "DIRECT_EXEC_SESSION", "PYTHON")) return false;
  } else if (!ExpectNoRecord(output)) {
    return false;
  }
  socket.mutable_exit()->set_result(fd);
  socket.mutable_exit()->set_errorno(0);
  return SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket) && ExpectNoRecord(output);
}

bool VerifyNetworkFamilies(int output, const std::string& remote, const std::string& control) {
  if (!RegisterProfile(control, kFifthID, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kFifthID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kFifthID, "container-start")) return false;

  // OCI init is CONTROL for provenance, but it is never the bounded direct
  // launch target that may receive trusted-control-network attribution.
  gvisor::syscall::Connect oci_connect;
  oci_connect.mutable_context_data()->set_container_id(kFifthID);
  oci_connect.mutable_context_data()->set_thread_group_id(1);
  oci_connect.mutable_context_data()->set_thread_group_start_time_ns(1);
  oci_connect.mutable_context_data()->set_parent_thread_group_id(0);
  oci_connect.mutable_context_data()->set_process_name("/bin/sh");
  oci_connect.set_address(SocketAddress(AF_INET, sizeof(sockaddr_in)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, oci_connect) ||
      !ExpectNetworkRecord(output, kFifthID, "CONNECT", "INET", "CONTROL_GROUP", "SHELL")) return false;

  gvisor::syscall::Execve root;
  root.mutable_context_data()->set_container_id(kFifthID);
  root.mutable_context_data()->set_thread_group_id(50);
  root.mutable_context_data()->set_thread_group_start_time_ns(500);
  root.mutable_context_data()->set_parent_thread_group_id(0);
  root.mutable_context_data()->set_is_exec_session(true);
  root.set_sysno(kSyscallExecve);
  root.set_pathname("python");
  gvisor::sentry::ExecveInfo rootResolved;
  *rootResolved.mutable_context_data() = root.context_data();
  rootResolved.set_binary_path("/usr/local/bin/python3.14");
  if (!SendDirectLaunchExec(output, client, root, rootResolved) || !ExpectRecord(output, kFifthID, "process-exec-expected")) return false;

  gvisor::syscall::Socket socket;
  socket.mutable_context_data()->set_container_id(kFifthID);
  socket.mutable_context_data()->set_thread_group_id(50);
  socket.mutable_context_data()->set_thread_group_start_time_ns(500);
  socket.mutable_context_data()->set_parent_thread_group_id(0);
  socket.mutable_context_data()->set_is_exec_session(true);
  socket.mutable_context_data()->set_process_name("python");
  socket.set_domain(AF_UNIX); socket.set_type(1); socket.set_protocol(0);
  if (!SendSocketCreate(output, client, socket, 3, false, kFifthID, "UNIX")) return false;
  socket.set_domain(AF_INET);
  if (!SendSocketCreate(output, client, socket, 4, false, kFifthID, "INET")) return false;
  socket.set_domain(AF_INET6);
  if (!SendSocketCreate(output, client, socket, 5, false, kFifthID, "INET6")) return false;

  gvisor::syscall::Syscall raw;
  *raw.mutable_context_data() = socket.context_data();
  raw.set_sysno(kSyscallSendtoX86);
  raw.set_arg1(4);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectTrustedNetworkRecord(output, kFifthID, "SENDTO", "INET")) return false;
  raw.set_sysno(kSyscallSendmmsgX86);
  raw.set_arg1(5);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectTrustedNetworkRecord(output, kFifthID, "SENDMMSG", "INET6")) return false;
  gvisor::syscall::Dup duplicate;
  *duplicate.mutable_context_data() = socket.context_data();
  duplicate.set_old_fd(4);
  duplicate.mutable_exit()->set_result(8);
  duplicate.mutable_exit()->set_errorno(0);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_DUP, duplicate)) return false;
  raw.set_sysno(kSyscallSendmsgX86);
  raw.set_arg1(8);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectTrustedNetworkRecord(output, kFifthID, "SENDMSG", "INET")) return false;
  gvisor::syscall::Close close_socket;
  *close_socket.mutable_context_data() = socket.context_data();
  close_socket.set_fd(8);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CLOSE, close_socket) || !ExpectNoRecord(output)) return false;
  socket.set_domain(AF_INET6);
  if (!SendSocketCreate(output, client, socket, 8, false, kFifthID, "INET6")) return false;
  raw.set_arg1(8);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectTrustedNetworkRecord(output, kFifthID, "SENDMSG", "INET6")) return false;

  gvisor::syscall::Clone clone;
  *clone.mutable_context_data() = socket.context_data();
  clone.set_flags(0);
  clone.mutable_exit()->set_result(51);
  clone.mutable_exit()->set_errorno(0);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CLONE, clone)) return false;
  gvisor::sentry::CloneInfo sentry_clone;
  *sentry_clone.mutable_context_data() = socket.context_data();
  sentry_clone.set_created_thread_group_id(51);
  sentry_clone.set_created_thread_start_time_ns(510);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_CLONE, sentry_clone) ||
      !ExpectNoRecord(output)) return false;
  *raw.mutable_context_data() = socket.context_data();
  raw.mutable_context_data()->set_thread_group_id(51);
  raw.mutable_context_data()->set_thread_group_start_time_ns(510);
  raw.mutable_context_data()->set_parent_thread_group_id(50);
  raw.set_arg1(4);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectNetworkRecord(output, kFifthID, "SENDMSG", "INET", "CONTROL_GROUP", "PYTHON")) return false;
  gvisor::syscall::Fork fork;
  *fork.mutable_context_data() = socket.context_data();
  fork.mutable_exit()->set_result(52);
  fork.mutable_exit()->set_errorno(0);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_FORK, fork)) return false;
  sentry_clone.clear_context_data();
  *sentry_clone.mutable_context_data() = socket.context_data();
  sentry_clone.set_created_thread_group_id(52);
  sentry_clone.set_created_thread_start_time_ns(520);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_CLONE, sentry_clone) ||
      !ExpectNoRecord(output)) return false;
  raw.mutable_context_data()->set_thread_group_id(52);
  raw.mutable_context_data()->set_thread_group_start_time_ns(520);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectNetworkRecord(output, kFifthID, "SENDMSG", "INET", "CONTROL_GROUP", "PYTHON")) return false;

  // Utility process classes are valid process telemetry but intentionally use
  // the bounded OTHER value in a network attribution envelope.
  raw.mutable_context_data()->set_process_name("/bin/cat");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectNetworkRecord(output, kFifthID, "SENDMSG", "INET", "CONTROL_GROUP", "OTHER")) return false;

  *raw.mutable_context_data() = socket.context_data();
  raw.set_arg1(8);
  socket.set_domain(kLinuxAFNetlink);
  if (!SendSocketCreate(output, client, socket, 6, false, kFifthID, "NETLINK")) return false;
  socket.set_domain(kLinuxAFPacket);
  if (!SendSocketCreate(output, client, socket, 7, true, kFifthID, "PACKET")) return false;

  gvisor::syscall::Connect connect;
  connect.mutable_context_data()->set_container_id(kFifthID);
  connect.mutable_context_data()->set_thread_group_id(50);
  connect.mutable_context_data()->set_thread_group_start_time_ns(500);
  connect.mutable_context_data()->set_parent_thread_group_id(0);
  connect.mutable_context_data()->set_is_exec_session(true);
  connect.mutable_context_data()->set_process_name("python");
  connect.set_address(SocketAddress(AF_UNIX, sizeof(sa_family_t)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectNoRecord(output)) return false;
  connect.set_address(SocketAddress(AF_UNIX, sizeof(sockaddr_un)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectNoRecord(output)) return false;
  connect.set_address(SocketAddress(AF_INET, sizeof(sockaddr_in)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectTrustedNetworkRecord(output, kFifthID, "CONNECT", "INET")) return false;
  connect.set_address(SocketAddress(AF_INET6, sizeof(sockaddr_in6)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectTrustedNetworkRecord(output, kFifthID, "CONNECT", "INET6")) return false;
  connect.set_address(SocketAddress(kLinuxAFNetlink, 12));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectNoRecord(output)) return false;
  connect.set_address(PacketSocketAddress());
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectNetworkRecord(output, kFifthID, "CONNECT", "PACKET", "DIRECT_EXEC_SESSION", "PYTHON")) return false;
  connect.set_address(SocketAddress(AF_UNSPEC, sizeof(sa_family_t)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectNoRecord(output)) return false;
  close(client);
  return ExpectRecord(output, kFifthID, "stream-end");
}

bool VerifySocketFault(int output, const std::string& remote, const std::string& control, const char* container_id, int domain, const char* reason) {
  if (!RegisterProfile(control, container_id, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(container_id);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, container_id, "container-start")) return false;
  gvisor::syscall::Socket socket;
  socket.mutable_context_data()->set_container_id(container_id);
  socket.mutable_context_data()->set_thread_group_id(1);
  socket.mutable_context_data()->set_thread_group_start_time_ns(1);
  socket.set_domain(domain);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket) || !ExpectRecord(output, container_id, "stream-fault", reason)) return false;
  close(client);
  return true;
}

bool VerifyMalformedNetworkFamilies(int output, const std::string& remote, const std::string& control) {
  return VerifySocketFault(output, remote, control, kSixthID, AF_UNSPEC, "SOCKET_AF_UNSPEC") &&
      VerifySocketFault(output, remote, control, kThirtySecondID, 0x7fff, "SOCKET_OTHER_FAMILY");
}

bool VerifyConnectFault(int output, const std::string& remote, const std::string& control, const char* container_id, const std::string& address, const char* reason) {
  if (!RegisterProfile(control, container_id, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(container_id);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, container_id, "container-start")) return false;
  gvisor::syscall::Connect connect;
  connect.mutable_context_data()->set_container_id(container_id);
  connect.mutable_context_data()->set_thread_group_id(1);
  connect.mutable_context_data()->set_thread_group_start_time_ns(1);
  connect.set_address(address);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectRecord(output, container_id, "stream-fault", reason)) return false;
  close(client);
  return true;
}

bool VerifyMalformedConnect(int output, const std::string& remote, const std::string& control) {
  return VerifyConnectFault(output, remote, control, kSeventhID, "\x02", "CONNECT_ADDRESS_TOO_SHORT") &&
      VerifyConnectFault(output, remote, control, kThirtyThirdID, SocketAddress(AF_UNIX, sizeof(sockaddr_un) + 1), "CONNECT_AF_UNIX_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, kThirtyFourthID, SocketAddress(AF_INET, sizeof(sockaddr_in) - 1), "CONNECT_AF_INET_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, kThirtyFifthID, SocketAddress(AF_INET6, sizeof(sockaddr_in6) - 1), "CONNECT_AF_INET6_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, kThirtySixthID, SocketAddress(kLinuxAFNetlink, 2), "CONNECT_AF_NETLINK_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, kThirtySeventhID, SocketAddress(kLinuxAFPacket, 2), "CONNECT_AF_PACKET_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, kThirtyEighthID, SocketAddress(0x7f, sizeof(sa_family_t)), "CONNECT_UNKNOWN_FAMILY");
}

bool VerifySocketPairDiagnosticPreservesFDAuthority() {
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID); context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990); context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true); context.set_process_name("artifact-claimed-local-socket");
  bool passed = true;
  for (const auto role : {ProcessState::Role::kControl, ProcessState::Role::kArtifact}) {
    ProcessState state;
    if (!RegisterGroup(&state, context, role, ProcessState::Provenance::kDirectExecRoot, false, true)) { passed = false; break; }
    state.groups[99].diagnostic_image = DiagnosticImage::kCargo;
    gvisor::syscall::SocketPair pair;
    *pair.mutable_context_data() = context;
    pair.set_domain(AF_UNIX); pair.set_type(SOCK_STREAM);
    pair.mutable_exit()->set_errorno(0); pair.mutable_exit()->set_result(0);
    pair.set_socket1(5); pair.set_socket2(6);
    std::string payload, id; const char* reason = nullptr;
    if (!pair.SerializeToString(&payload) || !ParseSocketPairDiagnostic(payload.data(), payload.size(), &id, &state, &reason)) { passed = false; break; }
    gvisor::syscall::Syscall raw;
    *raw.mutable_context_data() = context; raw.set_sysno(kSyscallSendtoX86); raw.set_arg1(5);
    if (!raw.SerializeToString(&payload) || ParseRawAndSend(payload.data(), payload.size(), output[0], &id, kProfileCargoResolver, &state, &reason) ||
        reason == nullptr || strcmp(reason, "FD_STATE_UNKNOWN") != 0 || !state.fd_states.empty() ||
        state.fault_raw_diagnostic.find("\"socketpair_fd_match\":true") == std::string::npos ||
        state.fault_raw_diagnostic.find("\"socketpair_source\":\"GUEST_RETURN_BUFFER\"") == std::string::npos) { passed = false; break; }
    // A forged guest pair buffer cannot replace an existing network FD with
    // LOCAL. The original network observation must remain visible.
    state.fd_states[99][5] = ProcessState::FDEntry{SocketClassification::kNetwork, AF_INET, false};
    if (!pair.SerializeToString(&payload) || !ParseSocketPairDiagnostic(payload.data(), payload.size(), &id, &state, &reason) ||
        !raw.SerializeToString(&payload) || !ParseRawAndSend(payload.data(), payload.size(), output[0], &id, kProfileCargoResolver, &state, &reason) ||
        !ExpectNetworkRecord(output[1], kFirstID, "SENDTO", "INET", role == ProcessState::Role::kArtifact ? "ARTIFACT_GROUP" : "DIRECT_EXEC_SESSION", "OTHER")) { passed = false; break; }
  }
  close(output[0]); close(output[1]);
  return passed;
}

bool VerifyCloseEntryPreservesReusedFD() {
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  const auto finish = [&](bool ok) { close(output[0]); close(output[1]); return ok; };
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID); context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990); context.set_is_exec_session(true);
  context.set_process_name("untrusted-name");
  for (const auto family : {AF_UNIX, AF_NETLINK, AF_INET}) {
    ProcessState state;
    if (!RegisterGroup(&state, context, ProcessState::Role::kArtifact,
        ProcessState::Provenance::kDirectExecRoot, false, true)) return finish(false);
    state.fd_states[99][8] = ProcessState::FDEntry{SocketClassification::kNetwork, AF_INET, false};
    std::string id = kFirstID;
    const char* reason = nullptr;
    auto consume = [&](const auto& message, uint32_t kind) {
      std::string payload; NormalizedCounts counts; TopologyState topology; Header header{};
      header.message_type = kind;
      return message.SerializeToString(&payload) && Handle(header, payload.data(), payload.size(), output[0],
          &id, kProfilePNPMResolver, nullptr, &state, &counts, &topology, &reason);
    };
    gvisor::syscall::Close closed;
    *closed.mutable_context_data() = context; closed.set_fd(8);
    if (!consume(closed, gvisor::common::MESSAGE_SYSCALL_CLOSE)) return finish(false);
    // The kernel frees the old descriptor before OnClose returns. Another
    // thread can allocate the same number before the old close exit arrives.
    gvisor::syscall::Socket created;
    *created.mutable_context_data() = context; created.set_sysno(41);
    created.set_domain(family); created.set_type(SOCK_DGRAM);
    if (!consume(created, gvisor::common::MESSAGE_SYSCALL_SOCKET)) return finish(false);
    created.mutable_exit()->set_result(8);
    if (!consume(created, gvisor::common::MESSAGE_SYSCALL_SOCKET)) return finish(false);
    gvisor::syscall::Syscall raw;
    *raw.mutable_context_data() = context; raw.set_sysno(kSyscallSendtoX86); raw.set_arg1(8);
    if (!consume(raw, gvisor::common::MESSAGE_SYSCALL_RAW)) {
      fprintf(stderr, "reused FD unavailable: family=%d reason=%s\n", family, reason == nullptr ? "NONE" : reason);
      return finish(false);
    }
    if (family == AF_INET) {
      if (!ExpectNetworkRecord(output[1], kFirstID, "SENDTO", "INET", "ARTIFACT_GROUP", "OTHER")) return finish(false);
    } else if (!ExpectNoRecord(output[1])) return finish(false);
    // An incompatible exit phase fails closed without erasing the new socket.
    closed.mutable_exit()->set_result(0);
    if (consume(closed, gvisor::common::MESSAGE_SYSCALL_CLOSE) || reason == nullptr ||
        strcmp(reason, "STREAM_FAULT") != 0 || state.fd_states[99].at(8).raw_family != family) return finish(false);
    // Entry revokes authority immediately; a failed exit cannot restore it.
    closed.clear_exit();
    if (!consume(closed, gvisor::common::MESSAGE_SYSCALL_CLOSE) || !state.fd_states[99].empty()) return finish(false);
    closed.mutable_exit()->set_result(-1); closed.mutable_exit()->set_errorno(9);
    if (consume(closed, gvisor::common::MESSAGE_SYSCALL_CLOSE) || !state.fd_states[99].empty()) return finish(false);
    if (consume(raw, gvisor::common::MESSAGE_SYSCALL_RAW) || reason == nullptr ||
        strcmp(reason, "FD_STATE_UNKNOWN") != 0 || !ExpectNoRecord(output[1])) return finish(false);
  }
  return finish(true);
}

bool VerifyFDMutationDiagnosticPreservesAuthority() {
  if (!VerifyCloseEntryPreservesReusedFD()) return false;
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  const auto finish = [&](bool ok) { close(output[0]); close(output[1]); return ok; };
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID); context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990); context.set_thread_id(101);
  context.set_thread_start_time_ns(1010); context.set_is_exec_session(true);
  context.set_process_name("untrusted-name");
  ProcessState state;
  if (!RegisterGroup(&state, context, ProcessState::Role::kArtifact,
      ProcessState::Provenance::kDirectExecRoot, false, true)) return finish(false);
  std::string id = kFirstID;
  const char* reason = nullptr;
  auto consume = [&](const auto& message, uint32_t kind) {
    std::string payload;
    NormalizedCounts counts; TopologyState topology; Header header{};
    header.message_type = kind;
    return message.SerializeToString(&payload) && Handle(header, payload.data(), payload.size(), output[0], &id,
        kProfilePNPMResolver, nullptr, &state, &counts, &topology, &reason);
  };
  // Closing a tracked socket revokes it; diagnostic history grants no authority.
  gvisor::syscall::Socket created;
  *created.mutable_context_data() = context; created.set_sysno(41);
  created.set_domain(AF_NETLINK); created.set_type(SOCK_RAW | SOCK_CLOEXEC);
  if (!consume(created, gvisor::common::MESSAGE_SYSCALL_SOCKET)) return finish(false);
  created.mutable_exit()->set_result(14);
  if (!consume(created, gvisor::common::MESSAGE_SYSCALL_SOCKET)) return finish(false);
  gvisor::syscall::Close closed;
  *closed.mutable_context_data() = context; closed.set_fd(14);
  if (!consume(closed, gvisor::common::MESSAGE_SYSCALL_CLOSE)) return finish(false);
  gvisor::syscall::Syscall raw;
  *raw.mutable_context_data() = context; raw.set_sysno(kSyscallSendtoX86); raw.set_arg1(14);
  auto rejected = [&]() {
    reason = nullptr;
    return !consume(raw, gvisor::common::MESSAGE_SYSCALL_RAW) && reason != nullptr &&
        strcmp(reason, "FD_STATE_UNKNOWN") == 0 && ExpectNoRecord(output[1]);
  };
  if (!rejected() || !state.fd_states[99].empty() ||
      state.fault_raw_diagnostic.find("\"fd_last\":\"CLOSE\",\"fd_last_family\":16") == std::string::npos ||
      state.fault_raw_diagnostic.find("\"fd_previous\":\"SOCKET\",\"fd_previous_family\":16") == std::string::npos)
    return finish(false);
  // Even forged LOCAL diagnostic history cannot classify or grant an FD.
  state.fd_mutation_diagnostics.clear();
  RecordFDMutationDiagnostic(&state, 99, 990, 14, "SOCKETPAIR", AF_UNIX);
  if (!rejected() || !state.fd_states[99].empty()) return finish(false);
  state.fd_states[99][14] = ProcessState::FDEntry{SocketClassification::kNetwork, AF_INET, false};
  if (!consume(raw, gvisor::common::MESSAGE_SYSCALL_RAW) ||
      !ExpectNetworkRecord(output[1], kFirstID, "SENDTO", "INET", "ARTIFACT_GROUP", "OTHER")) return finish(false);
  state.fd_states[99].clear();
  // Another identity, another FD and a reused group never appear as history.
  state.fd_mutation_diagnostics.clear();
  RecordFDMutationDiagnostic(&state, 98, 990, 14, "CLOSE", AF_UNIX);
  RecordFDMutationDiagnostic(&state, 99, 991, 14, "CLOSE", AF_UNIX);
  RecordFDMutationDiagnostic(&state, 99, 990, 15, "CLOSE", AF_UNIX);
  if (!rejected() || state.fault_raw_diagnostic.find("\"fd_last\"") != std::string::npos) return finish(false);
  RecordFDMutationDiagnostic(&state, 99, 990, -1, "EXEC", -1);
  if (!rejected() || state.fault_raw_diagnostic.find("\"fd_last\":\"EXEC\"") == std::string::npos) return finish(false);
  for (int i = 0; i < 100; ++i) RecordFDMutationDiagnostic(&state, 99, 990, 100 + i, "CLOSE", -1);
  if (state.fd_mutation_diagnostics.size() != 16 || !rejected() ||
      state.fault_raw_diagnostic.find("\"fd_last\"") != std::string::npos) return finish(false);
  return finish(true);
}

bool VerifyKernelSocketPairTracking() {
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  const auto finish = [&](bool ok) { close(output[0]); close(output[1]); return ok; };
  gvisor::common::ContextData context;
  context.set_container_id(kFirstID); context.set_thread_group_id(99);
  context.set_thread_group_start_time_ns(990); context.set_thread_id(101);
  context.set_thread_start_time_ns(1010); context.set_is_exec_session(true);
  context.set_process_name("untrusted-name");
  for (const auto role : {ProcessState::Role::kControl, ProcessState::Role::kArtifact}) {
    ProcessState initial;
    if (!RegisterGroup(&initial, context, role, ProcessState::Provenance::kDirectExecRoot, false, true)) return finish(false);
    std::string id = kFirstID;
    auto consume = [&](const auto& message, uint32_t kind, ProcessState* state) {
      std::string payload; const char* reason = nullptr;
      NormalizedCounts counts; TopologyState topology; Header header{};
      header.message_type = kind;
      return message.SerializeToString(&payload) && Handle(header, payload.data(), payload.size(), output[0], &id,
          kProfileCargoResolver, nullptr, state, &counts, &topology, &reason);
    };
    gvisor::syscall::SocketPair entry;
    *entry.mutable_context_data() = context; entry.set_sysno(53);
    entry.set_domain(AF_UNIX); entry.set_type(SOCK_STREAM | SOCK_CLOEXEC | SOCK_NONBLOCK);
    gvisor::syscall::SocketPairResult result;
    *result.mutable_context_data() = context; result.set_sysno(53);
    result.set_domain(AF_UNIX); result.set_type(entry.type()); result.set_success(true);
    result.set_socket1(5); result.set_socket2(6);
    ProcessState state = initial;
    if (!consume(entry, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR, &state) ||
        !consume(result, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR_RESULT, &state)) return finish(false);
    gvisor::syscall::Syscall raw;
    *raw.mutable_context_data() = context; raw.set_sysno(kSyscallSendtoX86); raw.set_arg1(5);
    if (!consume(raw, gvisor::common::MESSAGE_SYSCALL_RAW, &state) || state.fd_states[99].size() != 2 ||
        !state.fd_states[99][5].cloexec || !state.fd_states[99][6].cloexec || !ExpectNoRecord(output[1])) return finish(false);
    // Guest-return-buffer telemetry cannot change an unrelated network FD.
    state.fd_states[99][9] = ProcessState::FDEntry{SocketClassification::kNetwork, AF_INET, false};
    auto forged = entry; forged.mutable_exit()->set_result(0); forged.set_socket1(9); forged.set_socket2(10);
    raw.set_arg1(9);
    if (!consume(forged, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR, &state) ||
        !consume(raw, gvisor::common::MESSAGE_SYSCALL_RAW, &state) ||
        !ExpectNetworkRecord(output[1], kFirstID, "SENDTO", "INET", role == ProcessState::Role::kArtifact ? "ARTIFACT_GROUP" : "DIRECT_EXEC_SESSION", "OTHER")) return finish(false);
    ProcessState collision = initial;
    collision.fd_states[99][5] = ProcessState::FDEntry{SocketClassification::kNetwork, AF_INET, false};
    if (!consume(entry, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR, &collision) ||
        consume(result, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR_RESULT, &collision) ||
        collision.fd_states[99][5].family != SocketClassification::kNetwork || collision.fd_states[99].size() != 1) return finish(false);
    ProcessState reexec = state;
    ApplyExecCloexec(&reexec, 99);
    raw.set_arg1(5);
    if (consume(raw, gvisor::common::MESSAGE_SYSCALL_RAW, &reexec) ||
        reexec.fd_states[99].find(6) != reexec.fd_states[99].end() ||
        reexec.fd_states[99][9].family != SocketClassification::kNetwork) return finish(false);
    ProcessState duplicate_entry = initial;
    if (!consume(entry, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR, &duplicate_entry) ||
        consume(entry, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR, &duplicate_entry)) return finish(false);
    // Duplicate, unpaired, substituted syscall/identity/arguments and malformed
    // success/failure never grant descriptors.
    if (consume(result, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR_RESULT, &state)) return finish(false);
    for (int variant = 0; variant < 12; ++variant) {
      ProcessState invalid = initial;
      if (!consume(entry, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR, &invalid)) return finish(false);
      auto bad = result;
      if (variant == 0) bad.set_socket1(-1);
      if (variant == 1) bad.set_socket2(bad.socket1());
      if (variant == 2) bad.set_errorno(1);
      if (variant == 3) bad.set_success(false);
      if (variant == 4) bad.set_sysno(54);
      if (variant == 5) bad.set_domain(AF_INET);
      if (variant == 6) bad.set_type(SOCK_DGRAM);
      if (variant == 7) bad.set_protocol(1);
      if (variant == 8) bad.mutable_context_data()->set_thread_start_time_ns(1011);
      if (variant == 9) bad.mutable_context_data()->set_thread_group_start_time_ns(991);
      if (variant == 10) bad.mutable_context_data()->set_container_id(kSecondID);
      if (variant == 11) bad.mutable_context_data()->set_thread_id(102);
      if (consume(bad, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR_RESULT, &invalid) || !invalid.fd_states.empty()) return finish(false);
    }
    auto failed = result; failed.set_success(false); failed.set_errorno(14);
    failed.set_socket1(-1); failed.set_socket2(-1);
    ProcessState failure = initial;
    if (!consume(entry, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR, &failure) ||
        !consume(failed, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR_RESULT, &failure) || !failure.fd_states.empty()) return finish(false);
    ProcessState unpaired = initial;
    if (consume(result, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR_RESULT, &unpaired)) return finish(false);
    ProcessState limited = initial;
    for (int fd = 10; fd < 10 + static_cast<int>(kMaxTrackedFileDescriptorsPerGroup) - 1; ++fd)
      limited.fd_states[99][fd] = ProcessState::FDEntry{SocketClassification::kNetwork, AF_INET, false};
    if (!consume(entry, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR, &limited) ||
        consume(result, gvisor::common::MESSAGE_SYSCALL_SOCKETPAIR_RESULT, &limited) ||
        limited.fd_states[99].find(5) != limited.fd_states[99].end()) return finish(false);
    gvisor::syscall::Close closed; *closed.mutable_context_data() = context;
    closed.set_fd(5); raw.set_arg1(5);
    if (!consume(closed, gvisor::common::MESSAGE_SYSCALL_CLOSE, &state) ||
        consume(raw, gvisor::common::MESSAGE_SYSCALL_RAW, &state)) return finish(false);
  }
  return finish(true);
}

bool ExpectRawFaultRecord(int output, const char* id, const char* check, uint64_t fd,
                          const char* provenance, uint64_t executable_locator, const char* fd_history = "") {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  const std::string expected = std::string("{\"container_id\":\"") + id +
      "\",\"kind\":\"stream-fault\",\"reason\":\"FD_STATE_UNKNOWN\",\"fault_site\":\"RAW\",\"fault_raw\":{\"check\":\"" + check +
      "\",\"sysno\":44,\"fd\":" + std::to_string(fd) +
      ",\"kernel_image\":\"UNKNOWN\",\"role\":\"CONTROL\",\"provenance\":\"" + provenance +
      "\",\"executable_locator\":" + std::to_string(executable_locator) + fd_history + "}}";
  return size > 0 && std::string(buffer, size) == expected;
}

bool VerifyUnknownFDState(int output, const std::string& remote, const std::string& control) {
  if (!RegisterProfile(control, kEighthID, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kEighthID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kEighthID, "container-start")) return false;
  gvisor::syscall::Syscall raw;
  raw.mutable_context_data()->set_container_id(kEighthID);
  raw.mutable_context_data()->set_thread_group_id(1);
  raw.mutable_context_data()->set_thread_group_start_time_ns(1);
  raw.set_sysno(kSyscallSendtoX86);
  raw.set_arg1(99);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) || !ExpectRawFaultRecord(output, kEighthID, "TABLE", 99, "OCI_ROOT", 0)) return false;
  close(client);
  return true;
}

bool VerifyCloexecReexec(int output, const std::string& remote, const std::string& control) {
  if (!RegisterProfile(control, kNinthID, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kNinthID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kNinthID, "container-start")) return false;
  gvisor::syscall::Execve root;
  root.mutable_context_data()->set_container_id(kNinthID);
  root.mutable_context_data()->set_thread_group_id(90);
  root.mutable_context_data()->set_thread_group_start_time_ns(900);
  root.mutable_context_data()->set_parent_thread_group_id(0);
  root.mutable_context_data()->set_is_exec_session(true);
  root.mutable_context_data()->set_process_name("python");
  root.set_sysno(kSyscallExecve);
  root.set_pathname("python");
  gvisor::sentry::ExecveInfo resolved;
  *resolved.mutable_context_data() = root.context_data();
  resolved.set_binary_path("/usr/local/bin/python3.14");
  if (!SendDirectLaunchExec(output, client, root, resolved) || !ExpectRecord(output, kNinthID, "process-exec-expected")) return false;
  gvisor::syscall::Socket socket;
  *socket.mutable_context_data() = root.context_data();
  socket.set_domain(AF_INET);
  socket.set_type(02000000);
  socket.set_protocol(0);
  if (!SendSocketCreate(output, client, socket, 3, false, kNinthID, "INET")) return false;
  gvisor::syscall::Execve reexec = root;
  gvisor::sentry::ExecveInfo reexec_resolved = resolved;
  if (!SendSuccessfulExec(client, reexec, reexec_resolved) ||
      !ExpectUnexpectedProcessRecord(output, kNinthID, "SENTRY_EXEC", "PYTHON", "BOOTSTRAP_ENDED", "TRACKED_GROUP")) return false;
  gvisor::syscall::Connect after_reexec;
  *after_reexec.mutable_context_data() = root.context_data();
  after_reexec.set_address(SocketAddress(AF_INET, sizeof(sockaddr_in)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, after_reexec) ||
      !ExpectNetworkRecord(output, kNinthID, "CONNECT", "INET", "DIRECT_EXEC_SESSION", "PYTHON")) return false;
  gvisor::syscall::Syscall raw;
  *raw.mutable_context_data() = root.context_data();
  raw.set_sysno(kSyscallSendtoX86);
  raw.set_arg1(3);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) || !ExpectRawFaultRecord(output, kNinthID, "DESCRIPTOR", 3, "DIRECT_EXEC_ROOT", 0xb982a2d37572906dULL,
      ",\"fd_last\":\"EXEC\",\"fd_last_family\":-1,\"fd_previous\":\"SOCKET\",\"fd_previous_family\":2")) return false;
  close(client);
  return true;
}

bool VerifyProcessTrustBoundary(int output, const std::string& remote, const std::string& control) {
  if (!RegisterProfile(control, kSecondID, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kSecondID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kSecondID, "container-start")) return false;

  gvisor::common::ContextData bootstrap_context;
  bootstrap_context.set_container_id(kSecondID);
  bootstrap_context.set_thread_group_id(1);
  bootstrap_context.set_thread_group_start_time_ns(1);
  bootstrap_context.set_parent_thread_group_id(0);
  bootstrap_context.set_process_name("sleep");
  const auto bootstrap_shell = ExactOCIBootstrapShell(bootstrap_context);
  gvisor::sentry::ExecveInfo bootstrapResolved;
  *bootstrapResolved.mutable_context_data() = bootstrap_context;
  bootstrapResolved.set_binary_path("/usr/bin/sleep");
  bootstrapResolved.set_execfn("/bin/sleep");
  bootstrapResolved.add_argv("/bin/sleep");
  bootstrapResolved.add_argv("infinity");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, bootstrap_shell) ||
      !ExpectRecord(output, kSecondID, "process-exec-expected") ||
      !SendExactOCIBootstrapDemotion(output, client, kSecondID, bootstrap_context) ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, bootstrapResolved) ||
      !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve python;
  python.mutable_context_data()->set_container_id(kSecondID);
  python.mutable_context_data()->set_thread_group_id(20);
  python.mutable_context_data()->set_thread_group_start_time_ns(200);
  python.mutable_context_data()->set_parent_thread_group_id(0);
  python.mutable_context_data()->set_is_exec_session(true);
  python.mutable_context_data()->set_process_name("python");
  python.set_sysno(kSyscallExecve);
  python.set_pathname("python");

  gvisor::sentry::ExecveInfo pythonResolved;
  pythonResolved.mutable_context_data()->set_container_id(kSecondID);
  pythonResolved.mutable_context_data()->set_thread_group_id(20);
  pythonResolved.mutable_context_data()->set_thread_group_start_time_ns(200);
  pythonResolved.mutable_context_data()->set_parent_thread_group_id(0);
  pythonResolved.mutable_context_data()->set_is_exec_session(true);
  pythonResolved.set_binary_path("/usr/local/bin/python3.14");
  if (!SendDirectLaunchExec(output, client, python, pythonResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve pip;
  pip.mutable_context_data()->set_container_id(kSecondID);
  pip.mutable_context_data()->set_thread_group_id(30);
  pip.mutable_context_data()->set_thread_group_start_time_ns(300);
  pip.mutable_context_data()->set_parent_thread_group_id(0);
  pip.mutable_context_data()->set_is_exec_session(true);
  pip.mutable_context_data()->set_process_name("pip");
  pip.set_sysno(kSyscallExecve);
  pip.set_pathname("pip");
  gvisor::sentry::ExecveInfo pipResolved;
  *pipResolved.mutable_context_data() = pip.context_data();
  pipResolved.set_binary_path("/usr/local/bin/pip");
  if (!SendDirectLaunchExec(output, client, pip, pipResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve failed;
  failed.mutable_context_data()->set_container_id(kSecondID);
  failed.mutable_context_data()->set_thread_group_id(60);
  failed.mutable_context_data()->set_thread_group_start_time_ns(600);
  failed.mutable_context_data()->set_parent_thread_group_id(0);
  failed.mutable_context_data()->set_is_exec_session(true);
  failed.set_sysno(kSyscallExecve);
  failed.set_pathname("python");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, failed) || !ExpectNoRecord(output)) return false;
  failed.mutable_exit()->set_result(-1);
  failed.mutable_exit()->set_errorno(2);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, failed) || !ExpectNoRecord(output)) return false;

  gvisor::syscall::Execve failed_execveat = failed;
  failed_execveat.clear_exit();
  failed_execveat.set_sysno(kSyscallExecveat);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, failed_execveat) || !ExpectNoRecord(output)) return false;
  failed_execveat.mutable_exit()->set_result(-1);
  failed_execveat.mutable_exit()->set_errorno(2);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, failed_execveat) || !ExpectNoRecord(output)) return false;

  gvisor::syscall::Execve execveat = failed;
  execveat.clear_exit();
  execveat.set_sysno(kSyscallExecveat);
  execveat.set_pathname("python");
  gvisor::sentry::ExecveInfo execveatResolved;
  *execveatResolved.mutable_context_data() = execveat.context_data();
  execveatResolved.set_binary_path("/usr/local/bin/python3.14");
  if (!SendDirectLaunchExec(output, client, execveat, execveatResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve child;
  child.mutable_context_data()->set_container_id(kSecondID);
  child.mutable_context_data()->set_thread_group_id(21);
  child.mutable_context_data()->set_thread_group_start_time_ns(210);
  child.mutable_context_data()->set_parent_thread_group_id(20);
  child.mutable_context_data()->set_is_exec_session(true);
  child.mutable_context_data()->set_process_name("curl");
  child.set_sysno(kSyscallExecve);
  child.set_pathname("/usr/bin/curl");
  gvisor::sentry::ExecveInfo childResolved;
  *childResolved.mutable_context_data() = child.context_data();
  childResolved.set_binary_path("/usr/bin/curl");
  if (!SendProcessClone(output, client, kSecondID, 20, 200, 21, 210)) return false;
  if (!SendSuccessfulExec(client, child, childResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  child.mutable_context_data()->set_thread_group_id(22);
  child.mutable_context_data()->set_thread_group_start_time_ns(220);
  child.mutable_context_data()->set_process_name("python");
  child.set_pathname("/usr/local/bin/python");
  childResolved.mutable_context_data()->set_thread_group_id(22);
  childResolved.mutable_context_data()->set_thread_group_start_time_ns(220);
  childResolved.mutable_context_data()->set_process_name("python");
  childResolved.set_binary_path("/usr/local/bin/python");
  if (!SendProcessClone(output, client, kSecondID, 20, 200, 22, 220)) return false;
  if (!SendSuccessfulExec(client, child, childResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve shell;
  shell.mutable_context_data()->set_container_id(kSecondID);
  shell.mutable_context_data()->set_thread_group_id(40);
  shell.mutable_context_data()->set_thread_group_start_time_ns(400);
  shell.mutable_context_data()->set_parent_thread_group_id(0);
  shell.mutable_context_data()->set_is_exec_session(true);
  shell.mutable_context_data()->set_process_name("sh");
  shell.set_sysno(kSyscallExecve);
  shell.set_pathname("/bin/sh");
  gvisor::sentry::ExecveInfo shellResolved;
  *shellResolved.mutable_context_data() = shell.context_data();
  shellResolved.set_binary_path("/bin/sh");
  if (!SendDirectLaunchExec(output, client, shell, shellResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  child.mutable_context_data()->set_thread_group_id(41);
  child.mutable_context_data()->set_thread_group_start_time_ns(410);
  child.mutable_context_data()->set_parent_thread_group_id(40);
  child.mutable_context_data()->set_process_name("cat");
  child.set_pathname("/usr/bin/cat");
  childResolved.mutable_context_data()->set_thread_group_id(41);
  childResolved.mutable_context_data()->set_thread_group_start_time_ns(410);
  childResolved.mutable_context_data()->set_parent_thread_group_id(40);
  childResolved.mutable_context_data()->set_process_name("cat");
  childResolved.set_binary_path("/usr/bin/cat");
  if (!SendProcessClone(output, client, kSecondID, 40, 400, 41, 410)) return false;
  if (!SendSuccessfulExec(client, child, childResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;
  close(client);
  return ExpectRecord(output, kSecondID, "stream-end");
}

bool VerifySentryAuthoritativeBoundary(int output, const std::string& remote, const std::string& control) {
  auto make_exec = [](const char* container_id, int32_t group_id, int64_t start_time) {
    gvisor::syscall::Execve exec;
    exec.mutable_context_data()->set_container_id(container_id);
    exec.mutable_context_data()->set_thread_group_id(group_id);
    exec.mutable_context_data()->set_thread_group_start_time_ns(start_time);
    exec.mutable_context_data()->set_parent_thread_group_id(0);
    exec.mutable_context_data()->set_is_exec_session(true);
    exec.mutable_context_data()->set_process_name("python");
    exec.set_sysno(kSyscallExecve);
    exec.set_pathname("python");
    return exec;
  };
  auto make_sentry = [](const gvisor::syscall::Execve& exec) {
    gvisor::sentry::ExecveInfo sentry;
    *sentry.mutable_context_data() = exec.context_data();
    sentry.set_binary_path("/usr/local/bin/python3.14");
    return sentry;
  };

  if (!RegisterProfile(control, kTenthID, kProfilePyPI)) return false;
  int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kTenthID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kTenthID, "container-start")) return false;

  gvisor::syscall::Execve early = make_exec(kTenthID, 100, 1000);
  gvisor::syscall::Execve early_exit = early;
  early_exit.mutable_exit()->set_result(0);
  early_exit.mutable_exit()->set_errorno(0);
  gvisor::syscall::Execve next = early;
  gvisor::syscall::Execve next_exit = next;
  next_exit.mutable_exit()->set_result(0);
  next_exit.mutable_exit()->set_errorno(0);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, early) ||
      !SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, early_exit) ||
      !SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, next) ||
      !SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, next_exit) ||
      !ExpectNoRecord(output)) return false;
  gvisor::sentry::ExecveInfo launcher = make_sentry(next);
  launcher.set_binary_path(kBoundaryHelperPath);
  launcher.set_execfn(kBoundaryHelperPath);
  launcher.clear_argv();
  launcher.add_argv(kBoundaryHelperPath);
  launcher.add_argv(kLaunchMode);
  const gvisor::sentry::ExecveInfo target = make_sentry(next);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, launcher) ||
      !ExpectRecord(output, kTenthID, "process-exec-expected") ||
      !SendExactSetprivDemotion(output, client, kTenthID, target) ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, target) ||
      !ExpectRecord(output, kTenthID, "process-exec-expected")) return false;

  gvisor::syscall::Execve failed = make_exec(kTenthID, 101, 1001);
  gvisor::syscall::Execve failed_exit = failed;
  failed_exit.mutable_exit()->set_result(-1);
  failed_exit.mutable_exit()->set_errorno(2);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, failed) ||
      !SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, failed_exit) ||
      !ExpectNoRecord(output)) return false;

  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, target) ||
      !ExpectUnexpectedProcessRecord(output, kTenthID, "SENTRY_EXEC", "PYTHON", "BOOTSTRAP_ENDED", "TRACKED_GROUP")) return false;
  close(client);
  if (!ExpectRecord(output, kTenthID, "stream-end")) return false;

  if (!RegisterProfile(control, kEleventhID, kProfilePyPI)) return false;
  client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  start.mutable_context_data()->set_container_id(kEleventhID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kEleventhID, "container-start")) return false;
  gvisor::syscall::Execve unresolved = make_exec(kEleventhID, 103, 1003);
  gvisor::syscall::Execve unresolved_exit = unresolved;
  unresolved_exit.mutable_exit()->set_result(-1);
  unresolved_exit.mutable_exit()->set_errorno(2);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, unresolved) ||
      !SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, unresolved_exit) ||
      !ExpectNoRecord(output)) return false;
  close(client);
  if (!ExpectRecord(output, kEleventhID, "stream-end")) return false;

  if (!RegisterProfile(control, kTwelfthID, kProfilePyPI)) return false;
  client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  start.mutable_context_data()->set_container_id(kTwelfthID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kTwelfthID, "container-start")) return false;
  gvisor::sentry::ExecveInfo malformed;
  malformed.mutable_context_data()->set_container_id(kTwelfthID);
  malformed.set_binary_path("/usr/local/bin/python3.14");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, malformed)) return false;
  if (!ExpectRecord(output, kTwelfthID, "stream-fault", "EXEC_CORRELATION_INVALID")) return false;
  close(client);
  return true;
}

bool VerifyConcurrentPythonStreams(int output, const std::string& remote, const std::string& control) {
  if (!RegisterProfile(control, kThirtyFourthID, kProfilePyTorchCU130) ||
      !RegisterProfile(control, kThirtyFifthID, kProfilePyTorchCU132)) return false;
  const int anchor = ConnectRemote(remote);
  if (anchor < 0 || !Handshake(anchor)) return false;
  const int probe = ConnectRemote(remote);
  if (probe < 0 || !Handshake(probe)) { close(anchor); return false; }
  gvisor::container::Start anchor_start;
  anchor_start.mutable_context_data()->set_container_id(kThirtyFourthID);
  gvisor::container::Start probe_start;
  probe_start.mutable_context_data()->set_container_id(kThirtyFifthID);
  // The anchor remains open while the probe must make independent progress.
  const bool progressed = SendEvent(anchor, gvisor::common::MESSAGE_CONTAINER_START, anchor_start) &&
      ExpectRecord(output, kThirtyFourthID, "container-start") &&
      SendEvent(probe, gvisor::common::MESSAGE_CONTAINER_START, probe_start) &&
      ExpectRecord(output, kThirtyFifthID, "container-start");
  close(anchor);
  if (!progressed || !ExpectRecord(output, kThirtyFourthID, "stream-end")) {
    close(probe);
    return false;
  }
  close(probe);
  return ExpectRecord(output, kThirtyFifthID, "stream-end");
}

bool VerifyDelayedProfileRegistration(int output, const std::string& remote, const std::string& control) {
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kFourthID);
  std::thread registration([&control] {
    std::this_thread::sleep_for(std::chrono::milliseconds(150));
    RegisterProfile(control, kFourthID, kProfileNPM);
  });
  const bool started = SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) &&
      ExpectRecord(output, kFourthID, "container-start");
  registration.join();
  if (!started) return false;
  gvisor::syscall::Execve execve;
  execve.mutable_context_data()->set_container_id(kFourthID);
  execve.mutable_context_data()->set_thread_group_id(7);
  execve.mutable_context_data()->set_thread_group_start_time_ns(1);
  execve.mutable_context_data()->set_parent_thread_group_id(0);
  execve.mutable_context_data()->set_is_exec_session(true);
  execve.mutable_context_data()->set_process_name("trusted-test-process");
  execve.set_pathname("/usr/local/bin/node");
  execve.set_sysno(kSyscallExecve);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, execve)) return false;
  gvisor::sentry::ExecveInfo resolved;
  *resolved.mutable_context_data() = execve.context_data();
  resolved.set_binary_path("/usr/local/bin/node");
  resolved.set_execfn(kBoundaryHelperPath);
  resolved.add_argv(kBoundaryHelperPath);
  resolved.add_argv(kLaunchMode);
  gvisor::syscall::Execve exit = execve;
  exit.mutable_exit()->set_result(0);
  exit.mutable_exit()->set_errorno(0);
  const bool classified = SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, exit) &&
      SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, resolved) &&
      ExpectRecord(output, kFourthID, "process-exec-expected");
  close(client);
  return classified && ExpectRecord(output, kFourthID, "stream-end");
}

bool VerifyRoleHandoffAndCloneProvenance(int output, const std::string& remote, const std::string& control) {
  if (!RegisterProfile(control, kThirteenthID, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kThirteenthID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) ||
      !ExpectRecord(output, kThirteenthID, "container-start")) return false;

  gvisor::syscall::Execve launch;
  launch.mutable_context_data()->set_container_id(kThirteenthID);
  launch.mutable_context_data()->set_thread_group_id(130);
  launch.mutable_context_data()->set_thread_group_start_time_ns(1300);
  launch.mutable_context_data()->set_parent_thread_group_id(0);
  launch.mutable_context_data()->set_is_exec_session(true);
  launch.mutable_context_data()->set_process_name("python");
  launch.set_sysno(kSyscallExecve);
  launch.set_pathname("python");
  gvisor::sentry::ExecveInfo python;
  *python.mutable_context_data() = launch.context_data();
  python.set_binary_path("/usr/local/bin/python3.14");
  if (!SendDirectLaunchExec(output, client, launch, python) ||
      !ExpectRecord(output, kThirteenthID, "process-exec-expected")) return false;

  // npm's script-shell handoff is already uid/gid 1000 with empty groups and
  // zero capabilities. The exact -c marker removes CONTROL trust immediately;
  // it does not require a second privileged setpriv transition.
  if (!SendProcessClone(output, client, kThirteenthID, 130, 1300, 133, 1330)) return false;
  gvisor::sentry::ExecveInfo npm_handoff = python;
  npm_handoff.mutable_context_data()->set_thread_group_id(133);
  npm_handoff.mutable_context_data()->set_thread_group_start_time_ns(1330);
  npm_handoff.mutable_context_data()->set_parent_thread_group_id(130);
  npm_handoff.mutable_context_data()->set_is_exec_session(false);
  npm_handoff.mutable_context_data()->set_process_name("haa-boundary");
  npm_handoff.set_binary_path(kBoundaryHelperPath);
  npm_handoff.set_execfn(kBoundaryHelperPath);
  npm_handoff.clear_argv();
  npm_handoff.add_argv(kBoundaryHelperPath);
  npm_handoff.add_argv("-c");
  npm_handoff.add_argv("printf handoff");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, npm_handoff) ||
      !ExpectRecord(output, kThirteenthID, "process-exec-expected")) return false;
  gvisor::sentry::ExecveInfo artifact_shell = npm_handoff;
  artifact_shell.mutable_context_data()->set_process_name("sh");
  artifact_shell.set_binary_path("/bin/sh");
  artifact_shell.set_execfn("/bin/sh");
  artifact_shell.clear_argv();
  artifact_shell.add_argv("/bin/sh");
  artifact_shell.add_argv("-c");
  artifact_shell.add_argv("printf handoff");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, artifact_shell) ||
      !ExpectUnexpectedProcessRecord(output, kThirteenthID, "SENTRY_EXEC", "SHELL", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;

  gvisor::sentry::ExecveInfo handoff = python;
  handoff.set_binary_path(kBoundaryHelperPath);
  handoff.set_execfn(kBoundaryHelperPath);
  handoff.clear_argv();
  handoff.add_argv(kBoundaryHelperPath);
  handoff.add_argv(kPythonHandoffMode);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, handoff) ||
      !ExpectRecord(output, kThirteenthID, "process-exec-expected") ||
      !SendExactSetprivDemotion(output, client, kThirteenthID, python) ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, python) ||
      !ExpectRecord(output, kThirteenthID, "process-exec-expected")) return false;

  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, python) ||
      !ExpectUnexpectedProcessRecord(output, kThirteenthID, "SENTRY_EXEC", "PYTHON", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;

  if (!SendProcessClone(output, client, kThirteenthID, 130, 1300, 131, 1310)) return false;
  gvisor::sentry::ExecveInfo child = python;
  child.mutable_context_data()->set_thread_group_id(131);
  child.mutable_context_data()->set_thread_group_start_time_ns(1310);
  child.mutable_context_data()->set_parent_thread_group_id(130);
  child.mutable_context_data()->set_process_name("sh");
  child.set_binary_path("/bin/sh");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, child) ||
      !ExpectUnexpectedProcessRecord(output, kThirteenthID, "SENTRY_EXEC", "SHELL", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;

  // A fresh Docker exec may use a handoff as its first Sentry transition. It
  // establishes provenance and immediately removes trust without ever
  // activating the direct-root network exception.
  gvisor::sentry::ExecveInfo direct_handoff = python;
  direct_handoff.mutable_context_data()->set_thread_group_id(132);
  direct_handoff.mutable_context_data()->set_thread_group_start_time_ns(1320);
  direct_handoff.mutable_context_data()->set_parent_thread_group_id(0);
  direct_handoff.mutable_context_data()->set_is_exec_session(true);
  direct_handoff.set_binary_path(kBoundaryHelperPath);
  direct_handoff.set_execfn(kBoundaryHelperPath);
  direct_handoff.clear_argv();
  direct_handoff.add_argv(kBoundaryHelperPath);
  direct_handoff.add_argv(kPythonHandoffMode);
  gvisor::sentry::ExecveInfo direct_python = direct_handoff;
  direct_python.set_binary_path("/usr/local/bin/python3.14");
  direct_python.clear_execfn();
  direct_python.clear_argv();
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, direct_handoff) ||
      !ExpectRecord(output, kThirteenthID, "process-exec-expected") ||
      !SendExactSetprivDemotion(output, client, kThirteenthID, direct_python) ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, direct_python) ||
      !ExpectRecord(output, kThirteenthID, "process-exec-expected")) return false;
  gvisor::syscall::Connect direct_artifact_connect;
  *direct_artifact_connect.mutable_context_data() = direct_python.context_data();
  direct_artifact_connect.set_address(SocketAddress(AF_INET6, sizeof(sockaddr_in6)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, direct_artifact_connect) ||
      !ExpectNetworkRecord(output, kThirteenthID, "CONNECT", "INET6", "ARTIFACT_GROUP", "PYTHON")) return false;

  gvisor::syscall::Socket socket;
  *socket.mutable_context_data() = python.context_data();
  socket.set_domain(AF_INET);
  socket.set_type(1);
  if (!SendSocketCreate(output, client, socket, 3, false, kThirteenthID, "INET")) return false;
  gvisor::syscall::Syscall raw;
  *raw.mutable_context_data() = python.context_data();
  raw.set_sysno(kSyscallSendtoX86);
  raw.set_arg1(3);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectNetworkRecord(output, kThirteenthID, "SENDTO", "INET", "ARTIFACT_GROUP", "PYTHON")) return false;
  close(client);
  return ExpectRecord(output, kThirteenthID, "stream-end");
}

bool VerifyOCIBootstrapDemotion(int output, const std::string& remote, const std::string& control) {
  auto start_session = [&](const char* container_id) {
    if (!RegisterProfile(control, container_id, kProfilePyPI)) return -1;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return -1;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(container_id);
    return SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) &&
        ExpectRecord(output, container_id, "container-start") ? client : -1;
  };
  auto oci_context = [](const char* container_id) {
    gvisor::common::ContextData context;
    context.set_container_id(container_id);
    context.set_thread_group_id(1);
    context.set_thread_group_start_time_ns(1);
    context.set_parent_thread_group_id(0);
    context.set_process_name("sleep");
    return context;
  };
  auto sleep = [](const gvisor::common::ContextData& context) {
    gvisor::sentry::ExecveInfo message;
    *message.mutable_context_data() = context;
    message.set_binary_path("/usr/bin/sleep");
    message.set_execfn("/bin/sleep");
    message.add_argv("/bin/sleep");
    message.add_argv("infinity");
    return message;
  };
  auto demotion = [&](const gvisor::common::ContextData& context, const char* target,
                      bool wrong_argument) {
    gvisor::sentry::ExecveInfo message;
    *message.mutable_context_data() = context;
    message.set_binary_path(kSetprivPath);
    message.set_execfn(kSetprivPath);
    message.add_argv(kSetprivPath);
    message.add_argv(wrong_argument ? "--reuid=999" : "--reuid=1000");
    message.add_argv("--regid=1000");
    message.add_argv("--clear-groups");
    message.add_argv("--inh-caps=-all");
    message.add_argv("--ambient-caps=-all");
    message.add_argv("--bounding-set=-all");
    message.add_argv("--no-new-privs");
    message.add_argv("--");
    message.add_argv(target);
    message.add_argv("infinity");
    return message;
  };

  // container/start is itself authoritative for an OCI initial image. When
  // it carries the exact canonical shell tuple, no later Sentry re-exec is
  // required to prove the same bootstrap transition. This is common to every
  // ecosystem profile; PyPI wheel and sdist share kProfilePyPI.
  struct InitialImageCase { const char* id; const char* profile; };
  for (const auto& item : {InitialImageCase{"a1b2c3d4e5f60718", kProfileNPM},
                           InitialImageCase{"b1c2d3e4f5061728", kProfilePyPI},
                           InitialImageCase{"c1d2e3f405162738", kProfileGitHub}}) {
    if (!RegisterProfile(control, item.id, item.profile)) return false;
    const int initial_client = ConnectRemote(remote);
    if (initial_client < 0 || !Handshake(initial_client)) return false;
    gvisor::container::Start initial_start;
    initial_start.mutable_context_data()->set_container_id(item.id);
    initial_start.add_args("/bin/sh");
    initial_start.add_args("-ceu");
    initial_start.add_args(kOCIBootstrapCommand);
    if (!SendEvent(initial_client, gvisor::common::MESSAGE_CONTAINER_START, initial_start) ||
        !ExpectRecord(output, item.id, "container-start")) return false;
    const auto initial_context = oci_context(item.id);
    if (!SendEvent(initial_client, gvisor::common::MESSAGE_SENTRY_EXEC,
                   demotion(initial_context, "/bin/sleep", false)) ||
        !ExpectRecord(output, item.id, "process-exec-expected") ||
        !SendEvent(initial_client, gvisor::common::MESSAGE_SENTRY_EXEC, sleep(initial_context)) ||
        !ExpectRecord(output, item.id, "process-exec-expected")) return false;
    close(initial_client);
    if (!ExpectRecord(output, item.id, "stream-end")) return false;
  }

  if (!RegisterProfile(control, "d1e2f30415263748", kProfilePyPI)) return false;
  int wrong_initial_client = ConnectRemote(remote);
  if (wrong_initial_client < 0 || !Handshake(wrong_initial_client)) return false;
  gvisor::container::Start wrong_initial_start;
  wrong_initial_start.mutable_context_data()->set_container_id("d1e2f30415263748");
  wrong_initial_start.add_args("/bin/sh");
  wrong_initial_start.add_args("-ceu");
  wrong_initial_start.add_args(std::string(kOCIBootstrapCommand) + " ");
  if (!SendEvent(wrong_initial_client, gvisor::common::MESSAGE_CONTAINER_START, wrong_initial_start) ||
      !ExpectRecord(output, "d1e2f30415263748", "container-start")) return false;
  const auto wrong_initial_context = oci_context("d1e2f30415263748");
  if (!SendEvent(wrong_initial_client, gvisor::common::MESSAGE_SENTRY_EXEC,
                 demotion(wrong_initial_context, "/bin/sleep", false)) ||
      !ExpectRecord(output, "d1e2f30415263748", "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(wrong_initial_client);

  int client = start_session(kEighteenthID);
  if (client < 0) return false;
  const auto valid_context = oci_context(kEighteenthID);
  const auto valid_shell = ExactOCIBootstrapShell(valid_context);
  const auto valid_sleep = sleep(valid_context);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, valid_shell) ||
      !ExpectRecord(output, kEighteenthID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC,
                 demotion(valid_context, "/bin/sleep", false)) ||
      !ExpectRecord(output, kEighteenthID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, valid_sleep) ||
      !ExpectRecord(output, kEighteenthID, "process-exec-expected")) return false;
  gvisor::syscall::Connect connect;
  *connect.mutable_context_data() = valid_context;
  connect.set_address(SocketAddress(AF_INET, sizeof(sockaddr_in)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) ||
      !ExpectNetworkRecord(output, kEighteenthID, "CONNECT", "INET", "CONTROL_GROUP", "OTHER")) return false;
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, valid_sleep) ||
      !ExpectRecord(output, kEighteenthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kNineteenthID);
  if (client < 0) return false;
  const auto duplicate_context = oci_context(kNineteenthID);
  const auto duplicate_shell = ExactOCIBootstrapShell(duplicate_context);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, duplicate_shell) ||
      !ExpectRecord(output, kNineteenthID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, duplicate_shell) ||
      !ExpectRecord(output, kNineteenthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentiethID);
  if (client < 0) return false;
  const auto wrong_target_context = oci_context(kTwentiethID);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, ExactOCIBootstrapShell(wrong_target_context)) ||
      !ExpectRecord(output, kTwentiethID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC,
                 demotion(wrong_target_context, "/bin/false", false)) ||
      !ExpectRecord(output, kTwentiethID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentyFirstID);
  if (client < 0) return false;
  const auto wrong_argument_context = oci_context(kTwentyFirstID);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, ExactOCIBootstrapShell(wrong_argument_context)) ||
      !ExpectRecord(output, kTwentyFirstID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC,
                 demotion(wrong_argument_context, "/bin/sleep", true)) ||
      !ExpectRecord(output, kTwentyFirstID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentySecondID);
  if (client < 0) return false;
  const auto reordered_context = oci_context(kTwentySecondID);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, sleep(reordered_context)) ||
      !ExpectRecord(output, kTwentySecondID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentyThirdID);
  if (client < 0) return false;
  const auto root_context = oci_context(kTwentyThirdID);
  if (!SendProcessClone(output, client, kTwentyThirdID, 1, 1, 2, 2)) return false;
  auto clone_context = root_context;
  clone_context.set_thread_group_id(2);
  clone_context.set_thread_group_start_time_ns(2);
  clone_context.set_parent_thread_group_id(1);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC,
                 ExactOCIBootstrapShell(clone_context)) ||
      !ExpectRecord(output, kTwentyThirdID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentyFourthID);
  if (client < 0) return false;
  gvisor::common::ContextData direct_context;
  direct_context.set_container_id(kTwentyFourthID);
  direct_context.set_thread_group_id(3);
  direct_context.set_thread_group_start_time_ns(3);
  direct_context.set_parent_thread_group_id(0);
  direct_context.set_is_exec_session(true);
  direct_context.set_process_name("python");
  gvisor::sentry::ExecveInfo boundary;
  *boundary.mutable_context_data() = direct_context;
  boundary.set_binary_path(kBoundaryHelperPath);
  boundary.set_execfn(kBoundaryHelperPath);
  boundary.add_argv(kBoundaryHelperPath);
  boundary.add_argv(kLaunchMode);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, boundary) ||
      !ExpectRecord(output, kTwentyFourthID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC,
                 ExactOCIBootstrapShell(direct_context)) ||
      !ExpectRecord(output, kTwentyFourthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentyFifthID);
  if (client < 0) return false;
  auto wrong_binary = ExactOCIBootstrapShell(oci_context(kTwentyFifthID));
  wrong_binary.set_binary_path("/bin/dash");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, wrong_binary) ||
      !ExpectRecord(output, kTwentyFifthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentySixthID);
  if (client < 0) return false;
  auto wrong_execfn = ExactOCIBootstrapShell(oci_context(kTwentySixthID));
  wrong_execfn.set_execfn("/usr/bin/sh");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, wrong_execfn) ||
      !ExpectRecord(output, kTwentySixthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentySeventhID);
  if (client < 0) return false;
  auto wrong_flags = ExactOCIBootstrapShell(oci_context(kTwentySeventhID));
  wrong_flags.set_argv(1, "-ce");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, wrong_flags) ||
      !ExpectRecord(output, kTwentySeventhID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentyEighthID);
  if (client < 0) return false;
  auto modified_command = ExactOCIBootstrapShell(oci_context(kTwentyEighthID));
  modified_command.set_argv(2, std::string(kOCIBootstrapCommand) + " ");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, modified_command) ||
      !ExpectRecord(output, kTwentyEighthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kTwentyNinthID);
  if (client < 0) return false;
  auto missing_command = ExactOCIBootstrapShell(oci_context(kTwentyNinthID));
  missing_command.clear_argv();
  missing_command.add_argv("/bin/sh");
  missing_command.add_argv("-ceu");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, missing_command) ||
      !ExpectRecord(output, kTwentyNinthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kThirtiethID);
  if (client < 0) return false;
  auto extra_argument = ExactOCIBootstrapShell(oci_context(kThirtiethID));
  extra_argument.add_argv("unexpected");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, extra_argument) ||
      !ExpectRecord(output, kThirtiethID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kThirtyFirstID);
  if (client < 0) return false;
  const auto sleep_path_context = oci_context(kThirtyFirstID);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, ExactOCIBootstrapShell(sleep_path_context)) ||
      !ExpectRecord(output, kThirtyFirstID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC,
                 demotion(sleep_path_context, "/bin/sleep", false)) ||
      !ExpectRecord(output, kThirtyFirstID, "process-exec-expected")) return false;
  auto alias_sleep = sleep(sleep_path_context);
  alias_sleep.set_binary_path("/bin/sleep");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, alias_sleep) ||
      !ExpectRecord(output, kThirtyFirstID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);
  return true;
}

bool VerifyDemotionTransitionFaults(int output, const std::string& remote, const std::string& control) {
  auto direct_context = [](const char* container_id, int32_t group_id, int64_t start_time) {
    gvisor::common::ContextData context;
    context.set_container_id(container_id);
    context.set_thread_group_id(group_id);
    context.set_thread_group_start_time_ns(start_time);
    context.set_parent_thread_group_id(0);
    context.set_is_exec_session(true);
    context.set_process_name("python");
    return context;
  };
  auto boundary_launch = [](const gvisor::common::ContextData& context) {
    gvisor::sentry::ExecveInfo message;
    *message.mutable_context_data() = context;
    message.set_binary_path(kBoundaryHelperPath);
    message.set_execfn(kBoundaryHelperPath);
    message.add_argv(kBoundaryHelperPath);
    message.add_argv(kLaunchMode);
    return message;
  };
  auto python_target = [](const gvisor::common::ContextData& context) {
    gvisor::sentry::ExecveInfo message;
    *message.mutable_context_data() = context;
    message.set_binary_path("/usr/local/bin/python3.14");
    return message;
  };
  auto start_session = [&](const char* container_id) {
    if (!RegisterProfile(control, container_id, kProfilePyPI)) return -1;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return -1;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(container_id);
    return SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) &&
        ExpectRecord(output, container_id, "container-start") ? client : -1;
  };

  int client = start_session(kFourteenthID);
  if (client < 0) return false;
  const auto missing_context = direct_context(kFourteenthID, 140, 1400);
  const auto missing_target = python_target(missing_context);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, boundary_launch(missing_context)) ||
      !ExpectRecord(output, kFourteenthID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, missing_target) ||
      !ExpectRecord(output, kFourteenthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kFifteenthID);
  if (client < 0) return false;
  const auto duplicate_context = direct_context(kFifteenthID, 150, 1500);
  const auto duplicate_target = python_target(duplicate_context);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, boundary_launch(duplicate_context)) ||
      !ExpectRecord(output, kFifteenthID, "process-exec-expected") ||
      !SendExactSetprivDemotion(output, client, kFifteenthID, duplicate_target) ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, duplicate_target) ||
      !ExpectRecord(output, kFifteenthID, "process-exec-expected") ||
      !SendExactSetprivDemotion(output, client, kFifteenthID, duplicate_target,
                                "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kSixteenthID);
  if (client < 0) return false;
  const auto bare_context = direct_context(kSixteenthID, 160, 1600);
  const auto bare_target = python_target(bare_context);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, boundary_launch(bare_context)) ||
      !ExpectRecord(output, kSixteenthID, "process-exec-expected") ||
      !SendExactSetprivDemotion(output, client, kSixteenthID, bare_target) ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, bare_target) ||
      !ExpectRecord(output, kSixteenthID, "process-exec-expected") ||
      !SendExactSetprivDemotion(output, client, kSixteenthID, bare_target,
                                "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);

  client = start_session(kSeventeenthID);
  if (client < 0) return false;
  const auto reordered_context = direct_context(kSeventeenthID, 170, 1700);
  const auto reordered_target = python_target(reordered_context);
  if (!SendExactSetprivDemotion(output, client, kSeventeenthID, reordered_target,
                                "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) return false;
  close(client);
  return true;
}

bool RunFaultCase(int output, const std::string& remote, const std::string& control, bool mismatch) {
  const char* container_id = mismatch ? kFortySeventhID : kFortyEighthID;
  if (!RegisterProfile(control, container_id, mismatch ? kProfilePyPI : kProfileGitHub)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(container_id);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, container_id, "container-start")) return false;
  if (mismatch) {
    gvisor::sentry::ExecveInfo exec;
    exec.mutable_context_data()->set_container_id(kFourthID);
    if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, exec)) return false;
  } else {
    gvisor::sentry::ExecveInfo exec;
    exec.mutable_context_data()->set_container_id(container_id);
    if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, exec, 1)) return false;
  }
  const bool faulted = ExpectRecord(output, container_id, "stream-fault", mismatch ? "CONTAINER_MISMATCH" : "STREAM_FAULT");
  close(client);
  return faulted;
}

bool VerifyTopologyFailClosed(int output, const std::string& remote, const std::string& control) {
  // 1. Missing snapshot -> TOPOLOGY_NOT_READY
  {
    if (!RegisterProfile(control, kThirtyNinthID, kProfilePyPI)) return false;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return false;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(kThirtyNinthID);
    if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start, 0, /*send_snapshot=*/false) ||
        !ExpectRecordExact(output, kThirtyNinthID, "container-start")) return false;
    close(client);
    if (!ExpectRecordExact(output, kThirtyNinthID, "stream-fault", "TOPOLOGY_NOT_READY")) return false;
  }

  // 2. Duplicate snapshot -> TOPOLOGY_INVALID
  {
    if (!RegisterProfile(control, kFortiethID, kProfilePyPI)) return false;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return false;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(kFortiethID);
    if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) ||
        !ExpectRecord(output, kFortiethID, "container-start")) return false;
    auto dup = BuildCanonicalTopologySnapshot(kFortiethID, kProfilePyPI);
    if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_SNAPSHOT, dup) ||
        !ExpectRecordExact(output, kFortiethID, "stream-fault", "TOPOLOGY_INVALID")) return false;
    close(client);
  }

  // 3. Snapshot incomplete -> TOPOLOGY_INVALID
  {
    if (!RegisterProfile(control, kFortyFirstID, kProfilePyPI)) return false;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return false;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(kFortyFirstID);
    if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start, 0, false) ||
        !ExpectRecordExact(output, kFortyFirstID, "container-start")) return false;
    auto incomplete = BuildCanonicalTopologySnapshot(kFortyFirstID, kProfilePyPI);
    incomplete.set_snapshot_complete(false);
    if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_SNAPSHOT, incomplete) ||
        !ExpectRecordExact(output, kFortyFirstID, "stream-fault", "TOPOLOGY_INVALID")) return false;
    close(client);
  }

  // 4. Malformed topology: missing root -> TOPOLOGY_INVALID
  {
    if (!RegisterProfile(control, kFortySecondID, kProfilePyPI)) return false;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return false;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(kFortySecondID);
    if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start, 0, false) ||
        !ExpectRecordExact(output, kFortySecondID, "container-start")) return false;
    gvisor::sentry::MountTopologySnapshot no_root;
    *no_root.mutable_context_data() = start.context_data();
    no_root.set_mount_namespace_id(1);
    no_root.set_snapshot_complete(true);
    auto* m = no_root.add_mounts();
    m->set_mount_id(2);
    m->set_parent_mount_id(1);
    m->set_mountpoint("/tmp");
    m->set_filesystem_type("tmpfs");
    if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_SNAPSHOT, no_root) ||
        !ExpectRecordExact(output, kFortySecondID, "stream-fault", "TOPOLOGY_INVALID")) return false;
    close(client);
  }

  // 5. Malformed topology: invalid parent mount ID -> TOPOLOGY_INVALID
  {
    if (!RegisterProfile(control, kFortyThirdID, kProfilePyPI)) return false;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return false;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(kFortyThirdID);
    if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start, 0, false) ||
        !ExpectRecordExact(output, kFortyThirdID, "container-start")) return false;
    auto invalid_parent = BuildCanonicalTopologySnapshot(kFortyThirdID, kProfilePyPI);
    invalid_parent.mutable_mounts(1)->set_parent_mount_id(99);
    if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_SNAPSHOT, invalid_parent) ||
        !ExpectRecordExact(output, kFortyThirdID, "stream-fault", "TOPOLOGY_INVALID")) return false;
    close(client);
  }

  // 6. Missing required anchor (/haa-site for PyPI) -> TOPOLOGY_MISMATCH
  {
    if (!RegisterProfile(control, kFortyFourthID, kProfilePyPI)) return false;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return false;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(kFortyFourthID);
    if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start, 0, false) ||
        !ExpectRecordExact(output, kFortyFourthID, "container-start")) return false;
    auto missing_anchor = BuildCanonicalTopologySnapshot(kFortyFourthID, kProfileNPM);
    *missing_anchor.mutable_context_data() = start.context_data();
    if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_SNAPSHOT, missing_anchor) ||
        !ExpectRecordExact(output, kFortyFourthID, "stream-fault", "TOPOLOGY_MISMATCH")) return false;
    close(client);
  }

  // 7. Writable OCI root -> TOPOLOGY_MISMATCH
  {
    if (!RegisterProfile(control, kFortyFifthID, kProfilePyPI)) return false;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return false;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(kFortyFifthID);
    if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start, 0, false) ||
        !ExpectRecordExact(output, kFortyFifthID, "container-start")) return false;
    auto writable_root = BuildCanonicalTopologySnapshot(kFortyFifthID, kProfilePyPI);
    writable_root.mutable_mounts(0)->set_read_only(false);
    if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_SNAPSHOT, writable_root) ||
        !ExpectRecordExact(output, kFortyFifthID, "stream-fault", "TOPOLOGY_MISMATCH")) return false;
    close(client);
  }

  // 8. Post-seal topology mutation -> TOPOLOGY_MUTATION
  {
    if (!RegisterProfile(control, kFortySixthID, kProfilePyPI)) return false;
    const int client = ConnectRemote(remote);
    if (client < 0 || !Handshake(client)) return false;
    gvisor::container::Start start;
    start.mutable_context_data()->set_container_id(kFortySixthID);
    if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) ||
        !ExpectRecord(output, kFortySixthID, "container-start")) return false;
    gvisor::sentry::MountTopologyMutation mutation;
    *mutation.mutable_context_data() = start.context_data();
    mutation.set_mount_namespace_id(1);
    if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_MOUNT_TOPOLOGY_MUTATION, mutation) ||
        !ExpectRecordExact(output, kFortySixthID, "stream-fault", "TOPOLOGY_MUTATION")) return false;
    close(client);
  }

  return true;
}

bool VerifyRegisteredNestedMountBoundary() {
  for (bool read_only : {false, true}) {
    for (bool with_output : {false, true}) {
    for (int failure = 0; failure < (with_output ? 14 : 11); ++failure) {
      TopologyState topology;
      topology.expected = {
          {"/", "oci-root", "/", "", true, false, false, false},
          {"/tmp", "workspace", "/", "tmpfs", false, true, true, false},
          {"/haa-runtime", "helper", "/", "tmpfs", false, false, true, false},
          {"/tmp/haa-go-input", "workspace", "/tmp", "9p", read_only, false, false, false}};
      auto snapshot = BuildCanonicalTopologySnapshot(kFirstID, kProfileGoBuild);
      auto* input = snapshot.add_mounts();
      input->set_mount_id(4);
      input->set_parent_mount_id(2);
      input->set_mountpoint("/tmp/haa-go-input");
      input->set_filesystem_type("9p");
      input->set_read_only(read_only);
      auto* output = with_output ? snapshot.add_mounts() : nullptr;
      if (with_output) {
        topology.expected.push_back({"/tmp/haa-go-output", "workspace", "/tmp", "9p", false, false, false, false});
        output->set_mount_id(5);
        output->set_parent_mount_id(2);
        output->set_mountpoint("/tmp/haa-go-output");
        output->set_filesystem_type("9p");
        output->set_read_only(false);
      }
      switch (failure) {
        case 1: input->set_read_only(!read_only); break;
        case 2: input->set_parent_mount_id(1); break;
        case 3: input->set_filesystem_type("tmpfs"); break;
        case 4: input->set_noexec(true); break;
        case 5: input->set_nosuid(true); break;
        case 6: input->set_nodev(true); break;
        case 7: snapshot.mutable_mounts()->RemoveLast(); break;
        case 8:
        case 9: {
          auto* extra = snapshot.add_mounts();
          extra->set_mount_id(with_output ? 6 : 5);
          extra->set_parent_mount_id(failure == 8 ? 2 : 4);
          extra->set_mountpoint(failure == 8 ? "/tmp/unregistered" : "/tmp/haa-go-input/shadow");
          extra->set_filesystem_type("9p");
          extra->set_read_only(true);
          break;
        }
        case 10: topology.expected.push_back(topology.expected.back()); break;
        case 11: output->set_read_only(true); break;
        case 12: output->set_parent_mount_id(4); break;
        case 13: {
          auto* extra = snapshot.add_mounts();
          extra->set_mount_id(6);
          extra->set_parent_mount_id(5);
          extra->set_mountpoint("/tmp/haa-go-output/shadow");
          extra->set_filesystem_type("9p");
          break;
        }
      }
      int sockets[2];
      if (socketpair(AF_UNIX, SOCK_DGRAM, 0, sockets) != 0) return false;
      std::string payload, container_id = kFirstID;
      const char* reason = nullptr;
      const bool encoded = snapshot.SerializeToString(&payload);
      const bool accepted = encoded && ParseTopologySnapshot(payload.data(), payload.size(), sockets[0], &container_id, &topology, &reason);
      const bool correct = failure == 0
          ? accepted && topology.sealed && topology.snapshot_seen && topology.anchors.size() == (with_output ? 5U : 4U) &&
              ExpectRecordExact(sockets[1], kFirstID, "mount-anchors-ready")
          : !accepted && !topology.sealed && !topology.snapshot_seen && reason != nullptr &&
              strcmp(reason, "TOPOLOGY_MISMATCH") == 0;
      close(sockets[0]); close(sockets[1]);
      if (!correct) return false;
    }
    }
  }
  return true;
}

bool VerifyBoundedClassificationSemantics() {
  if (ClassifySocketFamily(AF_UNIX) != SocketClassification::kLocal ||
      ClassifySocketFamily(AF_INET) != SocketClassification::kNetwork ||
      ClassifySocketFamily(AF_INET6) != SocketClassification::kNetwork ||
      ClassifySocketFamily(kLinuxAFNetlink) != SocketClassification::kSpecialKernelLocal ||
      ClassifySocketFamily(kLinuxAFPacket) != SocketClassification::kNetwork ||
      ClassifySocketFamily(0x7fff) != SocketClassification::kUnknown) return false;

  ProcessState state;
  gvisor::common::ContextData direct;
  direct.set_thread_group_id(20);
  direct.set_thread_group_start_time_ns(200);
  direct.set_parent_thread_group_id(0);
  direct.set_is_exec_session(true);
  if (!RegisterGroup(&state, direct, ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, true, true)) return false;
  ProcessClassification expected = IsExpectedProcess("python", direct, kProfilePyPI, &state);
  if (!expected.expected || expected.process_class != ProcessClass::kPython) return false;

  gvisor::common::ContextData child;
  child.set_thread_group_id(21);
  child.set_thread_group_start_time_ns(210);
  child.set_parent_thread_group_id(20);
  child.set_is_exec_session(true);
  ProcessClassification unexpected = IsExpectedProcess("/usr/bin/curl", child, kProfilePyPI, &state);
  return !unexpected.expected && unexpected.process_class == ProcessClass::kUnknown &&
      strcmp(unexpected.reason, "PROCESS_PROVENANCE_UNKNOWN") == 0 && strcmp(unexpected.parent_relation, "UNTRACKED_PARENT") == 0;
}

TopologyState PNPMTargetLifecycleTestTopology(bool bun = false) {
  const char* project_mount = bun ? kBunTargetProjectMount : kPNPMProjectMount;
  TopologyState topology;
  topology.sealed = true; topology.snapshot_seen = true; topology.namespace_id = 99;
  topology.expected = {
      {"/", "oci-root", "/", "", true, false, false, false},
      {"/tmp", "workspace", "/", "tmpfs", false, true, true, false},
      {"/haa-runtime", "helper", "/", "tmpfs", false, false, true, false},
      {project_mount, "workspace", "/tmp", "9p", false, false, false, false}};
  topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
  topology.anchors.emplace(2, MountAnchor{2, "/tmp", "workspace"});
  topology.anchors.emplace(3, MountAnchor{3, "/haa-runtime", "helper"});
  topology.anchors.emplace(4, MountAnchor{4, project_mount, "workspace"});
  return topology;
}

bool VerifyPNPMTargetLifecycleShapeBoundary() {
  const std::string prefix = std::string(kPNPMProjectMount) + "/project/node_modules/.pnpm/";
  const std::string target = prefix + "is-odd@3.0.1/node_modules/is-odd";
  const std::string hashed_slot = std::string(87, 'a') + "_69b8023e95a408229c8fe59943dd2497";
  const std::string hashed_target = prefix + hashed_slot + "/node_modules/" + std::string(130, 'a');
  if (!ValidPNPMTargetLifecyclePath(hashed_target)) return false;
  for (const auto& slot : {hashed_slot.substr(0, 119), hashed_slot + "f", hashed_slot.substr(0, 87) + "-" + hashed_slot.substr(88), hashed_slot.substr(0, 119) + "g", hashed_slot.substr(0, 88) + "69B8023E95A408229C8FE59943DD2497"}) {
    if (ValidPNPMTargetLifecyclePath(prefix + slot + "/node_modules/pkg")) return false;
  }
  for (const auto& path : {target, prefix + "@scope+package@1.2.3/node_modules/@scope/package",
                         prefix + "use-sync-external-store@1.2.0_react@18.3.1/node_modules/use-sync-external-store"}) {
    if (!ValidPNPMTargetLifecyclePath(path)) return false;
  }
  for (const auto& path : {std::string(""), std::string("/tmp/package"), prefix + "folder/node_modules/pkg",
       prefix + "pkg@1/node_modules/../other", prefix + "pkg@1/node_modules/pkg/extra", prefix + "pkg@1/node_modules/@scope",
       prefix + "pkg@1/node_modules/@/pkg", prefix + "pkg@1/node_modules/pkg ", prefix + "pkg@1/node_modules/pkg;id",
       prefix + "pkg@1/node_modules/$(id)", prefix + "pkg@1/node_modules/pkg\\extra", prefix + "pkg@1/node_modules/pkg//extra",
       prefix + "pkg@1/node_modules/.", prefix + "pkg@1/node_modules/..", prefix + std::string(512, 'a')}) {
    if (ValidPNPMTargetLifecyclePath(path)) return false;
  }
  gvisor::sentry::ExecveInfo boundary;
  boundary.set_binary_path(kBoundaryHelperPath); boundary.set_execfn(kBoundaryHelperPath);
  for (const auto& value : {std::string(kBoundaryHelperPath), std::string(kLaunchMode), TestAdmissionNonce(350, 3500),
        std::string("/bin/sh"), std::string("-ceu"), std::string(kPNPMTargetLifecycleCommand), std::string("helox-pnpm-target"), target}) boundary.add_argv(value);
  const auto topology = PNPMTargetLifecycleTestTopology();
  if (!IsExactPNPMTargetLifecycleBoundary(boundary, kProfileNPM, &topology)) return false;
  if (IsExactPNPMTargetLifecycleBoundary(boundary, kProfilePNPMResolver, &topology) ||
      IsExactPNPMTargetLifecycleBoundary(boundary, kProfilePyPI, &topology) ||
      IsExactPNPMTargetLifecycleBoundary(boundary, kProfileNPM, nullptr)) return false;
  for (int test = 0; test < 10; ++test) {
    auto changed = topology;
    switch (test) {
      case 0: changed.sealed = false; break;
      case 1: changed.snapshot_seen = false; break;
      case 2: changed.namespace_id = 0; break;
      case 3: changed.expected.pop_back(); changed.anchors.erase(4); break;
      case 4: changed.expected.back().read_only = true; break;
      case 5: changed.expected.back().filesystem_type = "tmpfs"; break;
      case 6: changed.expected.back().mount_class = "helper"; break;
      case 7: changed.anchors.erase(4); break;
      case 8: changed.anchors.emplace(5, MountAnchor{5, target, "workspace"}); break;
      case 9: changed.expected[0].read_only = false; break;
    }
    if (IsExactPNPMTargetLifecycleBoundary(boundary, kProfileNPM, &changed)) return false;
  }
  for (int index : {0, 1, 2, 3, 4, 5, 6, 7}) {
    auto changed = boundary; changed.set_argv(index, "changed");
    if (IsExactPNPMTargetLifecycleBoundary(changed, kProfileNPM, &topology)) return false;
  }
  auto extra = boundary; extra.add_argv("extra");
  auto wrong_image = boundary; wrong_image.set_binary_path("/tmp/haa-boundary");
  return !IsExactPNPMTargetLifecycleBoundary(extra, kProfileNPM, &topology) &&
      !IsExactPNPMTargetLifecycleBoundary(wrong_image, kProfileNPM, &topology);
}

bool VerifyPNPMTargetLifecycleProductionPathForTarget(const std::string& target, bool bun = false) {
  const auto topology = PNPMTargetLifecycleTestTopology(bun);
  const char* command = bun ? kBunTargetLifecycleCommand : kPNPMTargetLifecycleCommand;
  const char* label = bun ? "helox-bun-target" : "helox-pnpm-target";
  const char* project_directory = bun ? "/tmp/haa-bun-input/project" : "/tmp/haa-pnpm-input/project";
  const std::string nonce = TestAdmissionNonce(350, 3500);
  gvisor::common::ContextData root;
  root.set_container_id(kThirtySecondID); root.set_thread_group_id(350); root.set_thread_group_start_time_ns(3500);
  root.set_parent_thread_group_id(0); root.set_is_exec_session(true);
  gvisor::sentry::ExecveInfo boundary;
  *boundary.mutable_context_data() = root;
  boundary.set_binary_path(kBoundaryHelperPath); boundary.set_execfn(kBoundaryHelperPath);
  for (const auto& value : {std::string(kBoundaryHelperPath), std::string(kLaunchMode), nonce, std::string("/bin/sh"),
       std::string("-ceu"), std::string(command), std::string(label), target}) boundary.add_argv(value);
  auto demotion = boundary; demotion.set_binary_path(kSetprivPath); demotion.set_execfn(kSetprivPath); demotion.clear_argv();
  for (const char* value : {kSetprivPath, "--reuid=1000", "--regid=1000", "--clear-groups", "--inh-caps=-all",
       "--ambient-caps=-all", "--bounding-set=-all", "--no-new-privs", "--", "/bin/sh"}) demotion.add_argv(value);
  auto shell = boundary; shell.set_binary_path("/usr/bin/dash"); shell.set_execfn("/bin/sh"); shell.clear_argv();
  shell.add_argv("/bin/sh"); shell.add_argv("-ceu"); shell.add_argv(command); shell.add_argv(label); shell.add_argv(target);
  gvisor::sentry::CloneInfo clone; *clone.mutable_context_data() = root;
  clone.set_created_thread_group_id(351); clone.set_created_thread_start_time_ns(3510);
  auto child = root; child.set_thread_group_id(351); child.set_thread_group_start_time_ns(3510); child.set_parent_thread_group_id(350);
  gvisor::sentry::ExecveInfo npm; *npm.mutable_context_data() = child;
  npm.set_binary_path(kNpmCLIPath); npm.set_execfn(kNpmPath); npm.add_argv(kNpmPath);
  for (const char* value : {"--prefix", project_directory, "rebuild", "--ignore-scripts=false", "--bin-links=false",
       "--foreground-scripts", "--offline", "--no-update-notifier", "--workspaces=false", "--no-audit", "--no-fund", "--dangerously-allow-all-scripts"}) npm.add_argv(value);
  npm.add_argv(target);
  auto node = npm; node.set_binary_path(kNodePath); node.set_execfn(kNodePath); node.clear_argv(); node.add_argv("node"); node.add_argv(kNpmPath);
  for (int n = 1; n < npm.argv_size(); ++n) node.add_argv(npm.argv(n));
  int sockets[2]; if (socketpair(AF_UNIX, SOCK_DGRAM, 0, sockets) != 0) return false;
  auto parse = [&](const gvisor::sentry::ExecveInfo& event, ProcessState* state, const char* kind,
                   const char* expected_reason = "CLASS_MISMATCH", const char* expected_class = "NODE",
                   const char* expected_parent = "TRACKED_GROUP") {
    std::string payload, id; const char* reason = nullptr;
    auto registration = TestProfileWithAdmission(kProfileNPM, kLaunchMode, nonce);
    if (!event.SerializeToString(&payload)) return false;
    const bool parsed = ParseSentryProcessAndClassify(payload.data(), payload.size(), sockets[0], &id, kProfileNPM, &registration, state, &reason, &topology);
    if (!parsed) {
      if (strcmp(kind, "REJECTED") != 0) fprintf(stderr, "PNPM target parse failed image=%s reason=%s\n", DiagnosticImageName(DiagnosticImageForPath(event.binary_path())), reason == nullptr ? "NONE" : reason);
      return strcmp(kind, "REJECTED") == 0 && reason != nullptr && strcmp(reason, expected_reason) == 0;
    }
    if (strcmp(kind, "REJECTED") == 0) return ExpectUnexpectedProcessRecord(sockets[1], kThirtySecondID, "SENTRY_EXEC", expected_class, expected_reason, expected_parent);
    return ExpectRecord(sockets[1], kThirtySecondID, kind);
  };
  auto baseline = [&](ProcessState* state) {
    std::string payload, id; const char* reason = nullptr;
    auto check = [](bool value, const char* step) { if (!value) fprintf(stderr, "PNPM target baseline failed step=%s\n", step); return value; };
    return check(parse(boundary, state, "process-exec-expected"), "BOUNDARY") && check(state->groups.at(350).pnpm_target_lifecycle == target, "ROOT_TARGET") &&
        check(parse(demotion, state, "process-exec-expected"), "DEMOTION") && check(parse(shell, state, "process-exec-expected"), "SHELL") &&
        check(clone.SerializeToString(&payload) && ParseSentryClone(payload.data(), payload.size(), -1, &id, state, &reason), "CLONE") &&
        check(state->groups.at(351).pnpm_target_lifecycle == target, "CHILD_TARGET") && check(parse(npm, state, "process-exec-expected"), "NPM") &&
        check(state->groups.at(351).pnpm_target_npm_pending, "PENDING");
  };
  ProcessState valid; bool ok = baseline(&valid) && parse(node, &valid, "process-exec-expected") &&
      !valid.groups.at(351).pnpm_target_npm_pending && valid.groups.at(351).pnpm_target_npm_consumed &&
      valid.expected_groups.at(351).process_class == ProcessClass::kNode;
  if (!ok) fprintf(stderr, "PNPM target positive production path failed\n");
  // Every negative starts after the complete original boundary/clone/launcher.
  for (int test = 0; ok && test < 16; ++test) {
    ProcessState state; if (!baseline(&state)) { ok = false; break; }
    auto changed = node;
    switch (test) {
      case 0: changed.set_argv(changed.argv_size()-1, std::string(kPNPMProjectMount)+"/project/node_modules/.pnpm/other@1/node_modules/other"); break;
      case 1: changed.add_argv("extra"); break;
      case 2: changed.set_argv(4, "install"); break;
      case 3: changed.set_binary_path("/usr/bin/node"); break;
      case 4: changed.set_execfn("/usr/bin/node"); break;
      case 5: changed.mutable_context_data()->set_thread_group_start_time_ns(3511); break;
      case 6: changed.mutable_context_data()->set_thread_group_id(999); break;
      case 7: state.groups.at(351).role = ProcessState::Role::kArtifact; break;
      case 8: state.groups.at(351).pnpm_target_npm_pending = false; break;
      case 9: state.groups.at(351).pnpm_target_npm_consumed = true; break;
      case 10: state.groups.at(350).start_time_ns++; break;
      case 11: state.groups.at(350).current_image_class = ProcessClass::kNode; break;
      case 12: state.groups.at(351).clone_creator_group_id = 999; break;
      case 13: state.groups.at(350).pnpm_target_lifecycle.clear(); break;
      case 14: state.groups.at(351).provenance = ProcessState::Provenance::kDirectExecRoot; break;
      case 15: changed.set_argv(changed.argv_size()-1, std::string(kBunTargetProjectMount)+"/project/node_modules/other"); break;
    }
    const char* expected_reason = test == 5 ? "PROCESS_IDENTITY_REUSED" :
        test == 6 ? "PROCESS_PROVENANCE_UNKNOWN" : test == 7 ? "ARTIFACT_ROLE" : "CLASS_MISMATCH";
    ok = parse(changed, &state, "REJECTED", expected_reason, test == 3 ? "OTHER" : "NODE", test == 7 ? "ARTIFACT_GROUP" : "TRACKED_GROUP");
    if (!ok) fprintf(stderr, "PNPM target negative failed index=%d\n", test);
  }
  // The exact transition cannot consume a weakened control/creator state.
  for (int test = 0; ok && test < 9; ++test) {
    ProcessState state; if (!baseline(&state)) { ok = false; break; }
    switch (test) {
      case 0: state.groups.at(350).role = ProcessState::Role::kArtifact; break;
      case 1: state.groups.at(350).provenance = ProcessState::Provenance::kCloneChild; break;
      case 2: state.groups.at(351).root_eligible = true; break;
      case 3: state.groups.at(351).root_consumed = false; break;
      case 4: state.groups.at(351).trusted_control_network_active = true; break;
      case 5: state.groups.at(351).demotion_pending = true; break;
      case 6: state.groups.at(351).launch_target_pending = true; break;
      case 7: state.groups.at(351).handoff_target_pending = true; break;
      case 8: state.groups.at(351).clone_creator_group_start_time_ns++; break;
    }
    if (IsExactPNPMTargetNpmNodeTransition(node, kProfileNPM, 351, state.groups.at(351), state.expected_groups.at(351), state) ||
        IsExactPNPMTargetNpmNodeTransition(node, kProfilePyPI, 351, state.groups.at(351), state.expected_groups.at(351), state)) {
      fprintf(stderr, "PNPM target role/creator boundary failed index=%d\n", test); ok = false;
    }
  }
  // Each required flag and the sole selector must survive the interpreter
  // transition unchanged; omission cannot silently disable script coverage.
  for (int index = 2; ok && index < node.argv_size(); ++index) {
    for (bool remove : {false, true}) {
      ProcessState state; if (!baseline(&state)) { ok = false; break; }
      auto changed = node;
      if (remove) changed.mutable_argv()->DeleteSubrange(index, 1);
      else changed.set_argv(index, "changed");
      if (!parse(changed, &state, "REJECTED")) {
        fprintf(stderr, "PNPM target flag boundary failed index=%d remove=%d\n", index, remove ? 1 : 0);
        ok = false; break;
      }
    }
  }
  // A successful interpreter transition does not re-arm itself.
  ok = ok && parse(node, &valid, "REJECTED", "BOOTSTRAP_ENDED") && !valid.groups.at(351).pnpm_target_npm_pending;
  close(sockets[0]); close(sockets[1]); return ok;
}

bool VerifyPNPMTargetLifecycleProductionPath() {
  const std::string prefix = std::string(kPNPMProjectMount) + "/project/node_modules/.pnpm/";
  return VerifyPNPMTargetLifecycleProductionPathForTarget(prefix + "is-odd@3.0.1/node_modules/is-odd") &&
      VerifyPNPMTargetLifecycleProductionPathForTarget(prefix + std::string(87, 'a') + "_69b8023e95a408229c8fe59943dd2497/node_modules/" + std::string(130, 'a'));
}

bool VerifyBunTargetLifecycleShapeBoundary() {
  const std::string prefix = std::string(kBunTargetProjectMount) + "/project/node_modules/";
  for (const auto& tail : {"pkg", "@scope/pkg", "pkg/node_modules/child", "@scope/pkg/node_modules/@other/child"}) {
    const std::string target = prefix + tail;
    if (!ValidBunTargetLifecyclePath(target) || ValidPNPMTargetLifecyclePath(target)) return false;
    gvisor::sentry::ExecveInfo boundary;
    boundary.set_binary_path(kBoundaryHelperPath); boundary.set_execfn(kBoundaryHelperPath);
    for (const auto& value : {std::string(kBoundaryHelperPath), std::string(kLaunchMode), TestAdmissionNonce(350, 3500),
         std::string("/bin/sh"), std::string("-ceu"), std::string(kBunTargetLifecycleCommand), std::string("helox-bun-target"), target}) boundary.add_argv(value);
    const auto topology = PNPMTargetLifecycleTestTopology(true);
    const auto pnpm_topology = PNPMTargetLifecycleTestTopology();
    if (!IsExactBunTargetLifecycleBoundary(boundary, kProfileNPM, &topology) ||
        IsExactPNPMTargetLifecycleBoundary(boundary, kProfileNPM, &topology) ||
        IsExactBunTargetLifecycleBoundary(boundary, kProfileBunResolver, &topology) ||
        IsExactBunTargetLifecycleBoundary(boundary, kProfileNPM, nullptr) ||
        IsExactBunTargetLifecycleBoundary(boundary, kProfileNPM, &pnpm_topology)) return false;
    for (int test = 0; test < 10; ++test) {
      auto changed = topology;
      switch (test) {
        case 0: changed.sealed = false; break;
        case 1: changed.snapshot_seen = false; break;
        case 2: changed.namespace_id = 0; break;
        case 3: changed.expected.pop_back(); changed.anchors.erase(4); break;
        case 4: changed.expected.back().read_only = true; break;
        case 5: changed.expected.back().filesystem_type = "tmpfs"; break;
        case 6: changed.expected.back().mount_class = "helper"; break;
        case 7: changed.anchors.erase(4); break;
        case 8: changed.anchors.emplace(5, MountAnchor{5, target, "workspace"}); break;
        case 9: changed.expected[0].read_only = false; break;
      }
      if (IsExactBunTargetLifecycleBoundary(boundary, kProfileNPM, &changed)) return false;
    }
    for (int index = 0; index < boundary.argv_size(); ++index) {
      auto changed = boundary; changed.set_argv(index, "changed");
      if (IsExactBunTargetLifecycleBoundary(changed, kProfileNPM, &topology)) return false;
      changed = boundary; changed.mutable_argv()->DeleteSubrange(index, 1);
      if (IsExactBunTargetLifecycleBoundary(changed, kProfileNPM, &topology)) return false;
    }
    auto extra = boundary; extra.add_argv("extra");
    if (IsExactBunTargetLifecycleBoundary(extra, kProfileNPM, &topology)) return false;
  }
  for (const auto& tail : {"", "@scope", "@/pkg", "../pkg", "pkg/child", "pkg/node_modules", "pkg/node_modules/", "pkg//node_modules/child", "pkg;id", "$(id)", "pkg\\child", "pkg/node_modules/../child"}) {
    if (ValidBunTargetLifecyclePath(prefix + tail)) return false;
  }
  return !ValidBunTargetLifecyclePath(prefix + std::string(512, 'a'));
}

bool VerifyBunTargetLifecycleProductionPath() {
  const std::string prefix = std::string(kBunTargetProjectMount) + "/project/node_modules/";
  return VerifyPNPMTargetLifecycleProductionPathForTarget(prefix + "pkg", true) &&
      VerifyPNPMTargetLifecycleProductionPathForTarget(prefix + "@scope/pkg/node_modules/@other/child", true);
}

bool VerifyExactNpmNodeInterpreterTransition() {
  gvisor::common::ContextData context;
  context.set_container_id(kThirtyFirstID);
  context.set_thread_group_id(290);
  context.set_thread_group_start_time_ns(2900);
  context.set_parent_thread_group_id(289);
  context.set_is_exec_session(false);
  context.set_process_name("npm");

  gvisor::sentry::ExecveInfo npm;
  *npm.mutable_context_data() = context;
  npm.set_binary_path(kNpmCLIPath);
  npm.set_execfn(kNpmPath);
  npm.add_argv(kNpmPath);
  npm.add_argv("install");
  npm.add_argv("--ignore-scripts=false");
  npm.add_argv("--no-audit");
  npm.add_argv("--no-fund");
  npm.add_argv("--offline");
  npm.add_argv("--no-update-notifier");
  npm.add_argv("/tmp/artifact.tgz");
  if (ProcessClassForPath(npm.binary_path(), kProfileNPM) != ProcessClass::kNpm ||
      !IsExactNpmCLILauncher(npm)) return false;

  ProcessState state;
  if (!RegisterGroup(&state, context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kCloneChild, false, true)) return false;
  auto group = state.groups.find(context.thread_group_id());
  if (group == state.groups.end() || !MayArmExactNpmNodeTransition(npm, kProfileNPM, group->second)) return false;
  group->second.npm_node_transition_pending = true;
  state.expected_groups.emplace(context.thread_group_id(),
      ProcessState::ExpectedGroup{context.thread_group_start_time_ns(), ProcessClass::kNpm});

  gvisor::sentry::ExecveInfo node;
  *node.mutable_context_data() = context;
  node.mutable_context_data()->set_process_name("node");
  node.set_binary_path(kNodePath);
  node.set_execfn(kNodePath);
  node.add_argv("node");
  node.add_argv(kNpmPath);
  node.add_argv("install");
  node.add_argv("--ignore-scripts=false");
  node.add_argv("--no-audit");
  node.add_argv("--no-fund");
  node.add_argv("--offline");
  node.add_argv("--no-update-notifier");
  node.add_argv("/tmp/artifact.tgz");
  auto expected = state.expected_groups.find(context.thread_group_id());
  if (expected == state.expected_groups.end() ||
      !IsExactNpmNodeTransition(node, kProfileNPM, context.thread_group_id(), group->second, expected->second)) return false;

  // Exact identities and argv are required; no pathname, argument or group
  // variation can arm or consume this one-shot transition.
  auto wrong_npm_path = npm; wrong_npm_path.set_binary_path("/tmp/npm-cli.js");
  auto wrong_npm_execfn = npm; wrong_npm_execfn.set_execfn("/tmp/npm");
  auto wrong_npm_argv = npm; wrong_npm_argv.set_argv(6, "--update-notifier");
  auto missing_notifier = npm; missing_notifier.mutable_argv()->DeleteSubrange(6, 1);
  auto extra_npm_argv = npm; extra_npm_argv.add_argv("extra");
  if (IsExactNpmCLILauncher(wrong_npm_path) || IsExactNpmCLILauncher(wrong_npm_execfn) ||
      IsExactNpmCLILauncher(wrong_npm_argv) || IsExactNpmCLILauncher(missing_notifier) ||
      IsExactNpmCLILauncher(extra_npm_argv)) return false;

  auto wrong_node_path = node; wrong_node_path.set_binary_path("/usr/bin/node");
  auto wrong_node_execfn = node; wrong_node_execfn.set_execfn("/usr/bin/node");
  auto wrong_node_argv = node; wrong_node_argv.set_argv(7, "--prefer-online");
  auto different_group = node; different_group.mutable_context_data()->set_thread_group_id(291);
  auto different_start = node; different_start.mutable_context_data()->set_thread_group_start_time_ns(2901);
  if (IsExactNpmNodeInterpreter(wrong_node_path) || IsExactNpmNodeInterpreter(wrong_node_execfn) ||
      IsExactNpmNodeInterpreter(wrong_node_argv) ||
      IsExactNpmNodeTransition(different_group, kProfileNPM, context.thread_group_id(), group->second, expected->second) ||
      IsExactNpmNodeTransition(different_start, kProfileNPM, context.thread_group_id(), group->second, expected->second)) return false;

  auto invalid_group = group->second;
  invalid_group.root_eligible = true;
  if (IsExactNpmNodeTransition(node, kProfileNPM, context.thread_group_id(), invalid_group, expected->second)) return false;
  invalid_group = group->second; invalid_group.root_consumed = false;
  if (IsExactNpmNodeTransition(node, kProfileNPM, context.thread_group_id(), invalid_group, expected->second)) return false;
  invalid_group = group->second; invalid_group.trusted_control_network_active = true;
  if (IsExactNpmNodeTransition(node, kProfileNPM, context.thread_group_id(), invalid_group, expected->second)) return false;
  invalid_group = group->second; invalid_group.provenance = ProcessState::Provenance::kDirectExecRoot;
  if (IsExactNpmNodeTransition(node, kProfileNPM, context.thread_group_id(), invalid_group, expected->second)) return false;
  invalid_group = group->second; invalid_group.provenance = ProcessState::Provenance::kOCIRoot;
  if (IsExactNpmNodeTransition(node, kProfileNPM, context.thread_group_id(), invalid_group, expected->second)) return false;
  invalid_group = group->second; invalid_group.role = ProcessState::Role::kArtifact;
  if (IsExactNpmNodeTransition(node, kProfileNPM, context.thread_group_id(), invalid_group, expected->second)) return false;

  expected->second.process_class = ProcessClass::kNode;
  group->second.npm_node_transition_pending = false;
  if (IsExactNpmNodeTransition(node, kProfileNPM, context.thread_group_id(), group->second, expected->second) ||
      IsExactNpmNodeTransition(npm, kProfileNPM, context.thread_group_id(), group->second, expected->second)) return false;
  return !group->second.root_eligible && group->second.root_consumed &&
      !group->second.trusted_control_network_active &&
      strcmp(NetworkProcessRelation(context, state), "CONTROL_GROUP") == 0;
}

bool SetupTestSession(int output, const std::string& remote, const std::string& control,
                      const char* container_id, const char* profile, int* client_out) {
  if (!RegisterProfile(control, container_id, profile)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) { if (client >= 0) close(client); return false; }
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(container_id);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) ||
      !ExpectRecord(output, container_id, "container-start")) {
    close(client);
    return false;
  }
  gvisor::syscall::Execve execve;
  execve.mutable_context_data()->set_container_id(container_id);
  execve.mutable_context_data()->set_thread_group_id(7);
  execve.mutable_context_data()->set_thread_group_start_time_ns(1);
  execve.mutable_context_data()->set_parent_thread_group_id(0);
  execve.mutable_context_data()->set_is_exec_session(true);
  execve.mutable_context_data()->set_process_name("python");
  execve.set_pathname("/usr/local/bin/python");
  execve.set_sysno(kSyscallExecve);
  execve.add_argv("python");
  gvisor::sentry::ExecveInfo execResolved;
  *execResolved.mutable_context_data() = execve.context_data();
  execResolved.set_binary_path("/usr/local/bin/python");
  if (!SendDirectLaunchExec(output, client, execve, execResolved) ||
      !ExpectRecord(output, container_id, "process-exec-expected")) {
    close(client);
    return false;
  }
  *client_out = client;
  return true;
}

gvisor::syscall::Open BuildOpen(const char* container_id, const char* path, uint32_t flags,
                                int32_t tid = 7, int64_t t_start = 1, uint64_t sysno = 257) {
  gvisor::syscall::Open open;
  open.mutable_context_data()->set_container_id(container_id);
  open.mutable_context_data()->set_thread_group_id(7);
  open.mutable_context_data()->set_thread_group_start_time_ns(1);
  open.mutable_context_data()->set_thread_id(tid);
  open.mutable_context_data()->set_thread_start_time_ns(t_start);
  open.mutable_context_data()->set_process_name("python");
  open.set_pathname(path);
  open.set_flags(flags);
  open.set_sysno(sysno);
  return open;
}

gvisor::syscall::OpenResult BuildOpenResult(const char* container_id, const char* path, uint64_t mount_id,
                                           bool success, int32_t errorno, uint32_t flags,
                                           int32_t tid = 7, int64_t t_start = 1, uint64_t sysno = 257) {
  gvisor::syscall::OpenResult result;
  result.mutable_context_data()->set_container_id(container_id);
  result.mutable_context_data()->set_thread_group_id(7);
  result.mutable_context_data()->set_thread_group_start_time_ns(1);
  result.mutable_context_data()->set_thread_id(tid);
  result.mutable_context_data()->set_thread_start_time_ns(t_start);
  result.mutable_context_data()->set_process_name("python");
  result.set_resolved_pathname(path);
  result.set_mount_id(mount_id);
  result.set_success(success);
  result.set_errorno(errorno);
  result.set_flags(flags);
  result.set_sysno(sysno);
  return result;
}

bool VerifyCargoBuildConfigurationBoundary() {
  int output[2]; if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
  timeval timeout{1,0};if(setsockopt(output[1],SOL_SOCKET,SO_RCVTIMEO,&timeout,sizeof(timeout))!=0){close(output[0]);close(output[1]);return false;}
  bool ok=true;
  for(const char* path:{"/tmp/haa-cargo-input/cargo-home/config.toml","/tmp/haa-cargo-input/cargo-home/config","/tmp/.cargo/config.toml"}) {
    for(bool relative:{false,true}) {
      ProcessState state;auto open=BuildOpen(kFirstID,relative?"relative-config":path,577);
      if(!RegisterGroup(&state,open.context_data(),ProcessState::Role::kArtifact,ProcessState::Provenance::kCloneChild,false,true)){ok=false;continue;}
      MountAnchor anchor{2,"/tmp","workspace"};
      if(!relative && ClassifyFilesystemOpen(open,state,kProfileCargoBuild,&anchor)!=FilesystemClass::kOutside)ok=false;
      std::string payload,id;const char* reason=nullptr;NormalizedCounts counts;
      if(!open.SerializeToString(&payload)||!ParseOpenAndSend(payload.data(),payload.size(),output[0],&id,kProfileCargoBuild,&state,&counts,&reason)){ok=false;continue;}
      if(!relative && !ExpectRecordExact(output[1],id.c_str(),"filesystem-outside-workspace"))ok=false;
      if(relative && !ExpectNoRecord(output[1]))ok=false;
      auto result=BuildOpenResult(kFirstID,path,2,true,0,577);
      TopologyState topology;topology.sealed=true;topology.anchors.emplace(2,anchor);
      if(!result.SerializeToString(&payload)||!ParseOpenResultAndSend(payload.data(),payload.size(),output[0],&id,kProfileCargoBuild,&state,&counts,topology,&reason))ok=false;
      if(relative && !ExpectRecordExact(output[1],id.c_str(),"filesystem-outside-workspace"))ok=false;
      if(!relative && !ExpectNoRecord(output[1]))ok=false;
      if(counts.immediate_records!=1 || counts.workspace_access!=0)ok=false;
      open.set_pathname(path);open.set_flags(0);
      if(ClassifyFilesystemOpen(open,state,kProfileCargoBuild,&anchor)!=FilesystemClass::kWorkspace)ok=false;
      open.set_flags(577);
      for(const char* profile:{kProfileGoBuild,kProfilePyPI,kProfileCargoResolver})
        if(ClassifyFilesystemOpen(open,state,profile,&anchor)!=FilesystemClass::kWorkspace)ok=false;
      state.groups[7].role=ProcessState::Role::kControl;
      if(ClassifyFilesystemOpen(open,state,kProfileCargoBuild,&anchor)!=FilesystemClass::kWorkspace)ok=false;
    }
  }
  close(output[0]);close(output[1]);return ok;
}

bool VerifyOpenResultNegativeMatrix(int output, const std::string& remote, const std::string& control) {
  // 1. Missing RESULT (pending open at stream-end) -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFortyNinthID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFortyNinthID, "/tmp/missing_result.txt", 0);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    close(client);
    if (!ExpectRecordExact(output, kFortyNinthID, "stream-fault", "STREAM_FAULT")) return false;
  }

  // 2. Duplicate RESULT -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftiethID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFiftiethID, "/tmp/dup_result.txt", 0);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res1 = BuildOpenResult(kFiftiethID, "/tmp/dup_result.txt", 2, true, 0, 0);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res1)) { close(client); return false; }
    auto res2 = BuildOpenResult(kFiftiethID, "/tmp/dup_result.txt", 2, true, 0, 0);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res2)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftiethID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 3. RESULT without ENTER -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftyFirstID, kProfilePyPI, &client)) return false;
    auto res = BuildOpenResult(kFiftyFirstID, "/tmp/no_enter.txt", 2, true, 0, 0);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftyFirstID, "stream-fault", "STREAM_FAULT", "OPEN_RESULT_CORRELATION")) { close(client); return false; }
    close(client);
  }

  // 4. Wrong thread_id -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftySecondID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFiftySecondID, "/tmp/wrong_tid.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kFiftySecondID, "/tmp/wrong_tid.txt", 2, true, 0, 0, 8, 1);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftySecondID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 5. Wrong thread_start_time_ns -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftyThirdID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFiftyThirdID, "/tmp/wrong_tstart.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kFiftyThirdID, "/tmp/wrong_tstart.txt", 2, true, 0, 0, 7, 2);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftyThirdID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 6. Wrong thread_group_id -> PROCESS_PROVENANCE_UNKNOWN
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftyFourthID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFiftyFourthID, "/tmp/wrong_tgid.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kFiftyFourthID, "/tmp/wrong_tgid.txt", 2, true, 0, 0, 7, 1);
    res.mutable_context_data()->set_thread_group_id(99);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftyFourthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) { close(client); return false; }
    close(client);
  }

  // 7. Wrong thread_group_start_time_ns -> PROCESS_PROVENANCE_UNKNOWN
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftyFifthID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFiftyFifthID, "/tmp/wrong_tgstart.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kFiftyFifthID, "/tmp/wrong_tgstart.txt", 2, true, 0, 0, 7, 1);
    res.mutable_context_data()->set_thread_group_start_time_ns(999);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftyFifthID, "stream-fault", "PROCESS_PROVENANCE_UNKNOWN")) { close(client); return false; }
    close(client);
  }

  // 8. Syscall kind (sysno) mismatch -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftySixthID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFiftySixthID, "/tmp/wrong_sysno.txt", 0, 7, 1, 257);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kFiftySixthID, "/tmp/wrong_sysno.txt", 2, true, 0, 0, 7, 1, 2);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftySixthID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 9. Effective flags mismatch -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftySeventhID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFiftySeventhID, "/tmp/wrong_flags.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kFiftySeventhID, "/tmp/wrong_flags.txt", 2, true, 0, 1, 7, 1);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftySeventhID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 10. Malformed failure result (non-empty resolved path) -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftyEighthID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFiftyEighthID, "/tmp/bad_fail.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kFiftyEighthID, "/tmp/bad_fail.txt", 0, false, 2, 0, 7, 1);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftyEighthID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 11a. Malformed success result (zero mount ID) -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kFiftyNinthID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kFiftyNinthID, "/tmp/zero_mount.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kFiftyNinthID, "/tmp/zero_mount.txt", 0, true, 0, 0, 7, 1);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kFiftyNinthID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 11b. Malformed success result (relative/non-normalized path) -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kSixtiethID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kSixtiethID, "relative.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kSixtiethID, "relative.txt", 2, true, 0, 0, 7, 1);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kSixtiethID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 11c. Malformed success result (non-zero errno on success) -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kSixtyFirstID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kSixtyFirstID, "/tmp/success_with_errno.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kSixtyFirstID, "/tmp/success_with_errno.txt", 2, true, 2, 0, 7, 1);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kSixtyFirstID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 12. Success with unknown mount ID -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kSixtySecondID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kSixtySecondID, "/tmp/unknown_mount.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kSixtySecondID, "/tmp/unknown_mount.txt", 999, true, 0, 0, 7, 1);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kSixtySecondID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 13. Success with path/mount disagreement -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kSixtyThirdID, kProfilePyPI, &client)) return false;
    auto open = BuildOpen(kSixtyThirdID, "/etc/shadow", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open)) { close(client); return false; }
    auto res = BuildOpenResult(kSixtyThirdID, "/etc/shadow", 2 /*mount 2 is /tmp*/, true, 0, 0, 7, 1);
    if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res)) { close(client); return false; }
    if (!ExpectRecordExact(output, kSixtyThirdID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  // 14. Second ENTER while same Task already has pending open -> STREAM_FAULT
  {
    int client = -1;
    if (!SetupTestSession(output, remote, control, kSixtyFourthID, kProfilePyPI, &client)) return false;
    auto open1 = BuildOpen(kSixtyFourthID, "/tmp/first.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open1)) { close(client); return false; }
    auto open2 = BuildOpen(kSixtyFourthID, "/tmp/second.txt", 0, 7, 1);
    if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open2)) { close(client); return false; }
    if (!ExpectRecordExact(output, kSixtyFourthID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
    close(client);
  }

  return true;
}

bool VerifyOpenResultPositiveMatrix(int output, const std::string& remote, const std::string& control) {
  int client = -1;
  if (!SetupTestSession(output, remote, control, kSixtyFifthID, kProfilePyPI, &client)) return false;

  // 1. Absolute workspace read -> aggregated, no immediate record
  auto open1 = BuildOpen(kSixtyFifthID, "/tmp/test_abs.txt", 0, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open1)) { close(client); return false; }
  auto res1 = BuildOpenResult(kSixtyFifthID, "/tmp/test_abs.txt", 2, true, 0, 0, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res1)) { close(client); return false; }
  if (!ExpectNoRecord(output)) { close(client); return false; }

  // 2. Relative workspace read with AT_FDCWD -> aggregated, no immediate record
  auto open2 = BuildOpen(kSixtyFifthID, "test_rel.txt", 0, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open2)) { close(client); return false; }
  auto res2 = BuildOpenResult(kSixtyFifthID, "/tmp/test_rel.txt", 2, true, 0, 0, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res2)) { close(client); return false; }
  if (!ExpectNoRecord(output)) { close(client); return false; }

  // 3. Pinned runtime read -> aggregated, no immediate record
  auto open3 = BuildOpen(kSixtyFifthID, "/usr/local/lib/python3.14/os.py", 0, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open3)) { close(client); return false; }
  auto res3 = BuildOpenResult(kSixtyFifthID, "/usr/local/lib/python3.14/os.py", 1, true, 0, 0, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res3)) { close(client); return false; }
  if (!ExpectNoRecord(output)) { close(client); return false; }

  // 4. Outside read -> filesystem-outside-workspace
  auto open4 = BuildOpen(kSixtyFifthID, "/root/secret.txt", 0, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open4)) { close(client); return false; }
  auto res4 = BuildOpenResult(kSixtyFifthID, "/root/secret.txt", 1, true, 0, 0, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res4)) { close(client); return false; }
  if (!ExpectRecordExact(output, kSixtyFifthID, "filesystem-outside-workspace")) { close(client); return false; }

  // 5. Honeytoken precedence -> honeytoken-access immediately on enter, clean result
  auto open5 = BuildOpen(kSixtyFifthID, "/tmp/.haa-honeytoken", 0, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open5)) { close(client); return false; }
  if (!ExpectRecordExact(output, kSixtyFifthID, "honeytoken-access")) { close(client); return false; }
  auto res5 = BuildOpenResult(kSixtyFifthID, "/tmp/.haa-honeytoken", 2, true, 0, 0, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res5)) { close(client); return false; }
  if (!ExpectNoRecord(output)) { close(client); return false; }

  // 6. Failed relative read with ENOENT -> clean completion, no false classification
  auto open6 = BuildOpen(kSixtyFifthID, "missing.py", 0, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open6)) { close(client); return false; }
  auto res6 = BuildOpenResult(kSixtyFifthID, "", 0, false, 2 /*ENOENT*/, 0, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res6)) { close(client); return false; }
  if (!ExpectNoRecord(output)) { close(client); return false; }

  // 7. Failed relative read with ENOTDIR -> clean completion
  auto open7 = BuildOpen(kSixtyFifthID, "file.txt/sub", 0, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open7)) { close(client); return false; }
  auto res7 = BuildOpenResult(kSixtyFifthID, "", 0, false, 20 /*ENOTDIR*/, 0, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res7)) { close(client); return false; }
  if (!ExpectNoRecord(output)) { close(client); return false; }

  // 8. Same TG with distinct interleaved Tasks
  auto open8a = BuildOpen(kSixtyFifthID, "/tmp/interleaved_a.txt", 0, 101, 500);
  auto open8b = BuildOpen(kSixtyFifthID, "/tmp/interleaved_b.txt", 0, 102, 600);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open8a)) { close(client); return false; }
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open8b)) { close(client); return false; }
  auto res8a = BuildOpenResult(kSixtyFifthID, "/tmp/interleaved_a.txt", 2, true, 0, 0, 101, 500);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res8a)) { close(client); return false; }
  auto res8b = BuildOpenResult(kSixtyFifthID, "/tmp/interleaved_b.txt", 2, true, 0, 0, 102, 600);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res8b)) { close(client); return false; }
  if (!ExpectNoRecord(output)) { close(client); return false; }

  // 9. Early write Finding + later RESULT without duplicate Finding
  auto open9 = BuildOpen(kSixtyFifthID, "/etc/issue", 1 /*kOpenWriteOnly*/, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open9)) { close(client); return false; }
  if (!ExpectRecordExact(output, kSixtyFifthID, "filesystem-outside-workspace")) { close(client); return false; }
  auto res9 = BuildOpenResult(kSixtyFifthID, "/etc/issue", 1, true, 0, 1, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res9)) { close(client); return false; }
  if (!ExpectNoRecord(output)) { close(client); return false; }

  // Close client to trigger stream termination
  close(client);
  // Total workspace access count: open1 (1) + open2 (1) + open8a (1) + open8b (1) = 4
  if (!ExpectCountedRecord(output, kSixtyFifthID, "filesystem-workspace-access", 4)) return false;
  return ExpectRecordExact(output, kSixtyFifthID, "stream-end");
}

bool VerifyNoBasenameTrust(int output, const std::string& remote, const std::string& control) {
  int client = -1;
  if (!SetupTestSession(output, remote, control, kSixtySixthID, kProfilePyPI, &client)) return false;

  // 1. Relative ENTER "wheels" resolving outside to /root/wheels on root mount 1 -> outside workspace!
  auto open1 = BuildOpen(kSixtySixthID, "wheels", 0, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open1)) { close(client); return false; }
  auto res1 = BuildOpenResult(kSixtySixthID, "/root/wheels", 1 /*root mount*/, true, 0, 0, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res1)) { close(client); return false; }
  if (!ExpectRecordExact(output, kSixtySixthID, "filesystem-outside-workspace")) { close(client); return false; }

  // 2. Relative ENTER "wheels" resolving to sealed workspace mount /tmp/wheels on mount 2 -> workspace!
  auto open2 = BuildOpen(kSixtySixthID, "wheels", 0, 7, 1);
  if (!SendEvent<gvisor::syscall::Open>(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open2)) { close(client); return false; }
  auto res2 = BuildOpenResult(kSixtySixthID, "/tmp/wheels", 2 /*workspace mount*/, true, 0, 0, 7, 1);
  if (!SendEvent<gvisor::syscall::OpenResult>(client, gvisor::common::MESSAGE_SYSCALL_OPEN_RESULT, res2)) { close(client); return false; }
  if (!ExpectNoRecord(output)) { close(client); return false; }

  close(client);
  // Exactly 1 workspace access
  if (!ExpectCountedRecord(output, kSixtySixthID, "filesystem-workspace-access", 1)) return false;
  return ExpectRecordExact(output, kSixtySixthID, "stream-end");
}

// A loader read is telemetry, not an authorization of the preceding exec.
bool VerifyPinnedUtilityLoaderReads() {
  for (const char* profile : {kProfilePyPI, kProfilePyTorchCPU, kProfilePyTorchCU126,
                             kProfilePyTorchCU130, kProfilePyTorchCU132}) {
    TopologyState topology;
    topology.expected.push_back(ExpectedMount{"/", "oci-root", "/", "", true, false, false, false});
    topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
    topology.namespace_id = 10; topology.snapshot_seen = true; topology.sealed = true;
    gvisor::sentry::ExecveInfo image;
    auto* context = image.mutable_context_data();
    context->set_container_id("aabbccddeeff0101");
    context->set_thread_group_id(91); context->set_thread_group_start_time_ns(910);
    context->set_parent_thread_group_id(90); context->set_process_name("python");
    image.set_binary_path("/usr/bin/uname"); image.set_execfn("/usr/bin/uname");
    image.add_argv("/usr/bin/uname"); image.add_argv("-p");
    int output[2]; if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
    auto check = [&](const gvisor::sentry::ExecveInfo& event, const TopologyState& mounts, bool readable) {
      ProcessState state;
      if (!RegisterGroup(&state, image.context_data(), ProcessState::Role::kArtifact,
                         ProcessState::Provenance::kCloneChild, false, true)) return false;
      ProfileRegistration registration; std::string payload, id; const char* reason = nullptr;
      if (!event.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(),
            output[0], &id, profile, &registration, &state, &reason, &mounts) ||
          !ExpectUnexpectedProcessRecord(output[1], id.c_str(), "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
      if (state.groups.at(91).role != ProcessState::Role::kArtifact || state.groups.at(91).root_eligible ||
          IsTrustedControlNetwork(event.context_data(), state)) return false;
      gvisor::syscall::Open read;
      *read.mutable_context_data() = event.context_data(); read.set_flags(557056);
      for (const char* comm : {"uname", "python", "worker"}) {
        read.mutable_context_data()->set_process_name(comm);
        for (const char* path : {"/etc/ld.so.cache", "/usr/lib/x86_64-linux-gnu/libc.so.6"}) {
          read.set_pathname(path);
          if ((ClassifyFilesystemOpen(read, state, profile) == FilesystemClass::kRuntimeRoot) != readable) return false;
        }
      }
      for (const char* path : {"/etc/shadow", "/usr/local/lib/python3.14/os.py", "/usr/lib/x86_64-linux-gnu/libutil.so.1"}) {
        read.set_pathname(path);
        if (ClassifyFilesystemOpen(read, state, profile) == FilesystemClass::kRuntimeRoot) return false;
      }
      read.set_pathname("/etc/ld.so.cache"); read.set_flags(kOpenWriteOnly);
      if (ClassifyFilesystemOpen(read, state, profile) != FilesystemClass::kOutside) return false;
      read.set_flags(0); read.mutable_context_data()->set_thread_group_start_time_ns(911);
      if (ClassifyFilesystemOpen(read, state, profile) == FilesystemClass::kRuntimeRoot) return false;
      // Re-exec replaces the kernel image classification; no inherited grant.
      auto replacement = event; replacement.set_binary_path("/tmp/uname");
      if (!replacement.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(),
            output[0], &id, profile, &registration, &state, &reason, &mounts) ||
          !ExpectUnexpectedProcessRecord(output[1], id.c_str(), "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;
      *read.mutable_context_data() = event.context_data();
      if (ClassifyFilesystemOpen(read, state, profile) == FilesystemClass::kRuntimeRoot) return false;
      return true;
    };
    bool ok = check(image, topology, true);
    auto fake = image; fake.set_binary_path("/tmp/uname"); ok = ok && check(fake, topology, false);
    auto unsealed = topology; unsealed.sealed = false; ok = ok && check(image, unsealed, false);
    auto writable = topology; writable.expected[0].read_only = false; ok = ok && check(image, writable, false);
    for (const char* shadow : {"/usr/bin/uname", "/etc/ld.so.cache"}) {
      auto substituted = topology; substituted.anchors.emplace(2, MountAnchor{2, shadow, "workspace"});
      ok = ok && check(image, substituted, false);
    }
    close(output[0]); close(output[1]); if (!ok) return false;
  }
  return true;
}

bool VerifyBoundedRuntimeCacheQuery() {
  for (const char* profile : {kProfilePyPI, kProfilePyTorchCPU, kProfilePyTorchCU126,
                            kProfilePyTorchCU130, kProfilePyTorchCU132}) {
    TopologyState topology;
    topology.expected.push_back(ExpectedMount{"/", "oci-root", "/", "", true, false, false, false});
    topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
    topology.namespace_id = 10; topology.snapshot_seen = true; topology.sealed = true;
    gvisor::sentry::ExecveInfo query;
    auto* context = query.mutable_context_data();
    context->set_container_id("aabbccddeeff0101");
    context->set_thread_group_id(91); context->set_thread_group_start_time_ns(910);
    context->set_parent_thread_group_id(90); context->set_process_name("spoofed");
    query.set_binary_path("/usr/sbin/ldconfig"); query.set_execfn("/sbin/ldconfig");
    query.add_argv("/sbin/ldconfig"); query.add_argv("-p");
    query.add_env("LC_ALL=C"); query.add_env("LANG=C");
    int output[2]; if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
    auto run = [&](const gvisor::sentry::ExecveInfo& event, const TopologyState* mounts,
                   bool expected, ProcessState* retained = nullptr) {
      ProcessState state;
      if (!RegisterGroup(&state, query.context_data(), ProcessState::Role::kArtifact,
                         ProcessState::Provenance::kCloneChild, false, true)) return false;
      ProfileRegistration registration;
      std::string payload, id; const char* reason = nullptr;
      if (!event.SerializeToString(&payload) || !ParseSentryProcessAndClassify(
            payload.data(), payload.size(), output[0], &id, profile, &registration, &state, &reason, mounts)) return false;
      const auto& group = state.groups.at(91);
      char record[1024];
      const ssize_t size = recv(output[1], record, sizeof(record), 0);
      const std::string received = size > 0 ? std::string(record, size) : "";
      const bool event_ok = expected
          ? received == "{\"container_id\":\"" + id + "\",\"kind\":\"process-exec-expected\"}"
          : received.find("\"kind\":\"process-exec-unexpected\"") != std::string::npos &&
            received.find("\"parent_relation\":\"ARTIFACT_GROUP\"") != std::string::npos;
      const bool ok = event_ok &&
          group.role == ProcessState::Role::kArtifact && !group.root_eligible &&
          !group.trusted_control_network_active && !IsTrustedControlNetwork(event.context_data(), state) &&
          group.runtime_cache_query == expected;
      if (retained != nullptr) *retained = state;
      return ok;
    };
    ProcessState accepted;
    bool ok = run(query, &topology, true, &accepted);
    auto invalid = query; invalid.set_binary_path("/tmp/ldconfig"); ok = ok && run(invalid, &topology, false);
    invalid = query; invalid.set_execfn("/tmp/alias"); ok = ok && run(invalid, &topology, false);
    invalid = query; invalid.add_argv("-C"); ok = ok && run(invalid, &topology, false);
    invalid = query; invalid.set_argv(1, "-v"); ok = ok && run(invalid, &topology, false);
    invalid = query; invalid.add_env("LD_PRELOAD=/tmp/fake.so"); ok = ok && run(invalid, &topology, false);
    invalid = query; invalid.set_env(0, "LOCPATH=/tmp"); ok = ok && run(invalid, &topology, false);
    invalid = query; invalid.clear_env(); ok = ok && run(invalid, &topology, false);
    ok = ok && run(query, nullptr, false);
    auto wrong = topology; wrong.sealed = false; ok = ok && run(query, &wrong, false);
    wrong = topology; wrong.expected[0].read_only = false; ok = ok && run(query, &wrong, false);
    wrong = topology; wrong.anchors.emplace(2, MountAnchor{2, "/usr/sbin", "system"}); ok = ok && run(query, &wrong, false);
    wrong = topology; wrong.anchors.emplace(2, MountAnchor{2, "/etc/ld.so.cache", "system"}); ok = ok && run(query, &wrong, false);
    gvisor::syscall::Open open;
    *open.mutable_context_data() = query.context_data(); open.set_pathname("/etc/ld.so.cache");
    const MountAnchor root{1, "/", "oci-root"};
    ok = ok && ClassifyFilesystemOpen(open, accepted, profile, &root) == FilesystemClass::kRuntimeRoot;
    open.set_flags(kOpenReadWrite);
    ok = ok && ClassifyFilesystemOpen(open, accepted, profile, &root) == FilesystemClass::kOutside;
    open.set_flags(0); open.set_pathname("/etc/ld.so.conf");
    ok = ok && ClassifyFilesystemOpen(open, accepted, profile, &root) != FilesystemClass::kRuntimeRoot;
    open.set_pathname("/etc/ld.so.cache");
    const MountAnchor substitute{2, "/etc/ld.so.cache", "system"};
    ok = ok && ClassifyFilesystemOpen(open, accepted, profile, &substitute) != FilesystemClass::kRuntimeRoot;
    // A child inherits ARTIFACT role, never this current-image operation.
    gvisor::sentry::CloneInfo clone;
    *clone.mutable_context_data() = query.context_data();
    clone.set_created_thread_group_id(92); clone.set_created_thread_start_time_ns(920);
    std::string clone_payload, clone_id; const char* clone_reason = nullptr;
    ok = ok && clone.SerializeToString(&clone_payload) && ParseSentryClone(
        clone_payload.data(), clone_payload.size(), output[0], &clone_id, &accepted, &clone_reason) &&
        accepted.groups.at(92).role == ProcessState::Role::kArtifact &&
        !accepted.groups.at(92).runtime_cache_query && !accepted.groups.at(92).root_eligible;
    // A later exec must discard query state and remain unexpected.
    invalid = query; invalid.set_binary_path("/bin/sh"); invalid.set_execfn("/bin/sh");
    ProfileRegistration registration; std::string payload, id; const char* reason = nullptr;
    ok = ok && invalid.SerializeToString(&payload) && ParseSentryProcessAndClassify(
        payload.data(), payload.size(), output[0], &id, profile, &registration, &accepted, &reason, &topology) &&
        ExpectUnexpectedProcessRecord(output[1], id.c_str(), "SENTRY_EXEC", "SHELL", "ARTIFACT_ROLE", "ARTIFACT_GROUP") && !accepted.groups.at(91).runtime_cache_query;
    close(output[0]); close(output[1]); if (!ok) return false;
  }
  return true;
}

bool VerifyUnexpectedExecDiagnosticRetention() {
  if (!VerifyPinnedUtilityLoaderReads() || !VerifyBoundedRuntimeCacheQuery()) return false;
  gvisor::sentry::ExecveInfo query;
  query.set_binary_path("/usr/sbin/ldconfig");
  query.add_argv("/sbin/ldconfig");
  query.add_argv("-p");
  if (!IsExactLdconfigCacheQuery(query)) return false;
  auto wrong_query = query;
  wrong_query.set_binary_path("/tmp/ldconfig");
  if (IsExactLdconfigCacheQuery(wrong_query)) return false;
  wrong_query = query;
  wrong_query.add_argv("-C");
  if (IsExactLdconfigCacheQuery(wrong_query)) return false;
  wrong_query = query;
  wrong_query.set_argv(1, "-v");
  if (IsExactLdconfigCacheQuery(wrong_query)) return false;
  // 1. Verify ClassifyCommandShape
  gvisor::sentry::ExecveInfo lock_exec;
  lock_exec.add_argv("sh");
  lock_exec.add_argv("-c");
  lock_exec.add_argv("npm install --package-lock-only");
  if (ClassifyCommandShape(lock_exec) != CommandPhase::kLockGeneration) return false;

  gvisor::sentry::ExecveInfo manifest_exec;
  manifest_exec.add_argv("sh");
  manifest_exec.add_argv("-c");
  manifest_exec.add_argv("echo '{\"name\":\"test\"}' > /tmp/haa-resolver/package.json");
  if (ClassifyCommandShape(manifest_exec) != CommandPhase::kManifestWrite) return false;

  gvisor::sentry::ExecveInfo lock_read_exec;
  lock_read_exec.add_argv("cat");
  lock_read_exec.add_argv("/tmp/haa-resolver/package-lock.json");
  if (ClassifyCommandShape(lock_read_exec) != CommandPhase::kLockRead) return false;

  gvisor::sentry::ExecveInfo readiness_exec;
  readiness_exec.add_argv("grep");
  readiness_exec.add_argv("CapInh");
  readiness_exec.add_argv("/proc/1/status");
  if (ClassifyCommandShape(readiness_exec) != CommandPhase::kReadinessCheck) return false;

  gvisor::sentry::ExecveInfo version_exec;
  version_exec.add_argv("npm");
  version_exec.add_argv("--version");
  if (ClassifyCommandShape(version_exec) != CommandPhase::kNpmVersion) return false;

  gvisor::sentry::ExecveInfo other_exec;
  other_exec.add_argv("ls");
  if (ClassifyCommandShape(other_exec) != CommandPhase::kOther) return false;

  // 2. Direct exec root group creation with CommandPhase
  ProcessState state;
  gvisor::common::ContextData root_ctx;
  root_ctx.set_container_id(kFirstID);
  root_ctx.set_thread_group_id(500);
  root_ctx.set_thread_group_start_time_ns(5000);
  root_ctx.set_parent_thread_group_id(0);
  root_ctx.set_is_exec_session(true);
  root_ctx.set_process_name("haa-boundary");
  if (!RegisterGroup(&state, root_ctx, ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, false, false,
                     CommandPhase::kLockGeneration)) return false;
  auto root_group = state.groups.find(500);
  if (root_group == state.groups.end() || root_group->second.command_phase != CommandPhase::kLockGeneration) return false;

  // 3. Sentry clone inherits CommandPhase
  gvisor::sentry::CloneInfo clone_msg;
  *clone_msg.mutable_context_data() = root_ctx;
  clone_msg.set_created_thread_group_id(501);
  clone_msg.set_created_thread_start_time_ns(5010);
  std::string clone_bytes;
  clone_msg.SerializeToString(&clone_bytes);
  std::string cid = kFirstID;
  const char* clone_reason = nullptr;
  if (!ParseSentryClone(clone_bytes.data(), clone_bytes.size(), -1, &cid, &state, &clone_reason)) return false;
  auto child_group = state.groups.find(501);
  if (child_group == state.groups.end() ||
      child_group->second.command_phase != CommandPhase::kLockGeneration ||
      child_group->second.provenance != ProcessState::Provenance::kCloneChild) return false;

  // 4. RecordUnexpectedExec latches the first unexpected exec
  if (state.first_unexpected_exec.present) return false;
  gvisor::common::ContextData child_ctx = root_ctx;
  child_ctx.set_thread_group_id(501);
  child_ctx.set_parent_thread_group_id(500);
  child_ctx.set_thread_group_start_time_ns(5010);
  child_group->second.diagnostic_image = DiagnosticImage::kEnv;
  child_group->second.trusted_control_network_active = true;
  state.expected_groups.emplace(501, ProcessState::ExpectedGroup{5010, ProcessClass::kNpm});
  const auto snapshot = CaptureExecDiagnostic(child_group->second, child_ctx,
      state.expected_groups.at(501).process_class, ProcessClass::kNode, DiagnosticImage::kNode);
  auto moved_expected = std::move(state.expected_groups);
  child_group->second.trusted_control_network_active = false;
  child_group->second.diagnostic_image = DiagnosticImage::kNode;
  RecordUnexpectedExec(&state.first_unexpected_exec, snapshot, "UNEXPECTED_INTERPRETER", "PARENT_CHILD");
  if (!state.first_unexpected_exec.trusted_control_network_active ||
      state.first_unexpected_exec.previous_image != DiagnosticImage::kEnv ||
      state.first_unexpected_exec.current_image != DiagnosticImage::kNode ||
      moved_expected.at(501).process_class != state.first_unexpected_exec.previous_class) return false;
  if (DiagnosticImageForPath("/tmp/private-secret-node") != DiagnosticImage::kUnknown ||
      strcmp(DiagnosticImageName(DiagnosticImageForPath("/tmp/private-secret-node")), "UNKNOWN") != 0) return false;
  if (!state.first_unexpected_exec.present ||
      state.first_unexpected_exec.phase != CommandPhase::kLockGeneration ||
      state.first_unexpected_exec.process_id != 501 ||
      state.first_unexpected_exec.parent_id != 500 ||
      state.first_unexpected_exec.previous_class != ProcessClass::kNpm ||
      state.first_unexpected_exec.current_class != ProcessClass::kNode ||
      strcmp(state.first_unexpected_exec.classification_reason, "UNEXPECTED_INTERPRETER") != 0 ||
      state.first_unexpected_exec.role != ProcessState::Role::kControl ||
      state.first_unexpected_exec.provenance != ProcessState::Provenance::kCloneChild ||
      strcmp(state.first_unexpected_exec.parent_relation, "PARENT_CHILD") != 0) return false;

  // 5. Subsequent unexpected execs do NOT overwrite the latched first unexpected exec
  ProcessState::GroupState other_group = child_group->second;
  other_group.command_phase = CommandPhase::kOther;
  gvisor::common::ContextData other_ctx = child_ctx;
  other_ctx.set_thread_group_id(999);
  other_ctx.set_parent_thread_group_id(0);
  const auto later = CaptureExecDiagnostic(other_group, other_ctx,
      ProcessClass::kShell, ProcessClass::kShell, DiagnosticImage::kShell);
  RecordUnexpectedExec(&state.first_unexpected_exec, later, "SECOND_EXEC", "NONE");
  if (!state.first_unexpected_exec.present ||
      state.first_unexpected_exec.process_id != 501 ||
      state.first_unexpected_exec.phase != CommandPhase::kLockGeneration ||
      strcmp(state.first_unexpected_exec.classification_reason, "UNEXPECTED_INTERPRETER") != 0) return false;

  // Diagnostic state cannot affect policy classification.
  ProcessState plain = state;
  plain.expected_groups = moved_expected;
  ProcessState diagnosed = plain;
  diagnosed.groups.at(501).diagnostic_image = DiagnosticImage::kBoundary;
  diagnosed.first_unexpected_exec = snapshot;
  const auto plain_result = IsExpectedProcess(kNodePath, child_ctx, kProfileNPM, &plain);
  const auto diagnosed_result = IsExpectedProcess(kNodePath, child_ctx, kProfileNPM, &diagnosed);
  if (plain_result.expected != diagnosed_result.expected ||
      plain_result.process_class != diagnosed_result.process_class ||
      strcmp(plain_result.reason, diagnosed_result.reason) != 0 ||
      strcmp(plain_result.parent_relation, diagnosed_result.parent_relation) != 0) return false;

  // 6. Verify enum names: bounded, non-empty, and free of whitespace / slashes
  const CommandPhase phases[] = {
      CommandPhase::kUnknown, CommandPhase::kNpmVersion, CommandPhase::kManifestWrite,
      CommandPhase::kLockGeneration, CommandPhase::kLockRead, CommandPhase::kReadinessCheck,
      CommandPhase::kOther};
  for (const auto phase : phases) {
    const char* name = CommandPhaseName(phase);
    if (name == nullptr || strlen(name) == 0 || strchr(name, ' ') != nullptr || strchr(name, '/') != nullptr) return false;
  }

  const FaultSite sites[] = {
      FaultSite::kNone, FaultSite::kRecvTrunc, FaultSite::kRecvShort,
      FaultSite::kProfileLookup, FaultSite::kEventLimit, FaultSite::kHeaderSize,
      FaultSite::kDroppedCount, FaultSite::kContainerStart, FaultSite::kSentryClone,
      FaultSite::kSentryExec, FaultSite::kExecSyscall, FaultSite::kOpen,
      FaultSite::kOpenResult, FaultSite::kTopologySnapshot, FaultSite::kTopologyMutation,
      FaultSite::kConnect, FaultSite::kSocket, FaultSite::kRaw, FaultSite::kFdTrack,
      FaultSite::kUnknownMessage, FaultSite::kRecvError, FaultSite::kUnsealedTopology,
      FaultSite::kPendingSockets, FaultSite::kPendingOpens, FaultSite::kWorkspaceSend};
  for (const auto site : sites) {
    const char* name = FaultSiteName(site);
    if (name == nullptr || strlen(name) == 0 || strchr(name, ' ') != nullptr || strchr(name, '/') != nullptr) return false;
  }

  // 7. Verify terminal fault site retention
  state.terminal_fault_site = FaultSite::kRecvError;
  if (strcmp(FaultSiteName(state.terminal_fault_site), "RECV_ERROR") != 0) return false;
  state.terminal_fault_site = FaultSite::kSentryExec;
  if (strcmp(FaultSiteName(state.terminal_fault_site), "SENTRY_EXEC") != 0) return false;

  return true;
}

bool VerifyResolverNpmVersionNodeTransition() {
  gvisor::common::ContextData context;
  context.set_container_id(kThirtyFirstID);
  context.set_thread_group_id(300);
  context.set_thread_group_start_time_ns(3000);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("node");

  // 1. Exact canonical NPM_VERSION transition message
  gvisor::sentry::ExecveInfo node;
  *node.mutable_context_data() = context;
  node.set_binary_path(kNodePath);
  node.set_execfn(kNodePath);
  node.add_argv("node");
  node.add_argv(kNpmPath);
  node.add_argv("--version");

  if (ProcessClassForPath(node.binary_path(), kProfileNPM) != ProcessClass::kNode ||
      !IsExactNpmVersionNodeInterpreter(node)) return false;

  ProcessState state;
  if (!RegisterGroup(&state, context, ProcessState::Role::kControl,
                     ProcessState::Provenance::kDirectExecRoot, false, true,
                     CommandPhase::kNpmVersion)) return false;
  auto group = state.groups.find(context.thread_group_id());
  if (group == state.groups.end()) return false;
  group->second.trusted_control_network_active = true;
  group->second.npm_version_node_transition_pending = true;
  state.expected_groups.emplace(context.thread_group_id(),
      ProcessState::ExpectedGroup{context.thread_group_start_time_ns(), ProcessClass::kNpm});

  auto expected = state.expected_groups.find(context.thread_group_id());
  if (expected == state.expected_groups.end() ||
      !IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                               group->second, expected->second)) return false;

  // 2. Changed command shape is rejected
  auto wrong_version = node; wrong_version.set_argv(2, "--help");
  auto missing_version = node; missing_version.mutable_argv()->DeleteSubrange(2, 1);
  auto extra_argv = node; extra_argv.add_argv("extra");
  auto empty_argv = node; empty_argv.clear_argv();
  if (IsExactNpmVersionNodeInterpreter(wrong_version) ||
      IsExactNpmVersionNodeInterpreter(missing_version) ||
      IsExactNpmVersionNodeInterpreter(extra_argv) ||
      IsExactNpmVersionNodeInterpreter(empty_argv)) return false;
  if (IsExactResolverNpmVersionNodeTransition(wrong_version, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second) ||
      IsExactResolverNpmVersionNodeTransition(missing_version, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second) ||
      IsExactResolverNpmVersionNodeTransition(extra_argv, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second)) return false;

  // 3. A consumed transition cannot be reused.
  auto invalid_group = group->second;
  invalid_group.npm_version_node_transition_pending = false;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;

  // 4. Wrong predecessor class is rejected
  auto invalid_expected = expected->second;
  invalid_expected.process_class = ProcessClass::kNode;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              group->second, invalid_expected)) return false;
  invalid_expected.process_class = ProcessClass::kShell;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              group->second, invalid_expected)) return false;

  // 5. Wrong current identity is rejected
  auto wrong_node_path = node; wrong_node_path.set_binary_path("/usr/bin/node");
  auto wrong_node_execfn = node; wrong_node_execfn.set_execfn("/usr/bin/node");
  auto wrong_node_argv0 = node; wrong_node_argv0.set_argv(0, "/usr/bin/node");
  auto wrong_node_argv1 = node; wrong_node_argv1.set_argv(1, "/tmp/npm");
  if (IsExactNpmVersionNodeInterpreter(wrong_node_path) ||
      IsExactNpmVersionNodeInterpreter(wrong_node_execfn) ||
      IsExactNpmVersionNodeInterpreter(wrong_node_argv0) ||
      IsExactNpmVersionNodeInterpreter(wrong_node_argv1)) return false;
  if (IsExactResolverNpmVersionNodeTransition(wrong_node_path, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second) ||
      IsExactResolverNpmVersionNodeTransition(wrong_node_execfn, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second) ||
      IsExactResolverNpmVersionNodeTransition(wrong_node_argv0, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second) ||
      IsExactResolverNpmVersionNodeTransition(wrong_node_argv1, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second)) return false;

  // 6. Wrong role is rejected
  invalid_group = group->second;
  invalid_group.role = ProcessState::Role::kArtifact;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  invalid_group.role = ProcessState::Role::kUnknown;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;

  // 7. Wrong provenance is rejected
  invalid_group = group->second;
  invalid_group.provenance = ProcessState::Provenance::kCloneChild;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  invalid_group.provenance = ProcessState::Provenance::kOCIRoot;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  invalid_group.provenance = ProcessState::Provenance::kUnknown;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;

  // 8. Wrong tracked-group / parent state is rejected
  auto different_group = node; different_group.mutable_context_data()->set_thread_group_id(999);
  auto different_start = node; different_start.mutable_context_data()->set_thread_group_start_time_ns(9999);
  if (IsExactResolverNpmVersionNodeTransition(different_group, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second) ||
      IsExactResolverNpmVersionNodeTransition(different_start, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second)) return false;

  // 9. Artifact role cannot use this transition
  invalid_group = group->second;
  invalid_group.role = ProcessState::Role::kArtifact;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;

  // 10. Handoff state cannot regain CONTROL
  invalid_group = group->second;
  invalid_group.handoff_target_pending = true;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;

  // 11. Descendant / reexec / same-path trust regain remains rejected
  expected->second.process_class = ProcessClass::kNode;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second)) return false;
  const ProcessClassification reexec = IsExpectedProcess(kNodePath, context, kProfileNPM, &state);
  if (reexec.expected || strcmp(reexec.reason, "BOOTSTRAP_ENDED") != 0) return false;

  // 12. Existing frozen Artifact npm lifecycle transition still behaves identically
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              group->second, expected->second)) return false;

  // 13. Unrelated node transitions still fail closed
  invalid_group = group->second;
  invalid_group.command_phase = CommandPhase::kLockGeneration;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  invalid_group.command_phase = CommandPhase::kOther;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  invalid_group = group->second;
  invalid_group.root_eligible = true;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  invalid_group = group->second;
  invalid_group.root_consumed = false;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  invalid_group = group->second;
  invalid_group.trusted_control_network_active = false;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  invalid_group = group->second;
  invalid_group.demotion_pending = true;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  invalid_group = group->second;
  invalid_group.launch_target_pending = true;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfileNPM, context.thread_group_id(),
                                              invalid_group, expected->second)) return false;
  if (IsExactResolverNpmVersionNodeTransition(node, kProfilePyPI, context.thread_group_id(),
                                              group->second, expected->second)) return false;

  return true;
}

// This drives the production SENTRY_EXEC classifier rather than testing the
// transition predicate in isolation. Each rejection below starts from the
// complete accepted boundary -> setpriv -> npm launcher path and changes one
// relevant fact only.
bool VerifyResolverNpmVersionProductionPath() {
  struct Path {
    gvisor::sentry::ExecveInfo boundary;
    gvisor::sentry::ExecveInfo demotion;
    gvisor::sentry::ExecveInfo npm;
    gvisor::sentry::ExecveInfo node;
  } path;
  gvisor::common::ContextData context;
  context.set_container_id(kThirtySecondID);
  context.set_thread_group_id(320);
  context.set_thread_group_start_time_ns(3200);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);

  *path.boundary.mutable_context_data() = context;
  path.boundary.mutable_context_data()->set_process_name("haa-boundary");
  path.boundary.set_binary_path(kBoundaryHelperPath);
  path.boundary.set_execfn(kBoundaryHelperPath);
  path.boundary.add_argv(kBoundaryHelperPath);
  path.boundary.add_argv(kLaunchMode);
  const std::string admission_nonce = TestAdmissionNonce(320, 3200);
  path.boundary.add_argv(admission_nonce);
  path.boundary.add_argv("npm");
  path.boundary.add_argv("--version");

  path.demotion = path.boundary;
  path.demotion.mutable_context_data()->set_process_name("setpriv");
  path.demotion.set_binary_path(kSetprivPath);
  path.demotion.set_execfn(kSetprivPath);
  path.demotion.clear_argv();
  path.demotion.add_argv(kSetprivPath);
  path.demotion.add_argv("--reuid=1000");
  path.demotion.add_argv("--regid=1000");
  path.demotion.add_argv("--clear-groups");
  path.demotion.add_argv("--inh-caps=-all");
  path.demotion.add_argv("--ambient-caps=-all");
  path.demotion.add_argv("--bounding-set=-all");
  path.demotion.add_argv("--no-new-privs");
  path.demotion.add_argv("--");
  path.demotion.add_argv(kNpmCLIPath);

  path.npm = path.boundary;
  path.npm.mutable_context_data()->set_process_name("npm");
  path.npm.set_binary_path(kNpmCLIPath);
  path.npm.set_execfn(kNpmPath);
  path.npm.clear_argv();
  path.npm.add_argv(kNpmPath);
  path.npm.add_argv("--version");

  path.node = path.npm;
  path.node.mutable_context_data()->set_process_name("node");
  path.node.set_binary_path(kNodePath);
  path.node.set_execfn(kNodePath);
  path.node.clear_argv();
  path.node.add_argv("node");
  path.node.add_argv(kNpmPath);
  path.node.add_argv("--version");

  int sockets[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, sockets) != 0) return false;
  auto parse_expected = [&](const gvisor::sentry::ExecveInfo& event, ProcessState* state,
                            const char* kind, const char* reason = nullptr) {
    std::string payload;
    std::string container_id;
    const char* parse_reason = nullptr;
    auto registration = TestProfileWithAdmission(kProfileNPM, kLaunchMode, admission_nonce);
    const bool serialized = event.SerializeToString(&payload);
    const bool parsed = serialized && ParseSentryProcessAndClassify(
        payload.data(), payload.size(), sockets[0], &container_id, kProfileNPM, &registration, state, &parse_reason);
    return parsed && ExpectRecord(sockets[1], kThirtySecondID, kind, reason);
  };
  auto parse_unexpected = [&](const gvisor::sentry::ExecveInfo& event, ProcessState* state,
                              const char* reason) {
    std::string payload;
    std::string container_id;
    const char* parse_reason = nullptr;
    auto registration = TestProfileWithAdmission(kProfileNPM, kLaunchMode, admission_nonce);
    const bool serialized = event.SerializeToString(&payload);
    const bool parsed = serialized && ParseSentryProcessAndClassify(
        payload.data(), payload.size(), sockets[0], &container_id, kProfileNPM, &registration, state, &parse_reason);
    return parsed && ExpectUnexpectedProcessRecord(sockets[1], kThirtySecondID, "SENTRY_EXEC", "NODE",
                                                   reason, "TRACKED_GROUP");
  };
  auto parse_rejected = [&](const gvisor::sentry::ExecveInfo& event, ProcessState* state) {
    std::string payload;
    std::string container_id;
    const char* parse_reason = nullptr;
    auto registration = TestProfileWithAdmission(kProfileNPM, kLaunchMode, admission_nonce);
    return event.SerializeToString(&payload) &&
        !ParseSentryProcessAndClassify(payload.data(), payload.size(), sockets[0], &container_id,
                                       kProfileNPM, &registration, state, &parse_reason);
  };
  auto baseline = [&](ProcessState* state, gvisor::sentry::ExecveInfo* node,
                      const gvisor::sentry::ExecveInfo* boundary,
                      const gvisor::sentry::ExecveInfo* npm) {
    const auto& initial = boundary == nullptr ? path.boundary : *boundary;
    const auto& launcher = npm == nullptr ? path.npm : *npm;
    if (!parse_expected(initial, state, "process-exec-expected") ||
        !parse_expected(path.demotion, state, "process-exec-expected") ||
        !parse_expected(launcher, state, "process-exec-expected")) return false;
    *node = path.node;
    return true;
  };

  bool ok = true;
  ProcessState state;
  gvisor::sentry::ExecveInfo node;
  ok = baseline(&state, &node, nullptr, nullptr) && parse_expected(node, &state, "process-exec-expected");
  const auto tracked = state.expected_groups.find(context.thread_group_id());
  const auto group = state.groups.find(context.thread_group_id());
  ok = ok && tracked != state.expected_groups.end() && group != state.groups.end() &&
      tracked->second.process_class == ProcessClass::kNode &&
      !group->second.npm_version_node_transition_pending &&
      group->second.npm_version_node_transition_consumed;
  // This begins with the real accepted resolver production path above. Each
  // negative changes one authorization fact for the exact pinned runtime
  // identity; no diagnostic state participates in the decision.
  auto runtime_open = BuildOpen(kThirtySecondID, "/proc/meminfo", 0, 320, 3200);
  *runtime_open.mutable_context_data() = node.context_data();
  ok = ok && ClassifyFilesystemOpen(runtime_open, state, kProfileNPM) ==
      FilesystemClass::kHelperOnly;
  ProcessState runtime_changed = state;
  runtime_changed.groups[320].command_phase = CommandPhase::kOther;
  ok = ok && ClassifyFilesystemOpen(runtime_open, runtime_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  runtime_changed = state;
  runtime_changed.expected_groups[320].process_class = ProcessClass::kNpm;
  ok = ok && ClassifyFilesystemOpen(runtime_open, runtime_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  runtime_changed = state;
  runtime_changed.groups[320].role = ProcessState::Role::kArtifact;
  ok = ok && ClassifyFilesystemOpen(runtime_open, runtime_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  runtime_changed = state;
  runtime_changed.groups[320].provenance = ProcessState::Provenance::kOCIRoot;
  ok = ok && ClassifyFilesystemOpen(runtime_open, runtime_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  auto write_runtime_open = runtime_open;
  write_runtime_open.set_flags(kOpenWriteOnly);
  ok = ok && ClassifyFilesystemOpen(write_runtime_open, state, kProfileNPM) ==
      FilesystemClass::kOutside;
  auto unpinned_runtime_open = runtime_open;
  unpinned_runtime_open.set_pathname("/proc/unpinned-runtime-proof");
  ok = ok && ClassifyFilesystemOpen(unpinned_runtime_open, state, kProfileNPM) ==
      FilesystemClass::kUnknown;
  auto workspace_runtime_open = runtime_open;
  workspace_runtime_open.set_pathname("/tmp/npm-runtime-proof");
  ok = ok && ClassifyFilesystemOpen(workspace_runtime_open, state, kProfileNPM) ==
      FilesystemClass::kWorkspace;
  runtime_changed = state;
  runtime_changed.groups[320].handoff_target_pending = true;
  ok = ok && ClassifyFilesystemOpen(runtime_open, runtime_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  runtime_changed = state;
  runtime_changed.groups[320].demotion_pending = true;
  ok = ok && ClassifyFilesystemOpen(runtime_open, runtime_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  runtime_changed = state;
  runtime_changed.groups[320].launch_target_pending = true;
  ok = ok && ClassifyFilesystemOpen(runtime_open, runtime_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  runtime_changed = state;
  runtime_changed.groups[320].npm_version_node_transition_consumed = false;
  ok = ok && ClassifyFilesystemOpen(runtime_open, runtime_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  auto descendant_runtime_open = runtime_open;
  descendant_runtime_open.mutable_context_data()->set_parent_thread_group_id(321);
  ok = ok && ClassifyFilesystemOpen(descendant_runtime_open, state, kProfileNPM) ==
      FilesystemClass::kUnknown;
  ok = ok && parse_unexpected(node, &state, "BOOTSTRAP_ENDED");  // one-shot/re-exec
  ok = ok && !state.groups.at(context.thread_group_id()).npm_version_node_transition_consumed;

  ProcessState changed;
  gvisor::sentry::ExecveInfo changed_node;
  ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed_node.set_argv(2, "--help");
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed_node.add_argv("extra");
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed_node.set_argv(1, "--version"); changed_node.set_argv(2, kNpmPath);
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed_node.mutable_context_data()->set_parent_thread_group_id(77);
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed_node.mutable_context_data()->set_thread_group_start_time_ns(3201);
  ok = ok && parse_rejected(changed_node, &changed);
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed_node.mutable_context_data()->set_thread_group_id(321);
  ok = ok && parse_rejected(changed_node, &changed);

  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed.expected_groups[context.thread_group_id()].process_class = ProcessClass::kNode;
  ok = ok && parse_unexpected(changed_node, &changed, "BOOTSTRAP_ENDED");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed.groups[context.thread_group_id()].role = ProcessState::Role::kArtifact;
  {
    std::string payload, container_id; const char* parse_reason = nullptr;
    auto registration = TestProfileWithAdmission(kProfileNPM, kLaunchMode, admission_nonce);
    ok = ok && changed_node.SerializeToString(&payload) &&
        ParseSentryProcessAndClassify(payload.data(), payload.size(), sockets[0], &container_id,
                                      kProfileNPM, &registration, &changed, &parse_reason) &&
        ExpectUnexpectedProcessRecord(sockets[1], kThirtySecondID, "SENTRY_EXEC", "NODE",
                                      "ARTIFACT_ROLE", "ARTIFACT_GROUP");
  }
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed.groups[context.thread_group_id()].provenance = ProcessState::Provenance::kCloneChild;
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed.groups[context.thread_group_id()].command_phase = CommandPhase::kOther;
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed.groups[context.thread_group_id()].trusted_control_network_active = false;
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed.groups[context.thread_group_id()].handoff_target_pending = true;
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed.groups[context.thread_group_id()].demotion_pending = true;
  ok = ok && parse_rejected(changed_node, &changed);
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed.groups[context.thread_group_id()].launch_target_pending = true;
  ok = ok && parse_rejected(changed_node, &changed);

  gvisor::sentry::ExecveInfo altered_boundary = path.boundary;
  altered_boundary.add_argv("extra");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, &altered_boundary, nullptr);
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");
  gvisor::sentry::ExecveInfo altered_npm = path.npm;
  altered_npm.add_argv("extra");
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, &altered_npm);
  ok = ok && parse_unexpected(changed_node, &changed, "CLASS_MISMATCH");

  // A descendant cannot create a fresh direct-root authorization record.
  changed = ProcessState{}; ok = ok && baseline(&changed, &changed_node, nullptr, nullptr);
  changed_node.mutable_context_data()->set_thread_group_id(321);
  changed_node.mutable_context_data()->set_thread_group_start_time_ns(3201);
  changed_node.mutable_context_data()->set_parent_thread_group_id(context.thread_group_id());
  changed_node.mutable_context_data()->set_is_exec_session(false);
  ok = ok && parse_rejected(changed_node, &changed);

  close(sockets[0]);
  close(sockets[1]);
  return ok;
}

// Drives the actual SENTRY_EXEC and SENTRY_CLONE path for the one fixed
// GenerateLockfile command. Each negative starts from the full valid boundary
// lineage and changes exactly one authorization-relevant condition.
bool VerifyResolverLockGenerationProductionPath(bool update = false) {
  struct Path {
    gvisor::sentry::ExecveInfo boundary;
    gvisor::sentry::ExecveInfo demotion;
    gvisor::sentry::ExecveInfo shell;
    gvisor::sentry::CloneInfo clone;
    gvisor::sentry::ExecveInfo npm;
    gvisor::sentry::ExecveInfo node;
  } path;
  gvisor::common::ContextData root;
  root.set_container_id(kThirtySecondID);
  root.set_thread_group_id(340);
  root.set_thread_group_start_time_ns(3400);
  root.set_parent_thread_group_id(0);
  root.set_is_exec_session(true);

  *path.boundary.mutable_context_data() = root;
  path.boundary.mutable_context_data()->set_process_name("haa-boundary");
  path.boundary.set_binary_path(kBoundaryHelperPath);
  path.boundary.set_execfn(kBoundaryHelperPath);
  path.boundary.add_argv(kBoundaryHelperPath);
  path.boundary.add_argv(kLaunchMode);
  const std::string admission_nonce = TestAdmissionNonce(340, 3400);
  path.boundary.add_argv(admission_nonce);
  path.boundary.add_argv("/bin/sh");
  path.boundary.add_argv("-ceu");
  path.boundary.add_argv(update ? kLockUpdateCommand : kLockGenerationCommand);

  path.demotion = path.boundary;
  path.demotion.mutable_context_data()->set_process_name("setpriv");
  path.demotion.set_binary_path(kSetprivPath);
  path.demotion.set_execfn(kSetprivPath);
  path.demotion.clear_argv();
  const char* demotion[] = {kSetprivPath, "--reuid=1000", "--regid=1000", "--clear-groups",
      "--inh-caps=-all", "--ambient-caps=-all", "--bounding-set=-all", "--no-new-privs", "--", "/bin/sh"};
  for (const char* argument : demotion) path.demotion.add_argv(argument);

  path.shell = path.boundary;
  path.shell.mutable_context_data()->set_process_name("sh");
  path.shell.set_binary_path("/usr/bin/dash");
  path.shell.set_execfn("/bin/sh");
  path.shell.clear_argv();
  path.shell.add_argv("/bin/sh");
  path.shell.add_argv("-ceu");
  path.shell.add_argv(update ? kLockUpdateCommand : kLockGenerationCommand);

  *path.clone.mutable_context_data() = root;
  path.clone.set_created_thread_group_id(341);
  path.clone.set_created_thread_start_time_ns(3410);

  gvisor::common::ContextData child = root;
  child.set_thread_group_id(341);
  child.set_thread_group_start_time_ns(3410);
  child.set_parent_thread_group_id(340);
  // gVisor preserves the Docker-exec origin across the shell's clone. This is
  // deliberately not an authorization condition for the exact one-shot path.
  child.set_is_exec_session(true);
  child.set_process_name("npm");
  *path.npm.mutable_context_data() = child;
  path.npm.set_binary_path(kNpmCLIPath);
  path.npm.set_execfn(kNpmPath);
  path.npm.add_argv(kNpmPath);
  std::vector<const char*> lock_args = {"install", "--package-lock-only", "--ignore-scripts", "--no-audit", "--no-fund",
      "--registry=https://registry.npmjs.org/", "--userconfig=/tmp/haa-user.npmrc",
      "--globalconfig=/tmp/haa-global.npmrc"};
  if (update) {
    lock_args[0] = "update";
    lock_args.insert(lock_args.begin() + 5, "--save=false");
  }
  for (const char* argument : lock_args) path.npm.add_argv(argument);

  path.node = path.npm;
  path.node.mutable_context_data()->set_process_name("node");
  path.node.set_binary_path(kNodePath);
  path.node.set_execfn(kNodePath);
  path.node.clear_argv();
  path.node.add_argv("node");
  path.node.add_argv(kNpmPath);
  for (const char* argument : lock_args) path.node.add_argv(argument);

  int sockets[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, sockets) != 0) return false;
  auto parse = [&](const gvisor::sentry::ExecveInfo& event, ProcessState* state,
                   const char* expected_kind, const char* expected_class = nullptr,
                   const char* expected_reason = nullptr) {
    std::string payload, container_id;
    const char* reason = nullptr;
    auto registration = TestProfileWithAdmission(kProfileNPM, kLaunchMode, admission_nonce);
    const bool serialized = event.SerializeToString(&payload);
    const bool parsed = serialized && ParseSentryProcessAndClassify(
        payload.data(), payload.size(), sockets[0], &container_id, kProfileNPM, &registration, state, &reason);
    if (!parsed) return false;
    if (strcmp(expected_kind, "process-exec-unexpected") == 0) {
      return ExpectUnexpectedProcessRecord(sockets[1], kThirtySecondID, "SENTRY_EXEC", expected_class,
                                           expected_reason, "TRACKED_GROUP");
    }
    return ExpectRecord(sockets[1], kThirtySecondID, expected_kind);
  };
  auto parse_clone = [&](ProcessState* state) {
    std::string payload, container_id;
    const char* reason = nullptr;
    return path.clone.SerializeToString(&payload) &&
        ParseSentryClone(payload.data(), payload.size(), -1, &container_id, state, &reason);
  };
  auto baseline = [&](ProcessState* state, const gvisor::sentry::ExecveInfo* boundary = nullptr,
                      const gvisor::sentry::ExecveInfo* npm = nullptr) {
    const auto& initial = boundary == nullptr ? path.boundary : *boundary;
    const auto& launcher = npm == nullptr ? path.npm : *npm;
    return parse(initial, state, "process-exec-expected") &&
        parse(path.demotion, state, "process-exec-expected") &&
        parse(path.shell, state, "process-exec-expected") && parse_clone(state) &&
        parse(launcher, state, "process-exec-expected");
  };
  auto reject = [&](const gvisor::sentry::ExecveInfo& event, ProcessState* state) {
    std::string payload, container_id;
    const char* reason = nullptr;
    auto registration = TestProfileWithAdmission(kProfileNPM, kLaunchMode, admission_nonce);
    return event.SerializeToString(&payload) && !ParseSentryProcessAndClassify(
        payload.data(), payload.size(), sockets[0], &container_id, kProfileNPM, &registration, state, &reason);
  };

  bool ok = true;
  ProcessState state;
  ok = baseline(&state) &&
      state.groups.at(341).lock_generation_npm_node_transition_pending &&
      state.expected_groups.at(341).process_class == ProcessClass::kNpm &&
      state.groups.at(341).command_phase == CommandPhase::kLockGeneration &&
      state.groups.at(341).clone_creator_group_id == 340 &&
      parse(path.node, &state, "process-exec-expected") &&
      state.expected_groups.at(341).process_class == ProcessClass::kNode &&
      !state.groups.at(341).lock_generation_npm_node_transition_pending;

  // The exact immutable glibc resolver configuration read observed during the
  // Node lock-generation transition is accepted only in that complete state.
  auto host_conf_open = BuildOpen(kThirtySecondID, "/etc/host.conf", 0, 341, 3410);
  *host_conf_open.mutable_context_data() = path.node.context_data();
  ok = ok && ClassifyFilesystemOpen(host_conf_open, state, kProfileNPM) ==
      FilesystemClass::kHelperOnly;
  ProcessState host_conf_changed = state;
  host_conf_changed.groups.at(341).role = ProcessState::Role::kArtifact;
  ok = ok && ClassifyFilesystemOpen(host_conf_open, host_conf_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  host_conf_changed = state;
  host_conf_changed.groups.at(341).provenance = ProcessState::Provenance::kDirectExecRoot;
  ok = ok && ClassifyFilesystemOpen(host_conf_open, host_conf_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  host_conf_changed = state;
  host_conf_changed.groups.at(341).command_phase = CommandPhase::kOther;
  ok = ok && ClassifyFilesystemOpen(host_conf_open, host_conf_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  host_conf_open.set_flags(kOpenWriteOnly);
  ok = ok && ClassifyFilesystemOpen(host_conf_open, state, kProfileNPM) ==
      FilesystemClass::kOutside;
  host_conf_open.set_flags(0);
  ok = ok && ClassifyFilesystemOpen(host_conf_open, state, kProfileGitHub) ==
      FilesystemClass::kUnknown;
  host_conf_open.set_pathname("/etc/nsswitch.conf");
  ok = ok && ClassifyFilesystemOpen(host_conf_open, state, kProfileNPM) ==
      FilesystemClass::kHelperOnly;
  host_conf_open.set_pathname("/etc/resolv.conf");
  ok = ok && ClassifyFilesystemOpen(host_conf_open, state, kProfileNPM) ==
      FilesystemClass::kHelperOnly;
  host_conf_open.set_pathname("/etc/host.conf.unrecognized");
  ok = ok && ClassifyFilesystemOpen(host_conf_open, state, kProfileNPM) ==
      FilesystemClass::kUnknown;

  // /etc/gai.conf is a separate exact immutable glibc address-selection
  // configuration identity. It uses the same complete Node lock-generation
  // state as /etc/host.conf and is not a broader /etc allowance.
  auto gai_conf_open = BuildOpen(kThirtySecondID, "/etc/gai.conf", 0, 341, 3410);
  *gai_conf_open.mutable_context_data() = path.node.context_data();
  ok = ok && ClassifyFilesystemOpen(gai_conf_open, state, kProfileNPM) ==
      FilesystemClass::kHelperOnly;
  ProcessState gai_conf_changed = state;
  gai_conf_changed.groups.at(341).role = ProcessState::Role::kArtifact;
  ok = ok && ClassifyFilesystemOpen(gai_conf_open, gai_conf_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  gai_conf_changed = state;
  gai_conf_changed.expected_groups.at(341).process_class = ProcessClass::kNpm;
  ok = ok && ClassifyFilesystemOpen(gai_conf_open, gai_conf_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  gai_conf_changed = state;
  gai_conf_changed.groups.at(341).provenance = ProcessState::Provenance::kDirectExecRoot;
  ok = ok && ClassifyFilesystemOpen(gai_conf_open, gai_conf_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  gai_conf_changed = state;
  gai_conf_changed.groups.at(341).command_phase = CommandPhase::kOther;
  ok = ok && ClassifyFilesystemOpen(gai_conf_open, gai_conf_changed, kProfileNPM) ==
      FilesystemClass::kUnknown;
  gai_conf_open.set_flags(kOpenWriteOnly);
  ok = ok && ClassifyFilesystemOpen(gai_conf_open, state, kProfileNPM) ==
      FilesystemClass::kOutside;
  gai_conf_open.set_flags(0);
  ok = ok && ClassifyFilesystemOpen(gai_conf_open, state, kProfileGitHub) ==
      FilesystemClass::kUnknown;
  gai_conf_open.set_pathname("/etc/gai.conf.unrecognized");
  ok = ok && ClassifyFilesystemOpen(gai_conf_open, state, kProfileNPM) ==
      FilesystemClass::kUnknown;

  // Docker's generated /etc/hosts is a distinct SYSTEM mount. The exact
  // LOCK_GENERATION Node read is accepted only after production topology
  // reconciliation supplies that independent anchor fact.
  auto hosts_open = BuildOpen(kThirtySecondID, "/etc/hosts", 0, 341, 3410);
  *hosts_open.mutable_context_data() = path.node.context_data();
  const MountAnchor docker_hosts{41, "/etc/hosts", "system"};
  ok = ok && ClassifyFilesystemOpen(hosts_open, state, kProfileNPM, &docker_hosts) ==
      FilesystemClass::kHelperOnly;
  // The lexical path alone never authorizes the access.
  ok = ok && ClassifyFilesystemOpen(hosts_open, state, kProfileNPM) ==
      FilesystemClass::kUnknown;
  const MountAnchor oci_hosts{1, "/", "oci-root"};
  ok = ok && ClassifyFilesystemOpen(hosts_open, state, kProfileNPM, &oci_hosts) ==
      FilesystemClass::kUnknown;
  const MountAnchor wrong_system_mount{42, "/etc/hostname", "system"};
  ok = ok && ClassifyFilesystemOpen(hosts_open, state, kProfileNPM, &wrong_system_mount) ==
      FilesystemClass::kUnknown;
  hosts_open.set_pathname("/etc/hostname");
  ok = ok && ClassifyFilesystemOpen(hosts_open, state, kProfileNPM, &docker_hosts) ==
      FilesystemClass::kUnknown;
  hosts_open.set_pathname("/etc/hosts");
  ProcessState hosts_changed = state;
  hosts_changed.groups.at(341).role = ProcessState::Role::kArtifact;
  ok = ok && ClassifyFilesystemOpen(hosts_open, hosts_changed, kProfileNPM, &docker_hosts) ==
      FilesystemClass::kUnknown;
  hosts_changed = state;
  hosts_changed.groups.at(341).provenance = ProcessState::Provenance::kDirectExecRoot;
  ok = ok && ClassifyFilesystemOpen(hosts_open, hosts_changed, kProfileNPM, &docker_hosts) ==
      FilesystemClass::kUnknown;
  hosts_changed = state;
  hosts_changed.groups.at(341).command_phase = CommandPhase::kOther;
  ok = ok && ClassifyFilesystemOpen(hosts_open, hosts_changed, kProfileNPM, &docker_hosts) ==
      FilesystemClass::kUnknown;
  hosts_changed = state;
  hosts_changed.expected_groups.at(341).process_class = ProcessClass::kNpm;
  ok = ok && ClassifyFilesystemOpen(hosts_open, hosts_changed, kProfileNPM, &docker_hosts) ==
      FilesystemClass::kUnknown;
  hosts_open.set_flags(kOpenWriteOnly);
  ok = ok && ClassifyFilesystemOpen(hosts_open, state, kProfileNPM, &docker_hosts) ==
      FilesystemClass::kOutside;
  hosts_open.set_flags(0);
  ok = ok && ClassifyFilesystemOpen(hosts_open, state, kProfileGitHub, &docker_hosts) ==
      FilesystemClass::kUnknown;
  // A clone carrying the direct-root network exception is not a valid
  // LOCK_GENERATION resolver state.
  hosts_changed = state;
  hosts_changed.groups.at(341).trusted_control_network_active = true;
  ok = ok && ClassifyFilesystemOpen(hosts_open, hosts_changed, kProfileNPM, &docker_hosts) ==
      FilesystemClass::kUnknown;
  hosts_open.set_pathname("/etc/nearby-unrecognized");
  ok = ok && ClassifyFilesystemOpen(hosts_open, state, kProfileNPM, &docker_hosts) ==
      FilesystemClass::kUnknown;

  // Re-exec cannot consume the already-cleared one-shot again.
  ok = ok && parse(path.node, &state, "process-exec-unexpected", "NODE", "BOOTSTRAP_ENDED");

  ProcessState changed;
  // Exec-session origin is inherited in production, but origin itself does
  // not authorize this path. The same exact lineage is accepted with the
  // opposite bounded origin value when every authorization condition remains
  // unchanged.
  auto non_exec_npm = path.npm;
  auto non_exec_node = path.node;
  non_exec_npm.mutable_context_data()->set_is_exec_session(false);
  non_exec_node.mutable_context_data()->set_is_exec_session(false);
  changed = ProcessState{};
  ok = ok && baseline(&changed, nullptr, &non_exec_npm) &&
      changed.groups.at(341).lock_generation_npm_node_transition_pending &&
      parse(non_exec_node, &changed, "process-exec-expected") &&
      !changed.groups.at(341).lock_generation_npm_node_transition_pending;

  auto changed_node = path.node;
  changed = ProcessState{};
  ok = ok && baseline(&changed);
  changed.groups[341].command_phase = CommandPhase::kOther;
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed);
  changed.expected_groups[341].process_class = ProcessClass::kShell;
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed);
  changed_node.set_binary_path(kNpmCLIPath);
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NPM", "BOOTSTRAP_ENDED");
  changed_node = path.node; changed = ProcessState{}; ok = ok && baseline(&changed);
  changed.groups[341].provenance = ProcessState::Provenance::kDirectExecRoot;
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed);
  changed_node.mutable_context_data()->set_parent_thread_group_id(342);
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");
  changed_node = path.node; changed = ProcessState{}; ok = ok && baseline(&changed);
  changed_node.mutable_context_data()->set_thread_group_start_time_ns(3411);
  ok = ok && reject(changed_node, &changed);
  changed_node = path.node; changed = ProcessState{}; ok = ok && baseline(&changed);
  changed.groups[341].role = ProcessState::Role::kArtifact;
  {
    std::string payload, container_id; const char* reason = nullptr;
    auto registration = TestProfileWithAdmission(kProfileNPM, kLaunchMode, admission_nonce);
    ok = ok && changed_node.SerializeToString(&payload) && ParseSentryProcessAndClassify(
        payload.data(), payload.size(), sockets[0], &container_id, kProfileNPM, &registration, &changed, &reason) &&
        ExpectUnexpectedProcessRecord(sockets[1], kThirtySecondID, "SENTRY_EXEC", "NODE",
                                      "ARTIFACT_ROLE", "ARTIFACT_GROUP");
  }
  changed = ProcessState{}; ok = ok && baseline(&changed);
  changed.groups[341].handoff_target_pending = true;
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed);
  changed.groups[341].demotion_pending = true;
  ok = ok && reject(changed_node, &changed);
  changed = ProcessState{}; ok = ok && baseline(&changed);
  changed.groups[341].launch_target_pending = true;
  ok = ok && reject(changed_node, &changed);
  changed = ProcessState{}; ok = ok && baseline(&changed);
  changed.groups[341].trusted_control_network_active = true;
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");
  changed = ProcessState{}; ok = ok && baseline(&changed);
  changed_node.add_argv("extra");
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");
  changed_node = path.node; changed = ProcessState{}; ok = ok && baseline(&changed);
  changed_node.set_argv(2, "--no-audit"); changed_node.set_argv(4, "--package-lock-only");
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");
  changed_node = path.node; changed = ProcessState{}; ok = ok && baseline(&changed);
  changed.groups[341].lock_generation_npm_node_transition_pending = false;
  ok = ok && parse(changed_node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");

  // A phase lookalike and altered launcher cannot arm the special state.
  auto altered_boundary = path.boundary;
  altered_boundary.add_argv("extra");
  changed = ProcessState{}; ok = ok && baseline(&changed, &altered_boundary) &&
      !changed.groups.at(341).lock_generation_npm_node_transition_pending &&
      parse(path.node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");
  auto altered_npm = path.npm;
  altered_npm.add_argv("extra");
  changed = ProcessState{}; ok = ok && baseline(&changed, nullptr, &altered_npm) &&
      !changed.groups.at(341).lock_generation_npm_node_transition_pending &&
      parse(path.node, &changed, "process-exec-unexpected", "NODE", "CLASS_MISMATCH");

  close(sockets[0]);
  close(sockets[1]);
  return ok;
}

bool VerifyAuthoritativeProcessGroupLifecycle() {
  auto context_for = [](int32_t group_id, int64_t start_time_ns, int32_t parent_group_id = 0) {
    gvisor::common::ContextData context;
    context.set_container_id(kFirstID);
    context.set_thread_group_id(group_id);
    context.set_thread_group_start_time_ns(start_time_ns);
    context.set_parent_thread_group_id(parent_group_id);
    context.set_is_exec_session(parent_group_id == 0);
    context.set_process_name("haa-boundary");
    return context;
  };
  auto exit_for = [](const gvisor::common::ContextData& context) {
    gvisor::sentry::ExitNotifyParentInfo event;
    *event.mutable_context_data() = context;
    return event;
  };
  auto parse_exit = [&](const gvisor::sentry::ExitNotifyParentInfo& event, ProcessState* state,
                        const char** reason = nullptr) {
    std::string payload;
    std::string container_id;
    const char* local_reason = nullptr;
    const bool result = event.SerializeToString(&payload) && ParseSentryExitNotifyParent(
        payload.data(), payload.size(), &container_id, state, &local_reason);
    if (reason != nullptr) *reason = local_reason;
    return result;
  };

  bool ok = true;
  const auto root = context_for(700, 7000);
  ProcessState state;
  ok = ok && RegisterGroup(&state, root, ProcessState::Role::kControl,
                           ProcessState::Provenance::kDirectExecRoot, true, false) &&
      TrackExpectedProcessGroup(root, ProcessClass::kPython, &state) == TrackResult::kTracked;
  state.fd_states[700][3] = ProcessState::FDEntry{SocketClassification::kNetwork, AF_INET, false};
  state.pending_sockets[700] = 1;
  state.launch_roots[700] = 7000;
  state.launch_root_set = true;
  state.launch_root_active = true;
  state.launch_root_group_id = 700;
  state.launch_root_group_start_time_ns = 7000;
  state.pending_opens[{700, 7000}] = ProcessState::PendingOpen{700, 7000, 257, 0, false};
  const auto root_exit = exit_for(root);
  ok = ok && parse_exit(root_exit, &state) && state.groups.empty() && state.expected_groups.empty() &&
      state.fd_states.empty() && state.pending_sockets.empty() && state.launch_roots.empty() &&
      !state.launch_root_set && state.pending_opens.empty();

  const char* reason = nullptr;
  ok = ok && !parse_exit(root_exit, &state, &reason) && reason != nullptr &&
      strcmp(reason, "PROCESS_EXIT_UNKNOWN") == 0;
  auto unknown_exit = exit_for(context_for(701, 7010));
  reason = nullptr;
  ok = ok && !parse_exit(unknown_exit, &state, &reason) && reason != nullptr &&
      strcmp(reason, "PROCESS_EXIT_UNKNOWN") == 0;
  std::string malformed_container;
  reason = nullptr;
  ok = ok && !ParseSentryExitNotifyParent("\xff", 1, &malformed_container, &state, &reason) &&
      reason != nullptr && strcmp(reason, "PROCESS_EXIT_INVALID") == 0;

  // Sequential churn must reclaim historical identities, while an actually
  // concurrent 65th group remains fail-closed at the fixed bound.
  ProcessState sequential;
  for (int32_t group = 1; group <= 65; ++group) {
    const auto context = context_for(group, 10000 + group);
    if (!RegisterGroup(&sequential, context, ProcessState::Role::kControl,
                       ProcessState::Provenance::kDirectExecRoot, true, false) ||
        !parse_exit(exit_for(context), &sequential) || !sequential.groups.empty()) ok = false;
  }
  ProcessState concurrent;
  for (int32_t group = 1; group <= static_cast<int32_t>(kMaxTrackedProcessGroups); ++group) {
    ok = ok && RegisterGroup(&concurrent, context_for(group, 20000 + group),
                             ProcessState::Role::kControl,
                             ProcessState::Provenance::kDirectExecRoot, true, false);
  }
  ok = ok && !RegisterGroup(&concurrent, context_for(65, 20065), ProcessState::Role::kControl,
                             ProcessState::Provenance::kDirectExecRoot, true, false);

  // Arbitrarily old replays cannot recreate a root after lifecycle reclaim.
  // Their historical identity is deliberately forgotten; only a live exact
  // admission may create a fresh direct root.
  ProcessState reuse;
  const auto old_context = context_for(720, 7200);
  ok = ok && RegisterGroup(&reuse, old_context, ProcessState::Role::kControl,
                           ProcessState::Provenance::kDirectExecRoot, true, false) &&
      parse_exit(exit_for(old_context), &reuse);
  for (int32_t group = 800; group < 930; ++group) {
    const auto context = context_for(group, 8000 + group);
    ok = ok && RegisterGroup(&reuse, context, ProcessState::Role::kControl,
                             ProcessState::Provenance::kDirectExecRoot, true, false) &&
        parse_exit(exit_for(context), &reuse);
  }
  // The old identity has deliberately non-default authority bits. None may
  // appear on the later numeric-ID reuse.
  ProcessState trusted_old;
  ok = ok && RegisterGroup(&trusted_old, old_context, ProcessState::Role::kControl,
                           ProcessState::Provenance::kDirectExecRoot, true, false);
  trusted_old.groups.at(720).trusted_control_network_active = true;
  trusted_old.groups.at(720).command_phase = CommandPhase::kLockGeneration;
  trusted_old.expected_groups[720] = ProcessState::ExpectedGroup{7200, ProcessClass::kNode};
  trusted_old.fd_states[720][9] = ProcessState::FDEntry{SocketClassification::kNetwork, AF_INET, false};
  ok = ok && parse_exit(exit_for(old_context), &trusted_old) &&
      RegisterGroup(&trusted_old, context_for(720, 7201), ProcessState::Role::kControl,
                    ProcessState::Provenance::kDirectExecRoot, true, false) &&
      !trusted_old.groups.at(720).trusted_control_network_active &&
      trusted_old.groups.at(720).command_phase == CommandPhase::kUnknown &&
      trusted_old.expected_groups.empty() && trusted_old.fd_states.empty();
  int sockets[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, sockets) != 0) return false;
  gvisor::sentry::ExecveInfo old_exec;
  *old_exec.mutable_context_data() = old_context;
  old_exec.set_binary_path(kBoundaryHelperPath);
  old_exec.set_execfn(kBoundaryHelperPath);
  old_exec.add_argv(kBoundaryHelperPath);
  old_exec.add_argv(kLaunchMode);
  const std::string old_nonce = TestAdmissionNonce(720, 7200);
  old_exec.add_argv(old_nonce);
  std::string payload, container_id;
  reason = nullptr;
  ProfileRegistration no_old_admission{kProfilePyPI, {}, std::string(64, '0'), {}};
  ok = ok && old_exec.SerializeToString(&payload) && !ParseSentryProcessAndClassify(
      payload.data(), payload.size(), sockets[0], &container_id, kProfilePyPI, &no_old_admission, &reuse, &reason) &&
      reason != nullptr && strcmp(reason, "PROCESS_PROVENANCE_UNKNOWN") == 0 && reuse.groups.empty();
  const auto reused_context = context_for(720, 7201);
  ok = ok && RegisterGroup(&reuse, reused_context, ProcessState::Role::kControl,
                           ProcessState::Provenance::kDirectExecRoot, true, false);
  reason = nullptr;
  ok = ok && !parse_exit(exit_for(old_context), &reuse, &reason) && reason != nullptr &&
      strcmp(reason, "PROCESS_EXIT_IDENTITY_REUSED") == 0 &&
      reuse.groups.at(720).start_time_ns == 7201;

  // An exec transition keeps the same live group; only ExitNotifyParent
  // performs reclamation.
  ProcessState exec_state;
  const auto exec_context = context_for(725, 7250);
  ok = ok && RegisterGroup(&exec_state, exec_context, ProcessState::Role::kControl,
                           ProcessState::Provenance::kDirectExecRoot, true, false);
  gvisor::sentry::ExecveInfo live_exec;
  *live_exec.mutable_context_data() = exec_context;
  live_exec.set_binary_path(kBoundaryHelperPath);
  live_exec.set_execfn(kBoundaryHelperPath);
  live_exec.add_argv(kBoundaryHelperPath);
  live_exec.add_argv(kLaunchMode);
  payload.clear(); container_id.clear(); reason = nullptr;
  ok = ok && live_exec.SerializeToString(&payload) && ParseSentryProcessAndClassify(
      payload.data(), payload.size(), sockets[0], &container_id, kProfilePyPI, &exec_state, &reason) &&
      exec_state.groups.size() == 1 && exec_state.groups.at(725).start_time_ns == 7250;
  ok = ok && ExpectRecord(sockets[1], kFirstID, "process-exec-expected");

  // A living child retains only its original creator identity. Reusing the
  // creator's numeric ID cannot turn that old lineage into a new one.
  ProcessState lineage;
  const auto parent = context_for(730, 7300);
  ok = ok && RegisterGroup(&lineage, parent, ProcessState::Role::kControl,
                           ProcessState::Provenance::kDirectExecRoot, true, false);
  gvisor::sentry::CloneInfo clone;
  *clone.mutable_context_data() = parent;
  clone.set_created_thread_group_id(731);
  clone.set_created_thread_start_time_ns(7310);
  payload.clear(); container_id.clear(); reason = nullptr;
  ok = ok && clone.SerializeToString(&payload) && ParseSentryClone(
      payload.data(), payload.size(), -1, &container_id, &lineage, &reason) &&
      lineage.groups.at(731).clone_creator_group_start_time_ns == 7300 &&
      parse_exit(exit_for(parent), &lineage) && lineage.groups.find(731) != lineage.groups.end();
  const auto reused_parent = context_for(730, 7301);
  ok = ok && RegisterGroup(&lineage, reused_parent, ProcessState::Role::kArtifact,
                           ProcessState::Provenance::kDirectExecRoot, false, true) &&
      !HasExactCloneCreator(lineage.groups.at(731), lineage) &&
      parse_exit(exit_for(context_for(731, 7310, 730)), &lineage) &&
      lineage.groups.find(730) != lineage.groups.end();

  // TaskExit is intentionally not a lifecycle transition. It remains an
  // unknown event unless a separate future contract consumes it.
  ProcessState task_state;
  const auto task_context = context_for(740, 7400);
  ok = ok && RegisterGroup(&task_state, task_context, ProcessState::Role::kControl,
                           ProcessState::Provenance::kDirectExecRoot, true, false);
  gvisor::sentry::TaskExit task_exit;
  *task_exit.mutable_context_data() = task_context;
  std::string task_payload;
  Header task_header{static_cast<uint16_t>(sizeof(Header)),
                     static_cast<uint16_t>(gvisor::common::MESSAGE_SENTRY_TASK_EXIT), 0};
  NormalizedCounts counts;
  TopologyState topology;
  reason = nullptr;
  ok = ok && task_exit.SerializeToString(&task_payload) && !Handle(
      task_header, task_payload.data(), task_payload.size(), sockets[0], &container_id,
      kProfilePyPI, &task_state, &counts, &topology, &reason) &&
      task_state.groups.find(740) != task_state.groups.end();

  gvisor::sentry::CloneInfo invalid_clone;
  *invalid_clone.mutable_context_data() = context_for(750, 7500);
  invalid_clone.set_created_thread_group_id(751);
  invalid_clone.set_created_thread_start_time_ns(7510);
  ProcessState clone_failure;
  payload.clear(); container_id.clear(); reason = nullptr;
  ok = ok && invalid_clone.SerializeToString(&payload) && !ParseSentryClone(
      payload.data(), payload.size(), -1, &container_id, &clone_failure, &reason) &&
      clone_failure.groups.empty();
  close(sockets[0]);
  close(sockets[1]);
  return ok;
}

// This is the production SENTRY_EXEC root-creation parser, deliberately
// exercised without the transport fixture's automatic arm helper. A
// boundary-shaped exec is never sufficient by itself: only an exact live
// admission may create the initial CONTROL / DIRECT_EXEC_ROOT state.
bool VerifyDirectExecAdmission() {
  gvisor::common::ContextData context;
  context.set_container_id(kSixtyFourthID);
  context.set_thread_group_id(760);
  context.set_thread_group_start_time_ns(7600);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("haa-boundary");
  const std::string nonce = TestAdmissionNonce(760, 7600);
  gvisor::sentry::ExecveInfo boundary;
  *boundary.mutable_context_data() = context;
  boundary.set_binary_path(kBoundaryHelperPath);
  boundary.set_execfn(kBoundaryHelperPath);
  boundary.add_argv(kBoundaryHelperPath);
  boundary.add_argv(kLaunchMode);
  boundary.add_argv(nonce);
  boundary.add_argv("/bin/true");
  int sockets[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, sockets) != 0) return false;
  auto parse = [&](const gvisor::sentry::ExecveInfo& event,
                   ProfileRegistration* registration, ProcessState* state,
                   const char** reason = nullptr) {
    std::string payload;
    std::string container_id;
    const char* local_reason = nullptr;
    const bool parsed = event.SerializeToString(&payload) && ParseSentryProcessAndClassify(
        payload.data(), payload.size(), sockets[0], &container_id, kProfilePyPI,
        registration, state, &local_reason);
    if (reason != nullptr) *reason = local_reason;
    return parsed;
  };

  bool ok = true;
  ProcessState accepted;
  auto admission = TestProfileWithAdmission(kProfilePyPI, kLaunchMode, nonce);
  ok = ok && parse(boundary, &admission, &accepted) && admission.pending_admissions.size() == 1 &&
      admission.pending_admissions.front().state == ProfileRegistration::AdmissionState::kConsumed &&
      accepted.groups.size() == 1 && accepted.groups.at(760).role == ProcessState::Role::kControl &&
      accepted.groups.at(760).provenance == ProcessState::Provenance::kDirectExecRoot &&
      !accepted.groups.at(760).root_eligible && accepted.groups.at(760).root_consumed &&
      ExpectRecord(sockets[1], kSixtyFourthID, "process-exec-expected");
  const char* reason = nullptr;
  ok = ok && !parse(boundary, &admission, &accepted, &reason) && reason != nullptr &&
      strcmp(reason, "PROCESS_PROVENANCE_UNKNOWN") == 0;
	// Process reclamation is independent of authority cleanup: an exact
	// ExitNotifyParent may reclaim the root group before the controller asks
	// STATUS/COMPLETE, but it must not erase the retained CONSUMED fact.
	gvisor::sentry::ExitNotifyParentInfo exit;
	*exit.mutable_context_data() = context;
	std::string exit_payload;
	std::string exit_container;
	reason = nullptr;
	ok = ok && exit.SerializeToString(&exit_payload) && ParseSentryExitNotifyParent(
		exit_payload.data(), exit_payload.size(), &exit_container, &accepted, &reason) &&
		accepted.groups.empty() && admission.pending_admissions.size() == 1 &&
		admission.pending_admissions.front().state == ProfileRegistration::AdmissionState::kConsumed;

  ProcessState missing;
  ProfileRegistration no_admission{kProfilePyPI, {}, std::string(64, '0'), -1, {}};
  reason = nullptr;
  ok = ok && !parse(boundary, &no_admission, &missing, &reason) && reason != nullptr &&
      strcmp(reason, "PROCESS_PROVENANCE_UNKNOWN") == 0 && missing.groups.empty();
  ProcessState wrong_nonce;
  auto wrong = TestProfileWithAdmission(kProfilePyPI, kLaunchMode, TestAdmissionNonce(761, 7610));
  reason = nullptr;
  ok = ok && !parse(boundary, &wrong, &wrong_nonce, &reason) && reason != nullptr &&
      strcmp(reason, "PROCESS_PROVENANCE_UNKNOWN") == 0 && wrong_nonce.groups.empty();
  ProcessState wrong_mode;
  auto mode = TestProfileWithAdmission(kProfilePyPI, kPythonHandoffMode, nonce);
  reason = nullptr;
  ok = ok && !parse(boundary, &mode, &wrong_mode, &reason) && reason != nullptr &&
      strcmp(reason, "PROCESS_PROVENANCE_UNKNOWN") == 0 && wrong_mode.groups.empty();
  ProcessState malformed;
  auto malformed_event = boundary;
  malformed_event.set_argv(2, "not-a-nonce");
  auto malformed_admission = TestProfileWithAdmission(kProfilePyPI, kLaunchMode, nonce);
  reason = nullptr;
  ok = ok && !parse(malformed_event, &malformed_admission, &malformed, &reason) && reason != nullptr &&
      strcmp(reason, "PROCESS_PROVENANCE_UNKNOWN") == 0 && malformed.groups.empty();
  close(sockets[0]);
  close(sockets[1]);
  return ok;
}

bool VerifyDirectExecAdmissionBound(int output, const std::string& remote, const std::string& control) {
  if (!RegisterProfile(control, kSixtyFifthID, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) {
    if (client >= 0) close(client);
    return false;
  }
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kSixtyFifthID);
  start.mutable_context_data()->set_thread_group_id(900);
  start.mutable_context_data()->set_thread_group_start_time_ns(9000);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) ||
      !ExpectRecord(output, kSixtyFifthID, "container-start")) {
    close(client);
    return false;
  }
  std::vector<std::string> pending_nonces;
  std::vector<std::string> consumed_nonces;
  for (int index = 0; index < static_cast<int>(kMaxPendingDirectExecAdmissions); ++index) {
    const std::string nonce = TestAdmissionNonce(900 + index, 9000 + index);
    if (!ArmAdmission(control, kSixtyFifthID, kLaunchMode, nonce)) {
      close(client);
      return false;
    }
    if (index % 2 == 0) {
      if (!SendRealDirectExecTransition(output, client, kSixtyFifthID, kLaunchMode, nonce, 9000 + index, 90000 + index)) {
        close(client);
        return false;
      }
      consumed_nonces.push_back(nonce);
    } else {
      pending_nonces.push_back(nonce);
    }
  }
  const std::string overflow = TestAdmissionNonce(1000, 10000);
  // The 65th live admission receives a negative ACK; mixed (pending + consumed) bound is enforced.
  if (ArmAdmission(control, kSixtyFifthID, kLaunchMode, overflow)) {
    close(client);
    return false;
  }
  // Completing one consumed slot releases capacity.
  if (!CompleteAdmission(control, kSixtyFifthID, kLaunchMode, consumed_nonces.front())) {
    close(client);
    return false;
  }
  if (!ArmAdmission(control, kSixtyFifthID, kLaunchMode, overflow)) {
    close(client);
    return false;
  }
  if (!CancelAdmission(control, kSixtyFifthID, kLaunchMode, overflow)) {
    close(client);
    return false;
  }
  for (size_t index = 1; index < consumed_nonces.size(); ++index) {
    if (!CompleteAdmission(control, kSixtyFifthID, kLaunchMode, consumed_nonces[index])) {
      close(client);
      return false;
    }
  }
  for (size_t index = 0; index < pending_nonces.size(); ++index) {
    if (!CancelAdmission(control, kSixtyFifthID, kLaunchMode, pending_nonces[index])) {
      close(client);
      return false;
    }
  }
  close(client);
  return ExpectRecordExact(output, kSixtyFifthID, "stream-end");
}

bool VerifyStrictControlParser() {
  const std::string id = "aabbccddeeff0011";
  const std::string generation(64, 'a');
  const std::string nonce(64, 'b');
  int sockets[2];
  if (socketpair(AF_UNIX, SOCK_SEQPACKET, 0, sockets) != 0) return false;
  ControlPeer peer{sockets[0], false, false, false, "", ""};
  std::map<std::string, ProfileRegistration> profiles;
  auto rejected = [&](const std::string& body) {
    return !ParseControlRecord(body.data(), body.size(), &peer, &profiles);
  };
  const bool ok = rejected("{\"op\":\"arm\",\"container_id\":\"" + id + "\",\"session_generation\":\"" + generation +
                  "\",\"mode\":\"--launch\",\"nonce\":\"" + nonce + "\",\"unknown\":\"x\"}") &&
      rejected("{\"op\":\"arm\",\"op\":\"arm\",\"container_id\":\"" + id + "\",\"session_generation\":\"" + generation +
                  "\",\"mode\":\"--launch\",\"nonce\":\"" + nonce + "\"}") &&
      rejected("{\"op\":\"arm\",\"container_id\":\"" + id + "\",\"session_generation\":\"" + generation +
                  "\",\"mode\":{},\"nonce\":\"" + nonce + "\"}") &&
      rejected("{\"op\":\"arm\",\"container_id\":\"" + id + "\",\"session_generation\":\"short\",\"mode\":\"--launch\",\"nonce\":\"" + nonce + "\"}") &&
      rejected("{\"op\":\"arm\",\"container_id\":\"" + id + "\",\"session_generation\":\"" + generation +
                  "\",\"mode\":\"--launch\",\"nonce\":\"" + nonce + "\"} trailing");
  close(sockets[0]);
  close(sockets[1]);
  return ok;
}

// Exercises the strict observer parser's terminal slot protocol directly. The
// production SENTRY parser above proves the only PENDING->CONSUMED transition;
// this latch verifies that control cleanup observes that state rather than
// treating an already-consumed nonce as ambiguous cancellation.
bool VerifyAdmissionTerminalControlProtocol(bool retain_kernel = false, int32_t wait_status = 0,
                                           bool query_target = false, bool load_target = false) {
  const std::string id = "aabbccddeeff0202";
  const std::string generation(64, 'd');
  const std::string nonce = TestAdmissionNonce(1300, 13000);
  int sockets[2];
  if (socketpair(AF_UNIX, SOCK_SEQPACKET, 0, sockets) != 0) return false;
  ControlPeer peer{sockets[0], false, false, false, "", ""};
  std::map<std::string, ProfileRegistration> profiles;
  auto parse = [&](const std::string& request, const std::string& expected) {
    if (!ParseControlRecord(request.data(), request.size(), &peer, &profiles)) return false;
    char reply[kMaxControlRecordBytes];
    const ssize_t size = recv(sockets[1], reply, sizeof(reply), MSG_TRUNC);
    return size == static_cast<ssize_t>(expected.size()) &&
        std::string(reply, static_cast<size_t>(size)) == expected;
  };
  const std::string topology = "/|oci-root|/||1|0|0|0;/tmp|workspace|/|tmpfs|0|1|1|0;/haa-runtime|helper|/|tmpfs|0|0|1|0;/haa-site|workspace|/|tmpfs|0|0|1|0";
  const std::string profile = std::string("{\"op\":\"profile\",\"container_id\":\"") + id +
      "\",\"profile\":\"pypi-wheel\",\"expected_topology\":\"" + topology +
      "\",\"session_generation\":\"" + generation + "\"}";
  const std::string profile_ack = std::string("{\"op\":\"ack\",\"ack_op\":\"profile\",\"status\":\"registered\",\"container_id\":\"") + id +
      "\",\"profile\":\"pypi-wheel\",\"expected_topology\":\"" + topology +
      "\",\"session_generation\":\"" + generation + "\"}";
  const auto request = [&](const char* operation, const std::string& item_nonce) {
    return std::string("{\"op\":\"") + operation + "\",\"container_id\":\"" + id +
        "\",\"session_generation\":\"" + generation + "\",\"mode\":\"--launch\",\"nonce\":\"" + item_nonce + "\"}";
  };
  const auto ack = [&](const char* operation, const char* status, const std::string& item_nonce) {
    return std::string("{\"op\":\"ack\",\"ack_op\":\"") + operation + "\",\"status\":\"" + status +
        "\",\"container_id\":\"" + id + "\",\"session_generation\":\"" + generation +
        "\",\"mode\":\"--launch\",\"nonce\":\"" + item_nonce + "\"}";
  };
  bool ok = parse(profile, profile_ack) &&
      parse(request("arm", nonce), ack("arm", "armed", nonce)) &&
      parse(request("status", nonce), ack("status", "pending", nonce)) &&
      parse(request("complete", nonce), ack("complete", "rejected", nonce));
  if (retain_kernel) ok = ok && parse(request("outcome", nonce), ack("outcome", "pending", nonce));
  if (query_target) ok = ok && parse(request("target", nonce), ack("target", "pending", nonce));
  auto registration = profiles.find(id);
  if (!ok || registration == profiles.end() || registration->second.pending_admissions.size() != 1) {
    close(sockets[0]); close(sockets[1]); return false;
  }

  // Consume through the production SENTRY_EXEC parser.  The terminal control
  // protocol must observe the authoritative transition, never a test-created
  // kConsumed slot.
  gvisor::common::ContextData context;
  context.set_container_id(id);
  context.set_thread_group_id(1300);
  context.set_thread_group_start_time_ns(13000);
  context.set_parent_thread_group_id(0);
  context.set_is_exec_session(true);
  context.set_process_name("haa-boundary");
  gvisor::sentry::ExecveInfo boundary;
  *boundary.mutable_context_data() = context;
  boundary.set_binary_path(kBoundaryHelperPath);
  boundary.set_execfn(kBoundaryHelperPath);
  boundary.add_argv(kBoundaryHelperPath);
  boundary.add_argv(kLaunchMode);
  boundary.add_argv(nonce);
  boundary.add_argv("/bin/true");
  int output[2] = {-1, -1};
  std::string payload;
  std::string parsed_container;
  const char* parse_reason = nullptr;
  ProcessState consumed_state;
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0 || !boundary.SerializeToString(&payload) ||
      !ParseSentryProcessAndClassify(payload.data(), payload.size(), output[0], &parsed_container,
                                     kProfilePyPI, &registration->second, &consumed_state, &parse_reason) ||
      !ExpectRecord(output[1], id.c_str(), "process-exec-expected")) {
    if (output[0] >= 0) close(output[0]);
    if (output[1] >= 0) close(output[1]);
    close(sockets[0]); close(sockets[1]); return false;
  }
  if (query_target) {
    auto& owner = registration->second;
    ok = ok && parse(request("target", nonce), ack("target", "waiting", nonce));
    auto different_mode = owner;
    ok = ok && RetainDirectExecTarget(&different_mode, context, BoundaryMode::kELFHandoff) &&
        !different_mode.pending_admissions.front().target_loaded;
    // Neither a child nor a reused root start can populate the exact slot.
    for (bool child : {false, true}) {
      auto foreign = context;
      foreign.set_thread_group_start_time_ns(13001);
      if (child) { foreign.set_thread_group_id(1301); foreign.set_parent_thread_group_id(1300); }
      auto copied = owner;
      ok = ok && RetainDirectExecTarget(&copied, foreign, BoundaryMode::kLaunch) && !copied.pending_admissions.front().target_loaded;
    }
    if (load_target) {
      auto event = boundary;
      event.set_binary_path(kSetprivPath); event.set_execfn(kSetprivPath); event.clear_argv();
      for (const char* arg : {kSetprivPath, "--reuid=1000", "--regid=1000", "--clear-groups",
           "--inh-caps=-all", "--ambient-caps=-all", "--bounding-set=-all", "--no-new-privs",
           "--", "/bin/sh", "-c", ":"}) event.add_argv(arg);
      auto parse_image = [&](const gvisor::sentry::ExecveInfo& image) {
        payload.clear(); parsed_container.clear(); parse_reason = nullptr;
        return image.SerializeToString(&payload) && ParseSentryProcessAndClassify(
            payload.data(), payload.size(), output[0], &parsed_container, kProfilePyPI,
            &owner, &consumed_state, &parse_reason) &&
            ExpectRecord(output[1], id.c_str(), "process-exec-expected");
      };
      ok = ok && parse_image(event) && !owner.pending_admissions.front().target_loaded;
      event.set_binary_path("/usr/bin/dash"); event.set_execfn("/bin/sh"); event.clear_argv();
      for (const char* arg : {"/bin/sh", "-c", ":"}) event.add_argv(arg);
      ok = ok && parse_image(event) && owner.pending_admissions.front().target_loaded &&
          parse(request("target", nonce), ack("target", "loaded", nonce));
      auto duplicate = owner;
      ok = ok && !RetainDirectExecTarget(&duplicate, context, BoundaryMode::kLaunch);
    }
  }
  close(output[0]); close(output[1]);
  if (retain_kernel) {
    auto& exact = registration->second;
    ok = ok && parse(request("outcome", nonce), ack("outcome", "running", nonce));
    gvisor::sentry::ExitNotifyParentInfo terminal;
    *terminal.mutable_context_data() = context;
    terminal.set_exit_status(wait_status);
    auto parse_exit = [&](const gvisor::sentry::ExitNotifyParentInfo& event, ProcessState* state,
                          ProfileRegistration* owner) {
      std::string bytes, container = id;
      const char* reason = nullptr;
      return event.SerializeToString(&bytes) && ParseSentryExitNotifyParent(
          bytes.data(), bytes.size(), &container, state, &reason, owner);
    };
    auto wrong_start = terminal;
    wrong_start.mutable_context_data()->set_thread_group_start_time_ns(13001);
    auto wrong_container = terminal;
    wrong_container.mutable_context_data()->set_container_id(kFirstID);
    ok = ok && !parse_exit(wrong_start, &consumed_state, &exact) &&
        !parse_exit(wrong_container, &consumed_state, &exact) &&
        !exact.pending_admissions.front().terminal_seen;
    // A real tracked descendant exit cannot complete the admitted root.
    auto child = context;
    child.set_thread_group_id(1301);
    child.set_thread_group_start_time_ns(13010);
    child.set_parent_thread_group_id(1300);
    child.set_is_exec_session(false);
    auto child_exit = terminal;
    *child_exit.mutable_context_data() = child;
    ok = ok && RegisterGroup(&consumed_state, child, ProcessState::Role::kArtifact,
        ProcessState::Provenance::kCloneChild, false, true, CommandPhase::kOther) &&
        parse_exit(child_exit, &consumed_state, &exact) &&
        !exact.pending_admissions.front().terminal_seen;
    for (int32_t invalid : {-1, 65, 127, 128, 257, 65535, 65536}) {
      auto copied_state = consumed_state;
      auto copied_owner = exact;
      auto bad = terminal;
      bad.set_exit_status(invalid);
      ok = ok && !parse_exit(bad, &copied_state, &copied_owner) &&
          !copied_owner.pending_admissions.front().terminal_seen;
    }
    ok = ok && parse_exit(terminal, &consumed_state, &exact) && consumed_state.groups.empty();
    if (query_target) {
      ok = ok && parse(request("target", nonce), ack("target", load_target ? "loaded" : "missed", nonce));
      auto late = exact;
      ok = ok && !RetainDirectExecTarget(&late, context, BoundaryMode::kLaunch);
    }
    const std::string status = "exited:" + std::to_string(wait_status);
    ok = ok && parse(request("outcome", nonce), ack("outcome", status.c_str(), nonce)) &&
        !parse_exit(terminal, &consumed_state, &exact) &&
        parse(request("status", nonce), ack("status", "consumed", nonce)) &&
        parse(request("complete", nonce), ack("complete", "completed", nonce)) &&
        exact.pending_admissions.empty() &&
        parse(request("outcome", nonce), ack("outcome", "rejected", nonce));
    close(sockets[0]); close(sockets[1]);
    return ok;
  }
  ok = parse(request("cancel", nonce), ack("cancel", "consumed", nonce)) &&
      registration->second.pending_admissions.size() == 1 &&
      parse(request("status", nonce), ack("status", "consumed", nonce)) &&
      parse(request("complete", nonce), ack("complete", "completed", nonce)) &&
      registration->second.pending_admissions.empty() &&
      parse(request("complete", nonce), ack("complete", "rejected", nonce));
	// Build the mixed 64-slot capacity through real ARM control requests, then
	// consume half through the production SENTRY_EXEC parser. PENDING and
	// CONSUMED must occupy the same live bound; no test may manufacture either.
	std::vector<std::string> pending_nonces;
	std::vector<std::string> consumed_nonces;
	int mixed_output[2] = {-1, -1};
	ok = ok && socketpair(AF_UNIX, SOCK_DGRAM, 0, mixed_output) == 0;
	ProcessState mixed_state;
	for (int index = 0; ok && index < static_cast<int>(kMaxPendingDirectExecAdmissions); ++index) {
		const std::string item_nonce = TestAdmissionNonce(1400 + index, 14000 + index);
		ok = parse(request("arm", item_nonce), ack("arm", "armed", item_nonce));
		if (!ok) break;
		if (index % 2 == 0) {
			pending_nonces.push_back(item_nonce);
			continue;
		}
		gvisor::sentry::ExecveInfo event = boundary;
		event.mutable_context_data()->set_thread_group_id(1400 + index);
		event.mutable_context_data()->set_thread_group_start_time_ns(14000 + index);
		event.set_argv(2, item_nonce);
		payload.clear(); parsed_container.clear(); parse_reason = nullptr;
		ok = event.SerializeToString(&payload) && ParseSentryProcessAndClassify(
			payload.data(), payload.size(), mixed_output[0], &parsed_container, kProfilePyPI,
			&registration->second, &mixed_state, &parse_reason) &&
			ExpectRecord(mixed_output[1], id.c_str(), "process-exec-expected");
		consumed_nonces.push_back(item_nonce);
	}
	const std::string overflow = TestAdmissionNonce(1500, 15000);
	ok = ok && parse(request("arm", overflow), ack("arm", "rejected", overflow)) &&
		registration->second.pending_admissions.size() == kMaxPendingDirectExecAdmissions;
	for (const std::string& item_nonce : consumed_nonces) {
		ok = ok && parse(request("complete", item_nonce), ack("complete", "completed", item_nonce));
	}
	for (const std::string& item_nonce : pending_nonces) {
		ok = ok && parse(request("cancel", item_nonce), ack("cancel", "cancelled", item_nonce));
	}
	ok = ok && registration->second.pending_admissions.empty() &&
		parse(request("arm", overflow), ack("arm", "armed", overflow)) &&
		parse(request("cancel", overflow), ack("cancel", "cancelled", overflow));
	if (mixed_output[0] >= 0) close(mixed_output[0]);
	if (mixed_output[1] >= 0) close(mixed_output[1]);
	const std::string invalidate = std::string("{\"op\":\"invalidate\",\"container_id\":\"") + id +
		"\",\"session_generation\":\"" + generation + "\"}";
	const std::string invalidate_ack = std::string("{\"op\":\"ack\",\"ack_op\":\"invalidate\",\"status\":\"invalidated\",\"container_id\":\"") +
		id + "\",\"session_generation\":\"" + generation + "\"}";
	ok = ok && parse(invalidate, invalidate_ack) && profiles.empty();
  close(sockets[0]);
  close(sockets[1]);
  return ok;
}

// The new mode uses real control ARM and production SENTRY_EXEC transitions.
// The Artifact never supplies the accepted-target or terminal authority.
bool VerifyPythonExecutionBoundary() {
  const std::string id="aabbccddeeff0304", generation(64,'c'), nonce=TestAdmissionNonce(1940,19400);
  int control_pair[2], output[2];
  if(socketpair(AF_UNIX,SOCK_SEQPACKET,0,control_pair)!=0)return false;
  if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0){close(control_pair[0]);close(control_pair[1]);return false;}
  ControlPeer peer{control_pair[0],false,false,false,"",""};
  std::map<std::string,ProfileRegistration> profiles;
  const std::string topology_record="/|oci-root|/||1|0|0|0;/tmp|workspace|/|tmpfs|0|1|1|0;/haa-runtime|helper|/|9p|0|0|0|0;/haa-site|workspace|/|9p|1|0|0|0";
  auto control=[&](const std::string& request,const char* expected){
    if(!ParseControlRecord(request.data(),request.size(),&peer,&profiles))return false;
    char bytes[kMaxControlRecordBytes];ssize_t count=recv(control_pair[1],bytes,sizeof(bytes),MSG_TRUNC);
    std::map<std::string,std::string> ack;
    return count>0 && count<=static_cast<ssize_t>(sizeof(bytes)) && ParseStrictStringObject(bytes,static_cast<size_t>(count),&ack) && ack["status"]==expected && ack["container_id"]==id && ack["session_generation"]==generation;
  };
  auto request=[&](const char* op,const char* mode,const std::string& token){return std::string("{\"op\":\"")+op+"\",\"container_id\":\""+id+"\",\"session_generation\":\""+generation+"\",\"mode\":\""+mode+"\",\"nonce\":\""+token+"\"}";};
  const std::string profile=std::string("{\"op\":\"profile\",\"container_id\":\"")+id+"\",\"profile\":\"python-execution\",\"expected_topology\":\""+topology_record+"\",\"session_generation\":\""+generation+"\"}";
  bool ok=control(profile,"registered") && control(request("arm",kELFHandoffMode,nonce),"rejected") && control(request("arm",kPythonHandoffMode,nonce),"armed");
  auto owner=profiles.find(id);
  if(!ok || owner==profiles.end()){close(control_pair[0]);close(control_pair[1]);close(output[0]);close(output[1]);return false;}
  TopologyState topology;
  topology.expected={{"/","oci-root","/","",true,false,false,false},{"/tmp","workspace","/","tmpfs",false,true,true,false},{"/haa-runtime","helper","/","9p",false,false,false,false},{"/haa-site","workspace","/","9p",true,false,false,false}};
  topology.anchors={{1,{1,"/","oci-root"}},{2,{2,"/tmp","workspace"}},{3,{3,"/haa-runtime","helper"}},{4,{4,"/haa-site","workspace"}},{5,{5,"/proc","system"}}};
  topology.namespace_id=17;topology.snapshot_seen=true;topology.sealed=true;
  const std::string module="cowsay.__main__", attr="main", command="cowsay", argument="a b; literal";
  const std::vector<std::string> invocation{kPythonExecutionPath,"-I","-S","-B","-c",kPythonExecutionProgram,module,attr,command,argument};
  gvisor::common::ContextData context;
  context.set_container_id(id);context.set_thread_group_id(1940);context.set_thread_group_start_time_ns(19400);context.set_parent_thread_group_id(0);context.set_is_exec_session(true);context.set_process_name("haa-boundary");
  gvisor::sentry::ExecveInfo boundary;
  *boundary.mutable_context_data()=context;boundary.set_binary_path("/usr/bin/dash");boundary.set_execfn(kBoundaryHelperPath);
  for(const auto& arg:std::vector<std::string>{kBoundaryHelperPath,kPythonHandoffMode,nonce})boundary.add_argv(arg);
  for(const auto& arg:invocation)boundary.add_argv(arg);
  auto image=[&](const gvisor::sentry::ExecveInfo& event,ProcessState* state,ProfileRegistration* registration,const TopologyState* mounts,const char* profile_name){
    std::string bytes,container;const char* reason=nullptr;
    if(!event.SerializeToString(&bytes) || !ParseSentryProcessAndClassify(bytes.data(),bytes.size(),output[0],&container,profile_name,registration,state,&reason,mounts))return false;
    return ExpectRecord(output[1],id.c_str(),"process-exec-expected");
  };
  for(const char* failure:{"rw","unsealed","shadow","runtime","mode","program","module","attr","argument","count","command"}) {
    auto event=boundary;auto mounts=topology;auto registration=owner->second;ProcessState failed;
    if(strcmp(failure,"rw")==0)mounts.expected.back().read_only=false;
    if(strcmp(failure,"unsealed")==0)mounts.sealed=false;
    if(strcmp(failure,"shadow")==0)mounts.anchors[6]={6,"/haa-site/child","workspace"};
    if(strcmp(failure,"runtime")==0)event.set_argv(3,"/usr/bin/python");
    if(strcmp(failure,"mode")==0)event.set_argv(1,kNodeHandoffMode);
    if(strcmp(failure,"program")==0)event.set_argv(8,"print('forged')");
    if(strcmp(failure,"module")==0)event.set_argv(9,"cowsay;forged");
    if(strcmp(failure,"attr")==0)event.set_argv(10,"main()");
    if(strcmp(failure,"argument")==0)event.set_argv(12,std::string(32769,'x'));
    if(strcmp(failure,"count")==0)for(int i=0;i<129;++i)event.add_argv("x");
    if(strcmp(failure,"command")==0)event.set_argv(11,"/usr/bin/python");
    ok=ok && !image(event,&failed,&registration,&mounts,kProfilePythonExecution) && !registration.pending_admissions.front().target_loaded;
  }
  ProcessState state;
  ok=ok && image(boundary,&state,&owner->second,&topology,kProfilePythonExecution) &&
      state.groups.at(1940).role==ProcessState::Role::kArtifact && !state.groups.at(1940).root_eligible &&
      state.groups.at(1940).root_consumed && !state.groups.at(1940).trusted_control_network_active && state.expected_groups.empty() &&
      control(request("target",kPythonHandoffMode,nonce),"waiting");
  auto demotion=boundary;demotion.set_binary_path(kSetprivPath);demotion.set_execfn(kSetprivPath);demotion.clear_argv();
  for(const auto& arg:std::vector<std::string>{kSetprivPath,"--reuid=1000","--regid=1000","--clear-groups","--inh-caps=-all","--ambient-caps=-all","--bounding-set=-all","--no-new-privs","--"})demotion.add_argv(arg);
  for(const auto& arg:invocation)demotion.add_argv(arg);
  auto wrong_demotion=demotion;wrong_demotion.set_argv(15,"substituted.module");
  {auto candidate=state;auto registration=owner->second;ok=ok && !image(wrong_demotion,&candidate,&registration,&topology,kProfilePythonExecution);}
  ok=ok && image(demotion,&state,&owner->second,&topology,kProfilePythonExecution) && state.groups.at(1940).handoff_target_pending && !state.groups.at(1940).demotion_pending;
  auto target=boundary;target.set_binary_path(kPythonExecutionPath);target.set_execfn(kPythonExecutionPath);target.clear_argv();
  for(const auto& arg:invocation)target.add_argv(arg);
  for(const char* failure:{"argv","execfn","module","image"}){
    auto event=target;auto candidate=state;auto registration=owner->second;
    if(strcmp(failure,"argv")==0)event.set_argv(9,"substituted");
    if(strcmp(failure,"execfn")==0)event.set_execfn("python");
    if(strcmp(failure,"module")==0)event.set_argv(6,module+".changed");
    if(strcmp(failure,"image")==0)event.set_binary_path("/tmp/python");
    ok=ok && !image(event,&candidate,&registration,&topology,kProfilePythonExecution) && !registration.pending_admissions.front().target_loaded;
  }
  ok=ok && image(target,&state,&owner->second,&topology,kProfilePythonExecution) &&
      state.groups.at(1940).python_execution_active && state.groups.at(1940).python_execution_target.empty() &&
      state.groups.at(1940).role==ProcessState::Role::kArtifact && !IsFilesystemControlGroup(context,state) && state.expected_groups.empty() &&
      control(request("target",kPythonHandoffMode,nonce),"loaded");
  const MountAnchor root{1,"/","oci-root"},proc{5,"/proc","system"},site{4,"/haa-site","workspace"};
  gvisor::syscall::Open event;*event.mutable_context_data()=context;event.set_flags(557056);
  for(const char* path:{"/usr/local/lib/python3.14/encodings/__init__.py","/usr/local/lib/libpython3.14.so.1.0","/etc/ld.so.cache"}) {
    event.set_pathname(path);
    ok=ok && ClassifyFilesystemOpen(event,state,kProfilePythonExecution,&root)==FilesystemClass::kRuntimeRoot;
  }
  for(const char* failure:{"write","site","pip","inactive","clone","role","start","image","profile","mount"}) {
    auto candidate=state;auto caller=context;auto mount=root;const char* profile=kProfilePythonExecution;std::string path="/usr/local/lib/python3.14/encodings/__init__.py";uint64_t flags=557056;
    if(strcmp(failure,"write")==0)flags=557058;
    if(strcmp(failure,"site")==0)path="/usr/local/lib/python3.14/site-packages/evil.py";
    if(strcmp(failure,"pip")==0)path="/usr/local/bin/pip";
    if(strcmp(failure,"inactive")==0)candidate.groups.at(1940).python_execution_active=false;
    if(strcmp(failure,"clone")==0)candidate.groups.at(1940).provenance=ProcessState::Provenance::kCloneChild;
    if(strcmp(failure,"role")==0)candidate.groups.at(1940).role=ProcessState::Role::kControl;
    if(strcmp(failure,"start")==0)caller.set_thread_group_start_time_ns(19401);
    if(strcmp(failure,"image")==0){candidate.groups.at(1940).current_image_class=ProcessClass::kShell;caller.set_process_name("python3.14");}
    if(strcmp(failure,"profile")==0)profile=kProfilePyPI;
    if(strcmp(failure,"mount")==0)mount.mount_class="workspace";
    ok=ok && !IsPinnedPythonExecutionRead(caller,candidate,path,flags,profile,&mount);
  }
  event.set_pathname("/haa-site/cowsay/__init__.py");
  ok=ok && ClassifyFilesystemOpen(event,state,kProfilePythonExecution,&site)==FilesystemClass::kWorkspace;
  event.set_flags(557058);
  ok=ok && ClassifyFilesystemOpen(event,state,kProfilePythonExecution,&site)==FilesystemClass::kOutside;
  event.set_flags(557056);event.set_pathname("/proc/1940/maps");
  ok=ok && ClassifyFilesystemOpen(event,state,kProfilePythonExecution,&proc)==FilesystemClass::kRuntimeRoot;
  event.set_pathname("/proc/sys/vm/overcommit_memory");event.set_flags(32768);
  ok=ok && ClassifyFilesystemOpen(event,state,kProfilePythonExecution,&proc)==FilesystemClass::kRuntimeRoot;
  for(uint64_t flags:{0ULL,557056ULL,32769ULL,32770ULL,33280ULL})
    ok=ok && !IsPinnedPythonExecutionRead(context,state,event.pathname(),flags,kProfilePythonExecution,&proc);
  for(const char* path:{"/proc/sys/vm/overcommit_ratio","/proc/sys/vm/overcommit_memory/","/proc/sys/vm/overcommit_memory.evil","/proc/sys/vm/../vm/overcommit_memory"})
    ok=ok && !IsPinnedPythonExecutionRead(context,state,path,32768,kProfilePythonExecution,&proc);
  for(const char* failure:{"mount","class","role","clone","inactive","pending","start","image"}) {
    auto candidate=state;auto caller=context;auto mount=proc;
    if(strcmp(failure,"mount")==0)mount.mountpoint="/sys";
    if(strcmp(failure,"class")==0)mount.mount_class="workspace";
    if(strcmp(failure,"role")==0)candidate.groups.at(1940).role=ProcessState::Role::kControl;
    if(strcmp(failure,"clone")==0)candidate.groups.at(1940).provenance=ProcessState::Provenance::kCloneChild;
    if(strcmp(failure,"inactive")==0)candidate.groups.at(1940).python_execution_active=false;
    if(strcmp(failure,"pending")==0)candidate.groups.at(1940).handoff_target_pending=true;
    if(strcmp(failure,"start")==0)caller.set_thread_group_start_time_ns(19401);
    if(strcmp(failure,"image")==0){candidate.groups.at(1940).current_image_class=ProcessClass::kShell;caller.set_process_name("python");}
    ok=ok && !IsPinnedPythonExecutionRead(caller,candidate,event.pathname(),32768,kProfilePythonExecution,&mount);
  }
  event.set_flags(557056);
  event.set_pathname("/proc/1/maps");
  ok=ok && !IsPinnedPythonExecutionRead(context,state,event.pathname(),event.flags(),kProfilePythonExecution,&proc);
  auto spoof=context;spoof.set_process_name("haa-boundary");event.set_pathname(kBoundaryHelperPath);*event.mutable_context_data()=spoof;
  ok=ok && ClassifyFilesystemOpen(event,state,kProfilePythonExecution,&root)!=FilesystemClass::kHelperOnly;
  gvisor::sentry::CloneInfo clone;*clone.mutable_context_data()=context;clone.set_created_thread_group_id(1941);clone.set_created_thread_start_time_ns(19410);
  std::string bytes,container;const char* reason=nullptr;
  ok=ok && clone.SerializeToString(&bytes) && ParseSentryClone(bytes.data(),bytes.size(),output[0],&container,&state,&reason) &&
      !state.groups.at(1941).python_execution_active && state.groups.at(1941).python_execution_target.empty();
  auto later=target;later.set_argv(9,"later");auto changed=state;auto registration=owner->second;
  ok=ok && !image(later,&changed,&registration,&topology,kProfilePythonExecution) && !changed.groups.at(1940).python_execution_active;
  gvisor::sentry::ExitNotifyParentInfo terminal;*terminal.mutable_context_data()=context;terminal.set_exit_status(37<<8);bytes.clear();container=id;reason=nullptr;
  ok=ok && terminal.SerializeToString(&bytes) && ParseSentryExitNotifyParent(bytes.data(),bytes.size(),&container,&state,&reason,&owner->second) &&
      control(request("outcome",kPythonHandoffMode,nonce),"exited:9472") && control(request("complete",kPythonHandoffMode,nonce),"completed");
  close(control_pair[0]);close(control_pair[1]);close(output[0]);close(output[1]);return ok;
}

bool VerifyNodeExecutionBoundary() {
  const std::string id="aabbccddeeff0303", generation(64,'c'), nonce=TestAdmissionNonce(1930,19300);
  int control_pair[2], output[2];
  if(socketpair(AF_UNIX,SOCK_SEQPACKET,0,control_pair)!=0)return false;
  if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0){close(control_pair[0]);close(control_pair[1]);return false;}
  ControlPeer peer{control_pair[0],false,false,false,"",""};
  std::map<std::string,ProfileRegistration> profiles;
  const std::string topology_record="/|oci-root|/||1|0|0|0;/tmp|workspace|/|tmpfs|0|1|1|0;/haa-runtime|helper|/|9p|0|0|0|0;/tmp/haa-node-execution|workspace|/tmp|9p|1|0|0|0";
  auto control=[&](const std::string& request,const char* expected){
    if(!ParseControlRecord(request.data(),request.size(),&peer,&profiles))return false;
    char bytes[kMaxControlRecordBytes];ssize_t count=recv(control_pair[1],bytes,sizeof(bytes),MSG_TRUNC);
    std::map<std::string,std::string> ack;
    return count>0 && count<=static_cast<ssize_t>(sizeof(bytes)) && ParseStrictStringObject(bytes,static_cast<size_t>(count),&ack) && ack["status"]==expected && ack["container_id"]==id && ack["session_generation"]==generation;
  };
  auto request=[&](const char* op,const char* mode,const std::string& token){return std::string("{\"op\":\"")+op+"\",\"container_id\":\""+id+"\",\"session_generation\":\""+generation+"\",\"mode\":\""+mode+"\",\"nonce\":\""+token+"\"}";};
  const std::string profile=std::string("{\"op\":\"profile\",\"container_id\":\"")+id+"\",\"profile\":\"node-execution\",\"expected_topology\":\""+topology_record+"\",\"session_generation\":\""+generation+"\"}";
  bool ok=control(profile,"registered") && control(request("arm",kELFHandoffMode,nonce),"rejected") && control(request("arm",kNodeHandoffMode,nonce),"armed");
  auto owner=profiles.find(id);
  if(!ok || owner==profiles.end()){close(control_pair[0]);close(control_pair[1]);close(output[0]);close(output[1]);return false;}
  TopologyState topology;
  topology.expected={{"/","oci-root","/","",true,false,false,false},{"/tmp","workspace","/","tmpfs",false,true,true,false},{"/haa-runtime","helper","/","9p",false,false,false,false},{"/tmp/haa-node-execution","workspace","/tmp","9p",true,false,false,false}};
  topology.anchors={{1,{1,"/","oci-root"}},{2,{2,"/tmp","workspace"}},{3,{3,"/haa-runtime","helper"}},{4,{4,"/tmp/haa-node-execution","workspace"}},{5,{5,"/proc","system"}}};
  topology.namespace_id=17;topology.snapshot_seen=true;topology.sealed=true;
  const std::string entry="/tmp/haa-node-execution/project/node_modules/fixture/bin/cli.js";
  gvisor::common::ContextData context;
  context.set_container_id(id);context.set_thread_group_id(1930);context.set_thread_group_start_time_ns(19300);context.set_parent_thread_group_id(0);context.set_is_exec_session(true);context.set_process_name("haa-boundary");
  gvisor::sentry::ExecveInfo boundary;
  *boundary.mutable_context_data()=context;boundary.set_binary_path("/usr/bin/dash");boundary.set_execfn(kBoundaryHelperPath);
  for(const auto& arg:std::vector<std::string>{kBoundaryHelperPath,kNodeHandoffMode,nonce,kNodePath,"--",entry,"a b; literal"})boundary.add_argv(arg);
  auto image=[&](const gvisor::sentry::ExecveInfo& event,ProcessState* state,ProfileRegistration* registration,const TopologyState* mounts,const char* profile_name){
    std::string bytes,container;const char* reason=nullptr;
    if(!event.SerializeToString(&bytes) || !ParseSentryProcessAndClassify(bytes.data(),bytes.size(),output[0],&container,profile_name,registration,state,&reason,mounts))return false;
    return ExpectRecord(output[1],id.c_str(),"process-exec-expected");
  };
  for(const char* failure:{"rw","unsealed","shadow","entry","runtime","profile","mode","argv-limit"}){
    auto event=boundary;auto mounts=topology;auto registration=owner->second;ProcessState state;const char* profile_name=kProfileNodeExecution;
    if(strcmp(failure,"rw")==0)mounts.expected.back().read_only=false;
    if(strcmp(failure,"unsealed")==0)mounts.sealed=false;
    if(strcmp(failure,"shadow")==0)mounts.anchors[6]={6,"/tmp/haa-node-execution/project/node_modules","workspace"};
    if(strcmp(failure,"entry")==0)event.set_argv(5,"/tmp/elsewhere.js");
    if(strcmp(failure,"runtime")==0)event.set_argv(3,"/usr/bin/node");
    if(strcmp(failure,"profile")==0){profile_name=kProfileNPM;registration.profile=kProfileNPM;}
    if(strcmp(failure,"mode")==0)event.set_argv(1,kPythonHandoffMode);
    if(strcmp(failure,"argv-limit")==0)event.set_argv(6,std::string(32769,'x'));
    ok=ok && !image(event,&state,&registration,&mounts,profile_name) && !registration.pending_admissions.front().target_loaded;
  }
  ProcessState state;
  ok=ok && image(boundary,&state,&owner->second,&topology,kProfileNodeExecution) &&
      state.groups.at(1930).role==ProcessState::Role::kArtifact && !state.groups.at(1930).root_eligible &&
      state.groups.at(1930).root_consumed && !state.groups.at(1930).trusted_control_network_active && state.expected_groups.empty() &&
      control(request("target",kNodeHandoffMode,nonce),"waiting");
  auto demotion=boundary;demotion.set_binary_path(kSetprivPath);demotion.set_execfn(kSetprivPath);demotion.clear_argv();
  for(const auto& arg:std::vector<std::string>{kSetprivPath,"--reuid=1000","--regid=1000","--clear-groups","--inh-caps=-all","--ambient-caps=-all","--bounding-set=-all","--no-new-privs","--",kNodePath,"--",entry,"a b; literal"})demotion.add_argv(arg);
  ok=ok && image(demotion,&state,&owner->second,&topology,kProfileNodeExecution) && state.groups.at(1930).node_execution_demoting;
  auto target=boundary;target.set_binary_path(kNodePath);target.set_execfn(kNodePath);target.clear_argv();
  for(const auto& arg:std::vector<std::string>{kNodePath,"--",entry,"a b; literal"})target.add_argv(arg);
  for(const char* failure:{"argv","execfn","entry","image"}){
    auto event=target;auto candidate=state;auto registration=owner->second;
    if(strcmp(failure,"argv")==0)event.set_argv(3,"substituted");
    if(strcmp(failure,"execfn")==0)event.set_execfn("node");
    if(strcmp(failure,"entry")==0)event.set_argv(2,entry+".changed");
    if(strcmp(failure,"image")==0)event.set_binary_path("/tmp/node");
    ok=ok && !image(event,&candidate,&registration,&topology,kProfileNodeExecution) && !registration.pending_admissions.front().target_loaded;
  }
  ok=ok && image(target,&state,&owner->second,&topology,kProfileNodeExecution) &&
      state.groups.at(1930).node_execution_active && state.groups.at(1930).node_execution_target.empty() &&
      state.groups.at(1930).role==ProcessState::Role::kArtifact && !IsFilesystemControlGroup(context,state) && state.expected_groups.empty() &&
      control(request("target",kNodeHandoffMode,nonce),"loaded");
  gvisor::syscall::Open open;
  *open.mutable_context_data()=context;open.set_pathname("/proc/1930/maps");open.set_flags(557056);
  const MountAnchor proc{5,"/proc","system"};
  ok=ok && ClassifyFilesystemOpen(open,state,kProfileNodeExecution,&proc)==FilesystemClass::kRuntimeRoot;
  open.set_pathname("/proc/1930/cgroup");
  ok=ok && ClassifyFilesystemOpen(open,state,kProfileNodeExecution,&proc)==FilesystemClass::kRuntimeRoot;
  for(const char* path:{"/proc/self/cgroup","/proc/1/cgroup","/proc/1931/cgroup","/proc/1930/cgroup/","/proc/1930/cgroup.evil","/proc/1930/../1930/cgroup","/proc/1930/environ"}){
    auto denied=open;denied.set_pathname(path);
    ok=ok && !IsPinnedNodeExecutionRead(denied.context_data(),state,denied.pathname(),denied.flags(),kProfileNodeExecution,&proc);
  }
  for(uint64_t flags:{0ULL,32768ULL,557057ULL,557058ULL,557568ULL})
    ok=ok && !IsPinnedNodeExecutionRead(open.context_data(),state,open.pathname(),flags,kProfileNodeExecution,&proc);
  for(const char* failure:{"write","foreign-map","inactive","control-role","clone","foreign-start","other-profile"}){
    auto event=open;auto candidate=state;const char* profile_name=kProfileNodeExecution;
    if(strcmp(failure,"write")==0)event.set_flags(557058);
    if(strcmp(failure,"foreign-map")==0)event.set_pathname("/proc/1/maps");
    if(strcmp(failure,"inactive")==0)candidate.groups.at(1930).node_execution_active=false;
    if(strcmp(failure,"control-role")==0)candidate.groups.at(1930).role=ProcessState::Role::kControl;
    if(strcmp(failure,"clone")==0)candidate.groups.at(1930).provenance=ProcessState::Provenance::kCloneChild;
    if(strcmp(failure,"foreign-start")==0)event.mutable_context_data()->set_thread_group_start_time_ns(19301);
    if(strcmp(failure,"other-profile")==0)profile_name=kProfileNPM;
    ok=ok && !IsPinnedNodeExecutionRead(event.context_data(),candidate,event.pathname(),event.flags(),profile_name,&proc);
  }
  const MountAnchor memory{6,"/sys/fs/cgroup/memory","system"};
  for(const char* path:{"/sys/fs/cgroup/memory/memory.soft_limit_in_bytes","/sys/fs/cgroup/memory/memory.limit_in_bytes"}){
    auto event=open;event.set_pathname(path);
    ok=ok && ClassifyFilesystemOpen(event,state,kProfileNodeExecution,&memory)==FilesystemClass::kRuntimeRoot;
    for(uint64_t flags:{0ULL,32768ULL,557057ULL,557058ULL,557568ULL})
      ok=ok && !IsPinnedNodeExecutionRead(context,state,path,flags,kProfileNodeExecution,&memory);
    for(const char* failure:{"inactive","clone","role","start","profile","mount","mount-class"}){
      auto candidate=state;auto caller=context;auto mount=memory;const char* profile_name=kProfileNodeExecution;
      if(strcmp(failure,"inactive")==0)candidate.groups.at(1930).node_execution_active=false;
      if(strcmp(failure,"clone")==0)candidate.groups.at(1930).provenance=ProcessState::Provenance::kCloneChild;
      if(strcmp(failure,"role")==0)candidate.groups.at(1930).role=ProcessState::Role::kControl;
      if(strcmp(failure,"start")==0)caller.set_thread_group_start_time_ns(19301);
      if(strcmp(failure,"profile")==0)profile_name=kProfileNPM;
      if(strcmp(failure,"mount")==0)mount.mountpoint="/sys";
      if(strcmp(failure,"mount-class")==0)mount.mount_class="workspace";
      ok=ok && !IsPinnedNodeExecutionRead(caller,candidate,path,557056,profile_name,&mount);
    }
  }
  for(const char* path:{"/sys/fs/cgroup/memory/foreign/memory.limit_in_bytes","/sys/fs/cgroup/memory/memory.limit_in_bytes.evil","/sys/fs/cgroup/memory/memory.soft_limit_in_bytes/","/sys/fs/cgroup/memory/../memory/memory.limit_in_bytes","/sys/fs/cgroup/memory/cgroup.procs"})
    ok=ok && !IsPinnedNodeExecutionRead(context,state,path,557056,kProfileNodeExecution,&memory);
  const MountAnchor sys{7,"/sys","system"};
  for(const char* path:{"/proc/meminfo","/proc/version_signature","/proc/sys/vm/overcommit_memory","/sys/devices/system/cpu/online"}){
    auto event=open;event.set_pathname(path);const MountAnchor& mount=HasPrefix(path,"/proc/")?proc:sys;
    ok=ok && ClassifyFilesystemOpen(event,state,kProfileNodeExecution,&mount)==FilesystemClass::kRuntimeRoot;
    for(uint64_t flags:{0ULL,32768ULL,557057ULL,557058ULL,557568ULL})
      ok=ok && !IsPinnedNodeExecutionRead(context,state,path,flags,kProfileNodeExecution,&mount);
    for(const std::string& changed:{std::string(path)+".evil",std::string(path)+"/",std::string(path)+"/../meminfo"})
      ok=ok && !IsPinnedNodeExecutionRead(context,state,changed,557056,kProfileNodeExecution,&mount);
    for(const char* failure:{"inactive","clone","role","start","profile","mount","mount-class"}){
      auto candidate=state;auto caller=context;auto changed=mount;const char* profile_name=kProfileNodeExecution;
      if(strcmp(failure,"inactive")==0)candidate.groups.at(1930).node_execution_active=false;
      if(strcmp(failure,"clone")==0)candidate.groups.at(1930).provenance=ProcessState::Provenance::kCloneChild;
      if(strcmp(failure,"role")==0)candidate.groups.at(1930).role=ProcessState::Role::kControl;
      if(strcmp(failure,"start")==0)caller.set_thread_group_start_time_ns(19301);
      if(strcmp(failure,"profile")==0)profile_name=kProfileNPM;
      if(strcmp(failure,"mount")==0)changed.mountpoint="/";
      if(strcmp(failure,"mount-class")==0)changed.mount_class="workspace";
      ok=ok && !IsPinnedNodeExecutionRead(caller,candidate,path,557056,profile_name,&changed);
    }
  }
  for(const char* path:{"/proc/cpuinfo","/proc/sys/vm/overcommit_ratio","/proc/sys/kernel/hostname","/sys/devices/system/cpu/possible","/proc/1930/environ"})
    ok=ok && !IsPinnedNodeExecutionRead(context,state,path,557056,kProfileNodeExecution,HasPrefix(path,"/proc/")?&proc:&sys);
  const MountAnchor dev{8,"/dev","system"};
  auto null_open=open;null_open.set_pathname("/dev/null");
  ok=ok && ClassifyFilesystemOpen(null_open,state,kProfileNodeExecution,&dev)==FilesystemClass::kRuntimeRoot;
  for(uint64_t flags:{0ULL,32768ULL,557057ULL,557058ULL,557568ULL})
    ok=ok && !IsPinnedNodeExecutionRead(context,state,"/dev/null",flags,kProfileNodeExecution,&dev);
  for(const char* path:{"/dev/random","/dev/urandom","/dev/zero","/dev/null/","/dev/null.evil","/dev/../dev/null","/tmp/null"})
    ok=ok && !IsPinnedNodeExecutionRead(context,state,path,557056,kProfileNodeExecution,&dev);
  for(const char* failure:{"inactive","clone","role","start","profile","mount","mount-class"}){
    auto candidate=state;auto caller=context;auto mount=dev;const char* profile_name=kProfileNodeExecution;
    if(strcmp(failure,"inactive")==0)candidate.groups.at(1930).node_execution_active=false;
    if(strcmp(failure,"clone")==0)candidate.groups.at(1930).provenance=ProcessState::Provenance::kCloneChild;
    if(strcmp(failure,"role")==0)candidate.groups.at(1930).role=ProcessState::Role::kControl;
    if(strcmp(failure,"start")==0)caller.set_thread_group_start_time_ns(19301);
    if(strcmp(failure,"profile")==0)profile_name=kProfileNPM;
    if(strcmp(failure,"mount")==0)mount.mountpoint="/dev/null";
    if(strcmp(failure,"mount-class")==0)mount.mount_class="workspace";
    ok=ok && !IsPinnedNodeExecutionRead(caller,candidate,"/dev/null",557056,profile_name,&mount);
  }
  // A real production clone never inherits the new active runtime authority.
  gvisor::sentry::CloneInfo clone;
  *clone.mutable_context_data()=context;clone.set_created_thread_group_id(1931);clone.set_created_thread_start_time_ns(19310);
  std::string bytes,container;const char* reason=nullptr;
  ok=ok && clone.SerializeToString(&bytes) && ParseSentryClone(bytes.data(),bytes.size(),output[0],&container,&state,&reason) &&
      !state.groups.at(1931).node_execution_active && !state.groups.at(1931).node_execution_demoting && state.groups.at(1931).node_execution_target.empty() &&
      state.groups.at(1931).role==ProcessState::Role::kArtifact;
  // A later successful exec cannot retain active Node runtime reads or turn
  // the historical target bit into fresh execution authority.
  auto later=target;later.set_argv(3,"later");auto changed=state;auto registration=owner->second;
  ok=ok && !image(later,&changed,&registration,&topology,kProfileNodeExecution) && !changed.groups.at(1930).node_execution_active;
  gvisor::sentry::ExitNotifyParentInfo terminal;
  *terminal.mutable_context_data()=context;terminal.set_exit_status(37<<8);bytes.clear();container=id;reason=nullptr;
  ok=ok && terminal.SerializeToString(&bytes) && ParseSentryExitNotifyParent(bytes.data(),bytes.size(),&container,&state,&reason,&owner->second) &&
      control(request("outcome",kNodeHandoffMode,nonce),"exited:9472") && control(request("complete",kNodeHandoffMode,nonce),"completed");
  close(control_pair[0]);close(control_pair[1]);close(output[0]);close(output[1]);
  return ok;
}

bool VerifyKernelTerminalOutcomes() {
  for (int32_t wait_status : {0, 37 << 8, 125 << 8, 126 << 8, 127 << 8, 255 << 8, 9, 15 | 128, 64}) {
    if (!VerifyAdmissionTerminalControlProtocol(true, wait_status)) return false;
  }
  return true;
}

bool VerifySessionAwareControl(const std::string& control) {
  const char id[] = "aabbccddeeff0101";
  const std::string first = TestSessionGeneration(id);
  const std::string second(64, 'c');
  const std::string nonce = TestAdmissionNonce(1100, 11000);
  const bool registered_first = RegisterProfileWithGeneration(control, id, kProfilePyPI, first);
  const bool armed_first = registered_first && ControlAdmissionWithGeneration(control, id, kLaunchMode, first, nonce, "arm");
  const bool invalidated = armed_first && InvalidateProfile(control, id, first);
  const bool old_cancel_rejected = invalidated && !ControlAdmissionWithGeneration(control, id, kLaunchMode, first, nonce, "cancel");
  const bool old_arm_rejected = old_cancel_rejected && !ControlAdmissionWithGeneration(control, id, kLaunchMode, first, nonce, "arm");
  const bool registered_second = old_arm_rejected && RegisterProfileWithGeneration(control, id, kProfilePyPI, second);
  const bool stale_rejected = registered_second && !ControlAdmissionWithGeneration(control, id, kLaunchMode, first, nonce, "arm");
  const bool armed_second = stale_rejected && ControlAdmissionWithGeneration(control, id, kLaunchMode, second, nonce, "arm");
  const bool cancelled_second = armed_second && ControlAdmissionWithGeneration(control, id, kLaunchMode, second, nonce, "cancel");
  return cancelled_second;
}

std::string TestTopologyForProfile(const char* profile) {
  std::string topology = "/|oci-root|/||1|0|0|0;/tmp|workspace|/|tmpfs|0|1|1|0;/haa-runtime|helper|/|tmpfs|0|0|1|0";
  if (strcmp(profile, kProfilePyPI) == 0 || strcmp(profile, kProfilePyTorchCPU) == 0 || strcmp(profile, kProfilePyTorchCU126) == 0) {
    topology += ";/haa-site|workspace|/|tmpfs|0|0|1|0";
  } else if (strcmp(profile, kProfileGitHub) == 0 || strcmp(profile, "terraform-provider") == 0) {
    topology += ";/work|workspace|/|tmpfs|0|0|1|0";
  }
  return topology;
}

bool ProfileOnFD(int fd, const char* container_id, const char* profile,
                 const std::string& generation, const char* status = "registered") {
  const std::string topology = TestTopologyForProfile(profile);
  const std::string request = std::string("{\"op\":\"profile\",\"container_id\":\"") + container_id +
      "\",\"profile\":\"" + profile + "\",\"expected_topology\":\"" + topology +
      "\",\"session_generation\":\"" + generation + "\"}";
  const std::string expected = std::string("{\"op\":\"ack\",\"ack_op\":\"profile\",\"status\":\"") + status +
      "\",\"container_id\":\"" + container_id + "\",\"profile\":\"" + profile +
      "\",\"expected_topology\":\"" + topology + "\",\"session_generation\":\"" + generation + "\"}";
  return ControlExchange(fd, request, expected);
}

bool VerifyExactControlPeerBinding(int output, const std::string& remote, const std::string& control) {
  const std::string generation_a(64, 'a');
  const std::string generation_b(64, 'b');
  const std::string fresh_generation(63, 'c');
  const std::string nonce_pending = TestAdmissionNonce(1200, 12000);
  const std::string nonce_consumed = TestAdmissionNonce(1202, 12002);
  const int a = ConnectControl(control);
  const int wrong_peer = ConnectControl(control);
  if (a < 0 || wrong_peer < 0) { if (a >= 0) close(a); if (wrong_peer >= 0) close(wrong_peer); return false; }
  bool ok = ProfileOnFD(a, kSixteenthID, kProfilePyPI, generation_a) &&
      // A second PROFILE and every authority operation before PROFILE are rejected.
      ProfileOnFD(a, kSixteenthID, kProfilePyPI, generation_a, "rejected");
  if (!ok) { close(a); close(wrong_peer); return false; }

  // Arm nonce_pending on peer A -> REAL PENDING slot exists.
  ok = ok && ControlAdmissionOnFD(a, kSixteenthID, kLaunchMode, generation_a, nonce_pending, "arm", "armed");
  if (!ok) { close(a); close(wrong_peer); return false; }

  // Wrong peer operations against A's existing pending slot are rejected.
  ok = ok && ControlAdmissionOnFD(wrong_peer, kSixteenthID, kLaunchMode, generation_a, nonce_pending, "arm", "rejected") &&
      ControlAdmissionOnFD(wrong_peer, kSixteenthID, kLaunchMode, generation_a, nonce_pending, "status", "rejected") &&
      ControlAdmissionOnFD(wrong_peer, kSixteenthID, kLaunchMode, generation_a, nonce_pending, "cancel", "rejected") &&
      ControlAdmissionOnFD(wrong_peer, kSixteenthID, kLaunchMode, generation_a, nonce_pending, "complete", "rejected");
  if (!ok) { close(a); close(wrong_peer); return false; }
  const std::string invalidate = std::string("{\"op\":\"invalidate\",\"container_id\":\"") + kSixteenthID +
      "\",\"session_generation\":\"" + generation_a + "\"}";
  const std::string rejected_invalidate = std::string("{\"op\":\"ack\",\"ack_op\":\"invalidate\",\"status\":\"rejected\",\"container_id\":\"") +
      kSixteenthID + "\",\"session_generation\":\"" + generation_a + "\"}";
  ok = ok && ControlExchange(wrong_peer, invalidate, rejected_invalidate) &&
      ControlAdmissionOnFD(a, kSixteenthID, kLaunchMode, generation_a, nonce_pending, "status", "pending") &&
      ControlAdmissionOnFD(a, kSixteenthID, kLaunchMode, generation_a, nonce_pending, "complete", "rejected") &&
      ControlAdmissionOnFD(a, kSixteenthID, kLaunchMode, generation_a, nonce_pending, "status", "pending");
  if (!ok) { close(a); close(wrong_peer); return false; }

  // Arm nonce_consumed on peer A, then transition it to CONSUMED via real Sentry exec event.
  ok = ok && ControlAdmissionOnFD(a, kSixteenthID, kLaunchMode, generation_a, nonce_consumed, "arm", "armed");
  if (!ok) { close(a); close(wrong_peer); return false; }
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) {
    if (client >= 0) close(client);
    close(a); close(wrong_peer);
    return false;
  }
  DrainDatagram(output);
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kSixteenthID);
  start.mutable_context_data()->set_thread_group_id(1199);
  start.mutable_context_data()->set_thread_group_start_time_ns(11990);
  ok = ok && SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) &&
      ExpectRecord(output, kSixteenthID, "container-start") &&
      SendRealDirectExecTransition(output, client, kSixteenthID, kLaunchMode, nonce_consumed, 1202, 12002);
  if (!ok) { close(client); close(a); close(wrong_peer); return false; }

  // Wrong peer operations against A's existing CONSUMED slot are rejected.
  ok = ok && ControlAdmissionOnFD(wrong_peer, kSixteenthID, kLaunchMode, generation_a, nonce_consumed, "status", "rejected") &&
      ControlAdmissionOnFD(wrong_peer, kSixteenthID, kLaunchMode, generation_a, nonce_consumed, "cancel", "rejected") &&
      ControlAdmissionOnFD(wrong_peer, kSixteenthID, kLaunchMode, generation_a, nonce_consumed, "complete", "rejected");
  if (!ok) { close(client); close(a); close(wrong_peer); return false; }

  // Verify peer A can inspect, complete, and clean its own slots.
  ok = ok && ControlAdmissionOnFD(a, kSixteenthID, kLaunchMode, generation_a, nonce_consumed, "status", "consumed") &&
      ControlAdmissionOnFD(a, kSixteenthID, kLaunchMode, generation_a, nonce_consumed, "complete", "completed") &&
      ControlAdmissionOnFD(a, kSixteenthID, kLaunchMode, generation_a, nonce_pending, "cancel", "cancelled");
  if (!ok) { close(client); close(a); close(wrong_peer); return false; }
  close(client);
  ok = ok && ExpectRecordExact(output, kSixteenthID, "stream-end");
  if (!ok) { close(a); close(wrong_peer); return false; }

  const int b = ConnectControl(control);
  if (b < 0) { close(a); close(wrong_peer); return false; }
  ok = ok && ProfileOnFD(b, kSeventeenthID, kProfilePyPI, generation_b) &&
      ControlAdmissionOnFD(b, kSeventeenthID, kLaunchMode, generation_b, nonce_pending, "arm", "armed");
  if (!ok) { close(a); close(b); close(wrong_peer); return false; }
  close(a);
  // The next request services A's HUP and proves B remains independently live.
  ok = ok && ControlAdmissionOnFD(b, kSeventeenthID, kLaunchMode, generation_b,
                                  TestAdmissionNonce(1201, 12001), "arm", "armed");
  if (!ok) { close(b); close(wrong_peer); return false; }

  // HUP erased A's profile and its live pending slot. A replacement controller
  // uses a fresh generation on a fresh accepted FD; no old admission state or
  // authority crosses that FD boundary.
  const int fresh = ConnectControl(control);
  if (fresh < 0) { close(b); close(wrong_peer); return false; }
  const std::string replacement_generation = fresh_generation + "c";
  ok = ok && ProfileOnFD(fresh, kSixteenthID, kProfilePyPI, replacement_generation) &&
      ControlAdmissionOnFD(fresh, kSixteenthID, kLaunchMode, replacement_generation, nonce_pending, "status", "rejected") &&
      ControlAdmissionOnFD(fresh, kSixteenthID, kLaunchMode, replacement_generation, nonce_pending, "arm", "armed") &&
      ControlAdmissionOnFD(wrong_peer, kSixteenthID, kLaunchMode, replacement_generation, nonce_pending, "status", "rejected") &&
      ControlAdmissionOnFD(fresh, kSixteenthID, kLaunchMode, replacement_generation, nonce_pending, "status", "pending") &&
      ControlAdmissionOnFD(fresh, kSixteenthID, kLaunchMode, replacement_generation, nonce_pending, "cancel", "cancelled");
  if (!ok) { close(b); close(wrong_peer); close(fresh); return false; }
  close(fresh);
  close(b);
  close(wrong_peer);
  return ok;
}

// HUP must erase the current profile and every live admission attached to the
// accepted FD. This runs without a remote stream, so stream-end cleanup cannot
// satisfy the assertion before RemoveControlPeer handles the EOF.
bool VerifyControlPeerHUPClearsLiveAdmission(const std::string& control) {
  const char id[] = "deadbeefdeadbeef";
  const char driver_id[] = "cafebabecafebabe";
  const std::string old_generation(64, 'd');
  const std::string driver_generation(64, 'e');
  const std::string fresh_generation(64, 'f');
  const std::string nonce = TestAdmissionNonce(1600, 16000);
  const int old = ConnectControl(control);
  if (old < 0) return false;
  bool ok = ProfileOnFD(old, id, kProfilePyPI, old_generation) &&
      ControlAdmissionOnFD(old, id, kLaunchMode, old_generation, nonce, "arm", "armed");
  if (!ok) { close(old); return false; }

  // Closing the accepted owner leaves a live PENDING slot solely for the
  // observer's production HUP path to erase.
  close(old);
  const int driver = ConnectControl(control);
  if (driver < 0) return false;
  ok = ProfileOnFD(driver, driver_id, kProfilePyPI, driver_generation);
  if (!ok) { close(driver); return false; }

  const int fresh = ConnectControl(control);
  if (fresh < 0) { close(driver); return false; }
  ok = ProfileOnFD(fresh, id, kProfilePyPI, fresh_generation) &&
      ControlAdmissionOnFD(fresh, id, kLaunchMode, fresh_generation, nonce, "status", "rejected") &&
      ControlAdmissionOnFD(fresh, id, kLaunchMode, fresh_generation, nonce, "arm", "armed") &&
      ControlAdmissionOnFD(driver, id, kLaunchMode, fresh_generation, nonce, "status", "rejected") &&
      ControlAdmissionOnFD(fresh, id, kLaunchMode, fresh_generation, nonce, "cancel", "cancelled");
  close(fresh);
  close(driver);
  return ok;
}

bool VerifyAcceptedPreAttributionDisconnect(const std::string& remote, pid_t helper) {
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) { if (client >= 0) close(client); return false; }
  close(client);
  int status = 0;
  return waitpid(helper, &status, 0) == helper && (!WIFEXITED(status) || WEXITSTATUS(status) != 0);
}

bool VerifyUnacceptedRemoteDisconnect(const std::string& remote) {
  const int client = ConnectRemote(remote);
  if (client < 0) return false;
  const char malformed[] = "not-a-remote-handshake";
  const bool sent = send(client, malformed, sizeof(malformed), 0) == static_cast<ssize_t>(sizeof(malformed));
  close(client);
  // The next verified profile registration exercised by the normal latch
  // sequence proves this unaccepted peer did not terminate the helper.
  return sent;
}


}  // namespace

// Provider profile mechanics are distinct from source approval and from the
// Go/Cargo resolver/compiler executable catalogues.
bool VerifyTerraformProviderProfileBoundary() {
 const char* profile = "terraform-provider";
 if (!IsCanonicalBootstrapProfile(profile) || MaximumRecords(profile) != 10000 ||
     ProcessClassForPath("/work/artifact", profile) != ProcessClass::kArtifact ||
     !IsBootstrapHandoffClass(ProcessClass::kArtifact, profile) ||
     !IsWorkspacePath("/work/artifact", profile)) return false;
 for (const char* path : {"/work/provider", "/usr/local/go/bin/go", "/usr/local/bin/node",
                         "/usr/local/bin/python3.14", kCargoBinary, kRustcBinary}) {
  if (ProcessClassForPath(path, profile) != ProcessClass::kUnknown) return false;
 }
 for (const auto process : {ProcessClass::kNode, ProcessClass::kGo, ProcessClass::kCargo,
                            ProcessClass::kCargoRustc, ProcessClass::kPython}) {
  if (IsBootstrapHandoffClass(process, profile)) return false;
 }
 return !IsWorkspacePath("/etc/passwd", profile) &&
   !IsBootstrapHandoffClass(ProcessClass::kArtifact, kProfileNPM) &&
   ProcessClassForPath("/work/artifact", kProfileGoBuild) == ProcessClass::kUnknown;
}

// Exercise the production request/result correlation and attested workspace
// dispatch, rather than only invoking the filesystem predicate directly.
bool VerifyRestrictedWorkspaceOpen(const gvisor::common::ContextData& context,
    const ProcessState& baseline, const TopologyState& topology, const char* profile,
    int output, int input, const std::string& path, uint64_t mount_id,
    uint32_t flags, bool outside, bool workspace=false,ProcessState* after=nullptr) {
  auto state=baseline;
  gvisor::syscall::Open open;
  *open.mutable_context_data()=context;
  open.mutable_context_data()->set_thread_id(context.thread_group_id());
  open.mutable_context_data()->set_thread_start_time_ns(context.thread_group_start_time_ns());
  open.set_pathname(path);open.set_flags(flags);open.set_sysno(257);
  std::string payload,id=kFirstID;const char* reason=nullptr;NormalizedCounts counts;
  if(!open.SerializeToString(&payload)||!ParseOpenAndSend(payload.data(),payload.size(),output,&id,profile,&state,&counts,&reason))return false;
  gvisor::syscall::OpenResult result;
  *result.mutable_context_data()=open.context_data();
  result.set_resolved_pathname(path);result.set_mount_id(mount_id);
  result.set_flags(flags);result.set_sysno(257);result.set_success(true);result.set_errorno(0);
  const bool parsed=result.SerializeToString(&payload)&&ParseOpenResultAndSend(payload.data(),payload.size(),output,&id,profile,&state,&counts,topology,&reason);
  if(after!=nullptr)*after=state;
  if(!parsed)return false;
  const bool counted=counts.immediate_records==(outside?1:0)&&counts.workspace_access==(workspace?1:0)&&state.pending_opens.empty();
  if(!counted){fprintf(stderr,"restricted workspace dispatch: profile=%s outside=%d findings=%llu workspace=%llu\n",profile,outside,static_cast<unsigned long long>(counts.immediate_records),static_cast<unsigned long long>(counts.workspace_access));DrainDatagram(input);return false;}
  return !outside||ExpectRecordExact(input,kFirstID,"filesystem-outside-workspace");
}

bool VerifyUVPythonQueryBoundary() {
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  auto run = [&](const std::string& scenario) {
    DrainDatagram(output[1]);
    const bool exported = scenario.compare(0, 7, "export-") == 0;
    const bool base_query = exported && scenario == "export-canonical";
    const bool cached = exported || scenario.compare(0, 6, "cache-") == 0;
    const std::string variation = exported ? scenario.substr(7) : (cached ? scenario.substr(6) : scenario);
    ProcessState state;
    gvisor::common::ContextData parent;
    parent.set_container_id(kFirstID); parent.set_thread_group_id(99);
    parent.set_thread_group_start_time_ns(990); parent.set_parent_thread_group_id(0);
    parent.set_is_exec_session(true); parent.set_process_name("renamed-uv");
    if (!RegisterGroup(&state, parent, ProcessState::Role::kControl,
        ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
    state.groups[99].current_image_class = ProcessClass::kUV;
    state.groups[99].trusted_control_network_active = true;
    state.expected_groups[99] = ProcessState::ExpectedGroup{990, ProcessClass::kUV};
    gvisor::sentry::CloneInfo clone;
    *clone.mutable_context_data() = parent;
    clone.set_created_thread_group_id(100); clone.set_created_thread_start_time_ns(1000);
    std::string payload, id; const char* reason = nullptr;
    if (!clone.SerializeToString(&payload) || !ParseSentryClone(payload.data(), payload.size(),
        output[0], &id, &state, &reason)) return false;
    TopologyState topology;
    topology.namespace_id = 10; topology.snapshot_seen = true; topology.sealed = true;
    topology.expected = {{"/", "oci-root", "/", "", true, false, false, false},
        {"/tmp", "workspace", "/", "tmpfs", false, true, true, false},
        {"/haa-runtime", "helper", "/", "9p", false, false, false, false}};
    topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
    topology.anchors.emplace(2, MountAnchor{2, "/tmp", "workspace"});
    topology.anchors.emplace(3, MountAnchor{3, "/haa-runtime", "helper"});
    topology.expected.push_back({"/haa-site", "workspace", "/", "9p", false, false, false, false});
    topology.anchors.emplace(6, MountAnchor{6, "/haa-site", "workspace"});
    gvisor::sentry::ExecveInfo image;
    *image.mutable_context_data() = parent;
    auto* child = image.mutable_context_data();
    child->set_thread_group_id(100); child->set_thread_group_start_time_ns(1000);
    // Kernel task_clone.go inherits OriginExec from the actual UV parent.
    child->set_parent_thread_group_id(99); child->set_is_exec_session(true);
    child->set_process_name("arbitrary-name");
    image.set_binary_path("/usr/local/bin/python3.14");
    image.set_execfn(variation == "canonical" ? "/usr/local/bin/python3.14" : "/usr/local/bin/python");
    if (scenario.compare(0, 5, "tool-") == 0) image.set_execfn("/haa-site/tools/cowsay/bin/python");
    if (variation == "tool-traversal") image.set_execfn("/haa-site/tools/../cowsay/bin/python");
    if (variation == "tool-case") image.set_execfn("/haa-site/tools/Cowsay/bin/python");
    if (variation == "tool-other-alias") image.set_execfn("/haa-site/tools/cowsay/bin/python3");
    if (variation == "tool-underscore") image.set_execfn("/haa-site/tools/cow_say/bin/python");
    const std::string cache_root = exported ? "/haa-site/uv-run-cache" : "/tmp/.tmpzw5M7c";
    const std::string cache_env = exported ? cache_root + "/archive-v0/" +
        (variation == "id-alphabet" ? "_-0123456789abcd" : "aCTj1Rn4JBKu2JYF") : (variation == "id-alphabet" ?
        "/tmp/.tmpA1b2C3/archive-v0/_-0123456789abcd" :
        "/tmp/.tmpzw5M7c/archive-v0/aCTj1Rn4JBKu2JYF");
    const std::string helper_root = exported ?
        (variation == "helper-other" ? "/haa-site/other-cache/.tmpA1s2D3" :
         (variation == "helper-short" ? "/haa-site/uv-run-cache/.tmpA1s2D" :
          (variation == "helper-punct" ? "/haa-site/uv-run-cache/.tmpA1s2D_" : "/haa-site/uv-run-cache/.tmpA1s2D3"))) : (variation == "helper-other" ?
        "/tmp/.tmpQ1w2E3/.tmpA1s2D3" : "/tmp/.tmp3l9KcP/.tmpADARLv");
    if (cached && !base_query) image.set_execfn(cache_env + "/bin/python");
    if (cached && variation == "tmp-length") image.set_execfn("/tmp/.tmpzw5M7/archive-v0/aCTj1Rn4JBKu2JYF/bin/python");
    if (cached && variation == "tmp-punct") image.set_execfn("/tmp/.tmpzw5M7_/archive-v0/aCTj1Rn4JBKu2JYF/bin/python");
    if (cached && variation == "bucket") image.set_execfn("/tmp/.tmpzw5M7c/archive-v1/aCTj1Rn4JBKu2JYF/bin/python");
    if (cached && variation == "id-short") image.set_execfn("/tmp/.tmpzw5M7c/archive-v0/aCTj1Rn4JBKu2JY/bin/python");
    if (cached && variation == "id-long") image.set_execfn("/tmp/.tmpzw5M7c/archive-v0/aCTj1Rn4JBKu2JYFQ/bin/python");
    if (cached && variation == "id-dot") image.set_execfn("/tmp/.tmpzw5M7c/archive-v0/.CTj1Rn4JBKu2JYF/bin/python");
    if (cached && variation == "alias") image.set_execfn(cache_env + "/bin/python3");
    if (cached && variation == "path-traversal") image.set_execfn(cache_env + "/bin/../bin/python");
    if (cached && variation == "double-slash") image.set_execfn("/tmp/.tmpzw5M7c//archive-v0/aCTj1Rn4JBKu2JYF/bin/python");
    image.add_argv(image.execfn()); image.add_argv("-I"); image.add_argv("-B"); image.add_argv("-c");
    const std::string launcher = "import sys; sys.path = [__import__(\"os\").fsdecode(b\"/tmp/.tmp3l9KcP/.tmpADARLv\")] + sys.path; from python.get_interpreter_info import main; main()";
    image.add_argv(launcher);
    if (exported || (cached && variation == "helper-other")) {
      auto code = launcher;
      code.replace(code.find("/tmp/.tmp3l9KcP/.tmpADARLv"), 26, helper_root);
      image.set_argv(4, code);
    }
    if (cached && variation == "cache-shadow") topology.anchors.emplace(7, MountAnchor{7, cache_root, "workspace"});
    if (cached && variation == "lib-shadow") topology.anchors.emplace(7, MountAnchor{7, cache_env + "/lib", "workspace"});
    if (cached && variation == "helper-shadow") topology.anchors.emplace(7, MountAnchor{7, helper_root + "/python", "workspace"});
    if (cached && variation == "tmp-nosuid") topology.expected[1].nosuid = false;
    if (cached && variation == "tmp-readonly") topology.expected[1].read_only = true;
    if (cached && variation == "tmp-duplicate") topology.anchors.emplace(7, MountAnchor{7, "/tmp", "workspace"});
    if (cached && variation == "tmp-class") topology.anchors[2].mount_class = "system";
    if (exported && variation == "output-missing") topology.anchors.erase(6);
    if (exported && variation == "output-kind") topology.expected[3].filesystem_type = "tmpfs";
    if (exported && variation == "output-readonly") topology.expected[3].read_only = true;
    if (exported && variation == "output-duplicate") topology.anchors.emplace(7, MountAnchor{7, "/haa-site", "workspace"});
    if (exported && variation == "output-class") topology.anchors[6].mount_class = "system";
    if (exported && variation == "cache-root") image.set_execfn("/haa-site/uv-run-cache-other/archive-v0/aCTj1Rn4JBKu2JYF/bin/python");
    if (variation == "binary") image.set_binary_path("/tmp/python3.14");
    if (variation == "execfn") image.set_execfn("/usr/local/bin/python3");
    if (variation == "argv0") image.set_argv(0, "python");
    if (variation == "no-isolation") image.set_argv(1, "-E");
    if (variation == "bytecode") image.set_argv(2, "-s");
    if (variation == "module") image.set_argv(3, "-m");
    if (variation == "arbitrary-code") image.set_argv(4, "print('query')");
    if (variation == "extra-code") image.set_argv(4, launcher + "; print('extra')");
    if (variation == "embedded-nul") image.set_argv(4, launcher + std::string(1, '\0'));
    if (variation == "traversal") { auto code = launcher; code.replace(code.find("3l9KcP"), 6, "../foo"); image.set_argv(4, code); }
    if (variation == "escaped-path") { auto code = launcher; code.replace(code.find("3l9KcP"), 6, "abc\\de"); image.set_argv(4, code); }
    if (variation == "wrong-module") { auto code = launcher; code.replace(code.find("get_interpreter_info"), 20, "get_interpreter_evil"); image.set_argv(4, code); }
    if (variation == "extra-arg") image.add_argv("/tmp/artifact.py");
    if (variation == "short-argv") image.mutable_argv()->RemoveLast();
    if (variation == "parent-class") state.groups[99].current_image_class = ProcessClass::kPython;
    if (variation == "parent-role") state.groups[99].role = ProcessState::Role::kArtifact;
    if (variation == "parent-start") state.groups[99].start_time_ns++;
    if (variation == "parent-expected") state.expected_groups.erase(99);
    if (variation == "parent-generation") state.expected_groups[99].start_time_ns++;
    if (variation == "parent-expected-class") state.expected_groups[99].process_class = ProcessClass::kUnknown;
    if (variation == "parent-network") state.groups[99].trusted_control_network_active = false;
    if (variation == "parent-eligible") state.groups[99].root_eligible = true;
    if (variation == "parent-unconsumed") state.groups[99].root_consumed = false;
    if (variation == "parent-pending") state.groups[99].handoff_target_pending = true;
    if (variation == "parent-clone") state.groups[99].provenance = ProcessState::Provenance::kCloneChild;
    if (variation == "oci-origin") child->set_is_exec_session(false);
    if (variation == "wrong-parent") child->set_parent_thread_group_id(98);
    if (variation == "direct-root") { child->set_parent_thread_group_id(0); child->set_is_exec_session(true); state.groups[100].provenance = ProcessState::Provenance::kDirectExecRoot; }
    if (variation == "artifact-child") state.groups[100].role = ProcessState::Role::kArtifact;
    if (variation == "missing-clone") state.groups[100].uv_python_query_candidate = false;
    if (variation == "child-network") state.groups[100].trusted_control_network_active = true;
    if (variation == "child-pending") state.groups[100].launch_target_pending = true;
    if (variation == "unsealed") topology.sealed = false;
    if (variation == "missing-snapshot") topology.snapshot_seen = false;
    if (variation == "writable-root") topology.expected[0].read_only = false;
    if (variation == "image-shadow") topology.anchors.emplace(4, MountAnchor{4, "/usr/local", "workspace"});
    if (variation == "runtime-shadow") topology.anchors.emplace(4, MountAnchor{4, "/haa-runtime/uv", "helper"});
    if (variation == "query-shadow") topology.anchors.emplace(4, MountAnchor{4, exported ? helper_root : "/tmp/.tmp3l9KcP", "workspace"});
    if (variation == "tmp-bind") topology.expected[1].filesystem_type = "9p";
    if (variation == "tmp-exec") topology.expected[1].noexec = false;
    if (variation == "missing-tmp") topology.anchors.erase(2);
    if (variation == "tool-missing-output") topology.anchors.erase(6);
    if (variation == "tool-output-kind") topology.expected[3].filesystem_type = "tmpfs";
    if (variation == "tool-output-readonly") topology.expected[3].read_only = true;
    if (variation == "tool-output-shadow") topology.anchors.emplace(5, MountAnchor{5, "/haa-site/tools/cowsay", "workspace"});
    const auto* selected = variation == "no-topology" ? nullptr : &topology;
    const char* profile = variation == "other-profile" ? kProfilePyPI : kProfileUVResolver;
    if (variation == "profile-cpu") profile = kProfilePyTorchCPU;
    if (variation == "profile-cu126") profile = kProfilePyTorchCU126;
    if (variation == "profile-cu130") profile = kProfilePyTorchCU130;
    if (variation == "profile-cu132") profile = kProfilePyTorchCU132;
    ProfileRegistration registration;
    const bool parsed = image.SerializeToString(&payload) && ParseSentryProcessAndClassify(
        payload.data(), payload.size(), output[0], &id, profile, &registration, &state, &reason, selected);
    const bool normal = variation == "normal" || variation == "canonical" || variation == "tool-normal" ||
        (cached && (variation == "id-alphabet" || (!exported && variation == "helper-other")));
    if (normal && (!parsed || !ExpectRecordExact(output[1], kFirstID, "process-exec-expected"))) return false;
    if (!normal) DrainDatagram(output[1]);
    if (state.groups[100].uv_python_query_active != normal) return false;
    gvisor::syscall::Open open;
    *open.mutable_context_data() = image.context_data(); open.set_flags(557056);
    open.set_pathname("/usr/local/lib/libpython3.14.so.1.0");
    MountAnchor anchor{1, "/", "oci-root"};
    auto readable = [&]() { return IsPinnedUVPythonQueryRead(open.context_data(), state, open.pathname(), open.flags(), profile, &anchor); };
    if (readable() != normal) return false;
    anchor = MountAnchor{2, "/proc", "system"};
    open.set_pathname("/proc/sys/vm/overcommit_memory"); open.set_flags(kOpenLargefile);
    if (readable() != normal) return false;
    if (normal && ClassifyFilesystemOpen(open, state, profile, &anchor) != FilesystemClass::kHelperOnly) return false;
    open.set_pathname("/proc/100/maps"); open.set_flags(557056);
    if (readable() != normal) return false;
    if (normal && ClassifyFilesystemOpen(open, state, profile, &anchor) != FilesystemClass::kHelperOnly) return false;
    if (!normal) return true;
    const MountAnchor cache_anchor{exported ? 6u : 2u, exported ? "/haa-site" : "/tmp", "workspace"};
    if (cached) {
      const MountAnchor tmp = cache_anchor;
      auto expected_cache_read = [&](const std::string& path) {
        return base_query && !IsAtOrBelowMountpoint(path, helper_root) ? FilesystemClass::kOutside : FilesystemClass::kHelperOnly;
      };
      auto check = [&](const std::string& path, uint64_t flags, FilesystemClass expected) {
        auto candidate = open; candidate.set_pathname(path); candidate.set_flags(flags);
        return ClassifyFilesystemOpen(candidate, state, profile, &tmp) == expected;
      };
      for (const std::string& path : {helper_root, helper_root + "/python", helper_root + "/python/__init__.py",
          helper_root + "/python/get_interpreter_info.py", cache_env, cache_env + "/lib",
          cache_env + "/lib/python3.14", cache_env + "/lib/python3.14/site-packages"}) {
        if (!check(path, 557056, expected_cache_read(path)) ||
            !check(path, 622592, expected_cache_read(path))) return false;
      }
      if (!check(cache_env + "/pyvenv.cfg", 557056, expected_cache_read(cache_env + "/pyvenv.cfg"))) return false;
      if (!check(cache_env + "/pyvenv.cfg", kOpenLargefile, expected_cache_read(cache_env + "/pyvenv.cfg"))) return false;
      for (const std::string& path : {helper_root, helper_root + "/python", helper_root + "/python/packaging",
          cache_env + "/lib/python3.14/site-packages"}) {
        if (!check(path, 624640, expected_cache_read(path))) return false;
      }
      for (const std::string& path : {helper_root + "/python/__init__.py",
          helper_root + "/python/get_interpreter_info.py", helper_root + "/python/packaging/__init__.py",
          helper_root + "/python/packaging/tags.py", cache_env, cache_env + "/lib",
          cache_env + "/lib/python3.14", cache_env + "/pyvenv.cfg"}) {
        if (!check(path, 624640, FilesystemClass::kOutside)) return false;
      }
      for (const std::string& path : {helper_root, helper_root + "/python/__init__.py",
          helper_root + "/python/get_interpreter_info.py", cache_env + "/lib/python3.14/site-packages"}) {
        if (!check(path, kOpenLargefile, FilesystemClass::kOutside)) return false;
      }
      for (const std::string& path : {cache_env + "/lib/python3.14/site-packages/evil.pth",
          cache_env + "/lib/python3.14/site-packages/sitecustomize.py",
          cache_env + "/lib/python3.14/site-packages/usercustomize.py",
          cache_env + "/lib/python3.14/site-packages/haa_uv_probe/__init__.py",
          cache_env + "/lib/python3.14/site-packages/evil.so",
          cache_env + "/lib/python3.14/site-packages/haa_uv_probe-1.0.dist-info/METADATA",
          cache_env + "/../" + cache_env.substr(cache_env.rfind('/') + 1) + "/pyvenv.cfg",
          helper_root + "/../" + helper_root.substr(helper_root.rfind('/') + 1),
          std::string("/tmp/.tmpQ1w2E3/archive-v0/OtherEnvironment/pyvenv.cfg"),
          std::string("/tmp/unrelated.py")}) {
        for (uint64_t flags : {kOpenLargefile, uint64_t{624640}}) {
          if (!check(path, flags, FilesystemClass::kOutside)) return false;
        }
      }

      for (const std::string& path : {cache_env + "/lib/python3.14/site-packages/evil.pth",
          cache_env + "/lib/python3.14/site-packages/sitecustomize.py",
          cache_env + "/lib/python3.14/site-packages/usercustomize.py",
          cache_env + "/lib/python3.14/site-packages/haa_uv_probe/__init__.py",
          cache_env + "/lib/python3.14/site-packages/evil.so", cache_env + "/bin/python",
          cache_env + "/lib/python3.14/site-packages/haa_uv_probe-1.0.dist-info/METADATA",
          std::string("/tmp/unrelated.py"), std::string("/tmp/.tmpOther1/archive-v0/OtherEnvironment/pyvenv.cfg"),
          std::string("/tmp/.tmpzw5M7c/archive-v0/OtherEnvironment/pyvenv.cfg"),
          helper_root + "/../" + helper_root.substr(helper_root.rfind('/') + 1) + "/python/__init__.py"}) {
        if (!check(path, 557056, FilesystemClass::kOutside)) return false;
      }
      const std::string foreign_helper = variation == "helper-other" ?
          "/tmp/.tmp3l9KcP/.tmpADARLv/python/__init__.py" : "/tmp/.tmpQ1w2E3/.tmpA1s2D3/python/__init__.py";
      if (!check(foreign_helper, 557056, FilesystemClass::kOutside)) return false;
      for (uint64_t flags : {uint64_t{0}, kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenAppend, kOpenTruncate}) {
        for (const std::string& path : {helper_root + "/python/__init__.py", cache_env + "/pyvenv.cfg", std::string("/tmp/query-write")}) {
          if (!check(path, flags, FilesystemClass::kOutside)) return false;
        }
      }
      auto candidate = open; candidate.set_pathname(cache_env + "/pyvenv.cfg"); candidate.set_flags(557056);
      for (const auto& wrong : {MountAnchor{4, exported ? "/haa-site" : "/tmp", "system"}, MountAnchor{4, cache_env, "workspace"}, MountAnchor{1, "/", "oci-root"}}) {
        if (ClassifyFilesystemOpen(candidate, state, profile, &wrong) != FilesystemClass::kOutside) return false;
      }
      const auto accepted_query = state;
      for (int mutation = 0; mutation < 13; ++mutation) {
        state = accepted_query; auto& query = state.groups[100];
        switch (mutation) {
          case 0: query.role = ProcessState::Role::kControl; break;
          case 1: query.uv_python_query_active = false; break;
          case 2: query.start_time_ns++; break;
          case 3: query.provenance = ProcessState::Provenance::kDirectExecRoot; break;
          case 4: query.trusted_control_network_active = true; break;
          case 5: query.current_image_class = ProcessClass::kUnknown; break;
          case 6: query.root_eligible = true; break;
          case 7: query.root_consumed = false; break;
          case 8: query.demotion_pending = true; break;
          case 9: query.launch_target_pending = true; break;
          case 10: query.handoff_target_pending = true; break;
          case 11: state.groups[99].start_time_ns++; break;
          case 12: state.expected_groups.erase(99); break;
        }
        // A stale generation has no authoritative filesystem group. Keep the
        // classifier's UNKNOWN result rather than inventing an ARTIFACT role.
        const auto rejected = mutation == 2 ? FilesystemClass::kUnknown : FilesystemClass::kOutside;
        if (!check(cache_env + "/pyvenv.cfg", 557056, rejected) ||
            !check(helper_root + "/python/__init__.py", 557056, rejected) ||
            !check(cache_env + "/pyvenv.cfg", kOpenLargefile, rejected) ||
            !check(helper_root, 624640, rejected) ||
            !check(cache_env + "/lib/python3.14/site-packages", 624640, rejected)) return false;
      }
      state = accepted_query;
    }
    const std::string env = "/haa-site/tools/cowsay";
    if(!VerifyRestrictedWorkspaceOpen(image.context_data(),state,topology,profile,output[0],output[1],env+"/lib/python3.14/site-packages/evil.pth",6,557056,true))return false;
    if(!VerifyRestrictedWorkspaceOpen(image.context_data(),state,topology,profile,output[0],output[1],env+"/pyvenv.cfg",6,kOpenWriteOnly,true))return false;
    if(!VerifyRestrictedWorkspaceOpen(image.context_data(),state,topology,profile,output[0],output[1],"/tmp/query-write",2,kOpenWriteOnly,cached,!cached))return false;
    if(scenario=="tool-normal"&&!VerifyRestrictedWorkspaceOpen(image.context_data(),state,topology,profile,output[0],output[1],env+"/pyvenv.cfg",6,557056,false))return false;

    anchor = MountAnchor{4, "/haa-site", "workspace"};
    for (const std::string& path : {env + "/pyvenv.cfg", env, env + "/lib", env + "/lib/python3.14", env + "/lib/python3.14/site-packages"}) {
      open.set_pathname(path); open.set_flags(557056);
      const auto expected = variation == "tool-normal" ? FilesystemClass::kHelperOnly : FilesystemClass::kOutside;
      if (ClassifyFilesystemOpen(open, state, profile, &anchor) != expected) return false;
    }
    for (const std::string& path : {env + "/lib/python3.14/site-packages/evil.pth", env + "/lib/python3.14/site-packages/sitecustomize.py",
        env + "/lib/python3.14/site-packages/cowsay/__init__.py", env + "/lib/python3.14/site-packages/evil.so",
        env + "/lib/python3.14/site-packages/cowsay-6.1.dist-info/METADATA", std::string("/haa-site/tools/other/pyvenv.cfg"),
        env + "/../cowsay/pyvenv.cfg", std::string("/haa-site/pylock.toml"),
        std::string("/tmp/../haa-site/tools/cowsay/lib/python3.14/site-packages/evil.pth"),
        std::string("/tmp/output-alias/evil.pth")}) {
      open.set_pathname(path); open.set_flags(557056);
      if (ClassifyFilesystemOpen(open, state, profile, &anchor) != FilesystemClass::kOutside) return false;
    }
    open.set_pathname(env + "/pyvenv.cfg");
    for (uint64_t flags : {kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenAppend, kOpenTruncate}) {
      open.set_flags(flags);
      if (ClassifyFilesystemOpen(open, state, profile, &anchor) != FilesystemClass::kOutside) return false;
    }
    open.set_flags(557056);
    for (const auto& wrong : {MountAnchor{5, "/haa-site/tools", "workspace"}, MountAnchor{4, "/haa-site", "system"}}) {
      if (ClassifyFilesystemOpen(open, state, profile, &wrong) != FilesystemClass::kOutside) return false;
    }
    anchor = MountAnchor{2, "/proc", "system"};
    for (const char* path : {"/proc/sys/vm/overcommit_ratio", "/proc/99/maps", "/proc/101/maps", "/proc/self/maps", "/proc/100/environ", "/proc/100/status", "/proc/100/cgroup", "/dev/null", "/sys/fs/cgroup/cpu/cpu.cfs_quota_us"}) {
      open.set_pathname(path); if (readable()) return false;
    }
    open.set_pathname("/proc/100/maps");
    for (uint64_t flags : {uint64_t{0}, kOpenLargefile, kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenTruncate, kOpenAppend}) {
      open.set_flags(flags); if (readable()) return false;
    }
    open.set_pathname("/proc/sys/vm/overcommit_memory");
    for (uint64_t flags : {uint64_t{0}, uint64_t{557056}, kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenTruncate, kOpenAppend}) {
      open.set_flags(flags); if (readable()) return false;
    }
    for (bool maps : {false, true}) {
      open.set_pathname(maps ? "/proc/100/maps" : "/proc/sys/vm/overcommit_memory");
      open.set_flags(maps ? 557056 : kOpenLargefile);
      for (const auto& wrong : {MountAnchor{2, "/proc/sys", "system"}, MountAnchor{2, "/proc", "workspace"}, MountAnchor{2, "/", "oci-root"}}) {
        anchor = wrong; if (readable()) return false;
      }
    }
    anchor = MountAnchor{1, "/", "oci-root"};
    open.set_pathname("/usr/local/lib/libpython3.14.so.1.0"); open.set_flags(557056);
    if (state.groups[100].role != ProcessState::Role::kArtifact ||
        state.expected_groups.count(100) != 0 || IsTrustedControlNetwork(*child, state) ||
        ClassifyFilesystemOpen(open, state, profile, &anchor) != FilesystemClass::kHelperOnly) return false;
    for (const char* path : {"/etc/ld.so.cache", "/usr/local/bin/../lib/libpython3.14.so.1.0", "/usr/local/bin/python3.14",
        "/usr/share/zoneinfo/Etc/UTC",
        "/usr/local/lib/python314.zip", "/usr/local/lib/python3.14", "/usr/local/lib/python3.14/encodings/__init__.py",
        "/usr/local/lib/python3.14/lib-dynload/_struct.cpython-314-x86_64-linux-gnu.so", "/lib/x86_64-linux-gnu/libc.so.6"}) {
      open.set_pathname(path); if (!readable()) return false;
    }
    for (const char* path : {"/etc/passwd", "/etc/resolv.conf", "/proc/100/environ", "/usr/lib/evil.so", "/usr/local/bin/pip",
        "/usr/share/zoneinfo/Etc/UTC.evil", "/usr/share/zoneinfo/Europe/London", "/etc/localtime",
        "/usr/local/lib/python3.14/site-packages/pip/__init__.py", "/usr/local/lib/python3.14/site-packages/evil.pth",
        "/usr/local/lib/python3.15/encodings/__init__.py", "/usr/local/lib/python3.14/../../evil.py", "/tmp/module.py"}) {
      open.set_pathname(path); if (readable()) return false;
    }
    open.set_pathname("/usr/share/zoneinfo/Etc/UTC");
    for (uint64_t flags : {uint64_t{0}, kOpenLargefile, kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenTruncate, kOpenAppend}) {
      open.set_flags(flags); if (readable()) return false;
    }
    open.set_flags(557056);
    for (const auto& wrong : {MountAnchor{2, "/usr/share/zoneinfo", "oci-root"}, MountAnchor{2, "/", "system"}, MountAnchor{2, "/", "workspace"}}) {
      anchor = wrong; if (readable()) return false;
    }
    anchor = MountAnchor{1, "/", "oci-root"};
    open.set_pathname("/usr/local/lib/libpython3.14.so.1.0");
    for (uint64_t flags : {kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenTruncate, kOpenAppend}) { open.set_flags(flags); if (readable()) return false; }
    open.set_flags(557056);
    for (const auto& wrong : {MountAnchor{2, "/usr/local", "oci-root"}, MountAnchor{2, "/", "workspace"}, MountAnchor{2, "/", "system"}}) {
      anchor = wrong; if (readable()) return false;
    }
    anchor = MountAnchor{1, "/", "oci-root"};
    const auto accepted = state;
    for (int subject : {0, 1, 2, 3}) {
      anchor = subject == 0 || subject == 3 ? MountAnchor{1, "/", "oci-root"} : MountAnchor{2, "/proc", "system"};
      open.set_pathname(subject == 3 ? "/usr/share/zoneinfo/Etc/UTC" :
          (subject == 0 ? "/usr/local/lib/libpython3.14.so.1.0" :
           (subject == 1 ? "/proc/sys/vm/overcommit_memory" : "/proc/100/maps")));
      open.set_flags(subject == 1 ? kOpenLargefile : 557056);
      for (int mutation = 0; mutation < 13; ++mutation) {
        state = accepted;
        auto& group = state.groups[100];
        switch (mutation) {
          case 0: group.role = ProcessState::Role::kControl; break;
          case 1: group.uv_python_query_active = false; break;
          case 2: group.start_time_ns++; break;
          case 3: group.provenance = ProcessState::Provenance::kDirectExecRoot; break;
          case 4: group.trusted_control_network_active = true; break;
          case 5: group.current_image_class = ProcessClass::kUnknown; break;
          case 6: group.root_eligible = true; break;
          case 7: group.root_consumed = false; break;
          case 8: group.demotion_pending = true; break;
          case 9: group.launch_target_pending = true; break;
          case 10: group.handoff_target_pending = true; break;
          case 11: state.groups[99].start_time_ns++; break;
          case 12: state.expected_groups.erase(99); break;
        }
        if (readable()) return false;
      }
    }
    anchor = MountAnchor{1, "/", "oci-root"};
    open.set_pathname("/usr/local/lib/libpython3.14.so.1.0"); open.set_flags(557056);
    state = accepted;
    // A query clone inherits neither the active role nor another candidate.
    *clone.mutable_context_data() = *child;
    clone.set_created_thread_group_id(101); clone.set_created_thread_start_time_ns(1010);
    if (!clone.SerializeToString(&payload) || !ParseSentryClone(payload.data(), payload.size(), output[0], &id, &state, &reason)) return false;
    auto descendant = image;
    descendant.mutable_context_data()->set_thread_group_id(101);
    descendant.mutable_context_data()->set_thread_group_start_time_ns(1010);
    descendant.mutable_context_data()->set_parent_thread_group_id(100);
    *open.mutable_context_data() = descendant.context_data();
    if (state.groups[101].uv_python_query_candidate || state.groups[101].uv_python_query_active ||
        !state.groups[101].uv_python_query_restricted || readable()) return false;
    open.set_pathname(env + "/lib/python3.14/site-packages/evil.pth");
    MountAnchor tool_anchor{4, "/haa-site", "workspace"};
    if (ClassifyFilesystemOpen(open, state, profile, &tool_anchor) != FilesystemClass::kOutside) return false;
    open.set_pathname("/usr/local/lib/libpython3.14.so.1.0");
    if (!descendant.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(), output[0], &id, profile, &registration, &state, &reason, selected) ||
        !ExpectUnexpectedProcessRecord(output[1], kFirstID, "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP") || readable()) return false;
    if (cached) {
      open.set_pathname(helper_root + "/python/__init__.py");
      MountAnchor tmp = cache_anchor;
      if (ClassifyFilesystemOpen(open, state, profile, &tmp) != FilesystemClass::kOutside) return false;
      auto read_mode = open;
      read_mode.set_pathname(helper_root); read_mode.set_flags(624640);
      if (ClassifyFilesystemOpen(read_mode, state, profile, &tmp) != FilesystemClass::kOutside) return false;
      read_mode.set_pathname(cache_env + "/pyvenv.cfg"); read_mode.set_flags(kOpenLargefile);
      if (ClassifyFilesystemOpen(read_mode, state, profile, &tmp) != FilesystemClass::kOutside) return false;
    }
    // A second exact exec in the original child cannot renew the role.
    *open.mutable_context_data() = *child;
    if (!image.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(), payload.size(), output[0], &id, profile, &registration, &state, &reason, selected) ||
        !ExpectUnexpectedProcessRecord(output[1], kFirstID, "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP") || readable() || IsTrustedControlNetwork(*child, state)) return false;
    if (cached) {
      open.set_pathname(cache_env + "/pyvenv.cfg");
      MountAnchor tmp = cache_anchor;
      if (ClassifyFilesystemOpen(open, state, profile, &tmp) != FilesystemClass::kOutside) return false;
      auto read_mode = open;
      read_mode.set_pathname(helper_root); read_mode.set_flags(624640);
      if (ClassifyFilesystemOpen(read_mode, state, profile, &tmp) != FilesystemClass::kOutside) return false;
      read_mode.set_pathname(cache_env + "/pyvenv.cfg"); read_mode.set_flags(kOpenLargefile);
      if (ClassifyFilesystemOpen(read_mode, state, profile, &tmp) != FilesystemClass::kOutside) return false;
    }
    return true;
  };
  bool passed = true;
  for (const char* scenario : {
      "normal", "canonical", "tool-normal", "tool-traversal", "tool-case", "tool-underscore",
      "tool-other-alias", "tool-missing-output", "tool-output-kind", "tool-output-readonly", "tool-output-shadow", "binary",
      "execfn", "argv0", "no-isolation", "bytecode", "module", "arbitrary-code",
      "extra-code", "embedded-nul", "traversal", "escaped-path", "wrong-module", "extra-arg",
      "short-argv", "parent-class", "parent-role", "parent-start", "parent-expected", "parent-generation",
      "parent-expected-class", "parent-network", "parent-eligible", "parent-unconsumed", "parent-pending", "parent-clone",
      "oci-origin", "wrong-parent", "direct-root", "artifact-child", "missing-clone", "child-network",
      "child-pending", "unsealed", "missing-snapshot", "writable-root", "image-shadow", "runtime-shadow",
      "query-shadow", "tmp-bind", "tmp-exec", "missing-tmp", "no-topology", "other-profile",
      "cache-normal", "cache-id-alphabet", "cache-helper-other", "cache-tmp-length", "cache-tmp-punct", "cache-bucket",
      "cache-id-short", "cache-id-long", "cache-id-dot", "cache-alias", "cache-path-traversal", "cache-double-slash",
      "cache-cache-shadow", "cache-lib-shadow", "cache-helper-shadow", "cache-tmp-nosuid", "cache-tmp-readonly", "cache-tmp-duplicate",
      "cache-tmp-class", "cache-profile-cpu", "cache-profile-cu126", "cache-profile-cu130", "cache-profile-cu132", "cache-binary",
      "cache-execfn", "cache-argv0", "cache-no-isolation", "cache-bytecode", "cache-module", "cache-arbitrary-code",
      "cache-extra-code", "cache-embedded-nul", "cache-traversal", "cache-escaped-path", "cache-wrong-module", "cache-extra-arg",
      "cache-short-argv", "cache-parent-class", "cache-parent-role", "cache-parent-start", "cache-parent-expected", "cache-parent-generation",
      "cache-parent-expected-class", "cache-parent-network", "cache-parent-eligible", "cache-parent-unconsumed", "cache-parent-pending", "cache-parent-clone",
      "cache-oci-origin", "cache-wrong-parent", "cache-direct-root", "cache-artifact-child", "cache-missing-clone", "cache-child-network",
      "cache-child-pending", "cache-unsealed", "cache-missing-snapshot", "cache-writable-root", "cache-image-shadow", "cache-runtime-shadow",
      "cache-query-shadow", "cache-tmp-bind", "cache-tmp-exec", "cache-missing-tmp", "cache-no-topology", "cache-other-profile",
      "export-normal", "export-id-alphabet", "export-helper-other", "export-tmp-length", "export-tmp-punct", "export-bucket", "export-id-short", "export-id-long", "export-id-dot", "export-alias", "export-path-traversal", "export-double-slash", "export-cache-shadow", "export-lib-shadow", "export-helper-shadow", "export-tmp-nosuid", "export-tmp-readonly", "export-tmp-duplicate", "export-tmp-class", "export-profile-cpu", "export-profile-cu126", "export-profile-cu130", "export-profile-cu132", "export-binary", "export-execfn", "export-argv0", "export-no-isolation", "export-bytecode", "export-module", "export-arbitrary-code", "export-extra-code", "export-embedded-nul", "export-traversal", "export-escaped-path", "export-wrong-module", "export-extra-arg", "export-short-argv", "export-parent-class", "export-parent-role", "export-parent-start", "export-parent-expected", "export-parent-generation", "export-parent-expected-class", "export-parent-network", "export-parent-eligible", "export-parent-unconsumed", "export-parent-pending", "export-parent-clone", "export-oci-origin", "export-wrong-parent", "export-direct-root", "export-artifact-child", "export-missing-clone", "export-child-network", "export-child-pending", "export-unsealed", "export-missing-snapshot", "export-writable-root", "export-image-shadow", "export-runtime-shadow", "export-query-shadow", "export-tmp-bind", "export-tmp-exec", "export-missing-tmp", "export-no-topology", "export-other-profile", "export-canonical", "export-output-missing", "export-output-kind", "export-output-readonly", "export-output-duplicate", "export-output-class", "export-cache-root", "export-helper-short", "export-helper-punct"}) {
    const bool result = run(scenario);
    if (!result) fprintf(stderr, "UV query boundary failed: %s\n", scenario);
    passed = result && passed;
  }
  close(output[0]); close(output[1]);
  return passed;
}

bool VerifyPNPMNodeQueryBoundary() {
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  auto run = [&](const std::string& scenario) {
    DrainDatagram(output[1]);
    ProcessState state;
    gvisor::common::ContextData parent;
    parent.set_container_id(kFirstID); parent.set_thread_group_id(99);
    parent.set_thread_group_start_time_ns(990); parent.set_parent_thread_group_id(0);
    parent.set_is_exec_session(true); parent.set_process_name("renamed-pnpm");
    if (!RegisterGroup(&state, parent, ProcessState::Role::kControl,
        ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
    state.groups[99].current_image_class = ProcessClass::kPNPM;
    state.groups[99].trusted_control_network_active = true;
    state.expected_groups[99] = ProcessState::ExpectedGroup{990, ProcessClass::kPNPM};
    gvisor::sentry::CloneInfo clone;
    *clone.mutable_context_data() = parent;
    clone.set_created_thread_group_id(100); clone.set_created_thread_start_time_ns(1000);
    std::string payload, id; const char* reason = nullptr;
    if (!clone.SerializeToString(&payload) || !ParseSentryClone(payload.data(), payload.size(),
        output[0], &id, &state, &reason)) return false;
    TopologyState topology;
    topology.namespace_id = 10; topology.snapshot_seen = true; topology.sealed = true;
    topology.expected = {{"/", "oci-root", "/", "", true, false, false, false},
        {"/tmp", "workspace", "/", "tmpfs", false, true, true, false},
        {"/haa-runtime", "helper", "/", "9p", false, false, false, false},
        {"/tmp/haa-pnpm-input", "workspace", "/tmp", "9p", false, false, false, false}};
    topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
    topology.anchors.emplace(2, MountAnchor{2, "/tmp", "workspace"});
    topology.anchors.emplace(3, MountAnchor{3, "/haa-runtime", "helper"});
    topology.anchors.emplace(4, MountAnchor{4, "/tmp/haa-pnpm-input", "workspace"});
    gvisor::sentry::ExecveInfo image;
    *image.mutable_context_data() = parent;
    auto* child = image.mutable_context_data();
    child->set_thread_group_id(100); child->set_thread_group_start_time_ns(1000);
    child->set_parent_thread_group_id(99); child->set_is_exec_session(true);
    child->set_process_name("arbitrary-name");
    image.set_binary_path("/usr/local/bin/node"); image.set_execfn("/usr/local/bin/node");
    image.add_argv(scenario == "canonical" ? "/usr/local/bin/node" : "node"); image.add_argv("--version");
    if (scenario == "binary") image.set_binary_path("/tmp/node");
    if (scenario == "execfn") image.set_execfn("/tmp/node");
    if (scenario == "argv0") image.set_argv(0, "npm");
    if (scenario == "code") image.set_argv(1, "-e");
    if (scenario == "alias") image.set_argv(1, "-v");
    if (scenario == "extra-arg") image.add_argv("/tmp/artifact.js");
    if (scenario == "short-argv") image.mutable_argv()->RemoveLast();
    if (scenario == "parent-class") state.groups[99].current_image_class = ProcessClass::kUV;
    if (scenario == "parent-role") state.groups[99].role = ProcessState::Role::kArtifact;
    if (scenario == "parent-start") state.groups[99].start_time_ns++;
    if (scenario == "parent-expected") state.expected_groups.erase(99);
    if (scenario == "parent-generation") state.expected_groups[99].start_time_ns++;
    if (scenario == "parent-expected-class") state.expected_groups[99].process_class = ProcessClass::kUnknown;
    if (scenario == "parent-network") state.groups[99].trusted_control_network_active = false;
    if (scenario == "parent-eligible") state.groups[99].root_eligible = true;
    if (scenario == "parent-unconsumed") state.groups[99].root_consumed = false;
    if (scenario == "parent-pending") state.groups[99].handoff_target_pending = true;
    if (scenario == "parent-clone") state.groups[99].provenance = ProcessState::Provenance::kCloneChild;
    if (scenario == "oci-origin") child->set_is_exec_session(false);
    if (scenario == "wrong-parent") child->set_parent_thread_group_id(98);
    if (scenario == "direct-root") { child->set_parent_thread_group_id(0); state.groups[100].provenance = ProcessState::Provenance::kDirectExecRoot; }
    if (scenario == "missing-clone") state.groups[100].pnpm_node_query_candidate = false;
    if (scenario == "artifact-child") state.groups[100].role = ProcessState::Role::kArtifact;
    if (scenario == "child-network") state.groups[100].trusted_control_network_active = true;
    if (scenario == "child-pending") state.groups[100].launch_target_pending = true;
    if (scenario == "unsealed") topology.sealed = false;
    if (scenario == "missing-snapshot") topology.snapshot_seen = false;
    if (scenario == "writable-root") topology.expected[0].read_only = false;
    if (scenario == "image-shadow") topology.anchors.emplace(5, MountAnchor{5, "/usr/local", "workspace"});
    if (scenario == "runtime-shadow") topology.anchors.emplace(5, MountAnchor{5, kPNPMToolPath, "workspace"});
    const auto* selected = scenario == "no-topology" ? nullptr : &topology;
    const char* profile = scenario == "other-profile" ? kProfileUVResolver : kProfilePNPMResolver;
    ProfileRegistration registration;
    const bool parsed = image.SerializeToString(&payload) && ParseSentryProcessAndClassify(
        payload.data(), payload.size(), output[0], &id, profile, &registration, &state, &reason, selected);
    const bool normal = scenario == "normal" || scenario == "canonical";
    const auto& group = state.groups[100];
    const bool query = parsed && group.pnpm_node_query_active && group.role == ProcessState::Role::kArtifact &&
        group.current_image_class == ProcessClass::kNode && !IsTrustedControlNetwork(*child,state) &&
        state.expected_groups.find(100) == state.expected_groups.end();
    if (query != normal) return false;
    if (!normal) {
      DrainDatagram(output[1]);
      // A rejected PNPM candidate cannot fall back to a broad CONTROL child.
      return !parsed || scenario == "other-profile" ||
          (state.groups[100].role == ProcessState::Role::kArtifact && state.expected_groups.find(100)==state.expected_groups.end());
    }
    if (!ExpectRecordExact(output[1], kFirstID, "process-exec-expected")) return false;
    // Neither a descendant nor a second identical exec gets a fresh query.
    *clone.mutable_context_data() = *child;
    clone.set_created_thread_group_id(101); clone.set_created_thread_start_time_ns(1010);
    if (!clone.SerializeToString(&payload) || !ParseSentryClone(payload.data(), payload.size(), output[0], &id, &state, &reason)) return false;
    if (state.groups[101].pnpm_node_query_candidate || state.groups[101].pnpm_node_query_active) return false;
    auto descendant = image;
    descendant.mutable_context_data()->set_thread_group_id(101);
    descendant.mutable_context_data()->set_thread_group_start_time_ns(1010);
    descendant.mutable_context_data()->set_parent_thread_group_id(100);
    if (!descendant.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,selected) ||
        !ExpectUnexpectedProcessRecord(output[1],kFirstID,"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP") || IsTrustedControlNetwork(descendant.context_data(),state)) return false;
    if (!image.SerializeToString(&payload) || !ParseSentryProcessAndClassify(payload.data(),payload.size(),output[0],&id,profile,&registration,&state,&reason,selected) ||
        !ExpectUnexpectedProcessRecord(output[1],kFirstID,"SENTRY_EXEC","OTHER","ARTIFACT_ROLE","ARTIFACT_GROUP") || IsTrustedControlNetwork(*child,state) || state.groups[100].pnpm_node_query_active) return false;
    return true;
  };
  bool passed = true;
  for (const char* scenario : {"normal","canonical","binary","execfn","argv0","code","alias","extra-arg","short-argv",
      "parent-class","parent-role","parent-start","parent-expected","parent-generation","parent-expected-class","parent-network",
      "parent-eligible","parent-unconsumed","parent-pending","parent-clone","oci-origin","wrong-parent","direct-root","artifact-child",
      "missing-clone","child-network","child-pending","unsealed","missing-snapshot","writable-root","image-shadow","runtime-shadow","no-topology","other-profile"}) {
    const bool result = run(scenario);
    if (!result) fprintf(stderr,"PNPM Node query boundary failed: %s\n",scenario);
    passed = result && passed;
  }
  close(output[0]); close(output[1]);
  return passed;
}

bool VerifyPNPMNodeQueryReadBoundary() {
  ProcessState state;
  gvisor::common::ContextData parent;
  parent.set_container_id(kFirstID); parent.set_thread_group_id(99); parent.set_thread_group_start_time_ns(990);
  parent.set_is_exec_session(true); parent.set_parent_thread_group_id(0);
  if (!RegisterGroup(&state,parent,ProcessState::Role::kControl,ProcessState::Provenance::kDirectExecRoot,false,true)) return false;
  state.groups[99].current_image_class=ProcessClass::kPNPM; state.groups[99].trusted_control_network_active=true;
  state.expected_groups[99]=ProcessState::ExpectedGroup{990,ProcessClass::kPNPM};
  auto child=parent; child.set_thread_group_id(100); child.set_thread_group_start_time_ns(1000); child.set_parent_thread_group_id(99);
  if (!RegisterGroup(&state,child,ProcessState::Role::kArtifact,ProcessState::Provenance::kCloneChild,false,true)) return false;
  auto& query=state.groups[100]; query.clone_creator_group_id=99; query.clone_creator_group_start_time_ns=990;
  query.current_image_class=ProcessClass::kNode; query.pnpm_node_query_active=true;
  const auto baseline=state;
  gvisor::syscall::Open open; *open.mutable_context_data()=child; open.set_flags(557056);
  MountAnchor anchor{1,"/","oci-root"};
  auto allowed=[&](){return IsPinnedPNPMNodeQueryRead(open.context_data(),state,open.pathname(),open.flags(),kProfilePNPMResolver,&anchor);};
  for (const char* path : {"/etc/ld.so.cache","/usr/lib/x86_64-linux-gnu/libstdc++.so.6.0.30","/usr/lib/x86_64-linux-gnu/libgcc_s.so.1","/usr/lib/x86_64-linux-gnu/libm.so.6"}) {
    open.set_pathname(path);
    if (!allowed() || ClassifyFilesystemOpen(open,state,kProfilePNPMResolver,&anchor)!=FilesystemClass::kRuntimeRoot) return false;
  }
  for (const char* path : {"/usr/lib/x86_64-linux-gnu/libcrypto.so.3","/usr/lib/x86_64-linux-gnu/libstdc++.so.6.evil","/tmp/libc.so.6","/proc/100/maps","/etc/passwd","/etc/ssl/openssl.cnf","/usr/local/lib/node_modules/npm/bin/npm-cli.js"}) {
    open.set_pathname(path); if (allowed()) return false;
  }
  open.set_pathname("/etc/ld.so.cache");
  for (uint64_t flags : {uint64_t{0},kOpenLargefile,kOpenWriteOnly,kOpenReadWrite,kOpenCreate,kOpenTruncate,kOpenAppend}) {open.set_flags(flags);if(allowed())return false;}
  open.set_flags(557056);
  for (const auto& wrong : {MountAnchor{2,"/etc","oci-root"},MountAnchor{2,"/","workspace"},MountAnchor{2,"/","system"}}) {anchor=wrong;if(allowed())return false;}
  anchor=MountAnchor{1,"/","oci-root"};
  for (const char* profile : {kProfileNPM,kProfileUVResolver,kProfileGitHub,kProfilePyPI,kProfileCargoResolver,kProfileGoResolver}) {
    if(IsPinnedPNPMNodeQueryRead(child,state,open.pathname(),open.flags(),profile,&anchor))return false;
  }
  for(int mutation=0;mutation<18;mutation++) {
    state=baseline;auto& group=state.groups[100];
    switch(mutation) {
      case 0:group.role=ProcessState::Role::kControl;break;
      case 1:group.pnpm_node_query_active=false;break;
      case 2:group.start_time_ns++;break;
      case 3:group.provenance=ProcessState::Provenance::kDirectExecRoot;break;
      case 4:group.trusted_control_network_active=true;break;
      case 5:group.current_image_class=ProcessClass::kUnknown;break;
      case 6:group.root_eligible=true;break;
      case 7:group.root_consumed=false;break;
      case 8:group.demotion_pending=true;break;
      case 9:group.launch_target_pending=true;break;
      case 10:group.handoff_target_pending=true;break;
      case 11:state.groups[99].start_time_ns++;break;
      case 12:state.expected_groups.erase(99);break;
      case 13:state.groups[99].role=ProcessState::Role::kArtifact;break;
      case 14:state.groups[99].trusted_control_network_active=false;break;
      case 15:state.expected_groups[99].process_class=ProcessClass::kUV;break;
      case 16:group.clone_creator_group_id=98;break;
      case 17:state.groups[99].current_image_class=ProcessClass::kUV;break;
    }
    if(allowed())return false;
  }
  return true;
}

bool VerifyUVRuntimeReadBoundary() {
  ProcessState state;
  gvisor::syscall::Open open;
  auto* context = open.mutable_context_data();
  context->set_container_id(kFirstID); context->set_thread_group_id(99);
  context->set_thread_group_start_time_ns(990); context->set_parent_thread_group_id(0);
  context->set_is_exec_session(true); context->set_process_name("renamed-uv");
  if (!RegisterGroup(&state, *context, ProcessState::Role::kControl, ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  state.groups[99].current_image_class = ProcessClass::kUV;
  state.groups[99].trusted_control_network_active = true;
  state.expected_groups[99] = ProcessState::ExpectedGroup{990, ProcessClass::kUV};
  const ProcessState baseline = state;
  open.set_pathname("/proc/sys/vm/overcommit_memory"); open.set_flags(557056);
  MountAnchor anchor{10, "/proc", "system"};
  auto allowed = [&](const char* profile = kProfileUVResolver) {
    return ClassifyFilesystemOpen(open, state, profile, &anchor) == FilesystemClass::kHelperOnly;
  };
  if (!allowed()) return false;
  for (const char* mount : {"/proc", "/sys/fs/cgroup/cpu", "/dev", "/", "/etc/hosts"}) {
    anchor.mountpoint = mount; anchor.mount_class = strcmp(mount, "/") == 0 ? "oci-root" : "system";
    for (const char* path : {"/proc/99/maps", "/proc/99/environ", "/proc/sys/vm/other", "/proc/self/cgroup",
        "/proc/1/cgroup", "/proc/100/cgroup", "/proc/1/mountinfo", "/proc/100/mountinfo",
        "/dev/zero", "/dev/urandom", "/sys/fs/cgroup/cpu/cpu.stat", "/usr/local/bin/python", "/usr/local/bin/pip", "/usr/local/lib/libpython3.14.so.1.0", "/usr/bin/dash", "/usr/local/bin/python3.14.evil",
        "/usr/lib/os-release.evil", "/etc/os-release", "/usr/lib/../lib/os-release",
        "/usr/local/sbin/evil", "/usr/local/sbin.evil", "/usr/local//sbin", "/usr/local/./sbin",
        "/usr/local/bin/../sbin", "/usr/bin/evil", "/bin", "/sbin", "/usr/local", "/usr",
        "/etc/hosts.evil", "/etc/resolv.conf", "/etc/../etc/hosts"}) {
      open.set_pathname(path); if (allowed()) return false;
    }
  }
  for (const char* path : {"/proc/sys/vm/overcommit_memory", "/proc/99/cgroup", "/proc/99/mountinfo",
      "/sys/fs/cgroup/cpu/cpu.cfs_quota_us", "/sys/fs/cgroup/cpu/cpu.cfs_period_us", "/dev/null", "/usr/local/bin/python3.14", "/usr/lib/os-release", "/etc/hosts",
      "/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin"}) {
    state = baseline;
    open.set_pathname(path);
    const bool image = strcmp(path, "/usr/local/bin/python3.14") == 0 || strcmp(path, "/usr/lib/os-release") == 0 ||
        strcmp(path, "/usr/local/sbin") == 0 || strcmp(path, "/usr/local/bin") == 0 ||
        strcmp(path, "/usr/sbin") == 0 || strcmp(path, "/usr/bin") == 0;
    const std::string mountpoint = image ? "/" : (strcmp(path, "/dev/null") == 0 ? "/dev" :
        (strcmp(path, "/etc/hosts") == 0 ? "/etc/hosts" : (HasPrefix(path, "/sys/") ? "/sys/fs/cgroup/cpu" : "/proc")));
    const std::string mount_class = image ? "oci-root" : "system";
    anchor.mountpoint = mountpoint; anchor.mount_class = mount_class;
    if (!allowed()) return false;
    context->set_parent_thread_group_id(1); if (allowed()) return false;
    context->set_parent_thread_group_id(0);
    context->set_is_exec_session(false); if (allowed()) return false;
    context->set_is_exec_session(true);
    for (uint64_t flags : {uint64_t{0}, kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenTruncate, kOpenAppend}) {
      open.set_flags(flags); if (allowed()) return false;
    }
    open.set_flags(557056);
    for (const char* profile : {kProfileNPM, kProfilePyPI, kProfilePyTorchCPU, kProfilePyTorchCU126,
        kProfilePyTorchCU130, kProfilePyTorchCU132, kProfileCargoResolver, kProfileGoResolver,
        kProfilePNPMResolver, kProfileBunResolver, kProfileBunProjectResolver}) {
      // Existing profiles have their own authorized semantics; the UV-specific
      // predicate must never supply them a new grant.
      if (IsPinnedUVRuntimeRead(*context, state, open.pathname(), open.flags(), profile, &anchor)) return false;
    }
    for (const char* mount : {"/", "/proc/sys", "/proc/99", "/sys/fs/cgroup", "/sys/fs/cgroup/cpu/x", "/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin"}) { anchor.mountpoint = mount; if (mount != mountpoint && allowed()) return false; }
    for (const char* root : {"/proc", "/sys/fs/cgroup/cpu", "/dev", "/etc/hosts"}) {
      anchor.mountpoint = root;
      if (allowed() != (anchor.mountpoint == mountpoint)) return false;
    }
    anchor.mountpoint = mountpoint; anchor.mount_class = "workspace"; if (allowed()) return false;
    anchor.mount_class = image ? "system" : "oci-root"; if (allowed()) return false;
    anchor.mount_class = mount_class;
    for (int scenario = 0; scenario < 11; ++scenario) {
      state = baseline; auto& group = state.groups[99];
      switch (scenario) {
        case 0: group.role = ProcessState::Role::kArtifact; break;
        case 1: group.provenance = ProcessState::Provenance::kCloneChild; break;
        case 2: group.current_image_class = ProcessClass::kUnknown; break;
        case 3: group.root_eligible = true; break;
        case 4: group.root_consumed = false; break;
        case 5: group.trusted_control_network_active = false; break;
        case 6: group.demotion_pending = true; break;
        case 7: group.launch_target_pending = true; break;
        case 8: group.handoff_target_pending = true; break;
        case 9: group.start_time_ns++; break;
        case 10: state.expected_groups[99].process_class = ProcessClass::kUnknown; break;
      }
      if (allowed()) return false;
    }
  }
  return true;
}

bool VerifyPNPMRuntimeReadBoundary() {
  ProcessState state;
  gvisor::syscall::Open open;
  auto* context = open.mutable_context_data();
  context->set_container_id(kFirstID); context->set_thread_group_id(99);
  context->set_thread_group_start_time_ns(990); context->set_parent_thread_group_id(0);
  context->set_is_exec_session(true); context->set_process_name("renamed-pnpm");
  if (!RegisterGroup(&state, *context, ProcessState::Role::kControl, ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  state.groups[99].current_image_class = ProcessClass::kPNPM;
  state.groups[99].trusted_control_network_active = true;
  state.expected_groups[99] = ProcessState::ExpectedGroup{990, ProcessClass::kPNPM};
  const ProcessState baseline = state;
  struct ReadCase { const char* path; const char* mount; const char* mount_class; };
  auto run = [&](const ReadCase& read) {
  state = baseline;
  const char* exact_path = read.path;
  const char* exact_mount = read.mount;
  const char* exact_class = read.mount_class;
  open.set_pathname(exact_path); open.set_flags(557056);
  MountAnchor anchor{10, exact_mount, exact_class};
  auto allowed = [&]() { return IsPinnedPNPMRuntimeRead(*context, state, open.pathname(), open.flags(), kProfilePNPMResolver, &anchor); };
  if (!allowed() || ClassifyFilesystemOpen(open, state, kProfilePNPMResolver, &anchor) != FilesystemClass::kHelperOnly) return false;
  context->set_parent_thread_group_id(1); if (allowed()) return false;
  context->set_parent_thread_group_id(0); context->set_is_exec_session(false); if (allowed()) return false;
  context->set_is_exec_session(true);
  for (uint64_t flags : {uint64_t{0}, kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenTruncate, kOpenAppend}) { open.set_flags(flags); if (allowed()) return false; }
  open.set_flags(557056);
  for (const char* profile : {kProfileUVResolver, kProfileNPM, kProfileGitHub, kProfilePyPI, kProfilePyTorchCPU, kProfilePyTorchCU126, kProfilePyTorchCU130, kProfilePyTorchCU132, kProfileCargoResolver, kProfileGoResolver}) {
    if (IsPinnedPNPMRuntimeRead(*context, state, open.pathname(), open.flags(), profile, &anchor)) return false;
  }
  for (const char* path : {"/lib/x86_64-linux-gnu/librt.so.1", "/usr/lib/x86_64-linux-gnu/librt.so.1.evil", "/usr/lib/x86_64-linux-gnu/../x86_64-linux-gnu/librt.so.1", "/proc/sys/vm/overcommit_memory.evil", "/proc/sys/vm/../vm/overcommit_memory", "/proc/sys/kernel/hostname", "/proc/sys/kernel/osrelease.evil", "/proc/sys/kernel/../kernel/osrelease", "/dev/random", "/dev/zero", "/dev/urandom.evil", "/dev/../dev/urandom", "/haa-runtime/uv", "/haa-runtime/pnpm.evil", "/haa-runtime/./pnpm", "/sys/devices/system/cpu/possible", "/sys/devices/system/cpu/online.evil", "/sys/devices/system/cpu/../cpu/online", "/sys/fs/cgroup/cpu/cpu.stat", "/sys/fs/cgroup/cpu/cpu.cfs_quota_us.evil", "/sys/fs/cgroup/cpu/../cpu/cpu.cfs_period_us", "/proc/self/mountinfo", "/proc/1/mountinfo", "/proc/100/mountinfo", "/proc/99/mountinfo.evil", "/proc/self/cgroup", "/proc/1/cgroup", "/proc/100/cgroup", "/proc/99/cgroup.evil", "/etc/hosts", "/etc/gai.conf.evil", "/etc//gai.conf", "/etc/../etc/gai.conf", "/tmp/gai.conf", "/etc/hosts.evil", "/etc//hosts", "/etc/../etc/hosts", "/tmp/hosts", "/etc/resolv.conf.evil", "/etc//resolv.conf", "/etc/../etc/resolv.conf", "/tmp/resolv.conf", "/etc/host.conf.evil", "/etc/../etc/host.conf", "/etc//host.conf", "/tmp/host.conf", "/etc/nsswitch.conf.evil", "/etc/../etc/nsswitch.conf", "/etc//nsswitch.conf", "/tmp/nsswitch.conf", "/proc/self/maps", "/proc/1/maps", "/proc/100/maps", "/proc/99/maps.evil", "/tmp/librt.so.1"}) {
    open.set_pathname(path); if (strcmp(path,exact_path)!=0 && allowed()) return false;
  }
  open.set_pathname(exact_path);
  for (const char* mount : {"/", "/etc/resolv.conf", "/etc/hosts", "/usr/lib", "/usr/lib/x86_64-linux-gnu", "/haa-runtime", "/proc", "/sys/fs/cgroup", "/sys/fs/cgroup/cpu/other", "/sys/devices/system/cpu"}) { anchor.mountpoint = mount; if (strcmp(mount,exact_mount)!=0 && allowed()) return false; }
  anchor.mountpoint = exact_mount;
  for (const char* mount_class : {"workspace", "system", "helper", "oci-root"}) { anchor.mount_class = mount_class; if (strcmp(mount_class,exact_class)!=0 && allowed()) return false; }
  anchor.mount_class = exact_class;
  for (int scenario = 0; scenario < 11; ++scenario) {
    state = baseline; auto& group = state.groups[99];
    switch (scenario) {
      case 0: group.role = ProcessState::Role::kArtifact; break;
      case 1: group.provenance = ProcessState::Provenance::kCloneChild; break;
      case 2: group.current_image_class = ProcessClass::kUnknown; break;
      case 3: group.root_eligible = true; break;
      case 4: group.root_consumed = false; break;
      case 5: group.trusted_control_network_active = false; break;
      case 6: group.demotion_pending = true; break;
      case 7: group.launch_target_pending = true; break;
      case 8: group.handoff_target_pending = true; break;
      case 9: group.start_time_ns++; break;
      case 10: state.expected_groups[99].process_class = ProcessClass::kUnknown; break;
    }
    if (allowed()) return false;
  }
  return true;
  };
  const ReadCase reads[] = {
    {"/usr/lib/x86_64-linux-gnu/librt.so.1", "/", "oci-root"},
    {"/etc/nsswitch.conf", "/", "oci-root"},
    {"/etc/host.conf", "/", "oci-root"},
    {"/etc/gai.conf", "/", "oci-root"},
    {"/etc/resolv.conf", "/etc/resolv.conf", "system"},
    {"/etc/hosts", "/etc/hosts", "system"},
    {"/proc/99/maps", "/proc", "system"},
    {"/proc/sys/vm/overcommit_memory", "/proc", "system"},
    {"/proc/99/cgroup", "/proc", "system"},
    {"/proc/99/mountinfo", "/proc", "system"},
    {"/sys/fs/cgroup/cpu/cpu.cfs_quota_us", "/sys/fs/cgroup/cpu", "system"},
    {"/sys/fs/cgroup/cpu/cpu.cfs_period_us", "/sys/fs/cgroup/cpu", "system"},
    {"/sys/devices/system/cpu/online", "/sys", "system"},
    {kPNPMToolPath, "/haa-runtime", "helper"},
    {"/dev/urandom", "/dev", "system"},
    {"/dev/null", "/dev", "system"},
    {"/proc/sys/kernel/osrelease", "/proc", "system"},
  };
  for (const auto& read : reads) if (!run(read)) return false;
  return true;
}

bool VerifyBunQueryReadBoundary(const char* selected_profile = kProfileBunResolver) {
  ProcessState state;
  gvisor::syscall::Open open;
  auto* context = open.mutable_context_data();
  context->set_container_id(kFirstID); context->set_thread_group_id(99);
  context->set_thread_group_start_time_ns(990); context->set_parent_thread_group_id(0);
  context->set_is_exec_session(true); context->set_process_name("renamed-bun");
  if (!RegisterGroup(&state, *context, ProcessState::Role::kControl, ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
  state.groups[99].current_image_class = ProcessClass::kBun;
  state.groups[99].trusted_control_network_active = true;
  state.expected_groups[99] = ProcessState::ExpectedGroup{990, ProcessClass::kBun};
  const ProcessState baseline = state;
  struct ReadCase { const char* path; const char* mount; const char* mount_class; uint64_t flags; };
  auto run = [&](const ReadCase& read) {
  state = baseline;
  const char* exact_path = read.path;
  const char* exact_mount = read.mount;
  const char* exact_class = read.mount_class;
  open.set_pathname(exact_path); open.set_flags(read.flags);
  MountAnchor anchor{10, exact_mount, exact_class};
  auto allowed = [&]() { return IsPinnedBunRuntimeRead(*context, state, open.pathname(), open.flags(), selected_profile, &anchor); };
  if (!allowed() || ClassifyFilesystemOpen(open, state, selected_profile, &anchor) != FilesystemClass::kHelperOnly) return false;
  if ((strcmp(exact_path,"/sys/devices/system/cpu/online")==0 || strcmp(exact_path,"/proc/99/cgroup")==0 || strcmp(exact_path,"/usr/share/zoneinfo/Etc/UTC")==0 || strcmp(exact_path,"/etc/host.conf")==0 || strcmp(exact_path,"/etc/nsswitch.conf")==0 || strcmp(exact_path,"/etc/gai.conf")==0 || strcmp(exact_path,"/etc/resolv.conf")==0 || strcmp(exact_path,"/etc/hosts")==0) && IsPinnedBunRuntimeRead(*context,state,open.pathname(),open.flags(),kProfileBunResolver,&anchor)) return false;
  context->set_parent_thread_group_id(1); if (allowed()) return false;
  context->set_parent_thread_group_id(0); context->set_is_exec_session(false); if (allowed()) return false;
  context->set_is_exec_session(true);
  for (uint64_t flags : {uint64_t{0}, uint64_t{32768}, uint64_t{557056}, kOpenWriteOnly, kOpenReadWrite, kOpenCreate, kOpenTruncate, kOpenAppend}) { if (flags == read.flags) continue; open.set_flags(flags); if (allowed()) return false; }
  open.set_flags(read.flags);
  for (const char* profile : {kProfilePNPMResolver, kProfileUVResolver, kProfileNPM, kProfileGitHub, kProfilePyPI, kProfilePyTorchCPU, kProfilePyTorchCU126, kProfilePyTorchCU130, kProfilePyTorchCU132, kProfileCargoResolver, kProfileGoResolver}) {
    if (IsPinnedBunRuntimeRead(*context, state, open.pathname(), open.flags(), profile, &anchor)) return false;
  }
  for (const char* path : {"/lib/x86_64-linux-gnu/librt.so.1", "/usr/lib/x86_64-linux-gnu/librt.so.1.evil", "/usr/lib/x86_64-linux-gnu/../x86_64-linux-gnu/librt.so.1", "/proc/sys/vm/overcommit_memory.evil", "/proc/sys/vm/../vm/overcommit_memory", "/proc/sys/kernel/hostname", "/proc/sys/kernel/osrelease.evil", "/proc/sys/kernel/../kernel/osrelease", "/dev/random", "/dev/zero", "/dev/urandom.evil", "/dev/../dev/urandom", "/haa-runtime/uv", "/haa-runtime/pnpm.evil", "/haa-runtime/./pnpm", "/sys/devices/system/cpu/possible", "/sys/devices/system/cpu/online.evil", "/sys/devices/system/cpu/../cpu/online", "/sys/fs/cgroup/cpu/cpu.stat", "/sys/fs/cgroup/cpu/cpu.cfs_quota_us.evil", "/sys/fs/cgroup/cpu/../cpu/cpu.cfs_period_us", "/proc/self/mountinfo", "/proc/1/mountinfo", "/proc/100/mountinfo", "/proc/99/mountinfo.evil", "/proc/self/cgroup", "/proc/1/cgroup", "/proc/100/cgroup", "/proc/99/cgroup.evil", "/etc/hosts", "/etc/gai.conf.evil", "/etc//gai.conf", "/etc/../etc/gai.conf", "/tmp/gai.conf", "/etc/hosts.evil", "/etc//hosts", "/etc/../etc/hosts", "/tmp/hosts", "/etc/resolv.conf.evil", "/etc//resolv.conf", "/etc/../etc/resolv.conf", "/tmp/resolv.conf", "/etc/host.conf.evil", "/etc/../etc/host.conf", "/etc//host.conf", "/tmp/host.conf", "/etc/nsswitch.conf.evil", "/etc/../etc/nsswitch.conf", "/etc//nsswitch.conf", "/tmp/nsswitch.conf", "/proc/self/maps", "/proc/1/maps", "/proc/100/maps", "/proc/99/maps.evil", "/tmp/librt.so.1", "/usr/share/zoneinfo/Etc/UTC.evil", "/usr/share/zoneinfo/Etc/../Etc/UTC", "/usr/share/zoneinfo/UTC", "/etc/localtime"}) {
    open.set_pathname(path); if (strcmp(path,exact_path)!=0 && allowed()) return false;
  }
  open.set_pathname(exact_path);
  for (const char* mount : {"/", "/etc/resolv.conf", "/etc/hosts", "/usr/lib", "/usr/lib/x86_64-linux-gnu", "/haa-runtime", "/proc", "/sys/fs/cgroup", "/sys/fs/cgroup/cpu/other", "/sys/devices/system/cpu"}) { anchor.mountpoint = mount; if (strcmp(mount,exact_mount)!=0 && allowed()) return false; }
  anchor.mountpoint = exact_mount;
  for (const char* mount_class : {"workspace", "system", "helper", "oci-root"}) { anchor.mount_class = mount_class; if (strcmp(mount_class,exact_class)!=0 && allowed()) return false; }
  anchor.mount_class = exact_class;
  for (int scenario = 0; scenario < 11; ++scenario) {
    state = baseline; auto& group = state.groups[99];
    switch (scenario) {
      case 0: group.role = ProcessState::Role::kArtifact; break;
      case 1: group.provenance = ProcessState::Provenance::kCloneChild; break;
      case 2: group.current_image_class = ProcessClass::kPNPM; break;
      case 3: group.root_eligible = true; break;
      case 4: group.root_consumed = false; break;
      case 5: group.trusted_control_network_active = false; break;
      case 6: group.demotion_pending = true; break;
      case 7: group.launch_target_pending = true; break;
      case 8: group.handoff_target_pending = true; break;
      case 9: group.start_time_ns++; break;
      case 10: state.expected_groups[99].process_class = ProcessClass::kUnknown; break;
    }
    if (allowed()) return false;
  }
  return true;
  };
  std::vector<ReadCase> reads = {{"/proc/sys/vm/overcommit_memory", "/proc", "system", 32768}, {"/proc/99/maps", "/proc", "system", 557056}};
  if (strcmp(selected_profile,kProfileBunProjectResolver)==0) { reads.push_back({"/sys/devices/system/cpu/online", "/sys", "system", 557056}); reads.push_back({"/proc/99/cgroup", "/proc", "system", 557056}); reads.push_back({"/usr/share/zoneinfo/Etc/UTC", "/", "oci-root", 557056}); reads.push_back({"/etc/host.conf", "/", "oci-root", 557056}); reads.push_back({"/etc/nsswitch.conf", "/", "oci-root", 557056}); reads.push_back({"/etc/gai.conf", "/", "oci-root", 557056}); reads.push_back({"/etc/resolv.conf", "/etc/resolv.conf", "system", 557056}); reads.push_back({"/etc/hosts", "/etc/hosts", "system", 557056}); }
  for (const auto& read : reads) if (!run(read)) return false;
  return true;
}

bool VerifyBunArtifactRuntimeReadBoundary() {
  ProcessState state;
  gvisor::common::ContextData parent;
  parent.set_container_id(kFirstID); parent.set_thread_group_id(98); parent.set_thread_group_start_time_ns(980);
  if (!RegisterGroup(&state, parent, ProcessState::Role::kArtifact, ProcessState::Provenance::kCloneChild, false, true)) return false;
  gvisor::syscall::Open open;
  auto* context = open.mutable_context_data();
  *context = parent; context->set_thread_group_id(99); context->set_thread_group_start_time_ns(990); context->set_parent_thread_group_id(98);
  if (!RegisterGroup(&state, *context, ProcessState::Role::kArtifact, ProcessState::Provenance::kCloneChild, false, true)) return false;
  state.groups[99].current_image_class = ProcessClass::kBun;
  state.groups[99].clone_creator_group_id = 98; state.groups[99].clone_creator_group_start_time_ns = 980;
  const ProcessState baseline = state;
  struct Read { const char* path; const char* mount; const char* kind; uint64_t flags; };
  for (const auto& read : std::vector<Read>{{"/","/","oci-root",622592},{"/etc/ld.so.cache","/","oci-root",557056},
       {"/usr/lib/x86_64-linux-gnu/libc.so.6","/","oci-root",557056},
       {"/proc/sys/vm/overcommit_memory","/proc","system",32768},
       {"/proc/sys/vm/mmap_min_addr","/proc","system",32768},
       {"/proc/99/statm","/proc","system",32768},
       {"/sys/fs/cgroup/memory/memory.soft_limit_in_bytes","/sys/fs/cgroup/memory","system",557056},
       {"/sys/fs/cgroup/memory/memory.limit_in_bytes","/sys/fs/cgroup/memory","system",557056},
       {"/proc/99/maps","/proc","system",557056},{"/proc/99/cgroup","/proc","system",557056},
       {"/sys/devices/system/cpu/online","/sys","system",557056},
       {"/usr/share/zoneinfo/Etc/UTC","/","oci-root",557056}}) {
    state = baseline; open.set_pathname(read.path); open.set_flags(read.flags);
    MountAnchor anchor{10,read.mount,read.kind};
    auto allowed = [&]() { return IsPinnedBunArtifactRuntimeRead(*context,state,open.pathname(),open.flags(),kProfileNPM,&anchor); };
    if (!allowed() || ClassifyFilesystemOpen(open,state,kProfileNPM,&anchor)!=FilesystemClass::kHelperOnly || IsTrustedControlNetwork(*context,state)) return false;
    for (uint64_t flags : {uint64_t{0},uint64_t{32768},uint64_t{557056},kOpenWriteOnly,kOpenReadWrite,kOpenCreate,kOpenAppend}) {
      if (flags==read.flags) continue; open.set_flags(flags); if (allowed()) return false;
    }
    open.set_flags(read.flags);
    for (const char* path : {"/proc/1/statm","/proc/100/statm","/proc/self/statm","/proc/100/maps","/proc/1/cgroup",
         "/sys/fs/cgroup/memory/foreign/memory.limit_in_bytes","/etc/passwd","/etc/hosts","/dev/random"}) {
      open.set_pathname(path); if (allowed()) return false;
    }
    open.set_pathname(read.path);
    for (const char* profile : {kProfileBunResolver,kProfileBunProjectResolver,kProfilePNPMResolver,kProfileUVResolver,kProfileGitHub,kProfilePyPI,kProfileGoResolver}) {
      if (IsPinnedBunArtifactRuntimeRead(*context,state,open.pathname(),open.flags(),profile,&anchor)) return false;
    }
    open.set_pathname(std::string(read.path)+".evil"); if (allowed()) return false; open.set_pathname(read.path);
    anchor.mount_class="workspace"; if (allowed()) return false; anchor.mount_class=read.kind;
    anchor.mountpoint="/foreign"; if (allowed()) return false; anchor.mountpoint=read.mount;
    for (int scenario=0;scenario<13;++scenario) {
      state=baseline; auto& group=state.groups[99];
      switch(scenario) {
        case 0: group.role=ProcessState::Role::kControl; break;
        case 1: group.provenance=ProcessState::Provenance::kDirectExecRoot; break;
        case 2: group.current_image_class=ProcessClass::kNode; break;
        case 3: group.root_eligible=true; break;
        case 4: group.root_consumed=false; break;
        case 5: group.trusted_control_network_active=true; break;
        case 6: group.demotion_pending=true; break;
        case 7: group.launch_target_pending=true; break;
        case 8: group.handoff_target_pending=true; break;
        case 9: group.start_time_ns++; break;
        case 10: state.groups[98].start_time_ns++; break;
        case 11: group.clone_creator_group_id=97; break;
        case 12: state.expected_groups[99]=ProcessState::ExpectedGroup{990,ProcessClass::kBun}; break;
      }
      if (allowed()) return false;
    }
  }
  return true;
}

bool VerifyBunArtifactRuntimeImageAndCommandBoundary() {
  auto topology=PNPMTargetLifecycleTestTopology(true);
  topology.expected[2].filesystem_type="9p"; topology.expected[2].nosuid=false;
  if (!IsBunArtifactRuntimeImage(&topology,kBunToolPath) || IsBunArtifactRuntimeImage(nullptr,kBunToolPath) ||
      IsBunArtifactRuntimeImage(&topology,"/tmp/bun") || ProcessClassForPath(kBunToolPath,kProfileNPM)!=ProcessClass::kBun) return false;
  gvisor::sentry::ExecveInfo boundary;
  boundary.set_binary_path(kBoundaryHelperPath); boundary.set_execfn(kBoundaryHelperPath);
  for (const auto& value : {std::string(kBoundaryHelperPath),std::string(kLaunchMode),TestAdmissionNonce(350,3500),
       std::string("/bin/sh"),std::string("-ceu"),std::string(kBunRuntimeTargetLifecycleCommand),
       std::string("helox-bun-target"),std::string(kBunTargetProjectMount)+"/project/node_modules/pkg"}) boundary.add_argv(value);
  if (!IsExactBunTargetLifecycleBoundary(boundary,kProfileNPM,&topology)) return false;
  const auto ordinary=PNPMTargetLifecycleTestTopology(true);
  if (IsExactBunTargetLifecycleBoundary(boundary,kProfileNPM,&ordinary)) return false;
  for(int scenario=0;scenario<9;++scenario) {
    auto changed=topology;
    switch(scenario) {
      case 0: changed.sealed=false; break;
      case 1: changed.snapshot_seen=false; break;
      case 2: changed.expected[2].filesystem_type="tmpfs"; break;
      case 3: changed.expected[0].read_only=false; break;
      case 4: changed.expected.back().filesystem_type="tmpfs"; break;
      case 5: changed.expected.back().noexec=true; break;
      case 6: changed.anchors.erase(4); break;
      case 7: changed.anchors.emplace(5,MountAnchor{5,kBunToolPath,"workspace"}); break;
      case 8: changed.expected.back().mountpoint="/foreign"; break;
    }
    if(IsBunArtifactRuntimeImage(&changed,kBunToolPath) || IsExactBunTargetLifecycleBoundary(boundary,kProfileNPM,&changed))return false;
  }
  for(int i=0;i<boundary.argv_size();++i){auto changed=boundary;changed.set_argv(i,"foreign");if(IsExactBunTargetLifecycleBoundary(changed,kProfileNPM,&topology))return false;}
  return true;
}

bool VerifyBunProjectLockInstallShapeBoundary() {
  gvisor::sentry::ExecveInfo message;
  message.set_binary_path(kBunToolPath); message.set_execfn(kBunToolPath);
  for (const char* argument : {"/haa-runtime/bun", "--cwd=/tmp/haa-bun-input/project", "install", "--lockfile-only", "--ignore-scripts", "--no-env-file", "--registry=https://registry.npmjs.org/", "--cache-dir=/tmp/haa-bun-input/cache", "--backend=copyfile", "--linker=hoisted", "--save-text-lockfile", "--no-progress", "--no-summary"}) message.add_argv(argument);
  if (!IsExactBunProjectLockInstall(message) || IsExactBunQuery(message)) return false;
  for (int i=0; i<message.argv_size(); ++i) {
    auto changed=message; changed.set_argv(i,"foreign");
    if (IsExactBunProjectLockInstall(changed)) return false;
  }
  for (const char* path : {"bun", "/tmp/bun", "/haa-runtime/./bun", "/haa-runtime/pnpm"}) {
    auto changed=message; changed.set_binary_path(path); if (IsExactBunProjectLockInstall(changed)) return false;
    changed=message; changed.set_execfn(path); if (IsExactBunProjectLockInstall(changed)) return false;
  }
  auto extra=message; extra.add_argv("--trust"); if (IsExactBunProjectLockInstall(extra)) return false;
  auto missing=message; missing.mutable_argv()->RemoveLast(); if (IsExactBunProjectLockInstall(missing)) return false;
  auto query=message; query.clear_argv(); query.add_argv(kBunToolPath); query.add_argv("--version");
  if (!IsExactBunQuery(query) || IsExactBunProjectLockInstall(query)) return false;
  auto frozen=message; frozen.add_argv("--frozen-lockfile");
  if (!IsExactBunProjectLockInstall(frozen)) return false;
  for (const bool seed : {false,true}) {
    auto prepared=message;
    if (seed) prepared.set_argv(1,"--cwd=/tmp/haa-bun-input/seed");
    else prepared.set_argv(3,"--offline");
    prepared.add_argv(seed ? "--offline" : "--frozen-lockfile");
    if (!IsExactBunProjectLockInstall(prepared) || IsExactBunQuery(prepared)) return false;
    for (int i=0;i<prepared.argv_size();++i) {
      auto changed=prepared; changed.set_argv(i,"foreign");
      if (IsExactBunProjectLockInstall(changed)) return false;
    }
    for (const char* intent : {"add","update","run","x"}) {
      auto changed=prepared; changed.set_argv(2,intent); if (IsExactBunProjectLockInstall(changed)) return false;
    }
    auto extra=prepared; extra.add_argv("--trust"); if (IsExactBunProjectLockInstall(extra)) return false;
    auto missing=prepared; missing.mutable_argv()->RemoveLast(); if (IsExactBunProjectLockInstall(missing)) return false;
  }
  for (const char* intent : {"add", "update"}) {
    auto selected=message; selected.set_argv(2,intent);
    if (IsExactBunProjectLockInstall(selected) != (strcmp(intent,"update")==0)) return false;
    selected.add_argv("--"); selected.add_argv(strcmp(intent,"add")==0 ? "@scope/probe@^1.0.0" : "@scope/probe");
    if (!IsExactBunProjectLockInstall(selected) || IsExactBunQuery(selected)) return false;
    for (const char* bad : {"--eval", "../escape", "/tmp/pkg", "file:pkg", "https://evil.test/pkg", "npm:alias", "probe\nnext", "probe@", "@scope/", "UPPER"}) {
      auto changed=selected; changed.set_argv(14,bad);
      if (IsExactBunProjectLockInstall(changed)) return false;
    }
    if (strcmp(intent,"update")==0) { auto changed=selected; changed.set_argv(14,"probe@1.0.0"); if (IsExactBunProjectLockInstall(changed)) return false; }
    for (int i=0;i<14;++i) { auto changed=selected; changed.set_argv(i,"foreign"); if (IsExactBunProjectLockInstall(changed)) return false; }
    auto extra=selected; extra.add_argv("--trust"); if (IsExactBunProjectLockInstall(extra)) return false;
  }
  return true;
}

bool VerifyPrivateNativeRuntimeAdmissionBoundary(bool pnpm, bool bun = false, bool bun_project = false, const char* bun_intent = "install", bool frozen = false, bool selector = false, const char* bun_preparation = nullptr) {
  const char* profile = bun ? (bun_project ? kProfileBunProjectResolver : kProfileBunResolver) : (pnpm ? kProfilePNPMResolver : kProfileUVResolver);
  const char* tool = bun ? kBunToolPath : (pnpm ? kPNPMToolPath : kUVToolPath);
  const ProcessClass tool_class = bun ? ProcessClass::kBun : (pnpm ? ProcessClass::kPNPM : ProcessClass::kUV);
  if (IsPythonProfile(profile) ||
      ProcessClassForPath(tool, profile) != tool_class ||
      ProcessClassForPath("uv", profile) != ProcessClass::kUnknown ||
      ProcessClassForPath("/tmp/uv", profile) != ProcessClass::kUnknown ||
      ProcessClassForPath("pnpm", profile) != ProcessClass::kUnknown ||
      ProcessClassForPath("/tmp/pnpm", profile) != ProcessClass::kUnknown ||
      ProcessClassForPath(tool, kProfilePyPI) != ProcessClass::kUnknown ||
      ProcessClassForPath("/usr/local/bin/pip", profile) != ProcessClass::kUnknown)
    return false;
  if (ProcessClassForPath(tool, pnpm ? kProfileUVResolver : kProfilePNPMResolver) != ProcessClass::kUnknown ||
      ProcessClassForPath(tool, kProfileGitHub) != ProcessClass::kUnknown) return false;
  int output[2];
  if (socketpair(AF_UNIX, SOCK_DGRAM, 0, output) != 0) return false;
  auto run = [&](const char* scenario) {
    TopologyState topology;
    topology.namespace_id = 1; topology.snapshot_seen = true; topology.sealed = true;
    topology.expected.push_back(ExpectedMount{"/haa-runtime", "helper", "/", "9p", false, false, false, false});
    topology.anchors.emplace(1, MountAnchor{1, "/", "oci-root"});
    topology.anchors.emplace(2, MountAnchor{2, "/haa-runtime", "helper"});
    if (strcmp(scenario, "unsealed") == 0) topology.sealed = false;
    if (strcmp(scenario, "missing-snapshot") == 0) topology.snapshot_seen = false;
    if (strcmp(scenario, "namespace") == 0) topology.namespace_id = 0;
    if (strcmp(scenario, "wrong-class") == 0) topology.expected[0].mount_class = "workspace";
    if (strcmp(scenario, "wrong-fs") == 0) topology.expected[0].filesystem_type = "tmpfs";
    if (strcmp(scenario, "noexec") == 0) topology.expected[0].noexec = true;
    if (strcmp(scenario, "suid") == 0) topology.expected[0].nosuid = true;
    if (strcmp(scenario, "missing-anchor") == 0) topology.anchors.erase(2);
    if (strcmp(scenario, "duplicate") == 0) topology.anchors.emplace(3, MountAnchor{3, "/haa-runtime", "helper"});
    if (strcmp(scenario, "shadow") == 0) topology.anchors.emplace(3, MountAnchor{3, tool, "workspace"});
    if (strcmp(scenario, "parent-shadow") == 0) topology.anchors.emplace(3, MountAnchor{3, "/haa-runtime", "workspace"});
    const TopologyState* selected = strcmp(scenario, "no-topology") == 0 ? nullptr : &topology;
    const bool artifact = strcmp(scenario, "artifact") == 0 || strcmp(scenario, "clone") == 0;
    gvisor::sentry::ExecveInfo event;
    auto* context = event.mutable_context_data();
    context->set_container_id(kFirstID); context->set_thread_group_id(99);
    context->set_thread_group_start_time_ns(990); context->set_is_exec_session(true);
    context->set_parent_thread_group_id(0); context->set_process_name("haa-boundary");
    const auto nonce = TestAdmissionNonce(99, 990);
    auto registration = TestProfileWithAdmission(profile, kLaunchMode, nonce);
    ProcessState state;
    auto parse = [&]() {
      std::string payload, id; const char* reason = nullptr;
      return event.SerializeToString(&payload) && ParseSentryProcessAndClassify(
          payload.data(), payload.size(), output[0], &id, profile, &registration, &state, &reason, selected);
    };
    if (artifact) {
      const bool clone = strcmp(scenario, "clone") == 0;
      context->set_is_exec_session(!clone); context->set_parent_thread_group_id(clone ? 98 : 0);
      if (!RegisterGroup(&state, *context, ProcessState::Role::kArtifact,
          clone ? ProcessState::Provenance::kCloneChild : ProcessState::Provenance::kDirectExecRoot, false, true)) return false;
    } else {
      event.set_binary_path(kBoundaryHelperPath); event.set_execfn(kBoundaryHelperPath);
      event.add_argv(kBoundaryHelperPath); event.add_argv(kLaunchMode); event.add_argv(nonce); event.add_argv(tool);
      if (strcmp(scenario, "no-nonce") == 0) {
        registration.pending_admissions.clear();
        return !parse() && state.groups.empty();
      }
      if (!parse() || !ExpectRecordExact(output[1], kFirstID, "process-exec-expected")) return false;
      event.set_binary_path(kSetprivPath); event.set_execfn(kSetprivPath); event.clear_argv();
      for (const char* arg : {kSetprivPath, "--reuid=1000", "--regid=1000", "--clear-groups",
          "--inh-caps=-all", "--ambient-caps=-all", "--bounding-set=-all", "--no-new-privs", "--", tool}) event.add_argv(arg);
      if (!parse() || !ExpectRecordExact(output[1], kFirstID, "process-exec-expected")) return false;
    }
    event.set_binary_path(tool); event.set_execfn(tool); event.clear_argv();
    event.add_argv(tool);
    if (bun_project) {
      for (const char* argument : {"--cwd=/tmp/haa-bun-input/project", "install", "--lockfile-only", "--ignore-scripts", "--no-env-file", "--registry=https://registry.npmjs.org/", "--cache-dir=/tmp/haa-bun-input/cache", "--backend=copyfile", "--linker=hoisted", "--save-text-lockfile", "--no-progress", "--no-summary"}) event.add_argv(argument);
      event.set_argv(2,bun_intent);
      if (bun_preparation != nullptr) {
        if (strcmp(bun_preparation,"seed")==0) {
          event.set_argv(1,"--cwd=/tmp/haa-bun-input/seed"); event.add_argv("--offline");
        } else if (strcmp(bun_preparation,"offline")==0) {
          event.set_argv(3,"--offline"); event.add_argv("--frozen-lockfile");
        } else return false;
      }
      if (frozen) event.add_argv("--frozen-lockfile");
      if (selector) { event.add_argv("--"); event.add_argv(strcmp(bun_intent,"add")==0 ? "probe@^1.0.0" : "probe"); }
    } else event.add_argv("--version");
    context->set_process_name("uv");
    if (strcmp(scenario, "execfn") == 0) event.set_execfn("/tmp/uv");
    if (strcmp(scenario, "argv0") == 0) event.set_argv(0, "uv");
    if (strcmp(scenario, "argv1") == 0) event.set_argv(1, "run");
    if (strcmp(scenario, "extra-arg") == 0) event.add_argv("file.js");
    const bool allowed_image = strcmp(scenario, "normal") == 0 || artifact;
    if (!allowed_image) return !parse();
    if (!parse()) return false;
    if (artifact) {
      return ExpectUnexpectedProcessRecord(output[1], kFirstID, "SENTRY_EXEC", "OTHER", "ARTIFACT_ROLE", "ARTIFACT_GROUP") &&
          state.groups.at(99).role == ProcessState::Role::kArtifact && !IsTrustedControlNetwork(*context, state);
    }
    return ExpectRecordExact(output[1], kFirstID, "process-exec-expected") &&
        state.groups.at(99).current_image_class == tool_class &&
        IsTrustedControlNetwork(*context, state) &&
        registration.pending_admissions.front().state == ProfileRegistration::AdmissionState::kConsumed;
  };
  bool passed = true;
  for (const char* scenario : {"normal", "artifact", "clone", "no-nonce", "unsealed", "missing-snapshot",
      "namespace", "wrong-class", "wrong-fs", "noexec", "suid", "missing-anchor", "duplicate", "shadow",
      "parent-shadow", "no-topology", "execfn", "argv0"}) {
    if (!run(scenario)) { fprintf(stderr, "%s admission scenario failed: %s\n", pnpm ? "PNPM" : "UV", scenario); passed = false; break; }
  }
  if (bun && (!run("argv1") || !run("extra-arg"))) passed = false;
  close(output[0]); close(output[1]);
  return passed;
}

bool VerifyUVRuntimeAdmissionBoundary() { return VerifyPrivateNativeRuntimeAdmissionBoundary(false); }
bool VerifyPNPMRuntimeAdmissionBoundary() { return VerifyPrivateNativeRuntimeAdmissionBoundary(true); }

namespace {
TopologyState PipxNativeTestTopology() {
  TopologyState topology;
  topology.namespace_id=17; topology.snapshot_seen=true; topology.sealed=true;
  topology.expected={{"/","oci-root","/","",true,false,false,false},
      {"/tmp","workspace","/","tmpfs",false,true,true,false},
      {"/haa-runtime","helper","/","9p",false,false,false,false},
      {"/haa-pipx","workspace","/","9p",true,false,false,false},
      {"/haa-wheels","workspace","/","9p",true,false,false,false},
      {"/haa-site","workspace","/","9p",false,false,false,false}};
  uint64_t id=0;
  for(const auto& mount:topology.expected){++id;topology.anchors.emplace(id,MountAnchor{id,mount.mountpoint,mount.mount_class});}
  return topology;
}

gvisor::sentry::ExecveInfo PipxNativeTestRoot(const char* operation="install") {
  gvisor::sentry::ExecveInfo image;
  auto* context=image.mutable_context_data();context->set_container_id(kFirstID);
  context->set_thread_group_id(99);context->set_thread_group_start_time_ns(990);
  context->set_parent_thread_group_id(0);context->set_is_exec_session(true);context->set_process_name("mutable-name");
  image.set_binary_path("/usr/local/bin/python3.14");image.set_execfn("/usr/local/bin/python");
  // This is a syntactic version slot only, never an actual runtime claim.
  for(const auto& arg:std::vector<std::string>{"/usr/local/bin/python","-I","-S","-B","-c",
      std::string(kPipxNativeProgramPrefix)+"1.2.3"+kPipxNativeProgramSuffix,operation})image.add_argv(arg);
  if(strcmp(operation,"--version")!=0){
    image.add_argv(strcmp(operation,"install")==0?"fixture==1.0":"fixture");
    image.add_argv("--backend");image.add_argv("uv");
    if(strcmp(operation,"install")==0){image.add_argv("--python");image.add_argv("/usr/local/bin/python");}
    image.add_argv(kPipxNativePipArguments);
  }
  return image;
}

bool PipxNativeTestParse(const gvisor::sentry::ExecveInfo& image,int output,int input,
                        ProcessState* state,ProfileRegistration* registration,const TopologyState& topology,
                        bool expected=true,const char* profile=kProfilePipxNativeResolver) {
  std::string payload,id=kFirstID;const char* reason=nullptr;
  if(!image.SerializeToString(&payload))return false;
  const bool parsed=ParseSentryProcessAndClassify(payload.data(),payload.size(),output,&id,profile,registration,state,&reason,&topology);
  if(expected)return parsed&&ExpectRecordExact(input,kFirstID,"process-exec-expected");
  DrainDatagram(input);
  return !parsed||state->first_unexpected_exec.present;
}

bool PipxNativeTestPrime(const gvisor::sentry::ExecveInfo& root,const TopologyState& topology,
                        int output,int input,ProcessState* state) {
  const std::string nonce(64,'c');
  auto registration=TestProfileWithAdmission(kProfilePipxNativeResolver,kPythonHandoffMode,nonce);
  auto boundary=root;boundary.set_binary_path(kBoundaryHelperPath);boundary.set_execfn(kBoundaryHelperPath);boundary.clear_argv();
  for(const auto& arg:std::vector<std::string>{kBoundaryHelperPath,kPythonHandoffMode,nonce})boundary.add_argv(arg);
  for(const auto& arg:root.argv())boundary.add_argv(arg);
  if(!PipxNativeTestParse(boundary,output,input,state,&registration,topology))return false;
  auto demotion=root;demotion.set_binary_path(kSetprivPath);demotion.set_execfn(kSetprivPath);demotion.clear_argv();
  for(const char* arg:{kSetprivPath,"--reuid=1000","--regid=1000","--clear-groups","--inh-caps=-all","--ambient-caps=-all","--bounding-set=-all","--no-new-privs","--"})demotion.add_argv(arg);
  for(const auto& arg:root.argv())demotion.add_argv(arg);
  if(!PipxNativeTestParse(demotion,output,input,state,&registration,topology)||
      !PipxNativeTestParse(root,output,input,state,&registration,topology))return false;
  const auto& group=state->groups[99];
  return registration.pending_admissions[0].state==ProfileRegistration::AdmissionState::kConsumed&&
      group.pipx_native_image==ProcessState::PipxNativeImage::kRoot&&IsPipxNativeActiveGroup(group)&&
      state->expected_groups.empty()&&!IsTrustedControlNetwork(root.context_data(),*state);
}

bool PipxNativeTestClone(const gvisor::common::ContextData& parent,int32_t child,int64_t start,
                        ProcessState* state,gvisor::common::ContextData* context) {
  gvisor::sentry::CloneInfo clone;*clone.mutable_context_data()=parent;
  clone.set_created_thread_group_id(child);clone.set_created_thread_start_time_ns(start);
  std::string payload,id=kFirstID;const char* reason=nullptr;
  if(!clone.SerializeToString(&payload)||!ParseSentryClone(payload.data(),payload.size(),-1,&id,state,&reason))return false;
  *context=parent;context->set_thread_group_id(child);context->set_thread_group_start_time_ns(start);
  context->set_parent_thread_group_id(parent.thread_group_id());context->set_process_name("renamed-child");
  return !state->groups[child].root_eligible&&state->groups[child].root_consumed&&!state->groups[child].trusted_control_network_active&&
      state->groups[child].pipx_native_image==ProcessState::PipxNativeImage::kNone;
}

gvisor::sentry::ExecveInfo PipxNativeTestUV(const gvisor::common::ContextData& context,const char* operation="version",bool upgrade=false) {
  gvisor::sentry::ExecveInfo image;*image.mutable_context_data()=context;image.set_binary_path(kUVToolPath);image.set_execfn(kUVToolPath);image.add_argv(kUVToolPath);
  const std::string env="/haa-site/pipx/venvs/fixture";
  if(strcmp(operation,"version")==0)image.add_argv("--version");
  else if(strcmp(operation,"venv")==0){for(const auto& arg:std::vector<std::string>{"venv","--python","/usr/local/bin/python","--allow-existing","--quiet",env})image.add_argv(arg);}
  else{
    for(const auto& arg:std::vector<std::string>{"pip","install","--python",env+"/bin/python","--quiet"})image.add_argv(arg);
    if(upgrade)image.add_argv("--upgrade");
    for(const char* arg:{"--no-build","--no-sources","--no-config","--no-cache","--no-python-downloads","--no-managed-python","--link-mode=copy","--index-url=https://pypi.org/simple/"})image.add_argv(arg);
    image.add_argv(upgrade?"fixture":"fixture==1.0");
  }
  return image;
}

gvisor::sentry::ExecveInfo PipxNativeTestQuery(const gvisor::common::ContextData& context,bool metadata,bool alias=true) {
  gvisor::sentry::ExecveInfo image;*image.mutable_context_data()=context;image.set_binary_path("/usr/local/bin/python3.14");
  image.set_execfn(alias?"/haa-site/pipx/venvs/fixture/bin/python":"/usr/local/bin/python");image.add_argv(image.execfn());
  if(metadata){image.add_argv("-c");image.add_argv(kPipxNativeMetadataQuery);}
  else {image.add_argv("-I");image.add_argv("-B");image.add_argv("-c");image.add_argv("import sys; sys.path = [__import__(\"os\").fsdecode(b\"/tmp/.tmp3l9KcP/.tmpADARLv\")] + sys.path; from python.get_interpreter_info import main; main()");}
  return image;
}

bool VerifyPipxNativeRootAndTopologyBoundary() {
  if(IsPythonProfile(kProfilePipxNativeResolver)||!IsPythonRuntimeProfile(kProfilePipxNativeResolver)||
      MaximumRecords(kProfilePipxNativeResolver)!=kMaxNormalizedRecordsPerConnection)return false;
  int output[2];if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
  bool ok=true;const auto topology=PipxNativeTestTopology();
  for(const char* operation:{"install","upgrade","--version"}){ProcessState state;if(!PipxNativeTestPrime(PipxNativeTestRoot(operation),topology,output[0],output[1],&state))ok=false;}
  auto root=PipxNativeTestRoot();
  for(int i=0;i<root.argv_size();++i){auto bad=root;bad.set_argv(i,i==7?"../foreign":"foreign");ProcessState state;if(PipxNativeTestPrime(bad,topology,output[0],output[1],&state))ok=false;}
  for(const std::string& value:{std::string(kPipxNativeProgramPrefix)+"1.2.3';print('bad')#"+kPipxNativeProgramSuffix,
       std::string(kPipxNativeProgramPrefix)+"1..3"+kPipxNativeProgramSuffix,std::string(kPipxNativeProgramPrefix)+"1.2.3"+kPipxNativeProgramSuffix+"print('extra')"}){
    auto bad=root;bad.set_argv(5,value);ProcessState state;if(PipxNativeTestPrime(bad,topology,output[0],output[1],&state))ok=false;
  }
  for(int scenario=0;scenario<11;++scenario){auto changed=topology;
    switch(scenario){case 0:changed.sealed=false;break;case 1:changed.snapshot_seen=false;break;case 2:changed.namespace_id=0;break;
      case 3:changed.expected[3].read_only=false;break;case 4:changed.expected[4].read_only=false;break;case 5:changed.expected[5].read_only=true;break;
      case 6:changed.expected[2].filesystem_type="tmpfs";break;case 7:changed.anchors.erase(5);break;
      case 8:changed.anchors.emplace(20,MountAnchor{20,"/haa-wheels/shadow","workspace"});break;
      case 9:changed.anchors.emplace(20,MountAnchor{20,"/haa-runtime/uv","helper"});break;
      case 10:changed.expected.push_back({"/extra","workspace","/","9p",false,false,false,false});break;}
    if(IsPipxNativeTopology(&changed)){ok=false;break;}
    ProcessState state;if(PipxNativeTestPrime(root,changed,output[0],output[1],&state))ok=false;
  }
  close(output[0]);close(output[1]);return ok;
}

bool VerifyPipxNativeCloneAndQueryBoundary() {
  int output[2];if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
  bool ok=true;const auto topology=PipxNativeTestTopology();
  auto run=[&](const char* operation,bool upgrade,bool metadata,bool alias,const std::string& fault){
    DrainDatagram(output[1]);ProcessState state;auto root=PipxNativeTestRoot(upgrade?"upgrade":"install");
    const bool clauses=fault.compare(0,8,"clauses-")==0;
    if(clauses)root.set_argv(7,"fixture>=1.0,<=1.0");
    if(!PipxNativeTestPrime(root,topology,output[0],output[1],&state))return false;
    gvisor::common::ContextData child;
    if(!PipxNativeTestClone(root.context_data(),100,1000,&state,&child))return false;
    auto image=PipxNativeTestUV(child,operation,upgrade);ProfileRegistration registration;
    if(clauses){
      std::string requirement="fixture<=1.0,>=1.0";
      if(fault=="clauses-repeat")requirement="fixture<=1.0,>=1.0,>=1.0";
      if(fault=="clauses-project")requirement="foreign<=1.0,>=1.0";
      if(fault=="clauses-version")requirement="fixture<=2.0,>=1.0";
      if(fault=="clauses-operator")requirement="fixture<1.0,>=1.0";
      if(fault=="clauses-missing")requirement="fixture>=1.0";
      if(fault=="clauses-extra")requirement="fixture<=1.0,>=1.0,!=1.0";
      if(fault=="clauses-empty")requirement="fixture<=1.0,,>=1.0";
      if(fault=="clauses-trailing")requirement="fixture<=1.0,>=1.0,";
      if(fault=="clauses-bare")requirement="fixture";
      image.set_argv(image.argv_size()-1,requirement);
      if(fault=="clauses-flags")image.set_argv(6,"--no-deps");
    }
    if(metadata||std::string(operation)=="query"){
      if(!metadata){if(!PipxNativeTestParse(image,output[0],output[1],&state,&registration,topology))return false;
        auto uv_context=image.context_data();if(!PipxNativeTestClone(uv_context,101,1010,&state,&child))return false;}
      image=PipxNativeTestQuery(child,metadata,alias);
    }
    const int id=image.context_data().thread_group_id();const int parent=image.context_data().parent_thread_group_id();
    if(fault=="binary")image.set_binary_path("/tmp/copied-runtime");
    if(fault=="execfn")image.set_execfn("/tmp/alias");
    if(fault=="argv0")image.set_argv(0,"foreign");
    if(fault=="argv")image.set_argv(image.argv_size()-1,"foreign");
    if(fault=="extra")image.add_argv("extra");
    if(fault=="origin")image.mutable_context_data()->set_is_exec_session(false);
    if(fault=="wrong-parent")image.mutable_context_data()->set_parent_thread_group_id(80);
    if(fault=="child-start")image.mutable_context_data()->set_thread_group_start_time_ns(777);
    if(fault=="child-network")state.groups[id].trusted_control_network_active=true;
    if(fault=="child-eligible")state.groups[id].root_eligible=true;
    if(fault=="child-pending")state.groups[id].handoff_target_pending=true;
    if(fault=="child-expected")state.expected_groups[id]={state.groups[id].start_time_ns,ProcessClass::kUV};
    if(fault=="missing-clone")state.groups[id].pipx_native_candidate=false;
    if(fault=="parent-role")state.groups[parent].role=ProcessState::Role::kControl;
    if(fault=="parent-start")state.groups[parent].start_time_ns++;
    if(fault=="parent-network")state.groups[parent].trusted_control_network_active=true;
    if(fault=="parent-image")state.groups[parent].pipx_native_image=ProcessState::PipxNativeImage::kNone;
    if(fault=="root-start")state.groups[99].start_time_ns++;
    const bool normal=fault.empty()||fault=="clauses-reorder"||fault=="clauses-repeat";
    if(!PipxNativeTestParse(image,output[0],output[1],&state,&registration,topology,normal))return false;
    const auto expected=normal?(metadata?ProcessState::PipxNativeImage::kMetadataQuery:(std::string(operation)=="query"?ProcessState::PipxNativeImage::kUVQuery:ProcessState::PipxNativeImage::kUV)):ProcessState::PipxNativeImage::kNone;
    if(state.groups[id].pipx_native_image!=expected||IsTrustedControlNetwork(image.context_data(),state))return false;
    if(normal&&(state.groups[id].role!=ProcessState::Role::kArtifact||!state.expected_groups.empty()||state.groups[id].pipx_native_candidate))return false;
    return true;
  };
  for(const char* operation:{"version","venv","install","query"})ok=run(operation,false,false,true,"")&&ok;
  ok=run("venv",true,false,true,"")&&run("install",true,false,true,"")&&run("query",false,false,false,"")&&run("version",false,true,true,"")&&ok;
  for(const char* fault:{"binary","execfn","argv0","argv","extra","origin","wrong-parent","child-start","child-network","child-eligible","child-pending","child-expected","missing-clone","parent-role","parent-start","parent-network","parent-image","root-start"}){
    ok=run("venv",true,false,true,fault)&&run("install",false,false,true,fault)&&run("query",false,false,true,fault)&&run("version",false,true,true,fault)&&ok;
  }
  for(const char* shape:{"clauses-reorder","clauses-repeat","clauses-project","clauses-version","clauses-operator","clauses-missing","clauses-extra","clauses-empty","clauses-trailing","clauses-bare","clauses-flags"})ok=run("install",false,false,true,shape)&&ok;
  close(output[0]);close(output[1]);return ok;
}

bool VerifyPipxNativeReadAndReexecBoundary() {
  int output[2];if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
  bool ok=true;const auto topology=PipxNativeTestTopology();
  for(bool metadata:{false,true}){
    ProcessState state;auto root=PipxNativeTestRoot();if(!PipxNativeTestPrime(root,topology,output[0],output[1],&state)){ok=false;break;}
    gvisor::common::ContextData child;if(!PipxNativeTestClone(root.context_data(),100,1000,&state,&child)){ok=false;break;}
    ProfileRegistration registration;
    if(!metadata){auto uv=PipxNativeTestUV(child);if(!PipxNativeTestParse(uv,output[0],output[1],&state,&registration,topology)){ok=false;break;}auto parent=child;if(!PipxNativeTestClone(parent,101,1010,&state,&child)){ok=false;break;}}
    auto image=PipxNativeTestQuery(child,metadata);if(!PipxNativeTestParse(image,output[0],output[1],&state,&registration,topology)){ok=false;break;}
    const auto baseline=state;gvisor::syscall::Open open;*open.mutable_context_data()=child;open.set_flags(557056);
    for(const auto& check:std::vector<std::tuple<std::string,uint64_t,uint32_t,bool>>{
        {"/haa-site/pipx/venvs/fixture/lib/python3.14/site-packages/evil.pth",6,557056,true},
        {"/tmp/query-write",2,kOpenWriteOnly,true},
        {"/haa-site/pipx/venvs/fixture/pyvenv.cfg",6,557056,false}}){
      if(!VerifyRestrictedWorkspaceOpen(child,state,topology,kProfilePipxNativeResolver,output[0],output[1],std::get<0>(check),std::get<1>(check),std::get<2>(check),std::get<3>(check)))ok=false;
    }

    MountAnchor site{6,"/haa-site","workspace"};const std::string env="/haa-site/pipx/venvs/fixture";
    for(const std::string& path:{env+"/pyvenv.cfg",env,env+"/lib",env+"/lib/python3.14",env+"/lib/python3.14/site-packages"}){open.set_pathname(path);if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&site)!=FilesystemClass::kHelperOnly)ok=false;}
    if(!VerifyRestrictedWorkspaceOpen(child,state,topology,kProfilePipxNativeResolver,output[0],output[1],env+"/pyvenv.cfg",6,kOpenLargefile,false))ok=false;
    for(uint64_t flags:{uint64_t{0},kOpenLargefile|kOpenWriteOnly,kOpenLargefile|kOpenReadWrite,kOpenLargefile|kOpenCreate,kOpenLargefile|kOpenAppend,kOpenLargefile|kOpenTruncate}){open.set_pathname(env+"/pyvenv.cfg");open.set_flags(flags);if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&site)!=FilesystemClass::kOutside)ok=false;}
    if(!VerifyRestrictedWorkspaceOpen(child,state,topology,kProfilePipxNativeResolver,output[0],output[1],env+"/lib/python3.14/site-packages",6,624640,false))ok=false;
    for(const std::string& path:{env,env+"/lib",env+"/lib/python3.14",env+"/pyvenv.cfg",env+"/lib/python3.14/site-packages/evil.pth",env+"/lib/python3.14/site-packages/sitecustomize.py",env+"/lib/python3.14/site-packages/fixture/__init__.py",std::string("/haa-site/pipx/venvs/other/lib/python3.14/site-packages")}){
      if(!VerifyRestrictedWorkspaceOpen(child,state,topology,kProfilePipxNativeResolver,output[0],output[1],path,6,624640,true))ok=false;
    }
    for(uint64_t flags:{uint64_t{624640}|kOpenWriteOnly,uint64_t{624640}|kOpenReadWrite,uint64_t{624640}|kOpenCreate,uint64_t{624640}|kOpenAppend,uint64_t{624640}|kOpenTruncate}){
      if(!VerifyRestrictedWorkspaceOpen(child,state,topology,kProfilePipxNativeResolver,output[0],output[1],env+"/lib/python3.14/site-packages",6,flags,true))ok=false;
    }
    open.set_flags(557056);
    for(const std::string& path:{env+"/lib/python3.14/site-packages/evil.pth",env+"/lib/python3.14/site-packages/sitecustomize.py",env+"/lib/python3.14/site-packages/fixture/__init__.py",env+"/lib/python3.14/site-packages/foreign.so",env+"/lib/python3.14/site-packages/fixture-1.0.dist-info/METADATA",std::string("/haa-site/pipx/venvs/other/pyvenv.cfg"),std::string("/tmp/../haa-site/pipx/venvs/fixture/pyvenv.cfg"),std::string("/tmp/output-alias/pyvenv.cfg")}){open.set_pathname(path);if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&site)!=FilesystemClass::kOutside)ok=false;}
    for(uint64_t flags:{kOpenWriteOnly,kOpenReadWrite,kOpenCreate,kOpenAppend,kOpenTruncate}){open.set_pathname(env+"/pyvenv.cfg");open.set_flags(flags);if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&site)!=FilesystemClass::kOutside)ok=false;
      MountAnchor tmp{2,"/tmp","workspace"};open.set_pathname("/tmp/query-write");if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&tmp)!=FilesystemClass::kOutside)ok=false;}
    open.set_flags(557056);
    for(const auto& input:std::vector<MountAnchor>{{4,"/haa-pipx","workspace"},{5,"/haa-wheels","workspace"}}){open.set_pathname(input.mountpoint=="/haa-pipx"?"/haa-pipx/pipx-runtime.zip":"/haa-wheels/fixture-1-py3-none-any.whl");if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&input)!=FilesystemClass::kOutside)ok=false;}
    MountAnchor runtime{1,"/","oci-root"};open.set_pathname("/usr/local/lib/python3.14/json/__init__.py");if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&runtime)!=FilesystemClass::kRuntimeRoot)ok=false;
    state.groups[99].start_time_ns++;if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&runtime)!=FilesystemClass::kOutside)ok=false;state=baseline;
    gvisor::common::ContextData descendant;if(!PipxNativeTestClone(child,102,1020,&state,&descendant)){ok=false;break;}
    *open.mutable_context_data()=descendant;open.set_pathname(env+"/pyvenv.cfg");if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&site)!=FilesystemClass::kOutside)ok=false;
    state=baseline;*open.mutable_context_data()=child;
    if(!PipxNativeTestParse(image,output[0],output[1],&state,&registration,topology,false)||
        !state.groups[child.thread_group_id()].pipx_native_query_restricted||state.groups[child.thread_group_id()].pipx_native_image!=ProcessState::PipxNativeImage::kNone||
        ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&site)!=FilesystemClass::kOutside)ok=false;
  }
  // The immutable DATA root is readable only by the active artifact root or
  // its admitted UV child, and writes/aliases remain outside the workspace.
  ProcessState state;auto root=PipxNativeTestRoot();if(!PipxNativeTestPrime(root,topology,output[0],output[1],&state))ok=false;
  gvisor::syscall::Open open;*open.mutable_context_data()=root.context_data();open.set_flags(557056);
  for(const auto& anchor:std::vector<MountAnchor>{{4,"/haa-pipx","workspace"},{5,"/haa-wheels","workspace"}}){
    open.set_pathname(anchor.mountpoint=="/haa-pipx"?"/haa-pipx/pipx-runtime.zip":"/haa-wheels/fixture-1-py3-none-any.whl");
    if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&anchor)!=FilesystemClass::kWorkspace)ok=false;
    if(!VerifyRestrictedWorkspaceOpen(root.context_data(),state,topology,kProfilePipxNativeResolver,output[0],output[1],open.pathname(),anchor.mount_id,557056,false,true))ok=false;
    if(!VerifyRestrictedWorkspaceOpen(root.context_data(),state,topology,kProfilePipxNativeResolver,output[0],output[1],open.pathname(),anchor.mount_id,kOpenWriteOnly,true))ok=false;
    for(uint64_t flags:{kOpenWriteOnly,kOpenReadWrite,kOpenCreate,kOpenAppend,kOpenTruncate}){open.set_flags(flags);if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&anchor)!=FilesystemClass::kOutside)ok=false;}open.set_flags(557056);
    open.set_pathname(anchor.mountpoint+"/../outside");if(ClassifyFilesystemOpen(open,state,kProfilePipxNativeResolver,&anchor)!=FilesystemClass::kOutside)ok=false;
  }
  close(output[0]);close(output[1]);return ok;
}

bool VerifyPipxNativeFirstOutsideDiagnostic() {
 int output[2];if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
 bool ok=true;const auto topology=PipxNativeTestTopology();ProcessState state;auto root=PipxNativeTestRoot();ProfileRegistration registration;
 gvisor::common::ContextData uv,query;
 if(!PipxNativeTestPrime(root,topology,output[0],output[1],&state)||
    !PipxNativeTestClone(root.context_data(),100,1000,&state,&uv)||
    !PipxNativeTestParse(PipxNativeTestUV(uv),output[0],output[1],&state,&registration,topology)||
    !PipxNativeTestClone(uv,101,1010,&state,&query)||
    !PipxNativeTestParse(PipxNativeTestQuery(query,false),output[0],output[1],&state,&registration,topology)){close(output[0]);close(output[1]);return false;}
 const std::string env="/haa-site/pipx/venvs/fixture";
 ProcessState first,second,later;
 if(!VerifyRestrictedWorkspaceOpen(query,state,topology,kProfilePipxNativeResolver,output[0],output[1],env+"/lib",6,624640,true,false,&first))ok=false;
 const std::string saved=first.first_outside_open_diagnostic;
 if(saved.empty()||saved.find(env)!=std::string::npos||saved.find("\"flags\":624640")==std::string::npos)ok=false;
 if(!VerifyRestrictedWorkspaceOpen(query,first,topology,kProfilePipxNativeResolver,output[0],output[1],env+"/lib/python3.14",6,624640,true,false,&second)||second.first_outside_open_diagnostic!=saved)ok=false;
 if(VerifyRestrictedWorkspaceOpen(uv,second,topology,kProfilePipxNativeResolver,output[0],output[1],"/usr/local/bin/pip",1,557056,false,false,&later)||
    later.first_outside_open_diagnostic!=saved||later.fault_open_diagnostic.empty())ok=false;
 auto emit=[&](const std::string& first_open){
  ProcessState diagnostic=later;diagnostic.first_outside_open_diagnostic=first_open;
  const std::string packed=FaultOpenWithFirstOutside(diagnostic,"STREAM_FAULT","");
  if(!Send(output[0],kFirstID,"stream-fault","STREAM_FAULT",nullptr,0,"OPEN_RESULT_CLASSIFICATION_IMAGE",UINT64_MAX,packed))return std::string();
  char bytes[2048];const ssize_t n=recv(output[1],bytes,sizeof(bytes),MSG_DONTWAIT);
  return n>0?std::string(bytes,static_cast<size_t>(n)):std::string();
 };
 const std::string record=emit(saved);
 if(record.empty()||record.size()>1024||record.find("\"first_outside\":"+saved)==std::string::npos||
    record.find("\"fault_open\":"+later.fault_open_diagnostic.substr(0,later.fault_open_diagnostic.size()-1))==std::string::npos||record.find(env)!=std::string::npos)ok=false;
 const std::string bounded=emit(std::string(2048,'x'));
 if(bounded.empty()||bounded.size()>1024||bounded.find("first_outside")!=std::string::npos||bounded.find("\"reason\":\"STREAM_FAULT\"")==std::string::npos)ok=false;
 close(output[0]);close(output[1]);return ok;
}

bool VerifyPipxNativeBaseQueryDirectoryRead() {
 int output[2];if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
 bool ok=true;const auto topology=PipxNativeTestTopology();ProcessState state;auto root=PipxNativeTestRoot();ProfileRegistration registration;
 gvisor::common::ContextData uv,query;
 if(!PipxNativeTestPrime(root,topology,output[0],output[1],&state)||
    !PipxNativeTestClone(root.context_data(),100,1000,&state,&uv)||
    !PipxNativeTestParse(PipxNativeTestUV(uv),output[0],output[1],&state,&registration,topology)||
    !PipxNativeTestClone(uv,101,1010,&state,&query)||
    !PipxNativeTestParse(PipxNativeTestQuery(query,false,false),output[0],output[1],&state,&registration,topology)){close(output[0]);close(output[1]);return false;}
 const auto baseline=state;MountAnchor anchor{1,"/","oci-root"};const std::string path="/usr/local/lib/python3.14/site-packages";
 auto permitted=[&](const gvisor::common::ContextData& context,const char* profile,const std::string& target,uint64_t flags,const MountAnchor& mount){return IsPipxNativeABIRead(context,state,target,flags,profile,&mount);};
 if(!state.groups[101].pipx_native_environment.empty()||!permitted(query,kProfilePipxNativeResolver,path,624640,anchor)||
    !VerifyRestrictedWorkspaceOpen(query,state,topology,kProfilePipxNativeResolver,output[0],output[1],path,1,624640,false))ok=false;
 for(const char* target:{"/usr/local/lib/python3.14/site-packages/evil.pth","/usr/local/lib/python3.14/site-packages/foreign.py","/usr/local/lib/python3.14/../python3.14/site-packages"})if(permitted(query,kProfilePipxNativeResolver,target,624640,anchor))ok=false;
 for(uint64_t flags:{uint64_t{0},kOpenLargefile,uint64_t{624640}|kOpenWriteOnly,uint64_t{624640}|kOpenReadWrite,uint64_t{624640}|kOpenCreate,uint64_t{624640}|kOpenAppend,uint64_t{624640}|kOpenTruncate})if(permitted(query,kProfilePipxNativeResolver,path,flags,anchor))ok=false;
 for(const auto& mount:std::vector<MountAnchor>{{1,"/","workspace"},{1,"/usr/local/lib/python3.14","oci-root"}})if(permitted(query,kProfilePipxNativeResolver,path,624640,mount))ok=false;
 for(const char* profile:{kProfileUVResolver,kProfilePipxRuntime,kProfilePyPI})if(permitted(query,profile,path,624640,anchor))ok=false;
 for(int scenario=0;scenario<9;++scenario){
  state=baseline;auto context=query;
  switch(scenario){
   case 0:state.groups[101].role=ProcessState::Role::kControl;break;
   case 1:state.groups[101].pipx_native_environment="/haa-site/pipx/venvs/fixture";break;
   case 2:state.groups[101].start_time_ns++;break;
   case 3:state.groups[100].start_time_ns++;break;
   case 4:state.groups[99].start_time_ns++;break;
   case 5:state.groups[100].role=ProcessState::Role::kControl;break;
   case 6:state.groups[101].trusted_control_network_active=true;break;
   case 7:context.set_parent_thread_group_id(99);break;
   case 8:state.groups[101].handoff_target_pending=true;break;
  }
  if(permitted(context,kProfilePipxNativeResolver,path,624640,anchor))ok=false;
 }
 state=baseline;gvisor::common::ContextData descendant;
 if(!PipxNativeTestClone(query,102,1020,&state,&descendant)||permitted(descendant,kProfilePipxNativeResolver,path,624640,anchor))ok=false;
 close(output[0]);close(output[1]);return ok;
}

bool VerifyPipxNativeUVProcReadBoundary() {
 int output[2];if(socketpair(AF_UNIX,SOCK_DGRAM,0,output)!=0)return false;
 bool ok=true;auto topology=PipxNativeTestTopology();topology.anchors.emplace(7,MountAnchor{7,"/proc","system"});topology.anchors.emplace(8,MountAnchor{8,"/sys/fs/cgroup/cpu","system"});topology.anchors.emplace(9,MountAnchor{9,"/dev","system"});
 ProcessState state;auto root=PipxNativeTestRoot();ProfileRegistration registration;
 if(!PipxNativeTestPrime(root,topology,output[0],output[1],&state)){close(output[0]);close(output[1]);return false;}
 gvisor::common::ContextData child;
 if(!PipxNativeTestClone(root.context_data(),100,1000,&state,&child)||!PipxNativeTestParse(PipxNativeTestUV(child),output[0],output[1],&state,&registration,topology)){close(output[0]);close(output[1]);return false;}
 const auto baseline=state;MountAnchor anchor{7,"/proc","system"};
 gvisor::syscall::Open comm;*comm.mutable_context_data()=child;comm.set_pathname("/proc/100/comm");comm.set_flags(557056);
 const auto before_comm=ClassifyFilesystemOpen(comm,state,kProfilePipxNativeResolver,&anchor);
 const std::string comm_diagnostic=FaultOpenDiagnostic(comm,state,anchor);
 if(IsPipxNativeABIRead(child,state,comm.pathname(),comm.flags(),kProfilePipxNativeResolver,&anchor)||
    ClassifyFilesystemOpen(comm,state,kProfilePipxNativeResolver,&anchor)!=before_comm||
    comm_diagnostic.find("\"subject\":\"PROC_SELF_COMM\"")==std::string::npos||
    strcmp(FaultOpenSubject(child,"/proc/99/comm",anchor),"PROC_PARENT_COMM")!=0||
    strcmp(FaultOpenSubject(child,"/proc/98/comm",anchor),"OTHER")!=0||
    strcmp(FaultOpenSubject(child,"/proc/self/comm",anchor),"OTHER")!=0||
    strcmp(FaultOpenSubject(child,"/proc/100/comm/extra",anchor),"OTHER")!=0)ok=false;

 auto thread=child;thread.set_thread_id(104);
 if(strcmp(FaultOpenSubject(thread,"/proc/104/comm",anchor),"PROC_THREAD_COMM")!=0||
    strcmp(FaultOpenSubject(thread,"/proc/100/comm",anchor),"PROC_SELF_COMM")!=0||
    strcmp(FaultOpenSubject(thread,"/proc/99/comm",anchor),"PROC_PARENT_COMM")!=0||
    strcmp(FaultOpenSubject(thread,"/proc/103/comm",anchor),"OTHER")!=0)ok=false;
 const std::string path="/proc/sys/vm/overcommit_memory";
 auto permitted=[&](const gvisor::common::ContextData& context,const char* profile,const std::string& target,uint64_t flags,const MountAnchor& mount){return IsPipxNativeABIRead(context,state,target,flags,profile,&mount);};
 const std::vector<std::pair<std::string,MountAnchor>> reads={
  {path,anchor},{"/proc/100/cgroup",anchor},{"/proc/100/mountinfo",anchor},{"/proc/99/comm",anchor},
  {"/sys/fs/cgroup/cpu/cpu.cfs_quota_us",MountAnchor{8,"/sys/fs/cgroup/cpu","system"}},
  {"/sys/fs/cgroup/cpu/cpu.cfs_period_us",MountAnchor{8,"/sys/fs/cgroup/cpu","system"}},
  {"/dev/null",MountAnchor{9,"/dev","system"}},
  {"/usr/local/bin/python3.14",MountAnchor{1,"/","oci-root"}},
  {"/usr/lib/os-release",MountAnchor{1,"/","oci-root"}}};
 for(const auto& read:reads){
  if(!permitted(child,kProfilePipxNativeResolver,read.first,557056,read.second)||
     !VerifyRestrictedWorkspaceOpen(child,state,topology,kProfilePipxNativeResolver,output[0],output[1],read.first,read.second.mount_id,557056,false))ok=false;
  for(uint64_t flags:{uint64_t{0},kOpenLargefile,uint64_t{622592},kOpenWriteOnly,kOpenReadWrite,kOpenCreate,kOpenAppend,kOpenTruncate})if(permitted(child,kProfilePipxNativeResolver,read.first,flags,read.second))ok=false;
  for(const char* profile:{kProfileUVResolver,kProfilePipxRuntime,kProfilePyPI})if(permitted(child,profile,read.first,557056,read.second))ok=false;
  if(permitted(root.context_data(),kProfilePipxNativeResolver,read.first,557056,read.second))ok=false;
 }
 MountAnchor quota{8,"/sys/fs/cgroup/cpu","system"};
 for(const char* target:{"/sys/fs/cgroup/cpu/cpuacct.usage","/sys/fs/cgroup/cpu/cpu.shares","/sys/fs/cgroup/cpu/cpu.stat","/sys/fs/cgroup/cpu/../cpu/cpu.cfs_quota_us"})if(permitted(child,kProfilePipxNativeResolver,target,557056,quota))ok=false;
 for(const auto& mount:std::vector<MountAnchor>{{8,"/sys/fs/cgroup","system"},{8,"/sys/fs/cgroup/cpu","workspace"},{8,"/","oci-root"}})if(permitted(child,kProfilePipxNativeResolver,"/sys/fs/cgroup/cpu/cpu.cfs_quota_us",557056,mount))ok=false;
 MountAnchor image{1,"/","oci-root"};
 for(const char* target:{"/usr/local/bin/python","/usr/local/bin/python3.13","/usr/bin/python3.14","/usr/local/bin/python3.14/extra"})if(permitted(child,kProfilePipxNativeResolver,target,557056,image))ok=false;
 for(const auto& mount:std::vector<MountAnchor>{{1,"/usr/local/bin","oci-root"},{1,"/","workspace"}})if(permitted(child,kProfilePipxNativeResolver,"/usr/local/bin/python3.14",557056,mount))ok=false;
 for(const char* target:{"/etc/os-release","/usr/lib/os-release/extra","/usr/lib/../lib/os-release","/etc/lsb-release"})if(permitted(child,kProfilePipxNativeResolver,target,557056,image))ok=false;
 for(const auto& mount:std::vector<MountAnchor>{{1,"/usr/lib","oci-root"},{1,"/","workspace"}})if(permitted(child,kProfilePipxNativeResolver,"/usr/lib/os-release",557056,mount))ok=false;
 MountAnchor device{9,"/dev","system"};
 for(const char* target:{"/dev/zero","/dev/random","/dev/tty","/dev/../dev/null"})if(permitted(child,kProfilePipxNativeResolver,target,557056,device))ok=false;
 for(const auto& mount:std::vector<MountAnchor>{{9,"/dev/other","system"},{9,"/dev","workspace"},{9,"/","oci-root"}})if(permitted(child,kProfilePipxNativeResolver,"/dev/null",557056,mount))ok=false;
 for(const char* target:{"/proc/sys/vm/overcommit_ratio","/proc/self/maps","/proc/100/maps","/proc/101/cgroup","/proc/self/cgroup","/proc/101/mountinfo","/proc/self/mountinfo","/proc/100/comm","/proc/98/comm","/proc/99/comm/extra","/proc/self/comm","/proc/sys/vm/../vm/overcommit_memory"})if(permitted(child,kProfilePipxNativeResolver,target,557056,anchor))ok=false;
 for(const auto& mount:std::vector<MountAnchor>{{7,"/proc/sys","system"},{7,"/proc","workspace"},{7,"/","oci-root"}})if(permitted(child,kProfilePipxNativeResolver,path,557056,mount))ok=false;
 for(const char* profile:{kProfileUVResolver,kProfilePipxRuntime,kProfilePyPI})if(permitted(child,profile,path,557056,anchor))ok=false;
 if(permitted(root.context_data(),kProfilePipxNativeResolver,path,557056,anchor))ok=false;
 for(int scenario=0;scenario<13;++scenario){
  state=baseline;auto context=child;
  switch(scenario){
   case 0:state.groups[100].role=ProcessState::Role::kControl;break;
   case 1:state.groups[100].current_image_class=ProcessClass::kPython;break;
   case 2:state.groups[100].pipx_native_image=ProcessState::PipxNativeImage::kUVQuery;break;
   case 3:state.groups[100].trusted_control_network_active=true;break;
   case 4:state.groups[100].start_time_ns++;break;
   case 5:state.groups[99].start_time_ns++;break;
   case 6:state.groups[99].role=ProcessState::Role::kControl;break;
   case 7:state.groups[99].pipx_native_image=ProcessState::PipxNativeImage::kNone;break;
   case 8:state.groups[100].handoff_target_pending=true;break;
   case 9:state.expected_groups[100]=ProcessState::ExpectedGroup{1000,ProcessClass::kUV};break;
   case 10:context.set_parent_thread_group_id(98);break;
   case 11:context.set_is_exec_session(false);break;
   case 12:state.groups.erase(99);break;
  }
  for(const auto& read:reads)if(permitted(context,kProfilePipxNativeResolver,read.first,557056,read.second))ok=false;
 }
 close(output[0]);close(output[1]);return ok;
}

bool VerifyPipxNativeProfileAdmission(const std::string& control) {
  constexpr char id[]="7b7b7b7b7b7b7b7b";
  if(!RegisterProfile(control,id,kProfilePipxNativeResolver))return false;
  const auto client=gControlClients.find(id);if(client==gControlClients.end())return false;
  const auto generation=TestSessionGeneration(id);
  const bool ok=ControlAdmissionOnFD(client->second,id,kELFHandoffMode,generation,std::string(64,'b'),"arm","rejected")&&
      ArmAdmission(control,id,kPythonHandoffMode,std::string(64,'c'))&&CancelAdmission(control,id,kPythonHandoffMode,std::string(64,'c'))&&
      ArmAdmission(control,id,kLaunchMode,std::string(64,'d'))&&CancelAdmission(control,id,kLaunchMode,std::string(64,'d'))&&
      VerifyPipxRuntimeReadiness(kProfilePipxNativeResolver)&&InvalidateProfile(control,id,generation);
  close(client->second);gControlClients.erase(client);return ok;
}

}

int main(int argc, char** argv) {
  if (argc == 2 && strcmp(argv[1], "--fd-mutation-diagnostic-only") == 0) {
    const bool passed = VerifyFDMutationDiagnosticPreservesAuthority();
    fprintf(stderr, "FD mutation diagnostic preserves authority: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if(argc==2 && strcmp(argv[1],"--cargo-build-program-query-only")==0){const bool passed=VerifyCargoBuildProgramRustcQueryBoundary();fprintf(stderr,"Cargo program query %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-program-null-only")==0){const bool passed=VerifyCargoBuildProgramBoundary(true,true,true);fprintf(stderr,"Cargo program null %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-config-only")==0){const bool passed=VerifyCargoBuildConfigurationBoundary();fprintf(stderr,"Cargo configuration %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-pinned-target-only")==0){const bool passed=VerifyCargoBuildHandoffPinnedTarget();fprintf(stderr,"Cargo build pinned target %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-program-maps-only")==0){const bool passed=VerifyCargoBuildProgramBoundary(true,true);fprintf(stderr,"Cargo build program maps %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-program-abi-only")==0){const bool passed=VerifyCargoBuildProgramBoundary(true);fprintf(stderr,"Cargo build program ABI %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-program-only")==0){const bool passed=VerifyCargoBuildProgramBoundary();fprintf(stderr,"Cargo build program %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-lld-entropy-only")==0){const bool passed=VerifyCargoBuildNativeCCChildBoundary(7);fprintf(stderr,"Cargo build lld entropy %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-rust-c-abi-only")==0){const bool passed=VerifyCargoBuildNativeCCChildBoundary(6);fprintf(stderr,"Cargo build Rust C ABI %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-link-sdk-only")==0){const bool passed=VerifyCargoBuildNativeCCChildBoundary(5);fprintf(stderr,"Cargo build link sdk %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-link-inputs-only")==0){const bool passed=VerifyCargoBuildNativeCCChildBoundary(4);fprintf(stderr,"Cargo build link inputs %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-rust-lld-only")==0){const bool passed=VerifyCargoBuildNativeCCChildBoundary(3);fprintf(stderr,"Cargo build rust lld %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-lld-launcher-only")==0){const bool passed=VerifyCargoBuildNativeCCChildBoundary(2);fprintf(stderr,"Cargo build lld launcher %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-collect2-only")==0){const bool passed=VerifyCargoBuildNativeCCChildBoundary(true);fprintf(stderr,"Cargo build collect2 %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-native-cc-only")==0){const bool passed=VerifyCargoBuildNativeCCChildBoundary();fprintf(stderr,"Cargo build native cc %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-rustc-only")==0){const bool passed=VerifyCargoBuildRustcChildBoundary();fprintf(stderr,"Cargo build rustc %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-driver-only")==0){const bool passed=VerifyCargoBuildDriverRuntimeRead();fprintf(stderr,"Cargo build driver %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-handoff-only")==0){const bool passed=VerifyGoBuildHandoffReadBoundary(kProfileCargoBuild);fprintf(stderr,"Cargo build handoff %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-configuration-only")==0){const bool passed=VerifyGoBuildConfigurationReadBoundary(kProfileCargoBuild);fprintf(stderr,"Cargo build configuration %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-spoof-only")==0){const bool passed=VerifyCargoBuildArtifactSpoofDenied();fprintf(stderr,"Cargo build artifact spoof %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if(argc==2 && strcmp(argv[1],"--cargo-build-bootstrap-only")==0){const bool passed=VerifyCargoBuildBootstrapReadBoundary();fprintf(stderr,"Cargo build bootstrap %s\n",passed?"PASS":"FAIL");return passed?0:1;}
  if (argc == 2 && strcmp(argv[1], "--cargo-vm-metadata-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary("/proc/sys/vm/overcommit_memory", "/proc", "system");
    fprintf(stderr, "Cargo root VM metadata %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-utc-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary("/usr/share/zoneinfo/Etc/UTC");
    fprintf(stderr, "Cargo fixed UTC boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-cloexec-entropy-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary("/dev/urandom", "/dev", "system", 557056);
    fprintf(stderr, "Cargo close-on-exec entropy boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--socketpair-kernel-only") == 0) {
    const bool passed = VerifyKernelSocketPairTracking();
    fprintf(stderr, "Kernel SocketPair descriptor authority: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-resolver-file-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary("/etc/resolv.conf", "/etc/resolv.conf", "system") &&
        VerifyCargoResolverRuntimeReadBoundary("/etc/hosts", "/etc/hosts", "system");
    fprintf(stderr, "Cargo resolver configuration file boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-dns-configuration-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary("/etc/host.conf") &&
        VerifyCargoResolverRuntimeReadBoundary("/etc/gai.conf") &&
        VerifyCargoResolverRuntimeReadBoundary("/etc/nsswitch.conf");
    fprintf(stderr, "Cargo DNS configuration boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-sysroot-directory-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/lib", "/", "oci-root", true);
    fprintf(stderr, "Cargo Rustc fixed sysroot directory boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-cpu-quota-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/sys/fs/cgroup/cpu/cpu.cfs_quota_us", "/sys/fs/cgroup/cpu", "system", true) &&
        VerifyCargoRustcQueryBoundary("/sys/fs/cgroup/cpu/cpu.cfs_period_us", "/sys/fs/cgroup/cpu", "system", true);
    fprintf(stderr, "Cargo Rustc CPU quota boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-mountinfo-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/proc/100/mountinfo", "/proc", "system", true);
    fprintf(stderr, "Cargo Rustc self mountinfo boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-cgroup-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/proc/100/cgroup", "/proc", "system", true);
    fprintf(stderr, "Cargo Rustc self cgroup boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-metadata-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so", "/", "oci-root", true) &&
        VerifyCargoRustcQueryBoundary("/proc/100/maps", "/proc", "system", true, true);
    fprintf(stderr, "Cargo Rustc metadata query boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-statm-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/proc/100/statm", "/proc", "system");
    fprintf(stderr, "Cargo Rustc self memory stats boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-maps-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/proc/100/maps", "/proc", "system");
    fprintf(stderr, "Cargo Rustc self maps boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-overcommit-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/proc/sys/vm/overcommit_memory", "/proc", "system");
    fprintf(stderr, "Cargo Rustc VM metadata boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-zlib-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/usr/lib/x86_64-linux-gnu/libz.so.1.2.13");
    fprintf(stderr, "Cargo Rustc Zlib ABI boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-abi-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary("/usr/lib/x86_64-linux-gnu/librt.so.1") &&
        VerifyCargoRustcQueryBoundary("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/libLLVM.so.23.1-rust-1.99.0-stable");
    fprintf(stderr, "Cargo Rustc ABI boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-rustc-only") == 0) {
    const bool passed = VerifyCargoRustcQueryBoundary();
    fprintf(stderr, "Cargo Rustc query boundary: %s\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-null-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary("/dev/null", "/dev", "system");
    fprintf(stderr, "observer latch %s: Cargo null input read boundary\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-ca-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary("/etc/ssl/certs/ca-certificates.crt", "/", "oci-root", 32768);
    fprintf(stderr, "observer latch %s: Cargo certificate bundle read boundary\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-entropy-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary("/dev/urandom", "/dev", "system", 32768);
    fprintf(stderr, "observer latch %s: Cargo entropy read boundary\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-tar-only") == 0) {
    const bool passed = VerifyGoBuildConfigurationReadBoundary(kProfileCargoResolver, true);
    fprintf(stderr, "observer latch %s: Cargo TAR configuration read boundary\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-configuration-only") == 0) {
    const bool passed = VerifyGoBuildConfigurationReadBoundary(kProfileCargoResolver);
    fprintf(stderr, "observer latch %s: Cargo configuration read boundary\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-self-maps-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary("/proc/99/maps", "/proc", "system");
    fprintf(stderr, "observer latch %s: Cargo resolver self maps boundary\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--cargo-runtime-only") == 0) {
    const bool passed = VerifyCargoResolverRuntimeReadBoundary();
    fprintf(stderr, "observer latch %s: Cargo resolver runtime read boundary\n", passed ? "PASS" : "FAIL");
    return passed ? 0 : 1;
  }
  if (argc == 2 && strcmp(argv[1], "--semantic-only") == 0) {
    return HasPinnedPodInitProfile() && HasBoundedProfileRecordLimits() && VerifyBoundedClassificationSemantics() &&
        VerifyFilesystemClassification() && VerifyExactNpmNodeInterpreterTransition() &&
        VerifyUnexpectedExecDiagnosticRetention() && VerifyResolverNpmVersionNodeTransition() &&
        VerifyResolverNpmVersionProductionPath() && VerifyResolverLockGenerationProductionPath() &&
        VerifyResolverLockGenerationProductionPath(true) &&
        VerifyAuthoritativeProcessGroupLifecycle() && VerifyDirectExecAdmission() &&
        VerifyAdmissionTerminalControlProtocol() && VerifyKernelTerminalOutcomes() && VerifyStrictControlParser() ? 0 : 1;
  }
  if (argc != 1) return 2;
  char entry_name[] = "haa_gvisor_observer";
  char help_option[] = "--help";
  char* default_entry[] = {entry_name, nullptr};
  char* help_entry[] = {entry_name, help_option, nullptr};
  if (observer_main(1, default_entry) != 0 || observer_main(2, help_entry) != 0) {
    fprintf(stderr, "inert observer entry FAIL\n");
    return 1;
  }
  char directory[] = "/tmp/haa-observer-latch-XXXXXX";
  if (mkdtemp(directory) == nullptr) return 1;
  const std::string remote = std::string(directory) + "/remote.sock";
  const std::string output_path = std::string(directory) + "/output.sock";
  const std::string control = std::string(directory) + "/control.sock";
  gAdmissionControl = control;
  const int output = BindDatagram(output_path);
  int readiness[2];
  if (output < 0 || pipe(readiness) != 0) return 1;
  const pid_t child = fork();
  if (child == 0) {
    close(readiness[0]);
    char ready_arg[32]; snprintf(ready_arg, sizeof(ready_arg), "--ready-fd=%d", readiness[1]);
    char* arguments[] = {const_cast<char*>("haa_gvisor_observer"), const_cast<char*>(remote.c_str()), const_cast<char*>(output_path.c_str()), const_cast<char*>(control.c_str()), ready_arg, nullptr};
    _exit(observer_main(5, arguments));
  }
  close(readiness[1]);
  char ready = '\0';
  const bool running = child > 0 && read(readiness[0], &ready, 1) == 1 && ready == 'R';
  close(readiness[0]);
  const bool profile = HasPinnedPodInitProfile();
  const bool profile_limits = HasBoundedProfileRecordLimits();
	const bool unaccepted_disconnect = running && VerifyUnacceptedRemoteDisconnect(remote);
  fprintf(stderr, "STARTING record_boundaries\n");
  const bool record_boundaries = running &&
      VerifyProfileRecordBoundary(output, remote, control, "0101010101010101", kProfileGoResolver, 200000, false) &&
      VerifyProfileRecordBoundary(output, remote, control, "0202020202020202", kProfileGoResolver, 200000, true) &&
      VerifyProfileRecordBoundary(output, remote, control, "0505050505050505", kProfileGoBuild, 250000, false) &&
      VerifyProfileRecordBoundary(output, remote, control, "0606060606060606", kProfileGoBuild, 250000, true) &&
      VerifyProfileRecordBoundary(output, remote, control, "0707070707070707", kProfileCargoResolver, 10000, false) &&
      VerifyProfileRecordBoundary(output, remote, control, "0808080808080808", kProfileCargoResolver, 10000, true) &&
      VerifyProfileRecordBoundary(output, remote, control, "0303030303030303", kProfileNPM, 10000, false) &&
      VerifyProfileRecordBoundary(output, remote, control, "0404040404040404", kProfileNPM, 10000, true);
  fprintf(stderr, "STARTING accessors\n");
  const bool accessors = running && VerifyPinnedAccessors(output, remote, control);
  fprintf(stderr, "STARTING network\n");
  const bool network = running && VerifyNetworkFamilies(output, remote, control);
  fprintf(stderr, "STARTING malformed_socket\n");
  const bool malformed_socket = running && VerifyMalformedNetworkFamilies(output, remote, control);
  fprintf(stderr, "STARTING malformed_connect\n");
  const bool malformed_connect = running && VerifyMalformedConnect(output, remote, control);
  fprintf(stderr, "STARTING unknown_fd\n");
  const bool unknown_fd = running && VerifyUnknownFDState(output, remote, control);
  fprintf(stderr, "STARTING process\n");
  const bool process = running && VerifyProcessTrustBoundary(output, remote, control);
  fprintf(stderr, "STARTING correlation\n");
  const bool correlation = running && VerifySentryAuthoritativeBoundary(output, remote, control);
  fprintf(stderr, "STARTING cloexec\n");
  const bool cloexec = running && VerifyCloexecReexec(output, remote, control);
  fprintf(stderr, "STARTING delayed\n");
  const bool delayed = running && VerifyDelayedProfileRegistration(output, remote, control);
  fprintf(stderr, "STARTING concurrent_python_streams\n");
  const bool concurrent_python_streams = running && VerifyConcurrentPythonStreams(output, remote, control);
  fprintf(stderr, "STARTING roles\n");
  const bool roles = running && VerifyRoleHandoffAndCloneProvenance(output, remote, control);
  fprintf(stderr, "STARTING oci_bootstrap\n");
  const bool oci_bootstrap = running && VerifyOCIBootstrapDemotion(output, remote, control);
  fprintf(stderr, "STARTING demotion\n");
  const bool demotion = running && VerifyDemotionTransitionFaults(output, remote, control);
  fprintf(stderr, "STARTING mismatch\n");
  const bool mismatch = running && RunFaultCase(output, remote, control, true);
  fprintf(stderr, "STARTING dropped\n");
  const bool dropped = running && RunFaultCase(output, remote, control, false);
  fprintf(stderr, "STARTING topology_fail_closed\n");
  const bool topology_ok = running && VerifyTopologyFailClosed(output, remote, control);
  const bool nested_mounts = VerifyRegisteredNestedMountBoundary();
  fprintf(stderr, "STARTING filesystem\n");
  const bool filesystem = VerifyFilesystemClassification();
  if(!VerifyCargoBuildArtifactSpoofDenied() || !VerifyGoBuildConfigurationReadBoundary(kProfileCargoBuild)){fprintf(stderr,"Cargo build boundary FAIL\n");return 1;}
  if(!VerifyCargoBuildBootstrapReadBoundary()){fprintf(stderr,"Cargo build bootstrap FAIL\n");return 1;}
  const bool cargo_runtime_read = VerifyCargoResolverRuntimeReadBoundary();
  const bool cargo_self_maps_read = VerifyCargoResolverRuntimeReadBoundary("/proc/99/maps", "/proc", "system");
  const bool cargo_config_read = VerifyGoBuildConfigurationReadBoundary(kProfileCargoResolver);
  const bool cargo_tar_read = VerifyGoBuildConfigurationReadBoundary(kProfileCargoResolver, true);
  const bool cargo_entropy_read = VerifyCargoResolverRuntimeReadBoundary("/dev/urandom", "/dev", "system", 32768);
  const bool cargo_cloexec_entropy = VerifyCargoResolverRuntimeReadBoundary("/dev/urandom", "/dev", "system", 557056);
  const bool cargo_ca_read = VerifyCargoResolverRuntimeReadBoundary("/etc/ssl/certs/ca-certificates.crt", "/", "oci-root", 32768);
  const bool cargo_null_read = VerifyCargoResolverRuntimeReadBoundary("/dev/null", "/dev", "system");
  const bool cargo_rustc_query = VerifyCargoRustcQueryBoundary();
  const bool cargo_rustc_metadata = VerifyCargoRustcQueryBoundary("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/librustc_driver-999a121de2f042be.so", "/", "oci-root", true) &&
      VerifyCargoRustcQueryBoundary("/proc/100/maps", "/proc", "system", true, true);
  const bool cargo_rustc_overcommit = VerifyCargoRustcQueryBoundary("/proc/sys/vm/overcommit_memory", "/proc", "system");
  const bool cargo_rustc_maps = VerifyCargoRustcQueryBoundary("/proc/100/maps", "/proc", "system");
  const bool cargo_rustc_statm = VerifyCargoRustcQueryBoundary("/proc/100/statm", "/proc", "system");
  const bool cargo_rustc_cgroup = VerifyCargoRustcQueryBoundary("/proc/100/cgroup", "/proc", "system", true);
  const bool cargo_rustc_mountinfo = VerifyCargoRustcQueryBoundary("/proc/100/mountinfo", "/proc", "system", true);
  const bool cargo_rustc_cpu_quota = VerifyCargoRustcQueryBoundary("/sys/fs/cgroup/cpu/cpu.cfs_quota_us", "/sys/fs/cgroup/cpu", "system", true) &&
      VerifyCargoRustcQueryBoundary("/sys/fs/cgroup/cpu/cpu.cfs_period_us", "/sys/fs/cgroup/cpu", "system", true);
  const bool cargo_rustc_sysroot_directory = VerifyCargoRustcQueryBoundary("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/rustlib/x86_64-unknown-linux-gnu/lib", "/", "oci-root", true);
  const bool cargo_dns_configuration = VerifyCargoResolverRuntimeReadBoundary("/etc/host.conf") &&
      VerifyCargoResolverRuntimeReadBoundary("/etc/gai.conf") &&
      VerifyCargoResolverRuntimeReadBoundary("/etc/nsswitch.conf");
  const bool cargo_resolver_file = VerifyCargoResolverRuntimeReadBoundary("/etc/resolv.conf", "/etc/resolv.conf", "system") &&
      VerifyCargoResolverRuntimeReadBoundary("/etc/hosts", "/etc/hosts", "system");
  const bool cargo_vm_metadata = VerifyCargoResolverRuntimeReadBoundary("/proc/sys/vm/overcommit_memory", "/proc", "system");
  if (!cargo_vm_metadata) return 1;
  const bool cargo_utc = VerifyCargoResolverRuntimeReadBoundary("/usr/share/zoneinfo/Etc/UTC");
  const bool socketpair_diagnostic = VerifySocketPairDiagnosticPreservesFDAuthority();
  const bool kernel_socketpair = VerifyKernelSocketPairTracking();
  const bool fd_mutation_diagnostic = VerifyFDMutationDiagnosticPreservesAuthority();
  const bool cargo_rustc_zlib = VerifyCargoRustcQueryBoundary("/usr/lib/x86_64-linux-gnu/libz.so.1.2.13");
  const bool cargo_rustc_abi = VerifyCargoRustcQueryBoundary("/usr/lib/x86_64-linux-gnu/librt.so.1") &&
      VerifyCargoRustcQueryBoundary("/usr/local/rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu/lib/libLLVM.so.23.1-rust-1.99.0-stable");
  const bool go_runtime_read = VerifyGoResolverRuntimeReadBoundary();
  const bool github_artifact_runtime_read = VerifyGitHubArtifactRuntimeReadBoundary();
  const bool pipx_runtime_read = VerifyPipxRuntimeReadBoundary();
  const bool pipx_native_root = VerifyPipxNativeRootAndTopologyBoundary();
  const bool pipx_native_clone = VerifyPipxNativeCloneAndQueryBoundary();
  const bool pipx_native_read = VerifyPipxNativeReadAndReexecBoundary() && VerifyPipxNativeUVProcReadBoundary() && VerifyPipxNativeBaseQueryDirectoryRead() && VerifyPipxNativeFirstOutsideDiagnostic();
  const bool pipx_native_admission = running && VerifyPipxNativeProfileAdmission(control);
  const bool pipx_runtime_admission = running && VerifyPipxRuntimeAdmission(control);
  const bool go_build_config_read = VerifyGoBuildConfigurationReadBoundary();
  const bool go_build_handoff_read = VerifyGoBuildHandoffReadBoundary();
  const bool cargo_build_handoff_read = VerifyGoBuildHandoffReadBoundary(kProfileCargoBuild);
  const bool cargo_build_pinned_target = VerifyCargoBuildHandoffPinnedTarget();
  const bool cargo_build_driver_read = VerifyCargoBuildDriverRuntimeRead();
  const bool cargo_build_rustc_child = VerifyCargoBuildRustcChildBoundary();
  const bool cargo_build_native_cc_child = VerifyCargoBuildNativeCCChildBoundary();
  const bool cargo_build_collect2_child = VerifyCargoBuildNativeCCChildBoundary(true);
  const bool cargo_build_lld_launcher = VerifyCargoBuildNativeCCChildBoundary(2);
  const bool cargo_build_rust_lld = VerifyCargoBuildNativeCCChildBoundary(3);
  const bool cargo_build_link_inputs = VerifyCargoBuildNativeCCChildBoundary(4);
  const bool cargo_build_link_sdk = VerifyCargoBuildNativeCCChildBoundary(5);
  const bool cargo_build_rust_c_abi = VerifyCargoBuildNativeCCChildBoundary(6);
  const bool cargo_build_lld_entropy = VerifyCargoBuildNativeCCChildBoundary(7);
  const bool cargo_build_program = VerifyCargoBuildProgramBoundary();
  const bool cargo_build_program_abi = VerifyCargoBuildProgramBoundary(true);
  const bool cargo_build_program_maps = VerifyCargoBuildProgramBoundary(true,true);
  const bool cargo_build_program_query = VerifyCargoBuildProgramRustcQueryBoundary();
  const bool cargo_build_program_null = VerifyCargoBuildProgramBoundary(true,true,true);
  const bool cargo_build_config = VerifyCargoBuildConfigurationBoundary();
  const bool go_build_runtime_read = VerifyGoBuildRuntimeMetadataBoundary();
  const bool go_build_creator_diagnostic = VerifyGoBuildCreatorDiagnosticBoundary();
  const bool go_build_tool_child = VerifyGoBuildToolChildBoundary();
  const bool go_build_cgo_gcc_creator = VerifyGoBuildCgoGccCreatorBoundary();
  const bool go_build_sdk_link_gcc = VerifyGoBuildCgoGccCreatorBoundary(true);
  const bool go_build_sdk_link_native = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kNativeLinkerSDKLink);
  const bool go_build_linker_resolver = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kNativeLinkerResolver);
  const bool go_build_cc1 = VerifyGoBuildGccChildBoundary();
  const bool go_build_cc1_headers = VerifyGoBuildGccChildBoundary(true);
  const bool go_build_as = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kAssembler);
  const bool go_build_collect2 = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kCollect2);
  const bool go_build_native_linker = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kNativeLinker);
  const bool go_build_linker_plugin = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kNativeLinkerPlugin);
  const bool go_build_linker_null = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kNativeLinkerNull);
  const bool go_build_linker_runtime = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kNativeLinkerRuntime);
  const bool go_build_as_null = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kAssemblerNull);
  const bool go_build_cc1_sdk_cgo = VerifyGoBuildGccChildBoundary(false, GoBuildGccChildFixture::kCc1SDKCgo);
  fprintf(stderr, "STARTING npm_node\n");
  const bool npm_node = VerifyExactNpmNodeInterpreterTransition();
  const bool pnpm_target_shape = VerifyPNPMTargetLifecycleShapeBoundary();
  const bool pnpm_target_production = VerifyPNPMTargetLifecycleProductionPath();
  const bool bun_artifact_reads = VerifyBunArtifactRuntimeReadBoundary();
  const bool bun_artifact_image = VerifyBunArtifactRuntimeImageAndCommandBoundary();
  const bool bun_target_shape = VerifyBunTargetLifecycleShapeBoundary();
  const bool bun_target_production = VerifyBunTargetLifecycleProductionPath();
  fprintf(stderr, "STARTING open_result_negative\n");
  const bool open_result_negative_ok = running && VerifyOpenResultNegativeMatrix(output, remote, control);
  fprintf(stderr, "STARTING open_result_positive\n");
  const bool open_result_positive_ok = running && VerifyOpenResultPositiveMatrix(output, remote, control);
  fprintf(stderr, "STARTING no_basename_trust\n");
  const bool no_basename_trust_ok = running && VerifyNoBasenameTrust(output, remote, control);
  fprintf(stderr, "STARTING unexpected_exec_diagnostic\n");
  const bool unexpected_exec_diagnostic_ok = VerifyUnexpectedExecDiagnosticRetention();
  fprintf(stderr, "STARTING resolver_npm_version_node\n");
  const bool resolver_npm_version_node = VerifyResolverNpmVersionNodeTransition();
  fprintf(stderr, "STARTING resolver_npm_version_production_path\n");
  const bool resolver_npm_version_production_path = VerifyResolverNpmVersionProductionPath();
  fprintf(stderr, "STARTING resolver_lock_generation_production_path\n");
  const bool resolver_lock_generation_production_path = VerifyResolverLockGenerationProductionPath() &&
      VerifyResolverLockGenerationProductionPath(true);
  fprintf(stderr, "STARTING process_group_lifecycle\n");
  const bool process_group_lifecycle = VerifyAuthoritativeProcessGroupLifecycle();
  fprintf(stderr, "STARTING direct_exec_admission\n");
  const bool direct_exec_admission = VerifyDirectExecAdmission();
	const bool admission_terminal_protocol = VerifyAdmissionTerminalControlProtocol();
  const bool kernel_terminal_outcomes = VerifyKernelTerminalOutcomes();
  const bool python_execution_boundary = VerifyPythonExecutionBoundary();
  const bool node_execution_boundary = VerifyNodeExecutionBoundary();
  const bool kernel_target_load = VerifyAdmissionTerminalControlProtocol(true, 0, true, false) &&
      VerifyAdmissionTerminalControlProtocol(true, 37 << 8, true, true);
  fprintf(stderr, "STARTING direct_exec_admission_bound\n");
  const bool direct_exec_admission_bound = running && VerifyDirectExecAdmissionBound(output, remote, control);
  fprintf(stderr, "STARTING session_aware_control\n");
  const bool session_aware_control = running && VerifySessionAwareControl(control);
	const bool exact_control_peer = running && VerifyExactControlPeerBinding(output, remote, control);
	fprintf(stderr, "STARTING control_peer_hup_live_admission\n");
	const bool control_peer_hup_live_admission = running && VerifyControlPeerHUPClearsLiveAdmission(control);
	fprintf(stderr, "STARTING pre_attribution_disconnect\n");
	const bool pre_attribution_disconnect = running && VerifyAcceptedPreAttributionDisconnect(remote, child);
  const bool terraform_profile = VerifyTerraformProviderProfileBoundary();
  const bool uv_python_query = VerifyUVPythonQueryBoundary();
  const bool pnpm_node_query = VerifyPNPMNodeQueryBoundary();
  const bool pnpm_node_query_read = VerifyPNPMNodeQueryReadBoundary();
  const bool uv_runtime_admission = VerifyUVRuntimeAdmissionBoundary();
  const bool pnpm_runtime_admission = VerifyPNPMRuntimeAdmissionBoundary();
  const bool bun_runtime_admission = VerifyPrivateNativeRuntimeAdmissionBoundary(false, true);
  const bool bun_query_read = VerifyBunQueryReadBoundary();
  const bool bun_project_read = VerifyBunQueryReadBoundary(kProfileBunProjectResolver);
  const bool bun_project_admission = VerifyPrivateNativeRuntimeAdmissionBoundary(false, true, true) &&
      VerifyPrivateNativeRuntimeAdmissionBoundary(false,true,true,"install",true,false) &&
      VerifyPrivateNativeRuntimeAdmissionBoundary(false,true,true,"add",false,true) &&
      VerifyPrivateNativeRuntimeAdmissionBoundary(false,true,true,"update",false,false) &&
      VerifyPrivateNativeRuntimeAdmissionBoundary(false,true,true,"update",false,true) &&
      VerifyPrivateNativeRuntimeAdmissionBoundary(false,true,true,"install",false,false,"seed") &&
      VerifyPrivateNativeRuntimeAdmissionBoundary(false,true,true,"install",false,false,"offline");
  const bool bun_project_shape = VerifyBunProjectLockInstallShapeBoundary();
  const bool pnpm_runtime_read = VerifyPNPMRuntimeReadBoundary();
  const bool uv_runtime_read = VerifyUVRuntimeReadBoundary();
  const bool passed = pipx_native_root && pipx_native_clone && pipx_native_read && pipx_native_admission && pipx_runtime_read && pipx_runtime_admission && bun_artifact_reads && bun_artifact_image && pnpm_target_shape && pnpm_target_production && pnpm_node_query_read && pnpm_node_query && pnpm_runtime_read && pnpm_runtime_admission && bun_runtime_admission && bun_query_read && bun_project_read && bun_project_admission && bun_project_shape && uv_python_query && uv_runtime_read && uv_runtime_admission && github_artifact_runtime_read && terraform_profile && running && profile && profile_limits && record_boundaries && accessors && network && malformed_socket &&
      malformed_connect && unknown_fd && process && correlation && cloexec && delayed && concurrent_python_streams && roles &&
      oci_bootstrap && demotion && mismatch && dropped && topology_ok && nested_mounts && filesystem && cargo_runtime_read && cargo_self_maps_read && cargo_config_read && cargo_tar_read && cargo_entropy_read && cargo_ca_read && cargo_null_read && cargo_rustc_query && cargo_rustc_metadata && cargo_rustc_abi && cargo_rustc_zlib && cargo_rustc_overcommit && cargo_rustc_maps && cargo_rustc_statm && cargo_rustc_cgroup && go_runtime_read && go_build_config_read && go_build_handoff_read && cargo_build_handoff_read && cargo_build_pinned_target && cargo_build_driver_read && cargo_build_rustc_child && cargo_build_native_cc_child && cargo_build_collect2_child && cargo_build_lld_launcher && cargo_build_rust_lld && cargo_build_link_inputs && cargo_build_link_sdk && cargo_build_rust_c_abi && cargo_build_lld_entropy && cargo_build_program && cargo_build_program_abi && cargo_build_program_maps && cargo_build_program_query && cargo_build_program_null && cargo_build_config && go_build_runtime_read && go_build_creator_diagnostic && go_build_tool_child && go_build_cgo_gcc_creator && go_build_sdk_link_gcc && go_build_sdk_link_native && go_build_linker_resolver && go_build_cc1 && go_build_cc1_headers && go_build_as && go_build_collect2 && go_build_native_linker && go_build_linker_plugin && go_build_linker_null && go_build_linker_runtime && go_build_as_null && go_build_cc1_sdk_cgo && npm_node &&
      cargo_rustc_mountinfo && cargo_rustc_cpu_quota && cargo_rustc_sysroot_directory && cargo_dns_configuration && cargo_resolver_file && cargo_utc && cargo_cloexec_entropy && socketpair_diagnostic && kernel_socketpair && fd_mutation_diagnostic && open_result_negative_ok && open_result_positive_ok && no_basename_trust_ok &&
      unexpected_exec_diagnostic_ok && resolver_npm_version_node && resolver_npm_version_production_path &&
      resolver_lock_generation_production_path && process_group_lifecycle && direct_exec_admission &&
		admission_terminal_protocol && kernel_terminal_outcomes && kernel_target_load && python_execution_boundary && node_execution_boundary && direct_exec_admission_bound && session_aware_control && exact_control_peer && control_peer_hup_live_admission && unaccepted_disconnect && pre_attribution_disconnect;
  const std::pair<const char*, bool> latches[] = {
      {"Pipx native fixed root and six-mount boundary",pipx_native_root},
      {"Pipx native ARTIFACT clone and query boundary",pipx_native_clone},
      {"Pipx native readonly inputs and query reexec boundary",pipx_native_read},
      {"Pipx native control registration/readiness only",pipx_native_admission},
      {"Pipx fixed ZIP runtime and read boundary",pipx_runtime_read},
      {"Pipx ARTIFACT-only Python admission",pipx_runtime_admission},
      {"Bun Artifact exact native runtime reads", bun_artifact_reads},
      {"Bun Artifact native image and command boundary", bun_artifact_image},
      {"UV exact interpreter query boundary", uv_python_query},
      {"PNPM exact Node query boundary", pnpm_node_query},
      {"PNPM Node query ABI boundary", pnpm_node_query_read},
      {"UV private tool admission boundary", uv_runtime_admission},
      {"PNPM private tool admission boundary", pnpm_runtime_admission},
      {"Bun private tool and exact query admission boundary", bun_runtime_admission},
      {"Bun exact query overcommit read boundary", bun_query_read},
      {"Bun project startup read boundary", bun_project_read},
      {"Bun exact project lock admission boundary", bun_project_admission},
      {"Bun project lock argv closed shape", bun_project_shape},
      {"PNPM exact GNU startup read boundary", pnpm_runtime_read},
      {"UV exact native startup read boundary", uv_runtime_read},
      {"readiness", running}, {"pod-init profile", profile},
      {"profile record limits", profile_limits}, {"finite profile record boundary", record_boundaries}, {"normalized accessors", accessors},
      {"socket family classification", network}, {"unknown socket family", malformed_socket},
      {"malformed socket address", malformed_connect}, {"unknown FD state", unknown_fd},
      {"process trust boundary", process}, {"exec correlation boundary", correlation},
      {"CLOEXEC/re-exec boundary", cloexec}, {"delayed profile registration", delayed},
      {"concurrent Python streams", concurrent_python_streams},
      {"role handoff/provenance", roles}, {"OCI bootstrap demotion", oci_bootstrap},
      {"setpriv demotion boundary", demotion}, {"container mismatch", mismatch},
      {"dropped events", dropped}, {"topology fail-closed", topology_ok},
      {"registered nested mount boundary", nested_mounts},
      {"filesystem normalization", filesystem},
      {"Cargo resolver runtime read boundary", cargo_runtime_read},
      {"Cargo resolver self maps boundary", cargo_self_maps_read},
      {"Cargo configuration read boundary", cargo_config_read},
      {"Cargo TAR configuration read boundary", cargo_tar_read},
      {"Cargo entropy read boundary", cargo_entropy_read},
      {"Cargo certificate bundle read boundary", cargo_ca_read},
      {"Cargo null input read boundary", cargo_null_read},
      {"Cargo Rustc query boundary", cargo_rustc_query},
      {"Cargo Rustc metadata query boundary", cargo_rustc_metadata},
      {"Cargo Rustc VM metadata boundary", cargo_rustc_overcommit},
      {"Cargo Rustc self maps boundary", cargo_rustc_maps},
      {"Cargo Rustc self memory stats boundary", cargo_rustc_statm},
      {"Cargo Rustc self cgroup boundary", cargo_rustc_cgroup},
      {"Cargo Rustc self mountinfo boundary", cargo_rustc_mountinfo},
      {"Cargo Rustc CPU quota boundary", cargo_rustc_cpu_quota},
      {"Cargo Rustc fixed sysroot directory boundary", cargo_rustc_sysroot_directory},
      {"Cargo DNS configuration boundary", cargo_dns_configuration},
      {"Cargo resolver configuration file boundary", cargo_resolver_file},
      {"Cargo fixed UTC boundary", cargo_utc},
      {"Cargo close-on-exec entropy boundary", cargo_cloexec_entropy},
      {"SocketPair diagnostic preserves FD authority", socketpair_diagnostic},
      {"Kernel SocketPair descriptor authority", kernel_socketpair},
      {"FD mutation diagnostic preserves authority", fd_mutation_diagnostic},
      {"Cargo Rustc Zlib ABI boundary", cargo_rustc_zlib},
      {"Cargo Rustc ABI boundary", cargo_rustc_abi},
      {"Go resolver runtime read boundary", go_runtime_read},
      {"GitHub artifact runtime read boundary", github_artifact_runtime_read},
      {"Go build configuration read boundary", go_build_config_read},
      {"Go build handoff read boundary", go_build_handoff_read},
      {"Cargo build handoff read boundary", cargo_build_handoff_read},
      {"Cargo build pinned target", cargo_build_pinned_target},
      {"Cargo build driver read", cargo_build_driver_read},
      {"Cargo build rustc child", cargo_build_rustc_child},
      {"Cargo build native cc child", cargo_build_native_cc_child},
      {"Cargo build collect2 child", cargo_build_collect2_child},
      {"Cargo build lld launcher", cargo_build_lld_launcher},
      {"Cargo build rust lld", cargo_build_rust_lld},
      {"Cargo build link inputs", cargo_build_link_inputs},
      {"Cargo build link sdk", cargo_build_link_sdk},
      {"Cargo build Rust C ABI", cargo_build_rust_c_abi},
      {"Cargo build lld entropy", cargo_build_lld_entropy},
      {"Cargo build program", cargo_build_program},
      {"Cargo build program ABI", cargo_build_program_abi},
      {"Cargo build program maps", cargo_build_program_maps},
      {"Cargo build program rustc query", cargo_build_program_query},
      {"Cargo build program null", cargo_build_program_null},
      {"Cargo build configuration", cargo_build_config},
      {"Terraform provider profile isolation", terraform_profile},
      {"Go build runtime metadata boundary", go_build_runtime_read},
      {"Go build exact creator diagnostic boundary", go_build_creator_diagnostic},
      {"Go build SDK tool child boundary", go_build_tool_child},
      {"Go build cgo GCC creator boundary", go_build_cgo_gcc_creator},
      {"Go build SDK linker GCC creator boundary", go_build_sdk_link_gcc},
      {"Go build SDK linker native ancestry boundary", go_build_sdk_link_native},
      {"Go build native linker resolver ABI boundary", go_build_linker_resolver},
      {"Go build CC1 creator and loader boundary", go_build_cc1},
      {"Go build CC1 readonly header boundary", go_build_cc1_headers},
      {"Go build assembler creator and loader boundary", go_build_as},
      {"Go build collect2 creator and loader boundary", go_build_collect2},
      {"Go build native linker creator and loader boundary", go_build_native_linker},
      {"Go build native linker fixed plugin boundary", go_build_linker_plugin},
      {"Go build native linker null device boundary", go_build_linker_null},
      {"Go build native linker C runtime input boundary", go_build_linker_runtime},
      {"Go build assembler null device boundary", go_build_as_null},
      {"Go build CC1 readonly SDK Cgo input boundary", go_build_cc1_sdk_cgo},
      {"exact npm CLI-to-Node", npm_node},
      {"PNPM target lifecycle shape and topology", pnpm_target_shape},
      {"PNPM target lifecycle production lineage", pnpm_target_production},
      {"Bun target lifecycle shape and topology", bun_target_shape},
      {"Bun target lifecycle production lineage", bun_target_production},
      {"OPEN_RESULT negative matrix", open_result_negative_ok},
      {"OPEN_RESULT positive matrix", open_result_positive_ok},
      {"no-basename-trust", no_basename_trust_ok},
      {"unexpected exec diagnostic retention", unexpected_exec_diagnostic_ok},
      {"exact resolver npm-version-to-Node", resolver_npm_version_node},
      {"resolver npm-version production path", resolver_npm_version_production_path},
      {"resolver lock-generation production path", resolver_lock_generation_production_path},
      {"authoritative process-group lifecycle", process_group_lifecycle},
      {"direct-exec one-shot admission", direct_exec_admission},
		{"direct-exec terminal admission protocol", admission_terminal_protocol},
      {"kernel direct-exec terminal outcomes", kernel_terminal_outcomes},
      {"kernel direct-exec accepted target image", kernel_target_load},
      {"Artifact Python selected console execution boundary", python_execution_boundary},
      {"Artifact Node readonly execution boundary", node_execution_boundary},
      {"direct-exec admission bound", direct_exec_admission_bound},
		{"session-aware control", session_aware_control},
		{"exact control peer binding", exact_control_peer},
		{"control peer HUP clears live admission", control_peer_hup_live_admission},
		{"unaccepted remote disconnect", unaccepted_disconnect},
		{"accepted pre-attribution disconnect fail-stop", pre_attribution_disconnect},
  };
  for (const auto& latch : latches) {
    fprintf(stderr, "observer latch %s: %s\n", latch.second ? "PASS" : "FAIL", latch.first);
  }
  if (child > 0) { kill(child, SIGTERM); waitpid(child, nullptr, 0); }
  close(output); unlink(remote.c_str()); unlink(output_path.c_str()); unlink(control.c_str()); rmdir(directory);
  return passed ? 0 : 1;
}
