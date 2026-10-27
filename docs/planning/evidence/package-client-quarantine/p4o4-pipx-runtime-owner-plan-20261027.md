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

## B1a — acquired ZIP immutable input의 세부 baseline

기준46b91a6, CI419는 source504e698의 기존 소비자까지 검증했다. B1a source 전 판정:

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| `PipxSupplyClosure.ZIP` 원본 member 공급 | YES | YES(supply) | YES(O3/A) | YES(supply only) | acquired ZIP snapshot 소비 |
| `wheelEntryPath`/`readZipEntry` | YES | YES(wheel) | YES(existing) | YES(wheel only) | ZIP 전체 bound·fixed pin·immutable file facts |
| acquired runtime ZIP typed input | NO | NO | NO | NO | caller buffer와 분리된 bytes/file digest snapshot, invalid zero value |
| `StaticInspector.openAsset`/bounded intake read | YES | YES(existing) | YES(existing) | YES(existing) | original HAA identity/size/digest에 묶인 ZIP typed input |

새 `pypi.PipxRuntimeZIP`은 selected runtime identity의 exact size/hash를 검사한 후
owned byte copy와 bounded member facts를 만든다. 기존 wheel path/entry reader를 재사용하고
정렬·중복·파일/디렉터리 충돌·Store/0444·고정 timestamp·전체 expanded budget·CRC를 확인한다.
Public constructor의 pin을 시험용 caller override로 열지 않는다. Synthetic 구조 회귀는 private
parser를 검사하고 actual selected ZIP은 explicit realcorpus 입력으로 public constructor를 검사한다.
외부 accessor는 copy만 반환한다. 임의 ZIP 지원·wheel identity·Policy Evidence는 만들지 않는다.

Inspection의 다음 소비자는 기존 controlled intake reader/DownloadReader와 original acquired
identity를 재사용하며 HAA release version을 pipx version으로 바꾸지 않는다. Static input 생성은
required inspection 완료가 아니다. Embedded 실제 RECORD/metadata와 Python surface·typed plan,
외부 observer/backend 연결은 B1b/B2에 남는다. 두 번째 검토에서도 installed-root reader를
새 acquisition의 대체 경로로 쓰지 않고 기존 release/native/wheel 구현은 유지한다.

B1a snapshot의 named local 결과는 [receipt](./p4o4b1a-pipx-zip-input-local-20261027.json)를 따른다.
`ports.DownloadReader`는 진행률 wrapper이며 자체 read bound가 없음을 source에서 확인했다.
다음 intake consumer는 `io.LimitReader`로 selected size+1을 명시하고 실제 길이/hash를
검사한다. 기존 helper 이름이나 total 인자를 byte budget으로 간주하지 않는다.

## B1b — ZIP 내부 metadata와 required observation 계획

기준294b70b에서 snapshot IMPLEMENTED=YES/WIRED=YES(parser)/QUALIFIED=YES(local)/
ACCEPTANCE_CLOSED=YES(snapshot only)다. ZIP plan은 네 상태 모두 NO이며 MISSING은
원본 embedded7 METADATA/WHEEL/RECORD의 완전 분할과 Python import surface, immutable
required unit 계획이다. 기존 `validateRecord`, `parseRFC822Metadata`, `parseEntryPointsTxt`,
`classifyWheelSurface`를 재사용한다. ZIP을 wheel로 위장해 `BuildObservationPlan`을 호출하지 않는다.

각 distribution의 namespace/dist-info/data를 원본 RECORD와 대조하고 미소유·중복 파일은
거부한다. RECORD self exemption은 기존 계약대로 검증하고 snapshot의 실제 size/hash를
분류 입력에 연결한다. 기존 classifier가 확인한 module/package import와 고정 loader target
`pipx.__main__`을 독립 required direct-import unit으로 동결한다. 이 target은 설치 후 사용자
command 기능 attestation이 아니라 검사 대상 runtime loader의 module이다. Optional dependency를
오류 출력으로 추가하거나 실행 후 coverage를 낮추지 않는다. ZIP의 data resource는 원본 위치를
유지하며 classifier의 installed destination을 실제 extraction 지시로 사용하지 않는다.

두 번째 검토: 고정 supply의 현재 순수 Python namespace만 받으며 일반 ZIP/namespace/native/
startup lane은 열지 않는다. 기존 observation unit type을 재사용하되 wheel의 Project/Version
또는 installation 계획을 재사용하지 않는다. Artifact identity는 다음 intake/composite에서 원본
HAA release identity로 유지한다. Plan 생성은 실행·외부 Evidence·Policy 완료가 아니다.

B1b metadata 원문 추가 대조에서 argcomplete console2·userpath console1과 platformdirs
pytest11 plugin1을 확인했다. Entry point가 없다는 가정을 사용하지 않는다. 기존
`ObservationPlan.importCoverage`의 사전 분류를 재사용해 console/gui-only target은
POST_INSTALL_COMMAND, 다른 plugin group은 required로 보존한다. 고정 loader module은
독립 required다. 실제 platformdirs.pytest_plugin의 `import pytest`와 pytest가 없는 현재
7-wheel 공급을 확인했으나 runtime 실패를 아직 실행 증거로 주장하지 않는다. 오류 기반
자동 추가나 required 완화는 하지 않으며 실제 검증/명시 prerequisites 책임은 후속으로 남긴다.

B1b 첫 Quick230.844s는 새 synthetic fixture의 WHEEL에 Root-Is-Purelib가 없어서
정상 case만 거부했다. 기존 toolTestWheel은 이 field를 생략하므로 새 fixture의 exact
WHEEL override를 RECORD 생성 전에 전달한다. 실제 selected7 WHEEL에는 모두 true가
있으며 product pure-Python guard를 완화하지 않는다. 첫 source/check 원본은 별도 보존한다.

## B1c — acquired HAA identity와 입력 reader 연결

B1b v2의 focused2.786s/Quick36.493s/PyPI 전체 Race9.334s/actual2.703s/docs/security는
PASS이며 static plan은7 distribution/9required/3post-install units다. Same ZIP의 fixed Python
runc 진단은 loader1.208s/기본 platformdirs0.495s exit0, pytest plugin0.600s는 missing pytest
exit1이다. Raw `p4o4b1b-import-diagnostic`에 argv/image/state/정리·보호5파일/Docker/원본 불변을
보존했다. GVisor/observer 결과가 아니며 원본 plugin 실패를 required 성공으로 바꾸지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| selected ZIP snapshot/metadata plan | YES | YES(explicit bytes) | YES(local) | YES(static only) | acquired artifact tuple 연결 |
| controlled intake reader | YES(other formats) | NO(pipx) | NO(pipx) | NO(pipx) | exact HAA source/name/asset/size/digest + bounded read + immutable tuple |
| required runtime completion | NO | NO | NO | NO | plugin import 실패 처리 방향, external observer/backend |

`StaticInspector.openAsset`를 재사용하는 private `readPipxRuntimeInput`을 추가한다.
입력은 원본 domain.AcquiredArtifact와 선택 ZIP의 immutable plan을 함께 보존한다.
고정 asset `pipx-runtime-linux-amd64.zip`만 받으며 original HAA version/handle을
pipx version으로 치환하지 않는다. Frozen manifest/version 일치는 기존 ReleaseSetInspector의
책임으로 유지한다. `io.LimitReader(selected size+1)`로 실제 read를 제한하고 진행률 wrapper는
bound 권한으로 쓰지 않는다. 길이/hash·context·close 실패는 typed input을 반환하지 않는다.

Dependency review: inspection/githubrelease → artifact/pypi는 original ZIP format/plan parser를
소비하는 구체 adapter 간 edge이며 runtimeidentity는 selected immutable pin만 읽는다.
현재 architecture allow map의 이 owner 두 edge만 추가한다. Core/Application·다른 consumer의
금지 edge는 유지하고 positive/negative 회귀로 scope를 검사한다. 새 service/package·Policy
예외나 passive inspection 성공 분기는 만들지 않는다. 두 번째 검토에서도 input reader가
required check/Evidence를 발행하지 않고 후속 composite만 판정을 연결하도록 유지한다.

B1c 첫 Quick34.181s는 새 test의 literal nil Context 호출을 Staticcheck SA1012가
거부했다. 제품 guard나 lint 설정은 유지하고 잘못된 test 호출만 제거한다. 명시 취소
Context의 cause 보존 회귀는 그대로 둔다. 최초 source/check 원본을 별도 보존한다.

사용자에게 진행 방향 결정을 요청했다. 현재 고정7-wheel 공급과 필수 plugin 조건은
유지하며 (1) pytest와 의존성을 명시 runtime 공급으로 추가해 전체 새 pin을 재검증하거나,
(2) 새 ZIP 범위에 한해 미사용 plugin 관찰 disposition을 별도 승인하거나,
(3) 현재 공급/규칙을 유지하고 pipx 경로를 차단 상태로 남기는 선택이다.
응답 전에는 어느 선택도 승인된 것으로 취급하지 않으며 독립 intake 검증만 마무리한다.

사용자가 “권장안대로 진행해”로 (1)을 승인했다. 기존 required coverage를 유지하며
pytest와 필요한 의존성을 명시적인 고정 runtime 공급에 추가하고 새 전체 bytes를 검증한다.
검사 중 recursive augmentation이나 plugin coverage 완화는 승인 범위가 아니다.
B1b/c 최종 [로컬 receipt](./p4o4b1bc-pipx-plan-intake-local-20261027.json)는
Quick44.492s/관련3 package 전체 Race39.114s/actual0.895s/docs/security PASS와
원본 HAA identity/handle 보존·읽는 도중 growth/cancel 거부를 연결한다.
다음 B1d는 새 공급의 publisher/whole RECORD/명시 root graph·namespace·NOTICE·bound와
기존 selected owner/generator/maintenance/transport 소비자 baseline을 먼저 대조한다.

## B2 — 새 공급 뒤 실행 owner의 수정 전 대조

기준1e5d21e/tree391df742/2772files다. B1d의11 distributions/20units(15required/5post-install)와
C1/C2 actual DATA는 로컬 검증했고 CI420의 공통 owner/checker hosted 회귀는10SUCCESS다.
Source1e5d21e와 checkout729b의 전체 tree391df742/2772 동일·원본 로그/정리를 확인했다.
이 대조는 source 변경 전 준비이며 새 observer profile·backend 완료가 아니다.

