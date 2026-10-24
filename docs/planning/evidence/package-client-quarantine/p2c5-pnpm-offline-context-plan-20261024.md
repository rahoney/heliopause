# P2c5 — frozen dependency/peer 검사 문맥

2026-10-24, baseline `40d4e25f6f276e1bdacc828eaea0194328dc3da7`.
[P2c4 실제 결과](./p2c4-pnpm-project-snapshot-local-20261024.json)의 transitive dynamic
INCOMPLETE를 해결하기 위한 같은 M13-004/P2의 다음 순차 작업이다.
현재는 **A: native offline fixture 의미 검증**만 착수한다. B의 production 연결은 A 결과를
대조한 정확한 수정 지도·두 번째 검토 뒤에 수행한다. 이미 구현된 npm backend를 교체하지 않는다.

## 1. Source 작성 전 baseline audit

| 책임 / 실제 symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 npm 단일 archive dynamic — Backend.Execute/npmLifecycleCommand/DynamicInspector | YES | YES | YES | YES | 기존 단일 범위 유지; transitive/PNPM의 closure 준비는 현재 계약 소비에 없음 |
| PNPM 선택/독립 source/complete snapshot — source40d4 | YES | YES | NO | NO | 단위 로컬 검사·single actual PASS; transitive 이후 전체 qualification 미완료 |
| immutable PNPM tool·Node·project volume·trace/정리 — withActualPNPMNativeRuntime 및 production resolver owner | YES | YES | NO | NO | qualified startup/선택 경계 재사용; 새로운 offline 설치 명령 의미만 검증 |
| PNPM native offline dependency/peer fixture | NO | NO | NO | NO | private file-source recipe·실제 native output·원본 controls·무실행/무네트워크 증거 |
| static-all → exact instance context → target lifecycle coverage | NO | NO | NO | NO | A 결과 뒤 B의 typed 입력/실제 static-all·closure 연결/negative·original 소비자 검증 |

기존 backend/dynamic은343과 동일하다. P2c4의 is-odd3은 integrity/static/action fact가
정상이나 dynamic INCOMPLETE다. One-archive 소개와 npm InspectGraph/Project 부재는 source로
확인했으며 stderr의 구체 오류/ENOTCACHED를 관측했다고 쓰지 않는다. PyPI의 기존 성공·
closure loader를 npm의 이미 구현된 기능으로 전용하지 않는다.

## 2. 공식 source에서 먼저 확인한 의미와 남은 질문

Pin은 기존 PNPM12.15.0/source91774e72a04d52ce61db06339696a433d383c764 그대로다.
`pnpm/crates/tarball/src/download/fetch.rs`는 file: archive를 HTTP 없이 읽는 경로를 가진다.
따라서 검증한 tarball만 fixed guest path에 소개하고 private recipe의 tarball locator만
file:로 바꾸는 방법을 먼저 검증한다. Public lock parser의 file/local 거부는 유지한다.
이것은 사용자 local dependency 지원이나 public selected graph 변경 승인이 아니다.

`pnpm/crates/cli/src/cli_args/rebuild.rs`는 explicit package 선택을 frozen/NoInstall
경로로 보내며 root pending projects와 구분한다. 다만 실제 lifecycle 선택/peer instance
전체 coverage·script-shell/observer role·build approval 의미는 아직 qualification하지 않았다.
Rebuild 성공 exit0만으로 target의 선언된 lifecycle을 관찰했다고 인정하지 않는다.
Store 내부 format을 수동으로 생성하거나 ambient cache·native npm 대체에 의존하지 않는다.

## 3. A의 정확한 수정 위치와 실행 범위

1. `internal/sandbox/pnpm_offline_fixture_real_test.go`에 gated 실제 test만 추가한다.
   기존 `withActualPNPMNativeRuntime`의 고정 tool/Node image/project tmpfs·trace/cleanup을
   재사용한다. Production Backend/Inspector/Policy/Core schema는 A에서 변경하지 않는다.
2. C4 transitive의 exact is-odd3.0.1/is-number6.0.0을 기존 public resolver로 획득하고
   SRI/static 검사를 먼저 통과시킨다. 파일은 동일 acquired SHA256로 guest 경로를 정한다.
   Known fixture의 두 identity/edge/constraints·SRI를 private YAML-compatible JSON recipe에
   그대로 두고 locator만 `file:`로 연결한다. 이 test용 recipe는 범용 PNPM codec이 아니다.
