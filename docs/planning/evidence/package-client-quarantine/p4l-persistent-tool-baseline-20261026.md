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
