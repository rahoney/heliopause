# M13-004 — previous signed view와 proof 보존 검증

기준 `5f69189`; [이전 source map/재검토](./activation-lifecycle-plan-20261020.md)의
첫 gap을 구현했다. 기존 installed reader의 검증 본문을 공유하고 active pointer의
전후 확인은 public reader에 유지했다. Root SDK/strict metadata/mandatory verifier를
대체하지 않았다. 전체 M13-004는 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| active/previous signed member 검증 본문 공유 | YES | YES, reader | YES, local/negative | YES, primitive |
| root public-read proof 보존/정확한 registration 전환 | YES | YES, concrete functions | YES, isolated mechanics | YES, primitive |
| root transaction/recovery/RPC/health/public caller | NO | NO | NO | NO |
| actual new signed native4 activation | NO | NO | NO | NO |

Proof는 SDK가 exact7 subjects를 검증한 뒤 fixed digest directory에 보존한다.
Filesystem storage 자체는 provenance/Policy authority가 아니다. Root installer에
제공하는 mandatory BundleVerifier도 같은 SDK를 쓰며 signature-only 자기 승인이나
user receipt를 받지 않는다. Previous-version 검증은 current pointer를 먼저 바꾸지 않는다.
Registration 복원은 exact old/new bytes만 허용하고 unknown bytes는 남긴다.

Atomic writer의 최대 body는 기존 proof1MiB를 수용한다. Journal128KiB/config16KiB는
각 기존 schema caller의 더 좁은 제한을 유지하며 generic file RPC는 없다.

원본은 `docs/github-logs/self-update-20261020/activation-proof-*`다.

- 같은 전체 tracked 입력과 명시 candidate에서 canonical Quick28.25s/security5.13s/
  vulnerability12.82s/docs0.22s, focused1.83s와 관련 전체 Race4.78s가 exit0이다.
- Proof namespace0.07s/exit0: production SDK는 unsigned native4를 쓰기 전에 거부했다.
  실제 과거 v0.1 manifest proof를 root store에 보존해 offline SDK로 다시 검증했고,
  public0444/root0/Nlink1/idempotence/1MiB bound/foreign directory 및 exact registration
  전환·복원/unknown 보존을 확인했다. Storage max-body positive는 crypto mock 데이터다.
- 같은 source/binary의 existing reader namespace0.07s와 journal namespace0.12s가
  exit0이다. 기존 absent/partial/journal gate, failed verification lock release/CLOEXEC,
  실제 old signature와 modified/foreign/hardlinked proof 거부, actual PREPARED/lock
  회귀를 유지했다. 새 native4 signed positive나 실제 root activation은 아니다.
- 각 완료 binary28,389,201B는 사용 직후 삭제했고 namespace payload는 종료로 소멸했다.
  Host installation/state는 전후 ABSENT_CONFIRMED이며 실제 config/service 변경은0이다.

두 번째 검토에서 original active pointer 보호, root-only storage와 actual signed bytes,
advisory lease와 Policy, public proof 읽기와 root-private journal의 권한을 구분했다.
다음은 위 source map의 root transaction/recovery와 실제 authenticated IPC/health 연결이다.
