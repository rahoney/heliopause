# Heliopause Artifact Airlock MVP Scope

이 문서는 `Heliopause Artifact Airlock`(`helox`) MVP의 범위에 관한 사용자 원문, 구조화된 결정, 구현 영향과 경계를 보존하는 상세 기준 문서다.

## 문서 사용 규칙

- 사용자 원문의 의미, 조건, 예외와 범위를 생략하지 않는다.
- 각 결정은 사용자 원문 → 구조화된 결정 → 구현 영향 → 누락 점검 순서로 기록한다.
- 상세 내용은 이 문서를 canonical source로 삼고 중앙 결정 색인에는 상태·요약·링크만 둔다.

## 사용자 원문 보존

### MVP-001 원문

```text
MVP Scope
1번: 최초 지원 Artifact 생태계

처음부터 특정 package manager에 종속되지 않는 다중 Artifact 생태계 지원을 전제로 설계한다.

MVP에서는 npm을 기준 end-to-end 구현 대상으로 삼고, Python/PyPI(pip)를 두 번째 package ecosystem으로 지원하여 package manager adapter의 확장성을 실제로 검증한다.

또한 GitHub Releases를 일반 Software Artifact source로 지원하여 package manager를 통하지 않는 binary/archive 등의 반입 흐름도 검증한다.

npm, PyPI/pip, GitHub Releases 각각의 특수 로직은 adapter에 한정하고, Artifact identity, acquisition, verification, quarantine, scanning, policy, evidence, staging, promotion 등의 공통 기능에는 특정 생태계의 개념을 직접 포함하지 않는다.

MVP에서 세 대상의 기능 수준을 반드시 동일하게 맞추지는 않으며, npm을 reference implementation으로 가장 완전하게 구현하고 PyPI/pip와 GitHub Releases를 통해 공통 구조의 확장성을 검증한다.

확장 범위는 이 세 가지에 한정하지 않는다. 향후 다양한 package manager, registry, release source 및 기타 Software Artifact 유형을 추가할 수 있도록 adapter 기반의 modular 구조와 명확한 공통 계약을 유지하며, 특정 생태계에 종속되지 않는 유지보수·확장 가능한 설계와 구현을 원칙으로 한다.

MVP의 확장성 검증은 동일한 공통 Core와 기존 계약의 의미를 깨뜨리지 않고 npm, PyPI/pip, GitHub Releases의 adapter를 연결해 각 흐름이 동작하는 것을 기준으로 한다. 새로운 생태계를 추가하기 위해 공통 Core에 특정 생태계 전용 로직을 추가하거나 기존 adapter와의 호환성을 깨뜨려야 한다면 구조를 재검토한다. 범용적으로 필요한 계약 확장은 후방호환성을 유지하는 범위에서 허용한다. 해당 내용을 정리하는건 좋은데 이전처럼 원문 내용을 누락하는 일이 발생하지 않도록 주의해.
```

## 구조화된 결정

### MVP-001 최초 지원 Artifact 생태계 확정

HAA는 처음부터 특정 package manager에 종속되지 않는 다중 Artifact 생태계 지원을 전제로 설계한다.

| 대상 | MVP 역할 | 기능 수준 |
| --- | --- | --- |
| npm | 기준 end-to-end reference implementation | 세 대상 중 가장 완전하게 구현 |
| Python/PyPI(pip) | 두 번째 package ecosystem과 package manager adapter 확장성 검증 | 공통 구조 검증에 필요한 수준 |
| GitHub Releases | package manager를 통하지 않는 일반 Software Artifact source와 binary/archive 반입 흐름 검증 | 공통 구조 검증에 필요한 수준 |

세 대상의 기능 수준을 MVP에서 반드시 동일하게 맞추지는 않는다. npm은 reference implementation으로 가장 완전하게 구현하고, PyPI/pip와 GitHub Releases는 동일한 공통 Core와 계약을 사용해 구조의 실제 확장성을 검증한다.

### 공통 Core와 Adapter 경계

npm, PyPI/pip, GitHub Releases의 특수 로직은 각 adapter에 한정한다. 다음 공통 기능에는 특정 생태계의 개념을 직접 포함하지 않는다.

- Artifact identity
- acquisition
- verification
- quarantine
- scanning
- policy
- evidence
- staging
- promotion

공통 Core는 package manager, registry, release source와 기타 Software Artifact 유형이 추가될 수 있는 명확한 공통 계약을 제공한다. 특정 생태계 전용 동작과 metadata 변환은 adapter가 담당한다.

### 확장성 판정 기준

MVP의 확장성은 다음 조건으로 검증한다.

1. npm, PyPI/pip, GitHub Releases adapter가 동일한 공통 Core에 연결된다.
2. 세 adapter의 각 대상 흐름이 기존 공통 계약의 의미를 깨뜨리지 않고 동작한다.
3. 새 생태계를 위해 공통 Core에 특정 생태계 전용 로직을 추가하지 않는다.
4. 새 adapter 추가가 기존 adapter와의 호환성을 깨뜨리지 않는다.
5. 범용적으로 필요한 계약 확장은 후방호환성을 유지하는 범위에서만 허용한다.
6. 위 조건을 지킬 수 없다면 adapter를 우회하거나 예외를 누적하지 않고 구조를 재검토한다.

