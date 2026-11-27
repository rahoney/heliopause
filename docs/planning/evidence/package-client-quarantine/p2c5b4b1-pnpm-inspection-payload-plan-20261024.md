# P2c5-B4b1 — bound layout만 Artifact 소개 payload로 재구성

Baseline2f577ce/tree1142f303. [B4 지도 §3/5/8](./p2c5b4-pnpm-production-inspection-plan-20261024.md),
[B3c 소개 경계](./p2c5b3c-pnpm-layout-binding-plan-20261024.md), Architecture02/03·Domain08/12,
artifact L15/L20·sandbox L08/L12/L18·evidence L11/L16을 읽고 같은 M13-004/P2를 계속한다.

## 1. Baseline audit

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 전체 acquired/instance/file/link/bin binding — PNPMInspectionInput.BindLayout | YES | YES(prepare) | YES(named local) | NO(P2) | 결과를 실행용으로 최소 재구성 |
| 정리 뒤 값 반환 — PNPMInspectionPreparer.PrepareInspection | YES | YES(actual) | YES(named local) | NO(P2) | fresh Artifact session의 consumer |
| private exact root manifest / immutable Entries·ReadFile / v2 digest | YES | YES | YES(local) | NO(P2) | 소개 payload와 readback에 결합 |
| bounded PNPM tar parser — ParsePNPMLayoutArchive | YES | YES | YES(local) | NO(P2) | payload-specific exact comparison |
| PyPI readback — verifyClosureInstallation·scanClosureTar | YES | YES(PyPI) | YES(existing) | NO(P2) | PNPM에 그대로 사용 불가: wheel RECORD 전용·link 거부 |
| PNPM 실행용 tar renderer/copy/readback | NO | NO | NO | NO | 이번 B4b1 |
| actual fresh instance backend/aggregate/Host/CLI | NO | NO | NO | NO | B4b2/B4c/후속; 이번 값으로 완료 처리 금지 |

기존 parser·PyPI scanner·단일 Backend.Execute를 재작성하지 않는다. PNPM의 원래 tar를
추출해 Host 파일로 만들지 않으며 원본 user controls/native whole tar를 그대로 전달하지 않는다.

## 2. 정확한 수정 위치와 값 경계

`sandbox/pnpm_inspection_payload.go`에 private `pnpmInspectionPayload`와
`preparePNPMInspectionPayload(ctx, bound)`를 둔다. Zero/취소/빈 bound digest·manifest·entries를
runtime 작업 전에 거부한다. 새 tar는 project/ 아래 다음 자료만 소유한다.

- project/package.json: bound.InspectionManifest의 private exact dependency/group data만.
- project/node_modules 이하: bound Entries/ReadFile의 검증된 package/dir/symlink/bin.
- .modules.yaml·.pnpm/lock.yaml·freshness cache는 제외한다. Raw controls·환경·author/config/
  user scripts 또는 별도 private 파일을 추가하지 않는다.

Header owner는1000:1000, root dir0700/root manifest0600이다. 그 밖의 검증된 mode·link와
모든 file bytes는 보존한다. Directory write restriction은 앞 prepare의 explicit view가 이미
적용했으며 여기서 추가로 권한을 부여하거나 다른 mode를 보정하지 않는다. 기존 payload/
workspace bound를 적용하고 native 성공 exit나 문자열을 source/완료 authority로 삼지 않는다.

Renderer는 sorted deterministic header와 zero timestamp를 사용한다. 전체 재구성한 tar의
SHA256과 base layout v2 digest를 별도로 보존한다. Opaque Domain value를 직접 JSON marshal해
digest를 생략하는 방법은 쓰지 않는다. Payload hash는 소개할 bytes의 문맥이며 Policy가 아니다.

## 3. Hardlink 의미 보존과 비교

원래 parsed layout의 내부 hardlink closure를 그대로 읽고, 소개할 **남은 file들**의 inode
동치 class를 계산한다. 제외할 metadata를 target으로 가진 alias도 class를 남은 member에
투영한다. Class의 lexicographic 첫 path를 regular로 먼저 쓰고 나머지는 그 path의 direct
hardlink로 쓴다. Forward/chain serialization을 제거하되 남은 file 간 공유 inode 의미와
bytes/mode/owner는 보존한다. 같은 class의 mode/owner 불일치는 추측해 보정하지 않고 거부한다.

