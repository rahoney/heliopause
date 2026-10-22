# P1 complete sync 반입 — C3e3 baseline과 수정 지도

기준선은 `7db4ba891ce68fd28a3ba84decafffd1fdd553d2`다. M13-004/P1 하나만
IN_PROGRESS다. 이 문서는 기존 [sync 지도](./p1e-uv-sync-baseline-20261022.md)의
C3e3 상세 owner를 소유한다. C3e2b의 실제 native 다섯 사례는 CI386 확인 전까지
미검증이며 이 감사는 새 반입 구현이나 전체 P1 완료를 뜻하지 않는다.

## Baseline audit

아래 YES는 명시한 기존 소비자 범위다. 새 complete sync의 qualification에 전용하지 않는다.

| Owner / 실제 symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| domain ProjectDependencySnapshot / NewDependencyFreeProjectSnapshot / NewProjectDependencyUpdateFromSnapshot | YES | YES | YES: 기존 Go/Cargo; UV parser CI384 | YES: 기존 범위 | 기존 primary-free 값을 재사용 |
| application ProjectInspectService / executeGuardedProjectUpdate, policy.M4.EvaluateProjectSet | YES | YES: Go/Cargo | YES: 기존 범위 | YES: 기존 범위 | UV request-scoped service/composition 연결 |
| promotion pypiVenvTransaction prepare/commit/rollback / pypi_venv_backup.go | YES | YES: pip/UV install/update | YES: 기존 소비자 CI384 | YES: 기존 merge 범위 | complete-only obsolete 계산과 explicit empty, same journal/rollback 검증 |
| pypiUpdateSession 원본 capture/verify, managedPyPIInventory/capturePyPINativeInventory | YES | YES: 요청 root upgrade | YES: 기존 소비자 | YES: 기존 범위 | root 없는 source guard의 공통 metadata snapshot 재사용 |
| pypiNativeEnrollment.selectOwnership / oldWheel / proveNativePythonWheelFootprint | YES | YES: selected graph의 이전 owner | YES: native UV CI383 | YES: 기존 범위 | 제외되어 삭제될 native distribution까지 전체 proof |
| LocalStaging.stageArtifact / projectCacheEvidence / seal·sync·no-replace helpers | YES | YES: 기존 staging 및 Go/Cargo cache | YES: 기존 범위 | YES: 기존 범위 | Python 전용 primary-free cache와 record 생성 |
| evidence.Generator / BuildVerifiedBundle / buildSBOM | YES | YES: nonempty requested-primary VerifiedSet | YES: 기존 pip/UV install/update | YES: 기존 범위 | complete project Manifest/SBOM 및 explicit zero; 기존 constructor는 유지 |
| PyPIPromotion.promoteHeldVenv / preparePyPIProject / validatedPyPIDestinations | YES | YES: nonempty VerifiedBundle | YES: 기존 소비자 | YES: 기존 범위 | project cache를 같은 offline materialization/검증 본문에 연결 |
| CLI packageClientDescriptors / bootstrap.pythonPackageClient / project-update-result presenter | YES | YES: 기존 명령 | YES: 기존 소비자 | YES: 기존 범위 | sync source/flag/preflight/SKIP·Python labels와 실제 binding |
| complete sync source guard→전체 검역→staging→삭제·반입→report | NO | NO | NO | NO | 아래 순서 전체 및 actual shared qualification |

Read: Architecture04, Contract09, M5 Contract14 §5, package-client UX09와 확장 계획20,
기존 P1e/C3d leaf, 위 실제 source/test 및 scripts/check/architecture.go.
Domain/Application/Policy engine을 다시 만들거나 기존 구현을 교체하지 않는다.

## 확인한 추가 gap

1. `pypi_venv_transaction.go:readState`는 version2 ownership file의 empty Files를
   거부한다. `prepare`의 empty 검사만 제거하면 다음 실행에서 스스로 만든 metadata를
   읽지 못한다. Global empty 허용이나 version2 의미 변경으로 고치지 않는다.
2. `persistLedger`는 ownership file publisher가 아니라 append-only recovery journal이다.
   실제 ownership publication은 `commit`의 backup-metadata→next-metadata→rename이며
   마지막 inode/body reconciliation도 같은 함수 안에 있다.
3. `selectOwnership`는 새 graph에 포함된 프로젝트만 이전 wheel로 증명한다.
   Complete sync가 지울 unselected native extra에는 이 범위가 부족하다.
4. 기존 `UpgradeInput`은 실제 요청 requirement를 필수로 가진다. 전체 metadata를 읽으려고
   임의 root/reference/빈 Artifact를 만들어 재사용하지 않는다. 두 inventory 함수가
   실제 사용하는 metadata 목록을 공통 인자로 좁히는 위치가 있다.
5. ProjectSnapshot에는 전체 Artifact/graph digest/control digest가 있지만 dependency edges나
   원본 control body가 없다. Static inspection Evidence도 Requires-Dist 본문이 아닌 bounded
   summary다. 이 값에 존재하지 않는 edges/filename/control body가 있다고 가정하지 않는다.
6. Architecture checker는 promotion→evidence import를 허용하지 않는다. SBOM 모델을 promotion에
   복제하거나 allowlist를 완화하지 않는다. 현재 consumer에 필요한 record builder는 bootstrap의
   명시적 주입 경계로 연결해야 한다.
7. 현재 packageClientByID는 client당 첫 install descriptor를 반환한다. 같은 client의 sync를
   추가하면서 ID만으로 leaf를 찾으면 install에 잘못 연결될 수 있다. 실제 command path/operation을
   같은 descriptor에서 식별해야 한다.
