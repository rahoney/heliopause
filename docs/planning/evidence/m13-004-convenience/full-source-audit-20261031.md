# 이름 결정 전 전체 코드·보안 감사

Status: IN_PROGRESS — local configuration/security PASS; CI447 ordinary/CPU FAIL, first macOS/Linux failures under investigation, required CUDA3 NOT_REACHED. Owner: M13-004. 사용자 2026-10-31 요청에 따라 전체 저장소 감사 범위를 명시적으로 확대한다. 이름 결정·등록·게시·main merge는 포함하지 않는다.

## 기준선과 완료 기준

- Source: `21a0a25214cce425d1204c63a47935170f68ca71`, branch `milestone/m13-release-build-packages`; 원격보다 local 두 commit 앞. 감사 시작 시 tracked 수정은 Queue뿐이며 기존 untracked 자료는 보존한다.
- 기존 구현: IMPLEMENTED=YES / WIRED=YES는 원래 bounded P2..P7 기능 범위에만 해당한다. 새 감사나 새 signed release의 완료 판정이 아니다.
- 감사 시작 baseline: IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO.
- 현재 교정/감사 연결: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO. Local configuration7·vulnerability/history·표적은 PASS이며 새 hosted ordinary/CPU·세CUDA가 남는다.
- 시작 MISSING: 전체 inventory/자동 검사 범위 매핑, 미사용·중복·복잡도 검토, 신뢰 경계별 사람 검토, 현재 취약점 DB·secret history 검사, 확정 문제의 최소 교정과 회귀, 최종 변경/검증/미검증 경계 대조. 현재 MISSING은 새 hosted qualification과 그 결과·정리·최종 상태 대조다.
- 완료는 모든 maintained owner의 검토/도구 coverage와 finding disposition, 필요한 교정/회귀를 연결했을 때만 판정한다. 알려진 취약점이 발견되지 않거나 테스트가 통과해도 모든 버그 부재를 증명하지 않는다.

## 범위와 방법

Tracked inventory: Go production681파일119060줄 / Go test761파일129623줄 / C++2파일17093줄 / shell18파일1760줄 / Python6파일293줄 / YAML6파일1482줄. Docs 안의 재현 script도 포함한다. 외부 gVisor 전체 upstream source는 이 저장소의 자체 코드가 아니며, 유지하는 patch와 호출/검증 경계는 포함한다.

| 검토 단위 | 자동 검사 / 기존 근거 | 사람 검토와 남은 일 | 상태 |
| --- | --- | --- | --- |
| 전체 Go·platform/tag·의존성 | AST1442/오류0, Linux default/all-tags·Darwin deadcode, canonical vulnerability/history PASS | package/entry reachability·중복/복잡도 검토; 아래 disposition | REVIEWED; final checks pending |
| acquisition·adapter·archive | 기존 bounded functional/CI446 범위만 재사용 | bounded reader·정규 경로/링크·redirect/인증·complete graph/원본 binding 검토; 새 재현 결함 없음 | REVIEWED |
| verification·evidence·policy | 기존 계약/회귀 | fresh whole/required Evidence·fail-closed·provenance와 first-trust 분리 검토; 새 재현 결함 없음 | REVIEWED |
| promotion·ownership·rollback | 보완 G122/G304·기존 회귀 | private staging·held FD/inode·no-replace/journal·root/user 경계 검토; F03 수정 | REVIEWED; final checks pending |
| sandbox·observer·helper | 새 CPP133 PASS·영향 Race PASS | framing·argv/env·kernel role/FD·완료 authority·cleanup 검토; F01/F02/F06 수정 | REVIEWED; hosted pending |
| bootstrap·release·CLI | 이름독립 unsigned 조립은 별도 제한 범위 | 기존 privileged/provenance caller·초기 획득·출력 경계 검토; UV inspect 누락은21a 수정, signed deployment 별도 | REVIEWED; hosted pending |
| checker·script·workflow·runtime pins | 추적된 shell/Python24개 syntax PASS·workflow PASS | CI 권한/고정 argv·공급망 pin/advisory 검토; F04/F05/F07/F08 수정 | REVIEWED; final checks pending |

