# P6-G4 — Bun 선언 runtime과 TypeScript entry의 남은 구현

사용자2026-10-31 교정에 따라 기능 gap을 채널 이름/배포보다 먼저 수행한다.
기준 HEAD는4d7c6e3이며 제품 소스는 검증된ec1과 같다. 기존 네 profile은 그 소스의
증거다. 이 문서는 기존 구현을 대체하거나 새로운 후보의 qualification을 주장하지 않는다.

Read: client09 §1/§3/Bun one-off lane, plan20 §4–6, P7 command matrix,
G3 baseline, domain05/격리 계약, lessons artifact L02/L19·sandbox L18·evidence L16·ci L06.

## 실제 source audit

| 책임 / 기존 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| native source/whole graph·현재 검사/Policy/records — bun_run_execution, BunExecutionInspector | YES | YES | ec1/CI442 YES | 기존 named YES | 새 entry도 같은 complete source/새 승인에 연결; 재구현하지 않음 |
| 별도 pinned Bun runtime/ARTIFACT·readonly inputs/capture/terminal — sandbox/bun_execution, node_execution | YES | YES | G3·ec1/CI442 YES | 기존 named YES | 새 선언/TS target을 기존 고정 command에 연결; 권한/한도/observer 변경 계획 없음 |
| 현재 Bun entry byte proof | 부분 YES(Node-compatible만) | YES(Node proof) | 기존 JS YES | 새 entry NO | BindNodeTarget/ReadNodeExecutionEntry가 Bun shebang·no-shebang TS를 거부; 별도 Bun proof와 같은 original-byte reader 필요 |
| 선언에 따른 Node/Bun 선택 | NO(Bun 선언 자동) | NO | NO | NO | run의 bool을 강제 선택에만 사용; 현재 measured entry로 runtime을 binding하고 runtime consumer는 context의 exact identity 사용 |
| 새 Bun entry의 complete native layout/readback | 부분 YES(Node-compatible만) | YES(기존) | 기존 JS YES | 새 entry NO | BindBunNodeExecutionLayout의 Node 검사와 새 Bun 검사만 분리하고 동일 whole layout facts/render/readback 재사용 |
| aliases/help/실제 새 entry·거부·기존 Node 소비자 | aliases YES / 새 설명 NO | 기존 YES | 기존 YES / 새 NO | 새 NO | 자동 Bun declaration·명시 Bun TS·잘못된 interpreter/변조/선언 불일치·기존 Node 회귀 및 실제 고정 Bun 실행 |

원래 client09는 모든 native 옵션 호환을 약속하지 않고 uv run/pnpm exec/bun run,
runtime provisioning/임의 registry/global 등은 초기 범위 밖이다. stdin/TTY/파일 반출은
별도 acceptance다. 이를 필수 누락과 혼합하거나 구현 완료/승인된 유예로 바꾸지 않는다.
다른 남은 목표도 이처럼 concrete source/acceptance gap으로 순차 대조한다.

## 최소 수정 지도

- `artifact/npm/node_execution_entry.go`: 기존 Node 공개 API/선언 검사는 그대로 두고
  원본 size/digest·bounded clone 부분만 private shared body로 재사용한다.
- 새 Bun entry proof는 같은 ExecutionEntry에 묶되 Node proof로 위장하지 않는다.
  닫힌 Node/Bun shebang·JS/TS 확장자만 인정하고 arbitrary interpreter/args는 거부한다.
  기본 선택은 명시 Bun shebang이면 Bun, 기존 Node declaration이면 Node다.
  `--bun`은 검증된 JS/TS entry에 고정 Bun을 명시한다. 선언 없는 TS의 기본 runtime을
  확장자만으로 추측하거나 원래 도구의 모든 실행 의미를 구현했다고 표시하지 않는다.
- `inspection/npm/execution_entry.go`: 기존 실제 archive capture/전체 inventory 일치·
  source/cancellation 본체를 Node/Bun 두 실제 소비자가 공유하고 각각 typed proof를 반환한다.
- `bootstrap/node_execution_prepare.go`/`node_execution_context.go`: 기존 현재 inventory 선택과
  Domain context 필드 결합을 공유한다. Node API/조건은 보존하고 Bun proof와 exact runtime
  결합을 같은 실제 workflow에 추가한다. 새 Core interface/설치 target/가짜 graph는 없다.
  G4 전환 후 production 소비자가 없어진 기존 `bindJavaScriptExecutionEntry`의 runtime
  override와 `newJavaScriptExecutionContext` 중간 wrapper는 Node 경로에 inline한다.
  Node proof 검사·고정 Node identity는 동일하고 Bun 회귀도 현재 typed Bun 생성자를
  사용한다. 이미 구현된 전체 검사/실행 기능을 대체하는 수정이 아니다.
- `artifact/npm/bun_execution_layout.go`: 기존 Node 검사는 보존하며 새 Bun binder만
  같은 complete layout·원본 재측정/entry/digest/size·facts/render를 재사용한다.
