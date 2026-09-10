# M2 Qualification — npm Static Inspect

이 문서는 M2의 exact npm resolve·controlled Intake·integrity verification·static inspection·Evidence·fail-closed Policy를 현재 milestone 브랜치에서 재현한 completion record다. M3 Dynamic Sandbox나 M4 Promotion을 대신하지 않는다.

## Qualification target

- Branch: `milestone/m2-npm-static-inspect`
- Qualified implementation head: `5f12eb2` (M2-005 vertical implementation)
- M2 implementation commits: `b632367`, `0fef849`, `bec6483`, `2c5e53f`, `2aefb3b`, `703ebb1`, `9e5a555`, `5f12eb2`
- Required external state: none; all vertical fixtures use injected in-memory `RoundTripper`

## M2 exit criteria

| Criterion | Evidence | Result |
| --- | --- | --- |
| exact npm identity | resolver tests for scoped/unscoped name, tag and exact version; response name/version consistency | PASS |
| bounded public transport | fixed production registry, redirect/host/status/content-type/body/timeout guards; no credentials or proxy | PASS |
| controlled Intake | Run-local `0700` directory, temporary `0600` file, sync+atomic rename, SHA-256 and SHA-512 stream hashes, partial cleanup | PASS |
| declared integrity | missing/malformed/mismatch normalized as completed `MISMATCH` finding; match as `VERIFIED`; observed SHA-256 remains content identity | PASS |
| static archive inspection | bounded gzip/tar stream, path/type/duplicate/entry/file limits, exact manifest identity, lifecycle key+length normalization; no extraction or execution | PASS |
| trusted Evidence | Intake-separated `0700` Evidence root, atomic `0600` JSON records, record SHA-256 and opaque references | PASS |
| fail-closed Policy | safe static=`MANUAL_REVIEW`, integrity/archive finding=`BLOCK`, resolve failure has no Policy | PASS |
| controlled vertical workflow | actual resolver→Intake→verification→inspection→Evidence→M2 Policy tests for safe, mismatch, unsafe archive and operational failure | PASS |

## Reproducible checks

- `go run ./scripts/check quick` — PASS
- `go run ./scripts/check docs` — PASS
- `go run ./scripts/check platform` — PASS
- `go test -race -timeout=5m ./...` — PASS
- `uvx check-jsonschema --check-metaschema schemas/operation-result-v1.schema.json` — PASS
- `git diff --check` — PASS

## Boundary audit

- public registry mutable state is never required by test success; all vertical fixtures are in-memory.
- `internal/application` remains ecosystem-neutral; npm parsing/transport stays in the adapter boundary.
- untrusted tarballs are never extracted, executed, installed, staged or promoted.
- no `internal/sandbox`, `internal/staging`, `internal/promotion`, dynamic CI job or future runtime placeholder was added.
- human and machine result presenters consume the same immutable Operation Result; JSON retains schema `helox.operation-result/v1`.

## Limitations and handoff

- npm registry signatures, provenance, attestation, dependency graph and vulnerability databases remain unavailable or out of M2 scope.
- clean public CI run/PR merge evidence is external to this local qualification; the branch is pushed for CI review.
- M3 opens exactly one entry decision: Linux Dynamic Inspect isolation backend, observation contract, limits and dynamic Policy rule. No M3 implementation starts in this qualification.
