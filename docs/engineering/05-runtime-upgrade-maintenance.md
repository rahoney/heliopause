# Runtime upgrade maintenance design

상태: 2026-10-15 UPG-003 구현 기준. Ownership/manifest/drift와 developer
plan/apply/verify는 구현됐으며 실제 단계별 검증 상태는 Queue/evidence를 따른다.
gVisor/compiler migration, installed qualification과 activation은 후속 목표다. 실행 순서와 작업 상태는
[유지보수 계획](../planning/19-runtime-upgrade-maintenance-plan.md)과
[Current Work Queue](../planning/02-current-work-queue.md)가 소유한다.

## 1. 목적과 경계

HAA가 관리하는 compiler·runtime·quality tool을 명시적으로 선택한 버전으로
갱신할 때, 연관 설정과 생성 결과의 누락을 막고 같은 절차로 검증·전환한다.
일상적인 patch upgrade는 pin·배포물 갱신과 검증으로 처리하고, 실제 호환성
결함이 있는 모듈만 수정한다. 모든 언어의 출시일에 맞춘 자동 latest 추종은 하지 않는다.

기존 Modular Monolith, Port/Adapter, Policy와 resource bound를 유지한다.
유지보수 command는 repository 개발 도구이며 `helox`의 새 사용자 operation,
daemon, plugin system 또는 범용 package manager가 아니다. 기존 lock·generator·
check·build·installer·lifecycle을 확장한다. 동작 중인 구현의 대체는 이 설계의 범위가 아니다.

관련 계약:
[Architecture](../architecture/01-foundation-and-dependencies.md),
[Trusted tooling](../threat-model/04-trusted-tooling-and-evidence.md),
[M8](../planning/11-m8-production-trust-hardening-contract.md),
[M10](../planning/13-m10-verified-distribution-bootstrap-contract.md),
[M12](../planning/16-m12-01-ecosystem-expansion-contract.md),
[CI](./04-ci-quality-gate.md).

## 2. 관리 대상과 단일 owner

단일화는 한 거대 파일에 모든 정보를 복제한다는 뜻이 아니다. 각 값의 입력
owner는 하나이고, 다른 소비자는 그 값을 읽거나 검증 가능한 생성 결과를 사용한다.

| 대상 | 입력 owner / 현재 소비자 | 갱신 책임 |
| --- | --- | --- |
| HAA Go compiler | `scripts/runtimes.lock.json`의 `go_image.go_version`을 선택 버전 owner로 사용 | `go.mod` floor, tool `setupGo`, CI/release setup, guest tag/version/digest와 host SDK를 연결 |
| Node/npm | runtime lock의 Node image와 bundled npm identity | Node tag/version과 실제 image contents 일치; resolver·probe·Promotion 공유 |
| Python/pip | runtime lock의 Python image, pip와 ABI/platform target | interpreter·pip·wheel tags·Promotion·PyTorch 지원 tuple 영향을 함께 계산 |
| Rust/Cargo | runtime lock의 Rust image/version/target | resolver/build와 runtime image manifest의 누락 없는 연결 |
| gVisor | runtime lock의 source commit·patch·builder·Bazel·bundle members | Go SDK 선택을 HAA Go owner에 연결하고 전체 bundle·observer를 재빌드 |
| Docker/containerd/Bazel | runtime lock의 exact package/build identities | 변경 계획과 호환성 검증; OS package 설치나 daemon 재설정을 자동 끼워 넣지 않음 |
| Quality tools | `scripts/tools.lock.json`의 package/release/assets | 도구 자체 version은 유지하거나 명시적으로 선택; compiler는 Go owner에서 파생 |
| 지원 검토 | `scripts/version-support.lock.json` | selected version 일치와 review provenance 확인; 검토 날짜·최신판·지원 판정을 기계적으로 꾸미지 않음 |
| Terraform Provider | 사용자 요청/constraints/`.terraform.lock.hcl`, registry metadata, HCL parser의 `go.mod` 의존성 | version/platform/hash/signature/protocol 검사; HAA-owned Terraform CLI는 현재 없음 |
| GitHub Release asset | 사용자 exact release/asset identity와 verifier 계약 | tag/asset/digest/signature 처리 회귀; 공통 언어 SDK와 독립 |
| PyTorch CPU/CUDA | M12 지원 tuple·named source·resource policy와 qualification evidence | Python target/package graph/source/common runtime 영향의 별도 판정 |

