# P1c C2 — native UV producer와 독립 data reader

기준 e3665ea. M13-004/P1만 IN_PROGRESS다. [C1c query 로컬 검증](./p1c-uv-query-local-20261022.json)은
고정 UV/Python의 실제 startup/query와 complete trace를 확인했다. CI372는 별도 확인 중이다.
이 단계는 public PyPI의 단일 요청 native 선택/완전한 graph 생산이며 update/sync 전체와
CLI를 먼저 노출하지 않는다. 기존 Policy/staging/promotion을 대체하지 않는다.

Read: [P1c 지도](./p1c-uv-resolver-baseline-20261022.md),
[client 설계](../../../user-journey-cli-ia/09-package-client-quarantine.md),
[계획](../../20-package-client-quarantine-plan.md),
[Artifact Port](../../../domain-model/07-contract-artifact-port.md),
[Dependency/Verified Set](../../../domain-model/05-dependency-verified-set.md),
[trusted tooling](../../../threat-model/04-trusted-tooling-and-evidence.md).

## Baseline audit

| 책임 / 기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| UV native/volume/admission/query — sandbox/uv_runtime*.go, uv_tool_supply.go, observer.cc | YES | YES(native/CI source) | YES(local query only) | NO(P1) | production typed-network 선택·data transport |
| native pylock/선택 WHEEL/whole archive·독립 release metadata — artifact/pypi/uv_lock.go, uv_release_metadata.go | YES | NO(production) | YES(scoped parser/corpus) | NO | actual producer·frozen controls 연결 |
| pip resolver setup/trace/cleanup — sandbox/pypi_resolver.go resolveDependencies | YES | YES(pip) | YES(existing scope) | YES(existing scope) | 같은 lifecycle의 UV 독립 metadata reader 재사용 |
| stream input/output·one-shot 승인 — direct_exec_admission.go, boundary_helper.go | YES | YES | YES(existing scope) | YES(existing scope) | UV compile stdin/고정 control file tar 사용 |
| installed tree bounded tar reader — pypi_closure_installation.go | YES | YES(existing closure) | YES(existing scope) | YES(existing scope) | UV 선택용 METADATA/WHEEL collector; 인증된 RECORD 검증기로 위장 금지 |
| production UV resolver/frozen graph — 새 sandbox/uv_resolver*.go | NO | NO | NO | NO | 아래 native 형식 확인·구현·실제 normal/negative/cleanup |

## 수정 위치와 방법

1. Exact0.13.4 compile은 stdin requirement와 --format pylock.toml/고정 output file을
   지원하는지 native pin에서 확인한다. Existing boundaryInputExecArguments와
   admission-aware RunInput이 docker exec -i를 이미 지원하므로 shell·Python writer나
   새로운 generic I/O port를 만들지 않는다. User spec은 기존 pypiRequirement의 typed
   grammar로 먼저 검증하고 stdin data만 전달한다.
2. pylock을 native sync/install에서 정확히 소비하는 command를 먼저 확인한다. Native
   JSON delta로 complete graph를 만들거나 solver를 pip로 대체하지 않는다. Wheel-only/
   no-build/no-config/no-managed-download·fixed interpreter/target·copy mode와 source를
   고정한다. --no-build/--only-binary 충돌은 이미 반증됐으므로 재도입하지 않는다.
3. UV runtime/session sibling은 기존 prepareUVTool·named root volume·startTrace/
   awaitMountAnchors·admissionAwareRunner와 typed ResolverNetworkPolicy를 재사용한다.
   NewUVResolver의 constructor/단일 ResolveDependencies만 실제 소비자로 추가한다.
   Unsupported source/stream capability는 network/target mutation 전에 거부한다.
4. 고정 pylock/control file은 existing streamingDockerOutput의 docker cp tar를 bounded
   data로 읽는다. 실행하거나 Host에 tar를 풀지 않는다. Native /haa-site의 METADATA/WHEEL은
   경로·중복·타입·전체 entry/byte 한도를 검사한 stream collector에서만 읽는다. Artifact
   metadata parsing은 artifact/pypi의 기존 RFC822/WHEEL parser를 재사용하고 선택 snapshot을
   기존 SelectInstalledWheels로 연결한다. 코드·INSTALLER·stdout 자체는 authority가 아니다.
   실제 native 조사에서 guest-only /tmp는 docker cp로 보이지 않았다. 출력은 기존
   closureVolume의 identity/attachment 검증을 재사용한 UV 전용 named tmpfs /haa-site에
   둔다. pylock.toml과 site/ 하위만 수집하며 runtime executable volume과 분리한다.
   UV-only expected mount topology를 함께 수정·검증하고 기존 Python profile은 유지한다.
