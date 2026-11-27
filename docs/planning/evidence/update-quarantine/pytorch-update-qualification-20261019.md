# SEC-UPD-001 — named PyTorch update qualification / 2026-10-19

Status: COMPLETE for the documented named managed-venv update subset.
IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES /
MISSING=none within that scope. Only
SEC-UPD-001 is active. The [baseline and correction review](./pytorch-update-baseline-20261019.md)
owns the before-source audit; [contract06](../../../user-journey-cli-ia/06-update-quarantine.md)
owns supported behavior. Source306 CPU/cu126/cu130/cu132 actual update/no-op,
unchanged refusals, confirmed restoration/cleanup and independently matched
CI336 are qualified. The chronological checkpoints below preserve the earlier
failures and then-current missing work; the final closure owns current status.
Together with the separately qualified public managed/native subsets, this closes
the reviewed bounded PyPI lane. Broader native named footprints are not qualified.
No candidate is permanently
activated; main merge, publication and CLI-UX-003 remain later work.

## Source and fixed local checks

Commit88f352ddcf011463c8de83e4638dd2ecfa7fee4a connects the existing named
source-pinned traversal, only-if-needed selection, complete final constrained
graph, four immutable controls, CLI, held transaction and named resource context.
The subsequent retirement correction changes only the original-state inventory
and guard in promotion/pypi_update.go and pypi_update_retired.go, with focused
tests. It reuses existing scanners, M5, prerequisite binding and offline Promotion.

The correction's same fixed Go source passes canonical format2.31s,
Quick96.13s, security8.15s, docs0.93s and targeted Race26.59s. Direct nested-tree
RED2.73s becomes GREEN3.56s. The positive and refusal fixtures cover exact
directory membership, inode/mode changes, new files and empty directories,
removal/replacement/symlinks, cumulative count and depth limits. Quick includes
compiler/test build, vet, pinned Staticcheck and default tests. The canonical
profiles precede targeted inspection; no scanner or required check is disabled.

The CLI candidate is27,141,710B, built in2.02s with pinned Go1.26.9/CGO0/trimpath.
Its exact executable identity is recorded as:

| Role | SHA-256 |
| --- | --- |
| Actual retirement-fix CLI | 9e962a6003cfeb10410c1910917796ad29bfb004d984ae5c7a3ff13e8eaff6bf |

Matching source inventories and receipts are local raw
pytorch-retired-tree-{quick,security,docs,race}-source.json and corresponding
receipts; only status documents changed after this build, with fresh checkpoint
docs0.27s/security5.11s. Test source and the candidate's Go inputs are identical.

## Actual CPU version change and direct regression

All actual operations used ordinary UID1000 through the authenticated installed
CLI path and the unchanged canonical helper lifecycle. The target interpreter
or installed packages were not executed on Host.

| Consumer | Source / actual result |
| --- | --- |
| CPU torch2.14.0+cpu seed installation | Source88f352d,535.25s,10 entries/required30/Evidence32, M5 ALLOW and completed promotion |
| CPU2.14.0 to2.14.1 update | Source88f352d,772.64s, same complete counts and ALLOW/promotion; user file and unrelated real constraint retained |
| Original same-version no-op | Source88f352d FAILED before selection in5.15s; missing-METADATA snapshot on nested retired tree |
| Fresh exact -U on the same real failing target | Correction candidate,777.42s,10 entries/required30/Evidence32, M5 ALLOW and completed promotion |
| Stale but syntactically valid prerequisite target | Refused125.11s, target unchanged |
| Nonexistent named version | Refused21.45s, target unchanged |
| Conflicting retained METADATA requirement | Refused88.37s, target unchanged |

The previous version left114 directories, no files and no METADATA under its
obsolete dist-info. The original transaction retains directories after removing
obsolete owned files. The old shallow empty-directory check then treated this
tree as an installed distribution. The new bounded classifier freezes the whole
directory-only tree in the existing guard. It grants no ownership or old safety
approval and still refuses nonempty/linked/changed metadata state. The real
regression preserved all114 directories; none were deleted to conceal the error.

The successful new no-op has ten empty first-phase deltas and ten fresh candidate
reports, followed by the full final traversal and inspection. All32 Evidence
handles are disjoint from the previous update. Each Manifest entry has its three
required verification/static/dynamic Evidence references; the two additional
command attestations are retained. The original input source, selected exact
version, real unchanged requirement and final report digest bind the four
controls. These are fresh safety checks, not reuse of a prior version's ALLOW.

