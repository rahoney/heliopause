# M13-004 — 기존 installer의 protected copy·verified rollback 확장

기준 parent `28b88b2`. [privileged source 전 감사·권한/복구 계획·두 번째 검토](./protected-activation-plan-20261020.md)를
따라 기존 Installer.Install/activate/atomic pointer를 재사용했다. 새 설치 엔진으로
교체하지 않았다. 전체 M13-004와 installer IPC/service/journal/public CLI는 미완료다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| fixed Linux amd64/root copy mode와 네 번째 역할 | YES | YES, shared installer/test | YES, local kernel mechanics | YES, 이 단위만 |
| 기존 version membership/bytes 재검증 후 pointer 복원 | YES | YES, shared mandatory verifier/pointer | YES, local user/root mechanics | YES, 이 단위만 |
| root RPC/old caller 인증·runtime lease·journal/service 전환 | NO | NO | NO | NO |
| 실제 새 signed target·public command·다음 실제 검역 | YES, 기존 검사 backend 재사용 | NO | NO | NO |

`NewSystemInstaller`는 Linux amd64/root와 고정 system/staging roots만 허용한다.
Input ancestors/owner/mode·regular file/link/set-ID·metadata bounds/schema와 네 native
roles를 확인한다. Root-readable native0555/metadata0444/directory0755를 사용하고
기존 user0700/0600 model은 유지한다. Mandatory BundleVerifier는 생략하지 않는다.
SDK authority·IPC·service/lease/journal은 이 copy primitive가 부여하지 않는다.

`ActivateExisting`도 verifier와 실제 versions membership·version/bytes를 다시 확인한
뒤 같은 pointer primitive를 사용한다. Bare pointer-switch API는 노출하지 않았다.
Context cancellation과 verifier의 원래 cause를 보존하며 copy/commit 경계도 취소를
확인한다. 전체 service transaction의 취소/복구는 다음 lifecycle owner 책임이다.

원본은 local `docs/github-logs/self-update-20261020/`가 소유한다.

- `protected-copy-final-source.json`/`protected-copy-final-inventory.json`: 최종 입력과
  전체 source bytes. `protected-copy-final-checks.json`의 focused0.92s,
  canonical Quick103.96s/security6.70s/vulnerability11.08s/docs0.23s,
  관련 전체 Race10.74s는 모두 exit0다.
- `protected-copy-final-namespace.json`/원본 log: exact final binary를 정적으로
  빌드하고 새 mount/network namespace의 tmpfs에서 실제 root로0.07s/exit0.
  Root-owned/Nlink1/modes, dropped UID1000의 read 성공/write·installer 진입 거부,
  기존 version rollback, foreign owner/hardlink/symlink/writable ancestor 거부를
  확인했다. 두 host 쓰기 경로는 전후 ABSENT_CONFIRMED이며 host activation은0이다.
  **Verifier는 synthetic fixture용 mock**이다. 새 signed release/전체 runtime
  교체/실제 sandbox/peer authentication 성공으로 전용하지 않는다.
- 위 최종 Linux test binary5,387,338B와 앞선 namespace binary5,385,746B는 각각
  실행 직후 삭제했다. Root fixture는 private tmpfs namespace 종료와 함께 소멸했다.
- `protected-copy-old-writer-red.json`/log: current source에서 writeNew 한 함수만
  이전 user-only writer로 되돌린 overlay는 실제 umask077의 system mode 회귀에서
  exit1/0.69s다. 새 mode 요구를 기존 user0700 구현의 shipped defect라고 부르지
  않는다. File.Chmod로 명시한 최종 modes와 위 실제 root/ordinary-user positive가
  해당 gap을 검증한다. 완료 overlay/operator 입력도 삭제했다.
- `protected-copy-portable-checks.json`: Darwin amd64/arm64 test compile only.
  Linux-only system mode의 macOS 실행 지원을 뜻하지 않는다. 출력은 즉시 삭제했다.
- 문서 전달·index/commit proof는 같은 디렉터리의 protected-copy-delivery 자료를 따른다.

두 번째 대조에서 system copy의 고정 root·root-owned input·role membership/bytes
검증과 signature 선행, 기존 pointer rollback/sync 소비자를 유지함을 확인했다. Core/Policy/일반 artifact
명령은 변경하지 않았다. 다음은 실제 설치 전용 helper와 per-request session, old runtime
cleanup/lease handoff, journal 및 고정 service 전환의 구현·fault qualification이다.
