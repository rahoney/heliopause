# P4-O4 — pipx 단일 runtime owner와 배포 공급

기준 ec07636/tree d2670b67692fe01e85e89c22b2c2357ca0a7f839/2694files다.
Read: Queue/plan20 §5·6/client09, D-012/013/015, maintenance §2·3·6·7,
O2/O3 plan·local receipt, 관련 artifact/sandbox/ci/evidence lessons.
M13-004 하나만 IN_PROGRESS이며 기존 공급·유지보수·배포 구현을 대체하지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Publisher/whole wheel/typed closure | YES | YES(O2 developer) | YES(local) | YES(O2) | selected owner와 입력 전체 binding |
| Deterministic ZIP/no-replace | YES(O3) | YES(candidate) | YES(local) | YES(candidate only) | release asset/NOTICE·selected owner binding |
| Python ZIP startup semantics | YES(fixed bootstrap) | YES(explicit runc probe) | YES(runc only) | YES(basic probe only) | gVisor/observer typed role·native operations |
| Runtime lock/generator/version-support | YES(other clients) | NO(pipx) | NO(pipx) | NO(pipx) | schema migration, selected version/input hashes/ZIP identity |
| Frozen maintenance plan/apply/verify | YES(UV/PNPM/Bun) | NO(pipx) | NO(pipx) | NO(pipx) | supply freezing/revalidation/patch·tool metadata consumers |
| Release producer/installer | YES(other clients) | NO(pipx) | NO(pipx) | NO(pipx) | immutable ZIP DATA0444 + passive notices, SBOM/runtime metadata |
| Protected installed reader/guest transport | YES(other clients) | NO(pipx) | NO(pipx) | NO(pipx) | fixed ZIP kind/mode/path and readonly volume |

## 공급 선택과 수정 지도

1. Runtime lock의 현재 schema4를 명시적으로 migration한다. `pipx_tool`이 선택 버전,
   architecture=amd64/runtime_image=python_image, 공급 입력27개의 exact relative path/hash,
   ZIP path/hash/size와 notices path/hash/size를 소유한다. 독립 pipx version lock은 만들지 않는다.
   Canonical `tools/pipx/supply`에는 wheel 이외20개의 frozen identity/metadata/제공 proof 원문을
   보존한다. Wheel7개는 CI/운영의 외부 취득 입력이며 repository에 binary archive를 넣지 않는다.
   외부 metadata/identity의 version 주장은 입력 DATA이며 선택 owner와 같을 때만 소비한다.
2. O2 private reader와 O3 ZIP generator를 그대로 사용한다. Production materializer는 전체
   공급 input path/hash 집합과 selected version/ZIP/NOTICE를 owner에 대조한 뒤 기존
   `publishNativeAsset`를 사용한다. Source/file 누락·추가·치환, wrong version/Python binding,
   no-replace·정확한 출력 role을 검사한다. Main proof를 unsigned userpath의 서명으로 확대하지 않는다.
   Notices는 각7개 distribution의 실제 dist-info license/NOTICE member 원문을 순서대로
   연결한다. `packaging/licenses/*.py` 코드를 license로 오인하지 않는다.
3. Generator/checker/version-support와 fixtures는 같은 owner를 소비한다. Proof/metadata 파일
   hash와 source owner의 일치, 누락 소비자·구 schema·unknown/duplicate field를 거부한다.
   현재 source compiler·Python pin은 자동 최신화하지 않는다.
4. Existing update-runtime plan/apply 경로에 pipx frozen supply를 연결한다. Plan이 실제27입력과
   검증한 derived ZIP/notices를 동결하고 apply 직전 같은 proof/bytes를 재검증한다. 현재 selected
   owner를 임의의 외부 path·live resolve로 바꾸지 않는다. 기존 source patch/journal/lock을 유지한다.
5. Runtime-image-manifest/release-manifest/SBOM/release workflow와 installer의 기존 asset
   처리에 정확한 pipx ZIP/NOTICE를 연결한다. ZIP은 DATA0444이며 ELF/executable role이 아니다.
   기존 release compatibility·protected root/current/atomic install과 실패 복구를 보존한다.
