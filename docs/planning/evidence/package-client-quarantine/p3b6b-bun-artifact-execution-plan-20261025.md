# P3-B6b 명시 Bun Artifact 실행·관찰 경계 감사

M13-004 하나만 IN_PROGRESS다. Baseline 74fd3668ab77c30e3600ca223cf21918625c2ed6/treee679f0f4d559ccb79f2c1a7bc77bfb810493c3d2/2470files.
Read: B6a 공급 감사·로컬 원본, B4d native lifecycle source 의미와 target admission,
client 설계§4/확장 계획§5, observer ProcessClassForPath/ParseSentryProcessAndClassify/
IsPinnedBunRuntimeRead/ClassifyFilesystemOpen, C++ latch 실제 production lineage 검증,
Bun runtime identity·ToolImage·Backend/SharedObserver, dynamic finding/M3 Policy와 관련 lessons.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| native runtime 고정 공급/새 session·전체 source-bound payload | YES | YES(explicit backend) | YES(B6a local) | YES(transport) | 실행 의미와 구분 |
| Node/npm exact lifecycle command·untrusted script attribution | YES | YES | YES(local/CI412) | YES(existing lane) | 별도 native Bun PATH mode의 exact command/topology |
| Artifact가 고정 Bun으로 JS를 실행하는 관찰/ABI reads | NO | NO | NO | NO | 실제 synthetic source fixture의 Bun-global assertion·false violation 재현과 좁은 read binding |
| 전체 inspector의 runtime 요구·fresh coverage/Policy/hosted | NO | NO | NO | NO | 이 실행 boundary 뒤 이어지는 whole wiring |

Bun은 resolver에서 CONTROL일 수 있지만 Artifact lifecycle에서 호출된 image는 계속 ARTIFACT다.
현재 M3는 process-exec-unexpected를 MANUAL_REVIEW, FS/honeytoken을 BLOCK으로 분류한다.
새 runtime을 이유로 이 관찰을 삭제하거나 ALLOW로 바꾸지 않는다. 공급/완료/JS fixture의
성공과 Policy 승인을 구분한다. 기존 native resolver/query reads를 Artifact에 그대로 열지 않는다.

## 수정 지도·두 번째 검토

1. 먼저 `bun_artifact_execution_real_test.go`에서 기존 구조 fixture와 실제 ArchiveInventory/
   bound layout를 재사용해 known lifecycle가 /haa-runtime/bun으로 JS를 실행하도록 한다.
   JS는 Bun global/version을 요구해 Node 대체를 거부한다. 실제 controller 종료·완전 관찰,
   ARTIFACT process observations·cleanup을 확인하며 첫 미지원 read/trace 실패를 보존한다.
   구조 source/SRI fixture를 public authentication 증거로 쓰지 않는다.
2. 필요한 좁은 owner만 수정한다. `bun_artifact_runtime.go`/instance owner는 native option에만
   fixed PATH command를 선택한다. 기존 Node command와 resources/budgets는 유지한다.
   `observer.cc` exact command admission은 host-attested native9p mount와 고정 target를 요구한다.
   Bun image는 kernel-resolved path+sealed topology로만 분류하며 Artifact role/lineage·네트워크
   귀속을 유지한다. 제어 root/one-shot/expected groups에 새 권한을 주지 않는다.
3. Artifact Bun의 읽기는 exact native image·실제 kernel process identity/ARTIFACT role·해당
   topology·정확한 mount/path/flags와 연결한다. 이미 관측된 fixed GNU ABI/self startup 또는
   새 실제 실패로 확인한 최소 입력만 검토한다. /proc·/sys·/etc·library tree 전체를 열지 않고
   다른 UID/group/process/profile·writes·shadow/foreign paths·stale exec에는 적용하지 않는다.
4. C++ latches로 image/role/creator/generation/root flags/mount/path/open flags/command
   negatives와 unchanged Node/PNPM·query boundary를 검증한다. Canonical pinned release
   compiler/latches는 Docker actual Host snapshot과 직렬 실행한다. Actual 정상·실패 JS와
   별도 native Node/PNPM consumer·canonical profiles/whole sandbox Race를 연결한다.

Bun native lifecycle trust 목록은 검사 완료 authority가 아니다. Whole inspector·fresh coverage/
Policy와 최종 hosted/CPU·세CUDA 영향 qualification은 다음 acceptance로 남긴다. 원격 CI는
이 의미 있는 runtime/end-to-end 묶음에서 한 번 실행하고 실제 elapsed 근거 timer를 소비한다.

## 첫 actual/진단

CPP119를 유지한 v1 actual12.161s와 test-only 기존 fault readout의 v2 actual11.868s에서
고정 native JS의 Bun1.4.4 body는 실행됐다. 두 ARTIFACT 관찰은 유지됐으나 observer는
STREAM_FAULT/OPEN_RESULT_CLASSIFICATION_IMAGE로 INCOMPLETE다. Exact OCI-root read
flags557056/locatorfbc430c15ef57dff는 /etc/ld.so.cache, 실행 locator7b96a1ffb2d233eb는
/haa-runtime/bun에 대응하며 image UNKNOWN/role ARTIFACT/provenance CLONE_CHILD다.
이는 첫 loader read의 미분류 원인만 확정한다. 다른 startup 입력/완전한 관찰의 해결은
미검증이며 권한을 무시하지 않는다. 두 actual의 helperwait0/Host 전체 동일을 확인했다.
원본과 source는 p3b6b-first-assessment/first-source 및 v1/v2 raw가 소유한다.

