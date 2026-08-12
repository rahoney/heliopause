# Step 10 — Formatter / Linter / Type Check / Test / Security Scan

이 문서는 Step 8의 Go module·package 구조와 Step 9의 Coding / Security Rules를 반복 가능한 로컬 검사로 집행할 도구, 명령과 실패 기준을 정의한다. CI job 구성과 merge 차단 조합은 Step 11에서 정하며, 이 단계에서는 실제 `go.mod`, script, tool configuration 또는 scaffold를 생성하지 않는다.

## 목표

품질 도구는 다음 질문에 각각 답한다.

```text
Formatting
→ 같은 Go source가 같은 형식인가?

Compile / Type Check
→ product와 test code가 선택한 Go version에서 유효한가?

Static Analysis
→ compiler가 허용하지만 오류 가능성이 높은 code가 있는가?

Architecture Check
→ Step 8의 import direction과 package 금지 규칙을 지키는가?

Test
→ Domain·workflow·Port 구현과 사용자 operation이 기대대로 동작하는가?

Security Scan
→ 알려진 취약점, 위험한 source pattern 또는 secret 노출이 있는가?
```

한 도구의 성공을 다른 검사의 대체물로 사용하지 않는다.

## 1. 채택 도구

| Area | Tool | Role |
| --- | --- | --- |
| Go version | Go toolchain | compile, type check, test와 표준 분석의 기준 |
| Formatting | `gofmt` | Go source canonical formatting |
| Module consistency | `go mod tidy -diff`, `go mod verify` | module graph drift와 module cache 변조 확인 |
| Compile / type check | `go build`, `go test` build-validity check | production·test package type validity 확인 |
| Standard analysis | `go vet` | Go standard analyzer 진단 |
| Linter | Staticcheck | 정확성·표준 library 오용·유지보수 문제 탐지 |
| Architecture | repository-owned Go checker | Step 8 import direction과 금지 dependency 검증 |
| Test | `go test` | unit, contract, integration, E2E, race, fuzz |
| Source security | gosec | Go AST/SSA 기반 보안 pattern 검사 |
| Vulnerability | govulncheck | Go vulnerability database와 호출 가능 code 분석 |
| Secret | Gitleaks | source tree와 Git 변경 이력의 secret 후보 탐지 |
| Documentation | repository-owned Go checker | local Markdown link와 code fence 무결성 검증 |

외부 도구는 각각 다른 공백을 채울 때만 채택한다.

- Staticcheck는 `go vet`보다 넓은 correctness 분석을 제공한다.
- gosec는 command, filesystem, crypto, TLS와 taint 등 security-specific pattern을 검사한다.
- govulncheck는 source pattern이 아니라 현재 module과 Go 표준 library의 알려진 취약점을 확인한다.
- Gitleaks는 Go analyzer가 다루지 않는 credential·secret 후보를 검사한다.

## 2. 의도적으로 채택하지 않는 도구

### `goimports`

초기 formatter는 standard library의 `gofmt`만 사용한다. import 추가·제거 자동화는 editor 편의 기능으로 사용할 수 있지만 canonical formatter나 Quality Gate로 요구하지 않는다. 불필요한 formatter dependency와 editor별 차이를 만들지 않기 위함이다.

### `golangci-lint`

초기에는 사용하지 않는다. 여러 analyzer를 한 binary로 묶는 편의보다 다음을 우선한다.

- 실제 사용하는 analyzer와 version을 명확히 식별
- 중복 진단과 광범위한 설정 억제
- transitive tool dependency와 공급망 면적 축소
- 도구 failure와 개별 finding의 원인 추적

`go vet + Staticcheck + gosec`로 확인되지 않는 구체적인 결함 유형이 반복될 때만 추가 analyzer 또는 aggregator 도입을 재검토한다.

### 범용 task framework

Make, Task, Mage 또는 language-independent hook framework를 필수 dependency로 두지 않는다. canonical 검사는 repository-owned Go check runner에서 실행하고 개별 도구 명령도 문서화한다. editor나 개인 pre-commit hook은 선택 사항이며 CI와 canonical 검사를 대체하지 않는다.

