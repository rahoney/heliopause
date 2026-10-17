# Go 1.26.9 single-version maintenance

Baseline: milestone/m13-production-release-operations at61a5baf, 2026-10-15.
The user requested retirement of1.26.5/1.26.7 and one maintained1.26.9 SDK.
M13-002 protected-branch qualification and its pending PR33 merge are separate.

## Baseline audit and scope

| Responsibility | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | Actual gap |
| --- | --- | --- | --- | --- | --- |
| Host build and quality tooling | YES | NO | NO | NO | PATH selected1.26.7; temporary1.26.9 absent; downloaded1.26.7/1.26.8 SDKs remained |
| Go guest resolver/build runtime | YES | YES | NO | NO | Lock and generated identity still selected1.26.8; new image and actual Go consumers required |
| Cross-version consistency checks | YES | NO | NO | NO | Bootstrap enforced its compiler; other profiles and host/guest agreement needed wiring |

Existing installation, runtime lock/code generator, checker and Go integration
tests are reused. No Core, Policy, sandbox budgets or observer predicates change.
Historical logs and module-parser fixtures retain their recorded Go versions;
they are not executable SDK installations. Independently pinned third-party
gVisor/Bazel build tooling is not redefined as the HAA product/guest SDK.

## Host activation and retirement

Official Go download metadata and the archive's SHA-256/size agreed before
extraction. The selected SDK is installed permanently at `$HOME/.local/go`.
Login and interactive shell configuration select its `bin`; stable `go` and
`gofmt` command symlinks under `$HOME/.local/bin` also cover existing non-login
PATHs. Persistent `GOTOOLCHAIN=local` disables additional automatic SDK downloads.
Actual default, login and interactive-shell selection is Go1.26.9.

Go1.26.5 was already absent. After checking old-tree identities, process
exe/cwd/fd/maps references and immediately rechecking the deletion inventory,
the exact1.26.7 installation and1.26.7/1.26.8 SDK caches were removed:12 paths,
878,216,358 logical regular-file bytes. No active references or unknown process
queries were present. Ordinary dependency modules and historical evidence remain.
This logical byte count is not measured disk or Windows VHDX space recovery.

## Guest and checker changes

The official linux/amd64 image was resolved, pulled by immutable digest and
independently executed as Go1.26.9:

```text
golang:1.26.9-bookworm@sha256:bcef992b77b1e2031aaaa51da75cebc32da8e9c182e12e633ea79023c79d9eff
```

`scripts/runtimes.lock.json` owns the new identity; the existing generator
updates `internal/runtimeidentity/runtime_lock_gen.go`. The normal read-only,
network-disabled image build smoke passed. It is not a substitute for the
authenticated HAA resolver/build integration suite.

Every checker invocation now rejects a compiler different from the tool lock.
The existing runtime-lock drift step also checks agreement between the repository
go directive, all quality-tool setup versions and the guest SDK. Eight regression
cases cover current agreement, retired SDKs, an old product floor/tool compiler,
partial future upgrade rejection and coordinated future version agreement.
The directive extraction applies to the repository-owned file; normal Go build
and module-drift checks remain the authority for complete go.mod syntax.

## Validation and execution boundary

Canonical bootstrap and final Quick, security, vulnerability, workflow,
release-gate and qualification-freshness passed under Go1.26.9. Quick includes
format/build/test compilation, architecture, vet, pinned Staticcheck and default
tests. Docs and final candidate checks are recorded at closure.

The first OneDrive Quick timed out in architecture package inventory after
format/module/build stages; the exact original failure remains preserved.
A full1500-tracked-file native projection was compared by bytes/modes before
use. A prematurely started check before that copy completed failed without
go.mod and is excluded. The first native candidate's external parser import was
correctly rejected by Architecture; the import was removed, without expanding
the allowlist. Staticcheck's initial ST1005 findings were corrected. Final native
Quick is an actual exit0 after these changes, not a relabelled earlier attempt.

The existing canonical observer build completed all102 latch groups, and the
six gVisor runtime members were independently verified against the unchanged
lock. Actual Go qualification uses those exact dependencies, ordinary-user
authenticated test clients and the existing helper lifecycle. The original
installed runtime was an older, incompatible local bundle; it is not evidence
for the new candidate.

Automatic approval review initially rejected privileged installation replacement
because temporary service disruption or failed restoration was not explicitly
approved at that scope. No command executed from that rejected request. After
candidate binaries/hashes and the11-file backup were reviewable, the user
explicitly approved the local installation change,16 existing Go integration
tests and successful1.26.9 activation. Unexpected failures require analysis;
uncertain termination must not authorize dependent privileged restoration.

Actual result: all16 existing Go integrations passed, including cgo, transitive
build and filesystem/missing-cache/unmanaged refusal. The exact installed
production CLI then completed a managed dependency-free download and build with
`COMPLETED / ALLOW` and an actual published output. The approved1.26.9 installation
remains active; original files are backed up, not restored over the new version.
Final readback confirmed all11 installed hashes, root ownership and protected
modes, Go1.26.9 CLI/helper metadata and the locked amd64 image digest. All three
helper lifecycles were reaped with wait0; final original/helper/client statuses
were0, helper state CONFIRMED_STOPPED, endpoints absent and running containers0.
The first ordinary-user final reader was denied by canonical0600 pod-init;
root readback passed without changing permissions. That reader failure was not
a guest/product failure.

Current local audit: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES /
ACCEPTANCE_CLOSED=YES / MISSING=none within this maintenance scope.
No Git remote write, CI dispatch, release or protected-branch merge is included.
New-source remote qualification, M13-002 acceptance and separate full-Race/release
findings remain open; historical CI is not reassigned to this candidate.

## Evidence custody

Local raw receipts, original failure logs, candidate/installed inventories,
reviewed operation script and final results are kept under
`docs/github-logs/go-single-20261015/`. They are intentionally outside the
versioned candidate; raw local paths and full API/tool responses are not copied
into this report. A hash reference does not imply remote custody of those files.
The [portable result](./result.json) binds the completed scope and original raw
receipt hashes. Final closure docs/security and candidate checks are recorded
separately in [closure checks](./closure-checks.json).

Official sources: [Go download metadata](https://go.dev/dl/?mode=json&include=all),
[official golang image](https://hub.docker.com/_/golang).
