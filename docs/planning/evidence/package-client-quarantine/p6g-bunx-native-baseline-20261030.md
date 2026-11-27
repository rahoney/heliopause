# P6-G — Bun one-off native/current baseline

Source0eeefca/tree5b7a0a21의 PNPM F1/F2/F3는 named local 검증을 마쳤고 CI439가 진행 중이다.
첫 실제20분 GET은09:21:54UTC/18:21:54KST다. 이 소스를 새 push로 취소하지 않는다.
같은 M13-004의 다음 missing은 bunx/bun x이며 이 문서는 source 작성 전 audit다.

최신 상태: G1 public exact/bare native·독립 metadata/SRI와 CPP132/Quick372.878s/영향 두 package Race55.672s/tagged Staticcheck8.247s가 PASS다. 아래 baseline 표는 source 작성 전 상태이며 Docs0.355s/security15.619s까지 PASS이며 현재 검증은 [G1 receipt](./p6g1-bun-cold-public-local-20261030.json)를 따른다. 현재 whole 검사·Node/Bun program·CLI와 general/hosted/final은 OPEN이다.

Read: PROJECT/Queue/docs router, client09§1–3/현재 Node lane, plan20§4–6, F baseline,
lessons INDEX·ci L06·sandbox L18의 Bun query/clone/runtime 소비자·artifact L02/L19·
evidence L16, 아래 고정 원본과 실제 owners. 기존 P3의 project 성공은 one-off 성공이 아니다.

| 책임 / 실제 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun 고정 도구·typed network/resource/관측·정리 — sandbox/bun_resolver.go, native_project_scope.go/native_project.go | YES | YES | 기존 project named YES/CI438 | 기존 named YES | 실제 bun x root/내부 add와 cold cache만 확인; 새 범용 권한·중복 resource owner 금지 |
| native tree/peer/lock와 독립 공개 원본 — artifact/npm/bun_lock*.go/bun_selection.go | YES | YES | 기존 project named YES | 기존 named YES | one-off 실제 controls/한 primary와 whole instance graph; cache 이름을 identity로 쓰거나 가짜 InstallContext를 만들지 않음 |
| Bun whole static/layout/instance 검사 — bun_inspection_context.go/bun_inspection_layout.go, sandbox/bun_inspection_prepare.go/bun_instance_backend.go | YES | YES | 기존 project named YES | 기존 named YES | target 없는 현재 실행 소비자 연결; 기존 구현 재작성 금지 |
| 현재 whole 검사·Evidence/Policy/records와 readonly Node 실행 — application/execution_*.go, bootstrap/node_execution_*.go, artifact/npm/execution_*.go | YES | YES | 기존 Node/PNPM named YES | 기존 named YES | Bun native choice/physical layout의 실제 차이와 entry만 결합 |
| 고정 Bun Artifact runtime — sandbox/bun_artifact_runtime.go/bun_target_lifecycle.go | YES | YES | 기존 Bun lifecycle named YES | 기존 named YES | 별도 one-off readonly program admission/current output·완료 결과; Node 성공을 Bun 성공으로 전용하지 않음 |
| bunx/bun x native DATA·현재 Node/Bun program·CLI | NO | NO | NO | NO | 실제 cold census부터 아래 순서로 구현 |

## 고정 원본과 확인한 차이

