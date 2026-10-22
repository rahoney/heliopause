# P1c C3 — guarded client 연결의 baseline audit

최초 기준 sourceab54d4a, 후속 조사 기준35c0f2d/문서948833a. C2 public UV/HTTP는
[CI376](./p1c-public-ci376-20261022.json)의 실제 단일/다중 graph와10SUCCESS로 qualified다.
이 문서는 다음 acceptance gap의 감사·수정 지도다. C3a의 사전 target guard부터 구현을
시작한다. M13-004/P1만 IN_PROGRESS이고 P2..P7로 이동하지 않는다.

Read: [P1c 지도](./p1c-uv-resolver-baseline-20261022.md),
[C2 producer 지도](./p1c-uv-native-producer-baseline-20261022.md),
[Client 계획](../../20-package-client-quarantine-plan.md),
[Verified Set](../../../domain-model/05-dependency-verified-set.md).

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Native UV graph — sandbox/uv_resolver.go + uv_resolver_reader.go | YES | YES(internal/CI), NO(app) | YES(CI376 public scope) | YES(single-root producer) | 기존 구현 재사용; app/guarded client 연결은 아래 별도 gap |
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

## C3a — 신규 설치의 사전 target guard / 현재 구현 단위

Fresh compile은 기존 설치의 retention을 표현하지 못하므로 첫 내부 설치 경로는 배포된
distribution이 없는 기존 venv로 제한한다. 이 제한은 후속 installed-aware 선택을 구현할
때 해소하며, 전체 UV install/update 지원 완료가 아니다. CLI는 실제 backend qualification
후 연결한다. 단순히 resolver를 기존 installer에 꽂아 nonempty 환경을 fresh solve하지 않는다.

| 기존/신규 위치 | 감사 | 정확한 변경과 검증 |
| --- | --- | --- |
| ports/update.go InstallUpdateSession | 기존 YES/WIRED/QUALIFIED; 신규 일반 install preparation NO | 공통 Resolver/Promotion/Close 수명만 InstallSession으로 추출하고 기존 update는 Reference를 추가로 요구한다. BeginInstall의 실제 UV 준비 port만 추가; 기존 update guard의 타입·의미 유지 |
| application/install_update.go Install | 기존 YES/WIRED/QUALIFIED; 일반 prepared install NO | per-request copy/rebind 부분만 private helper로 공유한다. 기존 update 요청·content session·실패 code/cleanup 순서는 유지하고 일반 INSTALL wrapper를 추가한다. Operation을 UPDATE로 위장하지 않음 |
| promotion/pypi_venv_transaction.go discoverPythonVenv | 기존 YES/WIRED/QUALIFIED; version_info adapter NO | bounded cfg 읽기/디렉터리·owner 검증은 공유, 기존 version parser 의미 보존. UV 준비는 version_info/CPython/include-system-site-packages=false와 선언된 Python target을 검증. uv 생성 버전은 history data이며 현재 UV 실행 pin 인증으로 쓰지 않음 |
| promotion/uv_install.go (신규 adapter) | NO/NO/NO/NO | 기존 venv transaction을 resolve 전에 획득한다. installed/legacy metadata와 기존 managed payload가 있으면 거부. cfg/원본 ledger/metadata inventory를 resolve 전후와 promote 직전에 재확인하고 기존 promoteHeldVenv를 사용 |
| sandbox/uv_resolver_reader.go runtime identity | 기존 YES/WIRED/QUALIFIED | 정확한 고정 identity 생성만 기존 sandbox owner의 함수로 공유해 준비 session이 결과를 교차 검증한다. Broad prefix·fake pip identity는 금지 |

UV session은 동일 reference/context에서 한 번만 resolve하고 완전 graph/정확한 UV runtime을
검증한 뒤에만 promote를 허용한다. 기존 staged/bundle·selected graph binding과 file ownership/
journal/rollback을 재사용한다. User target의 interpreter/.pth/module을 sandbox로 복사하거나
실행하지 않는다. 신규 생성 marker를 독립 인증 없이 native ownership으로 채택하지 않는다.

