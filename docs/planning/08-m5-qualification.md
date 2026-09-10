# M5 Qualification — PyPI/pip Expansion

## Scope

M5-007 verifies the M5 exit criteria against the common Artifact/Verified Set,
Evidence, Sandbox and Promotion boundaries. The audit covers the PyPI wheel and
sdist paths, the existing npm regression path, unsupported capability handling,
and the required Linux/macOS CI gates.

## Security-boundary audit

- sdist source and every PEP 517 build wheel must have an entry `ALLOW` and all
  required checks `SUPPORTED`/`COMPLETED` before the builder starts.
- the builder emits a distinct `derived-wheel`; it never aliases the source
  sdist and never accepts network, credentials, Host mounts or Host Python.
- completed gVisor build observation collection is represented by a required
  normalized Evidence/check on the derived artifact.
- source digest, sorted build-input digests, bounded executor identity and
  build-system configuration digest are retained as generic graph binding and
  serialized in the deterministic Manifest.
- observer loss, cleanup uncertainty, ambiguous output, failed build or
  incomplete required check returns no derived artifact and cannot produce
  `ALLOW`.

## Reproducible checks

```text
go run ./scripts/check quick
go run ./scripts/check docs
go test -race -timeout=5m ./...
git diff --check
```

The focused and full race checks passed locally after commit `42858b1`.
The controlled Linux qualification passed in GitHub Actions workflow
`heliopause-ci.yml` on the `milestone/m5-pypi-pip` ref: [run 32352463714](https://github.com/rahoney/heliopause/actions/runs/32352463714).
Quick, Docs, Minimum Go, macOS, gVisor Observer Helper, gVisor Linux
Integration and Required all succeeded.

## M6 handoff

M5-007 is complete after the successful controlled Linux run and npm regression
evidence. M6 entry remains a separate decision item; no M6 adapter, placeholder
or dependency is introduced by M5.
