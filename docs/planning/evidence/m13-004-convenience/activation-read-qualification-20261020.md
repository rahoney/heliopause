# M13-004 — protected state/proof reader와 runtime lease

기준 parent `496dd905140711c03982f1cd1c93203ef425c99e`.
[사전 책임 감사·수정 map·두 번째 검토](./protected-activation-plan-20261020.md#496dd90-이후의-다음-readlease-source-map)를
따라 Linux `releaseactivation` owner를 추가했다. 전체 M13-004는 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| strict registration·protected file/proof 읽기 | YES | YES, concrete adapter/tests | YES, local/kernel/offline proof 범위 | YES, 이 범위만 |
| runtime shared lock/취소/동일 FD 정리 | YES | YES, concrete API/tests | YES, kernel descriptor mechanics | YES, 이 범위만 |
| 새 signed native4 설치의 전체 read positive | YES, reader 구현 | NO, 새 target/public caller 없음 | NO | NO |
| old caller 인증/RPC·journal/service·public CLI | NO | NO | NO | NO |

Registration의 management/version/manifest digest/UID/GID는 strict scalar fields다.
Duplicate/unknown/case alias/null/invalid value를 거부한다. Record 자체를 provenance나
Policy로 사용하지 않는다. Reader는 fixed current pointer, root-owned/Nlink1/non-writable
files/ancestors, mode/size/hash, strict metadata/compiled runtime과 fixed-root SDK subject를
독립 재확인한다. Network 획득·임의 user path/UID 기반 privileged mutation은 없다.

확정된 marker 부재만 legacy다. Partial/journal/unreadable state는 normal mode로
돌리지 않는다. Lease는 root-owned persistent file의 advisory shared lock을 사용하며
취소·verification 실패에서 같은 FD를 unlock/close한다. Caller authentication과
실제 runtime cleanup/installer exclusive handoff wiring은 아직 별도 gap이다.

원본 owner는 local `docs/github-logs/self-update-20261020/`다.

- `activation-read-final-source.json`/`activation-read-final-inventory.json`은 전체 입력을
  고정한다. Source candidate focused1.64s와 canonical Quick35.12s/security5.18s/
  vulnerability10.82s/docs0.22s, 관련 Race4.74s는 모두 exit0다.
- `activation-read-namespace.json`/log: exact static binary를 새 mount/network namespace
  tmpfs에서 root로 실행해0.07s/exit0. Confirmed absent/partial/journal gate,
  failed-verification lock release/CLOEXEC, 실제 v0.1.0 stored proof의 offline SDK
  positive와 변조·foreign owner/hardlink 거부를 확인했다.
- 진짜 v0.1.0은 installer 역할이 없으므로 root-owned registration을 만들어도
  managed native4로 인정되지 않았다. **새 네 역할의 전체 installed positive나
  actual activation qualification으로 확대하지 않는다.**
- 완료 Linux binary27,752,661B는 즉시 삭제했다. Fixtures는 private tmpfs 종료로
  소멸했고 host `/usr/lib/heliopause`와 `/var/lib/heliopause`는 전후
  ABSENT_CONFIRMED다. 실제 host runtime/config/service 변경은0이다.
- 문서 전달·exact index/commit은 같은 디렉터리의 activation-read-delivery 자료가
  소유한다. Linux-only owner이므로 Darwin support/실행 성공을 주장하지 않는다.

두 번째 검토에서 expected filename/digest와 actual signed subject, root ownership,
kernel advisory lock을 서로 대체하지 않았음을 확인했다. 원래 user installer와
기존 artifact/Application/Policy 소비자는 canonical Quick에서 보존했다. 다음은 old
controller의 actual kernel identity와 per-request session을 검증하고 fixed privileged
RPC/journal/service 전환·복원·CLI를 연결하는 것이다.
