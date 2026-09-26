# M7 Qualification Contract — MVP Completion

## Purpose

M7 proves the MVP as a whole; it does not add an ecosystem, broaden an
artifact capability, or weaken an existing contract. npm remains the reference
end-to-end path. PyPI/pip and GitHub Releases prove that the same Core,
Policy, Evidence, Staging and Promotion meanings work through independent
adapters.

M7 can declare MVP completion only when every required evidence item below is
recorded as `COMPLETED` with reproducible evidence. `NOT_PERFORMED`,
`UNSUPPORTED`, skipped CI, unavailable runtime, missing fixture, missing
observer event, cleanup uncertainty and an unverified platform never count as
an `ALLOW` or as qualification success.

## Fixed qualification decisions

| Topic | Decision |
| --- | --- |
| Ecosystem scope | npm full flow; PyPI wheel and sdist flow; exact GitHub Release ELF/ZIP/tar.gz standalone flow are the MVP subjects. New sources, formats and capabilities are out of scope. |
| Common-boundary proof | The evidence must show that adapter-specific values stay outside Core/Application contracts and that all three flows retain identity/digest, Finding/Evidence, Policy and result semantics. |
| Policy/status proof | Representative fixtures must assert `ALLOW`, `MANUAL_REVIEW`, `BLOCK` and `COMPLETED`, `FAILED`, `PAUSED`, `NOT_PERFORMED` where applicable. A non-completed required check cannot reach automatic Promotion. |
| Platform proof | Linux and macOS require native CLI build/default-operation evidence. Windows is qualified only through an actual disposable WSL2 Linux environment; ordinary Linux CI is not WSL2 evidence. macOS/WSL2 do not imply a native macOS/Windows dynamic-inspection backend. |
| Dynamic-inspection proof | Only the pinned Linux Docker/runsc/observer path may claim dynamic evidence. Unsupported platform/format/runtime states must retain an explicit limitation and non-ALLOW outcome. |
| Fixture trust | Fixtures are synthetic, small, immutable in-repository inputs or deterministically generated local inputs. They contain no real secret, private Host asset or mutable public success dependency. |
| CI/security proof | Required CI and the scheduled vulnerability, secret and bounded-fuzz workflow must execute real checks. A placeholder, silent skip, advisory/network failure or unpinned tool is not a green gate. |
| Completion authority | M7-006 is the only item permitted to state MVP completion. Before then, documentation must not claim safe general use or official WSL2 support. |

## Qualification evidence matrix

| Evidence group | Required proof | Owner item |
| --- | --- | --- |
| Ecosystem reference flow | npm resolve → acquire → verify → static/dynamic inspect → Policy → Evidence/Manifest/SBOM → offline Promotion → result, with exact bytes rechecked at every trust transition | M7-002 |
| Adapter expansion flow | PyPI wheel/sdist and GitHub Release asset use the same neutral Core contracts without ecosystem-specific Core imports or re-resolution during Promotion | M7-002 |
| Fixture outcomes | safe, suspicious, malicious, corrupt, unavailable, timeout and resource-exhaustion fixtures have deterministic expected decision/status, no target publish on non-ALLOW, and no real network dependency for a successful fixture | M7-002 |
| Supported Host paths | Linux and macOS native `helox` build/default E2E; disposable WSL2 CLI build/default E2E with environment identity and command output retained | M7-003 |
| Inspection capability matrix | Linux dynamic capability, macOS CLI limitation, WSL2/Linux backend relationship, unsupported artifact/format behavior and no-ALLOW rule are published and exercised | M7-003 |
| Result and recovery | machine/human result, schema, Evidence, SBOM and Manifest references resolve to the exact inspected/published bytes; interrupted run, retention and failed cleanup preserve failure semantics and never expose partial output | M7-004 |
| Product security gates | pinned gosec, Gitleaks and govulncheck; bounded scheduled fuzz; required/scheduled workflow topology, least privilege and failure propagation are checked in CI | M7-005 |
| Final release decision | all prior evidence is current, no known Critical/High product security defect or unresolved trust-boundary violation remains, and installation/troubleshooting/limitation/security-reporting documentation is complete | M7-006 |

## Work breakdown

### M7-001 — MVP qualification entry·evidence matrix

**Read**

- [M7 milestone scope and exit criteria](./01-milestones.md)
- [MVP ecosystems and platforms](../mvp-scope/01-ecosystems-platforms-artifacts.md)
- [MVP inspection and verification](../mvp-scope/02-inspection-and-verification.md)
- [MVP results, Policy and completion](../mvp-scope/03-results-policy-and-completion.md)
- [CI and Quality Gate](../engineering/04-ci-quality-gate.md)

