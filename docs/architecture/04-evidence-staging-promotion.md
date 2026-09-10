# Architecture — Evidence, Staging, and Promotion

## 사용자 원문: Architecture-007

```text
### Architecture-007: Evidence / Result 저장 구조

Heliopause는 검사 근거와 결과를 보존하는 `Evidence Store`와 검증 완료 Artifact를 반입 전 보관하는 `Staging Area`를 논리적으로 분리한다. 두 영역은 향후 동일한 물리적 storage backend를 사용할 수 있으나 목적·권한·lifecycle은 독립적으로 관리한다.

Evidence는 Sandbox 및 외부 도구에서 수집된 **원본 관찰·검증 자료(Raw Evidence)**와 이를 해석·정규화한 Verification Result, Finding 및 기타 파생 결과를 구분하여 보존한다. Finding/Decision의 근거로 채택된 Raw Evidence는 정해진 retention 동안 보존하고, 대용량 원시 telemetry의 보존·요약 정책은 후속 결정으로 둔다.

모든 검사 자료는 하나의 추적 가능한 **Inspection Run**을 중심으로 연결한다. Inspection Run은 검사 대상 Artifact identity/digest, 실행 환경과 backend, 수행·미수행 검사, Verification Result, Evidence, Finding, SBOM, Policy Decision 및 관련 Manifest를 추적할 수 있어야 한다.

Evidence 수집과 저장은 신뢰된 Heliopause control/observer 영역에서 수행하며, 신뢰하지 않는 Sandbox 내부의 Artifact가 Evidence, Finding, Policy Decision 또는 검사 기록을 직접 생성·수정·삭제할 수 없도록 한다.

Evidence와 주요 결과물은 생성 이후 변경 여부를 확인할 수 있는 무결성 검증 수단을 가져야 한다. 구체적인 hashing, append-only storage, content-addressed storage 또는 signing 방식은 후속 storage/tooling 설계에서 결정한다.

Evidence, Finding, SBOM, Policy Decision 및 Verified Manifest는 자신이 어떤 Artifact와 dependency의 identity/digest에 대한 결과인지 추적할 수 있어야 하며, 검사된 Artifact와 실제 Staging/Promotion 대상의 동일성을 검증할 수 있어야 한다.

Raw Evidence, Verification Result, Finding, Policy Decision, SBOM, Verified Manifest, 사람용 결과와 machine-readable 결과는 각각의 의미와 책임을 유지하며 서로 참조 관계로 연결한다. 이를 하나의 불투명한 결과 blob으로 취급하지 않는다.

구체적인 storage engine, filesystem layout, database 사용 여부, serialization format, schema, retention 기간 및 cleanup 정책은 Architecture-007에서 고정하지 않고 이후 Domain Model·Storage/Tooling 설계에서 확정한다.

- 압축하자면 다음과 같다.

Artifact digest

      ↓

Inspection Run

      ↓

Raw Evidence → Finding → Policy Decision

      ↓

SBOM / Manifest / Report

그리고 Evidence Store ≠ Staging Area
```

## 구조화된 결정

### Architecture-007 Evidence Store와 Staging Area 분리

| 영역 | 목적 | 권한·lifecycle |
| --- | --- | --- |
| Evidence Store | 검사 근거와 결과 보존 | trusted control/observer가 수집·저장, 원본·파생 결과 추적 |
| Staging Area | 검증 완료 Artifact를 반입 전 보관 | 실행·외부 network 없이 승격 세트 보관·반출, Evidence Store와 독립 관리 |

두 영역은 향후 동일한 물리적 storage backend를 사용할 수 있지만 목적·권한·lifecycle은 논리적으로 독립이다. `Evidence Store ≠ Staging Area`를 기본 원칙으로 한다.

### Architecture-007 Evidence 계층

Evidence Store는 다음 자료를 구분하여 보존한다.

- Raw Evidence: Sandbox·외부 도구에서 수집된 원본 관찰·검증 자료
- Verification Result: Raw Evidence와 검증 결과를 정규화한 결과
- Finding: Evidence를 해석한 검사 계층의 보안 결과
- Policy Decision: Finding·Evidence·검사 상태를 종합한 최종 판정
- SBOM·Verified Manifest: 구성요소·승격 세트 및 digest 연결 결과
- 사람용 결과·machine-readable 결과: 동일한 검사 기록을 서로 다른 표현으로 제공하는 결과물