8. Python transaction의 기존 recovery는 rollback 불능 시 journal/원본을 보존하고 다음 mutation을
   차단하는 경계다. persistLedger/verifyBackupJournal과 interrupted-transaction 검사가 있지만
   journal을 읽어 자동 복원하는 Python recovery CLI/engine은 없다. Journal 존재를 복원 완료로
   표기하거나 본체 self-update의 별도 durable recovery와 합치지 않는다.

## 수정 순서와 경계

### A. 원본 source·설치 상태 guard

- `pypi_update.go:prepareInstallUpdate/verify`의 metadata directory census·retired-empty tree·
  bounded METADATA snapshot과 reverify를 private 공통 helper로 추출한다. 기존 update session은
  같은 helper를 호출하며 기존 requirement/selection/오류 의미를 유지한다.
- `pypi_native_inventory.go`의 managed/capture 본문은 실제 InstalledMetadataFile 목록을 받는
  private helper를 공유한다. 기존 UpgradeInput wrapper와 모든 기존 검사를 유지한다.
  원본 metadata1024/4MiB와 새 selected graph64 한도를 섞지 않는다.
- 새 `uv_sync` promoter/guard는 실제 canonical source path와 bytes, format/allow flag 및 target을
  request scope에 고정하고 기존 UV venv discovery·held transaction을 사용한다. Source는 regular,
  single-link/no-symlink이고 target/trusted roots·변경할 metadata와 alias/overlap하지 않아야 한다.
  Source body는 기존 ParseUVSyncInput 한도를 따른다. Resolve·inspect·staging 뒤와 mutation 직전에
  inode/mode/size/digest·parent identity를 확인한다. Frozen source를 다시 읽어 live resolve하지 않는다.
- `pypi_native_ownership.go:selectOwnership`의 실제 old-wheel/proof 본문을 private 프로젝트 목록
  helper로 공유한다. 기존 selected-graph wrapper는 그대로 유지하고 sync만 전체 frozen native
  inventory의 실제 이름을 정렬해 증명한다. 실제 이전 wheel reference는 합법적인 개별 old proof이며
  composite sync의 primary를 만들지 않는다. 전체 proof 실패/미지원 형식은 삭제 전에 거부한다.
- Managed ledger가 없다는 이유만으로 user file을 소유하지 않는다. Native distribution이 있으면
  explicit --adopt와 전체 old proof가 필요하다. 실제 distribution이 없는 target도 사용자 파일을
  보존하며 새 destination 충돌을 거부한다. Native shared RECORD claim·generated scripts·compiled
  caches·local/direct/VCS/build 등의 기존 capability/refusal을 유지한다.
- Native backend의 qualified 네 control protocol은 변경하지 않는다. 필요 old-proof control은
  신뢰된 promoter 측 selection wrapper가 자신의 frozen proof와 함께 원본/선택 양쪽에 bind하고
  snapshot digest를 재구성한다. Wrapper는 base native controls를 정확히 검증하고 extra proof를
  guest authority로 넘기지 않는다. Full-control replay·변조·다른 target을 commit이 거부해야 한다.

### A1 구현 전 재검토 — 원본 metadata 공통화만 먼저

첫 source 단위는 A 전체가 아니라 다음 공통화다. C3e2b 실제 qualification 뒤 착수한다.
`pypiUpdateSession`의 directories/metadata/emptyMetadata는 실제로 pypi_update.go의 capture와
verify 두 본문에서만 사용된다. 새 private `pypiInstalledSnapshot`과 capture/verify helper로
옮기고 session은 original snapshot을 보유한다. 기존 Begin의 adoption 검사·defer Close와
NewUpgradeInput의 실제 requirement·native inventory·Resolve/Promote 순서는 유지한다.
capture 함수는 transaction과 실제 metadata 목록만 다루고 요청 reference/Policy/target mutation을
생성하지 않는다. metadata Body는 snapshot 내부에 소유하고 외부 반환은 복사한다.

Inventory의 실제 metadata 소비는 managedPyPIInventory와 capturePyPINativeInventory의
MetadataFiles() 두 곳이다. 기존 UpgradeInput wrapper가 private file-list helper를 부르게 하며
동일 alias/owner/RECORD hash·resource·native refusal을 유지한다. Count 검사는 기존 공개
`artifactpypi.UpgradeInputDistributionLimit`을 사용한다. 이 단위에서는 complete deletion,
source-file guard/control wrapper·old proof 전체화·record/CLI를 구현하지 않는다.

기존 TestPyPIUpgradeGuardFreezesMetadataAndOwnershipBeforeSelection의 drift/cancel/lock,
retired directory 두 tests, TestNativePyPIInventoryOnlyFreezesClaimsAndDetectsOriginalDrift와
false path/link/hash/ambiguous claim, UV update의 exact publication/rollback을 재사용한다.
새 의미 검증은 root 없는 실제 metadata snapshot의 empty/65개 수집과 변경 감지에 한정한다.
65개는 graph64와 원본1024를 혼동하지 않는 경계다. Helpers에 성공을 미리 가정한 fake root나
삭제 권한을 넣지 않는다. Canonical5와 영향받은 promotion/Bootstrap/Application Race를 확인하고
소스·raw·기존 공개 동작과의 경계를 중간 커밋에 연결한다.

두 번째 검토에서 원본 capture 오류의 기존 wrapper code와 primary cause, Close 책임을 보존하는
위치를 확인했다. 위 세 전체 통합 미결정점은 A1 공통화의 입력이 아니며 A 나머지/B/C/D source
착수 전 각 해당 지도를 더 좁힌다. 기존 기능 재구현이나 새 미래 Port를 추가하지 않는다.

### A2 재검토 — 전체 old proof와 native 네 control 보존

