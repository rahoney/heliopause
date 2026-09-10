# Interface Contract — Evidence, Staging, and Promotion

## Contract-003: Evidence / Staging / Promotion Ports

### 목적

Heliopause는 검사 결과의 기록, 검증 완료 Artifact의 보관, 실제 사용자 환경으로의 반입을 서로 다른 책임으로 분리한다.

```text
Inspection Run → Evidence Port → Evidence Store

Package ecosystem:
Policy = ALLOW
      → Verified Set / Manifest
      → Staging Port
      → Verified Staging Area
      → Promotion Port
      → User Environment

Standalone:
Policy = ALLOW
      → exact identity / digest 재확인
      → Promotion Port
      → User Environment
```

핵심 경계는 다음과 같다.

```text
Evidence  → 무엇을 근거로 판정했는가
Staging   → 정확히 무엇이 반입 허가되었는가
Promotion → 그 정확한 Artifact를 실제 Host에 반입한다
```

### Evidence Port

`Evidence Port`는 Inspection Run에서 생성되는 Evidence와 관련 기록을 신뢰 가능한 Evidence Store에 저장·조회하기 위한 계약이다.

개념적인 입력은 Inspection Run, Artifact identity/digest, Verification Result, Evidence, Finding, Execution Status, Inspection Limitation, Policy Decision 및 관련 metadata다. 모든 정보가 하나의 물리적 record에 저장되어야 한다는 뜻은 아니며, 각 결과가 다음에 추적 가능해야 한다.

```text
Run + 실제 Artifact + 생성 근거
```

신뢰하지 않는 Artifact 또는 Sandbox 내부 process가 Evidence 기록을 임의로 수정·삭제할 수 없어야 한다. 기록은 trusted controller/observer 측에서 수행한다. Raw Observation/Tool Output과 Normalized Evidence는 분리하여 저장하거나 참조할 수 있다.

Evidence Port는 기록과 추적만 담당하며 Finding 해석, Verification 수행, Policy Decision 생성, Verified Set 승인 또는 Promotion을 수행하지 않는다.

### Evidence 무결성 실패

Policy Decision이나 Promotion의 근거가 되는 필수 Evidence가 누락·손상되거나 무결성을 확인할 수 없다면 정상적인 신뢰 근거로 사용하지 않는다.

```text
Required Evidence → missing / corrupt / unverifiable
                  → ALLOW 또는 Promotion 근거로 사용 금지
```

구체적인 Evidence integrity mechanism은 후속 구현 설계에서 결정한다.

### Staging Port

`Staging Port`는 `ALLOW`된 Verified Set을 실제 Host Promotion 전에 **검증 완료 상태로 보관하는 영역에 반입**하기 위한 계약이다.

```text
Policy = ALLOW
        ↓
Verified Set / Verified Manifest
        ↓
identity / digest 재확인
        ↓
Staging Port → Verified Staging Area
```

`MANUAL_REVIEW` 또는 `BLOCK` 상태의 Artifact를 자동 Staging하지 않는다.

Package ecosystem에서 Staging Port는 유효한 `ALLOW`, 현재 Verified Set, Verified Manifest, 실제 Artifact identity/digest 일치를 확인해야 한다. Quarantine/Inspection 영역에서 Staging으로 이동하는 신뢰 경계에서 digest를 다시 확인하며 불일치하면 Staging을 중단하고 기존 `ALLOW`를 다른 content에 적용하지 않는다.

### Verified Staging Area의 성격

Staging은 두 번째 Sandbox가 아니다. Artifact 실행, install/lifecycle script 실행, 외부 network 접근, 실제 credential 제공, 임의 content 변경을 허용하지 않는다.

목적은 다음과 같다.

```text
검사를 통과한 정확한 Artifact를
Promotion 직전까지 동일한 상태로 보존한다.
```

가능한 경우 immutable 또는 변경 탐지 가능한 방식으로 관리한다.

### Promotion Port

`Promotion Port`는 `ALLOW`된 정확한 Artifact 또는 Verified Set을 Heliopause의 신뢰 경계를 넘어 실제 사용자 환경으로 반입하는 계약이다.

```text
Verified Staging Area
        ↓
Verified Manifest 확인
        ↓
identity / digest 재확인
        ↓
Original Install Context 확인
        ↓
Promotion Port → User Environment
```

