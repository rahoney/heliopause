# SEC-UPD-001 — Cargo lock update qualification / 2026-10-19

Status: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES
for the reviewed bounded Cargo update lane; overall SEC-UPD-001 remains open.
Local automated and actual installed qualification: PASS.
MISSING: none in this bounded lane. CI338 final source/consumer/cleanup evidence,
protected local restoration and completed trial/tool cleanup are confirmed. Only SEC-UPD-001 is
IN_PROGRESS. The [reviewed contract](../../../user-journey-cli-ia/06-update-quarantine.md#cargo-lock-update--reviewed-initial-semantics)
and [pre-source audit](./cargo-update-baseline-20261019.md) own scope and edit plan.
Implementation source85ac4c19780faeb073211ffd90759ae2a6ed6d17.
Plan commit7640223 preceded the in-place implementation; no runtime, observer,
resource budget or existing requested-primary contract was widened.

## Implemented boundaries

A complete-project update value shares immutable original/selected control binding
without assigning a crate an arbitrary primary role. Old constructors and the
Go/Terraform requested-primary consumers remain strict. The existing private
Cargo source traversal admits only update or update -p validated-selector, binds
the selector to one original public lock entry and forbids any manifest/local
source change. Existing public acquisition, integrity/static inspection, fresh
Evidence, M3/M4, cache staging and held transaction are reused for the whole set.

Explicit --adopt enrolls supported native or deliberately changed controls, while
existing old approval/cache/Evidence remain independently validated. Implicit,
foreign-marker, corrupt state/cache, missing Evidence/lock and recovery cases
refuse. Lock-only publication retains unchanged manifest and no-op lock inode,
mode and mtime; changed lock preserves original permission mode. Exact backup
membership/rollback and retained-control identity checks remain fail closed.

The CLI admits no positionals and at most one -p/--package, rejecting repeated or
empty selectors and unsupported native flags before composition. Its static help
and English/Korean human report use existing presentation/progress. JSON includes
real per-entry checks/Evidence/Policy under a separate project-scoped schema.
Fresh inspection ALLOW cannot become publication success after stage/commit/close
failure; error causes and typed completed inspection facts survive independently.
Build-script/macro behavior remains separate helox cargo build inspection.

## Local checks and original evidence

Local raw owner: docs/github-logs/update-quarantine-20261018/. These untracked
original captures are not claimed to be committed or remotely available. Every
check has an exact input inventory, actual exit, elapsed time and log digest.

| Evidence | Actual bounded result |
| --- | --- |
| cargo-command-admission-red | Existing fixed command refusal3.28s; missing new scope |
| cargo-selection-first-green | PASS5.43s; typed data, selectors and private fake-runner selection |
| cargo-lock-retention-red / first-green | RED4.78s → GREEN6.15s; unchanged control identity and lock permissions; old Go/Terraform refusal consumers |
| cargo-enrollment-consumer-green | PASS12.61s; native/managed/changed intent, corrupt old authority, commit faults, foreign same-byte identity and affected build inputs |
| cargo-cli-complete-green | PASS2.28s; direct/preflight argv, English/Korean human/JSON and truthful failed publication |
| cargo-adverse-focused | PASS21.24s; first/second entry integrity/block/review/incomplete refusals, complete inspection and no staging/publication |
| cargo-adverse-final-format | PASS2.63s; canonical explicit formatter |
| cargo-final-quick | PASS259.64s on Host; canonical format/build/test-build/architecture/workflow/vet/pinned Staticcheck/default tests |
| cargo-final-security / docs | PASS13.44s /0.42s |
| cargo-final-race | PASS148.85s; changed domain/Cargo adapter/Application/promotion/sandbox/CLI and affected old consumers |
| cargo-update-candidate | Go1.26.9/CGO0 build4.39s; SHA256197528b45f0f38f2c59c391b4ef59d26e0d4e3ce2f356612da118ce06c814734,27235428B; same entire frozen check inputs |
| cargo-baseline-ci337-actual-scope | Prior document head22bbc delivery only:10 SUCCESS, actual checkout ac23206 has same whole tree1c39fed; native macOS, CPU full285.60s/five helper wait0/final confirmed cleanup. This contains none of the new Cargo source |

The synthetic adverse fixtures are not public malicious-package experiments or
general proof that a package is safe. Actual installed/gVisor success, fresh no-op
inspection and conservative selector behavior are independently verified below.

## Preserved intermediate failures

The first lock retention RED shows missing update semantics in the existing add
transaction, not an alleged defect in historical add/build. A subsequent compile
failure exposed the existing build-input consumer of verifyRetainedApproval; its
strict signature is retained and only explicit enrollment uses the new variant.
The first CLI negative test showed pinned pflag GetStringArray collapses an
explicit empty argument through CSV conversion; reading its actual SliceValue
preserves occurrence count and rejects empty/repeated input. Preflight's second
parse is also reset explicitly and verified in a full preflight→Execute test.

cargo-wired-quick failed184.35s because the tool sandbox prohibited Unix sockets;
the approved Host run above passes. cargo-wired-security flagged the guarded
arguments[2] access with G602; the existing bounded argument traversal now uses
range and the fixed security profile above passes without suppression. Original
failures remain separate from final input/result receipts.


## Actual installed Linux consumers and cleanup

Both successful campaigns use the same source85ac4c1-bound CLI SHA256197528b…14734
and the unchanged observer candidate dbc79f6…a49. The exact entire input inventory
and buildinfo bind the executable; private build metadata retains its earlier
local clone base and is not itself claimed to name implementation85ac4c1.
The source checkpoint independently matches all non-Markdown runtime/build inputs.
No product source changed between campaigns or after the harness correction.

| Actual evidence | Result and exact scope |
| --- | --- |
| cargo-update-installed/result.json | PASS13 phases, UID1000 installed path. Existing add seeds itoa1.0.16 and ryu1.0.20; unsupported --precise and implicit native enrollment refuse with original project preserved |
| native-one-update | COMPLETED/ALLOW/publication COMPLETED10.89s. Explicit --adopt -p itoa@1.0.16 selects itoa1.0.18 and retains unrelated ryu1.0.20; manifest identity/mode/time and original lock0640 permissions asserted |
| managed-all-update | COMPLETED/ALLOW/publication COMPLETED11.03s; ryu1.0.20→1.0.23 while the original manifest ranges/features/source remain exact |
| managed-one-noop | COMPLETED/ALLOW/publication COMPLETED10.87s; same full graph and exact original manifest/lock inode/mode/mtime; fresh independent checks/Evidence |
| changed-control-adopt-ko | Human Korean COMPLETED10.74s after an intentional comment edit; implicit request first refuses unchanged, explicit enrollment preserves actual edited controls and re-inspects the whole graph |
| approved-build | Existing observed Cargo build14.74s; actual ALLOW and published ELF/output receipt, preserved output identity/digest. This is separate build behavior, not dynamic inspection attributed to lock update |
| negative original preservation | Missing selector, implicit edited input, foreign HAA marker and corrupt independent approval under --adopt refuse with full target snapshots unchanged |
| cargo-update-edge2-installed/result.json | PASS7 additional phases; explicit native dependency-free update, managed dependency-free no-op, workspace/features enrollment and workspace no-op; local root/member selectors and source replacement config refuse |
| edge2 empty graph | JSON explicitly dependency_free=true/entries0; independent retained approval has its canonical nil entries and two fresh cache handles. No artifact Evidence is invented for an empty graph |
| edge2 workspace-retained-source / postanalysis | Root and member manifests plus two source files preserve bytes/inode/mode/mtime. The complete two-public-entry graph is inspected;8 new Evidence handles are disjoint from all16 nonempty-update handles in the first campaign |
| both successful lifecycle receipts and installed-after | Actual helper wait0/CONFIRMED_STOPPED, observer restore, canonical client RESTORED; fresh root four original protected hashes/owners/modes, process/socket and Docker queries PASS |
| cargo-update-cleanup / edge2-cleanup | Completed successful target/cache/lifecycle payloads37,820,491B and27,134,558B deleted after original record preservation |
| cargo-update-edge-cleanup | Completed failed-harness target/cache/lifecycle26,490,416B also deleted after original CLI result, failure, cause and restoration evidence preservation |
| cargo-completed-operator-cleanup | Completed18 files54,453,436B deleted:15 temporary operator copies, one used stage-path file, and obsolete PyPI/Cargo test CLI binaries. Exact archives/buildinfo/digests and active checker/source/cache/observer/shared SDK/protected rollback are retained |
| cargo-update-help-final | Actual same-candidate help7 languages PASS; English/Korean descriptions and bounded package/adoption flags, secondary-language fallback, no ANSI |

All completed trial payloads and temporary copies above total145,898,901 logical
bytes removed. Original Evidence/approval/cache receipt/build receipt/command logs
and exact operators remain in the local raw owner. No fixture or compiled output
is retained as an installed example. The normal installed CLI/observer are restored;
the test candidate was not permanently activated or published.

Three diagnostic assumptions were corrected without product changes. The initial
static-help assertion expected a Korean word where the reviewed help uses the
technical name Cargo.lock. The first post-analysis incorrectly demanded update
permission modes after the separate historical build's CommitSnapshot had
republished0600 controls; update preservation is asserted before that later
operation and is not attributed to build. The first extra campaign's empty update
itself returned COMPLETED/ALLOW/publication COMPLETED, but its validator called
len(null) on canonical internal dependency-free approval entries. Original failure
and confirmed restoration remain preserved as failed harness scope. The corrected
validator handles null only with explicit dependency-free semantics, rejects a
missing nonempty graph offline, and the fresh complete extra campaign passes.
Neither the failed harness nor its unexecuted later cases are relabeled successful.

## Post-implementation review and remaining remote boundary

After canonical checks, responsibility review reconfirmed the strict old primary
constructor/Go/Terraform consumers, original-to-selected control binding, unchanged
local source, full fresh inspection before cache/commit, retained-control identity,
exact backup membership, primary cause preservation and no publication success
after failed close. Existing build-input approval remains strict. No observer,
network role, filesystem scope, budget, scanner or old snapshot approval protocol
was replaced or widened. Actual first/add/build and empty/workspace consumers are
separate from the synthetic adverse/fault matrix.

Source85ac4c1 was ordinarily pushed to existing draft PR38. CI338/run39210397866
is completed SUCCESS with all ten jobs, including native macOS and Required.
Actual checkout56cad35b544c20a19572316e9b0a718f117487cc has the same complete
tree ee1a6ab7287b6112954d489b3986784044da7e9a as the implementation. Decoded
Quick/macOS/Linux logs and cargo-ci338-actual-scope.json independently bind the
source,36 related existing Go/Cargo/Terraform consumers, CPU full277.12s, five
helper wait0 and final original0/CONFIRMED_STOPPED/helper_stop0/client_error0/
client_restore0/NOT_REQUIRED. Existing public/transitive add, workspace/library/
build-script/proc-macro build and security consumers pass. These standard remote
consumers are distinct from the new actual UPDATE campaigns above; no live remote
Cargo UPDATE or named CUDA rerun is inferred. The bounded Cargo lane is closed;
Go get -u, Terraform and GitHub update lanes remain open under SEC-UPD-001.
No main merge, tag, package publication or readiness transition was performed.
