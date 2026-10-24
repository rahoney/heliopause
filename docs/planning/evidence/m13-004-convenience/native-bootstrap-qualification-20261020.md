# M13-004 — 독립 검증한 최초 native 파일 준비·정리 검증

Parent `20cecda`. [사전 감사/map/두 번째 검토](./native-bootstrap-plan-20261020.md)를 따랐다.
실제 `bootstrap-files`는 Root0/GID0/Linux amd64, 고정 root-private bootstrap set와 public
proof, root native config, 자신의 exact stage inode를 요구한다. 독립 검증한 실제
root-protected helper를 실행해야 한다는 first-trust 경계는 별도이며 자기 보고로 대신하지 않는다.

기존 wire의 offline SDK 본문만 공유했다. Update의 old record/proof/runtime/no-downgrade
검사는 그 앞에 그대로 있다. Initial bootstrap은 fake installed record를 사용하지 않는다.
SDK/subject/hash/metadata/compiled gVisor를 확인하고 root config에서 initial identity를
파생한다. Legacy/OS package/기존 managed state는 인수하지 않는다. 같은 작은 intent의
재시도만 허용하고 기존 system copier/partial cleanup을 재사용한다.

`bootstrap-files`는 파일만 준비하고 registration/runtime health를 게시하지 않는다.
실제 성공 출력도 `native-files=PREPARED registration=PENDING health=UNVERIFIED`다.
Blank locks가 일반 runtime 실행을 막는다. 설치된 helper의 register는 SDK/service/drain
확인 후 managed record를 게시하고, installed SDK로 증명한 initial source만 정리한다.
Source 정리가 중단됐어도 retained installed manifest와 일치하는 remaining subset만 지운다.
Unknown/foreign source는 보존하고 작은 intent와 public proofs/설치 bytes는 남긴다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| fixed initial file preparation/intent/재시도 | YES | YES, helper bootstrap-files | YES, local mechanics/negatives | YES, component |
| register 뒤 known source의 제한된 정리 | YES | YES, actual register | YES, local file mechanics; SDK negative | PARTIAL, actual signed positive 없음 |
| public initial acquisition/OS setup/ordinary doctor·nominal | PARTIAL, owners | NO, whole composition | NO | NO |
| 실제 새 signed native9/systemd/crash/reboot | PARTIAL | PARTIAL | NO | NO |

원본은 `docs/github-logs/self-update-20261020/native-bootstrap-*`다. 동일 source의
format/focused와 canonical Quick97.78s/security6.54s/vulnerability12.35s/docs0.20s,
관련 releaseactivation/installer Race6.78s는 모두 exit0이다. Focused1.16s는 shared
config/wire/등록과 inert/cancelled helper consumers를 확인했다.

Bootstrap namespace31 cases0.27s/exit0은 fresh/complete retry/interrupted-copy retry,
잘못된 proof/helper/legacy/OS package/foreign root/version/기존 record/journal/history/
lock/intent/current/consumer/verifier/config/bytes와 cancel 거부를 확인했다. Files-only
성공 뒤 registration 부재/일반 runtime lease의 Transition 거부도 확인했다. Source
cleanup full/subset/empty 및 unknown owner/mode/link/tamper/cancel 보존을 검사했다.
Positive SDK/own-helper/consumer/installed facts는 mock이며 production SDK/자체 executable
검사는 fabricated set를 거부했다. 실제 signed bootstrap이나 service 시작 성공이 아니다.

동일 입력의 기존 enrollment namespace36 cases0.37s/exit0과 actual UID1000 reader도
통과했다. Root temporary staging executable을 consumer 검사에 포함했다. 두 namespace
완료 binary29,109,210B씩을 삭제했고 host native installation/state는 전후
ABSENT_CONFIRMED다. 기존 설치·service/config를 변경하지 않았다.

Actual Go1.26.9 CLI build5.30s와 helper1.04s는 성공했고 helper71개 module tuple이
CLI82개에 모두 포함됐다. Actual default/help/version exit0, 일반 UID의 serve/recover/
register/bootstrap-files4개 exit1과 stable-only diagnostic을 확인했다. Darwin helper
TEST compile은 amd640.37s/arm640.40s/exit0이다. CLI42,316,868B/helper27,274,694B/
Darwin4,352,384B/4,132,114B와 전용 work dir/operator는 삭제했다.

두 번째 대조에서 Root admin의 independent verification 대상이 실제 Root helper
사본이어야 함을 명시했다. Initial 준비는 update old-peer 인증 예외나 SDK 통과만의
검역 ALLOW가 아니다. Public acquisition/OS setup composition·ordinary doctor/nominal,
새 signed base/target·실제 systemd/crash/reboot는 계속 남는다. M13-004는 IN_PROGRESS다.
