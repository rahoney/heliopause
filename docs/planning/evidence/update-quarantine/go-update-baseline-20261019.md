# SEC-UPD-001 — Go update baseline / 2026-10-19

Status: initial independent responsibility audit complete; pinned-tool semantics,
exact initial selection contract/edit map and second review remain pending.
No Go source or command flag has been changed. Cargo's reviewed lane is closed
for source85ac4c1 and CI338; only SEC-UPD-001 is IN_PROGRESS. Current existing
implementation is that same qualified Go/runtime input, with documentation-only
local follow-up. The local raw go-update-baseline-inventory.json freezes22 read
owners and exact SHA-256 values before implementation.

Read: [update contract](../../../user-journey-cli-ia/06-update-quarantine.md),
[M12 Go contract](../../16-m12-01-ecosystem-expansion-contract.md#4-m12-002--go-modules),
current Queue, artifact L19, sandbox L03/L08/L12/L18, evidence L11/L16 and the
corresponding existing owner functions below. Preserve the fixed public proxy,
independent SumDB proof, bounded intake, M3/M4, trusted cache/Evidence and isolated
observed build; do not transplant the diagnostic proxy into product admission.

## Independent responsibility audit

| Responsibility | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | Existing owner / MISSING |
| --- | --- | --- | --- | --- | --- |
| Explicit go get module@version; one frozen requested primary plus complete project | YES | YES | YES, existing scope | YES, existing scope | sandbox/gomodule_resolver.go ResolveProjectDependencyUpdate; application/gomodule.go Get; current CI338 actual public/transitive get/download/build consumers |
| Independent public acquisition/SumDB/static inspection, fresh project Policy/Evidence/cache | YES | YES | YES, existing scope | YES, existing scope | artifact/gomodule and verification/inspection adapters; ProjectInspectService; GoVerifiedCache; reuse |
| Original control guard, independent retained approval and rollback | YES | YES | YES, existing scope | YES, existing scope | promotion/go_approved_transaction.go Begin/Commit/CommitSnapshot; go_transaction.go; control_transaction_backup.go; reuse in place |
| Scoped native -u selection | NO | NO | NO | NO | Current validGoResolverCommand admits only exact two-token get module@version, download-all and graph. Missing a separate bounded upgrade command and original module membership/selected-version binding |
| Explicit existing native/changed-control enrollment for updates | NO | NO | NO | NO | Current Begin allows missing independent approval only for dependency-free mod plus empty/absent sum; matching marker is not authority. Define per-request explicit enrollment without changing old Begin/build contracts |
| Update-specific unchanged control identity and original permission preservation | NO | NO | NO | NO | Existing transaction stages and replaces both controls0600. New update must retain identical controls and preserve supported changed-control permissions; keep exact original/selected/backup validation and fault recovery |
| CLI/bootstrap complete fresh update reporting | NO | NO | NO | NO | cli/root.go AddGoModuleGet parses an exact version; no -u flag. Reuse go_module.go's public verifier/inspector/store/cache/promoter and shared guarded workflow; add per-request intent and truthful English/Korean human/JSON output |
| New update acceptance | NO | NO | NO | NO | Pinned executable selection/constraint probe; selector/enrollment and whole-set adverse/rollback matrix; actual installed benign update/fresh no-op/unchanged refusals; affected old consumers, restoration/cleanup and new-source Required |

## Findings that constrain the next plan

The existing resolver already captures original control bytes, selects once in a
private controls-only workspace, downloads the whole selected module list, obtains
mod graph, and freezes one requested graph plus the complete snapshot and controls.
Reuse that traversal. It cannot be converted to native -u by replacing an exact
version with latest: native upgrade's transitive selection is a separate behavior
that the pinned executable must establish. Keep all artifact verification fresh.

Unlike Cargo's zero-selector complete update, an initial explicitly named Go
module can retain its real requested-primary role after its exact selected version
is frozen. Inspect this option before adding new Domain representation or relaxing
the existing Go Commit HasRequestedResolution requirement. No arbitrary primary,
new graph edges or new fallback authority is justified.

Proposed minimal first scope for review is one already-required public module
with helox go get -u <module>, plus explicit enrollment where needed. No claim is
made yet for argument-free -u, package patterns, multiple modules, -u=patch, custom
proxy/source/toolchain flags or go/toolchain directive changes. Confirm native
behavior and its interaction with the existing controls-only boundary first;
record the exact supported form and guards in contract06 before product source.
Preserve ordinary explicit-version get, current download/build and strict snapshot
approval. The selected SDK must never silently update the Host/guest toolchain.

Current goProjectPlan/transaction lacks the new update's retained-file semantics;
implement them only in its explicit update mode. Its existing backupRecords already
tracks moved originals and unpublished selected members independently, so reuse
that exact cleanup ownership. Per-request native enrollment must continue validating
any existing independent old cache/Evidence; old markers and go.sum declarations
are not prior approval. It replaces only guarded dependency controls/approval,
not arbitrary old module-cache files or user build outputs.

Next: measure the repository-pinned Go1.26.9 executable using inert diagnostic
modules and no network, freeze selection/retained/transitive/no-op/toolchain facts,
clean all owned containers/fixtures, then finish the edit map and second review.
Such a probe is tool semantics only and never authenticates public module bytes,
qualifies SumDB, observes a product gVisor profile or closes actual CLI acceptance.
No observer, runtime or budget change is currently planned.
