# MVP Scope — Ecosystems, Platforms, and Artifacts

## 사용자 원문: MVP-001

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
