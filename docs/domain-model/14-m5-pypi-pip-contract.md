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
- 기존 resolver/dynamic factory가 configured executor를 전달하는 `probePython`은 image를 pull하거나 pip/project를 실행하지 않는다. macOS/Windows/non-amd64는 Docker command 전 `M5_PYPI_LINUX_AMD64_ONLY`, gVisor/Docker prerequisite failure는 `M5_PYPI_RUNTIME_*`, exact image absent는 `M5_PYPI_IMAGE_UNAVAILABLE`로 explicit non-ALLOW capability를 반환한다. Node image presence는 PyPI capability의 prerequisite가 아니다. 사용되지 않던 executor 없는 public wrapper는 M12-005에서 제거했으며 실제 capability probe와 조건은 유지한다.
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

Python dynamic inspection은 exact authenticated installed closure에 대한 controller-owned bounded observation experiments로 구성한다. 각 required unit의 identity, launch, externally observed process outcome, observer attribution, cumulative resource accounting, runtime destruction과 cleanup을 controller가 독립적으로 reconciliation한다. `COMPLETED`는 이 실험들의 실행·관찰·회계·정리를 뜻하며 Python import의 정상 반환, 모든 module body의 실행 또는 package safety를 증명하지 않는다. Artifact가 출력하거나 같은 interpreter에서 계산한 completion token은 coverage authority가 아니다. `sys.exit(0)`은 그 unit에서 외부 관찰된 zero-exit outcome일 뿐 다른 unit을 완료하거나 생략할 수 없다. 누락된 unit, unsupported required surface, required unit의 nonzero 또는 timeout/resource termination, uncertain attribution, 불완전한 observer evidence 또는 cleanup은 `MANUAL_REVIEW`로 남는다.

### Post-install command target의 bounded 관찰 (2026-10-03 정책 정교화)

인증된 `console_scripts`/`gui_scripts` metadata만으로 추가된 exact target module은
`POST_INSTALL_COMMAND` 관찰 단위다. 이는 후속 사용자 명령이며 base 설치의
필수 successful-import 조건은 아니다. 실행 전에 artifact plan owner가 coverage를
고정한다. 정상 import/native/plugin/startup 등 다른 required 역할과 겹치면 required가
우선한다. 다른 entry-point group은 기존 required 규칙을 유지한다. 동적 startup
statement가 있는 wheel은 reachability를 분리 입증하지 않으므로 command target도
required로 유지한다. Declarative path만으로 이 결정을 바꾸지 않는다.

모든 계획 단위는 기존 controller launch/관찰/회계/종료/정리를 거친다. 정확한
launch consumption과 외부 terminal zero, 완전한 trusted evidence, actionable Finding
없음이면 `ATTESTED`, 정상 nonzero이면 `NOT_ATTESTED`로 기록한다. 후자는 base
required check 실패가 아니다. 실행되지 않은 명령 실패, signal/timeout, missing unit,
불완전한 observer/회계/cleanup은 계속 fail-closed다. stderr·누락 dependency·출력은
coverage나 보조 dependency를 결정하지 않는다. 관찰된 network/exec/filesystem 등
Finding은 terminal 상태와 무관하게 기존 Inspector→Policy 판정을 유지한다.

Python runner의 typed outcome을 Inspector가 기존 required transaction Evidence에
추가한다. Evidence에는 exact module/unit/owner digest, disposition,
`functionality_attested=false`, `later_execution_enforced=false`가 포함된다. CLI 결과와
promotion manifest의 Evidence reference에도 `pypi-command-not-attested-*` 구분을
유지한다. `ATTESTED`조차 callable 실행·정상 import return·기능·package 안전성을
증명하지 않는다. HAA는 승격된 `NOT_ATTESTED` command의 나중 실행을 차단하지
않는다. 원래 요청 graph/승격 집합은 그대로이며 명령을 동작시키기 위한 recursive
inspection prerequisite 확장은 없다. 이 절은 이전 모든 exact entry-point target의
required-success 규칙을 이 범위에서 명시적으로 대체한다.

### 명시적인 inspection-only prerequisite (2026-10-02 정책 정교화)

