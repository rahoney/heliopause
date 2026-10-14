# M13-002 protected branch qualification

Status: IN_PROGRESS / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO. PR31 normal develop
merge and its earlier conversation rejection passed. PR32's separately approved
one-shot ordinary merge request was rejected HTTP405. Its reported expected-check
blocker does not isolate stale enforcement; further mutations have stopped.

## PR32 one-shot rejection — expected-check cause, stale not qualified

The [result](./stale-rejection.json) records the 2026-10-14 approval and sole request.
Immediately before execution, exact headff441d3, live develop84591ef and BEHIND
matched the approved scope. Required/App15368 and genuine CLA/pinned App128106
were successful and required, all eleven contexts were green, unresolved threads
were0, and the admin actor had bypass never under unchanged effective rules.

The ordinary exact-head merge returned HTTP405 with the precise server reason
`2 of 2 required status checks are expected.` Original request body, response and
receipt are preserved in [safe raw evidence](./raw/stale-merge-response.json).
Independent post-reads confirmed main2e196a1, develop84591ef and featureff441d3
unchanged, with PR32 open/unmerged/BEHIND and its head contexts still successful.
The entire API object is not unchanged: synthetic merge metadata moved from
8e5067f (old base/head) to3635960 (advanced develop/head); the latter had zero
checks/statuses. Read-only commit/check receipts preserve that distinction.

