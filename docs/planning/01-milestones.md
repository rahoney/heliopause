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
| M8 | Production Trust Hardening | external review에서 확인된 Host command·observer·privilege·observation trust gap remediation | M7 |
| M9 | Product Install UX | transactional default npm project·Python environment install와 간단한 CLI | M8 |
| M10 | Verified Distribution & Bootstrap | signed/attested native release, 필요한 runtime image와 trusted installer | M9 |
| M11 | Dynamic Detection Depth | production observation boundary 위의 richer bounded behavior detection | M10 |
| M12 | Ecosystem Expansion Before Public Release | M11 완료 후 PyTorch·Go·Cargo·Terraform 지원과 전체 qualification | M11 |
| M13 | Production Release & Operations | M12 및 M12-02 완료 후 보호된 운영 설정과 최종 public release | M12, M12-02 |

M5와 M6은 M4 이후 서로 독립적으로 진행할 수 있지만 Step 13은 기본적으로 하나씩 완료하여 공통 계약 변경 원인을 분명히 한다. M8 이후에는 trust correctness → usable product UX → verified bootstrap → detection sophistication → production release operations 순서를 유지한다. M10의 배포 workflow 구현과 M12의 실제 release activation/publication은 분리한다.

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

## 12. M8 — Production Trust Hardening

### 목적

M0~M7 qualification evidence를 보존하면서 external security review에서 확인된
Host command, gVisor observer, privileged firewall과 production observation
trust gap을 production-ready release 전에 닫는다.

### Scope

- absolute identity와 minimal environment를 사용하는 trusted Host tool executor
- 검증된 local Docker daemon endpoint와 actual registered runsc identity 확인
- HAA-owned observer helper supervisor, exclusive fixed endpoint와 single
  process-scoped observer composition
- per-connection container identity latch와 actual production observation
  completeness
- narrow typed API만 제공하는 least-privilege resolver network-policy helper
- runtime lock의 code/CI/runtime single source of truth
- hostile PATH/environment/socket/privilege/cleanup regression과 Linux
  requalification

### Exit criteria

- ambient PATH, Docker context, proxy 또는 remote daemon이 trusted execution을
  바꾸지 못한다.
- helper/observer lifecycle과 fixed endpoint ownership이 HAA에 의해 검증되고
  concurrent ownership 또는 helper failure가 fail-closed다.
- production transport가 만들지 않는 signal이 clean result나 `ALLOW`의 근거가
  되지 않는다.
- 일반 `helox` process는 root가 아니며 firewall helper는 arbitrary command를
  실행할 수 없다.
- runtime lock drift와 production composition regression이 required CI에서
  차단된다.
- M8 security qualification이 green이기 전에는 production-ready release를
  선언하지 않는다.

### Out of scope

- multi-process Host observer multiplexing service
- 일반 remote Docker daemon
- current project/active venv install UX
- public release와 runtime installer
- M11 수준의 richer behavior analysis

## 13. M9 — Product Install UX

### 목적

MVP security engine 위에 원래 사용자 여정인 간단한 install command를 제공하되
검사한 exact set과 실제 사용자 환경 변경의 transaction/rollback invariant를
유지한다.

### Scope

- `install plan freeze → mutate → verify → commit / rollback` 공통 계약
- npm current project의 `package.json`, lockfile와 `node_modules` 일관성
- active Python virtual environment의 site-packages, dist-info, RECORD와 script
  일관성
- canonical `helox pip install` UX와 internal `pypi` source ID 분리
- root help의 complete static command tree와 help side-effect 제거
- default install context detection과 optional advanced `--target`
- GitHub standalone safe default destination와 overwrite policy

### Exit criteria

- `helox npm install <package>`와 `helox pip install <project>`가 unambiguous
  supported environment에서 별도 internal path 지식 없이 동작한다.
- existing environment mutation은 frozen plan/Verified Set/Manifest 밖 content를
  반입하지 않고 interruption 시 이전 state 또는 explicit failed state를
  보존한다.
- ambiguous project, global Python, unsupported layout, concurrent mutation과
  rollback uncertainty는 fail-closed다.
- `--target`은 explicit advanced path로 유지되며 기본 UX의 필수 입력이 아니다.

### Out of scope

- arbitrary npm/pip option pass-through
- global/user package install
- 모든 workspace/environment manager
- M10 release distribution

## 14. M10 — Verified Distribution & Bootstrap

### 목적

사용자가 source clone이나 local Go build 없이 Heliopause를 설치하면서 최초
binary, helper와 runtime image를 신뢰할 수 있는 bootstrap chain을 제공한다.

### Scope

