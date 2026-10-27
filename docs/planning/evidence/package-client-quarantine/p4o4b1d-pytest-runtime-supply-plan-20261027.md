# P4-O4-B1d — 명시 pytest runtime 공급

기준044eff9다. 사용자는 B1b/c에서 확인된 pytest plugin import 실패에 대해
required coverage를 유지하고 pytest와 필요한 의존성을 고정 runtime 공급에 추가하는
권장안을 승인했다. 검사 중 자동 보조 설치·원래 root 실패 상쇄·coverage 완화는 하지 않는다.
Read: O4 owner/B1 map, plan20 §5·6, client09, M5 contract의 command/prerequisite 구분,
D-012/013/015, runtime maintenance §2·3·6·7, 관련 artifact/sandbox/ci/evidence lessons.
M13-004 하나만 IN_PROGRESS이며 기존 publisher/whole wheel/graph/ZIP/owner를 재사용한다.

## Source 전 baseline

| 책임/위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| `VerifyPipxDependencyPublish`, fixed publisher policy | YES(existing5) | YES(existing) | YES(existing) | YES(existing) | pytest/iniconfig/pluggy exact publisher, pluggy의 signed ref subject와 environment 부재 |
| unsigned wheel identity/metadata binding | YES(userpath) | YES(userpath) | YES(existing) | YES(existing) | Pygments의 제공 proof 부재를 구분한 exact binding |
| `BindPipxSupplyClosure`, whole RECORD/constraints | YES(7-wheel) | YES(old supply) | YES(old) | YES(old supply only) | 명시 pipx + pytest 두 공급 root의 각각 완전한 graph와 합집합 |
| 기존 installed role/classifier | YES | YES(existing) | YES(existing) | YES(existing) | pytest의 원본 `pytest`, `_pytest`, `py.py` 소유 namespace |
| deterministic ZIP/original bytes/no-replace | YES(Store) | YES(old) | YES(old) | YES(old only) | 추가 whole wheel을 기존8MiB transport 안에 보존하는 Deflate 표현 |
| lock/generator/check/frozen maintenance | YES(schema5) | YES(27inputs) | YES(old) | YES(old only) | 42inputs/31documents/11wheels·새 ZIP/NOTICE identity와 변경된 owner의 재검증 |
| static snapshot/plan/intake + C1/C2 DATA | YES | YES(local named consumers) | YES(old pin local) | YES(old named scope) | 새 pin/encoding/namespace·actual fixed DATA + import 결과 |
| runtime/observer/release/native pipx/CLI | NO(pipx complete path) | NO | NO | NO | B2 이후 기존 O4 map 전체 |

## 공식 입력과 공급 역할

PyPI JSON latest 조회와 version-specific 원문·실제 wheel SHA/크기를 따로 보존했다.
아래 후보는 아직 새 공급 acceptance가 아니다. Packaging26.3은 기존 선택을 재사용한다.

| 추가 후보 | 원본 wheel B | 원본 member 합 B | publisher 입력 |
| --- | ---: | ---: | --- |
| pytest9.1.1 | 386536 | 1363441 | pytest-dev/pytest, deploy.yml, refs/heads/release-9.1.1, workflow_dispatch, environment deploy |
| iniconfig2.3.1 | 7552 | 18341 | pytest-dev/iniconfig, test.yml, refs/tags/v2.3.1, push, environment pypi-upload |
| pluggy1.7.0 | 27082 | 82101 | pytest-dev/pluggy, deploy.yml, refs/tags/1.7.0, push, environment 없음 |
| Pygments2.21.0 | 1250147 | 4621166 | PyPI provenance endpoint404; unsigned identity/metadata binding, signature로 표시하지 않음 |

Pytest metadata의 Linux/CP314 active dependency는 iniconfig/packaging/pluggy/Pygments다.
Windows colorama, old-Python exceptiongroup/tomli와 dev extras를 넣지 않는다. 기존 marker/
Requires-Python/constraint parser로 실제 원문과 선택 결과를 검증하며 손으로 dependency edge를
위조하지 않는다. 두 root는 HAA runtime 공급의 명시 구성이다. Pytest를 pipx upstream의
Requires-Dist라고 기록하거나 사용자가 설치할 package graph에 추가하지 않는다.