| 실제 책임/위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | B2 MISSING |
| --- | --- | --- | --- | --- | --- |
| `readPipxRuntimeInput` + immutable `PipxRuntimePlan` | YES | YES(explicit intake) | YES(local) | YES(input only) | 동일 acquired bytes를 backend로 전달 |
| `pipxPreparedTool`/`pipxDataVolume` | YES | YES(installed/explicit fixture) | YES(local DATA) | YES(DATA only) | installed current 대신 acquired ZIP의 typed 공급 진입점 |
| `freezePythonObservationUnits` | YES(wheel) | YES(wheel) | YES(existing) | YES(wheel) | ZIP 원본 HAA digest/고정 loader program에 묶인20units |
| `observationTransaction`/CPU watch/`pythonTransactionExecutor` | YES | YES(wheel) | YES(existing) | YES(wheel) | 기존 loop/terminate/cleanup을 재사용하는 ZIP 준비·검증 전략 |
| `SharedObserver`/C++ profile·topology·process role | YES(existing profiles) | NO(ZIP) | NO(ZIP) | NO(ZIP) | fixed `/haa-pipx` DATA topology와 ARTIFACT Python handoff |
| `ReleaseSetInspector`/required composite | YES(native/passive) | NO(ZIP) | NO(ZIP) | NO(ZIP) | B2 완료 결과를 원본 HAA identity의 required check/Evidence로 연결 |

### 재사용할 구현과 실제 전제

- `InspectWheelWithPlan`은 Project/Version·typed wheel plan과 validated wheel destinations를
  요구한다. ZIP을 wheel로 위장해 호출하지 않는다. 현재 private input이 가진 원본 acquired
  artifact와 immutable plan을 새 fixed consumer가 함께 받도록 한다. 선택된 version/digest는
  HAA subject이며 distribution 내부 pipx version과 교체하지 않는다.
- `executeTrustedTransaction`의 앞부분은 whole wheel manifest 재검증·pip 설치·installed
  manifest readback이다. ZIP은 이 준비 부분을 통과할 대상이 아니다. 반면 이후 unit launch,
  `beginUnit`/`finishUnit`, CPU sampling, observer usage, `terminatePhase`, `cleanup`과
  `finalize`는 이미 완료 책임을 갖는다. 이 기존 경로를 private 공통 단계로 분리해 재사용하고
  동등한 새 ledger/cleanup 구현을 복제하지 않는다. 기존 wheel 준비와 부정 검사는 유지한다.
- `pythonDirectObservation`은 `/haa-site`의 PathFinder/origin 검증과 sys.path.insert(0)를
  전제로 한다. ZIP용 고정 loader는 `-I -S -B`, pinned stdlib 경로 우선과 append된 exact ZIP,
  원본 module 존재·origin 검증을 적용해야 한다. 현재 runc import 진단은 이 새 external
  admission program의 qualification이 아니며 Python token/stdout를 완료 근거로 쓰지 않는다.
- `pythonObservationCreateArguments`와 `startPhase`는 shared closure volume 및 observer
  phase와 cgroup lease를 연결한다. ZIP 준비 writer→RO anchor/reader의 기존 C2 검사를
  이 실제 lifecycle에 연결한다. Host tmpfs noexec와 guest9p flags를 동일한 사실로 쓰지 않는다.
- `ObservationResourceClient.Create`의 기존 `pypi` class는 고정 Python 자원 한도를
  반환한다. 새 consumer가 같은 한도를 쓰더라도 artifact identity를 PyPI로 바꾸지 않는다.
  CPU reserve·transition accounting·trace credits·wall/launch 제한은 기존 owner가 유지한다.

### profile 분리와 두 번째 검토

C++ `IsPythonProfile`은 runtime loader만이 아니라 `ClassifyProcess`의 pip/uname와
`IsWorkspacePath`의 `/haa-site`도 함께 포함한다. 새 ZIP profile을 이 목록에 단순 추가하면
wheel 준비 권한까지 승계하므로 그렇게 구현하지 않는다. 필요한 pinned Python loader/read·
ARTIFACT direct handoff만 기존 predicate를 좁혀 재사용하고, 새 profile의 exact DATA mount
및 비허용 pip·다른 path/role·network·symlink/foreign attachment 거부를 검증한다.
Go의 expected topology/trace ledger와 C++ admission/분류가 같은 역할을 가리켜야 한다.

실행 순서는 (1) acquired typed 공급/고정20units, (2) exact ZIP profile/topology·기존 observer
latch 회귀, (3) 재사용 transaction의 실제 gVisor 단일 ZIP/모든 required units·failure/cleanup,
(4) required release composite다. Common transaction을 건드릴 때 기존 wheel·CPU/CUDA 및
UV/PNPM/Bun의 영향 범위를 먼저 정하고 실제 필요한 소비자를 검증한다. 미확정 실패를
resource/trust 확대나 자동 dependency 추가로 처리하지 않는다. Release13-assets 확장과
설치/서명/복구 membership은 B2 이후 같은 O4 map 순서를 따른다.

추가 caller 대조: `bootstrap/self_update_linux.go`의 factory는 현재 native inspector만
`NewReleaseSetInspector`에 전달한다. B2 이후 여기서 실제 ZIP runner를 공급하고 absence를
성공으로 취급하지 않아야 한다. 기존 PyPI dynamic report는 wheel/source 조건을 요구하므로
그 inspector를 HAA ZIP에 직접 재사용하지 않는다. Generic Domain check/Evidence 및
관찰 summary·findings 정규화 책임은 유지하면서 fixed ZIP required check로 연결한다.
`inspection/githubrelease`의 현재 architecture allow map에는 sandbox edge가 없으므로
실제 typed runner/result binding을 정할 때 필요한 edge만 구체적으로 검토한다. Core/
Application에 구체 sandbox/pypi 의존성을 추가하거나 미래 placeholder interface를 만들지 않는다.

## B2a — acquired 공급과 고정 unit의 source 전 세부 지도

CI420 종료 checkpoint 뒤 시작한다. M13-004 하나만 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| immutable plan/intake | YES | YES(private intake) | YES(local) | YES(input only) | sandbox의 acquired input 구성 |
| C1 DATA introducer | YES | YES(installed/fixture) | YES(local DATA) | YES(DATA only) | current 설치를 읽지 않는 typed acquired 공급 구성 |
| wheel unit identity/freezer | YES | YES(wheel) | YES(existing) | YES(wheel) | 같은 identity 계산의 재사용과 ZIP용 고정 program |
| ZIP program의 origin/stdlib 사전검사 | NO | NO | NO | NO | fixed stdlib 우선·원본 ZIP origin·단위별 새 interpreter |

기존 `pipxPreparedTool.introduce`/DATA transport는 그대로 사용한다. 새 private constructor는
원본 HAA source/name/version/variant/digest/handle와 selected immutable plan을 함께 보존하고
그 bytes로만 도구를 만든다. 기존 unit identity 계산을 작은 공통 helper로 옮겨 기존 wheel의
계산 결과를 유지하고, 새20unit에는 원본 HAA subject와 고정 ZIP program을 묶는다.
Program은 -I -S -B·고정 Python/stdlib 순서·원본 ZIP 내부 target origin을 검증한 후
ZIP을 append해 import한다. Unit coverage와 기존 필수 plugin은 바꾸지 않는다.

Zero/changed/foreign identity·중복/추가/잘못된 unit·취소를 거부하고 실제 selected ZIP의
20unit/원본 identity/기존 DATA roundtrip, 고정 image의 program 정상·부정 실행을 확인한다.
이는 source/input/loader만의 묶음이다. 외부 observer·완료 원장 실행·release wiring은
다음 B2b 이후에 기존 owner를 재사용해 연결하며 B2a가 Evidence를 발행하지 않는다.

B2a [로컬 receipt](./p4o4b2a-acquired-input-program-local-20261028.json)는
Quick239.916s/sandbox 전체 Race25.955s/actual6.716s/docs/security PASS와 원본 HAA
subject/immutable DATA·20unit·기존 transport 재사용을 연결한다. 고정 program의 runc20개
exit0와 preloaded/invalid/missing3개 사전 거부, 보호5파일/Docker/원본 불변을 확인했다.
첫 operator의 잘못된 파일 경로는 Docker/Python 전 파일 read에서 실패했고 정확한 retained
asset 경로로 교정했다. 제품 guard나 selected digest를 바꾸지 않았으며 최초 원본을 보존했다.
외부 관찰과 실제 transaction은 다음 B2b 이후에 남는다. 이 helper만으로 CI를 재실행하지 않는다.

## B2b — fixed ZIP observer의 source 전 세부 지도

기준4ab40c2다. 기존 shared observer/control protocol·ARTIFACT one-shot handoff·trace budget을
재사용한다. 새 profile 식별자는 `github-pipx-runtime` 하나이며 일반 GitHub ZIP 지원이 아니다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| control session/generation/nonce·mount sealing | YES(existing) | YES(existing) | YES(existing) | YES(existing) | fixed ZIP profile 등록과 typed topology |
| wheel Python runtime/process predicates | YES | YES(wheel) | YES(existing) | YES(wheel) | pip/site 권한을 제외한 ZIP 전용 분기 |
| trace ledger·phase budget | YES | YES(existing) | YES(existing) | YES(existing) | 새 fixed profile의 동일 default budget 허용 |
| fixed DATA topology/profile | NO | NO | NO | NO | preparation RW·anchor/observation RO, 단일 ZIP 읽기·쓰기/foreign 경로 거부 |
| 새 profile의 실제 transaction | NO | NO | NO | NO | B2c 이후 준비/실행/회계/정리 연결 |

`IsPythonProfile`은 wheel 전용으로 유지한다. 별도 fixed profile에서 pinned Python
classifier와 필요한 immutable runtime read만 같은 기존 predicate로 재사용하고 pip binary/
site-packages·pip.conf, `/haa-site`를 승계하지 않는다. ARTIFACT role·관찰된 executable
identity·정확한 proc identity 검사는 유지한다. DATA는 exact ZIP과 parent만 읽고 write·다른
member 경로는 거부한다. B2b 최초 구현은 direct admission을 `--handoff-python`만 받도록 제한했다.
이 일괄 CONTROL launch 거부는 아래 B2c의 canonical readiness 연결 대조로 반증되어
superseded다. ELF handoff 거부와 ARTIFACT Python 원칙은 유지한다.

Go factory의 phase별 `/haa-pipx` topology와 C++ 실제 registration/admission·latch를 함께
검증한다. Host tmpfs noexec와 guest9p flags를 혼동하지 않는다. Quick/관련 Race·pinned
compiler의 전체 observer latches를 먼저 실행하고, 외부 gVisor transaction은 다음 B2c에서
실제로 연결한다. Runtime/profile 추가를 기존 완성 경로의 재구현이나 예산 확대로 사용하지 않는다.

B2b [로컬 receipt](./p4o4b2b-observer-profile-local-20261028.json)는 Quick204.610s/
sandbox Race25.051s/docs/security PASS와 canonical observer build274.583s/전체
124latches PASS를 연결한다. 새 fixed ZIP read와 ARTIFACT-only Python admission의
production protocol 거부/정상 회귀를 포함한다. Helper는 빌드했으며 Host에 설치·활성화하지
않았다. 실제 acquired ZIP의 gVisor transaction·완료 원장·CPU/cleanup은 B2c에 남는다.

## B2c — 기존 transaction 연결의 source 전 세부 지도