Bun1.4.4/tool SHA10702245766b225e70bbf6235fd6b47214fd94b4760d00ffaed8ab0df57ac235,
107,308,936B는 repository lock을 따른다. Observed source38afee6c는 binary source-build
인증이 아니다. [고정 bunx source](https://github.com/oven-sh/bun/blob/38afee6c84fcf48c0edb2466aa4b27d287a59c30/src/runtime/cli/bunx_command.rs)의
읽은 파일은59,701B/SHA0c2882488b20588f9bc0888afbaa800213ebb3b3024bf48b636efb78adca690f다.

- `cache_dir`는 install-cache와 TMPDIR의 `.bunx-<uid>`를 후보로 검사하고 실제 directory,
  현재 uid·private mode를 요구한다. 기존 Bun native volume은 BUN_INSTALL_CACHE_DIR과
  XDG_CACHE_HOME을 `/tmp/haa-bun-input/cache`, TMPDIR을 같은 root의 tmp로 이미 고정한다.
  Bun의 `scope.run`에는 PNPM 전용 earlyStore=true를 넣지 않는다.
- Cold install은 cache root에서 자신의 고정 binary로 `bun add <spec> --no-summary`를
  실행한다. `--ignore-scripts`는 내부 add에 전달되며 --trust보다 우선한다. Refresh 때는
  기존 bun.lock/bun.lockb를 지우지만 fresh controls의 실제 형상은 아직 미수집이다.
- Default/local/PATH/cache 조회와 bin 선택은 별도 단계다. 기본 helper의 first-bin fallback을
  HAA의 ambiguity 허용으로 옮기지 않는다. Explicit package/command와 Node/Bun runtime
  의미를 구분하며 native metadata 준비에서 artifact bin·사용자 argv를 실행하지 않는다.

## 최소 순서와 두 번째 검토

1. **G1 actual cold DATA:** 기존 Bun scope/보호 도구·관측·정리를 재사용한 tagged 준비 진단으로
   실제 cache/controls와 정확한 terminal을 먼저 확보한다. PNPM의 relative pkg 링크나 lock
   형상을 가정해 공용 reader를 확장하지 않는다. 고정 sealed missing command 후보는 pinned
   source와 대조하며 query 명분의 Node/Bun artifact 실행 권한을 선행 추가하지 않는다.
2. Source/관측 실패는 첫 코드·trace·원본을 보존하고 최소 owner를 조사한다. Expected stderr나
   native exit0는 완료 authority가 아니며 incomplete/cleanup 오류에서 selection을 비운다.
3. 실제 확인한 형상만 기존 Bun parser/독립 metadata·whole originals에 결합한다. 이후 기존
   current Policy/readonly Node와 명시 Bun runtime을 각각 검증하고 CLI를 연결한다. 현재
   PNPM/NPM 원본·layout/argv/출력 owner는 실제 두 소비자가 생기는 지점에서만 공유한다.
4. 싼 표적 검사 후 완성된 native/current/CLI 단위로 필수 검사를 모은다. CI439 대기 동안 같은
   소스의 full/native/CPP를 반복하거나 helper/문서별 CI를 추가하지 않는다.

작성 전 baseline에서는 G1 native cache 형상/실행·새 ABI 필요 여부가 미확정이었다. 새 package/interface/job·
완료 프로토콜·권한을 미리 만들지 않는다. Bun/일반 peer 동등성·hostile·P7/final acceptance는 OPEN이다.

## G1 첫 실제 결과 — 성공 미확정

기존 observer989b와 고정 도구10702245에서 9.00s/test·12.216s/operator FAIL이다.
Producer는 정확한 missing-binary exit1을 냈고 cold tar25,496,576B/309entries를 남겼지만
외부 관측은 PROCESS_PROVENANCE_UNKNOWN/SENTRY_EXEC/events13·session_complete=false다.
따라서 DATA 형상 관측에만 사용하며 native 선택 완료·Policy/프로그램 성공은 아니다.
원본은 raw `p6-g1-cold-native-v1`에 보존했다. Host5/Docker 동일·helperwait0/confirmed stop·
보호 profile 같은 inode 원복을 확인했다. 첫 표적은 기존 Bun project admission의
새 x root 인자이며 exact clone/내부 add 필요 여부는 아직 별도 확인 대상이다.

실제 cache 아래 `.bunx-1000/typescript@5.9.3+ignore-scripts+registry=cea06e0b14d1fa09`에
package.json/bun.lock/node_modules와 기존 registry cache가 있었다. Native-generated
package.json은 dependencies 단일 TypeScript5.9.3이며 lock schema2/config1이다.
Cache 경로·producer stderr를 원본 인증이나 관측 완료 authority로 쓰지 않는다.

G1 표적 root admission RED43.652s는 기존 observer에서 실패했다. 첫 후보는 합성 cwd/project와 실제 cwd/root가 달라 v2도 같은13events FAIL이므로 superseded다. 실제 root와 맞춘 최소 후보의 admission GREEN40.355s 뒤 v3 actual9.98s는 root를 통과한14events에서 OCI `/` directory·flags622592 읽기의 첫 STREAM_FAULT를 기록했다. 내부 add/clone은 아직 관측 완료가 아니다. 각 원본·Host 보존/원복을 따로 기록하고 같은 입력의 full/CI는 반복하지 않았다.

G1 v4 actual8.16s는 OCI directory read 이후 SENTRY_EXEC/events14에서 미완료였다. 권한을 추가하지 않은 private v5 observer의 bounded exec diagnostic은 실제 root9→clone10의 고정 Bun add7argv(--force/--ignore-scripts/고정 registry)를 확보했다. Root는 원래 bytes이며 진단 hook은 private builder에만 있었다. 첫 child 후보의 표적 negative는 foreign image를 거부하지 못해 FAIL했고 권장 제외다. 새 child는 ARTIFACT로 demote하고 exact creator/현재 root package·7argv·sealed tool·one-shot marker만 결합한다. 기존 IsTrustedControlNetwork의 direct-root 조건과 arbitrary artifact/descendant 권한 없음은 유지한다. 아직 native 전체 성공이나 public selector 구현은 아니다.

G1 v7의 ATTRIBUTION_FAILURE는 ARTIFACT installer를 DIRECT_EXEC_SESSION-only trusted-control-network로 보낸 후보의 연결 오류였다. 기존 CONTROL validator를 유지하고 고정 trusted-bun-install-network(ARTIFACT_GROUP/OTHER)를 Bun project profile에서만 소비한다. Go profile/decoder 음성·tagged vet/build와 C++ actual CONNECT 방출/후손 거부가 PASS다. v8은 새 attribution CONNECT16/SENDTO4를 소비했으나 OPEN_RESULT_CORRELATION/events15에서 FAIL했다. Pending/result의 실제 차이는 미확정이며 상관 검사 완화 없이 한정 진단한다. 기존 public Bun regression은 failfast로 미도달이다.

G1 v9 한정 진단은 sysno437/openat2·flags2654208(O_PATH|CLOEXEC|LARGEFILE), 성공한 private bunx cache bin의 실제 resolved path/mount와 pending_count0을 확인했다. 고정 gVisor는 Openat2도 kernel OpenResult를 내지만 entry 관측점은 없고, 고정 Bun bin.rs1509의 beneath O_PATH 호출과 일치한다. 후보는 기존 상관 검사를 삭제하거나 pending을 제조하지 않는다. 현재 exact installer의 성공한 private-cache path handle만 kernel result로 계상하며 일반 path·mount/shadow 검증과 workspace budget을 따른다. 실패·외부 경로·read/write·legacy orphan·기존 pending 불일치는 계속 거부한다. Runtime/SDK pin·job·완료 프로토콜은 바꾸지 않는다. 아직 이 후보의 actual 완료는 미확정이다.

G1 v10 exact TypeScript5.9.3의 cold census9.52s와 기존 Bun native/source9 cases54.49s는 관측·완료·정리까지 PASS다(operator66.073s). Observer5267ba46/C++92c5d29a와 새 Go wire receiver를 사용했고 tar25,496,576B/309entries는 여전히 unapproved DATA다. Host5/Docker 불변/helperwait0/confirmed stop/profile 같은 inode 복원을 확인했다. 같은 exact/기존9 native를 반복하지 않고 bare selector와 public cache/source binding을 계속한다. Whole CPP/Go 묶음·현재 Policy/program/CLI/새-source hosted/final은 별도다.

G1 새 bare v11은 SENTRY_EXEC/events14에서 FAIL했다. 기존 diagnostic binary를 재빌드 없이 private log bind로 재사용한 v12는 argc8/internal add에 --no-cache가 force 앞에 추가됨을 확인했고 이전 raw log hash도 보존했다. Exact의 argc7은 바꾸지 않는다. Bare DATA는 당시 latest TypeScript7.0.2/^7.0.2·platform optional graph이며 고정5.9.3/Node 프로그램 성공으로 쓰지 않는다. 새 bare 요청만 최소 cache-mode 수정 뒤 재검증하고 통과 exact/기존9 native는 반복하지 않는다.

G1 bare v13은 기존 exact/old9를 반복하지 않고10.43s/operator12.622s에 complete·forbidden0/정리 PASS다. Current observer59b2d92e/C++b4c55db1과 bare fixture1d30de8c이며 Native-network CONNECT96/SENDTO64, fresh DATA31,713,280B다. TypeScript7.0.2/platform optional graph의 원본 인증·Node/Bun 프로그램 지원은 별도다. Host5/Docker·helperwait0/프로필 같은 inode 원복을 확인했다. Exact+bare/기존9 actual을 재사용하고 최종 C++132와 공개 cache/parser/source 연결 뒤 Go 검증을 묶는다.
