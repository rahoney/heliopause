# P2c5-B5b3 — 기존 transaction과 독립 retained approval 착수 감사

Baseline bcc316b/tree64b51556291a60645c983841f497bd4616e3790c. 같은 M13-004/P2다.
CI406은 앞 fd3506b이고 첫 확인은03:40:25 KST다. 이 문서는 source 작성 전 감사이며
아래의 남은 지도 확정 전 transaction source는 수정하지 않았다.

Read: Queue·P2 baseline§P2d·B5a/B5b1/B5b2 leaf·client 설계§4·Domain09/13·Architecture02/03,
기존 npm_transaction/member/update, Go/Terraform retained approval·shared project lock,
application.executeGuardedProjectUpdate와 UV complete consumer, 관련 artifact/sandbox/evidence lessons.

## Baseline audit

| 책임 / 기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 실제 PNPM approved cache/records/private materialization | YES | YES(actual bootstrap) | YES(named local) | NO(P2) | target transaction 소비 |
| guard→select→inspect→stage→commit·원래 cause/close — executeGuardedProjectUpdate | YES | YES(Go/Cargo/others) | YES(existing) | NO(P2) | PNPM request-local typed native context 연결 |
| original backup/no-replace publish/rollback/known-only backup removal — npmProjectTransaction | YES | YES(npm) | YES(existing) | NO(P2) | PNPM fixed controls/member profile·retained state 연동 |
| exact pinned Root lock — acquireProjectTransactionGuard | YES | YES(Go/Cargo/Terraform) | YES(existing) | NO(P2) | npm과 같은 lock basename 사용 |
| 독립 retained approval — approvedGoProjectGuard/Terraform counterpart | YES | YES(other clients) | YES(existing) | NO(P2) | PNPM 이전 cache/records·actual Evidence/target census 결합 |
| PNPM guarded target publication | NO | NO | NO | NO | source/negative/actual 종단 소비 |
| PNPM native adoption·no-op·CLI/whole P2 | NO | NO | NO | NO | 후속 acceptance; 이번 감사로 완료 표시 안 함 |

기존 npm engine은 package-lock.json과 `.heliopause/npm-transaction.json`을 하드코딩하고
regular-only member/sync를 사용한다. 이름만 치환하거나 모든 symlink를 허용해서 재사용하지
않는다. 이미 구현된 engine을 대체·재작성하지 않으며 필요한 고정 PNPM profile 분기와
같은 bytes/identity guard의 좁은 추출만 검토한다. 기존 npm nil/default 경로는 유지한다.

## 확인한 연결 경계와 구현 순서

1. **선택 전에 guard:** 현재 PNPMVerifiedCache는 선택 뒤의 frozen context를 필수로 받는다.
   Begin을 그 cache 생성 뒤로 미루면 native selection 동안 원본 보호를 잃는다. Promotion의
   roots/Evidence/actual operation과 target guard는 먼저 만들고, Commit에서만 request-local
   selected-cache getter를 소비하는 방식으로 기존 workflow를 연결한다. Getter 결과 자체를
   승인으로 쓰지 않고 roots/operation/full frozen snapshot·actual staged set/evidence를 다시 검증한다.
   Core에 pnpm model을 넣거나 native 선택을 두 번 수행하지 않는다.
2. **고정 profile:** project lock basename은 `.heliopause-npm-transaction.lock`으로 공유한다.
   고정 surface는 package.json/pnpm-lock.yaml/node_modules/.heliopause다. Lock 부재는 원래
   pnpm-lock.yaml이 absent로 freeze된 경우만 허용한다. Engine의 backup→publish→metadata→
   root sync→output verify→durable independent approval→backup disposal 순서를 보존한다.
   Shared regular-only inventory/sync는 기존 npm 경로에서 그대로 두고 node_modules만
   typed PNPM owned census를 사용한다. Metadata directory는 여전히 regular-only다.
3. **private workspace 인계:** B5b2의 prepared tree는 이미 descriptor/entry identity·bytes와
   durable sync를 확인했다. Transaction에 member 소유권이 넘어가는 정확한 rename 성공 뒤에만
   workspace의 expected inventory에서 해당 prefix를 제거한다. 최종 Close는 남은 owned
   inventory만 정리한다. Partial publish/rollback에서 unknown data를 RemoveAll하지 않는다.
