# 원래 목표 보강 — 구현 설계와 완료 기준

작성일: 2026-11-01. 기준 commit: `df3b62da8c1e8e580ad9a7304bcc69c395fe65c3`.
상태: **설계 문서 작성. 아래 보강 구현은 모두 NOT_STARTED이며 실제 실행 승인이 아니다.**
사용자의 이번 요청은 설계까지다. 제품 코드·설치 환경·테스트 아티팩트를 변경하거나
새 CI·타이머를 실행하지 않는다. 후속 구현 지시 전까지 이 경계를 유지한다.

## 1. 목적·문서 책임·범위

이 문서는 [목표 대조 감사](./evidence/m13-004-convenience/goal-coverage-audit-20261101.md)의
B00–B07/S01–S09와 이번에 요청한 자동 용량 관리·시험 정리·불용 코드 확인을
실행 가능한 수정 단위로 연결한다. 감사는 현재 사실, 이 문서는 **구현할 변경과 검증 기준**,
[Queue](./02-current-work-queue.md)는 현재 실행 상태를 소유한다. 과거 qualification을 취소하거나
이 설계 작성만으로 미구현 기능을 COMPLETE로 바꾸지 않는다.

목표는 사용자가 일반적인 의존성 파일로 요청한 **전체 선택 집합**을 격리된 경계에서
검증·검사하고, 필요한 검사에 공백이 없을 때에만 동일 bytes를 반입하는 것이다.
공급자가 침해된 정상 배포물도 정적·동적 검사 대상이며 서명 성공만으로 허용하지 않는다.
더미 환경·응답은 조건부 행동을 관찰할 기회를 늘린다. 임의 함수·미지의 조건·장기 잠복까지
모두 실행했다거나 모든 악성 파일을 탐지한다고 주장하지 않는다.

기존 Architecture/Domain/Security invariant는 다음 문서가 계속 소유한다. 아래는 해당
계약을 지키는 보강안이며, 구현 단계에서는 변경되는 세부 계약만 해당 leaf에 반영한다.
계약과 충돌하는 구현 편의상의 완화는 허용하지 않는다.

