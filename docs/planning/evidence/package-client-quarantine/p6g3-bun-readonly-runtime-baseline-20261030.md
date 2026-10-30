# P6-G3 — 명시 Bun readonly 프로그램 실행의 baseline

G3 source 작성 전 audit다. M13-004의 G2 앱/Node CLI·grouped qualification을 먼저
완료하며, 아래는 이미 존재하는 owner와 그 이후 남은 acceptance gap만 기록한다.
Node 실행이나 project lifecycle Bun 성공을 readonly Bun 프로그램 성공으로 사용하지 않는다.

Read: client09 §1–3/초기 Bun Node lane, plan20 §4–6, G1/G2 baseline과 실제 증거,
lessons INDEX·sandbox L18·artifact L02/L19·evidence L16·ci L06. 현재 Git/Queue가 우선이다.

| 책임 / owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun native source·whole graph/원본·현재 검사/Policy/records — G1/G2 existing owners | YES | Node current YES | G1/G2 commit6d9d3bf named local YES; CI440 PENDING | NO | 명시 Bun runtime을 선택해도 동일 전체 검역·새 승인/기록을 유지 |
| Bun tool 공급/Host proof·private native volume — bun_tool_supply, native_runtime_volume, bun_artifact_runtime | YES | project lifecycle YES | 기존 named local/hosted YES | 기존 scoped YES | readonly one-off reader에 같은 고정 공급물 mount/cleanup을 결합; ambient PATH/자동 provisioning 금지 |
| readonly whole input·writer retirement·readback·attachment/정리 — node_execution_inputs/materialize | YES | Node current YES | 기존 Node named YES | 기존 scoped YES | Bun 전용 runtime identity/reader profile과 현재 approved layout을 결합; 별도 복사/완료 pipeline을 복제하지 않음 |
| private capture·external target/terminal·typed nonzero/timeout — captureAdmittedProgram, programExecutionClientResult | YES | Node/Python YES | 기존 named YES | 기존 scoped YES | Bun current admission·결과 adapter. 새 완료 token/marker/protocol 금지 |
| Bun Artifact startup read — IsBunImageStartupRead, Bun target lifecycle | YES | project/one-off installer YES | 기존 named YES | 기존 scoped YES | 일반 Bun 프로그램은 별도 exact image/ARTIFACT target/current generation에서만 기존 body 소비; CONTROL/network 권한 전용 금지 |
| readonly Bun program entry/context/profile/admission·명시 CLI runtime | NO | NO | NO | NO | 아래 구현 전 확인 항목과 실제 normal/nonzero·경계/정리·영향 소비자 검증 |

## 다음 구현 전 확인할 좁은 경계

- 현재 Node proof/context/runtime owner의 고정 Node identity와 source/Policy fingerprint를
  확인한다. Bun을 Node/PNPM으로 표시하거나 설치 snapshot으로 우회하지 않는다.
- 기존 application 요청·CLI runner가 typed runtime 선택을 어디에서 결합하는지 확인해
  필요한 최소 owner만 수정한다. 프로그램 argv에 HAA runtime 선택을 숨기지 않는다.
- 고정 Bun의 실제 entry 실행 recipe와 no-install/Node-Bun shebang 의미를 잠긴 원본에서
  확인한다. 임의 shell/interpreter·후속 dependency 다운로드를 허용하지 않는다.
- 기존 Node readonly input/정리·Bun 공급·capture/terminal을 재사용하고 exact reader의
  Bun image/entry/current generation/profile만 admission한다. 기존 Node profile과
  installer/project용 Bun network·CONTROL 경계를 넓히지 않는다.
- 싼 표적 거부를 먼저 확인하고 실제 명시 Bun normal/typed nonzero·취소/변조/오래된
  binding·보호 입력/자원 복원을 의미 있는 묶음으로 검증한다. G2 CI 진행 중 새 push로
  취소하거나 이미 통과한 불변 native/CPP/Go를 반복하지 않는다.