3. 기존 fixed project control 전송을 쓰고 추가 archive는 test-local bounded Docker tar
   stream으로 named tmpfs 아래 exact sha 파일에만 소개한다. Header uid/gid1000·고정 mode,
   단일 파일/hash·size를 확인하며 Host 추출이나 user argv는 없다.
4. 같은 native로 offline/frozen-lockfile/ignore-scripts/ignore-pnpmfile/ignore-workspace와
   private store/state를 명시한다. Linker 의미도 고정 source와 대조한다. Network none과
   원래 runtime provisioning/notifier 차단을 유지한다. Native 준비에서 lifecycle 실행은0이다.
5. 두 controls byte 동일과 실제 설치된 두 package identity·연결을 bounded data read로 확인한다.
   Native 명령 stderr는 qualification raw에만 제한적으로 보존하고 product Evidence로 노출하지 않는다.
   첫 오류면 후속 성공을 주장하지 않고 exact role/path/flags/trace/cleanup을 기록한다.

A는 private recipe/동일 native offline 배치의 원시 의미 검증이다. Archive 동적 관찰 완료나
Host promotion 권한·완성 CLI가 아니다. Mode/command/profile 한도와 보안 거부를 먼저
풀지 않는다. Actual first fault로 필요한 입력 owner가 확인될 때만 별도 좁은 검토를 한다.

## 4. A 검증과 자원 정리

현재 변경의 canonical format/quick/security/docs와 tagged test build를 먼저 실행한다.
Fresh 전용 namespace 자료/actual.log/result.json을 요구하고 이전 overlay·stale 로그를
다시 사용하지 않는다. 기존 보호12파일/images/networks/bridge/service/firewall과 helper
wait/CONFIRMED_STOPPED·container/volume/endpoint 정리를 확인한다. 사용이 끝난 payload와
전용 바이너리는 all-UID refs/query errors0 뒤 삭제한다. 작은 raw만 보존한다.

동일 입력에서 native offline 준비가 검증되면, 그때 B의 production 타입·고유 archive vs
peer instance mapping·static-all·per-target required coverage·동적 완료/실패·정리·기존 npm
단일 소비자 유지와 records/promotion 재사용 위치를 확정한다. Peer suffix를 단순 이름으로
합치거나 fake primary를 추가하지 않는다. Unknown build selection/coverage는 미완료다.

## 5. A의 두 번째 검토 — test source 작성 전

현재 source와 원본 오류를 다시 대조했다. Public solver 선택을 바꾸거나 static/Policy를
완화하는 방법은 필요하지 않다. Native offline 명령의 local-tarball 의미를 확인하기 전에
대규모 production 인터페이스를 만들지 않는다. A는 현재 필요한 실제 prerequisite이고
미래 milestone scaffold가 아니다. 기존 helper의 관찰/cleanup과 고정 runtime을 재사용하고
generic directory read·Host hardlink·network fallback을 추가하지 않는다. 아직 확인하지 않은
rebuild/peer coverage는 B의 미결정으로 분리하여 이미 구현됐다고 표시하지 않는다.

CI402/head940은55분 실제 타이머 뒤 Required 포함10SUCCESS와 checkout tree 일치를
확인했다. [CI402 영수증](./p2c2-pnpm-ci402-qualification-20261024.json)이 범위를 소유하며
이후 local commit 둘이나 본 A actual의 성공으로 전용하지 않는다.

## 6. A 첫 검증 — file locator 교체는 권장 제외

첫 canonical Quick30.847s/security4.681s/docs.225s는 PASS였지만 tagged build는 fixture의
uint64 size와 기존 launch 식별자 오류로 멈췄다. Guest는 NOT_EXECUTED이며 빈 전용7dirs를
all-UID refs/query errors0 뒤 제거했다. 해당 두 test 오류만 바로잡고 tagged build를 통과했다.

