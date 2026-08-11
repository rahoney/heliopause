# Domain Model / Interface Contracts

이 문서는 Heliopause의 공통 Domain Concept와 Interface Contract 결정을 기록한다. 특정 package manager, registry, scanner, sandbox 구현 또는 Host OS에 종속된 개념은 Core Domain에 넣지 않는다.

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
Acquired Artifact
   ↓
Inspection Run
   ├─ Verification Result
   ├─ Finding
   ├─ Evidence
   ├─ Capability
   ├─ Execution Status
   └─ Inspection Limitation
   ↓
Policy Decision
   ↓
ALLOW인 경우
   ↓
Verified Set
   ↓
Verified Manifest
   ↓
Promotion / Operation
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

Operation Result는 install/import뿐 아니라 사용자가 요청한 operation 전체의 최종 실행 상태를 나타내며, Promotion이 수행되지 않은 inspect, MANUAL_REVIEW, BLOCK 상태에서도 생성될 수 있다.
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

하나의 Heliopause 검사 실행을 나타내는 핵심 lifecycle 단위다. Artifact resolution/acquisition 이후 Verification, Inspection, Evidence 수집, Finding 생성, Policy Decision 등 하나의 검사 과정에서 발생한 결과를 연결한다.

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

Artifact, Adapter, verifier, inspector 또는 runtime backend가 어떤 검사를 수행할 수 있는지를 나타낸다.

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
→ install/import 같은 실제 사용자 작업 결과
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
Acquired Artifact
       ↓
Inspection Run
   ├─ Verification Result
   ├─ Evidence
   ├─ Finding
   ├─ Capability
   ├─ Execution Status
   └─ Inspection Limitation
       ↓
Policy Decision
       ↓
       ├─ MANUAL_REVIEW → review / 후속 판단
       ├─ BLOCK         → 반입 금지
       │
       └─ ALLOW
            ↓
        Verified Set
            ↓
        Verified Manifest
            ↓
        trusted Promotion
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

## Domain-002: Artifact Reference / Resolved Artifact Identity / Acquired Artifact 모델

### 사용자 원문

Heliopause는 사용자가 입력한 Artifact 표현, source에서 resolve된 정확한 논리적 Artifact identity, 실제 Controlled Intake를 통해 획득한 Artifact를 서로 다른 Domain Concept로 구분한다.

```text
Artifact Reference
        ↓
Resolve
        ↓
Resolved Artifact Identity
        ↓
Acquire / Controlled Intake
        ↓
실제 content digest 계산
        ↓
Acquired Artifact
```

이 구분은 mutable reference, source resolution 결과와 실제 다운로드된 bytes를 혼동하지 않고, 이후 Verification·Inspection·Verified Set·Promotion이 실제 검사한 대상과 정확하게 연결되도록 하기 위함이다.

Domain-002에서는 정확한 Go struct, serialization schema, parser 또는 ecosystem별 CLI syntax를 확정하지 않고 각 Artifact 단계가 표현해야 하는 최소 의미와 identity invariant를 정의한다.

### 1. Artifact 모델의 기본 원칙

```text
사용자가 가리킨 것
≠
source가 resolve한 것
≠
실제로 획득한 것
```

예:

```text
kenv@latest
      ↓
npm registry resolve
      ↓
kenv@1.2.3
      ↓
acquisition
      ↓
kenv@1.2.3
sha256: AAA...
```

```text
Artifact Reference
→ 사용자가 무엇을 요청했는가

Resolved Artifact Identity
→ source 기준으로 정확히 무엇을 획득하기로 했는가

Acquired Artifact
→ 실제로 어떤 bytes를 획득하여 검사하고 있는가
```

## `Artifact Reference`

### 역할

`Artifact Reference`는 사용자가 입력하거나 외부 요청에서 전달한 Artifact 지정 정보다. exact version, release 또는 asset이 확정되지 않았을 수 있으며 mutable reference를 포함할 수 있다.

예:

```text
npm
kenv

npm
kenv@latest

pip
requests

github
owner/repository
release = latest
asset = <selector>
```

### Artifact Reference가 표현해야 하는 의미

최소한 다음 정보를 표현할 수 있어야 한다.

```text
Source / Ecosystem
→ 사용자 입력에서 명시되었거나 안전하게 판별된 경우 어떤 Artifact source/ecosystem을 대상으로 하는가

Requested Coordinate
→ 사용자가 어떤 Artifact를 요청했는가

Requested Reference
→ version / tag / release 등 사용자가 지정한 reference

Asset Selector
→ 필요한 경우 어떤 asset을 요청했는가
```

모든 항목이 항상 존재해야 하는 것은 아니다. `helox npm install kenv`에서는 version이 생략될 수 있고 `helox github ...`에서는 repository, release 및 asset 식별 정보가 필요할 수 있다. 구체적인 field와 ecosystem별 입력 구조는 후속 CLI/Adapter Contract에서 정의한다.

### Mutable Reference

다음과 같은 입력은 최종 Artifact identity가 아니다.

```text
latest
version 생략
latest release
mutable tag
기타 source가 다시 resolve할 수 있는 reference
```

사용자 입력 편의를 위해 허용할 수 있지만 실제 acquisition 전에 exact identity로 resolve한다.

```text
Artifact Reference
kenv@latest

        ↓ Resolve

Resolved Artifact Identity
kenv@1.2.3
```

Artifact Reference 자체에 과거 검사 결과나 `ALLOW`를 연결하여 현재 Artifact의 안전성을 판단하지 않는다.

## `Resolved Artifact Identity`

### 역할

`Resolved Artifact Identity`는 Artifact Reference를 supported source를 통해 resolve하여 획득 대상으로 확정한 **정확한 논리적 Artifact identity**다.

```text
Artifact Reference
        ↓
Artifact Adapter / source resolution
        ↓
Resolved Artifact Identity
```

이 단계에서는 더 이상 `latest`나 version 생략과 같은 mutable reference를 최종 identity로 사용하지 않는다.

### 공통적으로 표현해야 하는 의미

```text
Source Identity
→ 어느 registry / repository / release source의 대상인가

Ecosystem / Source Type
→ npm / PyPI / GitHub Releases 등

Exact Artifact Coordinate
→ 해당 source에서 Artifact를 유일하게 식별하는 정확한 좌표

Exact Version / Release
→ 적용 가능한 경우 확정된 version, tag 또는 release

Asset Identity
→ 여러 asset 중 하나를 선택해야 하는 경우 확정된 대상

Artifact Type
→ package / wheel / sdist / binary / archive 등
```

정확한 공통 schema와 ecosystem-specific coordinate 표현 방식은 후속 세부 Domain Model에서 결정한다.

### Source도 identity의 일부다

동일한 이름과 version 문자열이라고 해서 source가 다른 Artifact를 같은 대상으로 취급하지 않는다.

```text
source A / package X / version 1.0
≠
source B / package X / version 1.0
```

logical identity에는 Artifact 이름/version뿐 아니라 어떤 source에서 resolve된 대상인지가 포함되어야 한다.

### Exact version도 content identity는 아니다

`Resolved Artifact Identity`가 exact version 또는 release를 가지고 있다고 해서 실제 bytes까지 이미 확정되었다고 가정하지 않는다. source compromise, republish, asset replacement 또는 기타 비정상 상황으로 실제 bytes가 달라질 가능성을 Domain Model에서 배제하지 않는다.

```text
Resolved Artifact Identity
≠
최종 content identity
```

실제 content identity는 acquisition 이후 계산한 digest와 결합하여 확정한다.

## Digest와 Integrity 정보의 구분

Artifact source가 acquisition 전에 checksum, integrity hash 또는 기타 검증 정보를 제공할 수 있다.

