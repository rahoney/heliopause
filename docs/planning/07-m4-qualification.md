# M4 Qualification — npm Install and Promotion

이 문서는 M4의 recursive npm inspection, Verified Set, trusted Staging과 offline Promotion 경계를 자격 검토하는 completion record다. M5 구현을 시작하지 않는다.

## Qualification target

- Branch: `milestone/m4-npm-install-promotion`
- Qualified head: `1dc22b8`; GitHub Actions run `31786209026`
- Resolver/Promotion runtime: fixed `node:22.23.1-slim@sha256:6c74791e557ce11fc957704f6d4fe134a7bc8d6f5ca4403205b2966bd488f6b3`, bundled npm `10.9.8`
- Dynamic runtime: M3에서 고정한 gVisor `release-20260810.0`, Docker Engine `29.6.2`, containerd `2.3.3`
- Automatic Promotion platform: Linux amd64; macOS와 그 밖의 platform은 unsupported operational failure이며 자동 반입하지 않음

## Exit criteria audit

| Criterion | Evidence | Result |
| --- | --- | --- |
| reference-to-result vertical | `helox npm install --target`이 Install Context → locked resolution → entry별 Run/검증/정적·동적 검사/Evidence → set Policy → Manifest/SBOM → Staging → Promotion → machine/human result 순서를 Application composition root에서 연결 | PASS |
| exact set identity | resolver context digest, graph entry identity/SRI, acquired SHA-256, independent inspection records, deterministic Manifest ID와 CycloneDX 1.7 binding을 검증하며 Staging과 target에서 bytes를 다시 hash | PASS |
| no mutable re-resolution | Promotion은 registry reference를 호출하지 않고 Manifest-bound target-local tarball과 `file:.heliopause/artifacts/<sha256>.tgz` lock만 생성 | PASS |
| offline constrained install | pinned image를 `--pull never --network none`, read-only root, non-root, cap drop, no-new-privileges, resource limit로 실행하고 `npm ci --offline --ignore-scripts --bin-links=false`만 허용 | PASS |
| target isolation and atomicity | Docker는 sibling temporary target만 mount하며 post-install exact package/path/name/version/no-symlink 검증 후 Linux `renameat2(RENAME_NOREPLACE)`로 새 target만 publish | PASS |
| Staging isolation and atomicity | Intake/Evidence/Staging root 분리, no-symlink, exclusive private writes, rehash, read-only seal와 fsync 후 no-replace atomic publish; raced destination은 보존 | PASS |
| fail-closed decisions | resolver/runtime/entry failure는 set `ALLOW`를 만들지 않고 `MANUAL_REVIEW`/`BLOCK`은 Manifest·Staging·Promotion을 호출하지 않음 | PASS |
| failure result semantics | Staging/Promotion operational failure는 기존 Policy `ALLOW`를 보존하면서 operation/promotion `FAILED`와 sanitized code를 출력하고 partial target을 정리 | PASS |
| quality and integration | canonical quick/docs/platform, full race, JSON metaschema, real pinned Docker offline Promotion, gVisor lifecycle와 resolver network-policy Linux integration | PASS |

## Reproducible checks

- `go run ./scripts/check quick`
- `go run ./scripts/check docs`
- `go run ./scripts/check platform`
- `go test -race -timeout=5m ./...`
- `uvx check-jsonschema --check-metaschema schemas/operation-result-v1.schema.json`
- GitHub Actions run `31786209026`: `Docs`, `Quick`, `Minimum Go`, `macOS`, `gVisor Observer Helper`, `gVisor Linux Integration`, `Required` success

## Limitations and handoff

- M4 automatic install은 Linux amd64의 새 empty target만 지원한다. existing project, global/workspace install, arbitrary npm option, lifecycle script와 bin link는 자동 Promotion 대상이 아니다.
- Resolver만 사전에 검증된 public npm registry endpoint로 제한된 network를 사용한다. Promotion은 network가 없으며 public registry 상태를 성공 fixture로 사용하지 않는다.
- completed Staging은 audit/retry를 위해 유지한다. retention/garbage collection은 아직 자동화하지 않으며 failed temporary cleanup 불확실은 operation failure로 보존한다.
- macOS CLI는 build/test gate를 유지하지만 M4 automatic Promotion capability를 주장하지 않는다.
- 다음에는 M5-001 PyPI/pip Expansion entry decision 하나만 연다. PyPI adapter, pip Promotion 또는 M6 source는 이 qualification에서 만들지 않는다.
