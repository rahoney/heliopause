# P5 — execution foundation / npx 구현 전 baseline

기준 source12034457a69eabed0429d05d96dbf79bbef3409a/tree d4b0e921/2883files다.
승인된28commit은 CI422에서 검증 중이며 이번 문서·상태 정정은 그 source에 포함하지 않는다.
최초 감사 당시 Root/private1281runtime source SHA는 C-e2/v2와 같았고 production 변경은 없었다.
후속 P5-A의 실제 수정·한정 결과는 아래 지도와 receipt를 따른다.
현재 단일 active work item은 M13-004다. 이 문서는 대기 중 독립 baseline 감사이며 P5
구현/qualification 완료나 새 runtime/CLI capability를 뜻하지 않는다.

Read: Queue/plan20 §4·5·7, client09 §1·3·4, D-006/008, Architecture-005,
Contract-002, Domain Run/Artifact/VerifiedSet의 invariant·Sandbox 계약과 해당 source owner,
lessons INDEX와 artifact L02/ sandbox L03·L12·L18/evidence L16.

## 실제 owner 재사용 판정

| 책임 / 실제 source·symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | P5 MISSING |
| --- | --- | --- | --- | --- | --- |
| `artifact/npm/pnpm_layout_binding_data.go:pnpmInventoryBins` | YES | YES(PNPM/Bun layout) | YES(existing layout tests/current Quick) | YES(layout scopes) | 공개·typed exact bin 선택 adapter와 current whole graph/entrypoint binding |
| `pnpm_inspection_layout.go:pnpmNodeBinTarget`/`ParsePNPMNodeBinShim` | YES | YES(layout proof) | YES(existing unit/actual layout) | YES(layout scopes) | P5 selected entrypoint의 closed Node interpreter binding; 새 shim/parser 재작성 금지 |
| 기존 acquisition/Inspection/Evidence·Policy/`LocalStaging.Stage` | YES | YES(install/quarantine) | YES(named earlier/current local) | YES(named scopes) | run에서도 fresh current whole set 검사를 연결; staging은 보관 전용 |
| `ports.Sandbox.Execute`/`domain.SandboxRequest`/`SandboxResult` | YES | YES(dynamic inspection) | YES(existing named/current consumers) | YES(inspection scopes) | full graph+exact bin+argv/runtime/policy를 가진 user-run request/result; 기존 request 위장 금지 |
| `sandbox.Backend`/PNPM·Bun instance execution/admission·observer/lifecycle | YES | YES(inspection) | YES(named scope; current common15) | YES(named scopes) | user-run의 distinct output/exit/cancel·fresh workspace lifecycle·network deny |
| `hosttool.Executor.RunOutput`/`RunInput`/`RunBounded` | YES | YES(trusted infrastructure) | YES(existing consumers) | YES(그 scopes) | stdout/stderr를 분리·bound·관측하고 exit/signal을 보존하는 좁은 실행 transport |
| `cli.safeTerminal` | YES | YES(references/paths/codes) | YES(existing CLI tests) | YES(240rune reference scope) | 임의 프로그램 stream의 UTF-8/CTL/OSC/크기 정책과 결과 분리 |
| npx/npm exec descriptor·fresh run application/context·결과 renderer | NO(P5 lane) | NO | NO | NO | user grammar→current whole 검사→exact sandbox run→별도 execution result |

이름이 PNPM인 `pnpmInventoryBins`는 Bun도 이미 재사용한다. Bin map/string, scoped package의
기본 command, archive 정규 경로·실제 member·command 문자를 검증하며 bundled/embedded
node_modules와 directories.bin은 현재 layout 범위에서 거부한다. 같은 parser를 새로 쓰지 않고
기존 inventory owner에 필요한 typed reader만 검토한다. Layout proof는 실행 승인 authority가 아니다.

