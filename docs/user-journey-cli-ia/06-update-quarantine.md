# Update quarantine — SEC-UPD-001

Status: required prepublication work; implementation/qualification incomplete.
This leaf owns the user-requested update-command contract. It supplements the
installation journey, never weakens existing Policy or adds implicit Host execution.

## User-facing scope

Users must be able to request supported ecosystem updates through familiar commands
prefixed by helox, including npm update, pip install --upgrade and cargo update.
The exact supported package selection, constraints, arguments and flags must be
listed before implementation; original tools' entire command surfaces are not
implicitly supported. Go's existing selected-version get path is reused. Terraform
upgrade and GitHub asset replacement need their own explicit supported contracts;
GitHub has no assumed universal native package-update command. All lanes remain
inside existing public-source/artifact/runtime support, not new private registries.

A package originally installed by another tool must not be silently adopted as a
trusted HAA-owned installation. Detect the supported project/venv/receipt state;
provide a bounded explicit enrollment flow or reject it without changing files.
Foreign files, unmanaged paths and ambiguous ownership stay protected. Native
package-manager operations outside helox are not transparently intercepted.

## Required update behavior

An earlier allowed package name/version is not trust for its next release. Resolve
and freeze the proposed dependency graph in the existing controlled environment.
Bind source, exact version/variant and digest for every selected artifact. Inspect
all entries required by the existing complete-set contracts, including changed
transitive dependencies even when the directly requested version does not change.
Do not weaken required checks or substitute version/name/checksum presence for
inspection. Registry integrity can verify publisher-provided bytes without proving
that a compromised publisher's release is benign.

Only after all required checks and Policy allow the exact proposed set may the
existing trusted staging/promotion path replace managed files and dependency
controls. No fresh live re-resolution or original-tool network download after
approval; the promoted bytes must match inspected bytes. Lifecycle/build/import
execution remains in its existing isolated role, never the user's Host merely to
complete an update. Findings and Evidence must be preserved on failure.

Guard the original project/venv/ownership state, concurrent changes, files and
lockfiles. Unsupported input, failed/incomplete inspection, review or block must
leave the user's installed state unchanged. Promotion failure must preserve the
primary cause and provide confirmed rollback or explicit recovery-required state;
never print update success because inspection alone allowed the set.

## Acceptance

Reuse the existing Inspect/Install, complete selected-project inspection, exact-set
Manifest, staging, guarded transactions and ownership/recovery implementations.
Do not build a parallel scanner, global transparent interceptor or generic raw
command executor. Each newly wired ecosystem has independent acceptance evidence.

Prove: benign v1 to permitted v2; same-name/source new harmful-fixture version is
blocked; transitive-only change is inspected; missing/incomplete verification is
rejected; source/digest swap after inspection is rejected; concurrent or foreign
state is not overwritten; injected commit failure restores the original state or
reports explicit recovery-required failure. Use harmless controlled regression
fixtures for negative behavior. Perform actual supported positive CLI consumers,
meaningful negative consumers and affected canonical/runtime checks. Human/JSON
results must agree on policy, update/install status and any rollback uncertainty.
Remove completed private fixture/cache/payloads after evidence is preserved.

Current gaps/reuse evidence and ordered work are in the
[baseline audit](../planning/evidence/prepublication-gaps/baseline-20261018.md).
An earlier bounded install/UX qualification does not close this update acceptance.

## Implementation map and scoped command semantics

This is a reviewed implementation plan, not implemented command support. Execute
one ecosystem at a time after CLI-UX-002; keep other lanes unmodified until their
own baseline/selection contract is verified. Minimum planned forms are:

| Lane | Planned form / behavior | Existing location / method to reuse or extend | Required gap before wiring |
| --- | --- | --- | --- |
| npm | npm update with zero or one package selector, respecting current package.json ranges and lock state; no implicit major-range rewrite | cli/root.go command registration; sandbox/npm_resolver.go NPMResolver.ResolveDependencies; application/install_inspect.go and install_promote.go; promotion/npm_transaction.go freezeNPMProject/verifyManagedOrEmpty/commit/rollback | Capture original controls before resolution; resolve UPDATE from the existing project rather than synthesize a new install-only manifest; preserve constraints and freeze full selected graph |
| PyPI | pip install --upgrade with one named distribution; native specifier normalization and default only-if-needed upgrade selection | cli/root.go AddPyPIInstallSources; sandbox/pypi_resolver.go PyPIResolver.ResolveDependencies; promotion/pypi_promotion.go promoteActiveVenv; pypi_venv_transaction.go prepare/commit/rollback | Model supported current venv inventory/constraints without executing its packages; freeze proposed complete set and validate conflicts with retained distributions. Unimplemented strategies/options reject before mutation |
| Cargo | cargo update with no selector or one -p selector; preserve manifests while updating the selected lock graph | sandbox/cargo_resolver.go project update resolver; artifact/cargo/project.go selected local manifests; application/project_update.go executeProjectUpdate; promotion/cargo_approved_transaction.go and cargo_transaction.go | Separate lock update selection from cargo add; preserve workspace/features/public-source restrictions and exact checksum binding |
| Go | Preserve existing go get module@version; qualify scoped go get -u selection separately | sandbox/gomodule_resolver.go ResolveProjectDependencyUpdate; artifact/gomodule/project.go; application/project_update.go; promotion/go_approved_transaction.go and go_transaction.go | Confirm -u selection/constraints against pinned guest Go; no silent toolchain/self-update, no change to current explicit-version path |
| Terraform | native-shaped terraform init -upgrade for the supported current-project Provider scope | artifact/terraformprovider/project.go/project_resolver.go; application provider workflow; promotion/terraform_approved_transaction.go/terraform_transaction.go | Existing CLI takes an explicit Provider reference. Current-project selection and native single-dash flag parsing are missing, not solved by relabelling that reference. Freeze lock/package/signature and ownership per selected Provider |
| GitHub | existing explicit github install owner/repo@tag#asset may replace a proven HAA-managed installed asset | artifact/githubrelease; application/install_promote.go; promotion/githubrelease_promotion.go | Identify prior target/manifest ownership and guarded replacement/rollback; never overwrite an unrelated user binary or pretend a universal gh update command exists |