기준3a5b3a2다. B2a input/program과 B2b profile/latches는 각각 로컬 검증했고 새 profile의
실제 acquired ZIP transaction은 아직 없다. M13-004 하나만 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| acquired bytes/20unit + fixed profile | YES | YES(named local helpers) | YES(local) | YES(named helper scopes) | 실제 backend entry |
| wheel manifest/install preparation | YES | YES(wheel) | YES(existing) | YES(wheel) | 기존 구현 유지; ZIP은 C2 DATA seed/bind/seal 사용 |
| unit loop·CPU watch·external finish·cleanup/finalize | YES | YES(wheel) | YES(existing) | YES(wheel) | 같은 기존 실행 함수를 ZIP에서도 소비 |
| actual ZIP gVisor transaction | NO | NO | NO | NO | helper/Host 사전상태·20unit·원본 identity·정리/회계 실제 검증 |

기존 `executeTrustedTransaction`의 wheel 검증 앞부분은 유지한다. 이후 shared ledger 생성,
unit loop와 cleanup/finalize를 private transaction spec을 받는 기존 공통 실행으로 옮긴다.
동등한 새 완료 원장·CPU watcher·정리 loop를 만들지 않는다. Spec은 두 trusted constructor만
만들고 arbitrary user mode/callback/path를 받지 않는다. Wheel은 기존 manifest/install/검증,
ZIP은 B2a input과 C2 volume/seed/bind/seal을 사용한다. 기존 wave의 명령·순서·coverage·
오류/observer/accounting 조건은 보존한다.

새 fixed ZIP entry는 wheel intake directory를 필수 입력으로 요구하지 않는다. 현재 backend의
command/probe/session 구성을 재사용하되 wheel introducer는 기존 wheel constructor만 요구한다.
새 entry는 원본 HAA identity와 frozen plan을 받고 pypi resource class의 기존 고정 한도만
사용한다. Image/CPU/memory/wall/trace 한도를 올리거나 context의 다른 PyTorch budget을
승계하지 않는다. ZIP phase container의 preparation memory도 기본 RuntimeMemory 이내다.

`startPhase`의 실제 mount/cgroup/observer 연결은 fixed content별 factory를 고른다. ZIP은
B2b typed topology와 같은 shared ledger를 사용한다. 각 unit 전후 C2 readback/readonly
attachment 검증을 연결하고 writer 제거 뒤 seal해야 관찰을 시작한다. Cleanup은 기존
terminate→resource close→owned volume disposal→finalize를 유지하며 UNKNOWN/첫 오류를
보존한다. B2b에서 빌드한 helper를 새 실제 입력의 observer로 사용하고 Host 보호 파일/
Docker 상태·source/helper identity·정리 원본을 묶는다. Fake wheel이나 Python stdout로
COMPLETED를 생성하지 않는다.

먼저 Quick/관련 전체 Race·정상/부정 회귀를 수행하고, 새 helper의 실제 ZIP transaction과
기존 wheel·작은 네 Python root policy 실패/정상 회계·필수 실제 CPU/hosted 소비자를 확인한다.
공통 observer/transaction의 전체 supported profile qualification은 각각의 실제 결과로
관리하고 대표 성공을 다른 CUDA profile로 확대하지 않는다. Release required composite와
최종 native pipx/CLI는 다음 연결 경계에 남는다.

B2c 첫 Go 연결 후보의 Quick202.663s/sandbox Race22.662s/docs/security는 PASS다.
그러나 source 대조에서 공통 `startPhase` → `awaitBoundaryHelper`가 고정 CONTROL
readiness launch를 필수 실행하므로 B2b protocol의 일괄 launch 거부와 충돌함을 확인했다.
실제 gVisor 실행 전의 확정된 연결 결함이며 runtime 실패 결과로 쓰지 않는다.
첫 source/checks는 `p4o4b2c-first-*` 원본으로 보존했다. 수정은 launch ARM을 허용하되
소비 시 helper binary/execfn·nonce·전체 고정 readiness argv만 받고, demotion 뒤 target도
같은 tuple로 재검증한다. 임의 shell/Python CONTROL·ELF handoff는 거부한다.
Diagnostic command phase는 권한에 사용하지 않는다. 기존 Go/C++ raw-string lexer로
readiness literal drift를 검사하고 실제 production parser의 정상/12개 부정 입력을 검증한다.

첫 actual v1은6.617s/INCOMPLETE, preparation의 SENTRY_EXEC/PROCESS_PROVENANCE_UNKNOWN으로
중단됐다(7expected events, Create/Register/Close각1, CPU227794usec). ZIP import는 미도달이다.
보호5파일/Docker/원본ZIP 불변, helper wait0/CONFIRMED_STOPPED·잔존helper0을 확인했다.
`p4o4b2c-first-actual-*` 원본과 assessment를 보존했다. 새 target predicate는 binary_path=/bin/sh를
요구하지만 기존 pinned OCI bootstrap은 /usr/bin/dash와 execfn=/bin/sh를 구분한다. 같은 정확한
resolved tuple과 실제 demotion→target chain 정상/변이5개 거부를 추가한다. 첫 trace에는 argv가
없으므로 첫 실패의 인과는 다음 actual 대조까지 미확정이다. Budget/role/topology와 전체 coverage는
완화하지 않는다.

같은 Go binary/선택 ZIP/default budget에서 target 경로 구분만 교정한 observer의
canonical build245.432s/124latches와 actual v2 87.436s가 PASS다. 정상39.79/39.36s 두 요청은
각20units/Register22/Close1·CPU12759290/12472671usec·post-install command5개를 완료했다.
중간 실제 Read 뒤 test-only 오류 주입6.43s는 INCOMPLETE/commands0/Register3/Close1이다.
실제 helper 자체의 장애 재현과 구분한다. Host 보호5파일/Docker/선택 ZIP 불변과
helperwait0/CONFIRMED_STOPPED·잔존helper0을 확인했다. Source condition 수정과 실제
준비→관찰 완료 대조로 첫 target 경로 오류의 named 해결을 확인했으며 한도를 변경하지 않았다.
기존 wheel/네 작은 Python root policy·prerequisite sequence 회귀와 최종 source 검증은 진행 중이다.

B2c [로컬 receipt](./p4o4b2c-acquired-runtime-transaction-local-20261028.json)는 최종 canonical
profiles·sandbox 전체 Race와 canonical124latches, 새 acquired transaction87.436s 및 기존
wheel/작은 네 policy·prerequisite sequence151.871s/noSKIP·helperwait0/Host 동일을 연결한다.
기존 완료 원장/CPU watch/unit loop/cleanup을 재사용한 named local 경계만 닫았다.
Required release composite/factory·배포/설치 membership과 native pipx/CLI·hosted/final full
profile matrix는 남는다. 다음 B2d에서 원본 HAA subject의 required check/Evidence로 연결한다.

## B2d — required inspector와 composition의 source 전 대조

기준548946a/tree c5b8cada/2784다. B2c acquired backend와 실제 transaction은 named local
QUALIFIED이며 전체 full profile/hosted 범위는 열려 있다. M13-004 하나만 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| acquired ZIP static reader/typed plan | YES | YES(explicit) | YES(local) | YES(input) | Inspector에서 같은 reader 소비 |
| ZIP external transaction | YES | YES(backend) | YES(named local) | YES(named local) | typed runner를 required check에 연결 |
| GitHub dynamic summary/findings normalization | YES | YES(native) | YES(existing) | YES(native) | 같은 함수를 fixed ZIP check에서도 소비 |
| post-install typed outcomes | YES(backend)/YES(wheel Evidence) | NO(ZIP report) | NO(ZIP report) | NO | exact five modules/owner/unit·disposition 검증과 HAA Evidence |
| release set/factory | YES(native) | NO(ZIP inspector) | NO(ZIP) | NO(ZIP) | fixed inspector/real resources 연결; complete DATA membership는 이후 배포 묶음 |

기존 GitHub dynamic의 Domain check/summary/Finding 정규화를 private 공통 함수로 추출해
native의 check/kind/오류/관찰 의미를 유지한다. 새 fixed ZIP inspector만 기존 static reader와
B2c runner를 연결한다. Required check는 `github-release-pipx-runtime`이고 원본 HAA identity/
digest를 보존한다. Completed라 해도 resource observation, 누락/중복/추가/잘못된 command
module·owner·unit은 완료 Evidence를 만들지 않는다. 기존 post-install 의미대로 exact typed
outcome을 같은 required check의 별도 Evidence에 담되 functionality/later enforcement는 false다.
Artifact 출력이나 wheel identity로 변환하지 않는다. 일반 ZIP/새 Core port는 만들지 않는다.

첫 source 지도는 inspection/githubrelease → sandbox typed result edge를 제안했다.
그러나 첫 Quick18.859s의 실제 test build에서 artifact/githubrelease 내부 test → inspector →
sandbox → terraformprovider → artifact/githubrelease 순환이 확정되어 이 제안은 superseded다.
Inspector가 소비할 bounded 결과/명령 타입을 소유하고 기존 composition root의 작은 adapter가
B2c typed 결과를 그대로 변환한다. 새 concrete import edge와 Core/Application 변경은 없다.
첫 source/check는 p4o4b2d-first 원본에 보존했다. 기존 artifact test를 우회하거나 compiler
검사를 줄이지 않는다.
기존 native pipeline fixture에는 ZIP runner 부재를 명시하고 선택되지 않은 옛 manifest는
유지한다. 실제 self-update factory는 held executor/observer와 기존 resource client로 새 backend를
전달한다. Manifest membership/설치·서명·복구의 완전한 DATA set은 다음 배포 묶음에서 함께
연결하며 지금 full release/pipx 지원 완료를 표시하지 않는다.

Go 기본/태그 회귀와 실제 selected ZIP intake→runner→Domain report/Evidence를 검증한다.
부정 typed 결과와 실제 정상·회계 오류는 완료를 분리하며 기존 native normalization/unsigned
release pipeline 회귀도 유지한다. 공통 runtime의 hosted 검증은 이 required inspection 경계와
묶고20분 실제 timer 조회를 적용한다.

두 번째 Quick58.828s는 import cycle을 통과한 뒤 새 단일 command array summary가
Domain Evidence의 기존1024B 상한을 초과해 기본 정상 회귀에서 실패했다. 원본 source/check는
p4o4b2d-second에 보존했다. 기존 wheel command Evidence처럼 명령별 bounded row를 같은
required check에 연결한다. 상한·Domain 계약을 바꾸지 않고 aggregate1+command5개의
Evidence를 요구한다. 실제 backend/typed outcomes와 원본 HAA identity는 유지한다.

세 번째 Quick42.413s는 command row summary 상한을 통과했고 긴 새 Evidence ID prefix가
기존64B identifier 상한을 초과한 오류를 검출했다. p4o4b2d-third 원본을 보존하고 기존
명령 Evidence처럼 짧은 `pipx-command-<disposition>-<unit hash>` identifier를 사용한다.
원본 HAA subject/check/kind와 전체64hex unit은 summary에 보존하며 한도는 바꾸지 않는다.

