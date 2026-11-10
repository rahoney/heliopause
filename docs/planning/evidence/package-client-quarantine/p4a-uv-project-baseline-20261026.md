# P4-A UV project 시작 감사·수정 지도

Baseline: `50fd3b2964669f27793c30caa0b2b61113df3e00`, tree
`d6a63abdc426da4065fdfa248151ad64323a29dc`, 2475 tracked files.
M13-004만 IN_PROGRESS다. CI413은 이 frozen source의 grouped Bun candidate를 검사한다.
P4 사전 감사와 로컬 구현은 독립적이며 CI413 실행 중 새 push는 하지 않는다.

## Read와 기존 owner

Client 설계§1–4, 확장 계획§3–6, Architecture foundation/dependencies와
inspection/sandbox/Policy, Domain operation/context, trusted tooling/Evidence,
update quarantine의 PyPI selection, canonical quality profiles를 대조했다.
P1 acceptance와 현재 source의 아래 실제 symbol을 기준으로 재사용한다.

| 책임 / source | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 고정 UV0.13.4/Python3.14.7 공급·격리 selection — sandbox UVResolver/prepareUVTool | YES | YES(uv pip) | YES(P1 bounded scope) | YES(P1) | project native command/attestation·실제 소비자 |
| wheel/source 독립 검증·complete pylock — artifact/pypi ParseUVProjectLock, UVSyncInput; sandbox bindUVNativeProjectSelection | YES | YES(uv pip sync) | YES(P1) | YES(P1) | native uv.lock·pyproject와 해당 target export의 결합 |
| complete-set 검사·current Evidence/Policy·venv transaction — application UVPipSyncService, promotion uv_sync* | YES | YES(uv pip sync) | YES(P1) | YES(P1) | project control와 installed environment의 원자적 commit/rollback |
| UV project input/intent/native lock — cli root/python client에는 uv pip만 존재 | NO | NO | NO | NO | frozen pyproject/uv.lock, add/sync/lock-upgrade 의미·guard |
| UV tool/pipx persistent metadata·노출 script ownership | NO | NO | NO | NO | exact backend/supply, metadata, retired scripts·collision/rollback, actual qualification |

`UVProjectLock`은 이미 구현된 **pylock.toml 1.0 complete selection**이다. 이름만 보고
native **uv.lock** reader로 재사용하거나 대체하지 않는다. 기존 parser·binder는 유지하고
native project→target export의 새 경계를 좁게 추가한다. P1 코드는 재구현하지 않는다.

## 공식 의미와 첫 범위

선택한 [UV0.13.4 sync 문서](https://github.com/astral-sh/uv/blob/0.13.4/docs/concepts/projects/sync.md)를
확인했다. Lock-only는 환경 설치가 아니고, sync는 선택에서 빠진 distribution 제거를 포함한다.
기존 lock의 버전을 선호하는 동작과 명시 upgrade를 구분한다. Frozen만으로 pyproject와 lock의
최신성 검사가 되지 않으므로 별도 native check/selection 결합이 필요하다. Native help/빈
project 출력은 `p4a-native-query-20261026` raw에 별도로 보존한다. runc 형식 조회를
production gVisor 관찰·Policy/설치 성공으로 확대하지 않는다.

첫 입력 adapter는 단일 **virtual project**, public PyPI, 고정 Python만 대상으로 한다.
`[project]`의 name/version/requires-python/dependencies와 선택적 `[tool.uv] package=false`를
엄격하게 읽고 원본 bytes를 보존한다. Native command 전에 workspace/build-system/local/
VCS/direct sources·임의 index/config·scripts·groups/extras/markers·알 수 없는 schema를
거부한다. 이 첫 제한은 이후 지원 목표 전체의 완료가 아니다. 의존성 specifier는 기존
ParseUpgradeRequirement의 public bounded 문법을 재사용하며 @ shorthand는 project PEP508
입력에 허용하지 않는다. 명시 dependencies=[]와 필드 부재를 구분한다. Runtime requirement는
현재 Python을 실제로 포함해야 하며 이름 정규화 충돌·root self dependency를 거부한다.

## Exact 수정 지도·두 번째 검토

1. `internal/artifact/pypi/uv_project_input.go`와 의미 있는 negative tests:
   inert frozen project input, typed add/sync/lock-upgrade intent, 원본 pyproject/선택적 native
   uv.lock의 presence·bytes·runtime binding. Parser는 승인·설치·삭제 권한을 만들지 않는다.
   TOML dependency는 이미 고정된 go-toml/v2를 사용하며 새로운 dependency/config는 없다.
2. 위 입력이 native 형식과 대응한 뒤 별도 native uv.lock/target export reader와 resolver
   map을 작성·재검토한다. 기존 ParseUVProjectLock/UVSyncInput은 변경하지 않는다.
3. Native resolver 뒤 existing complete-set inspection/cache/transaction을 연결할 때마다
   해당 owner audit·exact map을 갱신한다. Lock-only 결과를 설치 COMPLETED로 반환하지
   않는다. UV tool/pipx는 별도 같은 P4의 후속 substep이며 이번에 placeholder를 만들지 않는다.

두 번째 검토: root project는 로컬 artifact/build 대상이 아닌 virtual metadata다. 이를 wheel
선택에 끼워 넣거나 local project 실행을 허용하지 않는다. 원본 control bytes·presence와
intent/runtime은 선택 이후까지 고정하고 Host의 원래 UV 호출로 재resolve하지 않는다.
Native 형식 조회/입력 parser만으로 P4 CLI나 whole acceptance를 닫지 않는다.

## 검증 선택

입력 adapter는 canonical format→Quick/security/docs, 해당 artifact/pypi 전체 Race와
wrong runtime/unknown field/duplicate key/normalized duplicate/source injection/empty presence/
round-trip 및 control tamper negatives를 확인한다. 실제 native format query는 고정 도구·
이미지/command/result/cleanup을 보존한다. Source/observer/runtime/CI는 이번 parser 단계에서
바꾸지 않으므로 기존 실제 모든 client/세 CUDA full CI를 작은 adapter마다 반복하지 않는다.
다음 native consumer 연결과 최종 영향 matrix의 검증은 계속 열려 있다.

## Named local 결과

[입력 경계 영수증](./p4a-uv-project-input-local-20261026.json)은 final Quick43.181s/
security8.911s/docs0.249s와 artifact/PyPI 전체 Race5.379s를 연결한다. 실제 고정 UV의 빈
virtual project 원본 두 파일을 읽고 bytes/presence를 보존하는 회귀도 PASS다. 새 source는
inert parser와 control replay뿐이며 기존 native/Policy/transaction/CI owner는 변경하지 않았다.
P4-A named 입력 경계만 닫았고 native uv.lock/target export·gVisor resolver/CLI·영구 tools와
whole P4는 미완료다. CI413은 frozen50fd Bun source이며 이 P4 성공의 원격 근거가 아니다.
