# User Journey + CLI IA — Journey and Input

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
