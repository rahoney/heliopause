# P2c5-B3c — 원본 archive와 전체 inspection layout의 결합

2026-10-24, baseline5a051c1 (Go source는442af68과 동일). 같은 M13-004/P2다.
[앞 단계와 실제 결과](./p2c5b3-pnpm-layout-plan-20261024.md),
[B1 context](./p2c5b-pnpm-production-context-plan-20261024.md), Architecture02/03,
Domain08/12/13, artifact L15/L20·sandbox L08/L18·evidence L11/L16을 읽는다.

## 1. Source 변경 전 baseline audit

| 책임 / 실제 symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 전체 frozen lock/acquired/peer instance — PNPMInspectionInput | YES | YES | YES(B1 ordinary) | NO(P2) | layout에 실제 결합 |
| 원본 전체 file hash/manifest — Inspector.InspectArchive·ArchiveInventory | YES | YES(producer) | YES(local/ordinary) | NO(P2) | 모든 acquired subject의 정확한 coverage 결합 |
| native tar 입력 — ParsePNPMLayoutArchive | YES | YES(actual fixture) | YES(local) | NO(P2) | unverified payload를 원본 file/edge에 대조 |
| dirname / Node bin 전체 code — PNPMInstanceDirectory·ParsePNPMNodeBinShim | YES | YES(actual fixture) | YES(local; dirname ordinary 포함) | NO(P2) | 원본 bin 선언/target/header·Node paths에 결합 |
| 전체 inspection layout binding/생성 metadata 검증 | NO | NO | NO | NO | 이번 B3c |
| production native prepare/static-all/실제 per-instance aggregate | NO | NO | NO | NO | B3c 결과 뒤 정확한 lifecycle 지도와 기존 owner 재사용 |
| Host records/promotion/transaction/CLI·full qualification | NO | NO | NO | NO | 후속; 이번 inspection 값으로 Host 승격 안 함 |

기존 parser와 단일 Backend.Execute/Policy를 재작성하지 않는다. PNPM source의 store-entry
mode는 executable class와 umask를 사용하므로 original tar mode와 숫자가 항상 같은 것은
아니다. 공식 file_mode.rs의 Git blob을 대조해 raw owner에 보존했다. 허용 변환은 아래처럼
명시하고 mode/owner 검사를 없애지 않는다.

## 2. 정확한 수정 위치와 방법

### 2.1 artifact/npm/pnpm_inspection_layout.go

`PNPMInspectionInput.BindLayout(ctx, inventories, parsedLayout, expectedStoreRoot)`를 추가한다.
결과 `PNPMInspectionLayout`은 static input binding만 보존하는 immutable 값이며 Policy,
Verified Set/Host 승격 또는 dynamic 완료가 아니다. Raw native bytes를 직접 실행하는
API를 만들지 않는다. Exact acquired subject와 instance key로 target을 조회하고 내부 slice/
map을 노출하지 않는다. Zero/foreign/missing target은 오류다.

- Input.Valid와 모든 ArchiveInventory의 exact AcquiredArtifact(원래 Run/handle·size/SRI/
  digest 포함) 일치·중복 없는 전체 coverage를 요구한다. Inventory는 정적 producer가
  실제 bytes를 측정한 값이어야 하며 구조 constructor의 Valid를 source 인증으로 쓰지 않는다.
- 모든 locked snapshot을 기존 dirname codec으로 매핑하고 서로 다른 instance의 physical
  root 충돌을 거부한다. Unique archive를 합치더라도 peer instance는 하나도 버리지 않는다.
- 원본 inventory의 모든 regular file과 directory·필요 parent를 각 instance root 아래에
  매핑한다. Native regular/hardlink의 logical bytes size/SHA256을 원본과 비교한다. Alias는
  이미 구조적으로 닫힌 내부 regular bytes이고 **양쪽 expected file**도 같아야 한다. Link
  자체를 source identity로 쓰지 않고 foreign/extra/missing 파일을 허용하지 않는다.