사용자 project의 Go/Cargo/Provider/package 버전이나 서명된 검사 대상 binary는
HAA compiler 버전으로 재작성·재빌드하지 않는다. OS Python, 다른 프로젝트의
micromamba/nvm 환경도 자동 정리 대상이 아니다. HAA-owned SDK와 그 SDK로 빌드해
배포하는 helper/runtime을 누락 없이 관리한다.

기존 JSON schema는 명시적 version migration으로만 확장한다. Node version처럼
현재 tag에서 암묵적으로 얻는 값을 명시 필드로 추가하면 image tag와 양방향 검증한다.
Host Go archive의 platform/URL/size/SHA-256은 해당 Go 선택에 연결한 frozen proposal의
배포물 metadata로 둔다. 선택 버전의 owner는 runtime lock 하나로 유지하며,
developer SDK 준비만을 위해 product runtime lock schema를 확장하지 않는다.
새로운 독립 version lock을 만들지 않는다.

### Quality tool의 실제 compiler 확인

2026-10-15 local metadata 감사에서 source-installed Staticcheck/gosec/govulncheck/
actionlint는 Go1.26.9였고 official archive Gitleaks8.18.4는 Go1.22.1이었다.
Tool lock의 `setupGo=1.26.9`는 prebuilt archive의 compiler를 바꾸지 않는다.
이는 별도1.22.1 SDK 설치 발견이 아니라 binary의 build metadata다.

HAA가 관리하는 Go quality binary도 실제 compiler를 확인한다. Gitleaks는 현재
탐지 동작 때문에 선택한8.18.4를 유지하면서 공식 exact source/commit과 검증된
module graph를 고정한1.26.9 build 후보를 준비한다. 기존 source install 기능을
재사용하고 version/ldflags·source provenance·output digest를 연결한다. Source build
output에 기존 official archive hash를 붙이지 않는다. 기존 검토의 검출/미검출 사례를
회귀 corpus로 고정해 이전 archive와 새 binary를 비교하고 release 선택 이유와 탐지 범위를 유지한다.
호환성·source identity가 불명확하면 기존 archive의 단일화가 완료됐다고 표시하지 않는다.

## 3. 재사용과 변경 위치

- `scripts/generate-runtime-lock.go`: immutable product identities의 기존 owner.
  생성·검사 경로를 확장하며 기존 validation을 우회하거나 두 번째 generator를 만들지 않는다.
- `scripts/check`: canonical read-only 검사의 owner. 일관성·누락 검사를 기존
  foundation/runtime drift 단계에 연결한다. `quick`, Required와 release gate는 유지한다.
  `workflow.go` 등의 selected Go literal도 소비자다. 현재 선택값은 lock에서 읽게
  연결하며 retired image 금지 token·negative fixture의 과거 버전은 보존한다.
- `scripts/runtime-image-manifest`: runtime lock의 배포용 파생 결과. 현재 Node/Python/Go
  목록에 Rust를 포함하고 각 `runtime_version`이 해당 interpreter/compiler를 나타내게 한다.
  현재 Node entry가 npm version을 쓰는 점은 수정 대상이다.
- `scripts/build-gvisor-runtime-bundle.sh`, `scripts/build-gvisor-observer-release.sh`와
  CI observer build: 같은 source·patch·SDK 준비를 사용한다. Bundle member 검증은 유지한다.
- `internal/releaseinstall`, M8 trusted executor와 기존 integration lifecycle:
  검증된 배포물·설치·종료 계약을 재사용한다. 유지보수 도구에 별도 privileged
  lifecycle이나 임의 shell executor를 만들지 않는다.

새 command가 필요하면 `scripts/update-runtime` 하나에 한정한다. 초기에는
같은 directory의 몇 개 Go 파일과 작은 typed component switch로 구현한다.
새 production package·interface 계층·dependency는 실제 gap이 없으면 추가하지 않는다.
기존 parser/library 재사용도 Architecture 허용 범위를 확인하며 allowlist를 편의로 넓히지 않는다.

