# Heliopause Artifact Airlock 프로젝트 결정 문서

이 문서는 `Heliopause Artifact Airlock`(`helox`)를 만들기 전에 언어, 기술, 구조, 범위와 운영 방식을 하나씩 결정해 기록하는 단일 기준 문서다. 결정되지 않은 항목은 추측으로 채우지 않고 `미결정`으로 남긴다.

## 문서 사용 규칙

- 사용자가 선택하거나 실험으로 검증한 내용만 결정사항으로 기록한다.
- 변경이 생기면 기존 결정을 지우지 않고 결정일, 이유, 영향과 대안을 남긴다.
- 이 문서가 충분히 채워진 뒤 실제 구현 문서(PRD, architecture, ADR, WBS, test plan)로 분리한다.
- 구현 시작 전에는 미결정 항목과 차단 항목을 모두 확인한다.

## 1. 프로젝트 기본 정보

| 항목 | 상태 | 결정 | 비고 |
| --- | --- | --- | --- |
| 프로젝트명 | 결정 | Heliopause Artifact Airlock |  |
| CLI 명령어 | 결정 후보 | `helox` | 사용 가능 여부와 패키지 충돌은 확인 필요 |
| 목적 | 초안 | 외부 artifact·패키지를 검역하고 검증된 고정 artifact만 목표 환경에 반입 |  |
| 1차 사용자 | 미결정 |  | 개인 개발자, 팀, CI 운영자 중 선택 |
| 지원 운영체제 | 미결정 |  | macOS, Linux, Windows |
| 배포 형태 | 미결정 | Docker image, standalone CLI, package | 신뢰·재현성 검토 필요 |
| 라이선스 | 미결정 |  |  |

## 2. 위협 모델과 보안 경계

### D-001 보호 대상 확정

HAA는 다음 다섯 가지 공급망·실행 위험으로부터 사용자의 개발 환경과 목표 환경을 보호하는 것을 1차 보호 목표로 삼는다.

1. 원래부터 악성인 패키지
2. 정상 패키지가 탈취·변조되어 악성화된 패키지
3. 설치 순간 실행되는 악성 코드(lifecycle script, installer, hook 등)
4. 다운로드 과정에서 변조된 artifact
5. 검사 과정에서 격리 경계를 탈출하거나 Host를 공격하는 행위

이 결정은 특정 package manager나 검사 도구에 종속되지 않는다. 각 위험을 탐지·차단·격리·기록하는 정책과 성공 기준은 후속 결정에서 정의한다.

### D-002 보호 자산 범위 확정

HAA의 보호 범위는 artifact 자체에 한정하지 않고, 검역과 설치 과정에서 접근될 수 있는 다음 Host 자산과 실행 환경까지 포함한다.

1. Host OS와 파일시스템
2. 프로젝트 소스코드와 사용자 파일
3. SSH Key, API Key, GitHub Token 등 자격증명
4. `.env`와 환경변수
5. 브라우저·클라우드 인증정보
6. 내부 네트워크와 다른 장비
7. 실행 중인 다른 프로세스와 서비스

검역 환경은 위 자산에 대한 기본 접근을 차단하고, 꼭 필요한 접근만 명시적인 정책·권한·감사 기록을 통해 허용하는 것을 전제로 한다.

### D-003 기본 불신 원칙 확정

HAA의 기본 신뢰 모델은 **Untrusted by default**다. 공식 Registry나 공식 배포 경로를 사용했다는 사실만으로 제작자, 배포 권한, 서버, artifact 또는 외부 네트워크를 신뢰하지 않는다.

| 불신 대상 | 불신 이유 |
| --- | --- |
| 패키지·artifact 제작자 | 원래부터 악의적일 수 있음 |
| 정상 제작자의 계정·배포 권한 | 계정이나 배포 권한이 탈취됐을 수 있음 |
| Registry·배포 서버 | npm, PyPI, GitHub Releases 등 공식 경로도 침해·오배포 가능성이 있음 |
| Artifact 자체 | 공식 이름·공식 경로라도 검사 전에는 안전성을 확정할 수 없음 |
| Artifact가 접근하려는 외부 네트워크 | 검사 중 통신 시도와 상대 서버를 모두 기본적으로 신뢰하지 않음 |

