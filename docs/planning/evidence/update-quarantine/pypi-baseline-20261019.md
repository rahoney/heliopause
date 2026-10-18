# SEC-UPD-001 — PyPI upgrade baseline / 2026-10-19

Base29c58f0f52388e7d4a36e613fbdce50df9a60a77, milestone/m13-release-build-packages.
Tracked/index clean before this lane's audit; unrelated local notes/raw remain
untracked. The preceding bounded npm lane is locally qualified and pushed in
existing draft PR38; fresh CI330/run39138729054 is in progress, not yet successful.
Only SEC-UPD-001 is IN_PROGRESS. No PyPI production source has been edited.

Read: update contract06, M9 product-install contract, M5 qualification, current
M12 Python source/graph/derived-wheel and ownership responsibilities, artifact
lessons L01/L02/L15, evidence L11/L16 and sandbox L03/L08/L12. Existing scanner,
Policy, source/derived-wheel binding, optional post-install surface distinction,
resource policies and supported runtime identities remain intact.

| Responsibility | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | Actual owner / MISSING |
| --- | --- | --- | --- | --- | --- |
| Public PyPI/named-source inspect/install complete selected graph | YES | YES | YES historical | YES historical | sandbox/pypi_resolver.go ResolveDependencies; application/install_inspect.go; bootstrap Python composition; reuse |
| Exact offline wheel/derived-wheel output validation | YES | YES | YES historical | YES historical | promotion/pypi_promotion.go preparePyPIProject, validatedPyPIDestinations, promoteActiveVenv; reuse |
| Owned venv replacement/retained output/shared-owner collision/rollback | YES | YES | YES historical | YES historical | pypi_venv_transaction.go beginPyPIVenvTransaction/prepare/checkBefore/commit/rollback; existing real tests include managed replacement, every rollback phase, recovery conflict, ownership drift and shared owners |
| Early original-state guard before upgrade selection | NO | NO | NO | NO | Existing venv transaction starts during Promotion after inspection; update must begin/reuse it before resolution and retain exact inventory/metadata throughout |
| Native-shaped pip install --upgrade / -U | NO | NO | NO | NO | cli/root.go static pip install and AddPyPIInstallSources lack upgrade flags; current reference parser accepts name or name@exact, not native specifiers |
| Default only-if-needed from existing environment | NO | NO | NO | NO | PyPIResolver ignores InstallContext and resolves a fresh environment; missing bounded inert current metadata, retained constraints, complete proposed selection and drift/conflict validation |
| Explicit native venv enrollment | NO | NO | NO | NO | Existing default install protects foreign baseline; new update enrollment needs bounded METADATA/RECORD inventory, exact paths/owners, snapshot and selected-set inspection before replacement; never execute target Python/packages |
| Update operation/report and pre-resolution session wrapper | YES generic boundary | npm only | YES local npm | NO for PyPI | ports/update.go and application/install_update.go reuse the existing install pipeline; require concrete PyPI session/resolver and CLI binding, not a parallel scanner |

## Selection/inventory design checks before source

The first public-PyPI slice must preserve pip's default only-if-needed behavior,
retained distributions and actual dependency metadata, without trusting installed
files as prior inspection approval. Read Python3.14 venv controls, bounded inert
dist-info metadata and RECORD as data; never invoke its Python, import its modules
or copy executable package payloads to the resolver. Guard before these reads.
Unmanaged/ambiguous state either needs explicit supported enrollment or a truthful
unchanged rejection. Default install's foreign-collision rule remains unchanged.

Pinned pip26.2.1/Python3.14.7 offline controlled metadata probe confirms that an
upgraded root/dependency can be selected while a satisfied dependency stays at its
older installed version. It also proves that a fresh solve pinned only to the
proposed version numbers cannot preserve different requirements in the actual
retained environment metadata. Raw pip-semantics-probe.* is a semantics probe,
not production gVisor/inspection qualification or an approved implementation.

The final pip-semantics-closure probe PASS15.74s covers five normal/conflict/no-op/
transitive-only/missing-root cases. Freeze proposed exact versions and actual
unchanged metadata requirements as constraints while requesting only the root's
complete closure. This reuses the final one-primary report parser and avoids
invented roots/edges or fetching unrelated named-source distributions from PyPI.
The detailed [implementation sequence](../../../user-journey-cli-ia/06-update-quarantine.md#pypi-selection-review-before-implementation)
is canonical; raw probes remain bounded tool-semantics evidence only.

Second review caught these additional owner boundaries before production source:
the first delta can be empty or root-less and is never complete-set authority;
unrelated metadata must constrain changes even if publisher metadata differs;
the complete closure must retain source ownership without re-requesting unrelated
packages; and legitimate sdist-derived wheels extend the resolver graph, so the
npm no-derivation comparison is not a PyPI promotion check. Hold/reuse the existing
venv transaction across selection/private output rather than acquiring it twice.
Native RECORD ownership must account for old bytecode/entry points/shared files,
be bounded and never infer safety. New argv must reach actual observer consumers,
not only fake Go runners. No PyPI production source was changed during this review.

This is a design under pinned-tool validation; production implementation remains
MISSING. Check report primary/requested semantics, markers/specifiers, source and
runtime bounds, native shared/foreign ownership and cleanup before wiring. Named
PyTorch source selection has a distinct existing resolver path and must be audited
independently; public-PyPI progress cannot qualify its CPU/CUDA consumers.

## First implementation slice — input and selection data

New adapter-local upgrade_input.go and upgrade_selection.go normalize the scoped
native request, freeze inert installed METADATA into a bounded immutable control,
retain active unchanged requirements, accept genuine empty/root-less deltas and
bind the final complete single-root report to proposed exact versions and changed
source/URL/digest/metadata. They reuse current name/version/marker parsing and the
existing complete report/graph construction. Missing/null arrays, wrong runtime,
duplicate/unexpected roots and substituted/omitted changed candidates reject.

Canonical format and focused TestUpgrade tests passed at pypi-selection-first
(1.43s). The same production source passed pypi-data-quick101.57s,
security6.90s, docs0.26s and targeted Race6.50s. A later documentation-only
closure check includes this record; those results do not qualify runtime wiring.
This is IMPLEMENTED=YES / WIRED=NO /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO for this data slice, not command availability.
MISSING remains the concrete early venv session/inventory, pinned isolated resolver
binding, native enrollment, CLI/reporting and actual upgrade qualification. Public
and named-source existing install paths have not been changed by this slice.
