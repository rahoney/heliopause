# P1c C2 — native UV producer와 독립 data reader

기준 e3665ea. M13-004/P1만 IN_PROGRESS다. [C1c query 로컬 검증](./p1c-uv-query-local-20261022.json)은
고정 UV/Python의 실제 startup/query와 complete trace를 확인했다. CI372는 별도 확인 중이다.
이 단계는 public PyPI의 단일 요청 native 선택/완전한 graph 생산이며 update/sync 전체와
CLI를 먼저 노출하지 않는다. 기존 Policy/staging/promotion을 대체하지 않는다.

Read: [P1c 지도](./p1c-uv-resolver-baseline-20261022.md),
[client 설계](../../../user-journey-cli-ia/09-package-client-quarantine.md),
[계획](../../20-package-client-quarantine-plan.md),
[Artifact Port](../../../domain-model/07-contract-artifact-port.md),
[Dependency/Verified Set](../../../domain-model/05-dependency-verified-set.md),
[trusted tooling](../../../threat-model/04-trusted-tooling-and-evidence.md).

## Baseline audit

| 책임 / 기존 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| UV native/volume/admission/query — sandbox/uv_runtime*.go, uv_tool_supply.go, observer.cc | YES | YES(native/CI source) | YES(local query only) | NO(P1) | production typed-network 선택·data transport |
| native pylock/선택 WHEEL/whole archive·독립 release metadata — artifact/pypi/uv_lock.go, uv_release_metadata.go | YES | NO(production) | YES(scoped parser/corpus) | NO | actual producer·frozen controls 연결 |
| pip resolver setup/trace/cleanup — sandbox/pypi_resolver.go resolveDependencies | YES | YES(pip) | YES(existing scope) | YES(existing scope) | 같은 lifecycle의 UV 독립 metadata reader 재사용 |
| stream input/output·one-shot 승인 — direct_exec_admission.go, boundary_helper.go | YES | YES | YES(existing scope) | YES(existing scope) | UV compile stdin/고정 control file tar 사용 |
| installed tree bounded tar reader — pypi_closure_installation.go | YES | YES(existing closure) | YES(existing scope) | YES(existing scope) | UV 선택용 METADATA/WHEEL collector; 인증된 RECORD 검증기로 위장 금지 |
| production UV resolver/frozen graph — 새 sandbox/uv_resolver*.go | NO | NO | NO | NO | 아래 native 형식 확인·구현·실제 normal/negative/cleanup |

## 수정 위치와 방법

1. Exact0.13.4 compile은 stdin requirement와 --format pylock.toml/고정 output file을
   지원하는지 native pin에서 확인한다. Existing boundaryInputExecArguments와
   admission-aware RunInput이 docker exec -i를 이미 지원하므로 shell·Python writer나
   새로운 generic I/O port를 만들지 않는다. User spec은 기존 pypiRequirement의 typed
   grammar로 먼저 검증하고 stdin data만 전달한다.
2. pylock을 native sync/install에서 정확히 소비하는 command를 먼저 확인한다. Native
   JSON delta로 complete graph를 만들거나 solver를 pip로 대체하지 않는다. Wheel-only/
   no-build/no-config/no-managed-download·fixed interpreter/target·copy mode와 source를
   고정한다. --no-build/--only-binary 충돌은 이미 반증됐으므로 재도입하지 않는다.
3. UV runtime/session sibling은 기존 prepareUVTool·named root volume·startTrace/
   awaitMountAnchors·admissionAwareRunner와 typed ResolverNetworkPolicy를 재사용한다.
   NewUVResolver의 constructor/단일 ResolveDependencies만 실제 소비자로 추가한다.
   Unsupported source/stream capability는 network/target mutation 전에 거부한다.
4. 고정 pylock/control file은 existing streamingDockerOutput의 docker cp tar를 bounded
   data로 읽는다. 실행하거나 Host에 tar를 풀지 않는다. Native /haa-site의 METADATA/WHEEL은
   경로·중복·타입·전체 entry/byte 한도를 검사한 stream collector에서만 읽는다. Artifact
   metadata parsing은 artifact/pypi의 기존 RFC822/WHEEL parser를 재사용하고 선택 snapshot을
   기존 SelectInstalledWheels로 연결한다. 코드·INSTALLER·stdout 자체는 authority가 아니다.
   실제 native 조사에서 guest-only /tmp는 docker cp로 보이지 않았다. 출력은 기존
   closureVolume의 identity/attachment 검증을 재사용한 UV 전용 named tmpfs /haa-site에
   둔다. pylock.toml과 site/ 하위만 수집하며 runtime executable volume과 분리한다.
   UV-only expected mount topology를 함께 수정·검증하고 기존 Python profile은 유지한다.
