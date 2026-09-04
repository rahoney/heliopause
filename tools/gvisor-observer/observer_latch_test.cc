// Exercises the exact pinned remote-sink protobuf framing through the helper.
// It deliberately checks only normalized records and never retains payloads.
#define main observer_main
#include "tools/haa_gvisor_observer/observer.cc"
#undef main

#include <chrono>
#include <signal.h>
#include <fstream>
#include <iterator>
#include <sys/time.h>
#include <sys/wait.h>
#include <thread>
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

bool ExpectRecordExact(int output, const char* container_id, const char* kind, const char* reason = nullptr);
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

int ConnectRemote(const std::string& path) {
  int fd = socket(AF_UNIX, SOCK_SEQPACKET, 0);
  if (fd < 0) return -1;
  sockaddr_un address{}; address.sun_family = AF_UNIX;
  if (path.size() >= sizeof(address.sun_path)) return -1;
  strcpy(address.sun_path, path.c_str());
  return connect(fd, reinterpret_cast<sockaddr*>(&address), sizeof(address)) == 0 ? fd : -1;
}

static std::map<std::string, std::string> gRegisteredProfiles;

bool RegisterProfile(const std::string& path, const char* container_id, const char* profile) {
  gRegisteredProfiles[container_id] = profile;
  const int fd = socket(AF_UNIX, SOCK_DGRAM, 0);
  if (fd < 0) return false;
  sockaddr_un address{}; address.sun_family = AF_UNIX;
  if (path.size() >= sizeof(address.sun_path)) return false;
  strcpy(address.sun_path, path.c_str());
  std::string topology = "/|oci-root|/||1|0|0|0;/tmp|workspace|/|tmpfs|0|1|1|0;/haa-runtime|helper|/|tmpfs|0|0|1|0";
  if (strcmp(profile, kProfilePyPI) == 0 || strcmp(profile, kProfilePyTorchCPU) == 0 || strcmp(profile, kProfilePyTorchCU126) == 0) {
    topology += ";/haa-site|workspace|/|tmpfs|0|0|1|0";
  } else if (strcmp(profile, kProfileGitHub) == 0) {
    topology += ";/work|workspace|/|tmpfs|0|0|1|0";
  }
  const std::string body = std::string("{\"container_id\":\"") + container_id + "\",\"profile\":\"" + profile +
      "\",\"expected_topology\":\"" + topology + "\"}";
  const bool sent = sendto(fd, body.data(), body.size(), 0, reinterpret_cast<sockaddr*>(&address), sizeof(address)) == static_cast<ssize_t>(body.size());
  close(fd);
  return sent;
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
  if (strcmp(profile, kProfilePyPI) == 0 || strcmp(profile, kProfilePyTorchCPU) == 0 || strcmp(profile, kProfilePyTorchCU126) == 0) {
    add_mount(next_id++, "/haa-site", "tmpfs", false, false);
  } else if (strcmp(profile, kProfileGitHub) == 0) {
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

template <typename Message>
bool SendEvent(int client, gvisor::common::MessageType type, const Message& message, uint32_t dropped_count = 0) {
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

bool ExpectRecordExact(int output, const char* container_id, const char* kind, const char* reason) {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  if (size <= 0) return false;
  std::string expected = std::string("{\"container_id\":\"") + container_id + "\",\"kind\":\"" + kind + "\"";
  if (reason != nullptr) expected += ",\"reason\":\"" + std::string(reason) + "\"";
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
  return profile.find("syscall/execve/enter") != std::string::npos &&
      profile.find("syscall/openat/enter") != std::string::npos &&
      profile.find("syscall/connect/enter") != std::string::npos &&
      profile.find("syscall/socket/enter") != std::string::npos &&
      profile.find("syscall/execve/exit") != std::string::npos &&
      profile.find("syscall/execveat/enter") != std::string::npos &&
      profile.find("syscall/execveat/exit") != std::string::npos &&
      profile.find("syscall/socket/exit") != std::string::npos &&
      profile.find("syscall/close/exit") != std::string::npos &&
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
      profile.find("sentry/mount_topology_snapshot") != std::string::npos &&
      profile.find("sentry/mount_topology_mutation") != std::string::npos &&
      profile.find("ignore_missing") == std::string::npos;
}

bool HasBoundedProfileRecordLimits() {
  return MaximumRecords(kProfilePyTorchCPU) == 500000 &&
      MaximumRecords(kProfilePyTorchCU126) == 100000 &&
      MaximumRecords(kProfilePyPI) == 10000 &&
      MaximumRecords(nullptr) == 10000;
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
  close_socket.mutable_exit()->set_result(0);
  close_socket.mutable_exit()->set_errorno(0);
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
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) || !ExpectRecord(output, kEighthID, "stream-fault", "FD_STATE_UNKNOWN")) return false;
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
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) || !ExpectRecord(output, kNinthID, "stream-fault", "FD_STATE_UNKNOWN")) return false;
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
    if (!ExpectRecordExact(output, kFiftyFirstID, "stream-fault", "STREAM_FAULT")) { close(client); return false; }
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

}  // namespace

