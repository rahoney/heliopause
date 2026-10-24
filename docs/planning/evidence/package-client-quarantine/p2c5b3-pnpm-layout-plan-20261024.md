# P2c5-B3 — archive inventory와 native 전체 배치의 검증

2026-10-24, B2 commit83db86c8/tree7d1bba1c 기준. 같은 M13-004/P2이며
[B1](./p2c5b-pnpm-production-context-plan-20261024.md),
[B2](./p2c5b2-pnpm-target-lifecycle-plan-20261024.md), Architecture02/03,
Domain08/12/13 및 오답노트 artifact L15/L20·sandbox L08/L18·evidence L11/L16을 읽는다.

## 1. Source 변경 전 baseline audit

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| npm 정적 tar 순회·manifest·기존 composite/M3 — inspection/npm/static.go | YES | YES | YES | YES | 재작성하지 않음; optional inventory capture만 추가 |
| 전체 acquire/verify→adapter batch→record/Policy — application/dependency_batch_inspect.go | YES | YES | YES | YES | 기존 projectAwareInspection 재사용 |
| frozen context/acquired/복수 peer instance — artifact/npm/pnpm_inspection_context.go | YES | YES | YES(B1/CI403) | NO(full P2) | introduction 시 byte 검증과 실제 전체 layout 결합 |
| native offline store/fetch/install — sandbox/pnpm_offline_fixture_real_test.go | YES | YES(test) | NO(product) | NO | production 준비 owner·정리/관찰·전체 결과 검증 |
| exact target/관찰 lineage — sandbox/pnpm_target_lifecycle.go와 observer.cc | YES | YES(primitive) | NO(remote B2) | NO(full P2) | 검증 inventory가 고른 instance 전달·aggregate |
| per-file archive inventory/전체 layout 검증 | NO | NO | NO | NO | 아래 B3a와 B3b |
| product static-all/prepare/aggregate·Host/CLI | NO | NO | NO | NO | B3c 이후; 현재 transitive dynamic INCOMPLETE 유지 |

기존 promotion/npm_promotion.go의 validatePromotionOutput은 installed identity/set/controls와
staged archive digest를 확인한다. Tarball의 모든 file digest를 돌려주는 generic inventory는
없다. Promotion을 Sandbox에 import하거나 별도 tar parser를 복제하지 않는다.

## 2. B3a의 정확한 수정 지도 — 정적 순회에서 선택적 파일 사실 수집

1. artifact/npm/archive_inventory.go에 immutable ArchiveInventory와 ArchiveEntry를 둔다.
   정확한 AcquiredArtifact 전체와 package/ 상대 file·directory 경로/size/mode/SHA256,
   검증된 package.json bytes를 보존한다. Constructor는 구조만 검증하며 Verification,
   Policy 또는 실행 권한을 만들지 않는다. Getter는 copy를 반환하고 zero는 invalid다.
   Duplicate/escape/backslash/제어문자·file-as-parent 충돌, 잘못된 kind/size/mode/hash,
   manifest entry/body 불일치를 거부한다. 기존 entry/file/decoded 상한보다 넓히지 않는다.
2. inspection/npm/static.go의 기존 Inspect는 동일 내부 순회를 그대로 호출한다. 새
   InspectArchive만 optional capture를 전달한다. 기존 manifest 검증·finding·required check·
   Evidence 순서와 HostInstallAction 사실은 보존한다. Non-manifest bytes는 기존 tar reader에서
   SHA256을 streaming 수집하며 압축 해제·Host 설치는 하지 않는다. 별도 archive parser는 없다.
