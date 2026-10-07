# M12-02 Final Red-Team Fix List

- 파일명: `17-m12-02-fix-list.md`
- 시점: M12 Ecosystem Expansion 전체 qualification 완료 직후
- 상태: IN_PROGRESS / Ready: Yes

이 문서는 M12에서 추가한 **PyTorch, Go Modules, Cargo/crates.io, Terraform Provider**
지원과 기존 npm/PyPI/GitHub 경로를 함께 최종 red-team 검토한 뒤,
**public release를 실제로 막아야 하는 수정 사항만** 기록하는 문서다.

검토 범위:

```text
source/registry identity
exact dependency graph freeze
digest/checksum/signature verification
sandbox observation/attribution
Policy/Evidence binding
project transaction/rollback
cross-ecosystem regression
release integration impact
```

기록 규칙:

- fail-open, trust-boundary bypass, wrong-artifact Promotion, rollback/data-loss,
  required observation break처럼 release-blocking인 항목만 FIX-N으로 추가한다.
- 취향, 알고리즘 대안, 장기 개선 아이디어, 신규 ecosystem 제안은 넣지 않는다.
- release-blocking finding이 없으면 아래 한 줄로 종료한다.

```text
NO_RELEASE_BLOCKING_FINDINGS
```

Status: IN_PROGRESS
Ready: Yes — M12-001~005 기능 qualification·feature freeze 완료
Next: final red-team review and release-blocking remediation, if found

## 검토 기준선

기준 커밋은 `19411caa06009b614e18c7c58771b5e19f9d26af`다. 기존 구현과
M12-005의 검증은 재사용하며, 위 여덟 영역의 최종 검토와 확정 결함의 수정·회귀,
최종 후보 CI 및 재현 가능한 검토 근거가 이번 acceptance에 남는다.
[Baseline audit](./evidence/m12-02-red-team/README.md)가 상태 판정과 증거 경계를
기록한다. 검토 완료 전에는 release-blocking finding의 부재를 선언하지 않는다.

## FIX-01 — Go/Cargo rollback 보관 영역의 외부 파일 삭제

Status: IN_PROGRESS — remediation·동일 입력/기존 소비자 로컬 회귀 통과, 원격 검증 대기.

- 기준: `19411ca`, `internal/promotion/go_transaction.go`와
  `internal/promotion/cargo_transaction.go`의 commit/fail cleanup.
- 승인 publication 중 rollback 보관 디렉터리에 외부 파일이 추가되면 성공 commit과
  실패 후 rollback 모두 `RemoveAll`로 그 파일을 삭제했다. 실제 transaction을 사용하는
  `TestProjectTransactionPreservesUnexpectedBackupContent`의 두 생태계 × 두 종료 경로가
  모두 FAIL했다. 이는 사용자 데이터 보존·불확실한 rollback의 fail-closed 계약에 위배된다.
- 수정 범위: 기존 transaction owner에서 원래/선택한 control·metadata와 보관 경계의
  identity/content/mode를 대조하고, 정리는 확인한 파일의 개별 삭제와 빈 디렉터리
  삭제로 제한한다. 외부 파일·변경된 백업은 보존하고 다음 작업을 차단하는 journal을 남긴다.
- 최초 원본: local M12-02 `backup-before.log`; portable 검증 근거는
  [review evidence](./evidence/m12-02-red-team/README.md)에 연결한다.
- 최종 fixture를 기준 production에 적용하면 FAIL, 수정 후 새 40개 사례와 기존 Cargo
  publication-race 사례가 PASS다. 첫 수정의 전체 선검사는 안전한 원본 복원까지 막아
  기존 테스트에서 반증됐으며 파일별 검증으로 보완했다. canonical quick/security/
  vulnerability/fuzz/freshness/release-gate는 모두 actual exit 0다. 이 결과를 새 CLI·원격
  full qualification으로 확대하지 않는다.

