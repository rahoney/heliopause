# P3 Bun — baseline audit와 공급 검증 A 수정 지도

Baseline source9410c2d/treee4ea86c. Active item은 M13-004 하나다. P2의 같은 source ordinary408/
cu126409는 원격 검증했고 cu130410/후속cu132는 필수 qualification 대기다. 그 결과와 독립적인
이 supply-only 작업은 로컬에서 계속하며 원격 source를 새 push로 바꾸지 않는다. 전체 P2/P3
acceptance나 공개 지원·runtime 채택을 미리 닫지 않는다.

Read: PROJECT-DECISIONS, docs router, Queue, 확장 계획§2–5, client 설계§1/2/4/5,
Architecture-004/adapters, D-012 trusted tooling, runtime maintenance§1–4,
artifact L19/evidence L11·L16/ci L06·L17와 기존 P1/P2 supply verifier·bounded reader.

## 독립 baseline audit

| 책임 / 실제 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun 공급 publisher 검증 — releaseprovenance에 Bun 파일 없음 | NO | NO | NO | NO | 공식 key/clearsigned checksums 검증·exact archive bytes·actual/negative |
| 고정 native 공급의 bounded file reader — scripts/update-runtime/supply_file.go | YES | YES(UV/PNPM) | YES(기존 named scope) | NO(Bun) | 새 실제 Bun 소비자 연결 |
| 고정 OpenPGP library — go.mod ProtonMail/go-crypto1.5.2, verification/terraformprovider/signature.go | YES | YES(provider) | YES(provider) | NO(Bun) | Bun의 별도 publisher와 cleartext signature 형식. Provider의 signer policy는 불변 |
| generic exact npm source/acquisition/static/Evidence/Policy/transaction·PNPM full-context 경험 | YES | YES(기존 clients) | YES(각 scope) | NO(Bun) | Bun lock/layout/runtime/CLI의 실제 typed adapter와 전체 검역 |
| Bun lock/runtime pin·generated identity·bundle/provision/updater·CLI/observer profile | NO | NO | NO | NO | 아래 후속 단계; 이번 A에 placeholder를 만들지 않음 |

Scoped source/파일명 검색에서 기존 Bun 구현은 없고 PNPM의 forbidden runtime fixture만 있다.
IMPLEMENTED인 engine/verifier/provider를 대체하지 않는다. A는 existing releaseprovenance package와
update-runtime command에 실제 공급 소비자만 추가하며 Core/Policy·provider trust를 바꾸지 않는다.

## 공식 후보와 아직 미입증인 경계