지원 범위는 npm, PyPI/pip, GitHub Releases에 고정되지 않는다. MVP 이후 다양한 package manager, registry, release source와 기타 Software Artifact 유형을 추가할 수 있는 유지보수·확장 가능한 modular 구조를 유지한다.

## 구현 영향

- 공통 Core의 domain contract는 생태계 중립적인 Artifact identity와 단계별 요청·결과를 정의해야 한다.
- npm, PyPI/pip, GitHub Releases를 별도 adapter로 구현하고 Core가 adapter의 세부 type을 직접 참조하지 않도록 경계를 둔다.
- npm end-to-end 흐름을 reference implementation과 기준 fixture로 사용한다.
- PyPI/pip adapter는 package manager별 resolution·distribution 차이를 Core 밖에서 흡수할 수 있는지 검증한다.
- GitHub Releases adapter는 package manager 없이 binary/archive를 acquisition·verification·quarantine·staging·promotion할 수 있는지 검증한다.
- 공통 contract test를 세 adapter에 동일하게 적용하고 생태계별 추가 contract test를 adapter 내부에 둔다.
- dependency direction과 금지 import를 architecture test로 검사할 수 있도록 경계를 명시한다.
- 기능 수준 차이는 capability matrix로 공개해 지원하지 않는 기능을 성공으로 오인하지 않도록 한다.
- 공통 계약 변경에는 후방호환성 검사를 적용하고, 기존 adapter contract를 깨뜨리는 변경은 구조 재검토 대상으로 분류한다.

## 누락 점검

- [x] 처음부터 다중 Artifact 생태계를 전제로 설계
- [x] npm을 가장 완전한 end-to-end reference implementation으로 지정
- [x] Python/PyPI(pip)를 두 번째 package ecosystem으로 지원
- [x] PyPI/pip를 통해 package manager adapter 확장성을 실제 검증
- [x] GitHub Releases를 일반 Software Artifact source로 지원
- [x] binary/archive의 package manager 없는 반입 흐름 검증
- [x] 생태계별 특수 로직을 adapter로 제한
- [x] Artifact identity부터 promotion까지 공통 기능에 생태계 전용 개념을 직접 포함하지 않음
- [x] 세 대상의 MVP 기능 동등성을 강제하지 않음
- [x] npm은 완전도, PyPI/pip·GitHub Releases는 공통 구조 확장성 검증에 초점
- [x] 향후 다른 package manager·registry·release source·Artifact 유형 확장 가능
- [x] 동일 Core와 기존 계약의 의미를 유지하는 것을 확장성 기준으로 사용
- [x] 새 생태계 때문에 Core 전용 로직이나 기존 adapter 비호환이 필요하면 구조 재검토
- [x] 범용 계약 확장은 후방호환 범위에서 허용

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-001 | 2026-08-08 | npm을 reference implementation으로 구현하고 PyPI/pip·GitHub Releases adapter로 생태계 중립 Core의 확장성을 검증 | package manager와 일반 binary/archive source를 함께 지원하면서 특정 생태계 종속을 방지하기 위해 | adapter 계약, capability matrix, 공통 contract test와 후방호환성 검사가 MVP 필수 조건이 됨 |

## 사용자 원문: MVP-002

```text
2번: 지원 OS 및 검사 실행 환경

MVP의 격리 검역 및 동적 실행 환경은 Linux를 기준으로 구현한다. Heliopause의 주요 대상이 package manager, CLI, build tool, container, Helm 등 개발 생태계의 Software Artifact라는 점과 격리·관찰·권한 통제 환경의 구현 효율성을 고려하여 Linux를 최초 기준 환경으로 삼는다.

다만 Heliopause Core와 adapter 계약은 Linux에 종속되지 않도록 설계하며, 향후 macOS 및 Windows 검사 backend를 추가할 수 있는 구조를 유지한다.

npm·PyPI 등 다중 OS에서 동작하는 Artifact의 경우 Linux 환경에서 수행 가능한 검사를 우선 지원한다. macOS 또는 Windows에서만 나타나는 플랫폼 종속 실행 행위를 Linux 검사 결과만으로 안전하다고 보증하지 않는다.

Linux에서 동적 검사가 불가능한 플랫폼 전용 Artifact는 가능한 정적·출처·무결성 검사를 수행하되, 필수 동적 검증이 필요한 경우 ALLOW로 자동 판정하지 않고 검사 한계를 명시하거나 MANUAL_REVIEW로 처리한다.

macOS·Windows 네이티브 동적 검역 환경은 MVP 이후 확장 대상으로 둔다.
```

## 구조화된 결정

### MVP-002 지원 OS 및 검사 실행 환경 확정

