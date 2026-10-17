# M13-002 remaining acceptance decisions

Status: APPROVED — the owner explicitly chose both recommended S/C2 decisions
on 2026-10-17. Only the accepted work-order exception below applies; M13-002
missing-CLA acceptance and publication authority remain open.
Actual execution and source/receipt bindings are in
[post-delivery evidence](./post-delivery-review-20261017.json).

Normal protected delivery, target-specific Required/conversation/direct-push
cases and the explicitly accepted disposable nonFF method are complete within
their recorded scopes. They must not be repeated merely to close these decisions.
The existing main/develop Required/App15368 and license/cla/App128106 rules,
Strict policy, resolved conversations and no-bypass settings remain required.

## Decision S — stale operational evidence

Develop suite4516024742 and main suite4574721519 each identify an actual rejected
stale-candidate update and the failed Strict required-status-check rule. The
head's genuine Required/CLA successes and the proposed merge's own missing checks
are separate authority. Same parents/tree do not transfer those checks.

Approved disposition: accept **BOUNDED_OPERATIONAL_STRICT_REJECTION** for the
operational stale case under the existing
[Strict analysis boundary](./strict-check-analysis.json). Preserve the primary
REJECTED_NOT_ISOLATED / independent freshness-only NOT_QUALIFIED labels. This
accepts the observed protected flow, not an isolated internal lookup algorithm,
missing-CLA rejection or automatic overall qualification. No additional remote
attempt, head update or CI run is proposed.

Alternative: retain the independent evidence gap and require a separately
reviewed new method. Repeating the same old-head request is not that method.
The owner accepted the bounded operational disposition. Independent freshness-only
remains NOT_QUALIFIED; the historical primary results are unchanged.

## Decision C — unavailable real missing-CLA case

The owner previously confirmed no real unsigned contributor is available.
Actual missing-CLA rejection is **NOT_TESTED**. Required App wiring, signed PR
success and another context's rejection do not prove this negative. No fake
identity/contribution, status injection, signature reset or CLA/rule change may
manufacture it. The existing hosted-provider review found no applicable safe
negative-test mode; absence of that mode is not asserted universally.

The current contract requires this case before M13-002 acceptance closes. Two
explicit choices are ready for review:

| Choice | Exact effect | Boundary |
| --- | --- | --- |
| C1 — keep current order | M13-002 waits for a genuine eligible unsigned participant and applicable server rejection; M13-003..005 remain NOT_STARTED | Current acceptance stays unchanged; QUALIFIED=NO and ACCEPTANCE_CLOSED=NO |
| C2 — allow preparation while this case stays open | Keep M13-002 missing-CLA NOT_TESTED and its acceptance open; explicitly allow M13-003 implementation/build/package preparation as a work-order exception | No public release, package publication, release tag or publish activation may occur until the actual missing-CLA case closes or a separate explicit canonical decision changes that gate |

The owner approved **C2**. Its single canonical clause is now recorded in the
[M13 contract](../../18-m13-production-release-operations-contract.md#m13-002--protected-branch-qualification).
The Queue/decision index/root README track that exception. Keep only M13-003
preparation IN_PROGRESS; M13-002 remains BLOCKED on the actual missing-CLA case,
with QUALIFIED=NO / ACCEPTANCE_CLOSED=NO. Audit M13-003's existing implementation
before source edits. No CLA waiver, fake identity/status or publication is allowed.
The C1 alternative above is historical and is not the selected work order.

## Existing permission and cleanup

The prior approval already covered the disposable nonFF method and its five
writes. That test passed in its limited scope and both temporary resources were
deleted. No repeated force/probe approval is requested. Normal PR35/36 delivery
and actual main CI324 are also complete. Accepted decisions S/C neither reopen
old PR32/33 nor authorize production force, status fabrication or publication. They authorize
only the C2 implementation/build/package preparation exception.
