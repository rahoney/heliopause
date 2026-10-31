# M13-003 actual protected-tag qualification — 2026-10-17

Status: **COMPLETE** for the [M13-003 build/package scope](../../18-m13-production-release-operations-contract.md#m13-003--canonical-release-build-and-os-packages).
IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES / MISSING=none.
[Machine-readable result](./qualification-20261017.json) binds exact identities,
portable signed metadata and original local receipt hashes. Baseline and preparation
reports are historical observations; this report supplies their later evidence.

## Actual source and build

PR37 was ordinarily merged under the owner's specific approval. New
[main CI327](https://github.com/rahoney/heliopause/actions/runs/39031676341)
completed Required plus all nine prerequisites successfully on main
`ef5ef0e0e0437a9083133674ef8b9e6d693b83a9`, tree
`5b513f5d555e1cb25d132af2c567a0970f646392`. The owner separately approved the
protected `v0.1.0` tag at that exact commit. It was created once without retagging.

The naturally triggered [Release Build 1](https://github.com/rahoney/heliopause/actions/runs/39036210335)
(attempt1) succeeded on that exact source: native assets, observer102 latches,
OS packages, runtime metadata, manifest, SBOM, checksums, attestations and candidate
upload. Five Go executables report Go1.26.9, the exact source and unmodified VCS
state. The observer's separately pinned build is verified by its build workflow.
Actions artifact12102301625 has109179043 bytes and SHA-256
`17610e5e793957771ae39b948e8042f231fe12313570d81f1efefde1a3704db2`.
It expires2026-10-24T13:29:00Z; it is an Actions candidate, not a public release.
Any later publication must recheck retention, exact bytes and the canonical gate.

## Verification and OS lifecycle

Manifest attestation was verified first with official fixed gh2.102.0. The existing
Go bundle verifier then checked exact version/source/run, role/platform, sizes,
digests, metadata and the complete directory set. All12 actual files separately
passed signature verification against the same authenticated multi-subject bundle.
Signer workflow, exact case-sensitive tag, signer/source commit, hosted runner,
OIDC issuer and actual run/attempt all match. The signed metadata is retained in
[v0.1.0](./v0.1.0/helox-release-manifest.json); executable payloads are not committed.

The original manifest also passed with fixed trusted roots and no network in a
separate namespace after UID/GID drop. Wrong source, wrong-case tag and changed
manifest each failed under the same offline control. Generic tamper stderr is
not used to invent a lower-level failure explanation.

Actual network-disabled, unprivileged Ubuntu24.04 and Rocky9 containers passed
native dpkg/rpm install, reinstall and removal. Payload hashes, root ownership,
0555/0500/0444 modes, versioned layout, symlink, unit guards, protected existing
caller config preservation and removal were checked. Native verify outputs were
empty in the final strict checks. The Ubuntu test root explicitly included package
documentation because its base-image dpkg configuration excludes most docs.
`helox --help` exited0; doctor correctly exited1 with unavailable runtime checks.
No service autoactivation, valid caller-config invention or Host runtime mutation
was used to obtain these results.

## Acceptance ownership and remaining release work

The contract's M13-003 section owns generated assets and package lifecycle;
[M13-005 and section7](../../18-m13-production-release-operations-contract.md#m13-005--exact-publication-and-clean-host-qualification)
own exact public publication and booted clean-Host helper/systemd/doctor/smoke.
The latter are **NOT_RUN**. This corrects the preparation Queue's combined
remaining-work list without relaxing either acceptance scope. Container package
success is not booted Host readiness. M13 overall remains IN_PROGRESS.

M13-004 starts with its own baseline audit. The owner has no npm/PyPI accounts yet;
account setup, official package names and protected OIDC publishing remain open.
No GitHub Release draft/publication, registry upload, Homebrew repository creation,
publish dispatch or environment approval was performed. The PR37 direct-to-main
exception and specific tag approval do not authorize another merge or publication.
C3's missing-CLA case remains NOT_TESTED/DEFERRED with full hostile QUALIFIED=NO.

All task containers, newly acquired OS images, downloaded candidate bytes and
one-use verifier executables were removed. Existing images, active runtime,
rollback, SDK/shared modules and necessary evidence remain. The final private
check-cache cleanup is recorded after its last use. Full original operator logs
remain local under `docs/github-logs/m13-003-20261017`; only bounded portable
metadata and receipt hashes are included here.
