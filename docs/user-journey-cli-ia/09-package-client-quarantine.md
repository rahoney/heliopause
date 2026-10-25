# Package client / tool execution quarantine — 설계

상태: **P1 UV pip COMPLETE / QUALIFIED / ACCEPTANCE_CLOSED; P2 pnpm backend/CLI/native·실제 실패 named 검증, 최종 whole qualification 미완료; P3 Bun cache/transaction/CLI/native 인계·명시 runtime 경계 named local 검증, whole native runtime 검사 연결 중; P4..P7 미착수**. 2026-10-21 사용자 요청으로
uv pip·pnpm·Bun과 npx·uvx·bunx·pipx를 하나의 확장 범위로 계획한다.
[실행 계획·baseline·수정 지도·재검토](../planning/20-package-client-quarantine-plan.md)가
구현 순서와 검증을 소유한다. 기존 [업데이트 검역](./06-update-quarantine.md)을
대체하거나 새 도구 지원이 이미 완료됐다고 표시하지 않는다.

## 1. 제품 경계와 명령 범위

제품은 외부 artifact의 설치·업데이트 및 실행 전 검역을 제공한다. npm/PyPI는
source이고 pnpm/Bun/uv/pipx는 client다. Client 이름을 새 registry identity로
사용하지 않는다. 같은 registry라도 resolver 선택, lock, 배치와 실행 의미는 다르다.

아래는 구현 목표 구문이며 **현재 실행 가능한 명령 목록이 아니다**. Version,
선택 대상, target, entry point가 명확해야 한다. 지원하지 않는 option은 mutation과
네트워크 이전에 거부한다. 임의 argv를 원래 프로그램에 전달하는 fallback은 없다.

| Client | 이번 확장에 포함하는 흐름 | 별도 의미와 경계 |
| --- | --- | --- |
| uv pip | `helox uv pip install <spec>` / `install --upgrade <spec>` / `sync <requirements>` | pip 호출 alias가 아닌 uv resolver; upgrade와 exact sync의 삭제 대상도 transaction에 포함 |
| uv project | `helox uv add <spec>` / `sync` / `lock --upgrade` | pyproject.toml·uv.lock·환경의 독립 상태. Lock-only는 설치 완료가 아니며 나중 sync도 동일 graph 검사 필요 |
| pnpm | `helox pnpm add <spec>` / `install` / `update [name]` | pnpm-lock.yaml·virtual store·peer graph·내부 symlink/hardlink 배치 |
| Bun | `helox bun add <spec>` / `install` / `update [name]` | bun.lock·선택된 linker·lifecycle 정책. Bun runtime 사용은 별도 실행 identity |
| npx | `helox npx <package> [args]`, 명시 package/command 형식과 `helox npm exec` | package의 bin과 실행 argv를 분리. 로컬 검색·cache도 검역 우회 불가 |
| uvx | `helox uvx <tool> [args]`, `helox uv tool run` | package와 command가 다르면 명시 `--from`; extras/추가 dependency도 전체 set에 포함 |
| bunx | `helox bunx <package> [args]`, `helox bun x` | Node shebang 기본 실행과 명시 Bun 실행을 구분하여 runtime admission |
| pipx | `helox pipx install <spec>` / `upgrade <name>` / `run <tool> [args]` | 영구 tool venv·entrypoint 소유권과 임시 실행은 다른 작업; backend를 실행마다 자동 탐지하지 않음 |
| 같은 도구의 연관 진입점 | `helox uv tool install/upgrade`, `helox pnpm dlx` | 같은 typed operation에 연결. pnpx/pnx alias는 선택 버전의 실제 의미와 함께 검증 |

`uv run`, `pnpm exec`, `bun run`은 기존 프로젝트 실행·자동 sync·shell script까지
포함하므로 이번 초기 실행 범위에서 지원하지 않는다. 잘못된 자동 다운로드 우회가
되지 않게 명시 거부한다. Publish, bundler, test runner, runtime 자체 설치/업데이트,
임의 shell 실행, system Python/global 설치도 포함하지 않는다. 이미 지원하는 Go/Cargo
기능을 제거하지 않는다. 모든 native 옵션 호환을 약속하지 않는다.

