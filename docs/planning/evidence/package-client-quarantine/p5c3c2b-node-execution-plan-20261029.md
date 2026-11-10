# P5-C3c2b — actual Artifact Node 실행 경계 감사와 수정 지도

착수 기준은 e0dc39a/treefff27985/2959tracked·1324source다. 아래 감사/후속 검증은 시점별
기록이며 최신 실행·timer 상태는 Queue가 소유한다. CI427은 timer04 소비/Required10SUCCESS로
완료됐고 이후 Node source를 포함하지 않는다. 한 개 M13-004 아래 독립 작업이다.
Read: INDEX와 sandbox L03/L08/L12/L18·artifact L02·evidence L16·ci L05/L06,
Architecture-005/Contract-002/client09§3, P5 baseline/C3c2a/C3c2b/hosted wiring leaf를 대조했다.

| 실제 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| ExecutionPreparationService/PreparedExecution.ExecutionContext·Bundle·Staged | YES | YES(current preparation) | YES(CI426) | YES(storage-only) | 실제 run backend/result 소비; preparation을 실행 성공으로 위장하지 않음 |
| bootstrap.materializeNodeExecution + complete NodeExecutionLayout | YES | YES(production parent) | YES(local current CPP0c09/CI427 pending) | YES(named local) | 그대로 재사용 |
| prepareNodeExecutionInputs/startNodeInputContainer/sole RO/close | YES | YES(DATA validation) | YES(local fresh2) | YES(C3c2a) | reader profile을 등록/시작 전에 고르는 closed 실행 variant와 기존45s context 유지 |
| boundaryHelper/origin/direct admission/CPP demotion | YES | YES(existing launch/Python/ELF) | YES(existing) | YES(existing) | exact Node handoff·별도 profile; 새 Node는 CONTROL로 실행하지 않음 |
| target/outcome query + existing lease/COMPLETE | YES | YES(8root actual) | YES(local CPP130) | YES(DATA dependency) | current readonly entry/vector와 같은 Node root, Host capture, complete trace/cleanup 결합 |
| hosttool.RunCaptured | YES | YES(Host transport) | YES(C1) | YES(bounded Host) | leased Node command 소비; overflow가 guest 정리를 대신하지 않음 |
| Artifact Node 실행/결과 | NO | NO | NO | NO | 아래 normal/nonzero 실제 API·production parent; 취소/overflow/evil 전체와 application/CLI는 후속 |

## 현재 gap만 구현

새 node_execution.go의 Backend.ExecuteNodeExecution은 actual current Context와 MatchesExecution
layout, pinned Node, admission-aware 보호 Host capture capability를 I/O 전에 요구한다. 기존 clean
DATA writer를 그대로 사용하고 reader만 새 `node-execution` profile로 등록한다. 기존 writer와
RO reader overlap→writer 정리→sole RO/census를 재사용한다. 기존 private input owner에 이미
생성한45s context를 보존해 preparation 뒤 새45s를 부여하지 않는다. Data-only public API는
기존 false wrapper이며 body/기본 profile·volume 한도는 유지한다.

새 closed `--origin-handoff-node`/`--handoff-node`는 기존 immutable helper/setpriv demotion과
ordered lease/nonce 원본을 재사용한다. Go helper와 CPP exact OCI bootstrap literal을 함께
갱신하며 old modes의 argv/의미를 유지한다. CPP Node mode는 새 profile에서만 소비하고, 새
profile의 launch는 이미 검증된 exact readiness tuple만 허용한다. Root는 irrevocable ARTIFACT,
expected CONTROL/network group 부재다. Pinned OCI `/usr/local/bin/node -- <current absolute
readonly entry> <vector args>`를 helper 첫 kernel record의 bounded vector와 target SENTRY_EXEC
양쪽에서 대조한다. Argv는 private runtime 메모리뿐이며 log/Manifest/Evidence/hash로 저장하지
않는다. Clone와 이후 exec는 active runtime 권한을 상속/복구하지 않는다.

Node profile은 기존 topology/budget을 재사용하고 exact current readonly input mount와 immutable
Node image만 받는다. Runtime read는 이미 고정한 native ABI와 알려진 own maps 경계부터
profile/ARTIFACT/root group-start/current Node image/pinned root/mount/flags로 한정한다. 일반
NPM CONTROL/FS 예외를 넓히거나 process name/출력 문자열을 신원으로 사용하지 않는다.
추가 실제 startup 요구가 드러나면 최초 원본/책임 지점과 부정 회귀로 좁게 검토한다.

