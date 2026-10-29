# P5-C3d1 — selected Node signal·취소·독립 stream overflow 실제 검증

기준558d75a/treef5e4d9f9/2969tracked·1330runtime다. CI428은 이 source로 진행 중이며
첫20min timer01/session55237은00:17:29UTC(한국09:17:29)에 소비한다. 새 push로 취소하지
않고 같은 단일 M13-004의 독립 qualification gap만 수행한다. Read: lessons INDEX와
sandbox L03/L08/L12/L18·artifact L02·evidence L16·ci L05/L06, client09§3/plan20§4·5,
C3c2b actual receipt/owner map과 실제 아래 source를 대조했다.

| 실제 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Backend.ExecuteNodeExecution/nodeExecutionClientResult/closed Node profile | YES | YES(production parent) | YES(normal0/nonzero1 local) | YES(named foundation) | selected Node signal/취소/overflow actual |
| Hosttool.RunCaptured/CaptureExecutionOutput·independent1MiB/WaitDelay | YES | YES(same admitted Node runner) | YES(C1 and Node named normal) | YES(named transport) | actual Node stdout/stderr overflow와 guest cleanup 연결 |
| existing admission/target/outcome/lease/trace·readonly inputs/disposal | YES | YES | YES(8controller roots+Node0/1) | YES(named) | selected Node signal facts/caller cancellation 뒤 incomplete와 자원 정리 |
| defer ctx.Err/Cause join/current45s/cleanup background | YES | YES | NO(mid-program actual) | NO | 실제 body 이후 custom cause cancellation/no false Complete/no leaks |
| npm.Resolver.NewResolver controlled test seam/actual Acquire/IntegrityVerifier/static/dynamic/Policy/Evidence | YES | YES(existing tests/production owners) | YES(existing scopes) | YES(existing scopes) | bounded local scripted fixture에 동일 owner composition; fake approval/report/bundle 금지 |
| public application/CLI·output renderer | NO | NO | NO | NO | 후속; 이번 test-only unit에서 scaffold하지 않음 |

## 수정 지도와 두 번째 검토 — source 작성 전

새 Linux+realcorpus Bootstrap test 두 파일만 추가한다. Production code/CPP/고정 입력/권한/
role·일반FS/network/45s·1MiB bounds와 기존 parent는 그대로다. 기존 NewResolver의 명시적
controlled-test HTTP RoundTripper seam으로 한 deterministic npm fixture tarball/metadata만
제공한다. 요청 host/path/method를 제한하고 실제 네트워크는 사용하지 않는다. Production
NewPublicResolver/공급망 지원을 바꾸지 않는다. 실제 resolver parsing/Acquire의 measured
SHA256/SHA512와 canonical lock parser를 사용하고 current IntegrityVerifier·static/dynamic
composite·Evidence.Store·PolicyM3/M4·ExecutionInspectService로 실제 approved set을 얻는다.
단순 fixture-policy/CheckCompleted/VerifiedSet을 만들어 주입하지 않는다. Registry/native
solver 공급망은 CONTROLLED_TEST_FIXTURE/NOT_RUN이며 current 검역·Policy와 실제 runtime은
ACTUAL로 구분한다. 기존 nodeExecutionEntryBinder/newNodeExecutionContext와 materializer/
readonly backend를 그대로 호출한다. 가짜 VerifiedBundle/InstallTarget/SandboxResult는 없다.

첫 actual은6fresh cases다: literal argv vector, selected Node TERM/KILL 각각, body 준비
prefix가 관측된 뒤 caller custom-cause cancellation, stdout/stderr 각각1MiB 초과. Signal은
Host code/stdout가 아니라 기존 independent kernel outcome과 typed guest error/Complete를
검사한다. 취소는 underlying 동일 protected Executor의 CaptureExecutionOutput만 test wrapper로
위임하고 capture 진입 뒤 고정1s timer가 원래 caller context를 취소한다. 실제 반환된 private
prefix가 body 도달을 확인해야 한다. Prefix는 test oracle뿐이며 완료/권한이 아니다. Overflow는
원래 ErrExecutionOutputLimit identity/각 stream bound·incomplete를 요구한다. 각 case의
fresh supervisor/whole current approval·clean layout과 final helper/container/volume/endpoints0,
Host/source/고정42/ZIP/Docker 보존을 확인한다. 같은 failing input의 blind rerun은 하지 않는다.

두 번째 검토: fixture bytes/metadata/lock identities를 실제 검사·Policy까지 연결하며 fake
completed 검사를 넣지 않는다. Cancellation timer는 I/O 준비 전에 시작하지 않고 actual
Node capture 진입 뒤 시작한다. Returned prefix가 없으면 timing/body qualification FAIL이다.
Role/terminal authority는 기존 owner뿐이다. Test failfast와 최초 cause/원본/strict cleanup을
유지하며 future child/FS/network/CLI와 final profiles를 이6cases로 닫지 않는다. Source 전
상태는 이 unit IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다.

## Named local six-case actual 결과

[최종 receipt](./p5c3d1-node-boundary-local-20261029.json)는1332inputs(기존1330/production/CPP
불변+newtests2), canonical format2.571s/tagged vet11.191s/pinned Staticcheck13.927s/tagged build
3.176s/docs0.315s/security19.415s와 actual92.871s/parent90.87s의6fresh PASS를 연결한다.
Literal argv73B, TERM/KILL 각각kernel15/9와 stdout7B·typed guest error/Complete, body ready6B
뒤 custom cause 취소/incomplete, stdout/stderr 각각1048576B·다른stream0/원래 overflow cause와
incomplete를 확인했다. Current integrity/static/dynamic/Evidence/Policy는 각 case 실제 실행했고
공급망/native solver는 controlled fixture/NOT_RUN이다. Source/CPP/Host/42/ZIP/Docker 보존,
helperwait0/confirmedstop/owned resources·endpoints0/noSKIP다. 원본 source/check6/log14/assessment를
먼저 보존했다. 이 unit은 named local IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES지만
FS/network/child hostile·CLI/new hosted/final 범위로 확대하지 않는다. 다음은 C3d2 baseline이다.
