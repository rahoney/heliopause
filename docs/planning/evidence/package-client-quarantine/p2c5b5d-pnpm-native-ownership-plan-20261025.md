# P2c5-B5d — native PNPM 원본 소유권의 독립 source proof 착수 감사

Baseline a03ff13/tree50d52f2f5262406f648e4bd2fc49ebe37a700060. M13-004/P2 하나만 진행한다.
앞 e9ca85e의 CI407/run40128032402는 진행 중이며 첫 확인2026-10-25 06:02:54 KST다.
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
