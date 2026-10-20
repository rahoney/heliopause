# M13-004 — rc3 기준과 보완된 서명 후보 하나의 전체 전환 시험

상태: **OWNER_APPROVED / CI357 실제 Required+9 SUCCESS**. 사용자는 정확한 source의
rc5 단일 태그 생성과 이 계획의 시험 진행을 승인한 뒤 “이어가”로 재개를 요청했다.
기존 rc3/rc4의 승인 source4ff6c89와 두 immutable ref는 유지한다. Public Release·main
merge·Host 영구 migration·배포 채널 이름 결정은 포함하지 않는다.

| 입력 | 고정값 |
| --- | --- |
| 현재 base | v0.1.1-rc.3, source4ff6c89234698d6fe0b6b5366f7605538e47b967 |
| 승인 target | v0.1.1-rc.5, 원격 재확인 후 정확한 source에 생성 완료 |
| target source | 7fedaf3430acd4c8a9f37ceaffec33bce143b2ae |
| qualification | [CI357](https://github.com/rahoney/heliopause/actions/runs/39448071670)의 실제 Required+9 SUCCESS 확인 뒤 생성 가능 |
| 현재 scope | Linux/WSL2 amd64, 같은 runtime lock·기존 SDK·검역·Policy·Root protocol |

Source는 위 commit에 고정한다. 이 문서나 후속 CI 기록 commit을 묵시적으로 target
source로 바꾸지 않는다. CI 실패·다른 source가 필요하면 이 입력의 qualification을
재판정한다. 기존 보호 규칙·태그를 재지정하거나 unsigned 설치로 검증을 대체하지 않는다.

## Baseline과 재사용 근거

[실제 receipt](./signed-native-20261020/retest-qualification.json)와
[보완 계획/검토](./protected-health-cli-plan-20261021.md)를 재사용한다.

- Actual signed rc3 base의 SDK/first files/register/systemd/doctor/npm와 cold start는 PASS다.
- rc3의 observer로 actual signed rc4 selected7의 required18/ALLOW를 얻었다. Root 전환은
  FAILED/ROLLED_BACK이며 기존 버전은 정상이었다. 이 실패를 정상 완료로 바꾸지 않는다.
- 보완은 후보 CLI의 full health callback에만 있다. 기존 rc3 installer는 후보의
  `current.ClientPath()`를 실행하고 rollback은 full=false다. 따라서 base를 다시 서명·
  설치할 필요가 없는 변경 경계이며 새 target 하나의 실제 소비로 확인한다.
- Source7fedaf3의 RED4→GREEN semantic7, format/Quick/security/vulnerability/docs·관련
  Race와 전체 후보 Git tree/mode 대조는 PASS다. 실제 whole 성공·OS 인증 성공·추가
  fault/recovery는 MISSING이다. 기존 구현을 재작성하지 않는다.

## 실행 순서와 acceptance

1. Owner의 이 정확한 입력 승인과 CI357의 실제 열 개 SUCCESS, branch/commit·새 ref 부재·
   creation/immutable rules를 확인한다. Target 하나만 생성해 official tag-push build의
   source/ref/run과 exact13을 고정한다. 고정 independent verifier/trust root로 실제
   subject13의 offline 서명과 manifest/SBOM/runtime 결합을 확인한다.
2. 필요한 qualified rc3 전용 환경을 시작하고 같은 runtime·서비스·ordinary doctor/npm을
   확인한다. Actual public Release가 없는 경우 전용 HTTPS transport만 사용한다. 실제
   signature/root/source/tag/required checks/Evidence/Policy와 production caller는 유지한다.
3. rc3 production `self-update --version v0.1.1-rc.5`의 actual required18/ALLOW→동일
   보호 사본→전환→후보의 full health→COMMITTED/정리와 새 ordinary doctor/npm을 확인한다.
   검사 성공·helper exit0·ROLLBACK 성공만으로 업데이트 완료를 선언하지 않는다.
4. 각 fault case는 owned rc3 기준 사본을 순차 복원해 시작하고 일반 doctor를 재확인한다.
   같은 inner Docker volume을 동시에 쓰지 않는다. 다음을 서로 구분해 기록한다.
   - OS 관리자 인증 없음/실패는 Root 전환 전 거부, 실제 OS 인증 성공은 같은 터미널의
     sudo 인증 후 ordinary CLI 경로로 확인한다. 비밀번호를 제품 config/report에 넣거나
     전체 CLI를 Root로 실행하지 않는다.
   - 취소·동시 갱신은 단일 owner/잠금과 기존 버전 보존을 확인한다.
   - 실제 candidate service 실패는 신원을 확인한 시험 process에만 주입하고 자동 복원과
     old doctor/npm을 확인한다. 정상 경로의 원래 실패와 주입된 실패를 혼동하지 않는다.
   - 실제 Root process 중단은 durable journal/보호 사본·상태를 보존한다. 확인된 종료와
     독립적인 resource query 뒤 installed `recover`의 복원·journal 정리와 재실행을 확인한다.
     UNKNOWN 아래 강제 restore·stage glob 삭제를 하지 않는다.
   - Guest service/PID1 cold start와 Host kernel/WSL reboot의 범위는 분리한다. 실제로 하지
     않은 kernel reboot나 public Release API publication을 성공으로 보고하지 않는다.
5. 원본 결과·작은 public proof/manifest·필요 Evidence를 보존하고 완료된 download·binary·
   guest/image/volume·cache/operator는 마지막 소비와 참조 확인 뒤 삭제한다. 필요한 현재/
   직전 정상 버전과 복구 journal은 중간 정리 대상으로 삼지 않는다.

두 번째 대조: 새 tag 하나가 필요한 이유는 immutable rc3/rc4에 없는 교정 bytes를 official
신뢰 체인으로 소비하기 위해서다. 기존 qualified base/Root role/관찰/Policy를 재사용하며
새 runtime·privileged protocol·검역 예외를 도입하지 않는다. 원격 CI와 owner 승인이 끝난
정확한 source만 서명 입력으로 사용한다. M13-004만 IN_PROGRESS다.

CI357의 실제 Required 포함10개가 모두 SUCCESS다. 실제 checkout6d65dfc의 전체
tree/mode가 고정 source7fedaf3와 같고 native macOS/C++103/Linux45·CPU226.62s 및
helper5 wait0/final confirmed cleanup을 확인했다. [원격 receipt](./signed-native-20261020/health-cli-remote-ci.json)를
따른다. API 재확인에서 rc5는 부재이며 rc3/rc4는4ff6c89, 두 rules는 active, PR38은
draft7fedaf3다. 사용자 승인 뒤 동일 조건을 다시 확인하고 rc5 하나만 생성했다.
[Official build39456168381](https://github.com/rahoney/heliopause/actions/runs/39456168381)의
실제 서명 결과와 전체 전환은 [rc5 receipt](./signed-native-20261020/rc5-qualification.json)와
[통합 qualification](./signed-native-qualification-20261020.md#rc5-실제-정상-전환os-인증장애-복구와-취소-gap)에 기록했다.
정상/OS 인증/동시 실행 거부/서비스 rollback/Root SIGKILL·guest cold recovery·재실행은
통과했고, 취소 cleanup gap은 별도 보완 중이다. Qualified rc3 기준 환경은 순차
재생성했고 전용 private cgroup controller 준비 뒤 ordinary doctor5OK/npm required3·
ALLOW를 재확인했다. 최초 setup 실패와 원본 readback은 rc5-base-*에 보존한다.
