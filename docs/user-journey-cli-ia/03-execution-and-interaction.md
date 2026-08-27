# User Journey + CLI IA — Execution and Interaction

## User-Journey-004: 기본 End-to-End 실행 흐름

### 사용자 원문

기본 사용자 경험은 사용자가 기존에 수행하려던 Artifact 설치·반입 작업을 하나의 Heliopause 명령으로 요청하면, Heliopause가 실제 사용자 환경에 Artifact를 반입하기 전에 필요한 보안 검사 workflow를 자동으로 수행하고 그 결과에 따라 원래 작업을 계속하거나 중단하는 방식으로 구성한다.

사용자는 일반적인 설치 과정에서 acquisition, Verification, Static Inspection, Sandbox/Dynamic Inspection, Policy, Staging, Promotion 등의 내부 단계를 개별적으로 실행하거나 순서를 직접 조작할 필요가 없다.

예를 들어 다음 명령은:

```text
helox npm install kenv
```

사용자 관점에서는 하나의 설치 작업이지만, 내부적으로는 다음 End-to-End workflow를 수행한다.

```text
사용자 install 요청
        ↓
입력 해석 및 source/ecosystem 식별
        ↓
exact version/release/asset resolve
        ↓
acquisition / Controlled Intake
        ↓
Artifact identity + digest 확정
        ↓
dependency 및 관련 구성요소 resolution
        ↓
Verification
        ↓
Static Inspection
        ↓
필요한 경우 Sandbox / Dynamic Inspection
        ↓
Policy Decision
        ↓
        ├─ ALLOW
        │    ↓
        │  Verified Set / Verified Manifest 확정
        │    ↓
        │  identity / digest 재검증
        │    ↓
        │  Staging / trusted Promotion
        │    ↓
        │  사용자가 처음 요청한 설치·반입 작업 완료
        │
        ├─ MANUAL_REVIEW
        │    ↓
        │  자동 설치 및 반입 진행 보류
        │    ↓
        │  검토에 필요한 Finding·Evidence·검사 한계 제공
        │
        └─ BLOCK
             ↓
           설치·반입 금지
             ↓
           위험 실행 환경 및 임시 검사 Artifact 정리
             ↓
           차단 근거와 관련 결과 제공
```

### `install` End-to-End 원칙

`install`은 Heliopause의 대표적인 End-to-End 사용자 작업이다.

사용자가 `install`을 요청했다고 해서 Heliopause가 대상 Artifact를 실제 Host 또는 사용자 프로젝트 환경에 먼저 설치한 뒤 검사해서는 안 된다. 원래 설치 작업은 Heliopause의 검증·검사·Policy 과정을 통과하고 최종 Decision이 `ALLOW`인 경우에만 trusted Promotion 경계를 통해 계속 수행한다.

따라서 기본 의미는 다음과 같다.

```text
install 요청
≠ 즉시 설치

install 요청
= 보안 검사 선행 + ALLOW 시 원래 설치 작업 계속
```

`ALLOW`가 발생하면 사용자가 별도의 `promote` 명령을 다시 입력하도록 요구하지 않는다. 최초 install 요청의 ecosystem/source, operation 및 설치 대상 환경 등 원래 작업의 실행 context를 검사 workflow 동안 유지하며, `ALLOW` 이후에는 검증된 Artifact와 해당 원래 요청을 연결하여 작업을 계속 수행한다. 구체적인 option 전달·허용·검증 규칙은 후속 CLI IA에서 정의한다.

### `inspect` End-to-End 원칙

`inspect`는 설치·반입 없이 동일한 보안 검사 workflow를 수행하기 위한 작업이다.

```text
helox npm inspect kenv
```

`inspect` 역시 Artifact resolution, acquisition/Controlled Intake, Verification, Static Inspection, 필요한 경우 Sandbox/Dynamic Inspection 및 Policy Decision까지 수행한다. 그러나 최종 Decision이 `ALLOW`여도 Staging/Promotion을 통해 사용자 환경에 설치·반입하지 않는다.

