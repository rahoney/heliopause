# P2c5-B4c — static-all 수집 gate와 실제 instance 관찰 집계

같은 M13-004/P2다. B4 §4/5, B4b2 local receipt, Architecture02/03·Domain08/12/13,
artifact L15/L20·sandbox L08/L12/L18·evidence L11/L16을 대조했다. 기존 단일 npm과
Core 계약을 보존하고 현재 owner의 실제 acceptance gap만 연결한다.

## 1. Source 작성 전 baseline audit

| 책임 / symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 완전 archive 측정 — Inspector.InspectArchive | YES | YES | YES(named local) | NO(P2) | 전체 acquired set static-all 소비 |
| workspace complete gate — PNPMInspectionInput.CheckWorkspace | YES | YES(prepare) | YES(named local) | NO(P2) | 수집 도중 같은 charge로 조기 중단 |
| exact frozen context/acquired/instances — BindAcquired·InstancesFor | YES | YES(direct actual) | YES(named local) | NO(P2) | project adapter 소비 |
| production prepare/fresh instance — PrepareInspection·ExecutePNPMInstance | YES | YES(direct actual) | YES(named local) | NO(P2) | static-all 뒤 실제 aggregate 연결 |
| 단일 npm 해석 — normalizeDynamicResult·dynamicFindingCodes | YES | YES(single) | YES(existing) | NO(P2) | instance coverage·partial facts 해석만 별도 추가 |
| 공통 acquire/verify/Evidence/Policy — inspectDependencyBatch·ProjectInspectService | YES | YES(other clients) | YES(existing) | NO(P2) | provider error와 함께 반환한 유효 부분 Evidence 기록 |
| PNPM project inspector와 actual 종단 소비 | NO | NO | NO | NO | 이번 범위 |

B4b2를 재구현하지 않는다. Queue의 하나뿐인 IN_PROGRESS는 M13-004다.

## 2. 정확한 edit map과 순서

먼저 기존 `artifact/npm/pnpm_inspection_budget.go`의 charge owner를 확장한다.
`PNPMInspectionInput.BeginWorkspace()`가 모든 compressed bytes를 static 전에 계산하고
`PNPMInspectionBudget.Add(inventory)`가 exact acquired subject의 unique expanded/entry와
모든 peer instance logical expanded/entry를 즉시 계산한다. 기존128MiB/100000 한도는
그대로이며 duplicate/foreign/초과는 sticky 실패다. `Complete()`는 exact 전체 coverage를
요구한다. 기존 CheckWorkspace는 exact index 검증 뒤 같은 counter를 사용한다. 별도 식을
복제하지 않고 보존 tests와 incremental 중단/누락/실패 후 재사용 거부를 검사한다.

다음 `inspection/npm/pnpm_project.go`는 frozen context와 concrete static Inspector,
consumer-owned narrow prepare/instance interfaces를 받는다. 기존 Inspect 단일 API로 PNPM
context를 잃지 않도록 단일 호출은 명시 거부하고 InspectProject만 complete snapshot을 받는다.
BindAcquired → compressed gate → 모든 static InspectArchive와 Add → Complete → prepare →
각 subject의 모든 expected instance 순서다. Static finding/오류/예산 초과 시 native/dynamic은
0회다. Dependency-free는 정확한 empty snapshot만 빈 map으로 끝내며 primary를 제조하지 않는다.

`pnpm_dynamic.go`는 기존 dynamicFindingCodes를 재사용한다. 실제 결과별 Session ID와
instance key digest·runtime/layout digest 결합 Evidence를 남기고 전체 required coverage check를
별도로 둔다. 전체 대표 SandboxResult/Session을 만들지 않는다. Original ObservationSummary는
별도 bounded Evidence로 유지해 binding metadata 추가로1024byte ceiling을 넘기지 않는다.

Domain의 non-completed check는 final Findings를 가질 수 없으므로 invariant를 바꾸지 않는다.
실제 accepted 관찰의 해석은 독립 completed fact check로 Evidence/Finding을 보존하고,
required lifecycle coverage는 한 instance라도 실패/누락/중복 Session/잘못된 결과/자원 관찰이면
INCOMPLETE다. 이는 script/session completion 주장이 아니다. Composite constructor가 이미
composite인 입력의 executions를 재귀 flatten하지 않으므로 독립 leaf reports를 한 번만
조합한다. 기존 single normalization과 Core 생성자를 수정하지 않는다.

