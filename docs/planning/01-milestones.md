# Step 12 — Milestones

이 문서는 Step 1~11에서 확정한 범위와 의존 관계를 구현 가능한 milestone으로 배열하고 각 단계의 완료 조건을 정의한다. 상세 Domain schema, runtime, storage와 ecosystem 구현을 새로 확정하지 않으며, 해당 구현에 도달했을 때 필요한 canonical 문서를 읽고 좁은 결정을 내리도록 한다.

Milestone은 calendar 일정이나 작업 티켓이 아니다. Step 13 Work Queue가 현재 실행할 작은 작업과 우선순위를 관리하며, 이 문서는 구현 순서·경계·exit criteria의 canonical source다.

## 1. 첫 검증 사용자와 운영 형태

MVP의 첫 검증 사용자는 자신의 개발 환경에서 외부 Software Artifact를 설치하거나 검사하려는 **로컬 개발자/operator**로 둔다.

```text
Linux 또는 macOS Host의 개발자
        ↓
helox CLI를 명시적으로 실행
        ↓
inspect 또는 install 요청
        ↓
사람용 결과 확인

필요 시
        ↓
동일한 machine-readable 결과를 CI/automation이 소비
```

- 초기 제품 형태는 local CLI다.
- 사용자가 요청한 한 operation의 context와 결과를 보존한다.
- Linux와 macOS native CLI를 우선하고 Windows는 WSL2 qualification 후 공식 경로로 선언한다.
- CI/automation 소비는 지원하지만 local journey와 다른 보안 의미를 만들지 않는다.
- shared daemon, cloud service, 중앙 관리 server, 조직 전체 자동 install agent와 GUI는 MVP milestone에 포함하지 않는다.
- 실제 Host credential 또는 개인 project를 개발·CI fixture로 사용하지 않는다.

이 결정은 배포 대상 시장이나 최종 상용 운영 모델을 확정하는 것이 아니다. 첫 vertical slice와 MVP 검증 범위를 불필요하게 확장하지 않기 위한 구현 기준이다.

## 2. Milestone 원칙

### Vertical slice 우선

기술 계층을 모두 만든 뒤 연결하지 않고 사용자에게 의미 있는 가장 얇은 operation을 end-to-end로 연결한다.

```text
Fake 기반 inspect workflow
        ↓
npm static fail-closed inspect
        ↓
npm Linux dynamic inspect
        ↓
npm install + trusted Promotion
        ↓
PyPI/pip 확장
        ↓
GitHub Releases standalone 확장
```

### 신뢰 경계 순서

- `inspect`의 identity·acquisition·result·Policy 경계를 먼저 완성한다.
- 필수 검사가 없는 상태에서 임시 `ALLOW`를 만들지 않는다.
- Dynamic Inspection은 Linux isolation backend와 synthetic fixture가 준비된 뒤 활성화한다.
- Host Promotion은 exact identity/digest binding, Evidence, Policy와 Verified Manifest가 먼저 완성된 뒤 구현한다.
- package manager install보다 standalone direct Promotion을 먼저 구현해 Staging invariant를 우회하지 않는다.
- npm reference path가 end-to-end로 닫히기 전 PyPI/GitHub adapter를 병렬로 확장하지 않는다.

### 필요한 만큼만 생성

- milestone에 필요한 package·Port implementation·test fixture만 만든다.
- 미래 milestone의 placeholder package, empty interface, fake success job을 만들지 않는다.
- milestone 중 발견한 범용 요구를 ecosystem 전용 shortcut으로 Core에 넣지 않는다.
- 유보된 상세 결정은 해당 milestone의 entry decision으로 해결하고 기존 invariant를 변경해야 하면 설계 문서로 되돌아간다.

## 3. Milestone 개요

