# M13-004 — bounded payload 전달과 root-private stage 검증

기준 parent `334daf8edd29def236aea1bfcab30879d9c4c331`.
[사전 map과 두 번째 검토](./activation-wire-plan-20261020.md)에 따라 기존 strict
manifest, fixed-root SDK와 protected filesystem reader를 재사용했다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| fixed manifest/proofs/native4/metadata2 stream | YES | YES, concrete codec | YES, local | YES, primitive |
| auth/provenance 후 root-private 수신·정리 | YES | YES, concrete function | YES, isolated mechanics | YES, primitive |
| root RPC·journal/service 전환·public caller | NO | NO | NO | NO |
| 실제 새 native4 signed activation | NO | NO | NO | NO |

Production entry는 old kernel peer와 independently checked current view를 요구하고
새 manifest의 downgrade/runtime drift를 거부한 뒤 SDK로 exact7 subjects를 확인한다.
그 후에만 fixed root-private stage로 bytes를 받아 size/SHA와 실제 metadata schema를
검사한다. 임의 path/UID/argv/service field와 root HTTP는 없다. Stage는 ALLOW나
COMMITTED가 아니며 public command 또는 pointer/config 전환을 추가하지 않았다.

원본은 `docs/github-logs/self-update-20261020/activation-wire-*`다.

- source inventory의 같은 입력에서 canonical Quick30.30s/security5.11s/
  vulnerability12.78s/docs0.21s, focused1.69s와 관련 전체 Race4.29s가 exit0이다.
- namespace 첫 실행은 package-relative fixture를 repository cwd에서 열어 실패했다.
  소스 변경 없이 package cwd로 교정한 동일 SHA binary의 후속0.07s는 exit0이다.
  최초 실패도 보존하며 제품 수정 성공으로 기록하지 않는다.
- 격리된 root tmpfs에서 owner0/Nlink1/mode0600/exact bytes, digest/truncation/marker
  오류 뒤 부분 stage 삭제, inode replacement 보존과 cleanup uncertainty를 확인했다.
  Auth/current snapshot과 positive staging crypto는 test-only mock이다. Production
  SDK entry는 unsigned synthetic native4 set을 root 쓰기 전에 실제로 거부했다.
- 두 host installation/state 경로는 전후 ABSENT_CONFIRMED다. 완료 standalone
  binary28,032,761B는 각 실행 직후 삭제했고 namespace payload는 종료로 소멸했다.
  실제 host CLI/helper/config/service 변경은0이다.

두 번째 대조에서 candidate proof/bytes, old quarantine ALLOW와 activation authority를
구분하고 root가 user staging path를 따라가지 않는 것을 확인했다. 다음 작업은 durable
journal/config snapshot/transition lock과 실제 service lifecycle, RPC와 CLI 연결이다.
전체 M13-004는 IN_PROGRESS이며 실제 signed target qualification은 별도로 남는다.
