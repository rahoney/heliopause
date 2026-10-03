# Interface Contract — Artifact Port

## Contract-001: Artifact Port

### 목적

`Artifact Port`는 Heliopause의 Application/Workflow가 npm, PyPI, GitHub Releases 같은 구체적인 생태계 구현에 직접 의존하지 않고 Artifact를 **해석·resolve·획득**하기 위한 공통 계약이다.

```text
Application / Workflow
        ↓
Artifact Port
        ↓
Artifact Adapter
        ├─ npm
        ├─ PyPI
        └─ GitHub Releases
```

Core/Application은 npm registry API, pip command, GitHub API 등의 구체적인 사용법을 알지 않는다.

### Artifact Port의 책임

Artifact Port는 개념적으로 다음 세 책임을 제공한다.

```text
Identify / Parse
→ 사용자 입력을 Artifact Reference로 해석

Resolve
→ mutable/ambiguous reference를 exact Artifact identity로 확정

Acquire
→ 확정된 Artifact를 Controlled Intake로 획득
```

기본 흐름은 다음과 같다.

```text
사용자 입력
    ↓
Artifact Reference
    ↓
Resolve
    ↓
Resolved Artifact Identity
    ↓
Acquire
    ↓
Acquired Artifact
```

구체적인 Go interface와 method signature는 구현 직전 세부 설계에서 확정한다.

### Identify / Parse

Artifact Adapter는 자신이 담당하는 ecosystem/source의 입력을 해석하여 공통 `Artifact Reference`로 변환한다.

예:

```text
npm + kenv@latest
        ↓
Artifact Reference

PyPI + requests
        ↓
Artifact Reference

GitHub Releases + repository/release/asset selector
        ↓
Artifact Reference
```

Adapter는 ecosystem-specific 표현을 Core Domain까지 그대로 퍼뜨리지 않고 공통 Domain 의미로 변환한다.

입력이 안전하게 해석되지 않거나 필요한 정보가 모호하면 임의로 추측하지 않고 명시적인 오류 또는 추가 입력 필요 상태를 반환한다.

### Resolve

`Resolve`는 Artifact Reference를 실제 acquisition 대상으로 사용할 exact `Resolved Artifact Identity`로 확정한다.

예:

```text
kenv@latest
        ↓
Resolve
        ↓
kenv@1.2.3
```

또는:

```text
GitHub Release = latest
Asset selector
        ↓
Resolve
        ↓
Exact repository
Exact release/tag
Exact asset
```

Resolve가 끝난 뒤에는 acquisition 대상의 logical identity가 정확하게 결정되어야 한다.

`latest`, version 생략, mutable release reference 등을 최종 identity로 그대로 넘기지 않는다.

### Resolve와 안전성 판정의 구분

Artifact Adapter가 official registry 또는 repository에서 Artifact를 resolve했다고 해서 안전한 Artifact로 판정하지 않는다.

```text
Resolve 성공
≠
Artifact 안전
```

Artifact Adapter의 역할은:

```text
무엇을 요청했는가
→ 정확히 무엇을 받을 것인가
```

를 결정하는 것이다.

다음은 Artifact Port의 책임이 아니다.

```text
malicious 여부 판정
최종 ALLOW / BLOCK 결정
signature 안전성 정책
Finding severity 결정
```

이들은 Verification / Inspection / Policy 영역에서 처리한다.

### Acquire

`Acquire`는 `Resolved Artifact Identity`에 해당하는 실제 Artifact content를 Heliopause의 Controlled Intake 영역으로 획득한다.

```text
Resolved Artifact Identity
        ↓
Acquire
        ↓
Controlled Intake
        ↓
Observed Digest 계산
        ↓
Acquired Artifact
```

획득된 content는 Domain-002에서 정의한 exact identity와 observed digest에 연결되어야 한다.

### Acquire의 Host 경계

Artifact Adapter가 acquisition을 수행하더라도 임의의 Host filesystem 또는 실제 프로젝트 디렉토리에 Artifact를 직접 설치·전개하지 않는다.

```text
Artifact Adapter
        ↓
Controlled Intake

Artifact Adapter
        X
        ↓
User Project / Host Install
```

Acquire는 **반입 준비**이지 Promotion이나 install이 아니다.

Archive extraction, install script 실행, package lifecycle 실행 등 실행 위험이 있는 작업을 acquisition 책임에 포함하지 않는다.

필요한 extraction/inspection/install execution은 후속 Inspection/Sandbox workflow에서 통제한다.

