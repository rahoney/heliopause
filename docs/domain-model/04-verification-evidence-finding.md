# Domain Model — Verification, Evidence, and Finding

## Domain-005: Verification Result / Evidence / Finding 모델

### 사용자 원문

Heliopause는 Artifact 검증·검사 과정에서 생성되는 `Verification Result`, `Evidence`, `Finding`을 서로 다른 의미의 Domain Concept로 구분한다.

```text
Verification
├─ Verification Result
└─ Evidence

Static Inspection
├─ Evidence
└─ Finding

Sandbox
        ↓
Observation
        ↓
Dynamic Inspection
├─ Evidence
└─ Finding

Verification Result + Finding + Evidence + Capability
+ Execution Status + Inspection Limitation
        ↓
Policy Evaluation
        ↓
Policy Decision
```

```text
Verification Result → 검증 항목에 대해 무엇을 확인했는가
Evidence            → 실제로 무엇을 관찰·측정·검증했는가
Finding             → 그 Evidence가 보안상 무엇을 의미하는가
Policy Decision     → 그 결과 Artifact 반입을 허용할 것인가
```

세 개념을 최종 Policy Decision과 동일하게 사용하지 않는다. Domain-005에서는 정확한 Go struct, field, enum, severity taxonomy, Evidence 저장 schema 및 외부 도구 mapping을 확정하지 않고 역할·관계·추적 invariant를 정의한다.

## 1. Verification Result

`Verification Result`는 Artifact의 identity, source, integrity, signature, provenance, attestation 등 검증 영역에서 확인한 정규화 결과다.

```text
Source / Identity Verification
Digest / Integrity Verification
Registry Integrity Verification
Signature Verification
Provenance Verification
Attestation Verification
```

하나의 Artifact에 여러 Result가 존재할 수 있다.

```text
Acquired Artifact
├─ Integrity Verification Result
├─ Signature Verification Result
└─ Provenance Verification Result
```

모든 Verification 종류가 모든 Artifact에 적용되는 것은 아니다.

### Verification Result와 Execution Status의 구분

Result는 무엇이 확인되었는지, Execution Status는 검증 작업이 어떻게 실행되었는지를 표현한다.

```text
Digest Verification
Execution Status: COMPLETED
Result: expected digest matches observed digest

Digest Verification
Execution Status: COMPLETED
Result: digest mismatch
```

`COMPLETED`는 검증 절차가 정상 완료되어 해석 가능한 결과를 얻었다는 뜻이지 검증 성공이 아니다. verifier가 crash하면 `Execution Status: FAILED`일 수 있으며 이를 `Signature: INVALID`로 임의 변환하지 않는다.

```text
검증 도구 실행 실패
≠
Artifact 검증 실패
```

### 검증 정보 부재와 검증 실패 구분

Signature/provenance/attestation이 없거나, 존재하지만 invalid이거나, 존재 여부 확인 불가·Verifier 실행 불가는 서로 구분한다.

```text
검증 가능한 정보가 존재함
검증 정보가 존재하지 않음
검증 결과가 일치함
검증 결과가 불일치함
유효함
유효하지 않음
판단할 수 없음
```

정확한 result type은 후속 Verification 설계에서 정의한다.

```text
Absent ≠ Invalid ≠ Execution Failed ≠ Unsupported
```

Signature/provenance 부재만으로 자동 BLOCK하지 않으며 Policy상 필수인지에 따라 판단한다.

### Verification Result 정규화

외부 verifier·registry의 vendor-specific 결과를 Core에 그대로 사용하지 않는다.

```text
External Provider Result / Raw Output
        ↓
Provider / Adapter normalization
        ├─ Verification Result
        └─ Evidence
```

원본 출력은 필요하면 Raw Evidence로 보존하지만 Policy가 vendor schema에 직접 의존하지 않게 한다.

### Verification Result의 대상

최소한 다음에 연결 가능해야 한다.

```text
Inspection Run + Artifact identity + Observed Digest + Verification Type
```

