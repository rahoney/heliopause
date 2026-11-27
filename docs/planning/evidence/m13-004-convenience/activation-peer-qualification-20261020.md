# M13-004 — old controller의 kernel identity gate

기준 parent `5717520f5986767f2892a7523a62f8f4cd6f6d97`.
[사전 source map·두 번째 검토](./protected-activation-plan-20261020.md#5717520-이후의-old-controller-인증-map)에
따라 existing protected peer authorizer를 재사용하고 pidfd/group/lifecycle gate를 추가했다.
전체 M13-004는 IN_PROGRESS이며 실제 RPC/helper/lifecycle/public command는 미완료다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| actual active controller UID/GID/inode 확인 | YES | YES, managed lease 뒤 gate | YES, local/kernel mechanics | YES, primitive 범위 |
| root-side old Unix peer/pidfd/kernel groups gate | YES | YES, concrete function/tests | YES, local/kernel mechanics | YES, primitive 범위 |
| root RPC·journal/service 전환·public caller | NO | NO | NO | NO |
| actual newly signed native4 activation | YES, 기존 adapters 재사용 | NO | NO | NO |

Normal managed command는 checked installed view와 actual ordinary UID/GID 및
`/proc/self/exe` inode가 맞아야 lease를 받는다. Root-side는 SO_PEERCRED와 existing
PolicyPeerAuthorizer의 protected inode를 확인하고 pidfd로 실제 peer exit/reap을
검사한다. Groups는 bounded kernel proc status에서만 읽으며 request/SUDO_UID를
authority로 쓰지 않는다. Unsupported/unknown/closed/dead 상태는 authorized live
peer가 아니다. Root-owned signed digest 검증은 별도 installed reader가 유지한다.

원본 owner는 local `docs/github-logs/self-update-20261020/`다.

- `activation-peer-final-source.json`/`activation-peer-final-inventory.json`: exact 전체
  입력. Focused1.94s, canonical Quick35.78s/security4.92s/vulnerability11.97s/docs0.27s,
  관련 전체 Race5.18s는 exit0다. Pure malformed/missing/duplicate/excess groups와
  실제 known child pidfd의 live→exit/reap→Close를 확인했다.
- `activation-peer-namespace.json`/log: exact static binary를 별도 mount/network
  namespace에서0.22s/exit0. 보호된 actual executable로 실행한 UID1000/GID1000만
  승인했다. Same basename/different inode, foreign UID/GID와 root caller를 거부했고
  actual kernel group membership와 process death도 확인했다.
- 해당 namespace의 installed view는 **test-only crypto mock**이다. 실제 signed
  current native4·root RPC·새 helper/doctor/다음 실제 검역 완료가 아니다. Production
  view issuer는 기존 fixed-root SDK를 사용하는 ReadInstalledRelease뿐이다.
- 완료 binary27,918,092B를 즉시 삭제했고 fixtures는 private tmpfs 종료로 소멸했다.
  두 host root 경로는 전후 ABSENT_CONFIRMED이며 실제 runtime/config/service 변경은0이다.
- 문서 전달·index/commit proof는 activation-peer-delivery 자료를 따른다.

두 번째 대조에서 기존 authorizer의 empty expected digest를 digest pin으로 오인하지
않았고 signed file identity/kernel caller/liveness/Policy를 서로 대체하지 않았다.
Generic root command, broad signal/kill, user-chosen UID/target과 public 성공 stub은 없다.
다음은 이 peer gate 뒤에 fixed bounded bytes protocol/root stage와 실제 lifecycle을 연결한다.
