# M12-005 implementation and design review

Reviewed the existing source and current canonical Architecture/Domain/Security
contracts before making source changes. This is implementation qualification
review by the current author; the separately planned M12-02 red-team gate remains
unperformed. It is not an independent release approval or proof of bug absence.

## Responsibility and abstraction

| Responsibility | Existing owner | Review conclusion |
| --- | --- | --- |
| Composition and resource ownership | `bootstrap.Run`, ecosystem composition helpers | Concrete wiring stays here; owned executor/supervisor defers join cleanup after the original error. Help/doctor avoid runtime setup. Length mainly reflects explicit ecosystem wiring. |
| Generic project resolve/inspect/cache/build | `application.ProjectInspectService`, `ProjectBuildService`, guarded Go/Cargo resolution services, Core ports | Go/Cargo reuse neutral complete graph and approval contracts. No ecosystem SDK or filesystem implementation moved into Application. |
| Exact source/integrity | npm/PyPI/GitHub/Go/Cargo/Provider artifact and verification adapters | Different registry/SRI/SumDB/index/OpenPGP protocols require separate adapters. Shared Domain identity/Evidence do not replace independent authentication. |
| Runtime behavior and completion | shared observer, direct-exec admission, Python transaction/ledger, observed project builder | A single observation/attribution owner is reused. Profile predicates retain exact installed-image/creator/topology restrictions; child output is not completion authority. |
| Publication and reuse | ecosystem cache/guard/transaction owners | Original controls, independent approval, exact retained Evidence and full tree must agree. Drift/uncertain rollback is rejected without overwriting foreign data. |
| Result and Policy | generic CLI presenters, Policy, Evidence store | Operation completion and ALLOW are separate. An error after ALLOW remains an operational/promotion failure and retains that prior Policy. |

Canonical architecture/import validation, build/test compilation, vet and pinned
Staticcheck passed. The inspected Go package graph is acyclic. These checks are
mechanical evidence; the owner/authority decisions above also require source
review.

## Confirmed leftovers and actual defect

Removed the zero-consumer initial Go/Cargo resolution wrappers, unused Host SDK
methods and unused `go_path` registration, as recorded in the
[before/after boundary](./code-cleanup-boundary.json). The isolated resolvers,
guarded project flow and observed builds remain the existing implementation.
Unused exported scaffolding is not reliably identified by Staticcheck alone;
tracked references and actual bootstrap wiring established the deletion scope.

A subsequent read-only AST identifier triage found four more zero-production/
zero-test API groups. Removed the initial `TerraformResolutionService`, obsolete
string-only `AggregateBoundedRequirement`, unused `RegistryEndpoint` getter and
unwired executor-less `ProbePython` wrapper (40 production lines). Current
`TerraformInitService`, typed extra-preserving aggregation, endpoint constant and
configured-executor `probePython` remain their existing owners. See
[the exact additional boundary](./additional-code-cleanup-boundary.json).
Other triage candidates retain actual test/API/neutral-contract roles; identifier
counts alone do not justify deleting methods or repeated-name public APIs.
Earlier005 runs are historical source evidence; all selected full profiles are
qualified again on the reduced source before final feature freeze.

The remaining npm inspect MIME mismatch was reproduced with an actual official
response. The existing resolver was corrected without changing its source,
integrity or resource/Policy rules; see [the exact regression](./npm-media-type/result.json).
Its actual public CLI passed on the current source; the final document HEAD CI
is separately verified. No implemented supported
subsystem was replaced or recreated.

## Reviewed hotspots

[AST inventory](./structure-summary.json) counted 227 production Go files,
215 test files and 2,035 production functions after initial cleanup. Function
length/branch-node count prioritized review; neither is a quality/pass threshold.
The later npm change is a narrow parser correction, plus a test and existing CI
caller. It does not alter the reviewed long transaction functions. After the
additional unused API cleanup, the current inventory is 227 production Go files,
216 test files and 2,030 production functions; no new abstraction or subsystem
is introduced.

- `bootstrap.Run` (358 lines): sequential construction, explicit supported Host
  checks and lifecycle ownership. Ecosystem factory helpers already carry Go,
  Cargo and Terraform wiring. Creating a second registry/framework solely to
  reduce this function's size would duplicate ownership and obscure startup.
- `PythonDynamicBackend.executeTrustedTransaction` (261 lines): authenticate
  closure/plan, create one cumulative resource authorization, prepare and freeze
  the volume, then observe exact independent units. Final status is reconciled
  only after teardown, observer/terminal accounting and ledger completion. Its
  phase/ledger/volume helpers are already separate owners; moving finalization
  outside the transaction risks error-order and lifetime drift.
- `classifyWheelSurface` (234 lines): maps final installed destinations before
  classifying hooks, modules, namespaces, native extensions and resources.
  Nested cases encode pinned import/install semantics rather than a vendor or
  filename safety allowlist. Invalid/ambiguous surfaces remain explicit.
- `approvedTerraformProjectGuard.Commit` (214 lines): guarded original and
  selected controls/tree, durable transaction checkpoint, independent approval
  publication and identity-aware rollback. Existing before/after approval
  checks retain ownership/mode/content and foreign recovery members.
- `pypiVenvTransaction.commit` (208 lines): freeze, journal backup/publication,
  publish metadata, reconcile exact ownership and inode identities, then remove
  recovery state. Primary failure is joined with rollback/cleanup failure;
  uncertain recovery is retained. Simplifying this to an unchecked copy/rename
  would lose the required multi-scheme and existing-venv behavior.

`tools/gvisor-observer/observer.cc` is a further hotspot (4,904 lines). It keeps
protocol decoding, per-container process/topology state and profile predicates
in one translation unit. Existing small exact creator/admission/image predicates
and shared state owners avoid a second observer; pinned Bazel schema/latch tests
and actual Python/Go/Cargo/Provider consumers verify its behavior separately.
Growth of these predicates raises review cost, so future changes need precise
owner/negative/actual-consumer evidence. No behavior-preserving wholesale split
is established by this audit, and line count alone does not justify replacing
that completed security boundary.

Separate ecosystem transactions share patterns but differ in target/control,
graph, ownership and retained-approval semantics. A universal mutation engine
is not justified by a confirmed defect here. Existing small neutral types,
anchored input helpers and observed build owners supply the useful reuse.
Long security-critical functions remain review hotspots for changes that touch
their ordering; this review does not promise that their structure needs no
future improvement.

## Design and documentation reconciliation

[Requirement matrix](./requirement-matrix.json) binds all seven required
ecosystems to 11 scenario classes and existing M7/M8/M9/M11 hostile regressions.
Every cited controlled regression is a real passed current-candidate test;
actual installed/public/full consumers and their limitations are recorded
separately in remote qualification. GitHub's standalone asset has no invented
dependency graph, and Terraform complete independent Provider roots are not
presented as primary dependency edges.

Corrected current M3/M4/M11 runtime text to the existing canonical runtime lock,
and corrected the stale CUDA budget table to the values already implemented.
Historical qualification receipts retain their original versions and outcomes.
README/CLI examples are reconciled with implemented operations; first-release
support is bounded by the canonical M12 tuple decision, and public installer/
broader operations remain M13 work. No new release, runtime policy, ecosystem,
retention daemon, dependency or future placeholder is introduced by this review.

The resulting structure continues to serve the project's stated purpose:
inspect external artifacts under bounded isolation, keep authenticated exact
identity and normalized evidence through Policy, and publish only approved
bytes. HAA does not attest every future behavior, GPU computation, external
service RPC or general package safety. Unsupported/incomplete states stay
explicit and cannot become automatic Promotion.