The first seed operator incorrectly assumed exactly three Evidence per entry
and stopped after successful installation. Both the product success and failed
harness result remain recorded separately. The harness now validates per-entry
required Evidence and unique handles. It reused that successful seed rather than
repeating the complete installation.

## Affected public/native PyPI consumers

The same correction candidate passed eleven actual public/native consumers in
239.22s: ordinary24.2 installation; implicit-native refusal; forged user-file
RECORD claim; changed old bytes with a consistent forged RECORD; unsupported
bytecode header; invalid new prerequisite; explicit native24.2 to25.0 adoption;
managed only-if-needed repeat; exact -U using VIRTUAL_ENV; human result; unsupported
eager strategy. The normal adoption took39.45s, managed repeat27.40s, exact
repeat26.64s and human result26.51s. All structured positives completed their
required3/Evidence3, M5 ALLOW and promotion. All refusal targets were unchanged.
Authentic old-wheel/cache proof and user-file preservation remain enforced.

## Restoration, cleanup and remote boundary

Both final CPU and public/native lifecycles returned original0, actual owned
helper wait0/CONFIRMED_STOPPED, observer restoration and client_restore0/RESTORED.
Fresh root postguards verified the four original file identities/hashes/modes,
CLI symlink, runtime directories and process/socket/Docker absence. The installed
CLI is the original language-enabled version after each trial.

Evidence, Manifest/SBOM, outputs and exact operators were preserved. With complete
open-reference and Docker checks, the completed CPU fixture/cache/lifecycle/tools
2,029,581,107 logicalB and public/native trial29,400,444 logicalB were deleted.
Earlier two CPU lifecycle copies53MB were separately deleted after their own
confirmed cleanup. Shared SDK/runtime, protected rollback and active reusable
checker/cache/observer plus the next-profile CLI are retained. Completed cleanup
operators themselves were archived and removed.

Source88f352d CI334/run39162988352 completed Required plus nine prerequisites
SUCCESS. Actual checkoutd27411cfeed51f35052706f242bc13fa886c050a has the same full
tree919f798de07e5940565dea12c96d4dc09b12d032. Native macOS and45 Linux consumers,
CPU full282.88s, five helper wait0 and final original0/CONFIRMED_STOPPED are verified
from the decoded logs. This run covers the earlier wiring source, includes no
actual named update gate, and does not qualify the later retirement correction.

Raw owners: pytorch-wired-cpu-{installed,resume-installed} and
pytorch-retired-tree-cpu-installed for actual CPU attempts; reproduction,
postanalysis and cleanup receipts under pytorch-retired-tree-*;
pytorch-retired-public-* for eleven affected consumers; ci334-completed-qualification.json
and its three decoded job logs. Whole named/PyPI acceptance awaits actual
CPU/CUDA scope completion and new-source remote qualification. Native named
footprints remain outside the independently qualified public-native subset.

Source86c85c5 CI335/run39167405799 completed Required plus nine prerequisites
SUCCESS. Actual checkout9099844ad21af5fc93fe1d68f3a6530f66235f6e shares the exact
full treebd67b859180038b4d65db3b2f667fc599ef01f06. Native macOS,45 actual Linux
consumers, CPUfull242.16s, five helper wait0 and final original0/CONFIRMED_STOPPED
are verified from the decoded logs in ci335-completed-qualification.json. This
qualifies affected ordinary CI consumers for the retirement correction; actual
named CUDA UPDATE cases remain independently open.

## CUDA selection event-limit correction candidate

The source86 actual cu126 seed completed2186.76s,29 entries/all87 required
checks/Evidence93, ALLOW and completed promotion. The first operator stopped
only because it assumed one dynamic Evidence ID; the corrected offline validator
checks normal, not-applicable and supplemented records against actual check IDs,
entry identity and complete unique handles. It rejects a deliberately mismatched
required-check count. Reusing that successful seed reached actual UPDATE, but
failed409.04s before inspection with EVENT_LIMIT charged100000. The original
failure, helperwait0/restoration and fresh root postguard are preserved. Completed
first/resume lifecycle copies26,488,645B each were deleted; completed seed intake/
staging payload7,330,104,300B was deleted while retaining the active target and
Evidence/Manifest/SBOM. No update success is claimed from the seed.