기존 `SandboxRequest`는 acquired artifact 한 개만 보유한다. `SandboxResult`는 disposed Session의
bounded raw observations이며 user argv·stdout/stderr·프로그램 exit를 담는 결과가 아니다.
검사 완료를 user tool 실행 성공으로 바꾸거나 기존 constructor에 fake subject를 넣지 않는다.
기존 Host executor는 등록된 trusted tools만 실행한다. `RunBounded`의16KiB 진단창은 버리도록
명시돼 있어 프로그램 출력으로 serialize하지 않는다. `RunOutput`은 stderr/exit/overflow 정책을
합의하지 않은 infrastructure primitive다. `safeTerminal`의240rune reference 제한을 그대로
program stream 지원으로 확대하지 않는다.

## Exact npm source 확인 — DATA only

현재 runtime lock의 Node OCI image에서 npm11.19.0의 package.json/npx-cli.js/exec.js/
libnpmexec/index.js/get-bin-from-manifest.js를 never-started/network-none/read-only container로
export했다. [출처·원본 digest 기록](./p5-npm-source-20261028.json)은
고정 image·다섯 file SHA·실제 npm version·container 삭제/Host Docker 전후 동일성과
원본 export receipt digest를 연결한다.
JS/Node를 Host에서 실행하지 않았다. 원문은 같은 raw prefix에 한 번만 보존하며 code/SDK pin을
변경하거나 이 조사를 새로운 runtime 지원으로 표시하지 않는다.

Exact npx wrapper는 npm exec로 변환하며 첫 positional/`--` 이후 구분을 npm에 전달한다.
Exec는 local/global bin·package lookup, config/script-shell과 package 선택/설치를 다룬다.
`get-bin-from-manifest`의 실제 분기는 bin target 값이 모두 같으면 첫 key, 그 외 package basename
matching bin이 있으면 그 이름, 아니면 오류다. Comment 순서만으로 실제 분기 순서를 추측하지 않는다.
HAA 계약의 ambiguity/명시 package≠command 요구는 별도로 적용한다. Native inference 또는
local/global/cache를 검사 뒤 실행 source/완료 권한으로 사용하지 않는다. Source에 있는 broad
npm flags/allow-scripts·shell-call을 그대로 forwarding하는 것은 초기 lane의 구현 방법이 아니다.

## 남은 수정 지도 — source 작성 전 재검토 대상

1. 기존 graph/acquisition/전체 검사·staging readback과 inventory bin parser를 잇는 정확한
   ephemeral execution context를 정한다. Host InstallTarget을 가짜로 만들지 않고 identity,
   graph/entrypoint/runtime·source/policy fingerprints와 argv를 함께 동결한다.
2. Initial noninteractive/stdout lane의 요청/결과·명령 문법과 bounds를 고정한다. 원래 inspection
   port/result를 바꾸지 않고 필요한 책임에 최소 타입을 추가한다. Source의 existing owner를
   재사용할 수 없는 이유가 생기면 구체 영향과 승인 필요 여부를 먼저 다시 검토한다.
3. 검증한 bytes를 staging 밖 fresh sandbox에 복사·재확인한다. 기존 supervisor/observer,
   direct-exec admission·accounting/cleanup을 재사용한다. Env/PATH/Host project mount와
   runtime 설치·추가 dependency download를 artifact 입력에서 허용하지 않는다.
4. 실행 exit/signal/timeout과 quarantine/policy/output/cleanup 상태를 분리한다. Raw output은
   신뢰 데이터가 아니며 HAA human/JSON과 섞지 않는다. CTL/OSC/UTF-8/overflow와 stdout·stderr
   원래 원인 보존을 검증하고 stdin/TTY·파일 export는 명시 capability 전 미지원이다.
5. Bare package/bin 선택과 explicit package/command·`--` 경계부터 정상/ambiguous/missing bin,
   wrong/shebang/runtime/source substitution·cache 변조, re-download/추가 실행·shell injection,
   output 폭주·nonzero/signal·취소/child orphan·helper/FD/container/slice 누락을 검증한다.

