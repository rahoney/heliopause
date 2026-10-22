# P1 complete UV sync — 다음 gap의 감사와 수정 지도

Source3871c6f의 C3d actual CI381을 기다리며 기존 owner를 읽은 사전 지도다.
C3d qualification 전 새 sync source는 수정하지 않는다. 이 문서는 P1의 uv pip sync 범위다. uv add/sync/lock 등 uv project 명령은 확장 계획의 P4이며 이 내부 project snapshot 재사용과 구분한다. M13-004 하나만 IN_PROGRESS이며
P1 acceptance→Go1.27.3→P2 순서를 유지한다. 아래 미완료 항목은 새 승인 관문이 아니다.

Read: [P1 계획](./p1-uv-baseline-20261021.md),
[공통 guard와 실제 native 의미](./p1c-guarded-client-baseline-20261022.md),
[native UV 인계](./p1d-native-uv-ownership-baseline-20261022.md),
[M5 계약](../../../domain-model/14-m5-pypi-pip-contract.md),
[promotion 책임](../../../architecture/04-evidence-staging-promotion.md),
[promotion 계약](../../../domain-model/09-contract-evidence-staging-promotion.md),
[client UX](../../../user-journey-cli-ia/09-package-client-quarantine.md).

## 실제 baseline — 재구현하지 않을 owner

| 범위 / 기존 symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Domain ProjectDependencySnapshot / NewDependencyFreeProjectSnapshot / ProjectDependencyUpdateFromSnapshot | YES | YES: Go/Cargo | YES: 기존 소비자 | YES: 기존 범위 | UV 소비자만 없음. Primary-free와 explicit empty를 재사용 |
| application ProjectInspectService / executeGuardedProjectUpdate / policy.M4.EvaluateProjectSet | YES | YES: Go/Cargo | YES: 기존 소비자 | YES: 기존 범위 | UV native 선택·staging·mutation 연결만 없음 |
| promotion pypiVenvTransaction / prepare / commit / rollback / persistent journal | YES | YES: pip/UV install/update | YES: CI380 managed 범위 | YES: 기존 범위 | 현재 merge semantics와 empty refusal을 sync의 명시 complete intent와 구분 |
| artifact/pypi UVLock strict parser·installed WHEEL·독립 archive/metadata/Requires-Dist binding | YES | YES: single root | YES: CI380 범위 | YES: 기존 범위 | Primary-free nonempty/empty lock 소비자와 전체 선택 검증 |
| sandbox UVResolver native tool/observer/network-policy/private volumes·independent Python data reader | YES | YES: install/update | YES: CI380 범위 | YES: 기존 범위 | Complete sync source·empty transport 및 기존 lifecycle 재사용 |
| UV complete/multi-root/explicit-empty sync end-to-end | NO | NO | NO | NO | 아래 C3e1..3 및 실제 source qualification |

현재 source의 `pypiVenvTransaction.prepare`는 len(destinations)==0을 거부하고, 새 output에
있는 distribution만 교체하며 다른 owner는 남긴다. `preparePyPIProject`와 일반 staged bundle은
단일 primary가 있는 graph/nonempty Manifest를 요구한다. 이를 완화해 empty ALLOW를 만들거나,
sync를 요청 root 한 개의 InstallRequest로 위장하지 않는다. Domain과 Application의 기존
project snapshot/set/guard 순서를 그대로 사용하고 Python wheel·출력 adapter만 연결한다.

## Native 의미를 먼저 고정