3. inspection/npm/archive_inventory.go의 private capture는 같은 열린 파일에서 compressed
   bytes의 크기/SHA256을 다시 확인한다. Bounded reader와 gzip footer/CRC·전체 decoded
   한도, tar 종료 뒤 zero padding, 단일 gzip member 뒤 trailing compressed bytes 부재를
   확인한다. Partial/위반 report에는 inventory를 반환하지 않는다. Context 취소 원인을
   보존하고 추가 metadata/byte 결합 실패를 정상 inventory로 바꾸지 않는다.
   Optional capture의 더 엄격한 단일 표현은 기존 Inspect 지원 범위를 바꾸지 않는다.
   이 경로만 os.OpenRoot의 confined open/Lstat와 열린 FD SameFile 확인을 추가하여
   intake 밖으로 향하는 source link를 따라 읽지 않는다. 기존 Inspect open은 그대로다.
4. scripts/check/architecture.go에는 inspection/npm→artifact/npm의 값 type edge 하나만
   검토해 추가한다. 이미 PyPI가 같은 방향의 archive type을 소비한다. Core/Application의
   생태계 의존, Sandbox→Inspection 역방향, runtimeidentity/YAML parser 누출은 계속 거부한다.

B3a에서 native 배치 검증기나 runner interface를 미리 만들지 않는다. Inventory의 생성
성공은 실제 introduction/배치 byte 검증·관찰 완료가 아니며 기존 CLI 경로는 계속 같다.

## 3. 다음 연결 순서와 아직 닫히지 않은 세부

B3b는 B1의 모든 instance/edge와 B3a inventory를 입력으로 native node_modules 전체를
검증한다. Pinned PNPM source의 dirname codec·hash/default max length, 생성 metadata,
dependency/peer/root links와 bin wrapper·허용 mode/owner 변환을 실제 output과 대조하여
정확한 수정 지도를 **작성한 후** 구현한다. 단순 directory 이름/exit0은 identity가 아니다.
동일 physical path로 둘 이상의 instance가 매핑되는 충돌·missing/extra/foreign entry와
파일 내용 변경, symlink/hardlink escape·불명확한 alias/special file/owner·size 초과를 거부해야 한다.

B3c는 qualified native volumes·typed network helper·fixed early store env와 기존
store-add/fetch→detach/endpoint0→offline frozen ignore-scripts install을 production owner로
연결한다. Store format은 native가 생성한다. Native prepare session의 complete trace와
cleanup 뒤 immutable 검증 결과만 다음 fresh NPM session에 소개한다.

그 후 Inspection의 PNPM project adapter에서 static-all을 먼저 하고, 미완료 entry가 하나라도
있으면 **모든 entry의 required dynamic 미실행**을 명시한다. 각 archive의 모든 peer instance를
독립 fresh session에서 관찰하고 실제 session ID/원래 오류/이미 관측한 보안 사실을 보존한다.
가짜 aggregate SandboxSession·partial coverage ALLOW·실패 layout 재사용은 없다. Aggregate
표현/정확한 wiring map은 prepare 결과를 확인한 뒤 확정한다. Host staging/records/transaction/
CLI와 전체 P2 qualification은 이 경계 뒤에 남아 있다.

## 4. B3a source 작성 전 두 번째 검토와 검증 계획

기존 정적 순회가 이미 모든 tar header를 검증하므로 교체하거나 새 parser를 만들 필요가
없다. Manifest 이외 payload를 같은 reader에서 읽는 optional collector로 좁힌다. Archive
type은 source adapter package의 pure value이며 network/Policy/Sandbox 의존을 추가하지 않는다.
실제 byte hash는 capture가 계산하고 구조 constructor에 제공하며 constructor의 Valid만으로
source 검증을 대체하지 않는다. 이후 소개 단계에서도 다시 byte hash를 확인한다.

검토에서 gzip/tar EOF만으로는 CRC·trailing member·padding 안의 숨은 내용·decoded 상한을
검증할 수 없음을 확인했다. 새 optional 경계는 전체 bounded stream을 소모하고 단일 표현을
요구한다. 기존 Inspect 결과는 해당 새 API와 같은 것으로 소급 해석하지 않는다. Cancelled
copy, manifest/body mismatch·file ancestor conflict·getter 변조/foreign acquired subject,
동일 크기 archive mutation·size drift·truncation/CRC·second member·nonzero tar tail을 직접
negative로 검사한다. 정상 report는 기존 Inspect와 check/Finding/Evidence 전체가 같아야 한다.