int main(int argc, char** argv) {
  if (argc == 2 && strcmp(argv[1], "--semantic-only") == 0) {
    return HasPinnedPodInitProfile() && HasBoundedProfileRecordLimits() && VerifyBoundedClassificationSemantics() &&
        VerifyFilesystemClassification() && VerifyExactNpmNodeInterpreterTransition() ? 0 : 1;
  }
  if (argc != 1) return 2;
  char directory[] = "/tmp/haa-observer-latch-XXXXXX";
  if (mkdtemp(directory) == nullptr) return 1;
  const std::string remote = std::string(directory) + "/remote.sock";
  const std::string output_path = std::string(directory) + "/output.sock";
  const std::string control = std::string(directory) + "/control.sock";
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
  fprintf(stderr, "STARTING filesystem\n");
  const bool filesystem = VerifyFilesystemClassification();
  fprintf(stderr, "STARTING npm_node\n");
  const bool npm_node = VerifyExactNpmNodeInterpreterTransition();
  fprintf(stderr, "STARTING open_result_negative\n");
  const bool open_result_negative_ok = running && VerifyOpenResultNegativeMatrix(output, remote, control);
  fprintf(stderr, "STARTING open_result_positive\n");
  const bool open_result_positive_ok = running && VerifyOpenResultPositiveMatrix(output, remote, control);
  fprintf(stderr, "STARTING no_basename_trust\n");
  const bool no_basename_trust_ok = running && VerifyNoBasenameTrust(output, remote, control);
  const bool passed = running && profile && profile_limits && accessors && network && malformed_socket &&
      malformed_connect && unknown_fd && process && correlation && cloexec && delayed && roles &&
      oci_bootstrap && demotion && mismatch && dropped && topology_ok && filesystem && npm_node &&
      open_result_negative_ok && open_result_positive_ok && no_basename_trust_ok;
  const std::pair<const char*, bool> latches[] = {
      {"readiness", running}, {"pod-init profile", profile},
      {"profile record limits", profile_limits}, {"normalized accessors", accessors},
      {"socket family classification", network}, {"unknown socket family", malformed_socket},
      {"malformed socket address", malformed_connect}, {"unknown FD state", unknown_fd},
      {"process trust boundary", process}, {"exec correlation boundary", correlation},
      {"CLOEXEC/re-exec boundary", cloexec}, {"delayed profile registration", delayed},
      {"role handoff/provenance", roles}, {"OCI bootstrap demotion", oci_bootstrap},
      {"setpriv demotion boundary", demotion}, {"container mismatch", mismatch},
      {"dropped events", dropped}, {"topology fail-closed", topology_ok},
      {"filesystem normalization", filesystem},
      {"exact npm CLI-to-Node", npm_node},
      {"OPEN_RESULT negative matrix", open_result_negative_ok},
      {"OPEN_RESULT positive matrix", open_result_positive_ok},
      {"no-basename-trust", no_basename_trust_ok},
  };
  for (const auto& latch : latches) {
    fprintf(stderr, "observer latch %s: %s\n", latch.second ? "PASS" : "FAIL", latch.first);
  }
  if (child > 0) { kill(child, SIGTERM); waitpid(child, nullptr, 0); }
  close(output); unlink(remote.c_str()); unlink(output_path.c_str()); unlink(control.c_str()); rmdir(directory);
  return passed ? 0 : 1;
}
