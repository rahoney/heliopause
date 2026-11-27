# R3 로컬 기능 검증 체크포인트 (2026-11-18)

**R3 로컬 기능 범위 검증 완료. 전체 K·hosted·R8 qualification은 OPEN이다.** 새 범용 검사기나 복구 daemon을 만들지 않고 기존 성공 코드·실행 절차를 재사용했다. 다음 독립 작업은 U1을 제외한 기존 S02/S03·후손 drain 선별 준비다. D3 자동 회수 적용과 R4 전체 완료에는 별도 관문이 남는다.

## 정확한 소스·검증 범위

- 진입 commit `607d79f522f38d55dc005eba1a63eb800f9383f1`, tree `d7f94049851ae24525af99d9117cff91e836440a`.
- 실제 검증 prospective tree `a95c084fc77b8c5c45e99c44b3540ec11b170dfd`(commit 미생성 상태의 고정 사본). prior `cff4ab234fdf6985f5fbcf35ebb387e55f70ee53` 대비 시험2파일·Bun 실행 명령·그 명령을 확인하는 canonical 상수4파일만 다르며, 다른3372파일은 해시·모드 동일하다.
- 사용자가 승인한 시험 집합 예산: 각 기존 CLI 요청240s, 원래 전체 프로세스 pnpm5분·Bun만6분. 제품·Policy·관찰·개별 실험·CPU/메모리/디스크 한도와 원래 정상·거부 사례는 유지했다. canonical 상수 한 줄도 같은6분으로 동기화해 검사 우회를 만들지 않았다.
- workflow32.34s PASS, 기존 누락/중복/복원 계약4부모31하위35PASS/FAIL0/SKIP0(2.148s), 변경된 tagged bootstrap 시험binary만90.373s에 재빌드했다. 실제 고정 SDK1.27.4/binary SHA256 `d508d48a7181a3323e7d6ae32ea4843f036384f2407fc0f7eb30d9f82bd3f987`. 기존 제품 CLI/helper/sandbox3개는 byte와 product source가 동일해 재사용했다.
- mandatory `rebuild-step-check.sh 607d79f --full` PASS812.737s: build/testbuild/vet·canonical Quick/Staticcheck/default tests·실행 연결·CI selector. 선언추가189/제거19와 공개 무참조 목록은 기존 의도/직전 성공 출력과 정확히 동일하고 이번 시험 교정의 선언 증감은0이다.

## 원래 전체 pnpm·Bun 회귀

| 부모 | 전체 초 | 원래 사례 | 요청·보안·무결성 | 결과 |
| --- | ---: | --- | --- | --- |
| TestPNPMProjectCLIIntegration | 277.583 | single/peer/empty | 정상9·frozen2·공유cache·전역operationID·implicit-control/native-adoption 거부 | PASS/FAIL0/SKIP0,5분 이내 |
| TestBunProjectCLIIntegration | 304.698 | single/peer/empty | 같은 원래 검증 전부 | PASS/FAIL0/SKIP0,승인6분 이내 |

Go stdout의 부모/하위 terminal PASS와 마지막 fresh_operations=9 및 두 REFUSED 표기를 함께 대조했다. 사례만 따로 통과한 진단으로 전체 부모를 대체하지 않았다. 과거240초 FAIL과 이후 진단 원본은 그대로 유지한다.

pnpm PASS 직후 서버 재시작으로 RPC 출력 파이프가 닫혀 실행기의 print가 BrokenPipeError였다. 최초 실행기 raw status STOPPED를 수정하지 않고 제품 PASS/후행Bun 미수행을 분리했다. finally Host 원복을 독립 확인한 뒤 기존 실행기를 파일 출력으로 분리해 Bun만 재개했다. 새 제품 수정·pnpm 반복·예산 추가 변경은 없었다.

## 재사용 근거·미완료 범위

제품/호출/의존 source·고정runtime6/observer/helper·이미지·profile/Policy가 같고 최근 변경은 두 시험의 지역 context와 canonical 명령 literal뿐이므로 직전 실제23부모/production packaging25 정상·observer종료거부·CLI취소3case/Go6모듈tidy와 영향owner Race2138PASS를 해당 범위에서 재사용했다. 원래testbinary의 전체SHA가 같다고 주장하지 않으며 새bootstrap identity와 공통 인증 연결은 이번 전체2부모에서 실제 검증했다. 이로써 실제 native 부모는 중복 없이25PASS다. Go6은 대형Go/build 대체가 아니고 취소 intake RETAINED는 수동 정리를 제품C01 완료로 전용하지 않는다.

기존 같은 R3 runtime의 build791.911s/kernel7+하위8/CPP134+표적2=136/6member capture·verify를 이번 시험 전용 변경 때문에 반복하지 않았다. 이전security/vulnerability PASS는 그때의 source·입력·시각에 한정한 기록이며 새DB현재판정을 뜻하지 않는다. CPU/cu126/cu130/cu132/대형Go와 전체Required·hosted/최종R8은 영향 기반 묶음·최종 검증 대상으로 OPEN이다. S06·S07·C01 잔여·A01/A02도 기존 인계 조건대로 OPEN이다.

## 원복·보존·재개

각 bounded 창의 원14/current3·helper active/running/enabled를 확인했다. 마지막에는 프로세스/컨테이너/volume0, 기존network와 실제firewall규칙, 원개발HEAD·branch·index·tracked diff·SDK불변을 독립 대조했다. 시간 생성 주석만 다른 firewall raw는 편집하지 않고 보존했다. 이는 D4 전체Root백업 검증이 아니다.

새 시험/입력 보존40파일의 manifest `df9c4ab1664cb39990b6d8f8dbcd9f85889839875650b1e2e6d2c33cdb98a066`, stdout 오류 후 Bun 재개65파일 manifest `fe77ca0797360f6797ad06ef0d800006626285d3dc0ffe5dcb5694f4231305da`는 실제 복원/해시·모드 대조 PASS다. 최신 마감 보존·완료 산출물 정리 영수증은 로컬 실행 폴더에 이어 기록한다. 원본/고정 입력/백업/사용자 작업물은 수거하지 않는다.

기계 원장: [r3-local-checkpoint-20261118.json](./r3-local-checkpoint-20261118.json). 실제 source/명령·원본은 `/tmp/haa-r3-context-fix-wosbxnc8/`의 candidate-identity.json, scope.json, native/result.json, native-bun/result.json, native-two-qualified.json, post-native-full-result.json, final-declaration-scope-check.json, native-final-independent/에 있다. D3는 [독립 검토](./r4-d3-independent-review-20261118.md)를 따른다. 재개 위치는 Queue R5 마지막 줄이다.
