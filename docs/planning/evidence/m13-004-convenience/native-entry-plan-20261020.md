# M13-004 — 실제 native helper의 inert entry gap

Root transaction의 mock positive는 actual native set qualification이 아니다.
추가 source 대조에서 실제 실행 contract의 아래 acceptance gap을 확인했다.
검사/Policy를 완화하거나 native helper를 RESOURCE로 면제하지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 gap |
| --- | --- | --- | --- | --- | --- |
| generic M6 raw ELF 무인자 실행/관찰/종료 | YES | YES | YES, 기존 범위 | YES, 기존 범위 | 그대로 재사용 |
| observer 실제 socket/readiness 실행 | YES | YES | YES, 기존 범위 | YES, 기존 범위 | 기존 argc4/5와 identity 유지 |
| network helper 실제 typed service | YES | YES | YES, 기존 범위 | YES, 기존 범위 | 기존 ServeNetworkPolicy 재사용 |
| 두 helper의 inert default/help entry | NO | NO | NO | NO | 실제 새 릴리스의 dynamic check 완료 불가 |

`sandbox/github_elf.go`는 raw ELF를 무인자로 실행하고 nonzero status를
M6_DYNAMIC_EXECUTION_FAILED/INCOMPLETE로 보고한다. observer.cc의 argc1은 현재
usage/exit2이고 network helper는 무인자로 privileged service를 시작해 isolated rootfs의
config/runtime 부재로 exit1이다. 두 파일을 포함한 전체 required set은 실제로 ALLOW를
닫을 수 없다. 앞선 synthetic sandbox positive를 실제 바이너리의 성공으로 확대하지 않는다.

## 수정 위치와 순서

1. `tools/gvisor-observer/observer.cc`: argc1/--help만 usage/exit0로 먼저 종료한다.
   기존 --identity와 실제3 sockets/[--ready-fd] branch, observer state machine,
   attribution/ledger/CPU/limits/trace 처리는 변경하지 않는다.
2. `cmd/haa-network-policy-helper/main_linux.go`: 얇은 argv dispatcher를 추가한다.
   무인자/help/version은 inert이고 실제 서비스는 명시적 `serve`만 허용한다.
   Root API/Hosttool engine/wire는 기존 ServeNetworkPolicy를 호출한다. Install/recover/
   target/UID/임의 argv 등 설치 역할은 넣지 않는다.
3. 실제 소비자인 `scripts/ci-integration-cleanup.sh`, native `nativePolicyUnit`와
   `scripts/release-packages/main.go`의 ExecStart를 `... serve`로 함께 바꾼다.
   Root/mode/env/owning-shell wait/cleanup·root service/socket invariants는 유지한다.
   실제 다른 시작 호출은 rg inventory로 확인하고 historical evidence/raw는 편집하지 않는다.
4. Go dispatcher의 inert/reject/cancel 회귀와 기존 canonical helper lifecycle tests,
   native/package unit expectations를 검사한다. 무인자 native 실행을 default command
   자체의 exit0로 확인하며 echo/help 출력만 actual service나 검역 성공이라고 쓰지 않는다.
5. C++는 canonical pinned `scripts/build-gvisor-observer-release.sh`를 사용한다.
   기존 exact gVisor source/Bazel SHA/builder image와 full latch test를 재사용하고 실제
   built binary의 default/help/identity를 확인한다. Floating host compiler/SDK/latest로
   바꾸거나 observer runtime invariants를 면제하지 않는다. 완료 binary/build inputs는
   마지막 사용 후 제거한다. 실제 installed helper/host config는 바꾸지 않는다.

두 번째 검토에서 generic ELF의 nonzero를 완료로 바꾸는 방법, helper 제외/static-only,
artifact의 --help 출력으로 safety를 승인하는 방법을 제외했다. Actual service entry를
explicit하게 연결하고 generic 동적 검사는 그대로 유지한다. 새 signed native4의 전체
격리 검역/실제 protected activation은 별도 실제 qualification으로 남는다.
