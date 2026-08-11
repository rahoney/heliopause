# User Journey + CLI IA

이 문서는 Heliopause Artifact Airlock의 사용자 흐름과 CLI 정보 구조 결정을 기록한다. 세부 command syntax, 인자, flag, alias, interactive prompt, retry 방식 및 output 형식은 후속 결정에서 확정한다.

## User-Journey-001: 주요 사용자 Journey

### 사용자 원문

기본 사용자 Journey는 외부 Software Artifact를 지정하여 통제된 검사 흐름을 수행하고, 필요한 경우 격리된 환경에서 동적 검사를 수행하며, 검증 결과와 Policy Decision을 확인한 뒤 필요한 경우에만 사용자 환경으로 반입하는 과정으로 구성한다.

npm, PyPI/pip, GitHub Releases 등 지원 Artifact source는 사용자 관점에서 가능한 한 동일한 기본 Journey를 공유한다. 생태계별 입력 해석, acquisition, dependency 처리 및 검사 방식의 차이는 Artifact Adapter와 관련 내부 모듈에서 처리하며, 사용자가 생태계별 내부 workflow를 직접 조작하도록 요구하지 않는다.

기본 Canonical Journey는 다음과 같다.

```text
Artifact 지정 → Artifact 식별·획득 → Controlled Intake → Verification → Static Inspection → 필요 시 Sandbox/Dynamic Inspection → Policy Decision → 결과 확인
```

최종 Policy Decision에 따라:

- `ALLOW` → 검증 완료 상태로 전환한다. 사용자가 처음 요청한 작업이 설치·반입을 포함하는 경우에는 Verified Set과 Staging/Promotion 절차를 거쳐 원래 요청한 사용자 환경 설치·반입을 계속 수행한다. 검사만 요청한 경우에는 사용자 환경으로 반입하지 않고 결과를 제공한 뒤 종료한다.
- `MANUAL_REVIEW` → 자동 반입하지 않고 Finding, Evidence, 검사 한계와 관련 결과를 사용자에게 제공한다.
- `BLOCK` → 사용자 환경으로의 반입을 허용하지 않는다.

Heliopause는 사용자가 처음 요청한 작업의 목적을 유지하면서 그 실행 전에 보안 검사 workflow를 자동으로 삽입한다. 사용자가 설치·반입을 요청한 경우에는 검사 결과가 `ALLOW`이면 별도의 추가 Promotion 명령을 요구하지 않고 Verified Set, Staging 및 trusted Promotion 절차를 거쳐 원래 요청한 설치·반입을 계속 수행한다.

반대로 사용자가 검사만 요청한 경우에는 `ALLOW`가 발생하더라도 사용자 환경으로 Artifact를 설치·반입하지 않고 검사 결과를 제공한 뒤 종료한다.

따라서 Heliopause에서 검사와 Promotion은 내부 Architecture상 별도의 책임으로 유지하지만, 사용자 Journey에서는 설치 요청 하나의 End-to-End 작업 안에서 연속적으로 수행될 수 있다.

Heliopause는 다음 세 가지 대표 Journey를 지원한다.

- **Install / Import Through Heliopause** — 사용자가 Artifact의 설치·반입을 요청하면 Heliopause가 먼저 전체 검사 workflow를 수행한다. 최종 Policy Decision이 `ALLOW`이면 Verified Set과 trusted Promotion 절차를 거쳐 원래 요청한 설치·반입을 완료하며, `MANUAL_REVIEW` 또는 `BLOCK`이면 설치·반입을 중단하고 관련 결과와 경고를 제공한다.
- **Inspect Only** — Artifact를 검사하고 Policy Decision과 Result/Evidence를 확인하지만 사용자 환경으로 설치·반입하지 않는다.
- **Review / Continue Existing Run** — 기존 Inspection Run의 Result, Finding, Evidence, SBOM, Policy Decision 등을 조회하고 필요한 후속 작업을 수행한다.

Package ecosystem은 일반적으로 `Verified Set → Staging → Promotion` 경로를 사용하며, package manager가 필요 없는 standalone binary/archive는 Architecture-008에 정의된 조건에 따라 digest 재검증 후 trusted Promotion 경계를 통해 사용자 지정 위치로 직접 반입할 수 있다.

세부 command syntax, 인자, flag, alias, interactive prompt, retry 방식 및 output 형식은 후속 결정에서 확정한다.

## User-Journey-002: Artifact 입력 방식

### 사용자 원문

Heliopause의 사용자는 하나의 Primary Software Artifact를 검사 대상으로 지정하며, 해당 Artifact가 필요로 하는 dependency와 관련 구성요소는 Heliopause가 지원 생태계의 Artifact Adapter를 통해 식별·획득한다. 사용자가 dependency를 개별적으로 직접 지정하거나 내부 acquisition workflow를 조작하도록 요구하지 않는다.

MVP에서는 npm, PyPI/pip, GitHub Releases를 지원하며 각 생태계의 자연스러운 Artifact 식별 정보를 입력으로 받을 수 있도록 한다. npm과 PyPI package는 package identity와 version을, GitHub Releases는 repository/release와 대상 asset을 식별할 수 있어야 한다. 정확한 CLI syntax와 flag 이름은 후속 CLI IA 결정에서 확정한다.

입력된 Artifact reference가 source, ecosystem, version, release 또는 asset을 명확하게 식별하지 못하는 경우 Heliopause는 보안상 의미 있는 값을 임의로 추측하여 검사를 진행하지 않는다. URL이나 기타 입력 자체에서 source가 명확하게 판별되는 경우에는 해당 Adapter를 자동 선택할 수 있으나, 복수의 해석이 가능한 입력은 사용자가 명확하게 지정하도록 한다.

사용 편의를 위해 version이나 release를 생략하거나 `latest`와 같은 가변 reference를 입력으로 허용할 수 있으나, 실제 Artifact acquisition과 Inspection Run을 시작하기 전에 이를 정확한 version/tag/asset 등으로 resolve해야 한다. 이후 검사·Evidence·Policy Decision·Verified Set·Promotion은 resolve된 exact identity와 digest를 기준으로 수행하며 가변 reference 자체를 검증 대상 identity로 사용하지 않는다.

GitHub Release에 복수의 asset이 존재하여 검사 대상이 하나로 결정되지 않는 경우에는 자동으로 임의 asset을 선택하지 않고 대상 asset을 명확하게 resolve한 뒤 검사를 시작한다.

MVP의 직접 입력 범위는 지원 대상 Artifact source인 npm, PyPI/pip 및 GitHub Releases로 제한한다. 임의 local file, generic direct URL 및 기타 Software Artifact source의 직접 입력은 MVP에 추가하지 않으며, 필요성이 생기면 향후 별도의 Artifact Adapter/source capability로 확장한다.

입력값을 내부 공통 Artifact identity로 정규화하는 구체적인 schema, parser, syntax 및 CLI option은 이후 Domain Model과 CLI IA 단계에서 정의한다.

```text
사용자 → Primary Artifact 지정
              ↓
       source/ecosystem 식별
              ↓
       exact version/asset resolve
              ↓
        Canonical Journey 시작
              ↓
     acquisition / Controlled Intake
              ↓
      exact identity + digest 확정
```

## 구조화된 결정

### User-Journey-001 Canonical Journey

모든 지원 Artifact source는 사용자 관점에서 가능한 한 공통 흐름을 공유한다. 생태계별 입력 해석, acquisition, dependency 처리 및 검사 차이는 Artifact Adapter와 관련 내부 모듈이 담당한다.

```text
Artifact 지정 → 식별·획득 → Controlled Intake → Verification
→ Static Inspection → 필요 시 Sandbox/Dynamic Inspection
→ Policy Decision → 결과 확인
```