Finding/Decision의 근거로 채택된 Raw Evidence는 정해진 retention 동안 보존한다. 대용량 원시 telemetry의 보존·요약 정책은 후속 결정으로 두며, 각 자료는 고유 의미와 책임을 유지하고 참조 관계로 연결한다. 이를 하나의 불투명한 결과 blob으로 합치지 않는다.

### Architecture-007 Inspection Run 추적

모든 검사 자료는 추적 가능한 `Inspection Run`을 중심으로 연결한다.

```text
Artifact digest
      ↓
Inspection Run
      ↓
Raw Evidence → Finding → Policy Decision
      ↓
SBOM / Manifest / Report
```

`Inspection Run`은 최소한 다음을 추적할 수 있어야 한다.

- 검사 대상 Artifact identity/digest
- dependency identity/digest
- 실행 환경과 Sandbox backend
- 수행·미수행 검사와 사유
- Verification Result
- Raw Evidence와 파생 Evidence
- Finding
- SBOM
- Policy Decision
- 관련 Verified Manifest

### Architecture-007 신뢰 경계와 무결성

- Evidence 수집과 저장은 신뢰된 Heliopause control/observer 영역에서 수행한다.
- 신뢰하지 않는 Sandbox 내부 Artifact는 Evidence·Finding·Policy Decision·검사 기록을 직접 생성·수정·삭제할 수 없다.
- Evidence와 주요 결과물은 생성 이후 변경 여부를 확인할 수 있는 무결성 검증 수단을 가져야 한다.
- hashing, append-only storage, content-addressed storage, signing 중 구체 방식은 후속 storage/tooling 설계에서 결정한다.
- 검사된 Artifact와 실제 Staging/Promotion 대상의 동일성을 Artifact·dependency identity/digest 참조로 검증한다.

### Architecture-007 후속 결정 유보

다음 구현 세부는 Architecture-007에서 고정하지 않는다.

- storage engine
- filesystem layout
- database 사용 여부
- serialization format
- 세부 schema
- retention 기간
- cleanup 정책

위 항목은 이후 Domain Model·Storage/Tooling 설계에서 결정한다.

## Architecture-007 구현 영향

- `Inspection Run`을 모든 검사 자료의 추적 루트로 정의한다.
- Evidence Store와 Staging Area에 서로 다른 Port/권한/lifecycle 계약을 둔다.
- Raw Evidence와 Verification Result·Finding·Policy Decision·SBOM·Manifest·Report의 참조 관계를 모델링한다.
- 각 결과가 Artifact·dependency identity/digest를 참조하도록 한다.
- Evidence와 주요 결과물의 무결성 상태와 검증 근거를 기록한다.
- trusted control/observer만 Evidence Store에 쓰도록 하고 Sandbox 실행 영역에는 결과 저장 권한을 주지 않는다.
- Staging Area에는 검증 완료 세트의 Artifact·dependency와 관련 Manifest 참조만 허용하고 실행·외부 network 접근을 분리한다.
- 터미널용 요약과 machine-readable 결과가 동일한 Inspection Run·Policy Decision·digest를 참조하는지 검증한다.
- storage engine·layout·schema·retention·cleanup은 후속 Domain Model/Storage 결정에 맞춰 구현한다.

## Architecture-007 누락 점검

- [x] Evidence Store와 Staging Area 논리적 분리
- [x] 동일 물리 storage backend 사용 가능성 및 독립 목적·권한·lifecycle
- [x] Raw Evidence와 Verification Result·Finding·파생 결과 구분 보존
- [x] Finding/Decision 근거로 채택된 Raw Evidence의 정해진 retention 보존
- [x] 대용량 원시 telemetry의 보존·요약 정책 후속 결정
- [x] Inspection Run 중심 추적
- [x] Artifact identity/digest 추적
- [x] 실행 환경·backend 추적
- [x] 수행·미수행 검사와 사유 추적
- [x] Verification Result·Evidence·Finding·SBOM·Policy Decision·Manifest 추적
- [x] trusted control/observer 영역에서 Evidence 수집·저장
- [x] Sandbox Artifact의 Evidence·Finding·Policy·검사 기록 직접 변경 금지
- [x] 생성 이후 변경 확인을 위한 무결성 수단
- [x] Artifact·dependency identity/digest와 Staging/Promotion 대상 동일성 검증
- [x] 각 결과물의 의미·책임 유지와 참조 관계 연결
- [x] 불투명한 단일 결과 blob 취급 금지
- [x] storage engine·layout·database·serialization·schema·retention·cleanup 후속 결정 유보

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-007 | 2026-08-10 | Evidence Store와 Staging Area를 논리적으로 분리하고 Inspection Run 중심으로 Raw Evidence·Finding·Policy·SBOM·Manifest를 참조 연결하며 저장 기술은 후속 결정 | 원본 근거를 보존하고 검사 결과·실제 반입 대상의 추적성과 무결성을 확보하기 위해 | Evidence/Result·Staging Port, Inspection Run 모델, 무결성 계약, trusted writer와 후속 storage/schema 결정이 설계 기준이 됨 |

