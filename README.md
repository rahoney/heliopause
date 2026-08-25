# Heliopause Artifact Airlock

외부 Software Artifact와 package를 목표 개발 환경에 설치하기 전에 출처·identity·무결성과 정적·동적 행위를 검사하고, Policy가 허용한 정확한 Artifact만 trusted Promotion하는 도구의 실용성을 탐색한다.

- 프로젝트명: Heliopause Artifact Airlock
- CLI: `helox`
- 독립 저장소: `github.com/rahoney/heliopause`
- Go module path: `github.com/rahoney/heliopause`
- 구현 언어: Go
- CLI framework: Cobra
- 현재 상태: M0–M8 및 M9-001~003 완료. M9 Product Install UX를 진행 중이며 production-ready release는 보류
- 현재 작업: M9-006 — transaction hostile-boundary regression and qualification

## Project Notes

상세 설계·결정·작업 계획 문서는 로컬 작업 환경에서 관리하며 이 저장소의 배포 대상에 포함하지 않는다.

## Build

Heliopause는 Go `1.25.13` 이상이 필요하며, 검증된 개발 toolchain은
Go `1.26.7`이다.

```sh
go build -trimpath -o ./bin/helox ./cmd/helox
./bin/helox --help
```

dependency와 품질 도구까지 재현하려면 저장소 루트에서 다음을 실행한다.
품질 도구는 source tree 밖 전용 cache에 exact version으로 설치된다.

```sh
go run ./scripts/check bootstrap
go run ./scripts/check quick
```

## Commands

지원하는 공개 입력은 anonymous public npm Registry, PyPI와 public GitHub
Release asset이다. npm/pip install은 기본적으로 현재 managed project 또는
active virtual environment를 사용하며, `--target`은 고급 absolute destination
override다. GitHub install은 M9-005 전까지 `--target`을 요구한다.

```sh
helox npm inspect '<package>[@<version>]'
helox npm install '<package>[@<version>]'
helox npm install '<package>[@<version>]' --target /absolute/project

helox pypi inspect '<project>[@<version>]'
helox pip install '<project>[@<version>]'
helox pip install '<project>[@<version>]' --target /absolute/venv

helox github inspect '<owner>/<repo>@<tag>#<asset>'
helox github install '<owner>/<repo>@<tag>#<asset>'
helox github install '<owner>/<repo>@<tag>#<asset>' --target /absolute/new-target
```

`inspect`는 `ALLOW`여도 target에 반입하지 않는다. `install`은 모든 필수
검사와 Policy가 `ALLOW`인 exact Artifact set만 offline Promotion한다.
`MANUAL_REVIEW`, `BLOCK`, 불완전한 검사, cleanup 불확실성에서는 target을
게시하지 않는다.

## Supported paths

| Host / input | MVP support |
| --- | --- |
| Linux amd64 + pinned Docker/runsc/observer | npm, PyPI wheel/sdist와 GitHub Release ELF/ZIP/tar.gz 검사; 정책이 허용한 exact set의 새 target Promotion |
| macOS native | CLI build와 기본 실행만 검증됨. Linux dynamic backend와 automatic Promotion은 지원하지 않음 |
| Windows 11 + WSL2 Ubuntu 24.04 | WSL2 내부 Linux CLI build와 기본 실행만 검증됨. Windows-native backend가 아니며 dynamic/Promotion support를 뜻하지 않음 |
| 기타 OS/architecture/runtime | 지원하지 않으며 누락된 capability를 안전으로 간주하지 않음 |

전체 Linux 경로는 [runtime lock](./scripts/runtimes.lock.json)의 exact Docker,
gVisor source·runsc, Bazel, Node 및 Python identity 경계를 요구한다. `runsc-trace`는
[pod-init config](./tools/gvisor-observer/pod-init.json)로 보호된 shared observer
socket을 sandbox 시작 전에 연결해야 한다. 현재 저장소는 일반 Host용 runtime
installer를 제공하지 않는다. 이 구성이 없는 Host에서 동적 검사나 automatic
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
