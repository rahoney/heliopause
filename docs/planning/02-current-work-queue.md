# Step 13 — Current Work Queue

이 문서는 Heliopause에서 **지금 실행할 작업, 순서, 상태와 완료 증거**를 관리한다. Milestone의 범위와 완료 조건은 [Step 12 — Milestones](./01-milestones.md)가 소유하며, 이 문서는 현재 milestone만 구현 가능한 크기로 분해한다.

상세 Architecture·Domain·Engineering 결정을 복제하지 않는다. 각 work item은 필요한 canonical leaf 문서를 가리키고, Step 14는 해당 문서를 읽은 뒤 그 항목의 범위만 구현한다.

## 1. 현재 상태

```text
Current milestone: M12 — Ecosystem Expansion Before Public Release
Milestone status: IN_PROGRESS
Active work item: M12-001 — Official PyTorch source support
Next work item: M12-002 — public Go Modules
Next work item status: NOT_STARTED (blocked until M12-001 acceptance closes)
Ready: Yes
```

M0~M7은 원래 정의한 MVP qualification과 evidence를 완료했다. external
security review에서 확인된 production trust-boundary gap은 과거 M7 evidence를
삭제하거나 실패로 바꾸지 않고 M8 release-blocking remediation으로 처리한다.
M8은 완료되었다. M9-001은 canonical transaction contract를 확정했고,
M9-002는 HAA-managed npm project의 private mutation, complete Verified Set
commit과 rollback 회귀검증을 완료했다. M9-003은 active Python 3.14 venv의
baseline 충돌 방지와 HAA-owned file transaction 및 `pip` install CLI wiring을
완료했다. M9-004는 정적 root command tree와 help 무부작용 경계를 완료했고,
M9-005는 GitHub standalone 기본 destination과 no-overwrite 정책을 완료했고,
M9-006은 hostile transaction regression과 전체 qualification을 완료했다.
M10-001은 release signing identity, artifact manifest와 bootstrap trust contract를
확정했다. M10-002는 exact runtime lock을 소비하는 native/helper 빌드,
CycloneDX SBOM·release manifest 및 tag-only OIDC provenance candidate workflow를
완료했다. M10-003은 HAA-owned GHCR image가 실제로 필요하지 않음을 결정하고,
공식 Node/Python image를 digest/provenance manifest로 검증했다. M10-004는
verifier-gated atomic installer transaction과 fail-closed doctor를 완료했다.
public release publishing은 M10-007의 protected publication workflow에서만 수행한다.
M10-006은 Apache-2.0 outbound license, Harmony copyright-license CLA Option Five,
individual/entity contribution 절차와 pre-CLA source provenance를 확정했다.
M10-007은 exact tag/build-run candidate, manifest/checksum과 GitHub attestation을
검증하는 protected publication workflow implementation과 PR/CI qualification을
완료했다. 실제 production release environment·tag policy·main Ruleset activation·
`develop` branch 전환과 public release는 M12 및 M12-02 완료 후 M13에서 수행한다.

M11-001은 [M11 Dynamic Detection Depth Contract](./14-m11-dynamic-detection-depth-contract.md)에서
exact gVisor schema lock, bounded normalized vocabulary와 raw payload non-retention
contract를 먼저 확정했다. M11-002는 이 contract의 requested point/field가 pinned
source에서 실제로 compile·parse되는지를 Bazel contract test와 Linux lifecycle CI로
증명했다. M11-003은 이 bounded fact에 trusted backend classification과 actionable
Finding을 연결했고, exact helper와 Linux lifecycle CI로 검증했다.

M11 post-qualification release hardening은
[M11 이후 Release Hardening Fix List](./15-m11-02-fix-list.md)의 FIX-01~05만
수행한다. 새 architecture 또는 detection capability는 추가하지 않는다. 모든 fix의
local qualification을 완료하고 하나의 milestone branch push/CI evidence를 확인한
뒤에만 M12를 재개한다.

M12 생태계 확장 순서는 queue에 고정한다: 공식 PyTorch source profile → public Go
Modules → public Cargo/crates.io → public Terraform Provider → 전체 생태계
qualification·feature freeze → M12-02 final red-team/fix gate. M12 기능과 red-team
gate가 완료되기 전에는 M13 Production Release & Operations로 이동하거나 public
release를 게시하지 않는다. source checkout/local Go build 없는 public bootstrap은
M13-005가 완료하기 전까지 제공된다고 주장하지 않는다.

## 2. 상태와 queue 규칙

Milestone과 work item에 같은 네 상태를 사용한다.

```text
NOT_STARTED
IN_PROGRESS
BLOCKED
COMPLETE
```

- 동시에 하나의 work item만 `IN_PROGRESS`로 둔다.
- 표의 순서가 기본 실행 우선순위다. 선행 항목이 완료되지 않았으면 다음 항목을 시작하지 않는다.
- `Ready: Yes`는 선행조건과 필수 입력이 준비되어 바로 시작할 수 있다는 뜻이며 완료나 성공을 뜻하지 않는다.
- `BLOCKED`에는 blocker, 마지막 확인 결과와 해제 조건을 기록한다.
- `COMPLETE`는 acceptance criteria와 요구 check가 실제 evidence로 확인된 경우에만 사용한다.
- 구현 중 발견한 별도 작업은 현재 항목의 완료에 필수일 때만 좁게 추가한다. 미래 milestone 항목은 해당 milestone이 가까워질 때 연다.
- scope가 Milestone, Architecture, Domain invariant를 바꾸면 queue에서 우회하지 않고 canonical 설계 문서를 먼저 갱신한다.

## 3. M0 실행 Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M0-001 | Project Identity 결정 | Step 13 | COMPLETE | No |
| 2 | M0-002 | Go·Tool·CI Identity lock | M0-001 | COMPLETE | No |
| 3 | M0-003 | Go module과 최소 process bootstrap | M0-002 | COMPLETE | No |
| 4 | M0-004 | Canonical check runner foundation | M0-003 | COMPLETE | No |
| 5 | M0-005 | Static analysis와 Quick profile 활성화 | M0-004 | COMPLETE | No |
| 6 | M0-006 | Quick·Docs·Required CI foundation | M0-005 | COMPLETE | No |
| 7 | M0-007 | Minimum Go·macOS와 repository gate 검증 | M0-006 | COMPLETE | No |
| 8 | M0-008 | M0 qualification과 M1 handoff | M0-007 | COMPLETE | No |

### M1 실행 Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M1-001 | Domain workflow entry decision | M0 | COMPLETE | No |
| 2 | M1-002 | Domain identity와 Run state foundation | M1-001 | COMPLETE | No |
| 3 | M1-003 | Check result·Port와 deterministic fake | M1-002 | COMPLETE | No |
| 4 | M1-004 | Inspect orchestration과 Policy v1 | M1-003 | COMPLETE | No |
| 5 | M1-005 | CLI result contract와 synthetic vertical test | M1-004 | COMPLETE | No |
| 6 | M1-006 | M1 qualification과 M2 handoff | M1-005 | COMPLETE | No |

### M2 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M2-001 | npm Static Inspect entry decision | M1 | COMPLETE | No |
| 2 | M2-002 | npm reference·metadata resolve foundation | M2-001 | COMPLETE | No |
| 3 | M2-003 | Controlled Intake와 declared integrity verification | M2-002 | COMPLETE | No |
| 4 | M2-004 | npm tarball static inspection과 Evidence Store | M2-003 | COMPLETE | No |
| 5 | M2-005 | M2 Policy·CLI wiring과 controlled vertical test | M2-004 | COMPLETE | No |
| 6 | M2-006 | M2 qualification과 M3 handoff | M2-005 | COMPLETE | No |

### M3 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M3-001 | Linux Dynamic Inspect entry decision | M2 | COMPLETE | No |
| 2 | M3-002 | gVisor runtime lock과 capability probe | M3-001 | COMPLETE | No |
| 3 | M3-003 | Sandbox Session·Observation Domain과 Port | M3-002 | COMPLETE | No |
| 4 | M3-004 | gVisor Sandbox Session lifecycle | M3-003 | COMPLETE | No |
| 5 | M3-005 | gVisor raw observation collector | M3-004 | COMPLETE | No |
| 6 | M3-006 | Dynamic observation normalization | M3-005 | COMPLETE | No |
| 7 | M3-007 | M3 Policy와 npm inspect wiring | M3-006 | COMPLETE | No |
| 8 | M3-008 | M3 qualification과 M4 handoff | M3-007 | COMPLETE | No |

### M4 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M4-001 | npm Install and Promotion entry decision | M3 | COMPLETE | No |
| 2 | M4-002 | Install Context와 locked npm resolver boundary | M4-001 | COMPLETE | No |
| 3 | M4-003 | recursive dependency inspection과 set-level Policy | M4-002 | COMPLETE | No |
| 4 | M4-004 | Verified Manifest·SBOM과 trusted Staging | M4-003 | COMPLETE | No |
| 5 | M4-005 | offline npm Promotion과 atomic target | M4-004 | COMPLETE | No |
| 6 | M4-006 | M4 qualification과 M5/M6 handoff | M4-005 | COMPLETE | No |

### M5 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M5-001 | PyPI/pip Expansion entry decision | M4 | COMPLETE | No |
| 2 | M5-002 | Python/pip runtime identity lock과 PyPI reference foundation | M5-001 | COMPLETE | No |
| 3 | M5-003 | isolated PyPI resolver·target tag·egress policy | M5-002 | COMPLETE | No |
| 4 | M5-004 | wheel intake·integrity·static inspection | M5-003 | COMPLETE | No |
| 5 | M5-005 | Python dynamic inspection·PEP 517 sdist build | M5-004 | COMPLETE | No |
| 6 | M5-006 | generic staging·offline pip Promotion·CLI integration | M5-005 | COMPLETE | No |
| 7 | M5-007 | M5 qualification·npm regression·M6 handoff | M5-006 | COMPLETE | No |

### M6 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M6-001 | GitHub Releases Standalone entry decision | M5 | COMPLETE | No |
| 2 | M6-002 | exact public input·API boundary foundation | M6-001 | COMPLETE | No |
| 3 | M6-003 | GitHub Release resolve·Acquire·integrity | M6-002 | COMPLETE | No |
| 4 | M6-004 | standalone static inspection·M6 Policy | M6-003 | COMPLETE | No |
| 5 | M6-005 | Linux ELF dynamic inspection wiring | M6-004 | COMPLETE | No |
| 6 | M6-006 | standalone Promotion·CLI·Linux E2E | M6-005 | COMPLETE | No |
| 7 | M6-007 | M6 qualification·regression·M7 handoff | M6-006 | COMPLETE | No |

### M7 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M7-001 | MVP qualification entry·evidence matrix | M5, M6 | COMPLETE | No |
| 2 | M7-002 | ecosystem flow·fixture regression | M7-001 | COMPLETE | No |
| 3 | M7-003 | Linux·macOS·WSL2 CLI qualification | M7-002 | COMPLETE | No |
| 4 | M7-004 | evidence·result·resilience qualification | M7-003 | COMPLETE | No |
| 5 | M7-005 | security workflow·scheduled gate qualification | M7-004 | COMPLETE | No |
| 6 | M7-006 | MVP final audit·documentation·completion | M7-005 | COMPLETE | No |

### M8 Entry Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M8-001 | production trust hardening entry·canonical contract | M7, external review | COMPLETE | No |
| 2 | M8-002 | Trusted Host tool identity·executor·local Docker endpoint | M8-001 | COMPLETE | No |
| 3 | M8-003 | observer supervisor·exclusive endpoint·process-scoped composition | M8-002 | COMPLETE | No |
| 4 | M8-004 | connection identity latch·production observation completeness | M8-003 | COMPLETE | No |
| 5 | M8-005 | least-privilege typed resolver network-policy helper | M8-004 | COMPLETE | No |
| 6 | M8-006 | runtime lock single source of truth·CI integration | M8-005 | COMPLETE | No |
| 7 | M8-007 | hostile-boundary regression·Linux security requalification | M8-006 | COMPLETE | No |

