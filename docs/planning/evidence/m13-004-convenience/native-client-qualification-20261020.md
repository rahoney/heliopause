# M13-004 — sealed set Promotion과 old installer client 검증

Parent `5985999`. [사전 lifecycle map](./activation-lifecycle-plan-20261020.md)의
existing complete-set→fixed root IPC 연결 gap을 구현했다. M13-004는 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| NativeReleasePromotion의 exact sealed7-member/held FD 전달 | YES | YES, concrete adapter | YES, local | YES, component |
| old installed installer 실행/endpoint 검증/lease handoff client | YES | YES, fixed client | PARTIAL, codec/kernel endpoint | NO, actual signed transaction 필요 |
| 실제 session factory/공개 self-update 명령/native bootstrap | NO | NO | NO | NO |

Promotion은 기존 VerifiedBundle/LocalStaging records와 fixed native target, official
manifest-selected native4+metadata2+manifest를 비교한다. Private directory/regular0400/
single-link/owning UID/size/hash를 확인한 FD를 보유해 동일 descriptor만 전달한다.
새 Policy나 재다운로드는 없다. Root SDK/stream rehash는 기존 server에서 독립 유지한다.

Client는 old installed view와 controller를 다시 확인한 뒤 fixed `/usr/bin/sudo -n --
<old signed installer> serve`만 실행한다. PREPARED→actual cleanup→SH 해제→ACTIVATE
순서를 지킨다. Unknown root process를 죽이거나 COMMITTED 뒤 오류를 성공으로 덮지 않는다.
Root endpoint의 parent/UID/GID/mode/SameFile/SO_PEERCRED를 확인하며 sudo monitor PID를
server PID로 오인하지 않는다. sudo 인증은 OS에서 먼저 `sudo -v`로 받을 수 있다.

원본은 `docs/github-logs/self-update-20261020/native-client-*`다.

- Source/staging-only13 cases: exact held bytes와 path replacement 뒤 원래 FD 사용,
  변조/hardlink/symlink/쓰기 권한/크기/record/proof 오류/다른 target/저장소/member 누락 거부.
  Positive activation callback은 mock이며 새 signed target의 성공이 아니다.
- Client codec6 cases: 준비 전 handoff 없음, cleanup 실패·취소 뒤 ACTIVATE 없음,
  잘못된 준비 frame/변조 payload 거부, commit 응답 손실 보존. Actual child wait 결과
  보존/중복 reap 방지와 unmanaged/canceled pre-launch denial도 통과했다.
- 동일 final source의 canonical Quick30.44s/focused2.84s/activation 전체 Race5.31s는 exit0.
  Namespace-only test 추가 이전의 동일 production bytes는 security6.96s/
  vulnerability11.15s/docs0.24s와 Promotion 전체 Race112.870s를 통과했다.
  Test-only 수정이 이전 통합 Race와 겹쳤으므로 그 실행을 whole final-source 결과로
  쓰지 않는다. Final input/overlay notice와 unchanged Promotion source를 함께 참조한다.
- Final socket namespace0.07s/exit0: actual UID1000 child의 root SO_PEERCRED 확인,
  root client의 parent/mode/UID/GID negatives와 기존 live/stopped/unknown retention PASS.
  Actual signed Serve→activation/real systemd의 end-to-end 증거가 아니다.
- Final binary28,838,280B와 이전 completed probe28,838,344B는 각 종료 직후 삭제했다.
  Namespace payload는 종료로 소멸했고 host protected installation/state는 전후
  ABSENT_CONFIRMED다. 실제 설치/config/service 변경은0이다.

남은 gap은 session factory/cleanup/CLI, helper producer/native bootstrap와 실제 새 signed
native4/systemd/crash qualification이다. 기존 개발 설치를 새 native 관리 설치로 자동 인수하지 않는다.
