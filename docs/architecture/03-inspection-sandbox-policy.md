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

#### M3 seccheck remote observer 구현 결정

- Linux gVisor backend는 Docker에 설치한 `runsc-trace` runtime의 고정 `--pod-init-config`를 통해 trusted HAA observer의 protected shared Unix-domain `SOCK_SEQPACKET` endpoint를 gVisor seccheck `remote` sink에 제공한다.
- trace session은 Sandbox start 전 구성하고, remote sink setup failure를 무시하지 않는다. run별 동적 endpoint나 direct `runsc` OCI bundle은 MVP 범위가 아니다.
- gVisor remote sink의 protobuf framing·handshake·wire version을 그대로 사용한다. HAA는 별도 framing, `--strace`, stdout/stderr scraping 또는 일반 Host 파일 로그 fallback을 만들지 않는다.
- observer input은 Sentry compromise를 전제로 untrusted로 처리한다. 필수 trace point의 `container_id` context field와 connection/container/session mapping이 하나로 확정돼야 하며, protocol/version/size/drop/stream/mapping failure와 필수 event loss는 `INCOMPLETE`이고 `ALLOW` 경로가 없다.
- shared UDS를 통한 모든 Observation/Evidence/runtime state는 container/Sandbox Session별로 격리한다. observer socket은 Artifact container namespace·filesystem·environment에 전달하지 않고 Evidence Store/Policy/controller와 별도 trusted runtime asset으로 관리한다.

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

## M4·M5 resolver network-policy helper

M4 disposable npm resolver는 일반 Docker bridge 또는 proxy 환경변수에 의존하지 않는다. HAA가 Linux infrastructure adapter로 resolver 전용 Docker network와 Host firewall policy를 생성·적용·검증·정리한다.

- resolver container는 전용 network에만 연결하고 default-deny egress로 시작한다.
- 허용 대상은 hostname 하나가 아니라 npm metadata가 반환하는 tarball 흐름까지 검토해 확정한 사전 정의 registry endpoint 집합이다.
- Host trusted DNS preflight가 얻은 npm registry IPv4 결과는 firewall allow rule과 resolver container의 explicit `--add-host registry.npmjs.org:<ip>` 양쪽에 같은 집합으로 고정한다. resolver는 Docker embedded DNS나 DNS egress에 의존하지 않는다.
- Docker가 사용하는 Linux firewall backend를 Docker API의 `FirewallBackend.Driver`로 읽어 iptables 또는 nftables의 호환 backend를 선택한다. Docker가 bridge network 생성 시 firewall hook을 지연 생성할 수 있으므로, driver가 없으면 resolver 전용 network를 먼저 만든 뒤 container 시작 전 backend를 probe한다. Docker 기본 iptables의 `DOCKER-USER` hook을 직접 검증해 우선하며, 그 hook이 없을 때만 Docker의 `ip docker-bridges` nftables table을 검증한다. 두 검증이 모두 실패하면 resolver는 실행하지 않고 fail-closed 한다.
- controlled Linux CI는 Docker daemon의 `firewall-backend=iptables`와 `iptables=true`를 runtime setup에서 명시하고 `DOCKER-USER` hook을 assert한다. backend driver의 표시값은 host 구성을 설명할 뿐이며 CI acceptance는 실제 hook 검증까지 통과해야 한다. 이는 test host 구성의 pin이며 product helper가 host daemon 설정을 임의로 변경한다는 뜻이 아니다. 제품 host에서 backend 또는 hook을 검증하지 못하면 resolver는 계속 fail-closed 한다.
- unrestricted bridge fallback, Host 사전 방화벽/프록시 가정, proxy 환경변수만의 enforcement 주장은 금지한다.
- helper는 Sandbox/Infrastructure adapter에 둔다. Core/Domain/Application에는 firewall 명령, namespace, Docker network 또는 endpoint IP 구현 세부를 노출하지 않는다.

M5 PyPI resolver도 같은 helper lifecycle을 사용하되, 허용 endpoint는 `pypi.org` Simple API와 `files.pythonhosted.org` distribution 두 개로 고정한다. Host의 trusted DNS preflight가 얻은 IPv4 결과를 firewall allow rule과 resolver container의 explicit `--add-host` 양쪽에 같은 집합으로 고정한다. 따라서 resolver Sandbox가 Docker embedded DNS, Host proxy, 일반 bridge 또는 이름 기반 firewall rule에 의존하지 않는다. DNS preflight, IP set, policy apply/verify/cleanup, HTTPS redirect/endpoint validation 중 하나라도 불완전하면 graph를 반환하지 않는다.

