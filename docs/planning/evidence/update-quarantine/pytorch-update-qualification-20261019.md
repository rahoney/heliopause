# SEC-UPD-001 — named PyTorch update qualification / 2026-10-19

Status: IN_PROGRESS. Named source IMPLEMENTED=YES / WIRED=YES /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO for the complete named lane. Only
SEC-UPD-001 is active. The [baseline and correction review](./pytorch-update-baseline-20261019.md)
owns the before-source audit; [contract06](../../../user-journey-cli-ia/06-update-quarantine.md)
owns supported behavior. Actual CUDA updates remain MISSING; the retirement correction's
same-source CI335 remote qualification is complete. No candidate is permanently
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
