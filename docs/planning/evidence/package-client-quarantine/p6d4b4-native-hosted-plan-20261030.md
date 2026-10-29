# P6-D4B4 — current native source/console hosted 연결 감사

Baseline dc029d9891e94a0d64a2255e24625b27fbd86512/tree242bb537/3095tracked·runtime1379다.
단일 active work item은 M13-004다. B2 exported query/CPP132·UV195와 B3 native source/console
DATA binding은 named local qualified다. CI434 old d81의 Linux104SUCCESS/macOS120s·RequiredFAIL,
monitor 자연0/next none이며 원인은 미확정이다. Source 전 실제 Git/Queue와 아래 owners를 대조했다.

Read: PROJECT/Queue/docs routes와 lessons INDEX·artifact L01/L02·sandbox L03/L08/L12/L18
(tmpfs P5-C3b 포함)·evidence L16·ci L05/L06, Architecture001, client09§3·plan20§5,
engineering04 concurrency/retention/config, M12 예방4/5·운영v0.5§15, C2 wiring plan과 현재
ci-uv-tools.sh/uv_tools_workflow.go·test/heliopause-ci.yml·helper install/start/stop,
실제 UV public SourceBinding/NativeCensus 부모와 B2/B3 local/원본이다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| whole source/startup/RO/query/native console DATA | YES | YES (actual local) | YES (named local) | YES (named local) | hosted consumer |
| protected sandbox tagged client/helper·current step | YES | YES | YES (CI434 Linux) | YES (old named) | current step 끝에 기존 install/stop/start 재사용 |
| closed profile/command·exact log guard·negative tests | YES | YES (기존7consumers) | YES (기존) | YES (기존) | uv-run-source profile와 exact 부모2/marker3/noSKIP/error 순서 |
| 새 B2/B3/current source hosted | NO | NO | NO | NO | 로컬 wiring qualification 후 묶음 FF push/실제20min GET |
| production UVx resolver/adapter·generic equivalence/aliases/CLI/final | NO | NO | NO | NO | 다음 별도 acceptance |

## Source 전 exact map·두 번째 검토

기존 구현을 대체하지 않는다. 네 existing files만 수정하며 runtime membership1379는 유지한다.
- scripts/ci-uv-tools.sh: closed enum/fallback exclusion에 uv-run-source를 넣고 ordinary-user
  확인 뒤 sandbox package의 exact 두 부모를 세 명시 flag로 조기 dispatch한다. 기존 DATA
  exporter와 다른 소비자는 유지하고 불필요한 Python interpreter DATA export는 만들지 않는다.
- scripts/check/uv_tools_workflow.go: 같은 closed command/scope validator와 current exact run
  owner에 native consumer를 추가한다. 기존 bootstrap client의 Python7 소비 후 기존 helper를
  canonical install에서 stop하고 current sandbox client로 rebind/start한다. 새 protocol은 없다.
  두 exact top parent·unique counts/noSKIP와 source/census/console 세 bounded marker를 확인한다.
  Marker는 test coverage이며 Policy/실행 authority가 아니다. command→cat 원래 오류를 보존한다.
- scripts/check/uv_tools_workflow_test.go: profile fallback/flags/parent/scope·기존 helper/client
  유지 및 native parents/세 marker/noSKIP/error guards의 누락·완화 negative를 확인한다.
- .github/workflows/heliopause-ci.yml: 기존 current execution step 끝에 위 exact body를 연결한다.
  새 job/runner/permissions/timeout/Required dependency/coverage 변경은 없다. 기존104와 새2의
 106 top-level을 예상하며 CPP는132 named/UV195다. Linux120min은 유지한다.

Canonical format/workflow/Quick·checker 전체 Race/docs/security와 Bash syntax, 저장된 실제
B2/B3/D4A 로그의 inert DATA guard를 먼저 확인한다. 새 source는 workflow/checker뿐이므로
동일 native actual과 CPP build/CPU E2E는 로컬에서 반복하지 않는다. 중요한 B2+B3+wiring 묶음을
한 번 normal FF push하고 actual source/run/start와 실제20분 GET worker를 등록한다. CI434
107m18s에 새 두 부모 local 약55s를 더한 약108분은 추정이며 macOS 과거 원인 해결은 아니다.
실행 중 새 push 없이 production resolver/current adapter 감사·구현을 계속한다. 완료된 사본만
원본/참조0/errors0 뒤 정리하며 shared caches·실패 원본은 보존한다.

## Local wiring qualification

[Receipt](./p6d4b4-native-hosted-local-20261030.json)은 runtime1379/기존1376 bytes 유지와 canonical8·Quick80.247s/전체 checker Race61.829s·저장된 실제 로그 DATA26개 PASS를 확인한다. Exact 두 부모·세 marker/noSKIP/unique/command-before-cat와 helper rebind를 기존 owner가 검사한다. 새 native/CPP/CPU actual을 로컬에서 반복하지 않았다. 다음 중요한 B2+B3+wiring 묶음을 한 번 push하고 hosted 결과/20분 GET을 별도 기록한다.