The reviewed in-place correction returns each bounded report from its producing
trusted pip invocation, and reuses only identical same-phase aggregate intent.
The report-reader regression is RED3.34s (six extra launches) to GREEN2.83s;
shared-dependency RED2.92s (six selections) to GREEN2.93s (three, preserving fresh
first/final reports). A changed aggregate still reselects. Source/profile,
constraints/extras, Simple/Core Metadata checks, complete controls and all observer
limits remain. The public update script's bytes are exactly unchanged.

The same final Go inputs pass canonical format2.36s, Quick105.15s,
security6.60s, docs0.26s and relevant Race10.04s. A networkless pinned
Python3.14.7/pip26.2.1 runtime confirms root/dependency empty delta, exact fresh
report, pip failure with a stale report and oversize refusal. Failed commands
return no stdout; successful bytes equal the actual bounded report. Disposable
fixture bytes were deleted after the synchronous containers ended and an empty
Docker query. These are wrapper semantics, not artifact qualification.

New CLI27,150,062B built3.03s, Go1.26.9/CGO0/trimpath, SHA-256
`d42bdc2a4af53ecbf2812dea2233132af6dc92a02d9bc17923fe21b47b867aa6`.
Raw pytorch-report-final-* binds its fixed source/check/build inputs;
pytorch-report-runtime owns exact wrapper scripts/results, and
pytorch-report-public-script-identity.json owns public-script equivalence.
The new actual cu126 trial reuses the successful seed without reinstalling it.
Actual update/no-op/negative results, other affected named profiles, confirmed
restoration/cleanup and this new Go source's remote qualification remain MISSING.
CI335 belongs to source86 and does not qualify this subsequent candidate.


## Source306 actual cu126 closure and remote qualification

The same fixed candidate d42 completed actual cu1262.14.0 to2.14.1 UPDATE in
2755.32s and a fresh same-version native -U in2848.96s. Each has29 entries,
all87 required checks completed,93 Evidence, M5 ALLOW and promotion COMPLETED.
All handles are disjoint from the seed and the other update. Both phases bind
all29 exact candidate identities/digests to the Manifest; the initial update
has28 empty deltas and the no-op29, with a fresh candidate report for every node
and a complete final constrained graph. The two controls contain29 records per
phase; deduplication grants no previous safety approval.

Inspection-only NumPy is excluded from the installed set, supplemented original
environment remains NOT_ATTESTED, and Triton viewer remains explicitly
NOT_ATTESTED/functionality false/later-enforcement false. The synthetic retained
requirement and user file survive normal updates. Missing version63.20s and
retained conflict243.26s both refuse with complete target snapshots unchanged.
These are real ordinary UID1000 installed CLI consumers, not tool-only probes.

The owning lifecycle ends0/5958.90s with helper1441435 actual wait0,
CONFIRMED_STOPPED, original observer/client restored. A fresh root postguard
confirms four originals, symlink and process/socket/Docker absence. Raw Evidence,
Manifest/SBOM, outputs and exact operators remain; completed cu126 target/cache/
lifecycle13,921,488,798 logicalB were deleted after reference checks. Earlier seed
and version-update completed payloads were separately trimmed after validation.

CI336/run39172203301 completed Required plus nine prerequisites SUCCESS.
Actual checkout8431892daf2922eccd5a8480b6becbf2074d0794 has the exact full
tree78f9b5f33301f138a13672f43e1e0b47a51ec3ed of source3060952cefd2554ab5a211d6823e6f10483473a1.
Decoded logs confirm native macOS,45 Linux consumers, CPU full280.93s, five
helper wait0 and final original0/CONFIRMED_STOPPED. This ordinary CI has no actual
named UPDATE gate. Its independent source match does not qualify unexecuted
CPU/cu130/cu132 UPDATE cases. The new same-candidate CPU trial is active; only
SEC-UPD-001 remains IN_PROGRESS and no next ecosystem or UX implementation starts.

Raw owners: pytorch-report-cu126-{installed,postanalysis.json,cleanup.json},
its lifecycle and fresh installed-after receipt; ci336-completed-qualification.json
and three decoded job logs. The preceding source86 EVENT_LIMIT failure remains
preserved separately and is not relabeled as a successful update.

## Acceptance regression ownership cross-check

