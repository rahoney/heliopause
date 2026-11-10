# Current Work Queue §1 현재 상태 이력 (2026-10-31 이전 checkpoint)

2026-11-10 R0-4 문서 정리로 `02-current-work-queue.md`의 원본 줄 125-3070을 그대로 옮겼다. 상대 링크 경로만 `archive/` 깊이에 맞게 조정했고 본문은 바꾸지 않았다. 당시 기록이며 현재 상태 판정이 아니다. 현재 상태는 [Queue](../02-current-work-queue.md)를 따른다.

### 2026-10-31 G4 기능 acceptance·CI446 완료

[Qualification](../evidence/package-client-quarantine/ci446-bun-declared-entry-hosted-qualification-20261031.json)은 sourcee3b2b70/tree0cdaeaa2의 전체10SUCCESS·실제checkoute757cc1b/tree동일·Linux126 unique PASS/noSKIP/CPU192.40s를 연결한다. 새 Bun 선언/TS 앱2/25.34s·CLI3/39.35s와 기존 Node/Bun·다른 소비자, helper18wait0/최종4original0·정리/복원을 확인했다. 실제20분GET5/04:06:10UTC에서 종료를 소비했고 worker909486/session81628의 자연exit0/nextnull을 직접 관측했다. 원본10logs1331441B는 기존8개를 재사용하고 마지막2개만 새로 확보했다. 서버 전체87m46s이며 과거 지연 원인을 수리했다고 주장하지 않는다. 원래 bounded 기능 acceptance는COMPLETE다. M13-004전체는 이름/publisher binding·채널 구현/검증·새 signed 배포가 남아IN_PROGRESS이며, 문서를 모아 정리한 뒤 실제 이름 결정을 진행한다.

### 2026-10-31 G4 Bun 선언/TS 기능 묶음 CI446 등록

[등록](../evidence/package-client-quarantine/ci446-bun-declared-entry-registration-20261031.json)은 sourcee3b2b70/tree0cdaeaa2/3252files·검증19source의 committed bytes와 normal FF ec1→e3, 실제run41159991137/02:26:06UTC 시작을 연결한다. [로컬 결과](../evidence/package-client-quarantine/p6g4-bun-declared-entry-local-20261031.json)의 public새앱2/CLI3·기존Node2, 최종Quick/Race/tagged/docs/security는PASS다. 기존Bun run cohort에두parent만연결해126/CPP133/Required10/Linux150분을기대하며권한/제품한도는불변이다. 실제20분GET worker909486/UID1000/start8920720/session81628을확인했고첫due는02:46:06UTC/11:46:06KST다. 등록시actualGET0이며CI결과/checkout원본/최종정리는PENDING이다. 새push없이최초실패또는완료결과를소비한다. 이름/배포는기능acceptance뒤다.

### 2026-10-31 H1/H2·승인된 Linux150분 전체 묶음 CI442 등록

[등록 영수증](../evidence/package-client-quarantine/ci442-registration-20261031.json)은 sourceec1c260/tree84e70a9e와 정상 FF1ee→ec1·검증 source51의 committed bytes 일치를 확인한다. 실제 run41094467995는16:57:01UTC 시작이며 기존20분 GET worker75370/PID799301의 첫 due는17:17:01UTC(한국02:17:01)다. 등록 시 timed queries0이며 완료 조회로 쓰지 않는다. 초기9jobs/Docs SUCCESS이고 Required10·ordinary124/CPP133·승인된 전체150분/개별 한도·보안 경계는 유지한다. H1/H2 named local은 통과했으나 새 hosted/최종 네 profile·whole acceptance는 PENDING이다. 실행 중 새 push 없이 같은 P7의 최종 입력·지원표 선행조건을 확인한다.

### 2026-10-30 P7 grouped CI441 등록

[등록](../evidence/package-client-quarantine/ci441-registration-20261030.json)은 source1ee3788/treea5eedf7a·일반 FF4commits/42paths와 실제 run41060068858/start13:03:20UTC/22:03:20KST·초기9jobs를 대조했다. 기존20분 공개 GET worker58611/PID657507/UID1000·start4109728은 첫13:23:20UTC/22:23:20KST를 등록했고 actual_queries0으로 미래 조회 완료를 주장하지 않는다. CI439·새 G1/G2 hosted·G3 local 합계 약114m36은 추정뿐이다. Ordinary122/CPP133/Required10/Linux120min과 제품 한도를 유지하며 source는 고정한다. [P7 local](../evidence/package-client-quarantine/p7-cuda-profile-routing-local-20261030.json)의 Quick63.797s/checker Race84.552s·strict freshness·docs/security는 PASS다. 현재 원격 source의 ordinary/CPU와 세 CUDA·지원표/whole final은 미완료이며 대기 중 독립 문서/acceptance 작업을 계속한다.

### 2026-10-30 P6-G3 명시 Bun JS 실행 named local 검증

[G3 결과](../evidence/package-client-quarantine/p6g3-bun-readonly-program-local-20261030.json)는 기존 whole 검사/현재 Policy·기록/readonly input/capture와 보호 Bun 공급을 재사용한 앱2/65.68s·forced CLI3/177.83s, controlled first9+selected remainder3/73.39s의 12종 coverage와 기존 Node 앱2/64.11s를 연결한다. 첫 comm-spoof parent FAIL은 ARTIFACT/direct-root UNKNOWN의 안전한 거부를 BUN 신원 유지 oracle로 판정한 test 문제였고 제품 권한을 늘리지 않았다. 단일 full12 parent PASS로 쓰지 않는다. 같은 observer09615d38의 전체CPP133/59.054s와 최종25 source Root/private 동일·Quick51.429s/영향3 Race53.679s/checker Race77.532s/tagged 고정 Staticcheck2.832s가 PASS다. 첫 Quick의 C++ 입력 시점 불일치 원본을 보존하고 최종 입력에서 성공 cache를 재사용했다. 기존 Bun supply/helper/job에 새3parents를 연결해 미래 expected122/CPP133/Required10/Linux120min을 유지한다. 완료 사본120names/118inodes 약6.36GiB 논리 용량은 ref0/query errors0/helper0/container0 뒤 정리했고 원본·현재 observer·logs/cache는 보존했다. 새-source hosted·Bun-only entry 자동 선택/일반 동등성·P7/final은 OPEN이다.

### 2026-10-30 CI440 Bun cold/current 앱·Node CLI 전체 묶음 등록

[등록](../evidence/package-client-quarantine/ci440-registration-20261030.json)은 source6d9d3bf/tree dad13b09·일반 FF2commits/56paths·remote ref와 actual run41048941282/start11:33:38UTC/20:33:38KST·초기9jobs를 확인했다. 기존20분 공개 GET worker18946/PID539348을 실행해 첫11:53:38UTC/20:53:38KST를 등록했으며 actual_queries0으로 미래 조회 완료를 주장하지 않는다. CI4396254s와 새 cold/app/CLI/refusal 실제 합계로 약109m26을 추정하되 완료 보장은 아니다. 기존 helper/job 안에 새 cold1/current3parents를 연결해 expected119/CPP132/Required10/Linux120min을 유지한다. 대기 중 G3의 명시 readonly Bun 프로그램 gap을 계속하며, 이 후보로 bare 전체/general/hostile/final 지원을 선언하지 않는다.

### 2026-10-30 P6-G2 Bun 현재 앱·Node CLI named local 검증

[결과](../evidence/package-client-quarantine/p6g2-bun-current-cli-local-20261030.json)는 native 모든 placement/edge와 독립 원본을 설치 snapshot 없이 기존 whole 검사/Policy/records·readonly Node에 연결한 앱2/60.97s·CLI3/190.91s·거부2/50.00s를 확인했다. Implicit Rimraf12nodes/13edges 전체를 검사하며 매 요청에 새 승인/기록을 만들었다. Offline 첫 kernel openat2 성공 O_PATH/pending0를 표적 RED로 재현하고 exact14argv/current-root/project workspace만 계상하는 최소 수정으로 통과했다. Host5/Docker·helperwait0/설정 같은 inode 원복이다. 동일 observer57c186ea 전체CPP132/50.004s·Quick351.507s/영향5package Race92.964s/checker Race72.786s/tagged 고정 Staticcheck2.874s/docs0.342s/security18.055s가 PASS다. 새 cold1/current3parents를 기존 Bun 공급·helper에 연결하고 저장한 실제 로그로 empty/duplicate/SKIP 거부를 확인했다. 새 job/rebind 없이 expected119/Required10/CPP132/Linux120min과 기존 제품 한도를 유지한다. 명시 Bun program·새-source hosted·일반 동등성/hostile/P7/final은 OPEN이며 G3는 source 전 baseline만 기록했다.

### 2026-10-30 CI439 PNPM native/current 앱·CLI hosted 완료

[Qualification](../evidence/package-client-quarantine/ci439-pnpm-current-hosted-qualification-20261030.json)은 frozen0eeefca/tree5b7a0a21/all10checkout63f66bdf·Required10SUCCESS와 Linux115/CPP132를 확인했다. 새 PNPM cold6.59s·앱2/41.15s·CLI3/102.59s(implicit Rimraf12nodes/13edges)·거부2/35.94s 및 helper18wait0/final4original0·복원이다. 실제20분 GET6 뒤 worker68297은 자연0/next none이다. Completed8 logs는 재조회하지 않고 마지막2개만 추가해 decoded1350712B를 보존했다. 서버 경과6254s이며 later G1/G2 Bun/current·일반 동등성/whole final은 source 밖이다. G2 로컬 연결을 계속하고 작은 변경별 새 CI는 만들지 않는다.

### 2026-10-30 CI439 PNPM 전체 묶음 등록

[등록](../evidence/package-client-quarantine/ci439-registration-20261030.json)은 source0eeefca/tree5b7a0a21의 일반 FF3commits/49paths·remote ref와 실제 run41030454824/09:01:54UTC·초기9jobs를 확인했다. Worker68297/PID424488을 실행해 첫 실제20분 GET을09:21:54UTC/18:21:54KST로 등록했다. 실제 poll1은09:21:55UTC에8SUCCESS/Linux 진행을 확인했고 다음09:41:55UTC를 등록했다. 전체 완료는 아직 PENDING이다. CI438106m23s와 새 cold/app/CLI/refusal 실제 값을 합친 약111m03은 추정이다. Expected115parents/CPP132/Required10/Linux120분과 제품 경계는 유지한다. 대기 중 P6-G Bun native/current baseline을 계속하고 작은 변경을 push하지 않는다. 새-source hosted/일반 동등성/whole final은 OPEN이다.

### 2026-10-30 P6-F2/F3 PNPM 현재 실행·CLI named local 검증

[결과](../evidence/package-client-quarantine/p6f2f3-pnpm-current-cli-local-20261030.json)는 native instance와 독립 원본·기존 whole 검사/Policy/records·Node runtime을 재사용한 앱2/67.72s와 세 CLI alias151.83s/entry 거부2/49.60s를 확인했다. Implicit Rimraf는12nodes/13edges 전체 ALLOW 뒤 실행했으며 거부는 runtime/새 Manifest가 없다. 첫 native layout 실패는 정확한 빈 private-hoist 경로의 부재 조건이었고, 같은 helper RED/unsafe5 denial 뒤 최소 수정으로 실제 후속을 통과했다. Host5/Docker 보존/helperwait0/원복이다. Quick330.692s/관련5package Race90.770s/checker Race78.012s/workflow2.991s/tagged 고정 Staticcheck17.468s PASS이며 docs0.356s/security17.432s도 PASS다. Existing PNPM 공급물·현재 bootstrap helper 위치로 새115parents를 연결해 중복 rebind/job을 추가하지 않았다. 새로운 hosted·Bun/일반 동등성/hostile/whole final은 OPEN이다.

### 2026-10-30 CI438 pipx native/current 앱·CLI hosted 완료

[Qualification](../evidence/package-client-quarantine/ci438-pipx-current-hosted-qualification-20261030.json)은 frozen31acbfe/tree5b40eaf8/all10checkout7beb11ba·Required10SUCCESS와 전체decoded1369730B를 확인했다. Linux111/CPP132·새 pipx 앱3/85.43s/CLI6/195.72s와 기존 UV 앱87.89s/CLI150.64s·helper18wait0/final4original0·복원이다. 실제20분 GET6회 뒤 worker5733은 자연0/next none으로 종료했다. Completed8logs는 다시 받지 않고 마지막2개만 확보했다. 서버 경과106m23s이며 later F1/F2와 whole final은 해당 source 밖이다. 같은 M13-004의 현재 F2 첫 layout 불일치를 계속 조사하며 작은 변경별 새 CI를 만들지 않는다.

### 2026-10-30 P6-F1 PNPM cold 선택 DATA named local 완료

[F1 결과](../evidence/package-client-quarantine/p6f1-pnpm-cold-selection-local-20261030.json)는 기존 native scope/lock·독립 SRI owner로 fresh cold DATA169와 root1/package1/instance1/source1을 확인했다. 최종17argv는 sealed mount의 고정 missing command를 요청하며 artifact bin/추가 Node를 실행하지 않는다. 같은 Go/공급물의 기존 observer 비교10.67s가 PASS여서 실험한 C++ null-write 예외와 공용 tar reader 확장은 제거했다. Quick288.888s/영향 전체 Race54.282s/tagged vet·고정 Staticcheck/docs0.353s/security17.878s가 PASS이며 helperwait0/Host 보호5·Docker 보존/profile 같은 inode 원복이다. 잘못 가정한 absolute alias와 recorder·option·cwd 최초 실패는 원본에 보존했다. Current Policy/program/CLI·일반 동등성/hosted/whole final은 OPEN이며 다음 F2 원본·peer graph·entry 연결을 계속한다. CI438은 frozen31acbfe이며 이 단위로 push하지 않는다.

### 2026-10-29 CI433 current Python 경계·앱 hosted 완료

[Qualification](../evidence/package-client-quarantine/ci433-python-current-workflows-hosted-qualification-20261029.json)은 frozen source87c8/전체tree238aa647/runtime1368과 all10 checkout980e3509·Required10SUCCESS를 확인했다. Linux104/CPP132/fullCPU290.07s, 새 controlled8/137.63s·hostile5/70.94s·public 앱3/77.49s 및 helper17wait0/final4 original0이다. 실제20분조회5회 뒤 완료를 직접 확인해 전체decoded1317349B/59chunks를 보존했고 exact 소유 워커만 SIGTERM/session143으로 종료했다. 다음 타이머는 없다. 이후 D1/D2/D3 native/오류 보존/캐시 경계는 이 source 밖이며 native public golden/aliases/CLI/final과 과거 CPU/drop 인과는 OPEN이다.

### 2026-10-29 CI434 common observer checkpoint 등록

[Registration](../evidence/package-client-quarantine/ci434-registration-20261029.json)은 d81c444/tree4497e3c7/3074·1370의 정상 FF87c8→d81과 remote ref·actual run40890702427/start14:21:47UTC를 연결한다. Actual20min public GET worker12754/PID3270701은 첫14:41:47UTC(한국23:41:47)를 등록했고 즉시 run/jobs HTTP200 조회를 확인했다. 직전CI4336368s를 따른 약106분은 추정이며 기존104parents/CPP132/Linux120min은 유지한다. D1 native DATA parent는 로컬만 실행했고 hosted filter에 추가하지 않았다. Current first-error/CPP와 기존 소비자 hosted 결과는 PENDING이다. 실행 중 새 push 없이 같은 D3 outside7 test-only 진단을 계속하며 acceptance/public native/aliases/final은 OPEN이다.

### 2026-10-30 CI438 E2/E3 전체 묶음 등록

[등록](../evidence/package-client-quarantine/ci438-registration-20261030.json)은 source31acbfe/tree5b40eaf8·일반 fast-forward4commits/48paths와 초기9jobs를 확인했다. 05:58:50UTC/14:58:50KST 시작이며 실제20분 공개 GET worker5733/PID330825를 실행해 첫 due06:18:50UTC/15:18:50KST를 등록했다. 등록 시 actual_queries0으로 향후 조회 완료를 주장하지 않는다. CI437111m23s+새 로컬 앱143.10s/CLI283.94s를 기준으로 약118m30s를 예상하지만 완료 보장은 아니다. Required10/CPP132/expected111parents·기존120분 job/제품 한도는 유지한다. 새-source hosted와 whole final은 PENDING이며 실행 중 작은 변경을 push하지 않고 같은 M13-004의 독립 P6-F baseline을 계속한다.

### 2026-10-30 P6-E3 pipx current 앱·CLI named local 완료

[결과](../evidence/package-client-quarantine/p6e3-pipx-current-cli-local-20261030.json)는 기존 native session/whole 원본·current inspection/Policy/runtime/records와 CLI presentation owners를 재사용한 앱3/143.10s·pipx CLI6/283.94s·기존 UV CLI5/217.40s를 확인했다. 기본 sole-console 요청을 explicit command로 전달한 최초 실패와 mode fallback/테스트 기록 oracle 오류는 원본과 함께 보존했다. 최종 operator503.702s는 exit0/helperwait0/noSKIP/Host 보호5·Docker 보존·설정 원복이다. Canonical Quick81.012s/workflow/checker Race73.525s·영향5package Race86.008s와 final tagged checks가 PASS이며 generic console/Core/CPP·제품 한도를 바꾸지 않았다. Existing acquired42/private installed fixture 한 번으로 앱3/CLI6를 연결했고 새 job 없이 exact parent/unique case/no-SKIP guard를 유지한다. 새 source grouped hosted와 남은 aliases/일반 동등성/profile/whole final은 OPEN이다. 다음은 이 E2/E3 전체 묶음 CI와 같은 M13-004의 독립 P6-F baseline이다.

### 2026-10-30 CI437 공통 FD·새 UV hosted 완료

[Qualification](../evidence/package-client-quarantine/ci437-close-entry-uv-hosted-qualification-20261030.json)은 frozen f4f7abc/tree99e558c6/all10checkoute5a153ef·Required10SUCCESS와 Linux109/CPP132/fullCPU285.18s를 확인했다. 기존 Bun peer64.88s와 새 UV source resolver39.67s/current application94.97s/CLI163.19s가 PASS이며 helper18wait0/final4original0·보호 입력 복원을 확인했다. 실제20분 GET5회 뒤 예상 완료 시 MCP로 종료를 직접 확인해 마지막2logs만 새로 받아 전체1369998B readback/source proof를 보존했다. 정확한 소유 GET worker만 SIGTERM/session143으로 종료했고 다음 타이머는 없다. CI436의 역사적 이벤트 순서는 미확정이며 후속 E2 source/native local 검증과 현재 pipx app/CLI/일반 동등성/whole final은 이 source 밖이다.

### 2026-10-30 P6-E2-B pipx native-run resolver local 검증

[한정 결과](../evidence/package-client-quarantine/p6e2b-pipx-run-resolver-local-20261030.json)는 기존 session/check/defer와 whole original/control/payload/console 결합을 실제 public pipx 소비자에 연결했다. UV descriptor 역변환 bytes 동일·pipx 고유 identity·기본 Quick282.759s/전체 Sandbox Race65.573s·최종 tagged 검사가 PASS다. 기존 install/accounting/new frozen3사례 뒤 첫 public actual의 recorder phase oracle 실패를 보존하고 그 test-only 조건만 교정했다. Public pipx 정상/불완전 거부52.425s·영향 기존 UV50.264s는 실제 public source와 동일 원본 결합/graph0 거부·native Outside/network0/Host 보호5와 Docker 보존·helperwait0/설정 원상 복원을 확인했다. CPP와 기본 소스가 불변인 검사·성공3사례를 반복하지 않았다. Native cache/pipx_metadata는 unapproved DATA이며 현재 Policy/승인된 program 실행/app·CLI/일반 동등성/새-source hosted와 whole final은 OPEN이다. CI437/f4를 유지하고 새 push하지 않는다.

### 2026-10-30 P6-E2-A 고정 pipx run 준비 ABI local 검증

[한정 검증](../evidence/package-client-quarantine/p6e2a-pipx-run-preparation-local-20261030.json)은 최초 run root 거부와 기존 clone 연결 누락을 보존하고, 같은 root/creator/UV/query owner만 확장했다. Fixed17argv·whole entry/script 충돌 거부·임시 cache15/current image/parent/readonly negative와 C++132/48.917s, tagged compiler/vet/Staticcheck가 PASS다. 실제 기존42input/synthetic graph2의 준비13.142s가 inert terminal/관측 완료·자원각1/CPU2572031us/helperwait0/정리를 확인했다. Host close-enter profile을 현재 observer와 맞춘 뒤 helpers/containers/query 부재에서 같은 inode의 원래 bytes로 복원했고 보호5/Docker가 전후 동일하다. Census83은 아직 unapproved DATA이며 전체 원본/console binding·public resolver/current app/CLI·새-source hosted는 OPEN이다. CI437 sourcef4는 고정하고 새 push하지 않는다.

### 2026-10-30 공통 FD 수정 CI437 등록

[등록 영수증](../evidence/package-client-quarantine/ci437-registration-20261030.json)은 f4f7abc/tree99e558c6의 정상 FF와 실제 run40992454633/02:34:15UTC 시작을 확인했다. 기존20분 공개 GET worker85924/PID52695의 첫 due는02:54:15UTC(한국11:54:15)다. 조회 등록과 실제 조회를 구분하며 현재 결과는 PENDING이다. C++132와 canonical Quick/security/docs는 로컬 PASS이며 matching Bun·기존 소비자/새 UV hosted는 아직 OPEN이다. 실행 중 새 push 없이 같은 item의 E2 기존 owner·안전한 native query 경계 검토를 계속한다.

### 2026-10-30 재시작 준비·CI436 최초 실패 보존

[CI436 result](../evidence/package-client-quarantine/ci436-bun-peer-hosted-failure-20261030.json)은 source775c19c/whole treebcc64338/all10checkout01e8d632와 8SUCCESS·Linux/RequiredFAIL을 기록한다. Linux22개 뒤 기존 Bun peer의 FD_STATE_UNKNOWN이 최초 실패이며 새 UV hosted 사례는 미도달이다. CPP/Bun source는 이전 green source와 동일하지만 인과는 미확정이다. 타이머는 실제2회 후 중단돼 완료 확인을 놓쳤고 현재 PID는 없다. 실패를 수정하거나 CI를 재실행하지 않고 공간 정리·외부 재시작 checkpoint를 먼저 만든다. 재생성 가능한 Go build cache 약105.54GiB를 pinned Go clean -cache로 회수했고 source/logs/모듈·고정 도구/원본 입력은 보존했다. Windows C 물리 반환은 offline compaction 결과로만 확인한다.

재개 후 F checkpoint/result는 실제 압축 완료를 기록했다. 검증된 원본 archive에서 누락 입력을 복구해1443개 해시를 대조했고 실제 Host의 보호 runtime5·활성 helper0/container0/query errors0을 확인했다. 격리 shell의65534 소유자 표시는 실제 Host 변경이 아니었다. 현재 source를 고치기 전에 [기존 FD 조사](../evidence/package-client-quarantine/ci415-pnpm-fd-investigation-20261026.md)의 kernel close/FD 재사용 순서와 CI436의 SOCKET16→CLOSE16→SENDTO 진단을 좁혀 대조한다. 동일 현상으로 보이는 것과 실제 원인 확정은 구분하며 full/native/CI를 무수정 반복하지 않는다.

### 2026-10-30 E1 frozen original 공유 local 검증

[E1 receipt](../evidence/package-client-quarantine/p6e1-original-sharing-local-20261030.json)은 기존 UV 원본/native body를 즉시 기존 public caller가 쓰는 두 private 함수로 분리한 source1/current1392의 canonical8/Quick131.983s/Sandbox Race27.14s를 확인한다. 역변환된 두 body는 기존 bytes와 동일하고 validation/context/defer 의미를 유지한다. Native/CPP actual·ELF·CI를 반복하지 않았다. [Pipx baseline](../evidence/package-client-quarantine/p6e-pipx-run-baseline-20261030.md)은 고정 ZIP source의 UV 위임/임시 venv 분기·pipx.run 우선순위·exportable home/.cache를 분리하며 native pipx run/현재 app/CLI는 아직 미구현이다. Frozen CI436은 변경하지 않고 E2의 whole temporary binding/필요한 scoped ABI를 계속 감사한다.

### 2026-10-30 grouped CI436 등록·실제20분 조회

[Registration](../evidence/package-client-quarantine/ci436-registration-20261030.json)은 D5/D6/D7/D8를 모은 source775c19c/treebcc64338/3120·1392의 normal FFb0→775와 원격 ref·actual run40945168003/start20:00:39UTC를 확인했다. 워커52419/PID3546388는 첫20:20:39UTC(한국05:20:39)부터20분마다 실제 public run/jobs GET/원본/다음 due를 저장하고 즉시 공개 HTTP200·initial9jobs도 확인했다. 기존106+새3=109/CPP132·UV195/Required10/Linux120min은 불변이며 약115분은 추정이다. 실행 중 새 push 없이 P6-E pipx run 기존 owner/고정 source와 gap을 감사한다. Native DATA/current Policy·선택 program은 분리하고 generic equivalence/final은 OPEN이다.

### 2026-10-30 D8 native-run hosted wiring local 완료

[D8 receipt](../evidence/package-client-quarantine/p6d8-native-run-hosted-local-20261030.json)은 기존 current execution step/protected Bootstrap와 Sandbox rebind/helper에 production selector 정상·불완전1/current app3/UV CLI5를 연결한다. 기존 source/census2·106parents를 보존해 새 top109를 기대하며 Required10/CPP132·UV195/Linux120min/권한은 불변이다. Canonical8/전체 checker Race·실제 저장 로그 DATA33cases가 PASS이고 product/CPP actual은 반복하지 않았다. D5/D6/D7와 중요한 end-to-end source 한 번만 push해 hosted gap을 확인한다. Generic/final은 OPEN이다.

### 2026-10-30 CI435 native DATA hosted 완료

[Qualification](../evidence/package-client-quarantine/ci435-native-source-hosted-qualification-20261030.json)은 frozen b0/전체treeeebd72a1/all10checkout334e96e3·Required10SUCCESS와 Linux106/CPP132/UV195/CPU298.02s를 확인했다. Native source/census2 및 exact DATA3markers, helper18wait0/final4original0·복원이 PASS다. Mac production89.115s marker interval/default 통과는 과거 timeout 인과 해결이 아니다. 실제20분 GET6회 후 worker13791 자연0/next none이며 1358654B/62parts를 보존했고 기존8logs는 다시 받지 않았다. D5/D6/D7는 이 source 밖이므로 D8에 한 묶음으로 연결한다. Server108분31초이며 작은 helper별 CI나 같은 입력 재시도는 없다.

### 2026-10-30 D7 native UV CLI named local 완료

[D7 receipt](../evidence/package-client-quarantine/p6d7-uvx-cli-local-20261030.json)은 aliases/현재 whole approval·별도 package-console graph2·normal0/typed guest2/literal help/privacy·준비 거부5개191.58s와 canonical10·Quick273.554s/영향 Race58.681s를 연결한다. First --from 실패는 authenticated metadata의 괄호형 숫자 조건 연결 gap이며 whole binding RED4→GREEN과 좁은 UV adapter 수정으로 확인했다. 기존 npm4와 CPP132는 unchanged 입력으로 재사용했다. 완료 사본6개165480990B를 root ref0/errors0로 정리해 누적251ELF+6cache/약10.97GiB logical이다. 다음 D8 native/current/CLI hosted 묶음을 계속하며 CI435/sourceb0 결과 밖인 named local 범위를 whole completion으로 쓰지 않는다.

