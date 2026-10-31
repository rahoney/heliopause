# P3-B5e Bun project consumers의 hosted CI 연결 감사

728f65013317f61e9b3bf3d102072e0f41cc1d47/treef033f4129c4b57055efcea884fdbd91c8211e428/
2459files에서 B5d native original local qualification·commit 뒤 착수한다. M13-004 하나만 IN_PROGRESS다. Read: B4d-e CI412 전체
로그/source qualification, B5a/b1/b2/c/d 원본·영수증, canonical 확장 계획§5, ci L06·관련
sandbox/evidence, `ci-bun-runtime.sh`, PNPM project supply·owning helper lifecycle와 workflow
validator/tests·quality/runtime owner fixture.

| 책임/기준 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun frozen archive/publisher/native/notices supply·exact cleanup | YES | YES(native/quarantine) | YES(local/CI412) | YES(existing scope) | project 소비자의 같은 공급 재사용 |
| Bun native9/경계5·quarantine2 hosted commands | YES | YES | YES(CI412 same7544 tree) | YES(existing scope) | 기존 경로 보존 |
| Bun cache/transaction/CLI/native ownership actual | YES | YES(local) | YES(local B5a..B5d named scope) | NO(hosted) | compiled bootstrap client와 typed owning helper 연결 |
| workflow/validator의 Bun project sequence | NO | NO | NO | NO | 고정 project mode·별도 bootstrap cwd·보호 client 복원 |

## Exact edit map·두 번째 검토

1. `scripts/ci-bun-runtime.sh`: 기존 zero-arg native mode를 보존하고 정확히 한 `project`
   argument만 별도 허용한다. Unknown/extra arg는 supply 전에 실패한다. Archive download/
   materialize/installed-root/primary-preserving cleanup body는 한 번만 재사용한다. Project mode는
   internal/bootstrap cwd에서 cache→transaction→CLI→native original의 exact opt-in 명령을
   failfast/기존 deadline으로 수행한다. Script가 client/helper를 설치·시작·소유하지 않는다.
2. Workflow owning shell은 기존 native mode 뒤 canonical haa_install_ci_client로 이미
   컴파일한 bootstrap test client를 설치하고 helper를 시작한 뒤 project mode를 호출한다.
   이후 sandbox client/helper를 복원하고 기존 후속 실제 소비자를 계속한다. PNPM/UV/CPU/CUDA
   순서·Required/jobs/timeouts와 source/version pins는 유지한다. Child가 자기 helper를 추가로
   시작하거나 echoed/skipped command로 coverage를 대신하지 않는다.
3. `scripts/check/bun_runtime_workflow.go`·tests: active exact mode/argument guard·공급 body·
   두 package subshell/모든 env/filter/deadline·성공 상태 전파, owning bootstrap/helper/restore
   순서를 검증한다. Missing/duplicate/comment/echo/||true/wrong cwd/filter/mode/client/helper
   negative와 기존 positive를 유지한다. YAML 정식 parsing/step owner를 기존 validator로 확인한다.
   다른 workflow validator의 count/fixture가 새 mode를 사용하는지 caller 대조 후 정확히 갱신한다.
4. 동일 수정 script의 project mode를 실제 private namespace에서 실행한다. Namespace 안에도
   기존 native installation이 없는 상태에서 frozen ZIP materialization부터 시작하고 actual
   cache/transaction/CLI/native proof를 수행한다. Child 뒤 runtime/supply work 부재, owning helper
   wait0/Host 전체 동일을 확인한다. Native/default mode는 공급/두 원래 명령이 변하지 않았음을
   source와 validator·필요한 영향 회귀로 연결한다. Canonical workflow/Quick/security/docs와
   checker 전체 Race 및 전체 index blobs/modes/disk/tree를 남긴다.

기존 공급/관찰기/Policy engine을 대체하거나 테스트를 skip하지 않는다. Final 원격 실행은
기능 묶음·공통 행동 변화 경계를 따라 한 번 push하고 같은 profile 최근 실측+실제 새 검사
시간으로 timer를 등록·소비한다. DUE와 commit은 종료 조건이 아니다. 명시 Bun Artifact runtime/
전체 P3와 최종 영향 profile qualification은 별도이며, 이 연결만으로 완료로 표시하지 않는다.

## 로컬 연결 검증

[로컬 영수증](./p3b5e-bun-project-ci-local-20261025.json)의 실제 동일 project script는
519.694s에 네 top-level 검사(cache46.97s/transaction151.89s/CLI182.70s/native124.28s)를
모두 실행했다. No-SKIP, helperwait0/CONFIRMED_STOPPED, child 설치 root/공급 work 부재와
Host 전체 동일을 확인했다. Namespace/operator의1200/1320s는 네 소비자 합계의 관찰
한도이며 각 명령5m·workflow jobs/timeouts는 유지한다. Workflow2.079s/Quick67.189s/
security11.872s/docs0.286s·checker 전체 Race38.437s가 PASS다. 기존 공급·cleanup·default
native subshell은 baseline과 bytes가 같다. 첫 validator scope 오류와 수정 근거도 보존했다.
Hosted qualification은 다음 기능 묶음 push의 실제 결과로 닫으며 지금은 로컬 연결만 closed다.
