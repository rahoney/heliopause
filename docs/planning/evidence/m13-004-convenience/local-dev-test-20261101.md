# 동일 WSL·micromamba 개발 검증 환경

Status: COMPLETE — local development test scope. Owner: M13-004의 사용자 요청 개발 환경 설정.

사용자는 기존 WSL의 micromamba `helox` 환경에서 다른 터미널로 최신 UI와 패키지 설치·업데이트를 직접 시험하도록 설정을 승인했다. 공식 릴리스·본체 self-update·공개 배포는 이 범위 밖이다. 제품 소스와 보안 경계는 변경하지 않는다.

## 시작 baseline

Source: `4288dddad065e5bb9b0b90f3cf3740e000dbc705`. micromamba prefix는 `$HOME/micromamba/envs/helox`, Python3.14.7이며 helox는 기존 root 소유 `/usr/libexec/heliopause/helox`를 가리킨다. Docker29.8.1은 current minimum29.9.2 미달이다. 컨테이너0, 정책 서비스 inactive/disabled다. 고정 이미지5개와 gVisor6개는 현재 lock에 일치해 재사용한다. 현재 observer/pod-init/CLI/helper는 최신 소스 세트로 전환해야 한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 개발 설치와 격리 owner | YES | YES (이전 설치) | YES (이전 범위) | YES (이전 범위) | 재구현 없음 |
| current source의 사용자 체험 환경 | YES | YES | YES | YES | MISSING=[]; 아래 실제 환경·소비자 결과 |

Read: [M8 Host trust](../../11-m8-production-trust-hardening-contract.md), [M10 배포 계약](../../13-m10-verified-distribution-bootstrap-contract.md), [Native bootstrap 경계](../../../user-journey-cli-ia/08-native-bootstrap.md). 기존 `.github/workflows/heliopause-ci.yml`의 개발 설치 위치와 `deploy/systemd/haa-network-policy-helper.service`를 재사용한다. 최초 native 정식 등록을 흉내 내거나 release 기록을 만들지 않는다.

## 적용·검증

기존 설치를 백업하고 Docker/CLI를29.9.2, containerd를2.4.2로 전환했다. 최신 CLI와 helper·검증된 observer4ca33a74·현재 pod-init을 적용했다. gVisor6개와 고정 image5개는 lock 지문을 확인해 재사용했다. 기존 정책 서비스를 활성화해 WSL의 다른 터미널에서도 사용할 수 있다. 제품 수정은 기존 unit의 ExecStart에 필요한 `serve` 인자를 추가한 한 줄뿐이다. systemd-analyze verify와 실제 running·두 endpoint root:1000/0660을 확인했다.

첫 진단은 기존 unit이 helper 도움말만 출력하고 exit0으로 종료해 정책 서비스가 unavailable이었다. 첫 Python 명령은 검사 전 tool-parent 오류로 중단됐고, unit의 serve 인자 교정 뒤 실제 설치·업데이트가 성공했다. 별도 Go 첫 명령은 실행 기록기의 미지원 강제 JSON 옵션을 시작 전에 거부했다. 제품 출력 계약을 바꾸거나 완료한 Python 검사를 반복하지 않고 Go 옵션만 제거해 다음 세 단계를 실행했다. 최초 실패 원본도 raw에 보존했다.

| 실제 helox micromamba 소비자 | 결과 | 경과 |
| --- | --- | --- |
| packaging24.2 설치 | COMPLETED/ALLOW/Promotion COMPLETED; 필수3검사·Evidence3 | 27.245s |
| packaging24.2→25.0 업데이트 | COMPLETED/ALLOW/Promotion COMPLETED; 필수3검사·Evidence3 | 29.149s |
| Go uuid v1.5.0 설치 | exit0, module1과 lock 게시 | 21.527s |
| Go uuid 업데이트 | v1.6.0 COMPLETED/ALLOW/Promotion COMPLETED, 필수2검사·Evidence2 | 19.521s |
| 실제 Go build ./... | COMPLETED/ALLOW, 격리 관찰과 build output 게시 | 67.366s |

