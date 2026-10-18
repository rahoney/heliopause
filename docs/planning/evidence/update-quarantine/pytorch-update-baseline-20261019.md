# SEC-UPD-001 — named PyTorch update baseline / 2026-10-19

Base2e8844f41d9aadea7c0ad2a0dbd59bdb9b239a8a, pushed to existing draft PR38.
Native public PyPI is locally qualified; its new CI332/run39154516210 is pending.
Tracked/index clean before this audit; local user notes/raw remain untracked.
Only SEC-UPD-001 remains IN_PROGRESS. No named-update source has been written.

Read: contract06, PyPI baseline/managed/native qualification, M5 and M12 Python
source/graph/resource contracts; artifact L01/L02/L15, evidence L11/L16 and
sandbox L03/L08/L12. Reuse existing scanners, Policy, source ownership, derived
wheel binding, optional post-install surface reporting and venv transactions.

| Responsibility | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | Actual owner / MISSING |
| --- | --- | --- | --- | --- | --- |
| Named profiles, exact per-node source ownership and source-pinned traversal | YES | YES ordinary install | YES historical scope | YES ordinary scope | pypi/source_profile.go and sandbox/pypi_resolver.go resolvePyTorchGraph/resolvePyTorchCandidate; reuse |
| Per-node Simple/Core Metadata cross-check | YES | YES ordinary install | YES historical scope | YES ordinary scope | Same candidate resolver, ParseInstallationReportForProfileWithExtras, CrossCheckReport; reuse |
| Full inspection, derivation, Evidence, M5 and offline Promotion | YES | YES install/public update | YES bounded existing scope | YES existing scope | Existing application, inspectors and PyPIPromotion; reuse |
| Guarded original venv ownership/replacement and rollback | YES | YES public update | YES bounded local | YES existing bounded scope | PyPIUpdatePreparation and original pypiVenvTransaction; reuse |
| Root-profile resource context/duration | YES | YES ordinary named install | YES historical scope | YES ordinary scope | pyTorchResourceInstaller and ResourcePolicy; wrap update too, do not use public defaults |
| Named native-shaped -U/--upgrade request and frozen source input | NO | NO | NO | NO | CLI and UpgradeRequirement currently public-only; explicitly bind selected profile/local version and canonical control |
| Named only-if-needed complete source-pinned update graph | NO | NO | NO | NO | Public two-phase selection and named ordinary traversal are distinct; source-aware new selection and retained constraints are missing |
| Actual named update qualification | NO | NO | NO | NO | Same-name/new bytes, transitive-only/no-op fresh Evidence, negative preservation, resource/helper and affected CPU/CUDA consumers |

## Findings and reviewed implementation direction

PyPIResolver.resolveDependencies dispatches named profiles to resolvePyTorchGraph
before its public upgrade branch. Removing only the public guard would therefore
silently run ordinary named installation selection, omit update controls and lose
only-if-needed semantics. The public pip whole-report path cannot replace this
traversal: named dependencies are deliberately resolved individually with --no-deps
and canonical ownership routing, never a multi-index selector. All actual named
candidates still require their original Simple/Core Metadata cross-check.

The existing traversal resolves fresh candidates and aggregates incoming bounded
requirements; update selection must additionally retain a compatible installed
dependency rather than eagerly choose a newer version. The requested root is
upgraded. A satisfied/no-op result still requires fresh authenticated complete
selection, inspection and Evidence. Local build suffixes belong to named profiles;
public input must continue rejecting them. Do not copy one root source identity
onto public dependencies or fabricate graph edges/extra roots from retained state.

Extend existing UpgradeRequirement/Input with a canonical source-aware constructor
and immutable source field. Public callers/default controls keep their current
behavior. Named controls must bind the exact named profile; roundtrip/runtime/
request checks reject a changed/unknown/missing named source. Use existing
ParseReferenceForSource, profile-aware version helpers and per-node report parsing.
Keep retained METADATA inert and preserve the original whole-environment guards.

