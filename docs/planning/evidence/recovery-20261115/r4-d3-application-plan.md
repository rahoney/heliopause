# R4 D3 강제 종료 복구 적용안

복구 브랜치의 진입점은 `1b894562f533285ee9a9cfd339eea5abbe84f9a7` / tree `dc489bdb206dddaec8620678054ea9bfefa07772`다. [R4 실제 통합](./r4-native-20261118.md)의 정상·거부·관찰 미완료·SIGINT를 보존한다. 이 절의 D3 기준 상태는 **IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO**였다. 현재 적용과 수용 상태는 아래 실행 기록을 따른다. 승인된 §6b 조건을 충족하는 구현을 제안한다. [기존 U1 독립 검토](./r4-d3-independent-review-20261118.md)의 REVISE를 유지한다.

## 성공 코드·원본·현재 차이

| 요구 | 마지막 실제 증거와 소스 | 현재와 처분 |
| --- | --- | --- |
| 정상 종료·동시 소유자 | `358adf9`의 strict supervisor, R4의 정상·관찰 거부·취소 실물. 현재 R4의 기존 supervisor·정책·회계 시험 부모 14개, 상위/하위 결과 21개 통과 | strict lock, endpoint inode, 기존 helper 종료 및 정책을 유지한다. 최초 단위 시험은 sandbox의 root 소유 표시 차이로 일부 실패·SKIP했다. 고정 Go1.27.4와 기존 보호 런타임 경로를 일반 사용자로 사용한 뒤 실패·SKIP 범위가 통과했다. |
| CLI·observer 상실 시 비신뢰 실행 중단 | 2026-11-12 원본 `kill-trials.log` 7–37행은 resolver 단계의 소유 실행 종료·재시작, 두 번째 요청 거부를 보여준다. 48–53행의 수정된 관찰 단계는 실행 중단 0.6172초와 Docker 소유 자원 0을 보였으나 slice 1개와 재실행 exit 255가 남았다. | U1의 이미 성공한 peer 연결 종료와 소유권 아이디어를 검토해 재사용한다. RAM lease 삭제·두 차례의 빈 조회·옛 PID/이름만 사용한 재승인·일괄 정리는 적용 대상에서 제외한다. |
| helper 상실·재시작·늦은 생성 | 현재 `hosttool/observation_resource_linux.go`의 `terminate`는 등록된 scope와 Docker ID의 독립 관측을 요구한다. `systemdObservationKeeper.Close`는 keeper/slice 해제를 확인한다. 현재 helper 재기동 시 RAM 세션은 새로 만들어진다. | 위 종료·검증 의미를 유지한다. 직전 신원과 아직 답을 받지 못한 작업을 실행 승인 전에 기록하고, 재시작 때 그 기록을 사용한다. helper 상실과 늦은 생성 검증은 OPEN이다. |

기존 검사 명령과 결과는 `/tmp/haa-r4-d3-fOrC2K2B/baseline-summary.json`과 함께 있다. 당시 근거는 원개발 작업 트리의 `docs/github-logs/ci479-20261110/analysis/local-20261112/kill-trials.log`와 오답노트 sandbox L08·L12다. 21번 설계 §7.1은 “외부 종료·observer drain 확인 뒤 owned runtime 자원/더미 삭제”, §7.3은 “PID 부재·오래된 mtime만으로 stale을 확정하지” 말고 UNKNOWN에서 삭제를 보류하도록 규정한다.

## 제안하는 적용 범위

