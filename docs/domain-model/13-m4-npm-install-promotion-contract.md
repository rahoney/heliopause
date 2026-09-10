# M4 Entry Decision — npm Install and Promotion Contract

M4는 M3의 `ALLOW` 가능한 npm inspection 결과를 실제 설치로 연결한다. 설치 대상은 primary artifact 하나가 아니라 exact dependency graph 전체이며, Promotion은 새 Artifact를 resolve·acquire·판정하지 않는 trusted 집행 경계다.

## 1. M4 MVP 범위

```text
helox npm install <package-reference> --target <absolute-directory>
  → isolated dependency resolution
  → each exact tarball: acquire → verify → static/dynamic inspect → policy
  → complete ALLOW Verified Set / Manifest
  → quarantine → staging digest recheck
  → script-free offline local npm ci
  → staging → target digest recheck / atomic publish
```

- `--target`은 필수이며, 호출 시 존재하지 않는 absolute directory여야 한다. Heliopause는 canonicalized non-symlink parent 아래 target sibling temporary directory를 만들고, 성공한 완성 directory만 atomic rename 한다.
- 기존 project/CWD, global install, workspaces, `package.json`/lockfile 병합, `npmrc`, registry override, credential, proxy, arbitrary npm option pass-through는 M4 범위 밖이다.
- M4 automatic Promotion은 Linux amd64 glibc 환경만 지원한다. macOS/unsupported Linux에서는 required dynamic inspection이 incomplete이므로 `ALLOW`와 Promotion이 되지 않는다.
- target parent 외 경로, 사용자 환경 변수·npm 설정, Host credential, Evidence Store 및 observer socket은 resolver/Promotion process에 전달하지 않는다.

이 제한은 기존 project를 `npm ci`가 삭제·변경할 수 있는 위험과 user/global npm configuration이 dependency graph를 바꾸는 위험을 제거한다.

## 2. Dependency resolution boundary

M4는 package manager semantics를 임의로 재구현하지 않는다. exact graph는 locked Node/npm resolver boundary가 생성한 `package-lock.json` v3에서만 도출한다.

```text
exact primary reference
  → per-operation empty resolver project
  → locked npm --package-lock-only --ignore-scripts
  → bounded package-lock v3
  → canonical lock parser
  → exact dependency candidates
```

- resolver는 M3 dynamic Sandbox와 별개의 disposable trusted infrastructure helper다. untrusted package code/lifecycle script를 실행하지 않으며, public `registry.npmjs.org` allowlist 외 egress, Host bind mount, credential, user/global `.npmrc`를 허용하지 않는다.
- exact Node image는 M3의 digest-pinned Node 22.23.1 image를 사용하고, 그 upstream bundled npm 10.9.8을 M4 runtime lock으로 고정한다. Node 22 image에 npm 11.15.0을 별도 설치하면 runtime identity와 resolver dependency boundary가 바뀌므로 사용하지 않는다. helper startup의 npm version mismatch는 fail-closed 한다.
- resolver command는 `--package-lock-only`, `--ignore-scripts`, `--no-audit`, `--no-fund`, empty HOME/cache와 서로 다른 빈 user/global npm config 경로 및 fixed public registry를 사용한다. package metadata resolution 이외 install·tarball execution·lifecycle 실행은 하지 않는다.
- canonical parser는 lockfile v3, root dependency와 package entry mapping, exact registry `resolved` URL, SHA-512 SRI, package name/version을 요구한다. git, URL/file/link/workspace/alias, bundled dependency, missing/duplicate/non-registry integrity, unsupported optional/platform branch 또는 unsupported lock semantics는 incomplete이며 Promotion하지 않는다.
- resolver output은 bounded and sanitized data만 Application으로 넘긴다. raw lockfile, registry response, URL query, Host path, npm stderr는 result/Evidence summary에 직접 노출하지 않는다.
- HAA owned network-policy helper가 확정·검증한 npm registry endpoint 집합만 egress를 허용한다. Host trusted DNS preflight의 IPv4 결과는 firewall allow rule과 resolver container의 explicit `--add-host registry.npmjs.org:<ip>`에 같은 집합으로 고정한다. 따라서 resolver가 Docker embedded DNS, Host proxy 또는 이름 기반 firewall rule에 의존하지 않으며, DNS egress를 추가로 열지 않는다. 이는 앞 문단의 단일 hostname 표현을 대체한다.
- network-policy helper는 resolver 전용 Docker network와 default-deny firewall policy를 create → apply → verify → cleanup한다. firewall backend, policy rule, endpoint resolution, cleanup 어느 단계라도 실패/불확실하면 runtime은 graph를 반환하지 않는다. unrestricted Docker bridge fallback 및 proxy environment-only enforcement는 금지한다.
- Linux firewall rule 적용에는 CAP_NET_ADMIN 또는 동등한 명시적 elevation이 필요하다. HAA는 권한을 우회하거나 암묵적으로 승격하지 않으며, 이 권한이 없으면 resolver는 fail-closed 한다. Linux CI는 controlled root integration으로 이 경로를 검증한다.

