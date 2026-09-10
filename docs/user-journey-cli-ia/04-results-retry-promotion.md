# User Journey + CLI IA — Results, Retry, and Promotion

## User-Journey-006: 결과 표시와 상세 조회 Journey

### 사용자 원문

Heliopause의 기본 terminal 출력은 내부 검사 과정을 모두 노출하기보다 사용자가 현재 작업의 결과와 다음 행동을 빠르게 이해할 수 있도록 요약 중심으로 제공한다.

상세한 Verification Result, Finding, Evidence, Execution Status, SBOM, Verified Manifest 및 기타 검사 기록은 Inspection Run에 연결하여 보존하고, 사용자가 필요할 때 `review`를 통해 단계적으로 상세 조회할 수 있도록 한다.

기본 원칙은 다음과 같다.

```text
일반 실행
→ 핵심 진행 상태 + 최종 결과

문제·불확실성 발생
→ 핵심 원인 + 다음 행동

상세 분석 필요
→ review <inspection-run>
```

### 기본 terminal 결과

`install`과 `inspect` 실행의 기본 출력에는 최소한 다음 정보가 포함되어야 한다.

- 검사 대상 Artifact의 resolved identity와 version
- 주요 검사 단계의 진행 또는 완료 상태
- 최종 Policy Decision
- 사용자 요청 작업의 최종 상태
- 중요한 경고 또는 주요 Finding
- Inspection Run 식별 정보

일반적인 `ALLOW` 결과에서는 정상적인 세부 Evidence와 모든 Finding을 전부 출력하지 않고, 작업 성공 여부를 빠르게 확인할 수 있는 수준으로 요약한다.

```text
$ helox npm install kenv

Inspecting kenv@<resolved-version>...
✓ Verification
✓ Static Inspection
- Dynamic Inspection: Not required
✓ Policy: ALLOW
✓ Installation completed

Run: <inspection-run>
```

실제 progress UI, symbol, 색상 및 문구는 후속 CLI presentation 설계에서 결정한다.

### `ALLOW` 결과 표시

`install` 요청이 `ALLOW`로 완료된 경우 사용자가 최소한 다음을 확인할 수 있어야 한다.

```text
Artifact
→ 정확히 무엇이 설치되었는가

Policy
→ ALLOW

Operation
→ 설치·반입이 실제 완료되었는가

Inspection Run
→ 필요할 때 어떤 검사 기록을 조회하면 되는가
```

중요한 경고 수준의 Finding이 존재하지만 Policy가 `ALLOW`인 경우에는 성공 결과와 함께 해당 경고의 존재를 표시한다.

정상적인 세부 Evidence, dependency 전체 목록, scanner 원본 출력 등을 기본 terminal 출력에 모두 노출하지 않는다.

### `MANUAL_REVIEW` 결과 표시

`MANUAL_REVIEW`에서는 단순히 상태 이름만 출력하지 않고 사용자가 왜 자동 작업이 보류되었는지 이해할 수 있는 핵심 정보를 함께 제공한다.

최소한 다음을 표시한다.

- `MANUAL_REVIEW` 상태
- install/반입이 보류되었다는 사실
- 자동 판단을 완료하지 못한 주요 이유
- 중요한 Finding 또는 검사 실패·미완료 상태
- 필요한 경우 검사 한계
- Inspection Run 식별자
- 상세 결과를 확인할 수 있는 `review` 경로

```text
Policy: MANUAL_REVIEW
Installation paused.

Reason:
- Required dynamic inspection could not be completed.

Run: <inspection-run>
Review: helox review <inspection-run>
```

기본 화면에서 모든 Evidence를 출력하지 않고 사용자가 판단을 시작하는 데 필요한 핵심 정보만 제공한다.

### `BLOCK` 결과 표시

`BLOCK`에서는 설치·반입이 차단되었다는 사실과 핵심 차단 이유를 명확하게 표시한다.

최소한 다음을 포함한다.

- `BLOCK` 상태
- 원래 요청한 install/반입이 수행되지 않았다는 사실
- 주요 차단 Finding
- 필요한 경우 위험 행동 또는 검증 실패 요약
- Inspection Run 식별자
- 상세 조회 방법

```text
Policy: BLOCK
Installation blocked.

Key findings:
- Credential access attempt detected
- Unexpected outbound connection attempted

Run: <inspection-run>
Review: helox review <inspection-run>
```

차단된 위험 Artifact나 Sandbox 실행 환경을 정리하더라도 판단 근거가 되는 Evidence와 Inspection Run 기록은 별도 retention 정책에 따라 보존한다.

### `inspect` 결과 표시

`inspect`는 설치를 수행하지 않으므로 최종 결과에서 검사 완료와 Policy Decision을 명확하게 구분한다.

```text
Policy: ALLOW
Inspection completed.
No artifact was installed.

Run: <inspection-run>
```

사용자가 `ALLOW`를 `설치 완료`로 오해하지 않도록 최초 operation의 결과를 함께 표시한다.

### `review` 상세 조회

`review`는 기존 Inspection Run의 결과를 자세히 확인하기 위한 공통 조회 진입점으로 사용한다.

```text
helox review <inspection-run>
```

기본 `review` 화면은 해당 Inspection Run의 종합 summary를 제공한다.

해당 Inspection Run에서 생성되었거나 적용 가능한 경우 다음 항목을 추적·조회할 수 있어야 하며, 생성되지 않았거나 적용되지 않은 항목은 그 상태와 사유를 확인할 수 있어야 한다.

