# P3-B2b2 Bun release native supply와 설치 tuple

Baseline03d4870(직전Bun owner b51290c). M13-004 하나만 active이며 P2 remote9410/cu132411의
95분 timer04:22:46UTC를 유지한다. Read: P3/B2b 지도·runtime maintenance·M13 release 계약,
기존 P2b2 공급/배포 지도, releaseinstall의 manifest/metadata/installer, runtime-image-manifest,
releasefixture와 activation/bootstrap 소비자, 관련 ci/sandbox/artifact/evidence 교훈.

## 독립 baseline audit

| 책임 / 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun signed ZIP/native data·candidate / B1,B2a | YES | YES | YES(named local) | NO(product) | production owner로 재검증한 native/notices 출력 |
| Bun schema4 owner·frozen maintenance / b51290c | YES | YES | YES(named local) | NO(whole P3) | release producer/installed tuple |
| no-replace native publisher / native_asset.go, PNPM 생산 소비자 | YES | YES | YES(named local) | NO(Bun production) | 같은 publisher를 실제 Bun 생산 caller에 연결 |
| generic runtime Tools·release asset/SBOM/package/installer/activation | YES | YES(PNPM) | YES(existing named) | NO(Bun) | Bun tool/notices complete set와 모든 actual 소비자/negative |
| Bun production materializer/asset/installed supply | NO | NO | NO | NO | 아래 두 부분; 기존 installer/observer engine 재구현 없음 |

## A — production materialization exact map·두 번째 검토

1. update-runtime/bun_materialize.go에 materialize-bun --root/--archive/--output/
   --notices-output를 실제 추가한다. Candidate command와 구분하고 source의 exact schema4
   Bun owner를 읽는다. readBunMaterializationOwner는 같은 bytes의 jsonSpans census/strict
   fields·version/amd64/runtime relation·official URL·publisher·source/native/notices bounds를
   검사한다. 실제 selectedVersion과 이 caller의 key census만 공용화한다.
2. Source owner가 지시하는 고정 signed checksums/passive notices를 bounded reader로 읽고
   hash/size/source prefix를 대조한다. Existing VerifyBunChecksums→ReadToolArchive로 complete
   archive/native를 검증하고 owner native SHA/size와 다시 대조한 뒤 publisher를 사용한다.
   0555 native 하나와0444 passive notices를 source 밖 새 경로에 출력한다. Host 실행/network0,
   root/guest/Policy 권한0이며 stdout은 실제 공급 receipt다. 기존 PNPM publisher는 불변이다.
3. main.go에 실제 dispatch, bun_materialize_test.go의 schema/duplicate/alias/unknown/path/size/
   foreign source·proof/drift·재출력 거부, realcorpus의 공식 production bytes·mode/owner/single-link·
   existing output/변조 거부와 기존 candidate/PNPM 회귀를 확인한다. 작은 local checkpoint다.

검토: signature가 version/source build를 서명한다고 주장하지 않는다. Owner adoption과 실제
runc version은 B2b1/B2a 근거이며 product gVisor는 여전히 후속이다. Input expected native를
caller flag로 받지 않는다. 경로를 두 번 읽은 데이터로 첫 검증을 무효화하지 않고 같은 read
bytes의 typed owner를 사용한다. Source내부/기존/symlink/foreign output은 덮어쓰지 않는다.

## B — complete producer/installed tuple의 exact map

A 뒤 아래 실제 소비자를 함께 연결하고 작은 literal 추가마다 원격 CI를 만들지 않는다.

- runtimeidentity/native_paths.go: 실제 Bun asset name을 generated architecture에서 파생.
- runtime-image-manifest/main.go: schema4 Bun owner의 별도 Tool record와 native/notices bounds.
  기존 v2 Tools array 계약을 사용하며 exact2 unique tools/current tuple 소비자를 함께 갱신.
- release-manifest/main.go: Bun owner를 읽어 native/notices hash/size를 binding, complete asset
  inventory와 SBOM의 Bun application component. release-packages/main.go는 기존 native version
  directory의 bun0555/notices0444와 passive doc copy를 포함한다.