후속 guarded sync preparation은 한 요청 안에서 Begin/selection/Verify/Commit/Close를 연결한다.
원본 전체 inventory를 고정하고 --adopt가 필요한 실제 native distribution이 있으면 새 선택에서
빠질 프로젝트도 포함해 기존 old-wheel proof를 수행한다. 성공한 전체 proof의 destinations만
transaction의 in-memory current ownership에 추가하며 on-disk metadata는 ALLOW/commit 전에
바꾸지 않는다. 기존 cache namespace absence·file identity·owner overlap 검사를 공유한다.

원본 controls는 기존 NewUVSyncOriginalControls의 네 항목과, native enrollment가 있을 때만
기존 pypi-update-native-ownership.json의 정확한 proof 한 항목이다. Guard가 실제 생성한 proof와
다른 body·중복·unknown control을 거부한다. Selection wrapper는 정확한 원본 네 항목만 native
resolver에 전달하고 그 actual update의 원본/선택 context와 immutable source/intent/runtime,
present selected lock을 검증한 뒤 같은 proof를 양쪽 controls에 더한다. 새 graph digest는 base
graph digest와 정렬된 전체 selected control digest를 domain-separated canonical record로 bind한다.
새 snapshot은 같은 실제 dependencies/context/source와 dependency-free 상태를 사용하고
NewProjectDependencyUpdateFromSnapshot으로 감싼다. Native backend의 네-control protocol과
기존 single/update 경로는 바꾸지 않는다.

Managed old state와 전체 이전 파일의 identity는 held transaction/installed snapshot이 별도로
소유한다. Source intent나 proof digest 하나를 삭제 권한으로 취급하지 않는다. Guard.Commit은
이 요청의 실제 update/전체 controls·staged approved set·원본 guard 불변을 요구하고, complete
update가 없는 CommitSnapshot 우회·repeated selection/commit·closed guard·다른 target을 거부한다.
Proof 실패/취소·누락된 unselected native proof와 forged control/update를 mutation 전 부정 회귀로
검증한다. 이는 A1 이후 해당 source 단위의 지도이며 전체 통합 검증 완료는 아니다.

### A2a 구현 전 감사 — rootless inventory의 metadata identity 전제

A1의 기존 production wrapper는 NewUpgradeInput을 먼저 거쳐 directory/METADATA identity와
중복 원본을 검증한다. 이후 FromMetadata helper는 directory identity와 HAA owner를 확인하지만,
Body Name/Version 대조와 원본 이름 중복 검사를 독립 수행하지 않는다. Native collector는 마지막에
inventory.projects[name]을 대입한다. 이 사실을 현재 공개 업데이트의 취약점이나 A1 회귀로
표기하지 않는다. 아직 없는 rootless whole-inventory guard가 같은 입력 전제를 충족해야 한다.

변경 없는 artifact/pypi.NewUpgradeInput parser의 기존 범위는 IMPLEMENTED=YES/WIRED=YES/
QUALIFIED=YES/ACCEPTANCE_CLOSED=YES다. A1 공통화의 같은-source qualification은 CI387에서
별도로 확인한다. 독립 rootless inventory 입력 검증은 NO/NO/NO/NO이며
MISSING은 actual metadata identity/원본 이름 중복 거부와 기존 소비자 검증이다. Baseline은36c6253,
정확한 수정 위치는 pypi_native_inventory.go:managedPyPIInventoryFromMetadata다.

새 source 전에 rootless file-list helper의 body identity mismatch/ambiguous headers와 같은 이름의
두 버전·반복 항목을 부정 입력으로 직접 확인한다. 기존 NewUpgradeInput wrapper는 수정하지
않는다. Shared helper에 기존 parseInstalledMetadata를 재사용해 Body의 normalized Name/Version을
directory와 대조하고, managed/native 구분 전에 같은 normalized project를 한 번만 허용한다.
Source metadata는 여전히 unapproved claims이며 이 검사로 파일 소유권을 만들지 않는다.
Native old-wheel/whole RECORD/파일 identity·resource·cache-absence proof는 별도로 필수다.

검증은 same negative before/after와 기존 원본0/65/managed/native inventory/UV·pip update 및
canonical5·promotion/Bootstrap/Application Race다. A1 CI387은 다른 source이므로 새 검증에
전용하지 않고 실행 중인 CI를 새 push로 취소하지 않는다. 두 번째 검토에서 기존 wrapper의
metadata semantics와 whole inventory의 중복 map membership은 다른 책임임을 대조했다.
기존 artifact RFC822/dependency parser를 약화하거나 fake UpgradeRequirement/graph를 만들지 않는다.

### A2b 구현 전 감사 — primary 없는 전체 original proof bridge

기존 pypiNativeEnrollment.selectOwnership의 oldWheel/whole footprint·UV cache namespace·control
생성 본문은 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES(기존 selected graph 범위,
CI386)이며 재구현하지 않는다. Complete original inventory를 직접 증명하는 호출은 NO/NO/NO/NO다.
MISSING은 동일 본문의 primary-free subject 연결과 새 선택에서 제외된 원본의 실제 proof coverage다.

수정 위치는 pypi_native_ownership.go:selectOwnership 한 owner다. 기존 함수는 실제 graph node의
project/source subject를 만드는 wrapper로 유지하고 native lookup·archive hash/filename/runtime·
whole RECORD/footprint·cache absences·bounded proof JSON 본문을 private subject helper로 공유한다.
새 selectAllOwnership은 A2a로 검증한 실제 native inventory의 전체 project를 정렬해 같은 helper를
호출한다. Composite ArtifactReference/primary graph를 만들지 않는다. oldWheel의 개별 실제 이전
wheel lookup은 그대로 필요하다. 기존5m proof 한도·resource/format·control schema/cleanup을 유지한다.

