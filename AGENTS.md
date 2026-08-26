# 작업 규칙

## 작업 재개

### 마일스톤 Baseline Audit

마일스톤 작업을 시작할 때는 source code를 작성하거나 수정하기 전에 repository baseline audit를 수행한다.
각 milestone work item은 다음 상태를 독립적으로 판정하고 근거를 남긴다.

- `IMPLEMENTED`: YES / NO
- `WIRED`: YES / NO
- `QUALIFIED`: YES / NO
- `ACCEPTANCE_CLOSED`: YES / NO
- `MISSING`: 실제 남은 acceptance 작업 목록

`IMPLEMENTED=YES`인 기능은 다시 구현하지 않는다. 기존 구현을 대체하거나 재작성해야
한다면 기존 구현 위치, 재사용할 수 없는 이유, 기존 구현 수정으로 해결할 수 없는 이유,
대체 시 영향 범위를 먼저 제시하고 승인받는다. 이후에는 `MISSING`으로 확인된 acceptance
gap만 구현한다. 상태 판정에는 가능한 경우 commit, file, symbol/function, test,
workflow/run evidence를 연결한다.

1. `PROJECT-DECISIONS.md`에서 현재 단계와 남은 결정을 확인한다.
2. `docs/README.md`에서 현재 작업에 필요한 canonical 문서를 찾는다.
3. `docs/planning/02-current-work-queue.md`에서 active/next work item, 선행조건과 acceptance를 확인한다.
4. 한 번에 하나의 work item만 `IN_PROGRESS`로 두고 그 범위만 수행한다.

현재 재개 지점은 `M8-005 — least-privilege typed resolver network-policy helper`이다. Queue가 이후 갱신되면 Queue의 현재 상태를 우선한다.

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

## 연속 작업

- 사용자가 “계속 진행”, “멈추지 말고”, “완주”를 요청한 활성 작업에서는
  커밋·푸시·개별 test 통과를 종료 조건으로 취급하지 않는다. 다음 work item 또는
  같은 item의 다음 구현·검증을 즉시 계속한다.
- progress commentary 뒤에는 실제 다음 tool action을 수행한다. 사용자 결정, 새
  권한, 외부 CI/비동기 결과 확인 또는 요청 범위의 실제 완료 전에는 `final`로
  turn을 종료하지 않는다.
- 외부 CI 결과가 완료 조건인 경우에만 현재 결과·권장 확인 시점과 함께 한 번
  종료할 수 있다. 종료 뒤에는 자동 재개할 수 없음을 정확히 알린다.

## Git

- 각 milestone은 `milestone/<id>-<topic>` 전용 branch에서 진행하고 milestone qualification 후 PR로 `main`에 merge한다.
- work item·독립적으로 검증 가능한 변경 단위별로 의미 있는 중간 commit을 남기고, 내용 없는 milestone 최종 commit은 만들지 않는다.
- milestone branch를 원격에 추적하되 `main`으로의 merge는 milestone acceptance·qualification이 통과한 후 reviewer가 수행한다.
- 커밋 제목과 본문은 한글로 작성한다. 기술명과 코드 식별자는 원문을 사용할 수 있다.
- 관련 없는 변경을 한 커밋에 섞지 않는다.
- destructive Git 명령과 repository-wide 설정 변경은 명시적 승인 없이 수행하지 않는다.