MVP의 격리 검역과 동적 실행 환경은 Linux를 최초 기준 환경으로 구현한다. 대상이 package manager, CLI, build tool, container, Helm 등 개발 생태계의 Software Artifact이고, Linux에서 격리·관찰·권한 통제 환경을 구현하는 효율성이 높기 때문이다.

| 범위 | MVP 결정 | 한계·확장 |
| --- | --- | --- |
| 격리 검역 환경 | Linux 기준 | MVP 이후 macOS·Windows native backend 확장 |
| 동적 실행 환경 | Linux 기준 | Linux에서 관찰 가능한 행위로 범위 제한 |
| Heliopause Core | OS 비종속 계약 유지 | backend가 OS별 구현을 제공 |
| Adapter 계약 | Linux 전용 개념을 직접 포함하지 않음 | macOS·Windows 검사 backend 연결 가능 구조 |
| 다중 OS Artifact | Linux에서 가능한 검사 우선 지원 | OS 전용 실행 행위는 Linux 결과만으로 안전 보증 금지 |
| Linux 동적 검사 불가능 Artifact | 정적·출처·무결성 검사 수행 | 필수 동적 검증이 필요하면 한계 명시 또는 `MANUAL_REVIEW`, 자동 `ALLOW` 금지 |

macOS·Windows 네이티브 동적 검역 환경은 MVP 이후 확장 대상으로 둔다. Core와 adapter 계약은 이 확장을 수용할 수 있도록 유지한다.

### MVP-002 구현 영향

- Linux 격리 backend를 MVP의 첫 실행 backend로 정의한다.
- Core의 Artifact identity, acquisition, verification, quarantine, scanning, policy, evidence, staging, promotion 계약에는 OS별 실행 구현을 직접 포함하지 않는다.
- OS별 격리·관찰·권한 통제 기능은 backend 경계에 둔다.
- Artifact별 검사 결과에 실행 OS와 수행된 검사 범위를 기록한다.
- Linux에서 동적 검사가 불가능하거나 플랫폼 전용 행위를 검증하지 못한 경우 `ALLOW`를 자동 판정하지 않는다.
- macOS·Windows 전용 동적 행위가 중요하면 검사 한계를 명시하고 `MANUAL_REVIEW` 또는 위험 수준에 따른 차단으로 전환한다.
- 다중 OS Artifact fixture와 Linux에서 검증 불가능한 platform-specific fixture를 준비한다.

### MVP-002 누락 점검

- [x] Linux를 MVP 최초 격리 검역·동적 실행 기준으로 지정
- [x] package manager·CLI·build tool·container·Helm 등 대상 맥락 반영
- [x] Core와 adapter 계약의 OS 비종속성 유지
- [x] 향후 macOS·Windows backend 확장 가능 구조
- [x] 다중 OS Artifact는 Linux에서 가능한 검사 우선 지원
- [x] Linux 검사만으로 macOS·Windows 전용 행위 안전 보증 금지
- [x] Linux 동적 검사 불가 Artifact의 정적·출처·무결성 검사
- [x] 필수 동적 검증이 필요하면 자동 `ALLOW` 금지
- [x] 검사 한계 명시 또는 `MANUAL_REVIEW`
- [x] macOS·Windows native dynamic quarantine은 MVP 이후 확장

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-002 | 2026-08-08 | Linux를 MVP 격리·동적 실행 기준으로 삼되 Core·adapter 계약은 OS 비종속으로 유지 | 개발 Artifact 대상에 대한 Linux 격리·관찰·권한 통제 구현 효율성과 향후 OS 확장성을 함께 확보하기 위해 | Linux backend, OS 비종속 contract, 플랫폼 전용 검사 한계와 `MANUAL_REVIEW` 판정이 MVP 조건이 됨 |

## 사용자 원문: MVP-003

```text
3번: CLI Host 지원 OS

MVP에서 Heliopause CLI는 Linux와 macOS의 네이티브 실행을 우선 지원한다. Windows에서는 우선 WSL2 기반 Linux 환경을 공식 지원 경로로 사용하며, Windows 네이티브 실행은 MVP 이후 확장 대상으로 둔다.

CLI가 실행되는 Host OS와 실제 Artifact를 검사하는 격리 환경은 분리하며, macOS에서도 검역·동적 실행은 앞서 정의한 Linux 기반 격리 backend를 사용한다.

Core와 adapter 계약은 특정 Host OS에 종속되지 않도록 설계하여 향후 Windows 네이티브 및 다른 실행 환경을 추가할 수 있도록 한다.
```

## 구조화된 결정

### MVP-003 CLI Host 지원 OS 확정