- Artifact identity, version, source, digest
- Primary Artifact와 관련 dependency
- Policy Decision
- Verification Result
- Finding
- Evidence
- Capability
- Execution Status
- 수행·미수행 검사 및 사유
- 검사 한계
- SBOM
- Verified Set / Verified Manifest
- Sandbox 또는 검사 환경 정보
- 최종 install/import 상태

예를 들어 acquisition 실패 또는 `BLOCK`으로 Verified Manifest가 생성되지 않은 경우 `Verified Manifest: Not generated — Policy: BLOCK`과 같이 그 상태와 사유를 표시한다.

세부 정보는 한 화면에 모두 펼치기보다 summary에서 시작하여 필요한 영역을 선택적으로 자세히 조회할 수 있도록 한다.

```text
review <run>
      ↓
Summary
      ├─ Verification
      ├─ Findings
      ├─ Evidence
      ├─ Dependencies / SBOM
      ├─ Execution Status
      └─ Verified Manifest
```

이를 실제 subcommand, flag 또는 다른 navigation 방식 중 무엇으로 구현할지는 후속 CLI IA에서 결정한다.

### Raw Evidence와 외부 도구 출력

일반 사용자가 `review`를 실행했다고 해서 scanner, verifier 또는 Sandbox의 대용량 Raw Evidence를 전부 즉시 출력하지 않는다.

정규화된 결과와 주요 Evidence를 우선 제공하고, 원본 자료가 필요한 경우 별도의 상세 조회 또는 machine-readable 결과를 통해 접근할 수 있도록 한다.

```text
기본 terminal
→ 사용자 작업 결과

review summary
→ 판단에 필요한 정규화 결과

detailed Evidence
→ 기술적 조사

Raw Evidence
→ 심층 분석·감사·재현
```

### Inspection Run 식별자

`install`, `inspect` 및 검사 결과를 생성하는 주요 작업은 사용자가 이후 해당 결과를 다시 찾을 수 있도록 Inspection Run 식별자를 제공한다.

Inspection Run ID의 구체적인 형식은 Domain Model에서 결정한다.

사용자는 Artifact 이름이나 terminal 출력에 의존하지 않고 Inspection Run을 기준으로 동일한 검사 기록을 다시 조회할 수 있어야 한다.

### 사람용 결과와 machine-readable 결과

사람이 terminal에서 읽는 결과와 CI·automation에서 사용하는 구조화 결과는 동일한 Inspection Run, Artifact identity/digest 및 Policy Decision을 참조해야 한다.

두 결과가 서로 다른 판정이나 Artifact를 나타내어서는 안 된다.

구체적인 JSON schema, output option, file format 및 machine-readable interface는 후속 Domain Model / CLI 설계에서 결정한다.

### 기본 출력 최소화 원칙

Heliopause는 정상적인 실행에서 다음과 같은 정보를 모두 기본 화면에 출력하지 않는다.

- 전체 dependency tree
- 모든 Raw Evidence
- scanner/verifier 원본 출력
- 전체 SBOM
- 모든 정상 검사 항목
- 내부 Sandbox telemetry
- 내부 Policy 평가 과정 전체

필요한 정보는 Inspection Run에 연결하여 보존하고 `review` 또는 구조화 출력에서 조회할 수 있도록 한다.

따라서 기본 UX는 다음을 지향한다.

```text
정상일수록 짧게
문제가 있을수록 핵심 이유를 명확하게
상세 정보는 필요할 때 깊게
```

### 후속 결정으로 유보한 항목

- terminal 색상·icon·progress UI
- 한 화면에 표시할 Finding 개수
- severity별 표시 방식
- `review`의 정확한 subcommand와 flag
- Raw Evidence 조회 syntax
- JSON 등 machine-readable output schema
- SBOM 출력 format
- Inspection Run ID 형식
- 결과 저장 위치
- pagination / filtering / sorting
- terminal verbosity option
- CI 출력 방식

위 항목은 후속 CLI IA, Domain Model, Evidence/Result 및 Tooling 설계에서 확정한다.

### 압축된 사용자 모델

```text
helox npm install kenv
        ↓
짧은 진행 상태
        ↓
최종 Policy + 실제 작업 결과
        ↓
Inspection Run ID

문제가 있거나 더 자세히 보고 싶음
        ↓
helox review <run-id>
        ↓
Summary
        ↓
필요한 Finding / Evidence / Verification / SBOM 등 상세 조회
```

Heliopause는 내부적으로 많은 보안 정보를 생성하더라도 일반 사용자의 기본 workflow를 복잡하게 만들지 않고, 필요한 만큼만 단계적으로 정보를 노출하는 것을 기본 원칙으로 한다.

### 구조화된 결정과 구현 영향

- 기본 terminal은 내부 전체 pipeline이 아니라 resolved Artifact identity/version, 핵심 진행 상태, 최종 Policy, 실제 작업 상태, 중요 경고/Finding, Inspection Run ID를 요약한다.
- `ALLOW` install 결과는 무엇이 설치되었는지, `ALLOW`인지, 실제 설치·반입이 완료되었는지, 어떤 Run을 review하는지 표시한다.
- `MANUAL_REVIEW`는 보류 사실, 자동 판단 불가 이유, 주요 Finding·실패·미완료·검사 한계, Run ID 및 review 경로를 표시한다.
- `BLOCK`은 반입 미수행, 핵심 차단 Finding·위험 행동·검증 실패, Run ID 및 상세 조회 방법을 표시한다.
- `inspect` 결과는 `ALLOW`여도 설치되지 않았음을 명시하여 검사 완료와 설치 완료를 혼동하지 않게 한다.
- `review <inspection-run>`은 Summary를 시작점으로 Verification, Findings, Evidence, dependency/SBOM, Execution Status, Verified Manifest를 단계적으로 조회한다.
- 대용량 Raw Evidence와 외부 도구 원본 출력은 기본 terminal이나 review summary에 모두 펼치지 않고 상세 조회·machine-readable 결과로 제공한다.
- `install`·`inspect`·주요 검사 작업은 Inspection Run ID를 제공하고 모든 사람용·기계용 결과가 동일 Run, identity/digest, Policy Decision을 참조하게 한다.
- 전체 dependency tree·Raw Evidence·원본 scanner 출력·SBOM·Sandbox telemetry·Policy 내부 평가 과정은 기본 출력에서 제외하고 Run에 보존한다.

