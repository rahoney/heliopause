# P6-F — pnpm dlx/pnx·bunx 원본 선택 baseline

Source31acbfe/tree5b40eaf8에서 E2/E3 named local을 마쳤다. CI438은 같은 frozen source의
hosted 검증 중이며20분 GET의 첫 due는06:18:50UTC/15:18:50KST다. 단일 item은 M13-004다.
이 baseline은 source 작성 전이며 새 alias/native/current 지원 완료를 뜻하지 않는다.

Read: PROJECT/Queue/docs router, lessons INDEX·ci L06/L17·sandbox L03/L08/L12/L18·
artifact L01/L02·evidence L16, client09§3/관련 current lanes, plan20§4–6, P2/P3/P5/P6
기존 baseline과 D-012/coding rules, 아래 실제 owner 및 pin과 대응하는 공식 source.

| 실제 책임 / 기존 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| fresh whole npm source/inspection/Evidence/Policy·실행 service/records·private output — application/execution_*.go, bootstrap/node_execution_*.go, cli/execution*.go | YES | YES | CI437 named YES | 기존 named YES | 실제 native one-off resolver/entry/runtime consumers만 연결 |
| native PNPM/Bun tool·network/resource/trace/cleanup — sandbox/native_project_scope.go, pnpm_native_scope.go, pnpm_resolver.go, bun_resolver.go | YES | YES | 기존 project named YES | 기존 named YES | target 없는 실제 dlx/bunx 원본·cache controls; project 성공을 alias에 전용하지 않음 |
| PNPM/Bun native lock·독립 registry SRI/metadata — artifact/npm/pnpm_selection.go, bun_selection.go | YES | YES | 기존 project named YES | 기존 named YES | 단일 one-off primary와 전체 graph/instance를 보존한 실행용 결합; 가짜 InstallContext 금지 |
| npm bin/member byte·Node/runtime/layout — artifact/npm/execution_entry.go/node_execution_entry.go/execution_layout.go | YES | YES | 기존 P5 named YES | 기존 named YES | client-specific default 선택·native peer/layout와 Bun runtime의 실제 차이만 추가; NPM physical layout 규칙 유지 |
| pnpm dlx/pnx, bunx/bun x public native/current/CLI | NO | NO | NO | NO | 아래 순서; namespace/일반 cache·PATH가 승인 authority가 되지 않도록 연결 |
| whole P6/aliases/equivalence/hostile/profile/P7 | partial | partial | NO | NO | 현재 named 검증과 구분해 기존 acceptance를 끝까지 대조 |

## 고정 원본에서 확인한 차이

PNPM pin12.15.0/source91774e72의 `DlxCommand::run`은 runtime/manager provisioning을 먼저
확인하고 별도 cache의 `add_package`로 설치한 뒤 bin을 실행한다. 기본 bin은 sole bin 또는
package basename이며 여러 후보에 basename이 없으면 거부한다. `--package`는 별도 command를
허용하고 `run_bin`은 shell_mode=false에서 `which_in`과 벡터 인자를 사용한다. Cache는
`cache/dlx/<key>/pkg` 링크와 실제 준비 directory/자체 package.json·pnpm-lock.yaml을 가진다.
`configure_cache_install`은 project-local virtual store와 자체 lock을 사용하며 primary build
allow-list도 만든다. 기존 project lock-only와 동일하다는 결론은 아직 미검증이다.

- [고정 dlx](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/cli/src/cli_args/dlx.rs)
- [고정 cache](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/cli/src/cli_args/dlx/cache.rs)
- [현재 공식 pnx/alias·provisioning 문서](https://pnpm.io/cli/pnx)

Bun pin1.4.4/source observed38afee6c는 bunx/bun x를 같은 command에 연결한다. `--package`의
명시 command와 기본 package bin 탐색이 다르고 local/cache/PATH와 daily refresh/fallback을
사용한다. 기본 bin helper는 preferred가 없으면 first bin도 고를 수 있어 canonical HAA의
ambiguity 거부를 임의로 넓히지 않는다. `--ignore-scripts`는 bun add에 전달되며 --trust보다
우선한다. Node shebang 기본과 --bun은 별도 runtime admission이 필요하다. Source observed
commit은 binary source-build provenance 서명을 의미하지 않는다.

- [고정 bunx](https://github.com/oven-sh/bun/blob/38afee6c84fcf48c0edb2466aa4b27d287a59c30/src/runtime/cli/bunx_command.rs)
- [현재 공식 bunx 문서](https://bun.sh/docs/pm/bunx)

Fetched source는 task 임시 원본 p6f-pnpm-sources와 p6f-bunx-source에서 SHA로 고정하며
native 실행/안전성/지원 성공으로 세지 않는다. 새 version/dependency/tool/job는 추가하지 않는다.

## 최소 구현 순서와 두 번째 검토

1. **F1 PNPM 실제 cold 선택 DATA:** 기존 PNPMResolver/nativeProjectScope·200MiB bounded
   layout parser를 소비한다. 고정 ignore-scripts/pnpmfile/workspace·공개 registry와 fresh private
   cache에서 `dlx --package <bounded bare/exact>`를 사용하고 command는 보호 Node의 절대 경로와
   고정 --version만 전달하는 후보를 먼저 검증한다. Artifact bin/사용자 argv는 이 준비 단계에
   넘기지 않는다. 기존 exact PNPM Node-query creator/image 경계를 재사용할 수 있는지 actual로
   확인한다. Native cache의 한 pkg 링크·실제 directory·regular controls를 읽고 기존 PNPM parser로
   결합한다. Provisioning 이름/foreign input은 resource I/O 전 거부하며 startup/lifecycle·관측
   불완전·정리 오류에서 partial selection을 비운다. Query stdout은 완료 authority가 아니다.
2. **F2 source/current 연결:** existing independent registry/archive 검사와 client별 source/entry
   choice를 현재 approved-original 파이프라인에 연결한다. Native instance/peer 관계와 physical
   layout은 기존 native validators를 재사용하며 NPM flat layout 또는 InstallTarget으로 위장하지
   않는다. 필요한 경우 실제 두 소비자가 생긴 기존 함수의 책임만 공유한다.
3. **F3 CLI/실제 회귀:** pnpm dlx/pnx/pnpx의 같은 typed operation, default/명시 package-console,
   runtime/provisioning·cache/source substitution/취소·출력·cleanup과 기존 npm 소비자를 묶는다.
4. **Bun 후속:** 위 공용 owner의 실제 재사용 지점을 확정한 뒤 bunx/bun x의 pinned native cache와
   Node/Bun current runtime만 연결한다. 아직 Bun query/node-child 권한을 선행 확장하지 않는다.

새 state/daemon/중복 resolver·검사 engine·미래 placeholder를 만들지 않는다. 기존 CPP나
pipeline의 구현을 대체하지 않으며 새 native 동작이 실제 필요로 하는 최소 지점만 고친다.
Cheap parser/ABI 검사를 먼저 모으고 완성된 단위의 canonical/관련 actual을 실행한다.
CI438을 새 push로 취소하지 않으며 작은 helper별 CI를 만들지 않는다.
