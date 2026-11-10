# Step 13 — Current Work Queue

이 문서는 Heliopause에서 **지금 실행할 작업, 순서, 상태와 완료 증거**를 관리한다. Milestone의 범위와 완료 조건은 [Step 12 — Milestones](./01-milestones.md)가 소유하며, 이 문서는 현재 milestone만 구현 가능한 크기로 분해한다.

상세 Architecture·Domain·Engineering 결정을 복제하지 않는다. 각 work item은 필요한 canonical leaf 문서를 가리키고, Step 14는 해당 문서를 읽은 뒤 그 항목의 범위만 구현한다.

## 1. 현재 상태

```text
Current milestone: M13 — Production Release & Operations
Milestone status: IN_PROGRESS — runtime maintenance main delivery COMPLETE
Active work item: M13-004 — installed self-quarantine update design and convenience channels
Last completed work item: CLI-UX-003 — COMPLETE; source4cfe29e/CI352 ten SUCCESS
Predecessor: SEC-UPD-001 — IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES
Protected-path disposition: M13-002 — ACCEPTED_WITH_EXCEPTION; QUALIFIED=NO / ACCEPTANCE_CLOSED=YES; missing-CLA NOT_TESTED / DEFERRED (C3)
Prepublication prerequisites: CLI-UX-001, CLI-UX-002, SEC-UPD-001 and CLI-UX-003 COMPLETE
Active work item status: IN_PROGRESS — prior bounded functionality/audit and local test setup retain their scoped evidence. User-goal reassessment OPEN: the installed baseline lacks pip/uv install -r; the B01 candidate is implemented/wired with scoped local actual evidence; grouped qualification remains open. D-008 environment-triggered dormant behavior induction/detection is not acceptance-closed. Names/deployment must not replace these gaps.
Last completed substep: Source compatibility/current Python and npm consumers are preserved in4857a61; Go full census/configured/default analyzer/current cache inb2c6ddb. Exact TypeScript113 source actual-v12 passed175.69s; unchanged syntax/corpus and CPP138 passed, fresh canonical1036-member source build1437.755s reproduced exact33ae4500/171651104B with Host unchanged. Python/npm scoped Race/quality/security and Go Race84.088s/foundation49.426s/security18.915s/tagged vet10.666s/pinned Static16.388s passed. These are scoped local checkpoints; whole CLI/Python/Go/hosted/S04 remain OPEN.
Next action: WSL offline compaction COMPLETE per F recovery-20261105 result/restart receipts: reclaimed131198877696B; checkpoint a936153/source3494/Node5/backups7 and persistent clean clone3488 files matched. See wsl-recovery-20261105.md for durable clone location and exact restart sequence. Read-only actual-Host pre-resume review09:46–09:48UTC confirmed rw/same UUID, all6 protected runtime bytes/metadata, pinned5 images+Semgrep SDK base, active/enabled policy and container/process0/query errors0. Observer directory is actually absent and must be restored under existing idle/ownership conditions before dynamic execution. Terminal-resume preparation COMPLETE; exact Semgrep supply/source collection/SDK inputs and updater user-v3 frozen state remain MISSING. Recovery work may resume; direct updater verify is NOT_READY. Reconstruct pinned inputs with existing owners before new development-user plan/apply/verify; interrupted verify remains NOT_COMPLETED. Bash/Go actual success6e74f22 and npm expectation correction5fd908b remain preserved. Whole S04/design/grouped/hosted acceptance remains OPEN. No CI/timer running.
Current substep: user-authorized controlled development installation CLOSED locally; Docker29.9.2/current CLI-helper-pod-init and running policy service, runtime4OK and Python install/update + Go get/update/build PASS. Formal release record remains unavailable. Approved simple-door artwork CLOSED locally. Logo only left of HELIOPAUSE, full centered second title, horizontal RGB gradient/fixed stars. Four menus/footer/NO_COLOR/lifecycle preserved. Final frozen source checks passed; selected3 PNGs and current contract retained, rejected untracked drafts archived with hashes. Artwork checkpoint had no remote CI/push/installation; the later user-authorized development Host setup is recorded separately and is not whole M13 completion.
Acceptance audit: keep IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED independently. Prior source/profile checkpoint is not functional completion. Native all-option equivalence is not promised by the original initial contract; stdin/TTY/file-export have separate acceptance. Do not mix explicit exclusions with required omissions or relabel unimplemented goals as approved deferral.
Independent local next: B00-B07 and S09 local checkpoints retain independently scoped evidence; their grouped hosted/final qualification is OPEN. C01 foundation baseline9e9260f and its scoped implementation/checks are recorded in storage-lifetime-baseline-20261102.md. C01 implementation/qualification/acceptance remain OPEN. Anyio and large-public-catalog compatibility remain OPEN. Current runtime maintenance and Host installation unchanged.
Then: M13-005 — NOT_STARTED; concrete publication remains separate
Runtime upgrade maintenance: UPG-001..006 COMPLETE; PR34 merged/main CI320 ten SUCCESS
CI gate (2026-11-10): CI452~478 27연속 성공 0·19→188분. CI478(c0af66e) gVisor Linux Integration 실패, 첫 실패 TestLinuxUVSyncCompleteCLIIntegration UPDATE_ORIGINAL_STATE_UNSUPPORTED, 원인 UNKNOWN. 새 hosted CI는 docs/lessons/ci_discipline.md §3 PLAN.md 통과 전 금지(분석 소유: docs/github-logs/ci478-20261110).
RESUME-20261110 (WSL 재시작 후 이어서): 저장소를 ~/projects/Heliopause-Artifact-Airlock(ext4)로 이전함. 기존 OneDrive 사본은 폐기 대상(최신 아님). 아래 순서로 진행하고 ★ 전에는 코드 수정·CI 금지.
  R0-1 (구 세션, WSL 종료 전) 산출물 정리 점검: ~/.local/state/heliopause/{m12-*,m13-*,runtime-upgrades,reviews}·~/.cache/heliopause의 남은 대용량을 사용자와 정해 삭제(INDEX 교훈 11). 삭제는 압축 전에 끝내야 vhdx 반환량이 크다. 2026-11-10 1차 정리 완료(ext4 139G→103G). 남은 선택 후보: m12-003 6.5G, m12-002 1.5G, runtime-upgrades 1.3G, m12-004 0.8G.
  R0-2 ★ WSL 종료 → vhdx compact(사용자 수행, 명령은 ci_discipline.md §7) → 재시작 후 df로 확인(vhdx 약 180GB, 사용 약 103GB).
  R0-3 (새 세션) 반드시 `cd ~/projects/Heliopause-Artifact-Airlock`에서 Claude를 시작한다. /mnt/c 경로에서 시작하지 않는다. 시작 확인: pwd, git status(큐 변경 1건 외 깨끗), SessionStart hook의 INDEX 로드(교훈 1~11), .claude/hooks/ci-push-gate.sh 동작.
  R0-4 (새 세션) 문서 정리 — AGENTS.md(28KB→≤20KB), 이 큐(587KB→≤200KB), lessons 4문서(각 ≤100KB). docs/lessons/INDEX.md "비대화 신호와 정리 규칙"대로 종결 항목은 docs/lessons/archive/·docs/planning/archive/로 이동하고 원 문서에는 ID·상태·한 줄 요약·링크만 남긴다. 증거 경로·반증/superseded 연결·실패 조건은 삭제하지 않는다. 이동 뒤 `go run ./scripts/check docs` 통과. 정리 전후 바이트와 항목 수를 기록. R1 시작 전 완료. **완료(2026-11-10):** 큐 590,678→약54KB(원본 줄 7259개·제목 414개를 docs/planning/archive/ 3파일로 무변경 이동, 색인은 큐 끝), AGENTS.md 28,422→11,488B(재개 지침 원문 줄10-139는 docs/planning/archive/agents-resume-history-20261110.md), lessons ci 127,202→99,693·artifact 263,831→93,490·sandbox 210,813→92,592·evidence 248,038→97,307B(날짜 있는 오래된 항목 28/170/104/122개를 docs/lessons/archive/로 이동, 날짜 없는 핵심 필드는 유지). 줄 단위 손실 0 검증, github-logs 제외 사본에서 docs 검사 통과. PROJECT-DECISIONS.md 38,327→22,321B(Current Stage·Remaining Stages 이력을 docs/planning/archive/project-decisions-stage-history-20261110.md로 이동, 요약·링크 유지). docs 크기 경고 0건. root 소유였던 docs/github-logs/system-recovery-20261105/check-state는 삭제하지 않고 소유권만 winamp로 변경해 저장소 전체 docs 검사가 열리게 했다github-logs 사본의 기존 끊어진 링크 4,514건 중 4,507건은 옛 소스 스냅샷 9개의 부분 사본 큐/문서라 고치면 기록된 소스 지문이 바뀐다. 증거는 그대로 두고 scripts/check/docs.go가 미추적 원본 폴더 docs/github-logs를 건너뛰게 했다(회귀 시험 추가, CI에는 이 폴더가 없어 동작 불변). 저장소 루트의 go run ./scripts/check docs는 경고·오류 0, foundation 통과. 참고: 같은 이유로 루트에서 quick/format은 github-logs의 미추적 .go 원본 때문에 실패하므로 추적 파일 사본에서 실행한다. 사본 quick의 유일한 실패 TestPatchDrift_I_ExactPatchedRuntimeSatisfiesCapability(internal/runtimeidentity)는 이번 변경과 무관한 Host 상태다: 시험이 로컬 설치 runsc(LocalRunscPath)를 호출하는데 c8a546d0 이전 바이너리라 sentry/filesystem_mutation이 없다(wsl-recovery-20261105.md:233, HEAD에서도 재현). 다음은 R1.
  R0-5 (2026-11-10 확인, Host runsc): Host 번들 /usr/libexec/heliopause/gvisor는 패치 588adad4(구)이고 설치된 helox(11-01 빌드)도 같은 해시를 내장한다. 현재 lock(c8a546d0)과 6개 member SHA512가 모두 일치하는 재빌드 번들이 /var/tmp/haa-resume-20261107/gvisor-bundle에 있다. Host 번들만 교체하면 helox doctor의 trusted-host-runtime이 깨질 수 있어 하지 않는다(교체하려면 번들·helox·observer·helper 재빌드+서비스 재시작의 Host 전역 변경이며 계획서와 승인이 필요하다). 로컬 TestPatchDrift는 Host 변경 없이 HELOX_RUNSC_BINARY=/var/tmp/haa-resume-20261107/gvisor-bundle/runsc 로 PASS(0.051s)한다. R1 (d)에서 로컬 gVisor 재현에 Host 교체가 실제로 필요한지 먼저 판단하고, 필요 없으면 교체하지 않는다. 이 번들은 /var/tmp 소실 시 build-gvisor-runtime-bundle.sh로 재현한다(2026-11-07 8m19s).
  R1 분석 보고서 1건(모델: 사용자 선택, 분석은 상위 모델 권장): (a) CI451(마지막 green)→HEAD 79커밋 diff 중 실패 시험·경로 대조, (b) docs/lessons grep으로 UV update/UPDATE_ORIGINAL_STATE_UNSUPPORTED·시험 시간 증가의 유사 사례, (c) analysis/compare_test_cost.py로 CI451 vs 478 시험 비용, (d) TestLinuxUVSyncCompleteCLIIntegration 로컬 재현, (e) 연쇄/오판/복잡도 3갈래 점검, (f) "처음부터 잘못 시작된 지점" 판단, (g) 외부 레퍼런스·단순한 해결 대안 2개 이상, (h) 작업 방식 문제(아래) 포함.
  R2 구조 대안 비교(시험군 matrix + fail-fast 해제, 공개망 시험 분리 등)를 사용자에게 보고 후 승인된 것만 구현.
  R3 해결 후 docs/lessons/ci.md 기존 항목 갱신(오답노트), ci_discipline.md §3 PLAN.md를 채운 뒤에만 CI 1회.
  작업 방식 문제 후보(검증됨): AGENTS.md 28KB(한도 20KB), 이 큐 587KB(한도 200KB), lessons 4문서 850KB로 읽기 비용 때문에 규칙이 맥락에서 소실됨 → 규칙은 짧게 INDEX/ci_discipline에 두고 이력은 archive로 이동(사용자 승인 후 별도 작업). push는 .claude/hooks/ci-push-gate.sh가 PLAN.md 없으면 거부, 있으면 확인 요구.
```

