# P2c5-B4b2 — bound payload의 독립 인스턴스 실행

같은 M13-004/P2다. [B4 §3/4/5](./p2c5b4-pnpm-production-inspection-plan-20261024.md),
[B4b1](./p2c5b4b1-pnpm-inspection-payload-plan-20261024.md), Architecture02/03·Domain08/12/13,
artifact L15/L20·sandbox L08/L12/L18·evidence L11/L16을 따른다. B4b1의 재개 검사가 진행 중이며
그 source 검증을 마친 후 이 consumer source를 작성한다. 기존 단일 Backend.Execute는 유지한다.

## 1. Source 작성 전 baseline audit

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| bound layout/정확한 subject·instance 조회 — PNPMInspectionLayout.Target | YES | YES(prepare) | YES(named local) | NO(P2) | 현재 runtime getter와 실행 consumer |
| 최소 payload/copy/project 전체 readback — pnpm_inspection_payload.go | YES | NO(production) | PENDING(B4b1) | NO(P2) | fresh 실행에 연결 |
| 정확한 target command/관찰 profile — pnpm_target_lifecycle.go·SharedObserver.StartPNPMTargetLifecycleProfile | YES | YES(synthetic actual) | YES(local/CI404) | NO(P2) | production 수명 consumer·hashed dirname |
| 생성/volume 신원·sole attachment/anchors/정리 — Backend·closureVolume·pnpm_runtime.go | YES | YES(existing consumers) | YES(existing scope) | NO(P2) | 같은 구현을 새 consumer에서 사용 |
| 불완전 stream의 기존 의심 사실 보존 — retainedELFProbeFacts | YES | YES(Terraform/ELF) | YES(existing scope) | NO(P2) | 같은 함수의 범용 이름·collector owner와 PNPM 소비 |
| fresh instance backend/complete-static gate/aggregate/Host/CLI | NO | NO | NO | NO | backend는 이번 단위, 나머지는 B4c/후속 |

## 2. 정확한 edit map

`artifact/npm/pnpm_inspection_layout.go`에 현재 input의 runtime identity getter만 더한다.
`sandbox/pnpm_instance_backend.go`의 `Backend.ExecutePNPMInstance(ctx, layout, subject, instance)`는
현재 runtime·exact layout.Target·고정 command·payload를 **runtime 작업 전** 확인한다. Zero/foreign/
취소/transport capability 부재는 실행하지 않는다. Run별 Session ID와 fresh PNPM project volume을
생성하고 기존 createArguments에 그 attested mount만 더한다. 별도 native PNPM tool mount나 resolver
network가 없으며 npm/Node는 원래 pinned Node image에서만 실행한다.

기존 exact target profile 시작 → container start → boundary/guest mount anchors → volume identity/
sole writer 확인 → 최소 payload copy/전체 readback → 실제 network-none·canonical null-network 신원/
주소 부재 재조회 → 동일 mount/attachment 확인 → 고정 target command 순서다. 대상은 bound 값에서
도출한 단일 positional path뿐이며 caller가 raw command/path를 전달할 수 없다. Network query는
bounded JSON을 사용하고 실패를 네트워크 부재로 해석하지 않는다.

각 실패 경로에서도 owned container 종료 → trace EOF/partial 사실 수집 → 기존 volume 정리를
순서대로 실행한다. 최초 실행/취소/timeout cause와 뒤 cleanup 오류를 errors.Join으로 보존하고
원래 원인에 해당하는 limitation을 유지한다. Partial stream이면 collector의 accepted kind counts에
기존 `retainedELFProbeFacts` 알고리즘을 그대로 적용한다. 이 함수 본문을 collector로 옮겨
`retainedSuspiciousTraceFacts`라는 공통 이름으로 재사용하며 기존 두 caller도 같은 함수로 바꾼다.
이는 새 parser/collector 대체가 아니며 과거 caller의 완료/관찰/한도 의미는 바꾸지 않는다.
관찰된 의심 사실은 결과에 남지만 incomplete가 completed/ALLOW로 바뀌지 않는다.

`pnpm_target_lifecycle.go`와 C++ `ValidPNPMTargetLifecyclePath`는 기존 검사를 유지하고
codec의 긴 dirname 형식(총120자, 87자 prefix + underscore +32 lowercase hex)을 좁게 연결한다.
Go의 실제 codec vector와 C++ 독립 literal vector, 길이/문자/경로/role/creator/replay negative를
확인한다. 파일명 형식은 control locator일 뿐 source/실행 승인의 authority가 아니다.

## 3. Source 작성 전 두 번째 검토와 acceptance

