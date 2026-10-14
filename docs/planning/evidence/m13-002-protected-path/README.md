# M13-002 protected branch qualification

Status: IN_PROGRESS / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO. PR31 normal develop
merge and its earlier conversation rejection passed. PR32's separately approved
one-shot ordinary merge request was rejected HTTP405. Its reported expected-check
blocker does not isolate stale enforcement; further mutations have stopped.

## Concrete forward-only preparation

The [frozen plan](./forward-only-plan.json) records current refs, exact candidate
SHAs, file/blob/mode inventories, eight bounded future writes, expected outcomes,
side effects and stop conditions. This phase performed read-only requests and
local preparation only; no remote write or CI was triggered. Existing PR31 and
PR32 execution evidence is reused byte-for-byte. Develop's actual Strict-rule
stale-candidate rejection is covered within its recorded scope; no repeat or
independent freshness-only claim is proposed.

| Future approval scope | Concrete content | Excluded / remaining dependency |
| --- | --- | --- |
| A: forward-only probes and PR preparation | One ordinary direct push each: main `6b3ae56d19a9b0d59e4df8585d834b647e9da558` from2e196a1; develop `68916c7ae69e6c9af8ad80c449a52f10fe6f3bc6` from84591ef. Each adds one six-line documentation marker. Then retarget PR32/headff441d3 to main2e196a1, create one exact inline thread, attempt one ordinary conversation/admin rejection, resolve only that thread after expected rejection. Open develop84591ef→main2e196a1 PR and use its necessary CI for one Required-only rejection if real CLA is already green | Max2 Git pushes +6 API mutations; no successful protected merge, force, rule edit, fabricated CLA, retry or automatic recovery. A negative unexpectedly accepted stops all further writes |
| Intermediate main advancement and main stale case | Separately review the new develop→main PR's exact84591ef/head,2e196a1/base,18 existing documentation/evidence files and fresh gates before a normal merge. Its actual returned main SHA then makes retained PR32 a possible main stale candidate | Returned merge SHA and new PR number cannot be fixed now; main stale needs its own exact base/one-call approval. This is an intermediate qualification merge, not final closure |
| Delivery and final main merge | After actual main advancement, prepare a forward delivery candidate including that main ancestry and chosen new records; its necessary PR CI can cover develop Required-only. Final develop→main follows only after applicable negatives and explicit disposition of remaining gaps | Local `de694147bee4ea6b35b287117a5eacc9ba060a8a` is a retained7d65a05 tree snapshot, not an approved future publish SHA. Final current head/base/diff/CI require separate reviewer approval |
| Non-fast-forward | Prefer an explicitly accepted protected disposable probe method that preserves production history | No probe setup authorized. Actual hostile main/develop update always remains a separate high-risk approval; disposable evidence does not silently satisfy exact-target acceptance |

The latest ordering supersedes the older generic sequence's step7 below: required
main stale evidence cannot depend on the final merge that requires that evidence.
An intermediate canonical develop→main advancement is needed unless another
evidence method is explicitly accepted. Future PR/thread IDs bind only to the
successful create receipt plus exact readback; never guess a PR number or accept
an arbitrary later head. Every write requires fresh refs/rules/required App and
thread guards. Base retarget can change timeline/review context, and any naturally
triggered CI must be observed; no manual rerun is proposed.

The user confirms no actual unsigned contributor is available. The isolated
missing-CLA cases are **NOT_TESTED**, deferred while independent work continues.
Existing required `license/cla`/App128106 wiring and actual signed PR successes
are preserved. No arbitrary account, fake contribution, signature reset, CLA
version change or injected status is permitted. The older participant sequence
below is conditional background, not an executable part of A.

