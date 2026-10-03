# Domain Model — Inspection Run and Status

## Domain-003: Inspection Run 모델

### 사용자 원문

`Inspection Run`은 Heliopause에서 하나의 검사 실행과 그 과정에서 생성된 결과를 추적하는 핵심 lifecycle 단위다.

하나의 Inspection Run은 사용자의 원래 Operation Request와 resolve된 Artifact를 기준으로 시작하며, 실제 획득한 Artifact, 수행된 Verification/Inspection, Execution Status, Evidence, Finding, 검사 한계 및 최종 Policy Decision을 하나의 추적 가능한 검사 기록으로 연결한다.

```text
Operation Request
        ↓
Artifact Reference
        ↓
Resolved Artifact Identity
        ↓
Inspection Run 생성
        ↓
Acquisition / Controlled Intake
        ↓
Acquired Artifact binding
        ↓
Verification / Inspection
        ↓
Evidence / Finding / Status / Limitation
        ↓
Policy Evaluation
        ↓
Inspection Run 종료
```

Inspection Run은 실제 Sandbox Session이나 개별 scanner 실행과 동일한 개념이 아니며, 하나의 Run 안에서 여러 검사 단계·외부 도구 실행·retry attempt·Sandbox Session이 발생할 수 있다.

Domain-003에서는 정확한 Go struct, persistence schema, 상태 enum 및 Run 관계 schema를 확정하지 않고 lifecycle, ownership, 불변성 및 추적 원칙을 정의한다.

## Inspection Run의 생성 시점

Inspection Run은 사용자의 Artifact Reference가 exact `Resolved Artifact Identity`로 확정된 이후, 실제 Artifact acquisition을 시작하기 전에 생성하는 것을 기본 원칙으로 한다.

```text
Artifact Reference
        ↓
exact identity resolve
        ↓
Inspection Run 생성
        ↓
Acquire
```

acquisition 자체가 실패하거나 중단되어도 해당 검사 시도와 실패 정보를 Run에 연결할 수 있어야 한다. source/ecosystem/version/release/asset이 모호하여 exact identity가 resolve되지 않은 상태에서는 정상적인 Run을 시작하지 않는다.

```text
입력 모호
→ exact identity 미확정
→ Inspection Run 시작 전 사용자 입력 요구
```

```text
Resolved Artifact Identity
→ Run 생성의 최소 Artifact 기준

Acquired Artifact
→ Run 생성 이후 실제 검사 대상 binding
```

## Inspection Run ID

각 Inspection Run은 다른 Run과 구분되는 고유 식별자를 가진다. Run ID는 terminal 출력, `review`, Evidence 조회, machine-readable output 및 후속 Run 관계 추적의 기준이다.

Run ID의 format, 생성 algorithm 및 serialization 방식은 후속 Domain/Storage 설계에서 결정한다. 같은 Artifact를 다시 검사하더라도 새로운 Run을 생성하면 기존 Run ID를 재사용하지 않는다.

## Operation Request와의 연결

Inspection Run은 어떤 사용자 작업으로부터 시작되었는지 추적할 수 있어야 한다.

```text
Operation Request
operation = install
artifact = kenv
ecosystem = npm

        ↓

Inspection Run
```

이를 통해 `ALLOW`일 때 원래 install context를 Promotion으로 이어가고, `inspect` 요청이면 Promotion 없이 종료한다.

```text
Inspection Run
        ↓
Original Operation Request
        ↓
install / inspect 등의 원래 사용자 의도 확인
```

Run이 Operation Request 전체를 직접 소유할지 identifier/reference로 연결할지는 후속 ownership 설계에서 결정한다.

## Primary Inspection Subject

각 Run은 어떤 Primary Artifact를 검사하기 위해 생성되었는지 추적할 수 있어야 한다.

Run 생성 시점에는 `Resolved Artifact Identity`가 기준이고, acquisition 완료 후에는 다음이 실제 검사 대상 binding이 된다.

```text
Resolved Artifact Identity
        +
Acquired Artifact
        +
Observed Digest
```

```text
Inspection Run
      ↓
Primary Subject
      ↓
Resolved Artifact Identity
      ↓ acquisition
Acquired Artifact
      ↓
Observed Digest
```

Acquired Artifact가 binding된 뒤에는 다른 content로 조용히 교체하지 않는다.

```text
Run A
subject = identity X + digest AAA
```

대상이 `identity X + digest BBB`로 변경되면 기존 결과를 현재 대상 결과로 이어서 사용할 수 없다. 정확한 reacquisition/retry 방식은 후속 설계에서 정의하되 Run이 실제 검사한 content는 항상 명확해야 한다.

## Dependency와 관련 Artifact의 범위

하나의 사용자 요청에서 Primary Artifact와 dependency·관련 Artifact가 함께 검사될 수 있다.

```text
Primary Artifact
├─ Dependency A
├─ Dependency B
└─ Dependency C
```

Inspection Run은 workflow에서 처리된 dependency와 관련 Artifact를 추적할 수 있어야 한다. 모든 dependency가 반드시 하나의 Run에만 속한다고 고정하지 않으며, 특정 dependency 또는 후속 검사가 별도 Run에서 검증되고 기존 Run과 연결될 수 있다.

```text
Primary Run A
      ↓
Dependency / 후속 검사
      ↓
Run B
```

최종 Verified Set에서는 각 Artifact/dependency가 자신을 실제로 검증한 Run에 추적 가능해야 한다. dependency graph, Run 간 dependency 관계와 Verified Set 구성은 Domain-006에서 구체화한다.

## Inspection Run Context

검사 결과를 나중에 해석·비교할 수 있도록 당시의 중요한 실행 context에 추적 가능해야 한다.

```text
Inspection Run Context
├─ 검사 시점
├─ 적용된 Policy 또는 Policy version
├─ 사용된 verifier / scanner / inspector 및 version
├─ Sandbox / Runtime backend
├─ Capability
├─ 검사 platform / environment
└─ 적용된 주요 검사 configuration
```

