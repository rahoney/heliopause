# P4-O4-C1 — fixed pipx DATA 공급과 전송

기준1166396, production012c7b4/CI418 whole tree3649f0bd/2726이다.
M13-004 하나만 IN_PROGRESS다. Read: O4 A/B map, Queue/plan20 §5/client09,
D-012/013/015, maintenance, self-update07, artifact L01/L02, sandbox L03/L08,
ci L06, evidence L11/L16. 기존 구현을 재작성하지 않고 아래 gap만 수정한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| selected ZIP/NOTICE identity | YES | YES(maintenance) | YES(local + existing CI418 consumers) | YES(A only) | installed supply DATA role |
| held installed file reader | YES(PNPM/Bun) | NO(pipx) | NO(pipx) | NO(pipx) | fixed0444 mode/asset/hash/size |
| bounded Docker tar producer/readback | YES(UV/PNPM/Bun) | NO(pipx) | NO(pipx) | NO(pipx) | fixed ZIP name/path/mode/8MiB bound |
| readonly mount lifecycle | YES(Python closure preparation/anchor) | NO(pipx runtime) | NO(pipx) | NO(pipx) | typed supply/readonly runtime integration |

C1은 `native_tool_transport.go`의 내부 fixed kind와 mode를 확장한다. Pipx ZIP의
`/haa-pipx/pipx-runtime.zip`은 UV의 `/haa-runtime/uv`와 겹치지 않는다. Caller가 filename,
mode·size/hash를 override하여 pipx transport 권한을 만들 수 없도록 compiled pin을 요구한다.
기존 UV image-source0755와 installed tool0555 의미는 보존한다.
`native_installed_tool.go`의 held root/current/file, owner/single-link·size/hash 전후 대조를
그대로 사용하고 fixed0444만 추가한다. 새 prepared wrapper는 bytes만 보유하며 Python을
실행하거나 public CLI/installer 경로를 활성화하지 않는다.

단일 RW `/haa-runtime` owner에 pipx를 넣지 않는다. Pipx kind의 일반 native-volume 생성은
거부하며 C2에서 기존 `pypi_closure_volume.go`의 preparation/anchor/readonly attachment
검증과 연결할 구체 runtime owner를 마련한다. Source file0444는 readonly mount 증명이 아니다.
C1 시험은 fixed identity/교차 tool 혼동·wrong mode/path/oversize·trailing tar·unknown
source/cancellation을 검사한다. 실제 원본 ZIP의 protected file과 tar 공급 시험은 disposable
namespace에서만 수행하고 원래 설치를 변경하지 않는다. 기존 UV/PNPM/Bun의 기본 회귀와
canonical Quick/security/docs 및 관련 Race도 확인한다. 작은 단위마다 push하지 않는다.

두 번째 검토: 기존 Python volume의 input/manifest 권한을 runtime tool 권한으로 자동 승계하지
않는다. C1 공급 성공은 provenance/Policy/ARTIFACT observation·release activation 성공이
아니다. C2 mount/observer와 B release/inspection, native pipx install·upgrade/CLI는 남는다.