Extend the existing named traversal in place with an optional private update
selection state. Ordinary install calls keep the existing path. Use immutable
pinned pip and inert metadata to choose the root and only-if-needed dependencies,
one canonical source at a time. A single-node empty pip delta is legitimate when
the installed version satisfies the request; perform a fresh exact candidate
lookup, not an old ALLOW reuse. Aggregate every incoming requirement and preserve
existing traversal/report/resource bounds. Do not mount or execute target packages.

After proposed versions are fixed, build exact constraints plus active requirements
of distributions whose versions remain unchanged, excluding superseded metadata.
Freshly authenticate the complete requested-root closure at those exact versions
through the same per-source candidate resolver with retained constraints. Reject
conflicts, missing/substituted candidates and any source/runtime/digest mismatch;
do not solve them with another source, fake root, omitted edge or lowered check.
Bind original input, proposed selection and final per-node reports into bounded
immutable controls with an unambiguous aggregate encoding. The existing named
16MiB report traversal bound is not permission to exceed an individual4MiB control;
enforce representation limits explicitly. Preserve each original raw report.

PyPIUpdatePreparation must validate the named source's controls and exact complete
graph, then keep the same held transaction through ordinary full inspection and
offline commit. Bootstrap must apply the original named resource context and
duration to each update attempt and retain prerequisiteCheckingResolver on the
new graph. Existing public-native footprint proof stays public/site bounded;
unproven native named files, generated scripts and oversized/ambiguous layouts
reject unchanged rather than grant ownership. Do not reinterpret an old named
local version as public PyPI or broaden native limits without independent proof.

## Concrete edit owners and second review

- artifact/pypi/upgrade_input.go: source-aware request/control roundtrip and exact
  named-local version intent, preserving public defaults and bounds.
- artifact/pypi/upgrade_selection.go and narrowly related named selection data:
  proposed versions, actual retained constraints and typed per-node final binding.
  Reuse Candidate/InstallationReport/BuildLockedGraph and source ownership.
- sandbox/pypi_update_resolver.go and pypi_resolver.go: extend the existing named
  traversal/candidate calls, retained selection and final controls; preserve
  lifecycle/observer/egress, per-source indexes, Simple/Core Metadata and ordinary
  install invocation. Add private code only for this active named-update gap.
- promotion/pypi_update.go: select profile-aware input/complete-control validation
  in place; reuse early guards, owned metadata validation, public native proof,
  derivation and same transaction. No new promotion engine or generic Port.
- bootstrap/bootstrap.go, cli/pypi_update.go and existing CLI catalogs: route the
  explicit source and profile resource wrapper, preserve early unsupported-flag
  refusal and accurate English/Korean help/reporting. Existing UPDATE JSON stays.

Before source, review this plan against real pinned pip single-source no-deps
semantics, final graph/source/constraint binding, public compatibility and resource
context. Unit tests must distinguish changed root, retained/transitive-only and
empty/no-op deltas; source/local suffix substitution, hidden new dependencies,
retained conflicts, omitted required checks, block/review and failed cleanup.
Run final canonical profiles and meaningful Race, actual installed CLI update/
no-op/negative consumers, existing public/native/npm consumers and affected named
CPU/CUDA qualification once on a stable candidate. Historical install/CUDA
evidence cannot qualify a new update command; profile success cannot transfer.
Preserve original installed identities and canonical helper cleanup; archive
Evidence and remove completed disposable payload/cache/operators after final use.

Second-review checklist: named dispatch cannot bypass new selection; no arbitrary
indexes or mixed-source ownership; original metadata/lock frozen before lookup;
only-if-needed retention and unchanged requirements remain real; complete final
closure and derived binding preserve source; no post-ALLOW resolution; new graph
prerequisites and root resource budgets remain wired; all failure paths leave
unproven original files and actual ledger state untouched. All missing named
states above remain NO until implementation and corresponding evidence exist.

## Pinned semantics and completed pre-source review

