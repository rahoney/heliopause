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

## P5-B2 — full execution context의 actual owner 대조 / source 전 지도

기준0626301/tree8afa32f1/2894files이며 runtime1285는 양쪽 사본에서 같다. CI423 source는
6dd3136으로 고정돼 있고 새 B2 변경을 포함하지 않는다. 아래 기존 owner를 실제 source로 확인했다.

| 책임 / 실제 source·symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | B2 MISSING |
| --- | --- | --- | --- | --- | --- |
| `core/domain/manifest.go:VerifiedSet/NewVerifiedSet`, `VerifiedBundle` | YES | YES(existing install) | YES(existing scopes) | YES(existing scopes) | execution context에 재사용; InstallContext가 없는 generic set이므로 대체 불필요 |
| `promotion/staging.go:LocalStaging.Stage/verifyDocuments/stageArtifact` | YES | YES(existing) | YES(existing named scopes) | YES(existing scopes) | 새 execution bundle과 현재 Evidence/source의 handoff 검증; fake target 불필요 |
| `evidence/manifest.go:Generator.Build/BuildVerifiedBundle/manifestEntries` | YES | YES(install manifest) | YES(existing) | YES(install scope) | existing entry/SBOM helpers 재사용하는 execution-only context/schema; target 필드를 거짓으로 채우지 않음 |
| `application/install_inspect.go:inspectDependency/inspectDependencyBatch`, `ProjectInspectService` composition | YES | YES(existing) | YES(existing) | YES(existing scopes) | current whole graph 검사 결과를 non-install execution outcome에 연결; 새 inspection engine 금지 |
| `LockedDependencyGraph/DependencyResolution/InspectedDependencySet` | YES | YES | YES(existing) | YES(existing scopes) | approved set과 원래 resolution 전체 동일성·선택된 primary/acquisition/member·argv/runtime/source/policy 동결 |
| `ExecutionEntry/NodeExecutionEntry` | YES | NO(P5 flow) | YES(local value) | YES(value scopes) | composition에서 generic context로 변환·canonical runtime binding |
| execution context/manifest/application run path | NO(P5 lane) | NO | NO | NO | 아래 순서로 missing만 추가 |

B2는 우선 기존 VerifiedSet/Resolution과 정확한 선택 member·OperationID·argv·required runtime을
immutable non-install context로 묶는다. Core는 artifact/npm/runtimeidentity를 import하지 않는다.
Node adapter의 bytes proof는 그대로 재사용하고 bootstrap의 기존 runtime owner로만 exact pin을
결합한다. Context 자체를 외부 승인 token이나 cached ALLOW로 쓰지 않는다. Primary entry는
현재 single-package lane에 한정하고 multi-package/derived build는 별도 capability 전 거부한다.

argv는 shell text가 아닌 복사한 vector다. Input 문법/bounds를 canonical client09에 정한 뒤
source를 쓴다. Arbitrary argv를 일반 로그·Evidence/Manifest·artifact/cache identity에 저장하지 않으며
run request 메모리에서만 보유한다. Source/policy fingerprint는 기존 normalized resolution/
Policy identities에서 도출한다. Staging은 보관 전용이고 실제 실행 전 whole current 검사와
introduction-time rehash·폐기/출력/취소는 다음 acceptance다.

구현 전 두 번째 검토에서는 existing graph/set/Policy 생성자와 source/policy 관련 canonical
계약을 다시 대조한다. New approval-set/새 staging framework·placeholder port를 만들지 않는다.
이 감사는 새 source 구현이나 B2 완료를 뜻하지 않는다. CI423 실제 timer2 due14:33:07UTC를
우선 소비하고 source 변경은 해당 source/검증 범위로만 기록한다.

### B2a 수정 지도 확정 / source 전 두 번째 검토

기존 `VerifiedSet`/`DependencyResolution`을 새 `domain.ExecutionContext`로 묶는다. 새 approval-set,
InstallTarget·SandboxRequest 위장 또는 기존 staging 변경은 없다. 정확한 primary acquired subject,
선택 member의 상대 locator/digest/size·command, required runtime identity와 argv를 복사한
immutable metadata context다. 승인 set의 전체 graph와 resolution의 nodes/edges가 같아야 하며
현재 lane은 derived/multi-package entry를 거부한다. Source fingerprint는 resolution lock digest,
Policy fingerprint는 실제 entry/set Policy ID/version만 사용하고 raw argv를 hash하지 않는다.

