# Domain Model — Core Concepts

## Domain-001: 핵심 Domain Concept 및 역할

### 사용자 원문

Heliopause의 Domain Model은 특정 package manager, registry, scanner, sandbox 구현 또는 Host OS에 종속된 개념이 아니라, Software Artifact를 식별·획득·검증·검사·판정하고 검증된 대상을 안전하게 반입하는 전체 lifecycle을 표현할 수 있는 공통 Domain Concept를 중심으로 구성한다.

Domain-001에서는 각 개념의 정확한 field, enum, schema 또는 Go type을 확정하지 않고, 어떤 개념을 독립적으로 구분하여 모델링해야 하는지와 각 개념의 기본 역할만 정의한다.

모든 Domain Concept를 동일한 종류의 Entity로 취급하지 않는다. 이후 세부 Domain Model에서 lifecycle과 identity를 가지는 Entity/Aggregate, 변경 불가능한 Value Object, 검사 결과 Record 등으로 구체적으로 분류한다.

### 핵심 Domain Concept

```text
사용자 요청
   ↓
Operation Request
   ↓
Artifact Reference
   ↓
Resolved Artifact Identity
   ↓
Inspection Run 생성
   ↓
Acquisition / Acquired Artifact binding
   ├─ Verification Result
   ├─ Finding
   ├─ Evidence
   ├─ Capability
   ├─ Execution Status
   └─ Inspection Limitation
   ↓
Policy Decision
   ├─ ALLOW + inspect
   │       ↓
   │   Promotion 없음
   │       ↓
   │   Operation Result
   │
   ├─ ALLOW + install
   │       ↓
   │   ├─ Package ecosystem
   │   │      ↓
   │   │  Verified Set → Verified Manifest
   │   │      ↓
   │   │  trusted Promotion
   │   │
   │   └─ Standalone
   │          ↓
   │      exact identity / digest 재확인
   │          ↓
   │      trusted Promotion
   │
   │       ↓
   │   Operation Result
   │
   ├─ MANUAL_REVIEW
   │       ↓
   │   자동 Promotion 보류
   │       ↓
   │   Operation Result
   │
   └─ BLOCK
           ↓
       Promotion 없음
           ↓
       Operation Result
```

Sandbox 기반 Dynamic Inspection이 필요한 경우에는 별도의 `Sandbox Session`과 `Observation`이 검사 과정에 참여한다.

### 1. `Operation Request`

사용자가 Heliopause에 처음 요청한 작업의 의도와 실행 context를 나타낸다.

예시 `helox npm install kenv`에서 `npm`, `install`, `kenv` 및 향후 허용되는 설치 대상 환경·범위·option 등이 원래 사용자 요청이다. 검사 workflow 동안 유지되며 `install + ALLOW`인 경우 최종 Promotion이 원래 작업을 이어서 수행하도록 한다.

```text
Operation Request
→ 사용자가 무엇을 하려고 했는가

Operation Result는 install뿐 아니라 inspect를 포함한 사용자가 요청한 operation 전체의 실행 상태를 나타내며, MANUAL_REVIEW나 BLOCK으로 Promotion이 수행되지 않은 경우에도 생성될 수 있다.
```

### 2. `Artifact Reference`

사용자가 입력했거나 외부 source를 가리키는 아직 완전히 확정되지 않은 Artifact 표현이다.

```text
kenv
kenv@latest
requests
owner/repository + release/asset reference
```

`latest`, version 생략 등 mutable하거나 불완전한 입력을 포함할 수 있으므로 최종 검증 대상 identity로 사용하지 않는다.

```text
Artifact Reference
→ 사용자가 무엇을 가리켰는가
```

### 3. `Resolved Artifact Identity`

Artifact Reference를 resolve한 뒤 검사 대상으로 확정된 정확한 논리적 Artifact identity다. package name·exact version·repository·release/tag·asset 등 생태계별 표현은 다를 수 있지만 Core에서는 공통 identity 개념으로 다룬다.

