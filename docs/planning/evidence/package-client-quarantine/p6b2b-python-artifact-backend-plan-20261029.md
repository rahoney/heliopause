# P6-B2-B — selected Python ARTIFACT backend 착수 감사

Baseline5c853a16d7c2fa75bf881223ced69a2f28b644e4/tree6b0c72f93d5758c3ca4101ab18395dba7f2c209c,
3036tracked/1356runtime. M13-004 하나만 active다. CI431/source0d96은 이 baseline의 공유 helper를
포함하지 않는 별도 pending hosted 범위다. Poll04는07:27:09.252690→07:27:10.790120UTC에
8SUCCESS/Linux running을 확인했고 다음07:47:10.790138UTC가 예약됐다. 새 push로 취소하지 않는다.

Read: PROJECT/docs/Queue, INDEX와 artifact L01/L02·sandbox L03/L08/L12/L18·evidence L16·ci
L05/L06, Architecture005·Contract002·M5§3/8·client09§3·plan20§5, B1 readonly DATA receipt와
B2-A 공유 구현 receipt, 아래 실제 source owners를 대조했다. 새 native 문서 해석/버전 변경은 없다.

| 현재 owner / 범위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| current whole Python preparation/Policy/context/opaque console source | YES | YES | YES (named actual) | YES (named) | 그대로 소비 |
| withPythonExecutionInputs 전체 source·manifest·install/census·RO anchor·CPU/trace/cleanup | YES | YES | YES (B1 named DATA) | YES (named DATA) | selected program callback |
| captureAdmittedProgram/private mapper/admission lifecycle | YES | YES (Node) | YES (B2-A local Node12/CLI4) | YES (named local) | Python family caller; hosted 별도 |
| 기존 immutable Python loader/runtime read·irreversible Python handoff·target/terminal | YES | YES (M5/Pipx) | YES (named prior) | YES (named prior) | exact selected-console argv와 별도 profile |
| 기존 whole startup-hook/customization gate | YES | YES (Pipx) | YES (named) | YES (named) | 같은 제한 본문을 새 isolated console consumer가 호출 |
| selected Python ARTIFACT profile/backend/typed outcome | NO | NO | NO | NO | 아래 bounded source map과 actual normal/nonzero |
| Python hostile boundary/native UVx golden/aliases/public CLI | NO | NO | NO | NO | B2 후속/full boundary 및 P6 후속; 이 단계로 완료 표시하지 않음 |

## 정확한 초기 실행 계약

첫 backend는 현재 승인된 public wheel 전체 closure와 정확한 console declaration을 받는다.
Python3.14.7의 고정 절대 interpreter를 -I -S -B -c와 고정된 controller로 실행한다. Module/attr/
command/private args는 각각 argv DATA이며 shell/eval/생성 shebang/PATH 탐색이 아니다. Controller는
stdlib 초기 sys.path/isolated/no-site/no-bytecode를 확인하고 readonly /haa-site 하나를 추가한다.
정확한 module을 import하고 attr chain을 호출하며 sys.argv[0]은 선택한 command다. Exit/출력은
프로그램 DATA이고 성공 authority가 아니다. 전체 startup hooks/customization은 기존 gate로
거부하여 -S가 native startup 의미를 조용히 바꾸지 않게 한다. Extras/UVx equivalence는 미지원이다.

새 python-execution profile은 기존 pypi-wheel inspection profile과 분리한다. Base readonly OCI
root/private tmpfs/helper/정확한 readonly /haa-site와 sole anchor를 같은 원장으로 검증한다.
Preparation·anchor는 기존 M5 profile을 유지하고 selected observation phase만 새 profile이다.
기존 Python handoff mode와 consumed admission/독립 target·terminal을 재사용한다. Node, M5,
Pipx의 profile·권한·한도·관찰 손실 guard/retry는 유지한다.

## Source 전 exact 수정 지도

- 새 internal/artifact/pypi/execution_startup.go: 기존 validatePipxMetadataSurface를 그대로 호출하는
  bounded startup restriction wrapper. 원본 canonical WheelInspection에만 소비하며 approval나
  parser authority를 만들지 않는다. 기존 Pipx function/body/callers/errors는 그대로다.
