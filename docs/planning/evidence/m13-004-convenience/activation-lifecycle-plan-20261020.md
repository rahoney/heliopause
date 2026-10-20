# M13-004 — root 전환·복구와 실제 caller 연결 map

[상위 승인 설계](./protected-activation-plan-20261020.md)를 구체화한다.
`3c93e39` 이후 service/config/drain adapter의 source map은
[별도 leaf](./activation-service-plan-20261020.md)가 소유한다. 다음 source 전 baseline은
아래와 같다. M13-004만 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 다음 수정 |
| --- | --- | --- | --- | --- | --- |
| 보호된 copy/read/peer/stage/journal/locks | YES | YES, primitive | YES, local | YES, primitive | 재사용 |
| fixed service/config/drain adapter | YES, 현재 candidate | YES, concrete functions | 진행 중 | NO, 실제 systemd 전환 | 기존 source 재사용, crypto mock 구분 |
| proof 보존·previous signed version 읽기 | PARTIAL, active reader만 YES | NO, recovery | NO | NO | installed reader factoring/proof store |
| root transaction·verified rollback·recovery | NO | NO | NO | NO | operation/recovery source |
| one-shot authenticated IPC/helper executable | NO | NO | NO | NO | server/client/cmd source |
| per-request quarantine/session/CLI/borrowed health | NO | NO | NO | NO | 기존 pipeline과 cleanup 연결 |
| 새 signed native4 end-to-end qualification | NO | NO | NO | NO | 실제 protected release/host 필요 |

## Root transaction와 crash recovery

1. `installed_linux.go`의 기존 manifest/member/mode/hash/metadata/proof 검증 본문을
   private recorded-version reader로 재사용한다. Public ReadInstalledRelease는 기존처럼
   root registration/current symlink의 일치와 전후 pointer identity를 검사한다. Recovery가
   journal의 old version을 검증하기 위해 현재 pointer를 먼저 덮어쓰지 않는다. 기존
   구현을 새 engine으로 대체하지 않고 같은 검증 본문을 공유한다.
2. `proof_linux.go`는 fixed proofs/<manifestSHA>/<bundleSHA>.json에 SDK로 검증한
   proof를 root-owned0444/parent0755로 저장한다. 기존 파일은 exact bytes/owner/link를
   확인하며 foreign 것을 고치지 않는다. PREPARED 이전 새 proof를 durable하게 남긴다.
   Root BundleVerifier는 같은 fixed SDK/strict manifest를 재사용하고 user receipt나
   Policy 문자열을 인수하지 않는다. Existing releaseinstall의 mandatory verifier는 유지한다.
3. Root transaction owner를 먼저 잡고 independently checked active release와 old
   service를 확인한다. Authenticated old CLI→verified root stage→strict config snapshot
   →proof persistence→durable PREPARED 순서다. Root server는 runtime SH를 추가로 잡은
   채 EX를 기다리지 않는다. Sender에게 PREPARED 뒤 기존 command-owned cleanup/SH 해제,
   ACTIVATE를 요구하고 그 전에는 pointer/config/service를 바꾸지 않는다.
4. Bounded EX 뒤 old installed view/current/config/peer와 drainage를 재확인한다.
   SWITCHING sync→confirmed old stop→exact derived configs→mandatory-verifier-gated
   existing Installer.Install→new registration→VERIFYING→new service start/identity→
   independently checked installed/config facts→ordinary-user narrow compatibility health
   →actual resource drainage→COMMITTED 순서다. Root가 caller UID/path/argv를 받지 않는다.
5. 실패는 original cause를 유지하고 새 service의 actual signed executable/process/stop을
   확인한 후에만 exact old config와 signed existing version을 복원한다. Partial configs는
   old/new derived bytes만 허용한다. 불명확하면 RECOVERY_REQUIRED를 durable하게 남기고
   marker와 previous payload/proofs를 보존한다. Blind restore/force kill은 없다.
6. Recovery는 root lock+EX와 같은 signed old/new views/journal/config/unit/process facts를
   재구성한다. PREPARED도 자동 성공이 아니다. COMMITTED/ROLLED_BACK은 independently
   healthy facts와 cleanup 뒤 fixed small history로 atomic하게 옮겨 runtime marker를
   해제한다. Response 손실은 이미 건강하게 commit한 설치를 몰래 rollback할 이유가 아니다.

## Fixed helper IPC와 lifecycle handoff

- `server_linux.go`와 `cmd/haa-release-installer`는 fixed one-shot serve/recover operations만
  소유한다. 무인자/help/version은 inert하다. Serve는 actual own executable이 installed
  signed installer inode인지 확인하고 root-owned fixed /run socket에서 old kernel peer를
  인증한다. New target helper를 먼저 root로 실행하지 않는다. Socket/partial stage cleanup은
  owned inode만 제거하고 primary error를 보존한다. Context는 blocking IO deadline도 닫는다.
- Request는 기존 fixed wire bytes와 정확한 PREPARED/ACTIVATE/COMMITTED frames뿐이다.
  UID/GID/path/service/command/ALLOW field, root HTTP/PATH gh는 없다. Old CLI의 sealed set
  adapter만 producer로 연결하고 old installed installer를 fixed absolute sudo invocation으로
  실행한다. Whole CLI sudo나 untrusted candidate command는 없다.
