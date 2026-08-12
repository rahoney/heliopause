# Step 13 — Current Work Queue

이 문서는 Heliopause에서 **지금 실행할 작업, 순서, 상태와 완료 증거**를 관리한다. Milestone의 범위와 완료 조건은 [Step 12 — Milestones](./01-milestones.md)가 소유하며, 이 문서는 현재 milestone인 M0만 구현 가능한 크기로 분해한다.

상세 Architecture·Domain·Engineering 결정을 복제하지 않는다. 각 work item은 필요한 canonical leaf 문서를 가리키고, Step 14는 해당 문서를 읽은 뒤 그 항목의 범위만 구현한다.

## 1. 현재 상태

```text
Current milestone: M0 — Implementation Foundation
Milestone status: IN_PROGRESS
Active work item: 없음
Next work item: M0-004 — Canonical check runner foundation
Next work item status: NOT_STARTED
Ready: Yes
```

M0-001 착수와 함께 M0가 `IN_PROGRESS`가 되었고, M0-003 완료 후 다음 실행 항목은 M0-004다.

## 2. 상태와 queue 규칙

Milestone과 work item에 같은 네 상태를 사용한다.

```text
NOT_STARTED
IN_PROGRESS
BLOCKED
COMPLETE
```

- 동시에 하나의 work item만 `IN_PROGRESS`로 둔다.
- 표의 순서가 기본 실행 우선순위다. 선행 항목이 완료되지 않았으면 다음 항목을 시작하지 않는다.
- `Ready: Yes`는 선행조건과 필수 입력이 준비되어 바로 시작할 수 있다는 뜻이며 완료나 성공을 뜻하지 않는다.
- `BLOCKED`에는 blocker, 마지막 확인 결과와 해제 조건을 기록한다.
- `COMPLETE`는 acceptance criteria와 요구 check가 실제 evidence로 확인된 경우에만 사용한다.
- 구현 중 발견한 별도 작업은 현재 항목의 완료에 필수일 때만 좁게 추가한다. 미래 milestone 항목은 해당 milestone이 가까워질 때 연다.
- scope가 Milestone, Architecture, Domain invariant를 바꾸면 queue에서 우회하지 않고 canonical 설계 문서를 먼저 갱신한다.

## 3. M0 실행 Queue

| Order | ID | Work item | Depends on | Status | Ready |
| --- | --- | --- | --- | --- | --- |
| 1 | M0-001 | Project Identity 결정 | Step 13 | COMPLETE | No |
| 2 | M0-002 | Go·Tool·CI Identity lock | M0-001 | COMPLETE | No |
| 3 | M0-003 | Go module과 최소 process bootstrap | M0-002 | COMPLETE | No |
| 4 | M0-004 | Canonical check runner foundation | M0-003 | NOT_STARTED | Yes |
| 5 | M0-005 | Static analysis와 Quick profile 활성화 | M0-004 | NOT_STARTED | No |
| 6 | M0-006 | Quick·Docs·Required CI foundation | M0-005 | NOT_STARTED | No |
| 7 | M0-007 | Minimum Go·macOS와 repository gate 검증 | M0-006 | NOT_STARTED | No |
| 8 | M0-008 | M0 qualification과 M1 handoff | M0-007 | NOT_STARTED | No |

## 4. Work Item

### M0-001 — Project Identity 결정

**Outcome**

구현과 배포 identity로 사용할 Go module path와 `helox` command 이름을 확정한다.

**Read**

- [Project Decisions](../../PROJECT-DECISIONS.md)
- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [User Journey — CLI Structure](../user-journey-cli-ia/02-cli-structure.md)

**Work**

- 현재 repository와 module root를 기준으로 canonical Go module path 후보를 확인한다.
- 현재 repository, 주요 executable namespace와 현실적인 배포 경로에서 `helox` 충돌 가능성을 확인한다.
- command 이름을 유지할 수 없으면 구현 전에 대안을 결정하고 관련 canonical 문서를 갱신한다.
- 결정 근거와 확인 시점을 Project Decisions에 연결한다.

**Acceptance**

- module path와 CLI command가 하나의 값으로 확정되어 후속 파일에서 추측할 필요가 없다.
- Heliopause 독립 repository root가 단일 Go module root라는 Step 8 결정을 유지한다.
- 실제 scaffold, `go.mod` 또는 package는 아직 생성하지 않는다.

