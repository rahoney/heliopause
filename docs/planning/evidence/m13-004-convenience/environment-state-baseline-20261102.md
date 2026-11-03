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


## 다음 누락 단위 — typed 상태 비교 Evidence

기준 `243dbfb`. 기존 `environmentState`/`compareEnvironmentStates`의 digest·차이와
npm runtime 종료 순서는 구현/연결됐지만 `SandboxResult`는 observation count만 전달한다.
`normalizeDynamicResult`는 기존1024B의 observation summary 하나만 저장하므로 상태
비교 증거의 IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다.

MISSING만 확장한다. 기존 SandboxResult에 최대 두 typed profile·fixture/before/after
SHA256·변화 수를 전달하고, 기존 Inspection의 같은 Artifact/CheckID에 환경별 Evidence를
추가한다. 기존 observation summary와 Evidence당1024B 한도는 유지한다. 원시 파일명/내용/
Host 경로는 저장하지 않으며 snapshot 사이의 생성 후 삭제는 아직 관찰하지 못한다는
범위를 명시한다. 현재 npm required profile은 상태 증거 부재도 거부하고 과거 profile의
승인 재사용은 기존 CheckPlan이 막는다. 다른 검사 결과의 계약·새 저장소/스키마 framework를
만들지 않는다. 실행 중 syscall coverage와 다른 생태계 연결은 이 단위 뒤에도 남는다.


### typed 상태 비교의 로컬 연결 결과

위 누락 단위의 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO다. 기존 SandboxResult에 두 공통 profile과 opaque digest·변화 수를
값으로 전달하고, Inspection이 같은 Artifact·CheckID에 별도 Evidence 두 개를 보존한다.
관찰 summary와 각 Evidence의1024B 한도는 그대로다. 현재 비교 필드는 file type/mode/
owner/size/content digest이며 실행 중 단기 변화는 `NOT_COVERED`로 명시한다.

필수 상태 증거가0/1개면 완료가 아니며 잘못된 profile/hash/count·불일치·중복은 거부한다.
입력/반환 slice 사본을 통해 결과를 바꿀 수 없다. 새 required profile은 image·gVisor patch·
공통 profile/schema version에 묶여 기존 one-shot와 count-only 승인을 거부한다. 다른 언어의
기존 observation 결과를 새 환경 검증이 완료됐다고 바꾸지 않는다.

표적13.121s, 영향5 package Race50.882s, Staticcheck24.263s와 canonical foundation70.534s가
PASS다. 원본·입력·범위는 같은 부분 원장의 `typed_state_checkpoint`가 소유한다. 새 typed
Evidence의 persisted actual은 다음 의미 있는 native 묶음에 모으며 앞3 native 성공을
이 변경의 성공으로 재사용하지 않는다. 다음은 기존 observer의 단기 filesystem event
지원 대조와 필요한 coverage 연결이며 다른 생태계·최종 qualification도 계속 OPEN이다.


## 단기 filesystem event baseline — 조사 중

기준 `6bac0e8`. 이 누락의 IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO다. 기존 성공 OPEN_RESULT와 mount/provenance·privacy 경계는
재사용하며 다시 구현하지 않는다. 현재 pod-init은 write/rename/unlink/chmod 성공 point를
요청하지 않고, C++ Handle 및 Go collector vocabulary에도 이 성공 종류가 없다. 기존
OPEN/OPEN_RESULT access 수를 실제 write·create·delete 완료로 바꿀 수 없다. Pinned
upstream `syscall.Write`는 exit/fd/path/count를 갖지만 현재 필요로 하는 sealed mount
identity가 없고, 현재 OpenResult도 socket FD table을 파일 object 권위로 만들지 않는다.

수정 후보는 기존 VFS의 성공 반환 지점과 actual Mount identity다. 읽은 경로는
`pkg/sentry/vfs/vfs.go`의 UnlinkAt/SetStatAt/RenameAt, `file_description.go`의 Write/PWrite,
기존 inotify와 gofer의 create/unlink/rename 처리다. Inotify의 watcher 객체만으로는
현재 Mount를 증명하지 않으므로 이를 곧바로 authority로 사용하지 않는다. 이 baseline 시점에는 kernel
point·C++/Go parser·profile 및 SDK를 수정하지 않았고 새 방법의 실행을 검증하지 않았다.

MISSING은 성공한 create/write/rename/unlink/permission operation의 정확한 kernel 정보와
기존 sealed workspace의 결합, raw 비보존·한도·누락 거부, controller/setup과 Artifact 관찰
범위 구분, create-then-delete/실패 동작/정상 대조 actual이다. 기존 공용 유지보수 절차와
자원 한도를 유지하고 필요한 observer/runtime 변경이 모인 뒤 검증한다.


### 2026-11-03 단기 filesystem 관찰 후보 — 격리 빌드 검증

기준 `6bac0e8`에서 기존 pinned upstream·VFS 성공 반환·remote sink를 재사용한다.
O_CREAT 요청 자체를 생성 성공으로 세지 않고 gofer의 실제 새 파일 생성, Write/PWrite의
양수 반환, 성공한 rename/unlink와 mode/owner/size 변경을 actual mount namespace/ID에
결합한다. O_TRUNC의 성공한 regular-file 크기 변경도 포함한다. 경로·FD 추측·파일 내용은
wire나 결과에 추가하지 않는다. mmap store 전체, 파일별 create/delete 짝 또는 Artifact만의
행위라는 주장은 하지 않으며 세션의 setup/installer 동작도 포함한 범위를 명시한다.

기존 topology snapshot은 실행 전 필수 mutation point와 container/group/start context의
활성 여부를 증명한다. C++은 그 증명이 없으면 seal/readiness 전에 거부하며 기존 helper
identity 검사도 새 ABI를 요구한다. 새 완료 channel/daemon/권한은 없다. 정상 workspace
동작은 중립 관찰이고 위반 Finding이 아니며 기존 raw/record·공유 trace 예산을 소비한다.

V2 kernel 표적4/33.658s와 C++ 새 matrix218.249s는 각각 당시 후보의 부분 증거다.
V4의 O_TRUNC/활성 증명을 포함한 **kernel7 + 전체 CPP134/254.002s PASS**, no-skip·source
불변·ABI identity를 확인했다. Go의 envelope/공유 예산/구형 helper/불완전 coverage 등
5 package 표적18.086s도 PASS다. 원본 `s08-filesystem-kernel-tests-v2`,
`s08-filesystem-observer-targeted-v1`, `s08-filesystem-kernel-observer-batch-v4`,
`s08-filesystem-go-targets`는 goal-implementation-20261101에 보존한다.

첫 kernel V1의 oldParentVD type compile 오류는 `.Mount()`로 교정했으며 성공으로 덮지
않는다. V4 patch `c8a546d074e4857eb28da198a4782c76cdebb045dec7eda7d7b95253f5a3bd68`는
canonical source preparation으로 동일 고정 SDK에 적용했다. Go/Bazel owner·버전 갱신 방식은
그대로다. coherent6 build/capture/strict verify86.563s와 main runtime 잠금 연결을 완료했다.
Persisted Evidence·다른 생태계 연결과 최종 qualification은 미완료다. 이 부분 증거를 전체 S01/S08
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED로 확대하지 않는다. 새 CI/영구 Host 변경 없음.


새 관찰 경로의 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO다. Actual V2/42.637s는 기본 설치/Host synthetic 값 미유입/fresh2/
단기6종 관찰4개와 필수 point 누락/context 누락 거부2개가 no-skip PASS였다. 누락은
Artifact 소개 전 `OBSERVATION_PROFILE_INCOMPLETE`이며 states는0개다. 단기 표본의
Artifact Node/shell은 기존 unexpected 관찰을 유지하므로 이 표본을 Policy ALLOW 성공으로
기록하지 않는다. 기존 hash/inode·Docker 자원·helperwait0을 복원하고 시험 사본368462440B를
제거했다. V1은 operator의 BUILD basename 오선택으로 자원 생성 전 NOT_RUN이며 원본을 보존한다.

같은 product 후보의 canonical foundation89.877s, 영향6 package Race47.879s와 고정
Staticcheck17.880s가 PASS다. 새 runtime은 Go1.27.3/같은 Bazel lock·중앙 owner를 유지하며
검증된 여섯 member와 patch만 교체했다. 단일 부분 원장의 `filesystem_operation_checkpoint`가
raw 입력·범위·한정 증거를 연결한다. 전체 S01/S08/C01 완료나 hosted/final 검증은 아니다.

