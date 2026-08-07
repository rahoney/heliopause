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

| 항목 | 상태 | 결정 |
| --- | --- | --- |
| 공격 대상 | 결정 | 원래 악성인 패키지, 탈취되어 악성화된 정상 패키지, 설치 즉시 실행 코드, 다운로드 변조, 검사 중 Host 공격 |
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
