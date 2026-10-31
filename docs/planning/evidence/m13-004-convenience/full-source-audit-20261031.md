# 이름 결정 전 전체 코드·보안 감사

Status: IN_PROGRESS. Owner: M13-004. 사용자 2026-10-31 요청에 따라 전체 저장소 감사 범위를 명시적으로 확대한다. 이름 결정·등록·게시·main merge는 포함하지 않는다.

## 기준선과 완료 기준

- Source: `21a0a25214cce425d1204c63a47935170f68ca71`, branch `milestone/m13-release-build-packages`; 원격보다 local 두 commit 앞. 감사 시작 시 tracked 수정은 Queue뿐이며 기존 untracked 자료는 보존한다.
- 기존 구현: IMPLEMENTED=YES / WIRED=YES는 원래 bounded P2..P7 기능 범위에만 해당한다. 새 감사나 새 signed release의 완료 판정이 아니다.
- 이번 감사: IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO.
- MISSING: 전체 inventory/자동 검사 범위 매핑, 미사용·중복·복잡도 검토, 신뢰 경계별 사람 검토, 현재 취약점 DB·secret history 검사, 확정 문제의 최소 교정과 회귀, 최종 변경/검증/미검증 경계 대조.
- 완료는 모든 maintained owner의 검토/도구 coverage와 finding disposition, 필요한 교정/회귀를 연결했을 때만 판정한다. 알려진 취약점이 발견되지 않거나 테스트가 통과해도 모든 버그 부재를 증명하지 않는다.

## 범위와 방법

Tracked inventory: Go production681파일119060줄 / Go test761파일129623줄 / C++2파일17093줄 / shell18파일1760줄 / Python6파일293줄 / YAML6파일1482줄. Docs 안의 재현 script도 포함한다. 외부 gVisor 전체 upstream source는 이 저장소의 자체 코드가 아니며, 유지하는 patch와 호출/검증 경계는 포함한다.

| 검토 단위 | 자동 검사 / 기존 근거 | 사람 검토와 남은 일 | 상태 |
| --- | --- | --- | --- |
| 전체 Go·platform/tag·의존성 | canonical Quick/security 기존21a PASS; 새 vulnerability/history 예정 | package/entry reachability, unused/duplicate, 환경·태그 누락 | OPEN |
| acquisition·adapter·archive | 기존 bounded functional/CI446 범위만 재사용 | 크기/경로/링크/redirect/인증/graph identity | OPEN |
| verification·evidence·policy | 기존 계약/회귀 | fail-closed, fresh whole binding, provenance/first trust | OPEN |
| promotion·ownership·rollback | 기존 계약/회귀 | no-replace, FD/inode, crash/concurrency, root/user 경계 | OPEN |
| sandbox·observer·helper | CPP133·기존 source/profile scope만 재사용 | argv/env/role/FD/완료 authority·cleanup·권한 | OPEN |
| bootstrap·release·CLI | 이름독립 review·unsigned19파일 조립은 별도 제한 범위 | privileged caller/provenance/초기 획득·delimiter/output | OPEN |
| checker·script·workflow·runtime pins | 고정 도구·workflow validators | CI 권한/주입/공급망·추적된 script/patch | OPEN |

Canonical high-signal gosec는 G104/G115/G122/G204/G301/G302/G304/G703를 기본 검사하지 않는다. 이번 전체 감사에서는 그 차이를 명시하고 보완 분석/사람 검토를 한다. 임의의 complexity 기준으로 함수/파일을 분할하지 않는다. 변경·새 위험이 없는 expensive native/CPP/CUDA/CI를 반복하지 않는다.

## 실행과 발견

첫 canonical vulnerability는 현재 작업 디렉터리의 보존된 미추적 Go source capture를 `./...`가 포함해 package-load 오류로 실패했다(186.879s/exit1). 이는 취약점 발견/검사 성공이 아니다. 같은 잘못된 cwd의 보완 gosec는 정확한 worker/child 신원 확인 후 중지(exit143)했고 deadcode는 미도달이다. 원본은 `/tmp/haa-full-audit-20261031/`에 보존한다. 과거 raw를 삭제·제외 규칙으로 숨기지 않고 clean detached worktree `/tmp/haa-full-audit-20261031/source`의 동일21a HEAD/3255 tracked files로 분석한다.

### F01 — 로컬 요청 framing의 읽는 동안 크기 제한 누락

`internal/networkpolicy/server.go:handle`과 `internal/hosttool/observation_resource_linux.go:serveObservationResources`는 authenticated peer 뒤에 `bufio.NewReaderSize(..., limit+1).ReadBytes`를 호출한다. Reader buffer 크기는 ReadBytes의 누적 할당 상한이 아니므로 newline 없는 입력은 사후 len 검사 전에 계속 쌓인다. Kernel peer UID·보호된 executable 인증과5초 deadline은 존재하므로 임의 Artifact의 원격 접근/권한 상승을 입증한 것은 아니다. 문제는 기존 bounded protocol 계약 위반/메모리 사용 경계다.

