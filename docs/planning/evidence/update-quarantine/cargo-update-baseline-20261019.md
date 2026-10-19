# SEC-UPD-001 — Cargo update baseline / 2026-10-19

Status: current pre-source audit after bounded PyPI qualification, confirmed
restoration and completed fixture/operator cleanup. SEC-UPD-001 is the only
active work item. No Cargo implementation, new native
tool probe or qualification has started. Source3060952 is the inspected baseline;
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

## Before-source work still required

After PyPI qualification/restoration/cleanup, confirm exact Cargo1.99.0 behavior
from the repository-pinned runtime: zero-selector updates all compatible lock
entries; one -p selector conservatively updates the requested package and only
dependencies required by it. Verify the pinned executable's accepted grammar,
manifest preservation, retained dependencies and empty graph. Current
[official Cargo update documentation](https://doc.rust-lang.org/cargo/commands/cargo-update.html)
supports the distinction but does not qualify the pinned binary.

Complete the narrow file/function/test edit map and second review in this same
document before source changes. Include typed selection and immutable controls,
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