## 4. 명시적 업데이트 흐름

구현할 developer entrypoint는 `plan`, `apply`, `verify` 세 단계다. 실제 flag와
정확한 파일 목록은 각 단계 구현 commit에서 help·tests·receipt로 확정한다.

```text
explicit component/version request
→ plan: baseline inventory + official identity resolution + affected consumers
→ frozen candidate: new lock metadata + bounded workspace patch
→ apply: precondition comparison + declared generated changes
→ verify: canonical checks + affected integrations/qualification
→ existing authorized installation procedure + doctor/readback
→ confirmed old-owned SDK retirement
```

### plan

- Git HEAD, tracked/index diff, selected input hashes와 설치 관측을 기록한다.
  원래 사용자 변경을 덮지 않으며 후보는 별도 directory/worktree에서 준비한다.
- 네트워크 접근은 official source에서 요청한 exact version의 배포물 identity를
  얻는 명시 단계다. URL·repository·architecture·digest/checksum을 확인한다.
  버전 번호만 바꿔 이전 hash를 붙이거나 tag를 실행 authority로 사용하지 않는다.
- Offline check는 이미 고정한 metadata만 읽으며 network resolve를 실행하지 않는다.
  Version equality만으로 image integrity나 executable contents 검증을 대신하지 않는다.
- 요청 component가 원인이고 동반 변경은 dependency edge로 설명한다.
  Unknown component/consumer/metadata·지원되지 않는 조합은 누락 대신 미완료로 보고한다.
- Same-version 요청은 version upgrade가 아닌 reconciliation이다. 반복하면 같은
  후보가 나오며 입력이 같을 때 semantic diff가 없어야 한다.

### apply

- plan의 baseline HEAD/hash와 현재 입력을 비교한다. 변경됐으면 후보를 다시 계산한다.
- 실제 적용은 같은 baseline의 새 isolated Git worktree에서 수행한다. 원래 checkout의
  사용자 source는 수정하지 않고 candidate diff와 검증 결과를 일반 Git review 경로로
  전달한다. 외부 state directory에 plan/journal을 두어 검사 source에 실행 로그를 섞지 않는다.
- 현재/목표 버전, 배포물 identity와 선언된 생성 파일만 변경한다. Historical log,
  qualification receipt, parser fixture, 사용자 project lock에는 일괄 치환하지 않는다.
- Go directive는 기존 `go.mod`의 지원 문법을 지키고, workflow는 선언된 setup field만
  구조를 확인해 편집한다. 예상 field 수·action identity가 달라지면 실패한다.
  전체 파일에 대한 무차별 숫자 치환을 하지 않는다.
- `apply`는 명시적인 source mutation 경로다. Canonical check profile은 계속
  read-only이며 formatter의 source mutation은 `format`만 소유한다.
- 다중 파일 쓰기를 원자적이라고 주장하지 않는다. Isolated candidate에서 준비하고
  frozen plan의 before/after bytes/hash와 write-ahead journal을 보존한다. 중단 뒤
  각 파일이 정확한 before/after 중 하나일 때만 남은 쓰기를 재개한다. 원래 checkout은
  수정하지 않으므로 source rollback을 가장하지 않는다. 외부 변경은 보존·진단한다.
- Generator 재실행은 byte-identical하고, same-version apply의 두 번째 실행은 no-op이다.

### verify와 Go bootstrap

Go update command는 pin 변경 전에 현재 검증된 SDK로 빌드해 둔다. Target SDK는
별도 후보 경로에 공식 hash/size 대조 후 준비하고, absolute path로 candidate checks를
실행한다. `GOTOOLCHAIN=local` 아래 새 `go.mod`를 이전 SDK로 실행해 실패하거나,
자동 SDK download를 켜서 해결하는 bootstrap 순환을 만들지 않는다.

적용 전 host 기본 SDK를 바꾸지 않고 target SDK로 검증한다. 최종 전환은 성공한
후보와 연결된 기존 설치 절차가 수행한다. Tool cache도 compiler identity에 맞춰
준비하고, 임의 PATH 도구나 floating latest로 대신하지 않는다.

### 구현된 명령과 기록

