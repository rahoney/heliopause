# P3-B2b Bun production runtime owner 착수 지도

Baseline6e927b4. M13-004 하나만 active다. P2 remote9410의 CI410/cu132 qualification은
독립적으로 유지한다. Read: P3 baseline/B1/B2a, 확장 계획§2–6, runtime maintenance§1–4,
D-012, 기존 P2b2 native 공급/배포 지도와 ci/sandbox/artifact/evidence 관련 항목.

## Baseline audit

| 실제 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun publisher/ZIP/ELF reader와 developer verify/candidate — 378d1e1/2e85c92/6e927b4 | YES | YES | YES(named local) | NO(whole P3) | production pin/frozen maintenance/installed supply |
| Bun1.4.4 native query — B2a runc actual | YES | YES(candidate diagnostic) | YES(runc version/revision only) | NO(product guest) | exact gVisor role/observer startup/version/lock |
| runtime lock/generator/support drift/updater — 기존 schema3 PNPM/UV | YES | YES | YES(기존 named scope) | NO(Bun) | schema4 Bun owner/strict proof/derived constants/plan/apply/no-op |
| release manifest/package/installer/activation — generic native tool records+PNPM | YES | YES | YES(PNPM) | NO(Bun) | 별도 Bun asset/notices의 complete tuple, release/설치/rollback 회귀 |
| Bun production pin·notices·maintenance·release·installed 공급 | NO | NO | NO | NO | 아래 단계별 실제 gap |

## 먼저 확인할 production 자료

공식 signed checksums는 archive digest만 인증한다. Source38afee6c와 version은 discovery/native
query 사실로 구분하고 SLSA source/build proof로 쓰지 않는다. 공식 source의 LICENSE.md는
Bun 자체와 다수 정적 연결 component를 따로 기술한다. PNPM MIT NOTICE를 Bun으로 바꾸어
재사용하지 않는다. Exact source의 원문과 Bun 내장 license 출력 유무를 network-none runc
진단으로 확인하고 실제 distribution notices의 bytes·출처·한도를 먼저 결정한다. 출력 확인은
법적 완료 판정이나 production admission이 아니다. 이 자료가 확정되기 전에 placeholder
Bun production NOTICE나 fabricated provenance를 만들지 않는다.

## 단계별 exact 수정 지도

1. B2b1: scripts/runtimes.lock.json의 schema4 bun_tool에 exact version/amd64·archive URL/SHA/
   size, 고정 publisher/checksums 경로·SHA, observed source, native path/SHA/size와 passive
   notices를 단일 owner로 추가한다. Node guest relation은 현재 locked Node image를 재사용한다.
   scripts/generate-runtime-lock.go runtimeLock/validate/render와 internal/runtimeidentity 생성값,
   scripts/check/runtime_versions.go owner/proof/support drift, scripts/version-support.lock.json의
   실제 source review를 연결한다. Old schema/current schema·missing/unknown/duplicate/hash/size/
   URL/architecture/current source-body drift를 거부하고 기존 PNPM을 유지한다. 생산 pin은
   공급/native diagnostic 근거이며 gVisor/전체 Bun 지원 완료를 뜻하지 않는다.
2. B2b1의 유지보수: scripts/update-runtime/{main,metadata,patch,apply}.go의 기존 journal과
   stable plan/apply/verify에 Bun frozen supply adapter를 연결한다. 기존 Bun reader를 공유하고
   explicit signed archive/native/notices를 state에 고정한다. Same-version은 network0/pin 보존,
   변경은 전체 proof/bytes를 재검증하며 stale/partial write/recovery·unknown data를 거부한다.
   readPNPMMaterializationOwner와 runtime-image-manifest의 current lock schema 소비자도 갱신해
   Bun 때문에 기존 PNPM이 깨지지 않게 한다. 각 실제 테스트 fixture만 current schema로 맞추고
   과거 raw/old-version 부정 증거는 수정하지 않는다.
3. B2b2: scripts/runtime-image-manifest·release-manifest/release-packages, releaseinstall/
   releaseactivation/releasebootstrap와 실제 release-build workflow에 별도 native Bun asset와
   notices를 연결한다. 기존 generic tool/activation loop를 재사용하고 추가로 필요한 exact set
   guard만 수정한다. Historical parsing과 current complete tuple을 구분하며 SDK/observer/
   installer 검증을 생략하지 않는다. 단일 producer materialization은 production owner를 다시
   확인하며 candidate command의 성공을 installed activation proof로 대체하지 않는다.
