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

## CI369 관측 뒤의 좁은 진단 보완

Source0fa2f85의 native run39631060147/job119329030437는 build77.93s PASS 뒤 default
test6m에서 FAIL이다. JSON summary는 started47/finished45·unavailable=false/truncated=false다.
Promotion package는 elapsed252138ms에 active root test가 없었고 출력에는 ok214.013s가
있었다. Update-runtime은88248ms, 마지막 root test는2256ms였다. 이 마지막 test를
6분 지연의 원인으로 단정하지 않는다. SDK test.go는 ok 출력 뒤 cache save를 수행한 후
deferred package event를 종료하므로 package-active와 test-body-active도 구분한다.
구체 native syscall/SDK 회귀·cache 지연은 아직 미확정이다. 원본 ci369-macos.log를 보존한다.

기존 positive verbose filtering은 완료된 root test 시간을 버렸다. go_test_capture.go의
같은 bounded activity owner에 local monotonic elapsed 기준1초 이상 완료 root 함수의
상위8개만 추가한다. 이미 검증하는 bounded source package/root 이름과 fixed status만
사용하고 dynamic subtest/Output/환경은 요약 metadata에 사용하지 않는다. 기존 raw1MiB·
active128/표시16·process cause·5m/6m·test selection은 유지한다. 완료 시간이 길어도
실패 authority로 쓰지 않는다. Top8 bound·동률 정렬·짧은 함수 제외·실패 보존을 검사한 뒤
canonical5/Race/Darwin compile로 확인한다. CI369 Linux를 새 push로 취소하지 않는다.

Production build의 trimpath와 default-test compilation의 flag 차이도 source에 있으나
원래 native timeout 인과는 미확정이다. runtime.Caller로 source를 찾는 sandbox tests가
있으므로 default test에 trimpath를 즉흥적으로 추가하지 않는다. 준비 step/한도를 늘리거나
테스트를 제외하는 대신 실제 hot root 구간부터 관측한다.

## CI371 완료 이벤트와 명령 deadline의 구분

Source3623d1e/native job119573087324는 default6m 종료에서47 started/47 finished·active0,
불가/누락0이고 개별 package는 모두 pass다. 그래도 실제 command deadline은 FAIL이며
이벤트로 성공을 만들지 않는다. Root test summary의 local elapsed는 pause·scheduler·
출력 전달 시간을 포함하므로 단일 test body/CPU 시간으로 해석하지 않는다. CI370의
같은 tree/native 성공은 별도 영수증이며 source371의 실패 원인은 아직 미확정이다.

Baseline owner: runCapturedCommand의 actual Cmd.Run/context/first cause와 Go JSON 수집은
IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES(진단 범위) / ACCEPTANCE_CLOSED=NO(원인).
명령 종료 상태·전체 package 완료 뒤 gap 진단은 IMPLEMENTED=NO / WIRED=NO /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다. MISSING은 다음 두 bounded scalar 관측이다.

1. runner.go의 기존 timeout 오류에 Cmd.ProcessState의 가용 여부, exited/success/exit-code와
   user/system milliseconds를 붙인다. Cmd.Run/cause/deadline/출력 한도/실행 옵션은 유지한다.
   프로세스가 아직 실행 중이어서 kill됐는지, exit0 뒤 pipe drain 등에서 deadline을 만났는지
   구분할 자료이며 exit0도 context failure를 덮지 않는다. PID·argv·새 OS/ps 조회는 추가하지 않는다.
2. go_test_capture.go의 같은 bounded collector에서 마지막 package terminal의 monotonic
   시각만 보존한다. started=finished>0/active0일 때 요약 시점과의 gap을 표시한다. Package
   이벤트는 계속 진단이며 완료 authority가 아니다. 새 이름/개별 로그·비밀을 수집하지 않는다.

기존 actual timeout/cause 시험의 terminal state assertion과 fake-clock completion-gap/
still-active 회귀, canonical5·checker Race·Darwin compile을 확인한다. 새 native CI 결과가
나오기 전 cache·SDK·pipe 원인을 단정하거나 timeout/준비 단계를 늘리지 않는다.
두 번째 검토: 관찰 두 scalar를 더하는 좁은 변경이며 기존 executor 교체, retry·새 daemon,
임의 프로세스 종료·package skip과 성공 판정 변경은 없다. Source/state 원본은
ci371-macos-failed.log와 ci371-macos-progress-facts.json이 소유한다.

Local implementation의 canonical5·checker Race15.425s와 Darwin test compile1.776s는
PASS다. [종료 상태/완료 gap 로컬 영수증](./macos-terminal-local-20261022.json)이 정확한
source와 범위를 소유한다. 실제 macOS 원인 관측은 새 source의 native CI에서 확인한다.

