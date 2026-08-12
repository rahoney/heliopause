# Architecture — Inspection, Sandbox, and Policy

## 사용자 원문: Architecture-005

```text
### Architecture-005: Sandbox / Inspection Backend 구조

`Inspection`과 `Sandbox/Runtime`의 책임을 분리한다. Inspection은 어떤 정적·동적 검사를 수행하고 관찰 결과를 어떻게 Finding/Evidence로 해석할지를 담당하며, Sandbox는 격리 환경의 생성·준비·실행·관찰·종료를 담당한다. Sandbox 자체는 Artifact의 안전성이나 최종 Policy Decision을 판단하지 않는다.

Sandbox는 공통 `Sandbox Port` 뒤의 교체 가능한 backend로 구성한다. MVP에서는 Linux backend를 구현하되 Core/Application/Inspection이 특정 container runtime, namespace, seccomp 또는 기타 Linux 구현 기술에 직접 의존하지 않도록 한다. 향후 macOS·Windows 또는 Rust 기반 hardened isolation backend를 기존 계약을 통해 추가할 수 있도록 한다.

각 Sandbox 기반 격리·동적 검사 실행은 원칙적으로 독립적인 **ephemeral Sandbox Session**에서 수행한다. 세션은 생성·준비·Artifact 반입·설치/실행·**관찰 데이터 수집**·종료의 lifecycle을 가지며 검사 완료 후 폐기한다. timeout, resource-limit 초과, 강제 종료 또는 이상 상태가 발생한 세션은 재사용하지 않는다.

신뢰된 Heliopause control/observation 영역과 신뢰하지 않는 Artifact 실행 영역을 분리한다. Sandbox 내부의 Artifact는 Heliopause controller, 최종 Evidence 저장소, Policy 상태 등을 직접 수정할 수 없어야 하며, 가능한 관찰 정보는 실행 대상 외부의 trusted observer에서 수집한다.

Sandbox에는 실제 Host filesystem, Credential, 환경변수, 내부 network 및 Host service/process를 노출하지 않는다. 동적 행위 검증에 필요한 경우 독립적인 simulated filesystem, dummy credential, honeytoken 및 가짜 service를 제공한다.

Network는 기본적으로 외부·내부 실 network 접근을 차단하며, 동적 분석이 필요한 경우 통제된 DNS/HTTP 등의 관찰 환경을 제공할 수 있다. 구체적인 network isolation 및 simulation 기술은 후속 구현 결정에서 확정한다.

Sandbox backend는 process, filesystem, network, resource, credential/honeytoken 접근 등 **raw observation과 실행 상태**를 반환한다. Inspection 계층은 이를 Finding/Evidence로 정규화하고, 최종 `ALLOW / MANUAL_REVIEW / BLOCK` 판정은 Policy 계층에서만 수행한다.

구체적인 Linux isolation 기술, container runtime, syscall filtering 방식, resource-limit 수치 등은 Architecture-005에서 고정하지 않고 후속 backend/tooling 설계에서 결정한다.
```

## 구조화된 결정

### Architecture-005 Inspection과 Sandbox 책임 분리

| 영역 | 담당 | 담당하지 않는 것 |
| --- | --- | --- |
| Inspection | 수행할 정적·동적 검사 정의, raw observation 해석, `Finding`/`Evidence` 정규화 | 격리 환경 생성·실행 lifecycle, 최종 Policy Decision |
| Sandbox / Runtime | 격리 환경 생성·준비·Artifact 반입·설치/실행·관찰·종료, raw observation·실행 상태 반환 | Artifact 안전성 판단, 최종 `ALLOW`/`MANUAL_REVIEW`/`BLOCK` |
| Policy | 정규화된 Finding/Evidence와 검증 결과를 종합한 최종 판정 | Sandbox 제어, 직접 검사 수행 |

Inspection과 Sandbox는 `Sandbox Port`를 통해 연결하며 Inspection·Core·Application은 특정 container runtime, namespace, seccomp 등 Linux 기술에 직접 의존하지 않는다.

### Architecture-005 Sandbox Backend와 OS 확장

- Sandbox는 공통 `Sandbox Port` 뒤의 교체 가능한 backend로 구성한다.
- MVP에서는 Linux backend를 구현한다.
- 향후 macOS·Windows 또는 Rust 기반 hardened isolation backend를 기존 계약으로 추가할 수 있도록 한다.
- Core/Application/Inspection은 특정 Linux 구현 기술을 직접 참조하지 않는다.
- 구체적인 Linux isolation 기술, container runtime, syscall filtering 방식은 후속 backend/tooling 설계에서 결정한다.

### Architecture-005 Ephemeral Sandbox Session lifecycle

각 Sandbox 기반 격리·동적 검사 실행은 원칙적으로 독립적인 ephemeral Sandbox Session에서 수행한다.

