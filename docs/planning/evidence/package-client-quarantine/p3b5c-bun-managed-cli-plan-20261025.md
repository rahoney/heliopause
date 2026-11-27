# P3-B5c Bun managed CLI 감사·수정 지도

M13-004의 B5b2 target transaction local qualification/commit 다음 착수한다. B5b2 검증 중에는
이 단계 source를 수정하지 않는다. Read: B5a/B5b1/B5b2 감사·실제 consumer, client 설계§3/4,
`cli/package_client.go`, `cli/pnpm_project.go`와 test, `bootstrap/pnpm_client.go`, bootstrap.Run,
기존 JSON mutation report와 CLI preflight/사용자 흐름·Runtime/Trust canonical 계약.

| 책임/실제 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| CLI descriptor registry·localization·preflight/report/progress | YES | YES(PNPM/UV) | YES(existing scope) | YES(existing scope) | Bun typed descriptor/request/service |
| Bun typed single-selection guarded workflow/target transaction | YES(B5b2 candidate) | YES(actual) | 진행 중 | NO | 최종 local 검증/commit 후 CLI 호출 |
| Bun add/install/update CLI | NO | NO | NO | NO | 등록/정적 identity/native factory/report·실제 user command |
| unmanaged/native adoption·explicit Bun executable lane·bunx | NO | NO | NO | NO | 이후 별도 acceptance |

## Exact edit map·두 번째 검토

1. `cli/bun_project.go`: PNPM의 bounded control read/target·symlink ancestry/args/duplicate
   옵션·version identity preflight를 재사용한다. Bun typed request/selection intent와 bun.lock
   부재 semantics를 전달한다. Typed Bun service만 호출하고 exact published Manifest/SBOM/cache
   records와 current verified context를 확인한 기존 JSON mutation report를 사용한다.
   현재 managed scope에는 --adopt를 추가하지 않는다. Host native install/실행/재resolve 없이
   static help/invalid args가 끝나는 기존 CLI lifecycle을 유지한다.
2. `cli/package_client.go`: 구현된 Bun add/install/update 세 descriptor만 registry에 추가한다.
   Source는 npm, client는 Bun으로 분리한다. `helox bun x`/bunx·workspace·other registries·
   명시 Bun Artifact runtime을 미리 등록하지 않는다. 도움말은 고정 Linux amd64·single hoisted
   public graph·새 검역·root scripts 보존/미실행·Node-compatible lifecycle/current managed scope를
   설명한다. 기존 PNPM/UV descriptor/aliases/언어/flags는 유지한다.
3. `bootstrap/bun_client.go`와 bootstrap.Run: 고정 BunVersion을 static annotation으로 먼저
   연결하고 선택 Bun의 플랫폼 검사·actual native/helper factory·request-local service를 연결한다.
   State/cache roots는 Bun 전용, intake/evidence Source owner는 기존 npm을 재사용한다.
   기존 PNPM/npm 분기와 owning runtime close·result/cleanup cause를 유지한다.
4. `cli/bun_project_test.go`, bootstrap static/actual CLI tests: 정상3operation·empty absence,
   잘못된 args/flags/path/control·link/mode/cross-client·missing completion records·help
   no-runtime/no-cache·JSON 계약/현재 native operation/Evidence/Policy·no-op inode·user 파일 보존을
   검증한다. Existing CLI/PNPM factory·launcher help/default checks 및 실제 PNPM CLI를 보존한다.
   Exact installed test client/owning typed helper·Host 전후 identity와 정리를 확인한다.

새 source registry/Domain policy/observer/network/filesystem 권한을 추가하지 않는다. 기존
구현을 대체하지 않고 실제 남은 caller gap만 연결한다. 현재 Bun Node lane과 나중 explicit Bun
runtime을 같은 지원 주장으로 합치지 않는다. CI412는 frozen7544의 결과만 확인하며 이후
cache/transaction/CLI 후보를 그 결과에 전용하지 않는다. 작은 helper별 push/CI 없이 의미 있는
최종 caller 묶음과 문서의 필수 qualification 경계를 따른다.