Core는 source/runtime adapter를 import하지 않으며 runtime 문자열 자체를 source 인증이나
실행 권한으로 해석하지 않는다. Exact Node pin 결합은 별도 bootstrap composition에서 Node
bytes proof와 함께 검증할 missing이다. Context는 fresh inspection service가 제공한 set과 함께
사용하며 cache/external JSON을 승인 authority로 재구성하지 않는다. CLI/run flow 연결은 아직 없다.

Client09의 새 input bounds(128args/총32KiB, UTF-8/NUL, 빈 값 보존)를 constructor와 독립 preflight
validator가 공유한다. Source는 기존 domain package의 새 execution_context.go와 그 회귀에
한정한다. Whole graph/primary/acquisition substitution·zero/unsafe member/invalid runtime·
argv copy와 경계·policy/source identity를 검증하고 canonical format/quick/domain fullRace를
실행한다. API 단독 성공을 runtime binding/application/실제 실행 qualification으로 전용하지 않는다.

### B2a named local metadata context 완료

[로컬 receipt](./p5b2a-execution-context-local-20261028.json)의 Quick288.508s/domain fullRace2.620s/
docs0.295s/security17.228s가 PASS다. 기존1285source는 같고 새2file만 추가했다. Generic
VerifiedSet/Resolution을 재사용하고 exact graph·primary/acquisition·member·runtime metadata,
source lock/실제 Policy identities와 private argv를 묶었다. Graph/다른 intake/secondary entry·
unsafe/zero/invalid runtime과 argv128/32KiB·복사 불변·Policy version 변화를 검증했다.
B2a는 IMPLEMENTED/QUALIFIED=YES(metadata API local), WIRED=NO(P5 actual flow), value acceptance만
YES다. Runtime 문자열은 인증/실행 권한이 아니다. 다음 B2b의 Node bytes proof→canonical runtime
composition과 fresh whole 검사/application/Manifest/staging·실제 backend/CLI는 미완료다.
CI423 source6dd를 이 결과로 확대하거나 새 source push로 취소하지 않는다.

B2a final record 뒤 첫 security6.576s는 sandbox의 shared Go cache read-only로 실패했다.
같은 환경의 go list가 정확한 cache file write 거부를 확인했으며 scanner의 cascading type
오류를 source 결함으로 단정하지 않는다. 최초 결과/진단을 보존하고 같은 pinned tool/cache를
사용 가능한 기존 권한에서 canonical docs/security를 다시 확인한다.

## P5-B2b — Node composition와 execution records, source 전 audit/map/재검토

기준49e3245/tree3f5d8a4c/2897files이며 양쪽 runtime1287이 같다. B2a final docs0.287s/
security10.961s는 기존 cache 사용 권한에서 PASS다. CI423 source6dd는 고정돼 있다.

| 책임 / 기존 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| npm NodeExecutionEntry와 domain ExecutionContext | YES | NO(P5 flow) | YES(value local) | YES(value scopes) | exact Node pin을 조립 책임에서 결합 |
| runtimeidentity.NodeImageReference / bootstrap composition | YES(existing pin/consumers) | YES(existing consumers) | YES(existing) | YES(existing) | Node proof→generic context 변환 |
| evidence manifestEntries/canonicalManifestObject/buildSBOM | YES | YES(install/project) | YES(existing) | YES(existing) | target 없는 execution schema/records 생성 |
| LocalStaging.Stage/VerifiedBundle | YES | YES(existing) | YES(existing) | YES(existing) | execution records handoff actual 검증은 다음 |
| Node context composition / execution manifest | NO(P5) | NO | NO | NO | 아래 작은 구현 및 회귀 |

수정 지도는 bootstrap/node_execution_context.go와 회귀, evidence/execution_manifest.go와 회귀다.
Bootstrap은 NodeExecutionEntry의 immutable facts를 domain 생성자로 전달하며 기존 canonical
NodeImageReference를 항상 사용한다. Runtime override 인자는 없다. Core/npm adapter import
allowlist·검사/Policy·staging·supervisor는 바꾸지 않는다. Proof가 같은 approved primary인지
검증하는 업무 invariant는 기존 domain 생성자가 담당한다. 함수는 wiring 변환만 담당한다.

Evidence의 BuildExecution은 기존 VerifiedBundle/entries/edges/canonical JSON/SBOM을 재사용한다.
새 helox.verified-execution-manifest/v1은 operation·resolver/lock·primary·전체 entry/Evidence/Policy,
exact command/member/digest/size/runtime와 source/Policy fingerprints를 기록한다. InstallTarget을
만들거나 install schema의 target을 채우지 않는다. Raw argv·argv hash와 program output을
기록하지 않으며 Manifest ID는 프로그램 인자의 identity가 아니다. 실제 argv 결합은 private
request와 기존 trusted admission 책임이다. 새 port/interface/package/job은 필요하지 않다.

