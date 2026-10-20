# Native 최초 설치 인계 — Linux/WSL2 amd64

상태: **소스 연결·로컬 검증 / 실제 서명된 새 설치와 systemd 검증 미완료**.
현재 개발용 설치에 실행하는 migration 절차가 아니다. [자체 업데이트 설계](./07-self-update.md),
[M10 신뢰 계약](../planning/13-m10-verified-distribution-bootstrap-contract.md),
[현재 Queue](../planning/02-current-work-queue.md)를 따른다. 아래 V는 승인된 exact
version이고 U/G는 실제 사용할 일반 계정의 숫자 UID/GID다. 예시를 실제 값으로 검토·치환한다.

## 1. 일반 사용자 파일 준비

새 소스의 `helox bootstrap prepare --version vX.Y.Z`는 공식 공개 릴리스의 고정된
세트를 획득하고 embedded trust로 출처와 전체 selected subject를 검증한다. 기존 개발용
설치본에는 이 명령이 아직 반영되지 않았다. 현재 공개 release 목록에 적합한 signed
native9가 없으면 파일 준비는 실패해야 하며, 다른 버전이나 임의 URL로 대체하지 않는다.

성공 리포트의 `PREPARED / VERIFIED / NOT_PERFORMED / PENDING`은 각각 파일 준비,
출처 검증, **검역 미수행**, **설치 대기**다. AI나 content safety 승인 결과가 아니다.
출력 directory에는 일반 계정 소유0700 디렉터리와0600 파일만 있으며 실행하지 않는다.
명시한 version과 compiled runtime lock 전체 SHA가 맞아야 한다. Root로 실행하지 않는다.

## 2. 관리자의 독립 검증·고정 사본 인계

처음 신뢰할 CLI/helper는 그 자신이 보고하는 성공으로 Root 실행 권한을 얻지 않는다.
관리자는 별도로 신뢰한 검증 도구·trust root로 공식 repository/tag/source/workflow/run과
attestation의 서명·전체 selected subjects를 확인한다. 단순 SHA 비교만으로 출처를
입증하지 않는다. 기존 [M13 공개 서명 검증 근거](../planning/evidence/m13-003-release-build/qualification-20261017.md)를
따르되 과거 v0.1.0의8개 asset 서명으로 새9개 asset을 승인하지 않는다.

관리자는 아래 **실제 Root 보호 사본**에 대해 재검증을 마친 뒤에만 helper를 실행한다.
사용자 파일을 검증한 후 그 mutable 경로를 sudo로 다시 여는 방식은 사용하지 않는다.
Directory와 모든 parent는 Root 보호·no symlink이며 입력 파일은 regular/single-link다.

| 위치 | 소유권·mode | 내용 |
| --- | --- | --- |
| `/var/lib/heliopause/install-staging/bootstrap` | Root0:0,0700 | exact native4+metadata3만 |
| 위 native4 | Root0:0,0500 | `helox-linux-amd64`, `haa_gvisor_observer-linux-amd64`, `haa-network-policy-helper-linux-amd64`, `haa-release-installer-linux-amd64` |
| 위 metadata3 | Root0:0,0600 | `helox-release-manifest.json`, `helox-release-sbom.cdx.json`, `helox-runtime-images.json` |
| `/var/lib/heliopause/proofs/<manifest-sha256>/<proof-sha256>.json` | Root0:0,0444 | 검증한 provenance bundle; public parent0755 |

Native runtime은 기존 고정 Docker/runsc·companion manifest·pod-init·image digest 설정을
사용한다. 자동 runtime 설치나 lock 변경은 제공하지 않는다. 실제 ordinary 계정의 Docker
접근 권한과 pinned image 준비를 별도로 확인한다. `/usr/libexec/heliopause`, OS package
`/usr/lib/heliopause/releases/`, 다른 managed state/foreign files는 자동 인수하지 않는다.
현재 개발 PC에는 기존 설치가 있으므로 clean-host 시험이나 별도 검토한 migration이 필요하다.

## 3. Root 설정·파일 준비

파일 준비 전에 아래 Root-owned0644 설정과 exact native unit이 있어야 한다. JSON은
`encoding/json.Marshal`과 동일한 키 순서·compact bytes이며 **끝 개행도 없어야 한다**.
U/G/V는 관리자 검토한 실제 값이다. Runtime endpoint는 실제 local Unix socket이다.

`/etc/heliopause/network-policy.json`:

```json
{"socket_path":"/run/heliopause-network-policy/network-policy.sock","client_path":"/usr/lib/heliopause/versions/V/helox-linux-amd64","client_uid":U,"client_gid":G}
```

`/etc/heliopause/host-tools.json`:

```json
{"docker_path":"/usr/bin/docker","docker_endpoint":"unix:///var/run/docker.sock","observer_helper_path":"/usr/lib/heliopause/versions/V/haa_gvisor_observer-linux-amd64"}
```

