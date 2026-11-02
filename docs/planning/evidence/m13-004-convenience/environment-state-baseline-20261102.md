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
변경 묶음으로 필수 native/CPP/hosted 검증을 모은다. 전체 S01/S08 Artifact 실행 연결과 새 CI는 아직 없다.
[C01 전체의 잔여 수명·quota acceptance](./storage-lifetime-baseline-20261102.md)는
설계16의 통합 종료까지 계속 OPEN이며 이 baseline으로 완료 처리하지 않는다.


## 2026-11-02 공통 profile·bounded DATA inventory — 부분 구현

`environment.go`가 최소/모의 개발환경의 synthetic env·설정 파일과 기존 exact canary의
역할을 한 곳에서 소유한다. Host 파일/env를 입력으로 읽지 않으며 fresh session마다 무효
marker와 profile identity가 다르다. `environment_state.go`는 기존 Python 설치 verifier의
streaming Docker DATA transport를 재사용하고 내용·종류·mode·소유자·size·digest를
임시 inventory로 비교한다. 원래 closure manifest 검사는 유지한다. 상대 링크를 따라가지
않고 escape/특수 파일/중복·ancestor 충돌/entry·byte·transport 한도/불완전 producer는
snapshot 전체 실패다. 비교는 선형이고 canonical inventory digest만 정렬 비용이 든다.

[부분 검증 원장](./environment-state-local-20261102.json)이 frozen source별 원본 receipt를
소유한다. 첫 DATA16.810s와 공통 profile/reader 연결 후13.773s는 runc·고정 image 도구만
사용했다. 후자는 실제 배치의 bytes/mode/UID/GID, 생성·수정·삭제, readonly keeper 선연결과
writer 제거 뒤 상태 보존을 확인했다. 첫 시험의 exact 소유 volume/container 부재도 별도
읽기 전용 query로 확인했다. Artifact 코드·외부 네트워크 실행은0이고 gVisor/observer는
NOT_TESTED다. 전체 sandbox Race35.371s PASS는 최초 공통 구현 범위이며 뒤의 legacy
TAR 상수 교정과 producer-failure 회귀는 별도 current receipt로 구분한다. 최초 Staticcheck
8.355s의 U1000/SA1019 실패는 보존한다. Current 전송/경계 표적11.935s·고정
Staticcheck18.725s·vet1.736s는 PASS다.

이는 다음 runtime 연결에 사용할 현재 구현 단위다. 두 환경을 실제 Artifact 검사에
연결하거나 S01/S08 네 상태를 YES로 올린 것이 아니다. 위 순서의2–5와 전체 acceptance는
계속 남는다. 새 helper/binary/framework·Host 설치 변경·원격 CI는 없다.


## 2026-11-02 gVisor DATA와 noexec 경계 — 현재 후보 미승인

기존 installed helper와 새 Go topology의 고정 DATA4.425s, 이어 suspicious 사실 부재까지
확인한13.522s는 PASS였다. 후자는16 expected process events/1824 bytes, session_complete를
확인했고 C++·Host 설치는 바꾸지 않았다. 그러나 이는 guest 실행 금지와 별도다.

`/tmp` 전체를 named volume/guest9p로 교체하는 후보는 backing tmpfs noexec를 실제 guest
noexec로 전용할 수 없다. 고정 ELF 제어의 도입 실패·139(SIGSEGV)는 최초 원본으로
보존하며 해당 시험으로 차단 성공을 주장하지 않는다. 단순 shebang 제어의 실제
`DATA_EXEC_EXIT=99`(13.303s, test FAIL)는 직접 실행이 가능함을 확인했다. 기대한126과
다르므로 후보는 **반증/제품 연결 금지**다. 기존 제품의 private `/tmp` tmpfs·권한·버전
입력은 바꾸지 않았고, 제품 Artifact 실행에는 연결하지 않았다.

다음은 같은 source의 실패 반복이나 권한 확대가 아니라 guest mount flags를 실제로
보존하는 지원 경로 조사다. 전후 DATA 읽기와 noexec를 모두 만족하기 전에는 두 fresh
Artifact 실험 연결·qualification을 진행하지 않는다. S01/S08의 네 상태는 계속 NO다.


### 최소 보완 경로와 기존 업데이트 owner

고정 upstream `7c619980`의 `runsc/boot/vfs.go:ParseMountOptions`는 OCI mount options에서
NoExec/NoSUID를 만들며, Docker named-volume backing 옵션이 이 guest options로 전용되지
않는다. 현재 제품 `/tmp` private tmpfs의 실제 guest flags를 바꾸지 않는다.