## User-Journey-006 누락 점검

- [x] 기본 terminal의 요약 중심 출력
- [x] 핵심 진행 상태·최종 결과·다음 행동 제공
- [x] resolved Artifact identity/version 표시
- [x] `ALLOW` 결과의 실제 설치·반입 완료 상태 표시
- [x] `MANUAL_REVIEW` 보류 이유·Finding·한계·review 경로 표시
- [x] `BLOCK` 차단 사실·핵심 Finding·상세 조회 방법 표시
- [x] `inspect` 설치 없음 명시
- [x] `review <inspection-run>` 공통 상세 조회 진입점
- [x] Verification·Finding·Evidence·Capability·Execution Status 조회
- [x] 수행·미수행 검사와 사유·검사 한계 조회
- [x] SBOM·Verified Set·Verified Manifest·Sandbox 정보·최종 상태 조회
- [x] Raw Evidence 및 외부 도구 원본 출력의 단계적 접근
- [x] Inspection Run ID 제공 및 재조회 기준
- [x] 사람용·machine-readable 결과의 동일 Run·identity/digest·Policy 참조
- [x] 기본 출력 최소화 및 상세 정보 보존
- [x] presentation·schema·format·pagination·CI 출력 후속 결정 유보

## User-Journey-007: 실패·재검사·재개 Journey

### 사용자 원문

Heliopause는 검사 workflow의 실패·중단·미완료 상태를 정상적인 검사 완료와 명확하게 구분하며, 기존 검사 결과를 임의로 성공 상태로 간주하거나 안전하지 않은 실행 상태를 그대로 재사용하지 않는다.

기본 원칙은 다음과 같다.

```text
일시적·재시도 가능한 실패
→ 제한된 범위에서 자동 retry 가능

보안상 필요한 검사가 미완료
→ 안전한 것으로 간주하지 않음
→ MANUAL_REVIEW 또는 BLOCK

완료된 Inspection Run 이후 재검사
→ 기존 기록을 덮어쓰지 않음
→ 새로운 Inspection Run으로 수행

비정상 종료된 Sandbox Session
→ 그대로 resume/reuse하지 않음
→ 폐기 후 새로운 Session에서 실행
```

### 실행 중 일시적 실패와 자동 retry

검사 workflow 도중 외부 도구 오류, 일시적인 프로세스 실패 또는 기타 재시도 가능한 문제가 발생할 수 있다.

Heliopause는 해당 문제가 보안 경계를 약화시키지 않고 동일한 Artifact와 동일한 검사 조건에서 안전하게 다시 수행할 수 있는 경우 제한된 횟수의 자동 retry를 수행할 수 있다.

```text
검사 단계 실행
     ↓
일시적 실패
     ↓
안전하게 retry 가능한가?
     ├─ Yes → 제한된 자동 retry
     └─ No  → 실패·미완료 상태 기록
```

retry를 수행하더라도 최초 실패 사실을 숨기지 않으며, 필요한 경우 각 실행 attempt와 최종 Execution Status를 Inspection Run에 기록한다.

구체적인 retry 횟수, backoff, retry 가능한 오류 종류 및 외부 도구별 정책은 후속 Tooling/Inspection 설계에서 정의한다.

### 보안상 필수 검사 실패

Policy가 안전한 자동 판정을 위해 필요로 하는 Verification 또는 Inspection이 실패·미완료·사용 불가 상태가 된 경우 Heliopause는 해당 Artifact를 임의로 안전하다고 간주하지 않는다.

```text
필수 검사
   ↓
실패 / 미완료 / Unavailable
   ↓
ALLOW로 간주하지 않음
   ↓
fail-closed
   ↓
MANUAL_REVIEW 또는 BLOCK
```

어떤 상태가 `MANUAL_REVIEW`이고 어떤 상태가 `BLOCK`인지에 대한 구체적인 rule은 후속 Policy 설계에서 결정한다.

### Sandbox 실패와 재실행

Sandbox/Dynamic Inspection 도중 timeout, resource limit 초과, crash, 강제 종료 또는 기타 비정상 상태가 발생한 경우 해당 Sandbox Session을 이어서 사용하거나 다음 검사에 재사용하지 않는다.

```text
Sandbox Session
      ↓
비정상 종료 / timeout / resource limit / failure
      ↓
관찰 가능한 Evidence 수집
      ↓
Session 종료·폐기
      ↓
필요한 경우 새로운 Sandbox Session 생성
      ↓
재검사
```

새로운 Sandbox Session은 이전 비정상 실행 상태를 신뢰하거나 그대로 이어받지 않는다. 이는 오염되었거나 불완전한 격리 환경을 재사용하여 후속 검사 결과에 영향을 주는 것을 방지하기 위함이다.