기존 snapshot-only schema의 승인 재사용 거부 표적1.035s와 canonical security24.569s도
PASS다. Actual 후 compiler 시험 출력36621774B를 exact hash/inode로 확인해 제거했다.
SDK/verified bundle은 다음 소비자 검증의 입력이므로 보존한다. 마지막 변경은 legacy
profile 표적 fixture와 문서뿐이며 위 native/Race product source와 구분한다.


## 다음 누락 단위 — 실제 정규화·저장 Evidence readback

기준 `22927b4`. npm Backend.Execute → DynamicInspector → 기존 Application Inspect →
Local Evidence Record/ReadReference의 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO다. Domain/normalizer·저장 경로는 재구현하지 않는다. 기존
`internal/bootstrap/npm_project_integration_test.go`의 trusted runtime 구성을 재사용해
작은 고정 공개 npm 패키지 한 개를 실제 검역하고, 현재 required profile과 같은 Run/subject의
저장된 두 환경 Evidence를 읽는다. MISSING은 실제 kernel 관찰이 기록된 summary와
fixture/baseline/final digest·scope/profile의 저장 후 동일성 및 정상 Policy 결과다.
기존 all-profile/모든 생태계 qualification을 이 한 사례로 대체하지 않는다.


실제 readback 단위는 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES /
ACCEPTANCE_CLOSED=YES(아래 한정 scope)다. `is-number@7.0.0`의 기존 public resolver →
실제 두 환경/VFS 관찰 → 현재 required check → Application → Local Record/ReadReference
→ Policy ALLOW를 native1/11.13s(operator13.413s), no-skip/helperwait0로 확인했다.
저장된 같은 Run/subject/check/profile과 두 환경의 fixture/baseline/final digest·scope·
operation 범위를 검증했다. 영구 설치 bytes/inode와 Docker 상태를 복원했고 root 보호
시험 사본395388457B 및 exact compiler 출력을 제거했다. 이는 이 npm readback gap만의
완료이며 다른 consumer와 전체 S01/S08/C01·hosted/final qualification은 계속 OPEN이다.


## Python 소비자 baseline — 기존 trusted transaction 확장 전

기준 `66fbdd5`. 최소/모의 환경 및 writable state 연결의 IMPLEMENTED=NO / WIRED=NO /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다. 기존 `executeTrustedTransaction`의 인증된 RO
closure, 단위별 fresh process, optional command/prerequisite semantics와 `startPhase`/
`terminatePhase`/공유 CPU·trace·wall 원장은 구현/연결돼 있으므로 대체하지 않는다.
`inspection/pypi/dynamic.go`는 현재 observation/command 결과를 보존하지만 환경 상태는 없다.

`observationPolicyForPython`와 Host resource service의 현재 launch 상한은 동일한
`2*MaxObservationImportsPerArtifact()+4`다. 매 단위마다 새 readonly keeper를 추가하면 두
환경에서 이를 넘으므로 그 방식이나 상한 확대는 사용하지 않는다. 검토할 좁은 선택은
환경별 owned volume 하나/readonly keeper 하나에 고정 단위별 subpath를 두고 각 writer에는
자신의 subpath만 `/tmp`로 제공하는 것이다. Prep/기존 closure anchor2 + 환경 keeper2 +
두 환경의 N writer = `2N+4`로 현재 launch 상한 안이며 CPU·trace·wall은 하나다.

이 mount 의미는 아직 확인되지 않았다. 먼저 고정 image/runc의 inert DATA로 Docker
volume-subpath 분리·inspect의 실제 identity·readonly keeper 선연결과 writer 제거 후
보존을 작은 시험으로 확인한다. gVisor 권위·Python Artifact 실행 성공으로 확대하지 않는다.
지원되지 않으면 원본 실패를 남기고 기존 종료/격리·예산을 약화하지 않는 경로를 재검토한다.
추가 MISSING은 actual subpath/guest topology 검증, 두 환경의 기존 단위/optional/prerequisite
재조정과 현재 profile 선언·저장 상태 Evidence, 정상/조건부/준비·취소·불완전 회귀다.


위 inert DATA trial V1/4.389s는 runc·고정 Node image에서 단위별 파일 분리, `..`로 다른
단위 미노출, readonly keeper 선연결·writer 제거 뒤 상태 보존과 exact 자원 정리를 PASS했다.
중요한 실제 입력은 Docker `Mounts.Source`가 subpath를 표시하지 않고 원래 volume root를
그대로 반환한다는 점이다. 실제 subpath는 기존 canonical inspect template이 이미 읽는
`HostConfig.Mounts[].VolumeOptions.Subpath`에만 있다. 따라서 새 query/검증 framework를
만들지 않고 기존 typed mount 구조와 검증에 기대 subpath의 정확한 일치를 연결한다.
현재 typed 구조는 Subpath를 무시하므로 먼저 이 누락의 negative 회귀를 확인한다.
원본은 `s01-python-subpath-data-v1.json`이며 gVisor/Python 실행 검증으로 쓰지 않는다.


기존 root mount 검증의 Subpath 무시를 Python/Go 양쪽7.896s RED로 확인했고 기존
`mountArgument`/`verifyContainerMount`/`verifyAttachments`에서만 정확한 기대값을 검사한다.
원래 호출은 빈 subpath만 허용하며 새 경로는 현재 `/tmp`의 RW/non-executable workspace,
controller opaque unit directory로 제한한다. 조회 template·CPU/wall/launch 한도는 그대로다.
새 identity 표적의 완전 입력0.941s PASS이며 전체 Python 연결/qualification은 아직 NO다.
첫 RED/부분 GREEN7.757s 원장의 frozen 목록에 수정 test 파일이 빠졌음을 확인했다.
원본은 보존하고 같은 test를419개 입력 목록에 포함한 후 format/표적을 확인했다. 두 부분
원장의 source_unchanged를 전체 변경 입력의 증명으로 확대하지 않는다.


### ordinary Python 현재 연결 — 로컬 부분 검증

기존 `pythonTransactionExecutor`에 공통 profile/volume/inventory를 연결했고, 현재 필요한
Python 환경 수명 코드는 같은 모듈의 `pypi_environment.go`에 둔다. 새 backend·generic
workflow·완료 protocol은 없다. 기존 Python/Node argv의 `/tmp`/env 변환만 공통 함수로
재사용하며 코드 closure·command·중앙 image owner는 그대로다. 환경별 whole-volume RO
keeper 두 개, 단위별 전용 subpath와 두 최종 snapshot을 사용한다. 전체 writable 용량을
두 환경이 나눠 쓰고 물리 launch `2N+4`, CPU·wall·trace·기존 import cap은 유지한다.
원래 단위를 두 실험으로 계상하며 optional command 결과는 원래 unit ID/owner에 합친다.
Root/readonly keeper의 실제 liveness·exact subpath/attachment를 검사하고 writer의 전체
종료·once-only drain 뒤 snapshot을 읽는다. Pipx 공급 도구·별도 사용자 프로그램 실행은
현재 이 변경의 완료 범위가 아니며 후속 책임 대조를 유지한다.

Native4/42.372s(no-skip/helperwait0)는 기존 npm 대조군8.24s, Python 기본9.62s, 두 import
단위/최소·모의 profile 분리 및 정상 credential cue 읽기12.64s, 단기6종 동작과 같은 최종
snapshot9.47s를 통과했다. 같은 kernel/CPP bytes를 재사용했고 불변 CPP/full을 반복하지
않았다. 보호 시험 사본368547205B와 exact compiler 출력을 제거했으며 기존 파일 hash/
inode·Docker 상태를 복원했다. 실제 관찰 전 Doc checkpoint 두 사본만 낡았음을 발견해
입력을 맞췄고0.447+0.142s cache build의 binary bytes가 동일함을 확인했다. 첫 build는
제품 compile 범위로 보존하며 그 상태에서 native를 시작하지 않았다.

