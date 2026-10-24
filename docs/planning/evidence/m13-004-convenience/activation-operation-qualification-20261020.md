# M13-004 — 실제 root transaction/helper source의 로컬 검증

기준 `6883959782d61d9290074ff48378ac682542006c`.
[사전 map/재검토](./activation-lifecycle-plan-20261020.md)에 따라 기존 owners를 연결했다.
Source 구현과 실제 signed native4/systemd activation을 구분하며 M13-004는 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| root 전환/복원/terminal history와 marker | YES | YES, concrete transaction | YES, local fault mechanics | PARTIAL, 실제 systemd/signature positive 없음 |
| fixed root SDK/kernel peer/stage IPC server | YES | YES, helper serve | YES, socket/negative와 compile | NO, actual signed request 필요 |
| root durable recovery entry | YES | YES, helper recover | PARTIAL, 구성 검토/negative | NO, 실제 signed crash/reboot 필요 |
| public self-update session/CLI/asset producer | NO | NO | NO | NO |

SWITCHING 전에 EX·old identity/config/drain을 확인하고 confirmed stop 뒤 derived
config/mandatory-verifier installer/registration/service/ordinary health를 연결한다.
Same-version와 이미 존재하는 완전한 candidate는 기존 ActivateExisting의 재검증을
재사용하며 fresh quarantine를 생략하지 않는다. 실패는 signed old 복원 또는 durable
RECOVERY_REQUIRED다. Child self-report는 독립 drainage/identity를 대신하지 않는다.
COMMITTED 이후 전달/정리 오류로 건강한 설치를 자동 rollback하지 않는다. Terminal
marker는 확인된 상태를 작은 protected history로 보존한 뒤 해제한다.

별도 cmd/haa-release-installer는 실제 serve/recover만 받고 default/help/version은
inert다. Whole CLI sudo, generic root argv/UID/target과 network helper 설치 API는 없다.
Root serve/recover process는 caller env를 제거한다. Root socket은 protected reserved
directory/type/owner/mode/group와 actual ECONNREFUSED를 확인할 때만 stopped node를
정리하며 live/unknown/replaced inode는 남긴다.

원본은 `docs/github-logs/self-update-20261020/activation-operation-*`다.

- Fault candidate의 전체 tracked+explicit input equality 뒤 canonical Quick26.54s/
  security4.63s/vulnerability10.47s/docs0.22s, focused1.79s와 관련 releaseactivation/
  bootstrap/CLI/installer entry Race7.59s가 exit0이다. 이전 source의 첫 검사도 별도 보존했다.
- Transaction namespace0.17s/exit0: actual root copy/pointer/config/registration/journal/
  locks/owned cleanup을8 cases로 확인했다. Success/same-version/existing-candidate,
  health/start 실패 뒤 old 복원, unknown stop/config와 child self-report 이후 independent
  drain 실패의 RECOVERY_REQUIRED를 확인했다. Crypto/old peer/systemd/health는 mock이다.
  동일 fabricated installation의 실제 production SDK는 실제 저장한 과거 proof로도 거부했다.
- Socket namespace0.07s/exit0: live endpoint 거부, strict stopped node reclaim과
  unknown replacement 보존을 실제 kernel Unix socket으로 확인했다.
- 각 completed binary28,754,391B는 즉시 삭제했다. Namespace payload는 종료로 소멸했고
  host installation/state는 전후 ABSENT_CONFIRMED다. 실제 설치/runtime/config/service 변경은0이다.

두 번째 actual payload 검토에서 [native entry gap](./native-entry-plan-20261020.md)을
확인했다. 기존 observer/network helper의 무인자 exit2/1은 unchanged M6 dynamic check를
INCOMPLETE로 만든다. 이를 새 실제 set의 성공으로 숨기지 않고 검역 우회 없는 명시적
entry/consumer 수정을 다음으로 수행한다. 이후 public caller/producer/native bootstrap와
actual signed/systemd/crash qualification이 남는다. Mock matrix는 그 증거를 대체하지 않는다.