```text
생성
  → 준비
  → Artifact 반입
  → 설치/실행
  → 관찰 데이터 수집
  → 종료
  → 폐기
```

Session은 검사 완료 후 폐기한다. timeout, resource-limit 초과, 강제 종료 또는 이상 상태가 발생한 Session은 재사용하지 않는다.

### Architecture-005 신뢰 경계와 관찰

- 신뢰된 Heliopause control/observation 영역과 신뢰하지 않는 Artifact 실행 영역을 분리한다.
- Sandbox 내부 Artifact가 Heliopause controller, 최종 Evidence 저장소, Policy 상태를 직접 수정할 수 없도록 한다.
- 가능한 관찰 정보는 실행 대상 외부의 trusted observer에서 수집한다.
- Sandbox backend는 process, filesystem, network, resource, credential/honeytoken 접근의 raw observation과 실행 상태를 반환한다.
- Inspection 계층은 raw observation을 Finding/Evidence로 정규화한다.
- 최종 Policy Decision은 Policy 계층에서만 생성한다.

### Architecture-005 Host 자산과 Network 정책

- 실제 Host filesystem, Credential, 환경변수, 내부 network, Host service/process를 Sandbox에 노출하지 않는다.
- 동적 검증이 필요한 경우 독립적인 simulated filesystem, dummy credential, honeytoken, 가짜 service를 제공한다.
- Network는 기본적으로 외부·내부 실 network 접근을 차단한다.
- 동적 분석에서 필요한 경우 통제된 DNS/HTTP 등의 관찰 환경을 제공할 수 있다.
- 구체적인 network isolation 및 simulation 기술은 후속 구현 결정으로 남긴다.

## Architecture-005 구현 영향

- `Sandbox Port`에 Session 생성·준비·반입·설치/실행·관찰·종료와 raw observation 반환 계약을 정의한다.
- Inspection Port/Provider는 Sandbox Port 결과를 받아 검사별 Finding/Evidence로 정규화한다.
- Sandbox backend는 최종 Policy Decision type이나 Policy 상태를 생성하지 않는다.
- Session identity, lifecycle 상태, 종료 사유, 폐기 여부와 재사용 금지 상태를 기록한다.
- timeout·resource-limit 초과·강제 종료·이상 상태를 실행 상태와 Evidence에 연결한다.
- trusted observer와 Artifact 실행 영역의 저장·통신 경계를 분리한다.
- simulated filesystem, dummy credential, honeytoken, 가짜 DNS/HTTP service fixture를 준비한다.
- Linux backend 구현 세부와 network·syscall·resource 수치는 후속 backend/tooling 결정으로 이관한다.
- macOS·Windows·Rust backend 추가가 기존 Sandbox Port를 깨뜨리지 않는지 contract test로 검증한다.

## Architecture-005 누락 점검

- [x] Inspection과 Sandbox/Runtime 책임 분리
- [x] Inspection의 검사 정의·관찰 해석·Finding/Evidence 정규화 책임
- [x] Sandbox의 생성·준비·실행·관찰·종료 책임
- [x] Sandbox 자체의 안전성·최종 Policy Decision 판단 금지
- [x] 공통 Sandbox Port와 교체 가능한 backend
- [x] MVP Linux backend
- [x] Core/Application/Inspection의 특정 Linux 기술 직접 의존 금지
- [x] 향후 macOS·Windows·Rust hardened backend 확장 가능성
- [x] 독립적인 ephemeral Sandbox Session 원칙
- [x] 생성·준비·반입·설치/실행·관찰 데이터 수집·종료 lifecycle
- [x] 검사 완료 후 Session 폐기
- [x] timeout·resource-limit 초과·강제 종료·이상 Session 재사용 금지
- [x] trusted control/observation 영역과 untrusted Artifact 실행 영역 분리
- [x] Artifact의 controller·Evidence 저장소·Policy 상태 직접 수정 금지
- [x] 외부 trusted observer 관찰 수집
- [x] 실제 Host filesystem·Credential·환경변수·내부 network·Host service/process 비노출
- [x] simulated filesystem·dummy credential·honeytoken·가짜 service 제공 가능
- [x] 외부·내부 실 network 기본 차단
- [x] 통제된 DNS/HTTP 관찰 환경 제공 가능
- [x] raw observation·실행 상태 반환
- [x] Inspection의 Finding/Evidence 정규화
- [x] 최종 Policy Decision은 Policy만 수행
- [x] 구체 isolation/runtime/syscall/resource 수치 후속 결정

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-005 | 2026-08-10 | Inspection과 Sandbox 책임을 분리하고 공통 Sandbox Port·ephemeral Session·trusted observer 경계를 채택하며 Linux 구현 세부는 후속 결정으로 유보 | 격리 실행 기술과 검사 해석을 교체 가능하게 분리하고 Artifact가 관찰·결과를 오염시키지 못하도록 하기 위해 | Session lifecycle, raw observation 계약, 모의 Host·통제 network, backend 교체성과 후속 tooling 결정이 설계 기준이 됨 |

