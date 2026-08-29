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

bool SendAll(int fd, const std::string& value) {
  return send(fd, value.data(), value.size(), 0) == static_cast<ssize_t>(value.size());
}

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

bool ExpectRecord(int output, const char* container_id, const char* kind, const char* reason = nullptr) {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  if (size <= 0) return false;
  std::string expected = std::string("{\"container_id\":\"") + container_id + "\",\"kind\":\"" + kind + "\"";
  if (reason != nullptr) expected += ",\"reason\":\"" + std::string(reason) + "\"";
  expected += "}";
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
  return profile.find("syscall/execve/enter") != std::string::npos &&
      profile.find("syscall/openat/enter") != std::string::npos &&
      profile.find("syscall/connect/enter") != std::string::npos &&
      profile.find("syscall/socket/enter") != std::string::npos &&
      profile.find("thread_group_start_time") != std::string::npos &&
      profile.find("parent_thread_group_id") != std::string::npos &&
      profile.find("is_exec_session") != std::string::npos &&
      profile.find("\"group_id\"") != std::string::npos &&
      profile.find("\"process_name\"") != std::string::npos &&
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
  execve.add_argv("raw-argv-must-not-leave-helper");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, execve) || !ExpectRecord(output, kFirstID, "process-exec-expected")) return false;
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
  socket.mutable_context_data()->set_process_name("trusted-test-process");
  socket.set_domain(2); socket.set_type(1); socket.set_protocol(0);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket) || !ExpectRecord(output, kFirstID, "network-attempt")) return false;
  close(client);
  return ExpectRecord(output, kFirstID, "stream-end");
}

std::string SocketAddress(sa_family_t family, size_t length) {
  std::string address(length, '\0');
  memcpy(&address[0], &family, sizeof(family));
  return address;
}

bool VerifyNetworkFamilies(int output, const std::string& remote, const std::string& control) {
  if (!RegisterProfile(control, kFifthID, kProfilePyPI)) return false;
  const int client = ConnectRemote(remote);
  if (client < 0 || !Handshake(client)) return false;
  gvisor::container::Start start;
  start.mutable_context_data()->set_container_id(kFifthID);
  if (!SendEvent(client, gvisor::common::MESSAGE_CONTAINER_START, start) || !ExpectRecord(output, kFifthID, "container-start")) return false;

  gvisor::syscall::Socket socket;
  socket.mutable_context_data()->set_container_id(kFifthID);
  socket.set_domain(AF_UNIX); socket.set_type(1); socket.set_protocol(0);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket) || !ExpectNoRecord(output)) return false;
  socket.set_domain(AF_INET);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket) || !ExpectRecord(output, kFifthID, "network-attempt")) return false;
  socket.set_domain(AF_INET6);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket) || !ExpectRecord(output, kFifthID, "network-attempt")) return false;

  gvisor::syscall::Connect connect;
  connect.mutable_context_data()->set_container_id(kFifthID);
  connect.set_address(SocketAddress(AF_UNIX, sizeof(sockaddr_un)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectNoRecord(output)) return false;
  connect.set_address(SocketAddress(AF_INET, sizeof(sockaddr_in)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectRecord(output, kFifthID, "network-attempt")) return false;
  connect.set_address(SocketAddress(AF_INET6, sizeof(sockaddr_in6)));
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectRecord(output, kFifthID, "network-attempt")) return false;
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
  socket.set_domain(domain);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_SOCKET, socket) || !ExpectRecord(output, kSixthID, "stream-fault", reason)) return false;
  close(client);
  return true;
}

bool VerifyMalformedNetworkFamilies(int output, const std::string& remote, const std::string& control) {
  return VerifySocketFault(output, remote, control, AF_UNSPEC, "SOCKET_AF_UNSPEC") &&
      VerifySocketFault(output, remote, control, AF_NETLINK, "SOCKET_AF_NETLINK") &&
      VerifySocketFault(output, remote, control, AF_PACKET, "SOCKET_AF_PACKET") &&
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
  connect.set_address(address);
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_CONNECT, connect) || !ExpectRecord(output, kSeventhID, "stream-fault", reason)) return false;
  close(client);
  return true;
}