[공식 sync 설명](https://docs.astral.sh/uv/pip/compile/#syncing-an-environment)은
requirements.txt 또는 pylock.toml의 설치 집합과 환경을 맞추며, 일반 install은 무관한 기존
패키지를 남긴다. 앞 data-only probe의 compile→sync 성공은 입력 requirements의 모든 native
문법·실제 제품 삭제 승인을 뜻하지 않는다. 일반 sync 입력을 compile root 입력으로 간주해
목록에 없는 dependency를 자동 추가하지 않는다.

고정0.13.4에서 bounded exact requirements/pylock 입력의 선택·빈 입력·누락 dependency·hash와
삭제 의미를 확인한 뒤 허용 grammar를 확정한다. Include/constraint/direct/local/VCS/editable/
sdist/build/다른 index/알 수 없는 marker·option은 새 지원을 검증하기 전 명시 거부한다.
실제 native 선택과 독립 wheel metadata의 전체 dependency closure가 다르면 거부한다.
추가 패키지를 임의로 설치해 성공시키거나 unsatisfied Requires-Dist를 생략하지 않는다.

## C3e1 — primary-free format과 독립 binding

artifact/pypi/uv_lock.go의 strict wire/header/Python/source/size/hash/alternative bounds,
SelectInstalledWheelControls와 whole archive→METADATA→Simple metadata 대조 본문을 공통 private
helper로 재사용한다. 기존 ParseUVLock/BindWheels는 단일 primary/nonempty 규칙과 기존 tests를
유지한다. 별도 complete 소비자는 primary 없는 selected artifacts와 검증된 whole graph를
기존 ProjectDependencySnapshot에 bind한다. 임의 첫 package를 primary로 만들지 않는다.

Explicit packages=[]와 명시 frozen control은 zero/누락 input과 다르다. Empty인 경우에만
NewDependencyFreeProjectSnapshot을 사용하고, control bytes·정확한 UV runtime/target·native lock·
full graph digest를 함께 고정한다. Nonempty에서는 모든 selected archive/metadata와 dependency
edge·constraint를 검증한다. Disconnected roots도 누락 없이 포함하되 duplicate project/version,
missing wheel/dependency, unknown fields와 budget 초과를 거부한다.

## C3e2 — 기존 native lifecycle에 complete source 연결

sandbox/uv_resolver.go의 tool 검증·endpoint/network-policy·observer·private output·attachment/
cleanup을 재사용한다. 이미 존재하는 install/update native selection의 의미는 유지한다.
Frozen source bytes를 private fixed path로 전달하고 실제 UV sync 선택을 얻는다. 원본 source
파일과 target cfg/METADATA/ownership은 선택 전에 guard하고 선택·검사·반입까지 다시 확인한다.
Host resolver/target Python을 실행하거나 artifact 출력을 완료 authority로 사용하지 않는다.

uv_resolver_reader.go의 independent metadata/whole archive transport도 재사용한다. 현재
withPythonResolverSession은 source 확인에 ArtifactReference를 쓰므로 complete consumer는
private session owner에 source-bound 진입을 추가하는 방향을 검토한다. Dummy reference/primary를
만들거나 정책 경계를 생략하지 않는다. Empty는 임의 네트워크 성공 대신 frozen source와 native
빈 lock·관측/종료·guard의 명시 증거를 요구한다. 기존 single-root 범위와 default 한도는 유지한다.

## C3e3 — exact complete publication / CLI

실제 owner별 후속 감사·상세 수정 지도와 source 착수 전 남은 검토는
[C3e3 publication baseline](./p1f-uv-sync-publication-baseline-20261023.md)이 소유한다.

1. 기존 application guarded-project-update 순서로 guard→native selection→전체 entry inspection/
   Evidence/entry Policy→set Policy→staging→commit→close를 연결한다. Empty도 실제 snapshot/set
   Policy와 guard를 거치며 가짜 artifact/check/성공 Evidence를 만들지 않는다.
2. Python project cache adapter는 기존 LocalStaging.stageArtifact·digest copy·recorded Evidence와
   exact receipt binding을 재사용한다. Go/Cargo cache layout을 Python에 강제로 사용하지 않는다.
   Artifact filename·digest·size와 전체 controls/runtime/graph·set approval을 receipt에 bind한다.
3. Offline Python materialization/validatedPyPIDestinations와 기존 transaction을 재사용한다.
   Complete intent에서만 제외한 모든 proven old owners를 obsolete로 계산한다. Empty commit은
   해당 intent와 explicit approved empty set에만 허용한다. 일반 pip/UV install/update의 merge/
   empty refusal은 유지한다. Journal은 이미 Current/Desired/file intent와 원본을 보존하므로
   삭제·중간 실패·rollback·복구에 같은 owner를 사용한다.
4. Native --adopt는 새 선택에서 제외되어 삭제될 distribution까지 모두 공식 이전 wheel로
   증명해야 한다. 선택된 graph만 old proof 하는 기존 경로를 그대로 사용해 unselected native
   extras를 지우지 않는다. Native generated scripts/compiled caches 등 현재 C3d의 refusal을
   유지하고, 인증되지 않은 파일·사용자 데이터·cfg/UV bootstrap 파일은 제거하지 않는다.
5. CLI는 같은 compiled client descriptor/composition에 sync operation을 연결한다. Runtime
   초기화 전 source/args/flags 검증과 lazy help·7개 locale의 기존 fallback을 유지한다. SourceID는
   pypi이며 미래 client enum/plugin을 추가하지 않는다. 결과는 기존 project report를 재사용해
   complete selected count/explicit empty·검사/Policy·삭제/반입 완료·오류를 구분한다.

## 검증과 두 번째 검토

Canonical format/Quick/security/docs/workflow를 먼저 실행하고 영향받은 parser·Application·
Policy·Python transaction·CLI/Bootstrap의 의미 있는 정상/부정 및 Race를 확인한다. 실제 native
source에서 two-root+dependency·기존 extra 삭제·version update·explicit empty·same-set fresh
Evidence·소스/target 변조·native ownership 거부·partial publication rollback을 연결한다.
Shared-owner conflict, disk/resource/unknown query·취소/cleanup의 최초 오류를 보존한다.
같은 source의 Required/Linux·기존 pip/native/public consumers/macOS와 exact cleanup을 확인한다.
기존 설치·업데이트의 single-node 성공을 complete sync나 whole P1 qualification으로 확대하지 않는다.

두 번째 검토: 삭제 권한은 native exit/empty JSON/RECORD filename이나 선택 graph 부재에서
생기지 않는다. Original ownership proof와 명시 complete intent, 전체 선택의 검역·set Policy,
frozen control/guard와 transaction이 함께 있어야 한다. Source control의 변조·symlink/alias·
누락/empty 구분, unselected native owners의 proof와 shared owners까지 검토한다. 기존 구현을
대체하는 새 engine·generic framework·future placeholder는 만들지 않는다. 고정 tool/image/
runtime/resource 한도와 지원 범위는 이 지도로 바꾸지 않는다. 완료한 시험은 작은 재현 원본을
보존하고 exact owned 자원·마지막 참조/성공한 부재 확인 뒤 정리한다.

## C3e2 사전 감사 — source-bound native producer와 reader

C3e1 source12d9a50의 새 CI384를 기다리며 실제 symbol을 대조한 다음 단위의 지도다.
C3e2 제품 source를 수정하기 전의 감사다. IMPLEMENTED=NO / WIRED=NO /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO이며 MISSING은 아래 native complete 소비자와
독립 reader·empty transport 및 실제 native 검증이다. 기존 infrastructure는 YES이며
다시 구현하지 않는다. C3e3 삭제·CLI는 별도의 후속 gap이다.

| 실제 위치 / 함수 | 재사용과 정확한 후속 수정 |
| --- | --- |
| artifact/pypi의 새 uv_sync_input.go | frozen source format/bytes와 allow-empty flag를 bounded control로 구분한다. Plain source의 초기 허용 grammar·중복 project·range/hash/include/marker 지원 여부를 구현 전에 확정하고 unsupported는 거부한다. Native compile에 원래 목록 밖 dependency를 추가할 권한을 주지 않으며 selected project/version과 원본 입력을 다시 대조한다. Explicit empty pylock은 기존 ParseUVProjectLock을 재사용한다 |
| sandbox/uv_resolver.go resolveNativeSelectionWithInput | 기존 tool/probe·typed network·observer·named tmpfs·ownership·attachments·종료/cleanup body를 private session 함수로 추출해 재사용한다. 기존 install/update selection 본문은 그대로 callback으로 연결하고 complete source는 별도 private callback으로 들어온다. 임의 primary/reference·Host solver/새 lifecycle을 만들지 않음 |
| 같은 native compile/sync 본문 | Plain 목록은 고정 tool의 --no-deps compile 후보를 검증한다. 저장된 실제0.13.4 compile help에 해당 option이 있으나 native complete 명령의 실제 성공은 아직 미검증이다. Compile과 독립 closure 확인 모두 원본 목록의 의미를 유지해야 한다. Pylock source는 strict 검증한 같은 bytes를 sync에 전달하고 전후 lock 불변을 확인한다. Default blank skip은 이 positive selection 경로에서 거부 |
| sandbox/uv_native_output.go scanUVNativeOutputWithInput | 공통 tar/path/owner/mode/한도·padding·동일 source·control 검증을 유지한다. 기존 single-root caller는 control0을 계속 거부한다. 별도 complete caller만 유효 explicit-empty lock·관측/종료가 있을 때 명시 empty census를 생성한다. Missing site/control·변조·extra/truncated output은 성공으로 바꾸지 않음 |
| sandbox/pypi_resolver.go withPythonResolverSession | 기존 reference/source 확인 wrapper를 보존하고 private source-bound session owner를 추출한다. 기존 pip/PyTorch/UV callback과 runtime·network·observer·cleanup을 공유한다. Complete callback은 실제 public source로 진입하고 가짜 reference나 DependencyResolution을 성공 결과로 만들지 않음 |
| sandbox/uv_resolver_reader.go bindUVNativeSelection | 기존 full archive·독립 exact-release metadata·stream bounds와 private workspace 정리를 공유한다. 기존 single-root는 BindWheels로, complete 소비자는 BindProject로 끝낸다. Empty는 검증된 native 완료와 source/flag/runtime/target/control을 요구하고 가짜 archive/Run/Evidence를 생성하지 않음 |
| 기존 ports.ProjectLockUpdateResolver / Domain.NewProjectDependencyUpdateFromSnapshot | complete 선택을 기존 primary-free update에 연결한다. Original/selected control 이름·개수가 같아야 하므로 guard 시작 때 native 결과 control의 absent 원본과 frozen source/intent/runtime를 준비하고 selected에서만 native lock을 present로 전환한다. Absent original은 선택 전 상태이며 empty 성공/삭제 권한이 아님 |

정확한 guard source 캡처·native 결과용 original absent control·동일 이름/개수 계약은
`project_update.go:newProjectDependencyUpdate`에서 확인했다. Native input codec나 source
세션을 추가한다고 이 계약을 완화하지 않는다. 원본 cfg/METADATA/ledger와 source 파일의
동일성은 C3e3 promoter guard가 소유하며 resolver만으로 원본 변경 권한을 만들지 않는다.
현재 UpgradeInput은 요청 requirement를 필수로 가지므로 sync 원본에 임의 패키지를 넣어
재사용하지 않는다. Installed state/ownership의 실제 C3e3 공통 추출 위치는 별도로 감사한다.

두 번째 검토: 단순 compile 성공이 complete source 보존 증거는 아니다. 고정0.13.4의
실제 two-root+dependency·누락 dependency·empty pylock/blank/명시 flag 의미를 검증하고,
source/flag/runtime/target·native output/readback의 바꿔치기와 cleanup 불확실성을 거부한다.
Shared source session 추출 뒤 기존 pip/PyTorch·single UV 소비자도 검사한다. Empty tar
예외는 새 complete entry에서만 허용하고 기존 collector의 빈 결과 거부는 유지한다.
Canonical profile·관련 Race/fuzz와 실제 native 실행·same-source 원격 소비자 검증을
연결하기 전 production 지원/qualification을 선언하지 않는다. 새 Core interface·정책·
버전 pin·CI job이나 미래 client scaffold는 만들지 않는다.

착수 grammar는 한 파일의 bounded UTF-8 plain requirements(빈 줄·주석·public final
`name==version` pins) 또는 strict UV pylock1.0이다. Plain duplicate normalized project,
range/extras/marker·include/constraint·option/hash continuation·URL/local/VCS/editable·
sdist/build는 이 단위에서 거부한다. 원본 bytes는 보존하고 선택된 name/version 집합이
원본 exact pins와 같은지 확인한다. --no-deps는 해당 목록 유지 후보일 뿐 독립 dependency
closure 확인을 생략하지 않는다. 확장 grammar를 나중에 지원한다면 같은 codec/native/
reader/CLI 부정 회귀를 연결한 실제 변경으로 다루며 지금 지원한다고 표시하지 않는다.
Pylock의 원본 전체 bytes는 native materialization 전후와 같은 control에 묶는다.
Blank plain의 기본값은 SKIP이고 explicit flag만 positive empty로 구분한다. Runtime
control은 실제 고정 UV/Python/target 검증을 수행한 producer가 생성하며 caller 문자열로
pin 인증을 대신하지 않는다. Control 개수/absent 원본의 실제 guard 배치는 C3e3 소비자와
함께 검사하며 미래 ownership placeholder를 지금 추가하지 않는다.

CI384 대기 중 data-only 추가 관측 계획: 잠긴 UV0.13.4/CPython3.14.7의 인증된 tool을
UID1000/read-only/network-none/no-cap/NPP 임시 runc에서만 실행한다. Guest loopback
index의 inert 자체 wheel에 대해 --no-deps compile→pylock→private target sync를 root-only,
complete-two, empty 세 입력으로 한 번씩 확인한다. 이전 plain sync 다섯 명령을 반복하지
않는다. Root-only는 native 목록 보존 관측일 뿐 HAA closure 승인/ALLOW가 아니다.
Actual native lock/count와 fixed guest data-only census, Host 전후와 exact label·image/ID
cleanup을 보존한다. 새 image를 만들거나 Host interpreter/artifact를 실행하지 않으며
작은 원본을 보존한 뒤 완료 workroot/client/operator를 제거한다. 이 조사도 실제 public
공급망·production gVisor/typed network·Policy·사용자 target 삭제 qualification은 아니다.

실제 [data-only 결과](./p1e-uv-sync-compile-format-20261023.json)는 세 compile/sync·독립
data census 모두 exit0, 13.789s다. Root-only는 dependency를 추가하지 않고1개, complete-two는
2개, empty는 유효 packages=[]/0개였다. 두 exact container 제거·label 부재·Host12파일/
images/containers/bridge/service 동일을 확인했고 새 image는0이다. 빈 workroot/client와
완료 operator 두 개는 작은 원본 보존 후 제거했다. --no-deps 후보의 해당 목록 보존은
확인됐지만 actual HAA independent binding/gVisor/Policy·guarded 사용자 반입은 미실행이다.


## 고정 native plain requirements 관측 / 구현 지도 보완

[관측 영수증](./p1e-uv-sync-format-20261023.json)은 inert wheel 다섯 native 사례의 exit0과
설치 census를 보존한다. 마지막 요약 writer가 lock의 uv_version을 version으로 잘못 읽어
operator 전체는 exit1이다. 이 오류는 사례·cleanup·Host post snapshot 뒤 발생했다. 원본을
고치거나 통과한 명령을 재실행하지 않고 개별 JSON/두 snapshot/새 label 부재 조회에서 관측을
복원했다. Corrected reproducer는 보존만 했으며 실행·전체 PASS로 표시하지 않는다. Owned
containers0·Host12파일/image/container/bridge/service 전후 동일 뒤 빈 workroot/client와 operator를
삭제했다. 이 데이터는 public 공급망/검역·Policy·실제 target 삭제 승인이 아니다.

고정0.13.4의 실제 의미:

| Source / flags | Native 결과 | HAA 구현 조건 |
| --- | --- | --- |
| root 한 개만 쓴 requirements | 필요한 dependency를 추가하지 않고 root만 설치 | 독립 Requires-Dist 전체 검증에서 incomplete closure 거부. Compile로 목록을 확장해 sync 의미를 바꾸지 않음 |
| complete three→two entries | 제외한 extra를 제거 | 모든 선택을 새로 검역하고 기존 proven owner의 exact 삭제를 같은 transaction에 bind |
| 빈 requirements, 기본 옵션 | 건너뛰고 기존 two entries 보존 | No-op: target/ownership/Evidence 무변경. Empty approved deletion snapshot으로 변환하지 않음 |
| 빈 requirements + --allow-empty-requirements | two entries 제거 | 명시 flag와 frozen empty source가 함께 있어야 positive empty selection 및 owned 삭제 승인 가능 |
| 앞 probe의 explicit empty pylock1.0 | packages=[]와 유효 header를 포함하고 환경을 비움 | Strict header/target/lock 및 frozen controls로 명시 empty를 확인. 누락 file/zero value와 구분 |

따라서 source control에는 입력 format/bytes와 --allow-empty-requirements의 실제 값도 포함한다.
Default skip의 receipt/selection을 명시 삭제에 재사용하지 못하도록 digest를 구분한다. CLI는
원래 flag를 같은 이름으로 제공하고 설명은 EN/KO를 완성한다. 아직 검증하지 않은 include/
constraint/hash continuation·range/options는 이 조사에서 지원한다고 선언하지 않는다.

기존 cli/cargo_update.go의 writeProjectUpdateReport는 private DTO와 project-update schema를
사용하며 Policy pointer가 선택적이다. UV sync는 이 presenter의 entry/check/Evidence 표현을
재사용하되 Cargo 전용 labels를 descriptor로 분리하고 sync intent를 정확히 표시한다. Blank
requirements 기본 skip은 명시 SKIPPED/NOT_ATTEMPTED·정책 미평가·entry0으로 설명하며, fake
empty-set ALLOW/Run/check/Evidence나 삭제 완료를 만들지 않는다. 읽기·판정/출력 위치는
C3e3 구현 전 static preflight와 backend composition의 실제 경계를 감사해 확정한다. 명시
empty 허용 경로만 기존 ProjectInspect/SetPolicy/staging/guard/rollback에 진입한다.

두 번째 검토: 빈 file 자체는 wipe 권한이 아니다. Native skip과 positive empty lock/명시 flag를
분리하고, source/flag의 변경·기존 cache receipt 재사용·unselected native owner의 증명·부분
실패 복원을 부정 회귀로 둔다. 현재 C3d의 새 source CI383이 끝나기 전 sync 제품 source는
수정하지 않으며 Go1.27.3 순서와 기존 단일-primary invariant를 유지한다.


## C3e1 착수 범위 / 정확한 함수와 두 번째 검토

C3d는 CI383의 same-tree/Required10SUCCESS로 닫혔다. C3e1만 새 제품 source 작업으로
시작하며 current item은 계속 M13-004 하나다. 이 단위의 IMPLEMENTED=NO / WIRED=NO /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO이며 MISSING은 complete format/독립 binding/회귀다.
Runtime producer·promotion·CLI wiring은 이후 C3e2/3의 MISSING으로 남긴다.

| 위치 / symbol | 현재 단위의 정확한 변경 |
| --- | --- |
| artifact/pypi/uv_lock.go ParseUVLock / BindWheels | strict TOML package/header/URL/hash/size/target 검증과 independent archive/metadata/Requires-Dist 대조를 private 공통 helper로 추출. 기존 public single-root API·root version/reachability/nonempty rejection과 tests를 유지 |
| 같은 package의 uv_project_lock.go 새 complete 소비자 | ParseUVProjectLock은 explicit packages=[]/strict header와 multi roots를 구분. BindProject는 existing PythonVenv InstallContext·frozen controls·authentic exact selected wheels를 기존 ProjectDependencySnapshot/NewDependencyFreeProjectSnapshot에 연결. 가짜 primary/새 Core graph 없음 |
| uv_lock.go / uv_installed_controls.go native selector | private packages/target-only selector를 공유. 기존 single-root empty refusal은 유지하고, parsed explicit-empty project에서만 exact empty controls/snapshot 허용. Null/omitted/extra controls는 미지원 |
| Complete snapshot digest | exact selected artifact identity/URL/integrity·target/Python·native lock 및 sorted full control digests를 bind. 다른 wheel alternative/flag/source/target의 approval을 같은 graph digest로 재사용하지 못함 |
| uv_project_lock_test.go | multi/disconnected roots·full dependency closure와 explicit empty 정상; missing/null/unknown/future/target/URL/hash/digest/wheel/control/constraint·duplicate·substitution/budget·mutable input 거부; 기존 single-root/root-version/empty 회귀 보존 |

두 번째 검토: parsed project snapshot은 Policy나 삭제 권한이 아니다. Default blank skip은
이 positive empty constructor 경로에 들어오지 않는다. Complete source/flag 일치, trusted
native runtime/원본 guard·entry/set Policy·staging·owned publication은 이후 실제 소비자에서
함께 검증한다. 선택 wheel의 filename·archive·Simple metadata와 constraints/graph budget을
검증한 공통 본문을 재사용하며 complete 목적 때문에 기존 검사를 생략하지 않는다.
새 external dependency/interface/config/CI job·미래 milestone scaffold는 만들지 않는다.

## C3e1 구현 경계 / 로컬 검증

`ParseUVProjectLock` / `UVProjectLock.BindProject`는 별도 primary 없이 기존 ProjectDependencySnapshot에
결합한다. 기존 UVLock은 nonempty/single-primary/connected graph 규칙을 유지한다.
Strict native TOML·wheel alternatives·installed METADATA/WHEEL·전체 archive SHA/RECORD·
독립 release metadata·Requires-Dist constraints의 본문을 공유하며, complete 소비자는
모든 선택과 control/실제 wheel URL·integrity/target의 digest를 고정한다. Native output의
누락/null census와 명시 empty census를 구분한다. Empty도 target·한도의 유효성·동일
native lock control을 확인하며, 양수 한도 검증을 InspectWheel 호출 전에 공통 적용한다.

Canonical format/Quick/security/docs/workflow와 기존 fuzz profile의 두 parser 실행이
PASS다. 관련 artifact/Application/promotion 전체 Race129.553s와 sandbox 선택 회귀10.937s도
PASS이며 [로컬 영수증](./p1e-uv-project-lock-local-20261023.json)에 exact source/check를
연결했다. 정확한 source의 원격 Required/native macOS/Linux 검증은 남는다. 이 단위는
IMPLEMENTED=YES / WIRED=YES(adapter 내부만; production sync는 NO) /
QUALIFIED=NO(새 원격 검증 대기) / ACCEPTANCE_CLOSED=NO다.
이 단위의 MISSING은 같은 source CI다. 이후 C3e2 native complete producer/reader, C3e3
guarded publication/CLI와 P1 actual whole qualification도 남는다. 파서 테스트 성공은 실제 삭제·검역
완료가 아니며, runtime pin·Policy·삭제 authority나 지원 CLI를 이 변경에서 추가하지 않는다.

두 번째 검토는 독립 roots/constraint·missing/self dependency, complete archive 변조,
명시 empty/누락/null/extra census, unknown/future/direct source·budget, 단일 root empty
거부와 control/target/실제 wheel 대안 바꿔치기를 포함한다. 기존 구현을 대체하지 않고
공통 검증을 재사용했다. 완료된 시험의 payload는 남기지 않으며, source/SDK/cache는
아직 필요한 다음 P1 검증 입력으로 유지한다.

## C3e2a — frozen source/intent codec의 로컬 경계

사전 지도6fc0629 뒤 `uv_sync_input.go`의 ParseUVSyncInput/Controls/ParseUVSyncControls와
ValidateSelection을 구현했다. 위 bounded plain exact pins/strict pylock grammar와 원본
UTF-8 bytes·명시 flag를 별도 present controls로 고정한다. 기본 blank skip은 positive
selection을 거부하고 explicit empty와 다른 intent digest를 가진다. Native package
name/version이 원본 목록과 다르거나 pylock 원본 bytes·Python target이 바뀌면 거부한다.
PEP440 trailing-zero equality는 기존 UpgradeRequirement.AcceptsVersion을 재사용한다.
이 codec의 목록 일치는 독립 전체 archive/metadata/dependency closure 검증이 아니며,
root-only 입력의 source match 뒤에도 기존 BindProject가 missing dependency를 거부한다.

이 codec의 IMPLEMENTED=YES / WIRED=YES(adapter 내부; native producer는 NO) /
QUALIFIED=NO(로컬만) / ACCEPTANCE_CLOSED=NO다. Canonical6와 전체 artifact Race5.270s가
PASS이며 [로컬 영수증](./p1e-uv-sync-source-local-20261023.json)이 source/check를 소유한다.
현재 CI384는 앞 C3e1 source12d9a50의 검증이며 이 후속 codec 검증으로 전용하지 않는다.
이 단위의 MISSING은 새 source 원격 검증이고, C3e2 native lifecycle/reader와 C3e3
actual guarded sync/CLI 및 whole P1 qualification은 별도로 남는다.

## C3e2b 구현 직전 검토 — 기존 session body 유지

Native owner는 private `withUVNativeSession`과 `uvNativeSession`(container/runtime/
platform/output)을 추출한다. 기존 single selection은 같은 본문을
`materializeUVSelection`에서 실행하고, complete는 `materializeUVSyncSelection`에서
위 fixed source/intent만 실행한다. 새 공통 interface/engine을 만들지 않는다. 기존
`withPythonResolverSession` wrapper는 source-bound private session에 기존 callback을
연결하며 실패·정리 오류에서 resolution을 비우는 동작을 유지한다. Whole archive
reader의 workspace/metadata/stream/cleanup도 두 실제 소비자가 공유한다.

`UVResolver.ResolveProjectLockUpdate`는 기존 port의 empty selector와 PythonVenv
context·source/intent/runtime/선택 전 absent lock controls를 먼저 검증한다. 실제
고정 tool/probe·관측/종료가 성공한 결과만 독립 reader와 BindProject로 보낸다.
`NewUVSyncOriginalControls`는 compile-time runtime/control을 준비하는 함수이며
runtime 인증이나 삭제 승인이 아니다. 실제 producer가 같은 고정 tuple을 검증한다.

Empty transport의 추가 경계: 기존 collector는 metadata/WHEEL 이외 private payload를
discard한다. Complete empty에서는 site 디렉터리와 유효 explicit-empty native lock,
명시 empty census를 요구하며 site의 임의 파일/하위 디렉터리를 허용하지 않는다.
기존 single collector의 control0 거부는 유지한다. 새 callback은 기존 named output에
fresh site 디렉터리를 trusted tar 입력으로 준비할 수 있지만, 디렉터리 생성 자체를
native 실행 완료/빈 선택/삭제 authority로 쓰지 않는다. Pylock 원본도 같은 bounded
tar로 고정 경로에 공급하며 별도 Host 실행·임의 guest script 권한을 추가하지 않는다.

검증은 기존 single/installed-aware UV·pip/PyTorch 회귀와 partial cleanup/첫 cause 보존,
complete source/flag/runtime/target/native lock 바꿔치기·empty payload 거부를 포함한다.
실제 native public 두 독립 root·complete dependency closure·missing dependency 거부·
explicit empty/skip 분기를 기존 고정 실행·network-policy caller·관찰기에서 검증한다.
이 단계도 guard/staging/사용자 반입 성공이나 P1 전체 acceptance로 확대하지 않는다.

### Full Race에서 드러난 기존 fixture 경합 — 수정 전 확인

C3e2b 첫 canonical5/기존 fuzz는 PASS지만 full sandbox/artifact Race27.809s는
기존 admission·SharedObserver 테스트7개에서 DATA RACE로 실패했다. Artifact는 PASS다.
변경 전344af5e의 정확한 전체 source를 별도 임시 디렉터리에 추출하고 같은 SDK/flags로
해당7개만 대조해8.013s/exit1·6개 경합을 재현했다. 나머지1개는 대조에서 경합이 보고되지
않았으며 재실행으로 성공/실패를 강제하지 않는다. 해당 테스트·admission/observer/
supervisor owner5파일은 현재 후보와 baseline bytes가 같다. 새 UV 변경의 원인으로
단정하지 않으며 원본 first failure와 별도 baseline을 보존한다.

정확한 교정은 테스트 fixture에 한정한다. `direct_exec_admission_test.go`의 다섯 테스트는
Unix peer goroutine의 일반 int counter를 caller가 동기화 없이 읽는다. 기존 sync/atomic의
Int32 Add/Load로 같은 횟수와 fail-closed 의미를 검사한다. `shared_observer_test.go`의
두 live receiver 테스트는 goroutine 시작 뒤 diagnostic pointer/bytes.Buffer를 직접
대입한다. 기존 supervisor setter와 같은 observer.mu 아래 sink를 설정한다. 실제 receive의
두 진단 writer는 이미 같은 mutex 아래 쓰고 done을 닫기 전에 완료하므로 production
writer/ledger·완료 순서·budget/timeout을 바꾸지 않는다. 단독 aggregation fixture는 concurrent
receiver가 없으므로 변경하지 않는다. 같은7개 Race와 full sandbox/artifact Race, canonical
profile을 확인하기 전 해결로 기록하지 않는다. 이 fixture 교정은 별도 커밋으로 분리한다.

동일 two-file 교정 뒤 baseline 전체 sandbox/artifact Race20.993s와 새 후보 전체 Race21.253s는
모두 PASS다. Canonical format/Quick42.492s/security/docs/workflow도 PASS다. 원본7개 실패·
baseline6개 재현을 삭제하거나 단순 재시도 성공으로 바꾸지 않았다.
[Fixture-only 영수증](./p1e-uv-observer-fixture-race-20261023.json)이 source/check를 소유한다.
대조용 source2020파일/33,549,068B와 archive는 정리했고 active source/SDK/cache만 유지한다.

## C3e1 원격 종료 / C3e2b 현재 경계

[CI384](./p1e-uv-project-lock-ci384-20261023.json)는 source12d9a50과 같은 전체 checkout
tree32e250의 Required 포함10SUCCESS다. Linux51·native UV 기존 소비자·CLI209.14s·
CPU full289.19s와 native macOS/Go1.26.10·취약점, helper5wait0/final confirmed cleanup을
확인했다. C3e1의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED는 해당 adapter 범위에서
YES이고 MISSING은 없다. 이 결과에 후속 codec0502a95나 C3e2b source는 포함되지 않는다.

C3e2b는 위 기존 native/source session·archive reader의 본문을 재사용해 existing
ProjectLockUpdateResolver에 complete 선택을 연결했다. Runtime control은 generated 전체
RuntimeLockSHA256과 UV resolver tuple을 bind하며 실제 producer는 고정 tool/runtime를
검증한다. Complete empty collector는 유효 explicit-empty lock·fresh site·no payload와
명시 empty census를 확인하고 기존 single empty refusal은 유지한다. Source intent와
전체 archive/독립 metadata/Requires-Dist를 확인한 뒤에만 primary-free snapshot을 만든다.

이 resolver 단위의 IMPLEMENTED=YES / WIRED=YES(existing resolver port; CLI는 NO) /
QUALIFIED=NO(로컬만) / ACCEPTANCE_CLOSED=NO다. Canonical5/fuzz·fixture 교정 뒤 full
sandbox/artifact Race가 PASS이며 새 same-source actual qualification이 MISSING이다.
[Native source 로컬 영수증](./p1e-uv-native-source-local-20261023.json)이 exact source·
체크와 실제 native 미실행 경계를 소유한다.
기존 필수 public UV integration에 two roots·전체 요청 closure·누락 dependency 거부·
명시 empty requirements/pylock 다섯 사례를 연결했고 실제 실행은 새 후보 검증에서 확인한다.
실제 source/target guard·기존 모든 native owner proof·complete cache/transaction·SKIP 보고·
CLI는 C3e3이며 P1 전체와 Go1.27.3 순서는 유지한다.

## CI385 최초 실패 / fixture CWD 교정 전 지도

[실제 CI385](./p1e-uv-native-source-ci385-20261023.json)는 source6318a4c/same-tree의
8SUCCESS/2FAIL다. 기존 public idna1node와 requests5node·4edge는 완료했지만 추가 helper의
`../artifact/pypi/testdata/uv-0.13.4-requests.pylock.toml` 읽기가 ENOENT였다. Precompiled
test binary는 caller의 repository cwd를 유지한다. 새 complete 다섯 사례는 실행 전이며
UV CLI/CPU full도 미도달이다. Helper2 wait0/final original1 RESTORED는 확인했다.
실제 native complete의 제품 실패/성공으로 해석하지 않는다. CI L04의 기존 Cargo
package-subshell 해법을 이번 신규 consumer에 적용하지 않은 작성자의 실행 위치 가정 오류다.

수정 전 지도: shared input 조립을 `uvPublicCompleteSourceCases`로 추출해 실제 native
helper와 data-only `TestUVPublicCompleteSourceInputs`가 같은 코드를 사용한다. 이 코드를
고정 compiler로 만든 한 binary의 repository cwd FAIL→sandbox package cwd PASS로
대조한다. Fixture bytes와 고정 tool/runtime·observer·native 명령·시간 제한은 바꾸지 않는다.
실제 workflow의 해당 public UV 호출을 기존 Cargo와 같은 package subshell로 감싼다.
`scripts/check/workflow.go`는 기존 active lifecycle run/command 판정을 공유하며 두 실제
fixture consumer의 정확한 package-subshell shape를 확인한다. Missing/commented/wrong/
외부 cd·중복 호출을 부정 회귀로 거부한다. 임의 Bash 실행 의미 전체를 증명한다고 하지 않는다.
Canonical profile과 같은 compiled data consumer/관련 tests를 확인한 뒤 fresh source CI에서
실제 five cases와 공유 소비자를 검증한다. 파일 복제·skip·expected hash 재생성이나 임의
Host 경로 탐색으로 입력을 찾아 성공시키지 않는다. C3e3 제품 source는 아직 미작성이다.

로컬 교정은 위 지도대로 완료했다. 같은 input constructor의 compiled consumer는
repository cwd에서 ENOENT/exit1, package cwd에서 다섯 입력 PASS/exit0이었다.
다만 operator 마지막 요약은 병렬 Quick이 쓰던 shared TMPDIR의 비어 있음을 잘못
요구해 exit1이었다. 삭제 전 영속화하지 않은 binary digest는 NOT_AVAILABLE이며
개별 로그·정리·후속 부재에서 좁은 관측만 복원했다. 전체 operator PASS로 바꾸거나
지나간 신원을 만들려고 다시 실행하지 않았다. 작은 원본은 유지하고 binary/workroot는
삭제했다. Workflow 두 실제 fixture consumer의 package scope 부정 회귀와 canonical
format/Quick47.213s/security5.900s/docs/workflow는 PASS다.
[CWD 교정 로컬 영수증](./p1e-uv-fixture-cwd-local-20261023.json)이 원본·source·범위를
소유한다. 이 교정의 로컬 실행 위치 검증만 끝났고, C3e2b의 QUALIFIED 및
ACCEPTANCE_CLOSED는 새 실제 native 다섯 사례와 공유 CI 검증 전까지 NO다.