4. **독립 retained state:** target 안의 marker/Manifest만으로 기존 payload 소유권을 인정하지
   않는다. 별도0700 state root의 target-bound canonical approval은 prior cache handle/digest,
   schema2 receipt 전체·records inventory/Manifest/actual Evidence digests와 Host census를 결합한다.
   이전 승인 검증에서 mutable reference를 다시 resolve하거나 old runtime을 current ALLOW로
   재사용하지 않는다. 새 selection은 항상 새 현재 검사/Policy를 거친다. 현재 full context가
   없는 과거 approval은 trusted saved receipt/Evidence/controls/provenance에서 offline 대조하며
   해당 helper를 current cache Open에서도 재사용한다. 이름만 같은 cache를 승인하지 않는다.
5. **새 설치와 인계 구분:** 단일 existing project/명시 target의 두 controls를 guard한다.
   선언된 dependencies가 있어도 기존 node_modules/HAA namespace가 모두 없으면 처음 설치할
   수 있어야 한다. 실제 기존 native payload는 별도 explicit adoption과 독립 source proof가
   필요하며 shared foreign hardlinks·unknown HAA namespace·corrupt prior state를 우회하지 않는다.
   지원하지 않는 new-target/runtime/global/workspace 형식은 조기 거부하고 전체 목표의 MISSING에
   남긴다. Native adoption의 정확한 proof/비교 절차는 구현 전에 추가 확정한다.
6. **no-op와 실패:** 원래 controls/native tree와 selected bytes가 같은 retained member는
   원래 inode/mode를 보존하는 branch를 검토한다. Fresh inspection/operation records는 별도로
   갱신한다. State publication late failure도 기존 target originals와 독립 state를 함께 rollback하며
   불확실하면 journal/unknown data를 보존하고 새 mutation을 거부한다. 취소가 cleanup/rollback을
   건너뛰거나 이미 완료된 Policy를 BLOCK으로 바꾸지 않는다.

## Source 작성 전 남은 지도 확정

아직 고정하지 않은 것은 old cache receipt/actual Evidence 재검증 helper의 정확한 추출,
PNPM member census와 기존 member.verify/remove의 책임 연결, state publication/rollback
callback의 phase fault 범위다. 먼저 기존 symbol의 호출/삭제 순서를 읽고 각 변경·정상 npm
회귀를 지도에 연결한 뒤 두 번째 검토를 완료한다. 감사 중 generic replacement engine,
future interfaces/flags 또는 CI job을 만들지 않는다. 이후 meaningful unit 단위로 source와
실제 consumer를 구현하며 source-bound canonical profiles/관련 Race와 negative/rollback/actual을
확인한다. CLI/adoption/whole P2는 독립 acceptance로 남는다.

## 정확한 edit map과 두 번째 검토 완료

2026-10-25 실제 owner/symbol과 B5b2 source를 다시 대조했다. 새 parallel transaction engine은
추가하지 않는다. 아래의 private fixed profile 분기만 기존 engine에 연결한다.

- `pnpm_cache.go`의 현재 Open body를 saved receipt에도 쓸 수 있는 private helper로 좁게
  추출한다. 현재 Open은 같은 frozen/current actual Evidence와 trusted record 재생성을 계속
  수행한다. 별도 `pnpm_retained.go`는 이전 trusted approval의 cache handle/receipt SHA와
  원래 Source/Run/subject/actual Evidence record digest를 읽는다. 이전 receipt만 보고 새 ALLOW를
  만들거나 Source metadata를 재resolve하지 않는다. Current와 retained 두 실제 소비자가 있다.
- `pnpm_owned_member.go`는 descriptor-relative closed native inventory를 캡처한다. Native member
  전체 canonical path/kind/mode/owner/size/hash/link census의 SHA를 독립 state에 기록하고,
  다음 Begin은 현재 전체 census와 비교한 뒤 그때의 inode snapshot을 고정한다. 100000 native
  entries 전체를4MiB state에 중복 직렬화하지 않는다. Existing native200MiB/entry bounds와
  records/controls의 기존 한도를 유지하고 hardlinks/escape/unknown types를 거부한다. B5b2의
  entry readback body를 workspace와 transaction member가 재사용하며 public workspace0700
  계약은 완화하지 않는다.
