# P7 — 명령 목표와 실제 acceptance 대조

검증 소스는 `ec1c260804e35915a4dac78e1720c2fc618264ff`, 전체 tree는
`84e70a9ea0ec50afa3d41d40bec8839b6811156f`다. 아래 현재 지원 lane의 ordinary/CPU와
세 CUDA profile은 같은 소스에서 Required10SUCCESS다. 최종 기록은
[qualification](./p7-final-qualification-20261031.json)이 소유한다.
명령·옵션·플랫폼의 정확한 지원 경계는 [client09](../../../user-journey-cli-ia/09-package-client-quarantine.md)를 따른다.

| 최종 profile | 실제 결과와 근거 | CPU full / 선택 CUDA full |
| --- | --- | --- |
| ordinary/CPU | [CI442](./ci442-ordinary-cpu-qualification-20261031.json), all10; Linux124 unique PASS/138m45s | 302.91s / 해당 없음 |
| cu126 | [CI443 attempt2](./p7-cu126-final-qualification-20261031.json), retained7+새3/all10 | 264.94s / 1208.54s |
| cu130 | [CI444 attempt2](./p7-cu130-final-qualification-20261031.json), retained8+새Mac/Required/all10 | 300.39s / 1145.81s |
| cu132 | [CI445](./p7-cu132-final-qualification-20261031.json), all10 | 292.29s / 1078.54s |

CUDA-only 실행은 ordinary의 client 회귀를 대체하지 않는다. 각 profile은 실제 checkout,
전체 로그, CPU+선택 CUDA full, exact qualification input과 helper/최종 정리에 연결된다.
검사 전용 NumPy와 `triton.profiler.viewer`의 `NOT_ATTESTED` 경계는 각 receipt를 따른다.
최종 변경 묶음의 [docs/security](./p7-final-documentation-checks-20261031.json)는 PASS다. 네 profile 종료 뒤 한 묶음으로 적용했으며 새 profile CI를 만들지 않았다.

| 목표군 / 현재 계약 | 기존 named 구현 근거 | 현재 소스의 검증·지원 경계 |
| --- | --- | --- |
| UV pip install/upgrade/sync — §8 | [P1 scoped acceptance](./p1-uv-acceptance-20261023.json) | CI442 existing install/sync·기존 소비자 및 위 네 profile; 기존 venv/public wheel/explicit empty·whole 승인/원복 |
| PNPM project add/install/update/adopt — §9 | [P2/P3 matrix](./p2p3-final-matrix-20261026.md) | CI442 project/peer/ownership/CLI·whole transaction와 위 matrix; 임의 workspace/registry 의미로 확대하지 않음 |
| Bun project add/install/update/adopt — §10 | 같은 matrix와 [CI413](./p3-grouped-bun-ci413-20261026.json) | CI442 Bun project/current runtime/CLI/원복; Bun과 Node identity 분리 유지 |
| UV project add/sync/lock --upgrade | [P4-K3](./p4k3-uv-project-lock-cli-local-20261026.json) | CI442 project/lock-only/CLI; lock-only 완료는 설치 완료가 아님 |
| UV tool install/upgrade/adopt | [CI417](./ci417-uv-tools-hosted-qualification-20261027.json) | CI442 fresh/managed/fixed native recipe·현재 승인; 다른 native recipe/일반 기본값 호환을 뜻하지 않음 |
| pipx install/upgrade/adopt | [CI423](./ci423-pipx-current-hosted-qualification-20261029.json) | CI442 고정 UV backend/전체 원본/script ownership/관리형 upgrade; inject/ensurepath는 지원 operation 아님 |
| npx/npm exec | [CI429](./ci429-node-cli-hosted-qualification-20261029.json) | CI442 public whole set/Node entry/CLI/controlled12·refusal; stdin/TTY/파일 반출/임의 child는 별도 |
| uvx/uv tool run | [CI437](./ci437-close-entry-uv-hosted-qualification-20261030.json), [H1](./p6h1-uvx-root-extras-local-20261030.json), [H2](./p6h2-uvx-supplemental-roots-local-20261031.json) | CI442 H1 single full8/300.66s, H2 CLI3/123.29s·native2/47.68s; root extras/독립 --with 전체 선택·fresh 승인/정상0·typed2/충돌 거부 |
| pipx run | [CI438](./ci438-pipx-current-hosted-qualification-20261030.json) | CI442 현재 앱3/CLI6·고정 UV backend/기본·명시 console; 다른 backend/module entry는 현재 지원 아님 |
| pnpm dlx/pnx/pnpx | [CI439](./ci439-pnpm-current-hosted-qualification-20261030.json) | CI442 cold/current 앱2/CLI3/refusal2; standalone aliases이며 pnpm pnx subcommand를 만들지 않음 |
| bunx/bun x 기본 Node | [G2](./p6g2-bun-current-cli-local-20261030.json), [CI440 named](./ci440-go-proxy-first-failure-20261030.json) | CI442 cold/whole graph/current Node/CLI/refusal; CI440의 후속 Go 실패를 whole 성공으로 쓰지 않음 |
| bunx/bun x 명시 --bun | [G3](./p6g3-bun-readonly-program-local-20261030.json) | CI441 named single12와 CI442 current 앱/CLI/controlled12; 고정 JS recipe의 별도 Bun identity만 검증 |