### 2026-10-30 D6 current native-run application named local 완료

[Qualification](../evidence/package-client-quarantine/p6d6-native-run-application-local-20261030.json)은 existing1384 불변/new3/current1387·canonical9/Quick72.206s/Bootstrap·Application Race8.09s와 실제 current3/115.44s를 결합한다. 각각 fresh whole Evidence/M3/M5 ALLOW 뒤 normal0/typed guest2는 complete이며 configured command 불일치는 runtime/새 기록 없이 거부됐다. 기존 binder/runtime/records와 old tool-install factory를 보존했다. Host/source/42/ZIP/Docker 동일/helperwait0/전체 정리 후 종료 사본2개55089722B를 제거해 누적245ELF+6cache/약10.81GiB 논리 용량이다. CI435/sourceb0는8SUCCESS/Linux 진행·next18:51:44UTC로 frozen이며 D5/D6는 그 밖이다. 다음 native-run CLI/alias baseline을 감사한다.

### 2026-10-30 D5 production native-run resolver named local 완료

[Qualification](../evidence/package-client-quarantine/p6d5-native-run-resolver-local-20261030.json)은 기존 session/whole freeze/RO mirror/console owner를 재사용한 ResolveToolRun을 current1384·canonical9/Quick89.765s·세 package Race32.605s와 실제 정상/incomplete 거부49.68s로 검증했다. Source/native real traces6개가 raw complete이며 offline 두 개 Outside0·정상만 selection 반환/거부 시 graph·entry0다. 원본/Host/Docker 보존/helperwait0/정리를 확인했고 사본2개31356900B를 제거해 누적243ELF+6cache/약10.76GiB 논리 용량이다. Native RECORD는 unapproved DATA이며 Policy/program/일반 동등성·CLI/final은 별도다. CI435는 oldb0/poll01의8SUCCESS(macOS89.114949s marker interval)·Linux 진행이고 D5는 그 밖이다. 다음 기존 current Python factory/runtime에 새 selector를 연결한다.

### 2026-10-30 grouped CI435 등록·실제20분 조회

[Registration](../evidence/package-client-quarantine/ci435-registration-20261030.json)은 sourceb0f4cbc/treeeebd72a1/3097·1379의 normal FFd81→b0와 remote ref·actual run40925338534/start17:51:42UTC를 대조했다. Public run/jobs HTTP200을 확인했고 worker13791/PID3398839는 첫18:11:42UTC(한국03:11:42)부터20분마다 실제 GET/원본/다음 due를 저장한다. 직전107m18s에 native 두 부모의 약55s를 더한 약108분은 추정이며 Linux120min/Required10은 그대로다. 기존104+새2/CPP132·UV195와 새 macOS 결과는 PENDING이다. 실행 중 새 push 없이 production resolver/current adapter의 기존 owner와 실제 gap을 감사한다. Native DATA 테스트의 Policy/selected program은 NOT_RUN이며 generic equivalence/aliases/CLI/final과 과거 원인은 OPEN이다.

### 2026-10-30 D4B4 current native hosted wiring local 완료

[Receipt](../evidence/package-client-quarantine/p6d4b4-native-hosted-local-20261030.json)은 기존 current step의 protected sandbox rebind/helper와 closed profile에 native public 두 부모/세 DATA marker를 연결했다. Canonical8/Quick80.247s/전체 checker Race61.829s와 저장된 실제 로그 DATA26개가 PASS다. Runtime1379/기존1376·CPP132/UV195/Required10/Linux120min/권한·기존104개를 유지하고 native/CPP actual은 반복하지 않았다. 다음 B2+B3+wiring을 한 번 묶어 push하고 실제 새 run/start에20분 GET을 등록한다. Hosted/macOS와 production resolver/adapter·generic equivalence/aliases/CLI/final은 OPEN이다.

### 2026-10-30 D4B3 native source/console DATA named local 완료

[Qualification](../evidence/package-client-quarantine/p6d4b3-native-source-console-local-20261030.json)은 current1379/기존1371 불변과 canonical8/Quick240.797s·영향 Race32.458s 및 실제 공개 원본 binding24.07s를 확인한다. Native controls/payload SHA/relocatable wrapper는 source와 일치했고 trace complete/Outside0·원본/Host 보존/helperwait0/정리도 PASS다. Native RECORD는 unapproved DATA이며 Policy/selected program/generic native equivalence/CLI는 미완료다. CPP132/UV195는 재사용했고 종료 사본2개31244251B를 제거해 누적241ELF+6cache/약10.73GiB 논리 용량이다. 다음 current hosted wiring까지 묶어 한 번 CI를 진행하며 새 push는 아직 하지 않았다.

### 2026-10-30 D4B2 exported-cache query/census named local 완료

[Qualification](../evidence/package-client-quarantine/p6d4b2-exported-cache-query-local-20261030.json)은 fixed output cache/helper의 narrow ABI와 같은 latch RED3→GREEN132/UV195·fresh274.556s/동일 binary, current9/영향 Race24.106s를 연결한다. Native 새 census24.30s/기존 TMP23.43s는 모두 complete/Outside0이며 새 DATA84·단일 env를 수집했다. 기존 current public Python2/72.84s의 fresh ALLOW·complete0/2, Host/source/42/ZIP/Docker 보존/helperwait0/정리도 PASS다. 완료 사본4개85971665B만 root refs0/errors0 뒤 제거해 누적239ELF+cache6/약10.70GiB 논리 용량이다. Native 원본/RECORD/relocatable wrapper·선택 동등성/current adapter/aliases/CLI/final은 별도이며 다음 B3 baseline을 계속한다. 새 push/CI는 시작하지 않았다.

### 2026-10-30 CI434 완료 / whole 실패·Linux named 성공

[Result](../evidence/package-client-quarantine/ci434-common-boundary-hosted-result-20261030.json)은 frozen d81/전체tree4497/모든10checkout2e20과 8SUCCESS·macOS/RequiredFAIL을 확인했다. Linux104/CPP132/fullCPU291.43s·current Python boundary8/140.11s/hostile5/72.51s/public앱3/78.83s 및 helper17wait0/final4original0·원복은 PASS다. Mac production build120s timeout/default 미도달의 직접 원인은 미확정이며 whole qualification은 NO다. 실제20분 GET6회/서버107m18 뒤 자연 워커 exit0·다음 타이머 없음과 full decoded1357090B/62chunks를 보존했다. Completed8은 재조회하지 않고 새2만 받아 대조했다. 문서/작은 helper별 CI와 동일 실패 입력 재실행 없이 D4B local을 계속하고 다음 중요한 묶음에서 hosted gap을 검증한다. D3-E2/D4A는 source 밖이며 main/release는 진행하지 않았다.

### 2026-10-30 D4B1 unsupported DATA export 경로 종료·정리

[Checkpoint](../evidence/package-client-quarantine/p6d4b1-native-census-checkpoint-20261030.json)은 actual-v1/v2의 native 준비·complete clean trace 뒤 DATA tar exit1/미완료를 보존한다. 실제 daemon stderr 원인은 미수집이며 status-only 진단을 원인 확정으로 쓰지 않는다. 공식 Docker tmpfs cp 제약과 기존 교훈에 따라 이 경로를 거부하고 추가 동일 CI/테스트 재시도 없이 B2의 exportable volume 감사를 계속한다. 미검증 private candidate는 원본 freeze 후 제거해 qualified Root/private1371을 보존했다. 종료 사본4개62082310B만 root102proc/ref0/errors0 뒤 제거해 누적235ELF+cache6/약10.62GiB 논리 용량이다. CI434는 별도 frozen source d81이며 실제20min GET/실행 중 push 금지를 유지한다.

### 2026-10-30 D4A public pre-query binding named local 완료

[Qualification](../evidence/package-client-quarantine/p6d4a-public-native-binding-local-20261030.json)은 test-only fixture extension/runtime1371·기존1369 불변과 canonical6/기본 입력 동일성·기존 native4/새 public1을 연결한다. 독립 public wheel의 whole original/startup gate와 RO seal/readback을 metadata query 전에 적용했고 새 native30.04s는 complete/Outside0·고정 EACCES·선택 프로그램 NOT_RUN이다. Public 획득의 허용된 관찰 network와 offline native trace를 구분한다. Host/source/고정 입력/Docker 불변·helperwait0/전체 정리 뒤 완료 사본4개62042442B만 삭제해 누적231ELF+cache6/약10.57GiB 논리 용량이다. Native census/equivalence/current adapter/aliases/CLI/hosted/final은 열어 두고 D4B baseline 감사를 계속한다. CI434/source d81은 별도이며 실제20분 GET을 유지하고 실행 중 push하지 않는다.

### 2026-10-29 D3 exact readonly ABI named local 완료

[Qualification](../evidence/package-client-quarantine/p6d3-cached-query-local-20261029.json)은 E1 실제7건을 cfg32768/네 dir624640의 PYTHON·ARTIFACT 읽기로 귀속하고, 같은 새 latch의 RED3→GREEN132·fresh276.430s/132·UV120 및 affected sandbox Race23.172s를 확인했다. Native4.56s는 complete/Outside0, 기존 public Python2/68.28s는 fresh ALLOW·complete0/2와 모든 정리가 PASS다. Test-only log overlay는 제거했다. 완료 ELF227개와 전용 cache6의 누적 논리 약10.51GiB를 정리하고 원본·현재 재사용 cache/고정 입력/CI434 monitor를 유지했다. Named D3만 닫으며 다음 D4의 public pre-query whole startup/source binding/golden은 감사 전이다. CI434는 old source d81의 Mac build2mFAIL 원본을 보존했고 Linux 진행·20분 GET을 계속한다.

### 2026-10-29 D3 cached query 부분 checkpoint·정리

[Checkpoint](../evidence/package-client-quarantine/p6d3-cached-query-checkpoint-20261029.json)는 CPP2/runtime1370·fresh315.655s/132/UV120과 canonical9·affected sandbox Race22.668s, native DATA4.66s complete/Go0·current public Python2/73.35s를 연결한다. Native outside7은 대상/분류 미확정이므로 QUALIFIED=PARTIAL/ACCEPTANCE_CLOSED=NO를 유지한다. Failed inputs/ELFs와 모든 실제 로그를 동결했고 source25452개를 보존한 완료 캐시6개 및 root83proc/ref0/errors0 뒤 완료 ELF4개를 삭제했다. 누적221개+cache6/~10.40GiB Linux 논리 용량이다. 다음 동일 D3의 bounded test-only outside 진단을 진행한다.

### 2026-10-29 P6-D3 고정 cached UV query 경계 착수

[Baseline·exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6d3-cached-query-plan-20261029.md)는 기존 UV owner와 named CPP latch를 재사용해 fresh TMP/archive-v0 ID16 query만 고정 image·exact creator·sealed topology·ARTIFACT/readonly helper/env 범위로 다룬다. 기존1370 membership/Go/Core/CLI/wire/pins/한도는 유지하고 CPP2만 변경한다. Canonical CPP RED→GREEN와 전체132/Go checks/새 native DATA·기존 public Python2 actual을 요구한다. Public startup/source binding·golden/aliases/final은 별도다. CI433 poll03은8SUCCESS/Linux 진행이며 next13:22:11UTC/worker71198을 계속 소비한다.

### 2026-10-29 D2 네 admission 최초 오류 보존 named local 완료

[Receipt](../evidence/package-client-quarantine/p6d2-admission-first-error-local-20261029.json)는 기존1368 불변/current1370·네 반환 join/기존 RunDiscard 보존과 same-test RED4/control6→GREEN10·canonical10/Quick333.493s/전체 sandbox Race34.713s를 연결한다. Actual native negative는 ExitError2/EACCES를 보존하면서 observer 미완료로 Go parent5.01s/exit1/FAIL을 유지했고(operator contract0), 같은 CPP132의 canonical Python actual2/79.40s는 fresh current ALLOW·complete0/2가 PASS다. Native query/golden/aliases/CLI/final은 여전히 OPEN이다. Root87proc/ref0/errors0/정확한6 zombie 재검증 뒤 완료 사본4개85908816B를 제거해 누적217/~7.26GiB Linux 논리 용량이며 원본/CI433 monitor를 보존했다. 다음 cached query의 기존 UV identity/topology/readonly/role 경계를 감사한다.

### 2026-10-29 P6-D2 기존 admission 최초 오류 보존 착수

[Baseline·exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6d2-admission-first-error-plan-20261029.md)는 기존 RunDiscard의 join을 재사용해 나머지 Output/RunInput/RunBounded/RunOutput 네 반환만 교정한다. Same-test RED4/GREEN와 consumed/unconsumed·실제 OS ExitError/원래 incomplete를 검증하고 nonce/lease/session/fail-stop·CPP/권한·한도는 유지한다. D1 cached-query observer FAIL/native public golden·aliases/CLI/final은 별도이며 CI433 frozen/actual20min GET을 계속한다.

### 2026-10-29 D1 native 원본 오류 관측·완료 사본 정리

[Checkpoint](../evidence/package-client-quarantine/p6d1-native-tool-run-probe-local-20261029.json)는 existing1368 불변/tagged1/runtime1369·canonical6/default 재사용과 두 실제 observer FAIL을 보존한다. V2 native exit2/stderr4683B는 cached archive-v0 query 뒤 /dev/null EACCES에 도달했지만 outer admission finishErr가 원래 오류를 가렸다. 기존 RunDiscard는 이미 join하므로 다음 네 wrapper만 교정할 baseline을 감사한다. Cache-query 권한/공개 golden/alias는 미완료다. Source/Host/42/ZIP/Docker·helperwait0/전체 정리는 유지했다. 첫 cleanup2회 삭제0 뒤 exact6 resource-empty zombie를 재검증한 root83proc/ref0/errors0에서 완료 사본4개61940507B를 정리해 누적213/~7.18GiB Linux 논리 용량이며 원본과 CI433 감시는 보존했다.

### 2026-10-29 CI433 actual poll01·독립 D1 첫 거부

21:22:09KST 실제 GET은8SUCCESS/Linux 진행을 기록했고 next12:42:09UTC다. 완료8 logs447248 UTF8B/21chunks와 모두 checkout980e3509/tree238aa647 일치·parent source87c8를 대조해 원본을 보존했다. [독립 D1 v1](../evidence/package-client-quarantine/p6d1-native-tool-run-probe-v1-20261029.json)은 canonical6/default 동일성 뒤 actual5.32s에서 IMAGE classification/OTHER·CONTROL·CLONE_CHILD로 실패했다. 원본과 Host/source/입력 불변·helperwait0/전체 정리는 유지했다. 실제 query/execfn 원인은 미확정이며 같은 tagged test의 underlying native stderr/verbose 관측만 보완한다. CPP/권한은 유지하고 native golden/alias 완료로 쓰지 않는다.

### 2026-10-29 P6-D1 native tool-run DATA 진단 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p6d1-native-tool-run-probe-plan-20261029.md)는 기존 network-none UV actual fixture에 tagged test1만 추가해 local raw wheel의 isolated 준비를 진단한다. 선택 프로그램은 실행 불가능한 상수 /dev/null이며 Artifact body·Policy/public golden/alias 완료로 쓰지 않는다. 기존1368/production·CPP132·권한·한도는 유지하고 첫 query/observer 실패는 그대로 보존한다. CI433/source87c817d는 frozen이며 실제20min worker를 계속 소비한다.

### 2026-10-29 grouped CI433 등록·실제20min 감시

[Registration](../evidence/package-client-quarantine/ci433-registration-20261029.json)은 source87c817d/tree238aa647/3063·1368의 normal FF9198→87c8/remote ref와 actual run40869995094/start12:02:09UTC를 대조한다. 공개 무인증 run/jobs GET HTTP200을 확인했고 worker/session71198/PID3172981의 첫 due12:22:09UTC(한국21:22:09) 뒤20분마다 실제 원본과 다음 due를 저장한다. CI432100m29s와 새16 local368.61s의 약107분은 추정이며 Linux120min은 유지한다. Boundary8/standard-library hostile5/workflow3의 새 hosted 결과는 PENDING이다. 실행 중 새 push 없이 독립 native UVx/golden baseline을 계속하며 aliases/CLI/final·prior fork-only/CPU/drop 원인 미확정은 유지한다.

### 2026-10-29 P6-C2 grouped hosted wiring local 완료

[Receipt](../evidence/package-client-quarantine/p6c2-grouped-hosted-local-20261029.json)는 existing current step에 controlled boundary8/hostile5/public workflow3를 추가하고 기존 refusal2/preparation3/readonly4/runtime2를 보존한다. Runtime1368 중 script/checker3만 변경했고 internal/cmd·CPP132/120min/Required10/permissions는 불변이다. Canonical7·전체 checker Race/Bash syntax와 saved actual logs DATA78cases가 PASS다. 새 native/artifact actual은 반복하지 않았다. 다음 normal grouped FF push 후 actual head/run/start와 새20min GET worker를 등록하며 hosted 결과는 PENDING이다. Native/golden/aliases/CLI/final·prior 원인 미확정은 OPEN이다.

### 2026-10-29 P6-C2 기존 current CI cohort 연결 착수

[Baseline·exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6c2-grouped-hosted-plan-20261029.md)는 runtime1368/production·CPP132를 보존하고 기존 closed dispatch/checker/current step에 boundary8/hostile5/workflow3만 연결한다. 기존 네 소비자와 Linux120min/Required10/permissions는 유지한다. Local canonical/전체 checker Race·저장된 actual 로그 DATA 거부를 확인한 뒤 grouped push와 새 actual20min GET 감시를 진행한다. Native UVx/golden/aliases/CLI/final은 별도다.

### 2026-10-29 P6-C current Python application workflow named local 완료

[Receipt](../evidence/package-client-quarantine/p6c-python-workflow-local-20261029.json)는 기존1365 불변/new3·현재1368의 canonical9/Quick323.823s/전체 Bootstrap·Application Race22.892s와 actual public3/95.89s를 연결한다. Fresh current ALLOW 뒤 normal0/typed guest2는 complete이고 missing console은 PREPARATION_FAILED/no runtime/no staging change다. Selector는 uv-tool-install 그대로다. Host/source/42/ZIP/Docker·helperwait0/전체 정리, root85proc/ref0/errors0 뒤 완료 사본2개54905002B 제거를 확인했다. 누적209/~7.12GiB Linux 논리 용량이며 원본은 보존한다. 다음 기존 current CI cohort에 boundary8/hostile5/workflow3를 연결하기 전 P6-C2 baseline을 감사한다. Native UVx/golden/aliases/CLI/final과 prior fork-only/CPU/drop 원인은 OPEN이다.

### 2026-10-29 P6-C current Python application runtime 착수

[Baseline·exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6c-python-workflow-plan-20261029.md)는 기존 generic ExecutionService와
current Python preparation/opaque console/backend를 재사용해 private runtime adapter/factory
source1·진입 회귀1·public actual3 test1만 추가한다. Source1365/new3/예상1368이며
Core/CLI/CPP·권한·한도는 유지한다. Native uv-tool-install identity를 uvx로 바꾸지 않고
normal0/nonzero2·missing-console after-current-Policy를 같은 workflow에서 확인한다.
CI432는 완료/타이머 종료 상태이고 의미 있는 workflow cohort 뒤 다음 push/20min 감시를
진행한다. Native/golden/aliases/CLI/final·이전 원인 미확정은 OPEN이다.

### 2026-10-29 D2 standard-library hostile5 named local 완료

[Receipt](../evidence/package-client-quarantine/p6b2d2-python-hostile-local-20261029.json)는 source1365/old1361 불변과 기존 refusal의 bounded
trace 진단 한 곳·tagged tests만 연결한다. Composed8/Quick396.34s/전체 sandbox·Bootstrap
Race38.254s 및 current actual5/81.36s가 PASS다. Outside/network·builtin comm-name 거부와
clone-exec의 실제 body/unexpected exec/CLONE_CHILD/ARTIFACT, readonly denied/complete0·
원본 보존을 확인했다. 같은 production/D1/CPP의 earlier8/153.86s는 별도로 재사용한다.
이전 raw fork-only/ctypes fault 원인과 general child/orphan/held-pipe는 OPEN이며 새 입력의
PASS를 인과 해결로 쓰지 않는다. Host/42/ZIP/Docker/source·helperwait0/전체 정리, root97proc/
ref0/errors0 뒤 완료 사본12개329229478B 제거를 확인했다. 누적207/~7.07GiB Linux 논리
용량이며 Windows VHD 축소는 아니다. 다음 기존 application execution interface에 Python
runtime adapter를 연결할 missing baseline을 감사하고 native/golden/aliases/CLI/final은 유지한다.

### 2026-10-29 D2 readonly positive·PID 관측

Readonly test success 경로의 nil error 호출 원본을 보존하고 한 분기만 교정한 actual
22.82s/operator25.058s가 denied prefix/exit0/complete·원본 보존/정리를 확인했다. 별도
clone PID 관측은 parent8/child9 body를 확인했으나 root proc fault와의 actor 대응은
미확정이다. [Plan의 다음 입력](../evidence/package-client-quarantine/p6b2d2-python-hostile-plan-20261029.md)은 builtin thread-name과 fixed subprocess
clone-exec로 ctypes 준비와 actual action을 분리한다. Native fork-only 원본/OPEN과
CPP/권한/한도는 유지하며 전체 hostile5/final 완료로 쓰지 않는다. CI432는 별도 녹색이다.

### 2026-10-29 CI432 current selected Python backend hosted 완료

[Qualification](../evidence/package-client-quarantine/ci432-python-selected-backend-hosted-qualification-20261029.json)은 source9198/tree98ecc/3046·1361과
all10 checkout5458/전체 tree·Required10SUCCESS, decoded1300284B/100chunks를 대조했다.
Actual20min poll05는10:40:44UTC9SUCCESS/Required 진행을 조회했고 바로 후속 MCP read가
완료를 확인했다. Server 전체100m29s, Linux101/noFAIL/noSKIP/CPP132·CPU288.55s/current
Python normal0/nonzero2 55.77s·기존 준비3/readonly4/Node12/CLI4와 helper17wait0/final4
original0·복원이다. 소유 monitor는 완료 확인 뒤 정확한 PID/script/config 신원으로 SIGTERM,
세션exit143을 확인해 종료했고 다음 timer는 없다. 이는 CI 취소나 worker 정상0 종료가 아니다.
D1/D2의 이후 source·fork/readonly test 교정/앱/CLI/final·원인 미확정 CPU/loss는 밖이다.
같은 item에서 readonly test nil 분기와 bounded parent8/child9 진단의 external provenance
차이를 계속하며 source/CPP/권한을 넓히지 않는다. 새 push는 하지 않았다.

### 2026-10-29 D2 v2 partial·readonly oracle 반증

동일 production/CPP132의 v2 actual51.241s는 outside/network PASS와 clone child-body=true를
확인했으나 external provenance가 DIRECT_EXEC_ROOT여서 clone 검증은 미확정이다. 별도
미도달2 선택 actual40.709s에서는 comm-spoof가 PASS이고 readonly는 complete/nil로 기존
oracle을 반증했다. [Plan의 후속 검토](../evidence/package-client-quarantine/p6b2d2-python-hostile-plan-20261029.md)는 failed open을 successful
open의 outside 분류와 혼동한 test-only 기대를 교정한다. 전체 원본/정리는 보존했고 native
CPP/권한은 유지한다. 다음 bounded parent/child PID 진단과 readonly positive만 진행하며
clone 및 D2 전체는 OPEN이다. 완료 첫 사본2를 정리해 누적195/~6.76GiB다.

### 2026-10-29 D2 first actual과 test-only v2 교정

Canonical8/Quick396.34s/affected Race38.254s 뒤 같은 CPP132의 D1 actual8/153.86s와 outside/
network2는 PASS다. Clone은 root의 별도 proc read에서 거부돼 기대 child provenance를
입증하지 못했고 전체 actual203.9s는 FAIL로 보존했다. Host/source/42/ZIP/Docker·helperwait0/
resources/endpoints0는 확인했다. [Plan의 v2 검토](../evidence/package-client-quarantine/p6b2d2-python-hostile-plan-20261029.md)에 따라
parent post-fork proc 조회를 건너뛰는 test-only libc.fork 입력과 기존 정확한 image/pinned
diagnostic 필드만 교정하며 production/CPP/한도는 유지한다. V1 Quick/Race/D1 actual의 동일
입력을 구분해 재사용하고 새 hostile5를 검증한다. 원래 failure와 child/held-pipe OPEN은 유지한다.

### 2026-10-29 D2 Python hostile 실제 경계 착수

[Baseline·exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6b2d2-python-hostile-plan-20261029.md)는 현재1363/D1 actual8/CPP132를
재사용하고 실제 outside/network/clone/comm-spoof/readonly5와 bounded existing trace 진단만
missing으로 기록했다. 기존 D1 wheel bytes·판정과 M5/Pipx default·권한·한도를 보존한다.
Python readonly write는 현재 restricted profile의 outside/incomplete로 검증하며 Node의 다른
positive 판정을 전용하지 않는다. CI432 actual poll03/next10:20:42UTC 감시를 유지하고 새 push
없이 같은 item의 canonical/affected Race와 실제5+기존8을 계속한다. Native/CLI/final은 OPEN이다.

### 2026-10-29 D1 current Python 실제 boundary8 named local 완료

[Receipt](../evidence/package-client-quarantine/p6b2d1-python-boundary-local-20261029.json)는 기존1361/CPP132 불변과
새 tagged test2/current1363의 composed7·Quick419.151s 및 actual8/191.36s PASS를 연결한다.
Raw fixture만 통제했고 current 실제 integrity/static/dynamic/Evidence/M3/M5·fresh approval 뒤
argv/guest37/TERM15/KILL9/원래 취소/각 stream1MiB/실제30s deadline을 확인했다. 신호는
외부 kernel/typed guest error, 취소·overflow·deadline은 incomplete로 보존한다. 첫 tagged
adapter port 누락은 원본 보존 뒤 test-only 거부 Resolve로 보완했고 production은 유지했다.
Host/source/42/ZIP/Docker·helperwait0/전체 정리와 root98proc/ref0/errors0 뒤 완료 사본2개
54849309B를 제거했다. 누적193/~6.71GiB Linux 논리 용량이며 Windows VHD 회수는 아니다.
CI432/source9198은 새 테스트 밖이고 실제20min 감시를 유지한다. 다음 D2 hostile5의 기존
owner 감사/actual gap을 계속하며 native UVx/adapter/CLI/final/미확정 CPU는 OPEN이다.

### 2026-10-29 D1 current Python 실제 boundary 착수

[Baseline·exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6b2d1-python-boundary-plan-20261029.md)는 이미 구현한
backend/CPP132를 그대로 소비하고 controlled raw fixture/current actual inspection·Policy 뒤
argv/exit37/두 신호/취소/두 overflow/real deadline8의 실제 missing 검증만 추가한다. Source 전
1361을 고정하고 test-only2/예상1363을 기록했다. Native/public auth와 fake approval를 구분한다.
CI432 poll01의8SUCCESS/원본444070B/all checkout5458/tree98ecc/CPP132를 확인했고 다음
09:40:39UTC/KST18:40:39 감시를 유지한다. 새 source push·완료 선언 없이 같은 item으로 계속한다.