Provider 오류와 reports가 함께 반환될 때 기존 shared batch는 곧바로 reports를 버린다.
`dependency_batch_inspect.go`의 해당 오류 분기에서 exact 전체 node coverage·실제 subject의
Evidence binding을 검증한 reports만 기존 Evidence.Record로 기록한 뒤 모든 pending run을
실패로 끝낸다. Policy/VerifiedSet은 만들지 않는다. 최초 provider cause와 기록 실패를 join하고
원래 context를 사용한다. 오류 map이 nil/불완전/foreign이면 신뢰하지 않는다. 기존 성공 경로와
다른 provider의 nil,error 의미는 유지한다. 취소 때문에 저장도 실패하면 그 사실까지 보존한다.

## 3. Source 작성 전 두 번째 검토와 검증

Raw observations의 해석 완료와 required dynamic 완료를 구분해야 partial suspicious facts를
보존하면서 coverage를 낮추지 않는다. 각 report는 자기 acquired subject만 참조한다. Session
중복은 전체 집계에서 거부하고 실패 뒤 새 실행으로 성공을 제조하지 않는다. 이미 관측한 facts와
앞 static 결과, 최초 cause를 돌려주고 아직 실행하지 않은 항목은 NOT_EXECUTED다. 실패 후에도
진행하지 않는다. Empty/finding/operational error를 ALLOW로 바꾸지 않는다.

Budget 단위 뒤 canonical format/Quick/security/docs와 관련 Race로 기존 gate를 확인한다.
Project adapter는 같은 static producer·실제 prepare/backend를 사용하는 realcorpus consumer에
즉시 연결한다. Static-all 순서·finding/error/예산 전 runtime0, peer별 exact coverage/fresh Session,
partial suspicious facts·최초 오류·Evidence persistence/Policy0을 unit과 actual로 구분한다.
B4b2에서 완료한 CPP build112는 source가 바뀌지 않으면 반복하지 않는다. B4 전체 local 후
원격은 기존 묶음/55분 timer 규칙을 따른다. Host/CLI·전체 P2 acceptance는 계속 MISSING이다.


## 4. 단계별 결과와 source 재검토

Incremental budget의 format0.545s/Quick99.643s/security11.355s/docs0.238s와 기존
npm/sandbox 전체 Race21.959s는 PASS다. 기존 complete prepare consumer와 peer 비용 의미를
보존했고 compressed 사전 거부·추가 즉시 초과·duplicate/foreign sticky 실패를 검사했다.

첫 project Quick72.963s는 새 source-error fixture만 FAIL했다. 기존 openInventorySource는
Lstat 오류/일반파일 부재를 normalized 구조 오류로 반환하므로 os.ErrNotExist 보존을 기대한
새 assertion이 기존 계약과 달랐다. Static producer를 바꾸지 않고 nonnil 오류·runtime0·
완전한 미실행 reports를 확인한다. 첫 원본 p2c5b4c-project-first-quick은 FAIL로 유지한다.

같은 새 해석 owner에서 full ObservationSummary가 subject/byte 한도를 넘으면 알려진 의심
사실까지 폐기하지 않도록 검토했다. 기존 네 dynamicFindingCodes만 saturating counts로
별도 bounded fallback에 남기고 actual Session과 summary_incomplete=true를 결합한다.
원래 summary 오류를 함께 반환해 required coverage는 INCOMPLETE다. Fake aggregate Session이나
한도 확장은 없다. Constructor의 incomplete/final-Finding invariant도 유지한다. 모든 결과를
단일 leaf 목록으로 조합하며 facts report가 오류와 함께 있어도 버리지 않는다.

## 5. 묶음 CI 소비자 연결의 source 작성 전 감사·지도·재검토

현재 `ci-pnpm-runtime.sh`는 고정 공급·기존 startup4만 IMPLEMENTED/WIRED/QUALIFIED=YES,
새 project/controlled instance 실행은 NO다. Workflow의 existing sandbox test-client/helper
교체·종료/복원은 YES지만 binary의 realcorpus tag와 native 공급 시점은 MISSING이다.
현재 script가 호출될 때 installed client는 제품 CLI여서 새 privileged consumer를 단순히
`go test`로 더하면 typed-helper executable identity와 맞지 않는다.

기존 workflow의 `go test -c -o ...helox-sandbox-test`에 realcorpus tag만 연결하고, 원래
`haa_install_ci_client`→`haa_start_policy_helper` 뒤에 기존 PNPM supply script 호출을 옮긴다.
Script의 startup4는 보존하고 **installed ordinary-user test client**로 public project 전체와
controlled success/exit37/long-name만 두 명령으로 실행한다. Public6을 두 consumer에서 중복
실행하지 않는다. 새 helper owner·추가 권한·인증 우회·수명 script 복제는 없다. 기존 parent의
same EXIT/restore/Required 경로를 유지하며 native supply는 script의 기존 own cleanup이다.