현재 작업 기준: 필요한 변경만 수행하고 기존 모듈·단순한 호출 흐름을 유지한다. 불필요한 함수·추상화·연결·파일 분할을 늘리지 않으며, 로컬 변경·검증을 모아 필수 시점에만 CI를 실행한다. 기존 20분 타이머를 재사용하고 새 CI나 타이머를 중복 등록하지 않는다.

2026-11-05 사용자 승인 개발 작업 정리 보완은 named local 범위로 연결·검증했다.
scripts/check-workspace.py의 영속 원장/개발 UID 실행/전용 cache·tmp/종료 수거와 재시작
미완료 차단, Host 회귀7개11.064s 및 실제 canonical docs exit0/수거를 확인했다.
기존 누락 사본2개111374336B도 전체 입력/신원·참조 대조 후 수거했다. 최종 후보 검사는
같은 wrapper로 기록하며 제품 C01/C02 전체 완료와 구분한다. 상세·범위는 복구 leaf가 소유한다.

2026-11-06 후속 순서는 보강 기능·필수 검증 완료 → 보강 설계 §11.2의 배포 자료
노출 억제 최종 검토 → §13의 저장소 운영 방식 검토 → 기존 명명·채널·배포 마무리로
기록했다. 내장/바이너리화/난독화·암호화 및 비공개 개발+공개 안내·배포 저장소는 검토
후보이며 아직 채택하지 않았다. 현재 입력 복구·updater verify의 재개 순서와 기존20개
gap·16개 작업 순서는 유지한다. 상세 결정 기준은 보강 설계가 소유한다.

2026-11-07 무인 재개: 현재 lock gVisor bundle을 기존 build script로 `/var/tmp/haa-resume-20261107`에
8m19s 재현했다(6 member 모두 lock 일치). S04 Semgrep updater verify는 **BLOCKED**다. 1036행
source/SDK 수집 owner가 저장소에 없고 공급 구조가 미결정이라 새 수집 도구 없이 복구할 수 없다.
해제 조건: 수집 경로 결정 또는 원본 supply 제공. 상세는 복구 leaf가 소유한다.
CI452(9ca552e)는 6 SUCCESS/3 FAILURE였다. compress 취약점·macOS test compile·lifecycle client 신원의 세 원인을 최소 교정했고 로컬 canonical4가 PASS다. 다음 hosted 결과는 복구 leaf가 소유한다.
CI453(ab63980)에서 세 교정은 hosted로 확인됐다. 남은 hosted 실패는 CI에 Semgrep 공급이 없어 생기며, S04 공급 결정 전까지 hosted ordinary·CUDA qualification은 **BLOCKED**다. macOS temp symlink 시험 교정은 로컬 검증 후 다음 CI에 묶는다.
2026-11-07 사용자가 Semgrep 공급 A(자체 build→같은 저장소 Release 운송)를 선택했다. 사라진 source offer를 보존 원본과 고정 upstream으로 재수집하는 단계 계획은 복구 leaf가 소유하며 Release 게시는 실행 직전 별도 승인이다.
2026-11-07 Semgrep 공급 재수집 완료: core `33ae4500` 정확 재현, SOURCES/NOTICES same-engine lock 갱신, CI Release 연결 구현. 다음은 Release 생성 승인→push/CI→CUDA 순차다. 상세는 복구 leaf.
2026-11-10 별도 과제(미구현): advisory 시험을 고정 응답 결정론 회귀와 live OSV 통합으로 분리한다. 현재 ALLOW 공개 fixture는 live OSV에 묶여 있다. 근거는 복구 leaf의 CI477 항목이 소유한다.

### 2026-11-03 S05 advisory batch/page — 로컬 부분 구현

