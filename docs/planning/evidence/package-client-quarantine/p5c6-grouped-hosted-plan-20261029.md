# P5-C6 — 현재 경계·workflow·CLI grouped hosted 연결

Baseline6521e76182a6908d4f5f077e9c61a3f35b7d9a0a/tree7bbc68c6/2994tracked·1343runtime.
단일 active item은 M13-004다. CI428/source558d75a는 all10SUCCESS, all10 same checkout/tree와
decoded1,278,395B·CPP131/Linux94/Node2/kernel8/정리를 확인했다. Timer05는01:42:43UTC에
최종 결과를 소비하고 종료했으며 active CI/timer는 없다. 정상 milestone push와 중요한 CI는
사용자가 명시 승인했다. Main merge/tag/publication은 별도다.

Read: Queue/Decisions/docs route, lessons INDEX와 ci L05/L06·sandbox L03/L08/L12/L18/
artifact L02/evidence L16, client09§3/current npx scope와 plan20§4..6, C3d3/C4/C5 receipt,
기존 heliopause-ci.yml Node preparation/kernel steps·ci-integration-cleanup.sh와 canonical
workflow checker/actual parent source를 대조했다.

| 기존 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Node preparation/current execution2 hosted step | YES | YES | YES(CI428 source558) | YES(named) | 유지 |
| controlled boundary12 current test source | YES | YES(local caller) | YES(local202.496s) | YES(named local) | 현재 whole source hosted caller |
| Application public2 workflow / npx CLI4 | YES | YES(local API/CLI) | YES(local58.861/113.529s) | YES(named local) | hosted current3 parents·actual UID/환경/source tree |
| exact gVisor/Host captures/Policy/backend/canonical Required | YES | YES | YES(existing named scopes) | YES(named scopes) | 구현 변경 없음 |

## 수정 지도와 두 번째 검토 — workflow 수정 전

새 job을 만들지 않고 기존 unconditional `Qualify production Node execution preparation` 단계에
세 parent를 순차 추가한다. 동일 현재 Bootstrap test binary를 기존 protected client 설치/정리
owner로 사용하며 기존 preparation parent는 유지한다. 각 부모는 별도 invocation의 failfast/
5min 한도를 유지한다. 합쳐 한5min command로 만들면 12case+CLI4의 총 시간이 개별 한도를
잘못 소모하므로 합치지 않는다. 기존 Go caller4min/runtime45s/stream1MiB/Required/job bounds·
network/권한·SDK·CPP는 그대로다.

각 추가 parent는 원래 command exit를 먼저 보존하고 cat 실패도 검출한다. 같은 로그에서
no-SKIP/정확한 parent PASS1 및 case12/2/4를 anchored actual 기록으로 검증한다. Program
stdout/marker는 authority가 아니며 test assertions가 real current runtime을 확인한 뒤 기록한
고정 metadata만 guard 집계에 쓴다. 로그/fixture를 safety 또는 Policy 승인 자료로 대체하지
않는다. 기존 kernel8·기존 소비자는 유지한다.

Canonical workflow/docs/security를 먼저 실행하고 Bash syntax 및 실제 보존 로그를 DATA로
읽는 guard positive/skip/missing/duplicate/wrong-parent/미완료 negatives를 확인한다. 이는
shell/guard 검증이지 새 actual runtime 결과가 아니다. Go/CPP1343/binary/source는 불변이므로
전체 Quick/Race/actual12를 로컬에서 이유 없이 반복하지 않는다. 새 source의 hosted가 actual
네 부모를 확인한다. Final docs/한글 commit 후 기존 milestone branch 정상 push, run/source/
checkout을 연결하고 시작 직후 actual20min timer로 계속 감시한다. 완료 전 hosted QUALIFIED나
whole P5/final support4profiles/signed/P6/P7 완료로 표시하지 않는다.

## Local wiring 결과

[Receipt](./p5c6-grouped-hosted-wiring-local-20261029.json)는 canonical workflow/docs/security와 Bash syntax0,
saved-log guard24/primary exit6을 연결한다. Go/CPP/runtime1343과 기존 공급·권한·예산은
불변이다. Saved logs는 DATA이며 새 actual이나 hosted 성공으로 표시하지 않는다.
다음은 최종 metadata·한글 commit/정상 push와 새 run/source/timer 등록이며 hosted
QUALIFIED/ACCEPTANCE_CLOSED는 결과 전 NO다.
