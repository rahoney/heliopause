# P3-B4c Bun production offline prepare 착수 지도

기준8ffb0c909d3e1f350c5b77eea59023d60bf2f278/tree3f6fdff65eaf9da7262ca7ab81984e7f90d44baf,
2384개 전체 후보다. M13-004 하나만 IN_PROGRESS다. Read: B4/B4b 지도·영수증과 라우팅된
Architecture/Domain/lessons, nativeProjectScope/controls·PNPM prepare/intake transport,
고정 Bun PackageManagerDirectories/extract_tarball source와 앞 file-cache 실패 원본.

## Baseline audit

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun context·whole layout/static source binder, d72287e/8ffb0c9 | YES | YES(named actual) | YES(named local subset) | YES(structural subset) | production prepare output 연결 |
| controlled intake byte freeze/native scope/cleanup/trace/PNPM offline 준비 | YES | YES(existing consumers) | YES(existing scope) | YES(existing scope) | Bun fixed native command와 measured cache 이동 |
| Bun production cache 준비·offline install | NO | NO | NO | NO | 본 단계의 native primitive→exact map→실제 gVisor 관찰·경계 |
| Bun lifecycle/Artifact runtime·Policy | NO | NO | NO | NO | native script-free 준비 뒤 별도 실행/coverage 단계 |

## 구현 전 확인할 file seed primitive와 두 번째 검토

앞 single file archive seed는 성공했지만 public registry cache key와 달랐다. 고정 source의
`cached_tarball_folder_name_print`는 SHA512의 앞16bytes를 포함한 @T@…@@@1 경로를 사용한다.
이 값을 source approval로 사용하지 않는다. 같은 source archive의 전체 bytes는 독립 SRI/static
검증을 유지한다.

1. 앞 actual transitive/peer archive를 원래 lock SRI로 다시 대조하고 network-none runc의 private
   archives data에 전달한다. Synthetic seed manifest는 이 frozen unique archives의 file dependency만
   담으며 public project controls는 별도 project에 원본 그대로 보존한다.
2. 고정 Bun의 `install --offline --ignore-scripts --no-env-file`과 같은 fixed cache/copyfile/hoisted
   flags로 seed를 읽는다. Artifact/root scripts와 Host Bun은 실행하지 않는다. 모든 source file은
   native Bun이 격리 안에서 펼치며 Host filesystem에 package를 추출하지 않는다.
3. Bounded 전체 native cache data·marker/alias 표현과 원본 controls·exact container cleanup을
   관측한다. Multi-archive file dependency/peer 해석과 원본 regular modes를 추측하지 않는다.
   Native .bun-tag는 데이터이며 완료·source·Policy authority가 아니다.
4. 결과에 따라 기존 bounded tree parser/inventory를 재사용하는 native cache entry 검증과
   registry cache data envelope의 exact edit map을 보완한다. Whole archive 검사를 다시 구현하거나
   Inspection/Sandbox import direction을 바꾸지 않는다. Unknown/extra source bytes와 marker
   collision은 거부하고 original packing facts·원본은 유지한다.

이 단계는 production source 작성 전 native 표현 조사다. 성공하더라도 gVisor/lifecycle/Policy·
Host install 또는 whole Bun qualification으로 쓰지 않는다. 기존 PNPM 구현·runtime permission은
실제 필요와 회귀 없이 변경하지 않는다. 새 push/CI는 없고 backend checkpoint에서 묶는다.

## 첫 file seed와 다음 primitive 검토

첫 transitive/peer seed는2.109/2.025s exit1이다. Native cache에 모든 file archive가 나타났어도
자식의 registry dependencies를 offline에서 해석하지 못했다. Failed output을 성공 단계로
채택하지 않는다. 원본 p3b4c-bun-file-seed-*-runc-v1의 명령/오류·controls/cleanup을 보존한다.
관측된 .bun-tag는95bytes의 원래 SHA512 SRI 문자열이며 authority가 아니다.