정확한 field는 Domain-003에서 고정하지 않는다. 같은 Artifact라도 다른 Policy·Tool·Capability·Environment에서는 결과 의미가 달라질 수 있으므로 과거 Run 재사용·비교 시 당시 context를 확인할 수 있어야 한다.

## Run 안의 검사 단계와 실행 Attempt

Inspection Run은 하나의 scanner 또는 Sandbox 실행이 아니다. 하나의 Run에서 여러 단계와 attempt가 실행될 수 있다.

```text
Inspection Run
├─ Acquisition
├─ Verification
│   ├─ integrity verification
│   ├─ signature verification
│   └─ provenance verification
├─ Static Inspection
├─ Dynamic Inspection
│   └─ Sandbox Session
└─ Policy Evaluation
```

각 단계는 여러 attempt를 가질 수 있으며 최초 실패를 숨기지 않는다.

```text
Verification
   ↓
Attempt 1 → Failed
   ↓
safe retry
   ↓
Attempt 2 → Completed
```

필요한 경우 각 attempt와 최종 Execution Status를 Run에서 추적한다. Attempt·Check·Stage의 정확한 하위 Domain 구조는 후속 Inspection/Status Model에서 정의한다.

## Inspection Run과 Sandbox Session의 구분

`Inspection Run`과 `Sandbox Session`은 서로 다른 lifecycle이다.

```text
Inspection Run
      ↓
Dynamic Inspection 필요
      ↓
Sandbox Session A
```

Session A가 비정상 종료되면 폐기하고 필요한 경우 새 Session B를 생성한다.

```text
Sandbox Session A → 폐기
Sandbox Session B → 새로 생성
```

전체 검사 의미에서는 동일 Run 내부 retry일 수도 있고 상황에 따라 새 Run으로 이어질 수도 있다. 정확한 retry/resume 경계는 후속 설계에서 정의하지만 비정상 Session을 복원·재사용해 정상 상태처럼 취급하지 않는 원칙은 유지한다.

## Inspection Run이 연결하는 검사 결과

Run은 해당 검사 과정에서 생성되거나 참조되는 다음 결과에 추적 가능해야 한다.

```text
Inspection Run
├─ Acquired Artifact
├─ Verification Result
├─ Evidence
├─ Finding
├─ Capability
├─ Execution Status
├─ Inspection Limitation
└─ Policy Decision
```

모든 항목이 모든 Run에 반드시 존재하는 것은 아니다. acquisition 실패 Run은 `Acquired Artifact`와 `Policy Decision`이 없을 수 있지만 Acquisition failure, Execution Status, failure reason, Run termination information은 기록할 수 있어야 한다.

항목이 존재하지 않는 것과 기록이 누락된 것을 구분할 수 있어야 한다.

## Policy Decision과 Inspection Run

Policy Evaluation까지 도달한 Run은 최종 `Policy Decision`을 연결할 수 있다.

```text
ALLOW
MANUAL_REVIEW
BLOCK
```

하나의 최종화된 Run에는 하나의 최종 Policy Decision만 존재하는 것을 기본으로 한다. 과거 `MANUAL_REVIEW`를 같은 Run에서 `ALLOW`로 덮어쓰지 않고 새 Run을 생성한다.

```text
Run A
Policy: MANUAL_REVIEW
        ↓
추가 검사 / 재검사
        ↓
Run B
Policy: ALLOW
```

## Policy Decision이 없는 종료

모든 Run이 `ALLOW / MANUAL_REVIEW / BLOCK`까지 도달한다고 가정하지 않는다. source/network/controller failure로 Artifact 획득 전에 종료될 수 있다.

```text
Inspection Run
Run outcome: failed / interrupted
Policy Decision: not produced
```

이 operational failure를 단순히 `BLOCK`으로 변환하지 않는다.

```text
BLOCK
→ Artifact에 대한 보안 판정

Operational Failure
→ 검사 workflow 자체가 완료되지 못한 상태
```

실제 Artifact를 확보한 뒤 필수 Verification/Inspection이 실패·미완료·Unavailable이면 fail-closed 원칙에 따라 `MANUAL_REVIEW` 또는 `BLOCK`을 생성할 수 있다. 어떤 실패가 Policy Evaluation 대상이고 어떤 실패가 Run-level operational failure인지는 Domain-004와 Policy 설계에서 구체화한다.

## Run Lifecycle State와 Execution Status의 구분

Run 자체의 lifecycle 상태와 개별 검사 단계의 `Execution Status`를 동일하게 취급하지 않는다.

```text
Inspection Run → 전체 검사 lifecycle 상태
Verification Check → Execution Status: Completed
Dynamic Inspection → Execution Status: Skipped
Policy Evaluation → Completed
```

```text
Run Lifecycle State
≠
개별 Check Execution Status
```

Run lifecycle의 정확한 enum은 Domain-004에서 결정한다. Domain-003에서는 Run 생성됨, 실행 중, 중단 또는 실패 가능, 최종화됨의 의미를 표현할 수 있어야 한다는 것만 고정한다.

## Inspection Run의 Finalization

검사 workflow가 정상적으로 완료되거나 현재 Run에서 더 이상 계속하지 않기로 확정되어 결과와 종료 상태를 최종 기록하는 시점에 finalization한다.

```text
검사 완료 → Policy Decision 확정 → Run Finalization
```

Policy Decision을 생성하지 못하고 종료한 경우에도 확보된 기록과 실패 상태를 확정한 뒤 terminal 상태로 만들 수 있다.

```text
Operational Failure
        ↓
확보된 기록 저장
        ↓
Run Finalization
```

최종화된 Run은 이후 다른 검사 결과로 덮어쓰지 않는다.

## Finalized Run의 불변성

최종화된 Run의 historical meaning을 변경하지 않는다. 다음을 사후 새 결과로 교체하지 않는다.

```text
검사한 Artifact identity / digest
수행된 검사
Execution Status
Verification Result
Finding
Evidence reference
Inspection Limitation
최종 Policy Decision
```