```text
kenv@latest
      ↓ resolve
kenv@<exact-version>
```

mutable reference와 구분하며 실제 bytes 획득 후 확정되는 digest와의 관계는 후속 Artifact Identity 모델에서 구체화한다.

```text
Resolved Artifact Identity
→ 정확히 어떤 논리적 Artifact를 획득할 것인가
```

### 4. `Acquired Artifact`

Controlled Intake를 통해 실제로 획득한 검사 대상 Software Artifact다. resolved identity, 실제 bytes/content 및 bytes를 식별하는 digest가 연결된 상태를 표현해야 한다.

```text
Resolved Artifact Identity + 실제 획득한 Artifact + digest
        ↓
Acquired Artifact
```

Verification, Inspection 및 이후 Verified Set은 실제 획득된 Artifact와 identity/digest를 기준으로 수행한다.

```text
Acquired Artifact
→ 실제로 어떤 것을 검사하고 있는가
```

### 5. `Inspection Run`

하나의 Heliopause 검사 실행을 나타내는 핵심 lifecycle 단위다. Artifact의 exact `Resolved Artifact Identity`가 확정된 이후 acquisition 전에 생성하며, 이후 Acquisition, Acquired Artifact binding, Verification, Inspection, Evidence 수집, Finding 생성, Policy Decision 등 하나의 검사 과정에서 발생한 결과를 연결한다.

```text
Inspection Run
├─ 검사 대상
├─ Verification Result
├─ Finding
├─ Evidence
├─ Capability
├─ Execution Status
├─ Inspection Limitation
└─ Policy Decision
```

완료된 Run은 후속 재검사로 덮어쓰지 않는다. 재검사에는 새로운 Run을 생성하고 필요한 경우 이전 Run과의 관계를 기록한다.

```text
Inspection Run
→ 이번 검사에서 실제로 무슨 일이 있었는가
```

### 6. `Verification Result`

Artifact가 무엇이며 어디에서 왔는지, 획득한 대상이 기대한 identity와 무결성을 만족하는지를 검증한 결과다.

연결 가능한 Verification 결과는 source/identity, digest/integrity, registry integrity information, signature, provenance, attestation이다. vendor-specific output을 그대로 Core 최종 결과로 사용하지 않고 공통 `Verification Result`로 정규화한다.

```text
Verification Result
→ 이것이 무엇이고 어디에서 왔으며 무결성이 확인되는가
```

### 7. `Evidence`

검증·검사 과정에서 확보한 관찰 사실과 판단 근거다.

```text
특정 파일 접근 기록
process 실행 기록
network connection attempt
digest verification 결과
signature verification 자료
scanner output 일부
Sandbox observation
```

Evidence는 Finding과 Policy Decision의 근거를 추적하도록 Inspection Run 및 Artifact와 연결한다. Raw Evidence와 정규화된 Evidence의 구조는 후속 Domain Model에서 정의한다.

```text
Evidence
→ 실제로 무엇을 관찰하거나 검증했는가
```

### 8. `Finding`

Evidence와 검사 결과를 보안 관점에서 해석하여 정규화한 결과다.

```text
Evidence
process attempted to read synthetic credential file

        ↓

Finding
credential access attempt
```

가능한 경우 근거 Evidence를 참조하며 severity와 위험 표현 방식은 후속 Policy/Domain Model에서 정의한다. `WARN`은 최종 Policy Decision이 아니라 Finding의 경고 수준이다.

```text
Finding
→ 관찰된 사실이 보안상 무엇을 의미하는가
```

### 9. `Capability`

`Capability`는 특정 Verification 또는 Inspection check를 현재 Artifact·platform·검사 환경에서 수행할 수 있는지를 나타낸다. Adapter 자체가 일반적으로 어떤 기능을 제공하는지는 별도의 Adapter capability/support 정보로 구분하며 Run-level `Capability`와 동일한 개념으로 취급하지 않는다.

