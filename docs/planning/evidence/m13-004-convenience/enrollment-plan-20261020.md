# M13-004 — 이미 검증·배치한 native 설치의 명시적 등록 경계

기준 `5ac400b`와 [producer 연결 map](./producer-plan-20261020.md).
등록은 최초 bootstrap의 일부이며 전체 clean Host bootstrap/공개 배포의 성공이 아니다.
현재 Root Role의 serve/recover 및 기존 copy/read/SDK/lease owners를 재사용한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| fixed signed native reader/Root transaction/runtime guards | YES | YES | PARTIAL, local | NO, actual signed transition | 재사용 |
| root registration publisher | YES, update swap/restore만 | YES, update | YES, local | YES, component | 최초 생성 경로 |
| 명시적인 initial native enrollment | NO | NO | NO | NO | 아래 고정 register operation |
| 최초 acquisition/copy/runtime setup·clean Host doctor | PARTIAL, 기존 owner | NO, public loader | NO | NO | 별도 bootstrap/실제 source qualification |

## 조건과 변경 위치

- `releaseactivation/enrollment_linux.go`의 실제 `Register`는 Linux/amd64 root UID/GID0만
  허용한다. Privileged helper의 fixed `register` operation으로 연결한다. Root-only이며
  업데이트 wire에 operation/UID/path/service 필드를 추가하지 않는다. Ambient SUDO_UID/
  SUDO_USER는 사용하지 않고 실제 entry는 기존처럼 caller environment를 비운다.
- 이 operation은 이미 fixed `/usr/lib/heliopause/versions/<version>/`에 **독립적으로
  검증한 최초 installer가 배치한** 완전한 native4+metadata3, root-protected current
  symlink와 stored public proof를 요구한다. 다운로드/copy/서비스 시작/설정 전환을 하지 않는다.
  Root bootstrap helper를 실행하기 전의 independent provenance verification도 필요하다.
  새 helper가 스스로 안전하다고 선언했다는 이유로 먼저 root 실행하지 않는다.
- UID/GID는 root-owned fixed network config에서만 읽는다. Current pointer/manifest와
  same exact native CLI/observer/unit config를 묶고 공유 recorded reader가 SDK/모든 byte를
  다시 검증한다. 실제 own executable도 그 signed installed installer inode여야 한다.
  OS package `/releases/`·개발용 `/usr/libexec/`는 자동 인수하지 않는다.
- Installer owner를 잡고 persistent blank runtime/update locks를 root0444로 준비한다.
  기존 파일은 root0/single-link/readonly/empty/exact metadata만 허용한다. Runtime EX와
  실제 서비스 PID/socket/cgroup·pinned Docker/runsc/images·전체 observation/consumer drain을
  확인한 뒤 root0644 registration을 atomic publish한다. Unknown은 absence가 아니다.
- Existing registration은 정확한 동일 record의 재검증만 허용한다. 다른 version/hash/UID/GID를
  register로 바꿀 수 없다. Registration이 없는데 update journal/history가 있으면 partial
  기존 설치로 취급하고 등록하지 않는다. Pending journal을 이 operation으로 해제하지 않는다.
  초기 blank lock 생성 중 중단된 경우 같은 authoritative native set/config로 재시도할 수 있다.
- `config_linux.go`의 기존 native config parsing과 `journal_linux.go`의 saved-config 검증만
  private helper로 공유한다. Invalid fake journal을 만들어 parse validation을 우회하지 않는다.
  기존 실제 journal validation/state machine은 유지한다.
- Root public directory0755/native0555/metadata0444와 ordinary UID가 읽는 config의
  read permission을 확인한다. Root-only runtime query 성공을 ordinary-user doctor나
  전체 bootstrap 성공으로 보고하지 않는다. 최초 setup의 마지막 acceptance는 등록 뒤
  **설정된 일반 사용자로 실제 `helox doctor`와 nominal quarantine**를 확인하는 것이다.

## 검사·재검토와 이후 순서

Canonical Quick/security/docs/vulnerability와 관련 Race를 먼저 실행한다. Actual SDK가
미서명 native9를 거부하는 negative와 disposable namespace의 root file/lock/registration
mechanics를 구분한다. Wrong layout/UID/current/manifest/proof/helper inode/config/modes,
foreign registration/journal/history, nonempty/link/foreign locks, service/drain failure 및
정확한 동일 record의 repeated register를 확인한다. Mocked crypto/service 결과를 actual
native bootstrap으로 확대하지 않는다. 완료한 test binary/payload는 사용 직후 삭제한다.

두 번째 대조에서 initial registration을 updater의 old-peer 인증 예외로 넣지 않았다.
Serve의 old protected controller/검역/SDK gate는 유지한다. Register는 기존 활성 세트를
교체하지 않고 Root admin이 고정 경로에 검증해 배치한 동일 세트의 identity만 등록한다.
Producer/등록 source 이후에도 bounded verified initial loader, 실제 새 signed native9
base/target/systemd와 crash/reboot qualification은 남는다. Public tag/publish/main merge와
host 영구 migration은 source 구현만으로 실행하지 않는다.

## 구현 후 두 번째 대조의 보완

기존 registration이 있는데 persistent runtime/update lock이 없으면 새 lock을 만들어
복구한 것으로 처리하지 않는다. 등록 전 blank lock 준비 중 중단된 최초 설치만 재시도한다.
또한 공개 metadata 검증만으로는 잠근 FD와 현재 pathname의 inode가 같다고 입증할 수
없으므로 실제 보유 lock FD와 최종 pathname을 SameFile로 연결한다. 두 조건은 먼저
기존 후보에서 namespace 회귀로 반증하고 보완한다. 첫 Quick의 SA1012는 새 테스트가
API에 nil Context를 전달한 것이며, nil 사용 시험을 제거하고 취소 Context 검사를 유지한다.
