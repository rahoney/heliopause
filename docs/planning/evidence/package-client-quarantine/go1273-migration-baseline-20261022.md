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


### 첫1.27.3 실제 configuration 실패 — module 형식 경계

90392d6의 첫 frozen plan21dc7b1e를 별도 candidate에 적용했다. 공식 archive의 SHA/size와
전체 SDK inventory·실제 compiler1.27.3을 확인했고 bootstrap은 PASS다. Quick의 format/runtime
lock/support freshness는 통과했으나 `checkModuleDrift`가 require 블록 차이를 거부했다.
이는 [Go1.27 공식 tidy 변경](https://go.dev/doc/go1.27)의 duplicate require block 통합과 맞는다.
새 language/API 실패나 Staticcheck analysis 성공으로 해석하지 않는다. Staticcheck 등 실제
analysis·gVisor/native/전체 tuple qualification은 아직 남는다.

첫 plan·SDK inventory·bootstrap/Quick 최초 로그는 local raw `go1273-first-*`와
`go1273-first-module-drift-20261023.json`으로 원본 보존했다. 기존 frozen plan/journal은 고치지
않고 이 실패 결과를 qualification 성공으로 사용하지 않는다.

추가 수정 위치는 기존 `go.mod` 한 파일의 require 블록 형식이다. 유일한 Go directive는 현재
Root pin1.26.10을 유지한다. 별도 private snapshot에서 실제 authenticated1.27.3 `go mod tidy`의
결과를 만들고 exact122개 require의 path/version/directness·기타 directive와 go.sum 불변을
기계 대조했다. 새 floor1.27.3에서 tidy 재실행이 no-op이며, 동일 정리 결과의 floor를현재
1.26.10으로 유지했을 때 기존1.26.10의 `tidy -diff`도 PASS다. Raw
`go1273-module-normalization-proposal-20261023.json`이 신원·입력·검증 범위를 소유한다.

이 검증된 standard-tool 출력의 require grouping만 현재 baseline에 반영한다. 새 dependency
version이나 Go1.27 전용 formatter/예외 코드를 추가하지 않으며 canonical module drift는 유지한다.
Current SDK의 Quick/security/docs와 source diff를 확인해 baseline commit으로 보존한 뒤 NEW
state/frozen plan을 생성한다. 변경된 HEAD를 예전 plan에 붙이거나 receipt를 수동 수정하지 않는다.
실제 SDK/도구 호환성이나 전체 migration 완료는 그 후의 새 후보 검증으로만 판정한다.


모듈 grouping 보완의 [current SDK 검증](./go1273-module-layout-local-20261023.json)은
Quick48.866s/security/docs PASS다.122개 require/directness·기타 directive·go.sum은 유지했고
Root Go floor는1.26.10이다. 완료한 private proposal clone과 driver는 결과·신원을 남긴 뒤
삭제했다. 첫 frozen plan/SDK inventory/실패 원본은 별도 보존하며 다음 계획은 새 state를 쓴다.

Host preflight에서 기본 개발 SDK의 실제 version은1.26.9였다. 이는 이 branch의 current
qualification SDK1.26.10 및 격리된 target1.27.3과 다르며, source-only adoption과 Host 전환을
구분해 기록한다. Local raw `go1273-host-sdk-before-20261023.json`이 exact configured/resolved
경로·readback을 소유한다. 최종 전환은 Host의 실제 새 SDK/quality/runtime readback과 복구를
확인해야 하며1.26.9/1.26.10의 old owned 경로·cache는 역할/마지막 참조를 확인하고 retire한다.

### 두 번째 configuration 실패 — Staticcheck export data 호환성

새 baseline fe225e8의 frozen plan ed2d463f/verification-841668246은 공식 SDK1.27.3의
bootstrap·format·runtime/support·module drift/integrity·production/test build·architecture·
workflow·go vet까지 PASS다. 다음 Staticcheck2026.1(v0.7.0)이 `cmp` 등의 export data
version5를 maximum2 decoder로 읽지 못했다. Raw `go1273-staticcheck-incompatibility-20261023.json`은
첫 오류와 각각의 원본 hash를 보존한다. 앞 module 형식 교정의 회귀나 새 runtime의 성공으로
해석하지 않는다. 기존 frozen plan/SDK receipt는 변경하지 않는다.

공식 [Staticcheck2026.2 release](https://github.com/dominikh/go-tools/releases/tag/2026.2)는
Go1.27 지원과 module version v0.8.0을 명시한다. 해당 [go.mod](https://github.com/dominikh/go-tools/blob/2026.2/go.mod)의
minimum은1.26.0으로 current1.26.10에서도 build 가능하다. MIT와 checksum 검증/source build를
유지한다. 신규 SA9010과 기존 진단 변화·upstream SA5011 비활성화도 검토 대상이며 프로젝트의
suppression은 추가하지 않는다. 이 공식 지원 선언만으로 actual analysis PASS를 주장하지 않는다.

수정 지도: `scripts/tools.lock.json`의 Staticcheck version/expectedVersion만2026.2/
`staticcheck 2026.2 (0.8.0)`으로 바꾸고 setupGo는 current1.26.10을 유지한다.
`scripts/version-support.lock.json`의 해당 한 record만 실제 검토 날짜·release·이유로 갱신한다.
Engineering03의 현재 도구 선택에 이 변경을 연결하고 historical M0/fixture의2026.1은 보존한다.
기존 canonical bootstrap으로 exact module의 checksum·binary/compiler를 확인한 뒤 current
SDK Quick/security/docs를 실행한다. 분석 결함이 실제 확인되지 않은 다른 도구는 갱신하지 않는다.
로컬 검증/중간 commit 후 NEW state/plan으로 target SDK bootstrap·실제 analysis·전체 canonical을
재검증한다. 완료한 실패 candidate/cache는 원본 보존·마지막 참조0 확인 후 retire한다.

두 번째 검토: 기존 bootstrap·tool lock/지원표·Go coordinator가 이 선택을 이미 소비하므로 새
tool updater/version owner/예외가 필요하지 않다. Exact release와 actual output을 대조하고
새 Go pinned setup은 기존 Go renderer가 연결한다. 현재 후보에는 gVisor/native/CPU/CUDA 결과가
없으며 그 검증·설치/복구·정리는 다음 동일 G127 범위에 남긴다.

최초 current SDK bootstrap8.701s는 exact 새 release 설치 뒤 binary가 실제 출력한
`staticcheck 2026.2 (0.8.0)`과 추정했던 `(v0.8.0)`의 차이를 strict identity 검사로 거부했다.
Module version과 display 문자열을 구분해 expectedVersion만 actual output으로 교정한다.
최초 원본 `go1273-staticcheck-baseline-bootstrap-20261023.*`는 보존하고 비교 검사를 완화하지 않는다.

[Current SDK 로컬 결과](./go1273-staticcheck-current-local-20261023.json)는 exact2026.2의
bootstrap34.276s·실제 Go1.26.10 buildinfo·Quick56.004s/security4.591s/docs PASS를 연결한다.
현재 기준선의 분석·기존 tests는 통과했고 target1.27.3 analysis는 다음 NEW plan에서 검증한다.
첫 실패 state의 전체 frozen bytes/modes·process/FD/maps/mount 참조0·조회 오류0을 확인한 뒤
완료 payload4,103,892,746B를 retire했고 frozen plan/SDK inventory/원본은 보존했다.

### 2026.2 실제 target 반증과 공식 patch 선택

9694149의 새 plan98869da1/verification-1876054408은 bootstrap/build/vet PASS 뒤
Staticcheck2026.2/v0.8.0의 decoder가 export data5/max4로 FAIL다. Current SDK PASS와
공식 Go1.27 계열 지원을 actual1.27.3 지원으로 확대하지 않는다. 기존 권장 target 선택은
superseded이며 첫 원본은 raw go1273-staticcheck20262-target-failure-20261023.json으로 보존했다.

첫 검토에서2026.2 exact tag만 확인하고 후속 stable patch를 확인하지 못했다. 공식
[최신 release2026.2.2](https://github.com/dominikh/go-tools/releases/tag/2026.2.2)는
Go1.27.2의 변경된 export data 지원과 crash/진단 수정을 명시한다. 실제 release 목록과
[go.mod](https://github.com/dominikh/go-tools/blob/2026.2.2/go.mod)의 floor1.26.0·x/tools0.51.0을
대조했다. Latest 추종은 하지 않고 요청한 compatibility gap을 고치는 exact patch를 선택한다.

추가 수정 전에 이미 authenticated target SDK와 보존한 checked source에서 외부 bin directory에
exact2026.2.2만 checksum 검증/source build하여 offline 전체 분석을 먼저 실행한다. 기존
canonical target profile의 첫 실패 후 원인을 좁히는 probe이며 configuration PASS로 기록하지 않는다.
Frozen source/plan/기존 binary는 수정하지 않고 실제 display/buildinfo·소스 검사 전후 hash를 보존한다.
성공한 경우 tools.lock의 Staticcheck version/expectedVersion과 support record만 patch로 갱신하고
current bootstrap/Quick·보안·문서로 정상 기준선을 확인한다. Target SDK 새 canonical plan은 그 후다.
두 번째 검토에서도 updater·guard·다른 도구·제품 의존성을 바꿀 필요는 없다.

Raw go1273-staticcheck202622-target-probe-20261023.json의 actual Go1.27.3 exact patch install6.217s·
display `staticcheck 2026.2.2 (0.8.2)`·buildinfo·offline full analysis8.613s가 PASS다.
Frozen checked source의 전후 hash는 일치한다. 공식 latest stable endpoint도2026.2.2로
재확인했으며 원본은 go1273-staticcheck202622-official-review-20261023.json에 보존했다.
최종 선택/expectedVersion은 이 실제 출력으로 확정한다. Current bootstrap9.151s도 PASS다.
다음 NEW frozen state의 quality cache는 같은 actual SDK의 준비 cache를 독립 파일로 복사해
bootstrap 비용을 줄일 수 있다. Hardlink/symlink alias를 만들지 않고 transfer inventory와
원본·목적지를 보존하며 canonical bootstrap·identity·module verify를 생략하지 않는다.
Cache 재사용을 cold-cache qualification으로 표시하지 않는다.

[Patch 로컬 결과](./go1273-staticcheck-patch-local-20261023.json)는2026.2 실패/반증과
actual2026.2.2 target 전체 분석·current canonical Quick55.705s/security/docs를 연결한다.
두 번째 완료 실패 state4,896,518,774B도 참조0/오류0·frozen bytes/modes 확인 후 정리했다.
두 완료 state의 원본은 보존하고 아직 쓰는 target SDK·cache는 다음 후보 준비까지 유지한다.

### Target Quick 전체 통과 뒤 별도 gosec 호환성 gap

ce6ec1c의 새 candidate는 canonical bootstrap/Quick가 PASS다. Staticcheck patch와 기존 기본
tests의 실제 target 검증 범위가 닫혔다. 다음 security에서 gosecv2.28.0이 export data5/max4로
실패했다. Summary Issues0은 import/type failure를 상쇄하지 않는다. 첫 plan/SDK/Quick/security
원본은 raw go1273-gosec-target-failure-20261023.json이 보존한다.

이 단계에서 govulncheckv1.7.0도 같은 frozen checked source/actual1.27.3의 기존 canonical
vulnerability로 먼저 실행했고12.432s/PASS다. 따라서 최신 v1.8.0이 있어도 이 작업에서
govulncheck pin을 바꾸지 않는다. 실패한 scanner와 정상 scanner를 구분한다.

공식 [gosec latest v2.29.0](https://github.com/securego/gosec/releases/tag/v2.29.0)의
Go1.27 지원·의존성 갱신과 [module](https://github.com/securego/gosec/blob/v2.29.0/go.mod)의
Go1.25.0 floor/x-tools0.49.0을 대조했다. 실제 이 export 형식 지원은 별도 probe로 확인한다.
외부 bin에 exact source/checksum build 후 기존 security.go의 동일 include/nosec 인자로
전체 source를 검사한다. Frozen source hash 전후·binary output/module/compiler를 기록하며
성공만으로 전체 configuration을 PASS로 바꾸지 않는다. Official latest·exact tag·license와
새 진단/의존성 차이를 보존한다. 공급 graph는 quality 도구에 격리하고 AI 옵션은 켜지 않는다.

성공하면 수정 위치는 기존 tools.lock의 gosec version/실측 expectedVersion과 support record
한 개·engineering03의 현재 선택뿐이다. setupGo/current floor1.26.10·제품 dependency graph·
다른 도구·기존 include/negative/보안 규칙은 유지한다. Current canonical bootstrap/Quick/security/
docs를 확인해 한 단위로 commit하고 NEW frozen Go plan의 전체 검증으로 이어간다. 두 번째
검토에서도 bootstrap/updater/보안 runner 재작성이나 warning suppression은 필요하지 않다.

앞 v2.29.0 target probe는 source install33.168s 뒤 같은 export data5/max4로 FAIL6.018s다.
따라서 Root pin은 바꾸지 않았다. Web latest 결과는v2.29.0이었으나 실제 GitHub API latest는
2026-10-23 09:22:19UTC에 발표된v2.30.0을 반환했다. Cached web 조회를 actual latest로
확정하지 않으며 raw go1273-gosec229-official-review의 응답 차이와 실패를 보존한다.
공식 [v2.30.0](https://github.com/securego/gosec/releases/tag/v2.30.0)의 Go1.27.3/1.26.10
업데이트·의존성/기존 G112/G402/G602 진단 수정과 exact tag를 추가 검토했다. 실제 raw와
Go proxy의 go.mod 모두 floor1.26.0이며 Apache2.0이다. Raw go1273-gosec230-official-review가
원본을 소유한다. 위 수정 범위/기존 규칙을 유지하고 이 exact version만 먼저 실제 probe한다.

Exactv2.30.0의 actual1.27.3 source build43.522s/기존 display·buildinfo/동일 규칙 전체
offline scan3.033s는 PASS376files/79,545lines/Issues0이며 frozen source 전후 hash도
일치한다. Raw go1273-gosec230-target-probe가 이 좁은 경계를 소유한다. 같은 version의
current SDK canonical bootstrap54.248s도 PASS다. 실패한 v2.29는 적용하지 않았고 정상
govulncheck/다른 도구·제품 graph/기존 보안 규칙은 변경하지 않는다. 세 번째 완료 state의
payload5,015,905,153B도 frozen source/참조0/조회 오류0 확인 후 retire했다. 작은 원본은 보존한다.

[gosec 로컬 결과](./go1273-gosec-local-20261023.json)는 actualv2.30 target scan과 current
SDK buildinfo·canonical Quick46.480s/security4.295s/docs PASS를 연결한다. 기존 scanner/
규칙/제품 source는 유지했고 다음 NEW frozen plan으로 전체 target configuration을 확인한다.