```text
Supported
Unsupported
```

정확한 상태 값은 후속 Status Model에서 확정한다. Capability는 실제 검사의 성공 여부를 의미하지 않는다.

```text
Capability
→ 이 검사를 수행할 수 있는가
```

### 10. `Execution Status`

특정 Verification 또는 Inspection 작업이 실제로 어떻게 수행되었는지를 나타낸다.

```text
Completed
Failed
Incomplete
Not Executed / Skipped
Unavailable
```

정확한 enum은 후속 Status Model에서 확정하며 Execution Status는 Artifact 안전성을 직접 의미하지 않는다.

```text
Execution Status
→ 검사가 실제로 어떻게 끝났는가
```

### 11. `Inspection Limitation`

검사에서 확인하지 못했거나 현재 backend/capability의 한계로 보장할 수 없는 범위를 명시적으로 표현한다.

```text
Linux backend에서는 macOS-specific runtime behavior를 검증하지 못함
필요한 verifier가 unavailable
특정 dynamic inspection을 수행하지 못함
```

검사 한계를 흩어진 로그 문자열이 아니라 Inspection Run과 Policy가 참조할 Domain Concept로 취급한다.

```text
Inspection Limitation
→ 이번 검사로 무엇을 확인하지 못했는가
```

### 12. `Policy Decision`

정규화된 Verification Result, Finding, Evidence, Capability, Execution Status 및 Inspection Limitation 등을 바탕으로 내리는 최종 반입 가능 여부다.

```text
ALLOW
MANUAL_REVIEW
BLOCK
```

외부 scanner/verifier의 verdict는 Heliopause의 `Policy Decision`과 동일하지 않다.

```text
Policy Decision
→ 이 검사 결과를 기준으로 Artifact를 반입할 수 있는가
```

### 13. `Verified Set`

Package ecosystem에서 최종 Promotion에 사용할 Primary Artifact와 실제 dependency의 정확한 검증 완료 집합이다.

단순 dependency 목록이 아니며 각 구성요소는 exact identity/digest 및 이를 검증한 Inspection Run에 추적 가능해야 한다.

```text
Verified Set
├─ Primary Artifact
├─ Dependency A
├─ Dependency B
└─ Dependency C
```

Verified Set 밖의 새로운 Artifact가 Promotion/install 과정에서 필요해지면 자동으로 추가하지 않는다.

```text
Verified Set
→ 실제로 Promotion할 수 있도록 검증된 정확한 Artifact 집합
```

### 14. `Verified Manifest`

Verified Set을 Promotion과 감사 과정에서 재확인할 수 있도록 표현한 검증 완료 Manifest다. Verified Set, Artifact identity/digest 및 관련 Inspection Run과 일관되게 연결되어야 한다.

```text
Inspection Run
      ↓
Policy = ALLOW
      ↓
Verified Set
      ↓
Verified Manifest
      ↓
Promotion
```

Verified Set이 실제 검증 완료 집합이라는 Domain 개념이라면 Verified Manifest는 그 집합을 전달·확인·추적하기 위한 명시적인 검증 기록 표현이다. 구체적인 schema와 file format은 후속 Domain/Evidence 설계에서 결정한다.

### 15. `Operation Result`

최종 사용자가 요청한 실제 작업의 실행 결과이며 `Policy Decision`과 구분한다.

```text
Policy Decision
→ Artifact의 보안 판정

Operation Result
→ 사용자가 요청한 operation 전체의 실제 실행 결과
```

다음과 같은 상태가 가능하다.

```text
Policy: ALLOW
Operation: install
Result: FAILED
Reason: permission denied
```

Artifact가 검사 정책을 통과했더라도 filesystem, 권한 또는 package manager 문제로 설치가 실패할 수 있다. 정확한 Operation Status와 failure taxonomy는 후속 Status Model에서 결정한다.

### Sandbox 관련 Domain Concept