5. 독립 exact-release JSON과 선택 archive 획득은 기존 isolated Python data-reader 역할을
   재사용한다. UV query child에 arbitrary Python/network 권한을 주거나 Intake.Resolve의
   isolated-resolution 경계를 Host HTTP fallback으로 바꾸지 않는다. 기존 pip resolver의
   공통 setup/trace/cleanup 구간만 두 실제 소비자용 private helper로 추출할 수 있다.
   기존 parser/solver·policy 의미·context/cleanup 순서와 오류는 유지하고 pip/PyTorch
   정상/실패·취소·cleanup 회귀를 검사한다. 전체 resolver 엔진 재작성은 하지 않는다.
6. 원본 archive는 private bounded 임시 저장소의 stable ReaderAt으로 BindWheels에 전달한다.
   모든 URL/size/SHA/RECORD/Requires-Dist와 독립 release metadata를 검증해 기존 단일-root
   graph를 만든다. Raw archive/temp install은 production target으로 반입하지 않는다.
   임시 자료는 마지막 reader 종료 뒤 제거한다. cleanup 불확실/trace incomplete이면
   graph/result를 비워 원래 오류와 함께 반환한다.
7. 반환 runtime/control은 solver UV와 실제 반출 installer identity를 정확히 구분한다.
   기존 pip report를 UV 결과로 위장하거나 broad identity prefix를 허용하지 않는다.
   Promotion control adapter·upgrade retention·다중-root/empty sync·CLI는 다음 acceptance
   gap이며 이 단계에서 placeholder나 지원 문구를 추가하지 않는다.

## native 형식 조사와 검증

먼저 동일 pinned Python/UV bytes를 owned read-only/runc/network-none/UID1000 fixture에서
사용한다. 자체 inert wheel은 데이터로만 전달하며 import·entry point 실행은 하지 않는다.
Compile stdin→pylock→native materialization을 확인하고 이 성공을 public registry/gVisor/
Policy qualification으로 확대하지 않는다. Container/volume/derived image는 owner 확인 뒤
no-prune로 제거한다. 이번 runc 형식 조사의 실제 snapshot은 Host file/image/container/bridge이며
service 증거로 확대하지 않는다. Production native session은 기존 service 불변 확인도
유지한다. 완료 data는 삭제한다.

구현 검증은 canonical format/Quick/security/docs/workflow, actual public 작은 wheel/복수
의존성 graph와 고정 replay·metadata/URL/hash/size/RECORD 불일치·unknown format·stream
과다/중단·parent/trace 실패·취소/cleanup을 연결한다. Shared lifecycle 변경 시 기존 pip/
PyTorch 및 필요 CPU/CUDA 소비자를 누락하지 않는다. 한도/timeout/필수 coverage를 늘리거나
complete trace/Policy를 작은 synthetic 성공으로 대체하지 않는다.

## 두 번째 검토

기존 구현을 버리지 않고 현재 실제 두 소비자의 같은 lifecycle/data 책임만 공유한다.
Native stdin 지원은 이미 existing admission parser에 있으며 새 shell 전달면을 만들 필요가
없다. Installed metadata는 selection hint이고 전체 archive/독립 registry 검증을 생략하지
않는다. 실제 명령·snapshot 형식은 아래 조사 뒤 확정한다. 미검증 pylock 소비 방식이나
metadata collector를 구현으로 가정하지 않는다. Production CLI/target mutation·update/sync
완료를 주장하지 않으며 P2로 이동하지 않는다.

## C2 output 부분 검증

[로컬 영수증](./p1c-uv-output-local-20261022.json)은 exact native0.13.4의 stdin compile,
pylock sync/list와 named tmpfs control transport를 확인한다. 첫 guest-only /tmp의 수집
실패와 corrected volume 성공은 다른 입력으로 보존한다. 이 wheel은 inert local fixture이며
public source/parser 승인이나 gVisor/Policy 결과가 아니다. 실제 METADATA/WHEEL은 uid/gid1000,
0644이며 INSTALLER는 uv였다. INSTALLER는 판정에 사용하지 않는다.