- 모든 root/dependency/peer symlink는 frozen snapshot의 정확한 instance로 향해야 한다.
  Generated private-hoist alias는 검증한 modules metadata와 같은 map이어야 하며 target은
  기존 frozen instance만 가능하다. Alias와 real file/dir/bin의 충돌은 거부한다.
- Owner는 기존 envelope의1000:1000을 유지한다. Regular/dir의 group/other write 및 특권
  bit는 거부한다. Regular는 owner read/write, directory는 owner read/write/search가 필요하다.
  원본 executable class 또는 **검증된 bin target**만 owner execute를 가질 수 있다. Nonexec
  원본에 임의 execute를 더하지 않으며 read bits/umask 차이만 허용한다. Symlink는 native의
  표시 mode0777이다. 검사한 내부 hardlink 관계도 identity와 별도로 보존한다.
- 기존 envelope 상한을 expected file/dir/link 생성에도 적용하여 큰 map을 무제한 만들지
  않는다. 실제128MiB native volume/기존 runtime budget은 그대로이며 상한을 올리지 않는다.

### 2.2 원본 manifest/bin의 결합

같은 package의 private helper에서 inventory.Manifest를 기존 strictControlJSON으로 검사하고
`bin` string/object를 해석한다. Name/version은 원래 inventory에 이미 묶여 있다. String bin의
기본 이름은 package의 unscoped name이다. Command/path 길이·안전한 원자·중복/escape를
검사하고 target이 원본의 regular file이어야 한다. Unknown directories.bin/bundled root
node_modules/지원하지 않는 bin 형식은 추측하지 않고 unsupported로 남긴다.

Self, 실제 dependency consumer, root direct dependency와 private-hoist consumer의 bin
위치를 각각 예상한다. 같은 command가 다른 target으로 충돌하면 자동 선택하지 않는다.
Expected target은 이 원본 manifest/graph에서만 만들며 native shim marker를 읽지 않는다.
`ParsePNPMNodeBinShim`으로 전체 code를 확인하고 Node module path도 실제 expected directory
및 해당 target/consumer의 허용된 module search directory에 대조한다. 순서도 문맥에 보존한다.
현재 지원 Node/no-extra-runtime-args와 일치하는 원래 target header를 확인하고 unknown
runtime/args·설치 뒤 생성될 누락 target은 required unavailable/incomplete로 설명할 입력이다.

### 2.3 artifact/npm/pnpm_layout_metadata.go

두 생성 file을 정확히 구분한다. Unknown generated file이나 이름만 맞는 임의 payload를
통과시키지 않는다. 기존 bounded JSON/YAML parser를 재사용하고 Core/Application에 schema를
넣지 않는다.

- `.pnpm/lock.yaml`: 원래 package.json과 input의 client version으로 ParsePNPMLock하고
  pnpmSnapshotLockState 전체를 frozen input과 비교한다. Format만 다른 YAML은 같은 의미로
  비교하지만 graph/SRI/peer/flags가 다른 virtual lock은 거부한다.
- `.modules.yaml`: 현재 native는 JSON이다. strictControlJSON·known-fields decode, 고정
  layout5/isolated·`.pnpm`/기존 dirname max120, input의 `pnpm@<client-version>`, 모든 included
  group true와 default hoistPattern `*`/public hoist 없음·skipped 없음을 요구한다.
  Pending build는 frozen instance 또는 root의 bounded data reference이며 실행 승인이 아니다.
  AllowBuilds는 이 HAA 기본 경로에서 비어 있어야 한다.
- StoreDir는 caller가 소유한 **inspection guest store root**와 현재 protocol suffix v11에만
  결합한다. Native store/index를 수작업으로 만들지 않는다. Arbitrary Host store path나
  실제 Host 인계·publication metadata로 전용하지 않는다.
- Hoisted map은 private kind·정확한 package 이름과 frozen snapshot만 참조하고 실제 hidden
  symlink와 일대일로 맞아야 한다. Metadata가 unlisted package/file을 만드는 authority는 아니다.
  PrunedAt는 bounded valid time data로만 취급하며 안전/완료 근거로 사용하지 않는다.

