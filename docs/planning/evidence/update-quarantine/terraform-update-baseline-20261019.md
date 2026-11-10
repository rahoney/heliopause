# SEC-UPD-001 — Terraform upgrade baseline and edit plan / 2026-10-19

Pre-source audit at4231370 (runtime source394c92d). Go CI340 remains pending;
this is read-ahead planning only. Terraform source changes wait for Go closure.
Only SEC-UPD-001 is IN_PROGRESS. Existing unrelated notes/raw remain unstaged.
The [update contract](../../../user-journey-cli-ia/06-update-quarantine.md) and
[M12 Provider contract](../../16-m12-01-ecosystem-expansion-contract.md#6-m12-004--terraform-provider)
own behavior. No implemented resolver, verifier, inspector or transaction is replaced.

## Independent baseline audit

| Responsibility | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | Evidence / MISSING |
| --- | --- | --- | --- | --- | --- |
| Exact public selection, frozen registry/package acquisition | YES | YES | YES existing init | YES existing scope | artifact/terraformprovider Resolver.ResolveSnapshot, Client.Resolve/Acquire/freezePackage; reuse discovery/version/package bounds and frozen bytes |
| Complete guarded configuration/lock and local-module capture | YES | YES | YES existing init | YES existing scope | project.go CaptureProject/Requirements/ParseLock; project_resolver.go ResolveProjectDependencyUpdate; current request has one exact primary and retains other locked versions |
| Signer/checksum/static/isolated probe, fresh Evidence and complete-set Policy/cache | YES | YES | YES existing init | YES existing scope | bootstrap/terraform.go existing composition; verification/terraformprovider, inspection/terraformprovider and promotion/terraform_cache.go; preserve all gates |
| Independent approval and guarded full provider publication/rollback | YES | YES | YES existing init | YES existing scope | TerraformProjectPromotion.Begin/verifyRetained/Commit and terraformProjectTransaction; existing fault/mode/foreign-backup tests; reuse |
| Current-project newest constrained selection without a primary | NO | NO | NO | NO | MISSING canonical stable selection from one frozen versions response per provider; full snapshot constructor already exists from Cargo, no fake primary/edges |
| Explicit upgrade enrollment and retained control identity | NO | NO | NO | NO | MISSING request-scoped BeginUpgrade; refuse implicit native/changed controls and foreign provider installation; keep old independent approval/cache/Evidence verification; preserve identical lock inode/mode/mtime |
| Native-shaped flag, Application/bootstrap and human/JSON update result | NO | NO | NO | NO | MISSING zero-positional init -upgrade, early strict normalization/validation, existing complete-project workflow/report reuse |
| Actual upgrade/refusals/old init and new-source CI | NO | NO | NO | NO | MISSING local installed campaign, canonical checks and actual exact-checkout CI/cleanup |

Overall new upgrade: IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO. MISSING is the last four rows only. Existing init behavior
is not rewritten and prior Go/Cargo/PyPI qualifications retain their own source.

## Selection and ownership decisions to add to the canonical leaf

Support `helox terraform init -upgrade [--adopt]` and `--upgrade` alias, no
positional provider. Normal exact `init namespace/type@version` stays unchanged.
The single native spelling is normalized only for the exact standalone token
in the Terraform init argv; never turn arbitrary single-dash flags into long flags.
Reject -upgrade=false/value, positional refs with upgrade, duplicate upgrade,
--adopt without upgrade, backend/module/mirror/lockfile/chdir flags and custom
runtime/source/configuration before runtime composition. Existing helox global
output/language/color options continue to work; upgrade allows explicit JSON.

Official [init documentation](https://developer.hashicorp.com/terraform/cli/commands/init#plugin-installation)
defines upgrade as ignoring old provider selections and taking the newest version
allowed by configuration. We implement this within the existing literal public
provider subset, without invoking Host Terraform. Existing pinned x/mod semver
and MatchesConstraint own comparison/constraint behavior; no new runtime/dependency.
Initial upgrade accepts stable releases and stable constraints only; explicit
prerelease constraints refuse. Validate all version identities/duplicates/bounds,
select the greatest matching stable version, then validate that exact selected
platform/protocol. Do not silently downgrade to an older platform-compatible
version. Every aggregated reachable-local-module constraint must hold. Preserve
configuration bytes and local-module identities; no remote module/backend work.

Select all required independent providers (1..32) from guarded controls, even if
old lock is absent. Existing old lock must parse and contain no orphan provider;
its selections never constrain a changed version. Same selected version still
requires existing lock hash agreement; retain its other platform hashes. New
versions use only independently verified new package hashes. Freeze discovery,
versions and exact package together once per provider; no second versions request
or latest re-resolution inside Client.Resolve/Acquire. All selected providers
receive new existing required signer/static/probe checks and Evidence, even no-op.
An empty provider configuration refuses explicitly rather than inventing an artifact.

Managed projects require the existing independent canonical approval/cache/
Evidence and complete installed tree. --adopt may deliberately supersede old
control equality only; still verify the old marker, canonical approval, cache,
Evidence and actual installed tree independently. Changed controls without --adopt
refuse. A providerless native configuration/lock may enroll only with --adopt.
Existing unmanaged .terraform providers/modules/backend/cache, marker without
independent state, corrupt/aliased authority or incomplete recovery refuse even
with --adopt. Native installed-provider ownership is not inferred from lock hashes.

Only upgrade mode admits the already existing snapshot-only update constructor;
old exact Commit retains requested-resolution admission. Reuse the same
transaction and full tree replacement. Keep an identical selected lock in place
with bytes/inode/mode/mtime; preserve ordinary existing permissions on changed
lock, absent new lock0600. Retained lock gets the same phase/postcheck/rollback
guards and no authority is transferred from its checksum. Configuration is never
published. No source/runtime/observer/budget/Policy/CI-job changes are needed.

## Exact in-place edit map and tests

| Existing owner | Minimal change | Required regression |
| --- | --- | --- |
| artifact/terraformprovider/provider.go, resolver.go, client.go | Shared validated stable selector and frozen snapshot-to-artifact binding; resolve upgraded package using its original discovery/version bytes | Newest unordered numeric versions, all constraints, duplicate/oversize/malformed lists, unsupported latest platform/protocol, prerelease refusal, cancellation; one discovery/versions/package traversal |
| artifact/terraformprovider/project_resolver.go | Extract current common complete selection; old exact route unchanged, ResolveProjectLockUpdate accepts empty selector only and returns snapshot-only selection | Two independent upgraded providers, local-module aggregate constraints, unrelated lock orphan, absent lock, same-version mismatched hash, full requested/cache subject binding; exact old init unchanged |
| application/terraform.go | Add TerraformUpgradeService using existing ProjectLockUpdateResolver and executeGuardedProjectUpdate | Full inspection/staging/guard/close required, no-primary form required, harmful/incomplete/source-swap and primary failure facts preserved |
| promotion/terraform_approved_transaction.go | Begin wrapper remains strict; BeginUpgrade request mode and explicit control-only enrollment; snapshot form admission scoped to mode | Native implicit refusal, providerless explicit enrollment, changed controls opt-in, corrupt/foreign/aliased authority and recovery refusal; old independent cache/Evidence still checked |
| promotion/terraform_transaction.go and existing published verifier | Upgrade-only lock retention/permission state; recheck identity/content/mode/mtime at original/backup/publication/approval/rollback boundaries | Direct RED for identical-lock replacement; changed mode preservation; every existing fault phase, retained foreign drift and backup sentinel survival; old exact consumer retained |
| bootstrap/terraform.go, bootstrap.go | Reuse the one actual client/verifier/inspector/Evidence/cache/promoter composition; per-request adapter chooses BeginUpgrade | Correct command dispatch, no mutable global adoption intent and no extra runtime setup on invalid args |
| cli/root.go, terraform upgrade helper, existing shared report/presentation | Exact standalone -upgrade normalization before parsing; static validation and wrapper around old init; full project update report terraform-registry | Help without runtime; EN/KO/fallback; JSON identities/checks/Evidence, display failure/stage/commit/close never success; false/value/duplicates/extra flags and exact init regression |

Do not add a generic upgrade resolver port: the existing ProjectLockUpdateResolver
already expresses this complete selection. Do not relax HasRequestedResolution
for ordinary init or change Core/Domain. Any demonstrated new gap needs an audit
amendment before widening this map.

## Qualification and cleanup plan

1. Commit reviewed plan/leaf before production edits. Add meaningful selection,
   admission and lock-retention REDs; preserve original failing output.
2. Focused artifact/promotion/Application/CLI/bootstrap and affected old Go/Cargo
   consumers; canonical explicit format, Host Quick, security, docs and relevant
   Race with exact complete input inventory. No floating tool replacements.
3. Freeze new CLI and source. Reuse installed canonical lifecycle with protected
   originals, ordinary UID, fixed observer/helper/policy, confirmed helper wait0
   and fresh before/after queries. Actual official and supported partner fixture:
   old version→new constrained version, fresh no-op, multiple independent roots,
   local module constraint, explicit native control-only enrollment and changed
   controls; unsupported/foreign/corrupt/constraint errors leave full target intact.
   Exercise old exact init and existing probe/signature negative consumers.
4. Capture every result/check/Evidence/cache/approval/control/output receipt before
   deleting completed target/cache/intake/lifecycle/operators/candidate binaries.
   Retain only active shared tooling/observer/source cache and protected originals.
   UNKNOWN is not absence and never authorizes restoration or cleanup.
5. Normal draft PR38 source push, then all ten CI jobs/Required, exact checkout
   tree and native macOS, actual old consumers/CPU/helper cleanup. No source edits
   during checks, no needless full CUDA rerun for untouched runtime/Python scope,
   no doc-only push interrupting active CI. Close only this documented lane, then
   begin independent GitHub replacement audit. Launcher remains deferred.

## Second review before source

The initial temptation to choose one arbitrary required provider as a primary is
rejected: existing snapshot-only constructor/workflow preserves independent roots.
A second live versions read after latest selection is rejected: exact frozen
versions must enter the same signed intake subject. Native installed .terraform
adoption is rejected without independent ownership, even if a lock hash matches.
--adopt is not permission to skip old approval/cache/Evidence checks. Retained lock
must branch before destination-absent publication, and remain represented in the
selected private backup inventory until removed by verified cleanup. Every phase
still checks the original retained file and preserves foreign changes on failure.
No old normal init guard or signing/probe limit is broadened. Review is a design
check only; all new acceptance flags remain NO until implementation and evidence.

## Activation after Go acceptance closure

Go sourcee00/CI341 is now qualified: all ten jobs SUCCESS, exact checkout tree,
36 existing Go/Cargo/Terraform consumers, CPU192.40s/five helperwait0/final
confirmed cleanup. Current documentation head1b38582 has no tracked source edits.
Between original baseline4231370 and this head, only the two audited Go required-
metadata files changed under internal; every Terraform/composition/report owner
audited above is unchanged. Thus this independent edit map/second review170140c
still applies. Terraform upgrade remains NO/NO/NO/NO with exactly its listed
MISSING work; existing exact init/proof/probe/cache/transaction remain reused.
Only SEC-UPD-001 is IN_PROGRESS. Proceed with direct REDs, then the scoped source.