Evidence retention이나 storage migration은 가능하지만 당시 무엇을 검사했고 어떤 결과를 냈는지의 의미는 변하지 않아야 한다. 정정·추가 검사·재판정은 새 Run 또는 명시적으로 연결된 후속 기록으로 표현한다.

## Active Run과 Finalized Run의 차이

실행 중인 Active Run에는 과정에 따라 Evidence·Finding·Execution Status·Policy 평가가 추가될 수 있다.

```text
Active Run
→ Evidence 추가
→ Finding 추가
→ Execution Status 갱신
→ Policy 평가
```

finalization 이후에는 historical inspection record인 Finalized Run으로 취급한다. 불변성은 생성 순간부터 모든 field가 불변이라는 뜻이 아니라 finalization 이후 기록 의미를 변경하지 않는다는 뜻이다.

## 중단된 Run과 Resume

사용자 interrupt, Heliopause process 종료 또는 시스템 오류로 Run이 중단될 수 있다. 가능한 경우 Run metadata, Execution Status, Evidence, Finding, Verification Result 등은 보존한다.

그러나 중단된 Run이 존재한다는 이유만으로 runtime state 전체를 그대로 이어서 실행하지 않는다. Sandbox Session, temporary execution state, untrusted process state는 resume 가능한 신뢰 상태로 간주하지 않는다.

중단된 Run을 checkpoint에서 계속 사용할지 기존 Run을 참조하는 새 Run을 생성할지는 후속 retry/resume 설계에서 결정한다. 이미 Finalized된 Run은 resume하지 않는다.

## 재검사와 Run 관계

동일 Artifact를 다시 검사하거나 문제 해결 후 재검사할 때 기존 Finalized Run을 수정하지 않고 새 Run을 생성한다.

```text
Run A
        ↓ 재검사
Run B
```

새 Run은 이전 Run을 참조하여 관계를 추적할 수 있다. 향후 필요한 관계 후보는 `retry-of`, `reinspection-of`, `follow-up-of`, `supersedes`, `related-to`지만 Domain-003에서 enum으로 확정하지 않는다.

```text
Run B가 Run A 이후 생성되었더라도
Run A의 historical record는 유지된다.
```

## 같은 Artifact의 Run 비교

동일한 `Resolved Artifact Identity + Observed Digest`라도 Run이 다르면 결과가 달라질 수 있다.

```text
Artifact: identity X / digest AAA

Run A: Policy version 1, Tool version 1 → MANUAL_REVIEW
Run B: Policy version 2, Tool version 2 → ALLOW
```

```text
same Artifact
≠
same Inspection Run
```

각 Run은 당시 검사 context와 결과를 독립적으로 유지하며 과거 결과 재사용 조건은 후속 cache/reuse 설계에서 결정한다.

## Evidence Store와 Inspection Run

Run이 모든 Raw Evidence bytes를 직접 포함할 필요는 없다. Architecture에서 정의한 Evidence Store와 reference로 연결할 수 있다.

```text
Inspection Run
      ↓
Evidence Reference
      ↓
Evidence Store
```

핵심은 어떤 Evidence가 어떤 Run에서 어떤 Artifact에 대해 어떤 검사로 생성되었는지 추적하는 것이다. Raw Evidence 저장 위치와 retention은 Evidence/Result 설계에서 결정한다.

## Promotion과 Inspection Run

Promotion은 유효한 `ALLOW`가 연결된 Run을 근거로 수행한다.

```text
Inspection Run
      ↓
Policy = ALLOW
      ↓
install operation인 경우
      ↓
Package ecosystem
→ Verified Set / Manifest
→ trusted Promotion

Standalone
→ exact identity / digest 재확인
→ trusted Promotion
```

Promotion 수행이 Run의 Policy Decision을 변경하지 않는다. 실제 설치·반입 결과는 Operation Result 또는 후속 Promotion Record로 Run과 연결한다.

```text
Inspection Run
Policy = ALLOW
        ↓ Promotion
Operation Result
Installation = FAILED
```

```text
Inspection Run → 검사와 보안 판정 기록
Operation Result → 실제 사용자 작업 결과
```

Operation Result는 Promotion이 발생한 install뿐 아니라 inspect, MANUAL_REVIEW, BLOCK 등 원래 Operation의 결과도 표현할 수 있으며 정확한 모델은 Domain-004/007에서 구체화한다.

## Inspection Run이 직접 소유하지 않는 것

### Sandbox Session

하나의 Run에서 0개 이상의 Sandbox Session이 발생할 수 있으며 비정상 Session은 폐기된다.

### External Tool Process

scanner/verifier/package manager process는 Run을 수행하는 implementation detail 또는 Provider 실행이다.

### Staging Area

Staging은 `ALLOW` 이후 Verified Artifact를 Promotion하기 위한 별도 Architecture 영역이다.

### Operation Result

Run과 연결되지만 실제 사용자 operation의 결과이므로 inspection lifecycle 자체와 동일한 개념으로 취급하지 않는다.

## Domain-003에서 확정하는 Invariant

### Invariant 1 — Exact Identity resolve 이후 Run을 시작한다

입력이 모호한 상태에서 정상적인 Inspection Run을 시작하지 않는다.

### Invariant 2 — Acquisition 전에 Run을 생성한다

acquisition 실패·중단 자체도 해당 검사 시도로 추적할 수 있도록 resolved identity 확정 후 acquisition 전에 Run ID를 생성한다.

### Invariant 3 — Run은 실제 검사 Artifact에 binding된다

Acquired Artifact가 확보되면 exact identity와 observed digest를 해당 Run의 검사 대상에 연결한다.

### Invariant 4 — Bound Artifact를 조용히 교체하지 않는다

Run에 binding된 실제 Artifact content가 변경되면 기존 결과를 새로운 content 결과로 사용하지 않는다.

### Invariant 5 — Run과 Sandbox Session을 구분한다

하나의 Inspection Run에서 여러 Sandbox Session과 tool execution이 발생할 수 있다.

