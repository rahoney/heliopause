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

아직 위 timing/OS 관측 코드를 작성하거나 새 진단 run을 dispatch하지 않았다.