| ID | Milestone | Primary outcome | Depends on |
| --- | --- | --- | --- |
| M0 | Implementation Foundation | 단일 Go module, 최소 CLI와 local/CI quality foundation | Step 1~11 |
| M1 | Domain Workflow Skeleton | fake Port로 `inspect` lifecycle과 상태 축 연결 | M0 |
| M2 | npm Static Inspect | exact npm Artifact를 획득·정적 검사하고 fail-closed 결과 제공 | M1 |
| M3 | Linux Dynamic Inspect | Sandbox observation을 Finding/Evidence로 해석한 npm inspect 완성 | M2 |
| M4 | npm Install and Promotion | dependency 포함 Verified Set을 원래 install context로 안전하게 반입 | M3 |
| M5 | PyPI/pip Expansion | wheel·sdist가 동일 Core/Port workflow로 동작 | M4 |
| M6 | GitHub Releases Standalone | exact release asset의 binary/archive inspect와 standalone Promotion | M4 |
| M7 | MVP Qualification | 세 ecosystem, platform, security와 결과 계약의 MVP 완료 검증 | M5, M6 |

M5와 M6은 M4 이후 서로 독립적으로 진행할 수 있지만 Step 13은 기본적으로 하나씩 완료하여 공통 계약 변경 원인을 분명히 한다.

## 4. M0 — Implementation Foundation

### 목적

코드가 생성되는 첫 순간부터 Step 8~11의 module, dependency direction와 품질 규칙을 적용할 최소 기반을 만든다.

### Entry decisions

- module path와 `helox` CLI 이름의 실제 사용 가능성
- 구현 시점의 pinned Go patch와 최소 지원 Go version
- versioned GitHub-hosted runner label과 필요한 Action full SHA
- Staticcheck 등 최초 활성화할 tool의 exact version

### Scope

- project root의 단일 `go.mod`와 필요한 경우 `go.sum`
- `cmd/helox`, `internal/bootstrap`, `internal/cli`의 최소 실행 경로와 process smoke test. help/version은 필요한 경우 포함한다.
- standard-library 기반 check runner의 `format check`, `docs`, architecture foundation
- exact tool lock과 source tree 밖 bootstrap cache 계약
- 활성 검사가 존재하는 `Quick`, `Docs`, `Required` CI job

### Exit criteria

- `helox` binary가 Linux/macOS target에서 build된다.
- module root 밖이나 별도 `src/`/nested module이 생성되지 않는다.
- format, docs, module consistency, build와 활성화된 static check가 local/CI에서 같은 entrypoint로 통과한다.
- CI가 full-SHA action, 최소 token permission, no-secret PR 원칙을 지킨다.
- required aggregate가 실제 활성 child failure/cancel/skip을 실패로 처리한다.
- placeholder Domain/Port/Adapter package가 없다.
- exact version과 공급망 검토 근거가 기록된다.

### Out of scope

- Artifact parsing/acquisition
- Inspection Run과 Policy
- external registry access
- Sandbox, Evidence Store와 Promotion

## 5. M1 — Domain Workflow Skeleton

### 목적

외부 network와 Host 반입 없이 fake Port를 사용하여 하나의 `inspect` Operation Request가 Inspection Run과 Operation Result로 끝나는 lifecycle을 검증한다.

### Entry decisions

- 최소 Domain type/schema와 ID 생성 방식
- Application request/result와 Port method signature
- Execution Status, Run Outcome와 Operation Status transition API
- 최소 Policy input/rule와 reason representation
- CLI exit code와 machine-readable result의 최초 version

### Scope

- `core/domain`, `core/ports`, `application`, `policy`의 필요한 최소 package
- fake Artifact/Verification/Inspection/Evidence implementation
- Operation Request → exact fake identity → Run 생성 → acquired binding → normalized result → Policy → Operation Result orchestration
- `ALLOW`, `MANUAL_REVIEW`, `BLOCK`과 operational failure test
- `inspect + ALLOW`에서도 Promotion이 호출되지 않는 test
- 상태 축과 error wrapping/CLI mapping test
- architecture checker의 실제 import rule

### Exit criteria