| 최종 Decision | 설치·반입 요청(`install`) | 검사 전용 요청(`inspect`) |
| --- | --- | --- |
| `ALLOW` | Verified Set·Staging·trusted Promotion 후 원래 요청한 사용자 환경 설치·반입 계속 | 반입하지 않고 결과 제공 후 종료 |
| `MANUAL_REVIEW` | 설치·반입 중단, Finding·Evidence·검사 한계·관련 결과 제공 | 반입하지 않고 결과 제공 |
| `BLOCK` | 설치·반입 금지, 위험 실행 환경·임시 Artifact 정리 후 차단 근거 제공 | 반입하지 않고 결과 제공 |

`WARN`은 독립적인 최종 Decision이 아니라 Finding의 경고 수준이다. 검사와 실제 Promotion은 내부 Architecture상 분리하지만, 설치·반입을 처음 요청한 사용자 Journey에서는 `ALLOW` 후 별도 Promotion 명령 없이 연속 수행할 수 있다.

Package ecosystem은 일반적으로 `Verified Set → Staging → Promotion` 경로를 사용하고, package manager가 필요 없는 standalone binary/archive는 Architecture-008 조건에 따라 digest 재검증 후 trusted Promotion 경계를 통해 사용자 지정 위치로 직접 반입할 수 있다.

| Journey | 목적 | 반입 동작 |
| --- | --- | --- |
| `Install / Import Through Heliopause` | 설치·반입 요청을 검사 후 안전하게 중개 | `ALLOW` 후 Verified Set·Staging·trusted Promotion으로 원래 작업 계속 |
| `Inspect Only` | 검사 결과·Policy Decision·Result/Evidence 확인 | 사용자 환경으로 반입하지 않음 |
| `Review / Continue Existing Run` | 기존 Inspection Run의 Result, Finding, Evidence, SBOM, Policy Decision 조회 및 후속 작업 | 기존 상태와 후속 결정에 따라 별도 수행 |

### User-Journey-002 입력 원칙

- 사용자는 하나의 Primary Software Artifact를 지정한다.
- dependency와 관련 구성요소는 Artifact Adapter가 식별·획득한다.
- npm과 PyPI/pip는 package identity와 version, GitHub Releases는 repository/release와 대상 asset을 입력으로 식별한다.
- source·ecosystem·version·release·asset이 모호하면 임의 추측하지 않고 명확한 입력을 요구한다.
- source가 명확한 URL은 Adapter를 자동 선택할 수 있지만, 복수 해석 입력은 사용자 지정이 필요하다.
- `latest` 등 가변 reference는 acquisition과 Inspection Run 전에 exact version/tag/asset으로 resolve한다.
- 검사·Evidence·Policy Decision·Verified Set·Promotion은 resolve된 exact identity와 digest를 기준으로 한다.
- GitHub Release asset이 여러 개면 임의 선택하지 않고 대상 asset을 명확히 resolve한다.

MVP 직접 입력은 npm, PyPI/pip 및 GitHub Releases로 제한한다. 임의 local file, generic direct URL 및 기타 source는 제외하며 향후 별도 Artifact Adapter/source capability로 확장한다. 공통 Artifact identity schema, parser, syntax 및 CLI option은 Domain Model과 후속 CLI IA에서 확정한다.

## 구현 영향과 누락 점검

- [x] Primary Artifact 하나 지정 및 dependency 자동 식별·획득
- [x] npm package identity·version 입력
- [x] PyPI/pip package identity·version 입력
- [x] GitHub repository/release·asset 입력
- [x] 모호한 reference의 임의 추측 금지 및 사용자 명확화
- [x] source가 명확한 입력의 Adapter 자동 선택 허용
- [x] 가변 reference의 exact identity/digest resolve
- [x] 복수 GitHub asset의 임의 선택 금지
- [x] MVP 직접 입력 source 제한
- [x] local file·generic direct URL·기타 source의 MVP 제외
- [x] schema·parser·syntax·CLI option 후속 결정 유보

## User-Journey-003: CLI 최상위 Command 구조

### 사용자 원문

Heliopause의 MVP CLI는 내부 Architecture의 세부 단계를 직접 노출하기보다, 사용자가 기존 개발 도구에서 수행하려던 작업을 Heliopause를 통해 안전하게 실행할 수 있도록 구성한다.

기본 CLI 구조는 **Artifact ecosystem/source와 사용자 작업(operation)**을 중심으로 구성한다.

```text
helox <ecosystem/source> <operation> <artifact>
```

MVP에서는 npm, PyPI/pip, GitHub Releases를 지원하므로 사용자 관점의 기본 구조는 다음과 같다.

```text
helox
├─ npm
│   ├─ install
│   └─ inspect
│
├─ pip
│   ├─ install
│   └─ inspect
│
├─ github
│   ├─ install
│   └─ inspect
│
└─ review
```

정확한 GitHub Releases 입력 syntax, asset 지정 방식 및 각 Command의 flag는 후속 CLI IA 결정에서 구체화한다.

#### `install`

`install`은 사용자가 실제로 Artifact를 자신의 환경에 설치·반입하려는 기본 작업이다.

예를 들어:

```text
helox npm install kenv
helox pip install requests
```

사용자가 `install`을 실행하면 Heliopause는 실제 사용자 환경에 즉시 설치하지 않고 먼저 내부적으로 전체 보안 검사 workflow를 수행한다.

```text
install 요청
    ↓
Artifact 식별 / exact version resolve
    ↓
acquisition / Controlled Intake
    ↓
Verification
    ↓
Static Inspection
    ↓
필요 시 Sandbox / Dynamic Inspection
    ↓
Policy Decision
```

Policy Decision에 따라:

```text
ALLOW
  ↓
Verified Set / Manifest 확인
  ↓
digest 재검증
  ↓
Staging / trusted Promotion
  ↓
사용자 환경 설치·반입 완료

MANUAL_REVIEW
  ↓
설치 중단
  ↓
경고 및 검토 정보 제공

BLOCK
  ↓
설치 금지
  ↓
위험 실행 환경·임시 Artifact 정리
  ↓
경고 및 차단 근거 제공
```

따라서 `install`은 사용자가 요청한 원래 작업을 Heliopause가 안전하게 중개하는 End-to-End 명령이다.

#### `inspect`

`inspect`는 Artifact를 사용자 환경에 설치·반입하지 않고 검사만 수행하려는 경우 사용한다.

```text
helox npm inspect kenv
helox pip inspect requests
```

`inspect` 역시 Artifact resolution, acquisition, Verification, Static Inspection, 필요한 경우 Dynamic Inspection과 Policy Decision까지 수행하지만, 최종 결과가 `ALLOW`여도 사용자 환경으로 Promotion하거나 설치하지 않는다.

즉:

```text
install = 검사 → ALLOW 시 원래 설치 작업 계속
inspect = 검사 → 결과 확인 후 종료
```

#### `review`

`review`는 ecosystem과 관계없이 기존 Inspection Run의 결과를 다시 조회하는 공통 명령으로 둔다.

```text
helox review <inspection-run>
```

Policy Decision, Verification Result, Finding, Evidence, Execution Status, 검사 한계, SBOM, Verified Manifest 등의 세부 결과는 `review`의 하위 조회 기능 또는 option으로 제공할 수 있으며 각각을 MVP 최상위 Command로 분리하지 않는다.

#### Promotion과 내부 단계

`Promotion`은 Architecture의 독립된 책임으로 유지하지만 일반 사용자가 매번 직접 호출해야 하는 핵심 CLI Command로 노출하지 않는다.

`install` workflow에서 최종 Policy Decision이 `ALLOW`인 경우 Application/Workflow가 trusted Promotion 경계를 호출하여 검증 완료 Artifact를 실제 사용자 환경으로 반입한다.

따라서 MVP의 일반 사용자용 최상위 Command로 다음 내부 단계를 직접 노출하지 않는다.

```text
acquire
verify
scan
sandbox
policy
stage
promote
```

이들은 Heliopause 내부 Architecture와 workflow를 구성하는 책임이며, 일반 사용자는 해당 단계를 이해하거나 순서대로 직접 실행할 필요가 없다.

사용자 관점의 핵심 모델은 다음과 같다.