Canonical format→Quick/security/docs, npm artifact/inspection·checker의 해당 전체 Race를
묶어 실행한다. 새 pure inventory에는 불필요한 전체 runtime actual을 반복하지 않으며 B3b/c
실제 native consumer가 연결될 때 A3/B2/public 검역을 다시 검증한다. 작은 commit마다
remote CI를 시작하지 않고 다음 실제 연결 checkpoint에서 B2와 함께 묶는다. 완료한 전용
bulk는 참조·query error0 뒤 삭제하고 현재 소비 중인 private source/cache만 보존한다.

기존 install_action_real_test.go의 두 public archive fixture도 같은 Acquire/SRI 검증 뒤
InspectArchive를 호출하도록 확장한다. 실제 파일 inventory·정확한 acquired subject와
원래 static report/HostInstallAction·저장 Evidence의 일치를 확인한다. 기존 public gate와
임시 intake 정리를 그대로 쓰며 script/PNPM runtime/Host 설치를 실행하지 않는다.

## 5. 첫 로컬 결과

선택적 inventory의 첫 canonical format1.106s/Quick164.383s/security7.871s/docs.253s는
PASS다. 기존 Inspect 결과와 direct negative 회귀를 포함한다. 첫 Quick의 경과 시간만으로
병목 원인을 추정하지 않는다. 실제 public archive와 전체 관련 Race 및 최종 후보 대조는
아직 실행 전이며, production layout/prepare/aggregate와 전체 P2 완료는 계속 열려 있다.

최종 format.370s/Quick36.294s/security5.627s/docs.253s와 artifact/npm·inspection/npm·
canonical checker 세 package 전체 Race26.274s는 PASS다. 실제 public 두 archive의
Acquire/SRI→inventory→기존 report/HostInstallAction·Evidence Store 일치도1.325s/PASS다.
is-number7.0.0은4 entries/3,730B, node-sass9.0.0은334 entries/390,705B였고 script/PNPM
runtime/Host 설치는 실행하지 않았다. 임시 intake/records는 t.TempDir 정리 경로를 따른다.

B3a는 IMPLEMENTED/WIRED(정적 producer)=YES, LOCAL VERIFIED다. Product static-all/전체
배치와 실제 prepare/aggregate의 WIRED·QUALIFIED·ACCEPTANCE_CLOSED는 NO다.
[로컬 영수증](./p2c5b3a-pnpm-inventory-local-20261024.json)이 exact source와 각 범위를 소유한다.
다음 B3b는 공식 dirname codec과 실제 native 전체 output을 대조하여 edit map을 구체화한다.

## 6. B3b0 — 전체 layout의 실제 표현 확인, source 작성 전 감사/두 번째 검토

Baseline7ff0cab8의 B3a는 정적 inventory producer만 연결했다. A3의 capture callback은 이미
native controls/identity/links·endpoint0을 검사한 뒤 bounded Docker tar를 읽게 할 수 있다.
전체 layout parser/verifier·production prepare는 IMPLEMENTED/WIRED/QUALIFIED/
ACCEPTANCE_CLOSED=NO이며 아직 재사용할 구현이 없다.

먼저 새 linux&&realcorpus gated fixture 한 개만 추가한다. A3의 transitive/peer 두 입력을
그대로 사용하고 capture callback에서 전체 node_modules tar를16MiB까지 RAM에 읽는다.
Header 경로/type/link/mode/UID/GID/size와 regular file SHA256을 escaped JSON으로 기록한다.
생성 .modules.yaml/.pnpm/lock.yaml와 .bin wrapper의 bounded bytes만 표현 조사 자료로
보존하며 일반 artifact body를 raw report에 복제하지 않는다. 읽기만 하며 Host extraction,
script 실행·새 Policy/완료 authority·새 production interface는 없다. 원래 complete trace/
typed helper·resource cleanup/Host 전체 불변 조건을 그대로 요구한다.

