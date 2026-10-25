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

## B 착수 재검토 (A commit3eaeaa1)

Producer의 Tools v2 array/SBOM/current native set과 설치 metadata를 함께 갱신한다. PNPM의
기존 semantic release 순서 검사는 공용 ordered-command helper로 기계적으로 추출하고,
Bun은 독립적인 expected command list를 추가한다. 두 실제 소비자의 같은 검사일 뿐
source에서 기대값을 추출하거나 arbitrary shell을 지원하지 않는다. Native build step의
Bun materializer는 한 줄의 exact command로 연결해 기존 PNPM multi-line argument census와
혼동하지 않는다. 새 native counts10/export11의 wire/preparation 실제 소비자·current unsigned
fixture만 갱신한다. 기존 historical signed 자료와 Root 1GiB/512MiB 예산은 유지한다.

B 전체 기본 회귀에서 추가 확인: NativeReleasePromotion.Promote(native_release_linux.go)는
same target/Policy ALLOW/manifest·stored Evidence·held descriptors를 검증한 뒤 기존9 inspections/
8 selected files를 요구한다. Bun 두 actual members를 포함하는11/10으로만 갱신하며 다른
Target/Unknown/동일성 guard는 유지한다. Domain/Application은 generic complete set 구현을
재사용해 수정하지 않는다. Github release complete-set과 promotion의 unsigned fixture와
producer releaseAssets helper가 PNPM-only여서 실패했다. 각각 Bun native/notices를 추가하고
실제 graph11nodes/10edges·native sandbox6·required checks/Evidence28을 확인한다. 과거
signed 자료를 바꾸거나 부분 집합을 현재 지원으로 인정하지 않는다. Raw first reviewed Quick은 유지한다.

추가 actual inspection owner: internal/inspection/githubrelease/release_set.go는 frozen manifest의
exact subject/digest/size 뒤 PNPM notices를 passive64KiB로 분류·검증한다. Bun notices에도
동일한 fixed passive 분류와 ValidateBunNotices를 연결한다. Bun native는 기존 native inspector에
남겨 필수 세 검사를 줄이지 않는다. 현재 완전한 집합은 native6×3 + (manifest+passive4)×2=28
required checks다. 남은 producer test의 asset count11 기대값은 현재13으로 갱신한다.

## B named local 결과

[Complete release/installed tuple](./p3b2b2b-bun-release-local-20261025.json)은 Bun native/notices를
현재 Tools2/SBOM/asset13과 실제 inspector/promotion/bootstrap/installer/activation에 연결했다.
Native6·wire10/export11·source11subjects/required28을 유지하고 historical 자료 읽기와 현재
admission을 구분한다. Final Quick109.377/security13.242/docs/workflow PASS, 이전 여덟 package
full Race PASS와 마지막 세 affected whole package Race41.833s PASS를 실제 test dependency
graph26variants로 구분해 기록했다. 첫 whole Race의 FAIL을 전체 PASS로 바꾸지 않는다.
Separate mount/network namespace의 실제12cases와 protected Host snapshot 동일을 확인했다.
이는 synthetic copy/activation protocol 자료이며 실제 Bun guest나 새 서명 release가 아니다.
기존 PNPM official production도 schema4/current source에서6.347s PASS다. 최초 문법 삽입,
누락된 current counts/unsigned fixture, passive Bun notice 분류와 producer 기대값을 각각 원본과
함께 보존했다. 완료한 namespace copy/build 두 directories75,007,792B는0refs/query errors0
뒤 exact owner/inode/hash로 정리했다. 첫 cleanup은 task 밖 namespace copy의 parent guard에서
삭제 전 중단됐고 두 exact 부모만 구분해 교정했다. Active supply/SDK/quality/source/observer는
보존한다. 다음 B2c는 설치된 공급/guest volume·정확한 gVisor startup/lock 역할이며 whole P3는
열려 있다. Small local 단위로 원격 push/CI는 만들지 않는다.
