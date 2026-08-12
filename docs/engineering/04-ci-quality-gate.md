# Step 11 — CI + Quality Gate

이 문서는 Step 10에서 확정한 local check profile을 GitHub Actions에서 언제, 어떤 신뢰 수준과 platform으로 실행하고 어떤 결과가 merge를 차단하는지 정의한다. 기존 Threat Model·Architecture·Domain 의미를 변경하지 않으며, Heliopause의 `Policy Decision`과 repository source code의 CI Quality Gate를 동일한 개념으로 취급하지 않는다.

Step 11에서도 workflow, branch rule 또는 implementation scaffold를 실제로 생성하지 않는다. 파일과 설정은 Step 14에서 첫 vertical slice에 필요한 범위로 만든다.

## 1. CI provider와 repository 위치

- CI provider는 현재 원격 저장소와 일치하는 GitHub Actions를 사용한다.
- Heliopause project/module root는 `experiments/heliopause-artifact-airlock/`이다.
- GitHub Actions workflow는 GitHub repository root의 `.github/workflows/`에만 둘 수 있으므로 다음 위치를 사용한다.

```text
.github/workflows/heliopause-ci.yml
.github/workflows/heliopause-security.yml
```

- workflow `run` step의 기본 working directory는 Heliopause module root로 고정한다.
- checkout·Go setup 같은 action step과 repository-level path는 working directory가 자동 적용되지 않으므로 입력 path를 명시한다.
- Heliopause가 향후 독립 repository로 이동하더라도 job/profile 계약과 required gate 이름은 유지하고 path만 조정한다.
- 초기에는 reusable workflow, composite action 또는 별도 CI framework를 만들지 않는다. 실제 중복이 생긴 뒤에만 추출한다.

## 2. Workflow 분리

### `heliopause-ci.yml`

source 변경의 merge 가능성을 판단하는 필수 검증이다.

```text
pull_request
push to main
merge_group
workflow_dispatch
```

### `heliopause-security.yml`

시간에 따라 입력이 달라지거나 긴 실행 시간을 요구하는 보안 재검증이다.

```text
schedule: weekly
workflow_dispatch
```

정기 workflow는 다음을 수행한다.

- 최신 vulnerability database 기반 govulncheck 재실행
- Heliopause 관련 path의 default branch Git history secret scan
- 구현된 fuzz target의 bounded fuzzing
- 구현 후 활성화된 장시간 security/integration profile

정기 검사 실패는 과거 merge의 상태를 소급 변경하지 않지만 현재 default branch의 유효한 문제로 triage한다. 자동으로 `ALLOW`, suppression 또는 dependency upgrade를 생성하지 않는다.

### Release workflow

release build, checksum, SBOM, signing, provenance/attestation과 publish는 CI workflow에 섞지 않는다. 배포 정책과 milestone이 확정된 뒤 별도의 trusted release workflow로 정의한다.

## 3. Trigger 정책

| Event | Trust | Required work |
| --- | --- | --- |
| `pull_request` | Untrusted | 모든 PR required gate; secret·write permission 없음 |
| `merge_group` | Untrusted merged candidate | merge queue candidate에 동일 required gate 재실행 |
| `push` to `main` | Trusted ref, untrusted source 가능 | merge commit에서 필수 gate 재검증 |
| `workflow_dispatch` | Repository policy에 따라 허용된 manual actor | 선택 가능한 고정 profile 재실행; arbitrary command 입력 금지 |
| weekly `schedule` | Trusted default branch | vulnerability, history secret, fuzz와 장시간 검증 |

- `pull_request_target`에서 PR code를 checkout·build·test·실행하지 않는다.
- issue, comment, label, branch name, PR title/body와 같은 untrusted context를 `run` script에 직접 삽입하지 않는다.
- manual input은 미리 정의한 profile enum과 bounded parameter만 허용하며 shell fragment, path 또는 arbitrary argument로 사용하지 않는다.
- branch push 전체에 중복 CI를 실행하지 않는다. feature branch는 local check와 PR CI를 사용하고, protected `main` push에서 결과를 재확인한다.
- merge queue를 사용하지 않더라도 `merge_group` trigger를 workflow에 포함해 활성화 시 required check가 누락되지 않게 한다.