6. 다음 typed transport는 `native_installed_tool.go`와 `native_tool_transport.go`를 수정하여
   기존 held root/owner/single-link/hash, bounded tar/readback을 재사용한다. 현재 reader는 ELF
   header를 검사하는 구현이 아니라0555 mode를 고정한 reader다. 이를 대체하지 않고 fixed kind의
   mode0444를 명시하며 임의 mode/path를 공개하지 않는다. GVisor/observer의 exact loader/ZIP
   root·fixed UV backend·installed Python Artifact query attribution은 별도 실제 qualification이다.

## 두 번째 검토와 종료 기준

위 기존 owner를 재작성하거나 새 generic framework/production package를 만들지 않는다.
한 helper의 local 성공으로 production pipx 지원을 표시하지 않는다. 첫 owner 후보는 canonical
format/quick/security/docs/workflow/runtime consistency, 영향받은 전체 Race와 실제 frozen
materialization/maintenance round-trip/negative를 확인한다. Release fixtures는 DATA role과
immutable identity·old release compatibility를 함께 검사한다. Common runtime 변경 때는 기존
UV/PNPM/Bun·Python profile 소비자 검증을 빠뜨리지 않는다. 원격 CI는 공통 runtime 또는
end-to-end 경계에서 묶고 작은 공급 helper마다 push하지 않는다.

O3 ZIP 후보는2505814B/SHA2562f0162285ed79d1de358dd4261e189731f43106c2ca8084167bc733c62ba0eb4다.
이는 검증된 후보 identity이며 이 문서만으로 installed activation/runtime admission을 부여하지 않는다.
O4 시작 시점에는 production source/runtime pin/배포 workflow를 아직 변경하지 않았다.

## 추가 소비자 감사와 A/B 경계

`NativeUpdateFiles`는 현재 native 실행 파일6개를 반환하며 source/inspection/promotion/
releasebootstrap/releaseactivation/installer가 그 집합을 직접 소비한다. Bootstrap은11subjects,
activation wire는10payloads를 요구한다. 현재 release source는 active runtime-lock hash와 같을
때만 업데이트를 받지만, 이것만으로 ZIP의 Python code를 passive metadata로 바꾸지 않는다.
`ReleaseSetInspector`의 passive metadata 분기나 native ELF 검사에 ZIP을 끼워 넣지 않는다.

따라서 O4-A는 실제 사용 가능한 selected-owner materializer와 frozen maintenance/생성 identity의
범위로 검증한다. O4-B는 release artifact 검사·typed DATA role·bootstrap/wire/activation까지
연결할 구체 수정 지도를 먼저 보완한다. A에서 schema5를 인식하는 기존 release generator는
기존 asset 집합만 만들며 pipx를 배포했다고 표시하지 않는다. 전체 O4 release/runtime acceptance는
열려 있다. 새 native role을 추가하는 것과 executable Python ZIP을 검사하는 책임은 구분한다.

첫 O4-A Quick60.233s는 기존 workflow literal 검사에서 `python_image`를 hand-copied identity로
오인했다. 이는 새 pipx owner의 검증된 runtime_image enum이 다른 owner key를 참조하는 값이다.
Exact `/pipx_tool/runtime_image=python_image`와 기존 Bun의 해당 enum만 identity 수집에서
제외하고 다른 field·unknown value와 실제 version/digest 검사는 유지한다. 원본 first Quick을
보존하며 후속 v4 Quick과 affected 전체 Race에서 교정을 확인했다. 추가 realcorpus 시험은 첫 format 이후 작성했으므로
후속 canonical format과 실제 tagged compile/실행에 포함한다.

두 번째 Quick13.612s는 새 회귀의 slices import 누락으로 test-build에서 중단됐다.
제품 owner/reference 교정과 이 시험 compile 오류를 분리하고 import를 추가했다.
동일 version의 no-acquisition/idempotent apply와 supply flag scope 거부도 기본 회귀에 포함한다.

