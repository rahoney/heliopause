# Heliopause Artifact Airlock Project Decisions

이 문서는 무엇이 결정됐고 무엇이 남았는지를 기록하는 상태 색인이다. 상세 설계와 사용자 원문은 [Documentation Guide](./docs/README.md)를 통해 canonical leaf 문서에서 확인한다.

## Project

| Item | Status | Decision |
| --- | --- | --- |
| Project name | Complete | Heliopause Artifact Airlock |
| CLI name | Complete | `helox` |
| Repository | Complete | `github.com/rahoney/heliopause` |
| Go module path | Complete | `github.com/rahoney/heliopause` |
| Purpose | Complete | 외부 Artifact를 통제된 환경에서 검사하고 허용된 정확한 Artifact만 목표 환경으로 반입 |
| Language | Complete | Go |
| CLI framework | Complete | Cobra |

## Decision Status

| Area | Decisions | Status | Detail |
| --- | --- | --- | --- |
| Threat Model | D-001~014 | Complete | [docs/threat-model/README.md](./docs/threat-model/README.md) |
| MVP Scope | MVP-001~009 | Complete | [docs/mvp-scope/README.md](./docs/mvp-scope/README.md) |
| Architecture | Architecture-001~008 | Complete | [docs/architecture/README.md](./docs/architecture/README.md) |
| User Journey + CLI IA | User-Journey-001~008 | Complete | [docs/user-journey-cli-ia/README.md](./docs/user-journey-cli-ia/README.md) |
| Domain Model | Domain-001~007 | Complete | [docs/domain-model/README.md](./docs/domain-model/README.md) |
| Interface Contracts | Contract-001~003 | Complete | [docs/domain-model/README.md](./docs/domain-model/README.md) |
| Documentation Structure | Navigation migration | Complete | [docs/README.md](./docs/README.md) |
| Directory Structure | Step 8 | Complete | [docs/engineering/01-directory-structure.md](./docs/engineering/01-directory-structure.md) |
| Coding / Security Rules | Step 9 | Complete | [docs/engineering/02-coding-security-rules.md](./docs/engineering/02-coding-security-rules.md) |
| Quality Toolchain | Step 10 | Complete | [docs/engineering/03-quality-toolchain.md](./docs/engineering/03-quality-toolchain.md) |
| CI + Quality Gate | Step 11 | Complete | [docs/engineering/04-ci-quality-gate.md](./docs/engineering/04-ci-quality-gate.md) |
| Milestones | Step 12 | Complete | [docs/planning/01-milestones.md](./docs/planning/01-milestones.md) |
| Current Work Queue | Step 13 | Complete | [docs/planning/02-current-work-queue.md](./docs/planning/02-current-work-queue.md) |
| Go·Tool·CI Identity | M0-002 | Complete | [Quality Toolchain](./docs/engineering/03-quality-toolchain.md), [CI Quality Gate](./docs/engineering/04-ci-quality-gate.md) |
| Go Module·Process Bootstrap | M0-003 | Complete | [Current Work Queue](./docs/planning/02-current-work-queue.md) |

## Current Stage

Step 14 — Implementation: M0 `IN_PROGRESS`

Documentation hierarchy and task routing migration: Complete

Next: M0-004 — Canonical check runner foundation

## Remaining Stages

| Step | Stage | Status | Outcome |
| --- | --- | --- | --- |
| 8 | Directory Structure | Complete | 구현 디렉터리, package 경계와 파일 배치 규칙 |
| 9 | Coding / Security Rules | Complete | 코드 작성·의존성·비밀값·오류 처리·보안 구현 규칙 |
| 10 | Formatter / Linter / Type Check / Test / Security Scan | Complete | 로컬 deterministic 검증 도구와 실행 명령 |
| 11 | CI + Quality Gate | Complete | CI workflow와 merge 차단 기준 |
| 12 | Milestones | Complete | 구현 단계, 의존 관계와 완료 조건 |
| 13 | Current Work Queue | Complete | 실행 가능한 현재 작업과 우선순위 |
| 14 | Implementation | In Progress | 라우팅된 설계 문서를 기준으로 점진적 구현 |

## Open Decisions

아래 항목은 후속 단계에서 확정한다. 상세 결정을 이 표에 작성하지 않고 해당 단계의 canonical 문서로 이동한 뒤 상태와 링크만 갱신한다.

| Item | Planned stage or context |
| --- | --- |
| 구체 runtime/container 기술과 배포 형태 | M3 entry decision 또는 backend 구현 직전 |
| 설정 형식과 schema | 관련 구현 직전 상세 설계 |
| Adapter 등록과 plugin 확장 방식 | Step 9 또는 구현 직전 ADR |
| 구체 동시성·resource limit 값과 backend별 제어 | 관련 구현 직전 상세 설계 |
| Evidence storage·retention·cleanup | 구현 직전 상세 설계 |
| Policy rule·version·reason code | 구현 직전 상세 설계 |
| repository-wide Actions/branch rule 적용 범위 | M0-007; 명시적 owner 승인 필요 |
| 라이선스·상표·배포 정책 | 배포 전 |
| MCP 제공 여부 | CLI 안정화 후 |

## Documentation Rules

- 이 문서는 결정 상태와 open decision만 기록한다.
- 작업별 문서 선택은 [docs/README.md](./docs/README.md)가 담당한다.
- 실제 결정은 관련 canonical leaf 문서에 한 번만 기록한다.
- 완료된 상세 결정을 이 문서에 장문으로 재서술하지 않는다.
- 문서 이동 시 상태 링크와 Documentation Guide를 같은 변경에서 갱신한다.