### 원격 AI 분석과 자동 수정

deterministic Quality Gate에서 AI 기반 scanner mode, 원격 code upload와 자동 fix를 사용하지 않는다. 특히 gosec의 AI recommendation 기능을 활성화하지 않는다. 자동 수정은 `gofmt`처럼 결과가 안정적인 formatter에만 허용한다.

## 3. Go와 tool version 고정

- Step 14 구현 시작 시 선택한 안정 Go patch version을 `go.mod`의 `go`/`toolchain` 정책과 CI setup에 명시한다.
- `go` directive는 최소 지원 language/toolchain 기준을, 명시적 toolchain version은 기본 개발·검증 patch version을 나타낸다.
- `GOTOOLCHAIN`의 자동 `latest` 추종에 의존하지 않는다.
- 최소 지원 version과 기본 pinned version의 실제 호환성 검증은 Step 11에서 별도 CI job으로 구성한다.
- Go 기반 외부 quality tool은 product `go.mod`에 `tool` directive로 추가하지 않는다. Tool dependency는 product와 같은 module graph를 사용해 product dependency 선택과 최소 Go version에 영향을 줄 수 있기 때문이다.
- repository-owned `scripts/tools.lock.json`에 tool package path, exact version, expected command name과 설치에 필요한 Go version을 기록한다.
- setup runner는 `go install <package>@<exact-version>`을 사용해 source tree 밖의 project-specific tool cache에 설치한다. `package@version` 방식은 product module graph를 사용하거나 변경하지 않는다.
- module proxy와 checksum database의 정상 검증을 유지하고, 설치 뒤 실제 tool version을 lock entry와 대조한다.
- canonical check runner는 `PATH`에서 임의 tool을 찾지 않고 해당 cache의 명시적 executable path만 실행한다.
- quality tool과 그 설치 graph도 Step 9의 dependency·license·취약점·유지보수 검토 대상이다.
- tool version을 `@latest`, floating branch 또는 unpinned download로 실행하지 않는다.
- tool upgrade는 release note, Go version compatibility, rule 변화, 새 false positive와 dependency graph를 검토한 명시적 변경으로 수행한다.
- Gitleaks는 현재 feature-complete·security-fix maintenance 상태이므로 보안 수정 제공 여부를 upgrade 검토 시 확인하며, 유지보수가 중단되면 대체 도구를 재평가한다.

정확한 Go와 tool version 숫자는 구현 시작 시점에 고정한다. Step 10은 제품명과 실행 계약을 고정하며 미래의 임의 최신 version을 미리 적지 않는다.

## 4. Canonical local entrypoint

구현 단계에서 standard library만 사용하는 repository-owned check runner를 `scripts/check/`에 둔다.

```text
go run ./scripts/check <profile>
```

이 runner는 Heliopause product workflow를 포함하지 않는 개발 도구이며 다음 규칙을 따른다.

- module root에서만 실행한다.
- executable과 argument를 분리하여 호출하고 shell command 문자열을 만들지 않는다.
- 필요한 tool과 version이 lock manifest와 project-specific tool cache에서 일치하는지 확인한다.
- check profile은 repository source, module file 또는 config를 수정하지 않는다.
- 각 단계에 timeout을 적용하고 첫 실패의 명령·분류·종료 상태를 secret 없이 표시한다.
- scanner stdout/stderr를 무제한 buffering하지 않는다.
- 외부 tool의 `no-fail` option으로 Quality Gate를 우회하지 않는다.
- 여러 검사를 병렬 실행해 원인과 resource 사용이 불명확해지지 않도록 초기에는 순차 실행한다.

format 변경은 명시적인 별도 profile이다.

```text
go run ./scripts/check format
```

`format`만 repository source를 변경할 수 있다. `bootstrap`은 source tree 밖 cache만 변경할 수 있으며, 나머지 check profile은 repository worktree에 대해 read-only여야 한다.