Result는 현재 Sandbox module의 실제 memory-only NodeExecutionOutcome이다. 기존 inspection
SandboxResult/VerifiedBundle/InstallTarget을 만들지 않는다. Complete는 accepted target load,
external kernel terminal, 일치하는 Host process 종료/독립 bounded streams, unchanged whole
RO inputs, 완결 trace와 reader/container/descendants/helper/volume 정리 후에만 true다.
Raw streams는 clone accessors로만 읽고 formatter/JSON에는 노출하지 않는다. Guest nonzero나
signal은 별도 typed NodeProgramError로 원래 Host error를 unwrap하며 Complete data와 구분한다.
Cancel/overflow/WaitDelay/unknown transport/trace/cleanup은 incomplete·최초 cause 보존이다.
Host client 종료나2s pipe wait는 guest stop 증거가 아니다.

기존 production preparation actual parent에 same current typed flow의 실행을 추가한다. Public
TypeScript5.9.3/current whole checks·Policy 뒤 fresh2의 정상 help와 명시 invalid option의 guest
nonzero를 구분하고 raw output은 log에 남기지 않는다. Preparation/DATA-only 단계 metadata와
actual user_program_run=true metadata를 분리한다. 새 public CLI·Core/Application run contract,
TTY/파일 반출·추가 runtime/미래 milestone scaffold는 만들지 않는다.

## source 수정 지도 / 두 번째 검토

기존 owner 수정: boundary_helper.go/direct_exec_admission.go(새 closed mode), shared_observer.go
(새 profile topology), node_execution_inputs.go(기본 false wrapper·기존 context/reader 선택),
boundary_helper_test.go(새 mode와 old defaults), bootstrap의 기존 production integration parent,
observer.cc/latch(동일 kernel/role/mount/file dispatch의 새 scoped Node 분기). 새 Go는 실제 backend/
private result/회귀와 profile owner에 한정한다. 일반 role/FS/network/지원·예산/Policy invariant는
그대로이고 기존 구현 교체/대체는 없다. Canonical profiles/전체 영향 Race·locked CPP 모든 latches,
동일 source actual·기존8root와 npm/Python/ELF 및 정확한 cleanup 전 QUALIFIED=NO다. First
normal/nonzero 성공은 전체 악성/cancel/overflow·public CLI/전체P5/final acceptance를 닫지 않는다.

## 구현 후보와 첫 전체 검증 전 검토

Active private는 `/tmp/haa-p5-c3c-node-20261029-source`다. Go-side precheck v1의 format2.378s/
focused9.155s/tagged Bootstrap10.415s가 PASS이며 이는 CPP/actual 성공이 아니다. 이 Go9file
원본·binary SHA·logs는 raw p5-c3c-node-precheck-v1-*에 동결했다. 첫 CPP build 전에 Node reader만
container --workdir를 current readonly project로 지정했고 old DATA defaults/origin은 유지했다.
기존 IsExactHAAELFHandoffDemotionRead의 고정 setpriv ABI(/proc cap_last_cap/own status·OCI
account/library)만 kernel-검증 demoting bit로 재사용했다. Node active phase는 별개며 해당
권한을 clone/later exec/comm 이름으로 복구하지 않는다. Entry는 mountpoint의512B 한도를
잘못 전용하지 않고 정규 absolute path/4096B native bound를 쓴다. 모두 source review이며
실제 회귀 발생 인과나 runtime 해결 검증으로 쓰지 않는다. 다음 v2 canonical/영향 Race·CPP131와
same-source actual 전 QUALIFIED=NO다. Source는 아직 private draft이며 active CI427은 e0dc다.

## canonical architecture 반증과 현재 capture Port 수정 지도

