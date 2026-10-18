# Update quarantine — SEC-UPD-001

Status: IN_PROGRESS; user authorized implementation after reviewed planning.
The bounded npm lane is qualified for29c58f0 including fresh CI330 ten SUCCESS.
Current lane: [PyPI input/metadata implementation after baseline and selection review](../planning/evidence/update-quarantine/pypi-baseline-20261019.md).
Public resolver/session/CLI and actual managed-venv consumers are locally qualified
in the [bounded PyPI record](../planning/evidence/update-quarantine/pypi-managed-qualification-20261019.md).
Native enrollment, named-source and new remote qualification remain missing. Other lanes wait for
their independent baseline/qualification. See the [npm evidence](../planning/evidence/update-quarantine/npm-qualification-20261018.md).
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

This is a reviewed implementation plan. The bounded npm implementation and local
qualification are recorded above; the remaining rows are planned support. Execute
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
[npm update](https://docs.npmjs.com/cli/v10/commands/npm-update/),
[pip upgrade behavior](https://pip.pypa.io/en/stable/development/architecture/upgrade-options/),
[Cargo update](https://doc.rust-lang.org/cargo/commands/cargo-update.html).
Implementation must check the repository-pinned guest tool behavior as well;
current external documentation alone does not qualify that executable.

### npm retained public controls / implementation baseline

The implementation audit at97ed00c found that generatedNPMProject writes local
file: tarball references for exact offline installation. Those references cannot
serve as public update constraints. Preserve the resolver's public package.json
and package-lock.json as bounded opaque control records, bind their digests/bytes
into trusted Manifest identity, and use local artifact recipes only in private
installation workspaces. Publish selected public controls only after exact output
validation and before the existing guarded atomic commit. The public lock digest
must match the one used for resolution. Legacy local-file controls must never be
silently treated as registry constraints; explicit migration must preserve user
intent, otherwise reject unchanged. The existing one-primary npm graph subset
remains the first supported lane; multi-root support must use complete project
snapshots, never fabricated primary edges. See
[implementation baseline](../planning/evidence/update-quarantine/baseline-20261018.md).

### npm first implementation slice — exact owners

- Extend DependencyResolution with optional bounded ProjectControlFile values
  (existing immutable control type), preserving old constructors/default behavior.
  The npm isolated resolver supplies public selection manifest/lock bytes; the
  evidence Generator includes these opaque controls in the hashed Manifest.
  Promotion validates their digests, lock digest and selected graph binding.
- Keep generatedNPMProject as the private local-tarball installation recipe.
  After offline output/digest validation, restore authenticated public selection
  controls in the private workspace before the existing atomic publish. Extend
  current install as needed so newly managed targets retain their constraints.
- Add an Application update wrapper around the existing InstallService pipeline.
  A typed Infrastructure update session captures and guards original controls/
  payload before selection, supplies one resolver and the existing Promotion
  boundary, and closes the same guard even on reject/cancel/failure. Do not build
  a second inspection/Policy pipeline. Per-request copies avoid shared mutable
  service rebinding. Update results explicitly identify UPDATE; old INSTALL
  results retain their existing fields/enums/exit behavior.
- npm update accepts zero or one validated package name, current project or an
  absolute --target, and explicit --adopt only for the qualified foreign-state
  subset. Isolated pinned npm update preserves root ranges and uses copied
  controls, ignore-scripts/no-audit/no-fund and fixed public registry/config paths.
  Never forward arbitrary flags or execute project scripts/interpreters on Host.
- Extend npmProjectPlan with optional early member snapshots used by updates,
  so all existing verifyUnchanged calls guard the originally captured payload
  rather than a fresh post-inspection baseline. Reuse the lock, backup, atomic
  commit, metadata and rollback implementation. Foreign/native adoption requires
  explicit intent, bounded regular-file inventory and no ambiguous prior ledger;
  malformed controls, local/git/workspace sources and unsupported roots reject
  without publication. Missing node_modules may be installed only from the
  inspected complete selected set.
- Legacy HAA local-file projects cannot recover lost original range intent.
  Reject an implicit migration; a newly inspected explicit install may establish
  retained public constraints. Never reinterpret a local path as a registry range.

The implementation slice must pass actual positive selection/replacement plus
blocked/incomplete/tampered/concurrent/foreign/cancel/rollback negatives before its
acceptance is closed. The remaining ecosystem lanes and broader supported project
coverage remain MISSING until their own actual evidence is recorded.

Current npm control/inventory bounds are4MiB per manifest/lock,32 levels of
JSON nesting,200,000 values, one public direct root,128-character public range/tag,
64MiB per original file,100,000 inventory entries and1GiB per transaction member.
Registry selection has a2-minute context with existing confirmed resolver cleanup.
The fixed npm update invocation must also be admitted by the existing observer's
exact boundary/launcher/interpreter tuples. Preserve the historical install tuple;
qualify the added update tuple with pinned observer build/latches and actual CLI.
Command changes cannot be validated by Go fake runners alone and never justify
broader Node filesystem/network authority.
Original package metadata must match the lock; unlisted packages, files outside
locked package directories, symlinks/hardlinks and foreign HAA namespace content
are not adopted. Missing package output can only come from the complete inspected
new set. UPDATE is an explicit additional operation value in the existing result
schema; INSTALL defaults remain unchanged. Stable controller-selected update
rejection codes distinguish adoption, ownership, unsupported originals and
concurrent changes with English/Korean explanations.

### PyPI selection review before implementation

The [PyPI baseline](../planning/evidence/update-quarantine/pypi-baseline-20261019.md)
separates existing owned-venv transactions from the missing upgrade selection.
Start with public PyPI; audit the distinct named PyTorch resolver separately and
never qualify its CPU/CUDA path using public-PyPI evidence. The scoped native
form is pip install --upgrade/-U with one project, supported exact/numeric public
specifier, active Python3.14 venv or absolute --target, default only-if-needed,
and explicit --adopt for supported native files. Eager strategy, arbitrary indexes,
extras/editable/direct URLs and unsupported metadata reject before composition.

1. Begin the existing venv transaction before selection. Freeze pyvenv.cfg,
   metadata/RECORD, ownership and affected original file identities as bounded
   data. Never run the target Python or import its packages. Explicit enrollment
   cannot turn local metadata/checksums into prior ALLOW or overwrite foreign
   paths; default install retains its current collision rules.
2. Supply only validated inert installed METADATA to the pinned isolated pip
   resolver and select with --upgrade --upgrade-strategy only-if-needed. Its
   dry-run delta can omit the root or be empty; it is not a complete graph and
   cannot authorize inspection completion/promotion.
3. Merge changed versions with the frozen inventory. Freshly resolve only the
   requested root closure before inspection, constrained by all proposed exact
   versions and the active requirements of unchanged installed distributions.
   Superseded old-version requirements must not constrain the changed packages.
   Do not request unrelated distributions as additional roots or re-resolve them
   from a different source. Retained constraints must prevent newly introduced
   conflicts with their actual metadata. Compare changed candidates across both
   selections and cross-check final candidates against canonical public metadata.
   Reuse the existing final single-primary report parser and complete graph.
4. Bind frozen input and final selection into the update's Manifest; inspect
   every final selected entry, including unchanged-version/transitive-only/no-op
   selections. Preserve the existing source-sdist/build-input/derived-wheel
   extension and binding: npm's strict no-derivation graph comparison cannot be
   reused to reject legitimate authenticated derived-wheel nodes. Reuse the same
   held transaction through existing private offline output validation and commit,
   rechecking originals/cancellation before publication. Do not acquire a second
   guard or re-resolve after ALLOW. Existing rollback/recovery semantics remain.

Pinned pip26.2.1/Python3.14.7 controlled offline probes verify normal upgrade,
retained-version behavior, retained-metadata conflict rejection, empty no-op delta,
transitive-only delta and missing installed root. These are tool-semantics proofs,
not production gVisor/inspection/enrollment qualification. New parser, filesystem,
actual supported CLI positive/negative, observer/lifecycle and affected canonical
consumers remain required before the PyPI lane closes.

The current input parser bounds each installed METADATA to the existing1MiB
wheel metadata limit, at most the existing1,024 report projects, and a4MiB opaque
input control including base64 metadata. Native selection supports bare names,
canonical exact versions (including existing name@version compatibility), and
the existing bounded numeric requirement grammar; unimplemented specifiers reject.
Active extra constraints, unsupported markers and folded option-like requirements
reject rather than lose meaning. These are input/selection limits, not enrollment
or runtime qualification. Native venv RECORD paths must use actual venv scheme
roots through the existing destination binder; the private --target output mapper
has different relative-depth semantics and cannot be applied unchanged to them.
For an existing HAA-owned distribution, use its validated transaction ledger for
exact installed ownership; its preserved RECORD can still contain the private
output recipe's cross-scheme depths. Never reinterpret that ledger-owned output
as a native pip layout. A mixed native/managed environment must not let a native
RECORD claim override existing HAA ownership. Native shared/ambiguous ownership
rejects rather than inventing an authenticated Artifact digest from a local file
snapshot. Unrelated retained output is protected, not granted new inspection
approval. Retained editable/direct-source inputs need truthful rejection when
their paths/source cannot be represented in the supported update selection.
An untrusted native RECORD must never gain authority to mark arbitrary user files
obsolete. Native enrollment needs independent canonical installed-footprint proof
before replacing/removing its claimed paths; otherwise reject it unchanged. The
first managed-venv wiring may use the existing validated ledger while native
enrollment remains MISSING. An explicit --adopt request alone is insufficient.