## 5. Check profile

| Profile | Network | Purpose |
| --- | --- | --- |
| `bootstrap` | Yes | pinned Go module dependency와 external quality tool을 source tree 밖 cache에 준비 |
| `format` | No | tracked Go source에 `gofmt` 적용; 유일한 mutating profile |
| `quick` | No | format check, module drift, compile, architecture, vet, Staticcheck, default test |
| `security` | No | gosec와 current tree Gitleaks 검사 |
| `race` | No | 지원 platform에서 race detector test |
| `vulnerability` | Yes | 최신 Go vulnerability database로 govulncheck 수행 |
| `integration` | 환경별 | 명시적 external tool/storage/runtime contract 검증 |
| `e2e` | 환경별 | build된 `helox`의 사용자 operation 검증 |
| `docs` | No | canonical Markdown local link와 code fence 검사 |

`quick + security + docs`는 `bootstrap`이 완료된 뒤 외부 service 없이 반복 가능한 기본 local 검증 집합이다. `quick`의 formatting 단계는 항상 `gofmt -l` 기반 read-only format check이며 source를 수정하지 않는다. 이 profile들은 `GOTOOLCHAIN=local`과 offline module resolution을 사용해 toolchain이나 module을 암묵적으로 내려받지 않는다. cache miss는 자동 network fallback이 아니라 prerequisite failure다. `vulnerability`는 advisory database가 시간에 따라 변하고 network가 필요하므로 deterministic 검사와 분리한다. network failure를 취약점 없음으로 취급하지 않는다.

`race`, `integration`, `e2e`는 platform·runtime prerequisite가 있으므로 별도 profile로 유지한다. Step 11에서 어떤 event와 runner가 각 profile을 의무 실행하는지 정한다.

## 6. Formatting

### 적용

- repository가 소유한 모든 `.go` file에 `gofmt`를 적용한다.
- generated Go file도 별도 upstream format을 요구하지 않는 한 검사한다.
- `vendor/`, runtime data, acquired Artifact와 외부 fixture source에는 formatter를 적용하지 않는다.
- `format`은 tracked/명시적 Go file 목록을 사용하고 임의 symlink를 따라 repository 밖을 수정하지 않는다.

### 검사

개념적인 개별 명령은 다음과 같다.

```text
gofmt -l <repository-owned Go files>
```

출력이 하나라도 있으면 format check는 실패한다. CI 또는 check profile에서 `gofmt -w`를 실행해 변경을 숨기지 않는다.

Markdown, JSON, YAML과 shell formatter는 해당 파일이 실제로 도입되고 안정적인 canonical formatter가 필요할 때 추가한다. 형식 도구를 미리 늘리지 않는다.

## 7. Module consistency와 dependency review

기본 명령은 다음과 같다.

```text
go mod tidy -diff
go mod verify
```

- `go mod tidy -diff` 결과가 있으면 `go.mod`/`go.sum` drift로 실패한다.
- `go mod verify` failure는 downloaded module content 무결성 실패로 처리한다.
- `go.sum` 삭제, checksum database 우회 또는 자동 `replace`로 해결하지 않는다.
- dependency 추가·upgrade 시 `go list -m -json all`과 `go mod graph`로 direct/transitive 변화를 검토한다.
- automated tool은 license·maintainer·목적 적합성을 완전히 판단할 수 없으므로 dependency 변경에는 Step 9의 수동 공급망 검토를 함께 요구한다.
- 초기에는 별도 license scanner를 채택하지 않는다. 신뢰할 수 있고 유지보수되는 도구와 허용 license policy가 확정되기 전 자동 결과를 승인으로 오해하지 않기 위함이다.

## 8. Compile과 type check

Go compiler가 canonical type checker다. 별도 type-check framework를 추가하지 않는다.

```text
go build ./...
go test -run '^$' ./...
```

