# Update quarantine — SEC-UPD-001

Status: IN_PROGRESS; user authorized implementation after reviewed planning.
The bounded npm lane is qualified for29c58f0 including fresh CI330 ten SUCCESS.
The reviewed bounded PyPI lane is qualified: [public managed venv](../planning/evidence/update-quarantine/pypi-managed-qualification-20261019.md),
[public native enrollment](../planning/evidence/update-quarantine/pypi-native-qualification-20261019.md)
and [named managed-venv CPU/cu126/cu130/cu132 updates](../planning/evidence/update-quarantine/pytorch-update-qualification-20261019.md).
Their source-specific actual/CI scopes remain independent; no broader native named
support is claimed. The named source306 has four actual update/no-op/refusal/
restoration/cleanup profiles and ordinary same-source CI336. The reviewed
[Cargo lane](../planning/evidence/update-quarantine/cargo-update-qualification-20261019.md)
is also qualified for source85ac4c1: local actual20 phases, restoration/cleanup
and same-tree CI338 ten SUCCESS. Current lane: [Go baseline/selection
review](../planning/evidence/update-quarantine/go-update-baseline-20261019.md) preceded source; its [wired implementation qualification](../planning/evidence/update-quarantine/go-update-qualification-20261019.md) is pending final canonical/actual/remote checks. Preserve the existing explicit-version get path.
Other lanes wait for their independent baseline/qualification. See the
[npm evidence](../planning/evidence/update-quarantine/npm-qualification-20261018.md).
This leaf owns the user-requested update-command contract. It supplements the
installation journey, never weakens existing Policy or adds implicit Host execution.

## Terraform project upgrade — reviewed initial semantics

Pre-source [independent audit, edit map and second review](../planning/evidence/update-quarantine/terraform-update-baseline-20261019.md)
identify the remaining work. Implementation waits for Go acceptance closure.
Support `helox terraform init -upgrade [--adopt]` (`--upgrade` alias), with no
positional provider. Preserve exact `init namespace/type@version`. Only the exact
standalone native -upgrade token is normalized; values/false/duplicates, extra
positionals, --adopt without upgrade and native backend/module/mirror/chdir/
lockfile options refuse before composition. Existing global presentation options
remain; upgrade permits explicit JSON.

