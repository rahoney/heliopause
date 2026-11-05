# WSL 중단 복구·Node 전환·저장공간 정리 — 2026-11-05

## 중단 지점과 보존 범위

기준은 `milestone/m13-release-build-packages`의 `5fd908b4cd459b3478c1dde97f5d26d46e769ea6`이다.
추적 source 변경은 없고 중단 상태를 기록한 기존 문서 두 개만 미커밋이었다.
`git fsck --full --no-reflogs --no-dangling`은 exit0/출력 없음이다. Git 객체 손상은 발견하지 않았다.

`6e74f22`의 Bash·실제 Go 성공과 `5fd908b`의 npm 시험 기대값 교정은 보존돼 있다.
Updater user-v3의 plan/apply는31.079s/exit0, candidate는 당시 전체 tracked Root와 같았다.
Verify는 bootstrap 중 ext4 `emergency_ro`/read-only 뒤 exit135로 끝났으며 **미완료**다.
제품 실패나 검증 통과로 재분류하지 않는다. 상세 원본은
`docs/github-logs/goal-implementation-20261101/host-emergency-ro-20261105.log`와
같은 원본 폴더의 `s04-bash-owner-user-v3.json`이다.

이번 재개에서는 ext4 rw/`emergency_ro` 없음, 기본 Python/Go/Node 실행과1MiB write/fsync/readback을 확인했다.
재부팅 후 journal에는 unclean shutdown/회전 기록이 있다. Windows Volsnap24/35의 C: 공간 부족도
확인했지만 ext4 하위 원인과 직접 인과는 **UNKNOWN**이다. 이전11월4일의 별도 container OOM을
11월5일 장애 원인으로 전용하지 않는다. Mounted filesystem의 읽기/쓰기 성공은 offline fsck 인증이 아니다.

## 재개 전 입력 복구

재부팅으로 `/tmp/haa-goal-implementation-20261101` 전체가 사라졌다. 원래 private source,
compiled builder, Semgrep core/source/notice 공급물, SDK 입력과 updater의 frozen state/candidate는
현재 사용할 수 없다. 기존 plan ID는 `a51c90caebe3f8b54bff7676fa6fa09db27ebe9e521eb8e393349ea84bbdbdb8`이며
stdout/실패 로그만으로 state bytes가 복구됐다고 주장하지 않는다.

1. INDEX·관련 오답노트·실제 Git과 이 문서를 읽고 F:의 새 checkpoint와 압축 결과를 확인한다.
2. 현재 source/고정 owner와 보존된 원본을 이용해 **사라진 입력만** 재구성한다. 이전10월30일
   F: checkpoint는 현재 공급물의 대체물이 아니다. Hash가 다른 옛 binary를 현재 승인물로 쓰지 않는다.
3. `scripts/runtimes.lock.json`의 Semgrep1.183.0/source commit·manifest·SDK/recipe와 기존
   `scripts/update-runtime`의 source/notice/build/materialize/plan/apply/verify owner를 재사용한다.
   공급 URL/자동 획득·release 연결은 원래도 MISSING이며 이번에 새로 구현하지 않았다.
4. 아래 정확한 공급 hash와 frozen 입력을 확인한 뒤 개발 UID1000의 기존 updater 경로로
   interrupted verify를 새 state에서 재구성한다. 이전 성공을 반복한 qualification으로 세지 않는다.
   동일 Bash/Go native·CPP138/core 검증을 이유 없이 재실행하지 않는다.

| 공급물 | bytes | SHA256 |
| --- | ---: | --- |
| semgrep-core-linux-amd64 | 171651104 | `33ae45003cb97dfc253e5db9076f9f19d78b51f65a0d23f7e188d972ad731229` |
| semgrep-SOURCES.tar | 328025088 | `d8b45712778330c21b9b4733e56b8c726cec76b0d1f7baf008b221d29a045536` |
| semgrep-NOTICES.tar | 1982464 | `066469f23a616cf245f98f156be59dcfb88f5f6f043bb4fd0fe4815fc0806f42` |