기존 synthetic wheel writer는 default wrapper를 유지한 named fixture helper로 본문을 공유한다.
기존 이름·version/data·ZIP member 순서/RECORD는 유지하며 실제 두 개의 서로 다른 synthetic
archive와 native UV 설치 원본을 구성한다. Fake ports는 고정 archive를 전달할 뿐이며 public/native
runtime qualification이라고 주장하지 않는다. 기존 selected graph 한 원본만 proof 하는 정상
동작과, primary 없는 전체 inventory 두 원본 proof를 같은 fixture에서 비교한다. 제외된 두 번째
archive의 손상이 전체 proof를 거부하고 HAA ownership·target publication을 만들지 않아야 한다.
두 원본의 cache namespace·각 intake scope cleanup과 기존 pip/UV proof·guard 회귀도 확인한다.

같은 기존 proof 본문 재사용·subject source/순서·ctx 오류와 optional proof control을 두 번째로
검토했다. 이 method만으로 actual sync guard가 WIRED되는 것은 아니다. Source file freeze·held
original reverify·complete deletion/cache/Policy/CLI는 후속이다. Canonical5와 관련 전체 Race를
확인하고 실행 중인 CI387 source와 새 후보를 구분해 기록한다.

### B. Complete-only transaction

- 기존 `prepare/commit` 입구는 merge와 empty refusal을 유지한다. 새 private complete 입구만
  검역된 전체 desired destinations와 proven original ownership을 사용한다. 원하는 경로에 없는
  proven old file을 obsolete로 계산하고 unmanaged/user data는 제외한다. Shared owners의 일부를
  제거·교체할 때 최종 모든 owner·content가 설명되는지 검증한다.
- 기존 file backup→publish→metadata→sync→reconcile→cleanup 본문과 journal/rollback을 공유한다.
  Empty가 아닌 complete commit도 exact desired set이며 unselected old owner를 남기지 않는다.
- Explicit approved empty에서는 기존 ownership metadata를 backup하고 빈 version2 file을 새로
  쓰지 않는 경로를 검증한다. 기존 metadata의 실제 absence와 모든 proven obsolete file의 absence를
  reconcile한다. metadataBacked=true/metadataPublished=false는 이미 rollback/removeBackup이
  다루는 상태다. 성공 뒤 기존 readState의 absence 의미를 사용하고 malformed empty v2는 계속 거부한다.
  실패 시 기존 metadata/owned files를 되돌리며 불명확한 상태는 recovery를 보존한다.
- `commit` 마지막 publishedMetadata inode/body 검사와 directory sync는 empty 경로에서 별도로
  absence를 확인해야 한다. 검사를 단순 생략하거나 early-return하여 journal/cleanup을 우회하지 않는다.

### B1 구현 전 감사 — complete-only file transaction

Baseline1cf5781의 pypiVenvTransaction prepare/commit/rollback/journal·renameNoReplace는
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES(기존 merge 소비자/CI387)다. 기존 prepare의
empty refusal과 선택 owner만 교체·나머지 owner 보존은 변경하지 않는다. Complete-only retirement/
explicit empty는 NO/NO/NO/NO이며 MISSING은 아래 private 입구·metadata 부재 reconciliation과
실패/기존 소비자 검증이다. 새 engine/ledger version/recovery CLI를 만들지 않는다.

정확한 source는 pypi_venv_transaction.go의 prepare574/commit796이다. 기존 prepare/commit은
legacy wrapper로 두고 private shared selection body에 complete mode를 전달한다. 새 private
commitComplete는 실제 complete snapshot의 dependency-free 의미와 destination empty 여부가
일치할 때만 empty를 허용한다. 이 method의 직접 호출은 controller 내부 계약이며 실제
검역·set Policy·original/source guard·approved cache는 후속 Guard.Commit에서 별도로 필수다.

Common prepare는 기존 output hash/size/inode/parent·unmanaged collision/owner validity를
그대로 사용한다. Complete mode에서만 모든 proven current owner가 교체·제거 대상이다.
Desired에 없는 current key를 obsolete로 계산하고, desired에 있으면 같은 scheme의 새 output을
사용한다. Managed shared owner의 일부만 새 set에 남아도 전체 intent에서 old owners 모두가
설명되어야 하며 새 desired의 exact owners/content만 남긴다. Legacy partial-shared refusal은
유지한다. Native shared RECORD refusal·unknown/user file은 이 경로로 소유하지 않는다.

Common commit의 backup/file publish/journal/rollback/cleanup을 공유한다. Explicit complete empty는
metadata를 backup하되 next-metadata를 생성·publish하지 않고, 마지막에 실제 metadata absence를
확인한다. Unknown stat/read error는 absence가 아니다. Empty version2 ownership file을 쓰거나
readState의 기존 empty-v2 refusal을 완화하지 않는다. Empty metadata branch에서도 sync·reconcile·
obsolete absence·retained file/inode 검증·backup cleanup을 우회하지 않는다. metadata가 필요 없는
경로에서 .heliopause directory를 새로 만들지 않으며 기존 empty directories/user files는 보존한다.
rename은 기존 no-replace confinement을 사용한다. Backup 뒤 외부 metadata가 나타나면 이를
덮어쓰거나 삭제하지 않고, rollback 불능 시 journal/원본/guard를 보존하고 다음 mutation을 거부한다.

새 단위 검증은 complete nonempty에서 unselected old files 삭제·새 exact owners와 user/cfg 보존,
명시 empty에서 metadata 부재/readState 정상/다음 transaction 재진입, 처음부터 ownership 없는
empty target에 controller directory 비생성, legacy empty 및 불일치 empty intent 거부다. Managed
shared owner subset의 complete 처리와 legacy refusal을 구분한다. Metadata/sync/reconcile 각 fault의
old bytes/inode/metadata 복원, 외부 metadata 경쟁 시 no-overwrite·primary error/retained recovery와
다음 mutation 거부를 확인한다. 기존 venv/PyPI/UV/namespace/rollback tests와 canonical5·관련 전체
Race를 실행한다. 두 번째 검토에서 backup-metadata/publishedMetadata 필드와 최종 body read,
renameNoReplace·removeBackup·Close 책임을 실제 source와 대조했다. 실제 complete CLI/Policy/
source guard가 연결되기 전 이 private transaction 기능만으로 P1 지원을 선언하지 않는다.

