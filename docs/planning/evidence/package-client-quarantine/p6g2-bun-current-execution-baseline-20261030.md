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

## G2 첫 현재 앱 실패 — 아직 미해결

G2 context/whole inspector/layout/current Node 연결의 표적4package15.858s·tagged vet3.255s/build2.745s는 PASS다. 실제 첫 현재 앱18.90s(operator22.239s)는 독립 source/cold 선택 뒤 기존 offline inspection 준비에서 `OPEN_RESULT_CORRELATION`/events17로 중단됐다. Policy/프로그램은 미도달이며 current 앱 성공으로 기록하지 않는다. Helperwait0/Host5·Docker 보존/같은 inode profile 복원을 확인했다. Raw `p6-g2-current-first-v1`을 보존하고 kernel syscall/path/flags/pending 진단부터 확인한다. G1·CI439의 이미 검증된 입력을 재실행하거나 일반 correlation 예외를 늘리지 않는다.

V2의 동일 Go ELF/입력과 판정 불변 private observer 진단17.42s(operator19.575s)는 `sysno437/flags2654208/success=true/pending0`, kernel 경로 `/tmp/haa-bun-input/project/node_modules/typescript/bin`·workspace mount36을 확인했다. 기존 G1은 bun x의 한 번 실행되는 ARTIFACT installer/private bunx-cache에만 한정되어 있어 이 CONTROL offline install의 project 경로는 대상 밖이다. 새 성공을 주장하지 않고 exact14argv offline install·current group·project workspace의 성공 O_PATH만 별도로 계상하는 최소 후보와 negative 회귀를 검토한다. Correlation 일반화나 synthetic pending은 만들지 않는다. Raw `p6-g2-open-diagnostic-v2`와 그 observer/Go hash를 따른다.

V3의 최소 수정은 exact14argv offline install에만 current-image marker를 두고 exec에서 지우며 clone에 상속하지 않는다. 같은 CONTROL direct-root/current generation/expected Bun·kernel 성공 O_PATH/정확한 project node_modules·workspace anchor만 계상한다. 표적 semantic RED53.187s→GREEN46.116s 뒤 동일 Go ELF의 앱2가60.97s(operator63.211s) PASS했다. 각 요청은 독립 cold source/원본/whole 검사·현재 Evidence/Policy ALLOW·새 Manifest/SBOM과 readonly Node normal0/typed nonzero1을 확인했다. Host5/Docker/helperwait0/같은 inode profile 복원을 확인했다. Raw `p6-g2-current-fixed-v3`가 원본이며 CPP/Go grouped qualification은 아직 PENDING, 명시 Bun program·CLI/hosted/general/final은 미완료다.

## 같은 로컬 묶음의 현재 CLI 연결 audit — CLI source 전

Bun current workflow는 G2 actual 앱2에서 사용됐지만 default CLI caller는 아직 없어 default source 연결도 acceptance gap이다. 기존 packageClientDescriptor·literal argv/출력·ExecutionRunner·bootstrap runNPMExecution·PNPM native request reader·공용 current CLI/거부 fixture는 IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES다. Bun 전용 descriptor/두 alias(`bunx`, `bun x`)·current factory 연결만 IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다. MISSING은 기존 native package/명시 command reader·기존 presentation/런타임 owner를 사용한 두 alias와 거부/영어·한국어·출력 경계 검증이다. Generic npm의 strict 기본 bin 선택을 그대로 쓰며 다중/이름 불일치는 `--package <pkg> <command>`로 명시한다. 명시 `--bun` runtime은 별도 미구현으로 거부하며 이 Node 한정 CLI로 지원 완료를 주장하지 않는다. 새 job/helper/rebind 없이 기존 Bun supply cohort에 모은다.

## Named local qualification 종료

현재 Node 앱2/60.97s와 CLI3/190.91s·entry 거부2/50.00s, Rimraf12nodes/13edges·Host/helper/원복을 확인했다. 같은 observer57c186ea의 전체CPP132/50.004s·canonical Quick351.507s/영향5package Race92.964s/checker Race72.786s·고정 tagged Staticcheck2.874s/docs0.342s/security18.055s는 PASS다. 최종 범위와 source hash는 [G2 결과](./p6g2-bun-current-cli-local-20261030.json)가 소유한다. 새-source hosted/명시 Bun program·일반/final은 OPEN이며 [G3 baseline](./p6g3-bun-readonly-runtime-baseline-20261030.md)의 실제 gap을 이어간다. 검증 뒤의 receipt/status 갱신은 문서 bookkeeping이며 불변 제품/CPP/native를 다시 실행하지 않는다.
