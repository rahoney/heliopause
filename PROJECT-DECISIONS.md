# Heliopause Artifact Airlock 프로젝트 결정 색인

이 문서는 `Heliopause Artifact Airlock`(`helox`)의 전체 결정 상태와 상세 문서 위치를 안내하는 중앙 색인이다. 상세 정책과 사용자 원문은 주제별 `docs/` 문서에 한 번만 보존한다.

## 읽기 순서

1. [`README.md`](./README.md)
2. 이 문서
3. 현재 작업에 관련된 상세 문서
4. 작업 queue 문서가 생성되면 `current_work_queue.md`

## 프로젝트 기본 정보

| 항목 | 상태 | 결정 | 비고 |
| --- | --- | --- | --- |
| 프로젝트명 | 결정 | Heliopause Artifact Airlock |  |
| CLI 명령어 | 결정 후보 | `helox` | 사용 가능 여부와 패키지 충돌은 확인 필요 |
| 목적 | 초안 | 외부 artifact·패키지를 검역하고 검증된 고정 artifact만 목표 환경에 반입 |  |
| 1차 사용자 | 미결정 |  | 개인 개발자, 팀, CI 운영자 중 선택 |
| 지원 운영체제 | MVP 기준 결정 | Linux 격리·동적 실행 환경 | macOS·Windows native backend는 MVP 이후 확장 |
| CLI Host 지원 OS | MVP 기준 결정 | Linux·macOS 네이티브, Windows는 WSL2 기반 Linux | Windows 네이티브 실행은 MVP 이후 확장 |
| MVP 지원 Artifact 유형 | MVP 기준 결정 | npm package, Python wheel/source distribution, GitHub Releases binary·archive | GUI installer, OCI image, IDE extension 등은 향후 확장 |
| MVP 정적 검사 범위 | MVP 기준 결정 | 출처·identity·무결성, 구조·코드·dependency·변경·악성 패턴 | 특정 scanner·database는 MVP Scope에서 미확정 |
| MVP 동적 설치·실행 검사 | MVP 기준 결정 | 격리 설치·제한 실행과 파일·프로세스·네트워크·자원 행위 관찰 | 실행 불가·필수 검사 미완료 시 자동 `ALLOW` 금지 |
| MVP Signature·Provenance·무결성 | MVP 기준 결정 | digest, Registry 정보, signature, provenance, attestation을 Evidence로 종합 | 서명·provenance 부재만으로 자동 `BLOCK`하지 않되 필수 정책 누락은 `MANUAL_REVIEW`/`BLOCK` |
| MVP SBOM·Evidence·결과물 | MVP 기준 결정 | 재현 가능한 검사 기록, SBOM, Staging 검증 세트 Manifest, 요약·구조화 결과 | 저장 방식·보존 기간·세부 schema는 Architecture/Domain Model에서 확정 |
| MVP 사용자 흐름·완료 기준 | MVP 기준 결정 | 지정→식별→격리 다운로드→검증→정적·동적 검사→Policy→Staging/결과 분기 | npm E2E 완성 및 PyPI/GitHub adapter 공통 흐름 수행, Core 생태계 전용 로직 금지 |
| 배포 형태 | 미결정 | Docker image, standalone CLI, package | 신뢰·재현성 검토 필요 |
| 라이선스 | 미결정 |  |  |

## 결정 주제 색인

### Threat Model

- 상태: 완료 — D-001~D-014 확정
- 기본 원칙: Untrusted by default
- Registry는 출처·버전·메타데이터 제공 인프라로 제한 신뢰
- Host 자산은 default deny이며 실제 Secret을 격리 환경에 노출하지 않음
- 다운로드 → 설치 → 필요 시 실행 순으로 단계별 검사
- 검사에 사용한 동일 Artifact 세트를 staging area로 승격
- Evidence 무결성·completeness와 fail-closed 적용
- 검사 이미지·scanner·verifier도 외부 Artifact로 검증
- resource limit·timeout과 검사 환경 폐기로 자원 고갈 대응
- 상세·사용자 원문: [`docs/threat-model.md`](./docs/threat-model.md)