- `go build ./...`는 production package와 command를 compile/type-check한다.
- `go test -run '^$' ./...`는 `*_test.go`, external test package와 test-only dependency의 build/type validity를 확인하기 위한 검사로 사용한다.
- 위 명령도 test binary를 만들고 실행 단계에 진입할 수 있으므로 `TestMain`, package initialization 또는 그 밖의 실행 부작용에 의존하지 않도록 test code를 작성한다.
- build-validity check 성공을 test 성공으로 표시하지 않는다.
- supported OS별 build constraint와 platform-specific file은 Step 11 matrix에서 실제 compile한다.
- Linux-only Sandbox 구현이 macOS build를 깨뜨리지 않도록 Port와 platform file 경계를 유지한다.

## 9. Static analysis와 lint

### `go vet`

```text
go vet ./...
```

표준 analyzer finding은 기본적으로 실패다. analyzer 자체 실행 실패와 finding을 출력에서 구분하되 둘 다 Quality Check 성공으로 처리하지 않는다.

### Staticcheck

```text
staticcheck ./...
```

- 기본 권장 check set에서 시작한다.
- 초기부터 모든 style option을 강제로 켜거나 광범위한 custom config를 만들지 않는다.
- suppression은 exact check ID와 인접한 기술적 이유를 요구한다.
- package/file 전체 suppression을 허용하지 않는다.
- 실제 config가 필요해질 때 module root의 `staticcheck.conf` 한 곳에 둔다.

## 10. Architecture와 documentation check

### Architecture checker

repository-owned checker는 `go list -json` 또는 동등한 Go package metadata를 읽어 최소한 다음을 검증한다.

- `core/domain`이 다른 Heliopause product package를 import하지 않음
- `core/ports`가 concrete Adapter/Provider/Backend를 import하지 않음
- `application`이 npm/PyPI/GitHub, scanner, Sandbox backend, Evidence storage, Promotion 구현을 직접 import하지 않음
- `policy`가 Adapter/Provider/Sandbox/Promotion을 import하지 않음
- `inspection`이 concrete Sandbox backend를 직접 import하지 않음
- production package가 `internal/testutil`을 import하지 않음
- concrete implementation wiring이 `bootstrap` 밖으로 새지 않음
- package import cycle이 없음

module path는 `go list -m`에서 읽으며 repository 이름을 checker에 중복 hard-code하지 않는다. 불가피하게 허용된 `core/domain` 외부 value library가 생기면 exact module/package와 별도 검토 근거를 좁은 allowlist로 기록한다.

Architecture check는 package 이름 문자열만 보고 책임의 의미를 완전히 증명하지 못한다. 따라서 자동 import rule과 code review를 함께 사용한다.

### Documentation checker

standard library 기반 checker는 project Markdown에서 다음을 검사한다.

- local relative link 대상 존재
- canonical router와 topic index의 broken link
- fenced code block의 열림/닫힘 균형
- source tree 밖으로 향하는 의도하지 않은 absolute local path link

외부 HTTP link의 생존 여부는 network·rate limit·site 변화 때문에 deterministic default gate로 검사하지 않는다.

## 11. Test taxonomy와 명령

### Default test

```text
go test -timeout=5m ./...
```

- package-adjacent unit test와 network/runtime을 요구하지 않는 contract test를 실행한다.
- public network, 실제 credential, 실제 home/project path와 Host global state에 의존하지 않는다.
- test는 `t.TempDir()`, synthetic credential, fake server와 in-memory/fake Port를 사용한다.
- test 순서, wall clock과 process-global state에 의존하지 않는다.
- 실패한 test를 자동 rerun해 성공으로 바꾸지 않는다.

### Contract test

Port 구현 공통 suite는 `test/contract`에 두되 외부 runtime 없이 실행 가능한 구현은 default test에 포함한다. 실제 registry/tool/runtime이 필요한 concrete implementation contract는 integration profile로 분류한다.

### Integration test

```text
go test -tags=integration -timeout=20m ./test/integration/...
```

