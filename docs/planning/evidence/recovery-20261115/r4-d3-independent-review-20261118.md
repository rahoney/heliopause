# R4 적용 전 D3 독립 검토 (2026-11-18)

**REVISE — 현 U1/D3 적용 보류.** 승인된 D5 범위에서 별도 검토자 `/root/d3_entry_review`가 원개발 HEAD4a43f2cd의 미커밋 U1과 실제 R3 treea95c084f를 읽기 전용 대조했다. build/test/Host/CI/소스 변경은 하지 않았다. 원개발 미커밋 파일은 R3에 적용된 구현이 아니며 경로·해시는 [검토 신원](./r4-d3-independent-review-20261118.json)에 결합한다. R3에는 U1 lease/label/flock 자동회수 없이 기존 supervisor가 유지된다.

| 항목 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 원개발 U1 기구 | YES | YES | NO | NO | 아래 실패·재시작·생성 종료 경계 |
| §6b 전체 D3 계약 | NO | NO | NO | NO | 영속 불확실 상태·재진입 차단·생성 종료·helper 재시작 |
| 사전 dirty/retained 방향 | NO | NO | NO | NO | 미구현 설계 방향, 안전성·실물 미검증 |
| F6/144/148 복구 적용 | 원개발 YES | 원개발 YES | 복구 NO | NO | R3 위 독립 선별·연결·동일후보 검증 |

## 적용을 막는 실제 코드

이 절의 경로는 원개발 미커밋 작업 트리 기준이다.

- `internal/hosttool/process_lease_linux.go:81–89` hold는 reclaim.complete=false여도 lease를 삭제하며 새 실행 차단 상태가 없다. 같은 파일114–129의 quiet2는 이미 승인된 늦은 create의 종료 근거가 아니다. 회수 중 lease가 유효하고 `observation_resource_linux.go:323`의 create가 새 session을 허용할 수 있다.
- `internal/bootstrap/bootstrap.go:225–230`은 cleanup 결과와 무관하게 releaseLease를 호출한다. 명시 release 뒤 helper는 잔여 확인 없이 소유 정보를 지운다. 오류 반환과 다음 실행 차단은 별개의 계약이다. supervisor가 observer 죽음만 확인한 것도 비신뢰 runtime 전체 중단의 증명이 아니다.
- `observation_resource_linux.go:628–629`는 재시작 때 sessions/leases RAM map을 새로 만든다. client는 lease 연결 단절을 감시하지 않는다. `policy_engine.go:115`의 RAM 기록 부재 시 nil은 방화벽 부재 증거가 아니다.
- `HoldProcessLease:245–250`의 config/socket 실패 nil lease와 create의 빈 Owner 허용이 결합되면 helper 기동 경합에서 무소유 session/label 없는 자원을 만들 수 있다.
- `observer_supervisor.go:218–222`는 retained WriteString의 길이·오류를 무시하고 Sync가 없다. 다음 시작은0바이트 lock을 회수하므로 기록 실패로 불확실성이 사라질 수 있다.
- 같은 파일322는 모든 endpoint를 unix stream으로 조회하지만 실제 output은 `shared_observer.go:468`의 SOCK_DGRAM, remote/control은 `observer.cc:7135/7142`의 SOCK_SEQPACKET이다. 신규시험도 stream만 만들어 실제형 stale/live 판정을 검증하지 못했다. 이 항목은 정적 불일치이며 런타임 재현을 수행했다고 쓰지 않는다.

과거 kill 로그는 좁은 성공 증거다. 관찰 단계의 교정된 SIGKILL 원본에도 rerun_exit255·slice1이 남아 전체 정리/재실행 성공으로 확대할 수 없다.

## 최소 재사용과 진짜 승인 경계

기존 authenticated helper·transaction/cgroup owner·Terminate의 외부 drain·정확한 Docker ID/정책 제거·supervisor 소유 inode 검사를 재사용한다. 새 daemon/범용 복구 체계를 만들 근거는 없다. 사전 보호 상태를 택한다면 어떤 비신뢰 자원 생성보다 먼저 기록·동기화가 완료돼야 하고, 실패면 시작하지 않아야 한다. cleanup 오류·helper 단절·생성 종료 불명에는 상태를 유지하고 동일 보호 범위 재진입을 거부해야 한다. 소유권·생성 종료·비신뢰 실행 전체 중단·session 종료·회수/격리가 모두 대조된 뒤만 해제를 검토한다. 상태 파일만으로 실행 중 payload가 멈추지는 않는다. helper 중단·재시작 reconciliation이 입증되지 않으면 수동 조건을 명시하고 acceptance OPEN이다. 이 새 dirty/해제/재시작 계약은 exact 후보·검증·독립 검토·사용자 승인 전 미채택이다.

일반 R0~R8 선별 사전 승인으로 **U1 없는 F6+144/148의 독립 선별·구현·검증은 재질문 없이 진행 가능**하다. R3 mandatory full과 기존 단계 절차가 선행한다. F6 f258669의 executor 추가는 StartContainerInputOutput이며 lease가 불필요하다. 148 drain은 기존 admissionAware runner·activeObserverSession·C++ clone/exit 원장에 의존한다. 원개발 responder.go:157의 ownerLabelCount 예외는 U1 전용 hunk이므로 제외한다. 기존 엄격한 supervisor를 보존하고 U1 자동재사용을 넣지 않는다. 이 부분 진행을 §8.3의 D3 승인+S 충족이나 R4 전체 완료로 바꾸지 않는다.

## 필요한 정상·거부 검증

- 정상 완료/거부/SIGTERM의 기존 소비자·최초 오류·정확한 정리/상태 해제.
- query/rm/policy/keeper/late/deadline 실패와 cleanup 실패 뒤 explicit release: 상태 보존·다음 실행 거부.
- quiet2보다 늦은 create/응답 유실/회수 중 create: 잔존 실행 중단 또는 보호 상태·재진입 차단 유지.
- retained 짧은 write/권한·용량/Sync 실패와 경계 SIGKILL: 재시작 뒤 불확실성 보존.
- CLI/observer/helper 개별·동시 kill/restart: 다음 요청 없이 모든 비신뢰 실행 중단 외부 확인. 불명은 격리/거부/OPEN.
- 실제 DGRAM/SEQPACKET stale/live·inode 교체·옛 lease/network/volume·동시 시작·무관 작업 보존.
- 독립 F6/148: 정상 dummy 읽기, 자극에서만 활성화되는 표본 탐지/차단, detached 정상 drain·timeout·UNKNOWN/관찰 소실 거부. 기존 wall/CPU/메모리 한도 유지.