두 번째 검토: 두 함수는 byte proof/whole set 생성자를 재사용한다. Cached/external ALLOW를
재구성하거나 runtime 문자열을 권한으로 해석하지 않는다. Node zero/다른 intake/graph와 argv
치환·canonical pin, record determinism/전체 graph·Evidence/SBOM/zero/cancel 및 argv 비저장을
검증한다. Canonical format/quick와 bootstrap/evidence 전체 Race를 수행한다. 실제 fresh whole
검사 application/Manifest→staging/Backend/CLI는 여전히 MISSING이며 value 시험으로 닫지 않는다.

### B2b named local composition / records 결과

[로컬 receipt](./p5b2b-node-composition-records-local-20261029.json)의 corrected Quick220.145s/
bootstrap·evidence fullRace26.899s/docs0.328s/security11.739s가 PASS다. 기존1287source는
양쪽에서 불변이고 새4file만 추가했다. Canonical Node pin·byte proof/다른 intake 거부,
whole set/current Evidence와 SBOM·canonical record digest·entry/source/policy 변화와
argv/hash 비저장·nil/zero/cancel 거부를 검증했다. 첫 Quick44.490s의 ST1005/SA1012는
새 오류 대문자와 nil negative test 표현을 교정했다. 최초 source/결과는 raw에 보존했다.
B2b는 IMPLEMENTED/QUALIFIED=YES(local composition/records), 실제 P5 flow WIRED=NO다.
Metadata helper와 record 생성 범위만 acceptance를 닫고 fresh whole application/실제 staging
handoff·backend/CLI는 남긴다. CI423 성공은 이전6dd source이며 이 source를 포함하지 않는다.

## P5-B2c — fresh complete resolved inspection, source 전 audit/map/두 번째 검토

기준338c2fd/tree e4c10fcd/2903files이며 runtime1291은 양쪽에서 같다. CI423은6dd만
ordinary named qualified이며 이 이후 P5 source는 제외한다. Active item은 여전히 M13-004다.

| 책임 / actual owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| InstallInspectService.inspectDependency / inspectDependencyBatch | YES | YES(existing install/project) | YES(existing) | YES(existing) | non-install resolved execution use case에서 호출 |
| dependencyAwareInspection.InspectGraph / whole-set Policy / VerifiedSet | YES | YES(existing) | YES(existing) | YES(existing) | 새 실행 요청에서도 fresh Evidence/Run·전체 집합 ALLOW 연결 |
| ProjectInspectService의 이미 해결된 per-entry component 조립 패턴 | YES | YES(existing project) | YES(existing) | YES(existing) | InstallRequest/target 없는 primary graph workflow |
| NPMResolver.resolveDependencies/ResolveDependencies | YES | YES(install) | YES(CI423) | YES(existing) | non-install caller wiring은 후속; 본체는 target을 사용하지 않음 |
| ExecutionInspectService / fresh result | NO(P5) | NO | NO | NO | 아래 이미 선택된 resolution 입력의 fresh 검사와 결과 |

수정 지도는 existing application package의 execution_inspect.go와 회귀만이다. 이미 선택된
DependencyResolution을 받고 ProjectInspectService처럼 기존 InstallInspectService의 per-entry
component를 조립한다. Resolver/InstallRequest/InstallTarget을 가짜로 만들지 않는다. 기존 private
inspectDependency와 InspectGraph용 batch helper를 그대로 호출하며 새 검사 엔진·Policy·
승인 집합/cache·port/interface/package를 만들지 않는다. Resolution 선택/Node proof/records/
staging/backend/CLI 조립은 다음 단계이며 이 service 성공을 actual 실행으로 표시하지 않는다.

결과는 private 생성자로만 만드는 ExecutionInspection(operation/resolution/VerifiedSet)이다.
실패 시 operation/resolution 진단은 보존하되 approved set은 없다. Complete denial은 기존
ProjectPolicyFailure와 같은 방식의 typed failure로 complete inspected facts/decision을 보존한다.
Provider/Evidence/ID/Policy 오류와 cancel은 원래 cause를 유지한다. Derived graph는 초기 lane에서
ports 호출 전에 거부한다. Graph-aware capability는 기존 install과 같은 선택 조건으로 재사용한다.

두 번째 검토: 반복 호출마다 새 operation/entry Run·acquire/verify/inspect/Evidence/entry/set Policy를
수행한다. 같은 resolution/digest를 cached ALLOW로 바꾸지 않는다. 모든 nodes와 Policy가 끝나고
현재 context가 살아 있을 때만 기존 NewVerifiedSet으로 성공한다. Entry/set 거부·missing/foreign
report·원본 provider+store 오류·취소/ID 실패와 repeated request의 fresh independent 기록을 검증한다.
Canonical format/quick와 application 전체 Race를 수행하고 기존 install/project 소비자를 보존한다.

