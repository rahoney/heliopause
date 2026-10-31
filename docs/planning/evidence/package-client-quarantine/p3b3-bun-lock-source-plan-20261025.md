# P3-B3 Bun native lock와 독립 source binding 착수 지도

기준01a32b9/tree338c3eea6fc341143a6e85451948d1bcaecc8688, M13-004 하나만 IN_PROGRESS다.
P2의 remote9410 CI411은 Linux/cu132 PASS·macOS/Required FAILURE다. 같은 source를 재실행하지
않고 다음 의미 있는 candidate의 macOS/Required 성공을 기다린다. 타이머04:22:46UTC는
04:22:58에 실제 소비했고 전체10decoded logs/source를 보존했다. 새로운 CI는 아직 없다.

Read: Queue/PROJECT-DECISIONS/docs router, package-client 설계§1–5·확장 계획§2–6,
P3 baseline/B2b/B2c 계획과 결과, Architecture adapters·inspection/sandbox/policy,
Domain artifact identity·dependency verified-set·inspection contracts, engineering dependency
규칙, lessons artifact L19·sandbox L08/L18·ci L06/L17·evidence L11/L16.

## Baseline audit

| 기존 책임/근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun1.4.4 publisher/ZIP/native/single owner/maintenance/release·installed supply —378d1e1..c9a6143 | YES | YES | YES(named local) | NO(whole Bun) | Bun lock/graph/layout/CLI/전체 qualification |
| Bun private runtime/observer exact version·revision query —1738160/114latches/actual5.02s | YES | YES | YES(named local query) | NO(native install/resolver) | exact lock-only shape·project input/cache/network scope |
| PNPM native project ownership/typed registry policy/trace/cleanup — pnpm_native_scope.go | YES | YES(PNPM Select/Prepare) | YES(PNPM scopes) | YES(existing PNPM scope) | Bun 실제 consumer의 고정 settings·공통 protocol 추출 |
| PNPM bounded project controls transport / native volume owner | YES | YES | YES(existing) | NO(Bun) | Bun fixed project/lock name과 complete project control census·explicit absent empty lock |
| public npm exact metadata/Acquisition/SRI/정적/동적/Policy/Domain snapshot | YES | YES(existing npm/PNPM) | YES(existing) | NO(Bun) | Bun-specific lock/edge/instance proof를 normalized snapshot에 연결 |
| Bun lock schema parser·registry binder | NO | NO | NO | NO | 실제 native nonempty sample·정확한 JSONC schema/edge와 독립 metadata |
| Bun layout/lifecycle/transaction/CLI | NO | NO | NO | NO | 후속 B4..backend/CLI acceptance; 이번 단계 성공으로 전체 완료하지 않음 |

## 고정 upstream과 최초 실제 의미

