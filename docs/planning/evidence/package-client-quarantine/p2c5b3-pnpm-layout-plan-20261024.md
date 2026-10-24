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
파일 내용 변경, symlink escape/hardlink/special file/owner·size 초과를 거부해야 한다.

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