The official CLA Assistant [README](https://github.com/cla-assistant/cla-assistant/blob/main/README.md),
[troubleshooting](https://github.com/cla-assistant/cla-assistant/blob/main/COMMON_ISSUES.md)
and [contribution guide](https://github.com/cla-assistant/cla-assistant/blob/main/CONTRIBUTING.md)
provide hosted signing, real-status recheck and self-hosted development guidance.
No applicable hosted negative-test mode was found in those reviewed sources.
Recheck does not make a signed contributor unsigned and can write PR status;
its URL was not invoked. An isolated self-hosted test does not prove production
App128106 enforcement. This bounded review does not assert no provider method
could exist.

Under the unchanged [M13-002 contract](../../18-m13-production-release-operations-contract.md#m13-002--protected-branch-qualification),
`missing CLA` is a required hostile case. NOT_TESTED therefore remains an
acceptance gap: QUALIFIED=NO / ACCEPTANCE_CLOSED=NO, while other checks may proceed.
Closure needs real applicable evidence or an explicit canonical acceptance/method
disposition. Deferral does not waive the requirement. Formal release is not ready
under the current ordered M13 plan; M13-003..005 are still NOT_STARTED. This is a
project acceptance judgment, not a claim that NOT_TESTED itself fails GitHub CI
or that the currently successful CLA service is broken. Final merge/release
authorization remains separate.

Both new direct-push candidates passed pinned canonical docs/security checks.
The first sandbox security invocation had a compiled-cache lookup failure;
its original FAIL is retained, and the same candidate/command passed with host
cache access. The delivery snapshot's whole tree/modes equal already checked
7d65a05, so those checks were reused. Raw captures, diffs, local retention refs,
bundle and profile logs remain in local `docs/github-logs/m13-002-forward-plan/`;
selected metadata/hashes are versioned in the frozen plan. Whole-worktree public
API response scanner findings are not relabelled by versioned-candidate checks.

## Read-only Strict-check analysis

The [analysis](./strict-check-analysis.json) adds GitHub rule-suite4516024742 to
the preserved one-shot result without repeating the request or its CI. Its target,
before84591ef, original actor and UTC-normalized time match the sole request;
the proposed commit message identifies PR32. GitHub exposes no HTTP request-ID
join, so these explicit correlation fields are retained. The failed rule is
`required_status_checks` in active develop ruleset24969937. Its exact two contexts
are `Required`/App15368 and `license/cla`/App128106. The other evaluations
(`pull_request`, `non_fast_forward`, `deletion`) passed for this ordinary merge.

| SHA role | SHA | Required | license/cla | Identity boundary |
| --- | --- | --- | --- | --- |
| PR32 head, registered Check Run source | ff441d3 | SUCCESS | SUCCESS |10 Check Runs/suite104052117715 and one genuine status on this SHA |
| Old test merge |8e5067f | absent | absent | parents2e196a1/ff441d3; tree7cbc074 equals the historically qualified head |
| Current potential/synthetic merge |3635960 | absent | absent | parents84591ef/ff441d3; tree2f2795b |
| Actual proposed update in the failed rule suite |0fd006b | absent | absent | parents84591ef/ff441d3; tree2f2795b, distinct SHA from3635960 |

Thus the expected contexts belong to rule evaluation of the proposed0fd006b
update, not a failed rerun of the old head's green checks. The rule-suite identifies
the evaluated update and failed rule; it does not reveal every internal status
lookup or establish that3635960 was the sole lookup SHA. Equal trees do not move
check authority between commits. `baseRef.target.oid` confirms live84591ef while
`baseRefOid` still reports the older2e196a1 snapshot.

[GitHub's Strict contract](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets#require-status-checks-to-pass-before-merging)
requires base freshness in addition to required checks. The documented
[head/test-merge fallback](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/troubleshooting-required-status-checks#conflicts-between-head-commit-and-test-merge-commit)
does not remove that freshness prerequisite. The observed rejection is consistent
with this combined Strict flow. Proven: a stale candidate was actually rejected
by the exact Strict required-check rule with no branch change. Still unproven:
freshness-only causality with missing candidate checks removed as a variable.
The original `stale-rejection.json` and its independent-stale NOT_QUALIFIED label
remain byte-identical. Repeating the same develop attempt is not recommended.
Accepting the bounded Strict-flow evidence for the operational stale case is an
explicit reviewer evidence decision, not an automatic PASS or a weakened rule.

The `non_fast_forward` PASS above means the proposed merge is forward from its
first parent84591ef. It is not a hostile force-push rejection. Rule-suite API
semantics are documented in [GitHub's repository rule-suite API](https://docs.github.com/en/rest/repos/rule-suites).

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

## Remaining qualification and approval plan

This coverage mapping reuses existing executions. It does not rerun M13-001,
PR31, historical full CI or the PR32 request. Apply target-specific claims only
to their recorded branch; a required App binding is not an isolated negative.

| Acceptance behavior | develop evidence | main evidence / remaining gap | Minimum next work |
| --- | --- | --- | --- |
| Normal feature → develop | PR31 actual merge PASS | Not this behavior's target | Reuse; another merge only if needed to deliver reviewed records |
| Required/CLA successful admission | Fresh PR31/32 results preserved | New develop→main candidate required | Use the required PR CI once per final candidate |
| Missing Required alone | Not isolated by the two-context PR32 rejection | NOT_RUN | During needed signed-candidate PR CI, capture pending Required only when genuine CLA is already green |
| Missing CLA alone | NOT_TESTED | NOT_TESTED | Deferred: user confirms no real unsigned participant; preserve required binding/positive results, no fabricated author/status/signature |
| Stale under Strict checks | Actual target-rule rejection proven; independent freshness-only remains unqualified | NOT_RUN | Decide bounded evidence sufficiency; reuse a needed main sibling for its target-specific stale case |
| Unresolved conversation | PR31 actual rejection PASS | NOT_RUN | One main thread/create/ordinary rejection/resolve sequence on a needed green PR |
| Ordinary admin blocked by a live rule | PR31 admin-permission conversation rejection already proves this bounded behavior | NOT_RUN | Reuse the main conversation attempt for this same ordinary-admin requirement |
| Direct push without PR | NOT_RUN | NOT_RUN | One reviewed documentation-only fast-forward candidate/attempt per target; preserve rule-specific evaluation even if other blockers also fail |
| Hostile non-fast-forward update | No actual rejection; normal-merge PASS is inapplicable | No actual rejection found | Separate history-preserving test-scope decision below |
| Normal develop → main / final audit | Delivery and remaining develop cases first | NOT_RUN | Exact fresh PR CI/CLA, reviewer-authorized merge, new main CI and final evidence audit |

The admin row follows this plan's declared ordinary-endpoint test. Reusing PR31's
admin actor under a real unresolved conversation does not claim a special CLI,
UI or API override mode was executed. A separately requested special mode would
need its own precise requirement; duplicating the ordinary admin request adds
no coverage. Similarly, do not repeat the already proven combined two-context
required-check rejection merely to collect another identical405.

### Minimum sequence without rewriting protected history

1. Decide two evidence boundaries before remote work: whether the existing bounded
   Strict-flow rejection satisfies the operational develop stale case, and what
   force-push evidence can satisfy the original target-specific requirement.
   Neither independent-stale nor force-push acceptance is silently waived.
2. Prepare exact doc-only fast-forward direct-push candidates from fresh main and
   develop heads. A future approved batch can attempt each once before changing
   either base through PRs. A rule-suite `pull_request` failure identifies the PR
   gate; additional missing-check failures are labelled combined. If accepted,
   stop: even forward-only success moves a protected ref and may start CI.
3. If both targets need isolated missing-CLA coverage, use one willing real
   unsigned contributor and one genuine doc-only probe commit based on the frozen
   current develop. Open its develop and main PRs before any base advancement or
   signature. Wait for each PR's genuine Required result, then make at most one
   approved exact-head rejection attempt per target while CLA is truly absent or
   failed. Only after both captures may the contributor personally sign. A prior
   signature is not revoked; no one signs for the contributor. If this state is
   unavailable, leave the isolated CLA cases open rather than manufacture it.
4. Keep the signed main probe as a sibling: its doc marker must be disjoint from
   the final delivery candidate and absent from that candidate. Do not merge the
   probe into develop, or its later inclusion/auto-closure could remove the main
   stale test. The develop probe can be closed only within an approved cleanup
   scope; preserve its branch, evidence and original head.
5. Deliver the reviewed records through a signed feature→develop PR. Capture its
   missing-Required case once while CLA is green and Required is genuinely pending;
   then let the same CI finish. Use a normal protected merge only for delivery,
   not to repeat PR31 qualification. Prepare local forward merges before publishing
   the final candidate, rather than knowingly publishing a stale intermediate head.
6. Open the actual develop→main PR. Its one required CI provides the main
   missing-Required opportunity under the same guards. After green CI/CLA, use one
   explicitly approved unresolved thread and ordinary admin rejection to cover
   both main conversation and ordinary-admin behavior, then resolve that thread.
7. The reviewer approves the exact final develop→main merge separately after its
   current conditions are reviewable. This necessary base advancement can make
   the retained main probe stale. If fresh reads show its original head/green
   contexts, a genuine new base delta, BEHIND, no conflict or unresolved thread,
   one specifically approved main stale attempt supplies that target's evidence.
   No extra base-advancing PR or repeated develop stale attempt is needed. If the
   probe is merged/closed, not stale or otherwise blocked, stop the dependent case.
8. Preserve actual rule-suite/ref evidence, verify the new main CI and complete
   the final acceptance audit. Final documentation delivery may require its own
   reviewed candidate; do not claim old CI applies to a new SHA. Close probe PRs
   only under an explicit cleanup scope, without deleting or rewriting refs.

Each final candidate gets only its necessary PR CI. Do not suppress checks, make
deliberately failing CI, revoke CLA, cancel/retry runs to manufacture a window,
or run CUDA/full product qualification for these documentation-only probes.
Isolated CLA cases may require two target PR runs even for the same contributor;
they cannot be replaced by one combined pending Required+CLA response. Actual
CI counts depend on participants, valid windows and final candidate changes;
no fixed total or guaranteed capture window is asserted.

### Approval bundles and separate decisions

| Scope | Can be reviewed as one bounded approval | Guard / separate approval |
| --- | --- | --- |
| Forward-only qualification batch | Listed doc-only feature pushes and PR creation; at most one targeted negative per named condition/ref; one main test thread/create/resolve sequence; conditional green feature→develop delivery; optional probe PR closure without ref deletion | First present exact candidates/diffs, actions/count limits and current refs. Fresh check/App/thread/rule guards before every operation. Any unexpected acceptance or transport uncertainty stops all further mutation; no recovery/retry |
| Genuine CLA participant | The above batch can include the contributor's actual commit and two negative captures before signing | Identify a willing unsigned contributor first. Their real signature is a personal action, not delegated agent approval |
| Final normal main merge | Exact develop→main candidate and its fresh Required/CLA/review evidence | Separate reviewer approval when the final SHA/base and CI are known; optionally name the conditional retained-main-probe attempt and its one-call limit in that same final scope |
| Force-push evidence | Local graph checks and existing rule-suite reads are already allowed | A protected disposable probe requires separately approved setup and an explicit acceptance-scope decision; it is not exact-main/develop execution evidence |
| Actual non-fast-forward main/develop update | Excluded from the forward-only batch | A separate high-risk decision is required; if enforcement fails it rewrites the protected ref. No automatic restoration is part of any proposal |

This is a plan, not new remote authorization. No push, merge, protection change,
release or CI dispatch occurred during this analysis. The forward-only bundle is
not risk-free: an expected-to-fail operation can succeed and advance a protected
ref. The finite candidate/action list must be made concrete before execution
approval; this document does not authorize arbitrary later SHAs or retries.

### Non-fast-forward evidence without damaging production history

Prefer matching existing live rule-suite evidence with an actual
`non_fast_forward` failure. None was found in the bounded queries (develop's
last24h failures were conversation/required-check cases; main's last7d fail
query was empty), and prior execution records contain no authorized hostile
force attempt. This does not prove absence outside those query windows.

Without such evidence, first validate the proposed ancestry locally. A future
separately protected disposable branch/repository can exercise the real GitHub
non-fast-forward rule while leaving production main/develop histories intact.
It needs approved setup with no weakening of existing rules. Its result proves
that probe's enforcement and may be combined with live main/develop wiring only
if the user/reviewer explicitly accepts that evidence method. Under the current
exact-target requirement, main/develop hostile execution otherwise stays open.

There is no guaranteed harmless live non-fast-forward update of an existing
main/develop ref if its protection fails. A wrong lease, dry-run, local refusal,
`force=false` API validation, a force flag on a fast-forward/no-op, or an atomic
push rejected only by another ref does not test the target rule. A backup ref
preserves reachability but does not make a production rewind history-preserving.
Do not execute a rewind to discover whether rollback will be available. If the
requirement remains exact-target hostile execution, safe closure needs either
matching real evidence or a separately accepted risk; otherwise M13-002 remains
open. M13-003 must not start to bypass that unresolved decision.

## Execution and evidence controls

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