```text
registry integrity metadata
release checksum
publisher-provided checksum
signature에 연결된 digest
```

이러한 값은 **expected / declared integrity information**이며 실제 획득한 bytes에서 Heliopause가 직접 계산한 digest와 동일한 의미로 취급하지 않는다.

```text
Source가 주장한 digest
        ↓
Expected / Declared Integrity

실제 획득 bytes에서 계산한 digest
        ↓
Observed Content Digest
```

Verification은 필요에 따라 두 값을 비교할 수 있지만 source가 제공한 digest가 존재한다는 사실만으로 실제 Artifact content identity가 확정된 것으로 간주하지 않는다.

## `Acquired Artifact`

### 역할

`Acquired Artifact`는 Heliopause의 Controlled Intake를 통해 실제로 획득하여 검사 대상으로 사용할 Software Artifact다.

```text
Resolved Artifact Identity
        +
Controlled Intake에서 획득한 content
        +
Observed Content Digest
        ↓
Acquired Artifact
```

Verification과 Inspection은 단순 Artifact Reference가 아니라 이 실제 획득 대상을 기준으로 수행한다.

### Acquired Artifact가 표현해야 하는 의미

최소한 다음 정보를 연결할 수 있어야 한다.

```text
Resolved Artifact Identity
→ 어떤 logical Artifact를 획득했는가

Observed Content Digest
→ 실제 bytes가 무엇인가

Controlled Content Reference
→ Heliopause가 실제 검사 content를 어디에서 참조하는가

Basic Content Metadata
→ 필요한 경우 size, content/type 정보 등
```

정확한 field 구성은 후속 Domain Model에서 결정한다.

### Controlled Content Reference

Acquired Artifact가 실제 content를 참조하더라도 Core Domain이 임의 Host filesystem path에 의존하도록 설계하지 않는다. 개념적으로 Heliopause가 통제하는 Intake/Quarantine content reference를 사용한다.

실제 filesystem path, object storage key, sandbox mount 방식 등의 storage 구현 세부사항은 Adapter/Infrastructure 영역에서 결정한다. 이 원칙은 Artifact Adapter가 arbitrary Host path에 Artifact를 직접 내려받아 Core에 전달하는 구조를 방지한다.

## Artifact Content Identity

실제 검사·Evidence·Verified Set·Promotion을 연결하는 기준은 단순 이름이나 version이 아니다.

```text
Resolved Artifact Identity
        +
Observed Content Digest
        ↓
검사 대상의 실제 content identity
```

다음은 동일한 logical identity를 가질 수 있지만 서로 다른 실제 검사 대상으로 취급한다.

```text
package X @ 1.2.3
digest AAA

package X @ 1.2.3
digest BBB
```

```text
같은 name
≠ 같은 Artifact content

같은 name + version
≠ 반드시 같은 Artifact content

같은 resolved identity + 같은 observed digest
→ 동일 content 여부를 판단하는 기본 기준
```

필요한 추가 identity 조건은 ecosystem별 Adapter Contract에서 정의할 수 있다.

## GitHub Releases의 Identity

GitHub Releases는 package ecosystem과 형태가 다르므로 package name/version 모델에 억지로 맞추지 않는다.

```text
Repository
      ↓
Exact Release / Tag
      ↓
Exact Asset
      ↓
Acquisition
      ↓
Observed Digest
```

복수 asset이 존재하면 임의로 하나를 선택하지 않는다.

```text
release
├─ linux-amd64.tar.gz
├─ linux-arm64.tar.gz
└─ darwin-arm64.tar.gz
```

여러 대상이 존재하면 실제 획득 전에 target asset을 하나로 resolve한다. exact asset name이 동일하더라도 실제 bytes 동일성을 보장한다고 가정하지 않으며 acquisition 이후 observed digest를 기준으로 실제 content를 식별한다.

## npm / PyPI의 Identity

Package ecosystem에서도 사용자 입력과 실제 검증 대상을 분리한다.

```text
npm

kenv
 ↓
kenv@latest
 ↓ resolve
kenv@1.2.3
 ↓ acquire
kenv@1.2.3 + digest AAA
```

PyPI도 같은 원칙을 사용한다. 하나의 Python release에 wheel과 sdist 또는 여러 platform-specific wheel이 존재할 수 있으므로 exact package version만으로 실제 획득 Artifact가 완전히 결정된다고 가정하지 않는다.

```text
package 1.2.3
├─ package-1.2.3.tar.gz
├─ package-1.2.3-py3-none-any.whl
└─ package-1.2.3-<platform>.whl
```

실제 검사 대상 distribution/file을 resolve해야 하는 경우 해당 정보도 `Resolved Artifact Identity`에 포함한다.

## Dependency에도 동일한 Identity 원칙 적용

Primary Artifact와 dependency에 서로 다른 identity 규칙을 사용하지 않는다.

```text
Artifact Reference 또는 dependency requirement
        ↓
Resolved Artifact Identity
        ↓
Acquired Artifact
        ↓
Observed Digest
```

Package manager가 dependency resolution 결과를 제공하더라도 실제 Promotion 대상 dependency는 exact identity와 digest에 추적 가능해야 한다.

```text
Primary Artifact
      +
Dependency A → exact identity + digest
Dependency B → exact identity + digest
Dependency C → exact identity + digest
      ↓
Verified Set
```

dependency requirement 표현과 dependency graph 자체의 Domain Model은 Domain-006에서 구체적으로 정의한다.

## Identity 변경 판단

다음 중 하나가 달라지면 기존 Artifact identity 또는 content 판정을 그대로 재사용하지 않는다.

```text
source 변경
exact version 변경
release/tag 변경
asset 변경
distribution/file 변경
observed digest 변경
```

특히 `latest`가 이전과 동일하더라도 resolve 결과가 달라지면 다른 검사 대상이다.

```text
어제
latest → 1.2.3 → digest AAA

오늘
latest → 1.2.4 → digest BBB

→ 서로 다른 검사 대상
```

exact version이 같지만 observed digest가 달라진 경우에도 기존 판정을 현재 Artifact에 적용하지 않는다.

```text
어제
1.2.3 → digest AAA

오늘
1.2.3 → digest BBB
```

## Artifact Identity와 Inspection Run

Inspection Run은 자신이 검사한 Artifact를 exact identity와 observed digest까지 추적할 수 있어야 한다.

```text
Inspection Run
      ↓
Acquired Artifact
      ↓
Resolved Artifact Identity
      +
Observed Digest
```

Verification Result, Finding, Evidence, Policy Decision, Verified Set, Verified Manifest, Promotion Record 등이 어떤 실제 Artifact content를 대상으로 생성되었는지 추적할 수 있어야 한다. Artifact name이나 mutable reference만을 기준으로 Inspection Run을 연결하지 않는다.

## Artifact Identity와 Promotion

Promotion 단계에서 사용자 입력 Reference를 다시 resolve하여 새로운 Artifact를 자유롭게 가져오지 않는다.

```text
Inspection에서 사용한 Acquired Artifact
        ↓
Verified Set / Manifest
        ↓
identity / digest 재확인
        ↓
Promotion
```

```text
사용자가 처음 입력한 Reference
→ 요청 추적용

Resolved Artifact Identity
→ logical target 추적용

Observed Digest
→ 실제 content 추적용

Verified Set / Manifest
→ 검증 완료 Promotion 대상 추적용
```

## Domain-002에서 확정하는 Invariant

### Invariant 1 — Mutable Reference는 최종 Identity가 아니다

`latest`, version 생략, mutable tag, latest release 등은 실제 acquisition 전에 exact identity로 resolve한다.

