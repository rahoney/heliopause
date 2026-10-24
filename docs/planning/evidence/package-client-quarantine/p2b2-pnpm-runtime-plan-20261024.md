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
  Native current role set은 기존4에서5, 전체 asset producer는 기존9에서11로 확대한다. Native pnpm과 별도 read-only 고지를 각각 포함한다.
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

### Native 출력·배포의 두 번째 검토 후속

실제 materialization consumer는 `ReadToolArchive`와 기존 조회가 같은 parser를 사용한다.
모든 late member/trailing 검증이 끝난 native bytes만 반환하고, 명시 외부 owner-only parent의
새 pnpm-linux-amd64에 readback/hash·0555·no-replace link·directory sync로 게시한다.
유지보수 script는 제품 `runtimeidentity`를 import하지 않는다. 기존 selectedVersion/render처럼
선택한 source의 runtime lock을 읽고 schema/정확한 PNPM 필드·고정 proof 경로·bounds를 확인한
뒤 실제 provenance/archive/native 검증을 한다. 초기 compiled tuple import는 canonical
architecture에서 거부됐으며 규칙을 확대하지 않는다. 실패 raw는 보존한다.

배포 입력 추가 점검에서 고정 source91774e72의 [pnpm LICENSE](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/LICENSE)와
[공식 제3자 고지](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/npm/pnpm/THIRD-PARTY-NOTICES.md)를 확인했다.
Native binary만 넘기는 최초10-asset 안은 고지 전달이 빠져 배포 완료안으로 사용하지 않는다.
Native5개는 유지하고 read-only 고지 asset 하나를 더해 전체11개로 연결한다.

배포 단계의 추가 수정 지도(아직 미구현):

- `pnpm_tool.notices`의 고정 path/hash와 production `tools/pnpm/NOTICE`가 고정 source의
  두 원문을 소유한다. Generated/drift는 source binding과 bytes를 확인한다. Frozen PNPM
  maintenance는 명시 NOTICE도 state metadata로 고정·render하고 새 source에 이전 고지를
  재사용하지 못하게 한다. 조회 전용 verify-pnpm-release의 기존3파일 계약은 유지한다.
- Native materialization 뒤 고지 출력은 같은 selected owner의 정확한 bytes만 0444로
  공급한다. Manifest asset은 pnpm-linux-amd64(`pnpm`, linux/amd64)와
  pnpm-NOTICE.txt(`release-metadata`, linux/amd64)다. 새 범위는 current11 assets를
  완전히 요구하며 기존8/9 구조 읽기를 새 native 승인으로 바꾸지 않는다.
- NativeUpdateFiles는 exact5 roles, 기존 metadata loop는 SBOM/runtime images/NOTICE3을
  재사용한다. 따라서 wire payload는 기존6에서8, bootstrap subject는 기존7에서9다.
  Installer의 선택·copy는 NOTICE에 metadata mode를 쓰고 executable로 취급하지 않는다.
  Installed verification·rollback/recovery의 기존 metadata loop를 재사용하고 필요 없는
  파일은 수정하지 않는다. Missing/misclassified/mutated notice도 current set에서 거부한다.
- OS package는 pnpm/0555와 같은 고지/0444를 versioned payload 및 doc 경로에 담는다.
  Workflow의 acquire→검증된 materialize→package→manifest/attestation 실제 순서와 독립
  checker expectations를 함께 갱신한다. 부분 산출물은 정식 release로 게시하지 않는다.

이는 현재 pnpm 배포의 구체 입력/복사 범위 보완이다. 새로운 signed release, 자동 runtime
전환 또는 사용자 main merge 승인을 추가로 추정하지 않는다.

추가 실제 소비자 대조: inspection/githubrelease/release_set.go의 고정 metadata 분기와
releaseactivation의 installed/stage/bootstrap source, releasebootstrap 준비의 metadata 검증에도
NOTICE의 exact current source bytes 검증을 연결한다. Native pnpm은 SBOM의 별도 application
component이며 Go build info로 위장하지 않는다. 합성 fixture만 현 tuple로 갱신한다.

전체 published-set 호출 경로의 추가 대조에서 promotion/native_release_linux.go의 기존
sealed7/selected6 고정 cardinality도 확인했다. 기존 held FD·exact footprint·close/journal/IPC
절차를 재사용하고 current inspected9/selected8만 요구한다. native_release_linux_test 및
cleanup fixture도 새 current member 집합을 따르며 과거 signed7 증거를 편집하지 않는다.
이는 native 배포의 남은 wiring gap이며 guard 프로토콜 대체가 아니다.

Producer의 최종 대조에서는 materialization 후 변경된 native/NOTICE가 source pin과
엇갈린 채 manifest/SBOM에 묶이지 않도록 release-manifest가 두 실제 asset의 SHA/size를
선택한 runtime owner와 독립 대조한다. Native Go buildinfo/metadata write 이전에 거부한다.
