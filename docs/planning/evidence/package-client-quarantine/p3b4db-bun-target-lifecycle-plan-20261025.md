# P3-B4d-b 정확한 Bun hoisted instance lifecycle primitive

기준da02f4e28068834a8658274e22e4633e00d7155e/treebf1c8bee89a13dd55340d3f0a3b8e4f63dc7c15f,
2401개 전체 후보다. M13-004 하나만 IN_PROGRESS다. Read: B4d 지도와 B4d-a 영수증,
기존 PNPM target lifecycle 지도·Go command/actual·observer creator/one-shot latches,
Architecture/Domain inspection 계약과 라우팅된 lessons를 따른다.

## Baseline audit

| 책임/근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun source/layout/prepare와 최소 실행 payload, 4420b48/da02f4e | YES | YES(named actual) | YES(named local) | YES(data subset) | 정확한 instance 실행 |
| PNPM target-only npm rebuild/Node lifecycle·sealed topology/creator/one-shot, pnpm_target_lifecycle.go·observer.cc | YES | YES(PNPM) | YES(existing scope) | YES(existing scope) | Bun hoisted locator·별도 topology 진입점 |
| Bun exact target lifecycle | NO | NO | NO | NO | 본 단계의 고정 명령/닫힌 경로 문법·관찰기·실제 정상/실패 |
| Bun production instance backend/complete inspection/Policy·Bun runtime supply | NO | NO | NO | NO | primitive 뒤 기존 owner와 typed layout 연결; 명시 Bun 실행 별도 검증 |

## 명령 의미와 exact edit map

이는 고정 Node/npm의 quarantine lifecycle executor를 Bun hoisted tree의 한 directory에
적용하는 primitive다. 기존 pinned npm rebuild의 exact directory 선택과 regular registry
package의 preinstall/install/postinstall 의미를 재사용한다. 고정 Bun source의 POSIX shell
chain과 대조하되, Bun trustedDependencies의 이름별 skip을 검사 완료 authority로 쓰지 않는다.
Bun runtime 자체가 필요한 script나 native Bun installer 전체 의미의 qualification은 후속이다.
실제 검증 전 Node/Bun runtime 등가성이나 whole lifecycle 완료를 주장하지 않는다.

1. `internal/sandbox/bun_target_lifecycle.go`: fixed Bun project prefix 아래 package 단위의
   hoisted/nested/scoped 경로만 허용한다. 고정 shell source에 경로를 삽입하지 않고 단일
   positional target을 넘긴다. npm/Node·offline/foreground/bin-links-off/allow-all 등 기존
   고정 실행 의미는 보존한다. Target은 control locator이고 identity/안전성 증명이 아니다.
2. `shared_observer.go`: 기존 npm profile에 Bun fixed project mount를 별도 attested 진입점으로
   추가한다. Default npm·기존 PNPM topology는 바꾸지 않는다.
3. `observer.cc`: 기존 exact PNPM shape는 유지하고 Bun command/locator/topology branch만
   추가한다. CLI→Node 전환의 fixed project prefix는 저장된 exact target의 닫힌 kind와 일치해야
   한다. Profile/CONTROL/direct consumed root/nonce/creator/start identity/one-shot guard와
   demotion·원래 attribution을 그대로 유지한다. 새 filesystem/network grant는 실제 최초 거부 전
   추가하지 않는다.
4. `bun_target_lifecycle_test.go`와 `observer_latch_test.cc`: shallow/nested/scoped 정상과 escape,
   다른 root/빈 target/누락·추가 argv/잘못된 command·selector·profile/topology/role/creator,
   target drift/replay를 검증한다. 기존117 latch와 PNPM 실제 소비자를 유지한다.
5. `bun_target_lifecycle_real_test.go`: 통제된 synthetic hoisted/nested graph로 target의 세 phase
   순서와 root/비대상0, install exit37의 원래 cause·fresh session·complete trace/정리를 확인한다.
   Synthetic fixture는 public source/authentication/Policy/Host 설치 증거가 아니다. Artifact shell
   attribution을 숨기거나 CONTROL로 올리지 않는다. Production bound layout backend는 후속이다.

두 번째 검토: PNPM 구현을 교체하지 않고 기존 원장·creator/one-shot 소비와 data helpers를
재사용한다. Runtime별 script 실행 지원을 같은 것으로 단정하지 않는다. 이번 primitive의
actual 성공과 전체 Bun 검역/Policy 완료를 분리한다. 작은 helper별 push/CI는 없으며 backend
연결 checkpoint에서 원격 macOS/Required gate와 실제 타이머 후속 확인을 수행한다.

Actual driver 추가 검토: `pnpm_target_lifecycle_real_test.go`의 기존 scoped body를 test-only
`withActualNPMTargetLifecycle`로 재사용한다. 기존 PNPM wrapper는 같은 fixed kind를 넘기고
Bun은 두 번째 현재 소비자다. Volume/root·typed observer/command 선택 외에 null-network proof,
원래 exit37/이벤트 census·Artifact attribution/trace/cleanup body는 유지한다. 새 범용 production
실행 API를 만들거나 PNPM 원본 명령을 교체하지 않는다.

최초 actual v1의 Bun shallow/nested-scoped 정상·exit37 네 case와 기존 PNPM target tests는
PASS였지만 operator65.289s의 전체 Host proof는 FAIL이다. 같은 작업자가 before/after 사이
Docker canonical compiler를 시작해 compiler exact container/docker0 carrier/firewall rule이
달라졌다. Runtime/source 실패로 해석하거나 snapshot에서 compiler를 제외하지 않는다.
원본 first-host-overlap-assessment는 pinned builder image/command와 exact ID를 대조한다.
원본 builder250.402s/119latches 종료 후 추가 container 부재·bridge/firewall 복원을 성공한
query로 확인했다. 같은 Go binary와 byte-identical canonical observer의 v2 actual은 다른
Docker 작업 없이 직렬 실행해 전체 Host proof를 다시 닫는다. 최초 raw는 그대로 보존한다.

최종 [로컬 영수증](./p3b4db-bun-target-lifecycle-local-20261025.json)의 직렬 v2 actual57.515s는
동일 Go binary/최종 canonical observer로 Bun4·기존 PNPM target 정상/실패·원래 attribution을
통과하고 helper wait0/Host 전체 동일을 확인했다. Primitive의 IMPLEMENTED/WIRED/QUALIFIED
(named local)는 YES, 전체 B4d/Bun acceptance는 NO다. 다음 production bound-layout backend와
whole 검사/Policy·명시 Bun runtime은 별도 검증한다.
