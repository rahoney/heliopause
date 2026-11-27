# P6-D4B3 — native source/console DATA 결합 baseline

Baseline114de4f2d8dd5d61d0e8496d74def5a2e86412b6/tree4a3b273e/3086tracked·runtime1372다.
단일 active item M13-004다. B2의 current query/export/census와 old public Python2는 named
local qualified이고 native selection/source/console binding/current adapter/aliases/CLI는 남는다.
CPP010cd8/source def84e/test ca7041·132named/UV195는 이번 Go 작업에서 그대로 재사용한다.
CI434는 이전 d81의 Linux104SUCCESS/macOS120s·RequiredFAIL로 끝났고 monitor 자연0/next none이다.
새 CI는 source/native proof와 필요한 hosted wiring의 의미 있는 묶음까지 시작하지 않는다.

Read: INDEX와 실제 Git/Queue/PROJECT/docsREADME routes, artifact L01/L02(새 B2 wrapper 관측),
sandbox L03/L08/L12/L18(기존 tmpfs corner case와 B2), evidence L16, ci L05/L06,
Architecture001와 client09§3·plan20§5/§6, B2 complete DATA84/whole source gate/actual/receipts,
canonical InspectWheel/InspectConsoleExecutionWheel/SelectConsoleExecutionEntry·InstalledFile mapping,
UVInstalledWheelControls/UVLock binding·ValidateUVNativeConsoleScript, promotion의 기존 native
footprint/record/script proof 책임과 current readonly closure installation census를 대조했다.

고정 UV7456b60b의 script.rs/blob2b167528(8489B), wheel.rs/blob7da775cf(58854B)를 Git blob와
UTF8 readback으로 확인했다. wheel.rs:128..158/367..390/975..995의 producer는 relocatable이면
scripts directory 기준 상대 interpreter와 realpath/dirname shell prefix를 사용한다. 기존 공통
Python launcher tail은 그대로다. 실제 native84의 cfg137B/relocatable=true·358B wrapper가 이에
해당한다. Source inventory와 전체 원본은 raw p6d4b3-uv-{script,wheel}-* 한 곳에 보존한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| canonical whole wheel/RECORD/source/startup freeze/RO mirror | YES | YES | YES (기존/D4A) | YES (named) | 재사용 |
| bounded native tar scanner/type·complete DATA84 | YES | YES (current tagged caller) | YES (B2 local) | YES (named) | production pure DATA owner에 같은 body 공유 |
| canonical opaque console inventory/selection | YES | YES (현재 Python app) | YES (기존/CI433) | YES (named) | native source/console consumer 연결 |
| nonrelocatable UV exact byte matcher | YES | YES (native ownership) | YES (기존) | YES (named) | 기존 의미 그대로 재사용 |
| native relocatable exact matcher | NO | NO | NO | NO | 아래 별도 DATA matcher/현재 consumer |
| whole native controls/source/console binding | NO | NO | NO | NO | 기존 frozen input/canonical reader/controls와 native census 결합 |
| native Host original/RECORD ownership proof | YES | YES (promotion) | YES (기존) | YES (named) | 이번 ephemeral DATA에 가짜 target/receipt로 호출하지 않음 |
| production UVx resolver/current adapter/aliases/CLI/final | NO | NO | NO | NO | 후속 acceptance; 현재 Python runtime/app owners 재구현 금지 |

## Exact map와 두 번째 검토 / 구현 전 범위

기존 구현을 대체·재작성하지 않는다. 현재 tagged uv_tool_run_census_real_test.go의
uvRunCensusFact와 scanUVRunCensus는 같은 package의 production pure DATA 파일로 본문을 그대로
공유한다. 기존 body를 production Go가 _test.go에서 참조할 수 없기 때문에 위치만 옮긴다.
기존 fixture/actual wrapper는 shared scanner를 계속 사용한다. 새 generic interface/package/
profile·I/O·future job는 없다. 현재 native binding consumer와 unit negatives가 즉시 사용한다.