### Invariant 2 — Source는 Identity의 일부다

동일 name/version이라도 source가 다르면 자동으로 동일 Artifact라고 간주하지 않는다.

### Invariant 3 — Exact version만으로 bytes 동일성을 가정하지 않는다

```text
same source + same package + same version
```

이라도 실제 acquired bytes가 달라질 가능성을 배제하지 않는다.

### Invariant 4 — 실제 content는 observed digest로 식별한다

Acquired Artifact의 실제 bytes에서 Heliopause가 직접 계산한 digest를 content identity의 핵심 기준으로 사용한다.

### Invariant 5 — Declared Digest와 Observed Digest를 구분한다

source 또는 publisher가 제공한 integrity 정보는 Verification input이며 실제 bytes에서 계산한 observed digest와 동일한 것으로 가정하지 않는다.

### Invariant 6 — 검사 결과는 실제 Acquired Artifact에 연결한다

Verification Result, Finding, Evidence 및 Policy Decision은 단순 Artifact Reference가 아니라 실제 검사한 exact identity/digest에 추적 가능해야 한다.

### Invariant 7 — Primary Artifact와 Dependency에 같은 원칙을 적용한다

dependency도 최종 Promotion 대상이 된다면 exact identity와 observed digest에 추적 가능해야 한다.

### Invariant 8 — Promotion은 검사한 Artifact Identity를 유지한다

Promotion 과정에서 mutable reference를 다시 resolve하여 검사하지 않은 다른 Artifact를 반입하지 않는다.

### Invariant 9 — Acquired Content의 identity/digest binding을 유지한다

Observed Digest가 계산된 Acquired Artifact는 해당 digest가 가리키는 content와 연결된 상태로 Verification과 Inspection에 사용한다. 검사 전에 content 변경이 감지되거나 무결성을 보장할 수 없는 경우 기존 digest와 검사 결과를 재사용하지 않는다.

## 별도 Domain Concept로 아직 분리하지 않는 항목

### `Digest`

Digest는 중요한 Value Object 후보지만 exact algorithm 지원 범위, 복수 digest 표현 및 serialization 방식은 후속 세부 Domain 설계에서 확정한다.

```text
Digest
→ algorithm + value
```

SHA-256 등 구체적인 기본 algorithm 정책은 후속 Verification/Security 설계에서 결정한다.

### `Source`

Source 역시 Artifact Identity를 구성하는 핵심 Value Object 후보지만 registry URL, repository identity, trust metadata의 정확한 field는 Artifact Adapter Contract에서 구체화한다.

### `Content Reference`

Controlled Intake에서 획득한 실제 content를 가리키는 handle/reference가 필요하지만 filesystem path 등 infrastructure 구조를 Core Domain에 고정하지 않는다.

## 후속 결정으로 유보한 항목

- Artifact Reference의 정확한 field/schema
- Resolved Artifact Identity의 Go type
- npm/PyPI/GitHub source-specific coordinate schema
- canonical string representation
- Artifact ID 생성 방식
- Digest 지원 algorithm과 canonical format
- 복수 digest 저장 방식
- registry integrity metadata 구조
- GitHub checksum/signature metadata 구조
- Content Reference 구현 방식
- Artifact size/content-type metadata의 필수 여부
- platform/architecture selector의 정확한 모델
- package distribution/file selector 구조
- dependency requirement와 dependency graph
- cache key
- identity equality 비교 구현
- serialization format

위 항목은 후속 Domain Model, Artifact Adapter Contract, Verification 및 Storage 설계에서 구체화한다.

## 압축된 모델

```text
사용자 입력
kenv@latest
      ↓
Artifact Reference

resolve
      ↓
kenv@1.2.3
      ↓
Resolved Artifact Identity

Controlled Intake
      ↓
실제 bytes 획득
      ↓
observed digest = AAA
      ↓
Acquired Artifact

검사
      ↓
Inspection Run
      ↓
Evidence / Finding / Verification / Policy

ALLOW
      ↓
Verified Set / Manifest
      ↓
동일 identity + digest 재확인
      ↓
Promotion
```

핵심 원칙은 다음과 같다.

```text
사용자가 가리킨 것
≠
source가 resolve한 것
≠
실제로 획득한 것

그리고

실제로 검사한 것
=
Policy가 판정한 것
=
Verified Set에 들어가는 것
=
실제로 Promotion하는 것
```

Heliopause는 Artifact 이름이나 version 문자열만으로 동일성을 판단하지 않고, source에서 resolve된 exact logical identity와 실제 획득한 content digest를 연결하여 검사·Evidence·Policy·Promotion 전체에서 동일한 Artifact를 추적한다.

## Domain-002 구조화된 결정과 구현 영향

- Artifact Reference, Resolved Artifact Identity, Acquired Artifact를 별도 모델로 유지한다.
- mutable reference는 acquisition 전 exact logical identity로 resolve하고 과거 Policy/ALLOW를 Reference에 연결하지 않는다.
- source identity, ecosystem, exact coordinate, version/release, asset, Artifact type을 공통 의미로 표현한다.
- source declared integrity와 Heliopause observed content digest를 별도 값으로 보존하고 Verification에서 비교한다.
- Acquired Artifact는 통제된 Intake/Quarantine content reference와 identity/digest를 연결하며 임의 Host path를 Core에 노출하지 않는다.
- 동일 name/version이라도 source 또는 observed digest가 다르면 다른 Artifact content로 취급한다.
- GitHub는 Repository → exact Release/Tag → exact Asset → Acquisition → Observed Digest를 사용하고 복수 asset 임의 선택을 금지한다.
- npm/PyPI는 package version만으로 distribution/file bytes가 확정되었다고 가정하지 않는다.
- dependency에도 Primary Artifact와 동일한 Reference → Resolved Identity → Acquired Artifact → Observed Digest 원칙을 적용한다.
- identity·digest·dependency·distribution 변경 시 기존 판정을 재사용하지 않고 필요한 verification/inspection을 수행한다.
- Inspection Run과 모든 결과가 exact identity/observed digest를 추적하고 Promotion은 검사한 Acquired Artifact/Verified Set을 유지한다.
- Digest·Source·Content Reference는 Value Object/contract 후보로 두되 정확한 field·algorithm·storage는 후속 결정으로 유보한다.

## Domain-002 누락 점검

- [x] Reference → Resolve → Resolved Identity → Acquire/Controlled Intake → Observed Digest → Acquired Artifact 흐름
- [x] 사용자 입력·source resolution·실제 bytes의 분리
- [x] Artifact Reference의 source/coordinate/reference/asset 의미
- [x] mutable reference의 exact identity resolve
- [x] Resolved Identity의 source·ecosystem·coordinate·version/release·asset·type
- [x] Source를 identity 일부로 취급
- [x] Exact version과 content identity 분리
- [x] Declared/Expected Integrity와 Observed Content Digest 분리
- [x] Acquired Artifact의 controlled content reference
- [x] 임의 Host filesystem path를 Core에 고정하지 않음
- [x] name/version 동일해도 digest가 다르면 다른 content
- [x] GitHub repository/release/tag/asset identity
- [x] npm/PyPI distribution/file identity
- [x] dependency 동일 identity 원칙
- [x] identity 변경 시 기존 판정 재사용 금지
- [x] Inspection Run·Evidence·Policy·Verified Set·Promotion 추적
- [x] Promotion 중 mutable reference 재resolve 금지
- [x] Acquired Content identity/digest binding
- [x] Digest·Source·Content Reference 분리 후보와 후속 결정 유보
- [x] 정확한 Go type·schema·parser·serialization 후속 결정 유보

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
Verified Set / Manifest
      ↓
