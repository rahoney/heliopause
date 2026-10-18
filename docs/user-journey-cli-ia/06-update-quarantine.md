# Update quarantine — SEC-UPD-001

Status: required prepublication work; implementation/qualification incomplete.
This leaf owns the user-requested update-command contract. It supplements the
installation journey, never weakens existing Policy or adds implicit Host execution.

## User-facing scope

Users must be able to request supported ecosystem updates through familiar commands
prefixed by helox, including npm update, pip install --upgrade and cargo update.
The exact supported package selection, constraints, arguments and flags must be
listed before implementation; original tools' entire command surfaces are not
implicitly supported. Go's existing selected-version get path is reused. Terraform
upgrade and GitHub asset replacement need their own explicit supported contracts;
GitHub has no assumed universal native package-update command. All lanes remain
inside existing public-source/artifact/runtime support, not new private registries.

A package originally installed by another tool must not be silently adopted as a
trusted HAA-owned installation. Detect the supported project/venv/receipt state;
provide a bounded explicit enrollment flow or reject it without changing files.
Foreign files, unmanaged paths and ambiguous ownership stay protected. Native
package-manager operations outside helox are not transparently intercepted.

## Required update behavior

An earlier allowed package name/version is not trust for its next release. Resolve
and freeze the proposed dependency graph in the existing controlled environment.
Bind source, exact version/variant and digest for every selected artifact. Inspect
all entries required by the existing complete-set contracts, including changed
transitive dependencies even when the directly requested version does not change.
Do not weaken required checks or substitute version/name/checksum presence for
inspection. Registry integrity can verify publisher-provided bytes without proving
that a compromised publisher's release is benign.

Only after all required checks and Policy allow the exact proposed set may the
existing trusted staging/promotion path replace managed files and dependency
controls. No fresh live re-resolution or original-tool network download after
approval; the promoted bytes must match inspected bytes. Lifecycle/build/import
execution remains in its existing isolated role, never the user's Host merely to
complete an update. Findings and Evidence must be preserved on failure.

Guard the original project/venv/ownership state, concurrent changes, files and
lockfiles. Unsupported input, failed/incomplete inspection, review or block must
leave the user's installed state unchanged. Promotion failure must preserve the
primary cause and provide confirmed rollback or explicit recovery-required state;
never print update success because inspection alone allowed the set.

## Acceptance

Reuse the existing Inspect/Install, complete selected-project inspection, exact-set
Manifest, staging, guarded transactions and ownership/recovery implementations.
Do not build a parallel scanner, global transparent interceptor or generic raw
command executor. Each newly wired ecosystem has independent acceptance evidence.

Prove: benign v1 to permitted v2; same-name/source new harmful-fixture version is
blocked; transitive-only change is inspected; missing/incomplete verification is
rejected; source/digest swap after inspection is rejected; concurrent or foreign
state is not overwritten; injected commit failure restores the original state or
reports explicit recovery-required failure. Use harmless controlled regression
fixtures for negative behavior. Perform actual supported positive CLI consumers,
meaningful negative consumers and affected canonical/runtime checks. Human/JSON
results must agree on policy, update/install status and any rollback uncertainty.
Remove completed private fixture/cache/payloads after evidence is preserved.

Current gaps/reuse evidence and ordered work are in the
[baseline audit](../planning/evidence/prepublication-gaps/baseline-20261018.md).
An earlier bounded install/UX qualification does not close this update acceptance.
