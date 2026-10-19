# SEC-UPD-001 — required Go module metadata correction / 2026-10-19

Status: cause confirmed; bounded additive correction and second review committed
in84a2dc0 before source. Implementation is wired with focused source/proof/cache/
consumer/final canonical and actual installed checks PASS; new-source CI340
failed the existing grpc transitive get consumer. The graph-pruning amendment
below must precede its narrow source correction. The independent audit below
preserves the pre-correction findings.
Only SEC-UPD-001 is IN_PROGRESS. Go acceptance remains open.
The earlier [Go audit](./go-update-baseline-20261019.md) and sourceb0a2ccb retain
their actual scope. This document supplements that edit map after a new actual
consumer exposed a missing representation. It does not replace implemented
selection, verification, cache, transaction or build responsibilities.

## Observed failure and causal boundary

CLI832d781/sourceb0a2ccb completed public Cobra1.9.1→1.10.2 and imported
pflag1.0.6→1.0.10, retaining unrelated uuid1.5.0. Update22.26s, fresh no-op22.12s
and changed-control explicit enrollment/Korean21.70s each inspected8 modules with
required16 checks;48 update Evidence handles are disjoint. Exact seed get and
retained download passed. The subsequent observed build failed before compiler
execution at OFFLINE_GRAPH/EXIT_STATUS_1. Later foreign-marker/corrupt-state actual
cases were NOT_REACHED, not successes. First installed campaign is FAILED145.0s.

Canonical helper2371841 wait0/CONFIRMED_STOPPED, original CLI/observer restoration
and fresh root four/process/socket/Docker postguard all passed. Completed lifecycle,
temporary operators and CLI832 candidate53,815,094B were deleted after evidence
preservation. The necessary project/approved cache remain an active reproduction;
reusable tooling/SDK/observer and protected original rollback remain separate.

Raw go-update-offline-diagnostic uses the exact locked Go1.26.9, no network,
metadata commands only and a disposable exact cache copy. Both download-all and
graph exit1: Cobra1.10.2 requires pflag1.0.9, whose go.mod is absent with GOPROXY=off.
The selected cache correctly contains pflag1.0.10. It lacks the lower-version
control needed to reconstruct MVS; the selected source archive itself is present.
MaterializeModule has only selected .mod/.zip/.ziphash/.info and extracted source.
That owner is unchanged from the qualified baseline; this observation does not
identify the historical first introduction or invalidate its earlier scopes.

A separate same-content diagnostic (download-all0.51s/graph0.49s, both exit0) puts the selected pflag control under the
required1.0.9 filename only in a disposable copy after declared h1 agreement. This
is causal isolation, never product approval: same bytes or a go.sum declaration
do not prove the old version's SumDB identity. Original cache remains unchanged;
diagnostic containers/copies are removed. Its final outcomes belong to
go-update-offline-diagnostic-same-content. The first derived operator stopped at
an exclusive existing raw-path assertion before execution; that preparation error
is preserved separately and did not alter product state.

## Independent responsibility audit

| Responsibility | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING / existing owner |
| --- | --- | --- | --- | --- | --- |
| Native named -u, unique original membership, frozen selected primary/full graph/controls | YES | YES | Local new selection scope only | NO | Existing Go resolver/upgrade service; preserve; complete new-source acceptance after correction |
| Fresh selected-module public acquisition, independent SumDB/static, Policy/Evidence and approved cache | YES | YES | Prior scope and actual new selected bytes | NO for new MVS build scope | Existing client/intake/verifier/inspector/cache; preserve current selected bytes and source authority |
| Derive exact lower-version go.mod references from the frozen graph and bind declared h1 | NO | NO | NO | NO | Adapter normalization must identify only required unselected versions, not every stale go.sum line |
| Bounded transport/acquisition and independent proof of those required controls | NO | NO | NO | NO | Add optional per-selected-module context to its existing acquired subject; no invented selected dependency/primary or metadata-only ALLOW shortcut |
| Approved required controls in read-only cache/receipt/build archive | NO | NO | NO | NO | Extend existing MaterializeModule/cache accounting in place; retain exact inventory/Evidence/guard and fixed resource bounds |
| Offline graph reconciliation, observed compilation and output publication | YES | YES | Existing scope only; new MVS consumer fails | NO for this lane | Existing checkGraph/GoBuildRunner/observed builder remain unchanged; required metadata must make the same consumer work |

## Additive design and exact edit map

