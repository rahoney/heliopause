# SEC-UPD-001 — Go update baseline / 2026-10-19

Status: independent responsibility audit, pinned-tool semantics, exact initial
selection contract/edit map and second review complete before source; implementation
is wired and qualification is pending in the [current report](./go-update-qualification-20261019.md).
The following audit preserves the pre-source findings. Cargo's reviewed lane is closed
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

The reviewed first scope is one already-required public module with
helox go get -u <module>, plus explicit enrollment where needed. Argument-free
-u, package patterns, multiple modules, -u=patch, custom proxy/source/toolchain
flags and go/toolchain directive changes are unsupported. Pinned behavior and
the controls-only boundary are confirmed below; canonical06 owns the exact form
and guards before product source.
Preserve ordinary explicit-version get, current download/build and strict snapshot
approval. The selected SDK must never silently update the Host/guest toolchain.

Current goProjectPlan/transaction lacks the new update's retained-file semantics;
implement them only in its explicit update mode. Its existing backupRecords already
tracks moved originals and unpublished selected members independently, so reuse
that exact cleanup ownership. Per-request native enrollment must continue validating
any existing independent old cache/Evidence; old markers and go.sum declarations
are not prior approval. It replaces only guarded dependency controls/approval,
not arbitrary old module-cache files or user build outputs.

## Pinned executable findings

The thirteen network-disabled operations in local raw go-update-semantics/result.json
use the exact locked Go1.26.9 image, own inert modules and a file proxy. Native
get -u example.test/a moves both a and its imported b from1.0.0 to1.1.0 while
unrelated direct c stays1.0.0. Exact get a@v1.1.0 retains b1.0.0; these forms cannot
be substituted. A second -u preserves both control byte sequences. The selected
download/graph covers all three public-shaped modules. Patch mode selects1.0.1
for a/b and is a distinct, initially unsupported option. An unknown selector is
added by native get, so prior unique require membership is mandatory in HAA.
Argument-free -u fails without project source. GOTOOLCHAIN=local rejects a module
requiring1.27.0, but a compatible1.26.9 dependency can still change the root go
directive. Check non-require directives after selection and refuse that drift.
This diagnostic's GOSUMDB=off is never a product source permission or proof.
Every owned container exited and was removed; the disposable fixture was deleted.
The exact operator and stdout/stderr/control snapshots remain in the raw owner.
No public acquisition, independent SumDB, gVisor or CLI acceptance is inferred.

## Exact in-place edit map

The canonical06 Go subsection owns supported syntax and product semantics. Only
the following currently missing responsibilities will change; existing source
85ac4c1 remains the qualified baseline. No implemented owner is replaced.