1. **생성 승인 전 소유 상태.** 기존 root helper가 두 소켓(네트워크 정책·관찰 자원)을 제공하기 전에 하나의 복구 상태를 읽는다. 현재 인증은 kernel UID와 보호 실행 파일을 확인한다. D3는 그 위에 정확한 프로세스 신원과 작업 세대, 연결 수명을 결합해 같은 UID의 서로 다른 실행도 구분한다. Docker 자원·keeper·방화벽 변경 전에 보호 저장소에 요청 신원, 예정 자원, 작업 세대 및 필요한 canonical 정책과 방화벽 식별자를 기록하고 동기화한다. 예정 자원·확인한 실제 ID·응답 유실을 구분한다. 저장 실패와 기존 기록 없음·깨짐·이전 형식의 상태도 구분한다. 비어 있다는 이유로 초기화하거나 해제하지 않는다. 저장 방식은 기존 protected journal 검증에 맞추고 순환 의존을 추가하지 않는다.
2. **시작 전 신원과 시작 후 CPU 회계를 분리.** 기존 Python·Node의 `Register`는 시작한 컨테이너의 PID와 exact cgroup을 확인하므로 그대로 유지한다. 시작 전에는 별도로 생성된 정확한 Docker ID, 해당 작업 소유 신원과 시작 의도를 확인한다. root helper의 같은 상태 아래에서 이 확인과 작업 진행 표식을 기록하고, 기존 Executor의 Docker 호출과 responder `start --attach`에 연결한다. 회수 전환과 새 요청 승인은 직렬화한다. 이미 발급한 요청에는 대상 ID와 완료 여부를 남긴다. client 종료/취소가 Docker의 진행 중 작업 종료를 증명하지 않는다. 늦은 create는 시작 전 신원 확인 없이는 실행할 수 없어야 한다. 이미 시작/exec를 요청한 ID는 실제 runtime 및 cgroup 중단을 확인하고, 늦은 start가 동일 대상을 다시 시작할 수 없는 외부 근거를 요구한다. 근거가 없으면 기록을 남기고 다음 작업을 거부한다. 시작 승인을 단순히 Docker 실행 직전 조회하는 구현은 회수 경쟁을 닫지 못한다.
3. **독립 종료와 시작 전 복구.** 살아 있는 helper는 U1에서 효과를 보인 연결 종료 감지를 재사용하고, 기존 supervisor는 관찰 연결 소실을 미완료로 처리한다. helper 자체가 종료되면 기존 systemd 단위에서 동일 보호 바이너리의 고정 인자 복구 동작을 실행하는 안을 검증한다. 첫 동작은 정확히 소유한 실행을 중단시키는 것이며, 이어지는 재시작 검증에서 남은 자원의 종료·회수·불명 상태를 해소한다. 두 RPC listener의 새 요청은 일치하는 소유 기록과 독립 자원 확인이 끝날 때까지 받지 않는다. native 단위의 15초와 기존 observer 종료 5초·관찰 stop 7초는 각각 유지한다. 15초 안에 전체 회수를 직렬로 완료한다고 가정하지 않고, 같은 한도에서 즉시 실행 중단을 실제 확인한다. 입증할 수 없으면 D3 완료를 보류한다.
4. **독립 확인 뒤 기록 해제.** 기존 Docker ID/cgroup drain·keeper 종료·방화벽 정책 owner·endpoint inode 검증을 사용한다. firewall 조회 실패나 RAM 기록 부재는 UNKNOWN이다. 오래된 소켓은 세대·inode와 해당 listener 종료를 확인한 뒤 실제 소켓 종류에 맞게 다룬다. 기록이 바뀌었거나 DGRAM/SEQPACKET 신원을 밝힐 수 없으면 유지한다. 정상 release와 재시작 모두 동일한 외부 근거를 요구한다. 최초 작업 오류와 후속 회수 오류를 함께 보존한다.

변경 책임은 기존 `bootstrap.Run`, `hosttool`의 authenticated helper/Executor/정책·관찰 자원 관리, `sandbox.ObserverSupervisor`, 보호 helper 실행과 systemd 단위의 원본·검증자에 한정한다. 기존 R4를 되돌리거나 U1을 전부 생략하면 현재 시작 거부를 보존하며 자동 중단·복구는 OPEN으로 남는다.

## 보호 단위와 적용 경계

