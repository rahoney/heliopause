# M6 Entry Decision — GitHub Releases Standalone Contract

M6는 package manager를 Core/Application/Policy에 추가하지 않고 public GitHub
Release asset 하나를 기존 Artifact → Verification → Inspection → Policy →
Verified Set → Manifest/SBOM → Staging → Promotion 흐름에 연결한다. GitHub REST
payload, release API, attestation CLI와 archive library의 고유 type은 Adapter
안에만 둔다.

## 1. Supported input and exact identity

M6 public CLI input is exactly:

```text
helox github inspect <owner>/<repo>@<tag>#<asset>
helox github install <owner>/<repo>@<tag>#<asset> --target <new-absolute-directory>
```

- `owner`, `repo`, `tag` and `asset` are required, bounded, normalized adapter
  input. `asset` is one basename only; slash, backslash, control character,
  whitespace ambiguity, URL, query/fragment and glob are rejected.
- `latest`, omitted tag/asset, release ID-only selection, source-code
  `zipball`/`tarball`, redirect URL, arbitrary direct URL, Git clone, private
  repository and environment-derived selector are outside M6 automatic scope.
- Resolve uses `GET /repos/{owner}/{repo}/releases/tags/{tag}` with the pinned
  GitHub REST API version. It requires a published, non-draft release whose
  returned `tag_name` equals the exact requested tag and exactly one uploaded
  asset with the exact requested name.
- The normalized resolved coordinate binds repository, release numeric ID,
  exact tag, asset numeric ID, asset name, declared size and declared
  `sha256:<hex>` digest. Missing/unsupported digest, duplicate asset name,
  released asset state other than `uploaded`, size outside the M6 bound, or a
  malformed response is operationally incomplete and never reaches `ALLOW`.
- A release may be GitHub-immutable or mutable. Immutability is normalized as
  provenance Evidence, not a shortcut to trust. Even an immutable release is
  verified against acquired bytes; a mutable release is safe for one operation
  only because Resolution is frozen before Acquisition and Promotion never
  resolves the reference again.

M6 supports public unauthenticated REST access only. The known public limit is
60 requests per hour per originating IP. Rate limiting, API error, redirect
failure or incomplete metadata fails closed; M6 does not read a token, proxy,
GitHub CLI credential or ambient configuration. Optional authenticated access
is a later explicit contract, not an implicit fallback.

## 2. Controlled acquisition and integrity

The adapter acquires only the resolved asset ID through GitHub's documented
release-asset API media flow. It uses a bounded HTTPS client, a fixed API
origin, bounded redirect count and strips any authorization state before a
redirect. The downloaded bytes are streamed only into HAA Controlled Intake;
they are not opened, extracted or written to the requested target by the
adapter.

- controller computes observed SHA-256 and exact byte size while streaming;
  both must equal the resolved API declaration before an `Acquired Artifact`
  is returned.
- redirect target, content-length disagreement, oversized stream, non-regular
  intake output, duplicate intake destination, digest mismatch or cleanup
  uncertainty returns no acquired Artifact.
- GitHub checksum, signature and provenance sidecar assets are **not** inferred
  by filename, extension or release text. M6 records GitHub API digest and
  immutable-release metadata as declared provenance only. A later verifier may
  consume a caller-selected, typed sidecar or cryptographically verified
  attestation; raw sidecar text and an unverified attestation cannot create
  `ALLOW`.

## 3. Format, static inspection and dynamic capability

M6 recognizes one acquired asset as `executable`, `zip` or `tar.gz` only when
bounded magic/type detection and the selected filename agree. Content-type is
sanitized metadata, not a security decision.

- zip and tar.gz inspection uses bounded readers only: archive/file count,
  compressed/uncompressed size and nesting depth are limited; duplicate path,
  absolute path, traversal, symlink, hard link, special file and archive-bomb
  shape are rejected. Archives are retained as opaque promoted bytes; M6 does
  not extract them into the target.
- an executable candidate must be a Linux amd64 ELF with bounded static header
  and interpreter/dependency metadata. PE, Mach-O, script, unknown binary and
  platform mismatch are `MANUAL_REVIEW`; no Host execution occurs.
- only a verified Linux amd64 ELF may receive M3's existing gVisor dynamic
  session. It is copied once to target-local sandbox storage, has no Host
  mounts/network/credentials, and is observed by the trusted observer. Missing
  capability, timeout, nonzero/unknown lifecycle, observer loss, incomplete
  stream or resource limit is required-check incomplete and cannot `ALLOW`.
- no executable is dynamically run merely because it is embedded in an archive.

## 4. Policy, Manifest and Promotion

M6 has one primary graph node and no package dependency resolver. It still uses
the common entry Policy and complete-set Policy; `ALLOW` requires completed
source digest verification, static inspection and, for Linux ELF, completed
dynamic inspection with no blocking Finding. Unsupported format/platform or a
missing required capability yields `MANUAL_REVIEW`; digest/security Findings
yield `BLOCK`.

M6 deliberately keeps the generic Verified Set, deterministic Manifest/SBOM
and trusted Staging path. The standalone exception means Promotion installs no
dependency tree, not that it may bypass the quarantine-to-staging rehash.

```text
exact acquired asset → verify/inspect/ALLOW → Manifest/SBOM → Staging rehash
→ exact staged asset rehash → new target directory/<selected asset basename>
→ atomic no-replace publish
```

- target is a new canonical absolute directory; an existing target, parent
  identity/symlink change, target race, permission uncertainty or cleanup
  failure leaves the target absent or untouched.
- promotion copies the exact staged bytes under the selected basename, sets a
  non-writable regular-file mode (and executable mode only for a verified ELF),
  fsyncs content and directory, rehashes, then publishes atomically. Sandbox
  never receives target-path authority.
- Promotion receives the staged Manifest-bound asset only and never calls the
  GitHub API, follows a Release URL or downloads another byte sequence.

## 5. M6 implementation queue

| Order | ID | Scope | Recommended model |
| --- | --- | --- | --- |
| 1 | M6-002 | exact public input parser, API response boundary and capability | Luna High |
| 2 | M6-003 | GitHub release resolve/acquire, declared SHA-256 verification and intake | Terra High |
| 3 | M6-004 | executable/zip/tar.gz bounded static inspection and M6 Policy | Terra High |
| 4 | M6-005 | Linux ELF dynamic inspection and trusted-observer wiring | Terra High |
| 5 | M6-006 | Manifest/Staging-bound standalone Promotion, CLI/bootstrap and E2E | Terra High; Luna High for fixtures |
| 6 | M6-007 | qualification, npm/PyPI regressions and M7 handoff | Luna High; Terra Medium for security audit |

M6-001 creates no product package, runtime/tool, external dependency, CLI
command or CI job. Each later item is opened only after its predecessor meets
its acceptance criteria.

## 6. Official decision basis

- [GitHub Releases REST API](https://docs.github.com/en/rest/releases/releases) — exact tag endpoint, release/asset identity and SHA-256 asset digest
- [GitHub REST rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api) — unauthenticated public request limit
- [Verifying release integrity](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/verify-release-integrity) — immutable release and local asset verification semantics
- [Artifact attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations) — provenance requires cryptographic verification and is not an independent safety verdict
- [Verify artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations) — verified attestation is an optional future input, not raw metadata
