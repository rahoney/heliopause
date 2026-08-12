# Step 8 — Directory Structure

이 문서는 Step 1~7에서 확정한 Architecture, Domain Model과 Interface Contract를 Go module·package·파일 배치 규칙으로 구체화한다. 기존 결정을 변경하지 않으며 실제 scaffold는 Step 14 Implementation에서 필요한 범위만 생성한다.

## 결정

### Module과 실행 단위

- 현재 Heliopause 프로젝트 디렉터리 자체를 단일 Go module root로 사용한다.
- `go.mod`와 `go.sum`은 프로젝트 루트에 둔다.
- 별도의 `src/`, 하위 Go 프로젝트 디렉터리 또는 중첩 Go module을 만들지 않는다.
- MVP의 Heliopause 본체는 단일 `helox` binary다.
- 실행 진입점은 `cmd/helox/main.go`에 둔다.
- Linux와 macOS에서 native Go CLI로 실행하고 Windows 공식 경로는 WSL2다.
- Dynamic Inspection은 Linux 격리 backend를 전제로 하지만 구체 runtime, container/VM 기술과 배포 방식은 이 단계에서 고정하지 않는다.
- scanner, verifier와 sandbox runtime은 Port 뒤에서 별도 process·tool로 실행될 수 있다. 이를 별도 Heliopause service나 Go module로 취급하지 않는다.

### 목표 구조

아래 구조는 책임과 배치 규칙을 나타낸다. Step 8에서는 디렉터리나 placeholder를 만들지 않고, Step 14에서 실제 코드가 필요해질 때만 생성한다.

```text
.
├─ go.mod
├─ go.sum
├─ cmd/
│  └─ helox/
│     └─ main.go
│
├─ internal/
│  ├─ bootstrap/
│  ├─ cli/
│  ├─ application/
│  ├─ core/
│  │  ├─ domain/
│  │  └─ ports/
│  ├─ policy/
│  ├─ artifact/
│  │  ├─ npm/
│  │  ├─ pypi/
│  │  └─ githubreleases/
│  ├─ verification/
│  ├─ inspection/
│  ├─ sandbox/
│  ├─ evidence/
│  ├─ promotion/
│  └─ testutil/
│
├─ test/
│  ├─ contract/
│  ├─ integration/
│  └─ e2e/
├─ testdata/
├─ schemas/
├─ examples/
├─ scripts/
└─ docs/
```

표시된 모든 하위 디렉터리를 처음부터 만들 필요는 없다. 책임이 아직 하나의 응집된 package로 충분하면 더 세분화하지 않는다.

## Package 책임

| Path | Responsibility | Must not own |
| --- | --- | --- |
| `cmd/helox` | process entrypoint; bootstrap 호출과 exit 처리 | workflow, 검사, Policy, Adapter 구현 |
| `internal/bootstrap` | Composition Root; 구체 구현 생성·등록·wiring | business rule, 검사 해석, Policy rule |
| `internal/cli` | Cobra command, 입력 변환, terminal/machine output | Adapter 직접 호출, 검사·판정 로직 |
| `internal/application` | install·inspect·review use case와 workflow orchestration | 구체 npm/scanner/runtime/storage 구현 생성·참조 |
| `internal/core/domain` | 생태계·도구·OS 비종속 type, invariant, 상태 모델 | Cobra, filesystem/database, 외부 SDK·도구 의존 |
| `internal/core/ports` | Artifact·Verification·Inspection·Sandbox·Evidence·Staging·Promotion 계약 | 구체 backend·provider 구현 |
| `internal/policy` | 정규화된 Domain Result를 평가한 최종 Policy Decision | Artifact 획득, scanner·sandbox 호출, Promotion |
| `internal/artifact/*` | 생태계별 parse·resolve·acquire·dependency resolution Adapter | Policy, Host installation, Sandbox lifecycle |
| `internal/verification` | verifier provider와 Verification Result/Evidence 정규화 | 최종 Policy Decision |
| `internal/inspection` | static/dynamic 검사와 Observation의 Evidence/Finding 해석 | Sandbox backend 구현, 최종 Policy Decision |
| `internal/sandbox` | Linux 격리 backend Adapter, Session lifecycle와 raw Observation | Finding 해석, Policy Decision, Host Promotion |
| `internal/evidence` | Evidence/Result/SBOM 저장 Adapter와 trusted writer | Finding 해석, Staging, Promotion |
| `internal/promotion` | Staging과 trusted Promotion Adapter | Policy 재판정, 재검사, 자유로운 dependency resolve |
| `internal/testutil` | 여러 package가 공유하는 test-only helper | production workflow 또는 Domain rule |

