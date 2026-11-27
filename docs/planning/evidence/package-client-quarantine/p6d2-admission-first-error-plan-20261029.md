# P6-D2 — 기존 direct-exec admission의 최초 오류 보존 감사

Baseline569c2d75f152d3bbce27b598697736736dfa85d7/treecfbb82d6867a955b39630bcd6e76c6b009b66fcc,
3068 tracked/runtime1369다. M13-004 하나만 active이며 CI433/source87c817d/1368은 frozen이다.
실제 poll01은8SUCCESS/Linux 진행, next12:42:09.907868UTC다. Worker71198/PID3172981을 유지한다.

Read: PROJECT/Queue/docs routes·lessons INDEX와 sandbox L03/L08/L12/L18·evidence L16·ci L05/L06·
artifact L01/L02, Architecture001·client09§3/plan20§5, D1 actual-v1/v2 source/raw/receipt와
direct_exec_admission.go 전체 runner methods·prepare/finalize 및 기존 protocol/native-exit/
lifecycle 회귀를 대조했다. D1 observer FAIL/cache-query missing은 이 변경과 별도다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Admission nonce/lease/session/arm/consume/cancel/complete·fail-stop | YES | YES | YES (기존 named/hosted) | YES (기존 범위) | 상태/권한/budget/완료 판정 유지 |
| RunDiscard의 commandErr+finishErr join·typed exit 제한 | YES | YES | YES (기존 회귀/소비자) | YES (named) | 기존 owner를 기준으로 재사용 |
| Output/RunInput/RunBounded/RunOutput의 dual-error 보존 | NO | NO | NO | NO | 네 반환에서 native 원인 보존 |
| 위 네 경로의 실제 native ExitError + cancel/consumed 회귀 | NO | NO | NO | NO | 동일 test의 RED4/GREEN·기존 RunDiscard positive 대조 |
| Native cached query/public golden·aliases/CLI/final | NO | NO | NO | NO | 이 오류 보존 수정으로 닫지 않음 |

D1 v2는 같은 validated Host executor의 native exit2/EACCES/stderr4683B를 recorder가 보존했지만
outer Output은 finishErr의 observer refusal만 반환했다. Source의 나머지 세 method도 동일 반환이고
RunDiscard는 이미 errors.Join을 사용한다. 이는 관찰기 IMAGE fault의 직접 원인이 아니다.

## Source 전 exact map·두 번째 검토

- 새 internal/sandbox/direct_exec_error_preservation_test.go: 기존 실제 OS exit status와
  startRealObserverForTest의 protocol fixture를 재사용한다. Consumed/unconsumed에서 네
  method와 already-correct RunDiscard의 원래 ExitError·errDirectExecNotStarted·출력 prefix를
  대조한다. Fake Approved/Policy/SandboxResult를 만들거나 protocol fixture를 실제 gVisor
  qualification으로 쓰지 않는다. 동일 새 test를 수정 전 실행해 네 실패를 먼저 보존한다.
- internal/sandbox/direct_exec_admission.go: 기존 Output/RunBounded의 return output,finishErr와
  RunInput/RunOutput의 return finishErr 네 곳만 errors.Join(commandErr,finishErr)로 바꾼다.
  RunDiscard/nonce/session/lease·준비/완료/poison/fail-stop/예산/실제 분기와 반환 output은 유지한다.

예상 runtime1370/new1/modified1/기존1368 불변이다. 새 interface/helper·CPP·권한·alias·CI job은
없고 구현된 admission을 재작성하지 않는다. Command 실패를 완료로 바꾸지 않으며 원래 오류와
완료 불확실성을 함께 반환한다. Native stdout/stderr는 기존 private transport 경계를 유지한다.

Canonical format/focused RED→same-test GREEN/Quick·sandbox 전체 Race/tagged vet/pinned
Staticcheck·sandbox/bootstrap test build/docs/security를 수행한다. 실제 D1 network-none probe는
원래 native EACCES가 errors.As로 보존되더라도 observer incomplete 때문에 계속 FAIL이어야 한다.
기존 current Python/Node/Pipx 영향 소비자는 source/CPP 동일성에 맞춰 선정하며 불필요한 동일
검증을 반복하지 않는다. CI433 frozen source와20min actual GET을 유지하고 이 작은 helper만
따로 push하지 않는다. 원본·Host·고정 입력·전체 teardown을 보존하고 완료 사본은 참조 확인 뒤 정리한다.

## Actual 선정

Canonical10/동일 회귀GREEN10·Quick333.493s/전체 sandbox Race34.713s를 통과했다. 실제 검증은
(1) 같은 network-none D1 probe의 native ExitError2/EACCES가 outer errors.As에 보존되면서
기존 observer fault로 Go parent는 계속 exit1/FAIL인 negative 계약과 (2) 수정하지 않은 canonical
ci-uv-tools.sh python-execution profile의 fresh public normal0/nonzero2 두 사례를 직렬 수행한다.
동일 CPP132와 기존 Node/Pipx/default 소비자 source를 재사용하며 성공 경로·lease/state/권한을
바꾸지 않은 네 반환에 대해서 다른 actual 전체를 자동 반복하지 않는다. Negative operator의
계약 검증0과 Go parent1/관찰기 미완료를 구분해 기록한다. Native cache-query/golden은 OPEN이다.

## Named local 결과

동일 회귀 RED4/기존 control6→GREEN10과 canonical10/Quick333.493s/전체 sandbox Race34.713s를
확인했다. Actual negative는 원래 native ExitError2/EACCES가 outer에서 보존되면서 원래 observer
미완료로 Go parent5.01s/exit1/FAIL을 유지했다(operator contract0). 같은 source/CPP의 canonical
Python actual2/79.40s는 fresh current ALLOW·complete0/2와 원본·Host·입력/전체 정리가 PASS다.
이로써 네 반환의 오류 보존만 닫고 native cache-query observer/golden/alias는 OPEN으로 유지한다.
Root87proc/ref0/errors0/정확한6 resource-empty zombie 재검증 뒤 완료 ELF4개85908816B를 제거해
누적217/~7.26GiB Linux 논리 용량이며 모든 원본과 CI433 monitor는 보존했다. 다음 실제 cached
query를 기존 UV owner의 좁은 identity/topology/readonly/ARTIFACT 계약으로 다룰 baseline을 감사한다.