### C. 정확한 Python cache·Manifest/SBOM·offline 반입

- Python 전용 ProjectCacheStaging adapter에서 전체 ALLOW와 every required recorded Evidence를
  다시 확인한다. LocalStaging.stageArtifact의 digest/size·exclusive copy를 재사용하고 controls,
  source intent/runtime·graph·target·실제 wheel filename·record digest를 bounded receipt에 고정한다.
  Intake의 filename record는 선택 URL/identity/hash와 대조하며 mutable 이름만 믿지 않는다.
- Whole dependency edges는 hash-verified approved wheel의 실제 Requires-Dist를 기존 marker/constraint
  helper로 정규화해 구한다. Static Evidence summary를 edge 원문으로 사용하지 않는다. Primary 없는
  Domain node ID를 사용하며 missing/foreign node·unsatisfied constraint를 거부한다.
- `evidence/manifest.go`의 deterministic serialization/SBOM 모델을 공유하는 현재 project builder를
  사용한다. 기존 BuildVerifiedBundle/nonempty constructor를 약화하지 않는다. 명시 dependency-free
  ProjectVerifiedSet만 zero components/edges를 기록할 수 있다. 실제 entry Run/Evidence 참조와
  set/entry Policy·source/control/runtime·archive identity/digest를 Manifest/SBOM에 연결한다.
- Record builder는 현재 adapter-local 계약으로 bootstrap에서 주입한다. Promotion의 evidence 직접
  import 또는 Core/Application의 PyPI 형식 branch를 추가하지 않는다. Cache load/commit에서도
  receipt·records·artifact census/rehash와 whole approval 일치를 확인한다.
- 기존 `preparePyPIProject/promoteHeldVenv`의 공통 wheel 준비·pinned offline pip/no-index/no-deps/
  require-hashes/no-compile·scheme relocation·validatedPyPIDestinations를 재사용한다. Source sync의
  목록을 Host UV/pip resolver에 넘기지 않는다. Empty에서는 가짜 wheel/materialization 없이 같은
  complete transaction을 사용한다. Staging은 Artifact를 실행하지 않는다.

### C1 재검토 — 실제 control 원문과 record builder 입력

StageProject가 받는 ProjectVerifiedSet에는 control body가 없으므로 digest에서 원문을 복원하지
않는다. Request-scoped sync preparation이 실제 Resolve 결과를 한 번 보유하고, Python cache에
읽기 전용 selected-update getter를 주입하는 위치를 사용한다. Getter는 frozen source/target을
재검증하고 cache는 승인 set의 context/source/graph/dependencies/control digests와 실제 update가
완전히 같은지 확인한다. 이 callback은 현재 cache/guard 소비자용 adapter-local 경계이며 새 Core
Port나 guest completion authority가 아니다. 다른 요청의 update·호출 전/반복 선택을 거부한다.

Record builder의 현재 필요한 입력은 ctx, 실제 OperationID, ProjectVerifiedSet, 실제 filename을
가진 []domain.LockedDependency, []domain.DependencyEdge, 전체 selected control 원문과 고정
runtime identity다. 반환은 manifest digest/document와 SBOM document다. 이 structural interface를
promotion이 소유하고 evidence.Generator의 project method를 bootstrap에서 주입한다. 두 package가
서로 import하지 않으며 evidence builder는 Domain 값만 읽는다. 이 단위의 source를 쓰기 전에
함수/record bounds와 실제 empty/nonempty 소비 tests를 확정한다.

OperationID는 positive sync 요청에서 한 번 생성하고 기존 NewProjectInspectService의
newOperationID collaborator와 같은 요청의 record builder에 전달한다. 실제 inspection operation과
기록이 같은 ID를 사용하며 ProjectVerifiedSet에 존재하지 않는 필드를 읽지 않는다. 기본 blank
skip에는 이 operation/Run/Evidence를 만들지 않는다. Cache namespace의 기존 무작위 RunID handle은
storage 이름이며 inspection Run 성공으로 표시하지 않는다.

Filename은 실제 selected native lock에서 resolved URL/integrity와 유일하게 일치하는 wheel을
선택하고 Intake filename record와 대조한다. Rehashed approved wheel의 metadata에서 기존
marker/constraint normalization을 공유해 edges를 얻는다. 필요 parser helper는 실제 cache 소비자와
함께 추가하며 독립 Simple 응답을 꾸며 만들어 기존 binder를 속이지 않는다. 기록용 node는
ProjectDependencyNodeID/실제 filename을 가진 개별 LockedDependency 값이며 primary graph로
조립하지 않는다. Manifest는 root 없는 project 형태와 dependency_free를 명시하고 기존 v1
requested-primary builder/schema는 그대로 보존한다. Empty에서도 실제 set Policy·controls·target·
runtime·operation을 기록하며 가짜 entry/Run을 생성하지 않는다.

Cache load/commit은 고정 receipt digest를 먼저 확인하고 actual approval/evidence/artifact census·
source/controls와 기록을 대조한다. 같은 request operation과 rehashed approved bytes로 record를
재생성해 canonical document와 비교하는 검증을 사용한다. Report의 Manifest/SBOM 참조는 이
검증된 결과에서만 가져오고 새 public operation-result schema를 만들지 않는다.

### D. Service·CLI·report와 실제 qualification

