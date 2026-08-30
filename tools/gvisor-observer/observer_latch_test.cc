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

bool SendAll(int fd, const std::string& value) {
  return send(fd, value.data(), value.size(), 0) == static_cast<ssize_t>(value.size());
}

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

bool RegisterProfile(const std::string& path, const char* container_id, const char* profile) {
  const int fd = socket(AF_UNIX, SOCK_DGRAM, 0);
  if (fd < 0) return false;
  sockaddr_un address{}; address.sun_family = AF_UNIX;
  if (path.size() >= sizeof(address.sun_path)) return false;
  strcpy(address.sun_path, path.c_str());
  const std::string body = std::string("{\"container_id\":\"") + container_id + "\",\"profile\":\"" + profile + "\"}";
  const bool sent = sendto(fd, body.data(), body.size(), 0, reinterpret_cast<sockaddr*>(&address), sizeof(address)) == static_cast<ssize_t>(body.size());
  close(fd);
  return sent;
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
               const gvisor::container::Start& original, uint32_t dropped_count = 0) {
  gvisor::container::Start message = original;
  auto* context = message.mutable_context_data();
  if (context->thread_group_id() == 0) context->set_thread_group_id(1);
  if (context->thread_group_start_time_ns() == 0) context->set_thread_group_start_time_ns(1);
	context->set_parent_thread_group_id(0);
  return SendEvent<gvisor::container::Start>(client, type, message, dropped_count);
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
  launcher.add_argv(boundaryLaunchMode);
  if (!SendSuccessfulExec(client, enter, launcher) ||
      !ExpectRecord(output, enter.context_data().container_id().c_str(), "process-exec-expected")) return false;
  return SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, target);
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
  return SendEvent(client, gvisor::common::MESSAGE_SENTRY_CLONE, clone) &&
      ExpectRecord(output, container_id, "process-clone");
}

bool ExpectRecord(int output, const char* container_id, const char* kind, const char* reason) {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  if (size <= 0) return false;
  std::string expected = std::string("{\"container_id\":\"") + container_id + "\",\"kind\":\"" + kind + "\"";
  if (reason != nullptr) expected += ",\"reason\":\"" + std::string(reason) + "\"";
  expected += "}";
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
  open.mutable_context_data()->set_container_id(kFirstID);
  open.mutable_context_data()->set_thread_group_id(7);
  open.mutable_context_data()->set_process_name("trusted-test-process");
  open.set_pathname("/root/.ssh/authorized_keys");
  open.set_flags(1); open.set_mode(0600); open.set_sysno(257);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_OPEN, open) || !ExpectRecord(output, kFirstID, "filesystem-outside-workspace")) return false;
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
  return ExpectRecord(output, kFirstID, "stream-end");
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
      !ExpectRecord(output, kFifthID, "process-clone")) return false;
  *raw.mutable_context_data() = socket.context_data();
  raw.mutable_context_data()->set_thread_group_id(51);
  raw.mutable_context_data()->set_thread_group_start_time_ns(510);
  raw.mutable_context_data()->set_parent_thread_group_id(50);
  raw.set_arg1(4);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectNetworkRecord(output, kFifthID, "SENDMSG", "INET", "UNKNOWN", "PYTHON")) return false;
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
      !ExpectRecord(output, kFifthID, "process-clone")) return false;
  raw.mutable_context_data()->set_thread_group_id(52);
  raw.mutable_context_data()->set_thread_group_start_time_ns(520);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_RAW, raw) ||
      !ExpectNetworkRecord(output, kFifthID, "SENDMSG", "INET", "UNKNOWN", "PYTHON")) return false;

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

bool VerifySocketFault(int output, const std::string& remote, const std::string& control, int domain, const char* reason) {
  if (!RegisterProfile(control, kSixthID, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kSixthID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kSixthID, "container-start")) return false;
  gvisor::syscall::Socket socket;
  socket.mutable_context_data()->set_container_id(kSixthID);
  socket.mutable_context_data()->set_thread_group_id(1);
  socket.mutable_context_data()->set_thread_group_start_time_ns(1);
  socket.set_domain(domain);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket) || !ExpectRecord(output, kSixthID, "stream-fault", reason)) return false;
  close(client);
  return true;
}

