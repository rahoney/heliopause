# P1b5 — UV 전용 실행 공간·admission 수정 지도

기준 c5d597a41563fc214c41921d159ccb1824db10e4. M13-004/P1만 IN_PROGRESS다.
Read: [P1b4](./p1b4-uv-provision-baseline-20261021.md),
[설계](../../../user-journey-cli-ia/09-package-client-quarantine.md),
[D-012/D-015](../../../threat-model/04-trusted-tooling-and-evidence.md).

| 책임 / 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| UV 인증 공급·data transport / uv_tool_supply.go | YES | NO(production) | NO | NO | running target·실제 native 실행 |
| 기존 Python create·helper demotion / pypi_resolver.go, boundary_helper.go | YES | YES | YES(기존 scope) | YES(기존 scope) | UV-only64MiB mount |
| 기존 observer topology·one-shot nonce / shared_observer.go, direct_exec_admission.go, observer.cc | YES | YES | YES(기존 scope) | YES(기존 scope) | UV-only profile/class·helper mount image 검사 |
| UV production resolver/CLI | NO | NO | NO | NO | native graph 전체·transaction·CLI; 이번 최소 runtime 연결 후 계속 |

## 수정 위치·방법

1. pypiCreateArguments의 기존 인수/출력은 보존하고 내부 고정 mount 인수를 받는 작은
   owner로 추출한다. uv 전용 sibling만 /haa-runtime64MiB를 선택한다. 기존 helper script,
   Python image·UID·capabilities·tmp/site·CPU/memory/time은 유지한다. 외부 mount 문자열
   설정이나 기존 pip1MiB 확대는 없다.
2. shared_observer.go에 uv-pip-resolver의 기존 Python 모양 topology를 명시 등록한다.
   isPythonObserverProfile에는 추가하지 않는다. Closure/ledger/PyTorch classification과
   기존 budget은 유지한다. 새 profile은 기본 finite collector ceiling을 사용한다.
3. observer.cc의 registration/canonical bootstrap whitelist만 새 profile을 추가한다.
   IsPythonProfile은 그대로 둔다. 새 UV class는 /haa-runtime/uv exact path만 분류하며
   외부 network record는 기존 OTHER vocabulary로 기록한다. UV 실행은 sealed namespace의
   정확히 하나 helper tmpfs에만 연결하고 nested shadow·다른 profile·다른 path는 거부한다.
   기존 one-shot root/demotion 경로를 그대로 사용한다. ARTIFACT/clone는 CONTROL이나
   trusted network로 승격하지 않는다. 새로운 file/runtime read는 관측 근거 없이 추가하지 않는다.
4. C++ latch는 UV path/profile·sealed topology·shadow·ARTIFACT role·nonce/demotion
   연결을 검사한다. Go는 기존 create의1MiB 보존·UV64MiB·환경 동일성·closure 비가입을
   확인한다. Canonical format/Quick/security/docs와 pinned C++ builder 전체 latch를 실행한다.
5. 실제 native probe는 새 qualified bundle/observer가 Docker daemon에도 보이는 전용
   환경에서 수행한다. Client-only private mount namespace의 B4 결과를 실행 증거로
   전용하지 않는다. Host 설치·서비스 활성화 없이 기존 owned guest/daemon pattern을
   재사용한다. --version 이후 실제 필요한 read는 최초 fault를 보존하고 별도 근거로 좁힌다.
6. Shared observer 소비자는 npm·wheel·sdist·GitHub·기존 promotion과 네 Python profile을
   포함한다. 로컬 최소 latch 성공을 이 matrix나 full CPU/CUDA qualification으로 확대하지 않는다.
   기존 Go1.26.10 원격 cu126 관측과 macOS timeout도 별도로 보존한다.

## 두 번째 검토

UV basename/PATH를 신뢰하지 않는다. Hash가 맞는 공급도 target topology·실제 runtime
admission을 대신하지 않는다. 신규 profile을 IsPythonProfile에 통째로 넣어 기존 broad
reads/closure 권한을 상속하지 않는다. 새 native startup이 실패하면 unknown event를
삭제하거나 limit을 늘리지 않는다. Raw/작은 receipt만 보존하며 완료된 C++ build/source/
guest payload는 참조·query errors0 확인 후 삭제한다. Production CLI 지원 완료는 별도다.

### 실제 startup 환경의 구체화