### Invariant 6 — Run은 모든 주요 결과의 추적 기준이다

Verification Result, Finding, Evidence, Execution Status, Inspection Limitation 및 Policy Decision은 해당 Run과 실제 Artifact에 추적 가능해야 한다.

### Invariant 7 — 모든 Run에 Policy Decision이 필수는 아니다

Policy Evaluation까지 도달하지 못한 operational failure Run이 존재할 수 있으며 이를 임의로 `BLOCK`으로 변환하지 않는다.

### Invariant 8 — Policy Decision과 Operational Failure를 구분한다

`BLOCK`은 보안 판정이며 workflow 자체의 실행 실패와 동일한 의미가 아니다.

### Invariant 9 — Finalized Run은 불변이다

최종화된 Run의 Artifact, 검사 결과, Evidence 관계 및 최종 Policy Decision을 후속 재검사 결과로 덮어쓰지 않는다.

### Invariant 10 — 재검사는 새로운 Run으로 표현한다

Finalized Run 이후 재검사가 필요하면 새 Run을 만들고 필요한 경우 이전 Run과 관계를 연결한다.

### Invariant 11 — 같은 Artifact라도 Run은 독립적이다

같은 identity/digest라도 Policy, Tool, Capability 또는 검사 환경이 다르면 별도의 Run 결과가 존재할 수 있다.

### Invariant 12 — Promotion 결과와 Run의 Policy를 분리한다

실제 사용자 operation 수행 결과가 실패하더라도 기존 Inspection Run의 `ALLOW`를 자동으로 `BLOCK`으로 변경하지 않는다.

## 후속 결정으로 유보한 항목

- Inspection Run ID format 및 생성 방식
- 정확한 Run Lifecycle State enum
- Run 시작·종료 timestamp field
- Stage / Check / Attempt의 정확한 Domain 구조
- Run Context field와 snapshot 방식
- Policy version 표현
- tool/provider/backend version 표현
- dependency별 별도 Run 생성 조건
- parent/child Run 관계
- `retry-of`, `reinspection-of`, `follow-up-of`, `supersedes` 등의 관계 enum
- interrupted Run의 checkpoint/resume 방식
- automatic retry가 같은 Run인지 새 Run인지 결정하는 정확한 경계
- Run cache/reuse 조건
- Evidence reference schema
- Run storage/persistence schema
- retention 기간
- concurrent Run 처리
- cancellation model
- Operation Result와 Promotion Record의 정확한 연결 schema

위 항목은 Domain-004 이후 Status Model, Verification/Finding/Evidence, Dependency/Verified Set, Policy, Evidence/Result 및 Tooling 설계에서 구체화한다.

## 압축된 모델

```text
사용자 요청
      ↓
exact Artifact resolve
      ↓
Inspection Run 생성
      ↓
Run ID 발급
      ↓
Artifact acquisition
      ↓
identity + observed digest binding
      ↓
Verification / Inspection
      ↓
Evidence / Finding / Status / Limitation
      ↓
Policy Evaluation
      ↓
      ├─ ALLOW
      ├─ MANUAL_REVIEW
      ├─ BLOCK
      └─ Policy 이전 operational failure 가능
      ↓
Run Finalization
      ↓
historical record로 불변 유지
```

재검사 시:

```text
Run A
Finalized
   ↓
재검사
   ↓
Run B
새 Run ID
```

핵심 원칙은 다음과 같다.

```text
Inspection Run
=
"이번 검사에서
정확히 무엇을 대상으로
어떤 환경과 검사 조건에서
무엇을 수행했고
무엇을 관찰했으며
어떤 판정에 도달했는가"
를 추적하는 기준
```

Heliopause는 Inspection Run을 단순 command 실행 로그나 Sandbox Session으로 취급하지 않고, 실제 Artifact identity/digest와 Verification·Inspection·Evidence·Finding·Policy를 연결하는 불변 가능한 감사·재현 단위로 사용한다.

## Domain-003 구조화된 결정과 구현 영향

- exact identity resolve 후 acquisition 전에 Run을 생성하고 Run ID를 발급한다.
- Run은 Operation Request, Primary Subject, Acquired Artifact, identity/digest를 연결한다.
- 하나의 Run 안에 여러 단계·provider 실행·retry attempt·Sandbox Session이 포함될 수 있다.
- Run context에 당시 Policy·tool/provider/backend·Capability·platform/environment·configuration을 추적한다.
- 결과가 존재하지 않는 상태와 기록 누락을 구분한다.
- Run-level operational failure와 Artifact 보안 판정 `BLOCK`을 분리한다.
- Run lifecycle state와 개별 Check Execution Status를 별도 모델로 둔다.
- Active Run은 결과를 추가할 수 있지만 Finalized Run은 historical record로 불변 유지한다.
- 중단 Run의 metadata·완료 단계·status·Evidence는 보존하되 Sandbox/temporary/untrusted runtime state는 resume하지 않는다.
- 재검사는 새 Run으로 표현하고 이전 Run과 관계를 연결할 수 있지만 정확한 관계 enum은 유보한다.
- 동일 identity/digest라도 context가 다른 Run은 독립 결과로 유지한다.
- Evidence Store는 Run과 reference로 연결하고 Raw Evidence 저장·retention은 별도 설계한다.
- Promotion은 Run의 유효한 `ALLOW`를 근거로 하며, Package ecosystem에서는 Verified Set/Manifest를, Standalone에서는 검사한 exact identity/digest를 유지하여 수행하고 Operation Result와 분리한다.

## Domain-003 누락 점검