동일 name/version이라도 digest가 다른 Artifact의 Result를 재사용하지 않는다. dependency Result도 exact identity/digest와 검증 Run에 연결한다.

## 2. Evidence

`Evidence`는 Verification 또는 Inspection에서 실제로 확인·관찰·측정한 사실과 판단 근거다.

```text
Observed digest = AAA
Expected digest = AAA

process attempted to read:
synthetic ~/.ssh/id_rsa

DNS request: example.invalid
child process created: sh -c ...
archive contained: ../../target
```

Evidence 자체는 최종 판정이 아니다.

```text
Evidence: process accessed synthetic credential
        ↓
Finding: credential access attempt
        ↓
Policy: BLOCK 가능
```

### Evidence의 출처

Evidence는 생성 source를 추적할 수 있어야 한다.

```text
Artifact Adapter
Verifier
Static Inspector
External Scanner
Sandbox Observer
Runtime Monitor
SBOM Generator
Heliopause 자체 Integrity Check
```

최소한 어떤 Run에서, 어떤 Artifact에 대해, 어떤 Check/Tool/Observation으로부터 무엇을 확인했는지 추적 가능해야 한다. 구체적인 Source schema는 후속 Evidence/Result Contract에서 정의한다.

### Raw Evidence와 정규화 Evidence

```text
Raw Evidence
→ tool output / log / telemetry / observation 원본

        ↓ normalize / extract

Evidence
→ Heliopause가 추적 가능한 형태로 정규화한 사실
```

예:

```text
Raw Sandbox Observation
        ↓
Evidence
ProcessAccess
target = synthetic credential
action = read
```

Raw Evidence는 감사·재현·심층 분석을 위해 Evidence Store에 보존할 수 있고 정규화 Evidence는 Raw Evidence reference를 가진다. storage format과 대용량 telemetry 정책은 후속 Evidence/Storage 설계에서 결정한다.

### Evidence와 Observation의 구분

```text
Sandbox Session
      ↓
Observation
      ↓
Inspection
      ↓
Evidence
      ↓
Finding
```

모든 Observation이 Finding을 생성하지는 않는다.

```text
Observation: process opened /tmp/file
→ 정상 행동 → Finding 없음
```

감사·검사 근거로 필요하면 Evidence로 보존할 수 있다. Sandbox는 Observation을 Finding이나 Policy로 직접 판정하지 않는다.

### Evidence의 불변성과 무결성

신뢰하지 않는 Artifact/Sandbox process가 Evidence를 수정·삭제할 수 없어야 하며 trusted observer/Evidence Store 경계를 유지한다.

```text
Untrusted Artifact
        X
        ↓
Evidence 수정 / 삭제
```

기록 후 historical meaning을 임의 변경하지 않는다. 정정·추가 정보는 기존 Evidence를 덮어쓰기보다 새 Evidence 또는 명시적 관계로 표현한다. hashing, append-only storage 등 구현은 후속 Evidence/Storage 설계에서 정의한다.

## 3. Finding

`Finding`은 하나 이상의 Evidence 또는 정규화된 검사 결과를 보안 관점에서 해석한 결과다.

```text
Evidence
process read synthetic ~/.ssh/id_rsa
        ↓
Finding
Category: Credential Access
Meaning: Artifact attempted to access credential-like data.
```

또는 archive path traversal 위험처럼 해석할 수 있다. 가능한 경우 근거 Evidence를 참조한다.

```text
Finding
        ↓
Evidence Reference 1
Evidence Reference 2
...
```

### Finding은 단순 Raw Tool Verdict가 아니다

외부 scanner의 `malicious`, `high risk`, `suspicious` verdict를 그대로 Finding이나 Policy로 복사하지 않는다.

```text
External Tool Output
        ↓
Raw Evidence
        ↓ 정규화
Finding
```

외부 verdict는 Evidence source로 활용할 수 있으나 category와 의미는 공통 Domain Model로 정규화한다.

### Finding Severity

