# P3-B4 Bun native layout·lifecycle·검역 연결 지도

기준 cc062c58099169736d0dad246984641e2582f2b8/treeca032b240e60c70630a1e74df99a7806f4dc8cf8,
2368개 전체 Root/private index·blob·mode·disk 일치다. M13-004 하나만 IN_PROGRESS다.
B3c typed native selection/source/snapshot의 named local 검증을 재구현하지 않는다.
작은 helper별 push/CI 없이 backend 연결 단위로 검증하며 다음 원격 후보의 macOS/Required
gate는 유지한다. CI를 실행하면 계획§5의 실제 타이머 확인과 후속 작업을 이어간다.

Read: Queue/PROJECT-DECISIONS/docs router, 확장 계획§3–6, package-client 계약§1–5,
Architecture adapters·inspection/sandbox/policy, Domain dependency verified-set·inspection 계약,
B3c 지도/영수증, 기존 PNPM inspection context/budget/layout/prepare/transport와 Bun lock/selection,
lessons artifact L19·sandbox L03/L08/L18·evidence L11/L16·ci L06.

## Baseline audit

| 책임/기존 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun native typed 선택·독립 source·snapshot, cc062c5/d1d3625 | YES | YES | YES(named local) | YES(named selection subset) | archive/layout 검역은 별도 |
| npm acquisition·SRI·archive inventory·static inspection | YES | YES(existing npm/PNPM) | YES(existing scope) | YES(existing scope) | Bun frozen 전체 archive/instance binding |
| PNPM bounded tree archive·exact acquired binding·native offline prepare | YES | YES(PNPM) | YES(existing scope) | YES(existing scope) | Bun hoisted layout/명령 의미 대조; PNPM store/layout 가정 전용 금지 |
| Bun offline native cache/import·hoisted tree | NO | NO | NO | NO | fixed native primitive로 실제 표현·원본 controls·archive/file 동일성 확인 |
| Bun lifecycle/runtime·static-all/동적 coverage·Policy 연결 | NO | NO | NO | NO | Bun과 Node identity/실행 의미를 분리하여 기존 검사 owner에 연결 |
| Bun cache/transaction/CLI | NO | NO | NO | NO | B4 검역 뒤 후속 단계; 이번 primitive 성공으로 완료하지 않음 |

## B4a: 구현 전 native primitive와 두 번째 검토

1. 고정 Bun binary SHA와 고정 Node base를 사용한다. 공개 실제 raw lock fixture의 exact
   tarball을 lock의 SHA512와 독립 대조하고 task-owned 입력으로만 보관한다.
2. Network-none/read-only runc diagnostic에서 UID/GID1000·cap drop·no-new-privileges·유한
   memory/pids/tmpfs와 readonly native mount를 사용한다. Host에서 Bun 또는 artifact를 실행하지
   않는다. 이는 명령/표현 조사이며 gVisor observer나 검역 qualification을 대체하지 않는다.
3. 고정 file archive를 script-free native seed로 넣은 뒤 원래 public manifest/bun.lock을
   `--offline --frozen-lockfile --ignore-scripts --no-env-file --backend=copyfile --linker=hoisted`
   로 읽을 수 있는지 확인한다. 실패는 원본 stdout/stderr/명령으로 보존하고 cache key를 추측해
   조작하거나 public lock을 local dependency로 바꾸어 성공을 주장하지 않는다.
4. 전체 bounded cache/tree tar와 원본 controls를 data로 수집한다. Host extraction·link following은
   하지 않는다. 실제 layout/metadata/bin 표현을 확인한 뒤 exact binder/production prepare의
   수정 지도를 보완한다. 새 observer 권한은 실제 최초 거부 근거 없이 추가하지 않는다.
5. Exact owned container 신원·종료·삭제·정상 query의 부재를 확인한다. 네트워크 없는 primitive
   exit0를 archive 승인·lifecycle 완료·Policy·Host 반입으로 확대하지 않는다.

두 번째 검토: PNPM 구현을 대체하지 않으며 source/cache 입력은 artifact 안전성 authority가
아니다. Root project scripts는 보존하고 실행하지 않는다. Cached package의 name/version만으로
bytes를 승인하지 않으며 후속 binder는 전체 acquired inventory와 동일성을 검증해야 한다.
현재 source 변경은 없고 위 diagnostic 결과에 따라 실제 MISSING만 다음 edit map에 적는다.

