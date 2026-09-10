# Domain Model and Interface Contracts

Heliopause의 Domain Concept, 상태 모델과 Port Contract를 정의한다.

## Documents

| File | Decisions | Read when |
| --- | --- | --- |
| [01-core-concepts.md](./01-core-concepts.md) | Domain-001 | 전체 Domain Concept와 관계 작업 |
| [02-artifact-identity.md](./02-artifact-identity.md) | Domain-002 | Artifact reference·identity·digest 작업 |
| [03-inspection-run-and-status.md](./03-inspection-run-and-status.md) | Domain-003~004 | Run lifecycle과 상태 축 작업 |
| [04-verification-evidence-finding.md](./04-verification-evidence-finding.md) | Domain-005 | Verification·Evidence·Finding 작업 |
| [05-dependency-verified-set.md](./05-dependency-verified-set.md) | Domain-006 | dependency·Verified Set·Manifest 작업 |
| [06-operation-request-context.md](./06-operation-request-context.md) | Domain-007 | Operation Request·Install Context 작업 |
| [07-contract-artifact-port.md](./07-contract-artifact-port.md) | Contract-001 | Artifact Port 작업 |
| [08-contract-inspection-sandbox-policy.md](./08-contract-inspection-sandbox-policy.md) | Contract-002 | Verification·Inspection·Sandbox·Policy 계약 작업 |
| [09-contract-evidence-staging-promotion.md](./09-contract-evidence-staging-promotion.md) | Contract-003 | Evidence·Staging·Promotion 계약 작업 |
| [10-m1-workflow-contract.md](./10-m1-workflow-contract.md) | M1 Entry | fake inspect Domain·Port·Policy·Application·CLI 계약 구현 |
| [11-m2-npm-static-contract.md](./11-m2-npm-static-contract.md) | M2 Entry | npm static inspect Adapter·Intake·Integrity·Policy·fixture 계약 구현 |
| [12-m3-linux-dynamic-contract.md](./12-m3-linux-dynamic-contract.md) | M3 Entry | Linux dynamic inspect runtime·observation·Policy 계약 구현 |
| [13-m4-npm-install-promotion-contract.md](./13-m4-npm-install-promotion-contract.md) | M4 Entry | npm dependency Verified Set·Staging·offline Promotion 계약 구현 |
| [14-m5-pypi-pip-contract.md](./14-m5-pypi-pip-contract.md) | M5 Entry | PyPI/pip wheel·sdist·isolated build·offline Promotion 계약 구현 |
| [15-m6-github-releases-contract.md](./15-m6-github-releases-contract.md) | M6 Entry | GitHub Releases exact asset·integrity·inspection·standalone Promotion 계약 구현 |

## Rule

- 이 README는 navigation 전용이다.
- 실제 결정의 canonical source는 각 상세 문서다.
- 작업에 필요한 상세 문서만 읽는다.