### MVP Scope

- 상태: 완료 — MVP-001~MVP-009 확정
- 최초 지원 생태계: npm, Python/PyPI(pip), GitHub Releases
- 검사 실행 기준: Linux 격리·동적 실행 환경
- CLI Host는 Linux·macOS 네이티브 우선 지원, Windows는 WSL2 기반 Linux를 공식 경로로 지원
- CLI Host OS와 Artifact 검사 격리 환경은 분리하며 macOS Host에서도 Linux 격리 backend를 사용
- MVP 지원 유형은 npm package, Python wheel/source distribution, GitHub Releases의 단일 binary·archive
- Python source distribution과 archive는 설치·압축 해제 과정 자체를 검사 대상으로 취급
- 확장자만으로 형식을 신뢰하지 않으며 중첩 archive·path traversal·압축폭탄 등 위험을 제한
- OS 전용 GUI installer, Docker/OCI image, IDE extension, MCP server, Git repository, CI component, `curl | sh` 설치 흐름은 MVP 제외
- 설치·실행 전 출처·identity·무결성·metadata·구조·코드·dependency·변경·악성 패턴을 정적으로 검사
- 직접·전이 dependency와 manifest/lock 불일치, 이전 버전 대비 주요 변경을 분석하고 공통 Finding/Evidence로 정규화
- Heliopause 자체 검사와 외부 scanner·database 조합을 허용하되 특정 도구는 아직 확정하지 않음
- 정적 통과 또는 추가 검증이 필요한 Artifact를 격리 설치·제한 실행하고 파일·프로세스·shell·네트워크·Credential/honeytoken·권한·자원을 관찰
- 설치 전후 filesystem·주요 상태 diff와 설치 중 lifecycle/build 코드를 Evidence로 기록
- 실제 Host와 분리된 모의 filesystem·honeytoken·통제 네트워크에서 수행하며 resource limit·timeout·fail-closed 적용
- Checksum/Digest를 Artifact 동일성·무결성의 기본 식별 수단으로 사용
- Registry integrity 정보, Publisher signature, provenance, attestation 및 GitHub Releases checksum/signature를 가능한 경우 검증하고 Evidence로 기록
- Signature/provenance 부재는 자동 `BLOCK`하지 않고 다른 Evidence와 종합하되, 정책상 필수 정보가 없으면 자동 `ALLOW`하지 않음
- 모든 결과는 Artifact identity·검사·Finding·Evidence·최종 Policy Decision을 연결해 재현 가능한 기록으로 생성
- dependency·구성요소는 가능한 범위에서 표준 SBOM으로 생성하되 구체 표준·도구는 후속 확정
- Staging 승격 시 Artifact·dependency digest와 검사 결과·Evidence를 연결한 검증 완료 세트 Manifest 생성
- 사람이 보는 요약과 CI·MCP·자동화용 machine-readable 결과를 함께 제공
- 사용자 지정부터 식별·격리 다운로드·검증·정적·동적 검사·Policy 판정·Staging/설치·결과 제공까지의 전체 흐름을 MVP 기준으로 정의
- `ALLOW`는 원칙적으로 Staging을 거쳐 반입하되 독립 binary/archive는 D-011 digest 재검증 후 직접 반입 가능, `WARN`은 Finding 수준이고 `MANUAL_REVIEW`는 최종 추가 판단 상태, `BLOCK`은 반입 금지
- npm end-to-end 완성, PyPI/pip·GitHub Releases의 동일 Core·adapter 흐름 수행을 MVP 완료 기준으로 삼음
- 생태계 추가를 위해 Core에 전용 로직을 넣어야 하면 MVP 완료로 보지 않음
- npm을 가장 완전한 end-to-end reference implementation으로 구현
- PyPI/pip와 GitHub Releases로 adapter·공통 Core 확장성을 검증
- 생태계별 특수 로직은 adapter에 한정하고 Core는 생태계 중립 유지
- macOS·Windows native dynamic quarantine은 MVP 이후 확장
- Linux에서 검증할 수 없는 플랫폼 전용 동작은 자동 `ALLOW`하지 않음
- 범용 계약 확장은 후방호환 범위에서 허용하며, Core 전용 로직이나 기존 adapter 비호환이 필요하면 구조 재검토
- 상세·사용자 원문: [`docs/mvp-scope.md`](./docs/mvp-scope.md) (MVP-001~MVP-009)

