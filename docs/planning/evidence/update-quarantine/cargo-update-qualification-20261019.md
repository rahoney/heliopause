# SEC-UPD-001 — Cargo lock update qualification / 2026-10-19

Status: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO.
MISSING: actual installed public update/add/build consumers, protected restoration,
completed trial cleanup and new-source remote qualification. Only SEC-UPD-001 is
IN_PROGRESS. The [reviewed contract](../../../user-journey-cli-ia/06-update-quarantine.md#cargo-lock-update--reviewed-initial-semantics)
and [pre-source audit](./cargo-update-baseline-20261019.md) own scope and edit plan.
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

These synthetic checks are not public malicious-package experiments, actual
installed/gVisor update success or general proof that a package is safe. No-op
fresh inspection and actual conservative selector behavior remain to qualify.

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
