# M13-004 — 최초 native 파일 준비와 독립 bootstrap trust

기준 `20cecda`. 최초 설치는 기존 HAA의 검역 성공이 아니라 독립적으로 검증한
bootstrap이다. 기존 user installer와 Root system copier/reader/SDK/enrollment를 재사용한다.
Public tag/publish/main merge 및 현재 개발용 설치의 영구 migration은 실행하지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| bounded official acquisition/frozen set·offline SDK | YES | YES, update | YES, local scope | YES, component | first-install 호출부 |
| fixed system copier/journal-bound partial cleanup | YES | YES, update | YES, local | YES, component | initial reservation owner |
| fixed native registration | YES | YES, register | YES, local | YES, component | 배치 전 준비 경로 |
| initial native file preparation | NO | NO | NO | NO | 아래 fixed operation |
| 전체 clean Host bootstrap/ordinary doctor·nominal 검사 | PARTIAL | NO | NO | NO | actual signed payload와 OS setup/외부 initial verification |

## First trust와 좁은 범위

초기 helper를 root로 실행하기 **전에** 관리자가 독립된 신뢰 도구로 protected official
workflow/tag/source/attestation/digest를 확인해야 한다. Helper가 자신의 검증 성공을
선언한 뒤 그 자신에게 root 권한을 주는 순서는 허용하지 않는다. 고정 SDK는 그 다음
root boundary에서 exact source bytes를 다시 확인한다. `sudo helox` 전체 실행이나
user receipt/ALLOW/ambient SUDO_UID를 권한으로 사용하지 않는다.

다음 source 단위는 파일 준비 전용 `haa-release-installer bootstrap-files`다. Root0/GID0,
Linux amd64만 허용하고 env를 지운다. 입력은 관리자가 사전 검증해 배치한 고정 root-private
`/var/lib/heliopause/install-staging/bootstrap`의 native4/metadata3와 고정 public proof뿐이다.
UID/GID는 기존 root-owned fixed native config에서 읽고 exact manifest/version/native unit/
CLI/observer identity와 연결한다. Arbitrary path/URL/UID/service/command 인수는 없다.
이 operation은 service start/config 작성/runtime 설치 또는 건강함의 최종 게시를 하지 않는다.

## 수정 map과 순서

- 기존 wire release의 provenance 검증 본문을 private helper로 공유한다. Update의 기존
  old record/proof/same-runtime/no-downgrade gate는 먼저 그대로 수행한다. Initial SDK 호출을
  위해 fake installed record를 만들지 않는다. Initial 경로는 compiled runtime/image metadata,
  모든 selected subject와 실제 자신의 root-stage executable inode를 확인한다.
- Native config snapshots/manifest→registration 파싱을 enrollment와 private helper로 공유한다.
  Invalid journal이나 환경 UID를 만들지 않는다. Root initializer는 existing registration/
  update journal/history, legacy `/usr/libexec/heliopause`와 OS-package `/releases/`를 인수하지 않는다.
- 작은 고정 `native-bootstrap.json` intent를 root-only/canonical/exact initial record로 남긴다.
  최초 intent 이전에는 native versions/current와 해당 copy reservation이 실제 absent/empty여야
  한다. 재시도는 같은 independently verified manifest/config/record만 허용한다. 이 intent는
  provenance나 managed registration의 대체 authority가 아니다.
- 기존 installer owner와 persistent blank runtime/update locks·runtime EX를 사용한다. Locks를
  먼저 준비해 incomplete 설치의 일반 CLI 실행을 막는다. Same-file locks와 root-protected
  consumers의 부재를 확인한다. Root-private staging executable도 consumer 검사에 포함한다.
- 동일 intent의 interrupted copy는 기존 mandatory verifier/DiscardSystemCopy로만 처리한다.
  새 파일 준비는 기존 Install이고 complete same-candidate 재시도는 기존 ActivateExisting으로
  재검증한다. Current/target이 다른 설치를 가리키면 고치지 않는다. 임의 glob/강제 삭제는 없다.
- 파일 준비 exit0는 HEALTHY/bootstrap 완료가 아니다. 관리자가 exact native configs/unit과
  trusted Docker/gVisor/images를 준비한 뒤 **설치된** helper의 register, 설정된 일반 UID의
  실제 doctor와 nominal quarantine를 확인해야 한다. 공개 채널 wrapper/최초 획득 연결과
  stage의 마지막 사용 후 정리는 이 전체 경로와 별도 acceptance로 계속 추적한다.

## 검증과 두 번째 검토

Canonical Quick/security/docs/vulnerability와 relevant Race를 실행한다. Foreign/legacy/
OS package/changed intent/config/proof/version/current/helper/lock/consumer의 무변경 거부,
동일 initial intent의 interrupted/complete 재시도와 actual SDK의 unsigned 거부를 분리한다.
Disposable root file mechanics의 crypto positive는 mock이며 actual signed bootstrap이 아니다.
완료 test binaries와 payload는 삭제한다. Global runtime/host 설치는 변경하지 않는다.

두 번째 대조에서 initial file preparation을 serve의 old-peer 인증 예외로 넣지 않았다.
Initial provenance verifier의 분리는 trust gate를 줄이지 않으며 update는 예전 설치 chain을
계속 요구한다. OS 설정·service 시작과 Root admin initial verification이 아직 끝나지 않은
파일 준비를 전체 설치 완료로 보고하지 않는다. 공개 API의2026-10-20 read-only 응답은 빈
release 목록이었다. 이 관측만으로 draft/private 상태나 과거 v0.1.0 evidence의 삭제 원인을
추정하지 않는다. 새 signed native9 positive와 실제 systemd/crash/reboot는 계속 미완료다.

### 준비 단계의 구체적인 재사용 경계

`wireRelease.verifyProvenance`는 update의 old-chain gate 뒤에 있던 SDK 본문만 공유한다.
Bootstrap은 같은 offline SDK/모든7 subject와 자체 root-stage executable의 actual hash/inode,
compiled gVisor와 passive metadata를 확인한다. Enrollment의 root config snapshots와
manifest→record 파싱은 private helper로 공유한다. Installed reader의 public0555/0444 계약을
바꾸거나 initial 파일을 InstalledRelease로 위장하지 않는다.

Bootstrap source는 native0500/metadata0600의 exact7개 root-private files이며 public proof는
기존 고정 proof store다. 성공한 파일 준비도 registration을 게시하지 않는다. 설치된 helper의
register가 실제 installed SDK/service/drain을 통과했을 때만 managed record를 게시한다.
Initial intent와 source가 존재하면 같은 exact SDK/record로 확인한 source만 정리하며,
Root-private staging executable도 global consumer 검사에 포함해 실행 중인 bootstrap을
지우지 않는다. Intent는 작은 provenance-bound 준비 이력으로 남고 unknown source는 보존한다.

독립 검증의 대상은 실제 실행할 root-protected helper 사본이어야 한다. 사용자 파일을
검증한 뒤 그 mutable 경로를 sudo로 다시 여는 순서는 허용하지 않는다. Initial source
정리가 중단된 경우 full source manifest가 이미 삭제됐을 수 있으므로, installed SDK를
다시 검증한 뒤 그 retained manifest/bytes에 맞는 source의 remaining subset만 정리한다.
Source subset이나 intent 자체가 provenance authority가 되지 않는다.
