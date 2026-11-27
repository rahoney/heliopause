# M13-004 — fixed service/config/drainage source map

기준 `3c93e39`. [승인된 lifecycle](./protected-activation-plan-20261020.md)과
[journal 검증](./activation-journal-qualification-20261020.md)을 재사용한다.
Service/config/drainage는 IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다.
Journal/locks/copy/proof/peer/stage의 닫힌 primitive는 다시 구현하지 않는다.

- `hosttool/installation_linux.go`의 root-only constructor는 기존 `NewSystem`의
  pinned Docker/runsc/config/isolated environment owner를 재사용하고 fixed
  `/usr/bin/systemctl` identity만 추가한다. Network helper에 설치 API를 넣지 않는다.
- `releaseactivation/service_linux.go`는 이 executor의 기존 RunOutput/RunDiscard만
  사용한다. Machine query stdout은 별도 fail-on-limit writer로 bounded하게 받고,
  각 command에20초 context를 둔다. Query 실패/잘림/중복·누락 property는 UNKNOWN이다.
  임의 tool/argv/service를 외부 request에서 받지 않는다.
- Native registration이 명시적으로 소유한 fixed `haa-network-policy-helper.service`의
  unit bytes와 FragmentPath/DropInPaths/NeedDaemonReload/User/Group/LoadState/actual
  MainPID/ControlPID/ControlGroup를 확인한다. User/Group=root, drop-in 없음, canonical
  native version ExecStart와 actual kernel root executable inode/pidfd를 연결한다.
  Installed native bytes는 기존 SDK reader가 따로 확인한다. OS package의 다른 이름
  unit을 추정/stop하거나 현재 unmanaged host 설정을 인수하지 않는다.
- `releaseactivation/config_linux.go`는 journal의 fixed3개 snapshots를 strict canonical
  JSON/known unit template로 해석한다. Registered old CLI path/UID/GID/observer path가
  맞아야 한다. 바꿀 것은 new CLI/observer/정확한 unit ExecStart뿐이며 DockerPath와
  DockerEndpoint 및 file mode/GID는 보존한다. Rollback은 실제 현재 bytes가 saved old
  또는 exact derived new bytes일 때만 original bytes로 복원하며 unknown 변경은 남긴다.
- 실제 owner 대조에서 observation resource socket은
  `/run/heliopause-network-policy/observation.sock`이고 network socket과 parent가 같아야
  함을 확인했다. Native managed config/unit은 이 고정 directory와 network-policy.sock을
  사용한다. Package 예시의 다른 directory나 현재 host 설정이 호환된다고 추정하지 않는다.
- 중지 후 state/PID/cgroup와 retained pidfd exit, fixed 두 socket의 부재를 확인한다.
  새 시작 뒤 실제 service PID/inode/UID와 두 socket mode/owner/GID/SO_PEERCRED를 확인한다.
  `systemctl stop` return0만으로 종료를 판정하지 않는다. 공식 의미는
  [systemctl](https://www.freedesktop.org/software/systemd/man/256/systemctl.html) 및
  [systemd1 properties](https://www.freedesktop.org/software/systemd/man/262/org.freedesktop.systemd1.html)를 따른다.
- `releaseactivation/drain_linux.go`: 기존 동작의 HAA Docker prefix, active observation
  slices, fixed observer endpoints/lock와 protected-version executable consumers를 확인한다.
  Root query/신원 오류는 부재가 아니며 controller/installer/검증된 service 외 실행본이
  있으면 BUSY다. No broad kill/rm·강제 자원 삭제·이름만으로 restore 권한을 만들지 않는다.

기본 회귀는 bounded query/누락·중복 property·inactive/unknown PID, unit/config/UID drift,
부분 config swap 뒤 exact restore/unknown retention, stop/query/start fault를 실제
adapter 경계에서 확인한다. Kernel process/socket은 disposable namespace에서 검사하고
fake runner/crypto mechanics를 actual systemd/native4 signed activation과 구분한다.
Canonical Quick/security/docs/vulnerability와 관련 Race 뒤 완료 payload/binary를 삭제한다.

두 번째 검토에서 ordinary NewSystem은 systemctl을 등록하지 않는다는 것을 확인해
root-only 전용 constructor로 필요한 한 identity만 추가하기로 했다. 전체 CLI sudo,
root PATH lookup/HTTP, caller-chosen unit, query failure의 성공 처리와 unknown rollback은
제외했다. 실제 lifecycle/RPC/health entry/CLI/first-bootstrap 연결은 이후까지 남는다.