```text
기존 작업
npm install kenv

        ↓ Heliopause를 통해 실행

helox npm install kenv

        ↓

Heliopause가 내부 보안 검사 자동 수행

        ↓

안전함 → 설치 계속
문제 있음 → 설치 중단 + 경고
```

`help`, `version` 등 일반 CLI 기능과 향후 필요할 수 있는 `doctor`, `config` 등의 운영 보조 Command는 핵심 Journey와 분리하여 후속 Tooling/CLI 설계에서 필요성을 결정한다.

각 ecosystem/source Command의 정확한 인자, flag, alias, GitHub Releases asset 지정 방식, interactive 동작과 출력 형식은 후속 User Journey / CLI IA 결정에서 구체화한다.

### 구조화된 결정

MVP의 사용자-facing 기본 구조는 `helox <ecosystem/source> <operation> <artifact>`이며, 핵심 operation은 생태계별 `install`·`inspect`와 공통 `review`다.

```text
helox
├─ npm: install, inspect
├─ pip: install, inspect
├─ github: install, inspect
└─ review
```

| Command | 사용자 작업 | 검사·반입 동작 |
| --- | --- | --- |
| `<ecosystem> install <artifact>` | 기존 개발 도구의 설치 작업을 안전하게 중개 | 검사 workflow를 먼저 수행하고 `ALLOW`일 때만 Verified Set/Manifest 확인·digest 재검증·Staging/trusted Promotion 후 사용자 환경에 반입 |
| `<ecosystem> inspect <artifact>` | 설치·반입 없이 검사 | resolution·acquisition·Verification·Static/Dynamic Inspection·Policy Decision 수행 후 결과 확인으로 종료 |
| `review <inspection-run>` | 기존 Inspection Run 결과 재조회 | Policy Decision, Verification Result, Finding, Evidence, Execution Status, 검사 한계, SBOM, Verified Manifest 조회 |

`install`은 실제 사용자 환경에 즉시 설치하지 않고 전체 검사 workflow를 선행하는 End-to-End 명령이다. `ALLOW`면 원래 설치 작업을 계속하고, `MANUAL_REVIEW`면 설치를 중단한 뒤 경고·검토 정보를 제공하며, `BLOCK`이면 설치를 금지하고 위험 실행 환경·임시 Artifact를 정리한 뒤 차단 근거를 제공한다.

`inspect`는 `ALLOW`여도 Promotion/설치를 수행하지 않는다. `review`는 ecosystem과 무관한 공통 조회 명령이며 결과 종류별 최상위 Command를 별도로 만들지 않는다. Promotion은 독립 Architecture 책임으로 유지하되 `install` 내부에서 Application/Workflow가 trusted Promotion 경계를 호출한다.

`acquire`, `verify`, `scan`, `sandbox`, `policy`, `stage`, `promote`는 내부 workflow 단계로 일반 사용자용 최상위 Command에 직접 노출하지 않는다. `help`, `version`, `doctor`, `config`는 핵심 Journey와 분리하여 후속 Tooling/CLI 설계에서 결정한다.

### 후속 결정으로 유보한 항목

- 정확한 GitHub Releases 입력 syntax와 asset 지정 방식
- ecosystem/source별 인자, flag, alias
- interactive 동작과 confirmation
- terminal·구조화 output 형식
- `help`, `version`, `doctor`, `config` 세부 동작

## User-Journey-003 구현 영향과 누락 점검

- [x] 내부 Architecture 단계가 아닌 사용자 작업 중심 CLI 구조
- [x] `helox <ecosystem/source> <operation> <artifact>` 기본 형태
- [x] npm·pip·github의 `install`·`inspect` 구조
- [x] ecosystem과 무관한 공통 `review` 명령
- [x] `install`의 검사 선행 및 ALLOW 후 원래 설치 작업 계속
- [x] `MANUAL_REVIEW` 설치 중단·검토 정보 제공
- [x] `BLOCK` 설치 금지·위험 실행 환경·임시 Artifact 정리·차단 근거 제공
- [x] `inspect`의 설치·반입 없는 검사 후 종료
- [x] `promote`를 일반 사용자 핵심 Command로 직접 노출하지 않음
- [x] 내부 workflow 단계의 일반 사용자용 최상위 Command 비노출
- [x] `help`·`version`·`doctor`·`config` 후속 결정 유보
- [x] ecosystem/source별 syntax·flag·alias·prompt·output 후속 결정 유보

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

## User-Journey-006: 결과 표시와 상세 조회 Journey

### 사용자 원문

Heliopause의 기본 terminal 출력은 내부 검사 과정을 모두 노출하기보다 사용자가 현재 작업의 결과와 다음 행동을 빠르게 이해할 수 있도록 요약 중심으로 제공한다.

상세한 Verification Result, Finding, Evidence, Execution Status, SBOM, Verified Manifest 및 기타 검사 기록은 Inspection Run에 연결하여 보존하고, 사용자가 필요할 때 `review`를 통해 단계적으로 상세 조회할 수 있도록 한다.

기본 원칙은 다음과 같다.

```text
일반 실행
→ 핵심 진행 상태 + 최종 결과

문제·불확실성 발생
→ 핵심 원인 + 다음 행동

상세 분석 필요
→ review <inspection-run>
```

### 기본 terminal 결과

`install`과 `inspect` 실행의 기본 출력에는 최소한 다음 정보가 포함되어야 한다.

- 검사 대상 Artifact의 resolved identity와 version
- 주요 검사 단계의 진행 또는 완료 상태
- 최종 Policy Decision
- 사용자 요청 작업의 최종 상태
- 중요한 경고 또는 주요 Finding
- Inspection Run 식별 정보

일반적인 `ALLOW` 결과에서는 정상적인 세부 Evidence와 모든 Finding을 전부 출력하지 않고, 작업 성공 여부를 빠르게 확인할 수 있는 수준으로 요약한다.

```text
$ helox npm install kenv

Inspecting kenv@<resolved-version>...
✓ Verification
✓ Static Inspection
- Dynamic Inspection: Not required
✓ Policy: ALLOW
✓ Installation completed

Run: <inspection-run>
```

실제 progress UI, symbol, 색상 및 문구는 후속 CLI presentation 설계에서 결정한다.

### `ALLOW` 결과 표시

`install` 요청이 `ALLOW`로 완료된 경우 사용자가 최소한 다음을 확인할 수 있어야 한다.

```text
Artifact
→ 정확히 무엇이 설치되었는가

Policy
→ ALLOW

Operation
→ 설치·반입이 실제 완료되었는가

Inspection Run
→ 필요할 때 어떤 검사 기록을 조회하면 되는가
```

중요한 경고 수준의 Finding이 존재하지만 Policy가 `ALLOW`인 경우에는 성공 결과와 함께 해당 경고의 존재를 표시한다.

정상적인 세부 Evidence, dependency 전체 목록, scanner 원본 출력 등을 기본 terminal 출력에 모두 노출하지 않는다.

### `MANUAL_REVIEW` 결과 표시

`MANUAL_REVIEW`에서는 단순히 상태 이름만 출력하지 않고 사용자가 왜 자동 작업이 보류되었는지 이해할 수 있는 핵심 정보를 함께 제공한다.

최소한 다음을 표시한다.

- `MANUAL_REVIEW` 상태
- install/반입이 보류되었다는 사실
- 자동 판단을 완료하지 못한 주요 이유
- 중요한 Finding 또는 검사 실패·미완료 상태
- 필요한 경우 검사 한계
- Inspection Run 식별자
- 상세 결과를 확인할 수 있는 `review` 경로

```text
Policy: MANUAL_REVIEW
Installation paused.

Reason:
- Required dynamic inspection could not be completed.

Run: <inspection-run>
Review: helox review <inspection-run>
```

기본 화면에서 모든 Evidence를 출력하지 않고 사용자가 판단을 시작하는 데 필요한 핵심 정보만 제공한다.

### `BLOCK` 결과 표시

`BLOCK`에서는 설치·반입이 차단되었다는 사실과 핵심 차단 이유를 명확하게 표시한다.

최소한 다음을 포함한다.