## 사용자 원문: Architecture-006

```text
### Architecture-006: Verification / Inspection / Policy 책임 구조

Heliopause는 `Verification`, `Inspection`, 외부 Tool Provider와 `Policy`의 책임을 명확히 분리한다.

`Verification`은 Artifact의 identity, source, digest/checksum, Registry integrity, signature, provenance 및 attestation 등 **Artifact가 무엇이며 어디서 왔는지에 관한 신뢰 근거**를 검증하고 정규화된 Verification Result와 Evidence를 생성한다.

`Inspection`은 Artifact의 파일·코드·dependency·설치·실행 행위와 Sandbox에서 수집된 raw observation을 분석하여 **Artifact가 무엇을 포함하고 어떤 행위를 하는지**에 관한 Finding과 Evidence를 생성한다.

외부 verifier, scanner, vulnerability source 및 기타 security tool의 고유 verdict, exit code, severity, 출력 schema는 Provider/Adapter 내부에서 해석하고 공통 결과 모델로 정규화한다. 외부 도구의 verdict를 Heliopause의 최종 Policy Decision으로 직접 사용하지 않는다.

`Evidence`는 관찰·검증된 사실과 원본 근거를 나타내고, `Finding`은 해당 Evidence에 대한 검사 계층의 정규화된 보안 해석을 나타낸다. Finding은 가능한 경우 자신을 뒷받침하는 Evidence를 참조한다.

여러 verifier·scanner·정적·동적 검사 결과는 공통 Verification Result, Finding, Evidence 및 검사 실행 상태로 수집하며, 구체적인 점수 계산이나 vendor별 우선순위는 검사 계층에 두지 않는다.

기능의 지원·미지원(Capability)과 개별 검사의 성공·실패·미완료·미실행·사용 불가(Execution Status), 그리고 실제 보안 Finding을 서로 구분한다. 검사하지 못했거나 결과가 없는 상태를 안전한 결과로 간주하지 않는다.

최종 `ALLOW / MANUAL_REVIEW / BLOCK` 결정은 오직 `Policy` 계층에서 생성한다. Policy는 Verification Result, Finding, Evidence, Capability, 검사 실행 상태와 검사 한계를 종합하며, `WARN`은 독립적인 최종 Policy Decision이 아니라 Finding의 severity 또는 경고 수준으로 취급한다.

구체적인 severity 체계, 점수제 여부, Policy rule과 threshold는 Architecture-006에서 고정하지 않고 후속 Policy/Domain Model 설계에서 결정한다.

Verifier ───────────────┐

Scanner ────────────────┤

Static Inspection ──────┤

Dynamic Inspection ─────┼→ Result / Finding / Evidence

Sandbox Observation ────┘

                              ↓

                           Policy

                              ↓

             ALLOW / MANUAL_REVIEW / BLOCK
```

## 구조화된 결정

### Architecture-006 Verification·Inspection 책임 분리

| 영역 | 핵심 질문 | 생성하는 결과 |
| --- | --- | --- |
| Verification | Artifact가 무엇이며 어디서 왔는가? | 정규화된 Verification Result, Evidence |
| Inspection | Artifact가 무엇을 포함하고 어떤 행위를 하는가? | Finding, Evidence |
| External Tool Provider/Adapter | 외부 도구의 고유 결과를 어떻게 공통 모델로 바꾸는가? | 공통 Verification Result, Finding, Evidence, 실행 상태 |
| Sandbox Observation | 격리 실행 중 무엇이 관찰되었는가? | raw observation, 실행 상태; Inspection이 Finding/Evidence로 해석 |
| Policy | 모든 결과와 한계를 고려한 최종 반입 결정은 무엇인가? | `ALLOW`, `MANUAL_REVIEW`, `BLOCK` |

`Verification`은 identity·source·digest/checksum·Registry integrity·signature·provenance·attestation 등 Artifact의 신뢰 근거를 검증한다. `Inspection`은 파일·코드·dependency·설치·실행 행위와 Sandbox raw observation을 분석한다.

### Architecture-006 Evidence와 Finding 관계

- `Evidence`는 관찰·검증된 사실과 원본 근거를 나타낸다.
- `Finding`은 Evidence에 대한 검사 계층의 정규화된 보안 해석이다.
- Finding은 가능한 경우 자신을 뒷받침하는 Evidence를 참조한다.
- Evidence와 Finding은 최종 Policy Decision과 구분한다.
- 여러 verifier·scanner·정적·동적 검사 결과는 공통 Verification Result, Finding, Evidence와 검사 실행 상태로 수집한다.

