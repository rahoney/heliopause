# Runtime upgrade maintenance plan — M13 재개 전 선행 작업

2026-10-15 사용자 요청으로 M13 release 작업을 보류하고 업데이트 유지보수를
먼저 수행한다. 기술 결정의 owner는
[설계](../engineering/05-runtime-upgrade-maintenance.md)이며 이 문서는 작업 순서,
baseline audit와 acceptance를 소유한다. 실제 상태는
[Current Work Queue](./02-current-work-queue.md)를 따른다.

## 1. 시작 기준선과 M13 보존

- Source baseline: `f5a03530f514f7fdf1a3ec7ce52f15dd515b1019`.
- 보존 branch: `milestone/m13-production-release-operations`.
- 유지보수 branch: `milestone/runtime-upgrade-maintenance`, 같은 baseline에서 분기.
- 시작 시 tracked/index clean. 기존 사용자 문서·lessons·local raw evidence는
  미추적 상태 그대로 보존하며 전체 staging이나 ignore 변경을 하지 않는다.
- M13-001 COMPLETE는 유지한다. M13-002는 `ON_HOLD`이며 acceptance 미완료다.
  M13-003..005는 시작하지 않는다. 보류는 사용자 요청에 의한 우선순위 변경이며
  원격 규칙·branch·PR을 변경하거나 기존 승인 범위를 확장하지 않는다.