두 번째 검토: 이는 기존 구현의 교체가 아니라 두 실제 준비 수명의 공유와 새 adapter다.
Update의 Reference marker를 없애거나 모든 일반 install을 update session으로 취급하지 않는다.
UV ordinary session에는 update Reference marker를 주지 않는다. 표준 venv/UV cfg, 중복·충돌·
잘못된 target, nonempty/legacy metadata, resolve 전후 원본 변경, wrong runtime/ref/context,
반복 resolve/promote·staging 불일치·취소·Close 실패와 기존 Update 전체 회귀를 검사한다.
Canonical format/Quick/security/docs, 관련 application/promotion/sandbox Race와 이후 실제
UV→entry/set 검사→staging→target 반입을 연결하며 source cache/완료 출력 정리를 유지한다.

구현 전후 대조에서 transaction 시작 자체도 기존 discoverPythonVenv를 다시 호출함을
확인했다. 기존 wrapper는 그대로 두고 private begin-with-discovery를 공유해 UV cfg를 잠금
획득 뒤에도 검증한다. 신규 session은 고정 Python target과 version/version_info의 충돌을
검사하며 uv 생성 버전은 bounded history로만 처리한다. 현재 solver pin과 일치해야 한다는
조건을 cfg history에 잘못 부여해 향후 도구 patch 시 기존 venv를 막지 않는다.

Selected graph뿐 아니라 Manifest의 resolver runtime/lock digest·target path/requires_new·
전체 controls도 같아야 한다. 기존 Update의 검증 body를 실제 두 소비자가 쓰는 private
verifyPythonSelectionManifest로 공유하고 기존 실패 메시지를 유지한다. Runtime/digest/
target/control substitution과 staged tamper는 materializer 실행 전에 거부해야 한다.

### C3a 내부 준비 단위의 로컬 checkpoint

[로컬 검증](./p1c-prepared-install-local-20261022.json)은 format/Quick/security/docs/workflow와
application 전체 Race9.153s·관련 promotion37.885s/sandbox21.436s를 통과했다. 기존 Update/
content session과 ordinary INSTALL의 intent/원본 guard·공유 Manifest binding, 취소·변조·
부분 게시 rollback을 확인했다. 첫 Staticcheck의 사용하지 않은 fixture 변수만 교정했고
억제 규칙은 추가하지 않았다. 모든 검사 종료 뒤 task TMPDIR의 항목0도 확인했다.

현재 IMPLEMENTED=YES / WIRED=YES(internal library), NO(composition/CLI) /
QUALIFIED=YES(local scoped only) / ACCEPTANCE_CLOSED=NO다. Fixture materializer 성공을
실제 UV→검사→사용자 target 반입으로 표시하지 않는다. 다음은 C3b의 typed client/source
descriptor와 bootstrap factory에 실제 준비 서비스를 연결하고 native end-to-end를 검증하는
것이다. Nonempty install/upgrade·sync의 남은 gap은 계속 별도로 유지한다.

## C3b — typed client/source와 실제 CLI 구성 연결

기준 d89c934. C2는 qualified이고 C3a 내부 준비 단위는 local checks/Race를 통과했다.
이번 범위는 실제 `helox uv pip install`의 빈 기존 venv 설치 연결이며 nonempty/upgrade/sync를
준비되지 않은 fresh solver로 처리하지 않는다. 기존 구현은 모두 재사용한다.

