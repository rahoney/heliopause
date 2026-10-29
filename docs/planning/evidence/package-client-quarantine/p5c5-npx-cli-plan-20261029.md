# P5-C5 — npx/npm exec와 분리된 안전한 출력

Baseline2720632cd3e97ac3938e41ea5e65707237176b51/treec61e6835/2985tracked·1337runtime.
같은 M13-004다. CI428/source558d75a는01:22:24UTC 8SUCCESS/Linux 진행이며 실제 timer05
due01:42:24UTC/session32835를 소비한다. 진행 중 새 push하지 않는다.
Read: INDEX routed sandbox/artifact/evidence/ci, Queue/Decisions/docs route, client09§1..4,
plan20§4..6, Architecture001/005/007, Contract002·Domain003, 기존 CLI 실행/표시 계약과
package_client/presentation/root/bootstrap.Run·C4 workflow/actual을 직접 대조했다.

| owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| C4 application/concrete factory | YES | YES(API) | YES(public2 local) | YES(named API) | public CLI 연결과 runtime adapter 호출 여부의 별도 scalar |
| current descriptor/preflight/localized help/progress | YES | YES(existing clients) | YES(existing scope) | YES(existing scope) | 두 npx/npm exec descriptor·초기 bounded grammar |
| safeTerminal | YES | YES(reference240rune) | YES(existing scope) | YES(reference only) | raw arbitrary stream 표현에는 그대로 전용하지 않음 |
| program stream renderer/public execution CLI | NO | NO | NO | NO | 명시 출력 선택·ASCII escaped streams, independent summary, real CLI requests |
| existing backend/CPP/permissions/budgets | YES | YES | YES(public2/controlled12) | YES(named backend) | 변경 없음; 동일12/CPP 재실행 없음 |

## 수정 지도와 두 번째 검토 — source 전

Initial grammar: `helox npx [--command <name>] [--show-output] <package> [--] [args...]`와
동일 의미의 `helox npm exec ...`를 현재 descriptor에 연결한다. 모든 HAA/global 옵션은
package 앞에 둔다. 첫 positional 뒤는 literal argv이고 직후 `--` 하나는 구분자로 제거한다.
Literal `--`를 전달하려면 구분자 뒤에 하나 더 둔다. Package와 command는 기존 npm parser/
exact bin ambiguity 규칙을 사용하며 count128/32KiB·UTF8/NUL/command214bytes를 I/O 전에
검사한다. Shell/-c/임의 package list/backend/runtime/target/stdin/TTY/download 옵션은
등록하지 않는다. Package 뒤 flag 모양은 프로그램 데이터다. Package 앞 같은 옵션 반복은
모호한 선택을 막기 위해 거부한다. Help는 parsed HAA help만 준비를 생략한다. 기존 Bootstrap의
전체 argv help 검색을 이 두 execution descriptor에 그대로 쓰면 program --help를 가로채므로
해당 lane만 parsed help/source-empty와 연결하고 기존 다른 명령은 유지한다.

Application은 runtime adapter 호출 여부를 scalar로 추가한다. 이는 guest body 시작 증명이
아니며 runtime 미호출/INCOMPLETE를 구분하는 보고 자료다. Existing result private copies와
검사/Policy/실제 kernel completion authority는 변경하지 않는다. CLI 보고는 quarantine/Policy,
storage preparation, execution/exit/signal, output promotion NOT_REQUESTED, execution guest
cleanup을 분리한다. Backend complete만 guest cleanup CONFIRMED이며 전체 process/service close는
기존 Bootstrap deferred lifecycle/error가 소유한다. 명시되지 않은 cleanup을 확정하지 않는다.

프로그램 출력은 기본 bytes count만 표시하고 argv/raw cause는 출력하지 않는다.
`--show-output`을 package 앞에 명시하면 stdout/stderr를 독립 필드의 ASCII quoted data로만
표시한다. `strconv.QuoteToASCII`로 CTL/ESC/OSC/C1/bidi/invalid UTF8/newline을 inert text로
표현하며 최대 capture1MiB/stream은 그대로다. 임의 secret 탐지/완전 redaction을 보장하지
않는다. Host secrets/credentials/env/실제 project는 계속 공급하지 않고 raw streams는 일반
format/Manifest/Evidence에 저장하지 않는다. Display-only DTO는 runtime 승인 자료가 아니다.

CLI error는 fixed message/code+Unwrap으로 원래 cause를 유지하며 machine/nonterminal에서도
main이 raw backend stderr를 재출력하지 않게 한다. Completed guest nonzero/signal의 exit는
독립 terminal에서 계산하고 incomplete는1, Policy review/block은10/20이다. Writer 오류는1이며
완료0으로 바꾸지 않는다. Human summary/en/ko와 JSON은 같은 facts를 사용한다.

Source: existing application/execution_run.go와 descriptor/presentation/bootstrap wiring의 좁은
추가, 새 CLI execution parsing/report 파일·의미 있는 negative/render tests와 Bootstrap actual
parent. 기존 준비/검사/backend/capture/CPP 구현은 대체하지 않는다. Registry에 future alias나
새 package/job을 만들지 않는다. Grammar/invalid bounds/도움말·package 뒤 literals와 safe
stream·writer failure를 local 검사하고 canonical Quick/security/docs·전체 영향 Race/tagged
분석/빌드 후 실제 두 alias의 public TypeScript 정상/비정상과 human/JSON 네 요청을 확인한다.
CLI fixture recorder와 report DTO tests는 positive runtime 승인이 아니며 actual이 별도로 증명한다.
Whole P5/final support4profiles/signed·파일 출력/TTY·P6/P7은 계속 OPEN이다.

## Named local 결과

[Receipt](./p5c5-npx-cli-local-20261029.json)는9profiles/전체 영향 Race와 actual CLI4/113.529s를 연결한다.
두 alias·human/JSON·en/ko·private/explicit output·정상0/typed guest1과 four fresh records를
확인했다. Backend/CPP/권한·한도는 그대로이며 Host/source/42/ZIP/Docker 보존·helperwait0/
resources/endpoints0/noSKIP다. Original producer의 마지막 stale1332 assertion은 별도
독립1343 source validator로 기록만 확인했고 제품 검사는 반복하지 않았다.
초기 CLI 범위만 네 상태 YES이며 grouped hosted/final/wholeP5는 OPEN이다.
