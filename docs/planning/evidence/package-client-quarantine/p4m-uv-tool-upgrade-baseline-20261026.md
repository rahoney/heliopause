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

## M1 진단 결과·입력 책임 지도·두 번째 검토

Offline 고정 UV/Python 진단 v1(2.426s)은 원본 package code를 제거한 기존 native venv에서
bounded1.0→1.1 및 haa-alt retirement/haa-new 노출, exact pin no-op, positional <1.1 no-op를
확인했다. V2(1.515s)는 고정 runtime 구성과 METADATA만으로 새 환경을 만들었으나 native
uninstall이 RECORD 부재로 실패했다. V3(2.324s)는 controller가 METADATA와 RECORD 두 경로만
기록한 synthetic RECORD를 추가해 같은 세 시나리오·여섯 명령을 통과했다. 원본 RECORD나
package/startup code를 전달하지 않았으며 세 번 모두 exact owner container 제거와 Docker
inventory 동일을 확인했다. 원본 p4m-tool-upgrade-metadata-v{1,2,3}-*를 보존한다.
이는 offline native 의미/최소 입력 진단이며 gVisor 관찰·public source·Policy·Host upgrade
qualification은 아니다. 기존 fresh output parser는 retained WHEEL/명령 scaffold까지 요구하므로
이 진단만으로 upgrade의 whole-output/source binding이 구현됐다고 하지 않는다.

M1은 typed original/intent input만 구현한다. 기존 UpgradeInput의 METADATA identity/marker/
source/수량·aggregate bound/정규화를 재사용하고 strict UV receipt parser도 유지한다.
NewUVToolUpgradeInput은 원본 receipt 요구와 positional upgrade 조건을 별도로 보존하고
같은 public root 및 원본 root version의 원래 조건 충족을 확인한다. 원본 installed version은
새 positional 조건 밖이어도 되며 이것을 잘못된 원본으로 처리하지 않는다. Receipt의 inert
options를 native에 그대로 전달하지 않도록 public index/copy/no-build/no-sources 고정 recipe만
받고 find-links/no-index/다른 recipe는 before-native에서 거부한다.

IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO였던 tool 전용 입력의
MISSING은 canonical roundtrip, combined byte bound, root/receipt/request 결합, defensive copies와
recipe 거부다. internal/artifact/pypi/uv_tool_upgrade_input.go와 해당 test만 추가하고 기존
parser/resolver/transaction을 대체하지 않는다. 두 번째 검토: 기존 UpgradeInput은 재사용할 수
있지만 tool receipt 및 원본 요구와 positional 조건 둘을 표현하지 않으므로 단독으로는 부족하다.
신규 control은 데이터일 뿐 ownership/허용/입양/반입 권한을 부여하지 않는다. Native adapter,
current original whole proof, old external links 및 complete publication, CLI는 이후 gap으로 남는다.

M1 named input는 canonical Quick48.107s/security13.251s/docs0.252s와 전체 PyPI Race를
통과했다. 최초 Quick216.538s는 mismatch fixture가 JSON의 HTML-escaped 조건을 plain text로
치환해 assertion에 미도달한 실패이며 원본을 보존했다. Nested wire를 구조적으로 바꾸고
combined encoded byte bound도 검사했다. 기존 receipt/UpgradeInput 구현은 변경하지 않았다.
[M1 receipt](./p4m1-uv-tool-upgrade-input-local-20261026.json)가 정확한 source/check/미완료를
소유한다. M2는 held original roots·receipt·전체 managed metadata/소유 script·외부 link guard다.

## M2 original target guard baseline·수정 지도·두 번째 검토

기존 beginToolFreshTarget의 Linux/고정 venv/분리된 storage·두 root lock/interpreter DATA
fingerprint와 Close/recovery, pypi installed snapshot/managed inventory/checkBefore,
anchored toolLinkMatches는 IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES(named local)다.
Original receipt/metadata·소유 main scripts·노출된 old links를 같은 guard에 묶는 경로는
NO/NO/NO/NO다. MISSING은 before-native original capture/원본 변경 거부·원본 native 불채택
거부이며 publication/native/CLI를 이 단계에서 닫지 않는다.

기존 fresh entry는 common beginToolTarget(..., original=false)를 호출하고 original entry는
같은 root/lock/interpreter 구현에 original=true를 지정한다. Fresh site-empty/receipt-absence는
유지하며 original은 receipt regular file을 기존 tx.before에 넣고 installed snapshot 및 기존
managed inventory로 전체 distribution이 이미 소유됐는지 확인한다. Native adoption은 거부한다.
M1 typed input이 원래 receipt 요구와 positional 조건을 구분하며 main script owners/version/
artifact digest와 exact exposed symlink identity·target을 결합한다. 다른 tool의 bin 항목은
보존하며 해당 명령의 대소문자 충돌·누락·다른 link는 거부한다. Interpreter는 실행하지 않는다.