새 package/config/interface/job scaffold를 만들지 않았다. P5 source 구현 전에는 위 지도와
기존 implementation 위치를 다시 대조한다. CI422 결과 확인은 독립20min timer를 우선하며
source 변경을 새 push해 실행 중 후보를 취소하지 않는다. Whole current hosted/final4profiles/
signed와 P5/P6/P7 acceptance는 계속 OPEN이다.

## P5-A — exact bin 선택, source 작성 전 지도와 두 번째 검토

Local baseline ccbd93f/tree8153faad/2885files이며 current runtime1281은 CI422와 같다.
현재 단계는 M13-004 안의 독립 P5-A다. P4 hosted/final 결과를 전제로 실행 권한을 추가하지 않는다.
`pnpmInventoryBins`/`ArchiveInventory`는 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES
(기존 구조/layout 범위)다. P5 typed entry 선택은 네 상태 모두NO이며 MISSING은 정확한 command,
원본 acquired identity/handle·manifest·target file facts binding과 ambiguity 거부다.

수정은 기존 `internal/artifact/npm`의 새 `execution_entry.go`와 그 회귀에 한정한다. 기존 parser,
Node shebang validator·runtime/observer/inspection/Policy·CLI를 재작성하지 않는다. Constructor는
기존 bin parser를 호출하고 자동 선택은 단일 bin의 이름이 package basename과 같을 때만 허용한다.
다중 bin은 동일 target alias여도 명시 선택이 필요하며 단일 다른 command도 명시한다. 선택 결과는
전체 immutable inventory와 정확한 regular target facts를 보유하고 다른 intake handle/manifest/
member facts와의 substitution을 거부한다. 이 값은 구조적 선택일 뿐 bytes/Node interpreter/현재
전체 검사·Policy/실행 승인이 아니다. Host 파일 열기·실행·환경 탐지는 하지 않는다.

두 번째 검토: upstream npm의 단일 target alias 추론과 HAA의 명시 ambiguity 계약을 구분했다.
기존 parser의 strict JSON/path/bundle/실제 regular member 검사를 우회하지 않는다. 새 생성자는
다른 `SandboxRequest`/가짜 InstallTarget 또는 placeholder interface/package/job을 만들지 않는다.
회귀는 bare·scoped/string·명시 다른 command와 다중 bin/alias·누락·duplicate/unsafe target·
zero value·동일 이름의 다른 acquisition/manifest/target digest/size/mode 치환을 검증한다.
Canonical format/quick와 기존 npm 전체 Race로 기존 parser 소비자도 확인한다. CLI/run/전체 P5
acceptance는 미완료이며 이 작은 구현을 이유로 새 push/CI를 시작하지 않는다.

### P5-A named local 결과

[로컬 receipt](./p5a-execution-entry-local-20261028.json)의 focused2.040s/Quick37.506s/npm 전체
Race49.102s/docs0.267s/security14.043s가 PASS다. 기존1281source와 parser는 그대로이며
새2file만 추가했다. Exact/scoped/명시 command·다중 bin/같은 target alias와 다른 acquire handle/
manifest·target 및 다른 member digest/size/mode 치환·zero value를 검증했다. 첫 Quick265.854s는
새 test의 noncanonical base32 RunID로 inventory fixture 생성에서 실패했으며 기존 canonical
생성자로 고쳤다. 최초 실패 원본을 보존했고 production 검증을 완화하지 않았다.
P5-A 구조적 선택은 IMPLEMENTED=YES/WIRED=NO(실제 P5 run 흐름)/QUALIFIED=YES(local)/
ACCEPTANCE_CLOSED=YES(선택 범위)다. P5 전체 WIRED/QUALIFIED/ACCEPTANCE_CLOSED는NO이며
실행/CLI/현재 whole 검사·runtime/입출력·취소는 다음 missing이다. CI422에는 이 source가 없다.

## P5-B1 — selected bytes / Node runtime, source 전 audit·map·두 번째 검토