### Digest 처리

실제 획득한 content의 observed digest는 Heliopause가 직접 계산할 수 있어야 한다.

```text
실제 acquired bytes
        ↓
Observed Digest
```

Registry 또는 publisher가 제공한 digest/integrity metadata는 별도의 declared/expected verification input으로 취급한다.

```text
Declared Digest
≠
Observed Digest
```

Artifact Adapter는 source가 제공하는 integrity metadata를 수집할 수 있지만 이를 실제 observed digest 또는 안전성 판정으로 대체하지 않는다.

### Source Metadata

Artifact Adapter는 Verification 등에 필요한 source metadata를 제공할 수 있다.

예:

```text
registry integrity metadata
publisher information
release metadata
checksum reference
signature/provenance location
dependency metadata
```

그러나 이러한 정보는:

```text
Artifact Identity
또는
Artifact Safety Decision
```

과 동일하지 않다.

Artifact Adapter는 source-specific 정보를 공통 계약으로 전달하고 실제 검증과 해석은 해당 책임 영역에 맡긴다.

### Dependency Resolution

Package ecosystem Adapter는 dependency requirement와 resolution 결과를 제공할 수 있어야 한다.

```text
Primary Artifact
        ↓
Dependency Requirements
        ↓
Resolve
        ↓
Resolved Dependency Graph
```

각 실제 dependency는 Domain-002와 Domain-006에서 정의한 동일한 Artifact identity 원칙을 따른다.

```text
Dependency
→ exact identity
→ acquired content
→ observed digest
```

Artifact Port는 dependency를 발견·resolve하는 역할을 수행할 수 있지만, dependency를 검증되었다고 판정하거나 Verified Set에 자동 편입하지 않는다.

### 새로운 Dependency 발견

검사 또는 실제 install 준비 과정에서 기존 resolution에 없던 Artifact가 발견되면 Adapter가 이를 자동으로 신뢰하여 Host에 설치하지 않는다.

```text
새 Artifact 발견
        ↓
Artifact workflow로 반환
        ↓
Resolve / Acquire
        ↓
Verification / Inspection
```

새 Artifact가 Verified Set 경계를 우회하지 못하게 한다.

### Capability 표현

모든 Artifact Adapter가 동일한 기능을 제공해야 하는 것은 아니다.

예:

```text
npm
→ dependency resolution 지원

GitHub standalone binary
→ dependency resolution 해당 없음

특정 source
→ provenance metadata 조회 미지원
```

Adapter가 제공하는 기능과 지원 범위는 별도의 Adapter capability/support 정보로 명시적으로 표현할 수 있어야 한다. 이는 Domain-004에서 정의한 Verification/Inspection의 Run-level `Capability`와 동일한 개념으로 취급하지 않는다. 지원하지 않는 기능을 성공 또는 빈 결과로 위장하지 않는다.

```text
Unsupported
≠
Empty
≠
Success
```

정확한 Adapter 기능 지원 정보의 모델과 discovery 방식은 후속 interface 상세 설계에서 확정한다.

### Error와 Domain Result의 구분

다음과 같은 operational failure:

```text
registry timeout
network failure
authentication failure
malformed response
download failure
```

를 Artifact의 보안 판정으로 변환하지 않는다.

예:

```text
Acquire 실패
→ operational error

Acquire 실패
≠
Policy BLOCK
```

Application은 해당 오류를 Inspection Run / Operation 상태 모델에 따라 처리한다.

### Artifact Adapter의 금지 책임

Artifact Adapter는 다음 책임을 갖지 않는다.

```text
최종 Policy Decision
Finding 생성 정책
Dynamic Inspection 수행
Sandbox lifecycle 관리
Verified Set 승인
Staging
Promotion
Host installation
```

특히:

```text
Artifact Adapter
→ ALLOW
```

또는:

```text
Artifact Adapter
→ User Host에 직접 install
```

하는 구조를 사용하지 않는다.

### Contract-001 Invariant

#### Invariant 1 — Core는 ecosystem 구현에 직접 의존하지 않는다

npm/PyPI/GitHub 등의 세부 동작은 Artifact Adapter 뒤에 둔다.

#### Invariant 2 — Reference / Resolved Identity / Acquired Artifact를 구분한다

입력, exact logical identity, 실제 획득 content를 하나의 객체로 혼합하지 않는다.

#### Invariant 3 — Mutable Reference는 acquisition 전에 resolve한다

`latest` 등을 그대로 최종 검사 identity로 사용하지 않는다.