### B2c named local fresh whole inspection 결과

[로컬 receipt](./p5b2c-fresh-execution-inspection-local-20261029.json)의 corrected Quick40.217s/
application fullRace4.233s/docs0.306s/security13.163s가 PASS다. 기존1291source는 불변이며
새2file만 추가했다. 반복 resolution의 fresh operation/entry Run·acquisition/검사/Evidence,
whole graph batch·missing/foreign·entry/set denial과 false aggregate ALLOW·provider/store
original cause·ID/nil/zero/cancel·derived 거부를 확인했다. 첫 Quick257.574s는 새 test가 wheel을
sdist derivation source로 만든 invalid fixture였고 기존 Domain에서 service 진입 전에 거부됐다.
Valid sdist fixture로 고쳤으며 제품/Domain/검사 guard는 바꾸지 않았다. 최초 source/log는 보존했다.
B2c API 범위는 IMPLEMENTED/QUALIFIED=YES, actual P5 flow WIRED=NO다. Resolution 선택/
실제 adapter factory·selected bytes/Node proof/records/staging·Backend/CLI는 여전히 MISSING이다.

## P5-B2d — selected Node bytes의 실제 controlled reader, source 전 감사/지도/재검토

기준a4ce06b/tree3c143e4b/2906files와 runtime1293을 양쪽에서 확인했다. 단일 M13-004의
다음 gap이며 CI423 qualification은 여전히6dd에만 적용한다.

| 책임 / actual owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| inspection/npm Inspector.InspectArchive / inspect / archiveCapture | YES | YES(PNPM/Bun) | YES(existing suites) | YES(existing scopes) | 선택된 한 member bytes capture를 같은 bounded walk에서 결합 |
| openInventorySource / copyArchivePayload / capture.finish | YES | YES(existing) | YES(existing) | YES(existing) | 그대로 재사용; 새 tar/gzip parser나 전체 archive memory copy 불필요 |
| artifact/npm ExecutionEntry.Matches / BindNodeTarget | YES | NO(actual P5) | YES(local) | YES(value) | controlled actual source에서 측정한 body 전달 |
| Inspector.ReadNodeExecutionEntry | NO | NO | NO | NO | 아래 같은 원본/full inventory와 selected bytes reader |

수정 지도는 기존 inspection/npm/archive_inventory.go의 capture에 optional selectedPath/body를
더하고 execution_entry.go와 회귀를 추가하는 세 source다. 기존 parser/Inspect/InspectArchive의
기본 capture는 빈 selectedPath이며 report·위반·size/hash/원본 전체 읽기 의미는 바꾸지 않는다.
이미 구현된 기능을 대체/재작성하지 않고 기존 entry payload copy에 선택된 한 body의 writer만
결합한다. Default read의 writer는 기존 digest이며 선택된 payload에만 bounded buffer를 사용한다.

새 ReadNodeExecutionEntry는 기존 ExecutionEntry를 받고 같은 inspect/capture를 호출한다.
Regular controlled source·whole compressed size/SHA·tar/gzip/trailing·path/link/type/manifest·전체
member facts를 다시 확인하고 original entry.Matches인 경우만 기존 BindNodeTarget으로 넘긴다.
Target path를 Host filepath로 사용하거나 extract/execute하지 않는다. Bytes bound20MiB,
전체 decoded200MiB·entry1만개/manifest1MiB는 기존 owner 그대로다. 새 port/runtime/Policy는 없다.

두 번째 검토: archive 순서(선택 member가 manifest 앞/뒤), 빈 JS와 explicit command,
wrong interpreter·다른 member facts/manifest·원본 변조/source symlink·CRC/trailing·취소/zero를
검증한다. 원본 source가 맞아도 inventory를 거짓으로 만든 경우 proof를 반환하지 않는다.
이 proof는 현재 whole 검사/Policy·introduction rehash/실행 admission을 대신하지 않는다.
Canonical format/quick와 기존 소비자를 포함한 inspection/npm 전체 Race를 수행하며 actual
factory/staging/backend/CLI 연결은 이후 MISSING으로 유지한다.

### B2d named local controlled Node entry read 결과

