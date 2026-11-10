# P4-O2 — pipx wheel closure 공급

기준 e6a2c28/tree841e3ccb의 O1 fixed pipx publish 검증을 재사용한다. Read는 O1/P4-O
baseline, plan20 §5, client09, maintenance §2·3·6·7, D-012/013/015와 관련 lessons다.
같은 M13-004만 IN_PROGRESS이며 기존 구현을 대체하지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Pipx main publish/actual bytes | YES | YES(developer) | YES(O1 local) | YES(O1 only) | closure 전체 공급 소비 |
| Dependency publisher proof | NO(formal) | NO | NO | NO | 고정5 publisher의 signed claim 검증; userpath 미제공 경계 유지 |
| Whole wheel ZIP/RECORD/static surface | YES(InspectWheel) | YES(existing UV) | YES(existing scopes) | YES(existing scopes) | runtime closure 소비·의존성/marker·전체 경로 충돌 |
| Metadata normalization/constraints/graph | YES(candidateFromWheelMetadata/bindUVSelectedWheels) | YES(existing UV) | YES(existing scopes) | YES(existing scopes) | 실제 wheel+독립 exact-release metadata를 받는 공급 전용 연결 |
| Immutable materialization/loader/runtime admission | NO(pipx) | NO | NO | NO | closure manifest와 검증된 read-only guest 로딩·single runtime owner |

## 추가 정적 입력 확인

기존 InspectWheel로 실제7개 wheel을 읽은 사설 DATA 진단0.582s/exit0이 전체 ZIP/RECORD와
static surface를 통과했다. Argcomplete23/click22/filelock28/packaging29/pipx58/
platformdirs18/userpath12 files이며 native extension/data scripts는 없다. 이 진단의
digest는 읽은 bytes에서 계산했으므로 독립 digest authority가 아니다. 각 제공 proof5개의
certificate OID23/24는 예상 release/publish/pypi environment와 해당 repo subject를
DER UTF8String으로 포함한다. Crypto와 이 DATA parse를 합친 formal 검증은 아직 미완료다.
Local raw p4o2-static-source/stdout/stderr/result가 원본이며 Host Python/upstream code는 실행하지 않았다.

첫 formal actual1.783s는 pipx의 정상 man page를 새 namespace gate가 거부했다. 기존
InstalledFile의 SchemeData/RESOURCE와 import/site·dist-info를 분리해 공급 manifest에 보존한다.
Resource는 runtime 안전성이나 import 권한이 아니며 이후 loader의 import root에 넣지 않는다.
또한 기존 InspectWheel의 RECORD row는 정상 self-hash exemption으로 size0/hash empty다.
기존 parser 계약을 바꾸지 않고 이미 검증한 archive의 그 bounded member bytes만 읽어
immutable manifest의 실제 hash/size를 채운다. RECORD를 빈 파일로 추출하거나 재작성하지 않는다.
새 정상 resource/RECORD manifest와 scripts/외부 import namespace 거부 회귀를 함께 확인한다.

두 번째 actual2.133s는 기존 bounded requirement validator가 실제 `userpath!=1.9,>=1.9.2`의
numeric exclusion을 지원하지 않아 거부했다. 기존 UV/사용자 requirement 언어를 확대하지 않고
O2 공급 전용에서 final numeric `!=`를 기존 equality/trailing-zero 비교로 확인한 뒤 나머지
제약을 그대로 기존 validator에 위임한다. Excluded version·wildcard/prerelease exclusion은
거부하고 authenticated metadata의 원문은 보존한다. Resource 교정이 이 기존 언어 제한을
유발했다고 해석하지 않는다.

교정된 실제 command1.353s/exit0 뒤 operator는 종전 census의 source/stub148개를
`.py` 파일 수로 오해해 assertion으로 중단했다. 실제190개 파일에는 `.py`147개와
`.pyi`1개가 있다. 기존 main census52는51 `.py`+1 `.pyi`이며 dependency96 `.py`와
합쳐 source/stub148이다. CLI 성공과 operator의 실패/후속 미도달을 분리하고 같은 정상
입력의 재실행 대신 보존한 성공 stdout을 재사용한다. Raw census/log는 수정하지 않는다.

## 수정 지도와 두 번째 검토

1. 기존 releaseprovenance의 O1 내부 검증 로직을 private helper로 공유한다. 기존 exported
   PipxPublishIdentity/Receipt/VerifyPipxPublish 정책·정상/부정 tests는 유지한다. 새로운 exported
   generic publisher 설정은 만들지 않는다. Dependency API는 argcomplete/click/filelock/
   packaging/platformdirs만 fixed repository/workflow/IDs/environment/trigger로 선택한다.
   Packaging은 실제 main/workflow_dispatch이며 tag push로 꾸미지 않는다. Expected source/run/
   version/digest와 SCT/log/timestamp, signed environment/subject를 각각 대조한다. 작은 실제
   공식 proof fixture로 wrong source/run/digest/publisher·서명 변조·cross-authority를 검사한다.
2. `internal/artifact/pypi`에서 기존 whole archive/candidate/marker/constraint/graph helper를
   재사용하는 공급 전용 typed closure를 추가한다. Native lock/installed receipt를 조작해 호출하지
   않는다. 실제 wheel bytes와 독립 exact-release metadata의 URL/size/hash를 대조하고 complete
   reachable closure·현재 Python 제약·startup/native/설치경로 충돌을 검증한다. 결과는 static
   사실이며 runtime/Policy/설치 authority가 아니다. Default tests는 작은 synthetic wheels와
   metadata를 명시해 closure 누락/extra/변조/constraints/marker/path 충돌을 거부한다.
3. 기존 developer update-runtime에서 공급 proof와 typed closure를 조합한다. 필요 import는
   이 composition script의 artifact provider 소비로 한정하여 checker allowlist에 연결하고
   Core/Application/Policy의 dependency 방향은 유지한다. 별도 version lock이나 CI job은
   만들지 않는다. Single manifest는 후보 입력이며 현재 product runtime pin이 아니다.
   Userpath의 공식 signature endpoint404는 서명 성공으로 바꾸지 않고 독립 PyPI metadata의
   exact digest/크기와 전체 RECORD 검증 사실로 명시한다. 제공 proof는 전부 검증한다.

검증은 canonical format/Quick·영향받은 전체 Race와 frozen7-wheel 실제 공급을 순차 실행한다.
전체 index/blob/mode/disk와 docs/security를 대조하고 의미 있는 중간 commit으로 보존한다.
공급만의 성공을 immutable loader·runtime admission·native startup Artifact 역할·public CLI/
whole P4/최종 hosted matrix로 확대하지 않는다. 다음은 실제 공급 형태에 연결한 loader/runtime
수정 지도를 확정하며 작은 helper마다 push/full CI를 만들지 않는다.

## 한정 로컬 검증 완료

[O2 receipt](./p4o2-pipx-closure-local-20261027.json)에 source/input identity와 원본을 연결했다.
Final Quick232.433s/네 package 전체 Race77.646s/docs0.429s/security17.034s가 PASS다.
실제 공급1.353s는7 wheels/190files/제공 proof6을 확인했고 userpath는 NOT_PROVIDED다.
일곱 실제 변조 입력은 stdout 없이 거부됐으며 frozen27파일은 불변이다.
O2 공급 범위는 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED 모두 YES(local)다.
Immutable loader/runtime·native·CLI·전체 P4/final profiles는 별도 MISSING으로 유지한다.
