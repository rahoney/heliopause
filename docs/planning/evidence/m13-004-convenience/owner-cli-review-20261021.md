# M13-004 — 배포 전 owner CLI 확인

상태: **READY_FOR_OWNER_REVIEW**. M13-004만 IN_PROGRESS다. 기능 검증과 별도로
사용자가 최신 화면을 직접 확인하기로 했다. 기존 CLI-UX-003/native qualification을
재개방하거나 기능을 다시 구현하지 않는다.

최초 preview 기준: source918e329/tree38b2f3b, 실제 CI359 Required+9 SUCCESS.
현재 preview는 아래 full-name 보완 source1f2500e의 clean build다. 정확한
source/digest/build metadata는 local raw fullname-preview.json이 소유한다. 이전8d949a7은 교체됐다.
현재 PC의 `/usr/local/bin/helox`는 `/usr/libexec/heliopause/helox`를 가리키며
build metadata는4d6cc0f/modified=true다. 이 metadata만으로 당시 overlay 전체를
복원했다고 주장하지 않는다. 현재 설치본을 최신 source라고 안내하지 않는다.

## 확인 범위와 실행

이 WSL 세션에만 준비한 개발 검토용 실행 파일이다. 공개 설치 경로나 정식1.0.0
배포물이 아니다. 다른 PC/CI에 이 임시 파일이 자동으로 존재한다고 가정하지 않는다.

```bash
/tmp/helox-cli-review-20261021/helox --lang ko
/tmp/helox-cli-review-20261021/helox --lang ko --help
/tmp/helox-cli-review-20261021/helox --lang ko self-update --help
```

첫 명령을 일반 사용자의 실제 WSL/VS Code 터미널에서 실행한다. 방향키로 이동하고
Enter로 선택, Esc로 돌아가며 q로 종료한다. 시작 화면의 풀네임, 사용법/예시, 언어
선택, 도움말의 새 update/self-update 명령을 확인한다. 언어 설정과 cache 경로는
이 디렉터리 안의 config/cache로 분리했으며 기존 사용자 설정을 변경하지 않는다.
처음 preference는 auto이며 실제 config 조회는 ko/WINDOWS_UI를 보고했다.

이 검토용 user-owned binary는 보호된 설치에 등록하지 않았다. 실제 artifact 검역,
설치·본체 업데이트와 그 progress/result 확인은 인증된 신뢰 환경에서 별도 진행해야
한다. Preview의 readiness 거부를 제품 검역 실패나 설치 성공으로 바꾸지 않는다.
실제 signed self-update/취소·회복 결과는 [qualification](./signed-native-qualification-20261020.md)이
소유한다. 현재 Host CLI/helper/service·보호된 등록·runtime을 교체하지 않았다.

## 준비 검증과 보존

Read-only Git objects를 사용하는 깨끗한 private checkout에서 pinned Go1.26.9로
CGO_ENABLED=0/trimpath/buildvcs 빌드했다. 실제 build metadata는 source918e329와
vcs.modified=false다. 일반 help/self-update help/config language와 실제100×30 PTY의
시작 화면·명령 목록 End 이동·언어 화면·q 종료가 통과했다. Exact termios/cursor/
화면 복원도 확인했다. 첫 operator는 목록을 스크롤하지 않고 self-update가 보일
것으로 기대해 timeout됐다. 원본을 보존하고 실제 End 이동을 추가한 unchanged
binary의 후속 결과를 분리한다. 제품 source는 바꾸지 않았다.

원본: `docs/github-logs/owner-cli-review-20261021/`의 build/help/update-help/language,
first-pty-expectation-failure와 smoke2-*; 정확한 bytes/digest/build info는
smoke2-preview-identity.json이 소유한다. Build의 공유 read-only module stat-cache
쓰기 경고는 exit0와 분리해 보존했다.

빌드/검사 source·cache/operator는 소비 뒤 삭제한다. 사용자가 아직 확인하지 않은
preview binary·작은 설정 경로만 보존하고, owner 확인 종료 후 그 디렉터리도 삭제한다.
Preview를 배포 성과나 영구 Host migration으로 보고하지 않는다.

## Owner 피드백 보완 계획 — source 수정 전 검토