- `BLOCK` 상태
- 원래 요청한 install/반입이 수행되지 않았다는 사실
- 주요 차단 Finding
- 필요한 경우 위험 행동 또는 검증 실패 요약
- Inspection Run 식별자
- 상세 조회 방법

```text
Policy: BLOCK
Installation blocked.

Key findings:
- Credential access attempt detected
- Unexpected outbound connection attempted

Run: <inspection-run>
Review: helox review <inspection-run>
```

차단된 위험 Artifact나 Sandbox 실행 환경을 정리하더라도 판단 근거가 되는 Evidence와 Inspection Run 기록은 별도 retention 정책에 따라 보존한다.

### `inspect` 결과 표시

`inspect`는 설치를 수행하지 않으므로 최종 결과에서 검사 완료와 Policy Decision을 명확하게 구분한다.

```text
Policy: ALLOW
Inspection completed.
No artifact was installed.

Run: <inspection-run>
```

사용자가 `ALLOW`를 `설치 완료`로 오해하지 않도록 최초 operation의 결과를 함께 표시한다.

### `review` 상세 조회

`review`는 기존 Inspection Run의 결과를 자세히 확인하기 위한 공통 조회 진입점으로 사용한다.

```text
helox review <inspection-run>
```

기본 `review` 화면은 해당 Inspection Run의 종합 summary를 제공한다.

해당 Inspection Run에서 생성되었거나 적용 가능한 경우 다음 항목을 추적·조회할 수 있어야 하며, 생성되지 않았거나 적용되지 않은 항목은 그 상태와 사유를 확인할 수 있어야 한다.

- Artifact identity, version, source, digest
- Primary Artifact와 관련 dependency
- Policy Decision
- Verification Result
- Finding
- Evidence
- Capability
- Execution Status
- 수행·미수행 검사 및 사유
- 검사 한계
- SBOM
- Verified Set / Verified Manifest
- Sandbox 또는 검사 환경 정보
- 최종 install/import 상태

예를 들어 acquisition 실패 또는 `BLOCK`으로 Verified Manifest가 생성되지 않은 경우 `Verified Manifest: Not generated — Policy: BLOCK`과 같이 그 상태와 사유를 표시한다.

세부 정보는 한 화면에 모두 펼치기보다 summary에서 시작하여 필요한 영역을 선택적으로 자세히 조회할 수 있도록 한다.

```text
review <run>
      ↓
Summary
      ├─ Verification
      ├─ Findings
      ├─ Evidence
      ├─ Dependencies / SBOM
      ├─ Execution Status
      └─ Verified Manifest
```

이를 실제 subcommand, flag 또는 다른 navigation 방식 중 무엇으로 구현할지는 후속 CLI IA에서 결정한다.

### Raw Evidence와 외부 도구 출력

일반 사용자가 `review`를 실행했다고 해서 scanner, verifier 또는 Sandbox의 대용량 Raw Evidence를 전부 즉시 출력하지 않는다.

정규화된 결과와 주요 Evidence를 우선 제공하고, 원본 자료가 필요한 경우 별도의 상세 조회 또는 machine-readable 결과를 통해 접근할 수 있도록 한다.

```text
기본 terminal
→ 사용자 작업 결과

review summary
→ 판단에 필요한 정규화 결과

detailed Evidence
→ 기술적 조사

Raw Evidence
→ 심층 분석·감사·재현
```

### Inspection Run 식별자

`install`, `inspect` 및 검사 결과를 생성하는 주요 작업은 사용자가 이후 해당 결과를 다시 찾을 수 있도록 Inspection Run 식별자를 제공한다.

Inspection Run ID의 구체적인 형식은 Domain Model에서 결정한다.

사용자는 Artifact 이름이나 terminal 출력에 의존하지 않고 Inspection Run을 기준으로 동일한 검사 기록을 다시 조회할 수 있어야 한다.

### 사람용 결과와 machine-readable 결과

사람이 terminal에서 읽는 결과와 CI·automation에서 사용하는 구조화 결과는 동일한 Inspection Run, Artifact identity/digest 및 Policy Decision을 참조해야 한다.

두 결과가 서로 다른 판정이나 Artifact를 나타내어서는 안 된다.

구체적인 JSON schema, output option, file format 및 machine-readable interface는 후속 Domain Model / CLI 설계에서 결정한다.

### 기본 출력 최소화 원칙

Heliopause는 정상적인 실행에서 다음과 같은 정보를 모두 기본 화면에 출력하지 않는다.

- 전체 dependency tree
- 모든 Raw Evidence
- scanner/verifier 원본 출력
- 전체 SBOM
- 모든 정상 검사 항목
- 내부 Sandbox telemetry
- 내부 Policy 평가 과정 전체

필요한 정보는 Inspection Run에 연결하여 보존하고 `review` 또는 구조화 출력에서 조회할 수 있도록 한다.

따라서 기본 UX는 다음을 지향한다.

```text
정상일수록 짧게
문제가 있을수록 핵심 이유를 명확하게
상세 정보는 필요할 때 깊게
```

### 후속 결정으로 유보한 항목

- terminal 색상·icon·progress UI
- 한 화면에 표시할 Finding 개수
- severity별 표시 방식
- `review`의 정확한 subcommand와 flag
- Raw Evidence 조회 syntax
- JSON 등 machine-readable output schema
- SBOM 출력 format
- Inspection Run ID 형식
- 결과 저장 위치
- pagination / filtering / sorting
- terminal verbosity option
- CI 출력 방식

위 항목은 후속 CLI IA, Domain Model, Evidence/Result 및 Tooling 설계에서 확정한다.

### 압축된 사용자 모델

```text
helox npm install kenv
        ↓
짧은 진행 상태
        ↓
최종 Policy + 실제 작업 결과
        ↓
Inspection Run ID

문제가 있거나 더 자세히 보고 싶음
        ↓
helox review <run-id>
        ↓
Summary
        ↓
필요한 Finding / Evidence / Verification / SBOM 등 상세 조회
```

Heliopause는 내부적으로 많은 보안 정보를 생성하더라도 일반 사용자의 기본 workflow를 복잡하게 만들지 않고, 필요한 만큼만 단계적으로 정보를 노출하는 것을 기본 원칙으로 한다.

### 구조화된 결정과 구현 영향

- 기본 terminal은 내부 전체 pipeline이 아니라 resolved Artifact identity/version, 핵심 진행 상태, 최종 Policy, 실제 작업 상태, 중요 경고/Finding, Inspection Run ID를 요약한다.
- `ALLOW` install 결과는 무엇이 설치되었는지, `ALLOW`인지, 실제 설치·반입이 완료되었는지, 어떤 Run을 review하는지 표시한다.
- `MANUAL_REVIEW`는 보류 사실, 자동 판단 불가 이유, 주요 Finding·실패·미완료·검사 한계, Run ID 및 review 경로를 표시한다.
- `BLOCK`은 반입 미수행, 핵심 차단 Finding·위험 행동·검증 실패, Run ID 및 상세 조회 방법을 표시한다.
- `inspect` 결과는 `ALLOW`여도 설치되지 않았음을 명시하여 검사 완료와 설치 완료를 혼동하지 않게 한다.
- `review <inspection-run>`은 Summary를 시작점으로 Verification, Findings, Evidence, dependency/SBOM, Execution Status, Verified Manifest를 단계적으로 조회한다.
- 대용량 Raw Evidence와 외부 도구 원본 출력은 기본 terminal이나 review summary에 모두 펼치지 않고 상세 조회·machine-readable 결과로 제공한다.
- `install`·`inspect`·주요 검사 작업은 Inspection Run ID를 제공하고 모든 사람용·기계용 결과가 동일 Run, identity/digest, Policy Decision을 참조하게 한다.
- 전체 dependency tree·Raw Evidence·원본 scanner 출력·SBOM·Sandbox telemetry·Policy 내부 평가 과정은 기본 출력에서 제외하고 Run에 보존한다.

## User-Journey-006 누락 점검