- 실제 개발 helper는 `/etc/systemd/system/haa-network-policy-helper.service`에서 현재 실행 중이다. 확인한 값은 `Restart=on-failure`, stop 90초, `RuntimeDirectoryPreserve=no`다. 이전 R4 통합이 사용한 환경이다. 현재 `/var/lib/heliopause`는 없었다.
- 복구 소스의 native unit은 `internal/releaseactivation/config_linux.go:nativePolicyUnit`에서 `Restart=no`, stop 15초, `RuntimeDirectoryPreserve=yes`로 생성한다. `releaseactivation`은 업데이트·원복에서 이전·후보 unit의 정확한 바이트를 비교한다. 기존과 D3 후보를 등록된 버전·검증된 파일로 구분하고, 업그레이드 및 rollback에서 서로의 unit을 허용하지 않는 조건을 확인한다. OS package unit은 별도 생성자와 이전 규칙이 있으므로 이 변경의 적용 경로를 명시한다.
- 제품 코드 작업은 first-stop/recovery와 전환 검증까지 같은 계약으로 묶는다. 현재 개발 단위를 바꾸는 통합 창은 exact 보호 경로·소유 기록 위치·원복·격리가 확보된 별도 실행 범위다. 먼저 제품 소스와 시험 후보를 고정한다.

## 같은 후보의 확인 순서

| 검증 | 통과 기준 |
| --- | --- |
| 단위·Race | 보호 기록 쓰기·짧은 쓰기·sync·부분 획득, helper 연결 종료/재시작, 권한·세대·PID/inode 교체, 회수 중 동시 요청·늦은 생성, Docker/keeper/정책 조회 오류. 실패 시 새 시작 시도 0 또는 이미 승인된 정확 자원 종료, 기록 유지와 재진입 거부. native 단위의 업그레이드·rollback 원본 바이트 및 새로운/이전 형식 교차 거부 포함. |
| 기존 정상·거부 | strict supervisor와 정상 종료·취소·관찰 미완료의 실제 영향 경로. 정상 대상 설치·승격, 거부에서 대상 불변과 ALLOW 없음. |
| 실제 gVisor/Host 통합 | 기존 로컬 고정 입력과 전용 fixture. 정상 packaging 대조 후 CLI·observer·helper의 개별·동시 중단/재시작을 확인한다. resolver, 관찰, build의 영향받은 호출과 동시 정상 작업을 확인한다. 새 요청 전 실행 중단 및 exact 컨테이너/volume/network·방화벽·keeper·slice·socket·lease·대상과 관찰 결과를 대조한다. 다음 요청은 이전 소유권 해제를 확인한 뒤 정상 실행된다. 정리 불확실 시 거부한다. |
| 종료 | Queue·source/tree·입력·명령·결과·시간/자원·독립 원복을 기록한다. 실패·skip·미도달을 구분한다. 첫 실패는 기준선·오답노트 대조 뒤 한도를 유지하며 판단한다. 필수 검사와 승인된 영향 기반 K를 적용한다. |

## 독립 검토와 적용 승인

**D5 재검토: 구현 방향 승인 가능.** 초기 REVISE의 시작 전 신원 확인/시작 후 CPU 회계 분리, 두 helper 소켓의 공통 선행조건, native unit 보호 전환, 실제 소켓 종류를 반영했다. 구현 후 실제 시험에서 **시작 요청과 CPU 등록 사이에 CLI·helper가 종료되는 경우**를 확인한다. 관찰 자원 Register를 쓰지 않는 resolver도 같은 실제 시작·정지 경계를 입증해야 한다. 이 구간에서 늦은 실행이 일어나지 않거나 정확히 중단되고, 독립 확인 전 다음 요청이 차단돼야 한다.

2026-11-18 사용자가 복구 worktree의 제품 소스와 표적 시험 반영을 지시했다. 승인 전의 실제 개발 Host와 CI 보호 경계는 유지한다.

## 제품 소스 반영과 남은 수용 조건