세 번째 Quick206.673s는 기존 GVisor capture 소비자의 schema4 guard에서 실패했다.
실제 compiler/output 회귀가 누락된 schema 소비자를 검출했으며 bundle 자체를 재빌드하거나
기존 output을 변경하지 않는다. 같은 schema 감사를 통해 PNPM materialization owner도 확인해
두 guard를 schema5로 함께 이행한다. 기존 PNPM/Bun/new pipx 현재 owner 수용과 구 schema
거부를 한 회귀로 연결한다. 첫 두 오류와 원인을 합치거나 기존 gVisor 기능을 재구현하지 않는다.

## O4-A 로컬 검증 결과

[O4-A receipt](./p4o4a-pipx-runtime-owner-local-20261027.json)는 selected owner/frozen
maintenance의 검증 범위와 이전 후보 실패를 구분한다. 첫 후보는 Quick177.832s와
scripts 전체/runtimeidentity Race119.715s·canonical4가 PASS였지만 전체 staged diff check에서
NOTICE EOF 빈 줄이 검출됐다. 원본 license를 보존하고 생성기가 이미 있는 개행을
중복 추가하지 않도록 교정했다. 최종 NOTICE28839B와 generated identity를 재생성한
후보는 Quick248.69s/수정 owner와 runtimeidentity Race35.863s/actual4.637s PASS다.
Actual은 실제7-wheel 공급과 synthetic previous owner의 apply 중단·재개·반복 및 frozen
변조 거부이며 실제 이전 release upgrade가 아니다. Selected source CLI는 ZIP/NOTICE의
0444·선택 hash 일치/원본27입력 불변을 확인했다. Native runtime 활성화는 하지 않았다.
첫3개 오류·이전 후보 canonical 결과·staged EOF 실패 원본을 보존한다. 최종 전체 index와
문서·보안·workflow·release-gate 검사는 commit 직전 별도로 확인한다. 기존 release13assets/
2tools는 유지되며 O4-B/C·전체 pipx acceptance는 열려 있다.

## O4-B 소비자별 수정 지도 — source 작성 전

A의 schema5를 받아들이는 것과 B의 runtime ZIP 배포를 분리한다. 현재 actual code에서
`ReleaseSetInspector`는 exact native member를 기존 ELF composite에 보내며 Python ZIP
실행 inspector는 없다. `PythonDynamicBackend.InspectWheelWithPlan`은 wheel identity,
validated wheel destinations와 typed observation plan을 전제로 한다. GitHub ZIP을 fake wheel로
바꿔 이 전제를 통과시키지 않는다. 새 공급 검사를 이뤄야 하는 위치는 아래와 같다.

| 책임/위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| `ReleaseManifest.NativeUpdateFiles`, manifest producer | YES(native6) | YES(existing) | YES(existing) | YES(existing) | native6 보존, 별도 fixed pipx DATA member/NOTICE와 complete set 선택 |
| `ReleaseSetInspector`, self_update composition | YES(native/passive) | NO(Python ZIP) | NO(Python ZIP) | NO(Python ZIP) | immutable ZIP static facts와 실제 Python execution inspection, artifact에 묶인 required Evidence |
| `PythonDynamicBackend`/external observer | YES(wheel transaction) | NO(release ZIP) | NO(release ZIP) | NO(release ZIP) | ZIP namespace·loader의 ARTIFACT 실행 역할과 외부 완료 연결; wheel identity 위장 금지 |
| `HeliopauseReleaseSet.ResolveDependencies` | YES(existing) | NO(pipx) | NO(pipx) | NO(pipx) | fixed DATA role graph member의 동일 snapshot/size/hash/proof 연결 |
| `NativeReleaseUpdatePreparation`, bootstrapFiles, releasePayloads | YES(existing) | NO(pipx) | NO(pipx) | NO(pipx) | 선택 세트/IPC/protected copy/복구 membership에 ZIP/NOTICE 추가 |
| `releaseinstall.selectFiles`/copy modes | YES(existing) | NO(pipx) | NO(pipx) | NO(pipx) | exact ZIP0444와 passive NOTICE0444, 기존 executable0555·historical user install 유지 |
| runtime metadata/SBOM/workflow/package build | YES(13 assets/2 tools) | NO(pipx) | NO(pipx) | NO(pipx) | 생성/서명/설치/검역 소비자와 같은 변경에서 actual 공급 연결 |