이 원칙은 npm·PyPI·GitHub Releases 자체를 악성 운영자로 단정한다는 뜻이 아니다. 출처의 공식성은 provenance 확인의 한 요소일 뿐이며, 검증·정책 판정·격리 실행을 통과하기 전까지는 권한과 실행을 부여하지 않는다는 뜻이다.

### D-004 Registry 제한 신뢰 범위 확정

공식 Registry는 Artifact의 출처·버전·메타데이터를 제공하는 신뢰 가능한 인프라로 **제한적으로만** 취급한다. Registry에 존재하거나 공식 배포 경로를 통해 제공되었다는 사실만으로 Artifact의 안전성을 신뢰하지 않는다.

Publisher 계정 탈취, 악성 버전 배포, Registry 침해 가능성은 Registry의 provenance 기능과 별도로 다뤄야 하는 위협이다. 따라서 Registry가 제공하는 정보는 검증 입력으로 사용하지만, 안전성 판정이나 실행 허용의 단독 근거로 사용하지 않는다.

### D-005 Publisher compromise 처리 원칙 확정

정상 Publisher와 기존 신뢰 이력이 있더라도 새 Artifact나 새 버전을 자동으로 신뢰하지 않는다. Publisher 계정이나 배포 권한이 탈취될 수 있다고 가정하며, 모든 신규 버전은 독립적으로 검증한다.

서명과 provenance가 정상이어도 그것만으로 안전성을 확정하지 않는다. 검증에는 최소한 다음 비교가 포함되어야 한다.

- 이전 버전 대비 비정상적인 파일·의존성·스크립트·권한 변경
- 설치·빌드·실행 과정에서의 비정상적인 실행 행위
- 기존 Publisher·package의 정상적인 변경 패턴과 다른 release 특성

### D-006 Lifecycle script 및 실행 코드 처리 확정

Artifact는 실제 Host 환경에 반입하기 전에 다운로드·설치·실행의 단계별 검사를 수행한다.

| 단계 | 처리 원칙 |
| --- | --- |
| 다운로드 | Artifact를 격리된 환경으로 가져오고 실행 없이 checksum·digest·provenance·정적 검증을 수행 |
| 설치 | lifecycle script를 포함한 설치 동작을 격리된 환경에서 실행하고 파일·프로세스·네트워크·환경변수 변화를 관찰 |
| 설치 후 실행 | 필요할 때만 프로그램 자체를 격리된 환경에서 제한적으로 실행하여 동적 행위를 검사 |
| 반입 | 단계별 정책을 통과한 고정 Artifact만 목표 Host 환경으로 반입 |

모든 실행 단계에서는 Host 파일시스템, 자격증명, `.env`와 환경변수, 브라우저·클라우드 인증정보, 내부 네트워크, 다른 장비, 실행 중인 Host 프로세스와 서비스에 직접 접근할 수 없도록 제한한다. 격리 환경의 권한·네트워크·파일 접근 정책과 관찰 결과는 receipt에 기록한다.

### D-007 격리 환경 탈출 처리 범위 확정

격리 환경 탈출은 실제 위협으로 포함한다. 다만 컨테이너 런타임, 운영체제 커널, VM 등 기반 격리 기술 자체의 취약점을 완전히 해결하거나 안전성을 보증하는 것을 HAA의 1차 목표로 삼지 않는다.

대신 탈출 성공을 전제로 다음 다중 방어를 적용해 피해 가능성을 최소화한다.

- non-root 실행
- 최소 권한과 capability 제한
- Host filesystem mount 최소화 또는 금지
- 자격증명·secret 비노출
- 네트워크 기본 차단과 명시적 제한
- CPU·메모리·디스크·프로세스 등 resource limit
- 실행 중인 Host 프로세스·서비스와의 분리

탈출 자체를 적극적으로 방어하는 독자적인 격리 기술 구축은 범위를 크게 확장하므로 1차 구현에서 제외하고, 향후 별도 확장 과제로 기록한다.

### D-008 Host 자산 접근 통제와 미끼 자산 확정

격리 환경에서는 실제 Host filesystem, 자격증명, 환경변수, 내부 네트워크, Host 서비스와 프로세스를 기본적으로 노출하지 않는다. 모든 접근은 **default deny**를 원칙으로 한다.