`check/pnpm_runtime_workflow.go`는 기존 exact active command validator에 두 실제 명령과
동일 active lifecycle block의 tagged compile→install→helper start→supply 호출 순서만 추가한다.
Fixture 누락/주석/flags0/중복·공급을 client 준비 앞으로 이동하는 negative를 확인한다. 임의 Bash
의 실행 증명으로 문자열 순서를 확대하지 않고 canonical Bash/actionlint/workflow와 실제
소비자 결과를 구분한다. Quality와 product package 검증은 새 source 경계에서 한 번 묶고,
CPP source/lock은 그대로라 앞112-latch release build를 재사용한다.

최종 source 대조에서 layout.Target의 exact subject/key만으로는 같은 archive/instance를 갖는
다른 frozen controls의 layout 전체를 구분하는 getter가 없음을 확인했다. Concrete prepare는
입력을 그대로 사용하지만 adapter boundary도 이 결합을 명시한다. 기존 layout owner에
`MatchesInput`만 추가해 full snapshot/runtime·ordered acquired set을 비교하고 project adapter가
그 결과를 실행 전에 요구한다. 기존 BindLayout 알고리즘이나 허용 범위는 대체하지 않는다.
원래 acquired runs/원본 controls/runtime 변화 거부를 direct test로 확인한 뒤 actual consumer의
이 새 gate만 동일 public 입력으로 재검증한다.

## 6. 기존 C4 public Policy 소비자의 마지막 연결

B1 §3의 재검증 목록과 대조하니 `TestPNPMPublicNativeResolverIntegration`의 네 실제 검역
case는 아직 기존 single npm composite를 사용한다. Direct project actual만으로 이를 닫지
않는다. 이 기존 test의 selected.BindProjectSnapshot을 같은 owner의 BindInspectionContext로
바꾸고 그 frozen snapshot별 PNPMProjectInspector→기존 ProjectInspectService/Evidence Store/
M3·M4 Policy를 즉시 연결한다. Native8 selection과 single/transitive/peer/selected-update4개
검역·Host 미설치 assertion은 보존한다. Static·required coverage3개와 instance별 관찰 해석을
구분한 정확한 check/Evidence 수를 검증하며 optional fact check를 required 완료로 바꾸지 않는다.

기존 consumer를 재작성하지 않고 준비 composition과 결과 shape만 새 계약에 연결한다.
같은 installed authenticated client로 CI supply script에도 이 한 실제 test를 추가하고 command
누락 negative를 기존 validator에서 확인한다. Source selection과 frozen fixture의 scope는 다르며
같은 public fixture 반복을 whole qualification으로 합치지 않는다. 이 실제 네 Policy/Evidence
case와 final frozen-layout gate 소비를 확인한 뒤 B4 local checkpoint를 닫는다.


## 7. 최종 named local 결과와 다음 경계

Final production source의 format1.227s/Quick100.011s/security11.742s/docs0.227s,
관련 네 package 전체 Race32.714s와 workflow0.695s가 PASS다. 이 뒤 realcorpus C4 consumer/
CI 연결 변경의 format1.212s/Quick51.085s/workflow0.702s와 실제 test compile5.682s도 PASS다.
Production source는 그 Race 뒤 바뀌지 않았다. Bash syntax도 PASS다. 최종 staged 문서·영수증
전체에는 별도 security/docs와 전체 index/blob/disk/mode/tree gate를 적용한다.

최종 input/layout gate의 frozen public actual49.087s/test46.71s는 transitive2/peer4의
complete static-all→prepare→실제 독립 Session6과 정확한 instance/subject/runtime/layout
Evidence를 통과했다. 앞 gate 전48.738s와 구분하며 최종 결과로 대체 기록하지 않는다.

기존 native public actual114.917s/test112.51s는 native8 selection과 single/transitive/peer/
selected-update4개 검역의 총8 acquired entries를 기존 ProjectInspectService·실제 Evidence
Store·M3/M4 Policy ALLOW에 연결했다. 각 entry는 verification/static/aggregate required3개와
모든 instance의 completed 사실 해석을 구분하며 exact Evidence6개/instance1인 public 입력이다.
모든 actual에서 helperwait0/CONFIRMED_STOPPED·runtime directory 정리·보호 Host 구성 동일을
확인했다. 어떤 case도 Host 설치를 수행하지 않았다.

[로컬 영수증](./p2c5b4c-pnpm-project-inspection-local-20261025.json)과 final-project/public-policy
원본이 source와 범위를 소유한다. IMPLEMENTED/WIRED/QUALIFIED=YES(named local),
ACCEPTANCE_CLOSED=NO(whole P2)다. B4 묶음 remote qualification 뒤에도 retained cache/
provenance/Host records·transaction·CLI와 whole P2 acceptance는 MISSING이다. 다음은 이 실제
검사 결과를 사용하는 Host 반출 owner의 baseline audit·정확한 수정 지도이며 기존 npm
package-lock transaction을 PNPM layout으로 대체하지 않는다.
