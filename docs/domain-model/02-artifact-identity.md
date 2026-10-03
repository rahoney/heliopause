# Domain Model — Artifact Identity

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
Policy = ALLOW
        ↓
Package ecosystem
→ Verified Set / Manifest
→ identity / digest 재확인

Standalone
→ exact identity / digest 재확인
        ↓
trusted Promotion
```

```text
사용자가 처음 입력한 Reference
→ 요청 추적용

Resolved Artifact Identity
→ logical target 추적용

Observed Digest
→ 실제 content 추적용

Package ecosystem의 Verified Set / Manifest
→ 검증 완료 Artifact 집합 추적용

Standalone의 exact identity / observed digest
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

Inspection Run 생성
      ↓
Controlled Intake
      ↓
실제 bytes 획득
      ↓
observed digest = AAA
      ↓
Acquired Artifact binding

Verification / Inspection
      ↓
Policy

inspect
→ Promotion 없음

install + ALLOW + Package ecosystem
→ Verified Set / Manifest
→ 동일 identity + digest 재확인
→ Promotion

install + ALLOW + Standalone
→ exact identity + digest 재확인
→ trusted Promotion
```

핵심 원칙은 다음과 같다.

```text
사용자가 가리킨 것
≠
source가 resolve한 것
≠
실제로 획득한 것

그리고

Promotion이 발생하는 경우:

실제로 검사한 것
=
Policy가 허용한 것
=
실제로 Promotion하는 것

Package ecosystem에서는 추가로:

Policy가 허용한 Artifact 집합
=
Verified Set / Manifest가 고정한 Artifact 집합
=
실제로 Promotion하는 Artifact 집합
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
