# CI416 native tool fixture UID 조사

M13-004 하나 아래 CI416/source1b16544의 Linux·Required 실패를 조사한다. 현재 local
HEAD41a8272는 이후 문서만 추가됐고 실행 source는1b와 같다. All10 decoded logs의 checkout은
같은 d6953e0이며 full tree 대조는 이어 확인한다. First failure는 새 native tool producer의
base-python3.14 canonicalize EACCES, TestLinuxUVToolNativeAdoptionCLIIntegration0.77s다.
Native current inspection/승인은 NOT_REACHED다. 기존 tool 네 소비자·PNPM peer는 PASS이며
helperwait0/CONFIRMED_STOPPED·RESTORED/primary1은 보존됐다. CI415 원인 해결로 확대하지 않는다.

Read: Queue/plan20/client09, N2D source/actual receipt, shared uv_native_fixture writer·Docker
identity/cleanup와 tool-only bind mount, lessons artifact L15/sandbox L08/L18/ci L06/evidence
L11/L16을 대조했다. 원본은 local raw owner의 ci416-*-decoded.log/decoded-log-inventory다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 실제 native 원본 producer와 raw 경로 보존 | YES | YES(actual native consumer) | NO(hosted failure) | NO | UID가1000과 다른 Host에서 private bind 원본 생성 |
| Bounded Docker I/O/image identity/cleanup | YES | YES | YES(existing scopes) | YES(named scopes) | 재작성 없음; 새 UID 경계에서 그대로 확인 |
| Current native 승인/Manifest/publication | YES | YES | YES(local named) | NO(hosted) | producer 교정 뒤 정확한 named actual/새 hosted qualification |
| 기존 managed/fresh/current Evidence 및 PNPM peer | YES | YES | YES(CI416 named) | YES(named only) | whole profile acceptance와 CI415 원인은 별도 |

## 재현·수정 지도와 두 번째 검토

신규 tool fixture는 Host private root를 같은 absolute guest 경로로 bind하지만 create/exec와
두 tmpfs UID/GID가1000으로 고정돼 있다. Local 실행 user1000에서는 통과했고 hosted는
그 경로에서 EACCES였다. Remote uid의 numeric 값은 원본 로그에 없으므로1001이라고
추측만으로 확정하지 않는다. Product guard나 observer 오류라고 단정하지 않는다.

우선 실제 fixed UV/Python fixture를 별도 ordinary uid1001에서 실행해 같은 원인 유형을
재현한다. 전용 private source의 test driver만 사용하고 candidate driver source SHA를
고정한다. 기존 current product를 호출하지 않는 producer 진단이며 qualification이 아니다.
그 뒤 tool-only guest create/exec와 tmpfs owner만 실제 Host ordinary uid/gid에 맞춘다.
기존 default native pip의1000/network-none/no-bind 및 writer/export·cleanup은 보존한다.
0700을 넓히거나 Host 대상/원본을 chown하거나 container root/capability·budget으로 우회하지 않는다.
현재 tool test의 bounded log에 host/guest uid/gid를 기록해 다음 차이를 진단 가능하게 한다.

기존 구현을 대체하지 않고 이 acceptance gap만 교정한다. 같은 UID1001 driver의 RED→GREEN,
canonical Quick/docs/security 및 tagged build, concrete tool native actual과 기존 shared native
pip/project 회귀를 순서대로 확인한다. 실제 필요 경계의 새 grouped hosted CI에서 Linux·Required와
whole checkout/raw evidence를 대조하기 전에는 hosted 해결을 선언하지 않는다.


## 교정 전 UID1001 실제 재현

같은 fixture를 호출하는 고정 private driver의 UID1001/GID1001 실행은1.668s/exit1로
동일 base-python3.14 canonicalize EACCES를 재현했다. Guest exec는1000:1000이며 Host의
0700 원본은1001 소유다. NoSKIP/보호5파일·Docker 동일이고 product current approval은
NOT_RUN이다. Remote의 정확한 numeric uid는 계속 미관측이지만 후보의 UID 불일치 실패
조건은 직접 입증됐다. Driver/compile/binary/raw를 보존한다.

확정 edit 위치는 common fixture의 tool branch에서 actual uid/gid 선택, create/exec의
동일 user 문자열, 두 private tmpfs uid/gid뿐이다. Default branch는 계속1000:1000이고
기존 data writer/export·권한·bounded cleanup과 product source는 유지한다.


## UID1001 producer 교정 검증 — 전체 native/hosted 미완료

같은 driver SHA의 교정 후 실행은1.818s/exit0이며 실제 public native 원본/raw receipt를
생성했다. Host1001/guest1001:1001/private0700을 로그로 확인했고 보호5파일·Docker는
전후 동일이다. 이 RED→GREEN은 producer 한정이고 product current approval은 NOT_RUN이다.
Private driver는 후보 검사/실제 client 빌드 전에 제거했다. 기존 default native pip는
계속1000/1000이며 rootless 제품 경로·observer·Policy·resource budget을 바꾸지 않았다.
원본은 ci416-uid-{before,after}-*와 고정 driver text다.

All10 checkoutd6953e0의 전체tree616b506b/2657은 source1b16544와 같다. Metadata가
고정하는 두 parents도 baseef5ef0e/source1b이며 별도 whole-checkout-proof를 보존했다.
CI416의 PNPM peer PASS는 이전 FD_STATE_UNKNOWN의 원인 해결 증명이 아니고 CI415는
계속 OPEN이다. Fresh/managed tool4 current 승인은 이 source의 named hosted 결과이며
실패 뒤 CPU/final profile 실행은 성공으로 표시하지 않는다.


## Corrected current native 실제 소비자

Final code의 canonical Quick281.578s/Bootstrap 전체 Race17.703s/tagged compile2.616s가
PASS다. 같은 binary8182690e의 native actual168.356s는 exact named1/noSKIP로 현재 원본
exact6.0 승인·Manifest10/반입, managed 반복의 새 Evidence/Manifest8, authentic bounded
6.1→6.0과 원래 조건 보존·old metadata 제거, 실제 Evidence permission failure의 원본
보존을 통과했다. Host1000 full current scope와 UID1001 producer-only scope를 구분하며
helperwait0/final CONFIRMED_STOPPED·보호5파일/Docker 동일도 확인했다.
[Local receipt](./ci416-native-fixture-uid-local-20261027.json)가 명령/원본/source hash를 연결한다.
Affected default native project/complete sync actual은 진행 중이고 hosted 해결은 아직 미검증이다.


## Affected default 실제 결과와 최종 후보

동일 binary의 affected default actual275.437s는 exact named2/noSKIP이며 기존 native
project add/sync117.70s와 complete multi-root/retained/empty/native sync155.88s를 통과했다.
Helperwait0/CONFIRMED_STOPPED·보호5파일/Docker 동일이며 actual의 모든 실행 입력은
최종 코드와 같다. 이후 차이는 문서뿐이다. Supply/기존 bounded Docker·data writer/native
pip export 세 body는 baseline41a8272와 bytes가 같다. No product source/runtime/observer/
Policy/resource budget 변경이 없고 새 driver/package/config/job을 후보에 남기지 않았다.

수정은 IMPLEMENTED=YES/WIRED=YES/named local QUALIFIED=YES이며 corrected hosted
ACCEPTANCE_CLOSED=NO다. 최종 staged docs/security·whole candidate 결과는 companion
raw evidence에 연결하고, 원인/입력이 바뀐 이 후보의 실제 hosted acceptance에서만 새 CI를
실행한다. CI416 실패나 CI415 FD 원인을 새 로컬 PASS로 덮어 쓰지 않는다.