### Architecture

- 상태: 완료 — Architecture-001~Architecture-008 확정
- 방향: Go 기반 Modular Monolith CLI, 하나의 배포 가능한 본체·주 코드베이스, 제한적 Ports/Adapters
- 핵심 경계: CLI → Application/Workflow → Core/Domain·Policy·Ports/Contracts → Adapters/Backends/Providers, Composition Root wiring
- 최상위 책임 영역: CLI, Application/Workflow, Core/Domain, Artifact, Verification, Inspection, Sandbox/Runtime, Policy, Evidence/Result, Staging/Promotion
- Static/Dynamic inspection, ecosystem adapter, Evidence/SBOM/Result는 관련 영역 내부 하위 구성으로 유지하고 Staging/Promotion은 별도 최상위 영역으로 분리
- Scanner/Inspection은 Finding/Evidence, Verifier/Verification은 Verification Result/Evidence, Sandbox/Runtime은 raw observation/실행 상태를 생성하며 최종 `ALLOW`/`MANUAL_REVIEW`/`BLOCK`은 Policy만 담당
- Core·Application은 npm·PyPI·GitHub Releases·Cobra·특정 scanner·container runtime에 직접 의존하지 않음
- MVP는 microservice로 분할하지 않으며, Rust 저수준 isolation component 분리는 향후 선택지로 유지
- 의존성은 Core·공통 계약을 향하며 Core/Application의 구체 구현 직접 의존을 금지
- Composition Root/Bootstrap만 구체 구현 생성과 dependency wiring을 담당
- 순환 의존과 금지된 dependency direction은 architecture test·import rule로 검증
- Artifact Adapter와 External Tool Provider/Adapter를 분리하고 Capability(지원·미지원), Execution Status(미실행·실패 등), Security Result(Finding·Evidence)를 안전 판정과 구분
- MVP 동적 plugin loading은 구현하지 않고 Composition Root에서 명시적으로 생성·등록
- Inspection과 Sandbox/Runtime 책임을 분리하고 공통 Sandbox Port·ephemeral Session·trusted observer를 사용
- Linux isolation·container·syscall·resource 수치는 후속 backend/tooling 결정으로 유보
- Verification은 Artifact 출처·identity 신뢰 근거를, Inspection은 내용·행위를 분석하고 각각 정규화 결과를 생성
- Evidence는 사실·원본 근거, Finding은 Evidence에 대한 검사 계층 해석으로 분리
- Capability, Execution Status와 실제 Security Result(Finding·Evidence)를 구분하며 검사 불가·결과 없음은 안전으로 해석하지 않음
- Policy만 최종 `ALLOW`/`MANUAL_REVIEW`/`BLOCK`을 생성하고 severity·score·threshold는 후속 결정
- Evidence Store와 Staging Area는 논리적으로 분리하고 Inspection Run 중심으로 Raw Evidence·Finding·Policy·SBOM·Manifest를 추적
- Finding/Decision 근거로 채택된 Raw Evidence는 정해진 retention 동안 보존하고 대용량 원시 telemetry 정책은 후속 결정, trusted control/observer만 Evidence를 기록
- 저장 engine·layout·schema·retention·cleanup은 후속 Domain Model·Storage/Tooling 설계로 유보
- Quarantine/Sandbox와 Staging을 분리하고 `ALLOW` Verified Set만 승격·반입
- Quarantine→Staging 및 Staging→사용자 환경 반입 직전에 digest 재검증
- Package 설치는 원칙적으로 Verified Set만 사용하고 외부 dependency는 acquisition/verification·필요 시 full quarantine으로 전환
- 독립 binary/archive는 D-011 기반 digest 재검증 후 Staging 우회 직접 반입 가능
- Promotion은 Policy 재판정·재검사 없이 Application의 `ALLOW` 확인 후 동일성·승격·반입만 담당
- `ALLOW` Decision·Verified Set·Verified Manifest의 기준 Inspection Run은 서로 일치해야 하며, 각 Artifact/dependency는 선언된 identity/digest와 정확히 일치하고 자신을 검증한 관련 Inspection Run까지 추적 가능해야 함. 참조 관계를 벗어난 구성요소나 검증되지 않은 Artifact/dependency 혼합 시 Promotion 금지
- 상세·사용자 원문: [`docs/architecture.md`](./docs/architecture.md) (Architecture-001~Architecture-008)

