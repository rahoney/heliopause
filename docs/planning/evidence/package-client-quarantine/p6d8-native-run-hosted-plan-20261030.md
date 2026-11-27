# P6-D8 — native-run important grouped hosted checkpoint audit

D7가 아직 IN_PROGRESS이며 이 문서는 독립적인 다음 변경 범위 감사다. D7 qualification/commit 전 source를 바꾸지 않는다. 단일 work item은 M13-004다. 현재 Root d7a333b/runtime1387, private D7 candidate1392이고 CI435/sourceb0는 active20min public GET worker가 확인한다. 새 push/CI는 없다.

Read: PROJECT/Queue/docsREADME와 lessons INDEX·ci L05/L06·artifact L01/L02·sandbox L03/L08/L12/L18·evidence L16, current plan20§5 grouped test/cleanup와 client09, B4 hosted plan/local, D5 resolver/D6 current app/D7 CLI plans/local evidence, scripts/ci-uv-tools.sh, check/uv_tools_workflow.go 및 기존 weakening tests, 기존 current execution workflow step과 protected-client cleanup owner를 대조한다.

| Work slice | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Existing native source/census DATA2 | YES | YES | local YES / CI435 PENDING | named hosted PENDING | CI435 finished whole evidence |
| D5 production native selector normal+incomplete | YES | local YES / hosted NO | named local YES | named local YES / hosted NO | 기존 source profile에 exact parent·cases·trace guards 추가 |
| D6 current native app3 | YES | local YES / hosted NO | named local YES | named local YES / hosted NO | 기존 never-started Python DATA exporter를 쓰는 별도 current app profile·guard |
| D7 current aliases5 | YES | local YES / hosted NO | PENDING | NO | first metadata gap 수정 qualification 뒤 exact CLI profile·cases/whole graph/refusal guards |
| Existing Required/jobs/limits | YES | YES | CI435 PENDING | 이전 source 경계만 | 변경하지 않음 |

기존 구현을 대체하지 않는다. 같은 current execution step 안에서 protected Bootstrap/helper를 유지한 채 D6 app3와 D7 CLI5를 추가한 뒤 기존 Sandbox rebind/source2에 D5 selector parent1를 결합한다. D6는 기존 고정 Python DATA exporter, D7는 불필요한 exporter 없는 early profile이며 production native/현재 ALLOW와 DATA-only를 구분한다. Source profile parents2→3, top-level Linux106→109가 예상된다. 기존 CPP132/UV195·Required10·Linux120min·권한·coverage·예산은 유지한다.

수정 지도는 scripts/ci-uv-tools.sh, scripts/check/uv_tools_workflow.go, 해당 기존 tests, .github/workflows/heliopause-ci.yml이다. Closed profile list·exact flag/parent·ordinary user·cwd·helper rebind 순서·unique IDs/noSKIP·original command error before cat error를 기존 validator에서 확인한다. 새 병렬 parser/framework나 placeholder job을 만들지 않는다. 관련 source-shaped weakening negatives와 저장된 실제 로그 DATA로 exact guard 검사를 한 묶음에 실행하고, CI 전용 수정만으로 native/CPP actual을 자동 반복하지 않는다.

D7 source+metadata+현재 native/CLI/hosted wiring이 준비되고 CI435가 완료된 뒤 중요한 end-to-end 변경 묶음을 한 번 push한다. CI435를 취소하거나 active source 위에 작은 helper push를 하지 않는다. 새 remote source/tree/runtime·Required10와 old/new named consumers를 확인한다. 약108분의 기존 run에 D5~50초/D6~115초/D7실측을 더한 예상은 시간 보장이 아니며 120분 계약을 늘리지 않는다. P6 generic equivalence/pipx run/P7/전체 milestone·signed release acceptance는 별도로 남는다.

## Source 변경 전 최신 baseline

Root f8bdd0e248b9616496d707dcbdc0ab941a5df043/treedcec0cff/3117tracked/runtime1392다. D7는 canonical10와 실제UV5/191.58s로 named local QUALIFIED/ACCEPTANCE_CLOSED=YES이며 hosted MISSING만 남는다. CI435/sourceb0는 Required10SUCCESS/전체treeeebd72a1/all10checkout334e96e3, Linux106/CPP132/UV195/CPU298.02s와 108분31초로 완료됐다. Actual20min GET6회 뒤 worker13791 자연0/next none다. 신규 D5/D6/D7는 그 source 밖이다. 아래 map4를 기존 owner 안에서만 수정하며 no native/CPP rerun을 유지한다. 저장된 actual DATA guards·weakening negatives를 canonical workflow/Quick/affected checker Race와 묶는다.

## 구현·로컬 qualification

기존4개 source만 수정했다. 기존 current execution step/protected Bootstrap/helper에서 native app3와 CLI5를 검증한 뒤 기존 Sandbox rebind/source2에 production resolver 정상·불완전 parent1을 연결한다. Closed profiles/never-started DATA owner/early payload-free CLI와 original-error·cat failure precedence를 유지한다. Canonical8/전체 checker Race 및 실제 저장 로그 DATA33개 정상·거부가 PASS이며 새 native/CPP actual은 반복하지 않았다. 실제 새-source hosted 결과만 남고 source109/Required10/CPP132·UV195/Linux120min/권한은 유지한다. D5/D6/D7와 한 번에 push하며 docs-only/동일 source 재시도는 없다.
