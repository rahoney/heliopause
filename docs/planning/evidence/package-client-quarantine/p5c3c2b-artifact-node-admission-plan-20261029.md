# P5-C3c2b — Artifact Node admission/lifecycle gap 감사

기준 HEAD 4de6fadf632c8199de984c467f5e768e232acdc1 /tree095a6e5e/2950 tracked·1320 runtime inputs다.
단일 M13-004 아래 C3c2a는 named local 완료이며 CI426은 이전4db source로 실행 중이다.
Read: client09 §3, Architecture-005/Contract-002, C2/C3b/C3c1/C3c2a, INDEX와
sandbox L03/L08/L12/L18·artifact L02·evidence L16·ci L05/L06를 대조했다.

| 기존 owner / 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| npm.NodeExecutionLayout + Render/CompareNodeExecutionInputs | YES | YES(current DATA) | YES(local) | YES(C2/C3c2a) | 변경 없음; exact current layout/entry를 user execution에 전달 |
| prepareNodeExecutionInputs/owned cleanup/sole readonly attachment | YES | YES(DATA validation) | YES(local fresh2) | YES(C3c2a) | 새 Artifact profile 소비자를 연결; 기존 DATA 경로 유지 |
| boundaryHelper/directExecOrigin/ordered admission lifecycle | YES | YES(existing launch/Python/ELF) | YES(local+prior hosted) | YES(existing) | Node를 ARTIFACT로 넘기는 exact closed mode/profile; CONTROL 우회 금지 |
| CPP PendingAdmission/BindDirectExecAdmissionRoot/RetainDirectExecTerminal + directExecOutcome | YES | YES(kernel/root caller) | YES(local8roots/CPP129) | YES(C3c1 bounded) | root 종료와 별개 accepted target image-load fact 없음 |
| hosttool.Executor.RunCaptured | YES | YES(Host transport/local fixture) | YES(local) | YES(C1 bounded) | same admission/image/terminal + observer/descendant cleanup 결합 |
| actual Artifact Node/result/application/CLI | NO | NO | NO | NO | production request→exact entry/vector argv→Artifact role→bounded run/result; cancellation/overflow/evil/fresh tests와 hosted/final |

## 수정 지도 — C3c2b의 첫 dependency: target image-load 사실

Root terminal만으로 selected target start를 판정하지 않는다. 기존 CPP SENTRY_EXEC의
launch_target_pending 및 handoff_target_pending 수용 지점에서만 같은 consumed admission의
root group/start에 bounded target_loaded bit를 보존한다. Helper root bind나 syscall exec 시도,
clone child·foreign/reused group·stdout marker는 이 fact를 만들지 않는다. 기존 max64 slot과
같은 peer/generation/mode/nonce/7scalar ACK로 `target` 조회만 추가한다. 기존 outcome 및
ARM/STATUS/CANCEL/COMPLETE 의미/fields는 그대로다. `pending`, `waiting`, `loaded`,
`missed`(terminal이지만 target 없음)를 구분한다. 기존 slot이 이미 정상 해제됐으면 ordinary
소비자 event를 새 실패로 만들지 않는다. 아직 이 fact는 profile이 수용한 target class의 load를
증명할 뿐 미래 Node entry/argv/source 승인이나 complete output/drain/Policy를 증명하지 않는다.

수정 owner는 observer.cc PendingAdmission/ParseControlRecord와 기존 두 target-accept 지점,
Go direct_exec_admission.go의 exact semantic validator, 새 private target query/회귀다.
기존 tagged trusted8root fixture에 독립 target query를 연결하고 CPP latch는 실제 SENTRY_EXEC
생산 경로와 pending/missing/foreign/descendant/duplicate/terminal-without-target를 검증한다.
기존 C3c1 kernel fixture·protocol latch 본문은 가능한 그대로 재사용하고 새 mode/profile/FS
권한은 이번 dependency에서 추가하지 않는다. Canonical checks/영향 sandbox Race/lockedCPP
전체 latches 및 same-source actual8roots+existing npm/Python/ELF 전에는 QUALIFIED=NO다.

## 두 번째 검토

기존 구현을 교체하지 않으며 새 bool은 반환 DATA다. 기존 CONTROL/ARTIFACT role·root eligibility,
trusted network·demotion·classifiers·filesystem/trace/CPU/wall/output/cleanup 예산을 변경하지 않는다.
Profile의 exact target 수용 이후, state commit 이전에만 fact를 남긴다. Query 오류는 UNKNOWN이며
정상 started/zero exit로 바꾸지 않는다. Raw argv/출력·새 저장 schema/가짜 SandboxResult·Core/CLI
또는 미래 milestone scaffold는 없다. This dependency의 local 통과는 Node user-run 완료가 아니다.
그 뒤 전용 Node profile/mode의 실제 owner map/negative들을 추가 감사하고 missing만 구현한다.

## C3c2b 첫 dependency 구현 후보

Go source의 canonical7은 format1.376s/Quick284.548s/target+outcome focused1.691s/
전체 sandbox freshRace32.267s/tagged build7.934s/docs0.261s/security16.985s로 PASS다.
Go binary c79f6942/27,588,426B·전체 runtime census1322와 기존 변경4owner/newGo2를 동결했다.
CPP 첫 빌드 전 review에서 original admission mode까지 target fact를 결합했다. 같은 root의
후속 다른 handoff를 최초 target 중복으로 오인하지 않도록 exact mode mismatch는 fact를
만들지 않고 원래 slot을 유지한다. 이 검토는 actual 회귀 발생 증거가 아니다. Go/module/scripts
입력은 검토 전후 byte-identical이며 CPP source/latch 변경은 별도 exact canonical build에
연결한다. 원본7checks/Go binding/pre-build review/source·patch는 raw p5-c3c2b-*에 보존했다.
Locked CPP130 latch build와 same-source 실제8root/기존3소비자는 아직 진행 전/결과 대기다.
IMPLEMENTED=YES/WIRED=YES(existing kernel/query/controlled fixture), QUALIFIED=NO,
ACCEPTANCE_CLOSED=NO다. Artifact Node mode/profile/user-run/application/CLI/final은 별도 missing이다.

## 첫 dependency named local 검증 완료

[최종 receipt](./p5c3c2b-target-image-local-20261029.json)는 source1322/Go c79f6942와
CPP0c09f44f의 locked265.804s/130latches·actual8roots9.827s/parent7.85s 및 기존 npm/Python/
ELF3parents12.296s PASS를 연결한다. 각 root는 target-loaded를 terminal과 별도로 query했고
exit0/37/125..127/255·TERM/KILL/별도3B streams를 유지했다. 모든 source/42/ZIP/Host/Docker
보존·helperwait0/final0/owned workspace 부재가 대응한다. 위 QUALIFIED=NO는 이 후보의
first dependency local 범위에서 YES로 갱신된다. Profile이 수용한 target class load까지만
검증했으며 selected Artifact Node entry/argv 승인·프로그램 출력 완전성/drain/Policy/CLI는
별도 미완료다. 다음은 기존 Linux CI에 이8root parent의 noSKIP/exactPASS guard를 연결하는
독립 wiring 단위다. 원격 CI426의 all10SUCCESS는 이전4db source이며 이 CPP 변경은 제외된다.