```text
inspect 요청
        ↓
전체 검사 workflow
        ↓
Policy Decision
        ↓
Inspection Run 및 결과 기록
        ↓
사용자에게 결과 제공
        ↓
종료
```

따라서 `install`과 `inspect`는 검사 과정의 상당 부분을 공유하지만 최종 사용자 의도가 다르다.

```text
install
→ 검사
→ ALLOW
→ 안전한 Promotion
→ 원래 작업 완료

inspect
→ 검사
→ Policy Decision
→ 결과 제공
→ 종료
```

### Dynamic Inspection 실행

Sandbox/Dynamic Inspection은 모든 Artifact에 기계적으로 무조건 수행하는 고정 단계로 간주하지 않는다.

Artifact의 종류, 해당 Adapter/Provider의 Capability, 수행 가능한 검사, 정적 검사 결과 및 향후 정의되는 Inspection/Policy 규칙에 따라 Dynamic Inspection이 필요한 경우에 수행한다.

Dynamic Inspection이 보안상 필수인 상황에서 해당 검사를 수행할 수 없거나 실패·미완료 상태가 된 경우 이를 정상 또는 안전한 결과로 해석하지 않는다. 관련 Execution Status와 사유를 기록하고 기존 fail-closed 원칙에 따라 Policy Decision에 반영한다.

Dynamic Inspection의 구체적인 필수 조건, 생략 조건 및 검사 조합은 후속 Policy/Domain Model 및 Inspection 설계에서 확정한다.

### Dependency 처리

사용자는 Primary Artifact를 지정하며, 설치 과정에서 필요한 dependency와 관련 구성요소의 resolution·acquisition·검증·검사는 Heliopause가 내부 workflow에서 orchestration한다.

일반적인 과정에서 dependency별로 사용자에게 반복적인 검사 승인이나 내부 단계 선택을 요구하지 않는다.

Package ecosystem에서는 실제 검사·설치 과정에서 사용될 Primary Artifact와 dependency를 Verified Set으로 관리하며, 최종 설치·반입에는 검증 완료된 정확한 Artifact set만 사용한다.

검사 또는 최종 설치 과정에서 기존 Verified Set에 포함되지 않은 새로운 dependency나 Artifact가 요구되면 이를 자동으로 신뢰하거나 사용자 환경에 설치하지 않는다. 해당 대상을 acquisition/verification 흐름으로 되돌리고 필요하면 full quarantine 검사를 수행한다.

### 실행 실패와 Inspection Run

End-to-End workflow 중 일부 단계가 실패하더라도 Heliopause는 가능한 범위에서 해당 실행의 Inspection Run과 확보된 검사 정보를 보존한다.

다음과 같은 상태는 서로 구분하여 기록한다.

- `Completed`
- `Failed`
- `Incomplete`
- `Skipped / Not Executed`
- `Unavailable`

실패·미완료·사용 불가 상태를 단순한 검사 성공이나 안전 결과로 변환하지 않는다.

보안상 필수 단계가 완료되지 못한 경우에는 기존 fail-closed 원칙에 따라 `MANUAL_REVIEW` 또는 `BLOCK`으로 연결할 수 있으며, 정확한 Policy rule은 후속 Policy 설계에서 결정한다.

### 임시 환경과 Evidence 정리 원칙

`BLOCK` 또는 검사 실패가 발생하면 신뢰하지 않는 Sandbox Session과 위험한 임시 Artifact는 재사용하지 않고 정리한다.

다만 차단·실패 원인을 추적하는 데 필요한 Inspection Run, Raw Evidence, Finding, Verification Result, Policy Decision 및 관련 기록은 Evidence retention 정책에 따라 보존한다.

```text
위험 Artifact / Sandbox 실행 환경
→ 정리·폐기

Evidence / Inspection Run / 결과 기록
→ 보존
```

### Ecosystem별 차이

npm, PyPI/pip, GitHub Releases는 사용자 관점에서 가능한 한 동일한 End-to-End 원칙을 공유한다.

Package ecosystem의 `install`은 검증 완료된 Verified Set을 사용해 원래 설치 작업을 수행한다.