개념적으로 `Informational`, `Low`, `Medium`, `High`, `Critical` 등이 가능하지만 정확한 taxonomy는 후속 설계에서 결정한다.

```text
Finding Severity
≠
Policy Decision
```

`HIGH`가 반드시 BLOCK은 아니며 Policy가 다른 Result·Finding·Evidence·검사 완전성·규칙을 함께 고려한다. `WARN`도 최종 Decision이 아니라 Finding의 사용자-facing 경고 수준이다.

### Finding이 없는 것과 안전의 구분

```text
Finding 없음
≠ SAFE
```

Dynamic Inspection이 `UNAVAILABLE`이면 Finding이 없는 것이 위험 행동이 없다는 뜻이 아니다. Policy는 Finding 외에 Capability, Execution Status, Inspection Limitation, Verification Result, Evidence completeness를 함께 고려한다.

### 하나의 Evidence와 여러 Finding

하나의 Evidence가 여러 Finding을 지원하거나 여러 Evidence가 하나의 Finding을 지원할 수 있다.

```text
Evidence: Artifact spawned shell and attempted outbound connection
        ↓
Finding A: Unexpected Shell Execution
Finding B: Unexpected Outbound Network Attempt
```

```text
Evidence A + Evidence B + Evidence C
        ↓
Finding
```

Evidence와 Finding을 1:1로 제한하지 않는다.

### Verification Result와 Finding의 관계

Verification Result가 보안상 의미를 가지면 Finding을 생성할 수 있지만 모든 Result가 Finding을 가져야 하는 것은 아니다.

```text
Expected digest AAA / Observed digest BBB
→ Verification Result: mismatch
→ Finding: Artifact Integrity Mismatch
```

반면 expected와 observed가 일치하는 정상 Result는 위험 Finding 없이 기록할 수 있다.

```text
Verification Result → 검증 사실과 결과
Finding             → 그 결과 중 보안상 주목할 의미
```

### Finding과 Policy의 구분

Finding은 Policy input 중 하나이며 Policy를 직접 결정하지 않는다.

```text
Scanner / Inspector
→ Evidence
→ Finding
→ Policy
→ BLOCK
```

검사 모듈이 직접 `BLOCK`을 반환하지 않는다.

## 4. Evidence와 Policy Decision의 추적성

최종 Policy Decision은 가능한 경우 어떤 Verification Result, Finding, Evidence 및 상태를 근거로 했는지 추적할 수 있어야 한다.

```text
Policy Decision: BLOCK
        ↓
Reason / Basis
        ├─ Finding A → Evidence 1
        ├─ Finding B → Evidence 2, Evidence 3
        └─ Required Check → Execution Status: INCOMPLETE
```

Policy evaluation trace를 기본 terminal에 모두 출력할 필요는 없지만 최종 판정의 근거가 불투명한 black box가 되어서는 안 된다. reason/reference schema는 후속 Policy 설계에서 정의한다.

## 5. Evidence가 없는 Finding

직접 생성하는 보안 Finding은 가능한 경우 Evidence에 근거해야 한다. 직접 Raw Evidence가 없더라도 정규화된 Verification Result, Inspection Result, Execution/Capability state를 근거로 삼을 수 있다.

```text
Finding → 반드시 Raw Evidence 하나 직접 참조
```

로 강제하지는 않지만:

```text
Finding → 판단 근거에 추적 가능
```

이라는 invariant를 유지한다. 근거가 전혀 없는 Finding은 정상적인 Policy input으로 생성하지 않는다.

## 6. Inspection Limitation과 Finding의 구분

검사 한계를 반드시 위험 Finding으로 변환하지 않는다.

```text
Inspection Limitation:
macOS-specific runtime behavior not inspected

≠

Finding:
malicious behavior detected
```

Policy는 Limitation을 직접 입력으로 받을 수 있고 필요하면 사용자-facing 경고 Finding을 추가할 수 있다.

```text
검사하지 못한 것
≠
위험 행동을 발견한 것
```

