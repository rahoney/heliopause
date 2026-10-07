# M12-005 qualification

M12-005 source qualification and feature freeze: COMPLETE. [Result](./result.json)
binds all five audit states to current source `d328756` and the exact evidence.
M12-02 is NOT_STARTED / Ready: Yes; main merge and public release remain later gates.
The final documentation commit has a separate pushed-HEAD CI delivery check.

## Final candidate qualification

[77 scenario rows](./requirement-matrix.json), [current default regression](./current-default-checks.json),
[implementation/design review](./maintainability-review.md), [CLI review](./cli-help-review.json),
[compiled bounds](./compiled-profile-bounds.json), [source identity](./current-source-boundary.json)
and [five successful-root cleanup](./successful-test-cleanup.json) document the completed
acceptance. Default tests contain763 top-level/2217 case passes and87 explicit
environment-gated skips; actual installed and full executions are separate proof.

[Four actual profile scopes](./actual-scope-summary.json) independently qualify
CPU/cu126/cu130/cu132 on the reduced source. Every run has actual10SUCCESS, full
decoded raw bodies, whole checkout/source tree and mode equality,102 latches,
five helper parent waits0 and confirmed final cleanup. CUDA logs separately prove
viewer NOT_ATTESTED and inspection-only NumPy excluded from29 original promotion
entries/site-packages/scripts. [Support tuples](./support-tuples.json) retain those
limitations and exact existing resource bounds. GPU computation/driver/Toolkit
and public Host activation are outside this qualification. Historical bb28/8791
proofs below retain their original scopes and are not substituted for current runs.


## Baseline audit — 2026-10-07 (historical start state)

Baseline: `01175bbe62a2ab8d1b9ee873861839351de1e485`,
`milestone/m12-go-cargo-terraform`. The tracked worktree and index were clean.
User-owned untracked notes are outside the commit candidate.
The actual baseline PR CI [run37510273661](https://github.com/rahoney/heliopause/actions/runs/37510273661)
was independently queried as completed/success before work started.

| Item | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | Existing implementation / evidence |
| --- | --- | --- | --- | --- | --- |
| M12-001 | YES | YES | YES | YES | `artifact/pypi` source/resource policy, existing pip workflow; [closure](../m12-001-closure/result.json) |
| M12-002 | YES | YES | YES | YES | Go parser/client/SumDB, generic project inspection/build, approved cache/transaction; [qualification](../m12-002-go-build-qualification/result.json) |
| M12-003 | YES | YES | YES | YES | Cargo parser/client, private resolver, approved vendor cache/add/observed build; [qualification](../m12-003-cargo-build-qualification/result.json) |
| M12-004 | YES | YES | YES | YES | Provider discovery/signature/archive/probe and approved init transaction; [qualification](../m12-004-provider-qualification/result.json) |
| M12-005 | YES | YES | NO | NO | Existing CLI/bootstrap/application/ports, all ten Required CI jobs and bounded CUDA dispatch inputs; cumulative [remote qualification](../m12-002-004-remote-ci/result.json) |

The YES entries describe reusable implementation and the work-item acceptance
already established. They do not claim that older CUDA execution proves the
current candidate or that a registered profile is release-supported.

MISSING at start (all now closed by the final result above):

- Requirement-to-implementation/test/evidence mapping for every required ecosystem
  and hostile regression from M7/M8/M9/M11 on the final candidate.
- Current-candidate qualification boundaries for CPU/cu126/cu130/cu132 and the
  explicit first-release support tuple, resource bounds and limitations.
- Review of leftover code, module responsibilities, error/cleanup ownership,
  duplication and excessive or missing abstraction; correct established defects
  through existing owners rather than replace completed implementation.
- Reconcile current user-facing CLI/docs and canonical design with actual support.
- Remove unnecessary completed test data after preserving evidence and checking
  exact ownership, successful outcome and absence of active consumers.
- Final candidate canonical checks, pushed HEAD Required CI, feature-freeze
  acceptance closure and M12-02 handoff.

Read authority: [M12 contract](../../16-m12-01-ecosystem-expansion-contract.md),
[Milestones](../../01-milestones.md),
[Queue](../../02-current-work-queue.md), existing Architecture and quality profiles.
No architecture, Policy, support boundary, resource limit or Required job is
weakened by this audit. Main merge and production publication remain separate.

## Confirmed cleanup before source change

Tracked Go source search and current bootstrap wiring show no production consumer
of `application.GoModuleResolutionService`, `application.CargoResolutionService`
or `hosttool.Executor.RunGo/RunCargo/RunTerraform`. The Go wrapper has one obsolete
stub test; the guarded get failure test remains. `hosttool.Config.GoPath` only
registers the unused Host Go lane; no tracked configuration or current installed
configuration uses it. Remove these leftovers and their test-only legacy methods.
Existing isolated Go/Cargo runners, `GoBuildRunner`, provider probe, complete graph
inspection and approved transaction/build services remain the implementation.
This removes unwired initial scaffolding, not an implemented supported operation.
Old trusted configuration with `go_path` will be rejected by the existing strict
config parser; current install configuration does not contain that unused key.

Canonical baseline quick (including vet/Staticcheck/architecture/default tests),
docs and security completed with actual exit0 on the exact baseline native clone.
Staticcheck does not prove that exported but unwired APIs have consumers. Manual
reference/wiring review is the evidence for this cleanup; no Host exploit claim
is made. Final checks and actual installed Host/isolated ecosystem consumers must
qualify the reduced candidate.

## Confirmed npm inspect acceptance gap

The public npm registry returned HTTP200 and the requested
`application/vnd.npm.install-v1+json` representation. Replaying those exact
9,995 metadata bytes through the existing resolver failed with
`npm metadata response is not JSON`. Its prefix-only check rejected the
negotiated representation and also accepted invalid JSON media-type prefixes.
The existing owner now parses one unambiguous MIME header and accepts only that
negotiated type or `application/json`. The request, authority, bounds, identity,
SRI verification and downstream inspection/Policy remain unchanged.

[Same-input failure/correction evidence](./npm-media-type/result.json) includes
the official format reference, frozen capture and normal/negative regression.
A public `is-number@7.0.0` inspect CLI test is wired to the existing authenticated
Linux lifecycle job. Its actual 5.84s execution on `8791fdc` passed in
[run37556677169](https://github.com/rahoney/heliopause/actions/runs/37556677169):
COMPLETED/ALLOW, all three required verification/static/dynamic checks completed
and all three Evidence references retained. npm install uses a different resolver
and has its own actual promotion test in that same run. Final closure HEAD CI
is separately required.

## Additional zero-consumer cleanup

The read-only AST/reference and actual bootstrap/factory review found four
additional unused API groups, with no production or test callers:
`TerraformResolutionService`, string-only `AggregateBoundedRequirement`,
`RegistryEndpoint`, and executor-less `ProbePython`. The current guarded init,
typed aggregate retaining extras, endpoint constant and configured probe are
unchanged. [Exact before/after scope](./additional-code-cleanup-boundary.json)
records the 40-line removal. The M5 contract now names the actual configured
helper; capability/identity/Policy/resource rules are unchanged. Earlier005 remote
runs remain historical qualification; the new source has independent
CPU/cu126/cu130/cu132 and Required results linked in the final scope above.

Current-source ordinary run37561282942 also independently passed the public npm
inspect gate in5.65s: COMPLETED/ALLOW,3 required checks and3 Evidence records.
Its CPU full took291.53s; source tree/mode equality and actual45 consumers were
verified. Final document-closure HEAD CI remains a separate delivery result.
