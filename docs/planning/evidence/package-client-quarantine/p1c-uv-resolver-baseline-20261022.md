# P1c — UV native selection·guarded 설치/업데이트 수정 지도

기준 e25fd979e3baedf25fc79ad07d2de35ab3cf4cae. M13-004/P1만 IN_PROGRESS다.
[P1b5 영수증](./p1b5-uv-runtime-local-20261021.json)의 actual native startup과 C++105
latch를 로컬에서 확인했다. 같은 source의 CI368/run39622548139는 Linux native UV
startup3.25s·기존 integration·CPU259.85s가 PASS다. macOS default-test6m timeout으로
Required는 FAIL이며 신규 production UV resolver/CLI 지원은 아직 아니다. 기존 구현을
대체하거나 보안 invariant를 바꾸지 않는다.

Read: [client 설계](../../../user-journey-cli-ia/09-package-client-quarantine.md),
[전체 계획](../../20-package-client-quarantine-plan.md), [Artifact Port](../../../domain-model/07-contract-artifact-port.md),
[M8](../../11-m8-production-trust-hardening-contract.md),
[trusted tooling](../../../threat-model/04-trusted-tooling-and-evidence.md),
[update 계약](../../../user-journey-cli-ia/06-update-quarantine.md).

## Baseline audit

| 책임 / 기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| PyPI pip lifecycle·typed policy / sandbox/pypi_resolver.go, resolver network policy | YES | YES | YES(기존 scope) | YES(기존 scope) | 기존 동작 재구현 없음; UV selection sibling만 추가 |
| native pylock/선택 wheel/완전 archive·RECORD/공식 release metadata / artifact/pypi/uv_*.go | YES | NO(production) | NO | NO | actual resolver producer·frozen input 연결 |
| UV signed/native 공급·root volume·one-shot startup / sandbox/uv_tool_supply.go, uv_runtime*.go, observer.cc | YES | YES(native fixture) | YES(startup local/remote only) | NO(full resolver) | 실제 pip 명령·필요한 narrow query/read·macOS CI 진단 |
| guarded install/update·Policy/staging / application/install*.go, promotion/pypi*.go | YES | YES(pip) | YES(기존 scope) | YES(기존 scope) | UV candidate·runtime identity/controls·actual promotion 연결 |
| 다중 요구사항/empty complete set 모델 / domain/project_snapshot.go, ports/ports.go, application/project_{snapshot,update}.go | YES | YES(기존 project 소비자) | YES(기존 scope) | YES(기존 scope) | UV sync control/parser·Python transaction adapter |
| UV native install/upgrade/sync product adapter·CLI descriptor | NO | NO | NO | NO | 아래 단계와 normal/hostile/rollback·old 소비자 회귀 |

LockedDependencyGraph는 정확히 하나의 실제 primary와 connected closure를 요구한다.
requirements 전체 sync를 위해 가짜 primary/edge를 만들거나 그 invariant를 느슨하게 하지
않는다. 여러 독립 root와 빈 requirements는 기존 ProjectDependencySnapshot 및 명시
dependency-free 모델을 사용한다. Python adapter/transaction의 구체 연결은 별도 검증한다.

## 입력 조사에서 먼저 확인할 사항

이미 저장된 exact uv0.13.4 native help에는 install의 --dry-run/--output-format json이
있다. --report가 없다는 사실만으로 기계 출력이 없다고 결론 내리지 않는다. P1a는 실제
pylock/표준 WHEEL/전체 archive 경계를 구현했으며 이를 버리거나 재작성하지 않는다.
새 native JSON의 exact schema·실제 선택 URL/hash/complete-set 제공 범위를 확인한 뒤
필요한 곳에만 보조 자료로 쓴다. Tool JSON/INSTALLER/exit0 자체는 ALLOW authority가 아니다.

