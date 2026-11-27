# P6 — 실행 adapter 확장 전 기존 owner 감사

Baseline5a45a6f52164152f1da9910942ff231ff5d0f01f/treef8bf9c26/3000tracked·1344runtime다.
Production1343은 active CI429/source9eb854f와 같고 C7 actual refusal test1만 추가됐다.
단일 active work item은 M13-004다. CI429 first20min query02:17:12UTC는8SUCCESS/Linux 진행,
completed8 same checkoutb2eb8b6e/tree51a9a25c·decoded448735B/CPP131이다. Timer02/session61181
due02:37:12UTC(한국11:37:12)를 먼저 소비하며 실행 중 새 push하지 않는다.

Read: Queue/PROJECT-DECISIONS/docs README routes, INDEX의 ci L05/L06·sandbox L03/L08/L12/L18·
artifact L02/evidence L16, client09§1..4/current npx scope·plan20§4..6, P5 baseline/C3d3/C4/C5/C7
plan·receipts, Architecture001/005/007와 Contract002·Domain003, 다음 실제 source owners.
아래 판정의 YES는 명시한 기존 scope만 뜻하며 P6 전체 완료로 전용하지 않는다.

## P5 초기 lane의 현재 acceptance 경계

| 기존 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 남은 범위 |
| --- | --- | --- | --- | --- | --- |
| exact source/whole approval/bin/context/readonly Node/runtime·bounded output | YES | YES | YES | YES | C3d3 current12 named local 및 CI428의 이전 normal/nonzero named hosted 범위 |
| public npx/npm exec·independent report/private or escaped streams | YES | YES | YES | YES | C5 local CLI4 범위; 새 hosted는 CI429 대기 |
| Policy ALLOW 뒤 ambiguous/missing entry actual 거부 | YES | YES | YES | YES | C7 local refusal2; 새 test source는 CI429에 없음 |
| current whole1343 grouped ordinary hosted | YES | YES | NO | NO | CI429 Linux/Required·실제4parents/checkout/logs |
| whole P4/P5-P7/final4profiles/signed | NO | NO | NO | NO | 기존 plan의 final matrix/범위 대조; 대표 local 또는 ordinary로 대체하지 않음 |

파일 생성/반출·stdin/TTY는 초기 stdout lane의 미지원 capability다. Child body/held-pipe는
clone-startup 거부·정리 범위를 넘어 입증되지 않았다. 이를 기존 성공으로 표시하거나 grant를
넓혀 시험을 만들어 내지 않는다. P6 source 분석은 위 remote 결과와 독립적인 설계 감사다.

## 실제 공유 owner — 재구현하지 않는 영역

| source / 책임 / 기존 scope | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | P6 MISSING |
| --- | --- | --- | --- | --- | --- |
| domain/execution_context.go: NewExecutionContext·argv/command limits·whole set/current fingerprints (P5) | YES | YES | YES | YES | 이미 ecosystem-neutral이다. 새 context 종류나 중복 validator를 만들지 않고 Python entry의 실제 provenance를 결합 |
| application/execution_inspect.go: fresh whole resolved inspection (P5) | YES | YES | YES | YES | actual PyPI source/verifier/composite·현재 entry/set Policy를 factory에서 연결 |
| application/execution_prepare.go: resolver/binder/records/staging preparation (P5) | YES | YES | YES | YES | 별도 InstallTarget 없이 새 client resolver와 measured entry binder |
| application/execution_run.go: same-context runtime and private outcome/error axes (P5) | YES | YES | YES | YES | Python concrete runtime adapter; NodeProgramError를 Python 완료 증거로 전용하지 않음 |
| evidence/execution_manifest.go: Generator.BuildExecution·SBOM·no argv/output (P5) | YES | YES | YES | YES | 이미 ecosystem-neutral이다. Python console metadata/member 의미를 명시하고 기존 schema semantics와 대조 |
| neutral CapturedProcessOutput/Host protected capture·external target/terminal/cleanup (P5/common) | YES | YES | YES | YES | 현재 Python user-run admission·output/lifecycle 연결. 기존 inspection handoff의 scope를 넘어 일반 권한으로 사용하지 않음 |
| cli/execution_report.go·descriptor/preflight/error privacy (P5) | YES | YES | YES | YES | 실제 P6 grammar/alias가 준비된 뒤 같은 facts/presentation 재사용; future descriptor 없음 |

## 기존 Python/client owner와 실제 gap