4. B2c: 기존 sandbox native volume/tool supply·typed executor·observer exact profile에 실제
   Bun role만 연결한다. 이 단계의 exact call graph/argv/native lock 표본을 별도 재검토한 뒤
   구현한다. 글로벌 ELF/CPU/observer 예외나 ambient Host Bun은 없다.

## 두 번째 검토와 검사 경계

기존 engine/runtime maintenance/installer를 재작성하지 않는다. Bun 서명이 보장하지 않는
version/source/build assertion은 필드와 evidence에서 구분한다. Shared helper의 전역 한도를
올리지 않고 Bun native107,308,936B의 실제 한도는 Bun owner에서만 검증한다. Source version을
여러 descriptor/C++ 정책에 literal로 중복하지 않는다. Canonical Quick/security/docs와 실제
관련 full Race·deterministic generator/no-op/changed-plan·실제 supply를 로컬로 묶는다. 원격은
현재 P2 timer를 소비하고 다음 Bun 공통 runtime/end-to-end checkpoint에서만 실행한다.

## 실제 notices discovery와 해석

Exact source LICENSE.md14,772chars를 raw에 보존했다. Bun --help는 license flag를 열거하지
않으며, 별도 --license는 exit0이지만 general usage/help2,301B만 출력했다. 이 stdout을
notices로 쓰거나 supported_success라는 transport field를 기능 성공으로 전용하지 않는다.
원본 stdout/첫 receipt와 p3b2b-bun-license-interpretation을 함께 보존한다. 각 격리 container는
종료·삭제·정상 exact-ID query 부재 확인했다. Official source의 license inventory 원문과
source URL을 passive NOTICE로 보존하고 version/build provenance나 법적 완료 판정으로
표시하지 않는다. 실제 whole release qualification은 B2b2와 P3 최종 검증에 남긴다.

B2b1 세부 수정 검토: Bun verified supply는 이미 검증한 archive/signed bytes를 내부 struct에
보존해 freeze에서 경로를 다시 읽어 바뀐 입력을 저장하지 않는다. New Bun updater owner는
jsonSpans의 duplicate/정확한 key census를 적용한다. Canonical generator의 기존 전체 lock
strict decoder/validate를 유지하며 Bun type/format/bounds를 추가한다. PNPM/current manifest의
schema 소비자와 실제 fixture의 schema만4로 올리고 이 단계에서 Bun release Tool entry는
추가하지 않는다. 실제 asset/installer 연결은 B2b2에서 complete tuple과 함께 수행한다.

Schema4 소비자 보완 검토: 전체 Race의 기존 gVisor capture actual이 schema3 고정으로
실패했고 source 검색에서 release-manifest도 같은 current owner를 읽음을 확인했다.
update-runtime/gvisor.go captureGvisor의 current lock 판정과 release-manifest/main.go의
current lock 판정 및 main_test.go current fixture를4로 갱신한다. Bundle members/compiler/
release trust/asset set 검사는 유지하고 historical raw는 바꾸지 않는다. 이를 Bun guest
qualification이나 gVisor 재빌드 요구로 확대하지 않는다. 첫 Race/Quick 원본을 보존한다.

## B2b1 named local 결과

Schema4 Bun owner/proof/native/notices와 generated identity·support/drift를 연결했다. 기존
plan/apply/verify는 verified bytes 자체를 freeze하고 재검증하며 same-version/no-op은
network0/기존 pin을 유지한다. [B2b1 local receipt](./p3b2b1-bun-owner-local-20261025.json)의
canonical 및 여섯 package full Race69.257s와 actual frozen maintenance10.496s가 PASS다.
Actual은 중단 apply/재개·재실행과 고정 archive 변조 거부를 포함하며 fixture0.0.1은 과거
공식 Bun 실행 증명이 아니다. 첫 uppercase 오류 gate, 새 Bun이 PNPM verifier로 들어간
조건 오수정, 빠진 capture schema 소비자는 원본과 함께 보존했다. 교정 후 기존 capture/
release 소비자도 검사했다. B2b2 native release/package/installed tuple과 B2c gVisor/whole
P3는 미완료다. Source/build provenance가 없는 사실과 observed metadata 구분을 유지한다.
