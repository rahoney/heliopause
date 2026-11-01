# 일괄 의존성 설치와 원래 보안 목표의 구현 대조

Owner: M13-004. 감사 기준 source: `fed87ce9081e3abd07ef2dec76aed7961d3cda4d`.
Status: 감사·미지원 목록 작성 완료, 아래 기능·보안 acceptance는 **OPEN**.
이번 작업은 설계→실제 호출 경로→기존 검증의 대조와 CLI 사전 검증이다. 제품 코드,
설치본, Policy, Host 설정을 변경하지 않았으며 새 설치 시험·CI·악성코드 실행은 하지 않았다.

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
| B01 | pip·UV pip requirements 설치 | NO | NO | NO | NO | `-r`/`--requirement`와 여러 root를 하나의 frozen 입력·전체 graph·승인/원자 반영에 연결. 기존 pip/UV resolver·inspection·promotion 재사용. 한 줄씩 독립 설치하거나 sync로 바꿔 기존 패키지를 삭제하지 않음 |
| B02 | npm 기본 프로젝트 설치와 ci | NO | NO | NO | NO | package.json/lock 전체 설치, ci의 lock 필수·불일치 거부·lock 무변경과 target 교체/원복. 기존 npm whole graph/update/ownership owner 재사용, 다른 client로 몰래 대체하지 않음 |
| B03 | Cargo fetch-only | NO | NO | NO | NO | 기존 complete project resolve/inspect/cache 경로를 공개 명령에 연결. Cargo.toml만 있는 최초 lock 생성과 기존 lock 보존을 별도 검증. build를 대신 실행하지 않음 |
| B04 | Terraform 기본 init | NO | NO | NO | NO | 현재 configuration의 전체 provider와 기존 lock 선택 보존. lock 없을 때 선택, unchanged 재검사, ownership/원복. upgrade를 자동 붙이지 않음; backend/cloud 실행은 별도 범위 |
| B05 | Python 일괄 입력의 일반 요구사항 문법 | NO | NO | NO | NO | 버전 범위·extras·환경 marker·주석/빈 줄/CRLF, 반복 -r와 제한된 상대 include/-c, hash 검증. 중복·충돌·include cycle/경로 탈출/파일 변경·빈 파일 no-op·부분 설치 금지까지 B01 acceptance로 묶음. 임의 index·VCS·editable은 자동 허용하지 않음 |
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
