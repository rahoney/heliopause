# P6-H2 — UV 일회성 실행의 명시 추가 root

Source 전 기준은 e8e13a3/tree40dce85d다. H1은0434612 named local qualified이고,
CI441/source1ee는 Linux113PASS 뒤 job120분 한도로 취소됐다. 사용자가 승인한150분은
이미 local workflow/docs/security 검증·커밋했으며 push/새 CI는 아직 없다.
현재 단일 work item M13-004의 실제 미구현 `--with` gap만 다룬다.

Read: lessons INDEX·artifact L02/L19·sandbox L18·evidence L16·ci L06,
client09 §1/§3/P6 UV lane·plan20 §4–6·H1 receipt와 H baseline의 H2,
domain install/execution_context/project_snapshot/project_verified_set,
application execution_inspect/dependency_batch_inspect/project_inspect/execution_prepare/run,
policy M5, bootstrap Python runtime, sandbox UV native/frozen/console/runtime owners.
Pinned UV0.13.4 source7456b60의 run.rs1023–1059는 from과 with를 함께 resolve하고,
설치 environment 생성에 그 requirements 전체를 전달한다. 기존 source-discovery의 digest와
원본이 이를 소유하며 UV 출력이나 source 설명 자체를 실행 authority로 쓰지 않는다.

| 책임 / 기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 단일 root graph/검증 — domain/install.go NewLockedDependencyGraph | YES | YES | 기존YES | 기존scope YES | 불변식은 그대로 재사용. 명시된 독립 root 여러 개의 실제 graph를 함께 묶는 새 실행 입력 |
| primary execution/approved set — domain/execution_context.go | YES | YES | 기존YES | 기존scope YES | 전체 추가 root/노드/실제 edge와 새 전체 Policy 결합. 기존 context에 fake edge/설치 context를 넣지 않음 |
| project whole 검사 — project_snapshot/project_verified_set·application ProjectInspectService | YES | YES | 기존YES | 기존scope YES | InstallContext가 필수이므로 일회성 입력을 꾸며서 넣지 않음. per-entry/batch 원본 검사 본체는 재사용 |
| M3·M5 current 검사와 records/runtime — execution_inspect/prepare/run·M5·Python runtime | YES | YES | H1 포함named YES | 기존scope YES | 새 typed 실행 set의 전체 coverage·한 번의 전체 approval·Manifest/SBOM·sealed materialization. 기존 본체의 실제 공용 경계만 공유 |
| UV joint 선택/whole wheel/native census — uv_resolver/uv_run_resolver/pipx_native_input | YES | 기본/extra YES | H1 named YES | H1scope YES | 각 추가 requirement를 joint 선택·독립 전체 wheel/요청·fingerprint·고정 inert native 준비/census에 결합 |
| CLI 두 alias/literal argv/private report — cli/python_execution.go | YES | 기존YES | H1 named YES | H1scope YES | UV 전용 반복 --with typed 요청과 현재 consumer. pipx/설치/프로젝트 문법은 그대로 |
| H2 end-to-end | NO | NO | NO | NO | 아래 순서대로 구현/검증 |

## 가장 작은 표현과 기존 owner 재사용

새 실행 선택은 이미 검증된 primary LockedDependencyGraph와 각 명시 supplemental root의
실제 reachable graph를 보유한다. 각 graph는 기존 constructor로 검증하고 실제 edge를
그대로 유지한다. 공통 node는 source/artifact/record 정보와 outgoing edge가 동일해야 하며,
전체 union의 node/edge 한도와 서로 다른 명시 root를 검사한다. 추가 root가 기존 의존성인
경우에도 그 명시 요청을 fingerprint에 남긴다. Root 간 source 사실을 임의 edge로 잇지 않는다.
기존 connected graph/installed/project 모델은 확장하거나 대체하지 않는다.

공통 node는 한 번 acquire·verify·inspect·Evidence 생성한다. Per-entry/batch owner를 재사용하고
새 typed inspected/approved selection이 전체 노드와 실제 관계·각 fresh RunID를 결합한다.
M5의 동일 rule body로 전체 집합을 판단하며 primary 단독 ALLOW를 전체 승인으로 쓰지 않는다.
새 Manifest/SBOM와 readonly Python 입력은 같은 전체 set/fingerprint를 끝까지 소비한다.
기존 execution context를 부분 승인이나 fake 설치 snapshot으로 꾸미지 않는다.