- Domain이 ecosystem/tool/OS type 없이 lifecycle invariant를 표현한다.
- Inspection Run이 exact resolved identity 이후 acquisition 전에 생성된다.
- Operational Error, Domain Result, Policy Decision과 Operation Result가 type/API/test에서 분리된다.
- 필수 check 미충족은 `ALLOW`가 되지 않는다.
- fake workflow의 사람용·기계용 결과가 동일 Run/identity/decision을 참조한다.
- `inspect` workflow 어디에서도 Staging/Promotion이 생성·호출되지 않는다.
- unit, architecture, minimum Go, macOS와 race 중 활성화된 gate가 통과한다.

### Out of scope

- real npm/network/filesystem intake
- production Evidence storage
- Dynamic Inspection
- install/Promotion

## 6. M2 — npm Static Inspect

### 목적

실제 npm reference를 exact identity로 resolve하고 통제된 영역에 획득한 bytes를 Verification/Static Inspection/Policy로 연결하는 첫 실제 사용자 vertical slice를 만든다.

```text
helox npm inspect <package-reference>
```

### Entry decisions

- npm Artifact Reference/Resolved Identity mapping
- registry endpoint, redirect, timeout, size와 authentication-free MVP 범위
- Controlled Intake handle, root, permission와 cleanup
- npm tarball/manifest/dependency metadata의 bounded parser
- observed digest algorithm과 canonical format
- 최소 static check taxonomy, Evidence storage schema와 Policy requirement
- result schema/version과 Evidence reference

### Scope

- npm Identify/Resolve/Acquire Adapter
- exact version과 acquired content observed digest binding
- package/archive structure, path/link/special file/resource limit 검사
- registry declared integrity와 observed digest Verification
- lifecycle/install script와 dependency metadata static inspection
- trusted Evidence writer/store의 최소 production implementation
- Finding, performed/skipped check, Capability, Execution Status와 Limitation 결과
- Dynamic Inspection이 required지만 아직 없을 때 `MANUAL_REVIEW` 또는 Policy rule에 따른 `BLOCK`
- safe, malformed, digest mismatch, archive escape, timeout와 unavailable fixture

### Exit criteria

- mutable npm reference가 acquisition 전에 exact version으로 resolve된다.
- acquired bytes에서 observed digest를 직접 계산하고 declared integrity와 구분한다.
- registry success를 `ALLOW`로 해석하지 않는다.
- untrusted archive가 intake root를 벗어나거나 Evidence Store를 수정하지 못한다.
- static finding과 verifier operational failure가 다른 결과로 기록된다.
- Dynamic Inspection 부재를 빈 성공으로 숨기지 않고 자동 `ALLOW`하지 않는다.
- 사람용·기계용 inspect 결과에 Run, exact identity/digest, check 상태, Finding/Evidence와 Policy 근거가 연결된다.
- npm Adapter contract, malicious fixture와 network failure integration test가 통과한다.
- 필수 CI는 exact identity와 기대 결과가 고정된 controlled fixture를 사용하며, public registry 상태나 mutable reference에 성공 여부를 의존하지 않는다.

### Out of scope

- Artifact lifecycle 실행
- Linux Sandbox
- npm dependency 전체 acquisition/Verified Set
- Host install와 Promotion

## 7. M3 — Linux Dynamic Inspect

### 목적

Linux 격리 backend에서 npm Artifact의 제한된 lifecycle을 실행하고 raw Observation을 Inspection이 Evidence/Finding으로 해석하여 `npm inspect`의 전체 Policy 경로를 완성한다.

### Entry decisions

- Linux isolation runtime/backend와 exact version/image identity
- process, filesystem, DNS/network, honeytoken observation capability
- Sandbox request/session/observation schema
- CPU/memory/time/process/file/network limit
- synthetic filesystem, credential/honeytoken와 fake service fixture
- Dynamic check requirement와 Policy rule/version/reason code
- raw observation size/retention/redaction

### Scope