pypi에 고정 relative interpreter name python만을 위한 relocatable DATA matcher를 추가한다.
Fixed source의 exact shell header를 확인한 뒤 private memory의 비교용 header projection으로
기존 ValidateUVNativeConsoleScript의 canonical entrypoint grammar/UTF8 tail matching을 재사용한다.
실제 file/원본을 재작성하거나 shell을 실행하지 않는다. 기존 nonrelocatable matcher를 완화하지
않는다. 이 비교는 name/filename·interpreter inode/source/auth/Policy 권한을 부여하지 않는다.
그 연결은 complete raw/source inventory와 native env/bin/python alias·cfg/선택 script caller가 맡는다.

Native census는 unapproved DATA이며 source는 기존 owned frozen bytes/lock/pages다. Canonical
whole wheel reader/installed mapping과 native METADATA/WHEEL의 exact size/digest·complete set를
대조하고 UVLock.SelectInstalledWheelControls를 재사용한다. Root console declaration/member와
실제 native bin script의 정규 경로/type/uid/gid/mode/size/hash/body, pinned cfg·Python aliases를
결합한다. Opaque console reader/selector의 책임을 JSON inspection 또는 caller bool로 대체하지 않는다.
다른 env/missing/duplicate controls/extra distribution/changed payload/unknown link/script/header/
module/attr/runtime·foreign source·stale frozen input을 거부한다. 설치 목적지·Host ownership을
만들거나 ToolReceipt/uv-tool-install identity를 uvx로 이름만 바꾸지 않는다.

Original wheel RECORD는 기존 canonical reader가 검사한다. Native installed RECORD와 installer
auxiliary metadata는 source bytes/ownership/Policy authority가 아니다. 기존 promotion의 held
Host target·filesystem record parser를 가짜 install target으로 호출하지 않는다. Native payload를
게시하거나 실행하지 않으며 실제 프로그램은 후속 current whole inspection/Policy와 기존
readonly backend가 original wheel에서 구성한다. Native generated metadata의 보존/검증 범위를
실제 원본을 보고 정하며 모든 byte/ownership을 인증했다고 확대하지 않는다. Whole source payload
비교와 관측된 native metadata의 구분은 binder/qualification에 명시한다.

현재 실제 public corpus는 cowsay6.1이며 원본 SHA274b1e6f·단일 env/console DATA다. 일반 요청이나
여러 dependency/옵션에 대한 native equivalence·runtime/CLI 지원을 이 한 사례로 닫지 않는다.
필요한 production resolver의 request/closure/mode 제한과 실제 추가 matrix는 후속 감사가 소유한다.

## 검증 / 아직 시작하지 않은 범위

고정 formatter/Quick·영향 PyPI 및 sandbox Race·tagged vet/pinned Staticcheck/compile와 docs/
security를 완성 후보에 모은다. Source-preserving normal·negative pure DATA tests와 같은 current
native census의 실제 binding을 확인한다. CPP/고정 inputs/기존 unchanged consumers는 exact 입력
증명으로 재사용하고 같은 5분 CPP build나 무관한 전체 actual을 반복하지 않는다. Native failed
copy/raw logs·완료 사본은 원본 보존/ref0/errors0 뒤 정리한다. 새 CI는 중요한 source/native 및
hosted wiring 묶음에 한정하고 시작할 때 actual20min GET을 다시 등록한다. 이 문서는 구현/검증
완료를 뜻하지 않는다.

## B3 named local 완료

[Qualification](./p6d4b3-native-source-console-local-20261030.json)은 current1379/기존1371 불변과 canonical8(Quick240.797s/영향 PyPI·Sandbox Race32.458s), 실제 공개 원본 binding24.07s/operator26.071s를 연결한다. Whole original/source·native complete controls/payload SHA와 exact relocatable console을 결합했고 complete16events/1842B·Outside/Network/Honeytoken/UnexpectedExec0, Host/source/42/ZIP/Docker 보존·helperwait0/전체 정리를 확인했다. Native RECORD/activation/cache metadata는 unapproved DATA이며 Policy/selected program NOT_RUN·generic native 동등성/CLI는 미완료다. 고정 CPP132/UV195를 재사용하고 재빌드하지 않았다. 첫 producer parse-before-write와 tagged 전용 상수의 default U1000는 원본 보존 후 각각 교정했다. 종료 사본2개31244251B만 제거해 누적241ELF+6cache/약10.73GiB Linux 논리 용량이다. 다음 B4는 existing hosted cohort에 두 current native public 부모와 exact guards를 연결한 뒤 중요한 묶음 CI를 한 번 진행한다.