### User Journey + CLI IA

- 상태: 완료 — User-Journey-001~008 확정
- 범위: 주요 사용자 Journey, Artifact 입력 방식, CLI 최상위 Command 구조, 기본 End-to-End 명령, 자동 실행과 사용자 개입 경계, 결과 표시·상세 조회, 실패·재검사·재개, Promotion·Install 사용자 경험
- 결정 방식: 위 8개 항목을 순차적으로 확정
- User-Journey-001: npm·PyPI/pip·GitHub Releases의 공통 Canonical Journey, `Install / Import Through Heliopause`·`Inspect Only`·`Review / Continue Existing Run` 대표 흐름, 설치·반입 요청 시 `ALLOW` 후 Verified Set·Staging·trusted Promotion 연속 수행 원칙 확정
- User-Journey-002: Primary Artifact 단일 지정, 생태계별 입력 정보, 모호한 reference 명확화, 가변 reference의 exact identity/digest resolve, MVP 직접 입력 source 제한 확정
- User-Journey-003: MVP CLI를 `helox <ecosystem/source> <operation> <artifact>` 구조로 구성하고, ecosystem별 `install`·`inspect`와 공통 `review`를 제공하며 내부 workflow 단계와 `promote`를 일반 사용자 Command로 직접 노출하지 않는 원칙 확정
- User-Journey-004: `install`이 검사 workflow를 선행하고 `ALLOW` 시 별도 `promote` 없이 원래 설치·반입을 계속하며, `inspect`는 반입 없이 종료하는 End-to-End 흐름과 실패·dependency·Dynamic Inspection 처리 원칙 확정
- User-Journey-005: `ALLOW`는 원래 요청을 자동 계속하고, `MANUAL_REVIEW`는 보류·검토, `BLOCK`은 추가 확인 없이 차단하며 모호한 입력만 사용자에게 요구하는 개입 경계 확정
- User-Journey-006: 기본 terminal은 결과 중심으로 요약하고 `review`에서 Inspection Run의 Verification·Finding·Evidence·SBOM·Manifest 등을 단계적으로 상세 조회하며 사람용·기계용 결과의 동일성 유지 원칙 확정
- User-Journey-007: 제한된 retry, 필수 검사 실패의 fail-closed, Sandbox 폐기, Inspection Run 불변성, identity/digest 기반 재검사·Promotion 연결 원칙 확정
- User-Journey-008: 원래 install context 유지, Verified Set·Manifest·identity/digest 동일성, Promotion 전 재검증, standalone 예외, Policy와 Operation Status 분리 원칙 확정
- 상세·사용자 원문: [`docs/user-journey-cli-ia.md`](./docs/user-journey-cli-ia.md) (User-Journey-001~008)

### Domain Model