- [x] 기본 terminal의 요약 중심 출력
- [x] 핵심 진행 상태·최종 결과·다음 행동 제공
- [x] resolved Artifact identity/version 표시
- [x] `ALLOW` 결과의 실제 설치·반입 완료 상태 표시
- [x] `MANUAL_REVIEW` 보류 이유·Finding·한계·review 경로 표시
- [x] `BLOCK` 차단 사실·핵심 Finding·상세 조회 방법 표시
- [x] `inspect` 설치 없음 명시
- [x] `review <inspection-run>` 공통 상세 조회 진입점
- [x] Verification·Finding·Evidence·Capability·Execution Status 조회
- [x] 수행·미수행 검사와 사유·검사 한계 조회
- [x] SBOM·Verified Set·Verified Manifest·Sandbox 정보·최종 상태 조회
- [x] Raw Evidence 및 외부 도구 원본 출력의 단계적 접근
- [x] Inspection Run ID 제공 및 재조회 기준
- [x] 사람용·machine-readable 결과의 동일 Run·identity/digest·Policy 참조
- [x] 기본 출력 최소화 및 상세 정보 보존
- [x] presentation·schema·format·pagination·CI 출력 후속 결정 유보

## User-Journey-007: 실패·재검사·재개 Journey

### 사용자 원문

Heliopause는 검사 workflow의 실패·중단·미완료 상태를 정상적인 검사 완료와 명확하게 구분하며, 기존 검사 결과를 임의로 성공 상태로 간주하거나 안전하지 않은 실행 상태를 그대로 재사용하지 않는다.

기본 원칙은 다음과 같다.

```text
일시적·재시도 가능한 실패
→ 제한된 범위에서 자동 retry 가능

보안상 필요한 검사가 미완료
→ 안전한 것으로 간주하지 않음
→ MANUAL_REVIEW 또는 BLOCK

완료된 Inspection Run 이후 재검사
→ 기존 기록을 덮어쓰지 않음
→ 새로운 Inspection Run으로 수행

비정상 종료된 Sandbox Session
→ 그대로 resume/reuse하지 않음
→ 폐기 후 새로운 Session에서 실행
```

### 실행 중 일시적 실패와 자동 retry

검사 workflow 도중 외부 도구 오류, 일시적인 프로세스 실패 또는 기타 재시도 가능한 문제가 발생할 수 있다.

Heliopause는 해당 문제가 보안 경계를 약화시키지 않고 동일한 Artifact와 동일한 검사 조건에서 안전하게 다시 수행할 수 있는 경우 제한된 횟수의 자동 retry를 수행할 수 있다.

```text
검사 단계 실행
     ↓
일시적 실패
     ↓
안전하게 retry 가능한가?
     ├─ Yes → 제한된 자동 retry
     └─ No  → 실패·미완료 상태 기록
```

retry를 수행하더라도 최초 실패 사실을 숨기지 않으며, 필요한 경우 각 실행 attempt와 최종 Execution Status를 Inspection Run에 기록한다.

구체적인 retry 횟수, backoff, retry 가능한 오류 종류 및 외부 도구별 정책은 후속 Tooling/Inspection 설계에서 정의한다.

### 보안상 필수 검사 실패

Policy가 안전한 자동 판정을 위해 필요로 하는 Verification 또는 Inspection이 실패·미완료·사용 불가 상태가 된 경우 Heliopause는 해당 Artifact를 임의로 안전하다고 간주하지 않는다.

```text
필수 검사
   ↓
실패 / 미완료 / Unavailable
   ↓
ALLOW로 간주하지 않음
   ↓
fail-closed
   ↓
MANUAL_REVIEW 또는 BLOCK
```

어떤 상태가 `MANUAL_REVIEW`이고 어떤 상태가 `BLOCK`인지에 대한 구체적인 rule은 후속 Policy 설계에서 결정한다.

### Sandbox 실패와 재실행

Sandbox/Dynamic Inspection 도중 timeout, resource limit 초과, crash, 강제 종료 또는 기타 비정상 상태가 발생한 경우 해당 Sandbox Session을 이어서 사용하거나 다음 검사에 재사용하지 않는다.

```text
Sandbox Session
      ↓
비정상 종료 / timeout / resource limit / failure
      ↓
관찰 가능한 Evidence 수집
      ↓
Session 종료·폐기
      ↓
필요한 경우 새로운 Sandbox Session 생성
      ↓
재검사
```

새로운 Sandbox Session은 이전 비정상 실행 상태를 신뢰하거나 그대로 이어받지 않는다. 이는 오염되었거나 불완전한 격리 환경을 재사용하여 후속 검사 결과에 영향을 주는 것을 방지하기 위함이다.

### 실행 자체가 중단된 경우

사용자 interrupt, Heliopause process 종료, 시스템 오류 또는 기타 이유로 전체 workflow가 중단될 수 있다.

가능한 범위에서 이미 생성된 Inspection Run 정보, 수행 완료 단계, Execution Status 및 확보된 Evidence는 보존한다.

그러나 단순히 이전 실행이 어느 단계까지 진행되었다는 이유만으로 모든 내부 실행 상태를 그대로 복원하여 계속하지는 않는다.

재개 시에는 기존 결과의 무결성과 재사용 가능성을 확인하고, 안전하게 재사용할 수 없는 단계는 다시 수행한다. 특히 실행 중이던 Sandbox Session과 같은 ephemeral execution state는 resume 대상으로 간주하지 않는다.

```text
workflow 중단
      ↓
기존 Run / Evidence 보존
      ↓
재사용 가능한 결과 확인
      ↓
      ├─ 안전하게 재사용 가능 → 활용 가능
      └─ 재사용 불확실         → 해당 검사 다시 수행
```

어떤 검사 결과를 안전하게 재사용할 수 있는지에 대한 구체적인 validity 조건과 checkpoint 방식은 후속 Domain Model, Evidence/Result 및 Inspection 설계에서 정의한다.

### 완료된 Inspection Run의 불변성

최종 상태가 기록된 Inspection Run은 이후 재검사 결과로 덮어쓰지 않는다.

예를 들어 최초 검사에서:

```text
Run A
→ MANUAL_REVIEW
```

가 발생하고 이후 문제를 해결하거나 추가 검사를 수행하더라도 기존 Run A의 결과를 사후에 `ALLOW`로 수정하여 과거 상태를 없애지 않는다.

재검사가 필요한 경우 새로운 Inspection Run 또는 기존 Run과 명시적으로 연결된 후속 Run을 생성한다.

```text
Run A
Policy: MANUAL_REVIEW
        ↓
추가 검사 / 재검사
        ↓
Run B
Policy: ALLOW
```

Run B는 필요한 경우 Run A와의 관계를 기록할 수 있지만 두 Run의 Evidence, Execution Status 및 Policy Decision은 서로 구분한다. Inspection Run 간 관계를 표현하는 정확한 Domain Model은 후속 단계에서 정의한다.

### 동일 Artifact의 재검사

사용자는 이전에 검사한 Artifact를 다시 검사할 수 있다.

이 경우 Heliopause는 단순히 Artifact 이름이 같다는 이유로 과거 Policy Decision을 그대로 적용하지 않는다.

재검사에서 기존 결과를 활용하려면 최소한 실제 검사 대상의 exact identity와 digest가 일치해야 하며, 필요한 경우 검사 도구, Policy, Capability, 검사 환경 또는 기존 Evidence의 유효성도 고려해야 한다.

```text
이전 Artifact
name: kenv
version: X
digest: AAA

새 검사 대상
name: kenv
version: X
digest: AAA
```

동일성이 확인되더라도 과거 결과의 재사용 조건은 별도로 검증되어야 한다. 구체적인 cache/reuse 정책은 후속 설계에서 결정하며, 과거 `ALLOW`가 새로운 실행의 안전성을 자동 보장하지 않는다는 원칙만 고정한다.

### Artifact가 변경된 경우

다음과 같이 검사 대상의 실제 identity가 변경되면 기존 Inspection Run의 Policy Decision을 새 대상에 그대로 적용하지 않는다.

