# Threat Model — Trusted Tooling and Evidence

## 사용자 원문: D-012

```text
12번: 검사 환경 이미지 및 기반 도구 신뢰 정책

Artifact 검사를 수행하는 컨테이너 이미지, VM 이미지 및 기타 기반 실행 환경 역시 외부 Software Artifact로 간주하며 무조건 신뢰하지 않는다.

가능한 경우 이미지와 기반 구성요소는 버전 태그뿐 아니라 digest로 고정하여 동일성을 확인하고, 제공되는 경우 publisher signature, provenance 및 attestation을 검증한다.

검증된 이미지 또는 기반 도구가 업데이트되어 digest나 주요 구성요소가 변경된 경우 기존 검증 결과를 그대로 승계하지 않고 변경 내용을 다시 확인한다.

검사 환경에는 필요한 구성요소만 포함하고, 불필요한 패키지·권한·서비스를 최소화하여 검사 환경 자체의 공격 표면을 줄인다.

검사 환경에서 사용하는 scanner, verifier 및 기타 보안 도구 역시 검사 결과의 신뢰성에 영향을 주는 핵심 구성요소로 간주하여 출처와 버전을 관리하고, 가능한 경우 digest·signature 등을 검증한다. 검사 이미지와 기반 도구는 자동으로 최신 버전으로 교체하지 않으며, 검증된 버전을 유지하다가 새로운 버전을 별도로 검증한 후 통과한 경우에만 교체한다.
```

## D-012 검사 환경 이미지 및 기반 도구 신뢰 정책 확정

Artifact 검사를 수행하는 컨테이너 이미지, VM 이미지와 기타 기반 실행 환경도 외부 Software Artifact로 간주하며 무조건 신뢰하지 않는다.

- 가능한 경우 이미지와 기반 구성요소를 버전 태그와 digest로 고정한다.
- 제공되는 publisher signature, provenance와 attestation을 검증한다.
- 검증된 이미지·기반 도구의 digest나 주요 구성요소가 변경되면 기존 검증 결과를 승계하지 않고 변경 내용을 재검증한다.
- 검사 환경에는 필요한 구성요소만 포함하고 불필요한 패키지·권한·서비스를 최소화한다.
- scanner, verifier와 기타 보안 도구도 결과 신뢰성에 영향을 주는 핵심 구성요소로 취급한다.
- 보안 도구의 출처·버전을 관리하고 가능한 경우 digest·signature를 검증한다.
- 검사 이미지와 기반 도구는 자동으로 최신 버전으로 교체하지 않는다.
- 검증된 버전을 유지하고, 새 버전은 별도 검증 후 통과한 경우에만 교체한다.

| 항목 | 상태 | 결정 |
| --- | --- | --- |
| 검사 환경 이미지 신뢰 | 결정 | 외부 Artifact로 취급하고 digest·signature·provenance·attestation을 가능한 범위에서 검증 |
| 기반 도구 신뢰 | 결정 | scanner·verifier를 포함해 출처·버전·digest·signature를 관리 |
| 이미지·도구 업데이트 | 결정 | 자동 최신화 금지; 새 버전 별도 검증 후 교체 |
| 검사 환경 구성 | 결정 | 필요한 구성요소만 포함하고 패키지·권한·서비스를 최소화 |

#### D-012 구현 영향

- 검사 이미지와 기반 도구의 immutable reference, digest, provenance, signature와 검증 시점을 manifest에 기록한다.
- 이미지나 도구의 digest·주요 구성요소 변경을 감지하면 기존 `ALLOW`나 검증 receipt를 자동 승계하지 않는다.
- scanner·verifier 자체의 버전과 결과를 receipt에 연결한다.
- 이미지 갱신은 후보 다운로드 → 격리 검증 → 비교·정책 판정 → 승인된 버전 교체의 별도 흐름으로 둔다.

#### D-012 누락 점검

- [x] 이미지·VM·기반 실행 환경을 외부 Artifact로 취급
- [x] 버전 태그와 digest 고정
- [x] publisher signature·provenance·attestation 검증
- [x] digest·주요 구성요소 변경 시 재검증
- [x] 최소 구성요소·권한·서비스 원칙
- [x] scanner·verifier의 출처·버전·digest·signature 관리
- [x] 자동 최신화 금지와 별도 검증 후 교체

## 결정 로그: D-012

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| D-012 | 2026-08-08 | 검사 이미지·VM·기반 도구·scanner·verifier를 외부 Artifact로 취급하고 digest·서명·provenance를 검증하며 자동 최신화를 금지 | 검사 환경과 보안 도구 자체가 결과 신뢰성과 공격 표면에 영향을 주기 때문 | 최소 구성, 고정 버전, 별도 업데이트 검증과 통과 후 교체가 필수 조건이 됨 |