Promotion은 Sandbox나 Artifact Adapter가 아닌 trusted Promotion 구현만 수행한다.

Package ecosystem의 Promotion은 다음 조건을 요구한다.

```text
유효한 ALLOW
+ Verified Manifest
+ Staging의 실제 Artifact identity/digest 일치
+ Original Install Context
```

따라서 Package ecosystem에서는 `Quarantine → Staging`과 `Staging → Host` 두 신뢰 경계에서 identity/digest를 재확인한다.

### Promotion의 책임 경계

Promotion은 이미 내려진 Policy Decision과 Verified Manifest를 집행한다. Artifact 안전성 재판정, Finding 생성, Policy rule 평가, 새 dependency 승인, mutable reference 재resolve, 임의 Artifact 다운로드를 수행하지 않는다.

```text
Promotion은 무엇을 허용할지 결정하지 않는다.
이미 허용된 정확한 것을 반입한다.
```

Promotion 또는 실제 install 중 Verified Manifest에 없는 Artifact가 필요하면 중단한다. 새 Artifact는 Resolve, Acquire, Verification/Inspection, Policy, 새로운 Verified Set/Manifest 생성 workflow를 거쳐야 한다.

### Standalone Artifact 예외

Package manager가 없는 standalone binary/archive는 기존 Architecture 예외에 따라 Verified Staging Area를 거치지 않고 직접 trusted Promotion할 수 있다.

```text
Standalone Artifact → Inspection → ALLOW
                    → exact identity / digest 재확인
                    → Original target context 확인
                    → Promotion Port → Host
```

이 경우에도 검사한 Artifact, `ALLOW`된 Artifact, Promotion하는 Artifact가 동일해야 하며 Sandbox가 Host에 직접 쓰지 않는다.

### Promotion 결과와 Policy의 구분

Promotion 또는 실제 install 실패가 기존 Inspection Run의 `ALLOW`를 자동 변경하지 않는다.

```text
Policy Decision = ALLOW
Promotion = Host permission error
Operation Result = FAILED
```

보안 판정과 실제 작업 실행 결과를 분리한다.

### Contract-003 Invariant

#### Invariant 1 — Evidence Store와 Staging을 분리한다

검사 근거 저장소와 Promotion 대상 Artifact 보관 영역을 같은 책임으로 취급하지 않는다.

#### Invariant 2 — Evidence는 untrusted Artifact가 변경하지 못한다

Evidence 기록은 trusted observer/controller 경계에서 수행한다.

#### Invariant 3 — 필수 Evidence 무결성이 깨지면 신뢰 근거로 사용하지 않는다

누락·손상된 필수 Evidence를 정상적인 `ALLOW` 근거로 취급하지 않는다.

#### Invariant 4 — Staging에는 `ALLOW`된 Verified Set만 들어간다

`MANUAL_REVIEW / BLOCK` Artifact를 자동 Staging하지 않는다.

#### Invariant 5 — Staging은 실행 환경이 아니다

Artifact 실행·lifecycle·임의 network access를 Staging 책임으로 만들지 않는다.

#### Invariant 6 — 두 신뢰 경계에서 identity/digest를 재확인한다

Package ecosystem에서는 `Quarantine → Staging`, `Staging → Host` 각 경계에서 실제 Artifact와 Verified Manifest의 identity/digest 일치를 확인한다. Staging을 생략하는 Standalone Artifact는 Promotion 직전에 실제 Artifact의 exact identity/digest가 해당 `ALLOW` 검사 대상과 일치하는지 재확인한다.

#### Invariant 7 — Promotion은 Policy를 재판정하지 않는다

Promotion은 이미 허가된 Artifact를 정확하게 반입하는 집행 책임만 가진다.

#### Invariant 8 — Promotion은 새로운 Artifact를 자동 추가하지 않는다

Verified Manifest 밖의 Artifact가 필요하면 검증 workflow로 되돌린다.

#### Invariant 9 — Sandbox는 Host Promotion을 수행하지 않는다

Host 반입은 trusted Promotion 경계를 통해서만 수행한다.

#### Invariant 10 — Promotion 실패와 Policy Decision을 구분한다

실제 설치 실패가 기존 `ALLOW`를 자동 `BLOCK`으로 변경하지 않는다.

