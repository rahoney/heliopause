# 일괄 의존성 설치와 원래 보안 목표의 구현 대조

Owner: M13-004. 감사 기준 source: `fed87ce9081e3abd07ef2dec76aed7961d3cda4d`.
Status: 감사·미지원 목록 작성 완료, 아래 기능·보안 acceptance는 **OPEN**.
이번 작업은 설계→실제 호출 경로→기존 검증의 대조와 CLI 사전 검증이다. 제품 코드,
설치본, Policy, Host 설정을 변경하지 않았으며 새 설치 시험·CI·악성코드 실행은 하지 않았다.

후속 [보강 설계와 완료 기준](../../21-goal-completion-design.md)은 이 감사의 B00–B07/S01–S09와
자동 정리·시험 정리·불용 코드 확인 C01–C03의 수정 지도를 소유한다. 최초 설계 작성 당시 구현은 NOT_STARTED였다. 현재 구현 상태는 아래 표·후속 checkpoint와
Queue를 따른다. 이 감사에서 YES인 기존 retention API는 일반 자동 수거 완료를
뜻하지 않는다. 자동 수거·quota·crash 복구의 추가 baseline은 해당 설계 §2/§7을 따른다.

## 1. 기준과 증거

- 원래 요구: [D-005/D-009](../../../threat-model/01-threat-boundary-and-trust.md),
  [D-006/D-008/D-014](../../../threat-model/02-isolation-and-inspection.md),
  [D-010/D-011](../../../threat-model/03-fail-closed-and-promotion.md),
  [MVP-005/006/007](../../../mvp-scope/02-inspection-and-verification.md),
  [MVP-008/009](../../../mvp-scope/03-results-policy-and-completion.md),
  [Architecture-005/006](../../../architecture/03-inspection-sandbox-policy.md).
- 현재 좁은 구현 계약: [M2](../../../domain-model/11-m2-npm-static-contract.md),
  [M3](../../../domain-model/12-m3-linux-dynamic-contract.md),
  [M5](../../../domain-model/14-m5-pypi-pip-contract.md),
  [M6](../../../domain-model/15-m6-github-releases-contract.md),
  [M11](../../14-m11-dynamic-detection-depth-contract.md),
  [M12](../../16-m12-01-ecosystem-expansion-contract.md),
  [client09](../../../user-journey-cli-ia/09-package-client-quarantine.md).
- 기존 검증 재사용: [P7 명령별 증거](../package-client-quarantine/p7-command-acceptance-matrix-20261030.md),
  [기존 source 감사](./full-source-audit-20261031.md),
  [로컬 사용자 체험](./local-dev-test-20261101.md). 기존 성공은 그 입력·지원 범위에만 적용한다.
- 새 raw: `docs/github-logs/goal-coverage-audit-20261101/`의 `preflight.json`,
  `preflight-native-identity.json`, `source-inventory.json`, `docs.log`와 `validation.json`.
  두 `install -r`의 이전 실제 거부는 `docs/github-logs/local-dev-test-20261101/requirements-r-preflight.json`을 재사용한다.

처음 사전 검증의 네 명령은 실행 도구의 root UID 변환 때문에 Docker 소유권 검사에서
막혔다. 이 결과를 명령 미지원의 실행 증거로 쓰지 않았다. 그 네 명령만 실제 일반 사용자
신원으로 다시 확인했고, 나머지 여섯 문법 거부는 반복하지 않았다. 빈 임시 디렉터리를
사용했으며 사용자 프로젝트·개발 가상환경은 설치 대상으로 삼지 않았다.

## 2. 언어별 일괄 설치 지원

여기서 지원은 명령 이름만 존재한다는 뜻이 아니다. 원래 파일의 의도, 직접·전이 의존성,
검사·승인, 기존 설치 보존/제거와 반영까지 이어지는 현재 구현을 뜻한다. 아래 YES는
기존 계약 범위의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED가 모두 YES이며,
일반 native 전체 옵션의 동등성을 뜻하지 않는다.

| 생태계·입력 | 현재 helox 경로 | 판정·제한 | 코드·기존 증거 |
| --- | --- | --- | --- |
| Python requirements.txt | `pip install -r`, `uv pip install -r` | 둘 다 미지원. 기본 필수 추가 대상 B01 | `internal/cli/package_client.go`: configurePipInstallFlags/configureUVPipInstallFlags; 실제 이전 exit1 |
| Python 완전 고정 목록/pylock.toml | `uv pip sync <file>` | YES, public wheel의 **전이 의존성까지 포함한 완전 목록**. 정확한 버전만, 제외된 소유 패키지 제거. 일반 install의 대체가 아님 | `internal/cli/uv_pip_sync.go`, `internal/artifact/pypi/uv_sync_input.go`; TestLinuxUVSyncCompleteCLIIntegration/TestLinuxUVSyncClosureCLIIntegration |
| Python pyproject.toml + uv.lock | `uv sync` | YES, 단일 virtual project/public wheel/기존 고정 Python venv. group·extra·marker·workspace·프로젝트 자체 build 미지원 | `internal/cli/uv_project.go`, `internal/artifact/pypi/uv_project_input.go`; P7 P4 evidence |
| JavaScript/TypeScript package.json + package-lock.json | 인자 없는 `npm install`, `npm ci` | 둘 다 미지원 B02. 기존 `npm install <package>`와 `npm update`는 다른 의도 | `internal/cli/root.go`: initializeCommandTree/AddNPMInstall; 실제 사전 검증 |
| JavaScript/TypeScript package.json + pnpm-lock.yaml | `pnpm install` | YES, 기존 단일 프로젝트/고정 isolated layout. workspace 및 임의 native 옵션 미지원; 기존 lock은 manifest와 일치해야 함 | `internal/cli/pnpm_project.go`, `internal/bootstrap/pnpm_project.go`; TestPNPMProjectCLIIntegration/P7 |
| JavaScript/TypeScript package.json + bun.lock | `bun install` | YES, 기존 단일 프로젝트/고정 hoisted layout. workspace 및 임의 native 옵션 미지원 | `internal/cli/bun_project.go`, `internal/bootstrap/bun_project.go`; TestBunProjectCLIIntegration/P7 |
| Go go.mod + go.sum | `go mod download` | YES, 전체 선택 graph 검사·검증 캐시 준비. 임의 Host module cache 채우기/프로그램 실행과 구분 | `internal/cli/root.go`: AddGoModuleDownload; `internal/application/project_snapshot.go`; TestLinuxGoTransitiveGetDownloadIntegration/M12 evidence |
| Rust Cargo.toml + Cargo.lock | `cargo build` | YES, 전체 locked graph 검사·캐시 준비 뒤 관찰된 build. build.rs/proc-macro는 이 build 단계에서 관찰 | `internal/application/cargo_build.go`, `internal/bootstrap/cargo.go`; TestLinuxCargoBuildScriptBuildCLIIntegration/M12 evidence |
| Rust 의존성 준비만 수행 | `cargo fetch` | 미지원 B03. 기존 build 내부 준비를 재사용할 수 있지만 fetch-only 공개 경로 없음 | `internal/cli/root.go`: Cargo 등록 목록; 실제 도움말/exit0 |
| Terraform .tf + .terraform.lock.hcl | 인자 없는 `terraform init` | 미지원 B04. `init namespace/type@version` 또는 전체 provider를 갱신하는 `init -upgrade`는 있음 | `internal/cli/terraform_upgrade.go`: validateTerraformInitArgs; 실제 exit1 |
| GitHub Releases | `github install <owner/repo@tag#asset>` | 단일 release asset 흐름. 생태계 표준 requirements 같은 범용 일괄 설치 파일을 현재 계약으로 약속하지 않음 | `internal/cli/root.go`, M6 |

