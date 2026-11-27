# M13-004 — managed lease와 ordinary-user health channel 검증

기준 `c9814f6`; [이전 source map/두 번째 검토](./activation-lifecycle-plan-20261020.md)를
따른다. Whole self-update는 IN_PROGRESS다. 새 helper RPC/전체 전환은 아직 없다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| 일반 runtime의 shared lease/실제 cleanup 순서 | YES | YES, Bootstrap | YES, local/기존 회귀 | YES, wiring 범위 |
| Doctor/launcher 실제 diagnosis gate | YES | YES | YES, local | YES, wiring 범위 |
| inherited root health issuer/ordinary child 경계 | YES | YES, fixed internal CLI handler | YES, kernel negative/mechanics | PARTIAL, signed native4 positive 없음 |
| 전체 root transaction/recovery/RPC/public self-update | NO | NO | NO | NO |

일반 runtime은 Host tools 이전 SH를 잡고 기존 observer/executor cleanup 뒤 놓는다.
Help/settings는 gate를 열지 않으며 managed Doctor는 실제 signed native root를 진단한다.
Doctor tool cleanup uncertainty도 성공으로 숨기지 않는다. Health child는 root로 실행하지
않고 실제 parent의 root-created FD3, signed old installer FD4와 public old/current views를
검증해야 private borrowed context를 얻는다. Internal flag만으로 arbitrary FD를 닫거나
lease를 우회하지 않는다. Borrowed context는 recursive update owner를 받을 수 없다.

Root health issuer는 own protected inode를 먼저 확인하고 known ordinary child의 실제
UID/GID/exe를 HEALTHY/RELEASE handshake 전후로 확인해 Wait한다. Home/cache는
journal stage에서 유도한 별도 workspace다. Workspace는 independent drainage 뒤에만
owned inode를 삭제하며 unknown state는 recovery를 위해 남긴다. Child return0은 독립
signature/service/resource 검증을 대체하지 않는다.

원본은 `docs/github-logs/self-update-20261020/activation-health-*`다.

- 전체 tracked 입력과 명시 candidate equality 뒤 canonical Quick29.94s/security4.93s/
  vulnerability10.81s/docs0.20s, focused1.85s와 releaseactivation/bootstrap/실제 CLI
  관련 Race14.42s가 exit0이다. 실제 production CLI default/help 및 inherited channel
  없는 internal flag의 quiet exit1 회귀가 포함된다. Flag denial 의미는 Linux 범위다.
- Root mount/network namespace0.17s/exit0: 실제 보호된 root issuer로 실행해 ordinary
  UID1000 child에 실제 FD3/FD4를 전달했다. 불완전 signed old view를 borrowed context
  발행 전에 거부했고 callback 미실행, child EOF/Wait와 workspace idempotent cleanup/
  unknown inode replacement 보존을 확인했다. Issuer/old peer/installed crypto snapshot은
  test-only mock이다. Root가 child socket endpoint를 계속 보유해 EOF를 감추지 않았다.
- Darwin amd64/arm64의 bootstrap와 CLI 네 test binary는 compile-only exit0이다.
  Native macOS runtime/new update 지원으로 확대하지 않는다. 각각 즉시 삭제했다.
- 완료 Linux binary28,550,868B도 즉시 삭제했다. Namespace payload는 종료로 소멸했고
  host installation/state는 전후 ABSENT_CONFIRMED다. 실제 host CLI/helper/config/service
  변경은0이다. Signed native4 positive/actual systemd/다음 real quarantine health는 미실행이다.

두 번째 대조에서 ptrace 권한과 root-private journal을 ordinary child가 읽을 수 있다고
가정하지 않았고 creator credential과 child's current UID를 구분했다. 기존 검사/Policy/
cleanup owner를 재사용하며 다음은 실제 root 전환·복구/RPC와 public caller 연결이다.