- releaseinstall/manifest.go: 역사 자료 parsing과 새 native-update의 complete Bun+PNPM role set을
  구분. Bun128MiB/native와64KiB/notices 한도는 Bun role에만 적용하고 일반 limits를 올리지 않음.
  metadata.go는 exact2 tools와 current SBOM/native notices를 검사한다. installer.go는 실제
  Bun role/notices 선택·검증·copy를 기존 loop에 연결한다.
- releaseactivation/{installed_linux,stage_linux,bootstrap_files_linux}.go 및
  releasebootstrap/preparation_linux.go: 기존 protected reader/complete tuple에서 Bun notices를
  검사한다. 공용 file/role copy·stop/restore/recovery는 실제 gap이 없으면 그대로 재사용한다.
- internal/testutil/releasefixture/runtime.go는 unsigned current fixture만 exact tuple로 연결한다.
  각 producer/installer/activation/bootstrap의 named fixture·음성 검사를 갱신하며 보존된 실제
  signed manifest/attestation bytes는 수정하지 않는다. Historical read 성공은 current admission이 아님.
- release-build workflow의 기존 native build 단계에서 lock의 Bun URL/size로 archive를 준비하고
  production materializer와 manifest assets를 연결한다. scripts/check/workflow.go의 기존 semantic
  순서/누락/조건부 우회 검사를 Bun 실제 producer에 적용한다. 새 CI job/중간 자동 push 없음.

B 시작 전에 실제 call graph/fixture 검색에서 누락된 owner가 있으면 이 지도에 근거와 함께
갱신한다. 구현된 producer/installer engine을 대체하지 않는다. Canonical Quick/security/docs/
workflow와 관련 full Race·공식 supply output·기존 설치 namespace 실제 소비자 검증을 묶는다.
B2c gVisor exact role/startup/lock와 Bun resolver/layout/CLI·whole P3 acceptance는 남는다.

A의 파일 배치: candidate command는 bun_materialize.go에 보존하고 생산 command는
bun_production_materialize.go에 분리했다. Actual 두 소비자가 쓰는 selectedBunOwner와
readBunMaterializationOwner/type은 bun_owner.go다. readBunSupply/owner/frozen version의 같은
stable 문법만 validBunVersion으로 재사용하며 source authority나 한도를 변경하지 않는다.

B의 call graph 추가 확인: releaseactivation/wire_linux.go의 fixed native+metadata file count8은
Bun native/notices 두 files를 포함하는10으로 바뀌며 releasebootstrap/preparation_linux.go의
export count9는11로 바뀐다. 해당 wire/preparation 테스트도 같은 actual current set을 확인한다.
ParseReleaseManifest는 historical8/9/11 자료를 계속 구조적으로 읽되 current13의 Bun pair는
PNPM pair/installer와 함께 요구한다. NativeUpdateFiles/current metadata admission은 완전한
두 clients만 허용한다. 보호된 read/copy/stop/restore/attestation 조건은 유지한다. 전체 asset
512MiB/합계1GiB bootstrap 예산은 그대로이며 Bun role128MiB를 PNPM64MiB와 구분한다.

## A named local 결과

[생산용 materializer](./p3b2b2a-bun-production-local-20261025.json)는 source schema4 owner/proof/
native/notices를 다시 검증하고 native0555/notices0444를 외부 새 경로에만 출력한다. Shared
selectedBunOwner/validBunVersion은 실제 두 owner 소비자와 공급/유지보수에서 사용한다.
Quick178.761/security9.287/docs0.249s·관련 세 package full Race32.354s, official production
actual5.633s와 기존 frozen maintenance actual10.096s가 PASS다. 첫 realcorpus fixture의
PNPM-only member compile 오류는 보존했다. 그 한 제외 파일만 수정됐음을 실제 Go list/
source bytes 대조로 확인했으며 corrected realcorpus build/run과 최종 all-source format을
검사한다. Production/default compiled code는 불변이다. 다음 B는 앞 complete producer/
installed tuple 지도대로 진행하고 runtime gVisor/whole P3는 미완료다. 작은 A는 push하지 않는다.
