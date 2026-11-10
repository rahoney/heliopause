# P3-B4d-c Bun production instance backend 연결

기준3f328e27f09d2d25a2b756f95f245502a565dafb/treebd87e1bc4befc92dfb010dde91943f5aef7399c8,
2406개 전체 후보다. M13-004 하나만 IN_PROGRESS다. Read: B4d/B4d-a/B4d-b 지도·영수증,
기존 `pnpm_instance_backend.go`(최초 owner a0c9eb6)·tests/actual, `Backend` lifecycle와 payload,
Architecture/Domain inspection 계약 및 lessons sandbox L08/L12/L18·evidence L11/L16·ci L06.

## Baseline audit

| 책임/근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun complete source/layout/prepare·payload, 4420b48/da02f4e | YES | YES(named actual) | YES(named local) | YES(data subset) | production execution consumer |
| Bun exact target/typed topology/creator-one-shot, 3f328e2 | YES | YES(actual primitive) | YES(named local) | YES(primitive subset) | opaque layout에서 target 선택 후 전체 session lifecycle 연결 |
| PNPM Backend.ExecutePNPMInstance, a0c9eb6 | YES | YES | YES(existing named actual) | YES(existing scope) | 세션/정리 body를 Bun typed consumer에 재사용 |
| Bun Backend.ExecuteBunInstance | NO | NO | NO | NO | 본 단계의 typed preflight·session·bound payload·actual |
| Bun complete inspection/Policy·명시 Bun executable/cache/transaction/CLI | NO | NO | NO | NO | backend 뒤 원래 owner에 순차 연결 |

## Exact edit map와 두 번째 검토

1. `internal/sandbox/pnpm_instance_backend.go`: 기존 public typed preflight/target/payload 선택을
   유지하고 session 생성 이후 body를 private current-two-kind engine으로 기계 추출한다.
   PNPM caller의 고정 volume/root·typed trace starter·target command 의미를 유지한다. Capability,
   start/boundary/anchor/sole attachment·whole copy/readback/null-network proof, timeout, cleanup,
   trace/diagnostic·원래 cause·INCOMPLETE code/관찰·완료 fact body를 재사용한다.
2. `bun_instance_backend.go`: opaque Bun layout의 runtime과 exact acquired subject/instance를
   도구 호출 전에 검사한다. 검증된 layout.Target과 controller-owned payload만 private engine에
   전달한다. Bun fixed volume/root·typed trace starter와 exact command를 선택하며 다른 client의
   payload/target를 소비하지 않는다. Session/volume/container를 target마다 새로 만들고 결과는
   cleanup과 전체 trace 완료 뒤에만 반환한다.
3. `pnpm_instance_backend_test.go`의 상태 기반 Docker fixture를 fixed kind/root/command만
   선택하는 두 번째 test consumer에 재사용한다. 기존 PNPM tests의 기본값·의미는 유지한다.
   `bun_inspection_payload_test.go`의 fixture에서 같은 typed input을 반환하는 helper를 추출한다.
   새 `bun_instance_backend_test.go`는 fresh resources/관찰·정리, zero/foreign/cancel/transport/
   observer preflight0, setup/copy/readback/network/command+cleanup/trace 실패와 최초 cause를
   검사한다. Fake transport는 실제 native/Policy evidence가 아니다.
4. `bun_instance_backend_real_test.go`: fresh public peer source4/SRI/static→production Bun
   prepare→모든 exact instance를 production backend의 fresh session에서 관찰한다. 기존 PNPM
   production project와 같은 source/observer에서 회귀한다. Explicit synthetic script scope가
   필요하면 public source 승인과 분리한다. Artifact attribution/금지 관찰을 숨기지 않는다.

기존 PNPM 구현을 대체/재작성하지 않고 body와 기존 owner를 재사용한다. Private input/trace
starter는 현재 두 typed 소비자만 가지며 범용 외부 실행 API가 아니다. Sandbox는 raw result만
만들고 Policy/안전성을 판단하지 않는다. 현재 lane은 고정 npm/Node lifecycle이며 명시 Bun
runtime script는 후속 검증이다. C++/runtime pin/FS/network grant는 이 연결에서 변경하지
않는다. 같은 C++119latches와 byte-identical canonical observer는 재사용한다.
Docker Host 전후 snapshot을 검사하는 actual과 Docker compiler를 겹치지 않는다.
Helper별 CI는 반복하지 않고 whole 검사/Policy backend 연결 후보에서 묶는다.

첫 Quick72.605s는 공용 create argument helper로 옮긴 뒤 caller가 사라진 private
`pnpmInstanceCreateArguments` wrapper의 U1000에서 종료됐다. Public ExecutePNPMInstance와
기존 session body는 유지하고 사용되지 않는 private wrapper만 제거한다. Dummy 호출이나
rule suppression을 넣지 않는다. 정규화 body 동일성과 첫 소스/실패 원본을 보존한다.

최종 [로컬 영수증](./p3b4dc-bun-instance-backend-local-20261025.json)은 opaque layout/source
peer4의 production fresh sessions4(31.41s)와 기존 PNPM project2/4 sessions45.02s를 연결한다.
Combined actual79.128s/helper wait0/Host 동일과 Quick161.883s/전체 sandbox Race25.135s가
PASS다. 기존 session body의 정규화 동일성을 대조했으며 새 C++/runtime/FS/network 권한은
없다. Typed backend의 IMPLEMENTED/WIRED/QUALIFIED(named local)는 YES, 전체 Bun acceptance는
NO다. Next는 원래 Inspection owner에서 complete static-all/dynamic 집계·Policy를 연결한다.
