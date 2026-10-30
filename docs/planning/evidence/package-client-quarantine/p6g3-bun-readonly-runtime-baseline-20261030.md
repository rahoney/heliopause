# P6-G3 — 명시 Bun readonly 프로그램 실행의 baseline

G3 source 작성 전 audit다. M13-004의 G2 앱/Node CLI·grouped qualification을 먼저
완료하며, 아래는 이미 존재하는 owner와 그 이후 남은 acceptance gap만 기록한다.
Node 실행이나 project lifecycle Bun 성공을 readonly Bun 프로그램 성공으로 사용하지 않는다.

Read: client09 §1–3/초기 Bun Node lane, plan20 §4–6, G1/G2 baseline과 실제 증거,
lessons INDEX·sandbox L18·artifact L02/L19·evidence L16·ci L06. 현재 Git/Queue가 우선이다.

| 책임 / owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun native source·whole graph/원본·현재 검사/Policy/records — G1/G2 existing owners | YES | Node current YES | G1 named local; G2 qualification 진행 | NO | 명시 Bun runtime을 선택해도 동일 전체 검역·새 승인/기록을 유지 |
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
