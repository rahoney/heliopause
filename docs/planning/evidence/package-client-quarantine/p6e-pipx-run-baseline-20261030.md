# P6-E — pipx run baseline와 E1 원본 공유 지도

Baseline Root6f5535b0e9b7e777df5c31cabebc491531ce059f/tree8c3cfe09/3121tracked·runtime1392다. Active work item은 M13-004 하나다. CI436/source775c19c/treebcc64338/3120·1392는 frozen이고 worker52419/PID3546388가 실제20분 GET을 수행한다. 첫20:20:39UTC(한국05:20:39)이며 새 push하지 않는다. D8 hosted 결과는 PENDING이고 독립 DATA/source audit와 필요한 원본 공유만 진행한다.

Read: PROJECT/Queue/docsREADME·lessons INDEX/artifact L01/L02/sandbox L03/L08/L12/L18 및 P5-C3b tmpfs/evidence L16/ci L05/L06, Architecture001/기존 dependency/inspection 경계·client09§1/현재 CLI lane·plan20§4–6, P6 baseline·D5/D6/D7/D8 receipts/원본, pipx_resolver.go/native_command.go/native_session.go/native_output.go와 whole original/native input/data volume·source-console binder/current Python preparation/runtime·UV ResolveToolRun source를 대조했다.

## Baseline audit

| Owner / 현재 범위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Fixed pipx1.18.4 ZIP42 공급/installed DATA lease·runtime 입출력 | YES | YES | named install/update YES | 해당 범위 YES | run query geometry와 complete native temporary cache binding |
| PipxResolver.ResolveToolInstall/Upgrade + fixed backend UV | YES | YES | 기존 named YES | 기존 named YES | 별도 run semantics·request/selection identity·source-console binding |
| UV ResolveToolRun public source/frozen originals/offline native DATA | YES | YES | D5/D7 named local YES; CI436 PENDING | named local YES | pipx consumer가 같은 frozen originals를 재사용할 private function boundary |
| Current whole Python execution approval/binder/runtime/report | YES | YES | current named local/이전 hosted YES | named slice YES | pipx-specific selector와 immutable selected command coupling/CLI actual |
| Native pipx run/CLI/current application | NO | NO | NO | NO | 고정 native 경로·whole output binding·entry semantics·현재 approval/runtime·CLI/실제 소비자 |

## 고정 공급 원본의 관측 — 실행하지 않은 DATA

고정 ZIP SHA0d02e769a58079e1a123fd6a68dd305a23be4381e3f83e7ab1e1f30d6e4d7670에서 run.py/venv.py/main.py/run_uv.py 네 원본156878B와 paths.py를 직접 읽고 원본/SHA를 보존했다. 새 release/버전 채택이나 interpreter import/Artifact 실행이 아니다. Raw p6-e-pinned-pipx-source-inventory 및 paths-followup이 원본을 소유한다.

- run.py359..461: explicit backend UV라도 동일 package/command·spec 조건은 UV tool run으로 위임하지만 다른 command/spec·python_args 등은 Venv 경로를 쓴다. Backend를 매번 자동 탐지하거나 install을 run identity로 바꾸면 안 된다.
- run.py244..346/480..556: temporary venv·전체 request에 의한 cache·sole-app fallback, optional local __pypackages__, additional deps와 no-cache의 차이가 있다. HAA에 local 검색/extra/runtime override를 자동 수입하지 않는다.
- venv.py740..765: pipx.run group이 console/bin보다 우선하며 Python -c로 loading한다. 기존 canonical console entry와 같다고 가정하지 않는다. Native preparation에는 선택된 program/user argv를 보내지 않으며 inert query terminal도 whole original의 모든 관련 entry/script와 충돌하지 않음을 먼저 확인해야 한다. /dev/null 문자열만으로 pipx.run metadata 실행 불가를 증명하지 않는다.
- paths.py64..79/run.py564..578: 명시 PIPX_HOME에서는 cache가 home/.cache이고 hash는 native JSON field boundaries의 SHA256 앞15자리다. 기존 /haa-site/pipx/.cache는 exportable output volume 아래에 있으므로 새 가상 환경 변수나 private tmpfs Docker cp를 만들지 않는다. Native cache/metadata는 unapproved DATA다.
- 기존 CPP1032..1169/3333..는 fixed installed root/UV/query와 /haa-site/pipx/venvs/<project>를 분류한다. 임시 .cache/<15hex>에 그 permission을 전용하지 않는다. 새 ABI가 실제 필요하면 해당 current Python/parent/native operation/readonly geometry만 scoped audit·same-latch positive/negative 및 기존 소비자와 검증한다. 현재 source/CPP를 바꾸지 않았다.

