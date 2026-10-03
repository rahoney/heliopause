# MVP Scope — Inspection and Verification

## 사용자 원문: MVP-005

```text
5번: MVP 정적 검사 범위

MVP에서는 Artifact를 설치·실행하기 전에 출처·identity·무결성, metadata와 내부 구조, 설치·실행 코드, dependency, 이전 버전 대비 변경사항 및 악성·의심 패턴을 정적으로 검사한다.

npm lifecycle script, Python build/install 코드, shell script 등 자동 또는 간접적으로 실행될 수 있는 코드를 식별하며, executable·symlink·hardlink·비정상 경로와 archive 구조 등 파일 수준의 위험 요소도 검사한다.

Dependency는 직접 및 전이 의존성을 가능한 범위에서 분석하고, 신규·비정상 dependency, 알려진 취약점, manifest/lock 정보와 실제 구성의 불일치 등을 확인한다.

기존 버전이 존재하는 경우 script·dependency·실행 파일·파일 구조·권한 등 주요 변경사항을 비교하여 Publisher compromise나 비정상 release를 탐지하기 위한 Evidence로 활용한다.

Credential 탐색, 외부 전송, 난독화, 동적 코드 실행, process/shell 실행, 비정상 파일 조작 등 알려진 악성·의심 행위를 탐지 대상으로 포함한다.

정적 검사는 Heliopause 자체 검사와 외부 scanner·database 등을 조합할 수 있도록 설계하며, MVP Scope 단계에서는 특정 검사 도구를 확정하지 않는다. 각 검사 결과는 공통 Finding/Evidence 형태로 정규화하여 이후 Policy 판정에 사용한다.
```

## 구조화된 결정

### MVP-005 정적 검사 범위 확정

정적 검사는 Artifact를 설치·실행하기 전에 수행하며, 다음 범주를 MVP의 검사 대상으로 포함한다.

| 검사 범주 | 검사 내용 |
| --- | --- |
| 출처·identity·무결성 | provenance·출처, Artifact identity, checksum/digest·서명 등 무결성 근거 |
| metadata·내부 구조 | metadata, manifest, archive와 파일 구조, 형식·구성 일치 |
| 설치·실행 코드 | npm lifecycle script, Python build/install 코드, shell script 등 자동·간접 실행 코드 |
| 파일 수준 위험 | executable, symlink, hardlink, 비정상 경로, archive 구조 |
| dependency | 직접·전이 dependency, 신규·비정상 dependency, 알려진 취약점, manifest/lock과 실제 구성 불일치 |
| 이전 버전 비교 | script, dependency, 실행 파일, 파일 구조, 권한 등의 주요 변경 |
| 악성·의심 패턴 | Credential 탐색, 외부 전송, 난독화, 동적 코드 실행, process/shell 실행, 비정상 파일 조작 |

Dependency 분석은 가능한 범위에서 직접 및 전이 의존성을 모두 다룬다. 기존 버전이 있으면 주요 변경을 비교하고, Publisher compromise 또는 비정상 release 탐지를 위한 Evidence로 활용한다.

### MVP-005 검사 도구와 결과 계약

- 정적 검사는 Heliopause 자체 검사와 외부 scanner·database를 조합할 수 있도록 설계한다.
- MVP Scope 단계에서는 특정 정적 검사 도구나 database를 확정하지 않는다.
- 각 검사 결과는 공통 `Finding`/`Evidence` 형태로 정규화한다.
- 정규화된 Finding/Evidence는 이후 Policy 판정에 사용한다.
- 검사 도구 선택·버전·database 상태는 결과 신뢰성에 영향을 주므로 후속 결정에서 별도 관리한다.

### MVP-005 구현 영향

- 정적 검사 pipeline은 설치·실행 단계와 분리된 선행 단계로 정의한다.
- npm, Python, shell 등 실행 가능 코드의 식별 결과에 위치·종류·실행 가능 경로를 연결한다.
- 파일 수준 결과에는 executable·link type·경로·archive entry와 위험 신호를 기록한다.
- Dependency 분석 결과에는 direct/transitive 구분, source·version·digest 가능 여부, manifest/lock 일치 여부와 vulnerability 근거를 기록한다.
- 이전 버전 비교는 기준 버전 identity/digest와 비교 대상 identity/digest를 함께 보존하고 변경 Evidence를 생성한다.
- 악성·의심 패턴은 탐지 규칙, 위치, 관찰 근거를 공통 Finding으로 반환한다.
- 외부 scanner·database 연동은 adapter 또는 provider 경계에 두고 Core는 결과 계약만 의존한다.
- 특정 도구가 없거나 결과가 불충분하면 안전으로 간주하지 않고 앞서 정한 `MANUAL_REVIEW` 또는 위험 수준에 따른 `BLOCK` 정책을 적용한다.

