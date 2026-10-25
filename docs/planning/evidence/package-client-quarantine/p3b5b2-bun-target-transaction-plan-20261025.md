# P3-B5b2 Bun managed target transaction 감사·두 번째 검토

M13-004의 다음 순서이며 B5b1 local qualification/중간 commit 뒤 source를 수정한다.
Read: B5a/B5b1 지도·영수증, `pnpm_transaction.go`, `pnpm_transaction_profile.go`,
`pnpm_approved_transaction.go`, `pnpm_retained.go`, `npm_transaction.go`와 member owner,
Domain project_update·Application guarded update 및 Architecture staging/promotion 계약.

| 책임/기준 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| same-target shared guard·original census·independent retained approval, PNPM promotion | YES | YES(PNPM) | YES(existing scope) | YES(existing scope) | Bun fixed lock/marker/cache/provenance 선택 |
| backup/publish/sync/rollback/recovery journal, npmProjectTransaction | YES | YES(NPM/PNPM) | YES(existing scope) | YES(existing scope) | Bun typed profile와 explicit absent output |
| Bun current approved cache/private owned engine | YES | YES | YES(B5a/B5b1 named local) | YES(local subset) | target mutation 소비자 |
| Bun typed target transaction | NO | NO | NO | NO | guarded Commit·state·no-op·rollback·fresh records |
| explicit selected absent control, Domain ProjectDependencyUpdate | NO | NO | NO | NO | 별도 snapshot-only presence constructor; 기존 constructors guard 유지 |
| native/unmanaged Bun adoption·CLI·explicit Bun runtime | NO | NO | NO | NO | 이후 단계, native payload는 이 단계에서 거부 |

## Exact edit map

1. `project_update.go`의 기존 constructors는 selected-present guard를 유지한다. 새
   `NewProjectDependencyUpdateFromSnapshotWithControlPresence`만 같은 private binder의
   explicit control presence 경로를 사용한다. All names/count/digests/full snapshot/zero-primary/
   deep-copy 검사를 유지한다. Absent가 body를 가질 수 없다는 기존 Domain invariant는 그대로다.
   기존/새 constructor의 absent 거부/허용·잘못된 control/digest/중복/foreign 회귀를 확인한다.
2. 기존 PNPM promotion owner/guard·profile body를 재사용한다. Private fixed client enum과
   Bun typed selected cache callback만 추가하며 public PNPM factory/원본 proof 경로는 유지한다.
   Bun wrapper는 typed cache roots/operation/snapshot/controls를 전부 대조하고 기존 owning
   guard·prepared file engine을 사용한다. Fake PNPM frozen context/cache를 만들지 않는다.
   Cache preflight 뒤 Open→actual retained Evidence→Prepare 순서를 보존한다.
3. `pnpm_retained.go`는 기존 record/digest/Store 재조회 body를 공용 private helper로
   재사용한다. Bun은 exact package.json/bun.lock control 종류와 empty-only absent lock을
   추가 검증하고 `bun-provenance.json`을 요구한다. Save state·target marker만으로 새 ALLOW나
   current inspection을 만들지 않는다. Native proof callback 부재에서는 unmanaged payload를
   계속 거부하며 이 단계에 placeholder proof/parser를 추가하지 않는다.
4. `npmProjectTransaction`의 backup/publish/rollback engine과 순서를 재사용한다. Profile의
   fixed lock/marker 이름만 선택하고 기존 PNPM absent-original 조건을 그대로 보존한다. Bun
   selected absent lock만 workspace/target 양쪽 부재를 확인하여 publish를 생략한다. 부재 member는
   independent state에 명시적인 고정 absence digest로 저장하며 commit 직전/재개 때 Lstat 부재를
   다시 확인한다. 기존 nil member를 일반 성공으로 바꾸거나 PNPM lock deletion을 허용하지 않는다.
5. Typed Bun public factory/Begin/Commit/Close/PublishedRecords, nonempty/empty·첫 install/
   no-op·control mode/inode 보존·별도 state/Evidence/cache 변조 거부·late failure rollback·
   foreign journal/data 보존·다른 client lock 경합을 확인한다. Existing PNPM/NPM 전체 transaction
   회귀와 canonical checks를 먼저 사용한다. Actual bootstrap에서 fresh native selection→검역/
   current Policy/cache→target publication과 다음 no-op의 fresh records를 확인한다.

이는 기존 구현 교체가 아닌 닫힌 두-client parameter 연결이다. 새 source registry·Policy·observer/
network/filesystem grants는 없다. Bun empty lock을 제조하거나 source native cache를 승인으로
사용하지 않는다. Artifact 실행 없이 검역한 정확한 bytes만 publish한다. CI412 frozen7544
타이머는 active turn에서 소비하고 작은 변경별 push/CI는 하지 않는다. Bun native adoption와
CLI/whole P3는 named transaction 검증만으로 완료로 표시하지 않는다.

74102c8의 B5b1 전체 후보901df033/2434files에서 착수한다. Actual 소비자를 production guarded
workflow로 연결하기 위해 `application/bun_project.go`의 typed 서비스와
`bootstrap/bun_project.go`의 request-local typed session을 추가한다. 기존
`executeGuardedProjectUpdate`/ProjectLockUpdateResolver/ProjectMutation ports를 재사용하고
PNPM source service는 바꾸지 않는다. Bun session은 정확히 한 native 선택과 frozen context,
actual operation을 유지하고 초기 guard→선택→검역→Stage→Commit 순서를 강제한다. 이 단계의
실제 consumer가 즉시 사용하는 composition이며 CLI factory는 이후다. Native adoption callback은
추가하지 않는다. `bun_transaction_test.go`, Domain presence 회귀와
`bootstrap/bun_transaction_integration_test.go`의 fresh managed peer/empty·두 번의 실제 Apply와
fresh operation/Evidence/no-op inode를 확인한다. 새로운 approval gate나 source registry는 없다.

수정 전 caller 대조에서 native Bun controls는 package.json/bun.lock 순서지만 Domain update는
name 순서로 정렬한다. Bun의 original/selected 비교만 이미 존재하는 order-independent
`sameUVSyncControls`를 exact2 length와 함께 재사용하고 PNPM의 기존 순서 비교는 유지한다.
Bytes/presence/digest를 생략하거나 source producer의 immutable controls 순서를 변경하지 않는다.

## 첫 actual no-op과 보완 지도

Actual43.340s는 Bun single의 첫/둘째 guarded Apply가 완료된 뒤 payload inode 보존 assertion에서
실패했다. Helperwait0/Host 동일이며 peer/empty/PNPM actual은 미실행이다. 현재 Bun layout
fingerprint는 acquired ContentHandle을 포함하고 PNPM은 포함하지 않는다. 검역 source/run의 이
결합을 제거하거나 이전 ALLOW를 새 run에 전용하지 않는다. 같은 Graph/Runtime 아래 Bun의
새 prepared node_modules 전체 census를 기존 independently approved original census와 기존
`workspaceMember`/`npmPNPMMemberDigest`로 대조하는 좁은 no-op 보완을 추가한다. Mode/UID/GID/
kind/link/size/digest/full paths를 모두 비교하며 둘이 같은 경우에만 기존 member를 retain한다.
기존 PNPM layout equality 경로는 유지한다. 기계 fixture에서 서로 다른 bound layout fingerprint
아래 같은 bytes의 inode 보존을 수정 전 RED/후 GREEN으로 확인하고 fresh actual을 새 binary/
namespace에서 재실행한다. 원본 실패 log/receipt는 수정하지 않는다.