- 상태: Domain-001~007 확정 — Interface Contracts 설계 예정
- 범위: 핵심 Domain Entity, Artifact Reference/Identity, Inspection Run, Capability/Execution/Policy/Operation 상태, Verification/Finding/Evidence, Dependency/Verified Set/Manifest, Operation Request/Install Context
- 예정 Contract: Artifact Port, Verification/Inspection/Sandbox/Policy Ports, Evidence/Staging/Promotion Ports
- 결정 항목: Domain-001~Domain-007, Contract-001~Contract-003
- Domain-001: ecosystem·구현 중립 핵심 Concept, Inspection Run 중심 lifecycle, Verification/Finding/Evidence 분리, Capability/Execution Status/Inspection Limitation, Policy Decision, Verified Set/Manifest, Operation Result 및 Sandbox Session/Observation 역할 확정
- Domain-002: Artifact Reference·Resolved Artifact Identity·Acquired Artifact를 분리하고 source·exact identity·observed digest binding 및 declared/observed integrity 구분 원칙 확정
- Domain-003: exact identity resolve 후 Run 생성, acquisition 전 Run ID 발급, 실제 Artifact binding, 결과·context 추적, operational failure와 Policy Decision 분리, Finalized Run 불변성 및 재검사 새 Run 원칙 확정
- Domain-004: Capability·Execution Status·Run Lifecycle/Outcome·Policy Decision·Operation Status를 독립 축으로 분리하고 fail-closed, Policy 부재, Operation 결과 조합 원칙 확정
- Domain-005: Verification Result·Evidence·Finding을 구분하고 declared/invalid/failed 상태·Raw/normalized Evidence·Finding 근거·Severity/Policy 분리 및 실제 Artifact 추적 원칙 확정
- Domain-006: Dependency Requirement·Resolved Dependency·Verified Set·Verified Manifest를 분리하고 exact identity/digest·Run/Policy 추적, Set 일관성 및 양쪽 경계 재검증 원칙 확정
- Domain-007: Operation Request·Install Context·Operation Result를 분리하고 사용자 의도 보존, install/inspect 분기, Host/Sandbox context 격리, ALLOW·Manifest/exact Artifact·원래 context가 일치하는 trusted Promotion 원칙 확정
- 상세·사용자 원문: [`docs/domain-model.md`](./docs/domain-model.md) (Domain-001~007)

### 언어·프레임워크·실행 기반

| 항목 | 후보 | 결정 | 이유·검증 메모 |
| --- | --- | --- | --- |
| 주 구현 언어 | C++, C#, Go, Rust, Java | Go | 보안 도구·CLI·배포 생태계와 개발 생산성의 균형을 고려한 1순위 |
| CLI framework | 언어별 후보 | Cobra | Go 기반 CLI의 명령 구조·argument parsing·config·terminal UX에 대한 1순위 |
| 설정 형식 | YAML, TOML, JSON 등 | 미결정 | 스키마 검증과 주석 지원 |
| plugin/adapter 방식 | 미결정 |  | package manager와 검사 도구 확장 |
| 동시성 모델 | 미결정 |  | 병렬 다운로드·검사와 자원 제한 |
| 최소 실행 환경 | 미결정 |  | container image와 standalone binary |
| MCP 제공 여부 | 미결정 |  | CLI 안정화 후 AI client용 adapter로 검토 |

언어·CLI 선택 결정

- 메인 언어는 Go로 결정한다.
- CLI framework는 Cobra를 1순위로 사용한다.
- Rust와 `clap`은 향후 저수준 isolation component를 직접 구현할 경우의 우선 후보로 남긴다.
- Rust·`clap` 후보는 현재 메인 CLI 구현을 대체하는 결정이 아니라, 저수준 격리 구성요소가 필요할 때 별도 검토할 확장 선택지다.

이 결정은 현재 `PROJECT-DECISIONS.md`에 기록하고, Architecture 문서에서는 Go/Cobra를 전제로 모듈 경계·실행 계층·backend 계약을 설계한다. 저수준 isolation component를 Rust로 분리할 경우의 IPC·배포·권한 경계는 해당 Architecture/ADR 단계에서 별도로 확정한다.

## 기능 범위 초안

### 1차 필수 기능 후보

- package·artifact 요청과 버전·source 고정
- 다운로드 cache와 checksum/digest 기록
- provenance·서명·배포자 확인
- 격리된 검사 환경 생성과 lifecycle script 통제
- 정책 기반 검사 pipeline 실행
- 검사 결과, SBOM, manifest, provenance receipt 생성
- 통과 artifact의 명시적 반입
- 실패·보류·수동 승인 상태 관리
- 한국어·영어 메시지와 구조화된 machine-readable 결과

### 후순위 기능 후보

- 다중 package manager adapter
- 조직 정책과 allowlist/denylist
- CI/CD integration
- Docker image signing과 remote registry
- MCP server
- 재검사와 artifact quarantine archive
- 팀 단위 결과 공유와 감사 로그

