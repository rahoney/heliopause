# Architecture — Foundation and Dependencies

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