공식91774e72 source의 dep_path_to_filename.rs/crypto-hash/lib.rs·tests와 modules-yaml/lib.rs를
captured UTF8 bytes의 Git blob까지 대조했다. Max length 기본120이고 짧은 hash는 SHA256의
앞32 hex다. 과거 구현의 MD5를 추정해 넣지 않는다. Scoped slash/peer parentheses 변환,
uppercase/길이 초과 및 ambiguous non-registry escape의 hash input은 다른 분기다. 현재
parser가 거부하는 file/URL source 지원을 이번 codec 조사로 넓히지 않는다. 형식 유지보수는
이 pinned source와 native 실제 output을 함께 확인하고 collision을 별도로 거부해야 한다.

두 번째 검토에서 Docker tar의 hardlink 표현·native mode/ownership 및 생성 metadata/bin
bytes를 A3의 몇 identity/link 검사만으로 추정할 수 없음을 확인했다. 이 fixture의 성공은
**표현 관측과 기존 prerequisite 유지**이며 full layout의 검증/무해성/Policy ALLOW가 아니다.
관측 뒤에 exact allowlist·byte 비교·bin/metadata 처리와 부정 검증 지도를 먼저 확정한다.
Fresh namespace·보호된 compiled test/client/동일 observer를 사용하고 종료 뒤 references/
query errors0을 확인하여 전용 bulk를 삭제한다. CI403이나 B3a pure 검사로 이 actual을
대체하지 않는다. 작은 조사 fixture만으로 remote CI를 추가하지 않는다.

### B3b0 실제 결과와 계획 보완

Actual21.937s/test19.19s는 transitive23 headers(regular11/dir9/symlink3), peer89 headers
(regular56/dir25/symlink7/hardlink1)를 관측했다. 원래 controls/2·4 identity/1·5 links,
complete/금지0·Host전체동일/helperwait0/final original0도 PASS다. 모든 owner는1000:1000,
regular modes는0644/0755와 virtual lock0600, directories0755·symlink0777이었다.
이 수는 두 explicit fixture의 관측값이며 일반 artifact allowlist가 아니다.

Peer의 use-sync-external-store/LICENSE가 react/LICENSE를 가리키는 hardlink로 전송됐다.
따라서 정상 native 전체 output에 대한 무조건 hardlink 거부 가정은 superseded다. 후속
검증기는 양쪽 expected inventory의 size/digest/mode가 같은 이미 검증된 regular bytes로
닫히는 alias만 처리해야 한다. Foreign/escape/cycle·unverified/다른 bytes alias는 거부하며
hardlink라는 이유로 identity를 공유하거나 archive coverage를 줄이지 않는다.

Default native는 private hoist link도 만들며 .modules.yaml은 현재 JSON/layout5·default120,
.pnpm/lock.yaml은 frozen semantic lock이다. .bin은 단순 symlink가 아니라 Node shim script이며
자기 package와 의존 consumer 위치에 생성된다. 이 body는 이름/마지막 target 주석만으로
승인하면 안 된다. Full verifier 전에 pinned renderer와 hoist·generated metadata의 정확한
허용/정규화 방식을 추가 검토한다. 다른 package 파일이 bin으로 실행 가능해지는 mode 변환도
manifest bin/원래 content hash와 결합해야 한다. 관측 성공을 이 검증의 완료로 쓰지 않는다.

전용 runtime11files/165,489,462 logicalB는 모든 UID refs0/query errors0 확인 뒤 삭제했다.
작은 original log·derived shape/source receipt만 보존한다. [B3b0 결과](./p2c5b3b0-pnpm-shape-local-20261024.json)가
실제 범위와 남은 작업을 소유한다.