2026-10-25 공식 [release](https://github.com/oven-sh/bun/releases/tag/bun-v1.4.4)는1.4.4,
source38afee6c84fcf48c0edb2466aa4b27d287a59c30,2026-10-22 publication이다. 이전1.4.3은
과거 관측이다. Linux x64 zip51,794,595B/API SHA256bf8ccbfb2b0b80159e0f9a6d15c55317add8077534dbc6130a3edb52fccb395c와
SHASUMS256.txt/SHASUMS256.txt.asc assets가 존재한다. Release immutable=false다. 이 관측만으로
서명·ABI·runtime 안전성을 증명하지 않는다. Exact API/tag/workflow/Dockerfile은 raw p3-prep-bun-*에
보존한다. Native binary를 Host에서 실행하지 않는다.

Exact-tag [Dockerfile](https://github.com/oven-sh/bun/blob/38afee6c84fcf48c0edb2466aa4b27d287a59c30/dockerhub/debian/Dockerfile)는
공식 public key를 내장하고 full primary fingerprint F3DCC08A8572C0749B3E18888EAB4D40A7B22B59로
clearsigned checksum을 검증한다. Release workflow는 GPG sign/upload 경로다. Bun 전용 signer를
추가하되 HashiCorp/partner·HAA/UV/PNPM의 기존 정책은 그대로다. Publisher-signed filename/hash는
SLSA build provenance나 version/source commit에 대한 서명이 아니다. CLI receipt에서 그 차이를
명시하고 실제 runtime version/ABI는 후속 native qualification에서 별도로 입증한다.

공식 [설치 문서](https://bun.com/docs/installation)는 glibc2.17+/x64 SSE4.2와 runtime CPU dispatch를
설명한다. Baseline zip 이름만으로 다른 ABI라고 단정하지 않고 실제 ELF를 후속 측정한다.
[Lock](https://bun.com/docs/pm/lockfile)은 text bun.lock/JSONC, 자동 타 package-manager lock migration,
lock-only의 cache 쓰기를 설명한다. Exact source는 install Rust lockfile.rs/bun.lock.rs이며 과거
Zig 경로를 가정하지 않는다. [Lifecycle](https://bun.com/docs/pm/lifecycle)의 내장 trusted list나
trustedDependencies는 HAA ALLOW가 아니다. [Install](https://bun.com/docs/pm/cli/install)의 offline은
missing optional을 건너뛸 수 있고 default backend는 hardlink다. Whole selected-set coverage와
safe-copy/internal-link 검사는 필요하며 native flag 하나로 전체 성공을 대신하지 않는다.

## A의 exact edit map

1. internal/verification/releaseprovenance/bun_release.go와 keys/bun.asc: 기존 pinned OpenPGP
   library의 clearsign decoder/v2 signature verifier를 사용한다. Bound·single exact armor block·
   full pinned primary fingerprint·current/historical key binding·signature type/hash/time·exact
   unique safe checksum filename/hash를 검사한다. Caller가 key/fingerprint를 선택할 수 없다.
   별도 immutable Bun checksum receipt는 authenticated archive digest만 소유한다. Version/Source
   commit을 cryptographically verified라고 반환하지 않는다. Archive complete bytes도 측정한다.
2. bun_release_test.go/testdata/bun-1.4.4: 공개 실제 signed checksum과 pinned key로 positive,
   tamper/wrong subject/unknown signer/trailing or duplicate armor/weak/future signature·취소·
   누락/과대·잘못된 archive를 거부한다. 기본 검사는 작은 fixture로 재현하고52MB zip은 커밋하지 않는다.
3. scripts/update-runtime/bun_release.go와 main.go의 실제 verify-bun-release dispatch: 기존
   readFrozenSupplyFile로 explicit absolute input directory만 읽는다. Frozen identity/checksums/zip
   bytes를 대조해 publisher signature receipt를 출력한다. Expected release version/source는 observed
   metadata이며 native_version_verified=false/source_build_provenance_verified=false를 명시한다.
   Network·Host 실행·추출·pin 변경·승격 권한은 없다. CLI 실제 positive/byte tamper/상대 경로/누락을 확인한다.
4. Canonical format/Quick/security/docs와 관련 verifier/updater full Race, 작은 실제 offline 공급 명령을
   같은 후보에 연결한다. A의 로컬 commit만으로 full CI push를 반복하지 않는다. 다음 native/runtime
   admission과 end-to-end 묶음의 의미 있는 경계에서만 원격 검증한다.

## 두 번째 검토

공급 서명은 malware/원격 execution/현재 Policy proof가 아니다. API digest나 같은 다운로드에서
계산한 expected hash로 자기 확인하지 않는다. Pinned vendor key와 실제 signed text의 digest가
실제 zip complete bytes와 같아야 한다. Cleartext signature의 canonical signed bytes와 checksum
plaintext는 library가 구분하며 직접 whitespace를 고쳐 서명을 우회하지 않는다. Unknown/trailing
blocks·중복 filename·미래/만료/폐기 key·weak hash를 거부한다. 원래 provider signer code를 Bun
편의로 완화하지 않는다. SLSA가 없다는 관측은 아직 별도 확인 사항이며 GPG proof를 SLSA로
표시하지 않는다. 버전 문자열이나 source commit이 checksum에 서명되지 않았음을 receipt와
지원 상태에 남긴다. 현재 A acceptance gap만 구현하며 pin/layout/CLI/execution placeholder는 없다.

## 후속 순서

A signed supply → B ZIP/ELF·ABI/기존 runtime maintenance/provision·정확한 guest query admission →
C native text lock/public metadata independent binding → D full-context isolated/hoisted layout·all
archive/peer/lifecycle/Evidence·current Policy → E guarded cache/records/transaction/native ownership·
CLI → F final end-to-end/affected profiles/support. Source·관찰·반입 역할을 섞지 않으며 Node/Bun
실행 identity와 bunx(P6)를 구분한다. Exact native formats/ABI는 측정 전 UNKNOWN이다.

## A 최초 architecture gate와 exact dependency review amendment

첫 format1.542s 뒤 Quick45.639s는 production/test build를 통과하고 architecture에서 새
releaseprovenance의 OpenPGP3 imports와 updater의 semver direct import를 거부했다.
Default/crypto actual에는 아직 도달하지 않았다. First source/quick를 raw에 보존했다.

기존 module ProtonMail/go-crypto1.5.2와 go.sum은 변경하지 않는다. Provider가 이미 쓰는
packet/v2와 이번 Bun cleartext 형식에 필요한 clearsign만 releaseprovenance의 exact external
allowlist에 검토해 등록한다. 표준 library에는 OpenPGP가 없으며 자체 암호 구현은 하지 않는다.
BSD license/고정 module·기존 provider 사용은 재사용하고 새 cleartext parser의64KiB input,
단일 armor/packet·강한 hash/current+historical key validation을 별도로 검토했다. No network·
외부 key lookup·caller-provided trust는 없다. New reachable-code vulnerability profile을 실행한다.

Updater의 추가 semver import는 제거하고 기존 stable versionPattern과 bounded leading-zero
검사만 사용한다. 이미 잠긴 library라는 이유로 새로운 caller import를 자동 허용하지 않는다.
Core/Application/Policy/CLI와 다른 caller는 같은 external imports를 계속 거부하며 prefix/wildcard
허용은 추가하지 않는다. 이 실제 exact caller 검토는 layer 방향·publisher policy 변경이 아니다.
Provider signer 구현과 HAA/UV/PNPM 검증은 수정하지 않는다. 이 amendment와 scope negative를
추가한 뒤 다시 canonical 검사하며 최초 gate를 우회·suppression하지 않는다.

## A named local 결과와 다음 B

[A local receipt](./p3a-bun-supply-local-20261025.json)는 최종 source·format/Quick165.936s,
security11.613s/docs0.244s/네 관련 package full Race36.007s 및 canonical vulnerability15.497s를
연결한다. 실제 offline 명령의 공식52MB zip 성공과 zip·signed text 변조/상대 경로/누락 거부도
2.562s에 모두 확인했다. Receipt의 native_version_verified/source_build_provenance_verified는
false이며 signer/archive proof를 버전·source build·현재 Policy로 확대하지 않는다.

최초 caller architecture 실패를 원본 그대로 유지했다. V1 checker operator가 vulnerability
label을 Race command로 잘못 routing해33.907s Race를 한 번 더 실행한 사실은 별도 correction
receipt로 보존했다. 이를 vulnerability 성공으로 사용하지 않고 교정된 exact canonical command의
15.497s 성공만 사용한다. Module version/provider signer policy/실행 중9410 source 변경은0이다.

A는 IMPLEMENTED/WIRED/QUALIFIED=YES(named supply local), whole P3 ACCEPTANCE_CLOSED=NO다.
다음 B에 필요한 실제 ZIP 형태는 두 entries, native107,308,936B다. File을 mode0600 data로 읽은
readelf 진단은 ABI/guest execution qualification이 아니며 B의 정확한 수정 지도에서만 사용한다.
Supply zip·active verifier와 SDK/quality/관찰기·원본은 후속 소비를 위해 유지한다.