- Before ACTIVATE: 기존 ObserverSupervisor.Close→Executor.Close를 실제로 완료하고
  같은 RuntimeLease FD를 해제한다. Failure는 root에게 abort하고 설치를 전환하지 않는다.
  UpdateOwner는 전체 session 끝까지 유지하고 installer lock과 동일 FD를 혼동하지 않는다.

## Borrowed exclusive health와 pipeline 연결

- New CLI의 internal fixed health path만 root-created inherited Unix channel을 검증해
  EX 아래 narrow borrowed lease를 사용할 수 있다. Env/flag/token만으로 lease를 우회하지
  않는다. Socketpair SO_PEERCRED는 생성 시 root issuer의 credential임을 확인하고,
  Root issuer는 채널 생성 전에 자신의 actual old protected installer inode를 검증한다.
  Ordinary child는 root-private journal 또는 root proc exe 읽기 권한을 가정하지 않는다.
  Root가 전달하는 bounded old record와 read-only installer FD를 public signed old view에
  연결하고 Root creator UID/GID/PID와 실제 parent/pidfd를 확인한다. Kernel의
  [proc exe 접근은 ptrace 권한 검사 대상](https://man7.org/linux/man-pages/man5/proc_pid_exe.5.html)이다.
  Root는 child의 현재 ordinary UID/GID/executable/PID를 별도 kernel facts로 확인한다.
- Health issuer의 kernel UID/GID는 root0이고 creator PID는 child's actual parent여야 한다.
  FD3는 root-created socketpair, FD4는 old installer의 read-only descriptor다. Ordinary child가
  root-private journal을 읽거나 root proc exe의 ptrace 권한을 얻도록 보호를 완화하지 않는다.
  Channel의 bounded old record를 public stored-proof/member reader로 확인하고 FD4의
  inode/owner/mode를 그 signed old installer와 비교한다. Root server가 own executable을
  독립 확인한 뒤 이 channel을 만들며 child의 current UID와 creator UID를 혼동하지 않는다.
- Candidate health는 Root가 stage basename에서 유도한 fixed state-directory workspace를
  생성해 configured ordinary UID/GID로 소유시키고 HOME/XDG roots를 거기로 제한한다.
  사용자 기존 프로젝트/설정/cache를 쓰지 않는다. Root는 child Start의 known PID를 Wait까지
  소유하며 실제 child UID/GID/exe와 HEALTHY→RELEASE handshake 후 Wait0를 확인한다.
  Root가 health child 전체를 root로 실행하거나 임의 command를 health frame에 넣지 않는다.
  Workspace는 실제 resource drainage가 확인된 뒤 owned inode만 삭제하며 실패/중단의
  unknown 상태에서는 journal의 stage 이름으로 재구성 가능한 경로를 recovery까지 보존한다.
- Child의 요청은 doctor와 고정된 기존 nominal npm quarantine compatibility뿐이며 임의
  user command를 전달하지 않는다. Root는 borrowed health child의 종료/cleanup 뒤 실제
  installed bytes/service/socket/resource facts를 다시 검증한다. Child return0/출력은
  독립적인 safety/commit authority가 아니다.
- `ports.InstallUpdateContentSession`은 기존 session에 실제 사용하는 Artifact/Verification/
  Inspection port를 embed한다. InstallUpdateService의 per-request copies에만 세 binding을
  적용하며 shared base/정책/evidence/graph/manifest/staging 순서를 바꾸지 않는다. 기존
  npm/PyPI/GitHub/Go/Cargo/Terraform session은 그대로 동작한다.
- Bootstrap은 managed runtime lease를 tools 생성 전에 잡고 실제 owned cleanup 뒤 놓는다.
  Doctor/launcher의 실제 diagnosis도 같은 gate를 사용하고 help/settings는 inert하다.
  Self-update는 official frozen source+authenticity verifier+release-set inspector+기존 M6
  entry/M4 aggregate/Evidence/Manifest/LocalStaging과 root promoter를 한 session으로 연결한다.
  실제 서비스가 준비된 뒤 `helox self-update --version vX.Y.Z`와 7언어 설명/실제 progress를
  등록한다. Project package와 이름이 같다는 이유로 본체 업데이트를 선택하지 않는다.
- 실제 executable이 구현된 뒤에만 helper asset producer/manifest/SBOM/native bundle/
  packages/workflow/publish validators의 정확한 asset set을 함께 확장한다. 기존 v0.1.0
  증거는 수정하지 않는다. First native bootstrap/managed registration은 별도 검증 책임이며
  unmanaged legacy/OS package/계정 이름 미정 상태를 자동 인수하지 않는다.

## 재검토와 검증 범위

동일 입력 canonical check와 기존 생태계 regression, concurrent per-request binding,
primary/cleanup error, root copy/service/health fault와 partial/crash recovery를 분리한다.
Namespace crypto/service mocks는 actual signed target/systemd activation이라고 쓰지 않는다.
새 signed native4 target과 실제 첫 native bootstrap→update host qualification은 protected
release 후 별도 실제 증거가 필요하다. Public tag/publish/main merge나 host 영구 migration을
미리 실행하지 않는다. Package names는 name-dependent channels에만 남는다.

두 번째 대조에서 pointer-only 성공, source 파일 재다운로드, SH→EX 교착, candidate 자기 승인,
root health child 실행, unknown service 복원과 완료 output 삭제 누락을 제외했다. 설치본의
사용 가능 상태와 source/CI qualification을 계속 구분한다.