### MVP-005 누락 점검

- [x] 설치·실행 전 출처·identity·무결성 정적 검사
- [x] metadata와 내부 구조 검사
- [x] 설치·실행 코드 식별
- [x] npm lifecycle script 식별
- [x] Python build/install 코드 식별
- [x] shell script 식별
- [x] executable·symlink·hardlink·비정상 경로·archive 구조 검사
- [x] 직접·전이 dependency 분석
- [x] 신규·비정상 dependency와 알려진 취약점 확인
- [x] manifest/lock 정보와 실제 구성 불일치 확인
- [x] 기존 버전 대비 script·dependency·실행 파일·구조·권한 변경 비교
- [x] Publisher compromise·비정상 release 탐지 Evidence 활용
- [x] Credential 탐색·외부 전송·난독화·동적 코드 실행 탐지
- [x] process/shell 실행·비정상 파일 조작 탐지
- [x] Heliopause 자체 검사와 외부 scanner·database 조합 가능
- [x] MVP Scope에서 특정 검사 도구 미확정
- [x] Finding/Evidence 공통 정규화 및 Policy 연계

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-005 | 2026-08-08 | 설치·실행 전 출처·구조·코드·dependency·변경·악성 패턴을 정적으로 검사하고 결과를 Finding/Evidence로 정규화하되 특정 도구는 후속 결정으로 남김 | 설치 전 위험을 폭넓게 탐지하면서도 특정 scanner 종속을 피하고 교체 가능한 검사 구성을 확보하기 위해 | 정적 검사 pipeline, dependency·diff 분석, 악성 패턴 규칙, 공통 결과 계약과 후속 도구 선택이 MVP 조건이 됨 |

## 사용자 원문: MVP-006

```text
6번: MVP 동적 설치·실행 검사 범위

MVP에서는 정적 검사를 통과하거나 추가 행위 검증이 필요한 Artifact를 격리 환경에서 설치하고, 필요한 경우 제한적으로 실행하여 동적 행위를 관찰한다.

설치·실행 과정에서는 파일 생성·수정·삭제, 프로세스 및 자식 프로세스 생성, shell·외부 명령 실행, 네트워크·DNS 통신 시도, 환경변수·Credential·honeytoken 접근, 민감 경로 탐색, 권한 상승 시도, 자원 사용량 등을 관찰한다.

설치 전후 filesystem 및 주요 상태의 변화를 비교하여 새로 생성된 executable, script, 설정 및 비정상적인 변경을 Evidence로 기록한다.

npm lifecycle script와 Python build/install 과정처럼 설치 자체에서 코드 실행이 발생하는 경우 이를 동적 검사의 핵심 대상으로 포함한다.

GitHub Releases의 binary 등은 현재 Linux 검사 backend에서 안전하게 실행 가능한 경우에 한해 제한적으로 실행한다. 실행할 수 없거나 필수 동적 검사가 완료되지 않은 경우에는 그 한계를 결과에 명시하고 자동 ALLOW하지 않는다.

동적 검사는 실제 Host 자산과 분리된 모의 filesystem, honeytoken 및 통제된 네트워크 환경에서 수행하며, 앞서 정의한 resource limit·timeout·fail-closed 정책을 적용한다.
```

## 구조화된 결정

### MVP-006 동적 설치·실행 검사 확정

정적 검사를 통과했거나 추가 행위 검증이 필요한 Artifact는 격리 환경에서 설치하고, 필요한 경우 제한적으로 실행하여 동적 행위를 관찰한다. 동적 검사는 실제 Host 자산과 분리된 모의 환경에서 수행한다.

| 관찰 범주 | MVP 관찰 대상 |
| --- | --- |
| filesystem | 파일 생성·수정·삭제, 설치 전후 filesystem 및 주요 상태 변화 |
| process | 프로세스·자식 프로세스 생성, shell·외부 명령 실행 |
| network | 네트워크·DNS 통신 시도 |
| secret·경로 | 환경변수·Credential·honeytoken 접근, 민감 경로 탐색 |
| privilege·resource | 권한 상승 시도, CPU·메모리·저장공간·프로세스 등 자원 사용량 |
| 설치 코드 | npm lifecycle script, Python build/install 과정의 코드 실행 |

설치 전후 변화를 비교하여 새로 생성된 executable·script·설정과 비정상 변경을 Evidence로 기록한다. 설치 자체에서 코드 실행이 발생하는 npm lifecycle script와 Python build/install은 동적 검사의 핵심 대상으로 취급한다.

### MVP-006 Artifact별 실행 한계

