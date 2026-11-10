# P3-B6c whole Bun inspection의 native runtime 경로 연결 감사

B6b named local execution 경계 뒤 baseline 9fbc9817a931208c384997d9a40b2bf0f7367d32/tree2f870528ba0473d77a3fc2c7bbba5aa341e0b87c/2473files를 고정했다. M13-004만
IN_PROGRESS다. Read: B6a/b audit·원본, client 설계/확장 계획§5, BunProjectInspector/
BunInstanceExecutor, bun_dynamic fact/coverage, bootstrap composition/cache/records, M3 Policy.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| whole source static-all·per-instance fresh sessions/exact coverage/Evidence/Policy/cache | YES | YES(Node lane) | YES(existing named local/CI412/B5e) | YES(existing scope) | 고정 native Bun runtime consumer로 연결 |
| pinned Bun supply·Artifact 실행/고정 PATH·역할/관찰 | YES(B6a/b source) | YES(explicit backend) | YES(canonical121/actual normal·exit37/기존 Bun·PNPM) | NO(whole inspection) | whole inspector의 typed capability |
| Bun full CLI/native proof/transaction/remote qualification | YES | YES(Node lane) | YES(named local) | NO(whole P3) | native runtime whole local·hosted/영향 matrix |

## 수정 지도·두 번째 검토

기존 검사 엔진을 대체하지 않는다. `internal/inspection/npm/bun_project.go`의 consumer-owned
BunInstanceExecutor method/call만 ExecuteBunRuntimeInstance로 강화한다. 기존 static-all/
원본 binder·각 물리 instance fresh session/정확한 required coverage·partial suspicious facts/
current recorded evidence·Policy·same inspected bytes cache/handoff는 유지한다. 기존 Node-only
ExecuteBunInstance는 이미 검증된 legacy 소비자/회귀를 위해 bytes를 보존한다.

실제 caller는 기존 Backend의 새 typed method로 컴파일 연결되고 fake sequence/gate consumer
두 곳은 같은 새 method name을 구현한다. 새 generic interface/fallback/dynamic selection을
만들지 않는다. Bun runtime이 없거나 잘못됐을 때 기존 Node-only 경로로 몰래 돌아가지 않는다.
기존 M3의 unexpected Artifact process는 MANUAL_REVIEW, FS/honeytoken은 BLOCK이다.
Native script가 종료0이라고 이 사실을 없애거나 ALLOW를 만들지 않는다.

Public transitive/peer whole inspection의 static-all→native supply/실제 session→기록된 Evidence/
Policy·handoff와 managed CLI/native adoption 실제 회귀를 같은 canonical observer/binary source에
연결한다. Canonical Quick/security/docs·inspection/sandbox/bootstrap 관련 전체 Race와
whole index/blob/mode/disk/tree를 남긴다. Runtime-qualified 지원 문구는 exact named local/
remote 범위를 구분해 canonical product leaf/계획 표와 Queue에 한 번만 기록한다.

이 end-to-end 묶음을 준비한 뒤에만 기존 중간 commit들을 한 번 push하고 actual run/head·
최근 elapsed+새 검사 실측·예상 첫 확인을 기록한다. 실제 timer를 등록하고 active turn에서
결과·원본을 확인하며, 대기 중 같은 item의 다음 독립 작업을 계속한다. CUDA 등 최종 영향
matrix는 명시 범위대로 남기고 대표 성공을 전용하지 않는다.

## 이번 실제 검증 선택

B6b의 같은 canonical observer121을 재사용하며 CPP/PNPM 경로는 불변이다. 새 whole
Bun public quarantine와 Bun CLI9 fresh operations/native original3 cases를 직접 실행한다.
변경 없는 PNPM CLI를 이 작은 port 연결마다 다시 실행하지 않고 B6b 실제 PNPM full
project와 기존 CLI/원본 qualification을 범위대로 재사용한다. 최종 묶음 hosted CI는 기존
PNPM 실제 소비자도 그대로 포함한다. Canonical 제품 leaf§10와 계획의 현재 P3 행을
현재 구현·named local/remote 미완료에 맞춘다. CLI 도움말은 실제 지원 문구가 바뀌는 경우만
수정하며 runner/bunx 등의 명령은 미리 등록하지 않는다.

Public whole actual은57.90s/전체 operator60.498s에 transitive2·peer4의 recorded Evidence/
current Policy ALLOW·정확한 coverage와 helperwait0/Host 동일을 확인했다. 이 public fixture는
script-free이며 Bun JS 자체의 실행은 별도 B6b 정상/실패 fixture가 검증한다. 기존 actual
로그의 Node-only 문구는 이전 consumer 당시의 test 출력이므로 현재 producer 문구만
native runtime supply/별도 JS 검증 범위에 맞춰 갱신한다. Actual raw는 편집하지 않는다.

## Hosted native JS 소비자 연결의 추가 gap

B6b actual JS test는 IMPLEMENTED=YES/로컬 QUALIFIED=YES지만 hosted default Bun script에는
아직 WIRED=NO다. 현재 end-to-end checkpoint에서 `ci-bun-runtime.sh`의 기존 sandbox cwd/
고정 공급 뒤에 exact HELOX_BUN_ARTIFACT_EXECUTION 명령 하나를 추가한다. 기존 native/
whole quarantine와 project mode·owning helper/복원·jobs/timeouts는 유지한다. Validator의
세 sandbox 명령 scope/count·명령/env/filter/deadline/실패 전파와 missing/comment/echo/
ignored failure negatives만 갱신한다. 기존 materializer/cleanup body는 재사용하며 새 workflow
job을 만들지 않는다. Script 전체 actual의 기존 공급 검증(B4de/B5e)과 이번 각 동일 named
소비자 actual·고정 실행 순서 검사를 범위대로 연결하고 grouped hosted 실행에서 같은 script를
확인한다. 작은 문구·helper 수정별 remote CI는 반복하지 않는다.

## Named local end-to-end 결과

[로컬 영수증](./p3b6c-bun-whole-runtime-local-20261025.json)은 whole public 검역57.90s/
operator60.498s와 Bun CLI9fresh199.72s/native original3cases135.92s·operator338.154s를
연결한다. 두 scope 모두 helperwait0/Host 동일이며 exact source/instance coverage·현재
recorded Evidence/Policy·원본 source proof/반입·no-op inode/mode/사용자 파일·empty lock
부재를 확인했다. 기존 엔진의 변화는 typed method 이름뿐이고 Policy·cache/transaction/
CPP·legacy Node backend는 baseline과 bytes가 같다.

Hosted JS command의 workflow1.390s/Quick66.090s/security/docs와 기존 제품 네 package
Race54.598s·checker 전체 Race35.595s가 PASS다. 고정 공급/cleanup/project mode는 새 exact
JS 명령 한 줄을 제외하면 bytes가 같다. Script-free public 검역의 로그 문구만 실제 runtime
supply/별도 B6b JS 범위로 보정했고 원본 raw는 유지했다. Named local 전체 연결만 닫았으며
다음 grouped push의 exact-source hosted 결과와 최종 지원/영향 matrix는 여전히 열려 있다.