**Acceptance**

- the fixed decisions and evidence matrix make every M7 exit criterion owned by
  exactly one subsequent work item;
- platform, dynamic-inspection, fixture and CI limitations cannot be presented
  as successful evidence;
- the queue contains one ordered M7 work item at a time and this entry adds no
  product code, adapter, runtime, dependency, CLI command or CI job.

### M7-002 — ecosystem flow·fixture regression

Implement the synthetic fixture matrix and cross-ecosystem regression tests.
Exercise the complete npm reference flow plus the PyPI/pip and GitHub Release
flows, including non-ALLOW/no-Promotion assertions and Core boundary checks.

#### Prefixture implementation boundary

M7-002 adds no product interface, adapter capability, runtime or test-only
production abstraction. Qualification composes the existing Application,
adapter, Policy, Evidence, Staging and Promotion paths. The initial
qualification test belongs in `internal/application/m7_qualification_test.go`.
Its fixture builders stay test-local unless two existing test packages require
the identical byte-level builder; only then may a narrow test-only helper be
placed under `internal/testutil/`.

Existing focused tests remain their owning capability evidence and must not be
rewritten: `internal/application/m2_vertical_test.go` proves controlled npm
inspection outcomes, `install_inspect_test.go` proves complete locked-set
inspection, the three `internal/promotion/*_promotion_test.go` files prove
source-specific staged publication, and `internal/sandbox/linux_integration_test.go`
proves the pinned Linux runtime paths. M7 adds cross-flow assertions around
them rather than a second implementation of those flows.

| Scenario class | Required normalized outcome | Promotion assertion |
| --- | --- | --- |
| safe supported subject | `ALLOW`; completed required checks; `COMPLETED` operation | only the supported source path may publish its exact staged digest to a new target |
| suspicious observation or unsupported required capability | `MANUAL_REVIEW`; completed operation with the explicit limitation/reason | `NOT_PERFORMED`; no Manifest/Staging/target publication |
| malicious or unsafe content | `BLOCK`; completed operation with the normalized finding/reason | `NOT_PERFORMED`; no Manifest/Staging/target publication |
| corrupt declared/observed integrity | `BLOCK`; completed verification result | `NOT_PERFORMED`; no Manifest/Staging/target publication |
| unavailable resolve/acquire/Evidence provider | `FAILED` operation with sanitized operational error and no Policy decision | no Promotion invocation or target publication |
| timeout, observer loss or resource exhaustion | required check is non-completed; normalized `MANUAL_REVIEW` where an inspection result exists | `NOT_PERFORMED`; no partial target or success result |

The Luna implementation must create deterministic synthetic npm tarball, wheel,
sdist and GitHub Release ELF/ZIP/tar.gz inputs only as needed by this table.
It must not use a real registry, public Release asset, actual secret or Host
project as a successful fixture. After each fixture group, run focused tests
first, then canonical `quick`, race tests for changed packages and
`git diff --check`; do not expand a fixture or retry a failed scenario without
first preserving the failing normalized result.

#### Terra prefixture record

```text
Status: COMPLETE
Completed: test-local cross-ecosystem fixture binding, non-ALLOW Promotion stop, and acquisition-failure regression
Checks: go test ./...; go test -race ./internal/application ./internal/policy; go run ./scripts/check quick; git diff --check
Evidence: internal/application/m7_qualification_test.go; npm/pypi/github-release exact identity and SHA-256 binding; no-policy/no-Promotion failure path
Limitations: format-specific static fixtures remain owned by each adapter test; Linux runtime and Host-platform qualification remain M7-003
Next: M7-003 — Linux·macOS·WSL2 CLI qualification — NOT_STARTED / Ready: Yes
```

### M7-003 — Linux·macOS·WSL2 CLI qualification

Collect actual Linux and macOS native CLI evidence and run the documented
default CLI E2E in an approved disposable WSL2 environment. Record the exact
Host/runtime capability matrix; no CI Linux substitute is accepted for WSL2.

The native CLI E2E builds the repository's `./cmd/helox` entrypoint for the
current Host and executes its no-argument and `--help` default paths. It is an
offline startup proof only: it neither acquires an Artifact nor claims that a
dynamic-inspection backend is available. Linux and macOS evidence is collected
by their native CI runners. WSL2 remains a separate, approved disposable
Windows/WSL2 execution record.