Package manager가 필요 없는 standalone binary/archive는 Architecture-008에 정의된 예외에 따라 Staging을 생략할 수 있으나, `ALLOW`와 identity/digest 재검증 후 trusted Promotion 경계를 통해서만 사용자 지정 위치로 반입한다.

이러한 ecosystem별 내부 차이를 일반 사용자가 별도의 보안 workflow로 직접 조작하도록 요구하지 않는다.

### 후속 결정으로 유보한 항목

- npm/pip 등 원본 package manager option의 pass-through 범위
- `-D`, `-g`, `--save-exact` 등 ecosystem별 option 처리
- 검사 단계별 retry 정책
- interactive confirmation이 필요한 조건
- `MANUAL_REVIEW` 후 사용자가 취할 수 있는 후속 action
- CI/non-interactive 실행 방식
- terminal progress 표시 방식
- exit code
- 구체적인 cleanup·retention 기간
- Dynamic Inspection 필수·생략 rule

이 항목들은 이후 User Journey / CLI IA 및 Domain Model·Policy·Tooling 단계에서 구체화한다.

### 압축된 사용자 모델

```text
helox <ecosystem> install <artifact>
        ↓
Heliopause가 보안 검사 자동 수행
        ↓
        ├─ Policy = ALLOW → 원래 설치·반입 작업 완료
        ├─ Policy = BLOCK → 작업 차단 + 이유 제공
        └─ Policy = MANUAL_REVIEW → 작업 보류 + 검토 정보 제공
```

사용자는 복잡한 내부 보안 pipeline을 직접 운영하지 않고, 원래 수행하려던 개발 작업을 Heliopause를 통해 요청하는 것만으로 전체 보안 검사와 안전한 반입 절차를 적용받는다.

### 구조화된 결정과 구현 영향

- `install`은 사용자의 원래 설치·반입 의도를 유지한 End-to-End workflow로 동작한다.
- `install`은 실제 Host 또는 사용자 프로젝트 환경에 선설치하지 않고, 검사·Policy를 먼저 수행한다.
- `ALLOW` 시 별도 `promote` 명령 없이 원래 install context와 Verified Set을 연결해 trusted Promotion을 계속한다.
- `inspect`는 동일한 검사 workflow를 수행하지만 최종 `ALLOW`여도 Staging/Promotion을 수행하지 않는다.
- Dynamic Inspection은 Capability·검사 결과·Policy 규칙에 따라 선택적으로 수행하며 필수 검사 불가·실패·미완료는 안전 결과로 해석하지 않는다.
- dependency는 내부 orchestration하고 Verified Set 밖의 새 Artifact는 acquisition/verification 및 필요 시 full quarantine으로 되돌린다.
- 실행 상태 `Completed`·`Failed`·`Incomplete`·`Skipped / Not Executed`·`Unavailable`을 구분하고 실패를 안전 결과로 변환하지 않는다.
- 위험 Sandbox Session·임시 Artifact는 정리·폐기하되 Inspection Run·Evidence·Finding·Verification Result·Policy Decision은 retention에 따라 보존한다.
- Package는 Verified Set 경로를, standalone binary/archive는 Architecture-008의 Staging 생략·digest 재검증·trusted Promotion 예외를 사용한다.

## User-Journey-004 누락 점검

- [x] install 요청 전 보안 검사 선행
- [x] 입력·source 식별 및 exact version/release/asset resolve
- [x] acquisition·Controlled Intake·identity/digest 확정
- [x] dependency resolution 및 Verified Set 관리
- [x] Verification·Static Inspection·조건부 Dynamic Inspection
- [x] `ALLOW` 후 Verified Manifest·digest 재검증·Staging·trusted Promotion
- [x] `ALLOW` 시 원래 install 요청을 별도 promote 없이 계속
- [x] `MANUAL_REVIEW` 설치·반입 보류와 검토 정보 제공
- [x] `BLOCK` 설치·반입 금지와 임시 환경·Artifact 정리
- [x] inspect의 설치·반입 없는 동일 검사 workflow
- [x] Dynamic Inspection Capability·Execution Status·fail-closed 처리
- [x] dependency의 내부 orchestration과 새 dependency 재검역
- [x] Inspection Run 실행 상태 구분과 실패 정보 보존
- [x] 위험 환경 정리와 Evidence·결과 기록 보존 분리
- [x] npm·PyPI/pip·GitHub Releases 공통 원칙과 standalone 예외
- [x] 후속 CLI IA·Policy·Domain Model·Tooling 유보 항목 기록