#### Invariant 4 — Acquire는 Controlled Intake까지만 담당한다

Artifact Adapter가 Host project/environment에 직접 설치하지 않는다.

#### Invariant 5 — 실제 acquired content에 observed digest를 연결한다

Source가 제공한 digest만으로 실제 획득 content identity를 대신하지 않는다.

#### Invariant 6 — Resolution 성공은 안전 판정이 아니다

Official registry/source에서 정상 획득했다는 사실을 `ALLOW`로 해석하지 않는다.

#### Invariant 7 — Dependency도 동일한 Artifact pipeline을 따른다

새 dependency를 자동 신뢰하거나 Verified Set에 우회 편입하지 않는다.

#### Invariant 8 — Unsupported와 operational failure를 명시적으로 구분한다

지원하지 않는 기능이나 실행 실패를 정상적인 빈 결과로 숨기지 않는다.

#### Invariant 9 — Adapter는 최종 보안 책임을 갖지 않는다

Verification / Inspection / Policy / Promotion 책임을 Artifact Adapter 내부로 끌어들이지 않는다.

### 후속 결정으로 유보한 항목

Contract-001에서는 다음을 아직 확정하지 않는다.

- 실제 Go interface 이름과 method signature
- request/response struct
- `context.Context` 사용 방식
- Artifact Adapter registration 방식
- source authentication 전달 방식
- network client 구현
- retry/timeout 정책
- acquisition storage handle
- dependency graph schema
- source metadata schema
- Adapter capability/support API
- error type/code taxonomy
- npm/PyPI/GitHub별 상세 Adapter Contract

### 압축된 Contract

```text
Artifact Port

Identify / Parse
    ↓
Artifact Reference

Resolve
    ↓
Resolved Artifact Identity

Acquire
    ↓
Controlled Intake
    ↓
Acquired Artifact + Observed Digest

Dependency Resolution
    ↓
Resolved Dependency Graph
```

Artifact Port의 경계는 다음과 같다.

```text
"무엇인가?"
"정확히 무엇을 받을 것인가?"
"그 정확한 content를 통제된 영역으로 가져와라."
```

까지 담당한다.

```text
"안전한가?"
"허용할 것인가?"
"실제 Host에 설치하라."
```

는 Artifact Port의 책임이 아니다.

## Contract-001 구조화된 결정과 구현 영향

- Application/Workflow는 Artifact Port만 의존하고 npm·PyPI·GitHub Releases 구현은 Adapter로 격리한다.
- Port는 Identify/Parse, Resolve, Acquire와 필요 시 dependency resolution을 담당한다.
- 입력 Reference, exact Resolved Identity, 실제 Acquired Artifact와 observed digest를 분리해 추적한다.
- mutable 또는 모호한 reference는 acquisition 전에 exact identity로 resolve한다.
- Acquire는 Controlled Intake까지만 수행하며 Host install·archive extraction·lifecycle script 실행을 수행하지 않는다.
- source의 declared integrity metadata는 수집할 수 있으나 직접 계산한 observed digest를 대체하지 않는다.
- source metadata와 dependency resolution 결과는 검증 input일 뿐 Policy 안전 판정이나 Verified Set 편입을 의미하지 않는다.
- 신규 dependency는 동일한 Resolve/Acquire/Verification/Inspection workflow로 회귀한다.
- Adapter capability/support와 Run-level Capability를 분리하고 Unsupported·Empty·Success 및 operational error를 명시적으로 구분한다.
- Artifact Adapter는 Policy, Finding, Sandbox, Staging, Promotion, Host installation 책임을 갖지 않는다.
- Go interface·schema·authentication·retry·storage·error taxonomy 및 ecosystem별 세부 계약은 구현 직전 설계로 유보한다.

## Contract-001 누락 점검

- [x] Core/Application과 ecosystem Adapter 분리
- [x] Identify / Parse 책임
- [x] Resolve 책임과 exact identity 확정
- [x] Resolve와 안전성 판정 분리
- [x] Acquire와 Controlled Intake 경계
- [x] Host installation·실행 위험 작업 제외
- [x] declared/observed digest 구분
- [x] source metadata의 역할과 경계
- [x] dependency resolution과 동일 identity 원칙
- [x] 신규 dependency의 workflow 회귀
- [x] Adapter capability/support와 Run-level Capability 구분
- [x] Unsupported·Empty·Success 및 operational error 구분
- [x] Artifact Adapter의 금지 책임
- [x] 9개 invariant
- [x] 후속 결정 항목 유보