- ephemeral Sandbox Session lifecycle: create, prepare, introduce, execute, observe, terminate, dispose
- non-root/minimum privilege, Host filesystem·credential·internal network default deny
- bounded process tree, output와 resource cleanup
- Observation → normalized Evidence/Finding 해석
- credential access, unexpected process, filesystem mutation, DNS/network attempt와 timeout fixture
- `ALLOW`, `MANUAL_REVIEW`, `BLOCK` 대표 Policy scenario
- abnormal session 폐기와 fresh retry test

### Exit criteria

- Sandbox는 raw Observation/Execution Status만 생성하고 Policy/Finding을 직접 만들지 않는다.
- Inspection이 Observation을 Evidence/Finding으로 해석한다.
- 실제 Host secret/path/internal service가 untrusted Artifact에 노출되지 않는다.
- timeout, resource limit와 abnormal termination 후 process tree와 Session이 정리되고 재사용되지 않는다.
- required dynamic capability가 unsupported/unavailable/failed/incomplete이면 `ALLOW`하지 않는다.
- 정의된 required Verification/Inspection check가 모두 충족된 safe fixture에서만 `inspect + ALLOW`가 재현되며, suspicious fixture의 `MANUAL_REVIEW`와 malicious/integrity fixture의 `BLOCK` 결과가 재현된다.
- Linux integration/E2E와 scheduled fuzz/security fixture가 활성 CI에서 통과한다.

### Out of scope

- Host installation
- production dependency Verified Set/Staging
- macOS/Windows native dynamic backend

## 8. M4 — npm Install and Promotion

### 목적

npm primary Artifact와 dependency graph를 exact Verified Set으로 고정하고 `install + ALLOW`에서 원래 Install Context를 유지한 trusted Promotion을 완성한다.

```text
helox npm install <package-reference>
```

### Entry decisions

- Install Context schema, option allowlist, target/project 식별과 snapshot
- npm dependency graph/lock와 모든 실제 package acquisition 방식
- Verified Set/Manifest serialization과 validity binding
- Staging handle/layout, immutability/change detection와 retention
- npm offline/local install과 network 차단 enforcement
- target permission, overwrite, atomicity, rollback와 partial failure
- Operation Result/Promotion Record/Evidence 관계
- SBOM format/provider와 output contract

### Scope

- primary/dependency exact identity, acquisition, verification와 inspection 반복
- 새로운 dependency 발견 시 workflow 재진입
- `ALLOW`된 complete set의 Verified Manifest 생성
- Quarantine → Staging, Staging → Host 두 경계 digest 재확인
- original target/mode/options를 유지한 trusted npm Promotion
- install 중 network/new Artifact 요구 시 STOP
- `MANUAL_REVIEW`/`BLOCK`의 no-Staging/no-Promotion
- Promotion success, permission failure, partial failure와 cleanup Result
- Manifest, SBOM, Evidence와 사람용·기계용 결과

### Exit criteria

- npm 사용자 지정부터 install과 Operation Result까지 reference end-to-end가 완성된다.
- 검사한 set, Policy가 허용한 set, Manifest, Staging과 Promotion 대상이 identity/digest로 동일하다.
- Promotion이 mutable reference를 다시 resolve하거나 Manifest 밖 dependency를 추가하지 않는다.
- Sandbox/Artifact Adapter가 Host에 직접 쓰지 않는다.
- `ALLOW + inspect`에는 Promotion이 없고 `ALLOW + install`만 원래 operation을 계속한다.
- Promotion operational failure가 기존 Policy `ALLOW`를 `BLOCK`으로 변경하지 않는다.
- disposable target의 success/failure/rollback E2E와 Promotion contract test가 통과한다.
- npm reference implementation에 필요한 result, Evidence, SBOM와 cleanup 기준이 충족된다.

### Out of scope

- arbitrary npm option passthrough
- global/system-wide privileged install
- PyPI와 GitHub Releases implementation

## 9. M5 — PyPI/pip Expansion

### 목적

두 번째 package ecosystem을 기존 Core/Port 의미를 깨뜨리지 않고 연결하여 adapter 확장성을 실제로 검증한다.

### Entry decisions