[Baseline·수정 지도·부분 구현](./evidence/m13-004-convenience/advisory-baseline-20261103.md)은 기존 graph/report/판정 owner를 재사용한다. 중복 key 조회 합침·원래 순서 환원·유한 pagination/응답 예산·모호/부분 JSON·HTTP 실패·취소 거부를 구현하고 Race4.501s·canonical foundation55.335s/security14.955s 및 static/vet를 확인했다. 현재는 controlled HTTP 모델이며 record/영향 해석·production transport·graph/결과/판정·cache/CLI 연결은 MISSING이다. 전체 S05 네 상태는 NO이고 실제 live 응답·native·CI 성공으로 확대하지 않는다. M13-004 하나만 IN_PROGRESS다.

### 2026-11-03 S04 도입 baseline — 제품 미채택 / 공급 조건 미완료

[도입 검토](./evidence/m13-004-convenience/content-analysis-baseline-20261103.md)는 기존 구조 검사/inventory와 실제 내용 분석의 gap을 구분한다. 고정1.182.0 wheel 해시·private publisher의 offline 서명 검증2.401s, network-none/runc/read-only DATA의6언어12source/6marker·errors0·skip0/1.517s를 확인했다. Public tag와 private build의 대응·해당 버전 notices가 미확정이므로 prebuilt adoption은 BLOCKED이며 전체 S04 네 상태는 NO다. 같은 CE 공개 source-build 공급을 비교 검토하고, 선행08–09를 공유하는 S05의 독립 baseline도 진행할 수 있다. 제품 코드/lock/Host 설치·CI는 변경하지 않았다. 전체 보강 및 이름 독립 준비는 미완료다.

### 2026-11-02 S01/S08 baseline — IN_PROGRESS

[현재 owner·재사용·수명 대조](./evidence/m13-004-convenience/environment-state-baseline-20261102.md)는 최소 격리/관찰과 실제 모의 환경·파일 상태 비교를 구분한다. 반증된 guest noexec 후보를 기존 pinned runtime의 좁은 추가 제한으로 교정했고, 기존 npm archive 실행의 두 fresh 환경·공유 자원·종료 후 readonly inventory를 실제 경로에 연결했다. Scoped native와 실패/정리 회귀 원본은 같은 baseline/부분 원장이 소유한다. Typed 상태 digest/범위의 Evidence는 로컬 연결됐으며 단기 VFS 관찰과 활성 설정 누락 거부는 kernel7/CPP134/actual6로 한정 검증했고, npm의 persisted Evidence/ALLOW readback도 한정 완료했고 ordinary Python의 실제 저장·no-import·선정된 네 root profile 경계도 한정 확인했으며 npm/PNPM/Bun current instance/root 및 대표 App/store 소비자도 한정 확인했고 다음은 Go/Cargo build와 나머지 probe다. S01/S08 전체 네 상태는 모두 NO다. 기존 한도·권한·버전 업데이트 체계는 유지하며 C01 전체와 B01 호환 gap은 계속 OPEN이다.

### 2026-11-02 C01 foundation — IN_PROGRESS

[Baseline와 현재 부분 구현](./evidence/m13-004-convenience/storage-lifetime-baseline-20261102.md)은 기존 owner를 대조한 뒤 확인된 실패 근거/취소 유실과 공통 Evidence 예약 원장을 보강한다. 원래 기록·게시·Close 경계를 재사용하며 quota가 찼다고 미만료 자료나 알 수 없는 자료를 삭제하지 않는다. C01 전체 자동 정리/유한 저장소 완료와 final qualification은 계속 OPEN이며 새 CI/Host 변경은 없다.

### 2026-11-01 누적 보강 설계 검토 — 문서 교정 / 구현 NOT_STARTED

기준9aa7430의 [보강 설계](./21-goal-completion-design.md) 전체 순서·보안/정리/결과·배포 경계와 향후 planning22를 대조했다. A01에 기존 execution report/guest exit 구분과 유효한 선택 필드 부재를 명시하고, A02에 운영자 설정의 허용 경로·실행 중 취소·응답 유실 상태를 구체화했다. 새 AI 항목을 최종 보안 검토에 연결하고 command/mode coverage와 공통 회귀 증거 재사용을 구분해 전 조합 반복 검증을 요구하지 않는다. 기존 버전 유지보수·필수 동적 검사·자동 정리·16개 순서/묶음 CI는 유지한다. 이는 설계 교정이며 제품 구현·AI client 연결·Host·CI 변경 없음.

### 2026-11-01 코딩 AI 사용 흐름·Skill·최소 MCP — 설계 반영 / 구현 NOT_STARTED

[보강 설계 §4.8–4.9](./21-goal-completion-design.md)는 Python 외 JavaScript/Go/Cargo/Terraform/GitHub의 실제 명령별 결과·설치/업데이트·거부 후 대안 재검역을 A01 필수 acceptance로 둔다. 배포 Skill에는 raw manager·직접 다운로드·설정 완화로 우회하지 않는 규칙과 실패 분류를 담는다. 기존 JSON/project report와 Application owner를 재사용하며 일부 텍스트 경로만 보완한다. A02 최소 MCP는 기존 bootstrap/use case/lifecycle을 재사용하는 로컬 stdio 범위로 닫을 수 있으면 함께 구현·검증하고, 불충족 시 exact 근거와 미지원 상태를 남긴다. 기존20개 B/S/C gap·16개 순서에 이 AI 사용 acceptance를 연결했으며 새 milestone은 없다. Skill/MCP는 Host 권한 강제 통제가 아니고 planning22의 AI 코드 생성과 구분한다. 이번은 문서만이며 구현·AI 설정·Host·CI는 변경하지 않는다.

### 2026-11-01 AI 보조 실행 검사 — 향후 후보 기록 / DEFERRED

사용자 요청에 따라 [MCP 사용자 AI 협업과 자체 생성 후보](./22-future-ai-assisted-inspection.md)를 기록했다. 코드 생성/디버깅·전용 위치 제출·격리 실행·최대2회 시도와 실패 시 선택 검사 생략을 구분하며, 내장 모델/API/AI CLI는 후속 v2.0 핵심 기능 후보로 둔다. MCP 제공도 선행 확인 사항이다. 현재20개 gap·16개 순서·첫 배포 acceptance는 그대로이며 실제 구현·모델/API 연결·CI를 시작하지 않는다.

### 2026-11-01 보안 성능·실행 범위 재검토 — 설계 보완 / 구현 NOT_STARTED

[보강 설계 §3.1·§5.6·§11.1](./21-goal-completion-design.md)에 기존 버전 유지보수 보존, 설치와 제한 실행의 구분, 기본 동적 검사 유지와 실측 후 심층 profile 판단, 공격자가 source를 아는 전제의 최종 보안 검토를 연결했다. 마지막 검토는 기존16번에 포함하며 탐지·명시 차단·자동 반입 방지·정상 오탐·미완료를 분리한다. 측정 결과나 실제 대다수 악성코드 차단 성능은 아직 없으며, 모든 검사/새 옵션의 구현을 완료로 표시하지 않는다. 이번은 문서만이며 제품·Host·CI 변경 없음.

### 2026-11-01 최종 설계 대조 — 구현 준비 / 구현 NOT_STARTED

[보강 설계 §5·§6·§13](./21-goal-completion-design.md)을 원래 MVP-007과 기존 M13 배포 계약에 대조했다. 새 검사 도구·규칙의 실제 배포물/초기 설치/doctor·자체 업데이트와 schema 호환 연결, 서명 미제공과 검증 실패의 기본 판정, 명명/공개 배포 전환 조건을 보완했다. 기존 20개 gap·16개 순서는 유지한다. 해당 완료 조건을 실제로 닫으면 이름 결정·배포 마무리 단계로 진입할 수 있으며, npm/PyPI/Homebrew 채널 구현·publisher binding과 최종 서명 후보/clean Host·게시 검증은 기존 M13-004/005에서 마쳐야 한다. 이번은 설계 대조와 문서만이며 구현·Host·외부 publication·CI는 시작하지 않는다.

### 2026-11-01 보강 설계 — 문서 작성 / 구현 NOT_STARTED