| 범위 | MVP 결정 | 한계·확장 |
| --- | --- | --- |
| Linux Host | 네이티브 CLI 실행 우선 지원 | Linux 기반 격리 backend와 직접 연계 가능 |
| macOS Host | 네이티브 CLI 실행 우선 지원 | 검역·동적 실행은 Linux 기반 격리 backend 사용 |
| Windows Host | WSL2 기반 Linux 환경을 공식 지원 경로로 사용 | Windows 네이티브 CLI 실행은 MVP 이후 확장 |
| CLI Host와 검사 환경 | 서로 분리 | Host OS가 검사 backend의 OS를 자동으로 결정하지 않음 |
| Core·adapter 계약 | 특정 Host OS에 종속되지 않도록 설계 | Windows 네이티브 및 기타 실행 환경 추가 가능 구조 유지 |

MVP에서 CLI가 실행되는 Host OS 지원과 Artifact가 실제로 검사되는 격리 실행 환경 지원은 별개의 계약으로 관리한다. 따라서 macOS Host에서도 Artifact 검역·동적 실행은 앞서 MVP-002에서 정한 Linux 기반 격리 backend를 사용한다.

### MVP-003 구현 영향

- CLI의 Host capability와 격리 backend capability를 별도 모델과 판정 결과로 기록한다.
- Linux·macOS에서는 네이티브 CLI 실행 경로를 제공한다.
- Windows에서는 WSL2 기반 Linux를 공식 MVP 실행 경로로 문서화하고 검증한다.
- Windows 네이티브 실행을 MVP의 필수 지원 경로로 간주하지 않으며, 이후 별도 backend 또는 실행 환경 확장으로 다룬다.
- macOS Host에서 Linux 기반 격리 backend를 호출·관찰할 수 있는 연결 경계를 정의한다.
- Core와 adapter의 공통 계약에 Host OS별 분기 로직을 직접 넣지 않고, Host 실행 계층과 검사 backend 계층에서 흡수한다.
- 향후 Windows 네이티브 및 기타 실행 환경을 추가할 때 기존 Core·adapter 계약과의 호환성을 검증한다.

### MVP-003 누락 점검

- [x] Linux 네이티브 CLI 실행 우선 지원
- [x] macOS 네이티브 CLI 실행 우선 지원
- [x] Windows의 WSL2 기반 Linux 공식 지원 경로
- [x] Windows 네이티브 CLI 실행은 MVP 이후 확장
- [x] CLI Host OS와 Artifact 검사 격리 환경 분리
- [x] macOS에서도 Linux 기반 격리 backend 사용
- [x] Core·adapter 계약의 Host OS 비종속성
- [x] Windows 네이티브 및 다른 실행 환경 추가 가능 구조

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-003 | 2026-08-08 | Linux·macOS 네이티브 CLI와 Windows WSL2 기반 Linux를 MVP Host 경로로 지원하고, Artifact 검사는 분리된 Linux backend에서 수행 | CLI 사용성과 Linux 중심 격리·동적 검사의 일관성을 함께 확보하면서 Windows 네이티브·기타 환경 확장 여지를 유지하기 위해 | Host capability와 검사 backend capability 분리, WSL2 공식 경로 검증, OS 비종속 Core·adapter 계약이 MVP 조건이 됨 |

## 사용자 원문: MVP-004

```text
4번: MVP 지원 Artifact 유형

MVP에서는 npm package, Python package와 GitHub Releases를 통해 제공되는 일반 binary/archive를 실제 검사 대상으로 지원한다.

npm에서는 package artifact, Python에서는 wheel 및 source distribution을 지원하며, 일반 Artifact는 단일 executable binary와 .zip, .tar.gz 등의 archive를 우선 지원한다. 단, 동적 실행 검사는 현재 MVP의 검사 backend에서 실행 가능한 Artifact에 한정한다.

Artifact의 생태계별 파일 형식과 metadata 해석은 각 adapter가 담당하고, 공통 Core는 특정 확장자나 package 형식에 종속되지 않는다.

OS 전용 GUI installer, Docker/OCI image, IDE extension, MCP server, Git repository, CI component 및 curl | sh 형태 설치 흐름은 MVP의 직접 지원 범위에서 제외하고 향후 확장 대상으로 둔다.

지원 Artifact 유형에 해당하더라도 현재 MVP의 OS·검사 backend에서 필수 검사를 수행할 수 없는 경우에는 검사 한계를 명시하고 자동 ALLOW하지 않는다.

Python source distribution처럼 설치·빌드 과정에서 코드 실행이 발생할 수 있는 Artifact는 단순 archive가 아니라 실행 가능 설치 Artifact로 취급하여 격리된 설치·동적 검사 정책을 적용한다.

Archive는 압축 해제 자체도 검사 과정의 일부로 취급하며, 경로 이탈(path traversal), 절대경로, 비정상 symlink·hardlink, 특수 파일, 과도한 파일 수·용량 등 압축 해제 과정에서 발생할 수 있는 위험을 통제된 격리 환경에서 검사한다.

Artifact 유형은 파일 확장자만을 신뢰하여 판별하지 않고, 가능한 경우 파일 구조·metadata·content type 등 실제 형식과의 일치 여부를 확인한다. 확장자와 실제 형식이 불일치하는 경우 이를 위험 신호로 기록한다.

Archive 내부의 중첩 archive 역시 검사 대상으로 취급하며, 재귀 깊이·압축 해제 크기·파일 수 등에 제한을 적용하여 과도한 중첩이나 압축폭탄으로 인해 검사가 우회되거나 자원이 고갈되지 않도록 한다.
```