- 새 internal/sandbox/python_execution.go: PythonDynamicBackend.ExecutePythonExecution와 private
  outcome/typed PythonProgramError 및 fixed controller/argv. B1 withPythonExecutionInputs 안에서
  whole startup/entry member를 확인하고 새 readonly observation phase를 생성한다. Underlying
  programCapturedRunner와 captureAdmittedProgram을 호출해 같은 kernel reconciliation을 쓴다.
  Complete 전 callback은 reader full installed census와 anchor/volume attachments를 재확인한다.
  Program nonzero는 별도 변수에 보존하고 callback은 operational 결과만 반환하여 이후 CPU/trace/
  cleanup/ledger.finalize를 생략하지 않는다. Source/cleanup/caller fault는 outcome complete를 내린다.
- 새 python_execution_observer.go: 새 profile과 shared trace-budget의 현재 caller만 추가한다.
  Topology는 기존 pythonClosureExpectedTopology의 readonly observation 값을 재사용하고 새
  collector budget/permission을 만들지 않는다. Initial cwd는 고정 /다.
- pypi_trusted_transaction.go: private execution flag(default false)와 startPhase의 readonly
  observation 전용 분기만 추가한다. Preparation/anchor/Pipx/M5 default 생성·observer·register/
  AwaitMountAnchors·CPU transition/terminate/cleanup 본문은 유지한다. Trace ledger는 기존
  pypi-wheel의 preparation/anchor/reader 합산10k/2MiB이며 새 stream으로 보충하지 않는다.
- shared_observer.go: profile whitelist/topology에 위 readonly profile만 연결한다. 기존 Python
  inspection profile family와 observation/coverage 의미는 유지한다.
- tools/gvisor-observer/observer.cc: 새 profile과 고정 controller를 closed whitelist에 추가하고
  full target argv를 root helper→정확한 setpriv→고정 interpreter image 세 경계에서 동일하게
  검증한다. 기존 Node target vector 비교 본문만 private vector helper로 공유하며 Node wrapper는
  같은 비교를 소비한다. 새 family의 module/attr grammar/argc/private arg byte 한도는 제한적이며
  임의 -c/옵션·다른 root/CONTROL launch/clone/PATH/comm spoof로 target trust를 얻지 못한다.
  Readonly /haa-site topology와 target pending/loaded 상태에 결합된 immutable Python loader/stdlib
  ABI만 재사용한다. Pip/site-packages/ambient config는 user-run 권한으로 주지 않는다. Generic
  helper/comm 예외가 accepted target 이후 권한을 복원하지 않도록 새 profile을 Node와 같이 제한한다.
- observer_latch_test.cc: 실제 profile/admission/exec-event/readonly topology·target substitution/
  demoter/interpreter argv mismatch·wrong mode/role/comm/clone/readonly-loader scope negatives를
  기존 test harness에서 검증한다. 기존131 latches는 유지한다.
- 새 sandbox/python_execution_test.go: invalid/current request cancellation, argv vector 및 private
  outcome·default-zero, 새 readonly profile/기존 M5 topology 불변을 의미 있는 기존 harness로 확인한다.
- 새 bootstrap/python_execution_integration_test.go: existing real approved fixture/current preparation/
  canonical console reader로 cowsay6.1의 normal0·actual matching nonzero를 검증한다. Fake approved
  set/SandboxResult/InstallTarget은 만들지 않는다. 실제 boundary/alias/CLI qualification은 별도다.

## 두 번째 검토 — source 전

기존 기능은 재구현/교체하지 않고 기존 M5 input lifecycle·Node capture·Python image transition을
같은 current consumer가 재사용한다. Source verification은 complete 이전 lease 안에서 수행하고,
helper 반환/lease 해제 뒤 terminatePhase와 observer cleanup을 수행한다. Admission-aware wrapper의
exec를 같은 lease 안에서 재귀 호출하지 않는다. CPU watch의 정상 stop 취소와 genuine/late caller
취소를 혼동하지 않으며 typed program failure 때문에 operational finalize를 생략하지 않는다.

