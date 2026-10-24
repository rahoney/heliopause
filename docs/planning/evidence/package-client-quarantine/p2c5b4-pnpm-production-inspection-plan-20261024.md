# P2c5-B4 — production inspection의 준비·소개·실제 instance 관찰

Baseline91008cb/treeedc48687, 같은 M13-004/P2다. [B3c의 실제 결과와 폐기 경계](./p2c5b3c-pnpm-layout-binding-plan-20261024.md),
[B1 §4](./p2c5b-pnpm-production-context-plan-20261024.md#4-b2-이후의-순서와-필수-수명coverage-경계),
[B2 §3/4/7](./p2c5b2-pnpm-target-lifecycle-plan-20261024.md), Architecture02/03·Domain08/12/13,
artifact L15/L20·sandbox L08/L12/L18·evidence L11/L16을 읽었다. 기존 단일 npm 경로와
ProjectInspectService를 유지하고 누락된 production closure만 연결한다.

## 1. Source 작성 전 baseline audit

| 책임 / 현재 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| static full archive inventory — Inspector.InspectArchive | YES | YES | YES(named local) | NO(P2) | production 전체 입력 static gate/집계 자원 gate |
| full native layout — PNPMInspectionInput.BindLayout | YES | YES(actual fixture) | YES(named local) | NO(P2) | production prepare owner에서 소비 |
| immutable native supply/typed network/owned volumes·observation — PNPMResolver.SelectProject | YES | YES(selection) | YES(existing scope) | NO(P2) | 같은 수명 owner의 prepare 작업 재사용 |
| store-add/fetch/detach/offline install — qualifyPNPMOfflineFixture | YES(test primitive) | NO(production) | YES(named local) | NO(P2) | production 구현과 그 소비자 검증 |
| exact target command/attested mount/CONTROL lineage — pnpm_target_lifecycle.go·observer.cc | YES | YES(synthetic actual) | YES(local/CI404 latches) | NO(P2) | verified layout의 fresh 실행·long hashed locator |
| single normalization — inspection/npm/dynamic.go | YES | YES(single) | YES(existing scope) | NO(P2) | 모든 actual instance/Session ID와 partial 사실 보존 aggregate |
| batch workflow — application/project_inspect.go | YES | YES(existing batch clients) | YES(existing scope) | NO(P2) | PNPM adapter 연결; Core에 PNPM schema 추가 없음 |
| Host layout/cache/provenance/records/transaction/CLI | NO(full PNPM) | NO | NO | NO | 이 inspection 결과 뒤 별도 정확한 지도·qualification |

## 2. B4a — 기존 native 수명 owner와 준비 동작

`sandbox/pnpm_resolver.go`의 선택 의미·NewPNPMSelectionInput/ParseOutput은 보존한다.
동일한 owned name/label, native supply, typed policy, observer/profile, volume attestation,
mount/sole attachment와 cleanup 절차를 복제하지 않는다. 현재 SelectProject의 해당 infrastructure
block을 private `withNativeProject`로 이동해 selection과 prepare 두 **고정된 내부 callback**이
공유한다. 정책/명령을 사용자 callback으로 받는 public API를 만들지 않는다. 기존 selection의
call-order/주입 실패/실제 소비자를 회귀한다. 알고리즘·Policy·지원 범위를 대체하는 작업은 아니다.

`sandbox/pnpm_inspection_prepare.go`의 `PNPMInspectionPreparer`는 기존 PNPMResolver와
DockerArtifactIntroducer를 조합한다. `PrepareInspection(ctx, input, inventories)`는 complete
exact coverage/runtime·자원 gate를 **tool/DNS/resource 작업 전** 검사한다. B3c의 값을 반환하는
유일한 production 경로다. Explicit empty는 project adapter에서 별도로 끝내며 primary나 빈
native layout을 제조하지 않는다. 취소 cause와 처음 실패를 cleanup/후속 오류와 함께 보존한다.

`artifact/npm`에 현재 PNPM inspection workspace의 고정128MiB를 한 곳에서 소유하게 하고
기존 createPNPMProjectVolume도 그 값을 소비한다. Complete inventories의 compressed bytes,
unique expanded store와 모든 peer instance의 logical expanded footprint·entry 수를 계산해
초과를 native 실행 전에 거부한다. 이후 static-all caller도 inventory를 추가하는 동안 unique
compressed/expanded/file 수를 누적해 중단한다. 모든 거대한 입력을 모은 뒤 한 번 검사하지 않는다.
Full runtime budget/observer ceiling을 늘리지 않는다. Native metadata·bin overhead로 실제 volume이
가득 차도 operational incomplete이며 allowance를 넓혀 통과시키지 않는다.

`sandbox/pnpm_inspection_transport.go`는 기존 handle 해석을 재사용하되 새 PNPM 전송에는
read-only OpenRoot·Lstat/open/SameFile·exact size·전체 digest 재확인을 추가한다. M3 Introduce의
기존 동작은 변경하지 않는다. 고정 guest artifacts/<digest>.tgz에만 bounded tar로 소개하고
readback SHA256을 대조한다. Handle/Host path는 guest 명령·출력 authority가 아니다.

공유 고정 command builder는 이미 실제 확인한 store-add → native fetch → owned network
disconnect/실제 endpoint0 → offline/frozen/ignore-scripts install 순서만 제공한다. 원래 registry
lock을 사용하고 native opaque store/index를 제조하지 않는다. Early PNPM_CONFIG_STORE_DIR와
XDG_CACHE_HOME/TMPDIR는 기존 A2의 정확한 private guest 경로에 고정한다. 원래 controls가
바뀌지 않았는지 재확인하고 native tar를 bounded capture→ParsePNPMLayoutArchive→BindLayout한다.
Native container 종료/complete 관찰·금지0/두 volume·typed policy 정리까지 확인한 **후에만**
값을 반환한다. Metadata scope 안에서 Artifact 실행을 시작하지 않는다.

현재 B4a의 구체 symbol은 `NewPNPMInspectionPreparer(resolver, intakeRoot)`와 위 PrepareInspection,
private `PNPMResolver.withNativeProject(ctx, earlyStore, action)`이다. 기존 SelectProject의
preflight/selection 의미는 wrapper에, 기존 infrastructure block은 후자에 둔다. `earlyStore`는
현재 두 고정 경로의 내부 bool이며 외부 config/pass-through가 아니다. `artifact/npm`의
`PNPMInspectionInput.CheckWorkspace(inventories)`는 기존 BindLayout의 exact inventory index를
private helper로 재사용해 runtime 작업 전 complete coverage와 고정 logical footprint를 검사한다.
BindLayout 자체의 format binding은 기존대로 유지한다. Metadata/bin의 추가 사용은 hard volume
ceiling 아래이며 추정값을 native 작업 성공 보장으로 쓰지 않는다.

## 3. B4b — 검증된 payload 소개와 독립 실행

`sandbox/pnpm_inspection_payload.go`는 bound Entries/ReadFile에서만 새 tar를 만든다. 원래
controls/native whole tar를 복사하지 않는다. package payload·검증한 내부 links/bins만 소개하고
.modules.yaml/virtual lock/freshness cache는 제외한다. Root는 InspectionManifest의 private
exact dependency/group만 사용한다. Regular/hardlink의 readback content·entry/mode/link를
검사하여 소개 뒤 drift를 거부한다. Source-intake bytes 재검증과 이 layout 소개 검증은 별개다.

`Backend.ExecutePNPMInstance(ctx, layout, subject, instanceKey)`는 기존 Execute를 유지하면서
B2의 검증된 createArguments·fresh project volume·StartPNPMTargetLifecycleProfile·mount anchors·
고정 target command/cleanup helpers를 사용한다. Target은 bound.Target(exact subject,key)에서만
도출한다. 매 expected instance마다 fresh 실제 Session ID/container/volume이며 실패한 session은
재사용하지 않는다. 수집한 observation과 원래 run/cancel/timeout/cleanup cause를 결과와 함께
보존한다. 한 실패가 이미 관측한 보안 사실을 없애지 않는다. CONTROL→ARTIFACT demotion과
예산은 그대로다. Long hashed dirname에 @가 사라지는 기존 locator gap은 codec이 도출한
87-prefix/underscore/32 lowercase-hex/총120의 정확한 형식만 Go와 C++ 양쪽에서 허용하고,
negative latches·독립 codec vectors·controlled actual로 검증한다. Generic proc/Node 예외는 없다.

## 4. B4c — 전체 static gate와 실제 관찰 aggregate

`inspection/npm/pnpm_project.go`에 현재 PNPM 전용 adapter를 둔다. Frozen context와 snapshot
전체 일치·exact acquired coverage부터 확인하고 기존 Inspector.InspectArchive를 모든 unique
archive에 적용한다. 먼저 static-all을 마치고 finding/실패/자원 초과가 있으면 prepare/dynamic을
시작하지 않는다. 모든 항목의 required dynamic NOT_EXECUTED/INCOMPLETE를 명시하고 기존
static Finding/Evidence를 보존한다. Domain/Application에 PNPM 파싱이나 임의 primary를 넣지 않는다.

Inspector가 소유하는 좁은 prepare/instance 실행 interface는 현재 concrete Sandbox consumers와
회귀 fake에 즉시 연결한다. inspection/npm이 Sandbox package를 import해 기존 Sandbox 내부
시험의 역방향 import cycle을 만들지 않는다. Package manager별 분기를 Core에 산발적으로 넣지 않는다.

`pnpm_dynamic.go`는 original dynamicFindingCodes와 Domain Evidence/InspectionReport를 재사용한다.
Expected instance coverage와 실제 Session ID 중복/foreign/missing/partial을 검사한다. 실제 결과의
summary·완료 상태·instance와 runtime/layout digest를 canonical bounded Evidence에 결합한다.
전체를 대표하는 가짜 SandboxResult/Session ID는 만들지 않는다. Incomplete result의 관찰도
Evidence/Finding으로 보존하고 하나라도 불완전하면 required check는 완료가 아니다. 기존 single
normalizeDynamicResult는 그대로 유지하며 기존 단일 npm cache를 PNPM peer 실행에 재사용하지 않는다.

## 5. 두 번째 검토와 검증 순서

Native 수명 block은 선택 의미와 분리 가능한 기존 공용 책임이다. 같은 로직을 새200줄 owner로
복제하면 실패 복원/관찰을 두 곳에서 유지해야 하므로 기존 block 이동을 택했다. Scope 종료 전
callback 값 노출을 막는 outer 반환 gate를 selection/prepare 양쪽에 적용한다. Native success나
RESOURCE 결과를 ALLOW로 쓰지 않고 source archive/byte/instance/실제 관찰·Policy를 분리한다.
Root Host controls 보존과 Artifact payload의 최소화, UNKNOWN query 시 값 폐기, 관찰된 사실과
최초 오류 보존, exact Subject/Session binding은 별도 negative로 확인한다.

Unit은 기존 selection order/fault 회귀와 새 preflight 무작업, changed intake·introduction drift,
native command/detach/controls/layout/cleanup fault를 확인한다. 실제 producer fixture는 기존
public data/source binding helper를 공유하되 prepare를 한 번만 실행한다. 앞 positive가 실패하면
미도달 negative를 성공으로 해석하지 않는다. Fresh per-instance transitive/peer/script failure,
root scripts0·source bytes 보존·모든 expected coverage/partial findings를 검증한다. 필요한 C++
build/latches는 locator 변경 단위에 한 번 묶고 완료 bulk는 refs/query errors0 후 삭제한다.

각 독립 source 단위에 canonical format/security/Quick/docs와 영향받은 package Race를 적용한다.
문서 전용/작은 helper마다 full remote CI를 반복하지 않는다. B4 production 실행 소비자가 닫히면
묶음 push하고 직전 CI40453m16을 근거로55분 실제 timer를 둔다. 그 사이 독립 후속을 계속한다.
이 문서 작성만으로 B4 구현/qualification이나 PNPM Host/CLI·전체 P2 완료를 표시하지 않는다.