`verification`, `inspection`, `sandbox`, `evidence`, `promotion` 아래의 provider/backend/storage 하위 package는 구현이 둘 이상이거나 외부 의존성 격리가 실제로 필요할 때만 만든다. 단일 구현을 미리 `providers/default`, `backends/linux`처럼 중첩하지 않는다.

## 의존성 방향

```text
cmd/helox
    ↓
bootstrap ───────────────→ concrete Adapter / Provider / Backend
    ↓                                      │
cli → application → core/ports ←──────────┘
                    ↓
               core/domain

policy → core/domain
```

Dynamic Inspection의 허용 경로는 다음과 같다.

```text
application → Inspection Port
inspection  → Sandbox Port
sandbox     → Sandbox Port 구현 + core/domain
```

### 허용 규칙

- `core/domain`은 원칙적으로 Go standard library와 자기 package 내부 type에만 의존하며, 다른 Heliopause product package를 import하지 않는다.
- `core/ports`는 `core/domain`에 의존할 수 있다.
- `policy`는 `core/domain`에 의존한다.
- `application`은 `core/domain`, `core/ports`, `policy`의 안정적인 API에 의존한다.
- `cli`는 Application use case와 CLI presentation type에 의존한다.
- Adapter/Provider/Backend는 자신이 구현하는 `core/ports`와 필요한 `core/domain` type에 의존한다.
- `bootstrap`은 모든 구체 package를 import하고 연결할 수 있는 유일한 Composition Root다.

### 금지 규칙

- `core/domain`과 `core/ports`는 Cobra, npm/PyPI/GitHub SDK, scanner, runtime, filesystem/database 구현을 import하지 않는다.
- `application`은 `artifact/npm`, 특정 scanner, sandbox backend, Evidence storage, Promotion 구현을 직접 import하지 않는다.
- `policy`는 Adapter, verifier, inspector, sandbox 또는 Promotion을 호출하지 않는다.
- `cli`와 `cmd/helox`는 검사·Policy·Promotion business rule을 구현하지 않는다.
- Adapter/Provider/Backend는 최종 Policy Decision을 생성하거나 다음 workflow 단계를 임의로 시작하지 않는다.
- `inspection`은 구체 sandbox backend를 직접 import하지 않고 Sandbox Port에 의존한다.
- package 간 순환 import를 허용하지 않는다.

위 규칙은 Step 10의 architecture test 또는 import rule과 Step 11의 Quality Gate에서 자동 검증한다.

## 책임 영역별 배치

### CLI와 Composition Root

- `cmd/helox/main.go`는 가능한 한 얇게 유지하고 `internal/bootstrap`이 구성한 실행기를 호출한다.
- Cobra root/subcommand, flag parsing과 출력 presentation은 `internal/cli`에 둔다.
- CLI가 선택한 operation은 Application request로 변환한 뒤 Application use case를 호출한다.
- process signal, cancellation, exit code mapping의 정확한 소유권은 Step 9에서 정하되 Domain/Policy 상태와 혼합하지 않는다.

### Application과 Core

- install, inspect, review 같은 use case는 `internal/application`에 둔다.
- Application package가 커지면 기술 계층이 아니라 use case 응집도를 기준으로 하위 package를 만든다.
- Domain Entity, Value Object, Result Record와 상태 type은 `internal/core/domain`에 둔다.
- Port interface와 경계 request/result type은 `internal/core/ports`에 둔다. 다만 여러 Port가 공유하는 생태계 중립 type은 `core/domain`을 재사용한다.
- interface는 소비자 또는 안정적인 계약 경계에 두고 구현 package에 중복 선언하지 않는다.

### Artifact Adapter

- npm, PyPI, GitHub Releases의 생태계 전용 parsing·API·metadata·dependency resolution은 각각 `internal/artifact/npm`, `pypi`, `githubreleases`에 둔다.
- 공통 Artifact identity와 result는 `core/domain`, 계약은 `core/ports`를 사용한다.
- 생태계별 구현 간 공유 코드가 실제로 생기기 전에는 `artifact/common`을 만들지 않는다.

### Verification, Inspection과 Sandbox

- external verifier 호출과 result normalization은 `internal/verification`에 둔다.
- static/dynamic inspection 및 raw observation 해석은 `internal/inspection`에 둔다.
- Linux isolation backend, Session 생성·종료·폐기와 observation capture는 `internal/sandbox`에 둔다.
- 구체 runtime 선택 전에는 Docker, containerd, VM 이름을 package 경계나 공통 contract에 넣지 않는다.

