# P6-D6 — current native-run application 연결 baseline

Baseline80c8148c87b41ba54d31f9b866bdfbc6827aa4a5/tree7122cfa7/3105tracked·runtime1384다.
단일 active item은 M13-004다. D5 production ResolveToolRun 정상·incomplete 거부는 named local
qualified다. CI435는 old b0f4cbc/treeeebd72a1 frozen·실제20분 GET 감시 중이며 새 push하지 않는다.
Poll01은8SUCCESS/macOS build marker89.114949s/default reached·Linux 진행이고 next18:31:42UTC다.

Read: PROJECT/Queue/docs routes와 lessons INDEX·artifact L01/L02·sandbox L03/L08/L12/L18·
evidence L16·ci L05/L06, Architecture001/client09§3/plan20§5·§6, D5 local/처음 반례 원본,
existing freshPythonToolExecutionResolver/pythonConsoleExecutionBinder/newPythonConsoleExecutionContext/
newPythonExecutionPreparationService, pythonExecutionRuntime/newPythonToolExecutionService,
ExecutionPreparationService.Prepare/ExecutionService.Run, current readonly Python execution inputs/
backend·manifest, 실제 기존 Python workflow3/fixture와 unit refusal/cancel tests를 대조했다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| native-run selector/whole source/RO/native console/trace cleanup | YES | YES (production API) | YES (D5 named local) | YES (named local) | current app caller |
| current whole intake/integrity/static+dynamic/Evidence/M3/M5 | YES | YES (old selector) | YES (CI433/434 named) | YES (old scope) | 같은 constructor 재사용 |
| canonical current console binder/ExecutionContext/records | YES | YES | YES (기존) | YES (named) | native-configured command와 요청 command 일치 guard |
| readonly approved Python runtime/application outcome | YES | YES | YES (CI433 current3) | YES (old identity) | 그대로 재사용; tool-install identity 변경 금지 |
| native-run factory/current app | NO | NO | NO | NO | 아래 new3만 연결·실제 current3 |
| native aliases/CLI/hostile matrix/final | NO | NO | NO | NO | 후속 acceptance |

## Source 전 exact map·두 번째 검토

기존 구현은 재작성/대체하지 않는다. Existing production/Core/CLI/CPP1384 inputs를 모두 유지하고
새 bootstrap files3만 추가한다. 예상 runtime1387이다.
- internal/bootstrap/python_uv_run_execution.go: immutable explicit command를 가진 fresh resolver가
  canonical public bare/exact reference를 D5 ResolveToolRun으로 전달하고 실제 native-run identity/
  선택 command를 확인한다. Result의 DATA entry를 approval로 쓰지 않고 resolution만 current
  ExecutionInspectService에 전달한다. Binder는 fresh whole approval 뒤 기존 canonical binder를
  호출하고 configured command와 requested/default canonical command 일치를 확인한다.
  같은 native method와 다른 command의 current binding은 거부하며 cached receipt/set은 없다.
  New internal factory는 기존 intake/integrity/composite inspection/Evidence/M3/M5/records/staging와
  pythonExecutionRuntime을 조립한다. 기존 uv-tool-install factory의 이름/identity/분기는 유지한다.
- internal/bootstrap/python_uv_run_execution_test.go: configured concrete dependency 부재,
  unsupported source/reference/command·original cancellation cause/empty inspection/요청 command
  mismatch의 early refusal을 확인한다. Fake ALLOW/외부 inspection JSON은 만들지 않는다.
- internal/bootstrap/python_uv_run_execution_integration_test.go: 기존 current workflow3 actual의
  같은 public fixture/whole Evidence/records/runtime 검증을 새 factory로 직접 소비한다. Fresh
  native-run identity로 normal0/guest2를 실행하고 configured cowsay와 다른 요청 command는 current
  whole Policy 뒤 runtime/records 없이 거부한다. Program output은 private memory·prefix 범위를
  유지하며 selected native installed output를 실행하지 않는다.

Canonical format/Quick·영향 Bootstrap/Application Race·tagged vet/pinned Staticcheck/build/docs/
security를 모아 통과한 뒤 actual current3 한 묶음만 실행한다. D5 native normal/refusal과 unchanged
CPP132/UV195/기존 backend hostile/CPU actual은 반복하지 않는다. Runtime factory가 actual user
program을 실행하므로 새 identity·fresh current required checks/Evidence·same-context readonly
runtime·정상/typed guest error/불일치 거부·Host/input/Docker/helper/전체 정리를 직접 확인한다.
완료 사본만 원본/ref0/errors0 뒤 정리한다. CI435에는 push하지 않으며 aliases/CLI/final은 별도다.

## D6 named current application local 완료

[Qualification](./p6d6-native-run-application-local-20261030.json)은 existing1384 불변/new3/current1387과 canonical9·Quick72.206s/Bootstrap·Application 전체 Race8.09s, 실제 current native-run application3/115.44s(operator117.806s)를 연결한다. 각 fresh whole current required checks/Evidence/M3/M5 ALLOW 뒤 normal0·typed guest2가 complete이며 configured command 불일치는 runtime/새 staged records 없이 거부됐다. 기존 canonical binder/runtime/records를 재사용하고 tool-install identity는 유지했다. Host/source/42/ZIP/Docker 동일·helperwait0/전체 정리 뒤 종료 사본2개55089722B를 root91proc/ref0/errors0로 제거했다. 누적245ELF+6cache/약10.81GiB Linux 논리 용량이며 active CI435 timer와 원본은 유지했다. 다음 CLI baseline은 기존 npx report/privacy/descriptor/preflight·Python bootstrap owner를 재사용하고 uvx/uv tool run/명시 --from subset을 actual로 검증한다. CI435 oldb0는 poll02의8SUCCESS/Linux 진행으로 frozen이며 새 push하지 않는다.
