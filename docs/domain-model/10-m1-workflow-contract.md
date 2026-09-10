# M1 Entry Decision — Domain Workflow Contract

이 문서는 M1 fake 기반 `inspect` lifecycle을 구현할 때 필요한 최소 Go/domain/application 계약을 확정한다. 기존 Domain-001~007과 Contract-001~003의 invariant를 구체화하며 npm/PyPI/GitHub Adapter, production Evidence Store, Sandbox, Staging 또는 Promotion을 설계하지 않는다.

## 1. M1 vertical slice

M1의 단일 use case는 ecosystem-neutral `Inspect`다.

```text
validated Operation Request
        ↓
ArtifactPort.Resolve
        ↓ exact Resolved Artifact Identity
Inspection Run 생성
        ↓
ArtifactPort.Acquire
        ↓ Acquired Artifact binding
VerificationPort.Verify
InspectionPort.Inspect
        ↓ normalized reports
EvidencePort.Record
        ↓ Evidence references
Policy.Evaluate
        ↓ ALLOW / MANUAL_REVIEW / BLOCK
Run finalize
        ↓
Operation Result
```

- `Resolve`가 실패하면 exact identity와 Run이 생성되지 않는다.
- Run은 resolve 성공 후 acquisition 호출 전에 생성한다.
- `inspect` use case constructor에는 Staging/Promotion Port를 받지 않으며 workflow 어느 경로에서도 Promotion을 호출하지 않는다.
- M1 fake는 public network, Host filesystem, credential, external process와 runtime을 사용하지 않는다.

## 2. 최소 Domain type

모든 value type은 constructor/parser에서 공백, 길이, 허용 문자와 required field를 검증한다. raw string과 map을 Domain type 대신 사용하지 않는다.

### Request와 identity

```text
OperationID
RunID
OperationType = INSPECT

ArtifactReference
├─ SourceID
└─ Locator

ResolvedArtifactIdentity
├─ SourceID
├─ Name
├─ Version
└─ Variant

ContentDigest
├─ Algorithm = SHA256
└─ Value = lowercase 64-character hex

AcquiredArtifact
├─ ResolvedArtifactIdentity
├─ ContentDigest
├─ ContentHandle
└─ SizeBytes
```

- M1 `SourceID`는 `fixture` 하나만 fake에서 사용하지만 Domain enum으로 고정하지 않는다. source는 normalized lowercase identifier다.
- `Locator`는 사용자 reference를 보존하는 bounded opaque value이며 resolved identity로 사용하지 않는다.
- `Variant`는 distribution/asset 구분이 필요 없는 fake에서도 명시적 `default` 값을 가진다. 빈 문자열을 wildcard로 해석하지 않는다.
- `ContentHandle`은 trusted controlled-content reference이며 Host path가 아니다. M1 fake에서는 `fixture-content:<id>` 형식의 opaque handle을 사용한다.
- `ResolvedArtifactIdentity + ContentDigest`가 M1 content subject다.

M1 value의 최초 bounded contract는 다음과 같다. 길이는 UTF-8 byte 기준이며 제어 문자, 앞뒤 공백과 invalid UTF-8을 허용하지 않는다.

| Value | Bound / format |
| --- | --- |
| `SourceID` | 1~64, lowercase `[a-z0-9._-]`, 영숫자로 시작·종료 |
| `Locator` | 1~1024 bounded opaque text |
| Name / Version / Variant | 각각 1~256 bounded text |
| `ContentHandle` | 1~512 bounded opaque text |
| Operational failure code | 1~64 uppercase identifier `[A-Z0-9_]`, 영문자로 시작 |

### Identifier 생성

- serialized ID는 `op_` 또는 `run_` prefix와 lowercase base32-no-padding 128-bit payload를 사용한다.
- production 생성은 `crypto/rand.Reader`를 사용하고 실패를 반환한다.
- Domain constructor는 이미 생성된 ID를 검증하며 random source를 package global로 숨기지 않는다.
- Application `Inspect` constructor는 실제 소비자 요구에 따라 `func() (OperationID, error)`와 `func() (RunID, error)`를 주입받아 deterministic test ID를 허용한다. 이를 outbound Port package에 두지 않는다.

## 3. Run과 상태 전이

### Run lifecycle

```text
CREATED → ACTIVE → FINALIZED
```

- `CREATED`: exact identity와 request가 연결됐고 acquisition은 아직 시작하지 않음.
- `ACTIVE`: acquisition 또는 이후 검사 workflow가 진행 중.
- `FINALIZED`: terminal outcome과 확보된 결과가 고정됨.
- 역전이, finalized mutation과 같은 ID 재사용을 거부한다.

Run finalization은 다음 두 형태만 허용한다.

```text
FinalizeCompleted(policy decision)
→ Run Outcome: COMPLETED
→ exactly one Policy Decision required

FinalizeFailed(failure code)
→ Run Outcome: FAILED
→ Policy Decision absent
```