## 모듈 구조 초안

초기 구조는 모놀리식 실행 파일 내부에서도 책임을 분리한 모듈식 경계를 우선한다.

```text
helox/
├── cli/                 # 명령, 옵션, 사용자 출력
├── config/              # 설정·정책·스키마
├── acquisition/         # 다운로드·cache·source pinning
├── provenance/          # checksum·digest·signature·attestation
├── quarantine/          # 격리 환경과 권한·네트워크 정책
├── adapters/            # npm, PyPI, Cargo 등 package manager
├── scanners/            # 정적·악성·취약점·SBOM 검사 adapter
├── policy/              # pass/fail/review/allow 규칙
├── manifest/            # artifact receipt·결과·감사 기록
├── promotion/           # 검증 artifact의 목표 환경 반입
├── localization/        # 한국어·영어 등 메시지
└── tests/               # unit·integration·contract·E2E
```

모듈 간 의존 방향, public contract, 금지된 import와 plugin ABI는 구현 언어 결정 후 별도 ADR로 확정한다.

## 사용자 여정 초안

```text
검사 대상 지정
  → helox가 환경·package manager·정책 탐지
  → 다운로드 전 source/version/digest/provenance 확인
  → artifact를 quarantine으로 이동
  → 정적·서명·악성·취약점·SBOM 검사
  → deterministic policy 판정
  → pass면 promotion, fail이면 차단·격리
  → 결과 receipt와 재현 명령 제공
```

예상 사용자 경로와 오류 복구 경로는 첫 지원 package manager가 정해진 뒤 BDD scenario로 구체화한다.

## 개발 운영과 품질 도구

| 영역 | 후보 | 결정 |
| --- | --- | --- |
| formatter | 언어별 표준 도구 | 미결정 |
| linter | 언어별 표준 도구 | 미결정 |
| type checker | 컴파일러·언어 도구 | 미결정 |
| schema validator | JSON Schema 또는 대안 | 미결정 |
| static analysis | 언어·보안별 도구 | 미결정 |
| unit/integration/contract test | 언어별 표준 도구 | 미결정 |
| property/fuzz test | 필요 범위 결정 | 미결정 |
| architecture test | import·dependency rule 검사 | 미결정 |
| coverage/mutation | 적용 범위와 gate | 미결정 |
| secret/dependency scan | 도구 조합 | 미결정 |
| CI | GitHub Actions 등 | 미결정 |

## 마일스톤·WBS 초안

| ID | 작업 | 선행 조건 | 상태 |
| --- | --- | --- | --- |
| M0 | 위협 모델·신뢰 경계·성공 기준 확정 | 없음 | 완료 |
| M1 | MVP Scope 확정 | M0 | 완료 (MVP-001~MVP-009) |
| M2 | 언어·CLI framework·runtime·배포 형태 결정 | M1 | 부분 완료 (Go·Cobra 확정, runtime·배포 미결정) |
| M3 | 최소 vertical slice와 디렉터리 구조 구현 | M2 | 미결정 |
| M4 | 첫 package manager의 다운로드·digest·검역 구현 | M3 | 미결정 |
| M5 | 최소 scanner·policy·receipt 구현 | M4 | 미결정 |
| M6 | promotion과 실패·승인 흐름 검증 | M5 | 미결정 |
| M7 | Docker image reproducibility·signing 검증 | M5 | 미결정 |
| M8 | 다른 package manager와 CI adapter 확장 | M6 | 미결정 |
| M9 | MCP adapter와 AI review contract 검토 | M6 | 미결정 |

## AI review 경계

AI review는 deterministic 검사 이전의 기본 단계가 아니라, 검사 결과와 프로젝트 맥락을 제공받아 자동 규칙의 사각지대를 검토하는 선택 단계로 둔다.

