# P6-A1 — actual wheel parser를 재사용한 console entry DATA 선택

Baseline3f961985a9025301b4a37591a4fd08f90eaab541/tree04eeaccb/3002tracked·1344runtime.
M13-004 하나만 active다. CI429/source9eb는 poll02 02:37:27UTC 8SUCCESS/Linux 진행이며
timer03/session39633 due02:57:27UTC(한국11:57:27)를 먼저 소비하고 새 push하지 않는다.
Read: P6 baseline/source discovery·Queue/Decisions/docs routes, INDEX routed lessons,
client09§3/plan20§4..6, WheelInspection/InspectWheelForSource/parseEntryPointsTxt/RECORD,
EntryPoint/Model C, current static inspector/UV project records binding/tool command validator,
generic ExecutionContext/Application/Manifest와 기존 P5 opaque selection을 대조했다.

| 실제 owner / scope | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| canonical wheel ZIP/hash/RECORD/metadata/entry-point parser | YES | YES | YES | YES | 재작성하지 않고 same reader/limits로 호출 |
| canonical EntryPoint details and metadata source | YES | YES | YES | YES | console-only exact selection·private acquisition/whole inventory binding |
| generic context/inspection/workflow/records/output (P5) | YES | YES | YES | YES | 이번 단계 수정 없음 |
| opaque console execution inventory/selected entry DATA API | NO | NO | NO | NO | measured acquired wheel→opaque inventory→exact console selection/matches |
| P6 native selector equivalence/Python runtime/CLI | NO | NO | NO | NO | 별도 후속; 이 API 성공으로 실행/승인/지원 선언하지 않음 |

## 수정 지도 — source 전

현재 artifact/pypi의 새 execution_entry.go와 의미 있는 실제 ZIP 기반 unit tests만 추가한다.
공개 InspectConsoleExecutionWheel은 이미 acquired된 public PyPI wheel과 caller의 ReaderAt,
filename/target/현재 WheelLimits를 받아 기존 InspectWheelForSource를 호출한다. Declared SHA256/
acquired digest·실제 observed bytes·정확한 project/version/size를 대조한다. Derived wheel,
다른 source/variant는 초기 scope에서 거부한다. Artifact나 approval을 이 API가 생성하지 않는다.

생성한 opaque inventory는 actual parser의 normalized 전체 inspection fingerprint와 acquired
identity/handle, canonical wheelDistInfo 경로의 RECORD-bound entry_points.txt member, private
entry details를 가진다. 외부에서 WheelInspection을 만들어 승인된 inventory로 주입하는 API는
만들지 않는다. EntryPointDetails는 실제 entry_points.txt에서만 생기는 기존 parser 의미를
사용한다. Parser·Model C·RECORD·coverage를 수정하거나 META header를 callable proof로 바꾸지 않는다.

선택은 console_scripts만 대상으로 하고 explicit command가 없으면 단일 console이며 normalized
project 이름과 같은 경우만 허용한다. 다른 이름/여러 bin은 명시 선택을 요구한다. 기존
validToolCommand와 Domain command214bytes 규칙을 재사용하고 module attr가 없는 target·extras는
초기 실행 capability 밖으로 거부한다. GUI/plugin/일반 script를 console로 바꾸지 않는다.
선택 값은 exact module/attr·metadata member의 digest/size와 원래 opaque inventory를 유지한다.
Matches는 전체 fingerprint/acquired handle·metadata/entry를 비교하며 getter는 slice를 복사한다.

이 값은 구조·측정 DATA다. Interpreter binding, module/callable의 안전성/실제 실행, current
Policy ALLOW, runtime admission, Host install/파일 반출의 권한이 아니다. No fake VerifiedSet/
Bundle/InstallTarget/SandboxResult, no interpreter/static import execution, no network/cache/PATH lookup.

## 두 번째 검토 — source 전

WheelInspection은 외부에서 만들 수 있는 struct이므로 그 값을 public constructor 입력으로
받아 opaque proof로 승격하지 않는다. 실제 canonical reader 함수의 결과만 내부에 보관한다.
전체 snapshot fingerprint와 원래 acquisition을 결합해 같은 package/command의 다른 source,
RECORD member·metadata target·handle substitution을 거부한다. Fingerprint 자체를 승인 token으로
사용하지 않고 후속 actual handoff가 원본 bytes를 다시 확인해야 한다.

Metadata member는 실행 모듈의 body가 아니라 정확한 console declaration의 출처다. 후속
Python composition에서 그 의미와 generated closed caller를 명시하며 기존 Node entry나
Manifest semantics를 몰래 대체하지 않는다. Native UV install/run equivalence는 여전히 UNPROVEN이다.

Canonical format→quick/docs/security와 전체 PyPI package Race, actual canonical ZIP/RECORD 기반
정상·explicit/multi/alias/missing/wrong-group/no attr/extras/unsafe name, digest/source/identity/
handle/full-member substitution, zero value·getter isolation을 검증한다. 기존 parser 자체를
고치지 않으며 all existing1344 source 불변을 확인한다. Python/Node runtime·CPP 실제 재실행은
없다. Runtime/API 조립은 후속이며 이 내부 DATA 단위만의 local qualification을 기록한다.

## Named local 결과

[Receipt](./p6a1-console-entry-local-20261029.json)는 format1.618s/focused3.020s/Quick309.386s/whole PyPI Race11.740s/
docs0.315s/security17.400s와 source1346/new2·기존1344 불변을 연결한다. 기존 parser를
재사용한 실제 ZIP/RECORD 기반 DATA 선택·치환 거부가 PASS다. Python preparation/runtime/
CLI WIRED는 NO이며 프로그램·검사 승인·native selector equivalence를 입증한 것이 아니다.
다음은 실제 controlled intake reader/entry binding이고 CI429/source9eb는 보존한다.