| 작업 | 구현 전에 읽을 canonical 문서 |
| --- | --- |
| 공통 구조·단순성 | [Foundation](../architecture/01-foundation-and-dependencies.md), [Directory](../engineering/01-directory-structure.md), [Coding rules](../engineering/02-coding-security-rules.md) |
| 일괄 입력·선택·반입 | [Client 계약](../user-journey-cli-ia/09-package-client-quarantine.md), [Verified Set](../domain-model/05-dependency-verified-set.md), [Artifact Port](../domain-model/07-contract-artifact-port.md), [Promotion 계약](../domain-model/09-contract-evidence-staging-promotion.md) |
| 검사·판정·더미 환경 | [원래 검사 요구](../mvp-scope/02-inspection-and-verification.md), [격리 위협 모델](../threat-model/02-isolation-and-inspection.md), [Inspection 계약](../domain-model/08-contract-inspection-sandbox-policy.md), [M11 관찰·privacy](./14-m11-dynamic-detection-depth-contract.md) |
| 증거·용량 관리 | [Evidence와 Staging](../architecture/04-evidence-staging-promotion.md), [Evidence 계약](../domain-model/09-contract-evidence-staging-promotion.md), [M12 기존 자원 예산](./16-m12-01-ecosystem-expansion-contract.md) |
| 검증·CI | [Quality toolchain](../engineering/03-quality-toolchain.md), [CI gate](../engineering/04-ci-quality-gate.md), [기존 실행·타이머 규칙](./20-package-client-quarantine-plan.md#5-검증-matrix와-자원-정리) |

개인 오답노트 INDEX와 해당 artifact/sandbox/evidence/ci 주제도 작업 재개 때 읽는다.
오답노트를 새 제품 입력이나 CI 의존성으로 만들지 않는다.

## 2. Baseline과 재구현 금지선

아래 판정 단위는 각 행에 적힌 **전체 보강 기능**이다. 기존 부품은 마지막 열에서 따로
식별한다. YES인 부품을 새 이름의 service/class/helper로 다시 만들지 않는다.

| ID | 보강 기능 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 재사용 근거 / MISSING |
| --- | --- | --- | --- | --- | --- | --- |
| B00 | 미지원 명령의 명시적 실패 | NO | NO | NO | NO | CLI 사전 검증 있음; npm ci/Cargo fetch/Go mod tidy의 help exit0 교정 |
| B01 | pip/UV pip 파일·다중 root 설치 | NO | NO | NO | NO | Python resolver/inspection/transaction 있음; 하나의 일괄 INSTALL 연결 |
| B02 | npm 기본 install/ci | NO | NO | NO | NO | npm graph/update/ownership 있음; 프로젝트 입력·ci 의미 연결 |
| B03 | Cargo fetch | NO | NO | NO | NO | build 내부 project resolve/검사/cache 있음; fetch-only 공개 경로 |
| B04 | Terraform 기본 init | NO | NO | NO | NO | 명시 provider init/upgrade 있음; 기존 lock 보존 전체 init |
| B05 | requirements 일반 문법 | NO | NO | NO | NO | UV sync의 exact 목록 parser 있음; 범위·extra·marker·include·constraint·hash |
| B06 | project 선택/lock 옵션 | NO | NO | NO | NO | UV/PNPM/Bun 기본 흐름 있음; group/extra/marker/locked/frozen 의미 |
| B07 | Go mod tidy | NO | NO | NO | NO | Go graph/guard/관찰 build 있음; 소스 기반 선택·제거와 원자 controls 반영 |
| S01 | 모의 개발환경 | NO | NO | NO | NO | 최소 env/격리와 honeytoken 경로 분류 있음; 실제 더미 배치·소비·정리 |
| S02 | 모의 DNS/HTTP | NO | NO | NO | NO | 실제 외부망 차단 있음; 격리 내 제한 응답·행동 유도 |
| S03 | 조건부·지연형 목표 시험 | NO | NO | NO | NO | bounded ledger/hostile 시험 있음; 잠복→자극→행동→차단의 전체 증거 |
| S04 | 악성·의심 내용 분석 | NO | NO | NO | NO | 구조/metadata 검사 있음; 내용 규칙·위치·완전성·Finding |
| S05 | 사용자 의존성 취약점 검사 | NO | NO | NO | NO | whole graph/SBOM 있음; advisory 조회·누락/오류·Policy 연결 |
| S06 | 이전/새 버전 위험 변화 | NO | NO | NO | NO | update 원본/ownership guard 있음; 인증된 두 inventory의 보안 diff |
| S07 | 일반 package 공급자 증명 | NO | NO | NO | NO | digest 및 일부 전용 서명 검증 있음; 일반 배포물 증명 discovery/검증 |
| S08 | 동적 실행 전후 상태 변화 | NO | NO | NO | NO | observer summary/mount census 있음; 파일·권한 변화 및 관찰 한계 |
| S09 | 요청별 필수 검사 완전성 | NO | NO | NO | NO | 전달된 Required 실패는 거부; 아예 누락된 required unit 검출 |
| C01 | 제품 자동 정리·유한 보존 | NO | NO | NO | NO | session Close/실패 temp 정리/명시 DeleteRun 있음; 전체 수명·quota·crash 복구 |
| C02 | 이번 보강 시험 payload 정리 | NO | NO | NO | NO | 기존 시험 정리 방식 재사용; 새 시험별 inventory·회수 증거는 실행 뒤 생성 |
| C03 | 이번 변경의 잔여 코드 정리 | NO | NO | NO | NO | 기존 checker/좁은 caller 감사 재사용; 대체된 경로의 불용 여부는 구현 뒤 확인 |

C01의 추가 근거: `internal/evidence/local/store.go: Store.DeleteRun`은 명시적 retention
동작이며 현재 `internal`의 호출은 해당 store 시험뿐이다. `Store.Record`는 실패한 임시
기록을 정리한다. `internal/bootstrap/bootstrap.go: Run`은 runtime/executor/observer를
닫고, `internal/application/install_prepared.go`와 `project_snapshot.go`는 guard를 닫는다.
`internal/promotion/staging.go` 및 Go/Cargo/Python/PNPM/Bun/Terraform cache owner는
성공한 보관물을 만든다. **개별 Close 또는 DeleteRun 존재는 일반 자동 수거의 증거가 아니다.**
기존 정리가 전혀 없다는 뜻도 아니다. 이 확인은 source 대조이며 디스크 누적량 실측은 아니다.

기존 전체 graph 검사, digest 재확인, Host 자산 비노출, 최소 권한, observer 완료 권위,
현재 required fail-closed, exact set 승격·원자 반영/롤백은 원래 acceptance를 유지한다.
필요한 것은 그 앞뒤의 MISSING 연결이다. 기존 UI·명명·배포 채널·runtime 버전 변경은
이번 보강과 독립이며 수정 범위에 넣지 않는다.

## 3. 구현 방식과 의존 순서

### 3.1 수정 전 반드시 작성할 작은 책임 지도

각 작업은 Queue에서 하나만 IN_PROGRESS로 두고 다음 다섯 가지를 해당 작업 기록에 남긴다.
새 보고서 체계를 만들지 않고 기존 evidence leaf를 갱신한다.

1. 재현되는 입력/요구와 현재 결과, exact source·환경·최초 실패. 추정은 가설로 표시한다.
2. 기존 owner/file/symbol → 실제 caller → 결과/정리 경로와 고칠 한 지점.
3. 재사용할 함수·타입, 실제 변경할 파일, 새 코드가 필요한 이유와 직접 소비자.
4. 정상·거부·실패 시험, 검사 profile, CI가 지금 꼭 필요한지와 이유.
5. 삭제될 옛 분기/임시 진단, 보존할 증거, 완료 판정에 아직 부족한 항목.

기존 owner 수정으로 해결할 수 있으면 그곳을 고친다. 새 인터페이스는 실제 신뢰 경계나
현재 복수 소비자의 공통 책임이 있을 때만 만든다. 인자 전달만 하는 wrapper, 파일마다
한 함수 분할, 생태계별 동일 orchestration 복제, 미래 plugin framework는 만들지 않는다.
반대로 원자 transaction과 검사/삭제 권한을 거대한 하나의 함수로 합치지도 않는다.
구현 대체가 필요하면 AGENTS의 기존 위치·재사용 불가 이유·영향 제시와 승인 기준을 따른다.

### 3.2 공통 처리 경로

```text
CLI 의미 검증 → 입력/원본 freeze + 대상 guard
→ 기존 격리 resolver → 전체 선택 graph와 획득물 결합
→ 요청별 필수 check 집합 확정 → Verification + Inspection + Evidence
→ exact coverage 대조 → 기존 Policy → ALLOW 집합/Manifest
→ 기존 Staging → 동일성 재확인 → 기존 Promotion/Commit 또는 rollback
→ 소비자 종료·정리 확인 → 보존 정책 적용 → 사용자 결과
```

하위 패키지 하나의 성공으로 전체를 승인하지 않는다. 자료 획득과 실행 검사는 같은
network 권한을 공유하지 않는다. 결과/cleanup은 첫 오류를 보존하며 별도 상태로 합친다.

### 3.3 순차 작업 단위

모두 NOT_STARTED다. 번호는 실행 순서이며 새 milestone/동시 IN_PROGRESS를 만들지 않는다.
후행 작업의 독립적인 로컬 작업은 선행 원격 qualification 대기 중 계속할 수 있다.
다만 선행 실패 결과를 신뢰 전제로 사용해서는 안 된다.

| 순서 | 닫을 gap | 선행조건·경계 | 완료 시 필요한 핵심 결과 |
| --- | --- | --- | --- |
| 01 | B00 | 현재 실제 false success 재현 재사용 | 지원되지 않은 요청은 runtime 준비 전에 실패, 정상 help 유지 |
| 02 | B01+B05 | 01; Python input/INSTALL 의미 확정 | 두 client 동일 파일 문법·전체 승인·비관련 설치 보존·원복 |
| 03 | B02 | 01; 기존 npm guard | 기본 install과 ci의 서로 다른 lock 동작 |
| 04 | B03 | 01; 기존 Cargo project 준비 | fetch-only, 최초 lock/기존 lock, build 미실행 |
| 05 | B04 | 01; Terraform 전체 update 경로 | 기존 선택 보존 init, upgrade와 구분 |
| 06 | B06 | 02; 기존 UV/PNPM/Bun 의미 | 선택 group/extra/marker 및 명시 lock 옵션 |
| 07 | B07 | 기존 Go snapshot/update guard | 소스 기반 tidy와 download의 구분 |
| 08 | S09 | 앞선 command/operation 목록 확정 | expected vs actual check exact membership, 누락 자동 허용 없음 |
| 09 | C01 기본 수명·예산 | 08; §7 storage ownership 확정 | 모든 후속 검사의 생성 자원 등록·정리/보존 계약 |
| 10 | S01+S08 | 08–09; 기존 observer 권위 | 더미 배치·전후 diff·privacy·준비 실패 거부 |
| 11 | S02+S03 | 10; fake 응답 신뢰 경계 검증 | 조건 유도·행동 탐지·반입 차단·정상 대조 |
| 12 | S04 | 08–09; scanner 공급/규칙 검토 | 내용 분석 실제 실행·부분 parsing/실패 판정 |
| 13 | S05 | 08–09; graph identity 고정 | 전이 의존성까지 advisory 조회·불완전 거부 |
| 14 | S06 | 12–13; 인증된 이전 inventory | 버전 위험 변화 Evidence와 Policy 연결 |
| 15 | S07 | 08; 기존 verifier/공급자 trust 검토 | 일반 증명 subject/issuer 검증·부재/실패 구분 |
| 16 | C01 전체+C02+C03·통합 acceptance | 01–15의 누락·실패 해소 | 유한 디스크·잔여 코드/시험 payload 정리·목표 전체 증거 |

§9의 검증 묶음을 사용한다. 이 표의 각 행마다 CI를 실행한다는 뜻이 아니다.

## 4. 파일 기반 설치 보강 설계

### 4.1 B00 — 명령 실패를 성공으로 보이지 않게 한다

- 수정 owner: `internal/cli/presentation.go: PreparePresentation`의 command 탐색·flag/Args
  사전 검증과 `internal/cli/root.go: initializeCommandTree`. descriptor는
  `internal/cli/package_client.go`의 기존 정의를 사용한다.
- `Find`가 부모 command까지만 찾은 경우 남은 token을 부모 도움말로 소비하지 않는다.
  실행 가능한 등록 leaf인지, 그 leaf의 Args가 유효한지 runtime 초기화 전에 확인한다.
  특정 문자열 세 개를 blacklist로 붙이지 않는다. 기존 pipx 전용 중복 방어도 실제 공통
  검사로 흡수되는 부분만 제거한다.
- 부모만 입력한 도움말, `--help`, UI 시작, 유효한 leaf, 실행기의 `--` 뒤 argv 전달은 유지한다.
- 완료: npm ci/Cargo fetch/Go mod tidy의 현재 미등록 상태와 임의 다른 unknown child는
  exit≠0, resolver/Host runtime 호출 0. 이후 해당 명령 구현 시 정상 leaf로 같은 검증을 통과.
  도움말 exit0을 실제 설치 시험으로 집계하지 않는다.

### 4.2 B01+B05 — requirements와 여러 root를 하나의 INSTALL로 처리한다

**수정 지도:** CLI `package_client.go`의 pip/UV pip args·flags → bootstrap의
`python_client.go`/`uv_pip_sync.go`/`uv_project.go`에 있는 기존 Python 연결 →
`internal/artifact/pypi`의 입력/선택 adapter → `application`의 기존 project inspection →
`promotion/python_project_cache.go`, `pypi_venv_transaction.go`, `pypi_update.go`.
`uv_sync_input.go`의 exact sync 계약을 일반 requirements 의미로 덮어쓰지 않는다.

지원할 입력은 `helox pip install -r requirements.txt`, `helox uv pip install -r requirements.txt`,
반복 `-r`/`--requirement`, 여러 positional requirement와 그 조합이다. `helx`를 새 alias로
추가하지 않는다. 두 client의 입력 freeze/정규화는 공유하고 native resolver adapter만 다르다.

| 입력·상태 | 설계 동작 |
| --- | --- |
| 이름/정확 버전/버전 범위/extras/환경 marker | 고정 runtime의 공식 requirement 의미로 선택. Host 환경이 아닌 선언된 대상 Python/platform marker를 사용 |
| 주석·빈 줄·CRLF·공식 continuation | bytes/line 위치를 보존한 bounded parsing. shell 해석 금지 |
| 상대 `-r` include·`-c` constraint | 첫 파일의 허용 입력 root 안에서만 각 파일 기준 상대 경로 해석. 파일 수/총 bytes/깊이 상한, cycle·link escape 거부. constraint는 설치 root를 추가하지 않음 |
| hash 지정/require-hashes | 최종 선택 bytes의 digest와 결합. hash 모드의 전체 선택에 필요한 hash 누락·불일치 거부. hash가 source/검사를 대체하지 않음 |
| 중복 requirement | 정규화 identity로 합쳐 조건 교집합 검사. 충돌은 전체 실패, 마지막 줄 우선 처리 금지 |
| 빈 입력/marker로 모두 제외 | 명시적 no-op 결과, target/control 무변경. 기존 패키지를 삭제하거나 빈 graph 검사를 실행했다고 표시하지 않음 |
| 입력 변경·파일 교체 | 전체 입력을 한 번 freeze, 원본 identity/digest를 commit 전에 재확인. 바뀌면 재선택 없이 실패 |
| 임의 index/URL/VCS/editable/환경변수 확장 | 현재 source 계약에 없는 문법은 위치·이유와 함께 거부. `${SECRET}`를 Host env에서 읽지 않음 |

Parsing은 문법 경계/금지 입력/예산을 검증하고, version solving은 이미 고정된 native
resolver가 담당한다. 새 자체 dependency solver를 만들지 않는다. 기존 정확 버전 parser를
범위 parser처럼 재사용하거나 requirements를 줄별 shell 명령으로 바꾸지 않는다.

Project snapshot의 실제 roots와 graph를 사용하며 가짜 primary package를 만들어 단일
install API를 속이지 않는다. 기존 `ProjectDependencySnapshot`/`ProjectControlFile`,
whole-set inspection을 재사용한다. additive 설치 의도는 adapter와 Python transaction에
명시적으로 전달한다. `pypiVenvTransaction.prepareSelection/commitSelection`의 기존
complete 여부를 사용해 **unrelated owned package를 제거하지 않는다**. `uv pip sync`의
complete replacement 경로로 우회하지 않는다. 원래 대상에서 제공받는 dependency를
재사용할 때도 ownership/bytes를 인증하고 selected set의 검사에서 빠뜨리지 않는다.

`--upgrade`/`-U`는 요청 roots와 그 native 의존성 선택 의미를 지킨다. 업데이트 과정에서도
기존 설치 내용·entrypoint·records·ownership과 파일을 하나의 transaction으로 반영한다.
충돌/검사 실패/중간 설치 실패는 전체 무변경 또는 검증된 rollback이어야 한다.

완료 시험: 두 CLI 각각 기본 2 roots+공통 전이 1개, 범위/extra/marker, include/constraint/hash,
빈 입력, 충돌/escape/cycle/파일 변경, 전이 1개 BLOCK, 설치 중 실패, 기존 무관 패키지 보존,
업데이트/no-op. parser 시험과 실제 pinned native resolution·실제 promotion 시험을 구분한다.
현재 default/PyTorch CPU·CUDA source 제한과 예산은 유지한다. 공통 Python 선택/반영이
변경되므로 네 profile 영향과 필요한 최종 실제 회귀를 빠뜨리지 않는다.

### 4.3 B02 — npm 기본 install과 ci

수정 owner는 `cli/root.go`, `artifact/npm/project_controls.go`/`resolver.go`, 기존
`bootstrap/bootstrap.go` npm 연결, `promotion/npm_transaction.go` 및 npm update preparation이다.
명시 package install은 유지하고 인자 없는 요청을 기존 전체 project 선택으로 연결한다.

- `install`: 현재 package.json의 전체 dependency/peer/optional 의미를 고정 npm으로 해석.
  lock 없음/유효/manifest 변화 상태를 구분하고 필요한 선택·lock 갱신을 freeze한다.
- `ci`: lock 필수, manifest/lock 불일치 거부, lock bytes 불변. target 교체는 HAA ownership
  guard와 rollback 아래 수행하며 기존 native처럼 먼저 무조건 삭제하지 않는다.
- project 자체 lifecycle도 정적/동적 검사·source freeze의 대상이다. dependencies만 검사하고
  project postinstall을 Host에서 뒤늦게 실행해서는 안 된다. private/workspace/임의 source는
  지원 계약대로 명시 거부하며 PNPM/Bun으로 몰래 대신 실행하지 않는다.
- 완료: 두 intent의 clean/unchanged/changed manifest, missing/malformed lock, lifecycle 차단,
  peer·optional 선택, source 변경, ci lock 무변경, 전체 실패 시 기존 target 복원. 기존 단일
  install/update 소비자도 영향을 받은 범위에서 회귀를 확인한다.

### 4.4 B03 — Cargo fetch-only

수정 owner는 `cli/root.go` Cargo 등록, `bootstrap/cargo.go`,
`application/cargo_build.go: resolveProjectSnapshot` 호출 경로와
`application/project_snapshot.go`, `promotion/cargo_cache.go`다.
기존 resolve→전체 검사→검증 cache commit까지 재사용하고 관찰 build는 호출하지 않는다.
새 fetch가 나왔다고 CargoBuild의 준비·검사를 복사하거나 생략하지 않는다.

Cargo.toml만 있는 최초 선택/lock 생성과 기존 Cargo.lock 선택 보존을 구분한다.
root·target/platform과 필요한 optional/transitive 집합은 고정 Cargo의 공식 의미를 따른다.
fetch 결과는 **검사된 의존성 준비**이며 build.rs/proc-macro/application 동작 관찰 완료가 아니다.
실제 build에서는 기존 관찰 경로와 새 S01–S03 적용 범위를 계속 요구한다.
완료: cache cold/warm·최초/기존 lock, 변조/전이 거부/중단 정리, build invocation 0,
후속 기존 Cargo build의 동일 승인 cache 소비와 재검증 guard 유지.

### 4.5 B04 — Terraform 기본 init

수정 owner는 `cli/terraform_upgrade.go: validateTerraformInitArgs`와 `cli/root.go`,
`bootstrap/terraform.go`, `artifact/terraformprovider` resolver와
`promotion/terraform_transaction.go`/`terraform_cache.go`다.
명시 provider init·명시 upgrade는 유지하고 no-argument init을 별도 intent로 선택한다.

현재 configuration을 freeze하여 전체 provider를 선택하되 기존 lock이 있으면 해당 선택을
보존한다. constraint와 충돌하면 오류이며 자동 upgrade하지 않는다. lock이 없으면 최초
선택을 만들고 현행 signature/archive/dynamic 검사 후 하나의 transaction으로 반영한다.
backend 접속·cloud credentials·remote module 실행은 이 provider-only 지원에 포함되지 않는다.
지원하지 않는 configuration은 사전 검증에서 명시 거부한다.
완료: 최초 init/lock 유지/no-op/constraint 불일치/서명 실패/변경 controls/원복,
기존 명시 provider·upgrade 회귀, backend/cloud 외부 요청 0.

### 4.6 B06 — UV 선택 옵션과 frozen lock

`cli/uv_project.go`, `artifact/pypi/uv_project_input.go`/`uv_project_lock.go`,
`bootstrap/uv_project.go`, `cli/pnpm_project.go`/`bun_project.go`와 해당 adapters를 수정한다.
기존 parser가 거부하는 필드를 무시하도록 풀지 않고 지원할 필드와 의미를 명시해 읽는다.

UV는 dependency group/extra·marker, `--group`/`--only-group`/`--no-group`,
`--extra`/`--all-extras`, `--locked`/`--frozen`을 고정 UV 의미와 golden으로 대조한다.
상충 조합·존재하지 않는 선택·기존 기본 dev 선택을 명시한다. locked는 최신 lock 적합성
검증 후 무변경, frozen은 기존 lock을 재해결하지 않는 의미를 보존하되 exact graph/source와
HAA 보안 검사는 생략하지 않는다. native frozen을 locked로 조용히 바꾸지 않는다.
PNPM/Bun의 `--frozen-lockfile`은 실제 선택 고정·없는 lock/불일치 거부에 연결한다.
기존 PNPM lock 일치 검증은 보존한다. workspace·프로젝트 자체 build 확대는 별도다.

완료: default/선택/제외 group·extra와 marker의 전체 closure, lock bytes 무변경,
stale/missing/malformed lock, 해제되지 않은 flag 충돌, native 결과와 HAA 결과 비교,
동일 input의 기존 sync/add/install/update 회귀. 지원 flag 목록은 CLI 계약에 한 번만 관리한다.

### 4.7 B07 — Go mod tidy

`cli/root.go`, `bootstrap/go_module.go`, `artifact/gomodule` resolver,
`application/project_update.go`와 `promotion/go_transaction.go`의 기존 guarded controls
반영 경로를 사용한다. `go mod download`를 이름만 바꿔 연결하지 않는다.

고정 Go가 tidy에 사용하는 module source·test·build constraint 입력을 허용 project root에서
freeze한 뒤 격리된 resolver가 go.mod/go.sum 후보를 생성한다. Host toolchain 자동 다운로드,
프로젝트 실행, 임의 local replace/VCS/credential 접근은 허용하지 않는다. 선택 graph 전체를
검사하고 source/원본 controls 재확인 뒤 두 control 파일만 원자 반영한다.
완료: import 추가/제거·test dependency·build constraint·no-op·unsupported replace·source
교체·충돌·rollback. 실제 프로그램 실행 관찰과 tidy 완료를 구분한다.

## 5. 검사 완전성·모의 환경·행동 관찰

### 5.1 S09 — 검사 목록을 결과에서 역으로 추측하지 않는다

수정 owner: `core/domain/check.go`, 기존 Inspection/Verification report 타입,
`application/inspect.go`/`install_inspect.go`/project inspection,
`policy/m3.go`, 기존 ecosystem composite와 `bootstrap` 연결, `cli/result.go`/`install_result.go`.
Core에 생태계 이름별 if문과 runtime 실행 코드를 넣지 않는다.

1. bootstrap/adapter가 operation·subject kind·고정 profile에 맞는 expected check 집합을
   실행 전에 만들고 Application이 immutable하게 보유한다. Artifact의 출력으로 required를
   정하거나 inspector가 반환한 목록을 expected로 복사하지 않는다.
2. key는 run/subject digest/check capability/experiment identity를 구분한다. node별·whole-set
   검사를 명시하며 누락·중복·추가/다른 subject·다른 profile·잘못된 Evidence 결합을 거부한다.
3. 기존 `CheckExecution`의 capability/status/limitation과 현재 required fail-closed를 사용한다.
   새 generic workflow engine 대신 기존 report validation/Policy 직전 검증을 보강한다.
4. 준비 실패·timeout·scanner/DB 오류·observer 불완전은 미수행/불완전이다. findings 0 또는
   process exit0으로 completed를 만들지 않는다. 필수 하나라도 불완전이면 자동 ALLOW 없음.
5. 고정된 operation 적용 조건에 따른 NOT_APPLICABLE과 미지원 REQUIRED를 구분한다.
   불가능한 검사를 N/A로 돌려 제품의 원래 목표를 닫지 않는다. 개발 중 아직 등록하지 않은
   목표 검사도 사용자 지원표에는 미구현으로 남기며 전체 목표 완료라고 표시하지 않는다.

| 요청 종류 | 반드시 보이는 검사 범위/한계 |
| --- | --- |
| npm/PNPM/Bun 설치·실행 | whole graph, 공급 증명 상태, 정적/취약점, 해당 lifecycle/entry 실험과 더미·상태 diff |
| Python 설치·sync·실행 | 위 공통 검사와 선택된 import/hook/entry/build 실험, 대상 Python/profile·환경 자극 |
| Go/Cargo 의존성 준비 | whole graph·정적/취약점/공급 증명. build/runtime 동작은 미실행으로 표시 |
| Go/Cargo build | 위 준비 검사와 기존 관찰 build의 더미·상태 diff; 임의 application 실행은 포함하지 않음 |
| Terraform provider/GitHub ELF | format/공급 증명/적용 가능한 내용·취약점 검사, 제한 probe의 환경/상태 실험; RPC/임의 인자는 미실행 |
| update | 해당 위 검사 + S06의 인증된 이전/새 inventory 변화 |

결과는 `필수/실행/미실행`, 원인, 실제 실험 한계, policy, installation, cleanup을 구분한다.
기존 identifier 64 bytes/Evidence summary 1,024 bytes 등 Domain 한도도 유지한다.
여러 검사 결과를 거대한 summary 문자열에 넣지 않고 기존 check별 bounded record를 사용한다.
현재 `check.go`의 `VerificationReport`는 단일 execution/outcome이고 InspectionReport는
composite를 지원한다. 따라서 S07을 integrity 결과 하나에 덮어쓰지 않는다. 해당 작업에서
기존 Verification report에 독립된 check/outcome을 보존하는 collection을 최소 확장하고,
`PolicyInput`·Application의 evidence/check 수집·result consumer를 함께 연결한다. 기존
단일 verifier는 한 요소로 같은 경로를 사용한다. 별도 Verification framework는 만들지 않는다.
미완료 검사 때문에 이미 확정된 다른 check의 위반 Evidence를 잃지 않도록 각각 보존한다.
Policy는 첫 finding의 배열 순서로 최종 판정을 정하지 않고 BLOCK > MANUAL_REVIEW > ALLOW
우선순위를 전체 결과에 적용한다. 알려진 코드·required 상태와 바뀐 정책 version을 한 곳에서
관리하고 unknown finding은 현재 fail-closed를 유지한다.
필수 coverage profile을 새 기본값으로 활성화할 때 기존 정상 소비자와 미지원 처리·문구를
동시에 검사한다. 이전 evidence schema는 명시 migration 없이 새 completed 증거로 읽지 않는다.
완료: 아예 빠진 required check·중복·외부 subject·준비 실패·부분 응답을 모두 차단하고 정상
expected 집합만 허용한다. 사람이 읽는 화면과 machine result가 같은 사실을 나타낸다.

### 5.2 S01 — 더미 환경을 실제 실험에 배치한다

수정 owner: `sandbox/authority.go`, `boundary_helper.go`, `backend.go`,
`pypi_dynamic.go`, `github_elf.go`, project build backend 및 기존 typed observation plan.
분류는 `tools/gvisor-observer/observer.cc`와 현재 pod-init 계약을 따른다.
기존 권한 강하 helper를 더미 서비스의 만능 관리자나 새 완료 권위로 만들지 않는다.

환경 profile은 한 곳에서 버전 관리하는 typed 데이터다. 각 client가 같은 `.env` 생성
코드를 복사하지 않는다. 최소 환경과 모의 개발환경을 **서로 독립된 fresh 실험**으로 준비한다.
모의 profile은 run 전용 HOME/workspace, 내용 없는 가짜 project 구조, 무효한 credential
형태의 dummy env/.env 및 대표 인증 설정 파일을 가진다. 실제 Host 파일을 템플릿으로 읽지
않고 고정 synthetic fixture와 run 전용 무효 marker만 사용한다. 실제 인증될 수 있는 값·
사용자 이름·Host 경로는 넣지 않는다. Artifact가 fixture 경로나 응답 script를 선택할 수 없다.
더미 env로 loader/PATH/proxy/CA 설정을 덮어써 고정 실행 경계를 바꾸지 않는다. 해당 설정이
필요한 실험은 typed profile의 명시된 격리 입력으로만 취급한다.

배치는 권한·파일 형식·크기·내용 hash를 controller가 확인한 후 artifact를 시작한다.
Artifact에 fixture 생성/검사완료 기록 권한을 주지 않는다. 자극 profile 준비 실패는 해당
required experiment 미완료다. 기존 exact honeytoken 두 경로 분류를 일반 더미 파일이
자동 탐지된다고 간주하지 말고, 실제 guest 경로/범주와 controller 소유 배치 정보를 연결한다.

Env 값의 메모리 내 읽기는 syscall 관찰로 직접 증명하지 못한다. 증거에는 파일 열기,
synthetic token의 전송 시도처럼 실제 관찰 가능한 결과만 기록한다. 모든 getenv를 잡는다고
표현하거나 Host credential을 넣어서 탐지를 증명하지 않는다.
완료: 더미 부재 때 잠복하는 무해 표본이 배치된 실험에서만 활성화, 준비 오류 거부,
Host env/파일 미노출, 원래 정상 workload 동작, run 간 더미/오염 미공유, 종료 후 payload 부재.

### 5.3 S08 — 전후 파일 변화와 실행 중 관찰을 결합한다

기존 archive/payload inventory·hash·path guard를 재사용해 관찰 workspace와 dummy HOME의
실행 전후 snapshot을 만든다. `sandbox`가 관찰 사실을 수집하고 `inspection`이 Finding으로
정규화하며 `evidence`가 보존한다. mount topology 확인과 내용 diff는 구분한다.

- baseline 완료 → 전체 artifact process tree 종료/observer drain → final snapshot 순서다.
  created/deleted/modified, file kind·mode·size·content digest 변화를 비교한다.
- writable roots만 한 번 순회하고 device/proc/실제 Host를 재귀 스캔하지 않는다. symlink를
  따라가지 않으며 특수 파일·파일 수/bytes/time 초과·snapshot 불완전은 incomplete다.
- 두 snapshot 사이에 생성 후 삭제된 파일은 diff만으로 검출되지 않는다. 기존 syscall
  observation과 결합해 명시한 create/write/rename/unlink/permission-change coverage를
  채운다. 현재 observer에 없는 이벤트가 필요하면 동일 M11 helper/schema owner에서
  actual 지원을 검증해 추가하며, 단순 open count를 write 완료로 바꾸지 않는다.
- raw path/env/argv/network payload를 보존하지 않는 기존 privacy를 유지한다. 상세 위치가
  필요한 경우 Artifact 내부 상대 경로를 검증·escape·길이 제한하고, 동적 사용자 경로는
  범주/불투명 식별자·변화 수·digest로 기록한다. classifier 입력은 trusted 경계에서 transient 처리.
- 허용된 설치 파일 생성과 credential 접근·persist/권한 상승 시도를 구분한다. 정상 build가
  파일을 썼다는 이유만으로 모두 악성으로 판정하는 규칙을 만들지 않는다.

완료: 생성/수정/삭제/mode 변화·create-then-delete·동시 writer·큰 tree·link escape·observer
누락 반례, 정상 설치 대조, 종료 전 snapshot 오판 방지, raw 비보존 및 증거 digest 연결.

### 5.4 S02 — 실제 외부 연결 없이 제한된 모의 응답을 제공한다

기본 설계는 **네트워크가 없는 guest의 내부 loopback 서비스와 그 안의 DNS/HTTP responder**다.
실제 resolver registry 서비스나 Host에서 듣는 임의 proxy를 재사용해 인터넷을 열지 않는다.
고정 runtime이 이 동작과 관찰을 함께 지원하는지는 구현 첫 작은 실제 시험에서 확인한다.
불가능하면 원인·대안·영향을 먼저 설계에 반영하며 실제 egress 허용으로 우회하지 않는다.

응답 service는 신뢰된 고정 공급물로 run 내 별도 역할/권한을 갖는다. Artifact는 code/config를
변경하지 못한다. 완료 권위는 계속 외부 controller/observer다. trusted responder의 실행과
artifact process를 role로 구분하고 프로세스 이름으로 whitelist하지 않는다.

DNS는 허용 scenario 이름을 guest loopback으로만 해석하고 upstream forwarding을 하지 않는다.
HTTP는 고정 작은 status/body/state-machine만 제공하며 몇 단계 응답·요청 수·bytes·시간을
제한한다. 외부 URL redirect·서버 명령 실행·C2 code 전달은 없다. 임의 literal IP·외부 DNS·
IPv6·metadata/내부망·Host gateway로의 시도는 기존 격리로 차단하고 관찰한다.
모든 목적지 연결 성공이나 임의 HTTPS interception을 지원한다고 주장하지 않는다.
TLS가 반드시 필요한 표본은 별도 synthetic endpoint/시험용 trust만으로 한정하고 실제
검증을 끄지 않는다. 대상의 인증서 pinning으로 비활성화되면 관찰 한계로 기록한다.

응답을 받아야 후속 행동이 나오는 **별도 자극 실험**에서는 최초 네트워크 시도 이후에도
기존 시간/자원 한도 내 관찰을 이어갈 수 있다. 그 시도를 Policy에서 숨기거나 정상 egress로
바꾸지 않는다. 실제 탈취 목적지가 아닌 dummy sink에서의 시도/marker 일치 여부만 정규화하고
request payload·credential 값은 저장하지 않는다. 정상 네트워크 사용 자체를 악성 확정으로
표현하지 않으며 현재 network MANUAL_REVIEW 등 보안 판정은 별도 계약대로 유지한다.

완료: DNS→HTTP→dummy 읽기→전송 시도 chain, responder 미준비/timeout/변조·큰 응답,
Artifact의 responder 위장/완료 위조, 실제 외부/Host/내부망 송신 0, observer 완전성,
기존 network-none 실험 정상 유지와 두 process 역할의 확실한 정리.

### 5.5 S03 — 원래 위협 목적을 닫을 acceptance corpus

실제 악성코드·유효 secret을 받지 않고 기존 hostile fixture에 작은 무해 표본을 추가한다.
탐지는 제품의 실제 경로로 수행한다. 시험 표본의 success 문자열은 완료 증거가 아니다.

| 조건 표본 | 조건 없는 대조 | 자극된 실험에서 요구할 관찰 | 최종 조건 |
| --- | --- | --- | --- |
| dummy env/.env/key 파일 존재 조건 | payload 비활성 | dummy 접근 및 정해진 무해 후속 행동 | Finding/Policy와 반입 없음 |
| DNS/HTTP 응답 조건 | 실패 응답에서 비활성 | 제한 응답 뒤 동일 행동 | 외부 송신 없이 차단 |
| 짧은 지연/자식 프로세스 | 제한 내 실제 시간 경과 | 부모 종료 뒤 자식 행동 포함 | descendant 완료·관찰·cleanup |
| 복합 조건·탐지 회피 형태 | 한 조건 누락 시 비활성 | 준비된 조건 조합에서 행동 | 정확한 experiment별 증거 |
| 관찰 한도 밖 지연·지원 불가 조건 | 한도 초과/미도달 | incomplete/limitation | 자동 ALLOW 및 안전 보증 없음 |
| 정상 라이브러리·정상 build | 정상 실행 | 허용 filesystem/env 사용 | 오탐으로 일괄 차단하지 않음 |

실제 sleep을 쓰는 작은 표본과 fake-time 단위 시험을 구분한다. 무한 대기·clock 전역 조작·
조합 전수 탐색은 하지 않는다. 지원 profile에 명시한 유한 scenario 목록만 수행한다.
각 experiment의 limit는 기존 root resource budget 안에 포함하고 총량을 별도로 계산한다.
총 budget이 부족하면 profile 미완료로 보고한다. 검사 수가 늘었다고 승인 없이 기존 한도를
곱절로 늘리거나 일부 실험을 성공으로 건너뛰지 않는다.

Npm lifecycle, Python hook/import, Cargo/Go 관찰 build, 제한 ELF/provider probe에 실제
연결되는 fixture를 명시한다. 한 생태계의 성공을 나머지의 연결 증거로 쓰지 않는다.
최종 증거는 input/profile → 조건 배치 → 외부 관찰 → Finding → Policy → 원본 target
무변경 → 모든 owned resource 정리를 동일 run/subject로 연결해야 한다.

## 6. 정적 검사·취약점·변화·공급자 검증

### 6.1 S04 — 구조 검사 뒤 실제 내용 분석을 연결한다

기존 `inspection/npm/static.go`/`archive_inventory.go`, `inspection/pypi/static.go`,
`inspection/gomodule/static.go`, `inspection/cargo`와 `inspection/githubrelease/static.go`가
소유한 안전한 archive/member inventory를 재사용한다. 별도 경로로 다시 무제한 압축 해제하지
않는다. 생태계 composite에 내용 검사 결과를 합치고 기존 구조 검사 결과를 보존한다.

설계상 source analyzer 후보는 **고정 Semgrep CE의 offline 실행 + 저장소가 관리하는 작은
규칙 묶음**이다. 여러 언어의 새 AST/taint engine을 자체 개발하거나 자동 원격 rule download를
하지 않는다. 이는 이번에 dependency를 채택·설치했다는 뜻이 아니다. 구현 작업의 첫 단계에서
exact version/artifact digest/라이선스·재배포 조건/지원 문법/실제 runtime 호환성을 기존
trusted-tool 공급 절차로 검토하고 고정한다. 그 조건을 충족하지 못하면 tool adoption은
BLOCKED이며 다른 engine으로 조용히 바꾸지 않고 비교 근거와 변경 설계를 남긴다.

[Semgrep CE 공식 지원표](https://docs.semgrep.dev/semgrep-ce-languages)의 JS/TS/Python/Go/Rust/HCL
지원은 community 수준이며 분석 한계가 있다. 제품이 실제 선택한 언어·rule·parsing 결과로
coverage를 표시한다. 상용 interfile 분석을 CE 기능이라고 쓰지 않는다.

- trusted rule pack만 사용한다. Artifact 안의 `.semgrep*`, ignore/config, plugin, 자동 수정,
  metrics/upload를 실행 입력으로 채택하지 않는다. target은 읽기 전용·network 없음·출력 제한.
- 초기 rule 범주는 credential 파일/환경 탐색→전송, install-time 외부 명령/다운로드·실행,
  의심스러운 decode→dynamic execution, startup/persistence 변경이다. 단순 `getenv`·`eval`
  문자열 하나를 악성 확정으로 보지 않는다. rule ID/version/severity·bounded 위치·근거를
  기존 Finding/Evidence에 넣고 확정 불가능한 패턴은 MANUAL_REVIEW로 분류한다.
- exact 악성 signature와 의심 패턴을 구분한다. 정상 credential client·generated code·
  minified JS 대조군을 둔다. 언어 미지원/parse error/skip/timeout/truncated report는
  검사 완료 0건이 아니다. rule별 fixture 없는 새 규칙은 기본 Policy에 넣지 않는다.
- compiled ELF/native extension은 source AST 검사의 적용 대상이 아니다. 기존 binary format
  inventory에 bounded byte-pattern/위험 section·실행 권한 facts를 연결하는 별도 capability로
  표시한다. 자체 disassembler/범용 decompiler를 만들지 않는다. 분석하지 못하는 packed
  content와 지원하지 않는 형식은 limitation/required 상태로 전달한다.
- scanner exit code와 검사 finding 유무를 분리하고 subject digest·scanner/rule digest·
  실제 scanned/skipped inventory를 S09에 전달한다. scanner 공급물 자체도 untrusted
  Artifact와 분리된 고정 도구여야 한다.

완료: 각 지원 언어에서 정상/의심 fixture의 실제 scanner 결과, skip/parse/timeout/도구
변조·rule 변조·출력 위조/주입 거부, transitive artifact 연결, source/binary coverage 차이,
현재 static 검사를 보존한 Policy와 promotion 차단. 도구 설치 성공은 이 acceptance가 아니다.

### 6.2 S05 — 선택된 사용자 graph에 알려진 취약점을 대조한다

`application/install_inspect.go`와 project inspection의 **고정 graph 한 번**을 입력으로
기존 Inspection report에 결과를 넣는다. advisory I/O는 `internal/inspection` 아래의 좁은
adapter가 담당하고 Core/Policy에서 HTTP를 호출하지 않는다. 기존 `scripts/check/security.go`
저장소 검사를 제품 실행에 복사하거나 사용자 dependency 검사로 재명명하지 않는다.

기본 조회 설계는 OSV의 공개 batch API다. [공식 querybatch](https://google.github.io/osv.dev/post-v1-querybatch/)
응답은 ID/modified 중심이므로 세부 affected range/severity/withdrawn 판정에 필요한 record를
추가 취득하고 pagination을 끝까지 확인한다. source URL을 Artifact가 지정하지 못하게 한다.
typed network-policy에서 **읽기 전용 advisory 목적지**만 허용하고 실행 guest에는 그 권한을
주지 않는다. registry acquisition과 advisory 요청은 별도 목적/응답 schema로 검증한다.

1. ecosystem+정규화 package name+실제 version/source를 전체 node에 결합한다. npm 계열은
   npm, Python 계열은 PyPI, Go module은 Go, crates는 crates.io mapping을 검증한다.
   identity를 모르는 Terraform provider/GitHub asset을 이름 유사성으로 OSV package에 매칭하지
   않는다. 검증된 mapping이 없으면 unsupported로 표시하고 필수 profile은 자동 허용하지 않는다.
2. graph 내 동일 key는 한 번만 batch 조회하되 결과를 모든 해당 node로 환원한다. page/ID
   중복·missing result·wrong order/identity·무한 pagination·큰 record·TLS/HTTP 오류는 incomplete다.
3. live 응답의 provider·취득 시각·response hash·record modified를 기록한다. 이를 존재하지 않는
   전역 DB snapshot version이라고 부르지 않는다. frozen fixture 재생과 live 조회는 구분한다.
4. 기본 응답 cache freshness는 24시간, 크기는 §7의 유한 cache budget에 포함한다. 해당
   시간 내 인증된 complete 응답만 재사용하고 만료/조회 실패를 '취약점 없음'으로 처리하지 않는다.
   강제 refresh는 기존 지원 설정 경계에서 제공하며 무한 offline 예외는 없다.
5. 제안 기본 Policy는 확인된 affected HIGH/CRITICAL 또는 명시 악성 advisory는 BLOCK,
   나머지 affected/평가 불명은 MANUAL_REVIEW다. 취약점 ID·영향 version·수정 version/URL을
   bounded 정규화해 표시한다. severity 없음·withdrawn·alias 중복을 정상적으로 처리하며
   severity string 하나만으로 안전을 결정하지 않는다. 정책 변경은 기존 Policy leaf와 함께 검증한다.

완료: 직접/전이 vulnerability, 정상/수정 버전, alias/withdrawn, stale/missing/page 누락·
provider 장애·mapping 미지원, response tamper, 정상·차단의 실제 CLI 결과와 target 무변경.
서로 다른 DB 시점의 결과를 deterministic 회귀라고 비교하지 않는다.

### 6.3 S06 — 업데이트 위험 변화

`core/domain/project_update.go`의 frozen originals/selected, `application/install_update.go`,
`project_update.go` 및 각 promotion의 이전 ownership/receipt 검증을 재사용한다.
새 버전 inspection의 archive inventory를 공용 비교 입력으로 사용한다. 기존 update guard를
보안 diff service로 바꾸거나 파일 tree 전체를 다른 경로로 다시 다운로드하지 않는다.

비교 key는 source/name/variant와 검증된 artifact digest다. 이전 bytes 또는 당시 인증된
security inventory를 확인한 경우만 비교한다. 사용자가 변경할 수 있는 설치 directory나
기존 version string만으로 baseline을 신뢰하지 않는다. 이전 정리로 자료가 없다면 인증된
공급처에서 exact digest를 재취득하거나 baseline unavailable로 표시한다. 다른 최신 버전을
대신 이전 버전으로 삼지 않는다. 최초 install은 version diff N/A이며 신규 artifact 검사는 필요하다.

비교 항목은 dependency/source 추가·제거·교체, lifecycle/entrypoint·실행 파일 추가,
executable/set-id 등 mode 변화, static rule findings·advisory·공급 증명 상태의 변화다.
모든 텍스트 line diff나 버전 차이를 악성으로 판정하지 않는다. 새 script/출처 변경 등
위험 증가는 별도 Finding/MANUAL_REVIEW, 확정 위반은 BLOCK이며 unchanged라 해도 현재
필수 검사를 면제하지 않는다. 기존 설치를 유지할 때와 실제 새 선택 반영을 구분한다.

완료: 정상 변경/위험 script·dependency·mode 추가, 정상 서명된 위험 변경, 없는/변조된
baseline, 같은 version 다른 digest, source 변경, version rollback, change finding 후 무반입.
필요 baseline의 보존/만료 후 재취득은 C01 acceptance와 함께 확인한다.

### 6.4 S07 — 일반 package signature/provenance

`verification/npm/integrity.go`, `verification/pypi/integrity.go`,
`verification/githubrelease/integrity.go`와 기존 `bootstrap/bootstrap.go` 연결에 공급 증명
검사를 보강한다. 기존 Terraform signature와 `verification/releaseprovenance`의 고정 도구
검증은 유지한다. 고정 HAA/tool issuer allowlist를 임의 사용자 package에 적용하지 않는다.

| 공급 경로 | 구현할 discovery·검증 | 혼동하면 안 되는 사실 |
| --- | --- | --- |
| npm/PNPM/Bun | registry signature와 제공된 provenance를 metadata/실제 tarball identity에 결합 | registry signature는 registry 서명이며 publisher build provenance와 다름 |
| PyPI/pip/UV | Integrity API의 file별 provenance/attestation과 실제 wheel/sdist subject digest 검증 | Trusted Publisher identity와 사용자가 기대하는 source binding은 따로 확인 |
| GitHub release | 제공되는 asset attestation/signed manifest를 해당 repo·release·asset/digest에 결합 | asset API digest 또는 임의 `.sig` 파일 존재는 서명 성공이 아님 |
| Terraform/Go/Cargo | 기존 지원 verifier를 재사용하고 실제 제공 증명만 capability로 등록 | SumDB/checksum/index 무결성을 publisher signature로 바꾸지 않음 |

구현은 표준 verifier/library를 기존 고정 도구 공급 절차로 선택·pin하고 작은 adapter로
연결한다. 자체 암호 알고리즘·Sigstore/TUF 복제 구현은 하지 않는다. 사전에 받은 authenticated
metadata의 bounded bytes만 처리하고, attestation이 지시한 임의 URL/키를 trust anchor로
삼지 않는다. issuer·repository/workflow·subject digest·유효 기간/검증 시간·revocation·필요한
transparency proof를 선택한 표준의 정확한 의미로 확인한다. mismatch와 verification
unavailable을 구분한다. 최초 publisher binding이 불명확하면 guessed trust/자동 TOFU로
넘기지 않고 MANUAL_REVIEW에 필요한 정보를 표시한다.

결과는 verified / not published / unsupported / unavailable / invalid를 구분해 기존
Verification/Evidence 모델로 표현한다. invalid/mismatch는 BLOCK, required인데 증명이나
검증이 불충분하면 MANUAL_REVIEW다. absent를 verified로 치환하지 않는다. optional 허용
정책이 있다면 그 적용 범위를 명시하며 signed와 같은 보증 문구를 쓰지 않는다.

공식 의미 근거: [npm registry 서명](https://docs.npmjs.com/verifying-registry-signatures/),
[npm provenance](https://docs.npmjs.com/viewing-package-provenance/),
[PyPI Integrity API](https://docs.pypi.org/api/integrity/),
[PyPI attestation 검증](https://docs.pypi.org/attestations/consuming-attestations/).
문서상의 npm 검사 예시를 따라 Host에 먼저 설치하고 나중에 서명을 검사해서는 안 된다.

완료: valid/missing/wrong subject/wrong publisher·issuer/expired·revoked key/invalid proof·
network failure, 정상 서명되었지만 S04/S03에서 차단되는 표본, 현재 Terraform/고정 도구
검증 회귀. 한 registry 구현만으로 모든 공급자 증명 지원 완료라고 표시하지 않는다.

## 7. C01 — 제품의 자동 정리와 디스크 상한

### 7.1 수명별 삭제 기준

**검사 종료와 파일의 마지막 사용 종료는 다르다.** install은 검사 뒤 staging/promotion이
bytes를 사용하므로 그 전에 intake를 지우지 않는다. inspect-only는 결과/Evidence의 durable
기록과 관찰 종료 뒤 정리한다. 실패·취소·차단도 같은 owner가 정리한다.

| 자원 | 기존 owner | 정상 자동 삭제/보존 시점 |
| --- | --- | --- |
| download partial·intake·해제 사본 | `artifact/*`의 Acquire/intake | 마지막 staging/inspection 소비자 종료 뒤 삭제. 모든 실패 경로 포함 |
| dummy workspace·fake responder·sandbox volume/process | `sandbox`의 기존 session/Close | 외부 종료·observer drain 확인 뒤 owned runtime 자원/더미 삭제 |
| staging payload | `promotion/staging.go`와 project cache 구현 | commit/rollback 최종화 및 사용 lease 종료 뒤 transient 사본 삭제 |
| reusable verified cache | Go/Cargo/Python/UV/PNPM/Bun/Terraform cache owner | 유한 quota/TTL. 활성 참조를 보호하고 만료/회수 때 해당 entry와 파생 사본을 함께 정리 |
| Evidence/채택된 raw·SBOM·result | `evidence/local/store.go`와 기존 records | durable 기록 후 정해진 retention 동안 보존; 만료된 완전한 run 단위 삭제 |
| update rollback 자료·journal | 기존 transaction/backup owner | commit durable·원복 불필요 확정 후만 삭제. crash recovery 중에는 보호 |
| 사용자가 설치한 target·설치 관리 records | 기존 promotion/installation owner | 임시 사본이 아니므로 retention으로 삭제 금지. 정상 update/uninstall 경로로만 변경 |
| runtime image·고정 SDK/tool | 기존 runtime/release maintenance owner | 검사마다 삭제하지 않음. 별도 고정 버전 교체·참조 확인 절차 유지 |

현재 cache 중 후속 build가 직접 소비하는 경로는 단순 임시 파일이 아니다. 먼저 참조 관계와
재구성 가능성을 확인하고, evict된 경우 후속 명령은 새 검증/획득으로 복원하거나 명시 실패한다.
없어진 cache를 '승인됐던 것'이라며 검사를 건너뛰지 않는다. installed payload의 shared link/
hardlink·ownership record와 연결된 entry는 그 참조를 확인하기 전 삭제하지 않는다.

### 7.2 한 개의 수명 owner와 기존 삭제 경계

Application의 기존 operation owner가 시작/종료를 결정하고 bootstrap이 실제 store/policy를
주입한다. **새 CI daemon·별도 background 제품 service·언어별 GC를 만들지 않는다.**

기존 부품에는 cross-store 참조/예산 owner가 없으므로 현재 이 gap에 필요한 작은 공용 수명
조정이 허용된다. 우선 `application`의 기존 operation orchestration과 `core/ports`의 opaque
resource lifetime 계약을 확장하고, concrete 파일 I/O는 기존 artifact/promotion/evidence
owner에 둔다. 단순히 callback을 등록하는 것만으로 crash recovery가 된다고 보지 않는다.
공용 저장소 accounting 코드가 실제로 필요하면 기존 layout 책임에 맞는 **한 local adapter**로
모으며 새로운 범용 storage framework나 client별 cleanup class를 만들지 않는다.

구체 수정 후보는 다음으로 제한한다. 이것은 파일을 지금 생성하라는 지시가 아니다.

| 위치 | 변경 책임·기존 코드와의 분리 |
| --- | --- |
| `application/inspect.go`, `install_promote.go`, `project_snapshot.go`, execution use case | 기존 최상위 operation의 시작/마지막 사용/종료에 수명 호출 연결. node별 조기 삭제 금지 |
| `core/ports/storage.go` 신규 후보 | 현재 실제 소비자들이 공유하는 reservation/resource lease/finalization의 좁은 opaque 계약. 경로·Docker·Policy 재판단 없음 |
| `internal/storage/local.go` 신규 후보 | 현행 모듈에 없는 cross-store 원장·quota·reference/sweep 조정 한 곳. 파일 생성/검사/승격 알고리즘은 기존 owner에 유지 |
| 기존 `artifact/*`, `promotion/staging.go`·각 cache/transaction, `evidence/local/store.go` | 생성·durable 게시·사용 종료를 등록하고 해당 store의 안전한 삭제 경계 제공. 다른 store를 직접 삭제하지 않음 |
| `bootstrap/bootstrap.go`와 실제 client binding | 보호된 실제 root/policy/공용 store를 한 번 구성해 기존 소비자에 주입 |
| `cli`의 기존 config/result owner | storage 설정 조회·검증/명시 변경, cleanup partial failure·usage를 기존 표시 방식으로 전달 |

신규 local adapter의 이유는 §2에서 확인한 cross-store 수명 공백이다. 기존 구현을 대체하는
것이 아니며 Artifact별 collector/GC framework를 만들 근거도 아니다. 구현 착수 때 현재
코드에 동일 책임이 추가돼 있으면 그 owner를 사용하고 신규 package 후보를 제거한다.
새 package 채택 시 Directory leaf에 현재 책임·의존 방향만 반영한다.

원장 최소 schema는 version, operation/run, opaque resource ID/kind, trusted root의 상대
handle, owner/파일 identity, content identity, logical/reserved bytes, phase, lease owner와
generation, evidence/approval/journal reference, expiry/last-use, cleanup error다.
실제 secret·raw argv·Host 사용자 경로는 기록하지 않는다. 외부에서 넘어온 임의 경로를
이 schema의 handle로 승인하지 않으며 등록 전에 owner가 만든 위치와 대조한다.

구현 순서는 다음으로 고정한다.

1. 현재 Acquire/Stage/cache 생성·실패 정리/Close 지점을 inventory로 연결한다. operation과
   run ID, 생성 resource identity, 크기, 상태, 활성 소비자, retention 대상, journal 참조를
   controller 소유 원장에 원자 기록한다. Artifact가 원장/retention 설정을 쓰지 못한다.
2. 전체 operation의 성공·BLOCK·MANUAL_REVIEW·실패·취소에서 같은 finish 경로를 호출한다.
   `InspectService`의 dependency별 종료가 전체 install 전에 파일을 지우지 않게 한다.
   caller가 staged handle을 반환하는 경우 lease 이전 또는 마지막 소비 완료가 명시되어야 한다.
3. 기존 sandbox/transaction Close로 프로세스·FD·mount·commit/rollback을 먼저 정리한다.
   그 다음 사용 종료가 확인된 storage만 수거한다. canceled context 대신 기존 규칙에 맞는
   별도 bounded cleanup context를 사용하며 작업 deadline을 무제한 연장하지 않는다.
4. per-store 삭제는 root 아래 **생성 owner가 증명된 항목**만 처리한다. `Store.DeleteRun`의
   path check를 재사용하되 자동 호출에는 run 참조·expiry·active lock 검증이 추가로 필요하다.
   string prefix와 mtime만 보고 `RemoveAll`을 호출하는 GC는 만들지 않는다.
5. 종료 결과는 policy/installation/cleanup을 분리한다. install이 이미 commit됐는데 cleanup이
   실패하면 '설치됨·정리 실패'와 nonzero/정확한 recovery 정보를 남긴다. 과거 policy를 변경하거나
   설치 실패인 척 자동 재설치하지 않는다. 첫 실패와 secondary cleanup 원인을 함께 보존한다.

### 7.3 삭제 안전성과 crash 복구

삭제 전에 trusted root/ancestor·owner·실제 directory/file identity·symlink/reparse/hardlink·
mount/active reference를 검사한다. root-constrained/descriptor-relative 기존 I/O를 재사용해
TOCTOU를 막는다. 다른 명령과 동일 entry를 쓸 때 shared lease, 삭제할 때 exclusive ownership을
사용한다. 기존 project lock과 storage lock은 원장에 고정된 순서로 획득하고 긴 network 작업
동안 전역 lock을 잡지 않는다. lock 획득 실패는 사용 중이며 삭제 사유가 아니다.

`ALLOCATED → IN_USE → TERMINAL/RETAINED → DELETING → REMOVED`에 대응하는 최소
기록으로 중단 지점을 복구한다. 이 상태는 파일 수명이며 InspectionRun의 보안 상태를
대체하지 않는다. 다음 실행 시작에서 bounded sweep으로 이전 owned resource를 확인한다.
PID 부재·오래된 mtime만으로 stale을 확정하지 않고 process generation/lock·session/query
결과와 journal의 참조를 확인한다. query error/UNKNOWN이면 해당 삭제를 보류하고 정확한
blocker를 남긴다. 독립적으로 안전한 다른 항목은 정리한다.

SIGKILL/전원 차단 중에는 삭제할 수 없다. 다음 실행에서 복구하는 것이 설계 범위다.
CLI가 한 번도 다시 실행되지 않으면 즉시 수거를 보장하지 않는다. 남은 항목은 원장과 quota에
계속 계상하여 재시작 때 모른 채 새 사본을 누적하지 않는다. 알 수 없는 과거 디렉터리는
자동 소유권을 만들어 삭제하지 않고 inventory로 표시한다. 기존 사용자의 자료 이전은 별도
소유권 증거·용량 검사 후 수행하며 한 번에 전체 cache를 지우는 migration을 만들지 않는다.

### 7.4 유한 보존 정책과 공간 예약

아래는 **구현할 중앙 storage policy의 초기 설계값**이다. 아직 적용된 설정이 아니다.
값을 여러 client/helper에 하드코딩하지 않는다. 기존 config 경계에 typed storage 설정을
추가하고 모든 숫자는 단위·최소/최대·상호 관계를 검증한다. 정책 version/digest를 결과에
연결한다. 사용자 설정은 보존/전체 디스크 quota를 조정할 수 있지만 기존 artifact/runtime
보안 예산·권한·필수 검사를 확대하는 수단이 아니다.

| 항목 | 초기 설계값·규칙 |
| --- | --- |
| 종료된 transient payload | 보존 0; 마지막 소비 종료 직후 수거. 디버그용 무기한 유지 기본값 금지 |
| 재사용 payload cache | 총 16 GiB, 사용하지 않은 entry TTL 7일. 초과/만료 때 unreferenced부터 수거, 80% low-water까지 |
| 일반 완료 run evidence/raw 근거 | 30일 보존, 총 256 MiB 및 1,000 runs 한도. 보존 만료 전 자료를 공간 부족만으로 조용히 삭제하지 않음 |
| installed receipt/ownership·복구 원장 | 설치/복구 참조 동안 보호. 작업량에 맞는 별도 accounting, 예산 내 공간 없으면 새 변경 거부 |
| HAA 관리 작업 저장소 전체 | 기본 64 GiB, 현재 사용량+예약량 모두 포함. runtime 공급물/실제 사용자 설치 target은 별도 범주로 표시 |
| 파일시스템 여유 reserve | 기본 2 GiB. 실제 남은 공간과 기존 profile 임시 disk 필요량을 함께 검사 |
| advisory 응답 | freshness 24시간, cache 총량에 포함. 완전하고 source가 인증된 응답만 재사용 |
| sweep | 명령 시작의 bounded 복구 + 해당 operation 종료 정리. 매 package마다 전체 cache 재검색 금지 |

이 수치는 quota이며 미리 64 GiB를 할당한다는 뜻이 아니다. 기존 M12 CPU 임시 4 GiB,
CUDA 임시 24 GiB 같은 profile 최대값은 기존 resource policy에서 읽으며 여기서 대체하지
않는다. 실제 필요한 bytes가 retention quota보다 큰 일회성 artifact도 기존 profile/전체
quota/여유 공간 안에서 검사한 뒤 즉시 지울 수 있다. cache에 남기지 못했다고 검사 결과를
실패로 바꾸지 않되, 후속 사용에 필수인 승인 cache의 보관은 명령 완료 전에 보장해야 한다.

획득·압축 해제·staging·rollback에 동시에 필요한 최대 disk를 operation 단위로 예약한다.
같은 filesystem의 복수 명령은 예약 합계로 admission을 판단한다. 예상치만 믿지 않고 실제
쓰기 중에도 bytes/files 상한을 적용한다. 압축 폭탄·sparse/hardlink·동시 쓰기를 누락하지
않도록 logical 사용량과 filesystem available 측정을 구분한다. 하드 quota 적용이 가능한
기존 volume 경계는 유지한다. reservation 파일만 두고 실제 소비를 확인하지 않는 방식은 안 된다.

먼저 만료/회수 가능한 cache를 정리하고도 공간이 부족하면 새 다운로드/설치를 시작하지 않는다.
미만료 Evidence·active cache·rollback 때문에 정리할 수 없을 때는 정확한 사용량·보호 이유를
표시하고 fail-closed한다. quota 0/음수·overflow·unlimited 설정은 허용하지 않는다.
TTL/개수 제한만 있고 bytes 제한이 없거나, 실패 사본을 budget 밖에 누적하는 구현은 미완료다.

캐시 삭제 뒤에도 설치 관리에 필요한 작은 receipt와 S06 baseline inventory는 설치 records에
남긴다. 이 자료도 무제한 복제하지 않고 현재 설치와 필요한 rollback 세대만 소유한다.
일반 run history는 retention 만료 시 삭제하고, 오래된 evidence reference에는 명시적인
만료 상태를 반환한다. 만료된 evidence를 현재 새 ALLOW의 근거로 사용하지 않는다.
기존 approval이 Evidence를 직접 참조하는 cache는 그 Evidence도 pin하거나 cache와 함께
무효화해야 한다. payload만 지우고 영구 dangling approval을 남겨서는 안 된다.

### 7.5 자동 정리의 완료 시험

정상 install/update/inspect/no-op, BLOCK/MANUAL_REVIEW, 다운로드·검사·staging·promotion
실패, SIGINT, 강제 중단 뒤 재실행, disk full을 모두 포함한다. 다음 결과가 필요하다.

- terminal 뒤 owned transient 사본/volume/process는 0이며 작은 evidence/현재 설치 records와
  허용 cache만 남는다. 원래 사용자 파일·설치 target·다른 실행·runtime 공급물은 동일하다.
- 같은 입력 반복과 서로 다른 입력 반복을 각각 작은 fixture로 100회 수행해, 종료된 payload가
  run 수에 비례해 남지 않고 cache/evidence/total quota를 넘지 않음을 측정한다. 네트워크나
  multi-GiB corpus를 100회 받지 않는다. 큰 실제 profile은 필수 최종 실행의 정리 측정으로 확인한다.
- concurrent acquire/use/delete, shared reference, lock held, stale PID 재사용, metadata tamper,
  symlink/hardlink/path 교체, query failure, delete 중 crash에서도 잘못된 삭제/승인 없음.
- fake-clock TTL/rollover/expiry·retention full·quota reservation 경쟁과 실제 filesystem 정리
  시험을 구분한다. fake timer의 성공을 실제 디스크/프로세스 해제로 보고하지 않는다.
- 삭제 오류는 반복 가능한 recovery 정보를 남기고 다음 실행에서 idempotent하게 회수한다.
  deletion 실패가 무시되거나 quota 미계상으로 계속 설치되면 acceptance는 열어 둔다.

## 8. 단순성·성능·불용 코드 기준

### 8.1 비용 모델

`B`는 검사 대상 총 bytes, `V/E`는 dependency node/edge 수, `F`는 선택 파일 수,
`R`은 고정 활성 규칙 수, `C`는 현재 owned cache entry 수다. 입출력·hashing·공식 resolver·
외부 scanner 전체를 O(1)이라고 약속하지 않는다. 메모리는 bounded streaming과 필요한
identity index에 한정한다.

| 처리 | 목표·허용되는 비용 | 피할 구현 |
| --- | --- | --- |
| requirement/include 읽기 | O(B+include edges), visited map과 명시 예산 | 같은 include 반복 읽기·무제한 recursion |
| graph coverage/dedup | O(V+E), hash-key lookup 평균 O(1) | 모든 node 쌍 비교·package마다 whole graph 재검사 |
| archive/hash/state diff | O(B+F), 한 번 읽은 inventory 재사용 | 검사 단계별 동일 압축 해제·전체 파일을 메모리에 적재 |
| advisory 조회 | unique key별 batch, O(V+응답량) | node마다 같은 query·무한 retry/page |
| static scanner | 고정 R의 실제 시간/메모리 측정·bounded timeout | 임의 regex backtracking·전체 rule registry 자동 확대 |
| cleanup | 현재 자원 O(F), 선택된 cache sweep O(C) 상한 | 매 artifact마다 전체 홈/cache/Docker scan |
| 정렬된 결과·retention 후보 | 필요한 O(n log n)은 명시·한 번 수행 | O(n)을 주장하려고 의미/결정성을 제거 |

Dependency solving의 최악 복잡도는 native solver에 따라 선형이 아닐 수 있다. package 수,
결과 bytes·시간·재시도 한도로 제한하고 별도 solver를 구현하지 않는다. 원격 latency도 CPU
complexity와 구분한다. 다중 실행으로 속도를 숨기지 않고 기존 bounded concurrency/resource
budget을 지킨다. scan 결과 재사용 key에는 artifact digest뿐 아니라 tool/rule/profile/runtime/
Policy와 freshness를 포함한다. 이전 ALLOW cache를 새 실행의 무검사 권한으로 쓰지 않는다.

성능 acceptance는 바뀐 경로의 고정 small/medium fixture와 기존 큰 profile의 필수 actual에
연결한다. parsing/index/diff/cleanup fixture를 1×/2×/4×로 늘려 gross quadratic 증가가 없는지
확인한다. 평균·상한 시간, peak RSS, 읽고 쓴 bytes, peak/terminal disk를 기록한다. 공유 runner
한 번의 시간 차이로 인과를 단정하지 않는다. 새 검사의 정당한 비용은 단계별로 분리하며
기존 경로의 설명되지 않는 회귀는 분석·해소 전 완료 처리하지 않는다. 명시 budget 초과는
측정 평균이 빨라도 실패다. 이 수치를 얻기 위해 동일 full CI를 반복하지 않는다.

### 8.2 C03 — 변경에 따라 불용이 된 코드만 정리한다

각 구현 묶음의 canonical formatter/build/vet/pinned Staticcheck 결과를 먼저 본다.
그 다음 실제 바뀐 owner와 직접 caller를 `rg`/참조 검색으로 확인한다. OS build tag·generated
code·runtime entry·fixture는 단순 텍스트 검색 0건만으로 불용이라 판단하지 않는다.

제거 후보는 대체된 per-client wrapper/parser, 사용하지 않는 flag/상수/field,
중복 bootstrap binding, 더 이상 선택되지 않는 fallback, 임시 진단/시험 전용 production
우회와 새 기능으로 무효가 된 fixture다. 현재 source·call graph·compile/test 근거를 같이
남긴다. 사용되지 않는 정리 코드를 남겨 기능 구현이라고 주장하지 않는다.

관련 없는 파일 정리·일괄 rename·의존성 전체 업데이트·새 framework 도입은 금지한다.
함수 길이/파일 수/추상화 수를 줄이는 숫자 목표 때문에 책임 경계를 합치지 않는다.
새 일반화가 실제 두 소비자에 필요하지 않으면 기존 owner의 명시적인 작은 코드로 처리한다.
완료 시 before/after 책임 지도에서 하나의 동작을 수행하는 경로가 하나임을 확인한다.

## 9. 검증 묶음과 CI 20분 자동 확인

### 9.1 검증을 선택하는 기준

1. 입력/상태/Policy 변경은 작은 기존 package 시험으로 실제 실패→수정 성공을 먼저 확인한다.
   새로운 기능·보안 경계에 필요한 시험은 추가하되 구현 그대로를 재계산하는 자기확인 시험은
   만들지 않는다. 문서/작은 helper마다 full/native/CPP/CI를 시작하지 않는다.
2. 의미 있는 local 묶음에서 raw-free exact candidate로 canonical `scripts/check`의 해당
   profile을 실행한다. `quick`의 gofmt check/build/test-build/vet/pinned Staticcheck를
   임의 PATH 도구로 대체하지 않는다. `format` 외 profile은 read-only다.
3. 동작 수정이면 관련 정상/negative와 영향을 받는 Race, boundary/observer 수정이면 해당
   CPP/native actual을 연결한다. workflow 수정은 pinned actionlint/기존 workflow profile,
   도구 공급 변경은 기존 lock/supply 검사도 포함한다. 실제 profile 이름은 현재 checker를 따른다.
4. final 후보에는 필수 quick/security/docs/workflow 및 현재 계약이 요구하는 actual/hosted
   profile을 수행한다. 공통 Python/observer 변경이면 CPU/cu126/cu130/cu132의 실제 영향과
   필수 qualification을 포함한다. 대표 한 profile로 전체 성공을 대신하지 않는다.
5. source·입력 digest·runtime·Policy가 같고 이미 유효한 결과는 재사용한다. 수정된 경우
   영향받은 결과만 새로 얻는다. raw를 가진 원래 cwd에서 `./...`/docs/format을 실행하지 않는다.

| 변경 묶음 | 로컬로 먼저 모을 작업 | 원격 CI가 필요한 경계 |
| --- | --- | --- |
| B00–B07 | command/입력/고정 native 선택·원자 반영을 연속 검증 | 새 backend/native 의미가 후속 신뢰를 막거나 client end-to-end 후보가 완성된 때 |
| S09+C01 기본 | coverage·수명·quota/실패/동시성 | 기존 공용 완료/정리 경계의 실제 동작을 로컬에서 충분히 확인할 수 없고 후속 실험이 의존할 때 |
| S01–S03+S08 | 더미·diff·응답·조건 표본을 같은 경로에 완성 | 공용 observer/network/runtime boundary를 포함한 의미 있는 전체 후보 |
| S04–S07+C01 전체 | scanner/advisory/provenance/diff와 retention 연결 | 도구 공급·기존 소비자·최종 목표 matrix가 완성된 후보 |

이는 최대/최소 CI 횟수를 강제로 정하는 표가 아니다. 다음 로컬 구현이 원격 결과에 의존하지
않으면 계속 모은다. 보안 경계가 미검증인 채 dependent implementation을 신뢰해야 하는
경우에는 그 이유를 기록하고 앞당긴다. 성공한 동일 입력을 '한 번 더' 반복하지 않는다.
CI 진행 중 새 push로 현재 qualification을 불필요하게 취소하지 않는다.

### 9.2 실제 타이머 운영 계약

제품 기능과 작업자의 CI 감시는 구분한다. 현재 운영에 실제 public API 조회를 수행하는
20분 worker가 있다. 예를 들어 retained raw의
`docs/github-logs/package-client-quarantine-20261021/haa-ci436-autopoll.py`는 run/head를
검증하고 status/jobs를 기록하며 terminal까지 반복한다. 이 로컬 raw script를 새 제품 파일이나
CI 필수 입력으로 가져오지 않는다. **그 run의 오래된 설정을 재실행해서도 안 된다.**

새로 필요한 CI를 시작할 때 실제 실행 환경의 기존 감시 도구를 확인하고 다음으로 운영한다.

1. run ID/attempt/head/profile/시작 시각·최근 같은 profile의 소요 시간을 고정한다.
   최초 조회 deadline은 감시 시작부터 20분 이내이며 예상 완료가 1시간이어도 첫 확인을 미루지 않는다.
2. 해당 run의 worker가 이미 있으면 사용한다. 하나의 run에 중복 worker/예약을 만들지 않는다.
   실제 clock timer가 다음 시각에 status/conclusion/jobs를 조회하도록 연결한다.
3. queued/running이면 확인 시각과 다음 20분 deadline을 기록하고 같은 item의 독립 작업을
   계속한다. timer DUE 파일만 생성하는 것은 조회 완료가 아니다. API 오류는 UNKNOWN이며
   성공/실패/부재로 추정하지 않는다. 네트워크 요청 자체에도 bounded timeout을 둔다.
4. terminal이면 success/failure/cancelled/timed_out을 구분하고 actual source/checkout,
   Required·필수 jobs·실제 workload·첫 concrete failure를 확인한다. 완료한 log는 hash로
   재사용하고 매 tick 전체 log를 다시 다운로드하지 않는다.
5. 실패 원인·source·환경·입력을 먼저 대조한다. 동일 가설의 재실행을 반복하지 않는다.
   source 불변에서 외부 일시 실패가 확인된 경우에도 실패 job만 재검증할 수 있는지 판단하고
   성공한 GPU 등 필수 결과를 불필요하게 다시 실행하지 않는다.
6. worker가 결과를 기록하면 활성 응답에서 이를 소비하고 다음 작업/수정으로 이어간다.
   대기 도구는 한 번에 60초 이하로 사용해 소통·새 지시를 놓치지 않는다. worker만 남기고
   final로 끝내 '자동 작업 재개'가 된 것처럼 보고하지 않는다. 대화가 외부에서 종료되면
   분석/수정의 자동 재개를 보장할 수 없으며 다음 재개에서 기존 run/worker부터 대조한다.
7. run terminal 뒤 worker 종료·next timer 없음까지 확인한다. 감시 한도 초과/worker crash는
   CI 성공이 아니라 감시 실패로 기록한다. 새 CI 없이 timer만 계속 늘리지 않는다.

이번 문서 작업에는 CI가 없으므로 타이머를 만들지 않는다.

## 10. C02 — 시험 아티팩트 정리

시험 시작 때 source/runtime/input digest, 전용 root, 소유자, 생성 payload/cache/volume/
container·최대 예상 bytes를 기존 evidence inventory에 기록한다. 사용자 venv·프로젝트·
기존 SDK·공용 runtime image·진행 중 다른 시험을 삭제 대상으로 삼지 않는다.

시험 종료 후 작은 결과/원본 로그의 필요한 부분·hash·첫 실패·재현 입력 식별을 durable하게
보존하고, process/FD/mount/container/volume의 actual 부재·신원·참조를 확인한 뒤 명시적으로
목록화한 완료 payload/반복 build 사본을 삭제한다. 전체 Docker prune·홈 cache 삭제·광범위
`rm -rf`는 쓰지 않는다. active 재사용 cache는 묶음 검증 종료까지 한 벌만 유지한다.

실패 분석에 필요한 최소 frozen payload는 이유·owner·재검토 시점·bytes 상한을 붙여 유한하게
보존한다. 같은 archive를 여러 raw 폴더에 복제하지 않는다. 로그/hash만으로 재현이 불가능한
최초 실패의 유일한 bytes를 분석 전에 지우지 않는다. 분석 종료 후 그 payload도 삭제한다.
실제 credential·민감 파일은 시험 입력/보존 자료에 포함하지 않는다.

정리 receipt는 삭제 전후 logical/allocated bytes, 정확한 삭제 대상·남긴 대상/이유,
active reference 검사·query error, 실패와 재시도 결과를 기록한다. 성공한 delete 호출만으로
모든 kernel resource가 해제됐다고 표시하지 않는다. CI artifact가 새로 필요하면 기존 계약의
최대 14일 retention을 지키고 repository 전역 Actions 보존 설정을 임의 변경하지 않는다.

완료: 새 시험의 disposable payload 부재와 현재 필요한 cache/원본 보존이 확인되고 회수량이
기록되어야 한다. 단순 로그 한 줄 'cleaned' 또는 시험 process exit0으로 닫지 않는다.
이번 설계 작성에서는 사용자/시험 아티팩트를 실제 삭제하지 않는다.

## 11. 완료 판정과 증거 형식

각 B/S/C 행은 다음 네 상태를 **서로 독립적으로** 갱신한다.

| 상태 | YES 조건 |
| --- | --- |
| IMPLEMENTED | 해당 요구의 실제 코드가 기존 책임 owner에 존재하고 누락/임시 우회가 없음 |
| WIRED | 공개 CLI → 실제 bootstrap → 검증/판정/반입/정리 consumer가 연결됨 |
| QUALIFIED | exact candidate/입력에서 정상·거부·오류와 해당 실제 실행·필수 CI 증거가 있음 |
| ACCEPTANCE_CLOSED | 이 문서의 해당 완료 조건과 남은 회귀/정리/coverage 한계 기록까지 확인됨 |

증거의 최소 단위는 `gap ID → 요구 → file/symbol/caller → test/명령 → source/input/runtime/
policy → actual result → negative target 무변경 → cleanup → remaining limitation`이다.
작은 local 통과·compile·help exit0·fake backend·부분 profile을 actual/전체 qualification으로
승격하지 않는다. NOT_REACHED/SKIP/UNKNOWN은 별도 상태로 남긴다.

전체 보강 완료에는 다음이 모두 필요하다.

- B00–B07/S01–S09/C01–C03의 MISSING이 비어 있음. 승인된 범위 변경이 있다면 명시된
  이유·영향·남은 목표로 기록하며 미구현을 말만 바꿔 COMPLETE로 만들지 않음.
- requirements의 전체 root/전이 검사와 한 번의 반영, 다른 지원 client의 기본 파일 흐름,
  정상 install/update/no-op, 실패 시 원본 보존/rollback과 정확한 exit 의미.
- 잠복 조건 시험의 실제 활성화·관찰·차단·무반입, scanner/DB/provenance/diff의 실제
  consumer, required 누락/준비 실패/관찰 불완전의 자동 ALLOW 방지.
- 기존 정상 소비자와 네 Python profile 중 영향을 받은 필수 qualification 보존.
  지원하지 않는 형식/환경·동적 실행면의 한계는 결과와 지원표에 일치하게 표시.
- 유한 cache/evidence/전체 quota, success/failure/cancel/crash 뒤 자동 정리,
  concurrent use·unknown identity에서의 안전한 보류, 실제 사용자 데이터 무변경.
- 변경 경로의 불용/중복/임시 코드 제거, 비용·메모리·disk 예산 충족과 회수 receipt.
- 문서·지원표·Queue의 상태가 실제 결과와 같고 모든 필수 CI가 terminal로 확인됨.

위 조건 전에는 이름/배포 완료를 기능 완료의 대체로 삼지 않는다. 전체 목표의 한계와
이번 bounded acceptance를 설명하되 '오류/버그가 절대 없다'는 증명으로 표현하지 않는다.

## 12. 착수 전 확정할 입력과 멈춰야 할 경계

설계가 모호한 채 코드를 먼저 늘리지 않도록 다음 확인을 해당 작업 첫 단계에 포함한다.
이 확인들은 구현 작업의 일부이며 이번 문서에서 이미 검증됐다고 주장하지 않는다.

| 작업 | 정확히 확인할 입력 | 불충족 시 처리 |
| --- | --- | --- |
| B01/B05/B06 | 고정 pip/UV 공식 grammar·native golden, 기존 target/ownership 형태 | 문법 의미를 추측한 parser 수정 금지; 차이를 설계와 fixture로 확정 |
| S01/S02/S08 | 고정 gVisor loopback·관찰 event·responder 역할·종료 회계 | network/security boundary 완화 없이 작은 actual 근거로 설계 수정 |
| S04/S07 | 선택 도구/library exact version/hash·라이선스·공급/배포 가능성 | 미검증/floating latest 채택 금지, 해당 capability OPEN 유지 |
| S05 | provider schema/pagination·package mapping·typed network allowlist | 이름 추측/조용한 skip 금지, unsupported/incomplete로 표시 |
| C01 | 실제 store root/lease·approval 참조·filesystem 한도·기존 대용량 profile | 소유 불명 파일 삭제 금지, 필요한 이동/설정 변경의 영향부터 확정 |
| 모든 작업 | 현재 Git/Queue/필수 입력·최초 미해결 실패 | 과거 CI436 등을 현재 실패로 가정하지 말고 최신 상태와 원본 대조 |

이 설계에서는 새 package version이나 운영 환경 값을 추측으로 lock에 쓰지 않았다.
구현 지시를 받으면 01/B00부터 시작하되 이미 구현된 gap이 있는지 해당 owner만 다시 대조한다.
전체 저장소를 매번 재감사하거나 해결된 실패를 근거 없이 다시 열지 않는다.

이 문서 변경의 raw-free 후보 hash·문서/보안 검사 기록은
`docs/github-logs/goal-design-20261101/`에 보존한다. 이는 문서 후보 검사이며 위 보강 기능의
정상 동작·디스크 수거·새 runtime/CI qualification 증거가 아니다.
