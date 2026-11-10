# P2c5-B5d — native PNPM 원본 소유권의 독립 source proof 착수 감사

Baseline a03ff13/tree50d52f2f5262406f648e4bd2fc49ebe37a700060. M13-004/P2 하나만 진행한다.
앞 e9ca85e의 CI407/run40128032402는 Required 포함10SUCCESS이며 backend 범위만 닫혔다.
이 감사는 CLI/source proof를 앞 원격 성공으로 전용하지 않으며 새 Source 작성 전 지도다.

Read: Queue, client 설계§4/9, P2 baseline§P2d, B5b3/B5c audit/local receipts,
기존 UV original inventory proof·npm original payload validation, pnpm archive/layout/modules/
bin·inspection budget와 guard/member/transaction, artifact/evidence/sandbox lessons.

## 독립 baseline audit

| 책임 / 기존 symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| guarded managed/fresh PNPM CLI·transaction·retained approval | YES | YES | YES(named local) | NO(P2) | 독립 source가 입증된 unmanaged native 인계 |
| original census — capturePNPMOwnedMember/approvedPNPMProjectGuard | YES | YES(managed state) | YES(named local) | NO(native) | prior state가 없는 원본의 explicit proof 연결 |
| 공개 exact metadata/SRI/acquisition·static archive inventory | YES | YES(current selection) | YES(local/remote scopes) | NO(native ownership) | 원래 lock의 모든 unique archive 독립 취득/검증 |
| 전체 instance/edge/bin/code file closure — PNPMInspectionInput.BindLayout | YES | YES(guest inspection) | YES(existing scopes) | NO(Host original) | 아래 guest/Host 의미를 분리한 좁은 재사용 |
| native old-source complete proof | NO | NO | NO | NO | typed Host view/independent source closure/guard·negatives·actual |

기존 구현을 대체하지 않는다. 이름/version/lock 선언·unmanaged marker·새 버전 검사 성공만으로
원래 native tree 삭제 권한을 만들지 않는다. 첫 지원은 fixed PNPM isolated layout의 bounded
regular single-link copies와 안전한 mode/owner·exact internal links다. Foreign/shared hardlink,
unsafe modes/특수 파일·unknown namespace·corrupt prior state는 adopt로 우회하지 않는다.

## 현재 확인한 재사용 경계

1. **실제 원본 guard가 먼저다.** Begin은 same npm lock과 original controls/member census를
   고정한 뒤 explicit native proof를 수행한다. Independent state가 있으면 기존 검증만 사용하고
   corrupt state를 native fallback으로 바꾸지 않는다. Proof/selection/inspection 실패는 original을
   변경하지 않는다. 원본 전체 proof에는 새 selection에서 제외될 패키지도 포함한다.
2. **현재 guest 계약은 유지한다.** ParsePNPMLayoutArchive와 pnpmInspectionEntryMode는
   guest UID/GID1000을 요구하며 BindLayout.expectedStoreRoot는 caller-owned inspection guest
   path다. 원본 .modules.yaml의 storeDir을 expectedStoreRoot로 그대로 넣거나 Host UID를
   1000으로 위장해 기존 검사에 통과시키지 않는다. Guest admission/whole coverage는 불변이다.
3. **Host original은 별도 unverified type이다.** 실제 pinned census의 UID/GID를 그대로 담는
   private-data wrapper를 고려한다. Existing bounded archive parser의 기계 부분만 expected-owner
   인수로 좁게 추출하고 기존 public guest parser는1000을 계속 사용한다. Host wrapper는 현재
   inspection/prepare/cache에 전달할 수 있는 PNPMInspectionLayout을 반환하지 않는다.
4. **Source file closure만 두 실제 소비자가 공유한다.** Current BindLayout과 original ownership
   validator가 같은 archive file/instance/edge/bin/metadata 전체 비교를 사용하게 한다. 고정 guest
   store assertion은 current caller에 그대로 남긴다. Host metadata의 canonical store 경로는
   정보일 뿐이며 그 경로·외부 store를 열거나 follow하지 않는다. Freshness cache는 present일
   때 bounded generated JSON으로 accounting하고 ownership/Policy authority로 쓰지 않는다.
