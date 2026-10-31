# SEC-UPD-001 — bounded npm qualification / 2026-10-18

Status: QUALIFIED / ACCEPTANCE_CLOSED=YES for the scoped npm lane; complete
SEC-UPD-001 acceptance remains open. Qualified source29c58f0f52388e7d4a36e613fbdce50df9a60a77,
implementation base8e3e773. The other five
ecosystem lanes are not qualified by these results. Public publication and the
deferred interactive launcher have not started.

## Behavior and reused owners

npm update accepts zero or one direct package name, existing current project or
absolute --target, and explicit --adopt for the supported native subset. Preserve
the original public package.json constraints and copied v3 lock, resolve once in
the pinned isolated npm runtime, inspect the entire selected supported graph,
then promote only its exact inspected bytes. An earlier ALLOW is never reused as
new-version authority. Unsupported/multi-root/legacy local-file inputs fail closed.
The existing single-direct-root plus complete supported transitive graph remains
the scope; original tools' full argument/project surface is not claimed.

Existing InstallService entry inspection/Policy/Manifest/staging and npm project
guard/backup/commit/rollback are extended in place. Public selection controls are
bound into Manifest identity; generated local-tarball controls remain private
offline recipes and are replaced only after exact output validation. Early
original controls/payload and guard identity survive through selection and commit.
Backup members use the early inventory rather than a fresh post-inspection state.
Explicit adoption rejects foreign package paths/files, ambiguous HAA metadata and
unsupported aliases/size/graph semantics before mutation. Required failures,
review/block, concurrent changes, substituted selection/artifacts and cancellation
before publication never authorize replacement. Rollback uncertainty preserves
recovery originals and blocks another mutation rather than reporting success.

The observer reuses exact provenance/phase/launcher/interpreter admission for the
historical fixed install tuple and one new fixed update tuple. No generic Node,
filesystem, process or network allowance was added. Application consumes typed
update-session ports; its per-request resolver/promotion copies preserve ordinary
install bindings. CLI human/JSON explicitly report UPDATE, actual operation and
promotion status, with English/Korean explanations and existing language fallback.

## Actual evidence and limits

Raw owner: docs/github-logs/update-quarantine-20261018/ (local original evidence,
not an assertion that untracked captures are committed or available on GitHub).
Every check records exact source inventory, log digest, exit and duration.

| Evidence | Actual result / scope |
| --- | --- |
| npm-admitted-quick | PASS171.35s; canonical format/build/test-build/architecture/workflow/vet/pinned Staticcheck/default tests |
| npm-admitted-security / docs / workflow | PASS11.04s /0.47s /1.19s |
| npm-admitted-race | PASS21.93s; affected update/selection/transaction tests in application, promotion, sandbox, CLI, evidence and npm adapter |
| npm-observer-build | PASS326.90s; canonical pinned gVisor/Bazel source preparation/build and all102 observer latch labels; fixed install and update production-path positive/negative matrices both executed |
| npm-admitted-candidate | Go1.26.9/CGO0 CLI SHA256 e64b849c9e971028036899adc3812036e35ed941a013da621de90269810841b2 |
| npm-observer-build | observer SHA256 dbc79f6a08b52c88be3c17cbf04e31c39131a92617bec05dcb5e06f90c7e7a49 |
| npm-admitted-installed/summary.json | PASS seven actual installed-path CLI operations, UID1000; copied native project upgraded is-number6.0.0 to7.0.0 within unchanged >=6.0.0 <8.0.0 constraint; public controls and README.user preserved |
| same actual CLI operations | Ordinary install; implicit adoption rejection; explicit named update/adoption; zero-selector current-directory managed reinspection; managed human update; foreign-file and legacy-local-control rejection with unchanged target inventories |
| npm-admitted-installed/evidence and manifests | Original inspection Evidence and exact Manifest/SBOM copies preserved before removing disposable package/cache payloads; successful no-op update has fresh Evidence references |
| npm-admitted-installed-lifecycle / installed-after | Original0, helper wait0/CONFIRMED_STOPPED, observer restored, canonical client restore0/RESTORED; fresh root process/socket/Docker queries and all four original protected hashes/owners/modes PASS |
| npm-admitted-cleanup | Completed npm fixture/cache/lifecycle/client payloads53,235,006 logical bytes deleted; active reusable check source/cache, pending whole-source observer and necessary root rollback retained |
| CI330/run39138729054 | Required and all nine prerequisites SUCCESS; native macOS included; actual checkout d35577db1678b9e3f1f064eb86b6d2ad053b4ea3 has the exact same whole tree as29c58f0 |
| ci330-completed-qualification.json / original Linux and macOS logs | Linux45 top-level consumers PASS, CPU full278.20s, five helper wait0 and final original0/CONFIRMED_STOPPED/helper_stop0/client_error0/client_restore0/NOT_REQUIRED; original/default consumers retained |

Controlled component regressions additionally verify same-name primary v2 and
transitive-only v2 are independently acquired/verified/inspected/Evidence-recorded
and blocked on findings; integrity mismatch and incomplete required inspection
never reach Manifest/staging/publication. Actual filesystem tests cover native
adoption then managed replacement, original drift during resolution/offline work,
staged-byte swap, selected manifest/lock/digest substitution, replaced guard,
bounded/foreign/legacy input, commit-interval drift, partial publication rollback,
and rollback collision retaining original recovery state and primary cause. These
controlled regressions are not public-registry malicious-package experiments.

## Preserved failed attempts

baseline-quick failed in the tool sandbox because Unix socket creation was denied;
same-input actual-host baseline passed. The staged-swap fixture initially failed
while trying to write its deliberately sealed file; only that disposable fixture
was made writable to reach product digest re-verification. Architecture Quick
identified the new legitimate Promotion-to-Port dependency before its exact
allowlist was aligned with the existing architectural direction. Pre-cancelled
selection's resource creation/cause loss has separate red and green logs.

The first actual candidate f9ebff2 failed before package inspection because its
extra ordinary-install --save=false differed from observer exact admission. It is
not a successful update test. Restore the historical install command and add the
fixed update tuple; the final same-source actual seven-operation consumer above
then passed. Original failed raw logs/operators and confirmed original restore
remain preserved. No Node/proc allowance was used as a workaround.

Fresh remote Required/native macOS/ordinary Linux integration are complete for
29c58f0 in [CI330](https://github.com/rahoney/heliopause/actions/runs/39138729054).
Later complete SEC-UPD qualification remains pending. Historical language CI329
was not used as evidence for this source. PyPI data checkpoint268725e and later
source have their own checks and are not qualified by this historical CI.
Original installed CLI/observer were restored after this
local test; this report is not permanent local activation or release publication.
