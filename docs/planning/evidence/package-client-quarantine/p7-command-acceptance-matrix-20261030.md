# P7 — 명령 목표와 실제 acceptance 대조

현재 hosted 후보는 source1ee37880f7ea1c38ef4b541fe567f65c7c618ec3/treea5eedf7a,
CI441이며 결과는 PENDING이다. canonical 목표/현재 lane과 기존 증거를
대조하고 새로 구현한 H1의 named local 범위를 구분하는 인계용 matrix다. 정확한 구문·옵션·보안 한계는 [client09](../../../user-journey-cli-ia/09-package-client-quarantine.md)가
계속 소유한다. 아래 named 성공을 현재 source 또는 whole acceptance로 확대하지 않는다.

| 목표군 / 계약 구역 | 구현·연결과 기존 named 증거 | 현재 source에서 남은 acceptance |
| --- | --- | --- |
| UV pip install/upgrade/sync — §8 | [P1 scoped acceptance](./p1-uv-acceptance-20261023.json); 기존 venv/public wheel·explicit empty·전체 원본/삭제·현재 승인/rollback | 이후 공통 source의 최종 CPU/세 CUDA 및 지원표 결합 |
| PNPM project add/install/update/adopt — §9 | [P2/P3 matrix](./p2p3-final-matrix-20261026.md); native whole instance/peer·ownership·실패 원복 named 검증 | 현재 ordinary·세 CUDA/whole P2 |
| Bun project add/install/update/adopt — §10 | 같은 [matrix](./p2p3-final-matrix-20261026.md), [CI413](./p3-grouped-bun-ci413-20261026.json); 별도 Bun runtime/whole 검사/transaction | 현재 ordinary·세 CUDA/whole P3 |
| UV project add/sync/lock --upgrade | [P4-K3](./p4k3-uv-project-lock-cli-local-20261026.json); installed와 lock-only·원본 guard/whole approval/반영을 구분 | 현재 source·최종 matrix; lock-only 성공을 설치 완료로 쓰지 않음 |
| UV tool install/upgrade/adopt | [CI417](./ci417-uv-tools-hosted-qualification-20261027.json); fresh/managed/명시 fixed native recipe named 검증 | 현재 source·최종 matrix; 다른 native recipe·일반 기본값 미지원과 구분 |
| pipx install/upgrade/adopt | [CI423](./ci423-pipx-current-hosted-qualification-20261029.json); 고정 UV backend/whole original·script ownership·관리형 upgrade | 현재 source·최종 matrix; inject/ensurepath 등 별도 operation을 완료로 쓰지 않음 |
| npx/npm exec | [CI429](./ci429-node-cli-hosted-qualification-20261029.json); public whole set·선택 Node entry·CLI4/controlled12/거부2 named 검증 | 현재 source·최종 matrix; stdin/TTY·생성 파일 반출·arbitrary child 지원으로 확대하지 않음 |
| uvx/uv tool run | [CI437](./ci437-close-entry-uv-hosted-qualification-20261030.json), [D7](./p6d7-uvx-cli-local-20261030.json); native 선택/whole original·현재 승인·console/CLI named 검증 | H1 root extras가 기존 whole 경로에 연결됐고 실제3-wheel/두 alias/거부를 확인함; Quick/영향 Race/checker Race/docs/security PASS. 독립 --with의 H2는 미구현; 이후 현재 source/final |
| pipx run | [CI438](./ci438-pipx-current-hosted-qualification-20261030.json), [E3](./p6e3-pipx-current-cli-local-20261030.json); 기본/명시 console·고정 UV backend·앱3/CLI6 | 현재 source·최종 matrix; 다른 backend/module entry는 현재 지원 아님 |
| pnpm dlx/pnx/pnpx | [CI439](./ci439-pnpm-current-hosted-qualification-20261030.json); fresh native graph·원본/Node·앱2/CLI3/refusal2 | 현재 source·최종 matrix; pnpm pnx를 별도 subcommand로 만들지 않음 |
| bunx/bun x 기본 Node | [G2](./p6g2-bun-current-cli-local-20261030.json), [CI440 named 성공/whole 실패](./ci440-go-proxy-first-failure-20261030.json); cold·whole graph·Node/CLI/refusal | 현재 source ordinary·최종 matrix. CI440 뒤 Go 취득 실패를 whole 성공으로 쓰지 않음 |
| bunx/bun x 명시 --bun | [G3 local](./p6g3-bun-readonly-program-local-20261030.json); 별도 Bun identity·고정 JS recipe·앱2/CLI3/controlled12종 | CI441 single full12 parent 포함 새 hosted·최종 matrix; Bun-only shebang/TS entry 자동 선택·일반 native 동등성은 구현 완료가 아님 |

## 목표 gap과 현재 구현 상태

Client09 §1의 UVx 목표는 extras/추가 dependency도 전체 set에 포함한다고 명시한다.
초기 P6 UV lane은 package-only였으며 extras/추가 package를 거부했다. H1은 기존 owner에
root extras를 연결했지만 독립 추가 package는 H2의 별도 typed 실행 set 모델이 필요하다.
이는 기존 bounded lane의 실패가 아니지만 원래 목표의 미구현이다. 현재 한계를 새로
승인된 scope 축소나 deferral로 바꾸거나, named 성공으로 이 gap을 닫지 않는다.

Root extras H1은 `IMPLEMENTED=YES / WIRED=YES`, named actual/Go local qualification PASS이며
[H1 receipt](./p6h1-uvx-root-extras-local-20261030.json)가 범위를 소유한다. 독립 --with H2는
`IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO`다.
`MISSING`: 기존 requirement/UV native 선택/독립 wheel·whole graph/현재 승인/console/runtime
owner가 extras와 명시 추가 dependency를 보존하는지 먼저 감사하고 부족한 연결만 구현한다.
새 Core/실행 engine·placeholder나 cache/이전 ALLOW를 통한 우회는 만들지 않는다.
구체 source 지도·읽을 문서·재검토와 첫 표적/실제 검증은 다음 scoped baseline이 소유한다.

## 최종 profile 실행 순서

CI441은 새 G3 공통 실행 경계의 중요한 hosted checkpoint로 끝까지 소비한다. 새 push로
취소하지 않는다. 알려진 목표 gap의 필요한 source 수정이 남았으므로 최종 CUDA3를 지금
시작해 후속 코드 뒤 반복하지 않는다. 해당 구현과 의미 있는 로컬 검사들을 모은 뒤
최종 frozen source에서 ordinary/CPU·cu126·cu130·cu132를 각각 실제 판정한다.
기존 [profile baseline](./p7-final-profile-baseline-20261030.md)의 한도/Required/정리와
CUDA-only를 ordinary 성공으로 쓰지 않는 경계는 유지한다.

최종 지원표·help/예시·검증 결과·정리가 연결되기 전 P2/P3/P4/P5/P6/P7 whole acceptance는
OPEN이다. 서명 배포·공식 채널 이름/publisher binding·main merge/publication은 M13의 별도
미완료이며 이 표로 완료·승인되지 않는다.
