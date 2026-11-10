# M13-004 — 사전 검증·배치된 native 설치의 명시적 등록 검증

Parent `bd433a6`. [설계·두 번째 대조](./enrollment-plan-20261020.md)를 따랐다.
`Register`와 helper의 root-only `register`는 고정 current/native set·public proof·root
config에서 identity를 읽고 SDK/own installed inode/service/runtime/drain을 독립 확인한다.
고정 persistent locks를 잡고 전후 신원·설정·잠근 inode·live pidfd를 다시 확인한 뒤
정확한0644 registration만 게시한다. Download/copy/current/service 전환은 하지 않는다.
기존 serve의 old peer 인증에 예외를 넣지 않았다. UID/target/경로 인수와 ambient UID는 없다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| 명시적 initial enrollment/동일 record 재검증 | YES | YES, fixed helper register | YES, local mechanics/negatives | YES, component |
| 최초 verified loader/ordinary doctor·nominal quarantine | PARTIAL, owners | NO, loader | NO | NO |
| 실제 새 signed native9/systemd/reboot transition | PARTIAL | PARTIAL | NO | NO |

원본은 `docs/github-logs/self-update-20261020/enrollment-*`다. 최종 동일 source의
format/focused와 canonical Quick112.20s/security6.61s/vulnerability11.77s/docs0.23s,
관련 runtime/Promotion/acquisition/Bootstrap/helper 전체 Race182.33s는 exit0이다.
별도 focused1.31s는 기존 config/journal/service consumers를 재검사했다.

최종 disposable root namespace36 cases는0.37s/exit0이다. 최초/반복/blank-lock retry와
COMMITTED/ROLLED_BACK 이력의 동일 신원을 허용하고, foreign record/current/config,
미서명·변조 payload, wrong helper, pending/불명 history, private permissions,
nonempty/writable/foreign/link/symlink/missing/replaced/busy locks, service/runtime/drain
failure와 mid-query config 변경을 거부했다. 일반 UID1000의 registration/proof/lock 읽기,
SH 획득과 쓰기 거부도 실제 child에서 확인했다. SDK와 service의 positive는 mock이며
실제 production SDK/own executable은 해당 fabricated fixture를 거부한다.

두 번째 검토에서 기존 등록의 누락 lock 재생성과 pathname metadata만 확인한 교체 lock
승인이 발견됐다. 같은 두 회귀의 수정 전 실제 RED0.37s와 최종 GREEN을 보존했다.
기존 등록에는 lock 생성이 없고, 최종 pathname은 실제 보유 FD와 SameFile이어야 한다.
첫 Quick26.71s/exit1의 SA1012는 새 테스트의 nil Context 전달이었다. 해당 nil 사용 시험을
제거하고 취소 Context 검사를 유지했다. suppression이나 product guard 완화는 없다.

Darwin installer test compile은 amd640.80s/arm640.77s/exit0이며 native 실행이 아니다.
완료한 namespace test binaries28,920,457B/28,925,085B/28,930,801B와 Darwin4,352,384B/
4,132,114B는 삭제했다. Host의 native install/state 경로는 모든 namespace 전후 실제
ABSENT_CONFIRMED이며 기존 설치/서비스/설정은 변경하지 않았다. 초기 준비 source/cache는
같은 item의 후속 검증에 재사용한다. 다음은 cold terminal recovery와 verified loader,
새 signed base/target·실제 systemd/crash/reboot qualification이다. M13-004는 IN_PROGRESS다.