The canonical technical addition is in the M12 Go intake/cache contract. Preserve
HAA-GO-MODULE-1 for subjects without required metadata. An optional version2
envelope binds same-module lower-version go.mod bytes to the selected module's
acquired digest. Freeze those references from raw native graph vertices and the
exact guarded selected go.sum, then independently authenticate each old-version
record with the existing pinned SumDB client. Metadata is graph input only; it
does not select/install/extract another old archive or invent another primary.

| Owner | In-place change | Required evidence |
| --- | --- | --- |
| artifact/gomodule project.go/module.go and a focused required-mod helper/test file | Derive per-selected-path lower versions from the already validated raw graph, require one canonical matching go.sum /go.mod h1, sort/deduplicate/bound refs and append the canonical optional required-mods declaration. BuildProjectSnapshot uses it; actual BuildLockedGraph calls receive the same selected go.sum so requested subset matches complete snapshot. Preserve old current-pair forms when no metadata is needed | Direct RED showing existing MVS snapshot drops required metadata; selected/unselected/unused sums, missing/duplicate/malformed/oversize/context/source/version swaps; no fake graph nodes |
| sandbox/gomodule_resolver.go call site | Supply selected sums to the existing requested graph constructor; keep three fixed SDK commands, original controls/cleanup and all source/role/budget gates | Exact get/download/-u and cleanup/race regressions; no second selection |
| artifact/gomodule/intake.go | Add immutable optional required-control bodies and backward-compatible bounded envelope parsing/encoding. Keep current Mod/Zip semantics and envelope digest. Optional references use the same typed acquisition identity and exact byte binding | HAA1 unchanged, bounded/canonical HAA2 round trip; unknown/duplicate/trailing/empty/missing/extra metadata, version/path/hash/size/count attacks |
| artifact/gomodule/client.go | Fetch only frozen exact same-path .mod endpoints for declared lower versions. Do not copy resolver/global cache or discover versions from network/error text. Count16 and aggregate4MiB controls, same ZIP64MiB; complete metadata-bearing acquisition capped by the existing two60s request ceiling120s | Canonical endpoint/escaped identity, no redirect/private fallback, context/cancellation, aggregate/count refusal, error cleanup and actual source capture |
| verification/gomodule/integrity.go | Under the same60s/64-requests/1MiB-response/4MiB-total SumDB limits, verify each required /go.mod h1 against its own authenticated exact old-version record, plus current .mod/.zip. Same-content current proof cannot replace old-version binding | Signed fixtures with same/different old bodies, missing/wrong/outside-authenticated-record old sums, deadline/bounds and unchanged old proof negatives |
| inspection/gomodule/static.go | Inspect each bound required module control's official grammar and exact module path alongside current module/archive checks. Keep all non-executing source/ZIP checks | malformed/foreign required control refusal, old static source cases retained |
| artifact/gomodule/cache.go; promotion/go_cache.go accounting | Rehash full acquired envelope, materialize only the required lower .mod files in addition to existing selected payload. Verify body/identity/declaration and reject collisions; include them in the existing total file/byte inventory and receipt. Keep512MiB/20k-files/40k-entries/4MiB-receipt and existing archive/rollback owners | No lower ZIP/extracted source, old HAA1 inventory unchanged, metadata tamper/missing/extra/source swap and cache evidence negatives; old build cache copies exact approved inventory |
| existing Go observed builder/checkGraph/build source/cache owners | No implementation replacement, graph skip, network access or mutable cache permission. The existing fixed offline download-all/graph and compiler must consume the newly authenticated cache | Actual same-case new update/no-op/download/build and old get/download/build; immutable cache/source, original restoration and cleanup |

No Core/Domain representation, Application workflow/Policy, requested-primary
contract, observer, runtime identity, budgets, dependency pin or CI job is added
or rewritten for this correction. Existing snapshot graph digest already binds
raw graph and selected go.sum/control hashes; the optional declarations are a
deterministic projection of those frozen inputs. Required context participates
in the acquired subject digest and entry Evidence. Project module counts remain
the actual selected module set; no metadata artifact is counted as installed.

## Second review and execution order

Review against exact failure and current owners rejects these false fixes:
pin pflag back to avoid native -u, change approved .mod content, trust old go.sum,
alias same-content metadata in the production cache without its own old-version
proof, copy a resolver cache, add build networking or skip offline graph checks.
Each breaks a selection/source/cache/build invariant. Additive version2 retains
the existing version1 path and all authentication/Policy/transaction owners.