- version 변경
- release/tag 변경
- GitHub Release asset 변경
- digest 변경
- dependency 또는 Verified Set 변경
- 동일 version이라도 실제 Artifact bytes가 변경된 경우

이 경우 새로운 Artifact 상태로 취급하여 필요한 verification/inspection을 수행한다.

특히 mutable reference인 `latest` 등을 다시 사용했을 때 이전과 다른 exact version 또는 digest로 resolve되었다면 이전 Inspection Run을 현재 Artifact의 안전 판정으로 사용하지 않는다.

```text
latest
  ↓
어제 → 1.2.0 / digest AAA
오늘 → 1.3.0 / digest BBB

→ 서로 다른 검사 대상
→ 새로운 Inspection Run 필요
```

### `MANUAL_REVIEW` 이후 재검사

`MANUAL_REVIEW` 상태에서는 기존 설치 요청을 자동으로 계속하지 않는다.

사용자는 `review`를 통해 판단 근거를 확인하고, 필요한 경우 문제의 원인이 해결된 상태에서 추가 검사 또는 재검사를 수행할 수 있다.

```text
MANUAL_REVIEW
      ↓
review
      ↓
원인 확인
      ↓
추가 정보 / 환경 변화 / 재검사 필요
      ↓
새로운 검사 수행
      ↓
새 Policy Decision
```

새로운 검사 결과가 `ALLOW`가 되었다면 해당 `ALLOW`와 연결된 검증 결과, Artifact identity/digest 및 Verified Set을 기준으로 후속 설치·반입 여부를 결정한다.

기존 `MANUAL_REVIEW` Run 자체를 단순히 사용자 confirmation만으로 `ALLOW`로 변경하지 않는다. 향후 별도의 approval/override 기능을 도입할 경우에는 이 재검사 경로와 구분하여 명시적인 권한·감사 기록·Policy 의미를 갖도록 설계한다.

### `BLOCK` 이후 재검사

`BLOCK`은 해당 Inspection Run과 해당 Artifact 상태에 대한 설치·반입 금지 Decision이다.

사용자는 일반 confirmation으로 해당 Run의 `BLOCK`을 우회할 수 없다.

다만 이후 검사 대상 자체가 달라진 경우에는 새로운 검사 대상으로 다시 검사할 수 있다.

```text
package@1.2.3
→ BLOCK

새 version 1.2.4 공개
        ↓
새 Artifact identity / digest
        ↓
새로운 Inspection Run
```

동일 Artifact에 대한 재검사를 허용할지 여부와 그 조건은 후속 Policy에서 구체화할 수 있으나, 새로운 재검사 결과가 기존 `BLOCK` 기록을 삭제하거나 수정하지 않는다.

### 재검사와 Promotion의 연결

Promotion은 항상 현재 반입하려는 Artifact와 연결된 유효한 `ALLOW` Policy Decision을 요구한다.

과거의 다른 Inspection Run에서 생성된 `ALLOW`, 다른 digest의 Verified Set 또는 현재 Artifact와 연결되지 않은 Verified Manifest를 조합하여 Promotion하지 않는다.

```text
현재 Promotion 대상
        ↓
정확한 Artifact identity / digest
        ↓
해당 검증 결과
        ↓
해당 ALLOW Decision
        ↓
Verified Set / Manifest
        ↓
서로 일관되게 연결됨
        ↓
Promotion 가능
```

재검사 결과 새로운 `ALLOW`가 생성된 경우 최종 설치·반입은 새로운 검사 결과를 기준으로 수행한다.

### 사용자 관점의 재시도·재검사

일반 사용자가 내부 workflow 단계별로 직접 `verify`, `scan`, `sandbox` 등을 재실행하도록 요구하지 않는다.

Heliopause는 가능한 retry를 내부적으로 처리하고, 사용자 개입이 필요한 경우에는 작업 상태와 이유를 명확하게 제공한다.

```text
정상적 일시 오류
→ Heliopause가 가능한 범위에서 자동 retry

해결되지 않음
→ 작업 중단 / MANUAL_REVIEW 또는 BLOCK
→ Run ID 제공

사용자가 원인 확인
→ 필요하면 Artifact를 다시 검사
→ 새로운 결과 확인
```

재검사를 위한 정확한 CLI command, 기존 Run을 참조하는 syntax, `retry` 또는 `resume` command를 별도로 제공할지 여부는 후속 CLI IA에서 결정한다.

### 후속 결정으로 유보한 항목

- 자동 retry 횟수와 backoff
- retry 가능한 오류 분류
- 별도 `retry` / `resume` command 제공 여부
- 재검사 CLI syntax
- checkpoint 저장 방식
- 완료된 검사 결과의 cache/reuse 조건
- 이전 Evidence의 validity 기간
- tool/version/Policy 변경 시 재검사 조건
- Inspection Run 간 parent/retry/supersede 관계 schema
- `MANUAL_REVIEW` approval/override
- 동일한 `BLOCK` Artifact의 재검사 제한
- 재검사 후 원래 install 요청을 자동 재개할지 여부

위 항목은 후속 CLI IA, Domain Model, Policy, Evidence/Result 및 Tooling 단계에서 구체화한다.

### 압축된 사용자 모델

```text
검사 시작
   ↓
일시적인 오류
   ├─ 안전하게 retry 가능 → 제한된 자동 retry
   └─ 해결 불가 → 실패 상태 기록

보안상 필수 검사 미완료
   ↓
ALLOW 금지
   ↓
MANUAL_REVIEW / BLOCK

기존 Run 이후 다시 검사
   ↓
기존 기록 유지
   ↓
새 Inspection Run
   ↓
새 Policy Decision

Artifact identity / digest 변경
   ↓
기존 판정 재사용 금지
   ↓
새 검사 필요
```

Heliopause는 실패 상태를 숨기거나 과거 검사 결과를 편의상 무조건 재사용하지 않고, 재현 가능성과 감사 가능성을 유지하면서 안전하게 재시도·재검사할 수 있도록 하는 것을 기본 원칙으로 한다.

### 구조화된 결정과 구현 영향

- 일시적이고 보안 경계를 약화시키지 않는 실패만 제한된 자동 retry 대상으로 삼고 최초 실패·attempt·최종 Execution Status를 기록한다.
- 보안상 필수 Verification/Inspection의 실패·미완료·Unavailable은 `ALLOW`로 해석하지 않고 fail-closed로 `MANUAL_REVIEW` 또는 `BLOCK`에 연결한다.
- 비정상 Sandbox Session은 Evidence를 수집한 뒤 폐기하고 새로운 Session에서 재실행한다.
- 중단된 workflow의 Run·완료 단계·Execution Status·Evidence는 보존하되 ephemeral execution state를 그대로 resume하지 않는다.
- 완료된 Run은 불변으로 유지하며 후속 검사에는 새로운 Run을 만들고 Run 간 관계만 명시적으로 기록한다.
- 과거 결과 재사용은 exact identity·digest뿐 아니라 tool·Policy·Capability·환경·Evidence 유효성을 고려하며 과거 `ALLOW`를 자동 안전 보장으로 사용하지 않는다.
- version·release/tag·asset·digest·dependency·Verified Set·Artifact bytes가 변경되면 새 검사 대상으로 취급한다.
- `MANUAL_REVIEW`는 review와 새 검사로 이어지며 기존 Run을 confirmation만으로 `ALLOW`로 변경하지 않는다.
- `BLOCK`은 일반 confirmation으로 우회하지 않는다. Artifact가 변경된 경우에는 새로운 Run으로 검사하며, 동일 Artifact의 재검사 허용 여부와 조건은 후속 Policy에서 결정한다.
- Promotion은 현재 대상과 동일한 identity/digest에 연결된 유효한 `ALLOW`·Verified Set·Manifest만 사용한다.
- 일반 사용자는 내부 단계 재실행을 직접 조작하지 않고 Heliopause가 retry와 재검사 필요 상태를 안내한다.

## User-Journey-007 누락 점검

