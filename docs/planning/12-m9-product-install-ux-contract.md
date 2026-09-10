# M9 Entry Decision — Product Install UX

M9은 M8의 trusted Host·observer·privileged policy 경계를 바꾸지 않는다. 사용자가
내부 staging·Docker·gVisor를 알지 않아도 되는 install UX를 만들되, 검사된 exact
Verified Set 밖의 content를 기존 환경에 반입하지 않는다.

## Transaction invariant

모든 default install은 다음 순서만 허용한다.

```text
target discovery → exclusive mutation guard → plan freeze → inspect/policy
→ private mutate → post-mutation verify → atomic commit / rollback
```

- ambiguity, global environment, concurrent mutation, target identity change,
  post-verify failure 또는 rollback uncertainty는 fail-closed이며 `ALLOW`가 아니다.
- npm은 project root의 `package.json`, supported lockfile, `node_modules`를 하나의
  transaction set으로 취급한다.
- M9의 npm project mutation은 마지막 HAA committed control-state metadata와 현재
  control files가 정확히 일치하는 HAA-managed project만 허용한다. metadata가 없는
  project는 dependency가 없는 신규 project일 때만 첫 commit으로 bootstrap할 수
  있다. metadata 없는 기존 dependency graph의 임의 adoption·재해결은 금지하며,
  full-graph requalification/adoption은 deferred backlog다.
- pip UX는 internal source ID `pypi`와 분리한다. active virtual environment의
  site-packages, dist-info, RECORD, console script를 하나의 transaction set으로
  취급하며 global/user install은 지원하지 않는다. 기존 venv bootstrap 또는 사용자
  baseline은 재검역·변경하지 않는다. 첫 HAA install은 private output의 모든
  destination이 baseline에 없을 때만 허용하고, commit metadata에는 HAA가 소유한
  exact file digest set만 기록한다. 후속 install은 이 metadata와 현재 HAA-owned
  file state가 정확히 일치할 때만 허용하며 baseline 또는 다른 package가 소유한
  path와의 충돌은 fail-closed다.
- `--target`은 explicit advanced path로 보존한다. default target은 supported
  project/active venv discovery로만 선택하며 arbitrary option pass-through는 없다.
- root `helox --help`와 source-level `helox <source> --help`는 전체 명령 트리를
  정적으로 보여주며 runtime, Docker, observer, resolver를 시작하거나 파일·환경
  상태를 변경하지 않는다. 실제 실행 시에만 source별 trusted composition을
  주입한다.

## M9-004 CLI tree and target UX acceptance

- `npm`, `pip`, `pypi`, `github` source group과 inspect/install leaf가 root command
  construction 시점에 존재하여 root/source help가 runtime composition 없이
  완전한 tree를 보여준다.
- `helox npm install <package>`는 현재 working directory의 supported HAA-managed
  npm project를 default target으로 선택하고, `--target`은 고급 absolute target
  override로만 사용한다.
- `helox pip install <project>`는 `VIRTUAL_ENV`의 active Python 3.14 venv를
  default target으로 선택하고, `--target`은 고급 absolute venv-root override로만
  사용한다. active venv가 없으면 fail-closed한다.
- `helox github install <owner>/<repo>@<tag>#<asset>`는 현재 working directory
  아래 HAA 전용 asset-scoped directory를 default target으로 선택한다. 이
  directory 이름은 normalized selector와 collision-resistant selector digest로
  결정되며, 별도의 parent 생성이나 ambient global install을 수행하지 않는다.
- GitHub standalone Promotion은 target이 이미 존재하면 절대 덮어쓰지 않고
  fail-closed한다. `--target`은 고급 absolute directory override일 뿐이며
  overwrite/force 옵션은 제공하지 않는다.

## M9-006 transaction qualification acceptance

- unmanaged npm dependency graph, control-file drift, concurrent lock,
  interrupted commit과 partial publish는 모두 거부되며 기존 project state를
  보존한다.
- active venv의 baseline collision, HAA-owned file drift, symlink/special output,
  concurrent 또는 interrupted transaction은 모두 거부되며 기존 state와
  metadata를 보존한다.
- GitHub standalone의 existing target과 target race는 overwrite 없이
  fail-closed하고, default target은 기존 Host path를 재사용하지 않는다.
- 위 hostile regression suite와 전체 `go test -race ./...`, canonical `quick`,
  `docs`, architecture 및 CI workflow 검사가 통과해야 M9를 COMPLETE로 기록한다.

## Work breakdown

| Order | ID | Scope |
| --- | --- | --- |
| 1 | M9-001 | transaction·target discovery·rollback canonical contract |
| 2 | M9-002 | npm project transaction plan/freeze/private commit |
| 3 | M9-003 | pip active-venv transaction and CLI terminology |
| 4 | M9-004 | complete root CLI tree, defaults and advanced target UX |
| 5 | M9-005 | GitHub standalone default destination/overwrite policy |
| 6 | M9-006 | transaction hostile-boundary regression and qualification |

M10 distribution/bootstrap과 M11 detection depth는 M9 completion 이후의 별도 범위다.
