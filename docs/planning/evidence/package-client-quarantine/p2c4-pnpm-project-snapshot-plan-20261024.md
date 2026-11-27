# P2c4 — PNPM complete-project snapshot

2026-10-24, baseline `0f8ddccafed3e75cb0c4c96805a8b41a776519e7`.
[P2c 지도](./p2c-pnpm-selection-plan-20261024.md)와
[설치 동작 사실](./p2c3-pnpm-install-action-plan-20261024.md)에 이어 같은 M13-004/P2에서 수행한다.
기존 Domain ProjectDependencySnapshot/ProjectInspectService를 대체하지 않는다.

## 1. Source 작성 전 baseline audit

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| PNPM lock full peer wiring — `artifact/npm/pnpm_lock.go` ParsePNPMLock | YES | YES | NO | NO | 현 parser 재사용; expected client version을 immutable selection에 보존; 기존 로컬 검증은 P2c 결과 참조 |
| unique archive 공개 source binding — `pnpm_selection.go` BindPNPMLock | YES | YES | NO | NO | 이미 bound된 후보를 재resolve/재구현하지 않음; 전체 P2 qualification은 미완료 |
| complete project Domain — `core/domain/project_snapshot.go`, `project_verified_set.go` | YES | YES | YES | YES | PNPM adapter가 controls/runtime/full peer wiring을 결합한 snapshot 생산만 없음 |
| archive action → rootless M4 — P2c3 source0f8 | YES | YES | NO | NO | 새 snapshot을 기존 실제 검사 consumer에 연결; P2c3 로컬 검증과 전체 P2 qualification 구분 |
| native selection → PNPM snapshot | NO | NO | NO | NO | 아래 adapter method와 pinned producer helper, 실제 소비자 검증 |
| PNPM layout/records/transaction/CLI | NO | NO | NO | NO | 본 snapshot 완료 후 별도 착수 지도 필요 |

## 2. 수정 위치와 방법

- `artifact/npm/pnpm_lock.go`: ParsePNPMLock가 이미 검증한 expectedClientVersion을
  private 필드에 보존한다. 새 pin이나 runtimeidentity import는 만들지 않는다.
- 같은 package의 `pnpm_project_snapshot.go`: source-bound `PNPMSelection`의
  `ProjectSnapshot(InstallContext, []ProjectControlFile, runtimeIdentity)`를 추가한다.
  정확히 present/nonempty package.json·pnpm-lock.yaml 두 control만 받는다. 같은 version으로
  다시 parse하여 full normalized roots/packages/snapshots가 bound selection과 같은지 확인한다.
  다른 manifest constraints·lock/peer wiring·누락/중복/extra/absent control을 거부한다.
- Adapter 내부 deterministic JSON은 schema, target/mode, client version, producer runtime,
  두 control의 raw digest, 독립 source-bound artifact identity/URL/SRI, full roots/package
  metadata/peer-qualified snapshots·실제 dependency edges를 묶는다. Hash를 기존 Domain
  opaque graphDigest로 전달한다. Map order는 JSON의 key 정렬, 목록은 기존 parser의 정렬을
  재사용한다. 같은 controls의 인수 순서만 바뀌면 같은 digest다. Raw control bytes가 달라지면
  의미가 같아도 다른 frozen snapshot이다.
- Snapshot dependencies는 정확한 unique archive 목록이다. Peer instance는 결코 합치지 않고
  별도 full lock 구조와 digest로 남긴다. 명시 empty lock만 기존 NewDependencyFreeProjectSnapshot,
  nonempty는 NewProjectDependencySnapshot을 사용하며 arbitrary primary를 만들지 않는다.
- `sandbox/pnpm_resolver.go`: generated PNPM version/native SHA/Node image/runtime-lock SHA로
  `PNPMResolverRuntimeIdentity()`를 만든다. 현재 실제 producer와 독립 source binding을
  잇는 `PNPMNativeSelection.BindProjectSnapshot(ctx, source, install)` helper가 고정 identity와
  자기 immutable controls만 위 adapter에 전달한다. 이는 unapproved selection이며 Policy나
  cache/promotion 권한이 아니다. Artifact adapter가 runtimeidentity를 import하지 않는다.
- Native actual fixture는 이 helper를 소비하도록 좁게 연결한다. DependencyFree/고유 archive
  수·peer instance/controls/runtime digest를 대조한다. 기존 command/admission/cleanup은 불변이다.

허용 install mode는 현재 npm source의 NEW_TARGET/NPM_PROJECT뿐이다. 다른 ecosystem
mutation mode/빈 runtime/control byte/제어문자 runtime은 source adapter에서 거부한다.
Context나 runtime을 외부 native 출력에서 만들지 않는다. 실제 immutable producer의 고정
runtime을 composition이 전달하며 version 문자열만으로 실행 신원을 승인하지 않는다.

## 3. 검증 계획

정상 single/transitive/multiple-root/peer·같은 archive의 복수 peer instance·explicit empty,
control 인수 순서와 getter copy를 확인한다. Zero/unbound selection, 다른 manifest/lock,
peer edge drift, extra/duplicate/missing/absent controls, invalid mode/runtime을 거부한다.
Target/mode/runtime/version/constraints/control bytes가 달라지면 digest가 달라야 한다.
Runtime generated tuple와 actual helper의 source binding/checkpoint를 연결한다.

