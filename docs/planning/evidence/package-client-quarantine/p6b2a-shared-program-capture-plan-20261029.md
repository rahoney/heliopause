# P6-B2-A — 기존 captured program 실행 구현의 공유 감사

Runtime baseline0d96e23e665da91ab1954b27441c6547fb3f0186/tree20b74161/3032tracked·1355inputs.
단일 active work item은 M13-004다. CI431/source0d96은 pending이며 source를 새 push로 취소하지
않는다. Completed8의 checkoutf4d753ff/whole tree20b와 original444981B/CPP131을 확인했다.
공개 GET 자동 monitor/session61457은06:47:07UTC부터 실제 조회·다음20min 예약을 직접 기록한다.
이 source 단계는 CI431에 없으며 named local과 실제 hosted/final을 구분한다.

Read: Queue/PROJECT/docs routes, INDEX와 artifact L01/L02·sandbox L03/L08/L12/L18·
evidence L16·ci L05/L06, client09§3/plan20§5·Architecture005/Contract002·M5 계약과
B1 readonly actual/cause·P5 Node current boundary/CLI receipts, 실제 Node/direct admission/
Host capture·Python lifecycle/zero-unit ledger·SharedObserver/CPP owners를 대조했다.

| 기존 owner / 범위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| Domain.ExecutionContext/whole current Policy/source/argv/runtime | YES | YES | YES (named) | YES (named) | 그대로 재사용 |
| B1 whole readonly Python source/census/RO anchor·CPU/trace/cleanup | YES | YES (DATA) | YES (named local) | YES (named local) | actual selected program 소비자 |
| Node ExecuteNodeExecution의 captured admission/target/terminal/input-verify/완료 | YES | YES (Node) | YES (CI429/named) | YES (named) | 같은 본문을 private 공유 helper로 이동 |
| Node client exit reconciliation/typed error/출력 privacy | YES | YES | YES (named) | YES (named) | 동일 mapper를 family error factory와 재사용 |
| 기존 directExecOrigin/lease/nonce/arm/outcome/target/complete/abort | YES | YES | YES (named/CPP131) | YES (named) | 재구현 없이 호출 |
| Host CapturedProcessOutput/각 stream1MiB·WaitDelay/cancellation | YES | YES | YES (named) | YES (named) | 기존 interface를 같은 backend 내부에서 소비 |
| 현재 공용 captured program 구현 | NO | NO | NO | NO | 기존 Node 본문 공유 및 기존 소비자 검증 |
| Python ARTIFACT command/profile/current runtime/CLI | NO | NO | NO | NO | B2 후속 전용 source/RO profile·실제 정상/비정상/경계; alias는 별도 |

## 공유가 필요한 이유와 기존 구현의 재사용

공개 Node API는 NodeExecutionLayout/Node Backend에 결합돼 Python whole closure/CPU 원장과
직접 호환되지 않는다. 가짜 Node layout/context 또는 fake SandboxResult로 호출하지 않는다.
새 실행 workflow를 복제하거나 기존 기능을 대체하지 않고 같은 본문을 private helper로 옮겨
원래 Node caller가 같은 구현을 호출한다. Public Node API/outcome fields/getters/privacy·typed
NodeProgramError·기존 input lifecycle은 유지한다. 기존 구현 수정·본문 공유로 해결 가능하며
대체 구현/새 Core Port·package·runtime/CI scaffold는 필요 없다.

## Exact 수정 지도 — source 전

- 새 internal/sandbox/program_execution.go: 기존 nodeCapturedRunner의 동일한 structural
  capture capability와 private result carrier를 둔다. ExecuteNodeExecution의 실제
  active session→lifecycle lease→nonce/arm→기존 boundary exec→bounded capture→독립
  kernel terminal/target→client reconciliation→source verify→complete→lease release 본문을
  private captureAdmittedProgram으로 이동한다. Caller의 고정 ARTIFACT mode/command와
  기존 source verify callback·family error mapper만 받는다. Node의 command/argv는 그대로다.
- nodeExecutionClientResult의 기존 분기 본문은 generic private mapper로 이동하고 기존
  Node wrapper는 같은 incomplete error/typed error constructor로 호출한다. Normal0,
  실제 matching nonzero/signal와 Host mismatch/WaitDelay/overflow를 같은 방식으로 구분한다.
- internal/sandbox/node_execution.go: 원래 preflight/input prepare/context 및 inputs.close
  defers를 유지한다. 같은 wrapper의 underlying capture runner로 공유 본문을 호출하고 원래
  readonly tree→attachments 검증을 complete 이전 callback에서 수행한다. 결과를 기존
  private outcome fields에 복사한다. Named result의 cleanup/caller error downgrade는 유지한다.
  os/exec/time 등 이동한 본문의 imports만 새 owner로 이동한다.

Runtime1356/newGo1·modifiedGo1·기존1354 byte-identical을 목표로 한다. CPP131/프로필/호출
mode/관찰 손실 guard·재시도/권한/한도·Domain/Application/inspection/parser는 바꾸지 않는다.
기존 tests를 그대로 사용하며 새 helper를 Node가 즉시 소비하므로 미사용 scaffold가 아니다.
Python 전용 command/profile/backend·새 capability/interface·실제 지원 표시는 이 단계 밖이다.

## 두 번째 검토 — source 전

Target-loaded/terminal은 같은 consumed admission의 독립 kernel 사실이고 stdout/Host exit만으로
완료하지 않는다. Source verify는 complete 이전에 같은 lifecycle lease 안에서 수행한다.
Lease는 input/observer teardown 이전에 해제돼야 한다. Callback이 닫힌 observer를 잡거나
admission-aware wrapper로 재귀 호출하지 않는다. 원래 C5/C3d의 정상·nonzero·signal·취소/
독립 overflow·readonly/밖 접근/comm spoof/cleanup 범위를 보존하고 child body 미입증을 유지한다.

앞 B1 normal stop 취소와 late caller cause를 혼동하지 않는다. Python 후속에서 nonzero는
operational cleanup/CPU finalize와 분리해야 하며, zero observation units를 검사 완료나
사용 프로그램 성공으로 쓰지 않는다. 이번 공유 단계에서 그 후속 profile/권한을 만들지 않는다.

Canonical format/Quick·기존 Node focused·sandbox/Bootstrap 전체 Race·tagged vet/pinned
Staticcheck/bootstrap build를 수행한다. 같은 source/binary/기존 CPP131로 actual controlled
Node12와 current public CLI4 parent를 검증한다. 원본/Host·고정42/ZIP/Docker/helper·workspace/
endpoint 정리를 연결하고 완료 불필요한 copied binaries/data는 증거 보존·참조 확인 뒤 제거한다.
CI431 결과는 이전 source이므로 새 공유 구현의 hosted qualification에 전용하지 않는다.