Owner는 `scripts/update-runtime` 한 directory다. 기존 고정 YAML parser를 setup-go
scalar의 구조/위치 확인에 재사용한다. JSON은 표준 decoder의 token span으로 선언된
field만 편집하며 나머지 bytes와 historical fixture/receipt는 보존한다. 새 dependency나
product interface는 없다. Linux와 macOS를 대상으로 하며 Windows에서는 WSL 경로를 사용한다.

깨끗한 versioned repository에서 현재 SDK로 coordinator를 먼저 빌드한다.
State는 repository와 겹치지 않는 absolute directory이며 재부팅 뒤 복구가 필요하면
`/tmp`가 아닌 영구 저장소를 선택한다. 각 요청은 새로운 state를 사용한다.

```sh
go build -trimpath -o /absolute/persistent/update-runtime ./scripts/update-runtime
/absolute/persistent/update-runtime plan --root . --state /absolute/persistent/go-reconcile --component go --version 1.26.9
/absolute/persistent/update-runtime apply --root . --state /absolute/persistent/go-reconcile
/absolute/persistent/update-runtime verify --root . --state /absolute/persistent/go-reconcile
```

`plan`의 component는 `go`, `node`, `python`, `rust`, `terraform`, `github`다.
앞의 네 component는 exact stable `X.Y.Z`를 받는다. 새 버전은 공식 release metadata와
공식 OCI index에서 linux/amd64 manifest를 고정하고 body digest까지 확인한다.
Node는 공식 release의 npm, Python은 해당 CPython tag의 ensurepip pip를 연결하며,
실제 image contents와 실행 호환성은 후속 runtime qualification에서 확인한다.
Go는 공식 supported download 목록의 Linux amd64/macOS amd64/arm64 archive를 고정한다.
Registry의 anonymous token은 process 안에서만 사용하며 plan/log에 저장하지 않는다.

새 버전 요청에는 `--review /absolute/review.json`을 추가한다. 입력은 기존
`scripts/version-support.lock.json`과 동일한 field를 가진 record의 JSON array다.
변경된 owner만 정확히 포함한다: Go1개, Node/npm2개, Python/pip2개, Rust/Cargo1개.
실제로 검토한 selected/latest/support/decision/reason/date를 제공하고 기존 role과
official source를 유지한다. 도구는 검토 날짜나 지원 결정을 자동 생성하지 않는다.
과거 기록·다른 component를 한꺼번에 새 검토로 표시하지 않는다.

같은 버전은 network resolve·review 갱신·SDK 준비 없이 기존 digest를 보존한다.
2026-10-15의 별도 live metadata probe에서 Node/Python/Rust 태그는 기존 lock과 다른
amd64 digest를 반환했다. 같은 version의 tag를 다시 조회한 것을 같은 frozen image로
취급하지 않으며, reconciliation에 digest refresh를 섞지 않는다.

Terraform/GitHub는 `--version` 없이 adapter/common-runtime 영향만 계획한다.
사용자가 지정한 Provider/package/release asset의 version·hash·signature는 수정하지 않는다.

`apply`는 source HEAD·tracked/index cleanliness·전체 tracked bytes/modes와 선언된
generator/patch를 대조한다. Untracked source/unknown workflow는 거부한다. State의
`candidate`에 detached worktree를 만들고 그곳만 수정한다. Full mode를 보존하며,
두 번째 적용은 이미 after인 파일을 다시 쓰지 않는다. `plan.json`과 `journal.json`을
fsync/rename으로 기록하고 operation lock을 사용한다. 실패 후 같은 `apply`로 재개하되
외부 변경·HEAD/index 변경·불명확한 상태에서는 덮어쓰지 않고 원인을 먼저 조사한다.
Candidate diff는 정상 branch/commit/PR 검토로 전달하며 자동 merge/push는 없다.