## 7. B3b1 — instance dirname codec의 정확한 수정 지도/두 번째 검토

Artifact/npm의 pnpm_instance_path.go 한 곳에 현재 parser가 받는 registry-only snapshot
key→virtual-store dirname 변환을 추가한다. pnpmPackageKey를 먼저 재사용하며 file/URL/
workspace/alias와 invalid input은 거부한다. 공식 pre-escape/peer flatten과 ASCII case/120byte
길이 분기, SHA256 앞32 hex를 따르며 별도 store/index를 만들지 않는다. Format constant120은
한 곳에 두고 native .modules.yaml/실제 output과 비교한다. Hash는 path codec일 뿐 source
authentication/Policy identity가 아니며 전체 layout은 모든 dirname 충돌을 별도로 거부한다.

Unit tests는 pinned upstream의 plain/scoped/multiple·nested peers/build metadata vectors와
현재 지원하는 uppercase version/긴 key의 독립 SHA256 vector·잘못된 source를 확인한다.
Gated shape fixture에서 원래 controls를 ParsePNPMLock하고 모든 snapshot의 encoded path에
실제 package.json header가 있음을 요구한다. 이는 path mapping 연결만 확인하며 전체
file/edge/bin 검증을 대신하지 않는다. 같은 fresh native actual을 새 candidate로 확인한다.

두 번째 검토에서 전체 PNPM source codec을 그대로 허용하면 현재 unsupported file/URL
source가 우회 진입하므로 reviewed registry subset만 구현하기로 했다. Long name의 hash
단축이 version separator @를 prefix 밖으로 밀어낼 수 있고 현재 B2 locator는 @를 요구한다.
그런 입력의 lifecycle admission은 아직 미완료이며, 전체 layout의 byte binding 뒤 별도
정확한 shape/negative·actual 검증 없이 observer 권한을 넓히지 않는다. Source drift·native
client 변경은 pinned codec vectors와 actual layout 비교를 통해 fail-closed로 발견한다.

Canonical format/Quick/security/docs·해당 npm 전체 Race와 tagged compile/native actual을
검사한다. 기존 단일 npm/observer 권한은 그대로다. 작은 codec checkpoint만 remote push하지
않으며 B2 이후 실제 prepare 연결 단위에서 묶는다. 끝난 namespace bulk는 다시 삭제한다.

### B3b1 로컬 결과

Canonical format.381s/Quick158.014s/security8.534s/docs.276s와 npm artifact 전체 Race5.807s는
PASS다. Fresh actual20.659s/test17.98s도 원래 controls에서 읽은 transitive2/peer4의 모든
instance dirname→실제 package.json 경로를 확인했다. 같은 native primitive의 원래 controls/
identity/links·complete/금지0·Host전체동일/helperwait0/final original0도 유지했다. 전체 file/
edge/bin/metadata·여러 peer context의 동적 검역 완료로 확대하지 않는다.

종료 뒤 전용 runtime11files/165,519,505 logicalB를 모든 UID refs0/query errors0 확인 후
삭제했다. [B3b1 로컬 결과](./p2c5b3b1-pnpm-codec-local-20261024.json)가 exact source와
검증/정리 범위를 소유한다. B3b2에서는 관측한 generated shim/hoist/metadata를 검증할
정확한 방식을 확정한 다음 full layout verifier를 구현한다. 관찰기와 기존 npm 실행은 불변이다.

## 8. B3b2a — bounded layout tar envelope의 수정 지도/두 번째 검토

Source1f410fd baseline에서 full layout verifier는 여전히 미구현이다. B3b0/1 gated test에만
Docker tar reader가 있고 production에 동일한 envelope parser는 없다. 그 test의 ad hoc 순회를
production의 format parser 한 곳으로 연결하여 실제 consumer를 만든다. 이것은 full graph/
byte binding이 아니라 이후 검증에 필요한 **unverified input 해석** 단계다.

