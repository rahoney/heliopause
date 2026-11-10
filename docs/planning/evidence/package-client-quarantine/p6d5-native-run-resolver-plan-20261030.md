# P6-D5 — production native UV isolated-run resolver baseline

Baseline3dff66eec849505d7459788672999319cc6337ff/treed5e8fc3d/3098tracked·runtime1379다.
M13-004 하나만 active다. CI435는 remote b0f4cbc/treeeebd72a1/3097·1379로 frozen이며 actual
20분 GET worker13791/PID3398839가18:11:42UTC부터 조회한다. 실행 중 새 push하지 않는다.

Read: PROJECT/Queue/docs routes, lessons INDEX·artifact L01/L02·sandbox L03/L08/L12/L18
(기존 tmpfs 실패 포함)·evidence L16·ci L05/L06, Architecture001/client09§3/plan20§5·§6,
B2/B3/B4 receipts와 실제 fixed UV script/wheel producer·census84, UVResolver.resolveNativeSelection/
materializeUVSelection/withUVNativeSessionForToolInstall, reader.withUVArchiveInputs,
freezePipxNativeInput/manifestID·UVLock startup whole BindPipxQueryWheels, current test-only
uvPublicRawMirror와 UV runtime fixture, pipxNativeSession dataKeeper/removeKeeper·wheel volume,
SharedObserver.startProfileWithBudget/expected topology, readUVNativeTar/scanner/source-console
binder, current Python preparation/runtime adapter와 generic ExecutionPreparationService다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| native UV proposal/independent archive·whole startup freeze | YES | YES (public actual) | YES (D4A/B3 local) | YES (named) | production run consumer |
| UV runtime/output/owned-container·trace/cleanup lifecycle | YES | YES (existing resolver) | YES (기존/CI434 Linux) | YES (old named) | 같은 owner의 frozen offline input branch |
| readonly raw-wheel mirror/anchor/writer removal/seal | YES | YES (tagged current actual) | YES (D4A/B3 local) | YES (named) | 같은 body를 production DATA owner에 공유 |
| complete native census/controls/payload/relocatable console binder | YES | YES (current actual) | YES (B3 local) | YES (named) | production resolution fingerprint에 연결 |
| current approved Python preparation/runtime/app | YES | YES (uv-tool-install selector) | YES (CI433 named) | YES (old identity only) | 후속 별도 native-run adapter; 이름 변경으로 대체 금지 |
| native-run production resolver/selection | NO | NO | NO | NO | 아래 exact map + 실제 production consumer |
| aliases/CLI/generic equivalence/final | NO | NO | NO | NO | 후속 acceptance |

## Source 전 exact map과 두 번째 검토

기존 구현을 재작성/대체하지 않는다. UVResolver의 현재 session owner에 즉시 쓰이는 선택적
frozen input 한 개를 추가한다. 기존 pip/project/tool 호출은 같은 body·기본 인자로 그대로
동작한다. Frozen input은 tool-install mode와 조합하지 않고 valid whole bytes를 probe 이전에
검증한다. Offline branch는 DNS/policy/network 생성을 하지 않고 network=none·RO raw volume을
사용한다. Existing defer의 owned container→trace→output/runtime/mirror 정리와 원래 오류를
유지하고 readback/cleanup/cancel 오류면 결과를 비운다. New Core/daemon/protocol/config/job는 없다.

- internal/sandbox/uv_resolver.go: 기존 withUVNativeSessionForToolInstall의 optional frozen
  input branch만 추가한다. 기존 online flow는 보존하며 offline network/RO mount/trace/readback을
  같은 owner에 연결한다. UV query의 fixed profile/roles/CPP/권한·한도는 바꾸지 않는다.
- internal/sandbox/uv_run_native_input.go (new): tagged uvPublicRawMirror의 같은 body를 옮겨
  production caller와 현재 actual이 공유한다. 이름은 DATA mirror이지 pipx runtime가 아니다.
  현재 actual의 exact mount arguments/topology를 작은 즉시 소비 helper로 공유한다.
- internal/sandbox/uv_run_resolver.go (new): ResolveToolRun(ctx, canonical public requirement,
  explicit command)은 fresh native proposal→independent whole original/startup freeze→canonical
  original console 선택→offline native preparation/census→source-console binder를 소비한다.
  Native는 고정 /dev/null EACCES까지만 도달하고 실제 선택 script를 실행하지 않는다. Complete
  original과 native selection을 결합한 별도 uv-tool-run-isolated identity/opaque selection을 반환하며
  Policy/Host ownership/installed ToolReceipt를 만들지 않는다. Request/lock/original/console/script
  fingerprint를 domain resolution에 결합한다. 현재 graph/4MiB census·64KiB controls·single source와
  canonical selected console subset만 지원하고 unsupported input은 거부한다.
- internal/sandbox/uv_tool_run_public_real_test.go: mirror type/method만 production으로 공유하며
  기존 actual source gate/whole original/helper cleanup assertion은 유지한다.
- internal/sandbox/uv_runtime_real_test.go: 같은 arguments/topology helper를 공유한다. 기존 startup/
  query/lock/old consumers를 완화하지 않는다.
