# P4-C UV project export의 native 결합과 독립 archive 검증

Baseline70eafe9f50a1193232786f1654a42d5fcd2c53b4/treeb98a36cb223b05a41cb66f853333d1d5654e7441/2486files.
Read: P4-A/B 감사·named 결과, client 설계§2/§4, P1 UVProjectLock.BindProject/
SelectInstalledWheelControls, UVNativeProjectLock.Parse/ValidateTargetExport와 기존 wheel
metadata/독립 Simple binding. 기존 Architecture/Domain/Policy 경계를 유지한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| native1/revision5 graph·strict wheel export 동일성 | YES(P4-B) | YES(artifact adapter) | YES(actual empty/synthetic nonempty) | YES(reader named scope) | 실제 project export의 index/sdist fields |
| 기존 wheel-only pylock/whole archive/metadata binding | YES(P1) | YES(uv pip sync) | YES(P1) | YES(existing scope) | native project provenance와 actual export의 frozen controls |
| project export 전용 adapter | NO | NO | NO | NO | known extra metadata exact native 대조·원본 보존·독립 whole archive 결합 |
| isolated native project resolver/transaction/CLI/tools | NO | NO | NO | NO | 이 adapter 다음 단계; 이번 범위로 완료하지 않음 |

## 실제 형식 조회와 해석

`p4c-export-query-v3`는 고정 UV0.13.4/Python3.14.7로 network-none runc에서 synthetic native
lock을 실제 export하고 native lock --check까지 exit0/2.779s를 확인했다. Retained wheel
메타데이터와 synthetic inert sdist를 사용했으므로 실제 공개 취득/검역 성공이 아니다.
실제 출력은 각 package에 `index`와 `sdist`를 포함한다. P1 strict reader의 거부는 현재
지원 계약이며 전역 완화하거나 native export에서 임의 unknown field를 삭제하지 않는다.

첫 v1/v2 보조 조회는 read-only rootfs/tmpfs에 Docker cp를 시도해 모두 같은 오류였다.
“시작 전이라서”라는 v2 가설은 반증됐다. V3는 같은 고정 컨테이너의 UID1000 Python으로
명시한 inert 두 입력만 작성했고 성공했다. 세 exact-label container는 stopped/제거를 확인했다.
이는 임시 형식 조회 operator의 범위이며 product volume transport 결함으로 확대하지 않는다.
원본 query receipts/logs/input/output가 정확한 입력과 실패를 소유한다.

## Exact edit map·두 번째 검토

- `uv_native_project_lock.go`: 기존 구현은 유지하고 validated manifest bytes/Python와
  inert sdist declarations를 immutable 결과에 함께 보관한다. 새로운 export consumer가
  원본 control의 동일성을 검사하는 데 필요한 provenance만 추가한다.
- 새 `uv_project_export.go`: strict project export wire에 알려진 index/sdist만 명시한다.
  Public index, native package/version/wheel URL/hash/size와 inert sdist identity를 대조한
  뒤 기존 wheel-only reader의 **내부 validation view**를 만든다. Actual export 원본,
  native uv.lock, pyproject 원본과 derived view는 각각 다른 이름의 frozen controls다.
  View를 실제 native export라 주장하지 않는다. 기존 P1 parser와 API는 bytes 불변이다.
- Typed UVProjectExport.BindProject는 이 네 control의 정확한 bytes/presence를 요구하고
  기존 whole archive/independent Simple/constraint·snapshot binder를 호출한다. 원본 export만
  읽거나 artifact 이름을 신뢰해 승인하지 않는다. 실제 installed census 선택도 기존 API를
  재사용한다. 새 generic engine/Policy/permission/config/dependency/CI job을 만들지 않는다.
- 기존 native fixture helper는 같은 한 번 생성한 archive/metadata를 함께 반환하도록 좁게
  추출한다. RECORD/ZIP의 서로 다른 생성 bytes를 같은 입력으로 가정하지 않는다.
- Tests는 actual offline replay 원본 세 파일·index/sdist/manifest/native/view tamper,
  기존 synthetic whole wheel+Simple binding·손상 archive 거부를 구분해 연결한다.
  원본 raw는 편집하지 않는다. Artifact/PyPI full Race와 canonical format/Quick/security/docs를
  적용하며 unchanged native backend/CUDA full CI를 helper마다 반복하지 않는다.

이 경계는 source/selection 인증과 Policy/설치 권한을 구분한다. Actual public native
resolver·project/venv atomic transaction·CLI/persistent tools 및 whole P4는 계속 MISSING이다.
CI413 frozen50fd timer02:20:16KST와 최종 P2/P3 영향 matrix는 유지하며 M13-004 하나로 계속한다.

## Named local 결과

[Export 영수증](./p4c-uv-project-export-local-20261026.json)은 Quick202.258s/security12.677s/
docs0.243s·artifact/PyPI fullRace5.880s와 actual offline export/check2.779s를 연결한다.
원본 fixture 세 파일은 raw bytes와 같다. 네 provenance control 변조·누락과 실제 archive
손상 거부, installed census/독립 whole archive+Simple binder 재사용을 검증했다. 기존8개
owner는 bytes 불변이며 actual public native resolver/transaction/CLI/tools·whole P4는 열려 있다.
