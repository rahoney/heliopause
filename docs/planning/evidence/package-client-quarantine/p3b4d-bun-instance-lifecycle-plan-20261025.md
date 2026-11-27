# P3-B4d Bun instance lifecycle 연결 지도

기준4420b48d6d8cb500341b982250eee5000dfa7f92/treee79ce9bd3c499c8c131fbf3d70198a8ca13f5f54,
2396개 전체 후보다. M13-004 하나만 IN_PROGRESS다. Read: Queue/PROJECT-DECISIONS/docs router,
B4/B4b/B4c 지도·영수증, Architecture inspection/sandbox/policy와 Domain inspection 계약,
기존 PNPM payload/instance backend/target lifecycle/whole inspector 및 라우팅된 lessons.

## Baseline audit

| 책임/기존 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun complete source/layout/native offline 준비, d72287e/8ffb0c9/1ea12d9/4420b48 | YES | YES(named actual) | YES(named local) | YES(preparation subset) | 실행 세션용 최소 payload와 전체 readback |
| PNPM minimal manifest·whole tree renderer/readback, pnpm_inspection_payload.go | YES | YES(PNPM) | YES(existing consumers) | YES(existing scope) | Bun typed layout와 fixed root 재사용 |
| Bun exact instance lifecycle/runtime | NO | NO | NO | NO | 대상 명령·sealed topology·창조자/one-shot admission·실제 정상/실패 경계 |
| Bun static-all/dynamic coverage·Policy/cache/transaction/CLI | NO | NO | NO | NO | 실행 결과를 기존 inspection/Policy owner에 연결; 단계별 후속 검증 |

## B4d-a: 검증된 실행 데이터의 이송

1. `internal/sandbox/pnpm_inspection_payload.go`: 기존 renderer와 copy/readback body를 private
   data-only layout interface/fixed native kind helper로 기계 추출한다. PNPM wrapper는 원래
   metadata 제외·root·오류 의미를 유지한다. Source bytes/digest/mode/UID/GID와 hardlink
   inode class, symlink·전체 census/한도·취소·최초 cause 검사를 재사용한다.
2. `bun_inspection_payload.go`: opaque Bun layout만 받아 같은 renderer를 사용한다. Bun에는
   PNPM metadata 제외 규칙을 적용하지 않는다. Private payload는 생성 kind를 보관하고
   fixed Bun root의 copy/readback만 허용한다. Original controls/cache/seed/marker는 넣지
   않고 layout의 controller-owned dependency-only manifest와 검증된 node_modules만 담는다.
3. `bun_inspection_payload_test.go`: complete positive에서 시작해 whole bytes/manifest,
   root scripts/config 부재, changed/missing/extra/mode/link/digest·다른 kind/zero/cancel 및
   copy/readback 첫 cause와 실패 후 dependent 작업 부재를 검증한다. 기존 PNPM tests 유지.
4. `bun_inspection_payload_real_test.go`: fresh public peer archives/SRI/static→production Bun
   prepare→renderer→fresh fixed Bun project scope copy→whole independent readback을 확인한다.
   Native package는 격리 안에서만 펼치고 Host extraction이나 artifact script 실행은 없다.
   공유 renderer의 기존 PNPM production project consumer도 같은 source로 확인한다.

두 번째 검토: 기존 구현을 대체/재작성하지 않고 검증된 body를 두 typed 소비자가 재사용한다.
새 Core/Policy/runtime protocol·범용 실행 API를 만들지 않는다. Fixed kind는 경로 선택이며
payload에 안전성/실행 권한을 부여하지 않는다. 새 interface는 현재 두 data 소비자만 가진다.
Observer/FS/network 권한·runtime pin은 이 data 단위에서 변경하지 않는다. No push/new CI;
다음 backend/end-to-end 묶음과 실제 타이머 규칙은 확장 계획§5를 유지한다.

## B4d 후속 실행 의미 조사

고정 Bun source38afee6의 `lifecycle_script_runner.rs`(Git blob8f92cfb730f1ce8c2c47703c850ffdca8c64878a)는
POSIX shell 존재 시 `sh -c`, 없을 때 `bun exec --no-env-file`을 사용하고 lifecycle event/script
환경을 각 chain에 넣는다. [공식 lifecycle 문서](https://bun.sh/docs/pm/lifecycle)의 trusted 목록은
package 이름 기준이며 설치된 전체 instance의 검사 완료 authority가 아니다. 현재 named
payload 검증을 Bun runtime 또는 whole lifecycle로 확대하지 않는다. Exact target와 Node/Bun
실행 의미·필수 runtime supply/admission은 이 데이터 단위 다음에 별도 지도로 확정한다.
원본 MCP capture와 upstream blob 일치 derived copy는 p3b4d-bun-lifecycle-script-runner raw가
소유하며 capture의 추가 LF1개를 원본에서 삭제하지 않았다.

B4d-a 첫 Quick56.550s는 caller가 사라진 private verify wrapper의 U1000에서 멈췄다.
원래 readback body는 공용 helper에 보존되어 있고 사용되지 않는 wrapper만 제거했다.
기존 renderer/copy/readback3개 body는 fixed-kind guard/경로 선택을 제외한 정규화 동일성을
확인했다. Actual Bun peer는 fresh public source4·layout63→execution payload65entries/
425,984B 전체 readback15.01s가 PASS다. 같은 source의 기존 PNPM production project는
transitive/peer의2/4 actual fresh instance sessions와 static-all/dynamic-completed47.53s가
PASS다. Combined operator65.119s·helper wait0/Host 전후 동일이며 Policy/Host 설치와
Bun artifact lifecycle는 미실행이다. 최종 Quick/security/docs는 별도 확인한다.