실행 문맥 fingerprint에는 frozen snapshot/runtime/acquired subjects와 canonical file/alias/
edge/bin metadata를 포함한다. Volatile timestamp를 안전 identity나 완료 token으로 쓰지 않는다.
이 값은 이후 cache/records에서 **추가** 문맥 결합에 사용하며 기존 npm 단일 cache를 그대로
PNPM peer 실행에 재사용하는 근거가 아니다. 최종 cache/wiring의 별도 감사는 남아 있다.

## 3. 실제 fixture 연결과 검증

기존 qualifyPNPMOfflineFixture의 동일 public source binding/Acquire/SRI를 재사용한다.
Static.Inspect 대신 InspectArchive에서 전체 inventory를 얻고, 이미 있는 sourceBound와
controls로 B1 context/input을 만든다. Callback에 이 두 typed 값만 추가한다. 기존 A3/B2
callers는 같은 작업을 유지하며 새 값은 필요한 실제 layout fixture만 소비한다. Registry
fixture/SRI를 새로 복제하거나 일반 Resolve로 peer packument 한도 실패를 반복하지 않는다.

Native layout fixture가 complete binding을 수행하고 static-acquired subjects/모든 expected
instance target을 확인한다. 이어 cloned tar의 package file 변조·누락/추가, 잘못된 alias/
edge/bin/metadata/virtual lock 등을 fresh **parser/binder 입력**으로 검사한다. Native 작업을
negative마다 다시 실행하지 않는다. 변조 bytes는 artifact를 실행하거나 Policy에 넘기지 않는다.

Unit tests는 기존 peer4와 synthetic six-archive/seven-instance fixture를 source 모형으로
사용하되 synthetic authenticity를 public actual로 주장하지 않는다. Missing/duplicate/foreign
inventory, executable/owner 변조, 모든 peer instance·exact symlink·cross-package 동일-byte
hardlink, bin target/code/module paths, unknown/duplicate metadata·snapshot drift·getter mutation·
cancel과 bound 초과를 확인한다. Same archive의 두 peer instance target이 모두 남아야 한다.

Canonical format/security/Quick/docs·npm 전체 Race와 fresh native transitive/peer actual을
묶어 확인한다. 기존 Mode/범위를 완화하거나 finding을 지워서 fixture를 통과시키지 않는다.
완료 namespace bulk는 all-UID references/query errors0 뒤 삭제한다. 작은 helper마다 remote
CI를 시작하지 않고 다음 production prepare/실제 closure 실행 단위에서 묶는다.

생산 static-all caller는 inventory를 쌓는 동안에도 aggregate file/manifest/expanded footprint를
계산해 상한에서 중단해야 한다. 모든 거대한 inventory를 메모리에 만든 뒤에만 binder의
상한을 검사하는 방법은 부족하다. 이 caller의 resource gate도 후속 wiring의 MISSING이다.

## 4. Source 작성 전 두 번째 검토

이미 측정한 inventory와 native envelope/Node code parser를 같은 결합 지점에서 소비하므로
parser/Policy/Store 재구현이나 Application의 PNPM 해석이 필요 없다. JSON summary나 native
성공 exit0로 file/edge/bin coverage를 추정하지 않는다. Native private-hoist 선택도 이 exact
layout의 관측 문맥이며 source archive·허용된 frozen instance 밖으로 확장할 수 없다.

File mode를 원본 숫자 그대로 요구하면 legitimate store normalization을 거부하므로 허용
변환을 제한했다. 반대로 execute/world-write·특권 bit를 임의로 허용하지 않는다. Full source
binding은 사실 생산자의 실제 rehash와 함께 필요한 값 결합이고 Policy ALLOW가 아니다.
Guest StoreDir와 타임스탬프는 Host 설치·캐시 승인에 사용할 수 없다. Root scripts 실행0과
기존 controls 보존·실제 per-instance complete observation은 다음 production owner가 별도로
검증해야 한다.