새 publisher의 certificate claims와 exact source commit workflow, GitHub repository/owner ID를
대조했다. 암호학적 acceptance는 기존 verifier를 연결한 뒤 별도로 확인한다. Pytest는 publish
후 tag를 만드는 실제 workflow이므로 tag signer를 지어내지 않는다. Pluggy의 environment=null,
OID23 부재와 OID24 exact `repo:pytest-dev/pluggy:ref:refs/tags/1.7.0`을 명시한다. 기존 환경형
publisher의 OID23/24 요구는 보존하고 null/빈 문자열/다른 ref/중복 claim을 혼동하지 않는다.

공식 근거: [pytest9.1.1](https://pypi.org/project/pytest/9.1.1/),
[iniconfig2.3.1](https://pypi.org/project/iniconfig/2.3.1/),
[pluggy1.7.0](https://pypi.org/project/pluggy/1.7.0/),
[Pygments2.21.0](https://pypi.org/project/Pygments/2.21.0/).
원본 준비 owner는 local `p4o4b1d-candidate-20261027`이며 versioned receipt가 최종 inventory를 연결한다.

## 수정 지도와 두 번째 검토

1. Publisher policy 세 개와 제공 proof9/미제공2를 정확히 구분한다. 같은 verifier와 trust root,
   source commit/run/digest/issuer/SAN/transparency/certificate identity 검사를 유지한다. 새
   environment 없는 publisher에서만 signed exact tag-ref subject와 OID23 부재를 요구한다.
2. 기존 공급 constructor는 유지하고 같은 private 검증 본체에 명시 runtime 구성 진입점을
   추가한다. Pipx와 pytest 각각의 reachable subset을 원본 metadata edges로 얻고 기존
   `BuildLockedGraph`로 각각 single-primary graph를 검증한다. 합집합이 전체 선택과 같아야
   한다. Domain의 exactly-one-primary invariant나 일반 UV/user root 언어를 바꾸지 않는다.
3. Pytest의 실제 top-level 소유 root만 기존 mapping/classifier와 연결한다. `_pytest`/`py.py`는
   새 신뢰 예외가 아니며 whole RECORD·충돌·stdlib shadow·entrypoint/실행 관찰은 유지한다.
   Runtime plan도 이 같은 owner mapping을 소비한다. PyPI wheel identity 위장은 없다.
4. 원본657 members의 합은8,560,611B로 기존 Store 전송의8,388,608B를 넘는다. Payload·metadata·
   license를 삭제하지 않고 ZIP의 표현을 고정 compiler의 deterministic Deflate로 변경한다.
   출력 ZIP과 C1/C2 DATA volume의8MiB 상한은 그대로 둔다. 확장 크기는 이미 적용 중인
   canonical PyPI graph의256MiB/파일 수 budget과 별도로 대조한다. Store에서 같았던 encoded
   transport size와 expanded graph size를 구분하며 CPU/memory/wall/volume budget은 올리지 않는다.
   Snapshot도 exact selected digest 후 bounded streaming CRC/expanded graph 검사를 수행한다.
   새로운 encoding/bytes에 기존 Store qualification을 승계하지 않는다.
5. Runtime lock schema5의 기존 owner shape를 유지하고 exact 입력 집합을42개로 갱신한다.
   생성 identity·NOTICES·pinned metadata와 frozen plan/apply/verify·실제 reader/transport를
   함께 검증한다. Root 선택은 reviewed runtime constructor가 소유하며 arbitrary roots 설정을
   새로 노출하지 않는다. Version/bytes 선택은 기존 lock의 input path/hash가 소유한다.
6. 기존 source parser·supply 정상/negative와 actual42inputs/9proof/whole RECORD/ZIP 원문 동일성,
   canonical Quick/security/docs/workflow/runtime drift, 관련 전체 Race와 frozen maintenance,
   actual C1/C2 새 pin·고정 Python plugin import를 순차 실행한다. 작은 helper마다 push하지 않고
   common selected-owner/runtime consumer 변경을 묶어 필요한 hosted 회귀로 보낸다.

현재 문서는 baseline/계획이며 새 공급의 검증·활성화·release/전체 pipx 완료가 아니다.

## D1 publisher 로컬 결과

[D1 receipt](./p4o4b1d1-pytest-publishers-local-20261027.json)는 Quick88.520s/전체
releaseprovenance Race7.606s/docs/security PASS와 기존5+새3 dependency proof의 실제
crypto 검증·wrong identity/publisher/env/ref/signature/statement 거부를 연결한다.
새 no-environment claim 검증은 JSON null과 signed OID23 부재/정확한 OID24 ref를 요구한다.
기존 main/환경형 공급 검사는 유지한다. 새11-wheel whole supply/ZIP/runtime acceptance는 남는다.