### 2026-10-29 grouped CI432 등록·실제20min 감시

[Registration](../evidence/package-client-quarantine/ci432-registration-20261029.json)은 source9198/tree98eccd3e/3046·1361의
정상 FF push0d96→9198·remote ref와 실제run40846289893/start09:00:39UTC를 확인한다.
공개 무인증 GET worker/session91578/PID2946324의 실제 run/jobs HTTP200 probe를 확인했고
첫 due09:20:39UTC(한국18:20:39)를 등록했다. Worker가 직접 원본을 저장하고 다음20분을
먼저 등록하며 완료하면 종료한다. 새 shared capture/CPP132/backend2의 hosted 결과는 미완료다.
실행 중 source를 새 push로 취소하지 않고 같은 M13-004에서 Python hostile-boundary의
기존 owner 감사/실제 gap을 독립 진행한다. CPU 미확정·native/adapter/CLI/final은 OPEN이다.

### 2026-10-29 B2-C selected Python hosted wiring local 완료

[Receipt](../evidence/package-client-quarantine/p6b2c-python-backend-hosted-local-20261029.json)는 기존 current step/owned image DATA
exporter의 closed python-execution profile과 fourth exact parent를 연결한다. Canonical7/
Quick321.486s/checker 전체 Race60.2s·Bash syntax와 inert DATA43이 PASS다. 기존 refusal2/
preparation3/readonly4·1361 membership/기존1358 bytes/internal-cmd/CPP132·권한·한도를 유지했다.
새 native/hosted 성공은 아니며 실제 current normal0/nonzero2는 B2-B의 독립 원본을 재사용한다.
최종 후보 검사 뒤 의미 있는 공용 capture/CPP/backend cohort를 한 번 정상 FF push하고 실제
new run/head/start와20min GET 감시를 등록한다. CPU 미확정·native/hostile/adapter/CLI/final은 OPEN이다.

### 2026-10-29 B2-C selected Python hosted 착수

[Baseline·exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6b2c-python-backend-hosted-plan-20261029.md)는
현재 실제 Python normal0/nonzero2 parent를 기존 created-never-started image DATA와 같은
protected current cohort에 연결한다. Backend/CPP132/권한·한도는 유지하고 source 전
3d8b59d/3044·1361을 고정했다. Existing Node refusal2/Python 준비3/readonly4와 새 exact parent/
uniqueIDs/noSKIP/error 우선순위를 검증한 뒤 의미 있는 grouped push와20min 감시를 수행한다.
새 hosted·Python hostile/native UVx/application adapter/CLI/final은 미완료다.

### 2026-10-29 B2-B initial Python backend named local 완료

[Receipt](../evidence/package-client-quarantine/p6b2b-python-artifact-backend-local-20261029.json)는 source1361/기존1352 불변과
기존 readonly input·공유 capture를 소비하는 closed Python ARTIFACT profile/backend를 연결한다.
Go9 검사 중 독립 동일 Go 증거와 v2 Quick56.6s/docs/security·CPP132/305.01s를 출처별로 구분했다.
Same f70/f69의 current Python normal0/nonzero2 71.28s와 기존 Node12 205.66s가 PASS다. 첫
missing exact Python ABI40.25s는 원본 보존 뒤 system/proc/exact flags32768 읽기만 한정 교정했다.
기존 Pipx45.08s CPU guard FAIL과 old-Go/new-CPP counter43.32s·current counter42.14s PASS는
따로 기록하고 원인 미확정/한도 불변을 유지한다. Current379samples/peak11713906us는 진단이지
완료 authority나 인과 해결이 아니다. Host/source/42/ZIP/Docker 보존/helperwait0/전체 정리를
확인했고 완료 사본12개328456846B를 root85proc/참조0·query errors0 뒤 제거했다. 누적191개
약6.66GiB이며 original baselines/CPP/source/raw/inputs/cache는 보존한다. Initial internal API만의
범위이고 Python hostile actual/hosted/native UVx/aliases/application adapter/CLI/final은 미완료다.
같은 M13-004에서 필요한 hosted wiring/backend 묶음 push와20min 감시를 계속한다.

### 2026-10-29 CI431 current inputs hosted 완료

[Qualification](../evidence/package-client-quarantine/ci431-current-input-hosted-qualification-20261029.json)은 source0d96/tree20b/3032·1355와
all10 checkoutf4d7/Required10SUCCESS·decoded1,294,511B/72chunk readback을 확인했다. Actual
20min poll05가07:47:10.790259→07:47:11.998224UTC에 완료를 직접 조회했고 monitor는 exit0로
종료했다. Server 전체99m03이며 Linux100/noFAIL/noSKIP/CPP131·CPU283.54s/current refusal2·
preparation3/readonly4와 helper17wait0/final4 original0·복원을 확인했다. B2-A 공유 구현과 새
Python user-runtime/native aliases/CLI/final은 이 source 밖이다. CI430 loss 원인 미확정은 유지한다.
완료8 원본을 재사용하고 두 로그의 임시 재조립44chunks를 full readback 뒤 제거했다. 같은
M13-004의 B2-B exact ABI 보완/새 actual을 계속하며 작은 helper별 push는 하지 않는다.

### 2026-10-29 B2-B Python selected-program backend 착수

[Baseline·exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6b2b-python-artifact-backend-plan-20261029.md)는
기존 whole readonly input/CPU 원장과 shared captured program/Python irreversible handoff를
재사용하고 새 selected-console profile/backend gap만 구현한다. Source 전1356/commit5c853a1을
고정했고 canonical fixed controller·전체 argv/readonly binding·operational finalization과 program
nonzero 분리를 검토했다. 새 runtime/hosted/UVx equivalence·CLI/final은 미완료다. CI431 poll04는
07:27:10UTC8SUCCESS/Linux running이고 next07:47:10UTC/KST16:47:10을 등록했다. 새 push는 없다.

### 2026-10-29 B2-A 공유 captured program named local 완료

[Receipt](../evidence/package-client-quarantine/p6b2a-shared-program-capture-local-20261029.json)는 기존 Node
본문/mapper를 private helper 한 곳으로 옮긴 source1356/기존1354 불변·canonical9/Quick304.561s/
전체 sandbox·Bootstrap Race35.809s와 같은4f9d binary의 controlled12/195.49s·public CLI4/110.08s
PASS를 연결한다. 공개 결과·privacy·typed error/preflight/input 정리와 CPP131/권한·한도는
유지했다. Host/42/ZIP/Docker/source 보존/helperwait0/전체 정리다. Child body/held-pipe는
미입증이며 Python runtime/native UVx/aliases/CLI·새 shared-source hosted/final은 미완료다.
완료 사본4개109400066B는 원본 보존/root86proc/참조0·query errors0 뒤 삭제했다.
누적179개 약6.35GiB의 Linux 공간 회수이며 C: VHD 축소로 전용하지 않는다. CI431을
새 push로 취소하지 않고 같은 M13-004에서 B2-B baseline/정확한 missing 경계를 계속한다.

### 2026-10-29 B2-A 기존 captured program 본문 공유 착수

[Baseline·exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6b2a-shared-program-capture-plan-20261029.md)는
기존 Node captured admission/terminal/target/source-verify 구현을 private helper로 옮겨 같은
Node caller가 사용하도록 한다. 공개 Node API/typed error/input lifecycle·CPP/권한/한도는
유지하고 Python fake Node layout 또는 별도 workflow 복제를 피한다. Source 전 runtime1355
감사를 기록했으며 새 source의 local 검증은 미완료다. CI431 자동 조회는06:47:07에 실제로
완료됐고 다음07:07:07을 등록했다. Completed8/444981B/checkoutf4d7/tree20b/CPP131은 부분 증거다.

### 2026-10-29 CI431 등록·실제20min 감시

[Registration](../evidence/package-client-quarantine/ci431-registration-20261029.json)은 source0d96/
tree20b74161/3032·1355의 정상 FF push와 원격 ref·실제run40826938443/start06:03:34UTC를
확인했다. Initial6SUCCESS/Linux·macOS·Quick running이며 timer01/session73616은06:23:34UTC
(한국15:23:34)에 실제 발화한다. 새 source를 push로 취소하지 않고 다음 조회 뒤20분을 갱신한다.
Python ARTIFACT runtime owner baseline을 독립 수행하며 새 hosted·user-run/UVx/CLI/final은 미완료다.

### 2026-10-29 B1 readonly4 hosted 연결 local 완료

[Receipt](../evidence/package-client-quarantine/p6b1-hosted-readonly-input-local-20261029.json)는 기존
current step/never-started image DATA owner에 readonly4 closed profile을 추가했다. Canonical7/
Quick257.065s/전체 checker Race54.527s·Bash DATA32가 PASS이며 source1355 membership와
기존1352/internal·cmd/CPP131·권한·한도/관찰 손실 guard는 유지한다. Final 후보 검증 뒤 승인된
정상 push/실제 새 run/20min timer를 등록한다. 새 hosted·user-runtime/UVx/CLI/final은 미완료다.

### 2026-10-29 CI430 Go local 진단·B1 hosted gap 착수

[Diagnostic](../evidence/package-client-quarantine/ci430-go-local-diagnostic-20261029.json)는 같은 Go
owner/CPP·기존 guard의 current1355 단일 grpc1.76.0 parent286.56s PASS와 전체 정리를 확인했다.
원래 CI의 완료 graph는 없으므로 같은 frozen graph 또는 관찰 손실 원인 해결로 쓰지 않는다.
Native write body의 exact upstream 동일성과 오류·retry 의미만 확인했고 config/guard는 유지한다.
[B1 hosted 지도·두 번째 검토](../evidence/package-client-quarantine/p6b1-hosted-readonly-input-plan-20261029.md)
뒤 기존 current cohort에 새 readonly4 gap만 연결한다. 새 의미 있는 source의 qualification을
위해 정상 push하며 timer를 다시 등록하고 완료된 불필요한 test 사본 정리도 이어간다.

### 2026-10-29 CI430 관찰 손실 실패 확인

[Failure record](../evidence/package-client-quarantine/ci430-observation-loss-failure-20261029.json)는 source3c85/
tree82c9/all10 checkouta84ae·decoded966355B readback와8SUCCESS/Linux+Required FAILURE를
연결한다. Linux56PASS/1FAIL/noSKIP의 첫 실패는 기존 grpc1.76.0 transitive Go CLI resolver의
STREAM_FAULT/DROPPED_COUNT다. 관찰 손실을 거부한 guard를 유지하며 손실 발생 조건은 미확정이다.
Helper9wait0/최종 original1·정리/복원을 확인했다. 후속 Node/current C7/Python·kernel 단계는
SKIPPED이며 통과로 쓰지 않는다. Timer03 실제 발화 뒤05:16:30UTC 결과를 소비·종료했다.
B1/source14487/1355와 whole final은 이 source 밖이며 동일 입력 blind rerun 없이 진단을 계속한다.

### 2026-10-29 완료 시험 산출물 정리

[Cleanup receipt](../evidence/package-client-quarantine/completed-test-artifact-cleanup-20261029.json)는
보존된 identity와 일치하는 완료 ELF 사본173개6.199GiB를 제거했다. Root85proc의 cwd/exe/FD/
maps/mount/argv 참조0·query errors0와 exact inode/owner/single-link/hash를 확인했고 최신
B1/A3 기준·CPP·고정 공급·source/raw logs/cache/SDK/Git을 보존했다. Linux /tmp 파일시스템의
공간 회수이며 C: 물리 여유 증가나 VHD 축소로 전용하지 않는다. 사용자 요청대로 작업을 계속한다.

### 2026-10-29 CI429 grouped Node/CLI hosted 완료

[Qualification](../evidence/package-client-quarantine/ci429-node-cli-hosted-qualification-20261029.json)은 source9eb/tree51a9/2996·1343과
all10 checkoutb2eb/Required10SUCCESS·decoded1,308,207B readback을 확인했다. Linux97/noFAIL/
noSKIP/CPP131·CPU300.35s/Node prep48.40s/controlled12 166.71s/application2 38.97s/CLI4
81.39s/kernel8 5.26s/helper16wait0/세 final0·복원이다. Child는 clone-startup 거부·정리만
입증했으며 body/held-pipe는 미입증이다. 전체100m21s이며 timer05 due03:37:40 뒤
03:37:45UTC 결과 소비·종료했다. Later C7/P6/whole final은 이 source 밖이며 계속 진행한다.

### 2026-10-29 P6-B1 named local readonly DATA 완료

[Receipt](../evidence/package-client-quarantine/p6b1-python-readonly-input-local-20261029.json)는 canonical10과
같은1355source/c380binary의 Python readonly4 PASS73.90s·별도 기존 pipx PASS39.40s를 연결한다.
원본·Host/고정42/ZIP/Docker·CPP 보존과 helperwait0/전체 정리를 확인했다. 첫 normal-stop 취소
오인은 한정 교정됐고 이전 pipx CPU guard 실패의 원인은 미확정으로 유지한다. Whole v2 attempt
PASS나 user-run/UVx/CLI 성공으로 바꾸지 않는다. CI430은 B1 source를 포함하지 않으며
현재20분 감시를 유지한다. 다음은 기존 owner를 재사용한 Python ARTIFACT runtime/admission 감사다.

### 2026-10-29 P6-B1 readonly Python inputs 착수

[Exact 지도·두 번째 검토](../evidence/package-client-quarantine/p6b1-python-readonly-input-plan-20261029.md)는
기존 M5 preparation/RO anchor 본문과 actual CPU/observer/cleanup을 공유하고 current
whole context/console source의 DATA 검증 gap만 구현한다. Required unit/Policy·CPP/user-run
완료 의미는 유지하며 source 전 baseline을 기록했다. CI430 timer01을 우선 소비한다.

### 2026-10-29 P6-B Python runtime 기존 owner 감사

[Baseline](../evidence/package-client-quarantine/p6b-python-runtime-baseline-20261029.md)는 이미 구현된
whole closure manifest/실제 consumed stream 인증/full installed census/RO anchor·CPU·cleanup을
확인했다. 실제 source가 허용하는 zero-unit accounting은 준비 DATA만의 경계이며 fake inspection
또는 user-run completion으로 쓰지 않는다. 다음 B1 exact 지도 뒤 current readonly input gap만
진행한다. Production source/CPP 변경 없이 CI430의20min timer를 유지한다.

### 2026-10-29 grouped CI430 등록

[등록](../evidence/package-client-quarantine/ci430-registration-20261029.json)은 검증된3c85d82/
tree82c911ab/3019·1351의 정상 FF push·원격 ref와 actual run40816740622/04:15:37UTC를
확인했다. Initial DocsSUCCESS/8running이며 timer01/session1333은04:35:37UTC(한국13:35:37)
due다. Source를 새 push로 취소하지 않고 실제20min 결과를 확인하며 독립 Python runtime
기존 owner 감사를 계속한다. Hosted/final/Python user-run·UVx/CLI는 미완료다.

### 2026-10-29 P6-A4 current hosted wiring local 완료

[Receipt](../evidence/package-client-quarantine/p6a4-grouped-hosted-local-20261029.json)는 기존 image DATA helper/current
Python profile·protected lifecycle과 C7 refusal2/A3 preparation3의 unconditional 연결이다.
Canonical7/Quick299.996s/전체 checker Race55.890s·Bash syntax와 DATA21cases가 PASS다.
기존 production Go/CPP·cohort/권한/한도는 유지했고 remote PASS나 새 실제 runtime을 만들지
않았다. 최종 docs/security 뒤 승인된 정상 grouped push와 실제20min timer를 등록한다.

### 2026-10-29 P6-A4 grouped current execution hosted 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p6a4-grouped-hosted-plan-20261029.md)는
실제 local qualified Python 준비3/Node refusal2를 기존 helper/고정 fixture/profile로
hosted 연결한다. Production Go/CPP와 기존 Node/kernel/Required·권한/한도는 유지한다.
로컬 wiring/guard 검증 뒤 승인된 정상 push·실제20min timer를 등록하고 후속 runtime 감사를 계속한다.

### 2026-10-29 P6-A3 current Python preparation actual local 완료

[Receipt](../evidence/package-client-quarantine/p6a3-python-preparation-local-20261029.json)는 canonical9/Quick249.346s/Bootstrap
전체 Race23.628s/tagged build와 public cowsay actual3/84.364s를 연결한다. 기존 generic
whole 검사·Policy/records/staging을 재사용해 console 준비2·missing 거부1/fresh Evidence3/
Manifest2를 확인했다. Metadata member는 console declaration 출처이며 user-run 성공이 아니다.
기존1348·CPP/Host/42/ZIP/Docker 불변/helperwait0/workspaces/endpoints0다. Native tool-install
identity는 유지하고 UVx equivalence/Python runtime/CLI·전체 final은 OPEN이다.

### 2026-10-29 P6-A3 current Python preparation 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p6a3-python-preparation-plan-20261029.md)는
기존 native tool-install identity와 current whole PyPI 검사·Policy, A2 console DATA를
generic 준비/보관 서비스에 연결한다. 실제 module user-run/native UVx equivalence·CLI는
별도 후속이며 CI429/source9eb/timer05를 유지한다.

### 2026-10-29 P6-A2 reader와 replay 교정 local 완료

[Receipt](../evidence/package-client-quarantine/p6a2-controlled-reader-local-20261029.json)는 held intake/Run root·한정된 owned
snapshot·실제 canonical parser와 symlink/변조/취소 거부를 연결한다. A1 지문의 map 순서
오류를 실제 RED3.013s 뒤 private 정렬만으로 교정했고 Quick287.612s/전체 두 PyPI Race
14.051s와 canonical6가 PASS다. 기존 parser/Model C/CPP/권한·한도는 유지한다.
Python preparation/runtime/CLI·native equivalence는 미완료이며 CI429/source9eb는 보존한다.

### 2026-10-29 P6-A2 controlled intake reader 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p6a2-controlled-reader-plan-20261029.md)는
기존 handle routing과 A1 parser를 재사용하고 held intake root/한정된 owned snapshot을 연결한다.
Python runtime/CLI/native selection은 별도이며 CI429/source9eb를 보존한다.

### 2026-10-29 P6-A1 exact console DATA local 완료

[Receipt](../evidence/package-client-quarantine/p6a1-console-entry-local-20261029.json)는 기존1344 불변/new2의 focused·
Quick309.386s/전체 PyPI Race11.740s와 canonical checks를 연결한다. 실제 canonical wheel
reader에서만 opaque inventory를 만들고 command/module/attr·원본 metadata/whole inspection/
acquisition을 결합하며 multi/alias/extras/invalid/source/RECORD 치환 거부를 확인했다.
현재 DATA API만 local qualified이고 Python preparation/runtime/CLI는 WIRED=NO다.
CI429는 poll03 8SUCCESS/Linux 진행이며 timer04 due03:17:34UTC/session48576로 유지한다.

### 2026-10-29 P6-A1 console entry DATA 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p6a1-console-entry-plan-20261029.md)는
실제 canonical wheel reader에서만 생성한 opaque inventory와 exact console selection gap을
새 DATA API로 추가한다. 기존 parser/Model C/context/workflow/records/runtime/CPP는 유지한다.
명령·module/attr·원본 metadata member와 전체 inventory/acquisition을 결합하며 selection은
Policy나 실행 권한이 아니다. Actual Python/user-run/native selector/CLI는 후속이고
CI429/source9eb/timer03 due02:57:27UTC/session39633를 보존한다.

### 2026-10-29 P6 existing owners/pinned UV source 착수 감사

[Baseline](../evidence/package-client-quarantine/p6-execution-adapter-baseline-20261029.md)는
Core context/Application/records/capture의 기존 구현을 그대로 재사용하고 actual native selection/
Python console binding/readonly runtime/CLI gap을 구분한다. [고정 UV source discovery](../evidence/package-client-quarantine/p6-uv0134-source-discovery-20261029.json)는
원본 commit7456b60b의5file Git blob/SHA256/readback을 확인했다. Tool install과 tool run의
의미 동일성은 아직 UNPROVEN이며 cache/PATH/env/stdin 기본값을 HAA 권한으로 수입하지 않는다.
새 HAA source/placeholder/패키지·설정·CI job·버전/권한 변화는 없고 CI429는 계속 감시한다.

### 2026-10-29 C7 real post-Policy CLI refusal local 완료

[Receipt](../evidence/package-client-quarantine/p5c7-cli-refusal-local-20261029.json)는 existing1343 불변/test-only1의
canonical6와 actual refusal2/33.886s를 연결한다. 실제 public TypeScript의 fresh Evidence2/
원본 SHA/current Policy ALLOW 뒤 ambiguous/missing bin을 거부하고 준비·실행·exit·출력·
manifest 부재를 확인했다. Production/CPP/권한·예산은 그대로이며 Host/source/42/ZIP/Docker
보존과 helperwait0/resources/endpoints0다. 새 테스트는 CI429 source9eb에 없고 active run을
취소하지 않는다. Next는 local commit 뒤 bounded acceptance/P6 owner baseline이다.

### 2026-10-29 C7 post-Policy CLI refusal 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p5c7-cli-refusal-plan-20261029.md)는
기존 entry 선택 거부/actual검사/CLI를 유지하고 real Policy ALLOW 뒤 ambiguous/missing bin의
보관·실행 NOT_STARTED 두 actual gap을 새 test parent 하나로 검증한다. Production/CPP/권한·
예산은 유지하며 fake approval/target/result는 없다. CI429 source9eb854f와 timer01
due02:17:02UTC/session84875를 보존하고 새 push하지 않는다.

### 2026-10-29 current grouped CI429 등록

[등록](../evidence/package-client-quarantine/ci429-registration-20261029.json)은 검증된9eb854f/tree51a9a25c/2996·1343의 normal FF push와 remote ref를 확인했다.
CI429/run40803448693은01:57:02UTC 시작/initial9running이며 준비·kernel과 current
boundary12/workflow2/CLI4를 포함한다. 실제20min timer01/session84875는
02:17:02UTC(한국11:17:02)가 due다. 새 hosted PASS로 표시하지 않고 실행 중 push를
보류한다. 같은 M13-004의 독립 P5 bounded acceptance/P6 owner baseline을 계속한다.

### 2026-10-29 C6 hosted wiring local 완료

[Local receipt](../evidence/package-client-quarantine/p5c6-grouped-hosted-wiring-local-20261029.json)는 기존 Node hosted step에
boundary12/workflow2/CLI4를 개별5min command와 noSKIP/unique cases/original exit guard로
연결한 canonical3·Bash syntax·guard24/primary6 결과다. Go/CPP/runtime1343과 기존
preparation/kernel/Required·권한·예산은 불변이다. 로그 guard는 DATA 시험이며 actual
qualification을 새로 만들지 않는다. 다음 승인된 정상 grouped push/run/timer 등록 전
최종 문서·보안을 확인하고 hosted/whole final acceptance는 OPEN으로 유지한다.

### 2026-10-29 C6 grouped hosted wiring 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p5c6-grouped-hosted-plan-20261029.md)는
기존 unconditional Node hosted 단계에 현재 boundary12/workflow2/CLI4를 개별5min invocation으로
연결한다. Go/CPP/권한·예산/Required와 기존 preparation/kernel 소비자는 유지한다.
Canonical workflow·Bash/로그 guard를 확인한 의미 있는 후보만 승인된 정상 push로 전달하고
시작 직후20min 타이머를 등록한다. Hosted/whole P5/final은 결과 전까지 OPEN이다.

### 2026-10-29 CI428 source558 backend hosted 완료

[Final qualification](../evidence/package-client-quarantine/ci428-node-backend-hosted-qualification-20261029.json)은 Required10SUCCESS와
all10 same checkoutd965d5bf/treef5e4d9f9/2969·1330 및 decoded1,278,395B readback을
연결한다. Linux94/noFAIL/noSKIP/CPP131·CPU296.32s/current public Node fresh2 48.93s의
정상0·typed guest1과 whole readonly/current Policy·kernel8cases5.41s/helper16wait0/
세 final original0·복원을 확인했다. 전체94m51s이며 timer05는01:42:43UTC 결과 소비 후
종료했다. Later local C3d/C4/C5는 이 source에 없고 grouped hosted가 다음이며
과거 native deadline 직접 원인·whole final/P5-P7/signed는 여전히 OPEN이다.

### 2026-10-29 C5 npx/npm exec actual CLI local 완료

[Receipt](../evidence/package-client-quarantine/p5c5-npx-cli-local-20261029.json)는 canonical9/전체 영향 Race와
actual public CLI4/113.529s를 연결한다. Program --help·literal argv는 실제 entry로
전달됐고 두 alias·human/JSON·en/ko·private/explicit output·정상0/typed guest1 및 fresh
records4가 PASS다. Backend/CPP/권한·한도는 유지하며 Host/source/42/ZIP/Docker 보존과
helperwait0/resources/endpoints0다. 마지막 stale1332 producer assertion은9checks PASS 뒤
기록 오류이며 독립 current1343 validator0로 확인하고 제품 검사를 반복하지 않았다.
다음 grouped hosted wiring은 먼저 baseline을 기록하며 CI428 진행 중 push하지 않는다.

### 2026-10-29 C5 safe output/npx CLI 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p5c5-npx-cli-plan-20261029.md)는
기존 C4/current backend를 재사용하고 bounded npx/npm exec grammar와 분리된 안전한
출력·실제 CLI 연결 gap을 구현한다. 기본 output은 bytes count이며 명시 show-output은
ASCII escaped untrusted data다. Parsed HAA help와 program --help를 구분하고 input bounds를
I/O 전에 검사한다. CI428 source558d75a는 timer05 due01:42:24UTC로 보존한다.

### 2026-10-29 C4 application workflow actual local 완료

[Local receipt](../evidence/package-client-quarantine/p5c4-execution-workflow-local-20261029.json)는 canonical9/전체 영향 Race와
actual public TypeScript2/58.861s를 연결한다. 기존 준비·materialize·Node backend를
Application interface와 concrete Bootstrap adapter로 연결했고 fresh current 검사/Policy/records/
private argv 및 정상0/비정상1 원래 cause를 확인했다. 결과의 terminal/output은 복사·비공개이며
준비 실패·모순 terminal·late caller 취소는 완료로 표시하지 않는다. 기존1332/CPP/old actual
parents·권한/예산은 그대로이고 Host/source/42/ZIP/Docker·helperwait0/resources/endpoints0다.
다음은 안전한 출력과 public CLI이며 CI428 진행 중 새 push는 하지 않는다.

### 2026-10-29 C4 application workflow 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p5c4-execution-workflow-plan-20261029.md)는
기존 준비/검사·Policy/materialize/Node backend를 재사용하고 application의 동일 요청 연결과
private 결과·원래 오류 유지 gap만 구현한다. 기존 actual parent와 controlled12는 유지하고
새 public2 parent로 실제 연결을 검증한다. CLI/renderer/final은 후속이며 CI428은 보존한다.

### 2026-10-29 C3d3 same-source12 actual local 완료