## 사용자 원문: Architecture-008

```text
### Architecture-008: Staging / Promotion 구조

Heliopause는 위험한 Artifact를 검사하는 Quarantine/Sandbox 영역과, 검증 완료 Artifact를 사용자 환경 반입 전 보관하는 `Staging Area`를 명확히 분리한다. Staging Area에는 최종 Policy Decision이 `ALLOW`인 Artifact만 승격할 수 있으며, `MANUAL_REVIEW` 또는 `BLOCK` 상태의 Artifact는 자동 승격·반입하지 않는다.

Package ecosystem의 경우 검사·설치 과정에서 실제로 사용된 Primary Artifact와 dependency를 하나의 **Verified Set**으로 관리한다. Verified Set의 각 구성요소는 identity, version, source, digest 및 관련 Inspection Run과 추적 가능하게 연결되어야 한다.

Staging Area는 검사 또는 실행 환경으로 사용하지 않는다. Staging 내부에서는 Artifact 실행 및 install script 실행을 허용하지 않으며, 외부 network 및 실제 Host Credential에 접근하지 못하도록 한다. 가능한 경우 Artifact를 immutable 또는 변경 탐지가 가능한 형태로 보존한다.

Artifact가 Quarantine에서 Staging으로 승격될 때와 Staging에서 실제 사용자 환경으로 반입·설치되기 직전에 digest를 재검증한다. 검사된 Artifact와 승격·반입 대상의 identity 또는 digest가 일치하지 않으면 Promotion을 중단하고 새로운 Artifact로 취급하여 필요한 검증 절차로 되돌린다.

Package 설치는 원칙적으로 검증 완료된 Verified Set만을 사용하여 수행한다. 실제 설치 과정에서 Verified Set에 포함되지 않은 새로운 dependency 또는 Artifact가 요구되면 이를 자동으로 신뢰하거나 설치하지 않고 acquisition/verification 흐름으로 되돌린다. 최소 검증으로 충분하지 않거나 위험·불확실성이 존재하면 full quarantine 대상으로 전환한다.

Package manager가 필요 없는 독립 binary/archive는 예외적으로 Staging Area를 반드시 거치지 않을 수 있다. 이 경우에도 검사된 대상과 실제 반입 대상의 identity/digest를 재검증한 후에만 사용자가 명시적으로 지정한 위치로 직접 반입할 수 있다.

여기서 직접 반입은 Staging Area만 생략한다는 의미이며, 신뢰하지 않는 Sandbox가 사용자 환경에 직접 접근하거나 쓰는 것을 허용한다는 의미가 아니다. 실제 반입은 trusted Promotion 경계를 통해 digest 재검증 후 수행한다.

Promotion 모듈은 Policy를 재판단하거나 Artifact를 다시 검사하지 않는다. Application/Workflow가 `ALLOW` 상태를 확인한 후 Promotion을 호출하며, Promotion은 검증 완료 Artifact의 동일성 확인, Staging 이동, Verified Set 적용 및 사용자 지정 환경 반입만 담당한다.

ALLOW Policy Decision, Verified Set 및 Verified Manifest의 기준 Inspection Run은 서로 일치해야 한다. 각 Artifact/dependency는 Verified Set/Manifest에 선언된 identity/digest와 정확히 일치하고 자신을 검증한 관련 Inspection Run까지 추적 가능해야 한다. 이 참조 관계를 벗어나 서로 다른 Decision·Verified Set·Manifest 또는 검증되지 않은 Artifact/dependency가 임의로 혼합되면 Promotion을 수행하지 않는다.

구체적인 Staging storage 방식, immutable 구현, local cache 구조, package manager별 offline/local install 방식 및 filesystem permission 세부사항은 Architecture-008에서 고정하지 않고 후속 Domain Model·Tooling·Implementation 설계에서 결정한다.

- 요약하면:

Quarantine

   ↓

Policy = ALLOW

   ↓

Verified Set

   ↓

Staging

   ↓

digest 재검증

   ↓

사용자 지정 환경

- 그리고 예외

Standalone binary/archive

   ↓

ALLOW

   ↓

digest 재검증

   ↓

사용자 지정 위치 직접 반입
```

