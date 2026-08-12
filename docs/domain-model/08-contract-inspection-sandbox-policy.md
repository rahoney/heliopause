# Interface Contract — Inspection, Sandbox, and Policy

## Contract-002: Verification / Inspection / Sandbox / Policy Contracts

### 목적

Heliopause는 Artifact의 검증, 내용·행동 검사, 격리 실행, 최종 보안 판정을 서로 다른 책임으로 분리한다.

```text
Acquired Artifact
      ├─ Verification Port → Verification Result + Evidence
      └─ Inspection Port → 필요 시 Sandbox Port → Observation
                                             ↓
                                    Evidence + Finding
                                             ↓
                                     Policy Contract
                                             ↓
                           ALLOW / MANUAL_REVIEW / BLOCK
```

각 영역은 자신보다 뒤 단계의 최종 판단 책임을 가져가지 않는다.

### Verification Port

`Verification Port`는 Artifact의 **identity·source·integrity·signature·provenance·attestation** 등을 검증하기 위한 계약이다.

```text
Input
├─ Acquired Artifact
├─ source / declared integrity metadata
└─ Verification Context

Output
├─ Verification Result
├─ Evidence
├─ Capability
├─ Execution Status
└─ 필요한 Limitation
```

Integrity, Signature, Provenance, Attestation 등 여러 check가 수행될 수 있다. 구현은 expected digest와 observed digest 비교, signature 및 provenance/attestation 검증, source identity 확인, 외부 verifier 실행, 결과 정규화와 Evidence 생성을 수행할 수 있다.

Artifact 전체의 malicious 여부 최종 판정, Finding severity 정책, `ALLOW` / `MANUAL_REVIEW` / `BLOCK` 결정 및 Promotion은 수행하지 않는다. 외부 verifier 결과도 `Verification Result / Evidence`로 반환하며 Final Policy Decision으로 직접 반환하지 않는다.

### Verification 결과와 실행 실패

Verification Contract는 **검증 결과**와 **검증 작업 실행 상태**를 구분한다.

```text
Execution Status: COMPLETED
Verification Result: digest mismatch

Execution Status: FAILED
Verification Result: 생성되지 않거나 불완전
```

Verifier 실행 실패를 Artifact의 `INVALID` 판정으로 자동 변환하지 않는다.

### Inspection Port

`Inspection Port`는 Artifact의 **정적 구조와 격리된 실행 중 행동을 검사하고 보안상 의미 있는 Finding/Evidence로 해석**하기 위한 계약이다.

```text
Acquired Artifact
        ↓
Inspection Port
        ├─ Static Inspection
        └─ Dynamic Inspection → Sandbox Port → Observation
                                             ↓
                                     Evidence / Finding
```

Inspection은 ecosystem별 또는 scanner별 구현 세부사항을 공통 Domain 결과로 정규화한다.

### Static Inspection

Static Inspection은 Artifact를 실제 Host에서 실행하지 않고 metadata/manifest, archive structure, path traversal, symbolic link, executable content, install/lifecycle script, dependency 이상, credential access pattern, obfuscation, dynamic execution pattern, known vulnerability scanner 결과 등을 검사할 수 있다.

출력은 `Evidence`, `Finding`, `Capability`, `Execution Status`, `Inspection Limitation`에 연결될 수 있다. Static Inspector가 직접 최종 Policy Decision을 반환하지 않는다.

### Dynamic Inspection과 Sandbox Port

Dynamic Inspection은 필요한 Artifact에 대해 Sandbox Port를 사용하여 격리 실행 결과를 검사한다.

```text
Inspection Port → Dynamic Inspection 필요 → Sandbox Port
                                             ↓
                                      Sandbox Session
                                             ↓
                                       Observation
                                             ↓
                                  Inspection 해석
                                             ↓
                                    Evidence / Finding
```

Dynamic Inspection은 Sandbox의 raw observation을 보안 Domain 의미로 해석한다.

```text
Observation: process read synthetic ~/.ssh/id_rsa
        ↓ Inspection
Evidence: synthetic credential read
Finding: Credential Access Attempt
```

Inspection Port는 Sandbox backend 자체 구현, 최종 Policy Decision, Verified Set 승인, Staging, Promotion, Host installation 책임을 갖지 않는다. 외부 scanner verdict는 `Evidence / Finding`으로 정규화한 뒤 Policy에 전달한다.

`Sandbox Port`는 신뢰하지 않는 Artifact를 격리된 환경에서 실행하고 **raw runtime observation과 실행 상태를 제공하는 계약**이다.

```text
Create → Prepare → Introduce Artifact → Execute / Install / Run
      → Observe → Terminate → Dispose
```

각 Dynamic Inspection 실행은 독립적인 ephemeral `Sandbox Session`을 기본으로 한다. Sandbox 구현은 격리 환경 생성, Artifact 투입, 명령/lifecycle 실행, process·filesystem·network/DNS·honeytoken·resource 관찰, timeout/resource limit 적용, process tree 종료와 Session 폐기를 제공할 수 있다.