- [x] exact identity resolve 후 acquisition 전 Run 생성
- [x] unique Run ID와 review/machine-readable 추적
- [x] Operation Request 연결
- [x] Primary Subject와 Acquired Artifact/observed digest binding
- [x] bound content 교체 금지
- [x] dependency·관련 Artifact 추적과 별도 Run 연결 가능성
- [x] Run Context의 Policy·tool·backend·Capability·environment 추적
- [x] 단계·attempt·retry·Sandbox Session 다중 실행
- [x] Run과 Sandbox Session lifecycle 분리
- [x] 결과 항목의 존재/미생성/누락 구분
- [x] Policy Decision 없는 operational failure
- [x] Run lifecycle과 Execution Status 분리
- [x] Finalization과 Finalized Run 불변성
- [x] Active Run과 Finalized Run 구분
- [x] 중단 Run 보존과 runtime state resume 금지
- [x] 재검사 새 Run 및 Run 관계 추적
- [x] 동일 Artifact라도 context별 독립 Run
- [x] Evidence Store reference 연결
- [x] Promotion Policy와 Operation Result 분리
- [x] ID·state·context·checkpoint·cache·relation·storage 후속 결정 유보

## Domain-004: Capability / Execution / Run / Policy / Operation 상태 모델

### 사용자 원문

Heliopause는 검사 workflow에서 발생하는 여러 종류의 상태를 하나의 성공/실패 값으로 합치지 않고 각각의 의미에 따라 독립적으로 구분한다.

```text
Capability
→ 이 검사를 수행할 수 있는가?

Execution Status
→ 해당 검사가 실제로 어떻게 실행되었는가?

Run Lifecycle / Outcome
→ 전체 Inspection Run은 어떤 상태이며 어떻게 종료되었는가?

Policy Decision
→ 검사 결과를 기준으로 Artifact를 반입할 수 있는가?

Operation Status
→ 사용자가 요청한 실제 작업은 어떻게 끝났는가?
```

```text
Capability
≠ Execution Status
≠ Run Outcome
≠ Policy Decision
≠ Operation Status
```

Domain-004에서는 다섯 상태 축의 의미와 기본 상태 집합을 정의한다. 정확한 Go enum 이름, error code, serialization schema, 사용자-facing 문자열 및 세부 failure taxonomy는 후속 설계에서 확정한다.

## 1. Capability

`Capability`는 특정 검사 또는 검증을 현재 Artifact와 검사 환경에서 수행할 수 있는지를 나타낸다.

```text
SUPPORTED
UNSUPPORTED
```

`SUPPORTED`는 필요한 기능을 수행할 구현 또는 backend capability가 있음을 의미하지만 실제 실행·성공을 뜻하지 않는다.

```text
Dynamic Inspection
Capability: SUPPORTED
Execution Status: FAILED
```

`UNSUPPORTED`는 현재 Artifact·platform·Adapter·backend 조합이 해당 검사를 지원하지 못함을 의미한다.

```text
macOS 전용 executable + MVP Linux Dynamic Inspection backend
→ macOS runtime behavior 검사
→ Capability: UNSUPPORTED
```

`UNSUPPORTED`를 안전으로 해석하지 않는다. 보안상 필수 검사라면 fail-closed에 따라 `ALLOW`하지 않고 Policy rule에 따라 `MANUAL_REVIEW` 또는 `BLOCK`으로 판단한다.

### Capability와 현재 실행 가능성의 구분

Capability는 지원 여부이고 현재 실행 시점의 장애와 구분한다.

```text
검사 기능 자체는 지원됨
필요한 scanner process가 현재 실행 불가

Capability: SUPPORTED
Execution Status: UNAVAILABLE
```

```text
backend 자체가 해당 Artifact 검사를 지원하지 않음

Capability: UNSUPPORTED
Execution Status: NOT_EXECUTED
Reason: capability unsupported
```

Backend 일반 capability와 특정 Run에 적용 가능한 capability의 정확한 모델은 후속 Contract 설계에서 결정한다.

## 2. Execution Status

`Execution Status`는 특정 Verification, Inspection, Check 또는 기타 검사 작업이 실제로 어떻게 수행되었는지를 나타낸다.

```text
COMPLETED
FAILED
INCOMPLETE
NOT_EXECUTED
UNAVAILABLE
```

### `COMPLETED`

의도한 실행 범위를 정상적으로 완료하고 해석 가능한 결과를 생성했음을 의미한다. Artifact 안전을 의미하지 않는다.

```text
Dynamic Inspection: COMPLETED
Finding: Credential access attempt detected
Policy: BLOCK
```

```text
Execution COMPLETED
≠ Security SAFE
```

### `FAILED`

검사를 시도했지만 tool/process/backend 오류 등으로 의도한 검사를 완료하지 못했다는 뜻이다. 일부 Evidence가 생성되었을 수 있으며 삭제하지 않는다.

```text
Verifier started → Verifier crashed
Execution Status: FAILED
```

### `INCOMPLETE`

일부 수행했지만 필요한 전체 범위를 완료하지 못했다는 뜻이다.

```text
Dynamic Inspection 시작
→ 일부 행동 관찰
→ resource limit 초과·강제 종료
→ Execution Status: INCOMPLETE
```

`FAILED`와 `INCOMPLETE`의 세부 기준은 후속 Inspection/Tooling 설계에서 정하지만 둘 다 정상 `COMPLETED`와 동일하게 취급하지 않는다.

### `NOT_EXECUTED`

검사가 실제로 시작되지 않았음을 의미한다. 이유 없이 단독 기록하지 않는다.

```text
Dynamic Inspection
Execution Status: NOT_EXECUTED
Reason: not required
```

또는:

```text
Execution Status: NOT_EXECUTED
Reason: capability unsupported
```

reason 후보는 `not required`, `not applicable`, `capability unsupported`, `precondition not satisfied`, `policy skipped`이며 정확한 taxonomy는 후속 설계에서 정한다.

### `UNAVAILABLE`

검사는 지원하지만 실행 시점에 필요한 tool, provider, backend 또는 자원을 사용할 수 없어 수행하지 못했다는 의미다.

```text
Capability: SUPPORTED
scanner binary missing
→ Execution Status: UNAVAILABLE
```

```text
UNSUPPORTED → 애초에 capability를 제공하지 못함
UNAVAILABLE → capability는 있지만 현재 실행할 수 없음
```

## 3. Attempt와 최종 Execution Status

