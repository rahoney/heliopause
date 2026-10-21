# Package client expansion — baseline·수정 지도·검증 계획

2026-10-21, baseline `72a90760bb3c46987642b5d0b3eda450a60310a0`.
[설계](../user-journey-cli-ia/09-package-client-quarantine.md)가 기능·보안 경계를 소유한다.
이 문서는 사용자 요청의 사전 설계 작업이며 신규 client source는 아직 작성하지 않았다.
Queue의 M13-004만 IN_PROGRESS다. 한 확장 범위를 작은 검증 가능한 단계로 진행한다.

## 1. Repository baseline audit

| 책임 / 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| npm/PyPI 기존 install/update — `internal/application/install*.go`, `internal/sandbox/{npm,pypi}_resolver.go`, `internal/promotion/{npm,pypi}_update.go`; [SEC-UPD 최종 검토](./evidence/update-quarantine/final-review-20261019.md) | YES | YES | YES | YES | 기존에 검증된 bounded scope에는 없음; 새 client에 qualification 전용 금지 |
| Canonical Policy/staging/transaction·observer — 기존 M8~M12 및 위 소비자 evidence | YES | YES | YES | YES | 새 layout/runtime/실행은 별도 gap |
| uv pip/project, pnpm, Bun, pipx tool 설치·업데이트 | NO | NO | NO | NO | 선택 버전/runtime, parser/resolver/layout/CLI 연결, 실제 install/update·negative·rollback |
| npx/npm exec, uvx/uv tool run, bunx/bun x, pipx run, pnpm dlx | NO | NO | NO | NO | frozen execution context, 입출력/격리/진입점/취소/cleanup, actual hostile qualification |
| 새 client maintenance | NO | NO | NO | NO | lock/schema·생성·배포물·updater·drift 검증 연결 |
| 기존 launcher — source8d949a7 local Quick/PTY·기존 CLI-UX-003 CI352 | YES | YES | YES | YES | 새 full-name brand 요구만 미완료; 현재 preview는 owner 확인 대기 |
| 새 full-name/background brand | NO | NO | NO | NO | 기존 renderer 좁은 수정·반응형/color/PTY 확인·preview 전달 |

신규 client 명령/production adapter가 없음을 root command tree, bootstrap dispatch,
scoped Go source 검색과 runtime lock을 대조해 확인했다. 기존 구현을 대체하지 않는다.
Generic install/update engine을 다시 만드는 승인은 요청하거나 가정하지 않는다.
구현 시 baseline 이후 변경과 실제 owner symbols를 다시 확인하고 MISSING만 수행한다.

## 2. 최신 버전 확인과 채택 절차

2026-10-21 07:06 UTC 공식 GitHub `/releases/latest` 다섯 endpoint 모두 HTTP200,
non-draft/non-prerelease. [원본 관측 요약·URL·응답 digest](./evidence/package-client-quarantine/version-review-20261021.json).

| Component | 관측 최신 안정판 | 현재 HAA pin | 판단 |
| --- | --- | --- | --- |
| uv / uvx | 0.13.4 | 없음 | 후보, 미채택 |
| pnpm / dlx | 12.14.0 | 없음 | 후보, v12 alias/provisioning 의미 확인 필요 |
| Bun / bunx | 1.4.3 | 없음 | 후보, Node/Bun 실행과 libc/CPU baseline 검증 필요 |
| pipx | 1.18.4 | 없음 | 후보, exact 버전 backend/metadata 확인 필요 |
| npm / npx | 12.2.1 | npm11.19.0 / Node24.21.0 | major 변경; npx 작업을 이유로 자동 업그레이드하지 않음 |

Python3.14.7/pip26.2.1/cp314·manylinux_2_36_x86_64, Go1.26.9 및 기존 runtime/image
pins는 유지한다. 위 API metadata만 읽었고 asset 다운로드·실행·서명 검증은 하지 않았다.
Registry dist-tag/PyPI release와 exact upstream tag, Node engines/Python Requires-Python,
Bun libc/CPU 요구를 구현 진입 시 대조한다. 불일치/실패는 UNKNOWN으로 남기고 날짜를
갱신한 공식 metadata부터 확인한다. 최신판을 지원 버전으로 미리 표시하지 않는다.