동결 후보 `8fee77c6a6c29b1b65a6c5e52a3aad9f8917636d`는 기존 보호 CLI·helper·Docker 실행을 하나의 인증 lease에 연결한다. cgroup·Docker 전체 ID·observer 신원을 저장한 후 시작을 승인한다. 관찰 회계 Register의 기존 의미는 보존했다. 종료 전에 Docker 조회 없이 기록된 scope를 중단할 수 있도록 순서를 조정했고, listener inode는 고정 경로에 공개하기 전에 동기화했다. 승격의 기존 오프라인 인자는 유지하며 Docker create와 start를 나눴다. 기존 통합 fixture는 동일 실행 조합으로 연결했다.

- 후보 `e815adf3`의 D5 REVISE에서 확인한 observer 사망 직전 승인된 실행, observer 기동 실패, 방화벽 jump 이후·keeper 종료 이후의 재시작, 보호 세션 없이 helper를 호출하던 기존 통합 fixture를 수정했다. 마지막 표적 compile에서 Python 경계 fixture의 가려진 context와 외부 Go fixture의 test-only export를 각각 확인해 고쳤다. 정상 제품의 시간 한도를 바꾸지 않았다.
- **직전 설치본이 제어하는 첫 self-update는 별도 확인이다.** 이전 updater는 새 helper 경로에는 이전 단위 형식만 렌더링한다. 보호된 새 helper는 관리 단위에 복구 종료 동작이 없으면 요청을 받기 전에 거부한다. 구·신 단위의 형식·바이트 검사는 유지하며, 이전 updater로부터의 안전한 전환 및 실제 롤백 검증은 OPEN이다.
- 구현 연결은 후보에 있다. CLI/observer/helper 강제 종료, 중단 당시 미결 Docker create/start, 실제 방화벽·keeper 잔류 및 재시작의 독립 확인은 QUALIFIED/ACCEPTANCE_CLOSED=NO다. 실제 미결 상태는 증명 전 재진입 차단 대상으로 남는다.

최종 로컬 결과: canonical rebuild-step/quick/Staticcheck/default 593.64초 및 보안 48.27초 PASS, 제거 0개, 추가 98개 (목록: 아래 3·5와 함께 확인). 관련 7패키지 Race는 2025개 PASS, 0개 FAIL, 129개 SKIP이다. 신규 D3 선택 시험은 모두 실행됐다. CLI·helper·responder와 세 realcorpus 시험 바이너리의 해시를 기록했다. 임시 unit 사본은 systemd-analyze verify exit 0이다. 로컬 증거는 `/tmp/haa-r4-d3-verified-ewtCGWJD/qualification-summary.json`에 있다.

### 실제 Host 검증에 추가로 필요한 범위

1. 이전 R4 11경로 임시 전환·원복 구성을 재사용한다. 추가 변경은 보호 서비스 단위 `/etc/systemd/system/haa-network-policy-helper.service`와 D3 전용 원장 `/var/lib/heliopause-recovery`다. 실제 Host의 서비스는 `root:root 0644`이고 현재 ExecStopPost가 없다. D3 디렉터리도 없다. Docker는 systemd cgroup v2다.
2. 원래 unit 바이트·권한·서비스 상태·기존 보호 파일과 Docker/방화벽 상태를 보존한다. 일반 사용자로 고정 환경과 dummy fixture를 사용한다. 정상 packaging 설치부터 확인한 뒤 CLI·observer·helper 종료 및 재시작, 승인→시작/CPU 등록 사이의 지연, resolver와 정리 재개를 시험한다. 원래 실행 한도를 유지한다.
3. 검증 중 실제 작업을 먼저 중단하고, 새 런타임 프로세스·cgroup·keeper와 Docker/방화벽·동시 기존 owner를 독립 대조한다. 결과 확인 후 원래 서비스 바이트·보호 파일·상태를 복원한다. 자원이 불확실할 때는 현재 승인을 허용하는 원 상태로 전환하지 말고, 안전하게 독립 수행할 수 있는 정리와 정확한 blocker 기록을 진행한다. 전체 원장 삭제를 복구 근거로 삼지 않는다.