하나의 Check는 retry로 여러 Attempt를 가질 수 있다.

```text
Check
├─ Attempt 1: FAILED
├─ Attempt 2: COMPLETED
└─ Final Execution Status: COMPLETED
```

최종 성공으로 집계하더라도 Attempt 1의 실패 기록을 삭제하지 않는다. Attempt 모델의 정확한 구조는 후속 Inspection/Tooling 설계에서 정의한다.

## 4. Run Lifecycle State

Inspection Run 자체 lifecycle은 개별 Check의 Execution Status와 구분한다.

```text
CREATED
→ RUNNING
→ FINALIZED
```

### `CREATED`

Resolved Artifact Identity가 확정되어 Run이 생성되었지만 주요 검사 workflow가 본격적으로 진행되지 않은 상태다.

### `RUNNING`

Acquisition, Verification, Inspection, Policy Evaluation 등이 진행 중인 상태다.

### `FINALIZED`

정상 완료 또는 operational failure 등으로 현재 Run에서 더 이상 검사하지 않기로 확정하여 historical record가 된 상태다. `FINALIZED` 자체가 성공을 의미하지 않는다.

## 5. Run Outcome

`Run Outcome`은 Run workflow 자체가 어떤 방식으로 종료되었는지를 나타낸다.

```text
COMPLETED
FAILED
INTERRUPTED
```

### `COMPLETED`

계획된 terminal 상태까지 도달했다는 뜻이며 `ALLOW`를 의미하지 않는다.

```text
Run Outcome: COMPLETED
Policy: ALLOW / BLOCK / MANUAL_REVIEW
```

### `FAILED`

operational failure로 정상적인 terminal 검사 경로를 완료하지 못했다는 의미다.

```text
exact identity resolve
→ Run 생성
→ Artifact acquisition 실패
→ Run Outcome: FAILED
→ Policy Decision: 없음
```

`FAILED`를 자동 `Policy = BLOCK`으로 변환하지 않는다.

### `INTERRUPTED`

사용자 interrupt, process termination 또는 시스템 중단으로 중간 종료된 상태다. 중단 시점까지 확보한 Evidence와 Result는 가능한 범위에서 보존하며 resume/checkpoint는 후속 설계에서 정한다.

## 6. Run Lifecycle과 Run Outcome의 관계

```text
Run Lifecycle State → 현재 Run이 진행 중인가, 최종화되었는가
Run Outcome         → 최종화된 Run이 어떤 방식으로 끝났는가
```

예:

```text
Lifecycle: FINALIZED
Outcome: COMPLETED
Policy: BLOCK

Lifecycle: FINALIZED
Outcome: FAILED
Policy: 없음
```

```text
FINALIZED ≠ COMPLETED
COMPLETED ≠ ALLOW
FAILED ≠ BLOCK
```

## 7. Policy Decision

`Policy Decision`은 Artifact와 검사 결과에 대한 보안 판정이다.

```text
ALLOW
MANUAL_REVIEW
BLOCK
```

`ALLOW`는 현재 Artifact identity/digest와 Run 결과가 현재 Policy에서 자동 반입 허용 조건을 만족한다는 뜻이지 절대적인 안전 보장이 아니다. `MANUAL_REVIEW`는 자동 ALLOW/BLOCK에 충분하지 않거나 사람 판단이 필요한 상태이며 자동 Promotion하지 않는다. `BLOCK`은 현재 결과와 Policy 기준으로 사용자 환경 반입을 허용하지 않는 보안 판정이며 operational failure와 구분한다.

## 8. Policy Decision의 부재

Policy Evaluation까지 도달하지 못한 Run에는 최종 Policy Decision이 없을 수 있다.

```text
Run Outcome: FAILED
Reason: Artifact acquisition failed
Policy Decision: not produced
```

`UNKNOWN`, `FAILED`, `ERROR` 같은 값을 Policy enum에 추가하지 않는다.

```text
Policy Decision이 없음
≠ 새로운 Policy 상태
```

Run operational 상태는 Run Outcome으로 표현한다. 실제 Artifact를 확보한 뒤 필수 검사가 실패·미완료·Unavailable인 경우에는 fail-closed에 따라 `MANUAL_REVIEW` 또는 `BLOCK`을 생성할 수 있다.

## 9. Generic SAFE / UNSAFE 상태를 만들지 않는다

Core에 `SAFE`, `UNSAFE`, `CLEAN`, `MALICIOUS` 같은 범용 최종 상태를 추가하지 않는다. 외부 scanner verdict는 Verification Result, Finding, Evidence로 정규화하고 최종 판정은 Policy가 생성한다.

```text
Scanner verdict: malicious
        ↓
Finding / Evidence
        ↓
Policy Evaluation
        ↓
BLOCK
```

## 10. Operation Status

`Operation Status`는 사용자가 처음 요청한 실제 operation의 종료 상태이며 Policy Decision과 별개의 축이다.

```text
COMPLETED
FAILED
PAUSED
NOT_PERFORMED
```

### `COMPLETED`

사용자 operation이 의도한 최종 상태까지 완료되었다는 의미다.

```text
Operation: install
Status: COMPLETED

Operation: inspect
Status: COMPLETED
Policy: BLOCK
```

두 번째는 inspect 자체가 정상 완료되고 결과가 BLOCK이라는 의미다.

### `FAILED`

operational error로 operation이 완료되지 못했다는 의미다.

```text
Policy: ALLOW
Operation: install
Operation Status: FAILED
Reason: target permission denied
```

Policy Decision은 그대로 `ALLOW`일 수 있다.

### `PAUSED`

사람 판단 또는 추가 입력을 기다리며 후속 진행이 보류된 상태다.

```text
Operation: install
Policy: MANUAL_REVIEW
Operation Status: PAUSED
```

continuation/approval/reinspection은 후속 설계에서 정의한다.

### `NOT_PERFORMED`

Policy 또는 보안 경계에 의해 실제 operation을 진행하지 않았다는 뜻이다.