5. 독립 exact-release JSON과 선택 archive 획득은 기존 isolated Python data-reader 역할을
   재사용한다. UV query child에 arbitrary Python/network 권한을 주거나 Intake.Resolve의
   isolated-resolution 경계를 Host HTTP fallback으로 바꾸지 않는다. 기존 pip resolver의
   공통 setup/trace/cleanup 구간만 두 실제 소비자용 private helper로 추출할 수 있다.
   기존 parser/solver·policy 의미·context/cleanup 순서와 오류는 유지하고 pip/PyTorch
   정상/실패·취소·cleanup 회귀를 검사한다. 전체 resolver 엔진 재작성은 하지 않는다.
6. 원본 archive는 private bounded 임시 저장소의 stable ReaderAt으로 BindWheels에 전달한다.
   모든 URL/size/SHA/RECORD/Requires-Dist와 독립 release metadata를 검증해 기존 단일-root
   graph를 만든다. Raw archive/temp install은 production target으로 반입하지 않는다.
   임시 자료는 마지막 reader 종료 뒤 제거한다. cleanup 불확실/trace incomplete이면
   graph/result를 비워 원래 오류와 함께 반환한다.
7. 반환 runtime/control은 solver UV와 실제 반출 installer identity를 정확히 구분한다.
   기존 pip report를 UV 결과로 위장하거나 broad identity prefix를 허용하지 않는다.
   Promotion control adapter·upgrade retention·다중-root/empty sync·CLI는 다음 acceptance
   gap이며 이 단계에서 placeholder나 지원 문구를 추가하지 않는다.

## native 형식 조사와 검증

먼저 동일 pinned Python/UV bytes를 owned read-only/runc/network-none/UID1000 fixture에서
사용한다. 자체 inert wheel은 데이터로만 전달하며 import·entry point 실행은 하지 않는다.
Compile stdin→pylock→native materialization을 확인하고 이 성공을 public registry/gVisor/
Policy qualification으로 확대하지 않는다. Container/volume/derived image는 owner 확인 뒤
no-prune로 제거한다. 이번 runc 형식 조사의 실제 snapshot은 Host file/image/container/bridge이며
service 증거로 확대하지 않는다. Production native session은 기존 service 불변 확인도
유지한다. 완료 data는 삭제한다.

구현 검증은 canonical format/Quick/security/docs/workflow, actual public 작은 wheel/복수
의존성 graph와 고정 replay·metadata/URL/hash/size/RECORD 불일치·unknown format·stream
과다/중단·parent/trace 실패·취소/cleanup을 연결한다. Shared lifecycle 변경 시 기존 pip/
PyTorch 및 필요 CPU/CUDA 소비자를 누락하지 않는다. 한도/timeout/필수 coverage를 늘리거나
complete trace/Policy를 작은 synthetic 성공으로 대체하지 않는다.

## 두 번째 검토

기존 구현을 버리지 않고 현재 실제 두 소비자의 같은 lifecycle/data 책임만 공유한다.
Native stdin 지원은 이미 existing admission parser에 있으며 새 shell 전달면을 만들 필요가
없다. Installed metadata는 selection hint이고 전체 archive/독립 registry 검증을 생략하지
않는다. 실제 명령·snapshot 형식은 아래 조사 뒤 확정한다. 미검증 pylock 소비 방식이나
metadata collector를 구현으로 가정하지 않는다. Production CLI/target mutation·update/sync
완료를 주장하지 않으며 P2로 이동하지 않는다.

## C2 output 부분 검증

[로컬 영수증](./p1c-uv-output-local-20261022.json)은 exact native0.13.4의 stdin compile,
pylock sync/list와 named tmpfs control transport를 확인한다. 첫 guest-only /tmp의 수집
실패와 corrected volume 성공은 다른 입력으로 보존한다. 이 wheel은 inert local fixture이며
public source/parser 승인이나 gVisor/Policy 결과가 아니다. 실제 METADATA/WHEEL은 uid/gid1000,
0644이며 INSTALLER는 uv였다. INSTALLER는 판정에 사용하지 않는다.

새 uv_native_output.go는 제한된 tar에서 pylock.toml/site/만 읽고 경로·타입·중복·소유권·
모드·파일/aggregate byte·entry 한도와 producer 종료/취소를 검사한다. Host 추출·artifact
import는 없다. uv_installed_controls.go는 기존 RFC822/WHEEL grammar와 SelectInstalledWheels를
연결하고 전체 archive BindWheels 검증을 유지한다. Canonical5·관련 Race는 PASS다.
Production resolver wiring/actual gVisor output mount·독립 archive reader는 계속 MISSING이다.
