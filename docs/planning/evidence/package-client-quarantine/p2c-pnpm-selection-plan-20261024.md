# P2c — pnpm selection·graph·layout 연결 지도

Baseline `343f03f`, M13-004/P2만 진행한다. P2b2의 native admission은
[LOCAL VERIFIED](./p2b2-pnpm-admission-local-20261024.json)이며 같은 source의 CI401
`40054664786`이 실행 중이다. 첫 확인은 2026-10-24 12:23:29 KST이며 실제 deadline
timer를 실행했다. 이 문서는 그 원격 결과에 의존하지 않는 parser/설계 작업부터 수행한다.
새 push로 실행 중 후보를 취소하지 않는다. Bun 이후 단계는 시작하지 않는다.

Read: [전체 P2 감사](./p2-pnpm-baseline-20261024.md),
[확장 계약](../../../user-journey-cli-ia/09-package-client-quarantine.md),
[검증 주기](../../20-package-client-quarantine-plan.md#5-검증-matrix와-자원-정리),
[Architecture](../../../architecture/01-foundation-and-dependencies.md),
[Artifact Port](../../../domain-model/07-contract-artifact-port.md),
[D-012/D-015](../../../threat-model/04-trusted-tooling-and-evidence.md),
[코딩 규칙](../../../engineering/02-coding-security-rules.md),
오답노트 artifact L15/L20, sandbox L08/L12/L18, evidence L11/L16, ci L05/L06.

## 1. 소스 감사 — 기존 구현과 실제 gap

| 책임 / 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| npm public registry Resolve/Acquire·SRI — artifact/npm/resolver.go | YES | YES | YES(기존 npm) | YES(기존 npm) | PNPM exact package/version/SRI와 독립 비교 |
| npm package-lock JSON·single-root — npm/lockfile.go, project_controls.go | YES | YES | YES(기존 npm) | YES(기존 npm) | PNPM v9 YAML·단일 importer/복수 root·peer instance를 그대로 표현 |
| generic opaque node/edge·record path — domain/install.go | YES | YES | YES(기존 graph) | YES(기존 graph) | full PNPM snapshot key에서 NodeID/edge를 만들기; Core 재작성 없음 |
| root 없는 complete snapshot — domain/project_snapshot.go | YES | YES | YES(기존 Go/UV) | YES(기존 Go/UV) | unique artifact set와 PNPM 전체 graph/control digest를 함께 결합 |
| native Root/Node query·named project mount — pnpm_runtime.go, observer.cc | YES | YES(offline) | NO(원격 진행) | NO | 실제 public selection·typed egress·고정 command/control 수집 |
| 기존 NPM guarded transaction — promotion/npm_transaction.go, npm_update.go | YES | YES | YES(기존 npm) | YES(기존 npm) | PNPM control names·정규화된 내부 link/layout·update semantics |
| CLI compiled client descriptor — cli/package_client.go | YES | YES(현재 Python families) | YES(기존 범위) | YES(기존 범위) | PNPM add/install/update의 정확한 route/factory/도움말과 actual |
| PNPM public selection부터 CLI까지 | NO | NO | NO | NO | 아래 순서 전체 |

`LockedDependencyGraph`는 NodeID 중복만 금지하므로 같은 tarball identity의 peer별 노드를
보존할 수 있다. 다만 정확히 하나의 primary와 그 노드에서의 연결성을 요구한다.
복수 직접 root의 complete project에는 가짜 primary를 만들지 않는다.
`ProjectDependencySnapshot`은 artifact identity를 deduplicate한 목록과 별도 graph digest를
보유한다. PNPM adapter가 전체 instance graph/controls를 유지하고 digest에 결합해야 한다.
기존 npm parser/transaction의 전역 symlink·schema 거부를 PNPM 때문에 완화하지 않는다.

## 2. 고정 source와 이미 관측한 형식

PNPM12.15.0 source `91774e72a04d52ce61db06339696a433d383c764`를 유지한다.
새 latest 선택이나 pin 교체가 아니다. 다음 공식 원본을 정확한 commit으로 대조했다.

- [Lockfile/settings](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/lockfile/src/lib.rs)
- [Package metadata](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/lockfile/src/package_metadata.rs)
- [Snapshot instance](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/lockfile/src/snapshot_entry.rs)
- [Importer](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/lockfile/src/project_snapshot.rs)
- [Resolution/resource vs tar integrity](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/lockfile/src/resolution.rs)
- [Importer dependency reference](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/lockfile/src/resolved_dependency.rs)
- [Peer-qualified key](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/lockfile/src/pkg_name_ver_peer.rs)

P2b1의 actual runc19개는 lock9.0, packages의 per-version metadata와 snapshots의
peer별 wiring, importers의 specifier/version 쌍을 확인했다. Loopback mirror가 tarball
URL을 바꾼 형식 관측이므로 그 HTTP URL을 production parser에 허용하지 않는다.
Raw 원본은 `pnpm-native-observation-*-controls.json`, 공식 원본 capture는
`p2c-pnpm-upstream-lockfile-*`이다. 새 sanitized fixture는 synthetic임을 표시하며
원본을 수정하거나 실제 public gVisor 성공으로 소급하지 않는다.

## 3. 현재 실행 단위 P2c1 — bounded unapproved lock와 source binding

첫 변경 위치는 `internal/artifact/npm/pnpm_lock.go`와 같은 owner의 schema/검증·test다.
기존 pinned `go.yaml.in/yaml/v3`를 사용하고 새 dependency/외부 parser/CLI placeholder를
추가하지 않는다. 기존 name/version/SHA512-SRI/public URL·strict JSON primitive를 재사용한다.

`ParsePNPMLock(manifest, lock, expectedClientVersion)`는 아직 승인되지 않은 adapter-local selection을 반환한다.
package metadata와 full snapshot key/실제 edges, 직접 root group/name/specifier를 구분한다.
Public 반환값은 copy로 제공하여 호출자가 원래 selection을 변경하지 못하게 한다.
Version/name만 같다고 peer instance를 합치지 않는다. Peer suffix 자체는 검증/권한 근거가
아니며 실제 연결 대상은 정확히 snapshots에 존재해야 한다.

입력과 첫 capability 범위:

- manifest/lock 각각4MiB, YAML document1개, bounded node/depth/token·graph4096/edge16384.
  duplicate key, alias/anchor/merge/custom tag, trailing document, unknown field를 거부한다.
- lock9.0, importer 정확히 `.` 하나. manifest의 prod/dev specifier와 importer가 정확히 같아야
  한다. 복수 root와 peer instance는 보존하고 empty graph도 명시 empty 상태로 구분한다.
- metadata의 exact public name/version, canonical SHA512 resource SRI가 필요하다.
  생략 가능한 registry URL은 독립 npm Resolve에서 얻는다. 명시 URL도 public HTTPS와
  같은 resolved URL/SRI여야 하며 문자열로 URL을 조립하지 않는다.
- settings의 resource integrity만 지원한다. tar integrityTarget/revision, aliases·local/VCS/
  workspace/catalog/patch/custom resolution과 hooks/외부 runtime/client 자동 선택은 거부한다.
  첫 slice의 platform/optional/bundled 해석 미완료도 fail-closed이며 전체 P2 완료로 줄이지 않는다.
- engines/deprecated/hasBin 등 실제 비선택 metadata는 형식·크기를 확인해 보존한다.
  lifecycle/HostInstallAction은 lock의 필드 부재로 안전하다고 결정하지 않는다. 후속 실제
  archive inspection과 promotion의 ignore-scripts/입력 검증을 계속 요구한다.
- 모든 snapshot/metadata는 roots에서 reachable해야 한다. dangling/unreachable/alias/서로
  다른 metadata pairing을 거부한다. 동일 tarball의 peer별 instance는 유지한다.

다음 binding 단위는 같은 owner의 `pnpm_selection.go`의 `Resolver.BindPNPMLock`에서 같은 bounded public registry transport와 identity 검증을
exact name@version으로 호출하여 native SRI와 metadata를 독립 비교한다. Native lock의
선택 출력은 Evidence/ALLOW가 아니다. 기존 Acquire·integrity/static/dynamic·entry/set
Policy를 모두 거친 뒤에만 verified set을 만든다. 새 Core interface는 필요하지 않다.

P2c1 검증: 정상 empty/single/transitive/multiple-root/scoped/peer 두 instance, 순서 변화의
동일 selection, getters mutation, manifest mismatch와 duplicate/anchor/flood/trailing/unknown,
missing/malformed SRI·HTTP/foreign URL·revision/tar hash·alias/unsupported fields,
missing snapshot/edge/metadata·unreachable·peer-context 충돌. Binding은 독립 metadata의
name/version/SRI/URL mismatch·누락·취소를 거부한다. 관련 전체 package Race와 canonical
format/quick/security/docs를 적용하며 parser commit만으로 remote full CI를 반복하지 않는다.

## 4. P2c2..4의 정확한 연결 책임과 검증 경계

1. **Public runtime producer:** `sandbox/pnpm_runtime.go`의 fixed Node/native/project volumes와
   기존 typed NPM resolver-policy service의 실제 owner를 재사용한다. 고정 controls만 전달하고
   clean HOME/config·registry·ignore-scripts·runtime download 금지·source drift/수집 제한을
   대조한다. 실제 gVisor first fault만 근거로 observer를 보완하며 CONTROL은 child에게 상속하지
   않는다. Offline7 성공을 public DNS/TLS/registry 성공으로 전용하지 않는다.
2. **Source/graph:** `artifact/npm`이 full peer wiring과 canonical constraints/source identity를
   freeze한다. single requested artifact와 complete project snapshot을 올바른 기존 Port로
   연결한다. Install/add/update의 manifest specifier 변화는 native 결과와 각각 대조한다.
3. **Promotion/layout:** 기존 transaction guard/freeze/commit/rollback을 재사용하되 PNPM
   own controls/metadata를 별도 소유한다. 같은 pinned PNPM의 offline ignore-scripts 출력만
   받으며 `.pnpm` 내부 package/peer/bin link inventory를 모두 graph에 결합해 검증한다.
   임의 symlink·Host hardlink/escape·외부 store·추가 artifact·사용자 파일 overwrite를 거부한다.
   전역 link 허용이나 tarball 이름으로 layout을 추측하지 않는다. Native script/build가 필요한
   package는 capability가 닫히기 전 실행/설치를 완료했다고 표시하지 않는다.
4. **CLI/factory:** `cli/package_client.go` compiled descriptor와 bootstrap의 source/client binding을
   같은 실제 PNPM operations에 연결한다. npm registry SourceID를 새 client SourceID로 바꾸지
   않는다. help/examples·한국어/영어 설명·JSON/human/실제 failure를 같은 지원표와 대조한다.
   미구현 명령/후속 Bun·일회성 실행기의 placeholder를 만들지 않는다.

Public single/transitive/peer/multiple root, native add/install/update/no-op와 managed mutation·
concurrent change·interrupt/rollback·link attacks를 검증한 뒤 영향받은 legacy npm/PyPI 및
CPU/CUDA/native macOS·Required를 최종 P2 후보에서 묶는다. 완료 payload는 마지막 소비와
참조 부재 확인 뒤 삭제하고 원본 로그와 작은 normalized evidence만 남긴다.

## 5. source 작성 전 두 번째 검토

현재 기존 구현을 대체하거나 invariant를 바꿀 이유는 없다. npm JSON을 YAML로 속이거나
flat version 목록을 peer graph로 간주하지 않는다. Root 없는 project와 primary graph의
계약 차이를 유지한다. Generic Core에 PNPM wire schema·link/API를 넣지 않는다.
추가 parser/binder는 현재 P2의 실제 gap에만 사용하며 기존 parser/installer 재작성은 없다.

P2c1에서 unknown/optional/platform 등을 거부하는 것은 부분 capability 판정이다. 이것을
전체 PNPM 지원 완료로 선언하거나 남은 acceptance를 삭제하지 않는다. Native form은
upstream exact commit과 실제 fixture가 동시에 기준이며 version별 상수를 코드 곳곳에
복제하지 않는다. CI401의 최종 결과와 기존 소비자 qualification은 독립적으로 닫는다.

첫 Quick16.190s는 실제 architecture gate에서 새 artifact/npm→runtimeidentity coupling과
미검토 YAML consumer를 차단했다. Version은 producer가 generated pin으로 전달하도록
바꾸어 기존 내부 dependency 경계를 보존한다. YAML은 현재 lock의 v3.0.4를 npm adapter
한 곳에서만 추가 소비하도록 필요성·local source/license/module graph를 검토한다.
Go1.16 floor·MIT/Apache license, official YAML organization fork라는 해당 version README를
확인했다. go.mod/go.sum/module 선택이나 전이 dependency는 변경하지 않는다.
Canonical engineering review와 exact npm external-import edge/다른 consumer 거부 회귀를
연결한다. 이는 Core/Application dependency invariant 변경이나 모든 adapter YAML 허용이
아니며 새 production 호출의 reachable vulnerability·배포 고지는 최종 P2에서 확인한다.

Binder는 public registry 한 origin과 exact native name/version/SRI/명시 URL을 모두
비교한다. 선택 전체에는2분 상한과 caller cancellation을 적용하고 부분 결과를 반환하지
않는다. 반환 selection은 native instance graph와 unique source-bound tarball 목록을 같이
보유하지만 아직 Acquire/검역·HostInstallAction·Domain graph/Policy/Promotion 연결은 아니다.
특히 prepare/lock 필드 부재를 Host lifecycle 안전성으로 바꾸지 않는다.

### Public source 첫 실행과 exact-version metadata gap — 수정 전 감사

Public source 전용 실제0.253s에서 single1개는 PASS이고 peer fixture는 기존 public
NPM metadata2MiB 제한에서 FAIL했다. Runtime/archives/Policy는 실행하지 않았다.
Raw p2c-pnpm-public-source-first를 보존한다. Native peer parsing/observer 실패가 아니며
packument 전체 이력을 늘려 허용하는 방식으로 해결하지 않는다.

기존 Resolver.Resolve→fetchMetadata의 bounded HTTPS·timeout·header/type/redirect·
origin·size·JSON/identity 구성은 IMPLEMENTED/WIRED/QUALIFIED=YES(기존 npm)다.
PNPM의 이미 고정된 exact version 하나를 registry의 /<package>/<version>에서 조회하는
소비자는 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=NO다. 기존 npm과 generic
selector/metadata limit는 유지한다. 기존 fetchMetadata의 HTTP body 부분만 두 실제
소비자가 공유하도록 추출하고, exact path segments도 같은 public origin 아래 구성한다.
기존 resolved-version identity/URL 검증 body를 helper로 기계적으로 추출하여 기존 Resolve와
새 PNPM exact endpoint가 함께 소비한다. 새 helper는 exact version만 받고 tag/latest를
받지 않는다. Metadata2MiB·10초와 전체 binding2분·취소·부분 결과 거부를 그대로 유지한다.

두 번째 검토: 이는 기존 NPM resolver 대체/contract 완화가 아니다. 다른 HTTP client나
새 source interface를 만들지 않는다. Public exact endpoint·scoped escaping·identity/SRI/URL
mismatch·과대/취소의 direct 회귀와 기존 모든 npm package 회귀, 실제 single/peer source를
다시 확인한다. Whole native producer/public quarantine 완료로 확대하지 않는다.

Exact endpoint의 첫 actual0.484s는 HTTP406이었다. Packument용 install-v1 Accept를
version document에도 전용한 원인이며, 두 번째 source failure도 원본으로 보존한다.
공식 [API](https://github.com/npm/registry/blob/main/docs/REGISTRY-API.md)와
[metadata 표현](https://github.com/npm/registry/blob/main/docs/responses/package-metadata.md)을
대조해 exact endpoint는 application/json을 명시한다. 기존 packument Accept와 size/time/
origin/identity·redirect 거부는 유지하고 exact scoped URL/Accept/과대 회귀를 확인한다.

### P2c1 로컬 구성요소 결과

최종 production source의 Quick116.395s/security6.348s/docs.224s/
vulnerability12.215s는 PASS다. Native selection과 별개인 실제 public metadata source
single/peer1.623s는 artifacts1/4·instances1/4·roots1/2로 PASS했다. 전체 NPM Race와
architecture checker Race도 통과했다. 최종 크기 bound fixture는 invalid text 대신
문법상 유효한 oversized empty lock으로 바꾸어 거부 원인을 구분했다.
[P2c1 결과](./p2c-pnpm-selection-local-20261024.json)가 세부 범위를 소유한다.
이는 bounded unapproved parser와 source binding의 로컬 검증이며 native public producer·
Domain graph/HostInstallAction·layout/transaction/CLI는 아직 MISSING이다.
