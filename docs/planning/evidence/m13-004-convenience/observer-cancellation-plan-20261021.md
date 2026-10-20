# M13-004 — 취소와 observer child 수명 경계 보완

상태: 사전 원인·두 번째 대조 뒤 최소 수리 / 로컬·원격 source 검증 완료, actual signed caller 남음. M13-004만 IN_PROGRESS다.
Actual signed rc3→rc5 정상 전환·OS 인증 성공은 보존한다. 이 항목은 검역·Policy·
Root 설치 역할을 재구현하지 않는다.

## Baseline과 정확한 gap

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 observer identity/readiness/explicit Stop/supervisor cleanup | YES | YES | 기존 정상 소비자·CI357 | 해당 기존 범위 YES | 재구현 없음 |
| operation 취소 뒤 owner의 graceful child 종료와 다음 검사 | YES, 현재 연결 결함 | YES | NO | NO | actual child RED, 최소 수리, 공통 소비자와 signed caller 재검증 |

Source7fedaf3의 `internal/hosttool/observer.go:Launcher.StartObserver`는 long-lived
helper를 `exec.CommandContext(ctx, ...)`로 실행한다. Parent operation 취소가
supervisor의 별도 bounded Stop보다 먼저 child를 강제 종료할 수 있다. 이미 종료한
child의 `observerProcess.Stop`은 ExitError를 반환하고 supervisor는 불확실한 정리의
lock을 유지한다. 실제 SIGINT 시험은 operation/context 오류와 cleanup 실패를
보존했고, 다음 npm은 owned endpoint로 거부됐다. Root journal/history는 없고
current rc3였다. 독립 Root readback에서 observer 프로세스·컨테이너·slice는 없었고
control socket/lock만 남았다. 원본 owner는 `self-update-signed-20261020/rc5-cancel-*`다.

## 수정 위치와 순서

1. 기존 검증 후 launch/readiness/Stop body만 private 함수로 추출한다. Public
   Launcher의 root-owned executable·SameFile·build identity·고정 argv/env 검사는
   그대로 앞에 남긴다. 실제 OS child로 readiness 후 cancel, readiness 중 cancel,
   정상 Stop과 예상 밖 death를 검사한다. 우선 unchanged lifecycle의 RED를 보존한다.
2. Long-lived helper만 explicit supervisor 소유로 실행한다. One-shot build identity
   조회의 CommandContext, readiness의 parent deadline, 시작 실패의 bounded Stop,
   supervisor Close/fail-stop, 실제 wait/Done와 unexpected death 오류는 유지한다.
   취소 오류를 성공으로 바꾸거나 nonzero child exit를 일반적으로 무시하지 않는다.
3. Canonical format/Quick/security/vulnerability/docs와 hosttool·sandbox·bootstrap
   관련 Race, 실제 observer/기존 runtime 소비자를 확인한다. 동일 source 원격 Required와
   native macOS도 필요하다. Source/edit inventory와 원본 RED/GREEN을 연결한다.
4. 이 결함은 업데이트를 실행하는 **기존 controller**에 있으므로 rc3로 새 target만
   검사해도 취소 수리를 검증할 수 없다. 교정된 caller의 actual signed qualification은
   별도 입력이다. rc5나 이전 immutable ref를 바꾸지 않고 구체 source/필요한 최소
   새 signed 입력을 원격 qualification 뒤 제시한다. 새 tag 승인을 묵시적으로 확장하지 않는다.

두 번째 검토: observer의 identity/권한/관찰 프로토콜이나 cleanup ownership을 바꿀
필요가 없다. Parent cancellation과 explicit lifecycle의 두 소유자를 하나로 맞추는
기존 owner 수정이다. Fake launcher만의 supervisor test는 exec 자동 kill을 거치지
않으므로 실제 OS child 회귀가 필요하다. 예상 밖 death·교체된 endpoint·UNKNOWN
보존을 계속 검사하며, 실제 signed cancellation과 작은 회귀의 성공을 구분한다.

## 확인된 로컬 범위

Private 추출만 적용한 실제 child RED는 readiness 후/중 취소2개 실패였고, 최소
수리 뒤4 cases가 GREEN(1.16s)이다. Canonical format과 Quick240.70s, vulnerability14.25s는
PASS다. Hosttool/Bootstrap/Releaseactivation Race는 PASS였고 확대 Sandbox suite는
기존7개 data race로 FAIL였다. 정확한 변경 전7fedaf3에서6개를 동일 선택 실행으로,
나머지 attribution1개는 bounded10회 대조에서 재현했다. 이 결과를 전체 Race 성공으로
쓰지 않는다. 영향받는 supervisor Race1.98s도 PASS다. 최종 전체 후보 docs/security와 실제 경로·blob/mode
대조를 source commit 전에 수행하며 원본 rc5-final-*에 보존한다. 새 source의 원격 CI와
actual signed caller는 별도 qualification이다.

두 번째 source 대조에서도 public Launcher의 절대 경로·Root ownership·SameFile·
build identity와 고정 argv/env, startup failure Stop·readiness 한도는 유지됐다.
새 capability나 observer protocol 변경 없이 기존 body의 process 수명만 owner에 맞췄다.

최종 source4a8c278의 [CI358](./signed-native-20261020/observer-remote-ci.json)은 Required
포함10 SUCCESS다. 실제 전체 checkout tree/mode와 native macOS/C++103/Linux45·
CPU291.99s/helper5 wait0/final confirmed cleanup을 확인했다. 새 CUDA full/전체
Sandbox Race 성공이나 실제 signed 취소 성공으로 확대하지 않는다. 다음 입력은
[고정 source와 단일 rc6 제안](./signed-caller-plan-20261021.md)이며 owner 승인 전이다.
