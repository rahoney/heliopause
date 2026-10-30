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

일반 native 옵션/recipe 동등성, Bun-only shebang·TypeScript entry 자동 선택, stdin/TTY,
생성 파일 반출·arbitrary child/held pipe 지원은 구현·검증 완료가 아니다. 이를 승인된 유예나
전체 지원으로 바꾸지 않는다. `uv run`, `pnpm exec`, `bun run`, runtime provisioning,
임의 private/index/global 설치 등의 명시 거부도 client09 계약대로 유지한다.
네 profile 성공은 이 목록의 구현 증거나 macOS sandbox 지원이 아니다.

## 인계와 남은 작업

이 표는 현재 문서화된 lane의 source/profile qualification을 연결한다. P2–P7 전체 목표의
일반 native 동등성이나 모든 미구현 capability acceptance를 닫지 않는다. 서명된 새 배포물,
공식 채널 이름/publisher binding·채널 구현·main merge/publication도 별도의 M13 미완료다.
현재 동일 소스의 full/native/CPP/ordinary/CUDA를 다시 실행하지 않는다.

첫 실패는 [CI443 Read](./ci443-cu126-first-failure-20261031.json),
[CI443 Mac build](./ci443-macos-first-failure-comparison-20261031.json),
[CI444 Mac wholejob/log 부재](./ci444-macos-first-failure-20261031.json)에 남는다.
한 번의 명시 operational 재검증 성공을 과거 하위 원인의 수리로 쓰지 않는다.
재검증은 실패 작업만 요청했고 기존 성공 작업은 source/실행시각/원본과 결합해 보존했다.