- 기존 executeGuardedProjectUpdate를 호출하는 sync use case를 추가한다. 원본 guard→단일 선택→
  전체 fresh entry inspection/Evidence/entry Policy→set Policy→staging→commit→Close 순서를 유지한다.
  Entry가 없더라도 explicit empty snapshot/set Policy/guard를 거친다.
- Descriptor는 실제 client+command operation/path로 조회하고 기존 install API는 install wrapper로
  유지한다. sync가 install validator/primary parser를 호출하지 않도록 한다. source/flags/target
  문법·중복 검증은 PreparePresentation의 static preflight에서 수행한다. Help/settings는 lazy다.
- 기본 blank requirements는 SKIPPED/NOT_ATTEMPTED, entry0/Policy 없음/Run·Evidence 없음/native와
  mutation 없음으로 보고한다. --allow-empty-requirements 또는 유효 explicit-empty pylock의
  positive empty와 구분한다. 오류를 blank/skip으로 해석하지 않는다.
- `cli/cargo_update.go`의 기존 project-update-result/v1 entry/check/Evidence/Policy 표현을 공유한다.
  Python package count·sync intent·explicit empty·검사 실패·반입 실패·실제 완료/SKIP을 구분한다.
  EN/KO 완전 지원, 다른 기존 locale의 reviewed label/English fallback과 전문 코드 원문을 유지한다.
  Presenter 실패를 rollback 성공으로 표현하지 않는다.
- Bootstrap은 실제 현재 UV descriptor·source/target·record builder·existing inspection/runtime를
  연결한다. Client ID/source pin/runtime version을 곳곳에 별도로 추가하지 않는다.

### D1 재검토 — static SKIP과 실제 guard의 시작 구분

초기 CLI는 `helox uv pip sync <source-file>` 한 파일과 --target, --adopt,
--allow-empty-requirements를 지원하는 범위로 좁힌다. Pylock filename 판정과 plain requirements의
허용 filename/flag 조합은 CLI source 착수 직전 고정 tool의 실제 grammar와 대조해 확정한다.
stdin/URL·여러 파일/include/range·다른 index/build를 지원한다고 광고하지 않는다.

PreparePresentation은 문법·필수 값·bounded source parse를 먼저 수행한다. Preflight의 파일
읽기는 transaction 원본 authority가 아니다. Positive 경로의 실제 guard가 같은 경로의 현재
파일을 다시 검증·고정한 시점부터 selection/inspection/staging/commit까지 불변을 요구한다.
Preflight와 guard를 같은 inode snapshot이라고 표기하지 않는다. Symlink/특수 파일·source/target
alias와 변조의 정확한 거부는 실제 guard 소비 tests로 확인한다.

기본 blank skip 때문에 runtime을 준비하지 않도록 descriptor의 현재 static action을 CLI의 공통
사전 실행 지점에서 처리하는 위치를 사용한다. Bootstrap은 help 처리 뒤, commandRuntime보다
먼저 이 hook을 한 번 호출한다. 실제 지원 blank를 다시 확인해 SKIP을 보고하며 positive이면
일반 guarded binding으로 진행한다. Source read/parse 오류는 SKIP이 아니다. 이 처리에 별도
SourceID·plugin·fake installer/inspection 또는 큰 source body의 annotation 저장을 만들지 않는다.
기존 install descriptors에는 해당 action이 없다. --help와 invalid/blank 입력이 native/helper/
mutation에 도달하지 않는 Bootstrap 회귀를 CLI source와 함께 검증한다.

## source 착수 전 남은 검토와 검증 계획

이 지도만으로 C3e3 전체가 source-ready라고 선언하지 않는다. 첫 구현 전에 다음을 확정한다.

1. 원본 guard의 target/source·old proof와 전체 selected controls를 bind하는 정확한 receipt 형태 및
   same-source replay 거부 검증. 기존 qualified native 네 control protocol은 유지한다.
2. Project record builder의 실제 입력/반환·operation/Run 추적과 sealed cache load의 검증 경계.
   기존 ProjectVerifiedSet에 없는 값을 있다고 가정하거나 fake operation/primary로 채우지 않는다.
3. CLI가 받을 source 개수/format 판정/flag 조합과 static preflight→request-scope frozen file 전달 방법.
   현재 codec은 단일 exact-pin requirements/strict pylock이며 여러 파일/include/range를 광고하지 않는다.

각 A/B/C/D 변경 단위 앞에 해당 검토를 끝내고 의미 있는 중간 커밋을 남긴다. Canonical
format→Quick/security/docs/workflow를 먼저 실행하고 관련 guard/transaction/cache/record/Application/
CLI·Bootstrap 테스트와 필요한 Race를 수행한다. 기존 single install/update·Go/Cargo/TF 소비자를
보존한다. 실제 qualification은 pinned native complete selection과 fresh inspection을 거쳐
multi-root+closure, version 변경·기존 extra 삭제, same-set 새 Evidence, default blank skip,
explicit empty 두 형식, 삭제할 native extra의 proof 거부/성공, source/target/control 변조,
부분 publish/metadata/sync/reconcile 실패의 rollback, rollback 불능 시 durable journal/원본 보존과
다음 mutation 거부, 취소/cleanup을 검증한다. 새 자동 Python recovery engine은 만들지 않는다.
Unknown query/정리 실패가 최초 오류를 덮지 않아야 한다. 같은 source Required/Linux/native macOS와
기존 필수 소비자·CPU full을 확인한 뒤에만 complete sync/P1 acceptance를 판정한다.

두 번째 검토에서 실제 owner를 다시 대조해 A의 source/metadata guard, B의 empty metadata
reconciliation, C의 Manifest/SBOM·edges 부재와 import boundary, D의 descriptor 중복 lookup 및
SKIP/positive-empty 차이를 확인했다. 위 세 미결정점은 구현 전 해소 대상으로 남긴다.
완료한 시험 binary/workroot/container/payload는 exact 정리하고 작은 원본만 보존한다.

