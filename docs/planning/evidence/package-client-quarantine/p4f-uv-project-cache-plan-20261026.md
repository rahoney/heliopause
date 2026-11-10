# P4-F native project approval/cache 연결

Baseline 9ac602a (P4-E), tree05205572d7782d5f2e654c0d34302eb841ddced9/2504files.
Read: current Queue/plan20, lessons INDEX·CI L06·sandbox L08/L14·artifact L15·evidence L16,
Architecture evidence/staging/promotion와 해당 Domain contract, P4-A..E와 기존
PythonProjectCache/UVSync publication/bootstrap inspection composition.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Native project selection/original six controls | YES | YES(typed resolver) | YES(P4-E local) | YES(named selection) | approval 진입 시 provenance 재검증 |
| Complete Python inspection/current Evidence·Policy | YES | YES(UV pip) | YES(P1) | YES(P1 scope) | native project selection 소비 |
| Python approved cache/whole wheel metadata/records/readback | YES | YES(UV pip) | YES(P1) | YES(P1 scope) | native provenance 검증 뒤 동일 engine 연결 |
| Native project Host transaction/CLI/tools | NO | NO | NO | NO | atomic manifest/lock+venv publication, ownership, lock-only와 installed 구분, CLI/tools |

## Exact map와 두 번째 검토

새 cache engine이나 Policy/Domain 모델을 만들지 않는다. Sandbox의 기존
parseUVProjectOriginalControls/SelectManifest/native lock/export parsers를 재사용해
완전한 typed update의 여섯 original/selected controls, 불변 intent/current runtime,
add conservation/selected version, export의 네 exact provenance controls 및 complete wheel
selection을 검증하는 순수 진입 함수를 추가한다. Snapshot의 digest 연결은 기존 Domain
constructor가 소유한다. 이 함수는 approval을 부여하지 않는다.

Promotion의 작은 UVProjectCache adapter는 frozen update를 검증하고 기존
PythonProjectCache.stageProject/openProject를 그대로 사용한다. Open도 provenance와
기존 current Evidence/receipt/파일 census를 다시 확인한다. 새 path 기반 신뢰나 native
출력의 approval authority는 없다. 기존 UV pip cache/session/observer bodies는 수정하지 않는다.

두 번째 검토에서 original opaque lock을 approval로 쓰지 않고 native parser로 읽는 점,
raw export와 derived view를 구분하는 점, 모든 selected control의 exact bytes/presence와
변경 금지 runtime/intent를 확인하는 점을 대조했다. Host publication은 이 data-only 단계의
성공에 포함하지 않는다. 이를 별도 outer transaction으로 덧붙여 기존 rollback을 잃지 않는다.

검증은 canonical format/quick/security/docs, affected tests/Race, 실제 public native
selection→기존 complete inspection/Evidence·Policy→cache/Manifest/SBOM readback 및
provenance/파일/Evidence tamper 거부다. 고정 공급물/canonical121 observer와 helper의 기존
namespace/Host 불변 절차를 재사용한다. 작은 helper별 push/full CI는 실행하지 않는다.

## Named local 결과

[Cache 영수증](./p4f-uv-project-cache-local-20261026.json)은 canonical profiles와 affected full Race,
actual51.282s/test48.22s를 연결한다. Two roots39.46s/empty7.13s의 실제 complete 검사와
current Evidence·Policy·six-control Manifest/SBOM/cache readback이 PASS다. Approval 부재,
주입한 current Evidence reader 접근 실패와 봉인된 Manifest 변조는 거부했고 Host target은
부재다. Same canonical121/helperwait0/Host 전후 동일이며 기존10 owner bytes는 불변이다.
Host transaction/CLI/tools와 whole P4·remote qualification은 미완료다.