- PyPI project/version와 wheel/sdist exact identity mapping
- distribution/filename/platform/architecture 선택 규칙
- registry metadata, digest와 provenance source
- pip dependency resolution/lock와 offline install 방식
- sdist build/lifecycle의 Linux Sandbox execution
- capability/limitation mapping

### Scope

- PyPI Identify/Resolve/Acquire와 dependency graph Adapter
- wheel structure/static/dynamic inspection
- sdist를 executable install/build Artifact로 격리 처리
- Verification/Evidence/Policy 공통 workflow
- Verified Set/Staging/pip Promotion
- wheel, sdist, digest mismatch, malicious build와 unsupported platform fixture
- Artifact Port와 Promotion contract suite 재사용

### Exit criteria

- Core/Application/Policy에 PyPI 전용 type/branch가 추가되지 않는다.
- wheel과 sdist가 exact identity/observed digest에 binding된다.
- sdist build code가 Host가 아닌 Sandbox에서만 실행된다.
- dependency도 primary와 동일한 verification/inspection pipeline을 거친다.
- unsupported platform/dynamic capability가 자동 `ALLOW`로 변환되지 않는다.
- 정의된 PyPI inspect/install 흐름과 common contract test가 통과한다.
- npm 기존 E2E가 변경 없이 계속 통과한다.

### Out of scope

- 모든 Python environment manager
- arbitrary build backend credential/network
- Windows/macOS native wheel dynamic behavior 보증

## 10. M6 — GitHub Releases Standalone

### 목적

package manager가 없는 exact release asset을 동일 Artifact/Policy 계약으로 검사하고 standalone exception에 맞게 trusted target으로 반입한다.

### Entry decisions

- repository/release/tag/asset exact selector와 ambiguity 처리
- GitHub API authentication-free/rate-limit 범위와 optional trusted auth
- checksum/signature/provenance asset 연결 규칙
- executable/zip/tar.gz format detection와 nested archive limit
- platform/architecture asset selection과 target context
- direct Promotion atomic write/overwrite/permission/rollback

### Scope

- GitHub Release Identify/Resolve/Acquire Adapter
- mutable latest/tag와 복수 asset의 exact resolution
- binary/archive Verification, Static/Dynamic Inspection와 Evidence
- content type/extension mismatch, path/link/nested archive 검사
- `inspect` 결과와 `install + ALLOW` standalone direct Promotion
- Promotion 직전 exact identity/digest 재확인
- ambiguous asset, digest mismatch, archive escape와 unsupported platform fixture

### Exit criteria

- package manager 개념을 Core contract에 추가하지 않고 standalone flow가 동작한다.
- asset이 exact repository/release/tag/name/platform identity와 observed digest에 binding된다.
- 복수 asset을 임의 선택하거나 mutable reference를 Promotion 시 재resolve하지 않는다.
- standalone은 Staging을 생략할 수 있지만 검사/ALLOW/Promotion content 동일성을 유지한다.
- Sandbox가 target path에 직접 쓰지 않는다.
- inspect/install, ALLOW/MANUAL_REVIEW/BLOCK와 Promotion failure E2E가 통과한다.
- npm/PyPI contract와 의미가 깨지지 않는다.

### Out of scope

- generic direct URL/local file
- Git repository clone
- GitHub Actions/CI component와 container image

## 11. M7 — MVP Qualification

### 목적

세 ecosystem과 지원 platform에서 기능, 보안 경계, 결과 추적성과 운영 가능성이 MVP 완료 기준을 충족하는지 검증한다.

### Scope

- npm full end-to-end regression
- PyPI/pip와 GitHub Releases common contract/flow regression
- Linux/macOS native CLI build·default E2E
- disposable WSL2 CLI qualification
- Linux Dynamic Inspection capability/limitation matrix
- safe, suspicious, malicious, corrupt, unavailable, timeout와 resource exhaustion fixture matrix
- Evidence/Result/SBOM/Verified Manifest reference integrity
- cleanup, retention, interrupted run와 no-secret verification
- local check, required CI, scheduled vulnerability/secret/fuzz status
- installation, troubleshooting, limitation와 security reporting documentation

