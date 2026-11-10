# SEC-UPD-001 — final cross-lane review / 2026-10-19

Status: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES.
MISSING: none within the reviewed bounded update-command coverage. Final
sourcecdac96d/CI344 closes the real-M6 test/document source boundary as well as
the earlier per-lane actual/negative/restoration/deletion evidence. Historical
pending text below is superseded by the final closure. CLI-UX-003 is next/ready;
its implementation remains NOT_STARTED.

The [canonical contract](../../../user-journey-cli-ia/06-update-quarantine.md)
owns command semantics; this leaf owns only the cross-lane completion review.
Each linked qualification owns its exact source, historical failures, inputs,
local/remote scope and raw references. Raw remains local/untracked.

| Bounded lane | Qualification owner / supported limit |
| --- | --- |
| npm | [npm](./npm-qualification-20261018.md); zero/one selector, one direct root plus supported transitive graph, original ranges; bounded native --adopt |
| Public PyPI | [managed](./pypi-managed-qualification-20261019.md), [native](./pypi-native-qualification-20261019.md); one named upgrade, only-if-needed, proven venv ownership and retained constraints |
| Named PyTorch | [CPU/cu126/cu130/cu132](./pytorch-update-qualification-20261019.md); exact named managed tuple/extras; broader native named footprints remain unsupported |
| Cargo | [Cargo](./cargo-update-qualification-20261019.md); complete lock-only workspace selection, preserved manifest/features, optional one selector and explicit registration |
| Go | [Go](./go-update-qualification-20261019.md); one existing require selected by get -u, full graph inspection; old-version metadata authenticated only when actually required by valid selected controls |
| Terraform | [Terraform](./terraform-update-qualification-20261019.md); latest stable constrained public providers across reachable local modules; installation-only, no plan/apply/cloud |
| GitHub | [GitHub](./github-replacement-qualification-20261019.md); same repository/exact tag+asset, explicit target/replace, independent prior approval; opaque archive or isolated supported ELF |

## Acceptance responsibility reconciliation

| Required behavior | Concrete existing evidence owner |
| --- | --- |
| New primary/transitive bytes receive independent acquire/verify/inspect/Evidence; unsafe/incomplete/mismatch never stages or promotes | [Install-update tests](../../../../internal/application/install_update_test.go) npm/PyPI and final real M6 v1→v2 cases; [Cargo](../../../../internal/application/project_update_test.go), [Go](../../../../internal/application/gomodule_upgrade_test.go), [Terraform](../../../../internal/application/terraform_upgrade_test.go) complete-set adverse tests. M6 has one asset and no transitive graph |
| Original controls/ownership held before selection; exact selected graph/bytes through approval | Each lane's resolver/session tests and actual Manifest/Evidence/installed digest comparison in its qualification; no second live resolve after ALLOW |
| Old success does not authorize a new version or fresh no-op | Independent new Run/Evidence in the actual version-update/no-op consumers; final GitHub17 and shared-consumer32 JSON-success records are disjoint and reconciled |
| Native state cannot authorize silent adoption; foreign/shared files remain protected | npm explicit native inventory, public PyPI independently authenticated old wheel/RECORD footprint, Cargo/Go/TF independent approval/cache tests; receiptless GitHub rejects even with explicit replacement |
| Substituted/staged/corrupt/concurrent state refuses | Per-lane held transaction tests and actual unchanged-target negatives; [three physical lock-owner matrix](../../../../internal/promotion/mutation_lock_identity_test.go) covers rewrite/unlink/recreate/mode/link/mtime and recovery FD close |
| Commit fault restores proven originals; uncertain foreign state survives and prevents success/retry | Existing npm/PyPI transaction rollback, Cargo/Go/TF upgrade faults and GitHub six-phase rollback/foreign selected/published/backup tests; late close failures remain failures |
| Human/JSON facts and static unsupported intent agree | CLI/report tests and actual EN/KO/JSON consumers; seven-language core help; unsupported arguments refuse before composition |
| Real runtime/original restoration and deletion | Per-lane canonical helper wait0/CONFIRMED_STOPPED, original identity/root postguards, preserved proof/inventory and completed payload/cache/operator deletion receipts |

## Final review findings and verification boundary

The GitHub lane revealed a directly reproduced lock identity gap in all three
physical guard families. Its [pre-edit correction](./transaction-lock-identity-correction-20261019.md)
extends the original exclusive FD lifetime; controlled matrix, full related Race
and small actual consumers cover the correction. A distinct selected-path chmod
failure is corrected using the original created FD. Existing resolver/inspection/
Policy/observer/runtime/bounds remain intact; full CUDA source-specific results
retain their original scope rather than becoming new final-source runs.

The final review also identified that generic npm/PyPI adverse-update tests did
not directly exercise M6's real entry/set policies. Pre-test plan5eb90f7 precedes
test commit94cc284. Allowed GitHub v1 then unsafe/incomplete/mismatched same-name
v2 now verifies fresh subject/digest/Run/Evidence and no publication. Same-input
Host Quick116.81s/security4.51s/docs0.27s/full Application Race3.42s pass.
The restricted-socket failure and overlapping preparation capture remain separate.

All non-test production/dependency/runtime/observer/CI bytes are identical to the
cb1f944 actual-qualified candidate. No new installed/GPU runs are claimed for
test/document changes. CI343 cannot qualify the added test commit; that source
requires its own ordinary CI. Main merge, permanent local activation, launcher,
protected tag, registry names and publication remain separate actions.

CI343 now closes the cb1 product boundary: ten completed/SUCCESS jobs, exact
whole checkout tree/native macOS, Linux45 verbose consumers/36 related/56 with
subtests, CPU267.82s, five helperwait0 and final confirmed cleanup. The
[GitHub qualification](./github-replacement-qualification-20261019.md#production-source-ci343-closure)
owns raw/tree/consumer details. Final test94cc and documentation await their own
ordinary CI; production identity alone does not qualify newly added tests.

## Aggregate acceptance closure

Final [CI344 receipt](./ci344-closure.json) records ten completed/SUCCESS jobs,
exact checkout tree/parents, native macOS Go1.26.9 production/default tests,
Linux45 verbose consumers/36 related/56 with subtests, CPU291.84s, five helper
waits0 and final confirmed cleanup. It qualifies the added real-M6 regression
without borrowing CI343's older test inputs. Actual archive/ELF and small prior
consumer evidence is reused only for identical production bytes; named CPU/CUDA
results retain their exact per-source scope. No broader source, native footprint,
CLI compatibility or transparent interception is added by this closure.

All local update trial/cache/candidate/operators and the final common source/cache/
observer/checker/restored binary backups are removed after evidence preservation.
Shared SDK/tool cache and small raw/normalized proofs remain. The postqualification
change is documentation/evidence only; delivery CI is not a new implementation or
another required functional/GPU requalification. PR38 remains draft/unmerged;
permanent local activation, package naming and publication remain separate.
