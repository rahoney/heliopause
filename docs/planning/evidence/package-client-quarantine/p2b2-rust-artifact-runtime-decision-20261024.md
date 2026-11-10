# P2b2 — Rust native ARTIFACT의 자기 메타데이터 읽기 변경안

상태: **APPROVED — 2026-10-24 사용자 “승인해.”**. Exact-self maps는 실제 관측 뒤 적용하며 overcommit은 해당 ARTIFACT 관측 뒤 적용한다. M13-004/P2b2는 IN_PROGRESS다.
이 문서는 [guest admission 지도](./p2b2-pnpm-admission-plan-20261024.md)의 독립
ARTIFACT lane에서 발견한 명시적 계약 차이를 검토한다. 일반 PNPM CONTROL startup과
빈 lock 생성 성공은 이 ARTIFACT 거부를 대체하지 않는다.

## 확인된 현재 상태

- 보완11의 C++108 latch33.777s PASS 뒤 actual CONTROL startup3.77s와 offline
  empty lock3.82s/lock9는 PASS·완전한 관찰·unexpected/outside/honeytoken/network0이다.
  이 빈 입력의 성공은 Node child query 실행을 입증하지 않는다.
- 같은 PNPM bytes를 기존 GitHub ELF backend의 `/work/artifact`로 실행한 실제5.19s는
  ARTIFACT/DIRECT_EXEC_ROOT/PROC_SELF_MAPS, flags557056/system `/proc`에서 STREAM_FAULT다.
  required dynamic 결과는 INCOMPLETE이며 ALLOW·승격·배포를 수행하지 않았다.
- 원본 `p2b2-pnpm-startup-native-marker-abi`가 두 성공/독립 실패를 소유한다.
  잔여 container/volume/observer endpoint0·private daemon wait0·보호12files/images/
  bridge/service 동일과 Host native installation 부재를 확인했다.
- 실제 ARTIFACT overcommit read는 아직 NOT_REACHED다. 같은 native bytes의 CONTROL
  startup에서는 `/proc/sys/vm/overcommit_memory` flags557056 읽기를 실제 확인했다.