## 구조화된 결정

### MVP-004 지원 Artifact 유형 확정

| 생태계·source | MVP 지원 형식 | 검사 조건·범위 |
| --- | --- | --- |
| npm | package artifact | npm adapter가 형식·metadata를 해석하고 실행 가능한 경우 격리 검사 |
| Python/PyPI | wheel, source distribution | source distribution은 설치·빌드 코드 실행 가능 Artifact로 취급 |
| GitHub Releases | 단일 executable binary, `.zip`, `.tar.gz` 등 일반 binary/archive | package manager 없이 반입하는 흐름을 지원 |

동적 실행 검사는 현재 MVP 검사 backend에서 실행 가능한 Artifact에 한정한다. 지원 형식에 해당하더라도 현재 MVP의 OS 또는 backend에서 필수 검사를 수행할 수 없으면 검사 한계를 명시하고 자동 `ALLOW`하지 않는다.

생태계별 파일 형식과 metadata 해석은 adapter가 담당한다. 공통 Core는 특정 확장자나 package 형식에 종속되지 않으며, Artifact identity·acquisition·verification·quarantine·scanning·policy·evidence·staging·promotion 계약은 형식 중립적으로 유지한다.

### MVP-004 직접 지원 제외 및 향후 확장

다음은 MVP 직접 지원 범위에서 제외하고 향후 확장 대상으로 둔다.

- OS 전용 GUI installer
- Docker/OCI image
- IDE extension
- MCP server
- Git repository
- CI component
- `curl | sh` 형태 설치 흐름

### MVP-004 설치·압축 해제·형식 판별 정책

- Python source distribution은 단순 archive가 아니라 설치·빌드 중 코드 실행이 발생할 수 있는 실행 가능 설치 Artifact로 취급하고 격리된 설치·동적 검사 정책을 적용한다.
- Archive 압축 해제 자체를 검사 과정으로 포함한다.
- 압축 해제 시 path traversal, 절대경로, 비정상 symlink·hardlink, 특수 파일, 과도한 파일 수·용량을 통제된 격리 환경에서 검사한다.
- 파일 확장자만으로 Artifact 유형을 판별하지 않고 가능한 경우 파일 구조·metadata·content type으로 실제 형식과의 일치 여부를 확인한다.
- 확장자와 실제 형식이 불일치하면 위험 신호로 기록한다.
- Archive 내부의 중첩 archive도 검사 대상으로 취급한다.
- 중첩 archive에는 재귀 깊이·압축 해제 크기·파일 수 제한을 적용해 검사 우회와 압축폭탄에 의한 자원 고갈을 방지한다.

### MVP-004 구현 영향

- npm, PyPI/pip, GitHub Releases adapter에 형식 및 metadata 판별 책임을 둔다.
- Core contract에는 확장자 기반 안전 신뢰나 생태계별 file type 분기를 포함하지 않는다.
- Artifact manifest에 선언된 형식, 관찰된 실제 형식, content type, 형식 일치 여부를 기록한다.
- 단일 binary, wheel, source distribution, `.zip`, `.tar.gz` fixture를 정상·변조·악성·오형식 사례로 구성한다.
- 설치·빌드·압축 해제 전용 단계와 동적 실행 가능 여부를 capability로 기록한다.
- archive extraction은 격리된 임시 영역에서 수행하며 path·link·special file 정책 위반을 `BLOCK` 또는 위험 수준별 판정으로 연결한다.
- nested archive depth, expanded size, file count와 전체 resource limit을 함께 적용하고 초과 시 Evidence를 기록한다.
- MVP 제외 유형은 “미지원”으로 명확히 반환하며 지원 Artifact로 오인해 자동 `ALLOW`하지 않는다.

### MVP-004 누락 점검

- [x] npm package artifact 지원
- [x] Python wheel 지원
- [x] Python source distribution 지원
- [x] GitHub Releases 일반 binary/archive 지원
- [x] 단일 executable binary와 `.zip`, `.tar.gz` 우선 지원
- [x] 동적 실행을 현재 MVP backend에서 실행 가능한 Artifact로 한정
- [x] 생태계별 형식·metadata 해석을 adapter에 배치
- [x] 공통 Core의 확장자·package 형식 비종속성
- [x] MVP 직접 제외 유형 7가지와 향후 확장 범위 기록
- [x] 필수 검사가 불가능한 지원 유형의 자동 `ALLOW` 금지와 한계 명시
- [x] Python source distribution의 실행 가능 설치 Artifact 취급
- [x] archive 압축 해제 자체를 검사 과정에 포함
- [x] path traversal·절대경로·symlink·hardlink·특수 파일 통제
- [x] 과도한 파일 수·용량 제한
- [x] 확장자 외 구조·metadata·content type으로 실제 형식 확인
- [x] 형식 불일치를 위험 신호로 기록
- [x] nested archive 검사
- [x] 재귀 깊이·압축 해제 크기·파일 수 제한으로 압축폭탄·자원 고갈 방지

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-004 | 2026-08-08 | npm package, Python wheel/source distribution, GitHub Releases binary/archive를 MVP 지원 대상으로 하고, 형식·압축 해제·설치 실행 위험을 격리 검사 | 핵심 개발 Artifact를 실제 검증하면서 생태계별 형식은 adapter로 격리하고 archive·설치 과정의 공급망 위험을 통제하기 위해 | 형식 중립 Core, adapter 판별, archive 안전성·중첩 제한, 실행 가능 설치 Artifact 정책과 명확한 미지원 판정이 MVP 조건이 됨 |

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

