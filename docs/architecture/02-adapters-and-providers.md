# Architecture — Adapters and Providers

## 사용자 원문: Architecture-004

```text
Architecture-004: Artifact Source / External Tool Adapter 구조

Heliopause의 외부 연동은 성격에 따라 Artifact Adapter와 External Tool Provider/Adapter로 구분한다. 서로 다른 책임을 하나의 범용 adapter 계약으로 억지로 통합하지 않는다.

Artifact Adapter는 npm, PyPI/pip, GitHub Releases 등 생태계별 입력 해석, Artifact/source 식별, version resolution, metadata 및 dependency 정보 획득, Artifact acquisition과 생태계별 integrity 정보 수집을 담당하고, 그 결과를 생태계 중립적인 공통 Artifact 계약으로 변환한다.

Artifact Adapter가 획득한 Artifact는 임의의 사용자/프로젝트 경로에 저장하지 않고 Heliopause가 관리하는 통제된 quarantine/intake 영역으로 전달한다. Adapter는 Artifact를 획득·식별할 뿐 압축 해제·설치·실행하지 않으며, 그러한 처리는 Inspection/Sandbox 경계를 통해 수행한다.

Artifact Adapter는 Policy 판정, Sandbox 제어, Promotion 등을 직접 수행하지 않는다.

External Tool Provider/Adapter는 verifier, scanner, vulnerability source, SBOM generator 등 외부 도구 또는 서비스를 감싸며, 도구별 command, API, exit code, 출력 형식과 오류를 내부에서 해석하여 공통 Verification Result, Finding, Evidence 등으로 정규화한다. Core와 Policy는 외부 도구의 고유 출력 형식에 직접 의존하지 않는다.

각 Adapter/Provider는 자신이 지원하는 기능을 Capability로 명시한다. 지원하지 않는 기능, 사용할 수 없는 정보, 실제 검사 실패와 정상 검사 결과를 구분하여 반환하며, 미지원 또는 미실행 상태를 안전 판정으로 해석하지 않는다.

MVP에서는 Adapter/Provider의 동적 plugin loading을 구현하지 않고, Composition Root에서 필요한 구현을 명시적으로 생성·등록한다. 향후 외부 확장 필요성이 실제로 생기면 기존 공통 계약을 유지하는 범위에서 plugin 구조를 추가할 수 있다.

생태계별 또는 도구별 특수 로직은 해당 Adapter/Provider 내부에 한정하며, 새로운 Artifact source나 검사 도구를 추가하기 위해 Core 또는 Policy에 해당 구현 전용 분기를 추가하지 않는다
```

## 구조화된 결정

### Architecture-004 외부 연동 책임 분리

외부 연동은 성격에 따라 두 계약군으로 구분한다. 서로 다른 책임을 하나의 범용 adapter 계약으로 통합하지 않는다.

| 계약군 | 대상 | 핵심 책임 | 반환 경계 |
| --- | --- | --- | --- |
| Artifact Adapter | npm, PyPI/pip, GitHub Releases 등 Artifact source | 입력 해석, Artifact/source 식별, version resolution, metadata·dependency 획득, acquisition, 생태계별 integrity 정보 수집 | 생태계 중립 공통 Artifact 계약 |
| External Tool Provider/Adapter | verifier, scanner, vulnerability source, SBOM generator 등 도구·서비스 | command·API·exit code·출력 형식·오류 해석 | 공통 Verification Result, Finding, Evidence 등 |

Artifact Adapter는 Policy 판정, Sandbox 제어, Promotion을 직접 수행하지 않는다. External Tool Provider/Adapter는 외부 도구의 고유 출력 형식을 내부에서 해석하며 Core와 Policy는 그 형식에 직접 의존하지 않는다.

### Architecture-004 Artifact Adapter 경계

- npm, PyPI/pip, GitHub Releases별 입력과 metadata를 adapter 내부에서 해석한다.
- Artifact/source identity와 version resolution을 adapter가 담당한다.
- dependency 정보와 acquisition 결과를 수집한다.
- 생태계별 integrity 정보를 수집하되 공통 Artifact 계약으로 변환한다.
- 획득한 Artifact를 임의의 사용자/프로젝트 경로가 아닌 Heliopause가 관리하는 통제된 quarantine/intake 영역으로 전달한다.
- Artifact를 획득·식별할 뿐 압축 해제·설치·실행하지 않으며, 해당 처리는 Inspection/Sandbox 경계를 통해 수행한다.
- Policy, Sandbox, Promotion을 호출하거나 직접 결정하지 않는다.

### Architecture-004 External Tool Provider/Adapter 경계

- verifier, scanner, vulnerability source, SBOM generator를 Provider/Adapter로 감싼다.
- 도구별 command, API, exit code, 출력 형식과 오류를 내부에서 해석한다.
- 공통 Verification Result, Finding, Evidence로 정규화한다.
- Core와 Policy가 외부 도구의 독자적인 출력 형식·exit code·API type을 직접 참조하지 않도록 한다.

### Architecture-004 Capability와 상태 구분

각 Adapter/Provider의 기능 지원 여부(Capability)와 개별 Inspection Run에서의 실제 실행 상태(Execution Status)를 서로 다른 개념으로 관리한다. 정확한 enum 이름은 Domain Model에서 결정한다.