추가 검토에서 **Host 원본 controls 보존과 Artifact session으로의 원문 소개는 다른 책임**임을
명시했다. 원래 package.json의 scripts/config/author 등에는 사용자 전용 값이 있을 수 있다.
Native metadata 준비는 scripts0인 CONTROL 단계이며, 그 뒤 Artifact lifecycle session에는
frozen root 이름·정확한 version/group에서 만든 최소 private manifest만 소개한다. 원본
user script/config·환경 값·불필요한 generated metadata를 실행 fixture에 복사하지 않는다.
Host 원본 bytes와 context digest는 그대로 보존하고 최소 manifest의 도출 관계도 검증한다.
No-Host-assets/sanitized-project-data invariant를 지키는 이 introduction 경계를 이후 owner
edit map/negative에 포함한다. B2의 controlled synthetic root fixture 성공은 사용자 manifest
전체를 그대로 전달해도 된다는 근거가 아니다.

본 단계 뒤에도 static-all 전역 gate, production prepare owner·정리 뒤 값 반환, 독립 fresh
per-instance 실행/관찰 aggregate, missing/duplicate/partial coverage 및 첫 오류·이미 관측한
보안 사실 보존, long hashed dirname의 기존 @ locator gap, Host records/transaction/CLI와
전체 P2 qualification은 남는다. 여기서 미확정 runner interface나 미래 client scaffold를
만들지 않는다. IMPLEMENTED/WIRED 여부는 실제 source와 결과 뒤 갱신한다.

## 5. 첫 actual의 누락 반증과 source 수정 전 경계 재검토

첫 Quick42.964s는 새 test의 literal nil Context를 SA1012로 거부했다. 해당 비관용 호출을
제거하고 실제 cancellation/cause 거부를 유지한 Quick117.370s/security12.250s와 전체 npm
Race7.107s는 PASS다. 별도 realcorpus compile은 기존 InspectArchive의 반환 순서
(inventory, report, error)를 새 caller가 뒤집은 연결 오류를 잡았다. 기존 producer는 바꾸지
않고 caller를 교정했으며 이 첫 준비에는 native 실행이 없다. 빈 준비 디렉터리는 전체 UID
참조/query errors0 뒤 삭제했다. 이 검사들을 아직 실제 전체 배치 성공으로 분류하지 않는다.

Actual9.563s/test6.92s는 transitive의 원래 controls·identity·links/offline·complete 관찰과
Host불변/helperwait0를 보존했지만 full binder가 거부했다. 앞 B3b0/B3b2b 원본 전체 목록의
`node_modules/.pnpm-workspace-state-v1.json`을 이 edit map이 누락한 것이 확인됐다.
문서의 “생성 file 두 개”는 **검사에 소비하는** 두 file만을 뜻하도록 정정한다. Unknown
file을 일괄 허용하거나 fixture에서 몰래 삭제하여 이 실패를 성공으로 바꾸지 않는다.

Pinned upstream91774e72의 workspace-state/src/lib.rs(6f583ccb), package-manager의
install/workspace_state.rs(8753e958)/workspace_state/write.rs(56f48a19)를 Git blob으로
대조했다. 이 세 번째 file은 pnpm의 다음 실행용 freshness cache이며 시간·filesystem stat·
재검증 주장·설정/프로젝트 요약을 보존한다. HAA의 archive/graph/관찰 authority가 아니다.
해당 cache schema와 정책을 HAA가 재구현할 이유가 없고 Artifact의 고정 npm lifecycle도
이 cache를 소비하지 않는다. 따라서 다음처럼 **inspection payload에서 명시 폐기**한다.

- 정확한 위 경로의 regular file 하나만 인지한다. 기존 owner/mode·envelope bound와
  strict bounded JSON object/duplicate-key 검사를 적용한다. Alias/symlink·임의 다른 파일을
  이 예외에 넣지 않는다. 그 cache 내부 settings/validation을 HAA가 검증했다고 주장하지 않는다.
- BindLayout의 complete native entry accounting에는 이 file을 포함하되, 성공 결과의
  Entries/ReadFile에서는 제거하고 digest에도 포함하지 않는다. Private native body를
  복제하거나 raw tar 전달 API를 추가하지 않는다. Payload에는 cache가 존재하지 않으므로
  native cache의 주장/경로/시간이 Artifact session이나 향후 Host 승격으로 전달되지 않는다.
