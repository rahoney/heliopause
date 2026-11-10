# AGENTS.md 재개 지침 이력 (2026-10-31 ~ 2026-11-01 시점)

2026-11-10 R0-4 문서 정리로 `AGENTS.md`의 "사용자 최우선 재개 지침 — 2026-11-01" 절 원본 줄 10-139를 그대로 옮겼다(본문 변경 없음). 당시 기록이며 현재 상태 판정이 아니다. 지속 규칙은 `AGENTS.md`의 "유지할 사용자 지침" 요약이, 최신 상태는 [Queue](../02-current-work-queue.md)가 소유한다.

### 사용자 최우선 재개 지침 — 2026-11-01

- 최신 사용자가 planning21 순차 구현을 승인했다. 과거 문서 전용/시작 금지 인계는
  superseded다. B00부터 하나씩 수행하고 설계 대조·필수 검증·정리까지 계속한다.
  기존 owner 재사용/최소 수정·묶음 CI·활성 응답의 실제20분 감시는 유지한다.

- 누적 설계 검토는 baseline9aa7430에서 실행기 schema/guest exit와 검역 판정 구분,
  MCP 운영자 허용 경로·취소/응답 유실, A01/A02의 공통 검증 재사용·최종 보안 검토 연결을
  명확히 했다. 현재도 문서 전용/구현 NOT_STARTED이며 순서·버전 owner·필수 검사를 유지한다.

- 최신 AI 사용 요청은 보강 설계 §4.8–4.9를 따른다. A01 전체 생태계 CLI 결과·대안
  재검역·우회 금지 Skill은 이번 필수 acceptance이고 A02 최소 로컬 stdio MCP는 기존
  owner 재사용·lifecycle/client 검증 조건을 충족하면 함께 구현할 후보다. 기존20개 B/S/C
  gap·16개 순서의 통합 acceptance에 연결했으며 현재는 문서 반영만/구현 NOT_STARTED다.
  Go/Cargo/Terraform 일부 update의 기존 JSON을 재구현하지 않고 실제 출력 공백만 닫는다.
  Skill/MCP를 Host 우회 강제 차단으로 표시하지 않는다. 아래 planning22 AI 생성 코드
  검사는 계속 향후 범위이며 기본 MCP 연결과 구분한다. 사용자 AI 설정·Host·CI 변경 없음.

- MCP 사용자 AI 코드 제출/최대2회 시도와 자체 모델/API/AI CLI 생성은
  `docs/planning/22-future-ai-assisted-inspection.md`의 향후/v2.0 후보로 기록만 한다.
  현재 보강20개 gap·16개 순서에 추가하거나 구현을 시작하지 않는다. MCP 제공 자체도
  현재 완료로 가정하지 않으며, AI 실패 뒤 생략을 검사 통과로 표시하지 않는다.

- 최신 후속 요청의 보안 검토는 보강 설계 §3.1·§5.6·§11.1과 Queue에 반영한다.
  기존 버전 단일 owner/plan-apply-verify를 보존하고 기본 필수 동적 검사를 옵션으로 내리지
  않는다. 기존16번의 마지막 보안 평가에서 source 공개 전제, 탐지/차단/오탐/미완료와 비용을
  구분한다. 심층 profile은 실측 후 판단하는 설계 후보이며 새 UI/제품 구현은 아직 시작하지 않는다.

- 최신 요청은 보강 설계의 최종 누락·충돌 점검까지이며 실제 구현은 시작하지 않는다.
  `docs/planning/21-goal-completion-design.md`와 Queue의 최종 설계 대조를 먼저 읽는다.
  기존 20개 gap·16개 순서는 유지하고 설치본 공급/호환성·기본 판정·명명/배포 인계를 보완했다.
  기능 완료는 명명/배포 마무리 진입 조건이며 채널 구현·서명 후보·clean Host/게시 검증까지
  완료했다는 뜻이 아니다. 아래 이전의 즉시 B00 착수 인계보다 최신 설계 전용 지시를 우선한다.