보완은 기존 pinned HAA gVisor patch 안의 좁은 **추가 제한**으로 검토한다. Controller가
요청한 versioned workspace policy만 고정 `/tmp`의 guest noexec/nosuid를 강제하고 알 수
없는 policy는 거부한다. 다른 mount와 executable child mount는 바꾸지 않으며 새 capability,
privileged manager, Artifact 선택 mount 또는 observer grant를 추가하지 않는다. Go observer는
실제 guest 두 flags를 필수로 요구해 구형/불일치 runtime에서 준비를 거부하도록 먼저 고쳤다.

새 runtime 후보는 engineering05의 기존 exact source/patch·Go owner·Bazel/builder·전체 bundle
capture/strict verifier·runtime generator를 그대로 사용한다. 기존 SDK·이미지 버전을 새로
선택하거나 다른 updater를 만들지 않는다. Patch hash와 실제 산출물 identity는 검증된
후보에서만 갱신하며 기존 installed runtime이나 과거 qualification을 성공으로 승계하지
않는다. 이 수정 전후의 작은 runtime 회귀와 실제 flags/직접 실행 거부를 먼저 닫는다.


## 2026-11-02 고정 runtime의 guest 제한 — 부분 경계 검증

기존 source/Go/Bazel/builder로9개 named upstream 회귀19.001s와 같은 캐시의 완전한
six-member bundle88.444s가 PASS다. 기존 `capture-gvisor`·strict verifier·compiler/trace
capability 검사를 재사용했다. 관측 member 지문을 canonical lock에 반영하고 기존
`generate-runtime-lock.go`로 파생 코드를 생성했다. Source commit·선택 Go1.27.3·이미지·
Bazel/builder 버전은 바뀌지 않았다. Node/Python 등의 버전 owner도 그대로다.

별도 PID/mount namespace의 고정 static DATA 대조1.696s는 정책이 없을 때 guest flags가
false이고 shebang script가99로 실행됨을 확인했다. 같은 입력에 정책을 넣으면 실제 guest
noexec/nosuid가 true이며 `execve`가 EACCES로 거부됐다. 임의 Artifact·외부 연결·설치된
runtime/Docker 설정 변경은0이다. 이 결과는 직접 OCI runtime 경계에 한정하며 새 bundle의
Docker/production observer·두 fresh Artifact 실험·합산 CPU/wall/event 원장 검증이 아니다.
Namespace 종료와 빈 runtime 상태 조회 뒤 보호된 임시 사본을 제거했고 고정 시험 binary
역시 지문 대조 후 제거했다. 개발 tool cache와 후속 소비자에 필요한 검증 bundle은 유지한다.

최초 upstream launcher는 성공했으나 XML의 named census가 비어 operator가 실패했다.
캐시를 재사용한 verbose 결과로9개를 확인했다. Direct OCI의 최초 정리 오류와 준비 단계
권한 오류 뒤의 의도치 않은 동일 operator 재실행은 숨기지 않는다. V1의 최초 원래 case
결과는 미확정이며 덮인 receipt에서 복구하지 못한 owned-root path도 NOT_KNOWN이다.
[부분 원장](./environment-state-local-20261102.json)과 별도 raw interpretation이 근거를
소유한다. V2는 최초 결과와 cleanup을 독립 보존한 뒤 PASS다.

S01/S08 전체의 네 상태는 계속 NO다. 다음은 기존 root 자원/완료 원장을 사용한 두 fresh
Artifact 실행 연결과 전 생태계·파일 변화·단기 생성/삭제·privacy/Policy acceptance다.
CI는 시작하지 않았고 설치 환경은 변경하지 않았다.


현재 묶음 foundation의 최초83.085s는 compile 단계들 뒤 기존 C01 파생 writer의
`sandbox → Core/Ports` 검사표 누락에서 실패했다. Architecture-001 방향을 유지하고
실제 opaque 계약만 등록했으며 concrete storage/bootstrap 거부도 같은 기존 회귀에
포함했다. 해당1.596s와 수정된 canonical foundation25.164s가 PASS다. Runtime 유지보수
3 package37.155s·보안34.484s도 PASS다. S01/S08 전체 구현·Docker/observer/Artifact
qualification은 이 검사들로 완료 처리하지 않는다.


## 다음 구현 단위 — npm 두 fresh 실행의 root 예산