수정 범위는 tool_target.go의 공통 진입과 original verification 분기, 신규 tool_original.go 및
해당 tests다. 기존 fresh publication이 original mode를 우회해 쓰이지 않도록 freezeCommands/
commitTool에서 명시 거부한다. Common begin 본문은 receipt capture/마지막 verify mode 외에
기준과 같은지 기계적으로 비교한다. 두 번째 검토: original guard는 현재 ledger의 파일 digest와
고정 원본을 검증하는 pre-query guard이며 이전 ALLOW의 재사용이나 native receipt만의 adoption이
아니다. 새 current full 검사·whole-output/source binding 및 old link retirement/rollback은 남는다.

M2 named guard는 canonical Quick171.037s/security11.569s/docs0.247s·전체 Promotion
Race160.938s와 기존 fresh CLI actual을 통과했다. Actual의 CLI78.29s는 JSON30.73s/human
공백 root29.56s/두 foreign 거부1.61s·16.38s이며 NoSKIP/helperwait0/Host 보호 snapshot 동일이다.
Native original upgrade나 Host command 실행으로 확대하지 않는다. Common begin 본문과
fresh publication은 original mode capture/거부 분기를 제외하면 baseline과 byte-identical이고
기존 transaction/snapshot/inventory/native resolver는 변경하지 않았다.
[M2 receipt](./p4m2-uv-tool-original-guard-local-20261026.json)가 source·검증 범위를 소유한다.

## M3 complete tool publication baseline·수정 지도·두 번째 검토

기존 complete PyPI venv backup/journal/retirement·원복, C fresh native receipt/새 exposed link
publication 및 M2 original guard는 IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES(named local)다.
기존 old exposed links의 backup/retirement·원복과 원래 receipt 교체는 NO/NO/NO/NO다.
MISSING은 retained/retired/new command 집합의 guarded 반영·원래 link inode 보존·두 filesystem
경계의 recovery이며 current native upgrade/source/inspection/CLI는 이후 acceptance다.

M3는 기존 toolPublication에 concrete optional original-link participant를 붙인다. Fresh nil
경로는 유지한다. g.freezeCommands는 original mode에서 새 receipt가 원본 receipt 조건을
유지하는지 확인하고 positional 요구도 실제 새 main version에 적용한다. 기존 명령은 M2의
exact old link를, 신규 명령은 exact absence와 foreign/internal collision을 고정한다. 기존
commitSelectionWithProject의 complete=true로 기존 owned files/retired scripts를 반영하고
원래 receipt regular file도 같은 backup/rollback에 둔다. Ledger schema는 유지한다.

외부 bin과 같은 filesystem의 소유0700 backup root에 old symlinks를 anchored/no-replace rename하고
journal에 device/inode/target과 intent/result를 남긴다. New link publication 전 old 전체를
고정하고 정상 readback에서 retired absence/new identities/old backups를 확인한다. Rollback은
새 links를 제거한 다음 old link inode를 되돌리고 기존 venv 원복을 수행한다. Unknown/foreign
link나 backup은 지우지 않으며 원본 journal·backup/두 lock을 보존한다.

Old link backup은 rollback에 필요한 데이터이므로 기존 private workspace cleanup에서 삭제하지
않는다. Common transaction의 post-rollback/private-cleanup 뒤 regular backup disposal 앞에서
명시적으로 처분한다. 처분 불확실성은 false success나 사라진 old inode의 복원으로 표시하지
않고 recovery error와 남은 regular backup/journal을 보존한다. 기존 fresh cleanup body는
그대로 재사용한다. 수정은 tool publication/original helper, 필요한 optional disposal hook 및
관련 tests에 한정하고 runtime/observer/권한·한도·native semantics를 변경하지 않는다.

두 번째 검토: 기존 fresh guard만 우회해서 old file을 덮는 방식은 외부 retired links와 원본
receipt를 설명하지 못한다. 별도 병렬 venv engine 대신 현재 complete engine을 사용하며
외부 symlinks의 identity-aware 참여만 추가한다. Current new-set ALLOW/Manifest가 없는 공개
호출 경로는 만들지 않는다. Fault/cancel/foreign/cleanup 실패 및 기존 fresh 실제 소비자를
검증하고 synthetic helper 성공을 public upgrade qualification으로 확대하지 않는다.
