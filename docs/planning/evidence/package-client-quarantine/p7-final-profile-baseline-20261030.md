# P7 착수 전 — 지원 범위와 최종 profile 실행 경계

기준은 G3 commit684a6df3a2b6689354abdf100a05de1eff4d37c3/tree84f97d4a다.
M13-004만 active다. 이 문서는 기존 기능을 다시 구현하거나 전체 acceptance를 닫지 않는다.
P6 named local 구현을 보존하고 새 source hosted·필수 영향 profile gap을 확인한다.

Read: Queue/PROJECT-DECISIONS/docs README, lessons INDEX·ci L06·sandbox L18·artifact
L02/L19·evidence L16, client09 §3/§5/현재 지원 lane, plan20 §4–6,
[P2/P3 matrix](./p2p3-final-matrix-20261026.md), CI contract §13/§14,
현재 workflow와 PyTorch qualification input/cleanup owner.

## 첫 실패의 disposition

[CI440 원본](./ci440-go-proxy-first-failure-20261030.json)은 frozen6d9d3bf의
8SUCCESS/Linux·Required FAIL, 새 Bun named cold/current/CLI/거부 PASS와 후속 Go41의
Host HTTP Do 실패를 구분한다. 하위 transport 원인·endpoint는 미수집이다. Current G3에서
기존 실제 Go41 get/download196.03s는 코드·HTTP 한도·retry 변경 없이 PASS했다.
현재 로컬 성공을 원격 원인 해결로 쓰지 않는다. 별도 unchanged full CI 재시도 대신
이미 검증한 G3와 아래 필요한 최종 검증 경계를 한 후보로 묶는다.

## 기존 owner와 실제 남은 acceptance

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| UV pip install/upgrade/sync | YES | YES | P1 accepted named source YES | P1 scoped YES | 이후 공통 source의 최종 영향 matrix |
| PNPM/Bun project, UV project/tools, pipx installed ownership/CLI | YES | YES | 각 canonical named receipts/CI YES | whole current NO | 현재 source·세 CUDA와 명시 지원표 결합 |
| npm/PNPM/Bun Node·UV/pipx Python one-off | YES | YES | 기존 named source/실제 whole approval/CLI YES | whole current NO | 새 source ordinary 및 final matrix |
| 명시 Bun Node-compatible JS program | YES | YES | [G3 local](./p6g3-bun-readonly-program-local-20261030.json) YES | hosted/whole NO | 새 source ordinary·단일 full12 controlled cohort·최종 matrix |
| CPU+선택 CUDA full·typed input·cleanup/Required10 | YES | YES | 과거 frozen source YES; 현재 NO | NO | 같은 최종 source의 ordinary/CPU·cu126·cu130·cu132 실제 결과 |
| 현재 규모에서 CUDA 전용 실행 범위 분리 | NO | NO | NO | NO | ordinary 소비자는 전부 유지하고 CUDA 전용에서 unrelated client 반복을 제거; 실제 profile 출력/원래 실패·cleanup 유지 |
| 최종 지원표·docs/help·원본/정리 인계 | 부분 YES | 부분 YES | 부분 YES | NO | 아래 지원 한계와 실제 profile 결과를 canonical leaf/Queue에 결합 |

각 명령/옵션의 의미는 client09의 현재 lane이 소유한다. 새 동등성·지원 약속을 만들지
않는다. Bun-only shebang/TypeScript entry 자동 선택·일반 native 동등성, 임의 native 옵션,
stdin/TTY·생성 파일 반출·arbitrary child/held pipe는 기존 미구현/미지원 범위를 보존한다.
이들을 검증 완료나 owner가 승인한 deferral로 바꾸지 않는다. 최종 인계에서 구현 미완료와
명시 거부를 구분하고, 전체 지원 완료로 표시하지 않는다. 서명 배포·공식 채널 이름과
publication은 M13의 별도 미완료이며 이 profile 작업으로 닫히지 않는다.

## 실행 전 시간 근거와 최소 수정 지도

