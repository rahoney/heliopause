# R3 로컬 검증과 pnpm·Bun 공유 예산 대조 (2026-11-18)

**이 문서는 최초 진단·실패 원장이다. 최신 승인 교정과 원래 전체 회귀 PASS는 [R3 로컬 체크포인트](./r3-local-checkpoint-20261118.md)를 따른다. 아래 과거 FAIL·미승인 상태는 당시 기록이며 원본을 바꾸지 않는다.**

**판정: R3 미완료.** 기존 성공 코드·실행 절차를 재사용했다. 로컬 canonical 검사는 통과했으나 원래 pnpm·Bun 부모 시험은 실패했다. R4 적용과 R3 완료 commit은 하지 않았다. 현재 재개 규칙은 원래 작업 트리의 23번 계획 §0이며 Queue R5의 마지막 줄이 재개 위치다.

- 진입: `607d79f522f38d55dc005eba1a63eb800f9383f1` / tree `d7f94049851ae24525af99d9117cff91e836440a`.
- 검증 후보: prospective tree `cff4ab234fdf6985f5fbcf35ebb387e55f70ee53`. commit 미생성, 소스 `/tmp/haa-r3-environment-d7puheoo/proposal`. 선언 +189/-19 의도 일치. 기존 3374파일의 code tree ca106814에 기존 근거 leaf 2개만 추가한 후보이며 full/security를 문서 변경만으로 반복하지 않았다.
- 재사용한 실제 성공 strict 빌드: 같은 compile-inside.sh·builder·bridge/CPU4/RAM·disk8GiB/22분. compile791.911s, kernel7·하위8 PASS, CPP 기존134+별도Go/Cargo2=136 PASS. 새 6member/manifest만 기존 canonical owner로 lock에 결합했다.
- canonical --full395.225s, security56.969s, docs2.594s, current vulnerability29.413s PASS. 영향9owner Race101.407s: 부모671+하위1467=2138 PASS/FAIL0. opt-in SKIP59는 미실행이며 실제 통합의 대체 증거가 아니다.

## 실제 정상·거부 검증

| 원래 전체 부모 | PASS 초 | 원본 실행 폴더 |
| --- | ---: | --- |
| `TestNPMEnvironmentEvidenceIntegration` | 11.898 | `/tmp/haa-r3-native-9h89e9dz` / native-1 |
| `TestPythonEnvironmentEvidenceIntegration` | 54.356 | `/tmp/haa-r3-native-9h89e9dz` / native-2 |
| `TestLinuxNPMInspectCLIIntegration` | 12.554 | `/tmp/haa-r3-native-9h89e9dz` / native-3 |
| `TestNPMPublicProjectCLIIntegration` | 45.424 | `/tmp/haa-r3-native-9h89e9dz` / native-4 |
| `TestNPMPublicProjectRootCLIIntegration` | 109.55 | `/tmp/haa-r3-native-9h89e9dz` / native-5 |
| `TestLinuxUVInstallCLIIntegration` | 287.429 | `/tmp/haa-r3-native-9h89e9dz` / native-6 |
| `TestLinuxUVProjectNativeAdoptionCLIIntegration` | 141.598 | `/tmp/haa-r3-native-9h89e9dz` / native-7 |
| `TestLinuxUVToolInstallCLIIntegration` | 92.742 | `/tmp/haa-r3-native-resume-74gvd46f` / native-1 |
| `TestLinuxPipxToolInstallCLIIntegration` | 117.294 | `/tmp/haa-r3-native-resume-74gvd46f` / native-2 |
| `TestCargoPublicFetchCLIIntegration` | 41.954 | `/tmp/haa-r3-native-resume-74gvd46f` / native-3 |
| `TestLinuxTerraformInitIntegration` | 151.334 | `/tmp/haa-r3-native-resume-74gvd46f` / native-4 |
| `TestLinuxTerraformInitNegativeIntegration` | 7.807 | `/tmp/haa-r3-native-resume-74gvd46f` / native-5 |
| `TestLinuxGoPublicBuildCLIIntegration` | 129.896 | `/tmp/haa-r3-native-resume-74gvd46f` / native-6 |
| `TestLinuxGoBuildSecurityCLIIntegration` | 64.928 | `/tmp/haa-r3-native-resume-74gvd46f` / native-7 |
| `TestLinuxCargoLibraryBuildCLIIntegration` | 35.173 | `/tmp/haa-r3-native-resume-74gvd46f` / native-8 |
| `TestLinuxCargoBuildScriptBuildCLIIntegration` | 39.523 | `/tmp/haa-r3-native-diagnostic-jrllpylk` / native-2 |
| `TestLinuxCargoEnvironmentTriggeredNetworkCLIIntegration` | 20.077 | `/tmp/haa-r3-native-diagnostic-jrllpylk` / native-3 |
| `TestLinuxCargoSecondEnvironmentFailureCLIIntegration` | 18.166 | `/tmp/haa-r3-native-diagnostic-jrllpylk` / native-4 |
| `TestLinuxGVisorFreshEnvironmentIntegration` | 8.133 | `/tmp/haa-r3-native-diagnostic-jrllpylk` / native-5 |
| `TestLinuxGVisorTransientFilesystemIntegration` | 7.803 | `/tmp/haa-r3-native-diagnostic-jrllpylk` / native-6 |
| `TestLinuxPythonFreshEnvironmentIntegration` | 13.196 | `/tmp/haa-r3-native-diagnostic-jrllpylk` / native-7 |
| `TestLinuxPythonTransientFilesystemIntegration` | 10.4 | `/tmp/haa-r3-native-diagnostic-jrllpylk` / native-8 |
| `TestLinuxGitHubReleaseELFDynamicIntegration` | 5.32 | `/tmp/haa-r3-native-diagnostic-jrllpylk` / native-9 |

