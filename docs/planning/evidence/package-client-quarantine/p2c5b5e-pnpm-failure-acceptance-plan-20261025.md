# P2-B5e transaction failure acceptance 대조

M13-004의 같은 P2 acceptance gap이다. 원격 frozen source9410c2d의 ordinary/cu126/cu130은
Required10SUCCESS, cu132411은 첫 timer04:22:46UTC 대기 중이다. 그 원격 source를 로컬
Bun c9a6143 이후 source 성공으로 전용하지 않는다. 새로운 engine·work item을 만들지 않는다.

Read: 확장 계획§5의 각 transaction 실패 matrix, P2 baseline§P2d, B5b3 착수 감사와 결과,
client 설계§4, 실제 npm_transaction.go·pnpm_transaction/profile/approved_transaction와
현존125개 PNPM test 진입점, cmd/helox/main.go의 signal.NotifyContext, 관련 lessons.

## Baseline audit

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| guard/select/fresh inspect/transaction/CLI/native ownership —9410 | YES | YES | YES(named sources/CI408–410) | NO(whole P2) | cu132와 마지막 matrix |
| shared no-replace backup/publish/rollback + independent state — npmProjectTransaction/PNPM profile | YES | YES | YES(기존 단계 오류·rollback conflict·no-op·cancel-before-commit) | NO(아래 실제 실패) | mid-publication cancellation/SIGINT·filesystem ENOSPC·subprocess crash/re-entry 증거 |
| PNPM 구체 실패 테스트125개와 기존 npm update 회귀 | YES | YES | YES(기존 범위) | NO(§5 전체) | 아래 세 가지 실행 상황의 직접 acceptance |

기존 모든 phase에 errors.New로 실패를 주입하는 시험은 실제 signal delivery/ENOSPC/process
termination과 다르다. `TestPNPMGuardRejectsDriftBeforeSelectedCacheAccess/cancel`은 Commit
이전 취소다. `TestPNPMRollbackConflictPreservesForeignOutputAndRecoveryJournal`은 rollback
불확실 상태와 재진입 거부를 검증하지만 실제 child process crash로 표시하지 않는다.
이 대조는 기존 성공을 취소하지 않으며 아직 실행하지 않은 실패 상황을 성공으로 덧붙이지 않는다.

## 수정 지도와 두 번째 검토

기존 `pnpmTransactionFixture`와 request/operation/cache/records를 재사용하는 acceptance-only
시험부터 추가한다. Production engine은 새로 구현하지 않는다. `owner.checkpoint`는 이미
정상·실패 검증을 위한 private seam이며 실제 signal/중단 시각을 동기화하는 데만 사용한다.

1. Mid-publication cancel: 실제 subprocess에 signal.NotifyContext와 SIGINT를 전달한다.
   Named after-publish phase에서 parent/child pipe로 동기화해 sleep 추측 없이 신호를 보낸다.
   Cancel 이후 최초 context.Canceled/cause, 전체 원래 target bytes/mode/inode와 독립 state,
   미게시 records·guard/owned temporary 정리를 확인한다. Test runner 자체를 죽이지 않는다.
2. Crash/re-entry: exact checkpoint에서 별도 child만 종료해 원래 backup/journal을 보존한다.
   새 guard/Begin이 이전 journal을 발견하고 mutation/cache 접근 전에 거부하며 unknown/원본
   자료를 지우지 않는지 확인한다. 실제 recovery 기능을 새로 약속하지 않는다.
3. ENOSPC: synthetic err 문자열만으로 disk full을 주장하지 않는다. 별도 mount/network
   namespace의 작은 task-owned tmpfs에서 실제 write ENOSPC를 확인하고 정해진 metadata/
   approval 경계에만 한시적으로 공간을 채운다. Host filesystem·공유 cache·사용자 project는
   채우지 않는다. 원래 cause·rollback·descriptor/잔여 data를 직접 확인한다.

새 직접 실패가 발견될 때만 exact owner를 수정하고 첫 source/log를 보존한다. 영향받은
canonical checks/전체 promotion Race/실제 소비자를 선택한다. 이 acceptance 작업은 아직
미실행이므로 P2를 COMPLETE로 올리지 않는다. 원격411 확인은 예정 timer에 실제 소비하며,
Bun query의 독립 고정 observer build는 그와 별도로 계속한다.

## 직접 acceptance 결과

[실제 실패 검증](./p2c5b5e-pnpm-failure-local-20261025.json)은 production transaction을
변경하지 않고 실제 SIGINT/SIGKILL6.736s 및 최종 Race15.162s, namespace의 실제 ENOSPC
0.173s PASS를 연결한다. Signal은 parent/child ready pipe로 exact partial publication에
동기화했다. SIGINT는 context.Canceled와 원본 bytes/inode/mode/독립 state 복원·미게시를
확인했다. SIGKILL 뒤에는 exclusive lock·원래 backup이 남고 cache 읽기 전에 Begin이
거부됐다. 실제 자동 recovery 기능은 주장하지 않는다. 실제1MiB tmpfs의 ENOSPC는 상태 게시
실패까지 전파됐고 원래 cause·완전한 원본 복원·partial/temp 부재·task FD0을 확인했다.

위 production owner 일곱 파일은 remote9410과 bytes가 같다. Tagged qualification source만
추가했으며 default compiler input은 바뀌지 않았다. B5e의 세 gap은 named local acceptance로
닫혔지만 CI411의 macOS/Required 실패는 별개로 남는다. Linux/cu132의 성공으로 macOS를
대체하거나 같은 실패 source를 근거 없이 재실행하지 않는다. 다음 의미 있는 Bun/runtime
checkpoint의 성공한 macOS/Required와 전체 지원 matrix가 P2 최종 인계의 남은 조건이다.