## 4. Path filtering을 기본 사용하지 않는다

현재 Heliopause는 monorepo 하위 실험이지만 초기 CI에는 workflow-level `paths`/`paths-ignore` filter를 두지 않는다.

이유는 다음과 같다.

- required workflow가 아예 생성되지 않아 PR이 pending 상태가 되는 조건을 피한다.
- 문서 router, root experiment index와 workflow 자체 변경이 누락되지 않는다.
- change classifier와 third-party path-filter action을 추가하지 않는다.
- 초기 codebase에서는 항상 실행하는 비용보다 gate 일관성이 중요하다.

따라서 repository의 모든 PR에서 Heliopause aggregate check가 생성된다. Heliopause와 무관한 변경에서도 check는 실제 current Heliopause tree를 검증한다. 실행 비용이 실측상 문제가 되면 repository-owned change classifier를 도입하되 aggregate gate는 항상 생성하고 `not applicable` 판단 근거를 명시적으로 출력한다.

## 5. Runner와 Go matrix

### Runner 원칙

- untrusted PR code는 GitHub-hosted ephemeral runner에서만 실행한다.
- 개인 workstation, 장기 credential 또는 internal network에 접근할 수 있는 self-hosted runner에서 fork/PR code를 실행하지 않는다.
- `ubuntu-latest`, `macos-latest`처럼 움직이는 label 대신 구현 시점에 지원되는 명시적 versioned runner image를 고정한다.
- runner image 변경은 installed tool, filesystem, compiler, certificate와 runtime 변화를 검토한 명시적 변경으로 수행한다.
- Linux Dynamic Inspection test가 추가되더라도 GitHub-hosted runner에 Host secret이나 내부망 권한을 부여하지 않는다.

### Go matrix

| Axis | Runner | Go | Scope |
| --- | --- | --- | --- |
| Primary | versioned Ubuntu | pinned default Go patch | 전체 required local profile의 기준 |
| Minimum | versioned Ubuntu | exact minimum supported Go patch | build와 default test |
| macOS CLI | versioned macOS | pinned default Go patch | platform-neutral build와 default test |
| Linux race | versioned Ubuntu | pinned default Go patch | race profile |

- Go version은 semantic range나 `stable`/`oldstable` alias가 아니라 exact patch로 지정한다.
- setup action이 임의 newer toolchain으로 전환하지 않도록 `GOTOOLCHAIN=local`을 사용한다.
- minimum Go job은 quality tool을 build하지 않는다. 품질 도구가 요구하는 Go version과 Heliopause 최소 지원 version을 결합하지 않기 위함이다.
- Staticcheck, gosec, govulncheck와 Gitleaks는 pinned tool setup Go로 설치되고 primary job에서 실행한다.
- Windows native build는 MVP 대상이 아니다.
- WSL2는 Linux build 성공으로 검증됐다고 주장하지 않는다. 공식 WSL2 지원을 선언하는 milestone 전 승인된 disposable WSL2 runner에서 CLI E2E qualification을 별도로 수행해야 한다.

exact runner label, Go patch와 tool version은 Step 13~14에서 당시 지원 상태를 확인해 lock한다.

## 6. Required CI jobs

각 capability가 활성화된 뒤 `heliopause-ci.yml`이 갖추어야 할 목표 job set은 다음과 같다. Step 14에서는 §18의 순서에 따라 실제 검사가 존재하는 job만 생성한다.