- internal/sandbox/uv_tool_run_census_real_test.go: fixed cache constant를 production과 공유한다.
- internal/sandbox/uv_run_native_input_test.go, uv_run_resolver_test.go (new): source-shaped fixture로
  exact readonly/offline/invalid input·request/command/source/control mismatch/identity/fingerprint/
  mutable originals·원래 cancellation/cleanup failure의 결과 부재를 확인한다. Existing binder의
  negative corpus를 재구현하지 않고 현재 owner를 소비한다.
- internal/sandbox/uv_run_resolver_real_test.go (new): actual public cowsay6.1를 새 production method로
  직접 호출해 fresh native-run identity/complete original/console binding·전체 정리를 검증한다.

총 modified4/new5·runtime1384 예상이다. Canonical format/Quick·영향 sandbox Race와 tagged
vet/pinned Staticcheck/build/docs/security를 완성 후보에 모은 뒤 current native resolver actual1을
진행한다. CPP132/UV195/기존 fixed inputs는 exact 재사용하고 같은 CPP rebuild/무관한 전체actual을
반복하지 않는다. 새 CI push는 하지 않는다. 다음 native-run factory가 기존 current whole
Evidence/Policy/runtime를 실제 소비하는 별도 범위를 구현하며 alias/CLI/final은 아직 미완료다.

## 첫 싼 검사 뒤 exact map 보완 / 기존 parser 의미 유지

첫 compile은 옮긴 mirror의 domain import 누락으로 멈췄고, 뒤 focused는 새 validator가
UpgradeRequirement.Reference의 intentional bare-project 의미를 exact constraint로 오해함을
확인했다. 기존 parser를 바꾸지 않고 requirement text 자체를 재parse한다. 다음 focused는
Domain generic single-name과 PyPI console 문법의 차이를 확인했다. Shell 문자는 native argv로
전달되지 않지만 잘못된 command를 네트워크 이전에 거부하려면 기존 PyPI owner를 소비해야 한다.
원본 소스/각 최초 로그를 먼저 보존했고 아직 Quick/tagged/native actual은 실행하지 않았다.

추가 수정 한 곳은 internal/artifact/pypi/execution_entry.go다. 기존 SelectConsoleExecutionEntry의
validToolCommand+domain.ValidateExecutionCommand 조건을 즉시 쓰이는 exported
ValidateConsoleExecutionCommand로 공유하고 기존 selector도 같은 조건을 호출한다. 기존
검증 의미/accepted grammar는 바꾸지 않으며 별도 lexer를 만들지 않는다. 새 native request가
이를 intake 이전에 호출한다. Modified5/new5/runtime1384이며 영향 Race는 PyPI/Sandbox/Bootstrap
전체로 모은다. Actual은 같은 production method의 정상과 real trace EOF 뒤 한정 incomplete
주입 거부를 하나의 검증 묶음으로 확인한다. Test-only recorder는 실제 observer를 그대로
호출하고 trace budget을 전달하며 approval/완료 원장을 대체하지 않는다.

## V4 전체 검사 뒤 수동 검토의 exact-conjunction 반례

V4 canonical9/Quick269.803s·PyPI/Sandbox/Bootstrap Race38.582s가 PASS였지만 사람 검토에서
== prefix로 exact를 추정하면 probe==1.0,>=1.0의 유효한 conjunction도 허용됨을 발견했다.
같은 production predicate의 새 test RED8.088s가 이를 직접 확인했다. V4를 최종 qualification으로
쓰지 않고 원본 소스/로그/binary identity를 보존했다. Upgrade parser를 변경하지 않고 동일
canonical ParseReference의 bare/exact grammar로 roundtrip해 해당 초기 subset을 거부한다.
변경한 후보의 현재 canonical/영향 Race/tagged build를 모아 확인한 뒤에만 actual을 시작한다.
CI435 sourceb0는 이 새 resolver 밖이며 새 push/CI 재실행은 하지 않는다.

## D5 named production resolver local 완료

[Qualification](./p6d5-native-run-resolver-local-20261030.json)은 current1384/기존1374 불변과 최종 canonical9·Quick89.765s/세 영향 package 전체 Race32.605s, 실제 production method의 정상·incomplete 거부49.68s/operator51.797s를 결합한다. 실제 source/native traces6개가 raw complete이고 offline 두 개는 각각16events/1842B·Outside/Network/Honeytoken/UnexpectedExec0다. 정상만 typed selection을 반환하고 한정 EOF incompleteness 주입은 entry/graph를 비운다. Host/source/CPP/42/ZIP/Docker 동일·helperwait0/전체 정리를 확인했고 완료 사본2개31356900B는 root103proc/ref0/errors0 뒤 정리했다. 누적243ELF+6cache/약10.76GiB Linux 논리 용량이며 모든 원본과 active CI435 timer는 보존했다. Native RECORD는 unapproved DATA이고 current whole Policy/program/adapter/aliases/CLI/final은 아직 별도다. 다음 D6는 기존 Python 실행 factory/runtime를 새 native-run selector와 연결하고 실제 current approval/normal0/nonzero2/불일치 거부를 확인한다. CI435/sourceb0는 새 D5 밖으로 frozen 상태다.