Dynamic Inspection 실행 상태는 Core Artifact 모델과 구분하여 Sandbox/Inspection 경계의 Domain Concept로 관리한다.

#### `Sandbox Session`

하나의 독립적인 격리 실행 lifecycle이다.

```text
Create
→ Prepare
→ Artifact Intake
→ Install / Execute
→ Observe
→ Terminate
→ Dispose
```

비정상 종료, timeout, resource limit 초과 등이 발생한 Session은 재사용하지 않는다.

#### `Observation`

Sandbox가 외부 observer를 통해 수집한 file/process/network/resource/honeytoken 등의 원시 실행 관찰 결과다.

Sandbox 자체는 Observation을 최종 Finding으로 판정하지 않는다.

```text
Sandbox
→ Observation
→ Dynamic Inspection
→ Evidence / Finding
```

Sandbox Session과 Observation의 정확한 구조는 Sandbox/Inspection Contract에서 후속 정의한다.

### 별도 최상위 Domain Concept로 두지 않는 항목

#### `SBOM`

SBOM은 Inspection Run 또는 Artifact/Verified Set에 연결되는 생성 결과물이며 현재 단계에서 독립적인 핵심 lifecycle Entity로 두지 않는다. SPDX/CycloneDX 등의 format은 후속 설계에서 결정한다.

#### Scanner / Verifier / Package Manager

Heliopause의 핵심 Domain Entity가 아니라 Port를 구현하거나 호출되는 외부 Provider/Adapter다.

#### Staging Area

Verified Artifact를 Promotion 전에 보관·전달하는 Architecture 영역이며 `Verified Set`과 동일한 Domain Entity로 취급하지 않는다. 실제 Staging record나 storage model이 필요한지는 후속 Staging/Promotion 설계에서 결정한다.

#### Raw tool output

외부 scanner/verifier/runtime의 원본 출력 자체를 Core Domain Model로 만들지 않는다. 필요한 자료는 Evidence Store에서 Raw Evidence로 보존하고 Core에는 정규화된 Result/Evidence를 전달한다.

## Domain Concept 관계 요약

```text
Operation Request
       ↓
Artifact Reference
       ↓
Resolved Artifact Identity
       ↓
Inspection Run 생성
       ↓
Acquisition / Acquired Artifact binding
   ├─ Verification Result
   ├─ Evidence
   ├─ Finding
   ├─ Capability
   ├─ Execution Status
   └─ Inspection Limitation
       ↓
Policy Decision
   ├─ ALLOW + inspect
   │       ↓
   │   Promotion 없음
   │       ↓
   │   Operation Result
   │
   ├─ ALLOW + install
   │       ↓
   │   ├─ Package ecosystem
   │   │      ↓
   │   │  Verified Set → Verified Manifest
   │   │      ↓
   │   │  trusted Promotion
   │   │
   │   └─ Standalone
   │          ↓
   │      exact identity / digest 재확인
   │          ↓
   │      trusted Promotion
   │
   │       ↓
   │   Operation Result
   │
   ├─ MANUAL_REVIEW
   │       ↓
   │   자동 Promotion 보류
   │       ↓
   │   Operation Result
   │
   └─ BLOCK
           ↓
       Promotion 없음
           ↓
       Operation Result
```

Dynamic Inspection이 필요한 경우:

```text
Inspection Run
      ↓
Dynamic Inspection
      ↓
Sandbox Session
      ↓
Observation
      ↓
Evidence / Finding
```

## Domain-001에서 확정하는 원칙

