# Domain Model — Dependency and Verified Set

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