Native 의미 확인은 공식 문서를 사용했다. [pip requirements](https://pip.pypa.io/en/stable/reference/requirements-file-format/)와
[UV 파일 설치](https://docs.astral.sh/uv/pip/packages/)는 여러 요구사항을 입력받는 설치다.
[npm ci](https://docs.npmjs.com/cli/v11/commands/npm-ci/),
[PNPM install](https://pnpm.io/cli/install), [Bun install](https://bun.com/docs/pm/cli/install),
[UV sync](https://docs.astral.sh/uv/concepts/projects/sync/)는 각 파일·lock의 의미를 보존해야 한다.
[Go download](https://go.dev/ref/mod#go-mod-download)와 [Cargo fetch](https://doc.rust-lang.org/cargo/commands/cargo-fetch.html)는
의존성 준비이며 프로그램 실행이 아니다. [Terraform init](https://developer.hashicorp.com/terraform/cli/commands/init)은
기존 lock 선택을 사용할 수 있지만 `-upgrade`는 새 버전을 선택하므로 서로 바꿔 부를 수 없다.
이 문서 조회는 새 tool/runtime 버전 채택이 아니다.

### 미지원·보완 backlog

아래 상태는 **행에 적은 미완료 기능 전체**를 기준으로 판정한다. 재사용 가능한 기존
부품은 위 표와 아래 보완 경로에 명시했으며 이를 다시 구현하지 않는다.

| ID | 실제 gap | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING / 보완 경로 |
| --- | --- | --- | --- | --- | --- | --- |
| B00 | 미지원 하위 명령의 명시적 실패 | NO | NO | NO | NO | `npm ci`, `cargo fetch`, `go mod tidy`가 도움말만 출력하고 exit0. 공통 CLI 사전 검증에서 거부하되 부모 도움말/정상 명령은 유지. 현재 runtime 준비 뒤 발견되는 일부 문법 오류도 runtime 전에 판정 |
| B01 | pip·UV pip requirements 설치 | YES | YES | NO | NO | `-r`/`--requirement`와 여러 root를 하나의 frozen 입력·전체 graph·승인/원자 반영에 연결. 기존 pip/UV resolver·inspection·promotion 재사용. 한 줄씩 독립 설치하거나 sync로 바꿔 기존 패키지를 삭제하지 않음 |
| B02 | npm 기본 프로젝트 설치와 ci | YES | YES | NO | NO | package.json/lock 전체 설치, ci의 lock 필수·불일치 거부·lock 무변경과 target 교체/원복. 기존 npm whole graph/update/ownership owner 재사용, 다른 client로 몰래 대체하지 않음 |
| B03 | Cargo fetch-only | NO | NO | NO | NO | 기존 complete project resolve/inspect/cache 경로를 공개 명령에 연결. Cargo.toml만 있는 최초 lock 생성과 기존 lock 보존을 별도 검증. build를 대신 실행하지 않음 |
| B04 | Terraform 기본 init | NO | NO | NO | NO | 현재 configuration의 전체 provider와 기존 lock 선택 보존. lock 없을 때 선택, unchanged 재검사, ownership/원복. upgrade를 자동 붙이지 않음; backend/cloud 실행은 별도 범위 |
| B05 | Python 일괄 입력의 일반 요구사항 문법 | YES | YES | NO | NO | 버전 범위·extras·환경 marker·주석/빈 줄/CRLF, 반복 -r와 제한된 상대 include/-c, hash 검증. 중복·충돌·include cycle/경로 탈출/파일 변경·빈 파일 no-op·부분 설치 금지까지 B01 acceptance로 묶음. 임의 index·VCS·editable은 자동 허용하지 않음 |
| B06 | 기존 project 명령의 흔한 선택/고정 옵션 | NO | NO | NO | NO | UV group/extra/marker·locked/frozen, PNPM/Bun의 명시적 frozen-lockfile 옵션. 현재 `--group`, 두 `--frozen-lockfile`는 실제 unknown flag. PNPM의 기존 lock 일치 검증 자체는 구현돼 있음; 옵션 부재를 lock 검증 전무로 표현하지 않음 |
| B07 | Go 소스에 맞춘 의존성 파일 정리 | NO | NO | NO | NO | `go mod tidy` 미지원. `mod download`와 다른 추가·제거/소스선택 의미이므로 별도 지원 후보. B01~04 기본 일괄 설치보다 후순위 |

`pip install a b`, `uv pip install a b`도 실제 ExactArgs(1)로 거부됐다. 여러 root
입력은 B01의 같은 owner에서 다루며 별도 중복 설치기를 만들지 않는다.

Poetry/Pipenv/Yarn/Conda, Maven/Gradle, NuGet, Composer, Bundler 등은 현재 등록된
client/생태계가 아니다. 이들의 모든 명령을 기존 구현 누락으로 소급하지 않는다.
현재 지원 client의 일반적인 일괄 설치부터 닫고, 다른 client 추가는 별도 요구 범위로 남긴다.
Private registry·임의 URL/VCS·workspace·다른 Python/OS ABI도 현재 제한을 우회해 지원하지 않는다.

## 3. 원래 보안 목표에서 확인한 공백

M2는 scanner/취약점 DB/diff/signature를 초기 구현에서 제외했고 M6는 일반 sidecar
검증을 후속으로 남겼다. M11 완료는 고정 syscall 관찰과 정규화 범위다. 이는 과거의
좁은 단계 성공 근거이며 **MVP-005/006/007 전체가 구현됐다는 근거가 아니다**.
원래 설계의 체크 표시는 결정 누락 점검이지 제품 구현·검증 체크리스트가 아니다.

| ID·원래 요구 | 현재 사실·기존 재사용 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- | --- |
| S01 D-008/MVP-006 모의 개발환경 | `sandbox/authority.go`는 HOME/PATH/LANG/LC_ALL만 추가. `boundary_helper.go`는 권한 강하 helper 준비. observer는 두 고정 honeytoken 경로 분류를 보유 | NO | NO | NO | NO | 실제 권한 없는 dummy env·.env·인증 파일/개발 파일 배치, 생태계별 준비·폐기와 조건부 분기 유도. 고정 경로 접근 분류만으로 배치 완료 주장 금지 |
| S02 D-008 모의 DNS/HTTP 응답 | 표준 npm/Python/ELF 동적 경로는 network none. resolver registry 허용망은 다운로드용이며 모의 응답 서비스가 아님 | NO | NO | NO | NO | 실제 외부·Host·내부망으로 나가지 않는 fake DNS/HTTP와 bounded 후속 응답, 통신·전송 시도 관찰/판정. 실제 인터넷 허용으로 대체 금지 |
| S03 환경 조건·지연형 위협 acceptance | 기존 bounded 실행 ledger·timeout·hostile fixture는 있음. 더미 존재/응답/시간 조건을 충족시키는 표준 시나리오와 전 생태계 목표 증거는 없음 | NO | NO | NO | NO | 무해한 조건부·지연형 표본: 조건 없음→잠복, 통제 조건 있음→행위→탐지→반입 없음. 정상 대조군·한도/관찰 실패도 검증. 무제한 대기나 모든 잠복형 탐지 보장은 아님 |
| S04 MVP-005 악성·의심 코드 패턴 | `inspection/npm/static.go`는 archive/manifest/lifecycle key, `pypi/static.go`는 wheel/sdist 구조·RECORD·실행면. Go/Cargo도 archive/구문 검사 | NO | NO | NO | NO | Credential 탐색·외부 전송·난독화·동적 코드 실행 등 실제 내용 분석과 규칙/위치·Evidence·Policy 연결. 구조 검사·실행면 목록을 악성 패턴 scanner로 부르지 않음 |
| S05 MVP-005 의존성 알려진 취약점 | 일반 Artifact inspection에 advisory/CVE DB 조회 owner·연결 없음. `scripts/check/security.go`의 govulncheck는 **Heliopause 저장소** ./... 검사 | NO | NO | NO | NO | 선택된 직접·전이 package/version별 알려진 취약점 조회, DB identity/시점/실패·미지원 상태, Finding·판정. 개발 CI scanner를 사용자 artifact 검사 증거로 전용 금지 |
| S06 MVP-005 이전 버전 위험 변화 | `core/domain/project_update.go`와 `application/install_update.go`는 원본/선택 controls·ownership·교체와 무변경 guard. 새 버전 전체 재검사는 있음 | NO | NO | NO | NO | 인증된 이전/새 artifact의 script·dependency·실행 파일·권한 변화와 위험 Evidence. 단순 version/hash 변화나 원자 교체를 publisher compromise 분석으로 치환하지 않음 |
| S07 MVP-007 일반 package 공급자 증명 | 일반 npm/PyPI/GitHub는 `verification/*/integrity.go`의 registry/API digest 대조. Terraform 서명, HAA·고정 도구 releaseprovenance 검증은 이미 있음 | NO | NO | NO | NO | 제공되는 일반 package signature/provenance/attestation을 발견·검증·동일 subject에 결합하고 부재/실패/미지원 구분. 기존 고정 도구 verifier를 일반 package 전체 지원으로 확대 금지; 정상 서명도 무악성 증명 아님 |
| S08 MVP-006 상세 행위·상태 변화 | 고정 pod-init/observer는 process·open-result·network 등 정규화 count. mount topology snapshot과 설치 payload census는 있음 | NO | NO | NO | NO | 일반 동적 설치 전후 filesystem/주요 상태 diff, 생성·수정·삭제/권한 변경 범위의 명시적 근거·정규화. 현재 mount snapshot을 파일 내용/권한 diff로 부르지 않음. env의 메모리 내 읽기는 syscall 관찰만으로 직접 입증 불가하며 별도 탐지 전략/한계 필요 |
| S09 D-010/MVP-008 필수 검사 범위와 한계 | `policy/m3.go`는 전달된 required check의 완료를 확인. `core/domain/check.go`는 미완료 사유를 표현. 등록되지 않은 S01~08을 이 경로가 자동 요구하지는 않음 | NO | NO | NO | NO | 요청·생태계별 필요한 검사 목록과 실제 수행 범위 대조, scanner/DB/환경 준비 실패·미지원 표시 및 자동 ALLOW 방지. 기존 failure/observer fail-closed는 유지하며 정책 변경은 명시된 acceptance와 함께 수행 |

S04~07은 검색 결과 부재만으로 판정하지 않았다. `bootstrap/bootstrap.go`의 일반
npm/PyPI/GitHub verifier→static/dynamic composite→Policy 연결과 각 구현 body,
`bootstrap/go_module.go`, `cargo.go`, `terraform.go`의 별도 경로를 함께 대조했다.
일반 production 코드와 고정 도구의 배포 verifier, 저장소 자체 품질 검사, 테스트 fixture를
서로 구분했다. S08은 generic 생산 경로의 aggregate summary와 backend lifecycle까지
확인한 판단이며 모든 생성 파일을 전혀 검증하지 않는다는 뜻은 아니다.

### 생태계별 현재 동적 검사의 의미

| 경로 | 실제 관찰 범위 | 완료로 확대하면 안 되는 범위 |
| --- | --- | --- |
| npm/PNPM/Bun 설치 | 고정 runtime의 lifecycle 관찰, 일부 명령의 실행 전 검역 | 모든 library API·조건부 후속 실행·모의 개발환경 자극 |
| Python | typed import/startup hook/entry/script 실험과 지원되는 sdist build; 선택 실행기는 별도 | 임의 함수 입력·오랜 잠복·Host와 동일한 모든 환경 |
| Go | download는 무결성·정적/graph 검사, `helox go build`는 관찰 build | 다운로드만으로 application 실행 행위 검증 완료 |
| Cargo | add/update의 구조·graph 검사, `helox cargo build`의 build.rs/proc-macro 등 관찰 | fetch/add만으로 build-time/runtime 악성 행위 검사 완료 |
| Terraform | 서명·archive와 고정 `-help` probe | Provider RPC·plan/apply·cloud 동작 검증 |
| 일반 GitHub ELF | 인자 없는 제한 실행과 관찰 | 모든 CLI 모드·설정·지연 분기 검증 |

Terraform RPC/실제 cloud, 임의 application API의 완전 실행, macOS/Windows native backend,
지속 Host 감시·감염 치료·독자 hypervisor는 원래 한계/후속 확장과 구분한다.
D-009의 완전 탐지 불가 원칙은 유지하되 이를 S01~09의 구체 누락을 방치하는 이유로 쓰지 않는다.

## 4. 이미 있는 보호와 남은 검증

아래는 기존 source/qualification으로 구현·연결·한정 검증된 기반이다. 이번 감사에서
재구현 또는 같은 입력의 full/native/CPP/CUDA/CI 재실행 대상으로 삼지 않았다.

| 기반 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 재사용 근거·한계 |
| --- | --- | --- | --- | --- | --- |
| 전체 선택 graph/무결성·static 구조 검사 | YES | YES | YES | YES | 기존 M12/P7/전체 source 감사; S04~07 전체 탐지와 구분 |
| 격리·Host 자산 비노출·자원 한도·observer 무결성 | YES | YES | YES | YES | M3/M8/M11·source358 qualification; S01~03 유도 환경과 구분 |
| 현재 required check 실패/불완전의 fail-closed | YES | YES | YES | YES | 기존 Policy·negative 회귀; 필요한 검사를 등록하지 않은 S09와 구분 |
| exact set·digest 재확인·승인 캐시·ownership·원자 반영/원복 | YES | YES | YES | YES | 기존 update/P7/source 감사; 악성 행위 미탐지를 보완하는 안전 판정은 아님 |
| Evidence·SBOM·retention과 CLI 결과 | YES | YES | YES | YES | 기존 result/schema/manifest 검사; 미구현 검사 목록·일반 명령 성공 오인 B00/S09는 남음 |

완료 여부가 잘못 넓어진 지점은 **단계별 축소 범위의 완료를 원래 사용자 목표의 완료로
전달한 것**이다. 설계 항목마다 구현 위치·공개 caller·정상/negative 시험·실제 결과를
연결하고, 누락 항목이 구현되지 않았는데 체크를 닫지 않는 이 표를 후속 작업에서 유지한다.
과거 CI의 성공 사실을 취소하거나 모든 기능을 다시 작성할 이유는 아니다.

## 5. 보완 순서와 완료 기준

1. B00 명령 실패 의미를 교정하고, B01/B05의 requirements 입력·원자 반영 계약을 확정한다.
   첫 기본 파일, 범위/조건·include/hash, 중복/충돌·파일 교체·부분 실패·기존 패키지 보존을
   기존 공통 owner에서 묶어 검증한다. 작은 helper마다 CI를 만들지 않는다.
2. 같은 재사용 원칙으로 B02 npm install/ci, B03 Cargo fetch, B04 Terraform 기본 init을
   순차 닫는다. B06/B07은 기본 일괄 설치와 구분해 우선순위를 유지한다.
3. S09의 필수 검사·한계 표현을 S01~03 모의 환경/응답·조건부 행위 탐지와 함께 설계하고
   연결한다. 기존 관찰 authority·정규화 privacy·실 network 차단은 유지한다.
4. S04~08의 scanner·취약점·변화·공급자 증명·상태 diff를 기존 Inspection/Verification에
   연결한다. 도구/DB 신뢰·bounded parsing·실패 상태와 정상 대조군을 포함한다.
5. 최종 acceptance는 정상 설치뿐 아니라 **조건 유도 → 탐지 → Policy → 대상 무변경**,
   관찰/검사 불완전 거부, 전체 의존성 누락 없음, 기존 정상 소비자 보존으로 확인한다.
   의미 있는 변경 묶음에만 필수 CI를 수행하고 진행 중이면 실제 20분 타이머를 소비한다.

이 순서는 감사 결과의 backlog이며 위 기능을 이번에 구현했다는 뜻이 아니다.
원래 목표 acceptance가 OPEN인 동안 이름·배포를 완료 조건으로 앞세우지 않는다.

## B00 구현 착수 — 2026-11-01

기준 a678960. IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO.
MISSING: 부모에 남은 unknown child의 help exit0 및 일부 leaf의 Args 사전 검증 누락.
기존 `cli.PreparePresentation`→`bootstrap.Run`은 runtime 구성 전에 호출된다.
`Find`는 non-runnable parent를 반환할 수 있고 pipx 문자열 전용 방어만 있다.
해당 owner에서 부모 잔여 인자를 공통 거부하고 모든 leaf의 기존 Args를 재사용한다.
기존 parser/descriptor/runtime 구성은 재작성하지 않는다.
정상 parent/help·각 기존 command Args·literal execution argv·등록되지 않은 child를
표적 회귀와 bootstrap no-runtime 시험으로 확인한 뒤 canonical quick/docs를 묶는다.
runtime/helper/network 변경이 없어 이 작은 수정만의 CI는 실행하지 않는다.
기존 pipx 전용 분기와 중복 validation 목록이 공통 검증으로 대체되면 제거한다.

### B00 로컬 종료

`PreparePresentation`의 부모 잔여 인자 공통 거부와 모든 등록 명령의 기존 Args 검증을
연결했다. pipx 전용 분기/명령별 validation 목록을 제거했다. 정상 help·literal argv 유지.
수정 전 표적 실패, 수정 후 표적2.672s·CLI/bootstrap Race102.707s PASS다.
Quick의 format/drift/build/architecture/vet/Staticcheck는 통과했다. 첫 archive 후보는
상위 `/tmp/.git` 때문에 VCS 준비 실패였고 독립 임시 Git 기준으로 해결했다. 이후 Quick의
기본 테스트는 실행 sandbox의 TCP/Unix socket 금지로 실패했다. 원본 오류는 보존하고
동일 source의 정확한 기본 테스트 단계만 unrestricted에서284.122s/exit0으로 확인했다.
Quick 원래 exit1을 전체 profile exit0으로 바꾸어 기록하지 않는다.
로컬 B00 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES, MISSING=[]다.
근거는 `docs/github-logs/goal-implementation-20261101/b00-closure.json`과 해당 raw다.
Runtime/observer/권한·고정 버전·설치본은 불변이며 새 CI 없이 후속 B01+B05로 진행한다.

## B01+B05 — requirements 전체 요청 / 로컬 검증 중

기준 commit `d7688a5`. 현재 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO. 아래 부분 증거를 전체 완료나 정식 설치본 지원으로 확대하지 않는다.

기존 단일 primary의 UpgradeInput/pip report와 exact pins·완전 교체인 UVSyncInput을
완화하지 않고, 반복 `-r`/`-c`와 여러 roots를 하나의 frozen 입력으로 연결했다. 파일 수·깊이·
전체 bytes를 제한하고 cycle/경로 탈출/links/민감 파일명/env 확장/임의 index·URL·VCS·
editable을 사전 거부한다. 주석·continuation·CRLF·범위·extras·marker·hash는 실제 고정
pip parser와 각 client의 native 선택을 사용한다. 오류는 원문 대신 안전한 상대 파일명·행을 반환한다.

재사용 owner는 `uv_sync_source.go`의 held guard/file witnesses, Python/UV native session,
독립 archive/RECORD·예산 검사, `ProjectDependencySnapshot`과 whole inspection/Policy,
`PythonProjectCache` 및 additive `commitSelection`이다. 원본 RFC822/identity 검증을 공유하고
새 requirements lane의 일반 PEP 조건만 같은 고정 parser로 대조한다. 기존 CUDA/UV bounded
metadata 해석과 runtime lock/generator/update 경로는 유지한다. 새 client 출처·전체 선택·
원본·source digests를 결합하며 가짜 primary/UV 결과로 pip 위장/한 줄씩 반입은 하지 않는다.
기존 무관 패키지·사용자 파일을 보존하고 no-op은 검사·staging·commit 없이 같은 install
schema의 SKIPPED/NOT_PERFORMED를 반환한다. preflight의 반복 slice reset도 기존 owner를 쓴다.

### 실제 검증과 최초 실패

원본은 `docs/github-logs/goal-implementation-20261101/`의 명시 label에 보존한다.
시험은 보호된 기존 client를 private mount namespace에서 시험 ELF로 바인딩하고 별도 policy
helper를 사용했다. Host 설치·global service는 교체하지 않는다. 성공을 표시한 operator는
helper wait0·소유 runtime 회수·보호 파일/global service/Docker 전후 동일성까지 확인한다.

| 증거 | 확인한 범위 | 한계 |
| --- | --- | --- |
| `b01-actual-v3`, `b01-actual-v6` | pip 기본6/225.86s, UV 기본6/236.63s PASS | 후속 일반 문법·실패 사례의 전체 증거 아님 |
| `b01-actual-v11` | pip 앞12: 기본·안전한 문법 실패·marker·include/constraint/extra7nodes·공통 전이7nodes·hash 정상/누락/불일치 PASS | 시험용 Finding 합성 오류로 parent FAIL 유지 |
| `b01-actual-v12-pip-tail` | 직전 동일9versions를 HAA로 새 준비한 뒤 transitive BLOCK·실제 격리 materialization 후 실패·source drift·최종 no-op PASS; operator293.768s | V11 앞12와 합성한 부분 coverage이며 단일 pip16 parent PASS 아님 |
| `b01-actual-v13-uv` | UV 전체16/772.58s PASS; operator776.265s, no-SKIP/helper wait0/Host·Docker 동일/관찰·policy runtime 정리 | pip 단일 parent·hosted·전체 profile 성공 아님 |
| `b01-original-pep-unit`, `b01-pinned-original-graph` | 원본 identity·일반 조건·extras·선택 밖 변경·필수 원본 필드 대조; unit10.837s/pinned DATA13·18.472s PASS | DATA는 artifact 실행/검역 완료 아님 |
| `b01-cli-final-green`, `b01-body-cleanup-unit` | CLI 전체4.868s/PyPI 전체7.064s PASS; no-op schema RED→GREEN | 전체 Quick 종료 성공 아님 |
| `b01-bundle-security` | canonical security23.117s PASS | 최종 전체 보강 보안 성능 평가 아님 |

V1은 operator namespace 비교 오류로 제품 미도달, V2는 private observer 경로를 외부 Docker
이 볼 수 없어 timeout이었다. V3 UV의 일반 Python 프로그램은 UV query 전용 profile에 맞지
않았다. CPP/loader 권한을 넓히지 않고 기존 Python reader를 사용했다. V4/V5는 UV compile에
잘못 전달한 `--requirement`를 실제 exit2/stderr로 확인해 positional source로 교정했다.
V7 PySocks는 공식 wheel/release의 Requires-Python 원문이 같아도 기존 matcher가 거부했다.
새 requirements lane에서만 byte-identical binding과 고정 PEP 검증을 함께 요구한다. V8 다음
설치의 `PySocks-1.7.1.dist-info`는 기존 `wheelDistInfoMatches`로 검증해 literal 대소문자
재구성 오류를 제거했다. 자세한 실패/교정 범위는 artifact L02가 소유한다.

V9/V10의 Anyio4.16.0/SHA c66ae634는 필수 pytest11 plugin import에서 pytest 부재로 실패했다.
같은 unit/bytes의 제한 capture와 공식 archive DATA로 원인을 확인했으며 capture overlay는
복구했다. 검사를 생략/완화하지 않는다. requests/httpcore의 certifi 공유로 B01 기본 공통
전이 acceptance를 분리했고, Anyio 환경 호환성·검사 미완료는 최종16번 보안 검토에 OPEN으로
남긴다. 이 실패는 악성 탐지 성능이나 해결 성공에 포함하지 않는다. sandbox L14를 따른다.

V11의 시험용 transitive Finding helper는 복합 report를 primitive로 다시 합성했다. 제품
Domain은 유지하고 real check/Evidence를 보존하는 helper를 RED→GREEN4.812s로 확인했다.
V12는 그 뒤의 네 사례만 실제 실행해 통과한 앞12의 불필요한 반복을 피했다.

### 묶음 검사·연결·남은 acceptance

첫 Quick129.750s는 새 generic 오류 문자열 ST1005로 실패했고 해당 literal만 교정했다.
Quick v2/366.807s는 static 전체와 default48/49가 PASS이며 CLI의 옛 복수 root/extra 미지원
negative 두 행만 실패했다. 해당 행을 새 positive로 옮기고 CLI 전체를 재확인했다. 원래
Quick exit1을 보존하며 같은 불변 package를 전체 반복하지 않는다.

영향8 package Race256.368s는 제품7 package가 모두 PASS이고 checker의 기존 unknown-profile
mutation이 새 mode를 포함하지 못해 실패했다. 기대 suffix만 교정한 checker 전체 Race
100.722s가 PASS다. 제품7개와 checker 후속의 합성 증거이며 첫 묶음 exit1은 보존한다.
기존 `ci-uv-tools.sh`에 requirements mode를 추가해 protected tagged client·package cwd·ordinary
분기와32개 고유 case·parent/no-SKIP/결과를 고정했다. 새 job은 없고 workflow2.813s와 실제
awk 포함 표적1.783s는 PASS다. 새 hosted CI는 아직 실행하지 않는다.

C03 점검에서 실제 소비자 없는 parser 원문 사본/Body accessor를 제거했다. 원본 bytes의
소유·hash·drift는 기존 RequirementsInput/guard가 계속 맡는다. 문서 profile0.400s도 PASS다.
MISSING은 전체32의 grouped hosted 증거, 공통 Python 변경의 네 profile 영향/필수
최종 실제 회귀, 최종 문서 대조다. 최종16번의 Anyio 검토도 남는다. 완료된 시험 실행파일13개
693,869,279B는88개 프로세스의 fd/maps 참조0·조회 오류0과 기록된 hash/inode/소유권을
확인해 회수했다. 설치본·원본 증거·현재 SDK/cache는 보존했다.

[로컬 checkpoint](./requirements-local-20261102.json)가 정확한 source inventory와 부분/후속
검사·실제 사례 범위·정리·미완료 acceptance를 연결한다. 다음 B02는 이 검증된 구현을
재구현하지 않고 진행하며 네 profile/hosted는 후속 의미 있는 변경 묶음에서 검증한다.

## B02 — npm 기본 install/ci 착수 감사

기준 `c6331e5`. IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO. 기존 단일 install/update는 구현돼 있으므로 다시 구현하지 않는다.
MISSING은 no-argument install/ci의 전체 manifest 선택, 최초/기존/변경 lock 의미,
peer/optional·project lifecycle, 전체 검사·cache·원자 반영, CLI 결과와 실제 실패 회귀다.

Read는 보강 설계 §4.3, client leaf §2/§4, package-client 계획 §3/§5, Architecture의
Core/Application 의존 방향·Inspection/Policy·Evidence/Promotion, Domain의 project
snapshot/update/verified-set와 operation context, fail-closed·runtime maintenance다.
공식 [npm ci](https://docs.npmjs.com/cli/v11/commands/npm-ci/)의 lock 필수·manifest 불일치
실패·control 무변경을 [lock v3 형식](https://docs.npmjs.com/cli/v11/configuring-npm/package-lock-json/)
과 대조했다. Native 고정 버전은 기존 generated Node/npm identity에서 읽는다.

| 기존 owner | 재사용 / 실제 gap |
| --- | --- |
| `artifact/npm/project_controls.go`, `lockfile.go` | 단일 root/primary 계약 유지. 같은 bounded strict JSON·registry URL/SRI·path/identity 검증을 쓰되 전체 project 입력/선택은 별도 adapter 값으로 표현 |
| `application/project_update.go`, `ProjectInspectService` | whole snapshot·guard·전체 검사/Policy·staging lifecycle 그대로 재사용; 가짜 primary 없음 |
| `sandbox/npm_resolver.go` | 기존 Node/npm runtime·typed network·observer·폐기 재사용. install/ci의 정확한 native argv·control freeze/readback 및 전체 placement를 연결 |
| `promotion/npm_transaction.go`, PNPM/Bun project cache·guard | 기존 backup/journal/rollback·independent approval·cache byte 검증 공유. Npm provenance/lock/regular hoisted layout만 해당 실제 adapter로 연결; 다른 client 실행으로 대체하지 않음 |
| `cli/root.go`, `package_client.go`, 기존 project presenter | 명시 패키지 install/update 유지. 인자 없는 install과 ci만 whole service로 연결하고 preflight/JSON 결과 owner 재사용 |

먼저 bounded controls와 install/ci intent·출력 불변성의 싼 검사를 진행한다. 그 뒤 실제
native 전체 선택/registry binding→전체 검사/배치/반영→CLI/실제 정상·거부·원복을 연결한다.
Root lifecycle과 OS/CPU optional 선택은 검사 범위에 포함하고 무시하지 않는다. Project 파일을
freeze하지 않은 상태에서 script를 실행하거나 필수 검사를 완화하지 않는다. 최종 acceptance
전에는 지원/완료로 표시하지 않는다. 새 CI·job·runtime pin·Host 설치 변경은 없다.

B02 입력/whole lock·registry binding·snapshot과 native install/ci 선택을 연결하는 후보를
작성했다. 기존 single-primary parser는 유지하며 graph의 archive 중복 제거와 설치 위치·peer
edge를 구분한다. 잘못된 optional OS/CPU를 선택 제외로 오인하는 표적 RED를 교정했고,
source binding/기존 npm 소비자 포함 표적9.325s와 native 경계·실패 회귀7.972s가 PASS다.
이는 실제 반입/전체 검사 완료가 아니다.

기존 npm resolver의 lifecycle을 두 실제 선택 소비자가 공유하도록 추출하고 보호된 argv와
Node/npm identity·typed network·trace·cleanup은 유지했다. 새 CI 선택은 정확한
`ci --dry-run --ignore-scripts ...`만 observer에 연결했다. Canonical builder/Bazel과 검증한
캐시 gVisor source를 사용한 전체 C++ latch53.287s가 PASS다. 이전 캐시 확인의 operator
오류는 patch의 신규 미추적 파일을 tracked diff와 혼동한 것으로 제품/빌드 미도달 원본을
보존했다. Runtime lock와 Host 설치본을 갱신하지 않았다.

실제 DATA V1은 최초2roots/3placements·ci 원문 보존·ci version 불일치 거부·변경 manifest
선택을 각각5.230/4.242/4.223/4.330s에 통과했고 optional-peer lock 해석에서 FAIL이다.
Operator25.600s/helperwait0·Host/Docker 동일·runtime 회수를 확인했다. 그 첫 실패만
고정 native readback의 bounded test capture로 조사한다. 통과한 앞4나 CPP를 재실행하지
않는다. 전체 B02 네 상태는 NO이며 후속 actual/검사·cache·반입·CLI가 MISSING이다.


B02 첫 metadata 원인은 실제 debug4.4.3 lock1405B/SHA3bca2a3a의 선언 없는
`peerDependenciesMeta`였다. 보호 executor의 marker를 보존한 capture V2로 입력을 확보했고,
정상 주석을 선언 peer로 강제하던 validator를 교정했다. 주석은 독립 metadata로 보존하며
의존성/검사 root/edge를 만들지 않는다. 동일 원본 fixture RED2.606s 뒤 npm 전체6.079s가
PASS다. Capture V1은 wrapper가 marker를 숨긴 별도 startup 실패였으며 제품 권한 변경 없이
교정했다. 상세 교훈/원본은 artifact L02/L15가 소유한다.

`b02-native-tail-v1`은 앞4를 반복하지 않고 metadata-only 주석4.819s·실제 선언 peer
(React/ReactDOM)6.004s·Darwin-only optional 제외4.063s를 통과했다. Tail parent16.25s
PASS와 operator의 helperwait0·Host/Docker 동일·runtime 정리를 확인했다. V1의 앞4와
합성한7사례 coverage이며 V1 parent FAIL은 보존한다. 이 단계는 native DATA/source
binding/snapshot 부분 검증이고 whole inspection·cache·원자 반영·root lifecycle·CLI는
계속 MISSING이다. CPP/통과 prefix를 다시 실행하지 않고 다음 실제 연결을 진행한다.


### B02 whole inspection·approved cache·임시 target 반입 checkpoint

Native 선택 뒤에도 원래 static-all·관찰 세션·전체 instance coverage·기록 Evidence·
entry/set Policy를 모두 거친다. Bun의 hoisted tree 대조·digest와 instance 검사 driver를
현재 npm 소비자가 공유하고, 중복 Bun fact wrapper는 제거했다. 고정 Node data
materializer와 PNPM/Bun의 기존 실행·4-member transaction·인증 cache 엔진을 재사용한다.
Npm 전용 opaque layout/명령·workspace·lineage는 다른 client 권한과 구분한다.

새 exact instance 명령/위치의 전체 C++ builder/latch46.867s PASS는 이전 ci 선택-only
53.287s와 다른 후보다. `b02-quarantine-v1`의 실제 중첩3archives/3placements24.843s,
bin(1archive/1placement)14.937s, peer(3archives/3placements)27.052s는 각각 whole quarantine/Policy까지 PASS다. Parent68.27s,
operator70.139s/helperwait0·Host/Docker 동일·runtime 정리가 확인됐다. 이 시점에는
root lifecycle·Host 반입을 검사하지 않았다.

이후 cache V1은 중첩 graph 검역/승인 cache 뒤 첫 commit에서 FAIL했다. Domain update가
control 이름을 정렬하지만 새 npm guard 연결이 순서까지 비교한 것이 확정 원인이다.
기존 Bun의 이름/존재/digest 비교를 재사용한 같은 입력 RED7.643s/GREEN10.581s와 기존
transaction 회귀를 확인했다. 공통 Domain이나 Go/PNPM control 의미를 바꾸지 않았다.

`b02-cache-v2`는 중첩25.274s와 bin14.752s 두 사례를 whole 검사→current approved cache→
script-free private copy→원자적 임시 target 반영→독립 old approval 재확인까지 통과했다.
명시 control 편집은 기존 approved payload/cache가 검증될 때만 새 선택을 허용한다.
Cache 변조는 replay/새 guard를 거부하며 원래 target control을 보존했다. Parent41.42s,
operator43.212s/helperwait0·Host/Docker 동일·runtime 정리 PASS다. Cache V1 FAIL과 최초
시험의 sandbox→promotion import cycle compile 실패를 보존하며 제품 권한을 완화하지
않았다. Composition 검사는 기존 의존 방향에 맞춰 bootstrap test로 옮겼다.

기존 PNPM complete-project Application workflow를 현재 npm consumer도 쓰도록 client
label만 분리했다. 새 npm composition session이 정확한 snapshot/runtime/control을 전달한다.
B02 전체 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED는 아직 NO다. Root project
source freeze·정적/동적 lifecycle, 공개 no-argument install/ci CLI, 실제 연속 사용자 흐름,
rollback/취소 및 grouped qualification이 MISSING이다. Runtime pin/generator와 Host
설치본은 바꾸지 않았고 새 CI는 실행하지 않았다.


Canonical Quick120.748s는 format/drift/module/build/test-build/Architecture/CI/go vet까지
PASS 뒤 Staticcheck의 새 npm composition unused로 FAIL했다. 공개 CLI가 아직 연결되지
않은 실제 중간 상태이며 warning 억제·가짜 소비자를 추가하지 않는다. 현재 수행하는
연속 install/ci Application parent 및 실제 CLI 연결을 마친 후 변경된 실패 단계부터
확인한다. Default test 단계에는 미도달했고 Quick 전체 PASS로 쓰지 않는다. 완료된
quarantine/cache3 namespace의 ELF 사본6개146476631B는92 processes의 fd/maps 참조0·
queryerror0과 inode/hash 확인 뒤 제거했다. 현재 빌드·SDK·cache와 원본은 보존했다.


`b02-workflow-v1`은 새 npm composition과 기존 guarded complete-project Application을
실제로 연결했다. Install1archive14.910s→ci1archive14.215s→수정 manifest install3archives
24.760s→ci3archives24.426s가 fresh operation으로 PASS다. 변경 없는 ci는 lock bytes와
payload inode를 보존한다. 이어진 ci 불일치는 staging 미도달/기존 lock·payload·user file
보존으로 거부됐다. Parent79.70s/operator81.570s/helperwait0·Host/Docker 동일·runtime
정리를 확인했다. Root lifecycle/공개 CLI actual은 이 parent의 검증 범위가 아니다.

이후 공개 CLI의 no-argument install/ci를 같은 descriptor·preflight·JSON report로
연결하고, 명시 package install/update는 기존 흐름을 유지하는 후보를 작성했다.
검토된 npm identity는 기존 generated NodeNPMVersion에서 공급한다. 새 CLI/bootstrap의
표적 회귀4.604s는 PASS다. Root source/lifecycle 구현 전 후보이며 실제 CLI·전체 Quick·
B02 완료를 주장하지 않는다. 아직 새 CI/Host 설치/커밋을 수행하지 않았다.

다음 root 책임 지도: 기존 Go source snapshot의 anchored bounded copy/identity 검증과
`project_build_archive.go`의 local transport 소유를 재사용할 수 있는 부분을 먼저 분리한다.
기존 build output/ApprovedProjectBuild는 output publication 전제이므로 npm root 검사의
가짜 승인으로 사용하지 않는다. Root는 `project-local` byte binding이며 npm registry
artifact나 dependency primary로 위장하지 않는다. 원래 guard 아래 source를 freeze하고,
metadata/credential namespace·owned node_modules를 별도 경계로 유지하며 실행/반입 전
source 불변성을 재확인한다. 기존 Run/Evidence/Policy와 정확한 root lifecycle 관찰을
연결하고, Host lifecycle/생성 효과에 대한 기존 M4 제한을 실행 성공만으로 완화하지 않는다.
이 부분이 현재 최초 MISSING이며 뒤의 B03 작업을 먼저 착수하지 않는다.


Root source 단계의 local candidate는 기존 Go source collector를 현재 npm guard와
공유한다. Go의 기존 한도/mode/제외 의미는 유지하며 npm은 기존 workspace 상한 안에서
single-link·anchored copy를 수행하고 실행 비트를 보존한다. `.env` 등 hidden namespace와
owned node_modules를 source 복사에서 제외한다. Local identity는 `project-local/npm-source`다.
Guard의 기존 payload 검증과 source 재검증은 재귀·중복 tree census 없이 연결했다.
`b02-root-source-reader-first`는 추출 시 int/int64 비교 compile 오류로 미도달했고, 교정한
`b02-root-source-reader-green`12.181s는 새 source의 unchanged/bytes/add/mode/replacement/
symlink 및 기존 Go source·Bun/PNPM transaction 회귀가 PASS다. 이는 root static/dynamic
실행·Evidence/Policy 연결 완료가 아니다. 현재 첫 MISSING은 source transport reader→
root lifecycle 계획/실행→실제 기록/Policy와 설치 report 연결이며, 공개 CLI actual과
root hostile/rollback/cancel이 뒤따른다. Root lifecycle이 아직 연결되지 않은 개발 후보를
설치/배포하거나 B02 완료로 표시하지 않는다. 최종 formatter/Quick/Race는 의미 있는 다음
묶음에서 수행하고 통과한 old DATA prefix/기존 C++를 이유 없이 반복하지 않는다.


Root transport reader의 exact mode/bytes/hidden/traversal/link/identity/cancel 회귀와 기존
Go/Cargo input 소비자7.346s가 PASS다. 새 root lifecycle은 별도 `project-local/npm-source`
subject로 기존 Inspect Run/Evidence/Policy를 쓰며, 외부 망 없는 private source+measured
archives만으로 고정 npm의 전체 ci lifecycle을 실행하도록 연결했다. Native 동작은 고정
[npm11.19.1 ci source](https://github.com/npm/cli/blob/v11.19.1/lib/commands/ci.js)와
[공식 scripts 의미](https://docs.npmjs.com/cli/v11/using-npm/scripts/)를 대조했다. Install7개
기본 단계뿐 아니라 조건부 `dependencies` hook과 implicit binding.gyp를 검사 필요 판단에
포함한다. 비어 있지 않은 script를 Unicode trim으로 무실행 처리하지 않는다.

Root C++ 정확 명령/위치/기존 one-shot lineage·negative/topology와 전체 기존 latch48.215s
PASS다. Runtime lock/SDK patch/Host 설치본은 불변이다. Root first CLI V1은21.25s FAIL /
operator23.111s/helperwait0·Host/Docker 동일·runtime 정리다. 정상 postinstall에서
M3_DYNAMIC_OBSERVER_FAILED·authority ambiguity가 발생했으며 원인은 아직 미확정이다.
동시에 새 root inspector가 incomplete check에 Findings를 붙여 Domain 불변식에 거부됐다.
Domain을 바꾸지 않고 기존 PNPM 방식대로 accepted facts의 optional completed check와
필수 실행 coverage를 분리했다. 이제 verification/static/coverage3필수+facts1optional이다.

Root partial-facts/policy exact membership·기존 관련 검사0.935s PASS 뒤 diagnostic V2는
같은 postinstall21.04s/operator22.912s FAIL이다. 원래 관찰기 실패는 그대로지만,4 checks와
partial accepted facts가 기록돼 Root Policy의 MANUAL_REVIEW/M3_UNEXPECTED_PROCESS를
정확히 보여준다. 다른 lifecycle 사례는 아직 미도달이다. 첫 진단 helper의 depth8에서는
bounded fault 노드를 확보하지 못했으므로 현재 exact case만 깊이 제한24의 오류 타입/기존
helper enum·locator로 재확인한다. 제품 observer 권한/필수 검사/예산을 완화하지 않는다.
이 최초 unresolved failure를 조사하며 B03·추가 broad/CI 검증으로 넘어가지 않는다.


### B02 로컬 구현·연결 checkpoint — grouped qualification OPEN

[npm 로컬 원장](./npm-project-local-20261102.json)이 source80 files/삭제1, native7의
composite 범위, 실제 전체 graph·cache·원자적 target·install/ci·root4·공개 CLI를 결합한다.
Npm 원래 intent와 버전 owner를 유지하고 기존 complete inspection/기존4-member transaction을
재사용했다. 임시 native capture wrapper는 제거했다. 상세 사용자 계약은
[기본 npm install/ci](../../../user-journey-cli-ia/09-package-client-quarantine.md#npm-기본-프로젝트-설치와-ci--b02-개발-후보)가 소유한다.

첫 root stream fault의 pinned Node loader 판독은 기존 console ABI helper 재사용으로
교정했다. ARTIFACT/실제 clone·image/mount/flags와 unexpected-process·M3/M4 판정을 유지한다.
CPP RED46.815s→전체 GREEN48.949s, 실제 정상 root20.854s·tail52.10s와 공개 install/ci31.16s가
PASS다. 두 nonzero hook의 exit37도 보존했고 원본 target은 바뀌지 않았다. 관찰 완료와
Host 스크립트 실행 승인을 구분하며 root가 있으면 설치는 계속 보류한다.

Quick95.813s의 SA4000은 두 seed directory의 side-effect charge를 명시한 두 회로 교정했다.
후속 Quick258.022s는 static 단계 전부와 default47/49를 통과하고 기존 두 syntax fixture에서
실패했다. 새 npm ci의 version/lock 전제를 추가하고 미지원 시험에는 잘못된 추가 인자를
사용했다. Targeted3.085s와 전체 영향6-package Race222.851s PASS다. 두 원래 Quick FAIL을
full PASS로 바꾸지 않는다. Security18.968s/docs0.317s/tagged compile10.659s, transaction10/
CI-order9.379s와 implicit-gyp4/10.589s가 PASS다. 이후 변경은 해당 syntax fixture뿐이며
다른 default47/native/CPP를 반복하지 않았다.

기존 CI job에 native/project exact coverage를 등록했다. Synthetic parser12는 PASS지만
hosted 실행이 아니다. Peer cache와 기존 단일 npm/update, 최종 후보의 grouped hosted/
four-profile 및 최종 보안·호환성 평가는 남는다. IMPLEMENTED/WIRED=YES,
QUALIFIED/ACCEPTANCE_CLOSED=NO다. B02 완료 판정을 전체 보강 완료로 확대하지 않는다.
시험 사본34개848163547B를 task digest·owner·프로세스 참조 부재 확인 뒤 제거했으며
Host 설치본·runtime lock·SDK/캐시·실패 원본은 유지했다. 새 CI/설치/배포는 하지 않았다.
독립적인 다음 로컬 항목은 B03 Cargo fetch-only다.


## B03 — Cargo fetch-only 착수 baseline (2dba2bd)

현재 기능 판정: IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO. 공개 fetch 명령과 typed native fetch admission은 없고,
기존 bootstrap의 invalid-command 회귀가 그 사전 거부를 확인한다.

기존 재사용 owner는 구현돼 있다. `application/cargo_build.go`의
`resolveProjectSnapshot`은 기존 lock의 준비·전체 검사·cache commit이고 실제 build가
뒤따른다. `application/project_update.go:executeGuardedProjectUpdate`는 원본 guard·선택
controls·전체 검사·stage·commit·실패 사실 보존을 이미 공유한다.
`sandbox/cargo_project_resolver.go`는 anchored source/private workspace/native 선택과
metadata--locked/source binding을 소유하고 `promotion/cargo_approved_transaction.go`와
`cargo_transaction.go`는 independent approval·control/기록 원자적 반입/복원을 소유한다.
이 구현을 대체하거나 fetch 요청에서 관찰 build를 대신 호출하지 않는다.

확인된 acceptance gap은 최초 lock이다. 기존 snapshot helper는 원래 controls와 같은
snapshot만 commit하고, no-primary update guard/transaction은 기존 Cargo.lock을 전제로 한다.
따라서 새 fetch 선택을 기존 guarded selected-controls workflow에 연결하고 최초 lock을
허용하는 typed guard를 기존 transaction에 추가한다. Cargo.toml과 기존 lock bytes/신원은
보존하며 기존 update의 lock/명시 enrollment 요구를 완화하지 않는다. 새 fetch 전용
복제 pipeline·일반 framework·미래 API를 만들지 않는다.

[공식 Cargo fetch](https://doc.rust-lang.org/cargo/commands/cargo-fetch.html)는 lock이 없으면
먼저 생성하고, lock이 있으면 그 선택을 내려받는다. 대상 옵션이 없으면 모든 target의
의존성을 획득한다. 현재 지원 경계에서 최초/기존 lock을 별도 고정하고 임의 SDK/config/
registry/target argv는 받지 않는다. 버전은 기존 runtime owner를 사용한다.

MISSING: closed native fetch 선택, 최초/기존 lock의 typed guarded commit,
기존 whole inspection/cache와 연결한 CLI structured dependency-preparation 결과,
cold/warm·변조·전이 거부·취소/원복·build invocation0 및 후속 기존 build의 cache 소비.
Read: 보강 설계 §4.4, M12 계약 §5 Cargo, Source/Policy/guard와 오답노트 L20.
기존 Cargo build와 업데이트 의미·동적 검사 및 200MiB/기존 observation 한도를 유지한다.


B03 진행: closed fetch admission RED7.455s→GREEN7.548s, 첫/기존 lock·변조/취소의
structural selection6/7.440s와 typed first-lock transaction/기존 update 회귀13.874s PASS다.
이것은 public source 인증·실제 SDK·CLI·build 미실행 qualification이 아니다. 서비스/guard를
기존 selected-controls owner에 연결하는 중이며 공개 명령은 아직 미등록이다.

추가 필수 대조: [Cargo metadata](https://doc.rust-lang.org/cargo/commands/cargo-metadata.html)의
기본 graph는 비활성 optional dependency를 제외하지만 [lock 생성 의미](https://doc.rust-lang.org/cargo/reference/resolver.html#features)는
workspace의 모든 feature를 고려한다. 현재 fetch 후보가 기존 build용 기본 metadata만 사용하면
다운로드 대상 일부를 누락할 수 있다. Native optional/all-target 사례와 lock 전체 census를
확인하고 fetch 모드만 보완한다. 기존 build/update의 기본 feature 의미는 바꾸지 않는다.


B03 현재 구현·연결 checkpoint: `CargoFetchService`는 기존 guarded update workflow,
실제 complete project inspector/cache, typed fetch guard와 lock-only transaction을 재사용한다.
별도 pipeline/SDK pin/Host 변경은 없다. 공개 `helox cargo fetch`는 no-argument만 받고
`helox.project-fetch-result/v1`의 FETCH/DEPENDENCY_PREPARATION으로 build 미실행을 표시한다.
기존 update/add/build의 feature/승인/예산 계약은 유지한다.

Actual optional DATA V3의 누락은 fetch-only all-features metadata + exact locked registry
identity/checksum census로 교정했다. DATA V4 parent9.73s/operator11.582s PASS이며
source authentication/whole quarantine/build 증거는 아니다. 별도 full CLI V2는
parent33.64s/operator35.509s, cold first-lock/warm existing-lock 두 required 검사/ALLOW,
빌드 스크립트 미실행, cache receipt 변조 거부와 lock/승인 보존, 후속 기존 관찰 build의
승인 ELF 생성, helperwait0/Host·Docker 불변/빈 runtime을 확인했다. CLI V1의 첫 두 fetch는
성공했지만 시험이 project marker에 cache handle이 있다고 가정해 실패했다. Controller의
독립 approval에서 handle을 읽는 시험만 수정했으며 제품 실패로 재분류하지 않는다.
기존 build는 retained approval/cache를 재검증하고 동일 locked bytes를 다시 검사·stage한다.
같은 handle을 무검사 소비하거나 원격 qualification이 끝났다고 표시하지 않는다.

첫 lock transaction 네 phase의 실패는 원래 absent lock/manifest 신원·mode/독립 승인 부재와
최초 오류를 복원했다. 기존 CI job에 native/CLI 소비자를 연결하며 첫 위치가 PNPM의 필수
compile/client/helper/restore 순서를 끊은 것을 checker가 거부했다. 기존 순서를 유지한
별도 Cargo 구간으로 이동한 checker4.140s PASS다. Security18.247s PASS이고 Quick와
관련 소비자 최종 대조는 진행 중이다. 원본은 `goal-implementation-20261101/b03-*`가 소유한다.
판정은 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다.
MISSING은 최종 소비자 회귀·변경 묶음 hosted/four-profile/final acceptance다.

최종 B03 로컬 checkpoint는 [정규화 receipt](./cargo-fetch-local-20261102.json)가 소유한다.
Quick246.721s의 정적 단계는 PASS이고 기본 테스트는 stale fetch negative 및 socket 권한으로
FAIL했다. Fixture를 unsupported extra 인수로 고친 뒤 영향받는 전체6패키지 Race250.376s가
PASS했다. 새 fetch도 첫/전이 의존성의 BLOCK/review/incomplete/integrity 거부를 기존 시험으로
확인했다. 전체 Quick exit0 또는 hosted 완료로 쓰지 않는다. Security18.247s/docs0.344s PASS,
완료된 protected 복사본8개179440646B는 전체 process 참조0/query error0 뒤 삭제했다.
B03의 코드·연결 로컬 단위를 기록하고 다음 로컬 단위 B04 baseline으로 진행한다.


## B04 — Terraform 기본 init 착수 baseline (05c1bfd)

IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO.
기존 명시 provider init와 -upgrade는 구현되어 있으며 다시 구현하지 않는다.
No-argument init은 cli/terraform_upgrade.go:validateTerraformInitArgs에서 거부된다.

재사용 책임 지도: cli/root.go:AddTerraformInit 및 terraform_upgrade.go의 현재 request 분기 →
bootstrap/terraform.go의 기존 client/promoter/inspection/cache →
application/project_update.go:executeGuardedProjectUpdate →
artifact/terraformprovider/project_resolver.go:resolveProjectSelection →
기존 서명·archive·격리 help 검사/Evidence/Policy →
promotion/terraform_approved_transaction.go:Commit와 terraform_transaction.go의 단일 반입/복원.
Provider graph의 독립 roots를 임의 primary/전이 edge로 꾸미지 않는다.

MISSING: 명시 no-primary 기본 init intent, 기존 lock의 선택·bytes·inode/mode 보존과 최초
lock 선택, 기존 전체 provider 검역/cache·원자 설치 연결, 명확한 INSTALL 결과와 지원 제한,
최초/반복 init·constraint 불일치·서명/원본 변경 거부·취소/원복 및 기존 명시 init/upgrade 회귀.
기존 resolver의 upgrade=true를 기본 init에 전용하면 최신 버전으로 바뀌므로 사용하지 않는다.
기존 lock이 있는 기본 init은 현재 lock의 모든 required provider/version/hash를 사용하며
누락/constraint 불일치/extra locked provider는 명시 거부한다. 없는 lock만 기존 stable
constraint selector를 재사용한다. Backend/cloud/remote module/ambient config 요청은0이다.

수정 범위: 기존 native-free registry resolver의 현재 selection helper, 기존 typed service/
bootstrap/CLI, 기존 guarded promoter의 no-primary init intent와 lock 보존 조건, 해당 실제
소비자 시험/현행 CI 연결/사용자 문서. Pipeline·도구 pin·서명 trust table·전역 runtime를
교체하지 않는다. 단일 복제 검사 service나 미래 abstraction은 만들지 않는다.
Read: 보강 설계 §4.5, M12 계약 §6, CLI09, artifact L21와 공용 sandbox/evidence 교훈.
공식 의미는 [init](https://developer.hashicorp.com/terraform/cli/commands/init)과
[dependency lock](https://developer.hashicorp.com/terraform/language/files/dependency-lock)의
기존 선택 보존/upgrade 구분을 따른다. 현재 bounded provider-only 지원은 그대로 유지한다.


B04 로컬 구현·연결 checkpoint: 기존 selected-controls workflow를 호출하는
`application/terraform.go:InitializeTerraformProject`, 기존 registry selection의 기본 init
모드와 typed `BeginInit`, 기존 lock 보존 transaction/CLI branch를 추가했다. 별도 service
class/검사 pipeline/SDK·서명 pin/Host 수정은 없다. No-argument init은 INSTALL이고 기존
명시 provider init·-upgrade/--adopt는 원래 의미를 유지한다. 기존 lock은 전체 required
provider·현재 constraints와 맞아야 하며 bytes·inode/mode를 보존한다. 없는 lock만 최초 stable
선택을 만든다. 불완전/빈 기존 lock은 원격 요청 전 거부하며 backend/cloud/module/ambient
mirror를 사용하지 않는다. 일반 Terraform의 자동 lock 확장과 구분되는 bounded 계약이다.

기존 required actual parent에 default-complete-project를 추가해 official random3.7.2와
partner github6.6.0의 최초/반복 CLI 두 calls를 parent33.08s/operator35.004s PASS로 확인했다.
각 Provider 필수3검사/새 Run·signed source/static/dynamic/Evidence/ALLOW/cache/원자 설치,
독립 공개 ELF pin·반복 lock 신원/0640 mode, helperwait0/Host·Docker 보존/빈 runtime을 확인했다.
기존 actual6calls를 로컬에서 반복하지 않았고 같은 CI parent가 새 case를 자동 소비한다.
첫 표적19.776s는 application 시험의 err 중복 선언 때문에 FAIL(다른4 package PASS)이며
변수 교정 뒤 application1.410s PASS다. 새 empty-lock negative1.048s FAIL은 존재하는 빈 파일을
부재로 오인하여 fake 공개 요청6회를 시작한 원인을 확정했다. Lock Present를 별도로 검증한
현재 Terraform/CLI preflight/공용 guard 표적4패키지 Race27.434s는 PASS다. 최초 설치 원복5
phase와 각 독립 root 거부도 포함한다. 실제 CLI는 마지막 empty-lock/nonempty-set 안전 거부
보강 이전 후보이며 final exact-source/hosted를 대신하지 않는다.

[정규화 B04 receipt](./terraform-init-local-20261102.json): IMPLEMENTED=YES / WIRED=YES /
QUALIFIED=NO / ACCEPTANCE_CLOSED=NO. Canonical/static/security·hosted/four-profile/final은
다음 의미 있는 변경 묶음과 합친다. 완료된 protected 복사본2개57363170B는 process 참조0/
query error0 확인 뒤 삭제했다. 다음 로컬 단위는 B06 UV/PNPM/Bun 선택·lock 옵션 baseline이다.


## B06 — UV 선택·lock 및 PNPM/Bun frozen 착수 baseline (77e5611)

전체 보강 기능은 IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO. 다음 기존 부품은 YES이며 다시 구현하지 않는다.

- PNPM/Bun native install은 기존 lock이 있으면 이미 --frozen-lockfile을 사용하고 parser와
  whole graph/independent source/승인/transaction을 거친다. CLI 명시 옵션만 없으며 lock 부재는
  기존 일반 install에서 최초 선택이다. `sandbox/{pnpm,bun}_resolver.go`와
  `artifact/npm/{pnpm,bun}_selection_input.go`를 그대로 재사용한다.
- UV virtual project는 현재 단일 기본 dependencies만 읽고 group/extra/marker는 명시 거부한다.
  `artifact/pypi/uv_project_input.go`의 기존 `uv-project-intent.json`이 intent/selector/runtime
  digest를 이미 bind한다. 새 options도 이 canonical control에 결합하고 별도 control/pipeline을
  추가하지 않는다. 기존 native resolver가 lock→frozen export→materialize, 독립 source binding,
  전체 wheel 검사/cache/프로젝트·venv 반입을 소유한다. 내부 export의 --frozen은 공개 frozen
  mode가 이미 구현됐다는 뜻이 아니다. 현재 --no-default-groups 고정도 새 dev 선택과 대조한다.

수정 책임 지도: CLI의 실제 요청/flag descriptor → bootstrap의 request-local session →
기존 frozen input/intent control → 고정 native 선택/lock/export → 기존 graph/source/전체 검사/
승인 cache/원자 반입·정리. PNPM/Bun frozen은 새 resolver mode/검사 service 없이 CLI request의
lock 필수 의도를 원본 guard 뒤 확인한 뒤 기존 lock-present native 경로를 사용한다. Preflight
뒤 lock이 사라져도 최초 선택으로 바뀌지 않도록 guarded controls에서 다시 확인한다.

UV Read/수정 owner: 보강 §4.6, CLI09, `cli/uv_project.go`, `bootstrap/uv_project.go`,
`artifact/pypi/uv_project_input.go`, `uv_project_lock.go` 및 실제 native lock/export parser,
`sandbox/uv_project_resolver.go`, 기존 promotion/project preparation. B01의 requirement 문법과
marker 지원은 재사용 가능 범위를 확인하며 기존 project 전체 graph/source 검사를 생략하지 않는다.
Unknown fields를 무시하거나 native frozen을 locked로 바꿔 통과시키지 않는다. Default dev/group,
선택/제외/충돌, extras/marker closure와 locked vs frozen의 서로 다른 의미를 실제 고정 UV와 대조한다.

MISSING: 위 공개 옵션과 bound input/실제 소비자, default/선택/제외/미존재/충돌의 명확한 규칙,
missing/stale/malformed lock·bytes 불변·원본 경쟁/취소/원복, 실제 native 결과와 검역/최종 반입 비교,
기존 sync/add/install/update 회귀 및 최종 묶음 canonical/security/hosted qualification.
먼저 기존 구현을 거의 그대로 쓰는 PNPM/Bun frozen 공개 연결을 완료한 뒤 UV 선택을 보강한다.
Runtime lock/generator/버전 유지보수·Host 설치는 변경하지 않는다. 새 CI는 시작하지 않는다.


B06 첫 PNPM/Bun frozen 연결 checkpoint: 공개 옵션을 install에만 등록하고 명시 false/중복/
다른 intent와 missing/stale/malformed lock은 preflight에서 거부한다. Request의 lock 필수
의도를 기존 guard 뒤 immutable controls에서도 재확인한 뒤 원래 native lock-present 경로를
사용한다. 새 resolver/input mode/pipeline은 없다. 기존 constructor에 request-local flag만
전달했고 새로운 독립 lock parser를 만들지 않았다. CLI·bootstrap 표적10.194s PASS다.
PNPM의 기존 ParseOutput은 frozen native argv만으로 lock byte 보존을 가정하여 단순 comment
추가를 수용했다. 직접 RED2.675s 뒤 기존 Bun과 같은 원본 bytes 비교를 PNPM의 기존 parser에
추가했다. 네 package selection/CLI/bootstrap 회귀13.752s PASS이며 실제 새 CLI/묶음 검증은
남는다. [PNPM](https://pnpm.io/cli/install)의 native frozen은 부재·불일치를 거부하지만
[Bun](https://bun.sh/docs/pm/cli/install)은 lock 부재에서도 잠그지 않고 설치할 수 있다.
HAA는 설계대로 frozen 요청의 lock 부재를 별도로 거부하며 native 전체 동등성을 주장하지 않는다.

UV는 아직 수정하지 않았다. 기존 strict native lock parser가 group/extra/marker와 여러
conditional variant를 읽지 못하므로 필드를 무시하여 풀지 않는다. 현재 fixed UV0.13.4의
실제 native lock/pylock/flag 의미를 DATA로 먼저 수집하고, 기존 canonical intent control에
선택을 bind하여 전체 source/graph/검사·target 원복을 보강한다. 이 발견을 새 framework나
새 runtime pin의 필요로 확대하지 않는다. B06 전체 I/W/QUALIFIED/ACCEPTANCE_CLOSED는
아직 완료가 아니며 원본은 goal-implementation-20261101/b06-*가 소유한다.


B06 native UV DATA 수집 V1(parent4.90s/operator6.766s)과 V2(4.51s/6.388s)는
모든 선택 후 lock 불변·helperwait0·Host/Docker 보존/빈 runtime을 확인했다. 실제 고정
UV0.13.4의 export는 비활성 platform marker와 같은 이름의 조건부 두 version도 포함하며,
중첩 group은 root dev-dependencies에 펼쳐지고 root default-groups가 별도로 기록된다.
따라서 native 전체 lock/portable export와 현재 target의 검사·설치 집합을 같다고 가정하지 않는다.

UV 후속 수정 지도 보완: 현재 strict wire에 실제 관측한 group/extra/marker/version/source
필드만 추가하고 universal identity를 name+version+source로 검증한다. Marker와 extra는
B01의 고정 native PEP 440/508 parser 및 authenticated metadata graph validator를 재사용한다.
기존 additive RequirementsSelection 전체를 가짜 project/installation context로 포장하지 않는다.
필요한 공통 metadata 검증 부분만 두 실제 소비자에서 함께 쓰고 기존 B01 전용 의미는 유지한다.
기존 uv-project-intent control에 선택 옵션을 bind하고 source/active target proof의 책임을
명시한다. Frozen은 기존 lock을 기준으로 선택하며 현재 manifest와 일치시켜 다시 풀지 않는다.
새 CLI/도메인 허용은 해당 실제 소비자가 완성되기 전에 지원 완료로 표시하지 않는다.

[부분 정규화 receipt](./project-frozen-local-20261102.json)는 PNPM/Bun frozen 연결과 UV DATA의
서로 다른 범위를 기록한다. 임시 DATA capture test는 원본 source/lock/export/help와 hash
inventory를 raw에 보존한 뒤 main/scratch에서 제거했다. 완료된 protected 복사본4개64799982B는
참조0/query error0 확인 뒤 삭제했다. UV parser/선택 구현은 아직 미착수이며 B06만 IN_PROGRESS다.
