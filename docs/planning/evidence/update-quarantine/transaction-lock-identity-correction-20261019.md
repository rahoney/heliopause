# SEC-UPD-001 — mutation lock identity correction / 2026-10-19

Only SEC-UPD-001 is IN_PROGRESS; this is a demonstrated cross-owner acceptance
gap found during the current GitHub lane, not another milestone or a replacement
of implemented transaction infrastructure. GitHub's first actual transaction unit
capture6.30s admits a foreign lock after unlink/recreate because the original
closed file's inode is reused. A local GitHub descriptor pin passes6.69s and
shared report/transaction coverage10.31s. Original failures remain in the raw owner
docs/github-logs/update-quarantine-20261018, not relabeled as successful tests.

## Independent affected-owner audit before shared edits

| Physical owner / callers | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Existing guards, original-state/transaction integration | YES | YES | YES historical bounded scopes | YES prior reported acceptance | Preserve existing selection/approval/commit/rollback/cleanup; do not reimplement |
| goProjectGuard / acquireProjectTransactionGuard: Go, Cargo, Terraform and new GitHub | NO stable lifetime pin | YES existing identity comparison | NO new failure scope | NO new gap | Descriptor is closed after creation; generic verify checks inode/regular file only. Demonstrate rewrite/unlink/mode/mtime failures through the actual helper and retain descriptor until release |
| npmProjectGuard: npm initial managed install/update | NO stable lifetime pin | YES existing identity comparison | NO new failure scope | NO new gap | Closed creation descriptor; verify checks inode/mode/single link but not stable lifetime/empty content or original mtime. Run direct same-input matrix before edits |
| pypiVenvGuard: public/native/named PyPI operations | NO stable lifetime pin | YES existing identity comparison | NO new failure scope | NO new gap | Closed creation descriptor; verify checks inode/regular/single link. Run direct same-input matrix before edits |

The old Go/Cargo foreign-lock tests preallocate a second file and rename it over
the original. That prevents immediate inode reuse and does not test modification
of the original inode's bytes/time. Their historical PASS is valid for that input,
not evidence that the newly demonstrated conditions are already covered. Code
similarity alone does not label the two other guard families fixed or failed;
their direct executable results are required next.

## Narrow edit and evidence plan

1. Commit this audit/edit plan/second review before shared production edits. Add
   one focused matrix across the three physical guard families: ordinary own
   verification/release; content rewrite; unlink/recreate empty/nonempty; chmod;
   hardlink and timestamp-only mutation. Log whether inode equality recurred;
   require verify/refuse, failed release and foreign pathname/bytes preservation.
   Existing rename-negative tests stay. Preserve the actual RED scope.
2. In go_transaction.go, npm_transaction.go and pypi_venv_transaction.go, keep the
   descriptor returned by each existing exclusive creation open. Reuse the
   existing acquisition/locking/path/root and transaction owners. A small private
   shared identity check may compare the held FD, original/current path identity,
   regular type, zero size,0600, single link and original mtime. No new public
   interface/package/configuration/runtime/SDK/budget or source selector.
3. Each existing release checks ownership while that descriptor still pins the
   inode, removes only its owned pathname, and closes all descriptors on both
   success/failure. Uncertain identity preserves the foreign pathname. Preserve
   initial creation/stat/sync errors and existing Close idempotence at session
   owners. Remove the redundant second pin from the new GitHub session so the
   same common owner performs the actual lifetime pin for all four consumers.
4. Same failing matrix GREEN, full affected promotion/Application/CLI suites and
   related Race. Canonical format/Host Quick/security/docs/checker architecture
   tests on complete exact inputs, then build one candidate. Keep all previously
   qualified resolver/inspection/runtime/Python profile bytes and limits unchanged.
5. Actual ordinary-UID installed smoke through the canonical helper lifecycle:
   small managed npm and public PyPI update/no-op, Go/Cargo/Terraform selected
   update consumers alongside GitHub's complete new public archive/ELF campaign.
   Confirm fresh Evidence, original restoration/helperwait0 and root postguards.
   Preserve normalized proof then delete all completed target/intake/cache/staging/
   candidates/operators. Existing named CPU/CUDA resolver/guest/runtime inputs
   are unchanged; no whole GPU rerun is inferred from this Host descriptor fix.
6. One new-source draft PR38 push, all ten CI SUCCESS/Required, exact checkout
   tree/native macOS and old Linux consumers/CPU/confirmed cleanup. Historical
   per-source qualifications remain bounded; full update acceptance waits for
   this new failure scope as well as GitHub acceptance.

## Second review

