# 이름 결정 전 전체 코드·보안 감사

Status: COMPLETE / IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES — 저장소 자체 코드·보안 감사 범위. Owner: M13-004. M13 전체 완료나 새 서명 배포의 판정은 아니다.


## 최종 감사 결과

검증한 코드 소스는 `358adf9b6aa8e7b237262d25588ee4eaa544d933`, 전체 tree는 `e1cac13504028e8bb4ab1fa8677336eaebb8087f`다. [단일 최종 증거](./full-source-audit-20261031.json)에 검토 범위·발견 사항·각 실행의 실제 소스와 원본 해시를 연결했다. 이후 완료 기록은 문서만 변경한다.

| 확인한 문제 | 최소 수정과 판정 |
| --- | --- |
| F01 소켓 요청 읽기의 할당 상한 누락 | 기존 reader의 bounded framing과 실제 정상/초과 입력 회귀 — CLOSED |
| F02 공개 resolver 주소 검증 누락 | 기존 preflight/typed constructor의 특수 주소 거부, 공개 예외 보존 — CLOSED |
| F03 반환하지 않는 출력의 무제한 capture | 같은 Cmd.Run의 discard 사용 — CLOSED |
| F04/F05 Node/Python 보안 패치 누락 | 고정 image/pin·현재 fixture·pip metadata exact 경로 연결 — CLOSED |
| F06 observer 소켓의 inherited umask 의존 | Go socket mode·단독 C++0077, 기존 권한 guard 유지 — CLOSED |
| F07 Docker/containerd 보안 기준 누락 | minimum/CI package와 실제 hosted 환경 갱신 — 코드 CLOSED; 로컬 Host 전환은 별도 |
| F08 중복 Docker 버전 비교의 비정규 값 허용 | 이미 공유하는 runtimeidentity owner 한 곳에서 비교·거부 — CLOSED |

소비자 없는 GoModule.Endpoints와 버리는 출력 capture를 제거했고, updater의 lock 실패 시 Close 오류도 첫 원인과 함께 보존한다. 테스트·다른 OS·기존 trace 계약이 쓰는 코드를 무차별 삭제하지 않았다. PNPM/Bun의 짧은 흐름은 서로 다른 회계 의미와 기존 공통 engine을 유지하며, 길이만으로 함수·클래스·모듈을 나누지 않았다. 구조·중복·신뢰 경계 검토와 도구 검사 범위는 아래 disposition을 따른다.

