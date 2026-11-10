# P3-B4b Bun hoisted layout 전체 archive 결합 지도

기준 d72287e0d3972467cb4633dbe84594ea894cd1c5/tree554294a92ea3ff7d2447786dfeec9506a30a1798,
2374개 전체 후보다. Read: B4 baseline/Read와 B4a 영수증, 기존 PNPM layout archive/inspection
budget/file closure/bin data, Bun lock tree/source/context, npm ArchiveInventory·실제 static producer.
M13-004 하나만 IN_PROGRESS이며 원격 push/CI는 없다.

## Baseline audit와 실제 표현

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun frozen context/acquired/static source, d72287e | YES | YES(named actual) | YES(named local) | YES(context subset) | native layout 결합 |
| bounded tar parser/whole inventory·budget/bin metadata, 기존 npm/PNPM | YES | YES(existing consumers) | YES(existing scope) | YES(existing scope) | client-neutral data만 재사용 |
| Bun hoisted physical file/bin/edge closure | NO | NO | NO | NO | 본 단계의 exact binder·공격/누락/반복 instance 회귀 |
| production offline prepare/lifecycle/Policy | NO | NO | NO | NO | 다음 단계; runc diagnostic을 대신 쓰지 않음 |

Network-none fixed native의 public frozen transitive/peer install은 각각1.835s/1.813s exit0다.
원본 controls와 exact container 삭제/부재를 확인했다. 전체 native regular files8/51은 lock의
SHA512로 검증한 archive entries와 digest/size가 모두 같다. Mode는 HAA가022를 제거해 준비한
private cache view와 일치한다. 아래 최초 actual 교정이 원본 mode 동일성 해석을 정정한다. Peer에는 정확히
`node_modules/.bin/loose-envify -> ../loose-envify/cli.js` 하나가 추가되고 hidden PNPM store나
generated shim은 없다. Raw p3b4b-bun-{transitive,peer}-layout-runc-v2가 원본을 소유한다.
앞 v1은 relative parent의 dirname이 empty로 끝나는 준비 루프 오류였다. 컨테이너 생성 전에
exact argv의 작업 프로세스2개만 SIGTERM143으로 종료했다. Original source·원인·부분 archive는
operator-v1-assessment와 원본에 보존한다. 제품/native 또는 역사적 메모리 부족 원인이 아니다.

## Exact edit map

1. `bun_layout_archive.go`: 기존 private bounded tree tar parser를 fixed node_modules/owner1000
   wrapper로 재사용한다. Bun의 unverified value와 immutable entries/file access만 추가한다.
   PNPM의 virtual-store·metadata·shim 의미를 Bun으로 복사하지 않는다. 기존 parser를 교체하지 않는다.
2. `pnpm_inspection_budget.go`와 `npm_project_inventory.go`: 현재 PNPM budget의 실제 charge/Add/
   Complete body는 유지하고 artifact와 instance-package 목록만 받는 private constructor로 좁게
   추출한다. 기존 공개 PNPM budget type은 alias로 보존하며 실제 Bun consumer도 같은 engine과
  128MiB 한도·entry bound를 쓴다. Complete exact inventory indexing도 기계 재사용한다.
3. `bun_inspection_budget.go`: frozen Bun context의 모든 instance를 같은 unique archive에 대응하고
   compressed+unique cache+모든 logical copy를 기존 budget으로 청구한다. 미측정/중복/foreign/
   oversized/불완전 inventory는 native 준비 전에 거부한다.
4. `bun_inspection_layout.go`·`bun_layout_binding_data.go`: fixed hoisted tree key를 scoped package
   단위로 물리 node_modules 경로에 대응한다. Complete original inventory의 모든 file/directory와
   exact dependency/peer resolution·source bin metadata를 대조한다. Regular bytes/size와
   UID/GID1000과 원본 packing mode에서022만 제거한 private cache mode를 요구하며 bin은 해당
   sibling modules/.bin의 정확한 relative link만 허용한다.
   Unknown/generated file, substituted/missing/extra file·edge, unsafe/foreign link와 collision은 거부한다.
   초기 copyfile lane의 hardlink/추가 mode 변환은 actual 근거 전 허용하지 않는다. Root scripts는
   Artifact session에 넣지 않는 exact dependency-only manifest를 생성한다. Acquired archive의
   lifecycle scripts는 보존하고 다음 단계에서 독립 검사하며 native script flag로 완료를 추론하지 않는다.