### Architecture-006 외부 도구 결과 정규화

외부 verifier, scanner, vulnerability source와 기타 security tool의 verdict·exit code·severity·출력 schema는 Provider/Adapter 내부에서 해석한다. 외부 도구의 고유 verdict를 Heliopause 최종 Policy Decision으로 직접 사용하지 않는다.

구체적인 점수 계산이나 vendor별 우선순위는 검사 계층에 두지 않고, 정규화된 결과와 근거를 Policy 입력으로 전달한다.

### Architecture-006 실행 상태와 보안 Finding 구분

| 구분 | 예시 | 처리 원칙 |
| --- | --- | --- |
| Capability | 지원 / 미지원 | `Unsupported`는 안전으로 해석하지 않음 |
| Execution Status | 성공(`Completed`) / 실패 / 미완료 / 미실행·`Skipped` / `Unavailable` | 실제 보안 Finding과 별도 기록하고 사유 보존 |
| Security Result | `Finding` / `Evidence` | 정규화하여 Policy 입력으로 사용 |

검사 실행이 실패·미완료·미실행·사용 불가이거나 Capability가 미지원이어도 이를 `ALLOW`에 해당하는 정상 결과로 해석하지 않는다.

### Architecture-006 Policy 최종 결정

- 최종 `ALLOW` / `MANUAL_REVIEW` / `BLOCK` 결정은 Policy 계층에서만 생성한다.
- Policy는 Verification Result, Finding, Evidence, Capability, 검사 실행 상태와 검사 한계를 종합한다.
- `WARN`은 독립적인 최종 Policy Decision이 아니라 Finding의 severity 또는 경고 수준이다.
- 구체적인 severity 체계, 점수제 여부, Policy rule과 threshold는 후속 Policy/Domain Model 설계에서 결정한다.

## Architecture-006 구현 영향

- Verification Port/Result와 Inspection Finding/Evidence Port를 별도 계약으로 정의한다.
- Evidence에 원본 관찰·검증 근거를, Finding에 관련 Evidence 참조와 정규화된 해석을 기록한다.
- 외부 Provider/Adapter가 vendor별 verdict·exit code·severity·schema를 내부에서 공통 결과로 변환한다.
- 실행 상태 type을 Verification Result·Finding·Evidence와 별도로 유지하고 상태별 사유를 기록한다.
- Policy 입력 모델에 Capability·Execution Status·검사 한계를 포함하되, 검사 모듈이 Policy를 직접 호출하지 않도록 한다.
- `WARN` severity와 최종 `MANUAL_REVIEW` Decision을 서로 다른 모델로 표현한다.
- severity·score·threshold의 구체 schema는 후속 Policy/Domain Model 결정으로 이관한다.

## Architecture-006 누락 점검

- [x] Verification·Inspection·External Tool Provider·Policy 책임 분리
- [x] Verification의 identity·source·digest/checksum 검증
- [x] Registry integrity·signature·provenance·attestation 검증
- [x] 정규화된 Verification Result·Evidence 생성
- [x] Inspection의 파일·코드·dependency·설치·실행 행위 분석
- [x] Sandbox raw observation 분석과 Finding/Evidence 생성
- [x] 외부 도구 고유 verdict·exit code·severity·schema 내부 해석
- [x] 외부 verdict를 최종 Policy Decision으로 직접 사용하지 않음
- [x] Evidence를 사실·원본 근거로 정의
- [x] Finding을 Evidence에 대한 정규화된 보안 해석으로 정의
- [x] Finding의 Evidence 참조
- [x] 공통 결과 모델과 검사 실행 상태 수집
- [x] vendor별 점수·우선순위를 검사 계층에 두지 않음
- [x] Capability의 지원·미지원과 Execution Status의 성공·실패·미완료·미실행·사용 불가 구분
- [x] Security Result의 Finding/Evidence 구분
- [x] 검사 불가·결과 없음 상태의 안전 해석 금지
- [x] Policy만 최종 ALLOW/MANUAL_REVIEW/BLOCK 생성
- [x] Policy 입력에 Capability·검사 실행 상태·검사 한계 포함
- [x] WARN을 Finding severity/경고 수준으로 취급
- [x] severity·score·rule·threshold 후속 결정 유보

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| Architecture-006 | 2026-08-10 | Verification·Inspection·Provider·Policy의 결과와 책임을 분리하고 Evidence/Finding·실행 상태를 정규화하며 Policy만 최종 판정을 생성 | 출처·무결성 근거와 내용·행위 관찰을 구분하고 외부 도구 결과를 교체 가능한 공통 모델로 통합하기 위해 | Verification/Inspection Port, Evidence 참조, 실행 상태 모델, Policy 입력·최종 Decision 경계와 후속 rule 설계가 기준이 됨 |