## 사용자 원문: D-013

```text
13번: 검사 결과 및 Evidence 무결성

검사 대상 Artifact가 검사 로그, Evidence, 판정 결과 및 검사 설정을 직접 수정하거나 삭제할 수 없도록 검사 대상과 관찰·기록 영역을 분리한다.

파일 접근, 프로세스 생성, 네트워크 통신, honeytoken 접근 등 주요 행위는 가능한 한 검사 대상 외부의 신뢰 가능한 관찰 계층에서 수집한다.

검사 결과에는 사용된 검사 도구와 버전, 검사 시점, Artifact identity 및 digest, 주요 관찰 결과를 함께 기록하고, Evidence는 생성 이후 변조 여부를 확인할 수 있도록 무결성을 관리한다.

필수 Evidence가 누락되거나 손상된 경우에는 검사가 정상적으로 완료된 것으로 간주하지 않으며, 앞서 정한 fail-closed 정책을 적용한다.
```

## D-013 검사 결과 및 Evidence 무결성 확정

검사 대상 Artifact가 검사 로그, Evidence, 판정 결과와 검사 설정을 직접 수정하거나 삭제할 수 없도록 검사 대상과 관찰·기록 영역을 분리한다.

파일 접근, 프로세스 생성, 네트워크 통신, honeytoken 접근 등 주요 행위는 가능한 한 검사 대상 외부의 신뢰 가능한 관찰 계층에서 수집한다.

검사 결과에는 다음 정보를 함께 기록한다.

- 사용된 검사 도구와 버전
- 검사 시점
- Artifact identity와 digest
- 주요 관찰 결과
- Evidence 무결성 확인 정보

Evidence는 생성 이후 변조 여부를 확인할 수 있도록 무결성을 관리한다. 필수 Evidence가 누락되거나 손상되면 검사를 정상 완료로 간주하지 않고 D-010의 fail-closed 정책을 적용한다.

| 항목 | 상태 | 결정 |
| --- | --- | --- |
| 검사 대상의 기록 영역 접근 | 결정 | Artifact가 로그·Evidence·판정 결과·검사 설정을 수정·삭제할 수 없도록 분리 |
| 주요 행위 관찰 | 결정 | 가능한 경우 Artifact 외부의 신뢰 가능한 관찰 계층에서 수집 |
| 결과 필수 메타데이터 | 결정 | 도구·버전, 검사 시점, Artifact identity·digest, 주요 관찰 결과를 기록 |
| Evidence 무결성 | 결정 | 생성 후 변조 여부를 확인할 수 있도록 관리 |
| 필수 Evidence 누락·손상 | 결정 | 정상 완료로 간주하지 않고 `MANUAL_REVIEW` 또는 `BLOCK` 적용 |

#### D-013 구현 영향

- 검사 대상 workspace와 observer·evidence store를 별도 영역·권한·프로세스로 분리한다.
- 관찰 계층은 Artifact가 결과 파일을 직접 쓰는 대신 외부 collector가 이벤트를 수집하도록 설계한다.
- receipt에 scanner·observer·버전·시점·Artifact identity·digest를 연결한다.
- Evidence manifest와 digest 또는 동등한 무결성 확인 정보를 생성한다.
- 필수 Evidence 목록과 completeness 검사를 deterministic validator로 수행한다.

#### D-013 누락 점검

- [x] Artifact와 관찰·기록 영역 분리
- [x] 로그·Evidence·판정 결과·검사 설정의 직접 수정·삭제 방지
- [x] 파일·프로세스·네트워크·honeytoken 행위의 외부 관찰 계층 수집
- [x] 도구·버전·시점·Artifact identity·digest·주요 관찰 결과 기록
- [x] Evidence 생성 후 변조 확인
- [x] 필수 Evidence 누락·손상 시 fail-closed

## 결정 로그: D-013

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| D-013 | 2026-08-08 | 검사 대상과 관찰·기록 영역을 분리하고 Evidence 무결성과 필수 completeness를 검증 | Artifact가 자기 로그·판정·설정을 조작하면 검사 결과를 신뢰할 수 없기 때문 | 외부 observer, Evidence manifest·무결성 검증, 누락 시 fail-closed가 필수 조건이 됨 |

## D-015 Host trusted execution과 privileged control boundary

M7 이후 external security review에서 검사 runtime 자체의 version lock만으로는
Host command execution을 신뢰할 수 없다는 gap이 확인되었다. ambient `PATH`,
Host environment, Docker context/endpoint 또는 교체 가능한 executable이 runtime
선택과 firewall enforcement를 바꿀 수 있으면 D-012의 tool identity와 D-013의
Evidence completeness가 성립하지 않는다.