V4의 Quick41.719s/영향4 package 전체 Race51.296s/selected input1.262s와 docs/security가
PASS다. 추가 source 검토에서 새 pipx unavailable code를 기존 native inspector에까지 적용하지
않도록 exact check name으로 범위를 좁혔다. 실제 focused RED는 native unknown code의 기존
SUPPORTED/INCOMPLETE 대신 UNSUPPORTED/NOT_EXECUTED가 된 것을 검출했다. 새 pipx check만
이 code를 해석하도록 교정하며 두 소비자 상태 회귀를 유지한다. V4와 RED 원본을 보존했다.

B2d [로컬 receipt](./p4o4b2d-required-runtime-inspection-local-20261028.json)는 최종 Quick42.543s/
inspector Race2.132s, 앞선 affected4 package full Race51.296s, selected ZIP1.308s/docs/security와
실제 required inspection41.606s를 연결한다. 원본 HAA acquired identity의 required check1개/
Finding0/Evidence6·commands5가 완성됐고 이후 intake 변조는 거부됐다. NoSKIP/helperwait0/
confirmed cleanup·보호5파일/Docker/원본ZIP 동일이다. 처음 세 실패와 native capability RED를
보존했으며 현재 상한·계층 경계·native 의미는 유지했다. 실제 complete release DATA graph와
build/sign/install/recovery·native pipx tools/CLI 및 final profiles는 남는다. 다음 B3에서
complete membership를 연결하고 이 required inspection 경계를 공통 hosted checkpoint로 묶는다.

## B3 — complete DATA 배포 연결의 source 전 대조

기준 e859487/tree6753df83/2789다. B2c actual external transaction과 B2d required inspector는
named local qualified다. 원격은1e5d21e이며 e859487까지 여섯 commit의 push가 자동 승인 검토에
거부되어 사용자 승인 대기다. 이를 우회하지 않고 같은 M13-004의 독립 로컬 작업을 계속한다.
새 CI/20분 timer 조회는 아직 시작하지 않았다.

| 책임/기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| manifest strict parser·native6 selector | YES | YES(existing) | YES(existing) | YES(existing scopes) | fixed ZIP/NOTICE pair·새15-asset set, native6 유지 |
| release graph/required inspection/frozen proofs | YES | YES(native)/YES(ZIP inspector only) | YES(named local) | YES(named scopes) | complete graph의 ZIP DATA 포함·actual current check |
| promotion/bootstrap/wire/protected copy/recovery | YES | YES(native+metadata) | YES(existing) | YES(existing scopes) | 같은 DATA membership와 nonexecutable modes·bounds·cleanup |
| installer/installed reader | YES | YES(native/C1 reader) | YES(named scopes) | YES(named scopes) | ZIP0444·NOTICE0444 선택/설치/검증·옛 설치 형식 보존 |
| release manifest/SBOM/runtime metadata | YES | YES(13 assets/2 tools) | YES(existing) | YES(existing) | selected pipx ZIP/NOTICE·whole supply metadata 표시 |
| build/package/attest/publish exact set | YES | YES(existing) | YES(existing) | YES(existing) | 같은 새 DATA set의 생성/검증·OS package 포함 |

한 번에 기존 완성 구현을 대체하지 않는다. 아래 책임 단위로 현재 gap만 잇는다.

### B3a — graph/protected DATA set

- `NativeUpdateFiles`의 native6와 기존 notices 요구는 유지한다. 같은 manifest owner에
  native selector를 재사용하는 complete payload selector를 두어, 현재 여러 owner가 반복하는
  native+metadata 선택에 정확한 ZIP DATA를 추가한다. 기존13-asset native set의10payload와
  새15-asset set의12payload를 각각 완전한 집합으로 검사한다. Manifest 자신을 포함한 subject는
  각각11/13개이며 고정 total1GiB/file512MiB 및 기존 proof/frame 상한은 올리지 않는다.
- ZIP의 정확한 role은 `pipx-runtime`, platform linux/amd64, NAME은 selected owner의
  `pipx-runtime-linux-amd64.zip`다. `pipx-NOTICE.txt`는 기존 release-metadata role이다.
  ZIP/NOTICE는 pair이며 기존 Bun/PNPM/installer가 있는 완전한 set에서만 허용한다.
  Structural reader의 역사8/9/11/13-asset 허용은 보존하고 producer는 최종15개를 요구한다.
  옛 형식 수용을 pipx 지원 qualification으로 표시하지 않는다.
- `ReleaseFile.IsRuntimeData`는 위 fixed name/role/platform을 식별하는 mode 분기다.
  DATA는 안전성·검사 완료 판정이 아니다. system 설치0444, private root staging0600으로 두며 user 설치의 기존 metadata0600은 보존한다.
  executable0555/0500 경로에 넣지 않는다. 같은 분기를 installer·root stage·installed hash·
  bootstrap/cleanup에 전부 연결해야 새 manifest가 실행 경로에 도달할 수 있다.
- 기존 graph·promotion·bootstrap·wire·installed read/cleanup owner는 complete selector를
  소비한다. 누락/추가/치환/중복·UNKNOWN·후속 cleanup이 첫 실패를 덮는 기존 거부를 유지한다.
  원본 HAA identity와 ZIP required check를 보존하고 native ELF 검사로 ZIP을 보내지 않는다.
- Exact NOTICE hash/size 검증은 기존 metadata owner에 추가한다. Actual ZIP/whole plan 검사는
  B2d inspector가 소유하고 installed 실행은 C1/C2 exact selected identity가 다시 검증한다.
  DATA transport로 artifact를 Host에서 import하거나 root 준비 중 실행하지 않는다.

### B3b — producer와 소비자의 같은 완전 집합

기존 materialize-pipx와 runtime owner를 그대로 사용해 frozen11-wheel/42input의 actual ZIP과
NOTICE를 생성한다. Release manifest/SBOM/runtime metadata/OS packages·build/attest/bundle
verification/publish list를 같은15-asset set으로 확장한다. Metadata의 DATA 기록을 native ELF
실행 역할로 사용하지 않는다. 전체 supplied wheel·제공 proof/NOT_PROVIDED 구분을 보존하며
새 dependency 자동 선택·source pin refresh·install-time download는 없다. 필요한 fetch도
selected frozen inputs만 받으며 기존 materializer가 전체 bytes/hash/proof를 재검증한다.

### 검증과 두 번째 검토

먼저 legacy parser/native6·새 pair/역할/크기·complete subject set·DATA mode의 정상/부정
회귀를 검사한다. 실제 selected ZIP→complete graph→required current Evidence·Policy가
도달하는지 확인하고, root copy/installed read/wire/rollback·foreign/unknown cleanup도 현재
owner에서 검증한다. 새 unsigned fixture는 authenticity 성공으로 쓰지 않는다. 실제 서명·
activation은 승인된 기존 controlled fixture/절차 범위만 사용하고 main merge·공개 배포·
영구 runtime 교체를 하지 않는다. 소유 경로의 canonical Quick/관련 Race/docs/security 및
producer/workflow/release-gate를 순차 실행하며 full profile/hosted 범위는 별도 증거로 닫는다.

B3a [로컬 receipt](./p4o4b3a-complete-data-selection-local-20261028.json)는 초기 Quick187.064s/
affected6 owner full Race171.897s와 추가 graph/wire/route 후보의 Quick42.684s/관련3 owner
Race14.630s/docs/security PASS를 연결한다. Native6와 legacy10/new12payload·11/13subjects,
strict pair/role/platform/bounds, DATA의 native fallback 부재와 unsigned wire의 실제 authenticity
거부를 확인했다. Actual selected ZIP/NOTICE의 private mount/network namespace 복사0.221s는
root single-link0444·bytes/Host/Docker 불변/noSKIP이다. 서명·artifact 실행·full release 성공이
아니다. Producer/metadata·실제 complete graph current Evidence/Policy와 root DATA 복구·native
pipx/CLI 및 hosted/final profiles는 남는다. 다음 B3b를 로컬에서 계속하며 e859487 push는 승인 대기다.

## B3b — producer/metadata/같은 build set의 source 전 세부 지도

기준3eab830/tree254e9d11/2795다. B3a의 parser/selector·graph/wire/protected mode는 named local
검증했고 새15-asset producer와 실제 complete current inspection/복구는 아직 없다.
E859487 checkpoint의 push는 승인 대기이며 재시도하지 않는다.

| 기존 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| selected42-input offline materializer/NOTICE | YES | YES(developer) | YES(named local) | YES(materializer scope) | release/CI의 같은 입력 준비·호출 |
| frozen official download·size/hash 검증 | YES(SDK downloader) | NO(pipx inputs) | NO(pipx fetch) | NO | 기존 downloader와 owner/metadata를 재사용하는 고정 wheel 취득 |
| runtime-image manifest/strict passive validator | YES(v2/2 tools) | NO(pipx DATA) | NO(pipx record) | NO | 명시 v3·DATA role/mode/Python binding, legacy v2 pair 경계 |
| release assets/SBOM/size-hash binding | YES(13 assets) | NO(pipx pair) | NO(pipx producer) | NO |15assets와11 distribution inventory·same selected owner |
| package payload/build/attest/bundle/publish | YES | YES(existing) | YES(existing scopes) | YES(existing scopes) | same ZIP/NOTICE/manifest 집합의 생성/복사/검증 |
| pipeline graph/current inspection | YES(B2/B3a) | YES(selected adapter) | YES(named boundaries) | NO(complete actual) | actual selected ZIP의 전체 graph/current Evidence·Policy와 root 복구 |

### 수정 범위와 재사용

1. `scripts/update-runtime`에 현재 release/CI 준비용 `fetch-pipx-supply`만 둔다. 현재 source
   owner의42개 path/hash와 canonical31 metadata/proof bytes를 먼저 확인한다. Frozen metadata에서
   exact filename/size/hash·HTTPS files.pythonhosted.org URL을 고르고 기존 bounded downloader를
   사용한다. Redirect 예외·ambient proxy/credential·live version resolve를 만들지 않는다.
   새 외부 private directory에만 input을 보관하고42개 bytes/hash 확인 뒤 입력 준비 scope를
   출력한다. Wheel/code는 실행·추출하지 않는다. 제공9proof와 두 NOT_PROVIDED를 그대로
   보존하고 기존 materialize-pipx가 whole wheel/proof/closure/ZIP/NOTICE를 다시 검증한다.
2. Runtime image manifest는 명시 v3으로 확장한다. 기존5images·PNPM/Bun records는 보존하고
   pipx record에 fixed DATA role, mode0444, pinned Python image와 ZIP/NOTICE identity를 둔다.
   기존 validator를 재사용하는 release-bound entry가 manifest의 DATA pair와 v2/v3·selected
   identity를 함께 검사한다. Legacy pair 부재의 v2는 유지하며 새 pair가 있는 release가
   old metadata로 완료되거나 반대 상태가 되는 것은 거부한다. Metadata가 runtime/tool을
   선택하거나 Host authority를 만드는 흐름은 없다.