## E1 source 변경 전 정확한 재사용 지도

기존 internal/sandbox/uv_run_resolver.go ResolveToolRun는 UV native proposal/independent archive→owned frozen originals→startup/source-console gate와 offline isolated native census/binding을 한 public body에 소유한다. Public result에는 source bytes가 없으므로 pipx가 이 public method만 호출해 같은 원본을 재사용할 수 없다. 다시 resolve/acquire하거나 ToolInstall을 위장하지 않는다.

E1은 이 기존 body를 private prepareUVRunOriginal(동일 native source/archive/freeze/entry 확인)과 resolveFrozenUVToolRun(동일 offline UV/census/source-console selection) 함수로 기계적으로 분리한다. 기존 public ResolveToolRun가 두 함수를 그대로 호출하므로 새 placeholder interface/config가 없고 모든 body는 즉시 기존 consumer가 사용한다. 원래 public validation/context/defer·오류/완료 의미와 원본 freeze·정리·Policy 분리는 유지한다. 구현을 대체하거나 새 resolver/parser를 작성하지 않는다. Before source와 extraction map/fragment hashes를 남기고 return carrier/첫 err 선언 이외의 body 차이를 거부한다.

E1의 IMPLEMENTED/원본 재사용 WIRED gap만 변경하며 canonical format/focused/Quick·영향 Sandbox Race·tagged build/vet/pinned Staticcheck로 검증한다. Native public actual/CPP와 원격 CI는 이 기계적 분리만으로 반복하지 않는다. E1 local verification을 pipx run/새-source actual qualification으로 확대하지 않는다. Native pipx run query/임시 cache/complete original payload·pipx.run precedence/두 native 경로의 bounded 동등성, 현재 app/CLI/alias/forced backend/cache negatives는 다음 E2/E3에서 기존 owner를 수정해 완성해야 한다.

## E1 로컬 검증

기존 UV public caller가 prepareUVRunOriginal/resolveFrozenUVToolRun를 즉시 사용한다. Formatter 뒤에도 두 fragment를 error carrier6/첫 err 선언1만 역변환하면 기존 body와 byte 동일하며 public context/validation/defer는 유지했다. Canonical8/Quick131.983s/전체 Sandbox Race27.14s 및 tagged vet/pinned Staticcheck가 PASS다. 새 native/CPP actual·ELF는 만들지 않았다. [E1 receipt](./p6e1-original-sharing-local-20261030.json)는 source-reuse slice만 닫고 실제 pipx run·임시 cache/ABI·entry precedence·현재 app/CLI/hosted는 NO로 유지한다. CI436 source는 고정이며 새 push하지 않는다.

## E2 다음 수정의 경계 — 아직 구현·qualification이 아님

