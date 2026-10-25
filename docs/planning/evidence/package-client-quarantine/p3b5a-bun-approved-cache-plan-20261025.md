# P3-B5a Bun 승인 data cache 연결

기준7544a0939512607325f858846c262e58b71e28b5/tree3b7dbd0343acc7c2b48eb2d6b2ed442d2a2f92f1,
2423개 전체 후보다. M13-004 하나만 IN_PROGRESS다. CI412는 같은7544a09에서 실행 중이며
첫 실제 확인은2026-10-25 13:16:17UTC/22:16:17KST다. 실행 중 source를 재push하지 않는다.
Read: B4d-d/e 영수증·확장 계획§5, PNPM B5a/B5b1의 approval/records 지도·구현,
Architecture evidence/staging·Domain verified set/records 계약·라우팅된 lessons.

## Baseline audit

| 책임/근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun complete quarantine·실제 records/Policy·exact layout handoff, 37fec98/7544a09 | YES | YES(actual) | YES(local), remote412 pending | YES(local subset) | approved staging consumer |
| bounded original intake reader·sealed cache inventory/sync/no-replace·Evidence 재조회 | YES | YES(PNPM/other clients) | YES(existing scope) | YES(existing scope) | Bun typed context와 instance prefix 선택 |
| PNPM approved cache·operation records/SBOM·native provenance | YES | YES | YES(existing scope) | YES(existing scope) | Bun hoisted instance graph·absent empty lock 표현 |
| Bun approved data cache | NO | NO | NO | NO | 본 단계의 Stage/Open·actual current Policy/Evidence/layout/data 결합 |
| Bun private materialization/retained approval/transaction/CLI | NO | NO | NO | NO | staging 뒤 별도 구현·실제 반입/rollback 검증 |

## Exact edit map와 두 번째 검토

1. `internal/promotion/pnpm_cache.go`와 `pnpm_cache_approval.go`: 기존 typed PNPM preflight를
   유지하고 공용 source 재해시·bounded tree renderer·recorded instance approval·sealed
   publication/open body를 private 두-client helper로 재사용한다. 기존 PNPM labels/receipt/
   provenance filename/root/limits·cause·cleanup은 유지한다. Private wrapper의 caller를 확인한다.
   Control·entries·required check·actual recorded Evidence·Session/layout/runtime/operation·Policy
   binding을 생략하거나 이름/단순 digest만으로 승인하지 않는다.
2. 새 `bun_cache.go`·`bun_cache_approval.go`: typed Bun context와 completed layout provider만
   소비한다. 같은 ProjectVerifiedSet에서 complete acquired runs를 다시 결합하고 모든 source
   bytes를 공용 intake reader로 재해시한다. Bun facts/coverage prefix와 모든 physical instance를
   실제 evidence에서 확인한 뒤에만 cache writes를 시작한다. Generic `project-cache` handle와
   별도 trusted cache root를 사용한다. Host action required/partial/foreign/stale를 거부한다.
3. `bun_cache_records.go`: 기존 ProjectRecordBuilder로 unique archives의 Manifest/SBOM을
   만든다. Bun tree key→unique node를 결합하고 실제 Targets의 instance edges를 투영한다.
   별도 native provenance가 모든 root/tree instance/edge를 보존하며 primary를 제조하지 않는다.
   Bun context의 기존 immutable Lock getter를 재사용하고 새 getter/parser는 만들지 않는다.
4. Empty는 explicit dependency-free context/approval만 허용한다. 부재 bun.lock은 receipt에
   absent control로 남기고 data file을 제조하지 않는다. Present controls·opaque node_modules
   data tar·records만 sealed regular files로 저장하고 어떤 link도 Host에 추출/실행하지 않는다.
   Inventory/readback·sync/renameNoReplace와 receipt/record/evidence 변조·extra/mode/link 거부를
   재사용한다. 전체128MiB data footprint와 기존 tar/receipt/records ceilings를 유지한다.
5. `bun_cache_test.go`와 shared PNPM 회귀: normal/empty·wrong snapshot/run/runtime/layout,
   actual record 누락/변조·중복/foreign Session·누락 coverage·source 변경·필수 Host action은
   writes0다. Stage/Open 후 receipt/data/record/mode/link/extra 변조와 취소/late sync failure의
   zero output/first cause/partial cleanup을 확인한다. Real bootstrap consumer는 fresh native
   selection/source→quarantine/evidence/Policy→Stage/Open을 연결하고 원본 재해시를 확인한다.

기존 cache·Evidence·record builder와 sealed publication을 교체/재작성하지 않는다. 공용
inventory/sync/transaction의 symlink 거부를 완화하지 않고 Bun native paths는 데이터로만
보관한다. 이 cache의 성공은 private Host materialization/ownership/transaction이나 명시
Bun runtime 지원을 뜻하지 않는다. 새 runtime/observer/network 권한·외부 CI는 시작하지
않는다. 현재 CI412의 실제 timer를 소비하고 independent local 작업을 계속한다.

## 부재 control의 record 경계 추가 검토

첫 Quick129.262s의 normal fixture는 고정 single lock6.0.0/name과 다른 manifest7.0.0을
주어 parser에서 먼저 종료됐다. 같은 raw의 empty는 실제 BuildProject의 all-present guard에서
종료됐다. 첫 실패를 보존하고 fixture를 원본 name/version으로 맞춘다. Empty 문제를 lock 파일
제조로 해결하지 않는다. 기존 `BuildProject`의 입력/출력/guard를 보존하고 같은 body를 private
helper로 추출한다. 새 `BuildProjectWithControlPresence`는 opaque Domain control의 실제 Present
상태와 exact snapshot digest를 모두 기록한다. Project 전용 control wire에 `absent,omitempty`를
추가하며 기존 present 문서 bytes는 그대로다. Bun cache의 builder interface만 새 메서드를
요구한다. Evidence는 생태계 parser/Policy를 대체하지 않으며 typed Bun context가 absence 의미를
검증한다. Existing Python/PNPM builder·Manifest wire/guard는 바꾸지 않는다. 정확한 absence와
present-zero-byte의 서로 다른 Manifest, 기존 메서드의 absent 거부, current operation/digest와
기존 deterministic records 회귀를 함께 검증한다. 수정 경로는 evidence/project_manifest.go·test,
Bun cache/builder/test, actual bootstrap consumer이고 미래 package/interface는 만들지 않는다.