- 실제 Secret·Credential은 격리 환경에 주입하지 않는다.
- 필요한 경우 **격리 환경 내부에만** 실제 개발 환경과 유사한 디렉터리 구조·권한·더미 filesystem을 만든다. 그 안에 권한 없는 더미 Credential·Secret·파일(honeytoken/canary)을 포함한 모의 Host 환경을 구성한다. 이 모의 환경은 외부나 실제 Host filesystem에 생성하지 않는다.
- Artifact가 민감정보를 탐색·읽기·복사·전송하려는 행동은 canary 반응과 격리 telemetry로 탐지한다.
- 프로젝트 파일이 필요해도 실제 Host filesystem을 직접 노출하지 않고, 격리 환경 내부에 별도로 복사·정제한 검사 전용 filesystem과 데이터만 제공한다.
- 네트워크는 기본적으로 실제 외부·내부 네트워크와 격리한다.
- 동적 분석에는 격리 환경 내부에서 실행하는 가짜 DNS/HTTP 서버 또는 통제된 격리 네트워크를 사용해 접속 대상과 통신 시도를 관찰·기록한다.

모의 filesystem, 디렉터리, 더미 자산, fake DNS/HTTP 서버와 관찰 telemetry는 모두 격리 환경의 경계 안에서 생성·실행·보관한다. Honeytoken·canary는 실제 접근 권한이나 유효한 자격증명을 포함하지 않아야 하며, 관찰 데이터도 격리 환경 밖으로 불필요하게 전송하지 않는다.

### D-009 방어 범위의 한계 확정

HAA는 외부 Software Artifact가 신뢰된 개발 환경으로 반입되기 전에 발생할 수 있는 공급망 위험의 **탐지·격리·피해 제한**을 목표로 한다. 모든 악성행위를 완전히 탐지하거나 Artifact의 절대적인 안전성을 보증하지 않는다.

다음 항목은 기본 범위에 포함하지 않는다.

- 이미 감염된 Host의 치료
- OS·커널·컨테이너·VM 등 기반 기술 자체의 취약점 해결
- 하드웨어·펌웨어 공격
- 범용 백신·EDR과 같은 지속적인 Host 감시

또한 정적·동적 검사에서 드러나지 않는 조건부·지연형 악성행위나 알려지지 않은 공격을 항상 탐지할 수 있다고 가정하지 않는다. 검사 결과는 안전 보증서가 아니라, 정의된 정책과 관찰 범위에 대한 판정과 근거로 제공한다.

### D-010 검사 실패 및 불확실성 처리 확정

필수 검사가 실패하거나 충분한 검증 근거를 확보하지 못한 경우 Artifact를 자동으로 안전하다고 판단하지 않는다. 보안에 중요한 검증 과정에서는 **fail-closed**를 기본 원칙으로 한다.

| 상태 | 처리 |
| --- | --- |
| 필수 검사 실패 | 기본 `BLOCK` |
| 결과 불충분 또는 판단 불가능 | 위험 수준에 따라 `MANUAL_REVIEW` 또는 `BLOCK` |
| 검사 완료 및 정책 통과 | 다음 단계 또는 `PROMOTE` 가능 |
| 위험이 발견되지 않음 | 수행된 검사 범위 안에서만 “미탐지”로 기록; 안전 보증으로 해석하지 않음 |

검사 실패와 “위험이 발견되지 않음”을 동일하게 취급하지 않는다. 결과에는 수행된 검사, 수행되지 않은 검사, 미수행 이유, 각 검사 상태, 정책 판정과 수동 검토 여부를 명확하게 기록한다.

