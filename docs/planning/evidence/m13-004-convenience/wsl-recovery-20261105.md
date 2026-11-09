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

## 압축 결과 확인·터미널 종료 후 재개 준비

2026-11-05 재개에서 실제 HEAD는 `a93615357d0120a5fad5344af46760dcd95abd91`이며,
tracked 변경은 없었다. F:의 `result.json`은
`COMPLETE_SOURCE_RUNTIME_NODE_VERIFIED_INPUT_RECONSTRUCTION_REQUIRED`,
`restart-verification.json`은 exit0/errors=[]/같은 HEAD다. VHDX는
259012952064B에서127814074368B로 줄어131198877696B(약122.2GiB)를 회수했다.
C: free는141240803328B이며 virtual maximum1TiB는 유지됐다. 이는 해당 Windows
worker의 완료 시점 값이며 현재 free 재측정이나 전체 WSL image backup을 뜻하지 않는다.
두 원본의 exact bytes 사본은 기존 raw 폴더의 `windows-compaction-result.json`과
`windows-restart-verification.json`에 보존했다. F: 원본/worker/guard는 수정·재실행하지 않았다.

현재 Git full fsck는 exit0/출력 없음이다. Checkpoint JSON 자체의 SHA256
`449680a5b42e826ac950c1d6c687ebc1fc720af9d7d0f847bc018ac600e394a9`와
소스·로컬 규칙3494개/Node5개/backup7개의 파일 해시·backup 크기를 대조했다.
설치 runtime은5개 해시가 일치했고 `/etc/heliopause/pod-init.json`은 현재 sandbox의
읽기 권한 거부로 재대조하지 못했다. 최초 assessment의 `FAILED`/Permission denied는
그대로 보존한다. 이를 runtime 변조나 전체 runtime 재검증 성공으로 바꾸지 않으며,
이전 root 재시작 검증의 성공과 구분한다. 이 범위는 제품 검증이 아니다.

재개용 clean clone을 다음 **저장소 내부의 기존 raw 영역**에 만들었다. `/tmp`와 달리
터미널 종료/WSL 재부팅으로 자동 소실되는 위치가 아니다. `.git` 객체는 hardlink 없이
복사했으며 HEAD는a936153/전체 tracked3488개 bytes가 checkpoint와 같다.

```text
docs/github-logs/system-recovery-20261105/resume-work/source
```

이 위치의 source를 updater의 `--root`로 사용한다. 같은 `resume-work` 아래의
`source-input`, `sdk-apks`, `supply`, 새 `state`는 source clone과 분리해 실제 입력을
복구할 때 만든다. 아직 이 디렉터리들에 공급물/state가 복구된 것은 아니다.
원본 Root나 raw 전체 cwd에서 checker를 실행하지 않고 해당 clean source에서 실행한다.
원본3494개 대조와 clone3488개 대조는 local rules6개 포함 여부가 다른 집합이다.

다음 재개 순서는 다음과 같다.

1. INDEX·관련 오답노트·Queue·이 문서를 읽고 실제 Git을 확인한다. 이 문서의 준비 commit은
   a936153 위의 문서 변경이며 제품 source를 바꾸지 않는다. Clone의 HEAD/clean 상태와
   현재 제품 source 변경 유무를 먼저 비교한다. 더 최신 제품 변경이 있으면 이전 clone을
   최신 후보로 가장하지 않는다. 기존 assessment는 실행 당시의 고정 결과다.
2. 위 표의 exact core/source/notice와 원래1036행 source collection·SDK APK 입력을
   기존 owner로 복구한다. F archive는 source/증거 백업이며 이 세 공급물이나 frozen
   state의 백업이 아니다. `s04-semgrep1183-source-inputs.json`도 upstream 검토 자료이며
   소실된1036행 `source-inputs.json` 전체의 대체물로 취급하지 않는다.
3. 기존 `scripts/update-runtime`을 고정 Go1.27.3으로 해당 clean source에서 build한다.
   `package-semgrep-source`/`rebuild-semgrep`/`materialize-semgrep`을 재사용하고 위
   bytes/hash를 확인한다. 복구를 위한 재생성을 기존 성공 qualification의 새 성공으로
   세지 않는다. 원본 Bash/Go/CPP/native 검증은 반복하지 않는다.
4. 기존 same-engine refresh를 재현하려면 별도 clean 후보에서 기존 reviewed projection의
   **세 owner 파일만** 원래 preimage로 되돌려 versioned baseline을 만든다. 원본 Root와
   준비된 checkpoint clone은 보존한다. Preimage/after 해시는
   `s04-bash-owner-reviewed-projection-v1.json`이 소유하며 다른 runtime 값은 유지한다.
   이미 최신 tuple인 clone에 같은 proof를 바로 plan하면 기존 owner가 unchanged로 거부한다.
5. 개발 UID1000으로 기존 `plan --component semgrep --version 1.183.0 --semgrep-proof`
   → `apply` → `verify --go ABSOLUTE_PINNED_GO`를 새 persistent state에 실행한다.
   `--root`/`--state`를 모두 명시하고 state는 source 밖에 둔다. 새 plan ID·candidate 전체
   bytes·journal·최초 실패/각 profile log를 기록한다. 옛plan ID/stdout로 state를 꾸미거나
   root로 verify하지 않는다. 현재 verify는 계속 `NOT_COMPLETED`다.
6. 각 단계의 입력/명령·exit/result·state 위치를 같은 raw와 이 leaf에 기록한 뒤 다음 단계로
   간다. 터미널이 종료되면 먼저 기존 PID/journal/result의 실제 상태를 확인하고 살아 있는
   실행을 중복 시작하지 않는다. 새 세션의 재개는 이 문서에서 가능하며 자동 실행 워커나
   새 CI/타이머를 등록한 것은 아니다.

재개 준비는 **완료**, 정확한 공급물·frozen state 복구와 updater verify는 **미완료**다.
전체 S04/보강/hosted acceptance의 상태는 바뀌지 않는다. 최초 평가 원본과 최종 준비
원장은 기존 raw의 `resume-assessment-*.json`/`resume-preparation.json`이 소유한다.

## 실행 전 Host 재검토 — 2026-11-05 09:46–09:48 UTC

사용자 요청으로 실행 전 실제 Host를 읽기 전용으로 재검토했다. 이 시점 Root는f080e09/
tracked clean이며 a936153 이후 변경은 위 문서 두 개뿐이다. 영속 clone은a936153/clean,
선택 owner5파일은 Root와 같고 reviewed preimage3개도 기존 Git에서 정확히 확인했다.

- 실제 Host의 root는 같은 UUID의 `rw`다. Sandbox의 root `ro`/가려진 proc·run을 Host
  장애·자원 부재로 전용하지 않는다. WSL available916175630336B/C: free140437155840B다.
- 보호된 runtime6파일 모두 bytes/mode/UID/GID가 checkpoint와 같다. 앞선 sandbox의
  `pod-init.json` 읽기 제한은 이 실제 Host 조회에서 해소됐으며 과거 FAILED 원본은 유지한다.
- Docker client/server29.9.2, 고정 runtime 이미지5개와 Semgrep cold SDK base의 정확한
  digest/amd64 조회가 모두 성공했다. 정책 서비스는 active/enabled다. 단순 image 목록의
  표시 수를 고정 digest의 부재로 판단하지 않는다.
- Container0/관련 build·observer·runsc·updater process0/query errors0이다. 실제 Host의
  `/run/heliopause-observer`는 부재다. 실제 동적 실행 전에 기존 부재/유휴 조건과 owner를
  다시 확인하고 UID/GID1000·0700으로 복구해야 한다. 이번 검토에서는 만들지 않았다.
- 공급물3개/source collection/SDK APK 입력/새 state와 이전 user-v3 state는 여전히 없다.
  기존 same-engine plan의 unchanged 거부와 frozen plan/journal/candidate 검증 경로를
  source에서 대조했다. 새 state의 apply 완료·candidate 확인 전에는 verify를 시작하지 않는다.

판정: **입력 복구 단계 재개 가능 / updater verify 즉시 실행 불가**. 이번에는 build·복구·
activation·cleanup·native 재시험·CI를 실행하지 않았다. 실제 원본은 기존 raw의
`pre-resume-host-review-*.json`과 `pre-resume-images-*.json`이다. 복구 준비 완료와
실행 준비 완료를 구분하며, 재부팅 뒤에는 위 volatile 조건을 다시 확인한다.

## 개발 작업의 정리 보완 — baseline e0fbde8

