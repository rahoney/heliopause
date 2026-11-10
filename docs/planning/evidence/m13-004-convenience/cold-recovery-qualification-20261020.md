# M13-004 — terminal 기록의 cold service 재개 검증

Parent `d311f01`. [사전 감사·map·재검토](./cold-recovery-plan-20261020.md)를 따랐다.
`Recover`가 SDK/활성 record/current/config와 EX를 확인한 뒤 private service 분기를
사용한다. Healthy active는 유지한다. Inactive이면 독립적인 stopped 판정과 global
resource drain을 먼저 요구하고 기존 fixed start로 실제 identity를 확인한다.
UNKNOWN/drain/cancel/start failure는 terminal marker를 해제하지 않는다. 이후의 기존
runtime health/owned cleanup/history retirement 순서도 유지한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| terminal cold service 분기 | YES | YES, actual Recover | YES, local branch/gate mechanics | YES, component |
| 실제 새 signed native9/systemd/crash/reboot | PARTIAL | PARTIAL | NO | NO |

원본은 `docs/github-logs/self-update-20261020/cold-recovery-*`다. 동일 전체 입력의
canonical format/focused와 Quick43.67s/security8.22s/vulnerability13.10s/docs0.25s,
관련 releaseactivation/installer Race7.29s는 exit0이다. Focused1.36s는8개의 새 terminal
경로와 기존 service/config consumers를 확인한다. Active 유지, inactive 재개, unknown,
busy, start failure, cancel과 비어 있는 잘못된 성공 결과를 각각 확인했다.

별도 service namespace0.12s/exit0은 실제 root PID/pidfd/Unix socket PID·GID,
고정 command fault cause, partial config restore/unknown 보존의 기존 경계를 확인했다.
Crypto/systemd command는 mock이며 실제 재부팅/서명 전환의 성공이 아니다.
완료한 binary28,946,435B는 삭제했고 host native installation/state는 전후
ABSENT_CONFIRMED다. Host service/config/기존 설치는 변경하지 않았다.

두 번째 대조에서 active query 실패만으로 재시작하지 않으며 최신 독립 stopped facts,
global drain, exact signed record/config와 고정 start의 실제 PID/socket/cgroup identity를
모두 요구함을 확인했다. 실패 시 복구 marker를 남기는 기존 동작은 유지한다.
Anonymous copy stage의 SIGKILL 소유권·잔여물 처리와 verified initial loader,
실제 새 signed base/target·systemd/crash/reboot qualification은 계속 남는다.
M13-004는 IN_PROGRESS이며 source component 통과로 전체 완료를 주장하지 않는다.
