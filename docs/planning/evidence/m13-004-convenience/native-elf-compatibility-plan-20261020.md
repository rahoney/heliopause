# M13-004 — 실제 native ELF의 runtime 읽기 호환성 보완

기준 source는 a22aa12, 실제 official rc1/rc2다. 계획 문서 HEAD d76e0ca는
제품 bytes를 바꾸지 않는다. 기존 M6 observer/classifier를 수정하며 새 검사 엔진이나
본체 전용 검사 면제를 만들지 않는다. M13-004만 IN_PROGRESS다.

Read: [M6 계약](../../../domain-model/15-m6-github-releases-contract.md),
[Architecture](../../../architecture/03-inspection-sandbox-policy.md),
[도구 신뢰](../../../threat-model/04-trusted-tooling-and-evidence.md),
[자체 업데이트](../../../user-journey-cli-ia/07-self-update.md),
[실제 서명 시험](./signed-lifecycle-plan-20261020.md),
`tools/gvisor-observer/observer.cc`, `observer_latch_test.cc`,
`internal/sandbox/github_elf.go`와 pinned Go1.26.9 runtime source.

## 실제 baseline과 남은 acceptance

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 공식13파일 build/실제 서명 검증 | YES | YES | YES, rc1/rc2 | YES, producer 범위 | 없음 |
| actual first files/register/systemd/ordinary doctor/npm | YES | YES | YES, disposable booted guest | YES, 이 범위 | 공개 최초 설치·kernel reboot는 별도 |
| actual signed7 quarantine와 native 전환 | YES | YES | NO | NO | native ELF의 runtime read 분류와 이후 Root/health/crash |
| 기존 M6 일반 ELF 검사/차단 | YES | YES | YES, 기존 qualification | YES, 기존 범위 | 현재 Go/C++ actual payload의 호환성은 별도 |

실제 `self-update`는7개 authenticity/static 검사를 완료했지만 네 ELF의 dynamic
required checks가 INCOMPLETE여서 MANUAL_REVIEW/NOT_PERFORMED였다. rc1 current는
유지되고 update journal은 생기지 않았다. 첫 실패로 공유 observer가 fail-stop한 뒤의
나머지 검사 거부를 네 독립 원인으로 해석하지 않는다.

원본 owner는 `docs/github-logs/self-update-signed-20261020/`다. `guest-update-positive`,
`guest-update-first-evidence-selected`, 독립 `guest-standalone-installer`와
`guest-standalone-observer`를 구분한다. Strace6.8-0ubuntu2의 sendto 진단은 코드·검사
규칙을 바꾸지 않은 원인 조사이며 qualification 성공 증거가 아니다.

- Go CLI의 actual typed fault는 OPEN_RESULT_CLASSIFICATION_PROC,
  ARTIFACT/DIRECT_EXEC_ROOT, PROC_SELF_CGROUP다. 고정 Go runtime의
  `defaultGOMAXPROCSInit`는 자체 cgroup/mountinfo를 읽는다. GOMAXPROCS 또는
  containermaxprocs 설정만으로 startup 조회를 없앤다고 가정하지 않는다.
- C++ observer의 actual fault는 OPEN_RESULT_CLASSIFICATION_IMAGE다. 기존 FNV
  locator4231346226092351379는 정확한 `/usr/lib/x86_64-linux-gnu/libm.so.6`와 일치한다.
  Actual ELF NEEDED는 libm/libstdc++/libgcc_s/libc/ld-linux다. `IsPinnedRuntimeRootRead`의
  GitHub 분기는 현재 ld.so.cache/libc만 분류한다. 다른 library 실패를 이미 관찰했다고
  쓰지 않되 고정 ELF dependency 목록과 실제 root image를 함께 검토한다.

## 수정 지점과 방법

1. 기존 `ClassifyFilesystemOpen`에 raw ELF artifact의 좁은 runtime-read predicate를
   연결한다. GitHub profile, 실제 handoff가 끝난 ARTIFACT/DIRECT_EXEC_ROOT,
   실제 group/start/current image, root consumed와 모든 pending/network 상태를 확인한다.
   Mutable comm, package 이름, 서명 성공이나 diagnostic hash는 허가 근거가 아니다.
2. System `/proc` anchor의 실제 자기 group PID cgroup/mountinfo와, 필요한 경우 기존
   고정 CPU quota metadata만 read-only로 분류한다. 다른 PID, maps/environ, 임의 proc/sys
   prefix, write/create/truncate와 mount 오인은 거부한다. Sandbox 자기 정보이며 실제
   Host namespace·filesystem을 추가로 노출하지 않는다.
