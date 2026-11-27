# P4-N2D explicit native UV tool CLI and actual original

Baselineced69f3와 보존된 N2C working delta를 먼저 구분한다. N2C의 canonical/default,
전체 affected Race와 기존 managed/fresh/current Evidence 실제 소비자 결과는 N2C receipt가
소유한다. 이 문서는 그 결과를 기다리는 동안 같은 M13-004에서 작성한 후속 수정 지도이며
새 active work item이나 whole native qualification을 만들지 않는다.

Read: Queue/plan20/client09, N/N2A/B/C leaf, uv_tool_upgrade descriptor/preflight와
package client/root의 기존 InstallUpdateRequest dispatch, 기존 실제 tool CLI/current
Evidence tests, uv_native_fixture의 bounded Docker transport와 native project 소비자,
artifact/sandbox/ci/evidence lessons를 대조한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Native 원본/whole source/current controls 및 existing publisher | YES(N2C draft) | YES(internal service) | NO(actual native) | NO | public native 원본의 current 전체 검사/Manifest/원자 반영 실제 증거 |
| Public uv tool upgrade와 machine/human 결과 | YES(managed) | YES | YES(named local) | YES(managed only) | 명시 --adopt 등록/검증/전달과 지원 한도 설명 |
| Native venv fixture의 locked UV DATA supply·bounded Docker/cleanup | YES | YES(existing UV pip/project tests) | YES(named) | YES(named) | tool용 fixed public recipe·실제 Host 경로의 raw receipt/script 생성 |
| Native tool public normal/negative/반복 승인 | NO | NO | NO | NO | 원본 전체 독립 source proof/current 검사·거부 불변/새 RunIDs/records |

## 정확한 수정 지도

uvToolUpgradeDescriptor만 --adopt bool을 등록하고 existing typed update request에 전달한다.
Fresh install에는 flag를 추가하지 않는다. Existing target/bin path·source·argument preflight는
유지하며 duplicate adopt와 unsupported options를 구성/네트워크 전에 거부한다. English/Korean
help는 fixed public wheel-only recipe(--no-build/--no-sources/--link-mode copy), explicit old-source
proof, 같은 버전 재검사와 unmanaged default recipes/추가 index·runtime 준비 미지원을 설명한다.
기존 다른 언어의 deterministic fallback과 machine schema/결과 owner는 유지한다.

기존 native venv fixture의 locked UV supply·bounded Docker I/O와 exact container cleanup을
재사용한다. Tool fixture는 별도의 concrete 진입점으로 자신의 새 private host 부모만 같은
absolute 경로로 guest에 제공하고, pinned image Python의 인증 DATA 사본을 그 부모에 둔다.
Raw receipt의 base interpreter와 생성 wrapper의 target/bin/python 경로를 수정하지 않는다.
Trusted native UV의 fixed public tool install만 fixture producer에서 수행하며 target program/
Host Python은 실행하지 않는다. Public 원본 생성에 필요한 guest network는 fixture에 한정하고
product의 gVisor/typed egress·observer/current Policy를 완화하지 않는다. 기존 pip fixture의
network-none/local-index 데이터 생성 경로와 모든 기존 assertions는 유지한다.

공개 sample의 실제 exact old 설치 후 no-adopt 거부, 원본/script·receipt 변조 거부와 원본 불변,
명시 adoption의 current required checks/Evidence/ALLOW·10 controls/raw receipt/source proof,
현재 반입 receipt와 old metadata 제거를 확인한다. 그 뒤 managed 반복 adopt는 새 RunIDs와
8 controls로 현재 재검사하며 원본 source proof를 이전 ALLOW로 쓰지 않는다. Native receipt의
exact old pin은 임의 unversioned 값으로 바꾸지 않고 유지한 상태의 정상 no-op부터 검증한다.
변경 버전 native recipe는 별도 실제 bounded requirement로 선택 의미를 확인한 경우만 추가한다.

## 두 번째 검토

