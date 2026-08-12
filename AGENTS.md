# 작업 규칙

## 작업 재개

1. `PROJECT-DECISIONS.md`에서 현재 단계와 남은 결정을 확인한다.
2. `docs/README.md`에서 현재 작업에 필요한 canonical 문서를 찾는다.
3. `docs/planning/02-current-work-queue.md`에서 active/next work item, 선행조건과 acceptance를 확인한다.
4. 한 번에 하나의 work item만 `IN_PROGRESS`로 두고 그 범위만 수행한다.

현재 재개 지점은 `M0-004 — Canonical check runner foundation`이다. Queue가 이후 갱신되면 Queue의 현재 상태를 우선한다.

## 구현과 문서

- 상세 결정은 관련 canonical leaf 문서에 한 번만 기록한다.
- 구현 전에 work item의 `Read` 문서를 확인한다.
- 기존 Architecture·Domain·Security invariant를 구현 편의로 변경하지 않는다.
- 미래 milestone의 placeholder package, config, interface 또는 CI job을 만들지 않는다.
- scaffold와 external dependency는 현재 work item에 필요한 범위만 추가한다.
- 실제 Secret, credential, 개인 환경 파일과 사용자 식별 가능 정보를 커밋하지 않는다.

## 자동화된 코드 점검

- 코드 점검은 수동 전수 확인보다 repository의 canonical check profile과 고정된 formatter, compiler/type checker, linter, validator를 먼저 실행한다.
- Go source가 존재하면 `gofmt` format check, `go build`/test build-validity, `go vet`, pinned Staticcheck와 해당 test를 우선 사용하고 결과를 근거로 필요한 부분을 정밀 검토한다.
- canonical `scripts/check`가 도입된 뒤에는 개별 명령을 임의로 재구성하지 않고 현재 작업에 해당하는 profile을 먼저 실행한다.
- check profile은 read-only 원칙을 지키며 source 변경은 명시적인 `format` profile에서만 수행한다.
- 도구가 없거나 version이 다르면 floating `latest`나 임의 PATH binary로 대체하지 않고 repository lock과 bootstrap 절차를 따른다.
- 자동화 도구의 성공은 Architecture·Domain·Security invariant에 대한 사람 검토를 대체하지 않는다.

## 상태 기록

- work item을 시작하거나 완료·차단하면 Current Work Queue를 같은 변경에서 갱신한다.
- milestone이나 다음 작업이 바뀌면 `PROJECT-DECISIONS.md`와 루트 `README.md`도 갱신한다.
- `COMPLETE`는 acceptance와 재현 가능한 check/evidence가 확인된 경우에만 사용한다.
- `BLOCKED`에는 exact blocker, 마지막 확인 결과와 해제 조건을 기록한다.

## Git

- 커밋 제목과 본문은 한글로 작성한다. 기술명과 코드 식별자는 원문을 사용할 수 있다.
- 관련 없는 변경을 한 커밋에 섞지 않는다.
- destructive Git 명령과 repository-wide 설정 변경은 명시적 승인 없이 수행하지 않는다.
