# M13-004 — 실제 native helper의 inert entry 검증

기준 `119a605`; [source 이전 map/재검토](./native-entry-plan-20261020.md)를 따랐다.
Generic M6의 raw ELF 무인자 실행/nonzero INCOMPLETE는 변경하지 않았다.
Actual service engine/observer state machine을 재구현하거나 RESOURCE로 면제하지 않았다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| observer default/help와 기존 identity/실제 argv 유지 | YES | YES | YES, canonical C++ full latch/actual binary | YES, entry 범위 |
| network helper inert entry/명시적 serve | YES | YES, CI/native/package 호출 | YES, Go/actual binary/local regression | YES, entry 범위 |
| actual signed native4 전체 quarantine/activation | NO | NO | NO | NO |

Network helper는 무인자/help/version에서 runtime을 준비하지 않고 actual typed service는
serve만 받는다. 기존 ServeNetworkPolicy를 재사용하며 설치/recover/임의 target/UID API는
추가하지 않았다. 실제 CI startup와 native/package ExecStart를 함께 바꿨다. Package의
RuntimeDirectory/socket 예시도 actual observation-resource parent와 맞췄으며 disabled
기본값/명시적 runtime 준비 조건은 유지한다. Historical raw evidence는 변경하지 않았다.

원본은 `docs/github-logs/self-update-20261020/native-entry-*`다.

- 전체 tracked+명시 candidate equality의 canonical Quick38.10s/security5.51s/
  vulnerability11.34s/docs0.23s, 관련 focused2.28s와 Race9.01s가 exit0이다.
- Canonical pinned gVisor source/Go SDK/Bazel SHA/builder image의 release observer
  build와 전체 latch test273.86s/exit0. Full latch의 실제 observer_main default/help
  회귀와 기존 profiles/ledger/attribution tests를 함께 통과했다. Built binary의
  default/help/identity를 실제 실행해 모두 exit0이고 identity는 unchanged pinned
  gVisor commit이다. Host floating compiler나 budget/skip은 사용하지 않았다.
- 실제 Go network helper binary의 default/help/version도 exit0이고 stderr는 비었다.
  Native probe operator 첫 호출은 Python syntax parsing에서 build 전에 멈췄으며 operator만
  고쳐 같은 source를 검사했다. 이를 제품 실패/수정 성공이라고 기록하지 않는다.
- Updated native unit의 root transaction8 cases를 같은 source namespace0.17s/exit0로
  다시 확인했다. Crypto/service/health mocks 범위이며 actual activation이 아니다.
- Completed observer3,084,896B, policy helper7,273,595B, namespace binary28,754,391B를
  사용 직후 삭제했다. Canonical C++ build inputs와 empty Docker config도 제거됐고
  namespace payload는 종료로 소멸했다. Host installation/state는 전후 ABSENT_CONFIRMED이며
  실제 설치/config/service 변경은0이다.

두 번째 대조에서 actual argv0/default와 service 시작을 구분하고 전체 set의 기존
synthetic positive를 실제 새 signed release의 성공으로 확대하지 않았다. 다음은
existing pipeline의 per-request content binding, root client/Promotion, public CLI와
실제 helper asset producer/native bootstrap 연결이다. 전체 M13-004는 IN_PROGRESS다.
