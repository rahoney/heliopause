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
부분 publish/metadata/sync/reconcile 실패의 rollback·durable recovery, 취소/cleanup을 검증한다.
Unknown query/정리 실패가 최초 오류를 덮지 않아야 한다. 같은 source Required/Linux/native macOS와
기존 필수 소비자·CPU full을 확인한 뒤에만 complete sync/P1 acceptance를 판정한다.

두 번째 검토에서 실제 owner를 다시 대조해 A의 source/metadata guard, B의 empty metadata
reconciliation, C의 Manifest/SBOM·edges 부재와 import boundary, D의 descriptor 중복 lookup 및
SKIP/positive-empty 차이를 확인했다. 위 세 미결정점은 구현 전 해소 대상으로 남긴다.
완료한 시험 binary/workroot/container/payload는 exact 정리하고 작은 원본만 보존한다.