Source proof는 소유권 자료이며 현재 approval이 아니다. 실제 원본 producer가 raw recipe를
만들지 못하거나 native output이 기존 supported inventory와 다르면 최초 오류와 concrete
gap을 기록하고 N2B gate/예산·관찰 권한을 낮추지 않는다. Fresh/managed CLI와 기존 native
pip/project 소비자, 현재 Evidence 실패의 원본 보존도 검증한다. Root-owned private mount
namespace에서 현재 test client/observer만 연결하고 owning shell의 helper wait0·confirmed
cleanup과 Host 보호 파일/Docker inventory 전후를 확인한다. Canonical Quick/security/docs,
CLI/Bootstrap/Promotion/PyPI affected Race와 actual tagged compilation을 사용한다.
기존 ci-uv-tools bootstrap scope에 새 native adoption 소비자의 명시 env와 exact test
선택을 추가하고 기존 checker의 skip/missing-consumer 부정 회귀를 확장한다. 기존 네
소비자와 두 client scope·helper/restore·12m 제한은 유지한다.
작은 helper별 push/full CI를 추가하지 않으며 hosted/final CPU·CUDA acceptance는 계속 열어 둔다.

## 첫 현재 후보와 actual — 완료 판정 아님

N2C는17293ac로 로컬 보존했다. N2D 초기 canonical Quick274.740s/security19.604s/
docs0.509s/workflow1.298s와 tagged compile3.748s가 PASS다. 기존 native fixture의 locked
supply/bounded Docker I/O·cleanup, guest writer와 native pip 데이터 생산/반출 세 구간은
N2C와 byte-identical임을 확인했다. 기존 default fixture 실제 소비자 검증은 아직 남는다.

Exact ci-uv-tools bootstrap actual344.580s는 앞 네 기존 소비자 PASS 뒤 새 native 원본
생성4.30s에 도달했다. No-adopt 호출은 UPDATE_ORIGINAL_STATE_UNSUPPORTED로 거부됐지만
새 assertion이 generic UPDATE_PREPARATION_FAILED를 기대해 실패했다. Current native
inspection/승인/Manifest/반입·Evidence 실패는 NOT_REACHED다. Helperwait0/confirmed stop와
Host 보호 파일/Docker 동일은 확인했다. `p4n2d-first-actual-assessment-20261027.json`과
first-tool-actual-v2 원본을 유지하며 이를 product guard의 회귀나 native 지원 완료로 쓰지 않는다.

교정은 no-adopt의 기존 거부 code family와 fixture regular/single-link 확인, bounded 원본
inventory/raw receipt DATA 진단이다. 원본을 바꾸지 않은 current Evidence 실패를 변조
시험보다 먼저 확인한다. 다음 actual은 기존 helper/data/env를 유지한 채 새 native consumer
하나만 선택하며, 성공 여부와 기존 네 소비자의 앞 결과를 각 실행/소스 범위대로 구분한다.

좁힌 v3 actual5.063s는 native 원본의 regular files/alias·활성화 scaffold·원본 receipt를
DATA로 남겼다. 이번 no-adopt는 generic preparation/layout 거부였으므로 한 코드만 고정한
assertion은 여전히 부정확했다. 첫 실행의 내부 원인은 원본 DATA가 없어 미확정이다.
두 관측된 원본 거부 코드만 허용하되 FAILED·current checks/references0·원본 inode/bytes/
mode/link 동일·ledger 부재를 함께 요구한다. 당시 explicit mutated adoption에는 generic preparation
거부를 요구했으나, 아래 v4 관측으로 같은 두 code family로 교정했다. Product gate는 바꾸지 않고 정상 native current 경로를 먼저 확인한다.
원본은 `p4n2d-native-only-v3-assessment.json`과 같은 prefix의 raw다.

V4는 genuine native current inspection 뒤 실제 Evidence permission failure와 원본 불변을
확인했다. Mutated original도 같은 두 기존 거부 코드 중 하나였으므로 arbitrary error가 아닌
그 code family/zero-current-check/원본 불변을 함께 요구한다. V5 native actual126.927s는
exact6.0 원본의 현재 승인/Manifest10·반입과 managed 반복 adopt의 새 Evidence/Manifest8을
모두 PASS했다. Helperwait0/Host 동일·exact named test1도 확인했다. 제품 gate 변경은 없다.

### 추가 actual version-change 범위와 재검토

기존 supported bounded requirement의 실제 버전 교체도 확인한다. Native producer가 원문
`cowsay>=6.0,<=6.1`로 실제6.1을 설치하고, explicit positional6.0은 그 원래 범위 안에서
6.0을 선택해야 한다. 이는 실제 downgrade를 포함한 version replacement 증거이며 native
원본 조건을 합성하거나 새로운 recipe/options를 허용하지 않는다. Raw receipt·old6.1 source
전체 proof/current6.0 검사/Manifest10 및 old6.1 METADATA 제거를 같은 publisher로 검증한다.
고정 native UV가 이 선택을 수행하지 못하면 최초 실패/의미를 확인하고 product guard나
원래 constraint를 바꾸어 통과시키지 않는다. 공통 fixture는 두 literal public requirements만
허용하고 default native pip의 network-none/data writer/export body는 유지한다.


