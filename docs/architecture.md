# Heliopause Artifact Airlock Architecture

이 문서는 Architecture 결정의 사용자 원문과 구조화된 설계 기준을 보존한다. 각 Architecture 결정은 사용자 원문 → 구조화된 결정 → 구현 영향 → 누락 점검 순서로 기록한다.

## 사용자 원문: Architecture-001

```text
Architecture-001

Heliopause Artifact Airlock은 Go 기반의 Modular Monolith CLI로 구성한다. 하나의 배포 가능한 애플리케이션 내부에서 책임별 모듈을 명확히 분리하며, microservice로 분할하지 않는다.

외부 생태계, 검사 도구, 격리 backend, 저장소 및 Promotion 구현과 Core 사이에는 명확한 Port/Adapter 경계를 둔다. Core와 Application 계층은 npm, PyPI, GitHub Releases, Cobra, 특정 scanner, container runtime 등의 구체 구현에 직접 의존하지 않는다.

Ports/Adapters 원칙은 외부 의존성을 격리하기 위한 범위에서 적용하며, 불필요한 추상화 계층이나 복잡한 framework를 도입하지 않는다. Heliopause 본체는 하나의 배포 가능한 CLI 애플리케이션과 하나의 주 코드베이스를 유지한다. 다만 package manager, scanner, verifier, container/sandbox runtime 등 외부 도구와 backend는 Port/Adapter 경계 뒤에서 별도 process·container 등으로 실행될 수 있으며, 이를 microservice 분산 구조로 간주하지 않는다.

향후 특정 저수준 isolation component를 Rust 등 별도 구현으로 분리할 수 있으나, 그것을 전제로 MVP를 분산 구조로 설계하지 않는다.

CLI ───────────────→ Application / Workflow
                         │
                         ├────→ Core / Domain
                         ├────→ Policy ─────→ Core / Domain
                         └────→ Ports / Contracts ─────→ Core / Domain
                                      ↑
                                      │ implements
                         Adapters / Backends / Providers

Composition Root
    └─ 구체 구현 생성·연결
```

## 구조화된 결정

### Architecture-001 전체 Architecture Style

- Heliopause Artifact Airlock은 Go 기반의 Modular Monolith CLI로 구성한다.
- Heliopause 본체는 하나의 배포 가능한 CLI 애플리케이션과 하나의 주 코드베이스를 유지하며 책임별 모듈을 분리한다.
- package manager, scanner, verifier, container/sandbox runtime 등 외부 도구와 backend는 Port/Adapter 경계 뒤에서 별도 process·container 등으로 실행될 수 있다.
- MVP에서는 microservice로 분할하지 않는다.
- 모듈 경계와 외부 경계는 유지하되, 불필요한 추상화 계층이나 복잡한 framework는 도입하지 않는다.
- 저수준 isolation component를 향후 Rust 등 별도 구현으로 분리할 수 있으나, MVP를 분산 구조로 설계하는 전제는 두지 않는다.

### Architecture-001 Core·Application과 외부 세계의 경계

외부 생태계, 검사 도구, 격리 backend, 저장소 및 Promotion 구현은 Port/Adapter 경계를 통해 연결한다. Core와 Application 계층은 다음 구체 구현을 직접 참조하지 않는다.

- npm
- PyPI
- GitHub Releases
- Cobra
- 특정 scanner
- container runtime

Ports/Adapters 원칙은 외부 의존성 격리에 필요한 범위에서만 적용한다. Core와 Application이 외부 구현을 알지 못하도록 하되, 모든 내부 책임을 과도하게 추상화하지 않는다.

### Architecture-001 기본 계층 및 Port/Adapter 배치

| 계층·경계 | 책임 |
| --- | --- |
| CLI / Cobra | 사용자 명령·인자·출력·CLI configuration entrypoint |
| Workflow / Application Orchestrator | 전체 검사 workflow의 단계 조정과 Port 호출 |
| Core / Domain | 생태계 중립 domain model, 불변 규칙, 공통 계약 |
| Artifact Ports | Artifact source·acquisition·identity 관련 외부 경계 |
| Verification Ports | Verifier 외부 경계 |
| Inspection Ports | Scanner/Inspection 외부 경계 |
| Sandbox/Runtime Ports | 격리 backend 외부 경계 |
| Evidence / Result Ports | Evidence·Result·SBOM 저장 외부 경계 |
| Staging / Promotion Ports | Staging·Promoter·결과 반입 외부 경계 |
| Artifact Adapters | npm·PyPI·GitHub Releases 특수 로직 구현 |
| Verification Providers | Verifier 특수 로직 구현 |
| Inspection Providers | Scanner/Inspection 특수 로직 구현 |
| Sandbox Backends | 격리 backend 특수 로직 구현 |
| Evidence / Result Adapters | Evidence·Result·SBOM 및 저장소 특수 로직 구현 |
| Staging / Promotion Adapters | Staging·Promoter 및 반입 특수 로직 구현 |

### Architecture-001 기준 흐름

```text
CLI ───────────────→ Application / Workflow
                         │
                         ├────→ Core / Domain
                         ├────→ Policy ─────→ Core / Domain
                         └────→ Ports / Contracts ─────→ Core / Domain
                                      ↑
                                      │ implements
                         Adapters / Backends / Providers

Composition Root
  └─ 구체 구현 생성·연결
```

Application은 Core, Policy 및 외부 기능에 대한 Port/Contract에 의존하고, Core는 외부 구체 구현에 의존하지 않는다. Adapter/Backend/Provider는 해당 Port/Contract를 구현한다. Heliopause 본체는 하나의 배포 가능한 CLI 애플리케이션과 하나의 주 코드베이스를 유지하지만, 외부 도구와 backend는 Port/Adapter 뒤에서 별도 process·container 등으로 실행될 수 있으며 이를 microservice 분산 구조로 간주하지 않는다.

## 구현 영향

- Go package 구조는 CLI·Application·Core·Port·Adapter 경계를 반영해야 한다.
- Core package가 npm·PyPI·GitHub Releases·Cobra·scanner·container runtime package를 import하지 않도록 한다.
- Application Orchestrator는 구체 구현을 직접 생성하기보다 Port와 dependency injection 경계를 통해 연결한다.
- Artifact, Verification, Inspection, Sandbox/Runtime, Evidence/Result, Staging/Promotion의 Port 계약을 각 책임 경계에 맞게 정의한다.
- 외부 도구나 backend가 별도 process·container로 실행되더라도 내부 모듈 경계를 흐리지 않는다.
- Rust isolation component를 나중에 별도 프로세스·binary로 분리할 수 있도록 해당 Port를 실행 방식과 분리한다.
- microservice 분산·네트워크 RPC·서비스별 배포를 MVP의 기본 구조로 도입하지 않는다.
- 모듈 간 public contract, dependency direction, 금지 import는 후속 Architecture 결정과 architecture test에서 구체화한다.

## 누락 점검

- [x] Go 기반 Modular Monolith CLI
- [x] 하나의 배포 가능한 애플리케이션
- [x] 하나의 주 코드베이스와 배포 가능한 CLI 본체 내부 모듈 분리
- [x] 외부 도구·backend의 별도 process·container 실행 허용
- [x] MVP에서 microservice 분할 금지
- [x] 외부 생태계·검사 도구·격리 backend·저장소·Promotion과 Core 사이 Port/Adapter 경계
- [x] Core·Application의 npm·PyPI·GitHub Releases 직접 의존 금지
- [x] Core·Application의 Cobra·특정 scanner·container runtime 직접 의존 금지
- [x] 외부 의존성 격리에 필요한 범위에서만 Ports/Adapters 적용
- [x] 불필요한 추상화·복잡한 framework 도입 금지
- [x] 향후 Rust 등 저수준 isolation component 분리 가능성 유지
- [x] Rust 분리를 MVP 분산 구조의 전제로 삼지 않음
- [x] CLI → Application, Application → Core/Policy/Ports, Adapter/Backend/Provider → Port 구현 구조 기록

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-001 | 2026-08-10 | Go 기반 Modular Monolith CLI와 Core·Application 중심의 제한적 Ports/Adapters 구조를 채택하고 본체는 하나의 배포 가능한 CLI·주 코드베이스로 유지 | 개발·배포 복잡도를 통제하면서 외부 도구·backend의 별도 process/container 실행과 교체 가능성, 모듈 경계를 함께 확보하기 위해 | Go package 경계, Port 계약, Adapter 연결, Core 직접 의존 금지, 후속 architecture test가 설계 기준이 됨 |