v2 format0.697s/focused7.021s는 PASS지만 Quick53.321s의 architecture가 Sandbox→Hosttool
직접 import를 거부했다. 새 node_execution.go의 opaque capture concrete type import가 책임
지점이며 이 후보는 qualification 실패다. 최초 source/로그/checks를 raw architecture-red/v2에
보존한다. 규칙을 완화하지 않는다. 현재 필요한 Core ports의 CapturedProcessOutput/
ExecutionCapture(독립 bounded streams/overflow flags만)와 Hosttool의 작은 adapter method를
추가해 기존 RunCaptured 본문·구체 타입·한도/오류 identity를 그대로 재사용한다. Sandbox는
같은 wrapped.runner의 Port capability를 소비해 runner identity를 바꾸지 않는다. Capture
overflow는 Port flags로 incomplete이며 원래 Host error/cause는 join해 유지한다. 새로운
Policy/완료 authority나 future scaffold가 아니라 지금의 실제 Node caller가 사용하는 경계다.
Capture Port/adapter/새 Sandbox input만 missing으로 추가하며 Core는 Hosttool/Sandbox를
import하지 않는다. 두 번째 검토에서 ABI·동일 Host runner·private output/typed guest error
경계를 유지했다. 새 canonical/affected Race와 CPP/actual 전 QUALIFIED=NO다.

현재 allowlist는 Sandbox→Domain은 허용하지만 Sandbox→ports도 직접 허용하지 않는다. 따라서
neutral CapturedProcessOutput 데이터 계약은 Domain에 두고 Core Port/Hosttool adapter는 그
반환 타입을 쓴다. Sandbox는 같은 method signature의 private capability로 현재 runner를
소비한다. 새로운 architecture 예외는 없고 Hosttool은 Port를 실제 구현/정적 대조하며 기존
RunCaptured source는 바꾸지 않는다. Overflow flag/원래 오류는 그대로 유지한다. 이 수정은
architecture-red의 현재 capture missing을 해결하는 범위이며 qualification은 재검증 전 NO다.

v3의 architecture는 기존 규칙 그대로 통과했지만 Quick111.199s의 pinned Staticcheck가 새
errors.New 두 곳 ST1005와 직접 nil Context negative SA1012를 거부했다. 최초 source/log/checks를
staticcheck-red/v3에 보존했다. Error 문구를 소문자로 시작하고 nil negative는 기존 table/value
형식으로 유지한다. Rule 억제·Context 허용 변경이나 runtime 실패 인과로 쓰지 않는다. 다음
v4 canonical/영향 Race·CPP/actual 전 전체 qualification은 NO다.

## 현재 동결 후보 — Go qualified / CPP·actual 대기

v4의 format1.727s/focused7.697s/Quick255.635s·Hosttool+Sandbox+Bootstrap 전체Race36.605s,
tagged Bootstrap4.142s/Sandbox7.877s가 PASS다. Docs 첫 실패는 private에 CI427 registration
문서를 복사하지 않은 별도 준비 누락이다. 원본7step은 v4-original-checks/개별logs로 보존했고
runtime1330 byte equality 뒤 docs1.209s/security19.053s만 수행해 qualified8을 구성했다.
제품 Go/CPP source는 이 문서 복사 전후 동일하며 실행 코드 검사를 이유 없이 반복하지 않았다.

현재1330 inputs(기존 수정8/newGo6)을 Root/private에 동결했다. Bootstrap fce76586/
51,025,783B와 Sandbox eba2ab6e/27,660,230B가 대응한다. Capture 기존 본문은 불변이고
새 Domain/Port/Hosttool adapter가 실제 소비자와 연결된다. CPP build는 canonical SDK/pins의
p5-c3c-node-v1-observer-build producer로 진행 중이며 예상131 latches/actual은 미완료다.
이 상태의 source는 아직 커밋·push되지 않았다. CI427/e0dc와 timer03 due22:57:50UTC/
session31913은 이전 공통 후보이며 새로운 Node 실행 검증에 전용하지 않는다.

## 첫 실제 실패와 test-only 원인 수집 수정 지도

CPP canonical247.008s/131latches는 PASS지만 first actual-v1은33.158s/parent31.04s에
readonly execution reader의 Docker start exit1과 HELPER_CRASHED로 실패했다. Artifact
admission/capture는 미도달·실제 Node run metadata0이다. Raw result의 user_program_run=true는
operator 의도 상수이며 관측이 아니다. Assessment가 이를 명시하며 원본을 편집하지 않는다.
Source1330/CPP/Host/42/ZIP/Docker 보존·policy helperwait0/confirmedstop·소유 workspace 부재는
확인했다. Execution owner는 IMPLEMENTED=YES/WIRED=YES/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다.

