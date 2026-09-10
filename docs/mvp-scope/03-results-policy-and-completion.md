# MVP Scope — Results, Policy, and Completion

## 사용자 원문: MVP-008

```text
8번: SBOM 및 Evidence/결과물 범위

MVP에서는 모든 검사 결과를 단순 PASS/FAIL로만 남기지 않고, Artifact identity, 수행된 검사, Finding, Evidence 및 최종 Policy Decision을 연결하여 재현 가능한 검사 기록을 생성한다.

결과에는 Artifact의 이름·버전·source·digest·유형과 dependency 정보, 사용된 검사 도구와 버전, 검사 시점, 수행·미수행 검사와 그 이유, 주요 정적·동적 관찰 결과를 포함한다.

Dependency와 구성요소는 가능한 범위에서 SBOM으로 생성하며, 표준 포맷 사용을 원칙으로 하되 구체적인 SBOM 표준과 생성 도구는 이후 단계에서 확정한다.

ALLOW되어 Staging Area로 승격되는 경우에는 실제 승격 대상 Artifact와 dependency의 digest, 검사 결과 및 Evidence를 연결한 검증 완료 세트 Manifest를 생성한다.

결과는 사람이 터미널에서 빠르게 확인할 수 있는 요약과, CI·MCP·자동화 도구가 처리할 수 있는 machine-readable 구조화 결과를 함께 제공한다.

Evidence와 결과물의 저장 방식·보존 기간·세부 schema는 MVP Architecture 및 Domain Model 단계에서 확정한다.
```

## 구조화된 결정

### MVP-008 재현 가능한 검사 기록

모든 검사 결과는 단순 `PASS`/`FAIL`만으로 남기지 않는다. 다음 요소를 연결하여 Artifact를 동일하게 다시 검사하고 판정을 재현할 수 있는 검사 기록을 생성한다.

| 연결 요소 | 포함 내용 |
| --- | --- |
| Artifact identity | 이름·버전·source·digest·유형 |
| 수행된 검사 | 수행·미수행 검사, 미수행 이유, 사용된 검사 도구·버전 |
| Finding | 정적·동적 검사에서 확인된 문제·위험 신호 |
| Evidence | 관찰 결과와 판정 근거 |
| Policy Decision | 최종 `ALLOW`, `MANUAL_REVIEW`, `BLOCK` 등 정책 결과 |
| 시점 | 검사 시점과 관련 실행 환경 정보 |
| dependency | dependency 정보와 가능한 구성요소 관계 |

결과에는 주요 정적·동적 관찰 결과와 수행되지 않은 검사의 이유를 함께 포함한다.

### MVP-008 SBOM과 Staging Manifest

- Dependency와 구성요소는 가능한 범위에서 SBOM으로 생성한다.
- SBOM은 표준 포맷 사용을 원칙으로 한다.
- 구체적인 SBOM 표준과 생성 도구는 이후 Architecture·Domain Model 및 도구 선택 단계에서 확정한다.
- Artifact가 `ALLOW`되어 Staging Area로 승격되는 경우 실제 승격 대상 Artifact와 dependency의 digest를 기록한다.
- 승격 세트의 검사 결과와 Evidence를 연결한 검증 완료 세트 Manifest를 생성한다.
- Manifest는 검사된 대상과 실제 반입 대상의 동일성을 확인할 수 있어야 한다.

### MVP-008 결과 표현과 후속 확정 항목

- 사람이 터미널에서 빠르게 확인할 수 있는 요약 결과를 제공한다.
- CI·MCP·자동화 도구가 처리할 수 있는 machine-readable 구조화 결과를 함께 제공한다.
- Evidence와 결과물의 저장 방식, 보존 기간, 세부 schema는 MVP Architecture 및 Domain Model 단계에서 확정한다.

### MVP-008 구현 영향

- 결과 생성기는 Artifact identity와 실행별 검사 기록을 루트로 하여 Finding·Evidence·Policy Decision을 참조 연결한다.
- 결과에 performed checks와 skipped checks를 구분하고 skipped 사유를 필수 필드로 둔다.
- 검사 도구·버전, 검사 시점, 실행 OS·backend, Artifact·dependency digest를 결과에 기록한다.
- 정적·동적 관찰 결과는 원본 Evidence와 요약 Finding을 연결한다.
- SBOM provider를 Core 결과 계약에 연결하되 특정 표준·도구 구현은 provider 경계에 둔다.
- Staging 승격 시 검증 완료 세트 Manifest에 Artifact·dependency identity/digest, 검사 결과와 Evidence 참조를 포함한다.
- 터미널 요약과 machine-readable 출력이 동일한 판정·identity·digest를 나타내는지 contract test로 검증한다.
- 저장소·보존·schema 확정 전까지 결과 포맷을 임의로 고정하지 않고 Architecture·Domain Model 결정으로 이관한다.