trusted Promotion
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

실제 install/import 결과가 실패하더라도 기존 Inspection Run의 `ALLOW`를 자동으로 `BLOCK`으로 변경하지 않는다.

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
- Promotion은 Run의 유효한 ALLOW·Verified Set/Manifest를 사용하며 Operation Result와 분리한다.

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
FAILED / INCOMPLETE / UNAVAILABLE / UNSUPPORTED
        ↓
안전한 것으로 간주하지 않음
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

## Domain-005: Verification Result / Evidence / Finding 모델

### 사용자 원문

Heliopause는 Artifact 검증·검사 과정에서 생성되는 `Verification Result`, `Evidence`, `Finding`을 서로 다른 의미의 Domain Concept로 구분한다.

```text
Verification → Verification Result → Evidence

Inspection / Observation → Evidence → Finding

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
Verification
      ↓
Verification Result
      ↓
Evidence
      │
      └──────────────┐
                     ↓
Inspection       Finding
    ↓                │
Observation          │
    ↓                 │
Evidence ────────────┘
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

## Domain-006: Dependency / Verified Set / Verified Manifest 모델

### 사용자 원문

Heliopause는 Package ecosystem에서 Primary Artifact와 dependency를 단순한 package 목록으로 관리하지 않고, 실제 acquisition·Verification·Inspection 결과와 연결된 정확한 Artifact 집합으로 추적한다.

특히 다음 개념을 구분한다.

```text
Dependency Requirement
→ 어떤 dependency가 필요한가

Resolved Dependency
→ 실제 어떤 Artifact가 dependency로 선택되었는가

Verified Set
→ 실제 Promotion에 사용할 수 있도록 검증이 완료된 Artifact 집합

Verified Manifest
→ 해당 Verified Set의 구성과 검증 근거를 고정하여 전달·확인하는 기록
```

기본 흐름은 다음과 같다.

```text
Primary Artifact
        ↓
Dependency Resolution
        ↓
Primary + Resolved Dependencies
        ↓
Acquisition / Verification / Inspection
        ↓
각 Artifact의 identity + digest + 검사 근거 확인
        ↓
Policy = ALLOW
        ↓
Verified Set 확정
        ↓
Verified Manifest 생성
        ↓
Staging / Promotion
```

Domain-006에서는 dependency graph의 정확한 schema, package manager별 locking 방식, Verified Manifest file format 및 Go type을 확정하지 않고 각 개념의 역할과 Promotion에 필요한 invariant를 정의한다.

## 1. Primary Artifact와 Dependency

사용자가 직접 지정한 검사 대상은 `Primary Artifact`다.

```text
helox npm install package-a

package-a → Primary Artifact
package-b, package-c, package-d → dependency
```

사용자는 일반적인 workflow에서 각 dependency를 직접 지정하지 않으며, dependency는 Artifact Adapter 및 관련 ecosystem resolution을 통해 식별한다.

## 2. Dependency도 Software Artifact다

Dependency는 Primary Artifact보다 낮은 신뢰 기준을 적용하는 별도 종류의 객체가 아니다. 각 dependency에도 Domain-002의 identity 원칙을 동일하게 적용한다.

```text
Dependency Requirement
        ↓
Resolve
        ↓
Resolved Artifact Identity
        ↓
Acquire
        ↓
Observed Digest
        ↓
Acquired Artifact
```

최종 Promotion 대상 dependency는 최소한 exact identity, source, observed digest, verification/inspection 근거 및 관련 Inspection Run에 추적 가능해야 한다. Primary Artifact가 검증되었다는 사실만으로 dependency를 자동 신뢰하지 않는다.

## 3. Dependency Requirement와 Resolved Dependency의 구분

Package metadata의 version range 또는 constraint는 최종 설치 Artifact identity가 아니다.

```text
Dependency Requirement
        ↓ ecosystem resolution
Resolved Dependency
        ↓
dependency-b@2.4.1
        ↓ acquire
digest AAA
```

따라서 `Dependency Requirement ≠ Resolved Artifact Identity ≠ Acquired Artifact`다. 정확한 requirement syntax와 version-range semantics는 각 ecosystem Adapter가 담당한다.

## 4. Dependency Resolution 결과

Heliopause는 실제 검사·설치에 사용될 direct 및 transitive dependency resolution 결과를 추적할 수 있어야 한다.

```text
Primary Artifact
      ↓
Dependency Resolution
      ↓
Dependency Graph
├─ Dependency A
├─ Dependency B
│    └─ Dependency C
└─ Dependency D
```

graph node/edge schema나 resolver algorithm은 Domain-006에서 확정하지 않지만, 최종 Promotion 대상이 되는 모든 dependency를 식별할 수 있어야 한다.

## 5. Dependency Resolution과 실제 설치의 일관성

검사 단계에서 resolve한 dependency 집합과 실제 사용자 환경 설치 단계의 dependency 집합이 달라져서는 안 된다.

```text
검사 시 Dependency Set
        =
실제 Promotion / Install Dependency Set
```

package manager가 실제 install 시 dependency를 자유롭게 다시 resolve하여 새 Artifact를 추가하도록 허용하지 않는다. 검사 당시 Primary+A+B였는데 실제 install에서 C가 추가되면 C를 자동 신뢰하거나 반입하지 않고 Promotion을 중단하여 acquisition/verification 및 필요한 inspection으로 되돌린다. 정확한 locking/offline install 방식은 후속 ecosystem Tooling에서 정의한다.

## 6. Verified Set

`Verified Set`은 실제 Promotion 대상으로 사용할 수 있도록 검증된 Primary Artifact와 dependency의 정확한 집합이다.

```text
Verified Set
├─ Primary Artifact
├─ Dependency A
├─ Dependency B
└─ Dependency C
```

`Dependency Resolution Result`가 설치에 필요하다고 계산한 집합이라면, `Verified Set`은 각 구성요소의 실제 identity/digest와 필요한 검증 근거가 확인되어 Promotion 대상으로 확정된 집합이다. 따라서 `Resolved Dependency Set ≠ Verified Set`이다.

## 7. Verified Set 구성요소

각 entry는 최소한 다음 사실에 추적 가능해야 한다.

```text
Artifact Role             → Primary / Dependency
Resolved Artifact Identity → 정확히 어떤 Artifact인가
Observed Digest            → 실제 어떤 content인가
Source                     → 어디에서 획득했는가
Inspection Run             → 어떤 검사에서 검증되었는가
Verification / Inspection Basis → 어떤 근거로 검증 완료인가
```

정확한 field와 schema는 후속 Domain/Manifest 설계에서 확정한다.

## 8. 모든 구성요소가 하나의 Inspection Run에 속할 필요는 없다

Verified Set의 모든 구성요소가 동일한 Run ID를 가져야 한다고 제한하지 않는다.

```text
Primary Artifact → Run A
Dependency B     → Run A
Dependency C     → Run B (추가 검증)
```

중요한 것은 각 Artifact가 자신을 검증한 유효한 Run과 근거에 추적 가능하고, 최종 Set 전체가 일관된 Promotion 대상이라는 점이다. `모든 Artifact가 동일 Run`을 강제하지 않고 `모든 Artifact가 검증 근거에 추적 가능`을 강제한다.

## 9. Verified Set과 Policy Decision

Verified Set은 임의의 검사 결과로 생성하지 않는다. 최종 Promotion에 사용하는 Set은 해당 구성과 연결된 유효한 `ALLOW` Policy 근거를 가져야 한다.

```text
Artifact / Dependency 검사
        ↓
Verification / Inspection
        ↓
유효한 Policy 근거
        ↓
