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
| HAA Go compiler | `scripts/runtimes.lock.json`의 `go_image.go_version`을 선택 버전 owner로 사용 | `go.mod` floor, README Build minimum/toolchain, tool `setupGo`, CI/release setup, guest tag/version/digest와 host SDK를 연결 |
| Node/npm | runtime lock의 Node image와 bundled npm identity | Node tag/version과 실제 image contents 일치; resolver·probe·Promotion 공유 |
| uv | runtime lock schema2의 uv_image/version/provenance/native tool identity | 공식 signed proof·전체 layer를 고정하고 offline plan/apply에서 재검증; Python ABI는 독립 유지 |
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

README의 `Build` section에 있는 Go minimum/toolchain 문장도 선언된 소비자다.
Go plan은 유일한 section/현재 두 값의 알려진 형식을 확인한 뒤 owner에서 파생해 갱신한다.
Read-only consistency 검사는 README·module·guest·tools의 일치를 요구한다. 누락·중복·혼용·
알 수 없는 문장은 거부하며 다른 section의 과거 버전과 qualification 원본은 수정하지 않는다.

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
실제 checks는 각 attempt의 standalone Git projection에서 실행한다. 선택한 Go의
VCS 탐색이 linked worktree의 .git file을 지나쳐 상위 저장소를 선택하는 것을 막는다.
Exact baseline HEAD의 local shared clone에 frozen after bytes와 전체 modes를 적용하고
checkCandidate로 원래 apply worktree와 checked source를 검사 전후 대조한다. 실제 경로는
receipt의 checked_source에 기록한다. VCS stamping이나 canonical checks를 끄지 않으며
원래 worktree·journal과 사용자 source는 그대로 유지한다.
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
- RAM은 시작 시 free 값만 읽지 않고 selected resource policy의 shared tmpfs와
  runtime memory, controller/OS 여유를 함께 계산한다. 이번 local execution driver의
  capacity preflight와 실제 peak/PSI/swap를 기록한다. 기존 product/CI resource
  한도·지원 tuple은 유지한다. WSL 전역 VM 용량 설정과 전체 재시작은 영향 범위를
  제시하고 해당 권한이 확인된 경우에만 적용한다.
- 같은 실패 입력의 무조건 재시도 대신 최초 오류와 exact stage/category를 보존한다.
  환경·네트워크·resource·fixture·제품 원인을 분리하고 새 가설/입력이 있을 때 재실행한다.
- 동일 인증 artifact의 download cache는 허용 계약 아래 재사용할 수 있지만,
  cache hit는 새 후보의 검사·승격·full qualification 성공이 아니다.
- Profile별 source commit/tree/mode, runtime digest, package graph/hash, Python ABI,
  source/resource policy, 실제 명령/exit, coverage와 cleanup을 연결한다.
  한 profile의 PASS를 다른 profile 또는 변경된 compiler의 PASS로 전용하지 않는다.

### Live dependency와 명시 qualification 입력 준비

2026-10-16 실제 cu126은 root가 허용한 범위에서 새 cuda-bindings12.9.10을 선택했다.
고정된 보조 입력의 target hash는12.9.9였으므로 기존 검증이 올바르게 거부했다.
현재 버전 문자열만 같은 root의 live graph를 과거 graph와 동일하게 취급하지 않는다.
Historical corpus/support receipt는 갱신 대상이 아니며 SDK 회귀로 분류하지 않는다.

보완 범위는 기존 bootstrap composition과 trusted integration fixture다. Production
CLI의 `--inspection-prerequisites`는 계속 exact caller pin이고 default는 empty다.
기존 resolver/loader·typed helper·namespace·원장·Policy·승격을 재사용한다.

- Test-only qualification 준비를 명시적으로 선택한 CUDA profile에서만 수행한다.
  알려진 plan ID는 cuda-bindings의 broader non-root probe와 고정 NumPy2.4.6의
  source/version/filename/SHA를 선언한다. Aux pin은 이 test-only plan 한 곳이 소유하며
  자동 최신화·artifact error/출력 기반 augmentation은 하지 않는다. CPU는 empty다.
- 기존 trusted resolver를 사용해 공식 source의 locked metadata graph를 먼저 얻는다.
  요청한 Torch root·profile/runtime/target과 선택된 public-PyPI non-root cuda-bindings가
  유일한지 확인한다. Version/URL/declared integrity와 graph/runtime fingerprint를
  qualification receipt에 고정한다. Expected target hash를 payload bytes에서 만들지 않는다.