## A1 공통화 구현 / 로컬 검증

Baseline4dedfd8 뒤 `pypi_installed_snapshot.go`로 기존 capture/verify와 metadata directory
helpers를 옮겼다. Session은 이 원본 snapshot을 보유하고 기존 adoption 검사·NewUpgradeInput의
실제 requirement·Resolve/Promote/Close를 그대로 사용한다. 두 inventory 함수는 기존 UpgradeInput
wrapper를 유지하면서 private file-list helper를 공유한다. Source guard/전체 native proof·complete
삭제·cache·record·CLI는 이 단위에 포함하지 않는다.

고정 formatter로 비교한 capture/verify 핵심 본문은 local files의 snapshot 보유·return type과
transaction 인자 치환 뒤 같다. Count는 같은1024 값의 UpgradeInputDistributionLimit을 공유한다.
새 rootless0/65 metadata 수집·불변 복사·원본 변경 거부·취소·소유권 비생성을 확인했고 기존
업데이트/retired tree/native RECORD/rollback 소비자는 canonical Quick111.420s를 통과했다.
Format/security/docs/workflow와 전체 promotion/Bootstrap/Application Race132.410s도 PASS다.
해당 Source/check/raw와 정확한 scope는 [로컬 영수증](./p1f-uv-original-snapshot-local-20261023.json)이
소유한다. 완료한 검사 뒤 owned Go TMPDIR은0이며 별도 native 자원은 준비하지 않았다.

A1의 IMPLEMENTED=YES/WIRED=YES(existing pip/UV 원본 guard), QUALIFIED=LOCAL ONLY,
ACCEPTANCE_CLOSED=NO이며 MISSING은 같은 source의 실제 shared CI다. CI386 결과를 이
후속 source에 전용하지 않는다. 다음은 검토한 A2 전체 원본 proof/source guard와 B/C/D의 나머지
gap이며, whole guarded sync/P1은 아직 미완료다.

## A1 실제 source qualification 종료 — CI387

[CI387](./p1f-uv-original-snapshot-ci387-20261023.json)은 source36c6253과 actual checkout6bdfb7b의
전체 treeb1cd59 일치/Required10SUCCESS다. 기존 Linux51·UV CLI194.13s·native installed
update23.87s·complete source 다섯 사례51.40s/CPU269.59s/native macOS/취약점과 helper5wait0/
original0 confirmed cleanup을 확인했다. A1의 네 상태는 해당 공통화 범위에서 YES이고 MISSING은
없다. 후속 A2a/b source, actual complete target 반입/CLI/P1 전체 및 새 CUDA full로 확대하지 않는다.

## A2a identity 전제 / 로컬 검증

사전 지도39e92ac 뒤 direct file-list helper의 다섯 부정 사례를 먼저 확인했다. Name/Version의
directory 불일치, 같은 프로젝트의 두 버전, 반복 metadata, 중복 Name header는 기존 helper에서
모두 받아들여져 focused Go test6.112s/exit1·다섯 FAIL였다. 이는 rootless 입력 전제의 관측이며
기존 공개 wrapper는 NewUpgradeInput으로 이 입력을 이미 거부한다.

Shared helper에서 기존 parseInstalledMetadata로 Body와 directory identity를 대조하고 같은
normalized project를 managed/native 분류 전에 한 번만 허용했다. 같은 test source digest에서
다섯 사례 모두4.930s/exit0·PASS다. 원래 실패를 보존했고 tests/허용 문법/기존 parser를 약화하지
않았다. Canonical format/Quick111.933s/security/docs/workflow와 전체 promotion/Bootstrap/
Application Race124.980s도 PASS다. [로컬 영수증](./p1f-uv-rootless-identity-local-20261023.json)이
exact source·before/after·checks와 실제 범위를 소유한다. 완료 operator는 삭제하고2,188B의
작은 재현 원본만 보존했다. 모든 local 검사 종료 뒤 owned Go TMPDIR0을 확인했다.

A2a IMPLEMENTED/WIRED는 shared inventory 범위에서 YES, QUALIFIED=LOCAL ONLY,
ACCEPTANCE_CLOSED=NO이며 같은 source의 shared CI가 MISSING이다. CI387은 앞 source36c6253의
검증이므로 이 변경에 전용하지 않는다. All-original authentic old-wheel proof·actual source/target
guard·complete publication/CLI 및 whole P1은 별도로 남으며 이 identity 검사로 소유권을 만들지 않는다.

## A2b proof bridge / 로컬 검증

기존 selectOwnership은 실제 선택 graph의 subject를 만드는 wrapper로 유지하고 기존 proof 핵심
본문을 selectOwnershipSubjects에 공유했다. 새 selectAllOwnership은 실제 native inventory 전체를
정렬해 같은 본문을 호출한다. 고정 formatter로 대조한 핵심 본문은 graph-node identity→subject
project/source 치환 뒤 같다. 새 wrapper의 합성 subject 목록은 새로운 Artifact/primary graph가 아니다.
개별 실제 old-wheel lookup·whole footprint/RECORD·cache absences·5m/format/cleanup은 유지한다.

기존 synthetic writer의 default 이름/데이터/member 순서를 유지한 named helper로 서로 다른 실제
archive 두 개를 구성했다. 같은 fixture에서 기존 선택 graph는 alpha 한 원본만 증명하고, 직접
전체 inventory는 alpha와 제외된 beta 둘을 증명한다. Beta archive 치환은 전체 proof control을
거부하며 두 원본의 cache namespace/declared hash·filename·files와 exact intake cleanup,
HAA ownership/metadata 비생성도 확인했다. Fake ports/data-only unit 범위이며 public/native
전체 원본 검증이라고 표현하지 않는다.

