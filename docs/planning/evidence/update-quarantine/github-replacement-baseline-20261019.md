# SEC-UPD-001 — GitHub replacement baseline / 2026-10-19

Read-ahead audit atf882921 while Terraform CI342 remains pending. No GitHub
source edits or activation yet. Only SEC-UPD-001 is IN_PROGRESS; Terraform
acceptance closes before this lane starts. The [canonical replacement contract](../../../user-journey-cli-ia/06-update-quarantine.md#github-managed-asset-replacement--reviewed-initial-semantics)
owns supported semantics. Existing M6 qualification is preserved for its own scope.

## Independent baseline

| Responsibility | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | Existing evidence / MISSING |
| --- | --- | --- | --- | --- | --- |
| Exact public release/asset resolution and frozen numeric-ID acquisition | YES | YES | YES existing M6 | YES existing scope | artifact/githubrelease Client.Resolve/ResolveDependencies/Acquire, ParseSelector/parseAssetLocator; M6 contract/qualification; no latest or credential fallback |
| Declared SHA-256, bounded format/static and isolated ELF dynamic inspection | YES | YES | YES existing M6 | YES existing scope | verification/githubrelease, inspection/githubrelease and existing Linux GitHub ELF integration; reuse all checks/budgets |
| Entry/set M6 Policy, Evidence, Manifest/SBOM and staged rehash | YES | YES | YES existing M6 | YES existing scope | application InstallInspectService/InstallService, policy.M6, evidence/local.ReadReference, staging.go; no replacement workflow copy |
| New-directory standalone publication | YES | YES | YES existing M6 | YES existing scope | GitHubReleasePromotion.Promote and exact/no-overwrite tests; existing target always refuses |
| Independent installed ownership receipt | NO | NO | NO | NO | MISSING controller state outside target plus bound target marker, exact installed asset/metadata inventory and original Evidence proof; current publisher saves only asset |
| Pre-resolution managed replacement session and transactional rollback | NO | NO | NO | NO | MISSING request-scoped guard/ownership proof, same-repository exact new selection and original/selected/state publication/rollback/cleanup; existing InstallUpdateService/ports already usable |
| Explicit CLI intent and bootstrap/reports | NO | NO | NO | NO | MISSING --replace with explicit --target and a dedicated bounded install mode; current NEW_TARGET context must continue forbidding overwrite |
| Actual managed replacement/refusals/restoration/cleanup and new-source CI | NO | NO | NO | NO | MISSING meaningful RED/GREEN, canonical/related Race, actual exact-tag asset campaign and exact remote checkout/native/old consumers/CPU/cleanup |

Overall new replacement: NO/NO/NO/NO. Only the last four gaps are implemented.
No implemented resolver/verifier/inspection/Policy/staging is rewritten. The old
no-overwrite constructor stays usable and strict; production composition extends
its publisher with receipt/state support, rather than replacing that publisher.

## Exact owner/edit map

| Owner | Bounded extension | Required evidence |
| --- | --- | --- |
| core/domain/install.go and install tests | Add current-lane managed GitHub asset mode/constructor; preserve all existing modes, Valid and RequiresNewTarget meaning | Old NEW_TARGET remains true; new mode false; foreign/zero modes invalid. No generic Core port/interface or future placeholder |
| artifact/githubrelease client/reference helpers | Expose bounded normalized reference/repository binding from the already validated frozen resolved locator, not lossy owner-repo display identity | Hyphen collision, exact owner/repo/tag/asset/numeric-ID binding, malformed locator, no second resolve/read; unchanged original parser/acquisition tests |
| promotion/githubrelease_promotion.go and scoped receipt/session/transaction helpers | Extend existing exact staged publication with optional configured independent state/Evidence and installed marker; implement existing InstallUpdatePreparation/Session for --replace | Direct RED for missing receipt/managed replacement; original no-overwrite retains refusal; new production installs record ownership without altering bytes/format/inspection |
| promotion existing guard/member/rooted no-replace/sync helpers | Reuse anchored parent guard and existing bounded member snapshots; protect exact old target and independent state, publish only selected staged asset, rollback conservatively | Every original/staged/state/parent swap, symlink/hardlink/mode/time drift, cancellation, concurrent guard, interrupted backup; unknown foreign files survive successful/failing cleanup |
| application existing InstallUpdateService | Reuse directly; no source edit unless an actual required gap is demonstrated | Existing ordering/ALLOW/close failure tests plus GitHub session binding; no inspection/staging skip or success after late failure |
| cli/root.go, presentation/catalog/translations and existing install report | Add --replace static validation and per-request new mode; bind update service only for explicit intent | Explicit --target required; unsupported adopt/latest/native flags refuse before runtime; old new/default target still strict; EN/KO explanations, seven-language core help and JSON same facts |
| bootstrap/bootstrap.go GitHub composition | Reuse same client/verifier/inspector/Evidence/Policy/staging/publisher; bind managed initial publisher and per-request replacement service | Static help without runtime, no shared mutable session, old install/inspect unaffected |

Initial approval records must retain exact selector/resolved identity/declared
digest, asset size/mode/hash, Manifest identity, required completed checks, entry/
set ALLOW and real Evidence references/file digests. Revalidate old record through
the existing trusted Evidence reader and exact installed footprint; marker alone,
a matching filename/hash or a caller-made Manifest cannot create ownership.
Keep bounded canonical state bytes, identities and target membership guarded
through selection/inspection/staging/commit/rollback. No raw REST types in Core.

## Qualification and cleanup

1. Commit leaf/plan and second review before source. Meaningful RED for missing
   receipt/replacement admission, original drift and foreign-backup preservation.
2. Focused Domain/artifact/promotion/Application/CLI/bootstrap tests. Retain old
   constructor/new-target behavior and shared npm/PyPI/Go/Cargo/Terraform consumers.
   Explicit format, canonical Host Quick/security/docs and related Race; all
   complete input inventories match the candidate build.
3. Actual installed exact public archives from two tags: initial managed install,
   explicit replacement, fresh same-reference reinspection, changed asset name
   within same repository, EN/KO/JSON; refuse missing --target, target not managed,
   marker-only/old receipt missing/corrupt Evidence, foreign file/alias, repository
   change, concurrent state/target and failed new selection without target changes.
   Exercise the existing isolated ELF consumer and its negative coverage as well;
   archive success must not be presented as execution of embedded binaries.
4. Existing canonical lifecycle, protected originals and fresh root guards. Preserve
   first failures plus normalized Evidence/Manifest/SBOM/approval/installed inventory
   and exact operators. Confirm helper wait0/restoration/idle references, then delete
   all completed payload/intake/cache/staging/lifecycle/candidates/operators. Keep
   only active common tooling and protected originals.
5. Normal draft PR38 source push; ten SUCCESS/Required, exact actual checkout tree,
   native macOS and old Linux consumers/CPU/confirmed cleanup. Hold doc-only push
   during CI. Close only documented acceptance; no main merge/tag/publication.

## Second review before activation

Rejected shortcuts: enabling overwrite in NEW_TARGET, inferring ownership from
an existing basename/digest, storing approval only inside the target, re-resolving
old/new tags at commit, splitting ambiguous owner-repo display identity, and
reusing a successful old inspection as new Evidence. --replace is explicit
authority to replace a proven managed directory, not to adopt arbitrary files.

Existing installations without an independent receipt remain unproven; no safe
retroactive ownership fact exists in the old publisher. Refuse and explain a new
managed target instead of fabricating migration authority. Old normal install's
no-overwrite is an invariant, so new request mode is necessary; the existing
InstallUpdateSession and ordering already supply the workflow and need no new
generic port. All unknown original/state/recovery members are preserved. Ordinary
rollback must verify ownership before restoring/removing anything and retain a
recovery directory if foreign drift prevents safe completion. Promotion remains
non-networking/non-executing and does not reevaluate Policy. Initial receipt
publication and every failure point need the same target/state recovery review
as replacement; metadata must never approve bytes that were not published.

This review is a plan, not implementation/qualification. Activate only after
Terraform CI342 acceptance, then recheck audited inputs for intervening changes.
