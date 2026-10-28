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
