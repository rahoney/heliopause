# Documentation Guide

이 문서는 Heliopause 작업에 필요한 canonical 설계 문서를 찾기 위한 라우터다. 설계 내용 자체는 각 주제의 상세 문서에만 기록한다.

## Areas

| Area | Purpose | Index |
| --- | --- | --- |
| Threat Model | 위협, 보호 자산, 신뢰 경계와 fail-closed 원칙 | [threat-model/README.md](./threat-model/README.md) |
| MVP Scope | MVP 지원 범위, 검사·검증과 완료 기준 | [mvp-scope/README.md](./mvp-scope/README.md) |
| Architecture | 책임, 의존성, Port/Adapter와 신뢰 경계 | [architecture/README.md](./architecture/README.md) |
| User Journey + CLI IA | 사용자 흐름, CLI 구조와 실행 경험 | [user-journey-cli-ia/README.md](./user-journey-cli-ia/README.md) |
| Domain Model | Domain Concept와 Interface Contract | [domain-model/README.md](./domain-model/README.md) |
| Engineering | 구현 디렉터리, 코딩·보안 규칙과 품질 자동화 | [engineering/README.md](./engineering/README.md) |

## Task Routing

### 디렉터리 또는 Go package 구조

Read:

- [Step 8 — Directory Structure](./engineering/01-directory-structure.md)
- [Architecture — Foundation and Dependencies](./architecture/01-foundation-and-dependencies.md)
- 구현하려는 책임에 해당하는 아래 task route

### Artifact Adapter 또는 dependency resolution

Read:

- [Architecture — Adapters and Providers](./architecture/02-adapters-and-providers.md)
- [Domain — Artifact Identity](./domain-model/02-artifact-identity.md)
- [Domain — Dependency and Verified Set](./domain-model/05-dependency-verified-set.md)
- [Contract — Artifact Port](./domain-model/07-contract-artifact-port.md)
- [MVP — Ecosystems, Platforms, and Artifacts](./mvp-scope/01-ecosystems-platforms-artifacts.md)

### Verification 또는 static inspection

Read:

- [MVP — Inspection and Verification](./mvp-scope/02-inspection-and-verification.md)
- [Architecture — Inspection, Sandbox, and Policy](./architecture/03-inspection-sandbox-policy.md)
- [Domain — Verification, Evidence, and Finding](./domain-model/04-verification-evidence-finding.md)
- [Contract — Inspection, Sandbox, and Policy](./domain-model/08-contract-inspection-sandbox-policy.md)

### Sandbox 또는 dynamic inspection

Read:

- [Threat Model — Isolation and Inspection](./threat-model/02-isolation-and-inspection.md)
- [Architecture — Inspection, Sandbox, and Policy](./architecture/03-inspection-sandbox-policy.md)
- [Contract — Inspection, Sandbox, and Policy](./domain-model/08-contract-inspection-sandbox-policy.md)

### Policy 또는 상태 모델

Read:

- [Threat Model — Fail-Closed and Promotion](./threat-model/03-fail-closed-and-promotion.md)
- [Domain — Inspection Run and Status](./domain-model/03-inspection-run-and-status.md)
- [Domain — Verification, Evidence, and Finding](./domain-model/04-verification-evidence-finding.md)
- [Contract — Inspection, Sandbox, and Policy](./domain-model/08-contract-inspection-sandbox-policy.md)

### Evidence, Result 또는 SBOM

Read:

- [Architecture — Evidence, Staging, and Promotion](./architecture/04-evidence-staging-promotion.md)
- [Domain — Verification, Evidence, and Finding](./domain-model/04-verification-evidence-finding.md)
- [Contract — Evidence, Staging, and Promotion](./domain-model/09-contract-evidence-staging-promotion.md)
- [MVP — Results, Policy, and Completion](./mvp-scope/03-results-policy-and-completion.md)

### Staging 또는 Promotion

Read:

- [Threat Model — Fail-Closed and Promotion](./threat-model/03-fail-closed-and-promotion.md)
- [Architecture — Evidence, Staging, and Promotion](./architecture/04-evidence-staging-promotion.md)
- [Domain — Dependency and Verified Set](./domain-model/05-dependency-verified-set.md)
- [Domain — Operation Request and Context](./domain-model/06-operation-request-context.md)
- [Contract — Evidence, Staging, and Promotion](./domain-model/09-contract-evidence-staging-promotion.md)

### CLI 또는 사용자 흐름

Read:

- [User Journey — Journey and Input](./user-journey-cli-ia/01-journey-and-input.md)
- [User Journey — CLI Structure](./user-journey-cli-ia/02-cli-structure.md)
- [User Journey — Execution and Interaction](./user-journey-cli-ia/03-execution-and-interaction.md)
- [Domain — Operation Request and Context](./domain-model/06-operation-request-context.md)

### Runtime, 배포 또는 trusted tooling

Read:

- [Threat Model — Trusted Tooling and Evidence](./threat-model/04-trusted-tooling-and-evidence.md)
- [MVP — Ecosystems, Platforms, and Artifacts](./mvp-scope/01-ecosystems-platforms-artifacts.md)
- [Architecture — Foundation and Dependencies](./architecture/01-foundation-and-dependencies.md)
- [Architecture — Inspection, Sandbox, and Policy](./architecture/03-inspection-sandbox-policy.md)

### Coding 또는 security rule

Read:

- [Step 9 — Coding / Security Rules](./engineering/02-coding-security-rules.md)
- [Step 8 — Directory Structure](./engineering/01-directory-structure.md)
- 구현하려는 책임에 해당하는 위 task route

### Quality toolchain 또는 CI

Read:

- [Engineering Index](./engineering/README.md)
- [Step 9 — Coding / Security Rules](./engineering/02-coding-security-rules.md)
- 해당 단계에서 생성된 Engineering 상세 문서

## Rules

- 현재 작업에 라우팅된 상세 문서만 우선 읽는다.
- 라우팅이 없으면 관련 주제의 `README.md`에서 시작한다.
- 실제 결정의 canonical source는 leaf 문서다.
- 이 파일과 주제 README에는 상세 결정을 복제하지 않는다.
- 문서를 추가·이동하면 관련 주제 README와 이 라우팅표를 같은 변경에서 갱신한다.
- `docs/` 전체를 기본 context로 한 번에 로드하지 않는다.