공식 source38afee6c84fcf48c0edb2466aa4b27d287a59c30의 Rust
[src/install/lockfile/bun.lock.rs](https://github.com/oven-sh/bun/blob/38afee6c84fcf48c0edb2466aa4b27d287a59c30/src/install/lockfile/bun.lock.rs)와
[lockfile.rs](https://github.com/oven-sh/bun/blob/38afee6c84fcf48c0edb2466aa4b27d287a59c30/src/install/lockfile.rs)를
connector decoded text로 보존하고 매 chunk readback/whole SHA를 확인했다. 원본과 경로·SHA는
raw p3b3-official-source-inventory.json 한 곳이 소유한다. Host Bun 실행은 없다.

- Upstream reader는 lock versions0..3을 구분한다. CURRENT=3이라고 fresh native output이
  항상3인 것은 아니다. Writer는 loaded schema를 보존하고 fresh/no scoped overrides인
  v2-clean walk는2로 내리며 legacy 조건에서는1을 쓴다. HAA는 실제 표본 전에 schema1/3을
  단정하거나 legacy reader의 느슨한 integrity 기준을 승인 기준으로 복사하지 않는다.
- Npm row는 [name@version, registry, INFO, integrity]이며 packages key는 tree-relative
  resolution 위치다. Dependency lookup은 own key에서 ancestor/root로 올라가고 scoped
  package 경계를 따로 처리한다. PNPM peer suffix나 virtual-store 경로로 치환하지 않는다.
- 고정 Bun1.4.4 empty runc 실제 관측은 lock-only 뒤 bun.lock 부재다. Native가 삭제한 빈
  lock을 만들어 채우지 않는다. Empty는 exact original/selected manifest와 실제 control
  census·closed operation에만 연결하고 nonempty format/source proof와 구분한다.
- INFO의 optional/peer, OS/CPU/libc, hasInstallScript/bin, root trustedDependencies는 별도다.
  Native trusted list/ignore-scripts/lock-only는 HAA Policy·전체 coverage·설치 완료가 아니다.

## 정확한 단계와 두 번째 검토

1. **B3a 실제 lock producer:** 기존 PNPM withNativeProject의 lifecycle을 동일 순서·guard로
   좁게 추출하여 두 실제 native consumer(PNPM/Bun)만 고정 enum/settings로 선택한다.
   PNPM wrapper/기존 prepare seam·earlyStore/volume/owner/profile/limits는 보존한다. Bun은
   자신의 root/hash/lock/protected reader와 Node ABI를 선택한다. 임의 image/path/profile/
   network/argv를 받을 공용 plugin engine이나 미래 client placeholder는 만들지 않는다.
2. Native owner/volume 설정은 Bun의 실제 project consumer에서만 확장한다. Private project
   mount·cache·registry endpoint preflight/typed policy·trace/drain/unknown cleanup을 재사용한다.
   Host filesystem/cache/HOME/config는 반입하지 않고 Bun data controls만 복사한다.
3. B3a의 첫 observer install admission은 이미 exact1.4.4 help와 runc empty에서 확인한
   lock-only/ignore-scripts/no-env-file/fixed cache/copyfile/hoisted/registry argv의 닫힌 shape다.
   Add/update·임의 script·interpreter override·child query 권한을 미리 허용하지 않는다.
   실제 최초 read 거부가 나오면 그 source·path/flags/role/topology만 검토한다. 기존 query
   두 shape와114latches/기존 PNPM actual을 보존한다.
4. Bun controls는 fixed package.json/bun.lock를 complete bounded project tar로 읽어 optional
   lock의 실제 부재를 관측한다. Docker cp 오류 문구만 보고 부재로 만들지 않는다. Unknown
   member, extra directory, alias/hardlink, wrong UID/mode·trailing stream은 실패다. Existing
   PNPM transport는 고정 PNPM wrapper를 유지하고 필요하면 두 실제 consumer의 공통 body만
   추출한다. Native cache는 project 밖의 고정 private input root 아래에 둔다.
5. **B3b parser/source:** 실제 nonempty single/transitive/peer output을 먼저 수집한다. JSONC를
   regex로 바꾸지 않으며 표준 JSON의 strict duplicate/UTF8/number/node/edge budget과 정확한
   comment/trailing-comma subset을 검토한다. 새 dependency는 engineering 필요성·license/
   유지보수·checksum 검토 없이 추가하지 않는다. Source/client를 섞지 않고 public npm의
   기존 exact-version lookup/SRI/URL 검증을 두 실제 binder가 재사용하도록 좁게 추출한다.
   Unknown schema/registry/local/VCS/workspace/override/runtime provisioning은 거부한다.
6. Frozen source/raw lock/controls/runtime·전체 unique archive와 실제 tree/peer edges를 연결한
   뒤 기존 Domain project snapshot을 생산한다. Bun selection은 unapproved다. Layout·fresh
   quarantine/cache·transaction/CLI는 독립 acceptance로 남긴다.

## 검사와 정리 경계

기존 구현을 재작성하지 않고 MISSING만 구현한다. Mechanical extraction은 이전 consumer의
조건/정리 순서 normalized equality와 기존 회귀를 확인한다. Canonical format/Quick/security/
docs·영향받은 전체 Race·CPP 모든 latch·실제 Bun/PNPM native source/cleanup을 source에
연결한다. 작은 helper별 CI를 만들지 않고 backend/end-to-end 묶음 또는 공통 premise를 막는
확정 회귀 경계에서만 push한다. 원격 CI는 실제 시간 기반 timer를 active turn에서 소비한다.

완료한 C1 namespace copies1,252,529,078B와 C2 query/ENOSPC copies·superseded observers
886,851,721B는 exact inventory/hash·Root 전체 process/FD/maps/mount 참조0/query errors0 뒤
제거했다. Final observer-v3·active signed archives/native candidate·SDK/quality/source는 유지한다.
디스크 정리 또는 tool-store duplicate 해제를 process RSS 회복으로 표시하지 않는다.

B3a 세부 구현 재검토: native scope의 기존 PNPM prepare seam은 wrapper에 유지하고 공통
private introducer interface에 compiled kind를 확인한다. PNPM은 기존 project constructors와
create-arguments를 그대로 호출하며 Bun만 별도 root/hash·128MiB data volume을 쓴다. Bun
초기 cache/TMPDIR은 고정 private root 아래이며 project는 executable=false다. Core나 외부
caller가 settings/identity를 선택할 API는 없다. PNPM의2분/정리 순서/typed network·earlyStore는
변하지 않는다. 기존 transport의 input tar body만 두 fixed wrapper에서 공유하고 Bun optional
output lock은 complete project census로 별도 관측한다.

## B3a 직접 관측으로 좁힌 경계

실제 constructor는 manual attested fixture와 달리 input volume root whitelist와 writable
volume close whitelist를 모두 소비한다. Bun root를 두 책임에 정확히 등록하고 nonexecutable/
unknown-safe owner·identity/attached-volume 거부는 유지했다. 두 최초 거부와 constructor 회귀,
complete/query/identity/attached/remove-failure 회귀 및 actual의 정상 정리가 별도 근거다.

고정 project 명령의 실제 read 거부는 cpu online → own cgroup → UTC zone → host.conf →
resolv.conf → nsswitch.conf → hosts → gai.conf 순서였다. 각 원본의 flags557056와 system 또는
immutable OCI anchor·direct consumed Bun CONTROL을 대조했다. 해당 project profile의 exact
경로만 보완했고 기존 query는 overcommit32768/own maps557056만 유지했다. Clone/ARTIFACT/
foreign image·PID/profile·쓰기·alias·다른 mount와 pending role transition은 계속 거부한다.
FNV는 raw locator 식별이며 인증/role 근거가 아니다. 빈 명령 성공을 nonempty 지원으로 전용하지
않고 마지막 full four-case에서 실제 lock bytes와 complete census/observation/cleanup을 확인한다.

원본 release builder249.527s의117latches와 canonical observer의 actual four cases27.09s가
PASS다. Actual manifest는 불변이고 empty는 lock 부재, single/transitive/peer의 lock은 각각
350/534/1056B·schema2/config1이었다. 진단용 v11 표본과 canonical output이 같은지는 최종
local receipt에서 별도 대조한다. Shared PNPM mechanical scope/input equality와 기존 실제
consumer 결과까지 연결한 경우에만 B3a named local qualification으로 표시한다.
