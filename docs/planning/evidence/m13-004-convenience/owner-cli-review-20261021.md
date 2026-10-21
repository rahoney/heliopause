# M13-004 — 배포 전 owner CLI 확인

상태: **READY_FOR_OWNER_REVIEW**. M13-004만 IN_PROGRESS다. 기능 검증과 별도로
사용자가 최신 화면을 직접 확인하기로 했다. 기존 CLI-UX-003/native qualification을
재개방하거나 기능을 다시 구현하지 않는다.

최초 preview 기준: source918e329/tree38b2f3b, 실제 CI359 Required+9 SUCCESS.
아래 owner 피드백 보완은 별도 변경이며 갱신 preview의 정확한 source/digest는
빌드 뒤 local raw feedback-preview.json이 소유한다.
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
최종 소스/문서 대조와 갱신 preview 기동 확인을 이어간다. 새 구현은
IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES(local checks)/ACCEPTANCE_CLOSED=NO다.
남은 owner 화면 검토·새 source 원격 CI와 공개 설치는 이 검사 성공으로 대체하지
않는다. 기존 native self-update qualification을 재실행할 이유는 없다.

## 다음 순서

Owner CLI 확인/피드백 → 필요한 보완 및 채널 준비·qualification → 기존 PR38의
보호된 main merge → main exact source CI → 승인된 최종 version/source의 공식
서명 build·publication 검증 순서다. 첫 stable 목표v1.0.0은
[canonical release boundary](../../18-m13-production-release-operations-contract.md#4-release-activation-boundary)를
따른다. 기존 rc ref를 바꾸거나 지금 새 release tag를 생성하지 않는다.