- `bootstrap/bun_run_execution.go`: force bool과 최종 runtime의 중복 권위는 제거한다.
  entry에서 binding된 context의 exact Node/Bun identity로 실제 기존 backend를 선택한다.
- `cli/bun_execution.go`: 실제 연결 후에만 두 alias/help의 Node-only 제한을 정정한다.
  source/runtime provisioning·네트워크/Host 권한·private argv/기록·cleanup 계약은 유지한다.
- 실제 public application/CLI는 기존 tagged bootstrap test와 fixture owner를 재사용한다.
  `scripts/ci-bun-runtime.sh`의 기존 run cohort와
  `scripts/check/bun_runtime_workflow.go`의 동일 exact guard에 새 두 parent를 연결한다.
  기존 세 parent·application/CLI/refusal/graph 검사는 유지하고 새 app2/CLI3의
  중복·누락·잘못된 runtime 결과·SKIP 거부를 같은 AWK owner에서 검증한다.
  기존 `scripts/check/pnpm_runtime_workflow_test.go`의 공용 PNPM/Bun fixture에
  Bun 분기만 확장하며 별도 중복 guard fixture를 만들지 않는다.
  새 job/profile/helper나 observer 변경은 없으며 기존10분 cohort 한도를 유지한다.

## 원본 의미와 검증 경계

[공식 bunx 문서](https://bun.com/docs/pm/bunx)는 Node shebang 기본과 명시 --bun,
Bun shebang의 Bun 실행을 구분한다. 고정 bun-v1.4.4 tag는
`38afee6c84fcf48c0edb2466aa4b27d287a59c30`이다. src/cli는 runtime/cli symlink이며
실제 src/runtime/cli/bunx_command.rs의 run_in_bun/Run::run_binary와 run_command.rs를
읽었다. 원본 조회는 source-build 인증이나 새 native 실행 성공이 아니다.

먼저 실제 원본/byte/runtime-choice의 싼 counterexample과 영향받은 기존 Node 거부를
검사한다. 이후 연결된 application/layout/CLI의 현재 whole 승인 및 고정 Bun의 실제
Bun 선언/TS 정상·typed nonzero·원본/관측 불완전 거부·정리를 검증한다.
현재 단계에서 기존 CPP/native/full/네 CI를 다시 돌리지 않는다. 필요한 영향 검증은
완성된 기능 묶음으로 모으며 제품/observer 변경이 실제로 확인된 경우에만 확대한다.
Hosted/final과 전체 목표 acceptance는 실제 근거가 생기기 전까지 OPEN이다.

## 두 번째 검토

기존 전체 검사/Policy·runtime/capture/records를 새로 구현하지 않는다. Shared body는
현재 Node와 새 Bun이라는 두 실제 소비자에만 연결한다. 이미 구현된 것의 교체가 아니라
Node-only predicate가 막는 새 entry acceptance만 보완한다. Typed entry/data는 승인이나
interpreter 실행 authority가 아니며, fresh whole ALLOW·고정 runtime·외부 target/terminal이
모두 필요하다. 보안 경계를 넓혀야만 가능한 관측이 나오면 그 최초 실패를 먼저 보존한다.

## 관측된 로컬 결과와 최종 후보

[G4 local receipt](./p6g4-bun-declared-entry-local-20261031.json)는 최종19 source hash,
처음 두 준비/fixture 실패와 교정, public Bun 선언/TS 앱2·CLI3와 기존 Node 앱2,
완료·정리·Host 원복을 연결한다. 첫 actual은 private Bun 공급물 누락으로 entry에
미도달했고 제품을 고치지 않았다. 기존 측정 공급을 재사용한 actual은211.563s/exit0이다.
기존 공용 guard fixture 누락도 표적 검사에서 검출해 같은 owner의 Bun 분기만 확장했다.

v5의 Quick325.409s·영향4 Race130.781s·checker Race91.082s가 PASS다. 실제 실행 뒤
production 소비자가 사라진 두 Node runtime override wrapper만 inline한 최종 v6은
Quick68.579s·Bootstrap Race15.841s PASS다. 같은 선택 함수·공통 context 필드 결합의
body와 새 Bun 실행 경로는 불변이며 실제 source snapshot과 최종 후보를 구분한다.
변경하지 않은 다른3/checker Race·CPP/actual/세 CUDA를 반복하지 않았다.

최종 [CI446 qualification](./ci446-bun-declared-entry-hosted-qualification-20261031.json)은 같은
최종 sourcee3b2b70/전체tree의Required10SUCCESS·126 unique PASS, 새 앱2/25.34s·CLI3/39.35s,
기존 소비자·helper18wait0/최종4 정리를 확인했다. 실제20분GET5에서 종료를 소비하고
워커의 자연exit0/nextnull까지 관측했다. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES /
ACCEPTANCE_CLOSED=YES(original bounded G4 scope), MISSING=[]다. ec1의 CUDA3를 새e3의
exact-source qualification으로 전용하지 않는다. 다음은 M13 채널의 공식 이름과 publisher binding이다.