기준 `bdb488c`. S01/S08 전체 상태는 위 네 NO를 유지한다. Npm 두 환경 실행의
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED도 NO다. 기존 `Backend.Execute`의
one-shot lifecycle과 소개/관찰/종료는 구현돼 있으므로 다른 backend로 대체하지 않는다.
공통 profile·volume/TAR/diff와 기존 `observationTransaction`, `observationCPUWatch`,
`ObservationResourceClient`를 재사용해 필요한 두 실행만 연결한다.

현재 privileged helper의 root budget profile은 PyPI/PyTorch만 허용한다. 최초 MISSING은
npm의 고정 profile을 같은 CPU30초 authorization에 연결하는 것이다. 두 환경 전체의
writer+readonly keeper4개가 root launch ceiling이며 실행마다 CPU/wall/event 한도를
새로 부여하지 않는다. Npm 기존45초 wall·메모리·tmpfs·관찰 예산은 유지한다. Source/
image/guest argv를 Artifact가 고르거나 raw manager로 우회하는 옵션은 없다.

그 뒤 기존 factory/Bootstrap에 같은 typed resource adapter를 전달하고 profile 배치 검증,
baseline→Artifact 실행→전체 tree 종료/trace drain→final inventory를 묶는다. 준비/관찰/
정리 실패와 취소는 complete를 만들지 않는다. 현재는 이 단위의 baseline이며 새 실행이
이미 제품에 연결됐다는 의미가 아니다.


Npm 연결 중 실제 고정 lifecycle의 `HOME=/tmp`를 확인했다. 모의 설정을 무시되는
`/tmp/home`에 두지 않고 기존 HOME의 run 전용 root에 배치하며 trusted argv/CPP 권한은
바꾸지 않는다. 합산 실행 연결 뒤에도 typed 상태 digest/범위와 expected profile·기존
one-shot 승인 재사용 거부를 닫아야 한다. 현재 제한된 변화 수를 전체 S08 증거 완료로
표시하지 않는다. 다른 기존 생태계 진입점도 별도로 남는다.


## 2026-11-03 npm 기존 lifecycle의 두 환경 연결 — 부분 검증

기준 `bdb488c`에서 기존 `Backend.Execute`와 소개/관찰/종료 owner를 확장했다. 같은
root authorization에 writer/readonly keeper 총4개를 등록하고 두 fresh 환경을 실행한다.
CPU30초·기존45초 wall·메모리/tmpfs·공유 trace 한도는 재부여하지 않는다. 실행 전 profile
배치를 검증하고, writer 전체 종료/trace drain 뒤 readonly final inventory를 비교한다.
등록 실패 뒤 이미 시작된 trace도 기존 once-only 수집기로 회수한다. 실행의 최초 code와
cause를 cleanup 오류가 덮지 않는다. 새 `npm-dynamic-lifecycle-environments-v2` 선언과
기존 CheckPlan 검증은 과거 one-shot 승인을 거부한다. 다른 project 검사 profile을 새
검사에 완료한 것으로 바꾸지 않는다.

이 npm 연결의 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO다. 한정 actual3/31.307s는 실제 Docker·새 pinned gVisor·현재
observer/helper에서 기본 설치, Host synthetic 값 미유입, 최소/모의 조건 및 fresh 파일
분리를 통과했다. 기존 설치 bytes/inode·Docker 자원·helper process 상태를 복원했고,
큰 시험 사본370078511B를 제거했다. ABI가 같은 기존 observer는 canonical4 source hash가
동일함을 확인해 재사용했다. 새 CPP/full 빌드나 CI를 중복 실행하지 않았다.

최초 timeout fallback panic과 수정된 표적 회귀, socket 제약으로 실패한 첫 Race 및
sandbox만의 정상 권한 재검증, 자원 생성 전 중단된 operator 종료값 오류를 원본 그대로
구분한다. [단일 부분 원장](./environment-state-local-20261102.json)의
`npm_environment_checkpoint`가 입력과 각 raw receipt를 연결한다.

MISSING은 typed 상태 digest/범위의 Evidence 연결, 실행 중 단기 파일 변화 관찰,
다른 생태계의 기존 실행 consumer와 대응 actual, 묶음 최종 qualification이다.
S01/S08 전체 네 상태는 여전히 NO이며 C01 전체와 B01 호환 gap도 OPEN이다.

이 연결 묶음의 canonical foundation70.578s·security24.750s와 영향4 package vet4.769s가
PASS다. 실제 실행 후 exact hash/신원 확인으로 원래 시험용 compiler 출력38421834B도
제거했다. 재사용할 verified SDK/bundle과 도구 cache·원본 source/log는 보존한다.