Native compile은 canonical primary와 supplemental requirements 전체를 함께 받는다.
고정 offline UV tool run에는 검증된 추가 requirement만 HAA argv로 결합하고 terminal은
계속 /dev/null이다. 사용자 console/argv는 CONTROL 준비에 전달하지 않는다. Frozen owned
bytes/startup gate/원본 SHA/Simple/metadata/constraints·전체 census 검증을 그대로 재사용한다.
현재 C++에는 --from/--with 값/고정 argv 개수 parser가 없고 같은 UV profile/고정 image를
사용한다. 새 CPP grant 필요를 미리 가정하거나 구현하지 않으며 실제 실패가 있으면 그 경계를
조사한다. Immutable CPP133 입력은 필요가 생기기 전 반복하지 않는다.

입력은 기존 public bare/exact+root extras 문법을 재사용하고 duplicate explicit project,
URL/VCS/local/index/runtime/option 주입을 거부한다. 기존128개/32KiB의 유한 입력 한도와
source graph resource policy를 적용한다. 설치 권한이나 Host/네트워크/출력 권한은 추가하지 않는다.

## 작업 순서와 완료 기준

1. 기존 graph를 묶는 opaque 실행 선택과 같은 source를 보유한 runtime/lock binding을 구현한다.
   독립 root·겹치는 dependency·다른 source/edge·누락/변조·입력 copy와 기존 graph의 거부를 표적으로 검사한다.
2. 기존 whole 검사/Policy/records/runtime 본체의 실제 공용 경계를 새 typed set에 연결한다.
   Shared node는 한 번 검사하고 missing/extra/중복 RunID·부분 승인·다른 context를 거부한다.
3. Native 선택/frozen input과 두 CLI alias의 --with를 연결한다. 프로그램 argv는 literal/private다.
4. 첫 actual은 기존 python-slugify8.0.4 + --with Unidecode1.4.0이다. H1의 같은3wheel과 달리
   실제 metadata edge는 primary→text-unidecode 하나이고 Unidecode는 명시 독립 root다.
   전체3개 새 승인/records·두 alias의 normal0/typed2·충돌/미완료 거부·원본/Host/cleanup을 검증한다.
5. 의미 있는 H2 변경 묶음에서 Quick/영향 Race/필수 actual·기존 lane·docs/security를 모은다.
   H1·승인된150분 조정과 함께 다음 중요한 frozen 후보에서만 새 hosted를 진행한다.

새 기능/모델의 구현 권한은 원래 extras/추가 dependency 목표 안에 있다. 기존 구현 대체나
불변식 변경이 실제로 필요해질 때에만 정확한 위치·재사용 불가 이유·영향을 제시하고 해당
규칙의 승인을 먼저 받는다. 현재는 그 변경이나 새 engine·future placeholder가 필요하지 않다.
H2/final four profiles/whole milestone과 publication은 아직 OPEN이다.

## H2-A 진행 결과

새 domain execution_selection.go는 기존 opaque connected graph들을 입력으로 보유하고
실제 shared node/edge의 일치와 union 한도를 검사한다. 기존 graph/프로젝트/실행 context
source는 변경하지 않았다. Source/runtime binding과 root/공유 node/충돌/copy·이전 graph 거부
표적은1.093s, 독립 graph들의 node/edge aggregate 우회 거부2개는0.635s PASS다.
Canonical format만 적용했고 full Quick/Race/native/CPP/CI는 반복하지 않았다.
이 새 모델의 application/native/CLI 연결·whole 승인과 H2 acceptance는 아직 미완료다.
H1 완료 사본17개/500814759B는 root ref0/errors0/helper0/container0을 확인해 삭제했으며,
정확한 실패 v6 source19개를 기존 SHA와 대조해 보존했다. 현재 qualified ELF/observer와 모든
로그·공유 cache/원본은 유지했다. 다음은 실제 공용 inspection 검증/전체 M5의 재사용 연결이다.

## H2-B–E source 연결과 표적 결과

Execution selection의 current coverage는 기존 project validation body를 공유하고,
application의 acquire/verify/batch/Evidence 및 M5 rule body도 실제 기존 owner를 공유한다.
Whole environment의 각 wheel은 전체 선택 환경에서 한 번 관찰하며 metadata edge는 그대로다.
새 typed current context/bundle은 기존 single-root/installed 모델을 꾸미지 않고 전체 root를
records와 Python readonly materialization까지 보유한다. Manifest entry/SBOM serializer,
staging rehash/seal/atomic publish, runtime capture/terminal/lifecycle 본체를 재사용한다.

Native compile은 canonical primary와 모든 supplemental requirements를 함께 받는다.
각 실제 reachable graph와 union은 원본 wheel/Simple/constraints/startup 검증에서 만들어지고,
frozen identity·offline UV --with 준비·whole native census·new source fingerprint에 연결된다.
두 UV alias에만 반복 StringArray --with와 typed consumer를 연결했고 old Core/pipx 문법은 유지한다.