Sandbox 전체9.160s PASS를 포함한 첫 두 package20.090s는 보조 환경 증거를 배열 마지막으로
찾던 기존 test만 실패했다. Kind로 정확히 선택하는 test 교정 뒤 PyPI 전체1.242s PASS다.
최초 새 fixture의 delete 인자 typo compile4.301s와 교정 후 연결 표적7.797s도 원본을 보존한다.
실제 native 관찰량을 보조 검사에 전달한1.070s RED는 metadata/관찰의 중첩이1024B를 넘음을
재현했다. 한도 확대 없이 별도 observation Evidence와 ID/summary SHA256으로 연결한
현재 PyPI 전체1.315s는 PASS다. Finding은 metadata와 실제 관찰 양쪽을 참조한다.

ordinary Python 연결의 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO다. MISSING은 정상화·persisted Evidence의 실제 경로, no-import/
optional/prerequisite와 지원 root profile의 묶음 acceptance, 다른 생태계 consumer 및 전체
S01/S08/C01·최종 qualification이다. 새 CI·영구 Host 설치 변경은 없다.

현재 Python 변경 묶음의 canonical foundation50.461s, 영향5 package Race53.971s,
고정 Staticcheck14.279s, canonical security14.366s가 PASS다. Native4는 backend/공통
workspace 경로의 앞선 정확한 source 범위이며 뒤의 보조 결과 encoding 수정까지 native로
검증했다고 확대하지 않는다. 현재 구현/연결과 남은 actual acceptance를 독립 유지한다.


### ordinary Python 실제 결과·저장과 경계 소비자 — 한정 완료

구현 기준 `559c900`의 product bytes를 바꾸지 않고 기존 Application/정적·동적 inspector/
LocalStore를 사용했다. 고정 공개 packaging24.2의 실제 검역·두 환경·VFS 관찰·필수 profile·
같은 Run/subject의 저장 Evidence readback·Policy ALLOW와, 승인 가능한 leaf
`typing_extensions4.15.0`을 사용한 별도 보조 검사의 v2 metadata/관찰 ID·SHA256/두 상태
저장을 확인했다. 보조 검사 자체를 원래 요청 환경의 성공으로 표시하지 않으며
`NOT_ATTESTED`와 원래 선택 집합을 유지한다. Native v4는50.00s(operator52.565s) PASS다.

별도 native v3에서 metadata-only/no-import6.34s와 기존 선택적 명령48.68s·보조 입력
순서75.90s가 PASS다. 후자의 두 기존 시험은 각각 CPU/cu126/cu130/cu132에서 선정한
네 사례이며, 보조 입력 첫 실패의 소유 digest/exit37·동일 observer의 다음 CPU 완료를
확인했다. 전체 large PyTorch closure/모든 profile 조합 qualification은 아니다. v3 전체는
idna 보조 입력 거부 때문에 FAIL이며 성공한 독립 사례만 재사용한다. v4에서 이들을
반복하지 않았다. 현재 cpp/kernel/Go·언어 image 버전 owner는 그대로다.

첫 새 시험의 helper 반환형 compile 오류와 내부 reference 문법, Evidence ID/Kind 혼동은
시험 오류다. Six1.17.0의 실제 source-binding 거부(v2)는 별도 호환성 MISSING이며,
공식 Simple/wheel Python 조건의 공백·순서 차이는 DATA로 보존했다. 실제 pip report를
수집하지 않았으므로 정확한 불일치 항목·인과 해결은 미확정이다. 이를 검사 예외로
허용하지 않았다. idna3.11은 optional extra 의존성 선언이 있어 기존 leaf-only prerequisite
계약이 거부한 것이다. 공식 metadata와 기존 validator로 입력 적합성을 먼저 확인한 후
현재 leaf로 교정했고 제품의 prerequisite 제한은 바꾸지 않았다.

추가 realcorpus Staticcheck가 기존 PNPM fixture의 옛 npm helper 호출을 검출했다.
호출 인자와3 scripts×2환경의 기대6만 고쳤고, 기존 비교 부분만 native7.82s
(operator10.219s) PASS다. 하위 PNPM fixture와 Policy는 미실행이며 원본의
`no tests to run`은 제외된 하위 시험 범위다. 태그 포함 Staticcheck15.366s와 앞선
canonical foundation26.100s PASS를 구분한다. 새 검사마다 전체 CPP/native/CI를
반복하지 않았다. 보호 사본은 각 시험에서 원상 복원 후 삭제했으며, 종료된 원본
바이너리7개194027732B도 root96 process/참조0/query 오류0·hash/inode 확인 후 제거했다.

이 단위의 완료는 실제 Python normalization/저장, no-import와 선정된 optional/prerequisite/
root-profile 경계다. S01/S08 전체·나머지 생태계 및 실패/취소/보안·최종 qualification,
C01 전체와 requirements 호환 gap은 계속 OPEN이다. 원본과 source별 검증 범위는
같은 local 원장의 `ordinary_python_acceptance_checkpoint`가 소유한다.


### 다음 연결 baseline — npm/PNPM/Bun project·instance

기준 `a0637df`. 기존 instance 실행(`executeNPMInstanceWithBunRuntime`), frozen layout/
payload·source binding, 정확한 target argv·Bun runtime 공급·null network·전체 trace/정리는
IMPLEMENTED=YES / WIRED=YES다. 이전 bounded qualification은 기존 P7 원장 범위이며,
현재 보강 후보의 이 owner 전체 QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다. 두 fresh 환경/
공유 총량·readonly snapshot·상태 Evidence 연결이라는 새 gap은 네 상태 모두 NO다.

재사용 위치는 `pnpm_instance_backend.go`의 기존 실행·정리, `environment_runtime.go`의
공통 자원·종료 원장, `environment.go`의 profile·배치, `shared_observer.go`와 기존
PNPM/Bun/Node/Bun-runtime topology 생성자, `environment_state.go`의 bounded `/tmp`
reader다. 이미 구현된 resolver/layout/target command를 다시 만들거나 별도 backend/
framework로 교체하지 않는다. archive 경로의 정상 동작과 기존 권한·버전 owner를 보존한다.

`DATA s01-node-nested-data-v1`3.692s는 고정 Node image/runc·Artifact0에서 부모 `/tmp`와
중첩 project volume을 같은 reader로 읽고 writer 제거 뒤 양쪽 내용 변화를 확인했다.
작은 owned volume2개/컨테이너를 정확히 정리했고 기존 Docker 상태는 동일하다.
이는 실제 gVisor/observer/악성 탐지/Policy proof가 아니다.

MISSING: 기존 네 topology의 실제 `/tmp` 추가 제한과 project RO reader, 두 환경의 fresh
payload·원래 총 CPU/wall/trace/launch 원장, descendant 종료 후 전체 writable tree snapshot,
profile-bound 상태 Evidence·실패/취소/정리와 정상 instance/root consumer의 한정 실제 검증.
선택적 사용자 프로그램 실행을 두 번 실행하는 변경과 다른 생태계 작업은 섞지 않는다.


## Node project·instance 현재 연결 — 로컬 부분 검증

기준 `a0637df`의 기존 네 실행 경로를 수정했고 별도 backend·완료 protocol은 만들지
않았다. 같은 `environmentInspection`이 archive와 instance의 자원·관찰·종료를 소유하며
기존 frozen payload/target argv/null network/Bun 공급은 원래 owner가 계속 검증한다.
두 fresh profile마다 writer와 전체 workspace의 RO reader를 만들고, writer 종료·trace
회수 뒤 기존 bounded `/tmp` DATA reader로 중첩 project도 포함해 비교한다. 기존 root
총량·CPU/wall/trace/launch 한도를 profile마다 재부여하지 않는다. 선택적 사용자 프로그램
실행은 이 두 검사 실험과 별개로 유지한다. Instance/root의 같은 check/subject/profile에
두 bounded 상태 Evidence를 기록하고, 두 환경 coverage가 빠지면 required 완료를 거부한다.
더 이상 쓰지 않는 instance trace callback/interface/create helper와 cleanup만 제거했다.