121 latches·canonical build268.237s와 첫 좁은 image/read 경계를 적용한 v3 actual12.323s는
같은 JS body 실행 뒤 다음 /proc read에서 INCOMPLETE다. Exact flags32768·bounded locator
7890d21290e5fd8d는937개 read-only proc path-name catalogue의 /proc/sys/vm/mmap_min_addr에
대응한다. File contents/diagnostic hash는 authority가 아니다. 현재 pinned Bun ARTIFACT
image/lineage guard 아래 이 exact read만 추가하며 generic proc/writes·CONTROL 예외는
주지 않는다. 두 ARTIFACT 관찰·helperwait0/Host 동일을 유지했다. V3 next-read assessment와
원본이 근거이며 전체 해결은 여전히 미검증이다.

Canonical v2 build252.137s/121 latches 뒤 v4 actual12.564s는 mmap_min_addr 다음의 exact
OCI-root / directory open flags622592·locatoraf63a24c860189fe에서 INCOMPLETE다. 고정
Bun ARTIFACT의 directory probe 하나만 검토하며 파일 내용/임의 directory·write 예외를
추가하지 않는다. JS body/두 ARTIFACT 관찰·helperwait0/Host 동일은 유지됐다. 이후 반복은
기존 frozen warm SDK의 DERIVED_LOCAL_DIAGNOSTIC으로 구분하고 마지막 unmodified
canonical build/actual은 별도로 요구한다. V4 next-read assessment가 원본을 연결한다.

Warm SDK diagnostic38.959s/121 latches 뒤 v5 actual12.622s는 fixed memory cgroup의
memory.soft_limit_in_bytes·557056에서 다음 read가 확인됐다. Exact system mount
/sys/fs/cgroup/memory와 locator 대응을 확인했다. 기존 canonical runtime의 고정 flat
soft/hard 숫자 limit 쌍만 같은 Bun ARTIFACT/lineage guard에 연결한다. Soft는 직접 관측,
hard counterpart는 기존 bounded runtime 입력으로 구분하며 따로 관측했다고 쓰지 않는다.
임의 cgroup 자식/쓰기·budget 확대는 없다. V5 next-read assessment가 범위를 소유한다.

Warm memory-limit diagnostic37.425s/121 latches 뒤 v6 actual11.734s는 own process
/proc/40/statm·32768(PROC_SELF_STATM)에서 다음 거부가 확인됐다. Exact current kernel
thread-group-id의 statm 읽기만 같은 /proc system mount·Bun Artifact/lineage에 연결한다.
다른 PID/임의 proc·writes는 허용하지 않는다. JS body/Artifact2·helperwait0/Host 동일은
유지했고 전체 해결은 아직 미검증이다. V6 next-read assessment가 원본을 연결한다.

Warm own-statm diagnostic38.100s/121 latches 뒤 v7 actual18.929s에서 정상 JS가 완전 관찰/
COMPLETED, exit37 JS는 기존 cause/INCOMPLETE로 완료됐다. 두 경우 ARTIFACT process2를
보존하고 FS/honeytoken/network 추가 사실0·helperwait0/Host 전체 동일이다. 이는 diagnostic
소비자 성공이며 최종 canonical 결과로 확대하지 않는다. 최종 C++ latches에는 foreign PID/
self-alias/foreign cgroup/config 파일 거부를 추가하고 actual 실패 test는 typed exit37도 요구한다.

## 최종 named local qualification

[로컬 영수증](./p3b6b-bun-artifact-execution-local-20261025.json)은 unmodified canonical
build242.161s/전체121 latches와 v8 actual18.791s의 정상 JS·typed exit37를 연결한다. 정상은
완전 관찰/COMPLETED, 실패는 원래 exit37/INCOMPLETE이며 각각 ARTIFACT process2·baseline
FS/honeytoken/network0를 보존했다. 같은 observer의 Bun legacy Node28.73s/native supply39.09s/
PNPM full project46.18s 회귀 operator116.551s도 PASS다. 각 helperwait0/Host 전체 동일을
확인했다. 기존 default Go의 Quick125.115s/sandbox Race31.180s와 최종 tagged vet1.220s/
compile·security/docs, 나중 CPP read/negative 변경의 최종 canonical/actual 범위를 구분했다.

최초 각 실패의 source/원본을 보존했고 warm diagnostic은 canonical 성공으로 전용하지 않았다.
CONTROL/one-shot/네트워크·Policy/예산/버전은 바뀌지 않았다. 명시 backend 실행 경계만
닫혔으며 whole inspector의 native capability 연결과 recorded Evidence/Policy·hosted/전체
P3 qualification은 B6c 이후로 남는다.