Docker cp가 어느 inode member를 regular header로 먼저 기록하느냐는 tar serialization 순서다.
그 선택을 안전성으로 사용하지 않는다. Readback은 canonical class representative를 계산해
**동일한 공유 inode partition**을 요구한다. 동일-byte라는 이유로 독립 파일을 hardlink로
합치거나 기존 공유 파일을 독립 copy로 바꾸는 것도 거부한다. TypeDir/symlink와 regular/alias
분류·모든 실제 file digest/size·mode/owner·정확한 symlink target·전체 이름 coverage는 별도로
검사한다. Alias 검증을 basename 또는 digest 동일성만으로 대체하지 않는다.

## 4. 고정 전송과 readback

`copyPNPMInspectionPayload(ctx, runner, container, payload)`는 exact container ID에 고정
Docker cp --archive stdin만 수행한다. Host bind·임의 대상 인수·셸 option을 받지 않는다.
고정 private project volume 이외 대상은 없다. `verifyPNPMInspectionPayload`는 기존 bounded
Docker output으로 **project 전체**를 읽는다. Native node_modules parser의 구현을 private
fixed-root helper로 공유하고 `ParsePNPMInspectionPayloadArchive(ctx, body)`가 고정 project
root를 소비한다. 기존 ParsePNPMLayoutArchive는 고정 node_modules root를 그대로 유지한다.
Public arbitrary-root option은 추가하지 않고 원래 모든 header/owner/PAX/alias/parent/EOF/
resource 거부를 공유한다. Root dir/manifest를 포함한 전체 expected entries와 bytes·mode를
정확히 비교한다. Manifest와 node_modules만 따로 읽어 root .npmrc 같은 extra 파일을 놓치는
방법은 쓰지 않는다. 기존 controls reader는 이번 payload scanner로 대체하지 않는다.

Native layout을 다시 BindLayout으로 검사하면 제외한 metadata를 다시 요구하므로 적절하지
않다. 이미 bound된 실행용 expected entry/class와 새 actual을 정확히 비교한다. Extra/missing/
foreign file·변조·다른 sharing·symlink·owner·mode·opaque output/EOF/취소/transport 실패는
거부한다. Expected 상태를 actual output으로 갱신해 통과시키지 않는다. Pending dynamic 실행은
copy와 complete readback이 끝난 뒤에만 가능하다. 실패한 filesystem은 다음 instance에 재사용하지 않는다.

## 5. Source 작성 전 두 번째 검토와 검증 순서

이 별도 renderer는 raw unverified tar를 실행 경계로 넘기지 않기 위한 현재 work item의
누락 기능이다. 기존 formats/scanner를 변경하는 대체 구현이 아니다. 전체 byte/metadata
검증 뒤에도 root private controls 소개와 hardlink sharing은 새로운 책임이므로 따로 확인한다.
Fingerprint v2의 명시 digest projection과 byte hash를 사용하고 manifest/private 원문 유출을
피한다. Readback이 실제로 비교되기 전에는 intro 성공·Artifact 실행 완료라고 표시하지 않는다.

현재 sandbox unit의 기존 injected exact-version input fixture를 재사용해 작은 typed bound
layout을 만든다. 이는 structural controlled fixture이며 public archive authenticity/실행 승인이
아니다. NewArchiveInventory를 producer의 byte 측정 대신 사용하는 product 경로는 추가하지 않는다.
Unit은 root metadata 제외·private manifest·deterministic bytes·빈 값/취소·same-byte 공유와
독립 inode 구분·다른 representative·missing/extra/byte/mode/owner/link·transport failure를
확인한다. 기존 parser가 거부한 setup을 intended semantic negative로 분류하지 않는다.