### Exit criteria

- npm에서 resolve/acquire/verify/static/dynamic inspect/policy/evidence/install/promotion/result 전체 흐름이 완료된다.
- PyPI/pip와 GitHub Releases가 동일 Core와 기존 Port 의미로 정의된 MVP 흐름을 수행한다.
- 생태계 추가를 위해 Core에 ecosystem 전용 logic이 필요하지 않다.
- `ALLOW / MANUAL_REVIEW / BLOCK`과 `COMPLETED / FAILED / PAUSED / NOT_PERFORMED`가 대표 scenario에서 정확하다.
- 필수 검사 unavailable/unsupported/failed/incomplete 상태가 자동 `ALLOW`되지 않는다.
- 실제 검사한 content와 Promotion content의 identity/digest binding이 모든 경로에서 유지된다.
- Linux/macOS CLI와 WSL2 공식 경로가 각각 실제 qualification evidence를 가진다.
- required CI와 scheduled security workflow가 활성 범위에서 green이다.
- 실제 Secret, 개인 Host asset 또는 mutable external fixture가 test/CI에 없다.
- 알려진 Critical/High security defect와 unresolved trust-boundary violation이 없다.
- 지원 범위와 검사 한계가 사용자에게 공개된다.

M7 통과 전에는 MVP 완료, 안전한 일반 사용 또는 공식 WSL2 지원을 선언하지 않는다.

## 12. Cross-cutting track

다음 항목은 별도 최종 milestone로 미루지 않고 관련 capability가 처음 등장하는 milestone에서 함께 구현한다.

| Track | Activation |
| --- | --- |
| Documentation routing | 모든 milestone 문서/contract 변경 시 |
| Architecture check | M1 package dependency 생성 시 |
| Security source/secret scan | 각 pinned tool이 준비되는 즉시 |
| Minimum Go/macOS CI | build/test code가 존재하는 M0~M1 |
| Race | concurrent ownership code가 등장하는 milestone |
| Fuzz | parser/path/archive/decoder가 등장하는 M2 이후 |
| Evidence integrity/redaction | M2 production Evidence부터 |
| Integration/E2E | concrete network/runtime/storage/Promotion 경계 등장 시 |
| Dependency review | product/tool dependency 추가·upgrade 시 |
| User/machine output compatibility | M1 최초 schema부터 |

cross-cutting 검사를 나중의 “hardening 단계” 하나로 몰아 security debt를 정상 상태로 만들지 않는다.

## 13. Global Definition of Ready

milestone 또는 그 안의 work item은 다음 조건을 충족한 뒤 시작한다.

- 필요한 upstream milestone exit criteria가 충족됨
- 작업에 필요한 canonical document route가 식별됨
- unresolved decision이 결과를 크게 바꾸면 entry decision으로 분리됨
- external tool/runtime/source의 trust·version·license·network 요구가 식별됨
- test fixture가 synthetic/sanitized이며 expected result가 정의됨
- Host write, credential, network 또는 destructive behavior가 있다면 exact target과 rollback/cleanup 경계가 정의됨
- 완료를 증명할 deterministic test/check가 정의됨

조건을 충족하지 못한 항목을 코드로 추측하여 시작하지 않는다.

## 14. Global Definition of Done

모든 milestone은 개별 exit criteria와 함께 다음을 충족해야 한다.

- scope에 포함된 실제 사용자/contract path가 stub 없이 동작함
- 정상, 거부, operational failure와 cleanup test가 있음
- 새 Port/Adapter가 contract test를 통과함
- Architecture dependency direction이 자동 검사됨
- format, build, vet, Staticcheck, test와 활성 security gate가 통과함
- required capability가 skipped/unsupported/unavailable인데 success로 표시되지 않음
- 실제 Secret·credential·개인 path와 raw unbounded output이 없음
- 코드와 함께 canonical leaf 문서·router·status가 필요한 범위에서 갱신됨
- 유보/제한/known risk가 사용자에게 보이게 기록됨
- milestone completion evidence로 commit, test/check와 대표 scenario 결과를 참조할 수 있음