| 수정 위치 | 변경 방법 | 검증 |
| --- | --- | --- |
| cli/package_client.go (현재 두 Python client의 신규 작은 descriptor) | 실제 pip/UV의 ID·command path·기존 composition route·validator/reference parser·EN/KO 도움말을 한 owner에 둔다. Domain source는 pypi, 기존 pip route는 유지 | 직접/leading global flags/`--`·도움말·잘못된 spec/target/반복 target·미지원 flags의 runtime 이전 거부 |
| cli/root.go initializeCommandTree / AddPyPIInstallSourcesAndUpdates | 기존 pip static 구성과 install binding만 두 실제 client용 helper로 공유; 기존 public wrapper 유지. UV는 target/global flags만 노출, bare/exact `==`를 canonical reference로 정규화 | 기존 pip/PyTorch install/update 전체 회귀, UV fake injected installer의 정확한 INSTALL/source/context |
| cli/presentation.go PreparePresentation | 실제 선택 leaf의 descriptor에서 client ID와 기존 route를 읽는다. Bootstrap의 모든 source 조건에 uv 문자열을 추가하지 않음 | help/settings의 lazy composition·source/client 독립성·JSON/KO |
| bootstrap/python_client.go / bootstrap.go Python 구성 | 한 registry에서 실제 resolver/검증된 installer binding을 고른다. Pip는 기존 named source·derivation·update/adoption 그대로; UV는 실제 UV resolver·C3a prepared installer와 wheel-only 구성. 미지원 OS는 runtime 준비 전에 거부 | 잘못된 client/platform 거부, 실제 bootstrap CLI native UV result/Manifest runtime 확인 |
| presentation_catalog.go / human_report.go | descriptor의 새 EN/KO 설명과 일반 설치 준비/정리 오류 설명 연결. 기존 5개 언어 fallback 유지 | 7개 locale help/구조화 JSON, EN/KO 설명 완전성 |
| bootstrap/uv_install_integration_test.go (신규 실제 소비자) | 기존 NPM inspect처럼 bootstrap.Run을 실제 installed bootstrap-test client 안에서 실행한다. UV cfg fixture→public idna native resolve→entry/set 검사→staging→실제 파일 반입·원본 보존 검증 | nonempty 두 번째 요청 거부·기존 payload/소유 기록 불변·추가 Evidence 부재 |
| 기존 CI lifecycle / scripts/check UV 명령 validator | bootstrap-test 설치·policy helper 재바인딩 뒤 새 opt-in 실제 test를 조건 없이 추가. 새 job/권한/인증 우회 없음 | command 누락/conditional 회귀와 실제 source CI·helper drain/복원 |

두 번째 검토: 실제 production 소비자가 있는 함수만 공유하고 미래 client entry/flag/설정을
미리 만들지 않는다. Native `name==version`은 typed canonical reference로만 바꾸고 native
argv를 전달하지 않는다. Source/runtime/Policy·staging·rollback 의미와 기존 pip 동작을 유지한다.
UV의 실제 materializer는 현재 기존 pinned offline installer이며 solver identity는 UV로 기록한다.
사용자 target Python/.pth를 실행하지 않는다. CLI 시험의 venv는 제한된 cfg/디렉터리 fixture이며
user interpreter 실행 성공이나 native adoption으로 확대하지 않는다.

Canonical format/Quick/security/docs/workflow, 관련 CLI/bootstrap/application/promotion Race와
actual native CI까지 확인한다. 신규 설치만 통과하면 그 범위만 표시하며 P1 전체와 후속
installed-aware update/sync의 MISSING을 보존한다. 첫 실제 실패와 완료 시험 자료 정리를 남긴다.

### C3b 로컬 연결 결과 — native qualification 대기

Typed descriptor는 현재 pip/UV 두 client만 등록하며 source는 기존 PyPI로 유지한다.
Root command annotation의 선택은 표시 writer와 독립이고 bootstrap의 기존 source lane과
client factory를 연결한다. Pip/PyTorch의 resolver·sdist·named source·upgrade/adoption은
그대로 재사용한다. UV는 실제 UV resolver와 C3a prepared INSTALL/held venv transaction을
사용하며 pip resolver fallback이나 fresh solver 기반 nonempty 설치를 제공하지 않는다.

