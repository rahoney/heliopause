# R2 S09/C01 코드·연결 검증 체크포인트

복구 소스 진입 HEAD는0bb267b60e98b4a51d225a0393ffadfcfaf440e0, 실제 검증한 미커밋 snapshot tree는078da6764276e3d316cc70dfb6133c774f4f84be다. 이 기록은 R2 선별 구현·로컬/표적 실제 연결 범위이며 전체 복구/hosted/큰 프로필 qualification 완료가 아니다. 현재 체크포인트와 향후 commit의 제품 소스 동등성을 확인해도 과거 실행의 Git·바이너리 신원을 바꾸어 기록하지 않는다.

| 항목 | 확인된 결과 |
| --- | --- |
| S09/C01 선별 구현 | 기존9e9260f 및ecf1202/169d61f/c573120/ca3ffe7/3f9d52f, UV Operation·기존 opaque Port 등록·기존 schema 소비자 교정만 선별. Declared/actual/profile/Evidence와 예약/공유입력/마지막 소비자 경계 |
| Terraform 회귀 교정 | c573120 뒤 selector가 검사 Operation 생성 전에 디스크 Acquire를 호출해 invalid reservation. 기존 bounded c.prepared/다운로드를 공유해 중복 사전 디스크 왕복을 제거. 실제 Acquire의 Operation·quota·서명·정적/동적 검사·승격은 유지 |
| 정상 과거 원본 | R1 1967a63/tree6b4af848의 정상4case/8회와 거부 원본을 직접 대조 |
| 새 회귀 | 수정 전 RED31.100s, 관련 package Race35.795s:23부모/90PASS/FAIL0/SKIP0, 신규12case 모두 실행 |
| 필수 검사 | rebuild --full379.356s, docs2.394s, security47.758s, current vulnerability23.116s PASS. 전체 의도 선언+267/-0(기존264+private2+회귀1), ghost0/누락 selector0 |
| 실제 소비자 | 정확13부모/25하위=38Go PASS 이벤트, 각 호출 FAIL0/SKIP0. 일부 동일 scoped source/runtime/profile/Policy 증거만 재사용; JSON에 호출별 원본·재사용 범위 기록 |
| Terraform 실제 | 정상4case×2회95.161s, 거부4case7.766s. 추가 default-init7.489s와 init -upgrade8.126s:필수3/known optional 선언1·current profile/Evidence·새 run·사용자 control·intake REMOVED/Evidence RETAINED 확인 |
| Go/Cargo 실제 | Go public build74.186s, 보안 거부3case61.667s, Cargo library23.545s. 추가 Go tidy25.974s:정확6모듈·현재 approval/cache/승격·control inode/mode 보존; 이 tidy는 build 검증이 아님 |
| 기존 정상·거부 | packaging25 설치/import/승격, 실제 observer 종료 거부, CLI SIGINT 취소와 npm/UV/pipx/Cargo fetch는 관련 코드·runtime·profile 동등성 근거로 재사용 |
| 환경·복귀 | 고정 격리Go1.27.4/동반gVisor6/CPP/local images, 같은 source의 새 CLI/helper/test binary. 기존14 SHA/mode/UID/GID·서비스·3개자산/current 복귀, container/volume/CLI-observer-gVisor0·network4·firewall 동일을 독립 확인. 원 개발 Git/SDK 불변, D4 전체 Host 백업 검증 아님 |

## 최초 오류와 범위

- 원래 Terraform 예약 회귀는 실제 정상·거부·업데이트 경로로 교정 확인했다. 제품 수정은 기존 Terraform3파일이며 새 framework/scanner/정책·지원 축소 없음.
- 첫 full 검사806.730s는 실행기 긴 TMPDIR의109B Unix socket 주소 때문에 실패했다. 해당 시험은358/P3와 동일하며 ci 오답노트의 기존 짧은 root 해답을 적용했다. 동일표적25.103s와 전체379.356s PASS. 제품·시험·한도는 바꾸지 않았다.
- 추가 집계기는 선언4(필수3+선택 partial-facts1)를 총3으로 잘못 가정했다. 기존 CheckPlan.Validate/Inspector/설계21§5.1 및 오답L12와 대조해 집계기만 수정했다. 선택 검사를 필수 완료로 꾸미지 않았고 기존 정상/거부·초기 설치를 반복하지 않았다. 첫 추가 초기 설치의 pre-inode는 미기록이므로 미측정, 업데이트의 새 pre-inode는 실제 기록·대조했다.
- raw·잘못된 집계 결과는 보존했다. 마지막 controller의 과거 eight-parent 상태 label은 실제 실행 수가 아니다. JSON의 정확13개 고유 부모·각 원본과3개의 마지막 부모+2개의 별도 CLI fixture를 확인한다.

## 재사용과 남은 qualification

Source/runtime/profile/Policy가 바뀐 Terraform 결과를 과거 결과로 대체하지 않았다. 수정 후보의 실제 정상·거부·업데이트를 수행했다. 다른 생태계의 unchanged scoped 경로만 명시해 재사용했다. 보존 manifest: source3361 실제복원31b3eae3, source/BIN3·실행안900bd997, 최초 집계/다음 입력fb5748ac; 완료 compiler/static/temp5root2851060982B는 참조/조회오류0 및 목록/해시 보존·복원 뒤 수거했다. 원본 근거는 로컬 경로와 이 manifest가 소유한다.

공통 변경의 현재 PyTorch CPU/cu126/cu130/cu132·대형 Go 및 hosted/일반 Required 전체 qualification은 승인된 K의 의미 있는 묶음/최종R8에서 수행할 OPEN 항목이다. 작은6모듈을 대형 Go로 전용하지 않는다. R3~R8과 D1/D3 적용 전 결정·독립 검토, S06/S07·C01 전체TTL/GC/crash·A01/A02 후속 요구도 그대로 OPEN이다. 이 체크포인트를 R8 또는 전체 보강 완료로 표시하지 않는다.