`verify`는 bootstrap→quick→security→workflow→release-gate→docs→qualification-freshness를
기존 canonical checker로 실행한다. Go upgrade는 frozen official archive의 size/hash와
bounded safe extraction을 확인한 state-local SDK를 사용하며 전체 SDK inventory를 기록한다.
다른 요청은 선택 Go와 실제 compiler/platform이 같은 absolute executable을 사용한다.
`--go`는 현재 Go 선택을 유지하는 요청의 명시 SDK 경로이며 새 Go에는 ambient SDK를 받지 않는다.
Host 기본 SDK와 설치는 바꾸지 않는다. 각 `verification-*` directory의 logs/result와
최신 `verification.json`을 남겨 첫 실패를 다음 실행으로 덮지 않는다.
UPG-004 compiler 검사 연결 뒤 verify는 `state/quality-tools`의 독립 executable
cache를 명시한다. Caller의 기본 `HELOX_TOOL_CACHE`·Go config/workspace/flags로
후보 bootstrap이 active quality binaries를 덮어쓰지 않으며 cache/bin alias도 거부한다.

`configuration=PASS`는 설정 검증 결과다. `required_qualification`은 변경 종류의
installed/CPU/CUDA/Required/activation 작업을 별도로 남기며 `migration_complete=false`를
유지한다. 그 목록은 기존 Required/qualification 계약을 줄이거나 완료 authority가 되지 않는다.
UPG-003의 실제 검증과 한계는 [evidence](./evidence/runtime-upgrade-maintenance/upg-003.json)가 소유한다.

## 5. gVisor Go 전환

현재 pinned gVisor source의 `go.mod`는1.26.3이며 `MODULE.bazel`의
`go_sdk.from_file(go_mod="//:go.mod")`가 별도 SDK를 선택한다. Host PATH 교체는
이 선택을 바꾸지 않는다. SDK에는 upstream `go_types_memoize.patch`도 적용된다.
Installed runsc의 Go1.26.3은 HAA/guest1.26.9 전환의 남은 gap이다.

1. 정확한 기존 source/patch를 격리 경로에서 복원한다.
2. 선택 Go owner에서 파생한 SDK 변경을 명시적 patch로 적용한다. 공식 SDK identity,
   SDK patch application과 Bazel lock의 변화·재현성을 먼저 확인한다.
3. 같은 SDK/source 준비로 전체 six-member bundle과 observer를 빌드한다.
4. 실제 산출물의 compiler metadata, member size/hash, trace capabilities와102 latch
   groups를 확인한다. 새 hash는 실제 build output에서 얻고 기존 값으로 위장하지 않는다.
5. 새 patch/module/bundle identities를 후보 runtime lock에 연결하고 affected
   consumers를 검증한 뒤 coherent bundle로 전환한다.

같은 Go 전환 단계에서 위 archived quality binary의 source build도 검증한다.
gVisor와 scanner build의 원인·결과는 별도 commit/evidence로 구분하며,
scanner version 변경이나 탐지 규칙 완화로 compiler 전환을 대신하지 않는다.

기존 release builder는 현재 locked output과 즉시 비교하므로 새 compiler 산출물을
기존 hash로 승인할 수 없다. 좁은 candidate-capture 단계에서 실제 산출물을 모으고,
후보 lock을 고정한 뒤 기존 strict verifier로 재검증한다. Release/CI 경로의
hash 검사를 완화하지 않는다. 비호환이 발생하면 SDK patch/runtime-internal owner를
분석한다. Go1.26.9 비호환은 현재 확인되지 않았으며 최신 gVisor로의 무조건 변경도 하지 않는다.

### 구현된 SDK 준비와 observed-output 계획 연결

`scripts/prepare-gvisor-build-source.sh`가 exact commit/HAA patch를 복원하고,
`go_image.go_version`에서 upstream `go.mod`의 SDK 선택을 파생한다. Upstream 최소
Go보다 낮추지 않고 독립 toolchain selector나 변경된 SDK patch를 거부한다.
SDK patch 자체는 exact upstream bytes를 유지한다. Bundle과 observer의 기존
local/release/CI entrypoint가 이 준비와 locked builder image/Bazel을 공유한다.
Bundle의5개 Go binary는 실제 compiler metadata를 검사한다. Prewarmer는 shell이고
observer는 C++이므로 이 둘에 Go metadata를 요구하지 않는다. Observer의 C++
compiler는 locked builder image가 소유하며102 latches를 실행한다.

향후 Go 변경은 다음 두 계획으로 진행한다. 첫 계획은 새 SDK와 설정을 준비하고,
두 번째 계획은 실제 build outputs까지 포함한다. 기존 계획을 사후 수정하지 않는다.