Fixed controller는 canonical source의 상수이며 artifact module/attr/argv 문자열을 program source로
합성하지 않는다. Full argv binding은 image load/ARTIFACT 역할과 별도이고 root terminal은 target
load와 별도다. Readonly mount나 Policy ALLOW를 실행 완료로 쓰지 않는다. New profile은 old
Python inspection/Pipx/native UV/Node CONTROL/network/clone 권한을 상속하지 않는다. 기존 startup
제한을 통과했다고 native UVx와 같거나 모든 Python 프로그램이 지원된다고 표시하지 않는다.

Canonical format/focused/Quick·sandbox/PyPI/Bootstrap 전체 Race, tagged vet/pinned Staticcheck/
bootstrap build와 C++ release build/기존131+새 latch를 수행한다. 같은 source/binaries의 실제
new public Python normal/nonzero와 기존 M5/Pipx·Node consumers를 영향 범위에 맞게 검증한다.
실패 원본/source를 먼저 고정하고 원인 미확정 동일 재실행을 하지 않는다. Host5files/42/ZIP/
Docker/source identity/helperwait0/observer endpoints/owned workspace 정리를 확인한다. 완료 후
원본/재사용 input은 보존하고 불필요한 copied binaries는 inode/hash/전체 proc 무참조 뒤 삭제한다.

새 backend의 actual 정상·비정상 범위만 닫는다. Python signals/cancellation/overflow/hostile boundary,
native golden/aliases/CLI, 새 grouped hosted와 final CPU·세CUDA/signed support는 OPEN이다.

## Source 작성 중 검토 보완 — 검사 시작 전

기존 terminatePhase는 raw suspicious observations를 Inspector에 전달하며, 일반 DATA cleanup
성공만으로 user-run 금지 행위 거부를 대신할 수 없다. 새 execution/observation 분기에만 기존
retainedSuspiciousTraceFacts predicate를 소비해 operational error를 보존한다. 기존 M5/Pipx
관찰 수집·정리 본문과 의미는 유지한다. verifyAnchorAndVolume은 기존 map[id]readonly API를
그대로 사용한다. 새 profile의 불필요한 demoting bit는 추가하지 않고 기존 pending/role을 쓴다.

## 첫 actual 뒤 ABI 한정 보완

Canonical9/CPP132의 v1 첫 current cowsay user-run40.25s는 OPEN_RESULT_CLASSIFICATION_PROC/
VM_OVERCOMMIT_MEMORY/system/flags32768에서 거부됐다. 원래 source/입력/Host와 cleanup은
보존했고 원본 actual·CPP/Go identity를 고정했다. 기존 pinned Python ABI owner의 정확한
/proc/sys/vm/overcommit_memory 읽기만 새 profile에 연결한다. System /proc mount, exact flags32768,
현재 Python image·ARTIFACT/direct root/start/active/pending guards와 negative 회귀를 유지한다.
일반 proc·Node/M5/Pipx 권한·한도는 넓히지 않는다. v2의 source delta는 CPP2뿐이므로 동일한 Go
build inputs/binary의 focused/Race/vet/Staticcheck/build 증거와 새 Quick/docs/security·CPP release
결과를 출처별로 결합한다. 같은 실패의 blind rerun이나 v1 PASS 재분류는 하지 않는다.

## 기존 Pipx CPU 한도 재발 — 제한된 비교 계획

V2는 Python normal0/nonzero2 actual71.28s와 기존 Node12 actual205.66s가 각각 PASS지만
별도 Pipx parent45.08s는 DIRECT_IMPORT CPU authorization exhausted였다. 최초 원본과
Host/고정 입력/source/helperwait0/전체 정리는 보존했고 whole candidate qualified로 표시하지
않는다. B1에서도 같은 한도 사건이 있었으며 과거 원인은 미확정이다. 정책/한도/코드를 바꾸지
않고 (1) 보존된 pre-B2 Go c380 + current CPP f69, (2) current Go f70 + current CPP f69를
각 한 번의 readonly Host cgroup CPU counter 비교로 관찰한다. 전자는 새 Go 변경이 실패의
필요조건인지, 후자는 실제 누적 CPU/uncertainty 경계에 대한 새 계측이다. 혼합 baseline은
진단일 뿐 current backend qualification으로 쓰지 않는다. 이후 PASS도 최초 CPU 인과 해결로
쓰지 않고, 반증/새 계측 근거 없이 같은 variant를 다시 실행하지 않는다.
