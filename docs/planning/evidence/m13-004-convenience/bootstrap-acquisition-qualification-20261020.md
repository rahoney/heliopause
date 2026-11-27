# M13-004 — ordinary 최초 획득·리포트 로컬 검증

Parent `87063045328f0b808927fd1c548e32041b4e86be`. [사전 감사와 수정 map](./bootstrap-acquisition-plan-20261020.md)을
따랐고 기존 fixed public client/frozen set/SDK/진행 표시를 재사용했다. Root helper의
network·일반 입력·권한은 늘리지 않았다. [설치 인계](../../../user-journey-cli-ia/08-native-bootstrap.md)는
실제 OS 설정/first trust/등록/일반 계정 건강 검사와 남은 acceptance를 구분한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| whole lock bytes의 generated digest | YES | YES, ordinary/Root initial | YES, local | YES, component | 없음 |
| frozen held-FD reader/facts | YES | YES | YES, local | YES, component | 없음 |
| ordinary bounded acquisition/export | YES | YES, bootstrap prepare | YES, local fake-positive/SDK-negative | PARTIAL | actual signed native9 |
| 7언어 CLI/progress/정직한 report | YES | YES | YES, local | YES, component | 없음 |
| OS first trust/setup/health·인증 | PARTIAL | PARTIAL, operator handoff | NO, whole path | NO | clean signed host·실제 service/ordinary health/기존 sudo 인증 |
| signed base→target/systemd/crash/reboot | PARTIAL | PARTIAL | NO | NO | 실제 새 signed base/target |

Generated digest는64KiB 한도 안의 parse와 동일한 원본 bytes를 해시한다. Lock/runtime
버전은 바꾸지 않았다. Initial ordinary/Root 모두 compiled whole digest를 요구한다.
Frozen reader는 held FD의 독립 read-only descriptor를 사용한다. Live pathname 대체나
source close 뒤에도 같은 inode를 읽으며 resolve/download fallback을 만들지 않는다.

`helox bootstrap prepare --version vX.Y.Z`는 일반 UID/Linux amd64/exact version을
HTTP·directory 이전에 검사한다. Native4+metadata3와 provenance의 exact source를
SDK로 검증하고0700/0600의 owned export만 만든다. Source close/intake 정리 이후 실제
export inode/UID/mode/size/mtime/전체 digest를 다시 확인한다. Cleanup 실패는 primary
cause를 덮지 않고 known-owned 파일만 정리한다. PREPARED는 provenance VERIFIED이며
quarantine NOT_PERFORMED/installation PENDING이다. Root 실행·서비스 설치나 ALLOW가 아니다.

원본 owner는 `docs/github-logs/self-update-20261020/bootstrap-acquisition-*`다.
Final-source inventory가 private와 workspace의 전체 tracked+명시 candidate bytes를 대조한다.
고정 Go1.26.9 canonical format과 Quick125.17s/focused2.52s/security8.23s/
vulnerability21.54s/docs0.32s/관련 전체 Race33.75s는 exit0이다. Race는 generator,
GitHub source, releasebootstrap, ports, CLI/bootstrap, releaseactivation을 포함한다.
Root bootstrap namespace31 cases0.27s/exit0은 compiled digest 변경 후에도 기존
Root 경계를 유지했다. Positive crypto는 mock이고 production SDK는 unsigned set를 거부한다.
Namespace binary29,109,026B는 삭제했고 host native root/state는 전후 ABSENT_CONFIRMED다.

Actual CLI42,450,874B/helper27,274,694B의 build1.58s/1.05s는 exit0이다. Go1.26.9
module tuples에서 helper71개가 CLI82개에 포함된다. 일곱 언어 실제 도움말 exit0과
invalid/duplicate version2개 exit1을 확인했다. Darwin bootstrap/CLI TEST compile은
amd641.18s/0.77s, arm641.24s/0.82s 모두 exit0이다. Native macOS 실행이 아니다.
네 cross binaries와 두 실제 binary, telemetry를 포함한 work directory/operator를 삭제했다.

두 번째 대조에서 첫 FD fixture의 filename 오해와 테스트용 onClose 미호출을 고쳤다.
두 최초 focused 실패 로그는 준비 스크립트의 같은 filename 재사용으로 덮였으므로
제품 RED나 전체 원본 보존을 주장하지 않는다. operator-notice가 도구 세션과 한계를
보존하며 최종 검사에는 별도 immutable final prefix를 사용했다. Actual probe의 첫 정리는
Go가 만든 XDG telemetry 디렉터리를 flat-file finalizer가 거부해 receipt 전에 중단됐다.
별도 cleanup notice 뒤 새 실제 probe receipt를 확보했으며 제품 실패로 해석하지 않는다.

구현 후 Root helper imports/entry, old-peer update chain, Policy/Evidence와 실제 export
readback을 다시 대조했다. 현재 설치본은 변경하지 않았다. Public initial acquisition 소스
연결은 닫았지만 actual signed acquisition/whole OS bootstrap·ordinary health·기존 OS
관리자 인증과 real systemd/crash/reboot는 닫지 않았다. M13-004는 IN_PROGRESS다.

OS 인증은 기존 lifecycle plan의 `sudo -v`→fixed `sudo -n` 계약이다. 새 sudoers 또는
version별 위임 정책을 별도 필수 acceptance로 만들지 않도록 설치 안내를 대조·교정했다.