The server's expected-check reason and changed synthetic candidate do not prove
an isolated strict-freshness cause or a service/protection defect. The internal
evaluation/refresh cause remains unknown. Stale, isolated missing Required/CLA
and a separate admin-bypass case are not closed by this rejection. General
[GitHub head/test-merge semantics](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/troubleshooting-required-status-checks#conflicts-between-head-commit-and-test-merge-commit)
do not establish the cause of this specific evaluation. The one-shot approval
is consumed. No retry, push, rule change, recovery or release was performed.

Next, review the existing head/test-merge/effective-check evidence read-only and
define a rule-preserving isolation or disposition. The result owns the ordered
remaining M13-002 cases and exact approval boundaries. A new mutation requires
new scoped approval; M13-003 remains NOT_STARTED / Ready: No.

## Normal PR31 merge and PR32 stale-state observation

The [new result](./develop-merge.json) records the separately authorized normal
merge. Immediately before execution, PR32 Required/CLA were SUCCESS and its gate
was CLEAN. PR31 was also CLEAN at exact `e1e3aa8c9976ec7cf30c46b3a0103fc2899c08a2`,
targeting develop, with successful required contexts and no unresolved threads.
Both protections matched M13-001, with empty bypass lists and actor bypass never.

One ordinary exact-head PR31 merge request returned HTTP200/merged=true.
The [request](./raw/develop-merge-request.json),
[response](./raw/develop-merge-response.json) and
[receipt](./raw/develop-merge-receipt.json) retain their original bytes.
Develop advanced to `84591ef4296a852a1439781ab62cc4103a59db0a`, parents2e196a1/e1e3aa8,
with the same tree64e2961 as qualified PR31. Main stayed at
`2e196a188adce22005fce201c23da1dc27a43426`. Local result-record commits were excluded.

PR32 remains open/unmerged at exact `ff441d3d3bde941d0fd759972539ed8707475ee1`.
Its gate changed CLEAN→BEHIND while Required/CLA stayed SUCCESS and required.
No unresolved thread, requested change or content conflict was observed. Ancestry
comparison shows develop-only3 and PR32-only1 commits with merge base2e196a1.
This distinguishes the stale state from failed checks or a review/conflict blocker.
No automatic CI rerun was observed through 2026-10-13T14:41:12Z: run38403196699
remained the only head-filtered PR run, completed SUCCESS at attempt1.

PR `base.sha`/`baseRefOid` and synthetic merge metadata still identify the original
tested snapshot2e196a1/8e5067f. Actual develop advancement is independently established
by live Git refs, fetch and commit/compare APIs. The successful old checks are not
qualification of a new merge with the advanced base. Original metadata is preserved;
the reason for its refresh timing is not asserted. GitHub defines
[BEHIND as an out-of-date head](https://docs.github.com/en/graphql/reference/pulls#mergestatestatus).

During that historical phase no PR32 merge request, additional push, protection
change, main merge or release was performed; actual stale rejection was NOT_RUN.
The separately approved one-shot request above supersedes only that pending label.
Full original readbacks are local under `docs/github-logs/m13-002-base-advance/`
and the latest timestamped `docs/github-logs/m13-002-stale-preparation/remote/`.

## Historical PR31 CI/CLA verification before merge

The [remote result](./remote-result.json) records the earlier authorized push/PR
phase. PR31 was then open at exact e1e3aa8, targeting develop2e196a1.
CI38355837742 checked merge6950673, whose whole tree64e2961 matches the feature head.
No successful merge had occurred at that snapshot. Subsequent conversation rejection
and normal merge have separate results and authorizations.

CI38355837742 completed SUCCESS: Required and all nine prerequisite jobs passed.
GraphQL confirms Required/App15368 and license/cla/App128106 are required on
PR31. Its gate transitioned from BLOCKED during incomplete checks to CLEAN after
success; that snapshot was open/unmerged with auto-merge null and zero threads.
Ten complete original job logs are captured. Linux actual45 top-level tests,
CPU full287.35s, observer102 latches and all five helper waits0 with final
CONFIRMED_STOPPED/original0 cleanup are recorded in the new result. No fresh
CUDA full run is claimed. These results bind exact e1e3aa8; later local result
documentation is not a new PR candidate and is not automatically remotely qualified.

The CLA status was initially absent, then the actual `cla-assistant[bot]`
reported success. Its public App ID128106 matches the pinned rule; GitHub
GraphQL marks the status `isRequired=true` on this PR. This natural status
transition is not an executed missing-CLA merge rejection. Likewise, observing
BLOCKED while CI is incomplete is read-only gate evidence, not an attempted
merge. The subsequent actual conversation rejection is independently recorded below;
other mutation-based enforcement cases remain open.

Reader preparation failures are separate from CI: the combined `/status`
response omits detailed status `creator` fields, causing two initial local
readers to stop. Those responses remain preserved. The corrected reader uses
`/statuses`; it neither changes the service nor fabricates a status. An empty
repository webhook list does not enumerate GitHub App internal subscriptions.

## Actual conversation rejection — PR31 / develop

[The new case result](./thread-rejection.json) records a separately approved
single normal merge attempt against exact head `e1e3aa8`. The test created one
real inline review thread on this file's PR diff, RIGHT line8, comment4258640332,
thread `PRRT_kwDOT2M-Hs6r28gs`. It was not an issue/conversation comment.

Before the request, Required and CLA were SUCCESS and required, all eleven
contexts succeeded, exact head/base were unchanged, the sole review thread was
unresolved, and the gate had changed CLEAN→BLOCKED. Develop rules24969937 still
required conversation resolution, strict checks and no bypass; no approval-count
or merge-conflict blocker was present. The actor has admin permission but used
the ordinary endpoint, without an override/force/auto-merge option.

One `PUT /repos/rahoney/heliopause/pulls/31/merge` with exact head SHA and
`merge_method=merge` returned HTTP405. GitHub explicitly reported:
“A conversation must be resolved before this pull request can be merged.”
The [exact request](./raw/thread-merge-request.json),
[response](./raw/thread-merge-response.json) and
[receipt](./raw/thread-merge-receipt.json) are preserved byte-for-byte.
Post-rejection reads confirmed main/develop stayed at2e196a1 and PR31 remained
open/unmerged. Only this test thread was then resolved; that phase ended with gate CLEAN,
with zero unresolved threads, unchanged head/base and successful Required/CLA.

This case is PASS for actual develop conversation enforcement. It is not a
main-target test, separate override attempt, normal merge qualification, or a
missing-check/freshness/direct-push/force-push result. Overall M13-002 remains
IN_PROGRESS / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO. Full original readbacks are
local under `docs/github-logs/m13-002-thread-test/`. No additional push, successful
merge, settings change or release occurred in that conversation-test phase. Further remote mutations need a
separate scope of approval; the one-attempt authorization has been consumed.

## Independent stale-test PR preparation

[PR32](https://github.com/rahoney/heliopause/pull/32) was created under a new
approval limited to auxiliary feature push and develop PR creation/readback.
Its exact head is `ff441d3d3bde941d0fd759972539ed8707475ee1`, directly parented
by freshly read remote develop `2e196a188adce22005fce201c23da1dc27a43426`.
It adds only `docs/planning/evidence/m13-002-stale-probe/README.md` (12 lines).
M13-001, PR31 and local result-record commits are not ancestors; all other
tracked blobs/modes equal develop. The original milestone worktree is separate.

The [preparation result](./stale-pr-preparation.json) binds candidate, remote
state and original receipts. Its own CI38403196699 completed Required plus nine
prerequisites SUCCESS, and actual CLA reported SUCCESS; both contexts are required.
Its gate became CLEAN before the approved PR31 merge. Ten complete job logs and
whole head/test-merge tree equality are preserved. Linux45 actual top-level tests,
CPU286.41s, observer102 latches, five helper waits0 and final original0 confirmed
cleanup belong to this candidate's own run. No fresh CUDA qualification is claimed.
Local docs/security passed. The marker and PR31 modify disjoint paths.

The preparation phase performed no merge request, protected-ref push, rule change
or release. Its initially incomplete checks were not stale rejection evidence.
The later separately authorized PR31 merge and resulting PR32 BEHIND state are
recorded above. The later one-shot rejection now has its own result. Preserve
the auxiliary head; M13-002 qualification/acceptance remains open.

## Local result-record validation

The full working-directory security check failed with354 `generic-api-key`
findings in40 untracked raw API captures. [The triage](./security-triage.json)
binds every finding to the public App identifier returned by unauthenticated
GitHub APIs: GitHub Actions353 and CLA Assistant1. These are public `client_id`
values, not client secrets. The original failure and response bytes are preserved;
no scanner rule, ignore, product code or credential was changed.

Raw API custody was outside the commit candidate before this discovery because
of public profile data. The full versioned candidate is therefore exported with
all tracked files plus the explicitly listed result records and checked using
unchanged canonical `docs` and `security` profiles. This is a separate scope;
it does not turn the original working-directory failure into PASS. Candidate
path/mode/hash inventory, command results and complete logs are retained locally
in `docs/github-logs/m13-002-pr/final-candidate-validation/` for the earlier record.
The PR31 merge/stale-state record uses the same full-candidate boundary, with inventory
and canonical check logs under `docs/github-logs/m13-002-base-advance/record-validation/`.
The PR32 one-shot result uses that same explicit full-versioned-candidate boundary;
its inventory and final check receipts are under
`docs/github-logs/m13-002-stale-rejection/record-validation/`. Full original API
custody remains local and outside that versioned candidate.
The initial `remote-record-*` check files remain untracked preparation evidence
and are not final-candidate validation. Original raw custody remains local.

## Baseline audit — 2026-10-13

The local milestone branch starts at `b3c8b7dc487046afdad52a215c101d761f055098`.
Fresh `git ls-remote --heads origin`, fetch and authenticated REST GETs confirm:

- `main` and `develop`: `2e196a188adce22005fce201c23da1dc27a43426`.
- Remote `milestone/m13-production-release-operations`: exact `b3c8b7d`.
- No `feature/*` refs and no open PRs at this audit. There is no local develop
  branch; `origin/develop` was fetched. Historical milestone refs are not features.
- M13-001 is on the remote milestone branch and is not yet in main/develop.
  Its eleven changed files are documentation/evidence, with no product/workflow
  change. The feature candidate will include that delivery delta.
- The authenticated actor has admin permission, but both active branch rulesets
  report `current_user_can_bypass=never`, with empty bypass lists.
- Main/develop effective rules require PRs, resolved review threads, strict
  up-to-date checks `Required` (App15368) and `license/cla` (App128106), and prohibit
  deletion/non-fast-forward updates. Semantic fields match the M13-001 plan.
- Exact `b3c8b7d` has zero check runs and zero statuses. Old main/PR CI successes
  are historical evidence and will not qualify this candidate.

| Responsibility reused | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING for M13-002 |
| --- | --- | --- | --- | --- | --- |
| PR-triggered canonical CI and Required aggregate | YES | YES | NO | NO | Fresh exact feature/develop/main PR runs and logs |
| Hosted CLA authority and pinned required context | YES | YES | NO | NO | Fresh genuine service result and missing/failed case |
| Main/develop protected PR and strict freshness gates | YES | YES | NO | NO | Actual normal, stale, unresolved, direct and bypass cases |
| Non-fast-forward protection | YES | YES | NO | NO | Actual safe test design/authorization; readback alone is insufficient |

Existing implementation is reused: `.github/workflows/heliopause-ci.yml`
(`pull_request` has no base-branch filter; push CI targets main),
`scripts/check/required.go:validateRequiredResults`, CONTRIBUTING.md and rulesets
21396883/24969937. Qualification NO means new M13-002 execution is missing; it
does not revoke historical M13-001 configuration acceptance. No security setting
or implementation is replaced.

The original M13-001 `/tmp/haa-m13-admin-audit` directory is currently absent.
The versioned M13-001 plan/result/main-ci remain available. This audit preserves
new responses and never presents an old digest as a newly recovered raw response.

## Execution and evidence plan

Every live case begins with fresh refs, exact PR head/base/merge-candidate SHAs,
actor permission and effective-rule readback. Preserve the request, HTTP/exit
result, full response, check names/App identities, review-thread state, workflow
run/jobs and raw logs. After a rejected mutation, independently verify both
protected refs. A rejection must identify the intended rule; local non-fast-forward
refusal, permission denial, head mismatch, draft status or an unrelated blocker
does not qualify that rule. Combined blockers are labelled as combined evidence.
No-op/dry-run pushes and mergeability metadata alone are not enforcement PASS.

| Case | Actual proposed operation | Required observation and follow-up |
| --- | --- | --- |
| Normal feature → develop | Publish reviewed documentation candidate to `feature/m13-002-protected-path`, open non-draft PR to develop, wait for fresh CI/CLA, reviewer-authorized merge | Exact fresh Required plus nine prerequisite jobs succeed, real CLA service succeeds, threads resolved, current base included, merge receipt and develop ref agree |
| Missing/failed Required | Before Required succeeds, attempt the exact-head merge only after capturing its absent/failing state | GitHub rejects on Required; subsequently the genuine CI completes successfully. If a CI test fails unexpectedly, investigate/fix/regress rather than treating it as a planned negative |
| Missing/failed CLA | Use a genuine absent/failing CLA service state on the candidate and attempt exact-head merge | Rejection cites CLA; only the real contributor/service can clear it. No fabricated identity/signature, maintainer status injection or rule change |
| Stale branch | Prepare a sibling documentation PR before an authorized base-advancement PR; advance base only through a protected PR, then attempt stale candidate merge | Required/CLA success belongs to the old base and all other blockers are cleared; GitHub rejects freshness. Merge current base into feature, push with approval and run fresh checks before a normal merge |
| Unresolved conversation | Add an explicitly authorized inline test review thread on a changed documentation line; otherwise clear gates and attempt exact-head merge | GitHub rejects unresolved thread; record thread ID, resolve it explicitly, then qualify the normal path |
| Direct main/develop push | Attempt a reviewed documentation-only fast-forward commit not associated with a PR on that target | Server rejects the PR rule and ref stays unchanged. Require explicit push approval. An unexpected success only appends history; stop and preserve it, with no automatic reset |
| Admin/bypass merge | Under a captured blocking condition, use the admin actor to request the exact-head merge through the normal GitHub merge endpoint/UI | Admin is still blocked by the intended rule. Do not add bypass actors, alter settings or enable deferred auto-merge. If unexpectedly accepted, stop and preserve evidence |
| Force-push | A genuine non-fast-forward server update would be needed | NOT_RUN pending a design/decision consistent with history preservation; a force flag on a fast-forward/no-op does not test this rule |
| Normal develop → main | After develop qualification, open the actual develop-to-main PR, wait for that PR's fresh Required/CLA, clear freshness/threads and obtain reviewer merge approval | Exact head/base/merge SHA and new main CI are recorded. Old feature/main CI is never substituted |

Apply applicable negative cases to main and develop and record the target for
each result; a result on one ref is not silently assigned to the other. Prepare
additional doc-only sibling candidates as each phase's exact remote state is
known. A strict main freshness gap is resolved by an approved forward-only
synchronization PR (main → develop), not a direct push or reset. All changes
remain within M13-002 documentation/evidence.

## Authorization and safe stopping points

The user explicitly prohibits push and merge without approval. This includes
expected-to-fail pushes, ref-creation APIs and admin/negative merge attempts:
if a gate is defective, the attempt could succeed. Local preparation and GETs
are authorized; the read-only probe cannot issue a mutation.

The first approval identified the local feature commit, its full changed-path
inventory, its ancestry at `b3c8b7d`/main, and local check results. It covers only
publishing exact e1e3aa8 to that feature ref and opening its develop PR31; those
two actions are now complete. PR review-thread
writes and each successful/negative merge or protected-branch push require
explicitly scoped approval before execution. No release tag, release workflow
dispatch, publication, rule mutation or M13-003 implementation is included.

A genuine non-fast-forward update to existing main/develop cannot be guaranteed
harmless if enforcement fails. Current rules match only the exact two protected
branch names; a disposable branch would not exercise those rules. Do not run a
rewind, orphan replacement, atomic push with a deliberately failing second ref,
or treat an unprotected disposable-branch rejection as proof. Keep force-push
qualification open until the user agrees on a bounded approach or explicitly
decides its disposition; do not silently waive acceptance.

Missing CLA may need a willing real contributor/account that has not signed,
or a naturally captured absent status at PR creation. If the service already
reports success, do not revoke a signature or disable the app to manufacture a
failure. Report the exact additional participant/action needed at that point.

## Evidence custody and status

[Baseline result](./baseline.json) binds the new readback and raw receipts.
Original authenticated response bodies are under the local, existing untracked
`docs/github-logs/m13-002-20261013-baseline/` directory. They are outside the
commit candidate because full API responses may contain public profile data.
No credential is printed or saved. The probe obtains existing Git credentials
in memory, forbids redirects and uses repository-scoped GET requests only.
The initial sandbox invocation could not access the existing credential helper;
the explicitly approved host invocation succeeded. That access failure is not
a ruleset rejection or a product failure.

Raw receipt preservation is local until explicitly included in a suitable
sanitized evidence delivery; a summary hash is not a remote copy of its raw body.
The versioned baseline is a new observation of settings, not new qualification
PASS. Local checks are separately recorded in [check results](./checks.json).
An initial docs invocation ran before its two linked receipts existed and failed
on those two links. The receipts were then written, and the finalized docs check
passed. This preparation failure is retained as a terminal excerpt in the local
raw directory, not presented as an enforcement test or omitted from the history.

Current missing work: remaining live normal/negative cases, their raw evidence, authorized
reviewer merges and final acceptance audit.
The real CLA service has reported success for PR31. State remains IN_PROGRESS;
M13-001 stays COMPLETE and M13-003..005 stay NOT_STARTED. The approved push/PR
did not authorize additional remote writes. The subsequent single conversation
rejection test had its own explicit approval. Fresh raw responses, the creation
receipt and completed job logs are preserved locally in
`docs/github-logs/m13-002-pr/`; subsequent result recording is not a push to PR31.

Official semantics used in the plan:
[rules and expected App/freshness/thread enforcement](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets),
[checks on head and test merge commit](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/troubleshooting-required-status-checks),
[exact-head merge API](https://docs.github.com/en/rest/pulls/pulls#merge-a-pull-request).