| 항목 | 상태 | 결정 |
| --- | --- | --- |
| AI review 입력 | 미결정 | 결과 receipt, policy finding, project context |
| AI review 출력 | 미결정 | 누락·위험·우선순위·수정 제안 |
| 자동 수정 권한 | 미결정 | 기본 금지 또는 사용자 승인 |
| 비용·지연 상한 | 미결정 |  |
| 민감정보 전달 정책 | 미결정 |  |

## 다음 결정 queue

- [x] MVP Scope 확정 (MVP-001~MVP-009)
- [x] CLI Host 지원 OS와 격리 검사 환경 분리 정책 확정
- [x] MVP 지원 Artifact 유형과 제외 범위 확정
- [x] MVP 정적 검사 범위와 Finding/Evidence 정규화 방향 확정
- [x] MVP 동적 설치·실행 검사 범위와 관찰 정책 확정
- [x] Signature·Provenance·무결성 검증 범위와 부재 처리 확정
- [x] SBOM·Evidence·결과물 범위와 Staging Manifest 방향 확정
- [x] MVP 사용자 흐름·Policy 분기·완료 기준 확정
- [x] 메인 언어 Go와 CLI framework Cobra 확정
- [ ] runtime과 배포 형태 결정
- [x] Architecture 설계 시작 (Architecture-001)
- [x] Architecture-002 최상위 책임 영역과 Policy 판정 책임 확정
- [x] Architecture-003 Dependency Direction과 Composition Root 확정
- [x] Architecture-004 Artifact/External Tool Adapter 구조와 Capability 정책 확정
- [x] Architecture-005 Sandbox/Inspection Backend 구조와 Session lifecycle 확정
- [x] Architecture-006 Verification/Inspection/Policy 책임 구조와 결과 모델 확정
- [x] Architecture-007 Evidence/Result 저장 구조와 Inspection Run 추적 확정
- [x] Architecture-008 Staging/Promotion 구조와 Verified Set·digest 재검증 확정
- [x] User-Journey-001 주요 사용자 Journey 확정
- [x] User-Journey-002 Artifact 입력 방식 확정
- [x] User-Journey-003 CLI 최상위 Command 구조 확정
- [x] User-Journey-004 기본 End-to-End 실행 흐름 확정
- [x] User-Journey-005 자동 실행과 사용자 개입 경계 확정
- [x] User-Journey-006 결과 표시와 상세 조회 Journey 확정
- [x] User-Journey-007 실패·재검사·재개 Journey 확정
- [x] User-Journey-008 설치·반입(Promotion) UX 확정
- [x] Domain-001 핵심 Domain Concept 및 역할 확정
- [x] Domain-002 Artifact Reference / Resolved Identity / Acquired Artifact 모델 확정
- [x] Domain-003 Inspection Run 모델 확정
- [x] Domain-004 Capability / Execution / Run / Policy / Operation 상태 모델 확정
- [x] Domain-005 Verification Result / Evidence / Finding 모델 확정
- [x] Domain-006 Dependency / Verified Set / Manifest 모델 확정
- [x] Domain-007 Operation Request / Install Context 모델 확정
- [ ] Contract-001 Artifact Port 결정
- [ ] Contract-002 Verification / Inspection / Sandbox / Policy Ports 결정
- [ ] Contract-003 Evidence / Staging / Promotion Ports 결정
- [x] 최초 지원 Artifact 생태계 선정: npm, Python/PyPI(pip), GitHub Releases
- [ ] 첫 번째 정상·악성·변조 fixture 확보
- [ ] quarantine 신뢰 경계와 반입 계약 정의
- [ ] checksum/digest/signature/provenance 결과 schema 정의
- [ ] 최소 deterministic policy 정의
- [ ] 도구 설치·업데이트의 supply-chain 정책 정의
- [ ] Docker image build와 서명 모델 결정

## 전체 누락 점검

- [ ] 라이선스와 상표·배포 정책
- [ ] 지원 버전과 호환성 정책
- [ ] 성능·저장공간·네트워크 예산
- [ ] artifact 보존·삭제·복구 정책
- [ ] 장애·중단·rollback 정책
- [ ] audit log와 개인정보 처리
- [ ] 위협 모형과 fixture 기반 보안 테스트
- [ ] 설치·업데이트·마이그레이션 경로
