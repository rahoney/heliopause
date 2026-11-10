# P2c5-B4d — CI405의 compiled fixture 작업 디렉터리 교정

M13-004/P2의 같은 work item이다. CI/evidence/sandbox 오답노트와 canonical CI·quality·coding 규칙,
B4c 지도/영수증 및 실제 scripts/ci-pnpm-runtime.sh 호출부를 다시 확인했다.

## 변경 전 baseline audit

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| PNPM runtime 공급·기존 startup4 | YES | YES | YES(CI405) | NO(P2) | 없음 |
| 실제 project static-all/instance 집계 | YES | YES | YES(CI405 public6) | NO(P2) | 후속 전체 qualification |
| controlled instance actual3 | YES | YES | YES(named local) | NO | CI의 fixture package cwd |
| 기존 authenticated client/helper cleanup | YES | YES | YES(CI405 first failure 보존/restore) | NO(P2) | 없음 |
| PNPM CI package scope 검사 | NO | NO | NO | NO | 아래의 좁은 교정과 actual 소비자 |

CI405/head7e966e7은 Linux/Required failure이고 다른8개 job은 success다. Linux 최초 오류는
controlled-success의 `../artifact/npm/testdata/pnpm-v9-single.synthetic.yaml` ENOENT다.
직접 compiled binary 실행은 go test와 달리 package cwd로 전환하지 않는다. 로컬 actual은
package cwd였지만 새 CI 호출은 repository cwd였다. Fixture는 tracked 상태이며 파일 누락이나
PNPM product/메모리 오류로 해석하지 않는다. 바로 앞 public project6개는 PASS이고, 뒤 public
Policy consumer와 기존 Linux 후속 검증은 미도달이다. Helper wait0/CONFIRMED_STOPPED,
client_error0/client_restore0/RESTORED 및 original1이 최초 실패를 보존한다.

## 정확한 edit map과 source 작성 전 두 번째 검토

`scripts/ci-pnpm-runtime.sh`의 installed controlled 호출 하나만 기존 UV/Cargo처럼
`( cd internal/sandbox; exact-command )`의 여러 줄 subshell로 감싼다. Supply의 root cwd와
나머지 명령, fixture bytes/실행 filter/client identity/helper owner/timeout은 보존한다.
`scripts/check/pnpm_runtime_workflow.go`는 이 정확한 호출의 package subshell을 요구하고,
기존 test에서 missing/wrong/commented cd·unscoped·duplicate·command 밖 scope를 거부한다.
임의 Bash 의미나 privileged 실행 성공을 이 구조 검사로 대체하지 않는다.

실제 같은 constructor를 compiled data consumer로 repository cwd 실패→package cwd 성공 확인 후,
기존 canonical format/Quick/security/docs/workflow·Bash syntax를 실행한다. Actual installed ordinary
client의 controlled3과 미도달 public Policy 소비자를 기존 mount namespace/helper lifecycle로
확인한다. 이전 public project6 성공/CPP112 source는 변경하지 않았으므로 같은 소비자를 반복하지
않는다. Local B5a8a4는 failing remote7e 이후 변경이며 그 원인으로 합치지 않는다. 수정 단위와
B5a를 한 번 묶어 push하고 다음 원격 결과를 새 source/checkout에 연결한다.

원본: `docs/github-logs/package-client-quarantine-20261021/ci405-linux.log`, ci405-run.json,
ci405-jobs.json. 이 경로들은 보존 raw이며 새 CI 입력이 아니다. Whole P2 acceptance는 NO다.

## Named local 결과와 원격 경계

CI405의 source7e와 actual checkout5e2b42a52406883e14f47f7939bb6afadccf8372는 전체tree
d864b870061bb35fabb7b3fd0d96457fa33ba797가 같다. Decoded Linux UTF8 log420602B를
tool의 전체 decoded text와 byte-equivalent하게 대조했다. 원시 HTTP 압축 bytes라는 주장은 아니다.

같은 constructor의 compiled data consumer는 repository cwd0.021s/FAIL→package cwd0.021s/
PASS다. 교정된 caller/검사기의 canonical format2.221s/Quick66.975s/security8.764s/docs0.230s/
workflow0.673s와 Bash syntax가 PASS다. Actual client compile6.084s를 별도로 확인했다.

Repository cwd에서 script의 정확한 controlled package subshell을 실행한 actual134.277s는
controlled3/test17.36s와 기존 public8/test114.47s·그중 실제 검역4/8entries의 stored Evidence/
Policy ALLOW를 통과했다. Exit37의 원래 cause/사실 보존, 긴 이름130자도 그대로 확인했다.
Helperwait0/CONFIRMED_STOPPED·original0·client_error0/client_restore0, runtime directory 정리,
Host 보호 파일·Docker 구성 전후 동일이다. Production 검사·관찰·cache source를 바꾸지 않았다.

[로컬 영수증](./p2c5b4d-pnpm-ci-fixture-local-20261025.json)이 이 scope를 소유한다. 로컬 경로
교정 QUALIFIED=YES이며 원격 전체/whole P2 ACCEPTANCE_CLOSED=NO다. 다음 원격은 B5a와
묶고 CI405의 미도달 범위를 새 source에서 확인한다. 작은 helper별 재push/임의 rerun은 없다.
