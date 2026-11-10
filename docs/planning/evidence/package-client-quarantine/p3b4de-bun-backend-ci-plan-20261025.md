# P3-B4d-e Bun backend CI 연결과 checkpoint

기준37fec98dba795c27d72845b0a219abb5940ed4d1/tree514957baa919a6700ea798f4316094ee06f6e915,
2418개 전체 후보다. M13-004 하나만 IN_PROGRESS다. Read: 확장 계획§5, package-client 계약,
B4d-b/c/d 영수증, ci/evidence/sandbox lessons와 기존 CI supply/부모 cleanup·workflow validators.

## Baseline audit

| 책임/근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun Node-lane selection/source/whole quarantine/Policy, 37fec98와 선행 commits | YES | YES(local actual) | YES(named local) | YES(local subset) | remote exact-source 소비자 |
| Bun frozen archive/materialize-bun·generated supply pin | YES | YES(local/release) | YES(named local) | YES(supply subset) | hosted CI의 일시적 protected native 공급 |
| owning CI client/helper lifecycle·PNPM supply와 strict active workflow 검사 | YES | YES | YES(existing scope) | YES(existing scope) | Bun supply/actual child 연결 |
| Bun backend remote qualification | NO | NO | NO | NO | 본 단계의 script·active workflow·실제 consumer·macOS/Required |

## Exact edit map와 두 번째 검토

1. `scripts/ci-bun-runtime.sh`: 기존 hosted-CI-only PNPM 공급 패턴을 Bun exact pin과 이미
   검증된 materialize-bun에 적용한다. 사전 installation/extra argv는 거부하고 exact HTTPS
   archive 한도/전체 pin·native source를 확인한다. 새 owned CI version의 readonly native만
   공급하고 child는 helper/client 교체·service 재시작을 하지 않는다. Primary failure를
   보존하며 자신이 만든 fixed CI files/directories와 task work만 정리한다.
2. `.github/workflows/heliopause-ci.yml`: 기존 PNPM project의 strict 전환 sequence와 sandbox
   restoration/helper 시작을 유지하고 그 다음 `bash scripts/ci-bun-runtime.sh`를 연결한다.
   기존 job/Required/timeout/coverage는 바꾸지 않는다. 새 helper별 CI job을 만들지 않는다.
3. `scripts/check/bun_runtime_workflow.go`·test와 기존 workflow hook: exact materializer,
   non-root/absence/cleanup·실제 native/whole-quarantine filter/환경과 package subshell을
   검사한다. Active YAML run의 같은 owning lifecycle에서 sandbox restoration 뒤 실행되는지
   확인하며 comment/echo/빠진 env/filter/실패 무시/중복/잘못된 순서의 반례를 검증한다.
   기존 PNPM validator/body는 재작성하지 않는다.
4. Local qualification은 같은 script를 protected client/observer가 보이는 private mount
   namespace에서 실행한다. Host 실제 설치를 바꾸지 않으며 Docker compiler와 겹치지 않는다.
   Canonical Quick/security/docs/workflow·관련 Race/vuln과 exact whole candidate를 확인한다.

Helper는100644로 보존하고 Bash로 호출한다. Runtime/source/Policy/FS/network 권한은
바꾸지 않는다. 같은 backend의 여러 local commits를 한 번 push하여 필요한 ordinary CI를
시작한다. Run/head/전체 source·최근 시간에 근거한 첫 실제 timer를 기록하고 active turn에서
clock 대기→결과/전체 원본 확인→후속 작업을 이어간다. 기다리는 동안 independent approved
cache/transaction/CLI 작업을 계속하며 새 push로 실행 중 candidate를 취소하지 않는다.
macOS/Required 성공 전 gate를 닫지 않고 whole Bun/runtime/Host acceptance로 확대하지 않는다.

첫 workflow2.155s는 PASS였고 Quick78.954s의 default test는 runtimeOwnerFixture의 copy
목록에 새 ci-bun-runtime.sh가 없어 coordinated future Go positive를 거부했다. 실제 main
workflow/supply가 누락된 실패가 아니며 기존 fixture의 source inventory에 정확히 그 파일을
추가한다. Validator나 negative는 완화하지 않고 첫 fixture/Quick 원본을 보존한다.

최종 [로컬 영수증](./p3b4de-bun-backend-ci-local-20261025.json)은 동일 CI script142.666s의
고정 archive/materializer 공급·native9/실패5/검역2와 namespace 설치/work 정리·helper wait0/
Host 전체 동일을 확인한다. Workflow2.407s/Quick78.075s/checker Race33.607s/vulnerability
14.984s가 PASS다. Backend CI 연결의 IMPLEMENTED/WIRED/QUALIFIED(local)는 YES이며 remote
및 whole Bun acceptance는 NO다. Next는 묶음 push와 실제 timer/원본 확인이다.