다음은 완료로 인정하지 않는다.

```text
compile만 성공
happy path demo만 성공
milestone이 실제 boundary를 요구하는데 fake implementation 결과만 존재
required test를 skip
scanner no-fail
실제 Host에서 수동으로 한 번 동작
미구현 capability를 빈 결과로 반환
다음 milestone이 현재 invariant를 대신 보완해야 함
```

## 15. Status와 변경 관리

Milestone status는 다음 네 값만 사용한다.

```text
NOT_STARTED
IN_PROGRESS
BLOCKED
COMPLETE
```

- Step 12 확정 시 모든 구현 milestone은 `NOT_STARTED`다.
- Step 13 Current Work Queue가 현재 milestone과 concrete work item을 표시한다.
- 기본적으로 하나의 milestone만 `IN_PROGRESS`로 둔다.
- exit criteria 일부만 충족한 상태를 percentage로 `COMPLETE`처럼 표시하지 않는다.
- `BLOCKED`에는 exact blocker, 마지막 검증과 unblock condition을 기록한다.
- milestone scope 변경이 MVP 범위나 invariant를 바꾸면 planning 문서만 수정하지 않고 canonical 설계 문서를 먼저 갱신한다.
- 완료된 milestone의 regression은 새 work item으로 즉시 처리하며 과거 완료 evidence를 삭제하지 않는다.

## Step 12 Invariant

1. milestone은 Architecture layer가 아니라 end-to-end capability와 신뢰 경계 순서로 구성한다.
2. fake workflow, npm static inspect, dynamic inspect와 install/Promotion을 순차적으로 닫는다.
3. 필수 검사 미구현 상태를 임시 `ALLOW`로 연결하지 않는다.
4. npm reference end-to-end를 완성한 뒤 PyPI와 GitHub Releases로 공통 계약을 검증한다.
5. Host Promotion은 exact identity/digest, Policy, Evidence와 Manifest 경계가 준비된 뒤 구현한다.
6. standalone 예외가 package Verified Set/Staging invariant를 약화하지 않는다.
7. security, test, documentation과 CI를 마지막 hardening milestone로 미루지 않는다.
8. milestone 완료는 exit criteria와 재현 가능한 evidence로만 판정한다.
9. 유보된 구현 결정을 code shortcut으로 추측하지 않고 해당 milestone entry decision으로 해결한다.
10. M7이 npm full E2E와 PyPI/GitHub common Core 확장성 기준을 모두 충족해야 MVP 완료다.

## Step 13으로 전달할 항목

Step 13은 M0만 실행 가능한 작은 work item으로 분해한다.

- module/CLI identity 결정
- Go/runner/action/tool exact version 조사와 pin
- `go.mod`와 최소 `cmd/helox` bootstrap
- standard-library check runner의 최소 profile
- docs/architecture checker
- Quick/Docs/Required CI foundation
- M1 entry decision을 위한 schema/API design task

M1 이후 항목을 현재 queue의 실행 작업으로 미리 펼치지 않는다. milestone이 가까워질 때 관련 문서를 다시 읽고 entry decision과 fixture를 확정한다.

## 누락 점검

- [x] 첫 검증 사용자와 local CLI 운영 형태
- [x] vertical slice와 신뢰 경계 구현 순서
- [x] 8개 milestone과 dependency
- [x] M0 implementation/quality foundation
- [x] M1 fake 기반 Domain workflow
- [x] M2 npm static fail-closed inspect
- [x] M3 Linux dynamic inspect
- [x] M4 npm dependency/install/Promotion full E2E
- [x] M5 PyPI wheel/sdist expansion
- [x] M6 GitHub Releases standalone path
- [x] M7 MVP qualification
- [x] cross-cutting security/test/docs/CI activation
- [x] Global Definition of Ready/Done
- [x] milestone status와 변경 관리
- [x] Step 13 전달 범위
