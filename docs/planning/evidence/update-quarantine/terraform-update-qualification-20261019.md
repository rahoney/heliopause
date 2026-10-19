# SEC-UPD-001 — Terraform project upgrade qualification / 2026-10-19

Status: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES.
Only SEC-UPD-001 is IN_PROGRESS. The [pre-source audit/edit map/second review](./terraform-update-baseline-20261019.md)
was committed170140c; Go closure/activation was committed2fcd5d0 before Terraform
production edits. Existing exact selection, signer verification, archive/probe,
Evidence/Policy/cache and transaction are extended in place.
MISSING: none for this bounded Terraform lane. Local actual acceptance and
same-source CI342 are complete; historical pending descriptions below retain
their original scope and are superseded by the final evidence.
The early local tests use controlled synthetic fixtures and are separate from
the actual public artifacts below. No permanent installation/main merge/release.

## Wired bounded behavior

The current-project route selects every required independent public provider at
its greatest stable version satisfying every frozen constraint. It reuses one
validated discovery/version/package snapshot per provider, never chooses an
arbitrary primary and never reselects after approval. New source uses the existing
snapshot-only constructor/ProjectLockUpdateResolver and complete guarded workflow.
The old exact primary route and all old verification/probe/cache limits remain.
Newest unsupported platform/protocol refuses instead of choosing an older version.

BeginUpgrade intent is request-local. Providerless native or deliberately edited
controls require --adopt; old canonical approval/marker/cache/Evidence and complete
installed tree are still checked independently. Unmanaged installation, corrupt/
aliased authority and recovery refuse even with --adopt. Upgrade alone retains
identical lock inode/mode/mtime, preserves ordinary changed-lock permissions and
creates absent lock0600. All original, publication, rollback and verified-backup
cleanup paths guard retained controls; uncertain foreign state is preserved.

The exact standalone native -upgrade token is normalized during static preflight;
values/false/duplicates/extra positionals and unsupported native flags refuse
before runtime composition. Existing init reference behavior stays strict.
Application/bootstrap reuse the same actual acquisition/verifier/inspector/cache/
promoter composition. Human/JSON update reports share the existing schema and
fresh facts, have provider-specific labels/scope and never convert late failure
into success. English/Korean and seven-language core help use existing catalogs;
Terraform flags describe whole constrained selection, not pip only-if-needed.

## Current local captures

Raw owner: docs/github-logs/update-quarantine-20261018/. It is local/untracked and
not claimed remotely available. Existing old init evidence remains separately
qualified for its own sources; it does not qualify this new upgrade route.

| Capture | Actual result and scope |
| --- | --- |
| terraform-upgrade-admission-red | FAIL1.33s; native -upgrade was rejected as unknown shorthand before implementation |
| terraform-upgrade-lock-red | exit0/NO_TESTS_TO_RUN; NOT_VALID as a regression. Patch inserted a test function into an existing raw HCL fixture; restore fixture and move function to real Go file scope. No product success inferred |
| terraform-upgrade-lock-actual-red | FAIL3.18s; actual test executes and observes identical upgrade lock replacement/mode/mtime loss before retention branch |
| terraform-upgrade-first-green | PASS5.05s; actual admission/retention and existing exact initial/retained six-phase rollback, exact metadata/acquisition consumers |
| terraform-upgrade-selection-green | FAIL6.66s; new positive test list marshals existing decode-only Platform DTO with uppercase OS/Arch. Strict existing lowercase JSON validation correctly refuses; fixture wire keys are corrected without relaxing the production decoder |
| terraform-upgrade-wire-green | PASS2.83s; stable numeric ordering/all constraints, no platform/protocol downgrade, duplicate/malformed/prerelease refusal, cancellation, one frozen version list and complete two independent providers with no post-selection live reads; original target unchanged |
| terraform-upgrade-workflow-green | PASS3.95s; Application approval/close boundaries and two-root unsafe block/review/incomplete/integrity refusal, CLI facts/intent and old Go/Cargo report consumers. Bootstrap only compiles in this filtered capture; its zero-test output is not runtime qualification |
| terraform-upgrade-adverse-green | PASS7.94s; twelve enrollment/old-proof negatives/positives and twelve retained/changed-lock normal/fault cases, seven-language help, report and shared workflow consumers. Later foreign content/mode/timestamp/backup cases are separate pending checks |
| terraform-upgrade-final-format | PASS2.81s; explicit canonical formatter on the complete current source projection |

