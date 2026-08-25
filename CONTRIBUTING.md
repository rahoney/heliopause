# Contributing to Heliopause

Thank you for helping with Heliopause. Heliopause is distributed under the
Apache-2.0 `LICENSE`. Before an external contribution is merged, the
contributor must accept `CLA.md` and the automated CLA status check must be
successful. A missing or failed CLA status is a merge blocker; maintainer
approval does not override it.

## CLA process

The repository uses the hosted GitHub CLA Assistant as the canonical automated
PR status check. Maintainers must configure the `rahoney/heliopause` repository
in the service with this repository’s `CLA.md`, and require its status check in
branch protection before the first public release. The service is the PR-status
authority; it must not receive a maintainer PAT, repository signing key, or
repository write/signing authority. Any required app permissions are limited to
reading the PR and posting the CLA status/comment needed for the check.

The maintainer must configure the status check name as a required branch
protection check (for example `cla-assistant`). A PR is not mergeable until all
external authors have an accepted signature. Contributors should follow the
link and instructions posted by the hosted service on the PR.

## Individual and legal-entity contributions

- An individual confirms they own or can grant the rights in the contribution
  and that an employer or contract does not claim conflicting rights.
- A legal entity contribution must be submitted by an authorized
  representative and must cover the entity and relevant Affiliates that own
  the contribution or patent claims.
- If an individual created work within employment or a contract, obtain the
  employer/entity’s approval or have the entity execute the entity form before
  submitting it.

## Third-party and mixed contributions

Do not submit copied, generated, vendored, or adapted third-party code as if it
were wholly yours. Identify the upstream copyright owner and license in the PR
description, preserve required notices, and attach written permission or an
upstream license that permits the proposed use and relicensing. Maintainers do
not assume that a third-party contribution is relicensable merely because it is
present in a patch, dependency, fixture, or generated file. If ownership or
license scope is uncertain, the contribution is rejected or held until the
rights are verified.

## Existing code and CLA effective date

The repository history before CLA adoption was authored by the project
maintainer (`rahoney`); those commits are recorded as pre-CLA maintainer code.
The project does not backdate or fabricate CLA signatures for those commits.
Any historical third-party material discovered later is audited separately and
is not treated as relicensable without evidence from its copyright owner.

## Pull requests

Keep changes narrowly scoped, include tests and security evidence where
relevant, and do not include secrets or personal environment files. Run the
canonical checks described in the repository quality documentation before
requesting review. Maintainers will merge only after required CI and the CLA
status check are green.

For legal questions, contact the project owner before submitting a contribution;
do not rely on this policy as legal advice.