[로컬 영수증](./p1c-guarded-cli-local-20261022.json)의 canonical format/Quick/security/docs/
workflow와 CLI/bootstrap/application/promotion/checker 전체 Race는 PASS다. 직접/leading
flags/`--`·bare/exact 입력 정규화·source/client 독립성·미지원 spec/flags/반복 target의 사전
거부와 7개 locale 도움말을 검사했다. 기존 pip/PyTorch 회귀도 같은 Quick/Race에 포함된다.
실제 authenticated bootstrap-test 안의 UV→검사→staging→target 반입·manifest UV 신원·
nonempty 반복 거부/원본·Evidence 불변을 검증할 opt-in test를 기존 무조건 CI lifecycle에
연결했다. 이 actual native test는 아직 NOT_RUN이며 로컬 default test의 opt-in skip은
native 성공 증거가 아니다.

이 좁은 CLI 경로는 `IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO`다. MISSING은 같은 후보의 actual native CLI/기존 Linux 소비자와
native macOS를 포함한 Required CI다. Installed-aware 일반 설치·upgrade·native ownership·
complete/multi-root/empty sync는 계속 다음 C3 gap이고 P1 전체 acceptance는 열려 있다.
완료한 로컬 검사의 TMPDIR는 비어 있으며 다음 C3 검증에 쓸 SDK/source/quality/runtime만
보존한다. 별도 native daemon/image/volume이나 Host 설치·서비스 변경은 만들지 않았다.

## C3c 착수 지도 — installed-aware 선택 / C3b CI 대기 중 검토

기준8ec8de7. C3b의 actual CI377은 진행 중이고 이 절은 다음 gap의 source 감사다.
기존 구현을 대체하거나 C3b 결과를 선행 확정하지 않는다. 다음 code 착수는 C3b 실제
결과를 먼저 확인하고, 아래 전송 방식의 미확정 의미를 bounded native 입력으로 확인한다.

| 기존 책임 / symbol | IMPLEMENTED / WIRED / QUALIFIED / ACCEPTANCE_CLOSED | 실제 남은 일 |
| --- | --- | --- |
| upgrade_input.go NewUpgradeInput / MetadataFiles / Control | YES / YES(pip) / YES(existing) / YES(existing) | UV에도 필요한 bounded inert METADATA·원본 digest 의미 재사용; native 선택의 intent는 명시적으로 별도 binding |
| upgrade_selection.go UpgradeProposal / proposeUpgrade / ValidateFinal | YES / YES(pip/named) / YES(existing) / YES(existing) | UV preview change의 typed name/version/action을 archive-authenticated Candidate와 결합할 실제 adapter; fake pip report 금지 |
| pypi_update.go original metadata snapshot / managedPyPIInventory / verify / Promote | YES / YES(pip) / YES(existing) / YES(existing) | UV cfg 아래 같은 원본·ledger·retired-directory guard와 graph/manifest 검증 재사용, UV controls만 좁은 소비자로 분리 |
| uv_resolver.go resolveNativeSelection / uv_resolver_reader.go bindUVNativeSelection | YES / YES(fresh) / YES(C2) / YES(C2) | native installed-aware delta와 frozen original 입력 전송, 선택한 exact version으로 complete single-root closure 재인증 |
| uv_install.go BeginInstall / verifyOriginal | YES / YES(C3b) / NO(native pending) / NO | 비어 있지 않은 환경의 사전 inventory/guard 및 native original-input binding; fresh solve 거부는 준비 전까지 유지 |
| pypi_native_ownership.go oldWheel / proveNativePyPIWheelFootprint | YES / YES(pip) / YES(existing) / YES(existing) | UV uv_cache.json schema/body·generated entrypoint/bytecode layout의 독립 proof와 explicit enrollment; marker 단순 교체 금지 |

### Native 선택 전송의 미확정 지점과 유한 조사

