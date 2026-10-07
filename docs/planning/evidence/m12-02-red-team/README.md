# M12-02 최종 검토 근거

## 시작 baseline audit — 2026-10-07

기준 커밋: `19411caa06009b614e18c7c58771b5e19f9d26af`.
기존 milestone branch `milestone/m12-go-cargo-terraform`에서 이어간다.
추적 파일은 clean이며, 사용자 미추적 문서·오답노트는 commit 대상에 넣지 않는다.

| 판정 | M12-02 시작 상태 | 근거 |
| --- | --- | --- |
| IMPLEMENTED | YES — 기존 검사·승격·transaction 구현 재사용 | `internal/application`, `internal/artifact`, `internal/sandbox`, `internal/promotion/staging.go`, `internal/promotion` |
| WIRED | YES — 기존 public CLI와 Required 연결 재사용 | `internal/bootstrap`, `.github/workflows/heliopause-ci.yml` |
| QUALIFIED | NO — final red-team 미실행 | M12-005 기능 qualification은 완료했으나 이번 검토를 대신하지 않음 |
| ACCEPTANCE_CLOSED | NO | 최종 검토 결과·확정 결함의 remediation·최종 후보 CI가 필요 |
| MISSING | 아래 항목 | 검토 전 결함의 존재나 부재를 추정하지 않음 |

1. Source/registry identity, exact graph freeze, integrity, observation/attribution,
   Policy/Evidence, transaction/rollback, cross-ecosystem regression, release impact
   여덟 영역의 실제 책임 흐름과 거부·실패 조건 검토.
2. 발견한 release-blocking 결함의 동일 입력 재현과 기존 owner 수정, 정상·부정
   회귀 및 영향받는 실제 소비자 검증.
3. 현재 후보에 연결한 canonical check·최종 CI 원본과 검토 근거, FIX 목록 및
   완료·인계 상태 기록.

## 재사용 경계

- [M12-005 결과](../m12-005-qualification/result.json)는 동일 기능 소스의
  CPU·cu126·cu130·cu132 full, 77개 요구사항과 명시적인 지원 제한을 소유한다.
- 기준 HEAD의 [CI 37575918835](https://github.com/rahoney/heliopause/actions/runs/37575918835)는
  실제 10개 job success다. 이전 PASS를 수정한 입력의 새 PASS로 기록하지 않는다.
- 기능 소스가 바뀌면 변경된 책임과 실제 소비자를 확인하고 해당 qualification을
  새 후보에서 검증한다. 문서 전용 변경은 전체 실행 입력의 bytes/mode 동일성을
  확인한 범위에서만 기존 full 증거를 재사용한다.
- 공개 릴리스를 막을 확정 결함만 [FIX 목록](../../17-m12-02-fix-list.md)에
  기록한다. 범용 framework, 신규 기능 또는 취향에 따른 재작성은 이번 범위가 아니다.
- 현재 작성자의 검토를 외부 독립 보안 감사나 GitHub reviewer 승인으로 표시하지 않는다.

## FIX-01 로컬 인과 및 회귀

[fix-01-local.json](./fix-01-local.json)은 최종 fixture와 correction source의 SHA,
동일 fixture의 baseline FAIL → correction PASS, 첫 수정의 기존 Cargo 회귀 실패와
보완 후 PASS, canonical 여섯 profile의 실제 exit/log SHA를 연결한다. 원본 로그는
`raw/`에 bytes 그대로 보존한다. 원격 소비자·CI는 아직 대기 중이며 로컬 PASS로
대체하지 않는다.

Status: IN_PROGRESS. Remaining review and remote results pending.

## FIX-02 및 검토 matrix

[fix-02-local.json](./fix-02-local.json)은 npm/PyPI 최종 fixture의 동일 입력 인과,
48개 사례 PASS·기존 회귀와 canonical profile 결과, 실제 npm local preflight 거부를
구분한다. [여덟 영역 검토](./review-matrix.md)는 owner·거부 조건·변경 범위와 현재
판정을 연결한다. 원격 실제 소비자·CPU/세 CUDA full·최종 CI가 완료되기 전에는
FIX-01/02와 M12-02 acceptance를 닫지 않는다.

## FIX-03 및 첫 후보의 원격 실패

[fix-03-local.json](./fix-03-local.json)은 기존 npm 프로젝트 준비의 동일 fixture
기준선 FAIL과 수정 후 first/retained PASS, canonical 일곱 profile을 연결한다.
`193e1aa` ordinary CI의 실제 최초 오류는 private selected control의 EEXIST이며
첫 프로젝트 offline runner는 미실행이다. 전체 10개 job 원본은 외부 local M12-02의
`remote-ci/`에 보존하며 [실패 실행](https://github.com/rahoney/heliopause/actions/runs/37586753502)을
참조한다. 해당 실행의 43개 정상 소비자·CPU full275.46s는 부분 성공이다.
같은 후보의 cu126 취소는 후보 교체이며 새 후보의 qualification을 대신하지 않는다.
Source 수정은 기존 npm 준비 owner에 한정하며 FIX-01/02 remediation을 유지한다.