## 사용자 원문: Architecture-002

```text
Architecture-002 결정안

Heliopause의 Modular Monolith는 CLI, Application/Workflow, Core/Domain, Artifact, Verification, Inspection, Sandbox/Runtime, Policy, Evidence/Result, Staging/Promotion의 최상위 책임 영역으로 구성한다.

각 모듈은 단일 책임과 명확한 경계를 가지며, 생태계별 로직·외부 도구·격리 구현이 Core와 Policy에 직접 침투하지 않도록 한다.

Static/Dynamic inspection, Artifact ecosystem adapter, SBOM/Result 등은 MVP에서는 관련 책임 영역 내부의 하위 구성으로 유지하고, 독립적인 확장·배포 필요성이 생기기 전까지 불필요하게 최상위 모듈로 분리하지 않는다. Evidence/Result와 Staging/Promotion은 서로 다른 권한·lifecycle·Port를 가지므로 별도의 최상위 책임 영역으로 분리한다.

Scanner/Inspection은 Finding과 Evidence를 생성하고, Verifier/Verification은 Verification Result와 Evidence를 생성한다. Sandbox/Runtime은 raw observation과 실행 상태를 생성하며 이를 Finding/Evidence로 직접 판정하지 않는다. 최종 Policy Decision은 Policy만 생성한다.
```

## 구조화된 결정

### Architecture-002 최상위 책임 영역

Heliopause Modular Monolith의 최상위 책임 영역은 다음과 같이 고정한다.

| 최상위 영역 | 책임 경계 | MVP 하위 구성 예시 |
| --- | --- | --- |
| CLI | 사용자 명령·입력·출력·실행 진입점 | Cobra command, terminal presentation |
| Application / Workflow | 전체 use case와 단계 orchestration | 검사 workflow, Port 호출, 상태 전이 조정 |
| Core / Domain | 생태계 중립 domain model과 불변 규칙 | Artifact identity, 검사 계약, domain invariant |
| Artifact | Artifact source와 생태계별 반입 책임 | npm·PyPI·GitHub Releases adapter |
| Verification | integrity·provenance 및 검증 근거 생성 | digest, signature, provenance, attestation verifier |
| Inspection | 정적·동적 관찰과 Finding/Evidence 생성 | static/dynamic inspection, scanner |
| Sandbox / Runtime | 격리 실행·관찰 backend 경계 | Linux quarantine backend, runtime adapter |
| Policy | Finding·Evidence·검증 결과를 종합한 최종 판정 | `ALLOW`, `MANUAL_REVIEW`, `BLOCK` |
| Evidence / Result | 검사 근거·결과 보존 및 결과 표현 | Evidence/SBOM/Result |
| Staging / Promotion | 검증 완료 Artifact 보관·승격·반입 | Staging/Promotion |

각 영역은 단일 책임과 명확한 경계를 가지며, 생태계별 로직·외부 도구·격리 구현은 Core와 Policy에 직접 침투하지 않는다.

### Architecture-002 하위 구성 원칙

MVP에서는 다음을 관련 최상위 책임 영역 내부의 하위 구성으로 유지한다.

- Static/Dynamic inspection → Inspection 내부
- Artifact ecosystem adapter → Artifact 내부
- SBOM/Result → Evidence / Result 내부
- Staging/Promotion → Staging / Promotion 내부
- Scanner → Inspection 내부
- Verifier → Verification 내부
- Sandbox backend → Sandbox / Runtime 내부

Evidence / Result와 Staging / Promotion은 보안 권한·목적·lifecycle·Port가 다르므로 최상위 영역으로 분리한다. 그 외 하위 구성은 독립적인 확장·배포 필요성이 생기기 전까지 불필요하게 최상위 모듈을 늘리지 않기 위한 원칙을 적용한다. 하위 구성의 독립 package 분리는 책임 경계를 훼손하지 않는 범위에서 허용한다.

### Architecture-002 Policy 판정 권한

Scanner/Inspection은 `Finding`과 `Evidence`를 생성하고, Verifier/Verification은 `Verification Result`와 `Evidence`를 생성한다. Sandbox/Runtime은 raw observation과 실행 상태만 생성하며 이를 `Finding`/`Evidence`로 직접 판정하지 않는다. 최종 `ALLOW` / `MANUAL_REVIEW` / `BLOCK` 판정은 Policy 모듈만 담당한다.

- Scanner는 Finding과 관찰 Evidence를 생성하지만 최종 Policy Decision을 반환하지 않는다.
- Verifier는 integrity·provenance 검증 결과와 Evidence를 생성하지만 반입 여부를 직접 결정하지 않는다.
- Sandbox/Runtime은 raw observation과 실행 상태를 생성하며 Finding/Evidence로 직접 판정하지 않는다.
- Policy는 관련 Finding·Evidence·검증 결과를 받아 최종 Policy Decision을 결정한다.

## Architecture-002 구현 영향

- Go package 구조의 최상위 책임 영역을 CLI, Application/Workflow, Core/Domain, Artifact, Verification, Inspection, Sandbox/Runtime, Policy, Evidence/Result, Staging/Promotion으로 정렬한다.
- Static/Dynamic inspection과 scanner를 Inspection 내부에 둔다.
- npm·PyPI·GitHub Releases adapter를 Artifact 내부 하위 구성으로 둔다.
- SBOM·Result는 Evidence/Result 내부 하위 구성으로 두고 Staging·Promotion은 Staging/Promotion 내부 하위 구성으로 둔다.
- Verification과 Inspection은 각각 검증 결과와 관찰 Finding/Evidence의 책임을 분리한다.
- Sandbox/Runtime은 격리 실행 구현을 담당하되 최종 Policy Decision을 생성하지 않는다.
- Policy package만 최종 `ALLOW` / `MANUAL_REVIEW` / `BLOCK` 타입 또는 결정을 생성하도록 경계를 둔다.
- Finding/Evidence 생성 계약과 최종 Policy Decision 계약을 분리한다.
- 독립 확장·배포 필요성이 생기기 전까지 각 하위 구성을 별도 최상위 서비스나 배포 단위로 분리하지 않는다.

## Architecture-002 누락 점검

- [x] CLI 최상위 책임 영역
- [x] Application/Workflow 최상위 책임 영역
- [x] Core/Domain 최상위 책임 영역
- [x] Artifact 최상위 책임 영역
- [x] Verification 최상위 책임 영역
- [x] Inspection 최상위 책임 영역
- [x] Sandbox/Runtime 최상위 책임 영역
- [x] Policy 최상위 책임 영역
- [x] Evidence/Result 최상위 책임 영역
- [x] Staging/Promotion 최상위 책임 영역
- [x] 각 모듈의 단일 책임과 명확한 경계
- [x] 생태계별 로직·외부 도구·격리 구현의 Core·Policy 직접 침투 금지
- [x] Static/Dynamic inspection의 Inspection 내부 하위 구성
- [x] Artifact ecosystem adapter의 Artifact 내부 하위 구성
- [x] SBOM/Result의 Evidence/Result 내부 하위 구성
- [x] Staging/Promotion의 Staging/Promotion 내부 하위 구성
- [x] 독립 확장·배포 필요 전 불필요한 최상위 분리 금지
- [x] Scanner/Inspection의 Finding/Evidence 생성 권한
- [x] Verifier/Verification의 Verification Result/Evidence 생성 권한
- [x] Sandbox/Runtime의 raw observation·실행 상태 생성과 Finding/Evidence 직접 판정 금지
- [x] Policy만 최종 ALLOW/MANUAL_REVIEW/BLOCK 판정 담당

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-002 | 2026-08-10 | Evidence/Result와 Staging/Promotion을 분리한 10개 최상위 책임 영역과 하위 구성 원칙을 확정하고, 관찰·검증 계층과 최종 Policy 판정 책임을 분리 | Evidence 보존과 실제 Artifact 반입은 권한·목적·lifecycle·Port가 다르므로 분리하고, 나머지 기능은 불필요한 최상위 분리를 억제하기 위해 | Go package 경계, Evidence/Result·Staging/Promotion Port, Finding/Evidence 계약, Policy 전용 최종 판정과 후속 architecture test가 설계 기준이 됨 |

## 사용자 원문: Architecture-003

