# P4-O pipx immutable supply and runtime baseline

M13-004 하나 아래 후속 pipx 감사다. N2D의 fixed native UV tool 결과는 해당 receipt가
소유하며 pipx로 승계하지 않는다. 현재는 source 작성 전 baseline/구체 수정 지도를 준비한다.

Read: Queue/plan20/client09, P4-L L1/L2와 exact native source/배포물 census,
trusted-tooling D-012/D-013/D-015, runtime maintenance §2·3·6·7 및 기존 UV/PNPM 공급,
PyPI ToolReceipt.BindWheels/ParsePipxToolReceipt, 기존 tool root/transaction/current pipeline,
artifact/sandbox/ci/evidence lessons. Baseline은 N2D1b16544/tree616b506bb789cffff8359220262383782d66ab7b/2657 files다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Pipx1.18.4 metadata0.12/UV backend typed parser | YES | YES(pure parser) | YES(L1 named) | YES(parser only) | current backend/output/public CLI 연결 |
| Whole authentic wheel/graph/main commands/startup-surface gate | YES | YES(pure receipt binder) | YES(L2 named) | YES(binder only) | native query 전에 실제 artifact startup 역할을 보장하는 경계 |
| Immutable pipx executable closure/provenance/maintenance | NO | NO | NO | NO | exact official 배포물 서명·attestation 제공 여부, 전체 closure 검증과 단일 runtime owner |
| Real pipx install/upgrade selection/runtime/backend isolation | NO | NO | NO | NO | exact argv/고정 backend/metadata post-install Python 역할/complete source binding |
| Tool materialization/two-root current approval/publication | YES(existing UV engine) | NO(pipx) | YES(existing scopes only) | NO(pipx) | pipx receipt/외부 script links/constraint 제거 의미의 concrete adapter |
| Pipx public CLI/actual negatives/hosted/final profiles | NO | NO | NO | NO | 위 prerequisites 후 missing만 구현 |

## 이미 확인한 것과 다음 확인

P4-L은 exact1.18.4 source commit b6919968a12a675012feae5ee4ae91ab56636e8c와
release pyz727151B/SHA2569273476832b2672f0799a84f69c72fcd5008e7a4dab3723a48fd0e25c3ddd19f,
228 members/2581174 uncompressed bytes를 static census로 보존했다. 이 값은 공급 pin/admission이나
현재 runtime 성공이 아니다. 정전으로 사라진 /tmp source를 raw 자료와 exact upstream Git blob으로
복원하며 기존 문제가 해결됐는지 같은 실험을 되풀이하지 않는다.

Exact backends/uv.py, venv.py, venv_inspect.py를 upstream commit에서 다시 읽었다. Parser는
stored pip_args를 실행하지 않고 UV backend만 받는다. 실제 native는 bundled/PATH UV 탐색,
version probe와 설치 후 venv Python -c 조회를 수행한다. Host PATH·shiv mutable cache 또는
installed hook을 trusted DATA-only로 실행하는 방법은 사용하지 않는다. UV tool alias로 pipx를
대체하지 않고 pipx upgrade의 원래 version 조건 제거를 보존한다.

다음은 official1.18.4 배포물의 publisher proof 제공 여부와 immutable full module closure의
runtime 공급·loader를 구체화하는 read-only audit다. 제공 proof와 실제 bytes를 대조하기 전
새 runtime lock/config/interface/CI job을 만들지 않는다. 기존 공급 verifier/lock generator/
maintenance/guest admission의 재사용 위치와 exact missing edge를 먼저 적고 재검토한다.
Artifact startup 가능성이 있는 post-install Python을 현재 graph helper 성공만으로 CONTROL로
등록하지 않는다. 이 문서는 구현·신뢰 승인·pipx 지원·whole P4 acceptance를 닫지 않는다.


## 2026-10-27 공급 discovery — 암호 검증 전