기준 a20edc1. 기존 launcher.draw/items, 정적 launcherExamples, localizeHelp와
Doctor를 재사용한다. 기존 launcher와 여섯 생태계 업데이트는
IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES/ACCEPTANCE_CLOSED=YES다. 이번 추가
시각 강조·명령 발견성 요구만 IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO다. MISSING은 아래 변경과 해당 CLI 회귀·preview 확인이다.

1. internal/cli/launcher.go: home 화면의 width>=60/height>=22에서만 고정 다섯 줄
   HELOX wordmark를 표시한다. 풀네임을 유지하고 기존 paint/NO_COLOR/폭·높이
   clamp를 사용한다. 작은 화면과 사용법·설정·상세 화면은 기존 compact header다.
   같은 파일의 명령 목록에서 pip install의 --upgrade/-U, github install의
   --replace를 표시해 이미 있는 업데이트 예시를 발견할 수 있게 한다.
2. internal/cli/presentation_catalog.go: 영어·한국어 pip 설치 설명에 --upgrade,
   GitHub 설치 설명에 Release asset과 --replace를 명시한다. 번역 fallback과
   실제 명령 parser/검역·설치 owner는 유지한다. npx/pip update/git clone을 새로
   지원한다고 표시하거나 실행 경로를 추가하지 않는다.
3. 기존 launcher_test.go의 일곱 언어/폭·높이/color 경계 검사에 banner 전환 직전과
   직후 치수를 포함한다. Canonical format→Quick(빌드/vet/Staticcheck/default tests),
   security/docs와 실제 PTY 회귀로 검증한 뒤 동일 preview 경로를 갱신한다.
   완료 source/cache/operator는 제거하고 진행 중인 owner preview만 보존한다.

두 번째 검토: command label은 표시용이며 index→기존 Cobra command 대응을 바꾸지
않는다. wordmark는 화면 제어/입력/프로세스를 추가하지 않고 충분한 home body 공간을
남긴다. 기존 스크롤·End 이동·원상 복원·직접 명령·JSON 경로를 보존한다. 의존성,
Root 설치·서비스 설정, update contract, CI gate 변경이 필요하지 않다.

실제 진단 원본 feedback-diagnosis.json에서 preview/설치본 모두 release/network
UNAVAILABLE이고, 서비스는 inactive/dead/MainPID0, 설정된 socket은 ENOENT였다.
따라서 network 경고를 preview 신원 거부로 단정하지 않는다. 현재 Host 서비스는
이번 화면 보완에서 변경하지 않는다. pip는 install --upgrade/-U, GitHub는 Release
asset install --replace이며 기존 예시가 해당 구문을 이미 포함한다. npx는 미지원이다.

### 보완 소스 검사 결과

Canonical format10.09s, Quick193.20s, security7.85s, docs0.21s가 exit0였다.
Quick은 production/test build, vet, pinned Staticcheck, 기존 실제19 PTY와 기본
테스트를 포함한다. 일곱 언어의60×22 banner 경계·작은 화면·NO_COLOR를 검사했다.
원본 feedback-{format,quick,security,docs}.{json,log}를 보존한다.
최종 후보 전체 tree1ad5141이 private/root index와 동일함을 대조해8d949a7로
커밋했다. 최종 문서 후보 docs0.23s/security4.59s도 exit0였다. 새 구현은
IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES(local checks)/ACCEPTANCE_CLOSED=NO다.
남은 owner 화면 검토·새 source 원격 CI와 공개 설치는 이 검사 성공으로 대체하지
않는다. 기존 native self-update qualification을 재실행할 이유는 없다.

갱신 preview는 pinned Go1.26.9/CGO_ENABLED0/trimpath/buildvcs로 빌드했고
vcs.revision=8d949a7fd74ab920df1913771bf97b9f9e42fb35/modified=false다.
SHA256=875ca049b6747ac02f22921628d3d61dc8a35fe7e8527c8166f30b751503d423,
42,456,373B. 실제100×30 PTY의 큰 로고·풀네임, 사용법의 두 옵션·End 스크롤,
59×21→20×8→100×30 전환과 q exit0/exact termios/cursor/화면 복원을 확인한 뒤
동일 경로의 실행 파일을 원자적으로 갱신했다. pip/GitHub 도움말·언어 조회도
exit0였다. 원본 feedback-build/preview/preview-attempt/host-cli-unchanged와
operator 입력을 보존한다. Host CLI SHAeeb11993은 불변이다.

