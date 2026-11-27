# P2c5-B5b4 — 기존 CI의 PNPM backend 실제 소비자 연결

Baseline9adbfa5/tree1e86e19395e7a0bc5edfe6d18a915791ee9d180e. M13-004/P2 하나만 진행한다.
Read: Queue, B5b3 audit/local receipt, 확장 계획§5, 기존 CI/quality 계약과 ci L04/L06/L10/L17/L22,
M12 예방 규칙§4/5·운영 참고문서v0.5§15. 기존 source/helper/실제 호출과 종료 순서를 대조했다.

## 독립 baseline audit

| 책임 / 기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| B5b3 guarded backend와 실제 bootstrap 소비자 | YES | YES(local actual) | YES(named local) | NO(P2) | 원격 실제 소비자 호출 |
| native 공급 — scripts/ci-pnpm-runtime.sh | YES | YES(sandbox actual) | YES(CI406) | NO(new backend) | 같은 공급의 bootstrap 실제 소비자 |
| 정확한 protected client 교체/owning helper stop·wait·start — ci-integration-cleanup.sh | YES | YES(existing lifecycle) | YES(local/remote) | YES(existing) | 새 순서에 그대로 호출 |
| 새 bootstrap backend/cache actual의 CI 호출 | NO | NO | NO | NO | tagged compile/고정 profile/설치/helper/원복·canonical checker |

기존 PNPM 공급 script는 sandbox test binary를 전제로 하고 native 설치를 EXIT에서 정리한다.
Bootstrap actual을 go test로 직접 실행하면 installed client와 다른 executable이며, child shell은
parent의 helper를 wait/rebind할 수 없다. 이 책임을 새 helper나 generic callback으로 대체하지 않는다.

## Edit map과 두 번째 검토

1. 기존 공급 script에 두 실제 소비자에 한정한 `project` 인수 하나를 추가한다. 인수 부재는
   지금 sandbox 경로다. 다른 값/추가 인수는 공급 전에 거부한다. 같은 locked archive materializer/
   설치/EXIT cleanup을 재사용하고 project는 고정 bootstrap cache+transaction 두 actual만 실행한다.
   임의 명령/경로를 실행하는 인수나 제품 flag가 아니다. 원래 sandbox commands/cwd를 보존한다.
2. 기존 lifecycle run block에서 첫 sandbox 공급 뒤 bootstrap을 realcorpus tag로 compile한다.
   Parent의 기존 `haa_install_ci_client`가 helper를 stop/wait한 뒤 정확한 binary를 설치하고,
   같은 parent가 helper를 새로 시작한다. 고정 project 공급 호출 뒤 parent가 원래 sandbox
   binary를 같은 함수로 다시 설치/start하여 뒤 PyTorch/UV/Go/Cargo/Terraform 호출을 유지한다.
   공급이 끝난 native를 빌려 쓰지 않고 두 profile 각각 자기 설치를 만들고 정리한다.
3. 기존 pnpm_runtime_workflow checker에 실제 profile command/guard/early-exit 고정 scope와
   active lifecycle의 compile→install→start→project→sandbox restore→start 순서를 연결한다.
   기존 active command 검사에서 누락/comment/duplicate/조건·실패 무시를 거부한다. 잘못된
   tag/client/order/restore·profile/env/test filter의 회귀를 추가하며 새 parser/job을 만들지 않는다.

두 번째 검토: child가 parent helper를 소유한다고 가정하는 방식과 설치된 sandbox binary로
bootstrap test filter를 실행하는 0-test 성공을 제외했다. Test filter는 두 정확한 tracked function과
tagged compile에 결합한다. Root script는 cache 단계를 포함하며 전체 set/actual Evidence/Policy와
target publication을 검증한다. 각각 기존5분 test bound, 기존 helper wait/CI120분을 유지한다.
Canonical format/Quick/security/docs/workflow와 Bash syntax·checker 부정 회귀 뒤 checkpoint한다.
제품 source는 B5b3와 같으므로 CPU/CUDA/관찰기를 재빌드하지 않는다. 새 CI script의 실제
project 분기 자체를 mount-only namespace의 같은 protected client/helper에서 한 번 소비하고,
일반 Go default와 named actual의 범위를 구분한다. 해당 caller 변경의 검증이며 기존 제품 검사를
추가 수정 없이 반복하는 이유로 확대하지 않는다.
원격 새 후보의 실제 호출/정리·Required는 별도 확인하며 CLI/native adoption/whole P2는 열려 있다.

## 로컬 확인과 원격 범위

Format1.148s/workflow1.911s(고정 actionlint 포함)/Quick57.670s/security8.630s/docs0.234s와
checker package 전체 Race29.928s·Bash syntax가 PASS다. 잘못된 profile/추가 인수의 실제
shell 조기 거부, tagged package/client/순서/restore/helper 누락과 env/filter/comment/실패 무시
부정 회귀를 확인했다. 기존 sandbox commands와 fixture cwd는 유지했다.

실제 CI script `project` 분기를 protected bootstrap client와 owning helper로 실행한197.220s는
locked native 공급·cache peer/empty42.69s와 transaction single/peer/empty147.08s 모두 PASS다.
Transaction은 install/repeat/add/native update의 최종 assertion과 교정된 log label을 포함한다.
두 tracked function을 compiled list에서 먼저 확인하여0-test filter 성공을 배제했다. Host 보호
파일/Docker 구성 전후 동일, helperwait0/CONFIRMED_STOPPED와 native/runtime directory 정리를
확인했다. Parent의 새 client rebind/restore 순서는 canonical checker로 검증했으며 원격 actual
순서까지 로컬에서 실행했다고 주장하지 않는다. 새 CI/Required·native macOS는 별도다.

이번 여섯 완료 runtime/compile 디렉터리는 작은 script 원본·전체 inventory를 보존한 뒤 정확한
process/FD/maps/mount 참조0/query errors0과 inode/owner/mode/hash를 재확인하여947,906,618B
삭제했다. 첫 정리는 검사기 자신이 FD 목록 수집을 끝내며 닫은 열람 FD의 query 오류1로 삭제
전에 중단됐다. 같은 동작의 직접 재현 뒤 FD를 iteration 중 즉시 조회하도록 교정했고 새 전체
조회가 통과한 경우만 삭제했다. 불명 상태를 부재로 바꾸지 않았으며 source/SDK/quality cache/
관찰기·다음 입력 archive와 raw evidence를 보존했다. 이전 Codex process RSS 부족의 수리라고
해석하지 않는다. [로컬 영수증](./p2c5b5b4-pnpm-ci-consumer-local-20261025.json)이 범위를 소유한다.

Backend/records/private materialization/CI caller를 묶어 한 번 push하고 그 새 원격 후보의 실제
결과를 확인한다. CI406의54m32에 새 project caller197s와 준비·재bind 여유를 더해 첫 확인은
새 실행 시작 약60분 뒤로 정한다. 완료 보장이 아니며 실제 timer/run/head에 연결한다. 대기
중에는 같은 P2의 CLI/독립 source-proof 인계 acceptance를 진행하고 작은 helper마다 재push하지
않는다. Native adoption/CLI·whole P2 acceptance는 계속 NO다.