Rechecked metadata completeness: use both raw graph endpoints, include every
nonselected public version only under its actual selected path, ignore synthetic
go/toolchain vertices only after existing graph validation, and exclude unrelated
old sum lines. Require canonical version/hash/order/count and exact envelope
membership. Second review also checked the existing1024-byte Domain integrity
limit: keep it unchanged and refuse long-version declarations that exceed it;
never silently omit required metadata or increase the Core limit. Use the same helper for requested and complete project forms, and
for offline reconstructed snapshots. Proof, syntax, bytes, file inventory and
published cache must bind the same metadata; old header acceptance alone grants
no new authority. Missing or over-limit context refuses before publication.

Next: commit this reviewed follow-up before source, reproduce the missing
declaration/cache behavior with a focused RED, then implement only these gaps.
Run focused source/proof/cache/build negatives, canonical format/Quick/security/
docs and relevant Race on the final candidate. Re-enroll/reinspect the active
managed fixture with fresh Evidence under the corrected source, then verify the
same offline consumer and a fresh bounded native scenario; finish the previously
NOT_REACHED refusals. Preserve failed originals, confirm canonical stop/restore/
fresh postguard, delete completed reproduction payloads/tools, then new-source
Required/native macOS/actual old consumers before closing Go. CI339 remains
sourceb0a2ccb's separate legacy scope and cannot qualify this later correction.


## Implementation checkpoint — final qualification pending

The optional context is now wired through frozen requested/complete declarations,
exact public acquisition, HAA1/HAA2 intake, independent old-version proof, static
control inspection and approved cache/inventory. No Core/Domain or observed builder
source change was needed. Existing build archive consumes its existing receipt.
The sandbox's old MVS fixture now includes the lower control sum that real native
Go emitted; its previous omission cannot stand in for complete metadata context.

Local raw go-mvs-declaration-red reproduces the missing lower control declaration
in0.82s. First focused23.17s passes the new frozen declaration and old consumers.
Proof-focused27.57s includes independent signed old-version records, same-content
current-proof refusal, envelope/source/cache binding and malicious framing/bounds.
The first expanded consumer run fails2.10s only because a new test helper's byte
control variable collides with its existing control-digest variable; production
and other package checks pass. Rename the test variable, retain the failed raw,
then final focused22.13s passes required metadata, static foreign/malformed controls,
sealed cache missing/tamper refusal, requested/complete declaration equality,
missing/duplicate/unused sums and existing Go/Application/Cargo-report consumers.
Canonical pre-Quick security passes8.58s. These local fixtures do not authenticate
actual public update payloads or qualify the previously failed installed build.

Earlier CI339/run39220147935 is now all10 SUCCESS for sourceb0a2ccb only. Its actual
checkout3af740b treea3d602660a4fa893287d7dc73559b9986d01f9c8 equals sourceb0's whole
tree;36 top-level related Go/Cargo/Terraform consumers (56 including subtests),
five helper wait0 and final confirmed cleanup pass in captured decoded logs.
That prior scope cannot replace a new-source CI or the corrected actual consumer.

Completed seed/orphaned cache/intake payload21,815,379B were additionally deleted.
Native project, current cache run_bl7armqchbxg3n5y5wxtl6tiey, exact native approval
and recorded Evidence remain as the necessary active reproduction. The native
state SHA256849ab01028231e2685068d5333afb96f27e26d0593ccc1e89f2f02d81e0486d8 is
unchanged. All completed local diagnostic/cleanup operators are deleted; their
frozen captures and shared/protected resources remain. Final correction source,
actual successful replay, whole-fixture cleanup and remote acceptance remain open.


Final local candidate: canonical go-mvs-verified Quick128.83s, security8.96s,
docs0.41s and related Race79.42s all pass on identical full input inventories.
The earlier go-mvs-final-quick FAIL17.07s stops at pinned Staticcheck U1000 for the
now-unused legacy encodeBundle wrapper. Explicitly route context-free acquisition
through that existing wrapper and context-bearing acquisition through version2;
no suppression or deleted legacy behavior. The corrected final runs include the
same source and preserve the original failure. CGO0/Go1.26.9 CLI build4.25s is
27337786B, SHA256b18b24ff2f75a47bf82155c9139e12ba978380ddf2cdc495d12e127c8fc37d7f.
Its frozen runtime inputs match those checks. Actual same-case replay will use
current-project download to refresh authenticated metadata without reselecting
versions, then the unchanged offline observed build. A fresh new-source update/
no-op/enrollment/refusal campaign and new-source Required remain necessary.