[공식 resolution 문서](https://docs.astral.sh/uv/concepts/resolution/)는 pip interface의
platform-specific 선택과 target Python/platform 옵션을 설명한다. [CLI 문서](https://docs.astral.sh/uv/reference/cli/)
는 JSON schema가 실험적임을 명시한다. 현재 웹 문서를 pin의 실제 출력 대신 사용하지
않고 exact0.13.4 실행·공급 bytes와 함께 확인한다. Native query의 actual argv/역할·경로는
관찰 전에 추정 권한으로 추가하지 않는다. 다른 source/managed Python/build/임의 index는
이 작업에서 자동 활성화하지 않는다.

## 정확한 수정 위치·순서

1. Native production 동작 조사: owned full qualification 환경에서 pinned Python3.14.7,
   uv0.13.4, qualified Go1.26.10 bundle와 현재 observer를 사용한다. Typed network policy의
   actual acquire/bind/release·ordinary UID와 supervisor를 재사용한다. Guest/daemon은
   Host와 network/mount/cgroup 수명이 분리되어야 하며 이미 반증된 mount-only operator는
   재사용하지 않는다. Exact images·Root helper/사용자 identity·정리 전 inventory를 먼저 기록한다.
2. sandbox의 좁은 UV resolver sibling에서 기존 named endpoint resolver,
   ResolverNetworkPolicy, startTrace/awaitMountAnchors, admissionAwareRunner,
   uvRuntimeVolume/prepareUVTool과 bounded I/O를 사용한다. 기존 pip/PyTorch resolver
   전체를 복사하거나 일반 실행 엔진으로 재작성하지 않는다. 공유 추출이 필요하면 실제
   두 소비자의 같은 lifecycle 구간만 이동하고 기존 분기의 정상/실패·cleanup을 재검증한다.
3. Native selection은 fixed source/target·wheel-only·no-build/no-config·managed Python
   download 금지·private cache/copy mode로 고정한다. 사용자 shell/임의 native argv를 받지
   않는다. Python interpreter query/metadata read는 실제 첫 fault와 pinned tool 동작을
   확인한 뒤 exact profile/image/role/creator/argv/mount/flags에만 연결한다. Child에 CONTROL/
   network authority를 상속하지 않으며 IsPythonProfile을 통째로 확장하지 않는다.
4. 실제 producer의 pylock/표준 WHEEL 선택과 개별 wheel archive를 기존 UVLock API에
   연결한다. Independent exact-release 전체 JSON·identity/URL/SHA/size·Requires-Dist closure를
   검증한다. Filtered whole-project Simple, 임의 첫 wheel, 출력 이름만으로 graph를 만들지
   않는다. 필요한 intake reader는 existing Root/bounds/digest owner를 재사용한다. 각 자료의
   한도와 최종 graph digest/runtime identity를 하나의 frozen resolution에 묶는다.
5. Install은 기존 InstallService의 complete-set 검사/Policy/staging/guarded promotion을
   사용한다. Upgrade는 pypi/upgrade_input.go의 inert installed METADATA와 원래 guard를
   재사용하되 UV native의 선택 의미를 pip only-if-needed로 몰래 대체하지 않는다. Exact
   old/new 선택·retained root/전체 graph·native output control을 먼저 정의하고 변조/누락을
   거부한다. User target interpreter/.pth/실행 코드를 resolver 입력으로 복사·실행하지 않는다.
6. Sync는 bounded requirements grammar·source option 거부·source file digest와 target
   snapshot을 연결한다. Existing ProjectDependencySnapshot/guarded project workflow를
   재사용하고 Python의 기존 file ownership/backup/rollback에 같은 staged set만 전달한다.
   제거는 승인된 소유 파일에 한정한다. Input 변경·foreign package/file·동시 mutation·
   dependency-free와 marker/중복/충돌을 확인한다. CLI에 가짜 단일 package sync를 노출하지 않는다.
7. cli/root.go와 bootstrap/bootstrap.go의 실제 command/factory owner에서 source와 client를
   구분한다. uv pip install·install --upgrade·sync를 하나의 좁은 typed descriptor로 연결하고
   기존 pip/npm factory·lazy help/doctor를 유지한다. 새 capability가 실제 wired/qualified일 때만
   launcher/help/7개 언어 설명에 표시한다. 불지원 flag/backend는 composition/network 전에 거부한다.
8. Canonical format/Quick/security/docs/workflow와 변경 owner의 negative/old 회귀를 우선
   검사한다. 실제 native JSON/pylock·registry 교차검증·install/upgrade/sync/no-op·악성 wheel/
   incomplete observation·rollback/취소/cleanup을 final source에 연결하고 기존 Linux CI에
   필요한 회귀를 추가한다. SDK·CPU/wall·최대 sample gap·Required를 완화하지 않는다.

## 두 번째 검토와 아직 열린 조사

단일 요청 install과 전체 requirements sync를 같은 graph constructor에 억지로 맞추지
않는다. UV JSON은 pin의 실제 자료 범위를 확인하기 전 parser 설계를 확정하지 않는다.
Python query를 추측해서 broader profile에 포함하지 않는다. 기존 Policy/검사·staging·
rollback을 재작성하지 않고 client 선택·frozen controls·native layout의 gap만 추가한다.

아직 확인할 사항: exact native JSON 내용, interpreter discovery/query, 설치 파일의 native
선택/upgrade semantics, multi-root control/transaction 연결 세부. 조사 결과가 달라지면
여기 기존 항목을 수정하고 기존 구현을 대체해야 할 경우에는 위치·이유·영향을 먼저 제시한다.
모든 목표가 완료되기 전에는 P2로 넘어가거나 public UV CLI 지원을 선언하지 않는다.
완료된 owned test data는 query errors0·참조 부재 후 삭제하고 raw/작은 영수증만 보존한다.

### C1a 기계 출력 형식의 별도 조사 범위

Native JSON schema를 먼저 확인할 작은 fixture는 인증된 UV bytes와 같은 Python image를
사용하는 전용 read-only/unprivileged/network-none runc container에서 수행한다. 동일 fixed
image의 never-start carrier에 UV 한 파일을 복사한 owned derived image를 사용하며 build
cache/Host bind는 만들지 않는다. 공식 exact wheel을 데이터로만 전달하고 install/dry-run/
upgrade/sync JSON을 비교하며 target package를 import/실행하지 않는다. 이 조사 결과는
production gVisor/typed network/검역 성공이 아니다. 이후 actual qualification은 위 C1의
전체 trusted environment에서 별도로 진행한다. JSON이 delta-only이거나 URL/hash/closure가
없다면 그 한계를 보존하고 기존 P1a graph/metadata 경계를 그대로 사용한다. 완료한
container/derived image는 exact owner를 확인하고 no-prune로 제거하여 shared image를 보존한다.

## C1a 실제 결과

[Exact native JSON 영수증](./p1c-native-json-discovery-20261022.json)의4 cases가 exit0이다.
Schema.version은 preview이며 changes에 name/version/action과 dry_run만 있다. 24.2의
dry-run/install,24.2→25.0 upgrade의 remove/install과 same-version changes=[]를 확인했다.
URL·SHA·dependency edges·변경 없는 설치 전체는 출력하지 않는다. 이 자료로 graph/
ALLOW를 만들 수 없으므로 기존 P1a pylock/WHEEL/archive/independent metadata 경계를
그대로 production에 연결한다. JSON option의 존재를 무시하거나 대체 구현을 만들지 않는다.
Sync·production network/observer·복수 dependency upgrade semantics는 아직 미검증이다.

전용 runc container3/owned derived image1을 exact owner 확인 뒤 no-prune로 제거했고
Host 보호 파일·기존 image records·bridge는 before/after 동일하다. 실제 UV native hash는
pin과 같다. Target package는 import/실행하지 않았다. 이 성공은 형식 조사에 한정되며
production gVisor·typed resolver policy qualification으로 전용하지 않는다.

### Promotion identity/control 소비자의 추가 audit

현재 promotion/pypi_update.go의 selected-control 경계는 pip JSON delta/report 세 개와
python/pip/target/source 문자열을 정확히 재검증한다. pypi_native_ownership.go는 pip
INSTALLER/REQUESTED 표면을 검증하고, pypi_promotion.go의 offline wheel materialization도
pinned pip다. UV 결과를 pip 보고서로 위장하거나 새 identity를 broad prefix로 허용하지 않는다.

구체 보완은 한 곳의 typed client/runtime·frozen-selection 검증 adapter로 solver identity와
실제 반출 installer identity를 연결하는 것이다. 기존 pip 분기는 같은 bytes/계약을 유지한다.
UV sibling의 pylock/selection controls와 digest/graph binding을 독립적으로 검증하고, 동일
venv transaction/ownership 검사는 재사용한다. Native UV adoption의 marker/file inventory는
실제 pin에서 별도 확인하며, INSTALLER 값 하나를 출처/안전성 authority로 사용하지 않는다.
기존 parser/transaction을 대체하거나 old signed state를 재작성하지 않는다.

### C1b offline interpreter query 선행 관측

Production network resolution 전에 exact UV가 고정 Python을 조회하고 설치 metadata를
읽는 최소 동작을 별도로 관측한다. 기존 uv_runtime_real_test.go의 검증된 startup
fixture를 test-only callback으로 공유하고 uv pip list의 fixed Python·offline·no-config·
no-cache·no-download·JSON 명령만 추가한다. Host Python/사용자 target/.pth는 사용하지
않고 pure Python image의 pip identity와 완전한 외부 trace를 검사한다. 이 시험은 실제
resolver 선택/typed network/검역 성공이 아니며 production CLI를 열지 않는다.

새 전용 mount+network namespace의 독립 foreground daemon에 exact Python/UV images를
offline 전달한다. Namespace ID 불일치를 확인하고 기존 성공 recipe의 Root read-only
runtime/observer view·ordinary UID1000·one-shot admission을 유지한다. Host daemon/service,
보호 파일·image records·bridge before/after를 대조한다. 첫 fault가 나오면 원본을 보존하고
profile/image/role/creator/argv/mount/flags의 exact 원인만 검토한다. 예외를 추측해서 넓히지
않는다. Full production resolver의 cgroup/typed policy qualification은 앞 C1에서 별도로
수행한다. 종료 뒤 container/volume/endpoint·daemon wait를 확인하고 완료 data는 삭제한다.

C1b의 actual first query는 UV CONTROL/DIRECT_EXEC_ROOT의 system /proc own-PID cgroup
read(flags557056)에서 STREAM_FAULT였다. Native inventory는 실패했고 interpreter/graph
성공이 아니다. Container/volume/observer endpoint0·daemon wait0을 확인했다. 별도 최초
operator의 image save/load는 config/layers만 보존하고 digest 참조를 잃어 test 실행 전에
중단됐다. 종료된 private daemon에서 Host의 인증된 두 reference→config binding만 frozen
재현하고 config/RootFS/architecture/OS를 대조한 후 실제 query를 시작했다. Host image DB를
수정하거나 tagged/latest fallback을 사용하지 않았다.

첫 보완은 observer.cc의 기존 IsPinnedUVRuntimeRead에 현재 kernel TGID의 /proc/<pid>/cgroup
정확한 read만 추가하는 것이다. 기존 UV-only profile·sealed image/current generation·소비된
one-shot CONTROL root·system /proc mount·flags557056·exec parent0 조건을 유지한다.
다른 PID·unresolved self alias·maps/environ·쓰기·ARTIFACT/clone·pending/stale·다른 profile은
계속 거부한다. 기존 UV runtime-read latch의 정상/negative matrix를 두 allowed path 모두에
적용하고 실제 full pinned C++ recipe 뒤 같은 fixed inventory를 관측한다. Mountinfo/quota 등
다음 read는 아직 근거가 없으므로 이 변경에 미리 허용하지 않는다.

Own-cgroup 보완의 pinned C++105 latch는 PASS41.46s였다. 동일 actual 조회의 다음 최초
fault는 같은 UV root/system /proc/flags557056의 own-PID mountinfo다. 앞 cgroup read와
trace/container 정리는 통과했으나 inventory 성공은 아니다. 이제 같은 predicate와 세 경로
전체 negative matrix에 현재 TGID의 mountinfo만 추가한다. 다른 PID의 mountinfo와 별도
sysfs/quota 경로는 계속 거부하며 다음 fault를 관측한다.

Own-mountinfo 보완의 C++105는 PASS35.469s다. 다음 actual fault는 같은 UV root의
system CGROUP_CPU_QUOTA/flags557056이다. Bounded diagnostic locator5865f84329325e30을
known candidate와 대조하면 /sys/fs/cgroup/cpu/cpu.cfs_quota_us이며 mountpoint locator는
/sys/fs/cgroup/cpu와 대응한다. Locator hash는 진단 대조용이지 인증 authority가 아니다.
기존 UV predicate의 profile/image/role/generation 조건을 유지하고 이 exact system mount의
quota read만 추가한다. 기존 proc 세 경로와 새 quota 경로의 모든 negative를 검사하고
다른 sysfs·period·다른 PID는 계속 거부한다. Python query/child는 아직 승인하지 않는다.

Quota-only 보완의 C++105 PASS35.017s 뒤 actual 조회는 같은 system mount/flags의
locatoraea5f4520ac7eba5에서 거부됐다. Known exact cpu.cfs_period_us와 대응하며 앞 quota
경로와 구분했다. Container/volume/endpoint0·daemon wait0·Host 전후 동일이다. 같은
predicate에 이 period 한 경로만 추가하고 proc 세 경로+quota/period 전체의 flags·mount·
class/role/generation·타 profile 부정을 재검증한다. 다른 cgroup 파일은 계속 거부한다.

Period 보완의 C++105 PASS37.312s 뒤 다음 actual fault는 UV root의 DEV_NULL,
system mount locator0be3699c9bd6bd65(/dev), flags557056이다. 같은 strict root 조건에서
/dev/null의 정확한 read만 추가한다. /dev/zero·urandom·쓰기·다른 mount/역할은 거부한다.
이것은 query process에 대한 승인이 아니며 단순 stdout/exit0로 검역 완료를 만들지 않는다.

### 자동 승인 검토로 중단된 후보

2026-10-22 /dev/null read 후보를 포함한 다섯 번째 pinned C++ build 요청이 자동 승인
검토에서 거부됐다. 반복된 privileged runtime 시험과 observer 예외 누적 위험이 이유다.
해당 요청은 실행되지 않았으며 이 후보의 build/latch/actual inventory는 NOT_TESTED다.
권한 우회·동일 요청의 간접 실행을 하지 않는다. 기존 actual 결과는 앞 네 C++ 후보와
다섯 first-fault 원본 범위까지만 유지한다. Native inventory/resolver/CLI acceptance는 열려 있다.

재개 판단 대상은 새 generic runtime grant가 아니라 IsPinnedUVRuntimeRead의 동일 sealed
UV CONTROL root·exact flags/system /dev·/dev/null read 한 경로다. 다른 role/clone/profile,
쓰기와 임의 device는 계속 거부하는 구체 후보와 negative matrix를 review한다. 추가 실행은
사용자 결정 뒤 진행한다. 독립적인 macOS 진단과 완료 data 정리는 계속할 수 있다.

2026-10-22 사용자가 위 `/dev/null` read 후보의 빌드·격리 검증을 명시 승인했다.
중단 뒤 보완한 negative matrix는 거부할 경로를 /proc·CPU cgroup·/dev의 실제 각 mount에서
검사한다. 현재 후보의 기록은 uv-c1b-pending-candidate-static-review.json을 따른다.
이미 삭제한 incremental cache/operator를 복원하지 않고 canonical fresh build로 검증한다.
이 승인은 해당 fixed 후보와 offline inventory 시험에 한정되며 Host 활성화나 다음
미관측 read/child/network 권한을 사전 승인하지 않는다. 새 최초 fault가 나오면 원본과
현재 경계를 보존하고 다음 변경의 필요·범위를 다시 검토한다.

### 승인한 fixed 후보의 결과와 다음 owner 경계

Fresh canonical C++ build285.319s/전체105 latch와 canonical5가 PASS다. Actual startup은
4.09s·trace29/3306B/session complete로 PASS다. Offline inventory는3.97s 뒤 새 최초
fault로 FAIL이며 앞 read 후보 전체의 product acceptance 완료가 아니다. 새 fault는
CONTROL/CLONE_CHILD의 pinned Python3.14 image가 OCI root의 libpython3.14.so.1.0을
읽는 단계다. Diagnostic executable locatorb982a2d37572906d와 read32581434d67478c1을
기존 known Python paths에 대조했다. Hash는 진단이며 query 승인 authority가 아니다.
Container/volume/observer endpoint0·daemon waits[0,0]·Host 보호 상태 전후 동일이다.

다음 gap은 UV root metadata가 아니라 **고정 interpreter 정보 조회 child의 명시적 역할**이다.
IsPythonProfile(UV)를 true로 만들거나 CONTROL/clone의 라이브러리 전체 읽기를 허용하지
않는다. 기존 Cargo의 exact creator/query·sealed executable/runtime-read 경계를 먼저
대조한다. 현재 query argv/embedded script는 아직 확보하지 않았으므로 예외를 추가하지 않는다.

형식 조사에서는 같은 인증 UV native를 read-only/network-none/UID1000의 pure Python
derived fixture에서 실행하되 interpreter를 고정 Go stub으로 교체한다. Stub은 argv를
최대64KiB 데이터로만 기록하고 exit73하며 Python/script를 실행하지 않는다. Carrier는
실행 전에만 수정하고 source image·Host 파일·사용자 target을 바꾸지 않는다. 이 fixture는
production interpreter/검역 성공이 아니다. Captured query를 검토한 뒤 exact query identity,
creator/generation·read-only runtime·network 거부·demotion/exit 경계를 설계하고 negative
matrix를 먼저 작성한다. 완료한 carrier/derived image/stub/data는 exact owner로 정리한다.

실제 argv 조사에서 native UV는 Python에5개 인자를 전달했다: fixed interpreter,
-I/-B/-c와146B launcher다. Launcher는 UV의 임의 이름 private temp directory를 sys.path에
추가한 뒤 python.get_interpreter_info.main을 호출한다. Stub exit73/native exit2·Python
source 실행0을 확인했고 owned container3/image1은 제거, Host 전후 동일이다. Temp path
패턴만 허용하는 것은 Python code content/역할의 증명이 아니므로 runtime grant를 추가하지 않는다.

권한 확장 전에 고정 target 모드를 별도로 확인한다. 저장된 exact help의 --python-version/
--python-platform/--target와 [공식 CLI](https://docs.astral.sh/uv/reference/cli/)를 대조한다.
Wheel-only/no-build·고정3.14.7/manylinux_2_36 target에서 원래 도구가 interpreter 없이 처리할
수 있는지는 아직 미확정이다. 같은 no-execution stub과 자체 생성한 inert pure-wheel을
사용하는 offline runc 조사로 compile/install/list/sync의 command shape만 확인한다.
Synthetic wheel 성공을 public registry·gVisor·검역/설치 qualification으로 전용하지 않는다.
원래 native 동작이 이 경로를 지원하면 불필요한 child 권한을 만들지 않고, 지원하지
않으면 관측한 query 입력/고정 runtime 파일의 authority를 별도 설계한다.

첫 target-mode 조사에서 compile/install/sync는 --no-build와 --only-binary의 native CLI
충돌로 실제 명령 전 거부됐다. 이것을 interpreter 불필요/필수의 증거로 쓰지 않는다.
List는 stub을 실제 조회했다. 충돌 옵션만 제거한 별도 입력으로 나머지 경계를 확인하며
최초 원본을 덮어쓰지 않는다. Production flag 검증에는 이 상호 배타 의미를 반영한다.

충돌을 바로잡은 실제5 cases는 모두 interpreter query를 요청했고 stub73/native2였다.
Fixed target만으로 query를 없앨 수 있다는 가정은 이 pin/입력에서 반증됐다. Python
runtime source 조사에서는 UV가 생성한6개 regular files/44,055B를 코드 실행 없이 확보했다.
get_interpreter_info.py는28,127B이며 launcher는 -I/-B와 private UV runtime만 사용한다.
Vendored musl 경로는 subprocess 분기를 포함하므로 그 파일의 존재만으로 모든 플랫폼
query가 info-only라고 일반화하지 않는다. 초기 지원은 기존 pinned glibc Python image에
한정하고 query의 clone/추가 exec·network는 별도 권한을 얻지 못해야 한다.

## C1c — exact interpreter query 역할의 구현 전 감사·수정 지도

기준3623d1e. Root UV read/startup은 IMPLEMENTED=YES / WIRED=YES(native fixture) /
QUALIFIED=YES(startup 한정) / ACCEPTANCE_CLOSED=NO(full query/resolver)다.
Python query 역할은 IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO다. MISSING은 kernel clone/첫 exec에 묶인 조회 역할, 고정
interpreter ABI/stdlib read, negative matrix와 actual complete trace다. 기존 Cargo
query·Python profile·공용 read 분기를 대체하지 않는다.

수정 위치는 observer.cc의 GroupState, ParseSentryClone, ParseSentryProcessAndClassify와
새 UV-query-only predicate, observer_latch_test.cc의 clone/exec parser·open classifier 경계다. Group에
candidate/active 두 one-shot 값만 추가한다. Candidate는 admitted UV direct root의
kernel clone에서만 생성하고 매 exec에 소비한다. Parent TGID/start generation과
expected UV identity/소비된 admission/active control을 다시 대조한다. Query exec는
고정 OCI Python3.14 image·execfn/argv0 alias 일치·정확히 -I/-B/-c 및 관측한 launcher
문법이 함께 일치해야 한다. Kernel binary와 execfn은 sealed read-only OCI root로
검증한다. Query는 ARTIFACT 역할로 제한하며 expected_groups에 등록하거나 네트워크
authority를 주지 않는다. Clone/re-exec에서 query 권한을 상속·재생성하지 않는다.

Launcher의 두 random temp 성분은 이름만으로 신뢰하지 않는다. 실제 공급에서 검증한
고정 UV가 private namespace의 유일한 query-code producer이고, writable workspace는
Host/user target과 연결되지 않으며, resolver는 wheel-only/no-build로 artifact 코드를
실행하지 않는다는 기존 경계가 선행한다. -I는 사용자 Python 환경/PYTHONPATH를
배제하고 초기 site는 고정 read-only 이미지뿐이다. 이 producer/creator 경계와 exact
launcher를 함께 요구한다. 임의 child·다른 script·단순 temp path·mutable comm·진단
hash는 authority가 아니다. Native pin/query format/ABI 변경은 fail-closed 재검증한다.
조회 JSON도 선택 wheel/ALLOW/완료의 authority가 아니다.

Query read는 같은 active 역할/creator generation을 매번 확인하고 read-only OCI ABI와
Python stdlib에 한정한다. 기존 IsExactPinnedPythonSystemLibrary를 재사용한다. 기존
고정 loader candidate 목록만 순수 matcher로 추출해 두 소비자가 공유하고 원래 Python
gate·목록·반환 의미는 그대로 유지한다. 고정
stdlib의 디렉터리·sysconfig·확장 모듈과 Python executable/loader는 interpreter 실행의
실제 의존성이다. site-packages 디렉터리 조회와 내부 package 실행 권한을 구분한다.
사용자 wheel/site, 임의 library·Host bind·다른 mount·쓰기·network metadata를 포괄
허용하지 않는다. 기존 IsPythonProfile과 공용 Python read catalog의 허용 의미는 유지한다.
추가 system read가 필요하면 실제 첫 fault와 코드 근거를 보존한 뒤 범위를 검토한다.

검증 순서는 full candidate canonical format/Quick/security/docs/workflow와 pinned
C++ 전체 latch, 동일 isolated namespace의 native startup/query다. Latch는 실제
clone/exec parser와 open classifier를 통해 정상 두 alias, script/argv·isolated Python 형식, missing/stale/다른
parent·role·profile·root eligibility·pending·image shadow/writable/unsealed, 재실행과
grandchild·쓰기·타 path·network 거부를 검사한다. 새 동작은 native fixture에서만
확인하며 production resolver/CLI는 계속 미연결이다. Shared observer의 최종 qualification은
기존 Linux/npm/PyPI/Go/Cargo/GitHub 소비자와 필요한 CPU/CUDA를 별도로 확인한다.

두 번째 검토: 조회를 허용하려고 Python profile 전체를 열거나 Cargo predicate를
위장 호출하지 않는다. ABI catalog 재사용은 현재 helper에 한정하고 old Python read
동작은 유지한다. Captured source는 데이터로만 감사했고 실제 glibc query 완료는 아직
관측하지 않았다. Musl의 loader subprocess 분기에 권한을 부여하지 않는다. 실패한
inventory를 성공으로 덮지 않으며 완료 조사 data10경로/11,947,967B는 Root 참조와
query errors0 확인 뒤 제거했다. 작은 재현 source와 raw 결과만 남겼다.

추가 source 교차검토에서 pinned gVisor task_clone.go는 t.Origin을 child에 상속하고,
seccheck.go의 IsExecSession은 이 origin을 표시함을 확인했다. 따라서 docker exec에서
생긴 clone도 is_exec_session=true다. 초기 latch의 false origin 입력/거부 조건은 실제
kernel 의미와 불일치하므로 actual probe 전에 교정한다. 상속된 true는 직접 root의
authority가 아니며 실제 CloneChild provenance·creator TGID/start/UV expected generation을
계속 필수로 요구한다. 초기 후보의 작은 latch 성공을 actual query 성공으로 쓰지 않는다.
원본 uv-query-origin-field-audit.json과 각 후보 build source hash를 구분한다.

C1c actual의 최소 query는 기존 offline pip list에 empty private /haa-site target을
지정한다. 같은 exact-target stub 조사에서 list도 interpreter를 요청함을 이미 관측했다.
빈 target의 []와 complete external trace를 요구하고, 전체 image의 pip metadata를
읽기 위해 UV root 권한을 더하지 않는다. 기존 PythonInventory 시험/실패는 그대로
남기며 새 시험을 전체 inventory·selection·검역 완료로 쓰지 않는다. Native output이
예상과 다르거나 trace가 incomplete면 실패를 보존하고 권한을 자동 확대하지 않는다.

Origin 교정의 fresh C++106은 PASS277.421s다. 첫 empty-target actual은 startup3.72s가
PASS이나 query는4.01s/STREAM_FAULT다. Python 선택을 생략해 native UV root가 interpreter
자동 발견 도중 OCI /usr/bin/dash를 read(flags557056)하려다 차단됐다. Query child에 아직
도달하지 않았고 새로운 역할의 native qualification은 아니다. 기존 explicit Python
inventory는 이 자동 발견 read 없이 Python child까지 도달했으므로 최소 target probe에도
원래 계획의 --python /usr/local/bin/python을 명시한다. Shell read 권한을 추가하지 않는다.
원본 uv-query-role-actual/result는 보존하고 Host 전후 동일·container/volume/endpoint0·
daemon waits[0,0]을 확인한 뒤 완료 자료만 제거한다. 수정한 command는 별도 입력/영수증이다.

Explicit Python 입력은 startup3.69s PASS 뒤 query3.98s에서 다음 최초 거부다.
이번에는 actual PYTHON/ARTIFACT/CLONE_CHILD·sealed executable을 확인했으며,
system /proc의 /proc/sys/vm/overcommit_memory read(flags32768)다. 새 query 역할은
연결됐지만 native query 완료는 아니다. 기존 root UV read(flags557056)와 섞지 않고,
IsPinnedUVPythonQueryRead의 같은 exact active/creator guard 아래 이 한 read만
분류한다. 다른 proc/PID·sysfs·device·쓰기·다른 flags/mount·root/다른 child/profile은
계속 거부한다. ABI와 system 두 실제 subject의 role/신원/쓰기 부정을 함께 검사한다.
원본 uv-query-explicit-actual/result와 next candidate build를 별도로 연결한다. Host 전후
동일·container/volume/endpoint0·daemon waits[0,0]은 확인했다.

VM-only 후보의 fresh C++106 PASS269.049s 뒤 startup3.55s PASS, query3.90s FAIL이다.
다음 첫 거부는 같은 pinned PYTHON/ARTIFACT query의 system /proc 자기 TGID/maps
read(flags557056)다. VM-only 보완의 회귀/전체 조회 완료로 해석하지 않는다. 동일 query
active/creator guard와 system /proc 아래 현재 TGID의 maps 한 경로만 추가한다. 다른 PID,
literal self alias·environ/status/cgroup·쓰기·다른 flags는 거부한다. OCI ABI·VM read·own maps
세 실제 subject의 신원/role/flags/mount 부정을 함께 검사한다. 원본 uv-query-vm-actual/result,
후속 maps candidate와 완료 data 정리를 분리한다. Host 전후 동일·자원0·daemon wait0이다.

### 다음 production producer 연결에서 유지할 기존 경계

추가 owner 대조에서 pypi_resolver.go의 metadata 교차검증은 별도 고정 Python fetch
script를 isolated resolver에서 실행한다. 새 UV query child 역할은 이 범용 data-reader의
authority가 아니다. Intake.Resolve도 isolated resolver 사용을 명시 요구한다. 따라서
metadata HTTP를 Host resolution fallback으로 옮기거나 UV query에 임의 Python/network
권한을 덧붙이지 않는다. Existing typed policy·Python data-reader session과 streaming
Docker data transport의 실제 공통 구간을 재사용하는 연결을 먼저 구체화한다. Native
pylock/선택 WHEEL·whole archive/독립 metadata binding, promotion controls와 sync 모델은
여전히 MISSING이며 C1c query의 작은 성공으로 대신 닫지 않는다.

Own-maps 후보의 fresh C++106 PASS277.350s 뒤 startup3.70s PASS, query3.89s FAIL이다.
다음 거부는 같은 Python query의 readonly OCI /usr/share/zoneinfo/Etc/UTC read557056이다.
진단 locator07a7ea4562a20073을 기존 pinned-image 경로와 대조했으며 권한 authority가
아니다. 같은 query/creator/readonly OCI guard 아래 정확한 UTC 파일만 추가하고 다른
시간대·localtime alias·쓰기·타 flags/mount/역할은 거부한다. ABI/VM/maps/UTC 네 subject의
negative matrix를 검사한다. 원본 uv-query-maps-actual/result와 다음 UTC 후보를 구분한다.
Host 전후 동일·container/volume/endpoint0·daemon waits[0,0]을 확인했고 완료 data는 제거한다.

UTC 후보의 fresh C++106 PASS279.679s 뒤 startup3.74s PASS, query3.96s FAIL이다.
이번 첫 거부는 Python child가 아니라 UV CONTROL/DIRECT_EXEC_ROOT의 readonly OCI
/usr/local/bin/python3.14 read557056이다. 자식 역할과 분리된 기존 IsPinnedUVRuntimeRead의
같은 exact UV/admission/current expected generation guard 아래 이 실행 파일의 데이터
read만 추가한다. Python 실행 authority·clone·library catalog·다른 binary·쓰기·타 flags와
mount/profile은 계속 거부한다. System metadata6개와 새 OCI image1개에 실제 각 mount/
class negative matrix를 적용한다. 원본 uv-query-utc-actual/result는 보존하고, 전체 query
완료나 Root 검역/production 지원으로 확대하지 않는다. Host 전후 동일/자원0/waits[0,0]이다.

Parent-image 후보의 fresh C++106 PASS279.815s와 actual startup3.91s·empty-target
query4.10s가 모두 PASS다. Query는 []/trace31·3551B/session complete, startup은
trace29·3306B/session complete이며 Host 전후 동일·잔여 자원0·daemon waits[0,0]이다.
이는 고정 native query의 로컬 qualification이며 full inventory/selection/검역 결과가 아니다.
초기 whole-image inventory 진단은 original commit3623d1e/실패 raw로 보존하고 active Go
fixture에서는 제거한다. 불필요한 system pip metadata 권한을 추가하지 않는다. 실제로
통과한 startup/empty-target 두 시험만 기존 unconditional CI 명령과 validator에 연결한다.
Fixture 정리 뒤 같은 두 native 시험을 다시 확인하고 완료 data를 제거한다. Production
resolver·guarded install/upgrade/sync·CLI는 아직 MISSING이고 P2로 넘어가지 않는다.

### C1c local 종료와 다음 acceptance gap

[최종 로컬 영수증](./p1c-uv-query-local-20261022.json)의 정리된 fixture 재실행은
startup3.73s/query3.96s·각 complete trace29/31로 PASS다. 성공한 전용 data6,833항목
688,167,061B도 Root 참조·query errors0 확인 뒤 제거했다. 실제 자료 없는 [] 결과는
조회 qualification이며 Artifact 선택/안전성 ALLOW가 아니다. 단계 감사는 query
IMPLEMENTED=YES/WIRED=YES(native fixture·CI source)/QUALIFIED=YES(local query only)/
ACCEPTANCE_CLOSED=NO(P1 전체)다. 후속 [CI372](./p1c-query-ci372-20261022.json)는 같은 tree의
remote query/startup과 기존 Linux/CPU를 통과했다. macOS production build2m timeout/
Required FAIL은 별도로 남는다. C2 native output/production resolver는 이 query 증거로 닫지 않는다.

다음은 같은 P1c의 native selection/고정 data-reader·완전 archive/독립 metadata binding,
기존 guarded promotion/upgrade/sync와 CLI의 MISSING만 구현한다. 기존 엔진을 대체하거나
query JSON을 증거/Policy authority로 쓰지 않는다. P2..P7은 NOT_STARTED다.