```text
Architecture-003: Dependency Direction

Heliopause의 코드 의존성은 안정적인 Core와 공통 계약을 향하도록 구성하며, Core/Application이 외부 생태계나 구체 구현에 직접 의존하지 않는 Dependency Inversion 원칙을 적용한다.

`Core/Domain`은 가장 안쪽의 생태계·도구·OS 비종속 영역으로 유지하며 Cobra, npm, PyPI, GitHub Releases, scanner, container runtime, filesystem/database 구현 등 외부 기술에 직접 의존하지 않는다.

`Policy`는 Core의 정규화된 Verification Result, Finding, Evidence, Inspection 결과와 Capability, Execution Status 및 검사 한계를 입력으로 사용하여 최종 `ALLOW / MANUAL_REVIEW / BLOCK`을 결정하며, scanner·verifier·sandbox·artifact adapter를 직접 호출하지 않는다.

`Application/Workflow`는 Core, Policy 및 공통 Port/Contract에 의존하여 전체 workflow를 orchestration한다. npm adapter, 특정 scanner, Linux sandbox 등의 구체 구현을 직접 참조하지 않는다.

`CLI`는 Application use case를 호출하고 사용자 입력·출력을 담당하며 Artifact adapter, scanner, sandbox 또는 Policy 판단 로직을 직접 수행하지 않는다.

Artifact adapter, verifier/scanner provider, Sandbox backend, Evidence storage, Promotion implementation 등의 구체 구현은 자신이 구현하는 공통 Port/Contract와 필요한 Core type에 의존한다. Verification과 Inspection은 각각 정규화된 Verification Result/Evidence와 Finding/Evidence를 생성하며, Sandbox backend는 raw observation과 실행 상태만 생성한다. 이들 영역은 최종 Policy Decision을 생성하지 않는다.

Dynamic Inspection은 `Sandbox Port`를 사용할 수 있으나 특정 Linux/container 구현에 직접 의존하지 않는다.

Promotion은 Application이 `ALLOW` 판정을 확인한 후 호출하며, Promotion 자체가 Policy를 재판단하거나 검사 모듈을 직접 호출하지 않는다.

실제 adapter·backend·provider 구현을 Application에 연결하는 책임은 별도의 **Composition Root/Bootstrap 영역**에 두며, 이 영역에서만 구체 구현의 생성과 dependency wiring을 허용한다.

순환 의존성은 허용하지 않으며, 금지된 dependency direction은 이후 architecture test 및 import rule로 검증한다.

### 아주 짧게 요약하면

```
CLI ───────────────→ Application / Workflow
                         │
                         ├────→ Core / Domain
                         ├────→ Policy ─────→ Core / Domain
                         └────→ Ports / Contracts ─────→ Core / Domain
                                      ↑
                                      │ implements
                         Adapters / External Implementations

```

실제 구체 구현의 연결은 다음과 같이 별도의 Composition Root가 담당한다.

```
Composition Root / Bootstrap
    ├─ Application
    ├─ npm / PyPI / GitHub adapters
    ├─ Verifier / Scanner providers
    ├─ Sandbox backend
    ├─ Evidence storage
    └─ Promotion implementation

```

가장 중요한 금지 규칙은:

```
Core ─────────X──→ 외부 구현
Policy ───────X──→ Scanner / Adapter / Sandbox
Application ──X──→ 구체 Adapter / Scanner / Sandbox 구현
CLI ──────────X──→ 검사·판정 로직 직접 수행
Adapter ──────X──→ Policy 판단
Sandbox ──────X──→ 최종 Policy Decision

```
```

## 구조화된 결정

### Architecture-003 의존성 방향

코드 의존성은 안정적인 Core와 공통 계약을 향하도록 구성한다. Core/Application에는 Dependency Inversion 원칙을 적용하여 외부 생태계와 구체 구현이 안쪽 계층으로 침투하지 않게 한다.

| 영역 | 허용되는 의존 | 금지되는 의존·책임 |
| --- | --- | --- |
| Core / Domain | 생태계·도구·OS 비종속 domain type, 공통 계약 | Cobra, npm, PyPI, GitHub Releases, scanner, container runtime, filesystem/database 구현 |
| Policy | Core의 정규화된 Verification Result·Finding·Evidence·Inspection 결과와 Capability·Execution Status·검사 한계 | scanner·verifier·sandbox·artifact adapter 직접 호출, 검사 수행 |
| Application / Workflow | Core, Policy, 공통 Port/Contract | npm adapter, 특정 scanner, Linux sandbox 등 구체 구현 직접 참조 |
| CLI | Application use case, 사용자 입력·출력 | Artifact adapter·scanner·sandbox 직접 수행, Policy 판단 로직 직접 수행 |
| Adapter / Provider / Backend | 자신이 구현하는 Port/Contract, 필요한 Core type | Adapter가 Policy를 판단하거나 최종 Decision 생성 |
| Promotion | Application이 확인한 `ALLOW`, Promotion Port/Contract | Policy 재판단, 검사 모듈 직접 호출 |
| Dynamic Inspection | `Sandbox Port` | 특정 Linux/container 구현 직접 의존 |
| Composition Root / Bootstrap | 모든 구체 구현 생성과 dependency wiring | 업무 규칙·검사·Policy 판정의 직접 수행 |

### Architecture-003 Composition Root

실제 adapter·backend·provider 구현을 Application에 연결하는 책임은 별도의 Composition Root/Bootstrap 영역에 둔다. 구체 구현의 생성과 dependency wiring은 이 영역에서만 허용한다.

```text
Composition Root / Bootstrap
    ├─ Application
    ├─ npm / PyPI / GitHub adapters
    ├─ Verifier / Scanner providers
    ├─ Sandbox backend
    ├─ Evidence storage
    └─ Promotion implementation
```

Composition Root는 wiring을 담당하지만 Core·Policy·Application의 업무 규칙을 대신 수행하지 않는다.

### Architecture-003 순환 의존과 검증

- 순환 의존성은 허용하지 않는다.
- 금지된 dependency direction은 Go import rule과 architecture test로 검증한다.
- Core가 외부 구현을 import하는지, Policy가 검사 모듈을 직접 호출하는지, Application이 구체 Adapter/Scanner/Sandbox를 참조하는지 강제 검사한다.
- CLI가 검사·판정 로직을 직접 포함하는지, Adapter가 Policy를 판단하는지, Sandbox가 최종 Decision을 생성하는지 강제 검사한다.

## Architecture-003 구현 영향

- Core package를 외부 기술 import가 없는 가장 안쪽 계층으로 유지한다.
- Policy 입력은 정규화된 Verification Result·Finding·Evidence·Inspection 결과와 Capability·Execution Status·검사 한계로 구성하고 최종 Decision 생성 API를 Policy에만 둔다.
- Application은 Port/Contract를 통해서만 Artifact·Verification·Inspection·Evidence/Result·Staging/Promotion 기능을 호출한다.
- CLI는 Application use case를 호출하는 얇은 진입점으로 유지한다.
- Adapter·Provider·Backend는 공통 Port/Contract와 필요한 Core type을 구현·사용한다.
- Dynamic Inspection은 `Sandbox Port`를 사용하고 Linux/container backend의 구체 타입을 참조하지 않는다.
- Promotion은 Application이 확인한 `ALLOW` 뒤에 호출되며 자체 재판정·재검사를 하지 않는다.
- Composition Root/Bootstrap에 구체 구현 생성과 wiring을 모으고, 다른 모듈에서 직접 생성하지 않는다.
- 금지 import와 package dependency 방향을 architecture test로 고정한다.

## Architecture-003 누락 점검

