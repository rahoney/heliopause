# P1c C3 — guarded client 연결의 baseline audit

최초 기준 sourceab54d4a, 후속 조사 기준35c0f2d. C2 public UV/HTTP qualification의 CI375는
exact hosts read 분류 누락으로 FAIL이며 보완한 CI376을 확인 중이다. 이 문서는 다음
acceptance gap의 읽기 전용 감사다.
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
| Native 설치 enrollment — pypi_native_ownership.go oldWheel/proveNativePyPIWheelFootprint | YES | YES(pip) | YES(existing scope) | YES(existing scope) | 관측한 UV cfg/marker의 adapter; cache schema/생성 파일·entrypoint proof 및 actual enrollment |
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

## CI 대기 중 독립 data-only 형식 조사 계획

C3 제품 구현은 NOT_STARTED로 유지한다. 기존 C1a는 단일 packaging delta만 확인했으므로
반복하지 않고, 인증된 같은 UV/Python pin의 runc/network-none/readonly/UID1000/no-cap
컨테이너에서 inert 합성 wheel만 사용해 다음 미확정 의미를 조사한다. Host package/cache를
설치하거나 wheel의 module/entrypoint를 import·실행하지 않는다.

- Baseline root1/dep1/extra1을 private target에 두고 root2의 요구 범위가 dep1·dep2 모두
  허용할 때 기본 재설치, root만 --upgrade-package, 전체 --upgrade의 실제 delta와 잔류
  installed inventory를 비교한다. 각 경우 별도 target을 쓴다. JSON은 선택 진단용이다.
- 다중 requirements의 compile→pylock→sync 및 빈 requirements의 compile/sync를 따로
  관측한다. Root/dependency/extra의 제거와 empty-format 거부 여부를 기록하며 임의 root를
  만들어 production parser를 통과시키지 않는다.
- 같은 pin의 private venv에 설치한 뒤 pyvenv.cfg와 dist-info INSTALLER/REQUESTED/RECORD,
  전체 제한된 파일 inventory를 데이터로 수집한다. 이 marker는 native adoption authority가 아니다.

두 번째 검토: 개별 결과를 public registry/gVisor/Policy·actual guarded update 성공으로 쓰지
않는다. Native CLI가 실패하면 최초 명령/exit/stdout/stderr를 보존하고 다음 큰 시험을 멈춘다.
Owned container/image는 exact ID/image/label 확인 뒤 제거하고 image --no-prune를 사용한다.
Host 파일/image/container/bridge/service snapshot을 전후 대조하고 완료 payload/target과
derived image·일회성 작업 경로를 삭제한다. Query 실패는 부재로 처리하지 않는다.

첫 data-only 입력의 기본/root-only upgrade는 dep1을 유지하고 전체 --upgrade는 dep2를
선택했다. Extra는 세 경우 모두 남았으며 다중 compile/sync도 성공했다. 빈 compile은
조사 operator가 `empty-pylock.toml`이라는 잘못된 출력명을 지정해 native exit2로 중단됐다.
이것은 empty sync 미지원이나 제품 parser 결함이 아니다. 첫 원본과 Host 불변/owned 자원
정리를 보존하고 통과한 retention/multi 입력은 반복하지 않는다. 후속 조사는 허용된
`pylock.empty.toml`과 별도 venv/단일 sync의 삭제 범위에만 한정한다.

## 형식 조사 결과와 정확해진 수정 위치

[제한된 native 형식 영수증](./p1c-uv-retention-format-20261022.json)은 첫13개 성공 명령과
출력명 오류, 교정한 나머지12개 명령을 분리한다. 기본/root-only upgrade는 root2/dep1,
전체 --upgrade는 root2/dep2를 남겼다. 설치/update 세 경우 모두 기존 extra는 남았다.
Compile→sync의 완전 집합에서는 제외한 extra가 제거됐고, 빈 pylock은 packages=[]와
requires-python>=3.14.7로 생성됐으며 sync 뒤 installed inventory가 []였다. 이는 inert
local wheel/native 의미 확인이며 production parsing/검역·반출/삭제 승인이 아니다.

반영할 지점은 다음과 같다. C2 qualification 전 source 수정은 시작하지 않는다.