## User-Journey-005: 자동 실행과 사용자 개입 경계

### 사용자 원문

Heliopause는 일반적인 End-to-End 작업에서 불필요한 사용자 확인을 반복하지 않고, 사용자가 처음 요청한 작업과 Policy Decision을 기준으로 자동 실행과 사용자 개입의 경계를 결정한다.

기본 원칙은 다음과 같다.

```text
ALLOW
→ 원래 요청한 작업 자동 계속

MANUAL_REVIEW
→ 자동 작업 진행 보류
→ 사용자 검토 필요

BLOCK
→ 작업 자동 차단
```

### `ALLOW` 처리

사용자가 `install`을 명시적으로 요청했고 최종 Policy Decision이 `ALLOW`인 경우 Heliopause는 일반적인 상황에서 별도의 추가 confirmation을 요구하지 않고 검증된 Artifact를 사용하여 원래 요청한 설치·반입 작업을 계속 수행한다.

```text
helox npm install kenv
        ↓
검사
        ↓
Policy = ALLOW
        ↓
Verified Set / Manifest 확인
        ↓
trusted Promotion
        ↓
npm 설치 작업 완료
```

사용자는 이미 `install`을 실행함으로써 설치 의사를 명확히 표현했으므로, 정상적인 `ALLOW` 이후 다시 설치 여부를 묻는 것을 기본 동작으로 하지 않는다.

`inspect` 요청의 경우 `ALLOW`가 발생해도 설치·반입하지 않고 결과를 제공한 뒤 종료한다.

### `MANUAL_REVIEW` 처리

`MANUAL_REVIEW`는 Heliopause가 확보한 정보만으로 자동 `ALLOW` 또는 `BLOCK`을 결정하기에 충분하지 않거나 사람의 판단이 필요한 상태를 의미한다.

`MANUAL_REVIEW`가 발생하면 진행 중인 자동 설치·반입을 보류하고 사용자에게 다음 정보를 제공한다.

- 왜 자동 판단할 수 없었는지
- 주요 Finding
- 관련 Evidence
- Verification Result
- 실패·미완료·미지원 또는 사용 불가 검사
- 검사 한계
- Inspection Run 식별 정보

기본 동작에서 단순한 다음 confirmation으로 보안 경계를 우회하지 않는다.

```text
위험할 수도 있습니다.
그래도 설치하시겠습니까? [y/N]
```

즉 `MANUAL_REVIEW`를 단순한 경고형 `yes/no` prompt로 축소하지 않는다.

사용자는 우선 `review`를 통해 관련 결과를 확인할 수 있어야 한다.

```text
Policy = MANUAL_REVIEW
        ↓
자동 설치·반입 보류
        ↓
review 가능
        ↓
사람의 판단 또는 추가 검사 필요
```

사람의 검토 이후 기존 Inspection Run을 승인·재개할 수 있는 별도의 override/approval 기능을 제공할지, 제공한다면 어떤 권한·기록·재검증을 요구할지는 후속 CLI/Policy 설계에서 결정한다.

MVP에서 이러한 안전한 approval 경로가 아직 정의되지 않은 경우 `MANUAL_REVIEW` 상태에서 직접 설치·반입을 허용하지 않는다.

### `BLOCK` 처리

최종 Policy Decision이 `BLOCK`이면 사용자의 추가 confirmation 없이 설치·반입을 차단한다.

```text
Policy = BLOCK
        ↓
원래 install 작업 중단
        ↓
사용자 환경 반입 금지
        ↓
위험 Sandbox Session / 임시 Artifact 정리
        ↓
차단 이유와 주요 Finding 제공
```

일반적인 사용자 confirmation으로 `BLOCK`을 우회하지 않는다.

```text
BLOCK
→ "그래도 설치?" 제공하지 않음
```