1. 깨끗한 baseline에서 요청한 Go로 `plan`/`apply`하여 첫 격리 candidate를 만든다.
   Frozen official archive로 target SDK를 준비하고 candidate의 기존 builder를 실행한다.
2. 첫 candidate에서 `sh scripts/build-gvisor-runtime-bundle.sh --capture ABS_OUTPUT`을
   실행한다. 여섯 실제 산출물·compiler·source/SDK provenance를 확인한 외부 candidate
   lock과 `.receipt.json`을 만들고, 해당 lock으로 기존 strict verifier도 실행한다.
3. 같은 원래 baseline에 새 state를 사용해 동일 version/review와
   `plan ... --gvisor-capture ABS_OUTPUT/candidate-runtime-lock.json`을 요청한다.
   Receipt의 실제 source/bundle을 다시 읽어 capture를 재계산하고, source/patch/
   builder/architecture 변경 없이 member size/hash·Bazel module lock만 수용한다.
   기존 generator가 product identity를 생성한다. 이어 이 최종 계획을 `apply`/`verify`한다.
4. 최종 후보의 기존 strict build, installed integration, CPU/지원 CUDA와 Required를
   검증한 후 기존 설치 계약으로 전환한다. Configuration PASS만으로 전환하지 않는다.

`--capture` 성공 뒤에도 재검증에 필요한 build 작업 경로를 보존하고 그 위치를
출력한다. Receipt가 참조하는 source/build/module 입력을 최종 계획 검증 전에
삭제하거나 옮기지 않는다. 재부팅을 넘겨야 하면 영구 `TMPDIR`와 영구 output/state를
명시한다. Default strict build의 성공 cleanup은 유지하며 실패 입력은 두 모드 모두
보존한다. Capture 결과는 배포 승인이나 설치 authority가 아니다.

Capture는 HEAD 값만으로 실제 source 일치를 주장하지 않는다. 임시 Git repository와
별도 index에서 exact upstream+HAA patch를 재구성하고, 새 patch 파일을 포함한 실제
source와 비교한다. `go.mod`는 해당 결과에 선택 Go directive만 바꾼 bytes여야 한다.
원래 source/index는 수정하지 않는다. 생성된 Bazel module lock은 별도 hash 입력이며,
`tools/haa_gvisor_observer` overlay는 독립 observer build/latch의 검증 대상이다.

Frozen plan schema는2다. Schema1 계획을 자동 변환하지 않으며 깨끗한 동일 baseline에서
새 state로 다시 계획한다. 같은 version의 명시적 capture도 실제 identity가 달라지면
전체 Go 영향 qualification을 남긴다. 이미 일치한 capture는 source diff를 만들지 않는다.

## 6. 검증과 PyTorch 비용 관리

아래 표는 영향을 설명하는 최소 계획이다. 현재 Required·release acceptance가
상한을 정하며 자동 classifier가 이를 줄이지 못한다. 필요한 추가 검사는 합집합으로 더한다.

| 변경 종류 | 기본 검증 | PyTorch full 판정 |
| --- | --- | --- |
| 문서/업데이트 도구만, product/bundle 동일 | docs/security, 도구 회귀, generator drift, canonical Quick/워크플로 영향 검사 | 동일 실행 입력과 기존 계약이 허용할 때 기존 증거 참조; 새 실행 성공으로 표시하지 않음 |
| Node/npm만 | 위 공통 검사 + npm resolver/inspect/install/Promotion | Python·common executable identity가 달라지지 않았는지 근거 확인 |
| Rust/Cargo만 | 공통 검사 + Cargo add/transitive/offline build/security/rollback | 위와 같은 영향 근거 필요 |
| Terraform adapter/parser | 공통 검사 + provider integrity/signature/protocol/guard/rollback·actual init | 공통 검사/실행 경로 변경 여부에 따라 확대 |
| Python/pip/ABI 또는 PyTorch graph/source | 공통·wheel/metadata/entry-point/Promotion 회귀 + 실제 CPU | 영향을 받는 supported CUDA tuple 각각의 새 full 증거 필요 |
| HAA Go compiler·gVisor·observer·공통 sandbox/inspection/Promotion | cross-ecosystem installed integration + CPU + relevant security negatives | 현재 supported CPU/cu126/cu130/cu132의 새 qualification을 보수적인 완료 기준으로 계획 |
| 미분류 변경 | 공통/전체 영향으로 취급 | 근거 없는 skip 또는 green 재사용 금지 |

