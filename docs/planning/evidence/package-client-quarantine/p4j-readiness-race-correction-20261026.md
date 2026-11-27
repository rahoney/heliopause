# P4-J qualification의 기존 readiness 시험 경쟁

기준 c1a7659의 sandbox tracked files 전체와 현재 bytes가 같다. 첫 전체 관련 Race는
CLI/bootstrap/promotion/PyPI/application PASS이나 기존
`TestReadinessRetriesOrdinaryFailureWithFreshAdmissionNonce`의 local `nonces` slice에서
responder goroutine의 append와 시험 goroutine의 read가 충돌했다. 새 project adoption이
이 오류를 만들었다고 판단하지 않는다. 첫 원본/전체 비교는
`docs/github-logs/package-client-quarantine-20261021/p4j-readiness-first-assessment.json`과
`p4j-project-adoption-v2-full-affected-race`가 소유한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| ordinary readiness retry/fresh nonce 시험 | YES | YES | NO(current Race) | NO(current qualification) | local fixture slice synchronization |
| production admission/retry/nonce protocol | YES | YES | 기존 named 결과 보존 | 이번 수정 대상 아님 | production 변경 없음 |

기존 시험을 재구현하거나 coverage를 줄이지 않는다. 이미 import한 sync의 Mutex로 local
fixture의 callback append/status read와 최종 assertion read를 보호한다. 실제 두 probe와
서로 다른 fresh nonce·ordinary retry 정상 완료 assertion은 그대로다. 최초 full Race의 두 충돌 stack은 직접 실패 증거다. 미수정 focused20회는 PASS였으므로
항상 재현된다고 주장하지 않는다. 교정 뒤 같은 focused20회와 전체 sandbox Race 및
canonical Quick/security/docs로 확인한다.
초기 full Race의 다른 PASS 소비자는 소스 불변인 경우 보존하며 모두 재실행하지 않는다.

교정 결과: focused20회10.495s/전체 sandbox Race15.860s, 최종 canonical
Quick59.770s/security9.813s/docs0.258s가 PASS다. 최초 전체308.801s/exit1과
미수정 focused20회3.127s/PASS 원본을 유지한다. 교정 후 이 named fixture 범위는
QUALIFIED=YES/ACCEPTANCE_CLOSED=YES/MISSING=없음이다. 다른 다섯 package의 앞
실제 PASS를 소스 불변으로 보존한다. 모든 원본은 같은 raw owner의 p4j-readiness-*와
p4j-project-adoption-v3-*가 소유한다. 새 full CI는 실행하지 않았다.
