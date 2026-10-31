# M13-004 — 인증 뒤 root stage로 전달하는 고정 bytes protocol

[보호된 설치 계획](./protected-activation-plan-20261020.md)의 다음 source map이다.
검증된 copy/pointer와 installed state/proof reader를 재사용한다. Kernel caller
인증은 별도 구현 단위이며 그 실제 namespace 결과는 전체 설치 성공이 아니다.
Public command/서비스/journal 전환은 이 전달 구현의 성공으로 대체하지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 gap |
| --- | --- | --- | --- | --- | --- |
| exact root copy/pointer와 기존 version 복원 | YES | YES, primitive | YES, local | YES, primitive | 기존 releaseinstall 재사용 |
| root-owned installed proof/subject 읽기 | YES | YES, primitive | YES, local/stored proof | NO, 새 native4 positive | releaseactivation reader 재사용 |
| bounded authenticated update payload 전달 | NO | NO | NO | NO | 고정 역할 stream/codec와 root stage 필요 |
| fixed RPC server와 full activation | NO | NO | NO | NO | auth→codec→journal/service→실제 bootstrap/CLI 연결 필요 |

## 수정 위치·wire contract

1. `releaseactivation/wire_linux.go`는 codec만 소유한다. `HAAUPD1` magic,
   big-endian manifest length/raw JSON, bounded proof count와 length/raw bundle을
   전달한다. Manifest1MiB, proofs최대16개·각1MiB, 전체 selected bytes1GiB와
   기존 단일 native512MiB 제한을 유지한다. 임의 filename/path/UID/service/argv는
   wire field가 아니다. Version/source/runtime/file sizes는 strict manifest에서만 온다.
2. Manifest를 primary로 고정하고 선택은 native4 + SBOM/runtime-images의6개
   payload를 canonical name order로 전달한다. 각 파일 length는 manifest의 exact
   record를 사용한다. Root가 받는 바로 그 bytes의 size/SHA를 검사하며 다른
   source resolve/download fallback은 없다. Manifest 자체는 별도 exact subject다.
3. Root는 signature/subject/source/tag/run/runtime lock과 role set을 먼저 확인한다.
   Current보다 낮은 version 또는 같은 version의 다른 manifest는 거부한다.
   같은 version no-op도 old controller의 fresh quarantine을 생략하는 권한이 아니다.
4. `releaseactivation/stage_linux.go`는 실제 authorized old peer와 checked current
   view가 있을 때만 fixed root-private staging에 쓴다. Root-owned/non-writable
   ancestors를 선행 확인하며 foreign existing state를 chmod/chown해서 인수하지 않는다.
   Root의 소유 inode를 기록해 부분 copy/Close의 실제 cleanup을 확인한다.
5. Strict metadata/bytes/proof 재검증 뒤 `PREPARED`를 전달한다. 이후 old controller의
   실제 runtime cleanup/shared lease 해제와 `ACTIVATE` 확인 전에는 current/config/
   service를 바꾸지 않는다. Header/proof/stream 오류·EOF·취소·cleanup uncertainty는
   primary cause를 보존하고 fail-closed다. Codec 자체는 ALLOW/COMMITTED를 만들지 않는다.

기존 Source/Inspection/Policy/Staging engines와 root copy는 다시 구현하지 않는다.
Sender는 기존 sealed staging의 exact members만 열도록 후속 Promotion adapter에서
연결한다. New helper executable/command는 실제 handler가 준비된 뒤 등록한다.

## 검사·두 번째 검토

- Valid bounded round-trip과 duplicate/missing/unknown role, wrong magic/length/
  proof count/truncation/trailing control, malformed JSON/record mismatch/oversize/
  digest mismatch·취소를 확인한다. Wire가 arbitrary target/UID/command를 갖지 않는지 검토한다.
- Default codec tests는 caller-owned temporary buffers/files만 사용한다. Root stage/
  ownership/partial cleanup은 disposable namespace에서 별도로 검사한다. Crypto mock
  staged fixture와 실제 protected signed target qualification을 구분한다.
- Authenticated peer/old ALLOW, signature/provenance, data schema와 root copy authority를
  서로 대체하지 않는다. PREPARED는 activation 성공이 아니며 candidate 출력/return0도
  authority가 아니다. Whole lifecycle의 service/rollback/crash tests는 다음 owner에 남긴다.
- Canonical format/Quick/security/docs, 관련 Race와 reachable vulnerability 검사를 수행하고
  완료 binary/payload/operator는 마지막 실제 사용 직후 삭제한다.

두 번째 대조에서 generic file RPC/FD path following, root-side HTTP/임의 command,
opaque user receipt로 승인 전달, live metadata 재조회, signature-only 안전 판정을
제외했다. Single frozen manifest/7 exact subjects와 기존 workflow owners를 유지한다.