```text
Operation: install
Policy: BLOCK
Operation Status: NOT_PERFORMED
Reason: policy block
```

`NOT_PERFORMED` 자체가 BLOCK을 뜻하지 않으므로 reason과 Policy를 함께 추적한다.

## 11. Operation Type에 따른 Status 의미

같은 Policy Decision이라도 최초 operation에 따라 Operation Status가 달라진다.

```text
Operation: install + Policy: BLOCK
→ 설치하지 않음 → Operation Status: NOT_PERFORMED

Operation: inspect + Policy: BLOCK
→ 검사는 정상 완료 → Operation Status: COMPLETED
```

Policy만 보고 Operation Status를 결정하지 않고 Original Operation Request와 실제 결과를 함께 사용한다.

## 12. 대표 상태 조합

### 정상 install

```text
Run Lifecycle: FINALIZED
Run Outcome: COMPLETED
Policy: ALLOW
Operation: install
Operation Status: COMPLETED
```

### ALLOW 후 실제 설치 실패

```text
Run Lifecycle: FINALIZED
Run Outcome: COMPLETED
Policy: ALLOW
Operation: install
Operation Status: FAILED
Reason: target permission denied
```

### 수동 검토 필요

```text
Run Lifecycle: FINALIZED
Run Outcome: COMPLETED
Policy: MANUAL_REVIEW
Operation: install
Operation Status: PAUSED
```

### 보안 차단

```text
Run Lifecycle: FINALIZED
Run Outcome: COMPLETED
Policy: BLOCK
Operation: install
Operation Status: NOT_PERFORMED
```

### inspect에서 BLOCK 발견

```text
Run Lifecycle: FINALIZED
Run Outcome: COMPLETED
Policy: BLOCK
Operation: inspect
Operation Status: COMPLETED
```

### Artifact acquisition 실패

```text
Run Lifecycle: FINALIZED
Run Outcome: FAILED
Policy: 없음
Operation: install
Operation Status: FAILED
```

### 필수 Dynamic Inspection 사용 불가

```text
Capability: SUPPORTED
Execution Status: UNAVAILABLE

Run Lifecycle: FINALIZED
Run Outcome: COMPLETED
Policy: MANUAL_REVIEW 또는 BLOCK
```

실제 Policy Decision은 후속 Policy rule에서 결정한다.

### Dynamic Inspection이 필요하지 않은 경우

```text
Execution Status: NOT_EXECUTED
Reason: not required

Run Lifecycle: FINALIZED
Run Outcome: COMPLETED
Policy: ALLOW 가능
```

`NOT_EXECUTED`만으로 fail-closed를 적용하지 않으며 Policy상 검사가 필수였는지가 중요하다.

## 13. Check Requirement와 상태의 구분

검사가 `required`, `optional`, `not applicable`인지 여부는 Execution Status가 아니다.

```text
Check Requirement: REQUIRED
Execution Status: UNAVAILABLE
```

은 보안상 중요한 문제일 수 있고:

```text
Check Requirement: NOT_REQUIRED
Execution Status: NOT_EXECUTED
```

은 정상 상태일 수 있다.

```text
검사가 필요한가?
≠ 검사를 지원하는가?
≠ 실제로 실행됐는가?
```

Check Requirement의 정확한 모델은 Inspection/Policy Contract에서 정의한다.

## 14. Inspection Limitation과 상태의 관계

다음은 Inspection Limitation의 근거가 될 수 있지만 상태 값과 Limitation을 동일하게 취급하지 않는다.

```text
Capability: UNSUPPORTED
Execution Status: INCOMPLETE
Execution Status: UNAVAILABLE
```

예:

```text
Execution Status: INCOMPLETE
Inspection Limitation:
Runtime behavior was observed for only part of the required duration.
```

## 15. Fail-Closed와 상태 모델

Fail-closed는 모든 실패를 무조건 `BLOCK`으로 바꾸는 규칙이 아니다.

```text
Policy상 필요한 보안 검사
        ↓
Capability: UNSUPPORTED

또는

Execution Status:
FAILED / INCOMPLETE / UNAVAILABLE
        ↓
필수 검사 조건 미충족
        ↓
ALLOW 금지
        ↓
Policy rule에 따라 MANUAL_REVIEW 또는 BLOCK
```

반면 acquisition 이전의 일반 operational failure처럼 Artifact 자체를 평가할 수 없는 상태는:

```text
Run Outcome: FAILED
Policy Decision: 없음
```

으로 종료할 수 있다.

```text
fail-closed
≠ 모든 오류를 BLOCK으로 변환
```

## Domain-004에서 확정하는 Invariant

### Invariant 1 — 상태 축을 섞지 않는다

Capability, Execution Status, Run Lifecycle/Outcome, Policy Decision, Operation Status는 독립적인 의미를 가진다.

### Invariant 2 — `SUPPORTED`는 실행 성공을 의미하지 않는다

Capability가 존재해도 Execution은 실패·미완료·Unavailable일 수 있다.

### Invariant 3 — `COMPLETED`는 안전 판정이 아니다

Execution 또는 Run이 완료되었다는 사실만으로 `ALLOW`하지 않는다.

### Invariant 4 — `UNSUPPORTED`와 `UNAVAILABLE`을 구분한다

지원하지 않는 기능과 지원하지만 현재 사용할 수 없는 기능을 같은 상태로 표현하지 않는다.

### Invariant 5 — 실행하지 않은 검사는 이유를 남긴다

`NOT_EXECUTED`에는 왜 실행되지 않았는지 추적 가능한 reason이 필요하다.

### Invariant 6 — Run failure와 `BLOCK`을 구분한다

Operational failure를 임의로 보안 판정인 `BLOCK`으로 변환하지 않는다.

### Invariant 7 — Policy Decision이 없는 Run을 허용한다

Policy Evaluation 이전에 종료된 Run은 Policy Decision을 생성하지 않을 수 있다.

### Invariant 8 — Policy에 `UNKNOWN / ERROR`를 추가하지 않는다