The implementation follows the canonical [Terraform upgrade scope](../../../user-journey-cli-ia/06-update-quarantine.md#terraform-project-upgrade--reviewed-initial-semantics).
Final canonical Quick153.85s/security9.16s/docs0.33s and related Race38.84s pass
on identical complete input inventories. A separate same-input verifier Race2.12s
executes all five signer-proof test groups: official/partner positive, tampering/
ambiguity, community trust refusal, historical/current authority and expired/
revoked current keys. The shared Race filter selected only the optional integration
test in that package; it is not substituted for these actual unit executions.
Retained foreign content/mode/mtime and backup member preservation/recovery refusal,
old proof/enrollment and actual bootstrap static help test groups all execute and
PASS. Default canonical checks include the full old consumers' unit coverage.

Manual review after tools confirms request-only snapshot admission, independent
old authority checks even with --adopt, same-version lock agreement, exact frozen
package reuse and selected-private/retained/publication/rollback/cleanup identity
checks. Changed-lock permissions are applied to the newly created exclusive file
descriptor before sync/close, never by chmod on an unproven path. The old normal
init constructor/response/admission and existing Go/Cargo report branches stay.
No dependency, Core/Domain, runtime/observer/budget, signing policy or CI-job change.

The same complete input projection builds Go1.26.9/CGO0 CLI3.98s,27401540B,
SHA256e7cf7006af252ae2c243eea85357621eb3e0080ec4e565bd4bcae4c31028fbcc.
Actual candidate native -upgrade --help succeeds in all seven languages with no
runtime composition or wrong pip strategy text. Public metadata availability for
random3.7.1/3.7.2 and partner github6.6.0/6.7.0 was captured separately; HTTP
metadata presence is fixture preparation, not signer/artifact verification, and
its completed temporary operator was archived/deleted. Installed source evidence
is recorded below after actual execution. GitHub replacement and launcher remain pending.

## Installed sourcef882921 qualification

Sourcef882921f8962d808969d5e8da80fb9a800bf4bfb/tree94d973d44ab6dd5040d988967e657f17beafcc77
contains exactly28 changed paths; non-Markdown committed bytes match the final
complete checked inputs and candidate above. Normal draft PR38 source push starts
CI342/run39244171743. Documentation-only follow-up is held while that CI runs.

The first actual random3.7.1 exact init succeeds7.88s/Providers1 with all three
required Evidence. The operator then wrongly compares approval.files, including
root/subdirectory records, to a file-only actual list. This is an operator failure,
not a product failure; upgrade is NOT_REACHED in that capture. Preserve original
terraform-upgrade-installed/result and lifecycle14.00s/exit1, canonical helper
2481696wait0/CONFIRMED_STOPPED, original restoration and fresh root postguard PASS.
The corrected whole membership/kind/mode/file size/hash validator passes on the
actual seed and rejects a deliberate changed digest before reuse. Product and
all limits remain unchanged; exact operators and correction receipt are archived.

The resume reuses that seed and completes fifteen more actual calls133.35s, for
sixteen distinct phases overall. It independently authenticates official random
3.7.1/3.7.2 and partner github6.6.0/6.7.0 with the production signer/static/probe
pipeline; metadata availability is not used as signer authority.

| Actual consumer | Result |
| --- | --- |
| partner seed6.6.0 / complete old two-provider init | PASS9.79s /14.28s; both old versions installed |
| managed native -upgrade | PASS17.02s; random3.7.2 and github6.7.0, ordinary lock0640 retained, configuration unchanged |
| managed --upgrade no-op | PASS14.49s; all checks/Evidence fresh, both controls bytes/inode/mode/mtime unchanged |
| providerless native explicit --adopt | PASS14.55s; both providers freshly inspected and installed |
| edited managed controls, Korean --adopt | PASS14.57s; translated provider result/scope and fresh Evidence |
| native -upgrade=false; implicit native/edited controls; no matching version; foreign marker/backend; corrupt approval even with --adopt | All refuse; entire project unchanged in archived assertions and six before/after inventories |
| old exact init with both current providers retained | PASS14.94s; ordinary route remains valid |
| reachable local module constraints | PASS13.98s; both older constrained versions selected, root/child configuration unchanged |

Every updated entry completes required signed-package/static/dynamic checks and
ALLOW before complete publication. Five update phases produce30 fresh Evidence,
separate from12 seed and6 old exact records. Postanalysis verifies all48 unique
identity/run/digest/reference bindings and approval Evidence file hashes, final
complete directory/file membership and permissions, archive/executable digests,
controls and unchanged refusal snapshots. Preserve normalized Evidence/approvals/
verified-cache receipts and exact operators under the raw owner; these local raw
files are not claimed remotely available.

Canonical helper2482678wait0/CONFIRMED_STOPPED, original CLI/observer restoration,
all four protected installed identities, empty runtime/socket/Docker and fresh
root postguard PASS. Independent postanalysis PASS precedes deletion. Fresh root
process executable/cwd/argv/FD checks find no active/unknown references, then
718984752B of completed trial/cache/lifecycles/candidate/operators are deleted.
The last cleanup operator4736B is separately checked, archived and deleted.
Active shared checker/source/cache/observer/SDK and protected original backups
remain necessary; no completed Terraform payload or operator is retained in/tmp.
Raw receipts terraform-upgrade-postanalysis.json, terraform-upgrade-cleanup.json
and terraform-upgrade-cleanup-operator-deleted.json own those exact facts.

## Final same-source remote acceptance

CI342/run39244171743 completes with ten SUCCESS, including Required. Actual
Quick/macOS/Linux logs all checkout5aca96b821787a4cceafd4868a50b5b053afba13.
Its parents are main ef5ef0e and sourcef882921, and the entire Git tree
94d973d44ab6dd5040d988967e657f17beafcc77 equals the source tree. Native macOS
Go1.26.9 production build/default tests succeed; this is actual native execution,
not a cross-compile claim. Linux36 existing top-level Go/Cargo/Terraform consumers
(56 including subtests), CPU full192.80s and all five helperwait0 pass. Final
cleanup has original0/CONFIRMED_STOPPED/helper_stop0/client_error0/client_restore0
and restore_result=NOT_REQUIRED. This remote result is distinct from local
installed candidate restoration=RESTORED and the actual new upgrade campaign.

Raw terraform-ci342-final-jobs.json, terraform-ci342-checkout-verified.json,
terraform-ci342-final-scope.json and decoded Quick/macOS/Linux logs preserve exact
source, checkout, digests and results. The first harness failure remains recorded;
no legacy CI result substitutes for this source. Only this bounded lane closes.
Proceed to the independently reviewed GitHub replacement gaps; full SEC-UPD-001
and the deferred launcher/channel work are not claimed complete.
