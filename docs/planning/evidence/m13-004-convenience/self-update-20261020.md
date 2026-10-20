# M13-004 자체 업데이트 — 소스 변경 전 감사·실행 계획·재검토

기준: `milestone/m13-release-build-packages`,
`fd93fad0345544f7268350fdad25422cdb8d934f`.
tracked/index clean에서 조사했다. 기존 미추적 사용자 자료·raw·lessons는 별도다.
이 문서는 구현·qualification의 증거가 아니다. [제품 설계 제안](../../../user-journey-cli-ia/07-self-update.md)이
사용자 흐름과 보안 요구를 소유한다. 과거 M10/M13 acceptance는 재개방하지 않는다.

## 1. Baseline audit

YES는 아래에 명시한 기존 scope에만 적용한다. 전체 경로의 일부가 구현됐다고
새 self-update의 WIRED/QUALIFIED를 YES로 올리지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 기존 owner와 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| verifier-gated 로컬 사용자 설치·atomic pointer 복원 | YES | YES, 기존 private caller | YES, 기존 회귀 | YES, M10 범위 | `internal/releaseinstall/installer.go`: Installer.Install/activate/activateCurrentWithSync; installer_test.go의 기존8개 검사. 교체하지 않음. public caller·동시 transaction·검역 결과 binding·doctor 전 완료 보류는 별도 gap |
| installed user release doctor | YES | YES | YES, 기존 범위 | YES, M10 범위 | `internal/releaseinstall/doctor.go`: DefaultRoot/CheckInstallation. user-owned release만 검사하며 protected runtime doctor와 다름 |
| exact release asset/manifest·보호된 build/publish 소비자 | YES | YES | YES, 기존 candidate 범위 | YES, 기존 M13-003 범위 | `scripts/release-manifest/main.go`: assetSpecs/buildRelease, 기존 release workflows. 공개 배포 실행·runsc bundle 존재의 증거는 아님 |
| 일반 아티팩트 검역과 여섯 생태계 update | YES | YES | YES, SEC-UPD-001 | YES, 기존 범위 | `internal/application/inspect.go`: InspectService.Inspect, 기존 provider/Policy/Evidence. 본체 updater로 다시 구현하지 않음 |
| 설치된 본체의 공식 release 획득·production attestation verifier | NO | NO | NO | NO | `releaseinstall.BundleVerifier`는 계약이고 production caller가 없음. GitHub release declared digest provider는 OIDC attestation verifier가 아님 |
| 본체 release 전체를 frozen bytes로 검역한 뒤 활성화하는 orchestration | NO | NO | NO | NO | 기존 InspectService 재사용 필요. manifest/provenance·각 executable 검사·모든 ALLOW·activation set 동일성 연결이 없음 |
| protected CLI/helper 세트 활성화·drain·인증 전환·doctor·재부팅 복구 | NO | NO | NO | NO | user installer는 root 경로를 변경하지 않음. `hosttool.PolicyPeerAuthorizer.AuthorizePeer`는 kernel credential과 protected 실행 파일 inode/digest를 확인. 새 파일 설치만으로 peer가 바뀌지 않음 |
| 채널 wrapper·공식 이름·publisher binding | NO | NO | NO | NO | npm/PyPI 계정 rahoney 준비 완료. 이름·Homebrew tap/formula·실제 publisher binding은 미확정. 계정 가입을 다시 요구하지 않음 |

신규 자체 업데이트 전체: IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO. 소스fd93fad의 CI353 성공은 기존 작업의 완료이며 새 기능의
구현·설치 반영·공개 배포를 입증하지 않는다.

## 2. 변경 위치와 방법

아래는 승인 후 현재 gap에만 적용할 edit map이다. 미정 역할의 placeholder나 파일을
미리 생성하지 않는다. 기존 구현 대체·재작성은 계획하지 않는다.