[Local receipt](../evidence/package-client-quarantine/p5c3d3-node-deadline-readonly-local-20261029.json)는 actual202.496s/12cases와
canonical/tagged6checks를 연결한다. Caller30s deadline은 body ready 뒤 원래 cause/incomplete,
readonly self-write는 kernel 거부를 처리한 program0과 원본 전체 bytes 보존이다. 기존10과
source/Host/42/ZIP/Docker 보존·helperwait0/resources/endpoints0/noSKIP도 PASS다.
Production/CPP/권한·예산은 그대로이며 public application/CLI/final은 다음이다. CI428은
01:02:15UTC 8SUCCESS/Linux 진행이고 timer04 due01:22:15UTC/session34757로 유지한다.

### 2026-10-29 C3d3 actual deadline/readonly self-write 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p5c3d3-node-deadline-readonly-plan-20261029.md)는
local951f73e/기존1332의 current runtime/검사/Policy를 재사용하고 actual ordinary caller
WithTimeout 뒤 body DeadlineExceeded와 readonly 자기 파일 쓰기 거부 두 gap만 추가한다.
기존10까지 same-source12cases로 검증하며 production/CPP/45s/stream1MiB/caller4min은
유지한다. 쓰기 거부를 처리한 program completion과 파일 쓰기 성공을 구분한다. CI428은
source558d75a/timer03 due00:59:39UTC로 유지하고 새 push하지 않는다.

### 2026-10-29 C3d2 current10 hostile actual local 완료

[Local receipt](../evidence/package-client-quarantine/p5c3d2-node-hostile-boundary-local-20261029.json)의
same1332/production·CPP 불변·canonical/tagged6checks와 actual154.558s/10cases가 PASS다.
Outside path/network의 실제 fact와 comm spoof의 kernel NODE/ARTIFACT를 확인했고 child는
clone fault/ARTIFACT/incomplete/cleanup이다. Child WaitDelay=false·JS 시작 미입증을 명시했다.
V1 role 관측 한계와 v2의 actual0 후 tuple/list operator 오류를 각각 보존하고 독립 readback
검증으로 기록만 바로잡았으며 actual을 재실행하지 않았다. Source/Host/42/ZIP/Docker 보존·
helperwait0/resources/endpoints0/noSKIP다. Next C3d3은 body 뒤 actual deadline/readonly self-write이며
먼저 local unit commit만 보존한다. CI428/source558d75a/timer03 due00:59:39UTC는 유지한다.

### 2026-10-29 C3d2 actual hostile 경계 착수

[C3d2 baseline·지도·두 번째 검토](../evidence/package-client-quarantine/p5c3d2-node-hostile-boundary-plan-20261029.md)는
local a6e7ee6의 기존1332·current검사/Policy/runtime을 재사용해 outside path/guest-loopback
network/Artifact child/comm spoof 네 actual gap만 잡았다. 두 test만 수정하고 fixture bytes가
바뀌므로 기존6case까지 same source10case로 검증한다. Production/CPP/일반FS·network·role/
한도는 유지하며 child startup 거부를 성공한 child·pipe drain 증거로 표시하지 않는다.
CI428은558d75a/8SUCCESS/Linux 진행과 timer02 due00:38:26UTC를 유지하고 새 push하지 않는다.

### 2026-10-29 C3d1 selected Node 신호·취소·독립 overflow actual 완료

[Local receipt](../evidence/package-client-quarantine/p5c3d1-node-boundary-local-20261029.json)의
1332inputs/newtests2·기존1330/CPP 불변·canonical6checks와 actual92.871s/6fresh cases가 PASS다.
실제 current verifier/static/dynamic/Evidence/Policy 뒤 literal argv·kernel TERM15/KILL9,
body 준비 뒤 custom cause 취소/incomplete와 stream별1MiB overflow/incomplete·원래 cause를
확인했다. Host/source/42/ZIP/Docker 보존·helperwait0/owned resources/endpoints0/noSKIP다.
공급망/native solver는 controlled fixture/NOT_RUN이며 production/CPP/한도 변경은 없다.
현재 CI428에는 이 test source가 없고 새 push 없이 한글 local commit을 보존한 뒤 같은
M13-004 C3d2의 FS/network/child hostile baseline을 계속한다. CLI/final/wholeP5는 OPEN이다.

### 2026-10-29 C3d1 actual Node failure 경계 착수

[Baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p5c3d1-node-boundary-qualification-plan-20261029.md)는
source558d75a/1330의 기존 backend/Host capture/kernel/cleanup·실제 검사/Policy를 재사용하고
selected Node TERM/KILL·중간 caller cause 취소·stdout/stderr overflow·literal argv의6fresh
actual만 missing으로 잡았다. 두 Linux realcorpus test 파일만 추가하며 production/CPP/권한/
예산/기존 parent는 유지한다. Local controlled registry fixture는 공급망/native solver 성공이
아니며 actual current integrity/static/dynamic/Evidence·Policy와 runtime을 구분한다. CI428
source와 timer는 그대로이며 새 push 없이 같은 단일 M13-004에서 계속한다.

### 2026-10-29 승인된 Artifact Node foundation CI428 등록

[CI428 등록](../evidence/package-client-quarantine/ci428-registration-20261029.json)은 normal/nonzero
backend·CPP131/실제2requests·기존8roots/3consumers와 final docs/security를 확인한21경로의
한글 commit558d75a/treef5e4d9f9/2969·1330입력을 기존 milestone branch에 정상 FF push한 결과다.
Remote ref/source를 대조했고23:57:29UTC 시작 뒤 first20min timer01/session55237의 due는
2026-10-29 00:17:29UTC(한국09:17:29)다. 초기 Docs SUCCESS/나머지 진행이며 whole hosted는
PENDING이다. 실행 중 새 push로 취소하지 않는다. 같은 M13-004의 독립 C3d actual failure/
악성 qualification을 baseline audit부터 이어가며 그 이후 source와 public CLI/final은 밖이다.

### 2026-10-29 실제 Artifact Node normal/nonzero backend local 완료

[최종 local receipt](../evidence/package-client-quarantine/p5c3c2b-artifact-node-local-20261029.json)는
current1330/Go-v6 checks8·CPP131/canonical289.119s와 actual69.401s/fresh2의 guest0·guest1,
기존8roots9.678s/세 consumers12.050s PASS를 연결한다. Current whole Policy·readonly inputs,
accepted target/external kernel terminal/independent bounded streams와 complete trace/disposal,
Source/Host/42/ZIP/Docker 보존·helperwait0/owned resources/endpoints0을 확인했다. 앞 cwd/
runtime ABI 실패 원본을 보존했고 일반 FS/CONTROL/network/한도는 유지했다. Qualification은
이 named normal/nonzero foundation뿐이다. 다음은 같은 M13-004의 C3d 실제 취소/overflow/
child·악성 경계이며 먼저 current unit을 검토/중간 commit/중요 grouped CI로 전달한다.
New hosted/public application·CLI/wholeP5와 final profiles/signed는 OPEN이다.

### 2026-10-29 CI427 공통 후보 hosted qualification 완료

[최종 qualification](../evidence/package-client-quarantine/ci427-kernel-readonly-hosted-qualification-20261029.json)은
sourcee0dc/treefff27985/2959·1324입력과 all10checkout0e703657 일치·Required10SUCCESS다.
Timer04는23:18:00UTC 발화/23:18:15UTC query로 완료를 소비했고 final update까지78m45s다.
All10 decoded1,229,906B/chunk readback과 Linux94/noFAIL/noSKIP/CPP130·CPU238.05s,
production readonly fresh2 34.71s/kernel8cases5.62s·helper16wait0/세 cleanup original0/복원을
확인했다. 추가 timer는 없으며 이후 Node source1330/profile/backend·CPP131와 user-run/CLI/
wholeP4/P5-P7/final4profiles/signed는 이 run 밖이다. 같은 M13-004의 Node actual startup
self-cgroup read gap을 계속한다. Cwd 교정 후보는 reader 시작을 통과했지만 전체 Node 실행은
실패로 유지하며 새 CPP build/실제 검증 전 qualification으로 표시하지 않는다.

### 2026-10-29 Artifact Node 첫 actual 실패와 제한 진단

[현재 수정 지도](../evidence/package-client-quarantine/p5c3c2b-node-execution-plan-20261029.md)의
새 Node CPP131/canonical은 PASS지만 actual-v1은33.158s·readonly reader start 단계에서
실패했다. Artifact 실행은 미도달이며 첫 원본/Host·입력 보존·정리를 별도로 확인했다.
Default runtime DATA 세 cwd 조합 PASS를 gVisor 원인으로 일반화하지 않는다. 기존
ExitError.Stderr를 고정 공개 integration fixture에서만 제한 수집하는 test-only 변경으로
상세 원인을 확인한다. Node actual/전체 qualification은 NO이며 CI427 source는 불변이다.

### 2026-10-29 승인된 공통4commit CI427 등록

[CI427](../evidence/package-client-quarantine/ci427-registration-20261029.json)은 sourcee0dc39a/tree
fff27985/2959·1324입력의 누적4commit을 기존 milestone branch에 정상 FF push한 결과다.
Kernel target/terminal8roots와 clean readonly production fresh2·canonical/CPP130/whole 후보
일치를 확인한 구체 후보이며 main merge/tag/release는 없다. 21:56:16UTC 등록 뒤 실제20min
timer01/session20366의 첫 due22:16:16UTC(한국07:16:16)를 설정했다. New hosted는 PENDING이고
실행 중 source를 새 push로 취소하지 않는다. 같은 M13-004의 독립 Node mode/profile/lifecycle
감사를 계속하며 이후 소스와 user-run/CLI/final qualification을 이 run에 전용하지 않는다.

### 2026-10-29 기존 Linux CI kernel parent 연결 로컬 검증

[Wiring receipt](../evidence/package-client-quarantine/p5c3c-hosted-kernel-wiring-local-20261029.json)의
canonical7/checker fullRace47.748s·workflow9 negatives/Bash7guard와 current source1324/Go3575dbab/
CPP0c09 readonly production fresh2 actual56.226s가 PASS다. 기존8root/제품/CPP는 재구현하지
않고 기존 job/태그 빌드·protected client/helper를 재사용했다. Source/Host/42/ZIP/Docker·정리와
최초 producer hash guard 중단/교정은 개별 원본에 보존한다. 새 hosted는 아직 PENDING이며
완료한 C3c 단위를 한 번 push해 새run/20min timer를 등록한 뒤 같은 item의 Node profile을 계속한다.

### 2026-10-29 CI426 종료와 target-image dependency 검증

[CI426](../evidence/package-client-quarantine/ci426-offline-materialization-hosted-qualification-20261029.json)은
source4db/tree e8ad4e1d/2940·all10checkoutc106c554/decoded1,274,315B 일치와 Required10SUCCESS다.
실제 timer05 21:33:21UTC 발화 뒤21:33:27UTC에 완료를 소비했다. 전체95m30/Linux93/noSKIP/
CPP128/CPU303.82s/helper15wait0/final0와 production31.97s/fresh2·offline materialization이 PASS다.
추가 timer는 없으며 CI425 native 원인 미확정과 later C3c/new user-run/final 미완료는 유지한다.
[새 target fact local](../evidence/package-client-quarantine/p5c3c2b-target-image-local-20261029.json)은
CPP130/8root9.827s·기존3소비자12.296s/Host·입력 보존·정리를 확인했다. 같은 root의 helper
종료와 실제 accepted target load를 구분하며 Node user-run 성공으로 확대하지 않는다.
이8root parent를 기존 Linux CI에 연결한 뒤 누적된 공통 변경을 한 번 push하고 같은 item을 계속한다.

### 2026-10-29 P5-C3c2a 읽기 전용 whole-tree 입력 로컬 검증

[결과](../evidence/package-client-quarantine/p5c3c2a-readonly-node-inputs-local-20261029.json)는
source1320/Go31836d82/unchanged CPP4df의 actual54.439s·fresh2 PASS다. Current whole 검사/
Policy와 offline materialization 뒤 clean readonly tree 전체 metadata/bytes를 검증했다.
Canonical7/영향3package freshRace30.352s와 Source/Host/42/ZIP/Docker 보존·helperwait0/final0/
workspace 부재가 대응한다. 첫 actual 실패와 별도 DATA lifetime probe를 보존하고 기존 Python
owner의 reader 선연결→writer 종료→sole RO 순서를 재사용했다. Artifact user program은
미실행이며 같은 M13-004의 C3c2b admission/image-start/terminal/output/cleanup을 계속한다.
CI426은 이전4db source로 완료됐고 최종 결과는 상단 receipt를 따른다.

### 2026-10-29 P5-C3c1 외부 kernel root terminal 로컬 검증

[결과](../evidence/package-client-quarantine/p5c3c1-kernel-terminal-local-20261029.json)는 source1316/
Go b5166b73/CPP4df80d98의8fresh trusted controller roots9.014s와 기존 npm/Python/ELF3parents
11.163s PASS다. Kernel의 exit0/37/125/126/127/255·TERM/KILL과 별도3B stdout/stderr를 대조했다.
CPP129/canonical checks·원본과 Source/Host/42/ZIP/Docker 보존·helperwait0/final0정리가 대응한다.
Host client exit/stdout marker를 guest signal이나 coverage로 쓰지 않는다. Root 종료만으로 actual
Artifact Node image 로드·출력 완전성·descendant drain을 증명하지 않으며 C3c2가 다음이다.
일반 role/FS/network/예산은 유지했고 new hosted/whole common/CLI/final은 OPEN이다. 실행 중인
CI426은 이전 source4db0111이며 최신 타이머는 이 Queue 상단과 registration을 따른다.


### 2026-10-29 P5-C3b offline whole-set materialization 로컬 검증

[최종 receipt](../evidence/package-client-quarantine/p5c3b-offline-node-materialization-local-20261029.json)는
source1313/binary0c4bfe0e·기존 CPP128의 actual45.343s/parent43.57s/fresh2 PASS다. Current
whole checks·Policy ALLOW/storage-only 뒤 fixed no-script/offline materialization의 모든 파일과
entry를 strict C2 binder로 검증했다. 같은 layout digest와 Source/CPP/42/ZIP/Host/Docker 보존,
helperwait0/final0/Node workspace 부재를 확인했다. Canonical Quick77.668s/영향3package
fullRace29.883s/기타5checks와 개별 원본이 대응한다. 실제 artifact user program은 실행하지
않았으며 CLI/guest exit-output/final4profiles·signed/전체P5는 OPEN이다. 앞 v1-v4는 실패로
보존하며 일반 FS/role/CPU 예산을 바꾸지 않았다. 다음 C3c owner audit와 의미 있는 grouped
hosted/native Required qualification을 계속한다.


### 2026-10-29 CI425 완료 소비 — Linux preparation PASS / macOS·Required FAIL

[최종 결과](../evidence/package-client-quarantine/ci425-production-preparation-hosted-result-20261029.json)는
source418a/tree016df83d/2926·all10checkout3cc28f9e/decoded1,346,740B의 동일성을 확인했다.
실제 timer05 발화19:32:09UTC 뒤19:32:16UTC에8SUCCESS/macOS+Required FAILURE를 직접
소비했다. Final update까지89m03이며 Linux93parents/noSKIP/noFAIL/CPP128/CPU277.18s,
helper15wait0/final original0·복원과 새 production preparation19.28s/fresh2/current whole checks·
Policy ALLOW/storage-only가 PASS다. Later C3a/C3b·user program/CLI/final4profiles·signed는 밖이다.
Native default6m 직접 원인은 기존 observation대로 OPEN이며 Required 실패를 성공으로 바꾸지
않는다. Timer05는 종료됐고 새 timer/재실행은 없다. 같은 M13-004의 독립 C3b 진단을 계속하며
다음 의미 있는 grouped source에 native/Required qualification을 연결한다.


### 2026-10-29 P5-B2f production preparation 실제 local 검증

[B2f receipt](../evidence/package-client-quarantine/p5b2f-production-preparation-local-20261029.json)의
no-overlay30.930s/parent28.93s는 실제 factory/M3·exact TypeScript5.9.3/source SHA·required3/
Evidence4/Policy/fresh2회·staging PASS다. Source1301/CPP/원래42/ZIP·Host/Docker 불변과
helperwait0·Node workspace 부재를 확인했다. User program은 실행하지 않았다. 기존 Linux
job에 tagged parent와 exactPASS1/noSKIP guard를 연결했고 workflow·post-wiring Quick81.733s/
Bash8case가 PASS다. 새 hosted step/actual user-run/CLI는 별도다. CI424 두 번째 실제 조회
16:40:43UTC는8SUCCESS/Linux 진행이며 다음17:00:43UTC/session2975를 계속 소비한다.

### 2026-10-29 P5-B2e preparation 로컬 검증

[B2e receipt](../evidence/package-client-quarantine/p5b2e-execution-preparation-local-20261029.json)의
Quick229.702s/영향5package fullRace39.718s/docs/security가 PASS다.
Target 없는 resolver→fresh 검사→actual Node proof→current context→records→staging을 연결했다.
보관 handoff actual은 fixed DATA/M1 fixture이며 production dynamic/user program은 미수행이다.
Private request의 argv copy·format/JSON 보호와 cached entry/records·wrong staging/command·
최초 provider/취소 cause를 확인했다. 준비 내부 WIRED=YES, actual backend/CLI는 아직NO다.
[CI424 registration](../evidence/package-client-quarantine/ci424-registration-20261029.json)의 첫 timer
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

[B2d receipt](../evidence/package-client-quarantine/p5b2d-controlled-node-entry-read-local-20261029.json)의
Quick279.401s/inspection npm fullRace17.241s/docs/security가 PASS다.
기존 capture의 optional writer와 새 reader/회귀만 추가했다. Original 전체 source/inventory·
selected bytes를 결합하며 치환/원본 변조/link/압축/다른 interpreter/cancel은 proof를 주지 않는다.
Actual flow WIRED=NO이며 다음은 resolver/fresh inspection/context/records/staging 조립이다.

### 2026-10-29 P5-B2c fresh whole inspection 로컬 검증

[B2c receipt](../evidence/package-client-quarantine/p5b2c-fresh-execution-inspection-local-20261029.json)의
Quick40.217s/application fullRace4.233s/docs/security가 PASS다. Existing install/project per-entry·
batch helper와 Policy를 그대로 재사용한 새2file이다. 반복 resolution도 fresh whole 검사를
수행하며 누락/치환·entry/set 거부·부분 실패/store 원인·ID/취소/derived 거부를 확인했다.
Actual command flow는 아직 WIRED=NO이며 다음은 기존 archive capture의 selected bytes와
실제 resolver/inspection/Node/context/records/staging 조립이다. 단일 M13-004를 계속한다.

### 2026-10-29 CI423 current ordinary hosted qualification

[CI423 qualification](../evidence/package-client-quarantine/ci423-pipx-current-hosted-qualification-20261029.json)은
같은6dd/tree feab8ef9/2891·10checkout과 Required10SUCCESS/Linux92/noSKIP/CPP128/CPU251.33s,
UID1001·pipx current named parents/helper14wait0/final0을 확인한다. Decoded1,221,192B 전체를
매 chunk readback/SHA로 보존했다. 네 번째 실제20min 조회15:14:14UTC에서 완료를 소비했고
추가 timer는 없다. C-e4 ordinary named scope만 닫으며 final4profiles·signed와 후속 P5 source는
이 결과 밖이다. 다음은 같은 단일 M13-004의 P5 current whole 검사/application이다.

### 2026-10-29 P5-B2b Node composition / execution records 로컬 검증

[B2b receipt](../evidence/package-client-quarantine/p5b2b-node-composition-records-local-20261029.json)의
Quick220.145s/bootstrap·evidence fullRace26.899s/docs/security가 PASS다. 기존1287source는
불변이며 새4file은 exact Node pin 조립과 target 없는 records만 담당한다. Raw argv/hash는
저장하지 않고 whole Evidence/Policy/SBOM binding을 유지한다. Actual flow WIRED=NO이며
다음은 fresh whole inspection/application과 staging/backend/CLI다. 같은 단일 item을 계속한다.
CI423의 마지막 timer는 실제15:14:14UTC Required10SUCCESS로 소비했고 추가 timer는 없다.

### 2026-10-28 P5-B2a metadata context 로컬 검증

[B2a receipt](../evidence/package-client-quarantine/p5b2a-execution-context-local-20261028.json)의
Quick288.508s/domain fullRace2.620s/docs/security가 PASS다. 기존1285source와 VerifiedSet/
Resolution을 유지하고 새2file의 complete graph/primary acquisition/member/runtime metadata·
private argv와 source/Policy identity를 검증했다. 실제 P5 flow는 WIRED=NO이며 다음은 Node proof와
canonical runtime의 composition 및 current 검사/Manifest·run 연결이다. CI423은 third actual
20min query14:53:43UTC에도8SUCCESS/Linux 진행이며 next15:13:43UTC/session51968이다.

### 2026-10-28 P5-B2 기존 owner 재사용 감사

[B2 actual owner 감사](../evidence/package-client-quarantine/p5-execution-foundation-baseline-20261028.md)는
VerifiedSet/VerifiedBundle/LocalStaging이 generic이고 InstallContext를 요구하지 않음을 실제
source에서 확인했다. Existing inspection helpers·Manifest entry/SBOM builders를 재사용하고
non-install context/entry·argv/runtime/source-policy 동결의 missing만 수행한다. Source 전
계약·문법/bounds와 두 번째 검토는 진행 중이며 B2 구현/실제 run 연결은 아직 미완료다.
CI423 source6dd와 실제 다음 due14:33:07UTC/session84800은 그대로다.

### 2026-10-28 CI423 첫20min 소비와 P5-B1 로컬 검증

14:12:50.000193UTC timer1 발화 뒤14:13:07UTC same6dd/8SUCCESS/Linux 진행을 직접 조회했다.
완료8logs442775B/매 chunk readback·checkoutb571e02 전체tree feab8ef9/2891files가 source와
같다. Required/전체 Linux는OPEN이며 다음 due14:33:07UTC/session84800을 등록했다.
[P5-B1 receipt](../evidence/package-client-quarantine/p5b1-node-entry-bytes-local-20261028.json)의
Quick256.332s/npm fullRace49.546s/docs/security가 PASS다. 기존1283source와 validator는
불변이고 새2file의 bytes/Node 선언 value만 검증했다. Runtime 모듈 import 실패 후보를 보존하고
기존 Architecture를 유지했다. A/B1은 실제 P5 run 연결이 없어 WIRED=NO다. 다음 B2의 whole
graph/argv/runtime/source-policy와 전체 실행·CLI는 미완료다. CI423 source는6dd로 고정한다.

### 2026-10-28 P5-B1 selected bytes / Node runtime 착수

[P5-B1 audit/map/두 번째 검토](../evidence/package-client-quarantine/p5-execution-foundation-baseline-20261028.md)에
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

[로컬 receipt](../evidence/package-client-quarantine/ci422-qualification-input-fix-local-20261028.json)의
Quick124.549s/영향 fullRace28.622s/tag/docs/security·no-overlay resolver actual45.715s가 PASS다.
기존 full CPU와 같은 canonical pin만 positive에 연결했고 metadata hash를 wheel로 전용하는
negative를 추가했다. Production guard/SDK/CPP/CI job·예산은 그대로다. Source/Host/입력·helper
정리를 확인했으며 whole hosted/final/signed는 미완료다. 누적 source의 검토 후 기존 CI의
정확한 후보 재qualification을 준비한다. 새 public push/timer/재실행은 아직 없다.

### 2026-10-28 CI422 same-source report DATA 진단

[진단 receipt](../evidence/package-client-quarantine/ci422-pytorch-report-diagnostic-20261028.json)의
local12.244s는 pip report archive_info={}와 공식 Simple의 같은 wheel hash 부재를 확인했다.
Sidecar metadata SHA는 wheel hash가 아니다. 원래 CI report는 없으므로 같은 후보라는 주장은
추론으로 남긴다. Source/Host/입력 불변과 helper 정리를 확인했으며 기존 거부를 유지한다.
다음은 기존 qualified exact fixture·CI 입력 owner 감사이며 새 runtime pin/검증 완화·재CI는 없다.

### 2026-10-28 CI422 실패 원본·소스 대조

[실패 receipt](../evidence/package-client-quarantine/ci422-p4-current-cohort-failure-20261028.json)의
8SUCCESS/Linux·Required FAILURE와 all10 checkout/source whole tree 동일성을 확인했다.
Linux24PASS/noSKIP/CPP128 뒤 torch report candidate가 거부됐고 raw report/정확한 조건은 아직
미확정이다. Helper6wait0/최종 오류1 보존·원본 client 복원은 성공이다. Full CPU와 뒤 pipx는
미도달이며 whole qualification을 닫지 않는다. 실제20min timer2를13:13:49UTC 소비했고 현재
active timer/CI는 없다. 재실행 없이 same-source bounded DATA 진단을 계속한다.

### 2026-10-28 P5-A 구조적 진입점 선택 로컬 검증

[P5-A receipt](../evidence/package-client-quarantine/p5a-execution-entry-local-20261028.json)의
Quick37.506s/npm fullRace49.102s/focused/docs/security가 PASS다. 기존1281source/parser는
불변이고 새2file의 구조적 선택·원본 전체 binding만 검증했다. 실행·CLI/전체 P5는 미완료다.
첫 test ID 형식 오류와 교정을 보존했다. CI422는 두 번째 실제20min 조회13:13:49UTC에
Linux/Required FAILURE·나머지8SUCCESS다. 최초 torch pip report candidate 거부와 helper6wait0/
복원을 확인했으며 현재 원인 진단을 계속한다. Timer2는 소비 종료했고 새 timer/재실행은 없다.

### 2026-10-28 P5-A exact bin 선택 착수

[P5-A 지도·두 번째 검토](../evidence/package-client-quarantine/p5-execution-foundation-baseline-20261028.md)에
따라 기존 npm inventory/bin parser를 재사용하는 구조적 typed entry 선택만 추가한다. 기존
parser/runtime/검역 authority는 보존하며 현재 전체 검사·실행·CLI는 후속이다. CI422 source는
1203445로 고정하고 20분 타이머를 계속 소비한다. M13-004 하나만 active다.
종료 시험 ELF 사본18개/364359003B는 원본·전체 참조0/query errors0 확인 후 제거했으며 현재
바이너리/고정 입력·공유 cache/작은 raw는 보존했다. Raw p4-e-completed-copies-cleanup이 근거다.

### 2026-10-28 CI 대기 중 P5 baseline 독립 감사

[P5 baseline](../evidence/package-client-quarantine/p5-execution-foundation-baseline-20261028.md)은
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

[한정 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-e3-common-readiness-local-20261028.json)의
current UV4 152.771s/Bun exact4 165.549s/PNPM exact7 201.445s가 모두 PASS다. Public selectors,
transitive/peer exact quarantine·현재 Evidence/Policy와 Artifact 정상/exit37·long-name를 확인했다.
PNPM 첫4개는 isolated probes로 구분한다. Source1281/CPP/42input/ZIP·Host/Docker 불변,
helper각3wait0/부재·production client 복원/Host prefix 부재다. Fixture upper0700의 loader 실패와
trimpath/runtime.Caller 경로 실패를 보존했고 source를 바꾸지 않고 기존 CI recipe로 교정했다.
Local named scopes만 닫으며 current whole hosted/UID1001/macOS·final4profiles/signed 및
P5/P6는 미완료다. 같은 M13-004/C-e4에서 grouped candidate 검토·원격 qualification 준비를
계속한다. 새 push/CI/timer는 없다.