기존 public native8 fixture의 single/transitive/peer/selected-update 네 요청에서만 같은
snapshot을 기존 ProjectInspectService+public archive Acquire/SRI/static/dynamic+Store+
entry/set Policy까지 연결한다. 매 entry의 required3검사·Evidence4참조·정확한 action fact와
ALLOW를 확인하며 다른 네 native 요청의 결과와 구분한다. Installed project/promotion은
실행하지 않는다. 기존 CLI bootstrap의 npm composite/production Linux backend를 재사용하고
새 검사기·가짜 dynamic 완료를 만들지 않는다. 원격 전체 CI와도 구분한다.
Canonical format/quick/security/docs와 npm/sandbox 관련 전체 Race를 적용하고 작은 helper별
remote push는 하지 않는다. CI402/head940의14:57:13KST 실제 타이머는 그대로 유지한다.

## 4. 두 번째 검토 — source 작성 전

기존 complete Domain과 unique archive 검역을 재사용하면서 peer wiring을 opaque digest에서
잃지 않는지 대조했다. Native lock을 다시 파싱할 때 manifest까지 같은 version으로 검증하며
다른 root constraints를 bound artifact 목록만으로 승인하는 방법은 제외했다. 단순 raw lock
hash만으로 runtime/target/독립 source를 대체하지 않는다. Empty는 parsed+source-bound일 때만
명시 가능하다. 기존 npm graph/UV/다른 source Domain invariant는 변경하지 않는다.

Shared project manifest는 unique archive nodes/edges를 받으므로 PNPM의 복수 peer instance
layout·instance provenance/SBOM를 자동으로 모두 표현한다고 가정하지 않는다. 다음 records/
virtual-store 검증에서 full peer lock을 함께 결합할 정확한 표현과 symlink/bin/ownership을
감사한다. 이 남은 acceptance를 잊지 않고 Queue에 유지한다.

후속 caller 대조에서 P2c3 문서의 root project scripts 사전 거부 설명은 실제와 달랐다.
`samePNPMManifestMetadata`는 scripts를 포함한 원본 metadata를 보존하고 native resolver는
ignore-scripts다. Archive fact와 사용자 root metadata를 혼동하지 않고 P2c3 설명을
정정했다. 다음 layout/promotion에는 원래 scripts 보존·고정 ignore-scripts·실행0과
native 출력의 scripts 변조 거부를 포함한다. 현 snapshot은 두 원본 control bytes를 묶는다.

## 5. 실제 연결의 첫 거부와 남은 gap

첫 canonical Quick은 negative fixture의 직접 nil Context 호출을 pinned Staticcheck
SA1012가 거부했다. Invalid input table로 같은 거부를 검사한 다음 Quick117.244s/
security6.137s/docs.215s는 PASS다. 새 tagged actual fixture는 별도로 build하여 실행했다.

Actual first30.890s에서 single native→snapshot→public archive/SRI/static/dynamic/Store/
M3·M4 ALLOW는 PASS다. Transitive native selection/snapshot은 성공했지만 complete Policy가
거부하여 뒤 peer/add/frozen/update 요청은 미실행이다. Host12파일/images/networks/bridge/
service/firewall은 동일하고 helper wait0/CONFIRMED_STOPPED·original1을 보존했다.

원래 정상343과 `Backend.Execute`/`npmLifecycleCommand`/npm DynamicInspector는 동일하다.
현재 backend는 offline npm install에 target archive 하나만 소개하며 npm composite에는
InspectProject/InspectGraph가 없다. 따라서 PyPI의 closure-aware batch capability가 npm에도
있다고 가정하지 않는다. 이는 새 multi-package acceptance에서 확인한 준비 gap이며
현재 PNPM parser가 이미 정상인 기존 npm 구현을 망가뜨렸다는 증거가 아니다.

첫 진단 wrapper .729s는 완료한 private overlay가 남아 namespace preflight에서 멈췄다.
이때 복사된 actual.log/result.json은 이전 시험 자료이며 진단 실행으로 세지 않는다.
scope 정정 영수증을 보존하고 기존 qualified operator와 같은 exact overlay 정리57,825,162B를
all-UID refs/query errors0 뒤 수행했다. 제품 판정·observer·한도는 바꾸지 않았다.

Fresh bounded 진단30.264s/actual27.93s는 single ALLOW를 다시 확인하고 transitive의
M4_DEPENDENCY_REVIEW_REQUIRED를 확인했다. is-number6은 필수3검사 완료/ALLOW이고
is-odd3은 integrity/static 완료·action NOT_REQUIRED이지만 dynamic INCOMPLETE /
M3_DYNAMIC_EXECUTION_FAILED → M3_REQUIRED_CHECK_INCOMPLETE다. Runtime stderr의
구체 오류는 현재 report가 보존하지 않았으므로 ENOTCACHED를 관측했다고 단정하지 않는다.
원본 `p2c4-pnpm-project-{quarantine,fresh-policy-diagnostic}`가 scope/최초 거부를 소유한다.

Snapshot parser/source 단위의 성공과 transitive 동적 검역의 미완료를 분리한다.
다음은 full frozen dependency/peer context를 npm dynamic consumer에 공급할 추가 착수 감사다.
정적 검사 건너뛰기·missing dependency 자동 다운로드·incomplete를 ALLOW로 변경·가짜 primary는
금지한다. Closure/layout 표현과 정확한 native offline fixture 의미를 먼저 설계·검토하고
그 뒤 실제 같은 입력의 개선 여부를 검증한다. 현재 P2 qualification/acceptance는 NO다.

최종 canonical Quick31.397s/security4.504s/docs.225s, npm/sandbox 전체 Race23.050s와
실제 public metadata→single/peer snapshot1.698s는 PASS다. [로컬 결과·남은 거부](./p2c4-pnpm-project-snapshot-local-20261024.json)가
단위별 입력과 성공/미완료를 분리한다. 마지막 진단 소비 뒤 all-UID refs/query errors0,
container0/Host 보존을 확인하고 전용 runtime12파일188,106,872B도 삭제했다.
작은 원본·고정 공식 supply와 진행 중인 공용 개발 cache는 유지한다.