초기 root와 private cwd 권한 충돌은 아직 가설이다. 별도 default runc·고정 /bin/true DATA의
root-private/tmp·root-public/root·UID1000-private/tmp 세 경우가 모두 exit0·Docker 전후 동일이다.
이 결과는 original gVisor 실패의 직접 원인이나 정상 Node 실행 증거가 아니다. 기존
hosttool.Output가 보존한 *exec.ExitError.Stderr를 production integration의 실패 분기에서만
최대4096B 출력하는 test-only 진단을 추가한다. 고정 공개 fixture만 실행하며 user output을
제품 로그에 추가하지 않는다. 기존 Hosttool/capture/observer 권한·예산은 그대로다. 두 번째
검토에서 errors.As의 원래 error identity/cleanup와 normal 성공 assertion을 유지한다. 원래
1330 source·첫 actual 원본은 별도로 보존했고, 새 prefix의 format/tagged compile 뒤 같은
CPP로 bounded diagnostic을 수행한다. 이는 원인 수집이며 full qualification으로 표시하지 않는다.

Test-only v5 format4.513s/tagged build5.657s는 PASS지만 actual-v2-diagnostic3.169s는 기존
observer lock 때문에 constructor에서 미도달이었다. V1 helper crash 뒤 lock/control socket이
남았음을 확인했다. Root 전체79proc/FD·bound Unix·Docker container0/query errors0,
exact UID1000/inode/mode/size와 nonblocking exclusive lock 뒤 두 stale entries만 제거했다.
이는 standalone recovery이며 original Close 성공이나 Node 원인 해결이 아니다. V2 원본14개와
operator/build identity·복구 원본을 보존한다. V3는 빈 observer directory preflight와 after
inventory를 추가해 같은 진단만 수행한다. 최초 actual-v1의 cleanup 표현은 no live helper/
Docker/workspace에 한정되며 observer endpoint complete cleanup 주장은 제외한다.

## gVisor 초기 cwd 직접 원인과 현재 좁은 교정 지도

V3 diagnostic33.575s/parent31.72s의 원래 Docker ExitError.Stderr가 `failed to find initial
working directory /tmp/haa-node-execution/project: permission denied`를 확인했다. Test-only
추가를 제외한 product1330/CPP29837c4d는 first와 같고 Artifact run0이다. Runc DATA PASS는
해당 gVisor 오류를 반증하지 않는다. Helper global crash와 retained endpoint는 원래 실패에
수반된 별도 사실이며 cleanup 완전성을 주장하지 않는다. 같은 exact recovery를 새 inode로
다시 수행한다. V3 원본14개/operator/assessment를 동결했다.

현재 client09§3의 초기 noninteractive stdout lane은 Host project/cwd를 전달하지 않고
선택 JS를 전체 readonly graph의 절대 경로로 실행한다. Private project cwd를 OCI root에
선택한 새 후보는 반증됐다. 기존 createArguments/기존 qualified pipx native의 고정 `/`
초기 cwd를 execution reader에도 사용한다. nodeExecutionWorkingDirectory의 값/설명과 그
기존 회귀만 수정하며 same target absolute entry/vector는 유지한다. Post-demotion shell/
preload helper나 새 process stage가 필요하지 않다. File-producing 도구 지원은 별도 계약
미완료이며 이 교정에 포함하지 않는다. 두 번째 검토에서 cap-drop/UID1000 private tmpfs/
readonly graph·기존45s/role·ordered admission/CPP/whole binder를 유지했고 새 source canonical/
영향 Race/tagged build·actual과 기존8root/3consumer 전 QUALIFIED=NO다.

## Cwd 교정 후 실제 Node runtime read gap

V6 canonical8(format0.769/focused7.192/Quick138.664/영향전체Race33.898/tagged3.827·8.004/
docs0.366/security13.702s)가 PASS다. Source1330/Go5044bd93·6c77fa31/same CPP29837c4d의
actual-v4는35.610s/parent33.71s에 실패했다. Reader OCI init은 이제 시작했고 kernel fault가
NODE/ARTIFACT/DIRECT_EXEC_ROOT/current pinned image의 PROC_SELF_CGROUP/flags557056에서
발생했다. Node target startup은 관측됐지만 selected JS 시작/완료는 입증되지 않았다. Raw
user_program_run=false는 성공 metadata0을 뜻하며 startup0 증거가 아니다. Source/CPP/Host/
42/ZIP/Docker 보존·policy helperwait0/confirmedstop·reader/workspace/observer endpoints0이다.