- [x] 의존성 방향을 안정적인 Core·공통 계약을 향하도록 구성
- [x] Core/Application에 Dependency Inversion 적용
- [x] Core/Domain의 외부 생태계·도구·OS 비종속성
- [x] Core의 Cobra·npm·PyPI·GitHub Releases·scanner·container runtime·filesystem/database 직접 의존 금지
- [x] Policy가 정규화된 결과만 입력으로 사용
- [x] Policy만 최종 ALLOW/MANUAL_REVIEW/BLOCK 결정
- [x] Policy의 scanner·verifier·sandbox·artifact adapter 직접 호출 금지
- [x] Application의 Core·Policy·Port/Contract 의존
- [x] Application의 구체 Adapter·Scanner·Linux Sandbox 직접 참조 금지
- [x] CLI의 Application use case 호출 및 입력·출력 책임
- [x] CLI의 검사·판정 로직 직접 수행 금지
- [x] Adapter·Provider·Backend의 공통 Port/Contract·Core type 의존
- [x] Verification의 정규화된 Verification Result/Evidence 생성
- [x] Inspection의 Finding/Evidence 생성
- [x] Sandbox backend의 raw observation·실행 상태 생성
- [x] 이들 영역의 최종 Policy Decision 생성 금지
- [x] Dynamic Inspection의 Sandbox Port 사용과 구체 backend 직접 의존 금지
- [x] Promotion의 ALLOW 확인 후 호출 및 재판정·재검사 금지
- [x] Composition Root/Bootstrap의 구체 생성·dependency wiring 전담
- [x] 순환 의존 금지
- [x] architecture test·import rule 검증

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-003 | 2026-08-10 | Core·공통 계약을 향한 단방향 의존성과 Composition Root wiring을 확정하고, Policy만 최종 판정을 생성하도록 제한 | 외부 구현 교체 가능성과 핵심 규칙의 독립성을 유지하고 모듈 간 책임 침투·순환 의존을 방지하기 위해 | Go import rule, architecture test, Policy 전용 Decision, Port 기반 Application 연결과 Bootstrap 구성이 설계 기준이 됨 |

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

## 사용자 원문: Architecture-005

```text
### Architecture-005: Sandbox / Inspection Backend 구조

`Inspection`과 `Sandbox/Runtime`의 책임을 분리한다. Inspection은 어떤 정적·동적 검사를 수행하고 관찰 결과를 어떻게 Finding/Evidence로 해석할지를 담당하며, Sandbox는 격리 환경의 생성·준비·실행·관찰·종료를 담당한다. Sandbox 자체는 Artifact의 안전성이나 최종 Policy Decision을 판단하지 않는다.

Sandbox는 공통 `Sandbox Port` 뒤의 교체 가능한 backend로 구성한다. MVP에서는 Linux backend를 구현하되 Core/Application/Inspection이 특정 container runtime, namespace, seccomp 또는 기타 Linux 구현 기술에 직접 의존하지 않도록 한다. 향후 macOS·Windows 또는 Rust 기반 hardened isolation backend를 기존 계약을 통해 추가할 수 있도록 한다.

각 Sandbox 기반 격리·동적 검사 실행은 원칙적으로 독립적인 **ephemeral Sandbox Session**에서 수행한다. 세션은 생성·준비·Artifact 반입·설치/실행·**관찰 데이터 수집**·종료의 lifecycle을 가지며 검사 완료 후 폐기한다. timeout, resource-limit 초과, 강제 종료 또는 이상 상태가 발생한 세션은 재사용하지 않는다.

신뢰된 Heliopause control/observation 영역과 신뢰하지 않는 Artifact 실행 영역을 분리한다. Sandbox 내부의 Artifact는 Heliopause controller, 최종 Evidence 저장소, Policy 상태 등을 직접 수정할 수 없어야 하며, 가능한 관찰 정보는 실행 대상 외부의 trusted observer에서 수집한다.

Sandbox에는 실제 Host filesystem, Credential, 환경변수, 내부 network 및 Host service/process를 노출하지 않는다. 동적 행위 검증에 필요한 경우 독립적인 simulated filesystem, dummy credential, honeytoken 및 가짜 service를 제공한다.

Network는 기본적으로 외부·내부 실 network 접근을 차단하며, 동적 분석이 필요한 경우 통제된 DNS/HTTP 등의 관찰 환경을 제공할 수 있다. 구체적인 network isolation 및 simulation 기술은 후속 구현 결정에서 확정한다.

Sandbox backend는 process, filesystem, network, resource, credential/honeytoken 접근 등 **raw observation과 실행 상태**를 반환한다. Inspection 계층은 이를 Finding/Evidence로 정규화하고, 최종 `ALLOW / MANUAL_REVIEW / BLOCK` 판정은 Policy 계층에서만 수행한다.

구체적인 Linux isolation 기술, container runtime, syscall filtering 방식, resource-limit 수치 등은 Architecture-005에서 고정하지 않고 후속 backend/tooling 설계에서 결정한다.
```

## 구조화된 결정

### Architecture-005 Inspection과 Sandbox 책임 분리

| 영역 | 담당 | 담당하지 않는 것 |
| --- | --- | --- |
| Inspection | 수행할 정적·동적 검사 정의, raw observation 해석, `Finding`/`Evidence` 정규화 | 격리 환경 생성·실행 lifecycle, 최종 Policy Decision |
| Sandbox / Runtime | 격리 환경 생성·준비·Artifact 반입·설치/실행·관찰·종료, raw observation·실행 상태 반환 | Artifact 안전성 판단, 최종 `ALLOW`/`MANUAL_REVIEW`/`BLOCK` |
| Policy | 정규화된 Finding/Evidence와 검증 결과를 종합한 최종 판정 | Sandbox 제어, 직접 검사 수행 |

Inspection과 Sandbox는 `Sandbox Port`를 통해 연결하며 Inspection·Core·Application은 특정 container runtime, namespace, seccomp 등 Linux 기술에 직접 의존하지 않는다.

### Architecture-005 Sandbox Backend와 OS 확장

- Sandbox는 공통 `Sandbox Port` 뒤의 교체 가능한 backend로 구성한다.
- MVP에서는 Linux backend를 구현한다.
- 향후 macOS·Windows 또는 Rust 기반 hardened isolation backend를 기존 계약으로 추가할 수 있도록 한다.
- Core/Application/Inspection은 특정 Linux 구현 기술을 직접 참조하지 않는다.
- 구체적인 Linux isolation 기술, container runtime, syscall filtering 방식은 후속 backend/tooling 설계에서 결정한다.

### Architecture-005 Ephemeral Sandbox Session lifecycle

각 Sandbox 기반 격리·동적 검사 실행은 원칙적으로 독립적인 ephemeral Sandbox Session에서 수행한다.

```text
생성
  → 준비
  → Artifact 반입
  → 설치/실행
  → 관찰 데이터 수집
  → 종료
  → 폐기
```

Session은 검사 완료 후 폐기한다. timeout, resource-limit 초과, 강제 종료 또는 이상 상태가 발생한 Session은 재사용하지 않는다.

### Architecture-005 신뢰 경계와 관찰

- 신뢰된 Heliopause control/observation 영역과 신뢰하지 않는 Artifact 실행 영역을 분리한다.
- Sandbox 내부 Artifact가 Heliopause controller, 최종 Evidence 저장소, Policy 상태를 직접 수정할 수 없도록 한다.
- 가능한 관찰 정보는 실행 대상 외부의 trusted observer에서 수집한다.
- Sandbox backend는 process, filesystem, network, resource, credential/honeytoken 접근의 raw observation과 실행 상태를 반환한다.
- Inspection 계층은 raw observation을 Finding/Evidence로 정규화한다.
- 최종 Policy Decision은 Policy 계층에서만 생성한다.

### Architecture-005 Host 자산과 Network 정책

- 실제 Host filesystem, Credential, 환경변수, 내부 network, Host service/process를 Sandbox에 노출하지 않는다.
- 동적 검증이 필요한 경우 독립적인 simulated filesystem, dummy credential, honeytoken, 가짜 service를 제공한다.
- Network는 기본적으로 외부·내부 실 network 접근을 차단한다.
- 동적 분석에서 필요한 경우 통제된 DNS/HTTP 등의 관찰 환경을 제공할 수 있다.
- 구체적인 network isolation 및 simulation 기술은 후속 구현 결정으로 남긴다.

## Architecture-005 구현 영향

- `Sandbox Port`에 Session 생성·준비·반입·설치/실행·관찰·종료와 raw observation 반환 계약을 정의한다.
- Inspection Port/Provider는 Sandbox Port 결과를 받아 검사별 Finding/Evidence로 정규화한다.
- Sandbox backend는 최종 Policy Decision type이나 Policy 상태를 생성하지 않는다.
- Session identity, lifecycle 상태, 종료 사유, 폐기 여부와 재사용 금지 상태를 기록한다.
- timeout·resource-limit 초과·강제 종료·이상 상태를 실행 상태와 Evidence에 연결한다.
- trusted observer와 Artifact 실행 영역의 저장·통신 경계를 분리한다.
- simulated filesystem, dummy credential, honeytoken, 가짜 DNS/HTTP service fixture를 준비한다.
- Linux backend 구현 세부와 network·syscall·resource 수치는 후속 backend/tooling 결정으로 이관한다.
- macOS·Windows·Rust backend 추가가 기존 Sandbox Port를 깨뜨리지 않는지 contract test로 검증한다.

