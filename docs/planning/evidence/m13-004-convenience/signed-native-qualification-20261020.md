# M13-004 — 실제 서명 입력·native 초기 등록·ELF 호환성 검증

[실행 계획](./signed-lifecycle-plan-20261020.md),
[격리 환경](./native-systemd-fixture-plan-20261020.md),
[source 이전 보완 계획/두 번째 검토](./native-elf-compatibility-plan-20261020.md)를 따른다.
작은 [재현 receipt](./signed-native-20261020/qualification.json)와 두 실제 manifest/proof를
보존했다. 원본 owner는 `docs/github-logs/self-update-signed-20261020/`다.

## 통과한 범위

- a22aa12의 보호된 v0.1.1-rc.1/[build39402318182](https://github.com/rahoney/heliopause/actions/runs/39402318182),
  v0.1.1-rc.2/[build39405360601](https://github.com/rahoney/heliopause/actions/runs/39405360601)는
  official tag-push build/attestation SUCCESS다. 각13파일의 크기·digest·manifest/metadata와
  exact source/tag/workflow/run/subject를 확인했다. 고정 gh2.102.0의 actual UID1000,
  network namespace/offline root 검증에서 각13/13 통과했다. 임시 ZIP은 추출 뒤 삭제했다.
- Production Root SDK의 실제 rc1 파일 준비·같은 intent 재시도·ordinary 읽기/쓰기 거부와
  wrong-tag/source proof·CLI 변조 거부8 cases가 통과했다. 첫 파일 준비는 정직하게
  registration=PENDING/health=UNVERIFIED를 보고했다. 이 단계의 Host 원본 신원은 불변이다.
- 별도 실제 systemd/고정 Docker29.8.1·gVisor·네 runtime images 환경에서 rc1 파일 준비,
  service identity/cgroup, installed helper register가 통과했다. 일반 UID1000 doctor5OK와
  npm required3 완료/ALLOW도 실제 통과했다. 전체 본체 업데이트 거부 뒤에도 같은 doctor와
  nominal 검역이 다시 통과했다. Native bootstrap source 정리·current 유지·journal 부재와
  실제 서비스·자원 drain 및42개 Evidence를 보존했다.
- 실제 rc2 전체 세트의 authenticity/static7개씩은 완료됐다. 처음 native dynamic은
  미완료여서 MANUAL_REVIEW/NOT_PERFORMED였고 Root 전환은 시작하지 않았다. 원인을
  첫 Go artifact의 자기 cgroup 읽기와 독립 C++ artifact의 고정 libm 읽기로 구분했다.
  최종 후보에서는 추가 OS 표시 marker의 실제 fault도 정확히 확인해 보완했다.
- 기존 classifier에만 고정 sandbox own metadata/native ABI 분류를 추가했다. 새 positive는
  unchanged observer에서 실제 RED였고 나머지102 latch는 보존됐다. 추가 OS marker도 첫
  후보에서 별도 RED였다. 최종 고정 canonical builder/Bazel/Go의 전체103 latch는 PASS다.
  Artifact 역할·정확한 세대/handoff/mount/읽기 flags, honeytoken 우선순위와 관찰 한도는
  유지했다. 최종 Quick36.91s/security5.53s/docs0.21s도 통과했다.
- 미등록 개발용 fixture에서 후보 observer와 actual signed rc2 native4를 각각 독립 검사해
  모두 required3 완료/ALLOW를 확인했다. 기존 npm required3/ALLOW와 미등록 self-update의
  SELF_UPDATE_MANAGED_INSTALLATION_REQUIRED/exit1도 확인했다. 임시 HTTPS transport만
  fixture이며 signature root·검역·Policy·권한 검사는 면제하지 않았다.

## 해석 한계와 남은 작업

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| actual signed13 producer/독립 검증 | YES | YES | YES, a22aa12 rc1/rc2 | YES, 해당 입력 | 없음 |
| actual first files/register/systemd/doctor/npm | YES | YES | YES, disposable Linux guest | YES, 해당 mechanics | 공개 최초 획득/설치는 별도 |
| raw ELF 호환성 보완 | YES | YES | YES, local103 latch/native4/npm | YES, local 범위 | final source 원격 CI는 별도 기록 |
| 수정된 signed base→target 전체 native 활성화 | YES | YES | NO | NO | 새 official signed inputs와 Root/health/실패·복원/중단·재부팅/OS 인증 |
| M13-004 전체 설치 채널 | PARTIAL | PARTIAL | NO | NO | 위 native gap, owner 이름/publisher binding/각 채널 |

개발용 observer는 새 source의 unsigned 후보이며 native 관리 등록이 없다. 이 성공을
새 signed Root activation이나 전체 본체 업데이트 완료로 전용하지 않는다. 기존 rc1/rc2는
보완 전 observer를 포함하므로 재지정하지 않는다. 수정된 구체 후보와 원격 CI를 검토한 뒤
새 서명 시험 입력으로 이어간다. Public Release 발행·main merge·Host 영구 migration은 없다.
새 CUDA 전체·native macOS 실행이나 실제 Host kernel/WSL reboot를 이 로컬 결과로 주장하지 않는다.

## 준비 오류·정리·재검토

초기 image stream의 ref/ID 확인 실패, nested overlay EINVAL, copy2의 uid/gid83개 누락,
filesystem별 directory st_size 차이와 준비 직후 TLS readiness race는 원본에 별도로 남겼다.
내용·mode/owner·xattr·membership을 다시 확인한 원본 보존/reversible mount로 환경을 교정했고,
이를 제품 수정 성공이나 서명 오류로 합치지 않았다. 첫 doctor의 미지원 JSON 옵션과
Staticcheck cache의 read-only 권한 실패도 정상 결과로 재분류하지 않았다.

첫 managed guest는 증거 확보와 사후 정상 확인 뒤 종료·삭제했다. Stop command 성공과
PID1 exit130은 구분하며 reboot qualification으로 쓰지 않는다. 마지막 사용이 끝난 나머지
전용 guest/volume/image·payload/build/cache/operator도 정리하고 원본 cleanup receipt로
확인한다. 작은 public proof/manifest·필요 Evidence와 Host/shared runtime은 보존한다.

두 번째 검토: 서명 성공, 파일 준비, 등록/진단, native4 동적 호환성과 signed whole activation의
증거를 분리했다. 첫 fail-stop 뒤 후속 미완료를 별개 원인으로 확대하지 않았고 기존 source를
다시 구현하지 않았다. M13-004는 계속 IN_PROGRESS다.
