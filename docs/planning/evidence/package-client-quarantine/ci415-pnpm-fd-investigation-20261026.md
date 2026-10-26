# CI415 PNPM peer FD 추적 조사

M13-004 하나만 IN_PROGRESS이며 sourcea8389de/tree953c3fd2a2a7bc107a2b00d73c3590f8ef61a8a7
/2635의 CI415는8SUCCESS/Linux·Required FAILURE다. 최초 실패는
TestPNPMPublicProjectTransactionIntegration/peer, RAW SENDTO fd14 DESCRIPTOR
FD_STATE_UNKNOWN이다. Native resolver8와 approved cache·transaction single은 PASS였고
UV tool recipe는 NOT_REACHED다. Helperwait0/RESTORED/primary1은 보존됐다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Kernel SocketPairResult와 exact entry/result FD authority | YES | YES | YES(existing named latches/runtime) | NO(current regression) | peer fd14 유실/생성 event 원인 대조 |
| Existing socket/close/dup/fcntl/clone/exec FD lifetime | YES | YES | NO(current peer failure) | NO | 로그·source·targeted native 소비자에서 최초 불일치 확인 |
| CI 상태 감시 | YES(timer process) | YES | NO(early failure missed) | NO | 다음 필요한 CI의 실제10분 결과 소비 |

Guest SocketPair return buffer는 진단이며 LOCAL/network 판정 권한이 아니다. 신규 UV query
변경은 FD 공통 코드를 의도적으로 바꾸지 않았지만 이것만으로 인과 부재/원인을 단정하지 않는다.
Source 수정 전에 현재 kernel producer/observer event 순서와 FD lifetime을 대조한다. 기존
구현을 재작성하거나 진단 값으로 unknown FD를 허용하지 않는다. 재현 성공은 원격 실패의
해결 증명이 아니며 같은 입력의 무조건 full CI 재실행은 하지 않는다.

Read: plan20/client09, observer.cc의 ParseSocketPairEntry/Result 및 socket/close/dup/fcntl/
clone/fork/exec/RAW 처리, pod-init, pinned gVisor patch, PNPM transaction 실제 fixture,
lessons sandbox L08/L18·SocketPair authority, ci L06, evidence L16.

원본은 docs/github-logs/package-client-quarantine-20261021/ci415-*-decoded.log와
ci415-decoded-log-inventory.json(10logs897772B), initial/final snapshots 및 timer 기록이다.
Current N1 source는 이 CI 입력에 포함되지 않는다. 다음은 unchanged candidate의 peer
소비자를 좁혀 local/hosted 차이와 첫 FD 불일치를 확인하는 단계다.

## 첫 로컬 비교 — 해결 판정 아님

Unchanged D의 exact compiled bootstrap binary와 canonical121 observer로 peer만 실행한
ci415-pnpm-peer-baseline-v1은79.002s/peer74.49s PASS다. Current 전체 승인4entries/
반입·native no-op/user files 보존, helperwait0와 Host 전후 동일을 확인했다. Local fresh
resolve 입력은 hosted freeze replay가 아니며 원격 FD14 실패를 반증하거나 고쳤다는 증거가
아니다. 같은 무수정 실행을 반복하지 않고 event lifetime/순서의 진단을 준비한다.

All10 checkout39d88f3의 부모는 baseef5ef0e와 sourcea8389de이며 전체 tree953c3fd/2635가
source와 bytes 동일하다. Raw ci415-whole-checkout-proof.json에 연결했다. CI415 Quick은
Git2.55.0에서 SUCCESS다. CI414의 disposable Git cleanup 보완 후보에 대한 named 원격
성공이며 과거 detached maintenance가 원인이었다는 직접 증명은 아니다.
