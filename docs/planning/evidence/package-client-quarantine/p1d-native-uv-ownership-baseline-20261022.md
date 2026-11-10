# P1 native UV ownership — 유한 형식 조사와 착수 감사

기준 d1d4fdc, C3c2b actual CLI CI 대기 중의 독립 data-only 조사다. M13-004 하나만
IN_PROGRESS이며 C3c2b qualification을 건너뛰어 새 지원을 열지 않는다.

기존 `pypi_native_inventory.go`는 RECORD를 원본 claim으로만 freeze하고, exact METADATA/
ledger·alias·경로·device/inode·bounded hash/size를 검사한다. `pypi_native_ownership.go`의
oldWheel은 공식 이전 wheel을 독립 획득/정적 검증하며, proveNativePyPIWheelFootprint는
실제 wheel bytes와 pinned pip marker 및 제한한 timestamp cache removal을 대조한다.
이 기존 bounded pip 소비자는 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED YES다.
Generated console/gui scripts와 cross-scheme native footprint는 명시적으로 거부한다.

Native UV adoption은 네 상태 모두 NO다. MISSING은 UV auxiliary metadata의 실제 형식,
기존 독립 old-wheel proof와의 연결, 선택한 client별 정확한 installer/cache/record 판정과
managed update handoff·변조/rollback·actual native CLI다. `INSTALLER=uv` 문자열만 추가하거나
RECORD를 소유권으로 승격해서는 안 된다. 새 generic enrollment engine은 만들지 않는다.

## 먼저 확인할 실제 입력

같은 signed/검증된 UV0.13.4와 고정 Python3.14.7 image를 사용한다. 기존 format probe의
read-only/unprivileged/network-none runc·no-cap·NPP 경계를 재사용하며 Host/user venv나
artifact module을 실행하지 않는다. Inert wheel은 실행 시 즉시 실패하는 자체 payload다.

1. 일반 UV venv에 자체 pure wheel을 offline 설치하고 INSTALLER/REQUESTED/RECORD와
   uv_cache.json의 전체 bounded bytes·hash·모드를 수집한다. RECORD 밖 파일도 별도 census다.
2. 같은 wheel을 --target에 설치한 layout과 venv layout을 비교한다. Metadata가 같다는
   이유로 경로/상대 깊이·실제 소유권을 동일하다고 가정하지 않는다.
3. Console entry point가 있는 자체 wheel의 생성 wrapper/RECORD 경로를 데이터로만 확인한다.
   실행은 하지 않으며, 관측만으로 기존 native generated-script 거부를 해제하지 않는다.
4. Default와 명시 compile-bytecode의 file/cache 관계를 구분한다. Cache는 실행·승인하지
   않으며 pinned original source/runtime과 결합되지 않은 임의 .pyc를 소유권으로 받아들이지 않는다.

첫 오류에서 큰 후속 입력은 중단하고 exact argv/exit와 원본을 보존한다. 기존 native pip의
marker·runtime·ownership 검사를 완화하지 않는다. Fixture 성공은 public/gVisor/Policy/
actual adoption 성공이 아니다. 결과 뒤 구현 필요 위치와 negative matrix를 다시 구체화한다.

## 정리와 두 번째 검토

Exact owned container/image만 식별·제거하고 image rm은 --no-prune를 사용한다. Host 파일/
공유 image/container/bridge/service를 전후 대조한다. Root 참조·query errors를 확인한 뒤
완료된 전용 workroot/operator는 작은 원본·재현 source 보존 후 삭제한다. SDK/active bundle과
사용자 preview는 이 조사의 정리 대상이 아니다.

두 번째 검토: Installed JSON과 RECORD는 artifact 또는 사용자 입력이다. Auxiliary filename/
valid JSON 자체는 authority가 아니며 authentic old wheel과 정확한 생성 역할·원본 snapshot을
검증해야 한다. 구조가 미확정인 file/entrypoint/cache는 기존 refusal을 유지한다. 현재 CLI의
--adopt를 미리 추가하거나 native output을 HAA 소유 ledger로 곧바로 복사하지 않는다.


## 후속 입력 구분 — local wheel과 registry wheel

첫11개 data-only 입력에서 INSTALLER는 newline 없는 uv 두 bytes, REQUESTED는 empty,
local find-links wheel의 uv_cache.json에는 timestamp와 null commit/tags, 빈 env/directories가
있었다. Console wrapper는 bin 상대 경로로 RECORD에 포함됐고, 명시 compile-bytecode의
pyc는 RECORD에 없었다. 이는 local fixture 형식 관측이며 public registry 형식으로 전용하지 않는다.