### 실행 자체가 중단된 경우

사용자 interrupt, Heliopause process 종료, 시스템 오류 또는 기타 이유로 전체 workflow가 중단될 수 있다.

가능한 범위에서 이미 생성된 Inspection Run 정보, 수행 완료 단계, Execution Status 및 확보된 Evidence는 보존한다.

그러나 단순히 이전 실행이 어느 단계까지 진행되었다는 이유만으로 모든 내부 실행 상태를 그대로 복원하여 계속하지는 않는다.

재개 시에는 기존 결과의 무결성과 재사용 가능성을 확인하고, 안전하게 재사용할 수 없는 단계는 다시 수행한다. 특히 실행 중이던 Sandbox Session과 같은 ephemeral execution state는 resume 대상으로 간주하지 않는다.

```text
workflow 중단
      ↓
기존 Run / Evidence 보존
      ↓
재사용 가능한 결과 확인
      ↓
      ├─ 안전하게 재사용 가능 → 활용 가능
      └─ 재사용 불확실         → 해당 검사 다시 수행
```

어떤 검사 결과를 안전하게 재사용할 수 있는지에 대한 구체적인 validity 조건과 checkpoint 방식은 후속 Domain Model, Evidence/Result 및 Inspection 설계에서 정의한다.

### 완료된 Inspection Run의 불변성

최종 상태가 기록된 Inspection Run은 이후 재검사 결과로 덮어쓰지 않는다.

예를 들어 최초 검사에서:

```text
Run A
→ MANUAL_REVIEW
```

가 발생하고 이후 문제를 해결하거나 추가 검사를 수행하더라도 기존 Run A의 결과를 사후에 `ALLOW`로 수정하여 과거 상태를 없애지 않는다.

재검사가 필요한 경우 새로운 Inspection Run 또는 기존 Run과 명시적으로 연결된 후속 Run을 생성한다.

```text
Run A
Policy: MANUAL_REVIEW
        ↓
추가 검사 / 재검사
        ↓
Run B
Policy: ALLOW
```

Run B는 필요한 경우 Run A와의 관계를 기록할 수 있지만 두 Run의 Evidence, Execution Status 및 Policy Decision은 서로 구분한다. Inspection Run 간 관계를 표현하는 정확한 Domain Model은 후속 단계에서 정의한다.

### 동일 Artifact의 재검사

사용자는 이전에 검사한 Artifact를 다시 검사할 수 있다.

이 경우 Heliopause는 단순히 Artifact 이름이 같다는 이유로 과거 Policy Decision을 그대로 적용하지 않는다.

재검사에서 기존 결과를 활용하려면 최소한 실제 검사 대상의 exact identity와 digest가 일치해야 하며, 필요한 경우 검사 도구, Policy, Capability, 검사 환경 또는 기존 Evidence의 유효성도 고려해야 한다.

```text
이전 Artifact
name: kenv
version: X
digest: AAA

새 검사 대상
name: kenv
version: X
digest: AAA
```

동일성이 확인되더라도 과거 결과의 재사용 조건은 별도로 검증되어야 한다. 구체적인 cache/reuse 정책은 후속 설계에서 결정하며, 과거 `ALLOW`가 새로운 실행의 안전성을 자동 보장하지 않는다는 원칙만 고정한다.

### Artifact가 변경된 경우

다음과 같이 검사 대상의 실제 identity가 변경되면 기존 Inspection Run의 Policy Decision을 새 대상에 그대로 적용하지 않는다.

- version 변경
- release/tag 변경
- GitHub Release asset 변경
- digest 변경
- dependency 또는 Verified Set 변경
- 동일 version이라도 실제 Artifact bytes가 변경된 경우

이 경우 새로운 Artifact 상태로 취급하여 필요한 verification/inspection을 수행한다.

특히 mutable reference인 `latest` 등을 다시 사용했을 때 이전과 다른 exact version 또는 digest로 resolve되었다면 이전 Inspection Run을 현재 Artifact의 안전 판정으로 사용하지 않는다.

```text
latest
  ↓
어제 → 1.2.0 / digest AAA
오늘 → 1.3.0 / digest BBB

→ 서로 다른 검사 대상
→ 새로운 Inspection Run 필요
```

### `MANUAL_REVIEW` 이후 재검사

`MANUAL_REVIEW` 상태에서는 기존 설치 요청을 자동으로 계속하지 않는다.

사용자는 `review`를 통해 판단 근거를 확인하고, 필요한 경우 문제의 원인이 해결된 상태에서 추가 검사 또는 재검사를 수행할 수 있다.

```text
MANUAL_REVIEW
      ↓
review
      ↓
원인 확인
      ↓
추가 정보 / 환경 변화 / 재검사 필요
      ↓
새로운 검사 수행
      ↓
새 Policy Decision
```

새로운 검사 결과가 `ALLOW`가 되었다면 해당 `ALLOW`와 연결된 검증 결과, Artifact identity/digest 및 Verified Set을 기준으로 후속 설치·반입 여부를 결정한다.

기존 `MANUAL_REVIEW` Run 자체를 단순히 사용자 confirmation만으로 `ALLOW`로 변경하지 않는다. 향후 별도의 approval/override 기능을 도입할 경우에는 이 재검사 경로와 구분하여 명시적인 권한·감사 기록·Policy 의미를 갖도록 설계한다.

### `BLOCK` 이후 재검사

`BLOCK`은 해당 Inspection Run과 해당 Artifact 상태에 대한 설치·반입 금지 Decision이다.

