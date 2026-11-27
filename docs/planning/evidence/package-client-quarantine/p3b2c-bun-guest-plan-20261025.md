# P3-B2c Bun protected guest supply와 정확한 실행 역할

M13-004 하나만 IN_PROGRESS다. 기준7e4f3da의 전체 release/설치 tuple을 재사용한다.
Read: Queue, PROJECT-DECISIONS, docs router, 확장 계획§2–6, client 설계§1–5,
Architecture inspection/sandbox/policy, Threat isolation, Contract inspection와
P3 baseline/B2b/B2b2 수정 지도, lessons ci L06/L17·sandbox L08/L12/L18·evidence L11/L16.

## 착수 baseline audit

| 책임/근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Bun fixed supply/production native/notices/complete release·installed tuple — 378d1e1..7e4f3da | YES | YES | YES(named local) | NO(whole P3) | guest admission·Bun graph/layout/CLI·전체 qualification |
| protected PNPM source — pnpm_tool_supply.go/readInstalledPNPMTool | YES | YES | YES(PNPM named scope) | YES(PNPM source) | Bun fixed installed member reader |
| native_tool_transport.go / uv_runtime_volume.go | YES | YES(UV/PNPM) | YES(기존 named scope) | NO(Bun) | Bun107,308,936B의 별도128MiB bound/options와 substitution 회귀 |
| observer direct-exec / pnpm_runtime_real_test.go | YES | YES(PNPM) | YES(PNPM) | NO(Bun) | 정확한 Bun 역할/argv·startup/empty lock의 실제 gVisor 관찰 |
| Bun product guest startup·lock | NO | NO | NO | NO | 아래 후속 단계 |

## 수정 지도와 두 번째 검토

1. B2c1: pnpm_tool_supply.go의 exact protected reader와 metadata predicates를
   native_installed_tool.go로 기계적으로 추출한다. PNPM wrapper는 고정 kind와 기존 오류
   identity를 유지한다. Bun wrapper만 compiled Bun asset/size/hash를 선택한다. Caller path나
   PATH 실행, 사용자 pin, implicit runtime provisioning은 없다. before/opened/after inode,
   Root uid/gid/mode·single-link·current/versions 제한·전체 bytes hash·취소·close 오류를 보존한다.
2. nativeRuntimeToolIdentity에 실제 Bun compiled identity를 연결한다. UV/PNPM의64MiB와
   volume options는 그대로이고 Bun만128MiB다. Tar envelope64KiB/exact member/Root
   metadata/whole hash·unique attachment·unknown cleanup 보존을 검사한다.
3. 실제 두 fixed installed consumer가 공통 reader를 필요로 하므로 이 좁은 추출은 기존
   engine 교체가 아니다. 기존 PNPM helper tests는 동일 predicate를 새 이름으로 검사하고
   Bun size와 cross-kind path/hash/volume substitution을 추가한다. Namespace actual은
   별도 mount/network의 실제 pinned bytes로 공급만 검사하며 Host Bun 실행은 없다.
4. B2c2 이후에만 typed executor·container ownership·observer exact role/argv를 추가한다.
   먼저 exact Bun help/empty lock 표본을 관측하고 각 caller/negative를 다시 점검한다.
   Native file 수용은 실행 허가가 아니며 CPU/CONTROL/ARTIFACT 공통 예외를 넓히지 않는다.

## 검사와 CI 운영

B2c1 canonical format→quick/security/docs, 영향받은 sandbox 전체 Race와 actual protected
namespace를 실행한다. source/index/mode 전체 후보를 확인해 중간 commit한다. 현재 remote
9410의 cu132 CI411 due04:22:46UTC를 active turn에서 실제 소비하며 source push/중복dispatch는
없다. Bun 작은 helper마다 CI를 새로 실행하지 않고 실제 runtime/end-to-end checkpoint로 묶는다.

## B2c1 named local 결과

[공급 경로 결과](./p3b2c1-bun-supply-local-20261025.json)는 canonical Quick200.847s/
security12.156s/docs와 전체 sandbox Race22.281s PASS를 소유한다. 실제 PNPM source1.523s,
교정된 Bun source/tar2.932s에서 각각11개 hostile mutation 거부와 Host 보호 파일 불변을
확인했다. 최초 Bun test의 mixed-case 환경변수 SKIP는 미실행으로 보존하고 uppercase 수정
뒤 no-SKIP·actual107MB marker·11개 PASS를 요구했다. Default 검사의 입력과 별도 realcorpus
검사의 범위를 구분한다. Product Bun gVisor startup/lock과 전체 P3는 아직 미완료다.

## B2c2 exact query 착수 지도·두 번째 검토

기준c9a6143은 Bun 공급/transport만 IMPLEMENTED/WIRED/QUALIFIED=YES(named local),
ACCEPTANCE_CLOSED=NO(whole P3)다. 기존 Node boundary, direct-exec one-shot admission,
observer canonical loader와 private native image topology는 구현돼 있다. Bun typed query
container/observer role은 IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다.

이번 gap은 network-none/read-only Node ABI에서 fixed Bun --version/--revision 두 query다.
`bun_runtime.go`는 기존 Node create recipe와 Bun attested runtime volume만 사용한다.
`shared_observer.go`는 Bun의 root/tmp/private9p helper topology를 선언한다. C++ observer는
Bun profile/class/exact binary+execfn+argv0+두 query shape 및 canonical bootstrap 등록에만
연결한다. 기존 native admission latch에 세 번째 실제 consumer를 추가하고 argv 변조도 거부한다.
Bun child에게 Node query authority나 Python 원장·범용 script/임의 argv 권한을 주지 않는다.
Runtime read grant는 기존 immutable loader를 우선 사용하고 실제 최초 거부 없이 다른 runtime의
proc/CPU 예외를 복사하지 않는다. 공통 역할·CPU/시간/resource 한도는 그대로다.

기존 real PNPM fixture의 canonical ordinary user·typed helper·observer start/trace/drain·
mount-only protected installation operator를 재사용한다. Task test client/새 observer는 그
namespace에만 bind하고 원래 client/helper 복원과 Host file/image/container/volume/network/
firewall 상태 대조를 유지한다. Actual query 출력은 version diagnostic이며 Policy authority가 아니다.
Empty bun.lock·graph/설치는 후속으로 남고 정확한 install help를 같은1.4.4 runc에서 먼저 관측했다.
Canonical local/전체 sandbox Race와 모든 기존 observer latch를 확인하며 새 CI는 dispatch하지 않는다.

## B2c2 named local 결과

[Exact guest query](./p3b2c2-bun-query-local-20261025.json)는 Quick199.952s/security13.869s/
docs와 전체 sandbox Race23.534s, 최종 canonical observer114개 latch/228.561s 및 실제
gVisor query5.02s PASS를 소유한다. 처음 version 출력 뒤 overcommit32768 거부, 다음
own PID/maps557056 거부를 각각 보존했고 Bun exact query/direct CONTROL에만 두 읽기를
연결했다. 최종 version/revision·33events/session_complete와 helperwait0·original0/Host 동일을
확인했다. PNPM/UV의 read 조건·다른 PID/flags/clone/artifact 권한은 바꾸지 않았다.

동일1.4.4의 runc install help·빈 manifest 관측은 empty bun.lock을 삭제함을 보여줬다.
빈 파일을 만들어 format 성공을 주장하지 않는다. 비어 있지 않은 실제 lock/source binding과
선택·설치·lifecycle·CLI는 후속이다. CI411은 old9410의 Linux/cu132만 PASS, macOS/Required는
실패이며 이 로컬 Bun source의 원격 증거가 아니다.
