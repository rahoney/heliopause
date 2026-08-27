# M12 Entry Decision — Ecosystem Expansion Before Public Release

M12는 M11까지 완성된 Heliopause Artifact Airlock(HAA)의 공통 trust boundary를
추가 생태계에 재사용하여, 첫 public release 전에 Python AI·Go·Rust·Terraform
개발 흐름까지 지원 범위를 넓히는 마지막 기능 확장 milestone이다.

M12의 목적은 새로운 Core를 재설계하는 것이 아니다. 기존의
`resolve/acquire → exact identity/integrity → static/dynamic inspection →
Policy → Verified Set → transactional Promotion` 구조를 유지하면서
ecosystem-specific adapter, resolver, project transaction, sandbox profile과
qualification만 추가한다.

M12가 완료되면 기능 추가를 중단하고 M12-02 최종 red-team/fix gate를 거쳐
M13 Production Release & Operations로 이동한다.

---

## 1. Scope freeze

첫 public release 전 추가 지원 범위는 다음으로 고정한다.

| Ecosystem | Canonical public source | User-facing HAA command | Initial scope |
| --- | --- | --- | --- |
| Python / PyTorch | PyPI + official PyTorch index/download endpoints | `helox pip install` | 공식 PyTorch wheel source profile 추가 |
| Go | `proxy.golang.org` + `sum.golang.org` | `helox go get`, `helox go mod download`, `helox go build` | public Go Modules only |
| Rust | crates.io / official crates.io sparse index + declared crate download endpoint | `helox cargo add`, `helox cargo build` | public crates.io only |
| Terraform | `registry.terraform.io` + registry가 반환한 exact HashiCorp/Vendor provider download endpoint | `helox terraform init` | public Provider installation only |

다음은 M12 범위에서 제외한다.

- arbitrary PyPI/PyTorch index URL, `--extra-index-url`
- private Python index
- Go private module, `GOPRIVATE`, VCS `direct` fallback
- Cargo alternate/private registry, git dependency
- Terraform private registry, private provider, remote module acquisition
- `terraform plan`, `terraform apply`
- Java/Maven/Gradle, Helm, 기타 ecosystem
- 기존 Core/Policy/Evidence architecture의 전면 재설계

M12 이후 첫 release 전에는 새로운 ecosystem을 더 추가하지 않는다.

---

## 2. Cross-ecosystem invariants

모든 새 adapter는 기존 HAA invariant를 그대로 따른다.

```text
untrusted external source
→ bounded resolver
→ exact dependency graph freeze
→ exact artifact acquisition
→ digest/signature/checksum verification
→ static inspection
→ Linux gVisor dynamic observation where execution occurs
→ Policy
→ exact Verified Set
→ private mutation/build
→ post-mutation verification
→ atomic commit / rollback
```

공통 규칙:

1. **Source is explicit.**
   - ambient registry/proxy/index 환경변수를 상속하지 않는다.
   - 사용자가 임의 URL을 trusted source로 승격할 수 없다.

2. **No weaker fallback.**
   - canonical source가 실패하면 VCS, alternate registry, mirror, direct URL 등으로
     자동 우회하지 않는다.

3. **Exact graph only.**
   - resolver가 선택한 exact version, dependency edge, digest/checksum과
     source identity를 freeze한다.
   - 검사된 graph와 Promotion graph가 달라지면 실패한다.

4. **Project mutation is transactional.**
   - ecosystem control file과 dependency target을 transaction set으로 취급한다.
   - concurrent mutation, target drift, partial commit, rollback uncertainty는
     fail-closed한다.

5. **Execution is observed, not assumed safe.**
   - build script, provider executable, package install/build code가 실행되면
     M11 observer의 process/filesystem/network/honeytoken boundary를 사용한다.
   - required observation이 incomplete이면 ALLOW하지 않는다.

6. **Evidence stays normalized.**
   - raw argv/path/environment/credential을 Evidence에 그대로 보존하지 않는다.
   - source/version/digest/check/result/limitation만 canonical schema에 기록한다.

---

# 3. M12-001 — Official PyTorch source support

## Goal

기존 PyPI/pip adapter와 venv transaction을 재사용하면서 공식 PyTorch wheel
배포 경로를 지원한다. AI 개발자가 `torch`, `torchvision`, `torchaudio` 등
공식 PyTorch 패키지를 HAA를 통해 설치할 수 있게 한다.

## Public UX

예시:

```bash
helox pip install torch --source pytorch:cpu
helox pip install torch --source pytorch:cuXXX
```

실제 허용 source profile 이름과 CUDA profile 목록은 코드에 흩어놓지 않고
canonical source/runtime lock에서 관리한다.

일반 PyPI는 기존처럼:

```bash
helox pip install requests
```

를 유지한다.

## Source boundary

허용:

- `pypi.org` / PyPI Simple API
- `files.pythonhosted.org`
- official PyTorch index root
- 해당 official PyTorch index가 사용하는 검증된 wheel download host

금지:

- arbitrary `--index-url`
- arbitrary `--extra-index-url`
- user-supplied mirror
- credentials가 필요한 private index
- direct URL install

PyTorch source는 URL 문자열 입력이 아니라 **named source profile**로 선택한다.

## Resolver rule

PyTorch와 PyPI를 동시에 “가장 높은 버전을 주는 index” 방식으로 경쟁시키지 않는다.

권장 partition:

```text
requested PyTorch-owned package
→ selected official PyTorch source profile

ordinary transitive dependency
→ canonical PyPI

same project name이 두 source에 동시에 존재
→ explicit source ownership rule로 하나만 선택
→ ambiguous면 fail closed
```

dependency confusion을 막기 위해 source selection은 project name과 canonical
source policy에 의해 결정되어야 한다.

### Canonical source ownership

PyTorch source ownership은 resolver 내부의 여러 조건문에 흩어놓지 않고
**하나의 canonical policy table**이 소유한다.

최소 계약:

```text
PyTorch-owned project
→ 사용자가 선택한 named PyTorch source profile에서만 resolve/acquire

ordinary Python project
→ canonical PyPI에서만 resolve/acquire

동일 project name이 두 source에서 관찰됨
→ canonical ownership table과 일치하는 source만 허용
→ ownership이 없거나 ambiguous하면 fail closed
```

초기 PyTorch-owned project set은 실제 first-release support 범위만 명시적으로
등록한다. `torch`, `torchvision`, `torchaudio`처럼 지원을 선언한 project 이외의
이름을 package-name similarity만으로 PyTorch-owned라고 추론하지 않는다.

resolver가 생성한 dependency graph의 각 node에는 source identity를 함께 freeze한다.
따라서 같은 name/version/digest처럼 보여도 source identity가 달라지면 동일 artifact로
취급하지 않는다.

named source profile은 exact index/download endpoint policy와 결합되며,
사용자가 arbitrary URL을 profile처럼 주입할 수 없다.

## PyTorch resource policy

Large-artifact resource budget의 선택 기준은 artifact node의 source identity가
아니라 **install request의 named root SourceProfile**이다. 예를 들어
`helox pip install torch --source pytorch:cu126`의 transaction 전체는
`pytorch:cu126` resource budget을 사용한다.

```text
root resource profile: pytorch:cu126
graph node: torch              → source identity pytorch:cu126
graph node: ordinary dependency → source identity pypi
graph node: NVIDIA dependency   → canonical ownership에 따른 source identity
```

각 node의 `project/version/digest/source identity` freeze는 바꾸지 않는다.
다만 graph 전체의 artifact·disk·duration resource budget은 root named profile이
소유한다. 따라서 일반 `helox pip install <large-pypi-package>` 요청은 PyTorch용
확대 budget을 획득할 수 없으며, arbitrary URL/source/custom profile도 확대
budget을 선택할 수 없다.

### First-release bounded budgets

아래 수치는 target이 아니라 maximum fail-closed bound다. 어느 한 bound라도
초과하면 partial success 없이 transaction을 실패시킨다. default PyPI는 기존의
conservative limits를 유지한다.

| Root resource profile | Max artifact compressed | Max artifact uncompressed | Max files/artifact | Max bounded metadata file | Max graph compressed | Max graph uncompressed | Max transaction temporary disk | Qualification duration |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| default PyPI | existing conservative limits | existing conservative limits | existing conservative limits | existing conservative limits | existing conservative limits | existing conservative limits | existing conservative limits | existing conservative limits |
| `pytorch:cpu` | 256 MiB | 1 GiB | 20,000 | 2 MiB | 512 MiB | 2 GiB | 4 GiB | 15 minutes |
| `pytorch:cu126` | 1 GiB | 2.5 GiB | 20,000 | 2 MiB | 4.5 GiB | 8 GiB | 24 GiB | 40 minutes |