### M8-001 — production trust hardening entry·canonical contract

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [Trusted Tooling and Evidence](../threat-model/04-trusted-tooling-and-evidence.md)
- [Inspection, Sandbox and Policy Architecture](../architecture/03-inspection-sandbox-policy.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- M7 completion evidence를 보존하면서 M8이 production-ready release blocker임을
  canonical status와 public README에 정확히 표시한다.
- trusted local Docker endpoint, observer exclusive ownership, typed privileged
  helper와 truthful observation capability가 기존 invariant를 약화하지 않는
  entry contract로 확정된다.
- M8-002~007 각각의 dependency, acceptance와 check owner가 하나의 queue로
  정해지고 product implementation은 추가하지 않는다.

**Status**

```text
Status: COMPLETE
Checks: canonical docs profile; git diff --check
Evidence: M8 contract; D-015 Host trusted execution; M8 Host control plane architecture; M8~M11 milestone and seven-item M8 queue
Limitations: M8-001은 planning/contract만 변경하며 production code, helper, runtime 또는 CI job을 구현하지 않음
Next: M8-002 — NOT_STARTED / Ready: Yes
```

### M8-002 — Trusted Host tool identity·executor·local Docker endpoint

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [Trusted Tooling and Evidence](../threat-model/04-trusted-tooling-and-evidence.md)
- [Directory Structure](../engineering/01-directory-structure.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- Sandbox와 Promotion production process execution이 ambient PATH나 Host
  environment를 사용하지 않고 validated absolute tool identity와 minimal
  command-specific environment만 사용한다.
- Docker는 configured local endpoint의 transport, ownership와 daemon/runtime
  capability를 검증하고 arbitrary `DOCKER_HOST`, context, remote endpoint와
  proxy를 거부한다.
- writable/symlink/wrong-owner tool path, parent replacement, fake PATH, hostile
  Docker environment, wrong runsc digest/version와 registered-runtime mismatch
  fixture가 모두 fail-closed다.
- Core/Application/Policy에 Host executable·environment·Docker detail이
  유입되지 않는다.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/security; hostile tool/environment unit and contract tests; Linux Host executor integration; full CI Required
Evidence: commits 947b8af, fa0814e, 696289b, a3e1e56, 47052cb, da5a657; GitHub Actions run 32686523973 success including macOS, Minimum Go, Security, gVisor observer and Linux lifecycle
Limitations: observer supervisor와 privileged firewall helper는 후속 M8 item
Next: M8-003 — NOT_STARTED / Ready: Yes
```

### M8-003 — observer supervisor·exclusive endpoint·process-scoped composition

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [M3 Linux Dynamic Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Inspection, Sandbox and Policy Architecture](../architecture/03-inspection-sandbox-policy.md)

**Acceptance**

- HAA가 trusted receiver → pinned helper → readiness → Sandbox → helper/receiver
  cleanup lifecycle을 소유한다.
- npm, PyPI resolver/wheel/sdist와 GitHub backend는 process-scoped observer 하나를
  공유하며 factory별 endpoint unlink/rebind가 없다.
- MVP에서는 fixed endpoint exclusive ownership을 보장하고 다른 supervisor가
  소유하면 임의 재사용·삭제하지 않고 fail-closed한다.
- helper start/readiness/death, duplicate owner와 endpoint/helper cleanup failure
  integration이 non-ALLOW다. multi-process multiplexing은 별도 후속 확장이다.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/security; race; supervisor lifecycle unit/integration; git diff --check; full CI Required
Evidence: commits 06f769e, d377c55, 0163991, b32f329, 7a5aeb4; GitHub Actions run 32691425705 success including macOS, Minimum Go, Security, gVisor observer and Linux lifecycle
Limitations: Host-scoped multi-process multiplexing service 없음
Next: M8-004 — IN_PROGRESS / Ready: Yes
```

### M8-004 — connection identity latch·production observation completeness

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [M3 Linux Dynamic Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Inspection·Sandbox·Policy Contract](../domain-model/08-contract-inspection-sandbox-policy.md)

**Acceptance**

- helper가 최초 valid container ID를 connection identity로 latch하고 이후
  mismatch, malformed payload, dropped event와 abnormal end를 incomplete fault로
  전달한다.
- active Finding/Policy rule마다 production helper가 실제 생성하는 bounded
  signal과 Linux gVisor integration evidence가 있다.
- honeytoken, unexpected process와 filesystem violation을 생성할 수 없으면 해당
  capability는 unsupported/incomplete이며 synthetic test만으로 clean/ALLOW하지
  않는다.
- raw sensitive trace payload는 Core, Policy, CLI와 일반 Evidence에 유출되지
  않는다.

**Status**

```text
Status: COMPLETE
Checks: pinned Bazel helper test; parser/fuzz; actual gVisor observation integration; policy regression; git diff --check
Evidence: commits 794f933, 9bfa186, cfc2999; GitHub Actions run 32694540606 success including normalized parser fuzz, exact pinned Bazel protocol test, observer helper and Linux gVisor lifecycle
Limitations: richer optional detection은 M11 범위
Next: M8-005 — IN_PROGRESS / Ready: Yes
```

### M8-005 — least-privilege typed resolver network-policy helper

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Inspection, Sandbox and Policy Architecture](../architecture/03-inspection-sandbox-policy.md)

**Acceptance**

- ordinary-user `helox`와 root-owned network helper/service를 분리하고 helper
  authority를 필요한 network administration 범위로 제한한다.
- API는 create/verify/remove resolver policy typed operation만 받고 package name,
  hostname, shell fragment와 raw iptables/nft arguments를 받지 않는다.
- peer, session, HAA-owned network/subnet, approved public IP set와 default-deny
  rule을 독립적으로 검증한다.
- unauthorized peer, arbitrary operation, duplicate/stale session, helper crash와
  apply/verify/cleanup uncertainty가 fail-closed다.
- Linux integration은 one-time privileged service setup 후 실제 workflow를
  ordinary user로 실행한다. `sudo helox`는 supported UX가 아니다.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/security; typed protocol fuzz/authorization tests; privileged Linux integration; cleanup regression; git diff --check
Evidence: commits 80a933a, 06f6051, f861b61, 05efa0e, f71fa90, 92fb590, 5eeed45, 731c130, 731e927; GitHub Actions run 32702853334 Required success including ordinary-user privileged policy workflow
Limitations: arbitrary Host firewall administration과 sudo helox는 지원하지 않음
Next: M8-006 — IN_PROGRESS / Ready: Yes
```

### M8-006 — runtime lock single source of truth·CI integration

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [Quality Toolchain](../engineering/03-quality-toolchain.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Acceptance**

- canonical runtime/observer lock이 gVisor, runsc, Bazel, Docker와 Node/Python
  identity의 유일한 수동 입력이다.
- product code와 CI는 lock 또는 deterministic generated output을 사용하며
  generated drift validator가 required gate에 포함된다.
- workflow와 workflow checker의 hand-copied runtime identity를 제거하고 lock
  corruption, unknown field, missing platform, code/workflow mismatch를
  fail-closed test한다.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/platform/security; lock schema/drift/workflow validator; git diff --check
Evidence: commit dac187b; canonical quick/docs/platform/security and focused race checks passed locally
Limitations: release signing identity와 published artifact manifest는 M10 범위
Next: M8-007 — NOT_STARTED / Ready: Yes
```

### M8-007 — hostile-boundary regression·Linux security requalification

**Read**

- [M8 Production Trust Hardening Contract](./11-m8-production-trust-hardening-contract.md)
- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)
- [MVP Results, Policy and Completion](../mvp-scope/03-results-policy-and-completion.md)

**Acceptance**

- hostile PATH/environment/tool/socket/container attribution/privilege/cleanup
  matrix와 production bootstrap E2E가 expected fail-closed result를 낸다.
- canonical quick/docs/security/vulnerability/race와 Linux gVisor observer,
  npm/PyPI resolver/dynamic/Promotion integration이 current main candidate에서
  모두 green이다.
- known Critical/High production trust-boundary defect가 없고 limitation 및
  unsupported multi-process/Host path를 README에 정확히 공개한다.
- completion evidence와 CI run을 기록한 뒤에만 production-ready release
  blocker 해제를 선언한다. M7 evidence는 그대로 보존한다.

**Status**

```text
Status: COMPLETE
Checks: complete canonical suite; race; hostile-boundary matrix; pinned Linux production composition; Required CI
Evidence: commit dac187b; GitHub Actions run 32705277070 Required success including gVisor Linux lifecycle, observer Bazel helper, macOS, Minimum Go, Security and Vulnerability
Limitations: M9 product UX, M10 verified distribution과 M11 detection depth는 별도 milestone
Next: M9-001 — NOT_STARTED / Ready: Yes
```

### M7-002 — ecosystem flow·fixture regression

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Inspection·Sandbox·Policy Contract](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Evidence·Staging·Promotion Contract](../domain-model/09-contract-evidence-staging-promotion.md)

**Completion**

```text
Status: COMPLETE
Checks: go test ./...; go test -race ./internal/application ./internal/policy; canonical quick; git diff --check
Evidence: internal/application/m7_qualification_test.go; exact source-neutral identity/digest binding across npm, PyPI and GitHub Release; non-ALLOW Promotion stop; acquisition failure has no Policy decision
Limitations: existing adapter-owned format/runtime tests remain the source of detailed archive and Linux evidence; Host-platform qualification is M7-003
Next: M7-003 — NOT_STARTED / Ready: Yes
```