## 구조화된 결정

### Architecture-008 Quarantine·Verified Set·Staging 분리

```text
Quarantine / Sandbox
      ↓ Policy = ALLOW
Verified Set
      ↓
Staging Area
      ↓ digest 재검증
사용자 지정 환경
```

- Quarantine/Sandbox는 위험 Artifact의 검사·설치·실행 영역이다.
- `Staging Area`는 검증 완료 Artifact를 사용자 환경 반입 전 보관하는 영역이며 검사·실행 환경이 아니다.
- Staging Area에는 최종 Policy Decision이 `ALLOW`인 Artifact만 승격할 수 있다.
- `MANUAL_REVIEW` 또는 `BLOCK` 상태는 자동 승격·반입하지 않는다.

### Architecture-008 Verified Set

Package ecosystem에서는 검사·설치 과정에서 실제 사용된 Primary Artifact와 dependency를 하나의 `Verified Set`으로 관리한다.

각 구성요소는 다음과 연결되어야 한다.

- identity
- version
- source
- digest
- 관련 `Inspection Run`

Package 설치는 원칙적으로 검증 완료된 Verified Set만 사용한다. Verified Set에 없는 새 dependency나 Artifact가 실제 설치 과정에서 요구되면 자동 신뢰·설치하지 않고 acquisition/verification 흐름으로 되돌린다. 최소 검증으로 충분하지 않거나 위험·불확실성이 있으면 full quarantine 대상으로 전환한다.

### Architecture-008 Staging 제약

- Staging 내부에서 Artifact 실행과 install script 실행을 허용하지 않는다.
- Staging을 검사 환경으로 사용하지 않는다.
- 외부 network와 실제 Host Credential에 접근하지 못하도록 한다.
- 가능한 경우 Artifact를 immutable 또는 변경 탐지가 가능한 형태로 보존한다.
- Staging storage 방식·immutable 구현·local cache·offline/local install·filesystem permission 세부사항은 후속 설계에서 결정한다.

### Architecture-008 동일성 재검증과 Promotion 중단

Artifact가 Quarantine에서 Staging으로 승격될 때와 Staging에서 사용자 환경으로 반입·설치되기 직전에 digest를 재검증한다.

검사된 Artifact와 승격·반입 대상의 identity 또는 digest가 일치하지 않으면:

1. Promotion을 중단한다.
2. 대상 Artifact를 새로운 Artifact로 취급한다.
3. 필요한 acquisition/verification/quarantine 절차로 되돌린다.

### Architecture-008 독립 binary/archive 예외

Package manager가 필요 없는 독립 binary/archive는 예외적으로 Staging Area를 반드시 거치지 않을 수 있다. 이 경우에도 검사 대상과 실제 반입 대상의 identity/digest를 재검증한 후에만 사용자가 명시한 위치로 직접 반입할 수 있다.

여기서 직접 반입은 Staging Area만 생략한다는 의미이며, 신뢰하지 않는 Sandbox가 사용자 환경에 직접 접근하거나 쓰는 것을 허용한다는 의미가 아니다. 실제 반입은 trusted Promotion 경계를 통해 digest 재검증 후 수행한다.

### Architecture-008 Promotion 책임

| 담당 | 책임 |
| --- | --- |
| Application/Workflow | `ALLOW` 상태 확인 후 Promotion 호출 |
| Promotion | 검증 완료 Artifact 동일성 확인, Staging 이동, Verified Set 적용, 사용자 지정 환경 반입 |
| Policy | 최종 Decision 생성 |
| Inspection/Verification | Artifact 검사·검증 |

Promotion은 Policy를 재판단하거나 Artifact를 다시 검사하지 않는다.

### Architecture-008 Promotion identity invariant

`ALLOW` Policy Decision, `Verified Set` 및 `Verified Manifest`의 기준 `Inspection Run`은 서로 일치해야 한다. 각 Artifact/dependency는 Verified Set/Manifest에 선언된 identity/digest와 정확히 일치하고 자신을 검증한 관련 `Inspection Run`까지 추적 가능해야 한다. 이 참조 관계를 벗어나 서로 다른 Decision·Verified Set·Manifest 또는 검증되지 않은 Artifact/dependency가 임의로 혼합되면 Promotion을 수행하지 않는다.