```text
Capability
├─ Supported
└─ Unsupported

Execution Status
├─ Executed / Completed
├─ Failed
├─ Incomplete
├─ Not Executed / Skipped
└─ Unavailable
```

| 개념 | 의미 | 안전 판정 해석 |
| --- | --- | --- |
| Capability | Adapter/Provider가 해당 기능을 지원하는지 | `Unsupported`는 안전으로 해석하지 않음 |
| Execution Status | 개별 Inspection Run에서 실제 실행된 상태 | `Failed`, `Incomplete`, `Not Executed/Skipped`, `Unavailable`은 정상 검사 결과와 구분하고 사유 기록 |

Capability와 Execution Status를 공통 상태로 섞지 않는다. 미지원 기능, 사용할 수 없는 정보, 실제 검사 실패, 미완료·미실행 상태를 정상 검사 결과나 안전 판정으로 해석하지 않는다.

### Architecture-004 Plugin 정책

- MVP에서는 Adapter/Provider 동적 plugin loading을 구현하지 않는다.
- 필요한 구현은 Composition Root에서 명시적으로 생성·등록한다.
- 향후 외부 확장 필요성이 실제로 생기면 기존 공통 계약을 유지하는 범위에서 plugin 구조를 추가할 수 있다.
- plugin 도입 시에도 Core·Policy에 생태계·도구 전용 분기를 추가하지 않는다.

## Architecture-004 구현 영향

- Artifact Adapter Port와 External Tool Provider/Adapter Port를 별도 계약으로 정의한다.
- Artifact Adapter의 출력은 생태계 중립 Artifact identity·metadata·dependency·acquisition 계약으로 정규화한다.
- Artifact Adapter의 acquisition 출력은 통제된 quarantine/intake 영역으로만 전달되도록 한다.
- Adapter는 압축 해제·설치·실행을 수행하지 않고 Inspection/Sandbox 경계로 넘긴다.
- Provider 출력은 Verification Result·Finding·Evidence 계약으로 정규화한다.
- Capability와 Execution Status를 모든 Adapter/Provider 결과에 별도 포함한다.
- Capability와 개별 Inspection Run의 실행 상태를 구분하는 공통 모델을 둔다. 정확한 enum 이름은 Domain Model에서 결정한다.
- Composition Root에서 npm·PyPI·GitHub Releases adapter와 verifier·scanner·vulnerability·SBOM provider를 명시적으로 생성·등록한다.
- Adapter/Provider의 command·API·exit code·출력 파싱 오류는 내부 경계에서 처리하고 Core·Policy로 누출하지 않는다.
- 새 Artifact source나 검사 도구를 추가할 때 Core·Policy의 구현 전용 분기 없이 기존 Port/Contract를 구현한다.
- Policy는 Capability와 검사 상태를 포함한 정규화 결과를 받아 최종 결정을 내리며 Adapter/Provider를 직접 호출하지 않는다.

## Architecture-004 누락 점검

- [x] Artifact Adapter와 External Tool Provider/Adapter를 별도 책임으로 구분
- [x] 범용 adapter 계약으로 서로 다른 책임을 억지로 통합하지 않음
- [x] Artifact Adapter의 npm·PyPI/pip·GitHub Releases 입력 해석
- [x] Artifact/source 식별과 version resolution
- [x] metadata·dependency 획득
- [x] Artifact acquisition과 생태계별 integrity 정보 수집
- [x] 획득 Artifact의 통제된 quarantine/intake 전달
- [x] Adapter의 압축 해제·설치·실행 금지와 Inspection/Sandbox 위임
- [x] 생태계 중립 공통 Artifact 계약 변환
- [x] Artifact Adapter의 Policy·Sandbox·Promotion 직접 수행 금지
- [x] External Tool Provider/Adapter의 verifier·scanner·vulnerability source·SBOM generator 포괄
- [x] command·API·exit code·출력 형식·오류 내부 해석
- [x] Verification Result·Finding·Evidence 정규화
- [x] Core·Policy의 외부 도구 고유 출력 형식 직접 의존 금지
- [x] Adapter/Provider Capability 명시
- [x] Capability와 Execution Status의 개념 분리
- [x] Executed/Completed·Failed·Incomplete·Not Executed/Skipped·Unavailable 실행 상태 구분
- [x] 미지원·사용 불가·미실행·실패·정상 결과 구분
- [x] 미지원·미실행을 안전 판정으로 해석하지 않음
- [x] MVP 동적 plugin loading 미구현
- [x] Composition Root의 명시적 생성·등록
- [x] 향후 plugin 확장 시 기존 공통 계약 유지
- [x] 특수 로직을 해당 Adapter/Provider 내부에 한정
- [x] 새 source·도구 추가 시 Core·Policy 전용 분기 금지

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-004 | 2026-08-10 | Artifact Adapter와 External Tool Provider/Adapter를 분리하고 Capability·실행 상태를 정규화하며 MVP는 Composition Root 명시 등록으로 운영 | source 해석과 검사 도구 연동의 책임을 분리하고 외부 도구 교체·확장을 Core/Policy 변경 없이 가능하게 하기 위해 | 두 Port 계약, 상태·Capability 모델, 명시적 Bootstrap wiring, plugin 후순위 원칙이 설계 기준이 됨 |