[보강 설계](./21-goal-completion-design.md)에 B00–B07/S01–S09의 기존 owner·수정 경로·정상/거부/실패 완료 조건과 C01 자동 정리·유한 quota/retention, C02 시험 payload 정리, C03 불용·중복 코드 정리를 연결했다. 16개 순차 단위, 재사용·성능 기준, 묶음 CI와 실제 20분 조회/결과 소비를 명시한다. 현재 DeleteRun은 명시 삭제이며 production caller가 없다는 source 사실과 기존 session 정리를 구분했다. 사용자가 **실제 작업은 아직 하지 말라**고 요청했으므로 이번은 설계 문서만 작성하며 제품·Host·시험 payload·CI는 변경하지 않는다. 보강 기능의 네 상태는 모두 NO/NOT_STARTED다. 후속 구현 지시가 있으면 01/B00부터 진행한다.

### 2026-11-01 일괄 설치·원래 보안 요구 구현 대조 — 감사 완료 / 기능 OPEN

[구현 대조와 미지원 목록](./evidence/m13-004-convenience/goal-coverage-audit-20261101.md)은 기존 여섯 생태계의 파일 기반 의존성 흐름과 D-008/MVP-005~008을 실제 caller·검사 body·기존 qualification에 연결한다. B00~B07/S01~S09에 독립 baseline과 MISSING을 기록했다. 기본 npm install/ci·Cargo fetch·Terraform init도 미지원이며 npm ci/Cargo fetch/Go mod tidy의 도움말 exit0을 실제 재현했다. 구조 검사 외 악성 패턴·사용자 package 취약점 DB·위험 변화·일반 공급자 증명과 동적 유도/범위 보고가 남는다. 기존 보호·지원 흐름은 재사용한다. 이번은 감사·문서와 싼 preflight만 수행했으며 제품·Host·CI를 변경하지 않았다. 다음은 B00와 requirements 입력 묶음이며 이름·배포는 뒤다.

### 2026-11-01 requirements 입력·잠복 공급망 탐지 목표 — OPEN