Actual first7.224s는 native exit1이며 complete trace/금지 관찰0, helperwait0/Host전체 동일이다.
Native stdout를 성공으로 바꾸지 않고 기존 Go ExitError의 stderr를4KiB까지만 raw에 기록하는
진단을 추가했다. 기존 exact overlay 정리57,824,359B 후 fresh 진단6.623s에서
`ERR_PNPM_RESOLUTION_SHAPE_MISMATCH`를 실제 확인했다. Registry-style dependency key에
file: resolution을 붙이는 것을 PNPM 자체가 거부한다. 앞 §2·§3의 locator 교체 가설은
**반증/권장 제외**이며 native verifier를 끄거나 key/peer wiring을 임의 변경하지 않는다.
원본 `p2c5-pnpm-offline-{context,command-diagnostic}`와 rejected-source receipt가 scope를 소유한다.

### 수정한 A 방법과 두 번째 검토

같은 gated fixture만 수정하여 public lock의 registry keys·resource SRI·모든 edges/constraints를
그대로 둔다. 이미 검증한 두 archive를 fixed sha guest path에 소개한 뒤 공식
`pnpm store add <file:archives>`를 offline/private store에서 호출하고, 원래 public lock으로
offline/frozen/ignore-scripts install을 시도한다. 공식 store/add source는 설치 없이 archive를
ingest하며 local fetch source도 확인했다. 다만 다른 source kind의 cache가 registry lock의
integrity lookup에 실제 재사용되는지는 아직 미검증이므로 A에서 실제 확인한다.

Store/index schema를 직접 쓰거나 metadata/lock key를 조작하지 않는다. 동일 native의 cache
입출력과 두 원래 controls byte 보존·실제 두 identity/edge·완전한 관찰을 모두 확인해야 한다.
새 frozen store/추가 permissive option을 추측으로 넣지 않는다. 실패하면 정확한 native
오류와 source owner를 확인하며 B의 production API는 계속 미작성이다. 이 검토는 첫 가설의
실패를 덮지 않고 원래 지원·보안 의미를 보존한다. 현재 P2 acceptance는 NO다.

### Store-add 첫 관측과 정확한 원인 진단

Registry lock을 보존한 실제 store primitive6.622s는 store-add에서
ERR_PNPM_STORE_ADD_FAILURE로 끝났고 install은 미실행이다. 완전한 관찰/금지 관찰0과
Host 전체 동일/helperwait0/원래 exit1을 확인했다. Pinned 공식 store/add.rs의
report_add_outcome은 개별 실패 code를 warning event로만 내고 마지막에는 aggregate error만
반환한다. Silent reporter에서는 이 유일한 개별 원인이 빠진다. 따라서 test-only store
명령의 reporter만 append-only로 바꿔 bounded raw를 확인한다. Native lock/정책/자료·실행
권한은 바꾸지 않으며 구체 원인은 아직 미확정이다. Product Evidence에는 원문을 넣지 않는다.

Append-only 진단6.575s에서 실제 두 local tarball이 `../artifacts/<sha>.tgz`로 조회되어
ENOENT인 개별 code를 확인했다. Native resolver는 --dir 기준 상대 locator를 만들지만
현재 Docker exec는 작업 디렉터리를 명시하지 않는다. 기존 boundaryExecArguments의
Root helper/credential 전환은 유지하고 Docker exec의 고정 --workdir만 실제 project로
맞춘다. Copy 뒤 archive를 bounded tar read로 되읽어 acquired SHA와도 먼저 대조한다.
이 수정으로 해결됐다는 판정은 다음 같은 actual이 통과할 때만 한다. Native lock/schema/
권한·네트워크 거부는 불변이며 Product Backend/Inspector는 계속 미변경이다.

Workdir actual6.164s는 먼저 두 archive 되읽기 SHA 동일을 확인했으나 Root bootstrap의
OCI chdir가 uid1000/private0700 project에서 permission denied/exit126이었다. Native store는
미실행이며 이것을 PNPM 오류나 관찰기 fault로 기록하지 않는다. Source의 directExecOrigin도
고정 --user/helper 문법만 수용하므로 임의 Docker exec flag를 추가하는 해법은 제외한다.
Directory0700·Root capabilities·admission parser를 느슨하게 바꾸지 않는다.

Store-add는 project 설치가 아니므로 기존 고정 process cwd(`/`, container Config.WorkingDir로
확인)와 store command의 --dir만 `/`로 맞춘다. Actual cwd와 Root helper exec 문법은 그대로다.
Archive/store/state는 계속 private named tmpfs의 fixed absolute paths이며 원래 project lock을
사용하는 frozen install은 기존 --dir project를 유지한다. 이 좁은 test 교정의 실제 효과는
다음 native 결과로 판단한다. Root bootstrap을 바꾸는 production API는 필요하지 않다.