## 7. Verification / Evidence / Finding의 Artifact 연결

모든 결과는 package name이 아니라 실제 검사 대상에 연결되어야 한다.

```text
Inspection Run
      ↓
Acquired Artifact
      ↓
Resolved Artifact Identity + Observed Digest
```

를 기준으로 Verification Result, Evidence, Finding을 추적한다. dependency 결과도 해당 dependency identity/digest와 검증 Run에 연결하며 다른 Artifact/digest 결과를 현재 Artifact 근거로 임의 재사용하지 않는다.

## Domain-005에서 확정하는 Invariant

### Invariant 1 — Verification Result / Evidence / Finding을 구분한다

```text
Verification Result → 검증 결과
Evidence            → 관찰·검증 사실
Finding             → 보안적 해석
```

을 같은 객체나 상태로 사용하지 않는다.

### Invariant 2 — Verification Result와 Execution Status를 구분한다

검증 도구 정상 실행과 Artifact 검증 결과 유효성은 다른 의미다.

### Invariant 3 — 정보 부재와 invalid를 구분한다

Signature/provenance 부재, invalid 결과, 검증 자체 실패를 동일하게 표현하지 않는다.

### Invariant 4 — 외부 도구 결과를 정규화한다

vendor-specific output이나 verdict를 Core Policy가 직접 사용하지 않는다.

### Invariant 5 — Evidence는 실제 검사 대상에 추적 가능해야 한다

Evidence는 Inspection Run, exact Artifact identity/digest 및 생성 source에 연결 가능해야 한다.

### Invariant 6 — Raw Evidence와 정규화 Evidence를 구분한다

대용량 원본 output/telemetry를 Core 정규화 Evidence와 동일시하지 않는다.

### Invariant 7 — Finding은 판단 근거에 추적 가능해야 한다

Finding은 Evidence, Verification Result 또는 기타 명시적 검사 결과에 근거해야 한다.

### Invariant 8 — Evidence와 Finding은 1:1 관계가 아니다

하나의 Evidence가 여러 Finding을 지원하거나 여러 Evidence가 하나의 Finding을 지원할 수 있다.

### Invariant 9 — Finding Severity와 Policy Decision을 분리한다

High/Critical/WARN 등의 위험 표현을 `ALLOW / MANUAL_REVIEW / BLOCK`과 동일하게 사용하지 않는다.

### Invariant 10 — Finding이 없다는 사실은 안전 판정이 아니다

검사 미수행·실패·미완료에서도 Finding이 없을 수 있으므로 Finding 부재를 `ALLOW` 근거로 단독 사용하지 않는다.

### Invariant 11 — Inspection Limitation과 Finding을 구분한다

검사하지 못한 범위와 실제 위험 행동 발견을 같은 의미로 취급하지 않는다.

### Invariant 12 — Final Policy는 Policy 책임이다

Verifier, scanner, inspector 또는 Sandbox가 최종 `ALLOW / MANUAL_REVIEW / BLOCK`을 직접 결정하지 않는다.

## 후속 결정으로 유보한 항목

- Verification Result의 정확한 Go type
- Verification Type taxonomy
- Verification outcome enum
- Signature/provenance/attestation result schema
- Finding category taxonomy
- Finding severity enum
- confidence 또는 certainty 모델
- Evidence ID format
- Finding ID format
- Evidence Source schema
- Evidence type taxonomy
- Raw Evidence format
- Evidence reference schema
- Evidence integrity/hash 방식
- Evidence Store layout
- Raw Evidence retention
- Observation → Evidence normalization schema
- external scanner/verifier mapping rule
- Finding deduplication/aggregation
- Finding suppression/acknowledgement
- Policy Decision reason/reference schema
- terminal warning 표시 방식
- machine-readable Evidence/Finding schema

위 항목은 후속 Policy, Evidence/Result, Verification/Inspection 및 Interface Contract 설계에서 구체화한다.

## 압축된 모델