| 항목 | 상태 | 결정 |
| --- | --- | --- |
| 공격 대상 | 결정 | 원래 악성인 패키지, 탈취되어 악성화된 정상 패키지, 설치 즉시 실행 코드, 다운로드 변조, 검사 중 Host 공격 |
| 보호 자산 | 결정 | Host OS·파일시스템, 소스코드·사용자 파일, 자격증명, `.env`·환경변수, 브라우저·클라우드 인증정보, 내부 네트워크·다른 장비, 실행 중인 프로세스·서비스 |
| 기본 신뢰 모델 | 결정 | Untrusted by default |
| Registry 신뢰 범위 | 결정 | 출처·버전·메타데이터 제공 인프라로 제한 신뢰; Artifact 안전성은 별도 검증 |
| Publisher 신뢰 범위 | 결정 | 기존 신뢰 이력과 정상 서명도 신규 Artifact의 자동 신뢰로 승계하지 않음 |
| 모의 Host·filesystem 위치 | 결정 | 디렉터리·권한·더미 자산·fake DNS/HTTP를 포함한 모든 검사 환경은 격리 환경 내부에만 생성·실행·보관 |
| 설치 전 실행 허용 범위 | 미결정 |  |
| 네트워크 정책 | 미결정 | 기본 차단, allowlist, 제한된 outbound 중 선택 |
| host credential 접근 | 초안 | 차단 |
| 격리 경계 | 미결정 | container, VM, sandbox 중 선택 |
| 목표 환경 반입 조건 | 미결정 | checksum, digest, signature, policy, scan 결과의 조합 |
| 실패 시 동작 | 미결정 | 차단, 격리 보관, 사용자 승인, 경고 후 진행 |
| 로그·민감정보 정책 | 미결정 | secret redaction, 보존 기간, 로컬 저장 위치 |

## 3. 언어·프레임워크·실행 기반

| 항목 | 후보 | 결정 | 이유·검증 메모 |
| --- | --- | --- | --- |
| 주 구현 언어 | C++, C#, Go, Rust, Java | 미결정 | 보안, 배포, 생태계, 개발 생산성 비교 |
| CLI framework | 언어별 후보 | 미결정 | argument parsing, config, terminal UX |
| 설정 형식 | YAML, TOML, JSON 등 | 미결정 | 스키마 검증과 주석 지원 |
| plugin/adapter 방식 | 미결정 |  | package manager와 검사 도구 확장 |
| 동시성 모델 | 미결정 |  | 병렬 다운로드·검사와 자원 제한 |
| 최소 실행 환경 | 미결정 |  | container image와 standalone binary |
| MCP 제공 여부 | 미결정 |  | CLI 안정화 후 AI client용 adapter로 검토 |

## 4. 기능 범위

### 4.1 1차 필수 기능 후보

- package·artifact 요청과 버전·source 고정
- 다운로드 cache와 checksum/digest 기록
- provenance·서명·배포자 확인
- 격리된 검사 환경 생성과 lifecycle script 통제
- 정책 기반 검사 pipeline 실행
- 검사 결과, SBOM, manifest, provenance receipt 생성
- 통과 artifact의 명시적 반입
- 실패·보류·수동 승인 상태 관리
- 한국어·영어 메시지와 구조화된 machine-readable 결과

### 4.2 후순위 기능 후보

- 다중 package manager adapter
- 조직 정책과 allowlist/denylist
- CI/CD integration
- Docker image signing과 remote registry
- MCP server
- 재검사와 artifact quarantine archive
- 팀 단위 결과 공유와 감사 로그

## 5. 모듈 구조 초안

초기 구조는 모놀리식 실행 파일 내부에서도 책임을 분리한 모듈식 경계를 우선한다.

```text
helox/
├── cli/                 # 명령, 옵션, 사용자 출력
├── config/              # 설정·정책·스키마
├── acquisition/         # 다운로드·cache·source pinning
├── provenance/          # checksum·digest·signature·attestation
├── quarantine/          # 격리 환경과 권한·네트워크 정책
├── adapters/             # npm, PyPI, Cargo 등 package manager
├── scanners/             # 정적·악성·취약점·SBOM 검사 adapter
├── policy/               # pass/fail/review/allow 규칙
├── manifest/             # artifact receipt·결과·감사 기록
├── promotion/            # 검증 artifact의 목표 환경 반입
├── localization/         # 한국어·영어 등 메시지
└── tests/                # unit·integration·contract·E2E
```

모듈 간 의존 방향, public contract, 금지된 import와 plugin ABI는 구현 언어 결정 후 별도 ADR로 확정한다.

## 6. 사용자 여정 초안

