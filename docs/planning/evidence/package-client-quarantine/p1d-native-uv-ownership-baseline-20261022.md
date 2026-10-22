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
