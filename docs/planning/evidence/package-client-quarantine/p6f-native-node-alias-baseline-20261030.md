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
   cache에서 `dlx --package <bounded bare/exact>`를 사용한다. 최초 Node --version 후보는 아래
   실제 실패 뒤 폐기했다. 최종 command는 sealed helper mount에 존재하지 않는 고정 절대 경로
   `/haa-runtime/helox-pnpm-inert-query`다. Artifact bin/사용자 argv나 추가 Node를 실행하지 않는다.
   Native cache의 한 relative pkg 링크·실제 directory·regular controls를 읽고 기존 PNPM parser로
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


## F1 최초 실행과 폐기한 후보

First cache focused2.370s의 absolute pkg alias 거부는 잘못 가정한 fixture였다. 실제 v6 tar는
`pkg -> mvumgj10-9`의 relative basename이며 기존 bounded tar reader로 읽힌다. 공용 reader의
임시 확장과 cache-root 인자는 제거했다. 실제 v2는 recorder가 TrustedExecutor interface만 embedding해
기존 optional 실행 메서드를 잃은 상태에서 boundary 준비가 실패했다. Concrete executor를
embedding한 test-only v3는 그 단계를 통과해 미지원 --cache-dir/exit2를 보존했다. Pin source의
XDG_CACHE_HOME/pnpm 기본 경로와 기존 earlyStore=true를 재사용한다. v4의 project 부재는
새 디렉터리/가짜 manifest 대신 이미 있는 private volume root를 cwd로 써 교정했다.

폐기한 Node-query v5 actual10.31s의 최초 fault는 OPEN_RESULT_CLASSIFICATION_OTHER, DEV_NULL/flags557057이다.
기존 numeric fingerprint b9ac38063e52bfd7은 고정 /haa-runtime/pnpm과 일치하며 진단의 OTHER는
PNPM enum의 표시값이다. Current exact IsPinnedPNPMRuntimeRead는 flags557056만 받는다.
실험한 null-output predicate/CPP132 PASS 뒤 v6는 cache를 생성했으나 absolute-link fixture와
추가 exec 관측에서 실패했다. 추가 exec의 정확한 인과는 미확정이며 지원 완료로 쓰지 않는다.
불필요한 Node query를 제거한 최종17argv v7는 새 cold10.83s와 기존 startup3.70s/Node3.77s가
PASS였다. 같은 Go ELF/공급물과 기존 observer989b16c4의 cold 단독 비교10.67s도 PASS여서
실험한 C++ 예외가 최종 경로에 불필요함을 직접 확인했다. CPP2파일·pod-init·공용 tar reader는
3ef10c9 bytes로 복원했다. 해당 권한 예외를 해결법으로 남기거나 CPP/native를 반복하지 않는다.

최종 public SelectToolRun은 기존 scope/관측/정리와 native lock parser·독립 registry SRI binder를
재사용해 roots1/packages1/instances1/cache169/source-artifacts1의 unapproved DATA만 반환한다.
Expected missing-command stderr는 producer 준비 진단이며 완료 authority가 아니다. 기존 scope의
complete trace·정리 실패에서는 zero selection을 반환한다. 모든 actual은 helperwait0/Host 보호5·
Docker 보존/원래 profile 동일 inode 복원을 확인했다. Final scope/검사는 F1 receipt를 따른다.
현재 program/Policy/CLI·peer 전체 결합·일반 동등성·새-source hosted와 whole final은 OPEN이다.

## F2 source 작성 전의 실제 gap

| 책임 / 실제 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| PNPM exact registry/SRI와 root·snapshot parser — pnpm_selection.go/BindPNPMLock, pnpm_lock.go | YES | YES | 기존 project와 F1 named local YES | 기존 named YES | 다시 구현하지 않음 |
| source-neutral 실행 graph/whole 검사 — domain/install.go, application/execution_inspect.go | YES | YES | 기존 named YES | 기존 named YES | PNPM native instance를 같은 graph에 정확히 결합 |
| ephemeral PNPM source→실행 resolution | NO | NO | NO | NO | 한 primary, 모든 peer instance·edge, 독립 원본과 controls/runtime fingerprint; 가짜 InstallContext 금지 |
| client-specific PNPM bin 선택 | NO | NO | NO | NO | sole 또는 다중 basename 기본 선택을 기존 explicit SelectExecutionEntry에 위임 |
| PNPM current whole 검사·native 실행 layout/runtime/CLI | NO | NO | NO | NO | 기존 project inspection은 InstallContext, 기존 Node layout은 NPM physical topology 전용이므로 의미를 바꾸지 않고 필요한 연결만 수행 |

F2는 먼저 위 source resolution/entry의 실제 gap을 구현한다. `PNPMInstanceDirectory`와
기존 Domain constructor로 peer별 node/record path·native edges를 유지한다. 여러 instance가
공유하는 tarball을 하나의 instance로 합치지 않는다. Frozen controls/request와 독립 SRI를
결합하고 unsupported/disconnected graph는 기존 Domain 경계에서 거부한다. 그 다음 실제
current inspection/layout 소비자를 연결한다. Generic npm selector나 설치 target 계약을
넓히지 않으며 이 중간 단위로 새 CI를 시작하지 않는다.
