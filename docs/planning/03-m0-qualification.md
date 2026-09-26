# M0 Qualification — Implementation Foundation

이 문서는 M0 exit criteria와 Global Definition of Done을 재현 가능한 local·CI evidence에 대조한 canonical completion record다. M1의 Domain/API 결정을 대신하지 않는다.

## Qualification target

- Branch: `milestone/m0-foundation`
- Pull request: [#1](https://github.com/rahoney/heliopause/pull/1)
- Clean-checkout baseline: `52ddae4ba9a50ff0cacea4de766830603585890f`
- Checkout security update: `5ac4c52`
- Milestone completion: `011dd4f`
- Checkout v7.0.1 qualification CI: run `31664225839`
- Final milestone completion CI: run `31664383069`

## M0 exit criteria

| Criterion | Evidence | Result |
| --- | --- | --- |
| Linux/macOS build | PR Quick의 Ubuntu production build, Minimum Go Ubuntu platform, macOS 26 Intel platform | PASS |
| single module root | clean checkout에서 `go.mod` 하나; nested module과 `src/` 없음 | PASS |
| canonical quality entrypoint | fresh external cache bootstrap 뒤 `quick`, `docs`, `platform` 통과; 실행 후 clean worktree | PASS |
| CI trust contract | full-SHA Actions, `contents: read`, secret/write permission 없음, credential persistence와 Action cache 비활성 | PASS |
| fail-closed aggregate | 네 child success만 `Required` success; failure/cancelled/skipped/missing result test | PASS |
| no future placeholder | `cmd/helox`, `internal/bootstrap`, `internal/cli`, `scripts/check`만 존재; Domain/Port/Adapter/Sandbox/Promotion package 없음 | PASS |
| version/supply-chain record | Go, Cobra, Staticcheck, runner, Action release·license·support 근거와 exact identity 기록 | PASS |

## Global Definition of Done 적용

- M0 사용자 path인 `helox --help`가 process smoke test로 동작하고 unknown command, nil writer와 cancelled context failure가 test된다.
- M0에는 Port/Adapter, Artifact input, cleanup boundary와 security scanner capability가 없으므로 해당 contract/finding test는 적용 전이다. 이를 fake success나 `ALLOW`로 노출하지 않는다.
- architecture, format, build, vet, Staticcheck, default test와 활성 CI gate가 통과한다.
- source와 CI에 실제 Secret, credential, 개인 path, raw unbounded output이 없다.
- M0-001~008의 decision, implementation, check와 limitation은 Queue와 이 문서에서 commit/run으로 추적된다.

## Active checks

### Local canonical

- `bootstrap`: exact module과 Staticcheck를 source tree 밖 cache에 준비하고 identity를 확인
- `quick`: format, module drift/integrity, production/test build, architecture, workflow contract, vet, Staticcheck, default test
- `docs`: Markdown local link와 fence 검사
- `platform`: production build와 default test
- `format`: 유일한 명시적 source-mutating profile; gate에서는 실행하지 않음

### Required CI

- `Quick`: Ubuntu 24.04, Go 1.26.5
- `Docs`: Ubuntu 24.04, Go 1.26.5
- `Minimum Go`: Ubuntu 24.04, Go 1.25.12
- `macOS`: macOS 26 Intel, Go 1.26.5
- `Required`: 위 네 결과가 모두 `success`일 때만 성공

`actionlint` 1.7.12는 M0 qualification 진단에 사용했지만 repository lock/required gate에는 포함하지 않았다. known workflow contract는 repository-owned validator가 Quick에서 검사한다. actionlint를 required tool로 활성화하려면 별도 공급망 lock과 CI bootstrap 변경을 요구한다.

## Supply-chain review

| Component | Qualified identity | Review result |
| --- | --- | --- |
| Go default | `1.26.5` | 최신 stable patch; `crypto/tls`, `os` security fix 포함 |
| Go minimum | `1.25.12` | minimum supported exact patch; platform profile 실제 통과 |
| Cobra | `v1.10.2` | 최신 stable, Apache-2.0, module checksum 고정 |
| Staticcheck | `2026.1` / `v0.7.0` | 최신 stable, MIT, Go 1.25/1.26 분석 지원 |
| checkout | `v7.0.1` / `3d3c42e5aac5ba805825da76410c181273ba90b1` | 최신 signed stable; unsafe PR checkout 방어와 argument escaping 수정 때문에 M0 완료 전 갱신 |
| setup-go | `v7.0.0` / `b7ad1dad31e06c5925ef5d2fc7ad053ef454303e` | 최신 immutable signed stable |
| Runners | `ubuntu-24.04`, `macos-26-intel` | GA versioned GitHub-hosted runner; moving alias 사용 안 함 |

## Limitations and deferred risks

- Linux CI는 WSL2 qualification이 아니다.
- repository Actions는 현재 모든 Action을 허용하고 SHA pinning을 강제하지 않으며 `main` protection/ruleset이 없다. exact allowlist·SHA requirement·`Required` branch rule 제안은 Step 11에 기록했지만 별도 owner 승인 없이 적용하지 않았다.
- Race, gosec, Gitleaks, govulncheck, vulnerability, integration, E2E와 fuzz gate는 해당 code/capability와 exact tool lock이 없는 M0에서 활성화하지 않았다. 빈 success job은 없다.
- M0에는 Artifact, Domain workflow, Policy, Evidence, Sandbox와 Promotion이 없으며 이는 M1 이후 scope다.
- local 설계 문서와 agent 지침은 repository 배포 대상이 아니며 `.gitignore` 정책에 따라 PR에 포함되지 않는다.

## M1 handoff

다음 실행 항목은 `M1-001 — Domain workflow entry decision` 하나다. 최소 Domain type/schema, 상태 transition, Application/Port API, Policy input/reason, CLI exit/result contract와 synthetic fake scenario를 결정한 뒤에만 M1 구현 queue를 연다. 미래 Adapter, Sandbox, Evidence Store 또는 Promotion placeholder를 생성하지 않는다.
