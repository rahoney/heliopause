# M13-004 — 보호된 설치 전환의 권한·복구 계획

기준 source `976eb58e0de0e4c18f864dd72219e3cfee0a76d8`.

설치 전용 역할과 Linux/WSL2 amd64/same-runtime 범위는 사용자가 승인했다.
이 문서는 [본체 계획](./self-update-20261020.md)의 다음 실제 gap에 관한
privileged source 이전 baseline/edit map/두 번째 검토다. M13-004만 IN_PROGRESS다.
[전체 set 연결 검증](./release-set-qualification-20261020.md)은 설치 교체 증거가 아니다.

## 실제 owner 감사

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 재사용과 실제 gap |
| --- | --- | --- | --- | --- | --- |
| 검증된 bundle의 versioned copy/atomic pointer | YES | YES, user installer | YES, 기존 범위 | YES, 기존 범위 | releaseinstall.Installer.Install/activate/activateCurrentWithSync. user0700 permission model이며 protected lifecycle은 없음 |
| 실제 kernel peer의 root-protected executable 확인 | YES | YES, network/resource service | YES, 기존 범위 | YES, 기존 범위 | hosttool.NewPolicyPeerAuthorizer/AuthorizePeer, verifyExecutable. 기본 constructor의 expected digest는 빈 값이며 별도 signed digest pin은 없음 |
| command-owned observer/Host executor 정리 | YES | YES | YES, 기존 범위 | YES, 기존 범위 | bootstrap.Run의 deferred ObserverSupervisor.Close/Executor.Close. 설치 handoff 전에 실제 정리하는 연결이 gap |
| privileged 설치 전용 executable·root copy·전환 | NO | NO | NO | NO | network helper를 installer로 확장하지 않고 별도 역할 구현 |
| 전역 runtime lease/단일 update owner/handoff | NO | NO | NO | NO | 기존 observer endpoint lock은 user-owned exclusive file이며 모든 runtime/config 전환 lease의 대체물이 아님 |
| journal·정확한 service 전환·복원·crash recovery | NO | NO | NO | NO | pointer 하나의 rename만으로 config/helper/서비스 신원이 함께 전환되지 않음 |

기존 installer를 대체하지 않는다. Protected mode의 고정 root/permission과 네 번째
역할, 검증된 기존 version으로 돌아가는 API만 확장해 같은 copy/pointer primitives를
사용한다. 기존 user0700 install/doctor와 native3개 release의 의미는 유지한다.
Root-specific lifecycle은 별도 adapter가 소유하며 Core/Policy에 OS 권한을 넣지 않는다.

## 고정된 권한과 IPC

- 별도 `cmd/haa-release-installer`는 Linux amd64에서 root로만 실제 설치 operation을
  수행한다. 인자 없는 실행/help/version은 부작용 없이 종료하며 서비스 시작은 명시적
  고정 operation으로만 가능하다. sandbox의 기존 raw ELF 실험을 약화하지 않는다.
- Ordinary CLI가 공식 release의 전체 required 검역·aggregate ALLOW·sealed staging을
  마친 뒤 OS 관리자 인증으로 **보호된 기존 installer만** 실행한다. 전체 CLI를 root로
  실행하거나 artifact/helper 경로·명령·환경을 요청에서 지정하지 않는다.
- Root가 소유하는 고정 Unix socket에서 old CLI의 kernel UID/PID/executable inode를
  확인한다. Old protected manifest와 signed bytes/hash는 installer가 독립 확인한다.
  UID/path/Policy 문자열/user receipt/nonce/child exit0은 설치 authority가 아니다.
