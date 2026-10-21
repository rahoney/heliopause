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