## FIX-02 — npm/PyPI backup cleanup 및 npm rollback의 외부 파일 삭제

Status: IN_PROGRESS — remediation·동일 입력/기존 소비자 로컬 회귀 통과, 원격 검증 대기.

- 기준: `19411ca`, `internal/promotion/npm_transaction.go`와
  `internal/promotion/pypi_venv_transaction.go`.
- npm/PyPI의 정상 commit·실패 rollback 후 `RemoveAll`이 보관 영역의 외부 파일을
  삭제한다. npm rollback은 이미 publish된 `node_modules`/`.heliopause`의 외부 파일도
  삭제한다. 기존 transaction fixture와 instance-local PyPI fault seam을 사용하는
  여섯 실제 사례가 FAIL했으며 원본은 local M12-02 `legacy-backup-before.log`다.
- 수정 범위는 기존 owner의 backup/member identity 및 내용 검증과 확인된 member의
  개별 정리다. 원래 오류와 불확실한 데이터·recovery 영역을 보존하고 다음 작업을
  차단한다. 신규 기능·graph/Policy/observation 권한 변경은 하지 않는다.
- npm의 경쟁 destination 및 교체된 guard도 덮어쓰거나 삭제하지 않는다. PyPI는
  controller가 append한 journal의 길이·digest와 신원도 확인한다. 최종48개 사례와
  기존 npm/PyPI 회귀, canonical 일곱 profile은 actual PASS다. 새 실제 npm 프로젝트
  최초/retained 소비자는 기존 Required test에 연결했으며 local registered runtime
  identity preflight가 거부해 body는 NOT_RUN이다. Current CPU/세 CUDA full·실제
  프로젝트·전체 CI가 남는다. [로컬 근거](./evidence/m12-02-red-team/fix-02-local.json)와
  [여덟 영역 검토](./evidence/m12-02-red-team/review-matrix.md)를 참조한다.

## FIX-03 — 기존 npm 프로젝트의 selected control 준비 충돌

Status: IN_PROGRESS — 동일 fixture의 기준선 실패·수정 후 로컬 회귀 통과, 실제 소비자 대기.

- `193e1aa`의 [CI 37586753502](https://github.com/rahoney/heliopause/actions/runs/37586753502)는
  새 `TestLinuxNPMPromotionIntegration/project-first-and-retained`에서 runner 호출 전에
  `package.json: file exists`로 실패했다. 기존 new-target 설치 성공과 프로젝트 설치를
  구분한다. 이 실행의 CPU full·Go/Cargo/TF 소비자 성공은 전체 CI 실패를 덮지 않는다.
- `npm_promotion.go`는 기준 `19411ca`와 해당 실패 후보에서 동일하다.
  `privateWorkspace`가 복사한 frozen controls와 `preparePromotionProject`가 O_EXCL로
  생성하는 selected controls가 같은 경로여서 충돌했다. 같은 최종 fixture를 기준
  production에 적용해 actual exit 1로 재현했다. FIX-01/02의 회귀로 단정하지 않는다.
- 기존 private workspace의 원본 복사본은 유지하고 빈 `selected` 하위 디렉터리에서
  complete Verified Set의 controls를 생성한다. 기존 offline runner·검증·transaction을
  재사용하며 no-replace 쓰기와 host controls 보호를 유지한다.
- 수정 후 composed first/retained 회귀와 canonical 일곱 profile은 actual exit 0다.
  [로컬 근거](./evidence/m12-02-red-team/fix-03-local.json)는 원본·source·fixture를
  연결한다. 실패 후보의 진행 중 cu126은 수정 후보로 전환하기 위해 취소했으며 제품
  실패나 qualification 성공으로 표시하지 않는다. 수정 후보의 실제 npm 소비자와
  CPU/cu126/cu130/cu132·최종 CI가 완료되어야 FIX-03과 M12-02를 닫는다.
