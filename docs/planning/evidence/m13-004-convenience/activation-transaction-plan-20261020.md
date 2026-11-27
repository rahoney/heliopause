# M13-004 — durable 전환 원장·잠금 source map

기준 parent `bf3b0d6`; [상위 승인 설계](./protected-activation-plan-20261020.md)를
유지한다. 기존 protected copy/rollback, installed reader, peer와 wire/stage는 재사용한다.
단일 item M13-004에서 이제 실제 journal/동시 실행 gap만 추가한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 남은 작업 |
| --- | --- | --- | --- | --- | --- |
| protected copy·read·kernel peer·bounded stage | YES | YES, 각 primitive | YES, local | YES, 해당 primitive | 기존 함수 재사용 |
| durable root journal/config snapshot | NO | NO | NO | NO | 고정 config3개와 registration의 사전 bytes/identity 보존 |
| ordinary update owner/root transaction/exclusive lease | PARTIAL, shared만 YES | NO, whole lifecycle | NO | NO | 같은 inode lock·bounded handoff·crash gate |
| service/health/recovery/RPC/CLI | NO | NO | NO | NO | 다음 실제 lifecycle 연결; 이 단위 성공으로 완료하지 않음 |

## 구현 위치와 방법

- `releaseactivation/transaction_linux.go`: `update.lock`는 등록된 일반 UID가 읽고
  nonblocking exclusive lock으로 resolution 이전 중복을 거부한다. Root `installer.lock`는
  root만 만들며 durable root state의 신뢰를 먼저 확인한다. 둘 다 persistent inode,
  CLOEXEC, SameFile, idempotent Close이며 advisory lock을 인증 authority로 쓰지 않는다.
  Exclusive runtime lease는 기존 persistent runtime.lock만 열고 bounded context로 기다린다.
- `releaseactivation/journal_linux.go`: root-only canonical JSON schema의 durable journal.
  Caller로부터 path/argv/UID/service를 받지 않는다. Saved old/new native version과
  manifest digest, staging basename, original registration 및 fixed `/etc/heliopause/`
  `network-policy.json`, `host-tools.json`, `/etc/systemd/system/haa-network-policy-helper.service`
  세 파일의 bounded bytes/mode/GID/SHA를 보존한다. 실제 등록 설치는 이 고정 unit과
  config를 명시적으로 소유해야 한다. 기존 OS package의 `helox-network-policy.service`나
  이 host의 unmanaged setup을 자동 인수하지 않는다.
- Journal 상태는 PREPARED→SWITCHING→VERIFYING→COMMITTED 또는 안전한
  ROLLING_BACK→ROLLED_BACK, 불확실성은 RECOVERY_REQUIRED다. 상태마다 old/new/stage/
  snapshot identity를 유지한다. Marker가 있으면 일반 runtime admission을 막는 기존
  gate를 재사용한다. COMMITTED/ROLLED_BACK은 나중 lifecycle의 독립 검증 후에만 기록하며
  durable history로 옮기기 전까지 marker가 남는다. 불완전/손상/unknown 기록은 정상 부재가 아니다.
- Protected atomic writer는 fixed root-owned ancestors와 existing file identity를 확인하고
  새 root-owned0600 temp의 실제 mode/chown/write/fsync/close/rename/parent-fsync를 순서대로
  수행한다. 성공 전 오류 cause를 보존하고 rename 뒤 sync 실패도 성공으로 바꾸지 않는다.
  Root record/config를 user claim으로 교체하는 API를 공개하지 않는다.
- Saved config bytes는 현재 fixed old version path/UID/GID/socket 및 runtime 설정의
  검토 가능한 snapshot이다. 실제 config 해석/변환과 service ownership/stop/start·
  rollback은 다음 lifecycle owner가 이 데이터를 검증한 뒤 수행한다. Journal 자체로
  임의 파일 복원이나 command 실행을 허용하지 않는다.

## 검증과 두 번째 대조

Canonical format/Quick/security/docs/vulnerability와 관련 Race를 사용한다. 기본 회귀는
strict canonical JSON의 duplicate/unknown/case alias/trailing/size/hash/path/state 오류,
허용된 전이와 terminal 거부를 검사한다. Root namespace에서는 실제 lock exclusion/
release/CLOEXEC·취소·wrong inode와 durable root owner/mode/snapshot/record 교체 및 foreign
file 거부를 확인하고 완료 binary와 tmpfs를 삭제한다. 실제 service/crash/power-loss와 새
signed native4 activation은 이 단위의 성공으로 보고하지 않는다.

두 번째 대조에서 SH를 쥔 controller가 EX를 기다리는 교착, lock 존재만으로 root 인증,
unknown file 덮어쓰기, mutable user snapshot/임의 복원 경로, COMMITTED 출력만으로
완료 판정하는 경우를 제외했다. 다음 실제 service/health/RPC 연결이 필요하다.