| Stable check name | Runner | Step 10 work | Timeout | Required |
| --- | --- | --- | --- | --- |
| `Heliopause CI / Quick` | Ubuntu, pinned Go | `quick` | 15m | Yes |
| `Heliopause CI / Security` | Ubuntu, pinned Go | `security` + event-specific history secret scan | 15m | Yes |
| `Heliopause CI / Vulnerability` | Ubuntu, pinned Go | `vulnerability` | 10m | Yes |
| `Heliopause CI / Minimum Go` | Ubuntu, minimum Go | build + default test | 10m | Yes |
| `Heliopause CI / macOS` | macOS, pinned Go | build + default test | 15m | Yes |
| `Heliopause CI / Race` | Ubuntu, pinned Go | `race` | 20m | Yes |
| `Heliopause CI / Docs` | Ubuntu, pinned Go | `docs` | 5m | Yes |
| `Heliopause CI / Required` | Ubuntu | 현재 활성화된 required job의 aggregate result | 5m | Yes; branch rule target |

timeout은 job-level 상한이며 Step 10의 test command timeout도 유지한다. job timeout을 늘려 hung process를 정상화하지 않는다. 실제 duration 자료가 쌓이면 명시적 review로 조정한다.

### Required aggregate

`Heliopause CI / Required`는 `if: always()`에 해당하는 동작으로 모든 required dependency가 끝난 뒤 항상 생성된다.

```text
모든 required job = success
        ↓
Required = success

failure / cancelled / timed_out / unexpected skipped
        ↓
Required = failure
```

- aggregate job이 실제 검사를 다시 수행하지 않는다.
- 아직 구현되지 않아 활성화 전인 capability를 가짜 success job으로 만들지 않으며, 활성화된 required job은 모두 aggregate dependency에 포함한다.
- required child job 이름을 branch protection에 각각 등록하지 않고 stable aggregate check 하나를 요구한다.
- matrix 확대나 job 내부 분할이 branch protection 설정을 흔들지 않게 한다.
- expected capability가 없는 job을 `skipped` 성공으로 숨기지 않는다.
- GitHub incident나 runner shortage 때문에 실행되지 못한 검사를 성공으로 변환하지 않는다.

## 7. Integration, E2E와 capability activation

현재 runtime·storage·Promotion 구현이 없으므로 empty integration/E2E job을 만들지 않는다.

새 capability가 구현되면 같은 변경에서 다음을 수행한다.

1. immutable test tool/runtime identity와 synthetic fixture를 준비한다.
2. prerequisite probe와 bounded cleanup을 구현한다.
3. Step 10 `integration` 또는 `e2e` profile을 활성화한다.
4. CI job을 추가하고 required aggregate dependency에 포함한다.
5. 문서와 milestone 완료 조건을 갱신한다.

활성화된 required environment에서 runtime/tool 부재, startup failure 또는 capability unavailable은 실패다. 지원되지 않는 platform의 job을 억지로 생성하지 않으며 지원 범위는 MVP 문서와 일치시킨다.

Linux Dynamic Inspection E2E는 platform-neutral CLI E2E와 별도 job으로 둔다. 실제 Host credential, home directory, internal network 또는 사용자 project를 fixture로 사용하지 않는다.

## 8. Vulnerability gate

govulncheck는 PR마다 required job으로 실행한다.

- 호출 가능한 known vulnerability finding은 merge를 차단한다.
- vulnerability database, network 또는 tool failure도 clean 결과가 아니므로 job을 실패시킨다.
- 일시적 외부 장애라면 동일 commit에서 rerun하여 성공 결과를 얻어야 한다.
- suppression은 advisory ID, reachability/impact 분석, owner, 제거 조건과 만료 시점을 요구한다.
- vulnerability job을 장기간 non-blocking으로 낮추지 않는다.
- weekly schedule은 code 변경이 없어도 새 advisory를 재평가한다.

CI 취약점 검사는 Heliopause가 검사하는 Artifact의 Policy Decision을 만들지 않는다. 이는 Heliopause product source/toolchain의 merge gate다.

## 9. Secret scan 범위

| Event | Scan scope |
| --- | --- |
| Pull request / merge group | Heliopause current tree + base에서 candidate까지의 Heliopause 관련 commit range |
| Push to main | Heliopause current tree + push range의 Heliopause 관련 변경 |
| Weekly schedule | Heliopause tree와 Heliopause workflow/config path의 full reachable history |