Policy가 생성되지 않은 상태는 Run 상태로 표현한다.

### Invariant 9 — 외부 도구 verdict를 최종 Policy로 사용하지 않는다

외부 verdict는 Verification Result, Finding 또는 Evidence로 정규화한 뒤 Heliopause Policy가 최종 판정을 내린다.

### Invariant 10 — Policy와 Operation Result를 분리한다

`ALLOW`여도 실제 install이 실패할 수 있고, `BLOCK`이어도 inspect operation 자체는 정상 완료될 수 있다.

### Invariant 11 — Fail-closed는 `ALLOW` 금지를 의미한다

필수 보안 검사가 충족되지 않았다고 모든 경우를 자동 `BLOCK`으로 변환하지 않고 Policy가 `MANUAL_REVIEW / BLOCK`을 결정한다.

### Invariant 12 — 검사 필요 여부와 실행 상태를 구분한다

Required/optional/not-required 여부를 Capability나 Execution Status에 혼합하지 않는다.

## 후속 결정으로 유보한 항목

- Go enum/type 이름과 실제 constant 표현
- 상태 serialization 문자열
- Run Outcome의 추가 상태 필요 여부
- cancellation의 별도 Outcome 필요 여부
- Execution failure reason taxonomy
- `NOT_EXECUTED` reason enum
- Operation failure reason taxonomy
- Check Requirement의 정확한 모델
- Stage / Check / Attempt 상태 aggregation rule
- retry 후 최종 Execution Status 계산 방식
- timeout/resource-limit의 `FAILED / INCOMPLETE` 세부 mapping
- Policy rule별 required check 정의
- 상태 간 허용/금지 transition
- invalid state combination validation
- CLI exit code mapping
- CI status mapping
- terminal 표시 문자열
- machine-readable output schema

위 항목은 후속 Verification/Finding/Evidence, Policy, Inspection/Tooling 및 Interface Contract 설계에서 구체화한다.

## 압축된 상태 모델

```text
Capability
SUPPORTED / UNSUPPORTED
        ↓

Execution Status
COMPLETED
FAILED
INCOMPLETE
NOT_EXECUTED
UNAVAILABLE
        ↓

Run
Lifecycle:
CREATED → RUNNING → FINALIZED

Outcome:
COMPLETED / FAILED / INTERRUPTED
        ↓

Policy Decision
ALLOW / MANUAL_REVIEW / BLOCK
또는 Policy 미생성
        ↓

Operation Status
COMPLETED / FAILED / PAUSED / NOT_PERFORMED
```

핵심 원칙은 다음과 같다.

```text
"할 수 있었는가"
≠
"실제로 했는가"
≠
"검사 workflow가 끝났는가"
≠
"보안상 허용되는가"
≠
"사용자 작업이 성공했는가"
```

Heliopause는 상태들을 독립적으로 기록하여 검사 실패, 보안 판정, 사용자 작업 실패를 혼동하지 않고 각 Run이 실제로 어떤 조건에서 무엇을 수행했고 왜 최종 결과에 도달했는지를 추적한다.

## Domain-004 구조화된 결정과 구현 영향

- Capability는 `SUPPORTED / UNSUPPORTED`, Execution Status는 `COMPLETED / FAILED / INCOMPLETE / NOT_EXECUTED / UNAVAILABLE`로 의미를 분리한다.
- `SUPPORTED`는 실행 성공, `COMPLETED`는 안전, `UNSUPPORTED`는 안전을 뜻하지 않는다.
- NOT_EXECUTED에는 reason을 기록하고 Check Requirement와 섞지 않는다.
- Attempt와 최종 Execution Status를 분리하고 최초 실패를 보존한다.
- Run Lifecycle `CREATED → RUNNING → FINALIZED`와 Run Outcome `COMPLETED / FAILED / INTERRUPTED`를 별도 축으로 유지한다.
- Policy Decision은 `ALLOW / MANUAL_REVIEW / BLOCK`만 사용하고 Policy 미생성은 Run operational 상태로 표현한다.
- SAFE/UNSAFE/CLEAN/MALICIOUS generic 상태나 외부 tool verdict를 최종 Policy로 사용하지 않는다.
- Operation Status `COMPLETED / FAILED / PAUSED / NOT_PERFORMED`는 Original Operation과 Policy를 함께 해석하여 결정한다.
- `install + BLOCK`과 `inspect + BLOCK`, `ALLOW + install failure` 같은 조합을 표현한다.
- 필수 검사 미충족은 ALLOW 금지로 처리하되 모든 operational failure를 자동 BLOCK으로 변환하지 않는다.
- 정확한 enum·reason·transition·aggregation·exit code·schema는 후속 Contract/Policy/Tooling 설계에서 확정한다.

## Domain-004 누락 점검

- [x] Capability 의미와 SUPPORTED/UNSUPPORTED
- [x] Capability와 현재 실행 가능성 구분
- [x] Execution Status 기본 5종
- [x] COMPLETED가 안전을 뜻하지 않음
- [x] FAILED·INCOMPLETE·NOT_EXECUTED·UNAVAILABLE 의미
- [x] NOT_EXECUTED reason 추적
- [x] Attempt와 최종 Execution Status
- [x] Run Lifecycle CREATED/RUNNING/FINALIZED
- [x] Run Outcome COMPLETED/FAILED/INTERRUPTED
- [x] Lifecycle와 Outcome 분리
- [x] Policy ALLOW/MANUAL_REVIEW/BLOCK
- [x] Policy Decision 부재와 generic 상태 금지
- [x] 외부 verdict 정규화
- [x] Operation Status COMPLETED/FAILED/PAUSED/NOT_PERFORMED
- [x] Operation Type별 상태 의미
- [x] 대표 상태 조합
- [x] Check Requirement와 상태 분리
- [x] Inspection Limitation과 상태 분리
- [x] fail-closed와 operational failure 구분
- [x] 12개 상태 invariant
- [x] enum·reason·transition·schema·exit code 후속 결정 유보