1. 기존 PipxResolver/native_command/native_session owner 안에서 별도 run operation을 추가한다. Fixed backend UV/whole owned originals/DATA lease·동일 admission/accounting/trace/cleanup을 재사용한다. Install 결과·영구 ToolReceipt를 run identity/임시 ownership으로 바꾸지 않는다. E1의 두 함수는 native UV와 pipx가 동일 원본을 재resolve/acquire 없이 소비하도록 한다. Native UV 위임 branch와 pipx Venv branch의 frozen single-choice graph/원본 controls 동등성을 각각 증명해야 하며 현재 source 감사는 실제 동등성 qualification이 아니다.
2. Native 준비에 선택 console/user argv를 보내지 않는다. Fixed query name은 원본 primary project·전체 wheel의 script role 및 관련 entrypoint와 충돌하지 않아야 한다. `pipx.run`이 console보다 우선하므로 /dev/null EACCES 방식만 전용하지 않는다. 같은 canonical whole inspection의 Surface.EntryPointDetails/Scripts를 사용하며 resource basename을 실행 역할로 승격하지 않는다. 고정 inert terminal과 bounded stderr는 상태 구분일 뿐 native source/Policy authority가 아니다.
3. Existing exportable /haa-site/pipx/.cache/<15hex>의 전체 결과를 기존 bounded tar/census/installed-wheel control·원본 payload/console recipe로 결합한다. Native cache hash·pipx_metadata/RECORD·activation은 unapproved DATA이며 Host ownership이 아니다. Before/after lease/RO source/전체 input/trace·cleanup을 검사하고 query 실패/정리 오류는 최초 원인을 보존하며 result를 비운다. Private tmpfs Docker cp를 다시 시도하지 않는다.
4. 기존 CPP root argv·current Python/creator/query/readonly predicates를 읽고, run operation과 임시 cache15의 최소 경계만 같은 latch positive·변조 negative로 추가한다. Path prefix만의 grant, installed /venvs role 전용, parent/image/start/readonly 검증 생략은 금지한다. 변경이 필요하면 affected old pipx/UV/native program 소비자와 실제 새 준비를 한 묶음으로 검증하며 현재 CI436 source/기존 CPP132·UV195는 고정한다.
5. Current whole inspection/Evidence/M3/M5/approved original console·readonly Python runtime·privacy/report는 기존 D6/D7 owner를 재사용한다. Pipx-native identity를 UV로 위장하지 않는다. Bare/--spec·sole-app inference와 pipx.run group의 지원/명시 거부 범위를 actual current command binding에 결합한 뒤 descriptor/preflight를 추가한다. 미래 alias·추가 package·runtime provisioning/Host local cache 검색/TTY/export를 먼저 scaffold하지 않는다.

E2 source 작성 전 남은 판정은 정확한 safe native terminal/whole collision guard·새 cache의 실제 skeleton/control와 scoped query ABI다. Cheap syntax/whole source guards·native argv/입력/layout negatives를 먼저 모으고 큰 native/CPP actual은 완성된 경계에서 수행한다. 새-source hosted109의 current approval/runtime 결과는 CI436 timer/evidence에서 별도로 확인하며 active run 중 작은 helper push는 하지 않는다.

## E2-A 재개 범위 — 고정 원본의 안전한 준비 요청 진단

현재 원격 후보는 [CI437](./ci437-registration-20261030.json)의 f4f7abc로 고정한다.
이하 로컬 진단은 CI437 qualification이나 public pipx run 구현이 아니다.

| 재사용 owner / 현재 범위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Whole original/startup freeze·console metadata — pipx_native_input.go/uv_run_console_binding.go | YES | YES | 기존 bounded YES | 기존 범위 YES | 준비 명령과 pipx.run/console/gui/raw script 충돌 거부 |
| Typed resource·DATA/UV/output·observer·cleanup — pipxNativeSession | YES | YES | 기존 install/update YES | 기존 범위 YES | 임시 cache 조회의 실제 첫 경계 |
| Native run 준비 요청 진단 | NO | NO | NO | NO | 기존 realcorpus fixture/session을 소비하는 tagged probe 하나·고정 argv/진입점 거부 회귀 |
| Public pipx run/current application/CLI | NO | NO | NO | NO | 전체 임시 결과 binding·native 두 경로 의미·current 승인/실행·CLI |

고정 ZIP의 run.py471..477은 command와 normalized requirement name이 같으면 UV로
위임한다. 다른 고정 준비용 command는 Venv 경로이며 _prepare_venv530..549의 sole-app
fallback과 venv.py727..765의 pipx.run 우선 조회를 먼저 차단해야 한다. 준비용 이름은
고정 상대 basename으로 정하고 whole original의 실제 entry/script 역할과만 대조한다.
리소스 basename을 실행 권한으로 분류하거나 /dev/null을 그대로 전용하지 않는다.