[로컬 receipt](./p5b2d-controlled-node-entry-read-local-20261029.json)의 Quick279.401s/
inspection/npm fullRace17.241s/docs0.305s/security12.156s가 PASS다.
기존1292source는 그대로이며 archive capture의 optional selected writer 한 곳과 새2file이다.
기본 Inspect/InspectArchive report/whole source guards를 유지하고 실제 controlled read에서
manifest 순서·빈 JS·explicit command/alias, original 전체 inventory와 member bytes를 확인했다.
Selected/다른 member digest/size/mode·manifest·acquisition 치환과 CRC/trailing/source symlink·
wrong interpreter/options·nil/zero/cancel/cause를 거부한다. Extract/Host·Node 실행은 없다.
API 범위 IMPLEMENTED/QUALIFIED=YES이며 actual P5 command flow WIRED=NO다. 다음은 real
resolver/fresh whole inspection/Node/context/records/staging preparation 조립이며 Backend/CLI와
실제 hostile/final qualification은 여전히 MISSING이다. CI423에는 이 source가 없다.

## P5-B2e — preparation workflow의 owner 감사와 수정 지도

기준4ecf37b/tree d88cd820/2909files·runtime1295다. Root/기존 private의 exact6commit 공개
후보는 고정했고 별도 tmp clone에서만 다음 작업을 한다. 이전 승인은6dd까지이며 새 공개
push/CI는 별도 pending이다. 이 사본의 이후 source는6commit payload에 포함하지 않는다.

| 책임 / actual owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| NPMResolver.resolveDependencies / ResolveDependencies | YES | YES(install) | YES(CI423) | YES(existing) | 같은 본체의 target 없는 execution entry method |
| ExecutionInspectService.InspectResolved | YES | NO(actual command flow) | YES(local) | YES(API) | 준비 workflow에서 매번 호출 |
| Inspector.InspectArchive / ReadNodeExecutionEntry, npm selection/proof | YES | NO(actual P5 flow) | YES(local/shared inspector) | YES(API) | 현재 primary의 실제 source와 요청 command 조립 |
| bootstrap.newNodeExecutionContext / domain.ExecutionContext | YES | NO(actual flow) | YES(local) | YES(value) | concrete entry binder와 실제 private argv를 workflow에서 결합 |
| Generator.BuildExecution / LocalStaging.Stage | YES | YES(records API / existing storage) | YES(named local) | YES(API/existing) | 같은 요청의 current set/bundle/staged handoff |
| 실행 준비 request/workflow와 실제 adapter factory | NO | NO | NO | NO | 아래 비설치 준비 흐름; 실제 프로그램 실행은 별도 |

준비 흐름은 Application이 소유한다. 새 ExecutionPreparationRequest는 ref/optional explicit
command/private copied argv만 가진다. Domain의 기존 command/argv 검증을 preflight로 재사용해
invalid 입력을 어떤 resolver/entry I/O보다 먼저 거부한다. Runtime override/InstallTarget/Host
project·environment·stdin/TTY/output export를 요청에 추가하지 않는다.

Application이 현재 필요한 좁은 resolver/entry-binder/execution-record interfaces를 소유하며,
같은 변경에서 concrete NPM resolver·Node binder·Generator가 구현하고 실제 factory가 연결한다.
미래 port/package/job placeholder는 없다. 기존 Staging Port와 fresh inspection service를 그대로
사용한다. Resolve once→fresh whole inspect→actual selected Node proof→current request context
재결합→execution Manifest/SBOM→보관 전용 staging 순서다. Promotion/프로그램 실행은 없다.

Entry binder는 기존 Node context 조립을 호출하되 raw argv 없이 measured facts를 제공한다.
Application은 returned context의 승인/argv를 재사용하지 않고 실제 fresh operation/resolution/
VerifiedSet와 현재 request argv로 NewExecutionContext를 다시 호출한다. 그래서 오래된 primary
intake나 다른 command 치환은 거부되고 cache ALLOW/외부 context가 실행 권한이 되지 않는다.
Records provider는 실제 Generator이며 bundle의 완전한 immutable Domain set을 fresh set과
대조한다. Staging 결과의 manifest ID/handle도 같은 bundle이어야 한다. 원본 provider 오류와
완료된 검사 결과를 보존하고 모든 후속 경계에서 cancellation을 확인한다.

수정 지도: 기존 domain/execution_context.go의 command 검증을 동일 의미의 재사용 preflight로
추출하고 회귀를 추가한다. 기존 sandbox/npm_resolver.go에 target 없는 실행용 method를
추가하되 본체/parser/network/lifecycle는 그대로다. application/execution_prepare.go와 회귀,
bootstrap/node_execution_prepare.go와 회귀 및 resolver의 실행 진입점 회귀만 추가한다.
새 canonical runtime/pin·Policy/guard·기존 install/project 기능은 변경하지 않는다.

