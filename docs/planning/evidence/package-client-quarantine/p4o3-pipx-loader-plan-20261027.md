# P4-O3 — immutable pipx candidate와 isolated loader

M13-004의 O2 local receipt를 이어가는 baseline이다. Read: Queue, plan20 §5·6,
client09, D-012/013/015, maintenance §2·3·6·7, P4-O/O2, 관련 lessons.
공급 검증 결과와 runtime admission/제품 지원을 분리한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Whole wheel/proof/metadata/graph | YES | YES(developer O2) | YES(local) | YES(O2 only) | held bytes를 후보 생성에 전달 |
| Bounded immutable closure facts | YES(PipxSupplyClosure) | YES(O2) | YES(local) | YES(facts only) | manifest와 archive bytes 재대조·결정적 ZIP |
| Atomic protected no-replace output | YES(publishNativeAsset) | YES(PNPM/Bun) | YES(existing) | YES(existing) | pipx DATA ZIP0444 caller |
| Isolated fixed Python ZIP loader | NO(pipx) | NO | NO | NO | fixed interpreter/flags/bootstrap 실제 metadata/version/help |
| Runtime owner/maintenance/observer/CLI | YES(existing other clients) | NO(pipx) | NO(pipx) | NO(pipx) | pipx identity/role/transport/Artifact query/actual operations |

## 수정 지도와 두 번째 검토

1. O2 `verifyPipxClosure`의 입력 처리를 private reader로 분리해 typed closure와 이미 보유한
   archive readers를 반환한다. 기존 공개 command 출력과 proof 정책을 보존한다.
2. `PipxSupplyClosure`의 bounded ZIP 생성은 입력 archive 전체 digest/size와 모든 member의
   manifest digest/size를 다시 대조한다. 모든 원본 파일을 정렬·Store 방식·고정1980 timestamp·
   regular0444로 기록한다. RECORD와 license bytes를 재작성하거나 제외하지 않는다.
   현재 검토한 root module/dist-info 배치만 허용하고 `.data/purelib` relocation은 이 loader의
   추가 qualification 전 거부한다. Data resource는 원래 archive path로 보존하며 import root가 아니다.
   합계8MiB 제한은 이 candidate 소비자의 상한이며 기존 graph/runtime budget을 확대하지 않는다.
3. 기존 update-runtime의 `materialize-pipx-candidate`는 absolute root/input/new external
   `pipx-runtime.zip`을 요구하고 `publishNativeAsset(...,0444)`를 재사용한다. 공급 실패는
   output 미생성, 기존 output은 교체 금지다. Runtime pin/자동 설치/Host Python 실행은 없다.
4. 실제 후보는 기존 고정 Python3.14.7 image와 runsc의 readonly/network-none/cap-drop
   guest에서 `-I -S -B -c`의 고정 bootstrap으로 명시 ZIP만 stdlib 뒤 sys.path에 추가해
   pipx.__main__.cli를 호출한다. Version/help·7개 distribution metadata와 ZIP origin을
   확인한다. Host upstream import·shiv cache는 사용하지 않는다. 실제 startup query/Artifact
   handoff/observer authority·install/upgrade는 이 probe로 검증했다고 하지 않는다.
5. 성공 뒤 runtime lock/generator/maintenance/releaseinstall/native-tool transport와 observer의
   정확한 pipx DATA ZIP role을 잇는다. ELF kind로 위장하거나 일반 CONTROL 상속을 추가하지
   않는다. 기존 ToolReceipt.BindWheels/startup gate와 원본/current pipeline을 재사용한다.

검토: archive filename 자체를 import authority로 사용하지 않으며 O2의 isolated module root와
whole manifest를 요구한다. Source/input 바뀜·zero closure·duplicate/missing/extra input·changed
member·relocation·output collision을 검사한다. 일반 graph/parser/Policy 계약은 유지한다.
검증은 canonical format/quick/affected Race/docs/security와 실제 공급→ZIP bytes 대조,
이후 guest probe를 순차 수행한다. 작은 helper마다 원격 CI를 만들지 않는다.

## 고정 interpreter 근거

[CPython v3.14.7 zipimport](https://github.com/python/cpython/blob/v3.14.7/Lib/zipimport.py),
[metadata](https://github.com/python/cpython/blob/v3.14.7/Lib/importlib/metadata/__init__.py)를
local source evidence로 보존했다. Minor3.14 웹 문서는 현재 다른 patch일 수 있으므로 그 문서로
선택 pin을 바꾸지 않는다. 실제 고정 image의 ZIP/metadata 동작은 아직 NOT_TESTED다.

## Probe 환경 확인

현재 Docker에는 observer endpoint를 필수로 요구하는 `runsc-trace`만 등록돼 있다.
`/etc/heliopause/pod-init.json`의 remote sink는 ignore_setup_error=false다. Candidate의
Python ZIP 의미만 확인하는 첫 probe는 명시 runc/network-none/readonly guest로 한정하고,
runsc/observer qualification은 기존 supervisor와 typed runtime 공급을 연결한 뒤 수행한다.
설정·sink·fail-closed를 완화하거나 runc 결과를 gVisor 성공으로 기록하지 않는다.

첫 Quick89.010s는 Staticcheck SA1019(SetModTime deprecated)에서 중단돼 default tests는
미도달했다. 고정 timestamp를 FileHeader.Modified로 표현하며 writer의 표준 ZIP encoding을
사용한다. 첫 format/focused6.035s PASS를 전체 Quick 성공으로 확대하지 않는다.

첫 runc metadata probe0.687s는 ZIP import 전에 시험의 stdlib path prefix assertion에서
실패했다. Python의 python314.zip 표준 경로를 /python3.14 prefix로 잘못 제한한 전제다.
다음 fixture는 고정 image의 표준 세 경로를 exact list로 대조하며 site/cwd/user path 배제는
유지한다. 첫 metadata 원본과 version/help 미도달을 보존하고 제품 ZIP 코드와 섞지 않는다.
첫 실행의 보호5파일/Docker inventory·candidate bytes는 동일했고 owned container는 제거됐다.

## 한정 결과

[O3 receipt](./p4o3-pipx-loader-local-20261027.json)의 final Quick221.019s/affected 전체
Race37.284s/docs0.384s/security15.875s는 PASS다. 실제 공급 verifier stdout은 O2와 같고,
생성0.415s의 ZIP2505814B는190 original members를 모두 보존한다. 별도 생성도 같은 bytes,
기존 output 거부 뒤 inode/bytes와 frozen27파일은 불변이다. Runc Python3.14.7에서
metadata0.639s/version0.938s/help0.836s는 PASS이고 source origin이 ZIP으로 일치한다.
보호5파일/Docker inventories·candidate bytes도 동일하다. Materialization/basic loader만
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES(local)이며 runtime owner/observer/
실제 install·upgrade·CLI/whole P4는 MISSING이다. 다음은 그 기존 owner/consumer 연결 감사다.
