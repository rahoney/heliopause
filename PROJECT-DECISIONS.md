# Heliopause Artifact Airlock Project Decisions

이 문서는 확정된 설계 결정과 canonical 문서 링크를 기록하는 결정 색인이다. 상세 설계와 사용자 원문은 [Documentation Guide](./docs/README.md)를 통해 canonical leaf 문서에서 확인한다. 현재 milestone/work item 진행 상태는 [Current Work Queue](./docs/planning/02-current-work-queue.md)가 유일한 canonical owner다.

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
| Threat Model | D-001~015 | Complete | [docs/threat-model/README.md](./docs/threat-model/README.md) |
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
| Canonical Check Runner | M0-004 | Complete | [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Static Analysis·Quick Profile | M0-005 | Complete | [Quality Toolchain](./docs/engineering/03-quality-toolchain.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Quick·Docs·Required CI | M0-006 | Complete | [CI Quality Gate](./docs/engineering/04-ci-quality-gate.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Minimum Go·macOS·Repository Gate 검증 | M0-007 | Complete | [CI Quality Gate](./docs/engineering/04-ci-quality-gate.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M0 Qualification | M0-008 | Complete | [M0 Qualification](./docs/planning/03-m0-qualification.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M1 Domain Workflow Entry | M1-001 | Complete | [M1 Workflow Contract](./docs/domain-model/10-m1-workflow-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M1 Domain Identity·Run State | M1-002 | Complete | [M1 Workflow Contract](./docs/domain-model/10-m1-workflow-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M1 Check Result·Port·Fake | M1-003 | Complete | [M1 Workflow Contract](./docs/domain-model/10-m1-workflow-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M1 Inspect Orchestration·Policy v1 | M1-004 | Complete | [M1 Workflow Contract](./docs/domain-model/10-m1-workflow-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M1 CLI Result Contract | M1-005 | Complete | [M1 Workflow Contract](./docs/domain-model/10-m1-workflow-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M1 Qualification | M1-006 | Complete | [M1 Qualification](./docs/planning/04-m1-qualification.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M2 npm Static Inspect Entry | M2-001 | Complete | [M2 npm Static Contract](./docs/domain-model/11-m2-npm-static-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M2 npm Resolve Foundation | M2-002 | Complete | [M2 npm Static Contract](./docs/domain-model/11-m2-npm-static-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M2 npm Controlled Intake·Integrity | M2-003 | Complete | [M2 npm Static Contract](./docs/domain-model/11-m2-npm-static-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M2 npm Static Archive·Evidence | M2-004 | Complete | [M2 npm Static Contract](./docs/domain-model/11-m2-npm-static-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M2 npm Policy·CLI·Vertical | M2-005 | Complete | [M2 npm Static Contract](./docs/domain-model/11-m2-npm-static-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M2 Qualification·M3 Handoff | M2-006 | Complete | [M2 Qualification](./docs/planning/05-m2-qualification.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M3 Linux Dynamic Inspect Entry | M3-001 | Complete | [M3 Linux Dynamic Contract](./docs/domain-model/12-m3-linux-dynamic-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M3 gVisor Runtime Lock·Probe | M3-002 | Complete | [M3 Linux Dynamic Contract](./docs/domain-model/12-m3-linux-dynamic-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M3 Sandbox Session·Observation Domain | M3-003 | Complete | [M3 Linux Dynamic Contract](./docs/domain-model/12-m3-linux-dynamic-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M3 gVisor Sandbox Lifecycle | M3-004 | Complete | [M3 Linux Dynamic Contract](./docs/domain-model/12-m3-linux-dynamic-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M3 gVisor Raw Observation Collector | M3-005 | Complete | [M3 Linux Dynamic Contract](./docs/domain-model/12-m3-linux-dynamic-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M3 Dynamic Observation Normalization | M3-006 | Complete | [M3 Linux Dynamic Contract](./docs/domain-model/12-m3-linux-dynamic-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M3 Policy·npm Inspect Wiring | M3-007 | Complete | [M3 Linux Dynamic Contract](./docs/domain-model/12-m3-linux-dynamic-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M3 Qualification·M4 Handoff | M3-008 | Complete | [M3 Qualification](./docs/planning/06-m3-qualification.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M4 npm Install·Promotion Entry | M4-001 | Complete | [M4 npm Install and Promotion Contract](./docs/domain-model/13-m4-npm-install-promotion-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M4 Install Context·Resolver Boundary | M4-002 | Complete | [M4 npm Install and Promotion Contract](./docs/domain-model/13-m4-npm-install-promotion-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M4 Recursive Inspection·Set Policy | M4-003 | Complete | [M4 npm Install and Promotion Contract](./docs/domain-model/13-m4-npm-install-promotion-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M4 Verified Manifest·SBOM·Trusted Staging | M4-004 | Complete | [M4 npm Install and Promotion Contract](./docs/domain-model/13-m4-npm-install-promotion-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M4 Offline npm Promotion·Atomic Target | M4-005 | Complete | [M4 npm Install and Promotion Contract](./docs/domain-model/13-m4-npm-install-promotion-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M4 Qualification·M5 Handoff | M4-006 | Complete | [M4 Qualification](./docs/planning/07-m4-qualification.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M5 PyPI/pip Expansion Entry | M5-001 | Complete | [M5 PyPI/pip Expansion Contract](./docs/domain-model/14-m5-pypi-pip-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M5 Python/pip Runtime·Reference Foundation | M5-002 | Complete | [M5 PyPI/pip Expansion Contract](./docs/domain-model/14-m5-pypi-pip-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M5 isolated PyPI Resolver·Target Tag·Egress Policy | M5-003 | Complete | [M5 PyPI/pip Expansion Contract](./docs/domain-model/14-m5-pypi-pip-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M5 Wheel Intake·Integrity·Static Inspection | M5-004 | Complete | [M5 PyPI/pip Expansion Contract](./docs/domain-model/14-m5-pypi-pip-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M5 Python Dynamic Inspection·PEP 517 sdist Build | M5-005 | Complete | [M5 PyPI/pip Expansion Contract](./docs/domain-model/14-m5-pypi-pip-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |

| M5 Qualification·npm Regression·M6 Handoff | M5-007 | Complete | [M5 Qualification](./docs/planning/08-m5-qualification.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M6 GitHub Releases Standalone Entry | M6-001 | Complete | [M6 GitHub Releases Standalone Contract](./docs/domain-model/15-m6-github-releases-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M6 Exact Public Input·API Boundary | M6-002 | Complete | [M6 GitHub Releases Standalone Contract](./docs/domain-model/15-m6-github-releases-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M6 GitHub Release Resolve·Acquire·Integrity | M6-003 | Complete | [M6 GitHub Releases Standalone Contract](./docs/domain-model/15-m6-github-releases-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M6 Standalone Static Inspection·Policy | M6-004 | Complete | [M6 GitHub Releases Standalone Contract](./docs/domain-model/15-m6-github-releases-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M6 Linux ELF Dynamic Inspection | M6-005 | Complete | [M6 GitHub Releases Standalone Contract](./docs/domain-model/15-m6-github-releases-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M6 Standalone Promotion·CLI·Linux E2E | M6-006 | Complete | [M6 GitHub Releases Standalone Contract](./docs/domain-model/15-m6-github-releases-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M6 Qualification·Regression·M7 Handoff | M6-007 | Complete | [M6 Qualification](./docs/planning/09-m6-qualification.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M7 MVP Qualification Entry·Evidence Matrix | M7-001 | Complete | [M7 Qualification Contract](./docs/planning/10-m7-mvp-qualification-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M7 Ecosystem Flow·Fixture Regression | M7-002 | Complete | [M7 Qualification Contract](./docs/planning/10-m7-mvp-qualification-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M7 Host CLI Qualification | M7-003 | Complete | [M7 Qualification Contract](./docs/planning/10-m7-mvp-qualification-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M7 Evidence·Result·Resilience Qualification | M7-004 | Complete | [M7 Qualification Contract](./docs/planning/10-m7-mvp-qualification-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M7 Security Workflow·Scheduled Gate | M7-005 | Complete | [M7 Qualification Contract](./docs/planning/10-m7-mvp-qualification-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M7 MVP Final Audit·Completion | M7-006 | Complete | [M7 Qualification Contract](./docs/planning/10-m7-mvp-qualification-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M8 Production Trust Hardening Entry | M8-001 | Complete | [M8 Production Trust Hardening Contract](./docs/planning/11-m8-production-trust-hardening-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Post-MVP Product Install UX | M9 | Complete | [M9 Product Install UX Contract](./docs/planning/12-m9-product-install-ux-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Verified Distribution & Bootstrap | M10 | Complete | [M10 Verified Distribution & Bootstrap Contract](./docs/planning/13-m10-verified-distribution-bootstrap-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Dynamic Detection Depth | M11 | Complete | [M11 Dynamic Detection Depth Contract](./docs/planning/14-m11-dynamic-detection-depth-contract.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M11 Release Hardening Fixes | M11-FIX-01..05 | Complete | [M11 이후 Release Hardening Fix List](./docs/planning/15-m11-02-fix-list.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Ecosystem Expansion Before Public Release | M12 | Complete — merged / main CI verified | [M12 Ecosystem Expansion Contract](./docs/planning/16-m12-01-ecosystem-expansion-contract.md) |
| Official PyTorch Source Qualification | M12-001 | Complete | [Closure Evidence](./docs/planning/evidence/m12-001-closure/result.json), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Public Go Modules Qualification | M12-002 | Complete | [Qualification Evidence](./docs/planning/evidence/m12-002-go-build-qualification/result.json), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Public Cargo/crates.io Qualification | M12-003 | Complete | [Qualification Evidence](./docs/planning/evidence/m12-003-cargo-build-qualification/result.json), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Public Terraform Provider Installation Qualification | M12-004 | Complete | [Qualification Evidence](./docs/planning/evidence/m12-004-provider-qualification/result.json), [Remote CI Evidence](./docs/planning/evidence/m12-002-004-remote-ci/result.json), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Go·Cargo·Terraform Cumulative Remote Qualification | M12-002~004 | Complete | [Remote CI Evidence](./docs/planning/evidence/m12-002-004-remote-ci/result.json), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| Cross-Ecosystem Qualification and Feature Freeze | M12-005 | Complete | [Qualification Evidence](./docs/planning/evidence/m12-005-qualification/README.md), [Current Work Queue](./docs/planning/02-current-work-queue.md) |
| M12 Final Red-Team Fix Gate | M12-02 | Complete | [M12-02 Final Red-Team Fix List](./docs/planning/17-m12-02-fix-list.md) |
| Production Release & Operations | M13 | M13-004 self-update IN_PROGRESS; installation role/Linux amd64 approved; channel names pending separately | [M13 Production Release & Operations Contract](./docs/planning/18-m13-production-release-operations-contract.md) |
| Runtime Upgrade Maintenance | UPG-001..006 | UPG-001..006 complete/qualified; PR34 merged/main CI320 SUCCESS | [Design](./docs/engineering/05-runtime-upgrade-maintenance.md), [Plan](./docs/planning/19-runtime-upgrade-maintenance-plan.md) |

## Current Stage

2026-10-21 owner 요청으로 최신 CLI의 직접 사용 확인을 채널 등록보다 먼저 진행한다.
첫 정식 공개판 목표는 v1.0.0이며, 사용자 검토·보완과 채널 qualification 뒤 보호된
PR merge/main CI·exact 서명 build를 거쳐 배포한다. 실제 tag/publication 권한은
[release activation boundary](./docs/planning/18-m13-production-release-operations-contract.md#4-release-activation-boundary)를 따른다.

M13-001 저장소 보호, M13-003 보호된 v0.1.0 build·attestation·OS package 검증, M13-002 C3 조건부 유예, CLI-UX-001~003, 전체 코드·보안 감사(2026-10-31), 일괄 설치 목표 감사와 보강 설계(2026-11-01)의 상세 서술은 [단계 이력 보관](./docs/planning/archive/project-decisions-stage-history-20261110.md)에 있다. 현재 판정은 아래 링크가 소유한다.

- 현재 상태·남은 순서: [Current Work Queue](./docs/planning/02-current-work-queue.md)
- 목표·보강 설계: [보강 설계](./docs/planning/21-goal-completion-design.md), [구현 대조](./docs/planning/evidence/m13-004-convenience/goal-coverage-audit-20261101.md)
- 전체 감사: [full-source-audit-20261031](./docs/planning/evidence/m13-004-convenience/full-source-audit-20261031.md)
- 개발 Host 체험: [local-dev-test-20261101](./docs/planning/evidence/m13-004-convenience/local-dev-test-20261101.md)

Step 14 — Implementation: M0·M1·M2·M3·M4·M5-001..007·M6-001..007·M7-001..006·M8-001..007 `COMPLETE`

Documentation hierarchy and task routing migration: Complete

2026-10-15 버전 갱신 유지보수(UPG-001..006 완료·qualified)와 M12-001~005·M12-02 완료, M13-002 보호 경로 시험의 상세 이력은 [단계 이력 보관](./docs/planning/archive/project-decisions-stage-history-20261110.md)에 있다. 설계는 [Runtime upgrade design](./docs/engineering/05-runtime-upgrade-maintenance.md)과 [plan](./docs/planning/19-runtime-upgrade-maintenance-plan.md), M12 지원 결정은 [M12 계약](./docs/planning/16-m12-01-ecosystem-expansion-contract.md#first-release-support-tuples), M13-002 증거는 [검증 README](./docs/planning/evidence/m13-002-protected-path/README.md)가 소유한다.

M7~M12는 완료되었고 M13에서 repository activation과 public deployment를 수행한다. 흐름 요약은 [단계 이력 보관](./docs/planning/archive/project-decisions-stage-history-20261110.md)에 있다.

## Remaining Stages

| Step | Stage | Status | Outcome |
| --- | --- | --- | --- |
| 8 | Directory Structure | Complete | 구현 디렉터리, package 경계와 파일 배치 규칙 |
| 9 | Coding / Security Rules | Complete | 코드 작성·의존성·비밀값·오류 처리·보안 구현 규칙 |
| 10 | Formatter / Linter / Type Check / Test / Security Scan | Complete | 로컬 deterministic 검증 도구와 실행 명령 |
| 11 | CI + Quality Gate | Complete | CI workflow와 merge 차단 기준 |
| 12 | Milestones | Complete | 구현 단계, 의존 관계와 완료 조건 |
| 13 | Current Work Queue | Complete | 실행 가능한 현재 작업과 우선순위 |
| 14 | Implementation | Defined | 구현 진행 상태는 Current Work Queue가 소유 |

2026-10-21 client/runner 설계([설계](./docs/user-journey-cli-ia/09-package-client-quarantine.md), [계획](./docs/planning/20-package-client-quarantine-plan.md))는 기존 검역/Policy/transaction을 재사용한다. 지원 lane은 [최종 지원표](./docs/planning/evidence/package-client-quarantine/p7-command-acceptance-matrix-20261030.md)를 따르며, 한 소스의 qualification 결과를 후속 후보나 별도 범위로 확대하지 않는다. 상세 이력은 [단계 이력 보관](./docs/planning/archive/project-decisions-stage-history-20261110.md)에 있다.

단일 active item은 M13-004이며 상세 상태·남은 순서는 [Queue](./docs/planning/02-current-work-queue.md)가 소유한다. CLI 시작 화면은 [아트 확정 기록](./docs/planning/evidence/cli-launcher/art-refresh-20261031.md)을 따른다.

## Open Decisions

아래 항목은 후속 단계에서 확정한다. 상세 결정을 이 표에 작성하지 않고 해당 단계의 canonical 문서로 이동한 뒤 상태와 링크만 갱신한다.

| Item | Planned stage or context |
| --- | --- |
| 구체 runtime/container 기술과 배포 형태 | M3 entry decision 또는 backend 구현 직전 |
| 설정 형식과 schema | 관련 구현 직전 상세 설계 |
| Adapter 등록과 plugin 확장 방식 | Step 9 또는 구현 직전 ADR |
| 구체 동시성·resource limit 값과 backend별 제어 | 관련 구현 직전 상세 설계 |
| Evidence storage·retention·cleanup | 구현 직전 상세 설계 |
| M2 이후 Policy rule·version·reason code | 각 ecosystem 구현 직전 상세 설계 |
| 라이선스·상표·배포 정책 | 배포 전 |
| 기본 MCP 제공 | [보강 설계 A02](./docs/planning/21-goal-completion-design.md)의 조건부 현재 후보. CLI 결과 안정화 후 작은 재사용 연결로 닫을 수 있으면 구현·검증; 현재 미구현 |
| MCP AI 보조 실행 검사·자체 코드 생성 | [향후/v2.0 핵심 기능 후보](./docs/planning/22-future-ai-assisted-inspection.md). 사용자 요청으로 아이디어만 기록 / DEFERRED; 현재 보강·첫 배포 필수 범위 밖 |

## Documentation Rules

- 이 문서는 결정 상태와 open decision만 기록한다.
- 작업별 문서 선택은 [docs/README.md](./docs/README.md)가 담당한다.
- 실제 결정은 관련 canonical leaf 문서에 한 번만 기록한다.
- 완료된 상세 결정을 이 문서에 장문으로 재서술하지 않는다.
- 문서 이동 시 상태 링크와 Documentation Guide를 같은 변경에서 갱신한다.
2026-11-01~02 B00~B07, S01/S08/S09, C01의 로컬 구현·연결 서술은 [단계 이력 보관](./docs/planning/archive/project-decisions-stage-history-20261110.md)에 있다. 구현 상태는 Queue와 [구현 대조](./docs/planning/evidence/m13-004-convenience/goal-coverage-audit-20261101.md)가 소유하며 grouped qualification은 OPEN이다.