CI439 whole6254s와 CI440에서 실제 통과한 G1/G2 새 부모143.22s를 더하면 약106분37초다.
G3 앱/CLI/controlled12는 그 밖의 추가 실행이다. 과거 같은-source CUDA full은
cu1261244.38s/cu1301192.58s/cu132878.63s였다. 이는 추정 근거이며 현재 runner의 완료
보장이 아니다. 지금 모든 client를 CUDA마다 반복하면120분 한도를 넘을 위험이 있다.
기존 manual input은 이미 "bounded CUDA full only"를 명시하지만 실제 workflow는 전체
일반 소비자 뒤에 CUDA를 덧붙인다. 한도를 늘리거나 실패 뒤 재시도를 반복하지 않는다.

- 기존 workflow의 세 closed boolean input, CPU+선택 CUDA의 exact 명령/augmentation,
  strict freshness, 최대 한 profile, 120분 job/40분 CUDA test·제품 한도와 Required10을 유지한다.
- ordinary PR/default는 기존 모든 소비자와 G3 새3parents를 그대로 실행한다.
- CUDA-only manual은 같은 보호 설치·helper/observer·공통 runtime 확인·CPU+선택 CUDA
  full을 실행한다. 별도 job/helper/새 완료 protocol을 만들지 않는다. 해당 profile의 성공을
  ordinary whole 성공으로 사용하지 않고 같은 frozen source의 별도 ordinary 결과도 요구한다.
- 기존 setup·client compile/qualification·cleanup owner를 재사용하고 필요한 조건 연결만
  추가한다. 정확한 dispatch/일반 소비자 보존·CPU/CUDA inputs·원래 실패/cleanup은 싼
  maintained YAML/shell 검사와 실제 입력 block 회귀로 먼저 검증한다.
- 의미 있는 최종 묶음의 Quick/checker Race/docs/security 뒤 한 번 push한다. 현재 CI는
  종료됐고 timer는 없다. 새 run을 실제 확인한 뒤20분 GET을 시작한다. 같은 source의
  full/native/CPP는 이 CI 전용 수정 때문에 반복하지 않는다.

## 두 번째 검토와 종료 조건

기존 구현을 재작성하지 않는다. 검증 중 새 source 동작 gap이 나오면 최초 실패를 보존하고
해당 owner만 수정하며 지원/예산/Policy를 바꾸지 않는다. 네 profile은 동일 source에 연결해
각각 판정하고 실제 checkout·원본·target/terminal/cleanup을 확인한다. 새-source ordinary와
세 CUDA 및 지원표가 미완료인 동안 전체 P2/P3/P4/P5/P6/P7 acceptance는 OPEN이다.

최소 연결은 workflow19줄 추가와 기존 두 exact qualification owner의 동일 guard prefix 결합이다. 첫 cheap workflow 검사0.176s는 이 두 owner 누락을 올바르게 거부했고 source/실제 검증을 시작하지 않았다. 같은 닫힌 guard를 expected body에 추가한 format1.585s/workflow1.732s/실제 입력·분기 focused1.571s는 PASS다. 일반 실행은 모든 기존 명령의 순서를 유지하며, CUDA manual일 때만 세 일반 cohort·후속 실행 lane을 별도 ordinary qualification 대상으로 표시한다. 새 helper/job·기존 guard 삭제/우회·제품 변경은 없다. 최종 Quick/checker Race/docs/security와 새-source 실제 CI는 미완료다.

첫 Quick72.365s는 기존 kernel terminal Bash guard fixture에서 새 필수 job scope env가 없다는 unbound-variable 오류 두 개로 FAIL했다. 실제 workflow job env는 이미 고정됐고 fixture가 그 입력을 전달하지 않았다. Test의 ordinary 환경에 false를 명시해 원래 complete0/primary37·누락/중복/SKIP 거부를 그대로 검증한다. 원본 raw p7-first-quick-profile-input을 보존하고 checker만 먼저 재검증한다.

Ordinary job env를 명시한 기존 Bash fixture와 전체 checker14.583s·canonical workflow1.890s는 PASS다. 최대 한 CUDA 선택의 실제 Bash 거부와 ordinary/manual 분기·step scope override 거부도 확인했다. 제품/observer source diff는0이다. 최종6개 CI source를 Root/private hash로 고정한 뒤 Quick/checker Race를 진행한다.