출력은 주로 `Observation`, `Execution Status`, Sandbox execution metadata다.

```text
Sandbox
→ "무엇이 일어났는가"

Inspection
→ "그 행동이 보안상 무엇을 의미하는가"
```

### Sandbox의 Host 경계와 실패

Sandbox Port는 실제 Host 자산을 기본적으로 신뢰하지 않는 Artifact에 노출하지 않는다.

```text
Host credential / token / .env secret / SSH key
실제 사용자 filesystem / internal network / Host process/service
```

필요하면 synthetic filesystem, dummy credential, honeytoken, fake DNS/HTTP service, sanitized project data, controlled network를 제공한다. Sandbox backend가 Host isolation 경계를 무시할 수 있는 임의의 실행 API를 제공하는 구조를 사용하지 않는다.

비정상 종료, timeout, resource-limit 초과 또는 강제 종료된 Sandbox Session은 재사용하지 않는다.

```text
Session A → abnormal termination → terminate → dispose
Session B → fresh environment
```

기존 실패 Observation/Evidence는 삭제하지 않는다. Sandbox Port는 Finding, Finding Severity, Verification Result, Policy Decision, Verified Set, Promotion을 직접 결정하지 않는다.

### Policy Contract

`Policy`는 외부 security tool adapter가 아니라 Heliopause 내부의 **최종 보안 판정 책임**이다. 반드시 외부 Adapter를 가진 outbound Port일 필요가 없으며, Application이 의존하는 안정적인 **Policy Contract / Domain Service boundary**로 취급한다.

```text
Policy Input
├─ Verification Result
├─ Finding
├─ Evidence / Evidence completeness
├─ Capability
├─ Execution Status
├─ Inspection Limitation
├─ Check Requirement
└─ 현재 검사 대상 identity / Run context

Output: ALLOW / MANUAL_REVIEW / BLOCK
```

Policy Evaluation까지 도달하지 못한 operational failure에서는 Policy Decision 자체가 생성되지 않을 수 있다.

Policy는 필수 Verification과 Inspection의 충족 여부, Finding, 검사 한계, 자동 반입 허용 여부를 평가한다.

```text
Required Dynamic Inspection
Execution Status: UNAVAILABLE
        ↓
ALLOW 금지
        ↓
MANUAL_REVIEW 또는 BLOCK
```

Policy는 Artifact download, signature verification, static/dynamic scanning, Sandbox 실행, Evidence 수집, Staging, Promotion, Host installation을 직접 수행하지 않는다. 특정 scanner/provider의 vendor-specific schema에 직접 의존하지 않는다.

### Policy Decision의 추적성과 Application orchestration

Policy Decision은 주요 판정 근거를 추적 가능하게 해야 한다.

```text
Policy: BLOCK
      ↓
Basis
├─ Finding: Integrity Mismatch
│    └─ Evidence ...
└─ Required Check: COMPLETED

Policy: MANUAL_REVIEW
      ↓
Basis
└─ Required Dynamic Inspection
     Execution Status: UNAVAILABLE
```

내부 모든 rule evaluation step을 공개할 필요는 없지만 최종 판정 이유가 black box가 되어서는 안 된다.

Verification / Inspection / Policy의 상위 workflow 순서는 Application/Workflow가 조정한다. Dynamic Inspection에 필요한 격리 실행은 Inspection 영역이 Sandbox Port를 통해 요청하며, Application이 Sandbox lifecycle의 세부 동작을 직접 제어하지 않는다.

```text
Application
   ├─ Verification Port 호출
   ├─ Inspection Port 호출
   │      └─ 필요 시 Sandbox Port 사용
   └─ Policy Contract 호출
```

Verifier 또는 다른 Provider가 Sandbox·Policy·Promotion을 직접 연쇄 호출하지 않는다.

### Contract-002 Invariant

#### Invariant 1 — Verification과 Inspection을 구분한다

Verification은 identity/integrity/authenticity 계열 검증을 담당하고 Inspection은 content/behavior를 검사한다.

#### Invariant 2 — Verification Result와 Execution Status를 구분한다

검증 실행 성공 여부와 검증 대상의 유효성 결과를 하나의 상태로 합치지 않는다.

#### Invariant 3 — Sandbox는 Observation을 제공한다

Sandbox가 Finding 또는 최종 Policy Decision을 직접 생성하지 않는다.

#### Invariant 4 — Inspection이 Observation을 해석한다

Raw observation과 외부 scanner 결과를 Evidence/Finding으로 정규화한다.

#### Invariant 5 — 외부 도구 verdict는 최종 판정이 아니다

Verifier/scanner/backend의 vendor verdict를 `ALLOW / BLOCK`으로 직접 사용하지 않는다.

#### Invariant 6 — Policy만 최종 보안 판정을 소유한다

최종 `ALLOW / MANUAL_REVIEW / BLOCK`은 Policy 책임이다.

