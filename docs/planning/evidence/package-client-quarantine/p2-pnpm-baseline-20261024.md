# P2 pnpm — 재개 감사·수정 지도·검증

기준 `5245232` (기능 기준 `095450d`). G127은 [누적 acceptance](./go1273-acceptance-20261024.json)로
완료했다. M13-004/P2만 진행하며 이 문서는 pnpm 지원 완료를 뜻하지 않는다.

Read: [확장 설계](../../../user-journey-cli-ia/09-package-client-quarantine.md),
[실행 계획](../../20-package-client-quarantine-plan.md), [유지보수](../../../engineering/05-runtime-upgrade-maintenance.md),
[Architecture](../../../architecture/01-foundation-and-dependencies.md),
[D-012](../../../threat-model/04-trusted-tooling-and-evidence.md),
[Artifact Port](../../../domain-model/07-contract-artifact-port.md),
[코딩 규칙](../../../engineering/02-coding-security-rules.md), 관련 ci/artifact/sandbox/evidence 오답노트.

## 1. 독립 baseline

| 목표 / 실제 기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 공개 npm 획득·integrity·static/dynamic·Policy — artifact/npm/resolver.go, verification/npm, inspection/npm, application/install_inspect.go | YES | YES | YES | YES | 새 pnpm graph/runtime 소비자 검증은 별도 |
| 기존 npm project guard/원본 snapshot/transaction — promotion/npm_transaction.go, npm_transaction_member.go, npm_update.go | YES | YES | YES | YES | pnpm controls·정확한 내부 link inventory·same-owner 원자 전환 연결 |
| typed client descriptor — cli/package_client.go, bootstrap/python_client.go | YES | YES | YES | YES | pnpm add/install/update 등록·factory와 문구; 미지원 명령 노출 금지 |
| Sigstore offline root/검증 primitive — verification/releaseprovenance, scripts/update-runtime/uv_image.go | YES | YES | YES | YES | pnpm 고유 publisher·subject·workflow 검증; HAA/UV 정책은 유지 |
| pnpm 인증 공급·단일 pin·maintenance·배포·native admission | NO | NO | NO | NO | 아래 P2a, 이어 lock/generator/manifest/provision/관찰기·actual runtime |
| pnpm lock/peer snapshot→공개 registry graph binding | NO | NO | NO | NO | bounded YAML·단일 importer·peer-qualified node identity·독립 package metadata/archive 대조 |
| pnpm add/install/update와 layout·원본 인계·rollback | NO | NO | NO | NO | 실제 native 의미·guarded publication·negative·fresh no-op·최종 영향 matrix |

기존 npm의 ParsePackageLockV3/ParseProjectControls는 package-lock v3·단일 직접 root 전제다.
그 JSON 형식을 pnpm YAML로 위장하거나 peer snapshot을 name/version 하나로 합치지 않는다.
Core의 단일-primary LockedDependencyGraph와 root 없는 ProjectDependencySnapshot을 이미 구분한다.
기존 project inspection/Manifest/SBOM 경로를 재사용하며 여러 직접 root에 가짜 primary를 만들지 않는다.
현재 npm snapshot은 symlink를 거부하므로 전역 거부를 풀지 않는다. 기존 transaction의 lock·보존·
rollback을 재사용하고 pnpm 전용 정규화된 link inventory만 별도로 검증한다. 기존 구현 대체 없음.

## 2. 공식 버전·자료 대조

