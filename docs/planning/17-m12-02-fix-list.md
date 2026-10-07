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