```text
Acquired Artifact
      │
      ├─ Verification
      │     ├─ Verification Result
      │     └─ Evidence
      │
      ├─ Static Inspection
      │     ├─ Evidence
      │     └─ Finding
      │
      └─ Dynamic Inspection
              ↓
          Sandbox
              ↓
          Observation
              ↓
       Inspection 해석
          ├─ Evidence
          └─ Finding

Verification Result
+ Evidence
+ Finding
+ Capability
+ Execution Status
+ Inspection Limitation
        ↓
Policy Evaluation
        ↓
ALLOW / MANUAL_REVIEW / BLOCK
```

```text
Evidence
= "무엇을 실제로 확인했는가"

Finding
= "그 사실이 보안상 무엇을 의미하는가"

Verification Result
= "검증 대상에 대해 어떤 결론을 얻었는가"

Policy Decision
= "그 모든 결과를 기준으로 실제 반입을 허용할 것인가"
```

Heliopause는 외부 도구 verdict나 단일 Finding을 그대로 최종 판정으로 사용하지 않고, 실제 Artifact identity/digest에 연결된 Verification Result·Evidence·Finding과 검사 상태를 정규화하여 추적 가능한 Policy Decision의 근거로 사용한다.

## Domain-005 구조화된 결정과 구현 영향

- Verification Result는 identity/source/integrity/signature/provenance/attestation 검증을 정규화하고 Execution Status와 분리한다.
- Absent·Invalid·Execution Failed·Unsupported를 구분하고 정보 부재만으로 자동 BLOCK하지 않는다.
- 외부 provider raw output을 Raw Evidence로 보존할 수 있지만 Core/Policy는 vendor schema에 직접 의존하지 않는다.
- Evidence는 Run·Artifact identity/digest·Check/Tool/Observation source와 연결하고 trusted observer/Evidence Store가 무결성을 관리한다.
- Observation은 raw runtime 관찰, Evidence는 정규화된 사실, Finding은 보안 해석으로 분리한다.
- Finding은 가능한 경우 Evidence/Verification/Inspection 결과에 근거하고 Evidence↔Finding 다대다 관계를 허용한다.
- Finding severity/WARN은 Policy Decision과 분리하며 Finding 부재를 ALLOW로 단독 해석하지 않는다.
- Inspection Limitation은 Finding과 분리하지만 Policy input으로 사용한다.
- Verification Result·Evidence·Finding·Capability·Execution Status·Limitation은 실제 Artifact identity/digest와 Run에 추적 가능해야 한다.
- 최종 `ALLOW / MANUAL_REVIEW / BLOCK`은 Policy만 생성한다.
- 정확한 type·taxonomy·schema·mapping·storage·retention·CLI output은 후속 설계에서 확정한다.

## Domain-005 누락 점검

- [x] Verification Result / Evidence / Finding 의미 분리
- [x] Verification Result와 Execution Status 분리
- [x] 검증 정보 부재·invalid·실행 실패·unsupported 분리
- [x] 외부 verifier/registry 결과 정규화
- [x] 실제 Artifact identity/digest·Run 추적
- [x] Evidence source 추적
- [x] Raw Evidence와 정규화 Evidence 분리
- [x] Observation과 Evidence 분리
- [x] Evidence trusted observer·무결성 원칙
- [x] Finding의 Evidence/Result 근거 추적
- [x] Raw tool verdict와 Finding 분리
- [x] Finding severity와 Policy Decision 분리
- [x] Finding 부재와 안전 판정 분리
- [x] Evidence/Finding 다대다 관계
- [x] Verification Result와 Finding 선택적 관계
- [x] Finding과 Policy 책임 분리
- [x] Policy Decision 근거 추적성
- [x] Evidence 없는 Finding의 근거 추적 원칙
- [x] Inspection Limitation과 Finding 분리
- [x] 외부 결과의 Artifact 연결
- [x] 12개 invariant
- [x] type·taxonomy·schema·storage·mapping·output 후속 결정 유보