## Architecture-008 구현 영향

- Quarantine/Sandbox와 Staging Area에 별도 Port·권한·lifecycle을 둔다.
- Verified Set 모델에 Primary Artifact·dependency identity/version/source/digest와 Inspection Run 참조를 포함한다.
- Staging 승격 전과 사용자 환경 반입 직전에 digest 재검증 단계를 workflow에 고정한다.
- 동일성 불일치 시 Promotion 중단과 재검증 workflow 전환을 구현한다.
- Promotion 입력의 Policy Decision·Verified Set·Verified Manifest의 기준 Inspection Run이 서로 일치하는지, 각 Artifact/dependency가 Verified Set/Manifest의 선언 identity/digest와 정확히 일치하고 자신을 검증한 관련 Inspection Run까지 추적 가능한지 검증한다. 참조 관계를 벗어난 Decision·Verified Set·Manifest 또는 검증되지 않은 Artifact/dependency가 혼합되면 Promotion을 중단한다.
- Staging용 API에는 실행·install script·외부 network·실제 Host Credential 접근이 없도록 한다.
- Package manager 설치는 원칙적으로 Verified Set만 사용하며, Verified Set 밖의 dependency를 요청하면 acquisition/verification으로 되돌리고 필요 시 full quarantine으로 전환한다.
- 독립 binary/archive 직접 반입은 D-011과 연결된 예외 경로로 구현하고 digest 재검증을 필수화한다.
- 독립 binary/archive 직접 반입은 Staging Area만 생략하며, 신뢰하지 않는 Sandbox의 사용자 환경 직접 접근·쓰기를 허용하지 않고 trusted Promotion 경계를 통해 digest 재검증 후 수행한다.
- Promotion 구현은 Policy·Inspection 호출 권한을 갖지 않고 Application이 전달한 검증 완료 입력만 사용한다.
- storage·immutable·cache·offline install·permission 세부는 후속 Domain Model·Tooling·Implementation 설계와 연결한다.

## Architecture-008 누락 점검

- [x] Quarantine/Sandbox와 Staging Area 명확한 분리
- [x] Staging에는 `ALLOW` Artifact만 승격
- [x] `MANUAL_REVIEW`·`BLOCK` 자동 승격·반입 금지
- [x] Primary Artifact와 dependency의 Verified Set 관리
- [x] Verified Set 구성요소의 identity·version·source·digest·Inspection Run 연결
- [x] Staging을 검사·실행 환경으로 사용하지 않음
- [x] Staging 내 Artifact·install script 실행 금지
- [x] 외부 network·실제 Host Credential 비노출
- [x] immutable 또는 변경 탐지 가능한 보존
- [x] Quarantine→Staging 승격 시 digest 재검증
- [x] Staging→사용자 환경 반입 직전 digest 재검증
- [x] identity/digest 불일치 시 Promotion 중단과 재검증 전환
- [x] Package 설치의 원칙적 Verified Set 사용
- [x] Verified Set 밖의 새 dependency·Artifact 자동 신뢰·설치 금지
- [x] 새 dependency의 acquisition/verification 및 필요 시 full quarantine 전환
- [x] 독립 binary/archive의 Staging 우회 예외와 digest 재검증
- [x] 직접 반입이 Staging만 생략하고 trusted Promotion을 통해 수행됨
- [x] Application/Workflow의 ALLOW 확인 후 Promotion 호출
- [x] Promotion의 동일성 확인·Staging·Verified Set·반입 책임
- [x] Promotion의 Policy 재판단·재검사 금지
- [x] ALLOW Decision·Verified Set·Verified Manifest의 기준 Inspection Run 일치
- [x] Artifact/dependency의 Verified Set/Manifest 선언 identity/digest 일치 및 자신을 검증한 Inspection Run 추적
- [x] 참조 관계를 벗어난 Decision·Verified Set·Manifest 또는 검증되지 않은 Artifact/dependency 혼합 시 Promotion 금지
- [x] Staging storage·immutable·cache·offline install·permission 후속 결정

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-008 | 2026-08-10 | Quarantine·Verified Set·Staging·Promotion을 분리하고 승격/반입 전 digest 재검증과 독립 binary/archive 예외를 확정 | 검증된 실제 Artifact 세트의 동일성을 보장하고 Staging을 비실행·비네트워크 보관 영역으로 유지하기 위해 | Verified Set 모델, 이중 digest 검증, Promotion 책임 경계, dependency 재검역과 후속 storage 결정이 설계 기준이 됨 |