공식 release API는 기존 pyz727151B/digest와 일치했다. 해당 pyz digest의 GitHub attestation
endpoint는404였으며 이것만으로 전 세계 proof 부재를 단정하지 않는다. Exact release workflow의
zipapp job은 upload만 선언하고, PyPI publish는 별도 OIDC publishing job이다. PyPI의 exact
`pipx-1.18.4-py3-none-any.whl` provenance endpoint는200이며 publish/v1 statement에
wheel SHA2564672195db7e89526cda1e54cbea116164a58bd40e1c03fe12e380de16b5399a4가 들어 있다.
Publisher metadata는 pypa/pipx·release.yml·release environment다. 모두 decoded discovery이며
서명·SCT·Rekor·publisher certificate·subject는 아직 검증하지 않았다. Pyz와 다른 subject인
wheel의 proof를 pyz proof로 쓰지 않는다.

다음은 existing pinned Sigstore verifier/root로 이 별도 PyPI publish predicate와 exact
publisher/certificate/source/run/subject를 검증할 수 있는지 확인한다. 기존 HAA/UV/PNPM의
SLSA policy는 유지하며 별도 upstream의 사실을 그 기존 정책으로 통과시키지 않는다.
Wheel-backed immutable module closure가 필요하다면 bundled dependency 전체의 authentic
bytes/provenance·stdlib/loader 역할도 계산한다. Mutable shiv cache와 pyz wrapper를 신뢰하지
않으며 기존 단일 runtime lock/generator/maintenance/provisioning 재사용 지도를 먼저 확정한다.
아직 product source/lock/config/admission은 바꾸지 않았다.