Verified Set
```

Primary Artifact의 `ALLOW`만으로 검증되지 않은 dependency를 Set에 포함하지 않는다. 구성요소별 Run은 다를 수 있지만, 최종 Set 전체는 하나의 현재 유효한 기준 Run/Policy와 연결되고 각 entry는 자신을 검증한 Run까지 추적 가능해야 한다.

```text
기준 Inspection Run
        ↓
현재 Verified Set 구성
        ↓
Policy = ALLOW
        ↓
Verified Manifest

각 Set Entry → 자신을 실제로 검증한 Inspection Run
```

`Unverified Artifact → Verified Set 포함 금지` 원칙은 고정한다. bundle-level Policy aggregation rule은 후속 Policy 설계에서 정의한다.

## 10. Verified Set의 확정 이후 변경

Verified Set이 Promotion 대상으로 확정된 이후 구성요소를 조용히 추가·삭제·교체하지 않는다.

```text
Verified Set V1
├─ A digest AAA
├─ B digest BBB
└─ C digest CCC
        ↓ 새 Artifact D 필요
검증
        ↓
Verified Set V2
```

필요한 검증을 완료한 뒤 새로운 검증 완료 집합을 확정한다. Verified Set ID/versioning 방식은 후속 설계에서 결정한다.

## 11. Dependency 변경은 Verified Set 변경이다

다음 변경은 Verified Set의 실질적인 변경이다.

```text
dependency 추가 / 삭제
dependency version 변경
dependency source 변경
dependency digest 변경
Primary Artifact digest 변경
```

동일 package 이름과 version이라도 digest가 다르면 다른 Set으로 취급한다.

## 12. Verified Manifest

`Verified Manifest`는 확정된 Verified Set의 구성과 검증 연결 정보를 명시적으로 기록하여 Staging·Promotion·감사 과정에서 재확인할 수 있도록 하는 record다.

```text
Verified Set
        ↓
Verified Manifest
        ↓
Staging
        ↓
Promotion
```

Verified Set이 “어떤 Artifact 집합이 검증 완료되었는가”라는 Domain 의미라면, Manifest는 “그 집합을 어떤 identity/digest와 근거로 고정하여 전달하는가”를 표현한다.

## 13. Verified Set과 Verified Manifest의 구분

두 개념을 동일시하지 않는다.

```text
Verified Set
→ 검증 완료 Artifact 집합이라는 Domain 개념

Verified Manifest
→ 그 집합을 명시적으로 기록·전달·재검증하기 위한 record
```

Manifest의 JSON/YAML 또는 기타 file format은 후속 serialization/storage 설계에서 결정한다.

## 14. Verified Manifest가 추적해야 하는 정보

Manifest는 최소한 다음 관계를 추적할 수 있어야 한다.

```text
Verified Set identity
Primary Artifact
각 Dependency
각 Artifact의 exact identity
각 Artifact의 observed digest
각 Artifact의 source
관련 Inspection Run
Policy Decision 근거
Manifest 생성 시점 또는 version 정보
```

Manifest가 Artifact bytes 자체를 포함해야 하는 것은 아니며, 실제 content와 검증 근거를 식별·연결하는 것이 목적이다.

## 15. Manifest와 실제 Artifact의 일치

Manifest에 기록된 identity/digest와 실제 Staging/Promotion 대상 Artifact가 일치해야 한다.

```text
Verified Manifest digest AAA
        ↓ 실제 Artifact digest 계산
AAA → 일치 → 다음 신뢰 경계 진행
```

불일치하면 기존 Manifest와 `ALLOW`를 근거로 Promotion하지 않고 필요한 Verification/Inspection으로 회귀한다.

## 16. Quarantine → Staging 재검증

Quarantine에서 Staging으로 들어가는 Artifact가 검사한 Artifact와 정확히 동일한지 확인한다.

```text
Quarantine / Inspection
        ↓
Policy = ALLOW
        ↓
Verified Set / Manifest
        ↓
Staging 반입 직전 identity / digest 재검증
        ↓
Staging
```

## 17. Staging → Host 재검증

실제 사용자 환경으로 Promotion하기 전 다시 Manifest와 Artifact identity/digest의 일관성을 확인한다.

```text
Staging
        ↓
identity / digest 재검증
        ↓
trusted Promotion
        ↓
User Environment
```

Package ecosystem의 기본 경계는 `Inspection → ALLOW → Verified Set/Manifest → digest 확인 → Staging → digest 다시 확인 → trusted Promotion`이다.

## 18. Staging은 Verified Set이 아니다

```text
Verified Set → 무엇이 검증 완료되었는가
Staging      → 검증 완료 Artifact를 Promotion 전 어디에서 보관·전달하는가
```

Staging은 storage/runtime Architecture 책임이며 Verified Set은 Domain 개념이다. Staging에 파일이 존재한다는 사실만으로 Verified Set entry라는 뜻은 아니며, Staging은 Manifest를 기준으로 Artifact 일관성을 확인해야 한다.

## 19. Verified Set에서 새로운 Artifact 자동 추가 금지

Staging 또는 Promotion 과정에서 새 Artifact가 요구되면 기존 Verified Set에 자동 편입하지 않는다.

```text
새 Dependency D 필요
        ↓
Verified Set에 없음
        ↓
자동 download/install 금지
        ↓
D acquisition → Verification → 필요한 Inspection → Policy
        ↓
새로운 유효한 Verified Set / Manifest
```

Package manager의 최종 dependency resolution이 Heliopause 보안 경계를 우회하지 못하도록 한다.

## 20. Verified Set의 혼합 금지

서로 다른 Run이나 서로 다른 digest에서 가져온 결과를 검증 없이 임의 조합하지 않는다.

```text
Run A: Artifact X digest AAA, ALLOW
Run B: Artifact X digest BBB, ALLOW
```

Run A의 Manifest와 Run B의 Artifact를 섞어 Promotion하지 않는다. 다만 서로 다른 dependency가 각각 자신의 정확한 identity/digest와 유효한 검증 근거에 추적 가능하고 최종 Set 구성이 현재 Promotion에 유효하게 확정된 경우 여러 Run의 결과를 하나의 Set에 포함할 수 있다.

## 21. Verified Set과 재검사

새 검사가 수행되어도 기존 Verified Set 또는 Manifest의 historical record를 덮어쓰지 않는다.

```text
Verified Set V1
        ↓ Dependency B 재검사
새 결과
        ↓
필요한 경우 Verified Set V2
```

기존 V1은 당시 구성과 검증 근거를 추적 가능하게 유지한다. 정확한 lifecycle/version 모델은 후속 설계에서 정의한다.

## 22. Standalone Artifact와 Verified Set

`Verified Set`은 dependency가 존재하는 Package ecosystem의 핵심 모델이다. Package manager가 필요 없는 standalone binary/archive에 multi-artifact Set semantics를 강제하지 않는다.

```text
Standalone Artifact
        ↓
Inspection
        ↓
Policy = ALLOW
        ↓
identity / digest 재검증
        ↓
trusted Promotion
```

필요하면 단일 Artifact manifest/record를 사용할 수 있지만 package dependency set semantics를 반드시 적용하는 것으로 고정하지 않는다. standalone에도 `검사한 Artifact = ALLOW 대상 = 실제 Promotion Artifact` invariant는 동일하게 적용한다.

## 23. Dependency / Verified Set과 Evidence 추적

Verified Set entry는 단순히 `verified = true`만 가지는 값이 아니다.

```text
Verified Set Entry
        ↓
Artifact identity / digest
        ↓
Inspection Run
        ↓
Verification Result / Finding / Evidence
        ↓
