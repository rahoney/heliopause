# M13-004 — 요청별 frozen content binding 검증

Parent `d2ff970`. [Lifecycle source map](./activation-lifecycle-plan-20261020.md)의
기존 InstallUpdateService 연결 gap만 구현했다. M13-004는 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| 요청별 Artifact/Verification/Inspection 바인딩 | YES | YES, 기존 update pipeline | YES, local | YES, component |
| 실제 self-update session/Promotion/client/CLI | NO, 다음 연결 | NO | NO | NO |

`ports.InstallUpdateContentSession`은 기존 session과 실제 세 content port를 함께
제공한다. Application은 요청별 기존 service 복사본만 바인딩한다. Evidence, Policy,
Manifest, staging, derivation의 기존 owner와 순서는 그대로다. 기존 생태계 session은
추가 interface를 구현할 필요가 없다.

서로 다른2.0.0/3.0.0 graph의 resolve를 barrier로 겹친 실제 동시 호출에서 각 session의
Acquire/Verify/Inspect, 독립 operation ID와 해당 version을 확인했다. Shared base의
content/promoter는 사용되지 않았으며 이후 원래 Install은 기존 binding으로 성공했다.
새 session의 incomplete/BLOCK은 기존 Policy가 판정하고 Manifest/staging/promotion에
도달하지 않는다. 기존 update guard/primary-cleanup 회귀도 통과했다.

원본은 `docs/github-logs/self-update-20261020/content-session-*`다. 전체 source inventory는
tracked+명시 후보의 같은 bytes와 parent를 연결한다. Pinned Go1.26.9와 canonical
format 뒤 focused/Quick48.13s/security7.33s/vulnerability10.82s/docs0.23s,
기존 runtime 회귀 Race14.02s와 Application 전체 Race3.49s는 모두 exit0이다.
설치본·서비스·공개 release 변경은 없다. 새 signed native4/full activation qualification을
대신하지 않는다. Go checker 임시 실행 파일은 도구 종료 시 제거됐고 활성 private
source/cache는 바로 다음 client 연결 검사에 재사용한다.
