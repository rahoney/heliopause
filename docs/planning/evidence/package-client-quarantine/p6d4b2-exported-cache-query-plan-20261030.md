# P6-D4B2 — exportable private cache와 exact native query ABI

Baseline373d6b9d87b5b0f6df52941ef5b487f5dd533581/treed42b56ff/3083tracked·runtime1371이다.
M13-004 하나만 active다. Root와 현재 private source의 runtime1371은 byte-identical이다.
CI434는 source d81/whole tree4497의 Linux104/CPP132/fullCPU291.43s named SUCCESS지만
macOS production build120s/default 미도달·Required는 FAIL이다. 실제20분 GET6회 뒤 monitor
자연 exit0/다음 타이머 없음이다. 다음 중요한 구현 묶음 전 push/CI·같은 실패 입력 재시도는 없다.
D3-E2/현재 D4A는 그 source 밖의 named local 검증이며 whole qualification을 주장하지 않는다.

## Read / 현재 source의 확인

INDEX와 관련 artifact L01/L02, sandbox L03/L08/L12/L18(기존 P5-C3b tmpfs 전송까지),
evidence L16, ci L05/L06, PROJECT/docsREADME/Queue, client09§3·plan20§5/§6,
D3/E2·D4A·B1 canonical/raw, SharedObserver topology/UV fixture·existing private volume,
UVToolQueryEnvironment/IsExactUVPythonQueryProgram/IsExactUVPythonQuery·ClassifyFilesystemOpen,
현재 UV120 latch의 clone/start/image/role/network/reexec/정규화/readonly mutation을 읽었다.
고정 UV7456b60b의 run.rs:385..467/environment.rs:228..282/cache.rs:166..211/327..346/411..433와
interpreter.rs:1146..1190/query_cached·PythonEnvironment::from_root를 직접 대조했다.
후자의 과거 로컬 source 경로는 없어 동일 Git blob과 SHA256 원본을 다시 확인해 raw 한 곳에
복구했다. Source inventories p6d4b-uv-{cache,interpreter,python-environment}-*가 증거다.

Cache::from_path는 TempDir drop owner가 없고 CachedEnvironment는 archive-v0/ID16에 새
venv를 persist한다. InterpreterInfo::query는 cache.root() 아래의 새 .tmpXXXXXX에 고정 module을
쓴 뒤 pinned image의 -I/-B/-c exact launcher로 조회한다. PythonEnvironment::from_root도
같은 cache를 넘긴다. 따라서 cache만 /haa-site로 옮기고 helper launcher 경계가 그대로라고
가정하지 않는다. Native metadata query는 .pth/startup 실행 가능성을 갖기 때문에 D4A의
whole original/source/startup gate가 query 이전에 반드시 유지된다.

## 독립 상태 판정

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 독립 public 전체 원본/startup gate·RO source mirror | YES | YES | YES (D4A local) | YES (named) | 재사용 |
| 기존 output volume/observer topology/bounded native tar transport | YES | YES | YES (기존 owners) | YES (named) | 현 public UV run의 cache DATA consumer |
| TMP cached query와 tool-install environment readonly query | YES | YES | YES (D3 local/기존) | YES (named) | 기존132/UV120 그대로 보존 |
| Native cache를 기존 exportable volume에 보관 | NO | NO | NO | NO | fixed /haa-site/uv-run-cache의 현재 consumer |
| 같은 fixed output cache의 exact helper/env query 경계 | NO | NO | NO | NO | 아래 narrow CPP extension/negative latches |
| Native 전체 census/원본·console selection 동등성 | NO | NO | NO | NO | 실제 bounded census를 먼저 확인한 뒤 기존 wheel/controls/script matcher 연결 |
| Current approved execution adapter·uvx aliases/CLI/final | NO | NO | NO | NO | 후속 acceptance; 기존 Python backend/app는 재구현하지 않음 |

## Exact map와 두 번째 검토 — source 변경 이전

기존 구현을 대체/재작성하지 않는다. UVToolQueryEnvironment에 fixed
/haa-site/uv-run-cache/archive-v0/secureID16/bin/python branch를 추가하고 기존 TMP/영구 tool
branch의 유효 입력·의미를 보존한다. 현재 /haa-site의 expected/actual workspace·9p·RW·root
parent 한 개, sealed namespace/snapshot/readonly image·runtime과 cache/helper/env의 추가
mount shadow 부재를 확인한다. Prefix/name은 코드 인증이 아니며 실제 pinned Python/UV
image·UV creator/group/start/expected·ARTIFACT/clone child·active·consumed/pending/network
조건을 기존 owner가 계속 확인한다. 새 일반 control/runtime/shell/tar 권한을 만들지 않는다.

IsExactUVPythonQueryProgram은 현재 old26bytes TMP launcher를 보존하고 fixed output cache의
/.tmp+alnum6 root만 추가한다. 새 helper root와 cached environment가 같은 fixed cache인지
결합한다. Base-image query는 env가 비어 있고 새 helper root만 있을 수 있다. 다른 tool/cache
alias와 새 helper를 섞지 않는다. Canonical fixed Python binary·argv0/-I/-B/-c·exact code/argc,
기존 root/runtime image checks와 post-clone/reexec taint는 유지한다.