사용자가 작업용 스크립트 보완을 승인했다. 기존 CI별 EXIT handler와 제품 C01 Finish는
각자의 owner를 유지한다. 로컬 raw-free 문서 검사 operator에는 공용 생성 원장·종료 정리·
재시작 미완료 확인이 없다. 해당 개발 실행 경로의 IMPLEMENTED/WIRED/QUALIFIED/
ACCEPTANCE_CLOSED는 시작 시 NO다. 기존 원본 operator/log는 편집하지 않는다.
수정 범위는 로컬 검사 workspace 실행 스크립트 하나와 해당 실패/중단/삭제 거부 검증,
이 문서/Queue의 사용 경로다. 제품·CI·Host 설정은 별도다. 생성 source/cache/tmp를 한
전용 workspace에 모으고 외부 영속 원장·로그를 남긴다. 정리 미완료는 실행 성공과
분리해 비정상 종료하며, 재시작 시 유휴/신원/참조가 확인된 자체 workspace만 회수한다.

### 현재 개발 검사 실행 경로

`scripts/check-workspace.py`는 기존 `scripts/check`를 임시 source projection에서 실행하는
개발용 wrapper다. 실행 전 Git 관리 파일의 현재 bytes를 복사하고 전용 GOCACHE/TMPDIR를
사용한다. 새로 추가한 파일은 검사 전에 명시적으로 stage해야 포함된다. 로그/result와
PID/start/boot·workspace 신원은 아래 영속 state에 남는다. 이 경로에서 검증한 소비자는
canonical docs이며, Git history/native/Host fixtures가 필요한 검사는 기존 전용 owner를 따른다.

저장소 루트에서 다음 순서로 실행한다. `status`가 실패하면 `recover`로 이전 자료를
확인·수거하고 다시 status를 확인한다. 활성/불확실한 자료는 보존돼 계속 실패한다.

```sh
sudo -n python3 scripts/check-workspace.py status --state "$PWD/docs/github-logs/system-recovery-20261105/check-state"
sudo -n python3 scripts/check-workspace.py recover --state "$PWD/docs/github-logs/system-recovery-20261105/check-state"
sudo -n python3 scripts/check-workspace.py run --state "$PWD/docs/github-logs/system-recovery-20261105/check-state" -- /home/winamp/.local/bin/go run ./scripts/check docs
```

Host 전체 참조 조회를 위해 wrapper만 root로 실행하며 검사 명령은 source 소유 개발
사용자로 실행한다. 성공·실패·SIGTERM/SIGINT 모두 child 종료 후 같은 정리 경로를 거친다.
SIGKILL/재부팅에서는 영속 원장을 다음 시작에 검사한다. 정리 미완료는 새 실행을 차단하고
명령의 최초 nonzero exit는 보존한다. 명령 성공 후 정리 실패도 wrapper exit1이다.
결과의 command_exit와 cleanup을 함께 확인해야 완료다. 직접 명령이나 workspace 밖에
생성한 파일/Docker 자원까지 관리하는 강제 보안 경계는 아니며, 기존 runtime 정리 owner와
고정 SDK/모듈·재개 clone은 보존한다. 전역 Go cache는 자동 삭제하지 않는다.

Host 회귀7개는 성공/exit7·개발 UID·로그 보존, 열린 참조의 수거/새 실행 거부,
symlink/hardlink/신원 교체 거부, 조회 실패/불완전 원장 거부, SIGTERM143/수거,
SIGKILL 뒤 살아 있는 child 보호와 종료 뒤 복구, 정리 실패 시 exit1/원래 exit7 보존과
입력 파일 지문 기록을 검증했다. 첫 hardlink 시험의 EXDEV는
OneDrive→ext4 시험 준비 오류로 보존했고 같은 ext4의 두 링크로 교정한7.473s는 PASS다.
후속 최종7개11.064s도 PASS다. 실제 canonical docs는 첫 wrapper 실행에서 exit0/
CONFIRMED_REMOVED였다. 이전 누락 사본2개는 기존 신원·Git 전체3488개 파일+검사 문서
지문·참조/mount 부재/query error0을 대조한 뒤111374336B allocated를 수거했다.
영속 clone/원본/공유 cache는 보존했다. 이 경로의 IMPLEMENTED/WIRED/QUALIFIED/
ACCEPTANCE_CLOSED=YES는 위 개발 workspace·회귀·docs 소비자 한정이다. 제품 C01/C02
전체 및 모든 임의 명령/외부 자원 정리 완료로 확대하지 않는다. 개발 명령도 wrapper를
우회하면 자동 관리되지 않는다. 코드·원본·최종 검사·정리 증거는 같은 raw의 workspace-* /
completed-doc-copies-collection.json 및 check-state가 소유한다. 다음 updater 재개 때는
이번 개발 스크립트 변경도 최신 versioned baseline 대조에 포함하고 기존a936 clone은 보존한다.

2026-11-07 wrapper의 첫 canonical quick(7m37s)은 exit1/CONFIRMED_REMOVED였다. 최초 실패는
runtimeidentity PatchDrift의 실제 runsc 분기다. Host 설치 gVisor patch588adad4는 현재 lock
c8a546d0보다 이전이라 `sentry/filesystem_mutation`이 없고, lock SHA512 일치 후보는 `/tmp`
소실로 Host·F:에 없다. Assertion을 끄거나 존재하지 않는 HELOX_RUNSC_BINARY로 우회하지 않으며
현재 bundle 재구성 전까지 이 분기는 입력 MISSING이다. 두 번째 promotion socket 실패는 긴
`/tmp/haa-work-check-<32hex>/tmp` TMPDIR로 unix socket 108byte를 넘긴 wrapper 환경 오류다.
Workspace를 `/tmp/hcw-<16hex>`로 줄였고 Host 회귀7개8.468s와 같은 test(0.067s)가 PASS다.
이미 완료된 이전 이름 receipt는 재수거 대상이 아니다. Docs는 exit0과 tracked 두 문서의 크기
경고를 출력했다. 다른 quick 패키지는 같은 실행에서 ok였고 vet/Staticcheck가 앞서 통과했다.

같은 날 현재 lock의 gVisor bundle을 clean clone(f8de8d6)과 기존 `build-gvisor-runtime-bundle.sh`로
`/var/tmp/haa-resume-20261107/gvisor-bundle`에 재구성했다. 고정 builder image와 Bazel 빌드가
8m19s/exit0이며 6 member size/SHA512가 모두 lock과 같다. 이 runsc를 HELOX_RUNSC_BINARY로
명시한 runtimeidentity 전체가 PASS다. Host 설치 runtime은 바꾸지 않았다. `/var/tmp`는 재부팅에도
남지만 전체 백업은 아니다. Semgrep 공급물 3개와 1036행 source collection·SDK APK는 Host와 F:에
없다. 이를 만드는 수집 owner가 tracked source에 없고 기존 원장도 공급 구조를 미결정 MISSING으로
기록하므로, 새 수집 도구 없이 updater verify를 재개할 수 없다(BLOCKED). staging 기록
`s04-{opam,native,extra}-source-offer-staging-1183.json`은 출처·checksum 자료이며 원래 입력의
대체물이 아니다.

### CI452 — 119 commit 묶음의 첫 hosted 결과와 최소 교정

9ca552e를 정상 push해 PR #38의 [CI452](https://github.com/rahoney/heliopause/actions/runs/42279560971)를
실행했다. 단일 20분 worker의 GET2(19:48:36Z)가 terminal failure를 소비했고 worker는 자연 종료했다.
Quick·Docs·Security·Minimum Go·gVisor Observer·Wheel Corpus 6개는 SUCCESS, Vulnerability·
gVisor Linux Integration·macOS·Required는 FAILURE다. 세 최초 실패는 서로 독립이다.

1. Vulnerability: GO-2026-6865(klauspost/compress1.18.6 two-block frame panic)가
   `CaptureBuildOutput→tar.Writer.Close` 경로로 reachable이다. 공식 fixed1.20.2로 해당 module만
   올리고 clean clone에서 tidy했다. Suppression은 없다.
2. macOS: OS 제약 없는 `linux_integration_test.go`가 db32813부터 Linux 전용 adapter를 호출해
   Darwin test compile이 깨졌다. 해당 3함수만 byte 동일하게 `github_release_integration_linux_test.go`로
   옮겼다. darwin arm64/amd64·linux vet가 통과했다.