- 이 선언 metadata에서 exact targetSHA를 준비해 기존 CLI pin으로 전달한다. NumPy가
  이미 원래 graph에 있거나 target이 없거나/primary/다른 source/모호하면 review가
  필요한 오류로 거부한다. 보조 wheel·scripts는 계속 target에 승격하지 않는다.
- Production composition은 기존 parser/target validation을 resolver 직후에도 사용해
  stale/root/replacement pin을 main artifact intake 전에 거부한다. Loader의 기존
  검증도 유지한다. Metadata selection 사이 새 release가 나오면 정확한 pin 불일치를
  fail-closed로 알리며 자동 fallback/retry 또는 다른 target으로 바꾸지 않는다.
- 준비 helper는 test binary에만 존재한다. Product command/interface·새 privileged
  executor·CI job은 추가하지 않는다. CI는 세 profile에 동일한 명시 plan ID를 전달하고
  기존 NOT_ATTESTED viewer·NumPy exclusion·15m/40m·Required 계약을 유지한다.

검증: empty/default·unknown plan·CPU augmentation·wrong source/root/missing/duplicate
binding을 거부하는 작은 회귀, stale target에서 main intake0, 실제 작은 non-root
cuda-bindings+NumPy 준비/검사, workflow 실제 env 전달, 최종 같은 후보 CPU/세 CUDA와
Required. 기존 실패를 보존하며 새 metadata 입력/후보의 결과를 분리한다.

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

## 9. 현재 설치 전환과 운영 인계

UPG-006은 검증된 후보를 현재 controlled Linux 개발 설치에 적용하는 작업이다.
일반 사용자의 서명된 공개 release 설치와 배포는 M10/M13 계약을 그대로 따른다.
로컬 build나 임의 verifier로 release attestation을 만들지 않는다. 실제 전환의
hash·owner·mode·compiler·consumer·doctor와 정리 범위는
[UPG-006 evidence](./evidence/runtime-upgrade-maintenance/upg-006.json)가 소유한다.

### 전환 순서와 실패 처리

1. Queue의 선행 qualification, exact source/runtime/tool identities와 현재 설치를
   확인한다. 원래 사용자 checkout은 유지하고 updater에는 clean 별도 Git source를
   사용한다. SDK는 현재 선택된 absolute Go를 지정하며 자동 SDK download를 켜지 않는다.
2. 기존 설치 파일의 bytes/hash/UID/GID/full mode를 영구 backup에 보존한다.
   root의 설치 journal·민감 config를 일반 사용자가 읽지 못한 경우 필요한 root
   readback만 수행하며 권한을 낮추지 않는다. Git executable bit와 native full mode는
   다르므로 재부팅 후 임시 source 복원도 frozen inventory의 bytes/full modes를 따른다.
3. canonical bootstrap으로 기본 managed quality cache를 준비하고 actual compiler를
   확인한다. 기존 qualified output과 비교하며 version 문자열만으로 활성화를 판정하지
   않는다. caller active binaries와 updater의 독립 verification cache는 구분한다.
4. helper/container/process/handle 상태와 원본을 설치 직전에 확인한다. 기존 controlled
   installation과 canonical owning-shell helper lifecycle을 재사용해 coherent bundle을
   적용한다. 여러 파일의 순차 쓰기를 하나의 atomic transaction이라고 주장하지 않는다.
   완료 전에 끊기면 journal의 완료 목록과 실제 before/after hash를 먼저 대조한다.
5. 실제 ordinary-user production consumer와 installed doctor를 실행하고 helper의
   parent wait·cleanup 및 마지막 protected file identities를 확인한다. 설치 성공과
   테스트 준비 실패, doctor의 일부 readiness와 전체 healthy 상태를 각각 기록한다.
   원래 실패를 후속 cleanup이나 성공으로 덮지 않는다.
6. confirmed stop과 identity 검증 뒤에만 원본 복원 또는 성공한 후보 유지를 수행한다.
   UNKNOWN 아래 dependent restore/rebind/retirement를 하지 않는다. 재부팅으로 임시
   경로가 없어졌다면 영구 backup·기존 Git source·frozen modes를 복구해 검증한 뒤
   기존 절차로 이어간다. Protected journal 자체가 실행 권한이나 완료 authority는 아니다.

Controlled local activation은 보호된 파일의 전환이다. 이 작업의 helper는 실제
검증 동안 기존 소유 lifecycle로 실행하고 parent wait를 확인한 뒤 종료한다.
상시 service 활성화나 일반 사용자 공개 bootstrap 제공을 이 결과로 주장하지 않는다.

