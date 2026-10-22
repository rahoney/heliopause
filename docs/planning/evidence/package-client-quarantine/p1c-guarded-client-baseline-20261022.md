# P1c C3 — guarded client 연결의 baseline audit

기준 sourceab54d4a. C2 public UV/HTTP qualification의 CI375는 exact hosts read 분류
누락으로 FAIL이며 보완 검증 중이다. 이 문서는 다음 acceptance gap의 읽기 전용 감사다.
구현은 NOT_STARTED, M13-004/P1만 IN_PROGRESS이며 P2..P7로 이동하지 않는다.

Read: [P1c 지도](./p1c-uv-resolver-baseline-20261022.md),
[C2 producer 지도](./p1c-uv-native-producer-baseline-20261022.md),
[Client 계획](../../20-package-client-quarantine-plan.md),
[Verified Set](../../../domain-model/05-dependency-verified-set.md).

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Native UV graph — sandbox/uv_resolver.go + uv_resolver_reader.go | YES | YES(internal/CI), NO(app) | NO(CI375 FAIL) | NO | exact hosts read 보완·새 public actual/negative·실패 정리 |
| 신규 target/venv 반입 — promotion/pypi_promotion.go, pypi_venv_transaction.go | YES | YES(pip) | YES(existing scope) | YES(existing scope) | UV solver와 실제 installer 신원·frozen controls 연결 |
| Guarded update selection — promotion/pypi_update.go ResolveDependencies | YES | YES(pip/PyTorch) | YES(existing scope) | YES(existing scope) | UV native retention/완전 graph 및 선택 증거 adapter |
| Native 설치 enrollment — pypi_native_ownership.go oldWheel/proveNativePyPIWheelFootprint | YES | YES(pip) | YES(existing scope) | YES(existing scope) | UV-installed metadata/marker의 실제 pin layout 조사·adapter |
| Bootstrap/client routing — bootstrap.go pypiInstallDependencyResolver/installer·updater composition | YES | YES(pip/source routing) | YES(existing scope) | YES(existing scope) | Source와 client를 구분한 좁은 registry/서비스 연결 |
| Primary 없는 complete/empty project model — domain/project_snapshot.go, project_update.go, project_verified_set.go; application/project_inspect.go | YES | YES(existing project consumers) | YES(existing scope) | YES(existing scope) | UV resolver/venv transaction의 typed snapshot adapter; Domain 재구현 불필요 |
| Native multi-root/empty sync transaction | NO | NO | NO | NO | Root set/공유 dependency/삭제·retention/atomic rollback 계약과 실제 Native sync 조사 |

## 확인한 경계

- SourceID는 PyPI이고 native client는 pip/UV라는 별도 축이다. 현재 bootstrap의 source별
  resolver/installer map만 바꿔 UV를 pip의 별칭으로 만들지 않는다. 새 Core engine을 만들거나
  모든 계층에 문자열 분기를 넣지 않고 client 선택과 typed adapter의 소유 위치를 한 번 정한다.
- pypiUpdateSession.ResolveDependencies의 public 경계는 원본 입력과 pip delta/report 3개,
  exact python/pip/target/source 문자열·digest·graph를 확인한다. Native UV에는 같은 필드가
  없으므로 위조 report나 broad runtime prefix를 추가하지 않는다. 기존 pip/PyTorch 분기는
  같은 의미/bytes로 유지한다.
- 신규 반입의 실제 materializer는 pinned offline pip이며 solver와 다르다. Artifact 전체
  검역·Verified Set·완전 wheel 집합을 통과한 뒤의 반출은 기존 구현을 재사용할 수 있다.
  UV solver identity와 실제 installer identity를 정확히 구분해 기록해야 한다.
- Native enrollment의 oldWheel은 exact pip runtime을 요구하고 footprint의 INSTALLER/
  REQUESTED·entrypoint/cache 표면도 검증한다. UV-installed venv adoption은 실제 native
  layout을 관찰하고 별도 증거가 필요하다. INSTALLER 문자열만 바꿔 native 소유권을 승인하지 않는다.
