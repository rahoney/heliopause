# P6-H — UVx extras와 추가 입력의 source 전 감사

기준 source1ee3788/treea5eedf7a의 CI441은 고정·진행 중이다. Corpus SumDB setup 실패와
macOS default6m timeout은 별도 원본에 보존했고 아직 feature source를 수정하지 않았다.
이 항목은 같은 M13-004의 [목표 대조](./p7-command-acceptance-matrix-20261030.md)에서
확인한 UVx 목표 gap만 다룬다. 기존 구현을 다시 만드는 작업이 아니다.

Read: lessons INDEX·artifact L02/L19·sandbox L18·evidence L16·ci L06,
client09 §1/§3/현재 UV lane·plan20 §4–6, P6-D5/D6/D7의 source/현재 승인/CLI 증거,
아래 실제 owner와 native UV0.13.4의 고정 source. 설치/프로젝트의 지원을 자동 확대하지 않는다.

## 책임별 현재 상태

| 책임 / file·symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| CLI 별도 선택/literal argv — cli/python_execution.go AddUVRunExecution | YES | bare/exact YES | D7 named YES | 초기 lane YES | package root extras의 typed 요청·두 alias/help/거부; --with는 아래 별도 결정 |
| immutable public requirement — pypi/upgrade_input.go UpgradeRequirement | YES | 설치/기본 run YES | 기존 named YES | 해당 scope YES | 기존 parser를 넓히지 않고 UV run 전용 root extras를 별도 결합 |
| wheel 원본/metadata/marker — pypi/wheel.go WheelInspection, simple.go marker AST | YES | 기존 source/검사 YES | 기존 named YES | 기존 scope YES | authenticated Provides-Extra와 요청 extra 대조; 기존 bounded marker AST를 현재 UV 요청에만 소비. CUDA whitelist/기존 API 의미 유지 |
| native pylock/전체 archive binding — pypi/uv_lock.go, pipx_query_wheels.go | YES | 기본 UV/pipx YES | 기존 named YES | 기존 scope YES | 실제 extra 활성화에 따른 dependency edge/constraint만 복원; 새 root나 가짜 edge를 만들지 않음 |
| frozen whole/startup gate/native session — sandbox/uv_run_resolver.go, pipx_native_input.go | YES | 기본 UV/pipx YES | 기존 named YES | 기존 scope YES | full request를 source fingerprint/고정 offline --from에 결합; old bare/exact와 pipx body 재사용 |
| 현재 whole 검사/Policy/records/readonly Python — bootstrap/python_uv_run_execution.go, application execution owners | YES | YES | 기존 named YES | 기존 scope YES | root extras request를 concrete resolver에 결합하고 같은 whole pipeline으로 actual. 새 Core/engine/Policy/실행 권한 불필요 |
| root extras 전체 public/현재 앱·CLI | NO | NO | NO | NO | 아래 H1 |
| 독립적인 --with package 전체 | NO | NO | NO | NO | 아래 H2의 Domain 모델/범위 결정부터 필요 |

## H1 — primary root extras의 최소 구현 경계

Root extras는 승인할 같은 primary의 실제 Requires-Dist edge를 활성화한다. 기존
LockedDependencyGraph의 primary/연결성 계약을 유지할 수 있다. UpgradeRequirement와
CUDA boundedExtraSet은 기존 API/범위를 그대로 둔다. UV run에만 immutable한
base requirement+정규화 extras를 결합한다. 기존512-byte public requirement 한도와
resource policy·metadata/graph 한도를 재사용하고 local/VCS/URL/marker/runtime 요청은 거부한다.

Wheel parser가 이미 검증한 METADATA의 Provides-Extra를 DATA로 보존하고 해당 root의
요청만 확인한다. 같은 bounded marker AST로 활성 edge/constraint를 복원하며 unknown
요청/unsupported metadata·누락/추가 wheel·불만족 constraint·변조는 거부한다. Metadata의
declaration이나 native cache를 실행/승인 authority로 쓰지 않는다. 전이 dependency의 별도
extras나 arbitrary PEP508 전체 지원을 이 첫 연결로 주장하지 않는다.

기존 archive 검사·독립 Simple/SHA/RECORD·startup surface·frozen copy·native session·
current inspection/Policy/records·readonly runtime 본체를 재사용한다. New input일 때만
extra request를 source fingerprint와 offline --from에 결합하고 기존 bare/exact/pipx의
의미/identity를 보존한다. Native preparation은 계속 고정 /dev/null terminal만 요청하며
artifact console·사용자 argv를 CONTROL 단계에 전달하지 않는다.

첫 실제 후보는 기존 D7 공개 python-slugify8.0.4의 unidecode extra다. 실제 선언/metadata와
UV 선택을 먼저 대조하고 전체3-wheel 선택이 성립하면 새 ALLOW·console/CLI·정리를 검증한다.
이는 아직 실행한 결과가 아니다. Named negative·기존 default 보존 검사를 먼저 하고,
meaningful native/current/CLI 변경 묶음에서 필수 Go/Race/actual을 모은다. 기존 CPP 입력이
불변이면 반복하지 않으며 필요가 확인되기 전 observer grant를 추가하지 않는다.

## H2 — 독립 --with 입력이 기존 모델로 표현되지 않는 이유

