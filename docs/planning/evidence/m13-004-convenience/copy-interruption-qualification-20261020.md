# M13-004 — 중단된 system copy의 정확한 정리 검증

Parent `c25e55e`. [사전 감사/map/두 번째 검토](./copy-interruption-plan-20261020.md)를 따랐다.
기존 verifier/copy/pointer를 유지하고 system stage만 manifest SHA의 고정 basename으로
선택한다. PREPARED 이전 부재가 확인된 동일 reservation만 journal이 소유한다.
User installer의 random stage/permissions와 wire/schema는 유지했다.

`DiscardSystemCopy`는 mandatory verifier와 기존 strict bundle/hash/metadata checks 뒤
root/no-link·known membership·bounded exact source prefix·mode·size·inode를 확인한다.
전체 집합 검증 전에 아무 파일도 지우지 않는다. 각 삭제 직전 신원도 재검사한다.
SIGKILL의 zero/partial/완료 파일 및 umask 이후 Chmod 이전 권한 부분집합을 처리한다.
Unknown/foreign/changed 파일은 보존한다. Root는 EX/drain 뒤 journal의 정확한 copy만
먼저 정리하고 intake를 제거한다. PREPARED abort에는 미래 copy를 인수하는 예외가 없다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| system copy reservation/partial cleanup | YES | YES, shared copier/root lifecycle | YES, local mechanics | YES, component |
| 실제 새 signed/systemd/crash/reboot 전환 | PARTIAL | PARTIAL | NO | NO |

원본은 `docs/github-logs/self-update-20261020/copy-interruption-*`다. 동일 전체 입력의
format/focused와 canonical Quick112.44s/security7.98s/vulnerability11.86s/docs0.21s,
releaseinstall/releaseactivation/installer Race7.74s는 exit0이다. Focused1.23s는 기존
user/system copy·rollback/config/journal/terminal recovery 소비자를 확인했다.

Disposable copy namespace24 cases0.12s/exit0은 empty/prefix/complete/umask/installed-source/
반복 cleanup과 unrelated pending 보존, unknown/prefix mismatch/oversize/mode/link/owner/
group/dir/setgid/symlink/verifier/cancel/foreign-root/collision 거부를 확인했다.
실제 child는 첫 metadata copy 이후 test-only Context fault barrier에서 SIGKILL됐고,
WaitStatus SIGKILL·reservation 안 manifest만 존재·current 부재를 확인했다. 이후 제한된
cleanup과 재실행이 통과했다. Mid-write prefix는 별도 실제 file fixture로 검증했다.
Crypto는 mock이며 실제 signed power-loss/systemd 전환의 증거가 아니다.

Root transaction namespace9 cases0.22s/exit0은 PREPARED 이전 collision 거부/abort의
foreign file 보존과 기존 success/same-version/existing-candidate/health/start/unknown
stop/config/drain 오류의 복원·RECOVERY_REQUIRED를 확인했다. 실제 root copy/pointer/
config/record/journal/lock은 사용했으나 crypto/old peer/systemd/health는 mock이다.
완료 binary5,532,301B와28,986,139B를 삭제했다. Host native install/state는 전후
ABSENT_CONFIRMED이고 기존 설치/서비스/config는 변경하지 않았다.

Darwin 두 아키텍처의 releaseinstall/installer test compile4개는 exit0이다. 생성 파일은
각 실행 직후 삭제했으며 native macOS 실행으로 보고하지 않는다. 정확한 bytes/시간은
copy-interruption-darwin.json이 소유한다. 두 번째 검토에서 basename만으로 소유권을
만들지 않으며 source/proof가 불명확하면 journal과 intake를 보존함을 확인했다.
다음은 최초 verified native loader/ordinary-user health와 실제 새 signed base/target,
systemd/crash/reboot qualification이다. M13-004는 IN_PROGRESS다.