1. artifact/npm/pnpm_layout_archive.go에 ParsePNPMLayoutArchive(ctx, bytes)와 immutable
   PNPMLayoutArchive/entry를 추가한다. 고정 root node_modules, 총 raw200MiB/100,000 headers,
   경로4KiB/개별 regular20MiB와 bounded known PAX path/linkpath/timestamps만 처리한다.
   Native expected owner1000:1000, low permission bits만 수용하며 symlink의 표시 mode0777을
   regular file의 실행 신뢰로 쓰지 않는다. Special/xattr/unknown PAX와 non-link의 linkname,
   duplicate/canonical escape·외부 link·file/symlink parent, missing directory parent를 거부한다.
2. 같은 immutable tar bytes에서 regular digest를 측정한다. Hardlink는 내부 regular bytes로
   닫히는 bounded alias만 해석하고 foreign/missing/cycle·64hop 초과를 거부한다. Archive의
   실제 file digest와 같다는 뜻이 아니며 이후 B3a expected inventory와 반드시 비교한다.
   Symlink의 exact expected edge/target 존재는 전체 graph 검증에 남는다.
3. Tar EOF 뒤 모든 padding이 zero인지 확인한다. Getter는 사본이며 ReadFile은 regular와
   내부 hardlink의 bytes만 반환한다. 이 type에는 raw tar 실행/설치 API, ALLOW/Verified/Valid
   의미나 source/acquired subject가 없다. Zero/missing/다른 kind ReadFile은 오류다.
4. 기존 gated shape fixture가 이 parser를 실제 소비하고 기존 bounded JSON 관측/codec 경로
   검사·A3 관찰/정리/Host 보존을 유지한다. Native output을 Host에 추출하지 않는다.

두 번째 검토에서 parser와 full verifier를 구분하지 않으면 safe path/hash 측정만으로 실제
archive/edge/bin 검증까지 닫혔다고 오인할 수 있음을 확인했다. 이 단계의 출력은 unverified
format data이고 native shim/metadata/hoist authorization을 하지 않는다. 기존 B3a tarball
scanner는 다른 package/ gzip 입력이므로 재사용해 Docker layout 형식으로 바꾸지 않는다.
Adapter에 network/Policy/Sandbox 호출을 추가하지 않고 이 새 format parser를 한 곳에서
소비한다. Full input binding/renderer/metadata 지도는 그 후 별도로 확정한다.

정상 regular/directory/symlink와 실제와 같은 cross-package hardlink, immutable getters와
잘못된 root/path/parent/link/owner/mode/size/PAX/type/duplicate/tar tail/cancel을 단위 검증한다.
Canonical format/Quick/security/docs·npm artifact 전체 Race와 fresh native gated actual을
확인한다. CI404 source는 바꾸지 않고55분 타이머19:30:18KST를 유지한다. 새 actual의 완료
bulk도 refs/query errors0 뒤 삭제한다.

### B3b2a 첫 검사와 표현 교정

첫 Quick41.334s는 default tests/actual 전에 pinned Staticcheck의 SA1019로 실패했다.
Go tar.Header.Xattrs는 deprecated이고 PAXRecords가 현재 표현이다. 기존 gomodule build
input/output owner의 PAXRecords 처리를 대조했다. Unknown PAX 거부는 이미 있어서 생산
코드의 중복 deprecated field 참조를 제거하고 xattr negative도 SCHILY.xattr.* PAX record로
만든다. Gating 도구/거부 조건은 유지하며 전체 검증과 실제 native consumer는 아직 미실행이다.

PAX 교정 뒤 Quick153.866s는 PASS였지만 security5.730s는 G110으로 실패했다. Header size와
전체 raw 상한은 검사했으나 io.Copy 호출 자체가 payload 크기를 나타내지 않았다. 기존
bounded tar 소비자의 CopyN 패턴처럼 이미 검증한 item.Size만 정확히 읽는다. 추가 permission/
limit 확대나 scanner suppress는 없다. 이 변경은 security를 먼저 확인하고 최종 Quick/
관련 Race·native actual로 검증한다. 앞 Quick을 수정 뒤 source의 성공으로 전용하지 않는다.