- 실제 검증에 소비하는 .modules.yaml과 virtual lock은 앞 §2.3의 정확한 검증을 유지한다.
  미래 소개 owner는 이 둘도 실행에 불필요한 metadata로 복사하지 않는 앞 §4 경계를 따른다.
- Unit은 cache 필수 coverage·잘못된 JSON·duplicate·alias·getter 부재와 cache의 timestamp/
  임의 validation data 변화가 안전 fingerprint를 바꾸지 않음을 확인한다. Native cloned
  input에도 malformed cache 거부와 성공 payload에서 cache 부재를 추가한다. Actual positive
  실패 뒤 peer와 기존 negative는 미실행으로 두고, 교정 뒤 fresh 전체 시험으로만 닫는다.

이 방법은 같은 immutable binder의 새 누락 경계를 구현하는 것이며 이미 구현된 native
cache/Policy/producer를 대체하지 않는다. Unknown metadata의 무조건 신뢰, freshness를
검역 완료 근거로 사용, native whole tar의 Artifact 이송은 계속 금지한다. 첫 actual과
교정 전 source·실패 로그는 같은 raw owner에 보존한다.

## 6. 교정 후보의 로컬 실제 결과와 다음 경계

Cache 경계 교정의 Quick137.517s/security6.852s·npm artifact 전체 Race7.102s가 PASS다.
Fresh actual17.553s/test15.07s는 transitive2 archives/2 instances와 peer4/4를 원본 Acquire/
SRI/InspectArchive 전부에 대조했다. Self/consumer/private-hoist bin3개도 원본 manifest/
regular target/full reviewed code에 결합했다. 동일 captured native input의 parser/binder
변조7/8개는 모두 거부했고 negative마다 native 실행을 반복하지 않았다. Bound getters에
native freshness cache가 없으며 private root manifest는 exact dependency/group만 보존한다.
정상 source의 동일 LICENSE hardlink와 synthetic six-archive/seven-instance의 peer2개
보존도 unit에서 확인한다. Root bin/scoped와 production execution 전체 qualification은
이 두 public fixture 성공으로 닫지 않는다.

두 actual의 complete observation·금지0·Host 전부 동일/helperwait0를 확인했다.
첫 실패의 final original1과 교정 성공의 final original0을 각각 보존했다.
첫 실패는 실패로 보존했고 그 runtime165,938,454B와 교정 성공 runtime166,000,518B를
각각 all-UID reference0/query errors0 확인 뒤 삭제했다. 앞 literal nil/반환 순서 실패나
처음 freshness cache 누락을 원격 CI404/전체 P2 성공으로 바꾸지 않는다.

마지막 cache JSON 크기 상한 negative까지 포함한 동일 production source의 최종
format.428s/Quick36.543s/security5.494s·전체 npm Race6.579s도 PASS다. 새 unit-only
negative는 sandbox native binary에 포함되는 dependency package의 production bytes를
바꾸지 않으므로 위 actual을 불필요하게 반복하지 않았다.

[로컬 영수증](./p2c5b3c-pnpm-layout-binding-local-20261024.json)이 source·최종 canonical
검사·raw 원본의 범위를 소유한다. Baseline audit의 이번 binding은 IMPLEMENTED=YES /
WIRED=YES(actual fixture consumer) / QUALIFIED=YES(명시 local 범위) /
ACCEPTANCE_CLOSED=NO(full P2)다. Production static-all/aggregate resource gate·metadata
prepare owner의 정리 후 값 반환·fresh per-instance introduction/실제 관찰 aggregate,
long hashed dirname locator·cache/provenance/Host records/transaction/CLI와 전체 P2는
계속 MISSING이다. 다음에는 기존 PNPMResolver·Backend/단일 dynamic 정상 owner를 읽고
재사용 가능한 symbol/수정 위치·각 first error/coverage 거부를 확정한 지도부터 이어간다.