- production Host tool은 validated absolute identity로만 실행하고 child
  environment를 command별 최소 allowlist로 새로 만든다.
- Docker는 local transport, endpoint ownership와 daemon/runtime capability를
  확인한 configured endpoint만 허용한다. arbitrary remote daemon과 ambient
  `DOCKER_HOST`/`DOCKER_CONTEXT`는 신뢰 경계 밖이다.
- HAA가 필요로 하는 Host mutation 권한은 전체 CLI에 부여하지 않고 좁은 별도
  helper/service에 둔다.
- privileged helper는 typed resolver-policy operation만 수행하며 package name,
  hostname, shell fragment, executable path와 raw firewall argument를 받지 않는다.
- gVisor observer helper는 HAA supervisor가 identity, readiness, exclusive endpoint,
  connection attribution와 termination을 확인한다.
- production transport가 생성하지 못하는 observation은 clean Evidence가 아니며
  required capability 누락은 fail-closed다.

M3/M11 dynamic inspection은 runtime 중 helper를 설치하거나 UID 0 bootstrap을
유지하지 않는다. HAA는 exact upstream runtime image digest를 base로 재현적으로
빌드한 HAA-owned immutable inspection image를 사용한다. Node inspection image는
npm과 GitHub ELF가 공유하고, Python inspection image는 PyPI, PyTorch와 active
sdist builder가 공유한다. 각 image에는 고정된 `/haa-runtime/haa-boundary`가
`root:root`, mode `0555`로 baked-in 되며, derived image digest, upstream base
digest, helper content, build provenance/signature와 release identity를 기존
trusted-tool distribution contract로 검증·고정한다. Sandbox는 image start부터
`1000:1000`으로 실행하고, writable `/haa-runtime`, runtime `docker cp`, Host bind
mount 및 persistent privileged init을 사용하지 않는다. 이 image identity와
provenance는 M3/M11 canonical contract가 참조하며, trusted-tool 검증 원칙 자체는
이 문서가 소유한다.

### D-012/D-015 trusted-tool behavioral attribution scope

digest·provenance와 identity가 검증된 installer/runtime 및 HAA control runtime은
이 문서가 정의하는 trusted-tool boundary 안에 있다. M3/M11의 behavioral
attribution은 우선적으로 Artifact-controlled execution을 대상으로 한다. 따라서
검증된 installer/runtime이 hostile Artifact bytes를 처리하는 중 그 도구 자체가
exploit되는 경우는 trusted-tooling residual risk이며, M3/M11이 그 원인을 Artifact
lifecycle과 구별해 보장하는 범위는 아니다.

이는 도구 행위를 광범위하게 suppress하는 권한이 아니다. sandbox containment,
tool identity·pinning·provenance 검증, external observation 및 evidence
completeness의 fail-closed 요구는 그대로 적용된다. M3/M11에서 허용되는
trusted-control attribution은 exact HAA-created one-shot control root처럼
별도 contract가 검증한 좁은 사실에만 근거해야 하며, child·re-exec·unknown
attribution으로 상속되지 않는다.

그 좁은 사실의 provenance는 M3/M11 contract가 정의한다. OCI root는 검증된
`container/start` ContextData로, Docker exec root는 검증된 HAA direct-exec
launcher와 complete clone provenance로만 확정한다. `OriginExec`,
`parent_thread_group_id == 0`, pathname 또는 process class만으로는 trusted
root가 되지 않는다. HAA handoff helper는 privilege를 부여하는 인증 marker가
아니라 `CONTROL → ARTIFACT`만 수행하는 irreversible trust-removal marker다.
따라서 Artifact가 이를 다시 호출해도 trust를 얻지 못한다. 이 문서는
helper의 검증·고정·최소권한 원칙을 소유하고, M3/M11은 그 원칙을 반복하지
않고 관찰 귀속 계약만 소유한다.

구체 lifecycle, API bound와 M8 acceptance는
[M8 Production Trust Hardening Contract](../planning/11-m8-production-trust-hardening-contract.md)가
소유한다.

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| D-015 | 2026-08-24 | Host tool·local daemon·privileged helper·observer supervisor를 명시적 trusted execution boundary로 검증하고 ambient process state를 신뢰하지 않음 | 검증되지 않은 command/environment가 격리와 network enforcement 성공을 위조하지 못하게 하기 위해 | M8 release-blocking Host tool executor, typed privilege separation, observer lifecycle와 hostile-boundary regression이 필요 |
