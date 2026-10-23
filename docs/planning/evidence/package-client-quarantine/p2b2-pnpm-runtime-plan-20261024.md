# P2b2 pnpm runtime — 단일 pin·공급·배포·admission 지도

선행: [P2a](./p2a-pnpm-supply-local-20261024.json),
[P2b1](./p2b1-pnpm-native-local-20261024.json)의 실제 서명·archive/native identity와19개
native 형식 관측. 현재 Go/Node/npm/Python/UV/Rust pins와 설치 파일은 유지한다.
PNPM12.15.0 source91774e72/run39583314875의 공식 bytes를 연결하는 현재 P2 범위다.

Read: [P2 감사/전체 순서](./p2-pnpm-baseline-20261024.md),
[runtime 유지보수](../../../engineering/05-runtime-upgrade-maintenance.md),
[P1b 단일 pin·공급 owner](./p1b-uv-runtime-baseline-20261021.md),
[D-012](../../../threat-model/04-trusted-tooling-and-evidence.md), ci/sandbox/artifact/evidence 오답노트.

## 1. 실제 baseline 판정

| 목표 / 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| pnpm 공식 archive/native 검증 — releaseprovenance/pnpm_release.go·pnpm_tool.go, update-runtime/pnpm_release.go | YES | YES(offline command) | NO | NO | 제품 공급·원격 qualification; local 검사·실제 명령·runc 형식은 PASS |
| 기존 version owner/generator/drift/support/updater — runtimes.lock schema2와 generate-runtime-lock, check/runtime_versions, update-runtime | YES | YES | YES(기존 범위) | YES(기존 범위) | pnpm tool owner/schema·proof/bytes freeze·render/plan/apply/no-op |
| 기존 release asset/metadata/installer·native activation — release-manifest, release-packages, releaseinstall, releaseactivation | YES | YES | YES(기존 범위) | YES(기존 범위) | pnpm 외부 native asset와 current tuple/record/rollback 연결 |
| Node image·network/helper/session/observer/volume lifecycle — sandbox/npm_resolver.go, uv_tool_supply.go·uv_runtime_volume.go, observer.cc | YES | YES | YES(기존 범위) | YES(기존 범위) | pnpm exact tool 공급/root 역할·Node query child·실제 gVisor startup/query/lock |
| pnpm 단일 owner·전체 공급/배포·admission | NO | NO | NO | NO | 아래 순서 전체; 실제 typed network/Policy/project transaction/CLI는 P2c/d에 남음 |

기존 resolver/installer/서명·restore 프로토콜을 대체하지 않는다. Node/npm을 pnpm이라고
표시하지 않으며 PNPM Rust binary를 Go compiler 일치 검사 대상으로 잘못 분류하지 않는다.
기존 Rust/Cargo image pin은 별도 artifact toolchain으로 유지한다.

## 2. 선택한 공급 방식과 단일 owner

PNPM 공식 native release asset를 사용한다. 새 custom OCI image나 ambient pnpm/Corepack,
실행 중 latest 설치·자동 client/runtime 전환은 만들지 않는다. Generic descriptor에 실제
PNPM 항목만 추가하며 후속 Bun/pipx 등의 placeholder는 만들지 않는다.

- Runtime lock schema3의 `pnpm_tool`에 exact version/amd64·공식 archive URL/SHA/size,
  source/run·bundle 경로/hash, native SHA/size를 한 번 기록한다. 값은 P2b1 실제 검증에서
  얻고 수기로 추정하지 않는다. 공식 proof는 `tools/pnpm/release-proof.json` 한 곳에서 관리하고
  testdata를 production 공급 입력으로 사용하지 않는다. 과거 test fixture는 그대로 둔다.
- 생성된 runtimeidentity가 product·maintenance·drift·release 소비자에 같은 값을 전달한다.
  PNPM은 trusted native tool이며 npm registry라는 SourceID는 유지한다. Source 또는 native
  file hash를 client descriptor나 C++ 정책에 version별 literal로 중복 선언하지 않는다.
- 개발 유지보수의 PNPM plan은 명시 proof/archive를 offline 검증하고 state에 고정한다.
  Render/apply는 같은 source·proof·archive/native identity를 다시 검사한다. Same-version은
  network0/원래 pin 보존이며 stale/dirty/부분 write/다른 signer·subject·파일 교체를 거부한다.
- Release build는 고정 owner의 archive를 준비·재검증하고 pnpm native **하나만** 외부 dist에
  출력한다. 실제 두 소비자인 offline 조회와 materialization은 같은 parser를 공유한다.
  Host에서 pnpm을 실행하거나 archive를 그대로 파일시스템에 풀지 않는다. dist/node-gyp
  subtree를 설치·실행하지 않는다. 원래 archive UID1001을 Host 권한으로 승계하지 않는다.
- 서명된 HAA release set의 Linux amd64 native asset/installed role에 pnpm을 연결한다.
  Native current role set은 기존4에서5, 전체 asset producer는 기존9에서10으로 확대한다.
  pnpm bytes는 upstream proof와 canonical native hash·HAA whole-set provenance에 함께 묶인다.
  설치와 업데이트는 기존 protected stage·record·lease·health·rollback/복구 안에서 처리한다.
- Runtime image metadata는 새 v2에서 기존 exact5 images와 PNPM tool의 별도 record를 구분한다.
  외부 native binary를 OCI image로 위장하지 않는다. 현재 compiled tuple은 빠진 tool·다른
  version/hash를 거부한다. 과거 raw signed fixture를 편집해 새 tuple 성공으로 만들지 않는다.
  Historical 구조 읽기와 새 설치/activation의 완전성 판정을 구분한다.