### Policy, Evidence와 Promotion

- Policy evaluation은 `internal/policy`에 두고 normalized Domain Result만 입력으로 받는다.
- Evidence Store, Result/SBOM serialization과 trusted writer 구현은 `internal/evidence`에 둔다.
- Evidence Store와 Staging storage는 같은 구현 기술을 쓰더라도 package와 권한 경계를 분리한다.
- Staging 및 Host 반입 구현은 `internal/promotion`에 둔다.
- package ecosystem과 standalone의 Promotion 차이는 `core/ports` 계약과 `promotion` 구현 안에서 다루며 Sandbox에 Host write 권한을 주지 않는다.

## Test와 Fixture 배치

### Unit test

- unit test는 대상 package 옆의 `*_test.go`에 둔다.
- black-box API 검증이 가능하면 외부 test package(`package name_test`)를 우선하고, 내부 세부사항 검증이 필요한 경우만 동일 package를 사용한다.
- package 전용 fixture는 해당 package의 `testdata/`에 둔다.

### Cross-boundary test

| Path | Purpose |
| --- | --- |
| `test/contract` | Port 구현이 공통 Contract를 준수하는지 검증 |
| `test/integration` | 실제 또는 통제된 외부 tool/storage/runtime 경계 조합 검증 |
| `test/e2e` | `helox` binary 기준 사용자 operation과 결과 검증 |

- 여러 package가 공유하는 Go test helper만 `internal/testutil`에 둔다.
- 공유 helper가 production code에서 import되지 않도록 Step 10에서 검사한다.
- repo-wide 정상·악성·변조 Artifact fixture는 루트 `testdata/`에 둔다.
- fixture는 ecosystem과 의도를 드러내는 하위 경로를 사용하고 출처·생성 방법·expected behavior를 함께 기록한다.
- 비밀값, 실제 credential, 개인 프로젝트 사본을 fixture로 커밋하지 않는다.

## Schema, Example과 Script

- 공개 machine-readable contract schema는 루트 `schemas/`에 둔다.
- 특정 package 내부에서만 쓰는 schema는 해당 package 가까이에 둔다.
- 사용자용 sanitized config와 output 예시는 `examples/`에 둔다.
- build, generation, fixture 준비와 deterministic validation 보조 명령은 `scripts/`에 둔다.
- 일반 workflow logic을 script에 숨기지 않는다. 제품 동작은 Go package와 `helox` command에 둔다.
- generated file은 생성 원본과 가까이 두고 generated marker와 재생성 명령을 기록한다. 정확한 generation·commit 정책은 Step 9~10에서 정한다.

아직 schema, example 또는 script가 필요하지 않으면 해당 디렉터리를 생성하지 않는다.

## Runtime Data 경계

- Evidence Store, quarantine/intake, Sandbox working state, Staging과 cache는 소스 트리 안에 저장하지 않는다.
- production runtime data는 OS별 application data/cache/runtime 경로 또는 사용자가 명시한 trusted 경로를 사용한다.
- `go.mod`가 있는 프로젝트 루트 아래에 `.helox/evidence`, `.helox/staging` 같은 기본 저장소를 자동 생성하지 않는다.
- Evidence, quarantine, Sandbox temporary state와 Staging은 논리적·권한적으로 분리한다.
- test는 `t.TempDir()` 또는 test runner가 제공하는 격리 임시 경로를 사용한다.
- 정확한 path, permission, retention, cleanup과 atomicity는 관련 후속 설계와 구현 직전 결정으로 유보한다.

## Public API와 `internal`

- MVP 구현은 기본적으로 `internal/` 아래에 둔다.
- 외부 소비자를 위한 루트-level Go library package를 만들지 않는다.
- 재사용 가능성만으로 `pkg/`를 만들지 않는다.
- 실제 외부 Go API 요구와 compatibility 정책이 확정될 때만 별도 public package를 결정한다.
- Heliopause 내부의 package 간 공개 identifier는 필요한 최소 범위로 유지한다.

## 생성 시점 규칙

- Step 8에서는 문서만 확정하고 Go scaffold와 빈 디렉터리를 만들지 않는다.
- Step 14에서 첫 vertical slice가 요구하는 디렉터리와 package만 생성한다.
- 목표 구조에 표시됐다는 이유만으로 placeholder type, empty interface 또는 미래용 abstraction을 만들지 않는다.
- 새 package는 독립 책임, 별도 외부 의존성, 보안 권한 경계 또는 import cycle 해소 중 하나의 구체적 이유가 있을 때 만든다.
- package가 과도하게 커지기 전까지 책임 영역별 단일 package를 유지하고 실제 변경 축에 따라 분할한다.