- 최신 사용자 요청의 일괄 설치/원래 설계 구현 감사는 goal-coverage-audit-20261101.md가 단일 상세 목록을 소유한다. 감사 완료와 B00~B07/S01~S09 기능 OPEN을 구분한다. requirements 외 npm 기본 install/ci, Cargo fetch, Terraform 기본 init도 미지원이며 npm ci/Cargo fetch/Go mod tidy는 실제 help exit0이다. 다음은 기존 공통 CLI의 명시 실패와 requirements 묶음, 다른 기본 설치 순서다. 모의 환경 외 MVP-005~008의 악성 패턴·사용자 package 취약점·위험 변화·일반 공급자 증명·상태 diff/필수 범위도 남는다. 기존 narrow milestone의 제외를 사용자 원래 목표 완료로 바꾸지 않는다. 이번은 source/preflight/문서 감사이며 제품·Host·새 CI 변경 없음. 설치본·승인 아트는 보존한다. 감사 문서5경로는 local df3b62da8c1e8e580ad9a7304bcc69c395fe65c3으로 보존했고 raw-free canonical docs2.241s PASS·검사/실제 staged bytes 동일이다. Tracking보다5commit 앞이며 push하지 않았다. 원본과 commit/validation은 docs/github-logs/goal-coverage-audit-20261101이 소유한다.

- 최우선 최신 교정: 사용자의 핵심 목적은 감염된 공급자의 artifact가 분석환경에서는 잠복하다 실제 개발환경에서 활성화되는 위협을 설치 전에 유도·탐지하는 것이다. 두 helox pip/uv pip install -r는 실제 unknown shorthand flag/exit1로 미지원이고 D-008 모의 env/키 파일·격리 내부 응답과 조건부 잠복 유도 검증도 현재 표준 Python 경로에서 미완료다. 기존 전체 감사/CI/로컬 Host 완료를 이 목표의 완료로 쓰지 않는다. Queue와 local-dev-test-20261101.md의 목표 재대조 OPEN을 먼저 따른다. 이름·배포를 다음 완료 조건으로 앞세우거나 사용자의 원래 목적을 옵션 동등성 제외로 치환하지 않는다. 이번은 baseline/source/preflight 대조이며 제품·Policy·Host·CI를 변경하지 않았다. 상태 정정은 로컬 fed87ce9081e3abd07ef2dec76aed7961d3cda4d의 문서4경로로 보존했고 raw-free docs2.384s PASS다. 현재 tracking보다4commit 앞이며 push하지 않았다.

- 최신 사용자 요청은 같은 WSL의 micromamba `helox`를 다른 터미널에서 활성화해 패키지 설치·업데이트를 직접 체험할 수 있도록 설정하는 것이다. 사용자가 개발 Host 전환을 승인했고 설정은 COMPLETE다. 로컬 commit `0f4b933bb5c9df4645db1d6dce4383b1db34ef36`이며 정확한5경로(기존 unit의 serve 한 줄+관련문서)만 보존했다. 최신 CLI code는4288 소스이며 `/usr/local/bin/helox`가 root 소유 최신 개발 설치를 가리킨다. 이전의 설치본 교체 없음/Host29.8.1/정책 서비스 미준비 인계보다 이 항목과 실제 Queue를 우선한다.
- Docker/CLI29.9.2·containerd2.4.2, 최신 CLI/helper·검증된 observer4ca33a74·현재 pod-init을 적용했고 고정 gVisor6개/image5개는 재사용했다. 정책 서비스는 active/enabled, runtime 진단4개 OK다. 공식 release 기록은 만들지 않았으므로 RELEASE_INSTALLATION_UNAVAILABLE와 doctor exit1은 남는다. 개발 설치를 정식 배포·self-update 완료로 전용하거나 경고를 숨기지 않는다.
- 실제 동일 helox 환경에서 packaging24.2 설치27.245s→25.0 업데이트29.149s, Go uuid1.5.0 get21.527s→1.6.0 update19.521s→관찰 build67.366s가 COMPLETED/ALLOW다. container0·관찰 디렉터리 empty·보호파일 지문 동일을 확인했다. 첫 unit 누락 serve와 기록기의 Go 미지원 강제 JSON 오류는 raw/기존 L15에 보존했고 재구현 없이 필요한 부분만 교정했다. canonical raw-free docs2.214s PASS다. 같은 성공한 소비자/full/native/CPP/CUDA/CI를 재실행하지 않는다.
- 개발 micromamba의 Python3.14.7/패키지는 유지했다. 사용자 체험 폴더는 `$HOME/helox-tryout-20261101`이며 python/.venv는 비어 있고 go/에는 의존성이 없다. canonical 사용법은 docs/planning/evidence/m13-004-convenience/local-dev-test-20261101.md, raw는 docs/github-logs/local-dev-test-20261101이다. 기존 설치7파일/이전Docker3패키지는 /var/backups/heliopause/dev-test-20261101-4288ddd에 보존했고 root-only restore.py는 구문확인만 했다. 자동 복원·사용자 폴더/검사 Evidence 삭제는 하지 않는다.
- 현재 tracking보다 로컬3commit 앞이며 새 push/CI/타이머는 없다. M13-004 전체의 공식 이름/publisher binding·채널·현재 signed 배포는 별도다. 이름은 사용자가 검토하므로 재질문하지 않는다. 아래 아트 마감과 과거 Host checkpoint는 당시 기록이다.