현재 `helox doctor`의 release-installation은 공개 attested release가 없으면
UNAVAILABLE/exit1이다. 나머지 Host readiness가 OK여도 전체 doctor를 healthy로
바꾸지 않는다. 이 상태와 M13 미출시 범위를 보존하며 로컬 runtime activation을
공개 installer 제공 또는 production release 완료로 확대하지 않는다.

### 생태계별 갱신 책임

| 대상 | 기존 entrypoint 또는 적용 제외 이유 | 영향 검증 |
| --- | --- | --- |
| HAA Go·Go guest·gVisor·quality tools | `update-runtime plan --component go --version X.Y.Z`, apply/verify와 실제 bundle capture/build 및 승인된 설치 | compiler/drift, installed cross-ecosystem, CPU/지원 CUDA, actual Required/native platform |
| Node/npm | `--component node`; npm은 선택한 official Node의 bundled identity | named image/manifest, npm resolver·inspect·install·Promotion |
| Python/pip | `--component python`; pip/ABI는 선택한 interpreter와 함께 검토 | wheel/entry-point/Promotion 및 영향받은 CPU/지원 CUDA full |
| Rust/Cargo | `--component rust` | source/add/transitive/offline build와 security·rollback |
| Terraform Provider | `--component terraform`은 adapter/common-runtime 영향만 계획; 사용자 constraints/lock·서명된 provider는 재작성하지 않음 | actual registry integrity/signature/protocol·guard·init |
| GitHub Releases | `--component github`은 common-runtime 영향만 계획; 사용자 release/asset의 version·hash·signature는 독립 입력 | signed asset resolve/acquire/inspect·negative consumers |
| PyTorch CPU/cu126/cu130/cu132 | SDK updater의 자동 package-version 선택 대상이 아님; 기존 reviewed support tuple/named source와 generator가 owner | common Go/gVisor 변경에는 네 독립 full qualification; GPU/driver/Toolkit attestation으로 확대하지 않음 |

다음 실제 언어 버전은 official metadata·review·새 frozen plan과 필요한 actual
qualification을 다시 요구한다. 기존 coordinated next-version fixtures와 interruption/
dirty/digest/state-lock tests는 편집·복구 계약의 회귀이며 실제 미래 SDK나 지원 tuple의
검증 결과가 아니다. Same-version reconciliation은 새 SDK나 source diff를 만들지
않아야 하고 configuration PASS를 migration/activation authority로 사용하지 않는다.

### 보존과 정리

활성 SDK·기본 quality binaries·설치 runtime, 영구 rollback originals, 실제 원본
Evidence/log/manifest/receipt와 현재 복원에 필요한 후보는 보존한다. 이전 compiler로
빌드된 비활성 rollback binary는 활성 SDK 선택과 구분한다. 완료된 테스트의 targets,
intake/staging payload 사본, test executable·temporary source와 regenerable build
cache는 exact inventory 및 실제 proc/FD/maps/mount/container·operation-lock 확인 후
삭제한다. 공유 dependency module과 다른 사용자 환경을 임의 정리하지 않는다.
논리적 삭제 byte, 실제 filesystem available 변화와 Windows VHD 회수는 다른 수치다.

전체 qualification 뒤에도 separate full-Race fixture finding, release-installation과
reviewer merge는 해당 owner의 남은 작업이다. 유지보수 branch를 검토 가능한 상태로
전달하고 실제 reviewer merge/전달 CI와 새 M13 baseline 확인 전에는 보류 전의
M13 CLEAN·승인 후보를 재사용하거나 M13 보류를 자동 해제하지 않는다.


### UV 공급의 offline provenance 검증

P1의 UV 후보는 채택 전에 다음 명시 단계로 검증한다. Runtime pin/실행·Host 설치를
변경하지 않고, 같은 고정 Sigstore root로 upstream publisher와 OCI 자료 연결만 확인한다.

```sh
go run ./scripts/update-runtime verify-uv-image --input /absolute/frozen-uv-proof
```

디렉터리에는 identity.json(정확한 version/source_commit/workflow_run/index_sha256/
manifest_sha256), bundle.json, index.json, manifest.json, config.json이 필요하다.
Bundle은 GitHub의 해당 index digest attestation API 응답에서 bundle 객체만 보존한다.
원본 OCI 세 파일은 reformat 없이 digest/size를 확인한 bytes다. 출력은 검증된 source·
index/amd64·config·layer metadata의 영수증이며 layer bytes/실행 호환성/activation을
승인하지 않는다. CLI 자체가 네트워크를 호출하거나 runtime을 다운로드하지 않는다.

