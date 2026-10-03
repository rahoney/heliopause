# Domain Model — Operation Request and Context

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
        │  install이면
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

Package ecosystem에서는 Artifact identity와 dependency가 Verified Manifest와 일치해야 하고, Standalone에서는 검사한 exact Artifact identity/digest와 일치해야 한다. 실행 대상은 두 경우 모두 원래 trusted Operation/Install Context와 일치해야 한다.

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
Package ecosystem
→ Verified Manifest

또는

Standalone
→ exact Artifact identity / digest

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
