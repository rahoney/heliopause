# C3 승인 기록 — 실제 missing-CLA 시험의 조건부 유예

상태: **APPROVED / APPLIED** (2026-10-17). 사용자는 연결된 C3 검토안에
“승인할게.”라고 명시적으로 응답했다. 기여자 확보의 어려움 자체를 승인으로
해석하지 않았으며, 그 뒤의 별도 승인을 적용했다.

## 승인 입력과 canonical owner

승인한 원문 SHA-256은
`894c22e62b3527982f82d1841344e6db0eb82165a094324984337577f6659ab2`다.
원문은 로컬 `docs/github-logs/m13-003-20261017/c3-approval/approved-proposal.md`,
승인 receipt는 같은 directory의 `approval.json`에 보존한다. 이는 원본의
원격 저장을 주장하지 않는다. 승인 전 canonical docs 검사는 PASS이며 임시
검사 source/cache는 삭제했다.

조건의 유일한 canonical owner는
[M13-002 C3 절](../../18-m13-production-release-operations-contract.md#m13-002--protected-branch-qualification)이다.
기존 [S/C2 결정](./remaining-decisions-20261017.md)의 과거 승인과 원본 시험
결과는 유지한다. C3는 초기 유지보수자 릴리스의 명시적 위험 수용이며, 실제
missing-CLA 시험을 통과했다는 뜻이 아니다.

## 적용 결과와 다음 작업

M13-002는 ACCEPTED_WITH_EXCEPTION / IMPLEMENTED=YES / WIRED=YES /
QUALIFIED=NO / ACCEPTANCE_CLOSED=YES (변경된 승인 범위)다.
실제 missing-CLA는 NOT_TESTED / DEFERRED, independent freshness-only는
NOT_QUALIFIED로 보존한다. 소유자·재개 조건·보호 실패 시 중단은 canonical
절을 따른다. LICENSE/CONTRIBUTING·필수 CLA App·GitHub 보호 설정·서명은
변경하지 않았다.

M13-003만 IN_PROGRESS다. 정확한 source `1ac1bf6`의
[PR37 CI325](https://github.com/rahoney/heliopause/actions/runs/39023951914)는
Required 포함10SUCCESS, actual macOS 및 Linux45 consumers/cleanup PASS다.
준비 source 검증 완료와 실제 release qualification은 구분한다. PR37 최종
문서/검사를 정리하고 reviewer의 구체적 후보 검토·정상 전달 뒤 실제 main CI와
같은 source의 build/package qualification을 이어간다. PR merge·tag/publication
실행은 이 승인에 포함하지 않으며 M13-004/005는 아직 NOT_STARTED다.

전체 판단과 원본 연결은 [post-delivery evidence](./post-delivery-review-20261017.json),
현재 작업은 [Queue](../../02-current-work-queue.md), release 준비 결과는
[preparation](../m13-003-release-build/preparation-20261017.md)이 소유한다.