사용자는 일반 confirmation으로 해당 Run의 `BLOCK`을 우회할 수 없다.

다만 이후 검사 대상 자체가 달라진 경우에는 새로운 검사 대상으로 다시 검사할 수 있다.

```text
package@1.2.3
→ BLOCK

새 version 1.2.4 공개
        ↓
새 Artifact identity / digest
        ↓
새로운 Inspection Run
```

동일 Artifact에 대한 재검사를 허용할지 여부와 그 조건은 후속 Policy에서 구체화할 수 있으나, 새로운 재검사 결과가 기존 `BLOCK` 기록을 삭제하거나 수정하지 않는다.

### 재검사와 Promotion의 연결

Promotion은 항상 현재 반입하려는 Artifact와 연결된 유효한 `ALLOW` Policy Decision을 요구한다.

과거의 다른 Inspection Run에서 생성된 `ALLOW`, 다른 digest의 Verified Set 또는 현재 Artifact와 연결되지 않은 Verified Manifest를 조합하여 Promotion하지 않는다.

```text
현재 Promotion 대상
        ↓
정확한 Artifact identity / digest
        ↓
해당 검증 결과
        ↓
해당 ALLOW Decision
        ↓
Verified Set / Manifest
        ↓
서로 일관되게 연결됨
        ↓
Promotion 가능
```

재검사 결과 새로운 `ALLOW`가 생성된 경우 최종 설치·반입은 새로운 검사 결과를 기준으로 수행한다.

### 사용자 관점의 재시도·재검사

일반 사용자가 내부 workflow 단계별로 직접 `verify`, `scan`, `sandbox` 등을 재실행하도록 요구하지 않는다.

Heliopause는 가능한 retry를 내부적으로 처리하고, 사용자 개입이 필요한 경우에는 작업 상태와 이유를 명확하게 제공한다.

```text
정상적 일시 오류
→ Heliopause가 가능한 범위에서 자동 retry

해결되지 않음
→ 작업 중단 / MANUAL_REVIEW 또는 BLOCK
→ Run ID 제공

사용자가 원인 확인
→ 필요하면 Artifact를 다시 검사
→ 새로운 결과 확인
```

재검사를 위한 정확한 CLI command, 기존 Run을 참조하는 syntax, `retry` 또는 `resume` command를 별도로 제공할지 여부는 후속 CLI IA에서 결정한다.

### 후속 결정으로 유보한 항목

- 자동 retry 횟수와 backoff
- retry 가능한 오류 분류
- 별도 `retry` / `resume` command 제공 여부
- 재검사 CLI syntax
- checkpoint 저장 방식
- 완료된 검사 결과의 cache/reuse 조건
- 이전 Evidence의 validity 기간
- tool/version/Policy 변경 시 재검사 조건
- Inspection Run 간 parent/retry/supersede 관계 schema
- `MANUAL_REVIEW` approval/override
- 동일한 `BLOCK` Artifact의 재검사 제한
- 재검사 후 원래 install 요청을 자동 재개할지 여부

위 항목은 후속 CLI IA, Domain Model, Policy, Evidence/Result 및 Tooling 단계에서 구체화한다.

### 압축된 사용자 모델

```text
검사 시작
   ↓
일시적인 오류
   ├─ 안전하게 retry 가능 → 제한된 자동 retry
   └─ 해결 불가 → 실패 상태 기록

보안상 필수 검사 미완료
   ↓
ALLOW 금지
   ↓
MANUAL_REVIEW / BLOCK

기존 Run 이후 다시 검사
   ↓
기존 기록 유지
   ↓
새 Inspection Run
   ↓
새 Policy Decision

Artifact identity / digest 변경
   ↓
기존 판정 재사용 금지
   ↓
새 검사 필요
```

Heliopause는 실패 상태를 숨기거나 과거 검사 결과를 편의상 무조건 재사용하지 않고, 재현 가능성과 감사 가능성을 유지하면서 안전하게 재시도·재검사할 수 있도록 하는 것을 기본 원칙으로 한다.

### 구조화된 결정과 구현 영향

- 일시적이고 보안 경계를 약화시키지 않는 실패만 제한된 자동 retry 대상으로 삼고 최초 실패·attempt·최종 Execution Status를 기록한다.
- 보안상 필수 Verification/Inspection의 실패·미완료·Unavailable은 `ALLOW`로 해석하지 않고 fail-closed로 `MANUAL_REVIEW` 또는 `BLOCK`에 연결한다.
- 비정상 Sandbox Session은 Evidence를 수집한 뒤 폐기하고 새로운 Session에서 재실행한다.
- 중단된 workflow의 Run·완료 단계·Execution Status·Evidence는 보존하되 ephemeral execution state를 그대로 resume하지 않는다.
- 완료된 Run은 불변으로 유지하며 후속 검사에는 새로운 Run을 만들고 Run 간 관계만 명시적으로 기록한다.
- 과거 결과 재사용은 exact identity·digest뿐 아니라 tool·Policy·Capability·환경·Evidence 유효성을 고려하며 과거 `ALLOW`를 자동 안전 보장으로 사용하지 않는다.
- version·release/tag·asset·digest·dependency·Verified Set·Artifact bytes가 변경되면 새 검사 대상으로 취급한다.
- `MANUAL_REVIEW`는 review와 새 검사로 이어지며 기존 Run을 confirmation만으로 `ALLOW`로 변경하지 않는다.
- `BLOCK`은 일반 confirmation으로 우회하지 않는다. Artifact가 변경된 경우에는 새로운 Run으로 검사하며, 동일 Artifact의 재검사 허용 여부와 조건은 후속 Policy에서 결정한다.
- Promotion은 현재 대상과 동일한 identity/digest에 연결된 유효한 `ALLOW`·Verified Set·Manifest만 사용한다.
- 일반 사용자는 내부 단계 재실행을 직접 조작하지 않고 Heliopause가 retry와 재검사 필요 상태를 안내한다.