Policy 근거
```

Verified Set은 검사 근거와 단절된 단순 package lockfile이 아니다.

## 24. Verified Manifest와 SBOM의 구분

```text
SBOM
→ 어떤 software component가 포함되어 있는가를 기술하는 구성 명세

Verified Manifest
→ 어떤 exact Artifact가 Heliopause 검사와 Policy를 통과하여 Promotion 대상으로 확정되었는가를 기록
```

SBOM은 Verified Set 또는 Inspection Run의 결과물로 연결될 수 있지만 Verified Manifest를 대체하지 않는다.

## Domain-006에서 확정하는 Invariant

### Invariant 1 — Dependency도 동일한 Artifact identity 원칙을 사용한다

Primary Artifact와 dependency에 서로 다른 신뢰·identity 기준을 사용하지 않는다.

### Invariant 2 — Requirement와 Resolved Dependency를 구분한다

version range/constraint를 최종 Artifact identity로 사용하지 않는다.

### Invariant 3 — 실제 Promotion dependency를 모두 식별한다

direct/transitive 여부와 관계없이 실제 사용자 환경으로 반입되는 dependency는 추적 가능해야 한다.

### Invariant 4 — Resolution Result와 Verified Set을 구분한다

설치에 필요하다고 resolve된 집합과 실제 검증 완료되어 Promotion 가능한 집합을 동일시하지 않는다.

### Invariant 5 — Verified Set 구성요소는 exact identity/digest에 고정한다

이름/version 문자열만으로 Verified Set 동일성을 판단하지 않는다.

### Invariant 6 — 모든 구성요소가 동일 Run에 속할 필요는 없다

각 Artifact가 자신을 검증한 유효한 Run과 근거에 추적 가능하면 된다.

### Invariant 7 — 검증되지 않은 Artifact는 Verified Set에 포함하지 않는다

Primary의 `ALLOW`를 dependency 전체의 자동 신뢰로 확장하지 않는다.

### Invariant 8 — 확정된 Verified Set을 조용히 변경하지 않는다

구성요소의 추가·삭제·교체가 필요하면 필요한 검증 후 새로운 유효한 Set을 확정한다.

### Invariant 9 — Verified Manifest와 Verified Set을 구분한다

Set은 검증 완료 집합이라는 Domain 개념이고 Manifest는 이를 기록·전달·확인하는 record다.

### Invariant 10 — Manifest와 실제 Artifact가 일치해야 한다

Staging 및 Promotion에서 Manifest의 identity/digest와 실제 content의 일관성을 확인한다.

### Invariant 11 — Staging 전과 Host 반입 전 각각 재검증한다

Quarantine → Staging 및 Staging → Host 두 신뢰 경계에서 identity/digest를 재확인한다.

### Invariant 12 — 새로운 dependency가 Promotion 경계를 우회하지 못한다

Verified Set 밖의 Artifact가 요구되면 자동 설치하지 않고 검증 흐름으로 되돌린다.

### Invariant 13 — 서로 다른 검증 결과를 임의로 혼합하지 않는다

모든 구성요소와 검증 근거는 최종 Promotion 대상과 일관되게 연결되어야 한다.

### Invariant 14 — Standalone에 package-only Set semantics를 강제하지 않는다

Standalone은 Staging/Verified Set 예외를 사용할 수 있지만 exact identity/digest와 trusted Promotion invariant는 유지한다.

### Invariant 15 — Verified Manifest는 SBOM이 아니다

Software component inventory와 Heliopause의 검증 완료 Promotion record를 구분한다.

## 후속 결정으로 유보한 항목

- Dependency Requirement의 Go type/schema
- dependency graph node/edge schema
- ecosystem별 version range semantics
- dependency resolver 구현
- direct/transitive dependency 표시 방식
- optional/peer/dev dependency 처리
- platform-specific dependency resolution
- lockfile 연동 방식
- npm/pip offline install 방식
- resolver와 실제 package manager의 일관성 enforcement
- Verified Set ID/version
- Verified Set lifecycle
- Verified Set entry schema
- Verified Manifest schema
- Verified Manifest serialization format
- Manifest integrity/signature 방식
- Manifest storage 위치
- Manifest retention
- bundle-level Policy aggregation
- dependency별 Inspection Run 생성 기준
- dependency 결과 reuse/cache 조건
- Staging filesystem/layout
- Staging retention/cleanup
- standalone용 manifest 필요 여부
- SBOM과 Manifest 간 reference 방식

위 항목은 후속 Policy, Operation/Promotion, Evidence/Result, Artifact Adapter 및 Interface Contract 설계에서 구체화한다.

## 압축된 모델

```text
Primary Artifact
        ↓
Dependency Resolution
        ↓
Resolved Dependency Graph
        ↓
각 Artifact Acquisition
        ↓
exact identity + observed digest
        ↓
Verification / Inspection
        ↓
유효한 검증 근거
        ↓
Verified Set
        ↓
Verified Manifest
        ↓
identity / digest 재확인
        ↓
Staging
        ↓
identity / digest 재확인
        ↓
trusted Promotion
```

새 dependency가 발견되면:

```text
Promotion
    ↓
Verified Set 밖 Artifact 필요
    ↓
STOP
    ↓
Acquisition / Verification / Inspection
    ↓
새로운 유효한 Verified Set / Manifest
```

핵심 원칙은 다음과 같다.

```text
Dependency를 찾았다
≠
Dependency를 검증했다
≠
Promotion이 허용된 Verified Set에 포함되었다
```

```text
검사한 정확한 Artifact 집합
=
Policy가 허용한 Artifact 집합
=
Verified Manifest가 기록한 Artifact 집합
=
실제로 Promotion하는 Artifact 집합
```

Heliopause는 package manager가 resolve한 dependency 목록 자체를 신뢰하지 않고, 실제 사용자 환경으로 반입되는 Primary Artifact와 dependency 각각을 exact identity/digest 및 검증 근거에 연결하여 하나의 일관된 Verified Set으로 확정한 뒤에만 Promotion에 사용하는 것을 기본 원칙으로 한다.

## Domain-006 구조화된 결정과 구현 영향

- Dependency Requirement, Resolved Dependency, Acquired Artifact, Verified Set, Verified Manifest를 서로 다른 개념으로 모델링한다.
- Primary와 direct/transitive dependency 모두 source·exact identity·observed digest·검증 Run 추적을 요구한다.
- Resolution 결과와 Verified Set을 구분하고, 검사 Set과 실제 설치 Set의 일관성을 강제한다.
- Verified Set은 유효한 기준 Inspection Run/Policy `ALLOW`와 연결되며, 각 entry는 자신을 검증한 Run까지 추적한다.
- Verified Set 확정 후 구성요소를 조용히 변경하지 않고 변경 시 새 Set/Manifest를 생성한다.
- Manifest는 Set의 identity/digest·source·Run·Policy 근거를 고정하는 record이며 SBOM과 구분한다.
- Quarantine→Staging 전과 Staging→Host 전 identity/digest를 각각 재검증한다.
- Verified Set 밖의 새로운 dependency는 자동 반입하지 않고 acquisition/verification/inspection으로 회귀한다.
- 서로 다른 Run·digest 결과를 검증 없이 혼합하지 않으며, 여러 Run의 구성요소를 쓸 경우 최종 Set/Policy 일관성을 확인한다.
- Standalone binary/archive에는 package-only Set semantics를 강제하지 않되 identity/digest·trusted Promotion invariant는 유지한다.
- 정확한 resolver·lockfile·manifest schema·storage·aggregation·offline install은 후속 설계에서 확정한다.

## Domain-006 누락 점검

- [x] Primary Artifact와 dependency 구분
- [x] Dependency identity 원칙
- [x] Requirement와 Resolved Dependency 구분
- [x] direct/transitive dependency graph 추적
- [x] 검사 Set과 실제 설치 Set 일관성
- [x] Verified Set 역할·구성요소·Run 추적
- [x] 구성요소별 서로 다른 Run 허용 및 기준 Run/Policy 연결
- [x] 확정 Set 변경 시 새 Set 생성
- [x] Manifest 역할·Set과의 구분·추적 정보
- [x] Manifest와 실제 Artifact 일치 검증
- [x] Quarantine→Staging 및 Staging→Host 재검증
- [x] Staging과 Verified Set 구분
- [x] 새 Artifact 자동 추가 금지
- [x] Verified Set 혼합 금지와 여러 Run 구성 허용 조건
- [x] 재검사 시 historical record 보존
- [x] Standalone 예외
- [x] Evidence·Policy 근거 추적
- [x] SBOM과 Manifest 구분
- [x] 15개 invariant
- [x] 후속 결정 항목 유보

## Domain-007: Operation Request / Install Context 모델

### 목적

Heliopause는 **사용자가 무엇을 하려고 했는지**와, `ALLOW` 이후 그 작업을 정확히 이어서 수행하기 위해 필요한 **실행 context**를 검사 과정 전체에서 보존한다.

핵심 개념은 다음과 같다.

```text
Operation Request
→ 사용자가 무엇을 요청했는가