기존 IsPinnedNodeExecutionRead owner의 current accepted Node active phase에만 동일 TGID의
`/proc/<tgid>/cgroup` readonly557056을 추가한다. 기존 NPM Node runtime owner가 이미 같은
self cgroup ABI를 사용함을 대조했지만 그 broad CONTROL/clone paths는 재사용하지 않는다.
Literal self alias/foreign TGID/..·trailing/인접 이름/write·다른 flags와 inactive/clone/laterexec/
다른 profile/role/start는 거부한다. 두 번째 검토에서 current kernel group/start/image/role/
provenance·system proc mount/ordered target binding 경계를 유지했다. CPP2owner만 수정하며
Go 구현은 V6 그대로다. Current CPP canonical131+확장 negative와 same-source actual 전
새 후보 QUALIFIED=NO다. Initial cwd의 한정 해결을 전체 Node 실행 성공으로 확대하지 않는다.

## 자기 cgroup 후 memory-limit ABI 후보와 수정 지도

CPP-v2 canonical313.792s/131PASS 뒤 current Go-v6 byte-identical+CPPfa52의 actual-v5는
37.827s/parent35.92s FAIL다. Self cgroup 경계를 지난 다음 pinned NODE/ARTIFACT/root의
SYS read557056에서 거부됐다. Diagnostic mount FNV413fcef2f4134dfe/path99aa297467e264c0는
기존 Node/Bun runtime owner의 고정 `/sys/fs/cgroup/memory`와 `memory.soft_limit_in_bytes`
후보에 일치한다. Fingerprint는 진단뿐이며 collision-free path 증명이나 permission authority가
아니다. Source/CPP/Host/42/ZIP/Docker와 helperwait0/confirmedstop/owned resources·endpoints0은
확인했고 selected JS 시작/완료는 NOT_PROVEN이다. V5 raw14개/assessment를 먼저 보존한다.

기존 IsPinnedNodeExecutionRead의 같은 current active Node phase에만 system mountpoint
`/sys/fs/cgroup/memory` 아래 exact soft/hard memory-limit 두 native ABI file을 readonly557056
으로 연결한다. 기존 qualified Node runtime owner/ARTIFACT Bun owner의 같은 고정 pair와
flags/mount 조건을 대조했다. 일반 sys prefix·다른 container 경로·write·임의 flags·이름/role
신뢰·Control/clone 예외는 추가하지 않는다. 같은 함수/latch2owner만 수정한다. 두 번째 검토에서
kernel group/start/image/ordered target/active bit를 유지했고 proc와 OCI 분기는 그대로다.
Exact pair 정상과 foreign/suffix/traversal/wrong mount/write·다른 flags·clone/inactive/start/
profile negatives를 추가한다. 새 CPP canonical/actual 전 QUALIFIED=NO이며 Node Go bytes/
정상·취소·캡처 구현은 V6 그대로 재사용한다. 앞 자기 cgroup 통과를 전체 실행 성공으로 쓰지 않는다.

## 기존 pinned Node kernel ABI의 새 execution profile 연결 지도

CPP-v3 canonical277.925s/131PASS와 same Go-v6의 actual-v6는36.113s/parent34.20s FAIL다.
Memory-limit 경계를 지나 다음 PROC read557056가 FNV8918b20ae6ad5038로 거부됐고 기존 고정
`/proc/meminfo` 후보와 일치한다. Fingerprint는 계속 진단뿐이다. Source/CPP/Host/42/ZIP/Docker/
helperwait0/confirmedstop·owned resources/endpoints0은 PASS지만 selected JS 시작/완료는
NOT_PROVEN이다. 원본14개/assessment를 보존했다. 앞 stage 통과를 overall PASS로 쓰지 않는다.