**Completion**

```text
Status: COMPLETE
CLI command: helox
Repository: github.com/rahoney/heliopause
Go module path: github.com/rahoney/heliopause
Evidence: local repository/PATH와 현실적인 public executable/package namespace 확인
Limitation: 법적 상표 검토와 최종 배포 channel 등록 가능성은 배포 전 재검토
Next: M0-002 — Ready: Yes
```

### M0-002 — Go·Tool·CI Identity lock

**Outcome**

M0 구현에 필요한 기본 Go/minimum Go, 최초 Staticcheck, GitHub-hosted runner와 Action identity를 당시 공식 지원 상태에 맞춰 고정한다.

**Read**

- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- 공식 source에서 안정 Go patch와 최소 지원 Go patch를 확인한다.
- versioned Ubuntu/macOS runner label의 지원 상태를 확인한다.
- 필요한 GitHub-owned Action release와 full commit SHA를 검증한다.
- M0 Quick profile에 필요한 Staticcheck exact version, setup Go 요구, license, 유지보수와 dependency risk를 검토한다.
- 선택 결과와 upgrade 원칙을 구현 가능한 lock input으로 기록한다.

**Acceptance**

- floating `latest`, version range, branch 또는 tag만으로 식별된 실행 요소가 없다.
- default/minimum Go의 역할과 runner/Action/tool identity가 서로 구분된다.
- 아직 사용하지 않는 gosec, Gitleaks, govulncheck 또는 미래 runtime을 미리 pin하지 않는다.
- source URL, 확인한 release/tag와 exact identity가 review 가능하게 기록된다.

**Completion**

```text
Status: COMPLETE
Commit: c6a353b71d5a6659b97e00a17a2e61fbba9ef82a
Checks: official release/support source review; tag/SHA ls-remote verification; local go/staticcheck version verification; Markdown local link/fence validation; git diff --check
Decision/Evidence: docs/engineering/03-quality-toolchain.md and docs/engineering/04-ci-quality-gate.md M0-002 identity lock
Limitations: tools.lock consumer는 M0-005, workflow consumer는 M0-006에서 생성하며 미래 security tool은 capability 활성화 시 별도 pin
Next: M0-003 — Ready: Yes
```

### M0-003 — Go module과 최소 process bootstrap

**Outcome**

프로젝트 루트의 단일 Go module과 `helox`의 가장 얇은 실행 경로를 만든다.

**Read**

- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Architecture — Foundation and Dependencies](../architecture/01-foundation-and-dependencies.md)

**Work**

- project root에 `go.mod`를 생성하고 M0-002의 Go 정책을 적용한다.
- 필요한 범위에서만 `cmd/helox`, `internal/bootstrap`, `internal/cli`를 생성한다.
- process cancellation과 exit ownership을 경계에 맞게 두고 business workflow는 추가하지 않는다.
- 최소 process smoke test와 Linux/macOS build validity를 마련한다.

**Acceptance**

- 별도 `src/`, nested module과 미래용 placeholder package가 없다.
- `cmd/helox`는 bootstrap 호출과 process exit만 소유한다.
- CLI/bootstrap에 Domain, Policy, Adapter shortcut이 없다.
- module과 smoke test가 public network·credential·Host project 없이 재현된다.

**Completion**

```text
Status: COMPLETE
Commit: 5e99b6028009af9506232e291dbda60a09ca9450
Checks: gofmt; go mod tidy -diff/verify; go build; compile-only test; go vet; Staticcheck 2026.1; full test; Linux amd64/macOS amd64·arm64 build; offline module list/graph; package import boundary; Markdown local link/fence validation; git diff --check
Implementation: root go.mod; cmd/helox process entrypoint; internal/bootstrap composition root; internal/cli Cobra boundary; host-independent process smoke test
Limitations: business workflow·Domain·Policy·Adapter·future package와 canonical check runner는 현재 항목 범위에서 추가하지 않음
Next: M0-004 — Ready: Yes
```

### M0-004 — Canonical check runner foundation

**Outcome**

로컬과 CI가 공유할 standard-library 기반 `scripts/check` entrypoint와 deterministic foundation profile을 만든다.

**Read**

- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)

**Work**