## 7. Metadata 준비 단계 — source 수정 전 추가 검토

Root-context actual6.886s에서 두 input readback과 store-add는 PASS였고 frozen install이
ERR_PNPM_NO_OFFLINE_META(is-number)로 멈췄다. 기존 cache 위치의 outside-workspace2개도
보존한다. Closed-network 전체 fixture의 성공이 아니며 store-add를 완성된 offline registry
준비로 일반화하지 않는다. Pinned config settings는 cacheDir을 lockfile verifier memoization과
full metadata mirror로 설명한다. View command는 uncached full HTTP API이므로 metadata
mirror를 채우는 API로 가정하지 않는다. Cache 파일 format을 직접 생성하지 않는다.

공식 cli_args/fetch.rs의 fetch는 frozen lock을 사용하며 virtual-store-only/NoInstall이다.
Root manifest check를 생략하는 이 명령 앞에서 HAA가 exact manifest/lock graph와 acquired
SRI/static을 이미 검증해야 한다. Upstream source-kind/SRI/lock verification은 그대로다.

다음 A2는 test-only helper/fixture만 아래처럼 좁게 바꾼다.

1. pnpm_runtime_real_test.go의 기존 body를 공유하는 online-policy helper variant를 추가한다.
   기존 startup/offline callers는 network none이다. Online variant는 기존 systemEndpointResolver,
   npmNetworkArguments/NewResolverNetworkPolicy와 authenticated integration service의 public
   registry/TCP443·고정 hosts만 쓴다. 새 service/profile은 없다.
2. 이미 획득·SRI/static 검사한 두 archive의 store-add 뒤 native fetch/ignore-scripts로
   full registry 검증·cache 준비를 수행한다. CacheDir도 같은 private PNPM input volume의
   fixed cache로 둔다. Pinned native가 만드는 opaque format을 소비하며 직접 쓰지 않는다.
3. 같은 owned network를 Docker에서 disconnect하고 trusted inspect로 container endpoint0을
   확인한 다음 offline/frozen/ignore-scripts install을 수행한다. User code는 A 전체에서0이다.
   Disconnect/inspect 불확실성은 중단하며 unrestricted bridge/DNS fallback이 없다.
4. Native identity/edge·원본 controls 동일, complete trace·outside/security 관찰0과
   helperwait/cleanup·Host 전체 동일을 확인한다. Container 제거 뒤 policy Close가 오도록
   기존 defer owner 순서를 유지한다. 아직 production API나 지원 완료가 아니다.

두 번째 검토에서 private0700/Root helper/admission 문법 변경이 불필요함을 확인했다.
Public HTTPS는 metadata 제어 단계에만 쓰고 offline 실행 단계와 분리한다. Artifact code의
네트워크를 허용하지 않는다. B의 per-target session에는 서로 오염되지 않는 cache/layout와
static-all/required target coverage가 별도 필요하며 A 성공만으로 이를 닫지 않는다.

A2 first8.024s는 PNPM CLI가 --cache-dir을 받지 않아 exit2였다. Config의 cacheDir field가
존재하는 것과 실제 CLI flag 지원은 다르며 metadata fetch는 미실행이다. Native lock/
observer/권한은 변경하지 않았다. 공식 defaults.rs의 default_cache_dir은 XDG_CACHE_HOME을
명시 지원하므로 online test-helper variant에만 고정 guest XDG_CACHE_HOME을 넣고 해당
미지원 flag 세 곳을 제거한다. Host 환경은 읽거나 덮어쓰지 않는다. 기존 offline/startup
helper caller의 환경은 유지하고 같은 metadata primitive를 다시 확인한다.