| Host path | Required command evidence | Dynamic-inspection meaning | M7-003 evidence state |
| --- | --- | --- | --- |
| Linux native | build `./cmd/helox`; run no argument and `--help` on a Linux runner | only the separately pinned Docker/runsc/observer integration may claim dynamic capability | CI required |
| macOS native | build `./cmd/helox`; run no argument and `--help` on a macOS runner | no native macOS backend; capability absence is not an `ALLOW` | CI required |
| Windows through WSL2 | same Linux command path in a disposable WSL2 distribution, with environment identity and output retained | uses the Linux backend contract only; Windows-native dynamic inspection is out of scope | approved environment required |

The same test source is deliberately executed on every native platform. It
must not introduce a Host fallback, provision a runtime, obtain a package or
convert a missing dynamic backend into an `ALLOW` result.

For the approved disposable WSL2 Host, run
`scripts/qualify-wsl2-cli.sh EVIDENCE_FILE` from the repository root. The
script refuses a non-WSL2 kernel and an existing evidence target, creates its
output with owner-only permissions, records only OS/WSL/Go identity and the
two CLI outputs, and removes its temporary binary. The resulting evidence file
is an external qualification record: it is never a repository input, a secret
store or a substitute for the Linux dynamic-integration evidence.

#### Native evidence record

```text
Status: COMPLETE
Linux: GitHub Actions run 32443146082, Quick and Minimum Go platform profiles completed successfully for b469a041a89e62f2501e37001cce3352f0a7cc46
macOS: GitHub Actions run 32443146082, macOS platform profile completed successfully for b469a041a89e62f2501e37001cce3352f0a7cc46
Required gate: GitHub Actions run 32443146082 completed successfully, including the pinned Linux gVisor integration and observer-helper jobs
Local macOS: go test ./...; go run ./scripts/check quick; go test -race ./cmd/helox — passed
WSL2: COMPLETED in a disposable Windows 11 Enterprise Evaluation Hyper-V VM → Ubuntu 24.04.4 WSL2, kernel 6.18.33.2-microsoft-standard-WSL2, WSL 2.7.12.0, Go 1.26.5 linux/amd64; scripts/qualify-wsl2-cli.sh recorded successful native build, no-argument and --help output
Source: a3e5ced2d2dd4254e97470a5a385ea03ee86160a
Limitations: WSL2 qualifies the Linux CLI Host path only; no Windows-native or macOS-native dynamic backend is claimed
```

### M7-004 — evidence·result·resilience qualification

Verify result-schema and Evidence/SBOM/Manifest referential integrity, digest
binding to staged/published bytes, interruption/retry/retention behavior,
cleanup failure semantics and no-secret fixture/output rules.

**Read**

- [Evidence·Staging·Promotion Contract](../domain-model/09-contract-evidence-staging-promotion.md)
- [Evidence, Staging and Promotion Architecture](../architecture/04-evidence-staging-promotion.md)
- [MVP results, Policy and completion](../mvp-scope/03-results-policy-and-completion.md)
- [Coding and Security Rules](../engineering/02-coding-security-rules.md)

**Qualification boundary**

M7-004 validates and, only where a demonstrated failure violates the existing
contract, hardens existing local Evidence/Result/Staging/Promotion behavior.
It adds no source, Artifact format, retention daemon, background garbage
collector, product configuration, CLI command, remote store or synthetic
success dependency.

| Subject | Required proof | Failure meaning |
| --- | --- | --- |
| normalized Evidence batch | every returned reference resolves to exactly one committed record for its Run; a rejected batch publishes no partial Run directory | failed operation; no Policy input or `ALLOW` from incomplete Evidence |
| result presentation | human and JSON outputs expose the same normalized operation/identity/digest/check/Policy semantics; raw operational causes and fixture secrets never appear | failed presentation/operation is non-success and sanitized |
| Manifest/SBOM/Staging | Manifest identity rehashes, SBOM binds the Manifest, stored documents equal the sealed bytes, and every staged Artifact rehashes to its Manifest entry | no `StagedSet` or Promotion from mismatch |
| interruption/retry/cleanup | incomplete temporary output is removed before returning failure; cleanup uncertainty remains an error; successfully committed Evidence/Staging stays retained for audit/retry and a retry never overwrites it | no partial target/result and no silent success |

The qualification uses only existing deterministic local fixtures. It records
only normalized IDs, digests, bounded summaries and sanitized error codes; no
real Artifact, credential, Host file or raw observer payload is admitted.

**Completion record**

```text
Status: COMPLETE
Checks: go test ./...; go test -race -timeout=5m ./...; canonical quick/docs; git diff --check; GitHub Actions run 32476129928 Required aggregate successful
Evidence: private temporary Evidence batch commit with duplicate preflight; retained committed retry behavior; sanitized equivalent human/JSON failure result; exact staged/published Manifest, SBOM and artifact digest rehashing
Next: M7-005 — IN_PROGRESS / Ready: Yes
```

