# Go 1.27.3 계열 전환 — 우선순위와 착수 감사

계획 기준64122af. 2026-10-22 사용자는 GO-2026-6809를 확인하고 다음 Go 전환 대상으로
1.27.3을 선호하며, 현재 client 작업과의 순서 판단·Queue 기록을 요청했다.

## 결정과 순서

현재 저장소의 go.mod, runtime lock의 Go image, quality tools setupGo와 CI는 이미
1.26.10이다. [CI377](./p1c-guarded-cli-ci377-20261022.json)은 실제1.26.10과 Vulnerability를
포함한 열 개 SUCCESS다. 이 사실은 기존 설치/preview binary를 자동 갱신했다는 뜻이 아니다.

공식 [GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809)는 crypto/x509.Certificate.Verify의
영향 범위를1.26.10 미만과1.27.0-0 이상1.27.3 미만으로 명시한다. 따라서 이 advisory 때문에
지금1.26.10 소스를 긴급 교체할 필요는 없다. [공식 release history](https://go.dev/doc/devel/release)는
2026-10-19의1.27.3 x509 보안 수정과 두 개의 새 계열이 나오기 전까지 지원하는 정책을 확인한다.
1.26은 현재 지원 중이며1.28 출시 때 지원이 끝나는 정책이다. 종료일을 임의로 확정하지 않는다.

순서는 **P1 uv pip의 설치/업데이트/sync acceptance → M13-004/G127 Go1.27.3 전환 →
P2 pnpm → P3..P7**로 결정한다. 전체 client 확장 완료 뒤까지 전환을 미루지 않는다.
P1의 공통 검역·transaction 경계를 현재 검증된 compiler에서 닫아 기능 변화와 runtime
변화를 각각 비교할 수 있게 하고, 다음 client들은 새 검증 기준에서 시작한다.
새 reachable 취약점이나 지원 종료가 이 대기 순서의 근거를 무효화하면 우선순위를 재평가한다.

M13-004 하나만 IN_PROGRESS다. 2026-10-23 P1 acceptance 뒤 G127 착수 감사를 시작했다.
이 단계는 compiler/image/lock/workflow 변경이나 SDK 다운로드·Host activation을 시작하지
않는다. 목표는1.27.3이며 아래 착수 감사의 공식 release/advisory/지원·공급 metadata를 따른다. 새 버전을 silent latest로
채택하거나 예전 raw·fixture의 버전 문자열을 일괄 치환하지 않는다.

Read: [기존 유지보수 설계](../../../engineering/05-runtime-upgrade-maintenance.md),
[실행 절차](../../19-runtime-upgrade-maintenance-plan.md),
[1.26.10 감사와 검증 경계](./go12610-security-baseline-20261021.md),
[Go 1.27 release notes](https://go.dev/doc/go1.27), lessons ci L22/L05·sandbox L12/L18·evidence L11/L16.

## Repository baseline audit

| 책임 / 기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 update-runtime plan/apply/verify·capture·candidate 보호 | YES | YES | YES(existing maintenance scope) | YES(existing) | 재사용;1.27.3 후보의 actual 실행은 별도 |
| 현재 Go1.26.10 source와 공통 네 profile | YES | YES | YES(P1 candidate scope) | YES(P1) | ea287b2와 같은 전체 tree의 CI391/392/393/394 각10SUCCESS; Host 전환·과거 실패 원인은 별도 |
| Go1.27.3 coordinated source/compiler/runtime/tool/bundle 전환 | NO | NO | NO | NO | 착수 metadata/SDK·frozen plan·compatibility·actual 전체 member 재현 |
| Go1.27.3 실제 기존 ecosystem/CPU·세 CUDA/플랫폼/업데이트 경계 | NO | NO | NO | NO | 새 후보 canonical/actual runtime/remote Required와 tuple별 증거 |

기존 유지보수 구현을 재작성하지 않는다. 표의1.27.3 MISSING만 착수 때 구현·검증한다.
현재1.26.10 broad qualification의 과거 timeout/회계 원인은 새 버전이 고쳤다고 추정하지 않는다.

## 수정 위치와 Go 계열 변경의 추가 검토

| owner / 실제 위치 | 방법과 검토 |
| --- | --- |
| scripts/update-runtime의 metadata/patch/apply/verify | 기존 명시 Go 후보·frozen plan·독립 candidate·whole snapshot 검증을 사용한다. 새 SDK/OCI 공급과 before/after identity를 기록 |
| scripts/runtimes.lock.json go_image / scripts/generate-runtime-lock.go | 단일 Go owner에서1.27.3 image/tag/digest·generated identity·go.mod floor와 실제 compiler를 연결. Node/Python/UV/Rust 등 무관한 pin은 이 작업에 섞지 않음 |
| scripts/tools.lock.json / scripts/check/tools.go와 runtime validators | pinned Staticcheck2026.1/gosec/govulncheck 등 실제1.27.3 compiler 호환성을 확인. setupGo 변경만으로 binary 호환을 확정하지 않고 도구 변경은 실제 결함이 확인된 최소 범위 |
| scripts/prepare-gvisor-build-source.sh·build-gvisor-runtime-bundle.sh·build-gvisor-observer-release.sh / update-runtime/gvisor.go captureGvisor | 기존 capture 흐름으로 Go1.27.3을 사용하는 exact gVisor source/patch의6개 member와 observer를 재빌드·재현. 내부 ABI/compiler 결함은 해당 owner에서 수정하며 기존 hashes를 새 binary에 붙이지 않음 |
| 기존 CI/release workflows / version-support.lock.json | 기존 구조적 maintenance renderer·drift 검사를 사용해 host/guest/quality/release compiler 및 support review를 일치. 현재 지원표는 착수 전 변경하지 않음 |
| Go1.27 runtime/stdlib / 기존 sandbox·resolver·Host code | 공식 release notes와 실제 오류를 대조. timer channel/GODEBUG 변경, scheduler/CPU accounting, TLS/x509, ELF/symbol·macOS linker 영향을 검사. json/v2·generic methods 등 무관한 신규 기능으로 code를 재작성하지 않음 |
| macOS CLI/release 지원 문서 | Go1.27은 macOS13 이상을 요구한다. 기존 지원 계약·build metadata/최소 OS를 먼저 대조하고 CI macOS 성공만으로 이전 OS 지원을 주장하지 않음. 계약 변경 필요 시 정확한 차이부터 제시 |
| release/bootstrap/self-update | Go image/runtime tuple이 달라지는 경계를 검증한다. 기존 same-runtime self-update의 mismatch 거부를 유지하며 새 tuple의 최초 설치/기존 native bootstrap·복구/activation 절차를 따름. Host 전환은 source adoption과 별개 |

## 검증과 두 번째 검토

1. 착수 시 실제 Git/Queue와 P1 acceptance를 확인하고 위 표를 다시 판정한다. 공식 SDK
   archive SHA/size·image manifest/contents와 각 tool compiler를 확인한 다음 기존 절차로
   별도 candidate를 만든다. Host 기본 SDK나 설치 CLI를 먼저 교체하지 않는다.
2. 기존 canonical format/Quick/security/docs/workflow/release-gate/freshness 및 maintenance
   verify가 요구하는 profile과 별도 vulnerability/fuzz를 실행한다. Compiler 관련 모듈의
   Race·ABI/observer tests, 독립 bundle/member 재현과 실제 namespace/lifecycle 검사를 연결한다.
3. 새 source의 기존 npm/PyPI/PyTorch/GitHub/Go/Cargo/Terraform·UV 소비자, Linux/macOS 및
   CPU/cu126/cu130/cu132를 기존 qualification 계약대로 분리 실행한다. 기존 Go1.26.10의
   green이나 특정 CUDA 성공을 다른 후보·profile의 성공으로 전용하지 않는다.
4. 원본 code/cause·부분 성공·정리와 final helper/client 복원을 보존한다. 같은 실패를
   반복해서 green으로 만들지 않고 실제 원인을 좁힌다. 완결한 시험 자료와 superseded
   build/cache는 마지막 소비·참조/조회 오류0 확인 뒤 삭제하고 작은 재현·identity만 남긴다.

두 번째 검토 결과: 현재 advisory의 보안 패치는 이미 source/CI에 있으므로 P1 중간에
compiler와 sandbox tuple을 바꿀 이익보다 원인 분리가 어렵게 되는 비용이 크다. 반대로
P2..P7까지 기다리면 검증할 client가 더 늘어난다. 따라서 P1 다음 경계가 적절하다.
정확한1.27.3 도구/gVisor 호환성은 아직 검증하지 않았으며 단순 버전 치환이나 짧은
명령 하나로 완료된다고 약속하지 않는다. 실행 순서와 현재 상태의 단일 owner는 Queue다.

## 2026-10-22 후속 요청 — 버전 혼용 방지와 구버전 정리

기존 단일 owner/maintenance coordinator를 그대로 사용한다. 준비된 기반은 수작업 수정
지점과 누락 위험을 줄이지만 새 Go 계열의 실제 SDK/gVisor/도구 호환성을 보장하거나
필수 qualification 시간을 없애지는 않는다. 별도 compiler lock·임시 PATH fallback은 만들지 않는다.

전환 완료 전에는 go.mod/Go image/generated identity/CI·release setup/tools setupGo뿐 아니라
실제 SDK, quality binary build info, gVisor6개 member·observer의 capture/manifest와 설치 후
readback을 같은 선택 버전에 대조한다. Configuration만 새 버전이고 실행 binary는 이전
버전인 상태를 완료로 처리하지 않는다. 필요한 build-only 도구는 역할·compiler provenance를
기존 정책대로 구분하며 다른 compiler가 silently 사용되지 않게 한다.

구버전 SDK·build root·cache·다운로드·superseded bundle/observer·시험 환경은 inventory와
최종 소비자를 기록한다. 새 버전 qualification/전환·복구 검증 뒤 현재 process/FD/maps/mount,
설치/서비스·작업 문서의 실제 참조와 조회 오류0을 확인한 exact owned 대상만 삭제한다.
Rollback/recovery가 아직 참조하는 실행 파일은 먼저 없애지 않고 검증된 전환 종료 시점에
함께 retire한다. 원본 evidence·역사 fixture의 이전 버전 표기는 실행 잔재와 다르므로 보존한다.
공유 사용자 cache/다른 프로젝트·불명확한 경로·global prune는 정리 범위에 포함하지 않는다.
정리 후 남은 active SDK/tool/runtime identity, old 경로 부재와 설치·서비스 정상 readback을
다시 확인하고 작은 증거만 남긴다. 현재 P1 작업 중인 active1.26.10 입력은 먼저 삭제하지 않는다.


## 2026-10-23 실제 착수 감사·수정 지도·두 번째 검토

시작 HEAD는 P1 문서 종료7ce362b이며 실행 코드의 qualification은 ea287b2/tree9ece0a2다.
[P1 누적 결과](./p1-uv-acceptance-20261023.json)의 네 상태는 모두 YES/MISSING=[]다.
현재 source Go/guest/tools/실제 사용 중인 qualification SDK는1.26.10이다. 새1.27.3 코드를
아직 적용하지 않았고 Host 기본 SDK/CLI·서비스는 이 감사에서 변경하지 않는다.

공식 [release history](https://go.dev/doc/devel/release)의1.27.3·1.26.10과
[GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809)의 fixed 범위를 재확인했다.
[Go1.27 notes](https://go.dev/doc/go1.27)의 macOS13 최소, timer/GODEBUG·JSON 구현·gofmt/vet·
compiler symbol 변화는 호환성 검증 입력이다. 신규 API·실험 옵션을 도입하거나 오류 조건을
완화하지 않는다. 기존 macOS 계약에13 미만의 명시 support promise는 찾지 못했고 native CI의
현재 OS 성공을 모든13+ minor 실행의 증거로 확대하지 않는다.

공식 `https://go.dev/dl/?mode=json&include=all`의2026-10-23 08:49UTC 응답 SHA256은
6b83804b8dd2d6052bb93cc03e4aba2ddee834f82c15f7bf8faacf5f91825950이다. 정확한 SDK1.27.3의
Linux amd64 archive는70,588,678B/SHA256
65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40다.
Darwin 두 archive와 원본 metadata 신원은 local raw `go1273-official-sdk-discovery-20261023.json`이
보존한다. Metadata 발견이며 SDK 실행·image adoption·실제 호환성 성공이 아니다.

### 독립 gap 판정

| 책임 / 실제 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 plan/apply/verify·공식 SDK/bootstrap·두 단계 gVisor capture·source 보호 | YES | YES | YES(existing scope) | YES(existing) | 재사용. 새 compiler의 실제 결과는 별도 |
| README Build와 실제 Go owner의 자동 일치 | YES | YES | YES(current1.26.10 local) | YES(consumer unit) | [로컬 회귀/기록](./go1273-readme-consumer-local-20261023.json); target1.27.3 qualification은 별도 |
| Go1.27.3 SDK/OCI·pinned tools·gVisor/observer source compatibility | NO | NO | NO | NO | 실제 frozen 두 plan·target SDK/bootstrap·전체 bundle/observer capture/재현 |
| 새 tuple의 전체 소비자·플랫폼·CPU/CUDA·activation/retirement | NO | NO | NO | NO | 새 source별 실제 qualification·설치/복구·old owned 참조0 확인 |

README는 P1 종료에서 실제1.26.10 기준으로 수동 교정했다. 착수 기준ab2a161의 기존
`scripts/update-runtime/patch.go:renderChanges`의 Go branch는 go.mod/tools/workflows와 생성값만
갱신하고 README를 읽지 않는다. `scripts/check/tools.go:checkGoVersionConsistency`도
module/guest/tool만 검사한다. 이는 문서 소비자의 누락이며 기존 runtime binary가 혼용됐거나
Go1.27이 호환되지 않는다는 증거가 아니다. 기존 유지보수·검사기를 대체하지 않는다.

### 첫 source 단위 — 누락된 README 소비자만 연결

1. `scripts/update-runtime/patch.go`의 Go 렌더링에 README Build의 선언된 문장 두 Go 값을
   추가한다. 버전은 기존 `p.Current`/`p.Version`에서 파생하며 새 version owner/lock/flag는
   만들지 않는다. 유일한 Build section/현재 문장의 알려진 형식만 허용하고 missing/duplicate/
   mixed baseline을 거부한다. 과거 문구·다른 section·기술명이 같다는 이유로 전역 치환하지 않는다.
2. `scripts/check/tools.go`의 기존 consistency 검사에서 동일 README Build의 minimum/toolchain이
   module·guest·tools와 맞는지 확인한다. 스캐너 권한·Policy·runtime input은 바꾸지 않는다.
3. `scripts/update-runtime/update_test.go`의 기존 repository fixture에 실제 README를 포함하고
   Go upgrade·same-version·stale/unknown input 테스트를 재사용한다. Renderer가 두 값만 갱신하고
   다른 section의 historical 버전을 유지하며 malformed/duplicate/mixed field를 거부함을 확인한다.
   `scripts/check/tools_test.go`의 기존 mixed-SDK fixture도 README를 명시하고 stale minimum/
   stale toolchain/누락/중복·역사 문구 보존을 확인한다. 먼저 같은 negative test가 기존 checker에서
   실패함을 보존한 뒤 교정한다. Before/after 입력·최종 formatted source를 구분한다.
   Indirect `scripts/check/runtime_versions_test.go:runtimeOwnerFixture/checkOwnerFixture`도 같은
   consistency 검사를 소비한다. 새 README 입력을 baseline fixture에 복사하고 coordinated
   future-Go positive fixture의 두 값도 함께 맞춘다. 최초 Quick65.172s/두 package Race22.648s는
   이 fixture의 README 부재에서 실패했으며 제품 guard를 완화하지 않는다. 기존 negative의
   error/assertion은 그대로 보존하고 fixture 준비 누락만 교정한다.
4. 현1.26.10의 canonical format/Quick/security/docs/workflow와 두 tool package 회귀/Race를 실행해
   기존 정상 소비자를 보존한다. 이 단계는 Go target pin/runtime 동작을 바꾸지 않으므로 네
   product profile의 원격 CI를 다시 실행하지 않고 의미 있는 중간 commit만 보존한다.
   구현된 계약은 engineering05의 Go 소비자 목록에도 같은 코드 변경에서 반영한다.

### 다음 source 단위 — 기존 coordinator로 정확한1.27.3 전환

README 연결을 검증·commit한 깨끗한 baseline에서 현 SDK로 coordinator를 먼저 빌드한다.
기존 `plan --component go --version 1.27.3 --review ...` → 첫 candidate apply/verify의 공식 SDK
준비를 사용한다. 변경된 SDK를 ambient PATH나 자동 toolchain download로 대신하지 않는다.
첫 candidate의 exact source/SDK/HAA patch로 `build-gvisor-runtime-bundle.sh --capture`와 observer를
빌드한다. 그 실측 capture로 같은 baseline의 새 state/두 번째 frozen plan을 만든다. 첫 계획을
사후 수정하지 않고, final candidate의 strict build/재현·canonical verify와 별도 vulnerability/fuzz/
관련 Race를 검증한다. Current pinned tools는 먼저 실제 target SDK에서 확인하며 호환성 결함이
확인된 도구만 공식 근거·별도 수정 지도로 갱신한다. Source generation과 native tuple·CPU/세 CUDA/
플랫폼·설치/복구·retirement를 분리하고 최종 후보별 원격 push는 기능 경계로 묶는다.

두 번째 검토에서 기존 구현·입력 신뢰·source cleanliness·전역 치환 금지·새 SDK bootstrap 순환,
linked-worktree VCS projection·quality cache isolation과 처음 발견한 README gap을 대조했다.
UPG 기존 capture/verify는 이미 있어 재작성할 이유가 없다. 새 Go compatibility는 UNKNOWN이며
old hash 재사용·경고 suppression·unknown input의 성공 처리·과거 지연 원인 확정은 금지한다.
현재 target pin/source 구현은 아직 시작하지 않았다. 이 감사는 새 승인 관문을 만들지 않는다.


### 첫 source 단위 로컬 종료 — target pin은 미변경

[README 소비자 로컬 결과](./go1273-readme-consumer-local-20261023.json)는 같은 checker test SHA의
before1.819s/7negative FAIL → after0.984s/PASS를 연결한다. 초기 전체 검사에서 발견한
indirect runtimeOwnerFixture의 README 부재를 교정했고 기존 negative assertion은 유지했다.
교정된 canonical5/Quick45.509s·두 도구 package 전체 Race25.424s는 PASS다. 실제 isolated
plan/apply에서 Build의 두 값만 갱신하고 다른 section의 동일한 과거 문장을 보존했다.
Same-version no-op·unknown/mixed/duplicate 거부와 기존 모든 component/owner 회귀도 유지했다.
완료한 probe driver는 source/명령/원본 증거 연결 뒤 삭제했다. Go1.27.3 SDK/pin/Host 전환은
아직 미실행이며 이 로컬 consumer qualification을 새 compiler 지원으로 확대하지 않는다.
