# P4-O4-C2 — pipx DATA readonly volume lifecycle

기준25f9a8e/C1 whole treeb62dadc, active M13-004 하나다. Read: Queue, O4/C1 map,
plan20 §5·6, client09, artifact L01/L02/L15, sandbox L03/L08, evidence L11/L16,
ci L06. 기존 selected owner/held reader/tar producer와 Python volume을 재사용한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| fixed ZIP DATA reader/transport | YES(C1) | YES(private supply) | YES(named local) | YES(C1 only) | actual Docker volume 소비 |
| volume identity/mount/attachment query | YES(closureVolume) | NO(pipx) | NO(pipx) | NO(pipx) | fixed root-owned noexec destination, shadow 거부 |
| preparation→anchor 수명 유지 | YES(Python wheel transaction) | NO(pipx) | NO(pipx) | NO(pipx) | DATA-only seed/anchor/seal wrapper |
| live readonly 전달/정리 | YES(existing closure consumers) | NO(pipx) | NO(pipx) | NO(pipx) | actual ZIP·writer 제거·RO reader·빈 attachment cleanup |

`pypi_closure_volume.go`의 고정 project destination 집합에 `/haa-pipx`만 추가한다.
그 경로는 selected ZIP digest/8MiB/noexec/root0:0/mode0755를 강제한다. 기존 Python/Go/
Cargo/PNPM/Bun의 options는 유지한다. Pipx 경로에만 ancestor/nested mount shadow와
같은 volume의 다른 목적지 alias도 거부한다. Label의 manifest hash는 DATA identity일 뿐
wheel plan/Artifact admission/Policy 완료가 아니다. 생성 결과가 불명확하면 pipx는 handle과
오류를 반환하고 삭제하지 않는다. 정리는 기존 exact identity/성공한 빈 attachment query를
사용하며 unknown query를 부재로 처리하지 않는다.

새 private `pipxDataVolume`은 고정 RW preparation과 RO reader mount를 구분한다.
Seed는 유일한 live writer와 C1 공급/readback을 요구한다. 부분 전송 뒤 retry는 거부한다.
Anchor는 writer가 살아 있을 때 RO로 붙이고 같은 ZIP bytes를 확인한다. Seal은 writer가
제거되고 정확한 live RO anchor만 남은 뒤 가능하다. Consumer 검사는 sealed 상태와
anchor/consumer의 exact RO inventory·alive·bytes를 다시 요구한다. 생성/stop/remove와
observer 권한은 기존 trusted controller 책임이며 이 wrapper는 실행 완료 권한을 만들지 않는다.

검증은 default topology/unknown query/foreign writer/잘못된 순서·shadow/정리 거부와 기존
volume 소비자 회귀, canonical Quick/security/docs와 sandbox 전체 Race를 포함한다.
고정 Python image/runc의 실제 DATA 시험은 network none/cap-drop ALL/read-only root와
constant sleep keeper만 실행한다. 실제 ZIP을 RW preparation에 전송하고 RO anchor가
유지한 채 writer 제거→seal→RO consumer readback·쓰기 거부→owned cleanup을 확인한다.
Python/pipx/Artifact 코드는 실행하지 않으며 gVisor/observer·전체 pipx 지원 성공으로 확대하지 않는다.

두 번째 검토: 일반 native RW volume은 계속 pipx를 거부한다. 고정 destination의 특별
options를 caller가 임의 hash/capacity로 만들 수 없게 검사한다. 기존 실행 owner를 복제하거나
fake wheel transaction을 만들지 않는다. B release execution inspection과 C 실제 gVisor/native
install·upgrade/CLI는 계속 MISSING이다. 공통 volume 경계의 로컬 검증 뒤 grouped CI를 판단한다.

첫 actual2.481s는 전송 readback 거부이며 진단 actual2.321s의 실제 tar가 ZIP2505814B/
0444이되 uid/gid1000:1000임을 확인했다. 두 실행의 보호5파일/Docker·원본 ZIP은 같고
owned 자원은 정리됐다. C1 root0:0 guard는 보존한다. 기존 Python preparation은 root
keeper로 공급하고 Artifact 실행 때 별도 uid를 사용하는 owner다. DATA 시험의 writer만
고정 root sleep으로 맞추며 anchor/reader는1000:1000을 유지한다. Python/ZIP 코드는
실행하지 않는다. 이는 실제 공급 검증이며 gVisor 실행 권한 확대가 아니다.