Host 원본 controls·artifact bytes는 그대로이며 native metadata/사용자 root scripts/config를
Artifact session에 넣지 않는다. Source byte binding은 앞 static producer, 전송 후 filesystem
binding은 B4b1, 실제 관찰은 fresh Session이 각각 소유한다. Bound 값만으로 Policy ALLOW나 Host
설치를 허용하지 않는다. Runtime filesystem 쓰기·network/proc 허용·시간/resource ceiling은 유지한다.

Unit은 exact subject/instance·zero/취소/transport·runtime 거부의 무작업, fresh session/volume,
readback drift/foreign writer/network query 실패 시 target 미실행, 실제 원인과 cleanup 동시 실패,
complete/partial 의심 사실 보존·각 경로의 정리를 검사한다. Canonical format/quick/security/docs와
npm/sandbox 전체 Race 뒤 C++ 전체 latches와 실제 transitive/peer·controlled script failure를 묶는다.
기존 actual용 private runtime은 소실됐으므로 보존된 operator/lock과 실제 설치 신원부터 복원한다.
Host 운영 설치/daemon·정책을 영구 변경하지 않는다. 원본과 성공 범위는 새 prefix로 보존한다.

이 backend의 named local qualification 뒤에도 static-all 수집 중 자원 gate, expected instance
coverage와 실제 Session 결합/partial 집계, provenance/cache/Host records/transaction/CLI 및 전체
P2 qualification은 MISSING이다. P3로 먼저 넘어가지 않으며 작은 helper별 원격 CI는 반복하지 않는다.

## 4. 첫 연결 검사의 범위

첫 canonical format0.576s는 PASS이고 Quick31.981s는 production build 뒤 test-build에서
기존 Terraform test 두 곳의 옛 helper 이름 참조를 확인해 멈췄다. New backend unit은 아직
실행되지 않았다. 공통 helper 본문/기존 검증 의미는 그대로 두고 두 test caller의 이름을
맞춘다. 해당 `terraform_provider_test.go`도 명시 projection inventory에 포함한다.
Raw `p2c5b4b2-first-quick`은 첫 실패를 보존하며 후속 성공으로 덮어쓰지 않는다.

Caller 교정 뒤 Quick145.736s는 새 long-name fixture의 독립 vector에서만 실패했다.
100자 이름의 escaped filename은120자 이하라 codec이 정상적으로 hash하지 않는 입력이다.
실제120자 초과인130자 이름과 독립 SHA256 vector로 Go/C++ fixture를 맞춘다. Production
codec/locator를 이 fixture 오류에 맞춰 바꾸지 않는다. Fresh backend의 같은 default suite
실패는 없었으며 전체 profile은 FAIL로 보존한다.

130자 vector 교정 뒤 canonical format0.571s/Quick56.309s/security11.308s/docs0.229s와
npm/sandbox 전체 Race21.443s가 PASS다. Fresh backend의 구조/negative·first cause/partial
관찰 보존은 이 local 범위다. C++/actual과 whole P2는 아직 완료하지 않았다.

Observer canonical release build의 첫2.143s는 retained local partial clone의 promisor blob
부재로 source 준비에서 실패했다. Commit 존재 조회 성공은 전체 source 공급 증명이 아니다.
CPP compile/latch는 미실행이며 첫 원본을 유지하고 override 없는 canonical 공식 저장소의
동일 locked commit으로 fresh build를 진행한다. Product source/lock·검사 조건은 바꾸지 않는다.

공식 source의 canonical observer release build241.229s와 전체 latches는 PASS다. Actual consumer를
추가한 최종 Go source의 format0.641s/Quick59.301s/security8.529s/docs0.232s와 전체 관련
Race20.888s도 PASS다. Realcorpus compile5.641s와 정확한 한 test 이름 존재를 확인했다.
실제 실행 결과는 아직 별도다.

## 5. Actual 입력과 재부팅 후 운영 전제

보존 session에서 원래 `pnpm-public-namespace.py`/prepare operator의 실제 source를 복원해
동일 mount-only 구조를 재사용한다. Host Docker/network namespace는 그대로이며 private
`/usr/lib` overlay의 PNPM 공급과 보호 client/observer bind만 제시한다. Canonical 설치 Go1.27.3
gVisor bundle은 다시 빌드/활성화하지 않는다. 정확한 PNPM archive를 lock의 size/SHA256과
canonical materializer로 확인했고 Root-owned 새 입력을 준비했다. PNPM bytes는 Host에서
실행하지 않는다.

재부팅 뒤 두 `/run` directory가 없어 `.github/workflows/heliopause-ci.yml`의 canonical
runtime-directory 설정(Observer0700/UID1000, network-policy0755/Root)을 재현한다.
기존 상태가 있으면 채택하지 않고 거부하며 이번 생성 inode만 complete helper 종료/빈 endpoint
확인 뒤 rmdir한다. Source를 고쳐 missing socket을 우회하거나 Host daemon/config를 바꾸지 않는다.
전후 보호 파일·image/network/container/volume·bridge/service·방화벽과 native 설치 부재를 대조한다.