Edit map: 두 기존 reader에서 `ReadSlice`를 사용해 buffer-full을 기존 err 경로로 거부한다. 새 protocol/함수/authority를 만들지 않는다. 실제 Unix stream에서 newline 없는 limit+1 입력의 조기 거부와 정상 요청을 검증한다. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO (local PASS; combined hosted 미완료).

### F02 — resolver의 링크 로컬·broadcast 주소 거부 누락 — 반례 확정

`networkpolicy.NewResolverPolicy`와 `sandbox.validateResolverEndpoints`의 기존 public IPv4 조건은 private/loopback/multicast/unspecified만 검사한다. `169.254.169.254`와 limited broadcast에 대한 반례를 먼저 추가한다. Go/Cargo/PyPI/NPM의 기존 최종 공용 validator 호출을 재사용하고 predicate owner 두 곳만 고친다. DNS/Host 장악이나 실제 metadata 접근까지 입증한 것은 아니다. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO (local PASS; combined hosted 미완료).

### F03 — 반환하지 않는 Promotion 출력의 불필요한 무제한 capture

`internal/hosttool/executor.go:Executor.Run`은 `CombinedOutput`의 bytes를 버리고 고정 오류만 반환하지만 먼저 전체 stdout/stderr를 메모리에 모았다. Edit map: 같은 `exec.Cmd.Run`으로 실행해 nil output의 OS discard를 사용한다. argv/env/신원 검증·종료 상태·공개 오류는 유지하며 새 wrapper/버퍼/설정은 만들지 않는다. 실제 Artifact의 무제한 출력 공격을 재현했다고 확대하지 않는다. 기존 Promotion 소비자 회귀와 최종 native qualification을 연결한다. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO (local PASS; combined hosted 미완료).

### F04/F05 — 고정 Node/Python의 upstream 보안 수정 미반영 — OPEN