두 번째 검토: zero/invalid/bounded input은 I/O 전 거부, resolve는 요청당1회이고 반복 호출은
fresh inspection/Policy다. Pipeline의 provider/entry/records/stage 오류·wrong current set/member/
command/staging identity와 cancellation은 complete preparation을 만들지 않는다. 실제
concrete Node reader·canonical runtime/records/staging handoff와 기존 install resolver 소비자를
canonical format/quick 및 domain/application/bootstrap/evidence/sandbox 영향 Race로 검증한다.
준비 성공은 user program의 실행·stdout/exit/cancel/lifecycle qualification이나 CLI 지원이 아니다.

B2e source 전 최종 대조: command preflight는 기존 domain 생성자의214bytes/UTF-8/text/path
규칙 그대로이며 explicit command에만 I/O 전에 적용한다. 생략은 기존 exact bin 선택으로만
확정한다. 새 private request는 argv clone/읽기 copy와 일반 formatter redaction을 제공한다.
Context binder의 반환값은 measured entry facts만 소비하고 operation/set/resolution/argv는
현재 값으로 다시 구성한다. Records provider의 bundle set은 immutable Domain 값의 정확한
동등 비교로 current set과 대조하며 새로운 authorization/hash/cache identity는 만들지 않는다.

실제 factory의 입력 재검토: 일반 Inspection interface를 그대로 받으면 static-only 조립도
가능하므로 concrete Backend를 받고 기존 DynamicInspector와 static+dynamic CompositeInspector를
factory 안에서 조립한다. 기존 Policy/Composite 생성자 계약을 바꾸지 않는다. 별도의 DATA-only
fixture 시험은 실제 Node 실행/production dynamic qualification으로 표시하지 않는다.

B2e 첫 Quick105.434s는 private field-only request를 기본 json.Marshal에 직접 넘긴 새
privacy test의 SA9005다. Source/최초 로그를 보존했다. Request는 실행용 private 메모리 값이므로
explicit MarshalJSON이 generic private-request 오류로 직렬화를 거부하게 하고 회귀도 그 계약을
검증한다. Raw argv/log/Evidence 비저장 계약을 유지하며 scanner rule을 끄지 않는다.

### B2e named local preparation / DATA storage handoff 결과

[로컬 receipt](./p5b2e-execution-preparation-local-20261029.json)의 corrected Quick229.702s/
영향5package fullRace39.718s/docs0.271s/security16.332s가 PASS다.
기존1292source는 그대로이며 command preflight/회귀와 target 없는 resolver entry만 기존 owner를
확장했다. 새5file은 preparation workflow·실제 Node adapter 조립/회귀다. Invalid input 전 I/O,
매 요청 fresh 검사·cached intake/records·다른 command/staging receipt·원본 provider/취소 cause와
private argv/formatter·JSON 거부를 확인했다. 실제 archive/SRI/static/Evidence/Node proof/records/
staging 반복 handoff와 변조/다른 interpreter 거부는 DATA-only/M1 fixture다. Production M3 dynamic/
live resolver·사용자 프로그램 실행으로 확대하지 않는다. 실제 factory는 Backend로 기존 static+
dynamic Composite를 조립하며 준비 workflow 내부 WIRED=YES, actual command/backend/CLI는NO다.
첫 SA9005 source/log를 보존했고 raw argv/hash/output은 records에 없다. 다음은 실제 준비
qualification과 user-run transport/lifecycle이며 full P5·CLI/final은 계속 MISSING이다.

## P5-B2f — production preparation 실제 qualification 전 감사/지도/두 번째 검토

기준a6cc420/tree d4db9063/2916files와 runtime1300을 Root/현재 private에서 확인했다.
CI424 source4ec는 고정돼 있고 timer2의16:40:03UTC/session40231을 실제 소비한다.

| 책임 / actual owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| ExecutionPreparationService/Node binder/Generator/LocalStaging | YES | YES(preparation) | YES(local API/DATA) | YES(narrow local) | production actual source/M3 dynamic preparation |
| newNPMExecutionPreparationService의 actual Backend→static+dynamic 조립 | YES | YES(code) | NO(actual positive) | NO(actual) | 기존 native helpers로 full positive qualification |
| NPMResolver·Backend·ObserverSupervisor·M3/M4·staging | YES | YES(existing) | YES(named existing/CI423) | YES(existing) | 새 prepared context/Node proof/storage 소비자 연결 actual |
| 사용자 프로그램 backend/CLI | NO(P5) | NO | NO | NO | 이번 시험 범위 밖; 그대로 missing |