기존 UpgradeInput.MetadataFiles는 사용자 Python·module·.pth 없이 bounded METADATA를
전달할 수 있다. 하지만 UV가 METADATA만 있는 private target을 실제 installed state로
인식한다는 증거는 아직 없다. 앞 inert wheel retention 조사에서는 full private target을
사용했으므로 그 성공을 metadata-only 전송 성공으로 전용하지 않는다.

같은 인증된 UV/Python pin과 이전 inert root/dep/extra 관계를 사용하되 새 입력은 다음
미확정 지점만 조사한다. Host/user venv를 수정하거나 사용자 interpreter를 실행하지 않는다.

1. 기존 private installed state에서 METADATA만 전달한 target과 METADATA+WHEEL target을
   나누어 UV list 및 dry-run install의 실제 인식 결과를 확인한다. 필요한 최소 native
   controls가 확정되기 전 RECORD·module·entrypoint나 target Python을 임의로 복사하지 않는다.
2. 새 root가 dep1/dep2 모두 허용할 때 일반 install, root-only upgrade, 전체 upgrade의
   dry-run JSON을 비교한다. 기존 full-target retention 성공은 반복하지 않으며 이번
   metadata-only 결과가 앞에서 관측한 retention과 같은지를 확인한다. Unsupported flag/
   first fault는 원본 exit/stdout/stderr로 남기고 후속 큰 입력으로 확대하지 않는다.
3. same-version/no-op의 empty changes, replacement의 old uninstalled→new installed
   action pair, 신규 root/dep 선택을 구분한다. Preview JSON에는 URL/hash/edge가 없으므로
   선택 데이터일 뿐이며 전체 wheel/metadata·public source 검증을 대체하지 않는다.

이 조사는 readonly/unprivileged/network-none runc·inert wheels의 데이터 형식 확인이다.
성공하더라도 gVisor/admission/typed policy·public HTTP·검역/승격 qualification은 별도다.
Owned exact container/image만 정리하고 Host 전후 상태·참조 조회 실패를 부재로 바꾸지 않는다.

### 확인 후 수정할 위치 / 검증

- artifact/pypi의 UV 전용 parser에서 schema.preview·dry_run=true·bounded change list와
  known action을 strict하게 검증한다. 같은 name의 정상 old/new action pair와 중복 action을
  구분하고, unknown schema/action·원본에 없는 uninstall·version mismatch·불완전 pair를
  거부한다. Empty delta는 명시적 no-op 선택이며 빈 verified graph/자동 ALLOW가 아니다.
- 기존 UpgradeInput의 원본 installed version/metadata와 native delta를 연결한다. 모든
  selected version을 fixed constraints로 만들더라도 unrelated installed package를 가짜
  graph root로 만들지 않는다. Retained dependency의 원본 requirement와 changed version의
  fresh authenticated metadata를 교차 확인한다. 실제 Candidate가 없는 native JSON으로
  인증된 Candidate나 pip report를 조립하지 않는다.
- sandbox의 기존 UV resource/observer/policy 수명을 재사용하고 request와 original metadata를
  fixed bounded stdin transport로 전달한다. Interpreter는 기존 고정 image의 Python이다.
  Native delta 뒤 exact selected closure는 기존 pylock/WHEEL/whole archive/public metadata
  binder로 확인하고 UV identity·original input·delta·lock을 같은 selection controls로 묶는다.
  단일 primary 규칙은 유지하며 multi-root/empty sync는 기존 project snapshot 경로의 다음 gap이다.
- promotion에서는 기존 venv transaction/metadata·ownership 확인을 재사용한다. UV 결과만
  기존 pip delta/report parser에 통과시키거나 runtime prefix를 완화하지 않는다. Fresh install은
  INSTALL, --upgrade는 기존 UPDATE intent를 보존하고 no-op도 entry/set 검사 후 commit한다.
  Native enrollment가 준비되기 전 unmanaged selected package는 기존 explicit refusal을 유지한다.