Production API/새 session engine/권한/CI는 추가하지 않는다. 기존 realcorpus 입력42와
실제 두 wheel·synthetic lock을 재사용하는 tagged 진단에서 기존 prepare/start/cleanup과
동일 ledger/lease/admission을 호출한다. Host의 기존 보호 profile을 쓰는 실제 진단은
그 profile의 DATA 관측일 뿐 f4f7abc의 matching hosted 검증으로 확대하지 않는다.
먼저 저비용 entry/argv 거부와 tagged compiler/vet/Staticcheck를 확인하고, 실제 첫 거부와
임시 cache 형상이 확인된 뒤 그 missing만 production owner에 연결한다.

### E2-A 최초 실제 결과 — 준비 출력과 관측 완료를 구분

Tagged 준비 probe 하나의 canonical format/focused/vet/pinned Staticcheck는 PASS다.
기존 whole input42/userpath1.9.2·click와 synthetic lock의 단일 actual은14.317s/FAIL이다.
고정 missing-app 출력은 도달했지만 observer는 PROCESS_PROVENANCE_UNKNOWN/SENTRY_EXEC,
12events/1368B/session_complete=false다. ParsePipxNativeRoot가 install/upgrade/--version만
인정하는 기존 분기와 실제17argv run 요청이 대응한다. 캐시 query의 다음 오류까지 원인으로
확대하지 않으며 selection/Policy/current app/CLI는 미도달이다.

Host 기존 close/exit profile과 같은 구형 canonical132 observer의 진단이다. f4f7abc의
matching profile/native qualification이 아니다. 자원 create/register/close각1·CPU2136963us,
helperwait0/CONFIRMED_STOPPED·Host/Docker 동일·남은 helper/query errors0을 확인했다.
원본 source/실행·정리/첫 실패는 raw `p6-e2a-first-preparation-v1/inventory.json`이 소유한다.
다음 missing은 기존 root argv와 같은 creator/current-image/readonly 경계를 유지한
run 준비 ABI다. 기존 구현 재작성·일반 cache prefix grant·새 상태 구조는 추가하지 않는다.

### E2-A named local 검증 — 공개 run은 아직 미구현

[한정 영수증](./p6e2a-pipx-run-preparation-local-20261030.json)은 기존 root/clone/creator/UV/
query 함수에 fixed17argv exact 준비와 임시 cache15의 조회만 추가한 경계를 확인한다.
첫 후보는 기존 clone-candidate의 operation 연결 누락으로 두 latch가 실패했으며 원본을
보존했다. 해당 조건 한 곳을 보완한 targeted 회귀와 최종 C++132/48.917s가 PASS다.
새 상태 구조·네트워크 grant·관찰 이벤트·예산은 추가하지 않았다.

최종 tagged format/compiler/vet/Staticcheck와 실제13.142s/graph2 preparation이 PASS다.
현재 observer와 canonical close-enter profile을 실제 Host runsc가 함께 소비하게 맞춘 뒤
정리와 helper/container/query 부재를 확인해 같은 inode의 원래 profile bytes로 복원했다.
보호5/Docker는 전후 동일하고 자원각1/CPU2572031us/helperwait0을 확인했다.
Census83의 cache c3dc012f54be29e는 nonrelocatable wrapper·일반 UV pyvenv.cfg와
pipx_metadata.json을 가진 unapproved DATA다. Native metadata를 가짜 ToolReceipt로
승격하지 않으며 E2-B의 전체 원본·control·console 결합, production resolver·현재 앱/CLI는
여전히 MISSING이다. 실행 중인 CI437에는 이 새 source가 없고 새 push/CI는 하지 않았다.

## E2-B 변경 전 audit — 임시 결과와 기존 세션의 실제 소비

