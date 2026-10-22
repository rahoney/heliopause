# P1 complete UV sync — 다음 gap의 감사와 수정 지도

Source3871c6f의 C3d actual CI381을 기다리며 기존 owner를 읽은 사전 지도다.
C3d qualification 전 새 sync source는 수정하지 않는다. 이 문서는 P1의 uv pip sync 범위다. uv add/sync/lock 등 uv project 명령은 확장 계획의 P4이며 이 내부 project snapshot 재사용과 구분한다. M13-004 하나만 IN_PROGRESS이며
P1 acceptance→Go1.27.3→P2 순서를 유지한다. 아래 미완료 항목은 새 승인 관문이 아니다.

Read: [P1 계획](./p1-uv-baseline-20261021.md),
[공통 guard와 실제 native 의미](./p1c-guarded-client-baseline-20261022.md),
[native UV 인계](./p1d-native-uv-ownership-baseline-20261022.md),
[M5 계약](../../../domain-model/14-m5-pypi-pip-contract.md),
[promotion 책임](../../../architecture/04-evidence-staging-promotion.md),
[promotion 계약](../../../domain-model/09-contract-evidence-staging-promotion.md),
[client UX](../../../user-journey-cli-ia/09-package-client-quarantine.md).

## 실제 baseline — 재구현하지 않을 owner

| 범위 / 기존 symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Domain ProjectDependencySnapshot / NewDependencyFreeProjectSnapshot / ProjectDependencyUpdateFromSnapshot | YES | YES: Go/Cargo | YES: 기존 소비자 | YES: 기존 범위 | UV 소비자만 없음. Primary-free와 explicit empty를 재사용 |
| application ProjectInspectService / executeGuardedProjectUpdate / policy.M4.EvaluateProjectSet | YES | YES: Go/Cargo | YES: 기존 소비자 | YES: 기존 범위 | UV native 선택·staging·mutation 연결만 없음 |
| promotion pypiVenvTransaction / prepare / commit / rollback / persistent journal | YES | YES: pip/UV install/update | YES: CI380 managed 범위 | YES: 기존 범위 | 현재 merge semantics와 empty refusal을 sync의 명시 complete intent와 구분 |
| artifact/pypi UVLock strict parser·installed WHEEL·독립 archive/metadata/Requires-Dist binding | YES | YES: single root | YES: CI380 범위 | YES: 기존 범위 | Primary-free nonempty/empty lock 소비자와 전체 선택 검증 |
| sandbox UVResolver native tool/observer/network-policy/private volumes·independent Python data reader | YES | YES: install/update | YES: CI380 범위 | YES: 기존 범위 | Complete sync source·empty transport 및 기존 lifecycle 재사용 |
| UV complete/multi-root/explicit-empty sync end-to-end | NO | NO | NO | NO | 아래 C3e1..3 및 실제 source qualification |

현재 source의 `pypiVenvTransaction.prepare`는 len(destinations)==0을 거부하고, 새 output에
있는 distribution만 교체하며 다른 owner는 남긴다. `preparePyPIProject`와 일반 staged bundle은
단일 primary가 있는 graph/nonempty Manifest를 요구한다. 이를 완화해 empty ALLOW를 만들거나,
sync를 요청 root 한 개의 InstallRequest로 위장하지 않는다. Domain과 Application의 기존
project snapshot/set/guard 순서를 그대로 사용하고 Python wheel·출력 adapter만 연결한다.

## Native 의미를 먼저 고정

