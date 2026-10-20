# M13-004 — 보호된 건강 검사의 실제 CLI 호출 보완

2026-10-21 KST, 변경 전 baseline은689e8ef이며 product source는 승인된4ff6c89다.
실제 rc3→rc4 시험에서 required18 모두 COMPLETED/entry7·set ALLOW를 얻었지만 promotion은
FAILED였다. Durable `last-protected-update.json`은 ROLLED_BACK이며 current/등록/config는
rc3다. 사후 ordinary doctor5OK/npm required3·ALLOW도 통과했다. 원본 owner는
`docs/github-logs/self-update-signed-20261020/rc4-actual-*`, `rc4-first-failure-readonly`,
`rc4-base-after-failure-npm`, `rc4-health-argv-actual`이다. 첫 capture가 조회한
`protected-update-history.json`은 canonical history 이름이 아니므로 그 부재를 실제
history 부재로 쓰지 않는다. Canonical owner는 `operation_linux.go:historyPath`다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| actual signed acquisition/quarantine | YES | YES | YES, rc3→rc4 required18/ALLOW | YES, 해당 입력 | 없음 |
| production protected health callback | YES | YES | NO, full callback invalid CLI flag | NO | public CLI 호환 인자·실제 parser/dispatch/error 회귀 |
| actual Root transition/rollback | YES | YES | PARTIAL, actual ROLLED_BACK와 old health | NO | 보완 signed base/target의 정상 완료·fault/auth/crash |

## 확정 결함과 수정 위치

`internal/bootstrap/runtime_guard_linux.go:TryProtectedHealth`의 full callback은
`npm inspect is-number@7.0.0 --format json`을 composition root Run에 전달한다.
공개 CLI의 형식 옵션은 `internal/cli/presentation.go`의 `--output`이다. 동일 서명 CLI로
실행한 exact 기존 인자는0.14s/exit1 `unknown flag: --format`이었다. 이 경로는 정상 full
health를 완료할 수 없다. 아직 이 하나만으로 actual Root 실패의 모든 앞선 경계가
정상이었다고 확대하지 않으며 새 실제 signed 전환으로 확인한다.

기존 구현을 교체하지 않는다. 같은 파일의 내부 callback을 private 함수로 추출해 기존
diagnose/Run 의존성을 명시적으로 전달하고, full nominal 호출의 옵션 하나를 `--output`으로
교정한다. 실제 진입점은 여전히 RunProtectedHealth의 inherited FD/kernel peer/서명·EX
검증 뒤에만 callback을 호출한다. User config·새 public interface·권한/Policy 면제나 새
daemon은 추가하지 않는다. Doctor 거부·rollback의 full=false·nominal 실패 전파를 유지한다.

`internal/bootstrap/runtime_guard_linux_test.go`에는 실제 public CLI parser/dispatch와
기존 synthetic application/Policy fixture를 연결한 회귀를 추가한다. 검사 인자 배열을
복사하거나 source 문자열을 검사하지 않는다. Safe는 완료, REVIEW/BLOCK/operational
failure는 실패, unhealthy doctor는 probe 미실행, rollback은 nominal 미실행이어야 한다.
Mock 기반 결과는 actual signed Root activation의 대체 증거가 아니다.

## 순서·검증·두 번째 대조

1. 계획·Queue를 먼저 갱신한다. 동일 기존 인자를 쓰는 추출 callback과 새 의미 회귀를
   임시 source에서 실행해 RED를 보존한 뒤 옵션 하나를 교정해 GREEN을 확인한다.
2. Canonical format/Quick(compiler·vet·pinned Staticcheck·default tests), security,
   vulnerability/docs와 영향받은 bootstrap/releaseactivation Race를 실행한다. 전체
   후보의 files/blobs/modes를 연결하고 해당 canonical profiles의 순서를 유지한다.
3. 한글 중간 commit·기존 draft PR38을 갱신하고 같은 source의 실제 Required+9 CI와
   native macOS/기존 Linux 소비자를 확인한다. 새 C++ 변경·runtime rebuild는 없다.
4. 기존 rc3/rc4는 그대로 유지한다. 새 signing source/필요 태그는 보완·CI가 끝난 뒤
   구체화하며 현재 승인된4ff6c89 입력을 다른 source로 재지정하지 않는다. 실제 새 signed
   전환·health/auth/fault/recovery를 확인하기 전 전체 기능을 COMPLETE로 바꾸지 않는다.
   현재 수정은 후보 CLI의 callback에만 있다. 기존 rc3 Root installer의
   `runCompatibilityHealth`가 후보 `current.ClientPath()`를 실행하며 rollback은
   full=false다. 따라서 qualified rc3 base를 유지하고 보완된 target 태그 하나로
   이어갈 수 있는 경계다. 내부 FD/protocol·runtime lock은 그대로이며 실제 새 target
   전환 성공 전에는 이 재사용 판단도 전체 acceptance 성공으로 확대하지 않는다.
5. 마지막 소비가 끝난 payload·guest/image/volume·cache/operator는 삭제하고 작은 공개
   proof/manifest·원본 Evidence만 보존한다. Host 원본·공유 runtime은 보존한다.

두 번째 대조: correction은 이미 구현된 health callback의 공개 CLI 옵션 연결 gap이다.
Root role·SDK trust·검역·Policy·service 전환/복원·required coverage를 변경할 이유가 없다.
실제 signed callback의 실패가 드러낸 회귀를 좁게 보강하며 정상 전환은 후속 실제 서명
시험으로만 닫는다. Public Release·main merge·Host 영구 migration은 실행하지 않는다.

## 로컬 결과

예상 RED는 full4 cases 모두 실제 parser의 unknown flag였고, 교정 뒤 semantic7 cases가
통과했다. Canonical format/Quick88.63s/security4.92s/vulnerability11.91s/docs0.23s와
관련 bootstrap/releaseactivation Race39.72s도 PASS다. 최초 제한 sandbox Quick의
Unix socket permission 실패는 원본에 남기고 동일 source의 필요한 권한 실행과 구분했다.
Source·실제 실패/복원·인증 거부·정리 receipt는
[retest qualification](./signed-native-20261020/retest-qualification.json)을 참조한다.
Whole signed 성공은 미완료이며 다음은 같은 후보의 실제 PR38 CI다.