Policy가 생성된 뒤 Run을 `FAILED`로 바꾸거나 operational failure를 `BLOCK`으로 바꾸지 않는다.

### 독립 상태 축

기존 Domain-004의 값을 그대로 사용한다.

```text
Capability       = SUPPORTED | UNSUPPORTED
Execution Status = COMPLETED | FAILED | INCOMPLETE | NOT_EXECUTED | UNAVAILABLE
Run Outcome      = COMPLETED | FAILED
Policy Decision  = ALLOW | MANUAL_REVIEW | BLOCK
Operation Status = COMPLETED | FAILED | PAUSED | NOT_PERFORMED
```

M1 `inspect` mapping은 다음과 같다.

| Condition | Run Outcome | Policy | Operation Status |
| --- | --- | --- | --- |
| policy evaluation 완료 | `COMPLETED` | 세 값 중 하나 | `COMPLETED` |
| resolve 전 operational error | Run 없음 | 없음 | `FAILED` |
| Run 생성 후 unrecoverable operational error | `FAILED` | 없음 | `FAILED` |

`inspect + BLOCK`은 보안 판정까지 검사 operation이 정상 완료됐으므로 Operation Status가 `COMPLETED`다.

## 4. Check report와 Evidence

M1은 Verification과 Inspection 의미를 합치지 않되 공통 실행 envelope를 사용한다.

```text
CheckExecution
├─ CheckID
├─ CheckKind = VERIFICATION | INSPECTION
├─ Required
├─ Capability
├─ ExecutionStatus
└─ LimitationCode (optional)

VerificationReport
├─ CheckExecution
├─ VerificationResults
└─ Evidence

InspectionReport
├─ CheckExecution
├─ Findings
└─ Evidence
```

- `COMPLETED` report는 해석 가능한 result를 가져야 한다.
- `UNSUPPORTED`는 `Capability=UNSUPPORTED`, `ExecutionStatus=NOT_EXECUTED`로 표현한다.
- capability는 있지만 현재 tool/backend가 없으면 `SUPPORTED + UNAVAILABLE`다.
- provider가 normalized failed/incomplete report를 신뢰성 있게 만들 수 있으면 report와 nil Go error를 반환한다.
- malformed response, contract violation 또는 report 자체를 신뢰할 수 없는 provider failure는 Go error이며 Run-level operational failure다.
- Finding이 없다는 사실은 안전을 의미하지 않는다.

M1 Evidence는 bounded normalized fact만 포함한다.

```text
Evidence
├─ EvidenceID
├─ CheckID
├─ Subject identity + digest
├─ Kind
└─ Summary
```

`Summary`는 1 KiB 이하의 sanitized text다. raw tool output, Host path, environment 또는 secret을 포함하지 않는다. `EvidencePort.Record`는 trusted `EvidenceReference`를 반환하며 Policy와 Operation Result는 raw bytes 대신 reference를 사용한다.

## 5. Go boundary 계약

정확한 package identifier는 구현 시 Go naming convention에 맞추되 아래 method 의미와 input/output ownership은 유지한다.

```go
type ArtifactPort interface {
    Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifactIdentity, error)
    Acquire(context.Context, domain.ResolvedArtifactIdentity) (domain.AcquiredArtifact, error)
}

type VerificationPort interface {
    Verify(context.Context, domain.AcquiredArtifact) (domain.VerificationReport, error)
}

type InspectionPort interface {
    Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error)
}

type EvidencePort interface {
    Record(context.Context, domain.RunID, []domain.Evidence) ([]domain.EvidenceReference, error)
}

type Policy interface {
    Evaluate(domain.PolicyInput) (domain.PolicyDecision, error)
}

type InspectUseCase interface {
    Inspect(context.Context, application.InspectRequest) (domain.OperationResult, error)
}
```

- 모든 I/O 경계는 `context.Context`를 첫 인자로 받고 nil context를 허용하지 않는다.
- Port interface는 `internal/core/ports`, Domain type은 `internal/core/domain`, orchestration request/use case는 `internal/application`, Policy 구현은 `internal/policy`가 소유한다.
- Application은 concrete fake나 Cobra를 import하지 않는다.
- M1 fake 구현은 consumer contract test를 공유하고 `internal/testutil` 또는 `_test` package에 둔다. production bootstrap에 fake를 기본 wiring하지 않는다.
- Policy는 I/O를 하지 않는 deterministic Domain service다. 구현 오류에는 error를 반환할 수 있지만 operational input 상태를 error 대신 decision/reason으로 평가한다.

## 6. Application orchestration과 error ownership

`Inspect`는 다음 순서를 바꾸지 않는다.

1. request와 context 검증, Operation ID 생성
2. `Resolve`
3. Run ID 생성과 `CREATED` Run 생성
4. Run을 `ACTIVE`로 전이
5. `Acquire` 후 identity/digest subject binding
6. `Verify`, `Inspect`
7. Evidence를 한 번의 bounded batch로 기록
8. `PolicyInput` 구성과 `Evaluate`
9. completed finalization
10. immutable `OperationResult` 생성