- 이 절의 Heliopause current tree는 module root와 Heliopause 전용 workflow/config file을 포함한다.
- checkout history depth는 해당 scan 범위를 충족하는 최소값으로 명시한다.
- history scan은 Heliopause module root와 `heliopause-*.yml` workflow/config path로 제한하고 다른 실험의 과거 기록을 Heliopause suppression으로 소유하지 않는다.
- shallow history 때문에 요청 범위를 검사하지 못하면 실패한다.
- output은 secret 값을 redaction하고 raw patch 또는 전체 environment를 log에 쓰지 않는다.
- 실제 credential finding은 baseline 승인보다 revoke/rotate와 history 대응을 먼저 수행한다.
- synthetic credential/honeytoken fixture는 Step 10의 exact suppression 정책을 따른다.
- fork PR에는 repository secret을 제공하지 않으므로 secret scanner가 실제 secret을 필요로 하지 않는다.

## 10. Workflow security

### Action pinning과 allowlist

- 모든 `uses:` reference는 first-party GitHub action을 포함해 full-length commit SHA로 고정한다.
- 사람이 version을 식별할 수 있도록 검토한 release tag를 인접 주석으로 남긴다.
- repository settings에서도 full SHA pinning requirement를 활성화한다.
- 초기 allowlist는 GitHub-owned `checkout`, `setup-go` 등 실제 필요한 action만 허용한다.
- third-party action은 동일 기능을 repository-owned Go code 또는 fixed `run` step으로 안전하게 구현하기 어려운 경우만 Step 9 공급망 검토 후 추가한다.
- floating tag, branch, `@main`, `@master` 또는 marketplace의 자동 최신 version을 사용하지 않는다.
- Actions allowlist와 SHA requirement는 현재 shared repository 전체에 영향을 주므로 Step 14에서 다른 experiment workflow와 호환성을 확인하고 repository owner의 명시적 적용 범위를 결정한다.

### Token과 permission

- workflow top level은 필요한 최소 `GITHUB_TOKEN` permission만 명시하며, 기본적으로 `contents: read` 외 권한을 부여하지 않는다.
- job이 token을 필요로 하지 않으면 `permissions: {}`를 사용한다.
- checkout은 credential persistence를 비활성화한다.
- CI job에 `contents: write`, `pull-requests: write`, `packages: write`, `actions: write`, `id-token: write`를 부여하지 않는다.
- CI가 comment, label, commit, cache purge 또는 issue 생성 같은 외부 변경을 자동 수행하지 않는다.
- publish/attestation에 필요한 write/OIDC permission은 미래의 별도 release workflow job에만 좁게 부여한다.

### Secret과 untrusted input

- PR required workflow에는 repository, organization 또는 environment secret을 전달하지 않는다.
- source authentication이 필요한 integration은 fork PR required workflow에 넣지 않고 trusted environment와 별도 승인 경계를 요구한다.
- `${{ github.event.* }}`의 untrusted 문자열을 inline shell source에 삽입하지 않는다.
- 필요한 metadata는 environment/input으로 전달하고 repository-owned program에서 type·length·allowlist를 검증한다.
- workflow는 PR code가 생성한 output을 command, action reference, cache key의 executable fragment 또는 privileged path로 재해석하지 않는다.

### Shell과 external command

- workflow `run` block에는 repository가 고정한 command와 argument만 둔다.
- 복잡한 orchestration은 YAML shell string이 아니라 Step 10의 Go check runner에 둔다.
- pipe, command substitution과 dynamically assembled shell을 gate logic에 사용하지 않는다.
- `curl | sh`, runtime latest download와 downloaded script 직접 실행을 금지한다.

## 11. Dependency와 tool bootstrap

CI job은 다음 두 phase를 구분한다.

```text
Bootstrap phase — network allowed
→ exact Go toolchain
→ go mod download
→ tools.lock의 exact package@version 설치
→ checksum/version 확인

Check phase — default offline
→ GOTOOLCHAIN=local
→ GOPROXY=off 기반 offline module resolution
→ explicit cached executable path
→ canonical profile
```