첫 구현은 단일 프로젝트·기존 지원 registry/ABI·명시 target에서 시작한다. Workspaces,
monorepo, editable/local/VCS/direct URL dependency, private credentials, global install,
임의 index와 runtime provisioning은 capability가 별도로 검증되기 전에는 거부한다.
각 client lane을 완료할 때 실제 허용 option·selector·lock format·platform 표를
고정하고 help/examples와 같은 descriptor에서 노출한다. 위 목표 전체의 미완료는
실행 계획에서 보존하며 제한된 첫 lane 성공으로 전체 완료를 선언하지 않는다.

공식 의미: [uv pip 호환성](https://docs.astral.sh/uv/pip/compatibility/),
[uv tools](https://docs.astral.sh/uv/guides/tools/),
[npx](https://docs.npmjs.com/cli/v11/commands/npx/),
[pnpm layout](https://pnpm.io/symlinked-node-modules-structure),
[pnpm dlx/pnx](https://pnpm.io/cli/pnx),
[Bun install](https://bun.sh/docs/pm/cli/install),
[bunx](https://bun.sh/docs/pm/bunx), [pipx](https://pipx.pypa.io/stable/).
공식 문서 최신판이 선택한 버전과 항상 같지 않으므로 exact tag의 help/fixture도 대조한다.

## 2. 공통 흐름과 책임

```text
CLI의 명시 명령/옵션 검증
→ source + client + intent + frozen project/input context
→ 고정된 client로 격리 resolution (typed network policy)
→ exact graph/artifacts/lock/runtime/entrypoint 확정
→ 기존 acquisition·verification·static·dynamic·entry/set Policy
→ ALLOW한 동일 bytes를 staging에 보관, 반출 직전 digest 재검증
→ install/update: guarded promotion / run: 별도 제한된 sandbox 실행
→ 결과·Evidence·원상복구/정리 확인
```

기존 Artifact Adapter와 Policy가 검역 권한을 유지한다. Client별 resolver adapter는
원래 도구의 선택 의미를 기존 graph로 변환하고 알 수 없는 schema/edge를 거부한다.
인증된 graph 없이 lock 파일 이름만 바꾸거나 uv 명령을 pip로, pnpm을 npm으로 몰래
대체하지 않는다. npm archive·wheel 검사기는 재사용한다. npm과 Bun의 lifecycle
기본값 차이는 검사 실험과 승인 manifest에 남기며 skip을 완료로 바꾸지 않는다.

CLI command descriptor는 client/alias/intent/허용 옵션/help key를 소유하고 bootstrap은
명시적으로 factory를 연결한다. Core가 Cobra/uv/pnpm/Bun을 import하지 않는다.
하나의 작은 typed registry에서 실제 지원 항목만 선언하며 runtime plugin 로딩이나
범용 YAML workflow 엔진을 만들지 않는다. 다른 책임의 코드를 한 거대 switch로
옮기지 않는다. Version·format·capability의 owner와 교차 검증을 분명히 둔다.

## 3. 일회성 실행의 추가 보안 계약

설치 검역 통과는 모든 프로그램 동작이 안전하다는 보증이 아니다. 기존 staging은
**보관 전용**이다. 실행은 staging 밖의 별도 sandbox에 검증된 파일을 복사·재확인하고
기존 supervisor/observer·완료 원장으로 통제한다. 검사 뒤 Host에서 `npx <name>` 등을
다시 호출하거나 일반 PATH/cache에서 실행 파일을 찾지 않는다.

- Artifact identity, 전체 의존성, 정확한 bin/console entrypoint, interpreter/runtime,
  argv와 source/policy fingerprint를 실행 요청에 연결한다. Package 이름과 command
  이름이 다르거나 여러 bin이 있으면 ambiguity를 자동 추측하지 않는다.
- 임시 venv는 의존성 격리일 뿐 보안 sandbox가 아니다. uv/pipx backend와 Python,
  bunx의 Node/Bun을 고정한다. Artifact shebang의 임의 interpreter를 Host에서 실행하지 않는다.
- 기본 실행은 외부 네트워크·Host HOME/secret/credential·실제 프로젝트 mount 없이
  유한 CPU/memory/time/output 한도 내에서 수행한다. 필요한 프로젝트 입력은 명시
  선택된 snapshot만 전달한다. 프로그램이 후속 npm/pip 등을 실행해 새 dependency를
  요구하면 실패 이유를 보존하고 별도 검역 intake로 돌려야 한다. 자동 network 허용은 없다.
- 생성 파일이 필요한 도구는 private output workspace를 쓴다. 출력은 기존 승인 artifact의
  digest로 포장하지 않는다. 별도의 경로/link/크기/소유권 검사와 충돌 방지 transaction을
  거쳐 명시 target에 반출하며 코드 출력은 독립 artifact intake가 필요하다. 이 경로의
  계약과 검증이 닫히기 전에는 파일 생성형 도구를 지원한다고 표시하지 않는다.
- 실행 stdout/stderr는 신뢰되지 않는 데이터다. 제어문자/OSC·크기 폭주·secret 노출을
  방어하고, HAA 요약·JSON과 혼합하지 않는다. Structured result에는 quarantine
  status, policy, execution status/exit, output promotion, cleanup을 따로 기록한다.
  Artifact의 exit0/stdout marker는 검사 완료 authority가 아니다.
- 일반 실행 argv의 `--` 경계와 HAA global flags를 구분한다. Shell 문자열 eval,
  `npx -c`, 임의 interpreter override는 초기 scope에서 거부한다. stdin/TTY 전달은
  명시 capability이며 interactive 지원 전에는 필요한 tool을 조용히 headless로 실행하지 않는다.
- SIGINT/timeout/observer fault에 원래 오류를 보존하고 child/helper/container까지
  drain한다. 정리 불명확을 성공으로 바꾸지 않는다. 재실행은 오염된 cache를 승인으로 보지 않는다.

Host 자산을 노출하거나 임의 Host 실행을 허용해야만 되는 호환성 요구는 기존
[격리 계약](../threat-model/02-isolation-and-inspection.md) 변경이므로 구현 편의로
넓히지 않는다. 제한을 설명하거나 별도 결정으로 남긴다.

## 4. 설치·업데이트·cache 일관성

기존 transaction lock·ownership·rollback을 재사용한다. Npm project transaction의
package-lock 전제를 pnpm/Bun에 적용하지 않고, 각 client layout adapter에서 frozen
manifest/lock/tree/approval snapshot을 연결한다. pnpm 내부 link는 승인된 owned tree
안의 정확한 대상만 허용하며 foreign store 공유 hardlink로 승인 이후 수정이 전파되지
않도록 private copy/reflink와 반출 동일성을 검증한다. 공통 symlink 거부를 전역 완화하지 않는다.

Python은 같은 venv destination/entrypoint ownership을 유지한다. uv sync의 삭제,
pipx tool metadata·exposed scripts, uv project controls도 하나의 commit/rollback
단위로 포함한다. Native/unmanaged 환경은 기존 explicit adoption 경계에 따라 별도로
지원하고 알 수 없는 파일을 삭제하지 않는다. 실패·동시 mutation·disk full·취소 뒤
기존 설치가 유지되어야 하며, 복구 불명확이면 journal을 보존하고 새 mutation을 막는다.

공유 다운로드 cache, 승인 staging, 임시 실행 환경은 신뢰와 수명이 다르다. Cache hit도
exact identity/digest·현재 policy/runtime·ownership 재검증이 필요하다. 검사 결과를
무기한 ALLOW로 재사용하지 않는다. User cache·다른 프로젝트·진행 중 session은
정리 대상이 아니다. 완료된 task-owned payload/venv/store/image/volume/cache는 참조
부재를 확인하고 제거하며 작은 manifest·결과·실패 원인만 남긴다.

## 5. 버전과 지원 표시

버전 owner는 기존 [runtime maintenance](../engineering/05-runtime-upgrade-maintenance.md)를
확장한다. Runtime lock의 exact version/배포물 digest와 generated identity를 사용하며
quality-tool lock에 product client를 넣지 않는다. npx는 선택 npm, uvx는 선택 uv,
bunx는 선택 Bun의 identity를 공유한다. Aliases마다 별도 version pin을 만들지 않는다.

공식 최신판 발견 → release notes/engine·ABI·schema·license·보안 검토 → exact 후보
metadata/asset authenticity → frozen pin/image build → parser·negative·실제 검역·
transaction/실행 qualification → 지원표/서명 배포물 갱신 순으로 진행한다. 실행 중
`latest` 설치, ambient PATH version 탐지 또는 사용자 runtime 자동 교체는 없다.

[버전 관측 영수증](../planning/evidence/package-client-quarantine/version-review-20261021.json)은
최신 후보 발견일 뿐 채택/호환성/취약점 부재 증거가 아니다. 초기 execution 환경은
기존 Linux/WSL2 amd64 신뢰 환경에서 검증한다. macOS CLI 출력 성공만으로 macOS
sandbox 지원을 선언하지 않으며 다른 OS/CPU/ABI는 실제 qualification 전까지 미지원이다.

## 6. UV P1의 실제 selection / metadata 경계

선택한 uv0.13.4는 install --report를 제공하지 않는다. Native compile의 pylock1.0은
version과 복수 wheel 대안을 보존하며 dependency edges를 생략한다. Flat lock만으로
complete graph/ALLOW를 만들지 않는다. No-index/no-build private materialization의
표준 WHEEL 태그를 bounded snapshot으로 읽고, 같은 name/version/tag를 가진 lock
후보가 정확히 하나일 때만 선택한다. 동일 태그로 여러 URL이 남으면 추측하지 않는다.
선택 파일 전체를 기존 InspectWheel로 검증하고 인증된 Requires-Dist에서 graph를 만든다.

UV resolver의 독립 registry cross-check는 canonical project/version에서 유도한
`https://pypi.org/pypi/<project>/<version>/json`의 전체 exact-release 응답을 사용한다.
Response info identity와 모든 release file의 identity·URL·SHA256·size·Requires-Python·
yanked를 기존 정규화 경계로 검증한다. 전체 프로젝트 Simple 응답을 잘라 놓고 그 API를
검증했다고 주장하지 않는다. 기존4MiB/1024 file 한도와 public origin 제한을 유지한다.
Pip/PyTorch의 metadata 경로는 이 UV 경로와 별도로 유지된다. 자료 경계만의 구현과 실제
CLI/runtime/검역 qualification을 구분하며, 현재 검증한 지원 범위는 §8을 따른다.


## 7. UV runtime 공급의 provenance 경계

기존 HAA release 검증과 별개로 UV publisher를 고정 검증한다. GitHub OIDC issuer,
astral-sh/uv repository/owner ID, main의 publish-docker-image.yml builder와 release.yml
caller, workflow_dispatch·github-hosted, source commit과 invocation을 certificate와 SLSA
predicate 양쪽에서 확인한다. User 선택 regex나 trusted publisher 예외를 추가하지 않는다.
공개 Sigstore root와 pinned library는 기존 releaseprovenance owner의 것을 공유한다.

서명 대상은 ghcr.io/astral-sh/uv의 multi-platform index 하나다. 실제 index 전체 bytes의
SHA256과 유일한 linux/amd64 descriptor의 digest/size를 확인하고 child manifest→config의
전체 digest/size·version/source/revision·platform을 연결한다. 미검증 label·live tag·파일명만으로
버전을 채택하지 않는다. OCI index64 entries/1MiB, manifest/config 각64KiB, gzip layer 최대4개·
총 compressed64MiB의 UV 공급 한도이며 기존 artifact resource 정책을 넓히지 않는다.
Metadata/서명 확인만으로 layer bytes/ABI·observer admission/Policy/activation을 승인하지 않는다.


UV의 native 공급 자료 검증은 모든 signed layer의 compressed hash/size와 config diffID를
확인한다. 합계 compressed/expanded 각64MiB·entries32 이내의 reviewed uv/uvx regular
files와 io directory만 허용하고 root0:0/0755·static ELF amd64를 확인한다. 뒤 layer의
UV overwrite·whiteout·links/경로 우회는 거부한다. 도출한 native hash/size는 공급
동일성 자료이며 Host 실행·sandbox compatibility·검역 ALLOW를 의미하지 않는다.

UV 공급의 version/image/provenance/native identity는 runtime lock schema2의 한 owner에서
생성한다. Maintenance의 frozen plan은 source lock·support review·generated identity를 함께
갱신하고 모든 layer를 적용 시 다시 검증한다. Pin 채택과 실제 CLI/runtime 검증은 별개이며
P1은 production UV 호출·격리 admission과 §8의 실제 검역 범위까지 확인했다. Host 활성화나
uv project/tool/run의 지원으로 확대하지 않는다.

## 8. P1에서 검증한 UV pip 지원 범위

실제 지원은 Linux/WSL2 amd64의 고정 Python runtime·공개 PyPI wheel·기존 venv다.
UV 버전·공급 bytes는 runtime lock의 단일 owner를 따르며 Host의 uv/pip를 대신 호출하지 않는다.
[누적 실제 qualification](../planning/evidence/package-client-quarantine/p1-uv-acceptance-20261023.json)이
정확한 소스·네 profile·정리와 한계를 연결한다.

| 명령 | 허용 입력·옵션 | 검역·변경 의미 |
| --- | --- | --- |
| `helox uv pip install <name>` 또는 `<name>==<version>` | 정확한 `name@version`도 허용. `--target`의 절대 venv 경로 또는 활성 `VIRTUAL_ENV` | 호환되는 기존 버전을 유지하고 선택한 전체 wheel을 새로 검사 |
| 위 install의 `--upgrade`/`-U` | 요청 root와 선택된 dependency 갱신 | 변경·동일 버전 모두 새 검사와 승인·held transaction을 거침 |
| 위 install의 `--upgrade-package <root>` | 요청한 bare root 하나, `--upgrade`와 동시 사용 거부 | dependency는 필요한 경우만 변경하고 요청 root를 갱신 |
| update 옵션과 `--adopt` | 지원하는 native UV 기본 공개 wheel 형식 | 기존 wheel의 독립 proof와 전체 소유권 검증 후 인계. 실패 시 기존 target 보존 |
| `helox uv pip sync <source-file>` | 전체 dependency를 exact final version으로 나열한 requirements 또는 strict `pylock.toml`/`pylock.*.toml`; target 규칙은 install과 같음 | 모든 선택 wheel을 새로 검사하고 전체 승인 뒤 제외된 기존 owner를 제거. 사용자 파일 보존 |
| sync의 `--adopt` | 목록에서 제외될 native 원본도 독립 proof 필요 | 일부 원본만 증명해 삭제 권한을 만들지 않음 |
| sync의 `--allow-empty-requirements` | requirements 파일에만 허용 | 기본 blank skip과 구분해 명시 empty 전체 승인·기록·반입. 명시 empty pylock도 같은 승인 경로 |

Sync의 기본 blank는 Run/Evidence/Policy/cache/target 변경 없이 SKIPPED/NOT_PERFORMED다.
Source는 target·trusted 저장소 밖의 bounded regular single-link 파일이며 symlink·ancestry 변경·
동시 source/target drift를 거부한다. Includes·범위·editable/local/VCS/direct URL·private index·
sdist/build·임의 native flags와 system/global 설치는 이 P1 범위에 없다. Native 인계의 생성
scripts·compiled caches·cross-scheme도 거부한다. 기존 다른 command의 지원을 축소하지 않는다.
UV project는 P4이며 `uv run`과 tool/execution·다른 client의 실제 지원으로 확대하지 않는다.

## 9. P2 PNPM의 현재 CLI 검증 범위

P2의 기존 단일 프로젝트 backend와 아래 명령은 named local 검증을 마쳤다.
실제 source/검사·원격 상태는 [Queue](../planning/02-current-work-queue.md)와
[B5c 검증 지도](../planning/evidence/package-client-quarantine/p2c5b5c-pnpm-cli-plan-20261025.md)가
소유한다. Native/unmanaged 인계의 bounded copy lane도 named local 검증을 마쳤으며 전체 P2 remote qualification·acceptance는 미완료다.

| 명령·옵션 | 현재 입력과 동작 |
| --- | --- |
| `helox pnpm add <spec>` | 공개 npm 패키지 이름·버전·범위 하나를 native PNPM으로 선택하고 전체 set을 새로 검역한 뒤 manifest/lock/설치를 함께 반영 |
| `helox pnpm install` | 기존 package.json과 선택적인 일치하는 pnpm-lock.yaml을 사용. 명시한 empty도 전체 승인·records/transaction을 거침 |
| `helox pnpm update [name]` | 기존 직접 의존성 이름 하나 또는 전체를 native PNPM으로 선택하고 manifest/lock/전체 payload를 함께 반영 |
| `--target <directory>` | 기존 프로젝트 디렉터리. 생략하면 현재 디렉터리. Symlink ancestry·alias controls·중복/빈 인수 거부 |
| `--adopt` | 독립 승인된 HAA 관리 payload의 지원하는 설정 변경 또는 아래 원본 증명이 완료된 기존 PNPM 설치를 명시적으로 등록. Manifest/lock 일치 등 입력 검사를 생략하지 않음 |

Linux amd64와 repository의 고정 PNPM runtime을 사용하며 source identity는 `npm`이다.
선택된 모든 archive/peer instance를 검증·관찰하고 실제 Evidence/현재 Policy·frozen layout을
결합한다. Root project scripts는 보존하고 실행하지 않는다. 같은 버전도 새 operation으로
검사하며 같은 content의 native payload와 controls는 기존 inode/mode를 보존한다. 실패하면
원본/state를 rollback하며 불명확한 외부 파일과 recovery journal은 보존하고 재진입을 거부한다.

JSON install/add는 `helox.project-install-result/v1`의 `INSTALL`, update는 기존
`helox.project-update-result/v1`의 `UPDATE`를 사용한다. 검사·Policy와 실제 publication 상태를
구분하며 실제 Manifest/SBOM의 opaque record references를 제공한다. 영어/한국어 전체 문구와
나머지 언어의 기존 결정적 English fallback을 유지한다. Help는 runtime/cache를 준비하지 않는다.

기존 native PNPM 인계는 원래 manifest·lock의 모든 public archive를 독립 취득하고 SRI와
정적 inventory·전체 instance/edge/bin/file closure를 대조한다. 새 선택에서 제거할 원본도
포함한다. 첫 lane은 고정 isolated 형식의 nonempty safe single-link copies다. Shared hardlink,
unsafe mode·혼합 owner·추가/변조 파일·unknown namespace와 corrupt HAA state는 거부한다.
Store 경로는 metadata 형식만 확인하고 외부 store를 열지 않는다. 원본 증명은 독립 state에
기록하며 현재 검역·Policy나 새 승인 cache로 전용하지 않는다. 첫 인계에는 새 clean payload를
반영한다. Native empty layout의 불명확한 metadata는 이 lane에서 미지원이며 기존 fresh/managed
empty 경로는 유지한다. [B5d 지도](../planning/evidence/package-client-quarantine/p2c5b5d-pnpm-native-ownership-plan-20261025.md)가 named 검증 범위를 소유한다.
새 target 생성·global/workspace·local/direct/VCS/다른 registry·임의 native 옵션과 실행 명령은
이 검증 범위에 없다. pnpm dlx는 P6이며 이 CLI 연결로 지원 완료 표시하지 않는다.


## 10. P3 Bun의 현재 설치·업데이트 검증 범위

`helox bun add <spec>`, `helox bun install`, `helox bun update [name]`은 기존 프로젝트
하나의 공개 npm 의존성 전체를 선택·검사하고 `package.json`/`bun.lock`/hoisted payload를
함께 반영한다. `--target`과 `--adopt`의 명시 입력·소유권 원칙은 §4를 따르며 source는
`npm`, client는 `bun`이다. 현재 구현은 Linux amd64의 고정 Bun·Node runtime과 native
copyfile/hoisted 형식이다. Empty project는 실제 `bun.lock` 부재를 보존한다.

선택된 archive 전체와 모든 물리 instance를 검사하며 project 자체의 script는 실행하지 않는다.
Package lifecycle은 같은 고정 Node·Bun runtime을 격리 환경에서 사용한다. Runtime image가
고정돼 있어도 Artifact의 실행/파일·네트워크 관찰은 기존 Policy에 그대로 반영한다. Native
script의 종료0은 ALLOW나 Host 설치 완료를 뜻하지 않는다. Whole native 경로의 실제
Evidence/Policy·CLI/native 인계 qualification은 [B6c 지도](../planning/evidence/package-client-quarantine/p3b6c-bun-whole-runtime-plan-20261025.md)와
[Queue](../planning/02-current-work-queue.md)의 named local/remote 상태를 따른다.

기존 Bun 설치의 첫 인계는 nonempty safe single-link copies의 모든 원본 파일을 독립 공개
source/SRI·전체 layout과 대조한 뒤 새 선택의 검역을 거친다. Shared hardlink·mixed owner·
변조/추가 파일·unknown/corrupt HAA state에는 native fallback이 없다. 기존 fresh/managed
empty의 검증 경로는 유지한다. Manifest/SBOM·actual operation records와 원본/state 보존은
공통 transaction 계약을 사용한다. Global/workspace·다른 registry/local/VCS·임의 native 옵션과
`bun run`/`bunx`/`bun x`는 이 설치·업데이트 검증 범위에 포함하지 않는다.