```text
검사 대상 지정
  → helox가 환경·package manager·정책 탐지
  → 다운로드 전 source/version/digest/provenance 확인
  → artifact를 quarantine으로 이동
  → 정적·서명·악성·취약점·SBOM 검사
  → deterministic policy 판정
  → pass면 promotion, fail이면 차단·격리
  → 결과 receipt와 재현 명령 제공
```

예상 사용자 경로와 오류 복구 경로는 첫 지원 package manager가 정해진 뒤 BDD scenario로 구체화한다.

## 7. 개발 운영과 품질 도구

| 영역 | 후보 | 결정 |
| --- | --- | --- |
| formatter | 언어별 표준 도구 | 미결정 |
| linter | 언어별 표준 도구 | 미결정 |
| type checker | 컴파일러·언어 도구 | 미결정 |
| schema validator | JSON Schema 또는 대안 | 미결정 |
| static analysis | 언어·보안별 도구 | 미결정 |
| unit/integration/contract test | 언어별 표준 도구 | 미결정 |
| property/fuzz test | 필요 범위 결정 | 미결정 |
| architecture test | import·dependency rule 검사 | 미결정 |
| coverage/mutation | 적용 범위와 gate | 미결정 |
| secret/dependency scan | 도구 조합 | 미결정 |
| CI | GitHub Actions 등 | 미결정 |

## 8. 마일스톤·WBS·queue 초안

| ID | 작업 | 선행 조건 | 상태 |
| --- | --- | --- | --- |
| M0 | 위협 모델·신뢰 경계·성공 기준 확정 | 없음 | 미결정 |
| M1 | 언어·runtime·배포 형태 결정 | M0 | 미결정 |
| M2 | 최소 vertical slice와 디렉터리 구조 구현 | M1 | 미결정 |
| M3 | 첫 package manager의 다운로드·digest·검역 구현 | M2 | 미결정 |
| M4 | 최소 scanner·policy·receipt 구현 | M3 | 미결정 |
| M5 | promotion과 실패·승인 흐름 검증 | M4 | 미결정 |
| M6 | Docker image reproducibility·signing 검증 | M4 | 미결정 |
| M7 | 다른 package manager와 CI adapter 확장 | M5 | 미결정 |
| M8 | MCP adapter와 AI review contract 검토 | M5 | 미결정 |

### Queue list

- [ ] 언어와 CLI framework 후보 비교 기준 정의
- [ ] 첫 지원 package manager 선정
- [ ] 첫 번째 정상·악성·변조 fixture 확보
- [ ] quarantine 신뢰 경계와 반입 계약 정의
- [ ] checksum/digest/signature/provenance 결과 schema 정의
- [ ] 최소 deterministic policy 정의
- [ ] 도구 설치·업데이트의 supply-chain 정책 정의
- [ ] Docker image build와 서명 모델 결정

## 9. AI review 경계

AI review는 deterministic 검사 이전의 기본 단계가 아니라, 검사 결과와 프로젝트 맥락을 제공받아 자동 규칙의 사각지대를 검토하는 선택 단계로 둔다.

| 항목 | 상태 | 결정 |
| --- | --- | --- |
| AI review 입력 | 미결정 | 결과 receipt, policy finding, project context |
| AI review 출력 | 미결정 | 누락·위험·우선순위·수정 제안 |
| 자동 수정 권한 | 미결정 | 기본 금지 또는 사용자 승인 |
| 비용·지연 상한 | 미결정 |  |
| 민감정보 전달 정책 | 미결정 |  |

## 10. 결정 로그