Env actual8.924s는 metadata 준비·endpoint0·offline frozen install과 두 installed identity/
정확한 dependency symlink·원래 controls bytes 동일까지 PASS였으나 outside-workspace2로
전체 FAIL이다. Complete trace·Host전체 동일/helperwait0/원래 exit1은 보존한다.
Cache 위치를 바꾼 것만으로 두 관찰이 없어지지 않았으므로 cache 접근이 원인이라고 확정하지
않는다. Source IsWorkspacePath는 PNPM의 `/tmp/` 하위는 인정하지만 `/tmp` 자체는 제외하며
write-capable pre-open은 normalized finding을 낸다. 정확한 두 pathname은 현재 수집하지 않았다.
우선 online variant의 TMPDIR도 같은 private input volume의 고정 tmp로 두고 uid1000/0700
directory를 input tar로 준비하여 비교한다. 기존 caller·observer·Policy와 권한은 유지하며
다음 actual로만 효과를 판단한다. 남는 경우 bounded locator 진단으로 먼저 원인을 좁힌다.

TMPDIR actual8.886s도 기능 검사 전부 PASS/완전한 관찰이나 outside2로 전체 FAIL이다.
따라서 XDG/TMPDIR를 지정하는 것만으로 이 경고가 해결된다는 가설은 권장 제외다.
Private diagnostic26은 제품 observer/hosttool source를 변경하지 않고 별도 C++ 사본의
최대8개 hashed stderr와 시험 binary의 stderr 연결만 사용했다. Private Go source는 compile
뒤 원본으로 복원했다. C++110 latch PASS/actual8.985s는 동일 경고2를 PRE_OPEN,
CONTROL/DIRECT_EXEC_ROOT/flags524865로 좁혔다. 이 pathname은 pre-resolution 진단이며
mount는 미확정이다. Hash를 새 admission 권한이나 완성된 접근 위치 증명으로 쓰지 않는다.
공식 config의 store placement probe와 early explicit 설정을 추가 대조한다. 진단용 제품
변경은 커밋·배포하지 않으며 실제 수정 효과는 원래 qualified25 observer로 확인한다.

Diagnostic27/C++110 PASS 뒤 fresh actual8.206s는 두 PRE_OPEN 모두 `/_tmp_` 접두사·길이17·
flags524865임을 확인했다. 공식 store_path.rs의 host_can_link_between_dirs/path_temp_in과
같은 root store-placement probe 형식이다. 원문 경로는 보존하지 않았고 probe의 생성 성공을
의미하지 않는다. Config loading은 env/CLI setting layer 뒤 apply_store_derivations를
수행하며 explicit.store_dir가 있어야 기본 탐색을 생략한다. 기존 --store-dir만으로 이
초기 경계의 탐색이 사라지지 않았다는 실제 결과와 함께 해석한다.

### A2의 좁은 교정과 source 수정 전 두 번째 검토

Online fixture helper의 고정 환경에 `PNPM_CONFIG_STORE_DIR=<private-input>/store`만
추가한다. 기존 --store-dir도 같은 값으로 유지한다. 공식 apply_env_settings는 이 설정을
ExplicitPaths에 먼저 기록하므로 기본 저장소 탐색을 피한다. 사용자 Host 환경/설정을 읽지
않으며 새 권한·path 예외·관찰기 완화가 없다. 먼저 제품 source 불변/원래 observer25로
fresh 실제 전체 fixture를 통과시켜야 한다. Diagnostic27 통과110은 제품 qualification이나
경고 해결 증거가 아니며 진단 사본은 마지막 소비 뒤 삭제한다.

### A2 실제 결과 — transitive primitive 로컬 검증

원래 observer25/binary87dac65a로 수행한 early-store actual8.591s(실제 test6.29s)는 PASS다.
필수 입력 SRI/static·두 introduced archive readback SHA·공식 native store/fetch·endpoint0 뒤
offline frozen install·두 identity/정확한 dependency link·원래 controls byte 동일을 확인했다.
관찰은 session_complete=true/events39, expected exec38/workspace84이며 금지 관찰0이다.
Host12/images/networks/bridge/service/firewall 동일, Host native installation 부재와 helper
wait0/CONFIRMED_STOPPED/original0/client_error0/client_restore0도 확인했다. 진단 관찰기의
성공으로 대신한 결과가 아니다. 공식 source snapshot은 같은91774e72의 Git blob을 확인했다.

범위는 transitive2 primitive다. Peer instance·per-target lifecycle coverage·static-all product
연결·Host layout/promotion/CLI와 전체 P2 qualification은 아직 미완료다. 다음 B 착수 감사에서
이 상태를 분리하고 기존 단일 npm backend는 유지한다. Completed namespace/진단 입력은
마지막 소비 뒤 all-UID 참조/query errors0을 확인해 제거한다.