새 native executable의 공급망·ABI/observer admission은 기존 trusted tooling 계약으로
검증한다. Unknown schema/version은 조기 거부하고 parser compatibility fixture를 둔다.
Patch 업데이트도 canonical plan/apply/verify와 실제 영향 소비자 검증을 거친다. 변경 없는
코드 전체를 매번 재작성하지 않으며 lock·generated identity·support evidence를 함께 갱신한다.

## 3. 정확한 수정 위치와 재사용 방법

경로는 baseline에 존재한다. 새 파일이 필요한 곳은 기존 package 안의 좁은 adapter로
구현하며 여기서 placeholder를 생성하지 않는다. 실제 diff에는 아래 symbol과 caller/test를 연결한다.

| 기존 위치 / symbol | 계획한 변경 | 검증 경계 |
| --- | --- | --- |
| `internal/cli/root.go` `initializeCommandTree`, AddNPMInstall/AddPyPIInstallSourcesAndUpdates; `internal/cli/presentation.go` PreparePresentation | source와 client/intent 구분, 명시 command descriptor를 기존 command builder/preflight에 연결; alias와 flag 중복 선언 축소 | direct/help/JSON·unsupported-before-composition, `--`·empty/duplicate/unknown flags |
| `internal/bootstrap/bootstrap.go` Run, installDependencyResolver/pypiInstallDependencyResolver | descriptor로 필요한 client factory만 선택; 기존 lazy help/doctor 유지 | unsupported platform/refusal에서 tool launch0; adapter 선택과 shutdown |
| `internal/core/domain/operation.go`, `internal/core/ports/update.go` | 기존 install/update request 유지; 실행 context/result가 기존 모델로 표현 안 되는 gap만 typed 추가 | enum/schema version·old result consumers; run 실패가 INSTALL COMPLETED로 보이지 않음 |
| `internal/application/install.go`, install_inspect.go, install_promote.go, install_update.go; project_update.go executeGuardedProjectUpdate | 전체 set 검역/Policy/rollback 재사용; run은 ALLOW 뒤 별도 orchestration 단계 | BLOCK/incomplete에 mutation/exec0, no-op도 승인 동일성, failure cause 보존 |
| `internal/sandbox/npm_resolver.go` ResolveDependencies/ResolveDependencyUpdate; pypi_resolver.go pypiResolveArguments | 기존 npm/pip 유지, uv/pnpm/Bun resolver sibling adapter; typed network·bounded metadata parsing 재사용 | exact native solver와 graph 대조, peer/optional/extras/marker, network 우회·unknown format 거부 |
| `internal/artifact/npm/lockfile.go`, `internal/artifact/pypi/wheel.go` 및 entry point identity | archive/wheel 검사 재사용; client lock parsing은 기존 package 안의 별도 formatter/parser | renamed/case/ambiguous bins, hashes·RECORD·dependency closure 불일치 |
| `internal/promotion/npm_transaction.go` backupCurrent/verifyOriginalMember, npm_transaction_member.go; pypi_venv_transaction.go/pypi_update.go | current npm/venv 보호 유지; client별 control/layout adapter와 tool metadata/entrypoint transaction 추가 | pnpm internal link vs escape, foreign ownership, concurrent edit, rollback/journal |
| `internal/promotion/mutation_lock.go`, control_transaction_backup.go | 동일 target lock·old/new snapshot guard 재사용 | client를 바꾸어 동시 접근해도 동일 target serialize; late failure 원상복구 |
| `internal/sandbox` 실행 backend, `internal/hosttool` supervisor, `tools/gvisor-observer/observer.cc` | 기존 lifecycle/observer를 run lane에 연결; 새 native runtime의 exact admission만 실제 필요 시 추가 | parent/child/exec generation·외부 completion, timeout/SIGINT 후 leak0; broad exception 금지 |
| `scripts/runtimes.lock.json`, generate-runtime-lock.go → `internal/runtimeidentity/runtime_lock_gen.go` | schema migration으로 client identity·runtime relation 단일 입력, derived 값 생성 | old/new schema, duplicate alias pin 방지, contents/digest mismatch 거부 |
| `scripts/update-runtime/{metadata,patch,apply}.go`, runtime-image-manifest, `scripts/check/runtime_versions.go`, version-support.lock.json | 기존 maintenance owner에 client 후보/영향/pin·image·support drift 연결 | offline deterministic verify, stale apply precondition, asset integrity/지원표 일치 |
| `internal/cli/launcher_examples.go`, presentation_catalog.go, result rendering | 실제 wired capability만 menu/help 예시로 노출; 7개 언어 설명·안정된 영어 code | 영어/한국어 전체·나머지 fallback, 검사/실행/반출/cleanup 상태 분리 |

