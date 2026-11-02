# C01 수명·저장 예산 baseline

기준 `9e9260f`, M13-004의 다음 순차 단위다. 계약은
[보강 설계 §7](../../21-goal-completion-design.md#7-c01--제품의-자동-정리와-디스크-상한)이 소유한다.
현재 IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다.
기존 부분 구현을 대체하지 않고 아래 MISSING만 보강한다.

## 현재 owner와 재사용 경계

| 수명 | 기존 구현과 실제 연결 | 미완료 |
| --- | --- | --- |
| intake | `artifact/npm.Resolver.Acquire`, `pypi.Intake.Acquire`, `gomodule.Client.Acquire`, `cargo.Client.Acquire`, `terraformprovider.Client.Acquire`, `githubrelease.Client.Acquire`; run별 획득과 실패 사본 제거 | 성공 run의 마지막 소비·공통 등록·종료 수거·crash 복구 |
| 격리 실행 | `bootstrap`의 observer/executor Close와 기존 sandbox session/transaction Close | 기존 종료 확인 뒤 storage finish 연결. UNKNOWN에서는 관련 삭제 금지 |
| staging | `promotion.LocalStaging.stageDocuments`; 임시 tree 실패 정리, sync·원자 게시와 intake 재해시 | 게시된 transient의 마지막 소비·소유권·자동 삭제 |
| 재사용 cache | `promotion`의 Go/Cargo/Python/UV/PNPM/Bun/Terraform 및 npm project cache와 승인 검증 | 공통 quota/TTL·참조 pin·실제 bytes·동시 예약·회수 |
| evidence | `evidence/local.Store.Record`의 durable batch와 `DeleteRun` 명시 삭제 | 자동 retention caller, active/reference 보호와 유한 총량 |
| release 재생 사본 | `artifact/githubrelease.HeliopauseReleaseSet.captureReplayRun/DiscardReplayRuns`; descriptor-relative root·directory/file identity·UID/mode/link 검증 | 전체 저장소 GC가 아니다. 기존 한정 삭제와 Close 순서를 유지 |
| 사용자 target/rollback | 기존 promotion guard/commit/rollback·설치 records | target을 transient로 분류하지 않음. recovery 참조 및 별도 accounting 연결 |

`internal`의 파일 inventory에는 공통 storage/lifetime/retention owner가 없다.
기존 PyPI resource policy는 artifact/runtime 예산이며 cross-store retention이 아니다.
`cli/language_settings.go`는 표시 언어 설정만 소유한다. bootstrap은 현재 cache root에서
intake/evidence/staging 및 client를 구성한다. 새 공용 local adapter가 필요한 책임은
설계 §7.2의 ledger·예약·lease·참조·sweep에 한정한다. 기존 획득/검사/승격 구현은 유지한다.

## 먼저 확인된 종료 전 근거 보존 gap

- `application.InspectService.Inspect`는 inspection provider error 때 이미 완료된
  verification의 Evidence를 기록하지 않는다. 같은 단일 dependency 경로도 해당한다.
- `inspection/npm.CompositeInspector.Inspect`는 뒤 inspector가 실패하면 앞의 완료된
  report를 버린다. PyPI composite와 graph 경로는 실제 계약·테스트를 확인한 뒤 수정한다.
- `application.freezeCheckPlan`은 provider의 선언 실패만 반환하며 선언 중 취소된 context의
  원인을 별도로 보존하지 않는다. 직접 회귀로 확인한 범위만 수정한다.

이는 cleanup 전에 durable 근거를 남길 선행 작업이다. 부분 report는 정상 완료/ALLOW/
승인 세트가 아니며 필수 검사 선언을 충족한 것으로 취급하지 않는다.

## MISSING과 구현 순서

1. 종료·취소의 원인과 이미 확인된 근거를 보존하는 위 경로의 표적 RED/GREEN.
2. 실제 생성 owner와 기존 최상위 operation에 공통 수명·durable 원장·예약을 연결.
   install의 중간 dependency 종료로 intake를 지우지 않는다.
3. 기존 종료 경계 후 transient 회수와 bounded 시작 복구. identity/lease/참조 불명은 보존.
4. 기존 cache/evidence/rollback owner의 유한 정책·실제 bytes/files·공간 admission·pin·회수.
5. 설정/결과의 policy·설치·cleanup 축을 연결하고 설계 §7.5의 반복·동시성·crash·거부 시험.

foundation은 후속 S01/S08 등의 더미 자원 수명을 받는 단계이며 C01 전체 완료와 구분한다.
같은 입력의 full/native/CPP/CI를 반복하지 않고 S09와 변경 묶음으로 필수 검증한다.

## 2026-11-02 C01 선행 경로·Evidence 원장 — 부분 구현

- 확인한 취소/기존 근거 유실은 `c01-finalization-facts-red`1.511s에서 재현했다.
  단일 inspect/install 및 batch는 실패 시 이미 확인한 근거를 bounded 독립 context로 기록한다.
  npm/PyPI composite는 앞서 끝난 보고를 유지한다. Policy/승인 세트는 만들지 않는다.
  연결4 package3.977s와 foreign/empty child 등을 포함한 표적4.751s가 PASS다.
- 공통 `core/domain.StoragePolicy`는 §7.4의 숫자를 한 곳에 두고 단위/관계/overflow와
  digest를 검증한다. `core/ports.Storage`의 opaque 예약/게시/확인된 부재 해제와
  `internal/storage.Local`의 단일 durable append 원장을 실제 Evidence writer에 연결했다.
  기존 writer의 batch·sync·원자 게시를 재사용하며 같은 문서를 한 번 인코딩해 예약/기록한다.
- 원장은 짧은 exclusive lock과 incremental replay를 사용한다. 정상 예약마다 전체 cache를
  재해시하지 않는다. legacy Evidence inventory는 bounded 시작 단계에서만 읽고 삭제 소유권을
  새로 부여하지 않는다. 성공 자료의 실제 bytes와 미완료 예약을 재시작 후에도 계상한다.
  보존 한도/동시 명령/unknown journal/교체 경로/alias 거부의 named tests가 있다.
- 최초 원장 시험은 Go `T.TempDir`의0777 생성/umask와 HAA 기존0755 parent를 동일한0700
  private child로 취급해 실패했다. owner·foreign write 차단 parent와0700 Evidence/state child를
  구분했고 source의 실제 기존 parent/evidence 권한과 대조했다. 교정된 owner 시험0.720s PASS.
  기존 사용자 디렉터리의 chmod나 자료 자동 삭제로 통과시키지 않았다.
- 최신 기본 profile92.597s는 format/drift/freshness/module/build/test-build/architecture/docs PASS다.
  이 입력에는 아직 본 baseline 문서가 없으므로 현재 문서 검사는 별도로 보완한다.

C01 전체의 네 상태는 아직 NO다. 현재 원장은 Evidence에 먼저 연결한 foundation 부분이며,
intake/derived/staging/cache/rollback 및 전체64GiB 사용량·참조 pin·자동 TTL/sweep·crash 수거·
설정/결과 축·100회 반복은 MISSING이다. 기존/새 binary의 동시 쓰기와 정책 전환도 최종
호환 경계에서 검증한다. Evidence 상한 구현을 전체 저장소 정리 완료로 확대하지 않는다.

현재 Evidence foundation의 실제 Application operation/Run 연결을 포함한 영향5 package
Race26.628s, canonical security32.180s와 전체 입력 docs0.381s도 PASS다. 중간 checkpoint는
이 부분에만 해당하며 검사·설치 전체 수거 또는 hosted/native qualification이 아니다.

### Evidence pathname 교체 — 직접 재현과 교정

`c01-evidence-root-replacement-red`0.899s는 예약 직후 controller-owned 시험 callback이
Evidence 경로를 교체하고 새 경로에 foreign sentinel/충돌 파일을 만든 경우를 재현했다.
기존 writer의 absolute-path 실패 정리가 그 새 tree까지 삭제했다. 이는 제한된 filesystem
경합 재현이며 격리 Artifact의 Host 쓰기 권한이나 원격 exploit을 입증한 것은 아니다.

기존 `ReadReference`의 ancestor/descriptor identity 검사를 한 `Store.openRoot`로 공유하고
쓰기·게시·명시 retention을 그 pinned root에 연결했다. 새 실패 정리는 처음 만든 directory와
해당 batch의 정확한 문서만 확인해 제거하며 알 수 없는 member는 보호한다. 확인된 부재를
원장에서 해제하기 전에 parent sync를 수행한다. root 교체와 동일 tree의 unknown member,
기존 writer/reader 및 원장의 실제 filesystem 시험1.183s가 PASS다. C01 전체 수거는 여전히 OPEN이다.

또한 `hosttool.Executor.Close`는 Docker client state 삭제이며 전체 runtime 부재 증명이
아니다. `sandbox.Backend.Execute`는 cleanup 실패를 nil Go error의 INCOMPLETE 결과로 반환할
수 있다. 따라서 다음 intake 수거 연결은 최상위 error==nil이나 executor Close만으로
runtime drain을 판정하지 않고 해당 session/Close 사실을 연결해야 한다.

### 묶음 검사 최초 실패와 inert composition

C01/S09 Quick106.534s는 기본 단계/워크플로/go vet까지 PASS 후 새 오류 문자열의 ST1005에서
멈췄다. 문자열만 수정한 고정 Staticcheck14.263s는 PASS다. 미도달 default stage를 canonical
platform221.983s로 이어 실행했고50 package 모두 terminal, bootstrap의 pipx/uv 두 inert
preflight만 실패했다. 다른 package의 성공을 잃거나 실패를 숨기지 않는다.

원인은 bootstrap의 새 원장 생성자가 입력 오류/정적 도움말 처리보다 먼저 cache를 만든
것이다. 같은 Local adapter를 inert 구성하고 첫 실제 예약에서만 bounded 원장 초기화를
수행하도록 바꿨다. 검증을 복제하거나 초기화 우회 wrapper를 추가하지 않았다. 두 기존
실패와 원장 owner/alias/동시성 표적5.703s PASS이며, 전체 default를 같은 입력으로 반복하지
않고 변경된 owner와 실제 Application first-use를 추가 확인한다. C01 전체 및 hosted/native는 OPEN이다.

최종 영향 owner Race3.137s, 고정 표적 Staticcheck4.621s, canonical security12.788s와 Darwin
amd64 build1.523s는 PASS다. Darwin은 compile-only이며 native 실행 증거가 아니다.

## ecf1202 이후 — intake 등록 IN_PROGRESS

첫 npm 작은 HTTP fixture는 다운로드가 원장 등록보다 앞선 것을0.841s에 재현했다.
성공 fixture의 Content-Type도 기존 intake 계약에 맞춰 교정했다. 같은 Local의 root/opaque
key·quota를 intake에 확장하고 생성 owner에서 기존 tarball byte/file bound를 예약한 뒤
다운로드한다. oversize 판정용1byte까지 예약하며 기존 security limit은 바꾸지 않는다.

Npm writer/실패 정리도 held root를 사용하고 알 수 없는 잔여 member는 보호한다. 실제
HTTP 성공/실패·기존 npm owner·동일 Run의 intake/Evidence8.138s PASS다. 성공 intake는
실제 bytes/tree를 seal해 RETAINED로 계상하며 검사/staging 전에 삭제하지 않는다. 이 등록은 마지막 소비/실제 runtime
종료 증명·자동 수거 완료가 아니다. 다른 생성 owner와 finish/reference 계약을 계속 연결한다.

### 현재 command의 last-use — 부분 연결

Npm intake는 생성 시 IN_USE, 실제 bytes/tree hash 게시 뒤 RETAINED로 계상한다. Application은
기존 Evidence 기록 경로 한 곳을 재사용해 필수 실제 검사의 완료를 전달한다. Bootstrap의
최상위 defer는 기존 guard/runtime close 뒤 bounded Finish를 호출하며 명령 실패/검사 미완료는
보호한다. 재시작한 다른 owner의 항목은 이 메모리상 last-use 사실만으로 수거하지 않는다.

Root identity·seal hash·single-link를 확인한 뒤 DELETING을 durable 게시하고, 각 member의
identity/content를 제거 직전에 다시 읽고 REMOVED를 기록한다. 전체 inventory만 재사용한
최초 코드의 post-inventory mutation은0.859s RED였고 leaf 재대조로 교정했다. inode 교체/
내용 변조/추가 파일/hardlink/재시작 owner는 작은 filesystem 시험에서 보존됐다.

이는 현재 정상 완료 npm 경로의 부분 연결이다. 실패/취소/정책 거부·모든 runtime 자원과
다른 intake/cache/staging, active shared lease와 crash recovery의 전체 수명은 아직 MISSING이다.

현재 부분 연결의 영향 storage/application/npm Race63.064s, 고정 Staticcheck20.776s,
기존 bootstrap preflight16.577s, canonical security20.456s와 Darwin compile10.878s는 PASS다.
[부분 checkpoint](./storage-intake-local-20261102.json)는 실패/교정과 각 source 입력을 구분한다.
Darwin 실행·전체 C01 및 최종 hosted qualification은 여전히 OPEN이다.
