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
