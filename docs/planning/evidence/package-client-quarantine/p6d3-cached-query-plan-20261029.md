# P6-D3 — 고정 native UV cached query의 제한된 연결 전 감사

Baselinea2213dd2a01e090489e1dc77dadaeb2e732476a3/tree7b33f92c8b1eb3c2669f6f6e27a421827b48f555,
3071 tracked/runtime1370이다. M13-004 하나만 active다. D2 네 반환 오류 보존의 named local
qualification과 D1 관찰기 FAIL은 구분한다. CI433/source87c817d는 frozen이며 actual poll03은
13:02:10UTC8SUCCESS/Linux 진행, next13:22:11.775383UTC다. Worker71198/PID3172981을 유지한다.

Read: PROJECT/Queue/docs routes와 lessons INDEX·artifact L01/L02·sandbox L03/L08/L12/L18·
evidence L16·ci L05/L06, client09§3·plan20§5, D1/D2 exact source/raw/receipts, observer.cc의
GroupState·UVToolQueryEnvironment·IsExactUVPythonQueryProgram/Query·HasExactUVQueryCreator·
restricted workspace/Clone/exec transitions 및 기존 VerifyUVPythonQueryBoundary 전체, canonical
observer release build/source preparation·fixed lock을 대조했다. 고정 UV0.13.4 commit7456b60의
environment::from_resolution→Cache::persist→ArchiveId::new→fastid::Id::secure 원본3개의
Git blob SHA1/full UTF8/readback을 raw p6-d3-uv-cache-source-inventory.json으로 확인했다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| UV base/tool-home query의 고정 image/argv·kernel clone·ARTIFACT/stdlib/config 제한 | YES | YES | YES (기존 CPP132/actual/CI432) | YES (named) | 기존 의미 보존 |
| Native fresh cached interpreter alias 인식 | NO | NO | NO | NO | 좁은 fresh TMP/archive-v0 grammar·topology |
| Cached query의 private helper root·env DATA 읽기와 TMP taint | NO | NO | NO | NO | 같은 parser가 검증한 helper root 결합·readonly 범위 |
| Cached positive/negative·기존 creator/clone/reexec 회귀 | NO | NO | NO | NO | 기존 UV named latch 확장 RED→GREEN·actual |
| 네 admission method 최초 오류 보존 | YES | YES | YES (D2 named local) | YES (named) | 수정하지 않음 |
| Native public golden/pre-query whole binding·aliases/CLI/final | NO | NO | NO | NO | 이 DATA 진단과 별도 후속 |

## 근거와 한계

Native stderr는 /tmp/.tmpXXXXXX/archive-v0/ID16/bin/python 조회를 기록했고, 현재 query owner는
base 또는 /haa-site/tools/PROJECT/bin/python만 인식한다. 첫 fault의 pinned Python/libpython
FNV 후보는 보조 진단이며 이름/지문을 권한으로 쓰지 않는다. 실제 actor 수용 여부는 새 exact
kernel predicate·negative latch와 전체 actual trace로 검증한다. Native EACCES나 stderr를 완료
authority로 쓰지 않고 source/query/readonly/외부 observation·cleanup을 함께 요구한다.

고정 Cache::persist는 ArchiveId::new를 사용하고 secure ID는 ASCII `_ - 0..9 a..z A..Z`16bytes다.
다른 from_digest/installed/shared cache 형식은 허용하지 않는다. 이 문자열 grammar만으로 코드를
인증하지 않으며 고정 UV image·실제 consumed parent/group/start·sealed topology가 필수다.

## Source 전 exact map과 두 번째 검토

- tools/gvisor-observer/observer_latch_test.cc의 기존 VerifyUVPythonQueryBoundary에 cached
  positive와 길이/문자/다른 alias/normalization·root/tmp/metadata mount shadow·creator/role/
  pending/generation·clone/reexec·쓰기/다른 env/.pth/customization/package body 거부를 추가한다.
  기존 named latch/132개와 다른 소비자 회귀를 보존한다. 현재 production source로 새 positive가
  실패하는 canonical CPP RED를 먼저 보존한 뒤 predicate를 수정한다.
- tools/gvisor-observer/observer.cc의 기존 UVToolQueryEnvironment에 fresh cached branch만 추가한다.
  정확한 /tmp/.tmp+alnum6/archive-v0/ID16/bin/python과 root RO·고정 resolved Python·TMP tmpfs/
  noexec/nosuid/실제 anchor1·nested shadow 거부를 결합한다. 다른 환경·profile은 활성화하지 않는다.
- IsExactUVPythonQueryProgram의 기존 한 parser가 이미 검증한 private module root를 optional
  output으로 반환한다. 기존 bool/기존 caller 의미는 유지한다. Query의 kernel exec 시 이 root를
  private GroupState에 보존하고 clone에는 taint만 승계하며 active 권한은 새로 만들지 않는다.