2026-10-24 공식 GitHub release와 npm registry는 **12.15.0**이다. 계획 당시12.14.0은 과거
관측으로 보존한다. Exact source `91774e72a04d52ce61db06339696a433d383c764`, release
`v12.15.0`, Linux x64 archive25,369,328B를 발견했다. 이는 아직 채택/ABI/안전성 검증이 아니다.
[공식 릴리스](https://github.com/pnpm/pnpm/releases/tag/v12.15.0)와 exact workflow·Cargo source,
npm metadata·attestations 원본은 local raw의 p2-pnpm-discovery-20261024.json이 연결한다.

- v12는 Rust native이며 Node/npm alias가 아니다. npm 배포물은 platform optional binary와
  pn/pnx/pnpm/pnpx 진입점을 포함한다. 설치 문서의 npm installer 요구와 registry engines가
  다르므로 package engines만 보고 실제 런타임·자체 provisioning을 추론하지 않는다.
- 신규 tar integrityTarget/registry revision, config include·config dependency·자동 client/runtime
  전환을 확인했다. 첫 lane은 고정 공개 npm·resource SRI와 단일 프로젝트에서 시작한다.
  tar/revision·외부 include·hook·runtime 다운로드·다른 registry는 명시 capability 검증 전 거부한다.
  이 제한은 전체 P2 완료 범위를 축소한 판정이 아니며 미지원 표·남은 acceptance에 남긴다.
- lockfile crate는9/12 major 호환을 선언하지만 실제 native 출력 형식은 아직 NOT_CAPTURED다.
  공식 source의 importers/packages/snapshots·peer suffix를 읽고 실제 정상/peer fixture로 대조한다.
- 공식 update는 범위와 lock뿐 아니라 manifest range도 갱신할 수 있다. npm update의 range 보존을
  복사하지 않는다. Install/add/update의 controls 변화를 각각 freeze하고 선택 결과와 결합한다.
- 발견한 attestation 둘 중 GitHub immutable-release predicate와 SLSA build provenance는 다르다.
  SLSA subject는 Linux archive 하나, workflow release.yml/tag, source/run binding을 가진다.
  인증서/서명 검증 전 payload 읽기는 **미인증 관측**이다. SAN·issuer·repo/owner IDs·tag/source·
  invocation·predicate와 digest를 실제 암호 검증해야 한다.
- 실제 upstream build의 RunnerEnvironment는 **self-hosted**이며 exact workflow의 Blacksmith
  matrix와 대응한다. pnpm 전용 고정 publisher 정책에만 이 사실을 검증하고 HAA/UV의 기존
  github-hosted 정책은 유지한다. 이는 우리 저장소의 self-hosted runner 등록/구동이 아니다.

## 3. 구현 순서와 정확한 owner

1. **P2a 공급 증명:** 기존 releaseprovenance package의 새 pnpm_release.go에 고정 upstream
   certificate/predicate 정책과 immutable receipt를 둔다. 기존 public root와 pinned Sigstore만
   사용한다. 실제 small bundle/identity fixture와 wrong source/run/tag/publisher/subject·변조·누락·
   과대·취소 회귀를 추가한다. Version/hash는 입력·fixture이며 코드에12.15.0 literal을 두지 않는다.
   scripts/update-runtime/main.go의 기존 dispatch에 verify-pnpm-release 한 consumer를 연결하고
   새 pnpm_release.go에서 absolute proof directory의 identity/bundle/archive를 bounded read한다.
   archive SHA를 실제 bytes에서 계산하고 성공 receipt만 출력한다. 기존 UV bounded reader는
   supply_file.go의 readFrozenSupplyFile로 기계적으로 추출해 두 실제 consumer가 공유한다. 네트워크·Host pnpm 실행·
   추출·pin 변경·설치·Policy 권한이 없다. HAA Verify/UV Verify를 느슨하게 바꾸지 않는다.
2. **P2b runtime:** 인증 archive 내부 전체 inventory·ELF/ABI/지원 baseline을 먼저 관측한다.
   bundled dist/node-gyp payload는 선택/설치 중 임의 실행하지 않는다. Runtime lock/generator/
   support/updater·release manifest·provision을 같은 owner chain으로 연결하고 exact read-only
   native tool admission·실제 version/query/lock 출력과 cleanup을 확인한다. 이때만 pin을 채택한다.
3. **P2c 선택:** artifact/npm 안의 pnpm lock adapter와 sandbox/pnpm resolver를 추가한다.
   새 graph는 공개 metadata/archive와 독립 대조한다. YAML 중복/alias/tag/다중 document·과대·
   unknown schema를 거부하고 importer/specifier/packages/snapshots/peer context를 모두 묶는다.
   기존 resource/network/관찰기 거부는 유지하며 pnpm exact root 역할만 연결한다.
4. **P2d 설치·업데이트:** promotion의 기존 guard/transaction과 공통 검사 engine을 연결한다.
   private store/copy·승인된 tree 안의 exact links·peer graph와 manifest/lock/runtime를 결합한다.
   foreign/shared hardlink·escape·변경된 inode·동시 native mutation은 거부한다. 원본 인계는
   실제 native 형식과 독립 archive proof로 검증한 범위만 허용한다. CLI descriptor/bootstrap/
   help/examples·7개 언어는 실제 wired capability만 노출한다. pnpm dlx는 P6에 남긴다.

## 4. 사전 두 번째 검토와 검증 경계

검토 결과: 공급 서명 성공은 바이러스 검사나 runtime admission 승인이 아니다. SHA만 고정한
unsigned archive를 signed로 표시하지 않는다. GitHub release statement를 SLSA로 대체하거나
upstream self-hosted 사실을 모든 publisher 허용으로 일반화하지 않는다. Archive 안의 경로를
Host에 추출/실행하지 않고 먼저 bounded inventory로 검사한다. Add/update가 쓰는 manifest와
lock을 각각 보존하며 pnpm 내부 links를 공통 symlink 거부의 예외 문자열로 처리하지 않는다.

P2a는 canonical format/Quick/security/docs와 관련 verifier/updater Race, 실제 signed bundle의
offline 명령 성공·archive 변조/상대 경로/누락/과대 거부를 묶어 로컬 검증한다. Crypto negative는
실제 signed proof를 변경하거나 다른 publisher proof를 사용하고 self-confirming mock을 만들지 않는다.
이 supply-only commit 뒤 P2b를 계속하며 개별 helper마다 full CI를 push하지 않는다.

새 common runtime/observer 동작이 다음 단계의 전제를 막는 경계와 전체 P2 후보에서 원격 검증한다.
Linux actual install/add/update/peer graph/no-op·harmful lifecycle·link escape·정책/불완전 관찰 거부,
transaction 동시 변경·SIGINT/중단·rollback/원본 bytes 보존, 기존 npm/PyPI와 영향받는 CPU/세 CUDA,
native macOS/Required가 최종 acceptance다. 실제 source/run/profile과 타이머를 연결한다. 완료한
대용량 archive/store/test payload는 마지막 소비·참조 부재 뒤 지우고 작은 증거만 보존한다.

## P2a 로컬 checkpoint

[실제 공급·검증 결과](./p2a-pnpm-supply-local-20261024.json)는 format/Quick181.382s/security/docs,
실제 공식 archive의 offline command1.578s와 verifier/updater 전체 Race45.633s PASS를 연결한다.
공식 HAA/UV publisher 경계를 유지하고 공통 bounded reader만 기계적으로 추출했다. Runtime pin·
product CLI 지원은 미연결이며 P2a remote qualification/전체 P2 acceptance는 열어 둔다. 다음
P2b의 독립 local 작업을 계속하고 이 helper 단위만 full CI로 반복 push하지 않는다.

자료 관측: signed archive는543 members/expanded61,490,953B이며19개 hardlink는 bundled
node-gyp data 아래 있다. pnpm 자체는 regular57,822,504B/0755, archive UID1001다. ELF는
amd64 GNU interpreter와 libc/libgcc 등의 의존성을 선언한다. 이것은 in-memory 형식 관측이며
실제 ABI·실행 성공/안전성은 아니다. 원본 archive를 임의 Root 소유 도구로 간주하거나 전역
symlink/hardlink 거부를 완화하지 않는다. 다음 공급 단계는 인증한 archive의 pnpm regular 하나를
bounded read로 검증하고 guest 전용 copy로 다룬다. Bundled build 도구는 자동 실행하지 않는다.

## P2b1 — 인증 archive의 native file 연결 전 감사

P2a checkpoint9ad5055는 local VERIFIED다. 공급 archive SHA/서명과 공통 reader는
IMPLEMENTED/WIRED=YES이며 pnpm native file identity는 IMPLEMENTED=NO/WIRED=NO/
QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다. MISSING은 인증한 전체 archive에서 native 하나의
bounded identity를 계산해 실제 maintenance consumer가 사용하는 경계다. Pin·배포·실행은 별도다.

수정 위치는 releaseprovenance/pnpm_tool.go와 기존 PNPMReleaseReceipt, update-runtime의
verifyPNPMRelease다. 인증된 전체 compressed digest를 먼저 검사하고 compressed64MiB/expanded
96MiB·entries1024/native64MiB로 제한한다. Tar를 Host에 추출하지 않는다. pnpm regular/0755
하나만 검증하고 clean한 dist subtree의 regular/dir·기존 regular를 가리키는 data hardlink는
경로/중복/타입/한도를 검증한 뒤 버린다. Hardlink를 실행파일이나 Host 경로로 승격하지 않는다.
Executable은 ELF64 little-endian amd64, 유효한 load 범위와 GNU interpreter/기존 Node image의
libc 계열 의존성을 data로 검증한다. UID1001은 archive metadata이며 실행 owner 권한이 아니다.
새 receipt는 native hash/size만 제공하고 기존 유지보수 명령이 이를 출력한다. 자체 tool bytes의
실제 격리 version/query·ABI와 고정 pin·수명 관리는 이 결과 뒤에 연결한다.

두 번째 검토: UV static-ELF 조건을 느슨하게 바꾸지 않고 pnpm의 dynamic-GNU 형식을 별도로
검사한다. 인증되지 않은 synthetic receipt는 package 내부 구조 회귀에서만 사용한다. 실제 signed
archive의 전체 경로는 realcorpus에서 확인한다. Traversal/absolute/duplicate/link escape·실행
mode·ELF/architecture/loader·digest/truncated/over-limit/trailing data 및 cancelled/zero receipt를
거부한다. 완료된 P2a full Quick를 무조건 반복하지 않고 이 helper의 format·focused/Race·실제
archive 회귀를 먼저 수행하며 다음 native runtime 경계에서 canonical 전체 후보를 묶는다.

### 다음 독립 자료 관측: actual native와 형식

기존 정상 UV ownership-format-discovery operator의 Docker create/label·snapshot/cleanup을
재사용한다. 기본 Docker 서비스를 재시작하거나 전용 daemon을 만들지 않는다. 새 task-owned
carrier/image와 network-none/read-only/non-root/cap-drop/no-new-privileges·512MiB/CPU1 target에서
인증한 pnpm bytes만 실행해 version/help·add/install/update/lock/layout을 관측한다. Package
lifecycle은 ignore하고 Host 파일/서비스/bridge/images 전후 동일·label/ID 확인 후 no-prune 정리를
확인한다. Native package payload는 실행하지 않는다. 필요한 registry 형식 자료는 공식 공개
metadata·동일 tarball을 가져와 target 내부 loopback fixture로만 제공한다. 인터넷 fallback은 없다.
이는 native ABI/형식 관측이며 typed policy/gVisor/Evidence/Policy/실제 제품 지원 qualification이
아니다. 관측을 P2c source authentication의 대체로 쓰지 않는다. 실패 원본은 보존하며 아직
NOT_REACHED인 후속을 성공으로 표시하지 않는다.

## P2b1 로컬 결과와 다음 경계

[Native 공급·형식 결과](./p2b1-pnpm-native-local-20261024.json)는 bounded archive/ELF identity와
canonical Quick117.599s/security/docs·전체 관련 Race19.732s, 실제 signed archive 명령2.274s
PASS다. Native 형식 관측19case/10.319s에서 version/help·add/frozen install/update/transitive/
peer layout을 확인했고 Host 보호 파일·서비스·bridge·images 전후가 같다. 소유 container2와
파생 image를 no-prune 제거하고 빈 작업 디렉터리까지 정리했다. 다음 supply 입력 archive만 보존한다.

실제 lockfile은9.0, peer snapshot은 `use-sync-external-store@1.2.0(react@18.3.1)`이며
virtual-store 경로는 underscore peer suffix로 달랐다. Manifest의 exact6→exact7 update와
여러 직접 root·다른 버전 transitive를 관측했다. Symlink/파일 hardlink nlink1/2/3도 실제다.
이것은 registry-shaped mirror/runc 형식 자료이며 public gVisor/typed network/inspection/
Policy/guarded promotion의 완료로 쓰지 않는다. P2b1은 LOCAL VERIFIED, 전체 P2는 미완료다.
다음은 이 실제 bytes를 단일 runtime owner·maintenance·배포/공급·exact admission에 연결하는
수정 지도와 두 번째 검토다. Prod CLI는 아직 지원하지 않는다.