3. 고정 Node image의 OCI-root `/` anchor에서 확인된 native ABI libraries만 기존
   canonical SONAME/version matcher로 read-only 분류한다. 기존 libc/loader와
   libm/libstdc++/libgcc_s를 구분하며 arbitrary library, writable/shadow mount, user
   path는 허용하지 않는다. Artifact 역할·network/process 관찰·honeytoken 우선순위와
   ledger 한도·필수 dynamic 완료·Policy 권한은 그대로다.
4. 실제 후속 probe에서 새 첫 실패가 보이면 enum/고정 path/runtime 책임을 먼저
   기록하고 필요한 경계만 보완한다. 일반 proc/OCI root 읽기 예외나 본체만의 safe
   flag/RESOURCE 재분류·무검역 승격·compiler/runtime patch로 우회하지 않는다.

### 첫 후보의 실제 후속 확인

첫 source 후보의 canonical103 latch는 RED(새 항목만 실패)에서 GREEN으로 바뀌었다.
동일 후보의 Quick149.70s/security/docs도 통과했다. 미등록 개발 경로의 actual signed
installer/network helper/observer 세 개는 각 required3 완료/ALLOW였다. 이것은 native
SDK/Root 전환 성공을 뜻하지 않는다. CLI만 추가 첫 fault가 남았다.

`development-cli-typed-fault.json`의 PROC/OTHER locator4484167285412423496는
정확한 `/proc/sys/kernel/osrelease`와 일치한다. 기존 `displayWSL`의 optional OS
표시 언어 탐색이 이 고정 marker를 읽는다. Generic ELF의 같은 ARTIFACT/세대/완료된
handoff·read-only/system `/proc` 경계를 유지하고 이 sandbox OS marker 하나만 추가한다.
실제 Host 파일을 mount하거나 OS 언어 기능을 끄지 않는다. 인접한 kernel hostname,
vm metadata와 임의 proc/sys prefix는 계속 거부한다. 이 추가 positive가 첫 후보에서는
RED인지 확인하고 전체 latch·actual CLI/native4를 다시 확인한다. 새 path는 진단 hash가
아니라 실제 resolved fixed path와 sealed mount 사실로만 분류한다.

두 번째 대조: 새 marker는 sandbox OS의 공개 식별값이며 completion/resource/privilege
authority가 아니다. 기존 일반 Host의 Windows/macOS 언어 조회나 command grammar,
observer 네트워크 역할·한도·Policy를 바꾸지 않는다.

## 검증 순서

- Source 수정 전에 새 latch의 positive와 negative matrix를 고정한다. 기존 classifier에서
  필요한 positive가 실제 RED인지 확인한 뒤 같은 test를 수정 후보에 적용한다.
- Negative: 다른 profile/역할/image/세대, 미완료 handoff, CONTROL/network/pending,
  다른 PID/경로·쓰기 flags·틀린 mount class/point와 library lookalike. 기존 honeytoken,
  process/network·topology·budget negatives도 canonical 전체 latch에서 유지한다.
- 고정 runtime lock의 canonical observer builder/Bazel/Go/source/patch와 전체 latch를
  사용한다. Canonical docs/security/Quick와 실제 영향을 받는 소비자를 검증한다.
- 이미 실패 원인이 확보된 managed guest는 작은 원본·설정·service/current/미승격 증거를
  보존한 뒤 종료한다. 다음 개발용 guest는 native 등록을 가장하지 않는다. 기존 일반
  개발 설치 경로에서 후보 observer와 actual signed native4를 독립/전체 검사해 호환성을
  확인한다. 이 결과는 Root SDK가 새로운 서명을 검증한 native 전환의 대체 증거가 아니다.
- 수정된 실제 signed base/target의 전체 전환은 새 공식 signing inputs가 필요하다.
  기존 보호 태그는 재지정하지 않는다. 현재 승인된 두 태그의 source를 바꾸거나 공개
  Release를 발행하지 않으며, 새 태그 단계 전에는 구체 후보·검증 결과를 제시한다.
- 마지막 소비 뒤 전용 guest/volume/image·payload/build source/cache/operator를 정리한다.
  Host 기존 설치·shared runtime과 필요한 작은 public proof/Evidence는 보존한다.

두 번째 검토: 기존 M6 owner의 부족한 runtime 분류만 확장한다. Architecture/Domain의
비노출·비특권·관찰·Evidence·Policy invariant는 바꾸지 않는다. 첫 fault와 후속 fail-stop,
독립 C++ 원인, 환경 준비 오류와 제품 호환성 gap을 구분했다. 새 kernel metadata가
완료/resource authority가 되거나 검역이 생략되는 경로는 없다.