## Architecture-005 누락 점검

- [x] Inspection과 Sandbox/Runtime 책임 분리
- [x] Inspection의 검사 정의·관찰 해석·Finding/Evidence 정규화 책임
- [x] Sandbox의 생성·준비·실행·관찰·종료 책임
- [x] Sandbox 자체의 안전성·최종 Policy Decision 판단 금지
- [x] 공통 Sandbox Port와 교체 가능한 backend
- [x] MVP Linux backend
- [x] Core/Application/Inspection의 특정 Linux 기술 직접 의존 금지
- [x] 향후 macOS·Windows·Rust hardened backend 확장 가능성
- [x] 독립적인 ephemeral Sandbox Session 원칙
- [x] 생성·준비·반입·설치/실행·관찰 데이터 수집·종료 lifecycle
- [x] 검사 완료 후 Session 폐기
- [x] timeout·resource-limit 초과·강제 종료·이상 Session 재사용 금지
- [x] trusted control/observation 영역과 untrusted Artifact 실행 영역 분리
- [x] Artifact의 controller·Evidence 저장소·Policy 상태 직접 수정 금지
- [x] 외부 trusted observer 관찰 수집
- [x] 실제 Host filesystem·Credential·환경변수·내부 network·Host service/process 비노출
- [x] simulated filesystem·dummy credential·honeytoken·가짜 service 제공 가능
- [x] 외부·내부 실 network 기본 차단
- [x] 통제된 DNS/HTTP 관찰 환경 제공 가능
- [x] raw observation·실행 상태 반환
- [x] Inspection의 Finding/Evidence 정규화
- [x] 최종 Policy Decision은 Policy만 수행
- [x] 구체 isolation/runtime/syscall/resource 수치 후속 결정

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-005 | 2026-08-10 | Inspection과 Sandbox 책임을 분리하고 공통 Sandbox Port·ephemeral Session·trusted observer 경계를 채택하며 Linux 구현 세부는 후속 결정으로 유보 | 격리 실행 기술과 검사 해석을 교체 가능하게 분리하고 Artifact가 관찰·결과를 오염시키지 못하도록 하기 위해 | Session lifecycle, raw observation 계약, 모의 Host·통제 network, backend 교체성과 후속 tooling 결정이 설계 기준이 됨 |

## 사용자 원문: Architecture-006

```text
### Architecture-006: Verification / Inspection / Policy 책임 구조

Heliopause는 `Verification`, `Inspection`, 외부 Tool Provider와 `Policy`의 책임을 명확히 분리한다.

`Verification`은 Artifact의 identity, source, digest/checksum, Registry integrity, signature, provenance 및 attestation 등 **Artifact가 무엇이며 어디서 왔는지에 관한 신뢰 근거**를 검증하고 정규화된 Verification Result와 Evidence를 생성한다.

`Inspection`은 Artifact의 파일·코드·dependency·설치·실행 행위와 Sandbox에서 수집된 raw observation을 분석하여 **Artifact가 무엇을 포함하고 어떤 행위를 하는지**에 관한 Finding과 Evidence를 생성한다.

외부 verifier, scanner, vulnerability source 및 기타 security tool의 고유 verdict, exit code, severity, 출력 schema는 Provider/Adapter 내부에서 해석하고 공통 결과 모델로 정규화한다. 외부 도구의 verdict를 Heliopause의 최종 Policy Decision으로 직접 사용하지 않는다.

`Evidence`는 관찰·검증된 사실과 원본 근거를 나타내고, `Finding`은 해당 Evidence에 대한 검사 계층의 정규화된 보안 해석을 나타낸다. Finding은 가능한 경우 자신을 뒷받침하는 Evidence를 참조한다.

여러 verifier·scanner·정적·동적 검사 결과는 공통 Verification Result, Finding, Evidence 및 검사 실행 상태로 수집하며, 구체적인 점수 계산이나 vendor별 우선순위는 검사 계층에 두지 않는다.

기능의 지원·미지원(Capability)과 개별 검사의 성공·실패·미완료·미실행·사용 불가(Execution Status), 그리고 실제 보안 Finding을 서로 구분한다. 검사하지 못했거나 결과가 없는 상태를 안전한 결과로 간주하지 않는다.

최종 `ALLOW / MANUAL_REVIEW / BLOCK` 결정은 오직 `Policy` 계층에서 생성한다. Policy는 Verification Result, Finding, Evidence, Capability, 검사 실행 상태와 검사 한계를 종합하며, `WARN`은 독립적인 최종 Policy Decision이 아니라 Finding의 severity 또는 경고 수준으로 취급한다.

구체적인 severity 체계, 점수제 여부, Policy rule과 threshold는 Architecture-006에서 고정하지 않고 후속 Policy/Domain Model 설계에서 결정한다.

Verifier ───────────────┐

Scanner ────────────────┤

Static Inspection ──────┤

Dynamic Inspection ─────┼→ Result / Finding / Evidence

Sandbox Observation ────┘

                              ↓

                           Policy

                              ↓

             ALLOW / MANUAL_REVIEW / BLOCK
```

## 구조화된 결정

### Architecture-006 Verification·Inspection 책임 분리

| 영역 | 핵심 질문 | 생성하는 결과 |
| --- | --- | --- |
| Verification | Artifact가 무엇이며 어디서 왔는가? | 정규화된 Verification Result, Evidence |
| Inspection | Artifact가 무엇을 포함하고 어떤 행위를 하는가? | Finding, Evidence |
| External Tool Provider/Adapter | 외부 도구의 고유 결과를 어떻게 공통 모델로 바꾸는가? | 공통 Verification Result, Finding, Evidence, 실행 상태 |
| Sandbox Observation | 격리 실행 중 무엇이 관찰되었는가? | raw observation, 실행 상태; Inspection이 Finding/Evidence로 해석 |
| Policy | 모든 결과와 한계를 고려한 최종 반입 결정은 무엇인가? | `ALLOW`, `MANUAL_REVIEW`, `BLOCK` |

`Verification`은 identity·source·digest/checksum·Registry integrity·signature·provenance·attestation 등 Artifact의 신뢰 근거를 검증한다. `Inspection`은 파일·코드·dependency·설치·실행 행위와 Sandbox raw observation을 분석한다.

### Architecture-006 Evidence와 Finding 관계

- `Evidence`는 관찰·검증된 사실과 원본 근거를 나타낸다.
- `Finding`은 Evidence에 대한 검사 계층의 정규화된 보안 해석이다.
- Finding은 가능한 경우 자신을 뒷받침하는 Evidence를 참조한다.
- Evidence와 Finding은 최종 Policy Decision과 구분한다.
- 여러 verifier·scanner·정적·동적 검사 결과는 공통 Verification Result, Finding, Evidence와 검사 실행 상태로 수집한다.

### Architecture-006 외부 도구 결과 정규화

외부 verifier, scanner, vulnerability source와 기타 security tool의 verdict·exit code·severity·출력 schema는 Provider/Adapter 내부에서 해석한다. 외부 도구의 고유 verdict를 Heliopause 최종 Policy Decision으로 직접 사용하지 않는다.

구체적인 점수 계산이나 vendor별 우선순위는 검사 계층에 두지 않고, 정규화된 결과와 근거를 Policy 입력으로 전달한다.

### Architecture-006 실행 상태와 보안 Finding 구분

| 구분 | 예시 | 처리 원칙 |
| --- | --- | --- |
| Capability | 지원 / 미지원 | `Unsupported`는 안전으로 해석하지 않음 |
| Execution Status | 성공(`Completed`) / 실패 / 미완료 / 미실행·`Skipped` / `Unavailable` | 실제 보안 Finding과 별도 기록하고 사유 보존 |
| Security Result | `Finding` / `Evidence` | 정규화하여 Policy 입력으로 사용 |

검사 실행이 실패·미완료·미실행·사용 불가이거나 Capability가 미지원이어도 이를 `ALLOW`에 해당하는 정상 결과로 해석하지 않는다.

### Architecture-006 Policy 최종 결정