첫 실제 묶음 `s01-node-instance-native-v1`269.273s는 **전체 FAIL**이다. 기존 archive,
npm nested3·local-root2, PNPM 공개 transitive2와 Bun의 Node/native 공급 각4 instance는
PASS다. PNPM controlled-success는 새 state collector가 원래 payload의 정상 hardlink를
거부하는 조건으로 남았고 뒤 controlled 두 사례는 미도달이었다. 같은 hardlink inventory의
직접 RED7.918s 뒤, 이미 읽은 같은 tree의 일반 파일을 mode/owner와 함께 검증해 내용으로
계상하는 좁은 교정을 했다. 실제 추출·추가 파일 read는 없고 외부/forward/cycle/특수·
symlink target과 mode/owner 불일치는 계속 거부한다. inode 공유 관계 자체의 증명은 아니다.
표적 GREEN7.910s와 Race13.271s 뒤 실제 v2/31.613s는 실패 사례 및 미도달 exit37/긴 이름
세 사례만 PASS했다. 통과한 첫 묶음 소비자나 불변 kernel/CPP를 다시 실행하지 않았다.

별도 v3/225.092s는 새 실제 저장·읽기 acceptance다. 기존 npm bin1·PNPM/Bun single
Application의 검역/기록/ALLOW/cache/반입·재설치/업데이트 흐름을 사용해 각 instance의
두 상태를 같은 Run/subject/check/profile로 읽었다. 기존 original target/user-file/no-op
inode 보존도 PASS다. Node 계열 current 연결은 IMPLEMENTED=YES / WIRED=YES이며,
whole QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다. Go/Cargo build·ELF/provider probe, 전체
실패/잠복 corpus와 grouped/hosted/final qualification은 여전히 MISSING이다.

부분 원장의 원본별 입력·범위를 따른다. 공통 foundation42.159s, 전체 sandbox16.599s/
inspection npm1.664s, 두 package Race34.849s 및 태그 Staticcheck, 마지막 보안20.177s가
한정 PASS다. 앞선 model/fixture 오류와 unused cleanup 실패는 원본을 보존했다. 모든 실제
시험은 helper wait0/Host·Docker 동일과 정확한 설정/runtime 복원을 확인했으며 큰 보호
사본은 각 operator가 정리했다. 완료된 원본 바이너리·공급9개336114856B도 전체 proc
무참조/query error0 뒤 삭제했다. 중앙 버전 정의·업데이트 방법/권한·Host 설치·CI는 그대로다.


### Go/Cargo build 연결 전 baseline — `8293b20`

| 항목 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 observed build/input/output | YES | YES | NO | NO | 현재 보강 후보의 grouped/final qualification; 이전 bounded 증거는 원래 scope로 보존 |
| 새 두 환경·workspace 상태 연결 | NO | NO | NO | NO | fresh 작업 공간, 공유 총량, 종료 후 writable roots만의 상태, 같은 build check의 Evidence |

기존 Go의 `ObservedGoBuilder.observe`/`goBuildOperation`, Cargo의 `Build`/
`cargoBuildOperation`가 입력/정적 source·cache verifier, exact offline graph, compiler
명령, output capture와 종료를 소유한다. `inspection/projectbuild/build.go`가 관찰/binding을
같은 source 대상으로 정규화한다. 이 기능은 다시 구현하지 않는다. `GoBuildRunner`의
package selector/취소 검증과 Cargo의 고정 default build 의미도 유지한다.

최소 수정 지도는 다음과 같다.

- 각 fresh profile에서 기존 immutable input의 preparation→RO writer와 overlap→preparation
  종료 순서를 재사용한다. 동일 frozen input handle을 재생하고 두 실험은 기존3분
  context/동일 trace ledger를 공유한다. profile별 root timeout/
  event budget을 새로 만들지 않는다. 동시에 활성인 Artifact compiler는 하나다.
- 기존 `/tmp`512MiB와 output/target의 용량·실행 가능 역할은 유지한다. 각 profile의
  workspace와 output은 fresh이며 이전 profile의 자원 종료·부재 확인 후 다음을 만든다.
  기존 builder의 input/graph/command/출력 검증 body를 재사용하고 임의 argv를 추가하지 않는다.
- 기존 환경 profile·bounded DATA reader·volume/trace owner를 재사용한다. 상태 reader는
  고정 Node helper와 RO workspace/output만 사용하고 큰 immutable source/cache input은
  붙이지 않는다. 따라서 mutable roots를 읽기 위해 readonly dependency tree를 매번 다시
  전송·해싱하지 않는다. Artifact writer 종료/trace 회수 뒤 final을 읽는다.
- 공통 argv 변환은 Go/Rust의 고정 compiler/offline env를 보존해야 한다. 현재 npm/Python용
  변환은 추가 env를 허용하지 않으므로 그대로 적용하면 안 된다. 기존 generic env 구간과
  current builder가 소유한 fixed env를 구분하며 Host env 복제는 없다. Typed topology에서
  input RO/output RW 역할과 reader RO를 각각 증명한다.
- 두 실험 모두 성공·정리된 경우에만 원래 최소 환경의 검증된 build output을 반환한다.
  두 번째 output은 검증 후 폐기하며 비결정적 정상 build에 byte 일치를 강요하지 않는다.
  recipe/check profile은 두 환경·고정 reader image/관찰·상태 계약을 중앙 runtime 값으로
  결합한다. 선택적 사용자 실행·이미 만들어진 결과의 임의 재실행은 추가하지 않는다.

먼저 공통 args/topology와 상태/원장 전제의 싼 표적 검사를 수행하고 Go owner부터 실제
연결한다. 새로운 범용 backend/제네릭 실행 callback/manager·관찰 권한 예외를 선행
생성하지 않는다. 고정 kernel/CPP는 실제 실패나 ABI 변경이 필요하지 않으면 재사용한다.

검토 중 기존 L18의 tmpfs zero-attachment 반증을 대조해 input volume을 두 profile 사이에
단순 보존하는 가정은 제외했다. 별도 immutable keeper를 늘리지 않고 기존 input 전달/
readback을 fresh profile마다 수행하며 preparation 종료는 writer가 붙은 뒤다. State reader는
그 뒤 연결해 동시에 활성인 컨테이너를 원래 preparation+build의2개 안에 유지한다.
Output은 writer 전체 종료 뒤 RO reader에서 캡처하고 검증·정리된 첫 결과만 반입 후보로
유지한다. 이는 수정 지도 교정이며 새 native/보안 권한 검증 결과는 아니다.


### Go/Cargo 현재 후보와 최초 실패 — 실제 연결 미검증

기존 build body를 각 fresh 실험에서 재사용하고 같은3분 context/trace 총량을 공유하도록
연결했다. Node DATA keeper의 RO output에서 writer 종료 후 상태/출력을 읽고, 보조 실험
output은 기존 parser로 검증만 해 두 번째 intake 사본/Run 경로 충돌을 만들지 않는다.
필수 두 환경/VFS coverage 누락3종 거부와 check/subject/profile Evidence 표적8.235s,
foundation46.665s·태그 Staticcheck18.618s·vet4.373s·보안16.693s·영향3 package
Race36.167s는 PASS다. 현재 소스 연결 IMPLEMENTED=YES / WIRED=YES이고 QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO다. 원본·source 범위는 같은 부분 원장의 candidate checkpoint를 따른다.

첫 실제 Go CLI v1/18.362s는 PREPARATION 실패로 중단했고 Cargo는 미도달이다.
CLI의 test-only unwrap v2/17.782s는 nil이어서 그 코드는 제거했다. 기존 guarded 직접
Build output 소비자를 추가한 v3/17.352s가 `python closure volume identity already exists`를
보존했다. 기존 volume 이름은 destination/manifest가 아니라 transaction에서 파생되는데
새 연결이 input/workspace에 같은 실험 ID를 전달했다. Workspace에는 기존 상위 Sandbox
결과 owner를 사용하고 input에는 fresh 실험 ID를 유지하는 최소 교정의 실제 검증이 다음이다.
전체 이름 규칙이나 존재/부재 guard·권한을 바꾸지 않는다. FirstDiagnostic 기본 zero는
전체 trace 수가 아니며 이 실패를 Artifact 검사/반입 성공으로 기록하지 않는다.


