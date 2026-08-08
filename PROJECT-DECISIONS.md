# Heliopause Artifact Airlock 프로젝트 결정 색인

이 문서는 `Heliopause Artifact Airlock`(`helox`)의 전체 결정 상태와 상세 문서 위치를 안내하는 중앙 색인이다. 상세 정책과 사용자 원문은 주제별 `docs/` 문서에 한 번만 보존한다.

## 읽기 순서

1. [`README.md`](./README.md)
2. 이 문서
3. 현재 작업에 관련된 상세 문서
4. 작업 queue 문서가 생성되면 `current_work_queue.md`

## 프로젝트 기본 정보

| 항목 | 상태 | 결정 | 비고 |
| --- | --- | --- | --- |
| 프로젝트명 | 결정 | Heliopause Artifact Airlock |  |
| CLI 명령어 | 결정 후보 | `helox` | 사용 가능 여부와 패키지 충돌은 확인 필요 |
| 목적 | 초안 | 외부 artifact·패키지를 검역하고 검증된 고정 artifact만 목표 환경에 반입 |  |
| 1차 사용자 | 미결정 |  | 개인 개발자, 팀, CI 운영자 중 선택 |
| 지원 운영체제 | 미결정 |  | macOS, Linux, Windows |
| 배포 형태 | 미결정 | Docker image, standalone CLI, package | 신뢰·재현성 검토 필요 |
| 라이선스 | 미결정 |  |  |

## 결정 주제 색인

### Threat Model

- 상태: 완료 — D-001~D-014 확정
- 기본 원칙: Untrusted by default
- Registry는 출처·버전·메타데이터 제공 인프라로 제한 신뢰
- Host 자산은 default deny이며 실제 Secret을 격리 환경에 노출하지 않음
- 다운로드 → 설치 → 필요 시 실행 순으로 단계별 검사
- 검사에 사용한 동일 Artifact 세트를 staging area로 승격
- Evidence 무결성·completeness와 fail-closed 적용
- 검사 이미지·scanner·verifier도 외부 Artifact로 검증
- resource limit·timeout과 검사 환경 폐기로 자원 고갈 대응
- 상세·사용자 원문: [`docs/threat-model.md`](./docs/threat-model.md)

### MVP Scope

- 상태: 미결정
- 상세 문서: 필요할 때 `docs/mvp-scope.md` 생성

### Architecture

- 상태: 초안 후보만 존재
- 방향: 책임이 분리된 모듈식 구조
- 상세 문서: 필요할 때 `docs/architecture.md` 생성

### Domain Model

- 상태: 미결정
- 상세 문서: 필요할 때 `docs/domain-model.md` 생성

### 언어·프레임워크·실행 기반

| 항목 | 후보 | 결정 | 이유·검증 메모 |
| --- | --- | --- | --- |
| 주 구현 언어 | C++, C#, Go, Rust, Java | 미결정 | 보안, 배포, 생태계, 개발 생산성 비교 |
| CLI framework | 언어별 후보 | 미결정 | argument parsing, config, terminal UX |
| 설정 형식 | YAML, TOML, JSON 등 | 미결정 | 스키마 검증과 주석 지원 |
| plugin/adapter 방식 | 미결정 |  | package manager와 검사 도구 확장 |
| 동시성 모델 | 미결정 |  | 병렬 다운로드·검사와 자원 제한 |
| 최소 실행 환경 | 미결정 |  | container image와 standalone binary |
| MCP 제공 여부 | 미결정 |  | CLI 안정화 후 AI client용 adapter로 검토 |

## 기능 범위 초안

### 1차 필수 기능 후보

- package·artifact 요청과 버전·source 고정
- 다운로드 cache와 checksum/digest 기록
- provenance·서명·배포자 확인
- 격리된 검사 환경 생성과 lifecycle script 통제
- 정책 기반 검사 pipeline 실행
- 검사 결과, SBOM, manifest, provenance receipt 생성
- 통과 artifact의 명시적 반입
- 실패·보류·수동 승인 상태 관리
- 한국어·영어 메시지와 구조화된 machine-readable 결과