일반 bunx 동등성/hostile·모든 platform/profile/P7/support/signing·whole final은 별도 OPEN이다.
이 audit는 새로운 runtime 지원이나 완료 승인 근거가 아니다.

## 최소 구현 경계 확정 — source 전

G2는 commit6d9d3bf의 현재 앱2/Node CLI3/refusal2와 필수 로컬 profile을 통과했고 CI440이
frozen source로 진행 중이다. G3는 먼저 같은 검증된 Node-compatible JS bin을 `--bun`으로
고정 Bun에 명시 실행하는 lane을 닫는다. Bun-only shebang/TypeScript entry 자동 선택이나
일반 bunx 동등성까지 확대하지 않는다. 기존 Node entry byte proof는 JS 원본·닫힌 선언을
증명할 뿐 interpreter 실행 authority가 아니며 별도 Bun runtime identity를 결합한다.

Observed source38afee6c의 Arguments.rs/run_command.rs에서 no-install(런타임 자동 설치 방지),
no-env-file·no-macros flags와 별도 run_in_bun 의미를 확인했다. 이는 Bun1.4.4 바이너리의
source-build 인증이 아니며 실제 고정 ELF의 recipe 검증이 필요하다. 입력은 같은 readonly
node_modules tree/Node base image이고, 현재 existing native Bun volume의 root-owned0555
도구 공급·해시/단독 attachment와 cleanup을 재사용한다. 이 mount의 data 방식을 readonly
OCI layer로 오기하지 않는다. Artifact uid1000과 root 소유 보호·입력 RO·외부 관측이 유지된다.

기존 Node preparation/execution public 메서드는 유지하고 같은 private body의 실제 두
소비자로만 분기한다. Bun reader에만 기존 native volume을 연결해 소개/rehash하며 writer와
기존 Node reader는 그대로 둔다. 결과·capture/target/terminal과 source/Policy/records는 기존
owner를 사용한다. Bun program은 별도 current identity·ARTIFACT profile/entry handoff로
결합하며 CONTROL/native-installer 네트워크 권한을 재사용하지 않는다. `--bun`은 CLI의
신뢰된 실행 선택이며 프로그램 argv나 Core installation model에 숨기지 않는다.

## 현재 구현과 첫 표적 결과 — actual 미도달

Go의 Node public API와 기존 input/capture 본체를 유지한 G3 연결은 첫 표적5package15.153s PASS다. 별도 Bun profile/mode/current target·ARTIFACT demotion·native image/startup body를 연결한 첫 CPP 표적44.247s는 FAIL이었다. Fatal parser 거부가 임시 candidate를 폐기하므로 old state의 active flag를 성공한 reexec 권한처럼 비교한 마지막 test oracle이 잘못됐다. 해당 부당한 oracle만 제거하고 단계 진단을 추가한 동일 제품 C++6d72851a 후보는 Bun와 기존 Node boundary42.913s PASS(observer09615d38)다. 이 값은 full CPP/actual/current CLI 성공이 아니다. 원본은 raw p6-g3-cpp-targeted-v1이며 다음은 새 Go/런타임 문맥/flag 회귀와 실제 공개 Bun 앱이다. CI440/source6d9d3bf는 고정 상태를 유지한다.

G3 public TypeScript5.9.3의 actual 앱2는65.68s/operator67.839s PASS다. Native Bun source/whole 검사·현재 Evidence/Policy/새 기록과 별도 Bun program identity/handoff, normal0/typed Bun nonzero1을 확인했다. Observer09615d38/Cpp6d72851a이며 Host5/Docker 보존/helperwait0/profile 같은 inode 복원이다. Node-compatible JS의 명시 Bun program named local만 검증했으며 forced CLI/controlled boundary/full CPP·Go/hosted/general/final은 미완료다.

