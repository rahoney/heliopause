# P4-L persistent UV tool / pipx baseline

Baseline은 42424fc(K3), tree8379f65b74e71cfa69001535dace2f7ded935fb9/2555 files다.
M13-004 하나만 IN_PROGRESS다. Read: Queue/plan20/client 설계09, 관련 PyPI resolver·
complete inventory/source proof·entrypoint planner·materializer/venv transaction,
D012/D013/D015/M8/M10, runtime maintenance§2·3·6·7, lessons INDEX와 artifact/sandbox/
ci/evidence 관련 항목을 확인했다. 기존 qualified 구현을 대체하지 않고 actual schema와
호출 의미를 먼저 확인한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Public UV/PyPI 선택·독립 archive/source·complete graph/current inspection/Policy/cache | YES | YES | YES(named P1/UV project) | YES(named lanes) | tool intent와 실제 고정 backend 의미 연결 |
| Wheel scripts/entrypoint 및 기존 venv ownership/원복 | YES | YES | YES(existing PyPI scopes) | YES(named scopes) | native tool receipt와 외부 노출 command ownership |
| UV0.13.4 native supply/runtime/admission | YES | YES(pip/project) | YES(named current scopes) | YES(named scopes) | tool install/upgrade의 actual argv/layout/runtime-role |
| pipx pinned backend·공급/metadata·CLI | NO | NO | NO | NO | exact source/proof/pyz census, native metadata/backend/runtime/schema·입출력 |
| Persistent tool root/venv/receipt/exposed scripts atomic publication | NO | NO | NO | NO | original complete guard/native ownership, retired/conflicting command·rollback |
| Public uv tool/pipx install·upgrade | NO | NO | NO | NO | 위 prerequisites와 actual current approval/public CLI |
| Temporary runners P5/P6와 final profiles P7 | NO | NO | NO | NO | 후속 명시 범위; 이번 성공으로 닫지 않음 |

## 조사 경계·수정 지도·두 번째 검토

먼저 고정 UV/native pipx의 실제 metadata·entrypoint와 Python lifecycle을 전용 guest에서
관측한다. 현재 venv adapter는 target Python version/layout을 읽고 기존 환경 파일만
변경한다. Tool venv 생성·interpreter relocation·exposed scripts는 이 완료를 승계할 수 없다.
Runtime provisioning과 global/ambient target은 초기 범위 밖이다. Existing pinned-runtime
target의 관계를 명시하고 tool root/venv/bin root를 함께 고정하는 설계를 actual 뒤 확정한다.
Host에서 candidate tool/Artifact를 실행하거나 native client를 다른 client alias로 바꾸지 않는다.

기존 parser/source binding·entrypoint planner·current whole inspection/cache를 재사용한다.
새 native receipt/intent reader는 unknown schema/source·이전 marker/ALLOW를 authority로
취급하지 않는다. Tool 이름과 모든 실제 command의 소유·충돌/retirement를 결합한다.
실제 native 선택·offline materialization이 어떤 interpreter/code를 실행하는지 확인하기 전에
이를 DATA-only 또는 trusted-control로 일반화하지 않는다. 특히 post-install metadata 조회와
venv startup의 역할·Artifact hook 가능성을 exact source/실제 관측으로 확인한다.

새 config/interface/CI job이나 지원 placeholder는 만들지 않는다. Actual schema가 고정된
현재 gap에만 adapter를 추가하고, 공통 venv/backup engine의 확장은 정확한 map과 기존
소비자 검증을 먼저 붙인다. Runtime/observer 권한·budget·Policy·완료 기준은 유지한다.
새 supply/admission이 필요하면 기존 단일 runtime lock/generator/maintenance chain에 연결하며
독립 version lock이나 ambient PATH backend를 도입하지 않는다.

## 공식 discovery — 지원/공급 qualification 아님