### 2026-10-28 C-e2 original ZIP/current graph/native readiness 한정 완료

[한정 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-e2-readiness-original-local-20261028.json)의
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

[진단 assessment](../evidence/package-client-quarantine/p4o4c3c3b3c-e1-cpu-diagnostic-assessment-20261028.json)의
원본42.703s/13.012223s와 동일657payload ZIP encoding45.627s/13.127181s는 CPU 거부다.
Encoding 후보는 채택하지 않았다. 원본 ZIP의 별도 phase 계측88.730s는 기존20unit 정상→실제
Read 뒤 오류 주입/zero commands→20unit 정상으로 PASS이며 CPU12.815358/12.873832s다.
관측/종료 구간과 kernel user/system을 얻었고 원인 수리·일정 성능 보장은 하지 않는다.
Source1280/CPP/42input/원본 ZIP·Host/Docker 불변/helperwait0·부재다. Product 변경·예산 완화는
없으며 legacy production/current whole qualification은 OPEN이다. [Owner matrix](../evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)에 따라
C-e2의 grouped hosted/current whole acceptance 준비를 계속한다. 새 push/CI/timer는 없다.

### 2026-10-28 C-d2 existing hosted tools wiring 한정 완료

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-d2-hosted-tools-wiring-local-20261028.json)의
v3 Quick280.009s/영향4package Race188.792s/tag/docs/security와 unchanged checker Race39.590s와 exact CI tools-native 154.349s/
세 parent, tools-bootstrap 573.679s/네 parent/noSKIP가 PASS다. Existing job의 두
client/helper owner에 fixed namespace ZIP·ordinary UID·pinned interpreter DATA를 연결했다.
Short native의 -E/원래 RECORD를 독립 wheel·EP로 증명했다. 첫 frozen supply 누락/overlay
cleanup NO와 short 원본 거부·진단/복구는 보존하며 최종 Host/입력/source 불변과
helperwait0·workspace 부재를 확인했다. Hosted/UID1001·final/signed/legacy CPU gap은 별도다.
같은 M13-004/C-e1에서 남은 acceptance/matrix와 CPU owner audit를 계속한다. 새 push/CI는 없다.

### 2026-10-28 C-d1 fixed native pipx original 한정 완료

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-d1-native-original-local-20261028.json)의
Quick227.569s/영향4package 전체 Race179.722s/tag/docs/security 및 source45 actual366.291s
UV·pipx 두 parent가 PASS다. Genuine pipx raw exact6.0 원본의 독립 whole source/script/interpreter
증명 뒤 bare6.1 반영, native11/managed9controls·새 현재 Evidence, bounded native6.1 no-op와
원본/현재 Evidence 실패 보존을 확인했다. 기존 UV native10/8도 PASS이며 source/Host/입력/정리는
모두 확인했다. Native 기본 recipe/run·hosted/final/signed 및 legacy ZIP CPU gap은 남는다.
같은 M13-004/C-d2에서 기존 hosted 소비자의 grouped 연결 audit를 계속한다.

### 2026-10-28 C-c3 public production pipx CLI 한정 완료

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-c3-public-cli-local-20261028.json)의
최종 Quick63.237s/tag/docs/security·기존 전체 CLI/bootstrap Race27.752s 및 source36 actual374.394s
네 parent가 PASS다. 실제 UV·pipx JSON/한국어 human·spaces fresh/control6와 managed/control8·9,
현재 승인 Manifest·receipt/link/user data 및 무변경 거부를 확인했다. Distinct pipx factory와
명시 lifetime owner를 연결하고 English help도 descriptor와 대조했다. Native adoption/run/hosted/
final은 별도이며 같은 M13-004/C-d1의 독립 원본 source/script/interpreter proof를 계속한다.

### 2026-10-28 C-c2 managed approved pipx backend 한정 완료

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-c2-managed-approved-backend-local-20261028.json)의
Quick181.693s/영향3package 전체 Race170.689s/tag/docs/security와 actual311.132s가 PASS다.
실제 승인한6.0 원본에서 bare upgrade6.1 및 same-version의 current 검사/Evidence·Policy/control9를
확인했다. Current Evidence write 실패는 prior ALLOW를 우회하지 못하고 기존 inode/bytes/links를
보존했다. 기존 UV managed·Evidence 거부도 PASS이며 여섯 공통 owner는 unchanged다. Production
CLI/native adoption/hosted/final은 별도이고 같은 M13-004/C-c3의 CLI/factory를 계속한다.

### 2026-10-28 C-c1 fresh approved pipx backend 한정 완료

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-c1-fresh-approved-backend-local-20261028.json)의
Quick147.053s/영향3package 전체 Race225.525s/tag/docs/security와 actual128.572s가 PASS다.
기존 UV·새 pipx의 plain/space 네 설치는 current whole 검사/Evidence·Policy ALLOW/control6와
동일 승인 반영·user data/Host 보존 및 fresh 재진입 무변경 거부를 확인했다. 기존 공통 엔진을
재사용하고 internal app path와 외부 link 목적지를 분리했다. 실제 composition은 test fixture이며
production CLI/managed·native original/hosted/final은 별도다. 같은 M13-004/C-c2를 계속한다.

### 2026-10-28 C-b3b original-aware native upgrade/no-op 한정 완료

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-b3b-original-upgrade-local-20261028.json)의
Quick95.439s/전체 sandbox Race25.087s/tag/docs/security·CPP128 및 actual91.332s가 PASS다.
Fresh6.0→upgrade6.1→same6.1 no-op의 whole source/control4·7·7과 typed 세션3개/정리·Host
불변을 확인했다. 최초 output scaffold 거부는 보존하고 기존 pinned UV venv owner를 재사용했다.
Selection은 승인 전 상태이며 같은 M13-004/C-c에서 current 검사·Policy/cache를 연결한다.
Guarded Host transaction/CLI·hosted/final·독립 legacy ZIP CPU gap은 남는다.

### 2026-10-28 C-b3a inert original replay/port 한정 검증

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-b3a-inert-original-replay-local-20261028.json)의
Quick206.627s/전체 sandbox Race26.962s/tag/docs/security가 PASS다. 기존 UV serializer는
byte-identical하게 재사용하고 pipx 원본에는 generated metadata/scaffold·inert command만
넣었다. Exact tar/mode/RECORD·invalid/cancel/collision/변조 command0를 검증했으며 actual
original-aware upgrade/Policy/CLI는 미완료다. 같은 M13-004/C-b3b에서 연결과 실제 검증을 계속한다.

### 2026-10-28 C-b2 public pipx selection/factory 한정 완료

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-b2-public-native-selection-local-20261028.json)의
Quick86.100s/전체 sandbox Race27.758s/tag/docs/security와 actual51.055s가 PASS다. Cowsay6.1
public 원본 URL/SHA와 native graph1/command1/control4를 결합했고 idna3.11은 native transaction
뒤 zero selection으로 거부했다. Typed Create/Register/Close각2·no observer fault/Host·원본·source
불변/helperwait0/final0다. Fresh selection만 qualified이며 같은 M13-004/C-b3 original upgrade,
current 검사/Policy·cache·Host transaction/CLI·최종 profiles와 legacy CPU gap은 남는다.

### 2026-10-28 C-b1 고정 설치 공급물 entry 한정 완료

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c-b1-installed-native-local-20261028.json)의
Quick104.819s/전체 sandbox Race24.782s/tag/docs/security 및 same CPP18 actual16.823s가
PASS다. Root-owned0444/고정 current ZIP를 ordinary caller가 읽어 fresh controls2/graph2와
회계 거부 target0을 확인했다. 첫 overlay0700 launcher 오류는 제품 미도달로 보존했고
fixture0755 guard 뒤 정상이다. Host·원본·source 불변/helperwait0/final0이며 C-b2 public
selection/factory·original upgrade/current Policy/CLI/final profiles·legacy CPU gap은 OPEN이다.

### 2026-10-28 C-a 실제 native metadata 한정 완료

[로컬 receipt](../evidence/package-client-quarantine/p4o4c3c3b3c1-native-session-local-20261028.json)의
CPP128/actual17.003s는 fresh controls2/graph2·CPU2.128257s와 genuine accounting-refusal의
target0/zero selection을 검증했다. Source·고정42/ZIP·Host5/Docker 불변, helperwait0/final0·
부재다. C-a named offline scope만 qualified이며 C-b installed supply/public caller·original
upgrade/current Policy/CLI/final profiles와 기존 ZIP mixed CPU parent는 OPEN이다. 같은
M13-004에서 다음 missing audit/연결을 계속하며 새 push/CI/timer는 없다.

### 2026-10-28 C-a query/diagnostic 중간 검증

[한정 checkpoint](../evidence/package-client-quarantine/p4o4c3c3b3c1-native-query-diagnostics-checkpoint-20261028.json)의
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
wait0/부재를 확인했고 원본은 보존했다. [Owner map](../evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)의
source 전 tuple audit에 따라 필요한 native-only read만 검증한다. [중간 구현/checkpoint](../evidence/package-client-quarantine/p4o4c3c3b3c1-native-session-checkpoint-20261028.json)의
Go canonical/전체 영향 Race·CPP128는 통과했고 실제 native install은 미완료다. C-a QUALIFIED/
ACCEPTANCE_CLOSED는 NO, installed supply·upgrade/current Policy·CLI는 후속 MISSING이다.

### 2026-10-28 CI421 Required10 성공·원본 대조

[CI421 qualification](../evidence/package-client-quarantine/ci421-pipx-zip-primitives-hosted-qualification-20261028.json)의
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

[C3c3b3b receipt](../evidence/package-client-quarantine/p4o4c3c3b3b-native-observer-local-20261028.json)의
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

[C3c3b3a receipt](../evidence/package-client-quarantine/p4o4c3c3b3a-offline-runtime-local-20261028.json)의
Quick390.740s/전체 영향 Race24.889s/tag vet·build/docs/security와 actual12.907s가 PASS다.
고정 network none/cwd `/`/환경·sealed RO inputs와 same-frozen-ID output을 결합하고
기존 memory/tmp를 유지·per-process CPU를30으로 좁혔다. Actual은 runtime flag만 runc로
바꿔 원본 control/native install/no-op·RO/불변·정리를 확인했다. New Go6mount/default
trace와 old UV/ZIP4mount 분리를 검사했지만 CPP native profile/실제 session/Policy/CLI는
미완료다. 같은 M13-004/C3c3b3b에서 기존 CONTROL 권한을 넓히지 않는 별도 ARTIFACT
chain과 negative latches를 계속한다. 기존 push 승인 대기이며 새 CI/타이머는 없다.

### 2026-10-28 P4-O4-C3c3b2 complete native output 로컬 검증

[C3c3b2 receipt](../evidence/package-client-quarantine/p4o4c3c3b2-complete-native-output-local-20261028.json)의
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

[C3c3b1 receipt](../evidence/package-client-quarantine/p4o4c3c3b1-native-census-local-20261028.json)의
Quick57.091s/tag vet·build/docs/security와 actual12.554s가 PASS다. 기존 production/default
test bytes를 유지하고 bounded 전체 native tree의 facts/hash만 수집했다. Install84/508193B,
no-op85/510597B이며 차이는 새 로그1개와 pipx metadata다. Exact selected source5 SHA와
query의 실제 -c/import 순서도 대조했다. NoSKIP/Host5/Docker/42input/ZIP/source 불변과
정리를 확인했으며 data census는 production allowlist/observer/Policy가 아니다. 같은
M13-004/C3c3b2의 전용 complete output reader를 계속하고 기존 push 승인은 대기다.

### 2026-10-28 P4-O4-C3c3a frozen whole wheel/RO input 로컬 검증

[C3c3a receipt](../evidence/package-client-quarantine/p4o4c3c3a-frozen-readonly-wheels-local-20261028.json)의
Quick248.619s/전체 sandbox Race16.476s·PyPI cached/tag vet·build/docs/security와 실제
17.977s가 PASS다. Whole original/source/startup gate와 root0:0/0444/noexec 별도 RO
wheel input, single-use writer→anchor→제거→seal/readers를 결합했다. 실제 기존 ZIP과
새 wheel의 root write EROFS·foreign RW 거부/제거 후 정상, native install/no-op·config
대조/metadata/link/graph2/replay, noSKIP·Host5/Docker/42input/ZIP/source 불변과 정리를
확인했다. C2 공통 두 source의 좁은 fixed-root diff와 나머지 네 source 불변을 기록했다.
Production installed current/native session·gVisor/Policy/CLI는 미완료이며 같은 M13-004의
C3c3b baseline/edit map을 계속한다. 기존 e859487 push는 승인 대기이고 새 CI는 없다.

### 2026-10-28 P4-O4-C3c2 native program/offline 진단 로컬 검증

[C3c2 receipt](../evidence/package-client-quarantine/p4o4c3c2-native-program-offline-local-20261028.json)의
최종 Quick33.889s/tag vet·build/docs/security와 actual9.800s가 PASS다. Same product/default
test SHA의 첫 sandbox 전체 Race27.155s와 최종 cached repeat를 구분했다. Runc ordinary1000
native version/install/no-op·canonical demotion·strict metadata/link/graph2/replay와 guest
malformed config negative/positive, noSKIP/Host5·Docker·42input·ZIP·source 불변/정리를 확인했다.
첫 fixture Config.User 소유권과 준비 copy 실패 원본을 보존하고 product guard는 유지했다.
Native installed current/gVisor/Policy/CLI 성공은 아니며 diagnostic RW wheelhouse도 생산 RO
input으로 승계하지 않는다. 같은 M13-004/C3c3의 installed supply/RO input/session/profile
baseline audit를 계속한다. 새 CI는 없고 기존 e859487 payload push만 사용자 승인 대기다.

### 2026-10-28 P4-O4-C3c1 조회 전 whole wheel gate 로컬 검증

[C3c1 receipt](../evidence/package-client-quarantine/p4o4c3c1-prequery-wheel-gate-local-20261028.json)의
Quick251.469s/전체 PyPI Race9.934s/tag vet/docs/security와 원본 corpus2.640s가 PASS다.
Receipt 이전에 기존 whole archive/source/graph와 startup surface 거부를 연결했고 기존 UV/
fixed supply 의미를 유지했다. 42input hash/readback·원본 userpath/click graph2·변조 거부는
정적 parser-binding fixture 범위이며 native solver/query/Policy 성공이 아니다. 같은
M13-004/C3c2 fixed native program/argv/offline actual diagnostic을 계속한다. 기존 e859487
checkpoint push는 승인 대기이며 새 CI는 없다.

### 2026-10-28 P4-O4-C3b pipx original upgrade 입력 로컬 검증

[C3b receipt](../evidence/package-client-quarantine/p4o4c3b-original-upgrade-input-local-20261028.json)의
Quick248.212s/전체 PyPI Race9.898s/docs/security는 PASS다. 원본 constraint/version과 새로운
bare-project 요청, recipe/control/version/runtime 거부 및 complete METADATA/WHEEL replay를
검증했다. 기존 UV validation body/control·의미를 보존했다. Native query·ownership/current
Policy/CLI 성공은 아니다. 같은 M13-004/C3c1의 authenticated whole-wheel pre-query startup
surface gate를 계속한다. 새 CI는 없고 기존 e859487 payload push만 사용자 승인 대기다.

### 2026-10-28 P4-O4-C3a pipx inert layout 로컬 검증

[C3a receipt](../evidence/package-client-quarantine/p4o4c3a-receipt-layout-local-20261028.json)는
기존 parser/held layout를 재사용한 원본 receipt semantics·source interpreter 분리·opaque
recipe/생략·null 보존과 unsafe/changed claim 거부를 확인했다. Quick40.768s/PyPI 전체
Race9.298s/docs/security PASS이며 첫 fixture 타입 오류를 보존했다. Native/CLI·ownership/
current Policy 성공은 아니다. 같은 M13-004/C3b의 bounded metadata-only original input과
replay 검증 재사용을 계속한다. 새 CI는 없고 e859487 push 승인은 계속 대기한다.

### 2026-10-28 P4-O4-B3d 기존 hosted actual 소비자 연결·로컬 recipe 검증

[B3d receipt](../evidence/package-client-quarantine/p4o4b3d-hosted-actual-wiring-local-20261028.json)는
기존 Linux job에 fixed42input inspect/graph/namespace와 owned client/helper/production restore를
연결했다. Actual135.807s/로컬UID1000의6개 exact top-level PASS/noSKIP·helper4wait0/confirmed
cleanup·client 복원/입력 directory0·Host5파일/Docker/소스 동일을 확인했다. Framework
empty/SKIP/child-only/duplicate/exit와 준비·cleanup 오류의 Bash 회귀, 최종 Quick47.782s/
checker 전체 Race40.351s·workflow/release-gate/docs/security도 PASS다. Logical fixture 입력
누락·operator FD bind의 최초 실패는 보존했고 제품 guard/예산은 완화하지 않았다.
HostedUID1001/새 CI·native pipx/signed activation/full matrix는 아직 미완료다.
동일 M13-004의 C3 installed pipx runtime/native install·upgrade/CLI owner audit를 계속한다.

### 2026-10-28 P4-O4-B3c 실제 graph·Evidence와 DATA 복구 로컬 검증

[B3c receipt](../evidence/package-client-quarantine/p4o4b3c-current-graph-data-recovery-local-20261028.json)는
실제 selected ZIP20unit→13subjects/32checks/37Evidence의 current store readback·Policy/staging,
missing runner 차단·unsigned production SDK 거부를 확인했다. Native6 sandbox/promoter는
fixture 범위다. 실제12payload255942560B root stage와 DATA copy/부분 복구/변조·foreign 보존,
legacy bootstrap/stage·pending24case/SIGKILL이 PASS다. DATA0555 partial copy cleanup 허용을
같은 test SHA로 RED→GREEN 검증하고 기존 copier mode helper를 cleanup에 재사용했다.
Host5파일/Docker/입력 불변/noSKIP·helperwait0/confirmed cleanup이며 actual 서명·Host activation
또는 hosted/full profile 완료가 아니다. B3d hosted actual 입력/소비자 연결부터 계속한다.

### 2026-10-28 P4-O4-B3b complete DATA producer·metadata·package 로컬 검증

[B3b receipt](../evidence/package-client-quarantine/p4o4b3b-producer-data-local-20261028.json)는
frozen42input 취득→기존 offline materializer→명시 runtime metadata v3/3tools→15asset/19bundle을
검증했다. Actual original ZIP의11distribution과 SBOM 일치, DEB/RPM 원본 DATA root0444·
재생성 byte-identical·Host 보호5파일/Docker/입력 불변/noSKIP를 확인했다. 기본 Quick와 관련
전체 Race, workflow/release-gate/docs/security 및 명시 actual tests가 PASS다. 로컬 unsigned
fixture이며 서명·hosted qualification·패키지 설치·artifact 실행 성공이 아니다. 같은 M13-004의
B3c actual complete graph/current Evidence·Policy와 protected DATA 복구를 계속한다.
원격 e859487 checkpoint는 기존 사용자 승인 대기로 새 CI/timer를 시작하지 않았다.

### 2026-10-28 P4-O4-B3a complete DATA 선택·mode 로컬 검증

[B3a receipt](../evidence/package-client-quarantine/p4o4b3a-complete-data-selection-local-20261028.json)는
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

[B2d receipt](../evidence/package-client-quarantine/p4o4b2d-required-runtime-inspection-local-20261028.json)는
최종 Quick42.543s/inspector Race2.132s, affected4 package full Race51.296s와 selected ZIP/
docs/security, 실제 required inspection41.606s를 연결한다. 원본 HAA identity의 required
check1/Finding0/Evidence6·commands5와 후속 intake 변조 거부를 확인했다. Import cycle·
summary1024B·ID64B와 native capability의 첫 실패 원본을 보존하고 기존 경계/한도를 유지했다.
NoSKIP/helperwait0/Host 동일이며 source는 공통 hosted checkpoint로 묶는다. 새 ZIP actual은
현재 named local이며 기존 CI의 hosted 소비자 범위와 구분한다. 같은 M13-004/B3에서 complete
release DATA membership/build/sign/install/recovery를 계속한다. Whole pipx/CLI·final profiles는 남는다.

### 2026-10-28 P4-O4-B2c acquired ZIP 실제 transaction 로컬 검증

[B2c receipt](../evidence/package-client-quarantine/p4o4b2c-acquired-runtime-transaction-local-20261028.json)는
최종 canonical 검사·sandbox 전체 Race, canonical124latches와 새 ZIP actual87.436s의
20units 정상→회계 오류 주입 INCOMPLETE/commands0→20units 정상, 기존 wheel/네 작은 policy·
prerequisite sequence151.871s를 연결한다. 각 정상 Register22/Close1·CPU12.76/12.47s이며
한도는 유지했다. 첫 준비 실패의 shell resolved path를 정확한 pinned tuple로 교정했고
두 원본을 보존했다. NoSKIP/helperwait0/confirmed cleanup·보호5파일/Docker/원본 ZIP 동일이다.
Named local scope만 닫고 required release composite/factory는 같은 M13-004/B2d에서 계속한다.
Hosted/common/full CPU·CUDA 및 전체 pipx/CLI acceptance는 남는다. 개별 helper CI는 없다.

### 2026-10-28 P4-O4-B2c 공통 transaction 연결·readiness 교정 중