3. 기존 release producer는 fixed15assets를 요구하고 ZIP/NOTICE의 size/hash를 같은 runtime
   owner에 묶는다. SBOM은 기존 Go/native 목록에 pipx와 selected input의11 distribution
   census를 연결한다. Wheel filename은 source-owner inventory metadata로만 파싱하며 signature/
   safety authority가 아니다. 실제 whole wheel 의미는 materializer와 B2 inspector가 소유한다.
   Actual ZIP의 원본 plan census와 producer SBOM을 별도 대조한다. 새 version owner나
   임의 최신 dependency 목록을 만들지 않는다.
4. OS package에는 같은 ZIP과 NOTICE를0444 DATA로 추가한다. 기존 본체/helper modes와
   package lifecycle·network download 금지·native 관리 설치 자동 등록 금지를 유지한다.
5. Build workflow는 frozen fetch→기존 offline materialize→package/manifest→현재 attest 순서를
   명시한다. Asset/attestation/checksum/bundle verifier/publish 목록과 independent workflow
   기대값을 함께 잇고 누락/추가/치환/조건부·주석-only 명령을 거부한다. 작은 별도 CI job은
   만들지 않는다. 현재 공통 checkpoint push는 승인 전 실행하지 않는다.

### 검증과 재검토

가장 작은 metadata/asset/URL/owner·명령별 부정 검사부터 시작한다. Canonical Quick와 영향받은
전체 Race·workflow/release-gate를 순차 실행한다. Default test는 외부 wheel/cache를 요구하지
않고 synthetic/기존 공개 metadata fixture의 한계를 표시한다. 실제 frozen42input fetch/
materialize·15asset 생성/bundle/package 검사와 actual ZIP census 대조를 별도로 수행한다.
Metadata v2/legacy13 fixture와 v3/새15 pair를 혼합하지 않고 published 원본은 변경하지 않는다.
그 뒤 실제 complete graph/current Evidence·Policy와 protected DATA recovery를 검증한다.
서명·설치·공개 배포와 named local/hosted/full profile 범위를 구분하며 상위 acceptance를
부분 성공으로 닫지 않는다.


### B3b named local 결과 — 2026-10-28

[한정 영수증](./p4o4b3b-producer-data-local-20261028.json)이 source/각 체크/원본 inventory와
최초 operator·test compile 오류를 연결한다. 고정31문서·11wheel/42inputs는 network 전에 owner
대조 후 기존 downloader로 취득하고 기존 offline whole wheel/proof/closure 검증기로 같은
ZIP/NOTICE를 생성했다. 제공9proof/두 NOT_PROVIDED·예산·required 관찰은 변하지 않았다.
새 runtime metadata는 명시 v3/5images+3tools/DATA0444·Python/ZIP/NOTICE binding이며,
legacy v2와 DATA pair 혼합을 거부한다. 같은15assets/19bundle/DEB·RPM 원본 DATA0444와
SBOM11distribution을 실제 ZIP 원본 plan에 대조했고 두 package의 재생성도 byte-identical이다.
Host 보호5파일/Docker/입력은 같고 noSKIP다. Fixture v1.2.3/commit/workflow label은 서명·실제
hosted build identity가 아니며 native binaries compile과 package 생성만 수행했다.

B3b1 Quick229.165s/affected6 owner Race47.233s와 B3b2 Quick63.583s/두 scripts 전체
Race40.331s·workflow/release-gate/docs/security를 통과했다. 최종 test source는 별도 Quick와
realcorpus go vet 결과에 연결한다. Publish는 기존 bundle 검증기의 exact filename 목록을
이미 소비하므로 중복 목록·새 job을 만들지 않았다. Actual graph/current Evidence·Policy,
protected DATA 복구, native pipx/CLI·hosted/final profile 범위는 B3c 이후에 남는다.


## B3c — actual graph/current Evidence와 DATA 복구의 source 전 대조

기준e018227/tree8915d55c/2802다. B3b의 실제15asset/19bundle·metadata/SBOM/package 검증은
같은 source inventory로 닫았으며 서명/hosted/새 required 관찰을 대신하지 않는다.

| 기존 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| frozen HAA graph13/native6/DATA pair | YES | YES | YES(default graph mechanics) | NO(actual current pipeline) | coherent actual ZIP/NOTICE/v3 fixture의 전체 current pipeline |
| required ZIP inspector/production composition | YES | YES | YES(B2d actual isolated inspector) | NO(complete pipeline) | same graph/current Evidence·Policy·staging와 missing runner·unsigned proof 거부 |
| whole-set pipeline test fixture | YES | YES(legacy11 subjects) | YES(existing named scope) | YES(existing scope) | original selected DATA pair를 기존 fixture에 붙인 explicit realcorpus consumer |
| root wire/private stage/cleanup | YES | YES(complete payload selector) | YES(legacy namespace/default wire) | NO(DATA actual) | real ZIP의 full12payload stage/bytes/mode·corrupt/truncated refusal/cleanup |
| bootstrap/partial-copy recovery/source cleanup | YES | YES(complete payload selector) | YES(legacy namespace/B3a copy) | NO(DATA actual) | original ZIP/NOTICE가 있는 named namespace 복사·중단 재개·정리 |
| full new signed native/runtime/CLI/hosted | YES(existing owners) | YES(partial current tuple) | NO(current full set) | NO | new signed qualification·native pipx/hosted/final matrix |

### 재사용과 명시 범위

1. HeliopauseReleaseSet/NewReleaseSetInspector/기존 M6·M4 Policy/application install/staging을
   그대로 쓴다. 기존 test fixture를 DATA pair와 BindCurrentRuntime으로 첫 resolve 전에
   일관되게 갱신한다. 새 resolver나 graph/Policy 모델·production test hook은 만들지 않는다.
   외부 artifact integration test의 test-only typed adapter가 기존 sandbox/backend와 Host
   resource client를 연결한다. Production composition 자체의 검증 범위는 B2d actual로 유지한다.
2. 실제ZIP의 required20unit completion/Evidence6를 whole graph13subjects의 새 current
   검사·Evidence·Policy로 소비한다. 나머지 native6는 기존 source/staging fixture의 fake
   sandbox이며 그 실행 성공·authenticity로 확대하지 않는다. Positive의 integrity-only
   verifier도 fixture 전용이며 별도 production SDK는 unsigned set을 계속 거부해야 한다.
   Missing ZIP runner는 required incomplete로 staging/promoter에 도달하지 않아야 한다.
3. 기존 root namespace test의 auth/current/proof mock 경계를 그대로 표시한다.
   Production prepareWireRelease/readBootstrapSet/cleanupBootstrapSource의 unsigned
   거부와 private filesystem primitive의 positive를 분리한다. 실제 build candidate의
  12payload와 ZIP/NOTICE 원본·root private0600/installed0444를 검증하고 새 auth 권한이나
   persistent Host 설치를 만들지 않는다. 부분 copy/오류 후 정리는 source/대상 inode·bytes/
   mode를 확인하고 UNKNOWN/foreign 교체를 삭제하지 않는다.
4. 기본 테스트에는 외부 wheel/selected ZIP cache를 추가하지 않는다. Explicit realcorpus
   또는 root namespace 입력을 사용하고 고정 source/check binary/observer/ZIP·Host 보호5파일/
   Docker/helper 종료를 원본으로 연결한다. Quick·affected Race/tag vet와 actual을 겹치지 않는다.
   기존 C2 six-file guards·예산·required coverage를 보존한다.

수정/검증 대상은 기존 test fixture와 explicit actual tests이며 product 경로는 baseline 구현을
재사용한다. 실제 consumer가 구체 gap을 반증할 때만 해당 owner를 최소 수정한다.


### B3c2 partial-copy mode의 직접 반증 — 수정 전

기준e018227의 pending_unix.go/DiscardSystemCopy는 input.files의 intended mode를 모두0555로
설정하지만 installer.go의 기존 copier는 release-metadata/IsRuntimeData를 metadataMode
(system0444)로 쓴다. 같은 실제 source의 root namespace v1(10.645s)은 full12payload/
255,942,560B root0600 stage/세 오류 정리와 새 DATA bootstrap 정상·부분 복구를 통과한 뒤,
0555 DATA partial copy를 거부해야 하는 negative에서 실제 cleanup/재설치를 허용해 실패했다.
Host5파일/Docker/원본 ZIP은 같다. Auth/crypto filesystem doubles의 named 범위이며 실제
signed/public 설치 취약점 실행이나 B3a 도입 인과를 주장하지 않는다.

이는 기존 mode owner의 consumer gap이다. Copier의 기존 역할별 mode 선택을 private helper로
재사용해 pending cleanup도 system0444/0555를 똑같이 소비하게 한다. 기존 native mode나
public/user mode·prefix/inode/ownership/whole-set-before-delete guard를 변경하지 않는다.
동일 namespace test SHA의 RED→GREEN과 기존 affected checks/namespace를 확인한 뒤에만
해결로 기록한다. 기존 root operator의 named_pass substring은 하위 PASS를 부모 PASS로
집계했으나 exit1 때문에 전체 성공으로 닫지 않았다. 원본 scalar를 편집하지 않고 후속
operator는 exact top-level PASS 정규식으로 판정한다.


B3c2 v2의 동일 namespace test SHA6356a578은8.246s/네 top-level PASS(noSKIP)로
RED→GREEN이다. 실제12payload255942560B stage·세 실패 정리·DATA 정상/부분 복구·tamper
거부 뒤 재시도·foreign 보존과 legacy bootstrap/stage를 함께 확인했다. 보호5파일/Docker/
원본은 같다. 같은 owner의 기존 pending namespace complete fixture는 NOTICE도0555로
합성한 것을 source 대조에서 발견했다. 실제 copier/기존 system namespace 기대는
NOTICE0444이므로 독립 fixed notice 두 이름의 fixture mode만 바로잡고,24case/SIGKILL의
기존 실제 pending 시험도 수행한다. 발견 전 실행 실패를 만들어낸 것으로 기록하지 않는다.


### B3c named local 완료와 남은 소비자 — 2026-10-28

[한정 영수증](./p4o4b3c-current-graph-data-recovery-local-20261028.json)이 각 source/check/
actual과 최초 RED를 연결한다. 최종 composed source의 실제 ZIP20unit 관찰이 graph13subjects/
32checks/37Evidence와 current store readback·기존 Policy/staging으로 이어졌다. Missing runner는
required incomplete로 staging을 막고, 별도 production SDK negative는 실제 ZIP 관찰 완료와
무관하게 unsigned set을 거부했다. 나머지 native6는 기존 sandbox fixture·promoter는 spy이며
native 실행/authenticity/Host activation으로 확대하지 않는다.