새 uv_native_output.go는 제한된 tar에서 pylock.toml/site/만 읽고 경로·타입·중복·소유권·
모드·파일/aggregate byte·entry 한도와 producer 종료/취소를 검사한다. Host 추출·artifact
import는 없다. uv_installed_controls.go는 기존 RFC822/WHEEL grammar와 SelectInstalledWheels를
연결하고 전체 archive BindWheels 검증을 유지한다. Canonical5·관련 Race는 PASS다.
Production resolver wiring/actual gVisor output mount·독립 archive reader는 계속 MISSING이다.

## C2 named output 실제 mount와 compile first fault

Native Go fixture의 새 named output /haa-site는 startup4.22s/query4.79s와 complete
trace29/31을 통과했다. Host 전후 동일·container/volume/endpoint0·daemon waits[0,0]이다.
Compile은 OPEN_RESULT_CLASSIFICATION_IMAGE/OCI_ROOT/flags557056에서 중단됐다.
Executable hash는 /haa-runtime/uv와 일치하고 OTHER는 기존 UV의 공개 분류이므로
새로운 임의 executable/clone 역할 실패로 해석하지 않는다. Raw uv-c2-volume-native-*가
최초 입력·실패·완료한 정리를 소유한다.

Locator hash fa06db656ccd9f7f는 /usr/lib/os-release와 일치한다. 이 일치 자체는 실행
authority가 아니며 후보 경로 식별이다. 다음 후보는 기존 IsPinnedUVRuntimeRead의
exact admitted UV root/CONTROL·current generation·소비한 첫 실행·active/무대기 guard와
OCI root mount/flags557056을 유지한 채 이 경로만 data read로 검토한다. /etc·library
prefix·다른 executable·쓰기·다른 profile은 추가하지 않는다. 기존 latch matrix에 해당
path와 adjacent/alias/wrong mount·role·flags negative를 추가하고 fresh C++ 전체106 뒤
동일 native fixture로 실제 첫 거부가 사라지는지 확인한다. 부분 성공은 전체 compile/
sync/collector 완료나 production 지원으로 확대하지 않는다.

해당 후보의 fresh C++106은298.785s PASS이며 actual startup4.03s/query4.37s도
complete trace29/31이다. Native compile·sync는 모두0, trace35/4008B complete까지
진행했으므로 이 입력의 exact os-release read 거부는 해소됐다. 다음 실패는 수집기의
path/ownership/mode/count 공통 오류이며 아직 세부 필드를 확정하지 않았다. Raw
uv-compile-osrelease-*는 Host 전후 동일·container/volume/endpoint0·waits[0,0]까지
보존한다. 기존 기준을 바꾸기 전에 bounded scalar 진단으로 필드를 구분한다. 새로운
collector 실패를 C++ 역할 확장 실패나 전체 설치 검역 성공으로 바꾸지 않는다.

Bounded 진단 후보의 canonical5 뒤 actual은 entries4/uid1000/gid1000/mode0666/
size0/중복false/정상경로·non-root/non-pylock에서 거부됐다. Owner나 observer 실패가
아니며 native compile/sync0와 trace35 complete는 유지된다. 다음 좁은 후보는 native UV의
보조 잠금 파일로 예상되는 site/.lock 하나에 한해 regular/size0/mode0666을 수집 후
버리는 것이다. 실제 다음 입력으로 이 경로를 확인한다. 이 파일의 내용·모드·존재는
selection·Policy·completion에 사용하지 않는다. /haa-site는 attested private named tmpfs,
uid1000/mode0700·단일 확인된 attachment이며 artifact import는 없다. 다른 writable 파일,
빈 파일의 다른 경로/타입·nonempty lock·특수 mode는 기존 거부를 유지한다. 인증된
closure/promotion manifest의 mode 규칙을 이 보조 데이터 예외로 변경하지 않는다.

## C2 native output 로컬 종료와 다음 gap

[최종 영수증](./p1c-uv-native-output-local-20261022.json)의 actual startup4.10s/query4.25s/
compile-sync-collection5.07s는 모두 PASS, complete trace29/31/35다. 실제 보조 lock이
site/.lock임을 좁은 후보가 확인했고 selection/Policy/completion 자료로 쓰지 않았다.
원본 local-source pylock은 production parser가 계속 거부한다. Canonical5·관련 Race7.919s와
새 세 번째 native case의 CI selector/누락 거부를 검사했다. C++106의 새 후보는 exact
os-release data read만 추가하며 generic Python/clone/network·기존 profile은 확장하지 않는다.