- 사용자가 단순 문 로고 시안을 최종 확정했다. 로고는 HELIOPAUSE 왼쪽에만, 두 번째 ARTIFACT AIRLOCK은 전체 폭 가운데에 둔다. 19×5 단색 문 윤곽·회색 중앙 분할·짧은 노란 손잡이, 가로 RGB 그라데이션과 고정 별빛을 유지한다. 큰 gear/metal shading·우측 text column·거부된 크기 비교안을 다시 기본으로 진행하지 않는다. 시각 재승인은 필요 없다.
- 확정 아트는 로컬 commit `4288dddad065e5bb9b0b90f3cf3740e000dbc705`의 정확한10경로로 보존했다. 승인된 Go 두 파일은 canonical leaf의 SHA와 같고 raw-free final Quick93.086s/security21.332s/최종 docs0.348s PASS다. IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES, MISSING=[]는 이 local 표시 범위다. `docs/planning/evidence/cli-launcher/art-refresh-20261031.md`와 `docs/github-logs/cli-art-20261101/final-closure.json`이 증거를 소유한다. 선택된3개 PNG만 commit했고 미선택 초안18개와 이전 인계는 hash를 보존해 raw drafts로 옮겼다.
- 현재 원격 tracking보다 로컬2commit 앞이며 push·새 CI·설치본 교체·main merge/tag/publish는 수행하지 않았다. 활성 CI/타이머는 없다. 같은 불변 Quick/full/native/CPP/CUDA/CI를 반복하지 않는다. 전체 감사의 code358 qualification은 그 source 범위이며 새 아트의 hosted 성공으로 전용하지 않는다. M13-004 전체는 이름/publisher binding·채널·현재 서명배포/clean-Host acceptance가 남아 IN_PROGRESS다. 이름은 사용자가 검토하므로 재질문하지 않는다. 아래 과거 checkpoint의 당시 상태보다 이 항목과 실제 Git/Queue를 우선한다.