정책과 실제 서명 입력은 [P1b baseline](../planning/evidence/package-client-quarantine/p1b-uv-runtime-baseline-20261021.md)을
따른다. HAA release signer 정책을 UV publisher로 확대하지 않는다. 현재 plan/apply의
Metadata 검증 이후 단일 UV pin·frozen maintenance와 [data-only 공급 전달](../planning/evidence/package-client-quarantine/p1b4-uv-provision-local-20261021.json)까지 로컬 검증했다. Running runtime/admission과 production uv 명령은 미완료다.


`verify-uv-image`에 `--layers /absolute/frozen-layers`를 추가하면 manifest 순서의 모든
`<sha256>.tar.gz`를 확인해 UV native file까지 연결한다. Compressed digest/size·config
 diffID·root ownership·정확한 reviewed layout와 bounded static amd64 ELF를 검증한다.
출력은 `helox.uv-image-supply/v2`의 tool SHA256/size이며 실제 native startup·관찰기·Policy
승인이 아니다. `--layers` 없는 기존 metadata-only 호출은 v1을 유지한다.

### UV pin과 frozen maintenance plan

Runtime lock schema3의 uv_image가 version·official amd64 reference·signed index/source/run·
bundle/config digest와 native path/size/hash의 유일한 입력이다. Generator와 support drift는
이 owner를 사용한다. Runtime image manifest v2는 기존 네 이미지와 UV의 exact5 images 및 별도 pnpm tool record를 기록하며
custom_ghcr_image=false는 HAA 소유 이미지를 새로 게시하지 않는다는 뜻이다.

변경 계획은 다음과 같이 명시한 offline 공급만 사용한다. proof 디렉터리는 위 다섯 metadata
파일, layers 디렉터리는 검증된 모든 SHA256.tar.gz를 갖춰야 한다.

```sh
go run ./scripts/update-runtime plan --root /absolute/clean-repo --state /absolute/new-state \
  --component uv --version X.Y.Z --review /absolute/review.json \
  --uv-proof /absolute/frozen-proof --uv-layers /absolute/frozen-layers
go run ./scripts/update-runtime apply --root /absolute/clean-repo --state /absolute/new-state
go run ./scripts/update-runtime verify --root /absolute/clean-repo --state /absolute/new-state
```

Plan은 작은 proof bytes와 전체 layer descriptor를 묶고 실제 layer는 state/uv-supply에
owner-only로 고정한다. Render/apply는 같은 파일의 owner/mode·경로와 모든 digest/diffID·
native identity를 다시 확인한다. 누락/변조/symlink/권한 변경은 거부하며 source를 재작성하거나
live tag로 대체하지 않는다. 같은 버전은 supply flag를 거부하고 기존 pin/review를 유지한다.
Canonical configuration 검증과 실제 UV sandbox/observer/CLI qualification은 별도다.

### pipx publish 공급의 offline 검증

P4-O1의 developer command는 명시한 frozen directory의 identity.json(version,
source_commit, workflow_run, wheel_sha256), provenance.json 및
`pipx-<version>-py3-none-any.whl`을 읽는다.

```sh
go run ./scripts/update-runtime verify-pipx-publish --input /absolute/frozen-pipx-proof
```

단일 wheel의 실제 SHA-256과 bounded PEP740 publish/v1 proof, 고정 pypa/pipx
release.yml publisher의 certificate/tag/source/run·environment·token subject 및
기존 offline Sigstore root/SCT/log/timestamp를 검증한다. 성공 receipt의 범위는
`SINGLE_WHEEL_PUBLISH_PROOF_ONLY`다. 전체 dependency closure·immutable loader와
runtime admission은 [P4-O baseline](../planning/evidence/package-client-quarantine/p4o-pipx-supply-runtime-baseline-20261027.md)의
후속 acceptance이며 이 명령은 네트워크·추출·Python 실행·pin 변경·설치를 수행하지 않는다.
기존 HAA/UV/PNPM SLSA publisher 정책은 유지한다.

P4-O2의 전체 공급 검증은 같은 developer command에서 별도 scope로 수행한다.

```sh
go run ./scripts/update-runtime verify-pipx-closure --input /absolute/frozen-pipx-closure
```

