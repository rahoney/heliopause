# SEC-UPD-001 — public PyPI managed-venv local qualification / 2026-10-19

Base145db23 on milestone/m13-release-build-packages. This record covers the
public-PyPI managed-venv implementation after the independent
[baseline and selection review](./pypi-baseline-20261019.md). Exact production
source: local raw pypi-admitted-quick-source.json. Subsequent documentation changes
do not alter that Go source. No public release, permanent installed activation,
or main merge is claimed. The subsequent c93ce58 remote result is recorded below.

| Scope | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bounded public-PyPI managed venv upgrade | YES | YES | YES local only | NO for complete PyPI lane | New remote/native-platform qualification; native and named-source work below |
| Independent native venv enrollment | NO | NO | NO | NO | Authenticated original footprint/ownership proof; currently rejects unchanged even with --adopt |
| Named PyTorch upgrade | NO | NO | NO | NO | Independent profile selection/ownership audit, implementation and affected CPU/CUDA qualification; rejects before composition |

## Existing responsibilities reused

InstallUpdateService rebinds request-local copies of the original InstallService;
required inspections, Policy, Evidence, derived-wheel build/input binding,
Manifest, staging and Promotion retain their existing authority. PyPIResolver
shares its original isolated lifecycle, egress policy, Simple cross-check,
complete report/graph construction and confirmed cleanup. pythonSourceResolver
and prerequisiteCheckingResolver forward typed updates with prerequisite
validation; ordinary and unsupported-platform paths remain.

PyPIUpdatePreparation acquires the original pypiVenvTransaction before inventory
and selection. It freezes config/ledger/owned files, directory identities and
bounded inert METADATA. Native RECORD is never deletion authority. The public
resolver runs only-if-needed against data-only installed metadata, then selects
the complete requested-root closure constrained by frozen proposed versions and
unchanged actual requirements. Input/delta/final report, exact graph, runtime and
report digest bind one attempt. No resolution occurs after ALLOW.

Promotion uses that same held transaction, existing private offline wheel
installation, exact output validation and rollback. Original state/cancellation
is rechecked before commit. Resolver-owned nodes/edges cannot change; only the
existing authenticated source-to-derived-wheel extension is permitted. External
build inputs need not be resolver graph nodes. Small metadata reuses the bounded
fingerprint with an explicit1MiB limit; npm retains64MiB and large Python payload
fingerprinting is unchanged.

## Checks and actual consumers

Final frozen candidate: canonical Quick95.83s, security6.16s, docs0.27s and
targeted Race21.43s PASS. Quick includes format/runtime/module drift, production
and test builds, architecture/workflow validation, vet, pinned Staticcheck and
the full default suite. Race includes new PyPI input/resolver/session/CLI/
prerequisite/derivation tests and existing venv/npm update/generic pipeline
regressions. Receipts/inventories use pypi-admitted-* under local
`docs/github-logs/update-quarantine-20261018/`.

Controlled tests prove0.9.0→1.0.0 replacement, obsolete owned file removal,
unrelated retention, early metadata/payload/namespace/lock drift, cancellation,
partial-publication rollback preserving its cause, Manifest/staged tampering and
legitimate/tampered derived binding. Unsupported legacy/editable/oversized/symlink
metadata and unproven native ownership reject. CLI tests cover --upgrade/-U,
retained requirements, UPDATE/FAILED JSON, seven-language help and unsupported
flags/source/target rejection before runtime composition. Synthetic tests are
separate from actual CLI/observer results.

Pinned Go1.26.9 CGO0 build1.88s produced26,844,861B CLI SHA
`c6c364ac01a0152c4f053874ec1b9d75372c491a8020669810a6aadd08435343`.
The unchanged already-built observer SHA
`dbc79f6a08b52c88be3c17cbf04e31c39131a92617bec05dcb5e06f90c7e7a49`
was used through the canonical authenticated installed path. All seven UID1000
CLI operations passed: ordinary packaging24.2 installation; implicit native and
unproven --adopt rejection with unchanged files; managed only-if-needed
packaging>=24,<26 selecting25.0; exact native-U/default-VIRTUAL_ENV no-op with fresh
Evidence; human update; unsupported eager strategy with unchanged files.
Successful JSON installs/updates reported COMPLETED/ALLOW, three required checks,
three Evidence references and COMPLETED Promotion. No-op reused no old inspection.
The empty3.14 venv was prepared using only the pinned trusted guest without pip;
preparation is not quarantine or native enrollment qualification.

## First actual failure and correction

Earlier cb544bb passed local checks and actually updated24.2→25.0, but exact no-op
preparation failed. Original transaction cleanup retains empty obsolete dist-info
directories; new inventory incorrectly required METADATA in every directory.
TestPyPIUpgradeRecognizesEmptyRetiredMetadataDirectoryAndGuardsItsContents directly
reproduced it in pypi-retired-metadata-red2.43s. Inventory now excludes only truly
empty directories from installed distributions while preserving their identity
and emptiness checks. New contents or nonempty missing-METADATA state still reject.
The regression/new PyPI suite passed green4.39s, followed by final canonical checks
and all seven actual consumers above. Old checks are not transferred to this fix.
Raw pypi-reviewed-failure-assessment.json preserves the first cause.

## Restoration, cleanup and remaining scope

Both trials used the unchanged canonical helper lifecycle. Final helper
wait0/CONFIRMED_STOPPED, observer restoration and client_restore0/RESTORED were
confirmed. Fresh root postcheck verified four original hashes/owners/modes,
installed symlink, process/socket/Docker absence, runtime directories and durable
rollback. Installed CLI remains the prior language-enabled original.

Original Evidence/Manifest/SBOM and operators were preserved. After process/open
reference checks, completed first-trial54,134,660 and final-trial54,454,489 logical
bytes were deleted: both fixture/cache trees, test binaries/lifecycle copies and
all five one-use PyPI operators. Active reusable source/cache/observer, shared
tools/runtime and necessary protected rollback remain for continuing work. Raw
pypi-reviewed-trial-cleanup.json and pypi-admitted-trial-cleanup.json own counts.

Complete PyPI and SEC-UPD-001 acceptance stays open. Native enrollment, named
PyTorch/affected qualification and later ecosystem lanes remain missing. These
results do not qualify native macOS, new remote CI, CUDA or general pip argument
compatibility. CLI-UX-003 and publication stay deferred.

## Subsequent same-source remote qualification

Commitc93ce58db3ef13a72c9a2667eec5aecc5693bf7a passed CI331/run39148017283:
Required and all nine prerequisites SUCCESS. Quick, native macOS and Linux logs
identify checkout03867cef0aeceaa47a472144dc862866bd246fd5, whose full tree
138f0d291801789b0013293ef822f9396bea7419 equals the pushed source tree. Linux
passed45 top-level consumers (44 TestLinux and the public Terraform source test),
including CPU PyTorch full228.80s, and five helper waits0 with final original0,
CONFIRMED_STOPPED and client_error0/client_restore0. The parser's first local
receipt attempt counted only the44 TestLinux names and stopped before writing a
receipt; final count includes the separately named public source consumer.
Original logs were unchanged. This is source/platform regression qualification
for the managed implementation; the dedicated seven-operation update trial above
is local evidence. It does not qualify newer native enrollment source or named
PyTorch updates/CUDA. Local raw ci331-completed-qualification.json, completed jobs,
actual-checkout response and three decoded job logs own this linkage.