- 최신 사용자 요청은 이름 결정 전 전체 코드·보안 감사다. M13-004 안의 full-source-audit-20261031.md와 Queue를 따른다. 전체 범위가 명시적으로 승인됐으며 기존 bounded 기능/CI 성공을 전체 감사 완료로 쓰지 않는다. 실제 확정 문제만 최소 수정하고 검증을 모으며 이름은 재질문하지 않는다.
- 전체 코드·보안 감사는 COMPLETE다. 최종 code358/treee1ca의 일반/CPU448·cu126449·cu130450attempt2·cu132451이 모두 Required10SUCCESS이며 raw40개6691308B·actual source/skip0/helper 회수/복원을 대조했다. 기존 확정 F01..F08을 최소 수정했고 미사용·중복·복잡도/보안 경계 검토를 마쳤다. 상세와 감사 한계는 canonical audit MD/JSON을 따른다. 최종 문서/단일JSON5개는 e975e5c1fa0cb4de420e7870775baa01b732a42d 로컬 커밋으로 보존했고 raw-free canonical docs0.949s PASS·staged byte/mode/path 확인을 마쳤다. 원격 qualified code는358이며 문서만으로 full CI를 재유발하지 않도록 추가push하지 않아 tracking보다1commit앞이다. 재개 시 미추적 자료를 stage하거나 이 문서 commit만으로 새 CI를 만들지 않는다.
- 현재 검증 코드358/treee1ca의 감사 scope는 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES, MISSING=[]다. CI451 actual20minGET3/13:54:27.455226UTC terminal10SUCCESS·CPU290.53/cu1321064.79/noSKIP/CPP133/helper2wait0/confirmedRESTORED를 같은 활성turn에서 소비했다. 모든worker는자연exit0/nextnull이며 활성CI/타이머없다. CI450 첫Mac120s buildtimeout 하위원인은UNKNOWN이고 Mac/Required만한번retry해성공8/GPU를재사용했다. 같은불변full/native/CPP/CI나완료감사를반복하지않는다. 이름은사용자가검토하며재질문하지않는다. M13-004 전체는 이름/publisher binding·채널·현재 서명배포/clean-Host acceptance가남아 IN_PROGRESS다. 로컬Host29.8.1은새floor29.9.2 미달이며전역교체/재시작/영구activation을수행하지않았다. Mainmerge/tag/publish별도다. 최신상태는Queue·audit단일원장이고아래이전checkpoint는당시기록이다.

- CI447 당시 pushed sourcef6c0349/treef2cf의CI447/run41180577563은7SUCCESS·Mac/Linux/RequiredFAIL이다. 실제20minGET1/07:45:57.838467UTC 소비/worker1058029(session49581)자연exit0/nextnull이며활성CI·타이머없다. 후속원인확정: Mac새test의긴socket경로는shortprivate temp로고쳐같은53byteTMPDIR oldFAIL/newPASS다. Linux는newimmutablePython14.10/pip26.2.2 dist-info와CPP old26.2.1 exact경로의불일치다. 기존2경로만수정하고currentread/old·neighbor·write거부를같은latch에묶어newCPP133/54.526s PASS(helper4ca33a74, observer2f8dde13/latch8c57dc93)다. 같은CPP 재빌드하지않는다. 현재source3개+관련doc의canonical Quick/security/docs를모아검증한뒤새commit/한번push CI를진행하고같은20minworker를재사용한다. Runtime lock/Go production/SDK/pod-init은f6와같으므로localconfiguration7/vulnerability/history/OSV의불변범위만재사용하고hostedCPU/CUDA는미도달이다. 현재Git/Queue/audit원장과/tmp/haa-full-audit-20261031·docs/github-logs/full-source-audit-20261031이원본·pending을소유한다. 이름재질문/게시/Host전역activation없음, 연속작업을중간final로끝내지않는다.


- 재개 시 이 절·Current Work Queue·오답노트를 실제 Git/입력과 대조한다. 단일 active는
  M13-004, branch milestone/m13-release-build-packages다. CI446 당시 frozen/pushed HEAD는
  e3b2b7011825c6322e5aa8d031716ff51f752a27/tree0cdaeaa206c10aa9b93ac45c2c418aab68948d3d다.
  제품/검사19+관련문서10을한글commit으로기록했고 normal FF ec1→e3로한번push했다.
  Source19는검증지문/committedbytes가같다. Local232319f는 CI446 완료 문서12개, 현재HEAD21a0a25214cce425d1204c63a47935170f68ca71은 이름독립review/UV검사 최소수정을 더한 commit이다. 둘 다 추가push하지 않았다. 새 CI는 없다.
