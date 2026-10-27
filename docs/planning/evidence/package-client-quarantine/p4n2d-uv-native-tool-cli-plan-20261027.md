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
