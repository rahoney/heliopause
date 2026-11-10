# P3-B5f Bun transaction의 실제 중단·용량 실패 acceptance 감사

M13-004 하나만 IN_PROGRESS다. B5e project 공급 script 실제 검증과 독립적인 동일
transaction acceptance gap을 대조했다. Source 기준은 637dadadc950492b2c70765ec98fc64151aef34a/treee6bf1112d76cac21ad027f851fd64bac9b86a329/2461files다.
Read: 확장 계획§5, client 설계§4, B5b2/B5d/B5e 감사·영수증, P2-B5e 직접 실패
감사·원본, bun_transaction/test, pnpm_failure_process_real_test·enospc_namespace_real_test,
공용 npm transaction/profile·state publication, 관련 artifact/sandbox/evidence/ci lessons.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| typed Bun cache/guard/rollback·독립 승인/기존 설치 인수 | YES | YES | YES(named local B5a..d) | NO(whole P3) | 아래 실제 실패·명시 Bun runtime·원격 matrix |
| Bun 단계별 forward failure·rollback conflict/cancel-before-commit | YES | YES | YES(default tests·전체 Race) | YES(existing scope) | 실제 OS signal/process/filesystem 실패의 증거와 구분 |
| Bun 실제 mid-publication SIGINT/SIGKILL·1MiB tmpfs ENOSPC | NO(test consumers) | NO | NO | NO | 기존 PNPM qualification을 Bun typed owner/lock에 연결 |
| PNPM 동일 직접 실패 consumer | YES | YES | YES(P2-B5e) | YES(named local) | 변경 없는 원래 회귀 보존 |

## 수정 지도·두 번째 검토

Production engine을 다시 구현하거나 완화하지 않는다. 두 linux&&realcorpus qualification
파일을 기존 PNPM 직접 실패 consumer에서 기계 재사용한다. 기존 generic parent-ready
struct는 재사용하고 새 타입/프로토콜을 만들지 않는다. Bun fixture/typed factory·cache
callback은 실제 Bun 타입으로 연결하며 private checkpoint/state root만 owner.engine에서
사용한다. lock은 bun.lock, 명시 opt-in/test filter는 Bun 고정값이며 기존 PNPM 파일은
변경하지 않는다. Source/SRI는 기존 구조 fixture, static/local Evidence/cache/transaction은
실제 생산 owner다. 이 검사는 registry 인증/native selection/동적 Policy 검증을 대체하지 않는다.

1. ready pipe로 after-publish:bun.lock에 동기화한 child에 실제 SIGINT/SIGKILL을 보낸다.
   SIGINT는 context.Canceled/cause·원래 manifest bytes/inode/mode·독립 state/records 미게시·
   journal/guard 정리를 요구한다. SIGKILL은 원래 backup/exclusive lock을 보존하고 새로운
   typed Bun Begin이 cache 접근 전에 거부하는지 확인한다. 자동 recovery를 약속하지 않는다.
2. mount/net private namespace에서 task-owned 1MiB tmpfs를 state root에만 mount한다.
   before-state에서 실제 write ENOSPC를 확인하고 실제 approval write 실패에 같은 cause가
   남는지 확인한다. 원본 복원·partial/temp 미존재·FD0·filler 제거/unmount를 요구한다.
   Host 공유 filesystem·프로젝트/cache에는 공간 압박을 만들지 않는다.
3. 기존 PNPM 직접 실패도 같은 후보에서 함께 실행하고 no-SKIP/named PASS를 요구한다.
   Canonical format/Quick/security/docs·tagged vet/compile·관련 promotion 전체 Race를
   실행한다. 동일 source 전체 index/blob/mode/disk/tree와 production unchanged를 기록한다.

새 직접 실패가 나타날 때만 원본과 최초 cause를 보존하고 exact owner를 수정한다.
작은 acceptance 테스트마다 원격 CI를 반복하지 않는다. 이 범위를 닫아도 명시 Bun runtime/
전체 P3·영향 profile qualification은 열려 있고 다음 실제 MISSING으로 계속한다.

## 직접 acceptance 결과

[로컬 영수증](./p3b5f-bun-failure-local-20261025.json)은 두 client의 실제 SIGINT/SIGKILL
Race15.560s, bounded tmpfs ENOSPC0.474s와 promotion 전체 Race153.066s를 연결한다.
Bun typed factory/guard·원본 bytes/inode/mode·원래 cause·미게시 state/records·crash backup/
재진입 거부·FD0가 PASS다. 기존 PNPM 직접 소비자도 같은 source에서 모두 PASS다. 각
consumer는 client/owner/lock 치환을 되돌리면 원래 harness와 bytes가 같고 원래 PNPM
파일·production owner9개는 baseline637dada와 불변이다. Quick80.876s/security10.097s/
docs0.339s와 tagged compile12.652s/vet9.392s도 PASS다. Public registry/native/Policy의
실행 근거와 이 구조 source/SRI 실패 fixture를 구분한다. Named 세 gap만 닫았고 명시 Bun
Artifact runtime/hosted whole project/전체 지원과 영향 profile qualification은 계속 열려 있다.