기존 구현에 대응 resource-bound abstraction이 있으면 그것을 canonical owner로
재사용한다. 이 계약은 별도의 중복 abstraction을 요구하지 않는다.

### Runtime and staging resource contract

현재 PyPI dynamic/promotion composition의 256 MiB tmpfs, 128 MiB tmpfs,
512 MiB memory는 PyTorch profile에 그대로 적용할 수 없다. global runtime
limit를 상향하지 않으며, root SourceProfile이 선택한 bounded runtime resource
policy만 기존 sandbox/promotion composition이 소비한다. default PyPI runtime
limits는 변경하지 않는다.

staging과 dynamic introduction의 `source == "pypi"` 전용 허용은
`pytorch-cpu`·`pytorch-cu126` 문자열 예외를 여러 위치에 추가하는 방식으로
확장하지 않는다. 기존 canonical `SourceProfile`/`ProfileForSource` policy로
지원되는 official Python source identity를 판정한다.

### Resource preflight

대용량 acquisition 전에 가능한 범위에서 graph declared/content-length total,
artifact count, required temporary disk, execution capability를 preflight한다.
명백한 budget 초과는 수 GiB를 일부 내려받은 뒤가 아니라 가능한 가장 이른
boundary에서 fail closed한다. metadata가 불완전해 사전 확정할 수 없는 값은
streaming runtime counter로 계속 enforce한다.

## Reuse

가능한 기존 구현:

- PyPI reference/version normalization
- pip report normalization
- wheel/sdist integrity machinery
- venv transaction
- static inspection
- Linux dynamic sandbox
- Evidence/Policy/Verified Set

Required capability / implementation responsibilities:

- PyTorch source profile
- official index parser/normalizer
- source-specific endpoint allowlist/identity
- mixed-source dependency partition
- large wheel/resource bound handling
- PyTorch-specific integration fixtures

## Qualification

최소:

- CPU wheel 정상 install
- 최소 1개 pinned CUDA profile resolution/install qualification
- canonical source-ownership table에 등록된 PyTorch-owned project만 PyTorch source에서 resolve
- requested PyTorch package가 PyPI로 source-confusion 되지 않음
- ordinary PyPI dependency가 PyTorch index의 동명 project로 대체되지 않음
- source ownership이 없거나 ambiguous한 동일 project name은 fail closed
- dependency graph 각 node의 source identity가 freeze되고 Promotion까지 유지됨
- arbitrary index URL 거부
- tampered wheel/digest mismatch BLOCK
- incomplete dynamic observation은 ALLOW 금지
- existing PyPI regression 전부 통과

PR Required CI는 deterministic source ownership/resource-bound regression,
default PyPI conservative-limit regression, PyTorch CPU full Linux/gVisor E2E,
CUDA resolver/source/graph/resource-policy deterministic test를 실행한다.

full CUDA E2E는 M12 qualification 또는 bounded `workflow_dispatch`/release
qualification에서 수행한다. 실제 pinned CUDA profile의
resolution/acquisition/inspection/install/Promotion, Linux gVisor observation,
disk/resource preflight, timeout/resource fail-closed 결과와 evidence를 포함한다.
수 GiB CUDA download를 모든 PR의 required path로 만들지 않는다. 단,
M12-001을 `COMPLETE`로 닫기 전에는 최소 한 번의 실제 pinned CUDA profile full
E2E qualification evidence가 반드시 존재해야 한다.

다음 invariant는 resource-policy 구현과 qualification에서도 유지한다.

- arbitrary index·URL·custom source 없음
- default PyPI enlarged budget 없음
- node별 source identity freeze 및 graph source-confusion fail closed
- resource budget 초과 fail closed, partial staging/promotion 없음
- observer incomplete는 ALLOW가 아님
- Core/Policy/Evidence architecture 변경 없음

Execution status: see [Step 13 — Current Work Queue](./02-current-work-queue.md)

---

# 4. M12-002 — Go Modules

## Goal

public Go Module dependency를 HAA를 통해 resolve/verify/inspect한 뒤 Go project의
dependency state를 transactionally 갱신한다.

## Public UX

```bash
helox go get example.com/module@v1.2.3
helox go mod download
helox go build ./...
```

`go mod`는 command group이므로 첫 release에서 명시적으로 지원하는 `go mod`
operation은 `go mod download`로 한정한다.

`helox go build`를 first-release contract에 포함한다. 이유는 HAA가 검증한 module을
격리 cache에서 확인한 뒤 사용자가 일반 `go build`를 실행하면서 dependency를 다시
외부에서 받아버리는 trust gap을 남기지 않기 위해서다.