### MVP-008 누락 점검

- [x] 단순 PASS/FAIL을 넘어 재현 가능한 검사 기록 생성
- [x] Artifact identity 연결
- [x] 수행·미수행 검사와 미수행 이유 포함
- [x] Finding·Evidence·최종 Policy Decision 연결
- [x] Artifact 이름·버전·source·digest·유형 포함
- [x] dependency 정보 포함
- [x] 검사 도구·버전과 검사 시점 포함
- [x] 주요 정적·동적 관찰 결과 포함
- [x] 가능한 범위의 SBOM 생성
- [x] SBOM 표준 포맷 원칙
- [x] SBOM 표준·생성 도구 후속 확정
- [x] ALLOW된 Staging 승격 대상의 Artifact·dependency digest 기록
- [x] 검사 결과·Evidence를 연결한 검증 완료 세트 Manifest 생성
- [x] 터미널용 사람 중심 요약 제공
- [x] CI·MCP·자동화용 machine-readable 결과 제공
- [x] 저장 방식·보존 기간·세부 schema를 Architecture·Domain Model 단계에서 확정

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-008 | 2026-08-08 | Artifact·검사·Finding·Evidence·Policy Decision을 연결한 재현 가능한 결과와 SBOM·Staging Manifest를 생성하고 사람용·기계용 출력을 함께 제공 | 판정만 전달하지 않고 검사 근거와 실제 반입 대상의 동일성을 추적·재현하기 위해 | 결과 계약, SBOM provider, 승격 Manifest, 터미널·machine-readable 출력 및 후속 저장·보존·schema 결정이 MVP 조건이 됨 |

## 사용자 원문: MVP-009

```text
9번: MVP 사용자 흐름 및 완료 범위



사용자가 Artifact 지정

        ↓

Artifact/source 식별

        ↓

격리 환경으로 다운로드

        ↓

출처·digest·signature·provenance 검증

        ↓

정적 검사

        ↓

격리 설치 및 동적 검사

        ↓

Policy 판정

        ↓

 ┌───────────────┬───────────────┐

 ALLOW        MANUAL_REVIEW       BLOCK

  ↓               ↓               ↓

Staging       결과·근거 제공      반입 금지

  ↓

digest 재검증

  ↓

사용자가 지정한 환경에 설치

  ↓

Manifest / SBOM / Evidence / 결과 제공


`ALLOW`된 Artifact는 원칙적으로 검증 완료 Staging Area를 통해 사용자 환경으로 반입한다. 다만 package manager가 필요 없는 독립 binary/archive는 Threat Model D-011에 따라 검사 대상과 반입 대상의 동일성을 digest로 재검증한 후 사용자가 지정한 위치로 직접 반입할 수 있다. `WARN`은 독립적인 최종 Policy Decision이 아니라 검사 과정에서 발견된 경고 수준의 Finding으로 기록한다. 자동 판정이 불가능하거나 사람의 판단이 필요한 경우 최종 Policy Decision은 `MANUAL_REVIEW`로 처리한다. `BLOCK`된 Artifact는 사용자 환경으로 반입하지 않는다.

MVP는 npm에서 전체 end-to-end 흐름이 완성되고, PyPI/pip와 GitHub Releases가 동일한 공통 Core와 adapter 계약을 통해 정의된 MVP 흐름을 수행할 수 있는 상태를 완료 기준으로 한다. 특정 생태계를 추가하기 위해 공통 Core에 해당 생태계 전용 로직을 넣어야 하는 구조는 MVP 완료로 보지 않는다.
```

## 구조화된 결정

### MVP-009 사용자 흐름 확정

```text
사용자 Artifact 지정
  → Artifact/source 식별
  → 격리 환경 다운로드
  → 출처·digest·signature·provenance 검증
  → 정적 검사
  → 격리 설치·동적 검사
  → Policy 판정
      ├─ ALLOW
      │    ├─ 일반 경로 → Staging → digest 재검증 → 사용자 지정 환경 설치 → Manifest/SBOM/Evidence/결과 제공
      │    └─ 독립 binary/archive → digest 재검증 → 사용자 지정 위치 직접 반입 가능
      ├─ MANUAL_REVIEW → 결과·근거 제공, 추가 판단 대기
      └─ BLOCK → 사용자 환경 반입 금지
```

각 단계의 수행 여부와 결과는 공통 검사 기록에 연결한다. `ALLOW`된 Artifact는 원칙적으로 검증 완료 Staging Area를 통해 사용자 환경으로 반입한다. 다만 package manager가 필요 없는 독립 binary/archive는 Threat Model D-011에 따라 검사 대상과 반입 대상의 동일성을 digest로 재검증한 후 사용자가 지정한 위치로 직접 반입할 수 있다.

