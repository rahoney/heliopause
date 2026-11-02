# S01/S08 모의 환경·파일 변화 baseline

기준 `3f9d52f`, active work item은 M13-004 하나다. 계약과 완료 조건은
[보강 설계 §5.2–5.3](../../21-goal-completion-design.md#52-s01--더미-환경을-실제-실험에-배치한다)이 소유한다.

| 항목 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| S01 | NO | NO | NO | NO | 최소/모의 개발환경의 독립 fresh 실험, 검증된 dummy 배치, 조건부 행위·정상 대조, 전 생태계 연결 |
| S08 | NO | NO | NO | NO | 실제 writable roots의 전후 내용·mode 변화, 중간 create/delete 관찰, privacy·범위·결과·판정 연결 |

## 기존 구현과 재사용 경계

| 현재 owner | 구현·연결된 사실 | 남은 실제 연결 |
| --- | --- | --- |
| `sandbox/authority.go`, `boundary_helper.go` | Host env 복제 없이 HOME/PATH/LANG/LC_ALL 고정, 권한 강하·고정 helper 준비 | 공통 환경 profile 데이터와 controller-owned dummy 검증. 기존 helper를 범용 관리자/완료 authority로 바꾸지 않음 |
| `Backend.Execute`, `pnpm_instance_backend.go`, `bun_instance_backend.go`, `npm_project_inspection.go`, `node_execution.go` | npm 계열 lifecycle/현재 instance/선택 실행의 기존 격리·관찰·종료 | 각 기존 진입점의 최소/모의 환경과 전체 workspace 수명. 특정 경로 하나만 연결해 전체로 표시하지 않음 |
| `pypi_trusted_transaction.go`, `pypi_observation_transaction.go` | frozen unit·preparation/anchor/관찰 phase·외부 CPU/event/byte/launch/wall 원장 | 두 환경을 기존 root 총량 안에서 명시적으로 계상. runtime마다 새 root 한도를 부여해 총 예산을 곱하지 않음 |
| `gomodule_build.go`, `cargo_observed_build.go`, `inspection/projectbuild/build.go` | 관찰 build·입출력/graph binding·owned volume 종료 | build 작업의 dummy/diff. download/fetch를 build/runtime 실행으로 표시하지 않음 |
| `github_elf.go`, `terraform_provider.go` | 공통 제한 ELF/probe mechanics와 source별 판정 | 기존 probe의 환경·상태 변화. provider RPC·임의 argv 실행 추가 없음 |
| `pypi_closure_volume.go`, `pypi_closure_preparation.go`, `pypi_closure_installation.go` | owned bounded volume, attachment/RO 검증, reader 선연결→writer 종료, TAR stream의 bounded 파일 검증 | 관찰 workspace의 before/after inventory. 설치 closure census를 일반 state diff로 바꾸어 부르지 않음 |
| `observer.cc`, `pod-init.json`, `collector.go` | process/open-result/network 및 mount topology, 두 exact honeytoken 경로 분류 | 실제 dummy 배치와 경로/역할 결합, write/rename/unlink/mode의 실제 지원·성공 의미 확인. mount topology는 파일 내용 snapshot이 아님 |

`cmd`의 기존 CLI/정책 helper/installer와 Host resource snapshot에는 범용 guest filesystem
diff 제공자가 없다. 그러나 새 범용 helper·언어별 scanner를 선행 생성하지 않는다.
먼저 기존 named volume/readonly anchor와 bounded TAR reader를 현재 owner에서 재사용한다.

## 먼저 확인할 수명 경계

현재 일반 backend의 `/tmp`는 container-private tmpfs이며 완료 trace를 위해 container를
폐기한 뒤에는 같은 경로를 단순 조회해 final snapshot을 만들 수 없다. 과거의 actual DATA
시험은 마지막 attachment가 사라진 tmpfs의 내용 보존 가정을 반증했다. 관련 원본과 한정
교훈은 `docs/lessons/sandbox.md` L18의 P5-C3c2a/P5-C3b가 소유한다.

따라서 첫 작은 actual 검증은 inert DATA만 사용해 owned workspace/readonly keeper의
attachment 전환과 기존 streaming TAR의 읽기를 확인한다. 기존 mount·noexec·권한·용량·
identity를 함께 대조하며 zero-attachment나 runtime tmpfs의 Docker cp를 성공했다고 가정하지
않는다. 지원되지 않으면 같은 실패를 반복하거나 일반 CONTROL/파일 권한을 넓히지 않는다.

## 이 단위의 구현 순서

1. 위 owned workspace·reader·writer 종료/관찰·폐기 primitive를 먼저 확정한다.
2. 같은 sandbox 모듈의 한 profile 데이터 owner에서 synthetic 파일/env를 만들고 실제
   배치 bytes/mode/hash를 확인한다. Host 파일·secret·loader/PATH/proxy/CA 복제는 없다.
3. fresh 실험마다 baseline→Artifact 실행→모든 descendant 종료/observer drain→final
   inventory를 연결한다. 기존 TAR/hash/path guard와 resource policy를 재사용한다.
4. snapshot에 남지 않는 create-then-delete는 같은 observer/pod-init owner에서 필요한
   실제 event를 검증해 연결한다. open/write 시도를 write 성공으로 표시하지 않는다.
5. S09의 독립 선언/profile·실제 범위·Evidence와 같은 결과에 연결하고, 정상/조건부/준비
   실패·불완전·취소와 각 기존 생태계 진입점을 닫는다. 원래 한도를 늘리거나 검사를 생략하지 않는다.

실제 namespace/mount/observer 변경 전 해당 경계의 작은 시험부터 수행하고 의미 있는
변경 묶음으로 필수 native/CPP/hosted 검증을 모은다. 현재 S01/S08 실행·제품 코드·새 CI는 없다.
[C01 전체의 잔여 수명·quota acceptance](./storage-lifetime-baseline-20261102.md)는
설계16의 통합 종료까지 계속 OPEN이며 이 baseline으로 완료 처리하지 않는다.