| Owner | Minimal change and invariant | Necessary verification |
| --- | --- | --- |
| artifact/gomodule project-control owner, new upgrade.go helpers beside the reused project.go parser, and focused upgrade tests | Validate one exact module path with official module grammar, unique original require membership, and selected same real module. Compare every non-require directive's parsed tokens; reject source/root/go/toolchain/exclude/retract/tool/godebug/ignore drift; allow native require/indirect/MVS changes. Bind selected exact reference from one download record, never latest text or arbitrary primary | missing/duplicate/path@version/pattern/local/toolchain selectors; directive changes; selected missing/duplicate/path identity; proper-case module and indirect membership |
| sandbox/gomodule_isolated_runner.go and command tests | Admit only get -u validated-module-path in addition to existing exact get/download/graph; preserve environment, endpoint/role/runtime/budget gates | smallest direct command-admission RED then GREEN; empty/multiple/options/patch/pattern/URL refusals; old exact get/download/graph retained |
| sandbox/gomodule_resolver.go and upgrade tests | Extract the existing private selection traversal in place. New explicit upgrade entry validates original membership before execution, selects once in the same controls-only workspace, checks non-require directives, obtains selected exact real reference, then existing complete snapshot/primary graph/digest constructor. Existing exact entry retains old request validation | original presence/content race; private-only controls; one selection, complete transitive/unrelated graph, fresh no-op; selected/control/source/digest drift and cleanup failures; old update/download regressions |
| core/ports/ports.go; application/gomodule.go and tests | Add only the now-needed named-upgrade resolver contract. Reuse executeGuardedProjectUpdate and require HasRequestedResolution for this actual named module; return complete verified set for reports | complete entry checks before set verdict; first/second bad integrity/block/review/incomplete, cancellation, stage/commit/close errors preserve original cause and never stale success; no second resolve |
| promotion/go_approved_transaction.go | Add BeginUpgrade(ctx,install,adopt) and private begin mode; old Begin/build/download strict. Explicit native/changed enrollment snapshots controls only; independently validate any old canonical approval/cache/Evidence even if current controls changed. Absent state plus foreign marker/recovery/corrupt/aliased state refuses. Preserve verifyRetainedApproval strict wrapper used by build | native/managed/changed opt-in matrix; corrupt cache/state/Evidence, marker and recovery refusal; no implicit adoption; old get/download/build consumers |
| promotion/go_transaction.go and existing control_transaction_backup.go consumer | Scoped plan.upgrade, single-link/ordinary-mode validation, identical controls retained with inode/mode/mtime; changed controls retain original permission bits and an absent new go.sum is0600. Validate bytes/inode/mode at backup/publication/approval phases; four fault checkpoints, exact moved/published ownership and rollback in place. Reuse shared backupRecords unchanged | direct retention RED; changed/no-op/absent-sum; four phase faults and rollback faults; same-byte foreign inode/mode/link/retained drift and unknown backup refuse; old transaction regressions |
| bootstrap/go_module.go/bootstrap.go | Reuse one existing public verifier/inspector/Evidence/cache/promoter composition. Add immutable per-request upgrade/adopt adapter; keep exact get and build adapter strict | bootstrap/type/build checks; early CLI validation before composition; no ambient proxy/Host execution or mutable global flags |
| cli/root.go/presentation.go/presentation_catalog.go/human_report.go; cargo_update.go factored report and new Go binding/tests | Existing get leaf accepts one -u module plus --adopt only in upgrade mode; default exact form unchanged. Validate before runtime composition. Reuse/factor project report in place with source-specific labels/build explanation, same JSON schema and completed facts; preserve Cargo wording and behavior | direct and preflight-then-execute, unsupported flags and help/EN/KO plus seven-language fallback, complete/failure human/JSON, source/check/Evidence identity, old explicit Go JSON restriction and Cargo report regressions |

No runtime lock, observer, budgets, build/cache intake rules or CI jobs need a
change. The shared backup cleanup helper is an unchanged consumer, not a second
transaction engine. Generic report factoring must keep Cargo byte-level schema
and accurate source-specific human terms. Named Go selection retains the existing
strict primary value; Cargo's snapshot-only value is not used by Go promotion.

## Second review and execution order

Review repeated against actual owners and pinned outputs after drafting: corrected
the temptation to resolve latest instead of native -u; retained source-free private
selection; required prior membership because native get adds unknown modules;
added compatible go-directive drift refusal despite GOTOOLCHAIN=local; retained
the build's strict approval wrapper; scoped file retention/mode changes to upgrade;
included early preflight/JSON/help/bootstrap and source-specific report consumers.
No baseline YES responsibility needs replacement or unapproved broadening.

Next: commit this reviewed plan before source; add direct command and retention
REDs, extend only these gaps, focused checks then canonical format/Quick/security/
docs and affected Race. Qualify actual installed benign version update, fresh
no-op, explicit native/changed controls, transitive/retained behavior and unchanged
refusals using the canonical helper lifecycle. Recheck affected old Go get/download/
build and shared report Cargo consumers; observer/Python/CUDA inputs are unchanged.
Preserve failed raw attempts, verify stop/original restoration/fresh postguards,
delete completed disposable work/cache/operators after retaining evidence, then
ordinary new-source Required before Go acceptance closure. Terraform/GitHub and
the launcher remain deferred. No observer, runtime or budget change is planned.