Go/Cargo identity 최소 교정의 실제 v4/173.009s는 direct Go output52.61s, Go CLI의
두 독립 작업/동일 check 상태 기록 읽기95.14s, Cargo dependency-free 기존 CLI19.25s를
PASS했다. v3 대비 Go source 변경은 두 build owner의 ID 연결뿐이며 kernel/CPP는 그대로다.
표적8.233s도 PASS이고 모든 helper wait0/원래 Host·Docker 동일/정확한 복원·사본 정리다.
Cargo 최초 fixture는 ELF/receipt 존재만 검사하므로 상태 readback의 명시적 assertion은
다음 library/public fixture에서 확인한다. 다음은 library/public·기존 경계 거부와 모의 env에서만
네트워크 시도하는 무해 Cargo script 및 두 번째 실험 실패의 첫 provisional output 수거다.
첫 실패는 위 한정 범위로 교정됐으며 전체 네 상태를 완료로 올리지 않는다.


다음 actual v5/120.241s는 Go public pflag의 ENVIRONMENT_FINAL에서 EVENT_LIMIT로 끝나
후속은 미도달이다. Ledger16924에 counted summary3700을 더하면 고정20000을 넘는다.
Byte는81663/2097152이며 physical records와 logical count를 구분한다. 원본은 같은 부분
원장의 next_failure에 연결했다. 상한/관찰/두 환경을 완화하지 않고, 완료된 전체 선택 compile
뒤 package-name role query의 불필요한 dependency 재로딩을 pinned Go의 `list -find`로
생략하는 최소 후보를 검증한다. 전체 선택 compile과 output command·원래 순서는 유지한다.
고정 Go source `cmd/go/internal/list/list.go`가 -find의 IgnoreImports/package identification을
정의한다. 아직 quota 교정 성공이 아니며 표적/같은 실패 입력의 실제 검증이 다음이다.


### Go 공유 quota 변경 제안 — 사용자 결정 전 / 미구현

같은 입력 v6/118.126s도 EVENT_LIMIT(16920+3696>20000)다. `-find`는 quota 교정이
아니므로 제거했고 반복하지 않는다. 필요한 전체 compile/output/두 실험이나 관찰 사실을
빼지 않는다. 기존 계약의 Go normalized-event 총량20000은 그대로이며 다음 변경은 아직 없다.

구체 제안은 Go의 **작업 전체 공유 총량만40000**으로 versioning하는 것이다. 필수 두 fresh
실험의 preparation/compiler/readonly reader와 counted summary를 하나의 원장에 계속
합산하고 stream/profile별 초기화를 하지 않는다. Helper250000/connection, physical10000/
collector, summary saturation10000, aggregate2MiB, CPU1·memory512MiB·3분, mount/role/
network/필수 검사 경계는 모두 유지한다. Canonical Go budget leaf/보강 설계의 근거를 먼저
갱신하고 한 상수·정확 경계/교차 stream 회귀 및 Go recipe/check profile에 이 변경을 묶은 뒤
같은 pflag 실패와 미도달 matrix를 검증한다. Bare Go/Cargo v4 성공은 원래 한정 범위로 보존한다.