- 최종 `ALLOW` / `MANUAL_REVIEW` / `BLOCK` 결정은 Policy 계층에서만 생성한다.
- Policy는 Verification Result, Finding, Evidence, Capability, 검사 실행 상태와 검사 한계를 종합한다.
- `WARN`은 독립적인 최종 Policy Decision이 아니라 Finding의 severity 또는 경고 수준이다.
- 구체적인 severity 체계, 점수제 여부, Policy rule과 threshold는 후속 Policy/Domain Model 설계에서 결정한다.

## Architecture-006 구현 영향

- Verification Port/Result와 Inspection Finding/Evidence Port를 별도 계약으로 정의한다.
- Evidence에 원본 관찰·검증 근거를, Finding에 관련 Evidence 참조와 정규화된 해석을 기록한다.
- 외부 Provider/Adapter가 vendor별 verdict·exit code·severity·schema를 내부에서 공통 결과로 변환한다.
- 실행 상태 type을 Verification Result·Finding·Evidence와 별도로 유지하고 상태별 사유를 기록한다.
- Policy 입력 모델에 Capability·Execution Status·검사 한계를 포함하되, 검사 모듈이 Policy를 직접 호출하지 않도록 한다.
- `WARN` severity와 최종 `MANUAL_REVIEW` Decision을 서로 다른 모델로 표현한다.
- severity·score·threshold의 구체 schema는 후속 Policy/Domain Model 결정으로 이관한다.

## Architecture-006 누락 점검

- [x] Verification·Inspection·External Tool Provider·Policy 책임 분리
- [x] Verification의 identity·source·digest/checksum 검증
- [x] Registry integrity·signature·provenance·attestation 검증
- [x] 정규화된 Verification Result·Evidence 생성
- [x] Inspection의 파일·코드·dependency·설치·실행 행위 분석
- [x] Sandbox raw observation 분석과 Finding/Evidence 생성
- [x] 외부 도구 고유 verdict·exit code·severity·schema 내부 해석
- [x] 외부 verdict를 최종 Policy Decision으로 직접 사용하지 않음
- [x] Evidence를 사실·원본 근거로 정의
- [x] Finding을 Evidence에 대한 정규화된 보안 해석으로 정의
- [x] Finding의 Evidence 참조
- [x] 공통 결과 모델과 검사 실행 상태 수집
- [x] vendor별 점수·우선순위를 검사 계층에 두지 않음
- [x] Capability의 지원·미지원과 Execution Status의 성공·실패·미완료·미실행·사용 불가 구분
- [x] Security Result의 Finding/Evidence 구분
- [x] 검사 불가·결과 없음 상태의 안전 해석 금지
- [x] Policy만 최종 ALLOW/MANUAL_REVIEW/BLOCK 생성
- [x] Policy 입력에 Capability·검사 실행 상태·검사 한계 포함
- [x] WARN을 Finding severity/경고 수준으로 취급
- [x] severity·score·rule·threshold 후속 결정 유보

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-006 | 2026-08-10 | Verification·Inspection·Provider·Policy의 결과와 책임을 분리하고 Evidence/Finding·실행 상태를 정규화하며 Policy만 최종 판정을 생성 | 출처·무결성 근거와 내용·행위 관찰을 구분하고 외부 도구 결과를 교체 가능한 공통 모델로 통합하기 위해 | Verification/Inspection Port, Evidence 참조, 실행 상태 모델, Policy 입력·최종 Decision 경계와 후속 rule 설계가 기준이 됨 |

## 사용자 원문: Architecture-007

```text
### Architecture-007: Evidence / Result 저장 구조

Heliopause는 검사 근거와 결과를 보존하는 `Evidence Store`와 검증 완료 Artifact를 반입 전 보관하는 `Staging Area`를 논리적으로 분리한다. 두 영역은 향후 동일한 물리적 storage backend를 사용할 수 있으나 목적·권한·lifecycle은 독립적으로 관리한다.

Evidence는 Sandbox 및 외부 도구에서 수집된 **원본 관찰·검증 자료(Raw Evidence)**와 이를 해석·정규화한 Verification Result, Finding 및 기타 파생 결과를 구분하여 보존한다. Finding/Decision의 근거로 채택된 Raw Evidence는 정해진 retention 동안 보존하고, 대용량 원시 telemetry의 보존·요약 정책은 후속 결정으로 둔다.

모든 검사 자료는 하나의 추적 가능한 **Inspection Run**을 중심으로 연결한다. Inspection Run은 검사 대상 Artifact identity/digest, 실행 환경과 backend, 수행·미수행 검사, Verification Result, Evidence, Finding, SBOM, Policy Decision 및 관련 Manifest를 추적할 수 있어야 한다.

Evidence 수집과 저장은 신뢰된 Heliopause control/observer 영역에서 수행하며, 신뢰하지 않는 Sandbox 내부의 Artifact가 Evidence, Finding, Policy Decision 또는 검사 기록을 직접 생성·수정·삭제할 수 없도록 한다.

Evidence와 주요 결과물은 생성 이후 변경 여부를 확인할 수 있는 무결성 검증 수단을 가져야 한다. 구체적인 hashing, append-only storage, content-addressed storage 또는 signing 방식은 후속 storage/tooling 설계에서 결정한다.

Evidence, Finding, SBOM, Policy Decision 및 Verified Manifest는 자신이 어떤 Artifact와 dependency의 identity/digest에 대한 결과인지 추적할 수 있어야 하며, 검사된 Artifact와 실제 Staging/Promotion 대상의 동일성을 검증할 수 있어야 한다.

Raw Evidence, Verification Result, Finding, Policy Decision, SBOM, Verified Manifest, 사람용 결과와 machine-readable 결과는 각각의 의미와 책임을 유지하며 서로 참조 관계로 연결한다. 이를 하나의 불투명한 결과 blob으로 취급하지 않는다.

구체적인 storage engine, filesystem layout, database 사용 여부, serialization format, schema, retention 기간 및 cleanup 정책은 Architecture-007에서 고정하지 않고 이후 Domain Model·Storage/Tooling 설계에서 확정한다.

- 압축하자면 다음과 같다.

Artifact digest

      ↓

Inspection Run

      ↓

Raw Evidence → Finding → Policy Decision

      ↓

SBOM / Manifest / Report

그리고 Evidence Store ≠ Staging Area
```

## 구조화된 결정

### Architecture-007 Evidence Store와 Staging Area 분리

| 영역 | 목적 | 권한·lifecycle |
| --- | --- | --- |
| Evidence Store | 검사 근거와 결과 보존 | trusted control/observer가 수집·저장, 원본·파생 결과 추적 |
| Staging Area | 검증 완료 Artifact를 반입 전 보관 | 실행·외부 network 없이 승격 세트 보관·반출, Evidence Store와 독립 관리 |

두 영역은 향후 동일한 물리적 storage backend를 사용할 수 있지만 목적·권한·lifecycle은 논리적으로 독립이다. `Evidence Store ≠ Staging Area`를 기본 원칙으로 한다.

### Architecture-007 Evidence 계층

Evidence Store는 다음 자료를 구분하여 보존한다.

- Raw Evidence: Sandbox·외부 도구에서 수집된 원본 관찰·검증 자료
- Verification Result: Raw Evidence와 검증 결과를 정규화한 결과
- Finding: Evidence를 해석한 검사 계층의 보안 결과
- Policy Decision: Finding·Evidence·검사 상태를 종합한 최종 판정
- SBOM·Verified Manifest: 구성요소·승격 세트 및 digest 연결 결과
- 사람용 결과·machine-readable 결과: 동일한 검사 기록을 서로 다른 표현으로 제공하는 결과물

Finding/Decision의 근거로 채택된 Raw Evidence는 정해진 retention 동안 보존한다. 대용량 원시 telemetry의 보존·요약 정책은 후속 결정으로 두며, 각 자료는 고유 의미와 책임을 유지하고 참조 관계로 연결한다. 이를 하나의 불투명한 결과 blob으로 합치지 않는다.

### Architecture-007 Inspection Run 추적

모든 검사 자료는 추적 가능한 `Inspection Run`을 중심으로 연결한다.

```text
Artifact digest
      ↓
Inspection Run
      ↓
Raw Evidence → Finding → Policy Decision
      ↓
SBOM / Manifest / Report
```

`Inspection Run`은 최소한 다음을 추적할 수 있어야 한다.

- 검사 대상 Artifact identity/digest
- dependency identity/digest
- 실행 환경과 Sandbox backend
- 수행·미수행 검사와 사유
- Verification Result
- Raw Evidence와 파생 Evidence
- Finding
- SBOM
- Policy Decision
- 관련 Verified Manifest