### M7-003 — Linux·macOS·WSL2 CLI qualification

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [MVP ecosystems and platforms](../mvp-scope/01-ecosystems-platforms-artifacts.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Acceptance**

- Linux and macOS each build the native `helox` executable and execute its
  default help path without network, Artifact acquisition or dynamic-backend
  claims.
- WSL2 evidence comes from an approved disposable Windows/WSL2 environment;
  the retained evidence identifies the environment and preserves command
  output. Ordinary Linux CI and unsupported nested virtualization are not a
  substitute.
- the published capability matrix keeps dynamic inspection limited to the
  pinned Linux backend and preserves non-`ALLOW` behavior for unsupported
  host/backend combinations.

**Status**

```text
Status: COMPLETE
Checks: GitHub Actions run 32443146082 Required aggregate (Quick, Docs, Minimum Go, macOS, gVisor observer and Linux integration) successful; local macOS go test ./...; canonical quick; CLI race test; disposable Windows 11 VM WSL2 qualification command
Evidence: cmd/helox/main_test.go builds ./cmd/helox and runs no argument plus --help; scripts/qualify-wsl2-cli.sh; external WSL2 record: Windows 11 Enterprise Evaluation VM → Ubuntu 24.04.4 WSL2, Go 1.26.5, no-argument and --help output, source a3e5ced2d2dd4254e97470a5a385ea03ee86160a
Limitations: WSL2 is a Linux CLI Host path only; it does not claim Windows-native or macOS-native dynamic inspection
Next: M7-004 — IN_PROGRESS / Ready: Yes
```

### M7-004 — evidence·result·resilience qualification

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [Evidence·Staging·Promotion Contract](../domain-model/09-contract-evidence-staging-promotion.md)
- [Evidence, Staging and Promotion Architecture](../architecture/04-evidence-staging-promotion.md)
- [MVP results, Policy and completion](../mvp-scope/03-results-policy-and-completion.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- Evidence references, machine/human results, Manifest/SBOM, staged bytes and
  published bytes retain one exact normalized identity/digest chain.
- interruption, retry and cleanup tests prove no partial Evidence, staging or
  target success is exposed; cleanup uncertainty remains non-success.
- completed records are retained for audit/retry without adding automated
  retention/garbage collection; fixtures and results contain no secret or raw
  Host/observer data.

**Status**

```text
Status: COMPLETE
Checks: go test ./...; go test -race -timeout=5m ./...; go run ./scripts/check quick; go run ./scripts/check docs; git diff --check; GitHub Actions run 32476129928 Required aggregate successful
Evidence: duplicate Evidence batch preflight and private temporary directory commit prevent partial Run publication; retry preserves committed Evidence; human/JSON results suppress raw causes; staged/published Manifest, SBOM and artifacts rehash to one identity/digest chain
Limitations: this qualification adds no source, format, retention daemon, product configuration, CLI command, remote store or synthetic success dependency
Next: M7-005 — IN_PROGRESS / Ready: Yes
```

### M7-005 — security workflow·scheduled gate qualification

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [Quality Toolchain](../engineering/03-quality-toolchain.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- pinned gosec, Gitleaks and govulncheck install outside the product module
  graph with integrity and identity checks; tool/bootstrap/scanner failure is
  non-success.
- PR required CI runs source-security and vulnerability profiles and the
  aggregate includes every active required child fail-closed.
- a least-privilege weekly/manual security workflow scans default-branch
  reachable history, refreshes vulnerability data and executes only bounded
  existing fuzz targets; each real finding path is regression-tested with
  synthetic data only.

**Status**

```text
Status: COMPLETE
Checks: Go 1.26.7 local fuzz/quick/docs; GitHub Actions run 32555127293 Required aggregate successful; default-branch Scheduled Security run 32556503406 successful
Evidence: exact external tool pins and integrity checks; independent required Security/Vulnerability jobs; fail-closed aggregate; full-history secret scan; refreshed govulncheck; four 5-second fuzz targets with bounded one-minute process deadlines
Limitations: scheduled execution reports findings but does not retroactively change an older merge result; Gitleaks v8.18.4 remains an exceptional last-known-functional pin pending upstream rule repair and re-evaluation
Next: M7-006 — IN_PROGRESS / Ready: Yes
```

### M7-006 — MVP final audit·documentation·completion

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [MVP results, Policy and completion](../mvp-scope/03-results-policy-and-completion.md)
- [Inspection·Sandbox·Policy Architecture](../architecture/03-inspection-sandbox-policy.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)
- [Quality Toolchain](../engineering/03-quality-toolchain.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Acceptance**

- re-audit all Artifact, Sandbox, Evidence, Staging and Promotion trust
  boundaries and leave no known Critical/High product security defect or
  unresolved invariant violation.
- run the complete qualification suite and retain current reproducible Linux,
  macOS, WSL2, gVisor and scheduled-security evidence.
- publish installation, supported-path, limitation, troubleshooting and
  security-reporting guidance without overstating unsupported capabilities.
- record MVP completion only after every prior M7 item and final evidence is
  current and green.

**Status**

```text
Status: COMPLETE
Checks: canonical quick/docs/security/vulnerability; go test ./...; go test -race -timeout=15m ./...; full-rule gosec review; git diff --check; GitHub Actions run 32557123229 all children and Required aggregate successful
Evidence: resolver container/firewall cleanup commands have independent bounded contexts; Evidence/GitHub redirect close failures remain fail-closed; README publishes exact build/command/support/runtime/troubleshooting boundaries; SECURITY.md defines private reporting; prior M7 Linux/macOS/WSL2/scheduled-security evidence remains current
Security audit: all full-rule gosec findings reviewed; G115 conversions are guarded by explicit nonnegative and size limits, G122/G304 paths remain under trusted Host roots with symlink/type/digest revalidation, G204 commands use fixed adapter-owned executables/arguments, and G302 modes are owner-only directory/executable modes; no known Critical/High product defect or unresolved trust-boundary violation remains
Limitations: full automatic dynamic inspection and Promotion are qualified only on the pinned Linux amd64 Docker/runsc/observer path; macOS and WSL2 evidence is CLI-only; no general Host runtime installer, native Windows/macOS dynamic backend, automatic retention daemon, authenticated private source or absolute-safety guarantee is claimed
Next: MVP qualification complete; post-MVP work requires a new explicit queue decision
```

### M7-001 — MVP qualification entry·evidence matrix

**Read**

- [M7 Qualification Contract](./10-m7-mvp-qualification-contract.md)
- [M7 milestone scope and exit criteria](./01-milestones.md)
- [MVP ecosystems and platforms](../mvp-scope/01-ecosystems-platforms-artifacts.md)
- [MVP inspection and verification](../mvp-scope/02-inspection-and-verification.md)
- [MVP results, Policy and completion](../mvp-scope/03-results-policy-and-completion.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Acceptance**

- M7 exit criteria, fixture outcomes, platform evidence, result resilience and
  security gates are assigned to one ordered M7 item each.
- unavailable, unsupported, skipped and incomplete evidence cannot be treated
  as qualification success or automatic `ALLOW`.
- this entry creates no product implementation, runtime, dependency, CLI
  command or CI job.

**Completion**

```text
Status: COMPLETE
Checks: canonical docs; git diff --check
Decision/Evidence: docs/planning/10-m7-mvp-qualification-contract.md; explicit six-item fail-closed evidence matrix
Limitations: WSL2 evidence and scheduled security workflow remain M7 work; neither is claimed as green
Next: M7-003 — NOT_STARTED / Ready: Yes
```

M5-007 is complete. M6-001 resolves exact GitHub release asset identity,
provenance, target and standalone Promotion boundaries before implementation.

### M6-001 — GitHub Releases Standalone entry decision

**Read**

- [M6 GitHub Releases Standalone Contract](../domain-model/15-m6-github-releases-contract.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Inspection·Sandbox·Policy Contract](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Evidence·Staging·Promotion Contract](../domain-model/09-contract-evidence-staging-promotion.md)
- [M6 milestone scope and exit criteria](./01-milestones.md)

**Acceptance**

- exact public repository/release/tag/asset selector, API identity/digest,
  rate-limit and no-ambient-auth boundary are explicit.
- artifact format/static/dynamic capability, sidecar/provenance, target and
  no-re-resolution Promotion rules preserve existing common invariants.
- M6 implementation queue is one-item-at-a-time and M6-001 adds no source,
  runtime, dependency, CLI command or CI job.

**Completion**

```text
Status: COMPLETE
Checks: official GitHub Release/API/rate-limit/attestation review; canonical quick/docs; git diff --check
Decision/Evidence: docs/domain-model/15-m6-github-releases-contract.md; public exact tag+asset selector; API SHA-256 and no ambient auth; generic Manifest/Staging; no re-resolution Promotion
Limitations: M6-001 creates no product implementation; sidecar signature/checksum/attestation auto-verification is deferred until a typed verifier contract exists
Next: M6-002 — IN_PROGRESS / Ready: Yes
```

### M6-002 — exact public input·API boundary foundation

**Read**

- [M6 GitHub Releases Standalone Contract](../domain-model/15-m6-github-releases-contract.md)
- [Artifact Identity](../domain-model/02-artifact-identity.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Coding / Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- `owner/repo@tag#asset` parser accepts only one bounded exact selector and
  rejects latest, omitted fields, URL/path/query/fragment and ambiguous input.
- public API response boundary retains only exact release/asset ID, tag,
  uploaded state, declared SHA-256, declared size, immutable marker and
  publication time; missing/ambiguous data is rejected.
- no HTTP acquisition, ambient auth, CLI command, runtime or product dependency
  is introduced before M6-003.

**Completion**

```text
Status: COMPLETE
Commit: pending
Checks: focused unit/race; 5-second parser fuzz; canonical quick/docs; git diff --check
Decision/Evidence: internal/artifact/githubrelease exact selector and bounded Release metadata parser; GitHub REST API base/version constants; latest/unsafe input fail-closed
Limitations: API transport, controlled acquisition, integrity verification, static/dynamic inspection and CLI remain later M6 items
Next: M6-003 — IN_PROGRESS / Ready: Yes
```

### M6-003 — GitHub Release resolve·Acquire·integrity

**Read**

- [M6 GitHub Releases Standalone Contract](../domain-model/15-m6-github-releases-contract.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Verification·Evidence·Finding](../domain-model/04-verification-evidence-finding.md)
- [Coding / Security Rules](../engineering/02-coding-security-rules.md)

**Acceptance**

- Resolve uses only the pinned public release-by-tag API and freezes the exact
  repository, release ID, tag, asset ID, asset name, size and SHA-256.
- Acquire uses only the resolved numeric asset API media flow, bounded trusted
  redirects and a Run-private controlled intake directory; no auth, proxy or
  caller-controlled direct URL is used.
- observed SHA-256 and byte size must equal the API declaration before an
  acquired Artifact can exist; mismatch, stream/redirect failure and cleanup
  uncertainty fail closed.
- the source-specific verifier reports the declared SHA-256 check through the
  common Verification port.

**Completion**

```text
Status: COMPLETE
Checks: focused unit/race; 5-second selector fuzz; canonical quick/docs; git diff --check
Evidence: internal/artifact/githubrelease controlled API Resolve/asset-ID Acquire adapter; internal/verification/githubrelease declared SHA-256 verifier
Limitations: static format analysis, M6 Policy, Linux ELF dynamic inspection, Promotion and CLI remain later M6 items
Next: M6-005 — NOT_STARTED / Ready: Yes
```

### M4-003 — recursive dependency inspection과 set-level Policy

**추가 확정 scope**

- isolated npm resolver의 package-lock v3 parser와 graph 전체의 독립 Run/Evidence/entry Policy를 구현한다.
- resolver 전용 Docker network-policy helper가 default-deny egress policy를 create → apply → verify → cleanup하고, npm resolution에 필요한 검증된 registry endpoint 집합만 허용한다.
- Docker 일반 bridge fallback, Host 사전 firewall/proxy 의존, proxy 환경변수-only enforcement는 허용하지 않는다.
- Linux Docker firewall backend probe, policy apply/verify/cleanup failure와 runtime endpoint mismatch는 fail-closed로 graph를 반환하지 않는다.

### M4-004 — Verified Manifest·SBOM과 trusted Staging

**Completion**

```text
Status: COMPLETE
Commit: e43b780
Checks: canonical quick; focused race; git diff --check
Decision/Evidence: deterministic helox.verified-manifest/v1; CycloneDX 1.7; Quarantine-to-Staging rehash; exclusive/fsync/read-only/atomic filesystem tests
Limitations: offline npm install·target publish는 M4-005 범위; staging retention/GC 없음
Next: M4-005 — IN_PROGRESS / Ready: Yes
```

### M4-005 — offline npm Promotion과 atomic target

**Completion**

```text
Status: COMPLETE
Commit: ec05e2b
CI: GitHub Actions run 31784626998 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: canonical quick; focused race; real pinned Docker offline npm ci Promotion integration; git diff --check
Decision/Evidence: Manifest-bound target-local tarballs and generated local-file lock; --network none/--pull never; post-install exact-set validation; Linux renameat2(RENAME_NOREPLACE) publish
Limitations: Linux amd64 automatic Promotion only; scripts/bin links and existing/global/workspace targets are unsupported; no staging retention/GC
Next: M4-006 — IN_PROGRESS / Ready: Yes
```

### M4-006 — M4 qualification과 M5/M6 handoff

**Outcome**

M4 exit criteria와 fail-closed install/Promotion 경계를 전체 evidence에 대조하고 다음에는 M5 entry decision 하나만 열 수 있게 한다.

**Acceptance**

- exact inspected/allowed/Manifest/Staging/Promotion set 동일성과 no mutable re-resolution/no network를 재현 가능한 test와 Linux CI로 확인한다.
- `MANUAL_REVIEW`/`BLOCK`, staging/Promotion failure, existing target와 publish race에서 target 불변/no partial publish를 확인한다.
- result/Evidence/SBOM/cleanup 계약, minimum Go/macOS limitation과 활성 CI gate를 audit한다.
- M4 qualification record와 현재 stage를 갱신하고 M5 구현 없이 M5-001 entry decision 하나만 handoff한다.

**Completion**

```text
Status: COMPLETE
Commit: 1dc22b8
CI: GitHub Actions final qualification run 31786209026 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: canonical quick/docs/platform; go test -race -timeout=5m ./...; JSON Draft 2020-12 metaschema; M4 exit-criteria and production-boundary audit; git diff --check
Decision/Evidence: docs/planning/07-m4-qualification.md; exact set/Manifest/SBOM/Staging/Promotion binding; offline no-network target publish; fail-closed non-ALLOW and operational failure results
Limitations: automatic Promotion is Linux amd64/new empty target/script-free only; macOS and other platforms do not enter resolver Host commands; completed Staging retention/GC remains manual
Next: M5-001 — NOT_STARTED / Ready: Yes
```

## 4. Work Item

### M5-001 — PyPI/pip Expansion entry decision

**Outcome**

M5가 기존 generic Artifact/Verified Set/Promotion workflow를 유지한 채 PyPI wheel과 sdist를 연결하도록 exact identity, resolver, build, dynamic inspection과 offline Promotion contract를 확정한다. 이 항목은 implementation을 시작하지 않는다.

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [M4 npm Install and Promotion Contract](../domain-model/13-m4-npm-install-promotion-contract.md)
- [Dependency·Verified Set·Manifest](../domain-model/05-dependency-verified-set.md)
- [Artifact Port Contract](../domain-model/07-contract-artifact-port.md)
- [Verification·Inspection·Sandbox·Policy Contract](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Evidence·Staging·Promotion Contract](../domain-model/09-contract-evidence-staging-promotion.md)
- [M5 milestone scope and exit criteria](./01-milestones.md)

**Acceptance**

- PyPI project/version normalization, selected wheel/sdist identity, declared/observed digest and Linux amd64 target-tag rule가 unambiguous하다.
- Simple API, pip report, candidate graph, default-deny resolver egress와 unsupported/dynamic resolution fail-closed semantics가 정해져 있다.
- wheel static/dynamic inspection, sdist PEP 517 build, derived-wheel reinspection과 no-Host-execution boundary가 정해져 있다.
- generic Verified Set/Manifest/SBOM/Staging/offline Promotion invariant와 existing result boundary를 유지하는 방법이 정해져 있다.
- M5 구현 queue가 한 항목씩 수행 가능한 크기로 정해지고, M5-001에서 source, runtime, dependency, CI job 또는 placeholder가 생성되지 않는다.

**Completion**

```text
Status: COMPLETE
Checks: official PyPA/pip specification review; canonical documentation routing; git diff --check
Decision/Evidence: docs/domain-model/14-m5-pypi-pip-contract.md
Limitations: Python/pip runtime identity, PyPI resolver, wheel/sdist handling, Docker/gVisor execution, CLI and CI implementation 없음
Next: M5-002 — NOT_STARTED / Ready: Yes
```

### M5-002 — Python/pip runtime identity lock과 PyPI reference foundation

**Outcome**

M5 automatic path가 사용할 Linux amd64 CPython/pip resolver·Promotion runtime의 exact identity와 public PyPI project reference parser/capability boundary를 구현한다.

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [Engineering — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)
- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)

**Work**

- current stable CPython/pip releases와 official image provenance를 조사해 exact version, OCI image digest, binary metadata와 capability probe input을 lock한다.
- Linux amd64 / fixed interpreter-ABI-platform tag set을 explicit capability로 만들고 non-Linux 또는 identity mismatch가 resolver/Docker execution으로 bypass되지 않게 한다.
- `project[@exact-version]` parser를 name/PEP 440 canonicalization, unsupported extras/range/marker/URL/local-version rejection과 bounded sanitized error로 구현한다.
- product module에 PyPI/pip library dependency를 추가하지 않는다. network resolver, pip report, Docker policy, wheel/sdist parser와 CLI command는 다음 항목 범위다.

**Acceptance**

- floating Python/pip/image tag가 없고 runtime mismatch/unsupported platform이 explicit non-ALLOW capability다.
- parser가 exact supported reference만 normalized value로 만들며 raw input/environment/path를 public result에 누출하지 않는다.
- current product minimum Go, architecture, format/static analysis, deterministic unit/fuzz boundary tests와 existing npm regression이 통과한다.

**Completion**

```text
Status: COMPLETE
Checks: official Python/pip/Docker source review; focused unit/race; 5-second parser fuzz; canonical docs/quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: Python 3.14.7 + pip 26.2.1 immutable Docker Official Image lock; Linux amd64 cp314/cp314/manylinux_2_36_x86_64 target basis; bounded public PyPI reference parser and explicit capability probe
Limitations: PyPI Simple API/resolver/report, resolver egress policy, wheel/sdist handling, Docker/gVisor Python execution, CLI and CI integration 없음
Next: M5-003 — NOT_STARTED / Ready: Yes
```

### M5-003 — isolated PyPI resolver·target tag·egress policy

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)
- [Engineering — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- pinned Python/gVisor resolver에서 pip installation report v1을 bounded parse하고 Simple API JSON v1 metadata와 exact cross-check한다.
- fixed target compatible-tag closure를 verify하고, wheel-only M5-003 resolver profile에서 unsupported sdist/marker/extras/direct requirement를 fail-closed 한다.
- `pypi.org`와 `files.pythonhosted.org` public IP set만 default-deny network policy·explicit container host mapping에 적용하고, observer lifecycle까지 completion requirement로 둔다.
- Linux controlled integration/required CI에서 immutable Python image와 exact helper/runtime path를 실행한다.

**Acceptance**

- report/Simple URL·filename·SHA-256·Requires-Python/yanked/size, runtime/tag, egress endpoint 및 observer stream 중 어느 하나라도 incomplete면 graph가 반환되지 않는다.
- Core/Application/Policy에는 PyPI raw report/index/URL query/pip output/runtime implementation detail가 유입되지 않는다.
- focused parser/resolver unit·race, existing regression, canonical quick/docs와 controlled Linux PyPI integration이 통과한다.

**Completion**

```text
Status: COMPLETE
Commits: 22584ca, 79524d1, a82e348, aaaec7b, d7ec08d
CI: GitHub Actions run 32335942809 — gVisor Observer Helper, Linux PyPI integration, Quick, Minimum Go, macOS, Docs, Required success
Checks: focused unit/race; parser fuzz; canonical quick/docs; official PyPI Simple API v1 URL/hash validation; git diff --check
Decision/Evidence: pinned Python 3.14.7/pip 26.2.1; isolated runsc-trace resolver; default-deny PyPI endpoint policy; observer lifecycle and report/Simple cross-check
Limitations: wheel intake/integrity/static inspection, dynamic inspection, sdist build, staging/Promotion and CLI remain in later M5 items
Next: M5-004 — IN_PROGRESS / Ready: Yes
```

### M5-004 — wheel intake·integrity·static inspection

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)
- [Engineering — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Engineering — Quality Toolchain](../engineering/03-quality-toolchain.md)

**Work**

- exact resolver-selected wheel을 controlled intake에서 bounded ZIP reader로 읽고 observed SHA-256과 declared digest를 분리 검증한다.
- ZIP containment, compressed/uncompressed size·file count bound, regular-file-only, symlink/special/path escape/archive bomb를 fail-closed로 검사한다.
- normalized project/version과 `.dist-info/METADATA`, `.dist-info/WHEEL`, `.dist-info/RECORD`의 정확한 일치, supported Wheel-Version·target tag를 검증한다.
- RECORD의 permitted empty self-entry 외 모든 파일 digest/size, duplicate/unrecorded installable content, `.data/scripts`와 native extension surface를 bounded normalized evidence로 만든다.
- Core Metadata의 Requires-Dist, Requires-Python, Import-Name, entry points, license/provenance를 제한된 adapter evidence로 보존하고 Core/Policy에 raw wheel payload를 유입하지 않는다.

**Acceptance**

- malformed/ambiguous/mismatched wheel, path escape, archive bomb, symlink/special file, RECORD 누락/불일치, unsupported tag는 artifact 또는 static result를 반환하지 않는다.
- exact filename·normalized identity·observed/declared SHA-256·target compatibility와 bounded static evidence가 deterministic하게 재현된다.
- focused unit/property/fuzz 경계, race, existing npm/PyPI regression, canonical quick/docs와 architecture gate가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 6575594
CI: GitHub Actions run 32338211203 — gVisor Observer Helper, Linux integration, Quick, Minimum Go, macOS, Docs, Required success
Checks: go test ./...; go test -race -timeout=5m ./...; 5-second wheel parser fuzz; canonical docs/quick; git diff --check
Decision/Evidence: bounded non-executing ZIP intake; observed/declared SHA-256; target/WHEEL tag agreement; dist-info and RECORD digest/size containment; bounded static surface evidence
Limitations: dynamic wheel inspection, sdist build/derived wheel, staging/Promotion and CLI remain in later M5 items
Next: M5-005 — IN_PROGRESS / Ready: Yes
```

### M5-005 — Python dynamic inspection·PEP 517 sdist build

**Read**

- [M5 PyPI/pip Expansion Contract](../domain-model/14-m5-pypi-pip-contract.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Engineering — Coding / Security Rules](../engineering/02-coding-security-rules.md)

**Work**

- verified wheel만 fresh `runsc-trace` session에 private tmpfs로 반입하고 `pip --no-index --no-deps`로 설치한 뒤, static inspection이 확정한 bounded import surface만 실행한다.
- observer attribution, stream completion, command exit, resource limit, introduction/cleanup failure가 모두 non-ALLOW incomplete로 정규화되게 한다. artifact script, entry point, arbitrary module, Host Python은 실행하지 않는다.
- `.tar.gz` PEP 517 sdist를 bounded static reader로 검증하고 exact build-system requirement와 in-tree backend 경계를 정규화한다. legacy `setup.py`, malformed backend path, graph 밖 build requirement는 automatic path에서 제외한다.
- verified source와 already verified build wheel만 private/no-network gVisor session에서 `pip wheel --no-index --no-deps --no-build-isolation`으로 build한다. one derived wheel, source/build recipe binding, re-inspection boundary를 fail-closed로 만든다.

**Acceptance**

- wheel dynamic run은 verified local content와 declared import surface 이외를 실행하지 않으며 observer/lifecycle 불완전 시 completed inspection을 반환하지 않는다.
- sdist path/archive/metadata/build backend/build requirement/output wheel mismatch 및 dynamic dependency expansion은 derived wheel을 반환하지 않는다.
- raw pip/backend output, Host path, credentials, gVisor implementation detail는 Core/Application/Policy/public result에 유입되지 않는다.
- focused unit/fuzz/race, existing npm/PyPI regression, canonical quick/docs 및 controlled Linux integration이 통과한다.

**Completion**

```text
Status: COMPLETE
Commits: f442042, 2e76b0b, 18a8f4a, 9ae8c55, 039ad58
CI: GitHub Actions run 32343138560 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: focused unit/fuzz/race; canonical quick/docs; verified local wheel dynamic import; PAX sdist PEP 517 no-network derived-wheel build and static reinspection; git diff --check
Decision/Evidence: fresh runsc-trace session; trusted observer completion; controller-generated valid local artifact filenames; derived-wheel identity/source-build recipe binding; bounded raw-output discard and test diagnostics
Limitations: generic staging/offline pip Promotion, CLI/bootstrap/result wiring and final M5 qualification remain M5-006/M5-007 scope
Next: M5-006 — IN_PROGRESS / Ready: Yes
```

### M5-006 — generic staging·offline pip Promotion·CLI integration

**Completion**

```text
Status: COMPLETE
Commits: 4b96ab6, 110502b, 1bc6317, 4d99cee, 8f4f38b, 8eb949b
CI: GitHub Actions run 32349328808 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: focused race; canonical quick/docs; real pinned Docker offline pip Promotion; gVisor PEP 517 derived-wheel build; git diff --check
Decision/Evidence: exact primary PyPI inspect CLI; generic source→derived graph binding; controller SHA-256/filename record; derived static/dynamic reinspection; no-network hash requirements Promotion and post-install RECORD validation
Next: M5-007 — IN_PROGRESS / Ready: Yes
```

### M5-007 — M5 qualification·npm regression·M6 handoff

**Acceptance**

- M5 wheel/sdist exact identity, source-to-derived binding, Sandbox-only build,
  complete per-entry inspection and unsupported capability fail-closed behavior
  are audited against the canonical contract.
- existing npm inspect/install/Promotion regression remains green without
  changing npm semantics.
- canonical Quick/Docs/Minimum Go/macOS and controlled Linux gVisor/observer,
  npm resolver and PyPI resolver/dynamic/build/Promotion integration gates pass.
- qualification evidence and current stage are recorded; M6 entry decision is
  the only next work item and no M6 implementation is introduced.

**Completion**

```text
Status: COMPLETE
Commit: 42858b1
CI: GitHub Actions workflow_dispatch run 32352463714 — Quick, Docs, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: security-boundary audit; focused unit/race; canonical quick/docs; full go test -race -timeout=5m ./...; git diff --check; existing npm regression; controlled Linux PyPI resolver, dynamic wheel, PEP 517 sdist build, npm resolver and PyPI Promotion integration
Decision/Evidence: docs/planning/08-m5-qualification.md; complete ALLOW build-input boundary; derived-wheel build Evidence and generic source/input/config Manifest binding; npm semantics unchanged; pinned Linux runtime and observer qualification
Limitations: automatic PyPI Promotion remains Linux amd64/new empty target only; M6 GitHub Releases implementation is not started; M6 entry decision is next
Next: M6 entry decision — NOT_STARTED / Ready: Yes
```

## 5. Historical work items

### M3-001 — Linux Dynamic Inspect entry decision

**Outcome**

Linux dynamic inspection의 runtime identity, Session·observation boundary, resource limit, M3 Policy direction과 fixture를 확정한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)

**Completion**

```text
Status: COMPLETE
Checks: official gVisor/Docker/Node source review; gVisor release-20260810.0 annotated tag commit verification; canonical docs; git diff --check
Decision/Evidence: docs/domain-model/12-m3-linux-dynamic-contract.md
Limitations: current macOS workstation is unsupported for Linux runtime installation; no M3 package, CI job or runtime config created
Next: M3-002 — Ready: Yes
```

### M3-002 — gVisor runtime lock과 capability probe

**Outcome**

Linux-only gVisor runtime installer identity와 fail-closed capability probe를 current worktree에 필요한 범위로 구현한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Engineering — Coding and Security Rules](../engineering/02-coding-security-rules.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- release-20260810.0 binary/archive SHA-512와 Node image digest를 runnable lock input으로 record한다.
- Linux host/kernel/Docker/runsc/runtime/image capability를 probe하고 unsupported/failure를 explicit result로 구분한다.
- tool install과 image pull은 Linux integration environment만 수행하도록 하고 current macOS/default Quick profile에 실행 경로를 만들지 않는다.
- probe unit/contract test와 architecture gate를 추가한다.

**Acceptance**

- floating latest/tag runtime or image identity가 없다.
- macOS/non-Linux invocation이 hidden success가 아니라 explicit unsupported capability다.
- probe success가 Docker CLI 존재만으로 sandbox safety를 주장하지 않고 runsc/runtime/image identity를 확인한다.
- format, unit, static analysis와 architecture check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 5a7050a
Checks: canonical quick; focused race test; git diff --check
Decision/Evidence: bounded Session ID/status/observation types and consumer-owned Sandbox Port
Limitations: backend, observation collector, dynamic inspector and Policy wiring 없음
Next: M3-004 — Ready: Yes
```

### M3-004 — gVisor Sandbox Session lifecycle

**Outcome**

pinned gVisor runtime을 사용하는 Linux one-shot Session backend의 create/execute/terminate/dispose lifecycle과 cleanup을 구현한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)

**Work**

- injected process boundary로 gVisor Docker runtime command와 bounded cleanup을 구현한다.
- no bind mount, non-root, no-new-privileges, read-only rootfs, network none, resource limit command invariants를 test한다.
- one-time controlled Artifact introduction, timeout/forced termination/disposal와 non-reuse result를 구현한다.
- macOS/unsupported probe result가 backend execution으로 bypass되지 않게 한다.

**Acceptance**

- backend는 Finding/Evidence/Policy를 만들지 않고 raw `SandboxResult`만 반환한다.
- timeout, process failure와 cleanup failure는 completed result로 변환되지 않는다.
- untrusted Artifact에 Host path/socket/environment/credential을 전달하는 command option이 없다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: a3c731e
Checks: canonical quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: injected Docker command boundary; constrained lifecycle unit tests
Limitations: Linux gVisor runtime integration and observer trace ingestion remain unavailable on this macOS workstation
Next: M3-005 — Ready: Yes
```

### M3-003 — Sandbox Session·Observation Domain과 Port

**Outcome**

Dynamic runtime의 raw Session/Observation lifecycle을 ecosystem-neutral Domain과 consumer-owned Port로 정의한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Domain — Verification, Evidence, and Finding](../domain-model/04-verification-evidence-finding.md)

**Work**

- bounded Session identity, execution state, raw observation category와 limitation type을 정의한다.
- `core/ports`에 dynamic inspection consumer가 필요한 Sandbox contract를 추가한다.
- Sandbox가 Finding/Policy/Evidence reference를 직접 만들 수 없도록 type/API를 제한한다.
- constructor/state/Port contract test와 architecture gate를 추가한다.

**Acceptance**

- raw observation과 normalized Finding/Evidence/Policy Decision이 같은 type이나 API에서 섞이지 않는다.
- timeout/limit/observer failure는 completed clean result가 아니라 bounded non-completed execution으로 표현된다.
- Domain은 runtime/Docker/gVisor/package-manager type을 import하지 않는다.
- format, unit, static analysis와 architecture check가 통과한다.

### M3-005 — gVisor raw observation collector

**Outcome**

trusted gVisor trace observer 입력을 bounded raw `SandboxObservation`으로 변환하고 Session lifecycle에 연결한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [Architecture — Inspection, Sandbox, and Policy](../architecture/03-inspection-sandbox-policy.md)

**Work**

- trusted observer transport와 trace record의 bounded adapter contract를 정의한다.
- process/filesystem/network/honeytoken/resource observation을 raw bounded category/subject로 정규화한다.
- trace overflow, observer failure와 malformed record를 completed clean result가 아닌 explicit incomplete limitation으로 처리한다.
- artifact output, raw path, argv, environment, contents가 normal result에 포함되지 않는 contract test를 추가한다.

**Acceptance**

- Sandbox가 Finding/Evidence/Policy를 만들지 않고 raw observation과 execution limitation만 반환한다.
- observer가 Artifact process와 분리된 trusted boundary이고 artifact-controlled endpoint가 아니다.
- trace size/event bound와 failure reason이 deterministic하게 검증된다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: pending current commit
Checks: canonical quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: bounded trace collector and trusted observer-before-introduction lifecycle test
Limitations: a Linux gVisor seccheck transport implementation remains an integration-environment concern
Next: M3-006 — Ready: Yes
```

### M3-006 — Dynamic observation normalization

**Outcome**

M3 raw Sandbox observation을 Inspection의 bounded Finding/Evidence와 required dynamic check 상태로 정규화한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)
- [Domain — Verification, Evidence, and Finding](../domain-model/04-verification-evidence-finding.md)

**Work**

- M3 observation category를 documented Finding/Evidence와 incomplete execution 상태로 매핑한다.
- raw trace subject/payload가 Finding/Evidence summary에 전파되지 않게 한다.
- dynamic inspector Port/adapter를 구성하되 Sandbox와 Policy의 책임 경계를 유지한다.
- clean, suspicious, blocking, timeout/observer failure fixture를 deterministic하게 test한다.

**Acceptance**

- Sandbox는 Finding/Evidence/Policy를 직접 만들지 않는다.
- incomplete/unsupported dynamic check가 clean success나 ALLOW로 변환되지 않는다.
- 결과가 raw Host path, argv, environment, file contents, trace payload를 포함하지 않는다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: pending current commit
Checks: canonical quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: npm DynamicInspector converts raw M3 observations without carrying trace payloads
Limitations: static and dynamic inspection report composition and M3 Policy wiring are next work
Next: M3-007 — Ready: Yes
```

### M3-007 — M3 Policy와 npm inspect wiring

**Outcome**

정적·dynamic inspection을 한 M3 npm result로 구성하고 M3 Policy v1/CLI controlled vertical test를 연결한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)

**Work**

- static/dynamic Inspection report를 subject-preserving composite inspector로 구성한다.
- `m3-npm-dynamic-inspect` Policy v1을 dynamic required check 상태와 Finding taxonomy에 연결한다.
- production wiring은 unsupported Linux dynamic capability를 success로 바꾸지 않게 한다.
- controlled clean/manual/block/incomplete CLI vertical fixture를 추가한다.

**Acceptance**

- 모든 required verification/static/dynamic check가 complete인 clean fixture만 M3 ALLOW가 된다.
- dynamic incomplete/unsupported는 MANUAL_REVIEW이며 raw runtime payload가 CLI/Evidence에 없다.
- Honeytoken/filesystem violation은 BLOCK, network/unexpected process는 MANUAL_REVIEW로 재현된다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: pending current commit
Checks: canonical quick; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: M3 composite inspection/Policy and controlled allow/manual/block vertical fixtures
Limitations: current macOS composition explicitly returns M3_LINUX_ONLY; a production Linux runsc trace transport needs Linux integration validation
Next: M3-008 — Ready: Yes
```

### M3-008 — M3 qualification과 M4 handoff

**Outcome**

M3 구현 범위와 fail-closed runtime limitation을 qualification으로 기록하고 M4 entry decision 하나로 handoff한다.

**Read**

- [M3 Linux Dynamic Inspect Contract](../domain-model/12-m3-linux-dynamic-contract.md)
- [M3 milestone](./01-milestones.md#7-m3--linux-dynamic-inspect)

**Work**

- M3 source/contract/quality gates와 controlled vertical fixtures를 audit한다.
- macOS runtime limitation과 Linux integration validation requirement를 명시한다.
- M4 entry decision만 Ready 상태로 열고 future install/promotion source나 CI placeholder는 만들지 않는다.

**Acceptance**

- unsupported runtime이 ALLOW가 되지 않는 evidence가 있다.
- sandbox/inspection/policy 책임 경계와 raw-data redaction이 재현 가능하게 검증된다.
- M4 구현은 시작하지 않고 entry decision만 열린다.

**Completion**

```text
Status: COMPLETE
Commit: c4ab899
CI: GitHub Actions run 31694083568 — Docs, Quick, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Checks: canonical quick/docs/platform; go test -race -timeout=5m ./...; exact gVisor Bazel helper build; pinned Docker/runsc Linux lifecycle integration; git diff --check
Decision/Evidence: M3 Qualification — Linux Dynamic Inspect
Limitations: macOS/unsupported Linux dynamic runtime is incomplete and never ALLOW; Docker direct runtime remains MVP path
Next: M4-001 — Ready: Yes
```

### M4-001 — npm Install and Promotion entry decision

**Outcome**

npm dependency Verified Set과 trusted Promotion을 구현하기 전, M4의 Install Context·dependency resolution·manifest·staging·offline install·rollback 계약을 canonical 문서에 확정하고 이후 구현 queue를 연다.

**Read**

- [M4 — npm Install and Promotion](./01-milestones.md#8-m4--npm-install-and-promotion)
- [Architecture — Evidence, Staging, and Promotion](../architecture/04-evidence-staging-promotion.md)
- [Domain — Dependency and Verified Set](../domain-model/05-dependency-verified-set.md)
- [Contract — Evidence, Staging, and Promotion](../domain-model/09-contract-evidence-staging-promotion.md)
- [Threat Model — Fail-Closed and Promotion](../threat-model/03-fail-closed-and-promotion.md)

**Acceptance**

- original Install Context와 option allowlist, exact dependency set/manifest, staging와 Promotion failure semantics가 구현 전에 모호하지 않다.
- mutable re-resolution, manifest 밖 dependency, network access와 partial failure가 fail-closed로 처리된다.
- M4의 첫 구현 item만 queue에 추가하며 installer/staging/Promotion source 또는 CI placeholder를 만들지 않는다.

**Completion**

```text
Status: COMPLETE
Checks: official npm ci/package-lock/cache documentation and CycloneDX 1.7 specification review; canonical docs; git diff --check
Decision/Evidence: M4 npm Install and Promotion Contract
Scope: new empty target only; locked resolver graph; every entry inspection; manifest-bound staging; script-free offline local Promotion; atomic target publish
Limitations: existing project/global/workspace/custom registry/auth/npm option passthrough and Host lifecycle/native install are out of M4 MVP automatic Promotion
Next: M4-002 — Ready: Yes
```

### M4-002 — Install Context와 locked npm resolver boundary

**Outcome**

새 empty target Install Context와 disposable npm dependency resolver를 product Domain/Application/Port boundary에 도입하고, resolver output이 bounded exact graph candidate로만 흐르도록 구현한다.

**Read**

- [M4 npm Install and Promotion Contract](../domain-model/13-m4-npm-install-promotion-contract.md)
- [Domain — Operation Request and Context](../domain-model/06-operation-request-context.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [Architecture — Foundation and Dependencies](../architecture/01-foundation-and-dependencies.md)
- [Engineering — Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Work**

- bounded absolute new-target Install Context와 no-overwrite preflight Domain contract를 추가한다.
- consumer-owned Port에 sanitized locked dependency graph resolution contract를 추가하고, npm/runtime type이 Core/Application에 유입되지 않게 한다.
- deterministic fake/contract tests로 unsupported source, invalid lock semantics, duplicate/path escape, missing integrity와 existing target을 fail-closed로 검증한다.
- M4 resolver runtime/docker/network implementation, npm CLI invocation, graph acquisition/inspection, Staging/Promotion source와 CI job은 만들지 않는다.

**Acceptance**

- existing project·global install·workspace·option pass-through가 Install Context 또는 Port API로 우회 유입되지 않는다.
- resolved graph candidate가 exact name/version/registry source/SRI/edge information 없이 생성되지 않는다.
- Core Domain은 npm, Docker, Node/npm lockfile payload 및 Host filesystem type을 import하지 않는다.
- format, unit, static analysis와 architecture check가 통과한다.

**Completion**

```text
Status: COMPLETE
Checks: focused Domain/Application tests; canonical quick; git diff --check
Implementation: canonical new-target Install Context, bounded exact dependency graph Domain values, consumer-owned DependencyResolver Port and request boundary
Safety: Core receives no npm/Docker/Node payload or Host filesystem API; graph is connected, bounded, exact and integrity-required; existing target behavior remains outside a future trusted Promotion boundary
Limitations: no npm resolver runtime, graph acquisition/inspection, Staging, Promotion, CLI command or CI job
Next: M4-003 — Ready: Yes
```

### M0-001 — Project Identity 결정

**Outcome**

구현과 배포 identity로 사용할 Go module path와 `helox` command 이름을 확정한다.

**Read**

- [Project Decisions](../../PROJECT-DECISIONS.md)
- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)

**Work**

- 현재 repository와 module root를 기준으로 canonical Go module path 후보를 확인한다.
- 현재 repository, 주요 executable namespace와 현실적인 배포 경로에서 `helox` 충돌 가능성을 확인한다.
- command 이름을 유지할 수 없으면 구현 전에 대안을 결정하고 관련 canonical 문서를 갱신한다.
- 결정 근거와 확인 시점을 Project Decisions에 연결한다.

**Acceptance**

- module path와 CLI command가 하나의 값으로 확정되어 후속 파일에서 추측할 필요가 없다.
- Heliopause 독립 repository root가 단일 Go module root라는 Step 8 결정을 유지한다.
- 실제 scaffold, `go.mod` 또는 package는 아직 생성하지 않는다.

**Completion**

```text
Status: COMPLETE
CLI command: helox
Repository: github.com/rahoney/heliopause
Go module path: github.com/rahoney/heliopause
Evidence: local repository/PATH와 현실적인 public executable/package namespace 확인
Limitation: 법적 상표 검토와 최종 배포 channel 등록 가능성은 배포 전 재검토
Next: M0-002 — Ready: Yes
```

### M0-002 — Go·Tool·CI Identity lock

**Outcome**

M0 구현에 필요한 기본 Go/minimum Go, 최초 Staticcheck, GitHub-hosted runner와 Action identity를 당시 공식 지원 상태에 맞춰 고정한다.

**Read**

- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- 공식 source에서 안정 Go patch와 최소 지원 Go patch를 확인한다.
- versioned Ubuntu/macOS runner label의 지원 상태를 확인한다.
- 필요한 GitHub-owned Action release와 full commit SHA를 검증한다.
- M0 Quick profile에 필요한 Staticcheck exact version, setup Go 요구, license, 유지보수와 dependency risk를 검토한다.
- 선택 결과와 upgrade 원칙을 구현 가능한 lock input으로 기록한다.

**Acceptance**

- floating `latest`, version range, branch 또는 tag만으로 식별된 실행 요소가 없다.
- default/minimum Go의 역할과 runner/Action/tool identity가 서로 구분된다.
- 아직 사용하지 않는 gosec, Gitleaks, govulncheck 또는 미래 runtime을 미리 pin하지 않는다.
- source URL, 확인한 release/tag와 exact identity가 review 가능하게 기록된다.

**Completion**

```text
Status: COMPLETE
Commit: c6a353b71d5a6659b97e00a17a2e61fbba9ef82a
Checks: official release/support source review; tag/SHA ls-remote verification; local go/staticcheck version verification; Markdown local link/fence validation; git diff --check
Decision/Evidence: docs/engineering/03-quality-toolchain.md and docs/engineering/04-ci-quality-gate.md M0-002 identity lock
Limitations: tools.lock consumer는 M0-005, workflow consumer는 M0-006에서 생성하며 미래 security tool은 capability 활성화 시 별도 pin
Next: M0-003 — Ready: Yes
```

### M0-003 — Go module과 최소 process bootstrap

**Outcome**

프로젝트 루트의 단일 Go module과 `helox`의 가장 얇은 실행 경로를 만든다.

**Read**

- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Architecture — Foundation and Dependencies](../architecture/01-foundation-and-dependencies.md)

**Work**

- project root에 `go.mod`를 생성하고 M0-002의 Go 정책을 적용한다.
- 필요한 범위에서만 `cmd/helox`, `internal/bootstrap`, `internal/cli`를 생성한다.
- process cancellation과 exit ownership을 경계에 맞게 두고 business workflow는 추가하지 않는다.
- 최소 process smoke test와 Linux/macOS build validity를 마련한다.

**Acceptance**

- 별도 `src/`, nested module과 미래용 placeholder package가 없다.
- `cmd/helox`는 bootstrap 호출과 process exit만 소유한다.
- CLI/bootstrap에 Domain, Policy, Adapter shortcut이 없다.
- module과 smoke test가 public network·credential·Host project 없이 재현된다.

**Completion**

```text
Status: COMPLETE
Commit: 5e99b6028009af9506232e291dbda60a09ca9450
Checks: gofmt; go mod tidy -diff/verify; go build; compile-only test; go vet; Staticcheck 2026.1; full test; Linux amd64/macOS amd64·arm64 build; offline module list/graph; package import boundary; Markdown local link/fence validation; git diff --check
Implementation: root go.mod; cmd/helox process entrypoint; internal/bootstrap composition root; internal/cli Cobra boundary; host-independent process smoke test
Limitations: business workflow·Domain·Policy·Adapter·future package와 canonical check runner는 현재 항목 범위에서 추가하지 않음
Next: M0-004 — Ready: Yes
```

### M0-004 — Canonical check runner foundation

**Outcome**

로컬과 CI가 공유할 standard-library 기반 `scripts/check` entrypoint와 deterministic foundation profile을 만든다.

**Read**

- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)

**Work**

- module root에서만 실행되는 `go run ./scripts/check <profile>` 구조를 만든다.
- read-only `format check`, module drift, build/type validity와 `docs` 검사를 구현한다.
- local Markdown link, code fence와 의도하지 않은 absolute local link를 검사한다.
- 현재 존재하는 package에 대해 검증 가능한 import rule을 구현하고, 이후 새 package가 추가되면 같은 checker에 규칙을 증분한다. 존재하지 않는 미래 package를 위한 fake pass 규칙은 만들지 않는다.
- command/argument 분리, timeout, bounded output과 첫 실패 보존을 적용한다.

**Acceptance**

- `format` 외 profile은 source/configuration을 변경하지 않는다.
- check가 source tree 밖 임의 executable이나 PATH 우선순위에 의존하지 않는다.
- finding과 tool/check execution failure가 모두 non-zero이며 원인이 구분된다.
- 문서 router가 깨진 fixture와 정상 tree를 deterministic하게 판별한다.

**Completion**

```text
Status: COMPLETE
Commit: f307039b916408383f76391fdc313c2f93914126
Checks: canonical foundation/docs; go vet; Staticcheck 2026.1; full test; race test; Linux amd64/macOS amd64·arm64 build; module-root rejection; unknown-profile failure; read-only source hash comparison; git diff --check
Implementation: standard-library scripts/check; foundation/docs/format profiles; offline local-toolchain command runner; timeout/bounded output/first-failure semantics; current-package architecture rules; Markdown link/fence/absolute-local-path validator
Fixtures: healthy/broken Markdown tree; package shortcut; format drift; module drift; root guard; output truncation; sequential failure preservation
Limitations: Staticcheck lock/bootstrap, quick profile, security tools and CI jobs are deferred to their queued work items
Next: M0-005 — Ready: Yes
```

### M0-005 — Static analysis와 Quick profile 활성화

**Outcome**

Go 기본 검사와 pinned Staticcheck를 canonical `quick` profile로 연결한다.

**Read**

- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)

**Work**

- `scripts/tools.lock.json`에 실제 사용하는 Staticcheck identity만 기록한다.
- source tree 밖 project-specific cache를 사용하는 `bootstrap`을 구현한다.
- `quick`에 format check, module drift, build/type validity, architecture, `go vet`, Staticcheck와 default test를 연결한다.
- bootstrap network phase와 offline check phase를 분리하고 version mismatch를 실패시킨다.

**Acceptance**

- product `go.mod`가 quality tool dependency graph를 소유하지 않는다.
- `@latest`, floating download, `curl | sh`와 임의 PATH executable 실행이 없다.
- `quick`은 read-only이고 local clean checkout에서 반복 가능하다.
- M0-002의 pinned identity와 실제 실행 binary version이 일치한다.

**Completion**

```text
Status: COMPLETE
Commit: dff6682f152ce2cb59c85abdb2e3cec6da8fa961
Checks: official Staticcheck release recheck; strict lock tests; bootstrap source/config hash comparison; exact cached binary version; repeated canonical offline quick; canonical docs; go vet; Staticcheck; full/race test; Linux amd64/macOS amd64·arm64 build; git diff --check
Implementation: scripts/tools.lock.json Staticcheck-only identity; isolated source-tree-external cache; verified proxy/checksum bootstrap; exact Go setup check; offline quick with format/module/build/type/architecture/vet/Staticcheck/default test
Safety: product module download uses a cache-internal temporary module root; relative/in-tree/symlink-into-tree caches, floating lock values, unknown lock fields, setup Go mismatch, missing binary and version mismatch fail closed
Limitations: CI workflow and repository merge gate are deferred to M0-006/M0-007; security tools remain unpinned until their capability work item
Next: M0-006 — Ready: Yes
```

### M0-006 — Quick·Docs·Required CI foundation

**Outcome**

GitHub Actions에서 실제 local profile을 실행하는 첫 required workflow를 만든다.

**Read**

- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- repository root에 Heliopause 전용 CI workflow를 생성하고 module working directory를 명시한다.
- versioned Ubuntu와 pinned Go/Action identity로 `Quick`, `Docs`, `Required`를 활성화한다.
- `pull_request`, `push` to `main`, `merge_group`, bounded `workflow_dispatch` trigger를 적용한다.
- fork/PR secret 없음, 최소 token permission, checkout credential 비보존과 cache 비활성화를 적용한다.
- `Required`가 failure/cancel/unexpected skip을 성공으로 변환하지 않게 한다.

**Acceptance**

- CI가 local canonical profile과 다른 검사 명령을 재구현하지 않는다.
- 모든 `uses:`가 full SHA와 사람이 식별 가능한 release 주석을 가진다.
- placeholder Security/Vulnerability/Race/Integration/E2E job이 없다.
- 실제 child failure와 cancellation에 aggregate가 실패하는 test 또는 검증 evidence가 있다.

**Completion**

```text
Status: COMPLETE
Commit: 53fb846bdd5571d610ebf757b31128c566fc16f7
Branch: milestone/m0-foundation (origin tracking)
Checks: official runner/action identity recheck; action tag full-SHA ls-remote; YAML syntax parse; canonical quick/docs; gofmt; module tidy/verify; build/test-build; go vet; Staticcheck; full/race test; workflow security fixture; git diff --check
Implementation: pull_request/main push/merge_group/workflow_dispatch triggers; ubuntu-24.04; checkout v6.0.2 and setup-go v7.0.0 full SHA; exact Go 1.26.5; no Actions cache; credential persistence disabled; Quick/Docs/Required only
Aggregate evidence: success+success passes; failure, cancelled and skipped child results each return non-zero; missing child is execution failure; Required uses always() and depends on Quick+Docs
Limitations: feature branch push intentionally does not trigger CI; actual GitHub-hosted run occurs when the milestone PR is opened. Minimum Go/macOS jobs and repository settings are M0-007 scope
Next: M0-007 — Ready: Yes
```

### M0-007 — Minimum Go·macOS와 repository gate 검증

**Outcome**

지원 toolchain/platform 경계를 실제 CI에서 검증하고 stable aggregate를 repository merge gate로 사용할 준비를 마친다.

**Read**

- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)
- [Step 12 — Milestones](./01-milestones.md)

**Work**

- minimum Go Ubuntu와 pinned Go macOS build/default test job을 활성화한다.
- 두 job을 `Heliopause CI / Required` dependency에 포함한다.
- 독립 repository의 workflow 이름과 repository setting 적용 영향을 확인한다.
- branch rule과 Actions SHA/allowlist setting의 적용 범위를 제안하고, repository-wide 변경은 owner의 명시적 승인을 받은 경우에만 적용한다.

**Acceptance**

- primary, minimum Go와 macOS 결과가 서로 독립적으로 실제 실행된다.
- unavailable runner, failed 또는 skipped required job이 aggregate success가 되지 않는다.
- Linux build 결과를 WSL2 qualification으로 표시하지 않는다.
- repository-wide branch rule 적용은 M0 완료의 필수 조건이 아니다. 적용 권한·운영 범위가 없으면 제안과 미적용 사유를 기록하고, Heliopause CI 자체의 gate 동작 검증이 완료되면 M0 진행을 차단하지 않는다.

**Completion**

```text
Status: COMPLETE
Commit: 0ec5b6575f29ee5b8a91af96182650781ced5c0f
PR/CI: draft PR #1; run 31663728891; Quick, Docs, Minimum Go, macOS, Required success
Checks: actionlint 1.7.12; canonical quick/docs/platform; Go 1.25.12 platform; workflow validator tests; git diff --check
Repository settings: Actions enabled, allowed_actions=all, sha_pinning_required=false; main protection 없음; ruleset 없음
Decision/Evidence: Step 11 M0-007 repository setting 점검과 적용 제안
Limitations: Linux 결과는 WSL2 qualification이 아님. repository-wide Actions allowlist/SHA pinning/branch protection은 별도 owner 적용 승인 없이 변경하지 않음
Next: M0-008 — Ready: Yes
```

### M0-008 — M0 qualification과 M1 handoff

**Outcome**

M0 exit criteria를 evidence로 닫고 다음에는 M1 entry decision만 열 수 있게 한다.

**Read**

- [Step 12 — Milestones](./01-milestones.md)
- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- M0의 모든 exit criteria와 Global Definition of Done을 실제 command/CI evidence에 대조한다.
- version 공급망 검토, active check 목록, 제한과 유보 사항을 기록한다.
- PROJECT-DECISIONS, experiment README와 이 queue의 상태를 갱신한다.
- M1 entry decision을 실행 가능한 design work item으로 분해하되 M1 구현 항목 전체를 미리 만들지 않는다.

**Acceptance**

- M0-001~007이 모두 `COMPLETE`이고 미해결 필수 gate가 없다.
- clean checkout의 canonical local check와 활성 CI가 통과한다.
- M0 milestone completion commit과 검증 결과를 참조할 수 있다.
- 다음 active item은 M1 entry decision 하나이며 미래 Adapter/Sandbox/Promotion placeholder가 없다.

**Completion**

```text
Status: COMPLETE
Commit: 011dd4f
PR/CI: draft PR #1; checkout v7.0.1 qualification run 31664225839과 final completion run 31664383069의 Quick, Docs, Minimum Go, macOS, Required success
Checks: clean checkout fresh bootstrap; canonical quick/docs/platform; actionlint 1.7.12; single module/no placeholder audit; M0 exit criteria와 Global DoD 대조; git diff --check
Decision/Evidence: M0 Qualification — Implementation Foundation
Limitations: repository-wide Actions allowlist/SHA requirement/main protection은 별도 owner 승인 없이 미적용; WSL2와 미래 security/runtime capability는 qualification 대상 아님
Next: M1-001 — Ready: Yes
```

### M1-001 — Domain workflow entry decision

**Outcome**

M1 fake inspect lifecycle을 구현하기 전에 최소 Domain·Application·Port·Policy·CLI 계약과 synthetic scenario를 결정하여 하나의 실행 가능한 구현 queue를 열 수 있게 한다.

**Read**

- [Step 12 — M1 Domain Workflow Skeleton](./01-milestones.md#5-m1--domain-workflow-skeleton)
- [Architecture — Foundation and Dependencies](../architecture/01-foundation-and-dependencies.md)
- [Domain — Core Concepts](../domain-model/01-core-concepts.md)
- [Domain — Artifact Identity](../domain-model/02-artifact-identity.md)
- [Domain — Inspection Run and Status](../domain-model/03-inspection-run-and-status.md)
- [Domain — Verification, Evidence, and Finding](../domain-model/04-verification-evidence-finding.md)
- [Domain — Operation Request and Context](../domain-model/06-operation-request-context.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)

**Work**

- M1에 필요한 최소 Domain type/schema, identifier 생성과 ownership을 결정한다.
- Operation Request/Result, fake Artifact·Verification·Inspection·Evidence Port method와 Application orchestration signature를 결정한다.
- Execution Status, Run Outcome, Policy Decision, Operation Status의 허용 transition과 operational error 경계를 결정한다.
- 최소 Policy input/rule/version/reason과 CLI exit code·machine result schema의 최초 contract를 결정한다.
- `ALLOW`, `MANUAL_REVIEW`, `BLOCK`, operational failure와 `inspect`에서 Promotion 미호출을 증명할 synthetic scenario를 정의한다.
- 결정이 완료된 뒤에만 첫 M1 implementation work item을 Queue에 추가한다.

**Acceptance**

- exact fake identity resolve부터 Run 생성, acquisition/result, Policy와 Operation Result까지 순서·type·owner가 모호하지 않다.
- 네 상태 축과 operational error가 API/schema에서 합쳐지지 않는다.
- 필수 check 미충족은 `ALLOW`가 아니며 `inspect` path에 Staging/Promotion이 없다.
- CLI 사람용·기계용 결과가 같은 Run/identity/decision을 참조하는 contract가 있다.
- synthetic fixture와 expected result가 정의되고 실제 Host credential/network/runtime을 요구하지 않는다.
- 미래 npm Adapter, Sandbox, Evidence Store와 Promotion placeholder를 만들지 않는다.

**Completion**

```text
Status: COMPLETE
Commit: 93903ab
Checks: canonical docs profile; git diff --check
Decision/Evidence: M1 Entry Decision — Domain Workflow Contract
Limitations: 구현과 production fake wiring 없음; npm/network/Sandbox/Evidence Store/Promotion은 범위 밖
Next: M1-002 — Ready: Yes
```

### M1-002 — Domain identity와 Run state foundation

**Outcome**

ecosystem 비종속 identity, digest, Operation/Run identifier와 Run lifecycle을 불변식이 있는 Domain type으로 구현한다.

**Read**

- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [Domain — Artifact Identity](../domain-model/02-artifact-identity.md)
- [Domain — Inspection Run and Status](../domain-model/03-inspection-run-and-status.md)
- [Engineering — Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Work**

- `internal/core/domain`에 bounded identifier, Artifact Reference/Resolved Identity, SHA-256 digest와 acquired subject type을 추가한다.
- `crypto/rand.Reader`를 사용하는 Operation/Run ID generator와 deterministic test 주입에 필요한 parser/validator를 구현한다.
- `CREATED → ACTIVE → FINALIZED` 전이와 completed/failed finalization을 구현한다.
- invalid value, identity/digest mismatch, 역전이와 finalized mutation을 table-driven test로 거부한다.

**Acceptance**

- Domain package가 standard library 외 product/concrete dependency를 import하지 않는다.
- exact resolved identity가 없는 Run을 만들 수 없고 acquired subject mismatch를 결합할 수 없다.
- completed Run에는 Policy Decision이 하나 있고 failed Run에는 Policy Decision이 없다.
- random failure가 error로 반환되고 고정 ID 기반 test가 재현 가능하다.
- format, unit, static analysis와 architecture check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 0fef849
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: public npm reference grammar, abbreviated metadata, exact identity and bounded public-registry transport contract
Limitations: Acquire/Intake/declared integrity/Evidence/CLI wiring 없음; no external production registry를 test success input으로 사용
Next: M2-003 — Ready: Yes
```

**Completion**

```text
Status: COMPLETE
Commit: 2a11310
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: M1 Workflow Contract의 identity, identifier, Run lifecycle와 Policy Decision 추적 계약
Limitations: Port/fake/Application/Policy evaluation/CLI 구현 없음; production network/filesystem/Promotion 없음
Next: M1-003 — Ready: Yes
```

### M1-003 — Check result·Port와 deterministic fake

**Outcome**

Verification·Inspection·Evidence의 상태 의미와 네 outbound Port 계약을 정의하고 synthetic fixture용 fake가 같은 consumer contract를 만족한다.

**Read**

- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [Domain — Verification, Evidence, and Finding](../domain-model/04-verification-evidence-finding.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [Contract — Inspection, Sandbox, and Policy](../domain-model/08-contract-inspection-sandbox-policy.md)

**Work**

- Check Execution, Verification/Inspection Report, Finding, bounded Evidence/Reference type을 구현한다.
- Artifact·Verification·Inspection·Evidence Port를 `internal/core/ports`에 소비자 계약으로 추가한다.
- `safe`, `review`, `blocked`, `acquire-error`, `resolve-error` fake를 test-only 위치에 구현한다.
- contract test로 unsupported, unavailable, incomplete와 operational error를 빈 success와 구분한다.

**Acceptance**

- 모든 Port I/O가 context와 ecosystem-neutral Domain type을 사용한다.
- raw provider output, Host path와 secret이 Evidence에 들어갈 수 없다.
- fake는 network/filesystem/credential/process를 사용하거나 production bootstrap에 연결되지 않는다.
- 후속 호출 순서와 첫 error 이후 stop 여부를 test에서 관찰할 수 있다.
- format, unit, static analysis와 architecture check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: aeb8fb8
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: independent CheckExecution axes, bound Verification/Inspection reports, bounded Evidence/Finding, four M1 Port contracts and five deterministic scenarios
Limitations: fake는 testutil 전용이며 production bootstrap/CLI wiring 없음; Application/Policy evaluation/Promotion 없음
Next: M1-004 — Ready: Yes
```

### M1-004 — Inspect orchestration과 Policy v1

**Outcome**

M1 Port를 순서대로 조율하는 Inspect use case와 deterministic fail-closed Policy v1을 구현한다.

**Read**

- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [Architecture — Foundation and Dependencies](../architecture/01-foundation-and-dependencies.md)
- [Domain — Operation Request and Context](../domain-model/06-operation-request-context.md)

**Work**

- immutable Operation Result와 partial operational failure result를 구현한다.
- injected ID generator와 네 Port로 Inspect application service를 구성한다.
- `m1-fake-inspect` version 1의 ordered Policy reason을 구현한다.
- resolve→Run→acquire→verify→inspect→evidence→policy→finalize 순서와 첫 operational error stop을 test한다.

**Acceptance**

- `ALLOW`, `MANUAL_REVIEW`, `BLOCK`과 resolve/acquire failure가 서로 다른 기대 상태 축을 가진다.
- required check 미완료가 `ALLOW`가 되지 않는다.
- operational failure에는 Policy Decision이 없고 `%w` error cause와 partial result가 보존된다.
- Application과 Policy가 concrete fake, Cobra, filesystem 또는 Promotion을 import하지 않는다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 2c5e53f, 2aefb3b, 703ebb1
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: bounded archive/manifest/lifecycle-key inspection and intake-separated atomic Evidence records
Limitations: M2 Policy·CLI wiring과 controlled vertical flow 없음
Next: M2-005 — Ready: Yes
```

**Completion**

```text
Status: COMPLETE
Commit: 7091156
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: exact Inspect orchestration order, partial OperationResult, M1 Policy v1 ordered rules and synthetic ALLOW/MANUAL_REVIEW/BLOCK/failure tests
Limitations: CLI presenter/adapter와 production fake wiring 없음; retry/concurrency/Promotion 없음
Next: M1-005 — Ready: Yes
```

### M1-005 — CLI result contract와 synthetic vertical test

**Outcome**

동일한 Operation Result를 사람용·JSON 출력과 안정적 exit code로 변환하고 fake workflow의 end-to-end 계약을 검증한다.

**Read**

- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- `helox.operation-result/v1` JSON presenter와 bounded human presenter를 구현한다.
- exit code 0/1/2/10/20 mapping을 operation status와 Policy Decision으로부터 구현한다.
- injected Inspect use case를 사용하는 CLI adapter test와 다섯 synthetic vertical scenario를 추가한다.
- architecture checker를 새 package의 실제 import direction에 맞게 활성화한다.

**Acceptance**

- human/JSON 결과가 동일 Operation ID, Run ID, exact identity/digest, decision/reason을 참조한다.
- machine mode가 prompt를 표시하지 않고 operational error의 partial result를 버리지 않는다.
- fake source를 production 사용자 command나 bootstrap wiring으로 노출하지 않는다.
- `inspect + ALLOW`를 포함한 어느 경로에도 Staging/Promotion package·Port·호출이 없다.
- canonical quick/docs/platform check와 process/CLI contract test가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 81b6d1f
Checks: canonical quick/docs; go test -race -timeout=5m ./...; check-jsonschema Draft 2020-12 metaschema; production fake/Promotion/Staging audit; git diff --check
Decision/Evidence: helox.operation-result/v1 schema, human/JSON presenter, exit 0/1/2/10/20 mapping and five injected synthetic CLI scenarios
Limitations: presenter/adapter는 production root command에 fake로 wiring되지 않음; 실제 ecosystem command는 M2 범위
Next: M1-006 — Ready: Yes
```

### M1-006 — M1 qualification과 M2 handoff

**Outcome**

M1 exit criteria를 clean checkout과 CI에서 재현하고 실제 npm static inspect를 열기 위한 M2 entry item 하나로 handoff한다.

**Read**

- [Step 12 — M1 Domain Workflow Skeleton](./01-milestones.md#5-m1--domain-workflow-skeleton)
- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- M1 scope, architecture/import direction, no Promotion와 no production fake wiring을 audit한다.
- clean checkout에서 bootstrap, quick, docs, minimum Go, macOS와 활성 race gate를 재현한다.
- branch CI와 supply-chain pin을 확인하고 M1 qualification evidence를 기록한다.
- M1이 모두 충족된 뒤 M2 entry decision 하나만 Queue에 연다.

**Acceptance**

- M1 milestone exit criteria와 Global DoD 해당 항목에 재현 가능한 evidence가 있다.
- clean checkout과 required CI가 성공하고 ignored local 문서·비밀값·개인 환경 파일이 tracked되지 않는다.
- M1 범위 밖 placeholder와 fake success path가 없다.
- M2 구현은 아직 시작하지 않고 exact entry decision만 Ready 상태다.

**Completion**

```text
Status: COMPLETE
Commit: e1ba0d5
PR/CI: PR #2; implementation run 31669932193과 final completion run 31670097293의 Quick, Docs, Minimum Go, macOS, Required success
Checks: clean clone canonical quick/docs/platform; race; JSON metaschema; package/import/no production fake/no Promotion audit; git diff --check
Decision/Evidence: M1 Qualification — Domain Workflow Skeleton
Limitations: production ecosystem command/Adapter/Evidence Store/Sandbox/Promotion 없음; race는 local qualification이며 Required CI child 아님
Next: M2-001 — Ready: Yes
```

### M2-001 — npm Static Inspect entry decision

**Outcome**

실제 npm reference를 exact identity와 controlled acquired bytes로 연결하는 첫 production static inspect를 구현하기 전에 최소 Adapter·network·storage·verification·inspection·Evidence·Policy·CLI 계약과 fixture를 결정한다.

**Read**

- [Step 12 — M2 npm Static Inspect](./01-milestones.md#6-m2--npm-static-inspect)
- [Threat Model — Isolation and Inspection](../threat-model/02-isolation-and-inspection.md)
- [MVP — Ecosystems, Platforms, and Artifacts](../mvp-scope/01-ecosystems-platforms-artifacts.md)
- [MVP — Inspection and Verification](../mvp-scope/02-inspection-and-verification.md)
- [Architecture — Adapters and Providers](../architecture/02-adapters-and-providers.md)
- [Domain — Artifact Identity](../domain-model/02-artifact-identity.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)

**Work**

- npm reference와 registry metadata를 공통 Artifact Reference/Resolved Identity에 mapping하는 exact contract를 결정한다.
- registry endpoint, redirect, timeout, response/archive size, public authentication-free 범위와 error taxonomy를 결정한다.
- Controlled Intake handle/root/permission/cleanup과 observed digest ownership을 결정한다.
- bounded tarball/manifest/static check, declared integrity Verification, Evidence Store와 M2 Policy requirement를 결정한다.
- safe/malformed/digest mismatch/archive escape/timeout/unavailable controlled fixture와 result schema evolution을 결정한다.
- 결정 후에만 M2 implementation queue를 추가하며 Dynamic Sandbox와 Promotion placeholder를 만들지 않는다.

**Acceptance**

- mutable npm reference가 acquisition 전 exact version으로 resolve되는 type/API가 명확하다.
- network/content/archive/resource limit과 cleanup failure가 fail-closed operational 결과로 정의된다.
- declared integrity와 acquired bytes의 observed digest가 분리되고 mismatch가 정상 Verification Result로 표현된다.
- Dynamic Inspection 부재가 빈 success나 자동 `ALLOW`로 해석되지 않는다.
- fixture가 public registry 상태, 실제 credential, 개인 filesystem과 mutable reference에 성공 여부를 의존하지 않는다.
- production code와 dependency를 추가하기 전 하나의 실행 가능한 M2 queue가 열린다.

**Completion**

```text
Status: COMPLETE
Commit: M2-001 branch commit
Checks: canonical docs profile; git diff --check; npm public registry metadata/SRI contract 확인
Decision/Evidence: M2 Entry Decision — npm Static Inspect Contract
Limitations: npm Adapter/HTTP/Intake/Evidence/CLI implementation과 external dependency 추가 없음
Next: M2-002 — Ready: Yes
```

### M2-002 — npm reference·metadata resolve foundation

**Outcome**

public npm input을 bounded reference로 parse하고 abbreviated metadata에서 exact resolved identity와 declared tarball/integrity descriptor를 생성한다.

**Read**

- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [Contract — Artifact Port](../domain-model/07-contract-artifact-port.md)
- [Engineering — Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Work**

- `internal/artifact/npm`에 public registry 전용 reference parser와 metadata model을 추가한다.
- explicit injected HTTP client로 metadata endpoint, timeout/body/redirect/host guard를 구현한다.
- name/tag/exact version resolve, response consistency와 sanitized operational error를 contract test한다.
- generic Port/Domain에 필요한 최소 descriptor extension만 추가하고 npm raw struct를 Core에 전파하지 않는다.

**Acceptance**

- scoped/unscoped name, latest/tag/exact version이 one exact `ResolvedArtifactIdentity`로 끝난다.
- range/URL/git/file/auth/custom registry/redirect와 malformed/oversized metadata가 success로 변환되지 않는다.
- application은 npm package를 import하지 않고 production client가 credential/proxy/redirect를 사용하지 않는다.
- format, unit, static analysis와 architecture check가 통과한다.

### M2-003 — Controlled Intake와 declared integrity verification

**Outcome**

tarball을 trusted intake root에 bounded stream으로 확보하고 observed SHA-256과 declared SHA-512 SRI verification을 분리한다.

**Read**

- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [Domain — Artifact Identity](../domain-model/02-artifact-identity.md)
- [Threat Model — Fail-Closed and Promotion](../threat-model/03-fail-closed-and-promotion.md)

**Work**

- run-local intake root, permission, atomic write, containment, cleanup와 content handle을 구현한다.
- tarball host/status/body/timeout guard와 SHA-256/SHA-512 stream calculation을 구현한다.
- missing/malformed/mismatch SRI를 normalized Verification result/finding으로 만들고 I/O failure와 구분한다.
- controlled HTTP fixture로 limit, sync/cleanup, mismatch와 partial result를 test한다.

**Acceptance**

- acquired bytes만 observed digest를 만들고 declared integrity가 content identity를 대체하지 않는다.
- failed/partial content가 valid handle이나 Policy input으로 노출되지 않는다.
- intake와 Evidence/Staging root가 혼합되지 않고 Host/project path가 output에 없다.
- format, unit, static analysis, architecture와 race check가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: bec6483
Checks: canonical quick/docs; go test -race -timeout=5m ./...; git diff --check
Decision/Evidence: run-local intake, atomic tarball streaming, SHA-256 content identity와 SHA-512 declared SRI verification result/finding
Limitations: static tar/manifest inspection, Evidence Store, M2 Policy·CLI wiring 없음
Next: M2-004 — Ready: Yes
```

### M2-004 — npm tarball static inspection과 Evidence Store

**Outcome**

extract 없이 tarball/manifest 위험을 정규화하고 trusted local Evidence reference를 만든다.

**Read**

- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [MVP — Inspection and Verification](../mvp-scope/02-inspection-and-verification.md)
- [Contract — Evidence, Staging, and Promotion](../domain-model/09-contract-evidence-staging-promotion.md)

**Work**

- gzip/tar stream limit, path/type/manifest/lifecycle key check을 구현한다.
- unsafe archive/manifest/integrity finding과 trusted JSON Evidence writer를 구현한다.
- raw artifact/manifest script/Host path가 result or Evidence summary로 새지 않는 contract test를 추가한다.

**Acceptance**

- tarball을 Host filesystem에 extract하거나 lifecycle script를 실행하지 않는다.
- structural violation은 completed static inspection의 blocking finding이며 parser failure와 구분된다.
- Evidence reference가 exact Run/identity/digest를 추적하고 write failure는 operational failure다.
- format, unit, static analysis, architecture와 race check가 통과한다.


### M2-005 — M2 Policy·CLI wiring과 controlled vertical test

**Outcome**

real npm inspect CLI를 M2 Policy와 연결하고 deterministic controlled registry fixtures로 end-to-end 결과를 검증한다.

**Read**

- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)
- [M1 Workflow Contract](../domain-model/10-m1-workflow-contract.md)

**Work**

- `m2-npm-static-inspect` Policy와 dynamic unavailable limitation을 구현한다.
- production bootstrap에서 real npm Adapter/Intake/Evidence writer를 explicit wiring하고 `helox npm inspect` command를 추가한다.
- safe/manual, integrity mismatch/block, unsafe archive/block, timeout/operational failure vertical test를 추가한다.
- result schema compatibility와 exit 10/20/1을 확인한다.

**Acceptance**

- safe static tarball도 `ALLOW`가 아니라 `MANUAL_REVIEW`이며 no Promotion path를 유지한다.
- public registry mutable state를 CI success input으로 사용하지 않는다.
- human/JSON output이 exact npm identity, observed SHA-256, check limitation, evidence references와 decision을 공유한다.
- canonical quick/docs/platform, CLI/process contract와 controlled integration test가 통과한다.

**Completion**

```text
Status: COMPLETE
Commit: 9e5a555, 5f12eb2
Checks: canonical quick; go test -race -timeout=5m ./...; controlled vertical tests; git diff --check
Decision/Evidence: M2 Policy fail-closed decision, npm CLI wiring, safe/manual·integrity/archive block·operational failure fixtures
Limitations: Dynamic Sandbox/install/Staging/Promotion 없음
Next: M2-006 — Ready: Yes
```

### M2-006 — M2 qualification과 M3 handoff

**Outcome**

M2 exit criteria를 clean checkout/CI에서 재현하고 Linux Dynamic Inspect entry decision 하나로 handoff한다.

**Read**

- [Step 12 — M2 npm Static Inspect](./01-milestones.md#6-m2--npm-static-inspect)
- [M2 npm Static Inspect Contract](../domain-model/11-m2-npm-static-contract.md)
- [Engineering — CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- M2 scope, actual network restrictions, intake/Evidence separation, no extraction/execution/Promotion을 audit한다.
- clean checkout에서 all active profiles, race, controlled fixtures와 schema validation을 재현한다.
- branch CI/supply-chain change를 확인하고 M3 entry decision 하나만 Queue에 연다.

**Acceptance**

- M2 milestone exit criteria와 Global DoD 해당 항목에 reproducible evidence가 있다.
- external registry outage/mutable metadata가 required test success를 좌우하지 않는다.
- Dynamic Sandbox, install, Staging와 Promotion이 M2 source/CI placeholder로 생기지 않는다.
- M3 구현은 시작하지 않고 exact entry decision만 Ready 상태다.

**Completion**

```text
Status: COMPLETE
Commit: qualification record after 5f12eb2
Checks: canonical quick/docs/platform; go test -race -timeout=5m ./...; schema metaschema; git diff --check
Decision/Evidence: docs/planning/05-m2-qualification.md; all M2 vertical fixtures use in-memory controlled transport
Limitations: public CI run/PR merge is external; M3 implementation and Sandbox are not started
Next: M3-001 — Ready: Yes
```

## 5. Work Item 완료 기록

work item을 완료할 때 같은 변경에서 다음을 기록한다.

```text
Status: COMPLETE
Commit: <commit SHA>
Checks: <canonical profile 또는 CI run>
Decision/Evidence: <필요한 canonical link>
Limitations: <없음 또는 명시적 제한>
Next: <다음 ID와 Ready 여부>
```

- 상세 구현 일지를 이 문서에 누적하지 않는다. commit과 canonical 문서를 참조한다.
- 완료된 항목은 표에서 삭제하거나 번호를 재사용하지 않는다.
- 다음 항목이 시작 가능하면 `Ready: Yes`로 바꾸고, 실제 시작 전에는 `IN_PROGRESS`로 표시하지 않는다.
- 구현 도중 queue 자체를 바꿨다면 변경 이유와 영향을 해당 commit에서 설명한다.

## 6. Historical state records and current M12 status

아래 M0~M11 항목은 과거 완료·qualification 기록이다. 과거 work item의
`현재 재개 지점` 표현은 현재 실행 상태가 아니며, 현재 실행 상태는 이 절의
M12 canonical block만 소유한다.

```text
M0: COMPLETE
M0-008: COMPLETE
M1: COMPLETE
M1-001: COMPLETE
M1-002: COMPLETE
M1-003: COMPLETE
M1-004: COMPLETE
M1-005: COMPLETE
M1-006: COMPLETE
M2: COMPLETE
M2-001: COMPLETE
M2-002: COMPLETE
M2-003: COMPLETE
M2-004: COMPLETE
M2-005: COMPLETE
M2-006: COMPLETE
M3: COMPLETE
M3-008: COMPLETE
M4: COMPLETE
M5: COMPLETE
M6: COMPLETE
M7: COMPLETE
M8: COMPLETE
M9: COMPLETE
M9-001: COMPLETE
M9-002: COMPLETE
M9-003: COMPLETE
M9-004: COMPLETE
M9-005: COMPLETE
M9-006: COMPLETE
M10: COMPLETE
M10-001: COMPLETE
M10-002: COMPLETE
M10-003: COMPLETE
M10-004: COMPLETE
M10-005: COMPLETE
M10-006: COMPLETE
M10-007: COMPLETE
M11: COMPLETE
M11-001: COMPLETE
M11-002: COMPLETE
M11-003: COMPLETE
M11-004: COMPLETE
M11-005: COMPLETE
M11-FIX-01: COMPLETE
M11-FIX-02: COMPLETE
M11-FIX-03: COMPLETE
M11-FIX-04: COMPLETE
M11-FIX-05: COMPLETE
M12: IN_PROGRESS
M12-001: IN_PROGRESS
M12-002: NOT_STARTED
M12-003: NOT_STARTED
M12-004: NOT_STARTED
M12-005: NOT_STARTED
M12-02: RESERVED
M13: BLOCKED
Active work item: M12-001 — Ready: Yes
Next work item: M12-002 — Ready: No (M12-001 acceptance prerequisite)
```

M9-006까지 qualification을 완료했다. M10-001은 release identity·manifest 및
bootstrap trust를 확정했고 M10-002는 release asset build와 provenance candidate
workflow를 구현했다. M10-003은 HAA-owned runtime 구성이 필요하지 않음을
확정하고 공식 runtime image의 immutable identity, digest와 candidate provenance
binding을 검증했다. M10-004는 verifier-gated installer·atomic activation·doctor
경계를 구현했다. M10-005는 rollback/tamper qualification과 명시적인
fail-closed release gate를 구현했다. M10-006은 Apache-2.0 `LICENSE`, Harmony
copyright-license CLA Option Five, hosted GitHub CLA Assistant status 정책,
individual/entity contribution과 third-party provenance 절차를 기록했다.
CLA governing jurisdiction은 Republic of Korea로 확정되었다. remote PR/CI
qualification과 protected `release` environment 설정이 끝날 때까지 public release는
게시하지 않는다. M10의 verified distribution implementation과 M11 detection depth는
완료되었고, M12 ecosystem expansion 및 M12-02 final red-team/fix gate 이후 M13에서
production activation과 최종 배포를 수행한다.

### M11 work breakdown

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M11-001 | optional seccheck/context field와 detection schema entry decision | COMPLETE |
| 2 | M11-002 | bounded process/filesystem/network observation normalization | COMPLETE |
| 3 | M11-003 | unexpected process·honeytoken·workspace violation Finding | COMPLETE |
| 4 | M11-004 | sensitive payload suppression·Evidence summary·retention bound | COMPLETE |
| 5 | M11-005 | actual gVisor integration·policy regression·M12 handoff qualification | COMPLETE |

### M12 Ecosystem Expansion queue

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M12-001 | Official PyTorch source support | IN_PROGRESS |
| 2 | M12-002 | public Go Modules | NOT_STARTED |
| 3 | M12-003 | Rust/Cargo and public crates.io | NOT_STARTED |
| 4 | M12-004 | Terraform Provider installation | NOT_STARTED |
| 5 | M12-005 | cross-ecosystem qualification and feature freeze | NOT_STARTED |
| 6 | M12-02 | final red-team/fix gate | RESERVED |

### M13 Production Release & Operations queue (M12 완료 후)

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M13-001 | protected main/develop/tag/release environment/immutable release activation | BLOCKED |
| 2 | M13-002 | `develop → main` 실제 protected PR qualification | NOT_STARTED |
| 3 | M13-003 | canonical release build·manifest·attestation·OS package assets | NOT_STARTED |
| 4 | M13-004 | npm·PyPI/pipx·Homebrew convenience publication | NOT_STARTED |
| 5 | M13-005 | exact publication·clean Host bootstrap·doctor·cross-ecosystem smoke·quarantine | NOT_STARTED |

### M12 baseline audit status

```text
M12-001 — PyTorch
IMPLEMENTED: YES
WIRED: YES
QUALIFIED: NO
ACCEPTANCE_CLOSED: NO
Status: IN_PROGRESS
MISSING:
- PyTorch profile-specific bounded resource policy implementation/qualification
- CPU profile 실제 qualification
- 최소 1개 supported CUDA profile qualification
- graph node source identity가 install/Promotion boundary까지 유지되는 evidence
- Linux gVisor qualification
- CI qualification evidence

M12-002 — Go
IMPLEMENTED: YES
WIRED: NO
QUALIFIED: NO
ACCEPTANCE_CLOSED: NO
Status: NOT_STARTED

M12-003 — Cargo
IMPLEMENTED: YES
WIRED: NO
QUALIFIED: NO
ACCEPTANCE_CLOSED: NO
Status: NOT_STARTED

M12-004 — Terraform
IMPLEMENTED: NO
WIRED: NO
QUALIFIED: NO
ACCEPTANCE_CLOSED: NO
Status: NOT_STARTED
```

`WIRED: NO`는 코드가 전혀 존재하지 않는다는 뜻이 아니라, 해당 work item의
required user/runtime path 전체가 acceptance 수준으로 연결되지 않았다는 뜻이다.
M12-002~004의 상세 MISSING은 해당 work item이 시작될 때 별도 baseline audit로
확정하며, 현재는 M12-001 하나만 `IN_PROGRESS`다.

### M11 post-qualification release hardening

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M11-FIX-01 | release activation rollback/durability | COMPLETE |
| 2 | M11-FIX-02 | protected main·exact Required provenance | COMPLETE |
| 3 | M11-FIX-03 | candidate exact 10-file attestation | COMPLETE |
| 4 | M11-FIX-04 | draft asset binding·post-publish quarantine | COMPLETE |
| 5 | M11-FIX-05 | observer profile wait timing regression | COMPLETE |

## Step 13 Invariant

1. Current Work Queue에는 현재 milestone의 실행 항목만 둔다.
2. 한 번에 하나의 work item만 `IN_PROGRESS`다.
3. work item은 canonical 설계를 링크하며 상세 결정을 복제하지 않는다.
4. 선행조건과 acceptance가 없는 작업을 시작하지 않는다.
5. 실제 검증 evidence 없이 `COMPLETE`로 표시하지 않는다.
6. 외부 상태 변경과 repository-wide 보안 설정은 필요한 명시적 승인을 받는다.
7. 미래 capability의 placeholder package, config 또는 CI job을 queue가 요구하지 않는다.
8. 독립 repository에서는 현재 next item부터 Queue 순서대로 실행한다.

## 누락 점검

- [x] 현재 milestone, active item과 next item
- [x] 단일 상태 모델과 Ready 의미
- [x] M0의 순차 work item과 dependency
- [x] 각 항목의 canonical document route
- [x] Outcome, Work와 Acceptance
- [x] 완료 evidence 기록 형식
- [x] blocker와 queue 변경 규칙
- [x] repository-wide 변경 승인 경계
- [x] M1 이후 작업의 지연 생성 원칙
- [x] 독립 repository의 정확한 재개 지점