[CI372](./p1c-query-ci372-20261022.json)의 실제 첫 실패는 production build2m이며 default
test는 도달하지 않았다. Terminal available=true/exited=false/exit-1·user230765ms/
system108423ms가 관측됐다. 높은 CPU 사용은 그 build 명령의 관찰이며 이전 default
6m 실패의 인과나 cache/SDK 세부 원인을 확정하지 않는다. Bootstrap default cache와
checker child의 전용 cache, build trimpath/test 기본 flags가 다른 것은 source에서
확인했지만 실제 native rebuild 비용은 아직 분리 관측하지 않았다. Timeout·coverage/
prep 확대나 같은 입력 blind rerun은 하지 않는다.

## CI373 뒤 compilation cache 입력 정렬 — 구현 전 감사

Source9b0da57/CI373의 native macOS는 build94.05s PASS 뒤 default6m에서 실패했다.
47started/46finished·promotion package active214183ms/active root 없음과 exited=false/
exit-1·user559932/system356096ms를 보존한다. 마지막 package나 CPU 합계를 정확한 과거
정지 원인으로 확정하지 않는다. Raw ci373-macos-failed.log가 관측을 소유한다.

Baseline audit: production build/기본 시험 IMPLEMENTED=YES·WIRED=YES, 현 후보 전체
QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다. 기존 실행기·budget·scope는 재구현하지 않는다.
MISSING은 확인된 서로 다른 compilation 입력 정렬과 새 native qualification이다.

실제 setup log의 GOCACHE는 /Users/runner/Library/Caches/go-build인데 checker child는
HELOX_TOOL_CACHE/go-build를 사용한다. Build는 -trimpath, test-build/default test는 해당
flag 없이 실행한다. 고정 Go1.26.10 source cmd/go/internal/work/exec.go는 trimpath를
action hash에 넣고, 로컬 동일 stdlib errors의 두 go list -export 결과도 BuildID/export
digest가 달랐다. 원본 macos-cache-flag-mechanism.json은 cache-key 분리의 증거이며 cold
timing이나 macOS 전체 timeout 인과의 증명은 아니다.

수정 위치/방법:

1. runner.go의 test build validity/default test에 기존 production build와 같은 -trimpath를
   명시한다. test ./..., inner5m/outer6m, production2m와 오류/complete 판정은 유지한다.
2. boundary_helper_test.go와 collector_test.go의 두 runtime.Caller 기반 source lookup은
   다른 repository fixture처럼 go test의 package working directory에서 ../../tools/...를
   읽는다. 실제 검증 대상 observer source/비교값을 바꾸지 않는다. 검색한 repo Go source의
   runtime.Caller 사용은 이 두 곳이며 전체 canonical Quick로 다른 source lookup도 확인한다.
3. 기존 minimum-go/macOS의 bootstrap/platform step에 같은 GOCACHE를 HELOX_TOOL_CACHE의
   go-build 하위로 명시하고 outer go run도 -trimpath를 사용한다. GOFLAGS는 step에서
   명시적으로 비운다. runner.temp는 허용된 step env에서만 사용한다. Actions remote cache,
   setup-go cache, warm-up/test 생략·새 prep 단계·권한·job/Required 수는 바꾸지 않는다.
4. workflow validator는 실제 두 job의 조건 없는 bootstrap/platform step 명령·env를 YAML로
   확인한다. 한 job의 누락/잘못된 cache root·flag·조건부 실행은 다른 job/주석 문자열로
   대체할 수 없다. 기존 actionlint와 structural/Required policy도 유지한다.
5. canonical format/Quick/security/docs/workflow/platform과 관련 checker Race, Darwin test
   compile 뒤 새 native macOS/전체 Required를 확인한다. Cache 입력 정렬이 통과해도 과거
   모든 timeout이 해결됐다고 기록하지 않는다. 필요 없는 별도 cache·시험 바이너리는 삭제한다.

두 번째 검토: 같은 job/source/고정 SDK 안에서 기존 ephemeral build cache만 재사용한다.
서로 다른 PR/job의 cache restore/publish, module/tool trust나 네트워크 경계를 추가하지
않는다. 기존 두 source lookup은 lookup 방식만 이동하고 검사 내용은 보존한다. 실제 native
완료가 acceptance이며 로컬 BuildID 일치나 cross compile을 macOS 성공으로 쓰지 않는다.

[Cache 입력 정렬 로컬 영수증](./macos-cache-alignment-local-20261022.json)의 canonical
format/Quick/security/docs/workflow/platform은 PASS다. Quick124.093s·platform16.417s는
기존 로컬 cache를 사용한 결과이며 native cold timing으로 비교하지 않는다. Checker
Race36.760s(시험16.714s)·Darwin compile11.016s도 PASS다. Checker/CLI dependency 구조의
공유 package는205개이며 Sigstore 공유는0개다. 전체700개 의존성이 bootstrap과 같다고
과장하지 않는다. 완료 binary/일회성7파일12,607,903B는 Root 참조·query errors0 뒤 제거했다.
새 native macOS/Required 완료와 과거 timeout의 세부 원인은 계속 미확정이다.

