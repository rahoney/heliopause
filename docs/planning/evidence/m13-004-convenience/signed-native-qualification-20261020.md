# M13-004 — 실제 서명 입력·native 초기 등록·ELF 호환성 검증

[실행 계획](./signed-lifecycle-plan-20261020.md),
[격리 환경](./native-systemd-fixture-plan-20261020.md),
[source 이전 보완 계획/두 번째 검토](./native-elf-compatibility-plan-20261020.md)를 따른다.
작은 [재현 receipt](./signed-native-20261020/qualification.json)와 두 실제 manifest/proof를
보존했다. 원본 owner는 `docs/github-logs/self-update-signed-20261020/`다.

## 통과한 범위

- a22aa12의 보호된 v0.1.1-rc.1/[build39402318182](https://github.com/rahoney/heliopause/actions/runs/39402318182),
  v0.1.1-rc.2/[build39405360601](https://github.com/rahoney/heliopause/actions/runs/39405360601)는
  official tag-push build/attestation SUCCESS다. 각13파일의 크기·digest·manifest/metadata와
  exact source/tag/workflow/run/subject를 확인했다. 고정 gh2.102.0의 actual UID1000,
  network namespace/offline root 검증에서 각13/13 통과했다. 임시 ZIP은 추출 뒤 삭제했다.
- Production Root SDK의 실제 rc1 파일 준비·같은 intent 재시도·ordinary 읽기/쓰기 거부와
  wrong-tag/source proof·CLI 변조 거부8 cases가 통과했다. 첫 파일 준비는 정직하게
  registration=PENDING/health=UNVERIFIED를 보고했다. 이 단계의 Host 원본 신원은 불변이다.
- 별도 실제 systemd/고정 Docker29.8.1·gVisor·네 runtime images 환경에서 rc1 파일 준비,
  service identity/cgroup, installed helper register가 통과했다. 일반 UID1000 doctor5OK와
  npm required3 완료/ALLOW도 실제 통과했다. 전체 본체 업데이트 거부 뒤에도 같은 doctor와
  nominal 검역이 다시 통과했다. Native bootstrap source 정리·current 유지·journal 부재와
  실제 서비스·자원 drain 및42개 Evidence를 보존했다.
- 실제 rc2 전체 세트의 authenticity/static7개씩은 완료됐다. 처음 native dynamic은
  미완료여서 MANUAL_REVIEW/NOT_PERFORMED였고 Root 전환은 시작하지 않았다. 원인을
  첫 Go artifact의 자기 cgroup 읽기와 독립 C++ artifact의 고정 libm 읽기로 구분했다.
  최종 후보에서는 추가 OS 표시 marker의 실제 fault도 정확히 확인해 보완했다.
- 기존 classifier에만 고정 sandbox own metadata/native ABI 분류를 추가했다. 새 positive는
  unchanged observer에서 실제 RED였고 나머지102 latch는 보존됐다. 추가 OS marker도 첫
  후보에서 별도 RED였다. 최종 고정 canonical builder/Bazel/Go의 전체103 latch는 PASS다.
  Artifact 역할·정확한 세대/handoff/mount/읽기 flags, honeytoken 우선순위와 관찰 한도는
  유지했다. 최종 Quick36.91s/security5.53s/docs0.21s도 통과했다.
- 미등록 개발용 fixture에서 후보 observer와 actual signed rc2 native4를 각각 독립 검사해
  모두 required3 완료/ALLOW를 확인했다. 기존 npm required3/ALLOW와 미등록 self-update의
  SELF_UPDATE_MANAGED_INSTALLATION_REQUIRED/exit1도 확인했다. 임시 HTTPS transport만
  fixture이며 signature root·검역·Policy·권한 검사는 면제하지 않았다.

최종 source4ff6c89의 [CI356](https://github.com/rahoney/heliopause/actions/runs/39416778306)은
Required 포함10개 모두 SUCCESS다. [원격 receipt](./signed-native-20261020/remote-ci.json)는
실제 checkoutc0a0ddd와 전체 tree/mode 일치, native macOS·전체103 C++ latch, Linux45개
소비자와 CPU full293.82s, helper5개 wait0 및 final original0/CONFIRMED_STOPPED/
helper_stop0/client_error0/client_restore0를 연결한다. 실제 CUDA full은 이번 실행에 포함하지
않았다. 후속 시험은 [승인된 고정 소스 재서명 계획](./signed-retest-plan-20261020.md)을 따른다.

## 보완 소스의 새 signed base — rc.3

사용자가 source4ff6c89의 rc.3·rc.4 순차 생성을 승인했다. rc.3의
[official build39439522714](https://github.com/rahoney/heliopause/actions/runs/39439522714)는
SUCCESS이며 exact13파일·manifest/SBOM/runtime 결합·독립 UID1000 offline13/13를
확인했다. 실제 Root stage의 manifest/installer 독립 검증 뒤 production SDK first files,
새 systemd 환경의 실제 service/register, ordinary doctor5OK/npm required3·ALLOW가
통과했다. Runtime6/pod-init lock과 이미지4의 exact digest/RootFS/Config도 일치했다.
[manifest](./signed-native-20261020/rc3-manifest.json)와
[public proof](./signed-native-20261020/rc3-proof.sigstore.json)를 보존했다.

Docker29 containerd store는 처음부터 owned ext4 volume에 두었다. 활성 소비자 부재를
확인한 base 파일시스템 image는 후속 fault case의 순차 재현용이며 같은 volume의 동시
사용은 금지한다. Host 보호 신원10개는 이전 final과 일치했다. 완료된 rc3 다운로드13개
231,994,106B는 inode/hash·FD/maps/mount 참조와 query errors0 확인 뒤 삭제했다.
실제 설치와 필요한 기준 image/volume은 후속 소비가 끝날 때까지 보존한다.

원본은 같은 raw owner의 `rc3-acquisition`, `rc3-native-first-files`, `rc3-register`,
`rc3-doctor`, `rc3-npm-nominal`, `rc3-image-qualified-*`,
`rc3-qualified-baseline-image`, `rc3-completed-download-*`다. 초기 exact-bundle 검사에는
시험 proof가 payload 디렉터리에 함께 있어 파일 수가 거부됐다. Proof를 바깥으로 옮긴
unchanged official13의 canonical 검사는 PASS다. Snapshot commit 뒤 최초 returned-ID
readback도 비어 operator가 실패했으며, 실제 tag/label readback으로 존재를 확인했다.
원래 commit stdout은 보존되지 않았으므로 정확한 원인은 미확정이다. 두 준비 오류는
제품 결함이나 업데이트 성공 증거로 취급하지 않는다.

그 뒤 같은 source에 rc.4를 생성했다. [official build39442575693](https://github.com/rahoney/heliopause/actions/runs/39442575693)는
SUCCESS다. rc4도 exact13/독립 offline13/13/canonical bundle이 통과했다. 실제 파일은
observer와 runtime-images만 rc3와 같고 나머지11개는 달랐다. 같은 product source의
version/path/metadata 전환이며 code-change regression으로 확대하지 않는다.

## 실제 전체 검역·실패 복원과 건강 검사 CLI 보완

[새 receipt](./signed-native-20261020/retest-qualification.json)가 정확한 범위를 소유한다.
설치된 rc3의 production self-update에서 actual rc4 selected7의 required18가 모두
COMPLETED였고 entry7/set ALLOW였다. Promotion은 FAILED였으며 canonical durable
history `last-protected-update.json`은 ROLLED_BACK, current/등록/config는 rc3다.
사후 doctor5OK/npm required3·ALLOW도 실제 통과했다. 실제 Root 실패의 모든 앞선
경계가 정상이었다고 단정하지 않는다. 성공한 전체 전환은 아직 없다.

Full health callback의 기존 `--format json`은 public CLI에서 거부됐다. 정확한
[사전 계획·두 번째 검토](./protected-health-cli-plan-20261021.md) 뒤 기존 callback을
private 함수로 추출하고 옵션 하나를 `--output`으로 교정했다. 동일 인자의 actual
CLI0.14s/exit1과 실제 parser/dispatch RED4 cases를 보존했다. GREEN semantic7 cases는
ALLOW 완료, REVIEW/BLOCK/operational failure 전파, unhealthy 진단과 rollback의
nominal 미실행을 확인했다. Format/Quick88.63s/security4.92s/vulnerability11.91s/
docs0.23s와 bootstrap/releaseactivation Race39.72s는 PASS다. 최초 제한 sandbox의
Quick은 Unix socket operation-not-permitted였고 같은 source의 권한 교정 실행을
분리했다. 이 로컬 보완은 actual 새 signed target의 성공을 대신하지 않는다.

별도 OS 인증 거부 시험에서 fixture NOPASSWD grant를 잠시 제외하고 timestamp를
지웠다. 실제 sudo denial exit1와 required18/ALLOW 뒤 Root transaction 미생성,
current/history 불변을 확인하고 exact fixture grant를 복원했다. OS 인증 성공·정상
전환·추가 service/crash/recovery qualification은 여전히 남는다.

Evidence42개/24,176B와 작은 manifest/proof를 보관했다. 완료된 rc3/rc4 다운로드26개
합계463,988,221B는 hash/inode·FD/maps/mount·query errors0 확인 뒤 삭제했다. rc4
guest의 actual drain/transport 종료를 확인한 뒤 해당 container를 제거했다. 같은 owned
volume을 다른 container가 사용하지 않음을 확인하고 qualified rc3 image로 순차 복원해
실제 cold start/doctor/npm을 다시 통과했다. 이 과정에서 완료된 rc4 guest rootfs/transport와
시험 cache도 함께 제거됐다. 필요한 rc3 base/image/volume은 후속 target 검증용이다.
이 cold start는 Host kernel/WSL reboot의 증거가 아니다.

보완 source7fedaf3의 [CI357 원격 receipt](./signed-native-20261020/health-cli-remote-ci.json)는
Required 포함10개 실제 SUCCESS와 모든 job의 checkout6d65dfc를 확인한다. 전체 Git
tree/mode는 source와 일치하며 native macOS/C++103/Linux45·CPU full226.62s/helper5
wait0/final confirmed cleanup도 PASS다. 새 CUDA full 실행은 없다. 다음 입력은
[qualified rc3와 고정 rc5 target 하나](./signed-target-plan-20261021.md)로 구체화했다.
앞서 승인된 rc3/rc4의 source와 ref는 유지했고 새 target은 아직 owner 승인 전이다.
필요한 rc3 guest는 실제 drain/journal 부재 뒤 idle stop해 메모리를 해제했다. Host 보호
신원10개·runtime6/pod-init·공유 image4도 마지막 read-only 재대조에서 불변이었다.

## rc.5 실제 정상 전환·OS 인증·장애 복구와 취소 gap

사용자 승인대로 source7fedaf3의 rc5 하나만 생성했다. [Official build39456168381](https://github.com/rahoney/heliopause/actions/runs/39456168381)은
SUCCESS이며 exact13/고정 gh2.102.0·UID1000·독립 network namespace offline13/13와
canonical manifest/SBOM/runtime 결합 검사가 통과했다. [공개 receipt](./signed-native-20261020/rc5-qualification.json),
[manifest](./signed-native-20261020/rc5-manifest.json)와 [proof](./signed-native-20261020/rc5-proof.sigstore.json)를 보존했다.

- Actual rc3→rc5는 required18 COMPLETED/entry7·set ALLOW, promotion COMPLETED,
  durable history COMMITTED/current rc5였다(39.06s). 보호 사본7의 해시·Root 소유권·mode와
  ordinary doctor5OK/npm required3·ALLOW도 통과했다.
- 새 rc3 기준에서 fixture NOPASSWD grant를 제외한 sudo denial을 확인했다. 실제
  비밀번호를 같은 PTY의 OS sudo에만 전달하고 ordinary CLI로 갱신해 COMMITTED,
  새 doctor/npm과 exact copy를 확인했다. 비밀번호는 로그·인자·제품 설정에 저장하지 않았다.
- 동시 요청은 두 번째가 acquisition 전에 owned observer endpoint로 거부됐다. 첫
  요청만 COMMITTED였고 이후 doctor/npm은 정상이었다. 이를 두 installer 동시 진입
  성공이나 update-lock 오류 코드의 실제 관찰로 확대하지 않는다.
- 후보 서비스의 PID/exe/starttime/cgroup을 확인하고 pidfd SIGKILL을 주입했다. 원래
  required18/ALLOW 뒤 FAILED/ROLLED_BACK과 old doctor/npm이 확인됐다.
- Root installer를 VERIFYING에서 같은 방식으로 중단했다. 확인된 HAA process·자원
  정지 뒤 journal bytes를 보존해 guest PID1 cold start했다. Installed recover5.01s는
  ROLLED_BACK/journal 정리/old doctor/npm을 통과했고 동일 target 재실행52.76s도
  COMMITTED/새 doctor/npm/exact copy를 통과했다. Host kernel/WSL reboot는 아니다.
- SIGINT 취소는 기존 rc3와 Root journal 부재를 유지했지만 observer cleanup이 실패했다.
  독립 조회에서 프로세스·컨테이너·slice는 없고 control socket/lock이 남았으며 다음 npm이
  거부됐다. [사전 원인·수정 계획과 두 번째 검토](./observer-cancellation-plan-20261021.md)를
  따른 최소 owner 수리는 실제 child RED2→GREEN4다. 새 source의 전체 검사·원격 CI와
  **교정된 기존 caller**의 실제 서명 취소 검증은 별도다. 전체 acceptance를 닫지 않는다.

전용 환경의 초기 controller 준비 실패는 실제 cgroup 위임 후 동일 rc3/npm으로 교정했다.
Cold start는 Docker가 hosts를 재생성하고 fixture HTTPS도 비활성이었으므로 첫 재시도는
공개 API404였다. 원본 실패와 사본 감사의 current 기대 실패를 보존하고 transport 준비 뒤
같은 signed bytes로 재시도했다. 이를 제품 복구 실패나 처음부터 성공한 시험으로 쓰지 않는다.
완료된 rc5 다운로드13/231,996,591B는 실제 참조·query errors0 확인 뒤 모두 삭제했다.
완료된 guest/TLS/임시 비밀번호는 순차 기준 복원으로 제거했고, 필요한 rc3 기준만
ordinary 건강·drain/journal 부재 확인 뒤 idle stop했다.

## 현재 판정

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| actual signed13 producer/독립 검증 | YES | YES | YES, a22aa12 rc1/rc2와4ff6c89 rc3/rc4 | YES, 해당 입력 | 없음 |
| actual first files/register/systemd/doctor/npm | YES | YES | YES, disposable Linux guest | YES, 해당 mechanics | 공개 최초 획득/설치는 별도 |
| raw ELF 호환성 보완 | YES | YES | YES, local103 latch/native4/npm와 CI356 | YES, 해당 범위 | native 전체 전환은 별도 |
| signed base→target native 활성화·복구 | YES | YES | YES, rc3→rc5와 위 실제 범위 | YES, 정상/OS 인증/서비스·Root 중단/guest cold recovery | kernel power-loss는 미시험 한계 |
| operation 취소 뒤 다음 검사 | YES, owner 연결 수리 중 | YES | NO, actual 실패/로컬 child4 GREEN | NO | 교정 source의 전체 검사·원격 CI와 actual signed caller |
| M13-004 전체 설치 채널 | PARTIAL | PARTIAL | NO | NO | 위 native gap, owner 이름/publisher binding/각 채널 |

개발용 observer는 새 source의 unsigned 후보이며 native 관리 등록이 없다. 이 성공을
새 signed Root activation이나 전체 본체 업데이트 완료로 전용하지 않는다. 기존 rc1/rc2는
보완 전 observer를 포함하므로 재지정하지 않는다. 수정된 구체 후보와 원격 CI를 검토한 뒤
새 서명 시험 입력으로 이어갔으며 위 rc3 base까지 확인했다. Public Release 발행·main merge·Host 영구 migration은 없다.
새 CUDA 전체·native macOS 실행이나 실제 Host kernel/WSL reboot를 이 로컬 결과로 주장하지 않는다.

## 준비 오류·정리·재검토

초기 image stream의 ref/ID 확인 실패, nested overlay EINVAL, copy2의 uid/gid83개 누락,
filesystem별 directory st_size 차이와 준비 직후 TLS readiness race는 원본에 별도로 남겼다.
내용·mode/owner·xattr·membership을 다시 확인한 원본 보존/reversible mount로 환경을 교정했고,
이를 제품 수정 성공이나 서명 오류로 합치지 않았다. 첫 doctor의 미지원 JSON 옵션과
Staticcheck cache의 read-only 권한 실패도 정상 결과로 재분류하지 않았다.

두 전용 guest와 volume, task-label image 및 직접 가져온 base 참조는 삭제·부재 조회를
완료했다. Stop command 성공과 PID1 exit130은 구분하며 reboot qualification으로 쓰지 않는다.
완료된 payload/build/cache/operator37,051개/unique regular logical1,163,651,761B도 삭제했다.
[정리 receipt](./signed-native-20261020/cleanup.json)가 실제 범위를 연결한다. 마지막 CI 기록의
문서 검사에 계속 쓰는 source/cache/helper는 그 소비 뒤 정리한다. 작은 public proof/manifest와
필요 Evidence는 보존했고 Host 보호 신원10개·runtime6/pod-init·shared image4는 불변이다.

두 번째 검토: 서명 성공, 파일 준비, 등록/진단, native4 동적 호환성과 signed whole activation의
증거를 분리했다. 첫 fail-stop 뒤 후속 미완료를 별개 원인으로 확대하지 않았고 기존 source를
다시 구현하지 않았다. M13-004는 계속 IN_PROGRESS다.
