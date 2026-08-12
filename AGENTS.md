# 작업 규칙

## 작업 재개

1. `PROJECT-DECISIONS.md`에서 현재 단계와 남은 결정을 확인한다.
2. `docs/README.md`에서 현재 작업에 필요한 canonical 문서를 찾는다.
3. `docs/planning/02-current-work-queue.md`에서 active/next work item, 선행조건과 acceptance를 확인한다.
4. 한 번에 하나의 work item만 `IN_PROGRESS`로 두고 그 범위만 수행한다.

현재 재개 지점은 `M0-002 — Go·Tool·CI Identity lock`이다. Queue가 이후 갱신되면 Queue의 현재 상태를 우선한다.

## 구현과 문서

- 상세 결정은 관련 canonical leaf 문서에 한 번만 기록한다.
- 구현 전에 work item의 `Read` 문서를 확인한다.
- 기존 Architecture·Domain·Security invariant를 구현 편의로 변경하지 않는다.
- 미래 milestone의 placeholder package, config, interface 또는 CI job을 만들지 않는다.
- scaffold와 external dependency는 현재 work item에 필요한 범위만 추가한다.
- 실제 Secret, credential, 개인 환경 파일과 사용자 식별 가능 정보를 커밋하지 않는다.

## 상태 기록

- work item을 시작하거나 완료·차단하면 Current Work Queue를 같은 변경에서 갱신한다.
- milestone이나 다음 작업이 바뀌면 `PROJECT-DECISIONS.md`와 루트 `README.md`도 갱신한다.
- `COMPLETE`는 acceptance와 재현 가능한 check/evidence가 확인된 경우에만 사용한다.
- `BLOCKED`에는 exact blocker, 마지막 확인 결과와 해제 조건을 기록한다.

## Git

- 커밋 제목과 본문은 한글로 작성한다. 기술명과 코드 식별자는 원문을 사용할 수 있다.
- 관련 없는 변경을 한 커밋에 섞지 않는다.
- destructive Git 명령과 repository-wide 설정 변경은 명시적 승인 없이 수행하지 않는다.