Same test SHA의 DATA partial-mode RED→GREEN, actual12payload255942560B root private0600
stage/세 오류 정리·DATA installed0444/정상·재시도·부분 복구·변조 거부·foreign 보존과 legacy
bootstrap/stage 및 pending24case/SIGKILL을 통과했다. Copier의 기존 mode 선택만 공유해
cleanup ceiling을 맞췄으며 기존 원본 prefix/inode/ownership/whole-set guard는 유지했다.
Host5파일/Docker/selected ZIP 불변/noSKIP와 graph helperwait0/confirmed stopped를 확인했다.
최종 Quick37.624s/affected2 owner Race4.148s/tag vet/docs/security도 PASS다.

B3d에서는 existing hosted actual consumer의 명시 입력/setup/UID1001/cleanup 연결을 먼저
대조한다. 새 job을 만들거나 default test에 외부 cache를 숨기지 않는다. 그 뒤 installed pipx
runtime/native tools/CLI와 signed/current activation 및 full supported profiles는 별도로 닫는다.
e859487까지의 기존 push 승인 대기는 그대로이며 새 CI/20분 timer는 아직 시작되지 않았다.


## B3d — existing hosted actual 소비자의 source 전 대조

기준2c52c03/tree13388565/2806이다. B3c 최종 graph84.436s와 root8.246s/pending0.220s는
named local qualified이며 새 hosted 결과가 아니다. Push 승인 대기는 e859487까지의 기존
여섯 commit에 한정한다. 승인이 도착해도 이후 HEAD를 임의 포함하지 않는다.

| 책임/위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| frozen fetch/materialize-pipx | YES | YES(release build) | YES(actual42inputs) | YES(named scope) | ordinary Linux CI 실제 소비자의 명시 공급 |
| B2d actual required inspector | YES | YES(local protected bootstrap client) | YES(local) | NO(hosted) | 기존 tagged bootstrap test client에서 fixed input actual 호출 |
| B3c actual current graph test | YES | YES(local protected artifact test client) | YES(local) | NO(hosted) | 기존 job 내 artifact test binary/client/helper/cwd·입력 연결 |
| root DATA bootstrap/cleanup·legacy pending | YES | YES(local namespace) | YES(local) | NO(hosted) | 기존 job 내 explicit root mount/net namespace commands |
| full12payload255942560B wire actual | YES | YES(local build candidate) | YES(local file mechanics) | YES(named scope) | ordinary CI에 full producer build를 숨기지 않음; protected signed/full set qualification은 별도 |
| ci-integration-cleanup helper/client lifecycle | YES | YES(existing owners) | YES(existing named local/CI420) | YES(existing scope) | 새 concrete client의 같은 owned replacement/restore 순서 |
| independent workflow/script checker | YES(existing clients) | YES(existing) | YES(existing) | YES(existing) | 새 pipx mode/env/filter/package cwd·compile/client/helper/order/cleanup 누락의 부정 회귀 |

### 수정 지도·명시 입력과 재사용

- 새 CI job·image·dependency·timeout·관찰 예산을 만들지 않는다. 기존 Linux integration
  job/owned foreground helper/client replacement와 마지막 production restore를 재사용한다.
  기존 PNPM/Bun/UV concrete contiguous recipes를 보존한다. 추가 block은 기존 PyPI dynamic
  종료 뒤 production client restore 앞에 두고 tagged bootstrap client→inspect, 새 tagged
  artifact client→graph, explicit namespace tests→기존 production restore 순서로 잇는다.
- 현재 work에 필요한 scripts/ci-pipx-runtime.sh는 inspect/graph/namespace의 exact mode만
  받는다. 각 호출은 새 RUNNER_TEMP private directory에서 같은 frozen42inputs를 취득하고
  기존 offline materializer로 같은 ZIP/NOTICE를 만든다. 숨은 /tmp wheel cache·live version
  resolve·install-time augmentation은 없다. 기존 CI client가 호출을 소유하고 script는 부모
  helper lifecycle/trap을 바꾸지 않는다. 자기 mktemp directory만 정리하고 첫 오류를 보존한다.
- Graph는 현재 test fixture의 explicit SBOM 입력을 쓴다. Local B3c는 actual B3b producer
  SBOM을 썼고 hosted source/staging fixture는 canonical v0.1.0 producer SBOM을 명시 복사해
  기존 BindCurrentRuntime으로 synthetic identity/current runtime에 rebind한다. 원본은
  변경하지 않는다. Native6 fake sandbox/integrity-only positive와 unsigned SDK negative의
  한계를 유지한다. Full current producer11distribution census는 B3b actual가 소유한다.
  이 explicit historical input의 실제 local recipe도 별도로 검증한다.
- Namespace mode는 DATA bootstrap5case/기존 bootstrap·stage 및 기존 pending24case/SIGKILL을
  root mount+network namespace에서만 실행한다. Full original native payload wire 시험은
  B3c의 actual complete producer scope로 유지하며 ordinary CI에 미작성 producer fixture나
  임의 native stub 디렉터리를 몰래 만들지 않는다. No registration/runtime activation이다.
- 새 driver는 Go test exit만 보지 않고 exact top-level required PASS와 noSKIP를 확인한다.
  Empty filter/no-tests·하위 PASS·SKIP·exit failure를 기본 harness 회귀에서 거부한다.
  이것은 trusted Go test dispatch 검증이며 artifact stdout에 제품 완료 authority를 주지 않는다.
- 기존 scripts/check의 concrete script/env/filter/cwd와 client compile/owned helper/마지막
  restore 기대값을 새 owner 전용으로 확장한다. 주석/echo/누락/중복/다른 package/무시된
  오류/조건부 step과 wrong client·restore 생략을 부정 검사한다. Bash syntax와 실제 stubbed
  dispatch/failure cleanup, canonical Quick·checker 전체 Race/workflow/release-gate/docs/security,
  실제 fixed corpus/보호 client source·UID1001 조건을 순차 대조한다. 실행 후보를 변경하며
  actual/check를 겹치지 않는다.

새 scope를 기존 cohort의 qualification으로 포장하지 않고 exact inputs/source/client/helper/
cleanup/first failure를 보존한다. 사용자 승인이 없으면 새 push·CI/timer를 시작하지 않는다.


B3d의 initial concrete tests/workflow와 shell harness는 PASS였다. 첫 전체 Quick80.090s는
TestRuntimeVersionOwnersAllowCoordinatedFutureGoInputs의 runtimeOwnerFixture가 새
ci-pipx-runtime.sh를 복사하지 않아 strict workflow checker의 unavailable 거부에서 실패했다.
Logical future Go fixture의 기존 explicit helper inventory에 새 script만 포함한다. Guard나
runtime pin/미래 SDK 지원 범위를 바꾸지 않고 첫 source/Quick를 p4o4b3d-first에 보존했다.


B3d 첫 actual0.297s는 제품/driver 호출 전에 private app-parent overlay 준비의 원본 gVisor
inode assertion에서 중단됐다. Host5파일/Docker/입력·helper0은 같고 모든 actual test는
NOT_REACHED다. /tmp-only 별도 mount namespace의4조합 대조에서 libmount 기본 canonicalization은
held parent/child FD 모두 새 가림 경로를 다시 선택했고 --no-canonicalize는 둘 다 원본
dev/inode를 보존했다. Operator에 그 flag만 추가하며 제품 runtime permission·trusted path/
원본 inode guard를 느슨하게 하지 않는다. 첫 원본과4조합은 p4o4b3d-recipe-actual-v1 및
p4o4b3d-bind-probe-result.json에 보존했다. 새 source/check 변경이 아니므로 canonical PASS를
임의 반복하지 않고 같은 actual recipe를 corrected namespace operator로 검증한다.


### B3d named local recipe와 다음 경계 — 2026-10-28

[한정 영수증](./p4o4b3d-hosted-actual-wiring-local-20261028.json)이 actual135.807s의6개 exact
top-level/noSKIP, 네 helperwait0/confirmed cleanup·client 복원/입력 directory0·Host5파일/
Docker/소스 불변을 확인했다. 원본 gVisor dev/inode는 RO bind로 보존했고 client 교체는
private app-parent namespace만 바꿨다. Current frozen42input/ZIP/NOTICE와 explicit historical
SBOM source/staging fixture로 세 driver mode를 그대로 실행했다. 로컬 ordinaryUID1000
범위이며 hostedUID1001·새 CI qualification이 아니다.

Driver harness는 empty/no-tests·SKIP·child-only/duplicate PASS·nonzero exit·준비 실패와
cleanup 실패를 실제 Bash로 검증했다. Independent concrete workflow/driver guard와 최종
Quick47.782s/checker 전체 Race40.351s/workflow/release-gate/docs/security도 PASS다.
새 job/image/dependency/timeout/관찰 예산은 없고 PNPM/Bun/UV recipes 및 기존 owned helper/
마지막 production restore를 유지했다. Root auth/current/crypto와 native6 sandbox는 기존
filesystem/source fixture 범위이며 실제 Host activation으로 확대하지 않는다.

다음은 P4-O4-C3 installed pipx runtime/native install·upgrade/CLI owner audit다.
ParsePipxToolReceipt와 fixed installed DATA reader/RO transport는 이미 있으므로 새로 쓰지
않고 실제 consumer gap만 확인한다. Hosted/current signed/full support profile 범위는
별도이며 e859487까지의 push 승인 대기는 계속한다.


## C3 — installed pipx/native persistent tool의 source 전 baseline

2026-10-28 source `a324c452af58101f0c94cda9120c872de5a0bd5f`, tree
`f473631e66256a41c046a745778ed836014a0588`/2811파일의 tracked working tree는 clean이다.
Private check source의 HEAD는 과거 기준선이지만 index/blob/mode와 전체 disk는 위 후보와
동일하게 유지한다. 기존 e859487 checkpoint push만 사용자 승인 대기이며 새 CI는 없다.