### MVP-009 Policy 분기 및 반입 규칙

| Policy 결과 | 다음 동작 |
| --- | --- |
| `ALLOW` | 일반 경로: Staging Area 승격 → 반입 직전 digest 재검증 → 사용자가 지정한 환경에 설치 → Manifest·SBOM·Evidence·결과 제공. 독립 binary/archive 예외: Threat Model D-011에 따라 검사 대상과 반입 대상의 동일성을 digest로 재검증한 뒤 사용자 지정 위치로 직접 반입 가능 |
| `MANUAL_REVIEW` | 결과와 근거를 제공하고 사람의 추가 판단이 필요한 상태로 유지하며 자동 반입하지 않음 |
| `BLOCK` | 사용자 환경으로 반입·설치 금지 |

`WARN`은 독립적인 최종 Policy Decision이 아니라 Finding의 경고 수준을 나타내며, 사람의 추가 판단이 필요한 경우 최종 Policy Decision은 `MANUAL_REVIEW`로 처리한다.

### MVP-009 완료 기준

- npm에서 사용자 지정부터 설치·결과 제공까지 전체 end-to-end 흐름이 완성되어야 한다.
- PyPI/pip와 GitHub Releases는 동일한 공통 Core와 adapter 계약으로 정의된 MVP 흐름을 수행할 수 있어야 한다.
- 세 생태계의 기능 수준이 완전히 동일할 필요는 없지만, 공통 단계·결과 계약·Policy 분기를 깨뜨리지 않아야 한다.
- 특정 생태계를 추가하기 위해 공통 Core에 해당 생태계 전용 로직을 넣어야 하는 구조는 MVP 완료로 보지 않는다.

### MVP-009 구현 영향

- 전체 workflow orchestration은 각 단계의 입력·출력·상태·Evidence 참조를 연결한다.
- Policy 결과에 따라 일반 Staging 승격, 독립 binary/archive 직접 반입 예외, 추가 검토 대기, 반입 차단을 명시적으로 분기한다.
- `ALLOW` 일반 경로에서는 Staging 승격 후 digest 재검증을 필수 단계로 둔다.
- package manager가 필요 없는 독립 binary/archive의 직접 반입은 Threat Model D-011에 따라 검사 대상과 반입 대상의 digest 재검증이 완료된 경우에만 허용한다.
- 사용자 지정 환경 설치는 검증 완료 세트와 연결된 Manifest를 사용한다.
- `WARN`은 Finding severity로만 기록하고, 사람의 판단이 필요한 최종 결과는 `MANUAL_REVIEW`로 자동 설치 경로와 분리하며 사용자가 확인할 수 있는 근거를 제공한다.
- npm reference implementation과 PyPI/pip·GitHub Releases adapter에 동일한 workflow·contract test를 적용한다.
- Core에 생태계 전용 분기가 추가되는지 architecture/dependency rule로 검사한다.

### MVP-009 누락 점검

- [x] 사용자 Artifact 지정
- [x] Artifact/source 식별
- [x] 격리 환경 다운로드
- [x] 출처·digest·signature·provenance 검증
- [x] 정적 검사
- [x] 격리 설치 및 동적 검사
- [x] Policy 판정과 ALLOW/MANUAL_REVIEW/BLOCK 분기
- [x] WARN을 최종 Policy Decision이 아닌 Finding 경고 수준으로 기록
- [x] ALLOW Artifact의 원칙적 Staging Area 경유
- [x] 독립 binary/archive의 D-011 기반 digest 재검증 후 직접 반입 예외
- [x] Staging 후 digest 재검증
- [x] 사용자 지정 환경 설치
- [x] Manifest·SBOM·Evidence·결과 제공
- [x] MANUAL_REVIEW의 결과·근거 제공과 사람의 추가 판단 상태
- [x] BLOCK Artifact의 사용자 환경 반입 금지
- [x] npm 전체 end-to-end 완료 기준
- [x] PyPI/pip·GitHub Releases의 동일 Core·adapter 흐름 수행 기준
- [x] Core 생태계 전용 로직 필요 시 MVP 미완료 판정

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-009 | 2026-08-08 | 지정·식별·격리 다운로드·검증·정적·동적 검사·Policy·Staging/설치·결과 제공의 전체 흐름을 정의하고 npm E2E와 두 adapter의 공통 계약 수행을 완료 기준으로 삼음 | 사용자가 실제로 재현 가능한 반입 흐름을 사용하고 생태계 중립 Core 확장성을 검증하기 위해 | `WARN` Finding과 최종 Policy Decision 분리, 일반 Staging 경로·독립 binary/archive D-011 직접 반입 예외, digest 재검증, 공통 contract test와 Core 전용 로직 금지가 MVP 조건이 됨 |
