# P6-G2 — Bun 현재 whole 검사·실행 baseline

G2 source 작성 전 audit다. G1의 actual public exact10.07s/bare Semver7.8.5 7.98s,
CPP132/47.404s와 grouped Quick/Race/tagged Staticcheck는 named local PASS다.
Docs0.355s/security15.619s까지 PASS이며 G2 source는 작성하지 않았다.
CI439/source0eeefca는 별도 frozen 후보이며 실행 중 push하지 않는다.

Read: PROJECT/Queue/docs router, client09 §1–3/현재 Node·PNPM lane, plan20 §4–6,
G1 baseline/receipt, lessons INDEX·artifact L02/L19·sandbox L18·evidence L16·ci L06.
Bun project snapshot 성공을 target 없는 실행 context나 readonly program 성공으로 쓰지 않는다.

| 책임 / 실제 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Native one-off controls와 독립 metadata/SRI — BunResolver.SelectToolRun, ParseBunRunCache, Resolver.BindBunLock | YES | public method YES / current app NO | G1 named actual/Go YES | NO | 공개 선택을 현재 execution graph/context에 결합; raw cache는 unapproved DATA |
| 모든 native placement·peer edge/원본 — bun_selection.go, bun_project_snapshot.go, bun_inspection_context.go | YES | project YES | 기존 P3 named YES | 기존 project named YES | 실행 전용 resolution·controls/runtime fingerprint·전체 acquired coverage. 가짜 InstallContext/ProjectSnapshot 금지 |
| 전체 static-all/예산/준비/instance 관측 — inspection/npm/bun_project.go, sandbox/bun_inspection_prepare.go, bun_artifact_runtime.go | YES | project YES | 기존 P3 named YES | 기존 project named YES | 기존 body를 실행 입력에서도 소비하고 unique archive 보고서를 모든 instance node에 연결; 단일 primary 검사로 축약 금지 |
| 완전히 검사한 layout 인계 — BunProjectInspector.PreparedLayoutFor, BunInspectionLayout.MatchesInput | YES | project YES | 기존 P3 named YES | 기존 project named YES | execution resolution과 현재 acquired set에 같은 opaque layout을 결합; 설치 ownership 소비자는 project-only 유지 |
| 현재 Evidence/Policy/records와 readonly Node — application execution services, bootstrap/pnpm_run_execution.go, artifact/npm/execution_layout.go/execution_input_archive.go | YES | npm/PNPM YES | 기존 named YES | 기존 named YES | Bun 전용 원본/entry/layout proof와 현재 호출 연결; 현재 승인 전에 cache payload/프로그램 실행 금지 |
| 고정 Bun Artifact lifecycle — sandbox/bun_artifact_runtime.go/bun_target_lifecycle.go | YES | project YES | 기존 named YES | 기존 named YES | 일반 readonly one-off program과는 별개. 명시 Bun runtime 실행은 별도로 admission/완료 검증 |
| Bun 현재 실행 adapter / readonly Bun one-off program / CLI | NO | NO | NO | NO | 아래 순서의 현재 acceptance gap만 구현 |

## 최소 연결 순서

1. 기존 BunInspectionContext에 project와 execution의 상호 배타적인 결합을 두고
   native Key/물리 placement·모든 edge와 독립 original identity/SRI를 보존한다.
   기존 PNPM의 실행 결합과 현재 acquired-run 검증 owner를 재사용한다.
2. 기존 BunProjectInspector의 full body를 실제 두 소비자가 쓰는 private 입력 결합으로
   공유한다. 기존 static-all/원본 재측정/예산/native 준비/매 instance 관측·불완전 거부를
   다시 구현하지 않는다. Layout handoff 역시 현재 complete set만 인정한다.
3. Bun이 보증한 native layout/primary bin을 현재 Node 입력으로 결합하고 기존
   ExecutionInspect/Preparation/Evidence/Policy/records/Node runtime을 사용한다.
   Generic npm flat-only 입력과 PNPM native proof를 넓히지 않는다. 공용 복사·render는
   실제 Bun 소비자 연결이 필요한 부분만 공유하며 Bun identity를 PNPM으로 표기하지 않는다.
4. 먼저 싼 whole-graph/변조·거부 회귀와 실제 현재 normal/typed nonzero를 모은다.
   명시 Bun readonly runtime·CLI·hostile/equivalence 및 최종 검증은 각각 실제 gap을
   확인한 뒤 이어간다. 작은 helper/문서별 CI나 통과한 불변 native/CPP 반복은 없다.

G1 bare TypeScript7.0.2는 os/cpu/platform optional row를 포함한 unapproved DATA다.
기존 strict Bun parser의 지원 밖을 일반 지원으로 승격하지 않으며 Semver bare/current
named 검증과 구분한다. 설치 ownership/Host target·Core/Policy·보안 한도·runtime pin·
최종 profile/support/signing acceptance는 이 baseline으로 바뀌지 않는다.