입력에는 argcomplete/click/filelock/packaging/pipx/platformdirs/userpath의 일곱 하위
디렉터리가 필요하다. 각각 identity.json, 공식 exact-release metadata.json과 exact wheel을
두고 제공된 provenance.json도 함께 둔다. Pipx 본체 identity는 위 형식이며 서명된 다섯
dependency identity에는 project도 포함한다. Userpath identity는 version/wheel_sha256만
받고 signature `NOT_PROVIDED`를 유지한다. Userpath에 제공된 proof는 미지원 정책으로
거부하며 무시하지 않는다. 나머지 여섯 proof는 고정 publisher로 모두 검증한다.

기존 whole wheel ZIP/RECORD와 독립 metadata, 현재 고정 Python의 dependency/marker/
constraint 검사를 재사용하고 complete reachable closure·namespace/파일 충돌을 확인한다.
Receipt 범위는 `STATIC_WHEEL_CLOSURE_AND_PUBLISH_PROOFS_ONLY`다. 공급 bytes와 static
closure의 사실이며 immutable loader/runtime admission·native 동작·설치/지원 qualification은
후속 단계다. 입력의 후보 버전은 별도 product version lock을 만들거나 pin을 자동 채택하지 않는다.

### pipx readonly 후보의 명시 출력

```sh
go run ./scripts/update-runtime materialize-pipx-candidate --root /absolute/repo --input /absolute/frozen-pipx-closure --output /absolute/external/pipx-runtime.zip
```

위 공급 검증과 같은 held wheel bytes를 사용해 모든 원본 member·RECORD·license를
보존하는 결정적 ZIP을 생성한다. Whole archive와 member hash를 재대조하고 현재 검토한
root module/dist-info 배치만 허용한다. Relocation·충돌·8MiB 초과는 거부한다.
출력은 source tree 밖의 새 파일이며 기존 no-replace publisher로0444를 기록한다.
이 명령은 Python을 실행하거나 설치·runtime pin을 변경하지 않는다. Isolated loader,
gVisor/observer와 실제 pipx install/upgrade 지원은 별도 qualification이다.

### pipx selected owner와 frozen 유지보수

Runtime lock schema5의 `pipx_tool`이 선택 버전·Python owner 참조·공급 입력27개의
path/hash와 ZIP/NOTICE identity를 소유한다. `tools/pipx/supply`의20개 문서는 동결된
입력 DATA이며 wheel7개는 외부 취득 입력이다. Source identity/proof를 바꾸면 입력 hash와
실제 ZIP을 다시 검증한다. Version 선택을 별도 lock으로 복제하지 않는다.

```sh
go run ./scripts/update-runtime materialize-pipx --root /absolute/repo --input /absolute/frozen-pipx-closure --output /absolute/external/pipx-runtime-linux-amd64.zip --notices-output /absolute/external/pipx-NOTICE.txt
```

이 selected-owner 명령은 전체 공급 path/hash, canonical metadata와 원본 license/NOTICE,
생성 ZIP/notices identity를 대조하고 두 새 파일을0444로 보존한다. 설치·활성화나 검사 완료를
부여하지 않는다. 아직 release workflow/self-update의 executable ZIP 검사 경로에 연결되지
않았으며 기존 native asset 집합에 ZIP을 passive metadata로 끼워 넣지 않는다.

`plan --component pipx --version X.Y.Z --pipx-proof /absolute/frozen-supply --review JSON`은
다른 버전의 입력을 기존 external state 아래 동결하고 apply/verify 전에 같은 proof/bytes를
재검증한다. Same-version은 acquisition 없이 기존 supply/review를 보존하며 새 proof를
거부한다. Source 변경은 기존 candidate/journal/lock 경로를 사용한다. 실제 upstream release
검증과 synthetic 이전 owner의 plan/apply 회귀는 구분하며, gVisor/native operations/public CLI와
최종 profile qualification은 별도다.

### pnpm release 공급의 offline 검증

현재 pnpm 유지보수 consumer는 다음 명령으로 **획득한 Linux amd64 archive**의 SHA-256과
공식 build provenance를 함께 확인한다. 입력 디렉터리에는 identity.json(version, source_commit,
workflow_run, archive_sha256), bundle.json 및 pnpm-linux-x64.tar.gz가 필요하다.

```sh
go run ./scripts/update-runtime verify-pnpm-release --input /absolute/frozen-pnpm-proof
```