Install Context
→ ALLOW 이후 정확히 어디에 어떤 조건으로 설치를 이어가야 하는가

Operation Result
→ 최종적으로 사용자 요청이 어떻게 끝났는가
```

기본 흐름은 다음과 같다.

```text
Operation Request
        ↓
Inspection Run
        ↓
Policy Decision
        ├─ ALLOW
        │    ↓
        │  원래 Operation 확인
        │    ↓
        │  install/import이면
        │  trusted Promotion
        │    ↓
        │  Operation Result
        │
        ├─ MANUAL_REVIEW
        │    ↓
        │  자동 실행 보류
        │
        └─ BLOCK
             ↓
           반입하지 않음
```

Domain-007에서는 CLI argument의 정확한 schema, package-manager별 install option 및 Promotion API를 확정하지 않고 사용자 의도와 trusted Promotion 사이의 연결 원칙을 정의한다.

### Operation Request

`Operation Request`는 Heliopause에 들어온 하나의 사용자 작업 요청을 나타낸다.

최소한 다음 의미에 추적 가능해야 한다.

```text
Operation Type
→ install / inspect 등

Artifact Reference
→ 사용자가 지정한 Artifact

Ecosystem / Source
→ npm / pip / GitHub Releases 등

Operation Context
→ 해당 작업을 정확하게 수행하기 위해 필요한 context
```

예:

```text
helox npm install kenv
```

는 개념적으로:

```text
Operation Type = install
Ecosystem = npm
Artifact Reference = kenv
```

라는 요청이다.

Operation Request는 `Inspection Run`이나 `Policy Decision`과 동일한 개념이 아니다.

### Operation Type

MVP의 핵심 Operation Type은:

```text
install
inspect
```

이다.

두 operation 모두 Artifact 검사 workflow를 사용할 수 있지만 `ALLOW` 이후 행동이 다르다.

```text
inspect
→ 검사 결과 제공
→ Host 반입 없음

install
→ 검사
→ ALLOW
→ 검증된 정확한 Artifact/Set을 원래 대상에 설치
```

따라서 Policy가 `ALLOW`라고 해서 항상 Promotion이 발생하는 것은 아니다.

### Install Context

`Install Context`는 `install` 요청이 `ALLOW`된 뒤 원래 사용자의 설치 의도를 안전하게 이어가기 위해 필요한 정보를 나타낸다.

예를 들어 개념적으로 다음 정보가 필요할 수 있다.

```text
target environment
working/project context
package manager
installation mode
사용자가 명시한 install option
platform / architecture 정보
```

정확한 field와 ecosystem별 option은 후속 Artifact/Promotion Contract에서 결정한다.

핵심은:

```text
검사 전의 사용자 설치 의도
        =
ALLOW 이후 실제 수행하는 설치 의도
```

를 유지하는 것이다.

### Install Context는 Sandbox 입력이 아니다

Install Context에 실제 사용자 환경을 설명하는 정보가 존재한다고 해서 이를 Sandbox에 그대로 제공하지 않는다.

특히 다음과 같은 실제 Host 자산은 별도 보안 경계를 유지한다.

```text
실제 credential
token
.env secret
SSH key
browser/cloud credential
실제 Host filesystem
internal network access
```

즉:

```text
Install Context
→ trusted workflow / Promotion이 사용하는 실행 context

Sandbox Context
→ 격리·합성된 검사 환경
```

으로 구분한다.

Sandbox에는 실제 Host context 대신 필요한 경우 synthetic filesystem, dummy credential, honeytoken 및 sanitized data를 사용한다.

### ALLOW 이후 원래 Operation을 이어간다

사용자가:

```text
helox npm install kenv
```

를 요청했다면 Heliopause의 정상 UX는:

```text
install 요청
    ↓
검사
    ↓
ALLOW
    ↓
Verified Set / Manifest
    ↓
trusted Promotion
    ↓
원래 install 자동 수행
```

이다.

`ALLOW` 이후 사용자가 별도의 `promote` 명령을 다시 실행해야 하는 구조를 기본 UX로 사용하지 않는다.

반면:

```text
helox npm inspect kenv
```

라면:

```text
검사
→ ALLOW / MANUAL_REVIEW / BLOCK 결과 제공
→ Host 설치 없음
```

으로 끝난다.

### Promotion은 Operation Request를 임의 변경하지 않는다

Promotion 과정에서 사용자가 요청하지 않은 다른 작업으로 변경하지 않는다.

예:

```text
원래 요청:
install package X into target A
```

였다면 `ALLOW` 후:

```text
target B에 설치
다른 version 설치
새 dependency 자유롭게 resolve
다른 installation mode 사용
```

등으로 조용히 변경하지 않는다.

Artifact identity와 dependency는 Verified Manifest를 따르고, 실행 대상은 원래 trusted Operation/Install Context와 일치해야 한다.

### MANUAL_REVIEW / BLOCK과 Operation

`MANUAL_REVIEW`에서는 원래 install operation의 자동 진행을 보류한다. Standalone Artifact의 Host 반입 역시 동일하게 자동 진행하지 않는다.

```text
Policy = MANUAL_REVIEW
→ 자동 Promotion 없음
→ Operation Status = PAUSED 가능
```

`BLOCK`에서는 Artifact를 사용자 환경에 반입하지 않는다.

```text
Policy = BLOCK
→ Promotion 없음
→ install Operation Status = NOT_PERFORMED
```

그러나 `inspect` 요청에서는 `BLOCK`이 발견되어도 검사 요청 자체는 정상적으로 완료될 수 있다.

```text
Operation = inspect
Policy = BLOCK
Operation Status = COMPLETED
```

Policy Decision과 Operation Status는 Domain-004에서 정의한 대로 분리한다.

### Promotion 대상의 동일성

Install Context는 **어디에 어떻게 설치할 것인가**를 보존하고, Verified Manifest는 **무엇을 설치할 것인가**를 고정한다.

```text
Install Context
→ 어디에 / 어떤 방식으로

Verified Manifest 또는 exact Artifact identity/digest
→ 정확히 무엇을

Policy Decision
→ 반입이 허용되는가
```

따라서 trusted Promotion은 최소한 다음 세 요소가 일관된 경우에만 진행한다.

```text
유효한 ALLOW
+
Verified Manifest
또는 Standalone의 exact Artifact identity/digest
+
원래 Install Context
        ↓