The network-disabled pinned Python3.14.7/pip26.2.1 own-wheel probe passed ten
cases in15.22s: requested-root --upgrade --no-deps selects only the root; compatible
dependency selection without --upgrade emits an empty delta, needed dependency
selects a new version, and empty/no-op selections can be freshly resolved exactly
with --ignore-installed. Named-style local-version upgrade/no-op and exact fresh
selection retain the local suffix. Per-node final exact constraints preserve the
chosen version and reject a retained dependency conflict. Superseded old-root
requirements are excluded from the final constraint input. These are pinned-tool
semantics, not source provenance, production gVisor or named update qualification.

The probe mounted no Host directories or target code, ran in runc/network-none/
read-only with noexec tmpfs, and its exact container was confirmed removed. Own
wheel payloads vanished with tmpfs; Host operator/cid were archived and deleted.
Original source/stdout/receipt/cleanup are local pytorch-update-semantics-* raw.

Second review confirms that a generic public resolver call cannot implement this
lane. Reuse and extend the existing source-pinned traversal, with root upgrade
and dependency only-if-needed selection separated. Preserve per-node raw reports,
complete final constrained closure, extras/markers and canonical source ownership.
The root-profile resource wrapper must cover update selection through promotion.
Do not read an empty delta as absent dependency or use an exact old version as
old safety approval. Existing public defaults and public native ownership scope
remain intact. Implementation is authorized for the documented gap; no existing
implemented scanner/transaction is being replaced. Named IMPLEMENTED/WIRED/
QUALIFIED/ACCEPTANCE_CLOSED remain NO until corresponding source/evidence exists.

## First source slice — source-bound selection data

The existing upgrade_input.go now has ParseUpgradeRequirementForSource and an
optional canonical source_id in its immutable input. Public v1 controls remain
byte-for-byte unchanged. Named root intent preserves exact/local/numeric version
semantics and rejects unknown profiles or a public project under a named root.
upgrade_selection.go reuses the original proposal/retained-constraint owner for
source-aware deltas and mixed-source final binding. ProposeSourcePinnedUpgrade
and ValidateSourcePinnedFinal require a complete named traversal and compare all
fresh first-pass candidates, including same-version bytes, against the second
selection. Unrelated installed distributions remain constraints, not graph roots.

New upgrade_named_test.go exercises all four profile inputs, local/no-op/transitive
selection data, public compatibility, source substitution, incomplete graphs and
same-version content changes between traversals. Focused9.32s, canonical
Quick100.47s and targeted Race23.90s PASS for this fixed source. Final docs/security
receipts use pytorch-data-docs/security. This is implemented data only: named
resolver selection, CLI/Promotion binding, resource wrapper and actual named
update qualification remain MISSING. No named update command is newly enabled.

Further review before traversal: installed CUDA METADATA can request the existing
bounded cuda-toolkit extras. Public upgrade input currently rejects active extras
because pip constraints cannot encode them. Named support must preserve their
version constraint AND selected component requirements through the existing
extra-aware candidate traversal. Do not delete the extra list, treat it as a plain
constraint, or make arbitrary extras valid. Evaluate only the existing canonical
CUDA extra/marker subset as inert data, bind retained extra obligations and actual
new candidate edges, and test superseded metadata versus unchanged requirements.
This remains a named/CUDA implementation gap; data-input success above does not
qualify it. Existing public rejection stays until a separately validated scope
can represent the original requirements without loss.

## CUDA obligations and named incoming-version checkpoint

Direct pytorch-retained-extras-red1.17s reproduced the inability to represent the
supported retained CUDA extra request. Existing bounded requirements now expose
VersionConstraint while the immutable input separately retains typed extra
obligations. upgrade_extras.go derives them from unchanged old versions and fresh
selected metadata; unchanged Toolkit metadata is evaluated with the actual extra
set to preserve conditional component constraints. Changed parent/Toolkit metadata
is excluded. Candidate retains its requested extra context, and complete first/
final selections must preserve it and real component edges. Public updates may
retain this known CUDA metadata as constraints without making it another root;
arbitrary extras/unknown active forms remain unsupported. Contract06 owns the
reviewed boundaries. The red/focused first green9.09s and follow-up8.87s remain
separate exact-source receipts.

