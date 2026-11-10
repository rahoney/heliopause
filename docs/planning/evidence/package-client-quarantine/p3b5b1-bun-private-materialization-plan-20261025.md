# P3-B5b1 Bun private materialization 감사·수정 지도

기준은 7939092d4f0ac6bff4f3750e21217ed531f53dfe이며 B5a named local cache 검증을 마친 뒤 착수한다.
단일 active item M13-004의 다음 순서다. Read: B5a audit/최초 실패/actual·PNPM
`pnpm_materialize.go`, `pnpm_materialize_test.go`, `pnpm_owned_member.go`·Architecture staging/
promotion, Domain verified set/operation context 및 INDEX 라우팅 artifact/evidence.

| 책임/실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 승인 tar 읽기·hardlink를 독립 regular copy로 변환·internal symlink target 검증, pnpm_materialize.go hostEntries | YES | YES(PNPM) | YES(existing scope) | YES(existing scope) | Bun typed cache·absent lock/provenance 선택 |
| fresh sibling·descriptor/inode/owner/mode/full census/readback·sync·보수적 cleanup, PNPMPreparedProject | YES | YES(PNPM) | YES(existing scope) | YES(existing scope) | Bun typed Prepare consumer |
| Bun private prepared project | NO | NO | NO | NO | approved cache→private copy/Verify/Close·actual 소비자 |
| Bun retained ownership/target transaction/CLI | NO | NO | NO | NO | 다음 단계; 현재 Host target publication은 하지 않음 |

## Exact edit map·두 번째 검토

1. `pnpm_materialize.go`: typed PNPM `hostEntries`의 cache 승인 호출을 유지하고 그 뒤의
   receipt/digest·bounded tar→Host entry body를 private 두-client helper로 추출한다.
   Controls와 record builder closure는 typed caller가 전달하며 fixed provenance filename만 enum에서
   선택한다. Bun은 Present control만 물리 entry로 전달한다. `ParsePNPMLayoutArchive`의 공용
   bounded transport parser를 재사용하고 Host에서 source tar를 별도 해제하지 않는다.
2. 같은 파일의 Prepare body를 private helper로 추출한다. 기존 PNPM public 타입/메서드·all
   owner identity/census·beforeRemove/late failure·sync/cleanup은 그대로 재사용한다. Client별
   sibling prefix만 고정 enum으로 선택한다. 기존 wrapper caller를 전부 대조해 U1000을 피한다.
3. `bun_materialize.go`: typed Bun cache 승인과 record closure로 common Host entries/Prepare를
   호출한다. `BunPreparedProject`는 private PNPM ownership engine을 보유하고 Directory/Verify/
   Close만 위임한다. Public PNPM type alias나 두 client 간 interchangeable 승인 타입을 만들지
   않는다. 새 temporary object는 이 단계에서 실제 Prepare/검사 소비자가 즉시 사용한다.
4. `bun_materialize_test.go`: normal/explicit empty·Absent lock·full records·target 부재,
   independent file inode, data/mode/link/extra drift, changed parent, cleanup 중 bytes 변경,
   cancellation/late failure의 원래 cause와 owned-only cleanup을 확인한다. Existing PNPM full
   materialize/transaction tests를 유지한다. Actual bootstrap Bun consumer에서 Prepare/Verify/
   Close와 target 부재를 확인한다. 공통 body의 기계 정규화 대조와 전체 후보 checks를 남긴다.

기존 구현을 대체/재작성하지 않고 검증된 body를 재사용한다. Native install/재resolve/Artifact
실행·Host target mutation·network/observer grants는 추가하지 않는다. Bun의 source TGZ나 native
cache pathname은 approval이 아니며 cache의 current recorded Evidence를 먼저 재조회한다.
CI412가 실행 중이면 새 push 없이 로컬 구현/검사를 진행하고 실제 due의 결과 조회를 소비한다.
