# P2c5-B5a — 승인된 PNPM layout의 비실행 staging

Baseline7e966e7/tree d864b870, M13-004/P2 하나만 IN_PROGRESS다. B4c named local은
완료했고 CI405/run40112425913은 실행 중이다. 첫 확인은2026-10-25 02:32:37 KST다.
현재 후보를 재push하지 않고 결과와 독립적인 이 owner를 진행한다.

Read: Queue·B4/B4c leaf, client 설계§4·확장 계획§3/4/5, Architecture02/03,
Domain09/13, artifact L15/L19/L20·sandbox L08/L12/L18·evidence L11/L16.

## 1. Source 작성 전 repository baseline audit

| 책임 / 기존 symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 완전 static/실제 instance 관찰·기록·Policy — PNPMProjectInspector / ProjectInspectService | YES | YES(actual) | YES(named local), remote pending | NO(P2) | 승인 뒤 exact layout 인계 |
| current source bytes 재확인 — readPNPMIntakeArchive | YES | YES(native introduction) | YES(named local) | NO(P2) | 같은 reader의 staging 재사용 |
| 실제 stored Evidence 재조회 — projectCacheEvidence / local.Store.ReadReference | YES | YES(Go/Cargo) | YES(existing) | NO(P2) | PNPM expected instance/runtime/layout·실제 Session 비교 |
| sealed data cache / receipt / sync / no-replace publish — CargoVerifiedCache·projectCacheInventory·sealGoCacheTree·syncTree | YES | YES(other clients) | YES(existing) | NO(P2) | typed PNPM receipt/opaque data tar adapter |
| PNPM approved staging | NO | NO | NO | NO | 이번 단위 |
| root transaction — acquireNPMProjectGuard / acquireProjectTransactionGuard / npmProjectTransaction | YES | YES(npm/other) | YES(existing) | NO(P2) | PNPM controls·내부 links·independent approval·Host records/rollback는 B5b 이후 |
| PNPM CLI/Bootstrap complete operation | NO | NO | NO | NO | B5 staged 결과 뒤 별도 지도 |

기존 구현을 대체하거나 global symlink guard를 완화하지 않는다. `syncTree`,
`projectCacheInventory`, `npmTransactionMember`는 현재 symlink를 거부한다. PNPM tree를
그 함수에 일반적인 link 허용 옵션으로 밀어 넣지 않는다.

## 2. 비실행 staging 표현과 source 대조

Staging은 아직 Host tree를 추출하지 않는다. Bound node_modules를 controller가 렌더한
bounded data tar와 frozen 원본 package.json/pnpm-lock.yaml을 sealed **regular files**로
보관한다. 기존 common inventory/seal/sync는 그대로 사용할 수 있다. 실제 symlink 생성과
Host 반입·old ownership/transaction은 다음 B5b의 별도 typed owner다. Native store/index나
user cache를 복사하지 않으며 tar 안의 link는 앞 binder가 검증한 owned 내부 link뿐이다.

고정 공식91774e72의 modules-yaml writer Git blob cac2dbcd를 재확인했다. Linux의 virtual
store locator는 상대형이지만 storeDir는 문자열 필드이며 writer가 동일하게 재배치하지 않는다.
Guest path를 Host 탐색·쓰기 위치로 사용하지 않는다. Staging은 기존 full binder가 확인한
metadata를 원래 data로 보존하며 opaque store를 수동 생성하거나 metadata를 Host 권한으로
승격하지 않는다. Freshness cache는 B4에서 이미 폐기했다. Native PNPM을 Host에서 실행하거나
일반 PATH의 pnpm으로 replay하지 않는다. 외부 native 사용과의 interoperability는 이 staging
단위의 지원 주장이 아니다. 원본·framing 구분·exact Git blob 대조는 p2c5b5-source-capture다.

## 3. 정확한 edit map

1. `inspection/npm/pnpm_project.go`: completed project 검사에서만 immutable bound layout을
   request-local 인계한다. `PreparedLayoutFor(snapshot, acquired)`는 같은 frozen snapshot/
   exact acquired runs를 다시 결합하고 MatchesInput을 요구한다. 검사 시작·실패에는 이전
   값을 비우고 동시 같은 inspector 실행은 명시 거부한다. 짧은 mutex만 사용하며 runtime 실행
   전체를 lock 대기로 직렬화하지 않는다. 이 값은 구조/관찰의 인계이며 Policy 권한이 아니다.