## User-Journey-007 누락 점검

- [x] 일시적 실패의 제한된 자동 retry
- [x] 최초 실패·attempt·최종 Execution Status 기록
- [x] 필수 검사 실패·미완료·Unavailable의 fail-closed 처리
- [x] 비정상 Sandbox Session 관찰·폐기·새 Session 재실행
- [x] 중단된 Run·Evidence 보존과 ephemeral state 재사용 금지
- [x] 완료된 Inspection Run 불변성
- [x] 후속 검사에서 새 Inspection Run 생성
- [x] 동일 Artifact 재검사 시 identity/digest 및 유효성 확인
- [x] version·release/tag·asset·digest·dependency·bytes 변경 시 판정 재사용 금지
- [x] latest 등 mutable reference 변경 시 새 Run
- [x] MANUAL_REVIEW의 review·재검사 경로
- [x] 기존 MANUAL_REVIEW Run의 confirmation ALLOW 변경 금지
- [x] BLOCK의 일반 우회 금지와 새 Artifact 재검사
- [x] Promotion의 현재 Artifact·ALLOW·Verified Set·Manifest 일관성
- [x] 사용자에게 내부 retry·재검사 단계를 직접 조작시키지 않음
- [x] retry·resume·cache·checkpoint·Run 관계·approval 후속 결정 유보

## User-Journey-008: 설치·반입(Promotion) UX

### 사용자 원문

Heliopause에서 Promotion은 검사를 통과한 Artifact를 실제 사용자 환경으로 반입하기 위한 신뢰 경계이며, 일반 사용자가 별도의 내부 단계로 직접 조작하는 작업으로 노출하지 않는다.

사용자가 처음부터 `install` 또는 이에 해당하는 설치·반입 작업을 요청한 경우 최종 Policy Decision이 `ALLOW`이면 Heliopause가 원래 요청의 실행 context를 유지한 상태에서 안전한 Promotion 절차를 수행하고 원래 작업을 완료한다.

기본 원칙은 다음과 같다.

```text
사용자 install 요청
        ↓
검사 workflow
        ↓
Policy = ALLOW
        ↓
검사한 Artifact / Verified Set 확인
        ↓
identity / digest 재검증
        ↓
trusted Promotion
        ↓
원래 요청한 사용자 환경으로 설치·반입
```

`MANUAL_REVIEW` 또는 `BLOCK` 상태에서는 이 Promotion 경계로 진행하지 않는다.

### 원래 install 요청의 유지

Heliopause는 검사 이전에 사용자가 요청한 설치 작업의 의미를 유지한다.

예를 들어 다음과 같은 정보가 해당될 수 있다.

- ecosystem/source
- operation
- Primary Artifact
- resolved version/release/asset
- 프로젝트 또는 설치 대상 환경
- local/global 등 설치 범위
- 사용자가 명시한 허용 가능한 option
- standalone Artifact의 대상 위치

따라서 사용자가:

```text
helox npm install kenv
```

를 요청했다면 `ALLOW` 이후 임의의 다른 위치나 다른 방식으로 Artifact를 반입하는 것이 아니라 원래 요청과 연결된 설치 작업을 계속 수행한다.

원본 package manager option의 지원 범위와 각 option이 검사 결과 및 Promotion에 미치는 영향은 후속 ecosystem별 CLI contract에서 정의한다.

### 검사 대상과 실제 반입 대상의 동일성

Promotion에서 가장 중요한 원칙은 실제로 사용자 환경으로 반입되는 Artifact가 검사·검증된 Artifact와 정확히 연결되어 있어야 한다는 것이다.

Heliopause는 단순히 package 이름이나 version 문자열이 같다는 이유만으로 새로운 Artifact를 다시 외부 source에서 받아 설치하지 않는다.

Package ecosystem에서는 최종적으로 사용할 Primary Artifact와 dependency의 정확한 집합을 Verified Set으로 관리한다.

```text
Inspection Run
      ↓
Primary Artifact + dependencies
      ↓
exact identity / digest
      ↓
Verified Set
      ↓
Verified Manifest
      ↓
Policy = ALLOW
      ↓
Promotion
```

Promotion에 사용되는 Artifact와 dependency는 해당 Verified Set / Verified Manifest 및 관련 Inspection Run에 연결되어 있어야 한다. 다른 Run의 Decision, 다른 digest의 Artifact 또는 검증되지 않은 dependency를 임의로 섞어 설치하지 않는다.

### Package ecosystem의 Promotion

npm 및 PyPI/pip와 같이 dependency를 포함하는 Package ecosystem에서는 일반적으로 다음 경로를 사용한다.

```text
Quarantine / Inspection
        ↓
Policy = ALLOW
        ↓
Verified Set / Verified Manifest 확정
        ↓
Staging 반입 전 identity / digest 재검증
        ↓
Staging
        ↓
사용자 환경 반입 전 identity / digest 재검증
        ↓
trusted Promotion
        ↓
사용자 환경 설치
```

Staging은 사용자 환경에 직접 노출되기 전의 검증 완료 Artifact 보관·전달 영역이며, 새로운 검사를 수행하거나 Artifact를 실행하는 두 번째 Sandbox로 사용하지 않는다.