The final source's Quick covers the following existing owners; this table maps
acceptance rather than claiming new reruns or live malicious registry releases.
Controlled adverse fixtures remain inert test data. Ordinary CI and actual
profile results above qualify their separately stated execution scopes.

| Acceptance | Existing concrete regression owner |
| --- | --- |
| Fresh harmful primary-v2/transitive-only-v2, incomplete inspection and integrity mismatch stop before publication | [Application update tests](../../../../internal/application/install_update_test.go), TestInstallUpdateInspectsNewPrimaryAndTransitiveBytesBeforePolicy; generic npm/public-PyPI cases exercise M3/M5 and all selected acquire/verify/inspect/evidence calls |
| Block/review, original-state session and primary/cleanup cause preservation | Same Application owner: TestInstallUpdateRejectsBeforePublicationAndPreservesPrimaryAndCloseFailure and TestBaseInstallerCannotAcceptAnUpdateWithoutOriginalStateSession |
| Exact named source/runtime/input/constraints/report/digest/graph and early metadata guards | [Named session tests](../../../../internal/promotion/pypi_named_update_test.go), TestPyTorchUpgradeSessionBindsFrozenInputCompleteControlsAndResourceProfile; all four profiles |
| New source/control substitution, missing final candidates, real retained constraints and extras | [Source controls](../../../../internal/artifact/pypi/upgrade_source_controls_test.go), [named selection](../../../../internal/artifact/pypi/upgrade_named_test.go) and [CUDA constraints](../../../../internal/artifact/pypi/upgrade_cuda_test.go) |
| Concurrent original drift, cancellation, staged tampering and partial publication restore originals | [Held update transaction](../../../../internal/promotion/pypi_update_test.go), TestPyPIUpgradeUsesSameEarlyTransactionThroughExactOutputAndRollback |
| Rollback fault is recovery-required; original directories survive rollback | [Original venv transaction](../../../../internal/promotion/pypi_venv_transaction_test.go), TestVenvRollbackPhases, TestVenvRollbackFailureRetainsRecovery and TestVenvRollbackPreservesExistingDirectories |
| Actual command/source and resource/prerequisite forwarding | [CLI](../../../../internal/cli/pypi_update_test.go), [bootstrap](../../../../internal/bootstrap/pypi_update_test.go) and the named producer/session tests; live per-profile evidence remains required |

The current implementation reuses these owners; no alternate Policy, transaction
or old-ALLOW path was introduced. Complete named acceptance still waits for the
remaining same-candidate actual profiles, not another repetition of already
passing unchanged Quick/regressions.


## Source306 actual CPU closure

The same d42 candidate completed a fresh CPU2.14.0 seed599.49s, actual2.14.1
UPDATE877.59s and fresh native -U/no-op885.77s. Each result has10 entries,
all30 required checks completed,32 Evidence, M5 ALLOW and promotion COMPLETED.
The two updates have completely disjoint Evidence from the seed and each other.
Both phases retain all ten fresh exact candidates/Manifest digests; UPDATE has
nine empty first deltas and no-op ten. CPU supplies no inspection prerequisite.
Missing version20.82s and retained conflict90.05s refuse with full target snapshots
unchanged; sanitized resolver boundaries have no reported observation fault.

The owning lifecycle ends0/2493.64s, helper1638732 actual wait0,
CONFIRMED_STOPPED, observer/client restored and fresh root postguard PASS.
Completed CPU target/cache/lifecycle1,202,985,346 logicalB were deleted after
reference/Docker checks. Seed413,289,131B and update413,288,803B completed payloads
were previously trimmed. Original Evidence/Manifest/SBOM/results remain in
pytorch-report-cpu-installed; postanalysis, after, lifecycle and cleanup receipts
own this exact scope. Go source remains306 with its independently matched CI336.

Only cu130/cu132 actual cases remain in the named matrix; cu130 is now active
with the same fixed candidate after fresh original/capacity/input checks. Whole
named/PyPI qualification and SEC-UPD-001 acceptance remain open. No next ecosystem,
interactive launcher, merge or publication was started.


## Source306 actual cu130 closure

The same candidate d42 completed cu1302.14.0 seed1873.62s, actual2.14.1
UPDATE2435.60s and native -U/no-op2608.74s. Each result has29 entries, all87
required checks completed,93 Evidence, M5 ALLOW and promotion COMPLETED. Both
updates have Evidence disjoint from the seed and each other. First/final phases
bind all29 fresh exact candidates/Manifest digests; UPDATE has28 empty first
deltas and no-op29. NumPy remains inspection-only, with original supplemented
environment and Triton viewer NOT_ATTESTED distinctions preserved.