고정된 pnpm repository/workflow/tag·Fulcio identity·SLSA source/run/subject와 기존 offline
Sigstore root를 검증한다. 실제 upstream의 self-hosted build 조건은 pnpm 전용 정책에만
적용하며 HAA/UV publisher 정책을 변경하지 않는다. 명령은 네트워크·archive 추출·Host 실행·
runtime pin 변경·설치를 수행하지 않는다. 성공은 공급 bytes의 출처 확인이며 보안 검사·실제
runtime 지원 승인이 아니다. Runtime/lock/layout·설치/업데이트의 남은 acceptance와 수정 순서는
[P2 착수 지도](../planning/evidence/package-client-quarantine/p2-pnpm-baseline-20261024.md)를 따른다.

현재 pnpm 공급 명령의 v2 receipt는 인증한 전체 archive의 bounded tar 구조와 pnpm regular
ELF의 hash/size·declared shared libraries도 연결한다. Bundled dist 자료와 hardlink는 검증 후
버리며 Host에 추출하지 않는다. Shared-library 목록과 runc 형식 관측은 실제 gVisor admission·
제품 검역/설치 qualification이 아니다. [P2b1 결과](../planning/evidence/package-client-quarantine/p2b1-pnpm-native-local-20261024.json)가 정확한 입력·검증·정리 범위를 소유한다.

### pnpm pin과 frozen maintenance plan

Runtime lock schema3의 `pnpm_tool`이 공식 archive/native identity와 production proof 경로/hash의 단일 owner다.
`tools/pnpm/release-proof.json`은 배포 입력이며 testdata를 production 공급으로 사용하지 않는다.
현재 v2 runtime metadata는 exact5 images와 별도 PNPM native tool record를 확인한다.

```sh
go run ./scripts/update-runtime plan --root /absolute/clean-repo --state /absolute/new-state \
  --component pnpm --version X.Y.Z --review /absolute/review.json \
  --pnpm-proof /absolute/frozen-pnpm-proof
go run ./scripts/update-runtime apply --root /absolute/clean-repo --state /absolute/new-state
go run ./scripts/update-runtime verify --root /absolute/clean-repo --state /absolute/new-state
```

조회 전용 offline 검증은 위 세 파일을 사용한다. 새 PNPM plan의 proof 디렉터리는 추가로 고정 source의 LICENSE/제3자 고지를 합친 `NOTICE`를 명시한다. Plan은 작은 인증 metadata만 담고
archive는 state/pnpm-supply에 owner-only로 고정한다. Render/apply는 전체 archive의 서명·hash·
native identity와 state owner/mode를 재검증한다. Same-version은 network0이며 새 supply flag를
거부하고 기존 pin을 보존한다. 중단 후 재개·idempotence·변조 거부의
[로컬 검증](../planning/evidence/package-client-quarantine/p2b2-pnpm-owner-local-20261024.json)은
native 배포/설치·gVisor admission·제품 CLI 또는 새 원격 qualification 완료가 아니다.

### pnpm native asset의 명시 출력

```sh
go run ./scripts/update-runtime materialize-pnpm --root /absolute/source \
  --archive /absolute/acquired/pnpm-linux-x64.tar.gz \
  --output /absolute/external-dist/pnpm-linux-amd64 \
  --notices-output /absolute/external-dist/pnpm-NOTICE.txt
```

유지보수 도구는 선택한 source의 schema3 PNPM owner와 production proof를 읽는다.
전체 archive·내부 파일·ELF 검증이 끝난 pnpm regular bytes만 반환하고, 외부의 canonical
자기 소유 보호 디렉터리에 새 파일/0555로 출력한다. Readback/hash·no-replace 게시·directory
sync와 임시 이름 identity를 확인하며 기존 파일·symlink·foreign writable parent를 거부한다.
Host에서 pnpm을 실행하거나 bundled dist 자료를 풀지 않는다. 성공은 native asset bytes의
공급이며 릴리스 배포/NOTICE 연결·gVisor admission·제품 검역 지원 완료가 아니다.

현재 PNPM owner의 `notices`는 production `tools/pnpm/NOTICE`의 경로/hash/크기를 기록한다.
Frozen maintenance는 source-bound 고지 bytes도 계획에 고정하고 source가 바뀔 때 이전 고지의
재사용을 거부한다. 이는 reviewed passive source data이며 upstream archive 서명의 일부라고
주장하지 않는다. Native asset은0555, 고지는0444로 출력하며 두 파일 모두 signed release set과
OS package의 기존 설치/복구 범위에 포함한다. 전체 배포 qualification은 별도다.