bool VerifyMalformedNetworkFamilies(int output, const std::string& remote, const std::string& control) {
  return VerifySocketFault(output, remote, control, AF_UNSPEC, "SOCKET_AF_UNSPEC") &&
      VerifySocketFault(output, remote, control, 0x7fff, "SOCKET_OTHER_FAMILY");
}

bool VerifyConnectFault(int output, const std::string& remote, const std::string& control, const std::string& address, const char* reason) {
  if (!RegisterProfile(control, kSeventhID, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kSeventhID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kSeventhID, "container-start")) return false;
  gvisor::syscall::Connect connect;
  connect.mutable_context_data()->set_container_id(kSeventhID);
  connect.mutable_context_data()->set_thread_group_id(1);
  connect.mutable_context_data()->set_thread_group_start_time_ns(1);
  connect.set_address(address);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectRecord(output, kSeventhID, "stream-fault", reason)) return false;
  close(client);
  return true;
}

bool VerifyMalformedConnect(int output, const std::string& remote, const std::string& control) {
  return VerifyConnectFault(output, remote, control, "\x02", "CONNECT_ADDRESS_TOO_SHORT") &&
      VerifyConnectFault(output, remote, control, SocketAddress(AF_UNIX, sizeof(sockaddr_un) + 1), "CONNECT_AF_UNIX_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, SocketAddress(AF_INET, sizeof(sockaddr_in) - 1), "CONNECT_AF_INET_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, SocketAddress(AF_INET6, sizeof(sockaddr_in6) - 1), "CONNECT_AF_INET6_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, SocketAddress(kLinuxAFNetlink, 2), "CONNECT_AF_NETLINK_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, SocketAddress(kLinuxAFPacket, 2), "CONNECT_AF_PACKET_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, SocketAddress(0x7f, sizeof(sa_family_t)), "CONNECT_UNKNOWN_FAMILY");
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

  gvisor::syscall::Execve bootstrap;
  bootstrap.mutable_context_data()->set_container_id(kSecondID);
  bootstrap.mutable_context_data()->set_thread_group_id(1);
  bootstrap.mutable_context_data()->set_thread_group_start_time_ns(1);
  bootstrap.mutable_context_data()->set_parent_thread_group_id(0);
  bootstrap.set_sysno(kSyscallExecve);
  bootstrap.set_pathname("/usr/bin/sleep");

  gvisor::sentry::ExecveInfo bootstrapResolved;
  bootstrapResolved.mutable_context_data()->set_container_id(kSecondID);
  bootstrapResolved.mutable_context_data()->set_thread_group_id(1);
  bootstrapResolved.mutable_context_data()->set_thread_group_start_time_ns(1);
  bootstrapResolved.mutable_context_data()->set_parent_thread_group_id(0);
  bootstrapResolved.set_binary_path("/bin/sleep");
  if (!SendSuccessfulExec(client, bootstrap, bootstrapResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve python;
  python.mutable_context_data()->set_container_id(kSecondID);
  python.mutable_context_data()->set_thread_group_id(20);
  python.mutable_context_data()->set_thread_group_start_time_ns(200);
  python.mutable_context_data()->set_parent_thread_group_id(0);
  python.mutable_context_data()->set_is_exec_session(true);
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
  child.set_sysno(kSyscallExecve);
  child.set_pathname("/usr/bin/curl");
  gvisor::sentry::ExecveInfo childResolved;
  *childResolved.mutable_context_data() = child.context_data();
  childResolved.set_binary_path("/usr/bin/curl");
  if (!SendProcessClone(output, client, kSecondID, 20, 200, 21, 210)) return false;
  if (!SendSuccessfulExec(client, child, childResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  child.mutable_context_data()->set_thread_group_id(22);
  child.mutable_context_data()->set_thread_group_start_time_ns(220);
  child.set_pathname("/usr/local/bin/python");
  childResolved.mutable_context_data()->set_thread_group_id(22);
  childResolved.mutable_context_data()->set_thread_group_start_time_ns(220);
  childResolved.set_binary_path("/usr/local/bin/python");
  if (!SendProcessClone(output, client, kSecondID, 20, 200, 22, 220)) return false;
  if (!SendSuccessfulExec(client, child, childResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve shell;
  shell.mutable_context_data()->set_container_id(kSecondID);
  shell.mutable_context_data()->set_thread_group_id(40);
  shell.mutable_context_data()->set_thread_group_start_time_ns(400);
  shell.mutable_context_data()->set_parent_thread_group_id(0);
  shell.mutable_context_data()->set_is_exec_session(true);
  shell.set_sysno(kSyscallExecve);
  shell.set_pathname("/bin/sh");
  gvisor::sentry::ExecveInfo shellResolved;
  *shellResolved.mutable_context_data() = shell.context_data();
  shellResolved.set_binary_path("/bin/sh");
  if (!SendDirectLaunchExec(output, client, shell, shellResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  child.mutable_context_data()->set_thread_group_id(41);
  child.mutable_context_data()->set_thread_group_start_time_ns(410);
  child.mutable_context_data()->set_parent_thread_group_id(40);
  child.set_pathname("/usr/bin/cat");
  childResolved.mutable_context_data()->set_thread_group_id(41);
  childResolved.mutable_context_data()->set_thread_group_start_time_ns(410);
  childResolved.mutable_context_data()->set_parent_thread_group_id(40);
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
  launcher.add_argv(boundaryLaunchMode);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, launcher) ||
      !ExpectRecord(output, kTenthID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, make_sentry(next)) ||
      !ExpectRecord(output, kTenthID, "process-exec-expected")) return false;

  gvisor::syscall::Execve failed = make_exec(kTenthID, 101, 1001);
  gvisor::syscall::Execve failed_exit = failed;
  failed_exit.mutable_exit()->set_result(-1);
  failed_exit.mutable_exit()->set_errorno(2);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, failed) ||
      !SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, failed_exit) ||
      !ExpectNoRecord(output)) return false;

  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, make_sentry(next)) ||
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
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kFourthID, "container-start")) return false;
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
  std::thread registration([&control] {
    std::this_thread::sleep_for(std::chrono::milliseconds(150));
    RegisterProfile(control, kFourthID, kProfileNPM);
  });
  registration.join();
  gvisor::sentry::ExecveInfo resolved;
  *resolved.mutable_context_data() = execve.context_data();
  resolved.set_binary_path("/usr/local/bin/node");
  resolved.set_execfn(kBoundaryHelperPath);
  resolved.add_argv(kBoundaryHelperPath);
  resolved.add_argv(boundaryLaunchMode);
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
  launch.set_sysno(kSyscallExecve);
  launch.set_pathname("python");
  gvisor::sentry::ExecveInfo python;
  *python.mutable_context_data() = launch.context_data();
  python.set_binary_path("/usr/local/bin/python3.14");
  if (!SendDirectLaunchExec(output, client, launch, python) ||
      !ExpectRecord(output, kThirteenthID, "process-exec-expected")) return false;

  gvisor::sentry::ExecveInfo handoff = python;
  handoff.set_binary_path(kBoundaryHelperPath);
  handoff.set_execfn(kBoundaryHelperPath);
  handoff.clear_argv();
  handoff.add_argv(kBoundaryHelperPath);
  handoff.add_argv(kPythonHandoffMode);
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, handoff) ||
      !ExpectRecord(output, kThirteenthID, "process-exec-expected") ||
      !SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, python) ||
      !ExpectRecord(output, kThirteenthID, "process-exec-expected")) return false;

  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, python) ||
      !ExpectUnexpectedProcessRecord(output, kThirteenthID, "SENTRY_EXEC", "PYTHON", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;

  if (!SendProcessClone(output, client, kThirteenthID, 130, 1300, 131, 1310)) return false;
  gvisor::sentry::ExecveInfo child = python;
  child.mutable_context_data()->set_thread_group_id(131);
  child.mutable_context_data()->set_thread_group_start_time_ns(1310);
  child.mutable_context_data()->set_parent_thread_group_id(130);
  child.set_binary_path("/bin/sh");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, child) ||
      !ExpectUnexpectedProcessRecord(output, kThirteenthID, "SENTRY_EXEC", "SHELL", "ARTIFACT_ROLE", "ARTIFACT_GROUP")) return false;

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