Output volume/collector의 현재 감사: IMPLEMENTED=YES, WIRED=YES(native fixture·CI source),
QUALIFIED=YES(local native only), ACCEPTANCE_CLOSED=NO(P1 전체). 네 완료 native workspace
각6,833항목의 약2.75GB와 superseded observer/일회성26파일3,168,313B를 Root 참조·query
errors0 확인 뒤 제거했다. 현재 qualified observer·source·고정 SDK/quality는 다음 소비자용이다.

다음은 단일-root public PyPI native 선택과 isolated 독립 release JSON/archive data reader의
실제 연결이다. 기존 pip/PyTorch setup/trace/cleanup의 필요한 구간만 공유하며 Policy/staging/
promotion을 새 엔진으로 대체하지 않는다. 이 inert local wheel 시험을 public graph/설치
검역으로 확대하지 않으며 upgrade/sync·CLI와 P2..P7은 계속 MISSING/NOT_STARTED다.

## C2 public producer의 다음 정확한 수정 지점

기존 pypi_resolver.go resolveDependencies의 request 생성 전 lifecycle만 private
withPythonResolverSession으로 추출한다. Runtime probe→trusted endpoint/policy prepare→
create/start trace→start/anchors→고정 runtime 검증→project 준비와 defer의 remove→collect→
policy close, 최초 cause/cleanup 실패의 결과 삭제를 그대로 보존한다. 기존 pip/PyTorch body는
그 안에서 실행하며 request 검증과 resolve timeout 생성 위치·예산도 옮기지 않는다.
Operation callback은 내부의 두 실제 소비자만 사용하며 Core port나 공개 generic runner를
추가하지 않는다. UV의 독립 reader는 기존 public PyPI Python 역할의 같은 typed-network
lifecycle 안에서 exact-release JSON과 선택 wheel bytes만 읽고, UV profile의 query child를
범용 reader로 승격하지 않는다.

두 번째 검토: 기존 구현은 IMPLEMENTED/WIRED=YES이며 폐기·대체하지 않는다. 같은
lifecycle을 그대로 공유하는 좁은 수정이다. 기존 PyPI/PyTorch 정상·실패·취소·cleanup 및
source ownership 회귀와 canonical profile을 먼저 확인한다. Native UV 선택/전체 archive
binding을 완성하기 전 constructor/CLI 지원을 노출하거나 placeholder endpoint를 만들지 않는다.

Private bindUVNativeSelection은 native pylock/controls·stream capability·전체 archive byte
예산을 network 전에 검사한다. 독립 release JSON의 exact URL·filename·SHA/size/yanked가
일치한 뒤에만 고정 Python data-reader script로 wheel bytes를 읽고, private0700 workspace의
0600 file/명시적인 Write-only exact-size bound를 거쳐 기존 BindWheels로 연결한다. Redirect/
proxy를 받지 않고 JSON4MiB·선택 archive exact size/EOF·TLS 기본 검증을 유지한다. Reader
deadline은 setup 뒤 기존2m이며 상위 context를 상속한다. Archive FD는 마지막 BindWheels
reader 뒤 닫고 workspace를 제거한다. Subprocess cause는 unwrap으로 보존하되 raw text는
반환 메시지에 노출하지 않는다. Runtime identity는 solver UV와 Python target을 명시하며
pip report/installer identity를 위장하지 않는다. Production native 호출자는 아직 미연결이다.

고정 replay의 whole archive/metadata 불일치·short/extra stream·실패/취소·cleanup/trace
실패와 workspace 삭제, network 전 거부, writer fast-path/sticky bound를 검사한다. Replay
runner 성공은 실제 Python HTTP transport·public native resolver qualification이 아니다.