현재 재개 준비는 source/기존 실행 결과를 보존하고 입력 재구성부터 시작할 수 있다는 뜻이다.
즉시 verify 재실행 가능·전체 S04/보강/hosted acceptance 완료라는 뜻은 아니다. 새 CI/타이머는 없다.

## 사용자 Node와 helox 확인

사용자 nvm Node25.8.1은 공식 지원 종료 상태이며26.13.0 LTS를 공식 배포 체크섬 확인 후 병행 설치했다.
기본 alias는26.13.0이다. 기존25.8.1/22.22.2는 복원용으로 보존했다. 기존 pnpm10.33.0과
Codex0.166.0은 재설치/버전 변경 없이 파일별 SHA를 대조해 복사하고 새 Node에서 실행했다.
새 로그인과 같은 PATH의 shell에서 Node26.13.0/npm11.20.0/pnpm10.33.0/Codex0.166.0을 확인했다.
이미25.8.1을 상속한 shell은 계속 그 버전을 유지할 수 있으며 `nvm use default`로 전환한다.
프로젝트별 native addon·임의 사용자 프로젝트 전체의 호환성 인증은 아니다.

Node 자체는 현재 helox runtime updater의 사용자 nvm 설치 경로가 아니므로 nvm을 사용했다.
helox 고정 Node image·lock·설치 binary·policy는 변경하지 않았다. 최초 helox 시험은 재부팅으로
사라진 `/run/heliopause-observer`에서 거부됐다. 실제 container0/helper0과 기존 규칙 확인 후
이 디렉터리만 UID/GID1000·0700으로 복구했다. 이후 `helox npm inspect is-number@7.0.0`은
필수3검사 COMPLETED/최종 ALLOW다. Protected runtime6파일 bytes/mode/owner는 동일하다.
Doctor의 runtime4OK와 `RELEASE_INSTALLATION_UNAVAILABLE`/exit1을 구분한다.
원본은 `docs/github-logs/system-recovery-20261105/helox-npm-inspect-restored.json`이다.

## 정리·독립 Windows 압축

전체 root proc의 FD/maps/cwd/exe 참조0·query errors0·활성 build0 뒤 명시한 여섯 Go/Staticcheck/npm/pip
재생성 cache만 비웠다. 최초 inventory17341935616B, canonical checker build 후 삭제 직전
3236511744B이며 중간 감소14105423872B와 이번 명시 삭제량을 구분한다. 삭제 전 identity·일반파일·
소유권·link count를 확인했고 명시 삭제의 WSL available 증가는3253714944B다.
고정 tools/modules·Docker21images·실패 원본/과거 qualification state·환경·모델·Evidence는 보존했다.
`/tmp`의 소실은 이번 삭제 결과가 아니다.

Windows C:는 최초 약9.3GiB free, VHDX259012952064B였다. Linux 논리 정리는 Windows 물리 압축이 아니다.
새 보관 위치는 `F:\Heliopause-qualification\recovery-20261105`다. 모든 Git refs의 bundle과
현재 tracked source/로컬 규칙/이번 구현 원본 archive는 전체 readback으로 검증했다.
최종 checkpoint·변경 patch·runtime/Node hash·검사기·새 Windows worker도 같은 위치에 둔다.
전체 WSL image backup은 아니며 F:의 과거52GiB qualification image는 삭제하지 않는다.

새 worker는 현재 ready/backup/source/runtime/Node·WSL registry/VHD identity를 확인한 뒤 sync/fstrim,
60초 예고, Ubuntu-24.04 종료, detached 확인, readonly mount/Optimize-VHD Full/detach와 재시작 검증을
수행한다. Virtual maximum1TiB는 유지한다. 재시작 시 observer 디렉터리는 부재·container/helper0
조건에서만 같은0700으로 복구한다. 사용자 자료/volume 삭제·온라인 compact·강제 fsck는 하지 않는다.
실제 성공과 Windows 회수량은 F:의 **새 `result.json`과 `restart-verification.json`만** 소유한다.
준비/worker 시작은 압축 성공이 아니다. 실패/guard가 있으면 자동 반복하지 않는다.

이번 원본은 `docs/github-logs/system-recovery-20261105`에 보존한다. 공개 push/CI/제품 변경은 없다.