- 마지막 M13 관측은 A′의5/5 bounded writes, PR33의 Required/CLA 성공과
  시험 스레드 해결 후 CLEAN이다. 정상 main merge는 미승인이다.
  [A′ 원본](./evidence/m13-002-protected-path/bundle-a-prime-result.json)과
  [남은 계획](./evidence/m13-002-protected-path/README.md#remaining-qualification-and-approval-plan)을 보존한다.
  이는 2026-10-15 새 원격 조회 결과가 아니다.

## 2. Repository baseline audit

이 표의 상태는 **이번 작업의 목표 범위** 기준이다. 이미 있는 부분은 재사용하며,
QUALIFIED/ACCEPTANCE_CLOSED는 그 목표의 전체 완료를 뜻한다.
Source를 쓰기 전에 수행한 감사 결과이며 현재 목표별 gap을 분리한다.

| 목표 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 근거 / MISSING |
| --- | --- | --- | --- | --- | --- |
| Runtime lock와 product identity 생성 | YES | YES | YES | YES | `scripts/runtimes.lock.json`, `scripts/generate-runtime-lock.go`, `internal/runtimeidentity/runtime_lock_gen.go`; 기존1.26.9 drift/Quick 근거 재사용 |
| HAA host/product/quality setup/guest Go 일치 | YES | YES | YES | YES | `scripts/check/tools.go:checkGoVersionConsistency`, runner compiler check,8 regressions, f5a0353; archived binary compiler·gVisor 전체 목표와 구분 |
| 모든 managed Go quality binary의 actual compiler 일치 | NO | NO | NO | NO | source-installed4개는1.26.9; official archive Gitleaks8.18.4는1.22.1; `setupGo`만의 검사는 archive metadata를 증명하지 않음 |
| 모든 소비자의 version ownership inventory | NO | NO | NO | NO | 개별 lock/consumer는 존재; host/CI/release/gVisor/manifest/support 간 누락 없는 inventory·cross-check 필요 |
| 명시 upgrade plan/apply/verify operation | NO | NO | NO | NO | unified command 부재; 기존 generator/check를 조합하는 작은 개발 도구 필요 |
| 모든 runtime의 배포 manifest 정확성 | YES | YES | NO | NO | `scripts/runtime-image-manifest/main.go:buildManifest`는 Node/Python/Go만 내보내고 Node version에 npm을 사용; Rust와 실제 runtime version gap |
| gVisor의 Go owner 연동 및1.26.9 build | NO | NO | NO | NO | 기존 bundle/observer builders는 존재; upstream `go.mod`/Bazel 선택1.26.3, 새 compiler bundle·SDK patch·hash 검증 필요 |
| Terraform/GitHub의 유지보수 경계 | YES | YES | NO | NO | 기존 adapters·Go dependencies·exact artifact 계약 재사용; 업그레이드 영향 inventory 연결 필요 |
| Upgrade별 qualification 선택·기록 | YES | NO | NO | NO | canonical profiles/actual integration/CPU·3CUDA 근거 존재; 새 compiler/bundle 후보의 분류·실행·결과 연결 필요 |
| 검증된 upgrade의 host 전환·retirement | YES | NO | NO | NO | M8/M10 및 Go 단일화 설치 절차 존재; 새 workflow와 영구 backup/복구/readback 연결 필요 |

원본 범위는 [Go 단일화](../engineering/evidence/go-single-version-20261015/README.md),
[M12 qualification](./evidence/m12-005-qualification/README.md),
[M12 red-team](./evidence/m12-02-red-team/result.json)을 따른다. 새 updater나
새 gVisor 후보가 이미 qualified라는 의미로 과거 PASS를 사용하지 않는다.

감사에서 찾은 기존 구현은 대체하지 않는다. 위 manifest gap과 generator/검사의
연결을 좁게 수정한다. 다른 구현으로 대체가 필요해지면 기존 위치·재사용 불가 이유·
수정만으로 해결되지 않는 이유·영향을 먼저 제시한다.

## 3. 실행 순서

한 번에 하나의 item만 IN_PROGRESS로 둔다. 아래 순서는 dependency 순서이며
성급한 병렬 구현·여러 언어 동시 최신화·여러 CUDA 동시 실행을 하지 않는다.
각 item 시작/종료 상태와 검증 commit은 Queue에서 갱신한다.

### UPG-001 — baseline audit, 설계와 실행 계획

Read: lessons INDEX와 ci/artifact/sandbox/evidence, Architecture, trusted tooling,
quality/CI, M8/M10/M12/M13, 실제 locks/builders/consumers.

Acceptance:

- 목표별 다섯 audit 상태와 정확한 기존 구현·gap을 기록한다.
- M13 보류·branch checkpoint·미완료 acceptance·재개 조건을 기록한다.
- 단일 owner, Terraform/GitHub boundary, bootstrap·generation·verification·
  activation/retirement, PyTorch matrix와 실패 복구를 설계한다.
- docs/security·candidate diff 검사를 통과하고 한글 commit으로 보존한다.
- 이 item에서는 product source·runtime pin·설치·원격을 변경하지 않는다.

### UPG-002 — ownership inventory와 read-only drift 연결

Depends: UPG-001. Read: 설계2/3/6절, quality/CI, 실제 generator/check/manifest.

작업:

- Go/Node/Python/Rust/gVisor/tool/support/CI/release/host archive의 선택·파생 항목을
  기존 lock에서 연결한다. Node의 실제 runtime version과 Rust manifest gap을 수정한다.
- Terraform Provider·GitHub asset은 signed inspection subject로 구분하고 HCL 등
  구현 dependency와 common runtime의 영향만 연결한다.
- 기존 runtime drift 단계에 missing/duplicate/inconsistent consumer 검사를 붙인다.
  자동 network resolve·source rewrite·새 CI job 없이 현재 버전에서 먼저 검증한다.
- Workflow/release validator의 selected-version literal도 owner와 연결한다.
  과거 버전 금지 목록·negative fixture·historical receipt는 갱신 대상으로 오인하지 않는다.

Acceptance:

- 현재 pins를 보존한 재생성이 반복 byte-identical이며 canonical Quick이 PASS다.
- 실제 파일에서 Go setup/guest·Node/npm·Python/pip/ABI·Rust/manifest·support mismatch를
  각각 주입한 fixture가 실패한다. Unknown consumer는 성공으로 흘리지 않는다.
- 모든4개 runtime의 manifest entry와 image/version/architecture binding을 확인한다.
- Version review 날짜·지원 판정을 자동으로 갱신하지 않는다. Product/Policy 의미는 유지한다.

### UPG-003 — explicit plan/apply/verify와 같은 버전 rehearsal

Depends: UPG-002. Read: 설계4/7절, M8/M10, coding/security, 관련 lessons.

작업:

- 작은 repository maintenance command 하나로 baseline→proposal→frozen patch→
  canonical validation을 연결한다. 기존 checker·generator·tools를 호출한다.
- Go bootstrap은 기존 coordinator executable과 별도 target SDK로 해결한다.
- Exact official metadata의 획득과 offline validation을 구분한다. 영구 journal과
  hash precondition으로 dirty/stale input·부분 write·중단 복구를 처리한다.
- 테스트용 미래 version/metadata fixture는 로컬 생성 동작만 검증하며 존재하는
  공식 release 또는 qualification으로 주장하지 않는다.

Acceptance:

- 현재 버전 plan/apply 두 회에서 불필요한 semantic diff·추가 SDK가 생기지 않는다.
- 같은 frozen input은 동일 patch·consumer list·검증 목록을 만든다.
- Fixture의 coordinated update는 연결된 모든 설정을 수정하고 partial update,
  unknown component, wrong digest/architecture, stale HEAD/hash를 거부한다.
- Go·Node/npm·Python/pip/ABI·Rust/Cargo 각각의 독립 fixture로 변경 범위와
  동반 소비자를 확인한다. Terraform/GitHub는 사용자 artifact version을 변경하지
  않고 adapter/common-runtime 영향만 계획하는지 확인한다.
- Network acquisition failure, write interruption, 외부 동시 변경에서 최초 원인과
  복구 상태를 보존하며 source/host를 무단 덮어쓰지 않는다.
- Canonical Quick/security/workflow/release-gate/docs 및 목적 있는 updater 회귀가 PASS다.
- Host activation과 daemon 설정은 이 item의 실행에 포함하지 않는다.

### UPG-004 — gVisor·archived quality tool의1.26.9 후보 build

Depends: UPG-003. Read: 설계5절, existing builders/verifier/patch, M8/M11/M12.

작업:

- Exact 기존 gVisor source/patch와 Go1.26.9로 SDK 준비를 공통화한다.
- Candidate capture→actual member hashes/module lock 고정→strict verify 순서로
  전체 bundle·observer를 준비한다. CI/release 경로가 같은 준비 입력을 사용하게 한다.
- SDK patch conflict나 unsafe/internal-runtime 변화는 기존 owner에서 분석한다.
  실패를 runtime source 대체·budget 확대·검사 suppression으로 우회하지 않는다.
- Gitleaks는 현재8.18.4의 exact official source/commit/module identity로1.26.9
  source-build 후보를 준비한다. 기존 source-install 기능과 version 검증을 재사용하고
  새로운 output digest를 고정한다. Gitleaks와 gVisor는 독립 변경 단위로 commit한다.

Acceptance:

- Compiler metadata가1.26.9이고, six members·capabilities·102 latches가 맞는다.
- 동일 frozen build 입력의 독립 build로 member bytes/hash 재현성을 확인한다.
  차이가 생기면 stamp/path/toolchain 등 원인을 밝혀 재현 조건을 고정한다.
- New SDK·patch·Bazel/module/builder·member identities가 candidate lock에 연결된다.
- Wrong SDK/patch/bundle member·missing capability의 기존 부정 검사가 실패한다.
- 모든 managed Go quality tool의 actual compiler metadata가 선택 Go와 일치한다.
  Gitleaks의 기존 검토에서 확인한 검출/미검출 사례를 회귀 corpus로 고정해 official
  archive와 새 build에 각각 실행한다. 기존 탐지 범위·예외를 유지하며 불명확한
  source/차이는 조사한다. 과거 검토가 자동화된 corpus로 이미 존재한다고 가정하지 않는다.
- 아직 privileged 설치 또는 full qualification 완료로 표시하지 않는다.

### UPG-005 — installed cross-ecosystem와 CPU/CUDA qualification

Depends: UPG-004. Read: 설계6/7절, M12 tuple/resource 계약, canonical workflow/lifecycle.

작업 순서:

1. Exact candidate/source freeze와 native projection 필요 여부를 결정한다.
   Projection이면 copy exit0와 bytes/modes 대조 후 dependent checks를 시작한다.
2. Disk/resource·downloaded graph·installed backup·helper ownership·복구 preflight.
3. Canonical bootstrap/Quick/security/vulnerability/workflow/docs/release-gate/
   qualification-freshness 및 compile/platform 검사.
4. Live CUDA qualification은 설계6절의 명시 test-only plan으로 official selected
   metadata와 exact auxiliary target pin을 준비한다. Stale pin은 main intake 이전에
   거부하며 frozen corpus 또는 원래 dependency graph를 강제로 바꾸지 않는다.
   작은 실제 resolver/build/install과 observer/lifecycle 정상·부정 회귀.
5. 기존 npm/PyPI/GitHub/Go/Cargo/Terraform installed integration과 CPU full.
6. 같은 candidate로 cu126, cu130, cu132를 독립 full 실행한다.
7. Final candidate 검사, raw/source/runtime/result 결합과 실제 cleanup 확인.

Acceptance:

- 기존 profile별 required consumer와 coverage·resource·source boundary가 유지된다.
- Ordinary Required와 세 CUDA full 결과가 각각 새 후보에 연결되며 NOT_RUN/UNKNOWN을
  PASS로 바꾸지 않는다. 원격 실행이 필요한 항목은 실제 새 run의 완료를 확인한다.
- Host·network·disk·fixture·제품 원인을 구분하고 original error를 보존한다.
  원인 분석 없이 같은 실패를 반복하거나 timeout/coverage를 완화하지 않는다.
- Helper parent wait·CONFIRMED_STOPPED·원본 복구/성공 activation 상태가 명확하다.
- Remote write/CI dispatch는 필요한 exact 후보·profile·회수/시간을 준비한 후
  기존 사용자 권한 범위에 따라 수행한다. M13 보호 규칙이나 PR merge를 섞지 않는다.

### UPG-006 — 전환, retirement, 운영 안내와 M13 인계

Depends: UPG-005. Read: 설계7/8절, M8/M10, lessons sandbox/evidence.

Acceptance:

- 검증된 HAA/guest/gVisor와 managed Go quality binary의1.26.9 후보를 기존 설치 계약으로 일관되게 적용한다.
  실제 metadata/hash/owner/mode·doctor·작은 production consumer를 확인한다.
- 안전한 rollback backup·영구 journal과 중단 재개 절차를 실제 fault test 범위와
  함께 문서화한다. Helper UNKNOWN 아래 privileged restore를 하지 않는다.
- HAA-owned 이전 SDK/cache만 inventory/consumer 확인 후 정리한다.
- 모든 지원 생태계에 update entrypoint 또는 명시적인 적용 제외 사유/검증이 있다.
  사용자 Provider/package·다른 host 환경을 정리하지 않는다.
- 현재 버전 reconciliation과 다음 버전 fixture rehearsal을 canonical 경로로 재현한다.
- 독립 한글 commit·최종 audit·남은 외부 상태를 기록하고 M13 재개 여부를 판정한다.

## 4. 실행 전 공통 checklist와 오답노트

각 item 시작·재개·압축 후 INDEX와 해당 detailed lesson을 읽는다. 새 확정 원인,
검증·수정 반증과 item 종료 때 기존 항목을 갱신한다. Lessons를 executable input,
새 CI gate 또는 권한 확대 근거로 사용하지 않는다.

| 실패 유형 | 먼저 사용할 기존 교훈 / 예방 |
| --- | --- |
| Host/guest/외부 SDK 또는 오래된 cache 혼동 | ci L22, evidence E12: 실제 executable·digest·consumer와 source pin을 구분 |
| 준비 중인 projection에서 check 실행 | evidence L16: copy 실제 완료→bytes/modes 대조→검사 |
| optional/import/entry point·설치 역할 혼동 | artifact L01/L02: pinned installer 의미·required coverage 유지 |
| helper timeout/재부팅/불확실한 restore | sandbox L08/L09: 영구 원본·실제 ownership/wait·UNKNOWN 구분 |
| old green을 new source/runtime/profile에 전용 | INDEX6, evidence E12: exact input/candidate/profile별 증거 |
| 신규 parser/dependency로 Architecture 위반 | coding/security와 canonical Quick 우선; 기존 구현의 좁은 수정 |

Before each expensive run: frozen candidate와 첫 failure 확인, active process/lock,
disk·memory·duration·graph bounds, raw evidence 위치와 backup/restore owner를 확인한다.
전제조건 미충족은 준비·분석 단계로 돌아갈 이유이며 제품 실패/성공으로 바꾸지 않는다.

### 작업량과 주요 불확실성

달력 기간은 아직 추정하지 않는다. UPG-002/003은 기존 코드 연결과 결정론적 회귀가
중심이고, UPG-004의 SDK patch/Bazel 재현성과 UPG-005의 실제 qualification이
주요 변동 요인이다. 첫 build·CPU 결과를 얻으면 실제 소요·disk peak와 원인을
기준으로 남은 실행 계획을 갱신한다.

| 위험 | 먼저 하는 작은 확인 | 실패 시 다음 조사 |
| --- | --- | --- |
| gVisor SDK patch conflict | exact Go1.26.9 SDK에 patch application·작은 build | patch가 바꾸는 Go source와 upstream 차이; 검증 생략 없이 좁은 호환 수정 |
| compiler 변경에 따른 runtime 관찰 차이 | trace capabilities·102 latches·작은 실제 consumer | observer admission/accounting와 SDK/runtime owner |
| PyTorch source/graph drift | tuple·wheel ABI·metadata/hash와 availability 사전 확인 | source/packaging 변화와 기존 snapshot 차이; 임의 dependency 추가 금지 |
| 대용량 disk/time 부족 | pinned graph 합계·free space·retained data·복구 공간 | exact 사용량과 active references; 원본 증거/다른 환경 삭제 금지 |
| 외부 CI·registry 실패 | actual stage/status와 raw cause 보존 | 공식 상태·transport·rate limit·runner 차이; 제품 회귀와 분리 |

기존 qualification duration bound는 CPU15분, 각 CUDA40분이다. 이는 성공 예상
시간이 아니라 실패 상한이며, setup/build/queue 시간을 포함한 전체 일정 보장이 아니다.
최종 qualification은 한 frozen 후보로 모아서 수행하고 기능을 고칠 때마다 모든
대용량 검사를 즉시 반복하지 않는다. 수정 영향의 작은 회귀를 먼저 닫고 후보를 다시 고정한다.

## 5. 범위·완료와 M13 재개 조건

이번 유지보수는 현재 owned versions를 기준으로 자동화하고 Go/gVisor1.26.9
전환을 닫는다. Node/Python/Rust/PyTorch를 동시에 최신으로 올리지 않는다.
Terraform CLI·remote modules·plan/apply 지원, 새로운 CUDA profile, Docker daemon
재설정·OS 재시작·공개 release·보호 branch merge는 추가하지 않는다.

M13 재개 전 필요한 것:

- UPG-001..006 acceptance와 실제 근거가 모두 닫히고 새 runtime 후보의 영향·검증이 명확하다.
- 기존 separate full-Race finding과 release follow-up의 disposition을 갱신한다.
  이 작업 중 사라졌다고 추측하거나 green을 위해 숨기지 않는다.
- 유지보수 branch의 PR/reviewer merge와 전달 CI가 필요하면 정상 Git 계약으로 완료한다.
  오래된 M13 branch로 단순 switch하여 유지보수 변경을 잃지 않는다.
- Git/원격 보호 규칙·PR33 refs/head/checks/conversation/CLA를 새로 조회한다.
  보류 전 CLEAN 또는 승인 후보를 현재 상태로 재사용하지 않는다.
- 최신 merged maintenance source를 반영한 M13 baseline audit와 남은 acceptance를
  갱신한 후 M13-002를 재개한다. 정상 main merge의 별도 승인 조건은 유지한다.

UPG-001 문서 완료는 전체 유지보수 완료가 아니다. 이 계획이 기존 M13 acceptance를
대체하거나 public release readiness를 선언하지 않는다.