Following the [official provider upgrade semantics](https://developer.hashicorp.com/terraform/cli/commands/init#plugin-installation),
ignore old selections and select the greatest stable public version satisfying
all frozen literal constraints across reachable local modules. This initial lane
accepts stable constraints/releases only,1..32 explicit providers; no provider or
explicit prerelease constraint is unsupported. Validate the complete bounded
version list and then the exact selected platform/protocol; do not choose an
older version to hide unsupported latest platform/protocol. Reuse pinned x/mod
semver and existing constraint parsing; Host Terraform is never invoked.

Each provider uses one frozen discovery/versions/package selection; the same
bytes bind its signed acquisition. Complete snapshot selection has no requested
primary or fabricated dependency edges. Preserve all configuration bytes and
module identities. An old lock must parse without orphan providers. Same-version
hashes must still match and retain other-platform hashes; changed versions use
the new authenticated package. Every selected provider receives fresh existing
required verification/static/isolated probe/Evidence/entry and set Policy, even
when nothing changes. RPC/plan/apply/cloud and remote/backend/module work remain
outside Provider installation scope.

Managed projects validate their independent old approval/cache/Evidence and
complete installed tree. Explicit --adopt may supersede old control equality,
never those independent proofs. Providerless native config/lock requires --adopt;
unmanaged installed .terraform trees, foreign marker, corrupt/aliased authority
and incomplete recovery refuse unchanged, including explicit adoption attempts.
Lock hash presence does not grant native provider ownership.

Reuse the existing complete-project workflow/cache/transaction. Snapshot-only
update admission is confined to upgrade; ordinary exact init stays strict.
Upgrade retains identical lock bytes/inode/mode/mtime, preserves ordinary existing
permissions when changing lock, and creates an absent lock0600. Guard retained
files at all publication/approval/rollback boundaries; preserve foreign mutations
and recovery evidence. Human/JSON project-update results report source, selected
identities, actual checks/Evidence/Policy and confirmed operation/promotion; late
failure never becomes success. Existing signing, probe, bounds and runtime remain.

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

This is the reviewed implementation map. Qualified npm/PyPI scopes are recorded
above, and the reviewed Cargo lane is now qualified. Go -u, Terraform upgrade and
GitHub replacement remain planned. Execute
one ecosystem at a time after CLI-UX-002; keep other lanes unmodified until their
own baseline/selection contract is verified. Minimum planned forms are:

| Lane | Supported or planned form / behavior | Existing location / method to reuse or extend | Required boundary behavior / original baseline gap |
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
separated existing owned-venv transactions from the original upgrade-selection gap.
The implementation started with public PyPI and audited the named resolver independently;
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
first managed-venv checkpoint used the existing validated ledger while native
enrollment was still missing. The later qualified native path supplies the
independent proof below. An explicit --adopt request alone is insufficient.

### Native PyPI ownership review before implementation

The independent native baseline follows the managed checkpointc93ce58. Pinned
Python3.14.7/pip26.2.1 offline fixtures confirm that a real venv RECORD uses
../../../bin, ../../../share and ../../../include/site/python3.14 paths, unlike
the private --target recipe. Default compilation adds an unhashed
module/__pycache__/name.cpython-314.pyc row; INSTALLER is pip plus newline and
REQUESTED is empty. These are tool-semantics facts, not ownership or safety proof.
Original source, stdout, failures and cleanup remain in local native-layout raw.

Implementation sequence after the second review:

1. Under the existing early transaction, distinguish exact ledger-owned METADATA
   from unmanaged metadata. An incomplete conflicting HAA owner rejects. With
   explicit --adopt only, freeze native RECORD bytes, each bounded referenced
   original file and its parents before any external selection/lookup. Interpret
   canonical site-relative paths and exactly three-parent cross-scheme native
   paths through the existing venv binder; reject aliases, escapes, links,
   duplicate/ambiguous claims and overlap with a different HAA owner. These are
   untrusted snapshot claims, not transaction ownership. Bound each RECORD to1MiB,
   combined native RECORD controls to4MiB, all referenced files to10,000, each file
   to256MiB and combined payload to1GiB. Preserve unrelated native files unchanged.
2. After complete proposed selection but before inspection/ALLOW, look up only
   selected native old distributions through the existing isolated exact public
   resolver. Reuse Intake.Acquire and InspectWheel for the authentic old wheel,
   including its declared/observed digest and Surface.InstalledFiles. Compare
   canonical ordinary file hashes/sizes to the early snapshots, using that
   existing role/location map and venv binder. Require complete unambiguous
   footprint coverage. Old sdist/direct-source/unproven layouts reject unchanged;
   other old graph candidates do not become inspection nodes or approved files.
3. The initial native footprint supports site/dist-info wheel payload and the
   independently verified standard pip-generated RECORD/INSTALLER/REQUESTED.
   Selected native scripts/entry points/cross-scheme payloads remain unsupported;
   unrelated bootstrap pip scripts may be snapshotted and retained without new
   ownership. Default timestamp bytecode may be removed only at the exact cache
   location derived from an authenticated original Python module, with bounded
   original bytes, matching pinned runtime magic/cache tag, flags0, original
   source timestamp/size and stable fingerprint. Never load or execute bytecode;
   this proves a generated cache role for removal, not safe/authenticated bytecode
   contents. Optimization/hash-based or unknown caches reject. Any runtime-metadata
   probe reuses trusted Docker execution with only empty private output mounted,
   immutable Python -I, no packages/target interpreter/network and bounded output.
4. Recheck all originals, then add only proven selected native paths to the held
   transaction's in-memory current state with the authentic old wheel digest.
   Never derive Artifact integrity from local fingerprints or write enrollment
   metadata before new-set ALLOW. Bind old provenance, exact original hashes and
   generated-role distinctions into an immutable selection control. Reuse full
   new-set inspection, original derived-wheel rules and exact offline Promotion.
   Rollback must restore the original native payload and actual ledger absence,
   including failures after publication or metadata changes.

Second review explicitly preserves pre-selection snapshots, private/native RECORD
separation, exact HAA metadata ownership, generated-cache vs approved-byte
distinction, no post-ALLOW resolution, shared/foreign protection and nil-ledger
rollback. Tests must include a RECORD that falsely claims a user file, wrong old
wheel hash/source/metadata, changed original files after selection, ambiguous
owners, forged/unsupported cache roles, missing required new inspection and
partial commit/recovery conflicts. Actual native positive/default-bytecode and
unchanged refusal consumers plus existing managed/NPM/canonical regressions are
required. These checks subsequently passed within the linked bounded native
qualification; the review remains the design contract, not an open native blocker.

Concrete edit owners: new promotion/pypi_native_inventory.go owns bounded native
claims/snapshots and uses pypiVenvPlan.bind; new pypi_native_ownership.go owns old
intake/static footprint and bounded trusted runtime metadata, using existing
Artifact/DependencyResolver/DockerRunner collaborators. Extend pypi_update.go
in place for explicit adoption, pre-selection snapshot checks, proof control and
verified in-memory transaction enrollment. Add a constructor-time immutable
WithNativeEnrollment copy of PyPIUpdatePreparation; no new generic Port, scanner
or runtime config is needed. Bootstrap passes its existing intake/root and raw
source resolver for old-footprint metadata lookup, while the new upgrade still
passes through prerequisiteCheckingResolver and its actual new-graph validation.
The inspection prerequisite loader already uses that raw resolver for its own
distinct lookup; an old footprint query must not misapply the new inspection-only
prerequisite to a different old graph. Tests must prove the new selected graph's
prerequisite validation is preserved. Leave the original PyPIPromotion, wheel
mapper, full inspection and transaction engine intact, changing only proven gaps.
The existing pypiFileSnapshot.matches currently re-reads without a byte limit.
Native bounded snapshots must retain an optional private read bound through every
original-state/backup/rollback comparison; otherwise a grown native file can
escape its initial read limit. Extend that existing fingerprint in place, retain
the original default for large HAA-owned Python payloads and the explicit64MiB npm
bound, and test the affected existing consumers. This is no new global file cap.

Native source checkpoint after the reviewed baseline: bounded inventory, independent
old-wheel proof and runtime metadata live in pypi_native_inventory.go,
pypi_native_ownership.go and pypi_native_runtime.go; pypi_native_*_test.go covers
claims, original drift, canonical footprint/cache roles and nil-ledger rollback.
Reject intake/staging roots overlapping the venv before any mutation. Bound the
complete old-footprint lookup to five minutes under the original transaction;
the empty trusted-runtime metadata query has its own thirty-second limit. Every
existing ledger owner requires its exact single-owned installed METADATA before
selection, including retained distributions. Different paths to one original
file object reject as aliases. Unit fixtures alone are not actual native qualification. The bounded native
qualification record now links canonical and eleven installed-client consumers;
named-source work remains open. The [named qualification](../planning/evidence/update-quarantine/pytorch-update-qualification-20261019.md)
records CPU version-change/fresh no-op, affected public/native consumers and the
remaining CUDA/new-source remote boundary separately.

For original-state inventory, retired dist-info without METADATA may contain only
directories left by the existing transaction. Bound the cumulative retired tree
to100,000 directories, depth64 and relative path4096B; use root-scoped reads and
freeze exact membership/inode/mode/mtime in the existing guard. Files, symlinks,
unknown objects, new empty directories and any identity/content drift refuse
unchanged. This grants no old ownership or safety approval and does not delete
directories to bypass validation. pypi_update_retired.go owns this classifier;
the original transaction and required inspection/Policy remain unchanged.

### Named-source retained extras and version review

The [named baseline](../planning/evidence/update-quarantine/pytorch-update-baseline-20261019.md)
owns the independent source audit, pinned single-source selection semantics and
data checkpoint. Extend the existing source-pinned traversal, preserving its
source ownership and actual Simple/Core Metadata cross-check. A named update is
not enabled by the source-aware input/proposal data alone.

Before wiring CUDA, represent its existing bounded cuda-toolkit extra requirements
without putting extras into pip constraint lines or silently dropping components.
Keep original METADATA unchanged in the immutable input. Parse only the already
supported canonical CUDA extra/marker subset as inert data; arbitrary extras stay
unsupported. Retain the plain version constraint together with a separate typed
extra obligation, derived from unchanged installed distributions and the actual
new candidates. Superseded metadata contributes neither its old constraint nor
its outgoing extra request. Re-evaluate retained toolkit metadata using the
required extra set so conditional component requirements remain real constraints.

The existing bounded requirement owner should provide the plain constraint form;
the existing Candidate/report parser should retain its actual requested extra
context. Final source-pinned selection must carry every required extra through
the original extra-aware per-node parser and real dependency edges. An unselected
retained distribution remains protected original state, not an extra graph root.
Known retained CUDA extras may be represented for a public update only when this
same typed obligation and conditional-requirement model preserves their complete
meaning; keep the existing arbitrary-extra rejection. Do not infer extra safety,
approval, implicit installation or new Host execution from metadata.

Tests must cover unchanged parents/toolkit, changed parent/toolkit, combined
incoming extra requests, inactive/unknown markers, missing/forged component edges,
control roundtrip and new final source/bytes substitution. Pinning a compatible
installed version still requires a fresh complete candidate and full inspection.
Per-node no-deps selection uses inert metadata only; final exact constrained
traversal retains all required extras and original resource/report bounds.

Named local versions also need source-aware numeric requirement comparison in
the existing traversal: use the validated named version's public release part
for numeric bounds, while public-source local versions remain invalid. Reuse
existing parsing/comparison in place, with explicit source binding and tests;
do not allow a suffix to select a profile or make an arbitrary version valid.
These are remaining representation/selection gaps, not permission to replace an
implemented scanner, Policy, wheel mapper or transaction.

Named UPDATE returns a bounded pip report from the same trusted invocation after
successful selection; pip's human output is separate from report bytes. Failure
must not return a stale report. Within one traversal phase only an identical
source/project/aggregate requirement and extra context may reuse the already
fresh candidate, with every incoming requirement checked. Changed intent and the
next phase require fresh selection. This is an adapter efficiency constraint,
not permission to increase observer limits or reuse previous inspection approval.

### Cargo lock update — reviewed initial semantics

Support current-project helox cargo update, optionally one -p/--package <name>
or <name>@<locked-version>, plus explicit --adopt for supported enrollment.
The version in the selector identifies the old locked package; it does not pin
the proposed version as --precise would. A selector must uniquely identify a
public registry entry in the frozen Cargo.lock. Reject local/workspace, missing,
ambiguous and URL selectors. Require a supported anchored current project with
an existing lock. No selector updates the compatible complete lock graph; one
selector follows the pinned tool's conservative dependency behavior. Preserve
every selected manifest/range/feature and local source; exact pins remain exact.

Reject positional selectors, repeated -p, --precise, --recursive, --breaking,
--workspace, --dry-run, --locked/--offline, custom manifest/lock paths, alternate
registries, arbitrary configuration and toolchain flags before mutation. HAA's
existing language/output/progress options retain their own meaning.

Without --adopt, require proven current independent HAA project approval.
Explicit --adopt may enroll supported native or deliberately changed controls
only after bounded original capture and whole-new-set inspection. Validate any
old independent approval/cache/Evidence even when its equality to deliberately
changed inputs is being superseded. A foreign HAA marker without independent
approval, corrupt old state/cache or recovery journal still refuses. Old source
metadata and checksums grant no previous safety trust. Only guarded lock/control
approval is replaced; arbitrary old crate files and build outputs are not adopted.

Preserve unchanged Cargo.toml/local manifests including identity and permissions;
keep identical Cargo.lock in place on no-op. A changed lock contains exactly the
inspected selection and preserves its supported original permission mode. Reuse
the existing backup/commit/rollback and complete cache approval. CommitSnapshot's
current-state-only checks and old requested-primary commands remain strict.

Every selected public crate receives existing independent verification/static
inspection with fresh recorded Evidence and complete-set Policy. No-op has no old
ALLOW shortcut. Build scripts/macros retain the separate helox cargo build
isolated observation contract; lock selection does not execute crate builds.
Human/JSON success requires confirmed publication and successful close. Failure
and rollback uncertainty cannot become update success. The
[baseline/edit map and second review](../planning/evidence/update-quarantine/cargo-update-baseline-20261019.md)
own implementation locations, pinned-tool evidence and acceptance checks.

Cargo update JSON uses `helox.project-update-result/v1`, a project-scoped result
without a fabricated primary artifact. It includes operation/publication status,
source, optional selector, exact graph identity, dependency-free state and each
entry's identity/digest/run/checks/Evidence/Policy. Completed trusted inspection
facts can be retained with a failed update, but its publication status remains
NOT_COMPLETED. That state does not assert successful rollback. Existing
`helox.operation-result/v1` and other commands' output contracts are unchanged.

### Go named upgrade — reviewed initial semantics

The first supported upgrade form is `helox go get -u <module> [--adopt]` in the
current project's go.mod directory. One exact public module path must occur once
in the original require directives, whether direct or indirect. Preserve case
and official module-path grammar. Reject missing/duplicate membership, package
patterns/subpackages absent from require, path@version, URLs, multiple selectors,
argument-free -u, -u=patch, custom proxy/source/build/toolchain flags and go.work
selection. Ordinary `helox go get <module>@<version>` keeps its existing exact
semantics; --adopt is valid only for the upgrade form. Native Go commands run
outside helox do not receive this protection.

Use the pinned guest Go's actual get -u selection once in the existing private
controls-only workspace. Its imported dependency upgrades and MVS constraints
are distinct from substituting latest in the old exact-version get path. Keep
public proxy and independent SumDB proof, local toolchain and fixed isolation/
observation/resource bounds. Freeze the actual requested module's selected exact
version, full selected project graph, controls and content digests; reuse strict
requested-primary Domain and Go promotion checks. Every selected dependency,
including retained versions, receives fresh acquisition/verification/static
inspection/Evidence and complete-set Policy before cache staging/publication.

Only require directives (including native indirect/MVS changes) and go.sum may
change. Compare all other parsed directive tokens before/after selection and
refuse module/go/toolchain/exclude/retract/tool/godebug/ignore changes. This also
refuses an otherwise compatible dependency that causes Go to raise the root go
directive. GOTOOLCHAIN=local independently prevents downloading another SDK.
No project source, old module cache or build output is executed or adopted for
selection. Actual module builds retain separate helox go build inspection.

Require proven matching HAA approval or explicit --adopt for supported native or
deliberately edited controls. Native input is an untrusted bounded control
snapshot. Validate any independent old canonical approval/cache/Evidence even
when explicit adoption supersedes its equality with current controls. Foreign
markers without independent approval, corrupt or aliased state, recovery state,
symlink/hardlink/non-ordinary-mode controls and incomplete selected coverage
refuse. No implicit fallback to adoption or old approval reuse is permitted.

Retain byte-identical controls in place, including inode/mode/mtime; preserve
original ordinary permission bits for changed controls. A newly created absent
go.sum uses0600. Recheck retained/published content and identities throughout
the existing guarded commit/rollback; do not replace that infrastructure. Old
get/download/build behavior remains strict. Complete human/JSON project reports
use the existing helox.project-update-result/v1 schema with source go-proxy and
real module/check/Evidence facts. Inspection ALLOW is distinct from confirmed
publication and close. A failure retains its primary cause, completed diagnostic
facts and truthful NOT_COMPLETED status without asserting a successful rollback.

The [Go audit, pinned semantics, edit map and second review](../planning/evidence/update-quarantine/go-update-baseline-20261019.md)
precede product source. The synthetic file-proxy probe is diagnostic-only and
never permits custom product sources or replaces actual installed qualification.