## 개발 Host의 최초 두 검증 창과 updater 회귀 (2026-11-18)

| 후보와 시험 | 실측 결과 | 처분 |
| --- | --- | --- |
| `8fee77c6` / 개발 helper unit에 고정 `ExecStopPost` 추가 | 기존 `ProtectSystem=strict`와 `/run/heliopause-network-policy` 한정 쓰기로 원장 mkdir이 EROFS. 정상 설치 및 강제 종료 시험 미도달. 최초 자동 원복은 원복 도구의 후보 unit SHA 허용 누락으로 중단. | 정확한 승인 unit SHA를 허용하도록 원복 검증만 교정. 작업·원장 부재를 확인한 뒤 원본 14개·응답기 inode·helper·기존 네트워크·방화벽 원복을 독립 확인. 원본 `/tmp/haa-r4-d3-native-AJITEj52/helper-first-attempt.journal`, `activate.stderr`, `restore-ok.json`. |
| `802eca42` / 동일 제한 + `StateDirectory=heliopause-recovery`, mode `0700` | helper와 리스너 두 개가 기동. 기존 실제 `packaging@25.0` 정상 설치가 1.656초에 `observation resource response is untrusted`로 거부. Docker 실행 0, 대상 불변. 기존 capability만 적용한 읽기 전용 대조에서 observer의 사용자 소유 `0700` 경로 아래 `stat`은 EACCES. 일반 root에서는 충돌할 파일이 없어 ENOENT. 같은 제한에서는 새로 띄운 시험 계정 프로세스의 신호 확인도 EPERM. | 개발·OS package·native unit에서 전용 상태 디렉터리 provision과 root-owned 하위 원장을 일치시킴. 추가 capability·observer 쓰기 경로는 적용 승인 대기. 두 번째 창도 원복. 증거 `/tmp/haa-d3nr-pxKEKg5g/{result.json,normal.stderr,observer-capability-probe.log,independent-after.json}`. |
| 임시 mount namespace의 순수 회수 경계 | 별도 bind mount된 두 디렉터리 사이 rename이 `EXDEV(18)`을 반환. 같은 mount에서는 성공. 개발 단위에서 두 경로가 각각 mount되면 현재 observer→정책 디렉터리 격리 이동(`removeD3ObserverEndpoint`)을 그대로 쓸 수 없다. | 실물 재시도 전에 정확한 mount와 inode를 확인하는 같은 mount 회수 필요. 양쪽 mount 예외를 설정한 것만으로 강제 종료 회수가 성립한다고 기록하지 않는다. `/tmp/haa-d3nr-pxKEKg5g/isolated-mount-rename-probe.log`. |
| `76c08dc6` / 기존 updater의 로컬 격리 namespace 시험 | 초기 파일 준비·등록·트랜잭션 세 부모, 76개 하위 PASS. 일반 releaseactivation Race 117 PASS·0 FAIL·11 SKIP. 두 namespace wrapper 오류의 원본을 보존. 트랜잭션은 실제 root 파일 복사·경로/설정/원장 전환·롤백을 확인하며 서명·old peer·systemd·호환성 health는 대역. | `1b89456`에 추가된 설치 시 고정 responder 요구를 기존 `operationReleaseFixture`에서 누락했다. 테스트 전용 role과 digest를 추가하고 기존 `BindCurrentRuntime`으로 SBOM과 manifest를 연결했다. 기존 거부 기대값 유지. 근거 `/tmp/haa-r4-d3-updater-fixture-GGub3KHI/{transaction-namespace.log,transaction-namespace-final.log,releaseactivation-race.jsonl}`. |

