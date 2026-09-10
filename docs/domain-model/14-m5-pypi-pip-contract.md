# M5 Entry Decision — PyPI/pip Expansion Contract

M5는 npm의 공통 Artifact/Verified Set/Promotion 계약을 PyPI에 연결한다. PyPI의 distribution, resolver, Python build backend 또는 pip의 raw payload가 Core, Application, Policy에 유입되지 않는다.

## 1. M5 MVP 범위

```text
helox pypi inspect <project>[@<version>]
helox pypi install <project>[@<version>] --target <absolute-directory>
  → public PyPI Simple API exact candidate graph
  → every selected wheel/sdist: controlled intake → verify → static/dynamic inspect
  → sdist: verified build inputs + isolated build → derived wheel re-inspection
  → complete ALLOW Verified Set / Manifest / SBOM
  → trusted staging digest recheck
  → no-network, local-wheel pip Promotion → atomic new target
```

- source는 public PyPI `https://pypi.org/simple/` 하나다. private index/mirror, `--extra-index-url`, VCS/direct URL/local path, editable install, credentials, proxy, pip configuration/환경변수와 arbitrary pip option pass-through는 M5 자동 경로 밖이다.
- 입력은 PyPI project name과 선택적인 **하나의 exact PEP 440 version**이다. extras, requirement range, marker, URL과 local version은 입력으로 허용하지 않는다. project name은 PyPA name normalization으로 canonicalize한다.
- version이 생략되면 M5-pinned target runtime과 호환되는 가장 높은 non-yanked final release를 resolver가 exact version으로 확정한다. prerelease/dev release는 명시한 exact version일 때만 후보가 될 수 있다. ambiguous/invalid/non-canonical version, yanked distribution, unsupported marker 또는 하나로 정해지지 않는 candidate는 fail-closed다.
- automatic inspect/Promotion은 M5-002에서 exact identity를 고정할 Linux amd64 CPython/pip runtime만 지원한다. macOS, Windows 및 다른 Python/ABI/platform target은 explicit unsupported/incomplete이며 `ALLOW`하지 않는다.
- target은 존재하지 않는 absolute directory다. 기존 Python environment, global/user site, existing venv, project CWD, `requirements.txt` 병합은 지원하지 않는다.

M5-001은 contract/queue만 만든다. Go package, Python/pip image/runtime lock, resolver, Sandbox runner, CLI, CI job 또는 third-party Go dependency를 추가하지 않는다.

### M5-002 runtime identity와 target basis

M5 automatic path의 유일한 runtime은 Linux amd64 Docker Official Image `python:3.14.7-slim-bookworm@sha256:23c59390fc717bf09f9336908199a0ae75d9c4264bf296123f94ad772fea3b52`이다. 이는 mutable tag가 아닌 OCI index digest다. Python identity는 `3.14.7`, bundled pip identity는 `26.2.1`로 lock한다. CPython 3.14.7은 2026-08-05의 current stable maintenance release이고, pip 26.2.1은 2026-08-04의 latest stable release다. CPython 3.14 maintenance line에는 earlier pip archive-handling security fix를 포함한 pip 26.1 update가 backport되었으므로, M5는 그보다 최신인 26.2.1을 사용한다.

- fixed target tag **basis**는 `cp314` interpreter, `cp314` ABI, `manylinux_2_36_x86_64` platform이다. 이 값은 Debian bookworm glibc target을 나타내며 arbitrary Host tag·architecture로 fallback하지 않는다. M5-003 resolver는 locked container 안에서 reported Python/pip version 및 `pip debug --verbose` compatible-tag closure를 다시 확인해 candidate selection input을 만들며, mismatch·unknown tag·unavailable command는 incomplete다.
- `scripts/runtimes.lock.json`이 image reference, Python/pip version과 target basis의 single runnable lock input이다. `internal/sandbox.PinnedPythonRuntime`과 lock의 mismatch는 test failure다.
- `ProbePython`은 image를 pull하거나 pip/project를 실행하지 않는다. macOS/Windows/non-amd64는 Docker command 전 `M5_PYPI_LINUX_AMD64_ONLY`, gVisor/Docker prerequisite failure는 `M5_PYPI_RUNTIME_*`, exact image absent는 `M5_PYPI_IMAGE_UNAVAILABLE`로 explicit non-ALLOW capability를 반환한다. Node image presence는 PyPI capability의 prerequisite가 아니다.
- public input은 `project[@exact-version]`뿐이다. parser는 PyPA name normalization과 public PEP 440 normalization을 적용하고, extras/specifier range/marker/direct URL/local version/whitespace와 ambiguous locator를 resolver 전에 거부한다. normalized project/version만 later adapter input이 되며 raw user input은 result/Evidence에 쓰지 않는다.

