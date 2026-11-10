# P5-C4 — 준비와 실제 실행을 잇는 application workflow

Baseline98b1a9d8e0aa5f29a85234eb3b4818f0e52064e8/tree577ac6bf/2978tracked·1332runtime.
M13-004 하나만 active다. CI428/source558d75a는 poll03 01:02:15UTC 8SUCCESS/Linux 진행이며
timer04 due01:22:15UTC/session34757를 먼저 소비한다. 실행 중 새 push하지 않는다.

Read: Queue/PROJECT-DECISIONS/docs README route, client09§1..4/plan20§4..6,
Architecture001/005/007, Contract002 invariants, Domain003, Coding/Security rules와
lessons INDEX/sandbox L03/L08/L12/L18·artifact L02/evidence L16/ci L05/L06.
Existing execution_prepare/inspect, node_execution_prepare/materialize, Backend.ExecuteNodeExecution,
Host private capture와 current production/boundary actual callers를 직접 대조했다.

| 실제 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Application.ExecutionPreparationService.Prepare | YES | YES(factory/actual caller) | YES(named public2) | YES(storage preparation) | run workflow에서 동일 current result만 소비 |
| Bootstrap.materializeNodeExecution와 Backend.ExecuteNodeExecution | YES | YES(manual actual caller) | YES(public2/controlled12) | YES(named backend) | application이 호출할 좁은 concrete adapter |
| Current verifier/inspection/Evidence/Policy/records/staging | YES | YES | YES(named current scopes) | YES(named scopes) | 새 승인/가짜 InstallTarget 없이 그대로 연결 |
| Application one-request execution workflow/result | NO | NO | NO | NO | preparation→동일 execution context→runtime, 독립 결과·원래 오류·late cancellation |
| public CLI/안전한 stream renderer | NO | NO | NO | NO | C4 다음 별도 단위; 이번 source에 placeholder 없음 |

## 수정 지도 — source 전

Application의 새 execution_run.go는 현재 concrete preparation service를 재사용한다.
현재 필요한 runtime boundary만 Application interface로 선언하며 Domain.ExecutionContext를 받고
완료·독립 terminal과 private stdout/stderr accessors를 가진 interface 결과를 반환한다.
Application은 Sandbox/npm/Hosttool/Cobra를 import하지 않는다. Bootstrap의 작은 adapter가
기존 materializeNodeExecution→ExecuteNodeExecution을 호출한다. 기존 구현·권한·CPP·Policy·
45s/1MiB·입력/cleanup 규칙을 변경하거나 capture/검사/준비 본문을 복제하지 않는다.

Application.Run은 caller request를 먼저 검증하고 Prepare의 fresh current operation/set/argv를
그대로 runtime으로 넘긴다. 준비 실패/취소/invalid receipt에서는 실행0이다. 전체 result는
preparation과 runtime completion을 분리한다. Completed guest nonzero/signal은 원래 typed cause를
보존한다. Missing/모순 terminal, incomplete runtime, unexpected nil error, late caller cancel은
completed로 표시하지 않는다. 출력은 private memory이고 getter가 복사하며 formatter는 고정
문구/JSON 직렬화 거부다. 실행 결과를 Policy/Evidence 승인으로 재구성하지 않는다.

새 application unit tests는 기존 preparation fixture의 실제 constructor/workflow를 재사용하되
runtime/기존 port doubles는 orchestration contract 검사라는 범위로 표시한다. Fake positive
Bundle/InstallTarget/SandboxResult를 새로 만들지 않는다. New Bootstrap realcorpus parent는
기존 public TypeScript5.9.3 준비 fixture의 실제 source/integrity/static/dynamic/Evidence/Policy/
staged record/freshness assertions를 유지하고 새 application workflow의 정상0/nonzero1 두
requests를 실행한다. 기존 manual preparation/materialization parent와 controlled12 원본은
바꾸지 않는다. New actual은 별도 parent이며 source/binary/Host/고정42/ZIP/Docker/cleanup을 연결한다.

## 두 번째 검토 — source 전

Application이 기존 Sandbox.NodeExecutionOutcome 구체 타입에 의존하는 대안은 Architecture001에
맞지 않는다. Neutral interface를 현재 필요한 경계에만 두고 Bootstrap이 concrete result를
반환한다. Domain SandboxRequest/Result와 inspection 완료 status를 user program 결과로
재사용하지 않는다. Preparation.Valid만으로 실행 완료하거나 nonzero를 runtime fault로 합치지
않는다. Runtime complete는 backend의 whole guest disposal 범위이며 외부 서비스 전체 close는
기존 composition lifecycle이 소유한다. Public CLI/최종 close presentation은 후속 acceptance다.

Canonical format→quick/docs/security와 전체 영향 Application/Bootstrap Race, tagged vet/
pinned Staticcheck/build 및 실제 두 public requests를 검증한다. CPP/Host capture/공통 observer는
그대로이므로 CPP rebuild·기존12 actual 재실행은 하지 않는다. 실패/새 delta가 있으면 해당
원본을 보존하고 검증 범위를 재판정한다. Final/wholeP5/P6/P7은 계속 OPEN이다.

## Named local 결과

[Receipt](./p5c4-execution-workflow-local-20261029.json)는 canonical/tagged9checks·whole affected Race와 실제 public2
workflow 58.861s를 연결한다. 정상0/비정상1·원래 typed cause와 fresh current 검사/Policy/
records/context가 확인됐다. 기존1332/CPP/old actual parents/권한·예산은 그대로이며
Host/source/42/ZIP/Docker 보존·helperwait0/resources/endpoints0/noSKIP다. Application API/factory
범위만 네 상태 YES이고 public CLI/renderer/new hosted/final/wholeP5는 OPEN이다.