가능한 경우 실제 사용자 환경 설치는 외부 registry에서 Artifact를 다시 자유롭게 다운로드하는 방식보다 Heliopause가 검증한 local Verified Set을 사용하여 수행한다.

```text
검사한 것
=
실제로 설치하는 것
```

### 설치 과정에서 새로운 Artifact가 필요한 경우

최종 사용자 환경 설치 과정에서 기존 Verified Set에 존재하지 않는 새로운 dependency 또는 Artifact가 요구되면 Heliopause는 이를 자동으로 외부에서 받아 사용자 환경에 설치하지 않는다.

```text
Promotion / install 진행
        ↓
Verified Set에 없는 Artifact 필요
        ↓
자동 신뢰 금지
        ↓
Promotion 중단
        ↓
Artifact acquisition / Verification으로 회귀
        ↓
필요한 검사 수행
        ↓
새로운 유효한 Verified Set / Policy 결과 확보
```

package manager의 dependency resolution이 검사 단계와 실제 설치 단계 사이에서 달라지더라도 새로운 구성요소가 보안 경계를 우회하여 사용자 환경으로 유입되지 않아야 한다. 정확한 dependency locking, offline install 및 package manager별 enforcement 방식은 후속 ecosystem/Tooling 설계에서 정의한다.

### Promotion 전 재검증

Promotion 직전에는 실제 반입하려는 Artifact가 검사·검증된 대상과 동일한지 확인한다.

최소한 identity와 digest의 연결을 확인하며, package ecosystem에서는 Verified Set / Verified Manifest와의 일관성도 확인한다.

```text
검증된 digest
      ↓
Promotion 직전 실제 Artifact digest
      ↓
      ├─ 일치   → Promotion 계속
      └─ 불일치 → Promotion 중단
```

불일치가 발생하면 기존 `ALLOW`가 존재한다는 이유만으로 반입을 계속하지 않는다. 필요한 acquisition/Verification/Inspection 단계로 돌아가 새 상태를 기준으로 판단한다.

### Standalone binary/archive의 직접 반입

Package manager가 필요 없는 standalone binary/archive는 Architecture-008에 정의된 예외에 따라 반드시 별도의 Staging 단계를 거치지 않을 수 있다.

이 경우에도 Sandbox나 검사 환경이 사용자 Host에 직접 파일을 작성해서는 안 된다.

```text
Standalone binary/archive
        ↓
Inspection
        ↓
Policy = ALLOW
        ↓
identity / digest 재검증
        ↓
trusted Promotion
        ↓
사용자 지정 위치로 직접 반입 가능
```

Staging 생략은 Promotion 신뢰 경계를 생략한다는 의미가 아니다. 실제 사용자 환경으로의 쓰기 작업은 Heliopause의 trusted Promotion 책임을 통해서만 수행한다.

### 설치 대상이 모호하거나 충돌하는 경우

Promotion을 안전하게 수행하기 위한 사용자 환경의 대상 위치 또는 설치 context가 명확하지 않은 경우 Heliopause는 임의로 중요한 값을 추측하여 반입하지 않는다.

예를 들어 기존 파일 덮어쓰기, 서로 다른 설치 범위 또는 의미가 달라지는 대상 선택처럼 사용자의 의도가 추가로 필요한 상황에서는 설치를 진행하기 전에 명확한 입력을 요구할 수 있다.

다만 정상적인 `ALLOW` install마다 반복적으로 별도의 Promotion confirmation을 요구하지 않는다.

```text
정상적이고 원래 요청이 명확함
→ 추가 confirmation 없이 계속

Promotion 대상이나 의미가 모호함
→ 필요한 사용자 입력 요구
```

구체적인 overwrite 정책, confirmation 조건 및 filesystem conflict 처리는 후속 CLI/Promotion 설계에서 결정한다.

### Promotion 자체의 실패

`Policy = ALLOW`는 Artifact가 Heliopause의 검사 정책을 통과했다는 의미이지 실제 사용자 환경 설치가 반드시 성공했다는 의미는 아니다.

예를 들어 filesystem 오류, 권한 문제, package manager 오류 또는 대상 환경 문제로 실제 설치 작업이 실패할 수 있다.

따라서 Heliopause는 다음 두 상태를 구분한다.

```text
Policy Decision
→ Artifact에 대한 보안 판정

Operation Status
→ 사용자가 요청한 실제 install/import 작업 결과
```

예를 들어 다음과 같은 상태가 가능하다.

```text
Policy: ALLOW
Installation: FAILED
Reason: target permission denied
```

일반적인 설치 작업 실패 자체가 기존 `ALLOW` Policy Decision을 자동으로 `BLOCK`으로 바꾸지는 않는다. 다만 설치 중 검증되지 않은 Artifact 요구, digest mismatch 또는 기타 보안 invariant 위반이 발견되면 Promotion을 중단하고 필요한 verification/inspection 경로로 되돌린다.

구체적인 Operation Status와 failure taxonomy는 후속 Domain Model에서 정의한다.

### `inspect`와 Promotion

사용자가 `inspect`를 요청한 경우에는 최종 Policy Decision이 `ALLOW`여도 Promotion을 수행하지 않는다.

```text
helox npm inspect kenv
        ↓
Policy = ALLOW
        ↓
검사 결과 기록
        ↓
Promotion 없음
        ↓
종료
```

Promotion 여부는 Policy Decision만으로 결정되는 것이 아니라 사용자가 처음 요청한 operation과 Policy Decision을 함께 기준으로 한다.