현재 [UV tool 문서](https://docs.astral.sh/uv/guides/tools/)는 persistent tool이 독립 환경과
노출 executable을 사용하며, package 하나가 여러 command를 제공할 수 있고 upgrade가
기존 constraints를 보존함을 설명한다. Pip 설치 alias로 이를 대체하지 않는다.

2026-10-26 read-only 공식 metadata에서
[pipx release1.18.4](https://github.com/pypa/pipx/releases/tag/1.18.4)와
[PyPI metadata](https://pypi.org/pypi/pipx/json)의 version1.18.4가 같고 Requires-Python은
>=3.11이다. Release의 pipx.pyz는727151B/SHA256
9273476832b2672f0799a84f69c72fcd5008e7a4dab3723a48fd0e25c3ddd19f다.
이는 discovery일 뿐 pin·호환성·trusted role·설치·실행 성공이 아니다. 이전 stable docs 두 URL은
404였고 API에서 현재 docs가 RST 구조임을 확인했다. Exact tag source·배포물·실제 동작은
이어 확인한다. Raw owner는 `docs/github-logs/package-client-quarantine-20261021/`의
`p4l-persistent-tool-official-discovery-v1-*`다.

원격 CI는 작은 helper마다 실행하지 않는다. Plan20§5의 grouped/end-to-end·최종 profile
경계를 유지하고, 다음 필수 원격 실행에는 exact run/head·실측 예상 시간과 실제 타이머를
연결해 결과/원본을 소비하면서 독립 후속 작업을 계속한다.

## L1 native schema와 inert parser 구현 지도

공식 exact source23개와 추가8개는 Git blob을 확인했고 pipx core16개는 배포 pyz의
동일 module bytes와 대조했다. Pyz는 shiv 구조이므로 mutable shiv cache를 trusted
code로 실행하지 않는다. 진단은 root-owned read-only로 추출한 동일 module census와
고정 Python -I -S loader를 사용했다. Production supply/admission은 아직 미구현이다.

고정 guest의 help v2는4.692s, UV install/upgrade layout v1 두 명령은 성공했다.
같은 v1의 pipx는 --no-build/--only-binary 충돌로 설치 전 거부됐다. 중복 only-binary만
제거한 affected pipx v2는6.460s에 두 명령 exit0이다. 두 layout의 container와 label-owned
7개 image를 각 leaf-first 제거했고 Host snapshot은 동일하다. 입력은 startup hook·의존성·
data script 없는 synthetic wheel1.0.0/1.1.0이며 노출 command 자체는 실행하지 않았다.
Public source, Artifact observation/current Policy, Host publication 증거로 확대하지 않는다.

UV native receipt는 schema version과 설치 version이 없고 단일 requirements/specifier,
explicit Python, entrypoints(name/install-path/from)를 기록한다. Upgrade는 >=1,<2를 보존한다.
Pipx metadata0.12는 explicit UV backend, main package/version, typed Path 및 app inventory를
기록하며 upgrade는 version constraint를 제거한다. Exact release의
parse_specifier_for_upgrade도 remove_version_specifiers=True다. Native 의미를 UV와 통일하지
않는다. Pipx metadata에는 external bin root가 없으므로 별도 held exposed-link inventory가
필수다. `fetch_info_in_venv`는 설치 후 venv Python -c를 실행하므로 arbitrary startup hook이
있는 package의 metadata 단계를 DATA-only로 취급하지 않는다. Runtime-role 수용은 남아 있다.

이번 변경은 existing pypi package의 tool_receipt/uv_tool_receipt/pipx_tool_receipt parser와
actual native controls 네 fixture, 의미 있는 parser/layout 부정 회귀다. 기존 bounded requirement,
normalization/단일 source 의미를 재사용한다. JSON duplicate/depth walker는 기존 UV wrapper의
한도4/오류/동작을 유지하고 pipx의 실제 깊이에만6을 적용한다. Unknown fields/schema/backend,
local/direct source·injected/dependency/man/completion exposure, unsafe/duplicate commands와
mismatched paths를 조기 거부한다. Parser는 filesystem IO/native 실행·공급 pin·승인·반입을
하지 않으며 stored native options/pip_args는 inert bytes일 뿐 실행 recipe로 노출하지 않는다.

두 번째 검토: receipt의 주장을 ownership/selection authority로 쓰지 않는다. UV의 없는 version을
만들지 않고 pipx의 외부 bin root를 receipt에서 추론하지 않는다. Native command source는 main
package만 허용하고 complete authenticated wheel/installed inventory/current Policy와의 결합,
실제 root/venv/exposed-link transaction은 후속 gap으로 유지한다. 기존 구현 대체는 없다.
원본은 raw owner의 p4l-persistent-tool-{static,exact-sources,exact-backend-sources,help-query,
layout-query} 및 p4l-native-receipt-fixture-provenance-v1.json을 따른다.

L1 named local 결과: canonical Quick221.564s/security13.252s/docs0.261s와
PyPI 전체 Race6.086s(test2.563s)가 PASS다. 실제 네 fixture·command 교체/constraint 의미와
부정 schema/source/backend/path/duplicate 검증이며 public tool 지원 acceptance는 NO다.
[최종 receipt](./p4l1-native-tool-receipt-local-20261026.json)가 정확한 검사 범위를 소유한다.
다음 L2는 whole authenticated wheel/entrypoint binding이며 supply/runtime/transaction/CLI는 남는다.

## L2 whole archive/source와 entrypoint binding 지도·재검토

L1 commit b3e4655/treef4b2d9774c7f6431766d382d1d6cc40aa6b0fff7/2565를 기준으로 감사한다.
기존 UVLock.BindWheels/bindUVSelectedWheels/uvArchiveCandidates·InspectWheel·CrossCheckReport·
validateUVCandidateConstraints·BuildLockedGraph가 whole archive/source/closure를 구현·검증했다.
새 gap은 이 검증과 실제 native tool receipt의 main package/version/command 집합을 연결하는
것이다. 기존 graph/source parser를 다시 구현하거나 새 source/domain identity를 만들지 않는다.

uvArchiveCandidates의 기존 본문은 private inspection collector에 그대로 두고 기존 호출자는
nil collector로 유지한다. 새 ToolReceipt.BindWheels만 같은 검사에서 나온 root entrypoint와
전체 installed surface를 소비한다. CrossCheckReport/constraints/BuildLockedGraph를 재사용해
public source·전체 graph를 확인한 뒤에만 결과를 반환한다. Native receipt와 frozen root의
project·version 제약, main wheel console entrypoints의 exact name set을 대조한다.

Pipx post-install Python은 별도 trusted DATA-only role로 일반화할 수 없다. 초기 bounded
metadata lane은 전체 authenticated closure에서 root .pth와 sitecustomize/usercustomize의
module/package/extension 목적지를 거부한다. File suffix만 실행 역할로 추정하지 않고 기존
installed location mapping·SchemeSite와 surface를 사용한다. 이 gate는 public runtime admission의
대체가 아니며 hook-bearing tool 지원을 닫지 않는다. No target/native 실행·Policy 변경은 없다.

두 번째 검토: orphan/누락 dependency·Simple 해시 불일치·전체 wheel 변조를 receipt로 덮지 않는다.
Dependency가 같은 이름의 command를 제공해도 main package entrypoint를 대체할 수 없다.
Collector는 immutable 검사 결과의 in-memory 소비에만 사용하며 기존 nil 경로의 본문·한도·
순서를 보존한다. Actual native source/runtime/Host transaction qualification은 여전히 별도다.

L2 named local 결과: canonical Quick196.061s/security12.985s/docs0.278s와 PyPI 전체
Race5.834s(test2.569s)가 PASS다. [최종 receipt](./p4l2-tool-wheel-binding-local-20261026.json)가
fixture/source/whole graph/command·startup gate의 정확한 범위를 소유한다. 기존 collector nil
본문 보존은 p4l2-wheel-collector-mechanical-reuse-v1.json으로 대조했다.

후속 runtime 정적 근거: exact UV interpreter.rs는 -I/-B/-c이며 -S를 사용하지 않는다고 명시한다.
Tool install/upgrade의 get_environment → PythonEnvironment::from_root → Interpreter::query 경로도
확인했다. 따라서 위 UV static graph 성공을 native query clearance로 전용하지 않는다. 기존 환경
조회 전 원본 whole inventory/startup surface와 runtime 역할을 별도 고정해야 한다. 원본
p4l3-uv-interpreter-exact-sources-v1 및 environment-exact-v1이 source hash를 보존한다.

## L3-A fresh UV native tool install 선택 지도·재검토

L2 commit623925f/tree8de443062ef05001dc477606cfda38a6c3e9e7fb/2568을 기준으로 기존
UV native session/network policy/runtime/observer/volume cleanup, independent archive reader가
IMPLEMENTED/WIRED/QUALIFIED=YES인 점을 확인했다. Tool install native producer/output census와
public whole graph/receipt 묶음은 NO다. 이 gap만 우선 구현하고 기존 target native upgrade query는
별도 original guard가 준비되기 전 실행하지 않는다. Public install/upgrade CLI acceptance는 NO다.

새 fresh install selector는 기존 UVResolver session을 사용한다. UV_TOOL_DIR/BIN_DIR는 고정 private
output 아래 controller 값으로만 전달하고 Host 환경·설정·PATH·runtime provisioning을 사용하지
않는다. Native uv tool install과 explicit pinned Python/no-build/no-config/no-managed-download/
no-sources/copy를 사용하며 compile-bytecode는 false다. Native receipt, exposed links, runtime aliases,
bounded site metadata 전체를 읽는다. Code/entrypoint는 실행하지 않고 native exit만으로 성공하지 않는다.

설치된 exact metadata의 전체 version set을 기존 bounded inventory reader로 읽고 UV pip compile의
pylock으로 내보낸다. 이는 native tool 선택을 다시 추론하는 resolver alias가 아니다. 모든 version을
freeze하고 export와 native whole installed controls를 대조하며 source/archive/closure/receipt command는
기존 L2와 independent reader로 확인한다. Export가 package/version/wheel 선택을 바꾸면 거부한다.
Control·runtime·intent/census digest를 typed resolution과 결합하며 current Policy/Host 승인으로 표시하지 않는다.

정적 runtime 경계: fresh private root에는 기존 tool 환경이 없으며 base Python만 조회한다. Exact
create_venv는 Rust로 생성 뒤 interpreter.with_virtualenv를 사용한다. sync의 bytecode compilation은
false이며 finalizer는 Rust SitePackages/entrypoint/receipt 작업이다. 원본은
p4l3-uv-tool-install-runtime-sources-v1/environment-creation-sources-v1을 따른다. Existing 환경의
get_environment/query는 -S가 없으므로 이번 fresh 경로와 구분한다. Post-install hook을 trusted
CONTROL로 실행하는 예외나 observer pin 변경은 없다. 실제 native 회귀로 이 경계를 확인해야 한다.

수정은 sandbox/uv_tool_output(전체 bounded tar census), uv_tool_resolver(fresh native producer와
기존 source binder), 동일 package의 default/actual 회귀다. 기존 uvNativeOutput·transport·session
본문을 바꾸지 않고 sibling adapter로 소비한다. Unknown path/type/link/mode/extra/truncated producer,
바뀐 receipt/metadata/lock/누락 source에 결과를 남기지 않는다. 두 번째 검토에서도 original guard,
whole approval/cache/transaction/CLI·pipx production supply는 MISSING으로 유지한다.

L3 first actual: v1 local Quick221.774s/security13.479s/docs0.249s·전체 PyPI/Sandbox
Race24.579s는 PASS였으나 actual6.515s 첫 tool request3.91s가 ATTRIBUTION_FAILURE로 실패했다.
새 Docker exec --env prefix가 directExecOrigin grammar를 벗어난 것이 원인이다. 원본은
p4l3-first-native-admission-assessment에 보존했다. Tool 전용 create 환경으로 옮겨 기존 direct
exec/nonce/observer를 유지하고 nil/false 기존 session 본문을 대조한다. 기존 public UV actual은
failfast로 미도달이며 교정 후 새 actual 전에는 해결·native tool 지원을 주장하지 않는다.

### L3 native runtime directory boundary baseline

v2 direct-exec 교정은 canonical Quick106.295/security12.656/docs0.264·전체 Race23.383s를
통과했고 실제 nonce stage도 통과했다. 이어 actual7.182s/test4.66s가 UV의 OCI-root directory
read에서 STREAM_FAULT였다. Pinned Python image의 data-only catalogue6611개를 중지 상태
container(실행0)에서 읽어 locator2eaf681357327a9b가 /usr/local/sbin directory임을 확인했다.
Raw first export의 image 목록 순서 비교 오류와 dot-path 정규화는 별도 assessment로 보존하고
수정 catalogue·정렬 Host 전후 동일을 확인했다. Native 선택/Policy/반입 성공은 아니다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Pinned UV consumed direct CONTROL/runtime readonly read | YES | YES | YES(old UV lanes) | YES(named old lanes) | Tool의 고정 PATH directory enumeration만 추가 |
| Fixed Python image canonical PATH directories | NO(new read) | NO | NO | NO | UV-only exact dir/flags/OCI anchor+identity 및 부정 latches/actual |
| Other client/Artifact filesystem predicates | YES | YES | YES(named old scopes) | YES(named scopes) | 변경하지 않으며 기존 profile에 새 grant가 없는지 대조 |

C++ IsPinnedUVRuntimeRead에 pinned PATH의 네 canonical directory(/usr/local/sbin,
/usr/local/bin,/usr/sbin,/usr/bin)만 포함한다. /bin·/sbin은 고정 catalogue에서 각각
usr/bin·usr/sbin symlink이며 final resolved pathname을 쓴다. 하위 file/prefix 전체·코드 실행·
write 권한이나 다른 profile/clone/Artifact 예외를 추가하지 않는다. Existing UV runtime-read
latch에 finite positive와 path/flag/anchor/profile/role/identity negative를 추가한다.
기존 canonical builder로 새 observer를 만들고 실제 신규 tool과 기존 UV를 함께 확인한다.
두 번째 검토: 새 directory read가 안전성·Artifact execution authority가 되지 않으며 unknown
read는 여전히 실패한다. Final CPU/CUDA profiles와 public tools acceptance는 열려 있다.

L3 named local 최종: v2 Quick106.295s/전체 PyPI+Sandbox Race23.383s와 v3 docs0.328s/
security11.155s, unmodified canonical observer build227.363s/전체121latches가 PASS다.
C++ 변경 뒤 실제 Go binary는 v2와 SHA/size가 같으며 compiled/runtime pin을 별도로 대조했다.
Actual v3는92.114s이며 tool15.88s에서 cowsay6.1/1node/1command/4controls와 command 없는
idna3.11의 결과 없는 거부, 기존 UV73.76s의 단일/전이/whole/누락·empty 회귀가 모두 PASS다.
NoSKIP/helperwait0/Host protected snapshot 동일이다. [L3 receipt](./p4l3-uv-tool-native-install-local-20261026.json)가
정확한 scope와 최초 두 실패를 소유하며 original upgrade/current Policy/Host tool 설치 완료는 아니다.

Exact UV Shell::contains_path는 PATH의 각 directory를 same_file identity로 대조한다. Data-only
catalogue와 이 exact source를 연결했으며 directory 하위 파일이나 실행을 허용하지 않는다.
새 current canonical observer는 raw p4l3-uv-tool-directory-observer-canonical-v1.json이다.
이후 actual operator는 이 source/binary를 사용해야 하며 옛 B6b source hash를 현재로 전용하지 않는다.
다음은 L4의 명시 existing target·외부 command root를 함께 동결하는 guard/current 승인·반입 gap이다.
