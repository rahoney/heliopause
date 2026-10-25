# P4-H native project의 guarded installation 연결

Baseline c1e296d/tree63ba40400711f2d49caf60d2c4d34da2f650684c/2514files.
Read: Queue/plan20와 client 설계09의 명시 target/runtime provisioning 제외,
P4-A..G/각 원본·교정, lessons INDEX/artifact L02/L15/sandbox L08/evidence L16,
Domain InstallContext/ProjectDependencyUpdate/ports와 guarded project workflow,
기존 UVSyncPreparation/publication 및 Python complete cache/installed snapshot owner.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Fixed native project add/sync/version delta와 independent source binding | YES | YES(typed port) | YES(P4-E local) | YES(named selection) | held original guard에 연결 |
| Whole inspection/current Evidence·Policy/native approved cache | YES | YES(actual data-only composition) | YES(P4-F) | YES(named cache) | approved update와 Host publication의 동일성 |
| Project+venv shared commit/journal/rollback | YES | YES(filesystem engine) | YES(P4-G synthetic normal/negative) | YES(named primitive) | actual native approved selection 소비 |
| Original venv snapshot/managed ownership/materializer | YES | YES(UV pip) | YES(P1) | YES(P1 scope) | project request의 held lifecycle에서 재사용 |
| Native project request preparation/publication | NO | NO | NO | NO | single selection/approval/stage/commit guard, actual add/sync·retirement |
| CLI/native adoption/lock-only/persistent tools | NO | NO | NO | NO | 후속 같은 P4 acceptance |

## Exact map·두 번째 검토

Promotion에 한 request의 UVProjectPreparation/held session을 추가한다. 기존 discoverUVPythonVenv,
begin transaction, frozen project files, installed snapshot와 managed inventory를 재사용한다.
현재 첫 installed lane은 기존 fixed-runtime venv만 받으며 unmanaged native distribution은
선택/삭제 전에 거부한다. Native adoption은 이후 별도 proof acceptance다. 새 interpreter나
venv runtime을 provisioning하지 않는다. Empty/managed 소유권을 추정하지 않는다.

기존 typed resolver와 native provenance validator로 exact original six controls/context를
재검증하고 결과를 한 번만 보존한다. Application은 existing executeGuardedProjectUpdate를
사용하는 작은 UVProjectService로 연결한다. 새로운 Policy나 검사 순서를 만들지 않는다.
Stage는 기존 UVProjectCache를 사용하고 commit은 같은 staged set/current Evidence와 held
snapshot을 다시 확인한 뒤 G의 two-control+venv commit을 호출한다. Native 선택을 반복하지 않는다.

기존 UVSync의 approved cache→private materializer body만 공용 helper로 추출해 두 실제
소비자가 재사용한다. 검증/명령/cleanup body를 기계적으로 대조하고 기존 sync 소비자를 함께
검사한다. UV pip input을 project manifest로 해석하거나 의미를 대체하는 경로는 없다.
Cache의 기존 readback을 promotion 내부 material 반환 helper로 연결하고 공개 data API도
그 helper를 사용한다. PublishedRecords는 실제 commit 성공 후만 노출한다.

두 번째 검토에서 single-use lifecycle, context/original/selected/cache 동일성, current approval,
unmanaged source 거부, 기존 user files 보존, empty retirement와 G의 cleanup-before-backup-disposal을
대조했다. Lock-only를 이 installed 경로에 넣지 않는다. CLI·native adoption·lock-only·persistent
tools와 whole P4는 이 단계 완료 뒤에도 별도 missing으로 유지한다.

Canonical profiles/affected tests·Race, adversarial resolver/approval/stage/drift/반복 호출 거부,
같은 fixed UV/canonical121 helper 아래 실제 public add·업데이트·same-version 재검사·sync·empty
retirement/Manifest·SBOM/current Evidence/원상 종료와 기존 UV pip sync 소비자를 확인한다.
이 backend checkpoint는 helper별 push/full CI를 만들지 않는다.

## Actual에서 확인한 original lock 보완 지도

첫 실제116.176s는 add24.2→25.0·same-version sync·두 roots의 전체 검사/승인/반영을 확인했으나, manifest에서 deps를 지운 sync가 native 호출 전 기존 original-lock reader에서 거부됐다. Helperwait0/Host 동일이며 기존 UV pip 회귀는 failfast로 미도달이다. 원본 `docs/github-logs/package-client-quarantine-20261021/p4h-stale-original-lock-first-assessment.json`가 부분 성공과 최초 거부를 소유한다.

Original lock에는 이전 root requirements가 남을 수 있다. 현재 manifest와 selected lock의 일치는 그대로 엄격하게 유지하고, original native lock은 동일 virtual root/Python·공개 schema·이전 완전 graph를 검증하는 별도 error-only reader로 확인한다. 기존 strict reader와 budgets를 재사용하며 이전 requirements로 만든 파생 validation view를 반환하거나 approval/provenance로 사용하지 않는다. Sandbox original preflight만 이 reader로 연결한다. Source/role/예산/unknown schema 허용은 바꾸지 않는다. Stale original 허용/selected stale 거부와 foreign root/source/graph 거부를 추가 검증한 뒤 같은 actual case와 기존 UV pip 소비자를 다시 확인한다.

## Named local 결과

[Installed backend 영수증](./p4h-uv-project-publication-local-20261026.json)은 최종 Quick292.784s/
security15.138s/docs0.249s·focused Race25.544s/full affected Race283.054s와 actual318.313s를
연결한다. Project140.47s의 add24.2→25.0·same-version 재검사·두 roots·edited-empty retirement와
기존 UV pip sync175.04s/여섯 complete 요청/native original17files 인계가 모두 PASS다.
Current whole inspection/Evidence·Policy·Manifest/SBOM과 published controls/installed bytes를 대조했고
noSKIP/helperwait0/Host 전후 동일을 확인했다. 두 최초 실패는 원래 범위로 보존한다.
이전 requirement view는 original validation 뒤 버리고 selected/current strict reader와 독립 archive
binding/권한/한도는 유지했다. 기존11 owner는 bytes 불변이다. Public CLI/native project adoption/
lock-only/tools와 whole P4/remote qualification은 미완료이며 다음 CLI를 계속한다.
