# SEC-UPD-001 — bounded native PyPI enrollment / 2026-10-19

Baseb82f4e7, after the independent [baseline and second review](./pypi-native-baseline-20261019.md).
Only SEC-UPD-001 is IN_PROGRESS. [Contract06](../../../user-journey-cli-ia/06-update-quarantine.md#native-pypi-ownership-review-before-implementation)
owns supported layouts, limits and implementation responsibilities. Existing
inspection, M5 Policy, staging and venv transaction/rollback are reused.

| Scope | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Explicit public native wheel enrollment in documented site/dist-info subset | YES | YES | YES bounded local and remote | YES bounded subset | None within this subset; named/complete PyPI below |
| New-set full inspection before enrolled replacement | YES | YES | YES bounded local | NO complete PyPI | Named-source work and remaining lane acceptance |
| Named PyTorch updates | NO | NO | NO | NO | Independent source/selection/ownership audit and affected qualification |

## Candidate and checks

Exact production source inventory is local raw pypi-native-candidate-quick-source.json.
Fresh CLI d6ff000bfa4af8affe9d612ecfb527999560dcb4ac5e22d26588817b0c66ef46,
26,958,984B, pinned Go1.26.9/CGO0/trimpath build2.18s. Canonical format2.49s,
Quick92.48s, security6.86s, docs0.28s and targeted Race25.06s PASS with matching
inventories. Quick includes compiler, vet, pinned Staticcheck and default tests.
Only documentation changes follow these checks; the final docs/security checks
are recorded separately. Earlier sandbox socket setsockopt refusal is not a
product failure: identical inventory actual-Host focused tests passed9.89s.

Existing application update finding tests now exercise both M4/npm and M5/PyPI:
new primary and transitive-only findings block; incomplete required inspection
reviews; integrity mismatch blocks. All selected entries reach acquisition,
verification, inspection and Evidence, with no publication or stale version
evidence. Filesystem fixtures cover original drift, forged RECORD ownership,
canonical old wheel/cache roles, no ledger before new approval, partial-publication
rollback to original METADATA/RECORD/cache and actual ledger absence, and recovery
uncertainty refusing retry. These are controlled fixtures, not malicious public
packages or complete native runtime qualification.

The direct partial-ledger red2.43s and exact-owned-METADATA green12.41s establish
that a same-name owned payload cannot substitute for exact METADATA ownership.
No actual data loss was observed or inferred. The existing lesson L02 owns the
causal finding; the raw receipts retain both outcomes.

## Actual installed CLI consumers

Ordinary UID1000 executed eleven operations through the authenticated installed
path with the unchanged canonical helper lifecycle. The fixture's real native pip
installation used the pinned guest and previously inspected packaging24.2 wheel,
offline/default compilation. No target interpreter or package ran on Host.

| Consumer | Actual result |
| --- | --- |
| Ordinary HAA seed install24.2 | COMPLETED/ALLOW, required3/Evidence3, promotion COMPLETED;23.34s |
| Native update without --adopt | UPDATE_ADOPTION_REQUIRED, unchanged;1.35s |
| Forged RECORD claims user data | UPDATE_ORIGINAL_STATE_UNSUPPORTED, unchanged;28.53s |
| Changed old source with internally consistent RECORD | Same refusal, unchanged;28.47s |
| Unsupported native cache header | Same refusal, unchanged;28.85s |
| Invalid new-graph inspection prerequisite | Failed before replacement, unchanged;1.43s |
| Explicit native adoption24.2 to25.0, constraint >=24,<26 | UPDATE COMPLETED/ALLOW, required3/Evidence3, promotion COMPLETED;41.43s |
| Post-adoption managed only-if-needed no-op | Fresh inspection/Evidence and completed promotion;27.77s |
| Exact -U no-op using VIRTUAL_ENV | Fresh inspection/Evidence and completed promotion;26.91s |
| Human-result no-op | Update/ALLOW, completed;26.65s |
| Unsupported eager strategy | Early refusal, unchanged;0.01s |

The authenticated original wheel digest was
09abb1bccd265c01f4a3aa3f7a7db064b36514d2cba19a2f694fe6150451a759.
The adoption Manifest bound four opaque controls and an independently checked
ownership proof:22 canonical wheel files, RECORD, INSTALLER, REQUESTED and16
timestamp-cache removal roles. Control digests and original wheel bytes match.
This proof grants no safety approval to old code or bytecode. The complete new
set received ordinary inspection/M5 ALLOW. Original default caches were removed,
the true old metadata disappeared, the HAA ledger was published only after ALLOW,
and the user file remained unchanged.

## Restoration, cleanup and remaining work

Actual helper wait0/CONFIRMED_STOPPED, observer restoration and client_restore0/
RESTORED were confirmed. Fresh root postcheck verified four original hashes,
owners/modes/single links, CLI symlink, runtime directory state, process/socket/
Docker absence and protected rollback. The installed CLI remains the original
language-enabled version; no new update candidate is permanently activated.

Evidence, Manifest/SBOM, outputs and exact operators were preserved under local raw
update-quarantine-20261018. Completed payload/cache/client/lifecycle and one-use
operators56,370,333 logicalB were deleted after complete process/open-reference
and Docker checks. The cleanup operator itself was separately archived/deleted;
its exact size/digest is in pypi-native-cleanup-operator-removed.json. Active
reusable source/cache/observer, shared SDK/runtime and required rollback remain.
An initial postanalysis glob included the eager-strategy command's deliberately
empty stdout; JSON parsing stopped. Four structured negatives were subsequently
checked explicitly; no original result or raw bytes changed.

Raw owners: pypi-native-candidate-* check/build/lifecycle/postcheck receipts,
pypi-native-candidate-installed summary/operations/Evidence/manifests,
pypi-native-manifest-proof.json, pypi-native-postanalysis.json and
pypi-native-trial-cleanup.json. These qualify only this local source/subset.
Managed checkpointc93ce58 CI331 is separate historical evidence; new native
remote/platform qualification is now recorded below. Complete PyPI/SEC-UPD-001 acceptance
also requires named-source work and later ecosystem lanes. CLI-UX-003 remains
deferred, with no merge, tag, public publication or package-name decision.

## Same-source remote closure

Source2e8844f41d9aadea7c0ad2a0dbd59bdb9b239a8a passed CI332/run39154516210:
Required plus nine prerequisites SUCCESS. Quick, native macOS and Linux identify
checkout23e00971230a19e6d47771cf0a3d53aef36c5369; fetched full tree
0a49b47d29656adccc20d646a055f9fc92d80dd2 equals the source tree. Linux passed45
top-level consumers, CPU PyTorch full288.11s, five helper waits0 and final original0/
CONFIRMED_STOPPED/client_error0/client_restore0. Local raw ci332-fourth-jobs.json,
three decoded job logs and ci332-completed-qualification.json own the evidence.
This closes the bounded native subset's remaining remote gap. It does not qualify
newer named input/extras/version-comparison source or a named update command.
Complete PyPI/SEC-UPD-001 remains IN_PROGRESS with named/later lanes missing.