후속 [CI374](./cache-alignment-ci374-20261022.json)는 sourcef76c564/same tree의 Required와
9 prerequisites가 모두 SUCCESS다. Native macOS의 같은 cache root/trimpath 실행과
build55.562s·default test envelope171.294s 이내를 확인했다. 이후 public native caller는
별도 후보이며, 과거 모든 timeout의 정확한 인과를 이 한 실행으로 확정하지 않는다.


## CI389 production build — bounded compile progress 보완 전 감사

[CI389 결과](./p1f-cpu-completion-ci389-20261023.json)는 Linux51/CPU216.40s·뒤 네 검사와
helper5wait0/정리가 PASS지만 macOS production build120s/Required는 FAIL이다. Default test는
미도달이다. Go1.26.10/darwin-amd64·runner image와 실제 step cache/trimpath는 직전 정상
CI386/387/388과 같았다. User197369ms/system107962ms는 CPU 사용 관측이며 정확한 package,
GC/OS 압력·cache miss 또는 새 source 인과는 확정하지 않는다.

기존 build·bounded capture/timeout IMPLEMENTED/WIRED=YES, 해당 후보 macOS QUALIFIED/
ACCEPTANCE_CLOSED=NO다. Package progress는 NO/NO/NO/NO이며 MISSING은 다음 진단과 실제
native 결과다. 기존 실행기·cache·validator·platform을 대체하지 않는다.

정확한 수정 위치는 scripts/check/runner.go의 foundationSteps(Quick)/platformSteps 두 production build 호출이다.
고정 Go의 build -v만 추가한다. 같은 SDK help와 cmd/go/internal/work/exec.go의 BuildV 분기는
package import path 출력이다. JSON parser·새 progress framework·-x/-work·환경/임시 경로/인자를
추가하지 않는다. 기존1MiB bounded capture와 timeout의 partial stdout/stderr/error cause 보존을
재사용한다. 출력된 package는 컴파일 진행 자료이며 해당 package가 정지 원인이라는 증거가 아니다.
성공 시 기존 quiet 출력과 실패 분류, build2m/default6m·job/Required·artifact CPU/poll/wall 한도는
유지한다. Source rewriting/warm-up/cache trust·검사 생략은 없다.

두 번째 검토에서 두 실제 consumer와 runGo/runCapturedCommand의 timeout/outputLimit 보존을
대조했다. 두 flag 변경에 구현을 반복하는 새 unit test는 만들지 않고 canonical format/Quick/
security/docs/workflow/platform 및 기존 checker timeout/capture Race를 검사한다. Actual native
source CI가 진단의 유효성과 macOS 완료 기준이며 로컬 cache 성공을 native cold 성능으로 쓰지
않는다. 진단 추가를 빌드 timeout 원인 해결로 기록하지 않는다.


[진행 출력 로컬 검증](./macos-build-progress-local-20261023.json)의 실제 두 호출은 -v만 추가했고
기존 bounded partial capture·error cause를 재사용한다. Canonical format/Quick34.079s/security/
docs/workflow/platform21.553s와 기존 checker 전체 Race22.786s는 PASS다. 모든 검사 종료 뒤
owned Go TMPDIR0이며 새 native 자원은 없다. IMPLEMENTED/WIRED=YES/QUALIFIED=LOCAL ONLY/
ACCEPTANCE_CLOSED=NO다. Native macOS/Required가 MISSING이며 내부 지연 원인은 미확정이다.
로컬 platform은 기존 cache 결과이며 cold macOS 성능으로 비교하지 않는다.


## CI425/source418a — 같은 deadline 관측의 재발, 직접 원인 OPEN

[CI425 observation](./ci425-macos-deadline-observation-20261029.json)은 실제20min 조회18:08:03UTC의
7SUCCESS/macOS FAILURE/Linux 진행과 all8checkout3cc28f9e/tree016df83d/2926 동일을 연결한다.
Native default6m에서47started/45finished·active2/unavailable=false/truncated=false이며 promotion/
update-runtime은 active root 없이 ok215.960s/137.413s가 있었다. 새 Hosttool package는2.658s로
완료했다. Command available=true/exited=false/exit-1·user507825/system371003ms이고 individual
FAIL은 없다. 개별 ok/높은 CPU/마지막 root를 완료나 syscall·cache·SDK 원인으로 바꾸지 않는다.

직전 green CI424와 runner/capture/aggregate-file-budget test의 blob은 같다. 이 사실만으로
whole-source 인과를 부정하거나 새 기능을 원인이라고 단정하지 않는다. CI369와 같은 관측
형태를 보존하며 기존5m/6m/coverage/offline/Required를 유지한다. 같은 실패의 blind rerun이나
새 진단 framework를 추가하지 않고 Linux를 끝까지 소비한다. 독립 C3b 구현 후 다음 실제
의미 있는 grouped native macOS/Required가 열린 gate다. 이번 후보의 직접 원인은 OPEN이다.
