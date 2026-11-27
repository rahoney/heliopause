# P2b2 — pnpm의 실제 guest 공급·admission 수정 지도

Baseline `01a6e19`, [배포 로컬 결과](./p2b2-pnpm-distribution-local-20261024.json)와
[상위 runtime 지도](./p2b2-pnpm-runtime-plan-20261024.md)를 따른다. 동일 M13-004/P2b2이며
P2c graph/layout/CLI와 P3 이후 항목은 시작하지 않는다.

Read: [D-012/D-015](../../../threat-model/04-trusted-tooling-and-evidence.md),
[확장 계획 §5](../../20-package-client-quarantine-plan.md#5-검증-matrix와-자원-정리),
오답노트 sandbox L08/L12/L18, artifact L15/L20, evidence L11/L16, ci L05/L06.

## 1. 소스 감사와 실제 남은 것

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| PNPM pin/proof/archive/native·NOTICE와 배포/설치/복구 — runtimeidentity, releaseprovenance, releaseinstall/activation, native Promotion | YES | YES(배포 범위) | NO(새 전체 제품) | NO | 아래 actual runtime/ARTIFACT 소비자와 원격 |
| UV 인증 tool 공급/Root tmpfs/transport/cleanup — uv_tool_supply.go, uv_runtime_volume.go, uv_resolver.go | YES | YES | YES(기존 UV 범위) | YES(기존 UV 범위) | PNPM이 같은 transport/volume 규약을 좁게 소비 |
| Node image·NPM resolver helper/typed executor/observer — npm_resolver.go, boundary_helper.go, shared_observer.go | YES | YES | YES(기존 npm 범위) | YES(기존 npm 범위) | PNPM-only helper volume/profile·실제 native startup/query/lock |
| PNPM의 고정 설치 파일 read-only 공급·정확한 Root exec — 현재 없음 | NO | NO | NO | NO | 고정 경로/Root ownership/byte pin, guest 주입/readback, observation·cleanup |
| HAA release의 native PNPM을 ARTIFACT로 검사 — ReleaseSetInspector는 existing native inspector에 연결됨 | YES | YES(구조/합성 소비자) | NO | NO | 실제 PNPM payload의 github-elf probe; CONTROL 권한 승계 금지 |

기존 native update/volume/observer protocol을 새 engine으로 바꾸지 않는다. 현재 필요한 두
UV/PNPM 소비자 사이의 기계적인 shared helper 추출만 허용하며 wrapper/검증 조건을 유지한다.

## 2. 고정 공급과 경로

Source는 signed HAA native set이 설치한 PNPM asset이다. Canonical native install root의
`current/pnpm-linux-amd64`만 읽으며 PATH/Corepack/npm 설치/cache/user file/호스트 실행이나
자동 다운로드를 사용하지 않는다. 기존 releaseinstall.SystemInstallRoot의 **같은 값**을
runtimeidentity의 공용 canonical path owner로 옮겨 기존 public constant를 alias로 보존한다.
Sandbox는 이미 허용된 runtimeidentity만 소비하며 releaseinstall/hosttool 의존성을 추가하지 않는다.

Reader는 Root-owned canonical root/versions/version directory, Root-owned current의 정확한
relative versions target, regular/Root UID·GID/0555/single-link/exact size/hash 파일을 확인한다.
`os.Root`와 열린 FD로 경계를 고정하고 current·file 신원의 전후 변화를 거부한다. Version은
경로 구성요소일 뿐 approval authority가 아니다. Authority는 기존 보호된 설치와 generated
PNPM byte pin이며 source/proof 도입 때 검증한 공식 provenance를 새 user receipt로 대체하지 않는다.

기존 developer Host에 해당 native set이 없으면 fail-closed이며 raw user 경로로 fallback하지
않는다. Actual qualification은 별도 mount/network namespace에서 고정 bytes를 Root 소유로
배치하며 Host 설치 파일을 활성화하지 않는다.

## 3. 정확한 교체/확장 위치

| 실제 owner | 변경과 검증 |
| --- | --- |
| runtimeidentity의 canonical path와 releaseinstall/installer.go | 같은 기존 install root 값을 공유; 모든 기존 caller의 경로/권한/승인 의미 불변 |
| sandbox/pnpm_tool_supply.go | current/FD/UID·mode·link·size/hash/cancellation guard; fixed native body만 준비, Host exec 없음 |
| sandbox/uv_tool_supply.go의 single-file tar/introduce/readback | UV와 PNPM의 두 실제 소비자에만 공통 transport를 추출; enum으로 고정 name/path/size/hash 선택, unknown 거부; UV upstream mode0755와 guest0555 구분 유지 |
| sandbox/uv_runtime_volume.go의 create/verify/verifyTarget/close | protocol 그대로 공유; UV wrapper/identity 유지, PNPM은 별도 owner/prefix/label·tool hash. 64MiB/Root helper/9p/sole attachment/confirmed cleanup 불변 |
| sandbox/pnpm_runtime.go 및 npm_resolver.go 기존 fixed create | 고정 Node image와 PNPM-only helper volume, `pnpm-resolver` profile; 처음에는 offline startup/version/Node query/empty lock만. Project selection/CLI placeholder를 만들지 않음 |
| sandbox/shared_observer.go, direct_exec_admission.go의 실제 profile 소비자 | PNPM topology/nonce/root argv만 추가; npm/UV와 Python closure/ledger 예산은 보존 |
| observer.cc·observer_latch_test.cc | `/haa-runtime/pnpm`의 sealed single helper9p/nonce/clone lineage만 새 PNPM class로 인정. Native GNU loader/read/Node query는 실제 첫 fault와 exact argv·read-only ancestry 근거로만 확장 |
| 기존 github-elf 실제 probe | release asset PNPM은 계속 ARTIFACT이며 PNPM CONTROL 예외를 쓰지 않는다. 실제 version/help probe와 기존 거부/관찰 완전성을 별도 확인 |

`RESOURCE`/이름/경로/도구의 출력만으로 trusted-control 또는 clean Evidence를 만들지 않는다.
다른 profile·helper shadow·ARTIFACT·child/re-exec·unknown creator는 새 Root 권한을 얻지 못한다.

## 4. 구현 순서와 두 번째 검토

1. 고정 파일 source와 공통 transport/volume을 먼저 연결한다. Missing/foreign/symlink/hardlink/
   writable/substitution/current change/oversize/cancel·고정 두 tool 경계·sole attachment/unknown
   query 거부를 검사한다. 기존 UV profile 성공을 PNPM 실행 성공으로 쓰지 않는다.
2. 기존 one-shot helper와 PNPM profile·C++ 전체 latch를 연결한다. 고정 Node image의 실제
   PNPM startup/version→exact Node query→offline empty lock를 순서대로 확인하고 최초 fault를
   보존한다. 필요한 read/exec만 동일 role·image·argv·flags 아래 보완한다.
3. 별도 ARTIFACT lane에서 actual PNPM release member probe를 검사한다. Signed supply나
   trusted resolver startup의 성공을 artifact quarantine 성공으로 전용하지 않는다.
4. 해당 공통 native/observer 경계에서 누적 candidate를 한 번 push하고 실제 run/head/deadline을
   기록한다. 최근 같은 profile 실측 기반 타이머를 사용하며 대기 중 독립 작업은 계속한다.
   최종 P2의 기존 npm/PyPI·영향 CPU/CUDA·native macOS/Required 범위는 별도로 닫는다.

두 번째 검토: Root0 파일 source와 guest의 CONTROL 실행, 새 release의 ARTIFACT 검사,
공개 npm graph/SRI binding·promotion은 각각 다른 authority다. 공유 helper의 default/unknown
selector가 권한 확대가 되지 않게 실제 두 enum을 검사한다. 기존 타입/wrapper를 보존하는
기계적 추출은 기존 protocol 대체가 아니다. Source 설치가 없다고 user path/ambient binary를
허용하지 않는다. 실제 test payload와 사설 image/volume/binary는 마지막 소비·참조0 확인 뒤
삭제하고 작은 원본과 normalized 결과만 보존한다.

## 5. 첫 실제 GNU startup 경계

Quick174.844s/security/docs, 관련3package Race23.754s와 C++107 latch는 PASS다.
첫 actual PNPM startup4.21s는 flags557056/CONTROL/DIRECT_EXEC_ROOT/OCI_IMAGE의
librt read 분류에서 STREAM_FAULT이며 기존 UV startup4.37s/query4.51s는 PASS다.
Raw `p2b2-pnpm-admission-native-first`의 locator와 기존 Cargo guard를 대조했다.
Host protected12/images/bridge/service 불변·private daemon wait0×2·잔여 자원0이다.

보완 위치는 observer.cc의 PNPM 전용 exact admitted Root read predicate와 기존
ClassifyFilesystem callsite, 관련 negative latch다. `/usr/lib/x86_64-linux-gnu/librt.so.1`
한 경로/flags557056·oci-root `/`만 먼저 연결하고 다른 profile·ARTIFACT·clone·잘못된
mount/flags/expected group/exec context는 거부한다. 공용 loader allowlist는 바꾸지 않는다.
후속 실제 fault의 별도 원인을 기존 보완의 인과나 전체 성공으로 기록하지 않는다.
C++의 검증된 같은 canonical recipe는 개발 전용 사본의 성공 cleanup만 보류하여 활성
workspace를 재사용하고 마지막 소비 뒤 제거한다. 제품 build source/config는 바꾸지 않는다.

보완1 C++108 latch 뒤 actual3.83s는 librt 경계를 지나 새 PROC_SELF_MAPS read
(flags557056/CONTROL/DIRECT_EXEC_ROOT)에서 거부됐다. Protected source의 정상과11개
실제 변조/소유권/링크/경로 거부는 모두 PASS이며 Host 불변·daemon wait0/잔여0이다.
후속은 기존 Rust/Cargo의 자기 memory map guard와 같은 exact PID `/proc/<group>/maps`
읽기만 PNPM profile·admitted Root·`system /proc` 아래에 추가한다. Other PID/self alias/
wrong mount/ARTIFACT·clone·다른 profile 부정은 유지한다. Raw p2b2-pnpm-startup-1이 소유한다.

보완2 C++108 PASS 뒤 actual3.78s는 다음 VM_OVERCOMMIT_MEMORY read에서 거부됐다.
기존 UV/Cargo의 exact `/proc/sys/vm/overcommit_memory` 읽기와 대조해 같은 PNPM Root/
flags557056·system `/proc`에만 추가한다. 원본 p2b2-pnpm-startup-2, Host 불변/잔여0/wait0.

보완4는 product bytes를3과 동일하게 유지하고 잘못된 상호 negative fixture만 교정했다.
C++108 PASS34.521s 뒤 actual PNPM `--version`3.69s가12.15.0·완전한 session/잔여0/
wait0/Host 불변으로 PASS다. Raw p2b2-pnpm-startup-4가 startup 범위를 소유한다.
다음 opt-in 입력은 controller-owned 빈 package.json과 engines.node 범위다. `install
--offline --lockfile-only --ignore-scripts --ignore-workspace` 및 전용 store/state에서
lock9를 bounded read한다. Public graph/Policy/설치/Node query 권한 완료를 미리 주장하지 않는다.

### Empty-lock 입력 전송 누락과 수정 전 검토

첫 empty-lock fixture3.40s는 `docker cp --archive ...:/tmp`가1로 거부되어 PNPM
install에는 도달하지 않았다. 관찰은 complete/자원0/wait0/Host불변이다. 기존 UV의
`/haa-site`와 Cargo 입력은 Host가 볼 수 있는 named tmpfs/guest9p를 이미 사용한다.
이를 적용하지 않은 fixture 누락이며 PNPM resolver나 native startup 회귀가 아니다.
원본 p2b2-pnpm-startup-empty-lock-first를 보존한다.

실제 남은 project transport는 기존 closureVolume owner/transaction/hash·목적지·capacity·
attachment/cleanup을 재사용한다. 새 고정 목적지는 `/tmp/haa-pnpm-input` (Host tmpfs
128MiB, UID/GID1000,0700,noexec)이며 Node helper mount와 분리한다. `createInputVolume`의
기존 Cargo fixed config 검증에 이 concrete nonexecutable 목적지만 추가하고 임의 경로는
거부한다. PNPM create가 같은 owner의 project volume을 요구하며 observer topology에는
`workspace /tmp/haa-pnpm-input, parent /tmp,9p`만 추가한다. `/tmp`의 기존 workspace
predicate를 유지하여 C++ 새 경로 허용은 만들지 않는다. UV output cleanup의 동일 프로토콜은
고정 UV/PNPM 목적지 wrapper가 공유하고 nil/다른 목적지/unknown query/attached 거부는 보존한다.

Opt-in startup fixture도 같은 project volume을 준비·단독 부착·정리 검증한다. Empty 입력과
lock는 그 named mount에서만 전달한다. 기존 npm은 같은 argv, UV/Cargo/Go/Python volume은
같은 목적지/옵션을 유지하는 회귀를 함께 검사한다. 이는 실제 PNPM runtime input/output의
현재 gap이며 P2c public graph/CLI placeholder 추가가 아니다.

Project named mount 수정 후 actual startup4.77s PASS이며 같은 empty 입력 전송도 통과했다.
PNPM install은 다음 own-PID cgroup read flags557056/system `/proc`에서4.65s 거부됐다.
기존 UV exact-self cgroup guard와 대조해 동일 PNPM Root role/context에만 연결한다.
Alias/다른PID/ARTIFACT·clone·타profile/쓰기는 거부하며 새 daemon/완료 프로토콜은 없다.
Raw p2b2-pnpm-startup-project-first는 Host불변/잔여0/wait0과 첫 fault를 소유한다.

Cgroup 보완5 C++108 PASS 뒤 actual 빈 lock4.02s는 다음 own-PID mountinfo read에서
거부됐다. 기존 UV의 동일 flags557056/system `/proc` exact-self mountinfo guard만
PNPM Root에 연결하며 다른PID/alias/role/profile/쓰기 부정을 유지한다. Raw
p2b2-pnpm-startup-project-cgroup, Host불변/잔여0/wait0이다. Project 연결 Go의
canonical Quick38.108s/security6.593s/docs.226s는 PASS다.

보완6 C++108 PASS 뒤 빈 lock3.85s는 CGROUP_CPU_QUOTA/system CPU mount/flags557056에서
거부됐다. 기존 UV가 쓰는 quota/period 읽기 쌍을 같은 PNPM admitted Root에만 연결한다.
판정·budget은 바꾸지 않으며 CPU stat/다른 cgroup·mount/쓰기/다른 role/profile은 거부한다.
Period를 이미 관측했다고 쓰지 않는다. Raw project-mountinfo가 첫 quota 실패/정리/Host불변을
소유한다. Project Go 범위의 관련3package 전체 Race22.741s도 PASS다.

보완7 C++108 PASS 뒤 empty lock4.05s는 `/sys/devices/system/cpu/online` read
(flags557056/system `/sys`)에서 거부됐다. Locator4a3dbc4eb888ec56과 mount9c0d5f9c5cb6707b를
exact literal로 대조했다. 해당 PNPM admitted Root/read-only CPU count 한 경로만 추가하고
possible/alias/타mount/쓰기/다른role/profile은 허용하지 않는다. Raw project-cpu가 첫 실패와
Host불변/잔여0/wait0을 소유한다. Locator는 진단이며 실행·읽기 권한은 기존 kernel role다.

### Node query 원본 검토 — 실제 실행과 분리

동일 source91774e72의 [system_runtime_version.rs](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/cli/src/cli_args/pre_command/system_runtime_version.rs)와
[graph-hasher/engine_name.rs](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/graph-hasher/src/engine_name.rs)를
추가 확인했다. Native CLI는 자신의 고정 PATH에서 Node를 선택해 `node --version`을 실행하고
한 프로세스에서 결과를 한 번 cache한다. 이 cache는 HAA 검역 결과나 Evidence가 아니다.
Raw p2b2-pnpm-node-query-source-review.json이 source와 원본 파일 hash를 소유한다.

실제 query가 필요할 때는 기존 UV exact-query의 kernel clone creator/start identity·
one-shot first exec·ARTIFACT demotion·expected CONTROL group 제외·later exec revoke를
재사용한다. PNPM의 pinned helper와 Node OCI의 sealed/read-only image, exact version-only
argv를 함께 검사하며 artifact에게 Node code 실행·CONTROL/network 권한을 주지 않는다.
공식 source 검토만으로 실제 query·startup read·public selection 완료를 선언하지 않는다.

보완8 C++108 PASS 뒤 empty lock3.90s는 own executable `/haa-runtime/pnpm` read
(flags557056/helper `/haa-runtime`)에서 거부됐다. 같은 PNPM admitted Root의 고정 native
파일 한 개만 read로 추가하며 alias/UV file/다른helper/다른role·profile은 거부한다.
독립 ARTIFACT probe4.98s는 INCOMPLETE/M6_DYNAMIC_EXECUTION_FAILED/STREAM_FAULT다.
첫 fixture가 상세 FaultOpen을 남기지 않아 정확한 경로·원인은 NOT_CAPTURED다. 같은 기존
bounded FaultOpen 인터페이스를 test-only 출력에 연결해 먼저 원인을 확인하며 ARTIFACT
권한을 추측해 고치지 않는다. Raw p2b2-pnpm-startup-native-both-first가 해당 범위를 소유한다.

보완9 C++108 PASS 뒤 CONTROL empty lock3.85s는 exact `/dev/urandom` read
(flags557056/system `/dev`)에서 거부됐다. 기존 Cargo entropy 읽기와 대조해 같은 PNPM
Root에만 연결하며 random/null/alias/write·다른role/profile은 거부한다.

독립 ARTIFACT 재확인은4.98s의 같은 입력에서 bounded diagnostic을 추가한 결과이며
새 syscall 판정을 바꾸지 않았다. Actual5.16s의 첫 fault는 ARTIFACT/DIRECT_EXEC_ROOT/
OCI_IMAGE, `/usr/lib/x86_64-linux-gnu/librt.so.1`, flags557056·oci-root `/`로 확인됐다.
기존 [native ELF 호환 계획](../m13-004-convenience/native-elf-compatibility-plan-20261020.md)의
동일 GNU ABI matcher에 librt SONAME만 추가하고 기존 read-only/handoff/role/namespace·
network/clone·honeytoken/예산을 보존한다. PNPM CONTROL의 native-helper 예외는 재사용하지 않는다.
Raw p2b2-pnpm-startup-native-diagnostic이 두 first faults와 Host불변/잔여0/wait0을 소유한다.
이후 maps/VM 등 새 ARTIFACT fault는 아직 관측하지 않았으므로 자동 허용하지 않는다.

보완10 C++108 PASS 뒤 CONTROL lock3.85s는 fixed sandbox OS marker
`/proc/sys/kernel/osrelease` read(flags557056/system `/proc`)에서 거부됐다. 기존 M6 marker
읽기와 동일하게 PNPM Root 한 경로만 연결하며 hostname/alias/다른role은 허용하지 않는다.
ARTIFACT5.13s는 다음 fixed OCI `libpthread.so.0` 읽기에서 거부됐다. P2b1의 실제 ELF
NEEDED는 libc/libgcc_s/librt/libpthread/libm/libdl다. 기존 M13 ABI 구현 방식대로 현재
관측된 pthread와 선언된 dl SONAME을 같은 기존 read-only ARTIFACT guard/matcher에 연결하고
M6의 고정 ABI 목록만 갱신한다. 임의 library/prefix·maps/environ/다른PID·쓰기·network/
CONTROL·clone 권한은 유지한다. dl 실패를 이미 관측했다고 쓰지 않는다. Raw native-entropy-abi와
p2-pnpm-native-elf-observation-20261024.json이 실행/선언의 서로 다른 근거다.

보완11의 CONTROL startup3.77s/빈 lock3.82s는 PASS·완전한 관찰·의심0이나 Node query 실행은
아직 입증하지 않았다. 원본의 `check_runtimes`가 검사하는 `devEngines.runtime`에 pinned
Node version과 `onFail:error`를 명시한 별도 실제 입력을 추가한다. Auto-download 경로가
아니며 unsupported/runtime mismatch는 실패해야 한다. 같은 빈 lock helper를 재사용하고
producer/stdout만으로 query 완료를 주장하지 않는다. 독립 ARTIFACT는 실제 자기 maps
거부를 확인했으며 [명시 계약 변경안](./p2b2-rust-artifact-runtime-decision-20261024.md)의
사용자 승인을 요청한 상태다. 승인 전 ARTIFACT maps/VM grant는 미적용이다.

Compiled case 존재와 정확한 선택을 확인한 Node 강제 query 입력의 실제3.98s는
PNPM CONTROL/DIRECT_EXEC_ROOT의 `/dev/null` read(flags557056/system `/dev`)에서 거부됐다.
기존 UV의 같은 read-only guard와 대조해 PNPM Root에만 연결한다. Node exec 자체는
아직 NOT_REACHED이며 ARTIFACT maps/VM 승인 대기 규칙은 바꾸지 않는다. 원본
p2b2-pnpm-startup-node-query-present가 first fault/완전한 cleanup/Host불변을 소유한다.

### Node 질의의 기능/권한을 분리한 재감사

Null 보완12 C++108 PASS 뒤 실제 명시 Node 입력4.10s는 lock9·완전한 관찰·의심0으로
PASS했다. 그러나 source 대조에서 `IsExpectedProcess`의 기존 CONTROL clone fallback이
unknown class도 expected group에 넣고, 공용 library read가 CONTROL role을 소비함을 확인했다.
따라서 이 PASS만으로 새 PNPM child가 ARTIFACT로 demote됐다고 말할 수 없다.
기존 npm/helpers 소비자를 재작성하거나 fallback 전체를 바꾸지 않는다.

| 범위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 common CONTROL child로 Node --version 실행 | YES | YES(새 actual 입력까지) | NO(새 PNPM 역할 경계) | NO | Node query child의 ARTIFACT demotion/expected group 제외·exact argv/parent·재실행/후손 거부 |
| 기존 UV exact query protocol — HasExactUVQueryCreator·ParseSentryClone/Process·UV latch | YES | YES | YES(기존 UV) | YES(기존 UV) | 같은 kernel creator 검증을 PNPM source class에 재사용 |
| PNPM exact version-only query 경계 | NO | NO | NO | NO | 아래 좁은 consumer와 RED/전체 latch·실제 query |

변경 전 검토: 공통 `HasExactUVQueryCreator` body는 같은 조건의 private helper로 기계적으로
추출하고 UV wrapper/조건을 보존한다. Source class는 UV/PNPM 두 known enum만 받는다.
PNPM은 Root clone에서 후보를 기록하되 첫 exec에서만 sealed helper와 immutable Node
`/usr/local/bin/node`, source `node --version`의 exact argv/execfn, 살아 있는 원래
parent/start/expected class와 모든 consumed/pending/network 상태를 검사한다.
성공 child는 ARTIFACT/Node로만 남고 expected CONTROL group과 network 권한을 얻지 않는다.
후보의 unsupported exec도 PNPM 전용으로 demote하여 기존 generic fallback으로 새 권한을
얻지 못하게 한다. OCI bootstrap/readiness shell child 및 다른 profile은 유지한다.
새 exec마다 query 상태는 취소되며 후손에게 후보/active를 복제하지 않는다.

새 positive·wrong binary/argv/parent/start/class·candidate 누락·network/pending·OCI/root/clone·
mount shadow/unsealed·reexec/descendant matrix가 현재 fallback에서 RED인지 먼저 확인한다.
Node query read 권한은 actual demoted-child fault를 확인한 뒤 필요한 immutable ABI만
추가한다. 이는 GHI ARTIFACT maps/VM 승인 대기와 독립이며 그 거부를 바꾸지 않는다.

변경 전 test-only 보완13은 새 Node matrix만 FAIL/기존108개 PASS34.705s로 기존 fallback의
실제 역할 gap을 확인했다. 원본 observer-13을 보존한다. 보완14는 기존 UV creator body를
동일 조건의 두 known source helper로 추출하고 새 PNPM clone candidate/first exec에서만
ARTIFACT로 demote한다. Unsupported 후보도 generic CONTROL fallback을 쓰지 못하게 한다.
Node 경로를 PNPM 일반 executable catalogue에 넣지 않으며 새 read 예외는 아직 추가하지 않는다.
M6의 GHI maps/VM 거부와 사용자 승인 대기는 그대로다.

사용자가 2026-10-24 명시적 M6 두 읽기 변경안을 승인했다. 보완15는 이미 관측한 GHI
ARTIFACT exact-self maps/flags557056만 적용한다. overcommit은 여전히 해당 ARTIFACT
NOT_REACHED이며 코드 거부를 유지한다. M6 current 계약과 과거 계획의 이력 경계도 갱신한다.

Node 보완14의 actual4.02s는 NODE/ARTIFACT/CLONE_CHILD·pinned executable로 demote된
실제 `/etc/ld.so.cache` read(flags557056/oci-root `/`)에서 거부됐다. 기존 canonical
immutable-loader ABI matcher를 새 exact Node-query active/creator guard 아래에서만
재사용한다. npm script/config/CA/proc/임의 library나 추가 flags를 허용하지 않으며 새
Node ABI matrix를 추가한다. Raw node-query-role이 실제 demotion/read fault를 소유한다.

보완15의 기존109 latch는 PASS지만 새 Node ABI matrix가 FAIL했다. Predicate는 허용했으나
연결 위치가 기존 helper-only chain이어서 ARTIFACT runtime-access count 기대를 충족하지
못한 wiring 오류다. 새 Node predicate만 기존 GHI runtime-root 분기와 같은 count 보존
경로에 연결한다. 기존 UV/CONTROL 분류나 보안 거부 기대를 낮추지 않으며 원본 observer-15를
보존한다. 보완16은 같은 허용 조건을 유지하고 이 새 소비자의 관찰 분류만 바로잡는다.

보완16은 전체110 latch37.375s PASS다. 같은 실제 CONTROL startup3.95s·빈 lock4.22s·
명시 Node query3.87s는 완전한 관찰/의심0으로 PASS했다. ARTIFACT5.24s는 execution
failure지만 observer fault가 없으며 overcommit read는 NOT_OBSERVED다. 따라서 VM grant를
추가하지 않고 기존 no-argument handoff의 실제 종료 결과를 test-only bounded diagnostic으로
확인한다. 첫 diagnostic은 내부 handoff mode를 찾았지만 실제 runner argv는 origin mode라
출력을 수집하지 못했다. Production/허용 조건을 바꾸지 않고 exact origin mode로 교정한다.
Raw native-approved-maps-wired 및 artifact-terminal-diagnostic을 각각 보존한다.

### ARTIFACT의 no-argument 종료 — 수정 전 감사와 두 번째 검토

Exact origin 진단5.20s에서 PNPM은 보안 fault 없이 Usage를 stderr에 출력하고 exit2였다.
미관측 VM read를 허용하지 않는다. 기존 GitHubELFBackend.Execute의 no-argument 계약과
TerraformProviderBackend.probeTerminal의 고정·discard·full observation 경로를 대조했다.
ReleaseSetInspector는 실제 native 전부를 기존 Composite/DynamicInspector로 보내지만
PNPM의 정상 probe 인자를 선택하는 production 연결은 없다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 GHI ARTIFACT handoff·격리·관찰/정리 | YES | YES | YES(기존 native4) | YES(기존 native4) | PNPM 고정 probe 소비 |
| 고정 discard probe callback — Terraform | YES | YES | YES(기존 provider) | YES(기존 provider) | 같은 메커니즘 재사용 |
| HAA native PNPM의 --version ARTIFACT probe | NO | NO | NO | NO | exact member/pin 선택·bootstrap wiring·negative·actual |

수정: sandbox/heliopause_native.go의 실제 전용 wrapper에서 github-release/
rahoney-heliopause/current generated PNPM asset·native SHA/size가 모두 맞는 member만
기존 executeELF callback으로 /work/artifact --version을 RunDiscard한다. 역할은 github-elf/
ARTIFACT이고 pin은 probe 선택 조건일 뿐 clean 판정/CONTROL authority가 아니다. stdout을
해석하지 않고 exit0·완전한 관찰·기존 required dynamic/Policy·정리 전체를 유지한다.
다른 HAA native는 기존 no-argument backend를 그대로 소비하며 같은 PNPM 이름의 다른
source/name/hash/size는 version probe를 얻지 못한다. generic GitHub CLI는 기존 noargs다.

Bootstrap self_update_linux.go의 자기 native inspector만 새 wrapper에 연결하고 Core·Port·
ReleaseSetInspector·generic GitHub·Terraform 프로토콜은 변경하지 않는다. Test fixture는
실제 HAA member identity와 같은 production constructor를 사용하고 임시 terminal logger는
원인 기록 뒤 제거한다. 기존 command/role 및 bad binding·nonzero/observer incomplete·cleanup
실패를 unit/actual로 구분한다. 새 argv 때문에 후속 실제 read가 나타나면 원본을 보존한다.
두 번째 검토: help exit2를 성공으로 재분류하거나 arbitrary user probe option을 받는 변경이
아니며 기존 고정 callback 소비 하나의 추가다. VM은 계속 NOT_OBSERVED/DENIED다.

고정 probe actual5.59s는 VM_OVERCOMMIT_MEMORY·ARTIFACT/DIRECT_EXEC_ROOT·flags557056/
system `/proc`에서 처음 거부됐다. 사용자 승인한 실제 관측 조건이 충족되어 보완17은
exact overcommit 한 경로만 같은 GHI guard에 연결한다. Maps/VM 각각의 flags·stale/role/
profile/mount negative를 동일 matrix로 검사한다. Raw artifact-fixed-probe를 보존한다.
앞 Quick17.160s는 새 fixture의 존재하지 않는 domain 생성 API, 다음28.361s는 새 error
문자열 대문자를 고정 검사기가 탐지했다. 기존 SHA256 API/소문자 메시지로 교정했으며
이 두 cheap failure를 실제 runtime 권한 오류와 혼합하지 않는다.

### 동일 native 경계의 원격 연결 전 감사

보완17 전체110 latch40.735s PASS, 같은 final namespace 실제7개28.532s는 모두 PASS다.
PNPM startup4.26s/empty lock4.36s/Node query4.27s/독립 ARTIFACT5.39s와 UV startup4.37s/
query4.47s·기존 GHI /bin/true1.38s가 각 실제 실행됐다. Complete·의심0/Host불변/
private daemon wait0/잔여0이다. 같은 Go 범위의 관련3package full Race22.297s PASS다.
현재 CI는 UV actual과 공통 소비자를 호출하지만 PNPM 준비/실제4case 호출은 없다.

CI 수정은 현재 lifecycle step에서 scripts/ci-pnpm-runtime.sh 한 명령을 추가한다. 이 새
실제 소비자는 기존 lock/generated-check·bounded curl·materialize-pnpm을 재사용하여 공식
archive/proof/native/NOTICE를 검증한다. 고정 Root current 공급은 빈 hosted CI에만 임시
생성하고, 기존 보호 경로가 있으면 거부한다. HAA release 설치/활성화 성공으로 표시하지 않는다.
Four opt-in actual은 일반 UID로 기존 observer·gVisor를 소비하고 종료 trap에서 exact Root
files/directories 및 자기 mktemp만 제거한다. Payload를 Host에서 실행하거나 CI namespace의
기존 설치를 덮지 않는다. 기존 shared lifecycle/helper restore trap은 그대로다.

Checker는 같은 active lifecycle 명령과 helper의 exact actual4case 명령을 함께 검사한다.
Comment/conditional step/삭제·skip·Node/ARTIFACT 누락을 negative로 검사하고 기존 command
validator/actionlint를 재사용한다. Same pin 값은 lock 하나에서 읽고 workflow에 version/hash를
중복하지 않는다. Supply shell helper 자체는 fixed materializer/검사 command 연결과 bash syntax,
실제 local namespace의 같은 명령군/정리로 검증하며 hosted whole-result는 원격 결과가 필요하다.
두 번째 검토: 단순 legacy CI green을 새 PNPM actual qualification으로 확대하지 않으며
이 native admission checkpoint 이후에도 public graph/SRI/layout/CLI와 최종 P2가 남는다.

CI helper 실제 frozen replay27.636s는 official materialization과 PNPM 실제4개를 통과했다.
기존 설치 sentinel을 보존해 거부했고 자기 Root 공급과 임시 payload가 사라졌으며 private
daemon wait0/자원0/Host불변이다. Public HTTPS 다운로드는 재생했으므로 이 로컬 결과로
그 실행을 주장하지 않는다. Raw p2b2-pnpm-startup-ci-helper가 범위를 소유한다.
CI-connected 첫 Quick43.172s는 기존 future-owner fixture가 새 helper 입력을 복사하지
않아 실패했다. Checker 조건을 낮추지 않고 같은 repository helper를 fixture 입력에 추가했다.