최소 startup은 systemd/network helper가 필요 없는 network-none scope다. 새 privileged
systemd guest 대신 private mount namespace에 전용 foreground dockerd를 실행한다.
Namespace 안의 /run·/etc/heliopause와 canonical bundle/observer mount만 분리하고 daemon과
ordinary UID1000 test가 같은 new bundle을 보게 한다. Docker data/exec root·pid/socket은
전용이며 bridge/iptables/ip6tables/ip-forward/ip-masq는 모두 비활성화한다. Host daemon
등록·서비스·네트워크 규칙은 변경하지 않는다. 공급 이미지는 private daemon에 exact digest로
획득하고 uv target은 network none이다. 별도 data-only client namespace와 다른 실제 실행이다.
Foreground daemon의 parent wait·container inventory·namespace 종료·Host 보호 파일/기존
image 불변을 확인한다. Query 실패면 정리 성공을 선언하지 않는다. 이 환경은 향후 network
resolver/full install matrix나 systemd helper qualification을 대신하지 않는다.

### 실제 반증과 수정 지도 보완

기존 tmpfs64MiB 후보는 canonical Quick/security/docs와 C++104 latch가 PASS였지만
실제 running target의 introduce에서 실패했다. Data-only stopped writable-root의 B4
성공을 전용하지 않았다. 전용 daemon의 exact 공급 bytes로 bounded Docker 진단을
추가한 결과 `destination ...:/haa-runtime must be a directory`를 확인했다. Guest-only
tmpfs를 daemon archiver의 동일 mount로 가정한 새 연결 설계의 결함이다. UV 실행은0,
매 시험 trace complete·container0·observer endpoint0·daemon wait0·Host 불변이다.
원본 uv-native-actual-{first,diagnostic,copy-diagnostic}가 단계별 범위를 소유한다.

기존 동작을 대체하지 않고 UV-only helper 공간을 Docker local driver의 named tmpfs로
구체화한다. 새 uv_runtime_volume.go는 기존 dockerVolumeInspection/containerMount
자료형을 재사용하며 unique name/label/tool hash·성공한 사전 부재 조회·driver/options/
CreatedAt/mountpoint·actual target의 exact single mount를 검증한다. 크기64MiB,
root0:0/0755/nosuid/nodev이며 Host path bind는 없다. 기존 Python wrapper의 --tmpfs1MiB는
보존하고 UV sibling만 --mount의 private volume을 선택한다. 새 volume은 tool/고정 helper
전용이며 Python closure volume의 uid1000/artifact 입력 의미를 바꾸지 않는다.

Named tmpfs는 gVisor에서 기존 closure와 같은9p topology로 나타난다. UV profile만
/haa-runtime helper expectation을9p로 바꾸며 Host tmpfs flags는 volume options에서 별도로
검증한다. C++ UV-only exact image predicate도 sealed helper9p·shadow 부재를 요구한다.
기존 IsPythonProfile과 helper demotion/nonce/ARTIFACT 권한은 유지한다. 별도 Root exec
mode나 artifact 선택 파일 설치 권한을 추가하지 않는다. Docker cp --archive와 source/
target root0:0·0555·hash는 같은 공급 owner를 그대로 쓴다. Volume은 container 제거 후
부착 부재·동일 신원을 확인해서만 삭제한다. 실패를 정당한 UNKNOWN 없이 rm/prune하지 않는다.

실제 native fixture에 volume creation/attachment 재확인/마지막 cleanup을 연결하고 실패
진단용 직접 Docker 호출·tar dump는 최종 test에서 제거한다. Unit의 substitution/options/
foreign attachment/failed query·기존 mount 보존과 C++ 전체 latch를 새 후보에서 다시 검사한다.
그 뒤 동일 private daemon의 actual startup을 재검증한다. 이 수정 전 실패를 삭제하거나
기존 stopped transport를 잘못된 해결 방법으로 기록하지 않는다.

### 시험 daemon의 네트워크 격리 누락·복원

초기 시험은 mount만 분리하고 Host network namespace를 공유했다. Bridge/iptables 등을
끄는 flag만으로 Host 네트워크가 보존된다고 가정하면 안 된다. 세 시험 뒤 기존 Host
daemon의 bridge 설정은 남았지만 실제 docker0가 사라져 다음 C++ builder가 veth 부착
이전에 실패했다. 정확한 삭제 syscall은 수집하지 않았으며 UV source의 원인으로 쓰지 않는다.
성공했던 파일/image 불변 receipt는 네트워크 불변까지 확인한 증거가 아니다.

