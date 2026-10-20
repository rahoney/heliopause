# M13-004 — system copy의 중단 잔여물과 journal 소유권 연결

기준 `c25e55e`. 기존 `releaseinstall.activate`는 root copy에서도 anonymous
`.pending-*`를 만들고 normal defer로만 제거한다. SIGKILL 뒤 journal은 prepared
intake만 가리키므로 copy stage를 특정할 수 없다. 기존 copy/mandatory verifier/atomic
pointer를 유지하고 이 acceptance gap만 보완한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 copy/pointer·verifier·normal cleanup | YES | YES | YES, local | YES, 이전 component | 재사용 |
| journal-bound copy stage와 interruption cleanup | NO | NO | NO | NO | 아래 변경 |
| 실제 signed/systemd/crash/reboot | PARTIAL | PARTIAL | NO | NO | 별도 actual evidence |

## 수정 map과 순서

- `releaseinstall/installer.go`의 system mode만 manifest SHA에서 파생한 고정 pending
  basename을 exclusive mkdir한다. User mode의 random stage·permissions·API는 유지한다.
  `SystemCopyStageName`은 validated SHA의 pure basename owner이며 arbitrary path를 받지 않는다.
  공유 copy/installation record serialization을 재사용하고 별도 copy engine을 만들지 않는다.
- Root의 `newPreparedJournal`은 해당 basename이 실제 absent일 때만 PREPARED를 만들고
  이 durable manifest identity가 copy reservation을 소유한다. Pre-existing/foreign/unknown
  경로를 이번 journal의 것으로 인수하지 않는다. 기존 update wire/schema에 path나 UID를
  추가하지 않는다. Old/new proof와 root owner/EX/drain 순서는 그대로다.
- System-only `Installer.DiscardSystemCopy`는 기존 mandatory verifier/strict manifest/
  metadata/hash/platform checks를 거친 prepared 또는 installed bundle에서 expected
  contents를 얻는다. 정확한 reserved directory의 root ownership/mode/no-link·known
  membership와 각 파일의 verified source prefix/size/mode/inode를 먼저 전부 검증한다.
  SIGKILL의 partial writes는 검증된 byte prefix만 허용한다. Unknown/foreign/changed
  contents는 하나도 지우지 않는다. 검증 뒤 각 inode를 다시 대조해 remove하고 directory와
  parent를 sync한다. Directory pattern glob이나 arbitrary RemoveAll을 사용하지 않는다.
- Root `cleanupOwnedStage`는 journal identity와 앞선 service/resource drain 아래에서
  reserved copy를 먼저 처리한다. 후보가 완전하면 independently verified retained candidate를,
  그렇지 않으면 원래 root-private prepared bundle을 verifier 입력으로 재사용한다. Known
  copy가 남았는데 source/proof가 불명확하면 marker를 남기고 prepared stage도 보존한다.

## 검증과 두 번째 대조

Canonical Quick/security/docs/vulnerability와 두 owner/helper의 관련 Race를 실행한다.
Copy 실제 interrupted partial/empty/complete stage의 제한된 정리, repeated cleanup,
foreign/link/mode/unknown member/wrong prefix/wrong verifier/cancel의 무변경 거부,
pre-existing reservation 거부와 기존 same-version/rollback/cleanup fault를 확인한다.
가능한 actual SIGKILL 시험도 crypto mock/root file mechanics로 범위를 한정한다.
완료 test payload/binary는 삭제한다. 새로운 signed native9를 만들어낸 것으로 보고하지 않는다.

두 번째 대조에서 path 예측 가능성을 권한이나 provenance로 사용하지 않음을 확인한다.
Root-protected parent·PREPARED 이전 부재·durable journal·mandatory SDK·실제 exact bytes/
inodes가 함께 필요하다. Shared copy를 대체하지 않고 system stage 선택/known cleanup만
추가한다. `.haa-update-*` 같은 다른 파일을 이름 패턴으로 정리하지 않는다. Initial
verified loader와 actual signed/systemd/crash/reboot qualification은 계속 미완료다.