[B2c owner map](../evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md#b2c--기존-transaction-연결의-source-전-세부-지도)에 따라 기존 원장/CPU watch/unit loop/cleanup을
공유하고 acquired ZIP은 기존 C2 DATA seed/bind/seal만 사용한다. 첫 Go Quick202.663s/
Race22.662s PASS 뒤 source 대조에서 공통 readiness CONTROL launch와 B2b의 일괄 launch
거부 충돌을 확정했다. B2b의 해당 규칙은 superseded이며 canonical exact readiness tuple만
허용하는 후보의 Quick53.090s/Race22.217s/docs/security PASS를 확인했다. 실제 gVisor
transaction과 교정 observer의 canonical latches는 아직 진행 중이다. 작은 helper별 CI는
추가하지 않고 같은 M13-004 하나에서 계속한다.

### 2026-10-28 P4-O4-B2b fixed observer profile 로컬 검증

[B2b receipt](../evidence/package-client-quarantine/p4o4b2b-observer-profile-local-20261028.json)는
새 profile/topology·동일 shared budget의 Quick204.610s/sandbox Race25.051s/docs/security와
canonical observer build274.583s/전체124latches PASS를 확인했다. 기존 wheel profile을
유지하며 새 ZIP은 단일 DATA path와 ARTIFACT Python handoff를 사용한다. 최초의 pip/site·
CONTROL launch/ELF handoff 거부 중 CONTROL launch 일괄 거부는 B2c에서 반증되어 교정 중이다. Host helper 설치·활성화와 실제 ZIP transaction 완료는
아직 없으며 같은 M13-004/B2c에서 기존 실행/회계/정리 owner를 연결한다.

### 2026-10-28 P4-O4-B2a 입력과 고정 program 로컬 검증

[B2a receipt](../evidence/package-client-quarantine/p4o4b2a-acquired-input-program-local-20261028.json)는
원본 HAA acquired identity/handle와 immutable ZIP DATA·20unit의 결합, 기존 C1 transport
재사용을 확인했다. Quick239.916s/sandbox Race25.955s/actual6.716s/docs/security PASS와
새 고정 program의 runc20 exit0·사전 거부3개/보호5파일·Docker·원본 불변을 확인했다.
이는 source/input/program 범위이며 외부 observer·실제 gVisor transaction 완료가 아니다.
같은 M13-004/B2b의 fixed profile/topology로 이어가고 작은 helper 단독 CI는 실행하지 않는다.

### 2026-10-28 CI420 최종 확인과20분 직접 조회 요청

[CI420 qualification](../evidence/package-client-quarantine/ci420-pytest-supply-hosted-qualification-20261028.json)은
Required 포함10SUCCESS와 all10 checkout729b/source1e5d21e tree391df742/2772 동일을
확인했다. 시작13:48:39UTC→Required15:02:23UTC는73분44초, Linux73분7초다.
원본10logs1,163,500B의 readback/hash와 Linux79PASS/noSKIP·CPU248.37s·helper11wait0/
최종 confirmed cleanup을 대조했다. 기존 공통 공급/checker와 hosted 소비자 범위이며 새 ZIP
gVisor 외부 관찰/전체 pipx 지원 완료는 남는다. CI415의 역사적 FD 원인은 계속 OPEN이다.

사용자는 이후 CI를20분 타이머로 직접 조회하고 작업을 계속하도록 요청했다.
[검증·감시 규칙](../20-package-client-quarantine-plan.md#5-검증-matrix와-자원-정리)의 간격을
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
기다리는 동안 [B2 수정 전 대조](../evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md#b2--새-공급-뒤-실행-owner의-수정-전-대조)를
진행했다. 기존 transaction/cleanup을 재사용하고 wheel profile의 pip·/haa-site 권한을
새 ZIP에 자동 승계하지 않는다. 실행 중 후보의 source를 추가 수정하거나 재push하지 않았다.
원본 initial run/jobs readback과 source/tree/checks는 local ci420-start-scope가 소유한다.

### 2026-10-27 P4-O4-B1d 명시 공급과 로컬 소비자 검증

[D2/D3 receipt](../evidence/package-client-quarantine/p4o4b1d23-pytest-runtime-supply-local-20261027.json)는
11wheel/42inputs/657원본 파일·새 Deflate ZIP과15required/5post-install plan, 실제 frozen
유지보수·HAA identity·DATA 전달·보호 파일/Docker 불변을 확인했다. Fixed Python runc의
15필수 imports와 기존 pytest plugin도 exit0이다. Gitleaks는 공식 metadata MD5 두 건을
오탐했으며 exact 원본 SHA256/path/rule/위치 확인과 다른 current/history token 검출 회귀를
통과했다. Quick187.424s/checker Race36.759s/security와 sandbox Race/workflow/release-gate도
PASS다. Common owner/checker hosted 회귀로 이어가며 B2 외부 관찰·release/native pipx/CLI와
전체 지원 완료는 남긴다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-O4-B1d 승인된 공급 확대 착수

[B1d baseline/수정 지도](../evidence/package-client-quarantine/p4o4b1d-pytest-runtime-supply-plan-20261027.md)에
따라 기존 publisher·whole wheel·single-root graph를 재사용해 두 명시 runtime 공급 root를
검증한다. pytest/iniconfig/pluggy의 제공 proof와 Pygments의 proof 부재를 구분하고, 전체 원본을
보존하는 Deflate 표현으로 기존8MiB DATA 전송 용량을 유지한다. 새 source/owner/pin은 아직
qualification 전이며 기존 필수 검사·resource budget·Domain single-primary 계약은 유지한다.

### 2026-10-27 P4-O4-B1b/c 검증과 명시 공급 확대 승인

[B1b/c receipt](../evidence/package-client-quarantine/p4o4b1bc-pipx-plan-intake-local-20261027.json)는
Quick44.492s/관련 전체 Race39.114s/actual0.895s/docs/security PASS,7 distributions/
9required·3post-install 계획과 원본 HAA identity/handle·bounded growth/cancel 거부를 확인했다.
고정 Python runc에서 loader/base import는 exit0, pytest plugin은 pytest 부재 exit1이다.
사용자가 권장안인 pytest와 의존성의 명시 runtime 공급 추가·새 전체 검증을 승인했다.
Required 범위는 유지하고 자동 보조 설치·plugin 선택 검사 전환은 하지 않는다.
같은 item의 B1d 공급 baseline/official 입력 확인을 계속한다. 새 runtime/observer·release와
전체 pipx 지원은 아직 미완료이며 이 로컬 helper 묶음만으로 CI를 반복 실행하지 않는다.

### 2026-10-27 P4-O4-B1a ZIP snapshot 로컬 검증

[B1a receipt](../evidence/package-client-quarantine/p4o4b1a-pipx-zip-input-local-20261027.json)는
Quick256.943s/PyPI 전체 Race8.661s/actual2.347s/docs/security PASS를 연결한다.
Selected2505814B/190files의 immutable copy와 변조·잘림·추가 바이트 거부를 확인했다.
Python은 실행하지 않았으며 original HAA intake binding·surface/typed plan·observer는 남는다.
같은 M13-004/B1의 다음 입력 연결을 계속하며 이 helper 단독으로 push하지 않는다.

### 2026-10-27 P4-O4-B1a 착수

[기존 B1 수정 지도](../evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)의
세부 baseline에 따라 acquired ZIP immutable bytes/file facts부터 연결한다. Fixed selected pin과
원본 HAA identity를 보존하며 Python surface/typed plan·동적 관찰·release 전체 연결은 남는다.
기존 supply/wheel parser를 재사용하고 M13-004 하나만 IN_PROGRESS로 유지한다.

### 2026-10-27 CI419 최종 결과 확인

[CI419 qualification](../evidence/package-client-quarantine/ci419-pipx-data-hosted-qualification-20261027.json)은
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
기존 hosted 소비자 회귀 결과를 기다리는 동안 [B1 입력/실행 owner 대조](../evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)를
계속한다. 신규 actual pipx DATA 시험은 로컬 explicit fixture이며 이번 CI가 Python ZIP
실행이나 전체 pipx 지원을 검증한다고 표시하지 않는다. 작은 local 문서 변경을 재push해
실행 중 후보를 취소하지 않는다.

### 2026-10-27 P4-O4-C2 한정 로컬 검증

[C2 receipt](../evidence/package-client-quarantine/p4o4c2-pipx-readonly-volume-local-20261027.json)는
Quick258.661s/sandbox 전체 Race23.229s/docs/security/workflow PASS와 actual DATA5.020s를
연결한다. Root preparation→ordinary RO anchor/reader, writer 제거 전 seal 거부,
foreign RW attachment 거부와 root write EROFS·동일 ZIP/보호5파일/Docker 정리를 확인했다.
첫 두 readback 실패는 fixture writer UID1000의 실제 tar owner로 좁혀 원본을 보존했다.
별도 format cwd 오류의 raw4개는 역사적 원본 bytes로 복구했다. Python/ZIP 코드 실행은
없고 gVisor/observer·release·native pipx/CLI 및 전체 P4 acceptance는 남는다.
공통 volume 변경의 기존 hosted 소비자 회귀를 다음 묶음으로 확인하고 B1을 준비한다.

### 2026-10-27 P4-O4-C2 착수

[C2 baseline/수정 지도](../evidence/package-client-quarantine/p4o4c2-pipx-readonly-volume-plan-20261027.md)에
따라 기존 closure volume의 identity/attachment owner와 C1 DATA 공급을 연결한다.
Writer 제거와 live readonly anchor/consumer·실제 byte readback·정리를 검증하며
runtime/observer·release·native pipx/CLI 완료는 남는다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-O4-C1 한정 로컬 검증

[C1 receipt](../evidence/package-client-quarantine/p4o4c1-pipx-data-transport-local-20261027.json)는
fixed DATA reader/transport의 Quick230.736s/sandbox 전체 Race25.658s/docs/security PASS와
actual ZIP2505814B의 disposable namespace 보호 파일·15개 거부0.149s/noSKIP·원본 불변을
연결한다. Actual은 DATA tar roundtrip이며 Docker/Python 실행이나 readonly mount 완료가
아니다. 기존 UV/PNPM/Bun을 유지하고 다음 C2 실제 mount lifecycle로 이어간다.

### 2026-10-27 P4-O4-C1 착수

[C1 baseline/수정 지도](../evidence/package-client-quarantine/p4o4c1-pipx-data-transport-plan-20261027.md)에
따라 기존 held file reader/typed transport에 fixed pipx DATA를 연결한다. 기존 Python
preparation/anchor readonly owner는 다음 C2에서 재사용한다. Public runtime/CLI와 release
acceptance는 열려 있으며 M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 CI418 원격 검증 확인

[CI418 qualification](../evidence/package-client-quarantine/ci418-pipx-owner-hosted-qualification-20261027.json)에서
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

[O4-A receipt](../evidence/package-client-quarantine/p4o4a-pipx-runtime-owner-local-20261027.json)는
단일 selected owner/frozen maintenance의 실제 공급·apply 중단 복구·변조 거부를 연결한다.
최종 NOTICE 생성기 교정 후보 Quick248.69s/affected Race35.863s/actual4.637s가 PASS다.
이전 후보 scripts 전체 Race119.715s와 canonical 결과는 이전 bytes 범위로 보존했다.
ZIP/NOTICE0444·선택 hash/원본27입력 불변이며 release13assets/2tools는 유지된다.
O4-B active Python ZIP 검사·배포·activation과 O4-C typed transport/actual gVisor·observer·
native/CLI가 남는다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-O4 착수

[O4 baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p4o4-pipx-runtime-owner-plan-20261027.md)에
따라 기존 runtime lock/generator/maintenance/release·installed reader를 재사용한다.
새 pipx 공급 owner와 DATA ZIP role을 연결하며 native/observer·CLI acceptance는 열어 둔다.

### 2026-10-27 P4-O3 한정 로컬 검증

[O3 receipt](../evidence/package-client-quarantine/p4o3-pipx-loader-local-20261027.json)의
Quick221.019s/affected Race37.284s/docs·security와 실제 ZIP 생성·원본190파일 대조,
결정성·기존 출력 보존·frozen27파일 불변을 확인했다. Fixed Python runc guest의
metadata/version/help도 PASS이며 보호5파일/Docker·candidate bytes는 동일하다.
GVisor/observer/native pipx/CLI 검증으로 확대하지 않고 O4 single owner 감사로 이어간다.

### 2026-10-27 P4-O3 착수

[O3 baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p4o3-pipx-loader-plan-20261027.md)에
따라 O2 held bytes와 typed manifest, 기존 no-replace publisher를 재사용한다.
원본 RECORD·license를 보존한 readonly ZIP 후보와 fixed Python isolated loader를 검증한다.
공통 runtime/observer·native install·upgrade/CLI acceptance는 별도 MISSING이다.

### 2026-10-27 P4-O2 한정 로컬 검증

[O2 receipt](../evidence/package-client-quarantine/p4o2-pipx-closure-local-20261027.json)의
Quick232.433s/affected 전체 Race77.646s/docs·security와 actual1.353s가 PASS다.
7 wheels/190파일/제공 proof6과 실제 변조7종 거부·frozen27파일 불변을 확인했다.
공급 검증만 닫으며 같은 M13-004에서 O3 immutable loader baseline으로 이어간다.
Runtime/native/CLI/hosted·전체 P4 acceptance는 미완료이고 새 push/full CI는 없다.

### 2026-10-27 P4-O2 착수

[O2 baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p4o2-pipx-closure-plan-20261027.md)에
따라 O1 private 검증 로직과 기존 whole wheel/metadata/marker/constraint/graph를 재사용한다.
제공 dependency proof5개의 fixed publisher 검증과 실제7-wheel closure를 연결하며 userpath
서명 미제공 경계를 유지한다. DATA-only whole RECORD 진단0.582s는 PASS였고 formal 공급·
immutable loader/runtime·native·CLI acceptance는 미완료다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-O1 한정 로컬 검증

[O1 receipt](../evidence/package-client-quarantine/p4o1-pipx-publish-local-20261027.json)의
canonical Quick180.173s·두 affected package 전체 Race30.077s·actual supply1.680s·
docs0.420s/security18.779s가 PASS다. Fixed pipx publisher/certificate environment·subject와
single wheel 실제 bytes를 검증하고 frozen 입력 불변을 확인했다. Initial formatter 순서와
SA1012 시험 표현 실패는 원본과 구분해 보존했다. O1 developer 공급 사실 검증만 닫으며
dependency closure/immutable loader/runtime·native·CLI/hosted/final matrix는 미완료다.
같은 M13-004에서 O2 edit map으로 이어가고 새 push/full CI는 없다.

### 2026-10-27 P4-O1 착수

[O1 baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p4o1-pipx-publish-plan-20261027.md)에
따라 기존 pinned Sigstore/root와 developer frozen reader를 재사용한다. 별도 fixed pipx
publisher의 bounded PEP740/certificate environment·subject/단일 wheel bytes 검증만 추가한다.
전체 dependency closure/immutable loader/runtime·native/CLI·P4 acceptance는 후속 missing이다.
M13-004 하나만 IN_PROGRESS이며 작은 helper별 새 push/full CI는 하지 않는다.

### 2026-10-27 CI417 최종 검증

[CI417 qualification](../evidence/package-client-quarantine/ci417-uv-tools-hosted-qualification-20261027.json)을
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
진단은 [기존 baseline](../evidence/package-client-quarantine/p4o-pipx-supply-runtime-baseline-20261027.md)에
기록했으며 runtime 공급·native pipx 지원은 계속 미완료다.

### 2026-10-27 CI416 실패 소비

[실패 조사 지도](../evidence/package-client-quarantine/ci416-native-fixture-uid-investigation-20261027.md)는
TestLinuxUVToolNativeAdoptionCLIIntegration의 원본 producer EACCES/0.77s를 소유한다.
기존 tool 네 소비자와 PNPM peer는 PASS이고 native current inspection은 NOT_REACHED다.
8SUCCESS/Linux·Required FAILURE, helperwait0/final CONFIRMED_STOPPED·RESTORED/primary1과
all10 같은 checkout/raw logs를 보존했다. Tool-only bind root와 고정 guest1000 관계를
non1000 ordinary user에서 재현하며 product guard·권한·예산은 바꾸지 않는다. CI 재실행은
원인/후보 교정·로컬 검증 뒤 실제 hosted acceptance 경계에서만 수행한다.

### 2026-10-27 P4-N2D 실제 native 인계

[N2D receipt](../evidence/package-client-quarantine/p4n2d-uv-native-tool-cli-local-20261027.json)는
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
독립 후속은 [P4-O pipx baseline](../evidence/package-client-quarantine/p4o-pipx-supply-runtime-baseline-20261027.md)의
공급 증거·runtime 역할 감사이며 M13-004 하나만 IN_PROGRESS다.

### 2026-10-27 P4-N2C 정전 후 로컬 체크포인트

[N2C receipt](../evidence/package-client-quarantine/p4n2c-uv-native-current-update-local-20261027.json)는
단독 Quick208.494s/affected full Race163.552s와 security/docs·canonical observer122latches,
기존 UV tool 네 actual 소비자376.837s를 연결한다. 명시 native enrollment와 raw/source controls,
synthetic 원본의 ledger 부재/원복·취소/late foreign cleanup, managed 반복 adopt의 현재 재검사를
확인했다. Helperwait0/Host 보호 파일·Docker 동일이며 정전 후 사라진 두 /run directory만 별도
복구했다. 최초 시험 준비 오류와 중첩 실행의 두300s timeout은 원본으로 보존한다. 실제 PUBLIC
NATIVE 원본의 current 승인/Manifest/CLI는 아직 미검증이며 [N2D 지도](../evidence/package-client-quarantine/p4n2d-uv-native-tool-cli-plan-20261027.md)와
그 실제 소비자로 이어간다. M13-004 하나만 IN_PROGRESS이고 새 push/full CI는 없다.

### 2026-10-24 CI402 ordinary checkpoint 확인

실제55분 타이머 만료 뒤 [CI402 결과](../evidence/package-client-quarantine/p2c2-pnpm-ci402-qualification-20261024.json)를
확인했다. Required 포함10SUCCESS이며 source940과 checkoutcab8183의 전체tree36c5d030이
같다. PNPM 실제4개·두 job의 observer110·Linux job 전체57 top-level PASS/CPU294.69s·
native macOS, helper5wait0/final CONFIRMED_STOPPED·original0/client_error0/client_restore0도
확인했다. 이 ordinary checkpoint는 앞940 범위이며 새 public8 fixture의 미실행을 성공으로
표시하거나 이후 로컬0f8/40d/C5 검증으로 전용하지 않는다. P2 qualification/acceptance는
여전히 미완료이고 native offline dependency/peer context를 같은 M13-004에서 이어간다.

### 2026-10-24 P2c3 archive 설치 동작 사실 전달 착수

[착수 감사·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2c3-pnpm-install-action-plan-20261024.md)에서
기존 npm archive/graph M4/정규화 Evidence를 재사용하고 rootless npm의 사실 전달 gap만
확인했다. 단일/acquired/batch 세 경로와 subject/check/저장 참조 결합, UNKNOWN·필요 동작의
수동 검토가 이번 범위다. Layout/transaction/CLI·전체 P2 qualification은 계속 미완료다.
앞 네 commit은 한 번 push했고 CI402/head940의55분 실제 타이머 첫 확인은14:57:13KST다.
현재 source와 다음 local 구현을 섞어 성공을 주장하거나 재push로 실행을 취소하지 않는다.
M13-004 하나만 IN_PROGRESS다.

후속 구현은 single/acquired/batch와 exact subject/check/reference 결합을 연결했다.
최종 Quick33.846s와 앞 관련 여섯 package Race5.283s, 실제 public archive 두 개의
SRI/static/Store/true·false fact0.936s가 PASS다. 새 UNCONFIRMED code는 기존 다국어
필수 검사 미완료 설명을 재사용한다. [로컬 결과](../evidence/package-client-quarantine/p2c3-pnpm-install-action-local-20261024.json)가
현재 입력과 범위를 소유한다. Remote402는 앞940 후보이므로 이번 source 성공으로 전용하지 않는다.
다음은 unique archive coverage와 full peer wiring을 함께 묶는 complete snapshot이며
전체 P2/Host 설치·CLI qualification은 계속 열려 있다.

P2c3은 로컬0f8ddcc로 보존했고 push하지 않았다. [P2c4 수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2c4-pnpm-project-snapshot-plan-20261024.md)를
완료하여 기존 complete Domain과 parser/source binding을 재사용하는 snapshot 생산을 이어간다.
Full peer instance·runtime/target/controls 결합과 explicit empty가 범위이며 native layout/
records의 peer provenance·transaction/CLI는 후속 acceptance로 유지한다.

후속 actual에서 native single/transitive의 snapshot과 single required3/refs4·M3/M4 ALLOW는
확인했다. Transitive is-odd3은 integrity/static/action fact 정상이나 기존 single-artifact
offline dynamic에서 M3_DYNAMIC_EXECUTION_FAILED/INCOMPLETE여서 수동 검토다.
[P2c4 실제 결과·진단 범위](../evidence/package-client-quarantine/p2c4-pnpm-project-snapshot-plan-20261024.md#5-실제-연결의-첫-거부와-남은-gap)에
Host불변/정리·미실행 후속과 stale overlay 진단 NOT_EXECUTED를 보존했다.
다음에는 npm에 존재하지 않는 frozen dependency/peer fixture의 정확한 연결 지도를 먼저
작성·재검토하고 이 gap을 닫은 뒤 layout/transaction/CLI로 진행한다. M13-004만 IN_PROGRESS다.

Snapshot은 로컬40d4e25로 보존했다. [P2c5-A native offline 착수 감사·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2c5-pnpm-offline-context-plan-20261024.md)를
완료했고 먼저 fixed file-source recipe와 실제 pinned native 배치를 검증한다. 이것은 test
prerequisite이며 production closure 인터페이스는 그 결과 뒤 정확히 확정한다. 현재 작은
local commit 둘은 push하지 않았고402 후보도 유지한다.

P2c5-A2의 transitive2는 [로컬 결과](../evidence/package-client-quarantine/p2c5-pnpm-offline-local-20261024.json)에서
원래 observer25의 actual8.591s/complete/금지0·Host불변/helperwait0로 검증했다. 초기 store
설정을 고정해 default root probe를 피했고 제품 관찰기·권한은 유지했다. Peer fixture와
production static-all/per-target closure가 다음 순서이며 새 실제 P2 검역 완료는 아니다.
완료한 namespace·진단 자료는 참조/query error0 확인 뒤 삭제했고 작은 원본만 보존했다.

이어 A3 [peer 실제 결과](../evidence/package-client-quarantine/p2c5-pnpm-peer-offline-local-20261024.json)에서
기존 exact-version source API·versioned fixture를 재사용한 transitive6.01s/peer8.01s와
peer4 identity·links5개/complete/금지0·Host불변/helperwait0를 확인했다. 일반 packument
한도에서 멈춘 첫 준비 실패는 원본에 남겼다. Production closure는 아직 미구현이며 B의
상세 수정 지도·두 번째 검토를 다음으로 진행한다. M13-004 하나만 IN_PROGRESS다.

[B의 착수 감사·B1 정확한 수정 지도/두 번째 검토](../evidence/package-client-quarantine/p2c5b-pnpm-production-context-plan-20261024.md)를
작성했다. 기존 snapshot/source binding/project batch를 재사용하고 frozen snapshot과 acquired
전체 집합·복수 peer instance를 잃지 않는 immutable 값을 먼저 연결한다. 이후 실제 lifecycle
선택·격리·관찰 aggregate의 의미를 검증하며 미확정 runner 인터페이스를 앞서 만들지 않는다.

B1 [로컬 결합 결과](../evidence/package-client-quarantine/p2c5b-pnpm-context-local-20261024.json)는
same-digest 위조 snapshot·획득 누락/혼입과 다른 run 거부, six archives/seven peer instances
보존·실제 producer wrapper 재사용을 확인했다. Canonical Quick/security/docs·관련 전체 Race는
PASS다. 기존 단일 npm 실행/관찰기/Policy는 불변이며 실제 PNPM dynamic은 계속 미완료다.

위 다섯 로컬 commit을 db464fd까지 한 번 push했다. Ordinary CI403/run40075179512가
시작됐으며 직전402 실측53m11에 따라55분 실제 타이머의 첫 확인은17:37:12KST다.
그 전에는 반복 조회·작은 변경 재push를 하지 않고 B2 설계/로컬 구현을 이어간다.

[B2 정확한 수정 지도·검토·로컬 결과](../evidence/package-client-quarantine/p2c5b2-pnpm-target-lifecycle-plan-20261024.md)는
기존 install 권한을 유지한 별도 target/topology·실제 clone lineage·one-shot과 C++112,
transitive/peer의 synthetic positive/exit37·기존8 actual 회귀를 확인했다. Production layout/
static-all/aggregate와 공개 프로젝트 검역 연결은 B3에서 이어가며 이 primitive를 ALLOW로
표시하지 않는다. B2는 아직 원격403 source에 포함되지 않았다. 같은 M13-004만 IN_PROGRESS다.

### 2026-10-24 P2c5-B3 archive/layout 착수 감사

[B3 착수 감사·B3a 수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2c5b3-pnpm-layout-plan-20261024.md)를
완료했다. 기존 정적 tar 순회에 선택적 per-file inventory를 더하고 기존 Inspect 결과는
보존한다. 최종 Quick/security/docs·세 package 전체 Race26.274s, 실제 public
is-number4/node-sass334 entries의 Acquire/SRI·원래 report/Evidence 일치도 PASS다. 전체 native layout/prepare/static-all/aggregate는 순차 후속이며 current product의
transitive INCOMPLETE와 전체 P2 acceptance NO를 유지한다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 P2c5-B3b native 배치 형태와 dirname 연결

[B3의 §6·§7](../evidence/package-client-quarantine/p2c5b3-pnpm-layout-plan-20261024.md#6-b3b0--전체-layout의-실제-표현-확인-source-작성-전-감사두-번째-검토)에서
정상 native의 identical LICENSE hardlink·private hoist/생성 Node bin shim을 실제 확인했다.
해시 없는 일반 hardlink 허용이나 bin 파일명만의 승인은 하지 않는다. B3b1의 official registry
codec/관련 Race5.807s와 fresh actual20.659s의 transitive2/peer4 모든 instance 경로 비교도
PASS다. 두 전용 runtime bulk는 모두 refs/query errors0 뒤 삭제했다. Full layout byte/edge/
bin/metadata verifier·production prepare/static-all/aggregate가 다음이며 P2 acceptance는 NO다.
M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 B3b2a layout envelope 로컬 검증

[B3 §8](../evidence/package-client-quarantine/p2c5b3-pnpm-layout-plan-20261024.md)은 bounded native
입력 parser만 연결했다. Deprecated Xattrs/무제한 Copy 표기의 두 정적 점검 거부를
PAXRecords/정확한 CopyN으로 교정했으며 security/Quick/docs·npm 전체 Race6.037s와
actual21.319s의 original controls/identity/links·complete/금지0/Host불변/helperwait0를
확인했다. 끝난 전용165,567,367B는 refs/query error0 뒤 삭제했다. 이 type에는 acquisition/
Policy/실행 권한이 없고 전체 archive/edge/bin/metadata 검증은 남아 있다. 현재404 source는
앞1f410fd이고55분 타이머19:30:18KST를 유지한다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 B3b2b Node bin code 로컬 검증

[B3 §9](../evidence/package-client-quarantine/p2c5b3-pnpm-layout-plan-20261024.md)은 upstream
전체 코드 형식을 한 owner에서 제한했다. Canonical format/security/Quick/docs·npm 전체
Race6.168s와 actual18.382s의 세 Node shim/명시적 target·원래 관찰/Host보존/helperwait0가
PASS다. 전용165,592,448B는 refs/query error0 뒤 삭제했다. 원본 archive의 bin/file/전체edge와
metadata binding·production prepare/static-all/aggregate는 남아 있으며 전체 P2 acceptance는
NO다. CI404 source는 앞1f410fd로 유지하고 새 push 없이19:30:18KST 타이머로 확인한다.

### 2026-10-24 B3c 전체 inspection layout 결합 착수 감사

[정확한 수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2c5b3c-pnpm-layout-binding-plan-20261024.md)를
작성했다. 기존 inventory/envelope/dirname/Node code 형식과 B1 input을 한 지점에서 결합한다.
모든 peer instance·원본 file/edge/bin·생성 metadata를 확인하며 inspection 값은 Policy/Host
승격 권한이 아니다. 기존 parser/단일 Backend를 다시 구현하지 않는다. 이후 native prepare/
static-all/per-instance aggregate와 Host/CLI는 그대로 MISSING이며 M13-004 하나만 IN_PROGRESS다.

후속 [로컬 결과](../evidence/package-client-quarantine/p2c5b3c-pnpm-layout-binding-local-20261024.json)의
fresh actual17.553s는 transitive2/peer4 archives와 모든 instance/file/edge·원본 bin3개를
검증하고 cloned native input 변조7/8개를 거부했다. 처음 누락된 native freshness cache는
pinned source를 확인해 정확한 regular bounded JSON accounting 후 inspection payload/
getters/digest에서 폐기했다. 첫 실패와 교정 source를 구분하고 original controls·complete
관찰/금지0·Host불변/helperwait0를 보존했다. 끝난 두 runtime 총331,938,972B는 refs/query
errors0 뒤 삭제했다. Production native prepare/static-all/fresh per-instance 실행/aggregate와
Host/CLI는 아직 미완료이며, 다음 B4의 정확한 edit map부터 같은 M13-004에서 이어간다.

### 2026-10-24 B4 production inspection 착수 감사

[Baseline audit·정확한 edit map·두 번째 검토](../evidence/package-client-quarantine/p2c5b4-pnpm-production-inspection-plan-20261024.md)를
작성했다. 먼저 기존 PNPMResolver의 수명 owner를 선택/준비가 재사용하고 complete input/
logical footprint·재소개 bytes·network detach/실제 endpoint0·관찰/정리 뒤 값 반환을 연결한다.
이어 verified payload의 최소 소개·fresh per-instance 실제 결과·partial 관찰/첫 오류 보존을
연결한다. 단일 npm·Domain/Application의 기존 의미는 유지하며 이 문서만으로 새 구현/qualification을
주장하지 않는다. M13-004 하나만 IN_PROGRESS이고 PNPM product dynamic/전체 P2는 미완료다.

후속 B4a의 [로컬 결과](../evidence/package-client-quarantine/p2c5b4a-pnpm-production-prepare-local-20261024.json)는
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
native 동작은 유지하며 [직렬화 교정](../evidence/package-client-quarantine/p2c5b4b-pnpm-fingerprint-local-20261024.json)을
새 Artifact 실행 완료로 확대하지 않는다. 다음은 예정대로 verified payload와 fresh instance backend다.

### 2026-10-24 B4b1 실행용 payload 재구성 착수 감사

[Baseline audit·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2c5b4b1-pnpm-inspection-payload-plan-20261024.md)에서
기존 bound layout/private manifest와 bounded parser를 재사용한다. Raw native whole tar/
원본 user controls를 소개하지 않고 불필요 metadata를 제외하며 file bytes·link·공유 inode
partition을 보존한 새 deterministic tar와 project 전체 readback을 구현한다. Node_modules와
manifest만 검사해 root extra 설정 파일을 놓치지 않는다. 이어지는 fresh backend가 actual
consumer이며 이번 renderer만으로 Artifact 실행/Host/CLI·전체 P2 완료를 표시하지 않는다.

후속 source4개는 작성·format 확인했지만 Quick의 production build 두 번과 별도 focused
command가 각각 timeout이었다. Unit 성공이나 source finding은 확인되지 않았다. Host에서
codex PID1404의 RSS22,799,860KiB·MemAvailable534,648KiB·swap free0을 관측했으나 timeout의
단독 원인으로 확정하지 않는다. 반복 build는 중단하고 미검증 WIP와 [정확한 재개 위치](../evidence/package-client-quarantine/p2c5b4b1-pnpm-inspection-payload-plan-20261024.md#6-2026-10-24-host-자원-차단과-재시작-인계)를
보존한다. Host 자원을 회복한 뒤 같은 고정 검사·한도로 확인해야 해제할 수 있다.

2026-10-25 재개 후 [B4b1 로컬 결과](../evidence/package-client-quarantine/p2c5b4b1-pnpm-inspection-payload-local-20261025.json)는
동일한 보존 source의 canonical format/Quick/security/docs와 npm/sandbox 전체 Race25.352s를
확인했다. Minimal payload·전체 readback과 공유 inode 의미를 검증했으며 production backend는
다음 B4b2다. 과거 timeout/RSS 원인 미확정은 보존하고 전체 P2 acceptance NO를 유지한다.

[B4b2의 착수 감사·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2c5b4b2-pnpm-instance-backend-plan-20261025.md)에
따라 기존 fresh volume/target command/observer·partial 의심 사실 보존 helper를 연결했다.
[B4b2 로컬 결과](../evidence/package-client-quarantine/p2c5b4b2-pnpm-instance-backend-local-20261025.json)는
canonical Go/Race·C++112와 실제 public transitive2/peer4·controlled 정상/exit37, 별도 긴 이름
실행까지 PASS다. 각 실행의 complete trace·최초 실패와 사실 보존·helperwait0/정리·Host 보존을
확인했다. Named local consumer 범위만 닫고 B4c static-all/aggregate부터 이어간다. 전체 P2
acceptance와 Host/CLI·remote qualification은 NO다.

[B4c 착수 감사·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2c5b4c-pnpm-project-inspection-plan-20261025.md)를
기록했다. 기존 complete workspace gate를 같은 charge owner의 incremental counter로 확장하고,
이후 static-all/실제 instance별 Evidence와 오류 시 부분 사실 기록을 연결한다. Core의 미완료 check에
final Finding을 붙이지 않으며 관찰 해석 완료와 required lifecycle coverage를 별도로 보존한다.

[B4c 로컬 결과](../evidence/package-client-quarantine/p2c5b4c-pnpm-project-inspection-local-20261025.json)는
수집 중 고정 예산·전체 frozen layout 결합·실제 Session coverage/partial facts와 실패 시 exact
Evidence 기록을 연결했다. 최종 관련 네 package Race32.714s·canonical 검사/CI 호출 순서와
public transitive2/peer4 실제49.087s, native8 선택·네 실제 Policy/Evidence 검역114.917s가 PASS다.
각 실제 수명의 helperwait0/정리·Host 보존을 확인했고 Host 설치는 미수행이다. B4 production
inspection의 named local scope만 닫고 묶음 remote와 PNPM cache/provenance/Host records·
transaction/CLI를 이어간다. 같은 M13-004만 IN_PROGRESS이며 whole P2 acceptance는 NO다.

[B5a baseline audit·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2c5b5a-pnpm-approved-staging-plan-20261025.md)를
기록했다. 기존 stored Evidence reader와 source 재해시·sealed regular-file cache를 재사용해
승인된 PNPM layout을 비실행 data tar로 stage하는 gap부터 진행한다. Symlink Host 추출과
독립 retained approval/records·transaction·CLI는 다음 경계다. CI405의 frozen source는 유지한다.

[B5a 로컬 결과](../evidence/package-client-quarantine/p2c5b5a-pnpm-approved-staging-local-20261025.json)는
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

실제55분 타이머 만료 뒤 [CI404 결과](../evidence/package-client-quarantine/p2c5b3-pnpm-ci404-qualification-20261024.json)를
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

실제55분 타이머08:37:12UTC 만료 뒤 [CI403 결과](../evidence/package-client-quarantine/p2c5b-pnpm-ci403-qualification-20261024.json)를
확인했다. Source db464fd와 checkoutd5be7ee의 전체treeced85a91/2,173 files가 같고
Required 포함10SUCCESS다. Observer110씩·PNPM 실제4·Linux job 전체57 top-level/
CPU286.42s·native Go1.27.3 macOS와 helper5wait0/final original0 정리도 확인했다.
실제 elapsed51m17이며 B2의 새 observer112/target primitive 검증으로 전용하지 않는다.
다음 연결 단위를 로컬에서 계속 작업하고 작은 helper별 remote 재push는 하지 않는다.

### 2026-10-24 P2c2 actual public 선택·update/no-op 로컬 검증

[실제 public 결과](../evidence/package-client-quarantine/p2c2-pnpm-public-selection-local-20261024.json)는
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

[Producer 경계 결과](../evidence/package-client-quarantine/p2c2-pnpm-producer-boundary-local-20261024.json)는
add/install/update 사전 입력·허용된 manifest 변경, 기존 UV/PNPM exact owner 공유와
고정 lock-only command·전체 관찰/정리 후 결과 반환을 연결한다. Canonical Quick51.370s/
security/docs와 실제 empty/fault5cases31.01s가 PASS다. 앞 native notifier의 불필요한
registry 조회를 공식 source와 대조하여 false로 고정했고 임시 읽기 보완은 되돌렸다.
Observer product/binary는 qualified17/CI401과 같고 추가 거부 matrix110도 PASS다.
이는 fake/inert service의 closed-network 경계 검증이다. Actual public typed policy/HTTPS
single/transitive/peer·independent binding이 다음이며 Domain/HostInstallAction·layout/
transaction/CLI acceptance는 미완료다. M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 CI401 ordinary native admission 확인과 P2c2 전송 검증

실제 연장 타이머 뒤 [CI401 결과](../evidence/package-client-quarantine/p2b2-pnpm-ci401-qualification-20261024.json)를 확인했다.
Head343f03f와 checkout093cba1의 전체tree e735b1ec가 같으며 Required 포함10SUCCESS다.
새 PNPM actual4·observer110·기존 Linux47/CPU296.32s·native macOS와 helper5wait0/
최종 CONFIRMED_STOPPED·client_error0·restore0을 확인했다. P2b2 ordinary guest admission의
원격 gap만 닫는다. P2c1/e762와 이후 source는 로컬이며 native5 signed whole-update 및
P2 graph/layout/transaction/CLI acceptance는 계속 열려 있다.
P2c2 고정 project controls의 local Quick119.431s/security/docs PASS 뒤 같은 production
transport를 소비하는 empty-lock/Node actual과 manifest·lock의 metadata/크기를 검증한다.
새 fresh owned workroot의 actual8.884s도 PASS이며 [전송 결과](../evidence/package-client-quarantine/p2c2-pnpm-control-transport-local-20261024.json)로 범위를 연결했다.
마지막 소비 뒤 참조/query errors0 확인 후1,278,661,055B를 제거했다.
M13-004만 IN_PROGRESS이며 작은 helper별 원격 CI를 반복하지 않는다.

### 2026-10-24 CI401 첫 타이머 확인과 P2c2 준비

50분 실제 타이머가03:23:29.834UTC에 만료된 뒤 exact head343f03f의 run/jobs를 조회했다.
Quick/observer/macOS/docs/security/vulnerability/corpus/minimum Go8개가 SUCCESS이고
Linux integration/job120754864311은 아직 IN_PROGRESS다. Required는 Linux 결과를 기다린다.
15분 연장 타이머의 다음 확인은 **2026-10-24 12:40:19 KST**다. 재push/재dispatch하지 않는다.
P2c1은 전체2138files/tree 대조와 evidence-inclusive security/docs 뒤 로컬 `e762bdc`로
커밋했다. 원격에는343만 있으며 P2c1의 원격 qualification을 주장하지 않는다.
[P2c 지도 §6](../evidence/package-client-quarantine/p2c-pnpm-selection-plan-20261024.md#6-다음-p2c2-producer의-source-작성-전-수정-위치)의
actual producer/preflight/typed network/정리와 Domain·HostInstallAction 남은 경계를
확인하며 다음 source를 준비한다. M13-004만 IN_PROGRESS다.

### 2026-10-24 P2c1 bounded lock·독립 source binding 로컬 검증

[수정 지도와 결과](../evidence/package-client-quarantine/p2c-pnpm-selection-plan-20261024.md)는
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
대기 중 [P2c 수정 지도](../evidence/package-client-quarantine/p2c-pnpm-selection-plan-20261024.md)의
기존 parser/graph/transaction 감사와 공식 exact source·실제 형식 대조/두 번째 검토를
먼저 완료했다. 현재 독립 P2c1 bounded unapproved lock parser부터 진행하며 hosted
qualification과 전체 PNPM 지원/CLI 완료는 계속 열려 있다. M13-004만 IN_PROGRESS다.

### 2026-10-24 P2b2 native admission과 실제 CI helper 로컬 검증

[수정 지도](../evidence/package-client-quarantine/p2b2-pnpm-admission-plan-20261024.md)의
exact PNPM Root/Node query demotion과 승인된 GHI own maps·실제 관측 VM 한 경로를 연결했다.
보완17 C++110 latch40.735s PASS 및 실제 PNPM4/UV2/GHI1 전체28.532s PASS다.
PNPM noargs의 Usage/exit2는 실패로 보존하고 자기 release member만 기존 고정 discard
probe callback으로 --version을 검사한다. Generic GHI와 기존 native4/Policy는 유지한다.
관련3package full Race22.297s와 최종 CI-connected canonical Quick34.137s/security4.723s/
docs/workflow/release-gate/freshness는 PASS다. [로컬 결과](../evidence/package-client-quarantine/p2b2-pnpm-admission-local-20261024.json)가 범위를 소유한다.
CI helper 자체의 별도 frozen 공식 archive 재생27.636s는 실제 PNPM4·기존 설치 거부와
Root/temp 정리/daemon wait0/자원0/Host불변을 확인했다. Public HTTPS/hosted CI 성공을
주장하지 않으며 실제 native 전체 qualified·signed whole update·P2c public graph/layout/CLI는
미완료다. 완료된 namespace/Docker 시험 데이터·이전 바이너리944,957,519B를
모든 UID 참조0/unknown0 뒤 제거했다. 현재 milestone/item은 바뀌지 않으며 M13-004 하나만 IN_PROGRESS다.

### 2026-10-24 P2b2 M6 제한 변경 승인과 Node 역할 보완

사용자는 [두 exact 읽기 변경안](../evidence/package-client-quarantine/p2b2-rust-artifact-runtime-decision-20261024.md)을 명시 승인했다. 이미 관측한 자기 guest PID maps만 먼저 적용하며
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
자기 maps 읽기에서 INCOMPLETE다. [구체 계약 변경안](../evidence/package-client-quarantine/p2b2-rust-artifact-runtime-decision-20261024.md)을 작성·두 번째 검토했지만
maps/VM 예외는 미적용이며 승인 전에는 그 거부를 유지한다. 독립 CONTROL 검증은 계속하고
M13-004만 IN_PROGRESS다. 전체 P2 qualification/배포·승격은 미완료다.

### 2026-10-24 P2b2 exact guest startup 로컬 범위

[Guest 지도](../evidence/package-client-quarantine/p2b2-pnpm-admission-plan-20261024.md)의
고정 Root source·공통 UV/PNPM transport/volume·exact profile을 연결했다. Quick174.844s/
security/docs·관련3package Race23.754s, 실제 source 정상+11거부와 C++108 latch가 PASS다.
실제 PNPM --version3.69s는12.15.0/완전한 관찰·잔여0/wait0/Host불변으로 PASS다.
최초 GNU/Rust read 실패와 fixture 교정 원본은 보존한다. 현재 offline empty lock/Node
query와 독립 ARTIFACT lane·원격 경계가 남으며 P2 전체 QUALIFIED/ACCEPTANCE_CLOSED는 NO다.
완료된 첫 transport/archive·시험 사본491,429,837B를 참조0/unknown0 뒤 삭제했다.

### 2026-10-24 P2b2 actual guest 공급·admission 착수 감사

[새 실제 소비자 지도](../evidence/package-client-quarantine/p2b2-pnpm-admission-plan-20261024.md)는
기존 단일 pin/배포/설치와 UV/NPM helper·volume·observer를 재사용한다. 고정 Root 소유
current PNPM 파일과 `/haa-runtime/pnpm` CONTROL을 연결하되 release ARTIFACT probe는
독립 검증한다. 현재 이 actual runtime 범위는 IMPLEMENTED/WIRED/QUALIFIED/
ACCEPTANCE_CLOSED 모두 NO이며 새 source 작성 전에 수정 지도와 두 번째 검토를 기록했다.

### 2026-10-24 P2b2 native 배포·기존 설치/복구 로컬 검증

[배포 경로 결과](../evidence/package-client-quarantine/p2b2-pnpm-distribution-local-20261024.json)는
NOTICE의 source-bound owner/freeze, native+passive 출력, exact11 assets와 5native/3metadata의
producer·검사·설치·bootstrap·sealed Promotion을 연결한다. Final product source의
Quick119.521s·canonical6 및 관련13package 전체 Race146.917s PASS다. 뒤의 gated root fixture
두 개만 교정한 [실제12 namespace 검증](../evidence/package-client-quarantine/p2b2-root-fixture-qualification-20261024.json)도 PASS이며 Host12files/images/bridge/service가 동일하다.
완료 시험 바이너리6디렉터리219,678,968 logicalB를 참조0/unknown0 확인 뒤 제거했다.
Guest PNPM 공급·exact gVisor admission/startup/query/lock가 다음이며 제품 CLI/새 signed
release·원격 qualification은 미완료다. M13-004/P2b2만 IN_PROGRESS다.

### 2026-10-24 P2b2 native 명시 출력 로컬 checkpoint

[실제 출력 결과](../evidence/package-client-quarantine/p2b2-pnpm-materialize-local-20261024.json)는 같은 bounded parser의 native bytes 반환과
선택한 source owner의 명시 외부 출력/no-replace·readback·0555/단일 링크를 검증했다.
Quick29.766s/security/docs·관련 전체 Race21.186s 및 실제 official/변조 거부3.732s PASS다.
초기 architecture 위반과 기존 TERM fixture 실패는 원본/교정 범위를 분리 보존했다.
Native 배포에는 공식 LICENSE/제3자 고지 전달도 포함하여 전체11 asset/5native role로
연결한다. 이 다음 배포·실제 admission과 전체 P2 qualification은 미완료다.

### 2026-10-24 P2b2 단일 owner·유지보수 로컬 checkpoint

[로컬 결과](../evidence/package-client-quarantine/p2b2-pnpm-owner-local-20261024.json)는 schema3 owner/proof·생성 코드·drift,
v2 metadata의 exact5 images+native tool, 명시 offline PNPM plan/apply/no-op을 연결한다.
실제 공급물의 중단 후 재개·변조 거부와 Quick131.199s·관련 전체 Race42.209s 및 canonical
security/docs/workflow/release-gate/freshness가 PASS다. Native 배포/admission·제품 명령은
아직 미연결이며 전체 P2b2의 QUALIFIED/ACCEPTANCE_CLOSED는 NO다. 개별 helper CI 없이 다음
배포·설치 역할 연결을 계속한다. Host 설치/서비스는 변경하지 않았다.

### 2026-10-24 P2b1 native 공급·형식 로컬 검증

[실제 결과](../evidence/package-client-quarantine/p2b1-pnpm-native-local-20261024.json)는 signed archive와
regular native file identity, Quick117.599s/security/docs·전체 관련 Race19.732s PASS다. 고정 Node
이미지의 network-none/read-only/non-root 환경19개 형식 관측과 정확한 Host 보존·완료 container/
파생 image 정리를 확인했다. Product gVisor/Policy/transaction 지원을 의미하지 않는다. 다음은
[단일 pnpm runtime owner·유지보수·배포/공급·exact admission 지도](../evidence/package-client-quarantine/p2b2-pnpm-runtime-plan-20261024.md)를 따라 공급 owner부터 연결한다.

### 2026-10-24 P2 착수 감사·공급 검증 순서
+
+[P2 baseline·수정 지도·두 번째 검토](../evidence/package-client-quarantine/p2-pnpm-baseline-20261024.md)는
+기존 npm 검역·guard/transaction·typed descriptor·Sigstore owner를 재사용한다. 공식12.15.0의
+Rust native·lock/peer graph·tar integrity/revisions·설정 및 실제 upstream self-hosted signer를
+대조했다. 현재 P2a는 offline 공급 verifier/유지보수 consumer만 구현하며 runtime pin·CLI 지원은
+아직 추가하지 않는다. 전체 P2는 IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO,
+MISSING은 착수 지도의 supply/runtime/graph/layout/actual transaction·원격 qualification이다.
+
### 2026-10-24 G127 완료·P2 인계

[G127 누적 결과](../evidence/package-client-quarantine/go1273-acceptance-20261024.json)는
같은095450d의 ordinary·cu126·cu130·cu132 각각 Required10SUCCESS와 실제 native macOS,
observer106/helper 종료를 연결한다. 기본 SDK·quality tools5·설치 Go members7을1.27.3으로
전환하고 보호12 metadata·실제 npm/pip·owned doctor/stop/readback 및 updater no-op을 확인했다.
완료 사설 입력9.55GB를 삭제했고 운영 복구용 원본 SDK/runtime/tools는 보존했다. Release 설치는
UNAVAILABLE이며 public release/main 병합을 의미하지 않는다. 과거396/local40m 실패는 보존한다.
G127의 네 상태는 모두 YES/MISSING 없음이다. 다음 P2는 기존 구현 감사·공식 의미·수정 지도와
두 번째 검토부터 시작하며 M13-004만 IN_PROGRESS다. 아래 날짜별 NO/PENDING은 당시 이력이다.

### 2026-10-24 G127 CI395 성공·CI396 cu126 지연 조사

[최초 실패·정상본 대조](../evidence/package-client-quarantine/go1273-cu126-ci396-failure-20261024.json)를
보존했다. CI395는 c34365e/같은 whole tree의 Required10SUCCESS·Linux53/CPU278.97/native
macOS/observer106이다. 별도 cu126/CI396은 CPU263.98s·다른8jobs PASS지만 sample gap1.451s
때문에 FAIL이며 Required도 FAIL이다. 정리 성공으로 원래 실패를 덮지 않는다.

[현재 진단 지도·두 번째 검토](../evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)를
따라 기존 코드7개·normal cu126/CI392·오답노트를 대조했다. 같은 기존24GB WSL에서 허가된
임시 재검증은40m timeout으로 실패했고 confirmed drain 뒤 원본12 복원을 확인했다.
Journal은 종료 약115초 전에 시작되어 promotion 전체40분 지연 가설은 반증됐다. 정확한
단계 시각 누락만 기존 progress port의 bounded test observer로 보완한다. 예산·poll·coverage는
유지한다. New pressure 관측은
직접 인과 확정이나 SDK regression 단정이 아니다. G127 IN_PROGRESS/QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO, P2..P7 NOT_STARTED다. 영구 SDK/runtime activation은 미실행이다.

[단계 진단 로컬 검증](../evidence/package-client-quarantine/go1273-phase-diagnostic-local-20261024.json)은
기존 progress port 연결과 bounded/unknown/동시 호출 Race·canonical Quick/security/docs PASS다.
첫 socket EPERM은 실행 환경 교정과 함께 보존했다. 완료 local payload와 임시 clone 약17.65GB는
참조0 확인 뒤 정리했다. 한 후보를 push하고 실제 단계 시각과 결과를 타이머로 확인한다.

### 2026-10-23 G127 최종 로컬 검증·source 반영

[최종 로컬 결과](../evidence/package-client-quarantine/go1273-local-qualification-20261023.json)에서
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
[수정 지도·공식 지원·두 번째 검토](../evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)의
방법으로2026.2 exact pin만 갱신했다. 이후 새 target에서 export data5/max4로 다시 거부되어 공식2026.2.2 patch가 실제1.27.3 source build/전체 offline analysis8.613s를 통과했다. Current SDK canonical과 새 target plan은 별도다. [Current SDK 로컬 결과](../evidence/package-client-quarantine/go1273-staticcheck-current-local-20261023.json)는 bootstrap·Quick56.004s/security/docs PASS다. 다음 NEW plan의 target 분석은 남는다. 다른 도구·검사 조건을 바꾸거나 아직 실행하지 않은
gVisor/native/CPU/CUDA를 완료로 쓰지 않는다. 기존 두 frozen state를 수정하지 않고 새 baseline에서
다음 plan을 만든다. 원격 CI는 최종 Go tuple 기능 경계로 묶고 실제 시간 기반 타이머를 사용한다.

### 2026-10-23 G127 첫 configuration 확인 — 모듈 형식만 보완

첫 exact1.27.3 plan/apply와 공식 SDK 준비는 PASS이며 bootstrap도 PASS다. Quick은 Go1.27의
require 블록 통합 형식에서 module drift를 탐지했다. [수정 지도·대조 결과](../evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)의
방법으로122개 require version/directness·기타 directive·go.sum 불변과 old/new tidy no-op을
확인했다. [Current SDK canonical 검증](../evidence/package-client-quarantine/go1273-module-layout-local-20261023.json)의
Quick48.866s/security/docs도 PASS다. Root floor1.26.10을 유지한 grouping만 반영했고 NEW
baseline/plan을 만든다. 첫 plan/실패 원본은 보존한다. Host 개발 SDK의 실제 readback은1.26.9였으며 source/qualification
SDK1.26.10·격리 target1.27.3과 역할을 구분한다. 기존 Host는 아직 전환하지 않았고 gVisor·전체
target tuple qualification·coherent activation/old owned retirement는 미완료다.

### 2026-10-23 G127 착수 감사 — 기존 updater 재사용

P1을7ce362b로 문서 종료했고 target Go1.27.3의 공식 release/advisory와 세 SDK archive metadata를
재확인했다. [갱신된 감사·수정 지도·두 번째 검토](../evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)를
따른다. 기존 coordinator·SDK bootstrap·gVisor capture/verify는 재작성하지 않는다.
README Build의 자동 renderer/consistency 소비자 누락은 [같은 회귀/기존 소비자 검증](../evidence/package-client-quarantine/go1273-readme-consumer-local-20261023.json)으로
보완했다. Canonical5/전체 두 도구 Race25.424s는 PASS이며 version owner·거부 조건은 유지했다. 이후 깨끗한 baseline에서 기존 두 frozen plan과 실제 target SDK/
bundle·observer를 검증한다. 현재 pin/SDK는1.26.10이며 Host SDK/CLI·서비스는 변경하지 않았다.
다음 단계의 P2..P7은 시작하지 않는다. 단일 active work item은 계속 M13-004다.

### 2026-10-23 P1 COMPLETE — 실제 네 profile·문서 검증과 정리

[누적 실제 결과](../evidence/package-client-quarantine/p1-uv-acceptance-20261023.json)는
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

[CLI·공통 batch 로컬 결과](../evidence/package-client-quarantine/p1f-uv-sync-cli-local-20261023.json)는
complete sync의 Application/CLI/bootstrap·새 승인 records 리포트와 빈 입력 skip을 연결했다.
기존 정상 pip의 batch lifecycle·static-all/대상별 closure를 재사용하여 rootless 의존성 전달
누락을 교정했고 같은 pre-format test의 before FAIL/after PASS를 보존했다. 최종 canonical5,
관련7package 전체 Race138.116s와 실제7언어 CLI21cases는 PASS다. 임시 CLI 바이너리43,581,079B와
test payload를 정리했다. 실제 native complete/adopt/empty/closure 두 통합 시나리오는 기존 Linux
job에 연결했으며 당시 아직 실행 전이었다. Backend와 CLI commit을 한 번의 원격 후보로 올린 뒤 실제
native·공통 CI/Required와 세 CUDA를 확인한다. P1 acceptance는 열어 둔다.

[Complete backend 로컬 결과](../evidence/package-client-quarantine/p1f-uv-sync-backend-local-20261023.json)는
actual held source/target guard·전체 native 원본 proof 연결, 단일 complete 선택과 승인 cache·
Manifest/SBOM·기존 offline publication을 검증했다. Canonical5와 관련5package 전체 Race142.011s
PASS이며 empty 추출 회귀의 기존 정상 코드 대조·교정 및 최초 실패는 같은 영수증에 보존했다.
Synthetic 거래 검증이며 당시 실제 native 전체 검역·Application/CLI·CUDA3/P1은 미완료였다.
이후 D 연결의 로컬 결과는 위 영수증을 따르며 개별 helper별 새 원격 실행 없이 기능 경계로 검증을 묶는다.

[CI390 확인](../evidence/package-client-quarantine/p1f-ci390-timed-confirmation-20261023.json)은
source79e76dd와 actual checkout의 전체 tree 일치·Required 포함10SUCCESS를 확인했다.
Linux51/CPU297.93s, macOS 같은2m build 한도 안의 약59.17s/default test와 helper 정리는 PASS다.
앞 macOS timeout 내부 원인을 해결했다고 해석하지 않으며 actual complete sync/CLI·CUDA3와
P1 acceptance는 미완료다. 이전 CI388/389 항목은 당시 실패 이력으로 보존한다.

기존 예방 규칙을 현재 P1에 적용한 [검증 실행 주기](../20-package-client-quarantine-plan.md#5-검증-matrix와-자원-정리)를
적용한다. P1f A2c2/C/D의 실제 guard·cache·records와 CLI를 연결했고 작은 helper별 원격 완료 대기를
중단하고 backend 및 CLI end-to-end를 한 후보로 push/CI를 묶는다. 로컬 canonical/관련 회귀와
중간 commit은 유지한다. CUDA3 actual은 P1 완료 후보에서 수행하며 미검증 상태로 완료·승격하지
않는다. 새 공통 회귀나 native 선행조건이면 필요한 원격 검증을 먼저 수행한다.
CI 실행 시 최근 같은 profile의 경과 시간으로 첫 확인 시각을 잡고 대기 도구로 확인한다.

### 2026-10-23 CI389 macOS build — 당시 결과

[CI389](../evidence/package-client-quarantine/p1f-cpu-completion-ci389-20261023.json)은 CPU 회계 교정 source의 Linux/CPU full·뒤 네 소비자와 정리는 PASS다. macOS는 production build2m에서 timeout이고 default test는 미도달이다. [기존 macOS 진단 지도](../evidence/package-client-quarantine/macos-default-test-diagnostic-plan-20261022.md)의 bounded package progress 보완을 먼저 수행한다. 내부 지연 원인·새 source 인과와 전체 qualification은 미확정이다.

### 2026-10-23 CI388 선행 회귀

[최초 결과](../evidence/package-client-quarantine/p1f-uv-original-proof-ci388-20261023.json)의 native
UV/CLI·46개 Linux 선행 tests·다른8 prerequisites는 PASS지만 CPU full의 unit-outcome 회계 시각
역전으로 Linux/Required는 FAIL이다. 원본/복원을 보존했고 [수정 지도·두 번째 검토](../evidence/package-client-quarantine/p1f-cpu-unit-completion-baseline-20261023.md)를 먼저 따른다. [로컬 교정](../evidence/package-client-quarantine/p1f-cpu-unit-completion-local-20261023.json)의 same-test before FAIL→after PASS·canonical5/전체 관련 Race는 PASS이며 fresh source 실제 검증은 남는다.
B1/A2c1 local 후보를 이 실패 실행에 포함하거나 기존 CUDA 양의 지연과 같은 원인으로 쓰지 않는다.

### 2026-10-22 순서 결정 — P1 완료 후 Go1.27.3, 그다음 P2

사용자는 GO-2026-6809와 Go1.27.3 전환을 추가 요청하고 순서 판단을 위임했다.
현재 source/CI/runtime/tool pin은 이미 보안 수정1.26.10이며 [CI377 최종 결과](../evidence/package-client-quarantine/p1c-guarded-cli-ci377-20261022.json)의
Required/Vulnerability 포함10SUCCESS를 확인했다. 이 advisory의 미패치 source 복구와
다음 Go 계열 전환을 구분한다. 기존 Host/preview를 자동 업데이트한 것은 아니다.

| 순서 | 범위 / 상태 | 시작·완료 기준 |
| --- | --- | --- |
| 1 | P1 uv pip — IN_PROGRESS | C3b 빈 기존 venv 설치는 완료. C3c installed-aware 설치/업데이트·native ownership·complete/multi-root/empty sync와 P1 실제 qualification을 닫는다 |
| 2 | M13-004/G127 Go1.27.3 — NOT_STARTED | P1 acceptance 뒤 [착수 감사·수정 지도·검증·두 번째 검토](../evidence/package-client-quarantine/go1273-migration-baseline-20261022.md)를 갱신하고 기존 maintenance plan/apply/verify로 별도 후보를 검증한다. Compiler/guest/tools/gVisor bundle·observer·CI/release/support tuple과 actual 소비자/플랫폼/CPU·세 CUDA/전환 경계를 확인 |
| 3 | P2 pnpm → P3..P7 — NOT_STARTED | G127 qualification/acceptance 후 새 기준에서 확장을 재개한다. 전체 client 완료까지 Go 전환을 미루지 않음 |

M13-004 하나만 IN_PROGRESS이며 G127을 동시에 착수하지 않는다. Go1.26은 공식 지원 중이고
현재 취약점은1.26.10에서 수정되어 P1 기능 변화와 runtime 변화를 분리할 수 있는 경계가
적절하다. 새 reachable advisory/지원 종료가 이 판단의 전제를 바꾸면 긴급 순서를 재평가한다.
1.27.3은 전환 목표이며 현재 lock/workflow/Host SDK는 이 순서 기록에서 변경하지 않는다.
후속 요청에 따라 G127 완료 기준에는 실제 compiler/binary/bundle/설치 readback의 버전 일치와
전환·복구 검증/마지막 참조 확인 뒤 구버전 owned SDK·cache·bundle·시험 자원의 정리를 포함한다.
현재 P1의 active1.26.10 입력은 유지하며 역사 evidence의 버전 문자열을 실행 잔재로 삭제하지 않는다.

### 2026-10-21 owner scope — client expansion and full-name brand

CI386을 기다리는 동안 [C3e3 publication 감사·수정 지도](../evidence/package-client-quarantine/p1f-uv-sync-publication-baseline-20261023.md)를 추가했다. [CI386 종료](../evidence/package-client-quarantine/p1e-uv-native-source-ci386-20261023.json)는 같은 전체 tree/Required10SUCCESS와 실제 native 다섯 사례·기존 Linux51/UV CLI194.90s/CPU245.89s/native macOS·helper5wait0/정리를 확인했다. C3e2a/b의 네 상태는 해당 resolver 범위에서 YES다. 다음 A1은 기존 원본 metadata capture/verify와 inventory 공통화만 수행한다. Whole guarded sync/P1은 미완료이며 A2/C1/D1 지도와 각 후속 source 직전 검토를 따른다.

M13-004만 IN_PROGRESS로 유지한다. [확장 계획](../20-package-client-quarantine-plan.md)은
uv pip/project·pnpm·Bun·pipx 설치/업데이트와 npx·uvx·bunx·pipx run 등을 하나의
범위로 설계하고 단계별 실제 검증 후 지원을 연다. 기존 검역 엔진은 재사용한다.
설계·두 번째 검토 P0와 로컬 검사·preview 갱신을 완료했다. 실제 화면 결과는
[풀네임 타이틀 보완 기록](../evidence/m13-004-convenience/owner-cli-review-20261021.md)을 따른다.
사용자 계속 지시에 따라 [P1 uv baseline](../evidence/package-client-quarantine/p1-uv-baseline-20261021.md)의
P1a native lock/선택 경계를 커밋b710b56으로 보존하고 CI360을 확인한다. 다음은
[P1b runtime baseline](../evidence/package-client-quarantine/p1b-uv-runtime-baseline-20261021.md)의 metadata/native tool 공급은 [로컬 영수증](../evidence/package-client-quarantine/p1b-uv-tool-local-qualification-20261021.json)까지 확인했다. [Pin/maintenance 로컬 검증](../evidence/package-client-quarantine/p1b-uv-pin-local-qualification-20261021.json)도 actual CLI/canonical7 PASS다. 다음은 [cu126 회계 지연 조사](../evidence/package-client-quarantine/go12610-cu126-diagnostic-plan-20261021.md)와 UV production runtime 연결이다. P1은 IN_PROGRESS, P2~P7은 NOT_STARTED다.
[Go1.26.10 보안 선행조건](../evidence/package-client-quarantine/go12610-security-baseline-20261021.md)을
기존 유지보수 도구의 별도 candidate에서 [최종 로컬 검증](../evidence/package-client-quarantine/go12610-local-qualification-20261021.json)을 마쳤다. [원격 상태](../evidence/package-client-quarantine/go12610-remote-status-20261021.json)에서 같은 tree의 CI361 Required/CPU/native macOS PASS와 cu126 sample-late FAIL을 확인했다. cu132도 별도 full1054.68s·Required10SUCCESS이며 cu130의 Linux/full980.48s도 PASS이나 native production build2m timeout으로 Required는 FAIL이다. 새 진단 source270c7c8의 PR366 Required10SUCCESS·cu126 Linux/full1296.54s PASS를 확인했으나 manual native default-test timeout/Required FAIL이다. Go 전체 qualification은 미완료다. Host 설치·서비스는 유지한다.
[P1b5 원격 결과](../evidence/package-client-quarantine/p1b5-uv-runtime-remote-20261022.json)는
CI368의 신규 native UV3.25s·기존 Linux45/CPU259.85s PASS와 macOS default-test6m/
Required FAIL을 구분한다. [P1c 수정 지도](../evidence/package-client-quarantine/p1c-uv-resolver-baseline-20261022.md)는
native JSON이 delta-only임을 확인하고 기존 pip promotion identity/control과 multi-root
sync 모델의 연결 gap을 기록했다. Production UV resolver/install-upgrade-sync/CLI는 아직
미연결이다. [macOS 진단 지도](../evidence/package-client-quarantine/macos-default-test-diagnostic-plan-20261022.md)의
bounded 진행/완료 root 요약은 기존 한도·process cause를 유지한다. Source66aa7e0의
[CI370 최종 결과](../evidence/package-client-quarantine/p1c-baseline-ci370-20261022.json)는
same-tree checkout/Required 포함10SUCCESS·native macOS와 Linux 기존45/native UV3.20s·
CPU209.50s/정리를 확인했다. 과거 timeout 원인이나 이후 candidate qualification은 아니다.
승인한 fixed read 후보는 커밋3623d1e/fresh C++105/startup4.09s PASS이며 offline
inventory의 다음 최초 거부는 Python child libpython read다. [C1c 수정 지도](../evidence/package-client-quarantine/p1c-uv-resolver-baseline-20261022.md)의
exact clone/creator·첫 query exec·ARTIFACT 역할·readonly ABI/stdlib 후보를 구현 중이다.
초기 false exec-origin 전제는 실제 pinned kernel의 OriginExec 상속에 맞게 교정한다.
Native 최소 empty-target query 검증은 아직 남고 전체 inventory 성공으로 확대하지 않는다.
M13-004/P1만 IN_PROGRESS이며 production 지원/acceptance는 계속 미완료다.
후속 [CI371](../evidence/package-client-quarantine/p1c-checkpoint-ci371-20261022.json)은
source3623d1e/same tree의 Linux 기존45·UV3.67s/CPU196.37s/정리는 PASS, macOS
47started/47finished·active0 뒤 default6m/Required FAIL이다. [종료 상태·완료 gap 진단](../evidence/package-client-quarantine/macos-terminal-local-20261022.json)은
canonical5·Race15.425s/Darwin compile PASS이며 새 native는 남는다. C1c는 actual
query 역할을 관측했고 ABI·VM/own maps·UTC와 UV parent의 exact Python image data
read 후보를 C++106으로 검증 중이다. 개별 첫 거부·완료 자료 정리는 P1c 지도가 소유한다.
C1c parent-image 후보의 fresh C++106/startup3.91s/query4.10s·complete trace31은 로컬
PASS다. Whole-image inventory 진단은 초기 commit/raw로만 보존하고 실제로 통과한
startup/empty-target 두 fixture만 CI에 연결한다. Source 정리 후 final replay도 startup3.73s/query3.96s·complete trace29/31로 PASS다.
새 원격 qualification은 남는다. 다음 구현은 같은 P1c의 production native 선택/data-reader 연결이다.
[C2 baseline/수정 지도](../evidence/package-client-quarantine/p1c-uv-native-producer-baseline-20261022.md)는
기존 stdin admission·bounded tar와 isolated Python data reader를 재사용하는 단일-root
producer의 실제 남은 작업을 분리한다. [C2 output 로컬 검증](../evidence/package-client-quarantine/p1c-uv-output-local-20261022.json)은
native stdin→pylock→sync/list0과 named tmpfs의 bounded control transport, 새 collector/
RFC822-WHEEL adapter의 canonical5·Race9.454s PASS를 확인했다. Guest-only /tmp의
docker cp 실패는 별도 보존했고 두 probe의 소유 자원/완료 자료는 제거했다. 다음은 UV
output volume의 실제 gVisor mount 연결과 typed-network native 선택·독립 data reader다.
Production resolver/CLI는 아직 미연결이며 P1 acceptance는 열려 있다.
[C1c CI372](../evidence/package-client-quarantine/p1c-query-ci372-20261022.json)는 sourcee3665ea와
checkout tree 일치·remote startup3.03s/query2.90s·complete trace29/31, 기존 Linux/CPU283.96s와
helper5wait0/정리를 확인했다. macOS는 default test 이전 production build2m에서
exited=false/exit-1·CPU339188ms로 종료됐고 Required는 FAIL이다. 앞 default6m 실패의
원인으로 합치지 않으며 전체 qualification·production UV 지원은 계속 미완료다.
[C2 actual output 로컬 검증](../evidence/package-client-quarantine/p1c-uv-native-output-local-20261022.json)은
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
[독립 reader 로컬 검증](../evidence/package-client-quarantine/p1c-uv-reader-local-20261022.json)의
canonical5·Race10.055s와 기존 parser를 통한 whole archive/graph replay가 PASS다. Metadata/
archive mismatch·byte/stream·취소/cleanup/trace 실패의 graph 삭제와 임시 파일 제거도 확인했다.
실제 Python HTTP 및 public native UV caller/guarded CLI는 아직 미연결·미검증이다.
Reader source344a93d는 로컬 checkpoint로 보존했다. 반복된 macOS prerequisite에 대해
[cache 입력 정렬의 baseline/수정 지도·두 번째 검토](../evidence/package-client-quarantine/macos-default-test-diagnostic-plan-20261022.md)를
추가했다. Native CI373은 build PASS/default6m FAIL이며, 실제 cache root와 -trimpath
action ID 분리를 확인했다. Scope/timeout·원인 판정은 유지한 채 이 불일치만 정렬한다.
[Cache 입력 정렬 로컬 검사](../evidence/package-client-quarantine/macos-cache-alignment-local-20261022.json)는
canonical6·checker Race36.760s·Darwin compile11.016s PASS다. Source lookup의 실제 비교를
보존했고 완료 binary/operator7파일은 제거했다. 새 native macOS/전체 Required는 남는다.
[CI373 최종 결과](../evidence/package-client-quarantine/p1c-native-output-ci373-20261022.json)는
source9b0da57/checkout tree 일치, native startup3.24s/query3.13s/compile-sync-collection3.58s·
complete trace29/31/35와 기존 Linux/CPU287.67s·helper5wait0/정리를 확인했다. macOS는
build94.05s PASS 뒤 default6m/Required FAIL이다. 이 결과를 이후 reader344a93d나 cache
정렬 후보의 원격 qualification으로 전용하지 않는다. 새 후보 CI가 다음 확인 지점이다.
Sourcef76c564를 push했고 CI374가 진행 중이다. 다음 public native UV caller와 독립 reader의
순차 연결 후보를 구현했고 source/runtime/tool preflight·owned-container cleanup 회귀를
검증한다. App/guarded install-upgrade-sync·CLI는 미연결이며 public/HTTP actual은 남는다.
[Public caller 로컬 검증](../evidence/package-client-quarantine/p1c-public-caller-local-20261022.json)은
canonical5·관련 Race16.610s와 lock-only preflight/snapshot actual 세 fixture를 통과했다.
완료 native 자료688,406,687B는 제거했다. Public single/multi graph entry를 기존 CI의
인증된 test-client 단계에 연결했고 다음 source CI가 실제 public/HTTP 완료 조건이다.
[CI374 종료](../evidence/package-client-quarantine/cache-alignment-ci374-20261022.json)는
sourcef76c564/same checkout tree·Required 포함10SUCCESS다. macOS build55.562s와
default test envelope171.294s 이내, Linux native 세 시험/기존 통합·CPU186.73s 및 정리가
PASS다. Cache 정렬 후보의 native 완료를 확인했고 과거 모든 timeout의 원인은 단정하지 않는다.
이후 public caller sourceab54d4a는 별도 push했으며 실제 public/HTTP·guarded CLI는 남는다.
[Guarded client 다음 gap 감사](../evidence/package-client-quarantine/p1c-guarded-client-baseline-20261022.md)는
기존 promotion/update/enrollment·bootstrap의 재사용 위치와 UV/multi-root gap을 분리했다.
읽기 전용 baseline이며 구현은 시작하지 않았다. C2의 public actual 결과부터 확인한다.

[CI375 첫 public 결과](../evidence/package-client-quarantine/p1c-public-ci375-20261022.json)는
sourceab54d4a/same checkout tree에서 idna@3.11 native compile이 exact /etc/hosts의
read 분류 누락으로 STREAM_FAULT였다. Required/Linux FAIL, 나머지8 prerequisite SUCCESS,
helper wait0·confirmed stop/원래 client 복원은 확인했다. Public graph/독립 HTTP는
미통과다. C2 leaf의 수정 지도·두 번째 검토에 따라 UV direct root의 exact system hosts
읽기만 보완하고 전체 latch/canonical 및 새 actual public 입력으로 검증한다. C3 구현과
P2..P7은 시작하지 않는다.

C2의 새 CI376을 기다리는 동안 [native retention/empty/venv 형식 조사](../evidence/package-client-quarantine/p1c-uv-retention-format-20261022.json)를
독립 data-only 범위로 수행했다. 기본/선택 upgrade의 기존 dep 유지, 전체 upgrade의 dep
갱신, sync의 제외 항목/빈 집합 삭제와 uv venv의 version_info/생성 marker 차이를 확인했다.
잘못 지정한 첫 empty output filename의 exit2는 operator 오류로 별도 보존했다.
[C3 지도](../evidence/package-client-quarantine/p1c-guarded-client-baseline-20261022.md)에
수정 위치와 두 번째 검토를 반영했으며 제품 구현은 NOT_STARTED다. Owned containers6/
images2와 완료 자료3,143,502B는 제거했고 Host 전후 불변을 확인했다.

[CI376 최종 결과](../evidence/package-client-quarantine/p1c-public-ci376-20261022.json)는
source35c0f2d/same checkout tree·Required 포함10SUCCESS다. Native UV→독립 whole-archive
binding은 idna1node/requests5node·4edge를22.75s에 반환했다. C++106/Linux49 통합과
CPU295.93s/helper5wait0/final confirmed cleanup, native macOS도 PASS다. C2 single-root
producer의 acceptance만 닫고 P1 전체/guarded client·CLI는 미완료로 유지한다. 다음은 C3
지도의 기존 venv transaction/서비스 재사용과 client/source 분리 gap의 순차 구현이다.
C3a는 resolve 전 venv guard를 획득하는 내부 신규 설치 준비 adapter를 시작한다.
기존 Update의 per-request rebind와 transaction을 재사용하고, 배포된 distribution이 있는
환경의 fresh solve는 거부한다. Installed-aware 선택/CLI는 후속 실제 연결 단계다.
[C3a 로컬 checkpoint](../evidence/package-client-quarantine/p1c-prepared-install-local-20261022.json)는
canonical5와 application 전체/관련 promotion·sandbox Race3개를 통과했다. 내부 준비 서비스와
empty-venv guard를 구현했고 task TMPDIR는 비어 있다. Composition/CLI와 실제 target 반입은
아직 미연결·미검증이다. 다음 C3b는 client/source descriptor와 실제 factory 연결이다.
C3b 연결 구현을 시작한다. 실제 두 Python client만 descriptor와 factory에 등록하며 기존
pip/PyTorch 동작과 lazy help를 유지한다. UV 신규 설치는 빈 기존 venv에 한정하고,
installed bootstrap-test의 실제 검역·반입/비어 있지 않은 대상 거부를 기존 CI에 연결한다.
[C3b 로컬 결과](../evidence/package-client-quarantine/p1c-guarded-cli-local-20261022.json)는
canonical5와 CLI/bootstrap/application/promotion/checker 전체 Race PASS다. 실제 빈 venv
UV CLI 연결은 IMPLEMENTED/WIRED YES, QUALIFIED/ACCEPTANCE_CLOSED NO이며 다음은
같은 후보의 actual native CLI/Required CI다. 기존 설치·upgrade/native ownership·sync는 남는다.
CI 대기 중 C3 leaf에 C3c source 감사·metadata-only native 전송의 미확정 지점·유한 조사와
수정/회귀 지도를 기록했다. C3c code는 시작하지 않았으며 C3b actual CI 확인이 우선이다.
[후속 metadata-only 형식 조사](../evidence/package-client-quarantine/p1c-uv-metadata-only-format-20261022.json)는
14개 native 입력과 원본 불변·Host 동일성·owned 자원 정리를 확인했다. 다음 UV 선택 입력은
bounded inert METADATA로 구체화하며 gVisor/public/guarded update 성공으로 확대하지 않는다.
[CI377 종료 확인](../evidence/package-client-quarantine/p1c-guarded-cli-ci377-20261022.json)은
source8ec8de7/actual checkout334918f·동일 전체 tree와10SUCCESS다. 실제 UV CLI 설치25.50s,
nonempty 반복 거부·target/Evidence 불변, Linux50/CPU283.72s/helper5wait0/final confirmed
cleanup와 native macOS·Vulnerability를 통과했다. C3b 빈 venv 설치의 네 상태는 모두 YES다.
현재 다음은 C3c installed-aware install/upgrade/native ownership·sync이며 P1 전체는 미완료다.
C3c1 native installed-aware producer를 시작한다. C3 leaf의 사전 감사·정확한 수정 지도·두 번째
검토를 확인하고 strict delta/원본 metadata controls와 기존 UV native/독립 binder 연결만 구현한다.
Guarded update 반입·CLI/native ownership·sync는 후속이며 P1 밖으로 이동하지 않는다.
[C3c1 로컬 결과](../evidence/package-client-quarantine/p1c-installed-producer-local-20261022.json)는
canonical5/관련 Race3개 PASS다. Native resolver/CI는 연결했고 다음은 실제 public/Required
검증이다. Original metadata 한도와 selected graph 한도는 구분했고 actual native 성공은 아직 아니다.
[CI378 종료](../evidence/package-client-quarantine/p1c-installed-producer-ci378-20261022.json)는 같은 전체 tree와 Required 포함10SUCCESS, 실제 native 세 intent19.60s/4controls, Linux51/CPU231.72s·helper5wait0/최종 confirmed cleanup와 macOS/Vulnerability를 확인했다. C3c1 네 상태는 모두 YES다. 다음 C3c2a는 기존 original snapshot/held transaction과 UV 선택 controls를 연결하는 managed guard이며, CLI 변경·native adoption·sync는 후속이다. 정확한 지도와 두 번째 검토는 C3 leaf를 따른다.
[C3c2a 로컬 결과](../evidence/package-client-quarantine/p1c-managed-guard-local-20261022.json)는 canonical5와 promotion/application/artifact-pypi 전체 Race PASS다. Managed UV guard port와 exact controls/lock/graph·기존 반입/rollback 연결은 구현했으며, 실제 shared pip 소비자/Required CI와 production CLI·actual UV target update는 미완료다. 그 원격 검증 뒤 C3c2b binding을 진행한다.
[CI379](../evidence/package-client-quarantine/p1c-managed-guard-ci379-20261022.json)의 same-tree10SUCCESS와 실제 공통 소비자·macOS·정리로 해당 원격 gap을 닫았다. C3 leaf의 C3c2b 사전 감사/수정 지도/두 번째 검토대로 일반 installed-aware 설치와 두 UV upgrade intent를 연결한다. Native UV ownership과 sync·P1 전체는 아직 미완료다.
[C3c2b 로컬 결과](../evidence/package-client-quarantine/p1c-installed-cli-local-20261022.json)는 canonical5/전체 관련5package Race PASS다. 일반 installed-aware INSTALL과 --upgrade/-U·--upgrade-package의 UPDATE를 기존 guard/검역/반입에 연결했고 actual authenticated CLI5단계·부정 검사를 기존 CI에 추가했다. 다음은 같은 source의 실제 CLI/Required 확인이며 아직 native qualification 완료가 아니다.
CI 대기 중 [native UV 형식 감사](../evidence/package-client-quarantine/p1d-native-uv-ownership-baseline-20261022.md)의 자체 wheel/loopback14명령과 정리를 확인했다. Native public/adoption 또는 현재 CLI qualification으로 전용하지 않으며 C3c2b 완료 뒤의 구현 gap에 반영한다.
[CI380 종료](../evidence/package-client-quarantine/p1c-installed-cli-ci380-20261022.json)는 동일 tree/Required10SUCCESS와 실제 UV CLI5단계118.85s·Linux51/CPU293.60s·macOS·정리를 확인했다. C3c2b 네 상태는 YES다. 다음 C3d는 위 native UV 감사의 기존 old-wheel proof·namespace 보호·native fixture 검증을 재사용한다. Sync와 P1 전체는 미완료다.



공개 채널 이름·publisher binding·PR38/main CI·배포는 기존 미완료로 남는다.

### Runtime upgrade maintenance — 2026-10-15 priority change

사용자가 M13을 잠시 보류하고 버전 갱신 자동화의 상세 설계·계획을 먼저 정리한
뒤 구현하도록 요청했다. 기존 M13 branch는 `f5a0353`에 보존하고 같은 commit에서
`milestone/runtime-upgrade-maintenance`를 분기했다. Tracked/index clean 기준선과
기존 미추적 사용자 자료를 유지했다. 원격 branch/PR/ruleset/설치 변경은 없다.

기술 owner: [Runtime upgrade design](../../engineering/05-runtime-upgrade-maintenance.md).
Baseline audit·작업별 Read/acceptance·검증 순서·M13 재개 조건:
[Runtime upgrade plan](../19-runtime-upgrade-maintenance-plan.md).

| Item | Scope | Status | Acceptance evidence |
| --- | --- | --- | --- |
| UPG-001 | baseline audit / design / detailed plan | COMPLETE | 소스 변경 전 목표별 audit와 design/plan 작성; [closure evidence](../../engineering/evidence/runtime-upgrade-maintenance/upg-001.json) |
| UPG-002 | all owned version consumers / manifest / drift | COMPLETE | [ownership/manifest evidence](../../engineering/evidence/runtime-upgrade-maintenance/upg-002.json); canonical Quick, security,12 focused regressions/54 cases PASS |
| UPG-003 | explicit plan/apply/verify / same-version rehearsal | COMPLETE | [updater evidence](../../engineering/evidence/runtime-upgrade-maintenance/upg-003.json); actual no-op2/2, canonical verify7 profiles,10 tests/28 cases PASS |
| UPG-004 | gVisor·archived quality tool Go1.26.9 candidates | COMPLETE | [quality compiler](../../engineering/evidence/runtime-upgrade-maintenance/upg-004-gitleaks.json) / [gVisor build](../../engineering/evidence/runtime-upgrade-maintenance/upg-004-gvisor.json); actual/reproducible candidates and observed-output plans qualified |
| UPG-005 | installed ecosystem / CPU / cu126 / cu130 / cu132 qualification | COMPLETE | [qualification evidence](../../engineering/evidence/runtime-upgrade-maintenance/upg-005.json); same local full matrix and fresh ordinary/nativeMac/three independent remote CUDA Required PASS; originals/cleanup and final docs/security verified |
| UPG-006 | verified activation / retirement / M13 handoff | COMPLETE | [activation/handoff evidence](../../engineering/evidence/runtime-upgrade-maintenance/upg-006.json); actual default quality5/installed Go7 at1.26.9, production2 and scoped doctor, no-op/7canonical profiles, exact completed-test cleanup and final docs/security PASS; public release/Race/reviewer merge retain separate scope |

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

UPG-004 quality-tool component: [Gitleaks/compiler evidence](../../engineering/evidence/runtime-upgrade-maintenance/upg-004-gitleaks.json).
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

UPG-005 fixture correction closure: [partial qualification evidence](../../engineering/evidence/runtime-upgrade-maintenance/upg-005.json).
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
[single-version maintenance](../../engineering/evidence/go-single-version-20261015/README.md).

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
- Evidence: [report](../../engineering/evidence/security-go1269/README.md) and
  [machine-readable results](../../engineering/evidence/security-go1269/result.json).
  [SumDB diagnostic follow-up](../../engineering/evidence/security-go1269/sumdb-followup.md)
  and [remote result](../../engineering/evidence/security-go1269/remote-result.json)
  record the source, local before/after and actual remote validation. The original
  local result is a historical snapshot, not the current remote CI status.
- Read: [Quality Toolchain](../../engineering/03-quality-toolchain.md),
  [CI Quality Gate](../../engineering/04-ci-quality-gate.md),
  [Trusted Tooling](../../threat-model/04-trusted-tooling-and-evidence.md),
  [M10 runtime identity contract](../13-m10-verified-distribution-bootstrap-contract.md),
  [M12 Go contract](../16-m12-01-ecosystem-expansion-contract.md#4-m12-002--go-modules).
- Scope: host/product build toolchain and bounded SumDB failure diagnosis.
  `scripts/runtimes.lock.json`,
  generated runtime identities, qualified Go guest image and historical M12
  evidence remain exact inputs; no old qualification is relabeled for Go 1.26.9.
- Delivery follow-up: CI306 stalled in hosted APT acquisition and was cancelled
  by the next push, before runtime tests. The bounded official-mirror and
  timeout repair passes actual APT acquisition/install in CI307. APT audit:
  `IMPLEMENTED=YES`, `WIRED=YES`, `QUALIFIED=YES`, `ACCEPTANCE_CLOSED=YES`
  for package acquisition. The later Cargo failure remains separate; full
  delivery Required is not green. See the [APT follow-up](../../engineering/evidence/security-go1269/apt-followup.md).
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
  remain unchanged. See the [Cargo follow-up](../../engineering/evidence/security-go1269/cargo-gai-followup.md).

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

M11-001은 [M11 Dynamic Detection Depth Contract](../14-m11-dynamic-detection-depth-contract.md)에서
exact gVisor schema lock, bounded normalized vocabulary와 raw payload non-retention
contract를 먼저 확정했다. M11-002는 이 contract의 requested point/field가 pinned
source에서 실제로 compile·parse되는지를 Bazel contract test와 Linux lifecycle CI로
증명했다. M11-003은 이 bounded fact에 trusted backend classification과 actionable
Finding을 연결했고, exact helper와 Linux lifecycle CI로 검증했다.

M11 post-qualification release hardening은
[M11 이후 Release Hardening Fix List](../15-m11-02-fix-list.md)의 FIX-01~05만
수행한다. 새 architecture 또는 detection capability는 추가하지 않는다. 모든 fix의
local qualification을 완료하고 하나의 milestone branch push/CI evidence를 확인한
뒤에만 M12를 재개한다.

M12 생태계 확장 순서는 queue에 고정한다: 공식 PyTorch source profile → public Go
Modules → public Cargo/crates.io → public Terraform Provider → 전체 생태계
qualification·feature freeze → M12-02 final red-team/fix gate. M12 기능과 red-team
gate가 완료되기 전에는 M13 Production Release & Operations로 이동하거나 public
release를 게시하지 않는다. source checkout/local Go build 없는 public bootstrap은
M13-005가 완료하기 전까지 제공된다고 주장하지 않는다.