5. **독립 source는 실제 adapter가 공급한다.** Bootstrap의 공개 resolver가 원래 manifest/lock의
   exact versions를 다시 독립 binding하고 모든 old archive를 Acquire·integrity verify·정적
   inventory로 읽는다. Promotion에 inspection/verification package import를 추가하지 않는다.
   Promotion이 자기 early census에서 읽은 immutable Host view만 trusted callback에 전달하고
   proof 전후 원래 bytes/identity/absence를 재확인한다. Runtime/native Host 실행은 없다.
6. **현재 선택은 새 검역이다.** 원본 proof는 ownership만 설명하며 새 ALLOW/새 Session이나
   current cache/layout 승인이 아니다. Native adoption 첫 반영은 새 clean prepared payload와
   fresh operation/state를 사용한다. 관리 상태가 생긴 뒤의 no-op만 기존 최적화를 사용한다.

## 두 번째 검토 전에 남은 정확한 지도

Host metadata와 current guest-only parser의 공통 부분, optional freshness file/empty original의
명시 coverage, UID/GID wrapper와 per-entry mode checks, callback의 frozen input/proof 결과,
전체 old archive acquisition/workspace budget·실제 기록과 취소/최초 cause를 더 대조한다.
StoreDir 같은 producer claim은 source authority가 아니며 raw metadata를 고쳐 current guest
증거로 위장하지 않는다. Source fingerprint만으로 unknown original을 삭제하지 않는다.

수정 전 위 symbol의 exact edit map·두 번째 검토를 완료한다. Native copy positive와 implicit
adopt/old archive 변조·metadata identity 위조/추가·삭제·hardlink/escape/guard drift·새 Policy 실패/
rollback unknown의 정상·negative, actual public single/peer·기존 managed/fresh 소비자와 cleanup을
확인한다. Canonical profiles/관련 전체 Race·source-bound actual 이전에는 COMPLETE로 표시하지
않는다. CI407은 이 후속 source를 포함하지 않는다.

후속 실제 타이머 확인에서 앞 e9ca85e/CI407은 같은 전체tree의 Required10SUCCESS와 cache/
transaction·원복 소비자를 원격 검증했다. [CI407 범위](./p2c5b-ci407-qualification-20261025.json)를
prerequisite로 사용하되 CLI와 이 native original proof는 별도 source·acceptance다.

## Source 수정 전 exact edit map / 두 번째 검토

- artifact/npm/pnpm_layout_archive.go: bounded parser의 owner 인수를 private helper에만
  추출한다. 기존 두 public guest parser는 UID/GID1000 고정이다. 새 pnpm_original_layout.go의
  PNPMOriginalLayout은 private archive/실제 UID·GID만 담고 inspection layout getter가 없다.
- pnpm_layout_metadata.go: 모듈 format/graph 검사를 private base로 추출한다. 기존 guest
  wrapper는 caller-owned storeRoot+/v11을 그대로 강제한다. Original wrapper는 canonical
  absolute /v11 metadata 경로 형식만 확인하고 해당 경로를 열지 않는다. Optional freshness
  cache도 존재하면 기존 bounded JSON 검사를 거치며 삭제 권한이나 Policy로 쓰지 않는다.
- pnpm_inspection_layout.go: 전체 파일/instance/edge/bin/metadata closure만 private helper로
  추출해 guest와 original 두 실제 consumer가 공유한다. 기존 guest의 required cache·1000
  mode·discard·manifest·fingerprint 순서는 유지한다. Original은 actual owner/safe mode와
  single-link copy transport를 요구하고 현재 execution/cache layout을 반환하지 않는다.
  Original nonempty public isolated layout만 첫 lane이며 empty native metadata의 미검증
  모양은 추측해 허용하지 않는다. Existing dependency-free fresh/managed 경로는 유지한다.
- promotion/pnpm_original.go: early pinned census에서 actual UID/GID·bytes/mode/link를
  bounded tar로 읽는다. 매 entry와 guard를 재확인하고 callback 전후 VerifyUnchanged를
  수행한다. NewPNPMProjectPromotionWithOriginalVerifier는 기존 constructor를 재사용한다.
  Begin은 state absent + explicit adopt + node_modules만 callback으로 보낸다. Unknown
  .heliopause, corrupt prior state, unsupported ownership, missing original lock는 거부한다.
  Independent state에 original proof digest를 optional field로 기록하되 재승인에 쓰지 않는다.
