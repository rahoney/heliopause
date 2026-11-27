# P5-C3c hosted kernel parent — baseline / 수정 지도 / 두 번째 검토

기준47464b4/tree22ce647a/2955 tracked·1322runtime이다. M13-004 하나만 IN_PROGRESS다.
Read: Queue/client09§3/확장계획§5와 C3c1/C3c2a/C3c2b receipts, INDEX 및 ci L05/L06,
sandbox L03/L08/L12/L18/evidence L16을 대조했다. CI426은 이전4db의 all10SUCCESS로
실제 timer05를21:33:27UTC 소비했고 현재 active timer/CI는 없다.

| 실제 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| TestLinuxDirectExecKernelTerminalIntegration:8roots/target+terminal/streams/cleanup | YES | YES(local actual) | YES(local) | YES(named local) | 기존 Linux job에서 same source/observer 실제 실행 |
| Linux lifecycle step의 tagged sandbox binary / haa_install_ci_client / helper owner | YES | YES(existing PNPM/Bun/Python) | YES(CI426 existing) | YES(existing) | 같은 빌드 결과를 별도 unconditional kernel step에서 사용 |
| production Node preparation readonly fresh2 parent | YES | YES(existing step) | YES(local C3c2a) | NO(new hosted) | 새 grouped source에서 실제 remote 결과 소비; source 수정 없음 |
| activeJobRunLines/validateCIWorkflow/Required9+1 | YES | YES(canonical check) | YES(existing) | YES(existing) | kernel step의 closed active commands/order/noSKIP/exact parent+8cases 연결 |

수정 지도: 기존 workflow gvisor-integration job의 마지막에 `Qualify kernel direct-exec image
and terminal facts` step만 추가한다. 기존 tagged sandbox binary와 canonical CI client/helper
lifecycle을 재사용한다. umask077 private log에 실제 command exit를 먼저 저장하고 log 출력
오류와 구분한다. 성공 뒤에도 noSKIP/exact top-level parent1/eightcase8을 요구한다. Runtime
build/supply/새 job/Required·timeout·product source·CPP·권한·예산은 바꾸지 않는다.

새 kernel_terminal_workflow.go는 기존 YAML parse/activeJobRunLines로 owning job의 유일하고
조건 없는 step와 fixed block을 검증한다. Workflow validator 연결 한 줄과 의미 있는 negative
mutations 및 실제 Bash status/log guard fixtures를 추가한다. 이것은 fixed repository convention
검사이며 임의 Bash 전체 제어 흐름의 실행 증명이 아니다. 이미 실제 qualified인 Go/CPP/8root를
다시 구현하거나 다른 mode/profile/user-run 성공으로 확대하지 않는다. Read-only workflow/
Quick/docs/security·checker 전체 Race와 operator guard 전 QUALIFIED=NO다. 이후 source equality/
whole Git 후보를 확인하고 기존 milestone branch로 한 번 push해 새 grouped CI timer를 등록한다.

## 로컬 검증 완료와 다음 원격 후보

[최종 receipt](./p5c3c-hosted-kernel-wiring-local-20261029.json)는 source1324의 format1.434s/
workflow2.291s/guard1.65s/checker 전체Race47.748s/Quick283.313s/docs0.277s/security17.394s를
연결한다. 실제 Bash7case는 정상 parent1/8cases, primary37 보존·SKIP/누락/중복/7·9cases
거부이며 full-workflow9 negative와 구분한다. 기존 product/CPP/8root 소스는 byte-identical다.

추가 중요 조합은 같은 source1324/Go3575dbab/CPP0c09f44f의 production parent54.34s/전체
56.226s/fresh2 PASS다. Current whole checks/Policy ALLOW/storage-only, offline materialization과
sole readonly 입력 census·Host/Docker/42/ZIP/source 보존/helperwait0/final0/workspace 부재를
검증했다. 최초 producer v1은 old CPP copy path와 new receipt 불일치를 hash guard가 line14에서
잡아 helper/namespace/actual 전에 중단했다. 원본과 세 파일 inventory를 보존하고 새 v2에서
해당 경로만 교정했다. Runtime 오류·자격 실패로 소급하지 않는다. New hosted 전 QUALIFIED는
named local wiring/조합에서만 YES이며 실제 Artifact Node/user-run/CLI·final acceptance는 OPEN이다.