Source6dd3136/tree feab8ef9/2891files이며 Root/private runtime1283은 동일하다. CI423이 이 source를
검증 중이고 다음 변경은 그 payload 밖이다. Existing `ExecutionEntry`/`ArchiveInventory`와
`pnpmNodeBinTarget`는 기존 layout 범위의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES다.
ExecutionEntry API는 IMPLEMENTED/QUALIFIED=YES(local)이나 P5 run 연결은 WIRED=NO다.
P5 selected bytes와 Node runtime binding은 네 상태 모두NO이며 이 missing만 수행한다.

기존 npm package의 새 `node_execution_entry.go`와 회귀에서 `ExecutionEntry.BindNodeTarget`
을 추가한다. 기존 parser/Node validator를 재작성하지 않는다. ArchiveFileLimit20MiB 안의
size를 먼저 확인하고 caller bytes를 복사한 다음 원본 target digest와 기존 closed shebang/
JS-extension 규칙을 검증한다. 빈 JS도 original size/digest가 맞으면 구조적으로 유효하다.
결과는 immutable bytes·exact entry와 기존 `runtimeidentity.NodeImageReference`를 함께
보유한다. Generic runtime override/Host PATH/interpreter 추론·새 port/package/job은 없다.

두 번째 검토: metadata facts의 digest와 실제 bytes 일치는 필요하지만 source authentication,
현재 whole graph·검사/Evidence·Policy·실행 허가는 별도다. 이 value를 approval token으로
쓰지 않는다. Caller/returned slice 변경과 같은 크기의 다른 bytes·truncated/extra/oversize,
wrong interpreter/options·zero value와 fixed runtime·전체 inventory binding을 회귀로 검증한다.
기존20MiB와 Node pin은 유지하고 canonical format/quick/npm fullRace를 수행한다. 실행·CLI와
whole frozen graph/argv/policy context는 다음 P5 gap이며 이 API 성공으로 닫지 않는다.

### B1 첫 architecture 실패와 범위 교정

첫 Quick43.383s는 새 artifact/npm의 runtimeidentity import를 architecture gate에서 거부했다.
PyPI의 기존 특수 의존을 npm에도 허용된다고 일반화한 새 후보의 책임이며 기존 production
실패가 아니다. 첫 두 source/log를 raw에 보존했고 import allowlist를 넓히지 않는다.
B1은 bytes/closed Node entrypoint 형식의 immutable value만 담당한다. Type 자체가 Node를
요구하며 정확한 canonical runtime pin은 다음 조립 계층의 full execution context에서 결합한다.
즉 위 초기 B1 지도 중 exact pin 연결은 B2의 MISSING으로 정정한다. 기존 formatter/parser/
Node validator·byte/hash bound는 유지하고 교정 후보의 canonical 검증을 수행한다.

### P5-B1 named local 결과와 실제 연결 상태

[로컬 receipt](./p5b1-node-entry-bytes-local-20261028.json)의 corrected Quick256.332s/npm fullRace
49.546s/docs0.379s/security13.967s가 PASS다. Source1283과 기존 validator는 그대로이고 새2file만
추가했다. Bytes size/SHA·closed Node 선언·전체 원본 binding·복사 불변과 wrong bytes/length/
interpreter/options/oversize/zero·빈 JS를 검증했다. Architecture 실패 후보와 원본을 보존했다.
B1은 IMPLEMENTED/QUALIFIED=YES(value local), WIRED=NO(P5 run/CLI), ACCEPTANCE_CLOSED=YES
(value 한정)다. A의 parser 내부 재사용도 실제 P5 flow 연결을 뜻하지 않으므로 WIRED=NO로
명확히 기록한다. Exact runtime pin/whole graph·argv/source-policy는 B2에 남고 전체 P5는
WIRED/QUALIFIED/ACCEPTANCE_CLOSED=NO다. CI423은 이전6dd source이며 이 변경을 포함하지 않는다.