- [x] 일시적 실패의 제한된 자동 retry
- [x] 최초 실패·attempt·최종 Execution Status 기록
- [x] 필수 검사 실패·미완료·Unavailable의 fail-closed 처리
- [x] 비정상 Sandbox Session 관찰·폐기·새 Session 재실행
- [x] 중단된 Run·Evidence 보존과 ephemeral state 재사용 금지
- [x] 완료된 Inspection Run 불변성
- [x] 후속 검사에서 새 Inspection Run 생성
- [x] 동일 Artifact 재검사 시 identity/digest 및 유효성 확인
- [x] version·release/tag·asset·digest·dependency·bytes 변경 시 판정 재사용 금지
- [x] latest 등 mutable reference 변경 시 새 Run
- [x] MANUAL_REVIEW의 review·재검사 경로
- [x] 기존 MANUAL_REVIEW Run의 confirmation ALLOW 변경 금지
- [x] BLOCK의 일반 우회 금지와 새 Artifact 재검사
- [x] Promotion의 현재 Artifact·ALLOW·Verified Set·Manifest 일관성
- [x] 사용자에게 내부 retry·재검사 단계를 직접 조작시키지 않음
- [x] retry·resume·cache·checkpoint·Run 관계·approval 후속 결정 유보

## User-Journey-008: 설치·반입(Promotion) UX

### 사용자 원문

Heliopause에서 Promotion은 검사를 통과한 Artifact를 실제 사용자 환경으로 반입하기 위한 신뢰 경계이며, 일반 사용자가 별도의 내부 단계로 직접 조작하는 작업으로 노출하지 않는다.

사용자가 처음부터 `install` 또는 이에 해당하는 설치·반입 작업을 요청한 경우 최종 Policy Decision이 `ALLOW`이면 Heliopause가 원래 요청의 실행 context를 유지한 상태에서 안전한 Promotion 절차를 수행하고 원래 작업을 완료한다.

기본 원칙은 다음과 같다.

```text
사용자 install 요청
        ↓
검사 workflow
        ↓
Policy = ALLOW
        ↓
검사한 Artifact / Verified Set 확인
        ↓
identity / digest 재검증
        ↓
trusted Promotion
        ↓
원래 요청한 사용자 환경으로 설치·반입
```

`MANUAL_REVIEW` 또는 `BLOCK` 상태에서는 이 Promotion 경계로 진행하지 않는다.

### 원래 install 요청의 유지

Heliopause는 검사 이전에 사용자가 요청한 설치 작업의 의미를 유지한다.

예를 들어 다음과 같은 정보가 해당될 수 있다.

- ecosystem/source
- operation
- Primary Artifact
- resolved version/release/asset
- 프로젝트 또는 설치 대상 환경
- local/global 등 설치 범위
- 사용자가 명시한 허용 가능한 option
- standalone Artifact의 대상 위치

따라서 사용자가:

```text
helox npm install kenv
```

를 요청했다면 `ALLOW` 이후 임의의 다른 위치나 다른 방식으로 Artifact를 반입하는 것이 아니라 원래 요청과 연결된 설치 작업을 계속 수행한다.

원본 package manager option의 지원 범위와 각 option이 검사 결과 및 Promotion에 미치는 영향은 후속 ecosystem별 CLI contract에서 정의한다.

### 검사 대상과 실제 반입 대상의 동일성

Promotion에서 가장 중요한 원칙은 실제로 사용자 환경으로 반입되는 Artifact가 검사·검증된 Artifact와 정확히 연결되어 있어야 한다는 것이다.

Heliopause는 단순히 package 이름이나 version 문자열이 같다는 이유만으로 새로운 Artifact를 다시 외부 source에서 받아 설치하지 않는다.

Package ecosystem에서는 최종적으로 사용할 Primary Artifact와 dependency의 정확한 집합을 Verified Set으로 관리한다.

```text
Inspection Run
      ↓
Primary Artifact + dependencies
      ↓
exact identity / digest
      ↓
Verified Set
      ↓
Verified Manifest
      ↓
Policy = ALLOW
      ↓
Promotion
```

Promotion에 사용되는 Artifact와 dependency는 해당 Verified Set / Verified Manifest 및 관련 Inspection Run에 연결되어 있어야 한다. 다른 Run의 Decision, 다른 digest의 Artifact 또는 검증되지 않은 dependency를 임의로 섞어 설치하지 않는다.

### Package ecosystem의 Promotion

npm 및 PyPI/pip와 같이 dependency를 포함하는 Package ecosystem에서는 일반적으로 다음 경로를 사용한다.

```text
Quarantine / Inspection
        ↓
Policy = ALLOW
        ↓
Verified Set / Verified Manifest 확정
        ↓
Staging 반입 전 identity / digest 재검증
        ↓
Staging
        ↓
사용자 환경 반입 전 identity / digest 재검증
        ↓
trusted Promotion
        ↓
사용자 환경 설치
```

Staging은 사용자 환경에 직접 노출되기 전의 검증 완료 Artifact 보관·전달 영역이며, 새로운 검사를 수행하거나 Artifact를 실행하는 두 번째 Sandbox로 사용하지 않는다.

가능한 경우 실제 사용자 환경 설치는 외부 registry에서 Artifact를 다시 자유롭게 다운로드하는 방식보다 Heliopause가 검증한 local Verified Set을 사용하여 수행한다.

```text
검사한 것
=
실제로 설치하는 것
```

### 설치 과정에서 새로운 Artifact가 필요한 경우

최종 사용자 환경 설치 과정에서 기존 Verified Set에 존재하지 않는 새로운 dependency 또는 Artifact가 요구되면 Heliopause는 이를 자동으로 외부에서 받아 사용자 환경에 설치하지 않는다.

```text
Promotion / install 진행
        ↓
Verified Set에 없는 Artifact 필요
        ↓
자동 신뢰 금지
        ↓
Promotion 중단
        ↓
Artifact acquisition / Verification으로 회귀
        ↓
필요한 검사 수행
        ↓
새로운 유효한 Verified Set / Policy 결과 확보
```

package manager의 dependency resolution이 검사 단계와 실제 설치 단계 사이에서 달라지더라도 새로운 구성요소가 보안 경계를 우회하여 사용자 환경으로 유입되지 않아야 한다. 정확한 dependency locking, offline install 및 package manager별 enforcement 방식은 후속 ecosystem/Tooling 설계에서 정의한다.

### Promotion 전 재검증

Promotion 직전에는 실제 반입하려는 Artifact가 검사·검증된 대상과 동일한지 확인한다.

최소한 identity와 digest의 연결을 확인하며, package ecosystem에서는 Verified Set / Verified Manifest와의 일관성도 확인한다.

```text
검증된 digest
      ↓
Promotion 직전 실제 Artifact digest
      ↓
      ├─ 일치   → Promotion 계속
      └─ 불일치 → Promotion 중단
```

불일치가 발생하면 기존 `ALLOW`가 존재한다는 이유만으로 반입을 계속하지 않는다. 필요한 acquisition/Verification/Inspection 단계로 돌아가 새 상태를 기준으로 판단한다.

### Standalone binary/archive의 직접 반입

Package manager가 필요 없는 standalone binary/archive는 Architecture-008에 정의된 예외에 따라 반드시 별도의 Staging 단계를 거치지 않을 수 있다.

이 경우에도 Sandbox나 검사 환경이 사용자 Host에 직접 파일을 작성해서는 안 된다.

```text
Standalone binary/archive
        ↓
Inspection
        ↓
Policy = ALLOW
        ↓
identity / digest 재검증
        ↓
trusted Promotion
        ↓
사용자 지정 위치로 직접 반입 가능
```

Staging 생략은 Promotion 신뢰 경계를 생략한다는 의미가 아니다. 실제 사용자 환경으로의 쓰기 작업은 Heliopause의 trusted Promotion 책임을 통해서만 수행한다.

### 설치 대상이 모호하거나 충돌하는 경우

Promotion을 안전하게 수행하기 위한 사용자 환경의 대상 위치 또는 설치 context가 명확하지 않은 경우 Heliopause는 임의로 중요한 값을 추측하여 반입하지 않는다.