## B4a 첫 관측과 다음 수정 지도

`p3b4a-bun-offline-primitive-v4.json`의 network-none native seed는 exit0였으나 public frozen은
`--offline: "is-number" is not in the cache`로 exit1이었다. File archive cache는 실제
`@T@sha512-…` 경로와 그 alias를 사용했다. Public lock/manifest bytes와 exact container cleanup은
보존됐다. File seed를 public cache import로 권장하지 않는다. 앞 v1/v2는 Bun 전 입력 전달 실패,
v3는 seed 뒤 Docker cp의 tmpfs 조회 실패다. 원본·한계는 각 raw와 assessment에 남긴다.

공식 [global cache 문서](https://bun.sh/docs/pm/global-cache)는 registry cache의 name@version
배치를 설명한다. 다음 diagnostic은 검증한 동일 archive의 bounded regular entries만 이 문서의
배치로 전달하여 native가 frozen public lock을 읽는지 확인한다. 임의 cache 내용의 name/version을
authenticity로 사용하지 않고 실제 전체 파일 digest와 source archive의 일치가 후속 필수다.
이는 앞 file-cache 경로를 추측해 registry 경로로 바꾸는 구현이 아니다.

병행 가능한 실제 source gap의 exact map:

1. `internal/artifact/npm/npm_project_acquired.go`: 기존 PNPM BindAcquired의 exact identity/
   declared=observed SRI/size/digest/intake-run·unique-run 검사를 공용 private 함수로 기계 이동한다.
   기존 PNPM caller의 snapshot 비교와 원래 오류 sentinel을 유지한다.
2. `bun_inspection_context.go`: 기존 Bun ProjectSnapshot과 독립 bound selection을 보관하는
   typed context/input을 추가한다. 같은 공용 acquired 검사를 사용하며 전체 frozen snapshot의
   context/source/controls/dependencies/graph를 비교한다. 공유 archive의 모든 Bun tree instance를
   보존하고 getters는 caller mutation을 차단한다. 이 입력은 structural binding이며 static/
   runtime/Policy 권한이 아니다. Empty는 explicit dependency-free snapshot으로만 표현한다.
3. `internal/sandbox/bun_resolver.go`: 기존 source binding을 Bun InspectionContext로 연결하고
   기존 BindProjectSnapshot은 그 snapshot을 반환하게 한다. Native 명령/권한은 바꾸지 않는다.
4. Tests: empty/single/transitive/peer와 중복 archive의 여러 tree instances, wrong/partial acquired,
   copied digest의 다른 snapshot, 외부 getter mutation·zero values·source/control 변경 거부,
   기존 전체 PNPM acquired/snapshot 회귀를 확인한다.

두 번째 검토: 이미 구현된 acquisition·Domain snapshot·PNPM 검사를 재작성하지 않는다.
공용 helper는 두 실제 소비자만 가지며 client별 lock/layout은 합치지 않는다. 이 context 연결은
Bun offline/layout/lifecycle 검역 완료가 아니므로 B4 전체 acceptance는 NO로 유지한다.

실제 source consumer 추가 검토: `internal/sandbox/bun_inspection_source_real_test.go`는 현재
versioned empty/single/transitive/peer controls를 그대로 Parse→BindBunLock→InspectionContext로
읽고 기존 Acquire/IntegrityVerifier/InspectArchive를 모든 unique archive에 적용한 뒤 context의
완전 acquired coverage와 모든 instance를 대조한다. Test-only consumer이며 fake SRI fixture나
native/runtime/Policy 완료를 주장하지 않는다. 기존 source owner·한도·검사 기준은 유지한다.

Native cache 후속 source 대조: 고정 source `PackageManagerDirectories.rs`의 기본 name@version에는
`CacheVersion::CURRENT=1` suffix `@@@1`이 붙는다. Options의 did_override_default_scope는
옵션 존재가 아니라 최종 registry URL hash와 default hash의 차이다. 기본과 같은 URL을
explicit override로 해석한 v6는 잘못된 해석이며 실패 그대로 보존한다. 이 source 두 조건을
반영한 v7의 network-none frozen install은2.239s/exit0, 원본 controls 보존·container 부재 확인이다.
문서 축약 배치 v5/file seed v4는 실패이며 source/body 연결된 해석 영수증이 범위를 소유한다.
이 단일 package runc primitive를 production gVisor/layout/lifecycle qualification으로 쓰지 않는다.