- UpgradeInput의 설치 버전/metadata snapshot·target 재검증, 기존 venv journal/rollback·
  adoption 금지/동시 변경 거부를 재사용한다. Fresh empty target의 일반 native resolve는
  설치된 환경에서의 native upgrade/retention 의미를 입증하지 않는다.
- LockedDependencyGraph/현재 bind는 단일 primary root다. Multi-root/빈 requirements sync를
  가짜 Artifact root나 빈 graph의 ALLOW로 맞추지 않는다. 기존 Domain invariant를 유지할
  수 있는 transaction 표현과 삭제/retention 계약을 구체화한 뒤 해당 gap만 구현한다.

### 기존 complete/empty project 모델의 재사용 위치

후속 실제 source 대조에서 ProjectDependencySnapshot은 이미 primary 없는 complete set과
명시적인 NewDependencyFreeProjectSnapshot을 제공한다. ProjectDependencyUpdate의
NewProjectDependencyUpdateFromSnapshot은 requested-primary constructor를 완화하지 않고
original/selected control의 이름·presence·digest와 complete selected snapshot을 연결한다.
NewInspectedProjectSet/ProjectVerifiedSet은 기존 entry 검사 coverage와 독립 set Policy를
요구한다. 따라서 multi-root/empty 표현 자체를 새 Domain 모델로 다시 구현하지 않는다.

application/project_update.go executeGuardedProjectUpdate와 project_inspect.go는 실제
Go/Cargo 소비자의 guard→선택→원본 재확인→entry/set 검사→staging→commit→close를
공유한다. UV sync에서는 이 기존 흐름과 ports.ProjectMutationGuard를 검토 대상으로 삼되,
현재 pypiVenvTransaction은 이 port에 아직 연결되지 않았다. 기존 pip 단일 update와
Go/Cargo를 옮기거나 해당 모델의 기존 acceptance를 UV transaction 성공으로 전용하지 않는다.
빈 sync도 검증된 input/graph와 별도 Policy·소유 파일 삭제 transaction을 통과해야 한다.

두 번째 검토에서 남는 정확한 gap은 UV native multi-root/empty pylock의 strict parsing과
whole archive/edge 검증, frozen requirements 원본·target ownership controls, venv의 selected
set/retired files atomic commit adapter다. C2 단일-root binder를 root별로 반복 호출하면
공유 의존성 선택과 complete graph가 달라질 수 있으므로 전체 선택을 한 번만 수행해야 한다.
이 감사는 source 변경이나 빈 set 자동 승인을 허용하지 않으며 actual C2 이후 구현 순서를 유지한다.

## 다음 검토·검증

C2의 실제 public 결과를 먼저 읽고, pin의 복수 dependency update/retention과 native
venv marker·file inventory를 데이터 전용 조사로 확인한다. 기존 C1a의 단일 update/delta
성공을 전체 retention/sync로 확대하지 않으며 같은 관측을 불필요하게 반복하지 않는다.

Guarded 수정 전에 client별 frozen controls·해시·original input/전체 graph 재검증과
solver/installer 신원, native adoption·multi-root/empty sync의 정확한 수정 위치/방법을
canonical leaf에 확정하고 다시 검토한다. 기존 PyPI/PyTorch 정상/실패·source ownership,
original target 변경/extra file/알 수 없는 source/marker, 취소·cleanup·rollback·atomic publish
회귀와 실제 client/CLI qualification을 포함한다. 완료 owned 자료는 마지막 소비 뒤 제거한다.

새 구현·지원 문구·예외 승인은 이 감사로 생기지 않는다. 기존 구현을 폐기할 이유도 아직
없으며 재사용할 수 없는 지점이 확정되면 영향·대안부터 제시한다.
