# M13-004 — frozen release 획득 연결부의 로컬 검증

기준 parent `148368f39a22ac7a6b18a61b071254f6ed243000`.
[구현 전 감사·수정 위치·두 번째 검토](./acquisition-plan-20261020.md)의 실제 gap만
구현했다. 전체 M13-004는 계속 IN_PROGRESS이며 본체 CLI·protected activation은 미완료다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| 단일 응답에 release asset identity 고정 | YES | YES, adapter API/tests | YES, local | YES, 이 연결부만 |
| bounded public attestation 획득·decode | YES | YES, snapshot API/SDK test | YES, local/public read | YES, 이 연결부만 |
| 원본 FD의 exact bytes를 기존 검사 Run으로 복사 | YES | YES, 기존 InspectService/Evidence/Policy 회귀 | YES, local | YES, 이 연결부만 |
| 본체 metadata/native set 검역 orchestration | NO | NO | NO | NO |
| 보호된 CLI/helpers 교체·복원·다음 실제 검역 | NO | NO | NO | NO |

`internal/artifact/githubrelease`의 기존 Resolve/Acquire·parser를 유지하고
`FreezeRelease`, `ReleaseSnapshot.Attestations`, `NewFrozenIntake`를 추가했다.
snapshot lookup은 HTTP를 다시 호출하지 않으며 FD replay에는 network client가 없다.
attestation의 HTTP200·repository_id·immutable 속성은 설치 허가가 아니다.
공식 root를 사용하는 기존 provenance verifier와 실제 검사·Policy가 계속 필요하다.
기존 compression pin은 유지하고 새 Snappy consumer의 direct 표기만 반영했다.

## 실행과 정확한 범위

원본은 local `docs/github-logs/self-update-20261020/`의 아래 자료가 소유한다.
공유 도구·module cache를 재사용한 Go1.26.9/Linux amd64 검증이며 cold-cache 시험이 아니다.

- `acquisition-checks.json`: focused1.83s, canonical Quick102.50s,
  security6.33s, vulnerability11.21s, docs0.21s, 관련 Race4.50s 모두 exit0.
- 뒤이어 concurrent Acquire/Close 회귀를 강화했다. 첫 성공 copy를 실제 확인한 뒤
  Close와 동시 요청을 경합시킨다. production source 변경은 없으며
  `acquisition-final-checks.json`의 focused1.86s/Race3.78s가 exit0다.
- macOS의 temp 경로 alias도 실제 경로로 정규화하도록 test fixture만 보완했다.
  `acquisition-portable-checks.json`의 최종 focused1.96s/Race4.09s와
  Darwin amd64/arm64 test compile2.43s/2.28s가 exit0다.
  이는 native macOS 실행 증거가 아니다. 완료한 cross-compile 출력59,387,202B는 삭제했다.
- `acquisition-public-attestation.log`: 명시 opt-in한 공개 API→Azure raw Snappy→
  production SDK 검증이 실제 PASS1.215s, bundle1개/valid1개다.
  subject는 committed v0.1.0 manifest의 exact digest다. 새 self-update release
  payload 획득·설치나 본체 전체 end-to-end 성공으로 확대하지 않는다.
- `acquisition-delivery-source.json`은 최종 source/input identity와 이전 full-check
  입력 대비 test-only 차이를 기록한다. 최종 문서 전달 검사와 commit proof는
  같은 원본 디렉터리의 acquisition-delivery 자료를 따른다.

실제 기존 InspectService 회귀는 archive fixture의 required verification/static
checks2개, 새 Run/Evidence2개와 ALLOW를 확인한다. native executable 동적 검역을
이 archive 성공으로 대신하지 않는다. path 교체는 held FD의 원래 bytes를 유지하고,
in-place 변조·symlink/hardlink·잘못된 ownership/mode·기존 목적지 덮어쓰기는 거부한다.
취소·Close·partial copy 정리와 remote URL/query 비출력도 확인했다.

첫 focused 실패는 `testing.TempDir` child가 umask에 따라0700이 아니었던 fixture
준비 오류다. 실제 product 경계를 낮추지 않고 시험이 소유한 root만 chmod0700과
EvalSymlinks로 준비했다. 최초 실패 원본은 `acquisition-focused-first.log`에 유지한다.

## 두 번째 검토와 다음 실제 gap

기존 adapter와 inspection owners를 대조해 중복 resolve·download fallback,
URL 입력 권한, 자기 서명 root, 새 Policy/검사 엔진이 없음을 확인했다.
판정은 source acquisition의 신원 보존과 기존 archive 검사 연결 범위에 한정한다.
남은 작업은 엄격한 release metadata와 native 전체 set을 기존 application/install
흐름으로 검역한 뒤 승인된 별도 설치 역할에 전달하는 연결, protected runtime의
원자적 활성화·실패 복원·재인증 검증이다. 해당 권한·crash recovery 세부 계획을
먼저 확정하고 source를 수정한다. 공식 채널 이름은 이 core 작업의 blocker가 아니다.