새 abstraction은 두 실제 소비자의 공통 책임이 확인될 때만 추출한다. Command descriptor에
shell 문자열·권한·정책 예외를 넣지 않는다. 버전 숫자가 code 곳곳에 생기지 않도록 generated
identity를 사용하되 명시 capability/보안 한도까지 사용자 설정으로 느슨하게 만들지 않는다.

## 4. 순서와 종료 조건

각 단계 시작 시 Queue의 현재 단일 active item 아래 정확한 substep을 기록한다. 이 표는
일정 보장이나 병렬 IN_PROGRESS 항목이 아니다. 기존 채널 배포는 새 지원을 과장하지 않는다.

| 단계 | 시작 조건 / 수행 | 완료 조건 | 현재 |
| --- | --- | --- | --- |
| P0 설계/사전 검토 | 사용자 범위·기존 owner·공식 문서·버전 후보 대조 | 본 문서와 leaf, 재검토·canonical docs/security | 작성 및 재검토 완료, 최종 검사 대기 |
| P1 client foundation + uv pip | exact uv runtime·좁은 descriptor, 기존 install/update 재사용 | actual uv install/upgrade/sync, 기존 pip/npm 회귀, 실패 불변·정리 | NOT_STARTED |
| P2 pnpm | fixed resolver/lock/layout·공통 descriptor 재사용 | actual add/install/update/peer graph·link 공격·rollback | NOT_STARTED |
| P3 Bun | fixed Bun resolver/layout/lifecycle | actual install/update·Node vs Bun runtime 요구 분리·회귀 | NOT_STARTED |
| P4 uv project / persistent tools | uv add/sync/lock-upgrade, uv tool/pipx install·upgrade metadata·exposed scripts | project lock-only vs installed 상태, tool ownership/retired scripts/rollback | NOT_STARTED |
| P5 execution foundation + npx | full frozen set·entrypoint·sandbox run·output/exit/lifecycle 계약 | actual 정상/악성/취소/재다운로드 거부, stdout-only 우선; 생성 파일/TTY는 별도 acceptance | NOT_STARTED |
| P6 uvx·bunx·pipx run·pnpm dlx | P5 재사용, 각각 parser/runtime/selection adapters | 각 alias real end-to-end, forced runtime/backend·cache negatives | NOT_STARTED |
| P7 통합 qualification/인계 | 모든 목표 command/옵션 지원표·remaining gaps 대조 | exact-source CI/실제 runtime·기존 소비자·docs/help·cleanup 모두 연결 | NOT_STARTED |

같이 계획하되 모든 도구를 한 commit에서 구현하지 않는다. 단계마다 의미 있는 한글 commit,
경계별 evidence와 실제 미완료를 남긴다. 구현 중 새로운 invariant 변경 필요가 생기면 그
항목을 구체화하고 승인 전 해당 변경은 수행하지 않는다. 명칭/버전/계정·public release는 별도다.

## 5. 검증 matrix와 자원 정리

- Canonical format → quick (gofmt check/build/test build/vet/pinned Staticcheck/default tests)
  → security/docs. Runtime pin 변경이면 기존 runtime/release 관련 profile도 적용한다.
  Exact 필요한 profile 이름은 현재 scripts/check help/maintenance contract에서 확인한다.
- 각 client의 native golden resolution과 frozen replay를 분리한다. Public 실제 acquisition,
  registry authentication/digest, 실제 설치 tree/lock, 실제 gVisor observation·Policy와 결과를
  함께 기록한다. Fake resolver 성공이나 help exit0로 실제 지원을 닫지 않는다.
- 각 lane: clean install, selected/full update, no-op, constraints/peer/extras/optional,
  malformed lock/report, dependency substitution, digest/source mismatch, 악성 lifecycle,
  incomplete observation, cache tamper, unsupported ABI/platform/runtime.