[fixture provenance](./p5b2f-npm-fixture-source-20261029.json)는 공식 npm의 TypeScript5.9.3
metadata3,904B/SHA80f0d5d2와 bin tsc/tsserver·runtime dependency 부재·dist SRI를 확인한다.
Host에서 metadata DATA만 읽었고 artifact를 실행하지 않았다. 새 SDK/runtime pin이 아닌
기존 public npm source의 고정 시험 subject다. Actual resolver는 live지만 exact version/SRI를
대조한다. 원래42input/ZIP과 같은 frozen graph replay라고 표시하지 않는다.

수정 지도는 bootstrap/node_execution_prepare_integration_test.go의 새 realcorpus parent 한 개다.
Production 구현을 다시 쓰지 않는다. 기존 hosttool/System·ObserverSupervisor·typed NPM resolver/
Backend와 실제 factory/M3/M4/Evidence/staging을 직접 조립한다. Explicit tsc command와 private
argv를 준비하고 exact1subject/현재 required checks·Policy·Node entry/runtime·records/staging과
반복 요청의 fresh operation/Run/manifest를 확인한다. 사용자 프로그램 실행/Host install은 없다.

두 번째 검토: test marker/exit0는 안전성 authority가 아니다. Trusted Go assertions가 current
set/check/Evidence/동일 bytes를 검사하고 Canonical cleanup/helper·Host/Docker·source/CPP·원래
42/ZIP 전후 보존을 독립 확인한다. Existing4m context/5m test bound 안에서 준비만 수행한다.
Canonical format/quick와 exact CI flags의 tagged bootstrap binary build 뒤 같은 no-overlay binary로
actual parent를 수행한다. 최초 실패를 보존하고 production guard/예산/source pin을 완화하지 않는다.
실제 통과 전 production preparation QUALIFIED/ACCEPTANCE_CLOSED는NO이며 runtime/output/
exit/cancel/CLI와 full final profiles는 계속 MISSING이다.

### B2f actual 성공 뒤 CI 연결 전 재감사

No-overlay actual30.930s/동일 parent28.93s는 production factory/M3 dynamic/현재 Evidence4·
required3/Policy와 bin/tsc·같은 public SRI/SHA10e108c9의 fresh 준비2회/staging이 PASS다.
Helperwait0/confirmed stop·원래42/ZIP·source1301/CPP·Host/Docker 전후 불변을 확인했다.
첫 euid preflight는 namespace/제품 진입 전이며 기존 sudo 경로로 준비했다. User program은
실행하지 않았다. Local production preparation 범위만 검증했고 hosted 연결은 아직NO다.

CI existing gvisor-integration job과 Required·tagged bootstrap compile/client/cleanup은
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES(existing scopes)다. 새 preparation parent는
IMPLEMENTED/QUALIFIED=YES(local actual), WIRED/ACCEPTANCE_CLOSED=NO(hosted)다. MISSING은
기존 job에서 정확한 tagged parent를 호출하고 noSKIP/empty/duplicate/failure를 거부하는 연결이다.

추가 수정 지도는 .github/workflows/heliopause-ci.yml의 같은 Linux job 안의 새 qualification step
한 개다. 기존 compiled tagged bootstrap client를 설치하고 기존 initialize/helper/EXIT cleanup을
재사용한다. Job/Required/timeout/SDK/CPP/coverage를 새로 만들거나 바꾸지 않는다. Callback/stdout은
프로그램 안전성 authority가 아니며 trusted Go framework의 parent PASS1/noSKIP만 dispatch
완료 증거로 검사한다. 명령 오류/로그 출력 오류는 최초 exit를 보존한다. Canonical workflow와
실제 Bash dispatch harness의 empty/skip/child/duplicate/exit negative를 확인한 뒤 grouped 후보에
넣는다. 진행 중 CI424는4ec 그대로 두고 이후 user-approved autonomous push에서 qualification한다.

### B2f named local production preparation / hosted dispatch 연결 결과

[로컬 receipt](./p5b2f-production-preparation-local-20261029.json)의 no-overlay actual30.930s는
같은 tagged Go binary/CPP128/source1301로 parent28.93s PASS/noSKIP다. TypeScript5.9.3의
exact SRI/SHA10e108c9·명시 tsc/bin/tsc·현재 M3 required3/Evidence4/Policy와 fresh 준비2회/
새 operation/Run/manifest·staging을 확인했다. User program/Host install은 없다. Host/Docker·
원래42/ZIP·source/CPP 불변과 helperwait0/confirmedstop/부재를 확인했고 Node user-tmp
실제 빈 상태도 별도 readback했다. 원래 operator의 다른 fixture prefix 검사를 Node 증거로
전용하지 않았다. 첫 non-root preflight는 제품 미도달이며 sudo actual의 원본을 보존했다.