### 후순위 기능 후보

- 다중 package manager adapter
- 조직 정책과 allowlist/denylist
- CI/CD integration
- Docker image signing과 remote registry
- MCP server
- 재검사와 artifact quarantine archive
- 팀 단위 결과 공유와 감사 로그

## 모듈 구조 초안

초기 구조는 모놀리식 실행 파일 내부에서도 책임을 분리한 모듈식 경계를 우선한다.

```text
helox/
├── cli/                 # 명령, 옵션, 사용자 출력
├── config/              # 설정·정책·스키마
├── acquisition/         # 다운로드·cache·source pinning
├── provenance/          # checksum·digest·signature·attestation
├── quarantine/          # 격리 환경과 권한·네트워크 정책
├── adapters/            # npm, PyPI, Cargo 등 package manager
├── scanners/            # 정적·악성·취약점·SBOM 검사 adapter
├── policy/              # pass/fail/review/allow 규칙
├── manifest/            # artifact receipt·결과·감사 기록
├── promotion/           # 검증 artifact의 목표 환경 반입
├── localization/        # 한국어·영어 등 메시지
└── tests/               # unit·integration·contract·E2E
```

모듈 간 의존 방향, public contract, 금지된 import와 plugin ABI는 구현 언어 결정 후 별도 ADR로 확정한다.

## 사용자 여정 초안

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

## 개발 운영과 품질 도구

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

## 마일스톤·WBS 초안

| ID | 작업 | 선행 조건 | 상태 |
| --- | --- | --- | --- |
| M0 | 위협 모델·신뢰 경계·성공 기준 확정 | 없음 | 완료 |
| M1 | MVP Scope 확정 | M0 | 미결정 |
| M2 | 언어·runtime·배포 형태 결정 | M1 | 미결정 |
| M3 | 최소 vertical slice와 디렉터리 구조 구현 | M2 | 미결정 |
| M4 | 첫 package manager의 다운로드·digest·검역 구현 | M3 | 미결정 |
| M5 | 최소 scanner·policy·receipt 구현 | M4 | 미결정 |
| M6 | promotion과 실패·승인 흐름 검증 | M5 | 미결정 |
| M7 | Docker image reproducibility·signing 검증 | M5 | 미결정 |
| M8 | 다른 package manager와 CI adapter 확장 | M6 | 미결정 |
| M9 | MCP adapter와 AI review contract 검토 | M6 | 미결정 |

## AI review 경계

AI review는 deterministic 검사 이전의 기본 단계가 아니라, 검사 결과와 프로젝트 맥락을 제공받아 자동 규칙의 사각지대를 검토하는 선택 단계로 둔다.

| 항목 | 상태 | 결정 |
| --- | --- | --- |
| AI review 입력 | 미결정 | 결과 receipt, policy finding, project context |
| AI review 출력 | 미결정 | 누락·위험·우선순위·수정 제안 |
| 자동 수정 권한 | 미결정 | 기본 금지 또는 사용자 승인 |
| 비용·지연 상한 | 미결정 |  |
| 민감정보 전달 정책 | 미결정 |  |

## 다음 결정 queue

- [ ] MVP Scope 확정
- [ ] 언어와 CLI framework 후보 비교 기준 정의
- [ ] 첫 지원 package manager 선정
- [ ] 첫 번째 정상·악성·변조 fixture 확보
- [ ] quarantine 신뢰 경계와 반입 계약 정의
- [ ] checksum/digest/signature/provenance 결과 schema 정의
- [ ] 최소 deterministic policy 정의
- [ ] 도구 설치·업데이트의 supply-chain 정책 정의
- [ ] Docker image build와 서명 모델 결정

## 전체 누락 점검

- [ ] 라이선스와 상표·배포 정책
- [ ] 지원 버전과 호환성 정책
- [ ] 성능·저장공간·네트워크 예산
- [ ] artifact 보존·삭제·복구 정책
- [ ] 장애·중단·rollback 정책
- [ ] audit log와 개인정보 처리
- [ ] 위협 모형과 fixture 기반 보안 테스트
- [ ] 설치·업데이트·마이그레이션 경로