별도 production: packaging25.0 정상 설치·required3/ALLOW·승격·import, 정확한 observer 종료에 따른 관찰 미완료 거부, CLI SIGINT 취소와 거부 대상 불변을 확인했다. 취소 intake RETAINED 사실은 보존하며 시험 후 수동 정리를 제품 C01 완료로 쓰지 않는다. Go6모듈 tidy26.934s는 정상 실제 fixture일 뿐 대형 Go/build 검증을 대신하지 않는다. UV 설치287.429s는 원래 context300s 이내이나 여유가 작다.

## 첫 실패별 원인·교정 및 성공 대조

| 사건 | 원본 결과 | 원인·교정·재검증 |
| --- | --- | --- |
| UV tool 입력 | 0.313s 제품/하위 미도달 | 기존 성공 고정 interpreter DATA/SHA 환경변수2개 누락. 원래 입력 그대로 복구, UV92.742s·Pipx117.294s 전체 부모 PASS |
| pnpm native 공급 | single3.755s graph 전 거부 | R2 전용 ZIP 공급만 복제하며 R1 고정 Bun/PNPM2개 배치를 누락. 보존된 원래 pin/원자 배치·owned inode 회수 재사용, 다운로드0. 이후 시간 실패는 별도 |
| pnpm 공유 deadline | 전체240.510s FAIL; single120.16/peer118.15 PASS 뒤 empty2.01 FAIL | 한 context240s가 3fixture의 모든 반복/거부 요청을 공유. 원래 empty만 별도 진단15.046s 및 후단 native-adoption 거부 PASS지만 원래 부모 FAIL은 유지 |
| Bun 공유 deadline | 전체241.158s FAIL; single138.25 PASS, peer102.66 FAIL, empty 미도달 | 같은 공유 context. 필요한 dynamic 관찰 미완료를 제품이 거부했다. 무관한 통과 부모는 재실행하지 않음 |
| 기존 성공 코드 대조 | R2 pnpm180.064s·Bun202.285s 전체 부모 PASS | 같은 Host/CPU2/고정 image·native assets/원래 context240s/프로세스5분. 보존한 R2 실제 바이너리·당시 runtime6/CPP/pod-init를 짝지어 사용, R3와 혼합/재빌드0 |

R2 control 실제 소스는 git archive607이다. 바이너리는 당시 검증된 build snapshot078이며 제품 Go/lock1525경로를 그 원본 신원 manifest와 직접 해시 대조했다. raw result의 상속된 base 0bb267b/d720은 그대로 보존했고 실제 소스와 혼동하지 않는다. R3 변경 묶음의 비용 증가와 공유 context 소진은 확인했지만 S01/S08 각각의 비용 기여나 최소 제품 최적화는 미확정이다. R2 PASS를 R3 성공으로 전용하지 않는다.

## 미적용 후보와 다음 경계

검토 가능한 최신 diff: `/tmp/haa-recovery-rules-viq0l2f5/pnpm-bun-test-context-proposal/measured-minimal-test-budget.patch`. 기존 시험2개에서공유240s context를각기존CLI요청으로옮기고,`scripts/ci-bun-runtime.sh`의Bun CLI부모한개만외부5→6분으로정한다(pnpm5분유지). 원래3fixture·공유cache·fixture간operationID중복거부·모든정상/거부assertion을같은프로세스에서유지한다. **전체허용은pnpm240→최대300s·Bun240→최대360s이므로시험한도승인이필요하며미적용·미실행이다.** 제품·runtime·Policy·관찰·개별experiment한도및지원범위변경0. 앞선두부모모두5분안은Bun실측합297.34~304.44s에여유가없어교체했고원본을보존한다. 과거FAIL은유지하며승인뒤정확후보·기존도구로표적점검/해당tagged binary만재빌드/원래전체2부모검증후판정한다. 안전성입증없는bundle신원검사cache·삭제를성능수정으로넣지않는다.