2. `artifact/npm/pnpm_intake.go`: 기존 `sandbox/readPNPMIntakeArchive`의 bounded reader와
   context read를 공용 owner로 이동한다. Canonical subject handle의 이름만 열고 기존
   Lstat/open/SameFile/parent identity/size/전체 SHA/취소/Close 실패를 유지한다.
   Sandbox wrapper는 기존 sentinel과 cause를 보존하며 같은 helper를 소비한다. 새 cache가
   두 번째 실제 소비자다. 과거 native 소개의 source/ordering 의미는 그대로다.
3. `promotion/project_cache_evidence.go`: 기존 실제 record 조회 body를 facts도 돌려주는
   private helper로 재사용한다. 원래 Go/Cargo wrapper/의미는 유지하고 중복 읽기 사이의
   서로 다른 snapshot을 섞지 않는다.
4. `promotion/pnpm_cache.go`: `NewPNPMVerifiedCache`는 분리된 intake/Evidence/cache roots,
   frozen context, 좁은 layout provider와 기존 ProjectEvidenceReader를 받는다. StageProject는
   exact complete ALLOW·frozen/acquired binding·required checks와 모든 실제 Evidence를
   먼저 확인한다. Expected instance digest별 실제 Session이 고유하고 runtime/layout digest가
   일치해야 한다. Required aggregate의 expected/attempted/completed도 정확히 같아야 한다.
   Evidence reference만 기록하고 실제 record 검사를 생략하지 않는다.
5. 같은 adapter의 private renderer는 bound Entries/ReadFile만 tar에 쓰며 native Kind/Mode/
   link/bytes를 보존한다. Header 순서는 deterministic이고 hash·타입·size를 재확인한다.
   모든 acquired source bytes도 공용 intake reader로 다시 해시한다. Source/target/trusted roots
   중첩과 missing/finding/partial/foreign layout이면 cache writes0다. Explicit empty는 controls와
   빈 complete approval을 기록하고 layout/native 실행을 제조하지 않는다.
6. `StageProject`는 private data directory에 tar/두 controls를 exclusive write하고 기존
   seal/inventory/sync/renameNoReplace를 사용한다. Receipt는4MiB 고정 bounded JSON이며
   full snapshot·current policy·실제 Evidence record digests·native layout/runtime·전체 data
   inventory를 결합한다. 한도는 기존 PNPM tar/controls 및 common receipt 한도를 따른다.
   실패 뒤 partial cache를 제거하며 최초 cause를 보존한다.
7. `OpenProjectCache`는 typed staged set의 receipt digest, frozen/실제 stored Evidence/current
   policy와 data 전체 inventory/bytes·sealed modes·single links·foreign extra를 다시 확인한다.
   반환은 trusted data directory이지 실행·Host 반입 권한이 아니다. Persistent target adoption/
   독립 retained state/Manifest·SBOM·rollback는 아직 B5b MISSING이다.

## 4. 두 번째 검토와 검증

새 staging은 filesystem symlink를 생성하지 않으므로 공용 inventory/sync 의미를 바꿀 이유가
없다. Native tar는 parser/binder의 검증을 거친 immutable 값에서만 새로 쓰고 원본을 이름만
바꿔 복사하지 않는다. Tar의 전체 identity는 receipt와 actual layout Evidence에 별도로 묶는다.
Guest metadata의 경로는 data이며 Host root 선택 권한이 아니다. Request-local provider는
저장·Policy·receipt의 독립 검증을 대신하지 않는다. Artifact/metadata 자체의 성공 token이나
경로를 완료 authority로 쓰지 않는다.