Canonical format/Quick124.481s/security/docs/workflow 및 전체 promotion/Bootstrap/Application
Race128.679s는 PASS다. [로컬 영수증](./p1f-uv-original-proof-local-20261023.json)이 source·body
대조·checks와 정확한 범위를 소유한다. 모든 local 검사 뒤 owned Go TMPDIR0이다. Shared selected
wrapper는 WIRED=YES지만 actual complete guard caller는 아직 NO다. 새 all-original bridge는
LOCAL ONLY/ACCEPTANCE_CLOSED=NO이며 fresh source/shared CI와 actual complete guard의
연결·전체 원본 native qualification이 MISSING이다. 다음은 source/target guard와 B/C/D다.


## B1 complete-only transaction / 로컬 검증

사전 지도a511744 뒤 기존 prepare/commit을 legacy wrapper로 보존하고 private selection mode와
commitComplete를 추가했다. Common journal/backup/publish/rollback/cleanup 본문을 공유하며
complete에서만 proven old owners 전체를 교체·제거한다. Explicit empty는 metadata backup 뒤
새 빈 v2 file을 쓰지 않고 실제 부재를 reconcile한다. 기존 readState/legacy empty refusal,
partial-shared refusal·unmanaged collision/no-replace·recovery 경계를 유지한다.

새 여섯 회귀 group은 제외 owner/script 정리·user/cfg 보존, empty metadata 부재/재진입,
already-empty controller directory 비생성, 두 empty intent 불일치, 세 fault의 원본 bytes/inode
복원, foreign metadata no-overwrite·primary cause/복구 보존/다음 mutation 거부와 whole shared-owner
교체를 확인했다. Canonical format/Quick151.891s/security/docs/workflow와 관련 세 package 전체
Race170.772s는 PASS다. [로컬 영수증](./p1f-uv-complete-transaction-local-20261023.json)이 exact
source/check/raw·범위를 소유한다. 모든 검사 종료 뒤 owned Go TMPDIR0이며 별도 native 자원은 없다.

B1 private transaction은 IMPLEMENTED=YES/legacy body WIRED=YES/QUALIFIED=LOCAL ONLY/
ACCEPTANCE_CLOSED=NO다. Actual complete guard caller는 NO이고 fresh shared source CI와
원본/source guard·검역/Policy·cache를 통한 실제 반입 연결이 MISSING이다. CI388의 앞 source1cf5781
결과를 이 새 transaction에 전용하지 않는다. 다음은 A2c 원본 source/target guard와 C/D다.


### A2c1 구현 전 감사 — source file을 held transaction에 고정

Baseline1344424에서 bounded single-link/no-symlink file snapshot·pypiFileSnapshot.matches,
transaction.freezeParents/checkBefore와 UVSyncInput의 immutable body/format/allow intent는 각각
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES(기존 소비자 범위)다. 실제 sync source
path를 이 held transaction에 연결하는 helper는 NO/NO/NO/NO다. MISSING은 아래 연결과
source/parent drift·alias/overlap 부정 회귀이며 기존 fingerprint/transaction engine을 대체하지 않는다.

첫 source 단위는 private promotion/uv_sync_source.go:freezeUVSyncSource다. 양의 sync만 다루고
실제 ctx/held transaction·canonical absolute source path·trusted root 목록·format/allow flag를 받는다.
source와 target/trusted roots의 lexical overlap을 거부하고 기존 bounded snapshot으로 모든 parent의
symlink 및 regular/single-link를 확인한다. freezeParents로 source ancestry를 같은 transaction의
identity map에 넣는다. Snapshot과 bounded body read의 digest/size·재검증을 대조한 뒤 기존
ParseUVSyncInput과 실제 pinned Python을 사용한다. 읽기 전에 미리 정한 source bytes를 믿거나
file name에서 format/권한을 추론하지 않는다. 기존 parser maxReportBytes와 같은 한도를 exported
UVSyncSourceByteLimit으로 이름만 공유하고 parser 허용 문법/한도를 변경하지 않는다.

성공하면 source snapshot을 transaction.before에 보관해 같은 checkBefore와 최종 retained-file
검사가 source bytes/inode/mode·parent identity도 확인하게 한다. Source는 target 밖이라 backup/
obsolete 대상이 아니고 syncDirectories는 기존대로 target 내부만 다룬다. 성공한 immutable input은
이후 native 네 control 생성에 사용한다. Default blank SKIP는 CLI preflight의 별도 경로이며 이
positive helper에서는 거부한다. 누락/parse/취소/불명확한 filesystem 오류를 SKIP로 바꾸지 않는다.
이 private helper는 원본 proof·Policy·selected update·approved cache나 삭제 authority가 아니다.
Actual Begin/selection wrapper/Commit은 A2c2와 C 연결 뒤에만 WIRED로 판정한다.

검증은 양의 requirements/explicit empty pylock·allow-empty의 실제 원본 body 고정과 외부 반환
변조 불가, 기존 complete commit 후 source/user data 보존, content/inode/mode/parent 변경 시
checkBefore 및 commit 이전 거부다. File/parent symlink·hardlink·target/trusted overlap·비정규/과대
파일·noncanonical path·invalid/blank source와 취소를 거부한다. Canonical5와 관련 promotion/
Bootstrap/Application 및 parser 회귀를 확인한다. Source를 guard 밖에서 다시 읽거나 기존
pypiUpdateSession의 requirement/ownership semantics를 변경하지 않는다. 두 번째 검토에서
transaction.before의 최종 retained-file 검사와 backup 대상 목록의 분리, 외부 ancestry sync 제외,
single-link snapshot의 bounded recheck·default blank refusal·한도 단일 owner를 확인했다.