first release에서는 `go test`, `go run`, `go install`, `go generate`를 자동으로
`go build`와 동등 지원한다고 주장하지 않는다.

## Source boundary

허용:

```text
GOPROXY=https://proxy.golang.org
GOSUMDB=sum.golang.org
```

격리 resolver에서 다음 ambient 값은 신뢰하지 않는다.

```text
GOPROXY
GOSUMDB
GOPRIVATE
GONOPROXY
GONOSUMDB
GOVCS
```

M12에서는:

```text
proxy.golang.org only
sum.golang.org verification required
direct VCS fallback = forbidden
private module = unsupported
```

로 고정한다.

## Identity and integrity

freeze 대상:

- module path
- exact semantic version
- module `.mod`
- module `.zip`
- dependency graph
- `go.sum` h1 identity
- SumDB verification result
- source endpoint identity

requested version이 symbolic selector이면 resolver가 exact version으로 freeze한 뒤
그 이후 단계에서는 selector를 다시 평가하지 않는다.

## Project transaction

transaction set:

```text
go.mod
go.sum
HAA Go transaction metadata
```

`go get`은 private workspace에서 먼저 수행한다.

```text
target discovery
→ mutation guard
→ snapshot go.mod/go.sum
→ private resolver
→ graph freeze
→ acquire/verify/inspect
→ private go get
→ post-state verify
→ atomic control-file commit
```

기존 unmanaged project를 자동으로 HAA-managed 상태라고 주장하지 않는다.
초기 adoption policy는 npm M9 원칙과 같은 방향으로 bounded하게 정의한다.

`go mod download`와 `go build`는 HAA가 소유·검증하는 module cache boundary를
사용한다. 사용자 global module cache를 resolver trust boundary로 사용하지 않는다.

`go get`은 사용자가 지정한 exact module을 primary Artifact로 하는 기존
`DependencyResolution` 계약을 사용한다. 반면 `go mod download`와 `go build`는
현재 project의 complete lock snapshot을 다루므로 arbitrary dependency를 primary로
만들지 않는다. 이 두 operation은 project root·`go.mod`/`go.sum` digest·complete
verified graph를 함께 bind하는 project snapshot contract를 사용하며, snapshot이
없는 generic install/promotion workflow로 우회하지 않는다.

cache는 두 단계로 구분한다.

```text
private resolver/build cache
→ exact graph acquire + verify + inspect

HAA-managed verified module cache
→ 검증이 끝난 exact module만 digest/source identity와 함께 Promotion
```

일반 Host `GOMODCACHE`의 기존 content를 “이미 안전한 dependency”로 신뢰하지 않는다.
HAA-managed cache에 Promotion할 때는 exact module path/version/content digest와
SumDB/source identity를 다시 bind한다.

### `helox go build` contract

`helox go build`는 현재 project의 `go.mod`/`go.sum`을 snapshot한 뒤 exact dependency
graph를 다시 확인하고 **HAA-managed verified cache만 사용하여** isolated build를
수행한다.

권장 흐름:

```text
project discovery
→ mutation/read guard
→ snapshot go.mod/go.sum
→ exact graph re-validation
→ verified module cache materialization
→ network-disabled isolated build
→ process/filesystem/network observation
→ build completion + graph unchanged verification
→ bounded build output publish
```

build 단계에서는 dependency acquisition network를 허용하지 않는다. 필요한 module이
verified cache에 없으면 build 중 외부 download로 보충하지 않고 resolver 단계로
되돌아가 검증한 뒤 다시 build한다.

## Inspection

Go module source는 기본적으로 static inspection한다.

`helox go build`에서는 build-time execution을 M11 sandbox/observer boundary로
관찰한다. 특히 compiler/linker/cgo/helper subprocess, workspace 밖 filesystem access,
network attempt와 resource/incomplete observation을 normalized signal로 처리한다.

first release에서 다음 execution path는 지원하지 않는다.

- `go generate`
- arbitrary tool binary
- VCS hooks
- `go test`
- `go run`
- `go install`

따라서 `helox go build` qualification을 다른 Go command의 안전성 보증으로 확장해
주장하지 않는다.

## Qualification

