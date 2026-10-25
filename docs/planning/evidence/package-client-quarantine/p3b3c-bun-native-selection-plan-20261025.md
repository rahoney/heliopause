# P3-B3c typed Bun native selection·source 연결 지도

기준 d1d36258a5c3edc20752c10d61e21af0ca937a54/treec3e1a2fe6d679e67601af0fc8bd506747682b808,
2361개 전체 Root/private 후보 일치다. M13-004 하나만 IN_PROGRESS다. B3a raw producer와 B3b
parser/source/snapshot은 named local QUALIFIED이며 새 push/CI는 없다. 전체 Bun 및 P2의 다음
macOS/Required gate는 열린다. 이 단계는 그 다음 실제 연결 gap만 수행한다.

Read: Queue/PROJECT-DECISIONS/docs router, package-client 확장 계획§3–6, B3/B3b 계획·결과,
Architecture adapters·inspection/sandbox/policy와 Domain dependency verified-set/contracts,
lessons artifact L19·sandbox L03/L08/L18·evidence L11/L16·ci L06. 기존 PNPM resolver/input과
nativeProjectScope/controls·Bun actual fixture 및 exact observer shape를 함께 대조했다.

## Baseline audit

| 기존 책임/근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun fixed raw lock command/scope/control census, b658670 | YES | YES(private actual consumer) | YES(named local) | NO(public selection) | typed input와 reusable public producer 연결 |
| Bun parser/source/snapshot, d1d3625 | YES | YES(frozen actual bytes→HTTPS→snapshot) | YES(named local subset) | NO(whole Bun) | native output와 한 operation으로 결합 |
| PNPM typed input/nondependency metadata comparison/producer | YES | YES(PNPM) | YES(existing) | YES(existing scope) | Bun fixed intent/control 이름과 presence 의미 |
| Bun project observer exact13-argv install | YES | YES(raw producer) | YES117/actual4 | YES(named raw shape) | frozen install·add·update의 닫힌 argv와 negative |
| Bun layout/lifecycle/검역/cache/transaction/CLI | NO | NO | NO | NO | B4..후속 acceptance; 이번 단계 성공으로 전용하지 않음 |

## 직접 고정한 native CLI 의미

Bun1.4.4의 install help는 이미 exact candidate/native SHA를 network-none/read-only runc에서
조회했다. 같은 방식의 add/update help도 exit0와 exact owned container 종료/삭제를 확인했다.
원본 p3b2c2-bun-install-help와 p3b3c-bun-add/update-help가 argv/stdout SHA/신원을 소유한다.
`--lockfile-only`, `--ignore-scripts`, `--no-env-file`, `--backend=copyfile`, `--linker=hoisted`가
공통이고 frozen은 `--frozen-lockfile`이다. Update의 `--latest`는 기본 범위를 바꾸므로 추가하지
않는다. Help는 명령 의미 조사이며 gVisor/실제 selection/Policy 지원 검증이 아니다.

## Exact edit map / 두 번째 검토

1. `internal/artifact/npm/bun_selection_input.go`와 tests: install/add/update만 typed enum으로
   받으며 immutable package.json/bun.lock controls·client version을 재검사한다. Unknown/extra/
   zero present file·runtime/config/local selector는 도구/DNS/volume 전에 거부한다. 기존 Bun
   manifest/lock parser를 재사용하고 selected output의 presence를 보존한다. 기존 private
   samePNPMManifestMetadata의 공통 npm nondependency 비교 body는 그대로 재사용한다.
   Add는 대상 한 direct constraint만, update는 지정 대상 또는 전체 direct constraints만
   변경할 수 있고 install은 direct constraints를 바꾸지 못한다. Exact add는 선택된 version도
   대조한다. Frozen install은 기존 lock bytes/presence도 그대로 요구한다.
2. `internal/sandbox/bun_resolver.go`와 tests: 기존 newBunNativeProjectScope를 보유하는 좁은
   producer를 연결한다. Reconstructed input와 compiled Bun pin을 먼저 확인하고 동일 scope의
   copy→정확한 native 명령→complete controls census→ParseOutput을 사용한다. Scope 최종
   observation/cleanup 오류가 있으면 captured output을 버린다. Runtime identity는 generated
   Bun version/tool SHA/Node image/runtime-lock SHA에서만 만든다. Source BindBunLock과 기존
   Domain snapshot을 연결하며 새 Core/Policy·daemon·generic plugin API를 만들지 않는다.
3. `tools/gvisor-observer/observer.cc`와 latches: 기존13-argv fresh install을 보존하고 추가한
   frozen/add/update의 exact argv/count/선택적 마지막 -- selector만 허용한다. 모든 binary/
   execfn/argv0·profile/topology/direct consumed CONTROL/nonce guard는 그대로다. Selector는
   옵션/URI/제어문자/alias가 될 수 없고 fixed backend/linker/cache/registry를 바꾸지 못한다.
   Unknown flags·extra/missing/reordered argv·query/profile/role/creator 거부를 확장한다.
   새로운 filesystem/network grant는 실제 최초 거부 근거 없이 추가하지 않는다.
4. 실제 tests: 기존 raw producer와 같은 protected installed supply·canonical helper/Host 전후
   신원 확인에서 fresh empty/single/transitive/peer, add/frozen install/selected/full update를
   native output→independent metadata→exact controls/runtime snapshot으로 연결한다. 실제
   version transition과 no-op을 구분하고 synthetic old native-readable 입력이면 그렇게 기록한다.
   기존 PNPM 실제 소비자와 all-latch/전체 관련 Race를 보존한다. Typed rejection/cancellation/
   original cause/incomplete observation/cleanup failure에서는 partial selection이 없어야 한다.

두 번째 검토: 이미 완성된 supply/parser/PNPM engine은 재작성하지 않는다. 새 Bun facade와
현재 acceptance에 필요한 argv만 연결한다. B3b의 제한된 tag/prerelease/optional/platform/peer
범위를 몰래 확대하거나 native script flag를 안전성으로 쓰지 않는다. 지원 범위 밖 입력은
explicit error이며 argv/client/path를 ambient 환경에서 선택하지 않는다. 이 단계 뒤 layout/
실제 archive 검역·transaction·CLI는 남는다. 작은 helper별 CI 없이 backend/end-to-end 단위로
push를 묶고, 원격 실행 때만 실제 timer를 active turn에서 소비한다.

## 최종 named local 결과

새 filesystem/network grant 없이 원본 builder250.792s/117latches가 PASS다. 최종 native
selection9/source/snapshot57.54s와 실패 경계5/29.67s가 PASS이며 helperwait0/Host 전체 불변이다.
실제 selected_versions는 synthetic old frozen6.0.0→selected/full update7.0.0을 확인한다.
최종 기존 PNPM boundary5/public8/quarantine4도 같은 observer/client에서139.641s/operator,
helperwait0/Host 불변이다. 준비 receipt의 오래된 mode 문구는 compile-interpretation으로
분리했고 실제 final client SHA와 test selector를 고정했다. 최종 Quick111.564s·두 package
전체 Race59.056s, tagged vet2.825s/vulnerability16.694s가 PASS다. 상세 원본·제한·최종
staged checks는 local receipt가 연결한다. 다음은 exact layout/lifecycle/archive 검역이며
실제 CLI/Host publication이나 whole Bun 완료를 주장하지 않는다.