## 3. 수정 위치와 방법

| 실제 owner / symbol | 좁은 변경과 검증 |
| --- | --- |
| scripts/runtimes.lock.json·version-support.lock.json, tools/pnpm/release-proof.json | 첫 verified native pin/proof; 초기 support record는 공급/형식과 제품 검역 미완료를 분리 |
| scripts/generate-runtime-lock.go `runtimeLock`, `render`, 검증; runtimeidentity/runtime_lock_gen.go | schema3·strict pnpm owner/proof hash·generated identity; byte-identical 재생성, missing/unknown/duplicate·size/hash/URL/platform 거부 |
| scripts/check/runtime_versions.go `runtimeVersionOwners`, `checkRuntimeVersionOwners` 및 tests | pnpm source/support/generated 소비자·proof drift; 다른 owner와 Go compiler 범위 유지 |
| scripts/update-runtime/{main,metadata,patch,apply,uv_supply}.go의 기존 flow | PNPM component와 명시 frozen supply adapter; 기존 apply journal/checked source·idempotence 재사용, UV supply 변경 금지 |
| releaseprovenance/pnpm_tool.go 및 update-runtime/pnpm_release.go | 검증된 native bytes의 명시 output consumer; 조회와 출력이 같은 parser 사용, no-replace external output·변조/상대경로/취소 회귀 |
| .github/workflows/heliopause-release-build.yml, scripts/release-{manifest,packages}, scripts/check/{workflow,workflow_structure,release}.go | 고정 공급→검증→native asset→OS package→whole manifest/attestation의 실제 순서; 주석·누락·조건부 우회·다른 hash 거부, 새 CI job 없음 |
| scripts/runtime-image-manifest, releaseinstall/{manifest,metadata,installer}.go, testutil/releasefixture | v2 images+tool 계약과 exact complete asset set; new current fixture binder만 갱신, historical proof 불변 |
| releaseactivation/{installed,stage,copy,wire,health,recovery}의 실제 role 소비자, releasebootstrap/preparation_linux.go | 같은-runtime/native role 확장·기존 complete evidence/ownership 보존; missing/foreign/substituted/중단·rollback·복구 negative |
| sandbox의 기존 tool/volume/session helper와 typed executor | pinned PNPM file만 read-only private guest tool로 전달; digest/size·Root copy·nonce/generation·single owner/attachment와 confirmed cleanup 확인 |
| tools/gvisor-observer/observer.cc·observer_latch_test.cc | PNPM profile의 exact exec/class/loader·Node version query lineage만 연결; 기존 npm/UV/Python/ARTIFACT predicate 전역 완화 금지 |
| bootstrap actual runtime fixtures·CI Linux entry | 실제 PNPM version/query/lock와 기존 소비자, helper wait/restore·Host 전후 일치; 구현된 target만 연결 |

표의 activation 파일 중 수정이 필요 없는 공용 loop는 그대로 재사용한다. 파일 개수를 목표로
모든 owner를 고치지 않는다. 실제 ref/symbol과 first failure를 대조한 필요한 gap만 반영한다.

## 4. 순서·두 번째 검토·검증

1. 단일 owner/proof/generator와 read-only drift, frozen maintenance를 먼저 연결한다.
   실제 current-version no-op·explicit wrong inputs·부분 apply/복구와 canonical profile을 묶는다.
2. Native materialization와 release/installer/namespace 소비자를 연결한다. Payload/명칭/version/
   path가 다른 native file은 거부한다. Root 전환 시험은 기존 namespace·backup/stop/source guard를
   재사용하고 실제 Host 설치는 새 qualification 전에 바꾸지 않는다.
3. Guest 공급/admission과 실제 startup/query/lock를 연결한다. 공식 native source의 자동 pnpm/
   runtime provisioning·packageManager/devEngines/config include/hook·공유 store를 우회하지
   않도록 설정과 허용 project input을 고정한다. 새로운 fault는 원본/정확한 syscall·class·기존
   정상 소비자를 대조하고 PNPM exact role만 좁게 보완한다.
4. 이후 P2c graph/public binding, P2d guarded publication/CLI를 계속한다. Runc 관측의
   loopback tarball URL을 public 인증으로 쓰거나 nlink>1을 일반 허용으로 바꾸지 않는다.

사전 두 번째 검토: version pin 하나의 변경도 배포물·현재 compiled runtime tuple·native
원본 inventory에 영향을 준다. Tool record 누락을 default로 허용하거나 old record를 새 tuple로
자동 승격하지 않는다. Pnpm update의 manifest 변경/peer context는 선택 adapter의 책임이며
이번 공급 단계가 npm install/update engine을 대체할 이유가 아니다. HAA self-update의 기존
same-runtime 거부·OS 인증·전체 파일 검역/복구 조건을 유지한다. 새 runtime-changing 자동
self-update나 공개 배포를 이 단계에 추가하지 않는다.

검증은 format/Quick/security/docs와 변경된 workflow/release-gate/freshness·관련 전체 Race,
실제 공급/no-op/materialization/namespace·exact gVisor를 경계별로 묶는다. Native 공통 전제의
원격 checkpoint와 최종 P2 candidate의 기존 npm/PyPI/영향 CPU·CUDA·macOS/Required 범위를
구분하고 실제 동일 source와 profile을 연결한다. Helper/문서마다 full CI를 push하지 않으며
실행한 CI는 해당 profile 실측·현재 시각으로 타이머를 잡고 확인한다. 마지막 consumer가 끝난
테스트 payload/사설 image/volume/cache는 참조 부재와 실제 종료·보존 원본 확인 뒤 삭제한다.