The existing traversal also used a source-less numeric predicate for incoming
named dependencies. A source-shaped torchvision-to-torch production-method
fixture passed2.69s with Candidate.SatisfiesBoundedRequirements. Overlaying only
the previous predicate in the private projection made the same fixture fail with
the exact requirement mismatch on CPU/cu126/cu130/cu132; original source hash was
restored. The new method validates canonical source/project and applies numeric
bounds to a named build's public release part. Public local versions and wrong
projects still reject. This narrow consumer proof is not live install evidence.

This current fixed source passed canonical format2.33s, Quick99.03s and targeted
Race24.98s. Source inventories/raw causal evidence are pytorch-extra-core-*,
pytorch-named-local-causal.json and the matching green/legacy-red logs. Final
docs/security receipts use pytorch-extra-core-docs/security. Named update resolver,
per-node no-op/first-final report controls, session/CLI/resource binding and actual
CPU/CUDA update qualification remain MISSING. No named upgrade is enabled yet.

## Source-pinned producer and command wiring checkpoint

The subsequent current candidate implements the missing producer and wiring in
place. sandbox/pypi_named_update.go extends the existing source-pinned traversal
with private update state. The root upgrades; dependencies use only-if-needed
against frozen inert METADATA. Every selected node, including an empty delta,
gets a fresh exact candidate and the original Simple/Core Metadata checks.
The complete first traversal fixes versions and retained constraints; the second
source-pinned traversal accepts only the same exact candidates and extra context.
upgrade_node_selection.go and upgrade_source_controls.go bind original input,
first/final per-node raw reports and actual retained constraints in four canonical
controls. Each encoded control remains bounded to4MiB, and each phase retains
the64-record/16MiB aggregate bounds. A first record contains both its single-node
delta and fresh candidate report; this is not a claim of64 physical pip calls.

promotion/pypi_update.go validates the complete controls, graph, runtime and digest
before reusing its original held transaction and new-set inspection/promotion.
Bootstrap routes each named updater through the existing root resource wrapper;
the resolver requires the matching context and preserves the15/40-minute named
duration. Public defaults and independent public-native footprint proof remain.
CLI explicit --source and native -U/--upgrade selectors now reach this path, with
accurate English/Korean help. Unproven native named files still reject unchanged.

The same fixed candidate passes canonical Quick130.31s, security9.27s, docs1.45s
and targeted Race29.61s. Four-profile fixtures cover changed/no-op producer data,
source/input/constraint/graph/runtime/digest substitution, held original guards,
explicit CLI intent and named resource deadlines. A CUDA source-shaped producer
fixture preserves retained cublas+cudart extras, conditional constraints and real
component edges through the final exact lookup. Focused5.53s passes. The first
Quick's graph-negative fixture changed a reference without changing resolved
nodes; its corrected fixture substitutes the artifact hash. The first CUDA
fixture lacked authenticated Requires-Python; both own report and HTML metadata
were corrected. Neither fixture error demonstrates a product defect or justifies
relaxing production authentication. Failed raw results remain failed.

Raw inventories/receipts: local update-quarantine-20261018/pytorch-wired-*,
pytorch-cuda-session-{focused,green} and their format/source inventories. Candidate
CLI27,129,791B was built with pinned Go1.26.9/CGO0 in3.45s from the same inputs;
its exact identity is recorded in pytorch-wired-candidate.json. Actual UID1000
installed-path CPU2.14.0+cpu to2.14.1+cpu/no-op/negative tests are now running under
the unchanged canonical helper lifecycle. Source IMPLEMENTED/WIRED=YES;
QUALIFIED/ACCEPTANCE_CLOSED=NO. Actual per-profile outcomes, confirmed restoration,
completed disposable-data deletion and new-source remote qualification remain
MISSING. This checkpoint does not close named or whole-PyPI acceptance.