[M6 계약 §3](../../../domain-model/15-m6-github-releases-contract.md#3-format-static-inspection-and-dynamic-capability)
승인 전 문장은 `Other PIDs, maps/environ ... gain no permission`이었다. 기존
[native ELF 계획](../m13-004-convenience/native-elf-compatibility-plan-20261020.md)도
maps/vm 부정을 명시한다. AGENTS의 기존 Domain·Security invariant를 구현 편의로
변경하지 않는 규칙 때문에 이를 조용히 정상 runtime read로 바꾸지 않는다.

## 권장안과 적용 범위

사용자가 아래 두 exact 읽기의 명시적 계약 변경을 승인했다. 관측/조건/나머지 거부 경계는 그대로 적용한다.

| 입력 | 허용하려는 조건 | 계속 거부할 조건 |
| --- | --- | --- |
| `/proc/<실제 group PID>/maps` | `github-elf`의 실제 handoff 완료·같은 generation의 ARTIFACT/DIRECT_EXEC_ROOT, kernel current image ARTIFACT, system `/proc`, flags557056 | 다른 PID·self alias·environ/mem·쓰기/추가 flags·clone·CONTROL·pending/stale/handoff 미완료·다른 profile/mount |
| `/proc/sys/vm/overcommit_memory` | 같은 guard와 flags; 실제 해당 ARTIFACT first read를 확인한 뒤에만 적용 | 다른 vm/kernel 정보·prefix·alias/쓰기·다른 role/profile/mount |

이는 **모든 GitHub ELF 직접 검사에 적용되는 좁은 guest runtime-read 규칙**이다.
PNPM package 이름·서명·digest만 보고 특정 파일에 검사 면제를 주지 않는다. 메모리 맵은
검사 대상 자신의 격리된 guest 주소 공간 정보이며 Host 프로세스/namespace·파일·자격증명은
새로 노출하지 않는다. 그래도 기존 금지에서 허용으로 바뀌는 실제 동작이므로 승인이 필요하다.

ARTIFACT 역할·정규화된 runtime-access count·네트워크/예상 밖 exec/쓰기/honeytoken 관찰,
기존 resource/ledger·완전성·required dynamic/Policy는 유지한다. 이 두 읽기는 clean verdict,
완료 authority, CONTROL/network 권한을 만들지 않는다.

## 정확한 변경 위치와 방법

1. 승인 후 M6 계약의 해당 문장을 exact-self maps/관측된 overcommit 예외와 나머지
   거부 조건으로 갱신한다. 과거 raw/서명 fixture나 과거 자격 검증 결과는 수정하지 않는다.
2. `observer.cc/IsGitHubArtifactRuntimeRead`의 기존 system `/proc` 분기에
   exact group PID maps/flags557056만 추가한다. overcommit은 그 실제 ARTIFACT fault를
   확인한 뒤 같은 exact flags/path로만 추가한다. 공용 Python/CONTROL predicate나
   `IsExactPinnedLibraryRead`, 경로 prefix, nonce/완료 프로토콜은 변경하지 않는다.
3. `observer_latch_test.cc/VerifyGitHubArtifactRuntimeReadBoundary`에서 현재 maps
   부정 하나를 좁은 정상 사례로 대체하고, 다른 PID/alias/environ/mem·잘못된 flags·
   stale/pending/clone/CONTROL/network·profile/mount의 부정 matrix를 강화한다.
   현재 C++108개 기준과 전체 소비자의 실패 원본을 보존한다.
4. actual PNPM ARTIFACT probe를 다시 수행해 관찰 완전성·suspicious0·실제 exit0과
   cleanup/Host 불변을 확인한다. 새 first fault는 별도 MISSING이며 자동 허용하지 않는다.
5. 동일 최종 후보의 기존 GitHub native/차단 회귀와 npm/UV·공통 소비자, 관련 전체
   Race/canonical checks 및 묶은 원격 CI로 범위를 검증한다. 과거 green을 새 권한의
   검증으로 전용하지 않는다. P2c public graph/CLI·새 signed release는 별도다.

예상하는 predicate 차이는 기존 role guard 아래 다음 두 bounded 비교뿐이다.

```cpp
// Approved; exact self maps observed at native-marker-abi.
if (flags == 557056 && path == self + "/maps") return true;
// Apply the following only after its actual ARTIFACT first fault is observed.
if (flags == 557056 && path == "/proc/sys/vm/overcommit_memory") return true;
```

두 번째 검토: Host memory/전체 proc 공개나 PNPM CONTROL 예외 승계가 아니다. 반면 기존
명시적 maps 거부를 바꾸므로 단순 fixture 수정으로 처리하지 않는다. 승인이 없으면 현재
ARTIFACT 거부를 유지하고 해당 지원은 미완료로 남긴다. 독립적인 CONTROL Node-query와
기존 승인 범위의 회귀·정리는 계속할 수 있다. 대안은 기존 규칙을 유지하고 native PNPM의
검역·배포를 보류하는 것이다. 검사를 생략하거나 다른 client/version으로 몰래 대체하지 않는다.

## 승인 후 실제 관측과 적용

`p2b2-pnpm-startup-artifact-terminal-origin`은 no-argument PNPM의 Usage/exit2를
확인했다. 자기 native 전용 고정 version probe 연결 뒤
`p2b2-pnpm-startup-artifact-fixed-probe`5.59s에서 ARTIFACT/DIRECT_EXEC_ROOT/
VM_OVERCOMMIT_MEMORY·flags557056/system `/proc`를 실제 관측했다. 승인된 두 번째
읽기의 관측 조건이 충족됐으므로 보완17에서 같은 guard에 exact 경로 하나만 연결한다.
이후 실제 성공과 common 소비자 qualification은 아직 미완료다. 앞의 NOT_REACHED는 당시 이력이다.