이전 성공 R4 `1b89456`의 unit은 기존 observer가 직접 자기 디렉터리를 관리했기에 동일 제한에서 정상 설치를 실측했다. D3 `d3_observer_linux.go`는 새 root helper가 정확한 observer PID·endpoint와 소유 파일을 사전 기록하고 강제 복구하게 하므로 조회·종료·회수 권한을 따로 검증해야 한다. 서비스 구문 검사만으로 파일시스템과 UID 경계를 검증했다고 볼 수 없다. 상태 provisioning은 동일 개발 Host에서 기동·리스너까지 검증됐다.

**R4/D3 수용 보류.** 원래 Host를 활성 상태로 복구했다. 실물 정상 설치와 승격, resolver·관찰·build의 강제 종료, 동시·지연 생성과 회수·재시작은 새 후보에서 미도달이다. 직전 updater `1b89456:internal/releaseactivation/config_linux.go`는 새 version 경로에도 구 단위를 렌더링한다. 후보 helper `verifyD3Unit`은 해당 관리 단위의 고정 종료 동작이 없으면 원장 및 listener 준비 전 종료한다. 서명된 실제 구→신 전환 또는 실행 진입 전에 보호된 새 단위로 전환하는 승인 절차와 실패 롤백이 남았다. R5 진입은 R4 통과 뒤다.


## Observer 권한·볼륨 소유권 교정과 실제 packaging 검증

원래의 `ProtectSystem`과 observer 디렉터리 mode를 유지했다. 임시 개발 Host unit에만 `CAP_DAC_OVERRIDE`, `CAP_KILL`과 `/run/heliopause-observer` 쓰기 범위를 추가했다. observer endpoint는 같은 mount에서 mode `0700`으로 연 임시 디렉터리의 fd에 `RENAME_NOREPLACE`로 옮긴 뒤 기록된 inode만 회수한다. 다른 inode는 보존하고 거부하는 단위 회귀를 추가했다.

- `a8195859`에서 기존 `1b89456`의 Python resolver 실물 경로까지 다시 도달했다. 응답기 volume에 새 실행 owner 라벨이 추가되자 기존 두 라벨 검증이 거부했다. 정상 설치 16.274초/exit10, stager 없음, volume create/destroy 및 원복 확인. `/tmp/haa-d3win-JgWg1bR3/{normal-results.json,responder-volume-events.jsonl}`.
- `5290b1f9`에서 라벨 exact 비교는 단위 시험을 통과했지만 production `admissionAwareCommandRunner`가 owner 정보를 전달하지 않았다. 17.672초/exit10, stager 미도달, 원복 확인. `026e2d12`는 기존 wrapper에서 정확한 실행 owner를 전달하고 같은 wrapper로 정상/누락/다른 owner/추가 라벨을 각각 검증한다. 새 후보의 표적 Race 26 PASS, FAIL/SKIP 0 및 rebuild-step PASS. `/tmp/haa-d3lab-kRClVmxP/{result.json,independent-after.json}`, `/tmp/haa-r4-d3-forward-candidate-mwTShhvK/target-qualification.json`.

| `026e2d12` 실제 gVisor 시험 | 결과 |
| --- | --- |
| 최초 정상 설치 | 35.587초. 필수 검사 3개 완료, Evidence 5개, ALLOW, 승격, packaging 25.0 import와 작업 정리 확인. |
| observer SIGKILL과 다음 설치 | 29.195초/exit10, 필수 동적 검사 미완료, 대상 불변, ALLOW·승격 없음. 다음 설치 34.201초 PASS. |
| CLI SIGKILL과 다음 설치 | 25.935초/exit137, 대상 불변, ALLOW·승격 없음. 원장·작업 자원 정리 뒤 다음 설치 33.158초 PASS. |
| helper SIGKILL과 다음 설치 | 첫 창은 회수 뒤 새 helper가 준비되기 전에 다음 CLI를 시작해 `observation resource service unavailable`로 거부됐다. 두 번째 창에서 종료 사건 후 5.412초에 새 PID·재시작 1회·정확한 listener inode 2개를 확인했다. 강제 종료 시 23.749초/exit1과 대상 불변, ALLOW·승격 없음. 재기동 확인 뒤 다음 설치 30.533초 PASS. |