Actual source394/CLIb18 same-case replay and fresh full campaign now pass15 phases
with64 new Evidence, successful unchanged offline observed builds, canonical helper
wait0/original restoration/fresh postguards and completed trial/operator cleanup.
The [qualification](./go-update-qualification-20261019.md#corrected-source394--local-actual-qualification-complete)
owns final timings, file receipts, source binding and cleanup. Original b0 failure,
networkless diagnostic and all intermediate failures retain their historical scope.
CI340 failed; no earlier source's CI substitutes for it.

## CI340 graph-pruning amendment — reviewed before source

New-source run39227724529 has8 successful prerequisites, failed Linux integration
and failed Required. Actual checkoutafda1ea has the same whole tree611119f as394.
TestLinuxGoTransitiveGetDownloadIntegration fails32.01s at exact grpc1.76.0 get:
required Go control checksum is missing or invalid. Fifteen related passed test
rows retain their scope; later old builds/CPU are NOT_REACHED. Three launched
helpers have wait0; final CONFIRMED_STOPPED/client_error0/client_restore0/RESTORED
preserves primary exit1. Raw go-ci340-final-jobs/failure-scope and three logs own
the actual results. Completion was briefly misreported as success in commentary;
the actual conclusions are FAILURE and were immediately corrected.

Independent pinned Go1.26.9 private grpc1.76.0 get3.78s/download3.27s/graph0.02s
selects the same41 modules. Its raw graph mentions older protobuf1.5.0 and
go-cmp0.5.5/0.5.9, but all three have neither go.sum /go.mod entries nor cache
.mod files. The same cache with GOPROXY=off/GOSUMDB=off independently passes
download-all0.02s and graph0.02s without those files or any control change.
These are unexpanded requirements under [Go graph pruning](https://go.dev/ref/mod#graph-pruning),
not missing authenticated metadata. Selected ZIP/control pairs remain mandatory.
Pinned SDK cmd/go/internal/modload/buildlist.go readModGraph/enqueue distinguishes
mentioned requirements from actually loaded go.mod files. Native diagnostics
are semantics evidence, not installed gVisor acceptance or SumDB attestation.
Raw go-required-graph-diagnostic owns all commands/controls/graph/cache facts.

The394 projectModuleIntegrities incorrectly equates every lower graph vertex
with a loaded metadata requirement. Existing representation/acquisition/old-version
proof/static/cache/observed build are IMPLEMENTED=YES/WIRED=YES; new pruning
selection QUALIFIED=NO/ACCEPTANCE_CLOSED=NO. MISSING only correct graph-and-sum
membership, direct regression, canonical checks, same actual legacy grpc consumer,
Cobra metadata build/upgrade retention and fresh Required/cleanup.

Exact edit: artifact/gomodule/required_mod.go projectModuleIntegrities includes
only the intersection of lower same-path frozen graph vertices and valid unique
selected go.sum /go.mod declarations. ValidateProjectSums still validates every
line, duplicate and selected content pair first. Missing pruned declarations do
not become acquired subjects or forged checks. Apply the existing16-count and
1024-byte limits to included refs; ignore absent pruned refs before count, while
retaining graph identity/normalization validation. No resolver/cache input or
failure-message-driven fetch, no format/proof/cache/build/Domain/runtime change.

Regression: direct RED for a graph-mentioned lower version absent from sums,
then GREEN with an empty optional context; keep lower-present-context binding,
duplicate/malformed/selected-pair missing negatives, unused sum exclusion,
independent old-version proof/missing/tampered cache tests. A test that required
an error for every absent lower sum encoded the same incorrect assumption;
replace that single expectation with pruning coverage rather than weakening
actual source/proof tests. Include a many-pruned-ref case to avoid counting
unacquired references against16. Preserve exact snapshot/requested equality.

Second review: graph membership alone and go.sum history alone are each
insufficient; only their intersection is frozen acquisition context. Absence
is not verification success. Every included old control still has its independent
version-specific SumDB proof; the selected pair remains strict. Genuine cache
incompleteness still fails the unchanged offline download/graph pre/post build.
The previous Cobra required pflag1.0.9 case must remain captured and pass actual
build, while grpc pruning must restore the old actual consumer. Do not change
existing positive CI fixtures, skip them, raise limits or retry the unchanged
failed source. Terraform stays plan-only until Go acceptance closes.