AGENTS의 기존 security invariant 유지와 [Go budget 고정 계약](../../16-m12-01-ecosystem-expansion-contract.md#go-build-observation-budget)을
직접 대조해 현재 명세의 수치 변경은 구현 편의로 자동 승인하지 않는다. 이 제안은 code
변경/qualification이 아니며 사용자 결정 전에는20000을 유지한다. 독립적인 Cargo matrix는
같은 기존 한도 안에서 계속할 수 있다.


### Go quota 사용자 승인 — 구현 후보 / 실제 검증 전

2026-11-03 사용자가 위 구체 공유40000 제안에 `조정해도되`라고 승인했다. Canonical Go
budget과 보강 설계에 반영하고 한 상수만 수정했다. Go recipe에는 같은 상수를 명시적으로
넣고 schema/check profile을 revision해 이전 계약의 기록과 구분한다. Cargo profile은
그대로다. 두 환경/stream의 합산 exact40000과40001 거부·2MiB 및 다른 profile/physical
한도 불변을 표적 회귀로 검증한 뒤 최초 pflag 실패와 미도달 matrix를 계속한다. 아직 actual
성공은 아니며 위 failed -find 후보는 제거된 상태다.


### Go/Cargo 한정 실제 검증 — 승인된 공유 총량 반영

사용자 승인 뒤 Go-only 공유40000과 recipe/check profile revision을 반영한
`s01-build-environment-native-v7`382.694s의11개 사례가 모두 PASS했다. 최초 pflag 실패,
Go library와 invalid testdata/mixed package, Cargo public/library/build.rs 및 filesystem
거부·모의 환경에서만 활성화되는 network·두 번째 실험 실패·invalid source를 확인했다.
Cargo 조건부 표본은 최소 환경에서 잠복하고 모의 환경에서만 네트워크 시도1회를 남겼으며,
같은 현재 검사에 결합한 두 상태 Evidence를 실제 저장소에서 읽고 반입 거부를 확인했다.
두 번째 실험의 compile 실패 때 첫 임시 output도 남지 않았다. 원래 compiler 전체 선택/
명령 순서와 다른 profile·physical/bytes/CPU/memory/wall/권한 경계는 유지했다.

승인 경계 표적12.971s와 tagged Staticcheck21.685s, quota/topology Race14.413s 및 정확한
normalizer8개 사례 Race1.398s가 PASS다. 첫 final Race의 projectbuild는 잘못된 이름 필터로
미실행이었으므로 별도 exact test만 실행했다. 초기 후보의 foundation/vet/security/full
영향 Race는 원래 source 범위로 보존하고 최신 전체 후보 성공으로 확대하지 않는다.
실제11개 사례는 모두 helper wait0/정확한 runtime·설정 복원/Host 동일을 확인했다. Operator
사본395555292B와 완료된 원본 바이너리2개63714626B를 무참조/query error0 확인 뒤 제거했다.
SDK/cache/source와 원본 증거는 보존했다. 부분 원장의 SHA와 source 목록이 검증 범위를 소유한다.

Go/Cargo 현재 연결 IMPLEMENTED=YES / WIRED=YES이며 위 한정 로컬 소비자는 검증됐다.
전체 QUALIFIED=NO / ACCEPTANCE_CLOSED=NO를 유지한다. 다음 MISSING은 ELF/Terraform
제한 probe의 같은 환경·상태 연결이며, 전체 S01/S08/C01와 grouped/hosted/final qualification,
다른 보강 항목을 계속한다. 새 CI·push·Host 설치는 실행하지 않았다.


### GitHub ELF/Terraform 제한 probe 연결 전 baseline — `72a7a02`

| 항목 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 ELF introduction/고정 handoff/provider terminal | YES | YES | NO | NO | 새 보강 후보의 묶음 qualification; 기존 bounded 동작은 재사용 |
| probe 두 환경·종료 후 상태·공유 총량 | NO | NO | NO | NO | fresh 두 실험, 기존30s CPU/trace 공유, `/tmp`·`/work` 상태, 같은 required check Evidence |

`GitHubELFBackend.executeELF`가 source/profile·introduction·고정 실행·종료를,
`TerraformProviderBackend.probeTerminal`이 exact admission과 typed exit0/1을 소유한다.
현재 private tmpfs는 writer 삭제 뒤 읽을 수 없으며 두 번 실행만 추가하면 CPU/trace가
재부여된다. 기존 `environmentInspection`의 watch/ledger/owned volume/start/stop/finalize를
재사용하고 ELF introduction/terminal을 기존 owner에 유지한다. 별도 manager·완료 protocol을
만들지 않는다. 공유 자원 service에는 두 exact profile을 기존30s/4launch 계약으로 연결한다.

`/tmp`256MiB/noexec와 `/work`256MiB/exec 역할은 그대로다. 기존 owned volume의 두 fixed
destination을 추가해 `/work`는 reader의 `/tmp/haa-elf-work`에서 RO로 읽고 하나의 bounded
inventory에 포함한다. Writer 종료/drain 전 final 금지, 실제 Artifact ELF의 재해시와 두
profile 준비·canary placement를 유지한다. Canonical provider wall30s를 두 실험 전체에
적용하며 GitHub의 기존45s를 늘리지 않는다. Provider terminal은 환경마다1개/전체2개로
명시하고 arbitrary argv/RPC/cloud 작업을 추가하지 않는다. Pipx의 별도 GitHub normalizer
소비자는 이 ELF coverage 요구와 분리해 기존 동작을 보존한다.

먼저 fixed mount/공유 budget·terminal/coverage의 싼 검사 후 actual DATA/정상·거부/조건부
probe를 묶어 검증한다. 새 kernel/CPP는 현재 소비자에서 실제 필요가 확인된 경우에만
수정하며 기존 버전 owner/업데이트 경로는 유지한다. 전체 S01/S08 acceptance는 계속 OPEN이다.


ELF 공통 경로의 추가 소비자인 `HeliopauseNativeBackend`의 고정 PNPM `--version`도 같은
공유 자원 port를 composition root에서 연결한다. 별도 runtime 검증/업데이트 owner는 유지한다.
첫 표적12.681s에서 sandbox/GitHub/hosttool/bootstrap은 PASS였고 provider 두 fixture는 새
상태 Evidence2개를 기존1개로 기대해 FAIL했다. 실제 record 수로 fixture만 교정하고 필수
환경0/1/VFS coverage 누락 거부·세 exact resource profile의 동일30s/4launch 경계를 추가했다.
새 실행 성공이나 whole qualification으로 기록하지 않는다. 첫 format 준비는 sync 종료 전에
새 source가 없어 NOT_RUN이었으며 terminal sync 뒤에만 실제 format을 실행했다.


ELF 첫 actual `s01-elf-environment-native-v1`4.142s는 정상 GitHub fixture의 새 준비용
MKDIR/CONTROL/DIRECT_EXEC_ROOT·PROC open 분류에서 STREAM_FAULT였다. Artifact handoff와
provider·나머지 matrix는 미도달이다. 새 mkdir를 위한 권한/CPP 예외를 만들지 않고 같은
fixed 빈 mountpoint를 기존 `docker cp --archive` DATA transport로 준비하도록 교정한다.
아직 실제 교정 성공은 아니다. Helperwait0/Host 동일/정확한 복원·370502496B 사본 정리는
완료됐으며 처음 실패 원본을 유지한다. 앞 compile 실패는 제거한 생성 함수의 test 참조
두 곳이며 그 시험의 Host env/credential/권한 범위는 현재 ELF argv 시험으로 이동했다.
Production에서 더 이상 쓰지 않는 collectTrace facade도 제거하고 같은 diagnostic collector를
시험에서 직접 사용한다. Sandbox-only Race38.424s와 corrected tagged Staticcheck20.649s는
그 compile 교정 범위의 PASS이며 actual 성공을 대신하지 않는다.


Actual v2/44.166s의 정상2·거부3·공개 provider2가 모두 PASS했다. Runtime/CPP 불변,
helperwait0/정확한 복원·Host 동일/370503871B 사본 정리다. 별도 v3/10.700s는 GitHub 실제
normalizer→Record/ReadReference/같은 check·profile의 두 상태를 PASS했고, default Terraform
CLI는 선행 resolver Acquire의 `invalid storage reservation`으로 FAIL해 뒤 시험이 미도달이다.
C01의 기존 reservation은 OperationID를 요구하는데 현재 Terraform 준비에는 CLI context의
operation이 없다. 기존 bootstrap Terraform 어댑터 하나를 init/upgrade/default init에 함께
사용해 현재 요청 ID를 전파한다. Source client/공통 workflow·storage guard를 완화하지 않는다.

같은 경로의 retained cache는 증거3개와 check당1개를 고정해 새 상태2개를 거부한다. 현재
필수 source/static/dynamic 및 두 환경의 exact5 Evidence ID를 요구하고 동적 세 record의
profile 일치도 확인한다. 기존 fixture는 이 계약의 올바른5개로 갱신한다. 이는 연결 gap의
수정 지도이며 새 실제 CLI 성공은 아직 미검증이다. 첫 실패/정리와 GitHub 성공 범위는
원장 및 native-v3 원본을 그대로 유지한다.


CLI v4/29.606s는 새로운 작업 ID·exact5 cache 검증 후 두 provider의 필수3검사/각 상태2개와
entry/set ALLOW·반입 COMPLETED까지 도달했으나 outer storage Finish가 선택용 intake2개의
last-use 미기록으로 FAIL했다. Whole command 완료/성공으로 기록하지 않는다. 기존
freezePackage는 정확한 bundle을 독립 bytes로 복제해 보존하므로, bounded ReadIntake/정적
package/lock 검사와 freeze가 모두 끝난 해당 DATA Run에만 기존 InspectionStorage 종료
기록을 연결한다. 후속 별도 acquired Run의 서명·동적 검사/Policy는 그대로 필수이고 outer
guard/공유 consumer 종료 전 삭제하지 않는다. 기존 storage Finish를 무시하거나 모든 Run을
일괄 완료 처리하지 않는다. 같은 두-provider fixture에 실제 유한 storage와 Finish/사본 부재,
원래 frozen source 재생을 연결해 검사한다. Host 동일/정확한 복원·395592651B 사본 정리다.


### ELF/provider 연결의 한정 로컬 종료

Native-v5/65.670s는 default 전체 provider2개의 최초 설치와 재설치55.61s, 현재5개씩의
같은 source/Run/check/profile 상태 record readback, 필수3검사·entry/set ALLOW·cache·원자
반영 및 원래 lock identity/mode 보존을 PASS했다. Checksum mismatch/remote module/foreign
cache/동시 변경 거부4개7.60s도 PASS다. 실패한 선택 입력은 보호하며 이를 자동 정리 전체
완료로 확대하지 않는다. C01의 불확실한 실패·crash/lease/cache/retention은 별도 MISSING이다.

현재 ELF/provider 연결 IMPLEMENTED=YES / WIRED=YES이고 위 local 소비자는 검증됐다.
Whole QUALIFIED=NO / ACCEPTANCE_CLOSED=NO를 유지한다. 이전 v2의7개 probe와 v3의
GitHub stored Evidence 성공을 재사용했고 CPP/kernel·해결된 소비자를 재실행하지 않았다.
처음 compile/fixture/MKDIR/operation/last-use 실패는 원본 그대로 남겼다. 최신 Terraform
artifact/promotion/bootstrap의 표적 Race34.602s도 PASS다. Native PNPM 고정 probe, 공통
실패/취소·잠복/정상 corpus 및 grouped/hosted/final은 미완료이며 다음은 S02 primitive다.

모든 실제 실행은 helperwait0/정확한 runtime·설정 복원/Host 동일을 확인했고 operator의
큰 사본은 각각 제거했다. 마지막 완료 바이너리3개93987630B도 전체95 process의 무참조와
query error0·exact hash/inode를 검증한 뒤 삭제했다. Source/SDK/cache/원본은 보존했다.
상세 source별 검사·실패·최종 범위와 SHA는 같은 부분 원장이 소유한다. 새 CI/push/Host
설치 변경이나 전체 보강 완료는 아니다.


마지막 code 후보의 canonical foundation52.328s/security23.725s와 영향7 package의 tagged
Staticcheck27.494s·vet15.237s도 PASS다. Code는 native-v5 이후 불변이며 뒤 변경은 이
검증의 기록뿐이다. 추가 full native/CPP/CI를 만들지 않고 같은 M13-004의 다음 missing을
계속한다.


### S02 연결 전 baseline — `db32813`

| 항목 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| network-none·외부 observer·새 환경 원장 | YES | YES | NO | NO | 기존 한정 증거의 grouped/final qualification |
| 제한 fake DNS/HTTP·실제 후속 자극 | NO | NO | NO | NO | pinned guest loopback 가능성, 별도 responder 역할/변조 방지·유한 응답, 현재 관찰/Policy·cleanup 연결 |

[보강 설계 §5.4](../../21-goal-completion-design.md#54-s02--실제-외부-연결-없이-제한된-모의-응답을-제공한다),
D-008, Architecture-005와 trusted-tooling D-012/D-015를 대조했다. 기존 resolver 허용망과
Host service는 이 기능이 아니며 재사용해 인터넷을 열지 않는다. 현재 CPP는 CONNECT/
SOCKET 사실을 분류하지만 모의 responder의 독립 역할·DNS/HTTP chain 증명은 없다.
Process 이름/Artifact 출력으로 trusted role·완료를 만들지 않는다.

먼저 새 제품 API/서비스·권한 예외 없이 고정 Node image/기존 network-none·boundary와
observer만으로 inert DNS→HTTP loopback DATA primitive를 한 번 확인한다. Host listener/
외부 DNS/upstream·credential·Artifact 입력은0이다. 이 시험의 controller 소유 Node가
서버와 클라이언트를 모두 수행하는 사실은 별도 Artifact/responder 역할 분리나 악성 탐지,
S02 구현/완료를 증명하지 않는다. 성공해도 네 상태는 NO를 유지한다. 실제 runtime/
관찰 제약을 확인한 뒤 현재 두 환경·공유 resource owner에 필요한 최소 연결을 정한다.
기본 network-none 실험과 budget을 보존하고 fixture/profile 계약의 변경을 먼저 기록한다.


S02의 첫 compile0.757s는 좁은 CommandRunner를 Executor/RunBounded로 사용한 시험 오류로
NOT_RUN이었다. 기존 typed port를 구분한 build9.646s 뒤 actual-v1/2.018s는 CONTROL Node
직접 root의 PROC_SELF_MAPS 분류에서 STREAM_FAULT였다. DNS/HTTP는 미도달이며 transport
불가로 판단하지 않는다. Host 동일/정확한 복원/362087297B 사본 정리, helper NOT_STARTED다.
새 Control Node 권한을 넓히지 않고 기존 npm lifecycle의 고정 무해 fixture로 같은 loopback
교환을 확인한다. 기존 원장/두 fresh 환경/observer와 isolation을 재사용하며 신뢰 responder
역할의 증거로 사용하지 않는다. S02 네 상태는 계속 NO이고 실제 새 실행은 아직 미검증이다.


S02 actual-v2/6.595s는 Node Artifact/clone의 readonly `/etc/resolv.conf` 읽기에서
STREAM_FAULT였다(경로·mount FNV ba66dbef7df0a5a0 일치). 아직 교환 미도달이며 기존
차단을 완화하지 않는다. Fixed DNS 설정·별도 responder 역할은 현재 구현 gap이다.
같은 기존 Python Artifact/격리 원장에 literal IPv4·bounded UDP DNS-wire→HTTP 표본을
연결해 resolver 기본 설정 읽기와 분리한 최소 transport 확인을 계속한다. 이 성공도
Node 기본 DNS 지원이나 S02 완료를 의미하지 않는다. v2 test 원본과 source SHA를 별도로
보존했고 Host 동일/정확한 복원·370552904B 사본 정리/helperwait0를 확인했다.


S02 actual-v3/10.602s는 UDP DNS-wire 교환·TCP connect 뒤 accepted TCP fd5의 SENDTO에서
FD_STATE_UNKNOWN으로 끝났다. 기존 observer는 socket/socketpair/dup/close를 추적하지만
accept/accept4 결과를 구독·처리하지 않는다. Pinned kernel의 standard Accept Exit는 실제
반환 FD를 제공하며 peer address는 guest buffer여서 신뢰하지 않는다. 기존 NETWORK
리스너의 성공 결과만 같은 원장에 등록하고 LOCAL 권위를 만들지 않는 최소 교정을 진행한다.
미확인/LOCAL 리스너와 잘못된 결과는 차단을 유지하며 kernel patch/bundle·권한은 불변이다.
Host 동일/정확한 복원/helperwait0·370553952B 사본 정리를 확인했다. S02 네 상태는 NO다.


Observer-only C++135/54.556s와 exact profile/identity가 PASS다. 새 kernel/bundle 빌드는
하지 않았다. 같은 Python client/helper의 compiled 입력을 재사용한 actual-v4/13.290s는
두 network-none 환경에서 UDP DNS-wire→HTTP 응답과 NETWORK 사실10개를 PASS했다.
Host 동일/정확한 복원/helperwait0·370554440B 사본 정리를 확인했다. 미확인/LOCAL
listener, guest 주소 위조·재사용·한도 회귀도 보존했다. 이는 transport primitive만이며
독립 responder 역할/고정 DNS/행동 chain·S02 전체 완료는 계속 MISSING이다.


### S02 고정 응답기 구현 범위

현재 언어별 Artifact interpreter와 shell helper에는 공통 UDP/TCP server가 없다. 기존
resolver service를 guest에 노출하지 않고 현재 work item에 필요한 작은 static Go
`haa-sandbox-responder`를 추가한다. 별도 일반 서비스 framework/config/plugin은 만들지
않는다. 고정 IPv4 loopback DNS53/HTTP80, 유한 이름/고정 body·상태, 합산 요청/byte/시간
한도와 controller stdin STOP/stdout readiness만 처리한다. Artifact/Host 파일·환경·외부
DNS/upstream을 읽지 않는다. Guest의 다른 UID1001·root 소유 코드와 기존 admission/관찰/
resource/cleanup owner에 연결할 예정이며 프로토콜 단독 성공을 WIRED로 쓰지 않는다.
새 공급물의 서명 release/설치/doctor/self-update와 버전 유지보수 연결도 필수 MISSING이다.


응답기 실행은 stimulated writer의 기존 OCI init을 UID1001 응답기로 사용하고 Artifact
UID1000 및 readonly keeper는 유지한다. 새 daemon/container/권한 예외는 만들지 않는다.
고정 helper는 실행 전 root 소유0555 코드로 준비하고 stdin/stdout의 기존 Docker start
attach에 controller 채널을 연결한다. 장기 direct exec는 기존 admission gate를 점유해
Artifact 시작을 막으므로 사용하지 않는다. Stage/서명 공급물·고정 DNS·실제 role/정리
검증은 아직 MISSING이며 이 연결 전에 전체 성공으로 기록하지 않는다.


고정 응답기 프로토콜의 local Race1.696s는 PASS며 static build는3985568B다. 기존
helper tmpfs1MiB에 들어간다고 가정하지 않는다. DATA staging-v1/1.890s는 시작 준비
FAIL이고 v2/1.996s의 추가 stderr에서 첫 실제 start에 helper 경로가 없음을 확인했다
(Node image entrypoint가 이를 module로 실행해 MODULE_NOT_FOUND). Never-started
container의 cp 성공은 공급 유지 증거가 아니다. Tmpfs 수명/실제 readback을 먼저 대조하며
포트 권한·CPP·role을 바꾸지 않는다. 두 시험 모두 자기 container/volume 제거·Docker
원상태를 확인했다. 새 제품 응답기 연결 네 상태는 아직 NO다.


Prestart readback은 tmpfs에서 root만 남았고(local과 구분), 일반 local에서는399만B를
유지했지만 Config.User1001 시험이 파일 owner도1001로 만들었다. 기존 bootstrap처럼
Root로 준비하고 기존3 capability를 모두 버린 UID1001 init으로 교정한 local-v2/2.306s만
root0555 readback·DNS53/HTTP80·Artifact1000의 파일 write/chmod·signal 거부·정상 STOP
종료가 PASS다. 둘 모두 실제 egress 없이 자기 container/volume을 제거했다. 제품 연결은
고정 code-only local volume을 새 consumer에 readonly로만 붙이는 방식으로 좁힌다.
현재 constructor는 기존 protected-current reader·native TAR transport/동일 readback을
재사용한다. Third-party size/hash pin은 그대로 두고 HAA component의 GoVersion은 기존
runtimeidentity의 compiler owner를 따른다. 실제 role/완료 원장·환경 공간 합산 budget,
release 공급물/서명 설치·doctor/self-update와 final qualification은 아직 MISSING이다.


Current supply-data-v1/0.269s는 factory 전 NewSystem에서 이전 Host bundle identity 불일치로
NOT_REACHED였다. 같은 검증된 bundle을 child의 private mount namespace에만 공급한
v2/3.586s는 실제 protected-current reader/고정 compiler·static metadata·root0555
TAR readback, readonly 소비자·부착 중 삭제 거부와 전체 DATA 정리를 PASS했다. 전역
런타임/설정은 바꾸지 않았고 private328572898B 사본도 제거했다. 새 code-only local 공급
경로의 component IMPLEMENTED=YES / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다.
기존 UV/PNPM/Bun 전송/pin 표적 Race13.994s와 canonical foundation43.748s도 PASS다.
제품 observer role·controller 종료 및 모든 언어의 실제 검역/Policy·release 공급은 미완료다.


### S02 OCI role와 controller — 아직 실제 종료 미완료

새 role CPP-v1은 O_CLOEXEC의 표준 헤더 누락으로 compile10.913s/NOT_RUN이었다. v2는
기존135 PASS·새 role1 FAIL/47.310s다. Focus-v1/40.444s의 phase1에서 새 latch가
SENTRY_EXEC의 필수 profile registration을 null로 전달한 것을 확인했고 production
wrapper를 사용한 focus-v2/39.398s만 PASS다. 권한 predicate를 완화하지 않았다.
Go controller의 준비/STOP과 독립 종료 구분·취소/위조 frame 및 기존 pin 회귀13.553s도
한정 PASS다. 이후 trailing frame fault 보존/readonly reader·역할 준비 시험은 새 후보다.

Actual role-v1/5.435s는 fixture의 필수 closure digest label 부재로 resource Register
FAIL이었다. Label만 교정한 v2/5.261s는 등록 후 observer incomplete였다. 진단을
보존한 v3/6.352s의 최초 원인은 OPEN_RESULT_CLASSIFICATION_PROC/PROC_SELF_CGROUP,
CONTROL/OCI_ROOT/OTHER·flags557056이다. Go1.27.3 runtime의 시작 전 기본 CPU/cgroup
탐지와 대응하며 실제 종료/완료 성공은 아니다. 새 고정 init에만 GOMAXPROCS=1을 적용해
Artifact Config 환경·파일 권한을 유지한다. 해당 명령의 CPP focus-v3/40.908s는 PASS고
actual 확인은 다음이다. Source helper/policy는 go list -deps의 실제 production GoFiles와
go.mod/go.sum을 대조해 재사용하고, 변한 시험 바이너리만 다시 만든다.

모든 actual은 helperwait0/정확한 runtime·pod-init 복원/Host 동일이며 각각374758305B,
374758393B,374765454B 사본을 회수했다. 아직 Artifact/DNS-client 행동 chain·Policy,
실제 role 종료/전체 S02·지원 생태계 연결·release 공급물을 완료로 표시하지 않는다.


Actual role-v4/5.470s는 GOMAXPROCS=1에도 같은 PROC_SELF_CGROUP에서 실패했다.
고정 Go runtime의 defaultGOMAXPROCSInit는 설정 판정보다 먼저 OpenCPU를 호출하므로
해당 설정은 cgroup 읽기 회피책에서 제외한다. Init CPU 수 고정의 의미만 유지한다.
인증된 responder OCI init의 own cgroup/mountinfo readonly만 허용한 focus-v4/40.051s는
PASS였고, actual-v5/6.185s는 다음 CGROUP_CPU_QUOTA에서 실패했다. 기존 consumer의
정확한 CPU quota/period read와 같은 두 경로·system mount를 해당 역할에만 적용하는
후보를 검증한다. Artifact/clone/write/다른 mount/비슷한 경로는 계속 거부한다.
두 actual의 helperwait0/Host 동일/정확한 복원과374762110B,374763374B 정리는 확인했다.
이전 완료 primitive 바이너리3개69133121B도95 process 무참조/query error0 뒤 제거했다.


정확한 CPU quota/period·system mount의 authenticated responder read와 negative를
추가한 focus-v5/41.347s 및 actual role-v6/5.365s는 PASS다. Root 소유 readonly 공급,
OCI init1001/Artifact boundary1000 readiness, STOP prerequisite와 독립 resource 종료·
observer SessionComplete를 확인했다. Helperwait0/Host 동일/정확한 복원·374763238B 회수다.
이 표본은 Artifact/DNS-client/Policy가 없는 역할 선행조건이며 실제 행동 chain과
기존 검사 owner·배포 공급물 연결/전체 S02 qualification은 여전히 MISSING이다.


### S02 fixed resolver와 DATA chain — 한정 actual PASS

Chain-v1은 per-mode 구형 client 재복사로 새 proof에서 제외했다. SHA equality를 강제한
v2는 Node 직접 CONTROL 실행의 own maps 실패, v3는 잘못된 --handoff의 exit125였다.
기존 npm -c handoff clone을 재사용한 v4/6.585s에서 NODE/ARTIFACT/CLONE_CHILD의
정확한 /etc/resolv.conf(system/flags32768) 실패를 확인했다. v5/6.036s는 새 DATA
attestation이 Docker의 neutral search . 행 생략을 거부했다. 별도 inert runc DATA는
요청 metadata Search[.] 유지·실제 파일의 search 행 부재와 Root0:0/0644를 확인했다.
다른 nameserver/domain/options/쓰기 권한/alias/추가 파일은 허용하지 않는다.
v6/6.526s는 검증 후 immutable OCI /etc/nsswitch.conf의 읽기로 이동했다. 기존 locked
name-service의 exact resolver/hosts system mounts와 NSS/host.conf/gai.conf OCI files만
NODE Artifact clone에 적용했고 CONTROL/role/provenance/write/mount/비슷한 경로 거부를
같은 기존 latch에 연결했다. Full CPP136/49.676s와 actual chain-v7/5.513s가 PASS다.
후자는 UID1001 fixed responder와 UID1000 Node DATA client의 DNS→HTTP200→더미 파일→
sink204, ARTIFACT network6·CONTROL network4·완전 drain/독립 종료를 확인했다.
Host 동일·helperwait0·정확한 복원/374868109B 회수이며 실제 acquire/Policy/전체 S02
qualification을 뜻하지 않는다. 같은 primitive를 반복하지 않고 다음은 기존 owner 연결이다.


### S02 기존 owner 연결 — npm archive actual3 PASS / 나머지 계속 OPEN

고정 code-only 공급은 기존 prepare/current reader를 재사용하며 전체 inspection에 하나다.
Development writer만 readonly 공급·UID1001 init을 받고 keeper/minimal은 기존 init을 쓴다.
기존 pythonPhaseRuntime의 stopAndCollect에 STOP prerequisite→독립 resource 종료→완전
observer drain/제어 채널 종료를 같은7초 정리 context로 연결했다. 기존 CPU watch의 종료
전이·공유 trace 원장·launch 한도는 유지한다. Npm/instance/ELF의 workspace ceiling에서
4MiB를 예약하며 code SHA와 고정 DNS 인자를 FixtureDigest에 결합한다. 새 daemon/일반
proxy/미래 소비자 구조나 third-party pin 예외는 만들지 않았다.

첫 owner 모의 회귀8.616s는 invalid Bun supply 거부 전 responder 자원을 준비한 순서를
검출했다. 동일 deadline 내 native 공급물을 자원 준비 전에 검증하고 두 환경에서 그
immutable 준비 결과를 재사용한8.599s 및 current 표적 Race14.198s는 PASS다.
Canonical foundation41.600s도 PASS지만 이후 profile/test-only 연결과 구분한다.
Actual-v1/0.856s는 정식 /usr/lib/heliopause가 없는 Host의 bind 준비에서 NOT_RUN이다.
Root fixture의 동일 factory를 test-only callback으로 주입한 v2/15.847s는 기존 raw
integrationRunner의 새 init method 누락으로 실패했다. Test adapter에서 production
Host executor의 동일 메서드를 전달한 v3/31.218s의 정상/fresh/조건부 DNS chain3개는
PASS다. Network10을 Policy 입력에 보존하고 두 환경 완료/외부 종료를 확인했다.
Helperwait0/정확한 복원/Host 동일과374902182B 회수를 확인했다. 정식 설치 경로·공급자
acquire/Policy·instance/ELF 실제 검사·Python/Go/Cargo·전체 S02 qualification은 미완료다.

이후 source delta는 동일 asset name을 기존 runtimeidentity owner로 옮기고 현재 연결한
npm/instance/root·ELF/provider inspection revision과 공통 protocol/compiler identity를
갱신한 것이다. Old state-v1 승인을 거부하는 검사와 관련3 inspection package 전체
Race17.356s 및 영향 package pinned Staticcheck16.430s가 PASS다. 기존 GoVersion 한 곳의
업데이트를 그대로 따르며 helper 동작 변경 시 이 공통 inspection revision을 갱신한다.
동일 source의 kernel/bundle·CPP136을 다시 검증하지 않았다.

영향 package go vet1.550s와 diff whitespace도 PASS다. 완료한 role/chain/owner 시험
바이너리3개는96 process의 무참조/query error0 및 원본 SHA 확인 뒤91714779B를
삭제했다. 재사용할 고정 helper/SDK/cache·원본 evidence는 유지한다. 새 CI/push/설치는 없다.
