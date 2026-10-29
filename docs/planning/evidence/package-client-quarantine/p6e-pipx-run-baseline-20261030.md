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
