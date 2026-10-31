# P2c5-B2 — 정확한 PNPM instance의 npm lifecycle primitive

2026-10-24 baseline db464fdca8e4de9070d546d216ddd1861716edb6.
[B1 감사·후속 수명 경계](./p2c5b-pnpm-production-context-plan-20261024.md)와
[A3](./p2c5-pnpm-offline-context-plan-20261024.md#8-a3-peer-fixture--착수-감사수정-지도두-번째-검토)를
읽는다. 같은 M13-004/P2이고 새 milestone 또는 완료 관문이 아니다.

## 1. Source 작성 전 baseline audit

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 Backend.Execute와 npm-lifecycle exact install argv/원장·격리 | YES | YES | YES | YES | 그대로 유지; rebuild로 교체 안 함 |
| PNPM frozen controls/native offline layout·peer | YES | YES(test) | NO(full) | NO(full) | A3 실제 PASS; production static-all/prepare/이송은 후속 |
| B1 PNPMInspectionContext/Input/producer | YES | YES | NO(full) | NO(full) | 원격403 PENDING; 로컬 전체 입력·다중 instance/Race PASS |
| 정확한 native PNPM instance의 target-only lifecycle | NO | NO | NO | NO | 본 B2의 고정 command·관찰기 admission·primitive actual |
| per-instance dynamic aggregate/최종 검역·Host/CLI | NO | NO | NO | NO | B2 뒤 별도 상세 지도·실제 확인 필요 |

## 2. 고정 npm source 의미

Runtime lock의 npm11.19.0/tag commitbfacd33ccbcd908480610703b60455d2da5b57a9를 확인했다.
`lib/commands/rebuild.js`는 directory spec을 node.path와 정확히 비교한다. Arborist rebuild는
regular node의 preinstall/install/postinstall을 실행하고 link node에서는 prepare도 실행한다.
선택 생략은 전체 tree 실행이므로 금지한다. `bin-links=false`로 다른 package bin 재작성을
막되 native 준비가 만든 기존 dependency bin은 유지한다. Script approval 설정으로 skip될
수 있으므로 isolated quarantine 안에서만 고정 allow-all 옵션을 명시한다. 이는 Host 설치
동작 승인이나 M4_HOST_LIFECYCLE_UNSUPPORTED 해제가 아니다.

Source는 raw owner의 p2c5b-npm-rebuild-upstream-0..2.js에 pinned raw/contents 응답 일치를
확인해 보존했다. Exit0/사용자 script 출력만으로 coverage를 주장하지 않으며 실제 고정
fixture의 세 event 순서와 root/다른 dependency 실행0을 따로 검증한다.

## 3. B2a 수정 지도 — 기존 profile에서 별도 정확한 호출만 추가

새 profile을 만들고 수십 공용 predicate를 복제하지 않는다. 기존 npm-lifecycle profile에
**전용 topology 진입점 + 고정 command의 lineage**를 함께 요구한다. 기존 install predicate,
read 권한·network 차단·artifact demotion·budget은 유지한다.

1. sandbox/pnpm_target_lifecycle.go에 고정 guest prefix/command·argument producer를 둔다.
   Shell은 고정 문자열, 대상은 positional argv 하나이며 사용자 script text를 삽입하지 않는다.
   npm --prefix <fixed-project> rebuild에 exact real instance directory 하나와
   ignore-scripts=false/bin-links=false/foreground/offline/notifier-off/workspaces=false/
   audit-off/fund-off 및 quarantine-only allow-all을 고정한다. 빈/상대/escape/extra selector는
   consumer preflight에서 거부한다. Node/npm 버전은 runtime lock을 계속 사용한다.
2. shared_observer.go에 StartPNPMTargetLifecycleProfile이라는 좁은 진입점과 topology helper를
   추가한다. 기존 NPM mounts에 fixed PNPM project의 Host-attested named tmpfs/guest9p만
   추가하며 기존 StartProfile의 mount 기대값은 바꾸지 않는다. Host noexec와 guest9p flags를
   혼동하지 않는다. 실제 Docker volume owner/sole attachment 검증은 기존 owner를 재사용한다.
3. observer.cc에는 고정 boundary shape+nonce/CONTROL direct-root+sealed exact topology가
   함께 맞을 때만 private target state를 기록한다. Target 경로 문법은 fixed virtual-store
   namespace로 한정하되 이름 자체를 source/무해성/실행 권한으로 취급하지 않는다.
   기존 CloneInfo의 exact creator를 통해 그 target을 CONTROL child에만 계승한다.
4. 새 rebuild npm CLI→Node 전환은 pinned 두 image/execfn, 고정 flags·단일 target,
   실제 creator/start identity·저장된 target의 완전 일치 및 별도 pending bit를 요구한다.
   한 번 소비하고 demotion 뒤 권한을 되살리지 않는다. 기존 exact install/version/lock
   transition은 그대로이며 새 rebuild를 기존 install helper의 broad alternative로 넣지 않는다.
5. observer_latch_test.cc: 올바른 별도 shape부터 검증하고 wrong mount/default topology,
   unsealed/shadow, wrong profile/role/provenance/creator/PID generation, target drift,
   missing/extra flag·selector, wrong image·interpreter, replay/demotion을 거부한다. 기존110을
   보존한다. Go tests는 fixed argv와 topology/기존 default 불변을 검증한다.

Boundary target은 native output 또는 artifact가 써 준 이름으로 arm하지 않는다. 신뢰된
composition이 B1과 검증된 layout에서 선택한 경로를 전달해야 한다. B2의 path shape check가
그 후속 identity/bytes/edge 검증을 대신하지 않는다.

## 4. B2b 실제 primitive와 후속 production 이송의 구분

먼저 A의 native-generated project layout을 bounded Docker tar stream으로 읽고 별도 fresh
owned volume의 NPM container로 소개하는 **test-only primitive**를 작성한다. Host에
압축 해제하거나 artifact를 실행하지 않는다. Native preparer/container/volume의 마지막
소비·정리를 끝낸 뒤 새 관찰 session을 연다. 실행 container는 network none/endpoint0,
original NPM image/boundary/caps/budget이며 original observer 수신·정리를 요구한다.

세 event와 비대상 실행0을 입증할 controlled synthetic probe는 실제 공개 tarball을 승인한
검역 결과로 표시하지 않는다. 변경한 probe bytes·원래 layout bytes·scope를 별도 기록하고
Policy/Host staging을 호출하지 않는다. Missing target·script failure도 원래 상태와 complete
trace/잔여0·Host불변을 보존한다. 첫 positive가 admission에서 실패하면 후속 negatives를
같은 실패로 반복해 통과했다고 쓰지 않는다.

Production 이송은 별도 B3 상세 지도에서 archive bytes/전체 instance/edge/경로·중복·symlink/
bin/owner/mode/size 검증과 원본 controls·runtime 결합을 구현해야 한다. B2 시험 tar 통과를
범용 layout verifier나 product closure API로 전용하지 않는다. 미검증 cache/store format을
직접 생성하지 않는다. 각 expected peer instance는 fresh session이며 실행 뒤 layout을
다른 target에 재사용하지 않는다.

## 5. Source 수정 전 두 번째 검토와 검증 순서

기존 exact install 함수를 확장하는 안은 legacy admission에 새 argv 권한을 섞으므로 제외했다.
새 profile 전체 복제 대신 전용 topology와 exact boundary lineage/target pending만 추가한다.
단순 prefix 일치/comm 이름/argv만으로 CONTROL을 주지 않는다. Stored target은 입력과
실제 clone lineage의 상관관계이며 source 인증·Policy·완료 token이 아니다.

B2a는 canonical format/Quick/security/docs·Go 관련 tests 및 pinned C++ 전체110+새 matrix를
먼저 검증한다. 그 뒤 B2b 실제 positive/negative와 기존 단일 npm·PNPM/UV 핵심 실제 소비자를
영향에 맞춰 묶어 확인한다. 공용 관찰기 변경의 remote qualification은 다음 의미 있는
checkpoint에서 필요하며 현재403 source와 섞지 않는다. 새 helper마다 CI 재push하지 않는다.
완료한 전용 test bulk는 all-UID references/query errors0 확인 후 삭제한다.

## 6. B2a 첫 로컬 결과와 fixture 교정

첫 canonical format/Quick129.833s/security6.208s/docs.234s는 PASS다. C++29/30은 기존110과
새 shape/topology1 PASS, 새 production lineage1 FAIL이었다. Bounded test 진단은 positive
전체 경로가 지나간 뒤 첫 target-drift negative의 bare ExpectRecord matcher에서 멈춤을
확인했다. 실제 unexpected event에는 필수 attribution이 있어 bare matcher와 같을 수 없다.
제품 sourcec5e1c80c는 유지하고 과거 정상 production-path test의
ExpectUnexpectedProcessRecord를 재사용했다. Wrong generation의 실제 reject reason도
정확히 검사한 C++31 전체112 PASS36.249s가 fixture-only 교정을 입증한다. 권한을 완화하거나
negative를 삭제하지 않았다. 실제 lifecycle/production prepare는 아직 미검증이다.

B2b는 transitive native layout의 원본 검증/정리 뒤 별도 새 NPM session을 사용한다. Target의
세 script는 shell printf로 정해진 workspace event 파일에 순서대로 기록하고 prepare는
실패하게 한다. Root/다른 dependency의 scripts도 실패하도록 한 **의도적으로 변경한 synthetic
probe**다. 이 bytes를 공개 acquired archive의 검역 또는 ALLOW로 기록하지 않는다. 실제
세 event·root/dependency/target-prepare0·complete/금지0·network none/endpoint0과 원래
오류/Host불변/정리를 동시에 확인한다. A의 성공 fixture는 capture=nil이면 그대로다.

B2b first11.758s는 native prerequisite/complete/금지0·Host전체동일/helperwait0 뒤 새 NPM
container의 network=none 검사에서 멈췄다. NPM command는 NOT_EXECUTED이며 첫 검사에서는
어느 endpoint 필드가 거부를 일으켰는지 보존하지 않았다. Docker29.8.1의 기존 none network는
정확한 local/null driver다. [공식 none 계약](https://docs.docker.com/engine/network/drivers/none/)은
loopback만 만든다. 다음 fixture는 HostConfig.NetworkMode=none과 trusted inspect의 정확한
none NetworkID/local/null driver·실제 IPv4/IPv6 부재를 함께 요구하고 관리용 EndpointID의
존재 여부도 bounded boolean으로 기록한다. Query 실패나 bridge/unknown network를
부재로 바꾸지 않는다. Product의 기존 network=none 격리와 C++ source는 유지한다.

Null-network actual13.538s는 logical EndpointID는 있지만 정확한 null NetworkID와 주소 부재를
확인했고, 실제 target 세 event/정확한 순서·root/dependency/prepare0까지 PASS였다. Trace는
complete이며 filesystem/honeytoken/network0이지만 ARTIFACT shell의 unexpected exec3으로
fixture 전체는 FAIL이다. 이 셋은 실제 기록이고 숨기거나 CONTROL로 올리지 않는다.
기존 IsExpectedProcess의 ARTIFACT_ROLE과 Domain12의 M3_UNEXPECTED_PROCESS 수동 검토
계약을 대조했다. 첫 B2 fixture가 script-free A의 금지0 기대값을 그대로 전용한 누락이다.

교정 검증은 기존 단일 Backend.Execute의 three-script actual을 **먼저** 실행해 같은
attribution3을 확인하고, 새 target actual에도 정확히3을 요구한다. 낮아져0이 돼도 실패한다.
관찰기의 source/정책은 바꾸지 않는다. Fixture 성공은 command 선택과 기존 untrusted
분류 보존이며 M3 ALLOW가 아니다. production static-all/aggregate는 이 관찰을 그대로
Finding/Evidence/Policy에 전달해야 한다. 첫 source의 금지0 계획은 이 script 실행 범위에
대해 superseded이며 filesystem/honeytoken/network0 조건은 유지한다.

Attribution actual16.540s는 기존 단일 npm도 ARTIFACT shell3임을 먼저 확인한 뒤 새 target의
세 event/비대상0·동일 attribution3·complete와 다른 금지0·Host불변/helperwait0로 PASS다.
기존 정책을 바꾸지 않고 scope가 다른 fixture의 기대값만 교정했다.

다음 같은 controlled native layout을 immutable bytes로 한 번만 보관하고 서로 다른 fresh
NPM volume/session의 positive와 install `exit 37` negative를 순차 검증한다. 실패한 실행
layout을 재사용하지 않는다. Negative는 preinstall 한 event까지만 남고 ARTIFACT exec2,
원래 command exit37·timeout아님·complete/다른 금지0을 요구한다. Npm11.19.0의 pinned
error-message.js는 numeric error code를 exit status로 보존한다. Positive가 실패하면 이
negative를 같은 최초 실패에서 반복하지 않는다. Policy/Host staging은 계속 미실행이다.

## 7. B2 로컬 실제 결과와 남은 연결

Final synthetic peer-target actual37.742s는 기존 npm attribution3 확인 뒤 transitive/peer의
positive·script-failure 네 fresh session을 통과했다. Positive 세 event/정확한 순서,
failure는 preinstall 하나·원래 exit37·timeout아님, 비대상/root/target prepare0을 확인했다.
각 trace는 complete이며 ARTIFACT shell3/2를 보존하고 filesystem/honeytoken/network0이다.
Host전체동일·Host native installation 부재/helperwait0/CONFIRMED_STOPPED/original0다.

별도 regression46.978s는 npm 기존 환경격리, PNPM startup/empty/Node query/ARTIFACT4와
transitive/peer offline 배치, UV startup/query2 등8 top-level actual이 PASS다. Source c5e1c80c
및 binary7c46fa0c는 동일하고 C++33의 전체112 PASS39.589s는 모든 필수 argv의 누락/변조,
target/creator/role/one-shot 경계를 추가 확인했다. 기존110은 모두 유지했다.
Canonical Quick52.586s/security4.851s/docs.231s도 PASS다. 완료 namespace bulk는 all-UID
refs/query errors0 뒤 삭제했으며 작은 raw와 다음 소비자에 필요한 source/cache만 보존한다.

B2의 exact command/topology/관찰기 연결과 controlled 실제 primitive는 LOCAL VERIFIED다.
Production static-all, 모든 acquired bytes/instance/edge와 전체 layout 검증·prepare owner,
per-instance aggregate/기존 Policy 전달·Host staging/transaction/CLI와 전체 P2 qualification은
계속 미완료다. Native directory selection의 exit0만으로 대상 존재/coverage를 인정하지 않으며
B3가 expected target의 inventory와 byte binding을 실행 전에 반드시 확인해야 한다.

최종 npm/sandbox 전체 Race21.119s도 PASS다. 이 로컬 후보는 source db464fd의 CI403과
구분한다. CI403은55분 타이머 뒤10SUCCESS/whole tree 일치를 확인했으나 B2 source는
포함하지 않는다. [B2 로컬 영수증](./p2c5b2-pnpm-target-local-20261024.json)이 실제 입력과
검증 범위·작은 원본을 연결한다. 다음은 B3의 archive inventory/전체 layout·static-all/
production prepare owner 수정 지도를 먼저 감사·재검토하는 순서다.

후속 source1f410fd의 [CI404](./p2c5b3-pnpm-ci404-qualification-20261024.json)는 실제55분
타이머 뒤 Required10SUCCESS/whole tree 일치·두 job의 새112 latch/PNPM4/기존 Linux57·
CPU281.30s/native macOS/helper5wait0를 확인했다. B2 공용 observer의 ordinary 회귀를 닫는
범위이며 target synthetic actual은 앞 local receipt를 유지한다. 전체 P2 acceptance와
production full layout/prepare/aggregate는 계속 미완료다.
