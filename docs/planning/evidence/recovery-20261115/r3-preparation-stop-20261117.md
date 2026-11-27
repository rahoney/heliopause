# R3 선별 복구 — 빌드 준비 2회 실패 후 중단

R2는 `607d79f522f38d55dc005eba1a63eb800f9383f1` / tree `d7f94049851ae24525af99d9117cff91e836440a`에 보존됐다. Terraform의 예약 없는 조기 Acquire는 기존 bounded-memory 본문/캐시 재사용으로 좁혔고 정상4×2·거부4·실제 init/upgrade·정리 및 관련 Race90을 확인했다. R2의13부모/25하위38PASS는 각 실제 호출·유효 재사용 범위이며 R8 qualification이 아니다.

R3는 기존 S01/S08 117파일을 선별 적용한 미커밋 후보다. 진입은 위 R2이며 제품 projection tree는 `ac2c1b26cf79d93a4580b6377b36fd7fa12ff882`(Queue/후속 상태문서 별도)다. 추가188/삭제19 선언을 적용 전에 기록했다. 정상·거부 단위19부모/36하위55PASS/FAIL0/SKIP0(48.801s); 빌드/시험빌드/vet/실제 CI selector는 통과했다. 점검 도구는 358과 바이트가 같은 두 historical JSON의 옛 시험명을 현재 caller로 오인했다. 원본/제품 시험은 변경하지 않고 정확2경로+SHA만 기록 참조로 분류했다(원본2·변조2·active2 대조6PASS). 전체 mandatory --full은 아직 미수행이다.

| 비교 대상 | 과거 성공 원본 | 현재 사실 |
| --- | --- | --- |
| 기존 성공 기준선 | 358adf9/tree e1cac135, CI448~451 | 원본/pure 작업공간 유지; 새 S01/S08의 현재 성공 증거로 확대하지 않음 |
| 동일 S08 patch c8a546d0 | 원본 s08-filesystem-runtime-coherent-bundle-v4: coherent6/capture/strict86.563s; kernel7+CPP134/254.002s. 당시 Go1.27.3 | source는 재사용하되 승인된1.27.4로 새 bundle 필요. 과거 바이너리 재사용 불가 |
| 현재 승인 SDK의 성공 빌드 | 별도 Go1.27.4 strict884.226s, 같은 builder9af/Bazel8.3.1/upstream7c619980, 이전patch588adad4. compile network bridge | 새 patch c8a546d0, network none. 이 강화된 실행 환경은 과거 성공 환경과 다름 |
| 준비 실패1 | 기존 캐시에는 payload와 id-표식이 함께 있음 | 6.523s/exit37. payload SHA는 맞았으나 operator가24개 id-표식을 누락. 기존 정확 표식을 추가했고 rules_go 로드까지 진행됨 |
| 준비 실패2 | 성공 빌드는 Go SDK 목록을 조회할 수 있었음 | 12.680s/exit48. sdk.bzl fetch_sdks_by_version은 checksum 없이 go.dev/dl 목록 조회. 목록 bytes는 로컬에 있어도 이 URL 조회를 오프라인 CAS만으로 충족하지 못함 |

두 번 모두 compiler·kernel/CPP·제품 실제 시험 전 준비 실패다. 제품 회귀를 입증한 결과가 아니며 미도달을 PASS로 세지 않는다. 같은 오프라인 접근2회 실패 정지선에 따라 세 번째 빌드는 실행하지 않았다. 현재 runtime member 해시는 이전 값이 남은 DRAFT이므로 이 후보를 설치/실행/승격하지 않는다.

최소 재개안은 **기존 로컬 Docker builder의 compile 준비만 과거 성공과 같은 bridge network로 되돌리는 것**이다. 새 VM·클라우드·Host 설치/런타임 교체가 아니다. 정확한 명령은 `/tmp/haa-r3-runtime-20670k5d/resume-execution-proposal.json`이다. Go1.27.4/공식 SDK archive7226736c, upstream7c619980/new patch c8a546d0, builder9af/Bazel8.3.1, 같은 CPU4/메모리8GiB/전용디스크8GiB/compile22분 정지선은 유지한다. 실제 SDK 목록/선택 버전·archive digest와 생성 runtime6/manifest를 확인하고 어긋나면 사용하지 않는다. kernel/CPP는 network none을 유지하며 kernel은 원본의 정확7개를 복원한 filter/3target으로 검증한다. 계획만 제출하며 별도 확인 전 실행하지 않는다.

보안 계약·지원/시험·한도 변경은 없다. npm40k를 자동 재적용하지 않았고 C/C++ AST·Semgrep·S04·D1/D3도 도입하지 않았다. 원래 개발 브랜치4a43f2c/index/제품diff/Host Go1.27.3과14개 설치 파일·current/기존3개 공급물은 unchanged; helper active/enabled, CLI/observer/gVisor·container·volume0을 root 읽기 전용으로 확인했다. 제품 Host activation/CI/push/merge/main 변경0이다.

재개에 남는 범위는 새 runtime의 capture/정확 pin 생성, 전체 mandatory/relevant Race, 실제 영향 소비자의 정상·거부/취소·정리와 K 묶음 검증이다. b91fe64의 Linux 전용 adapter 함수 선별 이동과 eafb082/3b2829b의 consumer 연결도 해당 실제 호출 전 함께 처리해야 한다. 전체171 감사나 복구 계획23 수정은 하지 않았다. R8의 동일 후보 Required·CPU/cu126/cu130/cu132·대형Go는 OPEN이다.

원본·source/의도·별도 wrapper 준비 오류·캐시 원인·실제 명령/자원은 `/tmp/haa-r3-environment-d7puheoo/STOP.json` 및 `/tmp/haa-r3-runtime-20670k5d/{first-cache-identity-error,second-sdk-catalogue-error}`에 있다. 실패 후보 소스와 고정 입력은 보존하며 종료된 파생 캐시는 보존·참조 확인 후 수거한다. 보존 영수증과 정리 결과는 같은 실행 root의 JSON을 따른다.

정리 결과: 참조/조회오류0·보존264파일실제복원82d2229d 후 전용8root834,740,119B 수거. source/raw/SDK/고정CAS는 유지하고 파생 bazel symlink 잔존은0개다. 기존 개발 작업물/백업 삭제0이며 Windows 물리 회수는 측정하지 않았다. 이 실행 root의 이전 검사/빌드 wrapper는 캐시·TMP가 수거되어 재호출하면 안 된다.
