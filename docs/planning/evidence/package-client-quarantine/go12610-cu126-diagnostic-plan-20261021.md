# Go1.26.10 cu126 accounting-late 조사 계획

같은 source c2c1750의 ordinary Required/CPU는 PASS, 별도 cu126은 FAIL이다.
[원격 상태](./go12610-remote-status-20261021.json)와 raw failed log가 첫 오류를 소유한다.
후속 UV pin 후보는 CPU accounting/helper 구현을 수정하지 않았으며 해당 자료 검증은 독립 범위다.

## Baseline과 실제 gap

| 기존 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| sandbox/pypi_observation_cpu.go sampleAccountingLocked/run | YES | YES | YES(기존 bounded scope) | YES(기존 scope) | 새 source cu126의 실제 sample-late 실패 조사 |
| hosttool/observation_resource_linux.go ObservationResourceClient.call / serveObservationResources / observationResourceEngine.read | YES | YES | YES(기존 scope) | YES(기존 scope) | client/server 어느 구간이 늦었는지 신뢰 가능한 scalar timing |
| 기존 diagnostic test·에러 timing | YES | YES | YES | YES | 현재 Read589ms를 내부 latency/OS pressure로 더 분해 |
| 새 source cu126 full qualification | YES | YES | NO | NO | 실패 원인 관측, 필요한 수정·회귀, 같은 최종 source의 실제 full PASS |

CPU58.28/300s·wall잔여2337.34s이며 전체 예산 소진이 아니다. Read589.410876ms +
beforeRead445.640217ms + ledgerLock0.001132ms의 실제 합이1.035052225s로1s를 넘었다.
CANCELED와 PROFILE_LOOKUP_FAILURE는 뒤의 teardown 상태다. Go SDK 회귀·helper engine·
OS memory pressure 중 어느 것이 원인인지는 아직 확정하지 않는다. 과거 비슷한 지연에
대한 WSL memory-pressure 관측을 이번 GitHub runner의 증거로 전용하지 않는다.

## source 변경 전에 확정한 조사 경계

1. 기존 typed resource client에서 socket 신원 확인·dial·encode·response/EOF·최종 신원
   확인 시간을 분리한다. Server에서는 accept 뒤 peer authentication·request 수신/parse·
   engine 호출·response write/close를 같은 기존 RPC 흐름에서 측정한다. Unknown peer의
   요청 내용이나 nonce·secret·artifact payload를 diagnostic에 넣지 않는다.
2. Diagnostic은 bounded fixed enum과 trusted duration만 허용한다. 기존 wire schema,
   peer identity, request/response EOF 검사·timeout·poll cadence1/2·최대 gap1s·sticky failure·
   cpu300s·wall40m·stop reserve와 ALLOW 기준은 변경하지 않는다. Artifact 출력은 authority가 아니다.
3. 실제 영향을 받는 full CUDA 구간에는 1Hz의 bounded OS MemAvailable/Swap/PSI/vmstat
   관측을 연결한다. Runner 용량과 selected tmpfs12GiB + runtime4GiB + controller/OS 여유를
   함께 기록하되 free 값 하나로 peak 충분함을 선언하거나 임의 budget을 확대하지 않는다.
4. 새 관측은 최초 실패를 보존하고 helper/서비스/원래 client 복원 receipt와 연결한다.
   실패한 source를 원인 없이 다시 실행해 green을 만드는 것은 조사/해결이 아니다.
   정상·느린 단계·malformed/unauthorized·취소·cleanup을 결정적으로 검사하고 canonical
   Quick/security/docs/workflow 및 기존 helper 소비자를 확인한 뒤 원격 실제 관측을 수행한다.
5. 조사 결과가 실제 필요한 좁은 수정을 가리킬 때만 수정 위치·이유와 회귀를 추가한다.
   같은 원인의 해결과 후속 scheduling 조건의 일회성 PASS를 구분한다. 진행 중인 cu132와
   예약된 cu130의 독립 결과는 같은 c2c1750 범위로 보존하며 새 관측 source와 합치지 않는다.

2026-10-21 a9bd096 기준의 다음 구현을 시작한다. 새 진단 run은 아직 미dispatch다.

### 두 번째 검토와 구체적인 전달 위치

Client timing은 Host가 측정한 값만 ObservationResourceLease의 json:"-" local metadata로
되돌린다. Bootstrap의 기존 observationLease mapper가 sandbox port로 전달하고 기존
sampleAccountingLocked가 실패를 반환할 때만 fixed scalar를 붙인다. 새 RPC field·optional
interface·artifact token·성공 로그를 추가하지 않는다. Error나 lease의 timing은 CPU 허가·
분류·완료 판정에 사용하지 않는다. JSON response로 위조한 timing field는 unknown field다.

Server slow-read timing은 정확한 authenticated read에만 최대16건/서비스 수명으로 제한한다.
기존 connection을 닫은 뒤 stderr에 쓰므로 client의 EOF를 logging이 지연시키지 않는다.
추가 writer queue/goroutine·서비스·config는 만들지 않는다. Existing handler/engine·
requestLimit·timeout/peer/EOF 순서는 보존한다. 타임스탬프와 stage duration만 기록한다.
구체 수정은 hosttool/observation_resource_linux.go와 작은 같은-package diagnostic helper/test,
bootstrap/observation_resource_linux.go, sandbox/pypi_observation_cpu.go와 기존 timing 회귀다.
CI의 기존 CUDA 호출부·ci-integration-cleanup.sh를 읽고 OS 관측과 helper stderr 회수도
현재 lifecycle에 연결하며 Required job/프로필 예산은 유지한다.

## 구현·로컬 검증 중간 상태

Trusted client timing은 local-only lease metadata와 기존 bootstrap mapper를 통해 실패
진단에만 전달된다. Server는 connection close 뒤 authenticated read의 지연을 최대16건
보고한다. CI 종료는 마지막64KiB에서 형식이 맞는 최근16행만 회수하며 원래37 등의 실패
상태와 helper lifecycle을 보존한다. Full PyTorch test에는 같은 context 수명의1Hz
메모리/PSI/vmstat/controller RSS 관측을 연결했고 query 실패는 unavailable로 기록한다.
기존 canonical ResourcePolicy의 memory/tmpfs 값도 기록하되 충분성을 단정하지 않는다.

첫 transport-only Quick141.33s/security6.00s/docs0.208s, OS·CI 회수까지 포함한 후보의
Quick44.787s/security4.484s/docs0.198s/workflow0.64s가 PASS다. Targeted Race39.984s와
기존 실제 cleanup/script-file 회귀0.926s도 PASS다. 첫 추가 검증의 자동 승인 검토 시간
초과는 실행 전이었으며 허용된 한 번 재시도의 결과만 성공으로 기록한다. 전체 기존 Race
범위나 실제 cu126 지연 해결·새 원격 성공이 아니다. Darwin amd64/arm64의 bootstrap/sandbox/check TEST compilation6개도 PASS다. 실제 native 실행은 새 CI에서 따로 확인한다.
