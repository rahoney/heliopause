# SEC-UPD-001 — Cargo update baseline / 2026-10-19

Status: pre-source audit, pinned-tool selection review and second review complete;
implementation and local automated checks are complete; actual product
qualification remains NOT_STARTED. SEC-UPD-001 is the
only active work item. PyPI qualification, restoration and completed fixture/tool
cleanup are confirmed. The independent pinned-tool probe below is complete;
it is not product qualification. Source3060952 is the inspected Go baseline;
the thirteen-file read-only inventory is local raw
cargo-update-read-ahead-inventory.json under update-quarantine-20261018.
The [update contract](../../../user-journey-cli-ia/06-update-quarantine.md)
owns requirements and order. This audit does not reopen qualified Cargo add/build.

## Independent responsibility audit

| Responsibility | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | Existing owner / MISSING |
| --- | --- | --- | --- | --- | --- |
| Exact Cargo add selection and complete project inspection | YES | YES | YES, historical scope | YES, historical scope | sandbox/cargo_project_resolver.go ResolveProjectDependencyUpdate; application/cargo.go; project_update.go; reuse |
| Complete project snapshot without arbitrary primary | YES | YES | YES, historical scope | YES, historical scope | domain/project_snapshot.go; artifact/cargo/project.go BuildProjectSnapshot; existing complete-project/build consumers; reuse |
| Original source/control guard, approved cache and transaction | YES | YES | YES, historical scope | YES, historical scope | promotion/cargo_approved_transaction.go Begin/VerifyUnchanged/commitSelection; cargo_transaction.go; reuse |
| Lock-only update selection, zero/one package selector | NO | NO | NO | NO | Existing update resolver always invokes add crate@=version; missing a distinct lock update preserving every selected local manifest |
| Fixed isolated native update command | NO | NO | NO | NO | cargo_isolated_runner.go validCargoResolverCommand permits exact metadata and add only; native update tuple needs bounded admission and consumer verification |
| CLI/bootstrap update and explicit native enrollment | NO | NO | NO | NO | cli/root.go registers cargo add/build; bootstrap/cargo.go wires those two services; missing update request, original-state enrollment intent and truthful result |
| New update acceptance | NO | NO | NO | NO | Actual zero/one-selector version change, no-op with fresh evidence, adverse fixtures, rollback and affected add/build checks remain required |

## Findings that constrain the edit map

Existing ResolveProjectDependencyUpdate copies the anchored source to a private
workspace, compares the original guard's controls, invokes one exact add, freezes
the selected source/controls, then obtains locked metadata and builds a complete
snapshot. Reuse this traversal and failure/cleanup discipline in place. A lock
update must preserve Cargo.toml and all selected workspace/path manifests as well
as source bytes. The existing add comparison intentionally permits root manifest
changes; passing true to that comparison alone is insufficient for update.

LockedDependencyGraph requires one requested primary. ProjectDependencySnapshot
already represents a complete project without inventing one. The current
NewProjectDependencyUpdate constructor additionally requires a nonempty requested
resolution. Zero-selector update therefore has a representation gap; choosing an
arbitrary crate as primary is not an acceptable shortcut. Any additive extension
must keep the existing requested-primary constructor strict and bind original
controls, selected controls and the exact complete snapshot. Do not weaken graph,
identity, source or complete-coverage invariants.

CommitSnapshot is deliberately current-state approval: it rejects selected
control digests different from the original guard. resolveProjectSnapshot also
enforces that restriction. Do not relax either check to turn a download/current
approval into an update. Reuse the existing guarded commitSelection through an
explicit bound original-to-selected update. Its cache/Evidence approval, local
manifest binding, transaction, rollback and retained approval remain unchanged.

CargoProjectPromotion.Begin already treats unmanaged source as data and grants
no publication authority. It refuses a marker lacking independent retained
approval. For the new native-update consumer, require either proven current HAA
state or explicit bounded enrollment intent before selection; source metadata,
Cargo.lock checksums and matching project markers confer no prior safety trust.
Preserve the historical add/build entry points and native-source restrictions.

## Before-source selection review