성공한 network/전체 link/iptables inventory와 container0을 보존했다. 기존 daemon이
소유한 bridge IPAM·MTU 그대로 docker0를 복원하고, 고정 builder /bin/true의 기본 bridge
실행0과 Docker MainPID1829/active 불변을 확인했다. 서비스 재시작·전역 prune은 없다.
후속 operator는 반드시 mount+network namespace를 함께 unshare하고 /proc/1과 다른
netns를 직접 확인한다. 공급 exact images는 이미 전용 daemon state에 획득되어 있으므로
offline으로만 재사용하며 Host network fallback은 없다. Before/after는 Host bridge 주소도
포함한다. 원본 uv-native-host-network-{first-failure,restoration}.json과 두 C++ build log를 보존한다.

### Named volume 실제 결과와 startup read 수정 위치

새 후보는 canonical Quick/security/docs/workflow와 C++104 latch가 PASS다. Mount+netns가
실제로 분리된 offline daemon에서 volume identity·exact single attachment·root0:0/0555
UV 전체 hash의 running target 전달/재검증은 통과했다. 첫 UV 실행은 CONTROL/
DIRECT_EXEC_ROOT의 system/proc `VM_OVERCOMMIT_MEMORY`, flags557056에서
OPEN_RESULT_CLASSIFICATION_PROC로 차단됐다. UV의 partial version 실행을 성공으로
쓰지 않는다. Target/volume/endpoint0·daemon wait0·Host bridge 포함 불변은 확인됐다.

observer.cc의 새 UV-only predicate에서 실제 관측된 /proc/sys/vm/overcommit_memory
한 파일만 고정 read flags·system /proc anchor와 이미 kernel 분류된 UV 직접 CONTROL
group에 연결한다. 정확한 generation·consumed root·demotion/launch/handoff pending 부재·
active trusted-control 상태와 FilesystemProcessClass를 확인한다. ARTIFACT/clone/다른
profile·writable open·다른 proc 파일/다른 mount·stale image는 거부한다. 기존 Python/
Cargo/Go runtime predicates는 변경하지 않는다. 같은 새 C++ latch의 정상/negative와
전체 latch 후 actual startup에서 다음 최초 fault를 확인한다. 추측한 maps/cgroup 등은
미리 추가하지 않는다. 원본 uv-volume-native-actual-first와 frozen build receipt가 소유한다.

Native 최초 fault별 작은 변경을 검증하는 동안 전용 C++ build/cache를 재사용한다. 고정
canonical script의 recipe·source preparation·Bazel/image/hash는 그대로 사용하며 private
operator의 유일한 초기 차이는 성공 후 active workroot 보존이다. 재컴파일은 그 same
prepared source에서 관찰기 두 파일만 byte 대조 후 갱신한다. 완료 후 참조 확인하여 전체
source/cache를 삭제하며, 보존한 active workspace를 완료된 시험 자료로 무기한 남기지 않는다.

## 최소 runtime의 로컬 결과

[로컬 영수증](./p1b5-uv-runtime-local-20261021.json)의 새 C++105 latch와 실제 동일
관찰기/qualified1.26.10 bundle은 uv0.13.4의 native --version을4.93s에 완료했다.
Running root volume·tool hash·one-shot demotion·external trace29/complete와 마지막
container/volume/observer endpoint0·foreground daemon wait0을 확인했다. Mount/netns는
Host와 분리됐고 Host bridge 포함 before/after 신원이 같다. 실제 관측한 overcommit
read 하나 외의 추측 permission은 추가하지 않았다.

Go/CI 후보는 canonical Quick62.043s/security8.949s/docs0.259s/workflow0.911s가 PASS다.
기존 Linux CI의 unconditional step에 UV pin 읽기/pull과 같은 opt-in actual startup test를
연결했다. 누락 pull·환경값0을 검사하는 기존 workflow validator 회귀도 통과했다.
새 job·timeout/예산 확대는 없다. 새 전체 source의 원격 Required/기존 소비자는 pending이다.
Production UV resolver·graph·install/upgrade/sync·CLI 지원은 아직 미구현이며 P1은 열려 있다.

최종 검토: source의 새 연결은 UV-only profile·Host-verified root volume·
exact native path와 한 개의 관측된 read에 한정된다. 기존 Python profiles·closure/권한·
resource ceilings는 유지됐다. 전체 staged 후보의 canonical Quick37.510s와 나머지
profile 결과는 로컬 영수증의 final_checks를 따른다. 완료 native 시험의 전용 daemon/data·
Root runtime 사본·test binary/이전 observer6,869항목/unique regular858,952,195B와
빌드 전 환경 실패 자료5,027항목/103,028,353B를 참조/query errors0 확인 후 삭제했다.
실제 다음 resolver 작업에 필요한 source/SDK/quality cache·qualified base/new observer와
활성 C++ build/cache만 보존한다. 원격 CI와 production adapter가 남으므로 전체 지원은 미완료다.
