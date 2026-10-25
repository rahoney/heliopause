# P3-B2a — 격리 조회용 Bun native candidate 출력

같은 M13-004/P3 범위다. P2 remote9410의 cu130/cu132 qualification과 독립적인 local 작업이며
실행 중 remote를 push로 바꾸지 않는다. Read는 P3 baseline, runtime maintenance§1–3,
D-012, Architecture-004/007, 확장 계획§3–5와 ci L06/evidence L11·L16/sandbox L08·L12·L18이다.

Baseline commit 2e85c928756b8522b3d1bc865eb60244043371a1

## 착수 baseline audit

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| signed Bun ZIP/native data — releaseprovenance/bun_release.go·bun_tool.go / updater bun_release.go | YES | YES(offline verify) | YES(named local B1 commit2e85c92) | NO(whole P3) | executable candidate 출력 및 격리 native query |
| 안전한 exclusive native 파일 출력 — update-runtime/pnpm_materialize.go publishPNPMAsset | YES | YES(PNPM) | YES(PNPM) | NO(Bun) | 같은 no-replace·opened parent·readback·ownership 검사를 Bun 실제 소비자에 연결 |
| Bun 후보 native 출력 command | NO | NO | NO | NO | 아래 exact 지도와 실제 출력/재출력·오류 불변 검증 |
| Bun production pin/release asset/installer/guest role/runtime version | NO | NO | NO | NO | 이후 B2b의 실제 격리 버전/ABI 근거와 단일 pin·제품 경로; 이번 출력으로 지원 완료하지 않음 |

## exact 수정 지도

- scripts/update-runtime/bun_release.go: 기존 strict discovery/signed checksum/ZIP+ELF 읽기를
  readBunSupply(input, now) 하나로 추출하고 verify command가 같은 결과를 출력한다. Native
  slice는 개발 도구 내부에서만 다루며 caller가 signer/예상 native hash를 덮어쓰지 않는다.
- scripts/update-runtime/native_asset.go: 기존 publishPNPMAsset/pnpmOutputDirectoryOwned의
  검증과 fsync/atomic no-replace를 기계적으로 이동·이름만 공용화한다. 기존 PNPM caller와
  테스트를 새 이름으로 연결한다. PNPM body/mode/parent/link/동일성 조건은 완화하지 않는다.
  실제 두 소비자는 기존 PNPM 생산 asset와 새 Bun candidate다. 범용 extraction은 없다.
- scripts/update-runtime/bun_materialize.go·main.go: 실제 materialize-bun-candidate developer
  command. Explicit absolute canonical source/input/output, fixed bun-candidate filename, source
  밖 output만 허용한다. Signature+complete ZIP+native ELF 검증 뒤 기존 publisher로0555 파일
  하나만 쓴다. JSON은 candidate/native data identity이며 native version/ABI/Policy는 미검증이다.
- bun_materialize_test.go: invalid paths/source output/unknown args/missing or changed proof에서
  출력0, 기존 shared publisher unsafe parents/no-replace 회귀 유지. 실제 official candidate는
  별도 작은 realcorpus test에서 digest/size/mode/single-link/owner와 재출력 거부·foreign file
  보존을 확인한다. Host에서 Bun을 실행하지 않는다.

## 두 번째 검토

버전 문자열·readelf의 BUN_1.2 symbol은 실행 버전 증거가 아니다. Candidate 출력 성공이
production pin/installed role 또는 HAA ALLOW가 되지 않는다. Production current/runtime·
observer에는 아직 Bun 예외를 넣지 않는다. Output parent에 private/owner/no-write 검사를
재사용하고 source 내부·existing/symlink/foreign member를 덮어쓰지 않는다. 검증 실패 뒤
성공 JSON을 만들지 않는다. 기존 PNPM 및 verify-bun-release 소비자를 함께 검사하며 작은
중간 commit은 로컬로 보존하고 해당 묶음 경계 전에 원격 CI를 반복하지 않는다.

## 격리 진단 경계 추가 검토

Candidate actual 뒤 production pin 이전의 native 진단은 기존 PNPM B1과 같은 runc 격리를
사용한다. Runtime lock의 실제 Node24.21.0 slim@digest를 직접 읽어 image identity를 확인하고
network none/read-only/cap-drop ALL/no-new-privileges/nonroot1000·CPU1/memory512MiB/pids64로
실행한다. Candidate만 read-only bind하고 Host source·credentials·Docker socket은 넘기지 않는다.
먼저 --version과 --revision만 실행하며 결과를 source/build attestation 또는 gVisor admission으로
표시하지 않는다. Command/complete stdout/stderr/container exit/ID·이미지와 후보 bytes 전후를
기록하고 종료 뒤 task container를 exact ID로 정리한다. 공용 image를 prune하지 않는다.

## 결과와 다음 범위

Quick57.292s/security8.947s/docs0.238s·관련 네 package full Race33.867s, official candidate
actual6.686s 및 기존 verify command의5case가 PASS다. Shared publisher는 함수명/오류 prefix/
private temporary prefix와 outer whitespace를 제외한 조건·순서가 동일하다. 실제 runc5.182s는
Bun1.4.4/1.4.4+38afee6c8를 반환했고 두 container 종료·삭제/정상 목록 query의 부재와 후보
bytes/mode/inode 보존을 확인했다. 첫 version은 성공했지만 cleanup assertion이 Docker CLI의
absence 오류 문구를 잘못 가정해 중단됐다. 실제 rm 뒤 정상 exact-ID 목록 조회로 부재를
확인하고 v2 operator로 version/revision을 검증했다. 첫 source/stdout은 보존한다.
[local receipt](./p3b2a-bun-candidate-local-20261025.json)가 정확한 범위를 소유한다. 다음 B2b는
단일 production pin/notices/maintenance/release 연결이며 gVisor admission·whole P3는 아직 없다.