예를 들어 기존 파일 덮어쓰기, 서로 다른 설치 범위 또는 의미가 달라지는 대상 선택처럼 사용자의 의도가 추가로 필요한 상황에서는 설치를 진행하기 전에 명확한 입력을 요구할 수 있다.

다만 정상적인 `ALLOW` install마다 반복적으로 별도의 Promotion confirmation을 요구하지 않는다.

```text
정상적이고 원래 요청이 명확함
→ 추가 confirmation 없이 계속

Promotion 대상이나 의미가 모호함
→ 필요한 사용자 입력 요구
```

구체적인 overwrite 정책, confirmation 조건 및 filesystem conflict 처리는 후속 CLI/Promotion 설계에서 결정한다.

### Promotion 자체의 실패

`Policy = ALLOW`는 Artifact가 Heliopause의 검사 정책을 통과했다는 의미이지 실제 사용자 환경 설치가 반드시 성공했다는 의미는 아니다.

예를 들어 filesystem 오류, 권한 문제, package manager 오류 또는 대상 환경 문제로 실제 설치 작업이 실패할 수 있다.

따라서 Heliopause는 다음 두 상태를 구분한다.

```text
Policy Decision
→ Artifact에 대한 보안 판정

Operation Status
→ 사용자가 요청한 실제 install/import 작업 결과
```

예를 들어 다음과 같은 상태가 가능하다.

```text
Policy: ALLOW
Installation: FAILED
Reason: target permission denied
```

일반적인 설치 작업 실패 자체가 기존 `ALLOW` Policy Decision을 자동으로 `BLOCK`으로 바꾸지는 않는다. 다만 설치 중 검증되지 않은 Artifact 요구, digest mismatch 또는 기타 보안 invariant 위반이 발견되면 Promotion을 중단하고 필요한 verification/inspection 경로로 되돌린다.

구체적인 Operation Status와 failure taxonomy는 후속 Domain Model에서 정의한다.

### `inspect`와 Promotion

사용자가 `inspect`를 요청한 경우에는 최종 Policy Decision이 `ALLOW`여도 Promotion을 수행하지 않는다.

```text
helox npm inspect kenv
        ↓
Policy = ALLOW
        ↓
검사 결과 기록
        ↓
Promotion 없음
        ↓
종료
```

Promotion 여부는 Policy Decision만으로 결정되는 것이 아니라 사용자가 처음 요청한 operation과 Policy Decision을 함께 기준으로 한다.

```text
install + ALLOW
→ Promotion 가능

inspect + ALLOW
→ Promotion하지 않음

MANUAL_REVIEW
→ Promotion 보류

BLOCK
→ Promotion 금지
```

### Promotion 결과 기록

실제 설치·반입을 수행한 경우 해당 결과는 원래 Inspection Run과 검증된 Artifact identity/digest에 추적 가능하게 연결되어야 한다.

최소한 다음 사실을 확인할 수 있어야 한다.

- 어떤 Inspection Run을 근거로 했는가
- 어떤 Policy Decision을 사용했는가
- 어떤 Verified Set / Manifest를 사용했는가
- 어떤 Artifact identity/digest를 실제 반입했는가
- 어떤 사용자 operation을 수행했는가
- 실제 설치·반입이 성공했는가

구체적인 Promotion Record 또는 Operation Result schema는 후속 Domain Model / Evidence/Result 설계에서 정의한다.

### 후속 결정으로 유보한 항목

- npm/pip의 실제 offline/local install 구현 방식
- package manager별 dependency locking 방식
- Staging의 실제 filesystem 위치와 layout
- Staging retention / cleanup 정책
- ecosystem별 install option 전달 규칙
- local/global install 세부 처리
- overwrite / filesystem conflict 정책
- Promotion confirmation이 필요한 정확한 조건
- standalone binary/archive 대상 위치 syntax
- Promotion 실패 retry 방식
- Operation Status 세부 enum
- Promotion Record schema
- 권한 상승이 필요한 설치 작업 처리
- package manager별 transaction / rollback 지원 여부

위 항목은 후속 CLI IA, Domain Model, Staging/Promotion, ecosystem adapter 및 Tooling 설계에서 구체화한다.

### 압축된 사용자 모델

```text
helox npm install kenv
        ↓
보안 검사
        ↓
Policy = ALLOW
        ↓
검사한 정확한 Artifact / dependency set 확인
        ↓
digest 재검증
        ↓
trusted Promotion
        ↓
원래 요청했던 위치·환경에 설치
        ↓
설치 결과 기록
```

핵심 invariant는 다음과 같다.

```text
검사한 것
=
ALLOW의 대상
=
Verified Set / Manifest의 대상
=
실제로 Promotion하는 것
```

Heliopause는 사용자가 별도의 Promotion 단계를 직접 관리하지 않아도, 검사한 정확한 Artifact만 원래 요청한 사용자 환경으로 안전하게 반입되도록 하는 것을 기본 원칙으로 한다.

### 구조화된 결정과 구현 영향

- Promotion은 일반 사용자가 내부 단계로 직접 조작하지 않는 trusted boundary다.
- `install`의 원래 ecosystem/source·operation·Artifact·resolved reference·대상 환경·범위·허용 option·standalone 위치 context를 유지한다.
- 실제 반입 Artifact/dependency는 동일 Inspection Run·Verified Set·Verified Manifest·identity/digest와 연결되어야 한다.
- Package는 Quarantine/Inspection → ALLOW → Verified Set/Verified Manifest 확정 → Staging 반입 전 identity/digest 재검증 → Staging → 사용자 환경 반입 전 identity/digest 재검증 → trusted Promotion → 사용자 환경 설치 경로를 사용한다.
- Staging은 실행·재검사·두 번째 Sandbox가 아닌 검증 완료 Artifact 보관·전달 영역이다.
- Verified Set 밖의 새 Artifact/dependency가 요구되면 Promotion을 중단하고 acquisition/Verification/필요한 검사로 되돌린다.
- Promotion 직전 실제 Artifact identity/digest와 Verified Set/Manifest 일관성을 재확인하고 불일치 시 중단한다.
- standalone binary/archive는 Staging 생략이 가능하지만 trusted Promotion과 identity/digest 재검증은 생략하지 않는다.
- 설치 대상·범위·덮어쓰기 등 context가 모호하면 필요한 입력을 요구하되 정상 ALLOW마다 반복 confirmation하지 않는다.
- `Policy Decision`과 실제 `Operation Status`를 분리하며 `Policy: ALLOW`와 `Installation: FAILED`가 함께 기록될 수 있다.
- `inspect + ALLOW`는 Promotion하지 않고, `MANUAL_REVIEW`는 보류, `BLOCK`은 금지한다.
- Promotion Record/Operation Result는 Run·Policy·Verified Set/Manifest·실제 identity/digest·operation·성공 여부를 추적해야 한다.

## User-Journey-008 누락 점검

- [x] Promotion의 trusted boundary와 일반 사용자 직접 조작 금지
- [x] 원래 install 실행 context 유지
- [x] 검사 대상·Verified Set/Manifest·실제 반입 대상 동일성
- [x] Package의 Verified Set·Staging·digest 재검증 경로
- [x] Staging의 비실행·비검사 보관·전달 역할
- [x] Verified Set 밖 새 Artifact/dependency의 자동 신뢰 금지
- [x] Promotion 직전 identity/digest 재검증
- [x] standalone binary/archive의 Staging 생략 예외와 trusted Promotion 유지
- [x] 모호한 설치 대상의 사용자 명확화와 confirmation 최소화
- [x] Policy Decision과 Operation Status 분리
- [x] inspect ALLOW의 Promotion 없음
- [x] MANUAL_REVIEW 보류 및 BLOCK 금지
- [x] Promotion 결과의 Run·Policy·Manifest·identity/digest·operation 추적
- [x] offline install·locking·layout·overwrite·retry·schema·권한·rollback 후속 결정 유보