- module root에서만 실행되는 `go run ./scripts/check <profile>` 구조를 만든다.
- read-only `format check`, module drift, build/type validity와 `docs` 검사를 구현한다.
- local Markdown link, code fence와 의도하지 않은 absolute local link를 검사한다.
- 현재 존재하는 package에 대해 검증 가능한 import rule을 구현하고, 이후 새 package가 추가되면 같은 checker에 규칙을 증분한다. 존재하지 않는 미래 package를 위한 fake pass 규칙은 만들지 않는다.
- command/argument 분리, timeout, bounded output과 첫 실패 보존을 적용한다.

**Acceptance**

- `format` 외 profile은 source/configuration을 변경하지 않는다.
- check가 source tree 밖 임의 executable이나 PATH 우선순위에 의존하지 않는다.
- finding과 tool/check execution failure가 모두 non-zero이며 원인이 구분된다.
- 문서 router가 깨진 fixture와 정상 tree를 deterministic하게 판별한다.

### M0-005 — Static analysis와 Quick profile 활성화

**Outcome**

Go 기본 검사와 pinned Staticcheck를 canonical `quick` profile로 연결한다.

**Read**

- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)

**Work**

- `scripts/tools.lock.json`에 실제 사용하는 Staticcheck identity만 기록한다.
- source tree 밖 project-specific cache를 사용하는 `bootstrap`을 구현한다.
- `quick`에 format check, module drift, build/type validity, architecture, `go vet`, Staticcheck와 default test를 연결한다.
- bootstrap network phase와 offline check phase를 분리하고 version mismatch를 실패시킨다.

**Acceptance**

- product `go.mod`가 quality tool dependency graph를 소유하지 않는다.
- `@latest`, floating download, `curl | sh`와 임의 PATH executable 실행이 없다.
- `quick`은 read-only이고 local clean checkout에서 반복 가능하다.
- M0-002의 pinned identity와 실제 실행 binary version이 일치한다.

### M0-006 — Quick·Docs·Required CI foundation

**Outcome**

GitHub Actions에서 실제 local profile을 실행하는 첫 required workflow를 만든다.

**Read**

- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- repository root에 Heliopause 전용 CI workflow를 생성하고 module working directory를 명시한다.
- versioned Ubuntu와 pinned Go/Action identity로 `Quick`, `Docs`, `Required`를 활성화한다.
- `pull_request`, `push` to `main`, `merge_group`, bounded `workflow_dispatch` trigger를 적용한다.
- fork/PR secret 없음, 최소 token permission, checkout credential 비보존과 cache 비활성화를 적용한다.
- `Required`가 failure/cancel/unexpected skip을 성공으로 변환하지 않게 한다.

**Acceptance**

- CI가 local canonical profile과 다른 검사 명령을 재구현하지 않는다.
- 모든 `uses:`가 full SHA와 사람이 식별 가능한 release 주석을 가진다.
- placeholder Security/Vulnerability/Race/Integration/E2E job이 없다.
- 실제 child failure와 cancellation에 aggregate가 실패하는 test 또는 검증 evidence가 있다.

### M0-007 — Minimum Go·macOS와 repository gate 검증

**Outcome**

지원 toolchain/platform 경계를 실제 CI에서 검증하고 stable aggregate를 repository merge gate로 사용할 준비를 마친다.

**Read**

- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)
- [Step 12 — Milestones](./01-milestones.md)

**Work**

- minimum Go Ubuntu와 pinned Go macOS build/default test job을 활성화한다.
- 두 job을 `Heliopause CI / Required` dependency에 포함한다.
- 독립 repository의 workflow 이름과 repository setting 적용 영향을 확인한다.
- branch rule과 Actions SHA/allowlist setting의 적용 범위를 제안하고, repository-wide 변경은 owner의 명시적 승인을 받은 경우에만 적용한다.

**Acceptance**

- primary, minimum Go와 macOS 결과가 서로 독립적으로 실제 실행된다.
- unavailable runner, failed 또는 skipped required job이 aggregate success가 되지 않는다.
- Linux build 결과를 WSL2 qualification으로 표시하지 않는다.
- repository-wide branch rule 적용은 M0 완료의 필수 조건이 아니다. 적용 권한·운영 범위가 없으면 제안과 미적용 사유를 기록하고, Heliopause CI 자체의 gate 동작 검증이 완료되면 M0 진행을 차단하지 않는다.