### B3b2a 로컬 교정 검증과 다음 경계

CopyN의 canonical security12.630s→Quick58.003s/docs.281s와 npm artifact 전체 Race6.037s는
PASS다. Fresh actual21.319s/test18.54s는 production envelope parser가 native transitive/
peer 전체 tar를 읽고 내부 hardlink와 원래 instance 경로2/4를 보존했다. 기존 controls/
identity/links·complete/금지0·Host전체동일/helperwait0/final original0도 PASS다.
이는 format input 해석의 로컬 검증이며 모든 bytes의 acquired archive binding/graph/bin/
생성 metadata 승인·dynamic/Host 지원의 완료가 아니다.

전용 runtime11files/165,567,367 logicalB는 refs0/query errors0 뒤 삭제했다.
[로컬 영수증](./p2c5b3b2a-pnpm-envelope-local-20261024.json)이 원래 두 정적 점검 거부와
교정 결과를 연결한다. CI404는 앞1f410fd/19:30:18KST 타이머를 계속 유지하고 새 parser의
원격 검증으로 전용하지 않는다. 다음은 pinned Node bin shim의 전체 code/경로 검증과
B3a inventory·frozen instances/edges를 이 envelope에 결합하는 상세 지도다.

## 9. B3b2b — Node bin shim 전체 code 검증 지도/두 번째 검토

Baseline5641ed6에는 bounded layout envelope만 있고 생성 shim 검증기는 없다.
Pinned official sh.rs의 SH_SHIM_HEADER/SH_SHIM_PLATFORM·node exec block과 relocatable.rs의
BASEDIR_ABS_PRELUDE를 독립적으로 조합했다. Data인 exact target relative path와 optional
NODE_PATH block만 정규화한3,346B는 실제 세 shim 모두와 byte-identical이며 SHA256은
64d69cfde5def2f7644e5ca92e1e669eaeb87bcbd30355f2c2cd9ac3b20bf2d7이다. 이름/주석 한 줄이
아니라 나머지 **모든 code byte**를 같은 형식으로 제한할 수 있다. Source Git blobs/원본은
동일 raw owner의 sh-renderer/shim-relocatable/linking-paths source가 소유한다.

1. artifact/npm/pnpm_bin_shim.go 한 곳에 ParsePNPMNodeBinShim(bytes, shimPath,
   expectedTarget)를 둔다. 현재 Node/no-extra-shebang-args 형식만 처리한다. 두 경로는
   node_modules 안의 canonical safe ASCII path이며 relative target은 pure slash semantics로
   계산한다. Expected target은 최종 binder가 **원본 archive manifest/bin과 검증된 file**에서
   제공해야 한다. Native marker나 file basename은 source authority가 아니다.
2. Optional NODE_PATH는 exact5line block 하나만 parse한다. 두 export의 값 일치, 최대32개
   중복 없는 literal $basedir_abs/ 상대 path, node_modules 안의 module directory로만 제한한다.
   반환 경로는 뒤 full binder가 실제 directory/정확한 허용 범위에 결합해야 한다. 외부/CWD/
   shell substitution/quotes·다른 block·program/args는 거부한다.
3. Block을 제외한 모든 bytes에서 **알려진 target의7개 위치**만 placeholder로 바꾼다.
   정규화 전체 SHA256은 위 source-derived 형식 digest와 같아야 한다. 이 값은 단일 format
   identity이고 artifact digest/Policy·CONTROL 권한이 아니다. Binary version/name 값을
   여러 소비자에 박지 않으며 version은 기존 runtime lock에서만 온다. Native 형식 변경은
   source audit·세 golden/native 회귀에서 발견하고 이 owner 하나를 갱신한다.