- bootstrap/pnpm_original.go: 원래 controls/lock을 pinned parser로 읽고 공개 Resolver의
  BindPNPMLock→InspectionContext→모든 old unique archive Acquire→IntegrityVerifier→
  static.InspectArchive→workspace incremental budget→original full closure를 수행한다.
  선택에서 사라질 old package도 빠뜨리지 않는다. 전체 compressed acquisition 합계를
  128MiB로 제한한 뒤 기존 expanded/peer budget을 그대로 소비한다. 원본 proof는 static
  ownership이며 현재 검사·Evidence·Policy를 생략하지 않는다. 새 selected session은 별도다.
- promotion/pnpm_transaction.go 및 bootstrap/pnpm_project.go: 위 callback의 실제 연결만
  추가한다. Retained proof 실패를 native fallback으로 바꾸지 않고 기존 transaction/rollback을
  재사용한다. Native 첫 adoption에는 previous가 없어 기존 clean prepared payload를 사용한다.
- artifact/promotion/bootstrap tests: actual owner≠1000, guest 불변, full peer closure/extra/
  missing/altered file·link/bin/metadata와 source mismatch, implicit adoption, prior state,
  callback 실패/취소/drift, original lock 부재 및 rollback에서 원본 보존을 검증한다.

두 번째 검토: 수정 범위는 missing native ownership acceptance뿐이다. 일반 layout engine이나
transaction을 대체하지 않는다. Host owner를 guest1000으로 바꾸거나 unsafe directory chmod,
storeDir trust/외부 store follow, original proof를 current ALLOW로 승격하는 경로가 없다.
Original SRI와 static source inventory가 full closure의 입력이며 fingerprint만으로 증명하지 않는다.
초기 nonempty copy lane 밖의 native empty/shared hardlink/다른 layout은 명시적으로 거부한다.
Local/remote 실제 source와 검증 결과 이전에는 qualified/complete로 표시하지 않는다.

## 구현·named local 검증 결과

Exact missing gap만 연결했으며 [local receipt](./p2c5b5d-pnpm-native-ownership-local-20261025.json)가
최종 source inventory·canonical/일곱 package Race/checker Race·실제 세 native 사례와 기존
CLI9operation/npm·Darwin compile·실패·정리의 원본을 소유한다. IMPLEMENTED=YES / WIRED=YES /
QUALIFIED=YES(named local) / ACCEPTANCE_CLOSED=NO(whole P2/remote)다.

Original wrapper의 실제 owner와 current guest1000을 분리하고 whole file closure를 좁게 재사용했다.
Bootstrap은 old metadata/SRI/정적 archive 전체를 다시 측정하며 원본 proof는 independent state의
optional provenance다. Existing transaction/rollback/current Policy를 대체하거나 import map을
완화하지 않았다. First native adoption은 clean prepared payload만 게시한다. Exact6 update는6을
유지하며 explicit add7이 원래6 payload를 대체한다. Native peer4도 full proof 후 현재 검역한다.

첫 actual24.126s는 fixture supervisor와 production Run의 owner 수명 충돌이었다. V2의41.348s는
새 test의 JSON field tags/정확 버전 native 의미 기대가 잘못됐고 post-state/peer는 미도달이었다.
원본을 보존하고 test 수명·기대만 교정한 V3 actual128.739s/126.33s는3PASS다. 제품 권한이나
observer exclusivity를 바꾸지 않았다. Native V3 뒤 production 변화는 inert help/flag 문구뿐이며
최종 CLI 회귀는 그 문구를 포함한 새 binary다. Race와 actual의 source 경계를 receipt에 명시한다.

현재 MISSING은 새 전체 source의 remote Required/native macOS·CI project caller 전체와
영향받는 CPU/세CUDA, 최종 P2 support/acceptance다. 앞 CI407의 backend 성공을 이 source에
전용하지 않는다. Unsupported native empty/shared-link/unsafe variant를 성공으로 표시하지 않는다.