기존 Host14·current3/SDK/개발 HEAD·index·tracked diff는 독립 읽기 대조 PASS, helper active/running/enabled, CLI/observer/gVisor 프로세스·컨테이너·volume0, 기존 network와 firewall 규칙 보존. 고정 공급2개의 소유 inode는 각 창 finally에서 제거했다. 이는 D4 전체 Host 백업 검증이 아니다. 전체 CPU·CUDA3/대형 Go/hosted·R8은 OPEN, D3 적용 전 R4·D1 적용 전 R5 정책 경계를 유지한다.

기계 판독 원장: [r3-local-verification-20261118.json](./r3-local-verification-20261118.json). local/race 원본 `/tmp/haa-recovery-rules-viq0l2f5/r3-local-qualified.json`, `r3-race-qualified.json`; native 각 폴더의 result.json/stdout·정리/복원 원본, control `/tmp/haa-r3-r2-control-e8_d7gif/result.json`, 독립 최종 `/tmp/haa-r3-r2-control-e8_d7gif/independent-final-host-check/result.json`. 완료 산출물은 증분 보존·해시/복원 대조·열린 참조/질의 오류0 확인 후 정리하며 현재 후보 바이너리·검증 bundle·고정 입력·원본은 유지한다.

## 240초 실패 후 추가 원인 분리

원래 pnpm single과 후단 미검증 native 인수 거부를 동일 cff/기존 바이너리로 진단했다. 부모122.608s·single120.96s/FAIL0·SKIP0, 정상 CLI5회·frozen lock 불변·fresh ID·수정 control 거부·native 인수 거부를 유지했다. 제품/원시험 수정0, 내부240s·canonical 외부5분 그대로다. Docker events는 resolver10·writer10·anchor10과 두 환경 lifecycle10회를 보였다. 개별 명령 정체는 관찰하지 못했지만 snapshot/검증 등의 독점 시간까지 분리된 것은 아니다. container lifespan은 서로 겹치므로 합계를 독점 비용으로 쓰지 않는다. 전체 부모 FAIL은 그대로이며 single 진단을 전체 성공으로 전용하지 않는다. 원본 `/tmp/haa-r3-pnpm-cost-ah1w3sr3/{result.json,diagnostic-docker-events.jsonl,diagnosis.json}`. Host14/current3·개발Git·자원·방화벽원복은독립확인했다.

**실행기 메타데이터 교정:** 이전T/U의 실제 외부 process 설정은10m였고원래canonical scripts는5m다. 두실패는내부공유240s에발생했으므로외부10m가FAIL을PASS로바꾼것은아니다. raw명령은편집하지않는다. 현재진단부터canonical5m를명시했고향후수정후전체부모도5m를사용한다. 제품/관찰한도변경은없다. 공유context범위만옮기는구체적diff의모든다른줄동일을검증했으나전체허용240→최대300s변경이므로사용자질문이대기중이며미적용이다.

## Bun 미도달 사례·CPU 비용의 후속 검증

같은 source/runtime/compiled binary와 원래 내부240s·canonical외부5분에서 Bun peer151.25s·empty14.94s 및후단native인수거부가PASS했다(진단부모167.924s). 같은원peer에표준Go CPU profile을붙인진단도peer144.15s·부모145.735s PASS다. 기존single138.25s를포함해각정상/거부사례는범위내통과했으나원래3fixture합친부모FAIL·공유cache/ID전체검증은닫지않는다. source수정0·새실행도구/외부입력설치0,기존Go1.27.4 src/cmd/pprof를offline실행해분석했다(SDK에미리빌드된pprof도구가없었음).

프로파일145.48s중부모CPU샘플51.46s, SHA-512 block25.29s, 전체gVisor bundle검증경로누적19.06s다. 이는겹치는CPU수치이며전체wall/외부observer·runtimeCPU가아니다. `Executor.tool→verifyLocalGVisorBundle→verifyExactGVisorBundle→verifyExecutable`와capability probe의실제호출을확인했다. 이검사는기존성공owner에도있으므로표본만으로회귀전체원인으로단정하거나안전성근거없이cache·생략하지않는다. 실제Bun소비자가무조건막힌것과새필수두환경비용을기존집합용context로묶은시험문제를구분한다.

원본: `/tmp/haa-r3-bun-remaining-2_4n7p69/result.json`, `/tmp/haa-r3-bun-cpu-7hr833mj/{result.json,original-peer.cpu.pprof,pprof-source-flat.stdout,pprof-source-cumulative.stdout,pprof-source-sha512-callers.stdout}`. CPU진단뒤원Host14/current3·서비스·프로세스/container/volume0·기존network·firewall규칙·개발HEAD/index/diff/SDK는독립원복대조PASS다. 최신검토diff는시험3파일만이며새한도승인전미적용이다.
