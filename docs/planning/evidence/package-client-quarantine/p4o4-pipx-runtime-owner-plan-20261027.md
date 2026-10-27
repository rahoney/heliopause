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
member 경로는 거부한다. Profile의 direct admission은 `--handoff-python`만 받으며
CONTROL launch와 ELF handoff는 protocol에서 거부한다. 기존 profile 의미는 보존한다.

Go factory의 phase별 `/haa-pipx` topology와 C++ 실제 registration/admission·latch를 함께
검증한다. Host tmpfs noexec와 guest9p flags를 혼동하지 않는다. Quick/관련 Race·pinned
compiler의 전체 observer latches를 먼저 실행하고, 외부 gVisor transaction은 다음 B2c에서
실제로 연결한다. Runtime/profile 추가를 기존 완성 경로의 재구현이나 예산 확대로 사용하지 않는다.

B2b [로컬 receipt](./p4o4b2b-observer-profile-local-20261028.json)는 Quick204.610s/
sandbox Race25.051s/docs/security PASS와 canonical observer build274.583s/전체
124latches PASS를 연결한다. 새 fixed ZIP read와 ARTIFACT-only Python admission의
production protocol 거부/정상 회귀를 포함한다. Helper는 빌드했으며 Host에 설치·활성화하지
않았다. 실제 acquired ZIP의 gVisor transaction·완료 원장·CPU/cleanup은 B2c에 남는다.