The table identifies actual owners, not permission to send arbitrary native
arguments to Host tools. Add only the typed selection/request data demonstrated
necessary by that lane; do not scaffold future interfaces or weaken execution
admission to run a new command. Bootstrap wiring and human/JSON reports must be
included in each lane's change inventory. Check CLI leaf flags/argument parsing
before provider composition and verify unsupported options leave targets unchanged.

### Existing environments and original controls

For existing HAA-managed state, reuse its validated owner ledger, exact control
snapshot, mutation lock, backup and journal. npm verifyManagedOrEmpty already
rejects unmanaged dependency controls. PyPI prepare already supports replacement
of owned distributions, retains unrelated owned output, and rejects foreign or
partial shared-owner replacement. These are implemented responsibilities; audit
and extend only proven gaps. Do not rewrite transaction/rollback infrastructure.

For projects/venvs created by native tools, first design and qualify explicit
read-only inventory/enrollment. Package metadata/RECORD is untrusted input, not
proof of ownership or safety. Bound paths/count/bytes, reject aliases/escapes/
ambiguous owners, snapshot actual files and complete dependency state, then inspect
the proposed set before registering managed state or replacing any files. Existing
user packages/interpreters must not be executed to discover this inventory.
Unsupported environments receive a clear explanation with zero mutation. A clean
new target test does not qualify existing-developer update adoption.

Freeze original controls before untrusted resolution. The resolver's copied input,
inspection graph, staged set, commit controls and original-state check must belong
to one update attempt. Preserve directly requested constraints, retained packages,
extras/features/workspaces and shared ownership. Old evidence is not permission
for changed bytes; no-op updates must prove unchanged selection. Blocked/review/
failed operations must preserve original files and diagnostic evidence.

### Per-lane commit and validation plan

Before source: baseline the exact file/function/test inventory and original
hashes; add the smallest failing command/selection or mutation regression. Then
commit the independently qualified ecosystem change together with its tests and
Queue/evidence update. Use a separate checkpoint only for a meaningful, testable
selection/inventory change; no dead adapters or command-only stubs. Existing
scanner/Policy, install paths and supported runtime identities remain reusable.
If replacement of an implemented responsibility becomes unavoidable, present why
an in-place extension cannot work and obtain the required replacement approval.

Run meaningful focused tests, canonical format/Quick, then security/docs/workflow
as affected and actual bounded CLI update consumers. Required negatives include
harmful fixture v2 under the same name; transitive-only update; semver/constraints
and retained-version behavior; missing checks, review/block; source/digest swap;
foreign or shared-owner collision; concurrent control/file drift; cancellation,
commit fault and rollback fault; no stale success or swallowed primary cause.
Compare before/after controls, payload/ownership hashes and Evidence. Exercise
both zero/one selectors where supported; verify install remains unchanged.

Broaden integration only for actual affected execution/selection/promotion paths.
A stable candidate that changes shared PyPI behavior may require affected existing
CPU/CUDA consumers; a UI-only change does not. Do not rerun full GPU matrices for
every intermediate edit. Every real helper uses the canonical lifecycle and
confirmed cleanup; preserve failures/rollback state, delete completed disposable
payloads/caches after final use. Update lessons only for confirmed findings.

Native semantics references:
[npm update](https://docs.npmjs.com/cli/v8/commands/npm-update/),
[pip upgrade behavior](https://pip.pypa.io/en/stable/development/architecture/upgrade-options/),
[Cargo update](https://doc.rust-lang.org/cargo/commands/cargo-update.html).
Implementation must check the repository-pinned guest tool behavior as well;
current external documentation alone does not qualify that executable.