이후 문서 전달 커밋은 세 Go 파일을 변경하지 않는다. 전용 source/cache/operator는
마지막 문서 검사·커밋 뒤 삭제하며 feedback-cleanup.json이 실제 삭제를 소유한다.
아직 진행 중인 owner review용 binary·설정만 보존한다.

## 다음 순서

Owner CLI 확인/피드백 → 필요한 보완 및 채널 준비·qualification → 기존 PR38의
보호된 main merge → main exact source CI → 승인된 최종 version/source의 공식
서명 build·publication 검증 순서다. 첫 stable 목표v1.0.0은
[canonical release boundary](../../18-m13-production-release-operations-contract.md#4-release-activation-boundary)를
따른다. 기존 rc ref를 바꾸거나 지금 새 release tag를 생성하지 않는다.

## Full-name brand — source 수정 전 baseline / 두 번째 검토

기준72a9076. 기존 launcher draw/input/viewport/paint와 이전 local qualification은
유지한다. 새 요구인 풀네임 대형 표시·배경/다중 accent만 IMPLEMENTED=NO,
WIRED=NO, QUALIFIED=NO, ACCEPTANCE_CLOSED=NO다. 기존 owner 대체는 없다.

수정 위치: launcher.go draw의 header 구성과 선택 행 paint, 같은 cli package의 작은
private launcher_brand.go 표시 helper, launcher_test.go의 화면 경계/color 회귀.
외부 font/TUI dependency나 실행 경로를 추가하지 않는다. 고정 제품명 전용 3×5 glyph를
사용하고, 충분한 홈 화면에서 세 단어를 같은 크기로 그린다. 112열 이상은 한 줄,
68열 이상은 Heliopause / Artifact Airlock 두 줄, 44열 이상·28행 이상은 세 줄로 배치한다.
그 외에는 읽을 수 있는 fullname과 작은 helox label을 사용한다. 각 layout은 최소 메뉴
네 줄과 footer를 남긴다. Navy background/cyan·white·gold accents는 기존 paint 정책을
사용하며 NO_COLOR/never에서는 모든 SGR을 제거한다. Explicit always override는 유지한다.

두 번째 검토: 장식은 홈에만 확장하고 settings/help/detail은 compact다. 제품명 plaintext를
함께 두어 검색·복사도 가능하게 한다. Physical last column 미사용, line별 SGR reset,
Unicode cell 폭, all-language body/footer, 작은 resize/cancellation/termios 복원과 실제
preview를 검사한다. 기존 helox 단독 명령·doctor/실제 update·source/runtime를 바꾸지 않는다.
검사는 canonical format/quick/security/docs 및 실제 binary PTY로 수행한다. 완료한
private checkout/build/cache는 삭제하고 사용자 확인 중인 preview만 유지한다.

### Full-name 보완 로컬 검증 / 현재 preview

source1f2500e/treeef8eeb8, clean Go1.26.9 build(vcs.modified=false). 새 표시 요구는
IMPLEMENTED=YES / WIRED=YES, 로컬 검사 PASS다. 전체 QUALIFIED/ACCEPTANCE_CLOSED는
새 native macOS/원격 CI 및 owner 확인 전까지 NO로 둔다. 기존 기능 qualification은 유지한다.
[영수증](../package-client-quarantine/design-and-brand-qualification-20261021.json)에 정확한
binary identity와 검사 결과가 있다. Canonical format11.88s/Quick224.14s/security7.24s/
docs0.21s, 기본 native PTY와 신규 색상/경계 회귀를 포함해 통과했다. 실제 clean binary도
한국어 홈/사용법/End 이동, 여섯 geometry, NO_COLOR와 강제 color panel, q 종료와
termios/cursor/alternate-screen 복원을 통과했다. 그림과 글자가 실제 화면에 모두 표시되는지
캡처를 확인했다. Host CLI SHA는 기존eeb11993으로 유지됐다.

실행 경로는 위와 같은 `/tmp/helox-cli-review-20261021/helox --lang ko`다.
완료된 전용 source/cache/operator 정리는 local raw fullname-cleanup.json을 따른다.
진행 중인 owner preview 약42.5MB와 작은 증거만 보존한다. 신규 client 지원은 설계 단계이며
현재 메뉴에 실행할 수 없는 명령을 추가하지 않았다. 등록·서비스·main·공개 배포는 별도다.
