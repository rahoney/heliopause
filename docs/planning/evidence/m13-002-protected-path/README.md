# M13-002 protected branch qualification

Status: IN_PROGRESS — baseline/read-only audit and local candidate preparation;
remote write authorization pending. No live enforcement case has passed yet.

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

First approval will identify the local feature commit, its full changed-path
inventory, its ancestry at `b3c8b7d`/main, and local check results. It covers only
publishing that reviewed feature ref and opening its develop PR. PR review-thread
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

Current missing work: live normal/negative cases, their raw evidence, exact PR
CI/CLA qualification, authorized reviewer merges and final acceptance audit.
State remains IN_PROGRESS / awaiting first remote-write approval; M13-001 stays
COMPLETE and M13-003..005 stay NOT_STARTED.

Official semantics used in the plan:
[rules and expected App/freshness/thread enforcement](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets),
[checks on head and test merge commit](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/troubleshooting-required-status-checks),
[exact-head merge API](https://docs.github.com/en/rest/pulls/pulls#merge-a-pull-request).