ClassifyFilesystemOpen의 current cached-query branch에만 exported cache를 연결한다. 새 mode는
/haa-site 및 /tmp의 일반 workspace 예외보다 먼저 작동한다. 같은 exact helper subtree의
기존 RO module ABI와 inert env dirs/cfg만 허용한다. cfg flags32768/557056, dir557056/622592,
네 opendir624640 및 current Python image 조건을 그대로 적용한다. Package body/.pth/
customization/metadata·foreign env/helper·정규화/다른 anchor·write·role/creator/start 변조·
network·clone/reexec는 actionable ARTIFACT로 유지한다. Base query에는 임의 env dir/cfg를
허용하지 않는다. 권한/예산/profile/wire·runtime pins/Core/CLI·일반 filesystem 계약은 불변이다.

현재 UV query latch 하나의 기존120 cases를 바꾸지 않고 같은 lambda에 export mode cases를
추가해 새 정상 base/alias 및 ID alphabet과 malformed path/argc/code/image/topology/creator/
readonly/role/start/clone/reexec를 검사한다. Normal metadata 범위 밖 reads와 base-query의
임의 env 접근 거부도 확인한다. 새 latch source 동일 SHA의 old CPP RED→new CPP GREEN과
fresh canonical132·실제 native public DATA/기존 Python 및 영향받은 owners로 한정 검증한다.
로컬에서는 pinned retained-cache replay를 개발에 재사용하고 fresh build는 최종 후보에 모은다.

Go tagged consumer는 B1 원본을 수정해 /haa-site/uv-run-cache를 지정한다. 기존
readUVNativeTar의 허용된 pythonSitePath/.와 callback·bounded tar/정규화/count/expansion/
SHA256/중복/padding 검사를 사용해 전체 output을 DATA로 읽는다. Production exporter의
허용 경로를 임의 /tmp로 넓히지 않고 Host executor를 꺼내 우회하지 않는다. Native tool-run은
whole frozen original/RO seal 이후 offline/no-index/no-build/no-config/fixed Python/copy/
no-sources·고정 /dev/null EACCES만 유지한다. Artifact program은 실행하지 않는다.

B2의 첫 실제 census는 관측 DATA이며 원본/RECORD/script·선택 동등성이 별도 결합될 때만
그 named gap을 닫는다. canonical wheel parser/UVInstalledWheelControls/ValidateUVNativeConsoleScript와
기존 Model C installed mapping을 재사용하고 native tool-install receipt/identity를 uvx로
위장하지 않는다. Current whole Policy/프로그램 실행/CLI와 전체 final은 별도다.

## 검증 주기와 정리

같은 단위의 formatter/tagged vet/pinned Staticcheck/build/docs/security를 적용한다. 기존
production Go/default selected files와 bytes가 그대로면 Quick/전체 Race를 재사용하며 새 실패/
수정/미해결 우려 없이 전체 suite를 반복하지 않는다. CPP 공통 변경은 새 latch·영향받은 현재
native/Python consumers를 실제로 확인한다. Fresh CPP/full source checks/hosted wiring은 의미
있는 완성 후보로 모으며 작은 helper·문서·진단마다 push/full CI를 돌리지 않는다. 다음 actual CI를
시작할 때에만 실제20분 GET timer를 새 등록한다.

Host5/source/고정42/ZIP/Docker·complete trace와 outside/network/unexpected0·helperwait0/모든
owned 정리를 분리해 보존한다. 종료 사본만 전체 root references0/query errors0 뒤 제거한다.
원본·실행 중/reusable cache·SDK·fixed supplies·Git은 보존한다. 현재 누적235ELF+6 completed
cache의 약10.62GiB Linux 논리 용량이며 Windows C/VHD physical shrink 주장은 없다.

## B2 named local 완료

기존 CPP의 같은 latch RED47.499s/131·새 normal/id/base3 FAIL 뒤 candidate GREEN45.318s/132·UV195(기존120+새75)를 확인했다. Fresh canonical274.556s/132와 replay의 최종 binary010cd8은 byte-identical이다. Canonical9는 current7/정확한 default/Bootstrap 입력 재사용2이며 affected sandbox Race24.106s를 실행했다. Actual native public2/49.948s operator의 새 volume24.30s는 complete16events/1842B/workspace124·Outside/Network/Honeytoken/UnexpectedExec0, 기존 TMP23.43s도 complete/Outside0이다. Native 전체 DATA84/regular53/dir25/link6·225462B와 env하위41개를 수집했다. 기존 public Python2/72.84s parent는 fresh whole current Evidence/Policy ALLOW·complete0/2와 전체 정리가 PASS다. Host5/source/42/ZIP/Docker 동일·helperwait0/owned resources0과 원본을 보존했다. 종료 사본4개85971665B는 root 참조0/errors0 뒤 정리해 누적239ELF+cache6/약10.70GiB 논리 용량이다.

이는 fixed query/export/census named 완료이며 native 원본/RECORD/script 선택 동등성은 아직 미완료다. 관측된 cfg는 relocatable=true이고 console wrapper도 realpath/dirname 방식이다. 기존 nonrelocatable matcher를 확대 적용하거나 query 준비를 프로그램/Policy 성공으로 쓰지 않는다. 다음 B3는 고정 UV의 relocatable producer 원본과 existing wheel/installed/control/script owners를 감사해 실제 binding gap만 구현한다. CI434는 old d81의 macOS120s/RequiredFAIL·Linux104SUCCESS로 끝났으며 다음 타이머/새 CI는 없다.