Baseline a9bfc3a의 E2-A 준비 진단을 재사용한다. 기존 bindUVRunConsoleData의 whole
frozen wheel/control·payload·별칭·선택 wrapper 결합은 IMPLEMENTED/WIRED=YES,
QUALIFIED/ACCEPTANCE_CLOSED는 UV named 범위만 YES다. Pipx cache15의 일반 cfg와
nonrelocatable wrapper 연결은 WIRED/QUALIFIED/ACCEPTANCE_CLOSED=NO다. 기존
ValidateUVNativeConsoleScript가 이 wrapper를 이미 구현하므로 새 matcher를 만들지 않는다.
UV와 pipx 두 소비자가 같은 기존 결합 body를 사용하고 환경 형상·cfg·scaffold·wrapper
recipe의 실제 차이만 분기한다. Native pipx_metadata는 bounded DATA이며 ToolReceipt나
선택 authority로 해석하지 않는다.

기존 runPipxNativeOperation의 resource/admission/start/trace/cleanup은 모두 구현·연결된
owner다. Run용 복사 세션을 만들지 않고 같은 body를 callback 경계로 공유해 기존 install/
upgrade가 즉시 소비한다. E2-A의 test 전용 argv/collision guard를 실제 run 소비자로 옮긴다.
MISSING은 whole temporary binding, 같은 frozen 원본을 재획득하지 않는 production run
entry와 기존 세션 연결, 영향 회귀·새 production actual이다. 현재 app/CLI·sole-app/default
선택과 native 위임 의미/새-source hosted는 별도 미완료로 유지한다. 먼저 cheap original/
layout/recipe negatives를 모으고 최종 Quick/영향 Race/native를 의미 있는 변경 묶음으로
검증한다. CI437은 sourcef4 그대로이며 새 push하지 않는다.

### E2-B named local 결과

[검증 영수증](./p6e2b-pipx-run-resolver-local-20261030.json)은 실제 PipxResolver.ResolveToolRun,
원본을 재획득하지 않는 frozen replay와 기존 lifecycle/binder/fingerprint 공유를 확인한다.
원래 session의 transport/supply 검사와 lifecycle/defer bytes는 동일하고 기존 install/upgrade
callback은 들여쓰기만 바뀌었다. UV descriptor body도 pipx 분기를 역변환하면 byte 동일하다.
기존 API·보안 경계를 대체하지 않았다. Bare 요청은 같은 frozen graph의 exact primary로만
native 준비하며 요청 자체는 fingerprint에 남긴다. 선택 command의 pipx.run 우선순위는
명시적으로 거부하고 sole-app/default·CLI 지원은 다음 current 연결 범위에서 다룬다.

첫 fixture compile 실패, 실제 빈 .lock0666 거부와 기본 U1000을 보존했다. 기존 wrapper/
install reader의 좁은 빈 lock 조건을 재사용하고 실제 public 소비자를 완성했다. 최종
Quick282.759s/전체 Sandbox Race65.573s·tagged vet/Staticcheck와 저장 census 결합이 PASS다.
기존 native install·accounting 거부/new frozen run의3사례가 PASS이고, 이후 첫 public actual은
recorder가 실제 pipx-native-resolver 대신 UV phase를 기대해 FAIL이었다. 그 원본53.08s와
미도달 negative를 보존하고 test-only 두 조건만 고쳐 public 두 사례만 재실행했다.

최종 public pipx52.425s는 실제 source/원본·console 결합 정상과 관측 불완전 거부를 확인했다.
정상 graph1/거부 graph0이며 trace6 raw complete·주입된 native1만 delivered incomplete다.
영향받은 기존 UV 두 actual도 같은 ELF의50.264s로 PASS다. 각 native offline의 Outside/
network/unexpected/honeytoken은0이다. Host 보호5/Docker 전후 동일·helperwait0·설정 원상
복원을 확인했다. CPP source는 E2-A와 같아 반복하지 않았다. Public ZIP 공급은 private
namespace의 compiled pin fixture이며 release/publisher 활성화 증거가 아니다.

이 named source/native slice만 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES다.
Current whole inspection/승인·approved console 실행/app·CLI/aliases·일반 동등성/새-source
hosted와 whole final은 OPEN이다. 실행 중인 CI437/f4에는 E2 source가 없고 새 push하지 않았다.
