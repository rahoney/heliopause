# P3-B4d-d Bun complete inspection·Policy 연결

기준e212fa491462fafbdc9f71ec5cac2dd8caf2b94a/treea72b85c298e6daeca3d3c25a9e2d046b9a2cf254,
2411개 전체 후보다. M13-004 하나만 IN_PROGRESS다. Read: B4d-a/b/c 지도·영수증,
기존 PNPMProjectInspector/pnpm_dynamic 및 public quarantine actual, package-client 계약§1–5,
Architecture inspection/sandbox/policy·Domain evidence/inspection·fail-closed 계약과 lessons.

## Baseline audit

| 책임/근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun source/context/budget/layout/prepare/instance backend, e212fa4와 선행 commit | YES | YES(actual) | YES(named local) | YES(Node-lane backend subset) | complete-project inspection consumer |
| npm static archive Inspector·PNPM complete inspector/coverage·Policy service | YES | YES(existing scope) | YES(existing consumers) | YES(existing scope) | Bun typed context/instance adapter |
| Bun whole static-all/dynamic coverage·Policy 연결 | NO | NO | NO | NO | 본 단계의 typed adapter·모든 source/instance·실제 Policy/evidence |
| Bun explicit runtime/cache/transaction/CLI·전체 remote | NO | NO | NO | NO | 이 검역 연결 뒤 후속 단계 |

## Exact edit map와 두 번째 검토

1. `internal/inspection/npm/bun_project.go`: PNPM의 검토된 orchestration 순서를 Bun의 별도
   typed context/input/layout/instance에 적용한다. 기존 PNPM adapter를 교체하지 않는다.
   같은 static Inspector/공유 budget engine과 complete native preparer/backend를 사용한다.
   Complete snapshot/acquired binding 뒤 unique archive 전부를 정적 검사하고 finding/불완전/
   budget 실패 때 runtime을 시작하지 않는다. 모든 logical instance의 target binding·fresh
   session·완료/Resource/취소 검사를 적용하고 부분 facts와 required coverage를 분리한다.
   동시 실행/이전 요청의 prepared handoff 재사용을 막고 exact request에만 반환한다.
2. `pnpm_dynamic.go`: 기존 facts/coverage/partial summary body를 현재 두 client의 닫힌
   prefix spec과 data-only digest/runtime getter helper로 재사용한다. PNPM의 check/evidence/
   limitation/schema 문자열은 보존한다. `bun_dynamic.go`의 typed wrapper가 Bun prefix를
   선택한다. 동일 관찰의 기존 Finding 해석과 count/summary 한도·원래 cause를 유지한다.
   새 Policy rule·완료 authority를 만들지 않는다. Private wrapper의 실제 caller도 확인한다.
3. `bun_dynamic_test.go`: 기존 partial/coverage·중복 session/불완전/취소/overflow·관찰 보존
   matrix를 Bun state에 적용하고 PNPM tests를 유지한다. `bun_project_inspection_test.go`의
   controlled source는 정적 gate 이전 runtime0·첫 cause·source 오류/finding/예산과 exact
   prepared handoff/재진입 거부를 확인한다. Fake source를 실제 인증으로 쓰지 않는다.
4. `bun_project_inspection_real_test.go`: fresh public transitive/peer source와 frozen Bun
   context로 기존 Application ProjectInspectService·IntegrityVerifier·evidence store·M3/M4를
   사용한다. 모든 archive와 instance의 static/dynamic/coverage·실제 기록과 entry/set Policy,
   dependency-only request의 primary 부재·exact prepared handoff를 확인한다. 기존 PNPM
   public quarantine도 같은 source/unchanged observer로 회귀한다.

Typed client별 frozen 계약을 별도 adapter에서 유지하므로 PNPM context를 Bun으로 위조하거나
기존 PNPM orchestration을 범용 runtime protocol로 재작성하지 않는다. 실제 archive decoder,
finding 해석·Policy·evidence store와 source/graph validation은 기존 owner를 재사용한다.
Unknown/incomplete/부분 성공은 ALLOW 근거가 아니며 script-free public 표본의 ALLOW를 명시
Bun runtime 또는 Host install 승인으로 확대하지 않는다. C++/runtime pin/FS/network 권한은
변경하지 않는다. 해당 backend 묶음 뒤 필요한 CI를 한 번 시작하고 실제 타이머로 확인하며,
그동안 후속 독립 작업을 계속한다. 다음 macOS/Required 성공 gate는 유지한다.

최초 compile/Race와 Quick27.793s는 adapter에서 가정한 BunTreeInstance 이름이 존재하지
않아 build 단계에서 종료됐다. 실제 InstancesFor의 반환형 BunLockedPackage를 API에서
확인하고 typed state/fixture를 고쳤다. 기존 parser/source type을 새로 만들지 않으며 최초
소스와 compile/Quick 원본을 보존한다. 변경은 Root에서 수행하고 private 첫 후보 검사가
종료된 뒤 projection하므로 검사 도중 private source를 바꾸지 않는다.

최종 [로컬 영수증](./p3b4dd-bun-project-inspection-local-20261025.json)의 actual160.202s는
Bun transitive/peer의 모든 archive/instance·실제 기록·entry/set Policy ALLOW47.05s와 기존
PNPM public 선택8/검역4(110.56s)를 연결한다. Whole reports의 필수 integrity/static/lifecycle3과
instance facts·exact prepared handoff/partial 요청 거부, helper wait0/Host 전체 동일을 확인했다.
Quick149.369s/related Race24.431s가 PASS다. PNPM 세 facts body의 prefix/limitation을
고정한 정규화 동일성도 확인했다. Node-lane backend의 IMPLEMENTED/WIRED/QUALIFIED(named
local)는 YES, 전체 Bun acceptance는 NO다. Next는 frozen 공급/actual CI wiring과 backend
묶음의 원격 checkpoint이며 대기 중 후속 cache/transaction/CLI를 계속한다.
