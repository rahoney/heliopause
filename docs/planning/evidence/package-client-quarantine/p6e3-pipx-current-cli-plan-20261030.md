# P6-E3 — pipx current 실행·CLI baseline

Baseline2dcc1ad는 CI437/f4의 Required10SUCCESS를 기록하며 E2-B/eae6b8c의 pipx
source/native selector는 named local qualified다. 현재 item은 M13-004 하나다.
CI437 GET worker는 실제5회 조회 뒤 완료를 확인해 SIGTERM/exit143으로 종료했다.
다음 타이머는 없다. E2 source는 CI437에 없고 새 CI는 완성된 current 변경 묶음에서만 검토한다.

Read: PROJECT/Queue/docs router, lessons INDEX/artifact L01/L02/sandbox L03/L08/L12/L18/
evidence L16/ci L06, client09§1·기존 execution 계약, plan20§4–6, E2 baseline/receipts와
D6 current application/D7 CLI 지도·CI437을 대조했다. 실행 코드 전 아래 owner를 확인했다.

| 책임 / 기존 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| ConsoleExecutionInventory/SelectConsoleExecutionEntry·원본 context | YES | YES | 기존 named YES | 기존 범위 YES | pipx primary/sole-app 선택 adapter; 공용 기본 조건은 유지 |
| E2 pipx whole original/native binding·lifecycle/typed selector | YES | YES | named local YES | named source/native YES | default/sole-app를 같은 원본으로 선택; current 소비자 |
| Current whole integrity/inspection/M3/M5/Evidence/runtime | YES | YES (기존 Python/UV) | CI437 named YES | 기존 named YES | 실제 pipx selector/고유 identity를 같은 current 경로에 연결 |
| Descriptor/preflight/literal argv·report/typed exit/privacy | YES | YES (기존 npx/UV) | 기존 named YES | 기존 named YES | pipx run leaf·--spec·같은 parser/report 소비 |
| Current pipx run/app/CLI | NO | NO | NO | NO | 아래 concrete 연결·정상/guest nonzero/거부 actual |
| 전체 P6 aliases/hostile/profile matrix/P7/final | NO | NO | NO | NO | 후속 acceptance; 이 단계의 성공으로 닫지 않음 |

## 기존 구현 재사용과 현재 변경 범위

- 기존 generic SelectConsoleExecutionEntry의 기본 조건을 넓히지 않는다. Pipx 전용
  selector가 먼저 요청한 primary 이름을 찾고, primary/default 요청에만 단일 console
  fallback을 적용해 기존 canonical selector로 반환한다. 임의 missing explicit command와
  ambiguous/missing/unsupported entry는 거부한다. Native와 current approved 원본에서
  같은 함수를 소비하며 native metadata를 Policy나 ownership으로 사용하지 않는다.
- E2 original freeze/whole collision guard/native17argv/cache census/binder를 재사용한다.
  Default 선택에 필요한 원본 준비의 command gate만 기존 UV wrapper에 유지하고 shared
  original body를 두 실제 소비자가 사용한다. Fixed inert query·startup gate·no network
  replay·source/command binding·native identity/불완전 결과 비우기는 보존한다.
- D6 기존 factory의 current integrity/inspection/Evidence/M3/M5/records/staging/runtime
  조립을 기존 application.ExecutionResolver/ExecutionEntryBinder ports로 공유한다.
  Pipx의 thin concrete resolver/binder만 연결한다. Current binder는 같은 승인 원본에서
  console을 다시 측정하며 새 Core/state/daemon/receipt/권한·검사 생략은 추가하지 않는다.
- D7의 Python request parser를 selection flag(--from/--spec) 차이만 공유한다. 기존
  report/literal argv/private streams/ASCII opt-in/typed exit·cleanup 축은 그대로 사용한다.
  Pipx run descriptor·실제 bootstrap router/factory를 같은 묶음에서 소비하게 해
  tagged-only helper나 미래 placeholder를 남기지 않는다. 기존 install/upgrade는 유지한다.

## 검증과 완료 경계

Cheap source/default/explicit/ambiguous·raw options/literal argv·command mismatch/cancel
검사를 먼저 모은다. 완성된 source에 canonical format/Quick와 영향 package Race,
tagged vet/pinned Staticcheck를 실행한다. 실제 current whole approval/normal0/typed guest2/
불일치 거부와 pipx run/--spec/다른 package-console/sole-app/없는 console 거부를 확인한다.
공통 body 변경은 기존 UV 소비자와 비교한다. CPP·고정 입력·이미 검증한 native만의
실행은 source/입력 변화나 새 우려 없이는 반복하지 않는다. Whole run 및 후속 hosted의
예상 시간을 현재 CI437111m23s와 실제 새 cohort 시간에 대조한 뒤 CI 경계를 정한다.