trusted Promotion
```

검증되지 않은 Artifact를 추가하거나 mutable reference를 다시 resolve하지 않는다.

### Standalone Artifact

Package manager가 없는 standalone binary/archive도 동일한 Operation Request 원칙을 사용한다.

다만 기존 Architecture의 예외에 따라 Verified Staging Area를 거치지 않고 직접 trusted Promotion할 수 있다.

```text
Operation Request
        ↓
Standalone Artifact 검사
        ↓
ALLOW
        ↓
identity / digest 재확인
        ↓
원래 target context 확인
        ↓
trusted Promotion
```

이 경우에도 Sandbox가 Host에 직접 파일을 쓰지는 않는다.

### Operation Result

`Operation Result`는 사용자가 요청한 operation 전체의 최종 실행 결과를 나타낸다.

```text
Operation Request
        ↓
Inspection / Policy / 필요 시 Promotion
        ↓
Operation Result
```

따라서 `Operation Result`는 Promotion이 발생한 install에만 존재하는 개념이 아니다.

예:

```text
inspect + BLOCK
→ Operation Result 존재

install + MANUAL_REVIEW
→ Operation Result 존재

install + ALLOW + 설치 실패
→ Operation Result 존재
```

Operation Status는 Domain-004에서 정의한:

```text
COMPLETED
FAILED
PAUSED
NOT_PERFORMED
```

의 의미를 따른다.

정확한 Operation Result schema는 후속 Result/Interface Contract에서 정의한다.

### Domain-007에서 확정하는 Invariant

#### Invariant 1 — 사용자 의도를 보존한다

Inspection 과정이 원래 Operation Request의 의미를 임의로 변경하지 않는다.

#### Invariant 2 — `inspect`와 `install`을 구분한다

`ALLOW`라도 inspect 요청에는 자동 Promotion을 수행하지 않는다.

#### Invariant 3 — `install + ALLOW`는 원래 작업을 자동으로 이어간다

별도의 일반 사용자용 `promote` 명령을 기본 workflow로 요구하지 않는다.

#### Invariant 4 — Install Context와 Sandbox Context를 구분한다

실제 Host의 credential·filesystem·network context를 Dynamic Inspection 환경에 그대로 전달하지 않는다.

#### Invariant 5 — Promotion은 세 요소를 일치시킨다

```text
ALLOW
+
Verified Manifest / exact Artifact
+
Original Install Context
```

가 일관된 경우에만 trusted Promotion한다.

#### Invariant 6 — Promotion에서 Artifact를 다시 자유롭게 resolve하지 않는다

새 Artifact 또는 dependency가 필요하면 검증 workflow로 되돌린다.

#### Invariant 7 — MANUAL_REVIEW / BLOCK은 자동 Promotion하지 않는다

사람의 판단이 필요하거나 반입이 거부된 Artifact를 자동으로 Host에 반입하지 않는다.

#### Invariant 8 — Policy와 Operation Result를 구분한다

보안 판정과 실제 사용자 작업의 성공·실패·보류 상태를 동일시하지 않는다.

### 후속 결정으로 유보한 항목

Domain-007에서는 다음을 구체적으로 확정하지 않는다.

- Operation Request의 Go type/schema
- Operation ID 필요 여부
- Install Context의 정확한 field
- ecosystem별 install option
- working directory / project context 표현
- target environment 식별 방식
- 환경변수 전달 정책
- package manager option allowlist
- Install Context snapshot 방식
- MANUAL_REVIEW 이후 continuation 방식
- Operation Result schema
- Promotion Record와 Operation Result 관계
- CLI exit code
- CI 결과 mapping

위 항목은 Artifact/Promotion/Evidence Interface Contract와 후속 구현 설계에서 구체화한다.

### 압축된 모델

```text
Operation Request
        ↓
Artifact 검사
        ↓
Policy
   ┌────┼──────────────┐
   ↓    ↓              ↓
ALLOW  MANUAL_REVIEW  BLOCK
   ↓    ↓              ↓
operation 확인       Promotion 없음
   ↓
inspect → 결과만 반환

install
   ↓
Verified Manifest
+
Original Install Context
   ↓
trusted Promotion
   ↓
Operation Result
```

핵심 원칙은 다음과 같다.

```text
무엇을 검사할 것인가
→ Artifact Identity / Verified Manifest

반입해도 되는가
→ Policy Decision

어디에 어떻게 설치할 것인가
→ Install Context

실제로 어떻게 끝났는가
→ Operation Result
```

Heliopause는 검사를 별도의 사용자 작업으로 끊어버리지 않고, 원래 Operation Request를 검사 전체에서 보존한 뒤 `ALLOW`된 정확한 Artifact만 trusted Promotion을 통해 원래 사용자의 작업으로 안전하게 이어가는 것을 기본 원칙으로 한다.

## Domain-007 구조화된 결정과 구현 영향

- Operation Request, Inspection Run, Policy Decision, Install Context, Operation Result를 서로 다른 개념으로 모델링한다.
- MVP Operation Type은 `install`과 `inspect`이며, `ALLOW` 후 Promotion 여부를 원래 Operation Type에 따라 결정한다.
- `install + ALLOW`는 Verified Manifest 또는 standalone exact identity/digest와 원래 Install Context를 사용해 원래 설치 작업을 자동으로 이어간다.
- `inspect`는 Policy 결과와 무관하게 Host 반입 없이 검사 결과만 반환한다.
- Install Context는 target·project·package manager·mode·option·platform 등 설치 의도를 보존하지만 Sandbox Context와 분리한다.
- 실제 credential·secret·Host filesystem·internal network access를 Sandbox에 전달하지 않고 합성·정제된 검사 입력을 사용한다.
- Promotion은 사용자가 요청한 target·version·dependency·installation mode를 임의 변경하거나 mutable reference를 다시 resolve하지 않는다.
- `MANUAL_REVIEW`와 `BLOCK`에서는 자동 Promotion하지 않으며 Policy Decision과 Operation Status를 독립적으로 기록한다.
- Promotion은 유효한 `ALLOW`, Verified Manifest 또는 standalone exact identity/digest, 원래 Install Context의 일관성을 확인한 뒤 수행한다.
- standalone Artifact는 Staging 생략 예외를 허용하되 identity/digest 재확인과 trusted Promotion 경계는 유지한다.
- Operation Result는 inspect·보류·차단·설치 실패를 포함한 모든 사용자 operation의 최종 상태를 표현한다.
- 정확한 schema·option·snapshot·continuation·exit code·CI mapping은 Interface Contract와 구현 설계에서 확정한다.

## Domain-007 누락 점검

- [x] Operation Request의 역할과 추적 대상
- [x] Operation Request와 Inspection Run·Policy Decision 구분
- [x] MVP Operation Type인 `install`과 `inspect` 구분
- [x] Install Context의 역할과 사용자 설치 의도 보존
- [x] Install Context와 Sandbox Context의 보안 경계
- [x] `install + ALLOW` 자동 continuation UX
- [x] `inspect`의 Host 무반입 원칙
- [x] Promotion의 원래 요청 불변성
- [x] MANUAL_REVIEW / BLOCK의 자동 Promotion 금지
- [x] Policy Decision과 Operation Status 분리
- [x] ALLOW·Manifest/exact Artifact·Install Context 일관성
- [x] Promotion 중 자유로운 재-resolution 금지
- [x] standalone Artifact의 Staging 예외와 trusted Promotion
- [x] Operation Result의 전체 operation 적용
- [x] 8개 invariant
- [x] 후속 결정 항목 유보