- `go get`, `go mod tidy` mutation, `@latest`와 floating tool install을 CI에서 사용하지 않는다.
- bootstrap이 `go.mod`, `go.sum`, lock 또는 source를 변경하면 실패한다.
- tool version mismatch와 checksum verification failure는 실행 실패다.
- vulnerability profile의 database access와 명시적으로 활성화된 integration endpoint만 check phase network 예외다.
- 외부 network가 필요한 test는 default unit/contract test에 섞지 않는다.

## 12. Cache 정책

초기 CI는 GitHub Actions dependency/build/tool cache를 사용하지 않는다.

- `setup-go` 등 기본 cache를 제공하는 action에도 cache 비활성화를 명시한다.

이유는 다음과 같다.

- 초기 project 규모에서 bootstrap 비용이 아직 측정되지 않았다.
- cache는 signed trust source가 아니며 fork PR이 default branch cache를 읽을 수 있다.
- executable tool cache와 build cache poisoning surface를 만들지 않는다.
- cache key, write scope와 restore validation 복잡성을 먼저 추가하지 않는다.

실측상 필요해지면 별도 변경으로 도입하며 다음 조건을 모두 요구한다.

- secret, credential, Evidence, Artifact fixture output과 runtime data 저장 금지
- OS, exact Go, `go.sum`, `tools.lock`과 cache schema version을 key에 포함
- untrusted trigger의 default branch cache write 금지
- broad restore key 금지 또는 안전 근거 요구
- restored module에 `go mod verify`, tool에 version/identity 재확인
- cache miss에서도 동일 결과를 재생성할 수 있음

cache hit를 dependency integrity 또는 Quality Gate 성공 근거로 사용하지 않는다.

## 13. Concurrency, retry와 cancellation

- PR workflow는 workflow name과 PR/ref를 조합한 concurrency group을 사용한다.
- 같은 PR의 새 commit이 오면 이전 PR run을 취소한다.
- `main`, merge group, schedule과 release 성격의 run은 새 run 때문에 진행 중 검증을 취소하지 않는다.
- job/test timeout과 process cleanup은 cancellation에서도 유지한다.
- 자동 retry로 flaky test나 scanner finding을 숨기지 않는다.
- GitHub runner/network의 일시적 operational failure는 사람이 동일 commit을 명시적으로 rerun할 수 있다.
- rerun은 새 commit의 결과를 이전 commit에 적용하지 않는다.

## 14. Output, log와 artifact retention

- job name과 step name은 검사 책임을 안정적으로 드러낸다.
- 실패 시 command, profile, normalized failure category와 redacted summary를 출력한다.
- environment 전체, GitHub context 전체, credential-bearing URL, Artifact raw stdout/stderr와 사용자 식별 path를 dump하지 않는다.
- GitHub debug logging을 장기 기본값으로 활성화하지 않는다.
- 초기 CI는 별도 workflow artifact를 업로드하지 않고 bounded workflow log를 사용한다.
- test report, coverage, SARIF 또는 failure bundle이 실제로 필요해지면 synthetic/sanitized 내용과 최소 retention을 가진 독립 artifact로 추가한다.
- Heliopause 전용 artifact를 추가한다면 해당 artifact의 retention은 초기 14일 이하로 둔다.
- Actions log의 repository-wide retention 변경은 다른 experiment workflow에도 영향을 주므로 Heliopause CI만을 이유로 임의 변경하지 않고 repository 운영 결정으로 처리한다.
- PR CI가 build한 binary를 release 또는 trusted Promotion 대상으로 재사용하지 않는다.

## 15. Branch protection과 merge gate

Heliopause가 독립 repository가 되거나 현재 shared repository가 Heliopause gate를 default branch rule로 채택하면 다음 rule을 적용한다.

- pull request 없이 직접 merge하지 않음
- `Heliopause CI / Required` status check 성공 요구
- branch가 최신 base와 검증된 상태이거나 merge queue candidate로 검증됨
- unresolved review conversation이 있으면 merge 금지
- force push와 branch deletion 금지
- required CI bypass를 일상 workflow로 사용하지 않음