### Architecture-007 신뢰 경계와 무결성

- Evidence 수집과 저장은 신뢰된 Heliopause control/observer 영역에서 수행한다.
- 신뢰하지 않는 Sandbox 내부 Artifact는 Evidence·Finding·Policy Decision·검사 기록을 직접 생성·수정·삭제할 수 없다.
- Evidence와 주요 결과물은 생성 이후 변경 여부를 확인할 수 있는 무결성 검증 수단을 가져야 한다.
- hashing, append-only storage, content-addressed storage, signing 중 구체 방식은 후속 storage/tooling 설계에서 결정한다.
- 검사된 Artifact와 실제 Staging/Promotion 대상의 동일성을 Artifact·dependency identity/digest 참조로 검증한다.

### Architecture-007 후속 결정 유보

다음 구현 세부는 Architecture-007에서 고정하지 않는다.

- storage engine
- filesystem layout
- database 사용 여부
- serialization format
- 세부 schema
- retention 기간
- cleanup 정책

위 항목은 이후 Domain Model·Storage/Tooling 설계에서 결정한다.

## Architecture-007 구현 영향

- `Inspection Run`을 모든 검사 자료의 추적 루트로 정의한다.
- Evidence Store와 Staging Area에 서로 다른 Port/권한/lifecycle 계약을 둔다.
- Raw Evidence와 Verification Result·Finding·Policy Decision·SBOM·Manifest·Report의 참조 관계를 모델링한다.
- 각 결과가 Artifact·dependency identity/digest를 참조하도록 한다.
- Evidence와 주요 결과물의 무결성 상태와 검증 근거를 기록한다.
- trusted control/observer만 Evidence Store에 쓰도록 하고 Sandbox 실행 영역에는 결과 저장 권한을 주지 않는다.
- Staging Area에는 검증 완료 세트의 Artifact·dependency와 관련 Manifest 참조만 허용하고 실행·외부 network 접근을 분리한다.
- 터미널용 요약과 machine-readable 결과가 동일한 Inspection Run·Policy Decision·digest를 참조하는지 검증한다.
- storage engine·layout·schema·retention·cleanup은 후속 Domain Model/Storage 결정에 맞춰 구현한다.

## Architecture-007 누락 점검

- [x] Evidence Store와 Staging Area 논리적 분리
- [x] 동일 물리 storage backend 사용 가능성 및 독립 목적·권한·lifecycle
- [x] Raw Evidence와 Verification Result·Finding·파생 결과 구분 보존
- [x] Finding/Decision 근거로 채택된 Raw Evidence의 정해진 retention 보존
- [x] 대용량 원시 telemetry의 보존·요약 정책 후속 결정
- [x] Inspection Run 중심 추적
- [x] Artifact identity/digest 추적
- [x] 실행 환경·backend 추적
- [x] 수행·미수행 검사와 사유 추적
- [x] Verification Result·Evidence·Finding·SBOM·Policy Decision·Manifest 추적
- [x] trusted control/observer 영역에서 Evidence 수집·저장
- [x] Sandbox Artifact의 Evidence·Finding·Policy·검사 기록 직접 변경 금지
- [x] 생성 이후 변경 확인을 위한 무결성 수단
- [x] Artifact·dependency identity/digest와 Staging/Promotion 대상 동일성 검증
- [x] 각 결과물의 의미·책임 유지와 참조 관계 연결
- [x] 불투명한 단일 결과 blob 취급 금지
- [x] storage engine·layout·database·serialization·schema·retention·cleanup 후속 결정 유보

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-007 | 2026-08-10 | Evidence Store와 Staging Area를 논리적으로 분리하고 Inspection Run 중심으로 Raw Evidence·Finding·Policy·SBOM·Manifest를 참조 연결하며 저장 기술은 후속 결정 | 원본 근거를 보존하고 검사 결과·실제 반입 대상의 추적성과 무결성을 확보하기 위해 | Evidence/Result·Staging Port, Inspection Run 모델, 무결성 계약, trusted writer와 후속 storage/schema 결정이 설계 기준이 됨 |

## 사용자 원문: Architecture-008

```text
### Architecture-008: Staging / Promotion 구조

Heliopause는 위험한 Artifact를 검사하는 Quarantine/Sandbox 영역과, 검증 완료 Artifact를 사용자 환경 반입 전 보관하는 `Staging Area`를 명확히 분리한다. Staging Area에는 최종 Policy Decision이 `ALLOW`인 Artifact만 승격할 수 있으며, `MANUAL_REVIEW` 또는 `BLOCK` 상태의 Artifact는 자동 승격·반입하지 않는다.

Package ecosystem의 경우 검사·설치 과정에서 실제로 사용된 Primary Artifact와 dependency를 하나의 **Verified Set**으로 관리한다. Verified Set의 각 구성요소는 identity, version, source, digest 및 관련 Inspection Run과 추적 가능하게 연결되어야 한다.

Staging Area는 검사 또는 실행 환경으로 사용하지 않는다. Staging 내부에서는 Artifact 실행 및 install script 실행을 허용하지 않으며, 외부 network 및 실제 Host Credential에 접근하지 못하도록 한다. 가능한 경우 Artifact를 immutable 또는 변경 탐지가 가능한 형태로 보존한다.

Artifact가 Quarantine에서 Staging으로 승격될 때와 Staging에서 실제 사용자 환경으로 반입·설치되기 직전에 digest를 재검증한다. 검사된 Artifact와 승격·반입 대상의 identity 또는 digest가 일치하지 않으면 Promotion을 중단하고 새로운 Artifact로 취급하여 필요한 검증 절차로 되돌린다.

Package 설치는 원칙적으로 검증 완료된 Verified Set만을 사용하여 수행한다. 실제 설치 과정에서 Verified Set에 포함되지 않은 새로운 dependency 또는 Artifact가 요구되면 이를 자동으로 신뢰하거나 설치하지 않고 acquisition/verification 흐름으로 되돌린다. 최소 검증으로 충분하지 않거나 위험·불확실성이 존재하면 full quarantine 대상으로 전환한다.

Package manager가 필요 없는 독립 binary/archive는 예외적으로 Staging Area를 반드시 거치지 않을 수 있다. 이 경우에도 검사된 대상과 실제 반입 대상의 identity/digest를 재검증한 후에만 사용자가 명시적으로 지정한 위치로 직접 반입할 수 있다.

여기서 직접 반입은 Staging Area만 생략한다는 의미이며, 신뢰하지 않는 Sandbox가 사용자 환경에 직접 접근하거나 쓰는 것을 허용한다는 의미가 아니다. 실제 반입은 trusted Promotion 경계를 통해 digest 재검증 후 수행한다.

Promotion 모듈은 Policy를 재판단하거나 Artifact를 다시 검사하지 않는다. Application/Workflow가 `ALLOW` 상태를 확인한 후 Promotion을 호출하며, Promotion은 검증 완료 Artifact의 동일성 확인, Staging 이동, Verified Set 적용 및 사용자 지정 환경 반입만 담당한다.

ALLOW Policy Decision, Verified Set 및 Verified Manifest의 기준 Inspection Run은 서로 일치해야 한다. 각 Artifact/dependency는 Verified Set/Manifest에 선언된 identity/digest와 정확히 일치하고 자신을 검증한 관련 Inspection Run까지 추적 가능해야 한다. 이 참조 관계를 벗어나 서로 다른 Decision·Verified Set·Manifest 또는 검증되지 않은 Artifact/dependency가 임의로 혼합되면 Promotion을 수행하지 않는다.

구체적인 Staging storage 방식, immutable 구현, local cache 구조, package manager별 offline/local install 방식 및 filesystem permission 세부사항은 Architecture-008에서 고정하지 않고 후속 Domain Model·Tooling·Implementation 설계에서 결정한다.

- 요약하면:

Quarantine

   ↓

Policy = ALLOW

   ↓

Verified Set

   ↓

Staging

   ↓

digest 재검증

   ↓

사용자 지정 환경

- 그리고 예외

Standalone binary/archive

   ↓

ALLOW

   ↓

digest 재검증

   ↓

사용자 지정 위치 직접 반입
```

## 구조화된 결정

### Architecture-008 Quarantine·Verified Set·Staging 분리

```text
Quarantine / Sandbox
      ↓ Policy = ALLOW
Verified Set
      ↓
Staging Area
      ↓ digest 재검증
사용자 지정 환경
```