#### Invariant 11 — Standalone 예외에서도 identity binding을 유지한다

Staging을 생략할 수는 있지만 검사한 exact identity/digest와 Promotion 대상은 동일해야 한다.

### 후속 결정으로 유보한 항목

- 실제 Go interface와 method signature
- Evidence writer/reader API와 Evidence ID/reference schema
- Evidence integrity mechanism, Store 구현/layout, retention/cleanup
- Staging handle/schema, filesystem/layout, immutable/change-detection 구현
- Verified Manifest serialization
- Promotion request/result struct와 ecosystem별 Promotion Adapter
- npm/pip offline/local install 및 standalone target write 방식
- atomic Promotion, rollback, partial install 처리
- Promotion failure code와 Operation Result 연결 schema

### 압축된 Contract

```text
Evidence Port
"검사와 판정의 근거를 신뢰 가능한 영역에 기록하라."
        ↓
Evidence Store

Staging Port
"ALLOW된 정확한 Verified Set을 변경 없이 보관하라."
        ↓
Verified Staging Area

Promotion Port
"검증된 정확한 Artifact를 원래 사용자 target으로 반입하라."
        ↓
User Environment
```

Package ecosystem의 기본 신뢰 흐름은 다음과 같다.

```text
Inspection → Evidence Store → Policy = ALLOW
           → Verified Set / Manifest → digest 재확인
           → Staging → digest 재확인
           → trusted Promotion → Host
```

```text
기록하는 것 ≠ 보관하는 것 ≠ 실제 반입하는 것
```

을 분리하면서 다음 동일성을 끝까지 유지한다.

```text
Package ecosystem:

실제로 검사한 content
=
Policy가 허용한 content
=
Staging에 들어간 content
=
Host에 Promotion한 content

Standalone:

실제로 검사한 content
=
Policy가 허용한 content
=
Host에 Promotion한 content
```

## Contract-003 구조화된 결정과 구현 영향

- Evidence Port, Staging Port, Promotion Port를 각각 기록·보관·반입 책임으로 분리한다.
- Evidence는 Run·실제 Artifact identity/digest·생성 근거에 추적 가능하며 untrusted 실행 환경이 변경할 수 없다.
- 필수 Evidence가 누락·손상·검증 불가하면 자동 `ALLOW` 또는 Promotion의 근거로 사용하지 않는다.
- Staging은 유효한 `ALLOW`, Verified Set/Manifest, 실제 identity/digest가 일치할 때만 허용한다.
- Verified Staging Area에서는 실행·lifecycle·network·credential 제공·임의 변경을 금지하고 immutable/change-detection을 지향한다.
- Package ecosystem은 Quarantine→Staging과 Staging→Host 경계에서 identity/digest를 각각 재확인한다.
- Promotion은 원래 Install Context에 이미 허가된 정확한 Artifact를 집행할 뿐 Policy 재판정·재resolve·신규 Artifact 추가를 하지 않는다.
- Manifest 밖 새 Artifact는 전체 검증 workflow와 새 Verified Set/Manifest로 회귀한다.
- Standalone은 Staging 생략을 허용하되 Promotion 직전 identity/digest binding과 trusted Promotion 경계를 유지한다.
- Promotion 실패는 Operation Result에 반영하며 기존 Policy Decision과 분리한다.
- 구체적인 API·storage·integrity·serialization·offline install·atomicity·rollback·failure schema는 후속 구현 설계로 유보한다.

## Contract-003 누락 점검

- [x] Evidence Port의 저장·조회와 추적 책임
- [x] Evidence Store의 trusted write 경계
- [x] Raw/normalized Evidence 분리 가능성
- [x] Evidence Port의 금지 책임
- [x] 필수 Evidence 무결성 실패의 fail-closed
- [x] Staging 반입 조건과 `ALLOW` 제한
- [x] Verified Staging Area의 비실행 성격
- [x] Promotion 입력 조건과 Original Install Context
- [x] 두 신뢰 경계의 identity/digest 재확인
- [x] Promotion의 집행 책임과 금지 책임
- [x] 신규 Artifact 발견 시 workflow 회귀
- [x] Standalone Staging 예외와 identity binding
- [x] Promotion 실패와 Policy Decision 분리
- [x] 11개 invariant
- [x] 후속 결정 항목 유보