- 필요한 external tool, test registry/storage 또는 Linux runtime을 profile 시작 시 명시적으로 확인한다.
- 선택된 integration environment에서 prerequisite 부재를 `PASS`나 silent skip으로 처리하지 않는다.
- production credential과 public mutable `latest` reference를 사용하지 않는다.
- local fake service 또는 immutable test fixture를 우선한다.

### E2E test

```text
go test -tags=e2e -timeout=30m ./test/e2e/...
```

- 실제 `helox` binary entrypoint, CLI output와 exit behavior를 검증한다.
- Host install/Promotion test는 disposable target과 synthetic Artifact만 사용한다.
- Linux Dynamic Inspection E2E와 platform-neutral CLI E2E를 구분한다.

### Race test

```text
go test -race -timeout=15m ./...
```

- Go race detector가 지원되는 Linux/macOS runner에서 수행한다.
- race detector 실행을 위해 build system이 cgo/runtime를 사용하는 것은 product source에 cgo를 허용한다는 뜻이 아니다.
- 느리다는 이유만으로 `!race` exclusion을 추가하지 않는다. 실제 incompatibility가 있으면 좁은 근거와 별도 대체 test를 요구한다.

### Fuzz test

다음 경계에는 구현 시 fuzz target을 우선 고려한다.

```text
Artifact Reference parser
registry/tool response parser
path containment와 archive entry validation
digest/identity serialization
Evidence/Result decoder
CLI machine-readable input
```

committed seed corpus는 sanitized·작고 재현 가능해야 한다. fuzzing은 명시적 time budget을 가진 별도 profile/scheduled job으로 실행하며 무제한 Quality Gate로 두지 않는다. 발견된 crash input은 secret 여부를 확인한 뒤 regression corpus로 고정한다.

### Coverage

- coverage report는 가시성 자료이며 초기에는 전체 percentage 하나로 merge를 승인하거나 차단하지 않는다.
- 높은 line coverage가 security invariant 검증을 대신하지 않는다.
- Domain transition, fail-closed Policy, path/archive, command argument, redaction, digest binding과 cleanup에는 정상·거부·오류 test를 명시적으로 요구한다.
- 첫 vertical slice의 baseline이 생긴 뒤 Step 11 또는 milestone에서 package별 coverage regression 기준을 정할 수 있다.

Benchmark와 performance profile은 정확성 gate가 아니다. resource limit·large input 관련 regression이 실제 요구로 등장할 때 별도 bounded benchmark를 추가한다.

## 12. Security scan

아래 scanner 명령은 요구 동작을 나타내는 기준 예시다. 실제 canonical invocation은 Step 14에서 고정하는 pinned tool version이 해당 기능과 option을 지원하는지 확인한 뒤 확정한다. Tool version이 달라도 test code 포함, suppression rule·justification 요구, secret redaction과 `no-fail` 금지 등의 의미적 요구사항은 유지한다.

### gosec

```text
gosec -tests -nosec-require-rules -nosec-require-justification ./...
```

- production과 test helper를 함께 검사한다.
- 모든 unsuppressed finding과 scanner execution error는 실패다.
- `-no-fail`을 canonical check에 사용하지 않는다.
- `#nosec`에는 exact rule ID와 `--` 뒤의 구체적 justification을 요구한다.
- AI recommendation, TLS bypass와 remote source upload option을 사용하지 않는다.
- config가 필요할 때 module root 한 곳에 두고 path-wide exclusion을 허용하지 않는다.

### govulncheck

```text
govulncheck ./...
```

- pinned tool로 실행하되 최신 Go vulnerability database 조회는 network-backed input으로 취급한다.
- 호출 가능한 known vulnerability가 보고되면 실패한다.
- database/network/tool failure는 operational failure이며 `clean` 결과가 아니다.
- vulnerability가 없는 것이 Artifact 안전성 또는 Heliopause Policy `ALLOW`를 의미하지 않는다.
- false positive 또는 database mapping 문제는 advisory ID, 영향 분석과 임시 만료 조건을 기록한다.

### Gitleaks

