# Step 9 — Coding / Security Rules

이 문서는 Step 1~8에서 확정한 Threat Model, Architecture, Domain invariant와 Directory Structure를 Go 구현 규칙으로 구체화한다. 기존 보안 의미와 Domain 상태를 변경하거나 다시 정의하지 않으며, 코드가 기존 결정을 훼손하지 않도록 작성·검토할 기준만 정한다.

## 적용 범위

- Heliopause Go product code, test helper와 repository-owned script에 적용한다.
- 외부 scanner, verifier와 sandbox runtime 자체의 소스에는 적용하지 않지만 해당 도구를 호출하는 Adapter/Provider/Backend 코드에는 적용한다.
- generated code는 생성 원본과 generator를 우선 검토하되, 비밀값·위험한 command·금지된 dependency를 포함해서는 안 된다.
- 세부 formatter·linter·scanner와 enforcement 명령은 Step 10에서 정한다.
- CI 적용과 merge 차단 기준은 Step 11에서 정한다.

## 1. Go 버전

- 구현을 시작하는 시점의 안정 Go 버전을 `go.mod`에 명시한다.
- 최소 지원 Go 버전은 CI에서 별도 검증한다.
- 새 Go 버전이 발표됐다는 이유만으로 자동 추종하지 않는다.
- Go 버전 변경은 release note, compiler/runtime 변화, standard library 보안 수정, dependency와 toolchain 호환성을 검토한 명시적 변경으로 수행한다.
- 개발 환경과 CI가 서로 다른 toolchain을 암묵적으로 선택하지 않도록 version source를 한 곳으로 유지한다. 정확한 설치·pinning 방식은 Step 10~11에서 정한다.

## 2. Dependency와 공급망

### 기본 원칙

- Go standard library로 명확하고 안전하게 구현할 수 있으면 외부 dependency를 추가하지 않는다.
- 단순 문법 축약이나 작은 convenience만을 위한 dependency는 추가하지 않는다.
- 외부 dependency는 다음을 검토한 뒤 추가한다.

```text
실제 필요성
유지보수 상태와 release cadence
license와 배포 제약
공개 취약점과 security response
maintainer·ownership 변화
transitive dependency 규모
module path·release tag·checksum 신뢰성
대체 가능한 standard library 또는 더 작은 구현
```

- 직접·전이 dependency는 `go.mod`와 `go.sum`으로 고정한다.
- `go.sum`을 삭제하거나 검증을 우회해 dependency 문제를 해결하지 않는다.
- dependency 추가·major upgrade·maintainer 변경은 일반 코드 변경보다 공급망 검토를 강화한다.
- 사용하지 않는 dependency는 제거한다.
- `replace` directive, fork, pseudo-version, prerelease와 vendoring은 명시적 이유와 검토 없이 사용하지 않는다.
- external CLI/tool/container image는 Go module dependency가 아니더라도 동일한 trusted tooling 관점에서 version·identity·integrity를 관리한다. 정확한 pinning과 검증 도구는 Step 10 이후 결정한다.

### 금지

- 코드 또는 script에서 dependency를 실행 시 자동 최신 버전으로 교체하지 않는다.
- checksum database, TLS 또는 module verification을 편의상 우회하지 않는다.
- Artifact가 제공한 Go module, executable 또는 script를 Heliopause product dependency로 동적 import·load하지 않는다.
- MVP에 동적 Go plugin loading을 추가하지 않는다.

### M0-003 Cobra dependency review

2026-08-12 KST에 Heliopause의 확정된 CLI framework를 처음 사용하는 시점의 공식 release와 module graph를 검토해 `github.com/spf13/cobra v1.10.2`를 product dependency로 고정한다.

- `v1.10.2`는 확인 시점의 최신 stable release이며 GitHub에서 verified signature가 확인된 commit `88b30ab89da2d0d0abb153818746c5a2d30eccec`다.
- module의 Go floor는 `1.15`이므로 Heliopause minimum Go `1.25.12`와 default Go `1.26.5`에서 호환된다.
- source license는 Apache-2.0이며 실제 build-selected indirect dependency인 `pflag`는 BSD-3-Clause, `mousetrap`은 Apache-2.0이다. 배포 시 각 license/notice 의무를 dependency license 검토에 포함한다.
- Heliopause source가 직접 사용하는 것은 Cobra뿐이다. build-selected graph는 Cobra, `pflag`, `mousetrap`이고 upstream 전체 module graph에는 `go-md2man`, `blackfriday`, YAML과 check module도 포함된다. exact graph와 checksum은 `go.mod`, `go.sum`, `go list -m all`, `go mod graph`로 검증한다.
- 확인 시점의 GitHub global advisory API에서 Cobra package에 일치하는 공개 advisory는 없었다. 이는 안전 보증이 아니며 govulncheck가 활성화되기 전까지 dependency 변경 시 advisory source를 다시 확인한다.
- release와 dependency maintenance, security reporting 문서가 유지되고 있고 CLI framework로 널리 사용된다. 그래도 external parsing/presentation code이므로 Domain/Application에 전파하지 않고 `internal/cli`에만 격리한다.
- upgrade는 release note, Go floor, transitive graph, license, 공개 취약점과 CLI/error/help 동작 변화를 검토한 명시적 변경으로만 수행한다.