bool VerifyMalformedConnect(int output, const std::string& remote, const std::string& control) {
  return VerifyConnectFault(output, remote, control, "\x02", "CONNECT_ADDRESS_TOO_SHORT") &&
      VerifyConnectFault(output, remote, control, SocketAddress(AF_UNSPEC, sizeof(sa_family_t)), "CONNECT_AF_UNSPEC") &&
      VerifyConnectFault(output, remote, control, SocketAddress(AF_UNIX, offsetof(sockaddr_un, sun_path)), "CONNECT_AF_UNIX_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, SocketAddress(AF_INET, sizeof(sockaddr_in) - 1), "CONNECT_AF_INET_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, SocketAddress(AF_INET6, sizeof(sockaddr_in6) - 1), "CONNECT_AF_INET6_INVALID_LENGTH") &&
      VerifyConnectFault(output, remote, control, SocketAddress(0x7fff, sizeof(sa_family_t)), "CONNECT_UNKNOWN_FAMILY");
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
  bootstrap.mutable_context_data()->set_thread_group_id(10);
  bootstrap.mutable_context_data()->set_thread_group_start_time_ns(100);
  bootstrap.mutable_context_data()->set_parent_thread_group_id(0);
  bootstrap.set_pathname("/usr/bin/sleep");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, bootstrap) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::sentry::ExecveInfo bootstrapResolved;
  bootstrapResolved.mutable_context_data()->set_container_id(kSecondID);
  bootstrapResolved.mutable_context_data()->set_thread_group_id(10);
  bootstrapResolved.mutable_context_data()->set_thread_group_start_time_ns(100);
  bootstrapResolved.mutable_context_data()->set_parent_thread_group_id(0);
  bootstrapResolved.set_binary_path("/bin/sleep");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, bootstrapResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve python;
  python.mutable_context_data()->set_container_id(kSecondID);
  python.mutable_context_data()->set_thread_group_id(20);
  python.mutable_context_data()->set_thread_group_start_time_ns(200);
  python.mutable_context_data()->set_parent_thread_group_id(0);
  python.mutable_context_data()->set_is_exec_session(true);
  python.set_pathname("python");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, python) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::sentry::ExecveInfo pythonResolved;
  pythonResolved.mutable_context_data()->set_container_id(kSecondID);
  pythonResolved.mutable_context_data()->set_thread_group_id(20);
  pythonResolved.mutable_context_data()->set_thread_group_start_time_ns(200);
  pythonResolved.mutable_context_data()->set_parent_thread_group_id(0);
  pythonResolved.mutable_context_data()->set_is_exec_session(true);
  pythonResolved.set_binary_path("/usr/local/bin/python3.14");
  if (!SendEvent(client, gvisor::common::MESSAGE_SENTRY_EXEC, pythonResolved) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve pip;
  pip.mutable_context_data()->set_container_id(kSecondID);
  pip.mutable_context_data()->set_thread_group_id(30);
  pip.mutable_context_data()->set_thread_group_start_time_ns(300);
  pip.mutable_context_data()->set_parent_thread_group_id(0);
  pip.mutable_context_data()->set_is_exec_session(true);
  pip.set_pathname("pip");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, pip) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  gvisor::syscall::Execve child;
  child.mutable_context_data()->set_container_id(kSecondID);
  child.mutable_context_data()->set_thread_group_id(21);
  child.mutable_context_data()->set_thread_group_start_time_ns(210);
  child.mutable_context_data()->set_parent_thread_group_id(20);
  child.mutable_context_data()->set_is_exec_session(true);
  child.set_pathname("/usr/bin/curl");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, child) || !ExpectRecord(output, kSecondID, "process-exec-unexpected")) return false;

  child.mutable_context_data()->set_thread_group_id(22);
  child.mutable_context_data()->set_thread_group_start_time_ns(220);
  child.set_pathname("/usr/local/bin/python");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, child) || !ExpectRecord(output, kSecondID, "process-exec-unexpected")) return false;

  gvisor::syscall::Execve shell;
  shell.mutable_context_data()->set_container_id(kSecondID);
  shell.mutable_context_data()->set_thread_group_id(40);
  shell.mutable_context_data()->set_thread_group_start_time_ns(400);
  shell.mutable_context_data()->set_parent_thread_group_id(0);
  shell.mutable_context_data()->set_is_exec_session(true);
  shell.set_pathname("/bin/sh");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, shell) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;

  child.mutable_context_data()->set_thread_group_id(41);
  child.mutable_context_data()->set_thread_group_start_time_ns(410);
  child.mutable_context_data()->set_parent_thread_group_id(40);
  child.set_pathname("/usr/bin/cat");
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, child) || !ExpectRecord(output, kSecondID, "process-exec-expected")) return false;
  close(client);
  return ExpectRecord(output, kSecondID, "stream-end");
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
  if (!SendEvent(client, gvisor::common::MESSAGE_SYSCALL_EXECVE, execve)) return false;
  std::thread registration([&control] {
    std::this_thread::sleep_for(std::chrono::milliseconds(150));
    RegisterProfile(control, kFourthID, kProfileNPM);
  });
  const bool classified = ExpectRecord(output, kFourthID, "process-exec-expected");
  registration.join();
  close(client);
  return classified && ExpectRecord(output, kFourthID, "stream-end");
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

}  // namespace

int main() {
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
  const bool process = malformed_connect && VerifyProcessTrustBoundary(output, remote, control);
  if (!process) fprintf(stderr, "observer latch failure: process trust boundary\n");
  const bool delayed = process && VerifyDelayedProfileRegistration(output, remote, control);
  if (!delayed) fprintf(stderr, "observer latch failure: delayed profile registration\n");
  const bool mismatch = delayed && RunFaultCase(output, remote, control, true);
  if (!mismatch) fprintf(stderr, "observer latch failure: container mismatch\n");
  const bool dropped = mismatch && RunFaultCase(output, remote, control, false);
  if (!dropped) fprintf(stderr, "observer latch failure: dropped events\n");
  const bool passed = dropped;
  if (child > 0) { kill(child, SIGTERM); waitpid(child, nullptr, 0); }
  close(output); unlink(remote.c_str()); unlink(output_path.c_str()); unlink(control.c_str()); rmdir(directory);
  return passed ? 0 : 1;
}