Canonical format/security/Quick/docs와 npm/sandbox 전체 Race로 먼저 닫고, B4b2의 fresh
실제 instance backend가 이 producer/copy/readback을 소비할 때 transitive/peer·controlled
script failure/정리·root/다른 scripts0의 actual을 묶는다. Renderer만을 위해 불필요하게 또
native metadata 준비를 반복하지 않는다. 이 단계는 CLI/Policy/Host/전체 P2 acceptance가 아니다.

## 6. 2026-10-24 Host 자원 차단과 재시작 인계

이번 checkpoint는 **미검증 WIP 보존**이다. Renderer/copy/readback와 고정 project-root
parser consumer source는 작성했다(IMPLEMENTED=YES: source 존재만). WIRED=NO(production
fresh backend 미연결; controlled test consumer만 작성), QUALIFIED=NO,
ACCEPTANCE_CLOSED=NO다. Compile/unit 성공을 확인하지 못했으므로 앞 B4a·fingerprint의
PASS나 CI404를 이번 후보에 전용하지 않는다. 같은 M13-004/P2만 유지한다.

수정 source는 다음 네 곳이다. 기존 parser는 private fixed-root helper로 공유하며 public
arbitrary-root나 기존 PyPI scanner 대체는 없다.

- `internal/artifact/npm/pnpm_layout_archive.go`: 기존 node_modules parser와 새 project parser.
- `internal/sandbox/pnpm_inspection_payload.go`: 최소 payload/공유 inode partition/copy/readback.
- `internal/sandbox/pnpm_inspection_payload_test.go`: controlled positive/negative/transport 검사.
- `internal/sandbox/pnpm_inspection_prepare_test.go`: test fixture의 원본 root metadata 제외 확인.

Raw owner는 `docs/github-logs/package-client-quarantine-20261021`이다.
`p2c5b4b1-payload-first-format.json`은1.616s PASS다. First Quick147.888s와
low-memory Quick169.693s는 모두 production build의 기존2분 timeout이고 default tests는
미도달이다. 후자는 process GOMAXPROCS=1/GOMEMLIMIT=256MiB만 달랐으며 성공하지 않았다.
`p2c5b4b1-payload-focused.log`는 sandbox focused command의120s outer timeout 전까지의
capture다. Operator의 TimeoutExpired로 JSON receipt는 생성되지 않았고 unit 실행/성공은
확인되지 않았다. Security/전체 Race는 당시 미실행이다. 인계 문서를 포함한 canonical docs는
후속23.690s PASS(`p2c5b4b1-preservation-docs.json`)이며 build/unit qualification을 대신하지 않는다.

`p2c5b4b1-build-timeout-resource-snapshot.json`은 Host MemAvailable633,156KiB/swap free0,
codex RSS22,576,788KiB를 기록했다. 후속 read-only 조회는 같은 PID1404의 실제 Codex CLI
binary와 parent Node를 확인했고 RSS22,799,860KiB/MemAvailable534,648KiB/swap free0이었다.
이는 메모리 압박 관측이며 세 timeout의 단독 원인 확정이나 source 결함 증거가 아니다.
다른 process 종료·전역 설정·check timeout/coverage 완화는 수행하지 않았다. Disk 여유는
842GiB로 관측됐으며 cache 삭제를 메모리 회복 방법으로 사용하지 않는다.

재개 순서:

1. Codex의 실제 메모리 사용 process 종료/재실행으로 자원을 회복한다. 필요할 때만 Windows
   PowerShell의 `wsl --shutdown` 후 WSL을 다시 연다. 이 명령은 모든 WSL 배포판/VM을
   종료하므로 먼저 다른 WSL 작업도 저장한다. 재시작을 build PASS나 영구 수리로 간주하지 않는다.
2. INDEX와 관련 주제·실제 Git/Queue·이 leaf를 읽고 Host memory/swap, all-UID 잔여 Go tool과
   query errors를 다시 확인한다. 죽은 job의 이름이나 query 실패를 부재 증거로 사용하지 않는다.
3. Root의 네 source와 문서를 기준으로 private candidate를 동기화한다. 전체 index blob/disk
   bytes/mode/tree를 비교하며 stale private formatter의 copy-back으로 source를 되돌리지 않는다.