```text
install + ALLOW
→ Promotion 가능

inspect + ALLOW
→ Promotion하지 않음

MANUAL_REVIEW
→ Promotion 보류

BLOCK
→ Promotion 금지
```

### Promotion 결과 기록

실제 설치·반입을 수행한 경우 해당 결과는 원래 Inspection Run과 검증된 Artifact identity/digest에 추적 가능하게 연결되어야 한다.

최소한 다음 사실을 확인할 수 있어야 한다.

- 어떤 Inspection Run을 근거로 했는가
- 어떤 Policy Decision을 사용했는가
- 어떤 Verified Set / Manifest를 사용했는가
- 어떤 Artifact identity/digest를 실제 반입했는가
- 어떤 사용자 operation을 수행했는가
- 실제 설치·반입이 성공했는가

구체적인 Promotion Record 또는 Operation Result schema는 후속 Domain Model / Evidence/Result 설계에서 정의한다.

### 후속 결정으로 유보한 항목

- npm/pip의 실제 offline/local install 구현 방식
- package manager별 dependency locking 방식
- Staging의 실제 filesystem 위치와 layout
- Staging retention / cleanup 정책
- ecosystem별 install option 전달 규칙
- local/global install 세부 처리
- overwrite / filesystem conflict 정책
- Promotion confirmation이 필요한 정확한 조건
- standalone binary/archive 대상 위치 syntax
- Promotion 실패 retry 방식
- Operation Status 세부 enum
- Promotion Record schema
- 권한 상승이 필요한 설치 작업 처리
- package manager별 transaction / rollback 지원 여부

위 항목은 후속 CLI IA, Domain Model, Staging/Promotion, ecosystem adapter 및 Tooling 설계에서 구체화한다.

### 압축된 사용자 모델

```text
helox npm install kenv
        ↓
보안 검사
        ↓
Policy = ALLOW
        ↓
검사한 정확한 Artifact / dependency set 확인
        ↓
digest 재검증
        ↓
trusted Promotion
        ↓
원래 요청했던 위치·환경에 설치
        ↓
설치 결과 기록
```

핵심 invariant는 다음과 같다.

```text
검사한 것
=
ALLOW의 대상
=
Verified Set / Manifest의 대상
=
실제로 Promotion하는 것
```

Heliopause는 사용자가 별도의 Promotion 단계를 직접 관리하지 않아도, 검사한 정확한 Artifact만 원래 요청한 사용자 환경으로 안전하게 반입되도록 하는 것을 기본 원칙으로 한다.

### 구조화된 결정과 구현 영향

- Promotion은 일반 사용자가 내부 단계로 직접 조작하지 않는 trusted boundary다.
- `install`의 원래 ecosystem/source·operation·Artifact·resolved reference·대상 환경·범위·허용 option·standalone 위치 context를 유지한다.
- 실제 반입 Artifact/dependency는 동일 Inspection Run·Verified Set·Verified Manifest·identity/digest와 연결되어야 한다.
- Package는 Quarantine/Inspection → ALLOW → Verified Set/Verified Manifest 확정 → Staging 반입 전 identity/digest 재검증 → Staging → 사용자 환경 반입 전 identity/digest 재검증 → trusted Promotion → 사용자 환경 설치 경로를 사용한다.
- Staging은 실행·재검사·두 번째 Sandbox가 아닌 검증 완료 Artifact 보관·전달 영역이다.
- Verified Set 밖의 새 Artifact/dependency가 요구되면 Promotion을 중단하고 acquisition/Verification/필요한 검사로 되돌린다.
- Promotion 직전 실제 Artifact identity/digest와 Verified Set/Manifest 일관성을 재확인하고 불일치 시 중단한다.
- standalone binary/archive는 Staging 생략이 가능하지만 trusted Promotion과 identity/digest 재검증은 생략하지 않는다.
- 설치 대상·범위·덮어쓰기 등 context가 모호하면 필요한 입력을 요구하되 정상 ALLOW마다 반복 confirmation하지 않는다.
- `Policy Decision`과 실제 `Operation Status`를 분리하며 `Policy: ALLOW`와 `Installation: FAILED`가 함께 기록될 수 있다.
- `inspect + ALLOW`는 Promotion하지 않고, `MANUAL_REVIEW`는 보류, `BLOCK`은 금지한다.
- Promotion Record/Operation Result는 Run·Policy·Verified Set/Manifest·실제 identity/digest·operation·성공 여부를 추적해야 한다.

## User-Journey-008 누락 점검

- [x] Promotion의 trusted boundary와 일반 사용자 직접 조작 금지
- [x] 원래 install 실행 context 유지
- [x] 검사 대상·Verified Set/Manifest·실제 반입 대상 동일성
- [x] Package의 Verified Set·Staging·digest 재검증 경로
- [x] Staging의 비실행·비검사 보관·전달 역할
- [x] Verified Set 밖 새 Artifact/dependency의 자동 신뢰 금지
- [x] Promotion 직전 identity/digest 재검증
- [x] standalone binary/archive의 Staging 생략 예외와 trusted Promotion 유지
- [x] 모호한 설치 대상의 사용자 명확화와 confirmation 최소화
- [x] Policy Decision과 Operation Status 분리
- [x] inspect ALLOW의 Promotion 없음
- [x] MANUAL_REVIEW 보류 및 BLOCK 금지
- [x] Promotion 결과의 Run·Policy·Manifest·identity/digest·operation 추적
- [x] offline install·locking·layout·overwrite·retry·schema·권한·rollback 후속 결정 유보
