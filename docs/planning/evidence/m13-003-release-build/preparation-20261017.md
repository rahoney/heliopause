# M13-003 preparation — 2026-10-17

Initial scope: approved S/C2 preparation, based on `25e480c70ac7030462de34e574756d2f93fe900b`.
[Baseline audit](./baseline-20261017.md) completed before source edits.
IMPLEMENTED=YES / WIRED=YES (source) / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO.

## Prepared behavior

- `scripts/release-packages` uses exact nFPM `v2.47.0` as a Go library. No new
  standalone quality tool or compiler owner. Runtime lock/images remain unchanged.
  The pinned graph includes necessary transitive MVS changes; whole-consumer
  checks, security and vulnerability scans were run rather than assuming isolation.
- Existing native builds and installer remain. New package roles are added to
  `scripts/release-manifest`; checksum generation and `--verify-bundle` bind the
  complete asset/metadata set to version, source SHA and run URL. Unknown names,
  roles/platforms, duplicate/missing/changed assets, symlinks, extra entries and
  hostile checksum paths fail closed.
- Build workflow generates packages after native payload/runtime metadata and
  includes them in manifest/checksum/attestation/upload. Publish authenticates
  manifest provenance first, then uses only the verifier output for all candidate,
  provenance, draft upload/binding and immutable-release verification consumers.
  It retains protected-main ancestry, Required success, narrow permissions,
  draft digest/size checks and quarantine behavior. No release workflow was run.
- Package layout/lifecycle is specified once in the
  [canonical contract](../../18-m13-production-release-operations-contract.md#m13-003--canonical-release-build-and-os-packages).
  Unit tests inspect actual Debian ar/tar and RPM bytes with independent readers,
  verify metadata/payload/ownership/modes and prove byte reproducibility after input
  mtime/path changes. Fixtures are controlled payload bytes, not public release
  artifacts or clean-host installation evidence.

## Checks and original evidence

Raw owner: `docs/github-logs/m13-003-20261017/` (local untracked original evidence).
`prepared-candidate-check/source-inventory.json` binds every source file to SHA-256.
Each profile has an unedited log and result JSON with command label, exit, duration
and log digest. All checks use actual repository-selected Go1.26.9/pinned tools.

| Check | Result | Scope |
| --- | --- | --- |
| format | PASS | Explicit canonical formatter, isolated source clone |
| focused tests | PASS | Real deb/rpm bytes, manifest negatives, native installer and workflow regressions |
| Quick | PASS | Whole production/test builds, go vet, pinned Staticcheck and default tests; approved host environment |
| security | PASS | Canonical connected-candidate security checks |
| vulnerability | PASS | Canonical final call graph and pinned module graph |
| workflow | PASS | Repository policy, YAML and pinned Actions semantic validator |
| docs | PASS | Connected-source documentation; final documentation delta checked separately |
| release-gate | PASS | Static policy/version/workflow checks; no permission to publish |
| platform | PASS | Native Linux production build/default tests; this is not cross-compilation |
| Darwin cross-test compilation | PASS | amd64/arm64 compile-only (`-exec /usr/bin/true -run ^$`); native macOS execution remains remote evidence |

Earlier `package-first-check/quick.log` is retained as FAIL: sandbox loopback
TCP/AF_UNIX socket restrictions. The same first candidate passed host Quick
(142.8s). This is an environment boundary, not a confirmed package/dependency
regression. `connected-check` predates the added native-installer regression;
its focused PASS is not substituted for final-source checks.

## Actual source CI and later C3 approval

Exact source `1ac1bf6b30d23815c879d132693d9eeed5ada57c` passed
[PR37 CI325](https://github.com/rahoney/heliopause/actions/runs/39023951914): all10
jobs SUCCESS, including Required. PR checkout `25c4b274a3ec3ec97a6afbc75e836fb2de648c84`
has the same whole tree `3af3728a846821ef72130aafaa49c8e2d0c53f00`. Actual Go1.26.9
native macOS default tests and Linux45 consumers/CPU full192.25s/helper5wait0/final
CONFIRMED_STOPPED with client_error0/client_restore0 were checked in10 original
logs. Raw receipt: `ci325-final.json`, originals/hash inventory: `ci325-proof/`.
No actual tag or release provenance is inferred from source CI.

The owner subsequently approved [C3](../m13-002-protected-path/cla-deferral-proposal-20261017.md).
The canonical M13-002 clause supersedes C2's preparation-only ordering. Original
missing-CLA evidence remains NOT_TESTED / DEFERRED, full hostile QUALIFIED=NO;
M13-002 acceptance is closed only within that explicit exception scope.

## Remaining qualification

M13-003 remains IN_PROGRESS / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO. Source CI is
complete; final C3 documentation/check delivery, reviewer-approved normal merge
and actual main CI remain before protected-source release qualification. Actual
tag/build provenance, clean-host OS install/lifecycle and immutable publication
qualification are NOT_RUN. Concrete source/version/tag/publish actions require
the existing candidate/reviewer approval; C3 is not execution permission.
M13-004/005 remain NOT_STARTED and require their own baseline audits in order.

Temporary native source, private compiler/Staticcheck caches and test payloads are
removed after their final use. Original logs, user files, shared verified modules,
SDK, active runtime and rollback material are preserved. Cleanup receipt is kept
under the same raw owner; no WSL/VHD compaction is implied.