구현 순서는 ZIP static surface·dynamic 실행/관찰과 그 fail-closed inspector를 먼저 연결하고,
그 검사에 실제로 도달하는 complete release set을 확장한다. `nativeRuntimeToolIdentity`의
fixed DATA kind와 readonly transport는 이 실행 경로의 선행조건이다. 따라서 O4-C의 transport
부분은 B와 연결되는 순서로 먼저 수행하되 native install/upgrade/CLI 완료와 혼동하지 않는다.
기존 `native_installed_tool.go`의 held root/current/file·size/hash·single-link 검사는 유지하고
고정 kind의0444만 추가한다. caller가 arbitrary mode/path/image를 선택하는 API는 만들지 않는다.

두 번째 대조에서 현재 C++ observer의 `IsPythonProfile`은 wheel profile만 알고 있으며
`github-elf`는 `/work/artifact` native handoff를 전제로 함을 확인했다. 단순 ZIP hash binding,
`--help` exit0, Python 출력 receipt만으로 dynamic completed를 발행하지 않는다. 새로운 profile이
필요하면 기존 admission/trace/session/cleanup owner를 확장하고 실제 observer latches와
기존 UV/PNPM/Bun/Python 소비자 검증을 함께 수행한다. 일반 GitHub ZIP 지원 범위·Core/Policy
계약은 확대하지 않는다. 이 map은 source/runtime 연결 완료 증거가 아니다.

추가 transport 대조: `uv_runtime_volume.go`의 mountArgument/verifyTarget은 공통
`/haa-runtime`에 RW mount·단일 attachment를 요구한다. ZIP 파일의0444와 readonly mount는
서로 다른 사실이다. Pipx와 backend UV를 각각 같은 target에 붙이거나 readonly target에
기존 docker cp를 바로 호출하는 설계는 사용하지 않는다. 준비 시 writer와 실행 시 reader의
역할·volume 생존·정리, fixed 별도 mountpoint 또는 검증된 exact pair 공급을 구체화해야 한다.
기존 UV/PNPM/Bun의 검증된 mount 계약을 전체적으로 느슨하게 바꾸지 않는다. 이 준비/실행
경계의 actual test와 external observation 이전에는 공급 pin을 runtime 권한으로 쓰지 않는다.

## C2 뒤 B1 입력·실행 owner 대조 — 2026-10-27

기준504e698/tree d2b3853588cfd7a3c1ecaaeddb0b75355f5a8f04/2737files다.
C1/C2는 fixed DATA 공급과 실제 readonly volume만 named local QUALIFIED다.
공통 기존 소비자의 CI419는 진행 중이며 그 결과를 선취하지 않는다.

| 기존 위치/책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | B1 MISSING |
| --- | --- | --- | --- | --- | --- |
| `pypi.PipxSupplyClosure`/`ZIP` | YES(authenticated wheel→ZIP) | YES(materializer) | YES(O2/O3/A) | YES(supply only) | acquired release ZIP의 정적 surface 입력과 typed binding |
| `githubrelease.StaticInspector.openAsset` | YES(intake handle/path/size) | YES(native/passive) | YES(existing) | YES(existing) | fixed ZIP reader에서 재사용; bounded bytes/hash는 별도 확인 |
| `githubrelease.inspectZIP` | YES(archive path/size/type) | YES(generic static) | YES(existing) | YES(existing) | 실제 member bytes/CRC·Python surface·required plan |
| `ReleaseSetInspector` | YES(native/passive) | NO(Python ZIP) | NO | NO | 별도 required inspection 전략; metadata/native fallback 금지 |
| `pipxDataVolume` | YES | YES(private DATA) | YES(local runc DATA) | NO(runtime) | acquired artifact bytes와 관찰 대상 identity 연결 |
| `observationTransaction` | YES(typed units/외부 회계) | YES(wheel) | YES(existing) | YES(existing) | ZIP에 맞는 사전 unit 계획과 기존 완료 validator 연결 |
| `SharedObserver`/C++ process role | YES(wheel/native profiles) | NO(release ZIP) | NO | NO | 고정 loader/ZIP topology·ARTIFACT role·외부 완료와 회귀 |

