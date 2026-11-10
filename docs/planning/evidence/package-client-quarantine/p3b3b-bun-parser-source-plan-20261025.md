# P3-B3b Bun lock parser·독립 npm source binding 지도

기준 b658670d315a0dc234b62cdd651ad19fc973be69/tree6e100e81063996e668f285f0e52a697ce1d4741f,
2340개 tracked files 전체 index/disk/mode 일치다. M13-004 하나만 IN_PROGRESS다. B3a는 원본
observer117/actual Bun4·PNPM boundary5/public8/quarantine4·final checks로 named local
QUALIFIED다. 새 push/CI는 없으며 remote9410 CI411 macOS/Required와 전체 Bun acceptance는 열린다.

Read: B3 source 지도/고정 Rust 원본과 B3a 실제 lock samples, package-client 확장 계획§3–6,
Architecture adapters/inspection/sandbox/policy, Domain identity·dependency verified-set,
engineering dependency 규칙, lessons artifact L19·sandbox L08/L18·evidence L11/L16·ci L06.

## Baseline audit

| 책임/기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun fixed supply/query/native raw lock producer, internal/sandbox | YES | YES(private actual producer) | YES(named local raw producer) | NO(whole Bun) | backend public selection consumer·layout·CLI |
| npm strictControlJSON/project_controls.go | YES | YES(existing controls) | YES(existing scope) | YES(existing) | Bun JSONC의 bounded lexical normalization·UTF8 validation |
| Bun lock schema/instance parser | NO | NO | NO | NO | observed v2/config1, complete root/instance census와 ancestor edges |
| npm Resolver exact-version transport, pnpm_selection.go | YES | YES(PNPM) | YES(existing scope) | YES(existing) | 두 실제 consumer에서 private helper 공유·Bun exact SRI/source binding |
| Domain project snapshot/ordinary archive quarantine | YES | YES(existing) | YES(existing scope) | YES(existing) | Bun complete selection/controls/runtime fingerprint와 연결 |

## 직접 확인한 입력

Bun1.4.4/source38afee6의 fixed lock-only install은 single350B, transitive534B, peer1056B를
출력했다. 모두 lockfileVersion2/configVersion1, workspaces의 단일 empty key, packages의
[name@version, "", INFO, sha512-SRI]다. 실제 empty는 bun.lock 부재다. 원본은 raw
p3b3a-native-lock-sample-inventory.json과 p3b3a-bun-lock-v11-actual.log가 소유한다.
원본 로그의 quoted string을 decode한 표본의 전체 size/SHA를 inventory에 연결했으며 이
진단용/최종 canonical 네 출력이 완전히 같음을 별도 sample-match receipt로 확인했다.

Upstream reader supports0..3와 fresh writer2를 구분한다. 이번 parser는 관측된2/config1의
검토한 public npm subset만 허용한다. Unknown schema/config/field·alternate registry·VCS/
local/workspace/overrides/catalog/runtime provisioning은 fail closed다. Optional/bundled/
platform-specific 또는 unresolved peer는 지원했다고 표시하지 않고 명시 거부 회귀로 남긴다.
Native 출력은 unapproved selection이며 독립 source binding도 archive 검사/Policy/승격이 아니다.

## Exact edit map / 두 번째 검토

1. `internal/artifact/npm/bun_lock_json.go`: dependency를 추가하지 않는다. UTF8·4MiB 경계 뒤
   string/escape와 comment를 구분하는 bounded lexical scanner로 //, /* */ comments 및
   닫힘 직전의 실제 trailing comma만 normalize한다. 문자열 내용을 regex로 고치거나
   malformed comma를 유효 JSON으로 바꾸지 않는다. 기존 strictControlJSON의 depth32,
   token200000, duplicate-key/trailing-value 거부를 재사용한다. 그 기존 함수 의미는 바꾸지 않는다.
2. `bun_lock.go`/필요한 private wire: actual ProjectControlFile의 이름·Present와 원래 manifest를
   받는다. Empty lock 부재는 dependency-free manifest에서만 허용한다. UTF8/closed fields/
   exact2/config1/root-only constraints·동일 manifest name/dependencies/devDependencies,
   nodes4096/edges16384/roots256, public exact versions/SHA512를 검사한다. 모든 instance key는
   scoped package 단위를 구분한 canonical tree-relative name이다. Name/version metadata와
   tree location을 분리하고 package@version만으로 서로 다른 instances를 합치지 않는다.
3. 고정 Rust PkgMap::find_resolution의 own-child→ancestor→root walk를 범위 검토한다.
   Dependency/peer constraints와 실제 target name/version을 대조하며 missing/extra/unreachable
   instance·dangling/self edge는 거부한다. 일반 dependency와 peer의 upstream version-based
   재결합을 같은 것으로 가정하지 않는다. 여러 peer 후보로 native binding이 불명확하면
   단일 경로를 추측하지 않고 거부하고 후속 actual layout 검증에서 재검토한다.
4. 추가 baseline 대조에서 `Resolver.fetchMetadataBody(ctx,name,version)`가 이미 exact version의
   HTTPS/escaped endpoint/Accept·deadline·2MiB/redirect 거부를 소유함을 확인했다. 기존 PNPM
   코드는 변경하지 않고 이 owner와 resolvedVersion을 직접 재사용한다. Bun binder만 strict
   version JSON·독립 dependencies/peers/bin과 native INFO를 대조하고 optional/bundled/platform
   subset 밖 metadata를 거부한다. 같은 name@version unique archive만 fetch하며 모든 instance의
   source/SRI를 묶는다. Native URL로 registry를 재설정하지 않으며 Script flag는 frozen native
   metadata일 뿐 archive/lifecycle 검사 증거로 쓰지 않는다.
