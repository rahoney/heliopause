# M13-004 — terminal journal의 재부팅 후 서비스 재개

기준 `d311f01`의 `recovery_linux.go:Recover`와 후속 native enrollment를 대조했다.
이것은 같은 M13-004의 남은 crash/reboot gap이며 전체 signed qualification은 별도다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| signed old/new·EX·journal/config·고정 service query/start/drain | YES | YES | YES, local component | YES, component | 재사용 |
| COMMITTED/ROLLED_BACK 재확인·retire | YES | YES | PARTIAL | NO, reboot | inactive service이면 오류만 반환 |
| terminal cold service 재개 | NO | NO | NO | NO | 아래 좁은 분기 |

`Recover`는 terminal journal의 signed active version/current/registration/config를
확인한 뒤 현재 active service를 요구한다. Cold boot에서 정상 inactive/dead여도
마지막 marker를 해제할 수 없다. 기존 `serviceController.stopped`와 `start`를
재사용해 별도 generic root operation 없이 고정 native unit만 재개한다.

`recovery_linux.go`의 private `recoverTerminalService`는 먼저 실제 active를 확인한다.
이미 healthy active이면 유지한다. 그렇지 않으면 독립적인 최신 stopped query로
MainPID/ControlPID0, empty cgroup, socket absence를 확인하고 global resourcesDrained를
통과한 경우만 기존 fixed start를 호출한다. UNKNOWN/query failure, foreign process,
남은 observation/consumer에서 시작하지 않는다. 기존 Root runtime EX와 fixed SDK,
exact native configs/selected active version·UID/GID를 먼저 확인하는 순서는 유지한다.
서비스 재개 후에도 기존 runtimeHealthy/drain/owned cleanup/history retirement가 필요하다.

Shared copy engine이나 installed reader를 다시 구현하지 않는다. 아직 anonymous
`.pending-*` copy의 crash 보존/정리 감사와 실제 새 signed native9/systemd/reboot는
남는다. 미확인 directory를 패턴만으로 삭제하거나 로컬 mock을 real reboot로 보고하지 않는다.

검사는 active 유지/inactive 재개/query UNKNOWN/drain 실패/start 실패/cancel 경로,
기존 operation namespace 회귀와 canonical Quick/security/docs/vulnerability 및 관련
Race다. Public tag/publish/main merge와 host service 변경은 실행하지 않는다.
두 번째 대조에서 stopped 성공을 건강함으로 축약하지 않고 기존 start의 실제
PID/socket/cgroup/Exec identity 확인과 후속 health를 유지했음을 확인한다.