구체적인 예외·override 기능이 향후 필요해지더라도 일반 설치 흐름과 분리하고, 명시적 권한·감사 기록·재검증을 갖는 별도의 보안 기능으로 설계한다.

### 사용자 입력이 필요한 경우

Policy Decision 이외에도 Heliopause가 안전하게 다음 단계로 진행하기 위해 필요한 정보가 모호하거나 부족한 경우에는 사용자 입력을 요구할 수 있다.

예를 들어:

- ecosystem/source를 하나로 결정할 수 없는 경우
- GitHub Release에 복수의 대상 asset이 있어 하나를 선택할 수 없는 경우
- 설치 대상 환경이나 위치가 필요한데 명확하지 않은 경우
- 서로 다른 의미를 가질 수 있는 입력이나 option이 존재하는 경우

이 경우 Heliopause는 보안상 의미 있는 값을 임의로 추측하여 진행하지 않는다.

사용자 입력 요구는 정상적인 보안 검사 단계마다 반복되는 confirmation이 아니라, 안전한 실행을 위해 필요한 정보가 실제로 부족한 경우에 한정한다.

### 일반적인 confirmation 최소화

Heliopause는 다음과 같은 반복적인 confirmation을 기본 사용자 경험으로 만들지 않는다.

```text
Artifact를 다운로드할까요?
검사를 시작할까요?
Dynamic Inspection을 할까요?
dependency를 검사할까요?
ALLOW되었습니다. 설치할까요?
```

이러한 내부 단계는 사용자가 처음 요청한 `install` 또는 `inspect` 의도와 Heliopause의 검사·Policy 규칙에 따라 자동으로 orchestration한다.

따라서 일반적인 사용자 경험은 다음과 같이 유지한다.

```text
helox npm install kenv

→ 필요한 검사 자동 수행
→ ALLOW         : 설치 완료
→ MANUAL_REVIEW : 작업 보류 + 검토 필요
→ BLOCK         : 설치 차단
```

### Interactive 실행

사람이 직접 terminal에서 실행하는 interactive mode에서는 진행 상태와 최종 Decision을 사람이 이해할 수 있는 형태로 표시한다.

사용자 선택이 실제로 필요한 경우에만 prompt를 표시하며, 단순 진행 확인을 위해 각 검사 단계마다 입력을 요구하지 않는다.

구체적인 terminal UI, progress 표시, prompt 문구 및 선택 방식은 후속 CLI 출력 설계에서 정의한다.

### CI / Non-interactive 실행

CI 또는 기타 non-interactive 환경에서는 사용자 입력을 기다리는 prompt를 표시하지 않는다.

기본 동작은 다음과 같다.

```text
ALLOW
→ 요청 작업 성공 경로 계속

MANUAL_REVIEW
→ 자동 진행 중단
→ machine-readable 결과와 상태 반환

BLOCK
→ 자동 진행 차단
→ machine-readable 결과와 상태 반환
```

모호한 입력이나 사용자 판단이 필요한 상태가 발생했는데 non-interactive 환경에서 이를 해결할 수 없는 경우 값을 임의 추측하거나 안전 경계를 완화하지 않고 실행을 중단한다.

구체적인 exit code, CI status mapping 및 machine-readable output schema는 후속 CLI/Domain Model 설계에서 정의한다.

### 원본 개발 도구의 사용자 interaction

npm, pip 등 원본 ecosystem tool 자체가 특정 작업에서 사용자 interaction을 요구하는 경우 이를 무조건 그대로 통과시키거나 자동 승인하지 않는다.

Heliopause가 어떤 option과 interactive 동작을 허용·차단·중개할지는 후속 ecosystem별 CLI contract에서 정의한다.

Heliopause 자체의 보안 경계를 원본 package manager의 prompt나 option이 우회할 수 없도록 한다.

### 후속 결정으로 유보한 항목

- `MANUAL_REVIEW` 이후 approval / override 기능 제공 여부
- approval을 제공할 경우 필요한 권한과 감사 기록
- interactive prompt의 정확한 문구
- terminal progress UI
- CI exit code
- CI status mapping
- machine-readable output schema
- package manager별 interactive option 처리
- 관리자 policy와 사용자 override 권한
- 재검사·retry 이후 자동 재개 조건

