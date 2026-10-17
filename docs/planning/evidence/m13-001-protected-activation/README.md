# M13-001 protected repository activation

Baseline: `2e196a188adce22005fce201c23da1dc27a43426`, 2026-10-13.
PR29 and PR30 are merged. Main CI38328099413 and all ten jobs are SUCCESS;
PR CI308 is historical evidence for b4ee265, not a replacement for main CI.

## Baseline audit

| Existing responsibility | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | Evidence / missing |
| --- | --- | --- | --- | --- | --- |
| M10 release build and manifest | YES | YES | YES (historical implementation) | YES (M10 only) | `.github/workflows/heliopause-release-build.yml`, `scripts/release-manifest/main.go:buildRelease`; production package qualification belongs to M13-003 |
| M10 protected publication workflow | YES | YES | YES (historical implementation) | YES (M10 only) | `.github/workflows/heliopause-release-publish.yml`; real publication not performed |
| M10 local verified installation/doctor | YES | YES | YES (historical implementation) | YES (M10 only) | `internal/releaseinstall/installer.go:Install`, `doctor.go:CheckInstallation`, existing rollback/tamper tests; public clean-host entry belongs to M13-005 |
| Main protection | YES | NO | NO | NO | Existing ruleset21396883 is disabled; reuse exact checks and PR policy |
| Develop protection | NO | NO | NO | NO | Branch and ruleset absent |
| Release tag protection | NO | NO | NO | NO | No active tag rulesets |
| Release environment | NO | NO | NO | NO | Environments API200, count0 |
| Immutable releases | NO | NO | NO | NO | API200, enabled=false |

No existing implementation is replaced. `scripts/check release-gate` validates
local release contracts; its success cannot prove live GitHub protection.
The authenticated Git credential helper has repository admin access. The app
connector rejects the environments endpoint; that is a connector capability
limit, not evidence of an absent environment. The direct authenticated REST
read establishes actual state. Tokens stay in memory and are never evidence.

## Applied scope and acceptance

The [activation plan](./activation-plan.json) is the exact desired state.
Reuse main's existing required checks with their integration IDs (Required15368,
license/cla128106), strict up-to-date policy, resolved conversations, zero PR
approval count and no bypass actors. The existing approval policy is retained;
this is not a claim of independent code review. Develop receives the same rules.

Create develop at the qualified main SHA before enabling its rules. Release
`v*` creation is restricted to the repository owner; separate rules with no
bypass actors prohibit updates/deletion, including by that owner. The creation
exception does not bypass immutable-tag rules.

The release environment requires owner review. Self-review remains possible
for the owner-operated repository; review is still explicit, not automatic.
Disable administrator bypass and allow workflow dispatch only from main. The existing workflow checks out the
input release tag and independently verifies source/main/Required provenance.
This branch policy constrains the workflow execution ref, not the checkout ref.
Enable immutable releases. Do not create a release tag or dispatch publication.

Completion requires API readback of all exact rules, no branch bypass,
matching develop SHA, release reviewer/branch policy and enabled immutability.
Preserve before/after evidence and pass canonical docs/workflow/release-gate
checks. Actual hostile PR/push tests belong to M13-002; no no-op push is presented
as proof of enforcement. Successful API configuration is not end-to-end release
qualification.

## Entry and release follow-ups

M12 and M12-02 technical acceptance and final delivery are complete. The bounded
host security/APT/Cargo repair is merged and main CI is green. No regression
requires reopening historical GPU qualification. Existing guest Go1.26.8 patch,
local full-Race fixtures and module-only advisories remain recorded in the
[security report](../../../engineering/evidence/security-go1269/README.md).
Their release impact is not cleared by Required. The M13 condition of no known
release-blocking finding remains a release-readiness assessment; this task
performs explicitly authorized repository hardening and does not declare
public release readiness. M13-003/005 must resolve or explicitly qualify those
remaining inputs before candidate publication.

## Result

Live application and every independent API readback assertion PASS. Main
ruleset21396883, develop24969937, tag creation24969939 and immutable tags24969940
are active. Develop starts at main2e196a1. Release environment24138487071 requires
owner approval, admin bypass is false, branch policy62857129 permits only main,
and immutable releases are enabled. Canonical docs/workflow/release-gate PASS;
final closure docs/security checks are recorded before commit.
M13-001 configuration acceptance is COMPLETE: IMPLEMENTED=YES, WIRED=YES,
QUALIFIED=YES, ACCEPTANCE_CLOSED=YES. MISSING=none within this scope.
M13-002 actual protected PR qualification remains NOT_STARTED.
The [result](./result.json) contains sanitized before/after state and original
receipt hashes; [main CI](./main-ci.json) records the distinct merged-source run.
Original authenticated API responses remain outside Git under the task receipt
root; credentials are neither logged nor retained.

Official API references: [rulesets](https://docs.github.com/en/rest/repos/rules),
[environments](https://docs.github.com/en/rest/deployments/environments),
[branch policies](https://docs.github.com/en/rest/deployments/branch-policies),
[immutable releases](https://docs.github.com/en/rest/repos/repos#enable-immutable-releases).

Reproduce readback with an authenticated repository administrator: GET
`/repos/rahoney/heliopause/rulesets/{id}` for each recorded ruleset;
`/rules/branches/main` and `/rules/branches/develop` for effective rules;
`/branches/{main,develop}` for SHA/protected state; `/environments/release` and
`/environments/release/deployment-branch-policies`; `/immutable-releases`.
Compare the returned semantic fields against activation-plan.json/result.json.
Do not rerun configuration writes as a read-only check. This task creates no
release tag, public release or publication workflow run.