- GitHub Releases binary 등은 현재 Linux 검사 backend에서 안전하게 실행 가능한 경우에 한해 제한적으로 실행한다.
- 실행할 수 없거나 필수 동적 검사가 완료되지 않은 경우 결과에 검사 한계를 명시한다.
- 실행 불가 또는 필수 검사 미완료를 자동 `ALLOW`로 처리하지 않으며, 앞서 정한 `MANUAL_REVIEW` 또는 위험 수준에 따른 `BLOCK` 정책을 적용한다.

### MVP-006 격리·관찰·제한 정책

- 실제 Host filesystem·Credential·환경변수·네트워크와 분리된 모의 filesystem, honeytoken 및 통제된 네트워크 환경에서 수행한다.
- 동적 검사는 앞서 정의한 resource limit과 timeout을 모든 설치·실행 단계에 적용한다.
- 제한 초과나 강제 종료로 필수 관찰이 완료되지 않은 경우 fail-closed 정책을 적용한다.
- 관찰 계층은 Artifact가 직접 수정할 수 없는 외부 영역에 기록하고, 행위와 상태 변화의 근거를 Evidence로 남긴다.

### MVP-006 구현 영향

- 설치·실행 orchestration은 정적 검사 결과와 동적 검사 필요 여부를 입력으로 받아 격리 backend를 선택한다.
- 동적 세션에는 실행 OS·backend·격리 구성·resource limit·timeout·네트워크 정책을 기록한다.
- filesystem/process/network/secret/privilege/resource 관찰 결과를 공통 Finding/Evidence로 정규화한다.
- 설치 전 snapshot과 설치 후 snapshot을 비교하여 파일·권한·프로세스·주요 상태 diff를 생성한다.
- lifecycle/build script가 실행된 경우 script identity·위치·호출 경로·관찰 행위를 연결한다.
- binary 실행 가능성, 필수 동적 검사 완료 여부와 검사 한계를 Policy 입력으로 제공한다.
- 모의 Host 환경과 honeytoken 접근, 가짜 DNS/HTTP 등 통제 네트워크 관찰 fixture를 준비한다.

### MVP-006 누락 점검

- [x] 정적 통과 또는 추가 행위 검증 필요 Artifact의 격리 설치
- [x] 필요한 경우 제한적 실행 및 동적 행위 관찰
- [x] 파일 생성·수정·삭제 관찰
- [x] 프로세스·자식 프로세스 및 shell·외부 명령 실행 관찰
- [x] 네트워크·DNS 통신 시도 관찰
- [x] 환경변수·Credential·honeytoken 접근 관찰
- [x] 민감 경로 탐색과 권한 상승 시도 관찰
- [x] 자원 사용량 관찰
- [x] 설치 전후 filesystem·주요 상태 변화 비교
- [x] 새 executable·script·설정·비정상 변경 Evidence 기록
- [x] npm lifecycle script와 Python build/install의 설치 코드 실행을 핵심 대상으로 포함
- [x] Linux backend에서 안전하게 실행 가능한 GitHub Releases binary만 제한 실행
- [x] 실행 불가·필수 동적 검사 미완료 시 한계 명시와 자동 `ALLOW` 금지
- [x] 모의 filesystem·honeytoken·통제 네트워크 사용
- [x] resource limit·timeout·fail-closed 적용

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-006 | 2026-08-08 | 정적 통과 또는 추가 검증 필요 Artifact를 Linux 격리 backend에서 설치·제한 실행하고 행위·상태 변화·Evidence를 관찰하며, 실행 불가·미완료는 자동 `ALLOW`하지 않음 | 설치·실행 순간의 공급망 행위를 실제 Host와 분리해 관찰하고 동적 검증의 한계를 명시하기 위해 | 동적 orchestration, 모의 Host·통제 네트워크, 외부 관찰 계층, resource/timeout과 fail-closed가 MVP 조건이 됨 |

## 사용자 원문: MVP-007

```text
7번: Signature / Provenance / 무결성 검증 범위
Checksum / Digest
Artifact의 동일성과 무결성을 확인하는 기본 식별 수단으로 사용한다.
Registry integrity 정보
npm, PyPI 등 Registry가 제공하는 integrity·hash·metadata를 검증 입력으로 활용하되, 이것만으로 Artifact의 안전성을 확정하지 않는다.
Publisher signature
제공되는 경우 검증하며, 정상 서명만으로 안전성을 보증하지 않는다.
Provenance
Artifact의 출처와 가능한 경우 빌드·배포 과정을 확인하여 신뢰 판단의 Evidence로 활용한다.
Attestation
제공되는 경우 검증하고, 어떤 빌드·정책·환경에 대한 attestation인지 함께 기록한다.
GitHub Releases의 서명·체크섬
Release Asset과 함께 공식 checksum·signature 등이 제공되는 경우 검증하고 실제 다운로드 Artifact와 일치하는지 확인한다.
Signature / Provenance가 없는 Artifact
서명이나 provenance가 없다는 이유만으로 자동 BLOCK하지 않는다. 대신 해당 검증 근거가 없음을 결과에 명시하고, digest·출처·정적·동적 검사 등 다른 Evidence를 종합하여 판정한다.
다만 정책상 필수로 요구되는 검증 정보가 없는 경우에는 자동 ALLOW하지 않고 MANUAL_REVIEW 또는 BLOCK으로 처리할 수 있다.
```