`npm ci`는 lockfile과 manifest가 불일치하면 실패하고 manifest/lockfile을 수정하지 않는 frozen project install이며, M4는 이 성질을 Promotion에 사용한다. [npm ci 공식 문서](https://docs.npmjs.com/cli/v11/commands/npm-ci/)와 [package-lock v3 문서](https://docs.npmjs.com/cli/v11/configuring-npm/package-lock-json/)가 이 선택의 upstream 근거다.

## 3. Complete Verified Set과 inspection

Resolver가 제시한 primary 및 모든 supported dependency는 각각 M2/M3의 exact resolve, controlled intake, declared-integrity verification, static inspection, dynamic inspection, Evidence와 Policy 경로를 거친다.

- 한 entry라도 operational failure, required check incomplete, `MANUAL_REVIEW` 또는 `BLOCK`이면 set-level `ALLOW`를 만들지 않고 Staging/Promotion을 호출하지 않는다.
- 새 dependency가 resolver lock 또는 Promotion에서 나타나면 기존 set을 확장하지 않는다. 해당 operation은 stopped/incomplete로 끝나며 새 graph resolution과 inspection으로 돌아간다.
- M4 automatic Promotion은 lifecycle script 또는 implicit `node-gyp` install action이 있는 entry를 `M4_HOST_LIFECYCLE_UNSUPPORTED` `MANUAL_REVIEW`로 처리한다. 따라서 Host Promotion의 `npm ci`는 `--ignore-scripts`를 강제한다.
- 이는 lifecycle을 검사 환경에서 관찰한다는 M3 contract는 유지하면서, 관찰 결과만으로 Host에서 임의 install script를 실행하는 권한을 부여하지 않기 위한 MVP 제한이다.

## 4. Verified Manifest와 SBOM

모든 entry가 current set-level `ALLOW`를 만족하면 deterministic `helox.verified-manifest/v1` JSON을 생성한다. Manifest는 canonical UTF-8 JSON, lexicographic entry order, bounded field 길이와 controller-computed SHA-256을 사용한다.

```text
manifest
├─ manifest id / SHA-256
├─ operation id / target context snapshot
├─ resolver runtime identity / lockfile digest
├─ primary entry
└─ sorted entries
   ├─ role / npm name / exact version
   ├─ source / observed SHA-256 / declared SHA-512 SRI
   ├─ inspection run / evidence references / policy basis
   └─ lock package path and dependency edges
```

- Manifest는 package-lock이나 SBOM을 대체하지 않는다. package-lock은 npm installation graph input이고 Manifest는 Heliopause approval·digest binding record다.
- M4는 same set에서 CycloneDX 1.7 JSON SBOM을 생성한다. SBOM component와 dependency edge는 lockfile에서, package identity/digest and Heliopause evidence link는 Manifest에서만 보강한다. CycloneDX 1.7은 현재 공식 stable specification이다. [CycloneDX specification overview](https://cyclonedx.org/specification/overview/)
- Manifest/SBOM 생성 또는 validation failure는 Promotion 이전 operation failure다.

M4-004에서 Manifest ID는 `manifest_id`를 제외한 Manifest object를 JSON number precision 손실 없이 key-sorted compact JSON으로 직렬화한 bytes의 SHA-256으로 고정한다. 완성 Manifest는 이 ID를 `manifest_id`로 포함하고 CycloneDX metadata의 `heliopause:manifest-id`도 같은 값을 가진다. SBOM은 deterministic output을 위해 optional timestamp·serial number를 생성하지 않고 `pre-build` lifecycle을 명시한다.

## 5. Verified Staging

Staging root는 configured user cache 아래 `staging/<manifest-id>/`이며 Evidence/Intake root와 별도의 trusted root다.

```text
staging/<manifest-id>/
├─ manifest.json
├─ sbom.cdx.json
└─ artifacts/<observed-sha256>.tgz
```

- directory는 `0700`, files are written exclusive `0600`, `fsync`ed and atomically renamed; completed artifacts/records become controller-owned read-only. Canonical containment and no-symlink checks apply to every path.
- 완성 파일은 temporary tree에서 exclusive `0600`으로 쓴 뒤 `fsync`하고 `0400`으로 봉인한다. artifact directory와 Manifest directory는 `0700`을 유지하며, 봉인·directory `fsync`·no-replace atomic rename·root `fsync` 순서가 모두 성공해야 opaque `StagedSet` handle을 반환한다. Linux amd64는 `renameat2(RENAME_NOREPLACE)`로 final Manifest directory 경합도 덮어쓰지 않는다.
- Quarantine → Staging에서 Manifest entry의 intake bytes SHA-256을 재계산하여 일치할 때만 copy한다. copy/permission/fsync/rename/hash failure는 no-stage/no-promotion failure다.
- intake handle의 Run ID가 inspection Run ID와 일치해야 하며, source size·regular-file·no-symlink·observed SHA-256를 copy 중 다시 검증한다. 동일 digest가 여러 entry에 등장해도 모든 intake source를 독립적으로 재계산하고 staged bytes만 digest-addressed 파일 하나로 저장한다.
- Staging은 storage only다. Artifact execution, npm/cache command, lifecycle script, network, credential, mutable Manifest update를 수행하지 않는다.
- staging retention/garbage collection은 M4에서 자동화하지 않는다. failed temporary staging is removed best-effort and deletion failure is recorded; completed staging is retained for audit/retry until a later retention decision.

## 6. Offline Promotion

Promotion은 Staging artifact를 temporary target-local artifact directory에 exact byte copy하고 each SHA-256을 Manifest와 다시 비교한 후, generated root `package.json` + file-source lockfile v3로 locked npm을 실행한다.

```text
staged exact tarballs
  → target sibling temporary directory
  → generated file-source package-lock v3
  → locked npm ci --offline --ignore-scripts --no-audit --no-fund
  → target directory atomic rename
```

- Promotion runtime has no network, no user/global npm config, empty HOME/cache, `--offline`, `--ignore-scripts`, `--no-audit`, `--no-fund`, `--bin-links=false`, and only the generated local project/artifact tree as writable input. It receives no resolver egress capability.
- npm cache is not a Staging representation: upstream describes it as opaque and not a persistent/reliable package data store. M4 uses explicit Manifest-bound local tarballs instead. [npm cache documentation](https://docs.npmjs.com/cli/cache/)
- the generated lock is validated before execution and after output preparation; every `resolved` source must name one local tarball whose SHA-256 is in the Manifest. Any attempted network request, missing local source, lock mutation, source escape or unlisted Artifact requirement fails closed.
- target must not exist before Promotion. Promotion never overwrites/merges an existing directory. Before publish, trusted controller verifies generated target and parent containment, `fsync`s required files/directories, then performs one rename; failure before rename removes the temporary target best-effort.
- a failure after Policy `ALLOW` is `Operation FAILED` with a Promotion limitation. It never rewrites the completed inspection Policy Decision to `BLOCK`.

M4-005의 Promotion runtime은 digest-pinned `node:22.23.1-slim` image와 bundled npm `10.9.8`을 사용한다. Docker는 `--pull never`, `--network none`, read-only root, all-capability drop, no-new-privileges, bounded PID/memory/CPU, empty temporary HOME/cache/config로 실행하고 workspace 외 Host path를 mount하지 않는다. Docker client 자체도 빈 temporary `HOME`/`DOCKER_CONFIG`를 사용하여 Host registry credential을 읽지 않는다.

generated lock의 모든 `resolved`는 target-local `.heliopause/artifacts/<sha256>.tgz` file source이며 registry URL을 포함하지 않는다. runtime 종료 후 controller는 package/lock/Manifest/SBOM 불변, target-local tarball SHA-256, installed package path·name·version, exact package set, no-symlink/no-special-file를 다시 검증한다. Linux amd64 publish는 `renameat2(RENAME_NOREPLACE)`를 사용하여 validation과 rename 사이에 target이 생기는 경합도 덮어쓰지 않고 실패한다.

Linux amd64가 아닌 Host의 automatic install은 resolver DNS, Docker 또는 firewall 명령에 진입하지 않는 unsupported adapter에서 즉시 operational failure가 된다. 이 제한은 `npm inspect`의 platform별 dynamic capability 처리와 분리하며 unrestricted resolver/Promotion fallback을 만들지 않는다.

## 7. M4 policy and result contract

M4 Policy identity is `m4-npm-install-promotion`, version `1`.

| Condition | Decision / result |
| --- | --- |
| any exact graph entry has blocking verification/inspection result | `BLOCK`; no staging/promotion |
| required check/runtime/resolver/lock semantics unavailable or incomplete | `MANUAL_REVIEW`; no staging/promotion |
| any lifecycle/native install action needed for Host install | `MANUAL_REVIEW / M4_HOST_LIFECYCLE_UNSUPPORTED`; no staging/promotion |
| all entries verified/inspected and set manifest valid | `ALLOW`; stage then promote |
| staging/promotion operational failure after `ALLOW` | Policy remains `ALLOW`; Operation is `FAILED`; target is absent or previous untouched |

Human and machine result retain existing operation-result schema while adding bounded Manifest/SBOM references, exact set count and Promotion status. They never contain tarball paths, lockfile text, npm stderr, environment or credentials.

- `ALLOW` 후 Staging/Promotion operational failure는 `operation_status=FAILED`, sanitized failure code, original set-level `ALLOW`와 Manifest reference를 같이 보존한다.
- set-level `MANUAL_REVIEW`/`BLOCK`은 error가 아닌 `operation_status=NOT_PERFORMED`, `promotion_status=NOT_PERFORMED`로 표현하며 Manifest/Staging/Promotion을 호출하지 않는다.

## 8. Implementation queue

| Order | ID | Scope |
| --- | --- | --- |
| 1 | M4-002 | Install Context Domain/CLI boundary and locked npm resolver contract |
| 2 | M4-003 | lockfile v3 parser, recursive exact dependency intake/inspection and set-level Policy |
| 3 | M4-004 | Verified Set/Manifest/SBOM Domain and trusted Staging adapter |
| 4 | M4-005 | offline npm Promotion adapter, atomic target handling and failure contract |
| 5 | M4-006 | controlled full E2E qualification and M5/M6 handoff |

No M4 source, Docker resolver, Staging adapter, npm CLI command or CI job is created by this entry decision.
