# Heliopause Artifact Airlock

외부 artifact와 패키지를 목표 개발 환경에 설치하기 전, 격리된 검역 환경에서 provenance·무결성·악성행위 위험을 확인하는 도구의 실용성을 탐색한다.

- 프로젝트명: Heliopause Artifact Airlock
- CLI 후보: `helox`
- 상태: Domain-001~007 확정, Interface Contracts 설계 예정 (Architecture·User Journey + CLI IA 완료)

## 결정 문서

- 전체 결정 상태와 상세 문서 색인: [`PROJECT-DECISIONS.md`](./PROJECT-DECISIONS.md)
- 위협 모델 사용자 원문·상세 정책: [`docs/threat-model.md`](./docs/threat-model.md)
- MVP Scope 사용자 원문·상세 결정: [`docs/mvp-scope.md`](./docs/mvp-scope.md)
- Architecture 사용자 원문·상세 결정: [`docs/architecture.md`](./docs/architecture.md)
- User Journey + CLI IA 상세 결정: [`docs/user-journey-cli-ia.md`](./docs/user-journey-cli-ia.md)
- Domain Model / Interface Contracts 상세 결정: [`docs/domain-model.md`](./docs/domain-model.md)

다음 결정 단계: Contract-001~003 설계

상세 내용은 주제별 문서에 한 번만 기록하고, `PROJECT-DECISIONS.md`에는 상태·핵심 요약·링크만 유지한다.

## 초기 방향

```text
패키지·artifact 요청
  → 다운로드와 provenance 수집
  → checksum·digest·서명 검증
  → 격리된 검사 환경에서 정책 기반 검사
  → 결과·SBOM·manifest 생성
  → 통과한 고정 artifact만 목표 환경으로 반입
```

Docker 이미지는 재현 가능한 검역 환경과 배포 단위 후보로 검토한다. 단, 이미지 자체의 digest 고정, 서명 검증, 최소 권한·네트워크 격리 및 host credential 비노출이 함께 충족되어야 한다.

초기 구현과 도구 선택은 아직 시작하지 않는다. 먼저 위협 모델, 지원 package manager 범위, lifecycle script 처리, 검사 실패 정책, artifact 반입 방식, 이미지 신뢰·갱신 정책을 정한다.