4. 고정 Go1.27.3 SDK·quality bootstrap을 유지하고 canonical `scripts/check`의
   format/quick/security/docs와 npm/sandbox 전체 Race를 검사한다. 기존 operator는
   `/tmp/haa-uv-runtime-0_e6zmn4/check-active-candidate.py`와
   `/tmp/haa-go1273-yzyvqne9/check-p2c5b4a-race.py`이며 workspace metadata는 raw의
   `uv-runtime-workspace.json`이다. 새 evidence prefix를 사용하고 같은 입력 실패를 덮어쓰지
   않는다. 임시 경로가 사라졌으면 현재 Git candidate와 repository lock/bootstrap으로 복원한다.
5. 실제 compiler/test finding이 있을 때 해당 owner만 수정한다. 위 범위가 통과해야 B4b1을
   qualified로 바꾸고 B4b2 fresh Artifact session/backend를 이어간다. 실제 transitive/peer와
   실패/정리 검증은 해당 consumer가 생길 때 묶으며 P2 완료 전 P3로 넘어가지 않는다.

완료된 B4a 실제 시험 세 runtime 총498,955,765B는 refs/query errors0 뒤 이미 삭제했다.
현재 작업용 private source/고정 SDK·quality/supply는 후속 검증 입력이므로 유지한다.
이 인계 기록이나 local checkpoint는 push/merge/새 CI·배포 승인이 아니다.

## 7. 2026-10-25 보존 후보의 실제 재개

사용자가 계속 진행을 요청했다. 재부팅 후 소스4/문서2와 전체2,219개 index blob·disk bytes가
직전 보존 inventory/tree2940f056과 일치함을 확인하고 private standalone checkout으로
복원했다. Root가 입력 owner이며 사라진 private formatter의 결과로 되돌리지 않는다.
Host preflight는 MemAvailable23,338,708KiB/SwapFree6,291,456KiB, all-UID live Go tools0/
query errors0이다. 이는 현재 실행 전제 회복이며 과거 timeout의 원인 확정/새 검사 PASS가 아니다.

설치 Go1.27.3과 기존 quality cache를 재사용한다. 새 짧은 owned TMPDIR, 명시 PATH/GOROOT/
GOTOOLCHAIN=local/GOENV=off/GOWORK=off/GOPROXY=off를 유지하고 도구 process만
GOMAXPROCS=2로 제한한다. Canonical profile/timeout·Artifact budget은 유지한다. 전체 index와
disk 일치 뒤 format/quick/security/docs, 기존 두 package 전체 Race 순서로 검사한다.
Warm cache를 cold bootstrap으로 표기하지 않는다. Runtime/observer의 actual 소비자는 B4b2에서
필요한 신원·입력부터 별도로 복원하며 이 renderer 검증을 위해 대형 runtime을 먼저 재구축하지 않는다.

Raw `p2c5b4b1-resume-preflight-20261025.json`과 `p2c5b4b1-resume-projection-20261025.json`,
`p2c5b4b1-resume-workspace-20261025.json`이 관측/복원 입력을 소유한다. 검사/qualification은
실제 새 prefix의 결과로 갱신한다. 같은 M13-004만 IN_PROGRESS다.

동일한 네 source의 [로컬 결과](./p2c5b4b1-pnpm-inspection-payload-local-20261025.json)는
format2.072s/Quick229.309s/security13.662s/docs0.235s와 npm/sandbox 전체 Race25.352s가
PASS다. 최소 manifest·metadata 제외·결정적 bytes·전체 project readback·같은 byte의 독립/공유
inode 구분·대표 alias만 다른 동치 배치·변조/foreign root 설정/전송/취소 거부를 확인했다.
IMPLEMENTED=YES / WIRED=NO(production B4b2 pending) / QUALIFIED=YES(named local) /
ACCEPTANCE_CLOSED=NO(P2)다. 이전 timeout은 보존하며 이번 회복을 RSS 증가 원인의 확정
수리나 public actual/Host 설치 성공으로 확대하지 않는다. 다음 B4b2의 fresh 소비자를 연결한다.