| 위치 | 재사용/정밀 변경 | 관련 acceptance |
| --- | --- | --- |
| `internal/artifact/githubrelease/client.go` / 기존 intake storage | exact tag/asset ID·bounded HTTPS·credential 차단 재사용. release bundle 획득은 identity를 한 번 고정한 사본으로 전달; generic adapter의 임의 repo 기능을 본체 updater 권한으로 노출하지 않음 | A1/A2 |
| `internal/verification` / `releaseinstall.BundleVerifier` caller | 공식 source/workflow/tag/commit attestation verifier를 실제 연결. trusted verifier·offline root의 pin/rotation·출력 binding 확정 후 의존성 추가. subprocess라면 protected binary/minimal env/정확한 인자와 결과 검증; library라면 upstream 검토·pin·취약점 검사. verifier 선택은 아직 미완료 | A2 |
| `internal/application/inspect.go` / `internal/core/ports/ports.go` | 기존 InspectService와 Artifact/Verification/Inspection/Evidence/Policy 계약 재사용. 본체 use case는 생태계 중립 typed release/승인 세트를 전달하고, 고정 intake adapter로 동일 획득 바이트를 검사. Application에서 filesystem·GitHub SDK·Cobra 참조 금지 | A3/A4 |
| `internal/inspection/githubrelease/dynamic.go` / 기존 static inspector / `internal/sandbox/github_elf.go` | 기존 ELF bounded 검사·관찰·unsupported/cleanup 경계 재사용. CLI뿐 아니라 선택 executable 전체 검사. helper의 host 권한 필요는 격리 실험 성공 보장으로 바꾸지 않음. 새 required probe가 필요하면 해당 owner의 coverage 계약부터 정밀 보완 | A3/A5 |
| `internal/releaseinstall/installer.go` / installer_test.go | 기존 manifest/asset rehash·versioned stage·pointer/rollback 유지. 새 activation owner와 공통 검증 primitive를 재사용할 수 있는지 먼저 확인. user installer를 root installer로 묵시 변경하거나 현재 private API를 CLI path/boolean 신뢰로 노출하지 않음 | A4/A6 |
| 별도 설치 전용 privileged owner — 승인 전 파일 없음 | 고정 operation·authenticated old controller·root stage·identity rehash·drain·lock/journal·세트 전환·doctor·복원. 기존 `hosttool/policy_peer_linux.go`의 신원 확인 방식을 검토 재사용하되 network helper에 installer API 추가 금지. root 동작/IPC/보안 리뷰를 먼저 별도 구체화 | A4/A6/A7 |
| `internal/bootstrap` / `internal/cli` / 기존 presentation catalog | composition과 명령 입력/결과만 연결. 실제 신규 서비스가 준비되기 전 성공 stub/메뉴·명령을 만들지 않음. 일곱 언어·progress·JSON 재사용; source별 일반 update 입력 변경 금지 | A5/A8 |
| `scripts/release-manifest` / release-packages / 기존 release validators/workflows | 실제 추가 배포 payload가 필요한 경우에만 exact asset set·SBOM·digest·attestation·검사 소비자를 같은 변경에서 확장. 초기 same-runtime scope에서는 누락 runsc asset을 있다고 가정하지 않고 lock 변경 거부. `.deb`/`.rpm` 설치는 package DB와 함께 qualification할 때만 지원 | A1/A6/A9 |

## 3. 순서·착수 조건

1. 현재 audit와 제품 설계, 이 edit map을 작성하고 아래 재검토를 수행한다.
   Queue 상단·상세·README·PROJECT·router를 일치시킨다. 문서 검사를 실행하고
   검증한 문서 후보를 의미 있는 한글 계획 commit으로 보존한다.
2. **Architecture 결정:** resolver-only privileged executable 규칙에 별도 설치 역할을
   추가할지 사용자가 검토한다. 승인 없이 기존 helper 권한 확장·새 root executable을
   구현하지 않는다. 초기 Linux/amd64 + same-runtime-lock 지원 한계도 함께 명시한다.
3. verifier의 고정 구현/신뢰 root/획득 bounds와 IPC·관리자 인증·crash recovery의
   구체적인 상태 전이를 확정한다. 모호한 callback으로 보안 결정을 미루지 않는다.
   확정한 선택을 canonical leaf에 반영한 뒤 해당 gap의 소스만 작성한다.
4. acquisition/provenance와 frozen-asset 검역을 작은 독립 단위로 구현·검사·commit한다.
   이 시점의 준비 성공을 업데이트 성공으로 출력하지 않는다.
5. 설치 역할의 고정 transaction/잠금·peer 전환·doctor/복구를 구현한다. unit/fault
   회귀 뒤 disposable Linux host에서 root lifecycle과 재부팅 복구를 검사한다.
6. native CLI를 연결하고 기존 설치→새 세트→실제 검역 정상→업데이트 실패 시 이전
   세트 복원의 소비자 경로까지 qualification한다. 이전 version을 먼저 지우지 않는다.