Workflow 기존 Linux job 안의 새 preparation step은 같은 tagged client/cleanup을 재사용한다.
Canonical workflow·post-wiring Quick81.733s와 actual YAML Bash harness8case가 PASS다.
Empty/skip/child/duplicate/failure와 로그 실패·최초 exit 보존을 검증했다. Local production
preparation만 QUALIFIED이며 새 hosted step/UID1001과 user-run/output/exit/CLI는 미완료다.
진행 중 CI424 source4ec를 새 push로 취소하지 않고 이후 grouped source로 hosted 검증한다.


## P5-C1 — 분리된 유한 Host transport의 구현 전 감사/지도/두 번째 검토

기준98047df/tree2dbcd352/2919tracked·1301runtime이다. 단일 M13-004에서 실제 user-run의
첫 missing transport만 진행한다. CI424는 source4ec/별도 Host이며 진행 중 push하지 않는다.

| 실제 owner / 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| hosttool.Executor.command/tool/minimalEnvironment | YES | YES(existing) | YES(existing) | YES(existing) | 없음; 새 transport도 동일 검증을 호출 |
| RunOutput/RunInput/RunDiscard/RunBounded | YES | YES(existing) | YES(existing) | YES(existing scopes) | 기존 의미를 유지; 별도 stdout/stderr 유한 capture 없음 |
| admissionAwareCommandRunner/observedDirectExecExit | YES | YES(inspection) | YES(named) | YES(existing) | 새 user-run 소비자는 별도 연결; Host exit를 guest signal로 해석 금지 |
| PNPMInspectionPreparer/executeNPMInstanceWithBunRuntime/cleanup | YES | YES(inspection) | YES(named) | YES(existing) | execution context의 fresh full graph/runtime/별도 lifecycle 연결 |
| 분리된 bounded capture | NO | NO | NO | NO | 아래 concrete Host-process transport와 영향 회귀 |
| 실제 Node user-run/backend/CLI/output rendering | NO | NO | NO | NO | 이 좁은 transport 완료 뒤 계속 수행 |

수정 지도는 internal/hosttool의 새 execution_output.go와 test 두 파일이다. Executor에
RunCaptured를 추가하고 기존 command 검증·minimal environment·closed stdin을 재사용한다.
각 stdout/stderr는 최대1MiB의 독립 prefix만 private memory에 보유한다. 초과 즉시 private
command context를 취소하고 명시 overflow 오류를 보존한다. 정확히 한도인 정상 출력은 허용한다.
Host client exit/wait의 원본 error와 caller cancellation cause를 함께 보존하며 pipe drain은
WaitDelay2s로 유한하게 한다. 이는 guest/container/descendant 정리 완료 증명이 아니다.

Capture 값은 private bytes와 stream별 overflow만 노출하며 getter는 copy다. 일반 formatter는
내용을 표시하지 않고 JSON 직렬화를 거부한다. Raw bytes를 로그/Evidence에 저장하지 않는다.
Terminal sanitization·guest exit/signal·admission·observer·container/child/helper cleanup은
기존 관련 owner에 다음 단계에서 연결한다. 기존16KiB merged diagnostic 소비자는 불변이다.

두 번째 검토: 새 production tool 등록이나 PATH/shell fallback을 만들지 않는다. 회귀의 shell은
verifyExecutable로 실제 root-owned fixture 신원을 검증한 test-only registry에서만 사용한다.
분리된 binary stdout/stderr·exact limit/overflow·nonzero original ExitError·deadline/custom cause·
closed stdin/minimal environment·unavailable/changed identity·private formatting/copy를 확인한다.
Inherited pipe의 유한 wait도 실제 controlled child로 검사하고 test-owned child를 회수한다.
Canonical format/quick와 hosttool full Race, docs/security로 검증한다. 성공 범위는 Host transport이며
artifact 프로그램 실행 또는 전체 P5 qualification으로 기록하지 않는다.


### C1 named local transport 검증

[Receipt](./p5c1-bounded-host-output-local-20261029.json)은 canonical checks와 전체 Hosttool Race를
연결한다. 기존1301runtime source는 그대로이며 새2file만 추가했다. 분리된 binary output,
각 exact1MiB/overflow·원래 nonzero/signal·deadline/custom cause·inherited pipe2s bound와
독립 child의 명시 test cleanup, private copy/formatter/JSON 및 기존 trusted identity/env/stdin
경계를 확인했다. 실제 P5 user-run/admission/guest completion/terminal rendering/전체 lifecycle은
아직 미완료이며 same M13-004의 다음 missing으로 계속한다. CI424 source4ec에는 이 소스가 없다.