5. `bun_selection.go`/`bun_project_snapshot.go`: complete controls/runtime, unique artifacts와
   모든 root/instance/edge의 normalized fingerprint를 기존 Domain ProjectDependencySnapshot에
   연결한다. Pure parser/binder를 먼저 검증하고 실제 sandbox public selection wiring은
   별도 후속 단계다. Bun layout/cache/transaction/CLI나 새 Core protocol을 미리 만들지 않는다.
6. Tests/testdata: actual three fixtures와 explicit empty, scoped/nested conflict/peer semantics,
   comments/string escape/trailing comma/duplicate/UTF8/budgets, source/SRI/identity/registry drift,
   canceled/partial result/immutable copies, changed controls/runtime/complete snapshot을 확인한다.
   실제 HTTPS source binding과 fake transport를 구분하며 최종 affected package Race 및
   canonical Quick/security/docs를 실행한다. 작은 helper별 push/CI는 하지 않는다.

두 번째 검토: 이미 구현된 native scope/PNPM parser/layout을 재작성하지 않는다. Bun format은
PNPM v9나 npm lock으로 치환하지 않으며 peer ambiguity를 해결했다고 확대하지 않는다. Existing
stdJSON control scan·exact metadata transport·Domain snapshot을 재사용하고 Bun-specific grammar와
instance rules만 새로 구현한다. B3a 최종 기준을 위 commit/tree에 고정했다. 먼저 JSONC lexical subset을 구현·검증하고,
peer/constraint 및 새 external consumer 필요성은 lock interpretation source 전에 별도 대조한다.

## Constraint consumer 검토 / source 전 추가 대조

기존 lock의 github.com/Masterminds/semver/v3 v3.5.0을 새 npm adapter caller에서 소비한다.
새 module/version/checksum을 추가하지 않고 기존 indirect 표기만 direct 사용에 맞춘다.
기존 cache의 README/LICENSE/SECURITY/go.mod·constraint/version source와 go.sum을 대조하고
Go1.27.3/GOPROXY=off의 `go mod verify`7.539s/all modules verified를 확인했다. 해당 MIT
copyright/permission 의무를 유지한다. Frozen README는 npm/Cargo형 range를 설명하지만
모든 Bun/npm solver 의미가 같다는 주장은 하지 않는다. CHANGELOG 최상단은3.4.0이므로
3.5.0 release note나 현재 최신판을 확인했다고 쓰지 않는다. 원본 file SHA와 출력은 raw
p3b3b-semver-cache-review.json이 소유한다.

추가 exact paths: go.mod의 기존 version direct 표기, engineering dependency 검토 항목,
scripts/check/architecture.go의 npm exact import, workflow_structure_test.go의 허용·다른
owner 거부 회귀다. Core/Application/다른 adapter는 이 import를 허용하지 않는다.
Bun 선택 버전은 StrictNewVersion으로 읽고 module global coercion/IncludePrerelease를
바꾸지 않는다. Range는 길이128의 공개 constraint 문법·stable 선택 또는 exact prerelease만
먼저 지원한다. Mutable tag/비exact prerelease range는 이 단계의 명시 거부 범위이며 실제
native/source 의미를 별도로 검증하기 전 지원을 확대하지 않는다. Instance key는 길이2048,
package 단위 depth32로 묶고 scoped name을 한 단위로 센다. 여러 matching peer version의
모호함은 fail closed다. 최종 reachable vulnerability profile과 실제 비교 회귀를 수행한다.
Peer candidate 검사도 parse당16384회로 묶고 같은 name/spec 결과는 재사용한다. Tree edges의
16384회 경계와 별개로, 서로 다른 versions/constraints의 곱이 CPU 한도를 우회하지 않게 한다.
BunLock는 처음 parse한 manifest SHA와 lock digest/Present도 private 값으로 보관한다. Snapshot은
그 exact controls를 요구하고 모든 unique artifact의 identity/SRI/URL coverage를 다시 확인한다.
같은 normalized tree라는 이유만으로 바뀐 control bytes를 이전 source selection에 붙이지 않는다.
PNPM의 기존 snapshot 계약은 수정하지 않으며 Bun의 explicit absent control을 별도로 고정한다.

## 최종 검토에서 확인한 반례와 검증 경계

Generic empty `{}`/`[]` 동등 판정은 미커밋 source metadata6개/manifest8개의 잘못된 타입을
각각 SourceBound/dependency-free로 받아들였다. RED1.630/1.730s와 before source를 보존했다.
필드별 object/array/허용 false만 인정하고 정상 JSON 내부 공백을 보존하는 좁은 교정을 했다.
이 반례를 B3a/기존 PNPM 또는 배포된 Bun의 결함으로 소급하지 않는다. 최종 Quick101.552s,
npm 전체 Race50.962s·변경 없는 checker Race32.512s, vulnerability18.940s/tagged vet0.520s와
실제 공개 metadata4cases1.172s가 PASS다. 최종 staged docs/security·전체 후보 대조는 local
receipt에서 연결한다. Public selection connector/layout/검역/CLI·전체 Bun 지원은 아직 없다.