domain/install.go의 NewLockedDependencyGraph는 primary가 정확히 하나이고 모든 node가
그 primary에서 실제 edge로 도달해야 한다. domain/execution_context.go NewExecutionContext와
application/execution_inspect.go InspectResolved도 그 동일 approved primary graph를 요구한다.
독립 --with는 primary의 실제 metadata dependency가 아닐 수 있다. 가짜 dependency edge를
추가하면 source 증명을 위조하며, no-primary ProjectDependencySnapshot에 가짜 InstallTarget을
주면 일회성 실행과 설치·소유권 계약을 혼동한다. 두 방법 모두 사용하지 않는다.

기존 graph/실행 context 하나의 단순 수정으로 해결할 수 없는 현재 acceptance gap이다.
추가 입력을 지원하려면 primary/명시 supplemental roots·실제 metadata edges·전체 artifact와
현재 whole approval을 갖는 별도 typed 실행 입력 계약이 필요하다. 기존 connected graph와
install/update 소비자의 invariant는 유지해야 한다. 중복 engine 대신 기존 per-artifact
검사/M3·M5 rule body·records와 readonly capture/정리를 공유하되, 새 입력의 전체 집합/
fingerprint를 검사·Manifest/SBOM·runtime binding 끝까지 전달하는 설계가 필요하다.

예상 영향은 Domain 실행 입력, Application whole inspection/preparation, UV selection,
Evidence Manifest/SBOM와 Python readonly materialization binding, CLI --with다. 기존
LockedDependencyGraph를 느슨하게 하거나 기존 실행 context/flow를 재작성하는 방식은
AGENTS의 Domain invariant 보존 규칙에 걸리므로 수행하지 않는다. 먼저 기존 invariant를
유지하는 additive typed 모델과 기존 body 재사용 경계를 확정한다. 원래 목표의 구현 권한
안에서 가능한 추가는 진행하되 실제 기존 구현 대체/재작성이나 invariant 변경이 필요하다고
확정될 때에만 해당 규칙의 근거·영향과 승인을 먼저 다룬다. H2 placeholder는 만들지 않으며
H1은 독립적으로 진행한다. 초기 범위 밖으로 미루는 것도 owner가 정하기 전 승인된
deferral로 기록하지 않는다.

## 재검토와 완료 기준

H1은 원본 선언→동일 native 선택→독립 whole graph→매번 새 승인/records→실제 console/
typed nonzero·privacy·cleanup과 기존 basic lane 보존까지 검증한 범위에서만 닫는다.
H2 결정과 미구현·Bun-only/일반 동등성·현재 source hosted/final profile은 별도 OPEN이다.
CI441 actual20분 GET과 최초 실패를 계속 소비하고 작은 helper/doc별 push/full CI는 하지 않는다.

## H1 첫 actual의 identity owner — 2026-10-30

v6의 기존 base case1은56.10s PASS, extras case6은10.17s preparation refusal/Policy·program
미도달이었다. V7의 CLI cause unwrap은 nil로 원인 수집에 실패했고, v8은 불필요한 실행
fixture의 필수 입력 부재로 native 미도달이다. 이 진단을 제품 실패나 해결로 쓰지 않는다.
기존 sandbox native owner의 test-only v9/v10은 source trace complete 뒤
`UVInstalledWheelControls`의 실제 `Unidecode-1.4.0.dist-info`/Name `Unidecode`와
normalized `unidecode`를 기존 lowercase-only `wheelDistInfo` 문자열과 비교해 거부함을 확인했다.
원본은 `/tmp/haa-p6h-root-extras-v6` 및 `...-native-diagnostic-v9/v10`이고 실제 Host5·Docker/
helperwait0·같은 inode profile 복원은 모두 PASS다. 임시 source는 원본 저장 후 제거한다.

[Wheel 공식 규격](https://packaging.python.org/en/latest/specifications/binary-distribution-format/#escaping-and-unicode)은 소비자가 과거의 대문자·점 이름도 받아야 한다고 명시한다.
수정 범위는 기존 installed-control identity reader, digest/RECORD wheel reader와 그 실제
console metadata path consumer다. 같은 normalized project·exact version인 단일 실제
metadata directory만 사용하고 literal paths/RECORD를 보존한다. 외부/다중 alias directory나
identity 혼합은 계속 거부한다. 설치 upgrade/pipx supply의 별도 계약을 확장하지 않는다.
새 wheel parser나 native 권한·새 graph를 만들지 않고 기존 세 owner의 비교/전달만 보완한다.
Legacy directory RED와 보안 negative 뒤 extras actual의 미완료 사례만 계속한다.

## H1 named local 결과

[H1 receipt](./p6h1-uvx-root-extras-local-20261030.json)은 구현/연결·named local qualification을
확인한다. Actual v13은 같은 제품22 sources의 base case1와 extras case6/7/8 모두229.44s
PASS다. Whole3-wheel의 매 요청 새 ALLOW/0·typed2와 unknown extra의 no-runtime/no-records를
검증했다. Root/private25 sources와 actual22 bytes가 일치하고 Quick374.16s·영향5package
Race76.824s·checker Race93.755s·docs/security가 PASS다. CPP133/observer09615d38은 같은
입력으로 재사용했다. 기존 cohort에8cases/11원본만 연결했고 단일full8 actual/새 hosted는
아직 검증하지 않았다. H2·일반 native 동등성·whole final은 별도 OPEN이다.
