# macOS default-test deadline — 진행 구간 진단

M13-004/P1c의 선행 qualification 조사이며 다른 work item을 시작하지 않는다.
e25fd97의 CI368 macOS119299522014는 build60.71s PASS 뒤 default test6m에서
context deadline exceeded였다. 270c7c8/manual367도 같은 단계에서 FAIL이고 같은 source의
CI366 native는 PASS다. c2c1750/cu130의 build2m timeout과 분리한다. SDK 회귀·UV source·
특정 package/test의 원인은 아직 미확정이다. 원본 ci368-macos-failed.log가 해당 실행을 소유한다.

| 기존 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| scripts/check/runner.go default test·command timeout·bounded output | YES | YES | YES(기존 scope) | YES(기존 scope) | deadline 시 실제 active package/test 진단 |
| Go native default-test failure 원인 관측 | NO | NO | NO | NO | 같은 한도에서 streaming trusted Go metadata와 최초 실패 보존 |

## 정확한 수정 위치·방법

1. platformSteps/quickSteps의 같은 default test만 go test -json으로 호출한다. package
   timeout5m·command deadline6m·실행 범위./...·offline/pinned tool·Required는 그대로다.
   기존 Go plain output은 package completion 순서로 buffer될 수 있으므로 마지막 출력이
   마지막 실행 test라고 단정하지 않는다. JSON은 진단용이며 exit status를 대체하지 않는다.
2. scripts/check에 bounded Go-test capture를 추가한다. Stdout event를 스트리밍으로 읽어
   root test 함수·package start/finish·pause/continue와 local elapsed만 기록한다. Top-level
   test/ASCII bounded package 식별자만 사용하며 subtest 입력·stdout 내용·환경/secret을
   active metadata로 쓰지 않는다. 동시 stdout/stderr는 같은 한도 안에서 직렬화한다.
3. 기존 command runner의 absolute path·ctx·environment·Run·first error/timeout 분류는
   공통 실행 함수로 그대로 재사용한다. 기존 plain command capture와 새 Go capture가
   그 두 소비자다. Raw output1MiB 한도를 유지하고 positive verbose progress만 압축한다.
   최대64KiB event line·128 active metadata·실패 시16 entries의 bounded summary를 사용한다.
   Malformed/overflow는 진단 unavailable/truncated이며 command 성공/실패를 바꾸지 않는다.
4. 첫 fail event와 기존 실제 error output/cause를 보존한다. Metadata pass나 cached event로
   검사를 성공 처리하지 않는다. 진단 write/parse가 child EOF를 막거나 새 goroutine/daemon/
   external service를 만들지 않는다. Scope 밖의 command들은 기존 capture를 사용한다.
5. Partial chunk·malformed/oversize·parallel active tests·pause/complete·bounded diagnostic과
   실제 child timeout/first-error 회귀를 검사한다. Canonical format/Quick/security/docs/workflow,
   관련 Race와 native platform TEST compilation을 연결한다. 이후 실제 native CI의 active
   구간 관측을 확인하며 원인 없는 timeout 확대·무작정 retry는 하지 않는다.

## 두 번째 검토

이 변경은 timeout 해결책이 아닌 최초 구간 관측이다. 기존 runner를 다른 executor로
대체하지 않는다. SDK/toolchain·정책·test selection/필수 coverage·권한·resource budget은
바꾸지 않는다. Streaming event가 authority가 되지 않도록 기존 process status/cause가
최종 결과를 소유한다. CI368의 진행 중 Linux 결과는 보존하고 새 push로 취소하지 않는다.

후속 확인에서 CI368은 종료됐고 Linux integration·UV startup3.25s·CPU259.85s는
PASS, macOS와 Required는 FAIL이다. 해당 실행을 재실행하지 않고 새 진단 후보를 검증한다.
Oversize event는 bounded raw prefix를 보존하며 metadata 수집만 unavailable로 표시한다.
Offline cache 분류는 Go tool stderr/build-output의 GOPROXY=off만 사용한다. 일반 test
Output의 같은 fixture 문자열로 다른 실패를 prerequisite 오류로 바꾸지 않는다. 실제 고정
SDK의 offline missing-module probe에서 build-output 형식을 확인했고 fixture는 삭제했다.
