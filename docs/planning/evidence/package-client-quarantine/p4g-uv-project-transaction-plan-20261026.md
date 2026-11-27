# P4-G project controls와 venv의 공용 transaction

Baseline 0c851b829be8632937203f993e07f0f8443023cd/treec16fb954badb68cf4c3acb05c327f67f56924a95/2510files.
Read: Queue/plan20/P4-A..F, lessons INDEX·artifact L02/L15·sandbox L08·evidence L16,
Threat Model fail-closed/promotion, Domain operation context와 Evidence/promotion contract,
existing pypi_venv_transaction/backup/complete tests, uv_sync source/publication와
acquireProjectTransactionGuard의 descriptor/identity owner를 확인했다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Native project complete selection/current approved cache | YES | YES(data-only composition) | YES(P4-F local) | YES(named cache) | held Host transaction으로 전달 |
| Venv complete wheel ownership/retirement/backup/journal/rollback | YES | YES(UV pip) | YES(P1) | YES(P1 scope) | project controls를 같은 journal에 포함 |
| Descriptor-bound project guard | YES | YES(Go/Cargo) | YES(existing scope) | YES(existing scope) | UV project exact lock 이름/파일 경계 연결 |
| Project controls+venv publication | NO | NO | NO | NO | frozen source/presence·same commit/rollback와 failure 회귀 |
| Native ownership/CLI/lock-only/persistent tools | NO | NO | NO | NO | 후속 같은 P4 acceptance |

## Exact map·두 번째 검토

기존 transaction engine을 대체하지 않는다. Native project의 exact pyproject.toml/uv.lock만
freeze하고 기존 project guard·venv file/parent witnesses를 사용한다. Optional original lock
absence와 required manifest presence를 구분하고 source/target/trusted stores의 alias를 거부한다.

작은 private project-control plan을 기존 venv transaction에 연결한다. Wheel ownership state와
project controls는 역할이 다르다. Project controls를 wheel 소유 파일로 기록하지 않고,
같은 backup/publish/rollback 루프와 journal 안에서 별도 project-control scheme로 다룬다.
Journal의 optional project-root/selected-control 경로 정보를 보존한다. 기존 nil 소비자의
installation schemes/ownership state/실행 순서는 유지한다. Project guard를 모든 mutation과
reconciliation에서 검사하고 recovery 불명확 시 두 lock/원본을 보존한다. 원래 없던 lock은
rollback에서 부재로 복원한다. External project parent도 sync한다.

두 번째 검토에서 commit 뒤 별도 파일 변경은 rollback 원본을 잃는다는 점, selected controls를
wheel distribution metadata로 위장하면 안 된다는 점, 모든 project source를 같은 before-map에
넣고 parent/inode/hash/mode와 single-link를 재확인해야 한다는 점을 대조했다. Cross-filesystem
rename 실패도 원래 cause와 rollback 경계를 유지한다. 새 transaction primitive는 approval을
부여하지 않으며 실제 update/cache 동일성·venv native ownership 연결은 다음 guard가 소유한다.

Canonical profiles와 transaction normal/empty/기존 owner retirement/user files 보존,
각 control/wheel/metadata/sync/reconcile 실패와 drift·동시 guard·recovery 보존을 검증한다.
이 단계의 synthetic filesystem fixtures를 실제 native install/Policy/CLI acceptance로 확대하지
않는다. 실제 전체 소비자는 다음 composition checkpoint에서 검증하고 helper별 full CI는 없다.

## Named local 결과

[Filesystem 영수증](./p4g-uv-project-transaction-local-20261026.json)은 최종 canonical profiles와
focused/full promotion Race를 연결한다. Synthetic project controls+wheel outputs의 정상·empty,
기존 lock 부재/존재의 단계별 원복, drift/alias/동시 guard/recovery 보존이 PASS다.
새 wrapper의 late cleanup 순서는 직접 RED로 반증했고 같은 test source의 교정 GREEN 및
최종 전체 회귀를 확인했다. Exact private cleanup은 원본 backup 폐기 전 commit 내부에서
실행하며 오류도 동일 rollback에 들어간다. Unknown bytes와 두 recovery guard는 보존한다.
기존10 owner bytes와 선언한 삽입 외 공용 commit/verify/sync body는 불변이다.
실제 native project install·ownership/CLI/lock-only/tools·whole P4/remote는 미완료다.