최종 Python metadata는25.0 한 개이며 이전24.2는 없다. Go controls는uuid1.6.0이다. Docker container0·관찰 runtime directory empty·설치 지문 동일과 정책 서비스 active/enabled를 확인했다. 사용자 체험용 Python venv는 비어 있고 Go 폴더에는 아직 의존성이 없다. micromamba의 Python3.14.7과 개발 패키지를 변경하지 않았다.

Root 백업의 before/stage/applied와 실제 로그·지문·completion은 `docs/github-logs/local-dev-test-20261101`에 보존한다. 동일 source의 full/native/CPP/CUDA/CI는 반복하지 않았으며 새 remote CI도 없다. 문서 전달 검사 결과는 해당 로컬 commit과 raw closure에 기록한다.

## 다른 터미널에서 직접 체험

기존 WSL에서 다음 명령을 실행한다. CLI와 root helper는 WSL 시스템에 설치돼 있으며 micromamba 환경에는 의존성을 추가하지 않았다.

```sh
micromamba activate helox
helox
```

같은 prefix의 `micromamba run -n helox helox ...`로도 실행한다. 시작 화면은 확정한 최신 아트이며 방향키/Enter로 이동하고 q로 종료한다. 메뉴의 예시는 설명이며 패키지 명령을 자동 실행하지 않는다.

Python 체험은 준비한 별도 빈 Python3.14 가상환경을 명시한다. 개발용 micromamba 환경을 설치 대상으로 쓰지 않는다.

```sh
cd "$HOME/helox-tryout-20261101/python"
helox pip install packaging@24.2 --target "$PWD/.venv"
helox pip install --upgrade 'packaging==25.0' --target "$PWD/.venv"
```

Go 체험은 준비한 작은 UUID 출력 프로그램을 사용한다. Host Go 설치나 go mod init은 추가로 필요하지 않다.

```sh
cd "$HOME/helox-tryout-20261101/go"
helox go get github.com/google/uuid@v1.5.0
helox go get -u github.com/google/uuid
helox go build ./...
```

Go 기본 get/build는 기존 text 출력 계약을 유지하므로 강제 `--output json`을 붙이지 않는다. 작은 source라도 검사·격리 실행 때문에 각 명령에는 시간이 필요하다. 먼저 위 두 경로를 체험하고 PyTorch는 CPU profile의 별도 venv에서 이어간다. CUDA 전체 재검증은 이번 준비의 필수조건이 아니다.

`helox doctor`는 trusted-host-runtime/observer-helper/network-policy-service/runtime-images 네 항목이 OK여야 한다. 공식 release 기록은 만들지 않았으므로 RELEASE_INSTALLATION_UNAVAILABLE와 doctor exit1은 남는다. 이를 공식 설치·본체 self-update qualification으로 바꾸거나 경고를 숨기지 않는다.

## 정리와 복원

사용자 체험 폴더는 `$HOME/helox-tryout-20261101` 한 곳이다. 체험 뒤 그 폴더만 삭제할 수 있다. user cache의 검사 Evidence/intake/staging과 개발 micromamba 환경은 폴더 삭제와 별개이며 자동 일괄 삭제하지 않는다.

기존 설치·설정7개와 이전 Docker 패키지3개는 root 보호 경로 `/var/backups/heliopause/dev-test-20261101-4288ddd`에 보존했다. 실행 중인 helox/관찰기와 Docker 작업을 모두 끝낸 뒤 관리자가 다음으로 원래 개발 설치·Docker29.8.1·containerd2.3.5·서비스 disabled/inactive 상태로 복원할 수 있다.

```sh
sudo /usr/bin/python3 /var/backups/heliopause/dev-test-20261101-4288ddd/restore.py
```

이 복원 파일은 구문만 확인했으며 이번 설정을 되돌리는 실제 실행은 하지 않았다. 검증 전 원본 파일·패키지 hash를 확인하고 active 작업이 있으면 거부한다. 사용자 체험 폴더·Evidence는 보존한다. stale/unbound socket은 복원 대상이 아니다.