#### Invariant 7 — Policy와 operational failure를 구분한다

Policy Evaluation에 도달하지 못한 실행 실패를 임의로 `BLOCK`으로 변환하지 않는다.

#### Invariant 8 — 필수 검사 미충족은 자동 ALLOW하지 않는다

Required check가 `Capability: UNSUPPORTED`이거나 `Execution Status: FAILED / INCOMPLETE / UNAVAILABLE` 등으로 필수 검사를 충족하지 못한 경우 fail-closed 원칙에 따라 `ALLOW`하지 않는다.

#### Invariant 9 — Sandbox Session은 비정상 종료 후 재사용하지 않는다

필요하면 새로운 독립 Session을 생성한다.

#### Invariant 10 — Application이 workflow를 orchestration한다

각 Adapter/Provider가 다른 영역의 책임까지 연쇄적으로 수행하지 않는다.

#### Invariant 11 — Policy는 vendor-specific 구현에 의존하지 않는다

공통 Domain Result와 상태만을 입력으로 사용한다.

### 후속 결정으로 유보한 항목

- 실제 Go interface와 method signature
- Verification/Inspection/Sandbox/Policy request-result struct
- Verification Type, Inspection Check, Check Requirement, Observation taxonomy
- provider/tool registration과 Sandbox backend 선택
- timeout/resource-limit, retry/attempt, cancellation API
- Evidence writer 연결 방식
- Policy rule representation, versioning, reason code
- concurrent inspection 방식

### 압축된 Contract

```text
Verification Port
"Artifact의 identity/integrity/authenticity를 검증하라."
        ↓
Verification Result + Evidence

Inspection Port
"Artifact의 content/behavior를 검사하고 해석하라."
        ↓
Evidence + Finding

Sandbox Port
"격리해서 실행하고 실제로 무엇이 일어났는지 관찰하라."
        ↓
Observation + Execution Status

Policy Contract
"모든 정규화된 검사 결과를 바탕으로 반입 여부를 결정하라."
        ↓
ALLOW / MANUAL_REVIEW / BLOCK
```

```text
Verification
≠ Inspection
≠ Sandbox
≠ Policy
```

각 책임을 분리하고 Application이 이 계약들을 조합하여 전체 Inspection Workflow를 수행하는 것을 기본 원칙으로 한다.

## Contract-002 구조화된 결정과 구현 영향

- Verification은 identity·integrity·authenticity 검증 결과와 Evidence를, Inspection은 content·behavior 검사로부터 Finding/Evidence를 생성한다.
- Verification Result와 각 check의 Execution Status를 독립적으로 기록하며 verifier 실행 실패를 `INVALID`로 임의 변환하지 않는다.
- Dynamic Inspection은 Sandbox의 raw Observation을 받아 보안 Domain 의미의 Evidence/Finding으로 해석한다.
- Sandbox는 ephemeral Session에서 관찰과 실행 상태만 반환하고 Finding·severity·Policy Decision을 생성하지 않는다.
- Sandbox에는 실제 Host credential·secret·filesystem·internal network·process를 노출하지 않으며 합성·정제된 입력만 제공한다.
- 비정상 종료된 Sandbox Session은 폐기하며 기존 Observation/Evidence를 보존하고 추가 실행에는 새 Session을 사용한다.
- 외부 verifier/scanner/backend verdict는 정규화 결과일 뿐 최종 Policy Decision이 아니다.
- Policy는 공통 Domain Result, check requirement, limitation, Run context를 평가하여 최종 `ALLOW`·`MANUAL_REVIEW`·`BLOCK`만 결정한다.
- Policy Evaluation 이전의 operational failure와 Policy Decision을 구분하며 required check 미충족 시 자동 `ALLOW`하지 않는다.
- Application/Workflow가 Port 호출 순서를 조정하고, 각 Port가 Sandbox·Policy·Promotion을 임의로 연쇄 호출하지 않는다.
- 실제 interface/schema·taxonomy·backend·retry/cancellation·Evidence writer·Policy rule/version/reason code는 후속 설계로 유보한다.

## Contract-002 누락 점검

- [x] Verification Port 책임과 입력/출력
- [x] Verification Result와 Execution Status 구분
- [x] Static / Dynamic Inspection 구분
- [x] Observation의 Evidence/Finding 해석 책임
- [x] Inspection Port의 금지 책임
- [x] Sandbox lifecycle과 ephemeral Session
- [x] Sandbox의 raw observation 책임
- [x] Sandbox Host 경계와 합성 입력
- [x] 비정상 종료 Session 폐기와 Observation 보존
- [x] Sandbox Port의 금지 책임
- [x] Policy Contract의 내부 Domain Service 경계
- [x] Policy 입력·출력과 operational failure 구분
- [x] required check 미충족의 fail-closed
- [x] Policy Decision 근거 추적
- [x] Application의 orchestration 책임
- [x] 11개 invariant
- [x] 후속 결정 항목 유보
