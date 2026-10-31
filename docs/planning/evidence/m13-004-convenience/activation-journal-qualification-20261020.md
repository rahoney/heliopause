# M13-004 — durable journal와 handoff lock 검증

기준 parent `bf3b0d6`; [source 이전 map/재검토](./activation-transaction-plan-20261020.md).
Protected copy/read/peer/stage를 재사용하며 전체 자체 업데이트는 아직 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| fixed config snapshot/PREPARED/상태 전이 원장 | YES | YES, concrete functions | YES, local mechanics | YES, primitive |
| 중복 update/root installer와 EX runtime lock | YES | YES, concrete functions | YES, actual kernel locks | YES, primitive |
| service/health/rollback/recovery/RPC/public CLI | NO | NO | NO | NO |

Journal은 고정3개 config의 bytes/hash/mode/GID, 이전 registration과 새 version/
manifest/stage만 보존한다. Canonical JSON 외 unknown/duplicate/case alias/trailing,
잘못된 단계/경로/hash/권한/downgrade/same-version substitution을 거부한다. 새 root
temp→write/fsync/close→rename→parent sync를 사용하며 기존 identity/bytes가 맞아야
교체한다. 불명확한 기존 파일을 chmod/chown해서 인수하지 않는다. Persistent advisory
locks는 인증이나 ALLOW가 아니며 원래 source와 root boundary를 대체하지 않는다.

원본은 `docs/github-logs/self-update-20261020/activation-journal-*`다.

- 초기 Quick13.57s는 PREPARED 발행 함수의 미사용 U1000을 보고했다. 실제 그 발행
  경계의 positive/readback/closed-stage 거부 회귀를 추가했다. Export/ignore/skip으로
  숨기지 않았다. 최초 namespace 성공은 당시 입력의 제한된 범위로만 보존한다.
- 최종 전체 tracked 입력+명시 candidate의 두 작업 tree bytes를 대조했다. Canonical
  Quick28.54s/security6.41s/vulnerability13.70s/docs0.23s, focused1.82s와 관련
  전체 Race4.81s가 exit0이다.
- 최종 별도 mount/network root namespace0.12s/exit0: actual PREPARED 원장 생성과
  readback, root0/Nlink1/0600, terminal marker admission 차단, 단계 생략·foreign/
  hardlink/symlink·wrong prior bytes 거부, persistent installer/update 충돌과 실제
  SH→EX 대기 취소/해제/CLOEXEC를 확인했다. Positive stage/current crypto는 mock이다.
- Host installation/state 경로는 전후 ABSENT_CONFIRMED이며 실제 config/service 변경은0.
  최종 standalone binary28,115,787B와 앞선 binary는 사용 직후 삭제했고 fixture tmpfs는
  종료로 소멸했다. Active private source/cache는 다음 작업에서 즉시 재사용한다.

두 번째 검토에서 root record/lock/후보 출력이 안전성 authority가 되는 경로가 없고,
terminal journal도 lifecycle 검증/정리 전까지 정상 runtime으로 인정하지 않는 것을
확인했다. Service identity/confirmed stop, 검증된 복원, crash recovery와 실제 새 signed
native4 activation은 이 원장 mechanics의 성공으로 대체하지 않는다.