4. Testdata normalized vector는 MIT인 동일 PNPM source로부터 도출하며 기존
   tools/pnpm/NOTICE가 attribution/license owner다. Production binary에는 source script가
   들어가지 않고 code identity digest와 format parser만 있다. Template script는 실행하지 않는다.
5. Gated fixture의 명시적 fixture data에 expected bin target을 두고 실제 세 shim 전체를
   이 parser로 확인한다. 이는 public fixture의 알려진 path/format 검증이며 full archive/
   manifest binding 또는 lifecycle 실행 완료가 아니다. 기존 filename/marker에서 target을
   읽어 자기 자신을 승인하는 방법은 사용하지 않는다.

두 번째 검토에서 prefix hash/target 주석만 확인하거나 NODE_PATH를 임의 shell text로
받으면 injected code를 놓칠 수 있음을 확인했다. Exact export block·safe path grammar와
known target7회/나머지 전체 code byte identity를 모두 요구한다. Unknown runtime/args는
현재 parser에서 거부하고 future full inspector에서 required unavailable/incomplete로
설명해야 한다. Npm 기존 실행/관찰기/Host 권한은 그대로다. Full binder는 이 형식 확인과
별개로 원본 archive의 bin target/header·모든 payload/edge와 metadata를 검증해야 한다.

Unit negatives는 다른 expected target, missing/extra target occurrence, comment는 유지한
exec 변조·code 삽입·다른 program/args, NODE_PATH 양쪽 불일치/중복/escape/substitution,
unsafe/noncanonical path를 포함한다. Canonical format/Quick/security/docs·해당 npm Race와
fresh native actual을 확인한다. CI404의55분 타이머19:30:18KST는 유지하고 새 push는 없다.

### B3b2b 로컬 결과

Canonical format.406s/security12.107s/Quick141.274s/docs.384s와 npm artifact 전체 Race6.168s는
PASS다. Fresh actual18.382s/test15.87s는 public fixture의 own/private-hoist/dependency consumer
세 Node shim 모두를 명시적 expected target에 대해 전체 code 형식으로 확인했다. 같은
transitive/peer envelope·instance경로2/4·controls/identity/links·complete/금지0·Host불변/
helperwait0/final original0도 PASS다. Native marker를 target authority로 사용하지 않았다.

원본 archive의 bin 선언과 모든 graph/file binding을 수행한 것은 아니며 required dynamic/
Policy·Host 설치를 실행하지 않았다. 전용 runtime11files/165,592,448 logicalB는 all-UID
refs0/query errors0 뒤 삭제했다. [로컬 영수증](./p2c5b3b2b-pnpm-bin-local-20261024.json)이
Source-derived vector와 actual 범위를 연결한다. 다음에는 archive inventory/manifest bin과
전체 frozen instances/edges·generated metadata를 함께 결합할 정확한 지도를 확정한다.
CI404/1f410fd의19:30:18KST 실제 타이머는 그대로 유지한다.

## 10. CI404 종료와 현재 수정 지도 경계

[CI404](./p2c5b3-pnpm-ci404-qualification-20261024.json)는 source1f410fd/checkout31290a9의
whole tree32123575·Required10SUCCESS를55분 타이머 뒤 확인했다. 새 C++112 latch와 기존
PNPM4/Linux57·CPU281.30s/native macOS/helper5wait0/final0이 범위다. 이후 local5641ed6의
layout envelope와442af68의 Node bin code 형식 검증은 이 remote에 포함되지 않는다.

다음 full binder의 정확한 source 지도·metadata/bin/instance coverage 판정을 작성·재검토한
뒤 구현한다. 현재 static inventory, unverified native envelope, exact dirname와 Node code
format parser를 재사용하고 새 parser/Store/Domain Policy/기존 단일 Backend를 재작성하지
않는다. 아직 full archive/file/edge/bin/metadata 값 결합·production prepare/static-all/
per-instance aggregate·Host records/transaction/CLI는 MISSING이며 P2 acceptance는 NO다.
