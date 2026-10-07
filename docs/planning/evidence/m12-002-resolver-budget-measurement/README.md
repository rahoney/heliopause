# M12-002 — Go resolver 처리량 측정

2026-10-04, source baseline `87514a5117f39530d6a047fe1f4032b1dcbde351`.
[Result receipt](./result.json)은 실제 입력·원본 로그 hash·측정 범위를 소유한다.
이 결과는 최종 제품 후보 qualification 또는 M12-002 acceptance가 아니다.

## 측정 방법과 단위

제품 source·기존 profile 예산·관찰 판정·Policy는 변경하지 않았다. Exact pinned
gVisor source에서 별도 LOCAL observer를 빌드하여 신규 `go-module-resolver`만
500,000 charged records의 유한한 측정 상한을 사용했다. 완료 시 fixed scalar
counter와 observer `getrusage`를 기록하고 Docker/cgroup 자원값을 외부에서 읽었다.
Raw syscall payload를 저장하거나 artifact text/path를 승인 authority로 사용하지 않았다.

각 명령은 fresh pinned Go container와 operation-private cache에서 실행했다.
컨테이너 CPU 1개 분량, memory 512 MiB, tmpfs 512 MiB, pids 64, 주요 Go 명령
120 seconds를 유지했다. Resolver를 구성하는 `get`, `mod download -json all`,
`mod graph`를 네 public fixture에서 각각 세 번 실행했다. 총 36 source-command
connections는 실제 registered kernel observation과 authenticated helper를 사용해
완료됐다. Host command나 cache-only 실행을 이 결과로 대신하지 않았다.

`charged`는 기존 helper의 `normalized_records + immediate_records`다. Raw input
frame 수, collector가 받는 normalized record 수와 bytes는 다른 단위다. 기존
workspace summary count의 10,000 saturation도 그대로다. 모든 raw frame 수를
10,000 또는 200,000 charged quota와 같은 숫자로 취급하지 않는다.

## 실제 정상 source 처리량

| Fixture | Get charged | Download charged | Graph charged | Container cgroup peak |
| --- | ---: | ---: | ---: | ---: |
| pflag 1.0.9 | 16,726–16,728 | 2,195 | 1,690 | 48.2 MiB |
| Cobra 1.10.2 | 18,036–18,066 | 3,852–3,874 | 1,760–1,762 | 51.7 MiB |
| x/tools 0.49.0 | 17,806 | 19,941–19,966 | 1,690–1,692 | 94.0 MiB |
| gRPC 1.76.0 | 58,260–58,288 | 62,974–63,018 | 1,749 | 296.0 MiB |

Source matrix observer peak RSS는 10.4–10.5 MiB였다. 전체 CLI 실험까지 포함한
observer peak RSS는 34.1 MiB, 가장 큰 container cgroup peak 관측은 296.2 MiB,
가장 큰 sampled 단일 Host descendant RSS는 83.9 MiB였다. 이들을 동시에 발생한
전체 Host peak로 합산하지 않는다. Container CPU usage는 teardown 전 마지막
sample이므로 lower bound이며, `time -v`의 parent/reaped children usage는 Docker
daemon이 소유하는 sandbox CPU 전체를 포함하지 않는다. 모든 sample에 같은
resource limits가 적용됐고 sampling 오류는 없었다. 이 범위에서는 container
memory/CPU/time limit을 늘릴 근거가 없었다.

## Source 완료와 전체 CLI 승인 구별

원래 `TestLinuxGoGetDownloadIntegration` binary는 기존 10,000 한도에서 실제
FAIL했다. 같은 binary의 exact fixture는 별도 observer에서 두 번 Get와 retained
managed Download를 완료했다. Cobra도 전체 CLI에서 7개 module의 selection,
acquisition/SumDB/static inspection, entry/set ALLOW, cache/control publication과
retained Download를 완료했다.

x/tools와 gRPC는 source command 완료 뒤 전체 CLI의 set Policy에서 BLOCK됐다.
Typed `ProjectPolicyFailure`는 각각 8개·41개 entry의 completed facts를 보존했고,
blocked entry는 x/tools 0.49.0·0.34.0의 `M12_GO_SOURCE_INVALID`다. 두 subject의
envelope digest를 같은 `.mod`/`.zip` bytes에 직접 대조한 뒤 pinned parser의
동일 flags로 malformed `testdata` Go assets를 확인했다. 0.49.0은 1,283 Go files
중 invalid 11개 모두 `testdata`에 있었고, 0.34.0은 1,233 Go files 중 invalid
16개를 확인했다. 이 파일은 데이터로만 읽었으며 import/execute하지 않았다.

따라서 이 두 전체 CLI invocation은 FAIL 그대로 기록한다. 처리량 증가가 모든
module의 검역 승인을 보장하거나 이 BLOCK이 악성임을 입증한다는 뜻은 아니다.
일반 source와 의도적으로 invalid한 테스트 자료의 syntax 검사 범위는 별도
M12-002 compatibility gap이다. Package/vendor allowlist, test skip, Policy 완화나
failure-to-success 변경은 적용하지 않았다.

## Go 전용 후보와 남은 검증

신규 resolver helper의 연결별 **200,000 charged records**를 후보로 제안한다.
측정 최대 63,018의 약 3.17배 여유이며 통계적으로 증명된 최적값이나 모든 가능한
graph에 대한 보장은 아니다. Collector의 별도 10,000 records/2 MiB와 기존
CPU/CUDA/npm/PyPI/GitHub 예산, container resource limits는 보존한다. Go build와
Cargo의 작업량은 이 source 측정으로 정의하지 않는다.

원래 charging과 guard를 사용하는 exact remote-sink synthetic probe에서 측정용
Go 500,000과 후보 Go 200,000의 상한 아래 완료 및 추가 event의 `EVENT_LIMIT`
차단을 확인했다. 기존 npm은 여전히 10,000에서 차단됐다. 이 boundary probe는
실제 kernel-observed normal workload와 구분한다. 별도 200,000 후보 observer에서도
gRPC source의 세 명령은 charged58,293/63,018/1,749로 완료됐고, 원래 actual
Get+retained Download positive CLI gate도 같은 200,000 후보에서 완료됐다. 제품 source의 기본 Go 한도는
아직 10,000이며 후보 한도는 LOCAL experiment에만 존재한다.

최종 제품 적용·consumer qualification, syntax compatibility 교정, observed offline
build/output publish, 전체 M12-002 acceptance는 남는다. M12-003/004는 아직
NOT_STARTED이며 기존 M12-001의 remote PASS를 이 후보에 전용하지 않는다.
NO PUSH / NO REMOTE CI / NO MERGE.