Public transitive2/peer4는 기존 실제 acquisition/SRI/static producer → production prepare → 모든
fresh instance backend를 소비한다. 별도 controlled success/exit37은 명시적 structural fixture의
script이며 공개 package 인증 증명이 아니다. 이 두 범위를 합쳐 forged fixture를 실제 public
인증/Policy ALLOW/Host 설치로 확대하지 않는다. Native 준비·각 실행의 complete stream과 실패
관찰 보존, canonical helper wait0 및 정리가 실제 확인돼야 이 consumer 범위를 qualified로 닫는다.

첫 outer preflight는 data-only materializer 출력에도 Root 실행파일 소유권을 요구해 실행 전에
거부했다. 원래 준비 operator처럼 그 정확한 UID1000 data의 size/SHA를 검사하고 namespace의
Root0555 version에 복사한 뒤 제품 supply reader가 다시 검사하도록 역할을 맞췄다. Host control
실행파일의 Root/non-write 조건은 유지한다. Role-correction 원본은 NOT_EXECUTED다.

첫 actual은 public transitive2를 fresh complete session으로 통과한 뒤 peer fixture lookup에서
FAIL했다. 새 operator가 test compile에 더한 trimpath가 기존 runtime.Caller 기반 fixture 경로를
module path로 바꾼 것이므로 제품/codec을 바꾸지 않고 기존 non-trimpath actual test build를
재사용한다. Controlled 두 case는 미실행이다. Canonical helper wait0·빈 endpoint·이번 생성
runtime directory 부재 복원과 보호 Host 설정 보존을 확인했다.

Outer writer는 nested result와 final result의 같은 filename을 사용해 실제 실패/정리 뒤
AssertionError였다. 원본 namespace 결과를 final operator PASS로 해석하지 않는다. 두 결과
이름을 분리하고 fresh workroot로 재개한다. 첫 firewall snapshot 차이는 live chain counters뿐이며
원본을 유지하고 후속은 전체 raw capture와 counter만 제외한 rule/config 비교를 구분한다.
`p2c5b4b2-instance-first-scope-recovery.json`이 이 실제 부분 성공과 operator 실패 범위를 소유한다.

Fresh cwd-compatible actual58.306s/test55.93s는 public transitive2/peer4 전체 instance의 최소
payload/readback·새 session·complete 관찰·금지0을 통과했다. 별도 controlled 정상/exit37의
ARTIFACT shell3/2와 최초 exit37·실패 결과의 관찰 보존도 확인했다. Canonical helper wait0,
두 `/run` 생성 inode 정리와 보호 Host 구성 전후 일치(방화벽 live counters 별도 capture)를
확인했다. Raw `p2c5b4b2-instance-cwd-result.json`과 actual/namespace/firewall captures가 범위를 소유한다.

CPP/Go vectors 외 hashed dirname의 실제 consumer coverage를 닫기 위해 같은 controlled
fixture helper에 명시130자 package name만 추가한다. 기존 기본 fixture는 이름 is-number로
동일하게 사용하며 production codec/input API는 바꾸지 않는다. CPP source/실제 public
acquisition·prepare 소비자와 앞 두 controlled case의 의미는 그대로다. 새 binary에서는
`controlled-long-name`만 선택해 actual 검증하고 별도 범위를 기록한다.


최종 named fixture source의 format0.738s/Quick77.620s/security8.881s/docs0.250s와
npm/sandbox 전체 Race21.721s가 PASS다. Non-trimpath actual binary7.239s compile 뒤
새 workroot의 controlled-long-name만10.410s/test7.86s로 통과했다. Hash locator의 실제
CONTROL→ARTIFACT script3·complete trace·다른 금지0, helperwait0/CONFIRMED_STOPPED,
owned runtime directory 정리와 보호 Host 구성 전후 일치다. 앞 public6/controlled2는
앞 binary의 별도58.306s 결과이며 이번 한 case에 합쳐 같은 실행으로 주장하지 않는다.

B4b2는 IMPLEMENTED=YES/WIRED=YES(actual direct consumer)/QUALIFIED=YES(named local),
ACCEPTANCE_CLOSED=NO(whole P2)다. [로컬 영수증](./p2c5b4b2-pnpm-instance-backend-local-20261025.json)이
source/check/actual 범위를 연결한다. 다음 B4c의 static-all 수집 중 gate·실제 Session coverage와
partial aggregate, 이후 Host/CLI와 묶음 remote qualification은 MISSING이다.
