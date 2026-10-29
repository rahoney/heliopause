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
Active work item status: IN_PROGRESS — P1 uv pip and G127 Go1.27.3 COMPLETE; P2 pnpm named backend/CLI/native/failure qualification recorded; P3 Bun cache/private/transaction/CLI/native ownership locally qualified; whole P2/P3 final acceptance remains
Last completed substep: P6-B2-A shared existing captured program owner named local QUALIFIED; canonical9/Quick304.561s/affected Race35.809s; controlled Node12/195.49s and actual CLI4/110.08s PASS; source1356/old1354 unchanged/CPP131; cleanup179 copies/~6.35GiB.
Next action: automatic public GET monitor/session61457/pid2781030 queried07:27:09→07:27:10.790120UTC, 8SUCCESS/Linux running; next07:47:10.790138UTC/KST16:47:10. Read /tmp/haa-ci431-autopoll/state.json and consume new originals; no push. Continue B2-B Python execution baseline/exact map; native alias/CLI/final remain OPEN.
Current substep: M13-004/P6-B2-B Python selected-program profile/backend IN_PROGRESS; source1356 baseline and exact map/second review recorded before source. CI431/source0d96 excludes this helper and remains PENDING; no push while active.
Acceptance audit: CI429/source9eb all10SUCCESS/checkoutb2eb/tree51a9/2996·1343; Linux97/noSKIP/CPP131/controlled12/application2/CLI4/helper16wait0/final0; later C7/P6 outside. CI425 macOS deadline cause OPEN; whole P4/P5-P7/final4profiles/signed OPEN.
Independent local next: P6 Python ARTIFACT user-run/admission baseline reusing qualified whole readonly inputs/current context/console source; native UVx golden equivalence/CLI remain OPEN.
P3 Bun: CI413 named hosted/whole acceptance OPEN; P4 — named current pipx/UV hosted, final OPEN; P5 — current Node/CLI grouped CI429 QUALIFIED, whole final OPEN; P6 — console DATA/reader/current preparation3 local QUALIFIED, user runtime/CLI OPEN; P7 — NOT_STARTED
Then: M13-005 — NOT_STARTED; concrete publication remains separate
Runtime upgrade maintenance: UPG-001..006 COMPLETE; PR34 merged/main CI320 ten SUCCESS
```

### 2026-10-29 B2-B Python selected-program backend 착수

[Baseline·exact 지도·두 번째 검토](./evidence/package-client-quarantine/p6b2b-python-artifact-backend-plan-20261029.md)는
기존 whole readonly input/CPU 원장과 shared captured program/Python irreversible handoff를
재사용하고 새 selected-console profile/backend gap만 구현한다. Source 전1356/commit5c853a1을
고정했고 canonical fixed controller·전체 argv/readonly binding·operational finalization과 program
nonzero 분리를 검토했다. 새 runtime/hosted/UVx equivalence·CLI/final은 미완료다. CI431 poll04는
07:27:10UTC8SUCCESS/Linux running이고 next07:47:10UTC/KST16:47:10을 등록했다. 새 push는 없다.

### 2026-10-29 B2-A 공유 captured program named local 완료

[Receipt](./evidence/package-client-quarantine/p6b2a-shared-program-capture-local-20261029.json)는 기존 Node
본문/mapper를 private helper 한 곳으로 옮긴 source1356/기존1354 불변·canonical9/Quick304.561s/
전체 sandbox·Bootstrap Race35.809s와 같은4f9d binary의 controlled12/195.49s·public CLI4/110.08s
PASS를 연결한다. 공개 결과·privacy·typed error/preflight/input 정리와 CPP131/권한·한도는
유지했다. Host/42/ZIP/Docker/source 보존/helperwait0/전체 정리다. Child body/held-pipe는
미입증이며 Python runtime/native UVx/aliases/CLI·새 shared-source hosted/final은 미완료다.
완료 사본4개109400066B는 원본 보존/root86proc/참조0·query errors0 뒤 삭제했다.
누적179개 약6.35GiB의 Linux 공간 회수이며 C: VHD 축소로 전용하지 않는다. CI431을
새 push로 취소하지 않고 같은 M13-004에서 B2-B baseline/정확한 missing 경계를 계속한다.

### 2026-10-29 B2-A 기존 captured program 본문 공유 착수

[Baseline·exact 지도·두 번째 검토](./evidence/package-client-quarantine/p6b2a-shared-program-capture-plan-20261029.md)는
기존 Node captured admission/terminal/target/source-verify 구현을 private helper로 옮겨 같은
Node caller가 사용하도록 한다. 공개 Node API/typed error/input lifecycle·CPP/권한/한도는
유지하고 Python fake Node layout 또는 별도 workflow 복제를 피한다. Source 전 runtime1355
감사를 기록했으며 새 source의 local 검증은 미완료다. CI431 자동 조회는06:47:07에 실제로
완료됐고 다음07:07:07을 등록했다. Completed8/444981B/checkoutf4d7/tree20b/CPP131은 부분 증거다.

### 2026-10-29 CI431 등록·실제20min 감시

[Registration](./evidence/package-client-quarantine/ci431-registration-20261029.json)은 source0d96/
tree20b74161/3032·1355의 정상 FF push와 원격 ref·실제run40826938443/start06:03:34UTC를
확인했다. Initial6SUCCESS/Linux·macOS·Quick running이며 timer01/session73616은06:23:34UTC
(한국15:23:34)에 실제 발화한다. 새 source를 push로 취소하지 않고 다음 조회 뒤20분을 갱신한다.
Python ARTIFACT runtime owner baseline을 독립 수행하며 새 hosted·user-run/UVx/CLI/final은 미완료다.

### 2026-10-29 B1 readonly4 hosted 연결 local 완료

[Receipt](./evidence/package-client-quarantine/p6b1-hosted-readonly-input-local-20261029.json)는 기존
current step/never-started image DATA owner에 readonly4 closed profile을 추가했다. Canonical7/
Quick257.065s/전체 checker Race54.527s·Bash DATA32가 PASS이며 source1355 membership와
기존1352/internal·cmd/CPP131·권한·한도/관찰 손실 guard는 유지한다. Final 후보 검증 뒤 승인된
정상 push/실제 새 run/20min timer를 등록한다. 새 hosted·user-runtime/UVx/CLI/final은 미완료다.

### 2026-10-29 CI430 Go local 진단·B1 hosted gap 착수

[Diagnostic](./evidence/package-client-quarantine/ci430-go-local-diagnostic-20261029.json)는 같은 Go
owner/CPP·기존 guard의 current1355 단일 grpc1.76.0 parent286.56s PASS와 전체 정리를 확인했다.
원래 CI의 완료 graph는 없으므로 같은 frozen graph 또는 관찰 손실 원인 해결로 쓰지 않는다.
Native write body의 exact upstream 동일성과 오류·retry 의미만 확인했고 config/guard는 유지한다.
[B1 hosted 지도·두 번째 검토](./evidence/package-client-quarantine/p6b1-hosted-readonly-input-plan-20261029.md)
뒤 기존 current cohort에 새 readonly4 gap만 연결한다. 새 의미 있는 source의 qualification을
위해 정상 push하며 timer를 다시 등록하고 완료된 불필요한 test 사본 정리도 이어간다.

### 2026-10-29 CI430 관찰 손실 실패 확인

[Failure record](./evidence/package-client-quarantine/ci430-observation-loss-failure-20261029.json)는 source3c85/
tree82c9/all10 checkouta84ae·decoded966355B readback와8SUCCESS/Linux+Required FAILURE를
연결한다. Linux56PASS/1FAIL/noSKIP의 첫 실패는 기존 grpc1.76.0 transitive Go CLI resolver의
STREAM_FAULT/DROPPED_COUNT다. 관찰 손실을 거부한 guard를 유지하며 손실 발생 조건은 미확정이다.
Helper9wait0/최종 original1·정리/복원을 확인했다. 후속 Node/current C7/Python·kernel 단계는
SKIPPED이며 통과로 쓰지 않는다. Timer03 실제 발화 뒤05:16:30UTC 결과를 소비·종료했다.
B1/source14487/1355와 whole final은 이 source 밖이며 동일 입력 blind rerun 없이 진단을 계속한다.

### 2026-10-29 완료 시험 산출물 정리

[Cleanup receipt](./evidence/package-client-quarantine/completed-test-artifact-cleanup-20261029.json)는
보존된 identity와 일치하는 완료 ELF 사본173개6.199GiB를 제거했다. Root85proc의 cwd/exe/FD/
maps/mount/argv 참조0·query errors0와 exact inode/owner/single-link/hash를 확인했고 최신
B1/A3 기준·CPP·고정 공급·source/raw logs/cache/SDK/Git을 보존했다. Linux /tmp 파일시스템의
공간 회수이며 C: 물리 여유 증가나 VHD 축소로 전용하지 않는다. 사용자 요청대로 작업을 계속한다.

### 2026-10-29 CI429 grouped Node/CLI hosted 완료

[Qualification](./evidence/package-client-quarantine/ci429-node-cli-hosted-qualification-20261029.json)은 source9eb/tree51a9/2996·1343과
all10 checkoutb2eb/Required10SUCCESS·decoded1,308,207B readback을 확인했다. Linux97/noFAIL/
noSKIP/CPP131·CPU300.35s/Node prep48.40s/controlled12 166.71s/application2 38.97s/CLI4
81.39s/kernel8 5.26s/helper16wait0/세 final0·복원이다. Child는 clone-startup 거부·정리만
입증했으며 body/held-pipe는 미입증이다. 전체100m21s이며 timer05 due03:37:40 뒤
03:37:45UTC 결과 소비·종료했다. Later C7/P6/whole final은 이 source 밖이며 계속 진행한다.

### 2026-10-29 P6-B1 named local readonly DATA 완료

[Receipt](./evidence/package-client-quarantine/p6b1-python-readonly-input-local-20261029.json)는 canonical10과
같은1355source/c380binary의 Python readonly4 PASS73.90s·별도 기존 pipx PASS39.40s를 연결한다.
원본·Host/고정42/ZIP/Docker·CPP 보존과 helperwait0/전체 정리를 확인했다. 첫 normal-stop 취소
오인은 한정 교정됐고 이전 pipx CPU guard 실패의 원인은 미확정으로 유지한다. Whole v2 attempt
PASS나 user-run/UVx/CLI 성공으로 바꾸지 않는다. CI430은 B1 source를 포함하지 않으며
현재20분 감시를 유지한다. 다음은 기존 owner를 재사용한 Python ARTIFACT runtime/admission 감사다.

### 2026-10-29 P6-B1 readonly Python inputs 착수

[Exact 지도·두 번째 검토](./evidence/package-client-quarantine/p6b1-python-readonly-input-plan-20261029.md)는
기존 M5 preparation/RO anchor 본문과 actual CPU/observer/cleanup을 공유하고 current
whole context/console source의 DATA 검증 gap만 구현한다. Required unit/Policy·CPP/user-run
완료 의미는 유지하며 source 전 baseline을 기록했다. CI430 timer01을 우선 소비한다.

### 2026-10-29 P6-B Python runtime 기존 owner 감사

[Baseline](./evidence/package-client-quarantine/p6b-python-runtime-baseline-20261029.md)는 이미 구현된
whole closure manifest/실제 consumed stream 인증/full installed census/RO anchor·CPU·cleanup을
확인했다. 실제 source가 허용하는 zero-unit accounting은 준비 DATA만의 경계이며 fake inspection
또는 user-run completion으로 쓰지 않는다. 다음 B1 exact 지도 뒤 current readonly input gap만
진행한다. Production source/CPP 변경 없이 CI430의20min timer를 유지한다.

### 2026-10-29 grouped CI430 등록

[등록](./evidence/package-client-quarantine/ci430-registration-20261029.json)은 검증된3c85d82/
tree82c911ab/3019·1351의 정상 FF push·원격 ref와 actual run40816740622/04:15:37UTC를
확인했다. Initial DocsSUCCESS/8running이며 timer01/session1333은04:35:37UTC(한국13:35:37)
due다. Source를 새 push로 취소하지 않고 실제20min 결과를 확인하며 독립 Python runtime
기존 owner 감사를 계속한다. Hosted/final/Python user-run·UVx/CLI는 미완료다.

### 2026-10-29 P6-A4 current hosted wiring local 완료

[Receipt](./evidence/package-client-quarantine/p6a4-grouped-hosted-local-20261029.json)는 기존 image DATA helper/current
Python profile·protected lifecycle과 C7 refusal2/A3 preparation3의 unconditional 연결이다.
Canonical7/Quick299.996s/전체 checker Race55.890s·Bash syntax와 DATA21cases가 PASS다.
기존 production Go/CPP·cohort/권한/한도는 유지했고 remote PASS나 새 실제 runtime을 만들지
않았다. 최종 docs/security 뒤 승인된 정상 grouped push와 실제20min timer를 등록한다.

### 2026-10-29 P6-A4 grouped current execution hosted 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p6a4-grouped-hosted-plan-20261029.md)는
실제 local qualified Python 준비3/Node refusal2를 기존 helper/고정 fixture/profile로
hosted 연결한다. Production Go/CPP와 기존 Node/kernel/Required·권한/한도는 유지한다.
로컬 wiring/guard 검증 뒤 승인된 정상 push·실제20min timer를 등록하고 후속 runtime 감사를 계속한다.

### 2026-10-29 P6-A3 current Python preparation actual local 완료

[Receipt](./evidence/package-client-quarantine/p6a3-python-preparation-local-20261029.json)는 canonical9/Quick249.346s/Bootstrap
전체 Race23.628s/tagged build와 public cowsay actual3/84.364s를 연결한다. 기존 generic
whole 검사·Policy/records/staging을 재사용해 console 준비2·missing 거부1/fresh Evidence3/
Manifest2를 확인했다. Metadata member는 console declaration 출처이며 user-run 성공이 아니다.
기존1348·CPP/Host/42/ZIP/Docker 불변/helperwait0/workspaces/endpoints0다. Native tool-install
identity는 유지하고 UVx equivalence/Python runtime/CLI·전체 final은 OPEN이다.

### 2026-10-29 P6-A3 current Python preparation 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p6a3-python-preparation-plan-20261029.md)는
기존 native tool-install identity와 current whole PyPI 검사·Policy, A2 console DATA를
generic 준비/보관 서비스에 연결한다. 실제 module user-run/native UVx equivalence·CLI는
별도 후속이며 CI429/source9eb/timer05를 유지한다.

### 2026-10-29 P6-A2 reader와 replay 교정 local 완료

[Receipt](./evidence/package-client-quarantine/p6a2-controlled-reader-local-20261029.json)는 held intake/Run root·한정된 owned
snapshot·실제 canonical parser와 symlink/변조/취소 거부를 연결한다. A1 지문의 map 순서
오류를 실제 RED3.013s 뒤 private 정렬만으로 교정했고 Quick287.612s/전체 두 PyPI Race
14.051s와 canonical6가 PASS다. 기존 parser/Model C/CPP/권한·한도는 유지한다.
Python preparation/runtime/CLI·native equivalence는 미완료이며 CI429/source9eb는 보존한다.

### 2026-10-29 P6-A2 controlled intake reader 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p6a2-controlled-reader-plan-20261029.md)는
기존 handle routing과 A1 parser를 재사용하고 held intake root/한정된 owned snapshot을 연결한다.
Python runtime/CLI/native selection은 별도이며 CI429/source9eb를 보존한다.

### 2026-10-29 P6-A1 exact console DATA local 완료

[Receipt](./evidence/package-client-quarantine/p6a1-console-entry-local-20261029.json)는 기존1344 불변/new2의 focused·
Quick309.386s/전체 PyPI Race11.740s와 canonical checks를 연결한다. 실제 canonical wheel
reader에서만 opaque inventory를 만들고 command/module/attr·원본 metadata/whole inspection/
acquisition을 결합하며 multi/alias/extras/invalid/source/RECORD 치환 거부를 확인했다.
현재 DATA API만 local qualified이고 Python preparation/runtime/CLI는 WIRED=NO다.
CI429는 poll03 8SUCCESS/Linux 진행이며 timer04 due03:17:34UTC/session48576로 유지한다.

### 2026-10-29 P6-A1 console entry DATA 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p6a1-console-entry-plan-20261029.md)는
실제 canonical wheel reader에서만 생성한 opaque inventory와 exact console selection gap을
새 DATA API로 추가한다. 기존 parser/Model C/context/workflow/records/runtime/CPP는 유지한다.
명령·module/attr·원본 metadata member와 전체 inventory/acquisition을 결합하며 selection은
Policy나 실행 권한이 아니다. Actual Python/user-run/native selector/CLI는 후속이고
CI429/source9eb/timer03 due02:57:27UTC/session39633를 보존한다.

### 2026-10-29 P6 existing owners/pinned UV source 착수 감사

[Baseline](./evidence/package-client-quarantine/p6-execution-adapter-baseline-20261029.md)는
Core context/Application/records/capture의 기존 구현을 그대로 재사용하고 actual native selection/
Python console binding/readonly runtime/CLI gap을 구분한다. [고정 UV source discovery](./evidence/package-client-quarantine/p6-uv0134-source-discovery-20261029.json)는
원본 commit7456b60b의5file Git blob/SHA256/readback을 확인했다. Tool install과 tool run의
의미 동일성은 아직 UNPROVEN이며 cache/PATH/env/stdin 기본값을 HAA 권한으로 수입하지 않는다.
새 HAA source/placeholder/패키지·설정·CI job·버전/권한 변화는 없고 CI429는 계속 감시한다.

### 2026-10-29 C7 real post-Policy CLI refusal local 완료

[Receipt](./evidence/package-client-quarantine/p5c7-cli-refusal-local-20261029.json)는 existing1343 불변/test-only1의
canonical6와 actual refusal2/33.886s를 연결한다. 실제 public TypeScript의 fresh Evidence2/
원본 SHA/current Policy ALLOW 뒤 ambiguous/missing bin을 거부하고 준비·실행·exit·출력·
manifest 부재를 확인했다. Production/CPP/권한·예산은 그대로이며 Host/source/42/ZIP/Docker
보존과 helperwait0/resources/endpoints0다. 새 테스트는 CI429 source9eb에 없고 active run을
취소하지 않는다. Next는 local commit 뒤 bounded acceptance/P6 owner baseline이다.

### 2026-10-29 C7 post-Policy CLI refusal 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p5c7-cli-refusal-plan-20261029.md)는
기존 entry 선택 거부/actual검사/CLI를 유지하고 real Policy ALLOW 뒤 ambiguous/missing bin의
보관·실행 NOT_STARTED 두 actual gap을 새 test parent 하나로 검증한다. Production/CPP/권한·
예산은 유지하며 fake approval/target/result는 없다. CI429 source9eb854f와 timer01
due02:17:02UTC/session84875를 보존하고 새 push하지 않는다.

### 2026-10-29 current grouped CI429 등록

[등록](./evidence/package-client-quarantine/ci429-registration-20261029.json)은 검증된9eb854f/tree51a9a25c/2996·1343의 normal FF push와 remote ref를 확인했다.
CI429/run40803448693은01:57:02UTC 시작/initial9running이며 준비·kernel과 current
boundary12/workflow2/CLI4를 포함한다. 실제20min timer01/session84875는
02:17:02UTC(한국11:17:02)가 due다. 새 hosted PASS로 표시하지 않고 실행 중 push를
보류한다. 같은 M13-004의 독립 P5 bounded acceptance/P6 owner baseline을 계속한다.

### 2026-10-29 C6 hosted wiring local 완료

[Local receipt](./evidence/package-client-quarantine/p5c6-grouped-hosted-wiring-local-20261029.json)는 기존 Node hosted step에
boundary12/workflow2/CLI4를 개별5min command와 noSKIP/unique cases/original exit guard로
연결한 canonical3·Bash syntax·guard24/primary6 결과다. Go/CPP/runtime1343과 기존
preparation/kernel/Required·권한·예산은 불변이다. 로그 guard는 DATA 시험이며 actual
qualification을 새로 만들지 않는다. 다음 승인된 정상 grouped push/run/timer 등록 전
최종 문서·보안을 확인하고 hosted/whole final acceptance는 OPEN으로 유지한다.

### 2026-10-29 C6 grouped hosted wiring 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p5c6-grouped-hosted-plan-20261029.md)는
기존 unconditional Node hosted 단계에 현재 boundary12/workflow2/CLI4를 개별5min invocation으로
연결한다. Go/CPP/권한·예산/Required와 기존 preparation/kernel 소비자는 유지한다.
Canonical workflow·Bash/로그 guard를 확인한 의미 있는 후보만 승인된 정상 push로 전달하고
시작 직후20min 타이머를 등록한다. Hosted/whole P5/final은 결과 전까지 OPEN이다.

### 2026-10-29 CI428 source558 backend hosted 완료

[Final qualification](./evidence/package-client-quarantine/ci428-node-backend-hosted-qualification-20261029.json)은 Required10SUCCESS와
all10 same checkoutd965d5bf/treef5e4d9f9/2969·1330 및 decoded1,278,395B readback을
연결한다. Linux94/noFAIL/noSKIP/CPP131·CPU296.32s/current public Node fresh2 48.93s의
정상0·typed guest1과 whole readonly/current Policy·kernel8cases5.41s/helper16wait0/
세 final original0·복원을 확인했다. 전체94m51s이며 timer05는01:42:43UTC 결과 소비 후
종료했다. Later local C3d/C4/C5는 이 source에 없고 grouped hosted가 다음이며
과거 native deadline 직접 원인·whole final/P5-P7/signed는 여전히 OPEN이다.

### 2026-10-29 C5 npx/npm exec actual CLI local 완료

[Receipt](./evidence/package-client-quarantine/p5c5-npx-cli-local-20261029.json)는 canonical9/전체 영향 Race와
actual public CLI4/113.529s를 연결한다. Program --help·literal argv는 실제 entry로
전달됐고 두 alias·human/JSON·en/ko·private/explicit output·정상0/typed guest1 및 fresh
records4가 PASS다. Backend/CPP/권한·한도는 유지하며 Host/source/42/ZIP/Docker 보존과
helperwait0/resources/endpoints0다. 마지막 stale1332 producer assertion은9checks PASS 뒤
기록 오류이며 독립 current1343 validator0로 확인하고 제품 검사를 반복하지 않았다.
다음 grouped hosted wiring은 먼저 baseline을 기록하며 CI428 진행 중 push하지 않는다.

### 2026-10-29 C5 safe output/npx CLI 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p5c5-npx-cli-plan-20261029.md)는
기존 C4/current backend를 재사용하고 bounded npx/npm exec grammar와 분리된 안전한
출력·실제 CLI 연결 gap을 구현한다. 기본 output은 bytes count이며 명시 show-output은
ASCII escaped untrusted data다. Parsed HAA help와 program --help를 구분하고 input bounds를
I/O 전에 검사한다. CI428 source558d75a는 timer05 due01:42:24UTC로 보존한다.

### 2026-10-29 C4 application workflow actual local 완료

[Local receipt](./evidence/package-client-quarantine/p5c4-execution-workflow-local-20261029.json)는 canonical9/전체 영향 Race와
actual public TypeScript2/58.861s를 연결한다. 기존 준비·materialize·Node backend를
Application interface와 concrete Bootstrap adapter로 연결했고 fresh current 검사/Policy/records/
private argv 및 정상0/비정상1 원래 cause를 확인했다. 결과의 terminal/output은 복사·비공개이며
준비 실패·모순 terminal·late caller 취소는 완료로 표시하지 않는다. 기존1332/CPP/old actual
parents·권한/예산은 그대로이고 Host/source/42/ZIP/Docker·helperwait0/resources/endpoints0다.
다음은 안전한 출력과 public CLI이며 CI428 진행 중 새 push는 하지 않는다.

### 2026-10-29 C4 application workflow 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p5c4-execution-workflow-plan-20261029.md)는
기존 준비/검사·Policy/materialize/Node backend를 재사용하고 application의 동일 요청 연결과
private 결과·원래 오류 유지 gap만 구현한다. 기존 actual parent와 controlled12는 유지하고
새 public2 parent로 실제 연결을 검증한다. CLI/renderer/final은 후속이며 CI428은 보존한다.

### 2026-10-29 C3d3 same-source12 actual local 완료

[Local receipt](./evidence/package-client-quarantine/p5c3d3-node-deadline-readonly-local-20261029.json)는 actual202.496s/12cases와
canonical/tagged6checks를 연결한다. Caller30s deadline은 body ready 뒤 원래 cause/incomplete,
readonly self-write는 kernel 거부를 처리한 program0과 원본 전체 bytes 보존이다. 기존10과
source/Host/42/ZIP/Docker 보존·helperwait0/resources/endpoints0/noSKIP도 PASS다.
Production/CPP/권한·예산은 그대로이며 public application/CLI/final은 다음이다. CI428은
01:02:15UTC 8SUCCESS/Linux 진행이고 timer04 due01:22:15UTC/session34757로 유지한다.

### 2026-10-29 C3d3 actual deadline/readonly self-write 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p5c3d3-node-deadline-readonly-plan-20261029.md)는
local951f73e/기존1332의 current runtime/검사/Policy를 재사용하고 actual ordinary caller
WithTimeout 뒤 body DeadlineExceeded와 readonly 자기 파일 쓰기 거부 두 gap만 추가한다.
기존10까지 same-source12cases로 검증하며 production/CPP/45s/stream1MiB/caller4min은
유지한다. 쓰기 거부를 처리한 program completion과 파일 쓰기 성공을 구분한다. CI428은
source558d75a/timer03 due00:59:39UTC로 유지하고 새 push하지 않는다.

### 2026-10-29 C3d2 current10 hostile actual local 완료

[Local receipt](./evidence/package-client-quarantine/p5c3d2-node-hostile-boundary-local-20261029.json)의
same1332/production·CPP 불변·canonical/tagged6checks와 actual154.558s/10cases가 PASS다.
Outside path/network의 실제 fact와 comm spoof의 kernel NODE/ARTIFACT를 확인했고 child는
clone fault/ARTIFACT/incomplete/cleanup이다. Child WaitDelay=false·JS 시작 미입증을 명시했다.
V1 role 관측 한계와 v2의 actual0 후 tuple/list operator 오류를 각각 보존하고 독립 readback
검증으로 기록만 바로잡았으며 actual을 재실행하지 않았다. Source/Host/42/ZIP/Docker 보존·
helperwait0/resources/endpoints0/noSKIP다. Next C3d3은 body 뒤 actual deadline/readonly self-write이며
먼저 local unit commit만 보존한다. CI428/source558d75a/timer03 due00:59:39UTC는 유지한다.

### 2026-10-29 C3d2 actual hostile 경계 착수

[C3d2 baseline·지도·두 번째 검토](./evidence/package-client-quarantine/p5c3d2-node-hostile-boundary-plan-20261029.md)는
local a6e7ee6의 기존1332·current검사/Policy/runtime을 재사용해 outside path/guest-loopback
network/Artifact child/comm spoof 네 actual gap만 잡았다. 두 test만 수정하고 fixture bytes가
바뀌므로 기존6case까지 same source10case로 검증한다. Production/CPP/일반FS·network·role/
한도는 유지하며 child startup 거부를 성공한 child·pipe drain 증거로 표시하지 않는다.
CI428은558d75a/8SUCCESS/Linux 진행과 timer02 due00:38:26UTC를 유지하고 새 push하지 않는다.

### 2026-10-29 C3d1 selected Node 신호·취소·독립 overflow actual 완료

[Local receipt](./evidence/package-client-quarantine/p5c3d1-node-boundary-local-20261029.json)의
1332inputs/newtests2·기존1330/CPP 불변·canonical6checks와 actual92.871s/6fresh cases가 PASS다.
실제 current verifier/static/dynamic/Evidence/Policy 뒤 literal argv·kernel TERM15/KILL9,
body 준비 뒤 custom cause 취소/incomplete와 stream별1MiB overflow/incomplete·원래 cause를
확인했다. Host/source/42/ZIP/Docker 보존·helperwait0/owned resources/endpoints0/noSKIP다.
공급망/native solver는 controlled fixture/NOT_RUN이며 production/CPP/한도 변경은 없다.
현재 CI428에는 이 test source가 없고 새 push 없이 한글 local commit을 보존한 뒤 같은
M13-004 C3d2의 FS/network/child hostile baseline을 계속한다. CLI/final/wholeP5는 OPEN이다.

### 2026-10-29 C3d1 actual Node failure 경계 착수

[Baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p5c3d1-node-boundary-qualification-plan-20261029.md)는
source558d75a/1330의 기존 backend/Host capture/kernel/cleanup·실제 검사/Policy를 재사용하고
selected Node TERM/KILL·중간 caller cause 취소·stdout/stderr overflow·literal argv의6fresh
actual만 missing으로 잡았다. 두 Linux realcorpus test 파일만 추가하며 production/CPP/권한/
예산/기존 parent는 유지한다. Local controlled registry fixture는 공급망/native solver 성공이
아니며 actual current integrity/static/dynamic/Evidence·Policy와 runtime을 구분한다. CI428
source와 timer는 그대로이며 새 push 없이 같은 단일 M13-004에서 계속한다.

### 2026-10-29 승인된 Artifact Node foundation CI428 등록

[CI428 등록](./evidence/package-client-quarantine/ci428-registration-20261029.json)은 normal/nonzero
backend·CPP131/실제2requests·기존8roots/3consumers와 final docs/security를 확인한21경로의
한글 commit558d75a/treef5e4d9f9/2969·1330입력을 기존 milestone branch에 정상 FF push한 결과다.
Remote ref/source를 대조했고23:57:29UTC 시작 뒤 first20min timer01/session55237의 due는
2026-10-29 00:17:29UTC(한국09:17:29)다. 초기 Docs SUCCESS/나머지 진행이며 whole hosted는
PENDING이다. 실행 중 새 push로 취소하지 않는다. 같은 M13-004의 독립 C3d actual failure/
악성 qualification을 baseline audit부터 이어가며 그 이후 source와 public CLI/final은 밖이다.

### 2026-10-29 실제 Artifact Node normal/nonzero backend local 완료

[최종 local receipt](./evidence/package-client-quarantine/p5c3c2b-artifact-node-local-20261029.json)는
current1330/Go-v6 checks8·CPP131/canonical289.119s와 actual69.401s/fresh2의 guest0·guest1,
기존8roots9.678s/세 consumers12.050s PASS를 연결한다. Current whole Policy·readonly inputs,
accepted target/external kernel terminal/independent bounded streams와 complete trace/disposal,
Source/Host/42/ZIP/Docker 보존·helperwait0/owned resources/endpoints0을 확인했다. 앞 cwd/
runtime ABI 실패 원본을 보존했고 일반 FS/CONTROL/network/한도는 유지했다. Qualification은
이 named normal/nonzero foundation뿐이다. 다음은 같은 M13-004의 C3d 실제 취소/overflow/
child·악성 경계이며 먼저 current unit을 검토/중간 commit/중요 grouped CI로 전달한다.
New hosted/public application·CLI/wholeP5와 final profiles/signed는 OPEN이다.

### 2026-10-29 CI427 공통 후보 hosted qualification 완료

[최종 qualification](./evidence/package-client-quarantine/ci427-kernel-readonly-hosted-qualification-20261029.json)은
sourcee0dc/treefff27985/2959·1324입력과 all10checkout0e703657 일치·Required10SUCCESS다.
Timer04는23:18:00UTC 발화/23:18:15UTC query로 완료를 소비했고 final update까지78m45s다.
All10 decoded1,229,906B/chunk readback과 Linux94/noFAIL/noSKIP/CPP130·CPU238.05s,
production readonly fresh2 34.71s/kernel8cases5.62s·helper16wait0/세 cleanup original0/복원을
확인했다. 추가 timer는 없으며 이후 Node source1330/profile/backend·CPP131와 user-run/CLI/
wholeP4/P5-P7/final4profiles/signed는 이 run 밖이다. 같은 M13-004의 Node actual startup
self-cgroup read gap을 계속한다. Cwd 교정 후보는 reader 시작을 통과했지만 전체 Node 실행은
실패로 유지하며 새 CPP build/실제 검증 전 qualification으로 표시하지 않는다.

### 2026-10-29 Artifact Node 첫 actual 실패와 제한 진단

[현재 수정 지도](./evidence/package-client-quarantine/p5c3c2b-node-execution-plan-20261029.md)의
새 Node CPP131/canonical은 PASS지만 actual-v1은33.158s·readonly reader start 단계에서
실패했다. Artifact 실행은 미도달이며 첫 원본/Host·입력 보존·정리를 별도로 확인했다.
Default runtime DATA 세 cwd 조합 PASS를 gVisor 원인으로 일반화하지 않는다. 기존
ExitError.Stderr를 고정 공개 integration fixture에서만 제한 수집하는 test-only 변경으로
상세 원인을 확인한다. Node actual/전체 qualification은 NO이며 CI427 source는 불변이다.

### 2026-10-29 승인된 공통4commit CI427 등록

[CI427](./evidence/package-client-quarantine/ci427-registration-20261029.json)은 sourcee0dc39a/tree
fff27985/2959·1324입력의 누적4commit을 기존 milestone branch에 정상 FF push한 결과다.
Kernel target/terminal8roots와 clean readonly production fresh2·canonical/CPP130/whole 후보
일치를 확인한 구체 후보이며 main merge/tag/release는 없다. 21:56:16UTC 등록 뒤 실제20min
timer01/session20366의 첫 due22:16:16UTC(한국07:16:16)를 설정했다. New hosted는 PENDING이고
실행 중 source를 새 push로 취소하지 않는다. 같은 M13-004의 독립 Node mode/profile/lifecycle
감사를 계속하며 이후 소스와 user-run/CLI/final qualification을 이 run에 전용하지 않는다.

### 2026-10-29 기존 Linux CI kernel parent 연결 로컬 검증

[Wiring receipt](./evidence/package-client-quarantine/p5c3c-hosted-kernel-wiring-local-20261029.json)의
canonical7/checker fullRace47.748s·workflow9 negatives/Bash7guard와 current source1324/Go3575dbab/
CPP0c09 readonly production fresh2 actual56.226s가 PASS다. 기존8root/제품/CPP는 재구현하지
않고 기존 job/태그 빌드·protected client/helper를 재사용했다. Source/Host/42/ZIP/Docker·정리와
최초 producer hash guard 중단/교정은 개별 원본에 보존한다. 새 hosted는 아직 PENDING이며
완료한 C3c 단위를 한 번 push해 새run/20min timer를 등록한 뒤 같은 item의 Node profile을 계속한다.

### 2026-10-29 CI426 종료와 target-image dependency 검증

[CI426](./evidence/package-client-quarantine/ci426-offline-materialization-hosted-qualification-20261029.json)은
source4db/tree e8ad4e1d/2940·all10checkoutc106c554/decoded1,274,315B 일치와 Required10SUCCESS다.
실제 timer05 21:33:21UTC 발화 뒤21:33:27UTC에 완료를 소비했다. 전체95m30/Linux93/noSKIP/
CPP128/CPU303.82s/helper15wait0/final0와 production31.97s/fresh2·offline materialization이 PASS다.
추가 timer는 없으며 CI425 native 원인 미확정과 later C3c/new user-run/final 미완료는 유지한다.
[새 target fact local](./evidence/package-client-quarantine/p5c3c2b-target-image-local-20261029.json)은
CPP130/8root9.827s·기존3소비자12.296s/Host·입력 보존·정리를 확인했다. 같은 root의 helper
종료와 실제 accepted target load를 구분하며 Node user-run 성공으로 확대하지 않는다.
이8root parent를 기존 Linux CI에 연결한 뒤 누적된 공통 변경을 한 번 push하고 같은 item을 계속한다.

### 2026-10-29 P5-C3c2a 읽기 전용 whole-tree 입력 로컬 검증

[결과](./evidence/package-client-quarantine/p5c3c2a-readonly-node-inputs-local-20261029.json)는
source1320/Go31836d82/unchanged CPP4df의 actual54.439s·fresh2 PASS다. Current whole 검사/
Policy와 offline materialization 뒤 clean readonly tree 전체 metadata/bytes를 검증했다.
Canonical7/영향3package freshRace30.352s와 Source/Host/42/ZIP/Docker 보존·helperwait0/final0/
workspace 부재가 대응한다. 첫 actual 실패와 별도 DATA lifetime probe를 보존하고 기존 Python
owner의 reader 선연결→writer 종료→sole RO 순서를 재사용했다. Artifact user program은
미실행이며 같은 M13-004의 C3c2b admission/image-start/terminal/output/cleanup을 계속한다.
CI426은 이전4db source로 완료됐고 최종 결과는 상단 receipt를 따른다.

### 2026-10-29 P5-C3c1 외부 kernel root terminal 로컬 검증

[결과](./evidence/package-client-quarantine/p5c3c1-kernel-terminal-local-20261029.json)는 source1316/
Go b5166b73/CPP4df80d98의8fresh trusted controller roots9.014s와 기존 npm/Python/ELF3parents
11.163s PASS다. Kernel의 exit0/37/125/126/127/255·TERM/KILL과 별도3B stdout/stderr를 대조했다.
CPP129/canonical checks·원본과 Source/Host/42/ZIP/Docker 보존·helperwait0/final0정리가 대응한다.
Host client exit/stdout marker를 guest signal이나 coverage로 쓰지 않는다. Root 종료만으로 actual
Artifact Node image 로드·출력 완전성·descendant drain을 증명하지 않으며 C3c2가 다음이다.
일반 role/FS/network/예산은 유지했고 new hosted/whole common/CLI/final은 OPEN이다. 실행 중인
CI426은 이전 source4db0111이며 최신 타이머는 이 Queue 상단과 registration을 따른다.


### 2026-10-29 P5-C3b offline whole-set materialization 로컬 검증

[최종 receipt](./evidence/package-client-quarantine/p5c3b-offline-node-materialization-local-20261029.json)는
source1313/binary0c4bfe0e·기존 CPP128의 actual45.343s/parent43.57s/fresh2 PASS다. Current
whole checks·Policy ALLOW/storage-only 뒤 fixed no-script/offline materialization의 모든 파일과
entry를 strict C2 binder로 검증했다. 같은 layout digest와 Source/CPP/42/ZIP/Host/Docker 보존,
helperwait0/final0/Node workspace 부재를 확인했다. Canonical Quick77.668s/영향3package
fullRace29.883s/기타5checks와 개별 원본이 대응한다. 실제 artifact user program은 실행하지
않았으며 CLI/guest exit-output/final4profiles·signed/전체P5는 OPEN이다. 앞 v1-v4는 실패로
보존하며 일반 FS/role/CPU 예산을 바꾸지 않았다. 다음 C3c owner audit와 의미 있는 grouped
hosted/native Required qualification을 계속한다.


### 2026-10-29 CI425 완료 소비 — Linux preparation PASS / macOS·Required FAIL

[최종 결과](./evidence/package-client-quarantine/ci425-production-preparation-hosted-result-20261029.json)는
source418a/tree016df83d/2926·all10checkout3cc28f9e/decoded1,346,740B의 동일성을 확인했다.
실제 timer05 발화19:32:09UTC 뒤19:32:16UTC에8SUCCESS/macOS+Required FAILURE를 직접
소비했다. Final update까지89m03이며 Linux93parents/noSKIP/noFAIL/CPP128/CPU277.18s,
helper15wait0/final original0·복원과 새 production preparation19.28s/fresh2/current whole checks·
Policy ALLOW/storage-only가 PASS다. Later C3a/C3b·user program/CLI/final4profiles·signed는 밖이다.
Native default6m 직접 원인은 기존 observation대로 OPEN이며 Required 실패를 성공으로 바꾸지
않는다. Timer05는 종료됐고 새 timer/재실행은 없다. 같은 M13-004의 독립 C3b 진단을 계속하며
다음 의미 있는 grouped source에 native/Required qualification을 연결한다.


### 2026-10-29 P5-B2f production preparation 실제 local 검증

[B2f receipt](./evidence/package-client-quarantine/p5b2f-production-preparation-local-20261029.json)의
no-overlay30.930s/parent28.93s는 실제 factory/M3·exact TypeScript5.9.3/source SHA·required3/
Evidence4/Policy/fresh2회·staging PASS다. Source1301/CPP/원래42/ZIP·Host/Docker 불변과
helperwait0·Node workspace 부재를 확인했다. User program은 실행하지 않았다. 기존 Linux
job에 tagged parent와 exactPASS1/noSKIP guard를 연결했고 workflow·post-wiring Quick81.733s/
Bash8case가 PASS다. 새 hosted step/actual user-run/CLI는 별도다. CI424 두 번째 실제 조회
16:40:43UTC는8SUCCESS/Linux 진행이며 다음17:00:43UTC/session2975를 계속 소비한다.

### 2026-10-29 P5-B2e preparation 로컬 검증

[B2e receipt](./evidence/package-client-quarantine/p5b2e-execution-preparation-local-20261029.json)의
Quick229.702s/영향5package fullRace39.718s/docs/security가 PASS다.
Target 없는 resolver→fresh 검사→actual Node proof→current context→records→staging을 연결했다.
보관 handoff actual은 fixed DATA/M1 fixture이며 production dynamic/user program은 미수행이다.
Private request의 argv copy·format/JSON 보호와 cached entry/records·wrong staging/command·
최초 provider/취소 cause를 확인했다. 준비 내부 WIRED=YES, actual backend/CLI는 아직NO다.
[CI424 registration](./evidence/package-client-quarantine/ci424-registration-20261029.json)의 첫 timer
소비16:20:03UTC는8SUCCESS/Linux 진행·all8checkout/tree동일이다. 다음16:40:03UTC/session40231.

### 2026-10-29 사용자 연속 CI/push 권한

사용자는 취침 중 중요한 CI test와 milestone branch push를 다시 묻지 않고 진행하며 실제
20min timer 조회를 계속하도록 명시했다. 이 승인 이후 필요한 grouped branch push/기존 CI는
구체 후보 검토·canonical 검증 후 자율 진행한다. 실행 중인 CI424 source4ec를 새 push로 취소하지
않고 독립 B2e 작업을 계속한다. Main reviewer merge·milestone acceptance 규칙은 유지한다.

### 2026-10-29 승인된6commit/source4ec CI424 등록

사용자 '진행해' 승인으로6dd3136 다음부터4ecf37b까지6commit/26file만 기존 공개 milestone
branch에 정상 FF push했다. Whole candidate2909files/tree d88cd820/Root·기존 private의
index+disk blob/mode와 runtime1295를 재확인했다. Main 병합/tag/release와 후속 tmp B2e 변경은
payload 밖이다. [CI424](https://github.com/rahoney/heliopause/actions/runs/40721538953)는
15:59:44UTC 등록됐다. Initial Docs/corpus SUCCESS·나머지 진행이며 Required/whole result는
미확정이다. 실제 timer6768의 첫 due16:19:44UTC에 fresh run/jobs를 조회하고 계속이면
다음20min을 등록한다. 독립 B2e 작업은 별도 clone에서 계속하며 이 source를 취소하지 않는다.

### 2026-10-29 P5-B2d controlled Node entry bytes 로컬 검증

[B2d receipt](./evidence/package-client-quarantine/p5b2d-controlled-node-entry-read-local-20261029.json)의
Quick279.401s/inspection npm fullRace17.241s/docs/security가 PASS다.
기존 capture의 optional writer와 새 reader/회귀만 추가했다. Original 전체 source/inventory·
selected bytes를 결합하며 치환/원본 변조/link/압축/다른 interpreter/cancel은 proof를 주지 않는다.
Actual flow WIRED=NO이며 다음은 resolver/fresh inspection/context/records/staging 조립이다.

### 2026-10-29 P5-B2c fresh whole inspection 로컬 검증

[B2c receipt](./evidence/package-client-quarantine/p5b2c-fresh-execution-inspection-local-20261029.json)의
Quick40.217s/application fullRace4.233s/docs/security가 PASS다. Existing install/project per-entry·
batch helper와 Policy를 그대로 재사용한 새2file이다. 반복 resolution도 fresh whole 검사를
수행하며 누락/치환·entry/set 거부·부분 실패/store 원인·ID/취소/derived 거부를 확인했다.
Actual command flow는 아직 WIRED=NO이며 다음은 기존 archive capture의 selected bytes와
실제 resolver/inspection/Node/context/records/staging 조립이다. 단일 M13-004를 계속한다.

### 2026-10-29 CI423 current ordinary hosted qualification

[CI423 qualification](./evidence/package-client-quarantine/ci423-pipx-current-hosted-qualification-20261029.json)은
같은6dd/tree feab8ef9/2891·10checkout과 Required10SUCCESS/Linux92/noSKIP/CPP128/CPU251.33s,
UID1001·pipx current named parents/helper14wait0/final0을 확인한다. Decoded1,221,192B 전체를
매 chunk readback/SHA로 보존했다. 네 번째 실제20min 조회15:14:14UTC에서 완료를 소비했고
추가 timer는 없다. C-e4 ordinary named scope만 닫으며 final4profiles·signed와 후속 P5 source는
이 결과 밖이다. 다음은 같은 단일 M13-004의 P5 current whole 검사/application이다.

### 2026-10-29 P5-B2b Node composition / execution records 로컬 검증

[B2b receipt](./evidence/package-client-quarantine/p5b2b-node-composition-records-local-20261029.json)의
Quick220.145s/bootstrap·evidence fullRace26.899s/docs/security가 PASS다. 기존1287source는
불변이며 새4file은 exact Node pin 조립과 target 없는 records만 담당한다. Raw argv/hash는
저장하지 않고 whole Evidence/Policy/SBOM binding을 유지한다. Actual flow WIRED=NO이며
다음은 fresh whole inspection/application과 staging/backend/CLI다. 같은 단일 item을 계속한다.
CI423의 마지막 timer는 실제15:14:14UTC Required10SUCCESS로 소비했고 추가 timer는 없다.

### 2026-10-28 P5-B2a metadata context 로컬 검증

[B2a receipt](./evidence/package-client-quarantine/p5b2a-execution-context-local-20261028.json)의
Quick288.508s/domain fullRace2.620s/docs/security가 PASS다. 기존1285source와 VerifiedSet/
Resolution을 유지하고 새2file의 complete graph/primary acquisition/member/runtime metadata·
private argv와 source/Policy identity를 검증했다. 실제 P5 flow는 WIRED=NO이며 다음은 Node proof와
canonical runtime의 composition 및 current 검사/Manifest·run 연결이다. CI423은 third actual
20min query14:53:43UTC에도8SUCCESS/Linux 진행이며 next15:13:43UTC/session51968이다.

### 2026-10-28 P5-B2 기존 owner 재사용 감사

[B2 actual owner 감사](./evidence/package-client-quarantine/p5-execution-foundation-baseline-20261028.md)는
VerifiedSet/VerifiedBundle/LocalStaging이 generic이고 InstallContext를 요구하지 않음을 실제
source에서 확인했다. Existing inspection helpers·Manifest entry/SBOM builders를 재사용하고
non-install context/entry·argv/runtime/source-policy 동결의 missing만 수행한다. Source 전
계약·문법/bounds와 두 번째 검토는 진행 중이며 B2 구현/실제 run 연결은 아직 미완료다.
CI423 source6dd와 실제 다음 due14:33:07UTC/session84800은 그대로다.

### 2026-10-28 CI423 첫20min 소비와 P5-B1 로컬 검증

14:12:50.000193UTC timer1 발화 뒤14:13:07UTC same6dd/8SUCCESS/Linux 진행을 직접 조회했다.
완료8logs442775B/매 chunk readback·checkoutb571e02 전체tree feab8ef9/2891files가 source와
같다. Required/전체 Linux는OPEN이며 다음 due14:33:07UTC/session84800을 등록했다.
[P5-B1 receipt](./evidence/package-client-quarantine/p5b1-node-entry-bytes-local-20261028.json)의
Quick256.332s/npm fullRace49.546s/docs/security가 PASS다. 기존1283source와 validator는
불변이고 새2file의 bytes/Node 선언 value만 검증했다. Runtime 모듈 import 실패 후보를 보존하고
기존 Architecture를 유지했다. A/B1은 실제 P5 run 연결이 없어 WIRED=NO다. 다음 B2의 whole
graph/argv/runtime/source-policy와 전체 실행·CLI는 미완료다. CI423 source는6dd로 고정한다.

### 2026-10-28 P5-B1 selected bytes / Node runtime 착수

[P5-B1 audit/map/두 번째 검토](./evidence/package-client-quarantine/p5-execution-foundation-baseline-20261028.md)에
따라 기존 inventory/bin/Node validator를 재사용하고 선택 파일 bytes의 size/digest·closed
interpreter와 기존 runtime identity를 묶는다. 전체 검역·실행 authority/CLI는 다음 gap이다.
CI423 source6dd3136과 실제20min timer31554/due14:12:50UTC를 유지하고 새 push는 없다.

### 2026-10-28 승인된4commit CI423 시작

사용자가 exact4commit/18file 후보6dd3136의 기존 공개 milestone branch push·CI를 승인했다.
전체2891index/두 사본/commit tree feab8ef9와 원격1203445의 FF ancestry를 확인하고 해당
참조만 push했다. [CI423/run40700511803](https://github.com/rahoney/heliopause/actions/runs/40700511803)가
13:52:50UTC 시작돼9jobs 등록/Docs SUCCESS를 확인했다. 실제20min timer session31554의
첫 due14:12:50UTC(23:12:50KST)에 직접 상태를 조회한다. Source6dd3136 이후 local 변경은
포함하지 않았고 같은 M13-004/P5의 독립 다음 작업을 계속한다. Whole qualification은 결과 전 OPEN이다.

### 2026-10-28 CI422 canonical qualification 입력 연결 교정

[로컬 receipt](./evidence/package-client-quarantine/ci422-qualification-input-fix-local-20261028.json)의
Quick124.549s/영향 fullRace28.622s/tag/docs/security·no-overlay resolver actual45.715s가 PASS다.
기존 full CPU와 같은 canonical pin만 positive에 연결했고 metadata hash를 wheel로 전용하는
negative를 추가했다. Production guard/SDK/CPP/CI job·예산은 그대로다. Source/Host/입력·helper
정리를 확인했으며 whole hosted/final/signed는 미완료다. 누적 source의 검토 후 기존 CI의
정확한 후보 재qualification을 준비한다. 새 public push/timer/재실행은 아직 없다.

### 2026-10-28 CI422 same-source report DATA 진단

[진단 receipt](./evidence/package-client-quarantine/ci422-pytorch-report-diagnostic-20261028.json)의
local12.244s는 pip report archive_info={}와 공식 Simple의 같은 wheel hash 부재를 확인했다.
Sidecar metadata SHA는 wheel hash가 아니다. 원래 CI report는 없으므로 같은 후보라는 주장은
추론으로 남긴다. Source/Host/입력 불변과 helper 정리를 확인했으며 기존 거부를 유지한다.
다음은 기존 qualified exact fixture·CI 입력 owner 감사이며 새 runtime pin/검증 완화·재CI는 없다.

### 2026-10-28 CI422 실패 원본·소스 대조

[실패 receipt](./evidence/package-client-quarantine/ci422-p4-current-cohort-failure-20261028.json)의
8SUCCESS/Linux·Required FAILURE와 all10 checkout/source whole tree 동일성을 확인했다.
Linux24PASS/noSKIP/CPP128 뒤 torch report candidate가 거부됐고 raw report/정확한 조건은 아직
미확정이다. Helper6wait0/최종 오류1 보존·원본 client 복원은 성공이다. Full CPU와 뒤 pipx는
미도달이며 whole qualification을 닫지 않는다. 실제20min timer2를13:13:49UTC 소비했고 현재
active timer/CI는 없다. 재실행 없이 same-source bounded DATA 진단을 계속한다.

### 2026-10-28 P5-A 구조적 진입점 선택 로컬 검증

[P5-A receipt](./evidence/package-client-quarantine/p5a-execution-entry-local-20261028.json)의
Quick37.506s/npm fullRace49.102s/focused/docs/security가 PASS다. 기존1281source/parser는
불변이고 새2file의 구조적 선택·원본 전체 binding만 검증했다. 실행·CLI/전체 P5는 미완료다.
첫 test ID 형식 오류와 교정을 보존했다. CI422는 두 번째 실제20min 조회13:13:49UTC에
Linux/Required FAILURE·나머지8SUCCESS다. 최초 torch pip report candidate 거부와 helper6wait0/
복원을 확인했으며 현재 원인 진단을 계속한다. Timer2는 소비 종료했고 새 timer/재실행은 없다.

### 2026-10-28 P5-A exact bin 선택 착수

[P5-A 지도·두 번째 검토](./evidence/package-client-quarantine/p5-execution-foundation-baseline-20261028.md)에
따라 기존 npm inventory/bin parser를 재사용하는 구조적 typed entry 선택만 추가한다. 기존
parser/runtime/검역 authority는 보존하며 현재 전체 검사·실행·CLI는 후속이다. CI422 source는
1203445로 고정하고 20분 타이머를 계속 소비한다. M13-004 하나만 active다.
종료 시험 ELF 사본18개/364359003B는 원본·전체 참조0/query errors0 확인 후 제거했으며 현재
바이너리/고정 입력·공유 cache/작은 raw는 보존했다. Raw p4-e-completed-copies-cleanup이 근거다.

### 2026-10-28 CI 대기 중 P5 baseline 독립 감사

[P5 baseline](./evidence/package-client-quarantine/p5-execution-foundation-baseline-20261028.md)은
기존 bin/layout parser·inspection/staging·sandbox/Host output·terminal renderer의 실제 owner를
분리했다. Pinned never-started image에서 exact npm source DATA를 읽고 lookup/bin inference를
확인했다. Source/SDK·runtime pin을 바꾸지 않았으며 P5 production은 NOT_STARTED다.
Root README/PROJECT-DECISIONS의 낡은 e859 승인 대기/CPP 다음 단계 표기도 current CI422로
정정한다. 현재 단일 active item은 M13-004, 승인된 CI source는1203445로 유지한다.

### 2026-10-28 승인된28commit source1203445 CI422 시작

사용자의 필수 test 진행 지시로 검토한 exact28commit/163files를 기존 공개 repository의 같은
milestone branch에 정상 push했다. Remote e859→1203445이며 후속 local 변경은 포함하지 않았다.
[CI422/run40688745132](https://github.com/rahoney/heliopause/actions/runs/40688745132)가
12:33:12UTC 시작돼 initial9jobs 중 corpus/vulnerability/docs SUCCESS와 나머지 진행을 확인했다.
첫 timer가12:53:12UTC에 발화했고12:53:42UTC fresh run/jobs의 같은 head·8SUCCESS/Linux 진행을
직접 조회했다. Consumed receipt를 별도로 보존하고 다음 due13:13:42UTC/session29688을 등록했다.
Current source와 Required/원본 logs/checkout 일치 전 qualification은 OPEN이며 기존120min
Linux timeout과 coverage는 그대로다. 같은 M13-004의 독립 P5 baseline 감사를 이어간다.

### 2026-10-28 C-e3 common readiness named local 완료

[한정 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-e3-common-readiness-local-20261028.json)의
current UV4 152.771s/Bun exact4 165.549s/PNPM exact7 201.445s가 모두 PASS다. Public selectors,
transitive/peer exact quarantine·현재 Evidence/Policy와 Artifact 정상/exit37·long-name를 확인했다.
PNPM 첫4개는 isolated probes로 구분한다. Source1281/CPP/42input/ZIP·Host/Docker 불변,
helper각3wait0/부재·production client 복원/Host prefix 부재다. Fixture upper0700의 loader 실패와
trimpath/runtime.Caller 경로 실패를 보존했고 source를 바꾸지 않고 기존 CI recipe로 교정했다.
Local named scopes만 닫으며 current whole hosted/UID1001/macOS·final4profiles/signed 및
P5/P6는 미완료다. 같은 M13-004/C-e4에서 grouped candidate 검토·원격 qualification 준비를
계속한다. 새 push/CI/timer는 없다.

### 2026-10-28 C-e2 original ZIP/current graph/native readiness 한정 완료

[한정 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-e2-readiness-original-local-20261028.json)의
기존 validator/grep2reader 후보는 Quick232.928s/영향3package fullRace31.342s/tagged checks와
CPP256.108s/128latches, original required42.618s/mixed90.645s/current graph82.159s 및 exact
native3의163.064s가 PASS다. Mixed 정상20unit/5commands의 CPU11.758667/12.028750s와 실제
Read 뒤 오류 주입 거부를 확인했다. Helper/OCI·전체 FS/nonce/role guards/원본 pin·30/17/13
예산은 불변이며 Source1281/CPP/input42/ZIP·Host/Docker 동일/helperwait0·부재다. Graph는
unsigned13subject source/staging fixture와 production authenticity 거부이며 signed/native 전체가 아니다.
첫 CPU 거부와 직접 builtin reader의 stream fault/미채택, 실행 전 operator filename 교정도 보존했다.
같은 M13-004/C-e3에서 기존 common UV/Python/PNPM/Bun 영향과 grouped 후보를 계속 검증한다.
Current whole hosted/UID1001/final/signed·다른 native recipe/run은 미완료이며 새 push/CI는 없다.

### 2026-10-28 C-e1 CPU 진단 audit 한정 종료

[진단 assessment](./evidence/package-client-quarantine/p4o4c3c3b3c-e1-cpu-diagnostic-assessment-20261028.json)의
원본42.703s/13.012223s와 동일657payload ZIP encoding45.627s/13.127181s는 CPU 거부다.
Encoding 후보는 채택하지 않았다. 원본 ZIP의 별도 phase 계측88.730s는 기존20unit 정상→실제
Read 뒤 오류 주입/zero commands→20unit 정상으로 PASS이며 CPU12.815358/12.873832s다.
관측/종료 구간과 kernel user/system을 얻었고 원인 수리·일정 성능 보장은 하지 않는다.
Source1280/CPP/42input/원본 ZIP·Host/Docker 불변/helperwait0·부재다. Product 변경·예산 완화는
없으며 legacy production/current whole qualification은 OPEN이다. [Owner matrix](./evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)에 따라
C-e2의 grouped hosted/current whole acceptance 준비를 계속한다. 새 push/CI/timer는 없다.

### 2026-10-28 C-d2 existing hosted tools wiring 한정 완료

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-d2-hosted-tools-wiring-local-20261028.json)의
v3 Quick280.009s/영향4package Race188.792s/tag/docs/security와 unchanged checker Race39.590s와 exact CI tools-native 154.349s/
세 parent, tools-bootstrap 573.679s/네 parent/noSKIP가 PASS다. Existing job의 두
client/helper owner에 fixed namespace ZIP·ordinary UID·pinned interpreter DATA를 연결했다.
Short native의 -E/원래 RECORD를 독립 wheel·EP로 증명했다. 첫 frozen supply 누락/overlay
cleanup NO와 short 원본 거부·진단/복구는 보존하며 최종 Host/입력/source 불변과
helperwait0·workspace 부재를 확인했다. Hosted/UID1001·final/signed/legacy CPU gap은 별도다.
같은 M13-004/C-e1에서 남은 acceptance/matrix와 CPU owner audit를 계속한다. 새 push/CI는 없다.

### 2026-10-28 C-d1 fixed native pipx original 한정 완료

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-d1-native-original-local-20261028.json)의
Quick227.569s/영향4package 전체 Race179.722s/tag/docs/security 및 source45 actual366.291s
UV·pipx 두 parent가 PASS다. Genuine pipx raw exact6.0 원본의 독립 whole source/script/interpreter
증명 뒤 bare6.1 반영, native11/managed9controls·새 현재 Evidence, bounded native6.1 no-op와
원본/현재 Evidence 실패 보존을 확인했다. 기존 UV native10/8도 PASS이며 source/Host/입력/정리는
모두 확인했다. Native 기본 recipe/run·hosted/final/signed 및 legacy ZIP CPU gap은 남는다.
같은 M13-004/C-d2에서 기존 hosted 소비자의 grouped 연결 audit를 계속한다.

### 2026-10-28 C-c3 public production pipx CLI 한정 완료

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-c3-public-cli-local-20261028.json)의
최종 Quick63.237s/tag/docs/security·기존 전체 CLI/bootstrap Race27.752s 및 source36 actual374.394s
네 parent가 PASS다. 실제 UV·pipx JSON/한국어 human·spaces fresh/control6와 managed/control8·9,
현재 승인 Manifest·receipt/link/user data 및 무변경 거부를 확인했다. Distinct pipx factory와
명시 lifetime owner를 연결하고 English help도 descriptor와 대조했다. Native adoption/run/hosted/
final은 별도이며 같은 M13-004/C-d1의 독립 원본 source/script/interpreter proof를 계속한다.

### 2026-10-28 C-c2 managed approved pipx backend 한정 완료

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-c2-managed-approved-backend-local-20261028.json)의
Quick181.693s/영향3package 전체 Race170.689s/tag/docs/security와 actual311.132s가 PASS다.
실제 승인한6.0 원본에서 bare upgrade6.1 및 same-version의 current 검사/Evidence·Policy/control9를
확인했다. Current Evidence write 실패는 prior ALLOW를 우회하지 못하고 기존 inode/bytes/links를
보존했다. 기존 UV managed·Evidence 거부도 PASS이며 여섯 공통 owner는 unchanged다. Production
CLI/native adoption/hosted/final은 별도이고 같은 M13-004/C-c3의 CLI/factory를 계속한다.

### 2026-10-28 C-c1 fresh approved pipx backend 한정 완료

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-c1-fresh-approved-backend-local-20261028.json)의
Quick147.053s/영향3package 전체 Race225.525s/tag/docs/security와 actual128.572s가 PASS다.
기존 UV·새 pipx의 plain/space 네 설치는 current whole 검사/Evidence·Policy ALLOW/control6와
동일 승인 반영·user data/Host 보존 및 fresh 재진입 무변경 거부를 확인했다. 기존 공통 엔진을
재사용하고 internal app path와 외부 link 목적지를 분리했다. 실제 composition은 test fixture이며
production CLI/managed·native original/hosted/final은 별도다. 같은 M13-004/C-c2를 계속한다.

### 2026-10-28 C-b3b original-aware native upgrade/no-op 한정 완료

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-b3b-original-upgrade-local-20261028.json)의
Quick95.439s/전체 sandbox Race25.087s/tag/docs/security·CPP128 및 actual91.332s가 PASS다.
Fresh6.0→upgrade6.1→same6.1 no-op의 whole source/control4·7·7과 typed 세션3개/정리·Host
불변을 확인했다. 최초 output scaffold 거부는 보존하고 기존 pinned UV venv owner를 재사용했다.
Selection은 승인 전 상태이며 같은 M13-004/C-c에서 current 검사·Policy/cache를 연결한다.
Guarded Host transaction/CLI·hosted/final·독립 legacy ZIP CPU gap은 남는다.

### 2026-10-28 C-b3a inert original replay/port 한정 검증

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-b3a-inert-original-replay-local-20261028.json)의
Quick206.627s/전체 sandbox Race26.962s/tag/docs/security가 PASS다. 기존 UV serializer는
byte-identical하게 재사용하고 pipx 원본에는 generated metadata/scaffold·inert command만
넣었다. Exact tar/mode/RECORD·invalid/cancel/collision/변조 command0를 검증했으며 actual
original-aware upgrade/Policy/CLI는 미완료다. 같은 M13-004/C-b3b에서 연결과 실제 검증을 계속한다.

### 2026-10-28 C-b2 public pipx selection/factory 한정 완료

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-b2-public-native-selection-local-20261028.json)의
Quick86.100s/전체 sandbox Race27.758s/tag/docs/security와 actual51.055s가 PASS다. Cowsay6.1
public 원본 URL/SHA와 native graph1/command1/control4를 결합했고 idna3.11은 native transaction
뒤 zero selection으로 거부했다. Typed Create/Register/Close각2·no observer fault/Host·원본·source
불변/helperwait0/final0다. Fresh selection만 qualified이며 같은 M13-004/C-b3 original upgrade,
current 검사/Policy·cache·Host transaction/CLI·최종 profiles와 legacy CPU gap은 남는다.

### 2026-10-28 C-b1 고정 설치 공급물 entry 한정 완료

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c-b1-installed-native-local-20261028.json)의
Quick104.819s/전체 sandbox Race24.782s/tag/docs/security 및 same CPP18 actual16.823s가
PASS다. Root-owned0444/고정 current ZIP를 ordinary caller가 읽어 fresh controls2/graph2와
회계 거부 target0을 확인했다. 첫 overlay0700 launcher 오류는 제품 미도달로 보존했고
fixture0755 guard 뒤 정상이다. Host·원본·source 불변/helperwait0/final0이며 C-b2 public
selection/factory·original upgrade/current Policy/CLI/final profiles·legacy CPU gap은 OPEN이다.

### 2026-10-28 C-a 실제 native metadata 한정 완료

[로컬 receipt](./evidence/package-client-quarantine/p4o4c3c3b3c1-native-session-local-20261028.json)의
CPP128/actual17.003s는 fresh controls2/graph2·CPU2.128257s와 genuine accounting-refusal의
target0/zero selection을 검증했다. Source·고정42/ZIP·Host5/Docker 불변, helperwait0/final0·
부재다. C-a named offline scope만 qualified이며 C-b installed supply/public caller·original
upgrade/current Policy/CLI/final profiles와 기존 ZIP mixed CPU parent는 OPEN이다. 같은
M13-004에서 다음 missing audit/연결을 계속하며 새 push/CI/timer는 없다.

### 2026-10-28 C-a query/diagnostic 중간 검증

[한정 checkpoint](./evidence/package-client-quarantine/p4o4c3c3b3c1-native-query-diagnostics-checkpoint-20261028.json)의
Go canonical/focused Race·CPP128와 actual cfg32768/parent 관계/first-finding·trusted cleanup을
보존한다. v16은 outside3의 최초 exact venv site-packages624640와 후속 immutable
os-release557056에서 미완료다. 기존 ZIP actual의 normal-before20units/5commands와
회계 거부는 PASS, normal-after는 활성 CPU13s 경계 INCOMPLETE이며 전체 qualification은
OPEN이다. Budget/coverage나 CPU 인과를 바꾸지 않고 현재 native read gap을 계속한다.

### 2026-10-28 C-a native 관찰의 첫 실제 경계 확인

기존 Go canonical/영향 Race와 typed session은 구현됐고, 고정 원본42/ZIP의 실제 실행은
UV startup의 exact read에서 중단됐다. v5에서 자기 proc 읽기를 지나 최초 오류는
CPU quota/system/557056이었고, v6의 다음 오류는 CPU period 읽기였고 v7은 DEV_NULL 읽기에서 중단됐다. v8은 base query의 OCI directory624640과 별도
cleanup 실패를 보존했고, 정확한 잔여 DATA/keeper 소유권을 확인해 복구했다. v9의
query flag/새 trusted cleanup 경로는 실제 통과했으며 다음 active UV의 고정 interpreter
read557056를 검증했다. v10의 다음 proc/OTHER 읽기는 /proc/24/comm 후보이며
v11 판정 불변 실행에서 자기 TGID 관계는 반증됐다. 부모/current thread 관계의 진단만
확장한 v12 actual에서 PROC_PARENT_COMM을 확인했다. 기존 live root creator에 묶인
exact parent comm read는 v13에서 통과했고 그 뒤 첫 outside4와 후속 OCI fault를 보존했다.
현재 권한을 유지한 v14 first-outside diagnostic을 검증 중이다. 다른 PID/SELF·thread는 불허다. 각 실행의 Host/Docker/입력·source 불변과 helper
wait0/부재를 확인했고 원본은 보존했다. [Owner map](./evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)의
source 전 tuple audit에 따라 필요한 native-only read만 검증한다. [중간 구현/checkpoint](./evidence/package-client-quarantine/p4o4c3c3b3c1-native-session-checkpoint-20261028.json)의
Go canonical/전체 영향 Race·CPP128는 통과했고 실제 native install은 미완료다. C-a QUALIFIED/
ACCEPTANCE_CLOSED는 NO, installed supply·upgrade/current Policy·CLI는 후속 MISSING이다.

### 2026-10-28 CI421 Required10 성공·원본 대조

[CI421 qualification](./evidence/package-client-quarantine/ci421-pipx-zip-primitives-hosted-qualification-20261028.json)의
source e859487/checkout57a5633 전체 tree6753df83·2789파일이 같고 Required 포함10SUCCESS다.
10 decoded logs1,150,764B/readback·Linux79/noSKIP/helper11wait0/final confirmed lifecycle,
두CPP124 및 CPU236.66s를 확인했다. Workflow64m47이며 Source/supply/common와 기존
hosted 범위만 닫는다. New ZIP actual parent는 미실행이고 후속 B3/C3/native는 포함되지
않았다. 실제 timer4 발화00:16:10.000150UTC 뒤00:16:42UTC에 완료를 직접 조회했다.
추가 timer/push/재dispatch 없이 local C-a를 계속하며 CI415 원인은 OPEN이다.

### 2026-10-28 CI421 세 번째20분 직접 확인

Timer23:55:24.567997UTC 발화 뒤23:56:10UTC fresh run/jobs에서 같은 e859487 head와
8SUCCESS/Linux IN_PROGRESS를 확인했다. Raw ci421-poll03/timer-fired가 근거이며
다음20분 due2026-10-28 00:16:10UTC(09:16:10KST)를 등록했다. 새 push/취소 없이
local native C-a의 실제 소비자 연결을 계속한다. Required/qualification은 OPEN이다.

### 2026-10-28 CI421 두 번째20분 직접 확인

Timer23:34:47.000171UTC 발화 뒤23:35:24UTC fresh run/jobs의 같은 e859487 head와
8SUCCESS/Linux IN_PROGRESS를 직접 확인했다. 원본은 raw ci421-poll02/timer-fired다.
다음 실제20분 timer due23:55:24UTC(08:55:24KST)를 등록했다. Required/qualification은
OPEN이며 같은 C3c3b3c의 local native session baseline·연결 작업을 계속한다.

### 2026-10-28 C3c3b3b CPP 전용 Artifact profile 로컬 검증

[C3c3b3b receipt](./evidence/package-client-quarantine/p4o4c3c3b3b-native-observer-local-20261028.json)의
canonical128latches/production Open→OpenResult RED→GREEN·Quick/Race와 기존 실제10parent
범위를 확인했다. 첫 ZIP CPU 한도 거부는 보존했고, 동일 진단 binary의 old/new 대조는
각1회 PASS이며 제품/17s reserve/coverage를 바꾸지 않았다. 새 native gVisor 실행/전체
accounting·installed supply/current Policy/CLI는 남으므로 같은 M13-004의 C3c3b3c에서
관련 baseline/Read를 다시 확인하고 그 acceptance gap만 연결한다. CI421 e859487에는
이 후속 local source가 없고, 첫20분8SUCCESS/Linux 진행·다음23:34:47UTC timer를 유지한다.

### 2026-10-28 CI421 첫20분 직접 확인

실제 timer는23:14:33.000137UTC 발화했고23:14:47UTC에 fresh run/jobs를 직접 조회했다.
Head e859487가 같고8SUCCESS/Linux IN_PROGRESS이며 Required/qualification은 OPEN이다.
Raw ci421-poll01/timer-fired가 원본을 소유한다. 다음 실제20분 timer의 due는
23:34:47UTC(08:34:47KST)다. 재push/재dispatch/취소 없이 같은 local C3c3b3b를 계속한다.

### 2026-10-28 승인된6commit checkpoint CI421 시작

사용자의 권장안 진행 지시를 승인 대기 중이던 exact6commit payload의 승인으로 소비했다.
Fresh PR38 head1e5d21e를 확인하고 기존 milestone branch에 e859487까지만 fast-forward
push했다. 후속 local B3/C3 commits와 미완료 CPP 후보는 전송하지 않았다.
CI421/run40598317489 head e859487가22:54:33UTC 시작됐고 첫 실제9jobs IN_PROGRESS다.
최근 같은 workflow73m44를 완료 예상의 근거로만 쓰고, 실제20분 timer session27620의
첫 due23:14:33UTC(08:14:33KST)에 run/jobs를 직접 조회한다. 원본은 raw ci421-initial.json/
/tmp/haa-ci421-poll01.json이며 DUE만으로 조회 완료라 하지 않는다. Required/원본 로그/
source 대조 전 qualification은 OPEN이다. 같은 M13-004/C3c3b3b의 local 검증을 계속한다.

### 2026-10-28 P4-O4-C3c3b3a offline constructor/Go topology 로컬 검증

[C3c3b3a receipt](./evidence/package-client-quarantine/p4o4c3c3b3a-offline-runtime-local-20261028.json)의
Quick390.740s/전체 영향 Race24.889s/tag vet·build/docs/security와 actual12.907s가 PASS다.
고정 network none/cwd `/`/환경·sealed RO inputs와 same-frozen-ID output을 결합하고
기존 memory/tmp를 유지·per-process CPU를30으로 좁혔다. Actual은 runtime flag만 runc로
바꿔 원본 control/native install/no-op·RO/불변·정리를 확인했다. New Go6mount/default
trace와 old UV/ZIP4mount 분리를 검사했지만 CPP native profile/실제 session/Policy/CLI는
미완료다. 같은 M13-004/C3c3b3b에서 기존 CONTROL 권한을 넓히지 않는 별도 ARTIFACT
chain과 negative latches를 계속한다. 기존 push 승인 대기이며 새 CI/타이머는 없다.

### 2026-10-28 P4-O4-C3c3b2 complete native output 로컬 검증

[C3c3b2 receipt](./evidence/package-client-quarantine/p4o4c3c3b2-complete-native-output-local-20261028.json)의
Quick54.864s/전체 영향 Race23.501s/tag vet·build/docs/security와 actual13.048s가 PASS다.
전체 tree의 layout/owner/type/mode/bounds·receipt/link를 검사하고 actual 두 metadata
control을 immutable 원본 size/SHA에 결합했다. 기존 parser/graph/replay와 별도 capture의
일치·정상 nested payload/unknown·변조·producer error/cancel 거부와 모든 원본/Host
불변·정리를 확인했다. 첫 collision fixture는 기존 DIST_INFO_IDENTITY가 먼저 거부해
기대만 교정했고 production은 바꾸지 않았다. Unapproved native payload의 승인이나
gVisor/Policy/CLI 성공은 아니다. Fresh CI420/Required10SUCCESS와 PR38 head1e5d21e를
다시 확인했다. 같은 M13-004/C3c3b3 전용 offline profile/session을 계속하며 기존 push
승인은 대기다. 새 CI/타이머는 없다.

### 2026-10-28 P4-O4-C3c3b1 전체 native DATA census 로컬 검증

[C3c3b1 receipt](./evidence/package-client-quarantine/p4o4c3c3b1-native-census-local-20261028.json)의
Quick57.091s/tag vet·build/docs/security와 actual12.554s가 PASS다. 기존 production/default
test bytes를 유지하고 bounded 전체 native tree의 facts/hash만 수집했다. Install84/508193B,
no-op85/510597B이며 차이는 새 로그1개와 pipx metadata다. Exact selected source5 SHA와
query의 실제 -c/import 순서도 대조했다. NoSKIP/Host5/Docker/42input/ZIP/source 불변과
정리를 확인했으며 data census는 production allowlist/observer/Policy가 아니다. 같은
M13-004/C3c3b2의 전용 complete output reader를 계속하고 기존 push 승인은 대기다.

### 2026-10-28 P4-O4-C3c3a frozen whole wheel/RO input 로컬 검증

[C3c3a receipt](./evidence/package-client-quarantine/p4o4c3c3a-frozen-readonly-wheels-local-20261028.json)의
Quick248.619s/전체 sandbox Race16.476s·PyPI cached/tag vet·build/docs/security와 실제
17.977s가 PASS다. Whole original/source/startup gate와 root0:0/0444/noexec 별도 RO
wheel input, single-use writer→anchor→제거→seal/readers를 결합했다. 실제 기존 ZIP과
새 wheel의 root write EROFS·foreign RW 거부/제거 후 정상, native install/no-op·config
대조/metadata/link/graph2/replay, noSKIP·Host5/Docker/42input/ZIP/source 불변과 정리를
확인했다. C2 공통 두 source의 좁은 fixed-root diff와 나머지 네 source 불변을 기록했다.
Production installed current/native session·gVisor/Policy/CLI는 미완료이며 같은 M13-004의
C3c3b baseline/edit map을 계속한다. 기존 e859487 push는 승인 대기이고 새 CI는 없다.

### 2026-10-28 P4-O4-C3c2 native program/offline 진단 로컬 검증

[C3c2 receipt](./evidence/package-client-quarantine/p4o4c3c2-native-program-offline-local-20261028.json)의
최종 Quick33.889s/tag vet·build/docs/security와 actual9.800s가 PASS다. Same product/default
test SHA의 첫 sandbox 전체 Race27.155s와 최종 cached repeat를 구분했다. Runc ordinary1000
native version/install/no-op·canonical demotion·strict metadata/link/graph2/replay와 guest
malformed config negative/positive, noSKIP/Host5·Docker·42input·ZIP·source 불변/정리를 확인했다.
첫 fixture Config.User 소유권과 준비 copy 실패 원본을 보존하고 product guard는 유지했다.
Native installed current/gVisor/Policy/CLI 성공은 아니며 diagnostic RW wheelhouse도 생산 RO
input으로 승계하지 않는다. 같은 M13-004/C3c3의 installed supply/RO input/session/profile
baseline audit를 계속한다. 새 CI는 없고 기존 e859487 payload push만 사용자 승인 대기다.

### 2026-10-28 P4-O4-C3c1 조회 전 whole wheel gate 로컬 검증

[C3c1 receipt](./evidence/package-client-quarantine/p4o4c3c1-prequery-wheel-gate-local-20261028.json)의
Quick251.469s/전체 PyPI Race9.934s/tag vet/docs/security와 원본 corpus2.640s가 PASS다.
Receipt 이전에 기존 whole archive/source/graph와 startup surface 거부를 연결했고 기존 UV/
fixed supply 의미를 유지했다. 42input hash/readback·원본 userpath/click graph2·변조 거부는
정적 parser-binding fixture 범위이며 native solver/query/Policy 성공이 아니다. 같은
M13-004/C3c2 fixed native program/argv/offline actual diagnostic을 계속한다. 기존 e859487
checkpoint push는 승인 대기이며 새 CI는 없다.

### 2026-10-28 P4-O4-C3b pipx original upgrade 입력 로컬 검증

[C3b receipt](./evidence/package-client-quarantine/p4o4c3b-original-upgrade-input-local-20261028.json)의
Quick248.212s/전체 PyPI Race9.898s/docs/security는 PASS다. 원본 constraint/version과 새로운
bare-project 요청, recipe/control/version/runtime 거부 및 complete METADATA/WHEEL replay를
검증했다. 기존 UV validation body/control·의미를 보존했다. Native query·ownership/current
Policy/CLI 성공은 아니다. 같은 M13-004/C3c1의 authenticated whole-wheel pre-query startup
surface gate를 계속한다. 새 CI는 없고 기존 e859487 payload push만 사용자 승인 대기다.

### 2026-10-28 P4-O4-C3a pipx inert layout 로컬 검증

[C3a receipt](./evidence/package-client-quarantine/p4o4c3a-receipt-layout-local-20261028.json)는
기존 parser/held layout를 재사용한 원본 receipt semantics·source interpreter 분리·opaque
recipe/생략·null 보존과 unsafe/changed claim 거부를 확인했다. Quick40.768s/PyPI 전체
Race9.298s/docs/security PASS이며 첫 fixture 타입 오류를 보존했다. Native/CLI·ownership/
current Policy 성공은 아니다. 같은 M13-004/C3b의 bounded metadata-only original input과
replay 검증 재사용을 계속한다. 새 CI는 없고 e859487 push 승인은 계속 대기한다.

### 2026-10-28 P4-O4-B3d 기존 hosted actual 소비자 연결·로컬 recipe 검증

[B3d receipt](./evidence/package-client-quarantine/p4o4b3d-hosted-actual-wiring-local-20261028.json)는
기존 Linux job에 fixed42input inspect/graph/namespace와 owned client/helper/production restore를
연결했다. Actual135.807s/로컬UID1000의6개 exact top-level PASS/noSKIP·helper4wait0/confirmed
cleanup·client 복원/입력 directory0·Host5파일/Docker/소스 동일을 확인했다. Framework
empty/SKIP/child-only/duplicate/exit와 준비·cleanup 오류의 Bash 회귀, 최종 Quick47.782s/
checker 전체 Race40.351s·workflow/release-gate/docs/security도 PASS다. Logical fixture 입력
누락·operator FD bind의 최초 실패는 보존했고 제품 guard/예산은 완화하지 않았다.
HostedUID1001/새 CI·native pipx/signed activation/full matrix는 아직 미완료다.
동일 M13-004의 C3 installed pipx runtime/native install·upgrade/CLI owner audit를 계속한다.

### 2026-10-28 P4-O4-B3c 실제 graph·Evidence와 DATA 복구 로컬 검증

[B3c receipt](./evidence/package-client-quarantine/p4o4b3c-current-graph-data-recovery-local-20261028.json)는
실제 selected ZIP20unit→13subjects/32checks/37Evidence의 current store readback·Policy/staging,
missing runner 차단·unsigned production SDK 거부를 확인했다. Native6 sandbox/promoter는
fixture 범위다. 실제12payload255942560B root stage와 DATA copy/부분 복구/변조·foreign 보존,
legacy bootstrap/stage·pending24case/SIGKILL이 PASS다. DATA0555 partial copy cleanup 허용을
같은 test SHA로 RED→GREEN 검증하고 기존 copier mode helper를 cleanup에 재사용했다.
Host5파일/Docker/입력 불변/noSKIP·helperwait0/confirmed cleanup이며 actual 서명·Host activation
또는 hosted/full profile 완료가 아니다. B3d hosted actual 입력/소비자 연결부터 계속한다.

### 2026-10-28 P4-O4-B3b complete DATA producer·metadata·package 로컬 검증

[B3b receipt](./evidence/package-client-quarantine/p4o4b3b-producer-data-local-20261028.json)는
frozen42input 취득→기존 offline materializer→명시 runtime metadata v3/3tools→15asset/19bundle을
검증했다. Actual original ZIP의11distribution과 SBOM 일치, DEB/RPM 원본 DATA root0444·
재생성 byte-identical·Host 보호5파일/Docker/입력 불변/noSKIP를 확인했다. 기본 Quick와 관련
전체 Race, workflow/release-gate/docs/security 및 명시 actual tests가 PASS다. 로컬 unsigned
fixture이며 서명·hosted qualification·패키지 설치·artifact 실행 성공이 아니다. 같은 M13-004의
B3c actual complete graph/current Evidence·Policy와 protected DATA 복구를 계속한다.
원격 e859487 checkpoint는 기존 사용자 승인 대기로 새 CI/timer를 시작하지 않았다.

### 2026-10-28 P4-O4-B3a complete DATA 선택·mode 로컬 검증

[B3a receipt](./evidence/package-client-quarantine/p4o4b3a-complete-data-selection-local-20261028.json)는
native6 보존·ZIP/NOTICE pair·legacy10/new12payload와11/13subjects를 한 selector로 연결했다.
Graph/wire/route의 정상·변조·누락 runner 거부, unsigned fixture의 authenticity 거부와 초기
Quick187.064s/affected6 owner Race171.897s·추가 후보 Quick42.684s/관련3 owner Race14.630s를
확인했다. Actual selected ZIP/NOTICE의 private mount/network namespace 복사0.221s는 root
single-link0444·Host/Docker/원본 동일/noSKIP다. File mechanics 범위이며 producer/metadata와
actual complete graph current Evidence/Policy·root DATA 복구·native pipx/CLI·hosted/final profiles는
남는다. 같은 M13-004/B3b를 로컬에서 계속하고 원격 e859487 checkpoint는 승인 대기다.

### 2026-10-28 공통 checkpoint push의 외부 승인 대기

로컬 source e859487/tree6753df83/2789까지 검증·commit했다. 기존 원격 branch는1e5d21e다.
정상 `git push origin milestone/m13-release-build-packages`는 실행 전에 자동 승인 검토가
private 코드/영수증 외부 전송의 명시 목적지·payload 승인 근거 부족으로 거부했다.
거부를 우회하거나 재시도하지 않았다. 사용자에게 기존 GitHub rahoney/heliopause의
같은 milestone branch로21387da~e859487 여섯 commit push 승인을 비동기로 요청했다.
해제 조건은 해당 목적지/payload의 사용자 승인이다. 새 CI는 아직 시작되지 않았고
20분 실제 조회는 시작 뒤 연결한다. 이 checkpoint만 승인 대기이며 M13-004 전체는
IN_PROGRESS다. 같은 B3의 독립 로컬 DATA membership 작업을 계속한다. 로컬 원본
p4o4b2d-push-auto-review-rejection.json이 거부와 미실행 상태를 보존한다.

### 2026-10-28 P4-O4-B2d required 검사·Evidence 로컬 검증

[B2d receipt](./evidence/package-client-quarantine/p4o4b2d-required-runtime-inspection-local-20261028.json)는
최종 Quick42.543s/inspector Race2.132s, affected4 package full Race51.296s와 selected ZIP/
docs/security, 실제 required inspection41.606s를 연결한다. 원본 HAA identity의 required
check1/Finding0/Evidence6·commands5와 후속 intake 변조 거부를 확인했다. Import cycle·
summary1024B·ID64B와 native capability의 첫 실패 원본을 보존하고 기존 경계/한도를 유지했다.
NoSKIP/helperwait0/Host 동일이며 source는 공통 hosted checkpoint로 묶는다. 새 ZIP actual은
현재 named local이며 기존 CI의 hosted 소비자 범위와 구분한다. 같은 M13-004/B3에서 complete
release DATA membership/build/sign/install/recovery를 계속한다. Whole pipx/CLI·final profiles는 남는다.

### 2026-10-28 P4-O4-B2c acquired ZIP 실제 transaction 로컬 검증

[B2c receipt](./evidence/package-client-quarantine/p4o4b2c-acquired-runtime-transaction-local-20261028.json)는
최종 canonical 검사·sandbox 전체 Race, canonical124latches와 새 ZIP actual87.436s의
20units 정상→회계 오류 주입 INCOMPLETE/commands0→20units 정상, 기존 wheel/네 작은 policy·
prerequisite sequence151.871s를 연결한다. 각 정상 Register22/Close1·CPU12.76/12.47s이며
한도는 유지했다. 첫 준비 실패의 shell resolved path를 정확한 pinned tuple로 교정했고
두 원본을 보존했다. NoSKIP/helperwait0/confirmed cleanup·보호5파일/Docker/원본 ZIP 동일이다.
Named local scope만 닫고 required release composite/factory는 같은 M13-004/B2d에서 계속한다.
Hosted/common/full CPU·CUDA 및 전체 pipx/CLI acceptance는 남는다. 개별 helper CI는 없다.

### 2026-10-28 P4-O4-B2c 공통 transaction 연결·readiness 교정 중

[B2c owner map](./evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md#b2c--기존-transaction-연결의-source-전-세부-지도)에 따라 기존 원장/CPU watch/unit loop/cleanup을
공유하고 acquired ZIP은 기존 C2 DATA seed/bind/seal만 사용한다. 첫 Go Quick202.663s/
Race22.662s PASS 뒤 source 대조에서 공통 readiness CONTROL launch와 B2b의 일괄 launch
거부 충돌을 확정했다. B2b의 해당 규칙은 superseded이며 canonical exact readiness tuple만
허용하는 후보의 Quick53.090s/Race22.217s/docs/security PASS를 확인했다. 실제 gVisor
transaction과 교정 observer의 canonical latches는 아직 진행 중이다. 작은 helper별 CI는
추가하지 않고 같은 M13-004 하나에서 계속한다.

### 2026-10-28 P4-O4-B2b fixed observer profile 로컬 검증

[B2b receipt](./evidence/package-client-quarantine/p4o4b2b-observer-profile-local-20261028.json)는
새 profile/topology·동일 shared budget의 Quick204.610s/sandbox Race25.051s/docs/security와
canonical observer build274.583s/전체124latches PASS를 확인했다. 기존 wheel profile을
유지하며 새 ZIP은 단일 DATA path와 ARTIFACT Python handoff를 사용한다. 최초의 pip/site·
CONTROL launch/ELF handoff 거부 중 CONTROL launch 일괄 거부는 B2c에서 반증되어 교정 중이다. Host helper 설치·활성화와 실제 ZIP transaction 완료는
아직 없으며 같은 M13-004/B2c에서 기존 실행/회계/정리 owner를 연결한다.

### 2026-10-28 P4-O4-B2a 입력과 고정 program 로컬 검증

[B2a receipt](./evidence/package-client-quarantine/p4o4b2a-acquired-input-program-local-20261028.json)는
원본 HAA acquired identity/handle와 immutable ZIP DATA·20unit의 결합, 기존 C1 transport
재사용을 확인했다. Quick239.916s/sandbox Race25.955s/actual6.716s/docs/security PASS와
새 고정 program의 runc20 exit0·사전 거부3개/보호5파일·Docker·원본 불변을 확인했다.
이는 source/input/program 범위이며 외부 observer·실제 gVisor transaction 완료가 아니다.
같은 M13-004/B2b의 fixed profile/topology로 이어가고 작은 helper 단독 CI는 실행하지 않는다.

### 2026-10-28 CI420 최종 확인과20분 직접 조회 요청

[CI420 qualification](./evidence/package-client-quarantine/ci420-pytest-supply-hosted-qualification-20261028.json)은
Required 포함10SUCCESS와 all10 checkout729b/source1e5d21e tree391df742/2772 동일을
확인했다. 시작13:48:39UTC→Required15:02:23UTC는73분44초, Linux73분7초다.
원본10logs1,163,500B의 readback/hash와 Linux79PASS/noSKIP·CPU248.37s·helper11wait0/
최종 confirmed cleanup을 대조했다. 기존 공통 공급/checker와 hosted 소비자 범위이며 새 ZIP
gVisor 외부 관찰/전체 pipx 지원 완료는 남는다. CI415의 역사적 FD 원인은 계속 OPEN이다.

사용자는 이후 CI를20분 타이머로 직접 조회하고 작업을 계속하도록 요청했다.
[검증·감시 규칙](./20-package-client-quarantine-plan.md#5-검증-matrix와-자원-정리)의 간격을
갱신하며 timer DUE와 실제 조회를 구분한다. 이번 final은 사용자 재개 뒤 fresh fetch이며
지난 응답 종료 후 지속 자동 감시했다고 표시하지 않는다. 현재 진행 중인 CI는 없다.
같은 M13-004/B2의 준비된 baseline에 따라 acquired typed 공급/고정 관찰 unit부터 진행한다.

### 2026-10-27 CI420 첫10분 실제 조회

13:58:39UTC 타이머가13:58:41UTC에 발화했고13:58:55UTC fresh run/jobs 조회는
동일 source1e5d21e의7SUCCESS, macOS와 gVisor Linux Integration IN_PROGRESS다.
Required 최종 결과는 미완료이며 실패로도 성공으로도 추정하지 않는다. 원본 readback/hash와
실제 timer는 local ci420-poll01-scope/inventory에 보존했다. 다음 권장 조회는
14:08:55UTC(23:08:55KST), 예상 완료는23:53–다음날00:08KST다.
외부 CI가 남아 이 checkpoint에서 인계하며 응답 종료 뒤 자동 조회·재개는 없다.
다음 재개는 이 exact run의 전체 conclusions·checkout/source tree·원본 로그와 cleanup을
확인한 뒤 같은 M13-004/B2 기존 transaction/observer 연결로 이어간다. B2 audit는 준비했고
새 product source/profile 구현은 아직 시작하지 않았다. 실행 중 후보를 재push하지 않는다.

### 2026-10-27 CI420 새 공급과 공통 검사 소비자 검증 시작

누적6 commits를 milestone branch에 push했다. Source1e5d21e/tree391df742/2772files는
검사 사본의 전체 index/blob/mode·disk와 동일하며 최종 docs0.310s/security9.627s,
실제 새 commit을 포함한 security-history59.464s도 PASS다.
[CI420/run40515351597](https://github.com/rahoney/heliopause/actions/runs/40515351597)은
13:48:39UTC(22:48:39KST)에 시작했다. 초기9개 선행 job이 생성됐고 Docs는 SUCCESS다.
직전 CI41965분35초와 동일 workflow를 근거로65–80분,23:53–다음날00:08KST를 예상한다.
첫10분 실제 시계 타이머는13:58:39UTC(22:58:39KST)이며 조회 완료와 구분한다.
기다리는 동안 [B2 수정 전 대조](./evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md#b2--새-공급-뒤-실행-owner의-수정-전-대조)를
진행했다. 기존 transaction/cleanup을 재사용하고 wheel profile의 pip·/haa-site 권한을
새 ZIP에 자동 승계하지 않는다. 실행 중 후보의 source를 추가 수정하거나 재push하지 않았다.
원본 initial run/jobs readback과 source/tree/checks는 local ci420-start-scope가 소유한다.

### 2026-10-27 P4-O4-B1d 명시 공급과 로컬 소비자 검증

[D2/D3 receipt](./evidence/package-client-quarantine/p4o4b1d23-pytest-runtime-supply-local-20261027.json)는
11wheel/42inputs/657원본 파일·새 Deflate ZIP과15required/5post-install plan, 실제 frozen
유지보수·HAA identity·DATA 전달·보호 파일/Docker 불변을 확인했다. Fixed Python runc의
15필수 imports와 기존 pytest plugin도 exit0이다. Gitleaks는 공식 metadata MD5 두 건을
오탐했으며 exact 원본 SHA256/path/rule/위치 확인과 다른 current/history token 검출 회귀를
통과했다. Quick187.424s/checker Race36.759s/security와 sandbox Race/workflow/release-gate도
PASS다. Common owner/checker hosted 회귀로 이어가며 B2 외부 관찰·release/native pipx/CLI와
전체 지원 완료는 남긴다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-O4-B1d 승인된 공급 확대 착수

[B1d baseline/수정 지도](./evidence/package-client-quarantine/p4o4b1d-pytest-runtime-supply-plan-20261027.md)에
따라 기존 publisher·whole wheel·single-root graph를 재사용해 두 명시 runtime 공급 root를
검증한다. pytest/iniconfig/pluggy의 제공 proof와 Pygments의 proof 부재를 구분하고, 전체 원본을
보존하는 Deflate 표현으로 기존8MiB DATA 전송 용량을 유지한다. 새 source/owner/pin은 아직
qualification 전이며 기존 필수 검사·resource budget·Domain single-primary 계약은 유지한다.

### 2026-10-27 P4-O4-B1b/c 검증과 명시 공급 확대 승인

[B1b/c receipt](./evidence/package-client-quarantine/p4o4b1bc-pipx-plan-intake-local-20261027.json)는
Quick44.492s/관련 전체 Race39.114s/actual0.895s/docs/security PASS,7 distributions/
9required·3post-install 계획과 원본 HAA identity/handle·bounded growth/cancel 거부를 확인했다.
고정 Python runc에서 loader/base import는 exit0, pytest plugin은 pytest 부재 exit1이다.
사용자가 권장안인 pytest와 의존성의 명시 runtime 공급 추가·새 전체 검증을 승인했다.
Required 범위는 유지하고 자동 보조 설치·plugin 선택 검사 전환은 하지 않는다.
같은 item의 B1d 공급 baseline/official 입력 확인을 계속한다. 새 runtime/observer·release와
전체 pipx 지원은 아직 미완료이며 이 로컬 helper 묶음만으로 CI를 반복 실행하지 않는다.

### 2026-10-27 P4-O4-B1a ZIP snapshot 로컬 검증

[B1a receipt](./evidence/package-client-quarantine/p4o4b1a-pipx-zip-input-local-20261027.json)는
Quick256.943s/PyPI 전체 Race8.661s/actual2.347s/docs/security PASS를 연결한다.
Selected2505814B/190files의 immutable copy와 변조·잘림·추가 바이트 거부를 확인했다.
Python은 실행하지 않았으며 original HAA intake binding·surface/typed plan·observer는 남는다.
같은 M13-004/B1의 다음 입력 연결을 계속하며 이 helper 단독으로 push하지 않는다.

### 2026-10-27 P4-O4-B1a 착수

[기존 B1 수정 지도](./evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)의
세부 baseline에 따라 acquired ZIP immutable bytes/file facts부터 연결한다. Fixed selected pin과
원본 HAA identity를 보존하며 Python surface/typed plan·동적 관찰·release 전체 연결은 남는다.
기존 supply/wheel parser를 재사용하고 M13-004 하나만 IN_PROGRESS로 유지한다.

### 2026-10-27 CI419 최종 결과 확인

[CI419 qualification](./evidence/package-client-quarantine/ci419-pipx-data-hosted-qualification-20261027.json)은
Required 포함10SUCCESS와 전체 checkout05b3/source504e698 tree d2b385/2737 동일을
확인했다. 시작10:39:14UTC→Required11:44:49UTC는65분35초, Linux64분56초다.
원본10logs의 실제 test·helper wait0/최종 confirmed cleanup을 대조하고 전체 readback/hash를
보존했다. 첫10분 조회 뒤 종료한 대화가 지속 자동 감시했다고 표시하지 않는다.
C1/C2 변경의 기존 hosted 회귀만 닫으며 신규 pipx DATA actual은 로컬 범위를 유지한다.
Python ZIP 실행/외부 관찰·release·native/CLI·전체 P4는 남고 같은 item의 B1을 계속한다.
CI415의 역사적 FD 원인은 계속 OPEN이다. 현재 대기 중인 CI는 없다.

### 2026-10-27 CI419 첫10분 결과 조회

실제 타이머10:49:16UTC 발화 뒤10:49:43UTC에 run/jobs를 조회했다. 동일 source504e698의
8jobs SUCCESS와 gVisor Linux Integration IN_PROGRESS이며 Required 최종 결과는 미완료다.
첫 조회의 원본 run/jobs/clock은 `ci419-poll01-*`에 readback/hash로 보존한다.
다음 권장 조회는10:59:43UTC(19:59:43KST), 최근 실측 기반 예상 완료는20:44–20:59KST다.
외부 CI가 남아 현재 checkpoint에서 인계할 수 있으며 응답 종료 뒤 자동 조회·재개는 없다.
다음 재개는 이 exact run의 checkout/tree·최종10 conclusions와 named actual/cleanup을 먼저
확인하고, 같은 M13-004/O4-B1 acquired ZIP surface/typed observation plan으로 이어간다.
B1 audit만 추가했고 CI 실행 중 제품 source를 더 변경하거나 재push하지 않았다.

### 2026-10-27 CI419 공통 DATA volume 회귀 시작

C1/C2와 앞 CI418 결과 기록의3 commits를 묶어 push했다. Source504e698/tree
d2b3853588cfd7a3c1ecaaeddb0b75355f5a8f04/2737files는 검사 사본의 전체 index/blob/mode와
실제 disk가 같다. 최종 staged docs0.254s/security9.310s도 PASS다.
[CI419/run40490143737](https://github.com/rahoney/heliopause/actions/runs/40490143737)은
10:39:14UTC(19:39:14KST)에 시작했고 초기9개 선행 job은 IN_PROGRESS다.
직전 CI41865분29초/CI41776분46초를 근거로65–80분(20:44–20:59KST)을 예상한다.
조기 실패 조회의 실제 타이머를10:49:14UTC(19:49:14KST)에 두었으며 조회 완료와 구분한다.
기존 hosted 소비자 회귀 결과를 기다리는 동안 [B1 입력/실행 owner 대조](./evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)를
계속한다. 신규 actual pipx DATA 시험은 로컬 explicit fixture이며 이번 CI가 Python ZIP
실행이나 전체 pipx 지원을 검증한다고 표시하지 않는다. 작은 local 문서 변경을 재push해
실행 중 후보를 취소하지 않는다.

### 2026-10-27 P4-O4-C2 한정 로컬 검증

[C2 receipt](./evidence/package-client-quarantine/p4o4c2-pipx-readonly-volume-local-20261027.json)는
Quick258.661s/sandbox 전체 Race23.229s/docs/security/workflow PASS와 actual DATA5.020s를
연결한다. Root preparation→ordinary RO anchor/reader, writer 제거 전 seal 거부,
foreign RW attachment 거부와 root write EROFS·동일 ZIP/보호5파일/Docker 정리를 확인했다.
첫 두 readback 실패는 fixture writer UID1000의 실제 tar owner로 좁혀 원본을 보존했다.
별도 format cwd 오류의 raw4개는 역사적 원본 bytes로 복구했다. Python/ZIP 코드 실행은
없고 gVisor/observer·release·native pipx/CLI 및 전체 P4 acceptance는 남는다.
공통 volume 변경의 기존 hosted 소비자 회귀를 다음 묶음으로 확인하고 B1을 준비한다.

### 2026-10-27 P4-O4-C2 착수

[C2 baseline/수정 지도](./evidence/package-client-quarantine/p4o4c2-pipx-readonly-volume-plan-20261027.md)에
따라 기존 closure volume의 identity/attachment owner와 C1 DATA 공급을 연결한다.
Writer 제거와 live readonly anchor/consumer·실제 byte readback·정리를 검증하며
runtime/observer·release·native pipx/CLI 완료는 남는다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-O4-C1 한정 로컬 검증

[C1 receipt](./evidence/package-client-quarantine/p4o4c1-pipx-data-transport-local-20261027.json)는
fixed DATA reader/transport의 Quick230.736s/sandbox 전체 Race25.658s/docs/security PASS와
actual ZIP2505814B의 disposable namespace 보호 파일·15개 거부0.149s/noSKIP·원본 불변을
연결한다. Actual은 DATA tar roundtrip이며 Docker/Python 실행이나 readonly mount 완료가
아니다. 기존 UV/PNPM/Bun을 유지하고 다음 C2 실제 mount lifecycle로 이어간다.

### 2026-10-27 P4-O4-C1 착수

[C1 baseline/수정 지도](./evidence/package-client-quarantine/p4o4c1-pipx-data-transport-plan-20261027.md)에
따라 기존 held file reader/typed transport에 fixed pipx DATA를 연결한다. 기존 Python
preparation/anchor readonly owner는 다음 C2에서 재사용한다. Public runtime/CLI와 release
acceptance는 열려 있으며 M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 CI418 원격 검증 확인

[CI418 qualification](./evidence/package-client-quarantine/ci418-pipx-owner-hosted-qualification-20261027.json)에서
Required 포함10SUCCESS, 전체 checkout4b88/source012c7b4 tree3649f0bd/2726 동일과
원본10 logs readback/hash를 확인했다. Linux79PASS/noSKIP, actual native UV123.40s,
CPU245.96s, helper11wait0/final CONFIRMED_STOPPED·original/client/restore0이다.
Linux elapsed64분57초, workflow 시작→Required65분29초(최종 run 갱신65분30초)다.
이전70–90분 예상보다 빨리 완료됐으며 다음 동일 workflow 예상의 참고 실측으로 남긴다.
아래 시작/권장 조회 기록은 역사이며 현재 대기 중인 run은 없다. 자동10분 감시를 수행했다고
표시하지 않는다. 공통 runtime owner와 기존 hosted 소비자의 회귀 범위만 닫고 O4-B/C의
Python ZIP 검역·배포·readonly transport/native pipx/CLI 및 전체 P4 acceptance는 남긴다.

### 2026-10-27 CI418 공통 runtime owner 검증 시작

O1~O4-A 공급·single owner/schema5·maintenance의 누적6 commits를 milestone branch에
push했다. Source012c7b4/tree3649f0bd71148e402687a72344b42640413211f1/2726files는
검사 사본의 전체 index/blob/mode 및 실제 disk와 같다. 최종 staged docs0.276s/security14.350s/
workflow0.661s/release-gate0.302s도 PASS이며 named local 범위를 넘어 성공을 전용하지 않는다.
[CI418/run40476591687](https://github.com/rahoney/heliopause/actions/runs/40476591687)은
2026-10-27 08:52:39UTC(17:52:39KST)에 시작했다. 첫 조회9jobs 중 Docs/Representative Wheel
Corpus2SUCCESS, 나머지7IN_PROGRESS다. Required는 아직 생성/완료되지 않았다.
직전 CI417의 전체76분46초를 근거로 예상70–90분(19:02–19:22KST)을 기록하며 보장하지 않는다.
조기 실패의 다음 권장 조회는**09:02:39UTC(18:02:39KST)**다.
응답 종료 뒤 자동 조회·재개는 없으며 새 push로 이 후보를 취소하지 않는다. 원본 run/jobs는
local raw owner의 ci418-initial-*에 readback/hash와 함께 보존한다. Whole P4 acceptance는
미완료다. 독립 다음 작업은 O4-B map의 fixed ZIP execution inspection/readonly transport
준비·실행 경계이며 이번 push는 구현된 전체 pipx 지원이나 공개 배포가 아니다.

### 2026-10-27 P4-O4-A 한정 로컬 검증

[O4-A receipt](./evidence/package-client-quarantine/p4o4a-pipx-runtime-owner-local-20261027.json)는
단일 selected owner/frozen maintenance의 실제 공급·apply 중단 복구·변조 거부를 연결한다.
최종 NOTICE 생성기 교정 후보 Quick248.69s/affected Race35.863s/actual4.637s가 PASS다.
이전 후보 scripts 전체 Race119.715s와 canonical 결과는 이전 bytes 범위로 보존했다.
ZIP/NOTICE0444·선택 hash/원본27입력 불변이며 release13assets/2tools는 유지된다.
O4-B active Python ZIP 검사·배포·activation과 O4-C typed transport/actual gVisor·observer·
native/CLI가 남는다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-O4 착수

[O4 baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)에
따라 기존 runtime lock/generator/maintenance/release·installed reader를 재사용한다.
새 pipx 공급 owner와 DATA ZIP role을 연결하며 native/observer·CLI acceptance는 열어 둔다.

### 2026-10-27 P4-O3 한정 로컬 검증

[O3 receipt](./evidence/package-client-quarantine/p4o3-pipx-loader-local-20261027.json)의
Quick221.019s/affected Race37.284s/docs·security와 실제 ZIP 생성·원본190파일 대조,
결정성·기존 출력 보존·frozen27파일 불변을 확인했다. Fixed Python runc guest의
metadata/version/help도 PASS이며 보호5파일/Docker·candidate bytes는 동일하다.
GVisor/observer/native pipx/CLI 검증으로 확대하지 않고 O4 single owner 감사로 이어간다.

### 2026-10-27 P4-O3 착수

[O3 baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p4o3-pipx-loader-plan-20261027.md)에
따라 O2 held bytes와 typed manifest, 기존 no-replace publisher를 재사용한다.
원본 RECORD·license를 보존한 readonly ZIP 후보와 fixed Python isolated loader를 검증한다.
공통 runtime/observer·native install·upgrade/CLI acceptance는 별도 MISSING이다.

### 2026-10-27 P4-O2 한정 로컬 검증

[O2 receipt](./evidence/package-client-quarantine/p4o2-pipx-closure-local-20261027.json)의
Quick232.433s/affected 전체 Race77.646s/docs·security와 actual1.353s가 PASS다.
7 wheels/190파일/제공 proof6과 실제 변조7종 거부·frozen27파일 불변을 확인했다.
공급 검증만 닫으며 같은 M13-004에서 O3 immutable loader baseline으로 이어간다.
Runtime/native/CLI/hosted·전체 P4 acceptance는 미완료이고 새 push/full CI는 없다.

### 2026-10-27 P4-O2 착수

[O2 baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p4o2-pipx-closure-plan-20261027.md)에
따라 O1 private 검증 로직과 기존 whole wheel/metadata/marker/constraint/graph를 재사용한다.
제공 dependency proof5개의 fixed publisher 검증과 실제7-wheel closure를 연결하며 userpath
서명 미제공 경계를 유지한다. DATA-only whole RECORD 진단0.582s는 PASS였고 formal 공급·
immutable loader/runtime·native·CLI acceptance는 미완료다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-O1 한정 로컬 검증

[O1 receipt](./evidence/package-client-quarantine/p4o1-pipx-publish-local-20261027.json)의
canonical Quick180.173s·두 affected package 전체 Race30.077s·actual supply1.680s·
docs0.420s/security18.779s가 PASS다. Fixed pipx publisher/certificate environment·subject와
single wheel 실제 bytes를 검증하고 frozen 입력 불변을 확인했다. Initial formatter 순서와
SA1012 시험 표현 실패는 원본과 구분해 보존했다. O1 developer 공급 사실 검증만 닫으며
dependency closure/immutable loader/runtime·native·CLI/hosted/final matrix는 미완료다.
같은 M13-004에서 O2 edit map으로 이어가고 새 push/full CI는 없다.

### 2026-10-27 P4-O1 착수

[O1 baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p4o1-pipx-publish-plan-20261027.md)에
따라 기존 pinned Sigstore/root와 developer frozen reader를 재사용한다. 별도 fixed pipx
publisher의 bounded PEP740/certificate environment·subject/단일 wheel bytes 검증만 추가한다.
전체 dependency closure/immutable loader/runtime·native/CLI·P4 acceptance는 후속 missing이다.
M13-004 하나만 IN_PROGRESS이며 작은 helper별 새 push/full CI는 하지 않는다.

### 2026-10-27 CI417 최종 검증

[CI417 qualification](./evidence/package-client-quarantine/ci417-uv-tools-hosted-qualification-20261027.json)을
확인했다. Required 포함10SUCCESS이며 source518800c와 all10 checkout1fab621의 전체
똑같은 tree df689208/2660을 대조했다. Linux79PASS/noSKIP, 두 observer build의122latches,
실제 UID1001/native current138.06s의 원본/현재 승인/Manifest10·managed 재승인8,
기존 tool exact5·CPU282.58s·native macOS와 helper11wait0/final original0을 확인했다.
Raw decoded10logs1200139B를 편집 없이 보존하고 모두 readback/SHA를 대조했다.

CI416 fixture UID 문제와 supported UV tools의 named hosted gap은 닫는다. P4/M13-004
전체 acceptance, pipx·ordinary native default recipe·최종 profile matrix는 남는다. CI415의
FD 원인은 peer PASS만으로 닫지 않는다. 같은 item의 다음은 기존 P4-O baseline에서 확인한
공급/provenance/immutable loader·runtime 경계의 concrete edit map/두 번째 검토다.
현재 결과 반영은 문서만 변경하고 새 push/full CI를 만들지 않는다.

### 2026-10-27 CI417 교정 후보 인계

CI416의 tool fixture UID 고정을 ordinary1001에서 RED1.668s→GREEN1.818s로 대조했고,
Quick281.578s/Bootstrap Race17.703s·native current actual168.356s/affected default275.437s,
최종 staged docs/security·전체2660파일 index/disk/mode 일치를 확인해518800c로 보존·push했다.
제품 source/runtime/observer·Policy/resource budget은 바꾸지 않았다. Named local 결과와
corrected hosted acceptance는 구분한다.

CI417/run40449588924는 source518800c/tree df689208이며04:24:47UTC 시작했다. 첫 실제10분
query를04:35:03UTC에 소비했고8SUCCESS/Linux IN_PROGRESS, Required 결론은 아직 없다.
다음 권장 확인은04:45:03UTC(13:45:03KST)이다. 대화 종료 뒤 자동 재개를 약속하지 않는다.
예상70–90분과10분 early-failure 감시는 별개이며 원격 결과/whole checkout/decoded logs 전에는
hosted 해결을 선언하지 않는다. 독립 P4-O의 제공 proof5개와 wheel digest·module census 추가
진단은 [기존 baseline](./evidence/package-client-quarantine/p4o-pipx-supply-runtime-baseline-20261027.md)에
기록했으며 runtime 공급·native pipx 지원은 계속 미완료다.

### 2026-10-27 CI416 실패 소비

[실패 조사 지도](./evidence/package-client-quarantine/ci416-native-fixture-uid-investigation-20261027.md)는
TestLinuxUVToolNativeAdoptionCLIIntegration의 원본 producer EACCES/0.77s를 소유한다.
기존 tool 네 소비자와 PNPM peer는 PASS이고 native current inspection은 NOT_REACHED다.
8SUCCESS/Linux·Required FAILURE, helperwait0/final CONFIRMED_STOPPED·RESTORED/primary1과
all10 같은 checkout/raw logs를 보존했다. Tool-only bind root와 고정 guest1000 관계를
non1000 ordinary user에서 재현하며 product guard·권한·예산은 바꾸지 않는다. CI 재실행은
원인/후보 교정·로컬 검증 뒤 실제 hosted acceptance 경계에서만 수행한다.

### 2026-10-27 P4-N2D 실제 native 인계

[N2D receipt](./evidence/package-client-quarantine/p4n2d-uv-native-tool-cli-local-20261027.json)는
명시 --adopt와 genuine native exact6.0·bounded6.1→6.0의 current 전체 검사/새 Evidence/
Policy ALLOW·Manifest10/원자 반영, managed 반복 adopt의 새 Evidence/Manifest8을 연결한다.
현재 Evidence 권한 실패와 원본 변조/no-adopt는 ledger 부재·원본 identity/bytes/modes/links를
보존한다. 같은 binary의 기존 native project/complete sync도 PASS다. 두 실제 실행은
noSKIP/exact named tests/helperwait0/CONFIRMED_STOPPED·보호5파일/Docker 동일이며
고정 recipe의 named local 범위만 검증한다. Final Quick62.462s/full affected Race183.469s와
canonical workflow/docs/security도 PASS이며 hosted/pipx/
ordinary native default recipe/final profiles는 남는다. 기존 N2C의 native actual gap은 이
후속 범위에서만 닫는다. N2D1b16544의 전체tree616b506b/2657을 확인하고 누적6 commits를
기존 milestone branch에 묶어 push했다. CI416/run40441186696은02:55UTC 시작했으며
실제10분 상태 조회의 첫 결과를03:05:29UTC에 소비했다. 8SUCCESS/Linux integration
IN_PROGRESS이며 Required 결론은 아직 없다. 예상70–90분과 early-failure 감시는 구분한다.
다음 권장 상태 확인은03:15:29UTC(12:15:29KST)이며 종료 뒤 자동 재개를 약속하지 않는다.
독립 후속은 [P4-O pipx baseline](./evidence/package-client-quarantine/p4o-pipx-supply-runtime-baseline-20261027.md)의
공급 증거·runtime 역할 감사이며 M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-N2C 정전 후 로컬 체크포인트

[N2C receipt](./evidence/package-client-quarantine/p4n2c-uv-native-current-update-local-20261027.json)는
단독 Quick208.494s/affected full Race163.552s와 security/docs·canonical observer122latches,
기존 UV tool 네 actual 소비자376.837s를 연결한다. 명시 native enrollment와 raw/source controls,
synthetic 원본의 ledger 부재/원복·취소/late foreign cleanup, managed 반복 adopt의 현재 재검사를
확인했다. Helperwait0/Host 보호 파일·Docker 동일이며 정전 후 사라진 두 /run directory만 별도
복구했다. 최초 시험 준비 오류와 중첩 실행의 두300s timeout은 원본으로 보존한다. 실제 PUBLIC
NATIVE 원본의 current 승인/Manifest/CLI는 아직 미검증이며 [N2D 지도](./evidence/package-client-quarantine/p4n2d-uv-native-tool-cli-plan-20261027.md)와
그 실제 소비자로 이어간다. M13-004 하나만 IN_PROGRESS이고 새 push/full CI는 없다.

### 2026-10-24 CI402 ordinary checkpoint 확인

실제55분 타이머 만료 뒤 [CI402 결과](./evidence/package-client-quarantine/p2c2-pnpm-ci402-qualification-20261024.json)를
확인했다. Required 포함10SUCCESS이며 source940과 checkoutcab8183의 전체tree36c5d030이
같다. PNPM 실제4개·두 job의 observer110·Linux job 전체57 top-level PASS/CPU294.69s·
native macOS, helper5wait0/final CONFIRMED_STOPPED·original0/client_error0/client_restore0도
확인했다. 이 ordinary checkpoint는 앞940 범위이며 새 public8 fixture의 미실행을 성공으로
표시하거나 이후 로컬0f8/40d/C5 검증으로 전용하지 않는다. P2 qualification/acceptance는
여전히 미완료이고 native offline dependency/peer context를 같은 M13-004에서 이어간다.

### 2026-10-24 P2c3 archive 설치 동작 사실 전달 착수

[착수 감사·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2c3-pnpm-install-action-plan-20261024.md)에서
기존 npm archive/graph M4/정규화 Evidence를 재사용하고 rootless npm의 사실 전달 gap만
확인했다. 단일/acquired/batch 세 경로와 subject/check/저장 참조 결합, UNKNOWN·필요 동작의
수동 검토가 이번 범위다. Layout/transaction/CLI·전체 P2 qualification은 계속 미완료다.
앞 네 commit은 한 번 push했고 CI402/head940의55분 실제 타이머 첫 확인은14:57:13KST다.
현재 source와 다음 local 구현을 섞어 성공을 주장하거나 재push로 실행을 취소하지 않는다.
M13-004 하나만 IN_PROGRESS다.

후속 구현은 single/acquired/batch와 exact subject/check/reference 결합을 연결했다.
최종 Quick33.846s와 앞 관련 여섯 package Race5.283s, 실제 public archive 두 개의
SRI/static/Store/true·false fact0.936s가 PASS다. 새 UNCONFIRMED code는 기존 다국어
필수 검사 미완료 설명을 재사용한다. [로컬 결과](./evidence/package-client-quarantine/p2c3-pnpm-install-action-local-20261024.json)가
현재 입력과 범위를 소유한다. Remote402는 앞940 후보이므로 이번 source 성공으로 전용하지 않는다.
다음은 unique archive coverage와 full peer wiring을 함께 묶는 complete snapshot이며
전체 P2/Host 설치·CLI qualification은 계속 열려 있다.

P2c3은 로컬0f8ddcc로 보존했고 push하지 않았다. [P2c4 수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2c4-pnpm-project-snapshot-plan-20261024.md)를
완료하여 기존 complete Domain과 parser/source binding을 재사용하는 snapshot 생산을 이어간다.
Full peer instance·runtime/target/controls 결합과 explicit empty가 범위이며 native layout/
records의 peer provenance·transaction/CLI는 후속 acceptance로 유지한다.

후속 actual에서 native single/transitive의 snapshot과 single required3/refs4·M3/M4 ALLOW는
확인했다. Transitive is-odd3은 integrity/static/action fact 정상이나 기존 single-artifact
offline dynamic에서 M3_DYNAMIC_EXECUTION_FAILED/INCOMPLETE여서 수동 검토다.
[P2c4 실제 결과·진단 범위](./evidence/package-client-quarantine/p2c4-pnpm-project-snapshot-plan-20261024.md#5-실제-연결의-첫-거부와-남은-gap)에
Host불변/정리·미실행 후속과 stale overlay 진단 NOT_EXECUTED를 보존했다.
다음에는 npm에 존재하지 않는 frozen dependency/peer fixture의 정확한 연결 지도를 먼저
작성·재검토하고 이 gap을 닫은 뒤 layout/transaction/CLI로 진행한다. M13-004만 IN_PROGRESS다.

Snapshot은 로컬40d4e25로 보존했다. [P2c5-A native offline 착수 감사·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2c5-pnpm-offline-context-plan-20261024.md)를
완료했고 먼저 fixed file-source recipe와 실제 pinned native 배치를 검증한다. 이것은 test
prerequisite이며 production closure 인터페이스는 그 결과 뒤 정확히 확정한다. 현재 작은
local commit 둘은 push하지 않았고402 후보도 유지한다.

P2c5-A2의 transitive2는 [로컬 결과](./evidence/package-client-quarantine/p2c5-pnpm-offline-local-20261024.json)에서
원래 observer25의 actual8.591s/complete/금지0·Host불변/helperwait0로 검증했다. 초기 store
설정을 고정해 default root probe를 피했고 제품 관찰기·권한은 유지했다. Peer fixture와
production static-all/per-target closure가 다음 순서이며 새 실제 P2 검역 완료는 아니다.
완료한 namespace·진단 자료는 참조/query error0 확인 뒤 삭제했고 작은 원본만 보존했다.

이어 A3 [peer 실제 결과](./evidence/package-client-quarantine/p2c5-pnpm-peer-offline-local-20261024.json)에서
기존 exact-version source API·versioned fixture를 재사용한 transitive6.01s/peer8.01s와
peer4 identity·links5개/complete/금지0·Host불변/helperwait0를 확인했다. 일반 packument
한도에서 멈춘 첫 준비 실패는 원본에 남겼다. Production closure는 아직 미구현이며 B의
상세 수정 지도·두 번째 검토를 다음으로 진행한다. M13-004 하나만 IN_PROGRESS다.

[B의 착수 감사·B1 정확한 수정 지도/두 번째 검토](./evidence/package-client-quarantine/p2c5b-pnpm-production-context-plan-20261024.md)를
작성했다. 기존 snapshot/source binding/project batch를 재사용하고 frozen snapshot과 acquired
전체 집합·복수 peer instance를 잃지 않는 immutable 값을 먼저 연결한다. 이후 실제 lifecycle
선택·격리·관찰 aggregate의 의미를 검증하며 미확정 runner 인터페이스를 앞서 만들지 않는다.

B1 [로컬 결합 결과](./evidence/package-client-quarantine/p2c5b-pnpm-context-local-20261024.json)는
same-digest 위조 snapshot·획득 누락/혼입과 다른 run 거부, six archives/seven peer instances
보존·실제 producer wrapper 재사용을 확인했다. Canonical Quick/security/docs·관련 전체 Race는
PASS다. 기존 단일 npm 실행/관찰기/Policy는 불변이며 실제 PNPM dynamic은 계속 미완료다.

위 다섯 로컬 commit을 db464fd까지 한 번 push했다. Ordinary CI403/run40075179512가
시작됐으며 직전402 실측53m11에 따라55분 실제 타이머의 첫 확인은17:37:12KST다.
그 전에는 반복 조회·작은 변경 재push를 하지 않고 B2 설계/로컬 구현을 이어간다.

[B2 정확한 수정 지도·검토·로컬 결과](./evidence/package-client-quarantine/p2c5b2-pnpm-target-lifecycle-plan-20261024.md)는
기존 install 권한을 유지한 별도 target/topology·실제 clone lineage·one-shot과 C++112,
transitive/peer의 synthetic positive/exit37·기존8 actual 회귀를 확인했다. Production layout/
static-all/aggregate와 공개 프로젝트 검역 연결은 B3에서 이어가며 이 primitive를 ALLOW로
표시하지 않는다. B2는 아직 원격403 source에 포함되지 않았다. 같은 M13-004만 IN_PROGRESS다.

### 2026-10-24 P2c5-B3 archive/layout 착수 감사

[B3 착수 감사·B3a 수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2c5b3-pnpm-layout-plan-20261024.md)를
완료했다. 기존 정적 tar 순회에 선택적 per-file inventory를 더하고 기존 Inspect 결과는
보존한다. 최종 Quick/security/docs·세 package 전체 Race26.274s, 실제 public
is-number4/node-sass334 entries의 Acquire/SRI·원래 report/Evidence 일치도 PASS다. 전체 native layout/prepare/static-all/aggregate는 순차 후속이며 current product의
transitive INCOMPLETE와 전체 P2 acceptance NO를 유지한다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 P2c5-B3b native 배치 형태와 dirname 연결

[B3의 §6·§7](./evidence/package-client-quarantine/p2c5b3-pnpm-layout-plan-20261024.md#6-b3b0--전체-layout의-실제-표현-확인-source-작성-전-감사두-번째-검토)에서
정상 native의 identical LICENSE hardlink·private hoist/생성 Node bin shim을 실제 확인했다.
해시 없는 일반 hardlink 허용이나 bin 파일명만의 승인은 하지 않는다. B3b1의 official registry
codec/관련 Race5.807s와 fresh actual20.659s의 transitive2/peer4 모든 instance 경로 비교도
PASS다. 두 전용 runtime bulk는 모두 refs/query errors0 뒤 삭제했다. Full layout byte/edge/
bin/metadata verifier·production prepare/static-all/aggregate가 다음이며 P2 acceptance는 NO다.
M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 B3b2a layout envelope 로컬 검증

[B3 §8](./evidence/package-client-quarantine/p2c5b3-pnpm-layout-plan-20261024.md)은 bounded native
입력 parser만 연결했다. Deprecated Xattrs/무제한 Copy 표기의 두 정적 점검 거부를
PAXRecords/정확한 CopyN으로 교정했으며 security/Quick/docs·npm 전체 Race6.037s와
actual21.319s의 original controls/identity/links·complete/금지0/Host불변/helperwait0를
확인했다. 끝난 전용165,567,367B는 refs/query error0 뒤 삭제했다. 이 type에는 acquisition/
Policy/실행 권한이 없고 전체 archive/edge/bin/metadata 검증은 남아 있다. 현재404 source는
앞1f410fd이고55분 타이머19:30:18KST를 유지한다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 B3b2b Node bin code 로컬 검증

[B3 §9](./evidence/package-client-quarantine/p2c5b3-pnpm-layout-plan-20261024.md)은 upstream
전체 코드 형식을 한 owner에서 제한했다. Canonical format/security/Quick/docs·npm 전체
Race6.168s와 actual18.382s의 세 Node shim/명시적 target·원래 관찰/Host보존/helperwait0가
PASS다. 전용165,592,448B는 refs/query error0 뒤 삭제했다. 원본 archive의 bin/file/전체edge와
metadata binding·production prepare/static-all/aggregate는 남아 있으며 전체 P2 acceptance는
NO다. CI404 source는 앞1f410fd로 유지하고 새 push 없이19:30:18KST 타이머로 확인한다.

### 2026-10-24 B3c 전체 inspection layout 결합 착수 감사

[정확한 수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2c5b3c-pnpm-layout-binding-plan-20261024.md)를
작성했다. 기존 inventory/envelope/dirname/Node code 형식과 B1 input을 한 지점에서 결합한다.
모든 peer instance·원본 file/edge/bin·생성 metadata를 확인하며 inspection 값은 Policy/Host
승격 권한이 아니다. 기존 parser/단일 Backend를 다시 구현하지 않는다. 이후 native prepare/
static-all/per-instance aggregate와 Host/CLI는 그대로 MISSING이며 M13-004 하나만 IN_PROGRESS다.

후속 [로컬 결과](./evidence/package-client-quarantine/p2c5b3c-pnpm-layout-binding-local-20261024.json)의
fresh actual17.553s는 transitive2/peer4 archives와 모든 instance/file/edge·원본 bin3개를
검증하고 cloned native input 변조7/8개를 거부했다. 처음 누락된 native freshness cache는
pinned source를 확인해 정확한 regular bounded JSON accounting 후 inspection payload/
getters/digest에서 폐기했다. 첫 실패와 교정 source를 구분하고 original controls·complete
관찰/금지0·Host불변/helperwait0를 보존했다. 끝난 두 runtime 총331,938,972B는 refs/query
errors0 뒤 삭제했다. Production native prepare/static-all/fresh per-instance 실행/aggregate와
Host/CLI는 아직 미완료이며, 다음 B4의 정확한 edit map부터 같은 M13-004에서 이어간다.

### 2026-10-24 B4 production inspection 착수 감사

[Baseline audit·정확한 edit map·두 번째 검토](./evidence/package-client-quarantine/p2c5b4-pnpm-production-inspection-plan-20261024.md)를
작성했다. 먼저 기존 PNPMResolver의 수명 owner를 선택/준비가 재사용하고 complete input/
logical footprint·재소개 bytes·network detach/실제 endpoint0·관찰/정리 뒤 값 반환을 연결한다.
이어 verified payload의 최소 소개·fresh per-instance 실제 결과·partial 관찰/첫 오류 보존을
연결한다. 단일 npm·Domain/Application의 기존 의미는 유지하며 이 문서만으로 새 구현/qualification을
주장하지 않는다. M13-004 하나만 IN_PROGRESS이고 PNPM product dynamic/전체 P2는 미완료다.

후속 B4a의 [로컬 결과](./evidence/package-client-quarantine/p2c5b4a-pnpm-production-prepare-local-20261024.json)는
production prepare의 transitive2/peer4와 정리 응답 실패 시 bound 값 폐기·원래 cause 보존을
actual28.999s로 확인했다. 기존 선택5개 회귀28.48s·최종 Quick/security/두 package 전체
Race26.466s도 PASS다. Raw0777 directory 관측은 unverified entry view에서 write bits022만
제거해 소개할 값에 결합하고 원래 file/link/mode guard는 유지했다. 끝난 시험33files/
498,955,765B는 refs/query errors0 뒤 삭제했다. 이제 B4b의 검증 payload/fresh instance 실행과
B4c static-all 수집 gate·실제 관찰 aggregate를 잇는다. Product dynamic과 전체 P2는 계속 미완료다.

Payload 소개 전 검토에서 내부 layout fingerprint가 private Domain.ContentDigest를 직접 JSON
marshal해 entry digest를{}로 생략하는 gap을 확인했다. 같은 길이의 두 유효한 Node search
order가 같은 key를 내는 직접 RED2.358s를 재현했고 string entry projection/v2의 GREEN3.451s·
canonical Quick/security/전체 npm/sandbox Race24.805s가 PASS다. 원래 파일·code/path 검증과
native 동작은 유지하며 [직렬화 교정](./evidence/package-client-quarantine/p2c5b4b-pnpm-fingerprint-local-20261024.json)을
새 Artifact 실행 완료로 확대하지 않는다. 다음은 예정대로 verified payload와 fresh instance backend다.

### 2026-10-24 B4b1 실행용 payload 재구성 착수 감사

[Baseline audit·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2c5b4b1-pnpm-inspection-payload-plan-20261024.md)에서
기존 bound layout/private manifest와 bounded parser를 재사용한다. Raw native whole tar/
원본 user controls를 소개하지 않고 불필요 metadata를 제외하며 file bytes·link·공유 inode
partition을 보존한 새 deterministic tar와 project 전체 readback을 구현한다. Node_modules와
manifest만 검사해 root extra 설정 파일을 놓치지 않는다. 이어지는 fresh backend가 actual
consumer이며 이번 renderer만으로 Artifact 실행/Host/CLI·전체 P2 완료를 표시하지 않는다.

후속 source4개는 작성·format 확인했지만 Quick의 production build 두 번과 별도 focused
command가 각각 timeout이었다. Unit 성공이나 source finding은 확인되지 않았다. Host에서
codex PID1404의 RSS22,799,860KiB·MemAvailable534,648KiB·swap free0을 관측했으나 timeout의
단독 원인으로 확정하지 않는다. 반복 build는 중단하고 미검증 WIP와 [정확한 재개 위치](./evidence/package-client-quarantine/p2c5b4b1-pnpm-inspection-payload-plan-20261024.md#6-2026-10-24-host-자원-차단과-재시작-인계)를
보존한다. Host 자원을 회복한 뒤 같은 고정 검사·한도로 확인해야 해제할 수 있다.

2026-10-25 재개 후 [B4b1 로컬 결과](./evidence/package-client-quarantine/p2c5b4b1-pnpm-inspection-payload-local-20261025.json)는
동일한 보존 source의 canonical format/Quick/security/docs와 npm/sandbox 전체 Race25.352s를
확인했다. Minimal payload·전체 readback과 공유 inode 의미를 검증했으며 production backend는
다음 B4b2다. 과거 timeout/RSS 원인 미확정은 보존하고 전체 P2 acceptance NO를 유지한다.

[B4b2의 착수 감사·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2c5b4b2-pnpm-instance-backend-plan-20261025.md)에
따라 기존 fresh volume/target command/observer·partial 의심 사실 보존 helper를 연결했다.
[B4b2 로컬 결과](./evidence/package-client-quarantine/p2c5b4b2-pnpm-instance-backend-local-20261025.json)는
canonical Go/Race·C++112와 실제 public transitive2/peer4·controlled 정상/exit37, 별도 긴 이름
실행까지 PASS다. 각 실행의 complete trace·최초 실패와 사실 보존·helperwait0/정리·Host 보존을
확인했다. Named local consumer 범위만 닫고 B4c static-all/aggregate부터 이어간다. 전체 P2
acceptance와 Host/CLI·remote qualification은 NO다.

[B4c 착수 감사·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2c5b4c-pnpm-project-inspection-plan-20261025.md)를
기록했다. 기존 complete workspace gate를 같은 charge owner의 incremental counter로 확장하고,
이후 static-all/실제 instance별 Evidence와 오류 시 부분 사실 기록을 연결한다. Core의 미완료 check에
final Finding을 붙이지 않으며 관찰 해석 완료와 required lifecycle coverage를 별도로 보존한다.

[B4c 로컬 결과](./evidence/package-client-quarantine/p2c5b4c-pnpm-project-inspection-local-20261025.json)는
수집 중 고정 예산·전체 frozen layout 결합·실제 Session coverage/partial facts와 실패 시 exact
Evidence 기록을 연결했다. 최종 관련 네 package Race32.714s·canonical 검사/CI 호출 순서와
public transitive2/peer4 실제49.087s, native8 선택·네 실제 Policy/Evidence 검역114.917s가 PASS다.
각 실제 수명의 helperwait0/정리·Host 보존을 확인했고 Host 설치는 미수행이다. B4 production
inspection의 named local scope만 닫고 묶음 remote와 PNPM cache/provenance/Host records·
transaction/CLI를 이어간다. 같은 M13-004만 IN_PROGRESS이며 whole P2 acceptance는 NO다.

[B5a baseline audit·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2c5b5a-pnpm-approved-staging-plan-20261025.md)를
기록했다. 기존 stored Evidence reader와 source 재해시·sealed regular-file cache를 재사용해
승인된 PNPM layout을 비실행 data tar로 stage하는 gap부터 진행한다. Symlink Host 추출과
독립 retained approval/records·transaction·CLI는 다음 경계다. CI405의 frozen source는 유지한다.

[B5a 로컬 결과](./evidence/package-client-quarantine/p2c5b5a-pnpm-approved-staging-local-20261025.json)는
source 전체 재해시·실제 stored Evidence/expected instance·runtime/layout 결합과 비실행 data
cache Stage/Open을 확인했다. Canonical checks·관련 six-package Race148.769s/추가9.823s,
실제 public peer4/Evidence24와 explicit-empty data cache43.959s·helperwait0/Host 보존이 PASS다.
Empty의 로그상 고정 true는 실제 coverage가 아니므로 scope correction과 conditional 표시를
남겼다. B5a named local만 닫고 Host typed extraction·독립 retained approval/Manifest·SBOM·
transaction·CLI는 계속 MISSING이다. CI405 결과를 기다리며 이 source는 아직 push하지 않는다.

### 2026-10-25 B3 후속·B4 production inspection 묶음 원격 qualification

최종 B4c7e966e7/tree d864b870·2,232개 전체 후보의 index/blob/disk/mode 일치를 확인하고
로컬11개 commit을 기존 branch/Draft PR38에 한 번 push했다. CI405/run40112425913은
2026-10-25 01:37:37 KST 시작·ordinary IN_PROGRESS다. 직전 CI40453m16을 근거로55분
실제 타이머를 설정했고 첫 확인은02:32:37 KST다. 원본은 p2c5b4-ci405-initial-api와
p2c5b4-ci405-timer-20261025.json이다. 그 전에는 반복 조회/재push하지 않는다. Required·
실제 checkout·기존/new 소비자 결과 확인 전에는 remote qualification을 닫지 않는다.
같은 item의 다음 PNPM cache/provenance/Host records·transaction·CLI baseline audit를
이어가며 M13-004 하나만 IN_PROGRESS다. Whole P2 acceptance는 계속 NO다.

### 2026-10-24 CI404 묶음 checkpoint 확인

실제55분 타이머 만료 뒤 [CI404 결과](./evidence/package-client-quarantine/p2c5b3-pnpm-ci404-qualification-20261024.json)를
확인했다. Required 포함10SUCCESS, 모든 job의 checkout31290a9/source1f410fd 전체tree32123575/
2,190 files 일치다. 두 job의 observer112/0·PNPM 실제4·Linux job 전체57 top-level PASS,
CPU full281.30s·native Go1.27.3 macOS, helper5wait0/final original0/confirmed cleanup도
확인했다. 실제 elapsed53m16이다. Gated target/shape 검사의 별도 local 범위나 이후 두
local commit5641ed6/442af68을 이 remote 성공으로 전용하지 않는다. 전체 P2 acceptance는
계속 NO이며 전체 archive/edge/bin/metadata binding과 production prepare를 이어간다.
아래 PENDING/타이머 시작 문단은 만료 전 이력이다. 새 timer/반복 원격 조회는 없다.

### 2026-10-24 B2·B3a·B3b1 묶음 원격 qualification 시작

로컬에서 검증한 세 commit83db86c/7ff0cab/1f410fd를 한 번 push했다. 최종 workflow.837s도
PASS이며 CI404/run40082548666은 source1f410fd/tree32123575의 ordinary 후보다.
최근402/403 실측53m11/51m17을 근거로55분 실제 타이머를 설정했고 첫 확인은
2026-10-24 19:30:18KST(10:30:18UTC)다. 그 전에는 API 반복 조회·작은 변경 재push를 하지
않는다. B2 공용 observer의 remote qualification은 PENDING이며 그 결과에 의존하지 않는
B3b2 generated shim/hoist/metadata·전체 layout의 상세 지도와 로컬 작업을 계속한다.
M13-004 하나만 IN_PROGRESS이고 전체 P2 acceptance는 NO다.

### 2026-10-24 CI403 묶음 checkpoint 확인

실제55분 타이머08:37:12UTC 만료 뒤 [CI403 결과](./evidence/package-client-quarantine/p2c5b-pnpm-ci403-qualification-20261024.json)를
확인했다. Source db464fd와 checkoutd5be7ee의 전체treeced85a91/2,173 files가 같고
Required 포함10SUCCESS다. Observer110씩·PNPM 실제4·Linux job 전체57 top-level/
CPU286.42s·native Go1.27.3 macOS와 helper5wait0/final original0 정리도 확인했다.
실제 elapsed51m17이며 B2의 새 observer112/target primitive 검증으로 전용하지 않는다.
다음 연결 단위를 로컬에서 계속 작업하고 작은 helper별 remote 재push는 하지 않는다.

### 2026-10-24 P2c2 actual public 선택·update/no-op 로컬 검증

[실제 public 결과](./evidence/package-client-quarantine/p2c2-pnpm-public-selection-local-20261024.json)는
native single/transitive/peer/add, frozen install, selected/full update와 no-op8개를
actual46.82s에서 독립 registry binding과 함께 확인했다. Range를 보존한6→7과 frozen/
no-op의 두 control bytes 동일도 PASS다. C++25의110 matrix와 운영 보호12파일/images/
networks/bridge/service/firewall 동일·canonical helper wait0/CONFIRMED_STOPPED를 확인했다.
Native client 자체 notifier는 false이며 실제 public이 요구한 다섯 read만 exact PNPM
CONTROL root/flags/mount에 한정했다. Archives/Policy·Domain graph/HostInstallAction·
layout/transaction/CLI와 현재 후보 remote/whole P2 qualification은 미완료다.
현재 checkpoint를 검증한 뒤 Domain·archive lifecycle 사실 전달의 실제 gap을 감사한다.
M13-004만 IN_PROGRESS다.

### 2026-10-24 P2c2 native producer 경계 로컬 검증

[Producer 경계 결과](./evidence/package-client-quarantine/p2c2-pnpm-producer-boundary-local-20261024.json)는
add/install/update 사전 입력·허용된 manifest 변경, 기존 UV/PNPM exact owner 공유와
고정 lock-only command·전체 관찰/정리 후 결과 반환을 연결한다. Canonical Quick51.370s/
security/docs와 실제 empty/fault5cases31.01s가 PASS다. 앞 native notifier의 불필요한
registry 조회를 공식 source와 대조하여 false로 고정했고 임시 읽기 보완은 되돌렸다.
Observer product/binary는 qualified17/CI401과 같고 추가 거부 matrix110도 PASS다.
이는 fake/inert service의 closed-network 경계 검증이다. Actual public typed policy/HTTPS
single/transitive/peer·independent binding이 다음이며 Domain/HostInstallAction·layout/
transaction/CLI acceptance는 미완료다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 CI401 ordinary native admission 확인과 P2c2 전송 검증

실제 연장 타이머 뒤 [CI401 결과](./evidence/package-client-quarantine/p2b2-pnpm-ci401-qualification-20261024.json)를 확인했다.
Head343f03f와 checkout093cba1의 전체tree e735b1ec가 같으며 Required 포함10SUCCESS다.
새 PNPM actual4·observer110·기존 Linux47/CPU296.32s·native macOS와 helper5wait0/
최종 CONFIRMED_STOPPED·client_error0·restore0을 확인했다. P2b2 ordinary guest admission의
원격 gap만 닫는다. P2c1/e762와 이후 source는 로컬이며 native5 signed whole-update 및
P2 graph/layout/transaction/CLI acceptance는 계속 열려 있다.
P2c2 고정 project controls의 local Quick119.431s/security/docs PASS 뒤 같은 production
transport를 소비하는 empty-lock/Node actual과 manifest·lock의 metadata/크기를 검증한다.
새 fresh owned workroot의 actual8.884s도 PASS이며 [전송 결과](./evidence/package-client-quarantine/p2c2-pnpm-control-transport-local-20261024.json)로 범위를 연결했다.
마지막 소비 뒤 참조/query errors0 확인 후1,278,661,055B를 제거했다.
M13-004만 IN_PROGRESS이며 작은 helper별 원격 CI를 반복하지 않는다.

### 2026-10-24 CI401 첫 타이머 확인과 P2c2 준비

50분 실제 타이머가03:23:29.834UTC에 만료된 뒤 exact head343f03f의 run/jobs를 조회했다.
Quick/observer/macOS/docs/security/vulnerability/corpus/minimum Go8개가 SUCCESS이고
Linux integration/job120754864311은 아직 IN_PROGRESS다. Required는 Linux 결과를 기다린다.
15분 연장 타이머의 다음 확인은 **2026-10-24 12:40:19 KST**다. 재push/재dispatch하지 않는다.
P2c1은 전체2138files/tree 대조와 evidence-inclusive security/docs 뒤 로컬 `e762bdc`로
커밋했다. 원격에는343만 있으며 P2c1의 원격 qualification을 주장하지 않는다.
[P2c 지도 §6](./evidence/package-client-quarantine/p2c-pnpm-selection-plan-20261024.md#6-다음-p2c2-producer의-source-작성-전-수정-위치)의
actual producer/preflight/typed network/정리와 Domain·HostInstallAction 남은 경계를
확인하며 다음 source를 준비한다. M13-004만 IN_PROGRESS다.

### 2026-10-24 P2c1 bounded lock·독립 source binding 로컬 검증

[수정 지도와 결과](./evidence/package-client-quarantine/p2c-pnpm-selection-plan-20261024.md)는
복수 root/peer instance 보존·strict bounded YAML·공개 source SRI/URL 대조를 연결한다.
Final Quick116.395s/security/docs/vulnerability와 관련 전체 Race는 PASS다. 실제 public
metadata single/peer1.623s도 PASS이며, packument 과대/버전 endpoint Accept406의 원본과
기존 크기·timeout·origin 제한을 보존했다. 생성 pin의 version은 producer가 전달하고
새 YAML 외부 import는 검토한 npm adapter에만 허용한다. Native public selection이나
archive quarantine/Policy·Domain graph/HostInstallAction·layout/transaction/CLI 완료는
아니다. 다음 P2c2 public producer를 정확한 owner/실행·정리 경계로 연결한다.
CI401/head343f03f의 원격 결과와12:23:29KST 첫 확인 타이머는 계속 유지한다.

### 2026-10-24 P2c1 독립 selection 착수와 CI401 타이머

Native admission의 exact35경로/전체2128files·tree를 대조해 `343f03f`로 커밋하고 기존
PR38 브랜치에 한 번 push했다. CI401/run40054664786·head343f03f는02:33:29UTC에
등록됐다. 최근 ordinary397의46m38s를 근거로50분 타이머를 실행했고 첫 확인은
**2026-10-24 12:23:29 KST**다. 조회를 짧게 반복하거나 같은 CI를 재dispatch하지 않는다.
대기 중 [P2c 수정 지도](./evidence/package-client-quarantine/p2c-pnpm-selection-plan-20261024.md)의
기존 parser/graph/transaction 감사와 공식 exact source·실제 형식 대조/두 번째 검토를
먼저 완료했다. 현재 독립 P2c1 bounded unapproved lock parser부터 진행하며 hosted
qualification과 전체 PNPM 지원/CLI 완료는 계속 열려 있다. M13-004만 IN_PROGRESS다.

### 2026-10-24 P2b2 native admission과 실제 CI helper 로컬 검증

[수정 지도](./evidence/package-client-quarantine/p2b2-pnpm-admission-plan-20261024.md)의
exact PNPM Root/Node query demotion과 승인된 GHI own maps·실제 관측 VM 한 경로를 연결했다.
보완17 C++110 latch40.735s PASS 및 실제 PNPM4/UV2/GHI1 전체28.532s PASS다.
PNPM noargs의 Usage/exit2는 실패로 보존하고 자기 release member만 기존 고정 discard
probe callback으로 --version을 검사한다. Generic GHI와 기존 native4/Policy는 유지한다.
관련3package full Race22.297s와 최종 CI-connected canonical Quick34.137s/security4.723s/
docs/workflow/release-gate/freshness는 PASS다. [로컬 결과](./evidence/package-client-quarantine/p2b2-pnpm-admission-local-20261024.json)가 범위를 소유한다.
CI helper 자체의 별도 frozen 공식 archive 재생27.636s는 실제 PNPM4·기존 설치 거부와
Root/temp 정리/daemon wait0/자원0/Host불변을 확인했다. Public HTTPS/hosted CI 성공을
주장하지 않으며 실제 native 전체 qualified·signed whole update·P2c public graph/layout/CLI는
미완료다. 완료된 namespace/Docker 시험 데이터·이전 바이너리944,957,519B를
모든 UID 참조0/unknown0 뒤 제거했다. 현재 milestone/item은 바뀌지 않으며 M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 P2b2 M6 제한 변경 승인과 Node 역할 보완

사용자는 [두 exact 읽기 변경안](./evidence/package-client-quarantine/p2b2-rust-artifact-runtime-decision-20261024.md)을 명시 승인했다. 이미 관측한 자기 guest PID maps만 먼저 적용하며
overcommit은 해당 ARTIFACT first read 확인 전까지 계속 거부한다. 기존 다른 PID/environ/mem/
Host/쓰기/network 거부를 유지한다. Node 기능 PASS와 generic CONTROL fallback의 역할 gap도
분리했다. Test-only 새 Node matrix RED(기존108 PASS) 뒤 기존 UV creator 검증을 재사용한
보완14의109 latch가 PASS다. Actual Node는 ARTIFACT/CLONE_CHILD/pinned image까지 확인했고
immutable loader ABI 읽기를 같은 query guard 아래 연결하는 다음 검증을 진행한다.
아래 승인 대기 문장은 당시 이력이며 현재는 승인된 범위의 구현·실제 검증 단계다.

### 2026-10-24 P2b2 CONTROL lock와 M6 명시 계약 차이

Actual 보완11의 PNPM CONTROL startup3.77s/empty lock3.82s·lock9·완전한 관찰/의심0와
자원0/wait0/Host불변은 PASS다. 같은 입력의 Node child query는 입증하지 않았으며 다음
명시 runtime.onFail=error 입력으로 확인한다. 독립 ARTIFACT5.19s는 기존 M6가 명시 거부한
자기 maps 읽기에서 INCOMPLETE다. [구체 계약 변경안](./evidence/package-client-quarantine/p2b2-rust-artifact-runtime-decision-20261024.md)을 작성·두 번째 검토했지만
maps/VM 예외는 미적용이며 승인 전에는 그 거부를 유지한다. 독립 CONTROL 검증은 계속하고
M13-004만 IN_PROGRESS다. 전체 P2 qualification/배포·승격은 미완료다.

### 2026-10-24 P2b2 exact guest startup 로컬 범위

[Guest 지도](./evidence/package-client-quarantine/p2b2-pnpm-admission-plan-20261024.md)의
고정 Root source·공통 UV/PNPM transport/volume·exact profile을 연결했다. Quick174.844s/
security/docs·관련3package Race23.754s, 실제 source 정상+11거부와 C++108 latch가 PASS다.
실제 PNPM --version3.69s는12.15.0/완전한 관찰·잔여0/wait0/Host불변으로 PASS다.
최초 GNU/Rust read 실패와 fixture 교정 원본은 보존한다. 현재 offline empty lock/Node
query와 독립 ARTIFACT lane·원격 경계가 남으며 P2 전체 QUALIFIED/ACCEPTANCE_CLOSED는 NO다.
완료된 첫 transport/archive·시험 사본491,429,837B를 참조0/unknown0 뒤 삭제했다.

### 2026-10-24 P2b2 actual guest 공급·admission 착수 감사

[새 실제 소비자 지도](./evidence/package-client-quarantine/p2b2-pnpm-admission-plan-20261024.md)는
기존 단일 pin/배포/설치와 UV/NPM helper·volume·observer를 재사용한다. 고정 Root 소유
current PNPM 파일과 `/haa-runtime/pnpm` CONTROL을 연결하되 release ARTIFACT probe는
독립 검증한다. 현재 이 actual runtime 범위는 IMPLEMENTED/WIRED/QUALIFIED/
ACCEPTANCE_CLOSED 모두 NO이며 새 source 작성 전에 수정 지도와 두 번째 검토를 기록했다.

### 2026-10-24 P2b2 native 배포·기존 설치/복구 로컬 검증

[배포 경로 결과](./evidence/package-client-quarantine/p2b2-pnpm-distribution-local-20261024.json)는
NOTICE의 source-bound owner/freeze, native+passive 출력, exact11 assets와 5native/3metadata의
producer·검사·설치·bootstrap·sealed Promotion을 연결한다. Final product source의
Quick119.521s·canonical6 및 관련13package 전체 Race146.917s PASS다. 뒤의 gated root fixture
두 개만 교정한 [실제12 namespace 검증](./evidence/package-client-quarantine/p2b2-root-fixture-qualification-20261024.json)도 PASS이며 Host12files/images/bridge/service가 동일하다.
완료 시험 바이너리6디렉터리219,678,968 logicalB를 참조0/unknown0 확인 뒤 제거했다.
Guest PNPM 공급·exact gVisor admission/startup/query/lock가 다음이며 제품 CLI/새 signed
release·원격 qualification은 미완료다. M13-004/P2b2만 IN_PROGRESS다.

### 2026-10-24 P2b2 native 명시 출력 로컬 checkpoint

[실제 출력 결과](./evidence/package-client-quarantine/p2b2-pnpm-materialize-local-20261024.json)는 같은 bounded parser의 native bytes 반환과
선택한 source owner의 명시 외부 출력/no-replace·readback·0555/단일 링크를 검증했다.
Quick29.766s/security/docs·관련 전체 Race21.186s 및 실제 official/변조 거부3.732s PASS다.
초기 architecture 위반과 기존 TERM fixture 실패는 원본/교정 범위를 분리 보존했다.
Native 배포에는 공식 LICENSE/제3자 고지 전달도 포함하여 전체11 asset/5native role로
연결한다. 이 다음 배포·실제 admission과 전체 P2 qualification은 미완료다.

### 2026-10-24 P2b2 단일 owner·유지보수 로컬 checkpoint

[로컬 결과](./evidence/package-client-quarantine/p2b2-pnpm-owner-local-20261024.json)는 schema3 owner/proof·생성 코드·drift,
v2 metadata의 exact5 images+native tool, 명시 offline PNPM plan/apply/no-op을 연결한다.
실제 공급물의 중단 후 재개·변조 거부와 Quick131.199s·관련 전체 Race42.209s 및 canonical
security/docs/workflow/release-gate/freshness가 PASS다. Native 배포/admission·제품 명령은
아직 미연결이며 전체 P2b2의 QUALIFIED/ACCEPTANCE_CLOSED는 NO다. 개별 helper CI 없이 다음
배포·설치 역할 연결을 계속한다. Host 설치/서비스는 변경하지 않았다.

### 2026-10-24 P2b1 native 공급·형식 로컬 검증

[실제 결과](./evidence/package-client-quarantine/p2b1-pnpm-native-local-20261024.json)는 signed archive와
regular native file identity, Quick117.599s/security/docs·전체 관련 Race19.732s PASS다. 고정 Node
이미지의 network-none/read-only/non-root 환경19개 형식 관측과 정확한 Host 보존·완료 container/
파생 image 정리를 확인했다. Product gVisor/Policy/transaction 지원을 의미하지 않는다. 다음은
[단일 pnpm runtime owner·유지보수·배포/공급·exact admission 지도](./evidence/package-client-quarantine/p2b2-pnpm-runtime-plan-20261024.md)를 따라 공급 owner부터 연결한다.

### 2026-10-24 P2 착수 감사·공급 검증 순서
+
+[P2 baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p2-pnpm-baseline-20261024.md)는
+기존 npm 검역·guard/transaction·typed descriptor·Sigstore owner를 재사용한다. 공식12.15.0의
+Rust native·lock/peer graph·tar integrity/revisions·설정 및 실제 upstream self-hosted signer를
+대조했다. 현재 P2a는 offline 공급 verifier/유지보수 consumer만 구현하며 runtime pin·CLI 지원은
+아직 추가하지 않는다. 전체 P2는 IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO,
+MISSING은 착수 지도의 supply/runtime/graph/layout/actual transaction·원격 qualification이다.
+
### 2026-10-24 G127 완료·P2 인계

[G127 누적 결과](./evidence/package-client-quarantine/go1273-acceptance-20261024.json)는
같은095450d의 ordinary·cu126·cu130·cu132 각각 Required10SUCCESS와 실제 native macOS,
observer106/helper 종료를 연결한다. 기본 SDK·quality tools5·설치 Go members7을1.27.3으로
전환하고 보호12 metadata·실제 npm/pip·owned doctor/stop/readback 및 updater no-op을 확인했다.
완료 사설 입력9.55GB를 삭제했고 운영 복구용 원본 SDK/runtime/tools는 보존했다. Release 설치는
UNAVAILABLE이며 public release/main 병합을 의미하지 않는다. 과거396/local40m 실패는 보존한다.
G127의 네 상태는 모두 YES/MISSING 없음이다. 다음 P2는 기존 구현 감사·공식 의미·수정 지도와
두 번째 검토부터 시작하며 M13-004만 IN_PROGRESS다. 아래 날짜별 NO/PENDING은 당시 이력이다.

### 2026-10-24 G127 CI395 성공·CI396 cu126 지연 조사

[최초 실패·정상본 대조](./evidence/package-client-quarantine/go1273-cu126-ci396-failure-20261024.json)를
보존했다. CI395는 c34365e/같은 whole tree의 Required10SUCCESS·Linux53/CPU278.97/native
macOS/observer106이다. 별도 cu126/CI396은 CPU263.98s·다른8jobs PASS지만 sample gap1.451s
때문에 FAIL이며 Required도 FAIL이다. 정리 성공으로 원래 실패를 덮지 않는다.

[현재 진단 지도·두 번째 검토](./evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)를
따라 기존 코드7개·normal cu126/CI392·오답노트를 대조했다. 같은 기존24GB WSL에서 허가된
임시 재검증은40m timeout으로 실패했고 confirmed drain 뒤 원본12 복원을 확인했다.
Journal은 종료 약115초 전에 시작되어 promotion 전체40분 지연 가설은 반증됐다. 정확한
단계 시각 누락만 기존 progress port의 bounded test observer로 보완한다. 예산·poll·coverage는
유지한다. New pressure 관측은
직접 인과 확정이나 SDK regression 단정이 아니다. G127 IN_PROGRESS/QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO, P2..P7 NOT_STARTED다. 영구 SDK/runtime activation은 미실행이다.

[단계 진단 로컬 검증](./evidence/package-client-quarantine/go1273-phase-diagnostic-local-20261024.json)은
기존 progress port 연결과 bounded/unknown/동시 호출 Race·canonical Quick/security/docs PASS다.
첫 socket EPERM은 실행 환경 교정과 함께 보존했다. 완료 local payload와 임시 clone 약17.65GB는
참조0 확인 뒤 정리했다. 한 후보를 push하고 실제 단계 시각과 결과를 타이머로 확인한다.

### 2026-10-23 G127 최종 로컬 검증·source 반영

[최종 로컬 결과](./evidence/package-client-quarantine/go1273-local-qualification-20261023.json)에서
두 frozen plan·공식 SDK1.27.3·실제 도구5개·Go image·six-member strict 재현/manifest 일치·
observer106·namespace13·configuration7·vulnerability/fuzz/Linux platform/full Race를 확인했다.
최초 TLS acquisition/긴 TMPDIR fixture 실패와 수정 범위는 같은 영수증에 보존했다.
전체2,081개 frozen 입력을 대조하고 선언된10파일만 source에 반영했으며 완료 namespace 시험
payload는 삭제했다. 현재 owner1.27.3과 Host global SDK1.26.9/설치 runtime 역할을 구분한다.

IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다. MISSING은
동일 최종 후보의 원격 installed ecosystem/CPUthreeCUDA·Required/native platform과 controlled
activation/readback/old-owned retirement다. 한 번의 기능 후보 push와 실제 profile 시간 기반
타이머로 확인한다. G127은 IN_PROGRESS, P2..P7은 NOT_STARTED이며 배포·main 병합은 별도다.

### 2026-10-23 G127 도구 호환성 — 최초 오류를 보존하고 최소 pin 갱신

모듈 형식 보완 뒤 NEW target plan은 bootstrap/build/test-build/vet까지 PASS지만 기존
Staticcheck2026.1의 export decoder가 Go1.27 data version5를 읽지 못했다. 첫 실패를 보존했으며
[수정 지도·공식 지원·두 번째 검토](./evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)의
방법으로2026.2 exact pin만 갱신했다. 이후 새 target에서 export data5/max4로 다시 거부되어 공식2026.2.2 patch가 실제1.27.3 source build/전체 offline analysis8.613s를 통과했다. Current SDK canonical과 새 target plan은 별도다. [Current SDK 로컬 결과](./evidence/package-client-quarantine/go1273-staticcheck-current-local-20261023.json)는 bootstrap·Quick56.004s/security/docs PASS다. 다음 NEW plan의 target 분석은 남는다. 다른 도구·검사 조건을 바꾸거나 아직 실행하지 않은
gVisor/native/CPU/CUDA를 완료로 쓰지 않는다. 기존 두 frozen state를 수정하지 않고 새 baseline에서
다음 plan을 만든다. 원격 CI는 최종 Go tuple 기능 경계로 묶고 실제 시간 기반 타이머를 사용한다.

### 2026-10-23 G127 첫 configuration 확인 — 모듈 형식만 보완

첫 exact1.27.3 plan/apply와 공식 SDK 준비는 PASS이며 bootstrap도 PASS다. Quick은 Go1.27의
require 블록 통합 형식에서 module drift를 탐지했다. [수정 지도·대조 결과](./evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)의
방법으로122개 require version/directness·기타 directive·go.sum 불변과 old/new tidy no-op을
확인했다. [Current SDK canonical 검증](./evidence/package-client-quarantine/go1273-module-layout-local-20261023.json)의
Quick48.866s/security/docs도 PASS다. Root floor1.26.10을 유지한 grouping만 반영했고 NEW
baseline/plan을 만든다. 첫 plan/실패 원본은 보존한다. Host 개발 SDK의 실제 readback은1.26.9였으며 source/qualification
SDK1.26.10·격리 target1.27.3과 역할을 구분한다. 기존 Host는 아직 전환하지 않았고 gVisor·전체
target tuple qualification·coherent activation/old owned retirement는 미완료다.

### 2026-10-23 G127 착수 감사 — 기존 updater 재사용

P1을7ce362b로 문서 종료했고 target Go1.27.3의 공식 release/advisory와 세 SDK archive metadata를
재확인했다. [갱신된 감사·수정 지도·두 번째 검토](./evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)를
따른다. 기존 coordinator·SDK bootstrap·gVisor capture/verify는 재작성하지 않는다.
README Build의 자동 renderer/consistency 소비자 누락은 [같은 회귀/기존 소비자 검증](./evidence/package-client-quarantine/go1273-readme-consumer-local-20261023.json)으로
보완했다. Canonical5/전체 두 도구 Race25.424s는 PASS이며 version owner·거부 조건은 유지했다. 이후 깨끗한 baseline에서 기존 두 frozen plan과 실제 target SDK/
bundle·observer를 검증한다. 현재 pin/SDK는1.26.10이며 Host SDK/CLI·서비스는 변경하지 않았다.
다음 단계의 P2..P7은 시작하지 않는다. 단일 active work item은 계속 M13-004다.

### 2026-10-23 P1 COMPLETE — 실제 네 profile·문서 검증과 정리

[누적 실제 결과](./evidence/package-client-quarantine/p1-uv-acceptance-20261023.json)는
sourceea287b2/whole tree9ece0a2의 CI391·cu126/CI392·cu130/CI393·cu132/CI394를 연결한다.
각 Required 포함10SUCCESS이며 기존 Linux/CPU/macOS와 helper5wait0/final cleanup을 확인했다.
Complete103.23s/closure46.14s에서 동일 버전의 새 Run·records, 교체/제거·blank skip·두 explicit
empty·제외 native 원본 proof/adopt·5wheel/SBOM5nodes·4edges를 검증했다.
CUDA full은 cu1261244.38s/cu1301192.58s/cu132878.63s이며 각 실제 입력/소스·CPU와 분리했다.

IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES/ACCEPTANCE_CLOSED=YES, MISSING=[]다.
문서·보안 profile을 통과했고 같은 문서 후보를 마지막으로 확인해 기록한다. 다음은 G127의
기존 구현/실제 입력 감사·수정 지도·두 번째 검토이며 P2는 G127 qualification까지 대기한다.
중간 커밋을 보존한 한 번의 push 뒤 타이머로 확인했고 완료한 임시 CLI·검증 실행기와 test
payload를 정리했다. Active SDK/runtime/source/cache는 G127의 실제 전환·복구·마지막 참조
확인 뒤 retirement한다. 마지막 jobs 조회 ReadTimeout은 읽기 재조회로 복구했으며 CI 실패가 아니다.
아래 단계별 로컬/원격 실패는 당시 관측으로 유지한다. M13-004만 IN_PROGRESS다.

### 2026-10-23 CI390 확인과 검증 주기 조정

[CLI·공통 batch 로컬 결과](./evidence/package-client-quarantine/p1f-uv-sync-cli-local-20261023.json)는
complete sync의 Application/CLI/bootstrap·새 승인 records 리포트와 빈 입력 skip을 연결했다.
기존 정상 pip의 batch lifecycle·static-all/대상별 closure를 재사용하여 rootless 의존성 전달
누락을 교정했고 같은 pre-format test의 before FAIL/after PASS를 보존했다. 최종 canonical5,
관련7package 전체 Race138.116s와 실제7언어 CLI21cases는 PASS다. 임시 CLI 바이너리43,581,079B와
test payload를 정리했다. 실제 native complete/adopt/empty/closure 두 통합 시나리오는 기존 Linux
job에 연결했으며 당시 아직 실행 전이었다. Backend와 CLI commit을 한 번의 원격 후보로 올린 뒤 실제
native·공통 CI/Required와 세 CUDA를 확인한다. P1 acceptance는 열어 둔다.

[Complete backend 로컬 결과](./evidence/package-client-quarantine/p1f-uv-sync-backend-local-20261023.json)는
actual held source/target guard·전체 native 원본 proof 연결, 단일 complete 선택과 승인 cache·
Manifest/SBOM·기존 offline publication을 검증했다. Canonical5와 관련5package 전체 Race142.011s
PASS이며 empty 추출 회귀의 기존 정상 코드 대조·교정 및 최초 실패는 같은 영수증에 보존했다.
Synthetic 거래 검증이며 당시 실제 native 전체 검역·Application/CLI·CUDA3/P1은 미완료였다.
이후 D 연결의 로컬 결과는 위 영수증을 따르며 개별 helper별 새 원격 실행 없이 기능 경계로 검증을 묶는다.

[CI390 확인](./evidence/package-client-quarantine/p1f-ci390-timed-confirmation-20261023.json)은
source79e76dd와 actual checkout의 전체 tree 일치·Required 포함10SUCCESS를 확인했다.
Linux51/CPU297.93s, macOS 같은2m build 한도 안의 약59.17s/default test와 helper 정리는 PASS다.
앞 macOS timeout 내부 원인을 해결했다고 해석하지 않으며 actual complete sync/CLI·CUDA3와
P1 acceptance는 미완료다. 이전 CI388/389 항목은 당시 실패 이력으로 보존한다.

기존 예방 규칙을 현재 P1에 적용한 [검증 실행 주기](./20-package-client-quarantine-plan.md#5-검증-matrix와-자원-정리)를
적용한다. P1f A2c2/C/D의 실제 guard·cache·records와 CLI를 연결했고 작은 helper별 원격 완료 대기를
중단하고 backend 및 CLI end-to-end를 한 후보로 push/CI를 묶는다. 로컬 canonical/관련 회귀와
중간 commit은 유지한다. CUDA3 actual은 P1 완료 후보에서 수행하며 미검증 상태로 완료·승격하지
않는다. 새 공통 회귀나 native 선행조건이면 필요한 원격 검증을 먼저 수행한다.
CI 실행 시 최근 같은 profile의 경과 시간으로 첫 확인 시각을 잡고 대기 도구로 확인한다.

### 2026-10-23 CI389 macOS build — 당시 결과

[CI389](./evidence/package-client-quarantine/p1f-cpu-completion-ci389-20261023.json)은 CPU 회계 교정 source의 Linux/CPU full·뒤 네 소비자와 정리는 PASS다. macOS는 production build2m에서 timeout이고 default test는 미도달이다. [기존 macOS 진단 지도](./evidence/package-client-quarantine/macos-default-test-diagnostic-plan-20261022.md)의 bounded package progress 보완을 먼저 수행한다. 내부 지연 원인·새 source 인과와 전체 qualification은 미확정이다.

### 2026-10-23 CI388 선행 회귀

[최초 결과](./evidence/package-client-quarantine/p1f-uv-original-proof-ci388-20261023.json)의 native
UV/CLI·46개 Linux 선행 tests·다른8 prerequisites는 PASS지만 CPU full의 unit-outcome 회계 시각
역전으로 Linux/Required는 FAIL이다. 원본/복원을 보존했고 [수정 지도·두 번째 검토](./evidence/package-client-quarantine/p1f-cpu-unit-completion-baseline-20261023.md)를 먼저 따른다. [로컬 교정](./evidence/package-client-quarantine/p1f-cpu-unit-completion-local-20261023.json)의 same-test before FAIL→after PASS·canonical5/전체 관련 Race는 PASS이며 fresh source 실제 검증은 남는다.
B1/A2c1 local 후보를 이 실패 실행에 포함하거나 기존 CUDA 양의 지연과 같은 원인으로 쓰지 않는다.

### 2026-10-22 순서 결정 — P1 완료 후 Go1.27.3, 그다음 P2

사용자는 GO-2026-6809와 Go1.27.3 전환을 추가 요청하고 순서 판단을 위임했다.
현재 source/CI/runtime/tool pin은 이미 보안 수정1.26.10이며 [CI377 최종 결과](./evidence/package-client-quarantine/p1c-guarded-cli-ci377-20261022.json)의
Required/Vulnerability 포함10SUCCESS를 확인했다. 이 advisory의 미패치 source 복구와
다음 Go 계열 전환을 구분한다. 기존 Host/preview를 자동 업데이트한 것은 아니다.

| 순서 | 범위 / 상태 | 시작·완료 기준 |
| --- | --- | --- |
| 1 | P1 uv pip — IN_PROGRESS | C3b 빈 기존 venv 설치는 완료. C3c installed-aware 설치/업데이트·native ownership·complete/multi-root/empty sync와 P1 실제 qualification을 닫는다 |
| 2 | M13-004/G127 Go1.27.3 — NOT_STARTED | P1 acceptance 뒤 [착수 감사·수정 지도·검증·두 번째 검토](./evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)를 갱신하고 기존 maintenance plan/apply/verify로 별도 후보를 검증한다. Compiler/guest/tools/gVisor bundle·observer·CI/release/support tuple과 actual 소비자/플랫폼/CPU·세 CUDA/전환 경계를 확인 |
| 3 | P2 pnpm → P3..P7 — NOT_STARTED | G127 qualification/acceptance 후 새 기준에서 확장을 재개한다. 전체 client 완료까지 Go 전환을 미루지 않음 |

M13-004 하나만 IN_PROGRESS이며 G127을 동시에 착수하지 않는다. Go1.26은 공식 지원 중이고
현재 취약점은1.26.10에서 수정되어 P1 기능 변화와 runtime 변화를 분리할 수 있는 경계가
적절하다. 새 reachable advisory/지원 종료가 이 판단의 전제를 바꾸면 긴급 순서를 재평가한다.
1.27.3은 전환 목표이며 현재 lock/workflow/Host SDK는 이 순서 기록에서 변경하지 않는다.
후속 요청에 따라 G127 완료 기준에는 실제 compiler/binary/bundle/설치 readback의 버전 일치와
전환·복구 검증/마지막 참조 확인 뒤 구버전 owned SDK·cache·bundle·시험 자원의 정리를 포함한다.
현재 P1의 active1.26.10 입력은 유지하며 역사 evidence의 버전 문자열을 실행 잔재로 삭제하지 않는다.

### 2026-10-21 owner scope — client expansion and full-name brand

CI386을 기다리는 동안 [C3e3 publication 감사·수정 지도](./evidence/package-client-quarantine/p1f-uv-sync-publication-baseline-20261023.md)를 추가했다. [CI386 종료](./evidence/package-client-quarantine/p1e-uv-native-source-ci386-20261023.json)는 같은 전체 tree/Required10SUCCESS와 실제 native 다섯 사례·기존 Linux51/UV CLI194.90s/CPU245.89s/native macOS·helper5wait0/정리를 확인했다. C3e2a/b의 네 상태는 해당 resolver 범위에서 YES다. 다음 A1은 기존 원본 metadata capture/verify와 inventory 공통화만 수행한다. Whole guarded sync/P1은 미완료이며 A2/C1/D1 지도와 각 후속 source 직전 검토를 따른다.

M13-004만 IN_PROGRESS로 유지한다. [확장 계획](./20-package-client-quarantine-plan.md)은
uv pip/project·pnpm·Bun·pipx 설치/업데이트와 npx·uvx·bunx·pipx run 등을 하나의
범위로 설계하고 단계별 실제 검증 후 지원을 연다. 기존 검역 엔진은 재사용한다.
설계·두 번째 검토 P0와 로컬 검사·preview 갱신을 완료했다. 실제 화면 결과는
[풀네임 타이틀 보완 기록](./evidence/m13-004-convenience/owner-cli-review-20261021.md)을 따른다.
사용자 계속 지시에 따라 [P1 uv baseline](./evidence/package-client-quarantine/p1-uv-baseline-20261021.md)의
P1a native lock/선택 경계를 커밋b710b56으로 보존하고 CI360을 확인한다. 다음은
[P1b runtime baseline](./evidence/package-client-quarantine/p1b-uv-runtime-baseline-20261021.md)의 metadata/native tool 공급은 [로컬 영수증](./evidence/package-client-quarantine/p1b-uv-tool-local-qualification-20261021.json)까지 확인했다. [Pin/maintenance 로컬 검증](./evidence/package-client-quarantine/p1b-uv-pin-local-qualification-20261021.json)도 actual CLI/canonical7 PASS다. 다음은 [cu126 회계 지연 조사](./evidence/package-client-quarantine/go12610-cu126-diagnostic-plan-20261021.md)와 UV production runtime 연결이다. P1은 IN_PROGRESS, P2~P7은 NOT_STARTED다.
[Go1.26.10 보안 선행조건](./evidence/package-client-quarantine/go12610-security-baseline-20261021.md)을
기존 유지보수 도구의 별도 candidate에서 [최종 로컬 검증](./evidence/package-client-quarantine/go12610-local-qualification-20261021.json)을 마쳤다. [원격 상태](./evidence/package-client-quarantine/go12610-remote-status-20261021.json)에서 같은 tree의 CI361 Required/CPU/native macOS PASS와 cu126 sample-late FAIL을 확인했다. cu132도 별도 full1054.68s·Required10SUCCESS이며 cu130의 Linux/full980.48s도 PASS이나 native production build2m timeout으로 Required는 FAIL이다. 새 진단 source270c7c8의 PR366 Required10SUCCESS·cu126 Linux/full1296.54s PASS를 확인했으나 manual native default-test timeout/Required FAIL이다. Go 전체 qualification은 미완료다. Host 설치·서비스는 유지한다.
[P1b5 원격 결과](./evidence/package-client-quarantine/p1b5-uv-runtime-remote-20261022.json)는
CI368의 신규 native UV3.25s·기존 Linux45/CPU259.85s PASS와 macOS default-test6m/
Required FAIL을 구분한다. [P1c 수정 지도](./evidence/package-client-quarantine/p1c-uv-resolver-baseline-20261022.md)는
native JSON이 delta-only임을 확인하고 기존 pip promotion identity/control과 multi-root
sync 모델의 연결 gap을 기록했다. Production UV resolver/install-upgrade-sync/CLI는 아직
미연결이다. [macOS 진단 지도](./evidence/package-client-quarantine/macos-default-test-diagnostic-plan-20261022.md)의
bounded 진행/완료 root 요약은 기존 한도·process cause를 유지한다. Source66aa7e0의
[CI370 최종 결과](./evidence/package-client-quarantine/p1c-baseline-ci370-20261022.json)는
same-tree checkout/Required 포함10SUCCESS·native macOS와 Linux 기존45/native UV3.20s·
CPU209.50s/정리를 확인했다. 과거 timeout 원인이나 이후 candidate qualification은 아니다.
승인한 fixed read 후보는 커밋3623d1e/fresh C++105/startup4.09s PASS이며 offline
inventory의 다음 최초 거부는 Python child libpython read다. [C1c 수정 지도](./evidence/package-client-quarantine/p1c-uv-resolver-baseline-20261022.md)의
exact clone/creator·첫 query exec·ARTIFACT 역할·readonly ABI/stdlib 후보를 구현 중이다.
초기 false exec-origin 전제는 실제 pinned kernel의 OriginExec 상속에 맞게 교정한다.
Native 최소 empty-target query 검증은 아직 남고 전체 inventory 성공으로 확대하지 않는다.
M13-004/P1만 IN_PROGRESS이며 production 지원/acceptance는 계속 미완료다.
후속 [CI371](./evidence/package-client-quarantine/p1c-checkpoint-ci371-20261022.json)은
source3623d1e/same tree의 Linux 기존45·UV3.67s/CPU196.37s/정리는 PASS, macOS
47started/47finished·active0 뒤 default6m/Required FAIL이다. [종료 상태·완료 gap 진단](./evidence/package-client-quarantine/macos-terminal-local-20261022.json)은
canonical5·Race15.425s/Darwin compile PASS이며 새 native는 남는다. C1c는 actual
query 역할을 관측했고 ABI·VM/own maps·UTC와 UV parent의 exact Python image data
read 후보를 C++106으로 검증 중이다. 개별 첫 거부·완료 자료 정리는 P1c 지도가 소유한다.
C1c parent-image 후보의 fresh C++106/startup3.91s/query4.10s·complete trace31은 로컬
PASS다. Whole-image inventory 진단은 초기 commit/raw로만 보존하고 실제로 통과한
startup/empty-target 두 fixture만 CI에 연결한다. Source 정리 후 final replay도 startup3.73s/query3.96s·complete trace29/31로 PASS다.
새 원격 qualification은 남는다. 다음 구현은 같은 P1c의 production native 선택/data-reader 연결이다.
[C2 baseline/수정 지도](./evidence/package-client-quarantine/p1c-uv-native-producer-baseline-20261022.md)는
기존 stdin admission·bounded tar와 isolated Python data reader를 재사용하는 단일-root
producer의 실제 남은 작업을 분리한다. [C2 output 로컬 검증](./evidence/package-client-quarantine/p1c-uv-output-local-20261022.json)은
native stdin→pylock→sync/list0과 named tmpfs의 bounded control transport, 새 collector/
RFC822-WHEEL adapter의 canonical5·Race9.454s PASS를 확인했다. Guest-only /tmp의
docker cp 실패는 별도 보존했고 두 probe의 소유 자원/완료 자료는 제거했다. 다음은 UV
output volume의 실제 gVisor mount 연결과 typed-network native 선택·독립 data reader다.
Production resolver/CLI는 아직 미연결이며 P1 acceptance는 열려 있다.
[C1c CI372](./evidence/package-client-quarantine/p1c-query-ci372-20261022.json)는 sourcee3665ea와
checkout tree 일치·remote startup3.03s/query2.90s·complete trace29/31, 기존 Linux/CPU283.96s와
helper5wait0/정리를 확인했다. macOS는 default test 이전 production build2m에서
exited=false/exit-1·CPU339188ms로 종료됐고 Required는 FAIL이다. 앞 default6m 실패의
원인으로 합치지 않으며 전체 qualification·production UV 지원은 계속 미완료다.
[C2 actual output 로컬 검증](./evidence/package-client-quarantine/p1c-uv-native-output-local-20261022.json)은
named output의 startup4.10s/query4.25s/compile-sync-collection5.07s·complete trace29/31/35,
canonical5·Race7.919s와 자원0/Host 불변을 확인했다. 세 번째 native case를 CI 필수 selector와
누락 부정 시험에 연결했다. 완료 native 자료 약2.75GB/일회성26파일은 삭제했다.
다음은 동일 P1c의 public PyPI native 선택·독립 isolated metadata/archive reader이며
production resolver/guarded install-upgrade-sync·CLI와 전체 remote qualification은 남는다.
Public producer의 다음 수정은 기존 PyPI/PyTorch lifecycle을 private session helper로
공유하는 것이다. Native solver를 pip로 바꾸지 않고 독립 metadata/archive reader만 이
세션을 재사용한다. 구체 지도/두 번째 검토는 C2 baseline에 있으며 새 후보 검증은 진행 중이다.
기존 session 추출의 canonical5·관련 Race10.199s는 PASS다. Private 독립 archive reader와
고정 replay/실패 회귀를 추가했고 다음 canonical 검증 및 실제 native producer 연결이 남는다.
[독립 reader 로컬 검증](./evidence/package-client-quarantine/p1c-uv-reader-local-20261022.json)의
canonical5·Race10.055s와 기존 parser를 통한 whole archive/graph replay가 PASS다. Metadata/
archive mismatch·byte/stream·취소/cleanup/trace 실패의 graph 삭제와 임시 파일 제거도 확인했다.
실제 Python HTTP 및 public native UV caller/guarded CLI는 아직 미연결·미검증이다.
Reader source344a93d는 로컬 checkpoint로 보존했다. 반복된 macOS prerequisite에 대해
[cache 입력 정렬의 baseline/수정 지도·두 번째 검토](./evidence/package-client-quarantine/macos-default-test-diagnostic-plan-20261022.md)를
추가했다. Native CI373은 build PASS/default6m FAIL이며, 실제 cache root와 -trimpath
action ID 분리를 확인했다. Scope/timeout·원인 판정은 유지한 채 이 불일치만 정렬한다.
[Cache 입력 정렬 로컬 검사](./evidence/package-client-quarantine/macos-cache-alignment-local-20261022.json)는
canonical6·checker Race36.760s·Darwin compile11.016s PASS다. Source lookup의 실제 비교를
보존했고 완료 binary/operator7파일은 제거했다. 새 native macOS/전체 Required는 남는다.
[CI373 최종 결과](./evidence/package-client-quarantine/p1c-native-output-ci373-20261022.json)는
source9b0da57/checkout tree 일치, native startup3.24s/query3.13s/compile-sync-collection3.58s·
complete trace29/31/35와 기존 Linux/CPU287.67s·helper5wait0/정리를 확인했다. macOS는
build94.05s PASS 뒤 default6m/Required FAIL이다. 이 결과를 이후 reader344a93d나 cache
정렬 후보의 원격 qualification으로 전용하지 않는다. 새 후보 CI가 다음 확인 지점이다.
Sourcef76c564를 push했고 CI374가 진행 중이다. 다음 public native UV caller와 독립 reader의
순차 연결 후보를 구현했고 source/runtime/tool preflight·owned-container cleanup 회귀를
검증한다. App/guarded install-upgrade-sync·CLI는 미연결이며 public/HTTP actual은 남는다.
[Public caller 로컬 검증](./evidence/package-client-quarantine/p1c-public-caller-local-20261022.json)은
canonical5·관련 Race16.610s와 lock-only preflight/snapshot actual 세 fixture를 통과했다.
완료 native 자료688,406,687B는 제거했다. Public single/multi graph entry를 기존 CI의
인증된 test-client 단계에 연결했고 다음 source CI가 실제 public/HTTP 완료 조건이다.
[CI374 종료](./evidence/package-client-quarantine/cache-alignment-ci374-20261022.json)는
sourcef76c564/same checkout tree·Required 포함10SUCCESS다. macOS build55.562s와
default test envelope171.294s 이내, Linux native 세 시험/기존 통합·CPU186.73s 및 정리가
PASS다. Cache 정렬 후보의 native 완료를 확인했고 과거 모든 timeout의 원인은 단정하지 않는다.
이후 public caller sourceab54d4a는 별도 push했으며 실제 public/HTTP·guarded CLI는 남는다.
[Guarded client 다음 gap 감사](./evidence/package-client-quarantine/p1c-guarded-client-baseline-20261022.md)는
기존 promotion/update/enrollment·bootstrap의 재사용 위치와 UV/multi-root gap을 분리했다.
읽기 전용 baseline이며 구현은 시작하지 않았다. C2의 public actual 결과부터 확인한다.

[CI375 첫 public 결과](./evidence/package-client-quarantine/p1c-public-ci375-20261022.json)는
sourceab54d4a/same checkout tree에서 idna@3.11 native compile이 exact /etc/hosts의
read 분류 누락으로 STREAM_FAULT였다. Required/Linux FAIL, 나머지8 prerequisite SUCCESS,
helper wait0·confirmed stop/원래 client 복원은 확인했다. Public graph/독립 HTTP는
미통과다. C2 leaf의 수정 지도·두 번째 검토에 따라 UV direct root의 exact system hosts
읽기만 보완하고 전체 latch/canonical 및 새 actual public 입력으로 검증한다. C3 구현과
P2..P7은 시작하지 않는다.

C2의 새 CI376을 기다리는 동안 [native retention/empty/venv 형식 조사](./evidence/package-client-quarantine/p1c-uv-retention-format-20261022.json)를
독립 data-only 범위로 수행했다. 기본/선택 upgrade의 기존 dep 유지, 전체 upgrade의 dep
갱신, sync의 제외 항목/빈 집합 삭제와 uv venv의 version_info/생성 marker 차이를 확인했다.
잘못 지정한 첫 empty output filename의 exit2는 operator 오류로 별도 보존했다.
[C3 지도](./evidence/package-client-quarantine/p1c-guarded-client-baseline-20261022.md)에
수정 위치와 두 번째 검토를 반영했으며 제품 구현은 NOT_STARTED다. Owned containers6/
images2와 완료 자료3,143,502B는 제거했고 Host 전후 불변을 확인했다.

[CI376 최종 결과](./evidence/package-client-quarantine/p1c-public-ci376-20261022.json)는
source35c0f2d/same checkout tree·Required 포함10SUCCESS다. Native UV→독립 whole-archive
binding은 idna1node/requests5node·4edge를22.75s에 반환했다. C++106/Linux49 통합과
CPU295.93s/helper5wait0/final confirmed cleanup, native macOS도 PASS다. C2 single-root
producer의 acceptance만 닫고 P1 전체/guarded client·CLI는 미완료로 유지한다. 다음은 C3
지도의 기존 venv transaction/서비스 재사용과 client/source 분리 gap의 순차 구현이다.
C3a는 resolve 전 venv guard를 획득하는 내부 신규 설치 준비 adapter를 시작한다.
기존 Update의 per-request rebind와 transaction을 재사용하고, 배포된 distribution이 있는
환경의 fresh solve는 거부한다. Installed-aware 선택/CLI는 후속 실제 연결 단계다.
[C3a 로컬 checkpoint](./evidence/package-client-quarantine/p1c-prepared-install-local-20261022.json)는
canonical5와 application 전체/관련 promotion·sandbox Race3개를 통과했다. 내부 준비 서비스와
empty-venv guard를 구현했고 task TMPDIR는 비어 있다. Composition/CLI와 실제 target 반입은
아직 미연결·미검증이다. 다음 C3b는 client/source descriptor와 실제 factory 연결이다.
C3b 연결 구현을 시작한다. 실제 두 Python client만 descriptor와 factory에 등록하며 기존
pip/PyTorch 동작과 lazy help를 유지한다. UV 신규 설치는 빈 기존 venv에 한정하고,
installed bootstrap-test의 실제 검역·반입/비어 있지 않은 대상 거부를 기존 CI에 연결한다.
[C3b 로컬 결과](./evidence/package-client-quarantine/p1c-guarded-cli-local-20261022.json)는
canonical5와 CLI/bootstrap/application/promotion/checker 전체 Race PASS다. 실제 빈 venv
UV CLI 연결은 IMPLEMENTED/WIRED YES, QUALIFIED/ACCEPTANCE_CLOSED NO이며 다음은
같은 후보의 actual native CLI/Required CI다. 기존 설치·upgrade/native ownership·sync는 남는다.
CI 대기 중 C3 leaf에 C3c source 감사·metadata-only native 전송의 미확정 지점·유한 조사와
수정/회귀 지도를 기록했다. C3c code는 시작하지 않았으며 C3b actual CI 확인이 우선이다.
[후속 metadata-only 형식 조사](./evidence/package-client-quarantine/p1c-uv-metadata-only-format-20261022.json)는
14개 native 입력과 원본 불변·Host 동일성·owned 자원 정리를 확인했다. 다음 UV 선택 입력은
bounded inert METADATA로 구체화하며 gVisor/public/guarded update 성공으로 확대하지 않는다.
[CI377 종료 확인](./evidence/package-client-quarantine/p1c-guarded-cli-ci377-20261022.json)은
source8ec8de7/actual checkout334918f·동일 전체 tree와10SUCCESS다. 실제 UV CLI 설치25.50s,
nonempty 반복 거부·target/Evidence 불변, Linux50/CPU283.72s/helper5wait0/final confirmed
cleanup와 native macOS·Vulnerability를 통과했다. C3b 빈 venv 설치의 네 상태는 모두 YES다.
현재 다음은 C3c installed-aware install/upgrade/native ownership·sync이며 P1 전체는 미완료다.
C3c1 native installed-aware producer를 시작한다. C3 leaf의 사전 감사·정확한 수정 지도·두 번째
검토를 확인하고 strict delta/원본 metadata controls와 기존 UV native/독립 binder 연결만 구현한다.
Guarded update 반입·CLI/native ownership·sync는 후속이며 P1 밖으로 이동하지 않는다.
[C3c1 로컬 결과](./evidence/package-client-quarantine/p1c-installed-producer-local-20261022.json)는
canonical5/관련 Race3개 PASS다. Native resolver/CI는 연결했고 다음은 실제 public/Required
검증이다. Original metadata 한도와 selected graph 한도는 구분했고 actual native 성공은 아직 아니다.
[CI378 종료](./evidence/package-client-quarantine/p1c-installed-producer-ci378-20261022.json)는 같은 전체 tree와 Required 포함10SUCCESS, 실제 native 세 intent19.60s/4controls, Linux51/CPU231.72s·helper5wait0/최종 confirmed cleanup와 macOS/Vulnerability를 확인했다. C3c1 네 상태는 모두 YES다. 다음 C3c2a는 기존 original snapshot/held transaction과 UV 선택 controls를 연결하는 managed guard이며, CLI 변경·native adoption·sync는 후속이다. 정확한 지도와 두 번째 검토는 C3 leaf를 따른다.
[C3c2a 로컬 결과](./evidence/package-client-quarantine/p1c-managed-guard-local-20261022.json)는 canonical5와 promotion/application/artifact-pypi 전체 Race PASS다. Managed UV guard port와 exact controls/lock/graph·기존 반입/rollback 연결은 구현했으며, 실제 shared pip 소비자/Required CI와 production CLI·actual UV target update는 미완료다. 그 원격 검증 뒤 C3c2b binding을 진행한다.
[CI379](./evidence/package-client-quarantine/p1c-managed-guard-ci379-20261022.json)의 same-tree10SUCCESS와 실제 공통 소비자·macOS·정리로 해당 원격 gap을 닫았다. C3 leaf의 C3c2b 사전 감사/수정 지도/두 번째 검토대로 일반 installed-aware 설치와 두 UV upgrade intent를 연결한다. Native UV ownership과 sync·P1 전체는 아직 미완료다.
[C3c2b 로컬 결과](./evidence/package-client-quarantine/p1c-installed-cli-local-20261022.json)는 canonical5/전체 관련5package Race PASS다. 일반 installed-aware INSTALL과 --upgrade/-U·--upgrade-package의 UPDATE를 기존 guard/검역/반입에 연결했고 actual authenticated CLI5단계·부정 검사를 기존 CI에 추가했다. 다음은 같은 source의 실제 CLI/Required 확인이며 아직 native qualification 완료가 아니다.
CI 대기 중 [native UV 형식 감사](./evidence/package-client-quarantine/p1d-native-uv-ownership-baseline-20261022.md)의 자체 wheel/loopback14명령과 정리를 확인했다. Native public/adoption 또는 현재 CLI qualification으로 전용하지 않으며 C3c2b 완료 뒤의 구현 gap에 반영한다.
[CI380 종료](./evidence/package-client-quarantine/p1c-installed-cli-ci380-20261022.json)는 동일 tree/Required10SUCCESS와 실제 UV CLI5단계118.85s·Linux51/CPU293.60s·macOS·정리를 확인했다. C3c2b 네 상태는 YES다. 다음 C3d는 위 native UV 감사의 기존 old-wheel proof·namespace 보호·native fixture 검증을 재사용한다. Sync와 P1 전체는 미완료다.



공개 채널 이름·publisher binding·PR38/main CI·배포는 기존 미완료로 남는다.

### Runtime upgrade maintenance — 2026-10-15 priority change

사용자가 M13을 잠시 보류하고 버전 갱신 자동화의 상세 설계·계획을 먼저 정리한
뒤 구현하도록 요청했다. 기존 M13 branch는 `f5a0353`에 보존하고 같은 commit에서
`milestone/runtime-upgrade-maintenance`를 분기했다. Tracked/index clean 기준선과
기존 미추적 사용자 자료를 유지했다. 원격 branch/PR/ruleset/설치 변경은 없다.

기술 owner: [Runtime upgrade design](../engineering/05-runtime-upgrade-maintenance.md).
Baseline audit·작업별 Read/acceptance·검증 순서·M13 재개 조건:
[Runtime upgrade plan](./19-runtime-upgrade-maintenance-plan.md).

| Item | Scope | Status | Acceptance evidence |
| --- | --- | --- | --- |
| UPG-001 | baseline audit / design / detailed plan | COMPLETE | 소스 변경 전 목표별 audit와 design/plan 작성; [closure evidence](../engineering/evidence/runtime-upgrade-maintenance/upg-001.json) |
| UPG-002 | all owned version consumers / manifest / drift | COMPLETE | [ownership/manifest evidence](../engineering/evidence/runtime-upgrade-maintenance/upg-002.json); canonical Quick, security,12 focused regressions/54 cases PASS |
| UPG-003 | explicit plan/apply/verify / same-version rehearsal | COMPLETE | [updater evidence](../engineering/evidence/runtime-upgrade-maintenance/upg-003.json); actual no-op2/2, canonical verify7 profiles,10 tests/28 cases PASS |
| UPG-004 | gVisor·archived quality tool Go1.26.9 candidates | COMPLETE | [quality compiler](../engineering/evidence/runtime-upgrade-maintenance/upg-004-gitleaks.json) / [gVisor build](../engineering/evidence/runtime-upgrade-maintenance/upg-004-gvisor.json); actual/reproducible candidates and observed-output plans qualified |
| UPG-005 | installed ecosystem / CPU / cu126 / cu130 / cu132 qualification | COMPLETE | [qualification evidence](../engineering/evidence/runtime-upgrade-maintenance/upg-005.json); same local full matrix and fresh ordinary/nativeMac/three independent remote CUDA Required PASS; originals/cleanup and final docs/security verified |
| UPG-006 | verified activation / retirement / M13 handoff | COMPLETE | [activation/handoff evidence](../engineering/evidence/runtime-upgrade-maintenance/upg-006.json); actual default quality5/installed Go7 at1.26.9, production2 and scoped doctor, no-op/7canonical profiles, exact completed-test cleanup and final docs/security PASS; public release/Race/reviewer merge retain separate scope |

이번 범위는 owned versions의 갱신 절차와 Go/gVisor1.26.9 전환이다. 모든 언어의
최신판 동시 도입, 사용자 프로젝트/provider 재작성, 새 CUDA profile 또는 M13 공개
배포를 포함하지 않는다. UPG-001 완료와 전체 유지보수 완료는 구분한다.

UPG-002 start audit (2026-10-15, baseline `b4eaa5b`): existing lock/generator and
Go setup consistency IMPLEMENTED=YES/WIRED=YES; complete consumer/support/workflow
inventory and four-image manifest QUALIFIED=NO/ACCEPTANCE_CLOSED=NO.
MISSING: Node runtime version instead of npm in manifest; Rust entry and release
inspection; selected Go literals in workflow/tool validators; support/version/ABI
cross-checks and missing/duplicate/unknown consumer regression. Canonical checks
remain read-only. Images, tool releases, user artifacts and installed bundle stay
at the existing versions; gVisor/Gitleaks compiler rebuild belongs to UPG-004.

UPG-002 closure: selected Go values now feed all actual workflow setup validators;
reviewed versions and every active tool setup are cross-checked. Unknown workflow
or independent official-image literals in active Go source fail the existing
read-only runtime drift check. The existing manifest includes Rust and uses Node's
version instead of npm. Python image/version/ABI drift is rejected. Full PyTorch
integration uses generated reviewed torch versions and canonical Python target
paths/assertions; frozen inspection-prerequisite and historical fixtures remain.
Actionlint's existing v1.7.12 now has an actual official release review record.
Runtime/tool lock bytes and all four local amd64 image identities are unchanged.
Final native Quick, security/release-gate, generator idempotence and focused12/54
regressions passed. Current scoped audit: IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES/
ACCEPTANCE_CLOSED=YES/MISSING=none for UPG-002. Updater, actual compiler rebuild,
installed/CPU/CUDA/remote qualification and activation remain UPG-003..006.

UPG-003 start audit (baseline `16cfe2948b604605a20241547a0ab67e0bee4287`):
unified updater IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO.
MISSING: explicit official-metadata plan; isolated candidate apply with baseline/
hash preconditions and durable recovery journal; target-SDK canonical verify;
same-version no-op and coordinated four-language/negative/fault regressions.
Existing generator, selected-version checks and canonical profiles are reused.
Source edits remain separate from Host activation and privileged lifecycle.

UPG-003 closure: one developer command now resolves exact official metadata,
freezes coordinated patches and runs existing canonical checks in an isolated
Git worktree. Actual current Go1.26.9 plans2/apply2 produced identical plan IDs,
zero source diff and no additional SDK. Persistent state/journal and all7 verify
profiles passed. Four independent synthetic component upgrades and recovery/
stale-input/foreign-edit/architecture/digest/SDK negatives passed10 tests/28 cases.
Official live metadata resolution passed for all4 current runtimes; same-version
reconciliation preserves original image digests, even when a live tag differs.
Source/tool/runtime pins and installed binaries are unchanged. Scoped audit:
IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES/ACCEPTANCE_CLOSED=YES/MISSING=none for UPG-003.
Native macOS execution, actual new SDK activation, gVisor/quality compiler rebuild,
installed/CPU/CUDA/remote qualification and final migration remain separate.

UPG-004 start audit (baseline `adbf422c7ca82a15ee35d2ae8834172eba03bd77`):
existing complete bundle/observer builders, strict bundle verifier and source tool
installer IMPLEMENTED=YES/WIRED=YES; selected-Go SDK preparation/candidate capture,
Gitleaks exact source compiler build and actual compiler enforcement
IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO.
MISSING: Go1.26.9 gVisor SDK/patch/module-lock preparation shared by local/CI/release;
two reproducible complete builds, member/compiler/capability/102-latch evidence;
same-release Gitleaks source identity/flags/output and documented detection corpus
comparison; actual compiler checks for every managed Go tool and wrong-input
regressions. Existing builds/validators are extended, not replaced. Candidate
preparation does not install/activate the helper or claim full qualification.

UPG-004 quality-tool component: [Gitleaks/compiler evidence](../engineering/evidence/runtime-upgrade-maintenance/upg-004-gitleaks.json).
Gitleaks remains8.18.4 at the official tag/module h1 and now source-builds with the
selected Go; schema2 rejects archived active installs. All5 managed candidate
binaries prove actual Go1.26.9/package/platform metadata. Two independent Linux
builds plus canonical bootstrap produce identical Gitleaks output; Darwin both
architectures cross-build without claiming native execution. Three reconstructed
M7 detection/benign classes match the official archive and canonical security;
compiler/lock negatives pass21 cases. Actual updater verify passes7 profiles with
state-local executables and an untouched caller-cache sentinel. Quality component
audit is YES/YES/YES/YES/MISSING=none. At quality-component commit ed6adbc, overall
UPG-004 remained IN_PROGRESS with gVisor SDK/common preparation, capture and
full bundle/observer evidence still open. The later closure below records those
results. Global activation and full installed/CPU/CUDA/remote qualification remain open.

UPG-004 closure (2026-10-16): the shared preparation selects Go from the runtime
owner, preserves the exact upstream SDK patch and feeds existing local/CI/release
builders. Independent cold-cache bundle and C++ observer builds are byte-identical;
all5 Go members report1.26.9, four trace capabilities and102 latches pass. Actual
default strict bundle and shared observer entrypoints pass. Capture mode retains
its actual source/outputs/receipt; schema2 plans reconstruct source+patch and bind
observed member/module identities to coordinated generation. Same-SDK actual
plans/apply produce zero diff; synthetic changed outputs require both generated
identity and full Go qualification. Source/SDK/member/compiler/wiring negatives
and final canonical Quick/security/workflow/docs/release-gate pass; vulnerability/
platform/qualification-freshness also pass. The first concurrent-build Quick
5m timeout is retained separately; unchanged frozen inputs pass7 profiles after
build completion. Scoped audit: YES/YES/YES/YES/MISSING=none. Global installed
binaries, CPU/three CUDA, new remote Required and activation remain UPG-005/006.
UPG-005 is NOT_STARTED/ReadyYes and requires its own baseline audit.

UPG-005 start audit (2026-10-16, baseline c56bb3160f8ff74f125fac5754a0b4f8127dda3e):
existing installed lifecycle, seven-ecosystem consumers, CPU/three CUDA and
Required workflow IMPLEMENTED=YES/WIRED=YES. Qualification of the new bundle and
final product binaries QUALIFIED=NO/ACCEPTANCE_CLOSED=NO. MISSING: exact product/
helper/test-binary freeze; durable installed originals and bounded ownership/
resource preflight; actual small/lifecycle/hostile and cross-ecosystem consumers;
CPU plus independently cu126/cu130/cu132 with unchanged contracts; fresh native
platform/Required and supported-profile remote results; exact cleanup/restore
and final evidence. Reuse scripts/ci-integration-cleanup.sh and existing test/
workflow bodies; no replacement executor, new runtime contract or CI jobs.
UPG-004 canonical results establish configuration/build validity, not these
installed/full acceptances. The separate known full-Race fixture findings keep
their prior scope pending disposition; Required does not contain a Race job.

UPG-005 newly confirmed fixture gap: npm/PyPI resolver tests in
internal/sandbox/linux_integration_test.go still require euid0, while the existing
resolver_policy_integration_linux_test.go explicitly uses the ordinary-user
installed typed helper. The file is unchanged from preserved M13 baseline f5a0353.
Actual new-bundle resolver batch passed8 consumers but failed the PyPI guard
before resolver execution; helper wait0 and original12 restoration passed. Do not
run the entire client as root or weaken firewall/coverage. Narrow correction:
require ordinary-user in the two existing fixtures and retain the existing typed
helper and assertions. Fixture precondition start audit: IMPLEMENTED=NO/WIRED=YES/QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO; MISSING: corrected guards and actual two-resolver regression
plus canonical checks. Product resolver/helper implementations are reused.

UPG-005 fixture correction closure: [partial qualification evidence](../engineering/evidence/runtime-upgrade-maintenance/upg-005.json).
Both existing resolver tests now require the ordinary user and preserve the
installed typed helper/result assertions. Actual npm/PyPI resolvers pass2 cases;
root execution is rejected before resolver construction in2 negatives. Canonical
Quick/security/docs pass. Actual ordinary npm is-number7.0.0 and pip packaging25.0
install/promotion also pass; both cases confirm parent wait0 and all12 originals
restored. Scoped fixture audit YES/YES/YES/YES/MISSING=none. Initial batch FAIL and
its8 individual passes remain historical; final candidate/full CPU/three CUDA/
remote Required and activation remain open. No product resolver/helper, firewall
privilege, budget, support tuple or coverage assertion is changed.

UPG-005 actual full progress: final40dda27 CPU PASS484.09s with helper wait0 and
all12 originals restored. cu126 FAIL588.88s before inspection: explicit prerequisite
target12.9.9 hash is absent because permitted live metadata selected cuda-bindings
12.9.10. Actual wheel matches independent official PyPI SHA; both old/new identities
and publish times are recorded locally. No SDK regression or same-filename mutation
is established. cu130/cu132 NOT_RUN. Do not retry the stale input or change corpus.

New gap baseline audit before source changes: existing exact prerequisite parser/
loader/binding, production resolver and test/CI consumers IMPLEMENTED=YES/WIRED=YES.
Trusted live qualification preparation and stale-pin pre-intake validation
IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO. MISSING: explicit test-only
plan for fixed NumPy and unique public non-root selected cuda-bindings; existing
resolver metadata freeze/receipt; reused early target check; workflow plan/env
wiring and negative/actual small regressions; same final candidate full matrix.
Design owner is runtime upgrade design section6. Existing defaults, exact CLI
input, helper authority, resource/support/Required and promotion are unchanged.

UPG-005 preparation component verification: preview ff6ea504 passes canonical
Quick/security/workflow/docs/release-gate, selector/early-resolver negatives, and
actual installed preparation plus four-profile auxiliary authentication326.85s.
Wrong target rejects with main intake0; helper parentwait0 and originals12 restored.
Scoped preparation audit YES/YES/YES/YES; full-matrix audit remains QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO. Final code commit, same-candidate CPU/three CUDA and fresh
Required/native platform are still MISSING. Prior40dda27 results retain their own
source identity and are not rewritten as this new executable candidate.

UPG-005 final baa8512: small4 PASS23.93s, ordinaryCLI2 PASS35.36s and CPUfull
PASS482.43s, each primary0/restore0. cu126 FAIL884.09s during installation.
First accounting rejection is gap1.106363289s > existing1s, composed of
trusted Read619.130156ms + before-read487.232044ms + ledger lock0.001089ms.
Usage66.251083s <300s and wall remaining2328.65s; neither aggregate limit exhausted.
Later CANCELED/PROFILE_LOOKUP_FAILURE do not replace that first cause. Helper
wait0/originals12 restore confirmed. Target preparation/NumPy acquisition passed
this earlier boundary; cu130/cu132 NOT_RUN. Existing CPU accounting/helper owner
audit YES/YES/NO/NO. MISSING: narrow latency substage diagnosis and controlled
regression if a defect is proven, then same-candidate full acceptance. Budgets,
late rejection and authentication stay unchanged; no blind full retry.

UPG-005 retained installation diagnosis: first timing overlay FAIL232.03s/restore0;
request receive/decode1142.94ms while peerAuth0.08ms/engine33.87ms. Same code/input
with OS scalar telemetry PASS338.10s/restore0. That success does not fix the earlier
failure. Telemetry confirms peak shared memory12.53GiB, available memory minimum
295.48MiB, increased swap/direct reclaim; helper quota unlimited/no throttling.
Initial local RAM preflight missed peak working-set headroom. Preserve strict1s
rejection, CPU/wall/tmpfs/support and add conservative local execution capacity
readback before another full. Windows32GB/WSL~16GB, no .wslconfig; a24GB memory-only
proposal was prepared for separate user review because it affects all WSL distros.
User approved the testing24GB memory-only setting. The previously absent config
was created and independently read back from Windows/Linux with matching SHA.
Post-restart readback (2026-10-16): user restarted WSL; actual MemTotal
25,202,302,976B and MemAvailable23,311,159,296B exceed the unchanged local CUDA
capacity requirement19,327,352,832B. Config SHA matches its approved bytes.
The missing native source was reconstructed from baa8512 and all1,527 file
bytes/modes match the frozen inventory. Root audit confirms all12 originals,
zero installed consumers/containers/handles. First postrestart-cu126-full failed
at helper startup0.2s before CUDA execution because canonical /run directories
were missing. Historical UNKNOWN/restore1 is retained. A separate root complete
proc/executable/socket/Docker check and startup-before-executor source evidence
established recovery safety; originals12 restored and both canonical runtime
directories prepared. The local harness now checks directory identities before
installation; fresh small installed regressions precede another full case.
New CUDA qualification remains open; RAM readback alone is not a latency fix.
User authorized revalidation/restoration/runtime directories and removal of
completed unnecessary test data, with actual consumer/inventory checks.
Post-restart small4 PASS27.00s/primary0/restore0, helper parentwait0 and original12
readback PASS. Completed-test cleanup removed30,537 unnecessary files after full
SHA/metadata and root proc/FD/maps/mount/Docker reference checks: logical
9,231,393,994B, WSL available increase9,315,295,232B. Current candidate, originals,
distinct failed inputs and actual Evidence/raw/results remain. No VHD compaction
claim is made. Fresh postrestart-ready-cu126-full now runs with the same baa8512
candidate and unchanged40m/accounting/coverage bounds.
Final result: postrestart-ready-cu126-full PASS2344.78s/primary0/restore0,
helper parentwait0 and all12 originals verified. Viewer NOT_ATTESTED is retained
in result/promotion references; auxiliary input remains outside29 promoted
entries. This is a new full qualification result, not proof of the historical
latency's cause. Successful cu126 payload cleanup removed16,209 files, logical
13,914,708,718B/WSL available+13,959,614,464B, retaining93 actual Evidence and
qualification/staging/result/cleanup records. Fresh postrestart-ready-cu130-full
passed1959.83s/primary0/restore0 with the same candidate; root audit confirms
original12/consumer0/container0/handle0. Viewer limitation and auxiliary exclusion
from29 promoted entries are verified. Its completed payload cleanup removed
16,158 files, logical11,563,865,654B/WSL available+11,608,633,344B; actual Evidence,
manifest, qualification and cleanup receipts remain. The remaining cu132 follows
record validation. Same-candidate cross-ecosystem/fresh remote/activation
acceptance is still open.

Postrestart-ready-cu132-full PASS2005.17s/primary0/restore0; helper parentwait0,
all12 original identities and zero consumers/containers/handles verified. Viewer
NOT_ATTESTED and auxiliary exclusion from29 promoted entries are preserved.
Same baa8512 local full matrix is now CPU482.43s/cu1262344.78s/cu1301959.83s/
cu1322005.17s PASS. Historical failed attempts remain; this does not establish
the original latency's cause. Successful cu132 cleanup removed16,158 files,
logical11,437,557,165B/WSL available+11,482,320,896B. Remaining completed-test
cleanup removed109,640 files, logical8,858,768,369B/available+9,257,005,056B:
regenerable Go/Staticcheck/Bazel results, completed rehearsal snapshots/inactive
historical tool cache and CLI copies/targets. Actual root references and locks
were checked, structured CLI evidence preserved and cache symlink target
directories recreated empty. Historical tools were not byte-identical to active
tools and are recorded as obsolete verified cache, not reusable identical input.
Five deletion batches total188,702 files/logical55,006,293,900B and per-batch
available increases55,622,868,992B. This includes new output generated during
this work; it is not whole-turn net free-space gain or Windows VHD compaction.
Fresh remote Required/nativeMac/three independent CUDA qualification remains.
UPG-005 is IN_PROGRESS and UPG-006 NOT_STARTED until acceptance is confirmed.
Local-record commit8afeffa was pushed to the user-owned public maintenance branch
after owner/admin/push verification and current-tree/reachable-history secret
checks. Draft PR34 is open; CI313/run38920741069 tests merge36ce170, whose whole
tree equals branch8afeffa. Eight prerequisites including Linux passed; Required failed only because
native macOS117039701759 failed TestNativeBinaryDefaultPaths at120.01s:
nested go build -trimpath was killed at the existing2m context; preceding
production build succeeded. The fixture is unchanged from main. No blind rerun,
budget increase, test skip or production activation is performed.

UPG-005 native-build preparation gap baseline audit before source changes:
IMPLEMENTED=YES/WIRED=YES for native path assertions and canonical platform
build/test; QUALIFIED=NO/ACCEPTANCE_CLOSED=NO. MISSING: controlled cache-input
comparison for production build without trimpath versus the existing nested
trimpath build; minimal preparation alignment if confirmed; unchanged deadline/
assertions and actual new native macOS/Required qualification. The pinned Go
source includes BuildTrimpath in buildActionID. This establishes separate cache
keys, not all causes of remote elapsed time or a Go SDK regression. Existing
implementation is reused; product runtime/full local acceptances remain scoped.
Controlled selected-Go Linux comparison used independent empty build caches:
unmatched preparation caused356 nested compiler invocations/17.544s; matched
trimpath preparation caused0/1.121s. This confirms cache reuse only, not exact
macOS scheduling causality. Both existing canonical production-build steps now
use trimpath; native fixture flags/deadline/assertions and every selected package
remain. Candidate Quick/platform/security/docs/workflow and new native remote
qualification are pending before closure. Linux CI313 completed SUCCESS before correction publication; its actual
first-candidate result and decoded raw are preserved. No active run is cancelled.
Corrected candidate selected-Go Linux canonical Quick208.60s/platform7.09s/
security4.87s/workflow0.72s/docs0.16s PASS with read-only whole-source identity
checks. Native/default-path assertions and all existing tests remain; macOS
requalification is not inferred from Linux success. Final record checks precede
the scoped correction commit and fresh actual remote results.


UPG-005 final runtime qualification (2026-10-17): same baa8512 local full matrix
CPU482.43s/cu1262344.78s/cu1301959.83s/cu1322005.17s and original12 restoration
remain PASS. cfe7737 changes only documented preparation/cache alignment and a
fixture comment beyond the local candidate; product/runtime/policy source stays
unchanged. Fresh ordinary CI314/run38928160837 has native macOS/Linux/Required
and all10jobs SUCCESS. Same cfe7737 independent remote runs cu12638936267407,
cu13038985391376 and cu13238990568515 each have all10jobs SUCCESS and actual full
CUDA928.04s/758.80s/1098.68s (their CPU205.44s/192.07s/278.12s). Explicit profile
metadata, viewer NOT_ATTESTED, auxiliary exclusion from29 promoted entries,
each5helperwait0/finaloriginal0 CONFIRMED_STOPPED/clienterror0/clientrestore0
are verified from decoded raw. Requested cu130/cu132 one-hour timers were
actually3608s/3614s before terminal queries. Final evidence/record checks remain
before UPG-005 closure; activation/retirement belongs to UPG-006. Detailed actual
results and source/runtime boundaries are owned by upg-005.json and referenced
local raw receipts. No old failure/UNKNOWN is relabelled and no resource/coverage
bound or support tuple changed. Completed transport duplicates were deleted
after byte comparison; original evidence and rollback backup remain.


UPG-005 closure: qualification audit YES/YES/YES/YES/MISSING=none. Actual final
evidence docs/security passed with whole-source read-only checks; closure-only
state metadata checks accompany this commit. UPG-006 start audit before any
implementation/installation mutation is recorded in upg-006.json, baseline
cfe7737. Existing installation, bootstrap, doctor, lifecycle and updater are
IMPLEMENTED=YES/WIRED=YES and reused. Candidate components and prior recovery/
coordinated fixtures are qualified; operational closure remains NO. MISSING:
default Gitleaks activation, coherent global gVisor/candidate readback, bounded
doctor/production smoke, current no-op/future fixture rehearsal, safe retirement
and exact operator/M13 handoff. Original12/root consumer0 audit passes; current
global gVisor compiler1.26.3 and Gitleaks1.22.1 distinguish preparation from actual
activation. Host/default other quality compilers already1.26.9. Public attested
release installation remains M13 and unavailable doctor checks stay unavailable.
No existing implementation replacement, product protocol or new CI job is needed.


UPG-006 closure (2026-10-17): existing installation/bootstrap/lifecycle/updater
implementations were reused without product source replacement. Default quality5
and installed CLI/helper/five Go runtime members all report actualGo1.26.9;
Gitleaks matches the reproducibly qualified output. Original12 rollback backups
and protected installed hash/owner/modes remain verified. Ordinary npm/pip2
complete ALLOW/promotion in39.05s with confirmed helper wait0. Actual doctor has
four Host readiness OK under the owned helper, while full exit1 and unavailable
public release installation remain unchanged/M13. All helpers are then stopped;
this is controlled local file activation, not persistent/public service activation.
Current-version plan/apply/second no-op/verify passes all7 canonical profiles,
zero source changes/no new SDK/caller active quality binaries unchanged. Existing
coordinated future-version/recovery fixtures are included by unfiltered Quick;
no actual future SDK qualification is claimed. Completed private rehearsal and
test outputs40006 files/logical2458104726B were deleted after inventory, operation
lock and actual references/installed-file checks; actual Evidence10 preserved.
Separate first source/reader/output-parent failures and confirmed original restore
remain historical. Arbitrary mount query failure stays UNKNOWN; only proven
resource-empty stable zombies were separately classified for cleanup.
Operating guide/evidence docs/security PASS. Final audit YES/YES/YES/YES/MISSING=none
within UPG006; final metadata-head Required is observed separately after push.
Reviewer merge/main delivery and a fresh M13 baseline still block M13 resumption.
Known Race/release follow-ups are not closed by this result. Remaining final-check
caches/source copies are inventoried and disposed after verification, while active
inputs, rollback data and original evidence remain.

M13-001 COMPLETE는 유지한다. M13-002는 ON_HOLD이며 PR33·남은 CLA/stale/merge
acceptance와 기존 원본을 보존한다. M13 재개 시 새 Git/remote/PR/check 관측과
maintenance 변경을 반영한 baseline audit가 필요하다. 보류 전 CLEAN은 현재 상태가 아니다.

### SEC-2026-10-12 — Scheduled Security Go patch

#### 2026-10-15 local Go single-version maintenance

User-authorized prerequisite to M13 resumption: retain one active Go version,
1.26.9; retire host 1.26.5/1.26.7 and downloaded older SDKs. Existing host CI,
release and tool setup pins already use1.26.9. Baseline audit:
IMPLEMENTED=YES / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO for consistent
local/guest selection. The login PATH still selects the retained1.26.7 SDK,
the temporary1.26.9 install is absent, downloaded1.26.7/1.26.8 SDKs remain,
and the Go guest lock selects1.26.8. Existing installation, runtime-lock
generator and canonical checks are reused; no replacement implementation.
MISSING: permanent1.26.9 host activation, old-SDK cleanup, digest-bound1.26.9
guest update with affected-consumer checks, and reproducible final evidence.
This maintenance does not execute or qualify the pending protected PR merge;
At that maintenance phase M13-002 remained the sole IN_PROGRESS milestone work item.
The user-requested 2026-10-15 hold and UPG sequence above supersede that active status.

Progress: permanent1.26.9 installation, login/interactive/default command selection
and persistent GOTOOLCHAIN=local are verified. Exact12 old SDK/cache paths were
retired after active/unknown consumer0 and unchanged inventory checks. Official
1.26.9 guest digest is locked/generated and its network-disabled image build
smoke passed. The existing runtime-lock drift check now rejects host/product/
quality/guest version disagreement; final native canonical Quick/security/
vulnerability/workflow/release-gate/qualification-freshness and worktree docs PASS.
Original OneDrive timeout and excluded preparation/initial checker failures are
preserved. Automatic approval review initially rejected the privileged11-file
installation change; after concrete binaries/backups were prepared the user
explicitly approved actual16-test Go qualification and successful local activation.
That phase completed: all16 existing Go integrations and the installed production
CLI's actual managed download/build passed. The authorized1.26.9 bundle remains
active;11 installed hashes/root owners/modes and exact guest identity match the
candidate. Three helper waits0/final original0 CONFIRMED_STOPPED and no remaining
endpoints/containers were independently checked. Current local audit:
IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES;
MISSING=none for this local maintenance. New-source remote CI, separate full-Race/
release findings and M13-002 protected-path acceptance remain open. Details:
[single-version maintenance](../engineering/evidence/go-single-version-20261015/README.md).

- Status: `COMPLETE` for the bounded host security and CI repair. PR30 is merged
  at main `2e196a1`; main CI38328099413 is SUCCESS (Required plus nine jobs).
  PR CI308 also passed all45 Linux consumers, Cargo transitive build and PyTorch
  CPU full. The following baseline/follow-up paragraphs retain their historical
  scope; their pending labels are superseded by this delivery result.
  Guest runtime patch, full-Race fixtures and module advisory assessment remain
  separate release-readiness follow-ups; no public release approval is inferred.
- Baseline: `main@bdc2d891ca0edb719be8a28b1a8576decd3d01f2`;
  [scheduled run 38240142380](https://github.com/rahoney/heliopause/actions/runs/38240142380),
  job `114780428275`: history secrets PASS, vulnerability FAIL, fuzz SKIPPED.
- Baseline audit: `IMPLEMENTED=YES`, `WIRED=YES`, `QUALIFIED=NO`,
  `ACCEPTANCE_CLOSED=NO`. Existing `scripts/check/security.go:runVulnerability`,
  `scripts/check/tools.go:bootstrap` and four workflows already implement the
  gate; only the host/product Go security patch and fresh validation are missing.
- Current audit: `IMPLEMENTED=YES`, `WIRED=YES`, `QUALIFIED=YES`,
  `ACCEPTANCE_CLOSED=YES` within this bounded maintenance scope. Official archive integrity, canonical Quick/security/
  history/vulnerability/fuzz/docs/freshness, public Go and cold bootstrap
  integration, cross compile and CLI smoke passed under Go 1.26.9.
- Remote follow-up: Scheduled Security38249522013 SUCCESS. Candidate
  `a32cdb0` CI38249811910 failed at retained SumDB lookup. The bounded diagnostic
  repair `e01ba19` passes [CI38258121064](https://github.com/rahoney/heliopause/actions/runs/38258121064),
  Required and all nine prerequisites. The same transitive gRPC get/download
  passes82.97s; all45 Linux top-level consumers and cleanup pass. The previous
  underlying error was discarded and cannot be recovered. Exact Go1.26.8/1.26.9
  authentication comparisons and new full CI do not reproduce that failure;
  no specific historical timeout/outage or universal absence of a compiler
  regression is inferred. Future failures retain safe bounded diagnostics.
- Additional local finding: `go test -race -count=1 -timeout=15m ./...` failed on seven
  tests in `internal/sandbox/direct_exec_admission_test.go` and
  `shared_observer_test.go`. Six also fail on untouched `main@bdc2d89`
  with Go 1.26.8; the affected production/test files are unchanged. The seventh
  passed in that single baseline run; identical detection is not claimed.
- `MISSING`: none for the bounded host patch and CI repair; PR30 merge and
  main CI delivery are complete. The historical SumDB
  underlying cause remains unknown; the separate local full-Race finding and
  guest runtime patch remain follow-ups. Targeted verifier Race passes on both
  actual Go versions. No suppression, timeout increase, retry-to-green or
  runtime replacement is used to qualify this repair.
- CI scope correction: the current workflow aggregates nine prerequisites and
  does not include a Race job; `platform`/`quick` do not use `-race`. Local Race
  remains an unresolved broader qualification finding, independently of Required status.
  Following user authorization, `c73f3f5` was pushed and draft
  [PR30](https://github.com/rahoney/heliopause/pull/30) opened. Initial runs:
  [Required CI](https://github.com/rahoney/heliopause/actions/runs/38249446186),
  [Scheduled Security](https://github.com/rahoney/heliopause/actions/runs/38249522013).
- Evidence: [report](../engineering/evidence/security-go1269/README.md) and
  [machine-readable results](../engineering/evidence/security-go1269/result.json).
  [SumDB diagnostic follow-up](../engineering/evidence/security-go1269/sumdb-followup.md)
  and [remote result](../engineering/evidence/security-go1269/remote-result.json)
  record the source, local before/after and actual remote validation. The original
  local result is a historical snapshot, not the current remote CI status.
- Read: [Quality Toolchain](../engineering/03-quality-toolchain.md),
  [CI Quality Gate](../engineering/04-ci-quality-gate.md),
  [Trusted Tooling](../threat-model/04-trusted-tooling-and-evidence.md),
  [M10 runtime identity contract](./13-m10-verified-distribution-bootstrap-contract.md),
  [M12 Go contract](./16-m12-01-ecosystem-expansion-contract.md#4-m12-002--go-modules).
- Scope: host/product build toolchain and bounded SumDB failure diagnosis.
  `scripts/runtimes.lock.json`,
  generated runtime identities, qualified Go guest image and historical M12
  evidence remain exact inputs; no old qualification is relabeled for Go 1.26.9.
- Delivery follow-up: CI306 stalled in hosted APT acquisition and was cancelled
  by the next push, before runtime tests. The bounded official-mirror and
  timeout repair passes actual APT acquisition/install in CI307. APT audit:
  `IMPLEMENTED=YES`, `WIRED=YES`, `QUALIFIED=YES`, `ACCEPTANCE_CLOSED=YES`
  for package acquisition. The later Cargo failure remains separate; full
  delivery Required is not green. See the [APT follow-up](../engineering/evidence/security-go1269/apt-followup.md).
- CI307 follow-up baseline: `2eea071`, Linux job114896785383. APT acquisition
  completed and Go transitive download passed82.80s. Cargo transitive build
  failed during private selection with `OPEN_RESULT_CLASSIFICATION_IMAGE`.
  Its pinned CONTROL/direct-root Cargo read locator matches `/etc/gai.conf`,
  absent from `IsPinnedCargoResolverRead`. Observer/test/runtime lock are
  identical to `main@bdc2d89` and the successful CI305 source. Existing Cargo
  resolver is `IMPLEMENTED=YES`, `WIRED=YES`; this repair is `QUALIFIED=NO`,
  `ACCEPTANCE_CLOSED=NO`. The exact image catalogue confirms the diagnostic
  locator; the same final regression fails before and passes after the one-line
  path addition. All102 observer latch groups and canonical Quick/security/
  vulnerability/docs pass under Go1.26.9. `MISSING`: actual Cargo/common Linux
  consumers and fresh Required CI. M12 contracts and runtime identity checks
  remain unchanged. See the [Cargo follow-up](../engineering/evidence/security-go1269/cargo-gai-followup.md).

M0~M7은 원래 정의한 MVP qualification과 evidence를 완료했다. external
security review에서 확인된 production trust-boundary gap은 과거 M7 evidence를
삭제하거나 실패로 바꾸지 않고 M8 release-blocking remediation으로 처리한다.
M8은 완료되었다. M9-001은 canonical transaction contract를 확정했고,
M9-002는 HAA-managed npm project의 private mutation, complete Verified Set
commit과 rollback 회귀검증을 완료했다. M9-003은 active Python 3.14 venv의
baseline 충돌 방지와 HAA-owned file transaction 및 `pip` install CLI wiring을
완료했다. M9-004는 정적 root command tree와 help 무부작용 경계를 완료했고,
M9-005는 GitHub standalone 기본 destination과 no-overwrite 정책을 완료했고,
M9-006은 hostile transaction regression과 전체 qualification을 완료했다.
M10-001은 release signing identity, artifact manifest와 bootstrap trust contract를
확정했다. M10-002는 exact runtime lock을 소비하는 native/helper 빌드,
CycloneDX SBOM·release manifest 및 tag-only OIDC provenance candidate workflow를
완료했다. M10-003은 HAA-owned GHCR image가 실제로 필요하지 않음을 결정하고,
공식 Node/Python image를 digest/provenance manifest로 검증했다. M10-004는
verifier-gated atomic installer transaction과 fail-closed doctor를 완료했다.
public release publishing은 M10-007의 protected publication workflow에서만 수행한다.
M10-006은 Apache-2.0 outbound license, Harmony copyright-license CLA Option Five,
individual/entity contribution 절차와 pre-CLA source provenance를 확정했다.
M10-007은 exact tag/build-run candidate, manifest/checksum과 GitHub attestation을
검증하는 protected publication workflow implementation과 PR/CI qualification을
완료했다. 실제 production release environment·tag policy·main Ruleset activation·
`develop` branch 전환과 public release는 M12 및 M12-02 완료 후 M13에서 수행한다.

M11-001은 [M11 Dynamic Detection Depth Contract](./14-m11-dynamic-detection-depth-contract.md)에서
exact gVisor schema lock, bounded normalized vocabulary와 raw payload non-retention
contract를 먼저 확정했다. M11-002는 이 contract의 requested point/field가 pinned
source에서 실제로 compile·parse되는지를 Bazel contract test와 Linux lifecycle CI로
증명했다. M11-003은 이 bounded fact에 trusted backend classification과 actionable
Finding을 연결했고, exact helper와 Linux lifecycle CI로 검증했다.

M11 post-qualification release hardening은
[M11 이후 Release Hardening Fix List](./15-m11-02-fix-list.md)의 FIX-01~05만
수행한다. 새 architecture 또는 detection capability는 추가하지 않는다. 모든 fix의
local qualification을 완료하고 하나의 milestone branch push/CI evidence를 확인한
뒤에만 M12를 재개한다.

M12 생태계 확장 순서는 queue에 고정한다: 공식 PyTorch source profile → public Go
Modules → public Cargo/crates.io → public Terraform Provider → 전체 생태계
qualification·feature freeze → M12-02 final red-team/fix gate. M12 기능과 red-team
gate가 완료되기 전에는 M13 Production Release & Operations로 이동하거나 public
release를 게시하지 않는다. source checkout/local Go build 없는 public bootstrap은
M13-005가 완료하기 전까지 제공된다고 주장하지 않는다.

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

## 3. M0 실행 Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M0-001 | Project Identity 결정 | Step 13 | COMPLETE | No |
| 2 | M0-002 | Go·Tool·CI Identity lock | M0-001 | COMPLETE | No |
| 3 | M0-003 | Go module과 최소 process bootstrap | M0-002 | COMPLETE | No |
| 4 | M0-004 | Canonical check runner foundation | M0-003 | COMPLETE | No |
| 5 | M0-005 | Static analysis와 Quick profile 활성화 | M0-004 | COMPLETE | No |
| 6 | M0-006 | Quick·Docs·Required CI foundation | M0-005 | COMPLETE | No |
| 7 | M0-007 | Minimum Go·macOS와 repository gate 검증 | M0-006 | COMPLETE | No |
| 8 | M0-008 | M0 qualification과 M1 handoff | M0-007 | COMPLETE | No |

### M1 실행 Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M1-001 | Domain workflow entry decision | M0 | COMPLETE | No |
| 2 | M1-002 | Domain identity와 Run state foundation | M1-001 | COMPLETE | No |
| 3 | M1-003 | Check result·Port와 deterministic fake | M1-002 | COMPLETE | No |
| 4 | M1-004 | Inspect orchestration과 Policy v1 | M1-003 | COMPLETE | No |
| 5 | M1-005 | CLI result contract와 synthetic vertical test | M1-004 | COMPLETE | No |
| 6 | M1-006 | M1 qualification과 M2 handoff | M1-005 | COMPLETE | No |

### M2 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M2-001 | npm Static Inspect entry decision | M1 | COMPLETE | No |
| 2 | M2-002 | npm reference·metadata resolve foundation | M2-001 | COMPLETE | No |
| 3 | M2-003 | Controlled Intake와 declared integrity verification | M2-002 | COMPLETE | No |
| 4 | M2-004 | npm tarball static inspection과 Evidence Store | M2-003 | COMPLETE | No |
| 5 | M2-005 | M2 Policy·CLI wiring과 controlled vertical test | M2-004 | COMPLETE | No |
| 6 | M2-006 | M2 qualification과 M3 handoff | M2-005 | COMPLETE | No |

### M3 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M3-001 | Linux Dynamic Inspect entry decision | M2 | COMPLETE | No |
| 2 | M3-002 | gVisor runtime lock과 capability probe | M3-001 | COMPLETE | No |
| 3 | M3-003 | Sandbox Session·Observation Domain과 Port | M3-002 | COMPLETE | No |
| 4 | M3-004 | gVisor Sandbox Session lifecycle | M3-003 | COMPLETE | No |
| 5 | M3-005 | gVisor raw observation collector | M3-004 | COMPLETE | No |
| 6 | M3-006 | Dynamic observation normalization | M3-005 | COMPLETE | No |
| 7 | M3-007 | M3 Policy와 npm inspect wiring | M3-006 | COMPLETE | No |
| 8 | M3-008 | M3 qualification과 M4 handoff | M3-007 | COMPLETE | No |

### M4 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M4-001 | npm Install and Promotion entry decision | M3 | COMPLETE | No |
| 2 | M4-002 | Install Context와 locked npm resolver boundary | M4-001 | COMPLETE | No |
| 3 | M4-003 | recursive dependency inspection과 set-level Policy | M4-002 | COMPLETE | No |
| 4 | M4-004 | Verified Manifest·SBOM과 trusted Staging | M4-003 | COMPLETE | No |
| 5 | M4-005 | offline npm Promotion과 atomic target | M4-004 | COMPLETE | No |
| 6 | M4-006 | M4 qualification과 M5/M6 handoff | M4-005 | COMPLETE | No |

### M5 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M5-001 | PyPI/pip Expansion entry decision | M4 | COMPLETE | No |
| 2 | M5-002 | Python/pip runtime identity lock과 PyPI reference foundation | M5-001 | COMPLETE | No |
| 3 | M5-003 | isolated PyPI resolver·target tag·egress policy | M5-002 | COMPLETE | No |
| 4 | M5-004 | wheel intake·integrity·static inspection | M5-003 | COMPLETE | No |
| 5 | M5-005 | Python dynamic inspection·PEP 517 sdist build | M5-004 | COMPLETE | No |
| 6 | M5-006 | generic staging·offline pip Promotion·CLI integration | M5-005 | COMPLETE | No |
| 7 | M5-007 | M5 qualification·npm regression·M6 handoff | M5-006 | COMPLETE | No |

### M6 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M6-001 | GitHub Releases Standalone entry decision | M5 | COMPLETE | No |
| 2 | M6-002 | exact public input·API boundary foundation | M6-001 | COMPLETE | No |
| 3 | M6-003 | GitHub Release resolve·Acquire·integrity | M6-002 | COMPLETE | No |
| 4 | M6-004 | standalone static inspection·M6 Policy | M6-003 | COMPLETE | No |
| 5 | M6-005 | Linux ELF dynamic inspection wiring | M6-004 | COMPLETE | No |
| 6 | M6-006 | standalone Promotion·CLI·Linux E2E | M6-005 | COMPLETE | No |
| 7 | M6-007 | M6 qualification·regression·M7 handoff | M6-006 | COMPLETE | No |

### M7 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M7-001 | MVP qualification entry·evidence matrix | M5, M6 | COMPLETE | No |
| 2 | M7-002 | ecosystem flow·fixture regression | M7-001 | COMPLETE | No |
| 3 | M7-003 | Linux·macOS·WSL2 CLI qualification | M7-002 | COMPLETE | No |
| 4 | M7-004 | evidence·result·resilience qualification | M7-003 | COMPLETE | No |
| 5 | M7-005 | security workflow·scheduled gate qualification | M7-004 | COMPLETE | No |
| 6 | M7-006 | MVP final audit·documentation·completion | M7-005 | COMPLETE | No |

### M8 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M8-001 | production trust hardening entry·canonical contract | M7, external review | COMPLETE | No |
| 2 | M8-002 | Trusted Host tool identity·executor·local Docker endpoint | M8-001 | COMPLETE | No |
| 3 | M8-003 | observer supervisor·exclusive endpoint·process-scoped composition | M8-002 | COMPLETE | No |
| 4 | M8-004 | connection identity latch·production observation completeness | M8-003 | COMPLETE | No |
| 5 | M8-005 | least-privilege typed resolver network-policy helper | M8-004 | COMPLETE | No |
| 6 | M8-006 | runtime lock single source of truth·CI integration | M8-005 | COMPLETE | No |
| 7 | M8-007 | hostile-boundary regression·Linux security requalification | M8-006 | COMPLETE | No |

### M8-001 — production trust hardening entry·canonical contract

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [Trusted Tooling and Evidence](../threat-model/04-trusted-tooling-and-evidence.md)
- [Inspection, Sandbox and Policy Architecture](../architecture/03-inspection-sandbox-policy.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- M7 completion evidence를 보존하면서 M8이 production-ready release blocker임을
  canonical status와 public README에 정확히 표시한다.
- trusted local Docker endpoint, observer exclusive ownership, typed privileged
  helper와 truthful observation capability가 기존 invariant를 약화하지 않는
  entry contract로 확정된다.
- M8-002~007 각각의 dependency, acceptance와 check owner가 하나의 queue로
  정해지고 product implementation은 추가하지 않는다.

**Status**

```text
Status: COMPLETE
Checks: canonical docs profile; git diff --check
Evidence: M8 contract; D-015 Host trusted execution; M8 Host control plane architecture; M8~M11 milestone and seven-item M8 queue
Limitations: M8-001은 planning/contract만 변경하며 production code, helper, runtime 또는 CI job을 구현하지 않음
Next: M8-002 — NOT_STARTED / Ready: Yes
```

### M8-002 — Trusted Host tool identity·executor·local Docker endpoint

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [Trusted Tooling and Evidence](../threat-model/04-trusted-tooling-and-evidence.md)
- [Directory Structure](../engineering/01-directory-structure.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- Sandbox와 Promotion production process execution이 ambient PATH나 Host
  environment를 사용하지 않고 validated absolute tool identity와 minimal
  command-specific environment만 사용한다.
- Docker는 configured local endpoint의 transport, ownership와 daemon/runtime
  capability를 검증하고 arbitrary `DOCKER_HOST`, context, remote endpoint와
  proxy를 거부한다.
- writable/symlink/wrong-owner tool path, parent replacement, fake PATH, hostile
  Docker environment, wrong runsc digest/version와 registered-runtime mismatch
  fixture가 모두 fail-closed다.
- Core/Application/Policy에 Host executable·environment·Docker detail이
  유입되지 않는다.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/security; hostile tool/environment unit and contract tests; Linux Host executor integration; full CI Required
Evidence: commits 947b8af, fa0814e, 696289b, a3e1e56, 47052cb, da5a657; GitHub Actions run 32686523973 success including macOS, Minimum Go, Security, gVisor observer and Linux lifecycle
Limitations: observer supervisor와 privileged firewall helper는 후속 M8 item
Next: M8-003 — NOT_STARTED / Ready: Yes
```

### M8-003 — observer supervisor·exclusive endpoint·process-scoped composition

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [M3 Linux Dynamic Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Inspection, Sandbox and Policy Architecture](../architecture/03-inspection-sandbox-policy.md)

**Acceptance**

- HAA가 trusted receiver → pinned helper → readiness → Sandbox → helper/receiver
  cleanup lifecycle을 소유한다.
- npm, PyPI resolver/wheel/sdist와 GitHub backend는 process-scoped observer 하나를
  공유하며 factory별 endpoint unlink/rebind가 없다.
- MVP에서는 fixed endpoint exclusive ownership을 보장하고 다른 supervisor가
  소유하면 임의 재사용·삭제하지 않고 fail-closed한다.
- helper start/readiness/death, duplicate owner와 endpoint/helper cleanup failure
  integration이 non-ALLOW다. multi-process multiplexing은 별도 후속 확장이다.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/security; race; supervisor lifecycle unit/integration; git diff --check; full CI Required
Evidence: commits 06f769e, d377c55, 0163991, b32f329, 7a5aeb4; GitHub Actions run 32691425705 success including macOS, Minimum Go, Security, gVisor observer and Linux lifecycle
Limitations: Host-scoped multi-process multiplexing service 없음
Next: M8-004 — IN_PROGRESS / Ready: Yes
```

### M8-004 — connection identity latch·production observation completeness

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [M3 Linux Dynamic Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Inspection·Sandbox·Policy Contract](../domain-model/08-contract-inspection-sandbox-policy.md)

**Acceptance**

- helper가 최초 valid container ID를 connection identity로 latch하고 이후
  mismatch, malformed payload, dropped event와 abnormal end를 incomplete fault로
  전달한다.
- active Finding/Policy rule마다 production helper가 실제 생성하는 bounded
  signal과 Linux gVisor integration evidence가 있다.
- honeytoken, unexpected process와 filesystem violation을 생성할 수 없으면 해당
  capability는 unsupported/incomplete이며 synthetic test만으로 clean/ALLOW하지
  않는다.
- raw sensitive trace payload는 Core, Policy, CLI와 일반 Evidence에 유출되지
  않는다.

**Status**

```text
Status: COMPLETE
Checks: pinned Bazel helper test; parser/fuzz; actual gVisor observation integration; policy regression; git diff --check
Evidence: commits 794f933, 9bfa186, cfc2999; GitHub Actions run 32694540606 success including normalized parser fuzz, exact pinned Bazel protocol test, observer helper and Linux gVisor lifecycle
Limitations: richer optional detection은 M11 범위
Next: M8-005 — IN_PROGRESS / Ready: Yes
```

### M8-005 — least-privilege typed resolver network-policy helper

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Inspection, Sandbox and Policy Architecture](../architecture/03-inspection-sandbox-policy.md)

**Acceptance**

- ordinary-user `helox`와 root-owned network helper/service를 분리하고 helper
  authority를 필요한 network administration 범위로 제한한다.
- API는 create/verify/remove resolver policy typed operation만 받고 package name,
  hostname, shell fragment와 raw iptables/nft arguments를 받지 않는다.
- peer, session, HAA-owned network/subnet, approved public IP set와 default-deny
  rule을 독립적으로 검증한다.
- unauthorized peer, arbitrary operation, duplicate/stale session, helper crash와
  apply/verify/cleanup uncertainty가 fail-closed다.
- Linux integration은 one-time privileged service setup 후 실제 workflow를
  ordinary user로 실행한다. `sudo helox`는 supported UX가 아니다.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/security; typed protocol fuzz/authorization tests; privileged Linux integration; cleanup regression; git diff --check
Evidence: commits 80a933a, 06f6051, f861b61, 05efa0e, f71fa90, 92fb590, 5eeed45, 731c130, 731e927; GitHub Actions run 32702853334 Required success including ordinary-user privileged policy workflow
Limitations: arbitrary Host firewall administration과 sudo helox는 지원하지 않음
Next: M8-006 — IN_PROGRESS / Ready: Yes
```

### M8-006 — runtime lock single source of truth·CI integration

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [Quality Toolchain](../engineering/03-quality-toolchain.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Acceptance**

- canonical runtime/observer lock이 gVisor, runsc, Bazel, Docker와 Node/Python
  identity의 유일한 수동 입력이다.
- product code와 CI는 lock 또는 deterministic generated output을 사용하며
  generated drift validator가 required gate에 포함된다.
- workflow와 workflow checker의 hand-copied runtime identity를 제거하고 lock
  corruption, unknown field, missing platform, code/workflow mismatch를
  fail-closed test한다.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/platform/security; lock schema/drift/workflow validator; git diff --check
Evidence: commit dac187b; canonical quick/docs/platform/security and focused race checks passed locally
Limitations: release signing identity와 published artifact manifest는 M10 범위
Next: M8-007 — NOT_STARTED / Ready: Yes
```

### M8-007 — hostile-boundary regression·Linux security requalification

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)
- [MVP Results, Policy and Completion](../mvp-scope/03-results-policy-and-completion.md)

**Acceptance**

- hostile PATH/environment/tool/socket/container attribution/privilege/cleanup
  matrix와 production bootstrap E2E가 expected fail-closed result를 낸다.
- canonical quick/docs/security/vulnerability/race와 Linux gVisor observer,
  npm/PyPI resolver/dynamic/Promotion integration이 current main candidate에서
  모두 green이다.
- known Critical/High production trust-boundary defect가 없고 limitation 및
  unsupported multi-process/Host path를 README에 정확히 공개한다.
- completion evidence와 CI run을 기록한 뒤에만 production-ready release
  blocker 해제를 선언한다. M7 evidence는 그대로 보존한다.

**Status**

```text
Status: COMPLETE
Checks: complete canonical suite; race; hostile-boundary matrix; pinned Linux production composition; Required CI
Evidence: commit dac187b; GitHub Actions run 32705277070 Required success including gVisor Linux lifecycle, observer Bazel helper, macOS, Minimum Go, Security and Vulnerability
Limitations: M9 product UX, M10 verified distribution과 M11 detection depth는 별도 milestone
Next: M9-001 — NOT_STARTED / Ready: Yes
```

### M7-002 — ecosystem flow·fixture regression

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Inspection·Sandbox·Policy Contract](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Evidence·Staging·Promotion Contract](../domain-model/09-contract-evidence-staging-promotion.md)

**Completion**

```text
Status: COMPLETE
Checks: go test ./...; go test -race ./internal/application ./internal/policy; canonical quick; git diff --check
Evidence: internal/application/m7_qualification_test.go; exact source-neutral identity/digest binding across npm, PyPI and GitHub Release; non-ALLOW Promotion stop; acquisition failure has no Policy decision
Limitations: existing adapter-owned format/runtime tests remain the source of detailed archive and Linux evidence; Host-platform qualification is M7-003
Next: M7-003 — NOT_STARTED / Ready: Yes
```

### M7-003 — Linux·macOS·WSL2 CLI qualification

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [MVP ecosystems and platforms](../mvp-scope/01-ecosystems-platforms-artifacts.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Acceptance**

- Linux and macOS each build the native `helox` executable and execute its
  default help path without network, Artifact acquisition or dynamic-backend
  claims.
- WSL2 evidence comes from an approved disposable Windows/WSL2 environment;
  the retained evidence identifies the environment and preserves command
  output. Ordinary Linux CI and unsupported nested virtualization are not a
  substitute.
- the published capability matrix keeps dynamic inspection limited to the
  pinned Linux backend and preserves non-`ALLOW` behavior for unsupported
  host/backend combinations.

**Status**

```text
Status: COMPLETE
Checks: GitHub Actions run 32443146082 Required aggregate (Quick, Docs, Minimum Go, macOS, gVisor observer and Linux integration) successful; local macOS go test ./...; canonical quick; CLI race test; disposable Windows 11 VM WSL2 qualification command
Evidence: cmd/helox/main_test.go builds ./cmd/helox and runs no argument plus --help; scripts/qualify-wsl2-cli.sh; external WSL2 record: Windows 11 Enterprise Evaluation VM → Ubuntu 24.04.4 WSL2, Go 1.26.5, no-argument and --help output, source a3e5ced2d2dd4254e97470a5a385ea03ee86160a
Limitations: WSL2 is a Linux CLI Host path only; it does not claim Windows-native or macOS-native dynamic inspection
Next: M7-004 — IN_PROGRESS / Ready: Yes
```

### M7-004 — evidence·result·resilience qualification

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [Evidence·Staging·Promotion Contract](../domain-model/09-contract-evidence-staging-promotion.md)
- [Evidence, Staging and Promotion Architecture](../architecture/04-evidence-staging-promotion.md)
- [MVP results, Policy and completion](../mvp-scope/03-results-policy-and-completion.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- Evidence references, machine/human results, Manifest/SBOM, staged bytes and
  published bytes retain one exact normalized identity/digest chain.
- interruption, retry and cleanup tests prove no partial Evidence, staging or
  target success is exposed; cleanup uncertainty remains non-success.
- completed records are retained for audit/retry without adding automated
  retention/garbage collection; fixtures and results contain no secret or raw
  Host/observer data.

**Status**

```text
Status: COMPLETE
Checks: go test ./...; go test -race -timeout=5m ./...; go run ./scripts/check quick; go run ./scripts/check docs; git diff --check; GitHub Actions run 32476129928 Required aggregate successful
Evidence: duplicate Evidence batch preflight and private temporary directory commit prevent partial Run publication; retry preserves committed Evidence; human/JSON results suppress raw causes; staged/published Manifest, SBOM and artifacts rehash to one identity/digest chain
Limitations: this qualification adds no source, format, retention daemon, product configuration, CLI command, remote store or synthetic success dependency
Next: M7-005 — IN_PROGRESS / Ready: Yes
```

### M7-005 — security workflow·scheduled gate qualification

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [Quality Toolchain](../engineering/03-quality-toolchain.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- pinned gosec, Gitleaks and govulncheck install outside the product module
  graph with integrity and identity checks; tool/bootstrap/scanner failure is
  non-success.
- PR required CI runs source-security and vulnerability profiles and the
  aggregate includes every active required child fail-closed.
- a least-privilege weekly/manual security workflow scans default-branch
  reachable history, refreshes vulnerability data and executes only bounded
  existing fuzz targets; each real finding path is regression-tested with
  synthetic data only.

**Status**

```text
Status: COMPLETE
Checks: Go 1.26.7 local fuzz/quick/docs; GitHub Actions run 32555127293 Required aggregate successful; default-branch Scheduled Security run 32556503406 successful
Evidence: exact external tool pins and integrity checks; independent required Security/Vulnerability jobs; fail-closed aggregate; full-history secret scan; refreshed govulncheck; four 5-second fuzz targets with bounded one-minute process deadlines
Limitations: scheduled execution reports findings but does not retroactively change an older merge result; Gitleaks v8.18.4 remains an exceptional last-known-functional pin pending upstream rule repair and re-evaluation
Next: M7-006 — IN_PROGRESS / Ready: Yes
```

### M7-006 — MVP final audit·documentation·completion

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [MVP results, Policy and completion](../mvp-scope/03-results-policy-and-completion.md)
- [Inspection·Sandbox·Policy Architecture](../architecture/03-inspection-sandbox-policy.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)
- [Quality Toolchain](../engineering/03-quality-toolchain.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Acceptance**

- re-audit all Artifact, Sandbox, Evidence, Staging and Promotion trust
  boundaries and leave no known Critical/High product security defect or
  unresolved invariant violation.
- run the complete qualification suite and retain current reproducible Linux,
  macOS, WSL2, gVisor and scheduled-security evidence.
- publish installation, supported-path, limitation, troubleshooting and
  security-reporting guidance without overstating unsupported capabilities.
- record MVP completion only after every prior M7 item and final evidence is
  current and green.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/security/vulnerability; go test ./...; go test -race -timeout=15m ./...; full-rule gosec review; git diff --check; GitHub Actions run 32557123229 all children and Required aggregate successful
Evidence: resolver container/firewall cleanup commands have independent bounded contexts; Evidence/GitHub redirect close failures remain fail-closed; README publishes exact build/command/support/runtime/troubleshooting boundaries; SECURITY.md defines private reporting; prior M7 Linux/macOS/WSL2/scheduled-security evidence remains current
Security audit: all full-rule gosec findings reviewed; G115 conversions are guarded by explicit nonnegative and size limits, G122/G304 paths remain under trusted Host roots with symlink/type/digest revalidation, G204 commands use fixed adapter-owned executables/arguments, and G302 modes are owner-only directory/executable modes; no known Critical/High product defect or unresolved trust-boundary violation remains
Limitations: full automatic dynamic inspection and Promotion are qualified only on the pinned Linux amd64 Docker/runsc/observer path; macOS and WSL2 evidence is CLI-only; no general Host runtime installer, native Windows/macOS dynamic backend, automatic retention daemon, authenticated private source or absolute-safety guarantee is claimed
Next: MVP qualification complete; post-MVP work requires a new explicit queue decision
```

### M7-001 — MVP qualification entry·evidence matrix

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [M7 milestone scope and exit criteria](./01-milestones.md)
- [MVP ecosystems and platforms](../mvp-scope/01-ecosystems-platforms-artifacts.md)
- [MVP inspection and verification](../mvp-scope/02-inspection-and-verification.md)
- [MVP results, Policy and completion](../mvp-scope/03-results-policy-and-completion.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Acceptance**

- M7 exit criteria, fixture outcomes, platform evidence, result resilience and
  security gates are assigned to one ordered M7 item each.
- unavailable, unsupported, skipped and incomplete evidence cannot be treated
  as qualification success or automatic `ALLOW`.
- this entry creates no product implementation, runtime, dependency, CLI
  command or CI job.

**Completion**

```text
Status: COMPLETE
Checks: canonical docs; git diff --check
Decision/Evidence: docs/planning/10-m7-mvp-qualification-contract.md; explicit six-item fail-closed evidence matrix
Limitations: WSL2 evidence and scheduled security workflow remain M7 work; neither is claimed as green
Next: M7-003 — NOT_STARTED / Ready: Yes
```

M5-007 is complete. M6-001 resolves exact GitHub release asset identity,
provenance, target and standalone Promotion boundaries before implementation.

### M6-001 — GitHub Releases Standalone entry decision

**Read**

- [M6 GitHub Releases Standalone Contract](../domain-model/15-m6-github-releases-contract.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Inspection·Sandbox·Policy Contract](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Evidence·Staging·Promotion Contract](../domain-model/09-contract-evidence-staging-promotion.md)
- [M6 milestone scope and exit criteria](./01-milestones.md)

**Acceptance**

- exact public repository/release/tag/asset selector, API identity/digest,
  rate-limit and no-ambient-auth boundary are explicit.
- artifact format/static/dynamic capability, sidecar/provenance, target and
  no-re-resolution Promotion rules preserve existing common invariants.
- M6 implementation queue is one-item-at-a-time and M6-001 adds no source,
  runtime, dependency, CLI command or CI job.

**Completion**

```text
Status: COMPLETE
Checks: official GitHub Release/API/rate-limit/attestation review; canonical quick/docs; git diff --check
Decision/Evidence: docs/domain-model/15-m6-github-releases-contract.md; public exact tag+asset selector; API SHA-256 and no ambient auth; generic Manifest/Staging; no re-resolution Promotion
Limitations: M6-001 creates no product implementation; sidecar signature/checksum/attestation auto-verification is deferred until a typed verifier contract exists
Next: M6-002 — IN_PROGRESS / Ready: Yes
```

### M6-002 — exact public input·API boundary foundation

**Read**

- [M6 GitHub Releases Standalone Contract](../domain-model/15-m6-github-releases-contract.md)
- [Artifact Identity](../domain-model/02-artifact-identity.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Coding / Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- `owner/repo@tag#asset` parser accepts only one bounded exact selector and
  rejects latest, omitted fields, URL/path/query/fragment and ambiguous input.
- public API response boundary retains only exact release/asset ID, tag,
  uploaded state, declared SHA-256, declared size, immutable marker and
  publication time; missing/ambiguous data is rejected.
- no HTTP acquisition, ambient auth, CLI command, runtime or product dependency
  is introduced before M6-003.

**Completion**

```text
Status: COMPLETE
Commit: pending
Checks: focused unit/race; 5-second parser fuzz; canonical quick/docs; git diff --check
Decision/Evidence: internal/artifact/githubrelease exact selector and bounded Release metadata parser; GitHub REST API base/version constants; latest/unsafe input fail-closed
Limitations: API transport, controlled acquisition, integrity verification, static/dynamic inspection and CLI remain later M6 items
Next: M6-003 — IN_PROGRESS / Ready: Yes
```

### M6-003 — GitHub Release resolve·Acquire·integrity

**Read**

- [M6 GitHub Releases Standalone Contract](../domain-model/15-m6-github-releases-contract.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Verification·Evidence·Finding](../domain-model/04-verification-evidence-finding.md)
- [Coding / Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- Resolve uses only the pinned public release-by-tag API and freezes the exact
  repository, release ID, tag, asset ID, asset name, size and SHA-256.
- Acquire uses only the resolved numeric asset API media flow, bounded trusted
  redirects and a Run-private controlled intake directory; no auth, proxy or
  caller-controlled direct URL is used.
- observed SHA-256 and byte size must equal the API declaration before an
  acquired Artifact can exist; mismatch, stream/redirect failure and cleanup
  uncertainty fail closed.
- the source-specific verifier reports the declared SHA-256 check through the
  common Verification port.

**Completion**

```text
Status: COMPLETE
Checks: focused unit/race; 5-second selector fuzz; canonical quick/docs; git diff --check
Evidence: internal/artifact/githubrelease controlled API Resolve/asset-ID Acquire adapter; internal/verification/githubrelease declared SHA-256 verifier
Limitations: static format analysis, M6 Policy, Linux ELF dynamic inspection, Promotion and CLI remain later M6 items
Next: M6-005 — NOT_STARTED / Ready: Yes
```

### M4-003 — recursive dependency inspection과 set-level Policy

**추가 확정 scope**

- isolated npm resolver의 package-lock v3 parser와 graph 전체의 독립 Run/Evidence/entry Policy를 구현한다.
- resolver 전용 Docker network-policy helper가 default-deny egress policy를 create → apply → verify → cleanup하고, npm resolution에 필요한 검증된 registry endpoint 집합만 허용한다.
- Docker 일반 bridge fallback, Host 사전 firewall/proxy 의존, proxy 환경변수-only enforcement는 허용하지 않는다.
- Linux Docker firewall backend probe, policy apply/verify/cleanup failure와 runtime endpoint mismatch는 fail-closed로 graph를 반환하지 않는다.

### M4-004 — Verified Manifest·SBOM과 trusted Staging

**Completion**

```text
Status: COMPLETE
Commit: e43b780
Checks: canonical quick; focused race; git diff --check
Decision/Evidence: deterministic helox.verified-manifest/v1; CycloneDX 1.7; Quarantine-to-Staging rehash; exclusive/fsync/read-only/atomic filesystem tests
Limitations: offline npm install·target publish는 M4-005 범위; staging retention/GC 없음
Next: M4-005 — IN_PROGRESS / Ready: Yes
```

### M4-005 — offline npm Promotion과 atomic target

**Completion**

```text
Status: COMPLETE
Commit: ec05e2b
CI: GitHub Actions run 31784626998 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: canonical quick; focused race; real pinned Docker offline npm ci Promotion integration; git diff --check
Decision/Evidence: Manifest-bound target-local tarballs and generated local-file lock; --network none/--pull never; post-install exact-set validation; Linux renameat2(RENAME_NOREPLACE) publish
Limitations: Linux amd64 automatic Promotion only; scripts/bin links and existing/global/workspace targets are unsupported; no staging retention/GC
Next: M4-006 — IN_PROGRESS / Ready: Yes
```

### M4-006 — M4 qualification과 M5/M6 handoff

**Outcome**

M4 exit criteria와 fail-closed install/Promotion 경계를 전체 evidence에 대조하고 다음에는 M5 entry decision 하나만 열 수 있게 한다.

**Acceptance**

- exact inspected/allowed/Manifest/Staging/Promotion set 동일성과 no mutable re-resolution/no network를 재현 가능한 test와 Linux CI로 확인한다.
- `MANUAL_REVIEW`/`BLOCK`, staging/Promotion failure, existing target와 publish race에서 target 불변/no partial publish를 확인한다.
- result/Evidence/SBOM/cleanup 계약, minimum Go/macOS limitation과 활성 CI gate를 audit한다.
- M4 qualification record와 현재 stage를 갱신하고 M5 구현 없이 M5-001 entry decision 하나만 handoff한다.

**Completion**

```text
Status: COMPLETE
Commit: 1dc22b8
CI: GitHub Actions final qualification run 31786209026 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: canonical quick/docs/platform; go test -race -timeout=5m ./...; JSON Draft 2020-12 metaschema; M4 exit-criteria and production-boundary audit; git diff --check
Decision/Evidence: docs/planning/07-m4-qualification.md; exact set/Manifest/SBOM/Staging/Promotion binding; offline no-network target publish; fail-closed non-ALLOW and operational failure results
Limitations: automatic Promotion is Linux amd64/new empty target/script-free only; macOS and other platforms do not enter resolver Host commands; completed Staging retention/GC remains manual
Next: M5-001 — NOT_STARTED / Ready: Yes
```

## 4. Work Item

### M5-001 — PyPI/pip Expansion entry decision

**Outcome**

M5가 기존 generic Artifact/Verified Set/Promotion workflow를 유지한 채 PyPI wheel과 sdist를 연결하도록 exact identity, resolver, build, dynamic inspection과 offline Promotion contract를 확정한다. 이 항목은 implementation을 시작하지 않는다.

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [M4 npm Install and Promotion Contract](../domain-model/13-m4-npm-install-promotion-contract.md)
- [Dependency·Verified Set·Manifest](../domain-model/05-dependency-verified-set.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Verification·Inspection·Sandbox·Policy Contract](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Evidence·Staging·Promotion Contract](../domain-model/09-contract-evidence-staging-promotion.md)
- [M5 milestone scope and exit criteria](./01-milestones.md)

**Acceptance**

- PyPI project/version normalization, selected wheel/sdist identity, declared/observed digest and Linux amd64 target-tag rule가 unambiguous하다.
- Simple API, pip report, candidate graph, default-deny resolver egress와 unsupported/dynamic resolution fail-closed semantics가 정해져 있다.
- wheel static/dynamic inspection, sdist PEP 517 build, derived-wheel reinspection과 no-Host-execution boundary가 정해져 있다.
- generic Verified Set/Manifest/SBOM/Staging/offline Promotion invariant와 existing result boundary를 유지하는 방법이 정해져 있다.
- M5 구현 queue가 한 항목씩 수행 가능한 크기로 정해지고, M5-001에서 source, runtime, dependency, CI job 또는 placeholder가 생성되지 않는다.

**Completion**

```text
Status: COMPLETE
Checks: official PyPA/pip specification review; canonical documentation routing; git diff --check
Decision/Evidence: docs/domain-model/14-m5-pypi-pip-contract.md
Limitations: Python/pip runtime identity, PyPI resolver, wheel/sdist handling, Docker/gVisor execution, CLI and CI implementation 없음
Next: M5-002 — NOT_STARTED / Ready: Yes
```

### M5-002 — Python/pip runtime identity lock과 PyPI reference foundation

**Outcome**

M5 automatic path가 사용할 Linux amd64 CPython/pip resolver·Promotion runtime의 exact identity와 public PyPI project reference parser/capability boundary를 구현한다.

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [Engineering — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)
- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)

**Work**

- current stable CPython/pip releases와 official image provenance를 조사해 exact version, OCI image digest, binary metadata와 capability probe input을 lock한다.
- Linux amd64 / fixed interpreter-ABI-platform tag set을 explicit capability로 만들고 non-Linux 또는 identity mismatch가 resolver/Docker execution으로 bypass되지 않게 한다.
- `project[@exact-version]` parser를 name/PEP 440 canonicalization, unsupported extras/range/marker/URL/local-version rejection과 bounded sanitized error로 구현한다.
- product module에 PyPI/pip library dependency를 추가하지 않는다. network resolver, pip report, Docker policy, wheel/sdist parser와 CLI command는 다음 항목 범위다.

**Acceptance**

- floating Python/pip/image tag가 없고 runtime mismatch/unsupported platform이 explicit non-ALLOW capability다.
- parser가 exact supported reference만 normalized value로 만들며 raw input/environment/path를 public result에 누출하지 않는다.
- current product minimum Go, architecture, format/static analysis, deterministic unit/fuzz boundary tests와 existing npm regression이 통과한다.

**Completion**

```text
Status: COMPLETE
Checks: official Python/pip/Docker source review; focused unit/race; 5-second parser fuzz; canonical docs/quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: Python 3.14.7 + pip 26.2.1 immutable Docker Official Image lock; Linux amd64 cp314/cp314/manylinux_2_36_x86_64 target basis; bounded public PyPI reference parser and explicit capability probe
Limitations: PyPI Simple API/resolver/report, resolver egress policy, wheel/sdist handling, Docker/gVisor Python execution, CLI and CI integration 없음
Next: M5-003 — NOT_STARTED / Ready: Yes
```

### M5-003 — isolated PyPI resolver·target tag·egress policy

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)
- [Engineering — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- pinned Python/gVisor resolver에서 pip installation report v1을 bounded parse하고 Simple API JSON v1 metadata와 exact cross-check한다.
- fixed target compatible-tag closure를 verify하고, wheel-only M5-003 resolver profile에서 unsupported sdist/marker/extras/direct requirement를 fail-closed 한다.
- `pypi.org`와 `files.pythonhosted.org` public IP set만 default-deny network policy·explicit container host mapping에 적용하고, observer lifecycle까지 completion requirement로 둔다.
- Linux controlled integration/required CI에서 immutable Python image와 exact helper/runtime path를 실행한다.

**Acceptance**

- report/Simple URL·filename·SHA-256·Requires-Python/yanked/size, runtime/tag, egress endpoint 및 observer stream 중 어느 하나라도 incomplete면 graph가 반환되지 않는다.
- Core/Application/Policy에는 PyPI raw report/index/URL query/pip output/runtime implementation detail가 유입되지 않는다.
- focused parser/resolver unit·race, existing regression, canonical quick/docs와 controlled Linux PyPI integration이 통과한다.

**Completion**

```text
Status: COMPLETE
Commits: 22584ca, 79524d1, a82e348, aaaec7b, d7ec08d
CI: GitHub Actions run 32335942809 — gVisor Observer Helper, Linux PyPI integration, Quick, Minimum Go, macOS, Docs, Required success
Checks: focused unit/race; parser fuzz; canonical quick/docs; official PyPI Simple API v1 URL/hash validation; git diff --check
Decision/Evidence: pinned Python 3.14.7/pip 26.2.1; isolated runsc-trace resolver; default-deny PyPI endpoint policy; observer lifecycle and report/Simple cross-check
Limitations: wheel intake/integrity/static inspection, dynamic inspection, sdist build, staging/Promotion and CLI remain in later M5 items
Next: M5-004 — IN_PROGRESS / Ready: Yes
```

### M5-004 — wheel intake·integrity·static inspection

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)
- [Engineering — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Engineering — Quality Toolchain](../engineering/03-quality-toolchain.md)

**Work**

- exact resolver-selected wheel을 controlled intake에서 bounded ZIP reader로 읽고 observed SHA-256과 declared digest를 분리 검증한다.
- ZIP containment, compressed/uncompressed size·file count bound, regular-file-only, symlink/special/path escape/archive bomb를 fail-closed로 검사한다.
- normalized project/version과 `.dist-info/METADATA`, `.dist-info/WHEEL`, `.dist-info/RECORD`의 정확한 일치, supported Wheel-Version·target tag를 검증한다.
- RECORD의 permitted empty self-entry 외 모든 파일 digest/size, duplicate/unrecorded installable content, `.data/scripts`와 native extension surface를 bounded normalized evidence로 만든다.
- Core Metadata의 Requires-Dist, Requires-Python, Import-Name, entry points, license/provenance를 제한된 adapter evidence로 보존하고 Core/Policy에 raw wheel payload를 유입하지 않는다.

**Acceptance**

- malformed/ambiguous/mismatched wheel, path escape, archive bomb, symlink/special file, RECORD 누락/불일치, unsupported tag는 artifact 또는 static result를 반환하지 않는다.
- exact filename·normalized identity·observed/declared SHA-256·target compatibility와 bounded static evidence가 deterministic하게 재현된다.
- focused unit/property/fuzz 경계, race, existing npm/PyPI regression, canonical quick/docs와 architecture gate가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 6575594
CI: GitHub Actions run 32338211203 — gVisor Observer Helper, Linux integration, Quick, Minimum Go, macOS, Docs, Required success
Checks: go test ./...; go test -race -timeout=5m ./...; 5-second wheel parser fuzz; canonical docs/quick; git diff --check
Decision/Evidence: bounded non-executing ZIP intake; observed/declared SHA-256; target/WHEEL tag agreement; dist-info and RECORD digest/size containment; bounded static surface evidence
Limitations: dynamic wheel inspection, sdist build/derived wheel, staging/Promotion and CLI remain in later M5 items
Next: M5-005 — IN_PROGRESS / Ready: Yes
```

### M5-005 — Python dynamic inspection·PEP 517 sdist build

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Engineering — Coding / Security Rules](../engineering/02-coding-security-rules.md)

**Work**

- verified wheel만 fresh `runsc-trace` session에 private tmpfs로 반입하고 `pip --no-index --no-deps`로 설치한 뒤, static inspection이 확정한 bounded import surface만 실행한다.
- observer attribution, stream completion, command exit, resource limit, introduction/cleanup failure가 모두 non-ALLOW incomplete로 정규화되게 한다. artifact script, entry point, arbitrary module, Host Python은 실행하지 않는다.
- `.tar.gz` PEP 517 sdist를 bounded static reader로 검증하고 exact build-system requirement와 in-tree backend 경계를 정규화한다. legacy `setup.py`, malformed backend path, graph 밖 build requirement는 automatic path에서 제외한다.
- verified source와 already verified build wheel만 private/no-network gVisor session에서 `pip wheel --no-index --no-deps --no-build-isolation`으로 build한다. one derived wheel, source/build recipe binding, re-inspection boundary를 fail-closed로 만든다.

**Acceptance**

- wheel dynamic run은 verified local content와 declared import surface 이외를 실행하지 않으며 observer/lifecycle 불완전 시 completed inspection을 반환하지 않는다.
- sdist path/archive/metadata/build backend/build requirement/output wheel mismatch 및 dynamic dependency expansion은 derived wheel을 반환하지 않는다.
- raw pip/backend output, Host path, credentials, gVisor implementation detail는 Core/Application/Policy/public result에 유입되지 않는다.
- focused unit/fuzz/race, existing npm/PyPI regression, canonical quick/docs 및 controlled Linux integration이 통과한다.

**Completion**

```text
Status: COMPLETE
Commits: f442042, 2e76b0b, 18a8f4a, 9ae8c55, 039ad58
CI: GitHub Actions run 32343138560 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: focused unit/fuzz/race; canonical quick/docs; verified local wheel dynamic import; PAX sdist PEP 517 no-network derived-wheel build and static reinspection; git diff --check
Decision/Evidence: fresh runsc-trace session; trusted observer completion; controller-generated valid local artifact filenames; derived-wheel identity/source-build recipe binding; bounded raw-output discard and test diagnostics
Limitations: generic staging/offline pip Promotion, CLI/bootstrap/result wiring and final M5 qualification remain M5-006/M5-007 scope
Next: M5-006 — IN_PROGRESS / Ready: Yes
```

### M5-006 — generic staging·offline pip Promotion·CLI integration

**Completion**

```text
Status: COMPLETE
Commits: 4b96ab6, 110502b, 1bc6317, 4d99cee, 8f4f38b, 8eb949b
CI: GitHub Actions run 32349328808 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: focused race; canonical quick/docs; real pinned Docker offline pip Promotion; gVisor PEP 517 derived-wheel build; git diff --check
Decision/Evidence: exact primary PyPI inspect CLI; generic source→derived graph binding; controller SHA-256/filename record; derived static/dynamic reinspection; no-network hash requirements Promotion and post-install RECORD validation
Next: M5-007 — IN_PROGRESS / Ready: Yes
```

### M5-007 — M5 qualification·npm regression·M6 handoff

**Acceptance**

- M5 wheel/sdist exact identity, source-to-derived binding, Sandbox-only build,
  complete per-entry inspection and unsupported capability fail-closed behavior
  are audited against the canonical contract.
- existing npm inspect/install/Promotion regression remains green without
  changing npm semantics.
- canonical Quick/Docs/Minimum Go/macOS and controlled Linux gVisor/observer,
  npm resolver and PyPI resolver/dynamic/build/Promotion integration gates pass.
- qualification evidence and current stage are recorded; M6 entry decision is
  the only next work item and no M6 implementation is introduced.

**Completion**

```text
Status: COMPLETE
Commit: 42858b1
CI: GitHub Actions workflow_dispatch run 32352463714 — Quick, Docs, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: security-boundary audit; focused unit/race; canonical quick/docs; full go test -race -timeout=5m ./...; git diff --check; existing npm regression; controlled Linux PyPI resolver, dynamic wheel, PEP 517 sdist build, npm resolver and PyPI Promotion integration
Decision/Evidence: docs/planning/08-m5-qualification.md; complete ALLOW build-input boundary; derived-wheel build Evidence and generic source/input/config Manifest binding; npm semantics unchanged; pinned Linux runtime and observer qualification
Limitations: automatic PyPI Promotion remains Linux amd64/new empty target only; M6 GitHub Releases implementation is not started; M6 entry decision is next
Next: M6 entry decision — NOT_STARTED / Ready: Yes
```

## 5. Historical work items

### M3-001 — Linux Dynamic Inspect entry decision

**Outcome**

Linux dynamic inspection의 runtime identity, Session·observation boundary, resource limit, M3 Policy direction과 fixture를 확정한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)

**Completion**

```text
Status: COMPLETE
Checks: official gVisor/Docker/Node source review; gVisor release-20260810.0 annotated tag commit verification; canonical docs; git diff --check
Decision/Evidence: docs/domain-model/12-m3-linux-dynamic-contract.md
Limitations: current macOS workstation is unsupported for Linux runtime installation; no M3 package, CI job or runtime config created
Next: M3-002 — Ready: Yes
```

### M3-002 — gVisor runtime lock과 capability probe

**Outcome**

Linux-only gVisor runtime installer identity와 fail-closed capability probe를 current worktree에 필요한 범위로 구현한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Engineering — Coding and Security Rules](../engineering/02-coding-security-rules.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- release-20260810.0 binary/archive SHA-512와 Node image digest를 runnable lock input으로 record한다.
- Linux host/kernel/Docker/runsc/runtime/image capability를 probe하고 unsupported/failure를 explicit result로 구분한다.
- tool install과 image pull은 Linux integration environment만 수행하도록 하고 current macOS/default Quick profile에 실행 경로를 만들지 않는다.
- probe unit/contract test와 architecture gate를 추가한다.

**Acceptance**

- floating latest/tag runtime or image identity가 없다.
- macOS/non-Linux invocation이 hidden success가 아니라 explicit unsupported capability다.
- probe success가 Docker CLI 존재만으로 sandbox safety를 주장하지 않고 runsc/runtime/image identity를 확인한다.
- format, unit, static analysis와 architecture check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 5a7050a
Checks: canonical quick; focused race test; git diff --check
Decision/Evidence: bounded Session ID/status/observation types and consumer-owned Sandbox Port
Limitations: backend, observation collector, dynamic inspector and Policy wiring 없음
Next: M3-004 — Ready: Yes
```

### M3-004 — gVisor Sandbox Session lifecycle

**Outcome**

pinned gVisor runtime을 사용하는 Linux one-shot Session backend의 create/execute/terminate/dispose lifecycle과 cleanup을 구현한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)

**Work**

- injected process boundary로 gVisor Docker runtime command와 bounded cleanup을 구현한다.
- no bind mount, non-root, no-new-privileges, read-only rootfs, network none, resource limit command invariants를 test한다.
- one-time controlled Artifact introduction, timeout/forced termination/disposal와 non-reuse result를 구현한다.
- macOS/unsupported probe result가 backend execution으로 bypass되지 않게 한다.

**Acceptance**

- backend는 Finding/Evidence/Policy를 만들지 않고 raw `SandboxResult`만 반환한다.
- timeout, process failure와 cleanup failure는 completed result로 변환되지 않는다.
- untrusted Artifact에 Host path/socket/environment/credential을 전달하는 command option이 없다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: a3c731e
Checks: canonical quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: injected Docker command boundary; constrained lifecycle unit tests
Limitations: Linux gVisor runtime integration and observer trace ingestion remain unavailable on this macOS workstation
Next: M3-005 — Ready: Yes
```

### M3-003 — Sandbox Session·Observation Domain과 Port

**Outcome**

Dynamic runtime의 raw Session/Observation lifecycle을 ecosystem-neutral Domain과 consumer-owned Port로 정의한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Domain — Verification, Evidence, and Finding](../domain-model/04-verification-evidence-finding.md)

**Work**

- bounded Session identity, execution state, raw observation category와 limitation type을 정의한다.
- `core/ports`에 dynamic inspection consumer가 필요한 Sandbox contract를 추가한다.
- Sandbox가 Finding/Policy/Evidence reference를 직접 만들 수 없도록 type/API를 제한한다.
- constructor/state/Port contract test와 architecture gate를 추가한다.

**Acceptance**

- raw observation과 normalized Finding/Evidence/Policy Decision이 같은 type이나 API에서 섞이지 않는다.
- timeout/limit/observer failure는 completed clean result가 아니라 bounded non-completed execution으로 표현된다.
- Domain은 runtime/Docker/gVisor/package-manager type을 import하지 않는다.
- format, unit, static analysis와 architecture check가 통과한다.

### M3-005 — gVisor raw observation collector

**Outcome**

trusted gVisor trace observer 입력을 bounded raw `SandboxObservation`으로 변환하고 Session lifecycle에 연결한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)

**Work**

- trusted observer transport와 trace record의 bounded adapter contract를 정의한다.
- process/filesystem/network/honeytoken/resource observation을 raw bounded category/subject로 정규화한다.
- trace overflow, observer failure와 malformed record를 completed clean result가 아닌 explicit incomplete limitation으로 처리한다.
- artifact output, raw path, argv, environment, contents가 normal result에 포함되지 않는 contract test를 추가한다.

**Acceptance**

- Sandbox가 Finding/Evidence/Policy를 만들지 않고 raw observation과 execution limitation만 반환한다.
- observer가 Artifact process와 분리된 trusted boundary이고 artifact-controlled endpoint가 아니다.
- trace size/event bound와 failure reason이 deterministic하게 검증된다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: pending current commit
Checks: canonical quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: bounded trace collector and trusted observer-before-introduction lifecycle test
Limitations: a Linux gVisor seccheck transport implementation remains an integration-environment concern
Next: M3-006 — Ready: Yes
```

### M3-006 — Dynamic observation normalization

**Outcome**

M3 raw Sandbox observation을 Inspection의 bounded Finding/Evidence와 required dynamic check 상태로 정규화한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Domain — Verification, Evidence, and Finding](../domain-model/04-verification-evidence-finding.md)

**Work**

- M3 observation category를 documented Finding/Evidence와 incomplete execution 상태로 매핑한다.
- raw trace subject/payload가 Finding/Evidence summary에 전파되지 않게 한다.
- dynamic inspector Port/adapter를 구성하되 Sandbox와 Policy의 책임 경계를 유지한다.
- clean, suspicious, blocking, timeout/observer failure fixture를 deterministic하게 test한다.

**Acceptance**

- Sandbox는 Finding/Evidence/Policy를 직접 만들지 않는다.
- incomplete/unsupported dynamic check가 clean success나 ALLOW로 변환되지 않는다.
- 결과가 raw Host path, argv, environment, file contents, trace payload를 포함하지 않는다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: pending current commit
Checks: canonical quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: npm DynamicInspector converts raw M3 observations without carrying trace payloads
Limitations: static and dynamic inspection report composition and M3 Policy wiring are next work
Next: M3-007 — Ready: Yes
```

### M3-007 — M3 Policy와 npm inspect wiring

**Outcome**

정적·dynamic inspection을 한 M3 npm result로 구성하고 M3 Policy v1/CLI controlled vertical test를 연결한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)

**Work**

- static/dynamic Inspection report를 subject-preserving composite inspector로 구성한다.
- `m3-npm-dynamic-inspect` Policy v1을 dynamic required check 상태와 Finding taxonomy에 연결한다.
- production wiring은 unsupported Linux dynamic capability를 success로 바꾸지 않게 한다.
- controlled clean/manual/block/incomplete CLI vertical fixture를 추가한다.

**Acceptance**

- 모든 required verification/static/dynamic check가 complete인 clean fixture만 M3 ALLOW가 된다.
- dynamic incomplete/unsupported는 MANUAL_REVIEW이며 raw runtime payload가 CLI/Evidence에 없다.
- Honeytoken/filesystem violation은 BLOCK, network/unexpected process는 MANUAL_REVIEW로 재현된다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: pending current commit
Checks: canonical quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: M3 composite inspection/Policy and controlled allow/manual/block vertical fixtures
Limitations: current macOS composition explicitly returns M3_LINUX_ONLY; a production Linux runsc trace transport needs Linux integration validation
Next: M3-008 — Ready: Yes
```

### M3-008 — M3 qualification과 M4 handoff

**Outcome**

M3 구현 범위와 fail-closed runtime limitation을 qualification으로 기록하고 M4 entry decision 하나로 handoff한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [M3 milestone](./01-milestones.md#7-m3--linux-dynamic-inspect)

**Work**

- M3 source/contract/quality gates와 controlled vertical fixtures를 audit한다.
- macOS runtime limitation과 Linux integration validation requirement를 명시한다.
- M4 entry decision만 Ready 상태로 열고 future install/promotion source나 CI placeholder는 만들지 않는다.

**Acceptance**

- unsupported runtime이 ALLOW가 되지 않는 evidence가 있다.
- sandbox/inspection/policy 책임 경계와 raw-data redaction이 재현 가능하게 검증된다.
- M4 구현은 시작하지 않고 entry decision만 열린다.

**Completion**

```text
Status: COMPLETE
Commit: c4ab899
CI: GitHub Actions run 31694083568 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: canonical quick/docs/platform; go test -race -timeout=5m ./...; exact gVisor Bazel helper build; pinned Docker/runsc Linux lifecycle integration; git diff --check
Decision/Evidence: M3 Qualification — Linux Dynamic Inspect
Limitations: macOS/unsupported Linux dynamic runtime is incomplete and never ALLOW; Docker direct runtime remains MVP path
Next: M4-001 — Ready: Yes
```

### M4-001 — npm Install and Promotion entry decision

**Outcome**

npm dependency Verified Set과 trusted Promotion을 구현하기 전, M4의 Install Context·dependency resolution·manifest·staging·offline install·rollback 계약을 canonical 문서에 확정하고 이후 구현 queue를 연다.

**Read**

- [M4 — npm Install and Promotion](./01-milestones.md#8-m4--npm-install-and-promotion)
- [Architecture — Evidence, Staging, and Promotion](../architecture/04-evidence-staging-promotion.md)
- [Domain — Dependency and Verified Set](../domain-model/05-dependency-verified-set.md)
- [Contract — Evidence, Staging, and Promotion](../domain-model/09-contract-evidence-staging-promotion.md)
- [Threat Model — Fail-Closed and Promotion](../threat-model/03-fail-closed-and-promotion.md)

**Acceptance**

- original Install Context와 option allowlist, exact dependency set/manifest, staging와 Promotion failure semantics가 구현 전에 모호하지 않다.
- mutable re-resolution, manifest 밖 dependency, network access와 partial failure가 fail-closed로 처리된다.
- M4의 첫 구현 item만 queue에 추가하며 installer/staging/Promotion source 또는 CI placeholder를 만들지 않는다.

**Completion**

```text
Status: COMPLETE
Checks: official npm ci/package-lock/cache documentation and CycloneDX 1.7 specification review; canonical docs; git diff --check
Decision/Evidence: M4 npm Install and Promotion Contract
Scope: new empty target only; locked resolver graph; every entry inspection; manifest-bound staging; script-free offline local Promotion; atomic target publish
Limitations: existing project/global/workspace/custom registry/auth/npm option passthrough and Host lifecycle/native install are out of M4 MVP automatic Promotion
Next: M4-002 — Ready: Yes
```

### M4-002 — Install Context와 locked npm resolver boundary

**Outcome**

새 empty target Install Context와 disposable npm dependency resolver를 product Domain/Application/Port boundary에 도입하고, resolver output이 bounded exact graph candidate로만 흐르도록 구현한다.

**Read**

- [M4 npm Install and Promotion Contract](../domain-model/13-m4-npm-install-promotion-contract.md)
- [Domain — Operation Request and Context](../domain-model/06-operation-request-context.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [Architecture — Foundation and Dependencies](../architecture/01-foundation-and-dependencies.md)
- [Engineering — Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Work**

- bounded absolute new-target Install Context와 no-overwrite preflight Domain contract를 추가한다.
- consumer-owned Port에 sanitized locked dependency graph resolution contract를 추가하고, npm/runtime type이 Core/Application에 유입되지 않게 한다.
- deterministic fake/contract tests로 unsupported source, invalid lock semantics, duplicate/path escape, missing integrity와 existing target을 fail-closed로 검증한다.
- M4 resolver runtime/docker/network implementation, npm CLI invocation, graph acquisition/inspection, Staging/Promotion source와 CI job은 만들지 않는다.

**Acceptance**

- existing project·global install·workspace·option pass-through가 Install Context 또는 Port API로 우회 유입되지 않는다.
- resolved graph candidate가 exact name/version/registry source/SRI/edge information 없이 생성되지 않는다.
- Core Domain은 npm, Docker, Node/npm lockfile payload 및 Host filesystem type을 import하지 않는다.
- format, unit, static analysis와 architecture check가 통과한다.

**Completion**

```text
Status: COMPLETE
Checks: focused Domain/Application tests; canonical quick; git diff --check
Implementation: canonical new-target Install Context, bounded exact dependency graph Domain values, consumer-owned DependencyResolver Port and request boundary
Safety: Core receives no npm/Docker/Node payload or Host filesystem API; graph is connected, bounded, exact and integrity-required; existing target behavior remains outside a future trusted Promotion boundary
Limitations: no npm resolver runtime, graph acquisition/inspection, Staging, Promotion, CLI command or CI job
Next: M4-003 — Ready: Yes
```

### M0-001 — Project Identity 결정

**Outcome**

구현과 배포 identity로 사용할 Go module path와 `helox` command 이름을 확정한다.

**Read**

- [Project Decisions](../../PROJECT-DECISIONS.md)
- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)

**Work**

- 현재 repository와 module root를 기준으로 canonical Go module path 후보를 확인한다.
- 현재 repository, 주요 executable namespace와 현실적인 배포 경로에서 `helox` 충돌 가능성을 확인한다.
- command 이름을 유지할 수 없으면 구현 전에 대안을 결정하고 관련 canonical 문서를 갱신한다.
- 결정 근거와 확인 시점을 Project Decisions에 연결한다.

**Acceptance**

- module path와 CLI command가 하나의 값으로 확정되어 후속 파일에서 추측할 필요가 없다.
- Heliopause 독립 repository root가 단일 Go module root라는 Step 8 결정을 유지한다.
- 실제 scaffold, `go.mod` 또는 package는 아직 생성하지 않는다.

**Completion**

```text
Status: COMPLETE
CLI command: helox
Repository: github.com/rahoney/heliopause
Go module path: github.com/rahoney/heliopause
Evidence: local repository/PATH와 현실적인 public executable/package namespace 확인
Limitation: 법적 상표 검토와 최종 배포 channel 등록 가능성은 배포 전 재검토
Next: M0-002 — Ready: Yes
```

### M0-002 — Go·Tool·CI Identity lock

**Outcome**

M0 구현에 필요한 기본 Go/minimum Go, 최초 Staticcheck, GitHub-hosted runner와 Action identity를 당시 공식 지원 상태에 맞춰 고정한다.

**Read**

- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- 공식 source에서 안정 Go patch와 최소 지원 Go patch를 확인한다.
- versioned Ubuntu/macOS runner label의 지원 상태를 확인한다.
- 필요한 GitHub-owned Action release와 full commit SHA를 검증한다.
- M0 Quick profile에 필요한 Staticcheck exact version, setup Go 요구, license, 유지보수와 dependency risk를 검토한다.
- 선택 결과와 upgrade 원칙을 구현 가능한 lock input으로 기록한다.

**Acceptance**

- floating `latest`, version range, branch 또는 tag만으로 식별된 실행 요소가 없다.
- default/minimum Go의 역할과 runner/Action/tool identity가 서로 구분된다.
- 아직 사용하지 않는 gosec, Gitleaks, govulncheck 또는 미래 runtime을 미리 pin하지 않는다.
- source URL, 확인한 release/tag와 exact identity가 review 가능하게 기록된다.

**Completion**

```text
Status: COMPLETE
Commit: c6a353b71d5a6659b97e00a17a2e61fbba9ef82a
Checks: official release/support source review; tag/SHA ls-remote verification; local go/staticcheck version verification; Markdown local link/fence validation; git diff --check
Decision/Evidence: docs/engineering/03-quality-toolchain.md and docs/engineering/04-ci-quality-gate.md M0-002 identity lock
Limitations: tools.lock consumer는 M0-005, workflow consumer는 M0-006에서 생성하며 미래 security tool은 capability 활성화 시 별도 pin
Next: M0-003 — Ready: Yes
```

### M0-003 — Go module과 최소 process bootstrap

**Outcome**

프로젝트 루트의 단일 Go module과 `helox`의 가장 얇은 실행 경로를 만든다.

**Read**

- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Architecture — Foundation and Dependencies](../architecture/01-foundation-and-dependencies.md)

**Work**

- project root에 `go.mod`를 생성하고 M0-002의 Go 정책을 적용한다.
- 필요한 범위에서만 `cmd/helox`, `internal/bootstrap`, `internal/cli`를 생성한다.
- process cancellation과 exit ownership을 경계에 맞게 두고 business workflow는 추가하지 않는다.
- 최소 process smoke test와 Linux/macOS build validity를 마련한다.

**Acceptance**

- 별도 `src/`, nested module과 미래용 placeholder package가 없다.
- `cmd/helox`는 bootstrap 호출과 process exit만 소유한다.
- CLI/bootstrap에 Domain, Policy, Adapter shortcut이 없다.
- module과 smoke test가 public network·credential·Host project 없이 재현된다.

**Completion**

```text
Status: COMPLETE
Commit: 5e99b6028009af9506232e291dbda60a09ca9450
Checks: gofmt; go mod tidy -diff/verify; go build; compile-only test; go vet; Staticcheck 2026.1; full test; Linux amd64/macOS amd64·arm64 build; offline module list/graph; package import boundary; Markdown local link/fence validation; git diff --check
Implementation: root go.mod; cmd/helox process entrypoint; internal/bootstrap composition root; internal/cli Cobra boundary; host-independent process smoke test
Limitations: business workflow·Domain·Policy·Adapter·future package와 canonical check runner는 현재 항목 범위에서 추가하지 않음
Next: M0-004 — Ready: Yes
```

### M0-004 — Canonical check runner foundation

**Outcome**

로컬과 CI가 공유할 standard-library 기반 `scripts/check` entrypoint와 deterministic foundation profile을 만든다.

**Read**

- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)

**Work**

- module root에서만 실행되는 `go run ./scripts/check <profile>` 구조를 만든다.
- read-only `format check`, module drift, build/type validity와 `docs` 검사를 구현한다.
- local Markdown link, code fence와 의도하지 않은 absolute local link를 검사한다.
- 현재 존재하는 package에 대해 검증 가능한 import rule을 구현하고, 이후 새 package가 추가되면 같은 checker에 규칙을 증분한다. 존재하지 않는 미래 package를 위한 fake pass 규칙은 만들지 않는다.
- command/argument 분리, timeout, bounded output과 첫 실패 보존을 적용한다.

**Acceptance**

- `format` 외 profile은 source/configuration을 변경하지 않는다.
- check가 source tree 밖 임의 executable이나 PATH 우선순위에 의존하지 않는다.
- finding과 tool/check execution failure가 모두 non-zero이며 원인이 구분된다.
- 문서 router가 깨진 fixture와 정상 tree를 deterministic하게 판별한다.

**Completion**

```text
Status: COMPLETE
Commit: f307039b916408383f76391fdc313c2f93914126
Checks: canonical foundation/docs; go vet; Staticcheck 2026.1; full test; race test; Linux amd64/macOS amd64·arm64 build; module-root rejection; unknown-profile failure; read-only source hash comparison; git diff --check
Implementation: standard-library scripts/check; foundation/docs/format profiles; offline local-toolchain command runner; timeout/bounded output/first-failure semantics; current-package architecture rules; Markdown link/fence/absolute-local-path validator
Fixtures: healthy/broken Markdown tree; package shortcut; format drift; module drift; root guard; output truncation; sequential failure preservation
Limitations: Staticcheck lock/bootstrap, quick profile, security tools and CI jobs are deferred to their queued work items
Next: M0-005 — Ready: Yes
```

### M0-005 — Static analysis와 Quick profile 활성화

**Outcome**

Go 기본 검사와 pinned Staticcheck를 canonical `quick` profile로 연결한다.

**Read**

- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)

**Work**

- `scripts/tools.lock.json`에 실제 사용하는 Staticcheck identity만 기록한다.
- source tree 밖 project-specific cache를 사용하는 `bootstrap`을 구현한다.
- `quick`에 format check, module drift, build/type validity, architecture, `go vet`, Staticcheck와 default test를 연결한다.
- bootstrap network phase와 offline check phase를 분리하고 version mismatch를 실패시킨다.

**Acceptance**

- product `go.mod`가 quality tool dependency graph를 소유하지 않는다.
- `@latest`, floating download, `curl | sh`와 임의 PATH executable 실행이 없다.
- `quick`은 read-only이고 local clean checkout에서 반복 가능하다.
- M0-002의 pinned identity와 실제 실행 binary version이 일치한다.

**Completion**

```text
Status: COMPLETE
Commit: dff6682f152ce2cb59c85abdb2e3cec6da8fa961
Checks: official Staticcheck release recheck; strict lock tests; bootstrap source/config hash comparison; exact cached binary version; repeated canonical offline quick; canonical docs; go vet; Staticcheck; full/race test; Linux amd64/macOS amd64·arm64 build; git diff --check
Implementation: scripts/tools.lock.json Staticcheck-only identity; isolated source-tree-external cache; verified proxy/checksum bootstrap; exact Go setup check; offline quick with format/module/build/type/architecture/vet/Staticcheck/default test
Safety: product module download uses a cache-internal temporary module root; relative/in-tree/symlink-into-tree caches, floating lock values, unknown lock fields, setup Go mismatch, missing binary and version mismatch fail closed
Limitations: CI workflow and repository merge gate are deferred to M0-006/M0-007; security tools remain unpinned until their capability work item
Next: M0-006 — Ready: Yes
```

### M0-006 — Quick·Docs·Required CI foundation

**Outcome**

GitHub Actions에서 실제 local profile을 실행하는 첫 required workflow를 만든다.

**Read**

- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- repository root에 Heliopause 전용 CI workflow를 생성하고 module working directory를 명시한다.
- versioned Ubuntu와 pinned Go/Action identity로 `Quick`, `Docs`, `Required`를 활성화한다.
- `pull_request`, `push` to `main`, `merge_group`, bounded `workflow_dispatch` trigger를 적용한다.
- fork/PR secret 없음, 최소 token permission, checkout credential 비보존과 cache 비활성화를 적용한다.
- `Required`가 failure/cancel/unexpected skip을 성공으로 변환하지 않게 한다.

**Acceptance**

- CI가 local canonical profile과 다른 검사 명령을 재구현하지 않는다.
- 모든 `uses:`가 full SHA와 사람이 식별 가능한 release 주석을 가진다.
- placeholder Security/Vulnerability/Race/Integration/E2E job이 없다.
- 실제 child failure와 cancellation에 aggregate가 실패하는 test 또는 검증 evidence가 있다.

**Completion**

```text
Status: COMPLETE
Commit: 53fb846bdd5571d610ebf757b31128c566fc16f7
Branch: milestone/m0-foundation (origin tracking)
Checks: official runner/action identity recheck; action tag full-SHA ls-remote; YAML syntax parse; canonical quick/docs; gofmt; module tidy/verify; build/test-build; go vet; Staticcheck; full/race test; workflow security fixture; git diff --check
Implementation: pull_request/main push/merge_group/workflow_dispatch triggers; ubuntu-24.04; checkout v6.0.2 and setup-go v7.0.0 full SHA; exact Go 1.26.5; no Actions cache; credential persistence disabled; Quick/Docs/Required only
Aggregate evidence: success+success passes; failure, cancelled and skipped child results each return non-zero; missing child is execution failure; Required uses always() and depends on Quick+Docs
Limitations: feature branch push intentionally does not trigger CI; actual GitHub-hosted run occurs when the milestone PR is opened. Minimum Go/macOS jobs and repository settings are M0-007 scope
Next: M0-007 — Ready: Yes
```

### M0-007 — Minimum Go·macOS와 repository gate 검증

**Outcome**

지원 toolchain/platform 경계를 실제 CI에서 검증하고 stable aggregate를 repository merge gate로 사용할 준비를 마친다.

**Read**

- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)
- [Step 12 — Milestones](./01-milestones.md)

**Work**

- minimum Go Ubuntu와 pinned Go macOS build/default test job을 활성화한다.
- 두 job을 `Heliopause CI / Required` dependency에 포함한다.
- 독립 repository의 workflow 이름과 repository setting 적용 영향을 확인한다.
- branch rule과 Actions SHA/allowlist setting의 적용 범위를 제안하고, repository-wide 변경은 owner의 명시적 승인을 받은 경우에만 적용한다.

**Acceptance**

- primary, minimum Go와 macOS 결과가 서로 독립적으로 실제 실행된다.
- unavailable runner, failed 또는 skipped required job이 aggregate success가 되지 않는다.
- Linux build 결과를 WSL2 qualification으로 표시하지 않는다.
- repository-wide branch rule 적용은 M0 완료의 필수 조건이 아니다. 적용 권한·운영 범위가 없으면 제안과 미적용 사유를 기록하고, Heliopause CI 자체의 gate 동작 검증이 완료되면 M0 진행을 차단하지 않는다.

**Completion**

```text
Status: COMPLETE
Commit: 0ec5b6575f29ee5b8a91af96182650781ced5c0f
PR/CI: draft PR #1; run 31663728891; Quick, Docs, Minimum Go, macOS, Required success
Checks: actionlint 1.7.12; canonical quick/docs/platform; Go 1.25.12 platform; workflow validator tests; git diff --check
Repository settings: Actions enabled, allowed_actions=all, sha_pinning_required=false; main protection 없음; ruleset 없음
Decision/Evidence: Step 11 M0-007 repository setting 점검과 적용 제안
Limitations: Linux 결과는 WSL2 qualification이 아님. repository-wide Actions allowlist/SHA pinning/branch protection은 별도 owner 적용 승인 없이 변경하지 않음
Next: M0-008 — Ready: Yes
```

### M0-008 — M0 qualification과 M1 handoff

**Outcome**

M0 exit criteria를 evidence로 닫고 다음에는 M1 entry decision만 열 수 있게 한다.

**Read**

- [Step 12 — Milestones](./01-milestones.md)
- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- M0의 모든 exit criteria와 Global Definition of Done을 실제 command/CI evidence에 대조한다.
- version 공급망 검토, active check 목록, 제한과 유보 사항을 기록한다.
- PROJECT-DECISIONS, experiment README와 이 queue의 상태를 갱신한다.
- M1 entry decision을 실행 가능한 design work item으로 분해하되 M1 구현 항목 전체를 미리 만들지 않는다.

**Acceptance**

- M0-001~007이 모두 `COMPLETE`이고 미해결 필수 gate가 없다.
- clean checkout의 canonical local check와 활성 CI가 통과한다.
- M0 milestone completion commit과 검증 결과를 참조할 수 있다.
- 다음 active item은 M1 entry decision 하나이며 미래 Adapter/Sandbox/Promotion placeholder가 없다.

**Completion**

```text
Status: COMPLETE
Commit: 011dd4f
PR/CI: draft PR #1; checkout v7.0.1 qualification run 31664225839과 final completion run 31664383069의 Quick, Docs, Minimum Go, macOS, Required success
Checks: clean checkout fresh bootstrap; canonical quick/docs/platform; actionlint 1.7.12; single module/no placeholder audit; M0 exit criteria와 Global DoD 대조; git diff --check
Decision/Evidence: M0 Qualification — Implementation Foundation
Limitations: repository-wide Actions allowlist/SHA requirement/main protection은 별도 owner 승인 없이 미적용; WSL2와 미래 security/runtime capability는 qualification 대상 아님
Next: M1-001 — Ready: Yes
```

### M1-001 — Domain workflow entry decision

**Outcome**

M1 fake inspect lifecycle을 구현하기 전에 최소 Domain·Application·Port·Policy·CLI 계약과 synthetic scenario를 결정하여 하나의 실행 가능한 구현 queue를 열 수 있게 한다.

**Read**

- [Step 12 — M1 Domain Workflow Skeleton](./01-milestones.md#5-m1--domain-workflow-skeleton)
- [Architecture — Foundation and Dependencies](../architecture/01-foundation-and-dependencies.md)
- [Domain — Core Concepts](../domain-model/01-core-concepts.md)
- [Domain — Artifact Identity](../domain-model/02-artifact-identity.md)
- [Domain — Inspection Run and Status](../domain-model/03-inspection-run-and-status.md)
- [Domain — Verification, Evidence, and Finding](../domain-model/04-verification-evidence-finding.md)
- [Domain — Operation Request and Context](../domain-model/06-operation-request-context.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)

**Work**

- M1에 필요한 최소 Domain type/schema, identifier 생성과 ownership을 결정한다.
- Operation Request/Result, fake Artifact·Verification·Inspection·Evidence Port method와 Application orchestration signature를 결정한다.
- Execution Status, Run Outcome, Policy Decision, Operation Status의 허용 transition과 operational error 경계를 결정한다.
- 최소 Policy input/rule/version/reason과 CLI exit code·machine result schema의 최초 contract를 결정한다.
- `ALLOW`, `MANUAL_REVIEW`, `BLOCK`, operational failure와 `inspect`에서 Promotion 미호출을 증명할 synthetic scenario를 정의한다.
- 결정이 완료된 뒤에만 첫 M1 implementation work item을 Queue에 추가한다.

**Acceptance**

- exact fake identity resolve부터 Run 생성, acquisition/result, Policy와 Operation Result까지 순서·type·owner가 모호하지 않다.
- 네 상태 축과 operational error가 API/schema에서 합쳐지지 않는다.
- 필수 check 미충족은 `ALLOW`가 아니며 `inspect` path에 Staging/Promotion이 없다.
- CLI 사람용·기계용 결과가 같은 Run/identity/decision을 참조하는 contract가 있다.
- synthetic fixture와 expected result가 정의되고 실제 Host credential/network/runtime을 요구하지 않는다.
- 미래 npm Adapter, Sandbox, Evidence Store와 Promotion placeholder를 만들지 않는다.

**Completion**

```text
Status: COMPLETE
Commit: 93903ab
Checks: canonical docs profile; git diff --check
Decision/Evidence: M1 Entry Decision — Domain Workflow Contract
Limitations: 구현과 production fake wiring 없음; npm/network/Sandbox/Evidence Store/Promotion은 범위 밖
Next: M1-002 — Ready: Yes
```

### M1-002 — Domain identity와 Run state foundation

**Outcome**

ecosystem 비종속 identity, digest, Operation/Run identifier와 Run lifecycle을 불변식이 있는 Domain type으로 구현한다.

**Read**

- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [Domain — Artifact Identity](../domain-model/02-artifact-identity.md)
- [Domain — Inspection Run and Status](../domain-model/03-inspection-run-and-status.md)
- [Engineering — Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Work**

- `internal/core/domain`에 bounded identifier, Artifact Reference/Resolved Identity, SHA-256 digest와 acquired subject type을 추가한다.
- `crypto/rand.Reader`를 사용하는 Operation/Run ID generator와 deterministic test 주입에 필요한 parser/validator를 구현한다.
- `CREATED → ACTIVE → FINALIZED` 전이와 completed/failed finalization을 구현한다.
- invalid value, identity/digest mismatch, 역전이와 finalized mutation을 table-driven test로 거부한다.

**Acceptance**

- Domain package가 standard library 외 product/concrete dependency를 import하지 않는다.
- exact resolved identity가 없는 Run을 만들 수 없고 acquired subject mismatch를 결합할 수 없다.
- completed Run에는 Policy Decision이 하나 있고 failed Run에는 Policy Decision이 없다.
- random failure가 error로 반환되고 고정 ID 기반 test가 재현 가능하다.
- format, unit, static analysis와 architecture check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 0fef849
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: public npm reference grammar, abbreviated metadata, exact identity and bounded public-registry transport contract
Limitations: Acquire/Intake/declared integrity/Evidence/CLI wiring 없음; no external production registry를 test success input으로 사용
Next: M2-003 — Ready: Yes
```

**Completion**

```text
Status: COMPLETE
Commit: 2a11310
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: M1 Workflow Contract의 identity, identifier, Run lifecycle와 Policy Decision 추적 계약
Limitations: Port/fake/Application/Policy evaluation/CLI 구현 없음; production network/filesystem/Promotion 없음
Next: M1-003 — Ready: Yes
```

### M1-003 — Check result·Port와 deterministic fake

**Outcome**

Verification·Inspection·Evidence의 상태 의미와 네 outbound Port 계약을 정의하고 synthetic fixture용 fake가 같은 consumer contract를 만족한다.

**Read**

- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [Domain — Verification, Evidence, and Finding](../domain-model/04-verification-evidence-finding.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)

**Work**

- Check Execution, Verification/Inspection Report, Finding, bounded Evidence/Reference type을 구현한다.
- Artifact·Verification·Inspection·Evidence Port를 `internal/core/ports`에 소비자 계약으로 추가한다.
- `safe`, `review`, `blocked`, `acquire-error`, `resolve-error` fake를 test-only 위치에 구현한다.
- contract test로 unsupported, unavailable, incomplete와 operational error를 빈 success와 구분한다.

**Acceptance**

- 모든 Port I/O가 context와 ecosystem-neutral Domain type을 사용한다.
- raw provider output, Host path와 secret이 Evidence에 들어갈 수 없다.
- fake는 network/filesystem/credential/process를 사용하거나 production bootstrap에 연결되지 않는다.
- 후속 호출 순서와 첫 error 이후 stop 여부를 test에서 관찰할 수 있다.
- format, unit, static analysis와 architecture check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: aeb8fb8
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: independent CheckExecution axes, bound Verification/Inspection reports, bounded Evidence/Finding, four M1 Port contracts and five deterministic scenarios
Limitations: fake는 testutil 전용이며 production bootstrap/CLI wiring 없음; Application/Policy evaluation/Promotion 없음
Next: M1-004 — Ready: Yes
```

### M1-004 — Inspect orchestration과 Policy v1

**Outcome**

M1 Port를 순서대로 조율하는 Inspect use case와 deterministic fail-closed Policy v1을 구현한다.

**Read**

- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [Architecture — Foundation and Dependencies](../architecture/01-foundation-and-dependencies.md)
- [Domain — Operation Request and Context](../domain-model/06-operation-request-context.md)

**Work**

- immutable Operation Result와 partial operational failure result를 구현한다.
- injected ID generator와 네 Port로 Inspect application service를 구성한다.
- `m1-fake-inspect` version 1의 ordered Policy reason을 구현한다.
- resolve→Run→acquire→verify→inspect→evidence→policy→finalize 순서와 첫 operational error stop을 test한다.

**Acceptance**

- `ALLOW`, `MANUAL_REVIEW`, `BLOCK`과 resolve/acquire failure가 서로 다른 기대 상태 축을 가진다.
- required check 미완료가 `ALLOW`가 되지 않는다.
- operational failure에는 Policy Decision이 없고 `%w` error cause와 partial result가 보존된다.
- Application과 Policy가 concrete fake, Cobra, filesystem 또는 Promotion을 import하지 않는다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 2c5e53f, 2aefb3b, 703ebb1
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: bounded archive/manifest/lifecycle-key inspection and intake-separated atomic Evidence records
Limitations: M2 Policy·CLI wiring과 controlled vertical flow 없음
Next: M2-005 — Ready: Yes
```

**Completion**

```text
Status: COMPLETE
Commit: 7091156
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: exact Inspect orchestration order, partial OperationResult, M1 Policy v1 ordered rules and synthetic ALLOW/MANUAL_REVIEW/BLOCK/failure tests
Limitations: CLI presenter/adapter와 production fake wiring 없음; retry/concurrency/Promotion 없음
Next: M1-005 — Ready: Yes
```

### M1-005 — CLI result contract와 synthetic vertical test

**Outcome**

동일한 Operation Result를 사람용·JSON 출력과 안정적 exit code로 변환하고 fake workflow의 end-to-end 계약을 검증한다.

**Read**

- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- `helox.operation-result/v1` JSON presenter와 bounded human presenter를 구현한다.
- exit code 0/1/2/10/20 mapping을 operation status와 Policy Decision으로부터 구현한다.
- injected Inspect use case를 사용하는 CLI adapter test와 다섯 synthetic vertical scenario를 추가한다.
- architecture checker를 새 package의 실제 import direction에 맞게 활성화한다.

**Acceptance**

- human/JSON 결과가 동일 Operation ID, Run ID, exact identity/digest, decision/reason을 참조한다.
- machine mode가 prompt를 표시하지 않고 operational error의 partial result를 버리지 않는다.
- fake source를 production 사용자 command나 bootstrap wiring으로 노출하지 않는다.
- `inspect + ALLOW`를 포함한 어느 경로에도 Staging/Promotion package·Port·호출이 없다.
- canonical quick/docs/platform check와 process/CLI contract test가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 81b6d1f
Checks: canonical quick/docs; go test -race -timeout=5m ./...; check-jsonschema Draft 2020-12 metaschema; production fake/Promotion/Staging audit; git diff --check
Decision/Evidence: helox.operation-result/v1 schema, human/JSON presenter, exit 0/1/2/10/20 mapping and five injected synthetic CLI scenarios
Limitations: presenter/adapter는 production root command에 fake로 wiring되지 않음; 실제 ecosystem command는 M2 범위
Next: M1-006 — Ready: Yes
```

### M1-006 — M1 qualification과 M2 handoff

**Outcome**

M1 exit criteria를 clean checkout과 CI에서 재현하고 실제 npm static inspect를 열기 위한 M2 entry item 하나로 handoff한다.

**Read**

- [Step 12 — M1 Domain Workflow Skeleton](./01-milestones.md#5-m1--domain-workflow-skeleton)
- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- M1 scope, architecture/import direction, no Promotion와 no production fake wiring을 audit한다.
- clean checkout에서 bootstrap, quick, docs, minimum Go, macOS와 활성 race gate를 재현한다.
- branch CI와 supply-chain pin을 확인하고 M1 qualification evidence를 기록한다.
- M1이 모두 충족된 뒤 M2 entry decision 하나만 Queue에 연다.

**Acceptance**

- M1 milestone exit criteria와 Global DoD 해당 항목에 재현 가능한 evidence가 있다.
- clean checkout과 required CI가 성공하고 ignored local 문서·비밀값·개인 환경 파일이 tracked되지 않는다.
- M1 범위 밖 placeholder와 fake success path가 없다.
- M2 구현은 아직 시작하지 않고 exact entry decision만 Ready 상태다.

**Completion**

```text
Status: COMPLETE
Commit: e1ba0d5
PR/CI: PR #2; implementation run 31669932193과 final completion run 31670097293의 Quick, Docs, Minimum Go, macOS, Required success
Checks: clean clone canonical quick/docs/platform; race; JSON metaschema; package/import/no production fake/no Promotion audit; git diff --check
Decision/Evidence: M1 Qualification — Domain Workflow Skeleton
Limitations: production ecosystem command/Adapter/Evidence Store/Sandbox/Promotion 없음; race는 local qualification이며 Required CI child 아님
Next: M2-001 — Ready: Yes
```

### M2-001 — npm Static Inspect entry decision

**Outcome**

실제 npm reference를 exact identity와 controlled acquired bytes로 연결하는 첫 production static inspect를 구현하기 전에 최소 Adapter·network·storage·verification·inspection·Evidence·Policy·CLI 계약과 fixture를 결정한다.

**Read**

- [Step 12 — M2 npm Static Inspect](./01-milestones.md#6-m2--npm-static-inspect)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [MVP — Ecosystems, Platforms, and Artifacts](../mvp-scope/01-ecosystems-platforms-artifacts.md)
- [MVP — Inspection and Verification](../mvp-scope/02-inspection-and-verification.md)
- [Architecture — Adapters and Providers](../architecture/02-adapters-and-providers.md)
- [Domain — Artifact Identity](../domain-model/02-artifact-identity.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)

**Work**

- npm reference와 registry metadata를 공통 Artifact Reference/Resolved Identity에 mapping하는 exact contract를 결정한다.
- registry endpoint, redirect, timeout, response/archive size, public authentication-free 범위와 error taxonomy를 결정한다.
- Controlled Intake handle/root/permission/cleanup과 observed digest ownership을 결정한다.
- bounded tarball/manifest/static check, declared integrity Verification, Evidence Store와 M2 Policy requirement를 결정한다.
- safe/malformed/digest mismatch/archive escape/timeout/unavailable controlled fixture와 result schema evolution을 결정한다.
- 결정 후에만 M2 implementation queue를 추가하며 Dynamic Sandbox와 Promotion placeholder를 만들지 않는다.

**Acceptance**

- mutable npm reference가 acquisition 전 exact version으로 resolve되는 type/API가 명확하다.
- network/content/archive/resource limit과 cleanup failure가 fail-closed operational 결과로 정의된다.
- declared integrity와 acquired bytes의 observed digest가 분리되고 mismatch가 정상 Verification Result로 표현된다.
- Dynamic Inspection 부재가 빈 success나 자동 `ALLOW`로 해석되지 않는다.
- fixture가 public registry 상태, 실제 credential, 개인 filesystem과 mutable reference에 성공 여부를 의존하지 않는다.
- production code와 dependency를 추가하기 전 하나의 실행 가능한 M2 queue가 열린다.

**Completion**

```text
Status: COMPLETE
Commit: M2-001 branch commit
Checks: canonical docs profile; git diff --check; npm public registry metadata/SRI contract 확인
Decision/Evidence: M2 Entry Decision — npm Static Inspect Contract
Limitations: npm Adapter/HTTP/Intake/Evidence/CLI implementation과 external dependency 추가 없음
Next: M2-002 — Ready: Yes
```

### M2-002 — npm reference·metadata resolve foundation

**Outcome**

public npm input을 bounded reference로 parse하고 abbreviated metadata에서 exact resolved identity와 declared tarball/integrity descriptor를 생성한다.

**Read**

- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [Engineering — Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Work**

- `internal/artifact/npm`에 public registry 전용 reference parser와 metadata model을 추가한다.
- explicit injected HTTP client로 metadata endpoint, timeout/body/redirect/host guard를 구현한다.
- name/tag/exact version resolve, response consistency와 sanitized operational error를 contract test한다.
- generic Port/Domain에 필요한 최소 descriptor extension만 추가하고 npm raw struct를 Core에 전파하지 않는다.

**Acceptance**

- scoped/unscoped name, latest/tag/exact version이 one exact `ResolvedArtifactIdentity`로 끝난다.
- range/URL/git/file/auth/custom registry/redirect와 malformed/oversized metadata가 success로 변환되지 않는다.
- application은 npm package를 import하지 않고 production client가 credential/proxy/redirect를 사용하지 않는다.
- format, unit, static analysis와 architecture check가 통과한다.

### M2-003 — Controlled Intake와 declared integrity verification

**Outcome**

tarball을 trusted intake root에 bounded stream으로 확보하고 observed SHA-256과 declared SHA-512 SRI verification을 분리한다.

**Read**

- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [Domain — Artifact Identity](../domain-model/02-artifact-identity.md)
- [Threat Model — Fail-Closed and Promotion](../threat-model/03-fail-closed-and-promotion.md)

**Work**

- run-local intake root, permission, atomic write, containment, cleanup와 content handle을 구현한다.
- tarball host/status/body/timeout guard와 SHA-256/SHA-512 stream calculation을 구현한다.
- missing/malformed/mismatch SRI를 normalized Verification result/finding으로 만들고 I/O failure와 구분한다.
- controlled HTTP fixture로 limit, sync/cleanup, mismatch와 partial result를 test한다.

**Acceptance**

- acquired bytes만 observed digest를 만들고 declared integrity가 content identity를 대체하지 않는다.
- failed/partial content가 valid handle이나 Policy input으로 노출되지 않는다.
- intake와 Evidence/Staging root가 혼합되지 않고 Host/project path가 output에 없다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: bec6483
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: run-local intake, atomic tarball streaming, SHA-256 content identity와 SHA-512 declared SRI verification result/finding
Limitations: static tar/manifest inspection, Evidence Store, M2 Policy·CLI wiring 없음
Next: M2-004 — Ready: Yes
```

### M2-004 — npm tarball static inspection과 Evidence Store

**Outcome**

extract 없이 tarball/manifest 위험을 정규화하고 trusted local Evidence reference를 만든다.

**Read**

- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [MVP — Inspection and Verification](../mvp-scope/02-inspection-and-verification.md)
- [Contract — Evidence, Staging, and Promotion](../domain-model/09-contract-evidence-staging-promotion.md)

**Work**

- gzip/tar stream limit, path/type/manifest/lifecycle key check을 구현한다.
- unsafe archive/manifest/integrity finding과 trusted JSON Evidence writer를 구현한다.
- raw artifact/manifest script/Host path가 result or Evidence summary로 새지 않는 contract test를 추가한다.

**Acceptance**

- tarball을 Host filesystem에 extract하거나 lifecycle script를 실행하지 않는다.
- structural violation은 completed static inspection의 blocking finding이며 parser failure와 구분된다.
- Evidence reference가 exact Run/identity/digest를 추적하고 write failure는 operational failure다.
- format, unit, static analysis, architecture와 race check가 통과한다.


### M2-005 — M2 Policy·CLI wiring과 controlled vertical test

**Outcome**

real npm inspect CLI를 M2 Policy와 연결하고 deterministic controlled registry fixtures로 end-to-end 결과를 검증한다.

**Read**

- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)
- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)

**Work**

- `m2-npm-static-inspect` Policy와 dynamic unavailable limitation을 구현한다.
- production bootstrap에서 real npm Adapter/Intake/Evidence writer를 explicit wiring하고 `helox npm inspect` command를 추가한다.
- safe/manual, integrity mismatch/block, unsafe archive/block, timeout/operational failure vertical test를 추가한다.
- result schema compatibility와 exit 10/20/1을 확인한다.

**Acceptance**

- safe static tarball도 `ALLOW`가 아니라 `MANUAL_REVIEW`이며 no Promotion path를 유지한다.
- public registry mutable state를 CI success input으로 사용하지 않는다.
- human/JSON output이 exact npm identity, observed SHA-256, check limitation, evidence references와 decision을 공유한다.
- canonical quick/docs/platform, CLI/process contract와 controlled integration test가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 9e5a555, 5f12eb2
Checks: canonical quick; go test -race -timeout=5m ./...; controlled vertical tests; git diff --check
Decision/Evidence: M2 Policy fail-closed decision, npm CLI wiring, safe/manual·integrity/archive block·operational failure fixtures
Limitations: Dynamic Sandbox/install/Staging/Promotion 없음
Next: M2-006 — Ready: Yes
```

### M2-006 — M2 qualification과 M3 handoff

**Outcome**

M2 exit criteria를 clean checkout/CI에서 재현하고 Linux Dynamic Inspect entry decision 하나로 handoff한다.

**Read**

- [Step 12 — M2 npm Static Inspect](./01-milestones.md#6-m2--npm-static-inspect)
- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- M2 scope, actual network restrictions, intake/Evidence separation, no extraction/execution/Promotion을 audit한다.
- clean checkout에서 all active profiles, race, controlled fixtures와 schema validation을 재현한다.
- branch CI/supply-chain change를 확인하고 M3 entry decision 하나만 Queue에 연다.

**Acceptance**

- M2 milestone exit criteria와 Global DoD 해당 항목에 reproducible evidence가 있다.
- external registry outage/mutable metadata가 required test success를 좌우하지 않는다.
- Dynamic Sandbox, install, Staging와 Promotion이 M2 source/CI placeholder로 생기지 않는다.
- M3 구현은 시작하지 않고 exact entry decision만 Ready 상태다.

**Completion**

```text
Status: COMPLETE
Commit: qualification record after 5f12eb2
Checks: canonical quick/docs/platform; go test -race -timeout=5m ./...; schema metaschema; git diff --check
Decision/Evidence: docs/planning/05-m2-qualification.md; all M2 vertical fixtures use in-memory controlled transport
Limitations: public CI run/PR merge is external; M3 implementation and Sandbox are not started
Next: M3-001 — Ready: Yes
```

## 5. Work Item 완료 기록

work item을 완료할 때 같은 변경에서 다음을 기록한다.

```text
Status: COMPLETE
Commit: <commit SHA>
Checks: <canonical profile 또는 CI run>
Decision/Evidence: <필요한 canonical link>
Limitations: <없음 또는 명시적 제한>
Next: <다음 ID와 Ready 여부>
```

- 상세 구현 일지를 이 문서에 누적하지 않는다. commit과 canonical 문서를 참조한다.
- 완료된 항목은 표에서 삭제하거나 번호를 재사용하지 않는다.
- 다음 항목이 시작 가능하면 `Ready: Yes`로 바꾸고, 실제 시작 전에는 `IN_PROGRESS`로 표시하지 않는다.
- 구현 도중 queue 자체를 바꿨다면 변경 이유와 영향을 해당 commit에서 설명한다.

## 6. Historical state records and current M12 status

아래 M0~M11 항목은 과거 완료·qualification 기록이다. 과거 work item의
`현재 재개 지점` 표현은 현재 실행 상태가 아니며, 현재 실행 상태는 이 절의
M12 canonical block만 소유한다.

```text
M0: COMPLETE
M0-008: COMPLETE
M1: COMPLETE
M1-001: COMPLETE
M1-002: COMPLETE
M1-003: COMPLETE
M1-004: COMPLETE
M1-005: COMPLETE
M1-006: COMPLETE
M2: COMPLETE
M2-001: COMPLETE
M2-002: COMPLETE
M2-003: COMPLETE
M2-004: COMPLETE
M2-005: COMPLETE
M2-006: COMPLETE
M3: COMPLETE
M3-008: COMPLETE
M4: COMPLETE
M5: COMPLETE
M6: COMPLETE
M7: COMPLETE
M8: COMPLETE
M9: COMPLETE
M9-001: COMPLETE
M9-002: COMPLETE
M9-003: COMPLETE
M9-004: COMPLETE
M9-005: COMPLETE
M9-006: COMPLETE
M10: COMPLETE
M10-001: COMPLETE
M10-002: COMPLETE
M10-003: COMPLETE
M10-004: COMPLETE
M10-005: COMPLETE
M10-006: COMPLETE
M10-007: COMPLETE
M11: COMPLETE
M11-001: COMPLETE
M11-002: COMPLETE
M11-003: COMPLETE
M11-004: COMPLETE
M11-005: COMPLETE
M11-FIX-01: COMPLETE
M11-FIX-02: COMPLETE
M11-FIX-03: COMPLETE
M11-FIX-04: COMPLETE
M11-FIX-05: COMPLETE
M12: IN_PROGRESS
M12-001: COMPLETE
M12-002: COMPLETE
M12-003: COMPLETE
M12-004: COMPLETE
M12-005: NOT_STARTED
M12-02: RESERVED
M13: BLOCKED
Active work item: none (M12-002~004 remote qualification complete)
Next work item: M12-005 — NOT_STARTED / Ready: Yes
```

M9-006까지 qualification을 완료했다. M10-001은 release identity·manifest 및
bootstrap trust를 확정했고 M10-002는 release asset build와 provenance candidate
workflow를 구현했다. M10-003은 HAA-owned runtime 구성이 필요하지 않음을
확정하고 공식 runtime image의 immutable identity, digest와 candidate provenance
binding을 검증했다. M10-004는 verifier-gated installer·atomic activation·doctor
경계를 구현했다. M10-005는 rollback/tamper qualification과 명시적인
fail-closed release gate를 구현했다. M10-006은 Apache-2.0 `LICENSE`, Harmony
copyright-license CLA Option Five, hosted GitHub CLA Assistant status 정책,
individual/entity contribution과 third-party provenance 절차를 기록했다.
CLA governing jurisdiction은 Republic of Korea로 확정되었다. remote PR/CI
qualification과 protected `release` environment 설정이 끝날 때까지 public release는
게시하지 않는다. M10의 verified distribution implementation과 M11 detection depth는
완료되었고, M12 ecosystem expansion 및 M12-02 final red-team/fix gate 이후 M13에서
production activation과 최종 배포를 수행한다.

### M11 work breakdown

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M11-001 | optional seccheck/context field와 detection schema entry decision | COMPLETE |
| 2 | M11-002 | bounded process/filesystem/network observation normalization | COMPLETE |
| 3 | M11-003 | unexpected process·honeytoken·workspace violation Finding | COMPLETE |
| 4 | M11-004 | sensitive payload suppression·Evidence summary·retention bound | COMPLETE |
| 5 | M11-005 | actual gVisor integration·policy regression·M12 handoff qualification | COMPLETE |

### M12 Ecosystem Expansion queue

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M12-001 | Official PyTorch source support | COMPLETE |
| 2 | M12-002 | public Go Modules | COMPLETE |
| 3 | M12-003 | Rust/Cargo and public crates.io | COMPLETE |
| 4 | M12-004 | Terraform Provider installation and cumulative remote qualification | COMPLETE |
| 5 | M12-005 | cross-ecosystem qualification and feature freeze | COMPLETE |
| 6 | M12-02 | final red-team/fix gate | COMPLETE |

### M12-001 acceptance 종료 (2026-10-03)

`IMPLEMENTED=YES`, `WIRED=YES`, `QUALIFIED=YES`, `ACCEPTANCE_CLOSED=YES`로
닫는다. `MISSING: None`이며 추가 구현은 없다. 최종 qualification 후보
`8cb258ca882653511710b08bf8d09ffa1ef212bf`에서 ordinary Required와
CPU·cu126·cu130·cu132 full E2E가 모두 통과했다. 네 run 각각의 native macOS,
sdist·wheel·promotion 직접 소비자와 helper cleanup 성공도 확인했다.
Run/job, 실제 checkout SHA, 실행 결과와 acceptance 대응은
[M12-001 closure evidence](./evidence/m12-001-closure/result.json)가 소유한다.
별도 agent 문맥의 읽기 전용 종료 검토에서도 acceptance 누락은 발견되지 않았다.
이는 GitHub 승인 리뷰나 외부 독립 보안 감사가 아니다.

종료 변경은 README·PROJECT-DECISIONS·이 Queue·closure evidence뿐이다.
나머지 모든 tracked blob/mode가 qualification 후보와 동일하므로 기존 full
qualification을 그 입력 범위에서 재사용한다. 새 closure head의 ordinary
Required CI와 PR merge 및 merge 후 main CI는 아직 수행하지 않은 전달 단계다.
기존 green을 새 head의 CI 결과로 표기하지 않는다. Ordinary Required 자체에
CPU full E2E가 포함되며 별도 수동 CUDA full 재실행은 이 문서 변경에 필요하지 않다.

M12 전체는 `IN_PROGRESS`를 유지한다. 다음 M12-002 public Go Modules는
`NOT_STARTED / Ready: Yes`이며 아직 구현 작업을 시작하지 않았다.

### M12-002 시작 baseline audit (2026-10-04)

PR28 merge `62c3b595e8e27a4a1d836849643c372692a39b8b`와
[main CI run37132385788](https://github.com/rahoney/heliopause/actions/runs/37132385788)의
10개 jobs(Required/macOS 포함) success를 확인하고
`milestone/m12-ecosystem-qualification`에서 시작했다. 사용자 지시에 따라
원격 main을 다시 fetch하고 로컬 main을 같은 merge SHA로 fast-forward한 뒤
`milestone/m12-go-cargo-terraform`을 생성하여 기존 checkpoint를 그대로 이어갔다. M12-001은 COMPLETE를 유지한다.

`IMPLEMENTED=YES`, `WIRED=NO`, `QUALIFIED=NO`, `ACCEPTANCE_CLOSED=NO`다.
기존 `artifact/gomodule` parser, `sandbox/gomodule_resolver.go`/`gomodule_build.go`,
`promotion/go_get.go`/`go_transaction.go`, `application/gomodule.go`,
`domain/project_snapshot.go`, CLI/bootstrap을 재사용한다. 기본 단위/stub 회귀의
존재를 실제 production qualification으로 확대하지 않는다.

`MISSING`: resolved exact graph/content의 검사·Policy·Evidence와 transaction 연결,
재선택 없는 approved verified cache 승격, 실제 isolated/observed offline build와
output publication, Host에서 project/replace/toolchain content를 실행하지 않는
resolver 경계, pinned runtime 및 정상·tamper·cache poisoning·ambient env·race·
rollback/cleanup과 기존 직접 소비자의 실제 qualification이다. 현 `go get`은
resolution 뒤 별도 network-enabled selection을 하고 `go mod download`는
resolution만 보고한다. `GoBuildRunner`는 CLI에 연결되지 않았으며 현재 RunGo는
Host 실행이다. 이를 성공한 검사·승격·격리 build로 기록하지 않는다.

작업 범위는 기존 보안 모델 안에서 M12-002의 acceptance gap을 채우는 것이다.
현재 하나의 IN_PROGRESS만 유지하며 local checkpoint를 남긴다.
NO PUSH / NO REMOTE CI / NO MERGE. 다음 M12-003은 NOT_STARTED다.

### M12-002 parser/private snapshot checkpoint (2026-10-04)

[Parser baseline evidence](./evidence/m12-002-parser-baseline/result.json)는
실제 Go command 형식과 기존 stub의 차이, 수정 전 FAIL과 직접 정상/negative
회귀를 연결한다. Existing parser/resolver에 bounded JSON stream, official module
identity/escaping, project-bound main/synthetic/selected-MVS graph, checksum/control
binding, private-control reads와 cleanup failure 보존을 적용했다. x/mod dependency
검토는 [Coding/security rules](../engineering/02-coding-security-rules.md)에 있다.

이 checkpoint는 Go full operation closure가 아니다. Origin metadata는 source
attestation을 대신하지 않으며, host runner·재선택 promoter·검사/Policy/verified
cache·실제 isolated build/output publish와 qualification은 위 MISSING에 남는다.
`M12-002 IN_PROGRESS / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`를 유지한다.
PyTorch/공통 observer/기존 runtime lock/workflow·Policy 입력은 변경하지 않았다.

### M12-002 acquisition/SumDB checkpoint (2026-10-04)

[Acquisition evidence](./evidence/m12-002-acquisition/result.json)에 exact public
`.mod`/`.zip` intake, independent signed SumDB/Merkle/h1 verification, bounded
non-executing static inspection과 기존 entry Policy/Evidence 경로의 실제 빈 intake
정상/negative 검증을 연결한다. 신규 SumDB consumer에서 검출된 x/mod 인증
취약점은 official fixed `v0.40.0` pin으로 교정했고 동일 unlogged-hash fixture의
old FAIL/fixed PASS를 보존한다. 상세 source/dependency 계약은 M12 leaf와
coding/security rules가 소유한다.

이는 project command 전체 qualification이 아니다. Source-attested isolated
resolution, complete graph inspection/cache, 재선택 없는 transaction, observed
offline build/output publish와 필수 소비자 검증을 계속 진행한다.
`M12-002 IN_PROGRESS / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`를 유지한다.
M12-003/004는 아직 시작하지 않았다. NO PUSH / NO REMOTE CI / NO MERGE.

### M12-002 project inspection/verified cache checkpoint (2026-10-04)

기존 per-entry workflow와 M3/M4 complete-coverage semantics를 재사용하여 primary
없는 complete project 검사와 ALLOW-only typed verified cache staging을 연결한다.
Project checksum disagreement를 승인하던 동일 fixture의 before FAIL → after PASS,
입력/캐시 변조·누락·foreign Run/approval·incomplete checks의 fail-closed 회귀와
실제 public intake→독립 SumDB→project 검사/Evidence/Policy→cache 소비 결과는
[project/cache evidence](./evidence/m12-002-project-cache/result.json)가 소유한다.
이 fixture의 project snapshot은 명시한 qualification 입력이며 isolated resolver
실행 결과로 주장하지 않는다. `IMPLEMENTED=YES`의 재사용을 유지하고 전체
`WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`, M12-002 IN_PROGRESS를 유지한다.
남은 acceptance는 source-attested isolated resolver·재선택 없는 project transaction,
실제 observed offline build와 output publish, CLI wiring 및 전체/direct-consumer
qualification이다. 이 checkpoint 뒤 같은 item을 계속하며 다음 M12-003/004는
NOT_STARTED다. NO PUSH / NO REMOTE CI / NO MERGE.

### M12-002 isolated source/guarded transaction checkpoint (2026-10-04)

[Source/transaction evidence](./evidence/m12-002-source-transaction/result.json)에
고정 Go runtime의 registered source 경계, 실제 download/graph→complete snapshot,
하나의 private selection을 승인 후 재선택 없이 반영하는 guarded transaction과
required Evidence/cache 재검증을 연결한다. Raw project marker는 승인 권한이
아니며 기존 journal·rollback을 재사용한다. 동일 missing-record fixture의 이전
cache adapter FAIL과 보완 후 PASS, control/lock/foreign approval/journal 및
selected-byte substitution 회귀도 해당 기록의 범위로 남긴다.

실제 `go get`은 EVENT_LIMIT10000에서 실패한다. Budget scope 확인은 pending이며
한도·관찰·판정을 변경하지 않았다. 독립 download/graph source gate의 PASS를 get
PASS로 전용하지 않는다. 전체 download workflow, dependency-free project 표현,
observed offline build/output publish, complete CLI와 최종 qualification은 MISSING이다.
현재 `M12-002 IN_PROGRESS / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`를
유지하고 같은 item을 계속한다. M12-003/004는 NOT_STARTED다.
NO PUSH / NO REMOTE CI / NO MERGE.

### M12-002 guarded download/dependency-free project checkpoint (2026-10-04)

기존 download use case에 guard·complete project inspection/Policy/Evidence·verified
cache·current-snapshot transaction을 연결했다. 임의 primary 또는 두 번째 selection은
없다. Explicit dependency-free graph는 missing snapshot과 구분하고 empty payload
inventory에 bind한다. 같은 private-control drift fixture의 old resolver FAIL/new
PASS와 현재 실제 public/empty source snapshot, dependency-free CLI 최초 adoption과
retained reuse PASS 범위는 [download evidence](./evidence/m12-002-download/result.json)가 소유한다.

필수 positive product gate `TestLinuxGoGetDownloadIntegration`도 실제 실행했다.
Dependency-free project에서 exact pflag Get는 phase COMMAND/EVENT_LIMIT10000으로
FAIL하며 primary diagnostic과 원래 controls·approval 부재·guard cleanup을 보존한다.
이 gate를 expected-failure PASS로 바꾸지 않았다. 이전 사용자 지시의 budget 증가
금지가 신규 Go/Cargo profile의 최초 예산 정의에도 적용되는지 확인은 pending이다.
답변 전 기존/신규 한도와 event accounting은 변경하지 않는다.

M12-002는 IN_PROGRESS, 전체 WIRED/QUALIFIED/ACCEPTANCE_CLOSED는 NO다. Remaining
MISSING은 실제 Get와 public/transitive managed CLI download qualification,
observed offline build/output publish·build CLI, 최종 security/direct-consumer 및
CPU/CUDA 영향 검증이다. 다음 M12-003/004는 NOT_STARTED를 유지한다.
NO PUSH / NO REMOTE CI / NO MERGE.

### M12-002 Go resolver 처리량 측정 (2026-10-04)

사용자가 정상 작업량 측정을 지시하여 신규 Go만 별도 LOCAL observer의 finite
ceiling으로 조사했다. 제품 source·기존 profile budgets·charging·Policy는 불변이다.
네 fixture의 fresh source-command 36 connections 완료와 실제 처리량/CPU/memory,
전체 CLI pflag/Cobra PASS 및 x/tools/gRPC의 별도 syntax/Policy FAIL 범위는
[측정 evidence](./evidence/m12-002-resolver-budget-measurement/README.md)가 소유한다.
측정 최대 charged63,018에 근거한 신규 resolver 연결별200,000은 후보이며 제품
적용/qualification은 아직 아니다. 기존 10,000 및 후보 finite overflow 차단도 확인했다.

새로 확인한 MISSING은 Go module의 malformed negative-test assets와 일반 source의
syntax inspection 범위다. Raw filename을 승인 authority로 삼거나 package/vendor
allowlist·test skip·Policy 완화로 통과시키지 않았다. 기존 Get/build/최종 qualification
MISSING과 함께 닫아야 한다. M12-002 IN_PROGRESS, 전체 WIRED/QUALIFIED/
ACCEPTANCE_CLOSED NO와 M12-003/004 NOT_STARTED를 유지한다.
NO PUSH / NO REMOTE CI / NO MERGE.

### M12-001 종료 전 이력 (당시 상태, 현재 판정 아님)

아래 날짜별 NO·미검증·IN_PROGRESS 기록은 당시 검증 범위다. 현재 판정은 위의
종료 기록을 따른다. 과거 실패·checkpoint·원본 증거를 소급 변경하지 않는다.

M12-001 CI helper 종료 경쟁 로컬 수리 검증 (2026-10-03, baseline `87a9e2e`):
기존 공용 `scripts/ci-integration-cleanup.sh`의 lifecycle을 수정했다.
TERM 실패 뒤 동일 owner/PID/job-table/parent wait를 한 번 재확인하며 wait0만
CONFIRMED_STOPPED, 실제 nonzero wait는 그 status/UNKNOWN, live/query 불확실성은
최초 TERM 실패로 보존한다. Fixture의 ready/release·bounded polling·parent wait와
명시적 assertion으로 시간 가정 및 Bash3.2 false-PASS를 제거했다.
원본 cleanup은 두 Bash에서 두 경쟁 사례 FAIL, 수정 뒤 Bash5.2.21/3.2.57의
`-c`/파일 실행 각각 전체 fault matrix 및 lifecycle 30회(총960 child cases) PASS다.
실제 authenticated helper 시작·client 교체·복원/npm 3회, Linux trusted executor와
두 gVisor 직접 소비자 PASS; 원래 네 설치 파일 identity와 자원 부재를 확인했다.
최종 quick/security/docs와 Darwin amd64 TEST compile PASS다. 제품·Policy·observer
입력은 변경하지 않았으며 CPU/CUDA full E2E를 새 후보에서 재실행한 것으로 쓰지 않는다.
`IMPLEMENTED=YES`, `WIRED=YES`; native macOS 검증 전이므로
`QUALIFIED=NO`, `ACCEPTANCE_CLOSED=NO`다. 이 수리의 `MISSING`은 새 후보 native
macOS CI이며 Linux Bash3.2 성공으로 대체하지 않는다. M12-001은 IN_PROGRESS다.
원본·입력 hash·실행 결과는 로컬 reviews의 `m12-helper-race-87a9e2e/result.json`에
보존한다. NO PUSH / NO REMOTE CI / NO MERGE; 새 commit은 만들지 않았다.

M12-001 sdist observer 회귀 로컬 검증 완료 (2026-10-03): 같은 fixture bytes의
`1401274` PASS / `1704eb1` FAIL을 직접 대조했다. 최초 차이는 kernel-observed
`/usr/bin/uname` ARTIFACT clone의 sealed OCI root `/etc/ld.so.cache` 읽기가
runtime-root 대신 unknown/STREAM_FAULT로 분류되는 지점이다. 고정 실행 이미지의
최소 loader 읽기만 교정했고 mutable comm, 역할·network 신뢰, unexpected-exec 관찰과
기존 budget은 변경하지 않았다. 최초 build/observer/cleanup 실패는 따로 보존한다.
이번 로컬 회귀 범위는 `IMPLEMENTED=YES`, `WIRED=YES`, `QUALIFIED=YES`다.
필수 `TestLinuxPyPISdistBuildIntegration` **13.46초**, wheel **11.34초**, 네 root
utility/실패→CPU 회귀 및 ordinary의 모든 local integration을 실제 실행해 통과했다.
같은 최종 실행 후보 CPU full **499.06초**, cu126 full **2085.40초 PASS**;
원래 29개 승격, NumPy 제외와 viewer NOT_ATTESTED evidence를 확인했다.
quick/security/docs/corpus와 Darwin TEST compile은 최종 후보에서 PASS다.
Corpus는 공식 고정 입력 19 cases/18 wheels를 실제 확인했다. cu130/cu132는 focused common-path PASS이며
full은 재실행하지 않았다. 기존 `1704` remote PASS를 새 후보에 전용하지 않는다.
중간 cu126 회계 실패 두 건의 원인은 미확정으로 보존하고, 판정 불변의 bounded
수치 진단만 추가했다. 원래 설치 파일의 exact hash·소유권·mode와 자원 정리를 확인했다.
이번 로컬 회귀 범위의 acceptance는 충족했다. M12-001 전체
`ACCEPTANCE_CLOSED=NO`와 IN_PROGRESS는 유지하며 `MISSING`은 별도의
원격 Required CI/native macOS 및 milestone review·merge acceptance다.
상세 입력·원인 사슬·실행 범위는 [sdist 회귀 근거](./evidence/m12-sdist-regression/result.json)가 소유한다.
이번 작업은 NO PUSH / NO REMOTE CI / NO MERGE다.

M12-001 bounded Python library-query 진행 (2026-10-02): 사용자 승인으로 고정
read-only runtime의 exact `ldconfig -p` 조회를 ARTIFACT 역할에서 지원한다.
범위와 조건은 [M3 계약](../domain-model/12-m3-linux-dynamic-contract.md)의 bounded
library query 절이 소유한다. 기존 libutil/image identity/failed cleanup/resource-role
미커밋 correction을 보존한다. `IMPLEMENTED=YES`, `WIRED=YES`인 기존 transaction을
재구현하지 않는다. 새 capability의 negative/four-profile actual regression,
CPU full E2E(526.76초), canonical quick/security/docs는 로컬 후보에서 통과했다.
cu126 full E2E는 `cuda.bindings.cufile`의 NumPy 누락으로 비적격이며, 동일 실패
unit의 작은 재현으로 원인을 연결했다. cu130/cu132의 실제 bindings13.4.3도
`_v2.nvrtc`의 NumPy 누락을 확인했다(full qualification 아님). 선택된 graph는
NumPy를 포함하지 않는다. 사용자 승인된 [M5 inspection-only prerequisite 계약](../domain-model/14-m5-pypi-pip-contract.md)에 따라
비-root broader probes의 명시적인 leaf 입력 연결은 구현·로컬 검증되었다.
NumPy 자체 23개 단위와 bindings 전체 36/31개 단위가 하나의 기존 자원 승인 안에서
cu126/cu130/cu132 모두 완료됐다. 보조 입력은 원래 graph/승격 집합에서 제외된다.
직전 후보는 CPU 491.89초 PASS, cu126 1420.19초에 Triton viewer의 pandas
누락으로 실패했다. 2026-10-03 승인된 [M5 post-install command 관찰](../domain-model/14-m5-pypi-pip-contract.md)에 따라
console/gui-only target의 coverage를 사전에 분리했다. Required 역할 우선,
Finding/불완전 trusted evidence 차단과 승격 후 실행 enforcement 부재를 유지한다.
동일 executable 후보의 새 CPU full E2E **513.98초**, cu126 full E2E
**2078.51초 PASS**: 원래 29개 승격 대상, inspection-only NumPy 제외,
viewer `NOT_ATTESTED` Evidence/result/manifest reference를 실제 확인했다.
Triton의 원래 한-wheel closure/7 units는 cu126/cu130/cu132에서 각각
81.06/78.21/77.96초 완료했다. CUDA13 전체 qualification은 미실행이다.
Canonical quick/security/docs/required corpus 및 Darwin amd64 TEST compile
통과; native macOS/새 원격 CI 성공은 주장하지 않는다. 수동 CUDA CI에도
동일한 명시적 NumPy pin을 전달하며 CPU 입력은 empty다. 실제 workflow block의
정상/누락/잘못된 pin 회귀 및 actionlint/Required 연결을 확인했다.
`IMPLEMENTED=YES`, `WIRED=YES`, `QUALIFIED=YES`는 **로컬 CPU+대표 cu126**
범위이며 `ACCEPTANCE_CLOSED=NO`, `IN_PROGRESS`를 유지한다. `MISSING`은
현재 후보의 ordinary Required CI/native macOS 및 PR review·merge gate다.
cu130/cu132 release support는 각 full qualification 전에는 주장하지 않는다.


M12-001 clean-CI 복구 baseline (2026-10-01): `IMPLEMENTED=YES`,
`WIRED=YES`, `QUALIFIED=NO`, `ACCEPTANCE_CLOSED=NO`.
기준 `a43ccfa1265eb7b7db8d4ef559e49a4c94eda224`의 controller transaction은
유지한다. `MISSING`: default/corpus test 분리와 선언된 CI 입력 준비,
Darwin test compilation, NetworkX resource applicability 수정, 인증된 후속
integration 경로 및 clean-state 검증. 현재 작업은 M12-001 하나이며 CUDA
qualification을 수행하지 않는다.

M12-001 clean-CI 복구 결과 (2026-10-01): 위 범위의 bounded 수정과 로컬 검증을
완료하여 `READY_FOR_REVIEW`다. Clean export의 quick(전체 Go test·vet·Staticcheck·
Linux test build·platform 공통 단계), security, docs와 Darwin test compilation이
통과했다. 필수 CI corpus는 명시적 준비 후 19개 profile 사례를 통과하며, 누락·
변조 입력은 실패한다. Production packaging과 CPU PyTorch E2E, 후속 wheel/sdist·
promotion 통합도 통과했다. Snapshot·명령·원인·한계는
[복구 증거](evidence/m12-ci-repair/result.json)에 기록한다.
`QUALIFIED=NO`, `ACCEPTANCE_CLOSED=NO`는 유지한다. 남은 milestone acceptance는
remote Required CI, native macOS 및 CUDA qualification이다. CUDA 정적 matrix의
전체 bytes 미확인 항목은 cu126 11개, cu130/cu132 각각 13개이며 CLEAR로 간주하지 않는다.

M12-001 CI 연결 후속 수정 (2026-10-01): 독립 검토에서 발견한 workflow YAML
배치, Required job inventory와 EXIT cleanup 오류를 수정한다. 실제 YAML parser와
독립된 Actions 구조 정책, 변조 회귀 및 CI 공용 cleanup의 stub 실패 조합으로
검증한다. 이전 product/CPU 증거는 변경 없는 입력에만 재사용한다. 후속 결과는
[CI 연결 수정 증거](evidence/m12-ci-repair/workflow-followup.json)에 기록하며,
remote CI·native macOS·CUDA qualification을 완료한 것으로 간주하지 않는다.

M12-001 진행 근거 (2026-09-30): Model C Python 동적 관찰의 Pass 1 신뢰
런타임 기판을 구현했다. 동일한 인증 closure를 고정된 읽기 전용 volume으로
공유하고, controller가 독립 import unit마다 새 gVisor 런타임을 생성·제거하며,
systemd transaction cgroup의 누적 CPU와 다중 observer stream을 회계한다.
로컬 합성 wheel 통합, 전체 Go 테스트·vet, `scripts/check`의 platform·quick·security가
통과했다. 활성 `.pth`, entry point/script, 설치 최종 목적지 충돌 및 CPython ABI
의미 정리와 대표 PyTorch profile qualification은 남아 있다. 따라서 M12-001은
`IN_PROGRESS`이고 acceptance는 닫히지 않았다.

M12-001 Pass 2A 진행 근거 (2026-09-30): 정적 wheel role에서 import, 활성
`.pth` statement와 별도 installed-startup scenario의 typed unit을 확정하고,
인증 closure manifest에 묶은 controller 원장으로 전달한다. Entry point는 정확한
설치 모듈 관찰에만 연결하고 임의 설치 script는 `MANUAL_REVIEW`로 유지한다.
최종 설치 경로 충돌·CPython 3.14 x86_64 확장자 판정과 installer-generated
entry-point wrapper 회계를 정리했다. Digest 고정 대표 wheel 15개, 대상 Go
테스트, 전체 Go 테스트·vet, `scripts/check quick/security`는 통과했다. 별도로
실행한 합성 wheel gVisor 통합 재검사는 설치된 helper가
`/usr/libexec/heliopause/helox`만 인증하는 환경에서 Go test binary를 client로
사용했으므로 peer admission에서 중단됐다. 별도의 고정 systemd keeper 실환경
테스트는 통과했다. 인증된 production client를 통한 통합 재검증과 PyTorch
profile qualification은 남아 있으므로
acceptance는 닫히지 않았다.
Pass 2A 종료 판정: `IMPLEMENTED=YES`, `WIRED=YES` (지원되는 typed unit과
controller transaction), `QUALIFIED=NO`, `ACCEPTANCE_CLOSED=NO`.
`MISSING`: 인증된 production client를 통한 Python 동적 통합 및 대표 CUDA full
E2E qualification.

M12-001 Pass 2B 로컬 근거 (2026-09-30): 실행 권한이 없는 구형 import batch
inventory와 사용되지 않는 동적 오류 문자열 분류기·테스트를 제거했다. 15개 digest
고정 실제 wheel corpus는 `REPRESENTATIVE_REAL_CORPUS`로 통과했고, 네
PyTorch profile의 생산 resolver 정적 replay와 typed unit·누적 리소스·승격
소유권 테스트도 통과했다. 고정 gVisor 소스의 observer 빌드 및 C++ latch,
runtime bundle verifier, 전체 Go test·vet, platform·quick·security가
통과했다. Go test binary를 통한 실제 동적 통합은 설치된 privileged helper의
정상적인 peer 실행 파일 인증에서 `M5_PYPI_DYNAMIC_ACCOUNTING_UNAVAILABLE`로
중단되므로 trusted transaction end-to-end 로컬 acceptance 근거로 세지
않는다. M12-001은 `IN_PROGRESS`, `QUALIFIED=NO`, `ACCEPTANCE_CLOSED=NO`이며
`MISSING`은 인증된 production client를 통한 동적 통합과 대표 CUDA full E2E
qualification이다.

#### 이전 CPU checkpoint baseline audit (역사적 기록)

아래는 당시 판정이며 최종 qualification 후보의 상태가 아니다.

```text
M12-001 — PyTorch
IMPLEMENTED: YES
WIRED: YES
QUALIFIED: PARTIAL (CPU: QUALIFIED / CUDA: UNQUALIFIED)
ACCEPTANCE_CLOSED: NO
Status: IN_PROGRESS (CPU qualification checkpoint complete)
Evidence (PyTorch CPU):
- target: torch@2.9.1+cpu (source: pytorch:cpu)
- exact 10-distribution closure (torch owned by pytorch-cpu; 9 transitives owned by pypi)
- production CLI promotion PASS via helox pip install into active Python 3.14 venv
- 14,480 promoted files (site: 14,476, scripts: 3, data: 1)
- site/bin/share multi-root transaction reconciliation PASS
- real scheme outputs verified (bin/isympy mode 0755, share/man/man1/isympy.1)
- unrelated baseline adoption = 0, unrelated baseline files preserved unchanged
- Artifact network attempts = 0 (--network none verified)
- Zero Real Authority = PASS (cap-drop ALL, no-new-privileges, non-root user, read-only root)
- final security invariants = PASS
- functional check: torch import, version 2.9.1+cpu, CPU tensor op PASS, cuda.is_available() == False
- canonical tests/gates = PASS (go test ./..., format, quick, security, foundation, docs, diff --check)
- PyTorch CPU promotion tmpfs = 1 GiB, dynamic CPU limit = 30 CPU-seconds
- observation event cap = 10,000 (configured observation budget/cap, not measured runtime usage; raw numeric observer counters not all retained)
Explicit limitations:
- 위의 CPU 2.9.1 기록은 이전 checkpoint의 evidence다. 현재
  `01d07bffe776077a92f8cb7ad32f61fc640d1d95` ordinary CI와
  `torch@2.14.0+cpu` full integration은 PASS다.
- 같은 checkpoint의 수동 `torch@2.14.0+cu126` full integration은 static
  inspection 이후 첫 empty-import wheel에서 FAIL했다. 현재 작업 트리의
  임의 디렉터리 import 추론 제거, Python native extension과 .py 표면 union,
  fail-closed proven-no-dynamic-surface 모델 및 동일 내용의 shared site-file ownership 수정은
  local check를 통과했지만 독립 재감사와 CUDA runtime qualification은 남아 있다.
- `cu126`, `cu130`, `cu132`는 canonical lock에 등록되어 있으며 CUDA full
  integration PASS evidence는 아직 없다.
MISSING:
- 현재 작업 트리 변경의 독립 재감사 및 최소 한 pinned CUDA full E2E PASS
- CUDA qualification evidence에 따른 M12-001 acceptance closure
```

### M12-005에 남는 release support 결정

Deferred planning note: M12-001은 기존 representative pinned CUDA qualification
기준으로 닫고, broader first-release PyTorch/CUDA release support matrix 확정은
M12-005 feature freeze에서 수행한다. canonical lock 등록만으로 release support를
주장하지 않는다. 상세 계약은
`16-m12-01-ecosystem-expansion-contract.md`를 따른다.

### M13 Production Release & Operations queue (M12 완료 후)

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M13-001 | protected main/develop/tag/release environment/immutable release activation | COMPLETE |
| 2 | M13-002 | `feature → develop → main` 실제 protected PR qualification | ACCEPTED_WITH_EXCEPTION — C3; actual missing-CLA NOT_TESTED / DEFERRED |
| 3 | M13-003 | canonical release build·manifest·attestation·OS package assets | COMPLETE — protected v0.1.0 candidate qualification |
| 3a | CLI-UX-001 | multilingual terminal progress and structured human results | COMPLETE — local source / installed CLI qualification |
| 3b | CLI-UX-002 | OS preferred display-language auto | COMPLETE — native macOS/CI328 and local installed CLI PASS |
| 3c | SEC-UPD-001 | guarded ecosystem update commands | COMPLETE — final sourcecdac/CI344 qualified |
| 3d | CLI-UX-003 | branded interactive launcher | COMPLETE — source4cfe29e/CI352 ten SUCCESS |
| 4 | M13-004 | installed self-quarantine update / convenience channels | IN_PROGRESS — self-quarantine core; channel names pending separately |
| 5 | M13-005 | exact publication·clean Host bootstrap·doctor·cross-ecosystem smoke·quarantine | NOT_STARTED |

### M13-001 — protected repository activation

Status: COMPLETE — live configuration/readback scope
Baseline: main2e196a1; PR29/30 merged, main CI38328099413 ten SUCCESS.
Read: [M13 contract](./18-m13-production-release-operations-contract.md),
[M10 distribution](./13-m10-verified-distribution-bootstrap-contract.md),
[CI gate](../engineering/04-ci-quality-gate.md), CONTRIBUTING.md.

Baseline audit: IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO
for complete live protection. Existing M10 implementation and the disabled main
ruleset are reused; per-responsibility statuses and exact evidence are in the
[activation audit](./evidence/m13-001-protected-activation/README.md).

Scope:
- Confirm delivery and distinguish remaining guest/Race/module findings.
- Activate existing main protection and create/protect develop at qualified main.
- Restrict v* creation and prohibit tag modification/deletion; configure release
  environment approval and exact workflow-ref policy; enable immutable releases.
- Capture and verify live state independently of local source checks.

Acceptance:
1. Existing implementation is identified before changes; missing settings only
   are applied from the versioned plan.
2. Main/develop require PR, resolved conversations, up-to-date Required and
   license/cla from their pinned apps, with no bypass; deletion/force push blocked.
3. Tag creation authority is separate from no-bypass update/deletion protection.
4. Release environment requires the owner review and permits only main execution;
   immutable releases enabled. No tag/publication is triggered by this item.
5. Fresh API readback matches all settings; canonical docs/workflow/release-gate
   checks pass, and before/after receipts are preserved without credentials.
6. Remaining release findings retain owners/reopen conditions. Public release and
   M13-002 hostile PR/push qualification are not claimed by configuration readback.

Current audit: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES
for M13-001 configuration acceptance. Rules21396883/24969937/24969939/24969940,
release environment24138487071 with owner review and admin bypass disabled,
main-only deployment policy62857129 and immutable releases match the exact plan.
Canonical docs/workflow/release-gate PASS; final closure docs/security checked
before commit. MISSING: none in M13-001 scope. No item was IN_PROGRESS at that closure.
Before the user-requested hold M13-002 was IN_PROGRESS; it is now ON_HOLD.
M13-003..005 remain NOT_STARTED. Settings activation does not close the overall
no-known-release-blocking-finding entry assessment.

### M13-002 — protected branch qualification

Status: ACCEPTED_WITH_EXCEPTION — user approved C3 for the initial maintainer-only
release. Actual missing-CLA remains NOT_TESTED / DEFERRED; full QUALIFIED=NO.

Read: [M13 contract](./18-m13-production-release-operations-contract.md#m13-002--protected-branch-qualification),
[C3 approval](./evidence/m13-002-protected-path/cla-deferral-proposal-20261017.md),
[post-delivery evidence](./evidence/m13-002-protected-path/post-delivery-review-20261017.json).

Audit: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=YES
within the explicitly revised C3 acceptance scope. Normal PR35/36 delivery and
actual main CI324 are verified; PR32/33 are CLOSED/unmerged. Develop Required-only
suite4574545429 and approved disposable nonFF suite4575025940 close their bounded
gaps. Existing direct/conversation/ordinary-admin negatives and accepted bounded
operational Strict evidence are reused; independent freshness-only NOT_QUALIFIED
and exact-target hostile force NOT_RUN within the accepted method remain visible.
No replay or fake identity/status/signature reset is proposed.

MISSING (deferred evidence): actual eligible missing-CLA server rejection. A genuine
unsigned contributor remains unavailable. Owner: repository maintainer. Deadline:
complete the real test before the first external-contribution merge; until then
external contributions cannot merge. Unknown protection/App identity or actual
failure suspends C3. The single canonical C3 clause owns scope and release limits.
M13-003 qualification may proceed in order; no particular merge/tag/publish is
authorized by this acceptance. M13-004/005 remain NOT_STARTED. Historical paragraphs
below do not restore the earlier C2 blocker or grant new writes.

The following is historical pre-hold evidence; its PR33 merge proposal and CLEAN
state are not current execution authority:
A′ completed within5 approved writes. PR33 at exact
develop84591ef→main2e196a1 has fresh10job/CLA success and is open/unmerged/CLEAN
after main Required-only and conversation/ordinary-admin rejection plus test-thread
resolution. All three refs are unchanged. Normal merge is not authorized; remaining
target-specific cases/dispositions and final audit keep qualification open.
Current execution update: the later approved bundle A stopped at A3 after3/8
writes. Both target direct-push PR-gate rejections passed; PR32 retarget succeeded
but its main gate remained BLOCKED after read-only settlement. A4..A8 NOT_RUN,
zero merge requests/new PRs. Current PR32 base is main, not develop. See the
[bundle result](./evidence/m13-002-protected-path/bundle-a-result.json) and
[unapproved revised scope](./evidence/m13-002-protected-path/bundle-a-resume-plan.json).
Latest phase: user approved A′ maximum5 writes. PR33 is now the actual normal
develop84591ef→main2e196a1 candidate, original18 docs/evidence paths. One ordinary
exact-head PUT with realCLA success/Required absent was rejected405 explicitly
for `Required`; suite4521104354 fails only required_status_checks and refs are
unchanged. [A′ result](./evidence/m13-002-protected-path/bundle-a-prime-result.json)
is COMPLETE_WITHIN_APPROVED_SCOPE after exactly5/5 writes. Same natural
CI38574753115/attempt1 completed all10jobs SUCCESS with genuine requiredCLA;
CLEAN preceded one diff thread. Exact-head ordinary admin PUT returned405 for
conversation resolution only, suite4521761091 pull_request failure; refs unchanged.
Only threadPRRT_kwDOT2M-Hs6sXNWE was resolved, restoring CLEAN/threads0. PR33 is
still open/unmerged at84591ef/main2e196a1, all three refs unchanged. No PR32 mutation,
rerun, force, protection edit, release or positive merge.
[Exact normal merge approval](./evidence/m13-002-protected-path/normal-main-merge-approval.json)
is ready for user review, not authorized: one ordinary PR33 exacthead84591ef PUT,
then matching main receipt/ref/parents/tree and newmainCI readback. Intermediate
normal qualification only; no dependent stale call/final closure is included.
CLA negative stays NOT_TESTED; main stale/develop Required-only/nonFF method,
record delivery and final audit remain open. Overall audit remains IMPLEMENTED=YES /
WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO; M13-003..005 remain NOT_STARTED.
Read: [M13 contract](./18-m13-production-release-operations-contract.md),
[CI gate](../engineering/04-ci-quality-gate.md), CONTRIBUTING.md,
[M13-001 evidence](./evidence/m13-001-protected-activation/README.md).

Baseline audit: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO. Existing PR CI/CLA and active rules are reused unchanged.
Initial baseline GET/refs confirmed main/develop2e196a1, milestoneb3c8b7d,
zero feature refs/open PRs and zero checks/statuses for b3c8b7d. The admin actor
cannot bypass either branch ruleset. M13-001 is remotely delivered on the
milestone branch; its documentation delta has not reached main/develop.

Acceptance: actual feature → develop → main protected PR merges with exact fresh
Required/CLA evidence; server enforcement of stale/missing checks, unresolved
threads, direct pushes, non-fast-forward updates and admin bypass attempts;
preserved request/result/ref evidence and final checks. A readback, no-op push,
unrelated rejection or historical CI is not a new enforcement PASS.

Pre-merge delivery snapshot: [PR31](https://github.com/rahoney/heliopause/pull/31),
feature/m13-002-protected-path@e1e3aa8 → develop@2e196a1. New
[CI38355837742](https://github.com/rahoney/heliopause/actions/runs/38355837742)
completed SUCCESS with Required plus all nine prerequisites, independently of
historical main CI. Actual CLA service reports license/cla success. Both contexts
are isRequired=true on PR31, with App identities15368/128106. Main/develop remain
at2e196a1; rules are unchanged. The tested merge commit6950673 has the same whole
tree64e2961 as e1e3aa8. Gate BLOCKED during incomplete checks became CLEAN after
success; that snapshot was open/unmerged, auto-merge null, review threads0. All ten raw
job logs and fresh API/GraphQL receipts are preserved locally and bound by the
[new remote result](./evidence/m13-002-protected-path/remote-result.json).

New actual case: [conversation rejection](./evidence/m13-002-protected-path/thread-rejection.json)
PASS on develop/PR31. One real diff thread (comment4258640332) changed CLEAN→BLOCKED;
Required/CLA stayed green and required at exact e1e3aa8. The separately approved
single ordinary exact-head merge request returned HTTP405 explicitly citing the
unresolved conversation. Main/develop stayed at2e196a1, PR remained open/unmerged,
and resolving that test thread restored CLEAN. No bypass/force/auto-merge was used.

MISSING: independent stale disposition, remaining target-specific negative cases, normal develop-to-main merge and
final audit. This develop conversation case does not qualify main, missing
Required/CLA, stale branch, direct/force push or a separate override attempt.
The single-attempt conversation approval is consumed. A subsequent scoped approval
allowed independent feature/m13-002-stale-probe creation/push and develop PR32.
PR32 is headff441d3 with sole parent2e196a1, one new12-line markdown file and no
M13-001/PR31/local-record commits in its ancestry. Its paths are disjoint from
PR31 and local merge-tree has no conflict. Its own CI38403196699 completed all
ten jobs SUCCESS and genuine CLA succeeded; both contexts are required and its
pre-advancement gate was CLEAN. [Preparation evidence](./evidence/m13-002-protected-path/stale-pr-preparation.json)
binds these fresh results.

Previous authorized phase: [normal merge and stale observation](./evidence/m13-002-protected-path/develop-merge.json).
Both PRs were CLEAN with exact heads, required successful contexts, no unresolved
threads and unchanged protections immediately before one normal exact-head PR31
merge request. HTTP200/merged=true advanced develop to84591ef; its parents are
2e196a1/e1e3aa8 and tree64e2961 matches the qualified PR31 candidate. Main stayed
2e196a1. PR32 remains open at exactff441d3 and changed CLEAN→BEHIND, with successful
Required/CLA, no unresolved threads and no content conflict. No automatic CI rerun
was observed through 2026-10-13T14:41:12Z (one run, attempt1). PR base/merge metadata
still identifies the old tested snapshot; live refs, fetched commit and ancestry
comparison establish actual develop advancement. Old checks are not new-base CI.
At that phase, actual stale rejection was NOT_RUN and no PR32 merge request had
been sent. The subsequent one-shot execution below supersedes that pending label.

Executed phase: [PR32 rejection result](./evidence/m13-002-protected-path/stale-rejection.json).
On 2026-10-14 the explicitly approved ordinary exact-head merge request was sent
once after fresh HEADff441d3/develop84591ef/BEHIND, successful required Required/CLA,
zero unresolved threads and admin bypass never were confirmed. HTTP405 reports
`2 of 2 required status checks are expected.` Main2e196a1, develop84591ef and
featureff441d3 are unchanged; PR32 is open/unmerged/BEHIND. Its synthetic merge
metadata changed8e5067f→3635960 (advanced develop/head parents, zero checks/statuses),
which is not a protected-branch update. The specific internal refresh/evaluation
cause is unknown. This is a real rejection with an unexpected reported blocker,
not an isolated stale, missing Required, missing CLA or separate bypass PASS.
The one-shot authorization is consumed; no retry or other mutation was made.
Read-only follow-up: [Strict-check analysis](./evidence/m13-002-protected-path/strict-check-analysis.json)
correlates rule-suite4516024742 with the original request by target/before SHA,
actor, normalized time and PR32 commit/parents. The actual proposed update is
0fd006b, not headff441d3 or potential merge3635960. Only required_status_checks
in rule24969937 failed; its exact contexts are Required/App15368 and
license/cla/App128106. The head retains both successes;0fd006b/3635960 each lack
checks/statuses and share advanced-base parents/tree. This proves the bounded
Strict-rule stale-candidate rejection, not freshness-only causality. Original
stale-rejection bytes and its independent NOT_QUALIFIED label are preserved.
No remote mutation or CI was triggered; the same develop attempt will not be repeated.
The [remaining qualification plan](./evidence/m13-002-protected-path/README.md#remaining-qualification-and-approval-plan)
maps existing PR31 ordinary-admin/conversation proof without duplicate calls and
combines future main conversation/admin coverage. It separates bounded doc-only
forward operations from final reviewer main merge and force-push scope/risk.
The earlier generic plan listed evidence/participant/force method decisions;
the current concrete preparation and confirmed CLA deferral below supersede that
next-step list. A protected disposable probe cannot silently replace exact-main/
develop hostile evidence; no harmless
live non-fast-forward update is guaranteed if enforcement fails. Prepare exact
candidate/action inventories before requesting a forward-only approval bundle.
Target-specific remaining negatives, normal develop-to-main and final audit stay
open. Current audit remains IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO. M13-003..005 remain NOT_STARTED.

Current local preparation (source record7d65a05): the
[concrete forward-only plan](./evidence/m13-002-protected-path/forward-only-plan.json)
freezes two one-file direct-push candidates (main6b3ae56, develop68916c7), exact
current refs/diffs and an eight-write future approval scope. This scope combines
the two expected direct-push rejections, PR32 main retarget/thread/ordinary-admin
rejection/resolve, and develop84591ef→main2e196a1 PR creation plus a genuine
Required-only opportunity during its needed CI. No successful merge is included.
Original develop Strict rejection is reused within its bounded proven scope;
no repeat or independent freshness-only claim is proposed. Future allocated PR
and thread IDs bind only to exact successful receipts/readback.

The user confirms no actual unsigned contributor is available: isolated CLA
rejection is NOT_TESTED and deferred, with required App128106 wiring and actual
signed-PR success preserved. No account/fake contribution/signature/CLA change
will manufacture it. Official hosted-app documentation revealed no applicable
safe live negative mode in the reviewed sources; recheck/self-hosted tests do not
replace real production-App enforcement. Other qualification may proceed, but
unchanged M13-002 acceptance explicitly requires missing-CLA rejection: closure
needs real evidence or an explicit canonical disposition. NOT_TESTED is not PASS
or a broken CLA service result. Formal release remains unready under the ordered
M13 plan, independently of the current signed PR's successful CLA status.

Latest ordering supersedes the earlier generic plan where final main merge
preceded main stale evidence. A separate intermediate develop→main advancement
can first produce the actual main SHA for a main stale case; later forward
delivery incorporates that ancestry. Each future unknown SHA/diff is frozen for
its own approval, and the final develop→main merge remains separately approved
after required negatives/gap disposition. Actual main/develop non-fast-forward
and disposable-probe setup/method decisions remain separate. No remote write,
new CI, protection edit, release or M13-003 work occurred during preparation.
Next: review the finite scope A; no execution authorization is implied.

Bundle A execution (supersedes the preceding preparation's next action): direct
main6b3ae56 and develop68916c7 pushes each received GH013 with explicit PR-rule
failure plus expected-check failure, suites4519783788/4519788734. Both refs stayed
main2e196a1/develop84591ef. Retarget PR32/headff to main returned200; subsequent
REST/GraphQL BLOCKED prevented the planned isolated conversation case. Head
Required/CLA remain successful/required and threads0; actual main→head is1ahead/
0behind, while potential3635960 retains obsolete develop-parent metadata. That
observation does not establish the internal blocker. Writes stopped permanently
for this invocation at3/8; no merge API request, thread, new PR, rerun or repair.
Original frozen plan and prior results remain unchanged. CLA stays NOT_TESTED.

Next approval proposal: five bounded writes on one necessary actual develop→main
PR (create, guarded Required-only negative, then green-CI thread/conversation-admin
negative/resolve), exacthead84591ef/base2e196a1, existing18 documentation/evidence
paths. PR32 remains untouched and direct cases are reused. This revised scope is
not authorized; the normal positive merge approval is not ready until actual PR,
fresh CI/CLA and required negative results/dispositions exist. No guessed PR ID
or historical CI substitution. Current audit remains IMPLEMENTED=YES / WIRED=YES /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO; M13-003..005 stay NOT_STARTED.

Force-push needs a design/decision compatible with existing history preservation;
genuine missing CLA may need a willing real unsigned participant. Do not weaken
rules, fabricate statuses/identity or mark those cases waived/qualified.
Detailed baseline, target-specific matrix, evidence custody and authorization
boundaries: [qualification plan](./evidence/m13-002-protected-path/README.md).
Release tags/publication and M13-003 implementation remain outside this task.

### M13-003 — canonical release build and OS packages

Status: COMPLETE. Audit: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES /
ACCEPTANCE_CLOSED=YES / MISSING=none within the build/package scope.
Read: [M13 contract](./18-m13-production-release-operations-contract.md#m13-003--canonical-release-build-and-os-packages),
[M10 distribution contract](./13-m10-verified-distribution-bootstrap-contract.md)
and [CI gate](../engineering/04-ci-quality-gate.md).

Existing native/helper/runtime/installer responsibilities were reused. Actual
source CI325/326, approved PR37 ordinary merge, exact new main CI327 and separately
approved protected v0.1.0 build/run39036210335 are verified. Manifest-first and all12
actual-file attestations match exact source/tag/workflow/run; fixed-root offline
positive and three negatives passed. Actual network-disabled Ubuntu/Rocky native
package install/reinstall/remove, payload ownership/modes, unit guards, preserved
caller config and truthful unavailable doctor passed. No booted Host or public
publication success is inferred. Those remain M13-005 per the canonical contract;
the preparation-era combined MISSING list is reconciled in the qualification report.

Evidence: [baseline](./evidence/m13-003-release-build/baseline-20261017.md),
[preparation](./evidence/m13-003-release-build/preparation-20261017.md),
[actual qualification](./evidence/m13-003-release-build/qualification-20261017.md)
and [bound result](./evidence/m13-003-release-build/qualification-20261017.json).
Actions candidate retention expires2026-10-24T13:29:00Z; later publication requires
fresh exact-candidate availability/provenance checks, not a moved tag.
Completed payloads/tools/containers/new images are removed; private check caches
are removed after last use. Necessary evidence and existing shared/runtime files remain.

### CLI-UX-001 — user-facing terminal remediation

Status: COMPLETE — local source and installed development CLI qualification.
IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES for the
[terminal presentation contract](../user-journey-cli-ia/05-terminal-presentation.md).
[Baseline](./evidence/m13-cli-ux/baseline-20261018.md) preserves the initial gaps;
[qualification](./evidence/m13-cli-ux/qualification-20261018.md) owns the final
checks, seven-language PTY cases, actual npm inspection/install/JSON/failure,
protected local CLI activation/rollback and cleanup boundary. MISSING: none in
this bounded item. Native macOS execution and full ecosystem/CUDA reruns are not
claimed by local UX qualification. Inspection/Policy/exit codes are retained.
Public v0.1.0 predates this UI source and was not retagged or republished.

### Prepublication corrections — CLI-UX-002 then SEC-UPD-001

User clarified that auto means OS display-language preference and that guarded
updates are essential to protecting existing users from compromised new releases.
[Baseline audit and order](./evidence/prepublication-gaps/baseline-20261018.md)
precedes source changes. Existing inspection/Policy, transactions, settings and
catalogs are reused; no replacement or new runtime/CI scaffold is justified.

- SEC-UPD-001: COMPLETE. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES /
  ACCEPTANCE_CLOSED=YES / MISSING=none for the documented bounded coverage.
  Final sourcecdac/CI344 ten SUCCESS, exact whole tree/native macOS/Linux45/
  CPU291.84s/five helperwait0/final cleanup closes the added real-M6 regression.
  [Cross-lane review](./evidence/update-quarantine/final-review-20261019.md) maps
  each acceptance to existing owners and [normalized closure](./evidence/update-quarantine/ci344-closure.json).
  Read [update quarantine contract](../user-journey-cli-ia/06-update-quarantine.md).
- CLI-UX-002: COMPLETE. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES /
  ACCEPTANCE_CLOSED=YES / MISSING=none for the documented bounded OS-language scope.
  Exact local source checks, WSL/PTY16 cases, installed auto/JSON inspect and
  ordinary CI328 ten jobs including native macOS passed. [Qualification](./evidence/cli-os-language/qualification-20261018.md).
  Read [OS-language correction](../user-journey-cli-ia/05-terminal-presentation.md#os-display-language-correction--cli-ux-002).

The detailed plans, second review and CLI-UX-002 implementation are complete.
On2026-10-18 the user authorized SEC-UPD-001 and deferred CLI-UX-003 until update
acceptance. At activation only SEC-UPD-001 was IN_PROGRESS; ecosystems were implemented serially.
Current acceptance is the table below;
source-specific failures, checks and historical intermediate states belong to the
linked qualification owners rather than duplicate progress narratives here.

| Scoped update lane | IMPLEMENTED / WIRED / QUALIFIED / ACCEPTANCE_CLOSED | Evidence / remaining work |
| --- | --- | --- |
| npm | YES / YES / YES / YES | [Bounded npm qualification](./evidence/update-quarantine/npm-qualification-20261018.md), source29c58f0/CI330; actual native/managed consumers, restoration and cleanup |
| Public PyPI managed venv | YES / YES / YES / YES | [Managed qualification](./evidence/update-quarantine/pypi-managed-qualification-20261019.md), sourcec93ce58/CI331; actual update/fresh no-op/refusals and bounded retired-metadata correction |
| Bounded public-native PyPI enrollment | YES / YES / YES / YES | [Native qualification](./evidence/update-quarantine/pypi-native-qualification-20261019.md), source2e8844f/CI332 and separately scoped affected retirement regressions; authentic old wheel footprint, foreign-file protection and exact held transaction |
| Named PyTorch managed venv, CPU/cu126/cu130/cu132 | YES / YES / YES / YES | [Final named qualification](./evidence/update-quarantine/pytorch-update-qualification-20261019.md), source306/CI336; all four actual version updates/fresh no-ops/unchanged refusals, confirmed helper stop/original restoration/postguards and completed data/operator cleanup |
| Cargo lock update | YES / YES / YES / YES | [Reviewed audit](./evidence/update-quarantine/cargo-update-baseline-20261019.md) and [current qualification](./evidence/update-quarantine/cargo-update-qualification-20261019.md); source85ac4c1; final local checks and20 installed phases including old add/build, dependency-free/workspace, restoration and145.90MB cleanup PASS; CI338 ten SUCCESS; same checkout tree,36 related Go/Cargo/Terraform consumers, CPU277.12s/five helper wait0/final confirmed cleanup |
| Scoped Go get -u | YES / YES / YES / YES | [Reviewed audit/plan](./evidence/update-quarantine/go-update-baseline-20261019.md) and [metadata/pruning amendment](./evidence/update-quarantine/go-mvs-metadata-correction-20261019.md) precede their sources. [Current qualification](./evidence/update-quarantine/go-update-qualification-20261019.md): sourcee00 canonical Quick206.98s/security10.79s/docs/Race129.57s and installed15 phases PASS, including Cobra metadata build65.14s and old grpc41-module get114.11s/download87.14s;212 new update/grpc Evidence, canonical restoration/postguards and568.85MB new trial cleanup. Earlier394 CI340 FAILED at pruned lower checksum membership. CI341 ten SUCCESS; actual checkout whole-tree equality/native macOS,36 existing Go/Cargo/Terraform consumers/CPU192.40s/five helperwait0/final confirmed cleanup; MISSING none for bounded Go scope |
| Terraform project upgrade | YES / YES / YES / YES | [Pre-source audit](./evidence/update-quarantine/terraform-update-baseline-20261019.md) and [final qualification](./evidence/update-quarantine/terraform-update-qualification-20261019.md): sourcef882921 final canonical/related Race and signer-proof Race, actual16 phases/48 independent Evidence, restoration/postguards and718.98MB completed trial cleanup PASS. CI342 ten SUCCESS, exact checkout whole-tree/native macOS,36 existing Go/Cargo/Terraform consumers/CPU192.80s/five helperwait0/final confirmed cleanup. MISSING none for bounded Terraform scope |
| GitHub managed replacement | YES / YES / YES / YES | Final lane; [independent baseline/edit map/second review](./evidence/update-quarantine/github-replacement-baseline-20261019.md) precedes sourcecb1f944. [Qualification](./evidence/update-quarantine/github-replacement-qualification-20261019.md): same-input Quick150.58s/security8.75s/docs/full related Race221.88s; actual18 GitHub calls49.06s/17 fresh success Evidence plus15 affected npm/PyPI/Cargo/Go/Terraform calls241.65s/32 fresh success Evidence PASS. Canonical helperwait0/original restoration/fresh guard and353343187B completed trial cleanup PASS. CI343 ten SUCCESS/exact tree/native macOS/Linux45/CPU267.82s/five helperwait0/final cleanup PASS. Final sourcecdac/CI344 ten SUCCESS/exact tree/native macOS/Linux45/CPU291.84s/five helperwait0/final cleanup closes the added M6 test/document scope; MISSING none. [Shared lock correction](./evidence/update-quarantine/transaction-lock-identity-correction-20261019.md) acceptance is closed |

The qualified rows retain their source-specific bounded actual/CI scopes in the
linked reports. Broader native named PyPI footprints remain unsupported. Go's
original metadata/build failure and CI340 pruning failure stay historical; the
reviewed amendment and final sourcee00/actual15 phases/CI341 close bounded Go only.

SEC-UPD-001 acceptance is closed. The [final review](./evidence/update-quarantine/final-review-20261019.md)
reconciles per-lane actual/negative evidence, real-M6 adverse-update coverage and
sourcecdac/CI344 exact-source remote acceptance. Existing source/runtime/Policy
limits are preserved. The [shared FD correction](./evidence/update-quarantine/transaction-lock-identity-correction-20261019.md)
is also qualified. No completed local update test payload/cache/candidate/operator
or restored binary backup remains; raw/normalized proofs and shared SDK tools remain.

Follow-up CLI-UX-003 is COMPLETE at source4cfe29e/CI352; prior failures and narrower successes remain historical.
SEC-UPD-001 has no remaining acceptance work. Main merge, permanent local CLI activation
and publication remain separate; PR38 is draft/unmerged. Native commands outside
helox are not transparently intercepted. Earlier source/checkpoint descriptions
retain their original scope, superseded by the current acceptance table above.

### CLI-UX-003 — branded interactive launcher

Status: COMPLETE. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES /
ACCEPTANCE_CLOSED=YES. MISSING: none in this launcher scope. Source4cfe29e and
actual checkoutf3d6816 share treef0f700a; CI352 has Required and nine prerequisites
SUCCESS, including native macOS and Linux45 actual consumers/CPU/helper cleanup.
[Qualification](./evidence/cli-launcher/qualification-20261020.md) owns the exact
scope, historical failures and cleanup. The pre-source storage amendment preserves
all limits, original scale tests and final real durable-publication barrier.
The old CI351 exact slow test/cause remains UNKNOWN; a new success is not causal proof.
Final documentation-only delivery is evaluated on its own exact head; raw receipts
record that result without changing runtime qualification scope. Main merge,
permanent installed CLI activation and publication remain separate; PR38 is draft.

### M13-004 — convenience installation channels

Current P1 substep: [C3c2b managed UV CLI](./evidence/package-client-quarantine/p1c-installed-cli-ci380-20261022.json) is qualified for fresh/retained/new-root install and the two upgrade modes. Source d1d4fdc/CI380 has10SUCCESS with the same whole checkout tree, actual five-step CLI and existing Linux/macOS consumers.
[C3d native UV enrollment](./evidence/package-client-quarantine/p1d-native-uv-ownership-baseline-20261022.md) is qualified/closed in its bounded scope by [CI383 ten SUCCESS](./evidence/package-client-quarantine/p1d-native-uv-ci383-20261023.json). Native actual adoption/replay, existing Linux consumers and native macOS pass without budget/source-performance changes. Next is [C3e1 complete/multi-root/explicit-empty sync format and binding](./evidence/package-client-quarantine/p1e-uv-sync-baseline-20261022.md); audit/edit map/second review and [native plain input semantics](./evidence/package-client-quarantine/p1e-uv-sync-format-20261023.json) precede implementation. Whole P1 and P2..P7 remain incomplete/NOT_STARTED respectively.
Cache-aligned [CI374](./evidence/package-client-quarantine/cache-alignment-ci374-20261022.json) passed native macOS and all10 jobs. Historical timeout causes are not universally confirmed; scope/deadlines remain unchanged.

Status: IN_PROGRESS — self-update installation role and Linux/WSL2 amd64 scope approved on2026-10-20; official names pending only for channel wiring. CLI-UX-003, SEC-UPD-001 and CLI-UX-002 remain complete.
Audit: IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO for the
complete channel scope. Existing verifier-gated native installer/doctor and release
manifest/build/publish consumers are reused; their historical acceptance is retained.
Read: [M13 convenience contract](./18-m13-production-release-operations-contract.md#m13-004--convenience-publication),
[M10 trust contract](./13-m10-verified-distribution-bootstrap-contract.md), directory
and Architecture rules, existing installer and workflow consumers.

[Baseline and ordered plan](./evidence/m13-004-convenience/baseline-20261017.md)
records each responsibility independently. MISSING: owner-approved official
names, bounded verified native bootstrap wiring, npm/PyPI wrappers, reviewed
Homebrew formula, protected OIDC/provenance publication wiring and channel-specific
checks. On2026-10-18 the user reported both rahoney accounts/email/2FA/GitHub
links complete. This closes account preparation only; publisher binding and all
package names remain unconfirmed. No credentials are needed.

2026-10-20 user priority: installed Heliopause must quarantine its next version;
first bootstrap is separately verified. M13-004 is the sole active item. Its
[new baseline/edit map/second review](./evidence/m13-004-convenience/self-update-20261020.md)
and [product design proposal](../user-journey-cli-ia/07-self-update.md) precede source
changes. Existing owners are reused. MISSING adds production release acquisition/
attestation, frozen-set quarantine and protected installation lifecycle. The
current user installer does not activate root-protected CLI/helpers. The proposed
separate narrow installation role and initial Linux/amd64/same-runtime scope were
approved by the owner on2026-10-20. The production provenance primitive is locally qualified as documented in
[component evidence](./evidence/m13-004-convenience/provenance-qualification-20261020.md).
Frozen discovery/public attestation/FD replay adapters are locally qualified in
[acquisition evidence](./evidence/m13-004-convenience/acquisition-qualification-20261020.md).
The actual public proof read and SDK verification pass; the replay regression uses
the existing archive InspectService/Evidence/Policy, not native activation.
The strict manifest reader and authenticity Verification Port are also locally
qualified through existing InspectService/Evidence/Policy in
[metadata evidence](./evidence/m13-004-convenience/metadata-qualification-20261020.md).
The frozen native/metadata set now connects to the existing complete-set workflow
as documented in [release-set evidence](./evidence/m13-004-convenience/release-set-qualification-20261020.md).
Local source/staging and fail-closed SDK tests pass; an actual newly signed native
target, public self-update caller and protected activation are still missing.
The shared installer's fixed protected-copy mode and verified existing-version
rollback are qualified locally in
[copy evidence](./evidence/m13-004-convenience/protected-copy-qualification-20261020.md),
including actual root/ordinary-user filesystem mechanics in disposable namespaces.
Provenance was mocked there; RPC/lease/journal/service lifecycle remains next.
Protected registration/proof reading and advisory runtime lease primitives are
also locally checked in [reader evidence](./evidence/m13-004-convenience/activation-read-qualification-20261020.md).
Actual stored v0.1 signature verifies offline; missing installer-role admission
is correctly refused. Caller authentication, lifecycle wiring and new native4
signed qualification remain missing.
The old protected controller/kernel peer gate is locally checked in
[peer evidence](./evidence/m13-004-convenience/activation-peer-qualification-20261020.md).
Actual UID/GID/inode/pidfd negatives pass in isolated namespaces with a mocked
installed crypto snapshot. Fixed payload RPC/staging and full lifecycle remain next.
Bounded payload transfer and authenticated root-private staging are locally checked in
[wire evidence](./evidence/m13-004-convenience/activation-wire-qualification-20261020.md).
Actual root filesystem mechanics pass with mocked auth/positive crypto, and production
SDK rejects the unsigned new set before root writes. Durable journal/service lifecycle,
fixed RPC and public caller remain next. Durable root journal/config snapshots and
persistent update/installer/exclusive runtime locks are locally checked in
[journal evidence](./evidence/m13-004-convenience/activation-journal-qualification-20261020.md).
Actual PREPARED writer and kernel lock mechanics pass; service/health/verified rollback,
crash recovery and full RPC/caller remain missing. Fixed service/config/drain adapters
are implemented and locally checked in
[service evidence](./evidence/m13-004-convenience/activation-service-qualification-20261020.md).
Actual process/socket and partial-config mechanics pass; real systemd/global drainage,
full root transaction/recovery, authenticated helper RPC/health and public pipeline/CLI
remain missing. Their [next source map](./evidence/m13-004-convenience/activation-lifecycle-plan-20261020.md)
precedes source edits. Shared previous-version signature/member reading, mandatory root
BundleVerifier, proof persistence and exact registration swap/restore are locally checked in
[proof evidence](./evidence/m13-004-convenience/activation-proof-qualification-20261020.md).
Actual old offline proof and existing reader/journal namespace regressions pass; actual new
signed native4/full transaction/recovery/RPC/health/public command remain missing.
Managed runtime/Doctor shared-lease wiring and the fixed inherited-channel ordinary-user
health entry are locally checked in
[health evidence](./evidence/m13-004-convenience/activation-health-qualification-20261020.md).
Actual root→ordinary FD transfer/negative admission and production CLI flag denial pass;
new signed native4 positive/full root transaction/recovery/RPC/public caller remain missing.
The root transaction/recovery and actual fixed serve/recover helper now exist, with local
[operation/socket/fault evidence](./evidence/m13-004-convenience/activation-operation-qualification-20261020.md).
Actual root filesystem mechanics pass with crypto/service/health mocks; real signed
native4/systemd/crash qualification remains open. Actual helper payload review also found
[inert default-entry gaps](./evidence/m13-004-convenience/native-entry-plan-20261020.md),
which are next before final public caller/producer/native-bootstrap wiring. Generic M6
dynamic inspection is retained; helper assets are not exempted or relabeled as safe resources.
Inert default/help entries and explicit network service consumers are now implemented and
locally qualified in [native entry evidence](./evidence/m13-004-convenience/native-entry-qualification-20261020.md).
The canonical pinned observer build/full latch and actual default/help/identity probes pass;
actual newly signed complete-set quarantine/systemd activation remain open. Next are existing
pipeline content-session binding, root client/Promotion, public self-update CLI and actual
producer/native-bootstrap wiring.
Per-request content binding now reuses the existing pipeline, with concurrent distinct-version
isolation/unchanged base and non-ALLOW publication denial locally checked in
[content-session evidence](./evidence/m13-004-convenience/content-session-qualification-20261020.md).
Actual session factory/root client/Promotion/public CLI and native bootstrap remain missing.
The exact sealed-set Promotion and fixed old-installer client now exist, with
[local copy/codec/kernel endpoint evidence](./evidence/m13-004-convenience/native-client-qualification-20261020.md).
Actual UID1000 endpoint validation passes in disposable namespaces; signed whole activation
is still unqualified. Next are per-request source/session/cleanup and public command wiring,
followed by the real helper producer/native bootstrap and signed/systemd/crash qualification.
Explicit owned replay/staging payload retention is locally checked in
[cleanup evidence](./evidence/m13-004-convenience/native-cleanup-qualification-20261020.md).
Unknown payloads and final records are retained; the next actual session must call cleanup
only after confirmed runtime drainage. [Session/CLI source map](./evidence/m13-004-convenience/session-cli-plan-20261020.md)
records that wiring and the remaining public preflight/report gaps before implementation.
The actual session factory and public `self-update --version vX.Y.Z` are now wired through
the existing pipeline, with [local session/CLI evidence](./evidence/m13-004-convenience/session-cli-qualification-20261020.md).
Exact-version admission, seven-language help/progress and truthful cleanup/recovery reporting
pass locally; actual CLI tests reject this unmanaged development installation without runtime
or acquisition changes. The actual helper asset producer/manifest/package payload is now locally qualified in
[producer evidence](./evidence/m13-004-convenience/producer-qualification-20261020.md).
Explicit registration of an independently verified, preinstalled native layout is locally
qualified in [enrollment evidence](./evidence/m13-004-convenience/enrollment-qualification-20261020.md).
The terminal cold-service recovery branch is locally checked in
[cold recovery evidence](./evidence/m13-004-convenience/cold-recovery-qualification-20261020.md).
Journal-bound system copy interruption cleanup is locally checked in
[copy interruption evidence](./evidence/m13-004-convenience/copy-interruption-qualification-20261020.md).
Fixed independently verified initial file preparation and known-source cleanup are locally
checked in [native bootstrap evidence](./evidence/m13-004-convenience/native-bootstrap-qualification-20261020.md).
Ordinary first acquisition/export and `bootstrap prepare --version vX.Y.Z` are now locally checked in
[bootstrap acquisition evidence](./evidence/m13-004-convenience/bootstrap-acquisition-qualification-20261020.md).
[Native installation handoff](../user-journey-cli-ia/08-native-bootstrap.md) records actual Root-copy
independent verification, exact OS config/unit, register and ordinary health boundaries.
Owner approved test tags/official signing builds on2026-10-20. The
[signed lifecycle plan](./evidence/m13-004-convenience/signed-lifecycle-plan-20261020.md)
fixes CI355 source a22aa12, first rc1 then rc2, unchanged tag protection and explicit retirement limits.
Both approved rc tags now have successful official signing builds on exact source a22aa12.
Rc1 signed13 independent verification and actual production Root SDK/file preparation pass locally;
registration/health/systemd remain unqualified. The
[disposable systemd fixture plan](./evidence/m13-004-convenience/native-systemd-fixture-plan-20261020.md)
precedes that qualification without changing the Host installation or product trust gates.
CURRENT MISSING: rc2 material qualification and actual complete-set caller acquisition, whole clean-host OS bootstrap/
ordinary doctor and nominal quarantine, existing OS administrator authentication, actual
systemd activation and crash/reboot qualification. Full self-update remains IN_PROGRESS and is not installed or publicly released.
Package names block only name-dependent channels;
account preparation is complete. No future channel placeholder executable/Port/job was added.
SEC-UPD-001 and CLI-UX-003 stay COMPLETE; prior CLI-UX-001 scope remains historical.
M13-005 remains NOT_STARTED. Account setup is not
package publication approval; no registry upload, tap creation, Release draft or
publish dispatch has occurred. Concrete publication remains separately reviewed.

### M12 baseline audit status

```text
M12-001 — PyTorch
IMPLEMENTED: YES
WIRED: YES
QUALIFIED: YES (CPU, cu126, cu130, cu132)
ACCEPTANCE_CLOSED: YES
Status: COMPLETE
Evidence: evidence/m12-001-closure/result.json
Qualified source: 8cb258ca882653511710b08bf8d09ffa1ef212bf
MISSING: None (M12-001 technical acceptance)
Delivery pending: closure-head Required CI, PR merge, main CI
Release support: broader first-release CUDA matrix remains an M12-005 decision

M12-002 — Go
IMPLEMENTED: YES
WIRED: YES
QUALIFIED: YES (Linux/amd64 local qualification)
ACCEPTANCE_CLOSED: YES
Status: COMPLETE
Evidence: evidence/m12-002-go-build-qualification/result.json
MISSING: None (work-item local acceptance)
Delivery pending: new remote Required CI and native macOS; NO PUSH / NO REMOTE CI / NO MERGE

M12-003 — Cargo
IMPLEMENTED: YES
WIRED: YES (public add/build production CLI)
QUALIFIED: YES (Linux/amd64 local qualification)
ACCEPTANCE_CLOSED: YES
Status: COMPLETE
Evidence: evidence/m12-003-cargo-build-qualification/result.json
MISSING: None (work-item local acceptance)
Delivery pending: new remote Required CI and native macOS; NO PUSH / NO REMOTE CI / NO MERGE

M12-004 — Terraform
IMPLEMENTED: YES (existing owners reused and acceptance gaps implemented)
WIRED: YES (public initial/retained complete-project installation CLI)
QUALIFIED: YES (Linux/amd64 local qualification)
ACCEPTANCE_CLOSED: YES
Status: COMPLETE
Evidence: evidence/m12-004-provider-qualification/result.json
MISSING: None (work-item local acceptance)
Delivery pending: new remote Required CI/native macOS/PR merge; NO PUSH / NO REMOTE CI / NO MERGE
```

`WIRED: NO`는 코드가 전혀 존재하지 않는다는 뜻이 아니라, 해당 work item의
required user/runtime path 전체가 acceptance 수준으로 연결되지 않았다는 뜻이다.
M12-002~004의 상세 MISSING은 해당 work item이 시작될 때 별도 baseline audit로
확정한다. M12-002~004 local acceptance는 닫혔으며 다음 M12-005는 NOT_STARTED/Ready: Yes다.

### M11 post-qualification release hardening

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M11-FIX-01 | release activation rollback/durability | COMPLETE |
| 2 | M11-FIX-02 | protected main·exact Required provenance | COMPLETE |
| 3 | M11-FIX-03 | candidate exact 10-file attestation | COMPLETE |
| 4 | M11-FIX-04 | draft asset binding·post-publish quarantine | COMPLETE |
| 5 | M11-FIX-05 | observer profile wait timing regression | COMPLETE |


### M12-002 Go resolver/source 보완 checkpoint (2026-10-04)

사용자 승인으로 신규 Go resolver만 연결별 200,000 charged records를 적용했다.
기존 profile/collector/CPU/memory/time 예산과 observer 판정은 유지한다. 정적 검사의
최초 원인은 의도적으로 invalid한 testdata assets에 일반 source 문법을 강제한
것이며 pinned Go discovery/explicit compiler·동일 최소 fixture로 대조했다.
Cobra/tools 전체 CLI, pflag 실제 Get+retained Download와 기존 npm/GitHub ELF/
필수 sdist/wheel 및 생산 C++ finite-boundary/latch 회귀는 통과했다.
[Source correction evidence](./evidence/m12-002-go-source-correction/README.md)가
exact source/입력/결과/검사 범위를 소유한다.

큰 gRPC graph는 source/inspection을 통과한 뒤 별도 cache file bound로 FAIL했다.
당시 전체 preflight 11,572 files/약190.5 MiB이며 file cap20,000 승인은 pending이었다.
아래 cache checkpoint가 이 상태를 supersede한다. M12-002 IN_PROGRESS와 전체 WIRED/QUALIFIED/ACCEPTANCE_CLOSED NO,
M12-003/004 NOT_STARTED를 유지한다. Observed offline build/output publish와 최종
qualification MISSING을 계속한다. NO PUSH / NO REMOTE CI / NO MERGE.

### M12-002 Go verified cache file budget checkpoint (2026-10-04)

사용자가 Go cache aggregate file cap20,000을 명시 승인했다. 기존 512 MiB와
module archive10,000 entries 및 receipt4 MiB를 유지하고 directory 포함 파생
filesystem entry bound만40,000으로 맞춘다. Observer/Policy/runtime/workflow와
기존 profile 예산은 이 변경의 입력이 아니다.
[Cache budget evidence](./evidence/m12-002-go-cache-budget/README.md)가 같은
aggregate fixture의 old source FAIL→new PASS, exact20,000 경계·초과 reuse 거부,
aggregate20,002 승격 차단과 임시 cache 정리를 소유한다. Maintained actual CLI의
dependency-free Download 및 pflag/gRPC41 Get+retained Download도 PASS다.

Observed offline Go build/output publication과 최종 local qualification은
MISSING이다. 전체 `WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`, M12-002
IN_PROGRESS와 M12-003/004 NOT_STARTED를 유지하고 같은 item의 남은 작업을 계속한다.
NO PUSH / NO REMOTE CI / NO MERGE.


### M12-002 Go build input preparation checkpoint (2026-10-04)

[Build input evidence](./evidence/m12-002-go-build-inputs/README.md)는 기존 미연결
Go build wrapper의 command/cancellation 경계 before FAIL→after PASS와 retained
approval·Evidence/cache 재검사, bounded anchored source snapshot의 정상/변조/링크
회귀를 연결한다. 기존 components를 확장하며 원본 project code를 Host에서 실행하지
않는다. Local source identity를 registry attestation이나 Policy ALLOW로 사용하지 않는다.
이 checkpoint는 build 입력 준비이며 registered ARTIFACT Go/compiler의 실제 isolated
execution·offline graph revalidation·build-time Evidence/Policy·bounded output publish·
CLI wiring과 최종 qualification은 MISSING이다. 전체 WIRED/QUALIFIED/ACCEPTANCE_CLOSED
NO, M12-002 IN_PROGRESS와 M12-003/004 NOT_STARTED를 유지하고 같은 item을 계속한다.
NO PUSH / NO REMOTE CI / NO MERGE.

### M12-002 observed Go build / output wiring 진행 (2026-10-05)

신규 Go build profile·offline compiler 관찰과 separate local-source verification,
기존 Run/Evidence/M3 Policy, retained guard/cache 재검증과 bounded output publish를
실제 `helox go build`에 연결했다. Canonical Go leaf가 예산·source/output 경계를
소유한다. Dependency-free 연속 2회·public pflag·gRPC 41 modules의 production CLI
ALLOW/publish와 explicit invalid testdata의 실제 compiler 실패 보존을 확인했다.
Output 수집 첫 실패는 Docker archive reader가 gVisor private scratch를 읽지 못한
것이며 exact named output data volume으로 같은 최소 source fixture가 통과했다.
입력 RO·event charging·기존 profile의 예산·Policy를 유지한다.

이 wiring checkpoint 당시 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO였다. 아래 최종 local acceptance 종료가 이 상태를 supersede한다.
MISSING은 새 library/cgo/security 및 wired CI 호출의 실제 로컬 통합, 변경 observer의
기존 직접 소비자(sdist 포함), 같은 최종 실행 후보 CPU/cu126 full과 cu130/cu132
focused 공통 경로, final canonical/security/docs/corpus/Darwin test compilation 및
qualification evidence다. 과거 M12-001 remote PASS를 새 후보 PASS로 전용하지 않는다.
M12-002 IN_PROGRESS, M12-003/004 NOT_STARTED를 유지하며 계속 진행한다.
NO PUSH / NO REMOTE CI / NO MERGE.


### M12-002 local acceptance 종료 (2026-10-05)

`IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES`,
`COMPLETE / MISSING: None`으로 닫는다. Linux/amd64의 실제 source4·Get/download3·
build CLI9 정상/부정, offline compiler coverage·Policy·output publication과
기존 직접 소비자8(필수 sdist 포함), 네 profile 추가sequence3, ordinary 실제
npm/PyPI install/promote 및 canonical 검사를 같은 최종 실행 후보에서 확인했다.
CPU full526.62s/cu126 full2082.74s와 viewer NOT_ATTESTED·검사 전용 NumPy 승격
제외, exact helper wait0·원래 설치 복원도 PASS다. Exact source/binary·입력·각 scope와
causal proofs는 [qualification evidence](./evidence/m12-002-go-build-qualification/result.json)가 소유한다.

cu130/cu132는 실제 공통 경로 focused 회귀만 실행했으며 새 후보 full PASS로
기록하지 않는다. Native macOS와 새 remote Required는 NOT_RUN이다. Darwin
TEST compile이나 M12-001 원격 green을 그 증거로 전용하지 않는다. 이는 사용자
승인된 로컬 work-item 완료와 후속 진행이며 원격 전달·전체 M12 qualification·
CUDA RELEASE_SUPPORTED 결정은 완료 선언에 포함하지 않는다. Prior full failure의
미확정 원인은 evidence의 범위를 유지한다.

종료 문서만의 delta는 실행 입력 동일성을 대조하고 전체 후보 docs/security를
검사한다. 다음 M12-003은 `NOT_STARTED / Ready: Yes`이며 baseline audit 뒤 하나의
IN_PROGRESS로 전환한다. M12-004는 아직 NOT_STARTED다.
NO PUSH / NO REMOTE CI / NO MERGE.


### M12-003 시작 baseline audit (2026-10-05)

Baseline은 M12-002 로컬 완료 `bdd3a29`이며 `IMPLEMENTED=YES / WIRED=NO /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`다. 기존 `artifact/cargo`의 exact reference·
metadata/graph, `sandbox/CargoResolver`, `application/CargoResolutionService`,
`promotion/cargo_transaction.go`, CLI/bootstrap을 재사용한다. `cargo add`는 현재
resolve만 수행하고 `cargo build`는 static unavailable leaf다. 기존 stub/unit PASS는
실제 acquisition·Policy·transaction·observed build의 qualification이 아니다.

`MISSING`: 실제 metadata와 frozen Cargo.lock·독립 sparse-index checksum의 연결,
complete graph/feature/target 상태; anchored controls와 local source 및 pinned Rust/Cargo
runtime을 사용하는 isolated resolver; canonical crate intake/검증/정적 검사/Run·
Evidence·Policy와 verified cache; guarded Cargo.toml/Cargo.lock transaction 및 재선택
금지·경쟁/rollback/cleanup; offline observed build.rs/proc-macro/native subprocess와
bounded no-overwrite output publish; canonical leaf의 실제 정상·부정/security
qualification과 영향받는 직접 소비자 및 canonical/platform 검사다.

Read: [Cargo canonical contract](./16-m12-01-ecosystem-expansion-contract.md#5-m12-003--rust--cargo),
[Adapter 책임](../architecture/02-adapters-and-providers.md),
[Inspection/Sandbox/Policy](../architecture/03-inspection-sandbox-policy.md), 관련 Artifact/
Inspection/Evidence·Promotion contracts와 Engineering check/security 규칙을 따른다.
기존 구현을 대체하지 않고 확인한 acceptance gap만 같은 owner에서 보완한다.
M12-003만 IN_PROGRESS이며 M12-004는 NOT_STARTED다. 새 profile의 예산은 Go 예산을
자동 전용하지 않고 실제 처리량과 실패 경계를 따로 확인한다.
NO PUSH / NO REMOTE CI / NO MERGE.

### M12-003 실제 parser/source acquisition 부분 checkpoint (2026-10-05)

동일 고정 Cargo metadata/lock의 old resolver FAIL→frozen lock과 연결한 bounded parser
PASS를 확인했다. Complete nodes·edges·workspace/features/target 상태를 정규화하며
checksum 선언과 공식 source 인증은 구분한다. 공식 Rust1.99.0-bookworm OCI digest를
runtime lock에 고정하고 실제 SDK version query와 substituted/missing runtime 부정
회귀를 확인했다. 등록된 gVisor Cargo 실행은 아직 NOT_RUN이다.

빈 owned intake의 `itoa@1.0.17`은 canonical HTTPS acquire→독립 sparse-index/lock/bytes
검증→bounded static archive 검사→Evidence→기존 Policy ALLOW가 실제 PASS다.
Parser/index/archive/intake/integrity/runtime 부정과 canonical quick/security/docs/
vulnerability도 PASS다. [부분 evidence](./evidence/m12-003-cargo-acquisition/result.json)가
source/입력/실제 실행 범위를 소유한다. Source 성공은 project add/build 완료가 아니다.

`WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`와 M12-003 IN_PROGRESS를 유지한다.
MISSING은 anchored complete project/source controls, isolated resolver, complete snapshot과
verified cache, guarded frozen add transaction/rollback, offline observed build와 bounded
no-overwrite output 및 whole qualification/직접 소비자·platform 검증이다. M12-004는
NOT_STARTED다. NO PUSH / NO REMOTE CI / NO MERGE.

### M12-003 isolated source·verified cache 부분 검증 (2026-10-05)

같은 최소 itoa1.0.17 frozen source fixture가 registered gVisor에서 source PASS3.38s다.
판정 불변 kernel image/role/open/topology 진단으로 각 최초 차단을 연결했고, exact
Cargo CONTROL의 고정 runtime 입력과 SDK info-only child만 narrow하게 교정했다.
Mutable guest SocketPair return buffer는 FD authority로 쓰지 않는다. 실제 kernel
NewFDs 결과를 전용 typed point로 전달하고 exact entry/result·thread/start/args에
대조한 producer/consumer의 before FAIL→after latches 및 kernel tests가 PASS다.
Canonical six-member runtime build와 source/patch roundtrip을 확인했다. 기존 profile
예산·Policy는 유지한다. 새 common runtime의 전체 직접 소비자/qualification은 별도다.

공식 itoa의 빈 intake acquire→독립 sparse-index/lock/bytes 검증→static inspection→
Evidence2·Policy ALLOW→verified vendor cache 재사용도 실제 PASS0.213s다. 공통
Evidence/inventory helper의 기존 Go 소비자와 Cargo cache 정상/부정·initial 한도
경계는 PASS다. Cache 한도·reserved checksum 권한은 Cargo canonical leaf가 소유한다.
기존 Cargo control plan의 alias·oversize·동일 바이트 inode 교체 네 누락은 같은
final fixture의 old1270 FAIL→anchored bounded single-link/identity 교정 PASS다.
[부분 source/cache evidence](./evidence/m12-003-cargo-source-cache/result.json)가 source·runtime·원본 hash와
각 실행 범위를 소유한다. 같은 후보의 Cargo source4.10s·Go source6.54s·필수 sdist
15.94s·wheel dynamic8.55s 및 canonical quick/security/docs가 PASS다. 이를 whole
CLI add/build 또는 새 CPU/CUDA qualification으로 전용하지 않는다.

`IMPLEMENTED=YES / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`를 유지한다.
MISSING은 independent retained approval·original source/control guard와 guarded frozen
add publication/rollback, actual CLI wiring, offline observed build.rs/proc-macro/native
helper 및 bounded no-overwrite output, 이후 최종 후보의 실제 직접 소비자(필수
sdist 포함)와 CPU/cu126 full·cu130/cu132 focused, canonical/platform/
security/CI wiring 및 whole acceptance evidence다. Native macOS/new remote Required는
NOT_RUN이며 M12-003만 IN_PROGRESS, M12-004는 NOT_STARTED다.
NO PUSH / NO REMOTE CI / NO MERGE.

### M12-003 guarded Cargo add 부분 checkpoint (2026-10-05)

실제 최초 추가와 retained reuse가 itoa1 crate21.87s·serde7 crates25.15s에서 PASS다.
Complete graph의 acquire/독립 검증/검사·Evidence·Policy·vendor cache와 frozen control
publication을 연결했으며 공통 application workflow의 Go get 직접 소비자30.86s도
PASS다. Matching marker의 old internal primitive와 Cargo direct VM metadata read의
same-final before FAIL→after PASS, source/control/state/cache/Evidence·경쟁 publication·
rollback/uncertainty·foreign lock/recovery journal 부정과 shared Go/Cargo boundary matrix,
canonical quick/security/docs/Linux platform·Darwin TEST compile을 확인했다.
[부분 add evidence](./evidence/m12-003-cargo-add/result.json)가 각 실행과 source/test-only
후속 delta를 소유한다. Native macOS/new remote Required는 NOT_RUN이다.

M12-003만 IN_PROGRESS이며 `IMPLEMENTED=YES / WIRED=NO / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO`를 유지한다. Whole WIRED/qualification MISSING은 offline observed
Cargo build.rs/proc-macro/native helper·bounded no-overwrite output, final candidate의
필수 sdist 포함 직접 소비자·CPU/cu126 full·cu130/cu132 focused, CI wiring과 whole
acceptance evidence다. M12-004는 NOT_STARTED다. NO PUSH / NO REMOTE CI / NO MERGE.

### M12-003 observed build와 중단 재개 qualification (2026-10-06)

기존 Cargo/공통 project build owner에 실제 offline build와 bounded no-overwrite
publication이 연결됐다. 같은 실행 후보의 Cargo 정상·보안·retained13개, Go/Cargo
관련14개, 필수 sdist와 기존 직접 소비자 및 CPU full518.36s가 PASS다. Actual
source707·binaries·원본과 각 범위는
[build qualification evidence](./evidence/m12-003-cargo-build-qualification/result.json)가 소유한다.
`IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`다.

cu126 첫 회계 FAIL, WSL 중단 retry, native 재개40m timeout과 D-backed loop의
40m timeout은 각각 보존한다. 마지막 실행은 publish-intent journal fsync에서
중단됐으며 helper parent wait0·bounded Docker absence·original12 복원은 확인했다.
Incomplete project journal/lock/private workspace는 그대로 보존한다. Post-timeout
동기화200회는 native0.548s/loop4.385s로 측정됐지만 원래 aggregate 지연 인과는
미확정이다. 제품의 durable intent/해시/rollback/40m 상한을 바꾸지 않는다.

현재 MISSING은 같은 제품 후보의 최종 cu126 full과 전체 후보 종료 검사·evidence다.
Qualification harness만 별도 fresh cache root를 선택할 수 있도록 보완한다. 캐시는
D-backed ext4에, private workspace/target/journal은 같은 native filesystem에 둔다.
기존 verified copy가 두 filesystem을 잇고 publication rename은 native 내부에 남는다.
원래 observed provider/runtime/Profile·Policy·모든 assertion은 유지한다. 해당 환경의
실제 full 결과 전에는 성공으로 표시하지 않는다. M12-003만 IN_PROGRESS이며
M12-004는 NOT_STARTED다. NO PUSH / NO REMOTE CI / NO MERGE.

### M12-003 압축 후 qualification 재개 (2026-10-06)

후보708·보존 바이너리3·original runtime12·gVisor bundle의 동일성을 확인하고
사라진 native projection·실행 파일·canonical `/run`을 복구했다. F의 동일52GiB
ext4는 보관 증거 확인용 읽기 전용 mount로 연결했다. VHDX 길이 감소와 C의
실제 여유90GB 이상은 확인됐으나 압축 worker의 최종 종료 receipt는 미확인이다.
새 여유 공간에서 같은 실행 후보의 fresh native cache/target/private/journal로
cu126 full을 기존40분 상한으로 실행 중이다. 이전 full 실패를 보존하고 제품의
durability·Policy·관찰·자원 한도는 변경하지 않았다. 결과·parent-owned cleanup·
original12 복원과 최종 전체 후보 검사 전에는 QUALIFIED/ACCEPTANCE_CLOSED를
NO로 유지한다. M12-003 하나만 IN_PROGRESS, M12-004는 NOT_STARTED다.
NO PUSH / NO REMOTE CI / NO MERGE.

### M12-003 재개 full 통과와 종료 검사 (2026-10-06)

같은708-path/c88c test 후보의 fresh native cu126 full은2070.42s/actual0 PASS다.
16,117개 파일 게시, 검사 전용 NumPy의29개 승격집합 제외와 viewer NOT_ATTESTED의
결과·승격 Evidence 참조, helper parentwait0/bounded Docker absence/original12 복원을
확인했다. 이전 timeout·중단·capacity 실패는 별도 증거로 보존한다. 과거 지연의
정확한 원인을 새 PASS만으로 확정하지 않는다.

[현재 full 결과](./evidence/m12-003-cargo-build-qualification/reclaimed-native-cu126-full-result.json)와
[CI aggregate 시간 검토](./evidence/m12-003-cargo-build-qualification/ci-duration-review.json)가
실제 입력·소요 시간과 individual test/보안 한도 보존을 소유한다. 전체 후보 종료
검사·evidence가 남아 있어 M12-003 IN_PROGRESS/acceptance NO, M12-004 NOT_STARTED다.
NO PUSH / NO REMOTE CI / NO MERGE.

### M12-004 frozen public source·signature 부분 checkpoint (2026-10-06)

기존 metadata/resolver owner에 실제 schema·discovery·요청 version/platform/protocol
binding을 보완했고, 신규 acquired subject는 원래 registry bytes·signed checksum·
signature·ZIP을 Run에 고정한다. Acquire는 재선택하지 않는다. 독립 verifier는 full
signer/current key·partner endorsement와 exact archive checksum을 검증한다.
Actual official random3.7.2와 partner integrations/github6.6.0 source gate2.686s는
VERIFIED이며 frozen selection·mirror/redirect·intake alias·signature·expiry/revocation
부정 및 canonical quick/security/vulnerability도 exit0다.

Whole item의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED는 계속 NO다.
현재 init은 아직 resolution 출력만 연결되어 있다. MISSING은 bounded ZIP/executable/
h1 검사·실제 fixed dynamic probe, complete project/lock selection과 독립 승인 staging,
original guard 아래 provider-tree·lock·metadata 전체 publish/rollback 및 실제 CLI·
보안·경쟁·partial commit qualification이다. [부분 source evidence](./evidence/m12-004-provider-source/result.json)는
원래 FAIL과 source-bound PASS를 구분한다. 같은 M12-004를 계속하며 원격 작업은 없다.

### M12-004 시작 baseline audit (2026-10-06)

기준 commit은 `8c56f873eb7ce395af4ba984190413492b096e63`이다. 기존
`artifact/terraformprovider`의 exact reference·metadata parser·resolver·graph,
`application/terraform.go`와 CLI/bootstrap의 조회 연결,
`promotion/terraform_transaction.go`의 lock plan·private workspace·backup/rollback
primitive를 재사용한다. Full item은 `IMPLEMENTED=NO / WIRED=NO /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`다. 기존 조회 성공을 설치 완료로 쓰지 않는다.

MISSING은 실제 discovery/version/package 응답과 endpoint binding, frozen signed
acquisition과 full signer/trust 확인, ZIP/executable 정적 검사 및 fixed dynamic
probe, 독립 Evidence/Policy 승인, original guard 아래 provider tree·lock·metadata
전체 publish/rollback, retained state 검증, public official/partner 및 보안·경쟁·
부분 commit qualification이다. 관련 owner의 baseline hash와 재사용 경계는
[baseline evidence](./evidence/m12-004-provider-baseline/result.json)에 기록했다.
M12-004만 IN_PROGRESS이며 NO PUSH / NO REMOTE CI / NO MERGE를 유지한다.

### M12-003 local acceptance 종료 (2026-10-06)

`IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES`,
`COMPLETE / MISSING: None`으로 로컬 acceptance를 닫는다. Actual Cargo13·Go/Cargo14,
mandatory sdist 포함 기존 직접 소비자·ordinary CLI/promotion, 같은708 후보의
CPU523.65s/cu1262070.42s full과 terminal cleanup/restoration을 확인했다.
최종875-path candidate의 workflow/security/docs·Darwin TEST compile도 PASS다.
[Qualification 결과](./evidence/m12-003-cargo-build-qualification/result.json)가
원본·same-source/runtime·각 검증 범위와 과거 실패 보존을 연결한다.

최종 종료 문서는 실행 source708 대비 허용된6개 documentation/workflow/format-policy
경로만 바꾸며 전체 docs/security/whitespace를 확인해 로컬 checkpoint를 남긴다.
Native macOS·새 remote Required는 NOT_RUN이고 CUDA release-support matrix 결정은
M12-005에 남는다. 다음 M12-004는 NOT_STARTED/Ready: Yes이며 baseline audit 후
유일한 IN_PROGRESS로 시작한다. NO PUSH / NO REMOTE CI / NO MERGE.

### M12-004 local acceptance 종료 (2026-10-07)

`IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES`,
`COMPLETE / MISSING: None`으로 로컬 acceptance를 닫는다. 기존 resolver, generic
project update, GitHub ELF lifecycle과 Terraform transaction primitive를 재사용했다.
Public official random3.7.2/partner integrations/github6.6.0의 signed source,
bounded ZIP/ELF·HCL controls, fixed help0/1 observation, 독립 Evidence/entry+set ALLOW,
private verified cache와 guarded provider tree/lock/metadata/retained approval을 연결한다.
Actual 최초·retained 여섯 calls와 checksum/remote module/foreign cache/concurrent
네 product negatives, hostile ownership/mode/post-state/rollback 회귀를 확인했다.

최종 exact943-source timing 후보의 CPU507.82s/cu1261921.07s full, public probe/security,
기존 직접 소비자12·네 proc maps·실제 npm/pip·Go/Cargo14·Cargo13·promotion2는 모두
PASS다. 각 helper wait0, 독립 bounded Docker absence와 original12 bytes/modes 복원을
확인했다. Viewer NOT_ATTESTED와 검사 전용 NumPy 승격 제외는 유지한다. Go build9
634.10s/get-download3 242.79s/Cargo13 302.60s는 기존 CI 상한 안이다.

[Qualification 결과](./evidence/m12-004-provider-qualification/result.json)가 raw,
세 후보의 변경 경계, full input/target·stored Evidence와 actual CI budget을 연결한다.
첫 cu126815.92s late gap의 원인은 미확정이며 후속 PASS/오류 진단을 회계 수정으로
쓰지 않는다. Deliberate preparation98·진단 SourceID fixture·가상 mutex fixture의
실패도 보존했다. 전체1198-path 후보 docs/workflow/security와 staged 검사를 확인하고
최종 종료 변경은 명시적인 documentation/evidence/format-policy 경로만 바꾼다.

Scope는 Provider installation only이며 RPC/plan/apply/cloud behavior는 NOT_ATTESTED다.
Native macOS/new remote Required는 NOT_RUN, cu130/cu132는 공통 경로 focused scope다.
M12 전체는 IN_PROGRESS이며 broader release support/feature freeze는 다음 M12-005에
남는다. 다음 item은 `NOT_STARTED / Ready: Yes`이고 구현을 시작하지 않았다.
NO PUSH / NO REMOTE CI / NO MERGE / NO AGENTS.

### M12-002~004 원격 전달·CI 후속 시작 (2026-10-07)

사용자가 누적 변경의 push, PR CI, 원인에 따른 수정·재검증·오답노트 기록을
all green까지 반복하고 불필요한 성공 테스트 아티팩트를 정리하도록 요청했다.
기존 local acceptance와 실패 원본은 보존하며 현재 활성 item은 M12-004 후속 하나다.
M12-005는 NOT_STARTED/Ready: Yes이고 이 원격 후속 완료 뒤 baseline audit로 재개한다.

Baseline은 local6da451a와 remote main62c3b59다. Main 이후16 commits에
M12-002 bdd3a29, M12-003 8c56f87, M12-004 6da451a가 모두 포함된다.
IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES(local), remote qualification=NO,
ACCEPTANCE_CLOSED=YES(local), remote delivery closed=NO다. 기존 구현을 다시 만들지 않는다.
MISSING은 같은 최종 후보의 PR Required 및9 prerequisites 실제 성공, 필요한 오류
교정·회귀, raw run/job/source 연결, 완료 기록과 성공 테스트 전용 데이터 정리다.

이번 요청은 push와 PR CI를 허용한다. Main merge, M12-005 구현이나 support 범위
확장은 포함하지 않는다. 과거 NO PUSH/NO REMOTE CI는 당시 local 작업 범위 기록이다.
실패는 최초 원인과 정확한 입력을 보존하며 M12-001 green code와 비교해 확인된
회귀만 수정한다. Skip/Required 삭제/예산 완화로 성공을 만들지 않는다.
성공한 테스트 데이터는 보존 증거와 후속 사용 여부·현재 process/resource를 확인한
뒤 본 작업의 전용 경로만 제거한다. 실패 재현 입력·runtime originals는 유지한다.

### 원격 첫 macOS 실패와 동일 입력 교정 (2026-10-07)

PR29/run37497646753의 macOS job112386202588에서
TestPyTorchQualificationCacheIsFreshAndSeparate가 no-symlink parent 조건으로 실패했다.
실제 checkout은 PR merge0ec1e77이며 branch candidate1170ce2다. M12-001 main62c3b59에는
이 test/helper 추가가 없고 M12-003 8c56f87에서 추가됐다. Linux TMPDIR를 symlink alias로
고정한 동일 두 test도 beforeFAIL0.016s로 같은 오류를 재현했다. 정상 test의 임시
parent와 negative test의 기준 root만 EvalSymlinks로 고정한다. Helper의 symlink·mode
거부와 freshness 계약은 변경하지 않으며 negative를 다른 symlink 오류로 가리지 않는다.
Native macOS 재검증과 최종 all green은 아직 남아 있고 같은 후속 item을 계속한다.

### 원격 Cargo fixture 작업 디렉터리 교정 (2026-10-07)

c106172/run37499465550은 native macOS 포함8 prerequisites가 SUCCESS이고
Linux112393261447/Required112404529823은 FAIL이다. 최초 오류는 Cargo source
integration의 상대 fixture 경로 ENOENT다. Fixture는 tracked blob1eb7451이며
누락된 것이 아니다. 직접 설치한 test binary는 repository cwd를 유지하지만
일반 go test는 package cwd에서 실행한다. 같은 compiled metadata consumer는
repository cwd에서 FAIL, internal/sandbox cwd에서 PASS0.02s다. 기존 lifecycle
step의 Cargo source 명령만 package cwd의 subshell에서 실행하며 source/argv,
helper 신원·복원, Policy와 모든 예산을 유지한다. 원래 failed job의 helper wait0와
RESTORED는 제품 성공과 구분한다. Full 후속 CI와 Required는 아직 남아 있다.

### M12-002~004 원격 qualification과 성공 데이터 정리 완료 (2026-10-07)

PR29의 qualified head132baea, checkoutc375a45/run37503579865에서 Required와
9 prerequisites 모두 실제 SUCCESS다. 두 source tree의 전체 tracked blob/mode는
동일하다. Native macOS 재검증, 실제 Cargo source1.30s, Go download/build/Cgo와
Cargo add/build/security, Terraform probe/security/init 및 부정 검증, CPU full288.73s,
기존 PyPI sdist8.76s/wheel4.59s와 npm/PyPI promotion까지 실제 PASS를 확인했다.
CI production client를 명시적으로 다시 설치하고 helper parent wait0를 확인했다.
최종 cleanup은 original0/CONFIRMED_STOPPED/client_error0/client_restore0/NOT_REQUIRED다.
과거 최초 실패·cancelled와 이번 완료 결과를 구분하며30 job raw 본문/SHA를 보존한다.

IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES다.
MISSING=[]이며 source qualification과 데이터 정리의
[종료 근거](./evidence/m12-002-004-remote-ci/result.json)를 연결한다.
완료된 테스트 전용 네 root는 전체 파일 SHA/metadata·원본 Evidence와 현재 사용 여부를
확인한 뒤 삭제했고, 두 batch의 WSL 가용 공간 증가 합계는30,309,568,512B(약28.2GiB)다.
실패 cu126 재현 입력과 runtime originals는 유지하며 Windows VHDX 축소로 확대하지 않는다.

이 종료 변경은 문서·증거·raw 형식 예외만 포함하며 qualified source의 나머지
tracked blob/mode 전체를 대조한다. 새 종료 HEAD의 원격 Required는 push 뒤 별도로
확인한다. M12-004는 COMPLETE, active item은 없고 다음 M12-005는
NOT_STARTED/Ready: Yes다. M12 전체와 broader release/feature freeze는 아직 완료가 아니다.

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

### M12-005 baseline audit 시작 (2026-10-07)

현재01175bb와 실제 remote CI run37510273661 completed/success를 확인했다.
[Baseline audit](./evidence/m12-005-qualification/README.md)의 기존 구현을 재사용하고
MISSING qualification·지원 matrix·문서 일치성·구조 검토·성공 데이터 정리만 수행한다.
M12-005는 `IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`다.
사용자는 commit/push/CI 오류 수정과 불필요한 성공 테스트 데이터 정리를 포함한005
완주를 승인했다. 현재 하나의IN_PROGRESS를 유지하며 M12-02는RESERVED다.

### M12-005 추가 미사용 API 정리 checkpoint (2026-10-07)

AST identifier triage와 실제 bootstrap/factory·tracked reference를 대조하여
production/test consumer가 없는 초기 Terraform resolution service와 registry
getter, 옛 string-only PyPI aggregate, executor 없는 Python probe wrapper를
제거했다. 기존 guarded init·extras를 유지하는 typed aggregate·endpoint constant·
configured executor probe는 그대로다. [추가 source boundary](./evidence/m12-005-qualification/additional-code-cleanup-boundary.json)가
40-line scope를 소유한다. Canonical quick/build/architecture/vet/Staticcheck/default,
docs/security는 actual0다. 이는 완료된 subsystem의 대체 구현이 아니다.

앞선005 cu13037555305508/bb28b47와 ordinary37556677169/8791fdc의 실제 all10
SUCCESS, full 및 새 npm inspect 결과는 당시 source evidence로 보존한다. 추가 정리
전 cu13237560044655는 obsolete source를 실행하지 않도록 명시적으로 취소했으며
제품 failure나 qualification PASS로 해석하지 않는다. 새 source의 CPU·세CUDA와
Required를 독립 실행한 후에만005 acceptance/feature freeze를 닫는다. 현재005
IN_PROGRESS와 M12-02RESERVED/ReadyNo를 유지한다.

### M12-005 qualification·feature freeze 완료 (2026-10-07)

현재 source `d328756`의 네 profile에서 Required와9 prerequisites를 각각 실제
SUCCESS로 확인했다. CPU·cu126·cu130·cu132 full, 실제 npm/PyPI/GitHub/Go/Cargo/
Provider consumer,102 observer latches와 helper 종료·복원은 각각 원본 증거와
소스/checkout tree·mode에 대응한다. [Qualification result](./evidence/m12-005-qualification/result.json)가
기본763/2217 회귀,77개 요구사항,지원 tuple,CLI/문서·구조 검토와 정리 범위를 연결한다.

M12-005: `IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES`.
MISSING: 없음. 기존 구현을 재사용했고 미사용 초기 API201 net lines와 직접 재현한
npm media-type 결함만 기존 owner에서 정리했다. 성공 데이터5 roots를 삭제하고
Evidence282개를 해시 보존하여 약21.97GiB의 WSL 가용 공간을 확보했다. 실패 데이터와
기존 runtime/shared cache는 보존한다. 과거 미확정 CUDA 지연의 해결이나 Windows
VHDX 압축으로 기록하지 않는다.

네 torch2.14.0 profile의 명시적 bounded support만 확정하며 GPU 계산/driver/
Toolkit, 다른 ABI/OS/version 또는 torchvision/torchaudio를 검증했다고 주장하지
않는다. CUDA inspection-only NumPy와 viewer NOT_ATTESTED 제한을 유지한다.
Final documentation/evidence commit의 원격 Required와 whole checkout 동일성은
별도 전달 확인이며 실제 완료 결과는 PR29 및 외부 closure receipt에 기록한다.

Active item은 없고 M12-02는 `NOT_STARTED / Ready: Yes`다. 이번 변경은 red-team을
수행하거나 NO_RELEASE_BLOCKING_FINDINGS를 선언하지 않는다. M12 전체는
IN_PROGRESS이며 main reviewer merge와 public release는 후속 gate를 따른다.

### M12-02 final red-team baseline audit 시작 (2026-10-07)

사용자는005 완료 후보19411ca에서 최종 검토·확정 결함 수정·회귀와 최종 CI까지
진행하도록 승인했다. [Baseline audit](./evidence/m12-02-red-team/README.md)에
IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO와
여덟 검토 영역·실제 MISSING을 기록했다. 기존 구현과005 full 증거를 재사용하며
새 기능이나 Core/Policy 계약 변경은 하지 않는다. M12-02만 IN_PROGRESS다.
M13은NOT_STARTED이며 이번 범위는 reviewer의 main merge 전 검토 완료까지다.

### M12-02 검토·remediation 완료 (2026-10-07)

기준19411ca의 기존 구현을 재사용해 여덟 책임 영역을 검토했다. Go/Cargo와 기존
npm/PyPI backup의 외부 데이터 삭제 및 npm 프로젝트 selected control의 EEXIST를
동일 fixture로 재현했고 기존 owner에서 FIX-01~03으로 수정했다. 처음 FIX-01의
안전한 원본 복원 차단은 기존 회귀가 반증해 보완했다. 새 Core/Policy/runtime 역할,
허용 목록이나 예산 변경은 없다. [최종 결과](./evidence/m12-02-red-team/result.json)가
local before/after와 같은 기능 source68099a5의 네 원격 실행·원문 hash를 연결한다.

M12-02: `IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES`.
MISSING: 없음(이 작업의 기술 acceptance). CPU·cu126·cu130·cu132 각10SUCCESS,
실제45/46소비자·npm first/retained2회·102latches·5helperwait0와 최종 cleanup을
검증했다. 기존 bounded 지원 tuple과 viewer NOT_ATTESTED·NumPy 승격 제외를 유지한다.
작성자 검토를 외부 독립 감사나 reviewer 승인으로 표시하지 않는다.

이 완료 변경은 문서·증거만이며 qualified source와 실행 입력 bytes/mode의 일치를
[source boundary](./evidence/m12-02-red-team/source-boundary.json)에 기록한다.
문서 HEAD의 원격 Required 확인은 PR29와 외부 closure receipt가 소유한다.
Reviewer merge·main CI가 전달에 남고, M13은 NOT_STARTED다. M13-001의 기존
repository activation blocker도 해당 항목의 원래 해제 조건을 유지한다.

2026-10-20 actual signed lifecycle: rc1/rc2 official builds and independent13-file
verification passed. Disposable systemd base register/ordinary doctor/npm passed.
Actual update retained rc1 with MANUAL_REVIEW/NOT_PERFORMED; native ELF runtime
read classification gaps precede activation. The source audit/edit map/negative
matrix and second review are in [native ELF plan](./evidence/m13-004-convenience/native-elf-compatibility-plan-20261020.md).
No public Release, main merge, tag retarget or Host migration occurred.

2026-10-20 native ELF compatibility closure (local): the two actual faults and
additional sandbox OS marker are bounded in the existing classifier. Two RED
checks, final103 canonical C++ latches, final Quick/security/docs, actual signed
native4 independent required3/ALLOW, unchanged nominal npm and unmanaged update
denial passed. [Signed/native qualification](./evidence/m13-004-convenience/signed-native-qualification-20261020.md)
keeps original full update NOT_PERFORMED and the remaining fresh-signed
activation/health/crash/reboot/authentication scope explicit. PR38 remains draft.

2026-10-20 final source4ff6c89/[CI356](https://github.com/rahoney/heliopause/actions/runs/39416778306):
Required+9 actual SUCCESS; full checkout tree/mode match, native macOS/C++103,
Linux45 consumers/CPU293.82s, helper5 wait0/final confirmed cleanup verified in
[remote receipt](./evidence/m13-004-convenience/signed-native-20261020/remote-ci.json).
Completed trial containers/volume/images/payload/build/operator cleanup and Host
preservation are in [cleanup receipt](./evidence/m13-004-convenience/signed-native-20261020/cleanup.json).
[New signing proposal](./evidence/m13-004-convenience/signed-retest-plan-20261020.md)
freezes rc3/rc4 to4ff6c89. The owner subsequently approved both trial tags;
rc3 now points to that exact source and [official build39439522714](https://github.com/rahoney/heliopause/actions/runs/39439522714)
is SUCCESS with actual base/register/doctor/npm PASS. rc4 was then created at
the same source; [official build39442575693](https://github.com/rahoney/heliopause/actions/runs/39442575693) is SUCCESS. M13-004
remains IN_PROGRESS; signed whole activation/recovery and channel names remain open.

2026-10-21 health callback closure (source7fedaf3): actual rc3→rc4 reached
required18/ALLOW but rolled back; old doctor/npm and actual OS-auth denial passed.
The public CLI flag mismatch was corrected after documented review and RED4→
GREEN semantic7. Local format/Quick/security/vulnerability/docs/related Race and
[CI357](./evidence/m13-004-convenience/signed-native-20261020/health-cli-remote-ci.json)
Required+9 actual SUCCESS passed; exact whole checkout tree/mode, native macOS,
C++103/Linux45·CPU226.62s/helper5 wait0/final confirmed cleanup were verified.
Completed downloads26/463,988,221B, rc4 guest/transport and one-shot operators
were removed; qualified rc3 baseline was restored/checked and idle-stopped.
Host originals/shared runtime remain unchanged. [Next exact target proposal](./evidence/m13-004-convenience/signed-target-plan-20261021.md)
freezes only rc5 to7fedaf3 and awaits owner approval; no new ref yet. Whole
successful activation/auth-positive/crash/reboot and channel names remain open.

2026-10-21 approved rc5 execution supersedes that proposed-target state:
[official build39456168381](https://github.com/rahoney/heliopause/actions/runs/39456168381)
and exact13 independent offline/canonical checks passed. Actual normal rc3→rc5,
OS password authentication and concurrent second-call refusal reached COMMITTED;
candidate service SIGKILL rolled back. Root installer SIGKILL preserved VERIFYING
across guest PID1 cold start; installed recover rolled back and re-execution committed.
All corresponding ordinary doctor5/npm3 checks passed. Host kernel reboot/public
Release were not performed. [Receipt](./evidence/m13-004-convenience/signed-native-20261020/rc5-qualification.json)
also preserves the cancellation failure: old version kept but stale observer lock
prevented the next inspection. [Reviewed correction](./evidence/m13-004-convenience/observer-cancellation-plan-20261021.md)
targets the existing launcher's operation-context auto-kill, with actual child
RED2→GREEN4. Corrected-source remote CI and actual signed caller cancellation remain
open; M13-004 stays the sole IN_PROGRESS item. Completed rc5 downloads13/231,996,591B
were deleted; only the needed qualified rc3 base is retained idle-stopped.

2026-10-21 observer owner source closure: source4a8c278 has [CI358 receipt](./evidence/m13-004-convenience/signed-native-20261020/observer-remote-ci.json)
Required+9 actual SUCCESS, all10 checkoutdacc156/tree6a81838 equality, native macOS/
C++103/Linux45 actual PASS·CPU291.99s/helper5 wait0/final confirmed cleanup.
Canonical local source checks and related owner Race passed; expanded Sandbox Race7
preexisting failures remain separately recorded. Corrected actual signed caller is
still MISSING. [One minimal rc6 input](./evidence/m13-004-convenience/signed-caller-plan-20261021.md)
is frozen to source4a8c278; the owner approved this one ref and the reviewed tests
on2026-10-21 with “이어가”. Execution is IN_PROGRESS; no creation success is assumed. PR38 stays
draft. Completed rc5 payloads and operators are removed; only the necessary qualified
rc3 baseline is retained idle. M13-004 alone remains IN_PROGRESS.

2026-10-21 corrected signed caller closure: approved source4a8c278/tag rc6
[official build39519718710](https://github.com/rahoney/heliopause/actions/runs/39519718710)
and exact13 independent UID1000 offline/canonical verification passed.
[Receipt](./evidence/m13-004-convenience/signed-native-20261020/rc6-qualification.json)
connects actual rc3→rc6 required18/ALLOW→COMMITTED35.18s, corrected installed rc6
acquisition/native dynamic handoff cancellation2.00s/6.25s with original failure
preserved, unchanged current/registration/history/config and confirmed drain,
next doctor5/npm3 ALLOW, then same actual signed set fresh whole replay30.90s
required18/ALLOW→COMMITTED. Initial watcher runtime expectation failure is
retained separately; no SIGINT was sent then and it is not cancellation PASS.
Native same-runtime scope: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES /
ACCEPTANCE_CLOSED=YES / MISSING=none. Historical rc5 authentication/concurrency/
service rollback/Root crash/guest cold recovery stays within its recorded scope.
Completed download13, independent verifier and owned guest/image2/inner volume
are removed; Host10/runtime6/shared images4 remain unchanged. Final documentation
operators/cache are retired after their last consumer. No public Release, main
merge, permanent Host migration, kernel reboot or new CUDA full was performed.
M13-004 alone remains IN_PROGRESS for owner names/tap/formula and publisher
bindings, bounded wrappers and channel qualification; M13-005 is NOT_STARTED.

### 2026-10-25 P2 CI405 최초 오류와 B4d 로컬 교정

[실패 원본과 교정 범위](./evidence/package-client-quarantine/p2c5b4d-pnpm-ci-fixture-plan-20261025.md)의
CI405/head7e·checkout5e2b42a는 같은 전체tree d864b870이다. Required/Linux FAIL·나머지8SUCCESS며
startup4/public project6 뒤 새 compiled controlled fixture의 repository cwd ENOENT다. Native
product/메모리 실패로 합치지 않는다. Helperwait0/CONFIRMED_STOPPED·original1/RESTORED다.
동일 constructor의 data consumer에서 rootFAIL→packagePASS를 관측하고 해당 호출 하나만
package subshell로 고쳤다. Canonical5/Bash와 actual134.277s의 controlled3·public8/실제 검역4
통과, Host 전후 동일/정리를 확인했다. B5a8a4는 이 실패 원격 source 이후이며 새 원격과 Host
transaction/CLI·whole P2 acceptance는 열려 있다. 작은 수정별 재push 대신 이 단위를 묶어 전달한다.

### 2026-10-25 P2 B5b1 actual-operation records local checkpoint

[수정 지도·실제 결과](./evidence/package-client-quarantine/p2c5b5b1-pnpm-project-records-plan-20261025.md)에
기존 Generator를 사용한 Manifest/SBOM과 손실 없는 native roots/packages/peer instances를 연결했다.
실제 operation을 현재 ProjectInspectService와 같은 값으로 결합하고 schema2 receipt·현재 기록된
Evidence/Policy·records 전체 bytes/mode/links를 Stage/Open에서 재검증한다. Canonical4/관련 세
package 전체 Race216.546s, 실제 public peer4/empty records44.668s와 Host 불변/정리는 PASS다.
첫 realcorpus 변수명 충돌 compile 실패는 test-only로 교정·보존했다. CI406은 앞 fd3506b이고
Host extraction/independent retained state/transaction/adoption/CLI·whole P2 acceptance는 미완료다.

### 2026-10-25 P2 B5b2 private Host materialization local checkpoint

[수정 지도·실제 결과](./evidence/package-client-quarantine/p2c5b5b2-pnpm-private-materialization-plan-20261025.md)에서
sealed cache의 exact bytes/records를 private sibling으로 반출했다. Native hardlinks는 독립
single-link copy이고 내부 symlink만 유지하며 전체 readback과 known-only cleanup을 확인했다.
초기 inventory 뒤 bytes 변경 삭제 gap은 직접 RED5.485s→GREEN5.431s로 교정했다. Canonical4/
최종 production 관련 세 package 전체 Race193.207s, final fixture focused Race10.693s·Darwin
arm64 compile24.130s, public peer4/empty의 실제 private Prepare/Verify/Close45.074s·Host 불변/
helperwait0는 PASS다. 사용자 target은 부재이고 target commit/retained approval/adoption/CLI는
미완료다. CI406은 앞 fd3506b에만 적용한다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-25 CI406 named remote checkpoint 확인

실제55분 타이머 만료 뒤 [CI406 qualification](./evidence/package-client-quarantine/p2c5b-ci406-qualification-20261025.json)을
확인했다. Required 포함10SUCCESS이고 모든 job checkoutaf56a507과 sourcefd3506b의 전체tree
6307df7976c427324bca0a053234db6718339ee7/2,241files가 같다. 실제54m32다. 새 PNPM startup4/
public project6·controlled3와 public native8/실제 Policy4·8entries, Linux top-level60/CPU290.84s,
두 job observer112씩·native macOS Go1.27.3 build/default와 helper5wait0/final confirmed cleanup
original0/client_error0/client_restore0를 확인했다. CI405 cwd 실패의 원격 소비자 gap을 닫았다.
B5a source/default checks는 포함하지만 gated actual cache 소비자는 이 CI에서 실행하지 않았다.
후속 local aa204f2/bcc316b와 target transaction/CLI·whole P2는 이 성공 범위 밖이다. 현재는
[B5b3 기존 transaction 착수 감사](./evidence/package-client-quarantine/p2c5b5b3-pnpm-transaction-audit-20261025.md)의
두 번째 검토와 edit map을 확정하고 기존 engine의 고정 PNPM profile·독립 retained approval,
request-local application 연결을 구현 중이다. 첫 engine 범위 Quick162.799s는 PASS지만
bootstrap 소비자 포함 첫 검사는 U1000으로 FAIL이며 actual은 미실행이다. Canonical/변조·rollback
회귀와 실제 public 단일/peer/empty·fresh repeat/named update 및 기존 npm 소비자를 확인한다.
Native adoption/CLI·whole P2 acceptance는 미완료이며 M13-004 하나만 IN_PROGRESS다.

### 2026-10-25 P2 B5b3 guarded project transaction local checkpoint

[B5b3 실제 결과](./evidence/package-client-quarantine/p2c5b5b3-pnpm-project-transaction-local-20261025.json)는
기존 npm engine의 고정 PNPM profile·선택 전 원본 guard·독립 retained approval과 actual request-local
Application 소비자를 연결한다. Canonical4/네 package 전체 Race135.126s, public single/peer/empty
134.084s 및 기존 npm 실제6.243s가 PASS다. 별도 focused native update와 final fixture compile은
영수증을 따른다. Same-content 재설치는 새 검사를 하며 inode/mode를 유지하고 단계별 실패는
원본/state를 rollback한다. Foreign output/불확실 journal은 보존하고 재진입을 거부한다. Host 전후
동일/helperwait0/정리를 확인했다. Native payload adoption/CLI/원격·whole P2는 미완료다.
다음은 현재 CI actual consumer wiring과 CLI/독립 source-proof 인계 지도다. M13-004만 IN_PROGRESS다.

### 2026-10-25 P2 B5b4 existing CI actual consumer local checkpoint

[B5b4 결과](./evidence/package-client-quarantine/p2c5b5b4-pnpm-ci-consumer-local-20261025.json)는
새 job 없이 기존 PNPM 공급의 고정 project profile과 owning parent의 tagged client/helper/복원을
연결했다. Canonical5/Bash·checker full Race29.928s, 실제 CI script의 cache+final transaction
197.220s와 helperwait0/Host 구성 불변/정리는 PASS다. 완료 전용 디렉터리6개947,906,618B는
참조·query errors0와 정확한 inventory 확인 뒤 삭제했다. 처음 자기 열람 FD query의 거부와
교정 재조회는 원본에 보존했다. 실제 원격 순서·Required는 미확인이다. Backend 단위를 한 번
묶어 push하고 새 실행의 첫 확인은 최근54m32+새197s/준비 여유에 근거해 시작 약60분 후다.
대기 중 같은 P2의 CLI/native source-proof 인계를 계속하며 M13-004 하나만 IN_PROGRESS다.

### 2026-10-25 P2 B5c PNPM production CLI local checkpoint

[B5c 검증](./evidence/package-client-quarantine/p2c5b5c-pnpm-cli-local-20261025.json)에서 기존
PNPM backend를 typed descriptor/preflight와 production Run에 연결했다. npm source와 client를
분리하고 generated pin은 Bootstrap의 inert composition으로 전달한다. Actual CLI180.442s의
9fresh operations(single/peer/empty, install/repeat/add/update/explicit managed-controls)와
implicit/native-unproven refusal·원본 inode/0600/user files·실제 Manifest/SBOM을 확인했다.
기존 npm inspect actual9.234s, final canonical5·five-package Race134.175s/checker Race30.325s,
Darwin bootstrap/CLI compile6.736s도 PASS다. Helperwait0/Host 구성 보존 및 세 완료 디렉터리
501,366,905B 정리다. CLI actual의 고정 명령을 기존 CI project profile에 독립5분 호출로 추가했다.
CI407/e9ca85e는 앞 backend만 포함하며 첫 확인06:02:54 KST 전 추가 조회/재push하지 않는다.
Native payload 인계의 independent old-source proof가 다음이고 whole P2는 미완료다.

현재 후속 [B5d native original ownership 감사](./evidence/package-client-quarantine/p2c5b5d-pnpm-native-ownership-plan-20261025.md)는
guest store/UID 의미를 Host 원본 claim으로 대체하지 않는 분리와 기존 full source closure 재사용
지도를 두 번째 검토한 뒤 native source-proof 경로를 연결했다. Guest parser의 UID/GID1000·caller store 계약은 유지하고 실제 Host owner의 별도 unverified view와 전체 closure를 재사용한다. 첫 engine Quick195.642s는 PASS이며 composed guard/source/rollback와 actual 검증은 진행 중이다. Native proof는 아직 QUALIFIED/ACCEPTANCE_CLOSED가 아니고 M13-004 하나만 IN_PROGRESS다.

### 2026-10-25 CI407 backend named remote qualification

실제60분 타이머 만료 뒤 [CI407 결과](./evidence/package-client-quarantine/p2c5b-ci407-qualification-20261025.json)를
확인했다. Required 포함10SUCCESS이고 e9ca85e와 모든 checkoutc4da8b10의 전체tree1833b892/
2263files가 같다. 실제56m37다. 새 실제 cache peer/empty33.52s와 project transaction single/
peer/empty115.83s·fresh records/independent approval/native install·repeat·add·update/기존 inode/
0600·user files·root script 미실행을 확인했다. Linux top-level62/CPU280.90s·observer112씩/
native macOS Go1.27.3 build/default와 helper7wait0·final confirmed cleanup original0/clienterror0/
clientrestore0도 PASS다. Parent의 새 protected client rebind와 sandbox 복원 소비자 gap을 닫았다.
전체10 decoded UTF8 logs를 보존·대조했다. 이 성공은 뒤 a03ff13 CLI와 native proof/whole P2를
포함하지 않는다. 현재 B5d original source-proof audit를 계속하며 M13-004 하나만 IN_PROGRESS다.

### 2026-10-25 P2 B5d independent native original ownership local qualification

[B5d 결과](./evidence/package-client-quarantine/p2c5b5d-pnpm-native-ownership-local-20261025.json)는
actual Host census/owner와 guest1000을 분리하고 독립 old public metadata/SRI/전체 archive·peer/
instance/link/bin/file closure를 선택 전에 검증했다. Native first adoption은 새 clean inspected
payload와 current Policy/records를 사용하며 원본 proof를 독립 state provenance로 기록한다.
Native128.739s의 explicit add6→7/exact6 update/peer3사례·intended tamper/SRI/extra/implicit refusal와
user files/actual records가 PASS다. Fixture owner/JSON expectation의 두 실패를 보존하고 test만
교정했다. Canonical5/일곱 package Race157.139s/checker29.742s, final CLI9operation/npm actual·
Darwin compile, helperwait0/Host 불변과6개 완료 runtime 정리도 결과에 연결했다. Whole P2는
remote/영향받는 CPU·세CUDA와 최종 support acceptance 전까지 미완료다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-25 CI408/409 remote qualification과 연속 대기 교정

[Source9410c2d 결과](./evidence/package-client-quarantine/p2c5b5d-ci408-409-qualification-20261025.json)는
ordinary408/60m30와 cu126409/55m05 각각 Required10SUCCESS·전체2280파일 treee4ea86c 일치를
확인했다. Final CLI9operation/native3·cache/transaction·observer112씩·CPU289.90/182.04s 및
cu126927.76s/native macOS·helper7wait0/final confirmed clean이다. 두 timer는 DUE를 기록했지만
응답을 종료해 후속 소비를 놓쳤다. 사용자 지시에 따라 확장 계획§5에 active turn+clock 대기와
결과 확인 뒤 후속 구현 지속을 명시했다. 작은 수정별 full CI는 반복하지 않는다.

남은 기존 필수 cu130은 같은9410 source의 CI410/run40150614641이며 첫 clock 확인은
2026-10-25 11:31:10 KST다. 이후 cu132를 순차 검증하고 support/acceptance를 닫는다.
대기 중 같은 M13-004의 P3 Bun baseline·공식 버전/형식/재사용 경계를 확인하고, P2 결과와
독립적인 공급 검증은 두 번째 검토 뒤 로컬에서 진행한다. P2 source9410은 원격에서 고정하고
P2 최종 acceptance 전 전체 지원 완료를 표시하지 않는다. 이는 새 병렬 work item이 아니다. M13-004만 IN_PROGRESS다.

### 2026-10-25 P3-A Bun signed publisher supply local checkpoint

같은 M13-004에서 P2 fixed9410의 cu130/132 대기와 독립적인 [Bun 공급 검증](./evidence/package-client-quarantine/p3a-bun-supply-local-20261025.json)을
로컬 연결했다. [baseline/edit map/두 번째 검토](./evidence/package-client-quarantine/p3-bun-baseline-20261025.md) 뒤 기존 locked crypto와
bounded reader를 사용했고 실제 공식 signed checksums·52MB zip을 검증했다. Publisher/archive만
인증하며 version/source-build/실행/Policy를 승인하지 않는다. Canonical4/관련4package Race와
새 reachable parser의 canonical vulnerability·actual5cases를 확인했다. 첫 import review와
operator routing 오류를 보존했다. Runtime pin/새 Bun 설치·실행 지원은 아직 없다. 다음은 B의
bounded ZIP/ELF·ABI/기존 native maintenance와 guest admission이다. 개별 공급 단위로 원격 CI를
추가하지 않는다. M13-004 하나만 IN_PROGRESS이며 기존 profile timer를 active turn에서 소비한다.

### 2026-10-25 P3-B1 Bun ZIP/ELF local checkpoint

[B1 native data](./evidence/package-client-quarantine/p3b1-bun-native-local-20261025.json)는
signed complete ZIP→정확한 두 members/CRC·bounds→GNU ELF→native hash/size/libs를 연결했다.
기존 PNPM ELF 조건은 기계적 대조로 보존했고 Quick/security/docs와 관련 네 package 전체
Race·실제 offline5case가 PASS다. Native version/guest/production pin·전체 P3는 미완료다.
다음은 기존 no-replace native publisher를 재사용하는 격리 query용 candidate 출력이다.
P2 source9410/cu130410의11:31:10KST 타이머는 활성 turn에서 소비하며 작은 B1은 push하지 않는다.

P3-B2a는 [candidate audit·exact map·두 번째 검토](./evidence/package-client-quarantine/p3b2a-bun-candidate-plan-20261025.md)
뒤 착수한다. 기존 PNPM exclusive publisher와 B1 authenticated reader를 재사용하고 격리
조회용 후보 출력 gap만 구현한다. Production pin/runtime version/guest·전체 P3는 아직 미완료다.

### 2026-10-25 P3-B2a candidate/native query local checkpoint

[후보 출력과 격리 조회](./evidence/package-client-quarantine/p3b2a-bun-candidate-local-20261025.json)는
기존 exclusive publisher를 재사용하며 official bytes·mode/owner/single-link·재출력 거부와 기존
verify5case/PNPM 회귀를 통과했다. 고정 Node image의 nonroot/network-none/read-only runc에서
1.4.4/1.4.4+38afee6c8를 확인했고 exact task container3개는 제거/부재 확인했다. Runtime pin/
release/installed role·gVisor admission은 다음 B2b/B2c다. 로컬 중간 commit은 push하지 않는다.

P3-B2b는 [production runtime 착수 지도](./evidence/package-client-quarantine/p3b2b-bun-runtime-plan-20261025.md)로
기존 owner/maintenance/release 소비자의 구현 상태를 감사했다. 먼저 exact publisher proof와
공식 source/내장 notices 자료를 확인하고 schema4 단일 owner·기존 frozen updater gap만
구현한다. B2a 로컬 commit6e927b4도 push하지 않았으며 remote source9410은 유지한다.

### 2026-10-25 P3-B2b1 Bun owner/frozen maintenance local checkpoint

[Schema4 owner와 유지보수](./evidence/package-client-quarantine/p3b2b1-bun-owner-local-20261025.json)는
단일 publisher/archive/native/notices identity와 generated drift/support, 기존 journal의
plan/apply/verify를 연결했다. Actual changed plan·중단 후 재개·재실행/고정 archive 변조 거부와
canonical/관련 여섯 package full Race가 PASS다. 첫 source routing/빠진 capture schema/문구
gate를 보존하고 교정했다. 다음은 B2b2 release/package/installer/activation complete tuple이다.
Production Bun gVisor/CLI·전체 P3는 아직 열려 있고 remote9410/현재 CI410 timer를 유지한다.

### 2026-10-25 CI410 cu130 checkpoint와 연속 timer 소비

[CI410 cu130](./evidence/package-client-quarantine/p2c5b5d-ci410-qualification-20261025.json)은
65분 첫 timer에서8SUCCESS/Linux pending을 확인하고15분 다음 timer에서 Required 포함
10SUCCESS를 확인했다. Source/all checkout9410·whole treee4ea/2280files 동일, full decoded
logs10개를 매 chunk readback으로 보존했다. Linux65/observer112씩·PNPM cache/transaction/
CLI9operation/native3, CPU286.74s/cu1301094.44s·native macOS Go1.27.3, helper7wait0와 final
original0/CONFIRMED_STOPPED/stop0/client errors0를 확인했다. 전체 경과79m48이다.
Same remote9410에서 마지막 필수 cu132411을 한 번 dispatch했고95분 첫 timer는13:22:46KST다.
Active turn을 유지해 그 시각에 결과를 소비하며 Bun B2b2 로컬 작업을 이어간다. 작은 Bun
단위로 push/CI를 만들지 않았고 그 local source를 이번 성공으로 전용하지 않는다. P2 전체
acceptance/support는 cu132와 최종 matrix 전까지 열려 있다.

P3-B2b2는 [release 공급·설치 tuple 지도](./evidence/package-client-quarantine/p3b2b2-bun-release-plan-20261025.md)의
baseline/두 번째 검토 뒤 A production materializer부터 진행한다. Existing candidate/publisher/
owner를 재사용하고 실제 complete release/installer tuple B는 그다음 연결한다. 원격411의
95분 타이머는 유지하며 이 local 단위로 새 push/CI를 만들지 않는다.

### 2026-10-25 P3-B2b2-A Bun production supply local checkpoint

[생산용 출력](./evidence/package-client-quarantine/p3b2b2a-bun-production-local-20261025.json)은
source owner의 signature/archive/native/notices를 다시 확인해 native0555/passive0444를 새
외부 경로에만 게시했다. Canonical/관련 세 package 전체 Race·공식 생산 actual 및 기존
frozen maintenance actual이 PASS다. 첫 realcorpus fixture-only compile 오류와 default
profile의 정확한 제외 범위도 보존했다. 다음은 B2b2-B complete producer/installed tuple이다.
Remote9410/cu132411 timer04:22:46UTC는 유지하고 이 local 단위는 push하지 않는다.

### 2026-10-25 P3-B2b2-B complete release/installed tuple local checkpoint

[Bun 배포/설치 tuple](./evidence/package-client-quarantine/p3b2b2b-bun-release-local-20261025.json)은
현재 Tool2/asset13/native6/wire10/export11과 exact source11subjects/required28을 연결했다.
Final canonical4와 관련 full Race의 통과한 여덟 package/마지막 세 affected package를 정확히
분리했고 actual12 namespace cases·보호 Host snapshot 보존 및 기존 PNPM 생산 회귀가 PASS다.
완료 namespace 두 copies75,007,792B는0refs/query errors0 뒤 정리했다. Actual Bun guest/새
서명 release·whole P3는 미완료다. 다음 B2c installed Bun supply/guest volume·observer exact
startup/version/lock admission을 같은 M13-004에서 계속한다. Remote9410/cu132411의
04:22:46UTC 타이머와 실행 중 source는 유지하며 local Bun을 그 성공으로 전용하지 않는다.

### 2026-10-25 CI411 timer 실제 확인과 남은 gate

예정04:22:46UTC 뒤04:22:58UTC에 [CI411](./evidence/package-client-quarantine/p2c5b5d-ci411-observation-20261025.json)을
확인했다. Source/all10checkout9410·whole treee4ea/2280files 동일이며 모든 decoded log를
chunk readback으로 보존했다. Linux/cu132·PNPM9CLI/native3·observer112씩/Linux65와
helper7wait0/최종 original0·CONFIRMED_STOPPED/stop0/client errors0는 PASS다. MacOS의
default-test outer6분 timeout과 Required는 FAILURE여서 overall SUCCESS로 표시하지 않는다.
정확한 remote 소스·active snapshot을 보존했으며 마지막 테스트 이름을 원인으로 단정하지
않는다. 같은 source 무조건 재실행이나 작은 Bun commit push는 하지 않았다.

[직접 실패 acceptance](./evidence/package-client-quarantine/p2c5b5e-pnpm-failure-local-20261025.json)의
SIGINT/SIGKILL/Race·실제1MiB ENOSPC를 완료했고 production transaction은9410과 같다.
Bun [exact guest query](./evidence/package-client-quarantine/p3b2c2-bun-query-local-20261025.json)는
114latches/실제 version+revision/session complete와 Host 복원을 named local 검증했다. 다음은
비어 있지 않은 Bun lock·독립 source/graph binding이며 P2의 macOS/Required와 전체 Bun
qualification은 후속의 의미 있는 checkpoint까지 열린 상태다.

### 2026-10-25 P3-B3a Bun actual raw lock / PNPM shared scope

[B3a local receipt](./evidence/package-client-quarantine/p3b3a-bun-lock-local-20261025.json)는
고정 Bun의 lock-only/raw controls와 기존 PNPM lifecycle을 private fixed-kind scope로 재사용한
범위다. PNPM scope/input의 mechanical equality, 최종 Go Quick82.541s·전체 sandbox Race24.284s·
tagged vet6.088s, 원본 observer release build249.527s/117latches가 PASS다. Canonical observer의
actual Bun4case27.09s는 empty의 실제 lock 부재와 single350B/transitive534B/peer1056B,
manifest 불변·complete census/observation/cleanup을 확인했다. 기존 PNPM의 actual boundary5,
public selection8와 quarantine4도 같은 observer에서147.74s/operator·Host 전체 불변/helperwait0다.

최초 volume 생성/종료 whitelist 누락과 exact startup/DNS read 거부·준비 operator의 공식
archive 단일 member 오인은 원본을 보존했다. 권한은 Bun project 명령/경로/flags/역할에만
한정했고 query/PNPM/UV/ARTIFACT/clone·unknown cleanup 계약은 유지한다. 완료한 진단 사본
7개1,623,440,562B는 참조0/query errors0 뒤 정리했으며 RSS 회복으로 쓰지 않는다.
다음은 관측된 schema2/config1 parser와 독립 npm source/전체 instance binding이다. 아직
public Bun resolver/layout/cache/transaction/CLI/whole Bun acceptance는 없고 macOS/Required
원격 gate도 열려 있다. 작은 helper별 push/CI 없이 같은 M13-004 하나만 IN_PROGRESS로 계속한다.

B3a 중간 commit b658670/tree6e100e8·2340개 전체 후보 일치를 확인했다. 다음
[B3b parser/source 지도](./evidence/package-client-quarantine/p3b3b-bun-parser-source-plan-20261025.md)의
baseline·정확한 수정 경로·두 번째 검토를 마쳤고, 첫 lexical JSONC subset의 MISSING부터 진행한다.
Bun output/source binding을 Policy 또는 배포 지원으로 확대하지 않는다. 새 CI는 시작하지 않았다.

### 2026-10-25 P3-B3b Bun parser·독립 npm source·snapshot

[B3b local receipt](./evidence/package-client-quarantine/p3b3b-bun-source-local-20261025.json)는
실제 schema2/config1 JSONC의 bounded parser, scoped instance/ancestor dependency·단일 matching
peer와 전체 coverage, exact controls/runtime snapshot을 연결한 named local 범위다. 기존
fetchMetadataBody/resolvedVersion을 재사용했으며 PNPM source는 변경하지 않았다. 이미 고정된
semver3.5.0을 검토한 npm caller에서만 사용하고 version/checksum은 유지했다.

최종 Quick101.552s/npm 전체 Race50.962s와 unchanged checker32.512s, vulnerability18.940s/
tagged vet0.520s·실제 HTTPS metadata4cases1.172s가 PASS다. 실제 native manifest/lock bytes는
그대로 frozen하고 source의 SRI·dependency/peer/bin을 대조했다. Generic empty-type의 직접
RED14개와 교정·정상 공백 보존을 기록했으며 그 이전 green을 해당 타입 검증에 전용하지 않는다.
Source/snapshot은 unapproved이고 archive/lifecycle/Policy/Host install은 수행하지 않았다.
다음은 같은 M13-004에서 preflight typed selection과 Bun native install/add/update·frozen
output의 연결이다. Layout/cache/transaction/CLI와 제한 밖 grammar/peer/platform 및 whole
Bun acceptance는 열려 있고 새 push/CI는 없다. Remote9410 CI411 macOS/Required gate는 유지한다.

B3b 중간 commit d1d3625/treec3e1a2fe·2361개 전체 후보 일치 뒤
[B3c 연결 지도](./evidence/package-client-quarantine/p3b3c-bun-native-selection-plan-20261025.md)의
baseline·기존 구현/정확한 수정 경로·두 번째 검토를 마쳤다. 고정 add/update help의 실제 격리
조회도 종료/삭제까지 확인했으며, 같은 M13-004에서 typed native 선택 연결의 MISSING만 진행한다.

### 2026-10-25 P3-B3c typed native Bun 선택 연결

[B3c local receipt](./evidence/package-client-quarantine/p3b3c-bun-native-selection-local-20261025.json)는
immutable install/add/update/frozen input을 고정 native owner와 independent source/snapshot에
연결했다. 기존 scope/metadata 비교/HTTP/parser/Domain engine은 재사용했고 새로운 filesystem/
network grant는 없다. 원본 observer250.792s/117latches·최종 Quick111.564s/전체 npm+sandbox
Race59.056s/tagged vet2.825s/vulnerability16.694s가 PASS다.

최종 native 선택9/source/snapshot57.54s와 실패 경계5/29.67s, 기존 PNPM boundary5/public8/
quarantine4도 같은 observer/client139.641s/operator에서 PASS다. 두 owning helper는 wait0,
최종 original0/CONFIRMED_STOPPED/stop0/client errors0·Host 전체 동일이다. 합성 constraint를
가진 old6.0.0을 native frozen으로 읽은 뒤 selected/full update의 실제7.0.0을 확인했으며,
unmanaged target adoption·archive 검사·Policy로 확대하지 않는다.

다음은 같은 M13-004의 B4 exact native layout/lifecycle와 archive static-all/검역 연결이다.
Cache/transaction/CLI와 제한 밖 package semantics·전체 Bun remote acceptance는 열려 있다.

### 2026-10-25 P3-B4 native layout·lifecycle 착수

[B4 baseline·첫 primitive 지도](./evidence/package-client-quarantine/p3b4-bun-layout-lifecycle-plan-20261025.md)는
cc062c5의 typed selection/source를 재사용하고 아직 없는 Bun offline archive/cache·hoisted tree와
lifecycle 검역 연결만 수행한다. 첫 단계는 fixed native의 network-none script-free 표현 조사다.
PNPM store/layout의 재구현이나 가정 전용 없이 실제 입력/결과를 확인하고 production edit map을
보완한다. M13-004 하나만 IN_PROGRESS이며 새 push/CI는 없다.

B4a [frozen 검사 입력 로컬 영수증](./evidence/package-client-quarantine/p3b4a-bun-inspection-input-local-20261025.json)은
기존 PNPM acquired 검사의 기계 이동·Bun typed context와 전체 instance 보존을 확인한다.
Canonical Quick188.402s·npm/sandbox 전체 Race48.704s, 실제 공개 empty/single/transitive/peer
acquire/SRI/static-all→전체 acquired 결합1.423s(archive0/1/2/4)가 PASS다. 별도 runc offline
단일 package primitive는 정확한 native payload4파일/원본 controls를 보존했으며 production
gVisor/layout/lifecycle/Policy의 qualification은 아니다. 다음은 실제 hoisted/bin 표현과 그 전체
archive binder·bounded cache envelope를 연결한다. 같은 M13-004만 IN_PROGRESS다.

[B4b exact layout 지도](./evidence/package-client-quarantine/p3b4b-bun-layout-binding-plan-20261025.md)의
baseline·실제 hoisted/bin 표현·기존 tar/budget 재사용 경계를 두 번째 검토했다. Script-free runc
transitive/peer의59 native files와 정확한 bin link를 관측했으며 이 범위의 전체 archive binder만
연결한다. Production gVisor/동적 lifecycle/Policy는 다음 gap이고 새 CI는 없다.

B4b [전체 트리 결합 로컬 영수증](./evidence/package-client-quarantine/p3b4b-bun-layout-binding-local-20261025.json)은
canonical Quick188.001s·npm/sandbox 전체 Race47.291s와 actual source/native-capture1.422s를
확인한다. Transitive/peer의 archive2/4·entries11/63·전체 target 및 partial/bin substitution 거부가
PASS다. Packing0664→HAA prepared0644의 최초 해석/거부를 보존하고 source022 축소만 exact view로
대조한다. 앞 compile/SA1012·operator 준비 오류도 별도 원본으로 남겼다. 완료 복사본7곳의
778,244,261B는0refs/query errors0 뒤 정리했다. 다음은 production bounded cache/offline prepare와
gVisor lifecycle 관찰 연결이다. 전체 B4/Bun·macOS/Required remote acceptance는 열려 있다.

[B4c production offline 준비 baseline](./evidence/package-client-quarantine/p3b4c-bun-offline-prepare-plan-20261025.md)을
현재8ffb0c9에서 읽고 native file-archive seed/cache 표현을 먼저 확인한다. Source archive의 SRI/static
inventory와 기존 native scope/trace/cleanup을 유지하며 준비 출력만 whole binder에 연결한다.
새 work item/CI를 만들지 않고 같은 M13-004의 실제 MISSING을 계속한다.

B4c-a [cache 데이터 결합 영수증](./evidence/package-client-quarantine/p3b4ca-bun-cache-data-local-20261025.json)은
canonical Quick191.832s·npm/sandbox 전체 Race50.203s와 actual frozen-cache1.571s를 확인한다.
Fresh public source archive2/4와 native member 전체를 대조하고 marker를 제외한 registry data
24,576/423,424B·partial 거부가 PASS다. Native lookup index는 authority가 아니고 전체 global cache
검증을 주장하지 않는다. 두 expanded cache copies와 모든 logical placement를 같은128MiB
ceiling에 청구하며 implied directories도 제한한다. Production preparer/observer 연결과 실제
여러 version·lifecycle/Policy·cache/transaction/CLI/remote는 열려 있고 같은 M13-004에서 계속한다.

B4c-b는1ea12d9의 cache data를 재사용해 production seed/cache/whole-project 준비와 두 exact
observer argv shape를 연결한다. 기존 PNPM freeze/readback의 fixed-kind 추출을 대조했고 native
명령 전 disconnect proof를 재사용한다. [B4c-b 로컬 영수증](./evidence/package-client-quarantine/p3b4cb-bun-offline-prepare-local-20261025.json)의
Quick184.159s/전체 관련 Race57.353s·원본 builder267.752s/117latches와 actual Bun74.101s가
네 graph·5실패 경계를 확인했다. 기존 PNPM production prepare2/public selection8·검역4/
실패 경계5도173.946s/operator·helper wait0/Host 동일로 PASS다. Actual observer는 원본
builder 산출물과 byte-identical이며 불필요하게 같은 실행을 반복하지 않았다. FS/network
권한은 추가하지 않았다. 다음은 정확한 instance lifecycle/runtime·전체 검사/Policy 연결이다.
작은 helper별 push/CI는 시작하지 않았고 remote9410/CI411 macOS·Required gate는 유지한다.

[B4d instance lifecycle 지도](./evidence/package-client-quarantine/p3b4d-bun-instance-lifecycle-plan-20261025.md)는
4420b48의 준비 구현을 재사용한다. 먼저 기존 PNPM payload renderer/readback을 두 typed layout에
연결해 Bun 실행 세션용 최소 데이터의 전체 이송을 검증한다. 이후 exact instance lifecycle/
Node·Bun runtime/whole coverage·Policy를 연결하며 아직 실행하지 않은 범위를 완료로 쓰지 않는다.
같은 M13-004이고 작은 helper별 새 CI 없이 backend 묶음·실제 타이머 규칙을 유지한다.

[B4d-a 로컬 영수증](./evidence/package-client-quarantine/p3b4da-bun-execution-payload-local-20261025.json)은
Quick170.358s/전체 sandbox Race23.229s와 Bun peer payload 전체 이송15.01s를 확인한다.
공유 body의 기존 PNPM project2/4 fresh dynamic sessions도47.53s·전체 operator65.119s/
helper wait0/Host 동일로 PASS다. 원본 controls를 실행 manifest로 복사하지 않고 두 fixed kind
사이 payload 혼용도 거부한다. 다음은 정확한 Bun instance의 lifecycle 명령/관찰기 연결이며
Bun 실행·Policy/cache/transaction/CLI/remote acceptance는 계속 열린다.

[B4d-b target lifecycle 감사](./evidence/package-client-quarantine/p3b4db-bun-target-lifecycle-plan-20261025.md)는
da02f4e의 payload를 재사용한다. 먼저 Bun hoisted/nested/scoped 한 instance의 고정 npm/Node
quarantine lifecycle과 별도 attested topology/creator·one-shot 경계를 실제 primitive로 확인한다.
Bun executable이 필요한 script·production backend/whole inspection/Policy는 후속 범위다.
M13-004 하나만 IN_PROGRESS이며 backend 묶음 외 새 CI는 시작하지 않는다.

[B4d-b 로컬 영수증](./evidence/package-client-quarantine/p3b4db-bun-target-lifecycle-local-20261025.json)은
Quick199.536s/전체 sandbox Race26.591s·원본 builder250.402s/119latches와 직렬 actual57.515s를
연결한다. Bun shallow/nested-scoped 정상·exit37 네 case와 기존 PNPM target 회귀가 PASS다.
ARTIFACT shell3/2·비대상0/complete trace를 유지하고 helper wait0/Host 전체 동일을 확인했다.
첫 operator의 동시 compiler 간섭 실패는 원본으로 보존하며 compiler 종료/원상 복원 뒤 같은
binary로 직렬 확인했다. 다음은 opaque Bun layout를 소비하는 production instance backend이며
명시 Bun runtime·whole inspection/Policy/cache/transaction/CLI/remote는 계속 미완료다.

[B4d-c production backend 감사](./evidence/package-client-quarantine/p3b4dc-bun-instance-backend-plan-20261025.md)는
3f328e2의 target primitive를 opaque Bun layout 소비자에 연결한다. 기존 PNPM session/관찰/정리
body를 private 두-kind engine으로 재사용하고 typed preflight·fresh resource·whole readback 및
첫 cause/INCOMPLETE를 유지한다. 실제 public peer 전체 instance와 기존 PNPM을 확인한 뒤
whole inspection/Policy·명시 Bun runtime 연결을 계속한다. M13-004 하나만 IN_PROGRESS다.

[B4d-c 로컬 영수증](./evidence/package-client-quarantine/p3b4dc-bun-instance-backend-local-20261025.json)은
Quick161.883s/전체 sandbox Race25.135s와 actual79.128s를 연결한다. Fresh public Bun peer
archive4의 exact instances4를 production backend의 fresh sessions4에서31.41s로 확인했고
기존 PNPM project2/4 sessions도45.02s가 PASS다. Whole readback/trace·cleanup 뒤 반환,
helper wait0/Host 전체 동일을 확인했다. 다음은 전체 static/dynamic 집계·coverage·Policy
연결이며 명시 Bun runtime/cache/transaction/CLI·remote qualification은 아직 미완료다.

[B4d-d complete inspection 감사](./evidence/package-client-quarantine/p3b4dd-bun-project-inspection-plan-20261025.md)는
e212fa4의 backend를 Bun typed Project Inspector에 연결한다. 모든 unique archive의 static gate와
모든 logical instance의 실제 결과/coverage·정확한 prepared handoff를 기존 evidence/Policy 서비스에
전달한다. PNPM adapter·판정 규칙은 유지하고 shared facts renderer를 닫힌 두 client로 재사용한다.
실제 검역·기존 PNPM을 확인한 backend 묶음에서만 필요한 CI와 실제 타이머 확인을 수행한다.

[B4d-d 로컬 영수증](./evidence/package-client-quarantine/p3b4dd-bun-project-inspection-local-20261025.json)은
Quick149.369s/전체 inspection·sandbox Race24.431s와 actual160.202s를 연결한다. Bun transitive/
peer source2/4·모든 instance2/4의 필수 integrity/static/lifecycle3·정확한 facts/기록된 evidence와
entry/set Policy ALLOW(47.05s), exact prepared handoff/partial 거부가 PASS다. 기존 PNPM public
선택8·검역4도110.56s/helper wait0/Host 전체 동일이다. Script-free Node-lane backend 검증이며
명시 Bun runtime·Host 설치 승인이 아니다. 다음은 frozen Bun 공급/actual CI 연결 후 이 backend
묶음의 원격 checkpoint이고, cache/transaction/CLI는 CI 대기 중 독립적으로 계속한다.

[B4d-e backend CI 감사](./evidence/package-client-quarantine/p3b4de-bun-backend-ci-plan-20261025.md)는
37fec98의 local whole quarantine backend를 frozen 공급/actual child로 기존 owning CI lifecycle에
연결한다. Job/Required/timeout·기존 PNPM 전환은 유지하고 helper별 CI 없이 backend commits를
묶는다. 원격 실행 시 실제 timer를 소비하고 대기 중 후속 cache/transaction/CLI를 계속한다.

[B4d-e 로컬 영수증](./evidence/package-client-quarantine/p3b4de-bun-backend-ci-local-20261025.json)은
workflow2.407s/Quick78.075s·전체 checker Race33.607s/vulnerability14.984s와 실제 CI script
142.666s를 확인한다. 같은 script가 namespace 안에서 pinned 공급→native9/실패경계5/whole
quarantine2를 실행하고 설치/work 정리·helper wait0/Host 전체 동일을 확인했다. 다음은 이
backend local commits를 묶어 한 번 push하고 ordinary CI의 macOS/Required를 실제 timer로
확인한다. 기다리는 동안 independent cache/transaction/CLI를 계속하며 전체 Bun acceptance는
아직 닫지 않는다. Job/Required/기존 timeout·PNPM 전환과 runtime/Policy 권한은 유지한다.

CI412/run40192015597은7544a09/tree3b7dbd03(2423files)의 backend 묶음에서 실행 중이다.
시작12:11:17UTC와 ordinary408의60m30+새 Bun script142.666s를 근거로 첫 실제 timer를
13:16:17UTC/22:16:17KST에 등록했다. DUE만으로 완료라 하지 않고 결과/원본/source를 확인한다.
[B5a approved cache 감사](./evidence/package-client-quarantine/p3b5a-bun-approved-cache-plan-20261025.md)는
그 결과와 독립적인 typed Bun staging/records를 진행한다. 기존 publication/Evidence/byte reader를
재사용하고 같은 M13-004에서 cache→private materialization/transaction/CLI를 계속한다. 현재 CI
후보를 새 push로 취소하지 않는다.

[B5a 로컬 영수증](./evidence/package-client-quarantine/p3b5a-bun-approved-cache-local-20261025.json)은
Bun 승인 cache/actual operation records·전체 native provenance와 absent lock 보존을 연결한다.
실제 native 선택→검역/Evidence/Policy→Stage/Open은 peer4/empty0, 기존 PNPM 승인 cache·private
준비도 combined90.683s·helperwait0/Host 동일로 PASS다. 전체 evidence/promotion/bootstrap
Race158.098s와 canonical Quick/security/docs도 통과했다. 첫 fixture mismatch·all-present record
거부 원본을 보존하고 ordinary builder guard/present bytes를 유지했다. 다섯 공용 body의 PNPM
정규화 동일성을 대조했다. Local cache subset은 닫혔으며 private materialization/transaction/CLI·
명시 Bun runtime·remote/whole P3는 열려 있다. CI412 source7544를 취소하는 push는 하지 않는다.

[B5b1 private materialization 감사](./evidence/package-client-quarantine/p3b5b1-bun-private-materialization-plan-20261025.md)는
7939092의 cache를 현재 recorded approval 재확인 뒤 private sibling에 복원한다. 기존 PNPM
Host entry/owned Prepare·Verify/Close body를 재사용하며 absent lock을 만들지 않는다. Target
publication/retained ownership/CLI는 다음 단계다. M13-004 하나만 IN_PROGRESS이며 CI412
7544 source·실제 타이머를 유지하고 새 push 없이 로컬 검증을 계속한다.

[B5b1 로컬 영수증](./evidence/package-client-quarantine/p3b5b1-bun-private-materialization-local-20261025.json)은
승인 Bun peer4/empty0의 private sibling·Verify/Close와 lock 부재 보존, 기존 PNPM 실제 소비자를
combined85.368s/helperwait0/Host 동일로 확인했다. 두 기존 body의 기계 정규화 동일성과
create/owner/census/readback/cleanup 전체 불변을 확인했다. Quick147.243s/security/docs와
전체 promotion/bootstrap Race도 PASS다. Private materialization local subset만 닫혔고 target
publication/retained ownership/CLI/native 인계·명시 Bun runtime/whole P3는 미완료다.

[B5b2 target transaction 감사](./evidence/package-client-quarantine/p3b5b2-bun-target-transaction-plan-20261025.md)는
74102c8의 private engine을 같은 target lock·independent state·backup/publish/rollback engine으로
연결한다. Explicit absent selected lock은 별도 Domain snapshot constructor와 Bun fixed profile로
보존하며 기존 PNPM guard를 유지한다. Native/unmanaged adoption은 계속 거부하고 typed
Application/bootstrap의 실제 guarded Apply 소비자로 검증한다. CI412 타이머·frozen7544를
유지하며 새 push 없이 같은 M13-004 하나의 로컬 작업을 계속한다.

[B5b2 로컬 영수증](./evidence/package-client-quarantine/p3b5b2-bun-target-transaction-local-20261025.json)은
typed Bun request-local guarded Apply와 independent state·backup/publish/rollback engine을 연결했다.
실제 Bun/기존 PNPM single·peer4·empty 및 install/no-op/add6→7/selected update가 모두 PASS다.
매번 fresh operation/Evidence/Policy·record, no-op control/payload inode·mode와 user 파일, Bun lock
부재를 유지했다. Combined 실제 시간·helperwait0/Host 동일은 영수증에 기록했다. Quick216.022s/
security/docs·Domain/Application/promotion/bootstrap 전체 Race231.114s도 PASS다. 첫 actual
no-op 실패와 RED/GREEN을 보존하고 검역 handle binding을 제거하지 않은 전체 Host census
비교로 보완했다. Local managed transaction만 닫혔으며 CLI/native ownership/명시 Bun runtime/
remote·whole P3는 열려 있다. CI412 frozen7544의 타이머 확인과 독립 작업을 계속한다.

[CI412 backend qualification](./evidence/package-client-quarantine/p3b4de-bun-backend-ci412-20261025.json)은
65분 실제 timer13:16:17UTC를 소비하고13:16:26에10SUCCESS를 확인했다. 모든 checkout0bae4d5의
whole tree3b7dbd0/2423files가 frozen7544와 같고 전체 decoded logs10개/1,148,222B를 매 chunk
readback·SHA로 보존했다. Linux67/observer119씩, Bun native9/경계5/whole quarantine2·현재
recorded Evidence/Policy, 기존 PNPM CLI9·CPU288.90s·helper7wait0/final0와 native macOS가
PASS다. 실제 전체 경과53m51이다. 과거 CI411 timeout 원인은 확정하지 않으며 이후 local
cache7939092/private74102c8/transaction80bd2bd를 이 원격 결과에 전용하지 않는다.

[B5c managed CLI 감사](./evidence/package-client-quarantine/p3b5c-bun-managed-cli-plan-20261025.md)는
80bd2bd에서 남은 public caller를 연결한다. Bun add/install/update만 등록하고 npm source와
Bun client를 분리하며 help/invalid args는 runtime/cache 없이 끝낸다. Native adoption/explicit
Bun Artifact runtime/bunx는 별도 acceptance다. M13-004 하나만 IN_PROGRESS이고 작은 수정별
새 CI 없이 같은 작업을 계속한다.

[B5c 로컬 영수증](./evidence/package-client-quarantine/p3b5c-bun-managed-cli-local-20261025.json)은
Bun add/install/update의 static preflight·고정 factory·실제 guarded workflow/report를 연결했다.
실제 Bun CLI8 fresh operation170.52s/기존 PNPM9operation186.78s, 전체 operator 시간·helperwait0/
Host 동일을 영수증에 기록했다. Single6→7/peer4/empty·반복 no-op inode/mode·현재 Evidence/Policy/
cache와 target records·부재 lock/user 파일을 확인했다. Quick60.369s/security/docs·CLI/bootstrap
Race28.452s와 PNPM full preflight body 정규화 동일성이 PASS다. 현재 managed Node lane만
닫혔으며 native original ownership/명시 Bun Artifact runtime·hosted caller/whole P3는 열려 있다.
다음은 original full source proof를 거치는 explicit native 인계이며 작은 변경별 CI는 시작하지 않는다.

[B5d native original ownership 감사](./evidence/package-client-quarantine/p3b5d-bun-native-ownership-plan-20261025.md)는
58957ea의 managed CLI 다음에 explicit native 인계 gap만 연결한다. 기존 실제 Host census와
per-entry transport, Bun full layout closure를 실제 UID/GID를 보존한 별도 original wrapper로
재사용한다. Old 전체 공개 source proof는 ownership만 설명하며 새 선택의 검역/Policy는
그대로다. Unknown/corrupt state를 native fallback으로 바꾸지 않고 PNPM 원본 경로도 보존한다.
M13-004 하나만 IN_PROGRESS이며 이 단계의 actual/최종 qualification 전까지 완료로 표시하지 않는다.

[B5d 로컬 영수증](./evidence/package-client-quarantine/p3b5d-bun-native-ownership-local-20261025.json)은
Bun original 전체 source proof·실제 Host owner·explicit --adopt와 fresh current 검역을 연결했다.
Native Bun3/기존 PNPM3 actual270.704s와 같은 binary의 managed CLI9/9 회귀366.181s가
PASS다. Helperwait0/Host 전체 동일·원래 bytes/inode/user data 보존 거부·independent proof/state와
현재 Policy/records를 확인했다. Guest1000 full binder/기존 pinned transport의 기계 동일성,
Quick49.720s/security/docs·전체 관련 Race179.520s도 PASS다. 첫 test tuple/중복 helper 실패
원본을 보존했다. Nonempty copied hoisted native subset만 닫혔으며 hosted project consumer/
명시 Bun Artifact runtime·전체 P3/영향 profile qualification은 미완료다.

[B5e hosted project 소비자 감사](./evidence/package-client-quarantine/p3b5e-bun-project-ci-plan-20261025.md)는
728f650에서 기존 Bun 공급 body의 고정 project mode를 재사용해 cache/transaction/CLI/native
original 실제 소비자를 연결한다. Parent workflow의 이미 컴파일한 bootstrap client/helper와
그 뒤 sandbox 복원을 exact validator로 확인한다. 원래 native mode/PNPM 순서·jobs/Required/
timeouts는 유지한다. 아직 새 push/CI는 없으며 같은 M13-004의 로컬 검증을 계속한다.

[B5e 로컬 공급·연결 검증](./evidence/package-client-quarantine/p3b5e-bun-project-ci-local-20261025.json)은
기존 fixed ZIP/materializer/설치·cleanup을 쓰는 project mode의 네 actual 검사를519.694s에
마쳤다. Cache46.97s/transaction151.89s/CLI9fresh182.70s/native3cases124.28s, no-SKIP·
helperwait0/Host 전체 동일·설치 root/공급 work 정리가 PASS다. Workflow/Quick/security/docs·
checker 전체 Race와 원래 default mode 기계 동일성을 확인했다. Local wiring만 닫혔으며
hosted project/명시 Bun Artifact runtime/전체 P3 qualification은 미완료다. 같은 M13-004에서
Bun actual SIGINT/SIGKILL·작은 tmpfs ENOSPC의 직접 acceptance를 다음으로 진행한다.

[B5f Bun 직접 실패 감사](./evidence/package-client-quarantine/p3b5f-bun-failure-acceptance-plan-20261025.md)는
637dada에서 실제 SIGINT/SIGKILL과1MiB tmpfs ENOSPC의 acceptance만 추가한다. 기존 Bun
transaction/원본 source proof/Policy는 재사용하며 PNPM 직접 실패 회귀도 보존한다. Production
engine은 변경하지 않고 아직 실제 결과 전까지 QUALIFIED/ACCEPTANCE_CLOSED로 올리지 않는다.

[B5f 직접 실패 로컬 검증](./evidence/package-client-quarantine/p3b5f-bun-failure-local-20261025.json)은
Bun·PNPM의 실제 SIGINT/SIGKILL Race15.560s/1MiB tmpfs ENOSPC0.474s·전체 promotion
Race153.066s를 확인했다. 원본 bytes/inode/mode·cause/독립 state 미게시·crash lock/backup 보존·
재진입 거부·FD0가 PASS다. Canonical Quick/security/docs·tagged compile/vet도 PASS이며
production owner9개와 기존 PNPM qualification 파일은 baseline637dada와 같다. Named 세
실패 acceptance만 닫혔으며 명시 Bun Artifact runtime·hosted/전체 P3는 다음으로 이어간다.

[B6a runtime 공급 감사](./evidence/package-client-quarantine/p3b6a-bun-artifact-runtime-supply-plan-20261025.md)는
386777d에서 기존 native tool/readback/volume owner를 별도 Artifact session에 연결한다.
기존 Node 경로·CPP permissions·pin은 유지하며 실제 Bun 실행/admission과 whole coverage는
그 뒤 B6b로 구분한다. Transport 성공을 명시 Bun 실행 완료로 쓰지 않는다.

[B6a 로컬 공급 검증](./evidence/package-client-quarantine/p3b6a-bun-artifact-runtime-supply-local-20261025.json)은
fixed tool 전체 readback과 별도 native/project ownership을 쓰는 peer4 fresh sessions42.09s와
기존 PNPM full project48.69s를 확인했다. 전체 operator93.696s/helperwait0/Host 동일·owned
resource 정리가 PASS다. Quick181.662s/sandbox 전체 Race24.570s/tagged vet·security/docs와
기존 nil-runtime owner 기계 동일성도 PASS다. Node 명령은 그대로이며 명시 Bun Artifact
실행/coverage·Policy/hosted·전체 P3는 B6b와 최종 qualification으로 계속한다.

[B6b 명시 Bun 실행 감사](./evidence/package-client-quarantine/p3b6b-bun-artifact-execution-plan-20261025.md)는
74fd366에서 실제 JS의 Bun global/version을 요구하는 controlled boundary부터 검사한다.
Artifact process 관찰과 기존 M3 MANUAL_REVIEW/BLOCK 의미를 유지하고 불필요한 CONTROL/
네트워크·broad FS 권한을 추가하지 않는다. Whole inspector/hosted qualification은 아직 열려 있다.

[B6b 실제 Bun 실행 경계 검증](./evidence/package-client-quarantine/p3b6b-bun-artifact-execution-local-20261025.json)은
최종 canonical242.161s/121 latches와 actual18.791s의 정상 JS 완전 관찰·typed exit37 보존을
확인했다. 각 ARTIFACT process2·baseline FS/honeytoken/network0·helperwait0/Host 동일이다.
같은 observer의 기존 Bun Node/native supply/PNPM full project actual116.551s도 PASS다.
첫 loader/VM read 실패와 warm 진단을 보존했고 Policy/역할/네트워크·예산/버전은 유지했다.
이 named 경계만 닫혔으며 B6c whole inspector native 경로/Evidence/Policy·hosted는 계속한다.

[B6c whole native 경로 감사](./evidence/package-client-quarantine/p3b6c-bun-whole-runtime-plan-20261025.md)는
9fbc981에서 기존 BunInstanceExecutor capability와 호출만 고정 native runtime 경로로
강화한다. Static-all/정확한 coverage·Evidence/Policy·cache/transaction은 재사용하고 Node-only
fallback은 추가하지 않는다. Whole public 검역·Bun CLI/native 실제 회귀를 확인한 뒤 기존
중간 commit들을 한 번 push하고 실제 elapsed 근거 timer를 active turn에서 소비한다.

[B6c whole native 연결 로컬 검증](./evidence/package-client-quarantine/p3b6c-bun-whole-runtime-local-20261025.json)은
public transitive2/peer4의 현재 Evidence/Policy57.90s와 CLI9fresh199.72s/native original3cases
135.92s를 확인했다. 각 helperwait0/Host 동일·원본/data/inode/records/empty lock 부재를
유지했다. 기존 whole engine은 native capability 이름만 강화했고 Policy/cache/transaction/
CPP는 불변이다. Workflow/Quick/security/docs·관련 네 package Race54.598s·checker Race
35.595s와 hosted native JS exact caller 검증도 PASS다. 로컬 연결만 닫혔으며 중간 commit들을
한 번 push해 exact-source hosted/timer를 확인하고 최종 지원/영향 matrix를 계속한다.

### 2026-10-26 CI413 timer와 독립 P4-A 착수

Grouped source50fd3b2/tree d6a63ab/2475를 한 번 push하여 [CI413](https://github.com/rahoney/heliopause/actions/runs/40209998926)이16:05:16UTC 시작했다. 최근53m51+추가 project519.694s와 여유를 반영한 실제75분 timer의 첫 due는17:20:16UTC/02:20:16KST다. Due와 실제 Required 결과 확인을 구분한다. [P4-A baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p4a-uv-project-baseline-20261026.md)를 확인하고 같은 M13-004에서 기존 UV pip owners를 재사용하는 frozen virtual-project input만 먼저 진행한다. P4 CLI/native resolver·persistent tools와 전체 P2/P3/P4 acceptance는 미완료다. 실행 중 source를 새 push로 취소하거나 작은 helper마다 full CI를 시작하지 않는다.

### 2026-10-26 P4-A frozen virtual project input local checkpoint

[P4-A receipt](./evidence/package-client-quarantine/p4a-uv-project-input-local-20261026.json)의 canonical Quick43.181s/security8.911s/docs0.249s와 artifact/PyPI fullRace5.379s가 PASS다. Native offline 원본 형식/lock presence·runtime/intent replay·unknown schema/source/duplicate tamper 거부를 named 검증했다. Native uv.lock reader/target export·resolver/transaction/CLI·persistent tool은 미완료다. [최종 P2/P3 matrix](./evidence/package-client-quarantine/p2p3-final-matrix-20261026.md)는 기존 local/remote 범위와 CI413/현재 공통 observer의 남은 세 CUDA qualification을 구분한다. Timer due의 실제 결과를 확인하며 M13-004 하나로 계속한다.

### 2026-10-26 P4-B native lock/export 착수 감사

[P4-B map·두 번째 검토](./evidence/package-client-quarantine/p4b-native-uv-lock-plan-20261026.md)는 P4-A input과 P1 complete wheel parser를 재사용하며 native virtual root/public graph와 target export를 결합하는 MISSING만 추가한다. 실제 UV0.13.4 빈 lock과 exact upstream serializer를 대조했으며 synthetic 구조/actual native 의미를 구분한다. Native resolver/CLI/persistent tools·whole P4 acceptance는 열려 있고 CI413 frozen50fd를 취소하지 않는다.

### 2026-10-26 P4-B native UV lock/export local checkpoint

[Reader receipt](./evidence/package-client-quarantine/p4b-native-uv-lock-local-20261026.json)는 Quick202.858s/security12.989s/docs0.255s와 artifact/PyPI fullRace5.679s, actual empty native 원본·synthetic graph/export substitution negatives를 연결한다. 기존8 owner는 bytes 불변이다. Reader의 named 경계만 닫았으며 다음 isolated native project consumer/actual public nonempty export·독립 archive binding을 같은 M13-004에서 계속한다. CI413 timer·frozen50fd 원격 범위는 그대로 유지한다.

### 2026-10-26 P4-C actual export 형식과 독립 binder 지도

고정 UV의 offline native replay2.779s는 nonempty project export의 index/sdist와 lock --check 성공을 확인했다. Synthetic input의 형식 조회이며 공개 source/검역 성공이 아니다. [P4-C audit·map·두 번째 검토](./evidence/package-client-quarantine/p4c-uv-project-export-plan-20261026.md)에 따라 실제 원본·native/manifest와 derived validation view를 분리해 기존 whole archive binder로 연결한다. P1 strict parser/runtime/Policy는 유지하고 같은 M13-004에서 MISSING만 수행한다. CI413 timer는 그대로다.

### 2026-10-26 P4-C project export binding local checkpoint

[Export receipt](./evidence/package-client-quarantine/p4c-uv-project-export-local-20261026.json)의 Quick202.258s/security12.677s/docs0.243s·artifact/PyPI fullRace5.880s가 PASS다. Actual offline native export 원본과 synthetic complete wheel/독립 metadata binding을 구분하고 actual·native·manifest·view tamper를 거부한다. Native resolver/transaction/CLI/tools는 미완료다. 기존 예방§4/§5·운영 참고§15·확장 계획§5를 대조해 [최종 matrix](./evidence/package-client-quarantine/p2p3-final-matrix-20261026.md)의 세 CUDA full은 후속 공통 동작을 모은 final source checkpoint/P7에 묶는다. 현재 CI413 타이머 결과는 그대로 소비하며 필수 profile acceptance를 미리 닫지 않는다.

### 2026-10-26 P4-D native project consumer 착수

[P4-D baseline·map·두 번째 검토](./evidence/package-client-quarantine/p4d-uv-project-resolver-plan-20261026.md)를 확인했다. 기존 fixed UV native session/typed network·observer·cleanup과 independent whole archive binder를 재사용해 먼저 sync/lock-upgrade의 isolated selection만 연결한다. Native add manifest mutation·Host atomic transaction/CLI/tools는 뒤 단계이고 selection을 설치 성공으로 표시하지 않는다. Same item의 실제 missing만 수행하며 CI413 timer/원격 후보는 유지한다.

### 2026-10-26 P4-D native project selection local checkpoint

[Selection receipt](./evidence/package-client-quarantine/p4d-uv-project-resolver-local-20261026.json)의 Quick112.172s/security14.172s/docs0.236s·affected fullRace27.681s가 PASS다. Same canonical121 actual116.337s는 project empty/single sync+lock-upgrade34.61s와 기존 UV pip79.18s/noSKIP/helperwait0/Host 동일을 확인했다. Native argv 충돌/filename/기존 public index spelling의 세 원인만 교정했고 권한/한도는 불변이다. 임시 진단은 제거했으며 version-changing delta·add/transaction/CLI/tools는 미완료다. 같은 M13-004에서 다음 missing을 계속한다. CI413은 실제02:20:16KST due에 Required10SUCCESS를 확인했고 전체 tree/원본10logs 대조를 별도 영수증으로 기록한다.

### 2026-10-26 CI413 grouped Bun qualification과 실제 timer 소비

[CI413 영수증](./evidence/package-client-quarantine/p3-grouped-bun-ci413-20261026.json)은 source50fd/tree d6a63ab/2475, 모든 checkout38e27d8와 base/source 부모의 일치·Required10SUCCESS를 확인했다. 실제75분 timer가17:20:16UTC/02:20:16KST에 조회했고 전체 경과 metadata는57m14다. Full decoded10logs1,123,173B/64chunks를 전부 readback했다. Observer121씩/Linux72top-level·Bun native9·경계5·whole2/JS정상·typed실패/CLI9/native원본3·PNPMCLI9/CPU232.73s·native macOS와 helper9wait0/final0을 확인했다. P4-A/B/C/D를 기다리는 동안 계속했고 새 push로 취소하지 않았다. 최신 P4 source와 최종 CPU·세CUDA matrix는 범위 밖이며 P7/final affected checkpoint까지 whole acceptance는 열어 둔다. 다음 native add/version-changing selection을 같은 M13-004에서 계속한다.

### 2026-10-26 P4-E add/version-changing selection 착수

[P4-E audit·map·두 번째 검토](./evidence/package-client-quarantine/p4e-uv-project-add-plan-20261026.md)는 기존 D session/controls/census/독립 binder를 재사용해 add manifest conservation과 실제 version delta만 연결한다. Exact UV0.13.4 문서와 native help를 확인했다. Host atomic transaction/CLI/tools는 미완료이며 같은 M13-004에서 작은 helper별 새 CI 없이 진행한다.

### 2026-10-26 P4-E native add/version delta local checkpoint

[Add/delta receipt](./evidence/package-client-quarantine/p4e-uv-project-add-local-20261026.json)의 Quick222.363s/security14.467s/docs0.282s·affected fullRace26.492s와 actual121.396s가 PASS다. Native add exact24.2→25.0·bare/기존 constraint 보존·두 roots와 native sync가 재검증한 old24.2에서 lock-upgrade25.0/manifest 불변을 확인했다. Seed constraint만 synthetic이며11 native selection requests/noSKIP/helperwait0/Host 동일이다. 기존11 owner는 bytes 불변이고 별도 새 CI는 없었다. Host atomic transaction/cache/records/CLI/tools·whole P4는 미완료이며 같은 M13-004에서 다음 audit를 계속한다.

### 2026-10-26 P4-F native project cache 착수

[P4-F audit·map·두 번째 검토](./evidence/package-client-quarantine/p4f-uv-project-cache-plan-20261026.md)에 따라 기존 complete Python inspection/current Evidence·Policy/cache를 재사용한다. Native original/selected provenance를 입구에서 재검증하는 adapter만 추가하며 Host atomic publication/CLI/tools는 미완료다. M13-004 하나로 계속하며 작은 helper별 새 CI는 실행하지 않는다.

### 2026-10-26 P4-F native project approved cache local checkpoint

[Cache receipt](./evidence/package-client-quarantine/p4f-uv-project-cache-local-20261026.json)은 canonical profiles/affected fullRace270.474s와 actual51.282s를 연결한다. Public two roots/explicit empty native selection부터 complete 검사/current Evidence·Policy·six-control Manifest/SBOM/cache readback과 approval 부재·Evidence 접근 실패·sealed cache 변조 거부가 PASS다. Helperwait0/Host 동일이며 Host target은 생성하지 않았다. 기존10 owner bytes 불변/새 push·CI 없음이다. 같은 M13-004에서 atomic control+venv transaction/ownership/CLI/tools를 계속한다.

### 2026-10-26 P4-G project/venv transaction 착수

[P4-G audit·map·두 번째 검토](./evidence/package-client-quarantine/p4g-uv-project-transaction-plan-20261026.md)에 따라 기존 venv complete commit/backup/journal/rollback과 descriptor project guard를 재사용한다. Exact project controls를 wheel ownership과 구분해 같은 rollback에 연결하며 실제 native ownership/CLI/lock-only/tools는 후속 acceptance다. M13-004 하나만 IN_PROGRESS이며 helper별 새 CI는 없다.

### 2026-10-26 P4-G project controls/venv filesystem checkpoint

[Filesystem receipt](./evidence/package-client-quarantine/p4g-uv-project-transaction-local-20261026.json)은 최종 canonical profiles·focused/full promotion Race를 연결한다. 정상/empty·기존 lock 부재/존재의 모든 failure 원복과 source drift·동시 guard·uncertain private data/recovery 보존이 PASS다. Fixture 부재 판정 오류와 새 wrapper의 late cleanup 순서를 구분했고 직접 RED의 동일 test source 교정 GREEN을 확인했다. Project controls는 wheel ownership에 포함하지 않고 같은 journal/rollback에 묶으며 cleanup도 backup 폐기 전 처리한다. 기존10 owner는 불변이다. Native publication guard/ownership/actual/CLI/lock-only/tools는 미완료이며 같은 M13-004에서 계속한다. 새 push·CI는 없다.

### 2026-10-26 P4-H native project publication 착수

[P4-H audit·map·두 번째 검토](./evidence/package-client-quarantine/p4h-uv-project-publication-plan-20261026.md)에 따라 기존 venv original snapshot/managed ownership·native selection/current inspection·Policy/cache와 G transaction을 하나의 held lifecycle로 연결한다. Runtime provisioning 없이 기존 venv를 대상으로 하고 unmanaged 원본은 거부한다. Native adoption/CLI/lock-only/tools는 별도 acceptance이며 같은 M13-004에서 진행한다. 새 push·CI는 없다.

### 2026-10-26 P4-H native project installed backend checkpoint

[Backend receipt](./evidence/package-client-quarantine/p4h-uv-project-publication-local-20261026.json)은 Quick292.784s/security15.138s/docs0.249s·focused/full affected Race25.544/283.054s와 actual318.313s를 연결한다. Project add24.2→25.0/same-version sync/two roots/edited-empty140.47s와 기존 UV pip sync/native original 인계175.04s가 모두 PASS다. Original stale public graph와 selected/current strict equality를 분리했고 새 default composition 계약도 확인했다. Whole inspection/current Evidence·Policy·승인 기록/원자 control+venv 반영/noSKIP/helperwait0/Host 동일이며 기존11 owner는 불변이다. Public CLI/native project adoption/lock-only/tools는 미완료이고 같은 M13-004에서 uv add/sync CLI를 계속한다. 새 push·CI는 없다.

### 2026-10-26 P4-I public project CLI 착수

[P4-I audit·map·두 번째 검토](./evidence/package-client-quarantine/p4i-uv-project-cli-plan-20261026.md)에 따라 기존 descriptor/preflight/presentation과 qualified H backend를 연결한다. `--project`/현재 디렉터리와 명시 target 또는 project/.venv의 기존 venv를 사용하며 source/option/alias를 runtime 준비 전에 거부한다. Native adoption/lock-only/tools와 전체 P4는 후속 acceptance이고 같은 M13-004에서 계속한다. 새 push·CI는 없다.

### 2026-10-26 P4-I public UV project CLI checkpoint

[CLI receipt](./evidence/package-client-quarantine/p4i-uv-project-cli-local-20261026.json)의 Quick43.517s/security8.958s/docs0.251s·full CLI/Bootstrap Race23.232s와 actual327.758s가 PASS다. New CLI168.49s는 add24.2→25.0/same-version/two roots/human cwd/default venv/edited-empty/no-op의 일곱 성공과 unowned 거부를 확인했고 기존 UV pip/native 인계156.66s도 PASS다. 별도 explicit target10.494s는 default .venv 부재/다른 VIRTUAL_ENV에서도 정확한 target과 Manifest를 확인했다. Same product source/noSKIP/helperwait0/Host 동일이며 기존18 owner는 불변이다. Native adoption/lock-only/tools와 전체 P4/remote는 미완료이며 같은 M13-004에서 다음 missing을 계속한다. 새 push·CI는 없다.

### 2026-10-26 P4-J native project adoption 착수

[P4-J audit·map·두 번째 검토](./evidence/package-client-quarantine/p4j-uv-project-adoption-plan-20261026.md)에 따라 기존 native UV complete ownership/source proof를 project guard에 연결한다. Native six controls는 유지하고 인증한 controller proof를 별도 immutable record로 결합하며 current 검사/Policy/원자 반영을 재실행한다. Core/runtime/observer 권한·한도는 불변이다. 같은 M13-004에서 진행하며 lock-only/tools·whole qualification은 미완료다. 완료 F/H/I 사본 다섯 개2,153,572,640 logicalB는 원본 hash/전체 proc 무참조·query errors0 뒤 정리했고 현재 environment와 SDK/cache/source/observer/supplies·원본은 보존했다. 새 push·CI는 없다.

### 2026-10-26 P4-J native project adoption checkpoint

[Adoption receipt](./evidence/package-client-quarantine/p4j-uv-project-adoption-local-20261026.json)은 최종 canonical Quick59.770s/security9.813s/docs0.258s와 actual native add/sync129.885s·기존 UV pip184.466s를 연결한다. 기존 complete ownership/digest binder의 mapped body 동일성을 확인했고 unselected native original까지 authenticate한 뒤 current 전체 검사/Policy/원자 반영을 수행했다. Default 거부 무변경·seven controls/current records/user file 보존/noSKIP/helperwait0/Host 동일이다. 최초 full Race의 promotion 등 다섯 package는 PASS였고 기존 readiness fixture 경쟁만 실패했다. [독립 test-only 교정](./evidence/package-client-quarantine/p4j-readiness-race-correction-20261026.md) 뒤 전체 sandbox Race15.860s가 PASS이며 첫 실패는 보존한다. Whole P4/remote는 미완료이고 같은 M13-004에서 distinct lock-only를 계속한다. 새 push·CI는 없다.

### 2026-10-26 P4-K distinct lock-only 착수

[P4-K audit·map·두 번째 검토](./evidence/package-client-quarantine/p4k-uv-project-lock-only-plan-20261026.md)는 qualified native lock-upgrade와 complete 검사/cache를 재사용하고 project-controls 전용 context/원본 guard/lock 파일만의 publication·CLI를 연결한다. 기존 venv transaction을 가짜 plan으로 호출하거나 lock 반영을 설치 성공으로 표시하지 않는다. K1 mode/typed binding부터 시작하며 전체 public lock-only/tools/final profiles는 미완료다. M13-004 하나만 IN_PROGRESS이고 새 push·CI는 없다.

### 2026-10-26 P4-K1 control-only typed component checkpoint

[K1 receipt](./evidence/package-client-quarantine/p4k1-uv-project-control-mode-local-20261026.json)은 canonical Quick232.317s/security16.135s/docs0.256s·full affected Race250.827s PASS를 연결한다. Distinct project-controls context를 native lock-upgrade/독립 wheel 인증·전체 inspection/cache에 결합하고 venv approval·잘못된 intent 전용을 거부한다. 최초 두 fixture의 원본 lock 누락은 원본을 보존하고 교정했다. Public lock publication/CLI/actual과 whole P4는 미완료이며 같은 M13-004에서 K2 guard/transaction/workflow를 계속한다. Runtime/observer/권한·한도와 기존 installed 준비 경계는 불변이고 새 push·CI는 없다.

### 2026-10-26 P4-K2 lock-only approved publication checkpoint

[K2 receipt](./evidence/package-client-quarantine/p4k2-uv-project-lock-publication-local-20261026.json)은 canonical Quick168.957s/security11.164s/docs0.258s·focused Race11.848s/full affected Race146.556s와 actual173.014s PASS를 연결한다. Actual 기존 installed5와 control-only3은 현재 전체 검사/Evidence·Policy·승인 cache/readback을 재사용하고 distinct project context로 uv.lock만 반영했다. Manifest identity/mode·user files·임의 venv는 보존하고 venv 부재에서도 동작한다. 네 ENOSPC 경계 원복/foreign 원본·backup 보존/동시 guard·recovery 재진입 거부와 noSKIP/helperwait0/Host 동일도 확인했다. Public CLI/whole P4/remote는 미완료이며 같은 M13-004에서 K3 명령·별도 no-install 결과를 연결한다. 새 push·CI는 없다.

### 2026-10-26 P4-K3 public lock-only CLI checkpoint

[K3 receipt](./evidence/package-client-quarantine/p4k3-uv-project-lock-cli-local-20261026.json)은 canonical Quick71.547s/security10.488s/docs0.253s·full CLI/Bootstrap Race29.145s와 actual252.100s PASS를 연결한다. Existing CLI169.25s의 일곱 성공/무단 인계 거부와 새 lock-only80.15s의 JSON·human/default/explicit project/current no-op·empty 네 요청이 PASS다. Lock update 완료와 installation NOT_REQUESTED를 분리하고 venv 부재에서도 동작하며 임의/active venv와 manifest identity·lock mode·user files를 보존한다. 공통 facts/render/preflight mapped body 동일성과 unsupported/unsafe original·완료 기록 부재 거부도 확인했다. NoSKIP/helperwait0/Host 동일이며 runtime/observer/K2 backend는 불변이다. Whole P4/remote는 미완료이고 같은 M13-004에서 persistent tools baseline을 계속한다. 새 push·CI는 없다.

### 2026-10-26 P4-L persistent tool baseline 착수

[P4-L baseline·수정 지도·두 번째 검토](./evidence/package-client-quarantine/p4l-persistent-tool-baseline-20261026.md)에서 기존 UV/PyPI/venv/current inspection/cache를 재사용하고 native tool metadata·exposed scripts·runtime 관계만 missing으로 판정했다. Official pipx release/PyPI1.18.4 후보는 discovery이며 새 pin·admission·지원이 아니다. Exact source/배포물 census와 별도 guest-native layout/interpreter lifecycle을 확인한 뒤 현재 gap만 구현한다. Runtime provisioning/global target/ambient backend·일반 trusted-control 예외는 추가하지 않는다. M13-004 하나만 IN_PROGRESS이고 새 push·CI는 없다.

### 2026-10-26 P4-L1 native persistent tool receipt

[Baseline·native facts](./evidence/package-client-quarantine/p4l-persistent-tool-baseline-20261026.md)와
[L1 local receipt](./evidence/package-client-quarantine/p4l1-native-tool-receipt-local-20261026.json)를
기록했다. Actual synthetic install/upgrade controls 네 개와 strict parser/layout/exposed-link claims,
UV constraint 보존·pipx native constraint 제거를 구분한다. Canonical Quick/security/docs와 PyPI
전체 Race가 PASS다. No native/Host execution·ownership·Policy authority를 parser에 주지 않는다.
Public supply/admission/whole archive binding/atomic tools/CLI는 미완료이며 다음 L2를 계속한다.
M13-004 하나만 IN_PROGRESS이고 작은 helper 단위 CI를 추가하지 않는다.

### 2026-10-26 P4-L2 whole tool wheel binding

[L2 local](./evidence/package-client-quarantine/p4l2-tool-wheel-binding-local-20261026.json)은 기존 whole
archive/source/closure engine과 native receipt의 exact main command 집합을 연결했다. Canonical
Quick/security/docs와 PyPI 전체 Race가 PASS이며 기존 nil collector 경로 본문도 대조했다.
Pipx의 selected closure startup gate만 확인됐고 UV 기존 tool environment 조회도 Python -S를
사용하지 않는 exact source 경로를 확인했다. Native query clearance/원본 ownership/public CLI
증거로 확대하지 않으며 같은 M13-004의 L3 native selection/runtime 경계를 계속한다.

### 2026-10-26 P4-L3 fresh native UV tool selection

[L3 local](./evidence/package-client-quarantine/p4l3-uv-tool-native-install-local-20261026.json)과
[baseline·최초 실패·runtime map](./evidence/package-client-quarantine/p4l-persistent-tool-baseline-20261026.md)을
갱신했다. Canonical Go checks/전체 PyPI+Sandbox Race, 새 source의 canonical observer121latches와
actual92.114s(tool15.88/기존 UV73.76)가 PASS이며 helperwait0/Host 동일이다. Native 선택과 독립
전체 archive/source/receipt만 qualified이고 Policy/Host tool 설치·public CLI는 미완료다.
다음 L4는 existing target·external command root guard와 current 승인/반입 범위만 감사한다.
M13-004 하나만 IN_PROGRESS이고 grouped/final CI 경계를 유지한다.

### 2026-10-26 P4-L4A final interpreter materializer

[L4-A receipt](./evidence/package-client-quarantine/p4l4a-tool-interpreter-materializer-local-20261026.json)은
canonical3·Promotion 전체 Race와 별도 inert recipe diagnostic을 연결한다. 기존 strict output
validator가 잡은 최초 view 위치 오류를 보존하고 separate sibling으로 교정했다. Host Python/
Artifact command는 실행하지 않았고 backend/승인·반입/CLI는 아직 미완료다. 같은 M13-004의
L4-B target/명령 root·receipt 결합을 계속하며 작은 helper별 push/CI는 추가하지 않는다.

### 2026-10-26 P4-L4B target/receipt guards

[L4-B receipt](./evidence/package-client-quarantine/p4l4b-tool-target-receipt-local-20261026.json)은
canonical3·PyPI/Promotion 전체 Race와 strict native-layout projection·two-root/interpreter/
command guard의 부정 회귀를 연결한다. Fixture 권한으로 assertion 전 실패한 최초 원본은
보존했다. Actual approval/반입/CLI는 미완료이며 같은 M13-004에서 L4-C의 receipt/노출
links를 기존 원복에 참여시킨다. 작은 helper별 push/full CI는 추가하지 않는다.

### 2026-10-26 P4-L4C fresh receipt/links atomic helper

[L4-C receipt](./evidence/package-client-quarantine/p4l4c-tool-atomic-publication-local-20261026.json)은
기존 venv engine의 optional participant·전체 Promotion Race와 fault/cancel·foreign/late cleanup
원복을 연결한다. Native receipt는 distribution ownership으로 위장하지 않으며 두 root와
journal·복구 경계를 보존한다. 최초 Close lifecycle 오류는 원본에 남겼다. 실제 tool current
approval/반입·CLI는 아직 미완료이고 같은 M13-004에서 L4-D를 계속한다. 새 push/full CI는 없다.

### 2026-10-26 P4-L4D current approved tool backend

[L4-D receipt](./evidence/package-client-quarantine/p4l4d-uv-tool-approved-install-local-20261026.json)은
canonical3·네 package 전체 Race와 actual297.347s의 tool 두 요청/기존 UV CLI 일곱 요청을
연결한다. Constraint 절 순서만 바꾸는 native output을 같은 절 집합으로 한정 비교하고 원본
intent/receipt와 source/runtime/Policy를 보존했다. 최초 actual와 wildcard fixture 실패도
보존했다. NoSKIP/helperwait0/Host 동일이며 public CLI/원본 upgrade/pipx는 아직 미완료다.
같은 M13-004의 L4-E 공개 명령을 계속하고 grouped end-to-end CI 경계를 유지한다.