## First Vertical Slice의 최소 구조

첫 구현은 전체 목표 트리를 한 번에 만들지 않고 다음 최소 구조에서 시작한다.

```text
go.mod
cmd/helox/main.go
internal/bootstrap/
internal/cli/
internal/application/
internal/core/domain/
internal/core/ports/
```

첫 use case가 선택되면 필요한 Adapter와 결과 경계만 추가한다. 예를 들어 완전한 npm inspect vertical slice라면 `internal/artifact/npm`과 실제로 필요한 Verification·Inspection·Policy·Evidence package를 생성한다. Dynamic Inspection이 아직 필요하지 않으면 Sandbox 구현 package를, Host 반입이 없으므로 Promotion package를 미리 만들지 않는다.

## 유보 사항

다음은 Step 8에서 고정하지 않는다.

- Go module path
- 구체 Linux isolation/container/VM runtime
- runtime image와 CLI의 배포·업데이트 방식
- source authentication과 credential broker 구조
- 구체 scanner/verifier와 process protocol
- Evidence/Result storage engine과 filesystem layout
- Staging filesystem·immutable·atomic Promotion 구현
- config format과 환경변수 전달 정책
- public schema의 실제 종류와 versioning
- generated file·vendoring 정책의 세부사항
- 최초 vertical slice의 정확한 use case와 package 생성 순서

유보 사항은 Step 9~13 또는 해당 구현 직전 상세 결정에서 확정한다. 이 유보는 위의 책임 경계와 의존성 방향을 변경하지 않는다.

## Step 8 Invariant

1. 프로젝트 디렉터리는 단일 Go module root이며 별도 `src/`를 두지 않는다.
2. MVP Heliopause 본체는 `cmd/helox`의 단일 binary다.
3. 모든 product implementation은 기본적으로 `internal/`에 둔다.
4. 구조는 CLI / Application / Core / Artifact / Verification / Inspection / Sandbox / Policy / Evidence / Promotion 책임을 따른다.
5. Core와 Application은 구체 ecosystem·tool·runtime·storage 구현에 의존하지 않는다.
6. `bootstrap`만 구체 구현을 생성하고 wiring한다.
7. package를 미래 가능성만으로 미리 세분화하지 않는다.
8. unit test는 package 인접, cross-boundary test는 루트 `test/`, 공유 fixture는 `testdata/`에 둔다.
9. runtime data와 untrusted Artifact를 source tree에 저장하지 않는다.
10. Evidence, Sandbox/Quarantine, Staging과 Host Promotion의 권한 경계를 디렉터리 구조에서도 혼합하지 않는다.
11. Step 8에서는 scaffold를 만들지 않고 Step 14에서 필요한 package만 생성한다.
12. 구체 runtime·배포 기술을 package 이름이나 Core contract에 선반영하지 않는다.

## 구현 영향

- Step 14는 프로젝트 루트에 `go.mod`와 최소 vertical slice 구조를 생성하는 것으로 시작한다.
- Step 9는 이 package 경계에 적용할 coding·security rule을 정의한다.
- Step 10은 import direction, formatting, lint, test와 security scan을 자동 검증하는 도구를 결정한다.
- Step 11은 Step 10의 deterministic 검증을 CI Quality Gate로 연결한다.
- Step 12~13은 이 구조를 기준으로 milestone과 실제 work queue를 작성한다.

## 누락 점검

- [x] Go module root와 `go.mod` 위치
- [x] 별도 `src/` 및 nested module 금지
- [x] 단일 `helox` binary와 외부 process/tool 허용
- [x] Linux/macOS CLI·WSL2·Linux inspection backend 전제
- [x] 목표 디렉터리 구조
- [x] 책임별 package 소유권과 금지 책임
- [x] Composition Root 위치
- [x] 허용·금지 dependency direction
- [x] Artifact ecosystem Adapter 배치
- [x] Verification·Inspection·Sandbox 분리
- [x] Policy·Evidence·Promotion 분리
- [x] unit·contract·integration·E2E test 배치
- [x] fixture·schema·example·script 배치
- [x] runtime data의 source tree 외부 원칙
- [x] `internal/` 기본과 public package 유보
- [x] 과도한 package 선분할 금지
- [x] Step 14의 지연 scaffold 원칙
- [x] 최초 vertical slice 최소 구조
- [x] 구체 runtime·배포·storage·schema 유보
- [x] 12개 invariant