위 항목은 후속 User Journey / CLI IA, Policy, Domain Model 및 Tooling 설계에서 정의한다.

### 압축된 사용자 모델

```text
사용자가 install 요청
        ↓
Heliopause 자동 검사
        ↓
   ┌───────────────────────────┐
   │ ALLOW                     │
   │ → 별도 질문 없이 계속    │
   │ → 설치 완료               │
   ├───────────────────────────┤
   │ MANUAL_REVIEW             │
   │ → 자동 진행 보류          │
   │ → 검토 정보 제공          │
   ├───────────────────────────┤
   │ BLOCK                     │
   │ → 자동 차단               │
   │ → 일반 사용자 우회 불가   │
   └───────────────────────────┘
```

Heliopause는 정상적인 작업에서는 사용자의 개입을 최소화하고, 안전한 실행에 필요한 정보가 부족하거나 사람의 판단이 필요한 경우에만 사용자 입력 또는 검토를 요구하며, `BLOCK` 상태에서는 작업을 자동 차단하는 것을 기본 원칙으로 한다.

### 구조화된 결정과 구현 영향

- 명시적인 `install` 요청과 `ALLOW`가 함께 있으면 추가 confirmation 없이 원래 설치·반입을 계속한다.
- `inspect` 요청은 `ALLOW`여도 설치·반입하지 않고 결과를 제공한 뒤 종료한다.
- `MANUAL_REVIEW`는 자동 설치·반입을 보류하고 review 가능한 근거와 검사 한계를 제공한다.
- `MANUAL_REVIEW`를 단순 yes/no 경고 prompt로 처리하거나 안전 경계를 우회하는 수단으로 사용하지 않는다.
- MVP에서 승인·override 경로가 정의되지 않았다면 `MANUAL_REVIEW`에서 직접 설치·반입하지 않는다.
- `BLOCK`은 추가 confirmation 없이 자동 차단하고 일반 사용자 confirmation으로 우회하지 않는다.
- override가 필요해지면 일반 install과 분리된 권한·감사 기록·재검증 기능으로 설계한다.
- ecosystem/source·asset·설치 대상 환경·option이 모호하면 임의 추측하지 않고 필요한 입력만 요구한다.
- 내부 단계별 반복 confirmation은 사용하지 않고 처음 요청한 `install`·`inspect` 의도에 따라 자동 orchestration한다.
- interactive mode는 필요한 경우에만 prompt를 표시하고, CI/non-interactive에서는 prompt 없이 machine-readable 결과와 상태를 반환한다.
- non-interactive에서 모호성·사람 판단 상태를 해결할 수 없으면 안전 경계를 완화하지 않고 중단한다.
- npm/pip 등 원본 도구의 prompt·option이 Heliopause 보안 경계를 우회하지 않도록 ecosystem별 contract에서 통제한다.

## User-Journey-005 누락 점검

- [x] ALLOW·MANUAL_REVIEW·BLOCK 자동 실행 경계
- [x] install ALLOW 후 추가 confirmation 없는 원래 작업 계속
- [x] inspect ALLOW 후 설치·반입 없이 종료
- [x] MANUAL_REVIEW의 보류·review·근거 제공
- [x] MANUAL_REVIEW의 단순 yes/no 우회 금지
- [x] MVP approval 경로 미정 시 직접 반입 금지
- [x] BLOCK의 추가 confirmation 없는 자동 차단
- [x] BLOCK의 일반 사용자 override 금지
- [x] 모호한 source·asset·환경·option의 사용자 명확화
- [x] 반복 confirmation 최소화
- [x] interactive mode의 필요한 경우 한정 prompt
- [x] CI/non-interactive의 no-prompt·machine-readable 결과
- [x] non-interactive 모호성 발생 시 안전 경계 완화 금지
- [x] 원본 package manager interaction의 보안 경계 통제
- [x] approval·권한·감사·exit code·schema·retry 등 후속 결정 유보