```text
gitleaks dir --redact --no-banner .
```

- local default는 Heliopause module tree의 tracked·untracked file을 검사한다.
- output에서 secret 후보 값을 redaction한다.
- 실제 Secret을 baseline에 넣어 승인하지 않는다. 발견 시 노출 범위를 조사하고 실제 credential이면 먼저 revoke/rotate한다.
- synthetic credential/honeytoken fixture가 탐지되면 실제 유효 Secret이 아님을 확인하고 exact fixture path·rule·값 pattern에만 좁은 allowlist를 적용한다.
- broad directory ignore, generic entropy rule 비활성화와 blanket `gitleaks:allow`를 사용하지 않는다.
- Git history 또는 pull request commit range 검사는 Step 11에서 event별 범위를 확정한다.

## 13. Generated code, fixture와 build tag

- generated code는 marker와 재생성 명령을 가지며 formatter·compile·security scan 대상이다.
- generator가 필요해질 때 exact version을 pin하고 생성 결과 drift를 read-only check로 검증한다.
- test fixture와 golden file은 secret scan 대상에서 기본 제외하지 않는다.
- synthetic malicious fixture의 scanner suppression은 fixture 전체가 아니라 exact finding 단위로 관리한다.
- build tag는 environment capability를 분리하기 위해 사용하며 실패하는 production code나 flaky test를 숨기는 용도로 사용하지 않는다.
- 기본 `./...`에서 제외되는 tag 조합은 Step 11의 명시적 job 또는 구현 milestone에서 최소 한 번 compile/test한다.

## 14. Failure와 suppression 정책

### 실패 분류

```text
Finding
→ tool이 정상 실행되어 위반·취약점·secret 후보를 발견

Execution Failure
→ tool crash, timeout, malformed config, dependency load 또는 network failure

Unavailable / Unsupported
→ 현재 runner가 해당 profile prerequisite를 제공하지 않음
```

세 경우를 모두 빈 성공으로 변환하지 않는다. 다만 CI에서 어떤 runner/profile이 필수인지와 unavailable 처리 방식은 Step 11에서 정한다.

### Suppression

- exact tool, rule ID, code location과 이유를 기록한다.
- 실제 안전 근거, 제거 조건과 추적 가능한 work item을 요구한다.
- tool 전체, package 전체 또는 directory 전체 suppression을 기본 금지한다.
- config에서 severity threshold를 내려 finding을 숨기지 않는다.
- scanner 결과 파일 생성을 위해 exit code를 무조건 0으로 만드는 option을 gate 명령에 사용하지 않는다.
- flaky test quarantine을 장기 허용하지 않는다. 원인 수정 전에는 해당 필수 profile이 성공한 것으로 표시되지 않는다.

## 15. Tool configuration 배치

configuration은 필요할 때만 module root 또는 tool의 canonical 위치에 한 번 생성한다.

```text
go.mod / go.sum
→ product Go module과 dependency version source

scripts/tools.lock.json
→ external quality tool package·exact version·command·setup Go version source

staticcheck.conf
→ default와 다른 설정이 실제로 필요한 경우만

.gitleaks.toml
→ synthetic fixture 등 좁은 project-specific rule이 필요한 경우만

scripts/check/
→ canonical profile orchestration과 architecture/docs checker
```

빈 config, 미래용 allowlist와 아직 사용하지 않는 tool file을 미리 만들지 않는다. tool configuration 변경은 source code와 동일하게 review한다.

## 16. Step 11로 전달하는 Quality Gate 단위

Step 11은 아래 독립 결과를 조합한다.

```text
format
module consistency
compile/type check
architecture
vet
staticcheck
default test
security source scan
secret scan
race
vulnerability
integration
e2e
docs
```

하나의 거대한 job으로 합쳐 어떤 검사가 실패했는지 숨기지 않는다. 동시에 같은 source와 pinned configuration을 사용하며, local check와 CI가 서로 다른 rule set을 갖지 않게 한다.

## Step 10 Invariant

