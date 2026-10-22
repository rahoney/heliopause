# CI388 unit completion 회계 순서 — 착수 감사·수정 지도

현재 M13-004/P1 안의 선행 회귀다. [최초 CI388 결과](./p1f-uv-original-proof-ci388-20261023.json)가
exact source/전체 tree/원본 로그·실패·미도달·복원을 소유한다. 전체 Required는 FAIL이며 앞 CI387의
성공으로 덮지 않는다. B1/A2c1은 별도 로컬 후보이고 아직 이 실행에 포함되지 않았다.

## Baseline과 책임

Baseline1cf5781의 CPU accounting ledger·주기 watcher·외부 outcome reconciliation은
IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES/ACCEPTANCE_CLOSED=YES(기존 CI387 범위)다.
세 owner bytes는36c6253/1cf5781에서 같다. 새 sync/inventory나 Go 전환이 원인이라고 쓰지 않는다.
Unit final Read/수용 전체가 watcher와 같은 serialization에 들어가는 경로는 NO/NO/NO/NO다.
MISSING은 실제 completion caller의 경쟁 재현·같은 counter/time 순서의 narrow correction,
기존 sticky/fault/cleanup 소비자와 새 exact-source CPU/공용 gate 및 공통 네 profile 검증이다.

실제 최초 거부는 sympy DIRECT_IMPORT/unit-outcome의 gap=-149ns이며 failed_before=false,
termination=false/usage=prior4403874us/1s poll·wall약893s 잔여다. 양의 sample 지연·CPU 소진과
구분한다. 과거 CUDA positive-gap 거부의 원인으로 일반화하지 않는다. Remote의 정확한 두
thread interleaving은 원본에 없으므로 로컬 통제 경쟁과 구분한다.

## 정확한 수정 위치와 방법

pypi_trusted_transaction.go의 unit loop 마지막은 resources.Read 후 ledger.finishUnit(...,
time.Now())를 직접 호출한다. 주기 watcher는 pypi_observation_cpu.go:sampleAccountingLocked에서
sampleMu·ledger.mu 아래 별도 Read/시각 수용을 수행한다. 완료 쪽의 Read와 timestamp는 이
sampleMu 밖이며 timestamp 평가 뒤 ledger.mu 대기 중 새 주기 sample이 먼저 수용될 수 있다.
동일하거나 증가하는 counter에서도 거짓 순서 역전·오래된 counter 수용 경쟁이 가능하다.

새 private observationCPUWatch.finishUnit(ctx, externalUnitEvidence)는 실제 final Read부터 기존
ledger.finishUnit 수용까지 같은 sampleMu를 유지한다. Transition 진행 중에는 거부하고 Read는
기존 poll deadline 안에 끝내야 한다. fresh UsageUsec를 기존 evidence에 채우며 모든 외부 outcome/
observer/closure/drain·sticky/CPU/time/coverage 검사는 기존 ledger.finishUnit 그대로 사용한다.
Caller는 이 helper만 부른다. helper의 sampled bool은 기존 Read 실패와 ledger outcome 실패의
error code 구분만 보존하며 completion authority가 아니다. 새로운 clock tolerance/clamp,
음수 gap 허용·sample 무시·counter reset·poll/CPU/wall/reserve 변경은 하지 않는다.

원래 final Read+finish 본문의 동일 의미 추출을 before overlay로 보존하고 같은 barrier regression을
교정 전후 대조한다. Blocking trusted Read 중 동일 sampleMu의 periodic actor가 실행 가능한지
TryLock으로 결정해 실제 sampleAccountingLocked를 호출한다. Old completion은 이를 허용해
새 periodic100→101 뒤 stale final100이 거부된다. 교정은 이 actor를 final acceptance 뒤로 직렬화하고
다음 실제 sample도 받는다. 무작위 sleep·full 재시도로 옛 interleaving을 추측하지 않는다.

추가 부정 검증은 read missing/deadline/cancel·실제 late/counter decrease/CPU reserve·incomplete
external outcome/transition으로, 성공한 final Read가 실패한 ledger를 회복하거나 미완료 unit을
완료하지 못해야 한다. 기존 sample-order refusal과 terminal-failure polling·pending-read stop,
phase drainage/primary-cause regressions도 유지한다. Canonical5/full sandbox·artifact·관련
Bootstrap/Application Race와 기존 CPU/Linux/native macOS/Required를 검사한다. Common Python
회계 caller이므로 CPU/cu126/cu130/cu132 actual 범위를 별도로 확인하며 CPU를 CUDA 성공으로
전용하지 않는다. Runtime pin/정책/Host 활성화·기존 Go1.27.3 순서는 이 교정에서 바꾸지 않는다.

두 번째 검토에서 production final Read의 mutex 밖 위치와 timestamp 평가→ledger lock 순서,
watcher의 sampleMu→ledger.mu 순서, transition/종료/Close 책임과 기존 두 오류 code를 대조했다.
Read 중 sampleMu를 잡는 만큼 poll-bound context가 필수이며 lock 역순을 추가하지 않는다.
새 helper는 controller facts만 받으며 artifact token/stdout를 완료 증거로 쓰지 않는다. Existing
ledger를 대체하거나 fail-closed invariants를 수정할 필요가 없다. 원본 실패/통제 재현/후속 full
범위를 분리하고 완료한 temporary fixtures만 정리한다.