Domain/approval+M5 표적2.979s, whole inspection/application6.779s가 PASS다.
Records/runtime v5의 Domain/application/Sandbox/Bootstrap 표적은 PASS였고 신규 evidence fixture의
접근자 오기로 evidence compile이 실패했다. 실제 Evidence()로 고친 미도달 evidence 표적은0.437s PASS다.
V5 promotion은 no tests였으므로 staging 검증으로 쓰지 않는다. V8 actual 전 표적은
Artifact0.066s/Sandbox0.156s/Bootstrap8.299s/promotion0.218s PASS, CLI2개 FAIL이다.
CLI는 preflight/Execute 재파싱의 StringArray 누적 및 GetStringArray의 빈 값 소실이었다.
기존 Cargo의 raw GetSlice와 같은 preflight reset을 UV에 연결한 v9 UV/Cargo 표적2.762s는 PASS다.
V7 format의 새 test fixture 중괄호 오기도 보존했고, 제품/native 실패로 확대하지 않는다.

현재 Root/private source47 paths의 new workflow 표적/tagged vet·actual ELF 준비는
/tmp/haa-p6h2-actual-build-v10에 진행 중이다. Native actual부터 직접 정상/관측 불완전 거부를
검증하고, 이후 whole3-wheel CLI0/typed2/충돌 거부로 진행한다. 새 실제 parent 둘은 동일
기존 job/실행 step에 묶을 계획이며 아직 hosted에 연결·검증하지 않았다.
Quick/영향 Race/actual/hosted/final과 H2 acceptance는 아직 OPEN이다.

## 첫 실제 정상·거부 검증

동일 source47/기존 observer09615d38의 native parent58.29s/operator60.174s는 실제 공개
python-slugify8.0.4 + 독립 Unidecode1.4.0의 whole3-wheel·2roots/1real edge를 확인했다.
정상만 selection을 반환하고 frozen 관측 불완전 주입은 원본 raw complete/전달 incomplete를
구분해 selection/console을 비웠다. Source/native trace6개, offline Outside/network0이다.

이어 같은 build의 새 CLI parent3/137.75s(operator139.641s)는 두 alias의 whole3 fresh
ALLOW·새 Manifest/SBOM·정상0/typed guest2와 충돌 요청의 no Policy/runtime/new records를 확인했다.
Roots2/edges1 및 SBOM3/edges1, 각 fresh operation/Run/Evidence와 privacy/cleanup을 검사했다.
두 operator 모두 Host5/Docker 보존·helperwait0·설정 same-inode restore·query errors0이다.
원본·현재 검증 ELF 둘을 보존하고 실제 종료 사본6개/169266794B만 참조0 확인 뒤 제거했다.

기존 uv-run-source/uv-run-cli profile 각각에 별도5m/8m parent command를 하나씩 덧붙였다.
새 job/profile·예산/권한은 없고 expected ordinary parents는122→124다. H1 기존8은 그대로다.
첫 checker의 parent-count negative mutation 오기를 고쳤다. 실제 보존 로그 DATA replay는
새 guard가 기존 파일명까지 바꾼 오류를 거부했고, 정확한 uv_run_resolver_real_test.go로
workflow/고정 checker만 교정한 정상/누락/변조/SKIP10개가 PASS다. 이것은 DATA 검사이며
새 actual full8 성공이 아니다. Quick 진행 중 그 두 guard 파일만 Root에 교정했고 private
Quick 입력은 고정했다. 완료 뒤 수정된 workflow/checker를 표적/Race로 검사하며 unchanged
runtime47의 actual·Go 검증이나 CPP를 반복하지 않는다. 최종 영수증은
[p6h2 local receipt](./p6h2-uvx-supplemental-roots-local-20261031.json)가 소유한다.

## 모인 Go 검증

Quick325.699s, 변경 영향10 packages Race207.472s·전체 checker Race88.909s가 PASS다.
교정한 workflow1.823s도 PASS이며 51 source paths의 Root/private SHA가 동일하다.
Quick는 파일명 guard 교정 전 private 입력이었고 후속 workflow/전체 checker Race는 교정 후다.
Runtime47은 첫 실제 빌드와 동일하다. 이 정확한 재사용 경계는 source proof/receipt에 남겼고
native/CPP/Quick를 무변경 반복하지 않았다. 최종 docs/security와 Git checkpoint 이후에만
H1·승인150분·H2를 모은 새 CI를 시작한다. Raw는 로컬에 보존되며 GitHub 전달을 주장하지 않는다.

최종 상태: `IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES (named local) / ACCEPTANCE_CLOSED=NO`.
Final docs0.305s/security19.089s도 PASS다. Source51과 raw/cleanup·정확한 재사용 경계는
영수증에 연결했다. `MISSING`: 새 frozen ordinary hosted, 같은 source의 CPU/cu126/cu130/cu132,
최종 support matrix/whole acceptance다. Main merge·publication은 이 검증에 포함되지 않는다.