1. sandbox/uv_resolver.go의 fresh compile만 기존 target의 install/upgrade에 연결하면
   native retention과 달라진다. 기존 UpgradeInput의 불변 installed metadata를 private
   선택 환경에 data로 전달하고 선택 intent별 native delta/complete selection을 결합해야
   한다. --upgrade와 --upgrade-package를 같은 의미로 뭉개지 않는다. 원본 extra를
   유지하는 install/upgrade와 전체 selected set을 정확히 맞추는 sync를 구분한다.
2. artifact/pypi/uv_lock.go의 현재 ParseUVLock/BindWheels는 단일 primary와 nonempty를
   요구한다. 이 기존 invariant는 유지하고 complete/empty snapshot용 실제 소비자가 생길
   때 strict wire/whole archive/metadata/edge 검증의 공통 부분만 추출한다. Zero value와
   명시 empty lock은 다르며 전체 controls/digest·set Policy·owned 삭제 guard가 필요하다.
3. promotion/pypi_venv_transaction.go discoverPythonVenv는 cfg의 `version`만 읽는다.
   실제 uv venv는 `version_info`와 `uv` marker를 썼다. 기존 pip parser를 전역 완화하지
   않고 client-bound cfg adapter에서 exact interpreter/version·중복/충돌/알 수 없는
   format을 검증한 plan만 같은 transaction에 전달한다. Cfg 문자열은 실행 파일의
   신원이나 native ownership 증명이 아니다.
4. 실제 venv40 entries/31,647B에는 고정 Python link와 lib64→lib, root .lock(빈0666),
   INSTALLER=uv, 각 distribution의 빈 REQUESTED와 RECORD에 포함된 uv_cache.json이
   있었다. Transitive dep에도 REQUESTED가 있으므로 primary 판단에 사용하지 않는다.
   pypi_native_ownership.go proveNativePyPIWheelFootprint는 현재 exact pip marker와
   authentic archive footprint를 요구한다. UV marker/cache를 이름만 보고 허용하거나
   이 차이를 기존 pip 검증 우회로 만들지 않는다. Cache body/schema·console entrypoint는
   아직 조사하지 않았고, 이번 inert fixture에 .pyc는 없었다. 필요한 native enrollment의
   bounded generated-file proof는 별도 실제 검증 후 추가한다.

두 번째 검토에서도 현재 pip/PyTorch/runtime·Policy·원본 보호 계약은 유지한다.
후속 code와 native layout 검증이 없는 항목은 MISSING으로 남긴다. 두 조사의 owned
containers6/derived images2는 --no-prune로 제거했고 Host 파일/image/container/bridge/service는
전후 동일했다. Root 참조·query errors0 뒤 완료 자료16항목3,143,502B도 제거했다.

### Composition의 client/source 분리 — 기존 request 유지 우선

실제 bootstrap.Run은 PreparePresentation의 단일 source 문자열로 runtime 준비와 source
branch를 선택하고, Python에서는 installer/updater map과 pythonSourceResolver를 source별로
바인딩한다. AddPyPIInstallSourcesAndUpdates는 현재 pip/install leaf만 대상으로 한다.
새 UV 명령을 위해 runtime 초기화 조건마다 `uv` 문자열을 추가하지 않는다. CLI의 작은
typed descriptor가 command path/client key/기존 source lane/intent를 소유하고, composition은
검증된 client key의 factory를 한 번 선택하도록 한다. Source는 계속 pypi다.

처음부터 application.InstallRequest/Domain에 client enum을 넓힐 필요는 확인되지 않았다.
같은 기존 service를 UV resolver로 별도 구성하고 resolution의 정확한 runtime/control을
manifest에 보존할 수 있는지 먼저 검증한다. 실제 두 소비자의 PyPI command binding만
private helper로 공유하며 기존 public pip wrapper/옵션/에러 의미와 lazy help/doctor를 유지한다.
Factory가 없는 항목과 지원하지 않는 source/intent는 runtime/network 준비 전에 거부한다.
Descriptor에 shell/권한·정책 예외나 미래 P2..P7 placeholder를 넣지 않는다. UV runtime의
exact identity 재검증도 한 owner에서 생성해 promotion의 별도 문자열 prefix 허용을 피한다.
