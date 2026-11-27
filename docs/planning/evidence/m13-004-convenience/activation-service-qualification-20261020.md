# M13-004 — fixed service/config/drain adapter 검증

기준 `3c93e39`; [사전 map/재검토](./activation-service-plan-20261020.md)를 따른다.
기존 trusted executor/strict metadata/registration/journal owners를 재사용했다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| fixed unit/config transition과 exact restore | YES | YES, concrete functions | YES, isolated filesystem | YES, primitive |
| bounded unit facts/root process/socket pin | YES | YES, concrete functions | YES, local+kernel mechanics | YES, 해당 primitive |
| global drain/actual systemd stop/start | YES | YES, concrete adapter | PARTIAL, fault/codec only | NO, real systemd host |
| full root transaction/RPC/health/public caller | NO | NO | NO | NO |

Native managed unit/path/config와 kernel PID/inode/UID/GID/socket peer를 묶는다.
Root-only install executor는 기존 runtime/tool 검증과 격리 환경을 재사용하고 systemctl
한 identity만 추가한다. Query가 실패하거나 budget/property/identity가 불명확하면
confirmed stop/absence로 바꾸지 않는다. 설정 복원은 saved old/exact new bytes만
허용하고 host Docker 설정·file mode/GID를 보존한다. Actual global drainage/systemd
qualification과 newly signed native4 activation은 다음 전체 lifecycle에서 남는다.

원본은 `docs/github-logs/self-update-20261020/activation-service-*`다.

- 첫 production build는 peer PID int32→int의 명시 변환 누락으로 실패했다. 다음
  Staticcheck는 의도적인 nil-context 시험의 SA1012를 보고했다. 취소 context 회귀와
  도구 초기화 전 취소 검사를 사용했고 ignore/export/skip은 추가하지 않았다.
- 이후 성공 candidate를 재검토해 embedded bytes.Buffer의 promoted WriteString/
  ReadFrom이 machine-output Write budget을 우회할 수 있음을 발견했다. 기존 malformed
  query 거부는 memory budget 증거가 아니었다. Test-only overlay의 실제4B budget
  회귀가 unchanged candidate에서1.65s/exit1이고 private Buffer field로 교정한 최종
  입력의 회귀는 PASS다. SDK/실제 host exploit 또는 shipped executor의 결함으로 확대하지 않는다.
- 최종 `budget-fixed-source`는 전체 tracked files와 명시 candidate의 두 tree equality를
  기록한다. Canonical Quick27.87s/security4.78s/vulnerability11.10s/docs0.22s,
  focused1.70s와 관련 전체 Race4.68s는 exit0이다. 앞선 성공은 최종 검증 대신 쓰지 않는다.
- 최종 root mount/network tmpfs namespace0.12s/exit0: actual protected root process/
  pidfd live→reaped, 두 socket의 kernel PID/UID/GID와 foreign PID/GID 거부, fixed
  stop/reload fault의 원래 cause 보존, partial config의 exact restore와 unknown bytes
  보존을 확인했다. Systemd runner와 release crypto는 mock이다. Host proc view를
  공유하는 시험에서 unrelated host services의 부재를 synthetic positive로 주장하지 않았다.
- Completed final binary28,354,936B와 이전 각 binary/overlay operator는 사용 후 삭제했다.
  Namespace payload는 종료로 소멸했고 host installation/state는 전후 ABSENT_CONFIRMED다.
  실제 host CLI/config/service 변경은0이다. Active private source/cache는 바로 다음 단위에 재사용한다.

두 번째 검토에서 resource socket의 실제 parent, observer asset의 실제 underscore 이름,
ordinary executor에 없는 systemctl과 stdout budget의 interface 우회를 대조했다.
Return0/PID 문자열/partial restore를 전체 완료로 인정하지 않는다. 다음 구현의
[source map/재검토](./activation-lifecycle-plan-20261020.md)는 proof persistence,
root transaction/recovery, authenticated helper IPC와 기존 pipeline/CLI handoff를 소유한다.
전체 M13-004는 IN_PROGRESS다.