- 특정 ecosystem, package manager, scanner, sandbox 구현 또는 Host OS 전용 개념을 Core Domain에 넣지 않는다.
- 사용자 입력 reference, resolve된 logical identity, 실제 획득한 Artifact를 서로 구분한다.
- `Inspection Run`을 검사 lifecycle과 결과 추적의 중심 단위로 사용한다.
- Verification Result, Evidence, Finding을 서로 다른 의미의 개념으로 구분한다.
- Capability와 Execution Status를 보안 판정과 분리한다.
- 검사 한계를 Inspection Limitation으로 명시적으로 표현한다.
- 최종 Policy Decision은 `ALLOW / MANUAL_REVIEW / BLOCK`으로 유지한다.
- Package ecosystem의 Promotion 대상은 단순 dependency 목록이 아니라 Verified Set으로 표현한다.
- Verified Manifest는 Verified Set과 실제 Promotion 대상의 동일성을 추적하는 검증 기록으로 사용한다.
- Policy Decision과 실제 사용자 작업의 Operation Result를 분리한다.
- Sandbox Session과 Observation은 Dynamic Inspection을 지원하는 runtime-side Domain Concept로 구분한다.
- SBOM, 외부 scanner/verifier/package manager, Raw tool output 및 Staging storage 자체를 핵심 최상위 Domain Entity로 간주하지 않는다.
- 각 개념의 정확한 field, identifier, enum, schema, ownership 및 Go type은 후속 Domain 결정에서 정의한다.

## Domain-001 구조화된 결정과 구현 영향

- Domain Concept를 동일한 종류의 Entity로 취급하지 않고 후속 단계에서 Entity/Aggregate, Value Object, Result Record 등으로 분류한다.
- Operation Request는 원래 사용자 의도와 실행 context를 보존하고 Operation Result와 분리한다.
- Artifact Reference → Resolved Artifact Identity → Acquired Artifact를 mutable reference, logical identity, 실제 bytes/digest로 분리한다.
- Inspection Run을 Verification·Inspection·Evidence·Finding·Capability·Execution Status·Inspection Limitation·Policy Decision을 연결하는 lifecycle 단위로 둔다.
- Verification Result는 출처·identity·integrity 근거를, Evidence는 관찰·검증 사실을, Finding은 Evidence의 보안 해석을 표현한다.
- Capability와 Execution Status는 서로 분리하고 어느 것도 자체적으로 안전 판정이 되지 않게 한다.
- Inspection Limitation은 Policy가 참조할 수 있는 명시적 Domain Concept로 모델링한다.
- Policy Decision은 정규화된 검사 결과를 바탕으로 `ALLOW / MANUAL_REVIEW / BLOCK`만 생성한다.
- Verified Set은 Primary Artifact와 dependency의 정확한 검증 완료 집합이며 Verified Manifest는 이를 전달·확인·추적하는 기록 표현이다.
- Sandbox Session은 독립적인 격리 실행 lifecycle, Observation은 trusted observer가 수집한 raw 실행 관찰로 모델링하고 Sandbox가 Finding을 직접 판정하지 않게 한다.
- SBOM·외부 Provider/Adapter·Staging storage·Raw tool output은 핵심 최상위 Domain Entity가 아닌 결과·외부 경계·Architecture 영역으로 유지한다.
- 정확한 field·identifier·enum·schema·ownership·Go type은 Domain-002 이후 및 Contract 결정에서 확정한다.

## Domain-001 누락 점검

- [x] lifecycle 전체를 표현하는 ecosystem/구현 중립 Domain Concept 방향
- [x] Entity/Aggregate·Value Object·Result Record 분류 유보
- [x] Operation Request
- [x] Artifact Reference
- [x] Resolved Artifact Identity
- [x] Acquired Artifact
- [x] Inspection Run
- [x] Verification Result
- [x] Evidence
- [x] Finding
- [x] Capability
- [x] Execution Status
- [x] Inspection Limitation
- [x] Policy Decision
- [x] Verified Set
- [x] Verified Manifest
- [x] Operation Result
- [x] Sandbox Session
- [x] Observation
- [x] SBOM·Scanner·Verifier·Package Manager·Staging Area·Raw tool output의 최상위 Entity 제외
- [x] Domain Concept 관계와 ALLOW/MANUAL_REVIEW/BLOCK 흐름
- [x] 정확한 field·enum·schema·Go type 후속 결정 유보