| 기존 책임 / 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| `pypi.ParsePipxToolReceipt`, `ToolReceipt.UpgradeRequirement/ValidateLayout/ValidateExposedCommands`, `tool_receipt_test.go` actual installed/upgraded0.12 fixture | YES | YES (inert parser/tests) | YES (기존 named parser) | NO (C3 consumer) | 실제 native output/Host guard 소비 연결; parser 재구현 없음 |
| `ToolReceipt.BindWheels`, `tool_wheels.go` complete original wheel/source/lock 및 pipx startup-hook 거부 | YES | YES (기존 graph binding) | YES (기존 named parser/negative) | NO (C3 native consumer) | native pipx complete selection과 연결; `.pth`/customization 지원 확대 없음 |
| `pipx_tool_supply.go` installed current root/file-held DATA reader와 `pipx_data_volume.go` seed/RO anchor/seal/readers | YES | YES (B2 acquired consumer/C2 tests) | YES (named local) | NO (installed native consumer) | 현재 설치 supply로 native 실행 준비; RW native-volume 우회 금지 |
| `uv_resolver.go` native lifecycle/typed public network/trace/cleanup, `uv_tool_resolver.go` UV tool native selection | YES | YES (UV만) | YES (CI417 등 UV named scope) | YES (기존 UV named scope) | pipx fixed Python ZIP loader/closed operation/profile, backend UV child/query provenance 및 pipx receipt/complete census export adapter |
| `uv_tool_layout.go` explicit held-root metadata projection | YES (UV만) | YES (UV) | YES (UV named) | YES (UV named) | pipx internal app paths/source-interpreter를 명시적으로 구분하는 inert projection; UV client 거부 유지 |
| `uv_tool_upgrade_input.go`, native metadata-only original replay/installed wheel controls | YES (UV만) | YES (UV) | YES (UV named) | YES (UV named) | pipx schema/backend와 unconstrained upgrade 의미의 별도 bounded original input; stored pip_args/Host interpreter 실행 금지 |
| `UVToolInstallPreparation`, existing target/command locks/current whole-set Policy/atomic materializer/rollback, bootstrap/CLI descriptor | YES (UV만) | YES (UV) | YES (UV named) | YES (UV named) | 기존 transaction 재사용을 위한 pipx adapter, fresh/managed/native original entry와 CLI validation/help/current approval/retired scripts/negative/rollback |
| B3 complete ZIP/NOTICE release graph/installed DATA/producer/package/CI recipes | YES | YES | YES (named local) | NO (hosted/signing/native final) | 새 hosted UID1001, signed native activation 및 final matrix; C3 성공으로 대체하지 않음 |

### C3a — inert receipt projection의 수정 지도

첫 source 단위는 `internal/artifact/pypi/pipx_tool_layout.go`의 작은 adapter다.
이미 구현된 strict0.12 parser와 generic held layout/command validation을 그대로 호출한다.
Caller가 명시한 environment·별도 bin directory·source interpreter에 원본 app_paths와
source_interpreter만 재배치하고, 원본 bytes와 requirement/version/backend/options/노출 집합,
필드 생략·null을 보존한다. Pipx source_interpreter는 source Python claim이며 UV처럼
무조건 environment/bin/python으로 해석하지 않는다. 이 inert 변환은 Host 경로를 열거나
실행하지 않으며 ownership/runtime admission/Policy를 부여하지 않는다.

기존 UV `WithUVLayout/WithUVSelectionLayout`의 pipx 거부를 유지한다. Pipx projection도
UV/zero/캐시와 원본 불일치·잘못된 source interpreter·명령/환경 identity·중첩된 roots를
거부한다. Installed/upgraded actual fixture의 constraint/retired/new commands, opaque
pip_args와 원본의 불변, 누락/명시 null 보존을 검사한다. 해당 package 전체 Race와
canonical Quick/docs/security를 실행하고 named local 범위만 기록한다.

### 두 번째 검토와 후속 순서

고정 ZIP hash 또는 DATA 역할은 CONTROL 실행 권한이 아니다. C3 native runtime은 기존
C2 RO lifetime owner와 M8/D-015의 exact direct-root provenance를 별도로 연결한다.
기존 acquired observer의 ARTIFACT import profile을 trusted pipx CLI로 재명명하지 않는다.
고정 pipx1.18.4 UV backend는 child UV를 실행하고 installed Python metadata query는 -S가
없다. 이를 parser 성공으로 admission하거나 startup surface를 숨기지 않는다.

C3a 뒤에 pipx metadata-only original input/고정 실행 argv·환경과 native profile의 세부
수정 지도를 먼저 확정한다. Native fresh/upgrade whole selection→독립 source/wheel graph→
기존 검사·Policy→동일 bytes의 guarded publication→bootstrap/CLI 순서로 작은 단위를
이어간다. 원본 pip_args/Host PATH/interpreter forwarding, runtime/backend 자동 탐지,
오류 기반 공급 보강, helper별 CI, P6 run placeholder는 추가하지 않는다.


### C3b — metadata-only pipx upgrade input의 source 전 세부 지도

`UVToolUpgradeInput`은 UV의 positional constraint와 원본 options를 보존하는 기존 구현이다.
Pipx parser/UpgradeRequirement는 이미 별도로 구현되어 있어 UV input에 pipx를 위장하지
않는다. Pipx의 원본 receipt·complete inert METADATA·pinned Python을 한 bounded control로
묶는 consumer는 아직 없다(IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=NO).

`pipx_tool_upgrade_input.go`에 원본 receipt와 기존 `UpgradeInput`을 합성한다. Strict pipx
parser로 다시 읽고 cached claims 불일치·Python mismatch·root 누락/중복/version mismatch를
거부한다. 원본 requirement는 원본 metadata에 남기고 새 request는 기존
`ToolReceipt.UpgradeRequirement()`의 unconstrained canonical project만 유도한다. Caller가
new constraint를 주입하거나 원본 backend·source interpreter를 실행 recipe로 사용할 수 없다.

Stored pip_args는 opaque parser 데이터다. 이 새 실행 입력 경계에서는 empty/native default와
fixed public index/wheel-only/no-config/no-cache/no-download/no-sources/copy의 명시 option만
받으며 duplicate·unknown·다른 index·local find-links·offline/no-index·source build override를
거부한다. 원본 recipe를 argv로 반환하지 않고 원본 bytes만 보존한다. 추후 resolver는
controller-owned 새 fixed argv를 생성한다. 이 제한은 parser의 opaque 저장 계약이나 이미
지원하는 UV 의미를 바꾸지 않는다.

`UVToolUpgradeInput.ReplayWheelControls/ReplayWheelsControl`의 기존 검증 본문을 두 실제
소비자가 쓰는 private helper로 추출한다. UV wrapper와 canonical control name/body는
그대로 유지하고 pipx wrapper만 새 고정 control name을 사용한다. Complete metadata/hash,
WHEEL identity/tag/runtime/중복/합계 bytes/defensive copy 검사를 복제하거나 완화하지 않는다.
Original receipt·metadata claim은 authentic old archive/ownership/current approval가 아니며
installed code/startup/interpreter/원본 RECORD를 replay input에 포함하지 않는다.

검증은 canonical Quick와 전체 PyPI Race, UV 기존 input/replay 회귀, pipx old constraint와
새 unconstrained request·version/backend/recipe/혼합 control·encoded combined bound·수정 후
defensive copy 및 complete replay negatives다. Runtime/query admission과 public CLI는 여전히
MISSING이며 이 inert 입력 성공으로 acceptance를 닫지 않는다.


### C3a named local 결과 — 2026-10-28

[C3a receipt](./p4o4c3a-receipt-layout-local-20261028.json)의 Quick40.768s/전체 PyPI Race9.298s/
docs/security가 PASS다. Actual installed/upgraded0.12 fixture의 원본 semantics/opaque recipe와
생략·null, source interpreter/환경 interpreter 분리·별도 link inventory·wrong client와 changed
claim/unsafe roots 거부를 검증했다. 첫 Quick268.696s의 fixture []string/JSON []any 비교 오류는
원본을 보존하고 test 타입만 교정했다. Filesystem 실행·ownership·native/CLI 성공은 아니다.
다음은 위 C3b의 bounded original input과 기존 replay 검증 재사용이며 전체 acceptance는 열린다.


### C3b named local 결과 — 2026-10-28

[C3b receipt](./p4o4c3b-original-upgrade-input-local-20261028.json)의 Quick248.212s/전체 PyPI
Race9.898s/docs/security가 PASS다. 원본 requirement/version과 bare-project upgrade request,
정확한 main METADATA version·pinned Python, opaque recipe의 새 실행 입력 제한과 mixed/
noncanonical/control·encoded bound를 검증했다. Shared replay 본문은 명시 parameter 이름 외
기존 bytes와 같으며 전체 UV 회귀와 pipx complete census/hash/tag/runtime/copy가 PASS다.
Stored args/interpreter를 실행하거나 native/public CLI·ownership/Policy가 완료된 것은 아니다.

### C3c1 — native metadata 조회 전 whole wheel gate의 source 전 대조

| 기존 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| `ToolReceipt.BindWheels`의 pipx whole closure startup gate | YES | YES (receipt 이후) | YES (기존 L2) | YES (그 binder 범위) | native metadata query 이전의 실제 소비자 |
| `UVLock.BindWheels`, `bindUVSelectedWheels`, `inspectUVArchiveCandidates`의 original hash/RECORD·aggregate/source/constraints/graph | YES | YES (UV) | YES (기존 scopes) | YES (UV scopes) | 같은 authentic original input에 pipx pre-query gate 결합 |
| `BindPipxSupplyClosure/BindPipxRuntimeSupply`의 installed runtime 공급 startup gate | YES | YES (고정 supply) | YES (O2/B1d) | YES (supply scope) | arbitrary user tool 소비자가 아님; supply root/project/namespace 제약 유지 |
| Pipx user-tool pre-query whole graph entry | NO | NO | NO | NO | receipt를 조작/합성하지 않고 원본 whole wheel을 검증하는 좁은 entry |

C3c1은 `UVLock.BindPipxQueryWheels`에서 기존 native UV lock의 실제 original selected wheels,
independent source pages, pinned target/limits를 검증하고 `validatePipxMetadataSurface`를 전체
closure에 적용한다. 기존 receipt-after-selection binder를 가짜 receipt로 우회하지 않는다.
이미 구현된 `bindUVSelectedWheels`는 private collector helper에 위임하고 기존 UV wheel/
project caller는 nil collector로 같은 source/hash/expansion/empty/project 의미를 유지한다.
새 entry는 nonempty single root를 요구하고 authenticated Requires-Dist graph만 반환한다.

Whole archive hash/RECORD/filename/ABI·metadata/source URL/hash/yanked/Requires-Python,
missing/extra/duplicate wheel/pages·aggregate/closure 제약과 root/dependency의 `.pth` 및
customization surface 거부를 검사한다. 기존 ordinary UV의 supported surface를 pipx 제한으로
넓혀 막지 않는다. 선택 자료 검증은 safety/current Policy·runtime admission이 아니며 실제
native query 전 호출·installed RO supply/native program/profile/CLI는 후속 MISSING이다.

Native 실행의 후속 지도에서는 public acquisition과 Artifact-capable query를 분리한다.
고정 UV backend의 native resolution을 재사용해 immutable wheel 입력을 먼저 동결하고,
pipx는 network-none에서 해당 original wheel들만 소비하는 경로를 검토한다. 실제 pipx
frontend/receipt/census와 원본 constraints/upgrade semantics의 대조 없이 UV tool alias나
새 client 지원으로 선언하지 않는다. Backend child에 CONTROL/network 권한을 상속하거나
existing D-015/zero-authority 계약을 바꾸는 선택은 하지 않는다. 실행 argv/환경과 실제
선택의 source 전 지도·qualification은 C3c1과 별도로 필요하다.