M1은 병렬 실행과 retry를 하지 않는다. 먼저 발생한 operational error를 보존하고 이후 Port를 호출하지 않는다.

```text
OperationResult + error
```

- Go error는 `%w`로 원인을 보존한다.
- resolve 전 error result에는 Operation ID와 `FAILED` status가 있지만 Run ID가 없다.
- Run 생성 후 error result에는 Run ID, `FAILED` outcome과 확보된 exact identity/digest/evidence reference가 있을 수 있다.
- error message는 안정적인 machine code와 sanitized 사람용 message를 분리한다.
- CLI는 partial result를 버리지 않으며 machine output에도 같은 Operation/Run identity를 사용한다.

## 7. M1 Policy v1

Policy identity는 다음으로 고정한다.

```text
Policy ID: m1-fake-inspect
Policy Version: 1
```

평가 우선순위는 deterministic하다.

1. `M1_BLOCK_FINDING`이 있으면 `BLOCK / M1_BLOCK_FINDING`.
2. required check가 unsupported, failed, incomplete, not executed 또는 unavailable이면 `MANUAL_REVIEW / M1_REQUIRED_CHECK_INCOMPLETE`.
3. 모든 required check가 supported+completed이고 blocking finding이 없으면 `ALLOW / M1_REQUIRED_CHECKS_COMPLETED`.

Policy Decision은 decision, policy ID/version과 하나 이상의 ordered reason code를 가진다. M1에서 severity taxonomy, 사용자 override와 suppression은 도입하지 않는다.

## 8. Operation Result와 CLI contract

machine schema identity는 `helox.operation-result/v1`이다. JSON은 stable snake_case field를 사용하고 unknown/newer schema를 consumer가 성공으로 추측하지 않게 한다.

구현 schema는 `schemas/operation-result-v1.schema.json`이 소유하며 Draft 2020-12 metaschema validator로 검증한다. presenter contract test는 synthetic result가 이 field·상태 규칙과 같은 identity를 출력하는지 확인한다.

최소 result는 다음을 포함한다.

```text
schema_version
operation_id
operation
operation_status
run.id (when created)
run.outcome (when finalized)
artifact.reference
artifact.resolved_identity (when resolved)
artifact.digest (when acquired)
checks[]
evidence_references[]
policy.decision / id / version / reasons (when produced)
error.code / message (operational failure only)
```

사람용과 machine-readable presenter는 동일한 immutable `OperationResult`만 입력받는다. raw Evidence 또는 provider output을 기본 출력하지 않는다.

M1 CLI exit contract는 다음과 같다.

| Result | Exit code |
| --- | --- |
| inspect completed + `ALLOW` | `0` |
| operational/application failure | `1` |
| usage/input/schema error | `2` |
| inspect completed + `MANUAL_REVIEW` | `10` |
| inspect completed + `BLOCK` | `20` |

non-interactive/machine mode에서는 prompt를 표시하지 않는다. M1 production root는 fake source command를 사용자-facing command로 노출하지 않는다. CLI presenter와 command adapter는 injected fake use case를 사용한 test에서 검증하며 실제 ecosystem command는 M2에서 연다.

## 9. Synthetic acceptance scenarios

모든 scenario는 고정 ID, identity, digest와 expected result를 사용한다.

| Fixture | Verification | Inspection | Expected |
| --- | --- | --- | --- |
| `safe` | required, supported, completed | required, supported, completed, no blocking finding | Run completed, `ALLOW`, operation completed, exit 0 |
| `review` | completed | required, unsupported + not executed | Run completed, `MANUAL_REVIEW`, operation completed, exit 10 |
| `blocked` | completed | completed + `M1_BLOCK_FINDING` | Run completed, `BLOCK`, operation completed, exit 20 |
| `acquire-error` | not called | not called | Run failed, no Policy, operation failed, exit 1 |
| `resolve-error` | not called | not called | no Run, no Policy, operation failed, exit 1 |

추가 invariant test:

- Run은 resolve 이후 acquire 전에 생성됨.
- identity/digest mismatch binding과 finalized mutation이 거부됨.
- 첫 operational error 뒤 후속 Port가 호출되지 않음.
- `inspect + ALLOW`에서도 Promotion 생성·호출이 없음.
- human/JSON output이 같은 Operation ID, Run ID, identity, digest, decision과 reason을 참조함.
- unsupported/failed/skipped 의미가 빈 success로 변환되지 않음.

## 10. M1 범위 밖

- real ecosystem parsing, network acquisition와 authentication
- filesystem-backed Controlled Intake와 production Evidence Store
- Sandbox/Observation/Dynamic Inspection backend
- dependency graph와 Verified Set
- install, Staging, Promotion
- review persistence/query
- concurrent checks, retry/resume와 Run storage
- security scanner/tool 추가와 suppression

이 항목은 현재 계약을 우회하는 placeholder package/interface/config/job으로 만들지 않는다.