## 구조화된 결정

### MVP-007 Signature·Provenance·무결성 검증 확정

| 검증 입력 | MVP 처리 | 판정상 의미 |
| --- | --- | --- |
| Checksum / Digest | Artifact 동일성과 무결성을 확인하는 기본 식별 수단으로 사용 | 검사 대상·staging·반입 대상의 동일성 확인 |
| Registry integrity 정보 | npm·PyPI 등 Registry가 제공하는 integrity·hash·metadata를 검증 입력으로 활용 | Registry 정보만으로 안전성 확정 금지 |
| Publisher signature | 제공되는 경우 검증 | 정상 서명만으로 안전성 보증 금지 |
| Provenance | 출처와 가능한 빌드·배포 과정을 확인 | 신뢰 판단의 Evidence로 활용 |
| Attestation | 제공되는 경우 검증 | 대상 build·policy·environment를 함께 기록 |
| GitHub Releases checksum·signature | 공식 Release Asset과 함께 제공되는 checksum·signature 검증 | 실제 다운로드 Artifact와 일치 여부 확인 |

모든 검증 입력은 단독 안전 보증 수단이 아니라 출처·정적·동적 결과와 함께 종합하는 Evidence로 취급한다.

### MVP-007 Signature·Provenance 부재 처리

- Signature나 provenance가 없다는 이유만으로 자동 `BLOCK`하지 않는다.
- 부재한 검증 근거를 결과에 명시한다.
- Digest·출처·정적·동적 검사 등 다른 Evidence를 종합하여 판정한다.
- 정책상 필수 검증 정보가 없으면 자동 `ALLOW`하지 않는다.
- 필수 정보 부재는 `MANUAL_REVIEW` 또는 위험 수준에 따른 `BLOCK`으로 처리할 수 있다.

### MVP-007 구현 영향

- 검증 결과 모델에 검증 입력 유형, 제공 여부, 검증 상태, 대상 identity/digest, 발행자·출처, 관련 Artifact를 기록한다.
- Registry integrity·Publisher signature·provenance·attestation을 각각 독립적인 Evidence로 보존하고 서로 혼합해 단일 boolean 신뢰 값으로 축약하지 않는다.
- Attestation은 어떤 build·policy·environment에 대한 것인지 metadata를 함께 기록한다.
- GitHub Releases adapter는 Release Asset과 공식 checksum·signature를 연결하고 실제 다운로드 결과와 digest를 비교한다.
- Signature·provenance 부재와 검증 실패를 구분해 기록한다.
- staging 승격·사용자 환경 반입 전 digest를 재확인하고 검사 시점의 Artifact와 동일한지 확인한다.
- Policy 계층에서 “서명 없음”, “provenance 없음”, “정책상 필수 정보 없음”을 서로 다른 조건으로 평가한다.

### MVP-007 누락 점검

- [x] Checksum/Digest를 동일성·무결성 기본 수단으로 사용
- [x] npm·PyPI 등 Registry integrity·hash·metadata 활용
- [x] Registry 정보만으로 안전성 확정하지 않음
- [x] 제공되는 Publisher signature 검증
- [x] 정상 서명만으로 안전성 보증하지 않음
- [x] Artifact 출처와 가능한 빌드·배포 과정 provenance 확인
- [x] 제공되는 attestation 검증
- [x] attestation의 build·policy·environment 기록
- [x] GitHub Releases 공식 checksum·signature 검증
- [x] 실제 다운로드 Artifact와 Release Asset 검증 정보 일치 확인
- [x] Signature/provenance 부재만으로 자동 `BLOCK`하지 않음
- [x] 부재한 검증 근거를 결과에 명시
- [x] digest·출처·정적·동적 Evidence 종합 판정
- [x] 정책상 필수 검증 정보 부재 시 자동 `ALLOW` 금지
- [x] 필수 정보 부재의 `MANUAL_REVIEW` 또는 `BLOCK` 처리

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-007 | 2026-08-08 | digest를 기본 무결성 식별자로 삼고 signature·provenance·attestation·Registry 정보를 가능한 경우 검증하되, 부재와 실패를 구분해 종합 판정 | 단일 신뢰 신호에 의존하지 않고 Artifact 동일성·출처·빌드 근거를 함께 평가하기 위해 | 검증 입력별 Evidence 모델, Release Asset 일치 확인, 필수 정보 부재의 `MANUAL_REVIEW`/`BLOCK` 정책이 MVP 조건이 됨 |