- CLI는 실제 backend가 준비된 intent만 descriptor에 노출한다. UV install의 --upgrade와
  --upgrade-package 의미를 같은 동작으로 합치지 않고 fixed native 의미에 대응한다. 기존 pip/
  PyTorch install/update·named source·adoption 및 UV fresh install의 성공/거부를 함께 검증한다.

두 번째 검토: 기존 pip Update 서비스 전체를 재작성할 필요는 확인되지 않았다. 두 실제
소비자가 공유할 original-state snapshot/held transaction·Manifest·rollback 책임만 추출하고,
UV 선택 controls와 client intent adapter를 그 owner에 둔다. Native parser/전송 코드의 세부
형식은 위 actual probe 뒤 확정한다. 원본 변경·unrelated retention·changed dependency·same
version/no-op·구형 wheel 제거·부분 게시 rollback·동시 접근·취소/cleanup·unknown control/
runtime/source·native ownership 변조와 실제 native public/CLI를 acceptance에 포함한다.

### Metadata-only native 형식 조사 완료

[실제14개 입력 영수증](./p1c-uv-metadata-only-format-20261022.json)은 같은 UV0.13.4/
Python3.14.7의 runc/network-none/read-only/UID1000/no-cap 안에서 METADATA만 있는
private target과 METADATA+WHEEL target 모두 원래 세 distribution을 인식함을 확인했다.
두 형식 모두 dry-run 일반/root-only upgrade는 root1→root2만 선택하고 dep1을 유지했고,
전체 --upgrade는 root와 dep 모두2를 선택했다. Same-version은 dry_run=true/changes=[]다.
모든 원본 metadata bytes는 불변이었다. 빈 auxiliary .lock은 비교에서 구분했다.

위 전송 미확정 지점은 이 data-only 범위에서 확인됐다. Production은 기존
UpgradeInput.MetadataFiles의 bounded inert METADATA만 전달하는 방향으로 구체화하며
사용자 module/RECORD/entrypoint/.pth/Python을 선택 환경에 복사할 이유는 확인되지 않았다.
독립 whole archive/public metadata·정확한 native runtime/controls와 guarded transaction은
여전히 필요하다. Native gVisor/public/Policy·actual update qualification으로 확대하지 않는다.

Exact owned containers3와 파생 image1을 --no-prune로 제거했고 Host 파일/image/container/
bridge/service는 전후 동일했다. Root references0/query errors0 뒤 완료 작업 디렉터리와
일회성 script3항목10,888B를 삭제했다. 작은 원본 결과/재현 source·정리 증거만 보존했다.

## C3b 실제 CI377 qualification 종료

[최종 증거](./p1c-guarded-cli-ci377-20261022.json)는 source8ec8de7와 actual checkout334918f의
동일 전체 tree5ffc240 및 Required 포함10SUCCESS를 확인했다. 실제 installed authenticated
bootstrap-test 안에서 UV 선택→전체 검사/ALLOW→staging→기존 offline installer→target 반입이
25.50s에 성공했다. Manifest의 정확한 UV runtime, idna 파일·소유 기록과 cfg 불변을 확인했고,
nonempty 두 번째 요청은 선택 이전 거부되며 target/Evidence가 불변이었다. Public UV20.05s,
Linux50/CPU283.72s/helper5wait0/final confirmed cleanup, native macOS와 Vulnerability도 PASS다.

C3b의 빈 기존 venv INSTALL은 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED 모두
YES다. 앞 절의 native NOT_RUN/NO는 이 결과로 갱신한다. Installed-aware 설치/upgrade·native
ownership·complete/multi-root/empty sync/P1 전체는 여전히 MISSING이며 C3c 착수 지도를 따른다.
현재1.26.10 기준 P1을 닫은 뒤 [Go1.27.3 전환 계획](./go1273-migration-baseline-20261022.md)을
P2 전에 수행한다. 순서·상태의 단일 owner는 Queue다.
