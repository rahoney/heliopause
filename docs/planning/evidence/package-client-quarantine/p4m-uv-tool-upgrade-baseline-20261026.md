# P4-M persistent tool upgrade baseline

사전 감사 기준은 D commit294bb578c168baca86121d4ba75ce1afb30983c6다. E public fresh CLI는
canonical/전체 CLI·Bootstrap Race와 actual317.049s의 새 CLI 네 subcase 및 기존 UV
CLI 일곱 요청을 통과했다. 착수/단일 active work item은 Queue를 따르며 M13-004 하나다.
Read: Queue/plan20/client09, Architecture Evidence/Staging/Promotion, Domain installed update/
complete set, UV tool install·upgrade exact source, UV installed-selection transport, current
installed snapshot/complete ownership, tool receipt/guard/publication, lessons 관련 항목을 확인한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Fresh native UV tool/source/whole graph/current primary approval/atomic files | YES | YES | YES(D named local) | YES(named backend) | Public E named local PASS; 원격 checkpoint 별도 |
| Existing PyPI installed snapshot/managed ownership/complete retire·backup | YES | YES | YES(named old UV/project) | YES(named scopes) | Tool receipt·exposed old links와 결합 |
| UV0.13.4 native tool upgrade command | NO(adapter) | NO | NO | NO | Fixed original replay/query lifecycle·native selected output/source bind |
| Managed original tool metadata/receipt/command identities before native | NO | NO | NO | NO | Whole inventory·old receipt/constraint/bin-root/interpreter/collision proof |
| Old exposed links backup·retirement with complete venv publication | NO | NO | NO | NO | Retained/retired/new commands·no-replace·rollback/recovery |
| Native original tool adoption / pipx production / public tool upgrade | NO | NO | NO | NO | 별도 original proof/native/runtime/CLI acceptance |

## 확인된 native 의미

Exact pinned source uv-tool-commands/src/upgrade.rs: upgrade_tool은 receipt를 읽고
InstalledTools::get_environment를 호출한다. uv-tool/src/lib.rs는 이를
PythonEnvironment::from_root로 이어 query한다. 기존 조사대로 interpreter query는 -I/-B/-c이고
-S가 아니므로 원본 installed code의 startup을 CONTROL로 허용하지 않는다.

Upgrade requirements는 existing_tool_receipt.requirements를 사용하고 positional 조건은
추가 constraints로 처리한다. Outcome은 tool/dependency/environment/no-op를 구분한다.
Exact version pin은 유지하며 version upgrade를 건너뛴다는 hint가 나온다. 정상 no-op도
현재 HAA 전체 검사/승인·정확한 현재 반영을 생략하는 authority가 아니다.

Tool/environment 변경 시 remove_entrypoints 뒤 finalize_tool_install을 호출하며 기존
requirements와 python을 보존한다. 따라서 단순 UV pip upgrade alias나 selected wheel만의
원복으로 외부 retired commands를 설명할 수 없다. Original receipt와 exposed links를
같은 guarded publication에 참여시켜야 한다.

## 다음 검증과 재사용 경계

기존 UV pip installed resolver는 bounded METADATA-only 원본을 private guest에 전달한다.
Native tool은 receipt/venv/commands와 original environment query가 더 필요하다. 먼저 고정
UV/Python·inert synthetic wheels의 별도 offline diagnostic에서 metadata-only controlled
original view로 native upgrade가 동작하는지, 어떤 파일/query·no-op/version-change/retired
commands 의미가 필요한지 확인한다. 이 진단은 public source/Policy/Host upgrade qualification이
아니며 원본 프로그램을 Host에서 실행하지 않는다.

기존 구현을 복제·대체하지 않고 확인된 부족 부분에만 concrete adapter를 붙인다. 기존
installed snapshot/managed inventory/whole ownership와 complete transaction을 재사용하되,
원래의 unsafe/unknown/injected 상태를 receipt claim으로 덮지 않는다. Future generic interface나
pipx/run placeholder는 만들지 않는다. 실제 diagnostic 뒤 source map·두 번째 검토와 validation
범위를 확정하고 구현한다. Role/예산/지원/완료 기준의 변경은 이 문서에서 승인하지 않는다.