7. 이름 승인 후에만 npm/PyPI/pipx/Homebrew wrapper/binding을 연결한다. 일반 native
   manager 명령을 가로채는 기능과 arbitrary postinstall 실행은 추가하지 않는다.
   채널·지원 platform별 acceptance와 M13-005 공개 배포는 각각의 증거로 닫는다.

## 4. Acceptance 및 검사 계획

| ID | 실제 확인할 조건 | 반대 사례/실패 보존 |
| --- | --- | --- |
| A1 identity/acquisition | exact official version/tag/commit/asset set freeze, bounded bytes/request/redirect/time | 다른 repo/workflow/source, branch/latest/arbitraryURL, drift/duplicate/missing/extra asset, proxy/credential, timeout·oversize·redirect 거부 |
| A2 provenance | 공식 signed fixture와 trusted verifier/root로 subject/source/workflow binding 실제 검증 | matching digest만 있는 bundle, 새 bundle의 verifier/root, user PATH 위장 executable exit0, invalid/revoked proof, wrong subject/tag/source, offline root 부재 거부 |
| A3 quarantine | 기존 trusted controller에서 각 selected native executable의 required check/Evidence/Policy 확인 | 새 release의 self-report, required 미실행/unsupported/incomplete, BLOCK/REVIEW/UNAVAILABLE, observer/accounting/cleanup 실패 시 활성 파일 무변경 |
| A4 byte identity/authority | 검사·root copy·stage·promotion의 전체 세트 digest 동일, old peer 인증 | staged file/path swap, symlink/hardlink/mode/owner/member 변조, user receipt/claimed UID/다른 inode·같은 basename/PID 재사용/검사 뒤 download 거부 |
| A5 truthful capability | Linux/amd64 healthy trusted runtime에서 actual probes 완료 | macOS/arm64나 unhealthy runtime을 provenance-only 성공으로 바꾸지 않음; 기존 상태 보존과 구체 코드 |
| A6 transaction | 단일 owner, durable stage/journal, prior 보존, 새 protected CLI/helper 인증과 actual doctor/다음 검사 성공 | concurrent update/active run, failed copy/fsync/rename/service/doctor·취소·disk full·부분 전환, package DB 충돌, runtime-lock 변경 시 거부/복구 |
| A7 lifecycle/recovery | 소유 launch/실제 wait·드레인 확인, journal 기반 재시작 복구 | UNKNOWN에서 privileged rebind/restore 금지, kill/query timeout·중단 지점별 재부팅·rollback failure 분리; primary status 유지 |
| A8 UX/contracts | 실제 self-update 수행/exit·version/digest/Policy/Evidence·activation report 일치, EN/KO 완전·나머지 catalog/fallback | 도움말 exit0/echo/stage 완료를 업데이트 성공으로 처리하지 않음; JSON stdout 오염/secret·host path 누출 거부; 기존 여섯 생태계 회귀 보존 |
| A9 packaging/public boundary | 기존 exact asset set 소비자와 기존 installer/doctor/package test green | compile/native build/로컬 회귀를 booted package 설치·public publication·clean Host 성공으로 확대하지 않음 |

검사 실행은 고정 Go/tool bootstrap과 canonical `go run ./scripts/check` profile을
우선한다. 문서 후보는 `docs`, source 후보는 `quick`/`security`와 관련 native 회귀,
새 import의 `vulnerability`, 배포 payload 변경은 기존 `release-gate`/package 소비자를
적용한다. OS build validity와 native 실행을 구분한다. Architecture/보안 수동 검토는
자동 도구 성공과 별도다. 필수 CI/qualification budget·coverage·Policy를 완화하지 않는다.

실제 root lifecycle은 기존 canonical helper 운용·복원 owner를 재사용하는 disposable
host에서 수행한다. 현재 사용 중인 보호된 설치의 영구 변경은 이 계획 commit이나
검사 실행 권한만으로 수행하지 않는다. task-private source/cache/stage/operator는
마지막 소비와 증거 캡처가 끝나면 삭제하고 삭제 범위를 기록한다. shared pinned
SDK/tool cache·raw evidence·복구에 필요한 active/prior release는 보존한다.

## 5. 두 번째 검토 — 코드와 계획 교차 대조

2026-10-20, 초안 작성 뒤 아래 책임을 다시 대조했다. 독립 reviewer의 승인을
받았다는 뜻은 아니며, 설계 검토가 구현 검증을 대신하지 않는다.

