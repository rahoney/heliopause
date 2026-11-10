# P4-D isolated UV project selection consumer

Baseline dc777a1 (전체 SHA는 Git receipt), treec8d25cbbb65d0eb034ffcc6cd3db4cfead5ced0f/2493files.
Read: P4-A/B/C audit/results, client 설계§1/§2/§4, Architecture sandbox/Domain operation,
lessons L03/L08/L14/L16, UVResolver.withUVNativeSession·native output tar와 UV sync independent
archive binder·ProjectDependencyUpdate constructor를 확인했다. M13-004 하나만 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| frozen input/native graph/export/independent archive adapter | YES(P4-A/B/C) | YES(artifact) | YES(named local) | YES(named boundary) | isolated native producer |
| fixed tool/native session/typed network/helper/trace/cleanup | YES(P1) | YES(uv pip/sync) | YES(P1) | YES(existing scope) | project command consumer only |
| native project sync·lock-upgrade | NO | NO | NO | NO | closed argv/frozen controls/native result/target census/independent source |
| project add/Host transaction/CLI/persistent tools | NO | NO | NO | NO | later native manifest mutation/atomic ownership/CLI/tools; 이번에 placeholder 없음 |

## 첫 native 범위와 exact map·두 번째 검토

첫 consumer는 sync selection과 lock-upgrade selection이다. Add는 manifest 변경의 실제
검증 뒤 연결하므로 지금은 composition/network 이전 거부한다. Lock-only의 private wheel
materialization은 선택 증거일 뿐 Host 설치 완료가 아니다. User project/target Python/UV를
Host에서 실행하거나 native argv를 전달하는 fallback은 없다.

- 새 `internal/sandbox/uv_project_resolver.go`: 기존 UVResolver를 감싼 좁은 port adapter.
  NewUVProjectOriginalControls는 manifest/native lock/intent/current runtime와 아직 없는
  actual export/derived view를 같은 여섯 control namespace에 고정한다. Original native
  lock이 있으면 typed reader로 native/source/manifest 형식을 먼저 확인한다. 해당 control
  presence와 current runtime/intent가 틀리면 probe/network/tool launch 전에 거부한다.
- `withUVNativeSession`의 공급·private volume·authenticated typed network/observer/cleanup
  body는 bytes 그대로 재사용한다. 고정 `/haa-site/project`에 inert inputs를 소개하고
  정확한 project lock command(sync 기본/명시 upgrade)→원본 manifest bytes/typed native
  lock 읽기→frozen export→원본 unchanged 재확인을 수행한다.
- `uv_project_output.go`: exact named project controls를 regular/bounded tar로 읽고 introduction/
  freeze를 담당한다. 검증된 selected manifest/native/export를0400로, 별도 derived view를
  기존 pylock.toml 위치에 고정한다. 기존 pip sync/copy와 실제 installed wheel census를
  사용하고 expected project controls의 bytes/mode를 최종 census에서 재확인한다.
- 기존 `uv_native_output.go`: 기존 named pylock tar body를 동일 동작의 named-control helper로
  좁게 추출한다. 허용 경로는 세 exact project files만 추가한다. Existing scanner는 frozen
  expected map의 세 project controls/해당 directory만 추가하며 기존 original input·P1 nil
  경로/UID/GID/mode/link/budget/drain·missing checks는 보존한다. Complete scanner의 expected
  input variant는 기존 wrapper에서 nil로 위임해 기존 동작을 유지한다.
- Native census→UVProjectExport.SelectInstalledWheelControls→existing withUVArchiveInputs/
  independent whole archive/metadata→UVProjectExport.BindProject→기존 typed update를 연결한다.
  Cleanup/trace 실패는 native result를 버리는 기존 session owner를 유지한다. 별도 Policy나
  승인 cache/Host mutation을 만들지 않는다. Original/selected control 개수/namespace 일치와
  actual native/export/view 구분을 기존 domain constructor로 검증한다.

Unit/negative는 preflight0, exact args, named tar alias/owner/mode/truncation/producer error,
control readback·frozen census tamper와 기존 parser/sandbox full Race를 적용한다. Canonical
format/Quick/security/docs 뒤 동일 compiled binary/canonical121 observer로 actual empty/
public single·lock-upgrade 및 unchanged UV pip actual 소비자를 확인한다. Full Host/helper
snapshot와 Docker compiler/다른 namespace actual은 겹치지 않는다. 새 actual denial이 나오면
exact 최초 원본을 확인하고 기존 profile/role/한도를 편의로 넓히지 않는다. CPU·세 CUDA 최종
full은 앞서 대조한 묶음 규칙을 따르며 CI413 timer를 새 push/dispatch로 취소하지 않는다.

## 실제 첫 native 인수 거부

V1은 native lock/manifest 검증 뒤 export에서 UV0.13.4의 `--frozen`/`--no-sources`
상호 배타 규칙으로 exit2였다. First-export-arguments assessment/source와 원본 actual을
보존했다. 공통 lock/export 인수를 분리해 no-sources는 lock producer에 유지하고 frozen
export에서는 제외한다. Export 전 typed public native/source 검증·전후 원본 bytes 대조는
유지하며 observer/권한/한도는 바꾸지 않는다. V1 helperwait0/Host 동일이며 교정 actual은 대기다.

V2는 인수 충돌이 없어졌고 다음 filename 규칙에서 exit2였다. Native export는 pylock.toml
또는 pylock.<name>.toml을 요구한다. 기존 opaque provenance control 이름은 유지하고 guest
파일만 고정 project/pylock.toml로 매핑한다. Derived view는 별도 /haa-site/pylock.toml이다.
Readback/세 frozen project 파일/census가 같은 물리 mapping을 사용하고 P1 scope는 보존한다.
Second-export-filename assessment/source와 v2 raw를 보존했다. Observer/권한은 불변이다.

V3의 empty sync/lock-upgrade는 실제 PASS다. Public packaging25.0은 native reader의
source spelling에서 거부됐으며 v5의 bounded test-only readout이 기존 고정 public index의
https://pypi.org/simple/을 확인했다. B/C reader는 structural fixture의 no-slash만 허용했다.
기존 public profile URL과 그 한 번의 no-slash 표기만 허용하고 foreign/query/userinfo/
다중 slash는 거부한다. Artifact native/export 두 consumer가 같은 좁은 predicate를 사용하며
raw controls는 그대로 보존한다. 새 source identity/URL 일반 정규화/observer 권한은 없다.
임시 wrapper/readout은 제거했고 raw source/log/assessment를 남겼다. 교정 actual은 아직 대기다.

## Named local 교정 결과

[Selection 영수증](./p4d-uv-project-resolver-local-20261026.json)은 final Quick112.172s/
security14.172s/docs0.236s와 artifact/PyPI·sandbox fullRace27.681s를 연결한다. 같은 canonical
observer121/최종 바이너리 actual116.337s에서 empty/public single의 sync+lock-upgrade34.61s와
기존 UV pip public/complete consumers79.18s, noSKIP/helperwait0/Host 동일을 확인했다.
세 exact 원인 교정을 검증했고 임시 진단은 제거했다. Version-changing delta·native add/
atomic transaction/CLI/tools·whole P4는 미완료다. CI413 source50fd와 이 P4 source는 다르다.