기존 observer의 NPM current Node runtime ABI owner(`IsPinnedNpmRuntimeRead`의
Node-only tail)는 이미 meminfo/version_signature/overcommit_memory와 exact CPU-online 및
self maps/cgroup·memory-limit pair를 지원한다. 기존 owner는 그대로 두고 이 유한 data 목록의
나머지 네 파일만 새 current Node execution profile에 연결한다: `/proc/meminfo`,
`/proc/version_signature`, `/proc/sys/vm/overcommit_memory`, `/sys/devices/system/cpu/online`.
첫 파일만 이번 새 actual에서 지문으로 좁혀졌고 나머지는 existing qualified pinned Node ABI
근거로 함께 연결하는 후보이지 새 actual 관측으로 표시하지 않는다. 일반 proc/sys tree나
name-service·Host paths를 추가하지 않는다. Exact system mountpoint `/proc`/`/sys`,
readonly flags557056·accepted pinned Node/group-start/ARTIFACT/directroot/active를 요구한다.
기존 CONTROL/clone runtime owner와 Go bytes·정책·한도는 그대로다. 두 번째 검토에서 이
네 literal뿐이고 suffix/traversal/foreign/Wrong flags·mount/profile/role/start·clone/inactive
negatives를 검토했다. CPP2owner의 새 canonical/full131·actual와 기존 소비자 전 qualification은 NO다.

## DEV_NULL read 경계의 기존 consumer 대조와 좁은 지도

CPP-v4 canonical308.052s/131PASS와 same Go-v6의 actual-v7는39.814s/parent37.85s FAIL다.
앞 kernel-info 경계를 지난 current pinned NODE/ARTIFACT/DIRECT_EXEC_ROOT의 DEV_NULL,
system mount `/dev` read557056가 거부됐다. Raw14개/assessment를 먼저 보존했고 selected JS
시작/완료는 NOT_PROVEN이다. Source/CPP/Host/42/ZIP/Docker 보존·policy helperwait0/confirmedstop·
owned resources/observer endpoints0은 PASS다. 앞 syscall 통과를 전체 실행 성공으로 쓰지 않는다.

기존 IsPinnedNpmRuntimeRead의 canonical null read와 IsPinnedCargoBuildProgramRead의
untrusted program exact `/dev/null`/`/dev`/system/557056을 대조했다. 같은 조건을 새 profile의
현재 accepted Node active phase에만 연결한다. IsWriteCapableOpen의 선행 거부와 모든
kernel group/start/image/ARTIFACT/directroot/active 조건은 유지한다. 일반 device tree와
null write/RDWR·다른 flags·urandom/random/zero·alias/suffix/traversal·다른 mount/profile/role/
start·clone/inactive는 거부한다. 두 번째 검토에서 새 literal 하나뿐이며 기존 소비자의 owner는
불변이다. CPP2owner/latch의 정상·위 negative를 추가하고 locked canonical131·actual 및
기존 소비자 검증 전 qualification은 NO다. Go 구현은 byte-identical V6다.

## Normal/nonzero actual named foundation 완료와 남은 범위

[최종 local receipt](./p5c3c2b-artifact-node-local-20261029.json)가 current1330과 Go-v6의 동일
compiler/test inputs·8checks/Go5044bd93·6c77fa31, CPP-v5 canonical289.119s/131·b76b0c86을
연결한다. Actual-v8는69.401s/parent67.48s에 fresh2의 TypeScript help exit0/stdout4980B와
invalid-option exit1/stdout62B를 구분했다. 두 경우 stderr0·accepted target/external kernel
terminal·bounded capture·readonly whole input/trace/전체 disposal 확인 뒤만 Complete다.
Guest1은 typed NodeProgramError로 원래 driver error를 보존했다. 같은 helper의 기존8root
9.678s와 npm/Python/ELF3parents12.050s도 PASS다. 세 actual 모두 source/CPP/Host/42/ZIP/
Docker 보존·helperwait0/confirmedstop·owned resources와 observer endpoints0/noSKIP이다.

Initial cwd/finite kernel ABI/null read의 앞 실패와 원본은 그대로 보존했다. Normal/nonzero
backend foundation의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED는 named local 범위
YES이며, arbitrary 프로그램 안전성·Node signal/cancel/overflow/child/악성·public application/
CLI/new hosted/final profiles와 전체P5는 닫지 않는다. 다음은 같은 M13-004의 P5-C3d runtime
실패/악성 actual qualification이다. 먼저 현재 단위를 한글 중간 commit과 중요한 grouped CI로
전달하고20min timer를 등록한다. 이후 source는 실행 중 CI를 새 push로 취소하지 않는다.