Changing only pathname hashes or adding another stat does not prevent inode ABA.
Retaining the originally created FD prevents reuse of that inode until close and
lets the guard detect unlink through link count and current-path mismatch. Empty
content/0600/mtime checks additionally reject writes to the same inode. Do not
remove an uncertain name just to clear a lock, close the pin before verifying/
unlinking, ignore cleanup errors, change file limits, or add a new daemon/token.

This extends the existing guard lifetime; no existing implemented transaction is
replaced. Three physical owners and their callers are explicit; no unrelated
runtime/accounting/signature/source refactor or historical full re-audit follows
without a demonstrated cause. First prove the two additional families, then
confine edits to exactly confirmed owners and reuse existing checks.

## Direct shared-owner RED before production correction

mutation-lock-identity-red executes3.14s and fails in all three physical owners.
Ordinary verification/release passes in all three. Rewrite, unlink/recreate empty
and nonempty, and mtime-only mutation are admitted and then removed by all three;
the capture explicitly logs same inode=true for each unlink/recreate. Generic
Go/Cargo/Terraform also admits chmod and hardlink; PyPI admits chmod while its
hardlink negative passes. npm's existing chmod/hardlink negatives pass. Preserve
these distinctions rather than claiming every negative failed identically.
The common implementation files are still unchanged at this RED checkpoint.
Existing Go1.26.9 os/file_unix.go opens descriptors with O_CLOEXEC; retaining the
Host guard descriptor does not add it to the sandbox/exec input contract.

The release audit also finds pypiVenvTransaction.close's existing recovery branch:
it intentionally preserves the guard pathname and recovery directory instead of
calling release. That branch must still close the retained descriptor after its
final identity check. Add a recovery test that verifies name preservation,
continued acquisition refusal and closed FD. Ordinary and failure releases close
their descriptors exactly once; session owners retain their existing idempotence.

## Bounded same-input GREEN

After the committed pre-edit plan1ea19b7, mutation-lock-identity-green executes
6.60s and passes every ordinary/foreign condition across all three physical
owners, the PyPI recovery name/closed-FD/acquisition-refusal case, and the full
current GitHub transaction corpus. Each existing exclusive creator retains its
own FD; a private verifyPinnedMutationLock checks held/path/original identity,
regular empty0600/single-link content and mtime. Release closes the FD on success
and refusal while preserving uncertain names. The PyPI recovery branch closes
its FD without unlinking. GitHub removes its duplicate second open and reuses
the corrected common owner. Existing selection, Evidence, transaction structure,
SDK/runtime/observer/budgets remain unchanged.

MISSING: full canonical/default/Race, actual small affected installed consumers
with original restoration/deletion, and exact new-source remote acceptance.
This focused GREEN does not reclassify historical failures or imply those
remaining results already passed.

Complete final same-input Quick150.58s/security8.75s/docs0.34s and full related
Race221.88s now PASS on the shared final candidate recorded by the
[GitHub qualification](./github-replacement-qualification-20261019.md#final-local-candidate-before-installed-execution).
MISSING is now the installed small affected consumers/restoration/deletion and
new-source remote acceptance only. This does not claim fresh CUDA full runs.

## Installed shared-owner regression

Final sourcecb1/candidate7b184e04 passes15 installed calls241.65s: npm is-number
6→7, public PyPI packaging24.2→25.0, Cargo itoa1.0.16→1.0.18, Go pflag1.0.9→1.0.10
and Terraform random3.7.1→3.7.2, each followed by fresh managed no-op. All calls
use ordinary client UID and the canonical helper lifecycle. PyPI venv preparation
uses the exact registered Python3.14.7 image offline, without Host Python or pip.
Required checks/Policy/promotion pass,32 successful JSON Evidence are newly
created for their calls and bind actual identities/digests;39 total normalized
records also retain old add/get/init seed evidence. Cargo/Go/Terraform no-op
controls preserve bytes/inode/mode/mtime. All project locks are absent on success.

The same lifecycle runs GitHub's18 full actual calls first. Helper2536336wait0/
CONFIRMED_STOPPED, original CLI/observer restoration and fresh root four-file/
process/socket/Docker guard all PASS. Independent output/proof/inventory
postanalysis PASS; completed trial/cache/lifecycle/candidate/operators are deleted
after preservation. [GitHub qualification](./github-replacement-qualification-20261019.md#installed-sourcecb1-qualification-and-completed-cleanup)
owns combined cleanup totals and raw references, avoiding duplicate raw logs.
MISSING: CI343 final Linux/Required and exact remote consumer review only.
No observer/runtime/profile/budget/source selection change or full CUDA rerun is
implied by this bounded Host descriptor correction.