## 사용자 원문: MVP-008

```text
8번: SBOM 및 Evidence/결과물 범위

MVP에서는 모든 검사 결과를 단순 PASS/FAIL로만 남기지 않고, Artifact identity, 수행된 검사, Finding, Evidence 및 최종 Policy Decision을 연결하여 재현 가능한 검사 기록을 생성한다.

결과에는 Artifact의 이름·버전·source·digest·유형과 dependency 정보, 사용된 검사 도구와 버전, 검사 시점, 수행·미수행 검사와 그 이유, 주요 정적·동적 관찰 결과를 포함한다.

Dependency와 구성요소는 가능한 범위에서 SBOM으로 생성하며, 표준 포맷 사용을 원칙으로 하되 구체적인 SBOM 표준과 생성 도구는 이후 단계에서 확정한다.

ALLOW되어 Staging Area로 승격되는 경우에는 실제 승격 대상 Artifact와 dependency의 digest, 검사 결과 및 Evidence를 연결한 검증 완료 세트 Manifest를 생성한다.

결과는 사람이 터미널에서 빠르게 확인할 수 있는 요약과, CI·MCP·자동화 도구가 처리할 수 있는 machine-readable 구조화 결과를 함께 제공한다.

Evidence와 결과물의 저장 방식·보존 기간·세부 schema는 MVP Architecture 및 Domain Model 단계에서 확정한다.
```

## 구조화된 결정

### MVP-008 재현 가능한 검사 기록

모든 검사 결과는 단순 `PASS`/`FAIL`만으로 남기지 않는다. 다음 요소를 연결하여 Artifact를 동일하게 다시 검사하고 판정을 재현할 수 있는 검사 기록을 생성한다.

| 연결 요소 | 포함 내용 |
| --- | --- |
| Artifact identity | 이름·버전·source·digest·유형 |
| 수행된 검사 | 수행·미수행 검사, 미수행 이유, 사용된 검사 도구·버전 |
| Finding | 정적·동적 검사에서 확인된 문제·위험 신호 |
| Evidence | 관찰 결과와 판정 근거 |
| Policy Decision | 최종 `ALLOW`, `MANUAL_REVIEW`, `BLOCK` 등 정책 결과 |
| 시점 | 검사 시점과 관련 실행 환경 정보 |
| dependency | dependency 정보와 가능한 구성요소 관계 |

결과에는 주요 정적·동적 관찰 결과와 수행되지 않은 검사의 이유를 함께 포함한다.

### MVP-008 SBOM과 Staging Manifest

- Dependency와 구성요소는 가능한 범위에서 SBOM으로 생성한다.
- SBOM은 표준 포맷 사용을 원칙으로 한다.
- 구체적인 SBOM 표준과 생성 도구는 이후 Architecture·Domain Model 및 도구 선택 단계에서 확정한다.
- Artifact가 `ALLOW`되어 Staging Area로 승격되는 경우 실제 승격 대상 Artifact와 dependency의 digest를 기록한다.
- 승격 세트의 검사 결과와 Evidence를 연결한 검증 완료 세트 Manifest를 생성한다.
- Manifest는 검사된 대상과 실제 반입 대상의 동일성을 확인할 수 있어야 한다.

### MVP-008 결과 표현과 후속 확정 항목

- 사람이 터미널에서 빠르게 확인할 수 있는 요약 결과를 제공한다.
- CI·MCP·자동화 도구가 처리할 수 있는 machine-readable 구조화 결과를 함께 제공한다.
- Evidence와 결과물의 저장 방식, 보존 기간, 세부 schema는 MVP Architecture 및 Domain Model 단계에서 확정한다.

### MVP-008 구현 영향

- 결과 생성기는 Artifact identity와 실행별 검사 기록을 루트로 하여 Finding·Evidence·Policy Decision을 참조 연결한다.
- 결과에 performed checks와 skipped checks를 구분하고 skipped 사유를 필수 필드로 둔다.
- 검사 도구·버전, 검사 시점, 실행 OS·backend, Artifact·dependency digest를 결과에 기록한다.
- 정적·동적 관찰 결과는 원본 Evidence와 요약 Finding을 연결한다.
- SBOM provider를 Core 결과 계약에 연결하되 특정 표준·도구 구현은 provider 경계에 둔다.
- Staging 승격 시 검증 완료 세트 Manifest에 Artifact·dependency identity/digest, 검사 결과와 Evidence 참조를 포함한다.
- 터미널 요약과 machine-readable 출력이 동일한 판정·identity·digest를 나타내는지 contract test로 검증한다.
- 저장소·보존·schema 확정 전까지 결과 포맷을 임의로 고정하지 않고 Architecture·Domain Model 결정으로 이관한다.

