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
| Production Release & Operations | M13 | M13-003 and CLI-UX-001/002 complete; SEC-UPD-001 and owner package names required before M13-004 | [M13 Production Release & Operations Contract](./docs/planning/18-m13-production-release-operations-contract.md) |
| Runtime Upgrade Maintenance | UPG-001..006 | UPG-001..006 complete/qualified; PR34 merged/main CI320 SUCCESS | [Design](./docs/engineering/05-runtime-upgrade-maintenance.md), [Plan](./docs/planning/19-runtime-upgrade-maintenance-plan.md) |

## Current Stage

2026-10-17 사용자가 S/C2 이후 C3 조건부 유예를 명시 승인했다. M13-002는 `ACCEPTED_WITH_EXCEPTION / QUALIFIED=NO / ACCEPTANCE_CLOSED=YES`이며 실제 missing-CLA는 `NOT_TESTED / DEFERRED`로 보존한다. 정확한 범위·중단/종료 조건은 [M13 계약](./docs/planning/18-m13-production-release-operations-contract.md#m13-002--protected-branch-qualification) 한 곳이 소유하고, 승인 입력은 [C3 기록](./docs/planning/evidence/m13-002-protected-path/cla-deferral-proposal-20261017.md)을 따른다. 별도 승인된 PR37 일반 병합과 새 main CI327 전체 성공, 정확한 main ef5ef0e의 보호된 v0.1.0 태그·자연 build·12개 attestation·OS package lifecycle 검증으로 M13-003을 완료했다. [검증 결과](./docs/planning/evidence/m13-003-release-build/qualification-20261017.md)가 범위와 원본을 연결한다. 2026-10-18 사용자가 실제 로컬 CLI 체험 후 다국어·진행 표시·이해 가능한 결과를 우선 요구하여 CLI-UX-001 baseline audit 후 보완했고, [로컬 qualification](./docs/planning/evidence/m13-cli-ux/qualification-20261018.md)과 설치 CLI의 실제 검사·설치를 확인했다. npm/PyPI rahoney 계정 준비는 사용자 보고로 완료됐고 패키지 이름은 사용자 결정으로 미정이다. M13-004는 업데이트 검역과 이름 결정을 기다리며 M13-005 공개 배포·부팅된 Host 검증은 미착수다. 이후 사용자가 업데이트 검역과 OS 표시 언어 자동 감지를 명확히 요구했다. [배포 전 baseline](./docs/planning/evidence/prepublication-gaps/baseline-20261018.md)에 따라 상세 변경 계획·2차 검토 후 CLI-UX-002 구현과 로컬 반영을 완료했다. [언어 qualification](./docs/planning/evidence/cli-os-language/qualification-20261018.md)은 실제 WSL/설치 CLI와 native macOS를 포함한 CI328 전체 성공을 연결한다. SEC-UPD-001은 계획·2차 검토를 마쳤고, 2026-10-18 후속 사용자 승인으로 순차 구현을 시작했고, [npm qualification](./docs/planning/evidence/update-quarantine/npm-qualification-20261018.md)은 CI330 전체 성공까지 마쳤으며 [PyPI 관리 venv 검증](./docs/planning/evidence/update-quarantine/pypi-managed-qualification-20261019.md)은 실제 업데이트·재검사와 CI331 전체 성공을 확인했다. [일반 pip 인계의 로컬 검증](./docs/planning/evidence/update-quarantine/pypi-native-qualification-20261019.md)도 실제 11개 CLI·복원·시험 데이터 삭제까지 통과했다. 새 원격 검증과 named-source 및 나머지 생태계 acceptance는 열려 있다. 사용자가 요청한 Heliopause Artifact Airlock 대화형 시작 화면은 CLI-UX-003으로 기록하고 업데이트 검역 acceptance 뒤에 진행한다. 채널은 이 업데이트 검역과 사용자 패키지 이름 결정을 기다린다. 기존 locale-only 완료를 새 요구의 완료로 확대하지 않는다. 현재 작업은 Queue가 소유한다. 아래 이전 단계 문구는 역사적 기록이다.

Step 14 — Implementation: M0·M1·M2·M3·M4·M5-001..007·M6-001..007·M7-001..006·M8-001..007 `COMPLETE`

Documentation hierarchy and task routing migration: Complete

2026-10-15 사용자가 M13을 보류하고 버전 갱신 유지보수의 설계·계획을 먼저
정리하도록 요청했다. UPG-001 baseline audit/design/planning을
`milestone/runtime-upgrade-maintenance`에서 완료했고 UPG-002 owner/manifest/drift와
UPG-003 updater/no-op·복구·canonical verify와 UPG-004 compiler 후보 build·재현성·capture 계획 연결을 검증했다. UPG-005의 실제 local CPU/세 CUDA 및 fresh ordinary/nativeMac/세 CUDA remote qualification을 완료했다. UPG-006의 기존 구현·설치 상태를 source 변경 전에 감사하고 실제 controlled 전환·retirement·운영 인계를 완료했다. 최종 metadata-head CI와 reviewer 정상 병합은 별도로 확인하며 M13 보류·새 baseline/별도 병합 승인 조건은 유지한다. 상세 결정은
[Runtime upgrade design](./docs/engineering/05-runtime-upgrade-maintenance.md),
작업·완료·재개 조건은 [plan](./docs/planning/19-runtime-upgrade-maintenance-plan.md)과
Queue가 소유한다. 아래 M13 기록은 보류 전 관측이며 새 원격 상태를 뜻하지 않는다.

M12-001~004 acceptance 및 누적 원격 qualification을 완료했다. M12-005의
전체 생태계 qualification, 구현·설계 대조와 성공 테스트 데이터 정리를 완료하고
네 exact PyTorch tuple의 bounded support 및 feature freeze를 확정했다. 상세 지원
결정은 [M12 계약](./docs/planning/16-m12-01-ecosystem-expansion-contract.md#first-release-support-tuples)을 따른다.
M12-02의 여덟 영역 검토와 FIX-01~03 remediation을 완료했다. 같은 기능 후보의
CPU·cu126·cu130·cu132 및 각 Required CI를 검증해 M12 기술 acceptance는 `COMPLETE`다.
[검토 결과](./docs/planning/evidence/m12-02-red-team/result.json)가 소스와 원본을 연결한다.
PR29·PR30은 merge됐고 main2e196a1의 CI38328099413도 SUCCESS다. M13-001의
저장소 보호 설정 적용·API 검증을 완료했다. M13-002의 새 baseline 확인과
로컬 검증 후보 준비를 완료했다. 승인된 e1e3aa8을 feature 브랜치에 push하고
develop 대상 PR31을 생성했다. 새 Required 포함10개 CI job과 CLA 성공 및 두 필수
check 연결을 확인했다. 별도 승인된 미해결 리뷰 스레드 시험의 일반 병합 요청
1회가 HTTP405로 차단됐고, 브랜치 미변경과 스레드 해결 후 CLEAN 복귀를 확인했다.
독립 보조 PR32의 새 Required 포함10개 job·CLA SUCCESS와 CLEAN을 확인한 뒤,
별도 승인에 따라 정확한 e1e3aa8의 PR31을 정상 보호 경로로 develop에 병합했다.
develop은84591ef로 전진했고 main2e196a1과 PR32 headff441d3는 그대로다.
PR32는 다른 검사·리뷰·충돌 차단 없이 CLEAN→BEHIND로 변경됐다.
[병합 및 stale 관측 증거](./docs/planning/evidence/m13-002-protected-path/develop-merge.json)를 보존했다.
별도 승인한 PR32 일반 병합 요청 1회는 HTTP405로 거절됐다. 서버 응답은 필수 검사
2개가 expected라는 사유이며, 독립 stale 차단 PASS는 확정하지 않았다. PR32 미병합과
세 브랜치 HEAD 동일성을 확인하고 추가 요청을 중단했다. 기존 시험 원본은 그대로
보존하고 읽기 전용 rule-suite4516024742 분석으로 실제 업데이트 후보0fd006b와
실패한 Strict required_status_checks를 확인했다. 독립 freshness-only PASS는
선언하지 않았다. [분석](./docs/planning/evidence/m13-002-protected-path/strict-check-analysis.json)과
[남은 검증·승인 계획](./docs/planning/evidence/m13-002-protected-path/README.md#remaining-qualification-and-approval-plan)이
필수 참가자·조건부 forward-only 묶음·별도 main merge/force-push 결정을 구분한다.
현재는 [정확한 로컬 후보·유한 실행 계획](./docs/planning/evidence/m13-002-protected-path/forward-only-plan.json)을
준비했고 원격 변경은 하지 않았다. 사용자가 실제 미서명 기여자 부재를 확인하여
CLA 부정 시험은 `NOT_TESTED`로 보류한다. 기존 필수 설정·정상 성공 증거는 유지한다.
나머지 검증은 계속할 수 있으나 현재 계약의 acceptance gap은 종료 전에 별도
해소해야 한다. main stale을 최종 병합 전에 검증하기 위한 중간 보호 병합과
최종 develop→main 병합, 실제 non-fast-forward 위험은 각각 별도 승인 범위다.
후속 승인된 묶음 A는 두 direct push의 서버 PR-rule 차단을 확인했다. PR32의 main
retarget는 성공했으나 읽기 전용 재확인에서도 BLOCKED여서3/8회 쓰기에서 중단했다.
두 보호 브랜치·feature HEAD는 유지됐고 병합 요청·새 PR·수동 CI는0회다.
[실행 결과](./docs/planning/evidence/m13-002-protected-path/bundle-a-result.json)와
[수정 제안](./docs/planning/evidence/m13-002-protected-path/bundle-a-resume-plan.json)을 기록했다.
다음 정상 병합은 실제 develop→main PR·새 CI·main 부정 결과가 확보된 뒤 승인 요청한다.
승인된 수정 묶음 A′는 develop84591ef→main2e196a1 PR33을 생성했다. 실제 CLA
성공·Required 누락 상태의 단회 일반 PUT은 Required만을 명시한405로 거절됐다.
Suite4521104354 및 refs동일을 보존했고, 같은 자연CI38574753115/attempt1의10job과
CLA 성공 뒤 main 대화·일반관리자 요청의 단회405도 확인했다(suite4521761091).
생성한 시험 스레드만 해결하여 CLEAN으로 복귀했으며 A′는5/5회 쓰기로 완료했다.
[A′ 결과](./docs/planning/evidence/m13-002-protected-path/bundle-a-prime-result.json)가
현재 단계와 원본을 소유한다. [정확한 PR33 정상 병합 요청](./docs/planning/evidence/m13-002-protected-path/normal-main-merge-approval.json)은
검토 가능하지만 미승인이다. 실제 main 병합·새 mainCI 및 남은 acceptance는 별도다.
guest runtime·Race 등 출시 영향 판단은 남는다. 현재 milestone과
active/next work item은 [Current Work Queue](./docs/planning/02-current-work-queue.md)를 참조한다.

M7 MVP qualification evidence는 보존한다. external security review에서 확인된
production trust-boundary gap을 M8에서 remediation했고, M9 Product Install UX
transaction hardening과 M10 verified bootstrap implementation을 완료했다. M11
detection depth도 완료되었으며, M12 ecosystem expansion과 M12-02 final
red-team/fix gate가 완료된 뒤 M13에서 repository activation과 public
deployment를 수행한다.

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
| MCP 제공 여부 | CLI 안정화 후 |

## Documentation Rules

- 이 문서는 결정 상태와 open decision만 기록한다.
- 작업별 문서 선택은 [docs/README.md](./docs/README.md)가 담당한다.
- 실제 결정은 관련 canonical leaf 문서에 한 번만 기록한다.
- 완료된 상세 결정을 이 문서에 장문으로 재서술하지 않는다.
- 문서 이동 시 상태 링크와 Documentation Guide를 같은 변경에서 갱신한다.
