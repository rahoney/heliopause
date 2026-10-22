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

M13-004 하나만 IN_PROGRESS다. G127은 NOT_STARTED이며 이 문서는 compiler/image/lock/
workflow 변경이나 SDK 다운로드·Host activation을 시작하지 않는다. 목표는1.27.3이며 실제
착수 때 공식 release/advisory/지원과 공급 metadata를 재확인한다. 새 버전을 silent latest로
채택하거나 예전 raw·fixture의 버전 문자열을 일괄 치환하지 않는다.

Read: [기존 유지보수 설계](../../../engineering/05-runtime-upgrade-maintenance.md),
[실행 절차](../../19-runtime-upgrade-maintenance-plan.md),
[1.26.10 감사와 검증 경계](./go12610-security-baseline-20261021.md),
[Go 1.27 release notes](https://go.dev/doc/go1.27), lessons ci L22/L05·sandbox L12/L18·evidence L11/L16.

## Repository baseline audit

| 책임 / 기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 update-runtime plan/apply/verify·capture·candidate 보호 | YES | YES | YES(existing maintenance scope) | YES(existing) | 재사용;1.27.3 후보의 actual 실행은 별도 |
| 현재 Go1.26.10 source와 일반 Required CI | YES | YES | YES(CI377 scope) | YES(C3b scope) | 같은 source의 세 CUDA 전체 완료로 확대하지 않음; 기존 별도 한계는 보존 |
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