Canonical high-signal gosec는 G104/G115/G122/G204/G301/G302/G304/G703를 기본 검사하지 않는다. 이번 전체 감사에서는 그 차이를 명시하고 보완 분석/사람 검토를 한다. 임의의 complexity 기준으로 함수/파일을 분할하지 않는다. 변경·새 위험이 없는 expensive native/CPP/CUDA/CI를 반복하지 않는다.

## 실행과 발견

첫 canonical vulnerability는 현재 작업 디렉터리의 보존된 미추적 Go source capture를 `./...`가 포함해 package-load 오류로 실패했다(186.879s/exit1). 이는 취약점 발견/검사 성공이 아니다. 같은 잘못된 cwd의 보완 gosec는 정확한 worker/child 신원 확인 후 중지(exit143)했고 deadcode는 미도달이다. 원본은 `/tmp/haa-full-audit-20261031/`에 보존한다. 과거 raw를 삭제·제외 규칙으로 숨기지 않고 clean detached worktree `/tmp/haa-full-audit-20261031/source`의 동일21a HEAD/3255 tracked files로 분석한다.

### F01 — 로컬 요청 framing의 읽는 동안 크기 제한 누락

`internal/networkpolicy/server.go:handle`과 `internal/hosttool/observation_resource_linux.go:serveObservationResources`는 authenticated peer 뒤에 `bufio.NewReaderSize(..., limit+1).ReadBytes`를 호출한다. Reader buffer 크기는 ReadBytes의 누적 할당 상한이 아니므로 newline 없는 입력은 사후 len 검사 전에 계속 쌓인다. Kernel peer UID·보호된 executable 인증과5초 deadline은 존재하므로 임의 Artifact의 원격 접근/권한 상승을 입증한 것은 아니다. 문제는 기존 bounded protocol 계약 위반/메모리 사용 경계다.

Edit map: 두 기존 reader에서 `ReadSlice`를 사용해 buffer-full을 기존 err 경로로 거부한다. 새 protocol/함수/authority를 만들지 않는다. 실제 Unix stream에서 newline 없는 limit+1 입력의 조기 거부와 정상 요청을 검증한다. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO (local PASS; combined hosted 미완료).

### F02 — resolver의 링크 로컬·broadcast 주소 거부 누락 — 반례 확정

`networkpolicy.NewResolverPolicy`와 `sandbox.validateResolverEndpoints`의 기존 public IPv4 조건은 private/loopback/multicast/unspecified만 검사한다. `169.254.169.254`와 limited broadcast에 대한 반례를 먼저 추가한다. Go/Cargo/PyPI/NPM의 기존 최종 공용 validator 호출을 재사용하고 predicate owner 두 곳만 고친다. DNS/Host 장악이나 실제 metadata 접근까지 입증한 것은 아니다. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO (local PASS; combined hosted 미완료).

후속 [IANA 특별 주소 대조](https://www.iana.org/assignments/iana-ipv4-special-registry)는 GlobalUnicast가 공개 routability와 같지 않음을 확인했다. 새 정확한 입력100.100.100.200(shared space)/0.1.2.3/198.18.0.1/240.0.0.1의 기존 두 validator 반례를 추가했다. Edit map: 추가 IANA 검사는 기존 typed policy constructor 한 곳에 두어 client encode와 privileged decode가 함께 소비한다. Sandbox의 기본 preflight는 유지하고 기존 Bootstrap adapter/Create 경로가 강한 typed 검사를 반드시 거친다. Sandbox→networkpolicy 새 의존성이나 architecture allowlist 확대를 만들지 않는다. Public anycast192.0.0.9/.10은 원래 공개 주소 의미를 유지한다. 새 network service/config/package/helper는 만들지 않는다.

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

### F08 — 중복된 Docker 버전 비교가 suffix/잘못된 숫자를 허용 — 반례 확정 / local PASS

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
