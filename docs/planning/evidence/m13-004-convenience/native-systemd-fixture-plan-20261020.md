# M13-004 — 실제 systemd/검역 전환을 위한 일회성 시험 환경

제품 source는 a22aa12, 공식 rc1/rc2 tag build다. [서명 시험 계획](./signed-lifecycle-plan-20261020.md)의
MISSING인 실제 service/register/ordinary health/전환·중단 재개만 확인한다. Production
Root/SDK/Policy/command를 대체하거나 시험을 위한 product bypass를 추가하지 않는다.

Baseline: 기존 whole native workflow IMPLEMENTED=YES/WIRED=YES, 실제 booted service
QUALIFIED=NO/ACCEPTANCE_CLOSED=NO. rc1 actual signed13 독립 검증과 production
Root SDK/파일 준비8 cases는 PASS이며 registration/health는 PENDING/UNVERIFIED다.
Host Docker29.8.1/systemd cgroup2는 기존 lock과 같고 Ubuntu fixture는 아직 없다.

## 고정 입력과 순서

- 시험 OS만 공식 Ubuntu24.04의 `ubuntu@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac`로 고정한다. OCI index body digest와 amd64
  descriptor를 public registry에서 대조했다. 이 OS fixture는 제품 runtime lock이나
  새 배포 지원 platform이 아니다. Docker packages는 기존 CI와 같은29.8.1/2.3.5
  exact versions 및 공식 서명 APT source를 재사용한다. Systemd 등 OS packages의 실제
  version 목록과 만들어진 local image ID를 기록한다.
- 먼저 일반 image build 환경에서 OS packages만 설치한다. Booted test는 별도 label의
  container와 빈 전용 Docker volume에서 수행한다. Host directory/socket·사용자 프로젝트는
  전달하지 않는다. Systemd/inner Docker/gVisor에 필요한 privilege는 이 fixture에만 주고
  cgroup namespace는 private다. Host cgroup root나 다른 scope를 수정하지 않는다.
- [systemd container interface](https://systemd.io/CONTAINER_INTERFACE/)에 따라 API
  filesystem을 준비하고 sys/proc sys를 read-only, 해당 network sys만 writable로 두며
  private cgroup hierarchy와 delegate된 container scope를 실제 확인한다. [Docker 실행 계약](https://docs.docker.com/engine/containers/run/)의
  label/명시적 image ID/전용 volume/종료 신호를 사용한다. Unsupported mount/delegation은
  실패로 기록하고 host hierarchy bind나 보호 완화로 우회하지 않는다.
- Native first files를 먼저 준비한 뒤 canonical runtime bundle을 넣는다. `bootstrap-files`는
  `/usr/libexec/heliopause` 존재를 거부하는 반면 canonical gVisor bundle도 그 아래에 있으므로
  순서를 바꾸지 않는다. 이미 준비한 파일을 runtime 설치 뒤 반복 bootstrap하지 않는다.
  동일 lock의 기존 root-protected runtime6개와 companion/pod-init은 hash를 대조해 copy한다.
  기존 compiler/runtime을 새로 build하거나 임의 latest를 설치하지 않는다.
- Docker29의 실제 containerd data root도 전용 볼륨에 둔다. Guest overlay 위의
  `/var/lib/containerd`에서 실제 create가 nested overlay EINVAL로 거부됐다. Root/SDK/
  Policy를 바꾸지 않고 서비스를 중지한 뒤 owned data의 전체 파일 metadata/hash를
  대조해 같은 전용 볼륨으로 옮기고 guest 내부 bind mount를 등록한다. Host 경로나
  Docker socket을 bind하지 않는다. 초기 실패와 교정 뒤 실제 실행을 따로 기록한다.
- 실제 native config/unit/service/register, configured UID1000의 doctor와 nominal npm
  검역을 확인한다. Fixture의 sudo grant는 두 실제 signed installer의 `serve`에만 한정한다.
  이는 fixture OS 설정이며 제품에 새로운 sudoers 필수 관문을 추가하는 것이 아니다.
- Runtime images는 기존 lock의 exact4개다. 가능하면 Host의 검증된 image ID를 stream해
  복제하고 exact digest를 다시 확인한다. 임시 거대 archive를 중복 보존하지 않는다.
- Public Release가 없는 rc artifacts의 전체 caller 시험에는 fixture HTTPS transport만
  사용할 수 있다. Domain/명령/production client는 유지하고 guest에만 fixture CA/응답을
  준비한다. 실제 official signature/root/source/tag/모든 required 검사/Policy와 kernel peer/
  Root protocol은 그대로다. Fixture의 Release metadata를 actual public Release API
  성공이라고 보고하지 않는다. HTTP fixture 없이 할 수 있는 범위부터 증거로 닫는다.
- 정상 전환·OS privilege 거부·동시성·내용 변조·실제 service 실패·Root process 중단과
  journal recovery를 구분한다. Container PID1 재시작은 실제 guest service cold start지만
  Host kernel/WSL 재부팅의 증거가 아니다. 그 gap은 성공으로 치환하지 않는다.

## 정리와 두 번째 검토

새 image/container/volume의 label·정확한 ID/기존 부재를 기록하고 마지막 사용 뒤 해당
소유 자원만 삭제한다. Host Docker data와 설치/service/config·shared runtime은 보존한다.
타인이 사용하는 image/tag나 unknown reference는 지우지 않는다. CA private key나 login
credential은 commit/원본 report에 넣지 않는다. 전체 payload와 캐시는 최종 소비 후 지운다.

검토 결과: booted service가 없어 mock으로 대체하는 기존 한계를 실제로 줄이기 위한
시험 환경이다. 새 product config/Port/daemon/job·공개 Release·main merge·Host 영구
migration은 없다. Public API·물리 kernel reboot의 미검증 범위는 계속 명시한다.
