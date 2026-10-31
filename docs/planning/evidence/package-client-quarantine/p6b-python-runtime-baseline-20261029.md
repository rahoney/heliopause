# P6-B — readonly Python user-run 전 기존 owner 감사

Baseline3c85d8271e2002f6e5cc5df89fd20145d72be719/tree82c911ab/3019tracked·1351runtime.
CI430/run40816740622은 이 exact source의 pending hosted qualification이다. 첫 실제20min
query는 timer01/session1333 due04:35:37UTC(한국13:35:37)다. 새 push로 취소하지 않는다.
M13-004 하나만 active이며 이번 변경은 source audit/registration metadata다.

Read: Queue/PROJECT/docs route, INDEX의 artifact L01/L02·sandbox L03/L08/L12/L18·evidence L16·
ci L05/L06, Architecture005/006·D014·M5 contract§3/8·client09§3/plan20, A1/A2/A3/A4 및
CI429 qualification, 다음 실제 source/function owners를 대조했다.

| 실제 owner/scope | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Domain.ExecutionContext/current whole approval/argv/runtime/source-policy binding | YES | YES (A3) | YES (actual3 local) | YES (named) | 기존 계약 재사용 |
| Application inspection/preparation/run/records and safe output | YES | YES (Node/A3 preparation) | YES (named local/CI429 Node) | YES (named) | concrete Python runtime adapter |
| A2 controlled console source/current A3 Python preparation | YES | YES (internal) | YES (actual3 local) | YES (named) | selected console와 same whole readonly input 결합 |
| pypi_closure_manifest.go: buildClosureManifest | YES | YES (M5) | YES (named existing M5/current A3) | YES (named) | fresh approved execution context와 selected declaration binding |
| pypi_build.go: introduce / digest of consumed Docker input bytes | YES | YES | YES (named M5) | YES (named) | 그대로 재사용 |
| pypi_closure_installation.go: verifyClosureInstallation/full tar census | YES | YES | YES (named M5) | YES (named) | generation/observed bytes와 exact entry metadata 소비 |
| pypi_closure_volume/observation_runtime: identity/RO mount/attachments/anchor | YES | YES | YES (named M5) | YES (named) | same context 전용 readonly DATA view |
| pythonTransactionExecutor.startPhase/terminatePhase/verifyAnchorAndVolume/cleanup | YES | YES (inspection) | YES (named M5) | YES (named) | preparation-only operational consumer에서 기존 lifecycle 호출 |
| observation CPU watch/ledger/resources | YES | YES | YES (named) | YES (named) | real accounting 유지, inspection/user-run authority와 분리 |
| current approved Python readonly input validator | NO | NO | NO | NO | same source/metadata/whole install·sole RO anchor/complete trace/CPU/cleanup |
| Python console user-run closed admission/terminal/output | NO | NO | NO | NO | 별도 runtime profile/actual 정상·nonzero·신호·취소·overflow·악성/cleanup |
| native UVx golden/actual aliases·CLI | NO | NO | NO | NO | internal tool-install selection을 UVx라고 바꾸지 않고 별도 입증 |

## 실제 source에서 확인한 재사용 경계

- buildClosureManifest는 canonical wheel reader·RECORD/installed mapping·owner/collision과
  generated wrapper namespace를 이미 검증한다. Installed census는 실제 생성 bytes/mode/uid/gid
  전부를 별도 identity에 넣는다. Generated wrapper의 측정은 callable 실행 성공이 아니다.
- PythonArtifactIntroducer.introduce는 Docker가 소비한 stream을 TeeReader로 hash하고
  실제 consumed size와 original archive digest를 확인한다. 앞 parser pass만으로 두 번째
  file open을 인증하지 않는다. 이 기존 stream boundary를 재사용한다.
- 기존 preparation→RO anchor overlap→writer termination→full census/attachment checks를
  이용한다. Zero attachment tmpfs 보존을 가정하지 않는다. Existing create/read/kill/Close와
  CPU monotonic/transition/poll/cleanup bounds를 유지한다.
- newObservationTransaction은 실제 코드상 zero units를 허용한다. 이는 no-observation
  preparation의 operational CPU accounting에 사용할 수 있는 표현이며 새 consumer의 실제
  qualification은 아직 없다. Dummy required unit/가짜 SandboxResult/현재 Policy를 만들지 않는다.
  Data 준비에서 ledger finalize를 쓰더라도 의미는 accounting/teardown뿐이고 검사 coverage나
  selected program 완료 결과로 변환하지 않는다.
- executeObservationTransaction의 unit loop와 final SandboxResult는 inspection 의미를 가진다.
  User-run을 fake unit으로 넣지 않는다. 기존 install/census와 anchor transition의 동일 본문을
  private named helper로 추출해 원래 M5 caller와 current preparation-only consumer가 호출한다.
  기존 M5 codes/cause·Pipx branch/required coverage/Policy·resource/observer contract는 유지한다.

## 다음 B1의 수정 지도와 두 번째 검토 기준

첫 source 단계는 public-PyPI original wheels/default limits/pinned Python의 readonly DATA
validator만 다룬다. Actual A3 current context와 A2 opaque console entry를 함께 받아 full
acquisition/command/metadata member/path/SHA/size/runtime을 확인한다. Whole approved closure의
canonical manifest→authenticated stream→offline pip install→full census→sole RO anchor를
기존 owners로 수행하고 CPU/observer/전체 cleanup까지 기다린다. Context/Cause/첫 오류를 보존한다.

No Host target/cache approval/extra dependency/permission, no selected user program/import/callable,
no native UVx alias/descriptor. Python user-run admission/terminal/output/CLI는 별도 MISSING이다.
기존 공유 Python lifecycle 변경은 canonical Quick/관련 전체 Race와 실제 Python·Pipx 소비자로
검증한다. CPP131은 이 DATA 단계에서 변경하지 않는다. 새 shared helper는 같은 구현을
재사용하는 current gap이며 기존 기능을 다시 만들거나 대체할 필요는 없다.

Source 전에는 정확한 B1 file/symbol/test 지도와 독립 두 번째 검토를 추가한다. 새 runtime
완료를 source-only audit로 주장하지 않으며 Source1351 불변·canonical docs/security로 이
감사/등록 변경만 검증한다. CI430 timer를 먼저 소비하고 미완료면 actual query 기준 다음20min을
등록한 뒤 독립 작업을 이어간다.
