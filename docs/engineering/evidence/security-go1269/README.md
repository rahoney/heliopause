# Scheduled Security Go 1.26.9 repair

Local maintenance scope: `SEC-2026-10-12`, based on
`main@bdc2d891ca0edb719be8a28b1a8576decd3d01f2`. QA is not a separate repository.
The original checkout and its pre-existing untracked documents were preserved;
the repair was prepared in an isolated local branch `security/go1.26.9`.
The local validation records below predate the user-authorized push. On follow-up,
`c73f3f5` was pushed and draft [PR30](https://github.com/rahoney/heliopause/pull/30)
opened. CI runs are recorded separately from the immutable local raw logs.

Latest source `e01ba19` is validated: [CI38258121064](https://github.com/rahoney/heliopause/actions/runs/38258121064)
has Required plus all nine prerequisites SUCCESS. See the
[SumDB follow-up](./sumdb-followup.md) and [remote result](./remote-result.json).
The original `result.json` preserves the earlier local snapshot; its blocked
full-Race status is not the current Required status. The host patch and bounded
diagnostic repair are qualified; broader full-Race and guest-runtime follow-ups
remain separate.

## ROOT_CAUSE

[Run 38240142380](https://github.com/rahoney/heliopause/actions/runs/38240142380)
was triggered by `schedule` on October 12, 2026, without a new push. Job
`114780428275` passed bootstrap and history secret scanning, failed vulnerability
analysis, and skipped fuzz. The database is a changing network input even when
the source commit and compiler pin are unchanged.

Pinned `govulncheck@v1.7.0` reported ten reachable standard-library advisories
against Go 1.26.8, all fixed in Go 1.26.9: GO-2026-6617, GO-2026-6613,
GO-2026-6612, GO-2026-6611, GO-2026-6610, GO-2026-6608, GO-2026-6607,
GO-2026-6605, GO-2026-6604, GO-2026-6603. Affected packages were `net/http`,
`net/textproto`, `crypto/tls`, and `os`. The scanner exited 3; the canonical
wrapper and Actions step exited 1. The wrapper's `execution failure` label does
not make these successful scanner findings a CI execution fault. No harness
repair or product-logic regression is established by this incident.

The [official release history](https://go.dev/doc/devel/release#go1.26.9)
confirms the October 8 security release; the
[representative advisory](https://vuln.go.dev/ID/GO-2026-6617.json) was published
on October 8. The original scan also reported one imported-package and 23
required-module vulnerabilities without reachable vulnerable calls.

## CHANGED_FILES

- `.github/workflows/heliopause-{ci,security,release-build,release-publish}.yml`:
  all twelve host setup entries select exactly 1.26.9.
- `go.mod`: product minimum is 1.26.9. This directive specifies a minimum;
  exact CI/setup pins and `GOTOOLCHAIN=local` select the executing compiler.
  No new `toolchain` directive or major upgrade is needed.
- `scripts/tools.lock.json`, `scripts/check/tools.go`,
  `scripts/check/tools_test.go`: setup Go and exact lock validation agree.
  Tool package versions, archive digests and checksum verification are retained.
- `scripts/check/workflow.go`, `scripts/check/workflow_test.go`: four workflow
  policies require the patched pin; the existing negative case rejects 1.26.8.
- `scripts/version-support.lock.json`: refresh only the Go review, retaining the
  supported 1.26 line and accurately recording latest stable 1.27.2.
- `README.md`, `docs/engineering/03-quality-toolchain.md`,
  `docs/planning/02-current-work-queue.md`, this evidence: build guidance,
  host/guest distinction and bounded maintenance status.

Inventory review included tracked and hidden workflows, locks, bootstrap,
generator, runtime-image/release-manifest consumers, tests and documentation.
There is no tracked Dockerfile or independent host archive/digest lock to
update. Remaining 1.26.8 references are the separate runtime lock/generated
identity, corresponding image fixtures, historical parser fixtures/dependency
reviews and raw qualification evidence; the workflow negative case deliberately
uses the rejected old version. No dependency version or `go.sum` change is made.

## TOOLCHAIN_VERSION_AND_INTEGRITY

Official [download metadata](https://go.dev/dl/?mode=json&include=all) and HTTPS
archive `https://go.dev/dl/go1.26.9.linux-amd64.tar.gz` were fetched independently.
Before extraction, the actual archive matched the official byte size 66,935,201
and SHA-256:

```text
42d158b4d8f7b61ac0a830567c940a86098fb7aac52e467a5ebec03ef5cc2f8d
```

The extracted compiler reports `go version go1.26.9 linux/amd64`.
The official `actions/go-versions` manifest also lists 1.26.9 Linux x64 and
Darwin x64 assets, so both configured CI platforms have the exact setup input.
All local profiles use this absolute toolchain through PATH with
`GOTOOLCHAIN=local`, `GOENV=off`, `GOWORK=off` and task-specific external caches.
Canonical bootstrap retains the pinned package/module proxy and SumDB checks.
Fresh Go-built scanner/linter identities are checked separately from their
reported release versions; the checksum-locked Gitleaks archive is unchanged.

M12's guest toolchain is a distinct identity:

```text
golang:1.26.8-bookworm@sha256:abe4f87f354c4f6d7ee3fb11b241c6b6c24a32ca50a2ebcc30493a2168e14048
```

`scripts/runtimes.lock.json`, `internal/runtimeidentity/runtime_lock_gen.go`,
gVisor/Bazel inputs and installed runtime/helper state are unchanged. The old
guest is not represented as security-patched or requalified by this repair.
Changing its compiler/image would require its own official digest/provenance
review and bounded Go resolver/build/observation qualification.

## VALIDATION_RESULTS

| Check under Go 1.26.9 | Result |
| --- | --- |
| Canonical bootstrap, Quick | PASS |
| Format, module tidy/integrity, runtime drift, production/test build, architecture, workflow/actionlint, vet, pinned Staticcheck | PASS within Quick |
| Canonical vulnerability, gosec, current-tree and reachable-history Gitleaks | PASS |
| Four bounded fuzz targets, docs, strict freshness | PASS |
| Public Go acquisition and verified-cache integration | PASS: two actual tests, no skip |
| Cold module preparation to offline tidy integration | PASS: empty inner cache, warm outer compiler/module cache |
| Darwin amd64 TEST cross compile; Linux arm64 build | PASS; no native macOS execution claimed |
| Fresh CLI build and help; Go-built tool identities; old Go rejection | PASS |
| Full `go test -race -count=1 -timeout=15m ./...` | **FAIL**: seven test harness races |

[`result.json`](./result.json) binds commands, exit codes, toolchain, changed
inputs and raw logs. Tests not explicitly selected as integration are not
represented as executed runtime/E2E qualification.

The Race failure is a separate blocker. Five admission tests read
counters while their test server callbacks update them without synchronization;
two observer diagnostic tests assign a shared writer/buffer after starting the
receiver without synchronization. The original `main@bdc2d89` source and Go
1.26.8 reproduce six of the seven failures with `-race -count=1`; the four implicated
production/test files are byte-identical to this candidate. The seventh test,
`TestSharedObserverAttributesOneContainerAndClosesItsStream`, passed in that
single baseline run, so its failure under the old compiler is not established.
The baseline directly establishes a pre-existing local Race failure without
establishing identical detection across compiler versions. The original new
toolchain race trace is [`raw/race.log`](./raw/race.log); the portable baseline
comparison is in `result.json`, with its unedited original old-toolchain log
retained locally. No product fix or test suppression is justified merely by the
fact that these races were observed after the compiler update. Fixture repair
is kept as a separate follow-up within the existing owners.

The first sandboxed bootstrap stopped on a write restriction in Go's SumDB
cache, before installing tools. It was retried under approved execution
permissions with the same canonical command. This local permission failure is
separate from the original remote scanner findings. That retry shares the
partially populated task cache; it is not claimed as a pristine cold run.

The first sandboxed Quick run also failed observer directory protection checks
and the optional real-runsc capability test. Under approved host execution the
directory tests passed; the remaining installed runsc was 109,368,428 bytes and
did not match the lock or provide `syscall/socketpair_result`. This is the
previously documented local input mismatch, not evidence of a Go patch
regression. The preserved six-member bundle was copied to a task directory and
verified with `scripts/verify-gvisor-runtime-bundle.sh` against the unchanged
lock. The explicit `HELOX_RUNSC_BINARY` test input is its verified runsc,
109,388,472 bytes, matching the canonical SHA-512. No installed binary was
rebound or replaced. The verifier consumes exactly six runtime members; the
auxiliary bundle manifest remains separate from that input inventory.

Direct impact is the compiler/runtime/standard library used by the HAA CLI,
quality tools and release utilities. Existing local HTTP/TLS, archive/path,
filesystem, trust-boundary and transaction tests cover these consumers. Bounded
public Go acquisition/cache and cold bootstrap integrations exercise real
HTTP/TLS/SumDB and offline module preparation under the new compiler. Guest
images, helper code, syscall schema, policy/budgets and M12 feature contracts
are unchanged. No evidence currently requires another full CPU/CUDA campaign.
Old M12 results remain historical results for their original toolchain/inputs.

## REMAINING_VULNERABILITIES

Fresh `govulncheck -show verbose ./...` exited 0: **zero reachable
vulnerabilities, zero imported-package vulnerabilities, and 21 required-module
advisories without imported vulnerable packages/calls**. The exact advisory
IDs, affected modules and fixed versions are retained in
[`raw/vulnerability-detail.log`](./raw/vulnerability-detail.log) and
[`result.json`](./result.json). No suppression or dependency upgrade is made.
Reassess these advisories when importing the affected packages or updating the
module graph; their presence is not described as fixed or harmless.

The separate Go 1.26.8 guest image remains a runtime patch follow-up; a clean
host source scan does not scan that container filesystem or certify all runtime
components. It must not be confused with the locally remediated host/CLI build
toolchain.

## CI_RECHECK_NEEDED

The actual source candidate `e01ba19` passed the complete normal Required
matrix, including native macOS and Linux integration, in CI38258121064.
Scheduled Security38249522013 also passed history secrets, vulnerability and
bounded fuzz on the host patch. The current nine prerequisites contain no Race
job; `platform` and `quick` do not run `-race`.

Only the documentation closure HEAD needs its own fresh Required result after
push. Its executable, toolchain and runtime inputs remain identical to
`e01ba19`; no source qualification is relabeled for a different compiler.
Main remains unmerged and still selects Go1.26.8; rerunning that old main would
not consume this branch's patch. No merge was made.

## PUSH_READINESS

**READY for the bounded repair; source already pushed and Required SUCCESS.**
The documentation closure is validated separately before its authorized push,
then its final HEAD CI is checked. The original broader full-Race fixture
findings remain open, with targeted verifier Race passing on both Go versions.
Guest runtime qualification, reviewer merge and production release approval
remain separate. The historical SumDB failure's lost underlying cause is not
claimed solved by the diagnostic-only change; the affected real consumer and
the complete Required matrix now pass without weakening checks.