## 실제 구현 후 첫 저비용 검사

기존 pipx 지원 fixture가 이미 native pipx resolver를 반환하므로 이를 그대로 소비했고
새 fixture/lifecycle을 만들지 않았다. Current application/CLI actual 본문도 기존 UV와
공유한다. Source22개 Go 경로의 후보이며 generic console/Core/application ports/Policy/
CPP는 불변이다. 첫4package focused23.727s와 tagged vet는 PASS다. 첫 Quick310.522s는
old pipx-tool test의 run-is-unimplemented 기대값 한 곳에서 실패했으며 모든49package는
terminal이었다. 신규 run positive와 unsafe/native 옵션 거부는 유지하고 그 유효한 한 항목만
미지원 목록에서 뺐다. Failed focused2.872s 뒤 canonical Quick42.864s가 PASS이며 나머지
불변 package는 Go test cache를 재사용했다. Tagged vet/Staticcheck와 영향5package Race86.008s도 PASS다. Actual 후속은 아래를 따른다.


## 실제 실패와 최소 후속

최초 operator는0644 shell 직접 실행126으로 product 미도달이었다. Bash로 바꾼 두 번째는
새 workflow mode의 native fallback 제외 누락으로 empty-filter/PASS였으므로 앱 성공이
아니다. 이를 checker의 exact dispatch와 함께 고쳤다. 같은 실행의 CLI1–5는 PASS 뒤
기본 sole console6에서 FAIL했다. Subtest6만 재현한43.88s는 whole 검역/ALLOW 뒤 준비
INCOMPLETE/runtime NOT_STARTED다. Primary 이름을 explicit command로 넘겨 기존 application
일치 경계가 거부함을 source와 대조했다. Pipx adapter에서 기존 optional default 의미로만
전달한 green53.16s는 새 graph2/ALLOW/원본 slugify 정상0을 확인했다. Core/generic selector와
명시적인 다른 command의 일치 경계는 유지한다. Red operator의 마지막 summary 생성
KeyError는 product FAIL·이미 기록한 Host 원상 복원을 지우지 않으며 다시 실행하지 않는다.

최종 source 묶음의 canonical format1.417s/workflow1.819s/Quick81.012s,
checker 전체 Race73.525s/변경 bootstrap Race15.959s 및 tagged vet/Staticcheck는 PASS다.
App actual의 runtime identity/기존 command 불일치 거부는 통과했지만 emitter selector 문자열이
UV로 고정돼 있어 runtimeIdentity에서 출력하도록 test-only 한 줄을 고쳤다. 원본 로그는
바꾸지 않으며 이 출력 수정만으로 native·CPP·Quick/기존 actual을 다시 실행하지 않는다.
기존 acquired42/materialized ZIP/private installed fixture를 한 번 공유하는 실제 앱3/CLI6와
기존 UV CLI5의 최종 canonical recipe는 진행 중이며 완료 전 전체 성공으로 쓰지 않는다.


CLI failure 선택을 위해 subtest로 옮긴 테스트에서 기존 manifest 이름을 건너뛰던 continue를
return으로 바꾼 oracle 오류를 발견해 원래 continue로 복원했다. 이 때문에 누락된 marker와
aggregate 실패를 제품 실패나 성공으로 간주하지 않는다. Product/default source가 불변인
최종 tagged vet3.223s/Staticcheck3.372s·ELF build3.982s만 추가했다. 이미 통과한 current 앱3와
CPP/Quick/Race는 다시 실행하지 않고 영향을 받은 CLI와 아직 미도달인 기존 UV CLI를
같은 product source에서 검증한다. Raw p6-e3-current-final이 오류를 포함한 최초를 소유한다.


## Named local 완료

[한정 결과](./p6e3-pipx-current-cli-local-20261030.json)는 같은 product source의 실제 앱3
143.10s와 최종 pipx CLI6/283.94s·기존 UV CLI5/217.40s를 연결한다. 최종 operator503.702s는
exit0/noSKIP/helperwait0·보호5/Docker 보존·같은 inode의 profile 원상 복원을 확인했다.
Pipx의 새 operation records5/subject runs7과 source markers7도 확인했다. Actual 저장 로그의
CLI6/UV5 guard DATA는 PASS다. App label의 수정 DATA를 actual replay로 쓰지 않는다.
Current/CLI named local slice만 닫았고 새 source hosted·남은 aliases/일반 동등성·whole final은
열어 둔다. 작은 helper별 push 없이 E2/E3 전체 source/current/CLI 묶음을 한 번 검증한다.