[독립 reader 로컬 영수증](./p1c-uv-reader-local-20261022.json)의 canonical5·관련 Race10.055s는
PASS다. 기존 session 추출도 canonical5/Race10.199s로 따로 확인했다. 정상 replay는 완전
archive/RECORD/graph binding을 통과했고 metadata identity/hash/Requires-Python·byte bound,
archive hash/short/extra·command/cancel/cleanup/trace 부정 입력은 graph를 비웠다. 첫 정상
fixture의 Requires-Python 불일치는 합성 자료에서만 교정하고 부정 회귀로 보존했다.
실제 HTTP transport와 public UV native caller는 계속 MISSING이며 source9b0da57의 CI373은
별도 [최종 영수증](./p1c-native-output-ci373-20261022.json)에 연결했다. Same-tree native output
세 시험·기존 Linux/CPU287.67s/정리는 PASS, macOS default6m/Required는 FAIL이다.
이후 reader/cache 후보의 원격 성공이나 전체 CI 성공으로 쓰지 않는다.

## Public native caller 연결 후보 — 검증 전

uv_resolver.go는 기존 typed ports의 NewUVResolver/Linux factory에서 실제 native 선택과
bindUVNativeSelection을 순차 연결한다. Source/request/stream capability·runtime pin과
native tool bytes를 먼저 검사한다. Native compile stdin과 고정 pylock/sync 명령만 쓰고
solver를 pip로 대체하지 않는다. Profile/volume/trace/admission은 C2 검증한 기존 구현을
재사용하고 parent context/기존2m·128MiB output budget을 유지한다. Resource exhaustion은
검역 실패이며 source budget을 늘려 정상으로 만들지 않는다.

Container는 random session name/owner label과 성공한 absence query로 생성하고, 실제
ID/name/image reference/label을 실행 전에 확인한다. 생성 stdout이 불확실해도 이름으로
조회하되 다른 ID/owner·query 실패/모호한 inventory는 채택·삭제하지 않는다. Remove→trace
collect→각 volume identity/attachment 확인·제거→policy close 뒤에만 unapproved native
출력을 반환한다. Cleanup/trace 불확실이면 출력을 비워 독립 reader/graph 단계로 가지 않는다.
독립 Python session도 정리가 끝나야 graph가 반환된다. No Host resolver fallback·새 Core port나
새 CLI 지원 문구는 없다. 현재 caller IMPLEMENTED=YES(후보), WIRED=NO(app/guarded client),
QUALIFIED=NO, ACCEPTANCE_CLOSED=NO이며 실제 public/HTTP qualification은 다음 acceptance다.

Caller의 두 번째 검토에서 sync 전에 pylock의 source/형식/graph·declared byte budget을
검사할 지점을 확인했다. Compile 직후 fixed /haa-site/pylock.toml 한 파일을 기존 bounded
Docker tar transport로 읽고 ParseUVLock을 먼저 적용한다. 허용되지 않는 source/대안 수·
버전·aggregate budget이면 sync/추가 wheel 다운로드를 하지 않는다. Sync 뒤 전체 output의
lock bytes도 최초 snapshot과 같아야 한다. 같은 private transport를 두 실제 reader가 공유하고
lock-only의 owner/mode/type/한 파일·EOF/padding/byte bound와 producer failure를 검사한다.
Native metadata 탐색 자체는 기존 context/runtime/network 한도 안에 있으며 Policy를 대신하지 않는다.

추가 lock-only reader/동등성의 canonical5·관련 Race16.610s와 실제 network-none 세 fixture
(startup5.24s/query4.40s/compile-sync-collection5.04s·complete trace29/31/35)는 PASS다.
이는 transport와 snapshot의 실제 검증이며 public native/HTTP 성공은 아니다.
Public idna@3.11 단일 node와 requests@2.32.5의 다중 의존성은 기존 CI의 보호된 test-client
설치·policy helper 재바인딩 이후 실행한다. 일반 go-test 임시 binary로 caller 인증을 우회하지
않고, 첫 실패 입력 뒤 더 큰 graph를 반복하지 않는다. 조건 없는 기존 lifecycle step과
누락 거부 validator에 연결하며 새 job/권한을 만들지 않는다. 실제 결과는 다음 CI로 확인한다.

[Public caller 로컬 영수증](./p1c-public-caller-local-20261022.json)의 최종 canonical5는
PASS다. Actual lock transport/동등성의 세 fixture와 constructor/refusal/cleanup·reader/
기존 resolver의 Race16.610s를 연결했다. 완료 native workspace6,833항목688,406,687B는
Root 참조·query errors0 확인 뒤 제거했다. Public/HTTP entry는 기존 보호된 CI client에
연결했지만 아직 실제 실행 결과가 없으므로 qualification/CLI 지원을 닫지 않는다.
