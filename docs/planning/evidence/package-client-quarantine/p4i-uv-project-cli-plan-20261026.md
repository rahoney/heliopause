# P4-I public UV project add/sync CLI

Baseline 21312f8/tree67d50f384e4272fe6b583c8ad286b26f5e6efbac/2526files.
Read: Queue/plan20/client 설계09, P4-H qualified installed backend와 원본/교정,
lessons INDEX/artifact L15/ci L06/evidence L16, CLI presentation/descriptor preflight,
existing UV pip/Bun/PNPM command binding와 generic bounded control I/O를 확인했다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Native project add/sync/current approval/atomic publication | YES | YES(programmatic composition) | YES(P4-H local actual) | YES(named installed backend) | public CLI consumer |
| Shared CLI descriptor/preflight/presentation/project report | YES | YES(existing clients) | YES(existing scopes) | YES(existing scopes) | UV project typed request/help/routing |
| Bounded descriptor-relative control preflight | YES | YES(Bun/PNPM) | YES(existing tests) | YES(existing scope) | generic limit/optional parameters로 same body 재사용 |
| UV project public command/platform/bootstrap binding | NO | NO | NO | NO | add/sync options/parser/service와 actual CLI |
| Native project adoption/lock-only/tools | NO | NO | NO | NO | 이후 같은 P4 acceptance |

## Exact map·두 번째 검토

Compiled client descriptor에 UV project ID와 `uv add <name-or-requirement>`, `uv sync`를 추가한다.
Current directory 또는 `--project`의 virtual project를 읽고, `--target` 또는 project/.venv의
기존 fixed-runtime venv만 받는다. Runtime provisioning은 지원하지 않는다. Explicit target
root와 source controls의 overlap/alias·중복/unknown options·unsupported manifest/source를
runtime 준비 이전에 거부한다. 기존 uv pip descriptor와 routing은 유지한다.

CLI의 기존 bounded/single-link/nonblocking control reader body를 limit/optional 매개변수의
generic helper로 이동하고 기존 NPM wrapper도 그 body를 사용한다. Package grammar와 source
identity는 reader에 넣지 않는다. 원래 body를 mapped 비교하고 Bun/PNPM consumers를 함께
검사한다. Original uv.lock은 H의 error-only historical reader로 검증하고 현재 selected
authority로 사용하지 않는다. Actual held source/target/ownership 재확인은 H backend가 소유한다.

Bootstrap의 explicit UV project lane은 기존 runtime/executor/observer lifecycle 아래 H의
실제 factory를 구성하고 `uv add/sync`에 bind한다. Legacy Python pip-install binding을
project 의미로 바꾸지 않는다. CLI 결과는 기존 complete project report/progress/locale/JSON과
실제 committed Manifest/SBOM을 사용한다. 새 Policy/승인 상태/daemon은 없다.

두 번째 검토에서 default target 의미·existing runtime 요구, no arbitrary argv fallback,
preflight→runtime 순서, immutable current request와 current-source reinspection, named original
adoption 거부·empty retirement·실제 결과 참조를 대조했다. Lock-only는 이 installed result에
포함하지 않는다. Canonical profiles/CLI·bootstrap affected tests와 Race, 영어/한국어 도움말,
actual fresh/version change/same-version/add/sync/empty/거부·Host 종료를 확인한다. 작은 helper별
새 full CI는 없으며 최종 affected source/profile checkpoint로 묶는다.

## Named local 결과

[Public CLI 영수증](./p4i-uv-project-cli-local-20261026.json)은 최종 Quick43.517s/security8.958s/
docs0.251s/full CLI·bootstrap Race23.232s와 actual327.758s를 연결한다. New CLI168.49s의
일곱 성공 요청/소유권 거부 하나와 기존 UV pip sync156.66s/여섯 요청/native original 인계가
PASS다. Explicit external target은 별도 actual10.494s/test7.98s에서 absent default .venv와
다른 VIRTUAL_ENV보다 우선하며 Manifest target이 정확함을 확인했다. 두 actual의 noSKIP/
helperwait0/Host 전후 동일과 실제 records/controls·fresh entry inspection/user data 보존을 확인했다.
Final v4는 v3와 제품/기존 actual source가 같고 별도 target test만 추가했다. 최초 세 failure의
원본을 보존하며 diagnostic 억제나 필수 입력 완화는 없다. 기존18 owner는 bytes 불변이다.
Native project adoption/lock-only/tools·whole P4/remote는 미완료다.
