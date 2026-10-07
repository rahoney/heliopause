# M12-005 qualification

## Baseline audit — 2026-10-07

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

MISSING:

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