## 현재 지원과 미구현 범위

H1 root extras와 H2 독립 `--with`는 기존 owner·whole 검사/현재 Policy/records·readonly
실행에 연결됐다. H2는 실제 독립 root의 원래 graph를 보존하며 가짜 edge/InstallContext를
만들지 않는다. 최종 source의 CLI 옵션은 client09 현재 lane과 동일하며 과거 package-only
거부 문구는 제거했다. Named local만으로 hosted를 주장하지 않고 위 CI442 whole cohort를 사용한다.

위 표는 ec1의 과거 source checkpoint다. 후속 G4는 Bun-only shebang·TypeScript entry와
선언에 따른 runtime 선택을 구현·연결했고 실제 public 앱2/CLI3·기존 Node2를 로컬 검증했다.
[G4 지도](./p6g4-bun-entry-runtime-baseline-20261031.md)가 새 후보를 소유하며 새 hosted와
기능 acceptance는 아래 CI446 근거로 닫혔다. 선언 없는 TS는 명시 `--bun`이 필요하다.

모든 native 옵션/recipe의 동등성은 원래 client09의 약속이 아니다. stdin/TTY·생성 파일
반출은 별도 acceptance이며 현재 stdout/stderr 우선 lane에 포함하지 않는다. Arbitrary
child/held pipe는 현재 안전한 거부/불완전 처리의 검증 대상이다. 이 경계를 필수 미구현과
혼합하거나 승인된 유예·전체 지원으로 바꾸지 않는다. `uv run`, `pnpm exec`, `bun run`, runtime provisioning,
임의 private/index/global 설치 등의 명시 거부도 client09 계약대로 유지한다.
네 profile 성공은 이 목록의 구현 증거나 macOS sandbox 지원이 아니다.

## 2026-10-31 기능 우선 acceptance 재대조

이름/배포 준비를 기능 작업의 blocker로 두지 않는다. 아래는 원래 좁은 계약의 실제
명령·검사·원복 목표와 source evidence를 대조한 판정이다. 초기 scoped 범위의 완료를
모든 native 옵션·미지원 ABI/registry의 완료로 확대하지 않는다.

| 책임 / 실제 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| P2/P3 project add/install/update·peer/ownership/adopt·실패 원복 — 위 P2/P3 matrix·CI413, ec1 CI442–445 | YES | YES | YES(ec1 scoped) | YES(ec1 scoped) | 기존 matrix가 남긴 최종 profile은 ec1에서 확보됨; 새 기능을 다시 구현하지 않음 |
| P4 project add/sync/lock-only·UV/pipx install/upgrade·고정 native 인계 — 위 K3/CI417/CI423, ec1 CI442–445 | YES | YES | YES(ec1 scoped) | YES(ec1 scoped) | 일반 native 기본값·다른 recipe는 현재 명시 허용 범위가 아님 |
| P5/P6 기존 npx/uvx/pipx/pnpm/Bun JS·alias·전체 source/현재 승인·출력/오류·악성/불완전 거부 — 위 각 receipt·ec1 CI442–445 | YES | YES | YES(ec1 scoped) | YES(ec1 scoped) | 공유 entry reader의 새e3 회귀도 CI446에서 확인 |
| P6-G4 선언 Bun/TS byte proof·runtime 선택·실제 current 앱/CLI·기존 Node 회귀 — G4 지도와 local 원본 | YES | YES | YES(CI446 current e3) | YES(bounded) | 없음; 새 앱2/CLI3·기존 회귀·정리·전체10 원본 확인 |
| P7 새 후보의 기능 acceptance | YES(기존 qualification owner) | YES | YES(CI446 ordinary) | YES(original bounded scope) | 없음; 이전 CUDA3는 ec1/불변 소비자 범위로 유지하며 새e3 네 profile 성공을 주장하지 않음 |

생성 파일/TTY 등의 별도 확장을 이번 좁은 실행 계약의 누락으로 소급하지 않는다.
필요한 새 동작이 실제로 발견되면 exact gap으로 다시 판정한다. 새 후보의 필수 기능
acceptance가 닫히기 전에는 이름 선택·배포 단계로 넘어가지 않는다.

## 인계와 남은 작업

[CI446](./ci446-bun-declared-entry-hosted-qualification-20261031.json)은 sourcee3b2b70/실제checkout
전체tree동일·Required10SUCCESS/Linux126/noSKIP·새 Bun 앱2/CLI3·기존 소비자·정리와 전체 원본을
연결한다. 원래 bounded P2–P7 기능 목표는 완료했다. 초기 계약의 명시 제외/별도 확장과
M13의 새 서명 배포물·공식 채널 이름/publisher binding·채널 구현/검증·main review/publication은
분리한다. 같은 source의 full/native/CPP/ordinary/CUDA를 다시 실행하지 않는다.

첫 실패는 [CI443 Read](./ci443-cu126-first-failure-20261031.json),
[CI443 Mac build](./ci443-macos-first-failure-comparison-20261031.json),
[CI444 Mac wholejob/log 부재](./ci444-macos-first-failure-20261031.json)에 남는다.
한 번의 명시 operational 재검증 성공을 과거 하위 원인의 수리로 쓰지 않는다.
재검증은 실패 작업만 요청했고 기존 성공 작업은 source/실행시각/원본과 결합해 보존했다.