Docker executable은 검증된 `/usr/bin/docker` 또는 `/usr/local/bin/docker`다. Unit은
[현재 canonical 생성 함수](../../internal/releaseactivation/config_linux.go)의
`nativePolicyUnit(V)`와 byte-for-byte 같아야 한다. `/etc/systemd/system/haa-network-policy-helper.service`에
Root0:0,0644로 준비하며 drop-in이나 distribution package unit을 인수하지 않는다.
그 함수는 Root helper의 `serve`, 고정 runtime directory0755, NoNewPrivileges, Restart=no,
control-group stop과15초 stop timeout을 지정한다. 문서에 두 번째 unit template을 복제하지 않는다.

독립 검증한 고정 stage helper의 `bootstrap-files`를 관리자 Root0/GID0로 실행한다.
이 명령은 native versions/current와 잠금을 준비하며 설정·서비스를 만들거나 시작하지 않는다.
성공 출력은 `native-files=PREPARED registration=PENDING health=UNVERIFIED`다. 같은
intent/manifest의 중단 재시도만 가능하고 foreign state는 보존한다. 이때 일반 CLI는
미등록 상태이므로 runtime 실행을 허용하지 않는다.

## 4. 서비스·등록·일반 계정 확인

`bootstrap-files`가 완료된 뒤 canonical gVisor runtime bundle과 pod-init, 고정 runtime
images를 준비한다. 최초 파일 준비는 `/usr/libexec/heliopause`의 기존 설치를 거부하므로,
그 아래의 gVisor bundle을 먼저 배치하지 않는다. Runtime 설치 뒤 파일 준비를 반복하지
않고 다음 등록 단계로 진행한다. 기존 runtime lock과 설치 신원 검증을 그대로 따른다.

관리자는 exact unit의 daemon-reload/start와 실제 service identity/socket/cgroup를 확인한 뒤
**설치된** `/usr/lib/heliopause/versions/V/haa-release-installer-linux-amd64 register`를 실행한다.
Register는 설치 서명·전체 bytes·실제 helper inode·canonical configs·runtime·drain·잠금을
재확인한 뒤에만 managed record를 게시한다. 서비스를 대신 시작하거나 unknown 상태를
부재로 간주하지 않는다. Stage helper의 실행이 종료된 뒤 등록하며, 검증한 initial source의
정리는 register가 담당한다. Unknown source를 강제 삭제하지 않는다.

설정한 일반 UID/GID로 설치된 version의 `helox-linux-amd64 doctor`와 기존 nominal
`npm inspect is-number@7.0.0` 검역을 실행해 runtime/required checks/Evidence/Policy를
확인한다. Register 성공만으로 전체 bootstrap acceptance를 닫지 않는다. 로그인 PATH와
공개 launcher 연결도 관리자가 실제 실행 파일 신원과 함께 확인한다.

이후 `helox self-update --version vX.Y.Z`는 기존 controller가 검사한 세트만 좁은 Root
installer에 전달한다. Client는 `/usr/bin/sudo -n -- <실제 이전 설치 helper> serve`를 사용하므로
기존 OS 관리자 실행 권한과 유효한 인증이 필요하다. 관리자 계정은 같은 터미널에서
먼저 `sudo -v`로 인증한 뒤 일반 권한으로 self-update를 실행한다. 인증이 없거나 만료되면
실패하며 JSON/report 입력으로 비밀번호를 받지 않는다. 기존의 이 계약을 재사용하며 새
sudoers 규칙이나 version별 위임 파일을 필수조건으로 추가하지 않는다. 전체 CLI의 sudo/
setuid는 사용하지 않는다. 실제 signed/systemd 시험에는 OS 인증 성공·실패도 포함한다.

## 5. 완료·정리 기준

새 signed native9 base/target의 실제 획득→검역→전환→doctor/nominal, BLOCK/취소·동시성,
중단/재부팅 복구와 기존 OS 관리자 인증을 확인해야 전체 기능 완료다. 실제 signed 입력의 파일 준비·등록·systemd·일반 doctor/npm과
보완 후보의 native4 동적 호환성까지 확인했다. 새 signed source의 전체 활성화·중단 복구와
공개 획득 경로는 아직 미완료다. [실제 범위](../planning/evidence/m13-004-convenience/signed-native-qualification-20261020.md)를
따르며 시험 태그 생성은 공개 Release·main merge·기존 개발 설치 변경을 뜻하지 않는다.

최종 사용 뒤 완료된 ordinary export/test binaries/cache/operator를 정리한다. 현재·이전
정상 release, recovery journal, public proof와 원본 Evidence는 보존한다. UNKNOWN/실행 중
참조나 foreign replacement를 디렉터리 이름만으로 지우지 않는다.