C3c1 source 착수 기준은 `2f350163238a4dd41bb69bc0eb754a3bd90f80d9`,
tree `1aa93a1c2bb6b5fe0c8f55d5ea360a8e3f55a163`/2819파일이다. 추가 대조한 기존 supply
binder도 같은 startup gate를 이미 쓰지만 고정 pipx/pytest 공급 root와 namespace·실행 surface
제약을 소유한다. 이 owner를 arbitrary tool resolver로 넓히지 않고 shared wheel/source/graph
검증에만 새 pre-query entry를 연결한다. C3b 이후 source 교체나 별도 resolver engine은 없다.


C3c1 명시 actual corpus는 기존42개 frozen input의 원본 userpath/click 두 wheel과 각 exact
release metadata다. Canonical input hash42개를 먼저 대조하고 native-shaped lock은 test 내부의
parser-binding fixture임을 명시한다. 새 native solver/query 성공으로 쓰지 않는다. 전체 원본
archive/source/RECORD·dependency graph와 changed archive 거부를 검사하며 supply를
다운로드/보강하거나 artifact/Host interpreter를 실행하지 않는다. 고정 runtime supply와
일반 user-tool graph는 서로 다른 owner 범위로 유지한다.


### C3c2 — fixed native pipx program/argv의 source 전 세부 지도

| 기존 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| O3/B2 isolated `-I -S -B` ZIP loader/stdlib-first original namespace | YES | YES (developer/acquired observation) | YES (그 named scopes) | YES (그 scopes) | native CLI의 closed operation/argv·fixed offline 환경 |
| C2 DATA seed/RO anchor/seal/readers·UV protected supply/runtime volume | YES | YES (각 기존 consumer) | YES (named local/UV scopes) | YES (그 scopes) | native 실행 consumer 결합; DATA pin을 실행 권한으로 쓰지 않음 |
| `ParsePipxToolReceipt`, C3a projection/C3b original input | YES | YES (inert) | YES (named local) | NO (native consumer) | 실제 native metadata/layout와 대조 |
| Native fixed CLI program·offline fixed roots/argv | NO | NO | NO | NO | exact fixed program과 typed install/upgrade/version, explicit guest 환경 |

첫 source 범위는 `internal/sandbox/pipx_native_command.go`의 fixed program과 private
closed-operation argv/환경 builder다. Public CLI나 CONTROL admission/profile은 여기서
추가하지 않는다. Python path/flags·stdlib-only startup·original ZIP namespace를 고정하고
원본 pipx.__main__.cli를 호출한다. Requirement는 기존 public canonical parser의 단일
spec이며 upgrade는 bare project만 받는다. 임의 argv/path/program/backend/Host PATH 또는
원본 receipt pip_args·source_interpreter를 받지 않는다.

Install은 native `install <spec> --backend uv --python /usr/local/bin/python`, upgrade는
native `upgrade <project> --backend uv`에 controller-owned nonempty `--pip-args`만 준다.
Selected pipx 원본 `commands/upgrade.py:443`은 nonempty 새 pip_args가 저장 recipe보다
우선하며578은 원래 specifier를 제거한다. Fixed flags는 no-build/no-sources/no-config/
no-cache/no-python-downloads/no-managed-python/copy/public index이고 Python flag는 pipx에만
준다. Backend는 venv Python을 별도로 지정하므로 기존 UV common argv의 --python을
중복 전달하지 않는다. 원본 immutable source를 수정/monkeypatch하지 않는다.

환경은 고정 guest PATH의 protected `/haa-runtime/uv`, pinned Python, private `/haa-site/pipx`
HOME와 tool-bin/man/completion roots, bytecode off 및 explicit offline/no-index/frozen wheel
find-links를 사용한다. UV venv 단계에도 no-config/no-download가 적용되는지 exact selected
UV의 실제 동작으로 확인한다. Public network 획득은 이 native phase 밖에서 종료한다.
Source interpreter는 고정 guest Python이며 Artifact-capable query는 별도 admission 대상이다.

검증: canonical Quick/전체 sandbox Race·tag vet/docs/security와 operation/canonical source/
constraint·고정 argv/env의 negative를 먼저 실행한다. 명시 runc/network-none/read-only/
cap-drop/ordinary1000 guest에서 selected ZIP·fixed UV·기존 frozen userpath/click 원본의
version/native install/no-op upgrade와 strict0.12 metadata·separate link inventory를 대조한다.
C2 owner의 writer→RO anchor→writer 제거→seal/readers와 exact owned cleanup을 재사용한다.
검사 대상 명령 entrypoint는 실행하지 않는다. 이 runc diagnostic은 actual program/argv·
offline native 의미 범위이며 gVisor/external observation·installed current 공급·current
Policy/Host publication 또는 whole support qualification이 아니다. Changed-version/retired
commands, public native frozen selection과 runtime/profile·guarded promotion/CLI는 남는다.


### C3c1 named local 결과 — 2026-10-28

[C3c1 receipt](./p4o4c3c1-prequery-wheel-gate-local-20261028.json)의 Quick251.469s/전체 PyPI
Race9.934s/tag vet/docs/security가 PASS다. Whole original/source/graph와 pre-query startup
거부를 결합했고 기존 UV·fixed supply owner 의미를 유지했다. 명시 original userpath/click
corpus2.640s는 canonical42 input hash/readback·두 authenticated archive graph와 changed
archive 거부를 확인했다. Native-shaped lock은 test 내부 parser-binding fixture이며 실제
solver/query나 current Policy/Host 반입 결과가 아니다. 다음은 위 C3c2 fixed native program/
argv·offline actual diagnostic이며 native current/hosted/final acceptance는 계속 열린다.


C3c2 source 착수 기준은 `a7e7dd8e599d3784ba143dff43ac9197b1ea9b26`, tree
`dc49dbbafe6778f769f8238464a0318ee1f97f8d`/2823파일이다. Runc fixture의 임시 Python
container lifetime/name/label·AOU cleanup은 기존 native UV container owner를 재사용할 수
있으며 이것은 실행 class/profile·CONTROL/network admission이 아니다. Selected ZIP은 C2
별도 RO DATA owner를 유지하고 UV upstream supply는 기존 never-started pinned OCI reader로
읽는다. Host current pipx 설치나 caller path를 가장하지 않는다. 실제 pre-query corpus는
동일한 userpath/click 원본이며 새 입력·dependency 보강은 없다.


### C3c2 첫 actual — fixture 소유권 반증과 수정 전 대조

첫 Quick226.744s/sandbox 전체 Race27.155s/tag vet·build/docs/security는 PASS였으나 runc
actual5.642s는 UV introduce readback에서 실패했다. Pipx program/설치는 미도달이며 Host5/
Docker/42input/ZIP·source는 동일하고 owned 자원은 정리됐다. First actual/source를 보존했다.
별도 DATA4B-only6조건은 Docker stdin cp --archive가 명시 Config.User=1000이면 tar UID0도
1000으로, 명시0이면 tar UID1000도0으로 바꾸고, Config.User가 비어 있어야 원본0/1000을
각각 보존함을 확인했다. 두 probe는 Python/artifact를 실행하지 않고 Host/Docker가 같다.

Fixture native reader는 qualified UV의 implicit root Config.User와 canonical boundary bootstrap을
누락했다. Source 수정은 test fixture에 한정해 --user를 생략하고 기존 boundaryContainerCommand의
짧은 bootstrap·즉시 UID1000/empty groups/zero caps/NNP 하강을 재사용한다. 모든 native exec도
같은 고정 helper의 demotion을 통과시키며 readiness script로 상태를 확인한다. Runc test의
controller nonce는 helper framing뿐이며 observer admission/completion authority가 아니다.
Root PID1 sleep·명시 Config.User=0으로 대체하면 원본1000 control까지 root로 바뀌므로 그
방법은 사용하지 않는다. Product transport의 root owner/mode/hash guard와 C2 six source는
유지한다. 수정 뒤 새 결과는 아직 없고 이 방법을 해결 검증으로 기록하지 않는다.


C3c2 v2 Quick33.773s/tag vet/build/docs/security와 runc actual9.753s는 PASS다. 같은
product program/argv SHA에서 canonical PID1/exec readiness·native version/install/no-op,
strict0.12 receipt·별도 링크·원본 graph2·실제 metadata/WHEEL replay를 확인했다. Source와
Host5/Docker/42input/ZIP은 동일하다. 여섯 DATA-only 조건과 함께 fixture 소유권 교정의
한정 효과를 검증했다. 추가로 private guest HOME의 malformed uv.toml이 UV_NO_CONFIG 없는
고정 UV venv에서는 실제 거부되고, 같은 파일이 있는 native pipx venv는 고정 환경으로
성공하는지 negative/positive를 연결한다. Host config 또는 일반 package code를 실행하지
않으며 실제 native/gVisor/profile·Host current/Policy 완료로 확대하지 않는다.


C3c2 v3 Quick33.712s/tag vet/build/docs/security는 PASS였으나 actual5.594s는 fixture의
Docker cp→runtime /tmp 준비에서 exit1로 중단됐다. Source RunInput은 stderr를 보존하지
않으므로 daemon의 상세 원인은 미확정이다. Config parser/native install은 미도달이고
readiness와 Host5/Docker/42input/ZIP·source 불변/정리는 확인했다. Bootstrap 교정의
반증이나 product no-config 실패로 쓰지 않는다. Config 준비만 demoted guest의 고정
stdlib-only Python 데이터 쓰기로 바꾸고, 실제 parser negative 조건은 유지한다.


### C3c2 named local 결과 — 2026-10-28

[C3c2 receipt](./p4o4c3c2-native-program-offline-local-20261028.json)의 최종 Quick33.889s/
tag vet·actual build/docs/security와 actual9.800s는 PASS다. 동일 product/default test SHA의
첫 전체 sandbox Race27.155s를 보존했고 최종 repeat는 cached다. Runc ordinary1000의
version/install/no-op, canonical PID1/모든 exec demotion·strict0.12 receipt/별도 link/
원본 graph2·실제 metadata/WHEEL replay와 projection을 확인했다. Private malformed
uv.toml은 no-config가 없는 UV venv에서 실제 parser 거부되고 같은 파일이 있는 pipx
venv는 고정 환경으로 성공했다. NoSKIP·Host5/Docker/42input/ZIP·source/binary 불변과
owned 자원 정리다. 최초 소유권/두 번째 준비 copy 실패와 DATA-only6조건 원본은 보존했다.

구현은 closed native program/argv/fixed 환경까지다. Native current 설치 공급이나 gVisor
관찰/Policy/Host publication·public CLI 성공은 아니다. 현재 diagnostic wheelhouse는 private
RW output 아래에 있으므로 생산 경로의 immutable RO input으로 승계하지 않는다. 다음은
C3c3 installed supply·별도 RO wheel input/native session/profile의 baseline audit다. 고정
ZIP C2 owner와 기존 UV native/whole archive/current Policy/transaction을 재사용하며 새
CONTROL/network 상속·generic resolver 재작성은 하지 않는다. Whole acceptance는 열린다.
