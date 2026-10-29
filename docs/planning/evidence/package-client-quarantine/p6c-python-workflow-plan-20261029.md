# P6-C — current Python application runtime 연결 전 감사

Baseline516d3268896851092b37e7222ea3debae7786708/tree440d38d0/3056tracked·1365runtime.
M13-004 하나만 active다. D2 named standard-library5/81.36s와 동일 production/D1/CPP의
별도 eight153.86s가 기록됐다. Raw fork-only/ctypes first-fault와 general child/orphan/
held-pipe는 OPEN이며 새 입력의 성공을 인과 해결로 쓰지 않는다. CI432/source9198/tree98ecc/
1361은 Required/all10SUCCESS로 완료했고 monitor는 소유 SIGTERM/exit143을 확인했다.
현재 active CI/timer는 없으며 작은 helper마다 push하지 않고 실제 workflow cohort로 묶는다.

Read: PROJECT/Queue/docs routes와 lessons INDEX·artifact L01/L02, sandbox L03/L08/L12/L18,
evidence L16, ci L05/L06, Architecture001/005/007·Contract002·M5/client09§3/plan20§5,
P6 baseline/A3/B2/D1/D2·CI432 receipts, 실제 application/execution_run.go·execution_prepare/
execution_inspect, Node concrete runtime/factory/actual, current Python binder/preparation/backend/
console reader·uv_tool_install_integration fixture·canonical current-tools DATA exporter를 대조했다.

| 기존 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Generic ExecutionService/Runtime/Result·fresh prepare/late cause/private output/terminal axes | YES | YES (Node) | YES (named actual/CI432) | YES (named) | Python concrete adapter만 연결 |
| Python current preparation/native tool-install identity·opaque console binder/records | YES | YES (internal) | YES (actual/CI432) | YES (named) | 같은 객체를 service factory가 소비 |
| Python Backend/current readonly input/shared capture/typed PythonProgramError | YES | YES (internal) | YES (initial2/D1eight/D2five named) | YES (named) | same-context generic workflow 소비 |
| Python concrete ExecutionRuntime/service factory | NO | NO | NO | NO | source1와 actual public workflow3 |
| Caller cause before reader I/O·missing adapters refusal | NO | NO | NO | NO | adapter의 새 진입 경계 최소 회귀 |
| Native UVx/golden/run equivalence·aliases/CLI/final | NO | NO | NO | NO | 이 internal factory와 별도 후속 |

## Source 전 exact map

- 새 internal/bootstrap/python_execution_run.go: private pythonExecutionRuntime이 기존
  application.ExecutionRuntime을 구현한다. 같은 current context의 EntryArtifact를 기존
  StaticInspector.ReadConsoleExecutionInventory로 재측정하고 기존 SelectConsoleExecutionEntry로
  같은 command를 고른 뒤 기존 ExecutePythonExecution에 같은 context/entry를 전달한다.
  Zero/missing adapter와 취소를 reader I/O 전에 거부하며 원래 caller cause를 보존한다.
- 같은 파일의 newPythonToolExecutionService는 기존 newPythonExecutionPreparationService와
  application.NewExecutionService를 조립한다. Runtime/Policy/manifest·새 port/기능별 Core branch를
  만들지 않는다. Native identity는 기존 uv-tool-install 그대로이며 uvx라고 이름만 바꾸지 않는다.
- 새 python_execution_run_test.go: 취소된 caller의 원래 cause를 reader/검사 I/O 없이 보존하고
  missing concrete runtime이 완료로 보이지 않는 새 진입 경계만 검증한다. Fake approval나
  completed SandboxResult를 만들지 않는다. 기존 generic workflow unit 회귀는 재사용한다.
- 새 python_execution_run_integration_test.go(linux&&realcorpus): 기존 실제 approved fixture와
  same factory의 public cowsay6.1 요청을 사용한다. Normal0/nonzero2의 두 fresh current approvals/
  records와 missing console의 post-Policy preparation failure/no runtime attempt 세 사례를 확인한다.

예상 runtime1368/new3/기존1365 unchanged다. Generic application/domain/CLI·기존 Python reader/
backend/Node/CPP132/권한·한도·startup/trace·CPU 회계를 수정하지 않는다. 현재 owner를 대체할
이유가 없으며 이미 구현한 기능을 다시 만들지 않는다. Future alias/descriptor/CI job은 없다.

## 두 번째 검토와 acceptance

Preparation record는 staging 보관이며 user-run 허가 token이 아니다. 같은 context의 전체
current approval/runtime/source/console binding을 backend가 다시 소비하고, 프로그램 출력과
exit는 private runtime DATA다. Guest nonzero의 typed PythonProgramError와 operational incomplete를
같은 성공으로 합치지 않는다. Late caller cancellation은 기존 workflow의 완료 판정을 내리며
원래 cause/cleanup은 보존한다. Raw argv/output는 records/일반 formatter·JSON에 남기지 않는다.

Native public selector는 current tool-install recipe만 사용한다. 이 named workflow 성공으로
UVx/run/native cache semantics·aliases/CLI·설치 target 지원을 선언하지 않는다. No InstallTarget/
Host native wheel execution이나 tool-install identity relabel은 없다. 새 runtime이 unknown source/
console을 binder 밖에서 선택하거나 artifact shebang/PATH/shell을 사용하지 않는다.

Canonical format/focused/Quick·Bootstrap/Application 전체 Race, tagged vet/pinned Staticcheck/
bootstrap build/docs/security를 먼저 실행한다. 같은 source/current CPP132와 canonical created-
never-started pinned interpreter DATA exporter/보호 namespace에서 public actual3를 수행하고
Manifest/SBOM·fresh current Evidence/readback·source/Host5/42/ZIP/Docker·helperwait0/전체 정리를
확인한다. 실패는 첫 원본을 보존한 뒤 해당 원인만 교정하며 이전 통과 입력을 근거 없이 반복하지
않는다. 완료 사본은 원본/inode/hash/전체 proc 무참조 뒤 정리한다. 실제 workflow cohort가 닫힌
후 필요한 grouped push와 새 run/head/20min timer를 등록한다.

현재 단계의 완료 기준은 internal workflow3 named qualification이다. Native UVx/golden/
aliases/CLI/final profiles/signed와 earlier fork-only/CPU/drop 원인은 이후에도 OPEN이다.

## Named local 결과

기존1365 불변/new3·현재1368의 canonical9/Quick323.823s/Bootstrap·Application 전체 Race22.892s가 PASS다.
Actual public3/95.89s(operator98.513s)는 fresh current ALLOW 뒤 normal0/typed guest2와 missing-console
PREPARATION_FAILED/no attempt/no staging change를 확인했다. Native selector는 uv-tool-install이며
UVx equivalence가 아니다. Host5/42/ZIP/Docker/source 불변·helperwait0/전체 정리와 원본 readback을
확인했다. Root85proc/ref0/errors0 뒤 완료 ELF 사본2개54905002B만 제거해 누적209/~7.12GiB다.
다음은 기존 current CI step에 boundary8/hostile5/workflow3를 묶는 P6-C2 감사다.