- Cached query만 TMP 전체를 제한한다. 같은 exact creator/active ARTIFACT query가 읽는 검증된
  private helper module root와 exact env directories/pyvenv.cfg만 기존 readonly flags로 수용한다.
  그 밖의 TMP·다른 env·package/startup files·쓰기·shadow/alias는 general workspace보다 먼저
  Outside로 분류한다. 기존 base/tool-home query의 의미와 고정 stdlib/proc ABI는 유지한다.

Runtime1370 membership 유지/CPP2 modified·기존1368 bytes unchanged를 예상한다. Go/Core/CLI/
wire protocol·runtime pins·budget/Required/timeout은 변경하지 않는다. 문자열/comm/cache name은
identity가 아니며 CONTROL/network·general FS/import grant를 만들지 않는다. 현재 진단의 명령은
계속 실행 불가능한 /dev/null이다. 실제 package command를 native UV CONTROL 경로에서 실행하지
않는다. 이후 public native selector에는 whole original/startup gate와 source binding이 별도로
필요하며 이 fixture 성공으로 public authenticity/Policy나 native cache 지원을 선언하지 않는다.

## 검증·정리

Canonical CPP RED/GREEN release build와 전체132 latch를 실행한다. Go canonical format/Quick·
sandbox 전체 Race/tagged vet/pinned Staticcheck·두 test build/docs/security로 공용 소비자를 확인한다.
실제 network-none D1 준비 probe가 native EACCES+완전한 trace/정리로 PASS하는지 확인하고,
같은 후보의 기존 canonical current Python public normal0/nonzero2를 재검증한다. 기존 base/tool-home/
Pipx native boolean query helpers와 Node/PyPI/네 torch profile은 원래 CPP latch·profile negatives와
기존 predicate 의미 보존을 함께 대조한다. 대표 actual을 whole P6/네 final profile 지원으로 쓰지 않는다.
실패는 최초 source/log를 동결한 뒤 그 exact ABI만 재검토하며 permission/CPU/trace 한도를 넓히지 않는다.

Source/Host5/42/ZIP/Docker·전체 namespace/volume/helper·wait0/원본을 보존한다. 완료 복사본은
실제 root proc 참조/known resource-empty zombie 신원을 확인한 뒤 삭제한다. CI433 실행 중에는
새 push 없이20분 실제 GET을 소비하고 다음 의미 있는 grouped checkpoint에서만 hosted를 요청한다.

## Canonical CPP RED 확인

Production observer.cc ed5a2bd2는 그대로 두고 기존 UV latch54cases에 cached66을 추가한
120case source를 canonical release build로 컴파일했다. Actual275.246s/exit1은 새 긍정3
cache-normal/id-alphabet/helper-other만 거부했고 compile/setup failure가 아니다. 다른131 named
latches는 보존됐으며 전체132를 Green에서 다시 요구한다. Failed inputs /tmp/helox-release-observer.vweLjb와
full log/source/readback을 동결했다. 이제 기존 owner에만 map대로 cached branch/private root
binding·TMP readonly 제한을 추가하고 같은 latch SHA를 유지해 canonical GREEN을 검증한다.

## 첫 후보 실패와 진단 경계

첫 permission 후보의 canonical CPP309.977s/exit1은 같은 cached 긍정3에서 실패했고
다른131 named latch는 PASS다. Go canonical9는 별도로 Quick324.179s/전체 sandbox
Race33.364s/tagged vet·pinned Staticcheck·두 test build/docs/security까지 PASS다.
이는 CPP나 actual 성공이 아니다. 첫 후보 observer.cc/latch·build 원본/Go identity를
raw p6-d3-first-green-original-inventory.json으로 보존했다. 신원 generation mismatch의
기존 FindFilesystemGroup은 null→UNKNOWN을 반환하므로 새 mutation2의 Outside 기대가
잘못됐을 가능성을 진단한다. 모든 assertion/권한을 보존한 test-only line 진단을 canonical
빌드로 확인한다. 첫 진단 실행은 sandbox DNS 실패로 compile 미도달/exit128이며 별도로
보존했고, 허용된 네트워크 문맥의 고정 source build로 이어간다.


진단 자체는 lambda 밖 초기 socketpair 오류에도 scenario 변수를 참조해 compile error였고
named latch0/actual 미도달이다. Driver의 substring은 compiler가 인쇄한 source에도 일치하므로
진단 실행으로 해석하지 않는다. Original inventory에 full diagnostic source/log/receipt/producer를
보존했다. 기존 FindFilesystemGroup의 start mismatch→null 및 ClassifyFilesystemOpen의
null→UNKNOWN이 새 mutation2 oracle의 오류를 직접 설명한다. Diagnostic 출력을 제거하고
원래 RED assertions에서 해당 기대값만 UNKNOWN으로 바꿨다. 다른 긍정/negative·production
source6e934ebb는 유지하며 새 canonical CPP132/UV120을 요구한다. 최초 RED와 수정 후 latch의
SHA는 다르므로 같은-test-SHA GREEN을 주장하지 않는다.


