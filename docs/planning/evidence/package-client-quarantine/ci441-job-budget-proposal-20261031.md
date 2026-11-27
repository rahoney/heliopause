# CI441 — Linux 전체 job 예산 변경 검토안

상태: 2026-10-31 사용자가 “조정해도되.”로150분 조정을 승인했다.
한 줄 후보의 workflow2.135s/docs0.372s/security13.478s가 PASS다. Push/CI 재실행은 아직 하지 않았다.

[확장 계획 §5](../../20-package-client-quarantine-plan.md#5-검증-matrix와-자원-정리)은
“Workflow 자체·Required·timeout·coverage·보안 거부 조건은 유지한다.”라고 명시한다.
따라서 현재120분을150분으로 바꾸는 한 줄은 이 유지 조건의 명시적으로 승인된 예외다.
제품/개별 test timeout이나 필수 검사·test membership·Required10·새 job은 변경하지 않는다.

변경 후보: `.github/workflows/heliopause-ci.yml`의 `gvisor-integration.timeout-minutes`
`120 → 150`. 나머지 workload/권한·Source/CPU/CUDA/cleanup 조건은 그대로다.

[공식 실패·성공 기준선 비교](./ci441-time-budget-comparison-20261031.json)는
113 top parents가 PASS한 뒤2h 한도로 취소됐음을 annotation으로 확인한다.
CI439 대비 공통106개 합계674.07s 증가와 새7개563.79s가 누적됐다.
미완료9개의 이전521s를 더한 약129분은 실제 완료 결과가 아닌 보수적 추정이다.
현재 검증된 H1 새3사례179.72s까지 고려하면 약132분이므로150분의 유한 상한을 제안한다.

120분을 유지할 경우에는 설치/lifecycle과 current execution을 별도 실제 qualification
cohort로 분리하고 공통 protected setup·Required aggregate/검증 owner를 연결해야 한다.
이는 job 한 줄보다 큰 CI 구조 변경이므로, 사용자가 승인한 작은 변경으로 진행한다.

macOS default6분과 Corpus의 별도 문제는 이 제안으로 해결됐다고 쓰지 않는다.
새 remote qualification은 필요한 local source를 모은 의미 있는 frozen 후보에서 진행한다.
