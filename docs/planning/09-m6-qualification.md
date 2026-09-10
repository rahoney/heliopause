# M6 Qualification — GitHub Releases Standalone

## Scope

M6-007 verifies the GitHub Release standalone path against the common
Artifact/Verification/Inspection/Policy/Manifest/Staging/Promotion contracts.
This qualification records the final M6 security audit and the remaining
controlled CI evidence without overstating macOS-local Linux runtime coverage.

## Qualification checks

- exact selector and public API identity are bounded;
- API-declared SHA-256 and size are rechecked during controlled intake;
- ELF, ZIP and tar.gz static boundaries are bounded and non-executing;
- verified Linux amd64 ELF uses the trusted gVisor observer contract;
- incomplete observer/lifecycle/cleanup states cannot produce `ALLOW`;
- staged GitHub bytes are rehashed and published only to a new target;
- npm and PyPI paths remain green under common checks.

## Reproducible checks

```text
go test ./...
go test -race -timeout=5m ./...
go run ./scripts/check quick
go run ./scripts/check docs
go test -run=^$ -fuzz=FuzzParseReference -fuzztime=5s ./internal/artifact/githubrelease
git diff --check
```

Local qualification checks pass on the M6 branch. Linux gVisor E2E is
capability-gated in CI by `HELOX_GITHUB_RELEASE_INTEGRATION=1`; macOS does not
claim that evidence.

## Handoff boundary

The final audit reviewed redirect host policy, shared observer attribution,
ELF execution constraints, staged-target publication, cleanup uncertainty and
no-ambient-credential behavior. It tightened asset request time bounding,
ZIP special-file rejection and tar.gz size-overflow handling. Controlled Linux
CI has supplied the final external evidence required for M6 completion.

```text
Status: COMPLETE
Checks: full test/race; canonical quick/docs; GitHub selector fuzz; audit focused unit/race; GitHub Actions required suite
Evidence: M6-001..M6-006 commits, final static/intake audit fixes, pinned Linux integration wiring; workflow_dispatch run 32440274998 — Quick, Docs, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux Integration, Required success
Next: M7 MVP qualification entry decision — NOT_STARTED / Ready: Yes
```