- `npm_transaction.go`와 `npm_transaction_member.go`에는 private PNPM profile만 추가한다.
  기본 nil profile의 기존 names/metadata/regular-only snapshot/sync/rollback 순서는 그대로다.
  PNPM은 고정 lockfile/name list·original absent lock·typed node_modules census·이미 동기화한
  prepared workspace의 재검증·metadata marker·durable state hooks를 선택한다. Existing npm
  정상/실패/early inventory/rollback conflict 회귀를 함께 소비한다.
- `pnpm_transaction.go`는 roots/Evidence/actual operation과 **선택 뒤 cache getter**를 받는
  promotion owner다. Begin은 getter를 호출하지 않고 shared npm lock과 원래 controls/managed
  members·independent state를 먼저 고정한다. Commit에서 getter 결과의 roots/actual operation/
  full frozen update와 staged set을 재확인한다. Guard가 없는 캐시나 다른 요청의 cache로
  승격하지 않는다. Core ports/type은 변경하지 않는다.
- `pnpm_approved_transaction.go`는 prepared members의 정확한 rename 인계 뒤에만 workspace
  inventory를 줄이고 기존 engine을 호출한다. Fixed marker는 operation/Manifest/graph/cache
  identity를 담되 독립 state를 대체하지 않는다. State는 prior cache receipt 전체와 네 managed
  member census digest를 결합한다. Metadata/root sync와 output verify 후 state를 게시하고,
  backup disposal 전 phase failure는 originals/state를 함께 rollback한다. Cleanup/rollback
  unknown은 journal/unknown members를 보존하며 새 Begin을 막는다.
- No-op는 동일 byte controls를 원래 mode/inode로 유지하고, native member는 이전 trusted
  layout fingerprint와 새 실제 검사의 fingerprint 및 graph/runtime이 같을 때 유지한다. 기존
  `pnpmInspectionLayoutDigest`가 이미 원래 source/전체 entry와 typed metadata를 결합하며
  PrunedAt/virtual-lock formatting만 canonicalize함을 확인했다. 새로운 ignored-path 예외나
  metadata normalizer를 만들지 않는다. Fresh operation records와 approval은 계속 갱신한다.
- `application/pnpm_project.go`는 기존 executeGuardedProjectUpdate를 소비하는 작은 complete
  project adapter다. `bootstrap/pnpm_project.go`는 request별 native mode/selector→정확히 한
  SelectProject→typed frozen context→기존 ProjectInspectService/cache를 연결한다. Selected
  cache getter는 이 request-local 객체에서만 제공하며 실패/새 요청에 이전 값을 재사용하지
  않는다. Source npm과 client pnpm을 구분하고 CLI 노출은 backend qualification 다음 단위다.

첫 backend 단위는 existing single project의 fresh empty payload와 HAA managed payload를
대상으로 한다. 선언된 dependencies가 있어도 node_modules/HAA namespace가 없으면 fresh
install이며 optional original pnpm-lock도 exact absent/present로 freeze한다. Native/unmanaged
payload adoption의 독립 archive proof는 별도 후속이므로 unknown namespace/foreign shared
hardlink/corrupt state를 adopt 옵션으로 우회하지 않는다. Controlled existing input 재등록은
명시 adopt에서 controls만 허용하고 원래 managed payload/approval 검증은 유지한다.

두 번째 검토에서 확인한 핵심은 Begin 이전 cache 생성의 시간 역전, user-writable marker를
권한으로 쓰는 순환, per-file census의 state 크기 폭증, metadata timestamp로 no-op를 재작성하는
오류다. 위 연결은 기존 owner와 fingerprint를 유지해 이 네 gap을 닫는다. Post-commit backup
disposal 중 오류는 full rollback 성공으로 포장하지 않고 기존 fail-closed 경계를 유지한다.