공식 [override 문서](https://bun.sh/docs/pm/overrides)의 version-scoped key는 declared semver range에
적용되며 file/URL 입력에는 적용되지 않는다. 다음 diagnostic은 `name@* -> 동일 frozen file archive`
규칙으로 synthetic seed의 registry 자식 edge만 file 입력에 연결한다. 원본 public manifest/lock은
별도 project에서 보존하고 이 recipe나 생성한 lock을 public graph/source/Policy로 사용하지 않는다.
Native script-free cache import라는 목적만 검증하며 exit0와 전체 source bytes·정리 전에는 완료가
아니다. Scoped override의 private lock v3가 생겨도 public v2 parser 지원을 넓혔다고 쓰지 않는다.

Scoped file seed v2는 transitive2.061s/peer2.200s exit0와 원본 controls·exact container 부재를
확인했다. 다음은 lock-only seed로 node_modules 생성을 피하고 native cache의 모든 원본 file·
SRI marker를 측정 inventory와 대조한 뒤 marker를 제외한 registry cache envelope를 준비한다.
같은 컨테이너의 원래 public controls로 offline frozen install을 이어 실제 전체 흐름을 확인한다.
어떤 seed recipe/lock도 public graph로 채택하지 않는다. Intermediate file cache와 registry cache가
동시에 존재하므로 production 예산은 두 unique expanded copies와 최종 logical copies를 실제로
청구해야 한다. 기존 PNPM 한-store 예산이나 전체128MiB volume ceiling을 바꾸지 않는다.

## Native prototype 결과와 production exact map

Lock-only seed→전체 native cache file/marker 대조→registry envelope→public frozen install v3는
transitive2.081s/peer2.015s에서 두 명령 모두 exit0다. Seed에는 package.json/bun.lock만 있고
node_modules는 없다. Digest alias root를 사용한 v4도2.512/2.482s의 두 명령 exit0이며 모든
source file8/51과 원본 public controls·exact container 부재를 확인했다. Alias는 서로 다른
source bytes의 file root가 덮이지 않게 하는 내부 이름이며 public graph가 아니다. 이 두 표본의
성공을 아직 실행하지 않은 여러 version/순환 graph 전체 지원으로 확대하지 않는다.

1. `internal/artifact/npm/bun_native_cache.go`: frozen input의 exact acquired membership와 SRI에서
   native @T@ root를 계산하고 기존 bounded tree parser로 한 native cache tar를 읽는다. Original
   measured inventory의 전체 file/directory/owner/digest/size와 mode view를 대조한다. 정확한
   root .bun-tag SRI95bytes는 native metadata로 accounting 후 제거하며 원본 filename 충돌,
   unknown/duplicate/alias/변조는 거부한다. Partial SRI prefix는 locator일 뿐 인증이 아니다.
2. `bun_registry_cache.go`: complete bound cache entries만 fixed name@version@@@1 data envelope로
   표현한다. Marker·seed controls는 넣지 않고 original source mode에서022만 제거한다. Original
   source bytes와 packing facts는 유지한다. 초기 registry cache 이름의 stable version 범위 밖은
   native/DNS 전에 명시 거부한다. Host filesystem 추출이나 Inspector import는 추가하지 않는다.
3. `bun_inspection_budget.go`: 기존 shared engine을 유지하는 얇은 Bun wrapper가 extra unique
   expanded cache copy 한 개를 추가 청구한다. Core charge/Add/Complete와 PNPM 한-copy 의미는
   그대로다. Source compressed+두 cache copies+모든 logical placements·실패 sticky 회귀를 확인한다.
4. `internal/sandbox/pnpm_inspection_transport.go`: 기존 freeze body를 private artifact-list helper로
   기계 추출하고 원래 PNPM input guard를 유지한다. Bun wrapper가 같은 controlled intake open/
   digest/size/close guard를 재사용한다. Native seed data는 fixed artifacts/seed directories와
   digest alias file dependencies·version-scoped override만 생성한다. Original project는 별도다.
5. `bun_inspection_prepare.go`·`bun_inspection_cache_transport.go`: preflight/budget/whole byte freeze
   뒤 기존 native scope를 사용하고 command 전에 기존 disconnect+empty network proof를 재사용한다.
   Fixed lock-only offline seed→각 native cache bounded read/bind→complete registry envelope→fixed
   public offline frozen install→whole project census/원본 controls/whole layout binder를 연결한다.
   Whole project의 unknown entries를 제외해서 통과시키지 않는다. Scope의 command/trace/cleanup/
   cancellation 오류 뒤 bound output은 버리고 원래 cause를 보존한다.
6. Observer의 기존 exact Bun project predicate에 seed cwd의 lock-only+offline14args와 public cwd의
   offline+frozen14args만 추가한다. Registry/cache/linker/ignore-scripts/no-env, protected image,
   direct consumed CONTROL/nonce/profile/topology/creator guard는 유지한다. FS/network grant는
   actual 최초 거부 전 추가하지 않는다. Existing117 latches의 두 shape/negative·원본 builder와
   actual 소비자를 검증한다. 개발 반복에는 identity를 확인한 diagnostic cache를 재사용한다.
7. Tests: complete/partial/foreign/cache marker tamper/source collision/unsafe role·mode·alias와
   budget extra copy, zero/cancel/cause, frozen controls/unknown output·command/cleanup/trace 오류의
   output 폐기, 기존 PNPM 전체 회귀를 확인한다. Actual은 empty skip과 single/transitive/peer 및
   실제 여러 version source를 포함해 template 범위를 검증한다. 미실행 graph는 미완료로 남긴다.

두 번째 검토: bounded archive parser·static producer·resource owner를 교체하거나 새 Core/Policy/
daemon·범용 실행 API를 만들지 않는다. Native Bun이 source tar를 격리 안에서 펼치고 HAA는
기존 data parser로 그 결과를 원본과 대조한다. Internal seed의 graph/lock/marker는 approval
authority가 아니며 public controls와 final whole layout을 독립 대조한다. 동적 lifecycle/Policy와
cache/transaction/CLI·remote qualification은 이 준비 성공 뒤에도 별도 acceptance다.

동일 runtime/client를 재확인하는 context getter도 추가해 preparer가 generated Bun pin과 대조한다.
Data accessor 추가이며 opaque context의 생성·권한 의미는 바꾸지 않는다.

현재 B4c-a의 독립 검증 단위는 cache data binding/registry envelope·두-copy budget과 그 actual
frozen-cache consumer다. 기존 source/acquire/static helper를 재사용하고 Source corpus는 명시적
HELOX_BUN_CACHE_CORPUS로만 받는다. Native lookup index는 반환하지 않고 각 typed Source cache
member 전체만 원본과 대조한다. Global cache를 검증했다고 쓰지 않는다. Implied directory 구조도
기존100000-entry ceiling으로 native 전에 제한한다. Production preparer/observer 작업은 이 단위
뒤 같은 M13-004에서 계속하며 새 CI나 Runtime 권한은 이 data 단위에 추가하지 않는다.