review approval 인원수, signed commit과 linear history는 repository 운영 방식·기여자 수와 함께 별도 결정한다. CI 문서가 사람 review를 자동으로 통과시키지 않는다.

required check name은 다른 workflow와 충돌하지 않게 project prefix를 포함한다. branch rule에는 expected GitHub Actions source를 지정할 수 있으면 해당 App/source를 고정한다.

현재 `tech-lab`의 branch protection은 다른 experiment에도 영향을 주는 repository-wide 운영 변경이다. Step 11 문서 완료만으로 이를 즉시 변경하지 않으며 Step 14에서 repository owner가 적용 범위를 명시적으로 승인한 뒤 설정한다.

## 16. Failure 의미와 incident 처리

| State | Meaning | Gate |
| --- | --- | --- |
| Finding | check가 정상 실행되어 위반 발견 | Fail |
| Execution failure | tool, runner, network, config, timeout 실패 | Fail |
| Cancelled | 새 PR commit 또는 사람 취소 | Not success |
| Unexpected skipped | required prerequisite/job 미실행 | Fail aggregate |
| Unsupported before activation | 아직 지원 범위가 아닌 capability | job 자체를 만들지 않음; 문서에 남김 |
| Success | 요구된 모든 check 완료 | Pass |

- CI failure를 Heliopause Artifact `BLOCK`으로 변환하지 않는다.
- scheduled security failure는 owner가 원인을 분류하고 remediation 또는 좁고 만료되는 suppression을 기록한다.
- secret incident는 log/artifact 노출 범위를 확인하고 credential revoke/rotate를 우선한다.
- compromised action/tool 또는 cache가 의심되면 관련 version/SHA 사용을 중단하고 trusted source에서 재검증한다.
- 외부 서비스 장애 중에도 gate를 임의 green으로 만들지 않는다.

## 17. CI configuration 변경 규칙

다음 변경은 security-sensitive review 대상으로 취급한다.

```text
workflow trigger
permissions / secret / environment
uses action SHA와 allowlist
runner image
Go/tool version
cache path·key·write scope
required job와 aggregate dependency
scanner suppression
integration network/runtime
artifact upload와 retention
```

- workflow/config 변경도 해당 PR의 기존 CI에서 가능한 범위까지 검증한다.
- GitHub가 workflow를 parse하지 못하거나 required check를 생성하지 못하면 merge 가능한 success로 간주하지 않는다.
- 별도 workflow syntax checker의 채택은 Step 14에서 검토하며, 도입하면 Step 10의 tool lock·공급망 원칙을 적용한다.
- action update automation이 PR을 만들 수는 있지만 auto-merge하지 않는다.
- CI 변경 PR은 diff에서 permission 확대, pin 변화와 required gate 삭제를 사람이 확인한다.
- exact SHA와 인접 release tag 주석을 같은 변경에서 갱신한다.

## 18. Step 12~14 활성화 순서

Step 14에서 처음부터 모든 job을 placeholder로 만들지 않는다.

```text
1. go.mod + check runner
2. Quick + Docs + Required
3. unit test 등장 시 Minimum Go + macOS
4. concurrent code 등장 시 Race
5. Gitleaks, gosec, govulncheck 등 각 security tool이 pin되는 즉시 해당 Security/Vulnerability job 활성화
6. concrete boundary 구현 시 Integration / E2E
7. fuzz target 구현 시 scheduled Fuzz
8. 배포 결정 후 별도 Release workflow
```

각 job이 활성화되는 변경은 required aggregate와 branch rule을 함께 갱신한다. 보안 job을 구현 편의를 이유로 무기한 optional 상태에 두지 않는다.

## Step 11 Invariant