- 각 transaction: native adoption 거부/명시 허용 범위, arbitrary target·symlink/hardlink escape,
  other client 동시 mutation, disk full, SIGINT, process crash, late promotion failure, rollback
  실패·recovery 재진입, old project bytes/ownership 일치. 원래 파일을 모르는 채로 삭제하지 않음.
- 각 runner: local-first package/cache도 재검역, package≠bin, 다중 bin/잘못된 shebang,
  추가 dependency/remote fetch, shell injection, runtime download, argv delimiter, stdin/TTY,
  stdout control/OSC/크기 폭주, nonzero/signal/timeout, child orphan/FD/container/slice leak.
  Output 생성 경로는 traversal/link/conflict·partial export·기존 사용자 파일 보존 별도 검증.
- 영향 분석에 따라 기존 npm/PyPI/Go/Cargo/Terraform/GitHub와 CPU/cu126/cu130/cu132를 선정한다.
  공통 observer/Python runtime을 바꾸면 관련 네 profile 소비자를 빠뜨리지 않는다.
  별도 baseline Race 실패를 숨기거나 작은 focused PASS를 전체 성공으로 확대하지 않는다.
- 초기 Linux/WSL2 amd64 실제 검역과 Linux/macOS CLI native PTY는 서로 다른 acceptance다.
  CI는 actual checkout/tree·job conclusion까지 기록하고 실행하지 않은 플랫폼은 NOT_TESTED다.
- 한 task 전용 directory/image/container/volume/cache inventory를 만들고 완료 후 참조·query
  errors0을 확인해 삭제한다. Root·공유 cache·사용자 project는 일괄 삭제하지 않는다.
  작은 logs/result/hash/실패 원인만 보관한다. 사용자 확인 중 preview는 최종 소비 전까지 유지한다.

## 6. 두 번째 검토 — source 작성 전

2026-10-21 baseline 코드 위치, official behavior, 기존 계약과 사용자 요구를 다시 대조했다.

| 점검 | 처음 잘못 설계하기 쉬운 부분 | 최종 반영 |
| --- | --- | --- |
| source/client 구분 | pnpm registry 신설, uv를 pip alias 처리 | 동일 npm/PyPI source + 별도 resolver/layout |
| 실행기 동등성 | inspect 후 원래 npx 호출 | same bytes+graph+entrypoint/runtime binding; Host 재resolve 금지 |
| pipx 범위 | 일회성 전용으로 처리 | install/upgrade/run 및 backend 고정·persistent scripts ownership |
| uv 범위 | uv pip와 project/run 동등 취급 | pip/project/tool 별도 acceptance, uv run 자동 sync 명시 미지원 |
| pnpm 최신 alias | 오래된 dlx 의미만 적용 | v12 pnx/dlx provisioning 감지·runtime provisioning 거부 |
| snapshot/lock | 파일명만 npm lock으로 치환 | client별 typed parser/transaction, lock-only 설치 성공 금지 |
| runtime 버전 | 공식 최신판을 즉시 supported로 표시 | 후보 관측과 qualified pin 분리; npm major 유지 |
| 기존 코드 | 모두 새 공통 엔진으로 대체 | existing install/update/Policy/transaction 재사용, gap만 adapter 추가 |
| native executable | package검사만으로 uv/Bun runtime 신뢰 | 공급망·ABI·observer admission 및 배포 manifest/maintenance 영향 |
| 실행 결과 | exit0 또는 sandbox venv를 안전 증명으로 사용 | 검사/실행/output/cleanup 분리, post-install entrypoint coverage 경계 보존 |
| UI | 지원 전 example 추가, 모든 화면 거대 타이틀 | 실제 wired 명령만; 홈의 full-name wordmark·responsive fallback |
| 검증·정리 | 대표 성공으로 전체 지원, 테스트 bulk 보존 | 각 lane·platform 실제 증거, 참조 확인 후 owned output 삭제 |

남은 불확실성: 후보 버전의 실제 sandbox 호환성·lock/report 형식 세부·client 성능/지원 한도,
TTY/생성 파일 반출의 실제 필요한 범위. P1~P6 첫 작업에서 exact fixture·보안 한도를
확정한 뒤 source를 작성한다. 미확정을 완료로 치환하지 않는다. 문서화는 이 조사와
설계 범위에서 완료할 수 있으나 신규 client 구현 완료는 아니다.