| 동일 소스 qualification | 결과 | 실제 CPU / 선택 CUDA | 실제 관측 |
| --- | --- | --- | --- |
| [CI448 일반/CPU](https://github.com/rahoney/heliopause/actions/runs/41183473945) | 10 SUCCESS | 268.73s / 해당 없음 | 126 unique parent PASS·skip0·CPP133·helper18wait0·최종4 원복 |
| [CI449 cu126](https://github.com/rahoney/heliopause/actions/runs/41193740506) | 10 SUCCESS | 288.04s / 1270.55s | skip0·CPP133·helper2wait0·confirmed RESTORED |
| [CI450 cu130 attempt2](https://github.com/rahoney/heliopause/actions/runs/41198471800) | 10 SUCCESS | 215.77s / 855.49s | 원래 성공8 보존·Mac/Required만 재시도·skip0·정상 복원 |
| [CI451 cu132](https://github.com/rahoney/heliopause/actions/runs/41203161813) | 10 SUCCESS | 290.53s / 1064.79s | skip0·CPP133·helper2wait0·confirmed RESTORED |

총40개 qualification 로그6,691,308B의 크기/해시를 대조했다. 일반 CI의 실제 merge checkout은 코드 소스와 whole tree가 같고, 수동 CUDA 실행은 모두358을 checkout했다. 마지막 CI451은46m46s에 성공했고 실제20분 조회3회 중13:54:27.455226UTC에 terminal을 소비했다. 모든 조회 워커가 자연 exit0으로 종료했으며 활성 CI/다음 타이머는 없다. 기존 성공8개 로그를 재취득하거나 불변 CPP/native/full 검사를 반복하지 않았다.

Local configuration7·고정 formatter/compiler/vet/Staticcheck/test·보안·현재 Go vulnerability·reachable secret history와 supplied pipx/pip/npm13개의 OSV 조회를 연결했다. 보완 gosec346개는 문맥별로 검토한 결과이며 모든 규칙의 diagnostic0을 주장하지 않는다. 알려진 취약점 대조군과 응답 완결성을 확인한 OSV0도 해당 입력·조회 시각의 범위다.

감사 범위의 MISSING은 없다. 배포 전에는 로컬 Host Docker29.8.1을 새 minimum29.9.2 이상으로 전환·검증해야 한다. 이번 작업에서 전역 daemon 교체/재시작이나 영구 활성화는 수행하지 않았다. 원래 runtime component plan의 APPLIED와 현재 configuration/hosted 검증을 영구 activation으로 바꾸지 않는다. 공식 이름/publisher binding·채널·새 protected Release Build/서명·clean-Host 배포 acceptance와 main merge/tag/publish는 별도다. 기존 로컬 Buildx 미설치를 exact Release Build 성공으로 대체하지 않는다.

이 감사는 유지하는 저장소 코드·patch/호출 경계와 현재 도구/공식 advisory에 대한 검토다. 외부 upstream 전체 소스나 runtime OS package 전체의 독립 침투/CVE 검사, 모든 미공개 취약점·버그 부재를 보증하지 않는다. 아래는 최초 실패와 각 checkpoint의 당시 판정이며 최종 상태를 덮지 않는다.

## 기준선과 완료 기준

- 감사 시작 Source: `21a0a25214cce425d1204c63a47935170f68ca71`, branch `milestone/m13-release-build-packages`; 원격보다 local 두 commit 앞. 감사 시작 시 tracked 수정은 Queue뿐이며 기존 untracked 자료는 보존한다.
- 기존 구현: IMPLEMENTED=YES / WIRED=YES는 원래 bounded P2..P7 기능 범위에만 해당한다. 새 감사나 새 signed release의 완료 판정이 아니다.
- 감사 시작 baseline: IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO.
- 최종 교정/감사 연결: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES. Source358의 일반/CPU·cu126·cu130·cu132, local configuration/security와 원본·정리 대조를 완료했다.
- 시작 MISSING: 전체 inventory/자동 검사 범위 매핑, 미사용·중복·복잡도 검토, 신뢰 경계별 사람 검토, 현재 취약점 DB·secret history 검사, 확정 문제의 최소 교정과 회귀, 최종 변경/검증/미검증 경계 대조. 최종 감사 MISSING은 없다. 이름·채널·Host 전환·새 서명 배포는 아래 별도 운영 작업으로 유지한다.
- 완료는 모든 maintained owner의 검토/도구 coverage와 finding disposition, 필요한 교정/회귀를 연결했을 때만 판정한다. 알려진 취약점이 발견되지 않거나 테스트가 통과해도 모든 버그 부재를 증명하지 않는다.

## 범위와 방법

Tracked inventory: Go production681파일119060줄 / Go test761파일129623줄 / C++2파일17093줄 / shell18파일1760줄 / Python6파일293줄 / YAML6파일1482줄. Docs 안의 재현 script도 포함한다. 외부 gVisor 전체 upstream source는 이 저장소의 자체 코드가 아니며, 유지하는 patch와 호출/검증 경계는 포함한다.

| 검토 단위 | 자동 검사 / 기존 근거 | 사람 검토와 남은 일 | 상태 |
| --- | --- | --- | --- |
| 전체 Go·platform/tag·의존성 | AST1442/오류0, Linux default/all-tags·Darwin deadcode, canonical vulnerability/history PASS | package/entry reachability·중복/복잡도 검토; 아래 disposition | REVIEWED / QUALIFIED |
| acquisition·adapter·archive | 기존 bounded functional/CI446 범위만 재사용 | bounded reader·정규 경로/링크·redirect/인증·complete graph/원본 binding 검토; 새 재현 결함 없음 | REVIEWED |
| verification·evidence·policy | 기존 계약/회귀 | fresh whole/required Evidence·fail-closed·provenance와 first-trust 분리 검토; 새 재현 결함 없음 | REVIEWED |
| promotion·ownership·rollback | 보완 G122/G304·기존 회귀 | private staging·held FD/inode·no-replace/journal·root/user 경계 검토; F03 수정 | REVIEWED / QUALIFIED |
| sandbox·observer·helper | 새 CPP133 PASS·영향 Race PASS | framing·argv/env·kernel role/FD·완료 authority·cleanup 검토; F01/F02/F06 수정 | REVIEWED / QUALIFIED |
| bootstrap·release·CLI | 이름독립 unsigned 조립은 별도 제한 범위 | 기존 privileged/provenance caller·초기 획득·출력 경계 검토; UV inspect 누락은21a 수정, signed deployment 별도 | REVIEWED / QUALIFIED |
| checker·script·workflow·runtime pins | 추적된 shell/Python24개 syntax PASS·workflow PASS | CI 권한/고정 argv·공급망 pin/advisory 검토; F04/F05/F07/F08 수정 | REVIEWED / QUALIFIED |

Canonical high-signal gosec는 G104/G115/G122/G204/G301/G302/G304/G703를 기본 검사하지 않는다. 이번 전체 감사에서는 그 차이를 명시하고 보완 분석/사람 검토를 한다. 임의의 complexity 기준으로 함수/파일을 분할하지 않는다. 변경·새 위험이 없는 expensive native/CPP/CUDA/CI를 반복하지 않는다.

## 실행과 발견

첫 canonical vulnerability는 현재 작업 디렉터리의 보존된 미추적 Go source capture를 `./...`가 포함해 package-load 오류로 실패했다(186.879s/exit1). 이는 취약점 발견/검사 성공이 아니다. 같은 잘못된 cwd의 보완 gosec는 정확한 worker/child 신원 확인 후 중지(exit143)했고 deadcode는 미도달이다. 원본은 `/tmp/haa-full-audit-20261031/`에 보존한다. 과거 raw를 삭제·제외 규칙으로 숨기지 않고 clean detached worktree `/tmp/haa-full-audit-20261031/source`의 동일21a HEAD/3255 tracked files로 분석한다.

### F01 — 로컬 요청 framing의 읽는 동안 크기 제한 누락

`internal/networkpolicy/server.go:handle`과 `internal/hosttool/observation_resource_linux.go:serveObservationResources`는 authenticated peer 뒤에 `bufio.NewReaderSize(..., limit+1).ReadBytes`를 호출한다. Reader buffer 크기는 ReadBytes의 누적 할당 상한이 아니므로 newline 없는 입력은 사후 len 검사 전에 계속 쌓인다. Kernel peer UID·보호된 executable 인증과5초 deadline은 존재하므로 임의 Artifact의 원격 접근/권한 상승을 입증한 것은 아니다. 문제는 기존 bounded protocol 계약 위반/메모리 사용 경계다.

Edit map: 두 기존 reader에서 `ReadSlice`를 사용해 buffer-full을 기존 err 경로로 거부한다. 새 protocol/함수/authority를 만들지 않는다. 실제 Unix stream에서 newline 없는 limit+1 입력의 조기 거부와 정상 요청을 검증한다. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES (최종 동일 소스 네 profile 검증은 아래 원본 요약 참조).

### F02 — resolver의 링크 로컬·broadcast 주소 거부 누락 — 반례 확정

`networkpolicy.NewResolverPolicy`와 `sandbox.validateResolverEndpoints`의 기존 public IPv4 조건은 private/loopback/multicast/unspecified만 검사한다. `169.254.169.254`와 limited broadcast에 대한 반례를 먼저 추가한다. Go/Cargo/PyPI/NPM의 기존 최종 공용 validator 호출을 재사용하고 predicate owner 두 곳만 고친다. DNS/Host 장악이나 실제 metadata 접근까지 입증한 것은 아니다. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES (최종 동일 소스 네 profile 검증은 아래 원본 요약 참조).

후속 [IANA 특별 주소 대조](https://www.iana.org/assignments/iana-ipv4-special-registry)는 GlobalUnicast가 공개 routability와 같지 않음을 확인했다. 새 정확한 입력100.100.100.200(shared space)/0.1.2.3/198.18.0.1/240.0.0.1의 기존 두 validator 반례를 추가했다. Edit map: 추가 IANA 검사는 기존 typed policy constructor 한 곳에 두어 client encode와 privileged decode가 함께 소비한다. Sandbox의 기본 preflight는 유지하고 기존 Bootstrap adapter/Create 경로가 강한 typed 검사를 반드시 거친다. Sandbox→networkpolicy 새 의존성이나 architecture allowlist 확대를 만들지 않는다. Public anycast192.0.0.9/.10은 원래 공개 주소 의미를 유지한다. 새 network service/config/package/helper는 만들지 않는다.

### F03 — 반환하지 않는 Promotion 출력의 불필요한 무제한 capture

`internal/hosttool/executor.go:Executor.Run`은 `CombinedOutput`의 bytes를 버리고 고정 오류만 반환하지만 먼저 전체 stdout/stderr를 메모리에 모았다. Edit map: 같은 `exec.Cmd.Run`으로 실행해 nil output의 OS discard를 사용한다. argv/env/신원 검증·종료 상태·공개 오류는 유지하며 새 wrapper/버퍼/설정은 만들지 않는다. 실제 Artifact의 무제한 출력 공격을 재현했다고 확대하지 않는다. 기존 Promotion 소비자 회귀와 최종 native qualification을 연결한다. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES (최종 동일 소스 네 profile 검증은 아래 원본 요약 참조).

### F04/F05 — 고정 Node/Python의 upstream 보안 수정 미반영 — 수정·검증 완료

현재 Node24.21.0/npm11.19.0과 Python3.14.7/pip26.2.1은 2026-09-17 검토 값이다. [Node24.22.1 보안 릴리스](https://nodejs.org/en/blog/release/v24.22.1)는 Brotli out-of-bounds/RCE 위험 CVE-2026-111497, DNS/TLS 검증과 CA distrust 등을 수정한다. [Python3.14.10 보안 릴리스](https://www.python.org/downloads/release/python-31410/)는 CVE-2026-115557의 compromised CA key distrust와 CVE-2026-114583의 tar filter 수정 등을 포함한다. 기존 Go govulncheck PASS는 외부 runtime을 검사하지 않는다. 현재 image 실제 버전/contents를 읽고 기존 update-runtime plan/apply/verify·같은 ABI의 exact security target과 support review로 수정한다. Python/pip 공통 runtime 변경에는 CPU/각 지원 CUDA의 새 qualification이 필요하므로 개별 helper별 CI 없이 최종 묶음으로 수행한다. Host 영구 활성화·새 signed 배포는 별도다.

[Bun advisory](https://github.com/oven-sh/bun/security/advisories/GHSA-663w-cv8g-jj2p)의 fixed1.4.4와 [UV advisory](https://github.com/astral-sh/uv/security/advisories/GHSA-hq85-5cpj-mr62)의 fixed0.13.4는 현재 pin과 일치한다. [PNPM standalone advisory](https://github.com/pnpm/pnpm/security/advisories/GHSA-w8f5-6m3h-r453)의 affected10/11 범위를 현재12.15.0 자체의 취약성으로 확대하지 않는다. 이전 버전을 내려받는 자동 package-manager 전환은 기존 격리/고정 경계를 따로 대조한다.

### F06 — observer endpoint가 inherited umask에 의존 — 수정·검증 완료

첫 local Quick397.446s는 compile/vet/Staticcheck까지 통과했으나 default sandbox의 observer readiness6건이 실패했다. 전체 성공이나 F01/F02의 회귀 원인으로 쓰지 않는다. 실제 inherited umask는0002다. 동일 sandbox test binary의 기존 `TestObserverSupervisorOwnsOneReceiverHelperAndEndpoints`는 umask0002에서0.014s FAIL,0022에서0.013s PASS다. `newSharedObserver`는 socket을 만들고 명시 mode를 설정하지 않아0775가 되는데 supervisor는 group-write를 거부한다. 기존 helper C++ main의 remote/control bind도 inherited umask를 그대로 따른다.

Edit map: Go의 기존 생성 경로에서 owner-only socket mode를 명시하고, 단독 helper process의 C++ main에서 bind 전0077 umask를 명시한다. Global Go umask 변경·permission guard 완화·새 launcher는 하지 않는다. 기존 Go supervisor 정상/정리 테스트를 subprocess umask matrix로 확장하고 기존 C++ actual-main fixture의 소켓 mode를 확인한다. CPP가 바뀌므로 예전133 latches/old binary는 새 후보의 qualification이 아니다. 기존 pinned builder로 새 CPP 전체를 한 번 모아 검사한다. F04/F05 runtime 수정과 최종 CI를 합친다.

### F07 — Docker/containerd의 보안 패치 미반영 — 수정·검증 완료

현재 minimum/CI Docker29.8.1와 Noble containerd2.3.5 이후 [Docker 공식 보안 수정](https://docs.docker.com/engine/release-notes/29/)이 있다.29.8.2는 malicious DNS의 insecure registry fallback과 OCI index 처리 등을,29.9.1/containerd2.4.2는 registry 입력에 의한 crash/disk exhaustion을,29.9.2는 compromised CA 검증을 수정한다. 정확한 공식 Noble package metadata를 확인한 뒤 기존 lock/generator/support owner에서 minimum·CI package를 함께 고정한다. 새 updater component나 임의 daemon을 만들지 않는다. 로컬 Host daemon의 전역 교체/재시작은 이 source 수정과 별도이며, 기존 Host version을 새 안전 기준의 qualification으로 쓰지 않는다. 새 hosted environment에서 필수 actual 검증을 모은다.

### F08 — 중복된 Docker 버전 비교가 suffix/잘못된 숫자를 허용 — 수정·검증 완료

`internal/hosttool/executor.go:atLeastVersion`과 `internal/sandbox/probe.go:atLeastVersion`은 같은3-part Sscanf 비교를 중복 구현한다. Sscanf의 성공만으로 전체 문자열이 정규 stable version임을 증명하지 않는다. `29.9.2-rc.1`/`30.-1.0` 반례를 먼저 확인한다. Edit map: 기존 비교 로직을 이미 두 caller가 참조하는 `internal/runtimeidentity`의 단일 Docker version owner로 옮기고 numeric round-trip guard를 추가한다. 두 private 중복 구현/별도 minimum alias를 제거하고 기존 caller를 직접 연결한다. 새 package/interface/범용 parser dependency를 도입하거나 runtime capability 검사를 줄이지 않는다. 기존 구현의 이동·수정이며 별도 runtime executor를 대체하지 않는다.

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

당시 old image는 Node24.21.0/Brotli1.2.0/OpenSSL3.5.8·Python3.14.7/OpenSSL3.0.20이었다. Node V1 support input의 npm 추정을 updater가 source mutation 전에 거부한 원본을 보존했다. V2 plan/apply는 Node24.22.1/npm11.19.1을, 이어진 Python plan/apply는3.14.10/pip26.2.2를 기존 세 pin owner에 적용했다. 두 plan의 before/after hash를 연결해 main에 전달한 receipt는 `runtime-two-plan-transfer.json`이다. Docker29.9.2/정확한 Noble package·containerd2.4.2는 기존 lock/support/generator에서 함께 고정했다. 원래 plan은 APPLIED이며 VERIFIED/ACTIVATED로 확대하지 않는다. 최종 합쳐진 source의 canonical configuration 검증과 원래 CPU/세CUDA qualification을 별도로 남긴다.

## 잔여 진단 disposition과 최종 후보

- F02 추가 특수 주소 RED7.717s 뒤 constructor/wire/기존 preflight 표적은12.262s PASS다. Bootstrap package는 이 selector에서 실행된 테스트가 없고, Create/Verify/Remove가 모두 typed constructor를 거치는 source 연결을 확인했다. 이를 Bootstrap 동작 테스트 통과로 쓰지 않는다. CGNAT·documentation·benchmark·reserved 거부와 공개 anycast/CGNAT 경계 정상 입력을 함께 검사했다.
- F08 RED7.795s에서 malformed/prerelease의 실제 probe Available을 확인했다. 공용 `runtimeidentity.DockerEngineSupported`로 두 private 비교 구현/alias를 제거하고 numeric canonical round-trip을 추가했다. 기존 caller dependency를 재사용하며 새 package/interface는 없다. 실제 영향을 받는 네 success fixture는 compiled minimum을 사용하고 과거 frozen fixture는 보존했다. 표적13.158s/workflow0.843s PASS다.
- Supplemental gosec346건은 기본 canonical 제외8규칙의 원본 결과이며 clean scan으로 보고하지 않는다. G115121건은 길이/양수/프로토콜·graph 상한 뒤 cast, G70353/G304112건은 typed/고정 root·regular/bounded file·held descriptor와 identity 검증을 책임 owner에서 대조했다. G1227건은 checker의 신뢰된 source tree 또는 sandbox writer가 종료된 private staging/copy owner다. 같은 UID의 Host 변경 능력을 Artifact 권한으로 가정하지 않는다. G20413건은 고정 실행 registry/argv/minimal env·보호 executable 및 developer-owned Git/tool 호출이다. 선택적 표시용 PowerShell도 고정 명령/2초·4KiB 제한이다. G30117/G30222건은 private parent 아래 cache/data·실행 파일·authenticated Unix socket의 의도된 mode다. G1041건은 lock 실패 Close 오류를 첫 오류와 결합해 수정했다. 새 suppression/allowlist나 보안 경계 완화는 없다.
- Linux all-tags 미도달3개 중 사용되지 않는 Go Endpoints accessor를 제거했다. 나머지 두 `InspectionRun` accessor는 기존 Domain trace model의 reference/artifact 계약이며 모델 필드까지 지우거나 인위적인 caller를 추가하지 않는다. Production-only94개 및 Darwin의 Linux 전용 helper는 테스트·타 플랫폼 소비가 있어 불용으로 단정하지 않는다. 긴 composition/transaction/console 함수와 PNPM/Bun 짧은 중복은 기존 책임/회계 분기를 유지했다. 실제 중복인 Docker 비교만 통합했다.
- Bun1.4.4/UV0.13.4의 공식 advisory 수정 버전은 현재 pin에 포함된다. PNPM advisory의 영향10/11계열은 현재12.15.0과 다르다. Rust1.99.1 release의 플랫폼/FFI 수정에서 현 HAA 호출의 추가 보안 결함을 확인하지 못해 관련 없는 runtime 변경을 늘리지 않았다. 이 검토는 외부 upstream 전 source/OS package 전체의 독립 penetration test가 아니다.
- 새 immutable image를 Host mount·network 없는 read-only/cap-drop/non-root probe로 확인했다: Node24.22.1/OpenSSL3.5.9, Python3.14.10/OpenSSL3.0.22/pip26.2.2 모두 exit0. Node의 Brotli 표시 버전은 여전히1.2.0이며, 공식24.22.1은 version bump 대신 `49dff6f57a` cherry-pick으로 해당 보안 수정을 포함한다. 표시 문자열만으로 미수정/1.2.1로 오기하지 않는다. 원본은 `node-security-image-probe.json`/`python-security-image-probe.json`이다.
- 현재 Host daemon은 여전히 Docker29.8.1/containerd2.3.5이며 전역 교체·재시작하지 않았다. 새 최소 기준에서 이 Host를 안전한 native qualification으로 사용할 수 없다. 새 고정 package를 설치하는 hosted qualification과 실제 Host activation을 구분한다. 공식 APT metadata의 HTTPS/hash 대조는 local 독립 signature 검증이나 signed APT install 성공으로 확대하지 않는다.

MISSING: 최종 frozen combined source의 canonical 검증, 새 공통 Python/Docker/C++ 입력에 필요한 ordinary/CPU/세CUDA hosted qualification, 결과·최초 실패/정리 분석과 최종 상태 갱신. 기존 CPP source와133 latches는 불변이므로 다시 빌드하지 않는다. 알려진 결함의 수정·검증 완료와 모든 미래 취약점/버그 부재는 다른 주장이다.

### 최종 configuration 첫 실패 — 현재 test fixture의 이전 runtime 결합

`bc96cd9`의 same-version reconciliation plan/apply는 PASS지만 verify는 bootstrap PASS 뒤 Quick FAIL이다. Format/drift/build/architecture/workflow/vet/Staticcheck는 PASS이며 default49 package가 모두 종료했다. 최초 실패는 PyPI pipx receipt의 Python mismatch다. 영향5 package(PyPI/CLI/Sandbox/checker/Promotion)의 current fixture에 old exact Python/pip와 old copied Node image가 남아 있다. Timeout으로 판정하거나 제품의 pinned-version guard를 완화하지 않는다. Edit map: 기존 current test builder의 version literal을 compiled runtime identity로 연결하고, historical native capture를 읽는 기존 test loader에서 해당 runtime field만 명시적으로 projection한다. 원본 testdata/과거 receipt bytes는 보존한다. Checker의 copied-image negative도 현재 compiled image로 구성한다. 일반 parser의 explicit historical-version 검사와 관련 없는 fixture는 수정하지 않는다. Production protocol/한도/CPP/권한은 불변이며 먼저 영향5 package 표적을 모아 검사한다. 원본은 `combined-reconcile/verification-1774467124/verify-quick.log`와 `current-fixture-failures.json`이다.

후속 current fixture 교정은29개 기존 test source에만 적용했다. 역사적 testdata/원본 capture와 production 코드는 바꾸지 않았다. 첫 표적에서 새 Node 상수 이름 오기/공유 lower-bound fixture의 metadata mismatch를 확인했고, 잘못 바뀐 lower bound는 원복해 historical parser 검사를 보존했다. V2에서 CLI/Sandbox/Promotion/checker가 PASS74.438s operator이며 PyPI의 no-op mutation 한 건만 남았다. 그 원래 lower-bound needle을 복구한 PyPI 최종 표적5.487s가PASS다. 전체 저장소의 버전 문자열을 무조건 교체하는 방식 대신 current builder·명시적 runtime field projection·실제 변조 지점을 사용했다. 최종 combined configuration은 새 frozen commit에서 재확인하며 같은 Go/tool lock의 독립 실제 cache 사본을 재사용해 도구 재빌드를 피한다. 앞 실패/후속 표적을 최종 whole Quick 성공으로 확대하지 않는다.

## 최종 local 검증과 CI447 진행

Source `f6c03493a209fdd87ac122439eb6ff8f48bdfa60`/tree `f2cf8435d7afd40a7ebe7ed6729b571dbe30785c`/3258 tracked inputs를 고정했다. 기존 updater same-version plan5.551s/apply12.145s/verify598.969s는 configuration7 profiles 모두PASS다. 실제 독립 checked source는 `combined-final/verification-11566562/source`이며 source/후보 입력은 검사 전후 일치한다. 현재 canonical vulnerability28.434s·새 커밋을 포함한 security-history155.610s도PASS다. 후자는 기존 두 official wheel MD5 exact false positive만 구분하고 모든 scanner rule을 유지했다. 원본·명령·hash는 `/tmp/haa-full-audit-20261031/final-local-qualification.json`이 소유한다.

추가 [OSV batch 조회](https://google.github.io/osv.dev/post-v1-querybatch/)는 lock hash를 확인한 Python 도구 공급 identity와 bundled pip/npm13개에서 등록 vulnerability0을 반환했다. Known vulnerable requests2.19.1 대조군은10건이며 HTTP200/전체14 results/추가 page 없음까지 확인했다. 이는 해당 DB와 명시한13개 package 범위이며 모든 runtime OS package/미공개 취약점 부재를 증명하지 않는다. `supply-osv-query/response/result.json`이 입력·원본·현재 시각·SHA를 보존한다.

정상 FF `e3b2b70→f6c0349`로 한 번 push했고 [CI447/run41180577563](https://github.com/rahoney/heliopause/actions/runs/41180577563)은 ordinary/CPU 검증이다. 실제 시작07:25:57UTC/16:25:57KST, 첫20분 due07:45:57UTC/16:45:57KST다. 기존 읽기 전용 worker `/tmp/haa-ci-20min-get-worker.py`를 재사용했다. 단일 PID1058029/start ticks10720903/session49581/config `/tmp/haa-audit-ci447-autopoll-config.json`은 actual timed queries0으로 등록됐으며 완료 조회를 미리 주장하지 않는다. 최종 status/최초 실패/정리 확인 뒤에만 다음 CUDA profile을 순차 실행한다. 새 CI source 검증/보안 floor Host activation은 아직OPEN이며 기존 성공을 전용하지 않는다.

CI 대기 중 current contract의 version 표기를 대조했다. M3/M4/M5와 P1 baseline의 과거 Node/Python/Docker qualification은 역사적 범위로 명시하고 현재 값의 owner를 기존 runtime lock으로 통일했다. ABI/platform·exact digest·fail-closed 조건은 불변이며 이 문서 정정은 진행 중CI의 frozen source 입력을 바꾸거나 추가 CI를 만들지 않는다. Historical evidence와 testdata는 수정하지 않는다.

최종 local의 원본53개/405796B는 `docs/github-logs/full-source-audit-20261031/`에 보존했다. Temporary source/cache/SDK는 복사하지 않았다. `persistent-local-raw-index.json`은 각 실제 파일 SHA/size를 소유하며 최종 canonical qualification receipt에 연결한다. 현재 raw는 기존 Git 미추적 영역이고 원본 전체가 원격에 보관됐다고 주장하지 않는다.

## CI447 최초20분 결과 — FAIL / 원인 조사

Actual timed GET1은07:45:57.000159UTC 시작/07:45:57.838467UTC 종료에 terminal failure를 확인했다. Worker session49581/PID1058029는 자연exit0, next due null이다. Seven jobs SUCCESS, macOS/Linux/Required FAIL이며 source/실제 checkout4a4a49df의 whole tree는f2cf8435로 같다. Mac 최초 오류는 새 framing test의 긴 t.TempDir socket path가 sun_path 한도를 넘어 bind EINVAL인 것이다. Edit map은 같은 test의 private short temporary directory만 변경하며 macOS skip/guard 완화는 하지 않는다. Linux는 새 Docker29.9.2 설치·CPP133·trusted executor·UV native3/npx install 뒤 첫 packaging25 target tags에서 STREAM_FAULT/OPEN_RESULT_CLASSIFICATION_IMAGE/PYTHON CONTROL/kernel UNKNOWN이다. Source 변경 뒤의 실패만으로 Python patch/umask를 원인으로 단정하지 않는다. Exact fault/source/기존CI446/새 runtime bytes를 먼저 대조하고 existing owner에서 필요한 최소 수정만 한다. 전체 ordinary cohort·CPU full·CUDA3은 NOT_REACHED다. 원본 두 로그와 actual poll metadata는 `docs/github-logs/full-source-audit-20261031/ci447-*`에 보존했고 `ci447-failure-result.json`이 결과 범위를 소유한다.

Linux 원인 대조: fault flags624640은 read-only directory open이다. 고정 old/new image의 data-only catalogue를 조회해 해당 locator의 유일한 실제 path가 새 `/usr/local/lib/python3.14/site-packages/pip-26.2.2.dist-info`임을 확인했다(old에는 없음). FNV는 lookup 진단에만 사용하며 authentication으로 쓰지 않는다. 기존 `IsPinnedRuntimeRootRead`는 exact old pip-26.2.1 dist-info 두 경로만 허용한다. 현재 source/새 immutable image/실제 fault가 대응하므로 단순한 수정 뒤 발생 추측과 구분한다. Edit map: 기존 C++ 두 exact path를 현재 bundled pip26.2.2로 맞추고 기존 latch의 현재 directory/read 정상과 old/wrong-version·neighbor·write 거부를 확장한다. Site-packages 전체/Artifact/role/lineage·network 권한은 확대하지 않는다. Mac는 같은 regression test의 짧은 private temp directory만 교정한다. 새 구조/helper/config/job 없이 두 실패를 묶고, C++ source가 실제 바뀌므로 새 full133을 한 번 실행한다. `ci447-python-locator-lookup.json`이 immutable ref/실제 path/조회 범위를 소유한다.

CI447 후속 local: 같은53-byte TMPDIR에서 실패 source의 기존 binary exit1 / 짧은 temp directory 수정본 exit0을 확인했다. 새 socket path 최대79byte로 macOS104byte 한도 안이며 test-only 변경이다. 실제 새 C++는 동일 pinned builder/Go/Bazel/자원 한도에서 full133 PASS54.526s다. Source observer2f8dde13/latch8c57dc93·새 helper4ca33a7482d9943a06aa9a134b302862a023bbfbd54bb12c2172131f115e737e를 기록했다. Pod-init/SDK/runtime lock 세 owner는 불변이다. 기존 role/provenance/Artifact 경계를 유지하고 현 bundled pip metadata read·old/neighbor/write negatives를 같은 latch 안에 추가했다. 현재 runtime pin의 source와 전 platform Go production은f6와 같으므로 f6 local configuration/vulnerability/history·OSV의 해당 입력 범위를 재사용하되 새 C++/Mac/whole hosted 성공으로 전용하지 않는다. 다음은 모인 source3개와 관련 docs의 canonical Quick/security/docs 및 새로운 필수 hosted qualification이다. 원본은 `ci447-socket-longtmp-result.json`, `ci447-cpp-followup-source-freeze/log/json`이다.

## CI448 후속 묶음 등록 — IN_PROGRESS

새 source358adf9b6aa8e7b237262d25588ee4eaa544d933/treee1cac13504028e8bb4ab1fa8677336eaebb8087f의 canonical Quick369.397s/security21.329s/release-gate0.370s/docs0.328s는 모두PASS다. Standalone checked source는 `/tmp/haa-full-audit-20261031/ci447-followup-source`로 검사 후 clean이며, 새 C++133/소켓 counter와 f6 불변 runtime·Go production의 기존 검사 범위를 구분한다. 후속 원본18개/22176B를 같은 persistent raw owner에 추가했다.

정상 FF f6→358 한 번으로 [CI448/run41183473945](https://github.com/rahoney/heliopause/actions/runs/41183473945)을 시작했다. 실제08:10:03UTC/17:10:03KST 시작, 첫 due08:30:03UTC/17:30:03KST다. 기존 워커를 source-bound 단일PID1087983/start ticks10991317/session22590/config `/tmp/haa-audit-ci448-autopoll-config.json`로 재사용했으며 actual timed queries0으로 등록했다. CI447 worker는 자연 종료한 상태이고 중복 timer는 없다. 같은 활성 응답에서 actual20min 결과·terminal·최초 실패·정리를 소비한 뒤 현재 source의 CPU/세CUDA qualification을 마친다. 원본은 `ci448-registration.json`과 같은 monitor directory가 소유한다.

CI448 실제20min GET1:08:30:03.000144→08:30:03.812394UTC에서 macOS 포함8SUCCESS·LinuxIN_PROGRESS를 확인했다. 새CPP133을 hosted에서도 직접 확인했고 완료8로그는 한 번만 내려받아 raw owner에 보존했다(Docs 원본은 기존 다운로드 재사용). 실제8 checkout은 e77224024ca70ae7d0b147255c733fa7f1d708c8/treee1cac135로 frozen source 전체와 같다. Mac 경로 교정은 실제native에서 검증됐으며 Linux의전체성공/CPUfull/CUDA3은아직미완료다. 단일기존워커의다음due08:50:03.812394UTC/17:50:03KST를같은활성turn에서소비한다. `ci448-completed-small-logs.json`과 poll01 원본이 hash/범위를소유하며 이미완료로그를다시받지않는다.

CI448 actual GET2:08:50:03.812526→08:50:04.974976UTC도8SUCCESS·LinuxIN_PROGRESS다. 같은단일워커/고정source를유지하고새코드/CI를추가하지않았다. poll02 원본을같은raw owner에보존했고다음due09:10:04.974976UTC/18:10:05KST를같은활성응답에서소비한다. 전체결과·CPUfull/CUDA3은계속PENDING이다.

CI448 actual GET3:09:10:04.975123→09:10:05.753088UTC는8SUCCESS·LinuxIN_PROGRESS다. Same source/worker를유지하고poll03 원본을보존했다. 다음due09:30:05.753088UTC/18:30:06KST이며기존완료로그·동일local검사를반복하지않고전체종료/CPU·CUDA 결과를기다린다.

CI448 actual GET4:09:30:05.753215→09:30:07.055259UTC는8SUCCESS·LinuxIN_PROGRESS다. Frozen source/동일단일워커를유지하고poll04를보존했다. 다음due09:50:07.055259UTC/18:50:07KST이며전체결과가아직없다는사실을완료/실패로단정하지않는다.

CI448 actual GET5:09:50:07.055399→09:50:07.946694UTC도8SUCCESS·LinuxIN_PROGRESS다. 약100분경과는단독실패근거가아니며기존150분job/제품·개별test한도를바꾸지않는다. 같은source/워커를유지하고poll05를보존했다. 다음due10:10:07.946694UTC/19:10:08KST에서실제결과를소비한다.

CI448 actual GET6:10:10:07.946857→10:10:08.823527UTC는8SUCCESS·LinuxIN_PROGRESS다. 수신한step metadata는 runtime/lifecycle08:10:19→09:51:33 SUCCESS와 Node preparation09:51:33→09:57:19 SUCCESS를확인했다. 현재execution workflows/boundaries는09:57:19시작IN_PROGRESS, 이후kernel terminal-facts는pending이다. Main step 성공을원본parent/no-SKIP·CPUfull결과분석전에전체qualification으로확대하지않는다. 같은source/단일워커를유지하고poll06을보존했으며다음due10:30:08.823527UTC/19:30:09KST에서실제결과를소비한다.

## CI448 ordinary/CPU qualification — PASS

Actual GET7은10:30:08.823678→10:30:09.739171UTC에Required포함10SUCCESS를확인했다. Worker1087983/session22590자연exit0/nextnull이다. Server actual08:10:03→10:25:05UTC/8102s(135m02s)이며150분job·개별test제품한도는불변이다. 전체10logs1402952B를보존했고모든actualcheckoute77224024ca70ae7d0b147255c733fa7f1d708c8/treee1ca는frozen358 source전체와같다. Actual126uniqueparentPASS/중첩포함실제SKIP0/FAIL0, CPUfull268.73s, CPP133이다. Helper18은모두wait0이며최종4cleanup은original0/CONFIRMED_STOPPED/stop0/clienterror0/restore0, NOT_REQUIRED1·RESTORED3이다. 완료8로그는재사용하고마지막Linux/Required2개만추가로받았다. `ci448-ordinary-cpu-qualification.json`이Source/모든raw SHA·실제결과·검증범위를소유한다. 현재ordinary/CPU·nativeMac/새runtime의이경로는qualified이며CUDA3/Hostactivation/서명배포로전용하지않는다.

## cu126 CI449 등록 — IN_PROGRESS

Ordinary/CPU원본검증후현재remotehead358을확인했다. 연결도구의dispatch기능부재로기존repository Gitcredentialhelper를메모리에서만사용한공식ActionsAPI를재사용했고, secret/token을저장·출력하지않았다. Exact하나의cu126 boolean과다른두false/refmilestonebranch/return_run_details 요청을단한번POST해HTTP200/run41193740506을받았다. API2026-03-10·request/source·응답은 `cu126-dispatch.json`이보존한다. Runtime/source를바꾸거나새job/branch를만들지않았다.

[CI449](https://github.com/rahoney/heliopause/actions/runs/41193740506)은actual10:40:57UTC/19:40:57KST시작이며eventworkflow_dispatch/head358이다. 단일기존20min프로그램PID1127920/start11894010/session65415/config `/tmp/haa-audit-ci449-autopoll-config.json`의첫due는11:00:57UTC/20:00:57KST다. Actualtimedquery0으로등록했고이전CI448worker는종료됐다. 같은활성turn에서terminal/원본/CPU+cu126/정리를소비한뒤같은sourcecu130→cu132를순차검증한다. `ci449-registration.json`과raw owner가등록·범위를소유하며등록을실제조회/성공으로기록하지않는다.

CI449/cu126 actual GET1:11:00:57.000151→11:00:57.819192UTC는8SUCCESS·LinuxIN_PROGRESS다. 같은source/단일워커를유지하고poll01원본을보존했다. Docs actualcheckout358은같은treee1ca이며원본을한번받아보존했다. cu126을전체qualified로확대하지않고다음due11:20:57.819192UTC/20:20:58KST를같은활성turn에서소비한다.

CI449/cu126 actual GET2:11:20:57.819424→11:20:58.649591UTC는8SUCCESS·LinuxIN_PROGRESS다. Same source/단일워커를유지하고poll02원본을보존했다. 다음due11:40:58.649591UTC/20:40:59KST를같은활성응답에서소비한다. CPU+cu126원본/정리검증전qualified로확대하지않는다.

## cu126 qualification — PASS / cu130 CI450 등록

CI449 actualGET3은11:40:58.649743→11:40:59.527450UTC에10SUCCESS를확인했고worker1127920/session65415자연exit0/nextnull이다. Actualserver10:40:57→11:31:26UTC/3029s(50m29s)다. All10 directcheckout358/treee1ca, raw1915557B·실제6parentPASS/FAIL·SKIP0/CPU288.04s/cu1261270.55s/CPP133이다. Helper2wait0·최종original0/CONFIRMED_STOPPED/stop0/clienterror0/restore0/RESTORED를확인했다. Actualselectedplan/profilecu126·runtimePython3.14.10/pip26.2.2/cp314/manylinux_2_36_x86_64/sourcepytorch:cu126·torch2.14.0+cu126이일치한다. `cu126-qualification.json`이원본10SHA·시간·범위를소유한다. CPU의추가CUDAplanJSON은원래출력되지않는다. 처음reader가그marker까지요구한assertion은제품/CI실패가아니며, frozen source의CPU→선택CUDA 명령순서와실제두parentPASS·선택plan/runtime/torchidentity로대조했다. 원본편집·새CI·검사누락으로해결하지않았다.

이검증후remotehead358을재확인하고cu130만true인요청을단1POST해HTTP200/run41198471800을받았다. ExistingGit인증은메모리에서만사용했고저장·출력하지않았다. [CI450/cu130](https://github.com/rahoney/heliopause/actions/runs/41198471800)은actual11:49:53UTC/20:49:53KST 시작이다. 단일기존20minworkerPID1151204/start12305444/session74174/config `/tmp/haa-audit-ci450-autopoll-config.json`의첫due12:09:53UTC/21:09:53KST다. Actual0등록이며이전cu126worker는종료했다. 같은source/활성응답에서terminal·CPU+cu130원본/정리를소비한뒤cu132를진행한다. `cu130-dispatch.json`/`ci450-registration.json`이정확한요청/응답/source/범위를소유한다.

CI450/cu130 actualGET1:12:09:53.000148→12:09:53.835936UTC는7SUCCESS·MacFAIL·LinuxIN_PROGRESS다. Mac첫오류는production build의기존120초한도초과이며compiler/test진단이없고실제defaulttest는미도달이다. Actualcheckout358/Go1.27.3/macOS26.6.1/25G76과동일source의CI448/449성공을대조했다(CI449 productionbuild약46.895s). Capacity/coldcompile가설은미확정이며source회귀/환경원인으로단정하지않는다. 코드/한도를바꾸거나전체CI를다시시작하지않는다. 진행중cu130결과를보존한뒤실패Mac와그dependent Required만한번retry해동일입력에서검증한다. OfficialAPI의job/dependent retry와같은source보존범위를따르며[GitHub문서](https://docs.github.com/en/rest/actions/workflow-runs#re-run-a-job-from-a-workflow-run)를참조한다. 첫실패raw `ci450-macos-attempt1.log`/reader metadata와poll01을보존했고다음due12:29:53.835936UTC/21:29:54KST다.


## cu130 Linux PASS / Mac 단일 재시도

두 번째 실제20분 GET은12:29:54.627076UTC에attempt1 terminal failure를소비했다. Linux를포함한8SUCCESS를보존하며Mac/Required만실패다. Linux actual CPU215.77s/cu130855.49s·6parentPASS/noSKIP·currentruntime/torch2.14.0+cu130·CPP133·helper2wait0·최종original0/CONFIRMED_STOPPED/RESTORED를확인했다. 원본은`ci450-linux.log`/`ci450-linux-qualification-retry-pending.json`과attempt1 두poll/state다. Oldworker1151204는부재이며nextnull이다.

GitHub plugin으로실패Mac job124529340543만한번재시도해성공ack를받았다. 같은run의attempt2는12:33:44UTC시작이며8성공과GPU를재실행하지않는다. Source358와기존120초/전체한도는유지한다. FirstMac의하위원인은UNKNOWN이며재시도성공만으로과거원인을확정하지않는다. 단일20분worker1159466/start12564639/session56024의firstdue12:53:44UTC/21:53:44KST를등록했다. Actual0이며등록을완료조회로쓰지않는다. Cu130 전체QUALIFIED는새Mac/Required와재사용8개source/원본대조뒤판정한다. 그전cu132는시작하지않는다.


## cu130 qualification — PASS / cu132 CI451 등록

[CI450 attempt2](https://github.com/rahoney/heliopause/actions/runs/41198471800)은실제20분GET1/12:53:45.882597UTC에Required포함10SUCCESS를확인했다. Mac새원본은같은358/Go1.27.3에서productionbuild84.179s와defaulttestPASS다. 원래성공8개의source/start/end가모두같고attempt2시작이전종료임을대조해실제재실행과APIalias를구분했다. CPU215.77s/cu130855.49s·selectedplan/runtime/torch2.14.0+cu130·noSKIP·CPP133/helper2wait0/최종confirmedRESTORED까지qualified다. 전체원본1582205B와처음실패Mac별도원본/단일retry는`cu130-qualification.json`/`ci450-macos-rerun.json`이소유한다. 첫120s timeout하위원인은UNKNOWN으로유지한다. 재시도worker1159466은자연exit0/nextnull이며같은GPU/원본8logs를다시실행·취득하지않았다.

그뒤remotehead358·cu130qualified·기존worker종료를확인하고cu132만true인단1POST를실행해HTTP200/CI451 run41203161813을받았다. Actualstart12:54:25UTC, singleworker1163134/start12686726/session45372/config `/tmp/haa-audit-ci451-autopoll-config.json`의firstdue13:14:25UTC/22:14:25KST다. Actual0등록이며완료가아니다. Source/한도는불변이고같은활성응답에서terminal·actualCPU/cu132원본/정리/최종감사를이어간다. `cu132-dispatch.json`/`ci451-registration.json`이요청·응답·scope를소유한다.


CI451 actual20minGET1/13:14:25.842697UTC:8SUCCESS(Mac/Quick/Security/Vulnerability/Minimum/Corpus/CPP/Docs)·LinuxIN_PROGRESS/noFAIL을확인했다. 완료8logs446689B의actualcheckout이모두358인것을확인해한번만보존했다. Linux/Required로그는미취득이며이부분을전체qualified로쓰지않는다. 다음due13:34:25.842697UTC/22:34:25.842697KST다. 같은활성turn/worker45372를유지하며코드/CPP/CPU를반복하지않는다.


CI451 actual20minGET2/13:34:26.626694UTC도8SUCCESS·LinuxIN_PROGRESS/noFAIL이다. 완료8logs를재취득하거나같은GPU/로컬검사를반복하지않았다. 같은worker의다음due13:54:26.626694UTC/22:54:26.626694KST를유지하고현재실행의실제terminal을기다린다.


## cu132 최종 qualification — PASS

실제20분 GET3/13:54:27.455226UTC에 [CI451](https://github.com/rahoney/heliopause/actions/runs/41203161813)의 Required 포함10SUCCESS를 소비했다. 기존 완료8개446689B는 재취득하지 않고 마지막 Linux/Required2개만 추가했다. 전체1790594B·all actualcheckout358·CPU290.53s/cu1321064.79s·현재 plan/runtime/torch2.14.0+cu132·actual6parentPASS/noSKIP·CPP133/helper2wait0·최종original0/CONFIRMED_STOPPED/stop0/clienterror0/restore0/RESTORED를 대조했다. Worker1163134/session45372는 자연exit0/nextnull이며 실제47분 안팎의 실행을 끝까지 확인했다. Source/한도/필수 검사는 불변이다. `cu132-qualification.json`과 최종 poll03/state가 원본을 소유한다. 이로써 이번 감사의 필요한 네 hosted profile과 최종 정리 대조를 완료했다.
