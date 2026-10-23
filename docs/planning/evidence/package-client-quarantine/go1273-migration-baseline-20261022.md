# Go 1.27.3 계열 전환 — 우선순위와 착수 감사

계획 기준64122af. 2026-10-22 사용자는 GO-2026-6809를 확인하고 다음 Go 전환 대상으로
1.27.3을 선호하며, 현재 client 작업과의 순서 판단·Queue 기록을 요청했다.

## 현재 재개점 — cu126 최초 지연 조사

[CI395](https://github.com/rahoney/heliopause/actions/runs/39957836809)는 c34365e와 같은 전체
checkout tree252509fa에서 Required 포함10SUCCESS·actual Go1.27.3/Linux53/CPU278.97s·native
macOS·observer106과 helper5wait0/final cleanup을 확인했다. 같은 c34365e의
[cu126/CI396 최초 실패](./go1273-cu126-ci396-failure-20261024.json)는 별도로 보존한다.
CPU263.98s와 다른8jobs는 PASS지만 cu126398.73s는 install 중 첫 sample gap1.450981725s가
기존1s를 넘어 실패했다. Read962.750609ms+beforeRead488.230616ms+ledgerLock0.000500ms이며
CPU46.403455/300s·wall잔여2350.539868s라 전체 예산 소진은 아니다. Client identity413.709ms와
response548.849ms/server engine548.612ms로 지연 구간이 관측됐다. 뒤의 CANCELED/termination/
PROFILE_LOOKUP_FAILURE를 첫 원인으로 바꾸지 않는다. Helper3wait0/final CONFIRMED_STOPPED/
client_restore0/RESTORED가 original1을 보존했다.

기존 ci/sandbox/evidence 오답노트와 실제 정상 cu126/CI392를 읽었고 관련 owner7개 source는
byte-identical이다. 이번 MemAvailable 최저413,340KiB·Shmem 최대13,112,728KiB·direct reclaim/
swap/PSI 증가는 실제 관측이다. 정상 CI392에도 pressure가 있었고 controller peak는 오히려
더 높았으므로 Go27 heap 증가나 pressure만의 직접 인과를 단정하지 않는다. 음의149ns 순서
경쟁의 재발로 합치거나 같은 입력의 full rerun·max gap/CPU/wall/coverage 완화를 하지 않는다.

### 다음 진단의 baseline·방법과 두 번째 검토

기존 read timing/OS sampler·UPG006의 temporary install/owning-shell helper/confirmed-stop
restore는 IMPLEMENTED=YES/WIRED=YES다. 재작성하지 않는다. MISSING은 현재 late 지연의
원인 분리와 같은 최종 tuple의 CUDA qualification이다. G127 QUALIFIED/ACCEPTANCE_CLOSED는
계속 NO이며 P2..P7은 NOT_STARTED다. CI396을 green으로 재분류하지 않는다.

현재 WSL의 실제 RAM 약23.47GiB/available 약21.33GiB와 기존 conservative 실행 owner의
runtime4GiB+tmpfs12GiB+controller/OS2GiB를 대조했다. Resource owner source SHA는 과거 정상과
같다. 기존 task operator의 capacity/원본12/consumer guard를 그대로 사용해 c34365e의 독립
clean source에서 bootstrap.test를 빌드한다. 실제 설치·기본 SDK는 아직 원본이며 새 후보와
원본은 영구 state에 준비했다. 이번은 승인된 임시 재검증·복원이고 영구 activation은 아니다.

이 local diagnostic은 hosted 실행과 같은 환경의 재현이 아니다. Existing shell의 Go test
환경에 `GODEBUG=gctrace=1`만 더해 GC 진단을 수집하며 GOGC/GOMEMLIMIT·artifact 환경·
원장·신원·1s 최대 gap·CPU300s/wall40m/coverage/예산은 바꾸지 않는다. 원래 실행기 bytes와
이 한 환경 delta·actual source/binary/최초 failure/pressure/cleanup을 기록한다. GC trace는
진단이며 authority가 아니다. 진단 때문에 정밀한 timing 동등성을 주장하지 않는다.

두 번째 검토: 원본 Root0400 백업·Root0600 journal, ordinary metadata read 계약과 runtime
parent 준비를 대조했다. 기존 controller가 actual stopped/parent wait/client restore/후보12를
검증한 뒤에만 원본을 복원하며 UNKNOWN 아래 강제 restore/rebind는 하지 않는다. Source나
production 경로는 수정하지 않고 unproven 가설의 GC 강제·메모리 knob·보안 한도 변경을 하지
않는다. 결과를 본 뒤 좁은 필요 수정 또는 실행 용량 disposition을 결정한다. 유료 runner,
self-hosted 등록·외부 게시·상시 service 변경을 이 진단에서 추가하지 않는다.

### 2026-10-24 로컬 진단 종료와 다음 좁은 변경 검토

기존 temporary install의 local-cu126-gctrace는 test40m timeout으로 실패했다. 첫 restore는
살아 있는 installed consumer 때문에 거부됐고, 이후 독립 process/socket/FD 조회와 lock·
exact inode 확인 뒤 task가 남긴 네 endpoint만 정리했다. 변경하지 않은 restore guard가
통과했으며 원본12개의 bytes/SHA/UID/GID/mode와 최종 audit를 확인했다. Global SDK1.26.9와
기본 도구는 유지한다. 실패 실행이나 첫 restore 실패를 성공으로 덮지 않는다.

Timeout stack은 promotion이지만 journal birth는 종료 약115초 전이다. 전체40분이 그
거래에서 소모됐다는 해석은 반증됐다. 실제16,122 destination layout의 inert-body commit은
157.921s, retained1GB payload의 read-only snapshot은7.474s다. 둘 다 full qualification이
아니다. 보존 파일 시각은 완료된 write의 시각이며 정확한 단계별 시작/종료는 NOT_CAPTURED다.
원본은 raw go1273-local-cu126-timeout-analysis, phase-time-correction, phase-audit와
confirmed-drain-recovery/final-original-readback 영수증을 따른다.

Baseline: production pipeline·append-only journal·current ancestry guard와 progress port는
IMPLEMENTED=YES/WIRED=YES다. 다시 구현하지 않는다. 현재 test의 단계 시각 진단은
IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO이며 MISSING은 아래
observer 연결과 로컬 검증·실제 다음 실행의 시각 증거다. G127 전체 완료 판정은 계속 NO다.

수정 위치/방법: 기존 pytorch_pressure_test.go에 mutex로 직렬화한 bounded stage observer를
두고 pytorch_linux_integration_test.go의 기존 context에 ports.WithProgress로 연결한다.
Preparation start/end와 bootstrap return을 명시하고 기존 고정 stage의 전환·경과 시각만
기록한다. Download chunk마다 log하지 않으며 unknown stage·외부 문자열·host path·payload는
출력하지 않는다. 최대128개의 stage 전환 뒤에는 기록만 중지하고 검사는 계속한다.
Production source·원장·resource·timeout·poll·coverage·성공 조건·복구 절차는 바꾸지 않는다.

검증/두 번째 검토: non-TTY 기본 progress가 기존 context를 유지함을 presentation.go에서
확인했다. 실제 port를 통해 unknown/반복/chunk/상한과 concurrent 호출을 검사하고 canonical
Quick/security/docs 및 관련 Race를 한 번 묶는다. 기존 full profile은 모든 assertion을 유지한다.
Cache 함수는 invocation마다 새 경로를 만들며 production intake에는 retained-download hit가
없다. 이전 Evidence/ALLOW를 재사용하거나 새 cache 우회를 추가하지 않는다. 새 진단 후보의
실제 실행은 원인 확인용이며 과거 CI396 원인 해결이나 G127 acceptance로 선결론하지 않는다.

[진단 연결 로컬 결과](./go1273-phase-diagnostic-local-20261024.json)는 canonical format/
Quick142.914s/security4.910s/docs0.225s와 실제 port 경로의 unknown/반복/상한·동시 호출
Race15.848s PASS를 확인했다. 최초 restricted executor의 기존 socket EPERM 실패와
짧은 owned TMPDIR·허가 환경 교정을 보존했다. 다음은 이 한 후보의 실제 시각 관측이며
production/한도는 그대로다. 완료 local payload/임시 복제본17,647,185,607 logical bytes는
참조·조회 오류0 확인 뒤 삭제했고 작은 control97개와 로그·stack·복원 증거만 보존했다.

## 현재 검증 상태 — 로컬 완료, 원격·활성화 대기

[최종 로컬 결과](./go1273-local-qualification-20261023.json)는 baseline9670228의 기존
두 frozen plan, 실제1.27.3 SDK·도구5개·Go image와 전체 gVisor member 신원을 연결한다.
First capture419.733s와 최종 configuration7/388.785s가 PASS다. 독립 재빌드는 exact module
acquisition의 TLS 실패를 보존한 뒤 해당 인증 fetch만 복구하여363.643s/PASS했고 six members와
manifest가 byte-identical이다. Source reconstruction·whole lock·기존 strict verifier를 유지했으며
pristine cold acquisition으로 주장하지 않는다. Observer236.438s/106PASS·0FAIL도 확인했다.
Observer별 preparation 파일은 정상 cleanup 뒤라 NOT_CAPTURED이며 소급 생성하지 않는다.

최종 standalone checked source의 vulnerability11.715s/fuzz78.659s/Linux platform150.720s/
full Race224.671s가 PASS다. Platform 최초128.648s 실패는 긴 임시 경로의 Unix socket fixture였고
기존 오답노트의 제한을 실행기에 반영하지 못했다. Source·한도는 유지하고 짧은 owned TMPDIR만
교정했다. 별도 기존 namespace 설치·등록·복구13개와 Host 보호 파일 불변을 확인했다. 이는 기존
서명 mock을 사용하는 filesystem/control 검사이며 실제 signed 설치 qualification은 아니다.

2,081개 전체 추적 파일의 frozen before·checked after bytes/modes를 대조한 뒤 선언된10개
source 파일만 Root에 반영했다. 현재 source/CI/quality/guest owner는1.27.3이며 global Host
SDK1.26.9·설치 CLI/runtime/services는 아직 전환하지 않았다. 완료 namespace 시험39,209,261B와
긴 임시 경로는 참조0·조회 오류0 후 삭제했고 raw·실행파일 신원·launcher source를 보존했다.

현재 `IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`다.
새 동일 후보의 실제 installed ecosystem/CPU·cu126/cu130/cu132/Required·native macOS와
controlled activation/readback·구버전 owned retirement가 남는다. 한 후보로 push한 뒤 각 profile의
실측 시간으로 타이머 확인하며 P2는 이 acceptance까지 NOT_STARTED다. 아래 착수·실패 항목은
당시 상태와 원본이며 현재 완료 판정은 이 절과 Queue를 따른다.

## 결정과 순서

착수 감사 당시 저장소의 go.mod, runtime lock의 Go image, quality tools setupGo와 CI는
1.26.10이었다. [CI377](./p1c-guarded-cli-ci377-20261022.json)은 실제1.26.10과 Vulnerability를
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

### Target canonical7 PASS와 첫 gVisor nogo decoder 실패

a82fb6c의 planf91963a9/verification-3384193493은 실제1.27.3 SDK로 bootstrap·Quick·security·
workflow·release-gate·docs·freshness7개 모두 PASS419.983s다. Raw go1273-configuration-pass가
full plan/SDK inventory/각 원본을 소유한다. 이 결과는 actual gVisor/runtime qualification이 아니다.
동일 candidate의 canonical bundle capture382.193s는 Go stdlib/많은 package compile 뒤
`//pkg/gohacks:gohacks`의 nogo importer가 internal/abi export data5를 maximum2로 읽어 FAIL다.
원본 build/source preparation/MODULE·go.mod·전체 source diff는 raw
go1273-gvisor-nogo-first-failure-20261023.json에 보존했다. Old 정상1.26.10 first/repro 로그에도
`GOVERSION <source file src/go.mod>` 경고가 있으므로 이것을 새 첫 원인으로 쓰지 않는다.

책임 owner는 upstream gVisor의 `tools/nogo/check/check.go:gcexportdata.Read`와 실제 Bazel
`MODULE.bazel`의 x/tools 선언 minimum0.37.0과 upstream go.mod0.43.0에서 구성하는 resolved analyzer graph다. 선언 minimum을 실제 선택 버전으로 단정하지 않는다. 최초 binary의 resolved module version은 별도 수집되지 않았고 decoder maximum2 실패가 실제 근거다.
HAA compiler owner/생성기·SDK 준비·현재 source pin/분석 규칙은 재작성하지 않는다. 기존
정상1.26.10의 두 independent bundle·P1 네 profile은 보존하고 새 성공으로 전용하지 않는다.

수정 전 좁은 실험: 보존한 첫 실패 source controls/diff를 기준으로 task build workspace의
MODULE만 x/tools0.51.0과 그 명시 minimum x/mod0.41.0/x/sync0.23.0으로 조정한다. 이들은
이미 exact source/checksum 검증과 actual1.27.3 scanner에서 사용한 module이며 공식 module
요구를 대조한다. 원본 files·probe diff·module h1·선택 graph를 기록한다. 같은 pinned builder/
SDK·원래 nogo가 실행되는 `//pkg/gohacks:gohacks` target으로 첫 실패를 먼저 검증한다.
원래 HAA candidate/plan은 수정하지 않으며 cache 재사용 probe를 pristine full build로 쓰지 않는다.

성공하면 기존 `tools/gvisor/release-20260907.0.patch`에 MODULE의 검증된 좁은 delta를 합쳐
기존 runtime lock의 해당 patch hash와 generator만 갱신한다. 기존 SDK/upstream patch 보호·
nogo/보안/analyzer 규칙은 유지한다. 새 analyzer closure가 실제 runtime 공용 module 선택도
바꾸는지는 resolved graph/lock으로 확인하고 영향 범위를 그 결과로 기록한다. Current canonical
check/patch preparation negatives와 exact source 복원·target focused/full build를 검증한 뒤 NEW
baseline/two frozen plan을 진행한다. Old hash를 새 build에 붙이거나 build workroot 실험을
완료된 runtime으로 승격하지 않는다. Honnef/IR 등 후속 실제 호환성 오류가 나오면 그 원본을
보존하고 필요한 exact 지원 pin/API 수정만 별도 검토한다. 전체 gVisor release 교체는 하지 않는다.

두 번째 검토: 기존 patch reconstruction·module-lock observed capture·두 계획·strict verifier가
이 경로를 이미 지원한다. 새 configuration/별도 updater/컴파일러 literal 분산이 필요하지 않다.
남은 acceptance는 실제 nogo 경계·최종 six-member 재현/observer·native/CPUthreeCUDA/platform·
기존 설치/복구/retirement이며 P2는 계속 NOT_STARTED다.

최초 scoped build7.943s는 일반 gohacks library의 cache hit일 뿐 nogo aspect를 실행하지
않았다. 원본 해석과 log를 보존하고 probe1은 INSUFFICIENT로 교정했다. 기존
`tools/defs.bzl:go_library`가 별도 `<name>_nogo` test를 생성함을 확인했으며 실제
`//pkg/gohacks:gohacks_nogo`를 실행해 분석 경계를 검사한다. Library cache hit/exit0을
export decoder 해결 근거로 사용하지 않는다. Root runtime patch는 아직 변경하지 않았다.

Explicit nogo probe2는 실제 analyzer를 재빌드·386actions 경계/표준 library와 gohacks
분석을 실행하고1/1 test PASS99.958s다. 실제 external x/tools의 go.mod도0.51.0 source를
확인했다. Honnef0.5.1·보안 규칙·SDK patch를 바꾸지 않고 좁은 importer 경계만 검증했다.
첫 records에서 MODULE minimum0.37을 actual selected처럼 쓴 해석은 원본 보존 후 교정했다.
이후 전체 private release target을 같은 probe source/cache에서 실행해 후속 compile/link
compatibility를 먼저 확인한다. 여전히 clean/frozen capture/전체 runtime qualification이 아니다.
Resolved graph의 다른 common module 영향은 before/after diagnostics와 실제 repo metadata로
대조하고, Root HAA patch 변경·새 frozen plan은 그 결과 뒤 수행한다.

전체 private release probe200.290s는 PASS이며 실제 six outputs와 다섯 Go member의
compiler1.27.3을 확인했다. Raw go1273-gvisor-nogo-private-release/private-members가
cache 재사용·비정식 capture 한계를 명시한다. 검증된 MODULE 세 pin/hash delta만 기존
HAA patch에 합치고 runtime lock의 patch SHA를 갱신한다. Generator/현재 SDK canonical
format·Quick/security/docs와 기존 patch preparation·maintenance 회귀를 실행한다. 새
Root patch 신원은 go1273-gvisor-nogo-patch-application에 기록한다. Standard warning과
다른 resolved module diagnostics는 before/after 원본을 함께 보존하고 full clean runtime
qualification을 미리 주장하지 않는다. 새 SDK first/final capture는 새 baseline에서 진행한다.

[최소 nogo 수정 로컬 결과](./go1273-gvisor-nogo-local-20261023.json)는 기존 preparer의
exact upstream+새 HAA patch 복원/current SDK 선택·canonical5/Quick140.137s와 보존된
private Go1.27.3 실제 six outputs/trace4·nogo1/1을 연결한다. Source commit/SDK patch/
Honnef와 analyzer 규칙은 유지했다. Graph 경고가 실제 출력된 첫 capture와 probe1을
대조한 차이는 x/tools/mod/sync 세 개이며, cached full build의 경고 부재를 module 제거로
해석하지 않는다. 전체 resolved graph 불변이나 fresh capture 완료를 주장하지 않는다.
완료 private source/build/cache는 원본과 작은 검증된 upstream Git mirror만 남기고 참조0/
조회 오류0/Docker bind mount0을 확인한 뒤 retire했다. 새 baseline의 first/final frozen
plan·실제 전체 qualification과 기존 활성 SDK/CLI의 controlled 전환은 계속 남는다.