- Quarantine/Sandbox는 위험 Artifact의 검사·설치·실행 영역이다.
- `Staging Area`는 검증 완료 Artifact를 사용자 환경 반입 전 보관하는 영역이며 검사·실행 환경이 아니다.
- Staging Area에는 최종 Policy Decision이 `ALLOW`인 Artifact만 승격할 수 있다.
- `MANUAL_REVIEW` 또는 `BLOCK` 상태는 자동 승격·반입하지 않는다.

### Architecture-008 Verified Set

Package ecosystem에서는 검사·설치 과정에서 실제 사용된 Primary Artifact와 dependency를 하나의 `Verified Set`으로 관리한다.

각 구성요소는 다음과 연결되어야 한다.

- identity
- version
- source
- digest
- 관련 `Inspection Run`

Package 설치는 원칙적으로 검증 완료된 Verified Set만 사용한다. Verified Set에 없는 새 dependency나 Artifact가 실제 설치 과정에서 요구되면 자동 신뢰·설치하지 않고 acquisition/verification 흐름으로 되돌린다. 최소 검증으로 충분하지 않거나 위험·불확실성이 있으면 full quarantine 대상으로 전환한다.

### Architecture-008 Staging 제약

- Staging 내부에서 Artifact 실행과 install script 실행을 허용하지 않는다.
- Staging을 검사 환경으로 사용하지 않는다.
- 외부 network와 실제 Host Credential에 접근하지 못하도록 한다.
- 가능한 경우 Artifact를 immutable 또는 변경 탐지가 가능한 형태로 보존한다.
- Staging storage 방식·immutable 구현·local cache·offline/local install·filesystem permission 세부사항은 후속 설계에서 결정한다.

### Architecture-008 동일성 재검증과 Promotion 중단

Artifact가 Quarantine에서 Staging으로 승격될 때와 Staging에서 사용자 환경으로 반입·설치되기 직전에 digest를 재검증한다.

검사된 Artifact와 승격·반입 대상의 identity 또는 digest가 일치하지 않으면:

1. Promotion을 중단한다.
2. 대상 Artifact를 새로운 Artifact로 취급한다.
3. 필요한 acquisition/verification/quarantine 절차로 되돌린다.

### Architecture-008 독립 binary/archive 예외

Package manager가 필요 없는 독립 binary/archive는 예외적으로 Staging Area를 반드시 거치지 않을 수 있다. 이 경우에도 검사 대상과 실제 반입 대상의 identity/digest를 재검증한 후에만 사용자가 명시한 위치로 직접 반입할 수 있다.

여기서 직접 반입은 Staging Area만 생략한다는 의미이며, 신뢰하지 않는 Sandbox가 사용자 환경에 직접 접근하거나 쓰는 것을 허용한다는 의미가 아니다. 실제 반입은 trusted Promotion 경계를 통해 digest 재검증 후 수행한다.

### Architecture-008 Promotion 책임

| 담당 | 책임 |
| --- | --- |
| Application/Workflow | `ALLOW` 상태 확인 후 Promotion 호출 |
| Promotion | 검증 완료 Artifact 동일성 확인, Staging 이동, Verified Set 적용, 사용자 지정 환경 반입 |
| Policy | 최종 Decision 생성 |
| Inspection/Verification | Artifact 검사·검증 |

Promotion은 Policy를 재판단하거나 Artifact를 다시 검사하지 않는다.

### Architecture-008 Promotion identity invariant

`ALLOW` Policy Decision, `Verified Set` 및 `Verified Manifest`의 기준 `Inspection Run`은 서로 일치해야 한다. 각 Artifact/dependency는 Verified Set/Manifest에 선언된 identity/digest와 정확히 일치하고 자신을 검증한 관련 `Inspection Run`까지 추적 가능해야 한다. 이 참조 관계를 벗어나 서로 다른 Decision·Verified Set·Manifest 또는 검증되지 않은 Artifact/dependency가 임의로 혼합되면 Promotion을 수행하지 않는다.

## Architecture-008 구현 영향

- Quarantine/Sandbox와 Staging Area에 별도 Port·권한·lifecycle을 둔다.
- Verified Set 모델에 Primary Artifact·dependency identity/version/source/digest와 Inspection Run 참조를 포함한다.
- Staging 승격 전과 사용자 환경 반입 직전에 digest 재검증 단계를 workflow에 고정한다.
- 동일성 불일치 시 Promotion 중단과 재검증 workflow 전환을 구현한다.
- Promotion 입력의 Policy Decision·Verified Set·Verified Manifest의 기준 Inspection Run이 서로 일치하는지, 각 Artifact/dependency가 Verified Set/Manifest의 선언 identity/digest와 정확히 일치하고 자신을 검증한 관련 Inspection Run까지 추적 가능한지 검증한다. 참조 관계를 벗어난 Decision·Verified Set·Manifest 또는 검증되지 않은 Artifact/dependency가 혼합되면 Promotion을 중단한다.
- Staging용 API에는 실행·install script·외부 network·실제 Host Credential 접근이 없도록 한다.
- Package manager 설치는 원칙적으로 Verified Set만 사용하며, Verified Set 밖의 dependency를 요청하면 acquisition/verification으로 되돌리고 필요 시 full quarantine으로 전환한다.
- 독립 binary/archive 직접 반입은 D-011과 연결된 예외 경로로 구현하고 digest 재검증을 필수화한다.
- 독립 binary/archive 직접 반입은 Staging Area만 생략하며, 신뢰하지 않는 Sandbox의 사용자 환경 직접 접근·쓰기를 허용하지 않고 trusted Promotion 경계를 통해 digest 재검증 후 수행한다.
- Promotion 구현은 Policy·Inspection 호출 권한을 갖지 않고 Application이 전달한 검증 완료 입력만 사용한다.
- storage·immutable·cache·offline install·permission 세부는 후속 Domain Model·Tooling·Implementation 설계와 연결한다.

## Architecture-008 누락 점검

- [x] Quarantine/Sandbox와 Staging Area 명확한 분리
- [x] Staging에는 `ALLOW` Artifact만 승격
- [x] `MANUAL_REVIEW`·`BLOCK` 자동 승격·반입 금지
- [x] Primary Artifact와 dependency의 Verified Set 관리
- [x] Verified Set 구성요소의 identity·version·source·digest·Inspection Run 연결
- [x] Staging을 검사·실행 환경으로 사용하지 않음
- [x] Staging 내 Artifact·install script 실행 금지
- [x] 외부 network·실제 Host Credential 비노출
- [x] immutable 또는 변경 탐지 가능한 보존
- [x] Quarantine→Staging 승격 시 digest 재검증
- [x] Staging→사용자 환경 반입 직전 digest 재검증
- [x] identity/digest 불일치 시 Promotion 중단과 재검증 전환
- [x] Package 설치의 원칙적 Verified Set 사용
- [x] Verified Set 밖의 새 dependency·Artifact 자동 신뢰·설치 금지
- [x] 새 dependency의 acquisition/verification 및 필요 시 full quarantine 전환
- [x] 독립 binary/archive의 Staging 우회 예외와 digest 재검증
- [x] 직접 반입이 Staging만 생략하고 trusted Promotion을 통해 수행됨
- [x] Application/Workflow의 ALLOW 확인 후 Promotion 호출
- [x] Promotion의 동일성 확인·Staging·Verified Set·반입 책임
- [x] Promotion의 Policy 재판단·재검사 금지
- [x] ALLOW Decision·Verified Set·Verified Manifest의 기준 Inspection Run 일치
- [x] Artifact/dependency의 Verified Set/Manifest 선언 identity/digest 일치 및 자신을 검증한 Inspection Run 추적
- [x] 참조 관계를 벗어난 Decision·Verified Set·Manifest 또는 검증되지 않은 Artifact/dependency 혼합 시 Promotion 금지
- [x] Staging storage·immutable·cache·offline install·permission 후속 결정

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-008 | 2026-08-10 | Quarantine·Verified Set·Staging·Promotion을 분리하고 승격/반입 전 digest 재검증과 독립 binary/archive 예외를 확정 | 검증된 실제 Artifact 세트의 동일성을 보장하고 Staging을 비실행·비네트워크 보관 영역으로 유지하기 위해 | Verified Set 모델, 이중 digest 검증, Promotion 책임 경계, dependency 재검역과 후속 storage 결정이 설계 기준이 됨 |