3. Linux integration: npm lifecycle과 GitHub ELF 시험이 243dbfb·db32813 뒤 observation resource
   서비스를 쓰지만 CI는 plain `go test` binary로 실행했다. 서비스는 client_path inode만 승인한다.
   기존 `haa_install_ci_client`→`helox -test.run`→실제 helox 복원 방식으로 두 줄만 옮겼다.
   그로 인해 첫 restore 줄을 지우던 pipx negative 시험의 mutation 대상을 pipx 복원 위치로 좁혔다.

교정 후보의 canonical quick(380s, 재구성 runsc 명시)·vulnerability(120s)·workflow(107s)·
security(130s)가 wrapper에서 PASS다. 원본 job log 3개와 poll 기록은 `docs/github-logs/ci452-20261107`이
소유한다. 실제 hosted 통과는 다음 CI에서 확인한다.

### CI453 — 교정 확인과 hosted Semgrep 공급 차단

ab63980의 [CI453](https://github.com/rahoney/heliopause/actions/runs/42287744483)은 GET3(21:02:33Z)에서
terminal failure였다. Worker는 자연 종료했다. 7 SUCCESS에는 Vulnerability가 포함되고, Linux lifecycle·
GitHub ELF 두 시험도 PASS로 바뀌어 CI452 세 교정이 hosted에서 확인됐다. 남은 실패는 둘이다.

1. macOS: compile 문제가 풀리자 darwin `t.TempDir()`의 `/var`→`/private/var` symlink가 새 storage/
   evidence의 symlink 거부 경계에 걸린 5개 package 시험이 드러났다(CI452에서는 compile 오류에 가려져
   있었다). 제품 경계는 그대로 두고 기존 관례인 `filepath.EvalSymlinks(t.TempDir())`만 다섯 시험에
   적용했다. Linux의 symlink TMPDIR로 RED5→GREEN5와 일반 TMPDIR GREEN5, darwin vet을 확인했다.
   같은 조건의 전체 실행에서 나온 promotion 실패는 `_linux_test.go`만 해당하고 macOS promotion은
   hosted ok여서 이번 범위가 아니다.
2. Linux integration: 실제 `helox npm install`/`pip install`이 `immutable installed native tool supply is
   unavailable or changed`로 실패했다. S04 내용 분석이 production 설치 경로의 필수 검사인데 CI에는
   Semgrep core/rules 공급이 없다. 이는 위 S04 BLOCKED와 같은 공급 구조 미결정이며 CI 수정만으로
   닫을 수 없다. Hosted ordinary·CUDA qualification은 이 결정 전까지 **BLOCKED**다. 같은 실패를
   확인하려고 CI나 CUDA dispatch를 반복하지 않는다.

### Semgrep 공급 재수집 — 선택 A 진행 계획(2026-11-07)

사용자가 자체 build 공급을 같은 저장소 Release asset으로 운송하는 A를 선택했다. 저장소 공개 여부는
보류하며 Release 게시는 실행 직전 별도 승인을 받는다. 사라진 최종 1036행 `source-inputs.json`은
없고 lock은 그 hash만 소유한다. Git pack·직접 쓴 README/SOURCE-ORIGINS는 byte 재현이 불가능하므로
SOURCES/NOTICES hash는 같은 upstream identity의 기존 same-engine refresh(plan/apply)로 갱신한다.
Core는 같은 의미 입력이면 기존 `--expected-sha256 33ae4500…`으로 재현을 검증한다.

복원 근거는 `docs/github-logs/goal-implementation-20261101`의 보존 원본이다. 1025행
`s04-notice-package-first-inputs-1183.json`(member path/hash), APK160
`s04-source-builder-apk-inputs.json`(v3.23에서 160개 모두 HTTP200 확인), native15
`s04-native-source-archive-inputs.json`, Git pin8 `s04-git-input-notices-1183.json`,
최종 변경4·generated8 hash `s04-taint-sink-source-package-v3.json`, patch/rules 원문이다.

단계: ① 보존 기록과 고정 upstream으로 origins spec을 재구성해 저장소에 추적한다. ② 기존
`scripts/update-runtime`에 그 spec만 읽어 받고 hash를 검증하는 수집 명령을 추가한다(재부팅 재발
방지). ③ 기존 package/rebuild로 core 재현을 확인한다. ④ same-engine plan/apply와 S04 영향
qualification. ⑤ 승인 뒤 Release 운송·CI 연결·한 번의 CI와 CUDA 순차 실행.

재수집 시험(`/var/tmp/haa-semgrep-recollect-20261107/probe`, 제품·lock 불변): 1025행 기준으로
opam-repository 223개는 기록된 두 번째 고정 commit `922f67e0`에서 223/223 byte 일치했다(`78d29aba`만으로는
161/223). opam archive·extra source 158/158, native 15/15(gmp는 원 주소 연결 거부로 GNU mirror에서 받아
sha512·크기 대조), aports 97/97(보존 원문 84개+aports Git 13개), notice 원문은 275개 중 272개가 고정
archive/Git 파일과 완전히 같았다. 남은 것은 직접 쓴 README·catalogue와 원문 두 개(11347B·1076B)다.
대조 결과는 `docs/github-logs/system-recovery-20261105/semgrep-recollect-notice-match-20261107.json`이다.
Git pack·직접 쓴 문서는 새 bytes가 되므로 SOURCES hash 갱신 판단은 위와 같다.

재수집 결과(2026-11-07, 공급 재구성 완료·hosted 미검증):

- 보존 1025행은 원래 V6 교정 전 시점이라 opam 설치가 `rresult.0.7.0`에서 멈췄다(원래 V6와 같은 지점).
  공식 sha512의 8506B archive와 ISC 원문을 원래 교정대로 더했고, lock 236개 의존성과 대조해 다른 누락은 없다.
- 보존 SDK builder image `c51621f9`에서 같은 worker로 만든 grammar 생성기가 원래와 같은 `412cf8c9`였다.
  CLI 0.26.3으로 생성한 TypeScript/TSX generated 8개 hash가 기록과 모두 같다.
- 마지막 미확인 고지 `c70946fa`(11347B)는 이 저장소의 Apache-2.0 `LICENSE`다(HAA patch/rules/worker 고지).
  `e1d65ebd`는 ocaml-trace `f57744fe`의 `LICENSE`, 빈 고지는 원래 정리대로 제외했다.
- 최종 SOURCES 1036 files `f36d9199…`(327986176B, manifest `ffd27630…`), NOTICES 276 members
  `c30784ef…`(1997312B)로 원래 개수와 같다. 보존 행과 다른 것은 Git pack/config·직접 쓴 문서·
  최종 patch/rules/worker·generated8·rresult뿐이다.
- `rebuild-semgrep`의 첫 시도는 Host 설치 gVisor bundle(patch `588adad4`)이 현재 lock과 달라 executor가
  3초 만에 거부했다. 원래 S04 방식대로 private mount namespace에서만 lock 일치 bundle을 runtime 경로에
  bind했다(Host manifest 불변·잔여 mount 0). 재수집 입력과 최종 입력 모두 core `33ae4500…`을 정확히
  재현했다(1594s/1584s).
- Clean clone same-engine plan `6727a905…`/apply는 lock과 생성 Go 상수 두 파일만 바꿨고 upstream·core·
  rules·patch·worker는 불변이다.
- Updater verify 1차는 bootstrap/quick PASS, security FAIL이었다. gosec가 `encoding/base64` metadata와
  sandbox type 정보를 얻지 못했는데, 같은 시각 16GB 제한 core 재빌드가 동시에 실행 중이었다. 같은 source에
  CI 변경을 더한 작업 트리의 canonical security는 wrapper에서 PASS(215s)다. 원인은 자원 경합으로 추정하며
  확정하지 않았다. 2차는 실패 test 없이 quick default test 6분 예산 초과였다. 같은 verify를 반복하지 않고
  작업 트리 canonical profile로 대체 확인한다.
- CI 연결: `haa_ci_fetch_semgrep_supply`가 lock에서 도출한 tag `semgrep-<version>-<SOURCES sha 12자>`의
  Release asset core를 받고 rules는 checkout에서 가져온다. `haa_ci_semgrep_source`가 lock size/SHA·
  root 0555/0444를 확인하며 responder base와 pnpm/bun/pipx fixture version에 같은 파일을 설치한다.
- 사용자 승인 후 Release [semgrep-1.183.0-f36d91990edb](https://github.com/rahoney/heliopause/releases/tag/semgrep-1.183.0-f36d91990edb)
  (prerelease·not latest·immutable, target ab63980)에 core/SOURCES/NOTICES를 올렸다. GitHub digest와 익명 다운로드
  core SHA가 lock과 같다. 첫 시도는 저장소의 immutable release 설정을 확인하지 않고 즉시 공개해 asset 업로드가
  422로 거부됐다. 승인받아 빈 Release는 삭제(204)했으나 tag `semgrep-1.183.0-33ae45003cb9`는 보호되어 남는다.
  그래서 tag 도출을 core hash에서 SOURCES hash로 바꿨다. 이후 Release는 draft→upload→digest 대조→공개 순서만 쓴다.

계획 ①②의 저장소 추적 spec·수집 명령은 추가하지 않았다. Release asset이 SOURCES의 영속 보관소가 되고,
이번 재수집은 보존 hash로 검증한 개발 스크립트로 충분하다. 제품/updater에 미래 소비자 없는 수집 owner를 만들지 않는다.

재수집 스크립트·build/plan/verify 원본·요약은 `docs/github-logs/system-recovery-20261105/semgrep-recollect-20261107/`이 소유한다.

### CI454 — Semgrep 공급 hosted 확인과 Terraform 서명 재인증 교정

32cba86의 [CI454](https://github.com/rahoney/heliopause/actions/runs/42345891672)는 GET2(06:32:50Z)에서
terminal failure였다(8 SUCCESS·Linux와 Required FAILURE). macOS 다섯 시험 교정은 hosted로 확인됐다.
Linux에서는 Release 공급 core/rules로 `helox npm install`·`pip install`이 모두 COMPLETED여서 Semgrep 공급
연결이 hosted로 확인됐다. 그다음 public Terraform 시험의 `hashicorp/random@3.7.2`만 서명 검증에 실패했다.

원인: HashiCorp가 공식 key를 2026-02-18에 다시 서명(만료 2030)하면서 registry가 2021년 self-signature를
뺀 armor를 제공한다. 10월 보존 registry 응답은 2021년 self-signature를 가졌다. 검증기는 서명 시점
(2025-04-22)의 binding을 registry key로 요구해 실패했다. gpg와 고정 인증서로는 같은 서명이 유효하다.
사용자 승인으로 gpg/Terraform CLI와 같은 기준을 택했다. 고정 공식 인증서의 subkey가 현재 유효·비폐기이고
서명이 그 subkey로 검증되며 서명이 subkey 생성 이후이면 허용한다. 서명 시점 binding이 있으면 기존처럼
같은 key인지 확인하고 partner/community 규칙은 바꾸지 않았다. 실제 재인증 armor 고정 fixture의 회귀가
이전 코드 RED, 수정 후 GREEN이고 public 시험 두 provider도 로컬 PASS다. 원본은 `docs/github-logs/ci454-20261107`이다.

### CI455 — Terraform 교정 확인과 PNPM probe client 교정

020473c의 [CI455](https://github.com/rahoney/heliopause/actions/runs/42350029595)는 GET3(07:50:13Z)에서
terminal failure였다(8 SUCCESS·Linux와 Required FAILURE). public Terraform 시험 두 provider는 hosted PASS다.
다음 최초 실패는 `TestPNPMActualReleaseArtifactProbe`의 `observation resource response is untrusted`다.
CI452 교정과 같은 원인이다. 이 시험은 observation resource 서비스를 쓰지만 `ci-pnpm-runtime.sh`가
plain `go test` binary로 실행했다. 서비스는 설치된 client inode만 승인한다. 이전 CI에서는 앞선 실패가
이 지점을 가렸다. probe 하나만 이미 설치된 realcorpus client(`/usr/libexec/heliopause/helox -test.run`)로
옮기고 validator 상수와 negative(plain client·skip·주석·중복)를 더했다. 원본은 `docs/github-logs/ci455-20261107`이다.

### CI456 — PNPM probe 교정 확인과 instance 시험 resource adapter 누락

cc3f711의 [CI456](https://github.com/rahoney/heliopause/actions/runs/42354232529)은 GET3(08:50:27Z)에서
terminal failure였다(8 SUCCESS·Linux와 Required FAILURE). `TestPNPMActualReleaseArtifactProbe`는 hosted PASS다.
다음 최초 실패는 `TestPNPMActualProductionProjectInspection/transitive`의 instance coverage incomplete다.
instance 실행은 `Backend.resources`(observation resource client)가 없으면 `M3_DYNAMIC_ACCOUNTING_UNAVAILABLE`로
끝난다. 8293b20에서 instance real 시험에는 adapter를 붙였지만 이 시험과 `TestBunActualArtifactExecutionBoundary`는
`NewBackend`만 써 빠졌다. NewBackend를 쓰는 realcorpus 시험 전체를 대조해 두 곳에만 기존
`integrationObservationResourceAdapter(t)` 한 줄을 더했다. 로컬은 tagged vet·gofmt만 확인했다. 두 시험은 설치 client
교체가 필요해 hosted에서 확인한다. 원본은 `docs/github-logs/ci456-20261107`이다.

### CI457 — adapter 교정 확인과 instance trace 기대값 교정

67396b0의 [CI457](https://github.com/rahoney/heliopause/actions/runs/42361309449)은 GET3(10:30:35Z)에서
terminal failure였다(7 SUCCESS·Linux·macOS·Required FAILURE). PNPM project inspection 자체는 이제 완료된다.
남은 Linux 실패는 시험의 trace 수 단언(`expected 2, observed 9`)이다. 8293b20부터 instance 하나가 두 환경에서
trace 4개를 남기며 형제 시험 `TestPNPMActualProductionInstanceExecution`은 `before+4`로 갱신됐다. 이 시험은
앞선 실패에 가려 `before+expected+1`로 남았다. prepare 1 + instance당 4로만 고쳤다. prepare 전용 시험의
`before+1`은 instance 환경과 무관해 그대로다. macOS는 quick의 `internal/promotion` 실행 중 test deadline을 넘었다.
macOS에서 컴파일되는 source는 CI456과 같다. platform step은 이전 성공 394/407/320s, 이번 469s다. 하위 원인은
UNKNOWN(hosted 시간 변동)으로 보존하고 한도는 바꾸지 않았다. 원본은 `docs/github-logs/ci457-20261107`이다.

### CI458 — trace 교정 확인과 resolver·Bun project 시험 resource 누락

9285c7d의 [CI458](https://github.com/rahoney/heliopause/actions/runs/42364554565)은 GET3(11:16:44Z)에서
terminal failure였다(8 SUCCESS·Linux와 Required FAILURE). macOS는 SUCCESS다. PNPM project inspection과
instance execution 시험은 hosted PASS다. 다음 최초 실패는 `TestPNPMPublicNativeResolverIntegration/single`의
instance coverage incomplete다. 시험이 `NewLinuxBackendWithExecutor(..., nil)`로 resource client 없이
backend를 만들었다. CI456 대조는 출력 절단으로 이 생성자 호출을 놓쳤다. 전체 생성자 호출을 다시 대조해
nil인 실제 시험 두 곳(이 시험과 `TestBunPublicProjectQuarantineIntegration`)에만 기존 adapter를 넣었다.
나머지 nil은 생성자 거부 단위 시험이다. 원본은 `docs/github-logs/ci458-20261107`이다.

### CI459 — resolver quarantine 완료와 workspace evidence 기대값 교정

db608e1의 [CI459](https://github.com/rahoney/heliopause/actions/runs/42367949113)는 GET2(11:44:17Z)에서
terminal failure였다(7 SUCCESS·Linux·macOS·Required FAILURE). PNPM resolver quarantine 자체는 이제 ALLOW로 끝난다.
남은 Linux 실패는 시험의 evidence 수 단언이다. 8293b20부터 `npmProjectInstanceFacts`가 instance마다 환경별
`workspace-state` evidence 2개를 더한다(PNPM·Bun 공용). 그래서 `4+2*instances`를 `4+4*instances`로 고쳤다.
같은 단언을 가진 `TestBunPublicProjectQuarantineIntegration`도 함께 고쳤다. check 수(`3+instances`)는 analyzer가 없는
시험 구성이라 그대로다. 실패 시 실제 수를 출력하게 했다. macOS는 CI457과 같이 quick 실행 중 `internal/promotion`에서
test deadline을 넘었다(3회 중 2회). macOS source는 CI456과 같고 Mac 한도는 사용자 규칙상 바꾸지 않는다.
하위 원인은 UNKNOWN이다. 원본은 `docs/github-logs/ci459-20261107`이다.

### CI460 — PNPM·Bun evidence 교정 확인과 Node 지연 자극 미완료

183faa7의 [CI460](https://github.com/rahoney/heliopause/actions/runs/42369821766)은 GET3(12:31:54Z)에서
terminal failure였다(7 SUCCESS·Linux·macOS·Required FAILURE). PNPM resolver 시험은 hosted PASS다.
다음 최초 실패는 `TestLinuxStimulusNodePolicyCorpusIntegration/delayed-chain`이다. dynamic lifecycle이
`M3_DYNAMIC_EXECUTION_FAILED`로 끝났다. normal·config-read·conditional-chain은 같은 host에서 PASS다. 첫 환경 trace는
완료됐고 두 번째(모의 개발환경) trace 출력 전에 끝났다. 11-03 로컬 v1/v6에서는 같은 case가 network=10으로 PASS였다.
sandbox 오류는 dynamic inspector가 INCOMPLETE 코드로 접는 설계라 시험에서 원인이 보이지 않는다. 그래서 Node 시험에만
`Execute` 첫 오류를 보존해 실패 메시지에 붙이는 시험 측 wrapper를 더했다. 원인은 다음 hosted 결과로 확정한다.
macOS는 이번에 production build 120s 한도를 넘었다(CI450과 같은 형태). Mac 한도는 바꾸지 않았고 하위 원인은
UNKNOWN이다. 원본은 `docs/github-logs/ci460-20261107`이다.

### CI461 — 분리 자식 후속 행동 미관찰(설계 gap)과 macOS 반복 시간 초과

3694dc5의 [CI461](https://github.com/rahoney/heliopause/actions/runs/42373505608)은 GET3(13:22:26Z)에서
terminal failure였다(7 SUCCESS·Linux·macOS·Required FAILURE). 이번에는 `delayed-chain`이 network=10으로 PASS다.
CI460의 `M3_DYNAMIC_EXECUTION_FAILED`는 같은 source에서 재현되지 않아 하위 원인 UNKNOWN(간헐)으로 둔다.
최초 실패는 `detached-child`다. dynamic check는 COMPLETED·states=2인데 분리 자식의 50ms 뒤 network가 0이다.
현재 lifecycle은 `npm install` 반환 뒤 남은 후손을 기다리지 않고 writer를 종료한다. 후손이 남았는지 Go 쪽에서
보는 신호(trace diagnostic·종료 lease)도 없다. 그래서 자식 행동의 관찰 여부가 시간차에 달렸고 11-03 로컬
v5/v6 PASS는 이 시간차 안의 결과였다. 계획 21 §5.5는 "부모 종료 뒤 자식 행동 포함·descendant 완료·관찰"과
"한도 밖 지연은 incomplete/자동 ALLOW 없음"을 요구한다. 현재 결과(COMPLETED·행동 미관찰)는 이를 충족하지
않는 제품 gap이다. 교정은 sandbox 종료 경로(필요 시 observer)의 후손 대기/미완료 판정이다. 범위가 커서 사용자
확인 전에는 착수하지 않는다. macOS는 같은 image(macos-26-intel 20260824)에서 5회 중 4회 Mac 한도
(build 120s·default test 6m)를 넘었다. 성공 run은 build 약 1분이고 실패 run은 약 2분이다. Mac 한도는 사용자 규칙상
바꾸지 않았다. 원본은 `docs/github-logs/ci461-20261107`이다.


### 후속 교정 — 분리 후손 관찰 대기와 macOS 한도(사용자 승인)

사용자는 CI461 뒤 두 결정을 내렸다: 후손 대기는 설계대로(observer 방식) 구현하고, Mac 한도는 build 240s·default
test 9m(go test 8m)으로 올린다(Mac만, `scripts/check/runner.go`의 `darwinLimit`). Runsc `ps`/`debug --ps`는 비수확
PID1 아래 종료된 고아를 계속 나열해 살아 있는 후손과 구분하지 못하고, Host cgroup은 sentry만 보인다(로컬 probe 확인).
그래서 판단 근거를 observer의 처리된 clone/exit 원장으로 한정했다. Observer는 bound stream의 각 수락 event 뒤
OCI root 외 live thread group 수를 등록 정보에 반영하고, 새 control `processes`가 `unknown/running/drained`를
답한다(stream 종료 뒤·바인딩 전은 unknown). Go `awaitObservedProcessDrain`은 lifecycle 성공 뒤 기존 wall deadline 안에서
drained를 기다리며 만료는 `M3_DYNAMIC_TIMEOUT`, 원장 상실은 `M3_DYNAMIC_OBSERVER_FAILED`(Python은
`M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE`)로 INCOMPLETE다. npm 환경·PNPM/Bun/npm-project instance·Python unit 세 경로에
연결했다. 로컬 검증: 단위(drained/한도 만료/unknown/세션 없음)·pinned bazel observer build와 전체 latch(새 drain
control 포함, 6m54s) PASS, security·docs PASS다. Quick은 vet/Staticcheck와 sandbox 포함 49개 package가 PASS지만
Host 부하(load 10~14)에서 `internal/promotion`이 전체 병렬 5분 한도를 넘었다(단독 174s PASS, 최장 test 9s, drain 경로
미사용). 설치된 Host observer는 교체하지 않았으므로 실제 분리 자식 관찰과 promotion은 hosted CI가 판정한다.

### CI462 — 분리 후손 교정 확인과 staging 시험의 내용 검사 누락

8df8500의 [CI462](https://github.com/rahoney/heliopause/actions/runs/42389595817)는 GET3(16:46:55Z)에서 8 SUCCESS·Linux·Required
FAILURE였다. macOS(새 한도)·Quick·observer helper(새 latch 포함)는 SUCCESS다. Node·Python `detached-child`는 각각 network=10·13,
MANUAL_REVIEW·approval=false로 PASS해 후손 대기 교정이 hosted에서 확인됐다. 다음 최초 실패는
`TestNPMPublicWholeProjectQuarantineIntegration/nested`의 staging 거부("archive approval lacks current source/native/inventory
Evidence")다. 4857a61이 project cache staging에 내용 검사 증거를 요구했지만 이 시험은 source analyzer 없는
`inspectionnpm.NewInspector(intake)`를 써서 제품이 정상 거부한 것이다. 같은 형태로 staging하는 npm workflow·UV project cache
시험도 production 생성자(`newNPMStaticInspector`·`newPyPIStaticInspector`)로 맞추고 required 수를 내용 선언에서 계산했다.
제품 변경은 없다. 원본은 `docs/github-logs/ci462-20261108`이다.

### CI463 — 내용 검사 연결 뒤 선택적 native 신호 미실행

590bb6d의 [CI463](https://github.com/rahoney/heliopause/actions/runs/42393326279)은 GET3(17:31:01Z)에서 같은 8 SUCCESS·Linux·Required
FAILURE다. Production 정적 검사기를 쓰자 staging 앞 검사에서 "incomplete npm quarantine check"가 났다. 이 시험은 선택 검사까지
모두 COMPLETED를 요구했지만, 제품의 cache 승인(`pnpm_cache_approval.go`)과 root CLI 시험은 native binary가 없을 때 선택
`binary-content-signals`의 NOT_EXECUTED/`M13_BINARY_SIGNALS_UNAVAILABLE`을 허용한다. 같은 규칙만 허용하고 실패 메시지에
check/상태/제한 코드를 남겼다. Required 수는 integrity/static/lifecycle 3개와 필수 내용 검사 3개(source-content·binary-content·
content-inventory)를 합친 6이며, log와 `scripts/ci-npm-project.sh` 기대값을 실제 값으로 맞췄다. 제품 변경은 없다.

### CI464 — npm project 4개 시험 통과와 단일 실행 9m 초과·macOS job 한도

c7fddfc의 [CI464](https://github.com/rahoney/heliopause/actions/runs/42396833731)는 GET4(18:32:53Z)에서 Linux FAILURE·macOS CANCELLED였다.
npm whole-project(required_per_archive=6, 191s)·workflow(191s)·root CLI(134s)가 PASS했으나 같은 실행의 마지막
`TestNPMPublicProjectCLIIntegration`이 기존 `-test.timeout=9m` 안에 끝나지 못했다. 필수 source analysis가 production
경로에 들어가 시험이 실제로 길어진 것이다. 개별 한도 값은 바꾸지 않고 같은 9m 실행 두 번(검역·workflow / root CLI·CLI)으로
나눠 validator가 두 출력을 함께 읽게 했다(사용자 사후 확인 대상). macOS는 승인된 Mac 한도(build 4m·test 9m)가 job
`timeout-minutes: 15`(설치 약 5m 포함) 안에 들어가지 못해 platform 단계 10.4m에서 취소됐다. Job 한도만 25m로 맞췄다.

### CI465 — npm project 전부 통과와 PNPM 거래 시험의 같은 생성자 누락

2798c6f의 [CI465](https://github.com/rahoney/heliopause/actions/runs/42401970695)는 GET3(19:14:46Z)에서 Linux·Required FAILURE, 나머지 8개(macOS
포함) SUCCESS다. 분할한 npm project 5개 시험이 모두 PASS했다(CLI 40.8s). 다음 실패는 `TestPNPMPublicProjectTransactionIntegration/single`의
같은 staging 거부다. 이 시험은 bindings를 통해 production pnpm project 거래가 staging하므로 이전 grep(직접 Stage 호출)에 잡히지 않았다.
production cache staging(bun/pnpm/npm project·Python requirements/uv)으로 가는 realcorpus 시험을 전수 대조해 pnpm project·bun
transaction·pnpm/bun original·UV project publication·UV tool install 6개를 production 생성자로 맞췄다. 이들의 check 단정은 required만
보거나 `< 3`이라 내용 검사 추가와 맞는다. 제품 변경은 없다.

### CI466 — PNPM 거래 시험 통과 시작과 단일 실행 5m 초과

f77544c의 [CI466](https://github.com/rahoney/heliopause/actions/runs/42405482298)은 GET3(19:57:42Z)에서 Linux·Required FAILURE, 나머지 8개 SUCCESS다.
production 생성자로 바꾼 PNPM approved cache(81.6s)와 거래 `single`(97.3s, Policy=ALLOW)이 PASS했고 `peer`가 진행 중 cache·거래
공동 실행의 기존 5m에 닿았다. 개별 한도는 바꾸지 않고 PNPM·Bun 모두 cache와 거래 `single`/`peer`/`empty`를 각각 별도 5m 실행으로
나눴다. 기존 validator는 정확한 한 줄 명령을 소유하므로 고정 명령을 줄 목록으로 바꾸고 변형 반례도 개별 줄에 적용했다
(`go test ./scripts/check` PASS). `peer` 한 실행이 5m를 넘으면 시험 한도 변경은 사용자 결정 대상으로 남긴다.

### CI467 — PNPM project 전부 통과, Bun Artifact 실행 시험 wrapper 누락·macOS 시간 변동

a46f166의 [CI467](https://github.com/rahoney/heliopause/actions/runs/42409367653)은 GET4(21:05:57Z)에서 Linux·macOS·Required FAILURE, 7개 SUCCESS다.
사례별로 나눈 PNPM cache(69.6s)·거래 single/peer/empty(85.8/135.1/6.9s)·CLI(246.2s)·original(179.8s)이 모두 PASS했다.
다음 실패 `TestBunActualArtifactExecutionBoundary/normal`은 0.96s에 INCOMPLETE였다. 8293b20(2026-11-03)이 instance backend에
`discardCommandRunner` 조건을 추가했으나 이 시험의 출력 capture wrapper는 `RunDiscard`를 전달하지 않아 accounting 조건에서
멈췄다. wrapper에 기존 runner로의 `RunDiscard` 전달만 추가했다(제품 변경 없음). 관찰 대기는 활성 session으로 그대로 요구된다.
macOS는 같은 source 계열의 CI466에서 default test 5.6m이었으나 이번에는 production build 2m7s 뒤 default test가 승인된 9m에
닿았다. 모든 package가 `ok`를 출력했고 promotion 340s·update-runtime 194s가 마지막이었다. 하위 원인은 UNKNOWN(runner 속도
변동 후보)이며 한도는 바꾸지 않고 다음 실행에서 재관찰한다.

### CI468 — macOS 재관찰 SUCCESS, PNPM original 공유 4m context 초과

095fbd9의 [CI468](https://github.com/rahoney/heliopause/actions/runs/42414388980)은 GET4(22:08:42Z)에서 Linux·Required FAILURE, macOS 포함 8개 SUCCESS다.
CI467 macOS 시간 초과는 같은 한도에서 재현되지 않았다. 이번 Linux runner는 같은 시험 전반이 약 1.5배 느렸다(PNPM native resolver
87.7→132.6s, CLI 246→366s). `TestPNPMNativeOriginalIntegration`은 세 사례가 시험 전체의 4m context 하나를 공유하는데
single 63.1s·native-update 56.8s 뒤 peer가 240.17s 시점에 deadline에 닿았다(CI467 합계 179.8s). 한도 값은 바꾸지 않고
PNPM·Bun original을 사례별(`single`/`native-update`/`peer`) 5m 실행으로 나눠 각 사례가 자기 4m context를 갖게 했다.
CLI 시험은 이미 사례별 4m context다. validator 줄 목록·변형 반례를 갱신했고 `go test ./scripts/check` PASS다.

### CI469 — 간헐 실패 3건 재실행, Bun Artifact 시험의 남은 wrapper 누락

4d93544의 [CI469](https://github.com/rahoney/heliopause/actions/runs/42419501621) attempt1은 Quick(`TestLauncherNativePTY/ko` PTY 준비 시간 초과,
internal/cli 불변)·macOS(proxy.golang.org DNS i/o timeout)·Linux(PNPM 거래 `single`의 gVisor header `dropped_count!=0`
STREAM_FAULT, CI466–468 PASS·입력 불변)가 서로 무관하게 실패했다. 세 하위 원인은 UNKNOWN이며 코드 변경 없이 실패 job만 한 번
재실행했다(CI450 선례). attempt2에서 Quick·macOS SUCCESS, PNPM 전 사례·original 사례별 실행 PASS로 간헐임을 확인했다.
다음 Linux 실패 `TestBunActualArtifactExecutionBoundary/normal`은 minimal 환경 완료 뒤 development 환경에서 responder init이
`StartContainerInputOutput`을 요구하는데 시험 capture wrapper가 전달하지 않아 생겼다. 또 8293b20 뒤 production은 lifecycle을
`RunDiscard`로 실행해 출력을 버리므로 기존 `native-Bun:` 출력 oracle(CI448 PASS 당시 `Output` 경로)이 볼 수 없다. wrapper가
`StartContainerInputOutput`을 전달하고, 정확한 Bun lifecycle 명령만 같은 admission 소유 `Output`으로 실행해 기록하게 했다
(제품 변경 없음).

### CI470 — Bun Artifact 시험 PASS, 결과 schema 기대값 누락과 분할 기록 정정

66a25f4의 [CI470](https://github.com/rahoney/heliopause/actions/runs/42428092357)은 GET4(01:04:54Z)에서 Linux·Required FAILURE, Quick·macOS 포함
8개 SUCCESS다. Linux job은 00:04:37–00:59:47Z(55m)로 step 4 Bun 구간에서 멈췄다. Node·Python `detached-child`는 이번에도
network=10·MANUAL_REVIEW·approval=false로 PASS했다(해당 사례만의 관찰이며 전체 qualification이 아니다). Bun Artifact
`normal`/`script-failure`·cache·거래·CLI·original이 PASS했다. 최초 실패는 `TestLinuxBunDeclaredEntryCLIIntegration`의
결과 축 단정이다. 9e9260f(2026-11-02)가 검사 목록이 있는 실행 결과를 `helox.execution-result/v2`로 바꾸고 npm 실행 시험만
갱신했으며, 이 시험과 `TestLinuxUVRunSupplementalExecutionCLIIntegration`(완료 사례), UV/pipx tool CLI의
`operation-result/v1` 기대가 남았다(10/31 이후 순차 마스킹 구간). 완료 사례는 v2와 비어 있지 않은 필수 검사 완료를, 준비 거부
사례는 기존 v1·거부 축을 그대로 요구하도록 맞췄다. 실패 메시지에는 축 값만 넣고 프로그램 출력·argv는 넣지 않는다. 제품 변경은 없다.

분할 기록 정정: CI464–468의 분할을 "한도 불변"으로 적은 것은 부정확했다. 각 실행의 제한값은 유지했지만 실행 단위와 전체 허용
예산이 바뀌었다 — npm project 9m×1→9m×2, PNPM·Bun cache+거래 5m×1→5m×4, PNPM·Bun original 5m×1(시험 내부 4m 공유)→5m×3
(사례별 4m). 제품의 wall·resource·observer·cleanup 한도와 Linux job 150m은 바꾸지 않았다. CI470 원본에서 분할된 16개 하위
사례와 npm 4개 시험이 각각 정확히 한 번 PASS했고 `no tests to run`은 0건이다. 거래 시험은 사례별 TempDir, original 시험은
공유 cache를 쓰되 사례별 target 키로만 상태를 읽으며 사례 간 단정이 없어 분할로 사라진 상호작용은 없다(달라진 점은 뒤 사례가
앞 사례 cache를 보지 않는 것뿐이다). 이후 추가·개명된 사례가 조용히 빠지지 않도록 마지막 실행을 "이미 실행한 사례만
`-test.skip`하고 나머지 전부"로 바꿨다(로컬 probe로 의미 확인, `go test ./scripts/check` PASS). macOS 25m은 승인된 내부
build/test 한도와 별개인 macOS job 전체 예산 변경이다. CI467 macOS 시간 초과 원인은 여전히 UNKNOWN이다.

간헐 실패 보존: CI467 macOS default test 9m 초과, CI469 Quick PTY 준비 시간 초과·macOS DNS 시간 초과·Linux sink
`dropped_count`는 각각 원인 UNKNOWN이며 뒤 실행의 성공은 앞 실패의 해결 증거가 아니다. `dropped_count` 실행은 PNPM 거래
`single`의 세 번째 적용이 검사 단계(`inspect complete selected pnpm project`)에서 INCOMPLETE 오류로 끝나 승인 집합이 생기지
않았고 게시 기록도 없다(승격 차단 확인). Bun 시험 wrapper가 출력 oracle용으로 정확한 lifecycle 명령의 출력을 메모리에 받는 것은
시험 연결 보완이며 production의 출력 폐기 계약은 그대로다.

### CI471 — Bun 실행 CLI 프로세스 한도와 Linux job 예산

94a5b48의 [CI471](https://github.com/rahoney/heliopause/actions/runs/42437417300)은 GET5(03:31:16Z)에서 Linux·Required FAILURE, 나머지 8개
SUCCESS다. CI470의 schema 교정으로 Bun 선언 entry CLI가 PASS했고, 분할된 PNPM·Bun cache·거래·original 하위 사례도 모두 PASS했다.
실패는 Linux job 한도(당시 77분 시점)가 아니라 Bun run 블록 **프로세스** `-test.timeout=10m`이다. 5개 시험(선언 entry CLI
99.96s·두 workflow 47.93s/45.02s·RunCLI·거부)이 한 프로세스의 10m을 공유했고, RunCLI는 6m47s에 사례 1–2(typescript)를 끝내고
사례 3(`bunx rimraf@3.0.2`, 의존성 다수) 준비 중 gVisor bundle 검증을 실행하고 있었다(goroutine stack, 교착 아님). RunCLI는
CI446 76.76s·CI448 125.05s였다. 사례마다 시험 내부 4m context가 이미 있으므로 사례 한도는 바꾸지 않고, 같은 helper
(`testNodeExecutionCLIIntegration`)와 같은 package를 쓰는 Bun run·Bun program·PNPM run 블록의 CLI 시험만 각자 10m 프로세스로
분리했다(각 블록 10m×1→10m×2, `&&`로 연결해 앞 프로세스 실패를 가리지 않음). 출력이 파일에 모였다가 출력돼 사례별 시각은 없다.

전체 예산: 같은 시험 지점의 step 4 경과가 CI448 대비 26→36(PNPM cache)·32→46(PNPM original)·42→64(Bun original)·47→77분
(Bun 선언 entry CLI)으로 1.38–1.63×다. CI448 Linux job은 134분(step 4 101분)이었으므로 같은 비율이면 약 190–215분이다. 필수
내용 검사 추가 뒤의 전체 증가로 보고, 사용자 승인으로 Linux job 한도만 150→240분으로 올렸다(제품·개별 시험 한도 불변, 값을
고정하는 검증기는 없음). 다음 CI에서 단계별·전체 시간을 기록해 240분 적정성, 특정 단계의 비정상 지연, 중복 실행 여부를 본다.
계속 210분을 넘으면 독립 job 분할을 검토하고 추가 상향은 자동으로 하지 않는다. 원본은 `docs/github-logs/ci471-20261108`이다.

### CI472 — Linux 240분 첫 실행, Bun 거래 `peer` CPU 표본 지연(원인 UNKNOWN)

3447cd8의 [CI472](https://github.com/rahoney/heliopause/actions/runs/42453714452) attempt 1은 8개 SUCCESS, Linux·Required FAILURE다(step 4가 60분에
종료). `TestBunPublicProjectTransactionIntegration/peer`의 세 번째 적용이 Bun instance 검사 중 Python 관찰 CPU 표본 지연
(`gap_ns=12660981331`, poll 1s, 읽기 자체 7.9ms·직전 대기 12.65s)과 관찰 lifecycle stop bound 초과로 INCOMPLETE가 되어 승인 집합·게시
없이 실패했다(fail-closed 정상 동작). 같은 source의 같은 사례는 CI471에서 PASS했고 이번 변경은 이 경로를 바꾸지 않았다. 로그에 메모리
계측이 없어 runner 정지·메모리 압박 등 원인은 UNKNOWN이며, CI469 `dropped_count`와 같은 원인이라고 합치지 않는다. 가려진 뒤 구간을
확인하려고 실패 job만 재실행했다(재실행 성공은 이 실패의 해결 증거가 아니다). Attempt 2는 step 4 시작 약 2.5분 뒤 gVisor Bazel 공급의
`github.com/grpc/grpc` v1.75.0 다운로드가 504로 끝나 시험에 도달하지 못했으므로(exit 37) 검증 정보가 없고, attempt 3을 실행했다.

CI472 attempt 3(15:41–17:06Z, Linux·Required FAILURE, step 4 85분에 종료)은 attempt 1에서 실패한 `peer`를 포함해 분할 사례를 통과하고 Bun run
블록의 앞 프로세스(선언 entry CLI 103.21s·workflow 50.15s/46.86s·거부 239.19s)도 PASS했다. `TestLinuxBunRunExecutionCLIIntegration`이 488.82s에서
FAIL했다: 사례 1–2 PASS 뒤 사례 3(`bunx rimraf@3.0.2`, CI448 기준 12노드)이 실패했다. 로그의 결과는 시험이 앞 8192B만 남겨 부분 검사 노드 8개까지만 보인다(7개는
그래프 단위 `dependency-advisory`만 누락, 8번째 `wrappy@1.0.2`는 `bun-dynamic-lifecycle` INCOMPLETE). 처음에 적은 "12노드 중 11개 완료·마지막 노드 중단"은 잘린 출력에서 잘못 추정한 것이라 철회한다.
4m context 소진도 시간 계산에서 나온 추정일 뿐 미확인이다. fail-closed로 승격은 없다. 프로세스 분리(10m)는 작동했다.
CI448 대비 거부 시험(typescript 2건) 35.73→239.19s, 12노드 사례 >240s다. 코드상 `SourceAnalyzer.Analyze`가 노드마다 설치된 Semgrep core·rules를 다시 읽고 입력 volume을 새로 만들어
도입한다(`internal/sandbox/source_analysis.go:65,105`). 노드당 고정비 후보이나 실측은 없고 격리 경계라 변경하지 않았다.

### CI473 — Bun 실행 CLI 목표 경로 재현과 단계별 실측

cb88d21(임시 `[skip ci]` 진단, 준비 뒤 목표 시험만 실행)의 [CI473](https://github.com/rahoney/heliopause/actions/runs/42549208273)에서
`TestLinuxBunRunExecutionCLIIntegration`이 393.02s PASS했다. 사례 1 107.554s·사례 2 100.327s·사례 3(12노드) 185.112s이고 `context_err=<nil>`·`deadline_expired=false`다. 즉
사례 4m 만료는 이번에 재현되지 않았고 CI472 attempt 3의 실패 원인이 deadline이라는 직접 증거도 없다. 사용자 조건에 따라 시험 context 4→6분·프로세스 10→12분은 적용하지 않는다.
CI472 attempt 3은 job 후반의 같은 시험이 사례 1–2 합 약 248s(이번 208s, 약 1.2배, 사례 3 실패 시간까지 포함한 차분이므로 미확정)였다. 다음 전체 CI에서 재발하면 추가한 `node_execution_cli_timing` 줄과 원문 없는 요약이 만료 여부를 직접 보여 준다.

단계별(`docker events`, 시작→종료, 제품 변경 없음): typescript 사례의 Semgrep 스캔 81.7s·78.3s(사례 시간의 약 76%·78%). 12노드 사례는 Semgrep 세션 12개가 약 7.1s 간격으로 이어지며 volume 준비→스캔 시작 합 14.6s(세션당 약 1.2s), 스캔 합 48.7s(세션당 약 4s),
정리 합 1.1s, 동적 컨테이너 26개 실행 합 50.1s, 나머지는 세션 간 간격·resolver 등이다. 노드당 Semgrep 공급 재준비 비중은 volume 준비 약 1.2s와 세션 사이 미측정 구간(약 1.8s, 공급 읽기·해시·probe 포함)을 합쳐 상한 약 3s, 12노드 합 약 36s(사례 3의 약 19%)다. 비중이 작아 제품 공급 재사용 변경은 제안하지 않는다.
세션당 약 4s 스캔의 내부(규칙 적재 등)는 이 계측으로 나뉘지 않아 미확정이다. 큰 패키지의 스캔 시간이 지배적이며 이는 필수 검사 자체의 비용이다.

### CI474 — 사례 4분 만료 직접 확인과 조건부 시험 예산 적용

54016c8의 [CI474](https://github.com/rahoney/heliopause/actions/runs/42553746750) attempt 1은 Linux step 4 시작 22초에 `archive.ubuntu.com` 연결 실패(apt exit 100)로 시험 전 종료했고, 실패 job만 재실행했다.
Attempt 2는 step 4 88분에 `TestLinuxBunRunExecutionCLIIntegration`(510.08s)이 FAIL했다. 계측 줄이 사례 1 134.470s·사례 2 134.761s·사례 3 240.822s, `context_err=context deadline exceeded`·`context_cause` 동일·`deadline_expired=true`·`run_error_deadline=true`를 보여 사례 4m 만료가 직접 확인됐다.
결과 9437B 전체에서 부분 검사 노드 8개(7개는 그래프 단위 advisory만 누락, 8번째 `wrappy@1.0.2`는 `bun-dynamic-lifecycle` 진행 중)이고 12노드 중 나머지 4개는 동적 검사 전이다. 앞 노드가 순서대로 끝나고 있어 정체가 아닌 시간 부족으로 판단했다. CI473(job 초반) 대비 사례 1·2가 약 1.3배 느렸다.
사용자 조건부 승인에 따라 공용 helper 사례 context만 4→6분, Bun run·Bun program·PNPM run CLI 프로세스만 10→12분으로 올렸다(사례 합계 추정 약 134+134+310s에 준비·정리 여유). 제품 한도·검사 범위·거부 시험·앞 프로세스 10m·Linux job 240분은 유지한다. npm 프로세스는 기존 18m이다.

### CI475 — 조정 후보, 시험 전 Bazel 다운로드 504로 미검증

128779e의 [CI475](https://github.com/rahoney/heliopause/actions/runs/42565698323)는 8개 job SUCCESS, Linux·Required FAILURE다. Attempt 1–3 모두 시험 전 Bazel의 GitHub 다운로드가 `504 Gateway Time-out`으로 끝났다(`rules_java`, `bazel-skylib`, `protobuf`). CI472 attempt 2의 `grpc` 504를 포함하면 같은 유형이 4회이고,
CI474 attempt 1의 apt 연결 실패는 별개 유형이다. 재실행을 기본 해결책으로 쓰지 않으려고 attempt 3 뒤 중단했다.
따라서 사례 context 6분·프로세스 12분의 수정 후 실측과 Bun run 블록 뒤 Linux 구간은 미검증이다. CI474 attempt 2는 Bun run CLI 실패 시점이 step 4의 88분으로 CI448의 같은 지점(47.2분) 대비 약 1.9배여서 전체 시간이 240분에 근접하거나 넘을 위험이 있다(추정).

Bazel 504 분석(2026-11-09, 코드·workflow·릴리스 스크립트 불변): 고정 builder의 Bazel 8.3.1(lock sha512 일치)을 `--network none`으로 조회한 기본값은 `--experimental_repository_downloader_retries=5`, `--http_connector_attempts=8`, `--http_connector_retry_max_timeout=0s`이고 스크립트는 이 옵션들을 지정하지 않는다. 같은 값을 명시하는 변경은 효과가 없어 하지 않는다.
실패 4회는 모두 컴파일 전 `Computing main repo mapping` 단계이며, Bazel server 시작 뒤 32–65초에 URL마다 다른 archive에서 끝났다(`grpc`, `rules_java`, `bazel-skylib`, `protobuf`). 로그는 URL별 WARNING 1줄만 남기므로 실제 시도 횟수와 504를 낸 hop은 알 수 없다(UNKNOWN). 릴리스 스크립트의 `docker run`에는 proxy 설정이 없다.
로컬 재시도 경로는 `github.com` 302 → `release-assets.githubusercontent.com`(archive는 `codeload.github.com`) 200이고 protobuf·rules_java·skylib의 SHA-256이 BCR `source.json` integrity와 일치한다(local vantage는 hosted runner 경로의 대체 증거가 아니다).
GitHub status 공지와 시간·증상이 일치한다. [11-08 10:36–12:32 비인증 사용자 timeout](https://www.githubstatus.com/incidents/pyynvrm1vskw)은 CI472 attempt 2(10:46Z)와, [11-09 05:59 게시 anonymous traffic surge](https://www.githubstatus.com/incidents/c9px7cw2ghrs)는 CI475 attempt 1–3(05:23–05:56Z)과 겹치며 후자는 "CI jobs and release downloads may fail with a 504 … if they make anonymous calls"라고 명시한다. 앞서 05:53Z에 본 "status 정상"은 공지 게시(05:59Z) 이전 값이었다. 개별 요청의 직접 원인은 확정할 수 없어 "공지 장애와 일치, 요청별 경로 UNKNOWN"으로 둔다.
원본은 `docs/github-logs/ci475-20261109/recovery`다. 재실행 조건은 장애 종료 확인과 URL 재확인이고 허용 횟수는 1회다. 같은 504가 재발하면 중단하고 고정 integrity로 검증하는 사전 확보·`repository_cache`/`distdir` 최소안을 제시한다. 릴리스 스크립트 수정은 승인되지 않았다.

CI475 attempt 4(승인 1회, 09:21:58Z 시작): GitHub 장애 종료(09:09:56Z)와 URL 3회 확인(12건 200, 재시도 대상 3개 integrity 일치) 뒤 실행했고 504는 0건이다. Linux는 104.3분에 `TestLinuxUVToolNativeAdoptionCLIIntegration`에서 실패했다. Required는 이 실패를 따라 실패다. 원본은 `docs/github-logs/ci475-20261109/attempt4`다.
12노드 한도 실측: Bun 239.177s, Bun program 240.488s 모두 6분 안에 통과했다(`deadline_expired=false`). 프로세스는 478.97s/483.79s로 12분 안이다. Bun program 240.488s는 이전 4분 한도를 넘는 값이라 조정 근거를 뒷받침한다. 다른 사례는 119–122s다.
240분 예산은 미검증이다. Bun RunCLI 종료 지점이 81.3분으로 CI448의 47.2분 대비 1.72배이며, 같은 배율이면 전체는 약 231분이다(추정). 210분 초과 시 job 분할을 검토하고 한도는 올리지 않는다.
실패 원인: 시험이 Evidence root를 0500으로 바꿔 쓰기 실패를 주입하는데, `c573120`(11-02)부터 공유 저장소가 첫 intake 예약에서 root를 먼저 검사해 원본 wheel 획득 단계에서 거부한다. 로컬 재현 결과는 `storage child is not an owned private directory`다. selectOwnership이 이를 `UPDATE_ORIGINAL_STATE_UNSUPPORTED`로 감싸 hosted 로그의 내부 문구는 가려졌으나, 두 observer 세션 뒤 첫 예약이라는 순서가 일치한다. 원본은 불변이었고 fail-closed 자체는 정상이며 ledger도 남지 않았다. 앞선 단계 실패에 가려 이번에 처음 드러났다.
교정: `TestLinuxUVToolNativeAdoptionCLIIntegration`과 pipx 공용 helper에서 결함 주입만 실제 Evidence quota 소진(256MiB sparse 미등록 run)으로 바꿨다. 소유권 증명과 검사를 지난 뒤 실제 Store 기록이 거부되고, 원본 불변·ledger 없음 단정은 유지한다. 임시 검증(실제 `Store`+`Local`)은 intake 허용, `evidence retention quota is full`, temp/run 미잔존, filler 제거 뒤 기록 성공이다. 같은 유형 sweep에서 Evidence root 권한 변경 시험은 이 1개뿐이었다. 제품 코드는 불변이다. hosted 경로의 교정 결과는 미검증이다.

