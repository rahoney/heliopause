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

## 원인 확정 전의 진단

CI414source294bb57와 CI415sourcea8389de의 runtime lock·gVisor patch·pod-init·PNPM
transaction fixture는 bytes 동일하고, 기존10개 FD 처리 함수도 동일하다. 이는 원격 입력/
실행 순서가 같다는 증명이 아니다. Raw ci415-fd-source-comparison.json이 대조를 소유한다.

고정 kernel에서 Close는 FDTable.Remove 뒤 OnClose를 실행하고 일반 exit event는 syscall
함수 반환 뒤 만들어진다. Remote sink의 각 task caller는 marshal/Writev를 수행한다.
서로 다른 thread의 FD 재사용과 event 순서가 어긋날 여지는 source에 있으나 CI415의
정확한 interleaving은 아직 증명하지 못했다. Guest return buffer/진단 값을 권한으로 쓰지 않는다.

다음 임시 diagnostic은 production tree 밖에만 만들고 기존 source의 다섯 insertions를 제거하면
original bytes가 복원됨을 검증했다. 각 stream은 최대256개 numeric-only FD events를 유지하고
한정 trace directory에 최대60000B를 기록한다. Payload/network-address/path/process-name은 기록하지
않고 판정 입력·권한·budget은 바꾸지 않는다. 이 산출물의 actual은 qualification이 아니다.
N1 source는693ba4a/tree78a2255/2641 로컬 commit으로 보존했고 원격 push는 하지 않았다.

## 2026-10-27 재개 — bounded failure diagnostic 지도·재검토

임시 build259.763s/121latches와 actual73.938s는 PASS였고 helperwait0/Host 전후 동일이다.
12개 stream의303369B·retained3072 numeric rows에는 fault/unknown send가 없었다. 정상
source 중 FD14는 AF_NETLINK16으로 재사용됐지만 CI415 당시 같은 family였다는 증명은 아니다.
Raw ci415-fd-diagnostic-v1-build/result/trace-inventory가 관측을 소유한다. 같은 무수정 actual은
반복하지 않으며 원격 실패 원인/해결은 OPEN이다.

| 보완 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 RAW failure의 직전 FD mutation 원인 정보 | NO | NO | NO | NO | helper bounded scalar history→기존 FaultRawDiagnostic→error text와 정상/negative/1024B bound 검증 |

기존 helper envelope/collector/validator를 확장하고 권한 처리 함수는 재작성하지 않는다.
Per-stream 최대16개 FD mutation을 group/start/fd·fixed operation/family 숫자로만 기억하고,
실패한 FD에 해당하는 마지막 두 건만 기존 RAW failure에 넣는다. 원장·권한·예산·FD table은
이 기록을 읽지 않는다. 원본16건 이전은 UNKNOWN이며 history 부재를 정상으로 보지 않는다.

수정 지도는 observer.cc의 독립 기록 함수와 Handle/RAW diagnostic 연결, collector.go의
fixed fields와 표시, shared_observer.go의 enum/배치 검증, 기존 Go/C++ latch의 bound/forged/
close-after-reuse 진단 회귀다. Second review: diagnostic만 있는 경우도 FD_STATE_UNKNOWN과
SessionComplete=false를 유지하고 stream-end/다른 reason/임의 operation·혼합된 부재 필드를
거부한다. Helper1024B/기존 budget은 늘리지 않는다. Kernel patch/runtime lock/실제 FD 권한
predicate는 불변이며 CI 재실행 자체나 원인 해결로 표시하지 않는다.

## FD 오류 진단 구현·첫 검증

Existing RAW envelope의 fd_last/last_family/previous/previous_family를 연결했다.
최대16건에서 group/start/FD가 맞는 마지막 두 fixed operation/family만 표시하며
FD_STATE_UNKNOWN/SessionComplete=false를 유지한다. Canonical Quick113.572s/
security13.362s/docs0.276s와 Sandbox 전체 Race31.703s가 PASS다. 새로운 history 네 부분을
제거하면 observer.cc의 기존 전체 bytes가 복원됨을 기계적으로 확인했다.

첫 canonical C++241.733s는121PASS/1FAIL이었다. New diagnostic/forged LOCAL 거부는
PASS였고 기존 CLOEXEC fixture가 추가 fields 이전의 exact JSON만 기대했다. Test-only
expected JSON에 정확한 EXEC/-1→SOCKET/AF_INET 두 필드를 요구하도록 교정했다.
Unknown TABLE의 exact no-history 비교와 production bytes는 그대로다. First raw와
ci415-fd-history-first-latch-assessment.json은 보존하며 corrected full builder/actual은 대기 중이다.

## 한정된 최종 로컬 결과

Corrected canonical237.312s/122PASS·0FAIL와 최신 Go client의 actual80.832s/peer77.11s가
PASS다. Policy4entries·current 반입/native no-op·원본 inode/user files 보존, noSKIP/
helperwait0와 Host 전후 동일을 확인했다. 실패 JSON의 fixed history가 추가된 진단 경계만
IMPLEMENTED=YES/WIRED=YES/named local QUALIFIED=YES이며 CI415 원인·해결은
ACCEPTANCE_CLOSED=NO다. 원본을 고치거나 같은 CI를 재실행하지 않았다.

[로컬 영수증](./ci415-fd-history-local-20261027.json)이 canonical/실제 결과와 첫 실패·권한
bytes 불변을 연결한다. 다음 필요한 grouped CI에서 실제10분 상태 타이머를 소비하고, 새
실패가 있으면 이 진단으로 조사한다. 그 전에 독립 N2A authentic wheel/interpreter/script
footprint proof를 계속한다. Work item과 final profile support 범위는 바뀌지 않는다.