기본 입력은 비어 있다. `pip install --inspection-prerequisites '<JSON>'`은 trusted caller가 선택한 **비-root** artifact의 broader module probes에만 한 개의 leaf wheel을 제공한다. JSON은 `target_sha256`, `source`, `project`, `version`, `filename`, `sha256`, `reason`을 포함하며 reason은 `BROADER_MODULE_PROBES`다. Artifact의 오류·stdout·누락 import는 이 입력을 선택할 권한이 없다. 원래 요청 root는 항상 원래 전체 plan과 원래 dependency closure로 검사하며, 그 required 실패는 보조 환경 성공으로 상쇄할 수 없다. root/직접 요청의 augmentation은 지원하지 않는다.

- Bootstrap은 기존 격리 PyPI resolver와 intake/integrity verification으로 official `pypi` source의 caller pin을 교차 확인한다. CompositeInspector는 모든 원래 wheel과 보조 wheel의 구조·RECORD·CP314 ABI·설치 역할을 검사하고 동일 project 교체, import shadowing, 최종 파일/디렉터리 collision을 거부한다. 현재 leaf 계약은 `Requires-Dist` 또는 startup hook이 있는 보조 입력과 recursive/extra expansion을 지원하지 않는다.
- 보조 입력도 ARTIFACT이며 전체 own plan의 모든 unit을 target unit 앞에서 실행한다. 같은 frozen RO closure, preparation/anchor, CPU/memory/tmpfs/wall/observer authorization을 공유하고 비용을 원래 graph의 storage 한도에도 합산한다. 실패는 sticky이며 별도 budget reset·권한·네트워크를 받지 않는다.
- `pypi-dynamic-import-supplemented`는 별도의 required check다. generic Evidence summary에 input source/project/version/digest/reason, target/base/input/runtime/root-policy/plan binding과 정규화 관찰을 남기며 `original_environment=NOT_ATTESTED`를 명시한다. 해당 broader module이 보조 입력 없이 동작한다는 증거가 아니다. 원래 no-prerequisite 실패 실험은 별도 증거로 보존한다. 현재 evidence cache는 없고 매번 새 transaction을 실행한다.
- 보조 입력의 source ownership과 metadata 한도의 owner도 구분한다. 공식 PyPI에서 선택하더라도 기존 canonical root context의 Simple 한도를 적용한다. 기본 PyPI의 1,024개, 등록된 PyTorch root의 기존 8,192개 한도와 response byte bound는 변경하지 않는다. 이 선택은 패키지 이름이나 artifact 출력에 따르지 않는다.
- 원래 resolved graph, dependency edges, Verified Set, target SBOM 및 promotion ownership을 확장하지 않는다. 보조 wheel과 생성 script는 inspection volume에서만 사용되고 target에 복사하지 않는다. 사용자가 별도로 선택한 dependency와 혼동하거나 요청 설치의 실제 base 실패를 숨겨 ALLOW할 수 없다. Policy의 다른 필수 조건과 bounded observation의 assurance limit은 그대로다.

정확한 entry-point module이 인증된 site directory의 implicit namespace인 경우에도 독립 unit으로 검사한다. Admission은 controlled parent search에서 각 local component를 찾고 origin/namespace containment를 확인한 뒤 정확한 전체 module을 import한다. 이 사전 선택을 위해 parent artifact 코드를 먼저 실행하거나 mutable `__file__`/`__path__`를 독립 attestation으로 사용하지 않는다.

검증된 `RECORD`에서 선언된 `Import-Name`·`Import-Namespace`, Python `.py` 모듈 및 Python extension 모듈이 전혀 없고 구조적으로 실행 가능 Python 표면이 없음이 정적으로 입증된 wheel(metadata-only 또는 native library/header 등 native/data-only wheel)은 Python import 적용 대상이 아님을 정적으로 확정할 수 있다. 이 경우에도 동일한 격리 session에서 exact wheel closure를 offline 설치하고 설치된 distribution identity를 확인하며 observer 완료를 요구한다. 결과에는 import `NOT_APPLICABLE`을 명시한다. 설치 가능 payload가 있지만 import surface를 안전하게 확정할 수 없거나 모호한 wheel은 이 예외에 해당하지 않으며 기존처럼 fail-closed다.

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
- 서로 다른 distribution의 `RECORD`가 같은 `site-packages` 파일을 가리키면 모든 선언 hash·size가 설치된 동일 파일과 일치하는 경우에만 한 destination에 exact 다중 소유자(distribution/version/artifact digest)를 기록한다. scripts/data destination의 중복이나 서로 다른 내용은 거부한다. 공유 파일 소유자 일부만 갱신하는 transaction도 거부하여 이전 소유자의 파일을 조용히 교체·삭제하지 않는다.

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