1. standard Go toolchain을 formatting, compile/type check와 test의 기준으로 사용한다.
2. 외부 quality tool은 고유한 검사 공백이 있을 때만 추가하고 exact version으로 고정한다.
3. canonical check는 repository-owned entrypoint에서 동일 명령과 configuration을 사용한다.
4. check profile은 source와 configuration을 수정하지 않으며 `format`만 명시적으로 변경할 수 있다.
5. compile, static analysis, test와 security scan의 성공을 서로 대체하지 않는다.
6. architecture checker로 Step 8의 import direction을 자동 검증한다.
7. unit/default test는 public network, 실제 credential과 Host 사용자 환경에 의존하지 않는다.
8. integration/E2E/race/fuzz는 capability와 resource가 다른 별도 bounded profile로 실행한다.
9. govulncheck의 network/database failure를 취약점 없음으로 처리하지 않는다.
10. scanner finding과 scanner execution failure를 구분하되 둘 다 성공으로 숨기지 않는다.
11. 실제 Secret은 baseline으로 승인하지 않고 synthetic fixture suppression은 exact 범위로 제한한다.
12. suppression은 rule·위치·근거·제거 조건을 가진 좁고 추적 가능한 예외여야 한다.
13. coverage percentage와 external tool verdict가 기존 Domain/Policy 보안 의미를 변경하지 않는다.

## 구현 영향

- Step 11은 각 profile의 trigger, runner OS/Go matrix, cache, artifact, timeout과 merge 차단 기준을 확정한다.
- Step 12는 architecture checker, parser fuzzing, integration environment와 security fixture 준비를 milestone에 배치한다.
- Step 13은 Step 14 시작 시 생성할 `go.mod`, pinned tool과 check runner 작업을 queue로 만든다.
- Step 14는 첫 vertical slice와 함께 필요한 최소 tool declaration·checker·test만 생성한다.

## 유보 사항

- 구현 시작 시점의 exact Go patch version과 최소 지원 version
- Staticcheck, gosec, govulncheck와 Gitleaks의 exact pinned version
- module path와 실제 quality tool lock entry
- Step 11 CI event별 mandatory profile과 timeout
- supported OS/architecture compile matrix
- vulnerability database/cache와 network failure retry
- integration test runtime·tool image와 immutable identity
- E2E disposable target 구현
- first vertical slice 이후 coverage regression 기준
- release artifact SBOM·provenance·signature 도구
- 유지보수되는 license scanner 도입 여부

위 항목은 Step 11~13 또는 관련 구현 직전에 확정한다. 도구 제품을 바꾸더라도 Step 10의 검사 책임과 실패 의미는 유지한다.

## 공식 참고 자료

- [Go command documentation](https://pkg.go.dev/cmd/go)
- [Go toolchain selection](https://go.dev/doc/toolchain)
- [Go race detector](https://go.dev/doc/articles/race_detector)
- [Go vulnerability management](https://go.dev/doc/security/vuln/)
- [Staticcheck documentation](https://staticcheck.dev/docs/)
- [gosec](https://github.com/securego/gosec)
- [Gitleaks](https://github.com/gitleaks/gitleaks)

## 누락 점검

- [x] formatter와 non-mutating format check
- [x] Go/tool version pinning과 update policy
- [x] module tidy·integrity와 dependency review
- [x] production·test compile/type check
- [x] `go vet`와 Staticcheck
- [x] architecture import direction 자동 검증
- [x] unit·contract·integration·E2E 구분
- [x] race detector와 fuzz target 기준
- [x] coverage의 역할과 초기 threshold 유보
- [x] gosec source security scan
- [x] govulncheck known vulnerability scan
- [x] Gitleaks current tree·synthetic fixture 정책
- [x] documentation link·fence 검사
- [x] generated code·fixture·build tag 규칙
- [x] check profile과 network/capability 분리
- [x] finding·execution failure·unavailable 구분
- [x] suppression과 no-fail 금지
- [x] Step 11 Quality Gate 전달 단위