- 사용자교정대로 기능구현·acceptance를 먼저 마친 뒤 이름/배포를 진행한다. 과거이름질문은
  현재작업에대해superseded이며이름답변은blocker가아니다. P7 matrix의2026-10-31 기능
  우선재대조를따른다. 모든native옵션동등성은원래약속이아니며 stdin/TTY/파일반출의별도
  확장을필수누락과혼합하지않는다. 미구현을승인유예로바꾸거나checkpoint를전체완료로쓰지않는다.
- 현재 G4 Bun 선언/TS entry·runtime 선택은구현/연결·named local qualified다.
  p6g4-bun-entry-runtime-baseline과 p6g4-bun-declared-entry-local receipt가소유한다.
  실제public새앱2/56.51s·CLI3/83.63s·기존Node2/69.13s/operator211.563s는exit0·noSKIP/
  helperwait0/Host5·Docker동일/profile같은inode원복이다. 첫v2는privateBun공급누락으로
  entry미도달이었고제품변경없이기존고정공급을복구했다. 기존공용PNPM/Bun guard fixture
  누락은같은Bun분기만확장했고별도중복fixture를제거했다. 최초두실패원본은보존했다.
- Production소비자없는runtime override두wrapper를Node경로에inline한최종후보의
  Quick68.579s/BootstrapRace15.841s·taggedvet2.946s/고정Staticcheck2.456s/
  docs0.312s/security15.304s와최종metadata-docs가PASS다. V5영향4Race130.781s/
  checkerRace91.082s의불변3package/checker결과를재사용했다. 실제snapshot과최종후보의
  차이3파일·같은selector/commoncontext body를receipt에서구분했다. CPP/권한/한도는불변이다.
- CI446/run41159991137/sourcee3b2b70은 전체10SUCCESS로 종료됐다. 실제20분GET5/
  04:06:10UTC/13:06:10KST를 같은 활성 응답에서 소비했고 session81628/worker909486의
  자연exit0/nextnull을 직접 관측했다. 활성 CI/타이머는 없다. 전체10 logs1331441B는
  기존8개를재사용하고마지막2개만새로받았다. 실제checkoute757cc1b/tree0cdaeaa2는
  frozen source와같으며 Linux126 unique PASS/noSKIP·CPU192.40s·새앱2/25.34s·CLI3/39.35s,
  helper18wait0/최종4original0·확인된정리/복원을검증했다. CI446 hosted qualification receipt가
  원본/입력/검증경계를소유한다. 같은CPP/native/full/CI를다시실행하지않는다.
- 현재 이름독립review: /tmp/haa-m13-current-release-review-source는clean e3 worktree다.
  /tmp/haa-m13-current-release-review의6 Go1.27.3/VCS-clean build·unsigned19파일601611160B/
  15assets·기존verify-bundle와wrong-source거부가PASS다. 서명/공식tag/Host설치가아니다.
  고정 CPP/PNPM/Bun/Pipx입력을재사용했고 같은 native/full/CPP/CI를반복하지않았다.
- 확정gap: runtime manifest5개중UV를release workflow와validator가registry 검사하지않았다.
  기존3파일만수정해같은step의UV선택/inspect2줄·기존ordered helper·기존test6반례를연결했다.
  /tmp/haa-m13-uv-release-inspect-review의RED2.306s → format1.680/workflow1.703/
  focused1.595/Quick90.498/checkerRace94.716s PASS다. 새함수/job/profile/package는없다.
  최종docs0.335s/security12.563s·staged6파일/source3지문확인도PASS다. Local21a0a25로
  기존3파일14줄과문서3개만커밋했다. 추가push/CI는없다. 이름재질문하지않는다.
- 로컬Buildx미설치로exactinspect는미도달이다. 기존Docker manifest명령도같은digest를
  거부했으나공식GHCR원본669B의bodySHA/응답digest가lock과같다. 로컬Docker하위원인은
  UNKNOWN으로보존하며pin/보안조건은바꾸지않았다. 익명pull token은저장/출력하지않았다.
  이metadata확인을Buildx/실제protectedReleaseBuild성공으로전용하지않는다. raw/evidence는
  name-independent-release-review-20261031.json과baseline20261017 current review가소유한다.