- versioning, license, release asset, supported platform와 release signing
  identity ownership 계약
- source commit에서 native `helox`와 Linux observer helper까지 이어지는
  reproducible GitHub Actions build
- checksum, signature, SBOM와 provenance/attestation을 포함한 GitHub Release
- HAA-owned 구성이 실제로 필요한 경우에만 만드는 digest-pinned GHCR runtime
  image
- one-time runtime installer와 `helox doctor`
- Docker/runsc/pod-init/network helper identity 설치·검증·upgrade/rollback

### Exit criteria

- release signing identity의 owner, rotation, compromise/revocation과 verification
  절차가 명시된다.
- 사용자가 release binary/helper와 GHCR image의 source commit, digest,
  provenance를 검증할 수 있다.
- installer는 mutable latest나 unverified binary/image를 실행하지 않는다.
- clean supported Linux Host에서 설치·doctor·M9 install E2E가 재현된다.
- custom image는 upstream image의 단순 wrapper가 아니라 HAA-owned runtime
  구성이 필요한 경우에만 존재한다.

### Out of scope

- HAA 전체를 하나의 container image로만 배포
- unattended privileged auto-update
- Windows/macOS native dynamic backend

## 15. M11 — Dynamic Detection Depth

### 목적

M8에서 신뢰성과 completeness가 확보된 observation boundary 위에 bounded process,
filesystem과 network behavior 분석을 확장한다.

### Scope

- 필요한 upstream seccheck optional/context field와 schema lock
- bounded executable/path/argv와 filesystem operation normalization
- honeytoken, unexpected process와 workspace violation의 production detection
- sensitive payload suppression, Evidence summary와 retention bound
- actual gVisor integration fixture와 policy regression

### Exit criteria

- 모든 활성 detection은 production event와 actual integration evidence를 가진다.
- raw argv/path/content가 CLI나 일반 Evidence에 유출되지 않는다.
- event field 누락, schema drift, truncation과 classification ambiguity는 clean
  observation이 아니다.
- detection 확장이 M8 Host trust와 M9 Promotion invariant를 약화하지 않는다.

### Out of scope

- arbitrary full trace logging
- Host stdout/stderr scraping을 observation transport로 사용
- ML verdict를 직접 Policy Decision으로 사용

## 16. M12 — Ecosystem Expansion Before Public Release

### 목적

M11까지 완성된 공통 trust boundary를 공식 PyTorch source, public Go Modules,
public crates.io와 public Terraform Provider 설치 흐름으로 확장한다. M12는 기존
Core/Policy/Evidence invariant를 재설계하지 않고 ecosystem adapter, resolver,
transaction, sandbox profile과 qualification만 추가한다.

### Scope

- `helox pip install --source pytorch:<profile>` 공식 PyTorch source profile
- `helox go get`, `helox go mod download`, `helox go build` public Go Modules
- `helox cargo add`, `helox cargo build` public crates.io
- `helox terraform init` public Terraform Provider installation only
- 각 ecosystem의 exact graph, source identity, digest/checksum과 transactional Promotion
- build/install execution이 있는 경우 Linux gVisor observer와 incomplete-observation fail-closed
- 기존 npm/PyPI/GitHub regression과 cross-ecosystem qualification

### Exit criteria

- PyTorch named source profile이 arbitrary index fallback 없이 동작한다.
- PyTorch ownership table이 source confusion을 차단하고 graph node source identity를 freeze한다.
- Go proxy/SumDB exact graph, verified cache와 network-disabled `go build` qualification이 통과한다.
- Cargo add/build, checksum, registry boundary와 build-time observation qualification이 통과한다.
- Terraform Registry exact provider, checksum/signature, lock transaction과 dynamic qualification이 통과한다.
- 정상·transitive·tamper·source substitution·resolver failure·observation incomplete·mutation race·rollback 시나리오가 각 ecosystem에 대해 검증된다.
- existing npm/PyPI/GitHub regression, Linux gVisor와 canonical `Required` CI가 green이다.
- CLI가 지원/비지원 경계를 정확히 표시하고, 재현 가능한 commit·test·workflow evidence가 기록된다.

### Out of scope

- private registry, alternate mirror, arbitrary source URL, VCS/direct fallback
- Terraform `plan`/`apply`와 remote module acquisition
- M12 이후 새로운 ecosystem 추가

## 17. M13 — Production Release & Operations

### 목적

M10의 verified distribution workflow와 M11의 detection depth가 모두 완료된 뒤,
repository 운영 보호와 최종 public release를 한 번의 명시적인 production gate로
수행한다. 이 milestone 전에는 release asset을 public channel에 게시하지 않는다.

### Scope

