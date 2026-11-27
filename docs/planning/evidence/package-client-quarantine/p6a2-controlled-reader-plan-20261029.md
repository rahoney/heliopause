# P6-A2 — controlled intake snapshot과 console DATA reader

Baseline21ab828510d08b30e3a268d875736312b751fee5/tree45219ac4/3006tracked·1346runtime.
M13-004 하나만 active. CI429/source9eb/1343은 불변이고 timer04/session48576 due03:17:34UTC.
Read: Queue/PROJECT-DECISIONS/docs routes, routed lessons, client09§3/plan20§4..6,
P6 baseline/A1 receipt, StaticInspector.inspect/artifactPath, public Intake, npm anchored reader,
ResourcePolicy/default WheelLimits와 기존 실제 ZIP/RECORD fixture를 대조했다.

| owner/scope | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Intake handle/variant filename routing 및 canonical parser | YES | YES | YES | YES | 기존 routing을 pure helper로 공유; parser 재작성 없음 |
| A1 opaque inventory/exact console metadata selection | YES | NO (runtime/CLI) | YES (local DATA) | YES (named DATA) | 실제 controlled intake bytes를 공급 |
| held intake root→bounded owned snapshot→A1 inventory | NO | NO | NO | NO | anchored regular file/size/filename와 immutable snapshot, original digest, caller cause |
| native selector/Python runtime/preparation/CLI | NO | NO | NO | NO | 별도 후속; 이번 DATA reader 성공으로 입증하지 않음 |

## 수정 지도 — source 전

StaticInspector의 기존 artifactPath handle/variant routing만 pure artifactLocation helper로
분리하고 기존 Inspect/InspectWheel/InspectSdist의 의미를 유지한다. 새 execution_inventory.go의
ReadConsoleExecutionInventory(ctx, artifact)는 public wheel만 대상으로 같은 routing을 소비한다.
Held os.Root와 Run directory의 Lstat/open identity를 대조하고 symlink/비정규 filename·archive,
잘못된 size/filename/handle/variant를 거부한다. Filename record는512bytes 이내다.
초기 public execution lane은 기존 default wheel compressed64MiB/uncompressed256MiB 한도를
사용한다. Profile 확대나 graph resource charges의 중복 부과는 없다. 별도 current whole 검사와
승인을 대체하지 않는다.

Artifact archive를 context-aware bounded copy로 private memory snapshot에 소유하고 동일
snapshot ReaderAt을 A1 canonical parser에 공급한다. Mutable intake 파일을 여러 parser pass에서
직접 다시 읽지 않는다. Regular file/open identity·size와 종료 시 path identity도 확인한다.
읽기/close/원래 오류와 ctx.Err/context.Cause를 유지하고 어느 실패에서도 유효 inventory를
반환하지 않는다. No extraction/import/native command/network/Host PATH/cache/target/output.

Tests는 기존 prerequisiteFixtureWheel의 실제 inert ZIP/RECORD를 재사용해 canonical console
선택·snapshot 뒤 source 변경과 새 read 거부·파일/Run symlink·잘못된 size/metadata/handle·
취소 원래 cause를 검증한다. Fake approval/set/target/runtime result는 없다.

## 두 번째 검토 — source 전

기존 artifactPath는 filename을 직접 읽으므로 새 reader가 먼저 이를 호출하고 뒤에 root guard를
붙이지 않는다. Pure routing만 공유하고 새 reader는 held root에서 bounded filename을 읽는다.
외부 reader/WheelInspection JSON 주입 또는 mutable file을 opaque authority로 승격하지 않는다.
Snapshot은 소유 bytes의 측정 DATA이며 current Policy/실제 module body/callable/interpreter
실행 성공·native selection equivalence가 아니다. 원본 source·CPP/권한·예산은 그대로다.
Canonical format/focused/Quick, 전체 inspection/PyPI Race, docs/security와 source census를 확인한다.
기존1346 중 static.go 하나만 routing extraction으로 수정하고 new2; runtime1348이며 나머지1345는
bytes 불변을 확인한다. Actual runtime/CPP 재실행이나 작은 helper별 새 CI는 없다.

## 첫 Quick 반증과 좁은 수정 지도

V1 format1.622s/focused6.127s는 PASS였으나 Quick299.785s에서 같은 snapshot의 재측정
Matches가 false였다. 기존 validateRecord는 map을 순회해 Files를 만들며 classifyWheelSurface의
InstalledFiles/SiteHookLines도 그 순서를 따른다. A1의 json.Marshal(info) 지문은 이 순서를
직접 hash해 같은 원본의 재측정을 우연히 거부한다. 기존 parser의 순서 계약 위반이 아니라
새 A1 fingerprint 소비자의 canonicalization 누락이다. Native/CI/runtime 실패가 아니다.

기존 parser/Model C를 변경하지 않고 A1 내부 지문용 private snapshot의 Files·InstalledFiles·
SiteHookLines만 stable key로 정렬한다. 원래 SHA/RECORD·acquisition/member 치환 거부는 유지한다.
기존 execution_entry.go와 test에 실제 원본 반복 재측정 회귀를 추가한다. 따라서 이번 modified는
static.go와 A1 source/test 세 개이며 나머지1343 불변/new2/runtime1348이다. 먼저 같은 새 회귀의
RED를 보존하고 수정 후 GREEN·Quick/두 PyPI 전체 Race/최종 canonical checks를 확인한다.
이것은 구현 재작성/지원 확대가 아니라 A2가 드러낸 기존 A1 소비자 오류의 최소 교정이다.

## 최종 named local 결과

[Receipt](./p6a2-controlled-reader-local-20261029.json)는 첫 Quick299.785s/동일 실제 재측정 RED3.013s와 소비자 내부 정렬 뒤
GREEN·Quick287.612s/전체 두 PyPI Race14.051s·canonical6를 연결한다. 기존1343 불변/
modified3/new2/runtime1348이다. Reader/snapshot과 같은 원본 안정성만 닫았으며 preparation/
Python user-run/native UVx/CLI는 WIRED=NO다. 다음은 기존 generic Application에 current
PyPI 검사와 console declaration source를 결합하는 준비 경로이며 CI429를 보존한다.