| ID | 날짜 | 결정 | 이유 | 영향 |
| --- | --- | --- | --- | --- |
| D-000 | 2026-08-07 | 결정 문서를 단일 누적 문서로 사용 | 구현 전 선택사항과 미결정 항목을 한 곳에서 관리 | 이후 결정은 이 문서와 ADR에 연결 |
| D-001 | 2026-08-07 | 원래 악성, 탈취 후 악성화, 설치 즉시 실행, 다운로드 변조, 검사 중 Host 공격의 5개 위험을 1차 보호 대상으로 확정 | 패키지 설치 공급망과 검역 실행 환경의 주요 공격 경계를 함께 다루기 위해 | threat control·policy·fixture·acceptance criteria가 5개 위험을 모두 커버해야 함 |
| D-002 | 2026-08-07 | Host OS·파일시스템부터 실행 중인 프로세스·서비스까지 7개 자산을 보호 범위로 확정 | artifact 검사 과정의 Host 침해와 자격증명·내부망 탈취를 함께 방지하기 위해 | 기본 deny, 최소 권한, 네트워크 격리와 감사 기록이 설계의 필수 조건이 됨 |
| D-003 | 2026-08-07 | 제작자, 배포 권한, Registry·배포 서버, artifact, artifact의 외부 네트워크를 모두 기본 불신 대상으로 확정 | 공식 경로만으로는 계정 탈취·서버 침해·악성 배포를 배제할 수 없기 때문 | provenance·서명·정책 검증과 격리 전 실행 금지가 필수 조건이 됨 |
| D-004 | 2026-08-07 | 공식 Registry를 출처·버전·메타데이터 제공 인프라로 제한 신뢰하고 Artifact 안전성은 별도 검증 | Registry 존재·공식 경로만으로 Publisher 탈취, 악성 버전, Registry 침해를 배제할 수 없기 때문 | Registry metadata는 검증 입력이지 실행 허용의 단독 근거가 아님 |
| D-005 | 2026-08-07 | Publisher의 기존 신뢰 이력을 신규 Artifact에 승계하지 않고 모든 신규 버전을 독립 검증 | Publisher 계정·배포 권한 탈취와 정상 서명을 이용한 악성 배포를 고려하기 위해 | version diff·dependency diff·실행 행위 분석이 필수 검증 단계가 됨 |
| D-006 | 2026-08-07 | 다운로드·설치·실행을 단계별로 격리 검사하고 보호 자산 직접 접근을 차단 | lifecycle script와 설치 후 실행 코드가 Artifact 반입 순간 동작할 수 있기 때문 | 정적 검증, 격리 동적 관찰, 제한 실행, receipt 기록이 필수 pipeline이 됨 |
| D-007 | 2026-08-07 | 격리 탈출을 실제 위협으로 포함하되 독자적인 격리 기술 구축은 1차 범위에서 제외하고 다중 방어로 피해를 최소화 | 런타임·커널·VM 취약점까지 완전히 보증하는 것은 별도 보안 제품 수준의 범위이기 때문 | non-root, 최소 권한, mount·secret·network 제한과 resource limit이 필수 조건이 됨 |
| D-008 | 2026-08-07 | 실제 Host 자산은 default deny로 차단하고 모의 Host·더미 honeytoken·통제 네트워크로 탐지·관찰 | 실제 secret·filesystem·내부망을 노출하지 않고도 탐색·복사·전송·외부 통신 행위를 검증하기 위해 | 검사 전용 데이터, canary telemetry, fake DNS/HTTP와 격리 네트워크가 필수 조건이 됨 |
| D-009 | 2026-08-07 | 반입 전 공급망 위험의 탐지·격리·피해 제한을 목표로 하고 완전 탐지·절대 안전성은 보증하지 않음 | Host 치료, 기반 기술 보안, 하드웨어·펌웨어, 지속 감시와 미지·지연형 공격은 별도 영역이기 때문 | 결과는 정의된 검사 범위와 정책에 대한 판정이며 안전 보증서가 아님 |
| D-010 | 2026-08-07 | 필수 검사 실패·근거 부족은 fail-closed로 처리하고 `MANUAL_REVIEW` 또는 `BLOCK`으로 판정 | 검사 실패를 위험 없음으로 오해하면 검증 공백이 안전 판정으로 변하기 때문 | 검사 수행·미수행·사유·상태·정책 판정을 결과에 기록해야 함 |

## 11. 누락 점검

다음 항목은 향후 결정 문서가 충분히 채워졌는지 검토할 때 확인한다.

- [ ] 라이선스와 상표·배포 정책
- [ ] 지원 버전과 호환성 정책
- [ ] 성능·저장공간·네트워크 예산
- [ ] artifact 보존·삭제·복구 정책
- [ ] 장애·중단·rollback 정책
- [ ] audit log와 개인정보 처리
- [ ] 위협 모형과 fixture 기반 보안 테스트
- [ ] 설치·업데이트·마이그레이션 경로
