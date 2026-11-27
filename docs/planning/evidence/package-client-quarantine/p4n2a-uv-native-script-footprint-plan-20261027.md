# P4-N2A native UV console-script footprint

Baselineb530832/treec025da1/2642다. N1 matcher693ba4a는 pure DATA helper이며 실제 native
ownership/current approval에는 미연결이다. CI415/a8389de의 PNPM FD 실패는 원인 OPEN이며
N2A의 archive/installed-byte proof와 독립적이다. M13-004 하나만 IN_PROGRESS다.

Read: N baseline/N1 receipt, client09/plan20/Queue, pypi_native_inventory·ownership,
tool_target의 interpreter snapshot, uv_install의 config/discovery와 기존 native UV tests,
lessons artifact L01/L02/L15·sandbox L08/L18·evidence L16을 대조한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Exact public old wheel/Simple·archive/RECORD source authentication | YES | YES(existing pip/UV project) | YES(named) | YES(named) | tool original enrollment의 별도 연결 |
| Native inventory의 scripts 경로·중복/escape/hash/snapshot 검증 | YES | YES | YES(existing native inventory) | YES(named) | generated wrapper의 독립 source/held interpreter 결합 |
| N1 exact UV0.13.4 wrapper byte matcher | YES | NO(production ownership) | YES(DATA only) | YES(named helper) | 기존 footprint에 concrete optional tool proof 연결 |
| Existing snapshotToolInterpreter·transaction guard/config | YES | YES(managed tool) | YES(named local) | YES(named) | native script proof의 held snapshot 재사용 |
| Native UV tool script의 complete footprint proof | NO | NO | NO | NO | held interpreter/entrypoint/authenticated wheel·실제 installed bytes 증명 |
| Native original enrollment/current update/CLI | NO | NO | NO | NO | N2B 이후 raw receipt/projection·old source·전체 current 검사와 공개 opt-in |

## 수정 지도

기존 proveNativePythonWheelFootprint는 nil extension으로 공유 body에 위임한다. 일반 pip/UV
경로는 generated console/gui script를 계속 거부한다. 인증 wheel의 EntryPointDetails를 지우거나
InstalledFiles를 여과해 기존 검사를 속이지 않는다. 새로운 concrete UV tool proof만 console
entrypoint를 matched scripts destination으로 증명하고 기존 wheel-file/RECORD/marker/cache
검사와 같은 complete inventory를 통과시킨다. GUI/extra/data scripts·다른 installer는 거부한다.

Concrete proof는 held pypiVenvTransaction과 기존 interpreter snapshot을 사용한다. Pinned
CPython config·canonical root/bin/python 및 alias chain·terminal file의 identity/hash를 DATA로
고정하고 처음과 script read 전후에 다시 검사한다. 프로그램/Host Python은 실행하지 않는다.
EntryPoint name·module·callable은 인증된 old wheel에서 받고, exact scripts/name 목적지와
실제 파일의 executable mode/snapshot/digest/size 및 N1 byte matcher를 함께 검사한다.

기존 footprint의 public wheel identity/digest, 전체 archive member, RECORD self/INSTALLER/
REQUESTED, extra/cache 거부 및 정렬된 proof/destination 반환은 재사용한다. 새 role은 source
proof의 설명이며 이전 ALLOW/반입 권한이 아니다. enrollment/current update에 아직 연결하지
않고 이 단계는 complete footprint helper만 QUALIFIED 판정한다.

## 두 번째 검토

필수 반례는 ordinary pip/UV default 거부 유지, unauthenticated/다른 digest·EP·interpreter,
변조/비실행/추가/누락/중복 script, GUI/extra·기존 wheel-file·marker/cache 실패와 source/alias
경쟁이다. 원본 RECORD hash만으로 script를 승인하지 않는다. Synthetic archive는 독립 wheel
inspection으로 읽고 실제 native N1 fixture를 DATA로 비교하며 ownership ledger나 기존 승인
상태를 만들어 성공으로 표시하지 않는다. 전체 Promotion/PyPI 회귀와 canonical profile을 적용한다.

Raw native receipt의 Python은 CLI가 지정한 base interpreter일 수 있다. 이를 target/bin/python으로
추측해 normalize하지 않는다. raw bytes/source/receipt/link census와 terminal interpreter 관계,
controlled replay projection/current Manifest binding은 후속 N2B의 명시된 missing이다. Native
adoption/Host runtime provisioning/새 observer 권한·예산·지원 완료는 이 단계의 결과가 아니다.

## 한정 로컬 검증 결과

[영수증](./p4n2a-uv-native-script-footprint-local-20261027.json)의 final Quick193.732s/
security11.073s/docs0.276s와 PyPI·Promotion 전체 Race168.066s가 PASS다. Complete declared
footprint helper는 IMPLEMENTED=YES/WIRED=YES(shared helper)/QUALIFIED=YES(named local)다.
Production native enrollment WIRED=NO, whole native adoption ACCEPTANCE_CLOSED=NO다.

Synthetic1526B archive를 독립 wheel inspection으로 읽고 N1의 실제 native wrapper DATA와
비교했다. RECORD hash까지 맞춘 script 코드/interpreter 변조, 누락/추가/비실행 script,
marker/cache/wheel/entrypoint/GUI/extras/중복 및 held runtime/config/guard drift를 거부했다.
일반 pip/UV의 generated-script 거부와 ownership ledger 무변경을 확인했다. Wrapper/Host
Python 실행이나 실제 공개 source 인증·current Policy/반입·CLI는 하지 않았다.

기존 interpreter 검증 루프를 그대로 공통 함수로 추출했고, nil extension의 기존 footprint
전체 bytes와 네 inventory/update/original/publication owner 불변을 기계적으로 대조했다.
첫 Quick209.779s의 alias fixture는 동일 payload의 unlink/recreate로 distinct inode를 보장하지
않았다. 당시 inode 수치는 미기록이며 역사적 원인을 단정하지 않는다. Old symlink를 rename해
보존하고 새 inode가 다름을 독립 확인한 최종 fixture만 사용한다. 원본 첫 실패는 보존했다.

다음 N2B는 raw receipt/base interpreter/link census와 기존 public old-wheel enrollment를
연결하고 unrecorded cache 부재까지 기존 helper로 고정한다. 기존 A1 metadata-only selection
layout projection을 먼저 재사용할 수 있는지 확인하며 raw bytes를 바꾸거나 native default
recipe를 임의의 public recipe로 위장하지 않는다. Current complete approval와 M3 publication/
public opt-in·실제 qualification은 후속 acceptance다. 새 push/full CI는 하지 않았다.