공통 Go/gVisor 전환의 full CUDA 검증은 이 작업 완료 범위에 포함한다. 독립 profile을
차례로 실행하며 새 후보의 ordinary Required와 manual qualification을 구분한다.
GPU 계산/driver/Toolkit 기능 attestation으로 확대하지 않는다.

대용량 실행 전에는 작은 검사부터 순서대로 진행한다:

```text
format/schema/drift
→ compiler/test-build/vet/pinned lint + deterministic failure regressions
→ observer/lifecycle capability + small real installed consumer
→ real CPU full
→ cu126 / cu130 / cu132 independently
→ final candidate checks + exact source/runtime/evidence binding
```

- Python ABI 변경 때는 wheel availability와 supported graph를 먼저 확인해
  대용량 download 뒤에야 불가능한 조합을 발견하지 않도록 한다.
- Disk·memory·시간·artifact graph 예산과 retained data를 preflight에서 확인한다.
  기존 budget을 올리거나 필수 coverage를 낮춰 성공시키지 않는다.
- 같은 실패 입력의 무조건 재시도 대신 최초 오류와 exact stage/category를 보존한다.
  환경·네트워크·resource·fixture·제품 원인을 분리하고 새 가설/입력이 있을 때 재실행한다.
- 동일 인증 artifact의 download cache는 허용 계약 아래 재사용할 수 있지만,
  cache hit는 새 후보의 검사·승격·full qualification 성공이 아니다.
- Profile별 source commit/tree/mode, runtime digest, package graph/hash, Python ABI,
  source/resource policy, 실제 명령/exit, coverage와 cleanup을 연결한다.
  한 profile의 PASS를 다른 profile 또는 변경된 compiler의 PASS로 전용하지 않는다.

## 7. 기록·중단·전환·정리

기존 report/evidence 구조를 사용해 baseline, proposal, apply journal, candidate,
verification와 activation/retirement 결과를 연결한다. Reviewable summary는 버전
관리하고 큰 raw log·개인 경로는 local evidence owner에 둔다. SHA 참조만으로
raw 원격 보관을 주장하지 않는다. Receipt는 new parallel execution authority가 아니다.

실제 성공 전에는 PREPARED/APPLIED/VERIFIED/ACTIVATED/RETIRED를 구분한다.
Failure/UNKNOWN/NOT_RUN은 성공이나 부재로 바꾸지 않으며 하위 단계 결과로
상위 완료를 선언하지 않는다. 각 상태는 실제 hash와 결과에서 판정한다.

Privileged replacement 전 원본 bytes/hash/UID/GID/mode, helper launch ownership과
복구 절차를 영구 위치에도 보존한다. `/tmp`·EXIT trap만으로 재부팅 복구를 보장하지 않는다.
Helper stop이 UNKNOWN이면 dependent restore/rebind/activation은 하지 않고,
안전한 read-only 원인 분석과 독립 정리를 계속한다. 실패 원인과 복구 상태를 별도로 기록한다.

Retirement는 HAA-owned 이전 SDK/cache의 exact inventory와 active/unknown process
reference 확인 후에만 수행한다. 정상 전환 직후 rollback backup 보관과 active
old SDK 선택은 구분한다. 원본 증거·실패 입력·일반 dependency module·다른 환경은 보존한다.

## 8. 자동화 한계와 완료 의미

자동화는 반복 pin 수정·생성·검사·누락 탐지를 담당한다. 외부 API/ABI 변경,
지원 wheel 부재, SDK patch conflict, license/security/support 결정까지 자동으로
수리했다고 주장하지 않는다. 이런 경우 좁은 owner와 실제 오류를 제시한다.

업데이트 기능의 완료는 현재 버전 no-op, 조정된 fixture upgrade, 부분 갱신/잘못된
digest/중단·dirty 입력 거부, 실제 Go/gVisor1.26.9 후보의 검증·전환과 모든 지원
생태계의 영향 판정이 연결된 상태다. M13 보류를 해제하는 기준은 계획 문서에 둔다.