[공식 sync 설명](https://docs.astral.sh/uv/pip/compile/#syncing-an-environment)은
requirements.txt 또는 pylock.toml의 설치 집합과 환경을 맞추며, 일반 install은 무관한 기존
패키지를 남긴다. 앞 data-only probe의 compile→sync 성공은 입력 requirements의 모든 native
문법·실제 제품 삭제 승인을 뜻하지 않는다. 일반 sync 입력을 compile root 입력으로 간주해
목록에 없는 dependency를 자동 추가하지 않는다.

고정0.13.4에서 bounded exact requirements/pylock 입력의 선택·빈 입력·누락 dependency·hash와
삭제 의미를 확인한 뒤 허용 grammar를 확정한다. Include/constraint/direct/local/VCS/editable/
sdist/build/다른 index/알 수 없는 marker·option은 새 지원을 검증하기 전 명시 거부한다.
실제 native 선택과 독립 wheel metadata의 전체 dependency closure가 다르면 거부한다.
추가 패키지를 임의로 설치해 성공시키거나 unsatisfied Requires-Dist를 생략하지 않는다.

## C3e1 — primary-free format과 독립 binding

artifact/pypi/uv_lock.go의 strict wire/header/Python/source/size/hash/alternative bounds,
SelectInstalledWheelControls와 whole archive→METADATA→Simple metadata 대조 본문을 공통 private
helper로 재사용한다. 기존 ParseUVLock/BindWheels는 단일 primary/nonempty 규칙과 기존 tests를
유지한다. 별도 complete 소비자는 primary 없는 selected artifacts와 검증된 whole graph를
기존 ProjectDependencySnapshot에 bind한다. 임의 첫 package를 primary로 만들지 않는다.

Explicit packages=[]와 명시 frozen control은 zero/누락 input과 다르다. Empty인 경우에만
NewDependencyFreeProjectSnapshot을 사용하고, control bytes·정확한 UV runtime/target·native lock·
full graph digest를 함께 고정한다. Nonempty에서는 모든 selected archive/metadata와 dependency
edge·constraint를 검증한다. Disconnected roots도 누락 없이 포함하되 duplicate project/version,
missing wheel/dependency, unknown fields와 budget 초과를 거부한다.

## C3e2 — 기존 native lifecycle에 complete source 연결

sandbox/uv_resolver.go의 tool 검증·endpoint/network-policy·observer·private output·attachment/
cleanup을 재사용한다. 이미 존재하는 install/update native selection의 의미는 유지한다.
Frozen source bytes를 private fixed path로 전달하고 실제 UV sync 선택을 얻는다. 원본 source
파일과 target cfg/METADATA/ownership은 선택 전에 guard하고 선택·검사·반입까지 다시 확인한다.
Host resolver/target Python을 실행하거나 artifact 출력을 완료 authority로 사용하지 않는다.

uv_resolver_reader.go의 independent metadata/whole archive transport도 재사용한다. 현재
withPythonResolverSession은 source 확인에 ArtifactReference를 쓰므로 complete consumer는
private session owner에 source-bound 진입을 추가하는 방향을 검토한다. Dummy reference/primary를
만들거나 정책 경계를 생략하지 않는다. Empty는 임의 네트워크 성공 대신 frozen source와 native
빈 lock·관측/종료·guard의 명시 증거를 요구한다. 기존 single-root 범위와 default 한도는 유지한다.

## C3e3 — exact complete publication / CLI

1. 기존 application guarded-project-update 순서로 guard→native selection→전체 entry inspection/
   Evidence/entry Policy→set Policy→staging→commit→close를 연결한다. Empty도 실제 snapshot/set
   Policy와 guard를 거치며 가짜 artifact/check/성공 Evidence를 만들지 않는다.
2. Python project cache adapter는 기존 LocalStaging.stageArtifact·digest copy·recorded Evidence와
   exact receipt binding을 재사용한다. Go/Cargo cache layout을 Python에 강제로 사용하지 않는다.
   Artifact filename·digest·size와 전체 controls/runtime/graph·set approval을 receipt에 bind한다.
3. Offline Python materialization/validatedPyPIDestinations와 기존 transaction을 재사용한다.
   Complete intent에서만 제외한 모든 proven old owners를 obsolete로 계산한다. Empty commit은
   해당 intent와 explicit approved empty set에만 허용한다. 일반 pip/UV install/update의 merge/
   empty refusal은 유지한다. Journal은 이미 Current/Desired/file intent와 원본을 보존하므로
   삭제·중간 실패·rollback·복구에 같은 owner를 사용한다.
4. Native --adopt는 새 선택에서 제외되어 삭제될 distribution까지 모두 공식 이전 wheel로
   증명해야 한다. 선택된 graph만 old proof 하는 기존 경로를 그대로 사용해 unselected native
   extras를 지우지 않는다. Native generated scripts/compiled caches 등 현재 C3d의 refusal을
   유지하고, 인증되지 않은 파일·사용자 데이터·cfg/UV bootstrap 파일은 제거하지 않는다.
5. CLI는 같은 compiled client descriptor/composition에 sync operation을 연결한다. Runtime
   초기화 전 source/args/flags 검증과 lazy help·7개 locale의 기존 fallback을 유지한다. SourceID는
   pypi이며 미래 client enum/plugin을 추가하지 않는다. 결과는 기존 project report를 재사용해
   complete selected count/explicit empty·검사/Policy·삭제/반입 완료·오류를 구분한다.

## 검증과 두 번째 검토

Canonical format/Quick/security/docs/workflow를 먼저 실행하고 영향받은 parser·Application·
Policy·Python transaction·CLI/Bootstrap의 의미 있는 정상/부정 및 Race를 확인한다. 실제 native
source에서 two-root+dependency·기존 extra 삭제·version update·explicit empty·same-set fresh
Evidence·소스/target 변조·native ownership 거부·partial publication rollback을 연결한다.
Shared-owner conflict, disk/resource/unknown query·취소/cleanup의 최초 오류를 보존한다.
같은 source의 Required/Linux·기존 pip/native/public consumers/macOS와 exact cleanup을 확인한다.
기존 설치·업데이트의 single-node 성공을 complete sync나 whole P1 qualification으로 확대하지 않는다.

두 번째 검토: 삭제 권한은 native exit/empty JSON/RECORD filename이나 선택 graph 부재에서
생기지 않는다. Original ownership proof와 명시 complete intent, 전체 선택의 검역·set Policy,
frozen control/guard와 transaction이 함께 있어야 한다. Source control의 변조·symlink/alias·
누락/empty 구분, unselected native owners의 proof와 shared owners까지 검토한다. 기존 구현을
대체하는 새 engine·generic framework·future placeholder는 만들지 않는다. 고정 tool/image/
runtime/resource 한도와 지원 범위는 이 지도로 바꾸지 않는다. 완료한 시험은 작은 재현 원본을
보존하고 exact owned 자원·마지막 참조/성공한 부재 확인 뒤 정리한다.