공식 근거: [Python 3.14.7 release](https://www.python.org/downloads/release/python-3147/), [Docker Official Images Python manifest](https://github.com/docker-library/official-images/blob/master/library/python), [Python 3.14 slim-bookworm Dockerfile](https://github.com/docker-library/python/blob/master/3.14/slim-bookworm/Dockerfile), [pip 26.2.1 release](https://pypi.org/project/pip/), [pip changelog](https://pip.pypa.io/en/stable/news/), [CPython pip security backport](https://github.com/python/cpython/issues/149148), [PEP 440](https://peps.python.org/pep-0440/), [PyPA name normalization](https://packaging.python.org/en/latest/specifications/name-normalization/)와 [platform compatibility tags](https://packaging.python.org/en/latest/specifications/platform-compatibility-tags/).

## 2. Source, identity and selected distribution

PyPI Simple API의 JSON response를 primary index metadata로 사용한다. resolver는 response API version을 확인하고 지원하지 않는 future major version은 실패한다. HTML fallback, PyPI JSON API, project metadata sidecar는 M5에서 secondary corroborating Evidence일 수 있으나 selected distribution의 canonical resolver input을 대체하지 않는다.

```text
normalized project + optional exact version
  → pinned resolver runtime
  → pip dry-run/report candidate graph
  → Simple API JSON cross-check
  → exact distribution file per graph node
```

- pinned resolver pip은 disposable Linux gVisor Sandbox에서만 실행한다. `pip lock`과 `pylock.toml`은 current pip에서 experimental이므로 M5 lock input으로 사용하지 않는다. `pip --report` v1은 candidate discovery에만 쓰며 product lock/Manifest를 대체하지 않는다.
- report의 pip version, target environment, selected name/version, URL과 archive hash는 bounded parser로 읽고, Simple API JSON의 same file name/URL/SHA-256/yanked/Requires-Python metadata와 일치해야 한다. raw report, index response, URL query, pip output과 Host path는 Core/Policy/result에 전달하지 않는다.
- selected file URL은 HTTPS와 M5가 검증한 public PyPI distribution endpoint만 허용한다. redirect, endpoint mismatch, missing SHA-256, unsupported digest algorithm, missing size 또는 metadata disagreement은 incomplete다.
- node마다 exact filename, normalized project, PEP 440 canonical version, artifact type (`wheel`, `sdist`, `derived-wheel`), public source, declared SHA-256와 observed SHA-256을 분리해 기록한다. PyPI가 주장한 hash는 declared integrity이며 Controlled Intake의 observed digest를 대체하지 않는다.
- resolver가 다운로드하거나 build metadata 실행 중 새 distribution/build requirement를 요구하면 이를 기존 graph 또는 Verified Set에 추가하지 않는다. candidate graph를 새로 resolve해 모든 entry가 같은 pipeline을 거쳐야 하며, M5가 지원하지 않는 dynamic requirement는 `MANUAL_REVIEW`다.

M5 resolver egress는 HAA가 소유하는 default-deny network-policy helper를 일반화해 public PyPI index/distribution endpoints만 allow한다. policy create/apply/verify/cleanup, DNS result, redirect/endpoint validation 또는 Sandbox/observer 상태가 불완전하면 graph를 반환하지 않는다. unrestricted Docker bridge, Host firewall/proxy 가정과 proxy environment-only enforcement는 금지한다.

### M5-003 resolver profile

- resolver는 M5-002 immutable Python image의 `runsc-trace` gVisor container에서만 `python -I -m pip install --dry-run --report` v1을 실행한다. container runtime Python/pip identity와 `pip debug --verbose`의 `cp314-cp314-manylinux_2_36_x86_64` compatible-tag closure가 exact lock과 다르면 fail-closed다.
- M5-003은 build backend 실행을 막기 위해 `--only-binary=:all:`을 사용한다. wheel이 없는 graph, sdist, dependency extras, direct requirement 또는 marker는 M5-005 PEP 517 경계 전에는 incomplete/`MANUAL_REVIEW`이며 wheel-only resolver result로 위장하지 않는다.
- Host trusted DNS preflight는 `pypi.org`와 `files.pythonhosted.org`의 public IPv4 address만 얻는다. 그 exact address set은 HAA firewall allow rule과 resolver `--add-host`에 동시에 적용한다. Sandbox DNS, proxy 및 다른 endpoint는 allow하지 않는다.
- Python standard-library HTTPS fetcher는 Simple JSON v1 Accept/content type, no redirect, status/length/4 MiB body bound를 검증한다. selected report file은 `files.pythonhosted.org` HTTPS no-query/no-redirect URL과 Simple metadata의 same filename, SHA-256, yanked=false, Requires-Python, size를 모두 만족해야 한다.
- resolver는 `container_id` attribution이 confirmed된 trusted gVisor observer stream을 Sandbox start 전 등록하고, container disposal 뒤 stream end를 수집한다. observer connect/protocol/drop/mapping/stream failure는 parsed graph를 폐기한다.

## 3. Wheel 선택과 정적 계약

각 runtime dependency node는 M5-pinned target tag와 호환되는 distribution **하나**만 선택한다. 호환 wheel이 있으면 그것을 사용한다. wheel tag와 embedded `WHEEL` metadata가 target interpreter/ABI/platform과 모두 일치해야 하며, `py3-none-any`도 명시적으로 호환 판정을 거친다. 하나 이상의 동등 candidate, unsupported native tag 또는 wheel filename/embedded metadata disagreement은 자동 선택하지 않는다.

wheel 정적 inspection은 trusted controller에서 archive를 실행하지 않고 다음을 검증한다.

- zip containment, bounded compressed/uncompressed size·file count, regular file만 허용, symlink/special file/path escape와 archive bomb 거부
- exactly matching `.dist-info/METADATA`, `.dist-info/WHEEL`, `.dist-info/RECORD`; normalized name/version, supported Wheel-Version과 selected tag 일치
- `RECORD`의 permitted empty self-entry 외 모든 recorded file의 digest/size, no duplicate path와 no unrecorded installable content 검증
- `.data/scripts`는 regular file만 허용하고 console/gui entry point 및 native extension은 executable surface Evidence로 정규화
- Core Metadata의 `Requires-Dist`, `Requires-Python`, `Import-Name`, entry point와 license/provenance metadata는 bounded Evidence/next resolver input이며 safety decision 자체가 아니다.

wheel의 dynamic inspection은 M3 trusted observer/gVisor session에서 target-local private directory에 verified wheel만 `pip --no-index --no-deps`로 설치한 뒤 bounded declared import surface를 실행한다. Artifact가 제공한 script/entry point, `setup.py`, arbitrary module name 또는 Host Python을 실행하지 않는다. import surface를 안전하게 확정할 수 없거나 session/observation이 incomplete면 `MANUAL_REVIEW`다.

## 4. sdist와 derived wheel

sdist는 build를 필요로 할 수 있는 executable Artifact다. source archive 또는 PEP 517 backend는 Host에서 실행하지 않는다.

- sdist는 `.tar.gz`, single top-level directory, matching normalized filename/name/version, `PKG-INFO`, `pyproject.toml` 및 bounded regular-file containment를 요구한다. legacy `setup.py` fallback, missing `build-backend`, malformed/in-tree `backend-path`, archive path escape/symlink/special file은 M5 automatic path에서 unsupported다.
- static inspection은 `PKG-INFO`, `pyproject.toml`의 `[build-system]`, declared build requirements와 runtime requirements를 extraction 없이 제한된 archive reader로 확인한다. build requirements는 runtime dependency보다 낮은 신뢰 등급이 아니며 각각 exact resolution → intake → verification → static/dynamic inspection을 거친다.
- 모든 declared build requirement가 current Verified candidate graph에 포함된 뒤에만 M3 gVisor Sandbox에서 `pip wheel --no-index --no-deps --no-build-isolation`을 실행한다. Sandbox는 verified source/build wheels 외 mount, credential, network와 Host Python을 받지 않는다.
- PEP 517 hook이 graph 밖 requirement를 요청하거나 output wheel을 하나로 결정할 수 없으면 build를 중단하고 `MANUAL_REVIEW`한다. build runtime, source digest, build-requirement identities/digests, build-system config digest, command identity와 normalized observation은 Evidence로 연결한다.
- output wheel은 source sdist와 동일시하지 않는 `derived-wheel` Artifact다. controller가 observed SHA-256을 계산하고 source/build recipe binding을 확인한 뒤 wheel static·dynamic inspection을 다시 수행한다. PEP 517 build는 source와 모든 build wheel의 required check가 complete `ALLOW`인 경우에만 시작한다. completed build의 trusted gVisor observation collection은 derived wheel의 required check/Evidence로 기록하고, source digest·sorted build-input digest·bounded executor identity·build-system config digest는 generic source-to-derived graph binding으로 Manifest에 직렬화한다. source sdist, verified build inputs와 derived wheel 모두 `ALLOW`여야 Promotion 후보가 된다.

## 5. Verified Set, staging and offline pip Promotion

M4의 generic `Verified Set`, deterministic Manifest, CycloneDX 1.7 SBOM, Intake→Staging→Promotion rehash, exclusive/read-only storage와 no-replace atomic publish invariant를 그대로 사용한다. implementation은 npm tarball 전용 storage/promotion detail을 generic artifact representation으로 좁게 확장할 수 있지만 Core/Application/Policy에 PyPI branch를 추가하지 않는다.

sdist가 선택되면 source distribution과 build input은 Promotion wheel이 아니며, build가 만든 `derived-wheel`은 source와 동일한 Artifact가 아니다. 따라서 generic dependency graph/Verified Set에는 source node와 별도의 derived Artifact node 및 source digest·build-input digest·build backend/config·Sandbox/Evidence binding을 표현할 수 있어야 한다. Application은 이 generic binding만 다루고 PyPI archive/backend type은 adapter에 남긴다. derived-wheel static/dynamic reinspection과 entry Policy가 모두 `ALLOW`인 경우에만 source/build node와 함께 Manifest/SBOM에 포함되어 Promotion wheel subset이 된다. 이 node/binding을 만들 수 없거나 하나라도 불완전하면 set은 `MANUAL_REVIEW`이며 Promotion하지 않는다.

Manifest에는 resolver runtime identity/report digest, target tag set, selected distribution filename/type, declared/observed SHA-256, dependency edges, source-to-derived-wheel recipe/evidence binding과 actual Promotion wheel subset을 기록한다. SBOM에는 runtime distribution과 derived wheel component/edges를 모두 기록하되 Manifest를 대체하지 않는다.

```text
staged exact wheels only
  → target sibling temporary directory
  → generated fully pinned hash requirements
  → pip install --no-index --find-links --require-hashes --only-binary :all: --no-deps
  → controller RECORD/metadata/tree revalidation
  → atomic no-replace target publish
```

- Promotion Python/pip runtime은 M5-002에서 exact image/version/digest를 lock한다. runtime에는 `--network none`, read-only root, non-root, capability drop, no-new-privileges, bounded resource, empty HOME/cache/config과 generated local wheel directory만 제공한다.
- generated requirements는 every promoted distribution의 exact normalized name/version/SHA-256을 포함한다. `--no-deps`는 pip가 Manifest 밖 dependency를 다시 resolve하지 못하게 하며 network, index, cache, VCS, local project와 arbitrary build를 허용하지 않는다.
- promotion 후 controller는 frozen requirements/Manifest/SBOM/local wheel digest, expected `.dist-info/METADATA` name/version, `RECORD` containment/hash, exact installed distribution set와 no-symlink/no-special-file를 확인한다. target pre-existence, parent identity change, new distribution requirement, output mismatch와 cleanup uncertainty는 fail-closed다.

## 6. Policy and result semantics

M5 Policy identity is `m5-pypi-pip`, version `1`.

| Condition | Decision / result |
| --- | --- |
| selected wheel/sdist/build/derived-wheel entry has a blocking verification or inspection result | `BLOCK`; no staging/promotion |
| resolver/index/hash/tag/marker/runtime/Sandbox/observer/build semantics unavailable or incomplete | `MANUAL_REVIEW`; no staging/promotion |
| legacy sdist, dynamic graph expansion, unsupported platform tag or Host execution requirement | `MANUAL_REVIEW`; no staging/promotion |
| all runtime/build/derived entries are complete and exact set/Manifest is valid | `ALLOW`; stage then promote |
| staging/promotion operational failure after `ALLOW` | Policy remains `ALLOW`; operation is `FAILED`; target is absent or previous target untouched |

M4 `operation-result/v1` and the human result remain the public result boundary. They keep the existing sanitized operation/policy/Manifest/SBOM/Evidence/Promotion fields; raw PyPI/pip output, endpoint URL details, filesystem paths, environment and credentials never appear. No new public result schema is created in M5 unless a later implementation proves the generic contract insufficient.

## 7. M5 implementation queue

| Order | ID | Scope | Recommended model |
| --- | --- | --- |
| 1 | M5-002 | Python/pip runtime identity lock, public PyPI source/reference parser and capability probe | Terra Medium |
| 2 | M5-003 | Simple API/report cross-checked isolated resolver, target tag selection and PyPI egress policy | Terra High |
| 3 | M5-004 | wheel intake/integrity/static inspection and deterministic adapter contract tests | Luna High |
| 4 | M5-005 | Python gVisor dynamic inspection plus PEP 517 sdist build/derived-wheel boundary | Terra High; Sol Low only for persistent upstream/runtime incompatibility |
| 5 | M5-006 | generic staging adaptation, offline pip Promotion, CLI/bootstrap/result wiring and Linux integration | Terra High; Luna High for bounded test/fixture work |
| 6 | M5-007 | M5 qualification, npm regression, documentation and M6 handoff | Luna High for evidence/docs; Terra Medium for security audit |

## 8. Official sources and decision basis

- [PyPA Simple Repository API](https://packaging.python.org/en/latest/specifications/simple-repository-api/) — normalized project endpoints, JSON representation, file hashes, yanked/metadata/versioning behavior
- [PyPI JSON API](https://docs.pypi.org/api/json/) — PyPI-specific release/file metadata corroboration only
- [Name normalization](https://packaging.python.org/en/latest/specifications/name-normalization/) and [PEP 440](https://peps.python.org/pep-0440/) — project/version normalization and selection
- [Wheel binary distribution specification](https://packaging.python.org/en/latest/specifications/binary-distribution-format/), [platform tags](https://packaging.python.org/en/latest/specifications/platform-compatibility-tags/) and [installed project records](https://packaging.python.org/en/latest/specifications/recording-installed-packages/) — wheel/RECORD/tag validation
- [Source distribution format](https://packaging.python.org/en/latest/specifications/source-distribution-format/) and [PEP 517](https://peps.python.org/pep-0517/) — sdist/build backend and isolated build requirements
- [pip installation report](https://pip.pypa.io/en/latest/reference/installation-report/) — stable report v1 is not a lock input
- [pip lock](https://pip.pypa.io/en/latest/cli/pip_lock/) — experimental, excluded from M5 canonical lock path
- [pip secure installs](https://pip.pypa.io/en/stable/topics/secure-installs/) and [pip install](https://pip.pypa.io/en/stable/cli/pip_install/) — hash checking, `--no-index`, `--find-links`, binary-only and local offline install semantics