공급 materializer는 원본 wheel별 RECORD/metadata/공급 proof를 검증한 `PipxSupplyClosure`만
받는다. ZIP 소비자가 release artifact를 wheel이라고 위장하거나 wheel URL을 새로 resolve해
이 생성자를 통과시키지 않는다. Release의 acquired bytes와 그 identity/digest를 유지하며
기존 ZIP bound/path/metadata 및 Python 실행 surface 규칙을 재사용할 reader가 필요하다.
선택된 ZIP pin은 입력 동일성을 좁히지만 동적 검사 완료를 대신하지 않는다.

`preparePipxTool`은 **이미 설치된** fixed DATA 공급 owner다. Self-update의 새 acquired ZIP을
검사할 때 current 설치 파일을 읽어 대신 검사하지 않는다. `StaticInspector.openAsset`의
intake 선택과 기존 metadata 소비자의 bounded read/hash 패턴을 사용해 동일 acquired ZIP을 C1의 bounded DATA transport로
전달하고, 외부 Evidence는 HAA release의 원래 GitHub identity/version/digest에 묶는다.
Pipx package version을 HAA release version으로 바꾸거나 그 반대로 바꾸지 않는다.

`InspectWheelWithPlan`은 `ValidateTypedObservationPlan`과 `validatedWheelDestinations`를
필수 전제로 사용한다. 직접 호출해 가짜 wheel plan을 만드는 방법은 제외한다. 기존
`observationTransaction`의 unit 중복/누락·CPU/trace/cleanup 검사를 재사용하되 ZIP의
실제 요구 실험을 사전에 고정하는 adapter가 필요하다. 정확한 required unit 집합과
loader handoff를 정하기 전 새 ExecutionCompleted/Evidence를 발행하지 않는다.

`SharedObserver.pythonClosureExpectedTopology`와 C++ `IsPythonProfile`은 현재 wheel의
`/haa-site` preparation/anchor/observation을 소유한다. 여기에 새 profile 문자열만 추가하면
wheel의 설치·파일 권한을 잘못 승계할 수 있다. 새 ZIP 검사에서는 `/haa-pipx`의 RO DATA와
고정 Python loader를 구분하고, release ZIP 코드가 실행되는 시점부터 ARTIFACT로 귀속한다.
후속 trusted pipx resolver의 CONTROL/설치된 target Python query의 ARTIFACT 경계와도
섞지 않는다. 기존 admission·trace·resource ledger·helper stop/wait를 재사용하며 stdout,
help exit0, pin 일치나 Python token을 완료 authority로 쓰지 않는다.

따라서 다음 source 전 audit는 acquired ZIP surface/typed required plan → 해당 계획의
external observer/backend → required composite → complete release DATA set 순서다.
기존 native6·passive metadata·wheel/UV/PNPM/Bun 소비자의 성공 조건은 유지한다.
이 대조는 B1 구현 완료나 신규 observer profile 승인·qualification 기록이 아니다.

추가 정밀 대조: `openAsset` 자체는 handle/path/lstat size→open이며 protected installed
reader와 같은 held root/current/inode 검증이 아니다. 이보다 강한 보장을 이미 구현됐다고
표시하지 않는다. `inspectZIP`도 central-directory path/type/count/expanded-size를 검사하며
member body를 열어 CRC/metadata를 검증하는 함수는 아니다. 새 acquired ZIP consumer는
읽은 immutable bytes의 실제 길이/digest를 별도로 묶고 Python surface에 필요한 member
내용을 bounded하게 검사해야 한다. 이 확인은 기존 구현의 대체 요구가 아니라 B1 gap 구체화다.
