# P6-B1 — current approved Python readonly input validator

Baseline dedb92adb10f30913015a1d92b06b4ddf3557425/treeea9437af/3021tracked·1351runtime.
P6-B owner audit와 current A3 context/console DATA를 재사용한다. CI430/source3c85는 pending;
timer01/session1333 due04:35:37UTC를 먼저 소비한다. M13-004 하나만 active다.
Read: P6-B baseline의 canonical routes/실제 owners, INDEX 관련 상세 lessons, 기존 source
introduce의 consumed stream SHA, closure install/census/anchor, start/terminate/cleanup과
zero-unit CPU constructor/finalize, A3 actual fixture와 opaque console contract를 대조했다.

| 기존/새 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| M5 wheel install/stream/full census/RO anchor/lifecycle/CPU | YES | YES (inspection) | YES (named) | YES (named) | 기존 본문을 private helper로 공유 |
| A3 real current whole context/A2 opaque console source | YES | YES | YES (actual) | YES (named) | same acquisition/metadata/runtime와 readonly inputs 결합 |
| approved execution readonly input validator | NO | NO | NO | NO | exact source→same installed tree/sole RO anchor→accounting/trace/cleanup |
| Python user-run/UVx golden/runtime/CLI | NO | NO | NO | NO | B1의 DATA 검증과 별도 후속 |

## Exact 수정 지도 — source 전

- `internal/sandbox/pypi_trusted_transaction.go:executeObservationTransaction`의 기존 wheel
  introduction/pip install/full census와 anchor overlap→writer termination→seal 본문을
  `pypi_closure_preparation.go`의 private methods로 옮겨 원래 caller가 같은 순서로 호출한다.
  M5 limitation codes/phase diagnostics/원래 cause·Pipx branch·required unit loop·최종
  SandboxResult/coverage는 유지한다. 함수의 기능을 대체/재구현하지 않고 같은 구현을 공유한다.
- 새 `internal/sandbox/python_execution_inputs.go`의 ValidatePythonExecutionInputs는 current
  Domain.ExecutionContext와 A2 ConsoleExecutionEntry를 함께 받는다. Full acquisition handle,
  command/metadata path/SHA/size/pinned runtime과 original public wheel 전체 승인 closure를
  확인하고 existing manifest/consumed stream/census/RO anchor/CPU/observer/cleanup을 호출한다.
  Metadata는 callable body가 아닌 선언 출처다. Native tool/cache/Host path를 재조회하지 않는다.
- Existing observation resource policy/default public wheel bounds를 유지한다. 진짜 resource
  lease/watch/zero-observation-unit accounting을 사용하며 dummy unit이나 SandboxResult를
  생성하지 않는다. 반환 nil은 이 DATA 입력 수명 범위뿐이다. Selected program/import/callable,
  kernel terminal/output·inspection completion/Policy/Host install 권한을 의미하지 않는다.
- `python_execution_inputs_test.go`는 unavailable/missing/current-caller cancellation 및 cause
  보존을 검사한다. Positive approval은 unit에서 조작하지 않는다.
- Existing `toolApprovedIntegrationFixture`는 기존 signature wrapper를 유지하고 내부 helper가
  실제 backend를 추가 반환하도록 한정한다. 새
  `internal/bootstrap/python_execution_inputs_integration_test.go`는 실제 A3 current pipeline의
  두 fresh context/opaque entries로 readonly DATA 검증과 다른 acquisition·intake tamper 거부를
  확인한다. Fake set/bundle/target/result는 없다.

새 files4(shared helper/input API/unit/actual), modified2(M5 caller/기존 real fixture), runtime1355;
기존1349 sources와 CPP131/observer/Policy/Model C/입출력·resource 한도는 그대로다. Public API
외 helper는 current DATA validator와 기존 M5가 즉시 소비하며 미래 package/config/interface/job은 없다.

## 두 번째 검토 — source 전

원본 wheel을 actual Docker input stream에서 재인증한 뒤 offline pip가 소비하고, 설치 전체
census가 original RECORD·metadata/member와 일치해야 한다. 기존 generated output은 별도
observed digest/ownership이며 callable success 또는 artifact original digest로 바꾸지 않는다.
RO reader를 먼저 연결한 뒤 writer를 제거한다. Foregin RW/추가 attachment·알 수 없는 query는
오류이고 source/trace/CPU/teardown 불완전·caller 취소는 원래 cause와 함께 실패한다.

