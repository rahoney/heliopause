# P4-E native add와 실제 version-changing selection

Baseline ee89471ca147ea5d933d3f6a463c25a89a356167/tree1f65ed4ae0b8d8f755bac4f35874bdd6900c635b/2500files.
Read: P4-A/D audit와 qualified native selection, UVProjectInput/UVProjectResolver/
UVProjectExport·whole archive binder, client 설계§1/§2/§4와 확장 계획§5. 현재 단일 item은 M13-004다.
[UV0.13.4 dependency 문서](https://github.com/astral-sh/uv/blob/0.13.4/docs/concepts/projects/dependencies.md)와
보존한 exact native help를 대조했다. Add는 manifest/lock을 바꾸며 no-sync는 환경 동기화를
생략한다. 새 bare dependency에는 native가 선택한 compatible version의 하한을 기록하고,
명시 constraint 변경과 강제 upgrade는 다르다. Host 작업 대신 private native 결과를 검증한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| fixed project session/export/census/independent source binding | YES(P4-D) | YES(typed port) | YES(empty/single+legacy UV) | YES(named scope) | 새 add branch·실제 version change |
| add frozen intent | YES(P4-A) | NO(native preflight refuses) | YES(input only) | NO(native) | manifest mutation conservation·native producer |
| manifest mutation guard | NO | NO | NO | NO | only selected requirement changes; metadata/other requirements conserved |
| version-changing lock-upgrade | YES(command) | YES(P4-D) | NO(delta; only same version qualified) | NO(delta) | old selected version→new version actual independent binding |
| Host project/venv transaction·CLI·persistent tools | NO | NO | NO | NO | later phases; 이번 source로 완료하지 않음 |

## Exact map·두 번째 검토

- `artifact/pypi/uv_project_input.go`: SelectManifest/ValidateAddSelection을 좁게 추가한다.
  기존 strict input parser/controls는 재구현하지 않는다. Sync/lock-upgrade는 원본 manifest
  bytes 그대로만 허용한다. Add는 name/version/Python/tool configuration의 typed metadata가
  같고, 선택한 이름 외 모든 직접 requirement가 같으며 정확한 하나의 추가/변경만 허용한다.
  명시 constraint는 canonical clause 순서 차이 외 같아야 한다. 새 bare의 native 하한은
  실제 selected version과 일치해야 하고, 기존 bare 요청은 원래 constraint를 보존한다.
  이 검증은 native 선택 의미만 확인하며 source/Policy authority를 만들지 않는다.
- `sandbox/uv_project_resolver.go`: 기존 여섯 control/session/guard/cleanup body를 재사용한다.
  Add preflight를 연결하고 고정 add <typed selector> --no-sync argv만 추가한다. Native가 쓴
  manifest를 위 guard로 확인한 뒤 새 typed manifest로 native lock/export를 결합한다.
  Export/census 전후 동일한 selected bytes를 재확인한다. Native add의 환경 자동 sync는
  생략하고 기존 private wheel-only materialization·독립 archive binder를 그대로 사용한다.
- 기존 unit test의 add blanket-refusal은 잘못된 add/unknown intent 거부로 갱신한다.
  Metadata 변경/다른 requirement 삭제·완화·추가/source injection/잘못된 native bare bound와
  selected-version 불일치 negatives를 추가한다. Command 순서·namespace·permissions는 유지한다.
- 실제 consumer는 native add exact/bare·기존 dependency 변경·source-proven complete set과
  version-changing upgrade를 명확하게 구분한다. Upgrade의 초기 constrained seed를 편집해야
  하면 그 입력을 synthetic native-readable로 기록하고, native sync가 다시 생성·독립 검증한
  원본 selection에서 실제 upgrade한다. 이를 native Host adoption이나 기존 ALLOW로 쓰지 않는다.

Canonical format/Quick/security/docs·artifact/PyPI+sandbox full Race를 먼저 적용하고 같은
source binary/canonical121 observer로 실제 검증한다. Existing P4-D sync/lock-upgrade 소비자는
새 분기의 영향 범위로 같이 확인한다. 변화 없는 기존 전체 ecosystem/CUDA full CI는 반복하지
않으며 기존 group 규칙을 따른다. CI413은 실제 timer/full source/log qualification을 마쳤고
이후 source에 전용하지 않는다. Host atomic transaction/CLI/tools·whole P4는 계속 OPEN이다.

## Named local 결과

[Add/delta 영수증](./p4e-uv-project-add-local-20261026.json)의 Quick222.363s/security14.467s/
docs0.282s와 affected fullRace26.492s가 PASS다. Same canonical121/최종 바이너리 actual121.396s는
add/delta86.48s와 기존 sync32.31s의11 native selection requests/noSKIP/helperwait0/Host 동일을
확인했다. Exact24.2→25.0·bare existing 보존·새 bare idna3.20/두 직접 roots와 native sync가
재검증한 old24.2에서 실제 lock-upgrade25.0/manifest 불변을 구분한다. 최초 range seed만
synthetic이며 native Host adoption/ALLOW로 확대하지 않는다. 기존11 owner는 bytes 불변이다.
Host atomic transaction/cache/records/CLI/tools와 whole P4는 미완료이며 다음 audit로 이어간다.