### MVP-008 누락 점검

- [x] 단순 PASS/FAIL을 넘어 재현 가능한 검사 기록 생성
- [x] Artifact identity 연결
- [x] 수행·미수행 검사와 미수행 이유 포함
- [x] Finding·Evidence·최종 Policy Decision 연결
- [x] Artifact 이름·버전·source·digest·유형 포함
- [x] dependency 정보 포함
- [x] 검사 도구·버전과 검사 시점 포함
- [x] 주요 정적·동적 관찰 결과 포함
- [x] 가능한 범위의 SBOM 생성
- [x] SBOM 표준 포맷 원칙
- [x] SBOM 표준·생성 도구 후속 확정
- [x] ALLOW된 Staging 승격 대상의 Artifact·dependency digest 기록
- [x] 검사 결과·Evidence를 연결한 검증 완료 세트 Manifest 생성
- [x] 터미널용 사람 중심 요약 제공
- [x] CI·MCP·자동화용 machine-readable 결과 제공
- [x] 저장 방식·보존 기간·세부 schema를 Architecture·Domain Model 단계에서 확정

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-008 | 2026-08-08 | Artifact·검사·Finding·Evidence·Policy Decision을 연결한 재현 가능한 결과와 SBOM·Staging Manifest를 생성하고 사람용·기계용 출력을 함께 제공 | 판정만 전달하지 않고 검사 근거와 실제 반입 대상의 동일성을 추적·재현하기 위해 | 결과 계약, SBOM provider, 승격 Manifest, 터미널·machine-readable 출력 및 후속 저장·보존·schema 결정이 MVP 조건이 됨 |

## 사용자 원문: MVP-009

```text
9번: MVP 사용자 흐름 및 완료 범위



사용자가 Artifact 지정

        ↓

Artifact/source 식별

        ↓

격리 환경으로 다운로드

        ↓

출처·digest·signature·provenance 검증

        ↓

정적 검사

        ↓

격리 설치 및 동적 검사

        ↓

Policy 판정

        ↓

 ┌───────────────┬───────────────┐

 ALLOW        MANUAL_REVIEW       BLOCK

  ↓               ↓               ↓

Staging       결과·근거 제공      반입 금지

  ↓

digest 재검증

  ↓

사용자가 지정한 환경에 설치

  ↓

Manifest / SBOM / Evidence / 결과 제공


`ALLOW`된 Artifact는 원칙적으로 검증 완료 Staging Area를 통해 사용자 환경으로 반입한다. 다만 package manager가 필요 없는 독립 binary/archive는 Threat Model D-011에 따라 검사 대상과 반입 대상의 동일성을 digest로 재검증한 후 사용자가 지정한 위치로 직접 반입할 수 있다. `WARN`은 독립적인 최종 Policy Decision이 아니라 검사 과정에서 발견된 경고 수준의 Finding으로 기록한다. 자동 판정이 불가능하거나 사람의 판단이 필요한 경우 최종 Policy Decision은 `MANUAL_REVIEW`로 처리한다. `BLOCK`된 Artifact는 사용자 환경으로 반입하지 않는다.

MVP는 npm에서 전체 end-to-end 흐름이 완성되고, PyPI/pip와 GitHub Releases가 동일한 공통 Core와 adapter 계약을 통해 정의된 MVP 흐름을 수행할 수 있는 상태를 완료 기준으로 한다. 특정 생태계를 추가하기 위해 공통 Core에 해당 생태계 전용 로직을 넣어야 하는 구조는 MVP 완료로 보지 않는다.
```

## 구조화된 결정

### MVP-009 사용자 흐름 확정

```text
사용자 Artifact 지정
  → Artifact/source 식별
  → 격리 환경 다운로드
  → 출처·digest·signature·provenance 검증
  → 정적 검사
  → 격리 설치·동적 검사
  → Policy 판정
      ├─ ALLOW
      │    ├─ 일반 경로 → Staging → digest 재검증 → 사용자 지정 환경 설치 → Manifest/SBOM/Evidence/결과 제공
      │    └─ 독립 binary/archive → digest 재검증 → 사용자 지정 위치 직접 반입 가능
      ├─ MANUAL_REVIEW → 결과·근거 제공, 추가 판단 대기
      └─ BLOCK → 사용자 환경 반입 금지
```