### M0-008 — M0 qualification과 M1 handoff

**Outcome**

M0 exit criteria를 evidence로 닫고 다음에는 M1 entry decision만 열 수 있게 한다.

**Read**

- [Step 12 — Milestones](./01-milestones.md)
- [Step 8 — Directory Structure](../engineering/01-directory-structure.md)
- [Step 9 — Coding / Security Rules](../engineering/02-coding-security-rules.md)
- [Step 10 — Quality Toolchain](../engineering/03-quality-toolchain.md)
- [Step 11 — CI + Quality Gate](../engineering/04-ci-quality-gate.md)

**Work**

- M0의 모든 exit criteria와 Global Definition of Done을 실제 command/CI evidence에 대조한다.
- version 공급망 검토, active check 목록, 제한과 유보 사항을 기록한다.
- PROJECT-DECISIONS, experiment README와 이 queue의 상태를 갱신한다.
- M1 entry decision을 실행 가능한 design work item으로 분해하되 M1 구현 항목 전체를 미리 만들지 않는다.

**Acceptance**

- M0-001~007이 모두 `COMPLETE`이고 미해결 필수 gate가 없다.
- clean checkout의 canonical local check와 활성 CI가 통과한다.
- M0 milestone completion commit과 검증 결과를 참조할 수 있다.
- 다음 active item은 M1 entry decision 하나이며 미래 Adapter/Sandbox/Promotion placeholder가 없다.

## 5. Work Item 완료 기록

work item을 완료할 때 같은 변경에서 다음을 기록한다.

```text
Status: COMPLETE
Commit: <commit SHA>
Checks: <canonical profile 또는 CI run>
Decision/Evidence: <필요한 canonical link>
Limitations: <없음 또는 명시적 제한>
Next: <다음 ID와 Ready 여부>
```

- 상세 구현 일지를 이 문서에 누적하지 않는다. commit과 canonical 문서를 참조한다.
- 완료된 항목은 표에서 삭제하거나 번호를 재사용하지 않는다.
- 다음 항목이 시작 가능하면 `Ready: Yes`로 바꾸고, 실제 시작 전에는 `IN_PROGRESS`로 표시하지 않는다.
- 구현 도중 queue 자체를 바꿨다면 변경 이유와 영향을 해당 commit에서 설명한다.

## 6. 현재 재개 지점

M0-001과 M0-002는 완료되었다. 다음 작업은 **M0-003 — Go module과 최소 process bootstrap**이다.

```text
M0: IN_PROGRESS
M0-002: COMPLETE
M0-003: NOT_STARTED
Active work item: 없음
Next work item: M0-003 — Ready: Yes
```

M0-003에서는 M0-002의 Go identity를 적용해 단일 module과 최소 process bootstrap만 구현한다. check runner 또는 workflow는 후속 Queue 순서에 따라 생성하고, 한 번에 M0 전체를 구현하지 않는다.

## Step 13 Invariant

1. Current Work Queue에는 현재 milestone의 실행 항목만 둔다.
2. 한 번에 하나의 work item만 `IN_PROGRESS`다.
3. work item은 canonical 설계를 링크하며 상세 결정을 복제하지 않는다.
4. 선행조건과 acceptance가 없는 작업을 시작하지 않는다.
5. 실제 검증 evidence 없이 `COMPLETE`로 표시하지 않는다.
6. 외부 상태 변경과 repository-wide 보안 설정은 필요한 명시적 승인을 받는다.
7. 미래 capability의 placeholder package, config 또는 CI job을 queue가 요구하지 않는다.
8. 독립 repository에서는 현재 next item부터 Queue 순서대로 실행한다.

## 누락 점검

- [x] 현재 milestone, active item과 next item
- [x] 단일 상태 모델과 Ready 의미
- [x] M0의 순차 work item과 dependency
- [x] 각 항목의 canonical document route
- [x] Outcome, Work와 Acceptance
- [x] 완료 evidence 기록 형식
- [x] blocker와 queue 변경 규칙
- [x] repository-wide 변경 승인 경계
- [x] M1 이후 작업의 지연 생성 원칙
- [x] 독립 repository의 정확한 재개 지점