최종 CI-only6 source의 Quick63.797s·전체 checker Race84.552s·strict qualification-freshness0.141s는 PASS다. [Local receipt](./p7-cuda-profile-routing-local-20261030.json)가 exact source/최초 실패·후속 검증을 연결한다. G3 product/observer 입력을 변경하지 않았으므로 CPP/native/affected product Race를 반복하지 않았다. 새 grouped ordinary와 동일 후보의 CUDA3 actual 및 최종 지원표/인계는 여전히 OPEN이다.

## 2026-10-31 ec1 frozen 후보의 최종 입력 대조

H1/H2와 승인된 전체 Linux150분을 한 번 모은 sourceec1c260/tree84e70a9e의
[CI442](./ci442-ordinary-cpu-qualification-20261031.json)는 Required10SUCCESS로 종료됐다. 위120분 유지 기록은 이전
baseline이며 [사용자 승인 예외](../../20-package-client-quarantine-plan.md#5-검증-matrix와-자원-정리)가
현재 전체 job에 적용된다. CPU15분·선택 CUDA40분과 Required10·최대 한 CUDA/원래 실패·정리는 그대로다.

같은 source의 workflow 선택 block을 CI441/source1ee와 비교해 동일함을 확인했고
runtime lock/qualification owner/test 입력도 현재 commit bytes와 대조했다. 새 source/테스트를
추가하거나 freshness/full/native/CPP를 반복 실행하지 않았다. 기존 explicit
cuda-bindings-numpy(NumPy2.4.6 exact SHA)와 triton.profiler.viewer의 not-attested 입력은
qualification 전용이며 artifact 오류에서 추론하거나 정상 설치의 권한으로 확대하지 않는다.

| 최종 항목 | 현재 상태 | 다음 필수 증거 |
| --- | --- | --- |
| ordinary·CPU | CI442 QUALIFIED; all10/전체 tree·124parents·CPU full/정리 확인 | 재실행하지 않음; 같은 source의 CUDA3와 최종 지원표 결합 |
| cu126 | [CI443 attempt2 QUALIFIED](./p7-cu126-final-qualification-20261031.json); CPU264.94s/cu1261208.54s·all10/동일 source·정리 | 재실행하지 않음; 첫 실패 원인은 UNKNOWN 유지 |
| cu130 | [CI444 attempt2 QUALIFIED](./p7-cu130-final-qualification-20261031.json); retained8/새Mac·Required, CPU300.39s/cu1301145.81s·all10/source·정리 | 재실행하지 않음; 첫 Mac로그 부재·하위원인 UNKNOWN 유지 |
| cu132 | [CI445 QUALIFIED](./p7-cu132-final-qualification-20261031.json); CPU292.29s/cu1321078.54s·all10/source·정리 | 재실행하지 않음; 최종 지원표/문서 묶음 검증 |

임시 입력3개는 /tmp/haa-p7-ec1-final-input-audit-20261031에 준비했고 현재 cu126만 dispatch했다.
Ordinary 성공과 최초 실패 disposition·source 고정을 확인해 한 번에 하나씩 실행한다.
연결 GitHub 도구에는 dispatch 기능이 없고 현재 PATH의 gh도 없다. 이 repository push에
사용한 정상 Git credential helper의 사용 가능 여부만 확인했으며 자격 값은 출력·저장하지 않았다.
실제 manual 실행 때 이 기존 인증의 API 경로를 사용하고 성공한 run/source를 직접 확인한다.
이 준비를 실제 CI 실행이나 qualification으로 쓰지 않는다.

최종 지원표 대조 중 current UV lane에 남은 과거 --with 거부 문구와 P7 matrix의 CI441
PENDING/H2 미구현 문구를 정정했다. 변경은 문서에만 모았으며 CI442 중 push하지 않는다.
Product/source51은 frozen bytes 그대로다. Raw/실행 결과가 나오기 전 whole acceptance를 닫지 않는다.

CI442의 첫 실제20분 GET(17:17:01UTC)은 macOS/Corpus를 포함한8SUCCESS와 Linux 진행을
확인했다. 완료8 logs447497B의 checkout581dc94f/tree84e70a9e는 frozen 후보와 일치한다.
CI441의 Mac/Corpus 실패가 이번 실행에서는 재현되지 않았지만 역사적 하위원인은
미확정으로 유지한다. 같은8 logs를 다시 받지 않으며 Linux/Required의 종료 증거가 남는다.

CI442는 일곱 번째 정시 GET에서 Required10SUCCESS로 종료됐다. All10 checkout581dc94f/
전체tree84e70a9e·decoded1422924B, Linux124 unique PASS/138m45s·H1 single full8/H2 CLI3·
CPUfull302.91s와 helper18wait0/final4정리를 [확인](./ci442-ordinary-cpu-qualification-20261031.json)했다.
Worker는 COMPLETED/nextnull·Host프로세스부재/sessionclosed이며 bridge가 마지막 exec metadata를
보존하지 않아 직접 관측한 exit0은 기록하지 않는다. 다음 bridge는 전체 반환을 보존한다.

이후 exact ref/no-active-manual을 대조하고 cu126만true인 기존 workflow를 공식 API로 한 번
시작했다. API의 return_run_details가 실제 run41115499227/CI443을 반환했고 Sourceec1·
workflow_dispatch/start19:28:12UTC를 확인했다. [등록](./ci443-cu126-registration-20261031.json)의
등록 당시 첫20분 due는19:48:12UTC(한국04:48:12), 실제 정시조회는0이었다. Credentials는 메모리에서만
사용했고 source/runtime/input·job/profile·제품 한도를 변경하지 않았다. cu130/cu132는 아직미실행이다.

CI443 attempt1은 정시 GET2/20:08:13UTC에7SUCCESS/Mac·Linux·RequiredFAIL로 종료됐다.
원본10logs/1425555B·checkoutec1/정리를 보존했고 worker36432의 자연 exit0/nextnull을 확인했다.
[최초 cu126 실패](./ci443-cu126-first-failure-20261031.json)는 CPU291.08s PASS 뒤 preparation의
trusted Read1088.664ms/1s·대응 helper engine1167.707ms이며 하위원인은 UNKNOWN이다.
성공CI398의 관련8owner는 동일하고 당시 더 높은 pressure도 통과했으므로 pressure 단독 인과나
한도 변경을 정당화하지 않는다. Mac도 별도의 production2min deadline/default NOT_REACHED다.
Source/한도를 유지하고 실패3jobs만 한 번 명시 재검증하는 attempt2가20:11:55UTC 시작됐다.
기존7성공을 재실행하지 않는다. 같은 조회 실패가 반복되면 재시도 대신 해당 helper engine
구간만 진단하며, 성공해도 과거 지연을 수리했다고 쓰지 않는다. 현재20분 timer/실제 결과는
CI443 registration이 소유하고 cu130/cu132는 cu126 qualification 전 미시작이다.

CI443 attempt2는 정시GET3/21:12:00UTC에서 all10SUCCESS였다. [cu126 qualification](./p7-cu126-final-qualification-20261031.json)은 retained7+새3 원본1842515B와 동일source/CPU264.94s·cu1261208.54s/NumPy검사전용·NOT_ATTESTED/전체정리·worker자연0을 확인했다. 첫 Read/build 지연 원인은UNKNOWN이다. 다음cu130-only CI444가21:14:19UTC에 같은 source로 시작됐으며 cu132와 최종 지원표/whole acceptance는 아직OPEN이다.

최종cu132 CI445는 정시GET3/23:36:23UTC에all10SUCCESS로 종료됐다. CPU292.29s/cu1321078.54s·전체1779489B/동일source/explicit input/NOT_ATTESTED·정리와 worker89839자연0/nextnull을 확인했다. 네 profile의 현재 결과와 미구현 범위는 [최종 matrix](./p7-command-acceptance-matrix-20261030.md)와 [qualification](./p7-final-qualification-20261031.json)이 소유한다. 새 CI 없이 최종 문서 묶음만 검증한다.
