# Heliopause Artifact Airlock

외부 Software Artifact와 package를 목표 개발 환경에 설치하기 전에 출처·identity·무결성과 정적·동적 행위를 검사하고, Policy가 허용한 정확한 Artifact만 trusted Promotion하는 도구의 실용성을 탐색한다.

- 프로젝트명: Heliopause Artifact Airlock
- CLI: `helox`
- 독립 저장소: `github.com/rahoney/heliopause`
- Go module path: `github.com/rahoney/heliopause`
- 구현 언어: Go
- CLI framework: Cobra
- 현재 상태: M0–M11 qualification 및 post-qualification release hardening, M12-001~005 생태계 acceptance·전체 qualification·feature freeze와 M12-02 최종 red-team/fix gate 완료. PR29·PR30 merge와 main CI를 확인했고 M13-001 저장소 보호 설정 적용·재조회를 완료했다.
- 완료 작업: M12·M12-02 및 보안 패치 전달, M13-001 저장소 보호·M13-003 build/package 검증, CLI-UX-001·002 CLI 사용성·OS 언어 감지와 SEC-UPD-001 업데이트 검역 (`COMPLETE`)
- 최근 완료: CLI-UX-003 대화형 시작 화면 (`COMPLETE`). 소스4cfe29e의 [CI352](https://github.com/rahoney/heliopause/actions/runs/39316007445)에서 macOS·Linux 실제 소비자·Required 포함 열 개 검사가 성공했다. [Qualification](docs/planning/evidence/cli-launcher/qualification-20261020.md)이 소스·복원·정리와 과거 실패의 범위를 연결한다. 기존 보안·파일 한도와 게시 전 전체 동기화를 유지했다. [업데이트 검역](docs/planning/evidence/update-quarantine/final-review-20261019.md)은 기존 여섯 생태계 범위에서 완료 상태다. PR38은 draft이며 main 병합·정식 릴리스 활성화·공개 배포는 별도다.
- 현재 작업: M13-004 IN_PROGRESS. 이름 결정 전 [전체 코드·보안 감사](docs/planning/evidence/m13-004-convenience/full-source-audit-20261031.md)를 완료했다. 코드358의 일반/CPU·cu126·cu130·cu132가 모두 Required10SUCCESS이며 확인된 코드 문제와 회귀를 정리했다. Same-runtime 검역 업데이트·P1 UV·Go1.27.3 전환은 완료했다. [CLI 시작 화면](docs/planning/evidence/cli-launcher/art-refresh-20261031.md)은 단순 문 로고·가로 그라데이션·별빛으로 사용자 확정과 local 검증을 완료했다.
  현재 client/runner 지원 lane의 source `ec1c260`은 [CI442 ordinary/CPU](docs/planning/evidence/package-client-quarantine/ci442-ordinary-cpu-qualification-20261031.json)와
  [cu126](docs/planning/evidence/package-client-quarantine/p7-cu126-final-qualification-20261031.json)·[cu130](docs/planning/evidence/package-client-quarantine/p7-cu130-final-qualification-20261031.json)·[cu132](docs/planning/evidence/package-client-quarantine/p7-cu132-final-qualification-20261031.json)의 Required10SUCCESS로 검증했다.
  [지원표](docs/planning/evidence/package-client-quarantine/p7-command-acceptance-matrix-20261030.md)는 실제 지원과 미구현 범위를 구분한다.
  후속 Bun 선언/TypeScript entry는 [구현·실제 로컬 앱/CLI와 묶음 Quick/Race 검증](docs/planning/evidence/package-client-quarantine/p6g4-bun-declared-entry-local-20261031.json)과 [CI446](docs/planning/evidence/package-client-quarantine/ci446-bun-declared-entry-hosted-qualification-20261031.json)의 전체10SUCCESS/Linux126·원본·정리까지 확인해 원래 bounded 기능 acceptance를 마쳤다.
  [일괄 설치·원래 보안 목표 감사](docs/planning/evidence/m13-004-convenience/goal-coverage-audit-20261101.md)는 requirements·npm 기본 프로젝트 설치·Cargo fetch·Terraform 기본 init의 공백과 미지원 명령의 성공 종료, 악성 패턴·취약점·버전 변화·일반 공급자 증명·동적 유도/검사 범위의 미완료 항목을 기록했다. 기능 acceptance는 OPEN이다. 기존 검사·CI 통과가 이 목표의 완료를 뜻하지 않으며 이름·배포보다 먼저 다룬다.
  후속 [보강 설계](docs/planning/21-goal-completion-design.md)에 수정 위치·재사용·순서·완료 조건과 자동 용량 관리/시험 정리·성능·CI 운영을 명시했다. 2026-11-01 사용자가 순차 구현을 승인했다. B00 로컬 완료와 B01+B05·[npm 기본 install/ci 구현·연결](docs/planning/evidence/m13-004-convenience/npm-project-local-20261102.json)을 기록했으며 묶음 qualification은 남아 있다. [Cargo fetch 구현·연결](docs/planning/evidence/m13-004-convenience/cargo-fetch-local-20261102.json)도 로컬 검증했으며 [Terraform 기본 init](docs/planning/evidence/m13-004-convenience/terraform-init-local-20261102.json)도 로컬 검증했으며 B06 선택·lock과 B07 Go tidy도 로컬 구현·연결했고 [S09 검사 선언·증거·결과 경계](docs/planning/evidence/m13-004-convenience/check-coverage-local-20261102.json)를 연결했다. [C01 기본 저장 수명·예산 연결](docs/planning/evidence/m13-004-convenience/storage-lifetime-baseline-20261102.md)을 로컬 검증했다. [S01/S08 baseline·공통 DATA 검증](docs/planning/evidence/m13-004-convenience/environment-state-baseline-20261102.md)을 진행 중이며, 반증된 guest9p /tmp의 noexec 경계를 기존 pinned runtime에서 보완했고, npm의 두 fresh 환경·공유 자원 경로를 한정 native로 검증했다. Typed 상태 증거와 단기 VFS 동작 관찰을 연결하고 kernel7/CPP134/actual6를 한정 검증했다. npm 저장 Evidence의 실제 readback·ALLOW도 확인했다. Ordinary Python도 기존 원장에 두 환경/단위 분리와 상태 Evidence를 연결해 한정 native·공통 회귀를 통과했으며, 남은 시나리오·저장 actual·다른 consumer·묶음 qualification은 계속 OPEN이다. C01 전체 정리·lease/recovery·cache/retention은 설계16에서 마치며 grouped qualification과 B01 호환 gap도 OPEN이다. 후속 상태는 Queue를 따른다.
  최종 설계 대조에서 새 검사 공급물의 설치·업데이트 연결과 기본 판정, [명명·배포 인계](docs/planning/21-goal-completion-design.md#13-기능-완료에서-이름-결정배포로-넘기는-조건)를 보완했다. 기능 보강 완료 후 이름 결정·배포 마무리로 진행하며 채널 구현·서명 릴리스·clean Host/게시 검증은 기존 M13 완료 조건대로 남는다.
  후속 검토는 기존 업데이트 체계와 기본 동적 검사를 보존하고, 마지막 통합 단계의 [보안 성능 평가](docs/planning/21-goal-completion-design.md#111-마지막-보안-보강-검토--기존-16번의-필수-종료-단계)를 명시한다. 심층 검사 옵션은 탐지 이득·시간을 측정한 뒤 판단하며 현재 구현된 기능이 아니다.
  [AI 사용 흐름 설계](docs/planning/21-goal-completion-design.md)는 전체 지원 생태계의 CLI 결과·대안 재검역·우회 금지 Skill을 필수로 보완하고, 기존 경로를 재사용하는 최소 MCP는 조건부 구현 후보로 둔다. 현재는 설계만 반영했으며 Skill/MCP 제공이나 Host 우회 강제 차단을 완료로 표시하지 않는다.
  사용자는 공식 이름을 검토한다. [동일 WSL 개발 체험 환경](docs/planning/evidence/m13-004-convenience/local-dev-test-20261101.md)은 Docker29.9.2·최신 CLI/helper로 전환해 Python 설치·업데이트와 Go 설치·업데이트·빌드까지 확인했다. 공식 채널/publisher binding·새 서명 배포·배포 전 acceptance는 남는다. 원래 지원 제외·별도 확장과 필수 누락은 구분하며 실제 순서는 [Queue](docs/planning/02-current-work-queue.md)가 소유한다.
  Main 병합·게시 승인은 별도이며 첫 정식 공개판 목표는 v1.0.0이다.
- 선행 판정: M13-002는 [승인된 C3](docs/planning/18-m13-production-release-operations-contract.md#m13-002--protected-branch-qualification)에 따라 `ACCEPTED_WITH_EXCEPTION / QUALIFIED=NO / ACCEPTANCE_CLOSED=YES`다. 실제 missing-CLA는 `NOT_TESTED / DEFERRED`이며 최초 외부 기여 병합 전에 검증한다. 필수 CLA 보호·서명은 유지하고 구체적 merge·tag·publish는 별도 후보 검토를 따른다.
- 유지보수 완료: `UPG-001..006 COMPLETE / QUALIFIED`, PR34/main CI320 전달 완료. 공개 attested release·상시 service·기존 full-Race 후속은 별도 범위로 유지한다.
- 진행 상태의 canonical owner: [`docs/planning/02-current-work-queue.md`](docs/planning/02-current-work-queue.md)

Heliopause는 Apache-2.0으로 배포한다. 외부 기여는 `CLA.md`의 Harmony
Copyright License 및 Option Five 조건과 자동 CLA status check를 충족해야 하며,
자세한 절차는 [`CONTRIBUTING.md`](CONTRIBUTING.md)에 있다. Apache-2.0
`LICENSE`는 사용자에게 부여하는 outbound 권리이고, CLA는 contributor가
프로젝트에 제공하는 inbound 권리다.

## Project Notes

상세 설계·결정·작업 계획은 [Documentation Guide](./docs/README.md)를 통해
버전 관리되는 canonical 문서에서 확인한다. 개인 작업 메모와 로컬 실행 상태는
별도이며, source 문서가 runtime 배포나 일반 Host 설치 완료를 뜻하지 않는다.

## Build

Heliopause는 Go `1.27.3` 이상이 필요하며, 개발·CI toolchain은 Go `1.27.3`으로 고정한다.

개발용 Go는 영구 경로 `$HOME/.local/go`의 단일 SDK를 사용하고 해당 `bin`을
PATH에 둔다. `GOTOOLCHAIN=local`로 자동 SDK 추가 다운로드를 금지한다. 버전
업데이트 시 host SDK, `go.mod`, quality tool/CI pins와 Go guest runtime lock을
함께 갱신하고 검증한 뒤 이전 SDK를 제거한다. Canonical Quick은 이 버전들의
불일치를 거부한다.

Host·Go guest 단일화와 로컬 설치 검증은
[단일화 보고서](docs/engineering/evidence/go-single-version-20261015/README.md)를 따른다.
이전 보안 수리와 별도 full-Race 후속 항목은
[보안 검증 보고서](docs/engineering/evidence/security-go1269/README.md)에 보존한다.

다음 build 명령은 source checkout을 가진 개발자·검증자가 사용하는 경로다. 일반
사용자는 최종적으로 GitHub Release의 검증된 bootstrap entrance를 사용해야 하며,
그 public installer와 runtime/helper bundle activation은 M13-005 완료 전까지
제공된다고 주장하지 않는다.

```sh
go build -trimpath -o ./bin/helox ./cmd/helox
./bin/helox --help
```

dependency와 품질 도구까지 재현하려면 저장소 루트에서 다음을 실행한다.
품질 도구는 source tree 밖 전용 cache에 exact version으로 설치된다.

```sh
go run ./scripts/check bootstrap
go run ./scripts/check quick
go run ./scripts/check freshness
```

## Commands

지원하는 공개 입력은 anonymous public npm Registry, PyPI, named official PyTorch
source, public GitHub Release asset, Go proxy/SumDB, crates.io와 Terraform Provider다.
npm/pip install은 기본적으로 현재 managed project 또는
active virtual environment를 사용하며, `--target`은 고급 absolute destination
override다. GitHub install은 기본 user bin destination 또는 새 `--target`을 사용한다.

터미널에서는 사람용 결과를 기본 표시하고, 기존 npm/PyPI/GitHub 출력은
파이프로 보내면 JSON을 유지한다. 언어 auto는 Windows/macOS 표시 언어를 우선하고 조회 불가 시 locale·영어로 대체하며, 저장하거나
일회성으로 지정할 수 있다. 영어·한국어 전체 문구와 독일어·프랑스어·스페인어·
브라질 포르투갈어·일본어 핵심 설명을 지원하고 누락 번역은 영어로 표시한다.
[표시 계약](docs/user-journey-cli-ia/05-terminal-presentation.md)을 따른다.

```sh
helox config language ko       # 저장; auto로 시스템 언어 복귀
helox --lang en --help         # 이번 명령에만 적용
helox --output json npm inspect is-number@7.0.0
helox doctor

helox npm inspect '<package>[@<version>]'
helox npm install '<package>[@<version>]'
helox npm install '<package>[@<version>]' --target /absolute/project

helox pypi inspect '<project>[@<version>]'
helox pip install '<project>[@<version>]'
helox pip install '<project>[@<version>]' --target /absolute/venv

helox github inspect '<owner>/<repo>@<tag>#<asset>'
helox github install '<owner>/<repo>@<tag>#<asset>'
helox github install '<owner>/<repo>@<tag>#<asset>' --target /absolute/new-target

helox pip install 'torch@2.14.0+cpu' --source pytorch:cpu
helox go get 'github.com/spf13/pflag@v1.0.10'
helox go mod download
helox go build ./...
helox cargo add 'itoa@1.0.17'
helox cargo build
helox terraform init 'hashicorp/random@3.7.2'
```

`inspect`는 `ALLOW`여도 target에 반입하지 않는다. `install`은 모든 필수
검사와 Policy가 `ALLOW`인 exact Artifact set만 offline Promotion한다.
`MANUAL_REVIEW`, `BLOCK`, 불완전한 검사, cleanup 불확실성에서는 target을
게시하지 않는다.

## Supported paths

| Host / input | MVP support |
| --- | --- |
| Linux amd64 + pinned Docker/runsc/observer | npm, PyPI wheel/sdist와 GitHub Release ELF/ZIP/tar.gz 검사; approved managed project/venv 또는 새 target으로 exact set Promotion. PyTorch·Go·Cargo·Provider의 M12 qualification 상태는 아래 표 참조 |
| macOS native | CLI build와 기본 실행만 검증됨. Linux dynamic backend와 automatic Promotion은 지원하지 않음 |
| Windows 11 + WSL2 Ubuntu 24.04 | WSL2 내부 Linux CLI build와 기본 실행만 검증됨. Windows-native backend가 아니며 dynamic/Promotion support를 뜻하지 않음 |
| 기타 OS/architecture/runtime | 지원하지 않으며 누락된 capability를 안전으로 간주하지 않음 |

## M12 qualification

다음 생태계 경로의 acceptance와 M12-005 전체 qualification·feature freeze를 완료했다.
Qualified source 경로와 일반 사용자용 public installer/배포 완료는 각각의 acceptance를
따른다. Runtime/helper를 준비하지 않은 Host에서 동적 실행을 지원한다고 주장하지 않는다.

| Milestone | Implemented path | Status |
| --- | --- | --- |
| M12-001 | 공식 PyTorch source profile을 통한 `helox pip install` | acceptance complete; four bounded release tuples qualified |
| M12-002 | public Go Modules: `helox go get`, `helox go mod download`, `helox go build` | acceptance complete |
| M12-003 | public crates.io: `helox cargo add`, `helox cargo build` | acceptance complete |
| M12-004 | public Terraform Provider 설치: `helox terraform init` | installation acceptance complete; RPC/cloud behavior not attested |
| M12-005 | 전체 생태계 qualification 및 feature freeze | COMPLETE |
| M12-02 | 최종 red-team/fix gate | COMPLETE; reviewer merge pending |

PyTorch profile roles are CPU (primary non-CUDA), cu126 (compatibility), cu130
(primary CUDA), and cu132 (extended compatibility). `cu128` is removed and is
not selectable. CUDA qualification is explicit workflow-dispatch only.

First-release PyTorch 지원 tuple의 exact version·Python/ABI/platform·resource bound와
qualifying scope는 [canonical support tuple](./docs/planning/16-m12-01-ecosystem-expansion-contract.md#first-release-support-tuples)을
따른다. CUDA broader module 검사는 `--inspection-prerequisites`로 명시한 고정
NumPy2.4.6을 검사 전용으로 사용하며 target에는 설치하지 않는다. 기본 입력은 empty이고
필수 검사가 불완전하면 자동 승격하지 않는다. `triton.profiler.viewer`의
`NOT_ATTESTED` 결과는 기능 정상·나중 실행 차단·GPU computation 검증을 뜻하지 않는다.

`scripts/version-support.lock.json` records externally managed baselines.
Developer checks warn after 90 days; strict qualification freshness fails stale
records without network access or lock rewriting.

M12-001~005와 M12-02가 완료되면 기능 개발을 동결하고 M13에서 검증된 GitHub
Release, OS package, npm/PyPI/pipx 및 Homebrew bootstrap·운영 절차를 진행한다.

전체 Linux 경로는 [runtime lock](./scripts/runtimes.lock.json)의 exact Docker,
gVisor source·runsc, Bazel, Node 및 Python identity 경계를 요구한다. `runsc-trace`는
[pod-init config](./tools/gvisor-observer/pod-init.json)로 보호된 shared observer
socket을 sandbox 시작 전에 연결해야 한다. 현재 저장소는 일반 Host용 runtime
installer/bootstrap entrance를 아직 제공하지 않는다. M13-005에서 GitHub Release
검증부터 helox/runtime/helper atomic activation, `doctor`, 실제 전체 생태계 smoke와
quarantine 절차를 완료하기 전에는 이 구성이 없는 Host에서 동적 검사나 automatic
Promotion을 지원한다고 간주해서는 안 된다.

## Data and network boundaries

- 작업 데이터는 OS user cache 아래 `heliopause/intake`, `evidence`, `staging`에
  저장된다. 실제 credential이나 개인 Host 파일을 검사 fixture로 넣지 않는다.
- npm/PyPI dependency resolution은 Heliopause가 생성한 전용 Docker network와
  default-deny Host firewall policy를 사용한다. unrestricted bridge나 proxy-only
  fallback은 없다.
- target container에는 Docker socket, observer socket, Evidence store, Host
  environment 또는 일반 Host filesystem을 전달하지 않는다.
- 완료된 Evidence와 staging record는 audit/retry를 위해 유지된다. MVP에는
  자동 retention/garbage-collection daemon이 없다.

## Troubleshooting

- `automatic ... requires Linux amd64`: 이 Host에서는 automatic dependency
  resolution/Promotion을 지원하지 않는다. 다른 runtime으로 우회하지 않는다.
- `runtime is unavailable`, observer 또는 trace incomplete: exact lock과
  `runsc-trace` registration, protected observer socket, pinned image가 모두
  필요하다. 관찰 실패 상태는 `ALLOW`가 될 수 없다.
- resolver firewall backend/apply/verify/cleanup 오류: Docker의 iptables
  `DOCKER-USER` 또는 공식 nftables backend를 검증하지 못한 것이다. 일반
  bridge로 재시도하지 말고 Host Docker/firewall 구성을 복구한다.
- target exists/contains symbolic link: 새 absolute target을 사용한다. 기존
  경로 덮어쓰기나 symlink 경로는 지원하지 않는다.
- scanner, vulnerability database 또는 dependency download 실패: 검증 실패이며
  clean 결과가 아니다. network/cache/tool identity를 복구한 뒤 같은 exact
  source에서 다시 실행한다.

Heliopause의 `ALLOW`는 고정된 검사·Policy 범위를 통과했다는 뜻이며 Artifact의
절대적 안전성을 보증하지 않는다. 보안 문제는 [Security Policy](./SECURITY.md)에
따라 비공개로 신고한다.