각 단계의 수행 여부와 결과는 공통 검사 기록에 연결한다. `ALLOW`된 Artifact는 원칙적으로 검증 완료 Staging Area를 통해 사용자 환경으로 반입한다. 다만 package manager가 필요 없는 독립 binary/archive는 Threat Model D-011에 따라 검사 대상과 반입 대상의 동일성을 digest로 재검증한 후 사용자가 지정한 위치로 직접 반입할 수 있다.

### MVP-009 Policy 분기 및 반입 규칙

| Policy 결과 | 다음 동작 |
| --- | --- |
| `ALLOW` | 일반 경로: Staging Area 승격 → 반입 직전 digest 재검증 → 사용자가 지정한 환경에 설치 → Manifest·SBOM·Evidence·결과 제공. 독립 binary/archive 예외: Threat Model D-011에 따라 검사 대상과 반입 대상의 동일성을 digest로 재검증한 뒤 사용자 지정 위치로 직접 반입 가능 |
| `MANUAL_REVIEW` | 결과와 근거를 제공하고 사람의 추가 판단이 필요한 상태로 유지하며 자동 반입하지 않음 |
| `BLOCK` | 사용자 환경으로 반입·설치 금지 |

`WARN`은 독립적인 최종 Policy Decision이 아니라 Finding의 경고 수준을 나타내며, 사람의 추가 판단이 필요한 경우 최종 Policy Decision은 `MANUAL_REVIEW`로 처리한다.

### MVP-009 완료 기준

- npm에서 사용자 지정부터 설치·결과 제공까지 전체 end-to-end 흐름이 완성되어야 한다.
- PyPI/pip와 GitHub Releases는 동일한 공통 Core와 adapter 계약으로 정의된 MVP 흐름을 수행할 수 있어야 한다.
- 세 생태계의 기능 수준이 완전히 동일할 필요는 없지만, 공통 단계·결과 계약·Policy 분기를 깨뜨리지 않아야 한다.
- 특정 생태계를 추가하기 위해 공통 Core에 해당 생태계 전용 로직을 넣어야 하는 구조는 MVP 완료로 보지 않는다.

### MVP-009 구현 영향

- 전체 workflow orchestration은 각 단계의 입력·출력·상태·Evidence 참조를 연결한다.
- Policy 결과에 따라 일반 Staging 승격, 독립 binary/archive 직접 반입 예외, 추가 검토 대기, 반입 차단을 명시적으로 분기한다.
- `ALLOW` 일반 경로에서는 Staging 승격 후 digest 재검증을 필수 단계로 둔다.
- package manager가 필요 없는 독립 binary/archive의 직접 반입은 Threat Model D-011에 따라 검사 대상과 반입 대상의 digest 재검증이 완료된 경우에만 허용한다.
- 사용자 지정 환경 설치는 검증 완료 세트와 연결된 Manifest를 사용한다.
- `WARN`은 Finding severity로만 기록하고, 사람의 판단이 필요한 최종 결과는 `MANUAL_REVIEW`로 자동 설치 경로와 분리하며 사용자가 확인할 수 있는 근거를 제공한다.
- npm reference implementation과 PyPI/pip·GitHub Releases adapter에 동일한 workflow·contract test를 적용한다.
- Core에 생태계 전용 분기가 추가되는지 architecture/dependency rule로 검사한다.

### MVP-009 누락 점검

- [x] 사용자 Artifact 지정
- [x] Artifact/source 식별
- [x] 격리 환경 다운로드
- [x] 출처·digest·signature·provenance 검증
- [x] 정적 검사
- [x] 격리 설치 및 동적 검사
- [x] Policy 판정과 ALLOW/MANUAL_REVIEW/BLOCK 분기
- [x] WARN을 최종 Policy Decision이 아닌 Finding 경고 수준으로 기록
- [x] ALLOW Artifact의 원칙적 Staging Area 경유
- [x] 독립 binary/archive의 D-011 기반 digest 재검증 후 직접 반입 예외
- [x] Staging 후 digest 재검증
- [x] 사용자 지정 환경 설치
- [x] Manifest·SBOM·Evidence·결과 제공
- [x] MANUAL_REVIEW의 결과·근거 제공과 사람의 추가 판단 상태
- [x] BLOCK Artifact의 사용자 환경 반입 금지
- [x] npm 전체 end-to-end 완료 기준
- [x] PyPI/pip·GitHub Releases의 동일 Core·adapter 흐름 수행 기준
- [x] Core 생태계 전용 로직 필요 시 MVP 미완료 판정

## 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| MVP-009 | 2026-08-08 | 지정·식별·격리 다운로드·검증·정적·동적 검사·Policy·Staging/설치·결과 제공의 전체 흐름을 정의하고 npm E2E와 두 adapter의 공통 계약 수행을 완료 기준으로 삼음 | 사용자가 실제로 재현 가능한 반입 흐름을 사용하고 생태계 중립 Core 확장성을 검증하기 위해 | `WARN` Finding과 최종 Policy Decision 분리, 일반 Staging 경로·독립 binary/archive D-011 직접 반입 예외, digest 재검증, 공통 contract test와 Core 전용 로직 금지가 MVP 조건이 됨 |
