# CI414 Quick fixture lifecycle baseline

첫 실제65분 timer는03:48:42.522450UTC에 발화했고03:49:19UTC에 결과를 소비했다.
Source294bb57/checkout1a04336의 Quick 실패와 Linux Integration 미완료를 구분하며
Required는 아직 생성 전이다. CI413을 다시 열거나 새 push/재실행/취소하지 않았다.

Quick 실제 최초 오류는 TestAcquisitionFailureAndStateLock 본문 뒤 TempDir cleanup의
repository/.git/objects ENOTEMPTY다. 테스트 본문의 acquisition/state-lock assertion 실패나
새 package-client 제품 오류로 표기하지 않는다. 원본 ci414-first-status-*와 exact ci414-quick-decoded.log를 보존한다.
CI Git2.55.0과 로컬 Git2.43.0을 구분한다. 현재 test fixture는 git init/add/commit 후
명시적인 maintenance lifecycle 설정 없이 TempDir cleanup을 사용한다.

## 감사·수정 지도·두 번째 검토

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 runtime updater acquisition/state-lock와 fixture | YES | YES | YES(기존 범위) | YES(기존 범위) | 제품 재구현 없음 |
| Git auto maintenance와 fixture 종료 결합 | NO | NO | NO | NO | foreground lifecycle·관련 회귀/CI Git2.55 확인 |
| CI414 전체 qualification | YES(실행) | YES | NO | NO | Quick 실패·Linux/Required·whole checkout/전체 로그 |

고정 공식 Git v2.55.0 builtin/gc.c의 maintenance_run_tasks는 object database의 maintenance
lock을 획득하고 optional daemonize 뒤 background task/lock 해제를 수행한다. 공식 config는
maintenance.autoDetach가 미지정이면 gc.autoDetach를 사용하고 둘 다 미지정이면 true라고
명시한다. CI 당시 process trace는 없으므로 detached maintenance를 역사적 단일 원인으로
확정하지 않는다. 재현/검증은 fixture lifecycle 책임과 동일 Git 환경을 구분해 기록한다.

[고정 source](https://github.com/git/git/blob/v2.55.0/builtin/gc.c) 및
[고정 설정 문서](https://github.com/git/git/blob/v2.55.0/Documentation/config/maintenance.adoc)를 따른다.
새 dependency/도구 pin 또는 floating Git 설치, product/runtime/CI 예산·coverage 변경은 하지 않는다.
필요한 수정은 scripts/update-runtime/update_test.go의 disposable fixture에만 local
maintenance.autoDetach=false/gc.autoDetach=false를 설정해 명령 종료가 maintenance 완료까지
기다리게 하는 것이다. 실제 저장소/global Git config는 변경하지 않는다. 기존 fixture와 제품
source는 대체하지 않고 한정된 lifecycle gap만 보완한다. Canonical profile과 관련 실제 tests로
로컬 검증하고 CI Git2.55/전체 Required는 이후 묶음 후보까지 미완료로 남긴다.

두 번째 검토: retry/sleep 또는 실패 무시로 cleanup을 green으로 만들지 않는다. 테스트 자체가
만든 Git 작업을 살아 있는 상태로 남기지 않는 것이 책임이며 제품 acquisition/security/locking은
유지한다. 과거 CI 실패의 인과나 Git 버전 전체 지원으로 일반화하지 않는다.

최종 extension timer는04:09:19.000147UTC 발화/04:10:00UTC 실제 결과 소비로 끝났다.
CI414는8SUCCESS/Quick·Required FAILURE이며 Linux72/observer121/CPU258.26s·helper9wait0/
final confirmed cleanup은 통과했다. All10 checkout1a04336의 전체 tree5a140885/2596가 frozen
294bb57과 같고10 decoded logs1,175,342B/100chunks의 exact readback/SHA를 확인했다.
두 timer가 기다리는 동안 E/M1/M2/M3/M4-A1을 로컬 검증·중간 commit했고 새 push로 실행을
취소하지 않았다. Native upgrade/나중 local source/final CUDA로 이 성공을 확대하지 않는다.

Fixture 변경은 기존 본문에 scoped Git foreground 설정만 추가했고 제품 source는 불변이다.
Canonical Quick87.543s/security9.220s/docs0.246s 및 전체 updater Race가 로컬 Git2.43에서
PASS다. Historical 단일 원인 해결이나 CI Git2.55 qualification으로 쓰지 않으며 후자는 다음
필요한 기능 묶음 CI까지 OPEN으로 둔다. [관찰 receipt](./ci414-backend-and-fixture-observation-20261026.json)가
source·로그·timer·로컬 검증 및 남은 경계를 소유한다. 실제 repository/global Git 설정이나
CI coverage/timeout/실패 조건은 변경하지 않았다.