After PyPI qualification/restoration/cleanup, the probe confirmed exact Cargo1.99.0 behavior
from the repository-pinned runtime: zero-selector updates all compatible lock
entries; one -p selector conservatively updates the requested package and only
dependencies required by it. Verify the pinned executable's accepted grammar,
manifest preservation, retained dependencies and empty graph. Current
[official Cargo update documentation](https://doc.rust-lang.org/cargo/commands/cargo-update.html)
supports the distinction but does not qualify the pinned binary.

The following edit map and second review precede source changes. They include typed selection and immutable controls,
early guard/enrollment, exact isolated command admission, existing Application
inspection/staging/commit reuse, CLI parsing/bootstrap and human/JSON reporting.
The result must preserve real policy/evidence and distinguish inspection approval
from completed publication; errors and rollback uncertainty cannot print success.

Required negatives include changed manifests/source, unsupported selectors or
flags, alternate registry/git/outside paths, old approval substitution, harmful
primary/transitive new bytes, incomplete checks/review/block, source/digest swap,
concurrent original changes, cancellation and commit/rollback faults. Compare
whole selected graph and original controls, not just requested-crate counts.
No observer or resource budget increase is authorized by this audit. Run the
canonical profiles first, followed by affected actual Cargo update and existing
add/build consumers under the canonical helper lifecycle. Preserve exact evidence
and delete completed disposable fixtures/cache/tools after confirmed restoration.

## Pinned executable result and exact initial command scope

The immutable Rust1.99.0 image from runtimes.lock.json ran twelve operations using
network=none/runc and an inert synthetic local registry: version/help, initial
lock, complete update, one -p update, name@locked-version selection, nonexistent
selector refusal, exact no-op, complete metadata, and empty lock/update/metadata.
All PASS. One selected package upgraded its required shared dependency while
the unrelated package remained locked. A version in -p identifies the original
locked package; it does not mean --precise for the proposed version. All update
cases preserved manifest bytes; no-op preserved exact lock bytes. The missing
selector returned native101 and kept the original lock. Docker terminal states,
owned container removals and absence queries were confirmed. The fixture and
completed temporary operator were deleted; exact operators/results/locks/metadata
remain in local raw cargo-update-semantics and operators-cargo-update.

This establishes pinned tool semantics only. Synthetic source replacement is
never admitted into the product and does not authenticate public crates, prove
gVisor observation or close actual CLI acceptance.

The first product-boundary RED is cargo-command-admission-red3.28s: existing exact
admission rejects update, update -p name and update -p name@locked-version. Existing
metadata/add and unsupported forms retain their expected outcomes. This is the
documented new-command gap, not a regression in historical add/build support.
The first in-place changes add complete-project update data/selector binding and
private selection while preserving old primary-only consumers. Application/CLI
wiring, native enrollment, lock-only publication and final qualification remain
MISSING; no complete Cargo support is claimed from these changes.

Exact initial forms, selector/enrollment meaning, supported originals and rejected
flags are owned by [contract06's Cargo scope](../../../user-journey-cli-ia/06-update-quarantine.md#cargo-lock-update--reviewed-initial-semantics).
The observed pinned behavior above supports that scope; it is not permission for
native Cargo's broader command/flag surface or synthetic registry sources.

Updating controls does not execute crate builds. Existing verification/static
inspection applies to every selected registry crate. Build scripts/macros remain
in the existing separate helox cargo build isolated observation contract. No
universal malware detection or interception of native commands outside helox is
claimed.

## Narrow edit map before source

| Owner | Exact in-place extension | Meaningful verification |
| --- | --- | --- |
| core/domain/project_update.go | Add an explicitly complete-snapshot constructor and HasRequestedResolution to the existing immutable update value. Share original/selected control validation; keep the old requested-primary constructor strict. Never invent a primary or relax LockedDependencyGraph | Original constructor still rejects missing/mismatched resolution; new variant binds all controls/context/source/graph and explicit dependency-free snapshots; bytes remain immutable |
| core/ports/ports.go, application/project_update.go, application/cargo.go and existing Go/Terraform promotion Commit consumers | Preserve the requested-resolution requirement for the existing get/add/init entry points. Factor the existing guard→selection→complete inspection→staging→commit→close workflow for the new complete-project callback; retain errors.Is/primary cause and zero successful result after close failure | Existing pipeline order/fault matrix plus a new no-primary substitution refusal for old consumers; no new Go -u or Terraform upgrade functionality |
| core/ports/ports.go and application/cargo.go | Add only the now-needed lock-selection port and Cargo update use case, consuming one frozen complete update without ArtifactReference fabrication. Reuse ProjectInspectService, M3/M4 and ProjectCacheStaging | Harmful new/transitive bytes, incomplete integrity/inspection, review/block, no-op fresh runs, cancellation, stage/commit/close failures and no publication on any failed boundary |
| artifact/cargo/metadata.go and locked_metadata.go, small update helper | Reuse crate grammar and parseLock for bounded optional name@locked-version and unique public-lock membership; no URL/registry or local package selectors | Exact/bare/ambiguous/missing/local/invalid selectors; malformed/untrusted-source lock rejects before SDK |
| sandbox/cargo_project_resolver.go | Reuse anchored capture/privateWorkspace, original-control comparison, private environment, locked metadata and BuildProjectSnapshot. Add ResolveProjectLockUpdate and narrowly factor shared selection. Update permits only Cargo.lock bytes to change; every local manifest and source file remains exact | Frozen original mismatch, manifest/local source drift, unexpected added files, selected lock/source mutation, cleanup/cancellation, complete workspace/features/empty graph |
| sandbox/cargo_isolated_runner.go | Admit only update or update -p validated-selector in addition to existing exact commands. Use the existing fixed manifest path, immutable tool image, isolated network/profile, source transport and bounded output. Require original lock and identical selected manifest | Fixed argv/caller provenance, unsupported flags/paths, original data-only input, selected manifest/source refusal and existing add/metadata tests; actual gVisor update later |
| promotion/cargo_approved_transaction.go | Extend Begin internally for explicit lock-update enrollment. Existing Begin remains strict. Even with --adopt validate canonical project-bound old independent approval/cache/Evidence and retain its original snapshot; only its equality to deliberately changed current controls/local manifests may be superseded by a newly fully inspected set. Missing independent approval plus a foreign HAA marker, corrupt state/cache or recovery journal still refuses. Guard actual current source/controls and old retained state through publication | Managed/native/mixed changed-control intent; implicit adoption refusal; forged marker/corrupt old state/cache; concurrent source/state drift; rollback returns actual pre-update user controls and old approval, not historical pre-user-edit inputs |
| promotion/cargo_approved_transaction.go and cargo_transaction.go | Feed complete-snapshot updates into existing commitSelection with explicit lock-only intent. Preserve Cargo.toml/local manifests byte/inode/mode; keep an unchanged lock in place on no-op. Changed Cargo.lock preserves its supported original permission mode. Extend existing backup/publish/rollback bookkeeping to skip retained controls while still validating their exact original identity and selected digest. CommitSnapshot keeps its current-state-only restriction | Manifest mutation refusal; unchanged inode/mode; changed lock exact inspected bytes; selected tampering; concurrent retained-control replacement; every commit fault and confirmed rollback/recovery uncertainty; old add/build behavior unchanged |
| bootstrap/cargo.go and bootstrap.go | Reuse the already constructed public client/verifier/inspector/Evidence/cache/promoter; supply per-request explicit adoption through a small adapter, without mutable global service rebinding | New command wired to real use case, separate requests cannot share adoption/selector state, old add/build wiring retained |
| cli/root.go, presentation.go, cargo update presenter and presentation_catalog.go; reuse human_report.go | Register no-positional update with a bounded StringArray -p/--package to reject repeated occurrences and explicit --adopt. Use actual complete inspection facts for human/JSON checks, policy and Evidence. Success requires completed publication and close; failed/uncertain rollback never prints success. Preserve existing operation-result schemas; a project-scoped result may express complete-project context without a fake artifact primary. Localize English/Korean help/report explanations through existing presentation | Zero/one selector, repeated/unsupported args, target/cwd, user intent, EN/KO help/human, JSON no ANSI, actual checks/Evidence, policy refusal and uncertain commit failure agree on failure; no speculative rollback-success inference |

The retained-control review also inspected `promotion/control_transaction_backup.go`
(`54bde4146b4e899cacfc21307c3fef1e9ad14a600aa05fdd5338db73b28f5f98`).
Its existing backupRecords tracks moved originals and unpublished selected files
independently, so omitted backups and retained staging files already have exact
cleanup coverage. Reuse it without changing shared cleanup. Direct retention RED
4.78s confirms manifest replacement, permission change and no-op lock replacement;
the existing Go/Terraform no-primary refusals pass in that same input.

No scanner, approval cache, source protocol, transaction or observer is replaced.
An observer/runtime change is not planned. If the exact new command exposes an
admission gap, first preserve actual failure and confirm the exact predicate and
affected consumers; do not widen roles, filesystem, budgets or generic argv.

## Second review and acceptance sequence

The review found two additional owners absent from the first read-ahead: Terraform
also consumes ProjectDependencyUpdate, and the existing transaction publishes both
root controls even when bytes are equal. Preserve old Go/Terraform primary-only
consumers explicitly; add the narrow lock-only retained-control handling rather
than loosening CommitSnapshot or rewriting the transaction. These are missing new
update semantics, not newly alleged defects in qualified add/build behavior.

Explicit native/changed-control enrollment replaces only guarded controls and
controller approval, never arbitrary old crate files or build output. Unlike PyPI
native wheel replacement, it does not infer obsolete file ownership from RECORD.
Actual original files, retained old approval and new independent public acquisition/
verification/inspection are separate facts. All selected entries, including
unchanged and transitive-only entries, receive fresh inspection; no-op has no old
ALLOW shortcut. Existing source/workspace/features, checksum/index authentication,
cache/Evidence integrity and bounded graph/source limits stay intact.

Start with a direct RED test of the existing fixed command admission, then
implement the smallest typed selection/guard extensions and focused regressions.
Complete Application/bootstrap/CLI wiring in this same lane, keeping only this
work item active. Use meaningful checkpoints for independently testable changes;
do not commit command-only stubs or change later-lane selection.

Run canonical format and Quick (build/test validity, vet, pinned Staticcheck,
default tests), security/docs and relevant race checks on the final source. Actual
UID1000 installed CLI cases must cover native range-constrained v1→v2 enrollment,
managed no-op and -p, explicit changed-control enrollment, implicit/foreign/corrupt
refusals with whole-project snapshots, and human/JSON. Exercise affected existing
Cargo add/build and Go/Terraform direct consumers for the shared workflow/value;
do not rerun unrelated CUDA matrices for Cargo-only changes. Use the unchanged
canonical helper lifecycle and exact original restore/postguards before cleanup.
Preserve original failures, receipts and Evidence; delete completed private data,
containers and temporary operators. Fresh remote Required/native macOS scope is
independent of local and networkless tool-semantic success.

The lock-only retention GREEN6.15s passed its direct regression and affected old
transactions. Enrollment/fault GREEN12.61s additionally passes explicit native/
changed-input enrollment, current approval, corruption/cache/Evidence/recovery
refusals, all four commit fault phases, same-byte retained identity substitution
and existing Cargo build-input consumers. A prior compile failure exposed the
build consumer of verifyRetainedApproval; its strict existing signature now stays
intact and only the explicit update path calls the changed-input variant. This
is focused synthetic qualification, not actual public CLI acceptance.

Before CLI source changes, composition review confirms PreparePresentation parses
flags before the final Cobra Execute. The new StringArray package selector must
be reset after preflight validation so a legitimate single occurrence is not
appended twice during execution. Keep duplicate rejection before runtime setup;
verify both direct execution and preflight→execution. Allow explicit JSON only
for cargo update; preserve existing other Cargo/Go/Terraform JSON restrictions.
Reuse human_report renderChecks/reasons and add exact Cargo check descriptions.

The implemented slice and final local checks now belong to the linked
[qualification](./cargo-update-qualification-20261019.md). The historical NO rows
above remain the pre-source baseline, not a new claim that wired code is missing.
