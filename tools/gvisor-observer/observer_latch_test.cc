// Exercises the exact pinned remote-sink protobuf framing through the helper.
// It deliberately checks only normalized records and never retains payloads.
#define main observer_main
#include "tools/haa_gvisor_observer/observer.cc"
#undef main

#include <signal.h>
#include <fstream>
#include <iterator>
#include <sys/time.h>
#include <sys/wait.h>

namespace {

constexpr char kFirstID[] = "0123456789abcdef";
constexpr char kSecondID[] = "fedcba9876543210";
constexpr char kThirdID[] = "abcdef0123456789";
constexpr char kFourthID[] = "9876543210fedcba";

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

bool ExpectRecord(int output, const char* container_id, const char* kind) {
  char buffer[1024];
  const ssize_t size = recv(output, buffer, sizeof(buffer), 0);
  if (size <= 0) return false;
  const std::string expected = std::string("{\"container_id\":\"") + container_id + "\",\"kind\":\"" + kind + "\"}";
  return std::string(buffer, size) == expected;
}

bool HasPinnedPodInitProfile() {
  std::ifstream input("tools/haa_gvisor_observer/pod-init.json");
  if (!input.is_open()) return false;
  const std::string profile((std::istreambuf_iterator<char>(input)), std::istreambuf_iterator<char>());
  return profile.find("syscall/execve/enter") != std::string::npos &&
      profile.find("syscall/openat/enter") != std::string::npos &&
      profile.find("syscall/connect/enter") != std::string::npos &&
      profile.find("syscall/socket/enter") != std::string::npos &&
      profile.find("\"group_id\"") != std::string::npos &&
      profile.find("\"process_name\"") != std::string::npos &&
      profile.find("ignore_missing") == std::string::npos;
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
  const bool faulted = ExpectRecord(output, container_id, "stream-fault");
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
  if (!running) fprintf(stderr, "observer latch failure: readiness\n");
  if (!profile) fprintf(stderr, "observer latch failure: pod-init profile\n");
  const bool accessors = running && profile && VerifyPinnedAccessors(output, remote, control);
  if (!accessors) fprintf(stderr, "observer latch failure: normalized accessors\n");
  const bool mismatch = accessors && RunFaultCase(output, remote, control, true);
  if (!mismatch) fprintf(stderr, "observer latch failure: container mismatch\n");
  const bool dropped = mismatch && RunFaultCase(output, remote, control, false);
  if (!dropped) fprintf(stderr, "observer latch failure: dropped events\n");
  const bool passed = dropped;
  if (child > 0) { kill(child, SIGTERM); waitpid(child, nullptr, 0); }
  close(output); unlink(remote.c_str()); unlink(output_path.c_str()); unlink(control.c_str()); rmdir(directory);
  return passed ? 0 : 1;
}