원본은 local raw owner의 `p4o-{release,zipapp-attestations,wheel-provenance}.raw.json`과
`p4o-supply-discovery.json`이다. Official sources:
[release](https://github.com/pypa/pipx/releases/tag/1.18.4),
[exact workflow](https://github.com/pypa/pipx/blob/b6919968a12a675012feae5ee4ae91ab56636e8c/.github/workflows/release.yml),
[wheel provenance](https://pypi.org/integrity/pipx/1.18.4/pipx-1.18.4-py3-none-any.whl/provenance).


## 한정 offline publish proof 진단

기존 pinned sigstore-go1.3.0/root를 사용하는 사설 진단은 동일 statement/signature/certificate/
transparency entries를 표준 bundle로 표현하고 exact publisher/tag/source/repository IDs/subject를
검증했다. SCT1·transparency1·observer timestamp1을 요구했으며 정상 proof는 PASS, 다른
publisher/digest·transparency 부재·바뀐 payload는 모두 거부했다. Signed invocation은
39104119239/attempts/1이며 publish/v1의 null/empty predicate 의미는 공식 PEP740/명세와 대조했다.
이는 metadata publish proof에 한정하며 기존 SLSA verifier를 바꾸지 않았다.

공식 PyPI의 exact wheel181563B를 DATA로 취득해 signed SHA와 대조했고 52개 pipx module의
집합/bytes는 이전 release pyz census와 모두 같았다. 나머지 bundled dependencies의 전체
authentic closure와 supply/maintenance/admission은 미완료다. Scratch console은 도구 결과로
확인했지만 raw stdout file을 캡처하지 않았으므로 derived `p4o-publish-diagnostic.json`에 그
한계를 명시했다. 입력 원본/변환 bundle/진단 source/census는 보존했고 동일 무수정 실행을
raw 확보만을 위해 반복하지 않는다. 제품 helper·runtime pin·Host 실행/지원 qualification이 아니다.


## 남은 closure 범위

기존 zipapp의 Linux dependency6개는 argcomplete3.7.2/click8.5.0/filelock4.2.1/
packaging26.3/platformdirs4.13.4/userpath1.9.2다. 앞5개의 exact wheel provenance는200이고
userpath는404였다. 아직 그5 proof를 암호 검증하거나 전체 dependency wheel을 취득하지 않았다.
Windows marker의 colorama0.4.6과 UV extra는 Linux default closure에 자동 추가하지 않는다.
Upstream exact commit의 uv.lock Git blob/census와 각 wheel filename·size·hash를 대조한
`p4o-upstream-lock-binding.json`이 일치 여부를 소유한다. 이 정적 source lock 대조는 없는
서명을 만들어 주거나 dependency/module/runtime 신뢰를 완료시키지 않는다.

후속 구현 전 MISSING: 각 제공 proof의 fixed publisher/cert/subject 검증, 전체 archive/RECORD/
module closure와 dependency constraints/marker 대조, signature 미제공 항목의 정확한 provenance
경계, mutable cache를 사용하지 않는 loader·isolated Python/UV backend 의미, native startup
Artifact 역할 및 maintenance/generator/actual admission. 이 baseline에서 source/pin/권한은
변경하지 않았다. 공급 표현을 바꾸는 concrete edit map/두 번째 검토가 다음 단계다.


대조 결과 filelock4.2.1과 platformdirs4.13.4는 exact source uv.lock에 같은 wheel tuple이
없었다. 따라서 그 lock을 기존 release zipapp 전체 closure의 인증 자료로 전용하지 않는다.
현재 wheel publish proof는 pipx 자체만 묶으며, 새로운 exact dependency closure를 선택한다면
별도 source/proof/constraints·실제 module 집합을 검증해야 한다. 차이를 tampering이나
정전 이후 변경으로 단정할 증거는 없고 zipapp build의 실제 선택 재현도 아직 수행하지 않았다.


## Baseline 검증과 다음 재개

제품 source/runtime lock/공통 verifier는 바꾸지 않았고 문서의 canonical docs0.383s/
security12.191s가 PASS다. Source N2D1b16544의 CI416은 첫 실제10분 조회03:05:29UTC에
8SUCCESS/Linux IN_PROGRESS이며 Required와 exact hosted qualification은 미완료다.
Source checkout/full decoded logs·native tool exact5 실제 실행·FD 실패 진단은 CI 완료 후
결론과 연결한다. 이번 문서 전용 checkpoint는 그 실행 후보에 새 push하지 않는다.
다음은 pipx whole immutable closure의 concrete edit map/second review이며 공급 helper를
만들었다거나 pipx가 지원된다고 표시하지 않는다.


## CI417 대기 중 독립 공급 증거 감사

CI416 fixture-only 교정518800c/tree df689208은 별도 actual/canonical 검증 뒤 push했고
CI417/run40449588924는04:24:47UTC에 시작했다. 이 대기 중 제품 코드·pin은 바꾸지 않고
기존 P4-O 미완료인 dependency 증거를 확인했다.

공식 repository ID·owner ID와 각 workflow run의 source commit/event/path/success를 별도
조회한 뒤, 제공 proof5개와 실제 archive digest를 기존 pinned Sigstore root/library에서
검증하는 사설 offline 진단2.751s/exit0이 PASS였다. Argcomplete/click/filelock/packaging/
platformdirs 각각 exact SAN/source/ref/repo IDs/run/subject와 SCT1·log1·observer timestamp1,
wrong digest 거부를 확인했다. Source repo는 filelock=tox-dev/filelock이고 packaging의 실제
publisher는 tag push가 아닌 main/workflow_dispatch다. 한 publisher policy를 다른 도구로
일반화하지 않는다. Stdout/stderr/source/policy/input hashes는 p4o-dependency-publish-actual-*
및 고정 source text로 보존했다. 기존 HAA/UV/PNPM verifier는 변경하지 않았다.

여섯 wheel DATA census의 Python module96개는 기존 release pyz와 집합/bytes가 모두
같았다. Pipx 본체52개와 구분하며 전체 archive 구조·RECORD 검증, dependency constraints/markers,
immutable loader·runtime/post-install Artifact 역할·공급 pin/admission은 여전히 미완료다.
Userpath1.9.2는 signature endpoint404이며 DATA/hash/module 대조만 확인했다. 그 부재를
서명 성공이나 공급 authority로 바꾸지 않는다.

또한 사설 진단은 라이브러리가 노출하는 SAN/source/run identity를 검증했으며 PyPI publisher의
signed environment/subject OID(57264.1.23/.24)까지 검증한 production PEP740 bridge는 아니다.
정식 bounded decoder/publisher 정책은 그 경계와 negative 회귀를 concrete edit map에 포함해야
한다. 이 결과를 runtime 공급 전체 QUALIFIED, pipx 지원 또는 P4 acceptance로 표시하지 않는다.