## V2 교정의 한계와 V3 exact image guard

Stale-start 기대값 교정 뒤 canonical CPP279.127s/exit1도 같은3 positives에서 실패해,
이 oracle 수정만으로 해결됐다는 해석은 반증됐다. 모든 원본은 p6-d3-v2-oracle-original-inventory에
보존했다. Lambda 내부만 계측한 세 번째 진단290.293s는 실제131 named PASS와3개의
source_line7924/pyvenv.cfg expectedOutside→actualHelperOnly를 기록했다. 기존
HasExactNativeQueryCreator는 creator/role/start/provenance/consume/pending을 검사하지만
query의 current_image_class는 검사하지 않는다. 새 cached block의 mutation5 Unknown
이미지에도 cfg를 수용하는 빈 조건을 확인했다. 기존 IsPinnedUVPythonQueryRead와 같은
current_image_class==Python을 cached TMP read에 추가하는 제한 강화만 수행했고,
진단 출력을 제거한 v2 corrected latch SHA79d50846은 그대로 유지했다. V3 전체132/120과
실제 trace가 PASS하기 전 해결·actual 성공을 선언하지 않는다. Baseline의 Go compiler
입력은 전부 동일하고 observer.cc를 직접 읽는 Go tests는 sandbox 전체 Race로 재검증한다.

CI433은 이후 별도 final query에서 Required/all10SUCCESS/동일 checkout980e3509·tree238aa647,
104parents/CPP132·full CPU290.07s/new boundary8·hostile5·앱3 및 helper17wait0/final4 original0을
확인했다. Full decoded1317349B/59chunks를 동결했고 exact owned UID1000/PID3172981만
SIGTERM해 session143으로 종료했다. Next timer는 없다. Qualification은 source87c8에 한정하며
D1/D2/현재 D3를 포함하지 않는다. CI 기록 commit1c9f4de는 메타데이터2파일뿐이다.


## V4 집계 oracle·고정 replay와 완료 캐시 정리

V3 canonical266.793s/exit1은 더 이상 current-image mutation에서 막히지 않고 실제
restricted workspace dispatch outside1/findings1/workspace0과 새 테스트의 workspace1 기대가
충돌했다. Cached outside 접근은 workspace 사용으로 집계하지 않는 기존 의미가 맞다.
V4는 /tmp/query-write의 새 cached 경우만 workspace=false로 교정하고 legacy=true를 유지한다.
Production f5d6881a는 그대로다. 원래 V3 source/log/Go identity 및 compiled observer/latch
ELF 두 개를 별도 원본으로 동결한 뒤, 완료한 V3 build workspace의 latch source만 바꾸어
canonical script에서 추출한 동일 고정 Docker/Bazel 명령으로 supplemental replay했다.
47.352s/exit0/132 named PASS이며 최종 새 canonical release build는 별도 진행한다.
Replay를 fresh release build 또는 native actual로 쓰지 않는다. Raw v3-failed inventory와
v4-pinned-replay 원본이 이 구분을 소유한다.

완료 실패/replay 전용 .cache/bazel6의 generated entries만 root101proc/컨테이너1/
참조0/query errors0·정확한 resource-empty zombie6 검증 뒤 제거했다. Failed gVisor/CPP
regular source25452개 aggregate/원본 pair를 전후 대조해 전부 보존했고 raw/실패 compiled
ELF·새 active canonical build/shared cache/SDK/고정42/ZIP/Git은 유지했다. 제거 논리 크기는
3288367840B이며 동시에 새 build가 진행된 동안 /tmp free의 관측 순증은3860549632B다.
누적 ELF217개+전용 cache6의 논리 크기는 약10.32GiB이며 Windows VHD 물리 축소를
주장하지 않는다. 수량·개별 원본은 기존 completed-test-artifact-cleanup leaf가 소유한다.

## V4 actual checkpoint와 남은7findings

Fresh canonical315.655s/exit0/132·UV120과 동일 Go 입력의 canonical9 및 v3 affected sandbox Race22.668s가 PASS다. Native DATA parent4.66s/Go0은 expected /dev/null EACCES와 complete23events/2689B·정리를 확인했지만 outside7의 실제 대상/분류는 미확정이다. 이를 clean public Policy 성공으로 쓰지 않고 D3 acceptance는 OPEN으로 유지한다. 같은 후보의 canonical public Python2/73.35s는 fresh whole ALLOW·complete0/2와 정리가 PASS다. 모든 원본 보존 뒤 root83proc/ref0/errors0/정확한6zombie 검증으로 완료 사본4개85924576B를 제거했다. 다음 test-only observer 진단은 outside7의 actor/path/flags를 DATA로만 수집하며 승인/wire/권한을 바꾸지 않는다.