- GitHub `release` environment와 required reviewer
- `v*` release tag 생성·수정·삭제 보호
- main Ruleset 활성화와 `license/cla`, `Required` status check enforcement
- `feature → develop → main` PR 흐름과 main 직접 push 차단
- exact tag/source-run candidate, manifest/checksum/attestation 및 immutable release 검증
- 공개 release 후 asset verification, doctor smoke test, rollback/quarantine evidence

### Exit criteria

- M11 detection depth qualification과 전체 required CI가 완료된다.
- 보호된 environment, tag policy, Ruleset과 develop branch 흐름이 실제 repository에
  적용되고 우회 경로가 차단된다.
- release tag commit과 성공한 candidate run이 일치하고, 모든 asset의 manifest,
  checksum, signer provenance와 immutable release verification이 통과한다.
- release 이후 `gh release verify`, asset verification, 설치·doctor smoke test와
  rollback/quarantine 절차가 재현 가능한 evidence로 기록된다.

### Out of scope

- 새로운 detection capability 구현
- Ruleset·tag·environment 보호를 우회한 emergency public publish
- M11의 raw trace logging 또는 ML 기반 policy decision

## 18. Cross-cutting track

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

## 19. Global Definition of Ready

milestone 또는 그 안의 work item은 다음 조건을 충족한 뒤 시작한다.

- 필요한 upstream milestone exit criteria가 충족됨
- 작업에 필요한 canonical document route가 식별됨
- unresolved decision이 결과를 크게 바꾸면 entry decision으로 분리됨
- external tool/runtime/source의 trust·version·license·network 요구가 식별됨
- test fixture가 synthetic/sanitized이며 expected result가 정의됨
- Host write, credential, network 또는 destructive behavior가 있다면 exact target과 rollback/cleanup 경계가 정의됨
- 완료를 증명할 deterministic test/check가 정의됨

조건을 충족하지 못한 항목을 코드로 추측하여 시작하지 않는다.

## 20. Global Definition of Done

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

## 21. Status와 변경 관리

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

### Baseline Audit 상태 판정

마일스톤 또는 work item 작업을 시작하기 전에는 source code를 수정하지 않고
repository baseline audit를 먼저 수행한다. 각 항목은 다음 네 상태를 독립적으로
기록한다.

```text
IMPLEMENTED: YES / NO
WIRED: YES / NO
QUALIFIED: YES / NO
ACCEPTANCE_CLOSED: YES / NO
MISSING: 실제 남은 acceptance gap 목록
```

`IMPLEMENTED=YES`인 기능은 구현 부재로 해석하지 않으며 다시 구현하지 않는다.
기존 구현을 대체·재작성해야 할 때는 기존 위치, 재사용할 수 없는 이유, 기존 수정으로
해결할 수 없는 이유와 영향 범위를 먼저 제시한다. 상태 판정에는 commit, file,
symbol/function, test와 workflow/run evidence를 연결한다.

문서 ownership은 다음과 같이 고정한다.

- 이 문서(`01-milestones.md`): milestone 범위·의존 관계·exit/acceptance criteria
- `02-current-work-queue.md`: 현재 milestone/work item 상태와 active/next 순서의 유일한 canonical owner
- `PROJECT-DECISIONS.md`: 확정된 설계 결정만 기록하며 진행률의 canonical owner가 아님
- `README.md`: 사용자·개발자용 요약이며 진행률의 canonical owner가 아님

`02-current-work-queue.md`는 source commit과 그 시점의 milestone 상태가 함께 보존되도록
Git에서 추적한다.

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
11. M7 이후 발견된 production trust gap은 과거 evidence를 삭제하지 않고 M8
    release-blocking remediation으로 기록한다.
12. M8 trust correctness, M9 product UX, M10 verified bootstrap, M11 detection
    sophistication 순서를 유지한다.

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
- [x] M0~M7 MVP milestone과 M8~M11 post-MVP dependency
- [x] M0 implementation/quality foundation
- [x] M1 fake 기반 Domain workflow
- [x] M2 npm static fail-closed inspect
- [x] M3 Linux dynamic inspect
- [x] M4 npm dependency/install/Promotion full E2E
- [x] M5 PyPI wheel/sdist expansion
- [x] M6 GitHub Releases standalone path
- [x] M7 MVP qualification
- [x] M8 Production Trust Hardening
- [x] M9 Product Install UX
- [x] M10 Verified Distribution & Bootstrap
- [x] M11 Dynamic Detection Depth
- [x] cross-cutting security/test/docs/CI activation
- [x] Global Definition of Ready/Done
- [x] milestone status와 변경 관리
- [x] Step 13 전달 범위