검토 source:

- [Cobra v1.10.2 release](https://github.com/spf13/cobra/releases/tag/v1.10.2)
- [Cobra v1.10.2 module requirements](https://github.com/spf13/cobra/blob/v1.10.2/go.mod)
- [Cobra security policy](https://github.com/spf13/cobra/blob/v1.10.2/SECURITY.md)
- [Cobra Apache-2.0 license](https://github.com/spf13/cobra/blob/v1.10.2/LICENSE.txt)
- [GitHub global advisory query for Cobra](https://api.github.com/advisories?ecosystem=go&affects=github.com%2Fspf13%2Fcobra&per_page=100)

## 3. Package와 API 작성

- Step 8의 dependency direction과 package 책임을 따른다.
- `core/domain`은 다른 Heliopause product package에 의존하지 않으며, 외부 library 의존도 원칙적으로 두지 않는다. 불가피한 경우 생태계·도구·OS 비종속성과 Domain 순수성을 해치지 않는지 별도 검토한다.
- package 이름은 짧고 책임을 나타내는 소문자 단수형을 사용하며 `util`, `common`, `misc`, `manager`처럼 소유권이 불명확한 package를 만들지 않는다.
- exported identifier는 실제 package 경계에서 필요한 최소 범위로 제한한다.
- export한 type과 function은 계약·보안 전제·ownership이 불분명하지 않도록 Go doc을 작성한다.
- interface는 실제 Port 또는 소비자 요구에서 도출한다. 미래 mock이나 교체 가능성만으로 interface를 만들지 않는다.
- interface를 구현 package에 중복 선언하거나 `any` 기반 범용 Port로 합치지 않는다.
- constructor는 유효한 객체만 반환하고 invariant 검증 실패를 명시적 error로 반환한다.
- zero value가 유효한지 문서화한다. 유효하지 않은 type은 constructor를 요구하고 zero value를 조용히 정상 상태로 해석하지 않는다.
- package-level mutable global state와 암묵적 singleton을 사용하지 않는다.
- configuration, clock, random source, filesystem/network/process 경계는 필요한 곳에서 명시적으로 주입한다. 단순 pure logic까지 과도하게 interface화하지 않는다.
- production code가 `internal/testutil` 또는 test fixture를 import하지 않는다.

## 4. Domain type와 상태 변경

- 사용자 입력 문자열, 외부 provider 응답과 Domain type을 같은 값으로 취급하지 않는다. Adapter/경계에서 parse·validate·normalize한 뒤 Domain type을 생성한다.
- Artifact Reference, Resolved Identity, Acquired Artifact, Inspection Run, Policy Decision과 Operation Result의 기존 구분을 type/API에서 보존한다.
- Capability, Execution Status, Run Outcome, Policy Decision과 Operation Status를 하나의 범용 status 문자열로 합치지 않는다.
- enum 성격의 값은 허용 집합을 검증하고 알 수 없는 값을 성공·빈 결과로 변환하지 않는다.
- Finalized Inspection Run, Verified Set/Manifest와 Evidence record는 외부에서 임의로 mutate할 수 있는 setter API를 제공하지 않는다.
- 상태 변경은 유효한 transition을 확인하는 method 또는 Application workflow를 통해 수행한다.
- security-sensitive identifier, digest와 reference는 서로 바꿔 쓸 수 없는 별도 type 또는 명확한 wrapper를 사용한다.
- 원본 provider payload가 필요하면 bounded Raw Evidence로 보존하고 Domain Result나 Policy input에 vendor-specific map을 직접 전달하지 않는다.

## 5. `context.Context`, cancellation과 timeout

- I/O, network, subprocess, storage와 장시간 작업을 수행하는 public boundary method는 `context.Context`를 첫 번째 인자로 받는다.
- `context.Context`를 struct field에 저장하거나 nil로 전달하지 않는다.
- library·internal package가 호출자의 context를 `context.Background()`로 교체해 cancellation을 끊지 않는다.
- process entrypoint 또는 명확한 detached cleanup처럼 lifecycle owner가 분명한 경계에서만 root context를 만든다.
- cancellation과 deadline error를 감추지 않고 원인을 보존한다.
- 모든 외부 요청과 subprocess 실행에는 상위 deadline 또는 명시적 timeout이 있어야 한다.
- retry는 기존 deadline을 넘겨 새 무제한 context를 만들지 않는다.
- cleanup은 취소된 작업에서도 실행하되, 필요한 경우 별도의 짧고 bounded cleanup context를 lifecycle owner가 만든다.

## 6. 오류 처리

### 오류 축 분리

```text
Operational Error
≠ Domain Result
≠ Policy Decision
≠ Operation Result
```

- network failure, timeout, malformed response, storage failure와 subprocess failure는 Operational Error다.
- digest mismatch처럼 검증이 정상 실행되어 얻은 결과는 Domain Result다.
- 최종 `ALLOW / MANUAL_REVIEW / BLOCK`은 Policy Decision이다.
- 실제 install/Promotion 성공·실패·보류는 Operation Result다.
- 한 축의 값을 다른 축으로 임의 변환하지 않는다.

### Go error 규칙

- 정상적으로 예상되는 실패에 `panic`을 사용하지 않는다.
- 외부 입력, Artifact, network, file, provider output과 runtime failure로 panic이 발생하지 않게 한다.
- error를 추가 context와 함께 반환할 때 `%w`로 원인을 보존한다.
- 호출자가 분기해야 하는 안정적인 error만 typed error 또는 sentinel과 `errors.Is`/`errors.As`로 노출한다.
- 문자열 비교로 error 종류를 판별하지 않는다.
- 같은 error를 log하고 반환하여 여러 계층에서 중복 기록하지 않는다. 기록 책임은 operation context를 가진 경계에서 갖는다.
- error message에 secret, credential, raw environment, 인증 URL 또는 민감한 Artifact output을 포함하지 않는다.
- error를 무시하려면 안전상 영향이 없다는 근거가 코드상 명확해야 하며, 빈 assignment로 조용히 버리지 않는다.
- cleanup error는 주 error를 잃지 않게 결합·기록하되 cleanup 실패가 Evidence/Staging 무결성에 영향을 주면 성공으로 반환하지 않는다.

### `panic` 허용 범위

- package API와 workflow에서 복구 가능한 상황에 panic을 사용하지 않는다.
- compile-time/static constant로만 구성된 programmer invariant의 명백한 위반이나 process 시작 전 불변 bootstrap 실패처럼 정상 실행이 불가능한 경우에도 명시적 error 반환을 우선한다.
- `Must...` helper는 test 또는 정적으로 통제된 bootstrap 값에 한정하고 외부 입력을 받지 않는다.
- recover로 임의의 panic을 성공 상태로 바꾸지 않는다. process 경계의 최후 방어 recover가 필요하면 실패로 종료하고 민감정보가 없는 최소 진단만 남긴다.

### CLI exit code

- CLI exit code 변환은 CLI/Application 경계에서만 수행한다.
- Adapter/Provider/Domain/Policy package가 process exit code를 직접 반환하거나 `os.Exit`을 호출하지 않는다.
- `cmd/helox/main.go` 이외의 product package에서 `os.Exit`을 호출하지 않는다.
- 외부 tool exit code는 해당 Provider/Backend가 정규화된 실행 결과 또는 Operational Error로 변환한다.
- Heliopause exit code taxonomy의 정확한 값은 후속 CLI/quality 설계에서 확정한다.

## 7. Logging, Evidence와 redaction

- product diagnostic log는 Go standard library `log/slog` 기반 구조화 로그를 사용한다.
- log key는 안정적인 이름을 사용하고 사람이 읽는 message에 구조화 값을 반복 삽입하지 않는다.
- 가능한 경우 Operation ID, Inspection Run ID, check/provider 이름과 Execution Status를 상관관계 attribute로 기록한다.
- log는 Evidence의 canonical source가 아니다. 감사·판정 근거가 필요한 자료는 trusted Evidence Port를 통해 기록한다.
- Artifact stdout/stderr와 external tool raw output을 일반 log에 그대로 쓰지 않는다. 필요한 경우 size-limited Raw Evidence로 별도 보존하고 log에는 reference와 요약만 남긴다.

### 기본 redaction 대상

- token, password, API key, SSH key, cookie, authorization header와 credential
- secret 및 환경변수 값
- URL userinfo와 credential-bearing query/header
- 사용자 이름, home directory 등 사용자 식별 가능 경로 정보
- Artifact 또는 external tool 출력에서 탐지된 secret 후보
- sanitized 여부가 검증되지 않은 project content와 request/response body

Host path 전체를 일괄 제거할 필요는 없다. 작업 의미에 필요한 비식별 경로, basename, logical content handle 또는 trusted root 기준 상대 경로를 우선 기록하고, home·사용자명·project명 등 개인 식별 가능 부분은 redaction한다.

### 금지

- secret 값에 `String()`/`GoString()`을 구현하여 일반 formatting으로 노출하지 않는다.
- struct 전체를 `%+v`, JSON dump 또는 slog `Any`로 기록하지 않는다.
- request header, environment 전체, command environment 또는 config 전체를 debug log로 기록하지 않는다.
- redaction 실패 시 원문을 fallback으로 기록하지 않는다.

## 8. Secret과 source authentication

- 실제 Secret을 CLI positional argument나 일반 flag 값으로 받지 않는다. process list, shell history와 telemetry에 노출될 수 있기 때문이다.
- source authentication은 trusted Artifact Adapter 경계에서만 사용한다.
- credential은 최소 권한·최소 대상·최소 수명 원칙을 적용한다.
- Adapter에는 필요한 source/host의 credential만 전달하고 전체 Host credential store를 제공하지 않는다.
- 실제 credential, token, `.env`, SSH agent/socket과 cloud/browser credential을 Sandbox에 전달하지 않는다.
- credential을 environment로 전달해야 하는 외부 tool은 명시적 allowlist에 포함된 최소 변수만 받는다.
- redirect, retry 또는 mirror 전환 시 credential 대상 host·scheme·scope를 다시 검증한다.
- 실제 Secret/Credential 값 자체를 Evidence, Result, cache key, digest input 또는 error에 포함하지 않는다.
- synthetic credential/honeytoken의 식별자, 접근 여부, 접근 행위와 관련 metadata는 실제 유효 Secret을 포함하지 않는 범위에서 Evidence로 기록할 수 있다.
- secret 메모리 zeroization을 절대적으로 보장한다고 주장하지 않는다. 대신 복사·수명·노출 범위를 줄이고 장기 보관하지 않는다.
- credential source, broker와 저장 방식은 구현 직전 별도 결정으로 유보한다.

## 9. 외부 명령과 process 실행

외부 명령 실행은 Artifact가 Host 권한으로 명령을 주입하거나 lifecycle을 벗어나지 못하도록 강한 경계로 취급한다.

### 필수 규칙

- shell command 문자열을 조합해 실행하지 않는다.
- `exec.CommandContext`에 executable과 argument를 분리해 전달한다.
- Artifact, metadata 또는 사용자 입력 문자열을 shell syntax로 재해석하지 않는다.
- `sh -c`, `bash -c`, PowerShell command string과 유사한 shell 경유 실행은 금지한다. 검사 대상 lifecycle 자체를 Sandbox 안에서 실행해야 하는 경우에도 Sandbox Backend의 명시적 계약과 격리 정책을 통해 수행하며 Host command runner에 전달하지 않는다.
- executable은 trusted configuration/Adapter가 선택하며 Artifact가 임의 executable path를 지정하지 못한다.
- `PATH` 기반 검색은 가능한 한 피한다. 필요한 경우 허용된 executable name과 신뢰할 수 있는 search path를 제한하고, 실행 직전에 resolve된 executable path와 identity를 확인한다. 가능한 경우 검증된 absolute path를 사용한다.
- working directory는 통제된 directory로 명시하며 process 기본 cwd를 암묵적으로 상속하지 않는다.
- environment는 최소 baseline에서 명시적 allowlist로 구성한다. Host environment 전체를 `os.Environ()`으로 그대로 전달하지 않는다.
- stdin 제공 여부와 내용은 명시적으로 결정한다. interactive prompt를 예상하지 않는 command에는 닫힌 stdin을 사용한다.
- timeout, cancellation, stdout/stderr size limit과 process tree 종료를 적용한다.
- cancellation 시 child·grandchild process가 남지 않도록 OS/backend별 process group 또는 동등한 lifecycle control을 구현한다.
- stdout/stderr capture는 bounded하며 일반 log 대신 정규화 결과 또는 Raw Evidence 경계로 전달한다.
- exit code, signal, timeout과 launch failure를 서로 구분하여 Execution Status/Operational Error로 정규화한다.
- 비정상 종료된 Sandbox Session과 untrusted work directory는 재사용하지 않는다.

### 구현 경계

- `os/exec` 사용을 product 전체에 흩뜨리지 않고 external tool 또는 Sandbox 책임 package 안에 둔다.
- 공통 command runner가 필요하면 shell abstraction이 아니라 executable·args·env allowlist·cwd·limits가 명시된 좁은 API로 만든다.
- Verification/Inspection Provider는 external tool 결과를 정규화하지만 최종 Policy Decision을 만들지 않는다.

## 10. File, path, archive와 storage

### Root와 handle

- 임의 Host path 문자열 대신 통제된 storage/content handle과 trusted root 기준 상대 경로를 우선 사용한다.
- Intake/Quarantine, Sandbox work area, Evidence Store와 Staging root를 서로 다른 handle·권한·lifecycle로 유지한다.
- path validation은 lexical `Clean`/prefix 비교만으로 끝내지 않고 실제 filesystem resolution과 link 경계를 고려한다.
- untrusted path가 trusted root 밖으로 escape하지 못하게 한다.
- absolute path, volume/device path, `..`, alternate separator·encoding과 platform-specific path를 명시적으로 검증한다.
- symlink와 hardlink를 따라 trusted root 밖의 파일을 읽거나 쓰지 않는다.
- check-then-use 사이에 대상이 바뀔 수 있는 TOCTOU를 피하고 가능한 경우 열린 directory/file handle 기준 operation과 atomic primitive를 사용한다.

### 생성과 쓰기

- temporary directory/file은 안전한 OS API로 생성하고 예측 가능한 이름을 보안 경계로 사용하지 않는다.
- private runtime directory와 credential-related file은 최소 권한으로 생성한다.
- 기존 파일을 암묵적으로 truncate/overwrite하지 않는다. 새 파일 생성, replace와 overwrite를 구분한다.
- Promotion 대상 overwrite는 사용자 의도, target identity와 atomicity를 확인한 전용 Promotion 경계에서만 수행한다.
- partial write를 성공으로 노출하지 않고 임시 파일·fsync·atomic rename 필요성을 storage 특성에 맞게 검토한다.
- cleanup 실패가 untrusted content 또는 credential을 남길 수 있으면 기록하고 정상 완료로 숨기지 않는다.

### Archive와 acquired content

- archive entry 수, 전체·개별 size, nesting depth, compression ratio와 extraction 시간을 제한한다.
- path traversal, absolute path, symlink/hardlink escape, special file, device node와 unsafe permission을 검증한다.
- extension만으로 content format을 신뢰하지 않는다.
- extraction 중 limit 초과 또는 validation 실패 시 partial output을 검증된 content로 사용하지 않는다.
- acquired bytes에서 observed digest를 계산하며 declared digest를 대체 값으로 사용하지 않는다.
- digest 계산 이후 content 변경 가능성을 차단하거나 Promotion 경계에서 다시 검증한다.

### Runtime data

- production runtime data를 source tree에 쓰지 않는다.
- test는 `t.TempDir()` 또는 격리된 test root를 사용한다.
- Evidence와 Staging을 같은 directory 또는 동일 권한의 범용 blob store처럼 취급하지 않는다.
- untrusted Artifact/Sandbox process가 Evidence를 수정·삭제할 수 있는 write capability를 받지 않는다.

## 11. Network와 download

- 모든 외부 request에 context deadline/timeout과 response/download size limit을 적용한다.
- `http.DefaultClient`처럼 timeout 없는 process-global client에 의존하지 않는다.
- TLS certificate/hostname 검증을 끄거나 insecure transport를 허용하지 않는다.
- HTTP 사용이 불가피한 명시적 source가 별도 결정되지 않는 한 HTTPS를 요구한다.
- redirect 횟수, scheme과 destination을 제한·검증한다.
- redirect 시 authorization, cookie와 credential을 새 host에 전달하지 않는다.
- URL userinfo를 인증 수단으로 허용하지 않는다.
- response status, content length와 실제 bytes 수를 검증하고 limit 초과 시 중단한다.
- response body와 transport resource는 모든 경로에서 닫는다.
- retry는 idempotency, total deadline, attempt limit과 backoff를 명시하며 무한 retry하지 않는다.
- source metadata가 가리키는 URL도 신뢰 입력으로만 취급하고 destination·scheme·credential scope를 Adapter에서 검증한다.
- Sandbox network는 Artifact Adapter의 trusted source network와 분리한다. Sandbox에 실제 internal network나 Adapter credential을 제공하지 않는다.
- proxy 사용과 trusted CA 확장은 명시적 configuration·audit 경계를 요구하며 process environment에서 암묵적으로 상속할지는 후속 configuration 결정에서 확정한다.

## 12. Cryptography, digest와 random

- custom cryptographic algorithm 또는 자체 TLS/signature verification을 구현하지 않는다.
- Go standard library 또는 검토된 보안 library의 검증된 primitive를 사용한다.
- 약한 digest를 안전성 근거로 새로 도입하지 않는다. 지원 algorithm과 compatibility는 별도 Domain/verification 결정에 따른다.
- digest는 algorithm과 canonical encoding을 함께 표현하고 서로 다른 algorithm 값을 문자열만으로 비교하지 않는다.
- security-sensitive 비교는 library가 제공하는 안전한 verification API를 사용한다.
- identifier, nonce 또는 security token에 `math/rand`와 timestamp 기반 예측 가능한 값을 사용하지 않는다.
- 보안 randomness가 필요하면 `crypto/rand`를 사용하고 실패를 무시하지 않는다.
- checksum, signature, provenance가 성공해도 그것만으로 Policy `ALLOW`를 생성하지 않는다.

## 13. Input, serialization과 output

- CLI, config, registry metadata, external tool output와 stored record는 모두 untrusted input으로 취급한다.
- parse 전에 입력 byte/record/depth limit을 적용한다.
- parse 성공과 semantic validation 성공을 구분한다.
- required field, enum, identity/digest format과 cross-field invariant를 검증한다.
- 알 수 없는 값 또는 newer schema를 빈 값·성공·지원됨으로 해석하지 않는다.
- external provider schema는 compatibility가 필요한 범위에서 unknown field를 허용할 수 있지만, 사용되는 보안 필드는 반드시 검증한다.
- Heliopause가 소유한 machine-readable contract는 schema/version을 명시하고 모호한 polymorphic map보다 typed representation을 사용한다.
- duplicate key, integer overflow, excessive nesting, invalid Unicode와 ambiguous normalization을 고려한다.
- output serializer는 secret/redaction policy를 우회하지 않는다.
- terminal output과 machine-readable output은 동일 Domain Result/Policy Decision에서 생성하며 서로 다른 보안 의미를 만들지 않는다.
- raw Artifact/provider output을 사용자 terminal에 자동 출력하지 않는다. 명시적 review 경로에서도 size와 redaction을 적용한다.

## 14. Concurrency, ownership과 cleanup

- goroutine, channel, file, response body, process와 Sandbox Session의 owner와 종료 조건을 명확히 한다.
- unbounded goroutine, queue, channel, recursion과 parallel download/inspection을 허용하지 않는다.
- concurrency limit은 workload·resource policy에서 전달받으며 package 내부 상수로 무제한 확장하지 않는다.
- goroutine을 시작한 code가 완료·error·cancellation을 관찰하고 누수를 방지한다.
- shared mutable state는 최소화하고 필요한 경우 mutex/channel ownership과 invariant를 문서화한다.
- map의 동시 read/write, close-after-use와 double-close를 안전하게 처리한다.
- cancellation 시 partial result의 completeness와 Execution Status를 정확히 기록한다.
- retry attempt는 이전 Sandbox Session과 untrusted temporary state를 재사용하지 않는다.
- cleanup은 idempotent하도록 설계하고 여러 실패 경로에서 호출돼도 신뢰 경계를 약화하지 않는다.
- resource limit 초과를 단순 성능 문제로 숨기지 않고 Evidence/Execution Status에 연결할 수 있게 한다.

## 15. Testability와 결정론

- Domain/Policy pure logic은 network, filesystem, wall clock과 process-global state 없이 test할 수 있게 한다.
- time-dependent 판단은 필요한 경계에서 clock을 주입하되 모든 code에 불필요한 clock interface를 만들지 않는다.
- random identifier 생성은 생성 경계에 격리하고 test에서는 deterministic source를 주입할 수 있게 한다.
- network, process, filesystem과 storage failure를 test에서 재현할 수 있는 Port/Adapter 경계를 유지한다.
- test가 실제 home directory, credential, public registry 또는 internal network에 암묵적으로 접근하지 않는다.
- security test fixture는 synthetic credential과 isolated network/path를 사용한다.
- test 편의를 위해 production validation, TLS, path check, timeout 또는 redaction을 끄는 hidden flag를 추가하지 않는다.
- 테스트 전용 우회가 필요하면 test implementation을 Port 뒤에 주입하고 production binary에서 선택할 수 없게 한다.

## 16. 위험한 언어·runtime 기능

- `unsafe`와 cgo는 MVP 기본 구현에서 사용하지 않는다. 반드시 필요한 경우 별도 ADR, threat review, platform fallback과 전용 test를 요구한다.
- reflection은 typed 구현으로 대체하기 어려운 경계에서만 사용하고 security field validation을 우회하지 않는다.
- dynamic code loading, Go plugin, Artifact-provided shared library loading을 사용하지 않는다.
- finalizer에 보안 cleanup이나 필수 resource 해제를 의존하지 않는다.
- init function에서 network, filesystem mutation, subprocess 실행 또는 dependency wiring을 수행하지 않는다.
- process-global environment와 working directory를 library package에서 변경하지 않는다.

## 17. Generated code와 repository script

- generated file에는 generator와 재생성 방법을 식별할 수 있는 marker를 둔다.
- generator input과 version을 검토·고정하고 실행 시 network에서 최신 generator를 자동 다운로드하지 않는다.
- generated output이 수동 편집과 섞이지 않게 한다.
- repository script는 strict error handling을 사용하고 secret을 출력하지 않는다.
- script도 shell command injection, unsafe glob, broad recursive delete와 unresolved environment variable을 피한다.
- 제품의 security workflow나 Policy rule을 script에만 구현하지 않는다.
- generated file commit 여부와 검증 명령은 Step 10에서 확정한다.

## 18. 금지 패턴 요약

| Pattern | Rule |
| --- | --- |
| `sh -c` + 조합 문자열 | 금지 |
| Artifact 문자열을 executable/shell syntax로 실행 | 금지 |
| Host environment 전체 subprocess 전달 | 금지 |
| timeout 없는 network/process call | 금지 |
| TLS verification 비활성화 | 금지 |
| 실제 Secret/Credential 값 자체의 CLI argument·log·Evidence 저장 | 금지 |
| 외부 입력으로 panic | 금지 |
| Adapter/Provider의 Policy Decision 생성 | 금지 |
| Sandbox의 Host write/Promotion | 금지 |
| source tree 내부 production runtime data | 금지 |
| path 문자열 prefix만으로 root containment 판단 | 금지 |
| unbounded goroutine/download/extraction/output capture | 금지 |
| custom cryptography·예측 가능한 security random | 금지 |
| mutable global configuration/client/state | 금지 |
| `unsafe`, cgo, dynamic plugin의 무승인 도입 | 금지 |
| blanket lint/security suppression | 금지 |

## 19. 예외와 suppression

- 규칙 예외는 코드 편의가 아니라 구체적인 기술·보안 필요성을 근거로 한다.
- `nolint`, security scanner suppression, test skip과 allowlist에는 좁은 범위의 이유와 안전 근거를 인접 주석으로 기록한다.
- file/package 전체 blanket suppression을 사용하지 않는다.
- dependency, `unsafe`/cgo, TLS/network 완화, credential 범위 확대, Host path/write 권한 확대와 Sandbox 격리 약화는 일반 주석만으로 승인하지 않고 별도 결정 또는 ADR을 요구한다.
- 임시 예외에는 제거 조건과 추적 가능한 work item을 둔다.
- 예외가 기존 Threat Model·Architecture·Domain invariant를 변경해야 한다면 구현 예외로 처리하지 않고 해당 설계 단계로 되돌아간다.

## 20. Security Review Checklist

변경 작성자와 reviewer는 해당되는 질문을 확인한다.

### Boundary

- 이 변경은 올바른 package와 Port 경계에 있는가?
- Core/Application이 구체 Adapter·tool·runtime·storage를 새로 import하지 않는가?
- untrusted input이 Domain type으로 들어오기 전에 제한·검증되는가?
- Sandbox, Evidence, Staging과 Host Promotion 권한이 섞이지 않는가?

### Secret와 data

- secret·credential·환경변수 값이 argument, error, log, Evidence 또는 test fixture에 노출되지 않는가?
- 사용자 식별 가능 경로와 raw Artifact output에 redaction·size limit이 적용되는가?
- runtime data가 source tree 또는 과도한 권한 경로에 쓰이지 않는가?

### Execution와 network

- shell을 우회하고 executable/args/env/cwd가 분리·검증되는가?
- timeout, cancellation, process tree cleanup과 output limit이 있는가?
- TLS, redirect, destination, size와 credential scope를 검증하는가?
- retry가 bounded하고 전체 deadline을 지키는가?

### File와 lifecycle

- path traversal, link escape, TOCTOU, overwrite와 partial write를 처리하는가?
- archive/resource limit이 있는가?
- 모든 file/body/process/session/goroutine의 owner와 cleanup이 명확한가?
- cleanup 실패가 안전한 성공으로 숨겨지지 않는가?

### Result semantics

- Operational Error, Domain Result, Policy Decision과 Operation Result를 섞지 않는가?
- Capability와 Execution Status를 구분하는가?
- 미지원·미실행·실패·불완전을 빈 성공으로 변환하지 않는가?
- external verifier/scanner verdict가 최종 Policy Decision으로 직결되지 않는가?

## Step 9 Invariant

1. 기존 Threat Model·Architecture·Domain invariant가 Coding Rule보다 우선하며 이 문서는 이를 변경하지 않는다.
2. 안정 Go 버전을 명시적으로 고정하고 자동 최신 추종하지 않는다.
3. 표준 라이브러리를 우선하며 외부 dependency는 공급망 검토 후 고정한다.
4. Operational Error, Domain Result, Policy Decision과 Operation Result를 구분한다.
5. 정상 실패와 외부 입력에 panic을 사용하지 않고 error cause를 보존한다.
6. diagnostic log와 canonical Evidence를 분리하고 secret·식별정보를 기본 redaction한다.
7. 실제 Secret은 CLI argument·일반 log·Sandbox에 전달하지 않는다.
8. 외부 명령은 shell 문자열 없이 executable·args·env·cwd·limit을 명시한다.
9. path traversal, link escape, TOCTOU, unsafe overwrite와 unbounded archive를 방어한다.
10. network에는 timeout·size·TLS·redirect·credential scope 검증을 적용한다.
11. runtime resource와 goroutine에는 bounded ownership·cancellation·cleanup을 적용한다.
12. custom cryptography, 동적 code loading과 무승인 `unsafe`/cgo를 사용하지 않는다.
13. 예외와 suppression은 좁고 추적 가능해야 하며 설계 invariant를 우회할 수 없다.

## 구현 영향

- Step 10은 이 문서의 규칙을 formatter, compiler/type check, linter, test, architecture check와 security scan으로 매핑한다.
- Step 11은 deterministic check와 승인된 예외 정책을 CI Quality Gate로 연결한다.
- Step 12~13은 dependency review, fixture, backend와 security-sensitive 구현 작업을 milestone/queue에 반영한다.
- Step 14의 모든 vertical slice는 관련 Security Review Checklist를 적용한다.

## 유보 사항

- 실제 Go version과 module path
- dependency 승인 기록 형식과 자동 update 정책
- config format, credential source/broker와 proxy 정책
- Heliopause CLI exit code의 정확한 값
- log level, handler, output destination과 retention
- concrete storage path·permission·retention·atomicity
- concrete runtime process-tree control 구현
- approved digest/signature algorithm과 compatibility policy
- public schema/versioning과 generated file commit 정책
- 규칙을 검증할 formatter·linter·scanner 제품 및 세부 configuration

위 항목은 Step 10~13 또는 관련 구현 직전 결정에서 확정한다.

## 누락 점검

- [x] Go version pinning과 최소 버전 CI 검증 원칙
- [x] dependency 필요성·license·취약점·transitive·공급망 검토
- [x] package/API·constructor·zero value·global state 규칙
- [x] Domain type와 상태 축 보존
- [x] `context.Context`, timeout, cancellation 규칙
- [x] Operational Error / Domain Result / Policy / Operation 구분
- [x] panic·wrapping·typed error·exit code 규칙
- [x] `slog`, Evidence 분리와 민감정보 redaction
- [x] trusted source authentication과 Sandbox secret 금지
- [x] shell 금지, args/env/cwd/process tree/output limit
- [x] path traversal·link·TOCTOU·overwrite·archive 방어
- [x] network timeout·TLS·redirect·size·credential scope
- [x] cryptography·digest·random 규칙
- [x] input·serialization·output 검증
- [x] concurrency·ownership·cleanup 규칙
- [x] testability와 hidden security bypass 금지
- [x] `unsafe`·cgo·reflection·plugin·init 제한
- [x] generated code와 script 규칙
- [x] 금지 패턴 표
- [x] exception·suppression 정책
- [x] Security Review Checklist
- [x] 13개 invariant