[사용자 목표 재대조](./evidence/m13-004-convenience/local-dev-test-20261101.md#2026-11-01-사용자-핵심-목표-재대조--open)는 실제 두 install -r 거부와 표준 Python 검사 경로의 D-008 유도 환경·조건부 행위 검증 gap을 확인했다. 기존 격리·관찰·검사와 bounded CI 성공을 이 목표의 완료로 확대하지 않는다. IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED와 MISSING을 해당 leaf에 분리했다. 기존 테스트 환경과 과거 결과는 보존하며 이름·배포보다 미완료 목표를 우선한다.

### 2026-11-01 동일 WSL 사용자 체험 환경 — COMPLETE

[개발 검증 환경 설정](./evidence/m13-004-convenience/local-dev-test-20261101.md)을 완료했다. 기존 설치와 이전 Docker 패키지를 백업하고 Docker29.9.2/현재 CLI·helper·pod-init을 적용했다. 기존 unit의 누락된 serve 인자 한 줄만 고쳤다. 실제 helox 환경의 runtime4OK, packaging24.2→25.0 설치·업데이트, Go uuid1.5.0→1.6.0·관찰 build가 PASS다. 사용자 체험 폴더는 비어 있으며 개발 의존성은 보존했다. IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES, MISSING=[]는 이 local 체험 범위다. 같은 소스의 full/native/CPP/CI는 반복하지 않았고 정식 release 기록·본체 self-update·공개 배포 완료로 전용하지 않는다.

### 2026-11-01 CLI 시작 화면 아트 — COMPLETE (local 표시 범위)

사용자가 [단순 문 로고·가로 그라데이션·별빛의 최종 화면](./evidence/cli-launcher/art-refresh-20261031.md)을 확정했다. 기존 brand owner와 draw paint 연결 두 파일만 수정했고 final frozen Quick93.086s/security21.332s/docs0.366s PASS다. IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES, MISSING=[]다. 로고는HELIOPAUSE 왼쪽에만 두고 두 번째 줄은 전체 폭을 유지하며 네 메뉴/footer·NO_COLOR·terminal lifecycle은 기존 경계를 재사용한다. 불필요한 큰 심볼 배치 코드는 제거했고 새 runtime mode/package/dependency는 없다.

승인된3개 PNG만 현재 문서에 남겼다. 이번 작업의 미추적 초안18개와 이전 인계는 원본을 보존해 raw로 옮겼고 관련 없는 사용자 자료는 건드리지 않았다. Native macOS hosted·CPP/CUDA를 반복하거나 이전358 전체 CI를 새 artwork의 결과로 전용하지 않는다. 현재 설치본 교체·push·main merge/tag/publish와 M13-004 전체 완료는 별도다.

### 2026-10-31 사용자 요청: 이름 결정 전 전체 코드·보안 감사 — COMPLETE

단일 active M13-004 안에서 요청한 [전체 코드·보안 감사](./evidence/m13-004-convenience/full-source-audit-20261031.md)를 완료했다. 시작21a의 inventory/고정 도구/문맥 보완 분석과 신뢰 경계·미사용·중복·복잡도 검토를 연결하고, 확정 F01..F08과 연결 회귀를 기존 owner에서 최소 수정했다. 최종 코드358/treee1ca의 일반/CPU·세 CUDA가 각각 Required10SUCCESS이며 actual source·원본·skip0·helper 회수·복원을 확인했다. [단일 최종 증거](./evidence/m13-004-convenience/full-source-audit-20261031.json)가 범위/발견/원본 해시/당시 실패와 재사용을 소유한다.

감사 범위는 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES / MISSING=[]다. 모든 미지 버그·취약점 부재나 M13 전체 완료를 뜻하지 않는다. 로컬 Host29.8.1 전역 전환·현재 서명 배포는 수행하지 않았으며 이름·채널·배포 전 acceptance는 별도다. 기존 성공과 불변 full/native/CPP/CI를 되풀이하지 않고 사용자 이름 검토를 유지한다. 원래 runtime component plan의 APPLIED 상태를 영구 활성화로 바꾸지 않는다.


### 2026-11-05 최신 재개 상태 — 환경 차단 / BLOCKED

M13-004의 현재 실행은 BLOCKED다. WSL root `/dev/sdd` ext4가 `emergency_ro`이며 sandbox launcher가 read-only filesystem, sed가 Bus error, Python이 segmentation fault로 실패했다. 메모리 여유 약21GB 확인은 OOM 원인 확정이 아니며 정확한 filesystem 하위 원인은 미확정이다. Windows의 저장소 mount는 rw이고 마지막 branch ref는5fd908b4cd459b3478c1dde97f5d26d46e769ea6로 보존됐다.

Bash/Go 실제 성공은6e74f22, npm 시험 두 기대값 교정은5fd908b에 보존했다. Updater user-v3의 plan/apply는 성공했고 candidate는 Root의 모든 tracked file과 같았으나 verify bootstrap 중 작업 process가 exit135로 종료됐다. configuration PASS/필수 검사 완료로 처리하지 않는다. 이전 root verify-v2의 실패도 그대로 보존한다. CI와20분 CI worker는 시작하지 않았다. 마지막 실제 native 종료에서 Host/source 동일·helper 종료·자체 사본 삭제를 확인했지만 환경 장애 후 Docker 상태를 새로 조회하지 못했다.

해제 조건: WSL filesystem과 기본 도구의 정상 읽기/쓰기 복구 후 INDEX/실제 Git·입력/전용 updater state부터 확인한다. 자동 remount/fsck/WSL 종료·디스크 삭제는 실행하지 않았다. Root/private 코드 hash와 실패한 verification 상태를 확인한 뒤 기존 도구의 verify부터 이어간다. 검증된 Bash/Go native·CPP138/core33ae를 같은 입력으로 반복하지 않는다. 원본 docs/github-logs/goal-implementation-20261101/host-emergency-ro-20261105.log와 s04-bash-owner-user-v3.json을 참조한다. 전체 설계/S04/배포는 미완료다.

### 2026-11-05 WSL 재부팅 후 안전 재개 준비·사용자 Node 전환

[복구·재개 원장](./evidence/m13-004-convenience/wsl-recovery-20261105.md)은5fd908b/source와 중단된 verify의 경계를 보존한다. Git full fsck/현재 rw·fsync readback, runtime4OK 및 기존 observer0700 복구 후 실제 npm 필수3검사 COMPLETED/ALLOW를 확인했다. 환경 read-only 차단은 현재 관측에서 해제됐으나 `/tmp` private clone/정확한 Semgrep 공급물·SDK/frozen state는 소실돼 재구성이 먼저다. 기존 성공 native/CPP/core를 반복하지 않는다. 사용자 nvm Node26.13.0/npm11.20.0 전환과 기존 CLI 보존은 로컬 확인됐으며 helox pinned runtime은 불변이다. 명시 재생성 cache3236511744B를 정리했고 source/실패 원본은 F recovery-20261105의 검증 bundle/archive에 보관했다. Offline VHDX 압축 성공은 새 F result.json/restart-verification.json으로만 판단한다. 이번은 복구 준비이며 전체 S04/보강·배포 완료가 아니다.

## 2. 상태와 queue 규칙

Milestone과 work item에 같은 네 상태를 사용한다.

```text
NOT_STARTED
IN_PROGRESS
BLOCKED
COMPLETE
```

- 동시에 하나의 work item만 `IN_PROGRESS`로 둔다.
- 표의 순서가 기본 실행 우선순위다. 선행 항목이 완료되지 않았으면 다음 항목을 시작하지 않는다.
- `Ready: Yes`는 선행조건과 필수 입력이 준비되어 바로 시작할 수 있다는 뜻이며 완료나 성공을 뜻하지 않는다.
- `BLOCKED`에는 blocker, 마지막 확인 결과와 해제 조건을 기록한다.
- `COMPLETE`는 acceptance criteria와 요구 check가 실제 evidence로 확인된 경우에만 사용한다.
- 구현 중 발견한 별도 작업은 현재 항목의 완료에 필수일 때만 좁게 추가한다. 미래 milestone 항목은 해당 milestone이 가까워질 때 연다.
- scope가 Milestone, Architecture, Domain invariant를 바꾸면 queue에서 우회하지 않고 canonical 설계 문서를 먼저 갱신한다.


## Step 13 Invariant

1. Current Work Queue에는 현재 milestone의 실행 항목만 둔다.
2. 한 번에 하나의 work item만 `IN_PROGRESS`다.
3. work item은 canonical 설계를 링크하며 상세 결정을 복제하지 않는다.
4. 선행조건과 acceptance가 없는 작업을 시작하지 않는다.
5. 실제 검증 evidence 없이 `COMPLETE`로 표시하지 않는다.
6. 외부 상태 변경과 repository-wide 보안 설정은 필요한 명시적 승인을 받는다.
7. 미래 capability의 placeholder package, config 또는 CI job을 queue가 요구하지 않는다.
8. 독립 repository에서는 현재 next item부터 Queue 순서대로 실행한다.

## 누락 점검

- [x] 현재 milestone, active item과 next item
- [x] 단일 상태 모델과 Ready 의미
- [x] M0의 순차 work item과 dependency
- [x] 각 항목의 canonical document route
- [x] Outcome, Work와 Acceptance
- [x] 완료 evidence 기록 형식
- [x] blocker와 queue 변경 규칙
- [x] repository-wide 변경 승인 경계
- [x] M1 이후 작업의 지연 생성 원칙
- [x] 독립 repository의 정확한 재개 지점


## 보관된 이력 (R0-4, 2026-11-10)

아래 구간은 내용 변경 없이 `docs/planning/archive/`로 옮겼다(링크 경로만 조정). 각 항목은 원문 제목(날짜·ID·당시 상태)만 남긴 색인이며 당시 기록이지 현재 판정이 아니다.

### [Current Work Queue §1 현재 상태 이력 (2026-10-31 이전 checkpoint)](./archive/02-queue-history-20261031-and-earlier.md)

원본 줄 125-3070, 제목 242개.

  - 2026-10-31 G4 기능 acceptance·CI446 완료
  - 2026-10-31 G4 Bun 선언/TS 기능 묶음 CI446 등록
  - 2026-10-31 H1/H2·승인된 Linux150분 전체 묶음 CI442 등록
  - 2026-10-30 P7 grouped CI441 등록
  - 2026-10-30 P6-G3 명시 Bun JS 실행 named local 검증
  - 2026-10-30 CI440 Bun cold/current 앱·Node CLI 전체 묶음 등록
  - 2026-10-30 P6-G2 Bun 현재 앱·Node CLI named local 검증
  - 2026-10-30 CI439 PNPM native/current 앱·CLI hosted 완료
  - 2026-10-30 CI439 PNPM 전체 묶음 등록
  - 2026-10-30 P6-F2/F3 PNPM 현재 실행·CLI named local 검증
  - 2026-10-30 CI438 pipx native/current 앱·CLI hosted 완료
  - 2026-10-30 P6-F1 PNPM cold 선택 DATA named local 완료
  - 2026-10-29 CI433 current Python 경계·앱 hosted 완료
  - 2026-10-29 CI434 common observer checkpoint 등록
  - 2026-10-30 CI438 E2/E3 전체 묶음 등록
  - 2026-10-30 P6-E3 pipx current 앱·CLI named local 완료
  - 2026-10-30 CI437 공통 FD·새 UV hosted 완료
  - 2026-10-30 P6-E2-B pipx native-run resolver local 검증
  - 2026-10-30 P6-E2-A 고정 pipx run 준비 ABI local 검증
  - 2026-10-30 공통 FD 수정 CI437 등록
  - 2026-10-30 재시작 준비·CI436 최초 실패 보존
  - 2026-10-30 E1 frozen original 공유 local 검증
  - 2026-10-30 grouped CI436 등록·실제20분 조회
  - 2026-10-30 D8 native-run hosted wiring local 완료
  - 2026-10-30 CI435 native DATA hosted 완료
  - 2026-10-30 D7 native UV CLI named local 완료
  - 2026-10-30 D6 current native-run application named local 완료
  - 2026-10-30 D5 production native-run resolver named local 완료
  - 2026-10-30 grouped CI435 등록·실제20분 조회
  - 2026-10-30 D4B4 current native hosted wiring local 완료
  - 2026-10-30 D4B3 native source/console DATA named local 완료
  - 2026-10-30 D4B2 exported-cache query/census named local 완료
  - 2026-10-30 CI434 완료 / whole 실패·Linux named 성공
  - 2026-10-30 D4B1 unsupported DATA export 경로 종료·정리
  - 2026-10-30 D4A public pre-query binding named local 완료
  - 2026-10-29 D3 exact readonly ABI named local 완료
  - 2026-10-29 D3 cached query 부분 checkpoint·정리
  - 2026-10-29 P6-D3 고정 cached UV query 경계 착수
  - 2026-10-29 D2 네 admission 최초 오류 보존 named local 완료
  - 2026-10-29 P6-D2 기존 admission 최초 오류 보존 착수
  - 2026-10-29 D1 native 원본 오류 관측·완료 사본 정리
  - 2026-10-29 CI433 actual poll01·독립 D1 첫 거부
  - 2026-10-29 P6-D1 native tool-run DATA 진단 착수
  - 2026-10-29 grouped CI433 등록·실제20min 감시
  - 2026-10-29 P6-C2 grouped hosted wiring local 완료
  - 2026-10-29 P6-C2 기존 current CI cohort 연결 착수
  - 2026-10-29 P6-C current Python application workflow named local 완료
  - 2026-10-29 P6-C current Python application runtime 착수
  - 2026-10-29 D2 standard-library hostile5 named local 완료
  - 2026-10-29 D2 readonly positive·PID 관측
  - 2026-10-29 CI432 current selected Python backend hosted 완료
  - 2026-10-29 D2 v2 partial·readonly oracle 반증
  - 2026-10-29 D2 first actual과 test-only v2 교정
  - 2026-10-29 D2 Python hostile 실제 경계 착수
  - 2026-10-29 D1 current Python 실제 boundary8 named local 완료
  - 2026-10-29 D1 current Python 실제 boundary 착수
  - 2026-10-29 grouped CI432 등록·실제20min 감시
  - 2026-10-29 B2-C selected Python hosted wiring local 완료
  - 2026-10-29 B2-C selected Python hosted 착수
  - 2026-10-29 B2-B initial Python backend named local 완료
  - 2026-10-29 CI431 current inputs hosted 완료
  - 2026-10-29 B2-B Python selected-program backend 착수
  - 2026-10-29 B2-A 공유 captured program named local 완료
  - 2026-10-29 B2-A 기존 captured program 본문 공유 착수
  - 2026-10-29 CI431 등록·실제20min 감시
  - 2026-10-29 B1 readonly4 hosted 연결 local 완료
  - 2026-10-29 CI430 Go local 진단·B1 hosted gap 착수
  - 2026-10-29 CI430 관찰 손실 실패 확인
  - 2026-10-29 완료 시험 산출물 정리
  - 2026-10-29 CI429 grouped Node/CLI hosted 완료
  - 2026-10-29 P6-B1 named local readonly DATA 완료
  - 2026-10-29 P6-B1 readonly Python inputs 착수
  - 2026-10-29 P6-B Python runtime 기존 owner 감사
  - 2026-10-29 grouped CI430 등록
  - 2026-10-29 P6-A4 current hosted wiring local 완료
  - 2026-10-29 P6-A4 grouped current execution hosted 착수
  - 2026-10-29 P6-A3 current Python preparation actual local 완료
  - 2026-10-29 P6-A3 current Python preparation 착수
  - 2026-10-29 P6-A2 reader와 replay 교정 local 완료
  - 2026-10-29 P6-A2 controlled intake reader 착수
  - 2026-10-29 P6-A1 exact console DATA local 완료
  - 2026-10-29 P6-A1 console entry DATA 착수
  - 2026-10-29 P6 existing owners/pinned UV source 착수 감사
  - 2026-10-29 C7 real post-Policy CLI refusal local 완료
  - 2026-10-29 C7 post-Policy CLI refusal 착수
  - 2026-10-29 current grouped CI429 등록
  - 2026-10-29 C6 hosted wiring local 완료
  - 2026-10-29 C6 grouped hosted wiring 착수
  - 2026-10-29 CI428 source558 backend hosted 완료
  - 2026-10-29 C5 npx/npm exec actual CLI local 완료
  - 2026-10-29 C5 safe output/npx CLI 착수
  - 2026-10-29 C4 application workflow actual local 완료
  - 2026-10-29 C4 application workflow 착수
  - 2026-10-29 C3d3 same-source12 actual local 완료
  - 2026-10-29 C3d3 actual deadline/readonly self-write 착수
  - 2026-10-29 C3d2 current10 hostile actual local 완료
  - 2026-10-29 C3d2 actual hostile 경계 착수
  - 2026-10-29 C3d1 selected Node 신호·취소·독립 overflow actual 완료
  - 2026-10-29 C3d1 actual Node failure 경계 착수
  - 2026-10-29 승인된 Artifact Node foundation CI428 등록
  - 2026-10-29 실제 Artifact Node normal/nonzero backend local 완료
  - 2026-10-29 CI427 공통 후보 hosted qualification 완료
  - 2026-10-29 Artifact Node 첫 actual 실패와 제한 진단
  - 2026-10-29 승인된 공통4commit CI427 등록
  - 2026-10-29 기존 Linux CI kernel parent 연결 로컬 검증
  - 2026-10-29 CI426 종료와 target-image dependency 검증
  - 2026-10-29 P5-C3c2a 읽기 전용 whole-tree 입력 로컬 검증
  - 2026-10-29 P5-C3c1 외부 kernel root terminal 로컬 검증
  - 2026-10-29 P5-C3b offline whole-set materialization 로컬 검증
  - 2026-10-29 CI425 완료 소비 — Linux preparation PASS / macOS·Required FAIL
  - 2026-10-29 P5-B2f production preparation 실제 local 검증
  - 2026-10-29 P5-B2e preparation 로컬 검증
  - 2026-10-29 사용자 연속 CI/push 권한
  - 2026-10-29 승인된6commit/source4ec CI424 등록
  - 2026-10-29 P5-B2d controlled Node entry bytes 로컬 검증
  - 2026-10-29 P5-B2c fresh whole inspection 로컬 검증
  - 2026-10-29 CI423 current ordinary hosted qualification
  - 2026-10-29 P5-B2b Node composition / execution records 로컬 검증
  - 2026-10-28 P5-B2a metadata context 로컬 검증
  - 2026-10-28 P5-B2 기존 owner 재사용 감사
  - 2026-10-28 CI423 첫20min 소비와 P5-B1 로컬 검증
  - 2026-10-28 P5-B1 selected bytes / Node runtime 착수
  - 2026-10-28 승인된4commit CI423 시작
  - 2026-10-28 CI422 canonical qualification 입력 연결 교정
  - 2026-10-28 CI422 same-source report DATA 진단
  - 2026-10-28 CI422 실패 원본·소스 대조
  - 2026-10-28 P5-A 구조적 진입점 선택 로컬 검증
  - 2026-10-28 P5-A exact bin 선택 착수
  - 2026-10-28 CI 대기 중 P5 baseline 독립 감사
  - 2026-10-28 승인된28commit source1203445 CI422 시작
  - 2026-10-28 C-e3 common readiness named local 완료
  - 2026-10-28 C-e2 original ZIP/current graph/native readiness 한정 완료
  - 2026-10-28 C-e1 CPU 진단 audit 한정 종료
  - 2026-10-28 C-d2 existing hosted tools wiring 한정 완료
  - 2026-10-28 C-d1 fixed native pipx original 한정 완료
  - 2026-10-28 C-c3 public production pipx CLI 한정 완료
  - 2026-10-28 C-c2 managed approved pipx backend 한정 완료
  - 2026-10-28 C-c1 fresh approved pipx backend 한정 완료
  - 2026-10-28 C-b3b original-aware native upgrade/no-op 한정 완료
  - 2026-10-28 C-b3a inert original replay/port 한정 검증
  - 2026-10-28 C-b2 public pipx selection/factory 한정 완료
  - 2026-10-28 C-b1 고정 설치 공급물 entry 한정 완료
  - 2026-10-28 C-a 실제 native metadata 한정 완료
  - 2026-10-28 C-a query/diagnostic 중간 검증
  - 2026-10-28 C-a native 관찰의 첫 실제 경계 확인
  - 2026-10-28 CI421 Required10 성공·원본 대조
  - 2026-10-28 CI421 세 번째20분 직접 확인
  - 2026-10-28 CI421 두 번째20분 직접 확인
  - 2026-10-28 C3c3b3b CPP 전용 Artifact profile 로컬 검증
  - 2026-10-28 CI421 첫20분 직접 확인
  - 2026-10-28 승인된6commit checkpoint CI421 시작
  - 2026-10-28 P4-O4-C3c3b3a offline constructor/Go topology 로컬 검증
  - 2026-10-28 P4-O4-C3c3b2 complete native output 로컬 검증
  - 2026-10-28 P4-O4-C3c3b1 전체 native DATA census 로컬 검증
  - 2026-10-28 P4-O4-C3c3a frozen whole wheel/RO input 로컬 검증
  - 2026-10-28 P4-O4-C3c2 native program/offline 진단 로컬 검증
  - 2026-10-28 P4-O4-C3c1 조회 전 whole wheel gate 로컬 검증
  - 2026-10-28 P4-O4-C3b pipx original upgrade 입력 로컬 검증
  - 2026-10-28 P4-O4-C3a pipx inert layout 로컬 검증
  - 2026-10-28 P4-O4-B3d 기존 hosted actual 소비자 연결·로컬 recipe 검증
  - 2026-10-28 P4-O4-B3c 실제 graph·Evidence와 DATA 복구 로컬 검증
  - 2026-10-28 P4-O4-B3b complete DATA producer·metadata·package 로컬 검증
  - 2026-10-28 P4-O4-B3a complete DATA 선택·mode 로컬 검증
  - 2026-10-28 공통 checkpoint push의 외부 승인 대기
  - 2026-10-28 P4-O4-B2d required 검사·Evidence 로컬 검증
  - 2026-10-28 P4-O4-B2c acquired ZIP 실제 transaction 로컬 검증
  - 2026-10-28 P4-O4-B2c 공통 transaction 연결·readiness 교정 중
  - 2026-10-28 P4-O4-B2b fixed observer profile 로컬 검증
  - 2026-10-28 P4-O4-B2a 입력과 고정 program 로컬 검증
  - 2026-10-28 CI420 최종 확인과20분 직접 조회 요청
  - 2026-10-27 CI420 첫10분 실제 조회
  - 2026-10-27 CI420 새 공급과 공통 검사 소비자 검증 시작
  - 2026-10-27 P4-O4-B1d 명시 공급과 로컬 소비자 검증
  - 2026-10-27 P4-O4-B1d 승인된 공급 확대 착수
  - 2026-10-27 P4-O4-B1b/c 검증과 명시 공급 확대 승인
  - 2026-10-27 P4-O4-B1a ZIP snapshot 로컬 검증
  - 2026-10-27 P4-O4-B1a 착수
  - 2026-10-27 CI419 최종 결과 확인
  - 2026-10-27 CI419 첫10분 결과 조회
  - 2026-10-27 CI419 공통 DATA volume 회귀 시작
  - 2026-10-27 P4-O4-C2 한정 로컬 검증
  - 2026-10-27 P4-O4-C2 착수
  - 2026-10-27 P4-O4-C1 한정 로컬 검증
  - 2026-10-27 P4-O4-C1 착수
  - 2026-10-27 CI418 원격 검증 확인
  - 2026-10-27 CI418 공통 runtime owner 검증 시작
  - 2026-10-27 P4-O4-A 한정 로컬 검증
  - 2026-10-27 P4-O4 착수
  - 2026-10-27 P4-O3 한정 로컬 검증
  - 2026-10-27 P4-O3 착수
  - 2026-10-27 P4-O2 한정 로컬 검증
  - 2026-10-27 P4-O2 착수
  - 2026-10-27 P4-O1 한정 로컬 검증
  - 2026-10-27 P4-O1 착수
  - 2026-10-27 CI417 최종 검증
  - 2026-10-27 CI417 교정 후보 인계
  - 2026-10-27 CI416 실패 소비
  - 2026-10-27 P4-N2D 실제 native 인계
  - 2026-10-27 P4-N2C 정전 후 로컬 체크포인트
  - 2026-10-24 CI402 ordinary checkpoint 확인
  - 2026-10-24 P2c3 archive 설치 동작 사실 전달 착수
  - 2026-10-24 P2c5-B3 archive/layout 착수 감사
  - 2026-10-24 P2c5-B3b native 배치 형태와 dirname 연결
  - 2026-10-24 B3b2a layout envelope 로컬 검증
  - 2026-10-24 B3b2b Node bin code 로컬 검증
  - 2026-10-24 B3c 전체 inspection layout 결합 착수 감사
  - 2026-10-24 B4 production inspection 착수 감사
  - 2026-10-24 B4b1 실행용 payload 재구성 착수 감사
  - 2026-10-25 B3 후속·B4 production inspection 묶음 원격 qualification
  - 2026-10-24 CI404 묶음 checkpoint 확인
  - 2026-10-24 B2·B3a·B3b1 묶음 원격 qualification 시작
  - 2026-10-24 CI403 묶음 checkpoint 확인
  - 2026-10-24 P2c2 actual public 선택·update/no-op 로컬 검증
  - 2026-10-24 P2c2 native producer 경계 로컬 검증
  - 2026-10-24 CI401 ordinary native admission 확인과 P2c2 전송 검증
  - 2026-10-24 CI401 첫 타이머 확인과 P2c2 준비
  - 2026-10-24 P2c1 bounded lock·독립 source binding 로컬 검증
  - 2026-10-24 P2c1 독립 selection 착수와 CI401 타이머
  - 2026-10-24 P2b2 native admission과 실제 CI helper 로컬 검증
  - 2026-10-24 P2b2 M6 제한 변경 승인과 Node 역할 보완
  - 2026-10-24 P2b2 CONTROL lock와 M6 명시 계약 차이
  - 2026-10-24 P2b2 exact guest startup 로컬 범위
  - 2026-10-24 P2b2 actual guest 공급·admission 착수 감사
  - 2026-10-24 P2b2 native 배포·기존 설치/복구 로컬 검증
  - 2026-10-24 P2b2 native 명시 출력 로컬 checkpoint
  - 2026-10-24 P2b2 단일 owner·유지보수 로컬 checkpoint
  - 2026-10-24 P2b1 native 공급·형식 로컬 검증
  - 2026-10-24 P2 착수 감사·공급 검증 순서
  - 2026-10-24 G127 완료·P2 인계
  - 2026-10-24 G127 CI395 성공·CI396 cu126 지연 조사
  - 2026-10-23 G127 최종 로컬 검증·source 반영
  - 2026-10-23 G127 도구 호환성 — 최초 오류를 보존하고 최소 pin 갱신
  - 2026-10-23 G127 첫 configuration 확인 — 모듈 형식만 보완
  - 2026-10-23 G127 착수 감사 — 기존 updater 재사용
  - 2026-10-23 P1 COMPLETE — 실제 네 profile·문서 검증과 정리
  - 2026-10-23 CI390 확인과 검증 주기 조정
  - 2026-10-23 CI389 macOS build — 당시 결과
  - 2026-10-23 CI388 선행 회귀
  - 2026-10-22 순서 결정 — P1 완료 후 Go1.27.3, 그다음 P2
  - 2026-10-21 owner scope — client expansion and full-name brand
  - Runtime upgrade maintenance — 2026-10-15 priority change
  - SEC-2026-10-12 — Scheduled Security Go patch

### [Current Work Queue §3~§5 (M0 실행 Queue·Work Item·Historical work items·완료 기록)](./archive/02-queue-m0-m11-work-items.md)

원본 줄 3090-5086, 제목 69개.

- 3. M0 실행 Queue
  - M1 실행 Queue
  - M2 Entry Queue
  - M3 Entry Queue
  - M4 Entry Queue
  - M5 Entry Queue
  - M6 Entry Queue
  - M7 Entry Queue
  - M8 Entry Queue
  - M8-001 — production trust hardening entry·canonical contract
  - M8-002 — Trusted Host tool identity·executor·local Docker endpoint
  - M8-003 — observer supervisor·exclusive endpoint·process-scoped composition
  - M8-004 — connection identity latch·production observation completeness
  - M8-005 — least-privilege typed resolver network-policy helper
  - M8-006 — runtime lock single source of truth·CI integration
  - M8-007 — hostile-boundary regression·Linux security requalification
  - M7-002 — ecosystem flow·fixture regression
  - M7-003 — Linux·macOS·WSL2 CLI qualification
  - M7-004 — evidence·result·resilience qualification
  - M7-005 — security workflow·scheduled gate qualification
  - M7-006 — MVP final audit·documentation·completion
  - M7-001 — MVP qualification entry·evidence matrix
  - M6-001 — GitHub Releases Standalone entry decision
  - M6-002 — exact public input·API boundary foundation
  - M6-003 — GitHub Release resolve·Acquire·integrity
  - M4-003 — recursive dependency inspection과 set-level Policy
  - M4-004 — Verified Manifest·SBOM과 trusted Staging
  - M4-005 — offline npm Promotion과 atomic target
  - M4-006 — M4 qualification과 M5/M6 handoff
- 4. Work Item
  - M5-001 — PyPI/pip Expansion entry decision
  - M5-002 — Python/pip runtime identity lock과 PyPI reference foundation
  - M5-003 — isolated PyPI resolver·target tag·egress policy
  - M5-004 — wheel intake·integrity·static inspection
  - M5-005 — Python dynamic inspection·PEP 517 sdist build
  - M5-006 — generic staging·offline pip Promotion·CLI integration
  - M5-007 — M5 qualification·npm regression·M6 handoff
- 5. Historical work items
  - M3-001 — Linux Dynamic Inspect entry decision
  - M3-002 — gVisor runtime lock과 capability probe
  - M3-004 — gVisor Sandbox Session lifecycle
  - M3-003 — Sandbox Session·Observation Domain과 Port
  - M3-005 — gVisor raw observation collector
  - M3-006 — Dynamic observation normalization
  - M3-007 — M3 Policy와 npm inspect wiring
  - M3-008 — M3 qualification과 M4 handoff
  - M4-001 — npm Install and Promotion entry decision
  - M4-002 — Install Context와 locked npm resolver boundary
  - M0-001 — Project Identity 결정
  - M0-002 — Go·Tool·CI Identity lock
  - M0-003 — Go module과 최소 process bootstrap
  - M0-004 — Canonical check runner foundation
  - M0-005 — Static analysis와 Quick profile 활성화
  - M0-006 — Quick·Docs·Required CI foundation
  - M0-007 — Minimum Go·macOS와 repository gate 검증
  - M0-008 — M0 qualification과 M1 handoff
  - M1-001 — Domain workflow entry decision
  - M1-002 — Domain identity와 Run state foundation
  - M1-003 — Check result·Port와 deterministic fake
  - M1-004 — Inspect orchestration과 Policy v1
  - M1-005 — CLI result contract와 synthetic vertical test
  - M1-006 — M1 qualification과 M2 handoff
  - M2-001 — npm Static Inspect entry decision
  - M2-002 — npm reference·metadata resolve foundation
  - M2-003 — Controlled Intake와 declared integrity verification
  - M2-004 — npm tarball static inspection과 Evidence Store
  - M2-005 — M2 Policy·CLI wiring과 controlled vertical test
  - M2-006 — M2 qualification과 M3 handoff
- 5. Work Item 완료 기록

### [Current Work Queue §6 (Historical state records, M12 status)와 2026-10-07~10-26 checkpoint](./archive/02-queue-m12-m13-history-20261007-20261026.md)

원본 줄 5087-6509, 6534-7426, 제목 103개.

- 6. Historical state records and current M12 status
  - M11 work breakdown
  - M12 Ecosystem Expansion queue
  - M12-001 acceptance 종료 (2026-10-03)
  - M12-002 시작 baseline audit (2026-10-04)
  - M12-002 parser/private snapshot checkpoint (2026-10-04)
  - M12-002 acquisition/SumDB checkpoint (2026-10-04)
  - M12-002 project inspection/verified cache checkpoint (2026-10-04)
  - M12-002 isolated source/guarded transaction checkpoint (2026-10-04)
  - M12-002 guarded download/dependency-free project checkpoint (2026-10-04)
  - M12-002 Go resolver 처리량 측정 (2026-10-04)
  - M12-001 종료 전 이력 (당시 상태, 현재 판정 아님)
  - M12-005에 남는 release support 결정
  - M13 Production Release & Operations queue (M12 완료 후)
  - M13-001 — protected repository activation
  - M13-002 — protected branch qualification
  - M13-003 — canonical release build and OS packages
  - CLI-UX-001 — user-facing terminal remediation
  - Prepublication corrections — CLI-UX-002 then SEC-UPD-001
  - CLI-UX-003 — branded interactive launcher
  - M13-004 — convenience installation channels
  - M12 baseline audit status
  - M11 post-qualification release hardening
  - M12-002 Go resolver/source 보완 checkpoint (2026-10-04)
  - M12-002 Go verified cache file budget checkpoint (2026-10-04)
  - M12-002 Go build input preparation checkpoint (2026-10-04)
  - M12-002 observed Go build / output wiring 진행 (2026-10-05)
  - M12-002 local acceptance 종료 (2026-10-05)
  - M12-003 시작 baseline audit (2026-10-05)
  - M12-003 실제 parser/source acquisition 부분 checkpoint (2026-10-05)
  - M12-003 isolated source·verified cache 부분 검증 (2026-10-05)
  - M12-003 guarded Cargo add 부분 checkpoint (2026-10-05)
  - M12-003 observed build와 중단 재개 qualification (2026-10-06)
  - M12-003 압축 후 qualification 재개 (2026-10-06)
  - M12-003 재개 full 통과와 종료 검사 (2026-10-06)
  - M12-004 frozen public source·signature 부분 checkpoint (2026-10-06)
  - M12-004 시작 baseline audit (2026-10-06)
  - M12-003 local acceptance 종료 (2026-10-06)
  - M12-004 local acceptance 종료 (2026-10-07)
  - M12-002~004 원격 전달·CI 후속 시작 (2026-10-07)
  - 원격 첫 macOS 실패와 동일 입력 교정 (2026-10-07)
  - 원격 Cargo fixture 작업 디렉터리 교정 (2026-10-07)
  - M12-002~004 원격 qualification과 성공 데이터 정리 완료 (2026-10-07)
  - M12-005 baseline audit 시작 (2026-10-07)
  - M12-005 추가 미사용 API 정리 checkpoint (2026-10-07)
  - M12-005 qualification·feature freeze 완료 (2026-10-07)
  - M12-02 final red-team baseline audit 시작 (2026-10-07)
  - M12-02 검토·remediation 완료 (2026-10-07)
  - 2026-10-25 P2 CI405 최초 오류와 B4d 로컬 교정
  - 2026-10-25 P2 B5b1 actual-operation records local checkpoint
  - 2026-10-25 P2 B5b2 private Host materialization local checkpoint
  - 2026-10-25 CI406 named remote checkpoint 확인
  - 2026-10-25 P2 B5b3 guarded project transaction local checkpoint
  - 2026-10-25 P2 B5b4 existing CI actual consumer local checkpoint
  - 2026-10-25 P2 B5c PNPM production CLI local checkpoint
  - 2026-10-25 CI407 backend named remote qualification
  - 2026-10-25 P2 B5d independent native original ownership local qualification
  - 2026-10-25 CI408/409 remote qualification과 연속 대기 교정
  - 2026-10-25 P3-A Bun signed publisher supply local checkpoint
  - 2026-10-25 P3-B1 Bun ZIP/ELF local checkpoint
  - 2026-10-25 P3-B2a candidate/native query local checkpoint
  - 2026-10-25 P3-B2b1 Bun owner/frozen maintenance local checkpoint
  - 2026-10-25 CI410 cu130 checkpoint와 연속 timer 소비
  - 2026-10-25 P3-B2b2-A Bun production supply local checkpoint
  - 2026-10-25 P3-B2b2-B complete release/installed tuple local checkpoint
  - 2026-10-25 CI411 timer 실제 확인과 남은 gate
  - 2026-10-25 P3-B3a Bun actual raw lock / PNPM shared scope
  - 2026-10-25 P3-B3b Bun parser·독립 npm source·snapshot
  - 2026-10-25 P3-B3c typed native Bun 선택 연결
  - 2026-10-25 P3-B4 native layout·lifecycle 착수
  - 2026-10-26 CI413 timer와 독립 P4-A 착수
  - 2026-10-26 P4-A frozen virtual project input local checkpoint
  - 2026-10-26 P4-B native lock/export 착수 감사
  - 2026-10-26 P4-B native UV lock/export local checkpoint
  - 2026-10-26 P4-C actual export 형식과 독립 binder 지도
  - 2026-10-26 P4-C project export binding local checkpoint
  - 2026-10-26 P4-D native project consumer 착수
  - 2026-10-26 P4-D native project selection local checkpoint
  - 2026-10-26 CI413 grouped Bun qualification과 실제 timer 소비
  - 2026-10-26 P4-E add/version-changing selection 착수
  - 2026-10-26 P4-E native add/version delta local checkpoint
  - 2026-10-26 P4-F native project cache 착수
  - 2026-10-26 P4-F native project approved cache local checkpoint
  - 2026-10-26 P4-G project/venv transaction 착수
  - 2026-10-26 P4-G project controls/venv filesystem checkpoint
  - 2026-10-26 P4-H native project publication 착수
  - 2026-10-26 P4-H native project installed backend checkpoint
  - 2026-10-26 P4-I public project CLI 착수
  - 2026-10-26 P4-I public UV project CLI checkpoint
  - 2026-10-26 P4-J native project adoption 착수
  - 2026-10-26 P4-J native project adoption checkpoint
  - 2026-10-26 P4-K distinct lock-only 착수
  - 2026-10-26 P4-K1 control-only typed component checkpoint
  - 2026-10-26 P4-K2 lock-only approved publication checkpoint
  - 2026-10-26 P4-K3 public lock-only CLI checkpoint
  - 2026-10-26 P4-L persistent tool baseline 착수
  - 2026-10-26 P4-L1 native persistent tool receipt
  - 2026-10-26 P4-L2 whole tool wheel binding
  - 2026-10-26 P4-L3 fresh native UV tool selection
  - 2026-10-26 P4-L4A final interpreter materializer
  - 2026-10-26 P4-L4B target/receipt guards
  - 2026-10-26 P4-L4C fresh receipt/links atomic helper
  - 2026-10-26 P4-L4D current approved tool backend