Missing version52.48s and retained conflict250.26s refuse with full target
snapshots unchanged and expected sanitized adapter errors/no reported observation
fault. The owning lifecycle ends0/7266.00s, helper1747275 actual wait0,
CONFIRMED_STOPPED, observer/client restored and fresh root postguard PASS.
Completed target/cache/lifecycle11,570,834,928 logicalB were deleted after
reference/Docker checks; seed6,037,247,510B and update6,037,243,904B completed
payloads were previously trimmed. Raw pytorch-report-cu130-installed,
postanalysis/after/lifecycle/cleanup receipts own this scope.

Go source remains306/CI336. CPU, cu126 and cu130 are qualified for this candidate;
cu132 is now active after original/input/capacity checks. It remains the last
missing named profile, so whole named/PyPI acceptance stays open. Later ecosystem
updates and CLI-UX-003 have not started.

## Source306 actual cu132 and final named closure

The same candidate d42 completed cu1322.14.0 seed1928.85s, actual2.14.1
UPDATE2479.44s and native -U/no-op2592.89s. Each has29 entries, all87 required
checks completed,93 Evidence, M5 ALLOW and promotion COMPLETED. Both updates
have Evidence disjoint from the seed and each other. First/final phases each
bind29 exact candidate identities/digests to the selected Manifest. The first
UPDATE has18 empty deltas, while no-op has29; the seed/update Manifests show
eleven changed version/digest identities: torch plus ten transitive entries.
This is observed selection, not attribution to an upstream release time.
NumPy remains inspection-only; supplemented original environment and Triton
viewer remain explicitly NOT_ATTESTED with their existing distinctions.

Missing version49.59s and retained conflict243.19s both refuse with whole target
snapshots unchanged and the expected sanitized adapter boundary/no reported
observation fault. Owning lifecycle2018932 ends0/7345.87s; helper2018981 actual
wait0, CONFIRMED_STOPPED, observer/client restored and fresh root postguard PASS.
The postanalysis independently validates required Evidence, identities, four
controls, fresh disjoint handles, no-op deltas, auxiliary exclusion and refusal
scope. Completed target/cache/lifecycle11,449,132,037 logicalB were deleted after
reference/Docker checks. Completed seed5,997,264,548B and UPDATE5,999,669,600B
payloads were separately trimmed after phase validation. All four profiles are
now complete, so the six exact-archive-matched temporary operators31,802B were
also deleted after open-reference/script-process/Docker checks. Evidence,
Manifest/SBOM/raw/operator archives, active reusable check source/cache/observer/
candidate, shared SDK/runtime and required protected rollback remain.

| Same-source306 actual profile | UPDATE seconds | Fresh no-op seconds | Required checks / Evidence | Refusal, restoration, cleanup |
| --- | --- | --- | --- | --- |
| CPU | 877.59 | 885.77 | 30 / 32 | PASS |
| cu126 | 2755.32 | 2848.96 | 87 / 93 | PASS |
| cu130 | 2435.60 | 2608.74 | 87 / 93 | PASS |
| cu132 | 2479.44 | 2592.89 | 87 / 93 | PASS |

CI336 independently qualifies the same Go source's ordinary Linux/native macOS
consumers, with its full-tree match recorded above; it has no actual named UPDATE
gate. Existing public/native qualification and affected eleven-consumer retirement
regression remain separately scoped. Source306's final Quick includes those
direct regressions, and its public update script is byte-identical; the named
optimization does not turn historical public/native runs into new live runs.
No additional unchanged Quick/Race/CUDA rerun is implied by this document closure.

Raw owners: pytorch-report-cu132-{installed,postanalysis.json,installed-after.json,
installed-lifecycle.json,installed-lifecycle.log,cleanup.json}, its preserved
seed-update-identity-differences.json and
pytorch-report-completed-operators-cleanup.json. Earlier failures remain failures.
Only the bounded named/PyPI acceptance closes; SEC-UPD-001 stays IN_PROGRESS for
Cargo, Go, Terraform and GitHub. Next is the
[Cargo baseline](./cargo-update-baseline-20261019.md), completing pinned semantics,
edit map and second review before source changes. CLI-UX-003, main merge and
publication remain later work.