### M7-005 — security workflow·scheduled gate qualification

Activate only the required pinned security tools and scheduled workflow after
their current upstream identity, compatibility and supply-chain review. Prove
real vulnerability, secret and bounded-fuzz execution and include every active
required child in the aggregate gate.

**M7-005 implementation decision**

- Product `go.mod` remains independent of all quality-tool module graphs. The
  repository-owned bootstrap installs `gosec v2.28.0` and
  `govulncheck v1.7.0` with Go `1.26.7`, checksum database verification and
  exact package versions. Gitleaks `v8.18.4` is fetched only as a pinned,
  SHA-256-verified official release asset outside the source tree.
- Gitleaks `v8.30.1` is the current upstream release but has a public report
  that its default rules can return a clean result for a canonical GitHub PAT.
  The newer `v8.28.0` also failed the M7 synthetic credential probe, so neither
  is acceptable for this fail-closed gate. `v8.18.4` is the documented
  known-good control and passed the probe; every future M7 security review must
  re-evaluate the upstream fix before moving this exceptional pin.
- `heliopause-ci.yml` activates independent required `Security` and
  `Vulnerability` jobs. Its sole `Required` aggregate receives every active
  child result and treats failure, cancellation, skip or missing result as
  failure.
- `heliopause-security.yml` is a separate least-privilege weekly/manual
  workflow. It checks out full reachable history, runs history secret scan,
  fresh govulncheck data and the existing parser/archive fuzz targets with a
  fixed 5-second per-target fuzzing budget. Each process has a separate
  one-minute bounded deadline to cover cold compilation and teardown. It never
  receives write permission or repository secrets, and its failure is triaged
  rather than retroactively changing an older merge result.

**Completion record**

```text
Status: COMPLETE
Checks: Go 1.26.7 local fuzz/quick/docs; GitHub Actions run 32555127293 Required aggregate successful; default-branch Scheduled Security run 32556503406 successful
Evidence: pinned gosec/Gitleaks/govulncheck bootstrap and identity validation; required fail-closed Security/Vulnerability aggregation; full-history secret scan; refreshed vulnerability analysis; bounded parser/archive fuzz
Limitations: Gitleaks v8.18.4 is an explicit compatibility exception pending upstream rule repair; scheduled findings require triage but do not rewrite historical merge status
Next: M7-006 — IN_PROGRESS / Ready: Yes
```

### M7-006 — MVP final audit·documentation·completion

Perform the final trust-boundary audit, run the complete qualification suite,
publish supported paths and limitations, and record MVP completion only if all
M7 evidence is current and green.

**Completion record**

```text
Status: COMPLETE
Checks: canonical quick/docs/security/vulnerability; go test ./...; go test -race -timeout=15m ./...; full-rule gosec review; GitHub Actions run 32557123229 all required children and aggregate successful
Evidence: bounded resolver container/firewall cleanup; fail-closed resource close handling; public build/command/support/runtime/troubleshooting guidance; private vulnerability reporting policy; current Linux/macOS/WSL2/gVisor/scheduled-security evidence
Security audit: G115 guarded numeric conversions, G122/G304 trusted-root filesystem operations, G204 fixed infrastructure commands and G302 owner-only modes were manually reviewed after the canonical scanner gate; no known Critical/High product defect or unresolved trust-boundary violation remains
Limitations: automatic full flow is pinned Linux amd64 only; macOS/WSL2 are CLI-only; no native Windows/macOS dynamic backend, general runtime installer, authenticated private source, retention daemon or absolute-safety guarantee
Next: MVP qualification complete; no post-MVP item is active
```

## Completed external prerequisite

M7-003 used an approved disposable Windows 11 Enterprise Evaluation VM with
Ubuntu 24.04.4 WSL2 to collect the required Linux CLI evidence. This qualifies
only the WSL2 CLI build/default path; it does not claim a Windows-native or
WSL2 dynamic-inspection backend.

## M7-002 completion record

```text
Status: COMPLETE
Checks: canonical docs; git diff --check
Decision/Evidence: internal/application/m7_qualification_test.go; explicit six-item M7 queue; fail-closed evidence matrix
Limitations: disposable WSL2 evidence and scheduled security workflow are not yet active and are not claimed as qualification success
Next: M7-003 — Linux·macOS·WSL2 CLI qualification — NOT_STARTED / Ready: Yes
```