| 검토 항목 | 소스/계약 근거 | 결론과 반영 |
| --- | --- | --- |
| 기존 installer를 재구현하는가 | Installer.Install/activateCurrentWithSync 및 기존8개 회귀 | 재구현 제외. 새 root lifecycle·검역 gate gap만 명시 |
| signature=digest=safety로 혼동하는가 | BundleVerifier 계약, Architecture-006, GitHub declared digest verifier | 세 책임 구분. production verifier와 기존 Policy 재사용을 별도 acceptance로 둠 |
| CLI만 바꿔 실제 검역을 깨뜨리는가 | DefaultRoot의 user root, PolicyPeerAuthorizer의 protected inode/digest | protected 세트/인증 전환·다음 검역까지 acceptance 추가 |
| 현재 release가 full runtime을 제공하는가 | release-manifest.assetSpecs의8종, M8 runsc/companion identity | runsc/host config 누락 명시; same-runtime 입증 또는 거부. latest 보충 금지 |
| OS package install success를 runtime 준비로 오인하는가 | release-packages.packageReadme: no runtime activation/no service enable | package DB/migration/booted host를 별도 미완료로 둠 |
| helper 경계 변경을 묵시 승인으로 보는가 | Directory의 resolver-only privileged helper, M10 narrow installation | 별도 설치 역할은 제안이며 owner 결정이 남음. network helper 범위 유지 |
| macOS native CLI build를 동적 검역 지원으로 확대하는가 | 기존 ELF backend의 Linux dynamic capability | 초기 지원 한계·fail-closed와 승인 필요를 명시 |
| 이름 미정이 검역 우선순위·일반 project 명령을 바꾸는가 | 사용자 최신 결정, M13-004 namespace contract | native core 우선, 이름 의존 연결만 보류; project 동일 이름 hijack 금지 |
| 완료·청소·상태가 누락됐는가 | Queue 상단/상세 및 L08/L16 | plan/implementation/qualification 분리; 상태 요약 함께 수정; 종료 확인 전 삭제 금지 |

당시 남은 **설계 결정**은 설치 전용 권한 경계/초기 platform 범위 승인과 verifier·IPC·복구
상세였다. 공식 패키지 이름은 채널 연결의 별도 owner 입력이다.

## 6. 승인 후 재개 — 2026-10-20

사용자의 `진행해`로 별도 설치 역할과 Linux/WSL2 amd64 우선 구현이 승인됐다.
770c302의 계획 문서8개는 canonical docs/security와 staged blob/mode 검증을 통과한
후 커밋됐다. 완료한 임시 source/cache 331,624,286 logical bytes와 operator는 삭제했다.
M13-004는 IN_PROGRESS이며 이름 의존 채널 연결만 pending이다. 기존 audit의 NO는
승인만으로 YES가 되지 않는다.

첫 구현 단위는 production provenance verifier다. `internal/verification/releaseprovenance`에
고정 Sigstore library·public root·strict identity/predicate/subject 검증을 구현하고,
M13-003의 committed 실제 서명 bundle/manifest를 그대로 positive/negative 회귀에 사용한다.
기존 단순 GitHub digest provider와 `releaseinstall` API는 재작성하지 않는다. verifier
단독 완료를 전체 본체 업데이트 완료로 보고하지 않는다. IPC·activation 상태 전이는
해당 소스 변경 전에 별도 구체화하고 이 계획에 연결한다.

첫 구현 단위의 [로컬 검증](./provenance-qualification-20261020.md)은 production
provenance primitive의 IMPLEMENTED=YES / WIRED=NO / QUALIFIED=YES(local) /
ACCEPTANCE_CLOSED=NO다. 보안 수정 pin·actual signed/offline 회귀·기존 소비자
Quick/security/vulnerability/Race와 Darwin compile을 확인했다. 다음은 frozen release
획득/검역 연결이며, 본체 command/보호된 활성화/전체 qualification은 아직 없다.

다음 소스 변경은 [frozen acquisition baseline·정확한 수정 위치·second review](./acquisition-plan-20261020.md)를
따른다. 실제 API의 Azure/Snappy 응답을 반영하며 기존 Resolve/Acquire의 반복 호출이나
임의 URL·새 Policy·무제한 decoder로 대체하지 않는다.

획득 연결부의 [로컬 검증](./acquisition-qualification-20261020.md)을 마쳤다.
다음 source 변경은 [metadata/native set 감사·수정 위치·두 번째 검토](./native-set-plan-20261020.md)를
따른다. 기존 complete-set application/Policy/staging을 재사용한다.