5. Digest는 schema/graph/runtime·acquired subject/run identity·모든 entry의 명시 string digest·
   manifest를 결합한다. Private ContentDigest를 JSON의 {}로 직렬화하지 않는다. Targets/MatchesInput/
   getters는 exact acquired run과 전체 frozen context를 확인한다. Tar order 변화는 fingerprint를
   바꾸지 않으며 같은 길이 다른 bytes/link/runtime/subject는 바꾼다.
6. Tests: 실제 표본에 대응하는 unit layout, scoped nested/repeated archive·peer/bin, missing/extra/
   modified bytes/mode/owner/link/metadata·wrong run/context·budget·cancel/zero values와 기존 PNPM
   전체 회귀를 확인한다. Named actual source+native raw layout은 명시한 입력/production binder로
   대조한다. Default tests의 숨은 외부 corpus 의존성은 만들지 않는다.

두 번째 검토: 이미 구현된 tar parser/budget/정적 archive 검사기는 다시 구현하지 않는다.
새 engine은 Core/Policy/daemon/임의 path/flag를 추가하지 않는다. 구조·byte 결합 성공은
dynamic/lifecycle/Policy·promotion 권한이 아니며 B4 전체 acceptance는 계속 NO다.

## 첫 자동 검사와 actual consumer 재검토

첫 canonical Quick49.690s는 production build 뒤 test build의 unused domain import로 중단됐다.
새 test import를 제거하며 원본 test와 로그는 p3b4b-first-layout-test/first-quick에 보존한다.
이 결과는 unit/staticcheck/actual binder 성공이 아니다.

실제 consumer는 기존 `bun_inspection_source_real_test.go`의 source/acquire/SRI/static 전체 입력
준비를 test-only helper로 기계 이동해 두 실제 tests가 재사용한다. 새
`bun_inspection_layout_real_test.go`는 명시적 HELOX_BUN_LAYOUT_CORPUS의 frozen runc 원본만 읽는다.
기존 bounded project tar parser로 전체 census와 원본 controls를 대조한 뒤 node_modules 부분을
메모리 안의 고정 tar로 표현하고 production Bun parser/binder에 전달한다. Host 추출/링크 추적은
없으며 unknown root entry는 제외하지 않고 거부한다. Actual source의 fresh measured inventories와
모든 target을 확인하고 partial inventory/bin substitution을 거부한다. Default tests의 외부 입력이나
CI 새 job은 추가하지 않으며 runc 자료를 production gVisor/dynamic/lifecycle 성공으로 확대하지 않는다.

첫 reviewed Quick58.966s는 SA1012의 nil Context test에서 default tests 전에 멈췄다.
금지한 호출을 제거하고 production nil guard는 유지하며 analyzer를 suppress하지 않는다.
Actual1.322s는 transitive2/11entries PASS 뒤 peer를 거부했다. 원본 js-tokens5개는0664,
diagnostic 입력 writer가 `mode &0755`로 준비한 cache/native는0644였다. 처음 binder가 원본
packing mode를 그대로 요구한 것이 이 실제 입력과 맞지 않았다. 원본 inventory/자료는 보존하고
source mode에서022만 제거한 exact private data view를 대조한다. Writable native regular file,
추가 exec/mode 변경·bytes 변경은 계속 거부한다. 이는 HAA의 cache 준비 규칙이며 Bun extract의
자동 chmod가 검증됐다는 뜻이 아니다. First actual/mode assessment가5개 delta와 반증을 소유한다.

## 교정 후보의 named local 결과

Canonical Quick188.001s·npm/sandbox 전체 Race47.291s와 corrected actual1.422s가 PASS다.
Actual은 fresh independent acquire/SRI/static inventories와 frozen runc native transitive/peer의
entries11/63·모든 instance target을 결합하고 partial inventory와 bin substitution을 거부했다.
Source fixture 준비 body는 inventory retention 추가만 제외하면 B4a와 기계 동일하다. Shared
PNPM inventory/constructor body·charge/Add/Complete도 기존 의미를 유지한다. 각 mechanical
receipt가 원본/normalized SHA를 소유한다. Production gVisor/동적 lifecycle/Policy는 미실행이다.

완료한 B3c namespace3·source binary4의778,244,261 regular bytes는 exact hash/inode/owner와
전체 process/mount/FD 참조0·query errors0 뒤 제거했다. Active corrected test·source/SDK/quality/
native candidate/standalone canonical observer/diagnostic build cache와 원본은 보존했다. 이는
디스크 정리이며 process RSS 회복 증거가 아니다. completed-copies raw가 정확한 inventory를 소유한다.
