# M2 Entry Decision — npm Static Inspect Contract

M2는 M1의 ecosystem-neutral `Inspect` lifecycle에 public npm registry의 resolve·controlled acquisition·정적 검사·로컬 Evidence 기록을 연결한다. npm의 입력/metadata 특수성은 Adapter에만 두고 Core/Policy에 npm type이나 registry HTTP payload를 노출하지 않는다.

## 1. 범위와 security boundary

M2 사용자 path는 다음 하나다.

```text
helox npm inspect <package-reference>
  → public npm resolve → exact identity → controlled tarball intake
  → declared-integrity verification → static tar/manifest inspection
  → trusted Evidence record → M2 Policy → result
```

- registry는 고정된 `https://registry.npmjs.org/`만 지원한다. custom registry, scope registry override, auth, proxy와 credential은 M2 범위 밖이다.
- Adapter는 package를 획득할 뿐 압축 해제·install·lifecycle script·Node 실행을 하지 않는다.
- untrusted tarball은 host project, Evidence Store, Staging 또는 Promotion에 쓰지 않는다.
- Dynamic Inspection은 M3에서 열며 M2 `inspect`는 static result가 clean이어도 `ALLOW`하지 않는다.

공개 registry의 abbreviated metadata는 `GET /:package`와 `Accept: application/vnd.npm.install-v1+json`으로 얻는다. 선택한 version의 `name`, `version`, `dist.tarball`은 resolve에 required이며 `dist.integrity`는 원문 없이 bounded declared-integrity input으로 보존한다. integrity의 누락·형식 오류는 resolve 오류가 아니라 §5의 completed `MISMATCH` verification으로 정규화한다. 이는 npm registry의 [Package Metadata](https://github.com/npm/registry/blob/main/docs/responses/package-metadata.md)와 [Public Registry API](https://github.com/npm/registry/blob/main/docs/REGISTRY-API.md)를 따른다.

## 2. npm reference와 exact identity

M2는 registry package만 지원한다.

```text
<name>
<name>@<exact-version>
<name>@<dist-tag>
@scope/name
@scope/name@<exact-version>
@scope/name@<dist-tag>
```

- name은 npm package name grammar를 Adapter에서 parse/normalize한다. case, whitespace, URL, local path, tarball URL, git spec과 alias는 M2 usage error다.
- selector가 없으면 `latest` dist-tag로 resolve한다. tag는 metadata의 `dist-tags`에서 one exact version으로 바꾼다.
- semver range (`^`, `~`, comparator, `*`, union 등)는 M2가 지원하지 않으며 명시적인 exact version 또는 tag를 요구한다.
- adapter는 percent-encoded package name으로 한 번만 metadata endpoint를 만든다. response version object의 `name`과 `version`이 resolved name/version과 정확히 같지 않으면 operational/contract failure다.
- resulting `ResolvedArtifactIdentity`는 `SourceID=npm`, `Name=<normalized npm name>`, `Version=<exact registry version>`, `Variant=tarball`이다. 사용자의 mutable locator는 `ArtifactReference`에만 남긴다.

## 3. Network contract

M2는 explicit `http.Client`/transport를 injection 받아 test에서 controlled server를 사용한다. production transport는 다음을 강제한다.

| Boundary | Decision |
| --- | --- |
| scheme/host | HTTPS + `registry.npmjs.org` only for metadata and tarball URL |
| redirect | disabled; 모든 3xx는 operational error |
| authentication | header, token, cookie, proxy credential 미사용 |
| metadata | response body 2 MiB, request timeout 10 s, JSON object only |
| tarball | response body 50 MiB, request timeout 60 s, `200 OK` only |
| response | unexpected content type, missing/duplicate required data, limit 초과, TLS/DNS/read failure는 operational error |

request context cancellation/deadline을 그대로 보존하며, timeout/limit은 `OperationError` code와 sanitized message로 반환한다. 외부 URL, response body, header, redirect location과 raw error를 result/Evidence에 출력하지 않는다.

## 4. Controlled Intake와 content identity

Acquisition은 configured user cache root 아래 `intake/<run-id>/`에만 기록한다. Adapter constructor는 test용 explicit root를 받고 production bootstrap은 OS user cache directory 아래 `heliopause/intake`를 명시적으로 제공한다.

- root/run directory permission은 `0700`, new tarball file은 exclusive `0600`이다.
- `MkdirAll` 후 canonical root containment를 확인하고 symlink를 거부한다. file은 temporary same-directory file로 stream한 뒤 close+sync하고 atomic rename한다.
- content handle은 host path가 아닌 `intake:<run-id>:tarball` opaque reference다.
- stream 중 SHA-256과 declared SRI가 사용하는 SHA-512을 동시에 계산한다. M1 `ContentDigest`에는 observed lowercase SHA-256만 넣고, SHA-512는 Verification Evidence로만 사용한다.
- failed acquisition, size limit, sync/rename/cleanup failure는 Run operational failure다. incomplete file은 deleted; deletion 자체가 실패하면 primary failure를 `%w`로 보존해 함께 보고한다.
- M2는 retention/background cleanup을 만들지 않는다. successful intake는 Run-local evidence lifetime 동안 유지하고 future retention policy가 결정될 때까지 implicit deletion하지 않는다.

## 5. Declared integrity Verification

`dist.integrity`는 `<algorithm>-<base64>` Subresource Integrity value다. M2는 public npm release가 제공한 SHA-512 SRI만 required integrity로 지원한다. npm은 publish 시 integrity SHA-512를 registry에 기록하며 client가 strongest algorithm을 사용한다는 [npm publish 문서](https://docs.npmjs.com/cli/publish/)를 근거로 한다.

- malformed/missing integrity 또는 SHA-512 이외 algorithm은 `VerificationReport`: required, supported, `COMPLETED`, outcome `MISMATCH`와 `M2_DECLARED_INTEGRITY_INVALID` finding/evidence로 정규화한다. operational error가 아니다.
- declared SHA-512와 acquired stream SHA-512 mismatch도 같은 정상 verification result/finding으로 기록한다.
- match는 `VERIFIED`; registry declaration 자체는 safety decision, signature 또는 provenance의 대체가 아니다.
- ECDSA registry signature, provenance와 attestation은 M2에서 `UNAVAILABLE`/limitation으로 명시하고 M3/M2 후속 policy까지 성공으로 해석하지 않는다. npm signature format은 integrity에 묶인다는 [npm registry signature 문서](https://docs.npmjs.com/about-registry-signatures/)를 참고하지만 M2 verifier는 구현하지 않는다.

## 6. Static tarball / manifest inspection

M2 inspector는 acquired `.tgz`를 stream으로 gzip+tar parse하고 host에 extract하지 않는다.

| Limit / check | M2 value |
| --- | --- |
| gzip/tar decoded bytes | 200 MiB |
| entry count | 10,000 |
| individual regular file | 20 MiB |
| entry path | 4 KiB, normalized `package/` prefix required |
| allowed entry type | regular file/directory only |
| denied | absolute/empty/`..` escape, duplicate normalized path, symlink, hardlink, device, FIFO, sparse/unknown type |
| manifest | exactly one `package/package.json`, 1 MiB maximum, valid JSON object |

`package.json`의 `name`/`version`은 resolved identity와 일치해야 한다. `scripts` object의 lifecycle key(`preinstall`, `install`, `postinstall`, `prepublish`, `preprepare`, `prepare`)는 command 내용을 실행하거나 raw content를 Evidence에 넣지 않고 key·bounded length만 normalized Evidence로 기록한다.

- unsafe archive/identity mismatch/malformed manifest/limit exceed는 static inspection `COMPLETED`와 blocking Finding이다.
- parser 자체가 trusted result를 만들 수 없게 실패하면 Go error/Run operational failure다.
- dependency graph, vulnerability database, diff, obfuscation scanner와 signature verifier는 M2 implementation queue에 포함하지 않는다. 각 부재는 M2 Policy가 `ALLOW`를 만들지 않는 이유가 아니라 explicit capability/limitation으로 결과에 남는다.

## 7. Evidence store와 Policy v2

M2 Evidence Store는 Intake와 다른 configured `evidence/<run-id>/` trusted root에 record 하나당 `0600` JSON file을 atomic write한다. record에는 normalized Evidence ID, Run ID, exact identity/digest, check ID, kind, summary와 store-computed SHA-256을 넣는다. `EvidenceReference`는 `evidence:<run-id>:<evidence-id>`이며 raw tarball, manifest script body, response/headers/host path를 포함하지 않는다.

M2 Policy identity는 `m2-npm-static-inspect`, version `1`이다.

1. integrity mismatch 또는 archive/manifest blocking finding → `BLOCK` (`M2_INTEGRITY_MISMATCH` 또는 해당 finding code).
2. required verification/static report가 incomplete/failed/unavailable/unsupported → `MANUAL_REVIEW / M2_REQUIRED_CHECK_INCOMPLETE`.
3. static required checks가 complete이고 blocking finding이 없으면 → `MANUAL_REVIEW / M2_DYNAMIC_INSPECTION_UNAVAILABLE`.

따라서 M2 public npm inspect는 `ALLOW`를 반환하지 않는다. `BLOCK`과 `MANUAL_REVIEW` 모두 inspect operation 자체는 `COMPLETED`이고 exit code는 각각 20, 10이다.

## 8. Result evolution과 fixtures

`helox.operation-result/v1` schema는 유지한다. M2는 기존 field에 npm-specific raw metadata를 추가하지 않고 generic checks/evidence/policy reason으로 표현한다. `artifact.resolved_identity.source_id=npm`, `variant=tarball`과 content SHA-256은 human/JSON에서 동일해야 한다.

Production source를 fixture server로 연결하지 않는다. contract/integration test는 injected in-memory `RoundTripper` 또는 허용된 환경의 `httptest` controlled registry와 generated tarballs를 사용한다.

| Fixture | Expected |
| --- | --- |
| safe static tarball + matching SRI | `MANUAL_REVIEW / M2_DYNAMIC_INSPECTION_UNAVAILABLE` |
| declared SRI mismatch | `BLOCK / M2_INTEGRITY_MISMATCH` |
| archive path/link/type/limit violation | `BLOCK` with exact normalized finding |
| metadata/tarball timeout or body limit | operational failure, no Policy after failure |
| malformed metadata, missing selected version, host/redirect violation | operational failure, no Run before resolve failure or failed Run after acquisition stage |
| unavailable future verifier | explicit check limitation, never `ALLOW` |

## 9. M2 out of scope

- private/custom registry, authentication, scoped registry override and npm config parsing
- semver ranges, aliases, URL/git/file specs and dependency resolution
- tarball extraction, lifecycle/install execution, Node/npm subprocess, Sandbox and Dynamic Inspection
- production scanner/vulnerability/signature/provenance/attestation integration
- install, Verified Set, Staging and Promotion
- cache eviction/retention, concurrent downloads, retry/resume and registry fallback