고정 tag의 [installer 호출부](https://github.com/astral-sh/uv/blob/0.13.4/crates/uv-installer/src/installer.rs)는
비어 있지 않은 cache info만 전달하고, [wheel metadata writer](https://github.com/astral-sh/uv/blob/0.13.4/crates/uv-install-wheel/src/wheel.rs)는
전달된 cache에 한해 JSON을 생성한다. 따라서 cache 파일의 필수 존재를 가정할 수 없다.
추가 probe는 network-none guest loopback에 자체 wheel/index만 제공해 registry 형태의
install/census만 확인한다. Host/public network나 credential은 사용하지 않고 기존11개 입력은
반복하지 않는다. 이 probe 역시 public 공급망 또는 native adoption qualification은 아니다.


## 형식 조사 결과와 다음 검토 경계

[원본·재현·정리 영수증](./p1d-native-uv-format-20261022.json)은 첫11개 명령과 추가3개
loopback registry 명령을 구분한다. Registry-shaped 설치에는 uv_cache.json이 없었다.
Local wheel의 timestamp JSON을 공개 registry의 필수 설치 파일로 일반화하지 않는다.
Default에는 없던 명시 compiled pyc는 RECORD 밖에 있었으므로 RECORD claim만 읽는
기존 inventory로 해당 profile을 지원했다고 주장할 수 없다. Native adoption에서는 인증된
old wheel의 Python source 역할에 대응하는 cache 경로의 존재/부재와 변경도 보호해야 한다.

첫 후보는 public registry wheel의 기존 bounded native pip 수준인 site/dist-info와
정확한 UV marker/empty REQUESTED로 좁힐 수 있다. Unexpected UV cache/build/direct-url,
generated scripts/cross-scheme 및 compiled cache는 명시 거부를 유지한다. 그 제한만으로도
원본 claim의 namespace/bytecode 잔재를 누락하지 않는지 구현 지도에서 검토해야 한다.
이를 native UV 전체 지원으로 표현하지 않는다. 실제 public UV 생성물과 old-wheel proof→
새 검역→target 반입·변조/rollback 확인 뒤에만 지원을 연다.

Owned containers6·파생 images2는 --no-prune로 제거했고 두 Host snapshot이 각각 동일했다.
Root 참조0/미확정 query errors0 뒤 완료 workroot/operator6항목20,410B를 삭제했다.
초기 WSL zombie mountinfo EINVAL과 자체 source-copy wrapper 참조로 인한 보류는 원본에
구분했다. PID 이름만으로 생략하지 않고 실제 Z/동일 start time·빈 FD/maps/cmdline을 확인했다.
작은 원본·재현 source와 active SDK/quality/runtime만 보존했다.


## 구현 착수 수정 지도 / 두 번째 검토

선행 C3c2b는 actual CI380으로 닫혔다. C3d의 네 상태는 아직 NO이며 기존 pip의 bounded
native owner와 guard·old-wheel acquisition/inspection·rollback을 재사용한다. 첫 native UV
범위는 registry wheel의 site/dist-info 및 UV marker/REQUESTED다. Generated scripts,
cross-scheme·direct/local/VCS/build cache와 compiled native cache는 명시 거부한다.

| 정확한 위치 / symbol | 변경과 검증 |
| --- | --- |
| promotion/pypi_native_ownership.go pypiNativeEnrollment/oldWheel/proveNativePyPIWheelFootprint | private 두-client format을 도입하되 zero/default는 기존 pip. Exact lookup runtime과 marker/ownership-role만 client별로 고정하고 기존 authentic wheel/RECORD/alias/file proof 본문 재사용. UV lookup은 실제 UV resolver이므로 중앙 UV identity를 정확히 요구; pip identity로 위장/완화 없음 |
| 같은 owner selectOwnership와 shared pypi_update.go | UV는 인증된 old Python-source 역할의 __pycache__ directory가 없어야 함. Absence paths를 controller selection에 반환하고 기존 transaction.freezeParents/absentDirectories와 Lstat/verify로 mutation 전에 고정. RECORD 밖 cache도 누락하지 않으며 생긴 dir를 삭제·무시하지 않음. Proven files는 모든 검증 뒤 current ownership에 추가 |
| promotion/uv_update.go WithNativeEnrollment/Begin | 기존 immutable native collaborator 복사와 UV profile만 연결. Native source는 public, explicit adopt=true인 UPDATE만 허용. Unmanaged default refusal 유지하고 original guard/Manifest5controls·old proof→new set inspection 순서 보존 |
| cli/package_client.go / bootstrap/python_client.go | --adopt는 기존 두 upgrade 모드에서만 허용하고 descriptor EN/KO 설명에 실제 제한 표시. 두 native updater의 실제 UV lookup/intake를 연결. Ordinary install은 암묵 인계하지 않음 |
| promotion native/session 및 CLI tests | 기존 pip marker/cache·native tests 전체 유지. UV2bytes/emptyREQUESTED·wrong/omitted marker·extra cache/build/source/script·old-byte tamper·exact runtime·explicit adoption·cache absence concurrency/rollback·handoff/replay를 검사 |
| 기존 authenticated bootstrap UV integration + 좁은 test fixture helper | 첫 검역에서 보존한 authentic old wheel을 고정 UV/Python으로 network-none runc의 자체 loopback index에 설치해 실제 UV 원본을 생성. Host artifact/Python은 실행하지 않고 cfg/site를 bounded tar로 복사. Product old-wheel lookup은 다시 실제 public UV/독립 archive 검증을 수행. Native 위조/compiled-cache refusal 불변→adopt update→managed replay를 확인. Exact owned containers는 label/image 확인 후 제거하고 absence query 성공을 요구 |

두 번째 검토: native fixture의 runc 작업은 외부 프로그램이 만든 원본 데이터 생성이며 제품
검역/Policy를 대체하지 않는다. 입력 wheel은 앞 실제 검역의 exact staged bytes이며 marker를
직접 UV로 바꾸거나 원본을 가짜 report로 조립하지 않는다. Native tool은 기존 lock의 exact
image와 native hash/size, Python image를 쓰고 network-none/UID1000/read-only/no-cap/NPP 및
고정 guest paths를 유지한다. Host mount/credential·artifact import·native arbitrary args는 없다.
Fixture code는 실제 integration gate에서만 동작하며 새 CI job/privilege나 제품 observer role/
시간 예산을 추가하지 않는다. 현재 five-step controller/과거 pip qualification을 새 native
adoption 성공으로 전용하지 않는다. Canonical5/전체 관련 Race 뒤 actual source CI와 정리를
확인하고, 첫 actual fault를 보존한다. Whole closure/multi/empty sync는 그 다음 gap이다.

후속 코드 검토는 두 형식 조건을 구체화했다. 고정 tag의
[metadata writer](https://github.com/astral-sh/uv/blob/0.13.4/crates/uv-install-wheel/src/wheel.rs#L859-L906)는
REQUESTED를 requested 인자가 true일 때 만든다. 앞 실제 data-only probe에서는 transitive
dep에도 이 파일이 있었으므로 primary/직접 요청 여부를 이 marker로 추론하지 않는다. INSTALLER는 필수 UV 두 bytes,
REQUESTED는 있으면 정확한 empty/hash/size를 요구하며 부재 자체를 거부하지 않는다.
각 파일의 소유권은 여전히 authentic old wheel proof가 결정한다. 또한 UV의 bytecode census는
필수 import 역할 목록과 다르므로 site/dist-info의 인증된 모든 .py 파일의 __pycache__ 부재를
freeze한다. Resource Python 파일도 포함하며 기존 pip의 제한된 cache 검증을 바꾸지 않는다.
실제 native integration은 첫 새 검역에서 보존한 idna3.10 wheel로 외부 UV 원본을 만들고,
unrecorded-cache 무변경 거부→명시 adopt3.11→managed replay와 cfg/user file 보존을 확인한다.
네 상태는 원격 actual source qualification 전까지 acceptance 미완료다.

## 로컬 후보 검증과 남은 경계

[최종 로컬 영수증](./p1d-native-uv-local-20261022.json)은 source10경로와 canonical5·관련5package Race130.026s 성공을 연결한다. C3d는 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=LOCAL ONLY / ACCEPTANCE_CLOSED=NO다. UV marker·optional REQUESTED·old-wheel 변조/runtime·RECORD 밖 cache/resource cache·publication 전 cache 생성과 rollback/소유권 미게시를 검증했다. Native fixture는 실제 UV 컨테이너 두 개만 만들고 label/image를 확인한 정리와 성공한 부재 조회를 요구한다. 기본 테스트에서는 실제 fixture를 실행하지 않았다. Actual source CI의 native 원본→거부→adopt→managed replay·기존 공용 소비자/macOS/Required가 남으며, 이를 로컬 unit 성공으로 닫지 않는다.

## CI381 최초 실패와 fixture 전달 교정

[Actual CI381](./p1d-native-uv-ci381-20261022.json)은 같은 전체 tree에서8SUCCESS/2FAIL이다. Managed CLI5단계는 다시 통과했지만 fixture의 docker cp 입력이 read-only rootfs에서 거부되어 native UV 실행·인계는 도달하지 않았다. CPU full도 미도달이다. Helper3wait0·원본 RESTORED와 fixture 정리 assertion을 확인했으며 전체 qualification 성공으로 기록하지 않는다.

교정은 제품 guard/Policy를 바꾸지 않는다. 고정 Python의 길이 제한 stdin writer와 stdout tar로 live guest tmpfs의 데이터를 전달하며 rootfs를 read-only로 유지한다. 고정 UV binary 전용 tmpfs만 exec를 명시하고 wheel/HTML은0444, 별도 /tmp는 noexec다. [로컬 전달 영수증](./p1d-native-uv-fixture-local-20261022.json)은 첫 noexec 거부와 다음 inert fixture PASS6.509s·실제 tool/cfg/site 데이터·두 Host snapshot 동일·전용 자원 정리를 구분한다. 실제 공개 old-wheel proof/adopt/replay 성공은 아니며 새 source CI가 남는다.

교정 후보의 canonical format/Quick50.051s/security/docs/workflow와 bootstrap Race8.806s도 통과했다. 제품 promotion/CLI/Application 실행문은 기존 관련 전체 Race130.026s 후보와 같으며, REQUESTED 설명 주석만 정정했다. 임시 transport-only Go stub/operator/client는 작은 재현 원본 보존 후 삭제했고 Go TMPDIR0이다. 새 actual source CI가 다음 완료 조건이며 sync source는 아직 수정하지 않는다.

## CI382 실제 native 인계 성공 / 공용 CI gap

[Actual source0a16a07 결과](./p1d-native-uv-ci382-20261022.json)는 같은 전체 checkout tree의 Linux51·CPU full297.75s와 실제 UV CLI185.82s PASS를 연결한다. 고정 UV가 authentic staged idna3.10 wheel로 실제 원본17항목을 만들었고, RECORD 밖 cache의 target/Evidence 무변경 거부→old-wheel 독립 증명과 명시 adopt3.11/controls5→managed replay/controls4가 통과했다. Required checks/Evidence·cfg/user data 보존·old METADATA 제거와 helper5wait0·final confirmed cleanup도 통과했다. 이 source의 native bounded 기능은 실제 Linux에서 검증됐지만 C3d 전체 ACCEPTANCE_CLOSED는 NO다.

첫 회차는 macOS production build120s timeout으로8SUCCESS/2FAIL이며 기본 테스트는 도달하지 않았다. 직전381 성공 후보와 production 실행문은 같고 원인은 미확정이다. 실패 작업 재실행의 두 번째 회차는 GitHub의 unexpected internal startup error로 job0/check-run0이었다. 이전 macOS job은 현재 회차 소속이 아니어서 재실행 API가 거부했다. 제품 코드·한도·coverage를 바꾸지 않고 이 실제 증거 전달로 새 source CI를 진행한다. macOS/Required가 실제 통과하기 전에는 C3d나 P1 acceptance를 닫지 않는다.

## C3d scoped qualification 종료 / CI383

[Actual CI383](./p1d-native-uv-ci383-20261023.json)은 sourcea547957와 동일 전체 checkout tree에서 Required 포함10SUCCESS다. Native 원본→cache 무변경 거부→authentic old-wheel proof/adopt→managed replay와 managed5단계가145.34s에 통과했고 Linux51/CPU220.35s·helper5wait0/final confirmed cleanup을 확인했다. Native macOS는 동일 production 실행문과 기존2m 한도에서59.458s build 및 default tests PASS다. 앞382 timeout의 정확한 원인은 여전히 미확정이며 source 성능 수리나 budget 확대의 성공으로 기록하지 않는다.

C3d는 정의한 default public wheel/site/dist-info·no-cache/no-generated-script 범위의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED가 YES이며 그 scoped MISSING은 없다. Whole P1은 NO를 유지하며 다음 C3e complete/multi/explicit-empty sync와 실제 dependency closure qualification을 진행한다. 추가 조사 중인 held P1e 문서는 이 CI의 전체 HEAD/새 기능 qualification에 포함하지 않는다.