검증은 control enrollment/optional lock·fresh/empty/managed repeat/no-op, source/operation/cache
substitution, 실제 Evidence/cache/target tamper, other-client concurrent lock, original drift와 취소,
backup/publish/metadata/state 단계 fault·rollback conflict/journal 재진입을 포함한다. 실제 public
consumer는 request-local adapter에서 기존 guarded Application 경로 전체를 소비한다. 기존
`TestLinuxNPMPromotionIntegration`도 engine 변경의 실제 소비자로 확인한다. Canonical profiles/
관련 전체 Race와 named actual 뒤에만 이 backend 범위를 qualification하며 native adoption/CLI/
whole P2는 계속 MISSING이다.

## 실제 구현과 named local 검증

기존 npm engine에 private 고정 PNPM profile을 연결했고 nil/default npm 동작을 유지했다.
선택 전에 shared target lock·original controls/member census·독립 state를 고정하고, Commit에서만
선택 후 cache getter와 actual operation/full frozen context/현재 Evidence를 대조한다. Native
hardlinks는 이미 검증한 독립 single-link copy이고 내부 symlink census만 전용 member로 관리한다.
성공한 rename 뒤에만 prepared inventory의 prefix를 인계한다. 네 member의 persistent census
SHA와 full cache approval/receipt digest를 target 밖 state에 결합하며 state 게시 실패까지 rollback한다.
동일 byte controls와 같은 graph/runtime/layout의 native payload는 inode/mode를 보존하지만 매번
새 request/inspection/operation records를 만든다. Explicit control enrollment도 이전 payload와
독립 approval/cache/Evidence 검증을 생략하지 않는다. Unmanaged native payload는 계속 거부한다.

Engine-only Quick162.799s 뒤 첫 bootstrap Quick38.827s는 gated actual에서만 소비된 private
factory를 U1000으로 거부했다. Native 미실행이며 first-source/로그를 보존했다. Suppress나 export
이름 변경 없이 default에서 실제 bindings 필수성·선택 전 Inspect/Stage·재선택 거부를 검증했다.
교정 Quick129.955s/security10.486s/docs0.235s 및 promotion/bootstrap/application/evidence 네
package 전체 Race135.126s PASS다. Original drift·foreign state/cache/actual stored Evidence,
shared lock·다른 operation·native link/hardlink, 11 forward faults와 managed full-copy rollback,
foreign output 보존/incomplete rollback journal/new Begin 거부를 포함한다.

Public 실제134.084s/test131.66s는 single install/repeat/add(6→7), peer4 두 번, explicit empty 두
번을 request-local Application 경로 전체로 게시했다. Fresh operation/Manifest와 독립 approval,
동일 native/control inode·0600 보존, root script 미실행/user file 보존을 확인했다. Empty는 native
instance/per-entry Evidence를 invent하지 않는다. 기존 실제 npm new target 및 first/retained project
소비자도6.243s/test3.88s PASS다. 최종 single의 native update 모드·선택7 유지·native inode no-op는
별도 focused actual로 검증했다. 세 lifecycle 모두 helperwait0/CONFIRMED_STOPPED·Host 전후 동일/
owned runtime directories 정리다. Actual update log의 exact_constraint label은 manifest의 literal
constraint를 별도 검사하지 않으므로 근거로 쓰지 않는다. 원본은 보존하고 두 진단 문자열만
selected_version으로 교정했으며 assertions/production을 바꾸지 않고 final format/realcorpus compile을
확인했다. 구체 source/원본/focused 시간·Darwin compile 범위는
[로컬 영수증](./p2c5b5b3-pnpm-project-transaction-local-20261025.json)이 소유한다.

이 named backend는 IMPLEMENTED/WIRED/QUALIFIED=YES(로컬)지만 전체 P2 ACCEPTANCE_CLOSED=NO다.
Native/unmanaged adoption의 독립 archive proof, CLI/help/실제 사용자 진입점과 원격 qualification,
new target/global/workspace 범위는 완료하지 않았다. 다음 단위는 기존 canonical CI의 정확한
actual backend 소비자 연결과 이어지는 CLI/원본 인계 acceptance 지도이며 새 job/engine은 만들지 않는다.