## 최종 native 소비자 — named local 검증

V6 actual171.767s는 tagged binary `a54bd23e`와 frozen whole-source2656 file inventory를
연결하며 exact named test1/noSKIP/helperwait0/CONFIRMED_STOPPED·보호 파일5개와 Docker
전후 동일을 확인했다. Genuine exact6.0과 bounded6.1→6.0의 현재 complete required 검사,
서로 다른 Evidence4개씩, Policy ALLOW, Manifest10 controls와 raw receipt/old-source proof,
현재 Host receipt·반입을 검증했다. Bounded 원문은 보존하고 old6.1 metadata만 제거하며
사용자 파일을 보존한다. Managed 반복 --adopt는 새 Evidence4개/Manifest8로 재승인한다.
현재 Evidence의 실제 permission failure와 no-adopt/script·receipt 변조 거부도 원본 파일의
inode/bytes/modes/links와 ledger 부재를 보존한다.

이 결과는 fixed `--no-build --no-sources --link-mode copy` public-wheel recipe의 명시
native 인계에 한정한다. Ordinary native 기본 recipe·pipx·hosted/final profiles와 M13-004
전체 acceptance는 열려 있다. 앞 exact five-consumer script의 네 기존 PASS와 후속 native-only
PASS를 구분하며 첫 전체 실행을 all-success로 고쳐 쓰지 않는다. 원본 prefix는
`p4n2d-native-bounded-v6-*`, full source inventory는 `p4n2d-native-v6-source.json`이다.
기존 native sync/project의 affected actual과 최종 canonical/Race 결과는 local receipt에 연결한다.


기존 default native fixture affected actual281.476s는 동일 binary에서 exact named2/noSKIP로
native project add/sync122.81s와 complete sync156.69s를 통과했다. Multi-root·retained·empty·
explicit native 인계와 사용자 파일 보존을 확인했고 helperwait0/CONFIRMED_STOPPED·보호 파일5개/
Docker 전후 동일이다. 기존 default native writer·network-none/export 경계의 실제 회귀 증거이며
새 tool recipe 밖의 일반 native 지원 선언은 아니다. `p4n2d-native-default-fixture-v1-*`를 따른다.


## 최종 audit 및 체크포인트

| 현재 N2D 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 명시 --adopt/preflight/help/typed 요청 | YES | YES(public CLI) | YES(named local) | YES(named recipe) | hosted exact candidate 검증 |
| Authentic native 원본/current10 controls/실제 원자 반영 | YES | YES(existing engine) | YES(named local exact/bounded) | YES(named recipe) | ordinary native defaults/다른 recipe는 unsupported |
| 현재 Evidence 실패·변조/no-adopt·managed 재승인 | YES | YES | YES(named local) | YES(named cases) | hosted/final profile |
| 기존 native project/complete sync 영향 경로 | YES(reused) | YES | YES(same binary actual2) | YES(named regression) | broader profile acceptance는 별도 |
| P4/M13-004 전체 | YES(partial) | YES(partial) | NO(whole) | NO | pipx·hosted/final support matrix 및 P5..P7 |

Final canonical Quick62.462s/full affected PyPI·Promotion·Bootstrap·CLI·checker Race183.469s,
workflow1.088s/docs0.446s/security12.445s와 tagged compilation이 PASS다. 실제 whole-source2656의
모든 실행 입력은 최종 후보와 같고 이후 차이는 문서뿐이다. [Local receipt](./p4n2d-uv-native-tool-cli-local-20261027.json)가
명령/결과/원본 hash·source binding·첫 실패 보존을 연결한다. Raw는 local owner의 자료이며
SHA 참조를 원격 raw 보관으로 주장하지 않는다. 최종 문서 변화에는 docs/security만 적용하고
실행 입력이 바뀌지 않은 Quick/Race/actual을 근거 없이 반복하지 않는다.


## 후속 hosted 결과

[CI417 qualification](./ci417-uv-tools-hosted-qualification-20261027.json)이 source518800c의
supported fresh/managed/native tool exact5와 native UID1001 current 승인을 확인했다.
CI416 producer UID 교정의 named acceptance와 이 native hosted gap은 닫혔다. 앞 단계의
부분/실패 결과는 원래 scope로 유지하며 pipx·ordinary defaults·전체 profile/P4 acceptance는
이 결과로 닫지 않는다.