M5 resolver container는 fixed `runsc-trace`와 shared trusted observer endpoint를 사용한다. observer mapping·stream 종료·dropped event 또는 collection failure는 resolver 성공을 무효화한다. raw pip report와 Simple JSON은 adapter의 bounded parser 안에서만 처리하고, Core/Application/Policy에는 generic exact graph와 report digest만 전달한다.

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- |
| M4-Resolver-Network | 2026-08-14 | HAA 소유 resolver network-policy helper가 전용 Docker network와 default-deny Host firewall policy를 lifecycle 전체에서 관리 | package manager resolution의 외부 통신을 Host 사전 구성이나 proxy 설정에 의존하지 않고 제한하기 위해 | Linux firewall backend probe, endpoint policy contract, apply/verify/cleanup fail-closed 처리와 adapter-only 구현이 필요 |
| M5-PyPI-Resolver | 2026-08-20 | PyPI resolver가 fixed PyPI endpoint IP set, gVisor observer 및 bounded Simple/report cross-check를 함께 요구 | resolver candidate graph가 index metadata, runtime, egress 또는 observation 불확실성을 우회하지 못하게 하기 위해 | PyPI endpoint pinning, observer lifecycle, report/Simple parser와 Linux integration이 M5-003 acceptance가 됨 |

## M8 production Host control plane boundary

M8은 Sandbox/Promotion adapter가 각자 `exec.LookPath`와 ambient environment를
사용하는 구조를 제거한다. Composition Root가 concrete trusted Host tool registry,
process-scoped observer supervisor와 network-policy client를 생성하고 관련
Infrastructure adapter에 주입한다.

```text
bootstrap
  ├─ TrustedHostTool registry/executor
  │    ├─ verified local Docker endpoint
  │    └─ registered runsc/helper identity
  ├─ process-scoped Observer Supervisor
  │    └─ one helper + one normalized receiver
  └─ ResolverPolicy client
       └─ narrow privileged Host service

sandbox / promotion adapters
  └─ consumer-owned narrow interface만 사용
```

- `TrustedHostTool`은 Infrastructure implementation이며 Core/Application/Policy에
  Host path, environment, Docker endpoint와 runtime-specific type을 노출하지
  않는다.
- observer supervisor는 npm, PyPI resolver/wheel/sdist와 GitHub ELF backend가
  공유하는 process-scoped resource다. fixed endpoint를 exclusive하게 소유하며
  다른 owner가 있으면 unlink/reuse하지 않고 fail-closed한다.
- supervisor는 root-owned trusted helper executable을 시작 직전에 다시 확인하고,
  receiver bind → helper readiness pipe → remote socket identity 확인 순서만
  허용한다. helper exit는 receiver global fault가 되어 모든 active trace를
  incomplete로 끝내며, close는 자신이 만든 inode만 제거한다.
- M8 MVP는 여러 `helox` process를 하나의 observer service로 multiplex하지 않는다.
  향후 Host-scoped service는 container/session routing과 process authorization을
  새 contract로 해결한 뒤에만 추가한다.
- privileged resolver helper는 create/verify/remove policy라는 typed operation만
  제공한다. raw firewall command나 high-level package/hostname input은 허용하지
  않는다.
- Sandbox raw observation capability는 actual production helper schema와 함께
  선언한다. Inspection/Policy는 transport가 만들 수 없는 event를 absence-based
  clean finding으로 해석하지 않는다.

상세 invariant와 순서는
[M8 Production Trust Hardening Contract](../planning/11-m8-production-trust-hardening-contract.md)가
소유한다.

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| M8-Host-Control | 2026-08-24 | validated Host tool executor, process-scoped observer supervisor와 typed privileged network helper를 bootstrap-owned infrastructure boundary로 구성 | Sandbox·Promotion·firewall·observation 실행 성공이 ambient Host state나 누락된 helper lifecycle에 의존하지 않게 하기 위해 | M8 package boundary, dependency injection, exclusive resource lifecycle와 production composition E2E가 필요 |