bool RunFaultCase(int output, const std::string& remote, const std::string& control, bool mismatch) {
  const char* container_id = mismatch ? kSecondID : kThirdID;
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
      strcmp(unexpected.reason, "UNKNOWN_CLASS") == 0 && strcmp(unexpected.parent_relation, "UNTRACKED_PARENT") == 0;
}

}  // namespace

int main(int argc, char** argv) {
  if (argc == 2 && strcmp(argv[1], "--semantic-only") == 0) {
    return HasPinnedPodInitProfile() && HasBoundedProfileRecordLimits() && VerifyBoundedClassificationSemantics() ? 0 : 1;
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
  if (!running) fprintf(stderr, "observer latch failure: readiness\n");
  if (!profile) fprintf(stderr, "observer latch failure: pod-init profile\n");
  if (!profile_limits) fprintf(stderr, "observer latch failure: profile record limits\n");
  const bool accessors = running && profile && profile_limits && VerifyPinnedAccessors(output, remote, control);
  if (!accessors) fprintf(stderr, "observer latch failure: normalized accessors\n");
  const bool network = accessors && VerifyNetworkFamilies(output, remote, control);
  if (!network) fprintf(stderr, "observer latch failure: socket family classification\n");
  const bool malformed_socket = network && VerifyMalformedNetworkFamilies(output, remote, control);
  if (!malformed_socket) fprintf(stderr, "observer latch failure: unknown socket family\n");
  const bool malformed_connect = malformed_socket && VerifyMalformedConnect(output, remote, control);
  if (!malformed_connect) fprintf(stderr, "observer latch failure: malformed socket address\n");
  const bool unknown_fd = malformed_connect && VerifyUnknownFDState(output, remote, control);
  if (!unknown_fd) fprintf(stderr, "observer latch failure: unknown FD state\n");
  const bool process = unknown_fd && VerifyProcessTrustBoundary(output, remote, control);
  if (!process) fprintf(stderr, "observer latch failure: process trust boundary\n");
  const bool correlation = process && VerifySentryAuthoritativeBoundary(output, remote, control);
  if (!correlation) fprintf(stderr, "observer latch failure: exec correlation boundary\n");
  const bool cloexec = correlation && VerifyCloexecReexec(output, remote, control);
  if (!cloexec) fprintf(stderr, "observer latch failure: CLOEXEC/re-exec boundary\n");
  const bool delayed = cloexec && VerifyDelayedProfileRegistration(output, remote, control);
  if (!delayed) fprintf(stderr, "observer latch failure: delayed profile registration\n");
  const bool roles = delayed && VerifyRoleHandoffAndCloneProvenance(output, remote, control);
  if (!roles) fprintf(stderr, "observer latch failure: role handoff/provenance\n");
  const bool mismatch = roles && RunFaultCase(output, remote, control, true);
  if (!mismatch) fprintf(stderr, "observer latch failure: container mismatch\n");
  const bool dropped = mismatch && RunFaultCase(output, remote, control, false);
  if (!dropped) fprintf(stderr, "observer latch failure: dropped events\n");
  const bool passed = dropped;
  if (child > 0) { kill(child, SIGTERM); waitpid(child, nullptr, 0); }
  close(output); unlink(remote.c_str()); unlink(output_path.c_str()); unlink(control.c_str()); rmdir(directory);
  return passed ? 0 : 1;
}