- public module exact version
- transitive module graph
- SumDB mismatch
- proxy response tamper
- `direct` fallback 유도
- `GOPRIVATE`/ambient env injection
- user global `GOMODCACHE` poisoning이 verified cache로 신뢰 승격되지 않음
- `helox go build`가 network-disabled 상태에서 verified cache만 사용
- build 중 missing module이 있으면 direct download 없이 fail closed
- build-time unexpected process/filesystem/network observation
- concurrent `go.mod`/`go.sum` mutation
- build output publish/rollback fault
- existing npm/PyPI/GitHub regression

Execution status: see [Step 13 — Current Work Queue](./02-current-work-queue.md)

---

# 5. M12-003 — Rust / Cargo

## Goal

crates.io dependency를 exact Cargo graph로 freeze하고, Cargo project mutation과
build-time execution을 HAA sandbox에서 검사한다.

## Public UX

```bash
helox cargo add serde@1.0.XXX
helox cargo build
```

`cargo add`와 `cargo build`는 서로 다른 operation contract를 가진다.

```text
cargo add
→ Cargo.toml/Cargo.lock dependency transaction

cargo build
→ existing locked graph를 exact acquire
→ isolated build
→ build-time dynamic observation
→ verified build output Promotion
```

## Source boundary

허용:

- crates.io
- official crates.io sparse index
- index metadata가 선언한 canonical crate download endpoint

금지:

- alternate registry
- private registry
- git dependency
- path dependency가 project trust boundary 밖을 참조하는 경우
- user-provided registry replacement/mirror

격리 실행에서 ambient Cargo config와 credential은 상속하지 않는다.

## Identity and integrity

freeze:

- crate name
- exact version
- registry identity
- checksum
- dependency graph
- feature selection
- target/platform relevant resolution state
- Cargo.lock exact state

`Cargo.lock`의 checksum과 acquired `.crate` content가 일치하지 않으면 BLOCK한다.

## Dynamic risk boundary

Rust는 다음 build-time code execution 때문에 M11 observer 재사용 가치가 높다.

- `build.rs`
- procedural macro
- compiler/linker subprocess
- native build helper

`helox cargo build`는 Linux에서 gVisor sandbox + ecosystem profile을 사용한다.

관찰:

- unexpected process
- workspace 밖 filesystem access
- network attempt
- honeytoken access
- resource exhaustion / dropped event / incomplete stream

required observation incomplete → no ALLOW.

## Project transaction

`cargo add` transaction set:

```text
Cargo.toml
Cargo.lock
HAA Cargo transaction metadata
```

`cargo build` output set:

```text
locked graph
verified crate cache/input set
target build output selected for Promotion
HAA build record
```

기존 target directory를 무조건 덮어쓰지 않는다. private build tree에서
post-build verification 후 bounded commit한다.

## Qualification

- normal crate add
- transitive dependency graph
- checksum mismatch
- registry substitution
- git dependency rejection
- malicious `build.rs` network attempt
- malicious `build.rs` filesystem escape
- proc-macro execution observation
- concurrent Cargo.toml/Cargo.lock mutation
- build rollback/cleanup uncertainty

Execution status: see [Step 13 — Current Work Queue](./02-current-work-queue.md)

---

# 6. M12-004 — Terraform Provider

## Goal

Terraform public Provider 설치를 HAA transaction으로 보호한다.

첫 release의 Terraform scope는 **Provider installation only**다.

## Public UX

```bash
helox terraform init hashicorp/aws@5.50.0
```

지원하지 않음:

```text
terraform plan
terraform apply
private registry/provider
remote module acquisition
arbitrary provider mirror
```

project가 M12 범위를 벗어난 remote module acquisition을 요구하면 명확한
unsupported/fail-closed 결과를 낸다. HAA가 모르는 source로 자동 진행하지 않는다.

## Source boundary

canonical discovery:

```text
registry.terraform.io
```

Provider binary는 registry protocol이 선택한 exact:

```text
namespace/type/version/platform
→ download URL
→ checksum metadata
→ signature/signing identity
```

에 의해 연결된다.

HashiCorp/Vendor download host를 전역적으로 모두 신뢰하지 않는다.

```text
registry response가 exact provider/version/platform에 대해 반환한 URL
AND endpoint policy 허용
AND checksum/signature binding 성공
```

일 때만 해당 transaction에서 acquisition한다.

## Integrity and signing

검증 대상:

- provider namespace/type
- exact version
- OS/architecture
- archive SHA-256
- checksum document
- signature where provided/required
- signer identity/trust tier
- `.terraform.lock.hcl` hashes
- extracted provider executable digest

Policy 예시:

```text
trusted HashiCorp/vendor signer + exact checksum
→ inspection 계속

recognized partner signer + exact checksum
→ inspection 계속

self-signed/community signer
→ explicit policy에 따라 MANUAL_REVIEW 또는 pinned signer 요구

unsigned / mismatch / ambiguous identity
→ BLOCK
```

정확한 signer classification은 adapter의 canonical policy table 하나가 소유한다.

## Transaction set

```text
.terraform.lock.hcl
.terraform/providers/...
HAA Terraform transaction metadata
```

흐름:

```text
project discovery
→ mutation guard
→ current lock/provider state snapshot
→ registry resolution
→ exact download/checksum/signature verification
→ archive static inspection
→ provider executable dynamic probe where supported
→ private `.terraform` workspace
→ post-state verify
→ atomic commit / rollback
```

## Existing GitHub Release reuse

Provider download URL이 GitHub Release를 가리키는 경우 기존 GitHub acquisition의
저수준 HTTPS/download/content-digest component를 재사용할 수 있다.

그러나 Terraform provider identity는 반드시:

```text
registry metadata
+ namespace/type/version/platform
+ checksum/signature
+ .terraform.lock.hcl
```

에 bind되어야 한다.

기존 `github-release` adapter의 ALLOW를 Terraform Provider ALLOW로 그대로
재사용해서는 안 된다.

## Qualification

- HashiCorp official provider
- 최소 1개 vendor/partner provider
- checksum mismatch
- signature mismatch
- registry metadata/download URL mismatch
- alternate mirror injection
- lock-file drift
- provider executable dynamic observation
- concurrent init
- partial `.terraform` commit/rollback

Execution status: see [Step 13 — Current Work Queue](./02-current-work-queue.md)

---

# 7. M12-005 — Cross-ecosystem qualification and feature freeze

M12-001~004 완료 후 전체 system을 한 번에 qualification한다.

## Required paths

기존:

```text
npm
PyPI/pip
GitHub Release
```

신규:

```text
PyTorch official source
Go Modules
Cargo/crates.io
Terraform Provider
```

## Required qualification

각 생태계에서 최소:

```text
normal exact install
transitive dependency
source substitution attack
digest/checksum mismatch
resolver network failure
sandbox observation incomplete
unexpected process/filesystem/network signal where applicable
target mutation race
Promotion rollback
Evidence generation
CLI exit/result contract
```

을 검증한다.

그리고 기존 M7/M8/M9/M11 hostile regression을 다시 실행한다.

## Feature freeze condition

다음이 모두 만족되면 M12 기능 개발을 종료한다.

- [ ] PyTorch official source supported without arbitrary index fallback
- [ ] PyTorch canonical source-ownership table prevents cross-index dependency confusion
- [ ] Go proxy + SumDB exact graph qualification
- [ ] `helox go build` uses only HAA-managed verified modules and passes build-time observation qualification
- [ ] Cargo add/build and build-time observation qualification
- [ ] Terraform Provider init/checksum/signature qualification
- [ ] all existing npm/PyPI/GitHub tests remain green
- [ ] Linux gVisor integration green
- [ ] canonical `Required` CI green
- [ ] docs/CLI accurately state supported and unsupported paths

이 시점 이후 Java, Helm, private registries, alternate mirrors 등은 첫 release 뒤로
넘긴다.

Execution status: see [Step 13 — Current Work Queue](./02-current-work-queue.md)

---

# 8. Handoff to M12-02 red-team fix gate

M12 기능 구현 완료 직후 `17-m12-02-fix-list.md`에 final red-team 결과를 기록한다.

그 검토는 새로운 기능 제안을 찾는 과정이 아니라 다음 release-blocking class만 찾는다.

```text
fail-open
trust-boundary bypass
wrong source / wrong artifact identity
incomplete graph promoted as exact
sandbox attribution/observation break
transaction data loss / rollback violation
evidence-policy mismatch
cross-ecosystem regression
```

발견 사항이 없으면 `NO_RELEASE_BLOCKING_FINDINGS`로 닫고 즉시 M13으로 이동한다.

---

# 9. Exit

M12와 M12-02 fix gate가 완료되면 HAA의 first-release feature set은 고정된다.

```text
npm
PyPI / pip
PyTorch official source
Go Modules
Cargo / crates.io
Terraform Provider
GitHub Releases
```

Next milestone:

```text
M13 — Production Release & Operations
```

Execution status: see [Step 13 — Current Work Queue](./02-current-work-queue.md)