| source / 기존 scope | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | P6 MISSING |
| --- | --- | --- | --- | --- | --- |
| artifact/pypi/wheel.go: InspectWheel/RECORD/whole metadata | YES | YES | YES | YES | canonical parser를 재사용한 exact console selection·원본 byte/member binding |
| observation_plan.go: EntryPoint/Surface canonical details (Model C) | YES | YES | YES | YES | console name→module/attr는 metadata다. import completion·NOT_ATTESTED를 user-run 또는 callable 성공으로 바꾸지 않음 |
| ToolReceipt/UV layout·script proof/ownership (P4) | YES | YES | YES | YES | 영구 설치 ownership과 ephemeral 실행 provenance를 구분; Host install target을 가짜로 만들지 않음 |
| sandbox/uv_tool_resolver.go: ResolveToolInstall/native tool-install selection (P4) | YES | YES | YES | YES | run/uvx selection의 실제 의미·bounded equivalence 증거; install identity를 uvx라고 이름만 바꾸지 않음 |
| sandbox/pipx_resolver.go: ResolveToolInstall/Upgrade, backend UV + resource binding (P4) | YES | YES | YES | YES | pipx run의 실제 selection/cache/parser 차이, 새 ephemeral approval/entry/runtime 결합 |
| 기존 Python inspection/materialization/runtime supply/profile (P4/Model C) | YES | YES | YES | YES | whole frozen read-only 실행 layout와 선택된 console 호출·argv/bounded output·독립 terminal/cleanup actual |
| P6 uvx/uv tool run/Python user-run adapters | NO | NO | NO | NO | native source/selection proof→typed console entry→same fresh inspection/context→runtime→CLI/actual |
| P6 bunx/pipx run/pnpm dlx adapters | NO | NO | NO | NO | 각 native semantics와 current runtime/layout/entry 선택. npm alias로 몰래 대체하지 않음 |

## 고정 UV 원본 — DATA 감사

Runtime lock은 UV0.13.4 OCI digest f0a9c230과 provenance source commit
7456b60bc6bc348e13e06e3691bf08e78d3aef54를 이미 소유한다. 버전/공급을 갱신하지 않는다.
공식 Git commit/tree와 실제 crates directory를 조회했다. 현재 tool commands는 별도
`crates/uv-tool-commands`에 있으며 기존 uv commands module의 alias로 연결된다.
[Source discovery](./p6-uv0134-source-discovery-20261029.json)의 run/install/common/requirements 및
command alias 다섯 file의 decoded bytes·Git blob SHA1·독립 SHA256/readback을
확인했다. 이 source 감사는 native execution/selection equivalence 또는 지원 qualification이 아니다.

확인된 원본 의미:

- `run.rs:run`은 ToolRequest를 만들고 get_or_create_environment로 환경을 선택한 뒤 실행한다.
  command는 entry-point metadata만으로 고정되는 것이 아니며 명시 from에서 다른 환경/PATH의
  executable도 허용한다. PATH와 env-file 값·상속 stdio를 그대로 HAA capability로 수입하지 않는다.
- get_or_create_environment는 isolated가 아니고 latest 요청도 아니면 기존 tool receipt·환경을
  조회하며, 별도 CachedEnvironment 경로도 있다. 그 cache는 HAA의 fresh 검사·Policy authority가 아니다.
- 기존 HAA selector는 `uv tool install`의 fresh private home·receipt/whole controls/independent
  wheel binding을 사용한다. 그 selection이 bounded uvx와 같다는 결론은 아직 UNPROVEN이다.

원문: [run](https://github.com/astral-sh/uv/blob/7456b60bc6bc348e13e06e3691bf08e78d3aef54/crates/uv-tool-commands/src/run.rs),
[install](https://github.com/astral-sh/uv/blob/7456b60bc6bc348e13e06e3691bf08e78d3aef54/crates/uv-tool-commands/src/install.rs),
[common](https://github.com/astral-sh/uv/blob/7456b60bc6bc348e13e06e3691bf08e78d3aef54/crates/uv-tool-commands/src/common.rs),
[requirements](https://github.com/astral-sh/uv/blob/7456b60bc6bc348e13e06e3691bf08e78d3aef54/crates/uv-tool-commands/src/requirements.rs).
원본과 inventory는 raw p6-uv0134-* 한 곳에 보존한다. 새 HAA source/placeholder package/interface/
config/job·runtime pin·권한 변경은 없다.

## 다음 구현 순서와 두 번째 검토의 기준

1. Exact native source와 현재 Python wheel/entry APIs를 대조해 UVx 첫 bounded lane의 selector와
   metadata/byte proof 경계를 확정한다. Install reuse의 실제 증거 또는 별도 safe selection이
   있어야 하며 native run으로 아직 검사하지 않은 artifact를 CONTROL 역할에서 실행하지 않는다.
2. Python console selection의 metadata/member·function/runtime provenance만 해당 기존 owner에
   추가한다. Wheel parser/Model C/Domain context/Application·Manifest를 재구현하지 않는다.
3. 실제 fresh whole PyPI 검사와 measured entry를 현재 preparation에 연결하고 readonly whole
   layout/typed runtime admission·private output/terminal·cleanup actual을 별도로 닫는다.
4. 실제 지원된 grammar/alias만 CLI descriptor에 등록하고 native cache·source/entry substitution/
   unsupported runtime·malicious/cancel/overflow·normal/nonzero 및 기존 소비자를 검증한다.

위 순서는 코드 승인이나 구현 완료가 아니다. 각 작은 source 변경 전 exact owner map/두 번째
검토와 acceptance gap을 다시 기록한다. 기존 지원을 대체/재작성할 이유가 생기면 AGENTS의
기존 위치·수정으로 불가능한 이유·영향 범위를 먼저 제시한다. Host assets/credentials/network나
runtime/coverage invariant를 구현 편의로 넓히지 않는다. CI429 감시는 계속하며 새 push하지 않는다.