1. GitHub Actions는 Step 10의 canonical profile과 같은 source/configuration을 실행한다.
2. PR code는 GitHub-hosted ephemeral runner에서 secret과 write permission 없이 실행한다.
3. `pull_request_target`의 privileged context에서 PR code를 실행하지 않는다.
4. action은 full commit SHA로 고정하고 floating reference를 사용하지 않는다.
5. exact Go patch, versioned runner와 pinned tool을 사용하며 자동 latest 추종하지 않는다.
6. required check는 stable aggregate 하나로 노출하고 모든 필수 child 결과를 fail-closed로 결합한다.
7. finding, execution failure, cancelled와 unexpected skipped를 성공으로 변환하지 않는다.
8. minimum Go, primary Linux, macOS와 race를 서로 다른 capability로 실제 검증한다.
9. vulnerability database/network failure를 clean 결과로 처리하지 않는다.
10. actual secret은 PR workflow에 제공하지 않고 secret scan output을 redaction한다.
11. 초기에는 cache와 workflow artifact를 사용하지 않으며 도입 시 신뢰 경계를 별도 검토한다.
12. integration/E2E는 concrete capability와 immutable fixture가 생길 때 required gate로 활성화한다.
13. PR build output은 release/publish Artifact로 재사용하지 않는다.
14. CI Quality Gate와 Heliopause Artifact Policy Decision을 구분한다.

## 구현 영향

- Step 12는 check runner, CI foundation, security gate, platform qualification과 integration activation을 milestone에 배치한다.
- Step 13은 첫 workflow와 branch rule을 생성할 수 있는 작은 작업 단위로 queue를 작성한다.
- Step 14는 당시 지원되는 runner/action/Go/tool exact identity를 확인해 workflow와 lock을 생성한다.
- runtime, credential 또는 release permission이 필요한 job은 관련 설계 없이 CI foundation에 미리 추가하지 않는다.

## 유보 사항

- implementation 시점의 exact GitHub-hosted runner label
- exact Go patch, tool version과 action full SHA
- repository가 public/private인지에 따른 Actions 기능과 retention 한도
- WSL2 qualification runner와 절차
- concrete Linux isolation runtime의 CI 실행 방식
- integration endpoint·fixture와 external tool identity
- fuzz target별 time budget
- workflow syntax checker의 구체 구현
- branch review approval 수, signed commit과 linear history
- release build·SBOM·signing·provenance·publish workflow
- cache를 도입할 성능 기준

위 항목은 Step 12~14 또는 해당 capability 구현 직전에 확정한다. 유보 항목이 required gate의 fail-closed 의미를 약화하지 않는다.

## 공식 참고 자료

- [GitHub Actions에서 Go build와 test](https://docs.github.com/en/actions/tutorials/build-and-test-code/go)
- [GitHub Actions workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)
- [GitHub Actions script injection](https://docs.github.com/en/actions/concepts/security/script-injections)
- [GitHub Actions cache security](https://docs.github.com/en/actions/concepts/workflows-and-actions/dependency-caching)
- [GitHub Actions full SHA requirement](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository)
- [Securely using `pull_request_target`](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target)
- [Protected branch required checks](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)

## 누락 점검

- [x] GitHub Actions와 monorepo 내 module root
- [x] PR/main/merge queue/manual/schedule trigger
- [x] required workflow와 scheduled security workflow 분리
- [x] path filter와 stable status check 정책
- [x] GitHub-hosted runner와 self-hosted 금지 경계
- [x] exact Go·minimum Go·Linux·macOS·race matrix
- [x] WSL2 qualification 분리
- [x] required job·timeout·aggregate gate
- [x] integration/E2E capability activation
- [x] PR·main·weekly secret scan 범위
- [x] action full SHA와 minimal permission
- [x] fork PR secret 금지와 untrusted context 처리
- [x] bootstrap/check network 분리
- [x] 초기 cache 미사용과 후속 도입 조건
- [x] concurrency·retry·cancellation
- [x] log·artifact·retention·PR binary 경계
- [x] branch protection과 merge gate
- [x] failure/unsupported/skip 의미
- [x] CI configuration security review
- [x] 단계별 job activation
- [x] Step 12~14 유보 항목