Forced CLI3 actual는177.83s/operator179.981s PASS다. bunx/bun x와 두 언어·JSON/사람용·출력 privacy·normal0/typed Bun nonzero1 및 implicit Rimraf12nodes/13edges의 fresh whole 승인을 확인했다. Host5/Docker/helperwait0·원래 profile bytes/같은 inode 원복이다. Controlled12는 현재 진행 중이며 full CPP/영향 Node/Go/hosted 검증은 아직 OPEN이다.

Controlled 첫 actual182.59s는9사례 PASS 뒤 comm-spoof oracle에서 FAIL했다. 실제 spoof-ready 출력 뒤 ARTIFACT/DIRECT_EXEC_ROOT를 유지한 채 kernel image UNKNOWN/pinned=false·OPEN_RESULT_CLASSIFICATION_PROC로 fail-stop했다. Node의 BUN 신원 유지 기대를 전용한 test가 이 안전한 UNKNOWN 거부를 인정하지 않았다. 제품 권한/observer는 바꾸지 않고 Bun에서만 그 정확한 fail-closed 사실을 인정하며, 통과9개를 재실행하지 않고 남은 comm-spoof/readonly/deadline3개만 검증한다. UNKNOWN을 신원 확인 성공으로 기록하지 않는다. 원본 raw first-v3/Host 보존/helperwait0/profile 원복을 확인했다.

Remainder3 actual73.39s/operator75.520s PASS로 첫9개와 합쳐 12종의 coverage를 확인했다. 이는 한 번의 full12 parent PASS가 아니며 raw first-v3의 parent FAIL을 보존한다. 제품/observer는 동일하고 test oracle만 달라졌다. 12종의 측정 합계 약235초에 맞춰 Bun fixture 전체 context만5분으로 잡고 기존 Node4분·case deadline30초·제품/CI job 한도는 유지한다. Saved 실제 marker DATA guard는 앱2/CLI3/boundary12·세 parent를 수용했고 이는 native 재실행 증거가 아니다. 다음 기존 Node actual과 fullCPP133·묶음 Go checks를 마친다.

최종 canonical CPP133/59.054s PASS이며 binary09615d38는 앞선 공개 앱/CLI·controlled 경계에서 실제 사용한 observer와 동일하다. 임시 runner/진단 source는 최종 후보에 없다. 기존 Node actual과 묶음 Go/문서 검증은 진행 중이며 hosted 성공으로 확대하지 않는다.

첫 Quick343.861s는49package 종료 중 Sandbox의 Go/C++ OCI helper 문자열 일치1개만 FAIL했다. C++ private 사본을 최종 fullCPP에서 갱신하기 전에 시작한 검사였고, 최종25 source hash를 Root/private 대조한 뒤 동일 helper 문자열의 임시 byte 진단은 PASS(차이 없음)였다. 최초 판정 시 읽힌 C++ bytes의 SHA는 미수집이므로 정확한 그 버전을 추측하지 않는다. 제품/test 판정을 바꾸지 않고 임시 진단을 제거했으며, 입력을 고정한 canonical Quick(기존 성공 test cache 재사용) 뒤 미실행 Race를 이어간다. CPP/실제 앱·CLI·경계는 반복하지 않는다.

영향받은 기존 Bun→Node 앱2 actual64.11s/operator66.285s PASS다. 최종 shared input/helper/capture 경로와 observer09615d38에서 정상0/typed Node1·fresh 승인/기록·Host5/Docker/helperwait0/profile 같은 inode 복원을 보존했다. Quick/Race/문서 결과 외의 실제 소비자는 이제 미완료가 아니다.

최종25 source에서 Quick51.429s·영향3package Race53.679s·checker Race77.532s·tagged 고정 Staticcheck2.832s PASS다. [Named local 결과](./p6g3-bun-readonly-program-local-20261030.json)가 source hashes/실제 실행/최초 실패·한정 교정/CPP133와 정리 증거를 연결한다. 새3parents는 기존 공급/helper 안에서만 연결했고 미래 Linux122/CPP133/Required10·job120min이다. G3 hosted와 전체 P6/P7/support/signed acceptance는 OPEN이다.