현재 Node24.21.0/npm11.19.0과 Python3.14.7/pip26.2.1은 2026-09-17 검토 값이다. [Node24.22.1 보안 릴리스](https://nodejs.org/en/blog/release/v24.22.1)는 Brotli out-of-bounds/RCE 위험 CVE-2026-111497, DNS/TLS 검증과 CA distrust 등을 수정한다. [Python3.14.10 보안 릴리스](https://www.python.org/downloads/release/python-31410/)는 CVE-2026-115557의 compromised CA key distrust와 CVE-2026-114583의 tar filter 수정 등을 포함한다. 기존 Go govulncheck PASS는 외부 runtime을 검사하지 않는다. 현재 image 실제 버전/contents를 읽고 기존 update-runtime plan/apply/verify·같은 ABI의 exact security target과 support review로 수정한다. Python/pip 공통 runtime 변경에는 CPU/각 지원 CUDA의 새 qualification이 필요하므로 개별 helper별 CI 없이 최종 묶음으로 수행한다. Host 영구 활성화·새 signed 배포는 별도다.

[Bun advisory](https://github.com/oven-sh/bun/security/advisories/GHSA-663w-cv8g-jj2p)의 fixed1.4.4와 [UV advisory](https://github.com/astral-sh/uv/security/advisories/GHSA-hq85-5cpj-mr62)의 fixed0.13.4는 현재 pin과 일치한다. [PNPM standalone advisory](https://github.com/pnpm/pnpm/security/advisories/GHSA-w8f5-6m3h-r453)의 affected10/11 범위를 현재12.15.0 자체의 취약성으로 확대하지 않는다. 이전 버전을 내려받는 자동 package-manager 전환은 기존 격리/고정 경계를 따로 대조한다.

### F06 — observer endpoint가 inherited umask에 의존 — 원인 확정 / OPEN

첫 local Quick397.446s는 compile/vet/Staticcheck까지 통과했으나 default sandbox의 observer readiness6건이 실패했다. 전체 성공이나 F01/F02의 회귀 원인으로 쓰지 않는다. 실제 inherited umask는0002다. 동일 sandbox test binary의 기존 `TestObserverSupervisorOwnsOneReceiverHelperAndEndpoints`는 umask0002에서0.014s FAIL,0022에서0.013s PASS다. `newSharedObserver`는 socket을 만들고 명시 mode를 설정하지 않아0775가 되는데 supervisor는 group-write를 거부한다. 기존 helper C++ main의 remote/control bind도 inherited umask를 그대로 따른다.

Edit map: Go의 기존 생성 경로에서 owner-only socket mode를 명시하고, 단독 helper process의 C++ main에서 bind 전0077 umask를 명시한다. Global Go umask 변경·permission guard 완화·새 launcher는 하지 않는다. 기존 Go supervisor 정상/정리 테스트를 subprocess umask matrix로 확장하고 기존 C++ actual-main fixture의 소켓 mode를 확인한다. CPP가 바뀌므로 예전133 latches/old binary는 새 후보의 qualification이 아니다. 기존 pinned builder로 새 CPP 전체를 한 번 모아 검사한다. F04/F05 runtime 수정과 최종 CI를 합친다.

### F07 — Docker/containerd의 보안 패치 미반영 — OPEN

현재 minimum/CI Docker29.8.1와 Noble containerd2.3.5 이후 [Docker 공식 보안 수정](https://docs.docker.com/engine/release-notes/29/)이 있다.29.8.2는 malicious DNS의 insecure registry fallback과 OCI index 처리 등을,29.9.1/containerd2.4.2는 registry 입력에 의한 crash/disk exhaustion을,29.9.2는 compromised CA 검증을 수정한다. 정확한 공식 Noble package metadata를 확인한 뒤 기존 lock/generator/support owner에서 minimum·CI package를 함께 고정한다. 새 updater component나 임의 daemon을 만들지 않는다. 로컬 Host daemon의 전역 교체/재시작은 이 source 수정과 별도이며, 기존 Host version을 새 안전 기준의 qualification으로 쓰지 않는다. 새 hosted environment에서 필수 actual 검증을 모은다.

## 첫 source 교정 checkpoint — 전체 감사 완료 아님

`/tmp/haa-full-audit-20261031`의 원본을 따른다. 기존 source owner14개만 수정했고 새 production 함수·package·interface·dependency는 없다. F01은 실제 Unix stream의 정상 요청/oversized no-newline 반례, F02는 두 기존 validator 반례가 RED였고 교정된 표적8.173s가PASS다. F03은 버리는 capture를 기존 Cmd.Run으로 교체했다. 미사용 분석에서 전 platform/tag의 소비자가 없는 Go `Endpoints` accessor를 제거했고 updater의 유일한 G104 finding인 lock 실패의 Close 오류를 첫 원인과 결합했다(기존 updater 회귀9.648s PASS).

| 실제 검사 | 결과와 범위 |
| --- | --- |
| clean21a canonical vulnerability / security-history | PASS25.301s /97.359s; 현재 Go 호출 graph·reachable Git history, runtime image CVE 검사가 아님 |
| all tracked Go AST | 1442파일 parse 오류0, production4568함수; 길이/분기 수는 검토 위치 선정용이며 결함 판정이 아님 |
| official x/tools0.49.0 deadcode | Linux default/all explicit tags·Darwin arm64 분석 성공; Linux all-tags 미도달3개 중 Endpoints 제거, 나머지2개는 Run의 기존 trace model accessor로 분리 검토. Production-only94개를 테스트·다른 platform의 소비자 확인 없이 삭제하지 않음 |
| 문맥 규칙 gosec | 정상 분석/346 diagnostics(exit1): G115121/G70353/G1227/G20413/G304112/G30222/G30117/G1041. 전체 clean으로 보고하지 않으며 ownership/검증 경계와 disposition을 계속 대조 |
| shell/Python syntax | tracked24개 오류0; shellcheck 등 미고정 도구를 임의 추가하지 않음 |
| first local Quick | FAIL397.446s: compile/vet/Staticcheck PASS 뒤 default observer readiness6건. 같은 binary umask counter로 F06 원인 확정; Race/security는 이 operator에서 미도달 |
| F06 포함 표적 / 영향 세 package Race / security | PASS8.173s /40.935s /24.616s. 최종 pin 변경 뒤의 whole Quick/hosted 결과와 구분 |
| new C++ full | PASS55.167s /133 PASS, FAIL0; 동일 SDK/기존 builder cache 재사용. Source47d8a68e/3db99fb4, pod-init 불변. 실제 new helper2fc10cd2는 umask000에도 두 socket0700/readiness PASS; 정확한 자식 종료·회수 |

구조 검토에서12줄 이상 정확한 본문 중복은 PNPM/Bun CheckWorkspace 한 쌍뿐이다. 동일한 짧은 흐름이 서로 다른 BeginWorkspace/Add의 회계 의미를 호출하며 공통 engine은 이미 재사용한다. 이를 없애기 위한 새 interface/함수는 만들지 않는다. 긴 composition root는 기존 ecosystem별 concrete wiring을 소유하고, transaction/console binder의 분기는 journal·각 metadata/identity 거부 조건을 소유한다. 줄 수만을 이유로 쪼개거나 future consumer를 연결하지 않는다.

현재 실제 Node image는v24.21.0/Brotli1.2.0/OpenSSL3.5.8, Python image는3.14.7/OpenSSL3.0.20이다. Host mount·network 없는 일회성 read-only probe로 확인했으며 runtime source fix는 아직 미적용이다. Node V1 support input의 npm 추정을 updater가 거부했고 원본을 보존했다. 공식 exact index에 맞춘 V2 plan은Node24.22.1/npm11.19.1 및 `node:24.22.1-slim@sha256:e1582112fb9f152ed38eefcd712efc93fee1becb519fdfa57574d5e07eaa5e7d`를 고정했다. Updater가 읽는 clean e3의 세 pin owner는21a와 같고, 별도 candidate에만 적용한다. 이 준비를 main 최종 적용/검증·Host 활성화로 쓰지 않는다. MISSING은 runtime 패치·잔여 진단 disposition·최종 frozen 전체 검증이다.
