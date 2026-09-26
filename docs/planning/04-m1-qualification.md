# M1 Qualification — Domain Workflow Skeleton

이 문서는 M1 exit criteria를 tracked clean clone과 PR CI에서 재현한 canonical completion record다. 실제 npm Adapter와 M2 entry decision을 대신하지 않는다.

## Qualification target

- Branch: `milestone/m1-domain-workflow`
- Pull request: [#2](https://github.com/rahoney/heliopause/pull/2)
- Qualified implementation head: `81b6d1f`
- Milestone completion: `e1ba0d5`
- Implementation qualification CI: run `31669932193`
- Final milestone completion CI: run `31670097293`
- Local clean clone: `/private/tmp/heliopause-m1-qualification`의 tracked branch checkout

## M1 exit criteria

| Criterion | Evidence | Result |
| --- | --- | --- |
| ecosystem-neutral Domain | `internal/core/domain`은 standard library만 사용하고 source/tool/OS concrete type이 없음; architecture gate 통과 | PASS |
| Run 생성 순서 | application test가 `Resolve`만 호출된 시점에 Run ID 생성, 이후 `Acquire` 호출을 검증 | PASS |
| 상태와 error 축 분리 | Run Outcome, Policy Decision, Operation Status, sanitized Operation Error가 별도 type/constructor/test로 검증됨 | PASS |
| fail-closed Policy | safe=`ALLOW`, unsupported required check=`MANUAL_REVIEW`, blocking Finding=`BLOCK`; operational failure에는 Policy 없음 | PASS |
| 동일 결과 presentation | human/JSON presenter가 같은 Operation/Run/identity/digest/decision을 참조하고 다섯 synthetic scenario의 exit code를 검증 | PASS |
| no Promotion | production source와 package audit에서 Staging/Promotion 생성·호출 없음; Inspect constructor에도 해당 Port 없음 | PASS |
| active platform gates | Ubuntu Quick, Docs, Go 1.25.12 Minimum Go, macOS 26 Intel, Required 모두 success | PASS |

## Local clean-clone evidence

- canonical `quick`, `docs`, `platform` 통과
- `go test -race -timeout=5m ./...` 통과
- `check-jsonschema`로 `schemas/operation-result-v1.schema.json` Draft 2020-12 metaschema 검증 통과
- tracked inventory에 `AGENTS.md`, `PROJECT-DECISIONS.md`, `docs/`, secret·credential·개인 환경 파일이 없음
- `internal/testutil/fakeworkflow`는 production bootstrap/command에서 import되지 않음
- `internal/promotion`, `internal/staging`, `internal/sandbox`와 빈 placeholder directory가 없음
- 단일 `go.mod`, 기존 Cobra dependency graph 유지, 새 external product dependency 없음

## Active checks

- canonical Quick: format, module drift/integrity, build/type validity, architecture, CI contract, vet, Staticcheck, default test
- canonical Docs와 Platform
- local race test
- PR CI: Quick, Docs, Minimum Go, macOS, Required
- JSON schema metaschema validator와 synthetic presenter contract test

## Limitations

- M1 fake는 test-only deterministic implementation이며 실제 network/filesystem/credential/process를 사용하지 않는다.
- production root CLI에는 ecosystem inspect command가 아직 없다. 실제 npm command와 Adapter는 M2에서 구현한다.
- production Evidence Store, static scanner, Controlled Intake, Sandbox, install, Staging과 Promotion은 구현하지 않았다.
- race는 local qualification에서 실행했으며 현재 Required CI child로 추가하지 않았다.
- repository-wide Actions allowlist/SHA enforcement와 main protection은 M0 limitation을 유지한다.

## M2 handoff

다음 실행 항목은 `M2-001 — npm Static Inspect entry decision` 하나다. npm reference/identity mapping, bounded network acquisition, Controlled Intake, static check taxonomy, Evidence Store, result evolution과 fixture를 결정한 뒤에만 M2 implementation queue를 연다.