- IPC는 version/manifest digest, bounded proof와 고정 역할 파일 bytes만 전달한다.
  Root는 user staging path를 따라 쓰거나 임의 target/argv/service/UID를 받아들이지 않는다.
  Root-private stage로 복사하는 bytes를 다시 해시하고 fixed-root SDK와 strict readers로
  exact official release/role/runtime identity를 재검증한다. Privileged HTTP/PATH gh는 없다.
  Linux의 [SO_PEERCRED](https://man7.org/linux/man-pages/man7/unix.7.html)는
  connect/listen/socketpair 당시 credential이므로 inherited health channel의 root
  creator 확인과 ordinary child의 실제 실행 UID/PID 확인을 혼동하지 않는다.
- Native store와 root installation record는 고정 경로이고 root만 수정한다. CLI/observer/
  network helper/installer의 실제 version paths를 기록하며 public pointer만 바꾸고
  config가 old inode를 가리키는 상태를 정상으로 처리하지 않는다.
- OS package DB 소유 설치, foreign/unmanaged development 상태와 기존 link는 자동
  인수하지 않는다. Initial bootstrap의 서명 검증과 명시적 migration은 별도 책임이다.
  Unhealthy existing install을 first-install fallback으로 취급하지 않는다.

## 동시 실행과 handoff

1. Managed ordinary runtime commands는 root-owned persistent lease file의 shared lock을
   Host tools/observer 생성 전에 잡고 실제 cleanup 뒤에 놓는다. Help/settings는 runtime을
   준비하지 않는다. Launcher diagnosis/doctor도 실제 진단을 할 때 같은 보호를 적용한다.
   Record/journal query 오류는 legacy absence가 아니며 fail-closed다.
2. Update session은 resolution 이전 단일 update owner를 nonblocking으로 잡는다.
   다른 update는 network/staging 없이 BUSY로 끝나 shared lease를 해제한다. Root helper도
   별도 root-only transaction lock으로 중복 실행을 막는다.
   [flock](https://man7.org/linux/man-pages/man2/flock.2.html)은 advisory이며 open file
   description에 연결된다. Lease 자체를 privileged authority로 사용하지 않고 root
   credential/bytes 재검증을 유지한다. 별도 open으로 같은 lock을 다시 잡는 교착과
   fork/exec 뒤 inherited FD가 lock을 붙잡는 경우를 regression에 포함한다.
3. Old controller는 shared lease를 유지한 채 검역하고 root stage 복사를 요청한다.
   Root가 PREPARED를 durable하게 남긴 뒤 old command-owned observer/Host executor의
   실제 Close를 먼저 완료한다. 실패하면 전환하지 않는다. 같은 종료 owner와 원래 cause를
   재사용하며 임의 helper Stop/unknown PID/소유 socket 삭제로 대신하지 않는다.
4. Old controller의 shared lease를 놓은 뒤 installer가 bounded exclusive lease를 잡는다.
   다른 작업을 강제 종료하지 않는다. 고정 HAA Docker names/cgroups/observer endpoints와
   실제 protected process/service 신원을 조회해 drainage를 입증한다. Query 실패/남은
   작업/unknown process는 BUSY 또는 UNKNOWN이며 privileged rebind를 진행하지 않는다.
5. New runtime callers는 exclusive lease 동안 기다리고 journal의 전환/복구 상태를
   검사한다. Old UI/help 프로세스가 있더라도 stale controller가 new runtime을 사용해
   재인증을 우회할 수 없어야 한다. Initial scope에서 입증되지 않은 legacy caller는 거부한다.

## 상태 전이·진단·복구

`PREPARED → SWITCHING → VERIFYING → COMMITTED`를 root-owned durable journal로 기록한다.
실패는 old state로 verified rollback하거나 `RECOVERY_REQUIRED`로 남는다. Original config
bytes/hash/owner/mode, old/current candidate identity와 소유 stage를 먼저 보존하고 sync한다.

- SWITCHING을 먼저 sync한 뒤 고정 HAA systemd unit의 실제 ownership/Exec/MainPID와
  confirmed stop을 확인한다. Observer/client config와 고정 unit의 ExecStart를 새 실제
  version path로 함께 갱신하고 기존 installer primitives로 pointer를 전환한다.
  Docker/runsc/pod-init/다른 unit이나 host-specific runtime 설정은 덮어쓰지 않는다.
- 새 CLI 전체를 root로 실행하지 않는다. 필요한 candidate 진단은 configured ordinary UID로
  실행하며 root가 생성한 고정 inherited channel의 kernel credential로 제한한다. 이 좁은
  진단 경로만 exclusive lease의 borrowed session을 사용할 수 있다. User flag/token으로
  일반 검역의 lease나 권한을 생략하지 않는다.
- Candidate의 return0/자기 보고만으로 commit하지 않는다. Old trusted installer가 exact
  root-owned copy/서명/hash/config/service/socket/peer identity를 독립 재확인한다. 새 실행본의
  doctor와 실제 기존 quarantine workflow 작동은 compatibility 진단이며 이전 ALLOW,
  external lifecycle/cleanup·signed digest 검증을 대신하는 safety authority가 아니다.
- 실패 시 새 service가 confirmed stopped인지 먼저 확인한 뒤 검증된 old config/version/
  helper identity를 복원하고 다시 확인한다. Stop/query/restore uncertainty는 primary error를
  덮지 않으며 journal과 이전 payload를 남기고 새 요청을 막는다. Unknown 상태에서 추측해
  old config를 덮어쓰거나 PID/name으로 broad kill하지 않는다.
- Crash/reboot 뒤 recovery는 같은 journal과 root ownership/signed bytes/service facts를
  재검증한다. COMMITTED 이전 상태를 보고 임의 성공으로 처리하지 않는다. Commit 이후
  response 손실은 communication uncertainty이며 건강한 commit을 몰래 되돌리지 않는다.
- 전용 download/stage/완료 test payload는 마지막 실제 사용 뒤 삭제한다. Active/직전 정상
  version·필수 provenance/작은 journal·검사 원본은 복구 자료이므로 cache와 구분한다.

## 정확한 수정 map과 검사

- `releaseinstall/installer.go` 및 Linux protected-mode leaf: mandatory BundleVerifier를
  유지하고 고정 system root·root-readable native0555/metadata0444, fourth installer role,
  verified existing-version rollback을 기존 copy/atomic pointer로 처리한다. user mode는 보존한다.
  `NewSystemInstaller`는 Linux amd64/root와 고정 `/usr/lib/heliopause`만 허용한다.
  `Installer.ActivateExisting`은 verifier와 실제 기존 version membership/bytes를 먼저
  확인하고 같은 pointer primitive로 복원한다. Bare pointer-switch API는 노출하지 않는다.
- `releaseactivation` adapter와 `cmd/haa-release-installer`: fixed IPC/root copy/journal/
  service/protected registration/lease/rollback. Network-policy service에는 installer API를 넣지 않는다.
- `hosttool`: 실제 protected file/peer/lifecycle primitive만 필요한 범위에서 공유한다.
  새 설치 경계는 old constructor에 digest pin이 있다고 가정하지 않는다.
- `core/ports/update.go`, `application/install_update.go`: 현재 사용될 self-update session의
  per-request Artifact/Verification/Inspection binding만 기존 update pipeline에 연결한다.
  기존 생태계 session의 동작은 유지하고 concurrent isolation/negative stop을 검사한다.
- `bootstrap`: managed lease 획득/정리 순서와 per-request concrete adapter 연결, 좁은
  health entrypoint. Existing runtime cleanup owner를 재사용한다. `cli`에는 실제 서비스가
  준비된 뒤에만 self-update 명령/typed result/일곱 언어 설명과 실제 stage를 연결한다.
- 실제 helper가 구현된 뒤 release build/manifest/SBOM/package/bundle/publish validators의
  정확한 asset set을 함께 확장한다. 과거 v0.1.0 raw evidence는 변경하지 않는다.

Default checks는 root 없는 mechanics/negative tests로 실행한다. Kernel UID/inode/ownership/
socket/lock와 fault/recovery는 disposable namespace/host에서 별도 검증한다. Copy/fsync/
rename/service stop/start/health/cleanup failure, concurrent request/active job, client death,
partial config/crash/journal tamper/foreign ownership/package DB/downgrade/byte swap을 포함한다.
Synthetic trusted-verifier fault test와 실제 SDK/public signed target 검증을 구분한다.
새 네 역할의 signed target은 protected release 후 별도 실제 qualification이 필요하며,
이를 fake fixture/child exit0/기존 v0.1.0의 서명으로 대체하지 않는다.

두 번째 대조: pointer-only 전환, old shared lease를 쥔 채 root exclusive를 기다리는
교착, helper를 바꾼 뒤 old controller cleanup을 수행하는 순서, mutable user receipt,
candidate 자기 보고와 unknown service restore를 모두 제외했다. 기존 user installer와
application/Policy/관찰 owner를 재사용하고 실제 privilege gap만 구현한다.

## 496dd90 이후의 다음 read/lease source map

Copy/pointer 확장의 [로컬 검증](./protected-copy-qualification-20261020.md)은 해당
primitive 범위만 닫았다. Protected registration/proof reader와 runtime lease는
여전히 IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다.

- `releaseactivation/layout_linux.go`: fixed `/var/lib/heliopause`의
  `protected-installation.json`, `protected-update.json`, `runtime.lock`,
  `update.lock`, `installer.lock`, `proofs/<manifest-sha256>/` 및 고정 socket을 소유한다.
  Root-owned/non-writable ancestors와 bounded regular/Nlink1 files만 읽는다.
  Registration은 schema/management/version/manifest digest/client UID/GID만 선언하며
  extra/duplicate/case alias/잘못된 값은 거부한다. 임의 path/UID request 권한은 없다.
- `releaseactivation/installed_linux.go`: 기존 strict manifest/metadata readers와
  fixed SDK를 재사용해 actual installed7개 subject와 root mode/hash, current pointer,
  exact proof/source/runtime identity를 읽기 검증한다. Root-owned record 자체를
  provenance/safety proof로 사용하지 않는다. Active identity view는 private fields다.
- `releaseactivation/lease_linux.go`: marker의 확정 부재만 legacy로 처리한다.
  Journal/부분 상태/조회 오류는 fail-closed다. Root-owned persistent file의 shared
  lock을 확보한 뒤 current registration을 다시 검증하며 Close/handoff는 같은 FD를
  idempotently 해제한다. Signature/inspection/privilege authority는 lease에서 나오지 않는다.
- 초기 record 생성은 독립적으로 verified native bootstrap이 exact native paths와
  healthy runtime/config를 이미 확립한 뒤에만 가능하다. 이 reader가 foreign legacy
  설치를 자동 등록하거나 root client UID를 user claim/SUDO_UID에서 추론하지 않는다.
  Native first-bootstrap/channel/migration gap은 별도로 명시하고 정상 update로 숨기지 않는다.

Tests는 malformed/foreign owner·links·record/source/hash/runtime mismatch, unknown
marker/partial state, actual kernel shared/exclusive blocking·cancel·Close/handoff를
분리한다. 실제 signed native4 target의 positive는 아직 없으므로 crypto mock을
주입한 mechanics를 실제 signed activation으로 보고하지 않는다. Default test에 개인
경로나 privileged 환경의 숨은 입력을 넣지 않는다. 이 단위는 read/lease만 구현하며
public CLI/RPC 성공 handler를 미리 만들지 않는다. 두 번째 대조에서 root owner,
signed digest, kernel caller와 Policy를 서로 대체하지 않는지 다시 확인했다.

## 5717520 이후의 old controller 인증 map

Reader/lease의 [로컬 검증](./activation-read-qualification-20261020.md) 이후에도 old
caller gate는 IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다.
이 다음 변경은 `releaseactivation/peer_linux.go`에 국한하며 RPC 성공 command를
미리 등록하지 않는다.

- Normal managed runtime lease는 independently checked installed view와 actual
  ordinary UID/GID 및 `/proc/self/exe`의 active protected inode가 일치해야 한다.
  다른 user/root/stale CLI는 새 helper/config를 사용하기 전에 거부한다.
- Root-side Unix peer gate는 SO_PEERCRED의 실제 UID/GID/PID를 먼저 확인하고
  existing hosttool.PolicyPeerAuthorizer의 protected executable/SameFile 검사를
  재사용한다. 이전 기본 constructor의 빈 digest를 암묵적 hash pin으로 보지 않는다.
  Signed current bytes 검증은 이미 독립 installed reader가 수행한다.
- [pidfd_open](https://man7.org/linux/man-pages/man2/pidfd_open.2.html)으로 같은 peer
  lifecycle을 붙잡고 exit/reap readiness를 확인한다. PID 숫자/name/claimed UID와
  조회 오류를 살아 있는 authorized peer로 취급하지 않는다. Unsupported kernel과
  unknown state는 fail-closed다. broad signal/kill 경로는 추가하지 않는다.
- Health child에 필요한 groups는 authorized kernel peer의 bounded proc status에서만
  읽는다. Request/SUDO_UID/환경에서 UID·group을 선택하지 않는다. 전후 pidfd 생존을
  확인하고 같은 kernel UID/GID와 함께 보존한다. socketpair 생성 시 credential과
  dropped child's current credential은 앞의 SO_PEERCRED 계약대로 구분한다.
- 실제 imported hosttool과 pinned x/sys/unix API만 architecture map에 추가한다.
  Core/Application은 concrete OS auth를 참조하지 않는다.

회귀는 default pure parsing/자기 pidfd lifecycle과 disposable namespace의 actual
UID/inode/socket credentials를 분리한다. Right basename/wrong inode·foreign UID/GID,
closed peer/process death·stale executable·proc/group 오류·FD cleanup을 검사한다.
Crypto mock snapshot을 사용한 kernel gate 성공은 새 signed update나 whole runtime
activation 성공으로 확대하지 않는다. 이 범위의 두 번째 대조에서 lease/child output/
user receipt를 authorization으로 올리는 우회를 제외했다.
