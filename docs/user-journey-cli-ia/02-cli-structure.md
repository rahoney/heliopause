# User Journey + CLI IA — CLI Structure

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
