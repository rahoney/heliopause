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