Zero observation units는 실제 코드가 허용하는 preparation CPU ledger의 범위다. 기존 M5
inspection의 required ledger나 Model C semantics를 줄이지 않으며 새 API는 SandboxResult를
반환하지 않는다. CPU poll/transition/stop reserve/final counters와 trace/cleanup을 그대로 요구한다.
공통 코드 변경은 canonical format/Quick/docs/security, sandbox·PyPI inspection 전체 Race,
tagged vet/pinned Staticcheck/build와 실제 current wheel/readonly DATA·기존 pipx 소비자를 검증한다.
CPP/user-run 프로필은 이 단계에서 추가하지 않는다. 새 source는 CI430에 없고 active CI를 보존한다.

## 첫 실제 시도의 반증과 한정 수정 — 2026-10-29

Canonical10 PASS 뒤 actual-v1은36.88s의 첫 current readonly 소비자에서 context canceled로
실패했고 pipx parent는 미도달이다. Raw result/log/source/producer/Host·42/ZIP/Docker 불변과
helperwait0/전체 정리를 먼저 보존했다. 기존 observationCPUWatch.stop은 정상 종료에서도
자기 run context를 cancel한다. 새 B1 defer가 cleanup 이후 run.Err/Cause를 읽어 이 정상
내부 종료를 실패로 합친 것이 직접 확인된 새 소비자의 결함이다. 기존 watch/cleanup·M5
규칙을 바꾸지 않고 cleanup 전 run 오류를 보존한다. Cleanup이 기다린 watch.firstErr와
late caller cause는 계속 join하고 ledger.finalize가 cleanup 이후 wall/CPU/trace/정리를
검증한다. 실제 성공을 얻기 위해 취소/회계 guard를 제거하지 않는다. 같은4case actual과
기존 pipx parent, 새 current checks/build로 수정 범위만 재검증한다. 아직 해결 검증 전이다.

### v2 local 결과와 기존 pipx 비교

한정 수정의 canonical10/Quick92.105s/영향 전체 Race36.751s 뒤 같은 실제 Python4case는
73.90s PASS다. 첫 normal-stop 오인은 이 DATA 범위에서 교정됐고 원래 cause/guard를 유지했다.
동일 operator의 existing acquired pipx required consumer는43.25s/CPU authorization exhausted
(첫 stage=unit-termination39.087s, limit30s)로 실패했다. Original Host/source/CPP/42/ZIP/Docker
보존·helperwait0/모든 자원 정리는 PASS다. Shared seal 변경 또는 첫 cancel 교정의 인과로
단정하지 않는다. Hosted CI429의 같은 parent는35.10s PASS였지만 별도 환경이다.

수정 전 A3 binary61d56031/source1351, 같은 CPP131/b76b·같은 ZIP0d02/Host profile와30s
한도의 pipx-only differential baseline을 수행한다. 새 Python4case는 반복하지 않으며 이미
frozen v2 source/check/result/log와 분리한다. 같은 양상을 보이면 baseline runtime budget
한계로 구분하고 code/budget/required coverage를 임의 완화하지 않는다. B1 전체 local
qualification은 기존 consumer가 닫히기 전 OPEN이다.

## B1 named local 검증 완료

[Receipt](./p6b1-python-readonly-input-local-20261029.json)는 current1355/기존1349 불변·modified2/new4와
canonical10/Quick92.105s/영향 전체 Race36.751s·tagged vet/pinned Staticcheck/두 build를 연결한다.
같은 source/binaryc380에서 Python readonly4 PASS73.90s와 pipx-only PASS39.40s(outer41.25s)를
각각 확인했다. 두 actual source bindings/binary hashes는 동일하고 전후 Host/source/CPP/42/ZIP/
Docker 불변·helperwait0/전체 workspace/endpoints/cgroup 정리도 확인했다. Current pipx read-only
counter355samples의 peak11,545,051us는 진단이며 observer/accounting 완료 authority를 대체하지 않는다.

v1 cancel 결함은 해당 DATA4에서 해결 검증됐다. v2 encompassing operator는 후속 pipx CPU
authorization 소진으로 여전히 FAIL이며 separate named 소비자 통과를 whole attempt PASS로
바꾸지 않는다. Pre-B1 parent40.48s PASS와 current counter parent39.40s PASS는 최초 CPU 소진의
원인 해결이 아니다. 한도/required coverage/CPP131은 유지하고 원인을 미확정으로 남긴다.
B1 named local DATA scope의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES이며
current user program/ARTIFACT runtime·native UVx golden·CLI/new hosted/whole final은 미완료다.
CI430/source3c85는 B1을 포함하지 않고 timer03/session81287 due05:16:05UTC를 유지한다.