- 최신 사용자 지침: 이름은 사용자가 고민하므로 재질문하거나 이를 blocker로 삼지 않는다.
  이름과 무관한 미완료·미검토 책임을 실제 코드/검증과 대조해 완성도를 높인다. 첫 범위는
  기존 verified installer/provenance caller와 최초 bootstrap 연결을 먼저 대조했다. ordinary
  bootstrap prepare/root bootstrap-files도 구현·연결돼 있어 재작성하지 않는다. baseline-20261017의
  새 독립 review 표에 따라 current-source unsigned native/package/manifest 조립만 로컬 확인한다.
  구현/검토 완료 owner를 다시 작성하지 않으며 실제 확인된 gap만 처리한다.
- 원래 bounded P2..P7 기능 acceptance는COMPLETE다. M13-004전체는아직IN_PROGRESS이며
  사용자 이름 검토 동안 다음은 기존 producer의 current-source 로컬 native 조립 검토다. 채널/새signed배포는 별도다. 채널이름의이전
  질문은기능미완료일때superseded였고이제사용자가검토중이며그동안이름과무관한리뷰를진행한다. 함수/모듈을다시
  구현하지않고기존verifier/nativeinstaller를재사용한다. 이름답변없이publication식별자를
  임의확정하거나placeholder channel package를만들지않는다. Main merge/tag/publish는별도다.
- ec1/tree84e의이전ordinaryCI442·cu126CI443attempt2·cu130CI444attempt2·cu132CI445는각
  Required10SUCCESS다. p7-final-qualification과각receipt가그source/정리/첫실패UNKNOWN을
  소유한다. 새e3 tree의네profile성공으로전용하지않는다. G4는NPM/Bunentry/Bootstrap/CLI
  변경이며 Sandbox/C++/Python/runtime lock/CUDAowner는불변이다. 영향선정근거는G4receipt와
  예방규칙4/5·계획20§5다. 같은불변CUDA3를새helper마다재실행하지않는다.
- 사용자 추가 지침: 현재 필요한 작업 범위만 수정한다. 기존 모듈을 재사용하고, 새 함수·
  추상화·연결·파일 분할은 중복 제거나 책임 분리에 실제 도움이 될 때만 추가한다.
  단순한 호출 흐름과 명확한 모듈 책임을 유지하며 과도한 파편화로 유지보수를 어렵게 하지 않는다.
  로컬 변경과 필요한 검증을 모아 반드시 필요한 시점에만 CI를 실행한다. 진행 중인 CI는
  기존 20분 타이머로 실제 상태를 확인하며 같은 목적의 타이머나 CI를 중복 실행하지 않는다.
- 조사/수정은현재실패와acceptance에필요한범위만수행하고새구조·미래소비자용연결·관련없는
  수정을늘리지않는다. 싼표적검사부터필수검증을의미있는묶음으로모은다. UNKNOWN은부재가
  아니며신원/조회불명에서privileged restore/rebind하지않는다. 제품/개별test/Mac한도는불변이며
  전체Linux150분만사용자승인됐다. 관련없는untracked/원본/lessons를stage하지않는다.
- 사용자 강화 지침(2026-10-31): 현재 연속 작업에서는 CI 대기를 이유로 응답을 종료하지 않는다.
  조회 워커 등록·조회 한 번·부분 성공은 종료 조건이 아니다. 같은 활성 응답에서 시계 대기와
  기존 20분 타이머 결과 소비를 이어가고, CI 종료 시 실제 결과·최초 실패를 확인한 뒤
  후속 분석/수정 또는 acceptance 마무리를 계속한다. 일반 외부 CI 대기 종료 예외보다
  이 현재 사용자 지시를 우선한다. 새 타이머·CI·불필요한 코드 변경은 만들지 않는다.
- 이번에는활성응답을유지해CI종료·원본/정리분석과기능acceptance까지이어갔다. 이후새CI가
  필수가되면같은원칙으로기존타이머와실제결과소비를연결하고중간에응답을끝내지않는다.