두 helper 강제 종료에서 `ExecStopPost`가 먼저 `previous protected execution could not be confirmed stopped and recovered`로 exit1을 반환했다. 별도 확인에서 두 경우 모두 owner 해제·listener 0·소유 자원 부재를 확인했다. 첫 창의 다음 정상 설치는 재기동 전에 실행돼 거부됐다. 재기동 확인을 추가한 두 번째 창에서는 새 PID와 보호 원장에 연결된 listener 2개를 확인한 뒤 설치가 통과했다. 초기 훅 오류의 세부 지점은 미확정이다. 원본 `/tmp/haa-d3fwd-06bIv7Y1/{result.json,helper-restart.journal}`, `/tmp/haa-d3restart-87FWPPtH/{result.json,helper-readiness.json,helper-restart.journal,independent-after.json}`. 두 임시 창에서 원본 14파일·응답기 inode·helper·원래 network와 방화벽·개발 Git과 SDK가 복구됐고 새 container와 volume은 0이었다.

**범위:** 이 결과는 packaging 관찰 경로의 정상 4종(최초·세 종료 뒤)과 종료 거부 3종이다. resolver/build 중 종료, 지연 생성/정리 불확실성/동시 사용자 실물, 종료 훅의 초기 오류 세부 원인과 실제 서명 updater의 첫 전환·롤백은 OPEN이다. R4 수용 완료와 R5 진입은 기록하지 않는다. 임시 설정을 실제 배포 단위의 승인으로 사용하지 않는다.


## 기존 성공 증거 재사용과 resolver 종료 실측

23번 계획 §6b와 §8.3은 R4에서 관리 helper의 실제 종료·재시작과 작업 소유권별 회수를 요구한다. 위 적용안에서 함께 추적한 **이전 공식 서명 설치본의 새 unit으로 첫 전환·롤백**은 출시 호환성 OPEN으로 유지한다. 초기 파일 준비·등록·트랜잭션의 기존 로컬 회귀를 재사용하고, 서명 업데이트 전체를 R4 종료 관문으로 추가하지 않는다.

2026-11-12 `kill-trials.log`의 기존 정상 packaging resolver와 CLI 종료 조건을 그대로 사용했다. 후보 `026e2d12`의 소스·바이너리·필수 검사·예산은 변경하지 않았다. 기존 관찰 단계의 7종 통과를 재사용하고 resolver 구간만 새로 확인했다. 고정 `packaging==25.0`의 실제 resolver Docker `exec_start` 뒤 정확한 CLI PID에 SIGKILL을 보냈다. 10.807초/exit137, 설치 대상 불변, 승격·ALLOW·후속 분석 컨테이너 없음, 소유 자원 회수. 다음 정상 설치는 33.072초에 필수 검사·ALLOW·승격·import를 통과했다. 창 66.289초, client RSS 최대 137105408B. 원래 Host 14개와 helper, 기존 네트워크·방화벽 복구, 신규 컨테이너·볼륨 0 확인. 원본 `/tmp/haa-d3resolver-lLFkbdws/{result.json,resolver-cli-kill-owned-recovery.json,independent-after.json}`.

앞선 helper 종료 훅 두 번은 새 프로세스가 뜨기 전에 원장 owner·리스너가 이미 비워졌다. `RecoverProtectedExecution`은 초기 중단 오류와 최종 회수 결과를 함께 반환한다. 원장 갱신 순서상 두 기록은 최종 회수 실패를 가리키지 않는다. 첫 중단의 상세 지점은 원본에 없으므로 원인을 특정하지 않는다. R4는 §6b의 build·동시 소유·지연/불확실 생성과 이 첫 오류 해석이 남아 있다.