Unit은 검사 실패/중복 실행 뒤 인계 거부, 정확한 subject/controls/runtime, 새 source reader의
기존 변조·symlink·취소 회귀와 두 소비자, 실제 record 누락/변조·foreign/duplicate Session,
coverage/runtime/layout 불일치·partial/required Host action 거부·writes0을 검사한다. 정상/empty
staging과 tar/controls/receipt/extra file/mode/link tamper·failure cleanup을 확인한다. 실제
bootstrap 계층의 fresh public→검사/관찰→Evidence/Policy→새 cache Stage/Open consumer에
즉시 연결해 import cycle과 placeholder를 만들지 않는다. Host target 반입은 아직 실행하지 않는다.

Canonical format/Quick/security/docs와 영향받은 artifact/npm·inspection/npm·sandbox·promotion/
application/bootstrap의 적절한 Race를 묶는다. 새 native/observer exception·runtime lock 변경은
없으며 CPP build와 이미 닫힌 B4 actual을 이유 없이 반복하지 않는다. CI405의 결과는7e966e7에만
적용하고 이 뒤 B5 source를 그 결과에 포함했다고 주장하지 않는다.

## 5. 첫 검사 결과와 추가 검증 범위

비실행 cache/공용 reader의 첫 ready source는 format0.546s/Quick191.435s/security11.618s/
docs0.236s와 artifact/npm·inspection/npm·application·sandbox·promotion·bootstrap 전체
Race148.769s가 PASS다. Normal/empty·전체 cache content/mode/symlink/foreign hardlink/
receipt·실제 record 제거와 staging late failure/원래 cause·partial cache0을 확인했다.

Production source는 이 뒤 그대로이며, adverse reader가 runtime/layout/Session/coverage/
partial summary를 바꾼 경우의 writes0, active inspector의 동시 호출 즉시 거부·getter 거부,
실패 다음 요청의 이전 handoff 폐기를 추가 검증한다. 이 reader fixture는 실제 local Store가
변조된 byte를 원래 hash로 돌려준다는 주장이 아니다. Public peer/explicit empty의 실제
bootstrap Stage/Open consumer와 source/Evidence/Policy 경계를 다음으로 실행한다.


## 6. 실제 소비자와 named local 경계

추가 회귀의 format0.569s/Quick130.142s/security8.431s/docs0.234s와 focused Race9.823s는
PASS다. Production source는 앞 전체 six-package Race148.769s 뒤 바뀌지 않았다.
실제 bootstrap binary compile3.376s와 exact test 이름을 확인했다.

Public actual43.959s/test41.54s는 peer4 acquired entries/고유 Evidence24개와 실제 static-all/
모든 fresh instance/Policy ALLOW→sealed data tar Stage/Open·byte 변조 거부를 통과했다.
실행 성공 뒤 정확한 layout 인계와 취소된 다음 요청의 handoff 폐기도 확인했다. Explicit empty는
두 sealed controls만 보관했고 per-entry Evidence/layout tar/dynamic instance는 만들지 않았다.
두 case 모두 Host project 부재·helperwait0/CONFIRMED_STOPPED·owned runtime directory 정리와
보호 Host 구성 전후 일치를 확인했다.

첫 actual log는 empty에도 native_layout_data/stored_evidence/drift_rejected를 고정 true로
표시했다. Assertion과 실제 분기는 tar/변조 검사를 nonempty에만 실행하므로 empty의 그 세
범위는 미실행이다. 원본을 편집하지 않고 scope-correction JSON과 마지막 source의 conditional
로그로 바로잡았다. 이 log-only 변경 때문에 같은 native positive를 재실행하지 않으며 최종
realcorpus build validity는 별도로 확인한다. Empty의 정상 Stage/Open 성공은 그대로다.

[로컬 영수증](./p2c5b5a-pnpm-approved-staging-local-20261025.json)은 named scope의
IMPLEMENTED/WIRED/QUALIFIED=YES와 whole P2 ACCEPTANCE_CLOSED=NO를 기록한다.
최종 전체 stage 후보의 security/docs 및 index/blob/disk/mode/tree 일치 뒤 commit한다.
CI405는7e966e7 source만 소유하며 이 B5a source는 아직 원격에 올리지 않는다. 다음은 typed
Host 추출·독립 retained approval·실제 operation records와 기존 transaction 재사용의 지도다.

마지막 log-only source의 canonical format1.417s와 realcorpus compile4.378s도 PASS다.
