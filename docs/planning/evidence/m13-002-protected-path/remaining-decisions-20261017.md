# M13-002 remaining acceptance decisions

Status: READY_FOR_REVIEW / NOT_APPROVED. This document records proposed decisions;
it does not change the current acceptance, work order or publication authority.
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

Proposed disposition: accept **BOUNDED_OPERATIONAL_STRICT_REJECTION** for the
operational stale case under the existing
[Strict analysis boundary](./strict-check-analysis.json). Preserve the primary
REJECTED_NOT_ISOLATED / independent freshness-only NOT_QUALIFIED labels. This
accepts the observed protected flow, not an isolated internal lookup algorithm,
missing-CLA rejection or automatic overall qualification. No additional remote
attempt, head update or CI run is proposed.

Alternative: retain the independent evidence gap and require a separately
reviewed new method. Repeating the same old-head request is not that method.
Reviewer disposition is still pending; no new PASS is asserted here.

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

C2 is an approval proposal, not current permission. Its exact proposed contract
addition is:

> M13-002의 실제 missing-CLA 부정 시험은 NOT_TESTED로 남기고
> QUALIFIED=NO / ACCEPTANCE_CLOSED=NO를 유지한다. 실제 미서명 기여자 부재에
> 한해 사용자가 승인한 작업 순서 예외로 M13-003의 구현·build·package 준비를
> 진행할 수 있다. 실제 public release, 편의 채널 publish, release tag 생성과
> publish activation은 이 CLA 관문이 닫히거나 별도 명시적 canonical 결정이
> 있기 전까지 금지한다. 필수 CLA App·정상 서명·보호 설정은 유지한다.

If C2 is approved, apply that paragraph once in the M13 contract and update the
Queue/decision index/root README before starting a new item. Keep only one item
IN_PROGRESS; the waiting M13-002 remains open, not COMPLETE. Audit M13-003's
existing implementation before source edits and implement only its actual gaps.
This proposal is not a CLA test waiver or a claim that NOT_TESTED is PASS.

## Existing permission and cleanup

The prior approval already covered the disposable nonFF method and its five
writes. That test passed in its limited scope and both temporary resources were
deleted. No repeated force/probe approval is requested. Normal PR35/36 delivery
and actual main CI324 are also complete. Remaining decisions S/C neither reopen
old PR32/33 nor authorize production force, status fabrication or publication.
