# S04 내용 분석 — 도입 baseline과 공급 검토

기준 `7f4439a`, 단일 active item은 M13-004다. 완료 조건은
[설계 §6.1](../../21-goal-completion-design.md#61-s04--구조-검사-뒤-실제-내용-분석을-연결한다)이
소유한다. 이 문서는 scanner 도입 검토이며 제품 scanner 구현·지원 선언이 아니다.

| 항목 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| S04 내용 분석 | NO | NO | NO | NO | 승인 가능한 고정 공급물·정확한 notices/source 대응, 격리 실행·신뢰된 자체 규칙, 생태계 전체 연결, 완전성/Evidence/Policy·정상/실패 검증 |
| 기존 구조 검사·archive inventory | YES | YES | YES | YES | [기존 목표 감사](./goal-coverage-audit-20261101.md)의 범위만 유지; 내용 분석으로 재명명하거나 재구현하지 않음 |

## 기존 owner와 수정 경계

- npm `Inspector.inspect`/`InspectArchive`는 압축·해제 총량, 경로·타입·manifest와
  member digest를 소유한다. 현재 `archiveCapture`는 전체 source body를 보관하지 않는다.
  Scanner 때문에 두 번째 무제한 추출기를 만들거나 모든 member를 메모리에 누적하지 않는다.
- PyPI `StaticInspector.InspectWheel`/`CompositeInspector.InspectGraph`는 wheel 구조와
  graph-wide static→dynamic 의존 순서를 소유한다. 일반 composite wrapper로 교체하면
  closure-aware 동적 검사가 유실될 수 있으므로 기존 owner에서 연결해야 한다.
- Go `ReadIntake`/`CheckedFiles`, Cargo `ReadIntake`/`ArchiveFiles`, GitHub Release
  `StaticInspector`의 ZIP/TAR/ELF 검사와 실제 한도를 재사용한다. ELF와 native extension을
  source AST 검사 완료로 표시하지 않는다. Project build의 실행 검사는 별개다.
- `ports.Inspection`/`CheckPlanner`, 기존 check plan·composite report·Evidence/Policy를
  재사용한다. Inspection은 분석/정규화, Sandbox는 실행/자원/수명을 소유한다.
  Core/Application/Inspection에 Docker·Linux executor를 직접 넣지 않는다.
- 공급 선택의 입력 owner는 기존 runtime lock, 생성 결과는 runtimeidentity/manifest다.
  기존 update-runtime plan/apply/verify·releaseinstall·doctor·자체 업데이트 경로에
  연결한다. Quality-only `tools.lock.json`이나 별도 scanner version lock을 중복 owner로
  만들지 않는다. 현재 lock/schema/언어 버전·예산은 변경하지 않았다.

## 공식 배포물 확인과 실제 한정 검증

[공식 PyPI metadata](https://pypi.org/pypi/semgrep/1.182.0/json)의 Linux amd64
manylinux wheel은 71,826,876B,
SHA256 `19a82868830389c039bb71bfc0aeebb28385e5b65740120710eff050324ad14f`다.
실제 다운로드를 대조했으며 설치하거나 Host에서 실행하지 않았다. Wheel 전체189 members,
해제284,127,183B 중 core는264,463,264B,
SHA256 `a4a028bf4f12fcc9340868c545f7f5b5564266ff6b1f8c292b3c58d7c694b5d8`다.
Python CLI 직접 의존성27개와 core 자체의 ELF 공유 라이브러리 의존성을 구분한다.
Core+bundled library26개만 선택한 실제 inventory는 별도 원장에 고정했다.

[PyPI publish provenance](https://pypi.org/integrity/semgrep/1.182.0/semgrep-1.182.0-cp310.cp311.cp312.cp313.cp314.py310.py311.py312.py313.py314-none-manylinux_2_34_x86_64.whl/provenance)는
저장소의 고정 sigstore-go/root로 offline SCT·transparency log·timestamp·정확한 subject
digest와 signer/commit을 검증했다(2.401s). 실제 publisher는
`semgrep/semgrep-proprietary/.github/workflows/pro-release.yml@refs/heads/develop`,
private commit `e7523e61f523ab23b6a295fd04884df46b882f1b`, run39486195151/attempt1다.
이는 서명 실패나 감염의 증거가 아니다. 그러나 공개 v1.182.0 source commit
`240da9f30cd06fb4ee00c2c31c114c1c1f5ee0f4`와 해당 binary의 대응 관계를 입증하지 않는다.
기존 HAA/UV/PNPM/Pipx의 public-source verifier 정책을 느슨하게 바꾸지 않았다.
검토 전용 프로그램은 scratch에만 있으며 제품 admission 권한이 없다.

고정 Python3.14.10/bookworm image의 network-none/read-only/UID1000/cap-drop/512MiB
runc 컨테이너에서 core와 라이브러리만 실행했다. JS/TS/Python/Go/Rust/HCL의 정상6·자체
marker6, 총12 source는 실제 scanned12/marker match6/errors0/skipped_rules0으로
1.517s에 통과했다. Marker는 악성 규칙이 아니며 source를 실행하지 않는다. 이는 파서/ABI의
작은 DATA 검증이고 gVisor·실제 악성 탐지·전체 성능·제품 연결 qualification이 아니다.
최초1.708s scanner는 exit0였지만 진행 점+JSON 때문에 검토 operator가 실패했다.
[고정 source의 `-json_nodots`](https://github.com/semgrep/semgrep/blob/240da9f30cd06fb4ee00c2c31c114c1c1f5ee0f4/src/core_cli/Core_CLI.ml#L472)를
사용한 후속 결과와 원본을 분리한다. 임의 stdout 앞부분을 잘라 허용하는 제품 parser를
만들지 않는다. 두 실행 모두 자기 컨테이너 제거·원래 container ID 보존·입력 동일을 확인했다.

원본·member 해시·공급 metadata·최초 실패와 범위는
[검토 원장](./content-analysis-review-20261103.json)에 연결했다. 원본 파일은
`docs/github-logs/goal-implementation-20261101/s04-*`에 보존하며 원장에 파일명·hash를
기록한다. 원장의 runtime scope와 독립된 product acceptance를 혼동하지 않는다.
완료한 wheel·격리 DATA 두 사본·ELF 조사 사본은98process의 FD/maps 참조0·query error0을
확인한 뒤899,904,010B를 삭제했다.
원장의 cleanup과 원본 `s04-review-payload-cleanup.json`은 삭제 범위를 소유한다.
Metadata·작은 원본 결과/재현 operator와 기존
공용 image/runtime은 보존하며 다음 검토에서 삭제한 실행 입력을 존재한다고 가정하지 않는다.

## 공급 경로 비교와 현재 미완료 조건

| 후보 | 확인한 장점/재사용 | 아직 충족하지 못한 조건 |
| --- | --- | --- |
| 공식 Python CLI wheel+closure | 고정 wheel/publish proof, CLI 계약 | 직접27개·전이 closure/notice 전체 고정, CLI의 원격 설정/telemetry/target 무시 설정을 통제, public source 대응 |
| 같은 wheel의 core+libs만 공급 | Python CLI closure 불필요, actual6 parser 동작, 명시 target JSON | core+26 libraries의 exact notices/source 대응, 전체 동적 library closure·보안 검토·gVisor/수명·자원 검증 |
| 동일 CE 공개 source에서 자체 build | 공개 commit·자체 build provenance·source/notices를 함께 고정 가능 | OCaml5.4.0 fork/Flambda·opam lock/repository·OS package·builder immutable 입력을 기존 공급 절차로 검증, 실제 build·license/closure inventory·비교 qualification |

현재 **prebuilt 공급의 adoption은 BLOCKED**다. Wheel에는 이름이 license/licence/notice/
copying인 member가 없고, [공식 third-party credits](https://semgrep.dev/cli-credits)는
확인 당시1.183.0/337packages였다. 이를1.182.0의 검증된 notices로 전용하지 않는다.
[공개 source의 LGPL2.1](https://github.com/semgrep/semgrep/blob/240da9f30cd06fb4ee00c2c31c114c1c1f5ee0f4/LICENSE)과
wheel metadata의 `LGPL-2.1-or-later` 표시만으로 bundled/linked dependency의 source·고지
조건 충족을 선언하지 않는다. 이는 CE 자체의 사용 불가나 프로젝트 Apache2.0과의
비호환을 확정한 법률 판단이 아니다.

해제 조건은 해당 binary의 정확한 source/notices·재배포 범위를 고정하여 검증하거나,
같은 CE 공개 source build의 전체 공급 입력과 notices·실제 결과를 검증하는 것이다.
공개 build recipe도 Alpine tag/`apk add`를 그대로 실행하면 불변 공급이 아니다.
먼저 고정 source-build 경로를 검토하며 다른 engine으로 조용히 바꾸거나 scan을 optional/
깨끗한 결과로 대체하지 않는다. **S04 전체와 최종16번은 계속 OPEN**이다.

[공식 규칙의 별도 사용 조건](https://docs.semgrep.dev/licensing)을 engine license와
혼동하지 않는다. Registry/community pack이나 source submodule의 semgrep-rules는 제품
규칙으로 수입하지 않고 기존 설계대로 직접 작성한 작은 규칙과 정상/의심 대조를 사용한다.
이번 DATA marker는 제품 규칙이 아니다. Source-build 검토가 필요한 동안 독립적인 S05의
baseline/규약 작업은 설계의 선행08–09 범위에서 가능하며 S04 완료를 전제로 삼지 않는다.


## 2026-11-04 공개 CE source/build 한정 비교

기준987545b. S05의 local consumer·CI 연결 묶음 뒤 S04 공급 검토를 계속한다. S06은
설계14번의 선행12–13/inventory가 필요하므로 아직 source를 구현하지 않는다.
[공개 source 검토](./content-analysis-public-source-review-20261104.json)가 아래 새 사실과
원본을 소유한다. 이전 private-wheel 공급 BLOCKED와 S04 네 상태 NO는 유지한다.

- 고정 source8개36849B를 Git blob identity/size로 확인했다. Linux opam lock은238개
  dependency/원격 Git repository7개이며 compiler4046c817/opam repo78d29aba를 고정한다.
  이는 실제 static-link 전부의 판정이나 OS package 불변 공급 완료가 아니다.
- [Upstream run39486928566](https://github.com/semgrep/semgrep/actions/runs/39486928566)의
  전체 결론은 cancelled지만 Linux job118849638246은 success이고 artifact12294769458을
  남겼다. 실제 checkout은 PR merge d7359d1이다. Head/tag240da9와 같다고 추정하지 않고
  두 commit API의 tree a70ab55160149f38c0bd17a8cd0d639a8c90cce0 동일을 확인했다.
  API source/sha binding과 cryptographic build attestation을 혼동하지 않는다.
- Artifact ZIP64018684B/SHA13eb21fc를 API 선언과 대조했다. Core255588872B/SHA32f30ee9는
  static ELF64/x86-64·INTERP/dynamic section 없음이다. 이전 wheel core+26 공유 libraries와
  다른 후보이며 원래 성공을 전용하지 않는다. Upstream artifact는2027-01-18 만료라서
  영구 공급 경로로 채택하지 않는다. 정확한 durable 공급/build·trust 검증은 MISSING이다.
- Core 관련 vendored source notice49개388966B를 고정 ref/Git blob으로 보존했다.
  외부 static-link dependency closure/notices나 배포 조건의 완료 판정은 아니다.
- 동일 작은 JS/TS/Python/Go/Rust/HCL marker DATA의 새 static binary 실행은 runc,
  network-none/read-only/UID1000/512MiB에서0.959s/scanned12/matches6/errors0/skips0이다.
  이는 파서/ABI 검토이며 악성 탐지·gVisor·제품 Policy/규칙 qualification이 아니다.
  Own container 제거·원래 container IDs/입력 불변을 확인했다. ZIP과 DATA copy319607556B는
  106proc refs0/query errors0/신원 대조 뒤 삭제하고 현재 reference core1개와 작은 고지를
  보존했다. 실제 Host 설치·runtime/version owner·새 CI 변경은 없다.


고정 opam repository의225 package metadata와13 pinned package의7개 원격 repository/
local tree-sitter 경로를 대조했다. 관련 metadata/notices245개454777B의 Git blob을 보존했다.
이는 build dependency superset이며238개가 모두 core에 링크됐다는 주장이 아니다. Source
metadata의 원격 URL 단순 집합에는 본체 dev-repo가 섞였으므로 dependency repository는
8개가 아닌7개다. API/notice counts만으로 실제 source copyright·정적 C library 고지를
완료 처리하지 않는다. Upstream OS 입력의 공개 선택 로그에는 Alpine base digest와 설치
package version이 있지만 그 자체가 전체 재현 build 입력/암호 provenance의 완료는 아니다.
다음은 exact source archive의 고지 확인과 영구 공급/build 경로이며 제품 파일/lock은 불변이다.


### 고정 source build와 자체 규칙의 로컬 후속

범위·원본은 위 public-source-review가 소유한다. 제품 도입과 S04 네 상태는 계속 NO다.

- 고정 OPAM archive150묶음 중147의 named notice를 확인했다. Ambient-context의 exact
  README에는 MIT 원문이 있고 trace는 MIT 선언을 보존한다. Bigstringaf cached archive의
 18개 Git blob/실행 bit를 exact public tree와 독립 대조했다. Opentelemetry fork의 선언과
  전체 source/고지 제공 조건은 최종 공급 패키지에서 함께 다뤄야 한다.
- OpenGrep 비교는 채택 없이 종료했다. 서명된 Python CLI와 unsigned core를 혼동하지
  않는다. 같은 CE 공개 source를 유지하며 Host 설치·제품 version owner는 불변이다.
- 원래 Alpine base digest와 signature-verified160 APK를 동결한 별도 SDK를 사용했다.
  Source fetch의 Git history/file-size 실패는 같은 compiler commit의 depth1 준비로
  교정했고 download-only V3가 완료됐다. Cache 유무와 실제 offline build는 구분한다.
- 첫 offline V1은 compiler/static curl과240개 성공 설치 뒤 dev Git fetch로 실패했다.
  `--skip-updates` 단독 V2는 반증됐다. Exact local bare5/file-only Git의 V3는 나머지
  의존성·compiler pin을 통과하고 별도 tree-sitter source 부재에서 멈췄다. 기존 version/
  checksum으로 그 archive를 고정했으며 성공한 compiler/의존성은 재설치하지 않았다.

공개 참조 core의 자체11규칙은6언어/4행동 범주와 credential 파일·환경변수 전송을 대조했다.
정상 credential header client/generated/minified/decode/file/command도 포함한다. 별도
boundary V1–V3는 prefilter·parser flag만으로 missing-node 파싱 누락을 해결하지 못했다.
고정 Core_scan의 공용 연결부가 inserted_tokens/tolerated_errors를 버리는 사실을 확인했다.

동일 source에 그 연결부 한 곳과 offline archive 소비 두 곳만 고친 patch f4628b96을
결합했다. Checksum 검사는 유지하며 새 AST·외부 syntax checker나 별도 언어 version
selector는 만들지 않았다. Network-none/UID1000/read-only root의 실제 V4 core build는
367.918s에 완료됐다. New core SHA83aa1847/255845624B와 strip 결과 SHAc3f43f96/
169609664B는 기존 upstream artifact와 다른 계보다. Strip은86235960B를 줄였고 static ELF/
INTERP·dynamic section 없음/non-executable stack을 확인했다. Private128MiB 예상은
반증된 것으로 보존하며 기존 제품 예산을 올리지 않았다.

실제 두 후보의34source DATA는1.556/0.972s에 각각 기대19match·정상14/경고0·malformed
Python/JS의2개 오류 보고를 확인했다. Explicit artifact ignore/config는 설정으로 쓰이지
않으며 nosemgrep은 결과의 is_ignored=true로 남으므로 제품이 이를 억제 권한으로 받아서는
안 된다. Scanner exit0·scanned 목록·finding과 coverage 완결을 분리한다. 이는 runc의
source DATA이고 Artifact 실행/대표 탐지율/제품 Policy/gVisor qualification은 아니다.

완료한 실제 link의406 compiler archive와 OPAM package input150을 대조했고 external
archive219개의 C autolink 선언을 같은 compiler로 읽었다. 이는 conservative 입력 집합이며
모든 archive member가 실행 파일에 들어갔다는 뜻이 아니다. 그 native source15묶음은 모두
고정 checksum에 맞고62개 고지568930B를 보존했다. Curl/tree-sitter는 재사용하고 추가
source179687366B는 설치·실행·압축 사본 보관 없이 읽었다. 최종 source/notice 배포 패키지,
정식 build trust·영구 공급·runtime owner/업데이트·실제 scanner backend와 모든 소비자 연결은
아직 MISSING이다. 도구 빌드 성공으로 acceptance를 닫지 않는다.

완료한 DATA 실행 파일 사본7개1703399648B는 각 batch의 정확한 신원·해시와 proc
FD/maps 무참조/query error0 뒤 정리했다. 원본 후보·현재 source/SDK/cache·작은 규칙과
표본·각 첫 실패/최종 결과는 보존한다. 같은 입력의 full/native/CPP/CI 반복은 없었고 새
CI/push/제품 설치는 하지 않았다. 다음은 현재 입력을 기존 공급/유지보수 owner에 연결하는
최소 범위 대조이며 S06을 선행 구현하지 않는다.


### 공식1.183 후보로 파싱 보완을 수렴

위 원장의 public_prefix_follow_up이 원본과 실제 범위를 소유한다. 기존 공개4wheel의279개
source 중9개 rf-string 오류를10개 유효 prefix로 축소했다. 자체 mapper prototype의326개
대조8.874s가 통과했지만 공식1.183에 같은 수정이 이미 있으므로 자체 Python 수정은 채택하지
않는다. 고정 tag/commit·새 OPAM repository/lock을 확인하고 같은 compiler/SDK를 재사용했다.
43개 tool dependency version 변경을 반영한 offline build890.291s와 같은326-source8.550s가
PASS다. 기대29match/정상296·의도한 malformed2만 보고됐으며 기존9개 파싱 오류는 해소됐다.
실행 파일은171648864B/SHAdeffa9a1·static ELF/stackRW다. Native flags는 기존과 같고 actual
406archive/150package owner 모두 새 lock 안에 있다. Source archive148 중125는 재사용하고
23개 변경분만 기존 cache의 ordinary object에서 strong hash로 검증했다. 새 vendor 고지61개
중 기존49개는 그대로이며 나머지12개는 목록 확장이지 새 release에서 추가된 파일이 아니다.
제품 runtime lock/Host 설치/CI는 불변이다. 최종 source/notice 공급 패키지·build trust와 기존
유지보수 owner/제품 연결·gVisor qualification은 여전히 MISSING이며 S04 네 상태는 NO다.


### 소스 패키지 생성의 기존 구현 대조와 로컬 구현

Source-package writer는 IMPLEMENTED=YES/WIRED=YES, QUALIFIED=YES(local named scope)/
ACCEPTANCE_CLOSED=NO다. 최종 대응 source/notice payload·깨끗한 재빌드·trust와 제품 공급
승인은 여전히 MISSING이다. 기존 `scripts/update-runtime/native_asset.go::publishNativeAsset`
보안 게시 절차는 유지하고 writer 입력만 추가했다. 기존 byte 호출은 그대로 위임하며 source
전체와 readback을 메모리에 동시에 올리지 않는다. Held parent/no-replace/신원·권한·fsync·
foreign 보존은 그대로다. 새 `package-semgrep-source`는 명시된 ordinary file만 frozen SHA/
크기로 읽고 정렬·경로/중복·JSON·총량을 검증한다. Signature/전체 source closure/런타임 승인
검증기로 쓰지 않으며 출력 runtime_qualified는 항상false다.

위 원장의 source_package_writer가 canonical format/foundation40.393s/security13.456s/
update-runtime 전체 Race34.640s·vet/staticcheck와 원본을 소유한다. 실제749파일/306335744B
생성6.675s와 독립 TAR readback이 PASS다. Git9/OPAM148/native15·extra-source10에 native
APKBUILD 대응97개 파일도 Git blob으로 확인했다. SDK APK binary는 source 묶음에 넣지
않았다. 첫 실제 입력은 Python Path 구성요소 순서 때문에 전체 path byte 정렬과 달라 출력 전
거부됐으며 입력 생성만 교정했다. 같은 Go source의 회귀를 반복하지 않았다. 기존 언어 pin/
Host/CI는 불변이고 재배포 payload 마무리·canonical clean build/기존 유지보수 연결을 계속한다.


### 원문 고지 payload — 기존 writer의 projection

고지 writer는 IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES(local named scope),
ACCEPTANCE_CLOSED=NO다. 최종 source/clean build·trust/제품 공급은 계속 MISSING이다.
기존 pipxNotices의 typed wheel/64KiB 계약을 바꾸지 않고 source-package writer에 작은
`--notices` projection을 연결했다. 별도 renderer/class/charset 추정이나 재인코딩 없이
passive USTAR로 전달하며 기존 source 상한은 유지하고 notices만4MiB/ordinary444/
원문1MiB를 적용한다. 원문에는 UTF-8이 아닌 텍스트1개와 page separator133개가 있다.

위 원장의 notice_package_writer가 foundation32.574s/security12.446s/해당 Semgrep
Race4.976s·vet/staticcheck와 실제275파일1980416B/0.315s·독립 readback을 소유한다.
173개 source/build superset의273개 원문을 보존했고 모두 링크됐다고 주장하지 않는다.
첫 실제 실행의 empty 부가/test LICENSE2개는 고지 목록에서만 제외했다. 원래 source와
첫 실패는 유지했으며 Go source나 검증 조건을 바꾸거나 같은 회귀를 반복하지 않았다.
동일1.183 공식 supplier 고지는 trace0.12의 MIT 원문을 제공하므로 그 출처/스냅샷을 보존했다.
Opentelemetry fork의 원래 README/opam 선언과 전체 source도 그대로 제공한다. 없는
copyright를 만들지 않으며 archive 생성은 법률·runtime 승인 판정이 아니다.

자체11규칙과 실제 사용한3파일 patch는 `tools/semgrep`의 현재 공급 입력으로 보존했다.
검증한 bytes와 같으므로 같은326-source scanner DATA는 반복하지 않는다. 다음은 현재
소스/고지에 대응하는 canonical clean build/SDK 절차와 기존 유지보수/제품 연결이다.


### Canonical clean build — 구현 전 대조

Clean build는 IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다.
현재 private worker는 구현/실행됐지만 compiled OPAM cache를 재사용했으며 canonical 개발
명령이 아니다. 기존 source writer/held copier/native publisher와 실제 SDK 준비·source build
절차를 재사용한다. Source/SDK bytes는 고정 입력만 읽고 Host에서 source를 실행하지 않으며
SDK 서명 설치는 별도 container, compiler는 network-none/UID1000/read-only root에서 실행한다.
기존 source+OPAM 작업 공간은 실제 약8.16GiB다. 새 clean compiler 공간은12GiB tmpfs/
16GiB memory/2CPU로 유한하게 두며 output256MiB·log16MiB와 deadline/owned stop/wait를
적용한다. 이 공급 개발 도구의 상한은 기존 Artifact runtime/관찰·검역 한도를 바꾸지 않는다.
Language pin과 Host 설치를 바꾸거나 untraced runtime을 등록하지 않는다. 새 SDK/fresh
compiler·dependencies와 기준 core의 대응 및 실제 결과/정리를 확인해야 다음 공급 연결을 닫는다.


Host 실행은 기존 `hosttool.NewSystem`의 absolute identity/local endpoint/minimal env 검증을
재사용하며 기존 runtime 준비가 선행조건이다. 새 developer build를 위해 production
constructor를 우회하거나 새 untraced registration을 만들지 않는다. Source compiler의
큰 stdout과 bounded stderr를 분리하는 작은 stream method만 같은 command owner에
추가한다. 검증된 SDK 준비와 source 컴파일은 role이 별도이며 build 성공 자체는 Artifact
검역/실제 runtime qualification이 아니다.

첫 clean-build foundation38.483s는 production/test compile 뒤 architecture에서 새
`update-runtime → hosttool` 재사용 edge 미등록을 검출했다. 직접 Docker 실행을 복제하지 않고
기존 developer composition의 exact edge/positive fixture와 Directory 책임에 등록한다.
Core/Application/Policy 경계와 Host constructor 검증은 유지하며 원본 실패를 보존한다.

첫 actual0.323s(v1)는 기존 Host manifest/current source lock 차이를 시작 전 거부했다.
설치본·runtime 정책을 바꾸지 않고 기존 검증 bundle을 private mount namespace에서만
연결한 v2는44.765s/SDK168 packages 성공 뒤 worker의 jq 부재로 실패했다. Fixed SDK에
Python3가 이미 있어 stdlib로 source hash/JSON projection을 한 번 수행하고 새 의존성은
추가하지 않는다. Repo index 조회도 명시 빈 repository 목록으로 제거하며 APK 서명 검증은
그대로다. Owned container/SDK 정리는 오류0, 설치 manifest는 불변이며 source compile은
미도달이다. 실제 raw s04-canonical-build-v1/v2를 보존하고 다음 compiler 입력만 교정한다.

V3 actual42.649s는 source hash/tag/lock를 통과하고 curl configure에서exit126이었다.
Native archive의 configure0755와 inert tmpfs counterexample(default noexec126/explicit
exec0)으로 원인을 확정했다. 공급 developer compiler의 /work tmpfs만 exec를 명시하며
Artifact runtime·관찰 한도는 바꾸지 않는다. 정리 오류0/설치 manifest 불변이다.

1026-file308335104B 소스의 independent TAR readback0.500s와 고지275-file 부분집합
대조는PASS다. 현재 clean command는 IMPLEMENTED=YES/WIRED=YES/QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO이며 fresh compiler/native 대응·최종 정리가 MISSING이다.
V4 actual은 이 한 곳의 mount 옵션 교정 뒤 진행하며 source/rules/고지 bytes는 V3와 같다.

V4 fresh actual275.883s는 curl build 뒤 OPAM pin 동기화에서 미사용 Windows/proto
Gitlink fetch로 실패했다. 같은 Gitlink 이름을 pinned .gitmodules에서 읽어, source offer가
Linux에서 미사용으로 기록한 항목의 update만 none으로 설정한다. 전체 submodule/update
검증·네트워크를 완화하지 않는다. 별도 tree-sitter upstream test Makefile symlink의 rsync
warning과 최초 compiler source fetch 실패는 구분한다. Cleanup 오류0/설치 manifest 불변이다.

V5 dry-run95.580s는 pin-depends도 적용하지 않으므로 유효한 동기화 검사가 아니다.
V6 download-only209.281s는 수정한 Git pins를 실제 동기화한 뒤 rresult0.7.0의 미포함 source
하나에서 차단됐다. 고정 package metadata222개의 url 소유자와 source offer를 대조해
이 한 gap을 확인했다. 기존 cache의8506B 원본이 official metadata SHA512와 일치하며,
같은 archive의 ISC 원문741B도 추가했다. 기존 core link input에 rresult가 있으므로 이
고지는 생략하지 않는다. Original failure와148-archive/173-component 이전 proof는 당시
범위로 남기고 새149-archive/174-component source offer와 download-only를 재검증한다.

V7 download-only206.516s는 fixed SDK/network-none/fresh OPAM에서 source sync와
locked source 확보를 완료했다. Marker/exit0와 소유 container 제거를 확인했다.
수정된1028-file308347904B source와276-file1982464B notice의 독립 readback0.519s도
PASS다. Full fresh compiler/native는 V5 actual로 진행 중이며 이 선행 성공을 native
재현·runtime/product qualification으로 확대하지 않는다. Canonical foundation26.188s
(format/runtime drift/module/production/test compile/architecture/docs)도 PASS다.

V5 compiler 진행 중 완료된 V2/V3/V4의 source TAR·APK 중복 사본1414175096B를
정리했다. State의 원본 source manifest/origins·recipe·bounded log/첫 실패를 보존했고
current Docker2의 mount와121-process FD/maps에서 참조0/query 오류0을 확인했다.
Primary source/cache와 현재 V5는 제외했으며, 이 수동 시험 정리를 C01 runtime 자동
quota/retention의 완료로 쓰지 않는다. 원본 s04-failed-build-copies-cleanup-v1.json 참조.

### Fresh source build — 로컬 재현 확인

67fb9fe와 같은 source worker/SDK recipe의 V5 actual1453.045s는 PASS다. 새 SDK와
fresh compiler/dependencies로 만든171648864B native의 SHA256은 기존1.183 후보
`deffa9a1814d13ccdfa29b2a40a3e2e9b8878bc6e3b917bbdb4d2892c01964d4`와
bit-identical이다. 따라서 같은 byte의326-source DATA/29 matches/의도한 parse error2
시험은 재실행하지 않고 그 범위를 재사용한다. 이 결과를 gVisor/runtime/제품 판정으로
확대하지 않는다. 독립 terminal readback은 native0555/root 소유와 source308347904B의
앞선 독립 projection SHA 일치, 소유 container2/SDK image 부재, 임시 APK/native 사본
자동 제거를 확인했다. 설치된 manifest는 불변이고 cleanup 오류0다.

Clean command는 IMPLEMENTED=YES/WIRED=YES/QUALIFIED=YES(local named scope)/
ACCEPTANCE_CLOSED=NO다. 전체 S04 네 상태는 여전히 NO이며 다음 MISSING은 기존
runtime owner/update/release supply 연결과 실제 gVisor transport/lifecycle, 전 source
consumer/Evidence/Policy다. Go/Python/Node/Rust pin과 Artifact 예산을 바꾸지 않았다.
원본 s04-canonical-build-v5.json·terminal-readback과 source-projection-v2가 범위를 소유한다.

### Runtime supply 연결 — source 작성 전 baseline (419a7da)

이번 표는 **Semgrep 공급 연결**만 판정한다. 기존 managed runtime 경로를 대체하지 않는다.
S04 전체와 실제 scanner 실행/제품 판정은 별도 MISSING으로 계속 남긴다.

| 단위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 기존 재사용 / 실제 MISSING |
| --- | --- | --- | --- | --- | --- |
| 고정 native/source/notice 생산 | YES | YES(developer) | YES(local) | NO(product) | `semgrep_build.go:rebuildSemgrep`, `semgrep_sources.go:packageSemgrepSource`, V5 actual/독립 readback; product owner 결합 없음 |
| Semgrep 단일 version/identity owner | NO | NO | NO | NO | 기존 `scripts/runtimes.lock.json` schema5·`generate-runtime-lock.go:runtimeLock/validate/render`, `check/runtime_versions.go:runtimeVersionOwners` 재사용. 현재 semgrep field/constant/owner 없음 |
| 선택 owner의 native/source/notices/rules 검증·materialization | NO | NO | NO | NO | 기존 `native_asset.go:publishNativeAssetWith`, held copier·manifest type 재사용. Passive source writer를 runtime admission으로 쓰지 않음 |
| Semgrep plan/apply/verify·drift | NO | NO | NO | NO | 기존 `update-runtime/main.go:snapshot/qualification`, `patch.go:selectedVersion/renderChanges`, `apply.go` 재사용. Semgrep dispatch/supply binding/rollback·정상 무변경 및 불일치 거부 없음 |
| release producer·installer·doctor·self-update | NO | NO | NO | NO | `release-manifest:checkNativeAssetBindings`, `release-packages:buildPackages`, `releaseinstall/manifest.go` asset map·`metadata.go` native tool bindings 재사용. 새 core/rules/source/notice의 role/mode/크기·전체 집합·설치/정리 소비자는 없음 |
| 실제 gVisor scanner·inspection/Evidence/Policy | NO | NO | NO | NO | 기존 sandbox lifecycle/typed inspection·각 archive inventory 재사용. 공급 성공만으로 이 행을 YES로 바꾸지 않음 |

수정 순서는 선택 owner와 실제 materialization/기존 updater 소비자를 함께 연결한 독립 단위,
기존 release/installed 역할 소비자, 실제 scanner/모든 artifact 소비자 순이다. Product tool
선택은 runtime lock 한 곳에 추가하고 기존 generator/drift 검사로 파생한다. Developer
SDK만을 위한 product schema 확장은 하지 않는다. 새 Semgrep product field가 필요한
명시 schema migration 때 기존 schema5 소비자/정상 fixture를 함께 대조하며 과거 증거와
언어 version은 일괄 치환하지 않는다. 새 일반 runtime framework나 미래 placeholder는 없다.

첫 단위의 완료는 native/source/notice/rules가 같은 선택 owner에 결합되고, changed/same
version plan/apply/verify와 byte 변조/누락/역할 혼용/foreign output·원본 보호가 실제
표적 검사에 연결되는 것이다. 기존 Go/Python/Node/Rust selector·metadata·생성 결과의
변경 없음도 대조한다. Release/engine가 미연결이면 전체 S04 acceptance는 계속 열어 둔다.

공급 연결 초안: schema6 selected tuple/readonly generated value와 기존 owner/drift 검사,
materialize-semgrep의 native/source/notice/rules 스트리밍 결합을 구현했다. Actual30.049s는
4개481988570B의 mode/hash·임시 입력 링크 제거를 확인했다. Runtime/Artifact 실행은 없다.
기존 updater에 changed/same-engine 명시 proof·freeze/plan/apply/verify를 연결하며 원본
보호와 중단 재개, version 불변 시 upstream tag/commit/epoch 이동 거부를 검사 중이다.
Synthetic 첫 support record fixture의 누락 schema와 same-version generated drift guard는
각각 교정했으며, verified Semgrep metadata가 있는 명시 갱신만 허용하고 legacy guard는
유지한다. Source origin 조건은 plan 생성과 frozen apply/render에서 함께 확인한다.

첫 related54.138s에서 scripts/generator·runtime-image·release-manifest는 PASS였고
gvisor capture의 schema5 잔류, workflow의 semgrep runtime_image selector 오분류가
실패했다. Runtimeidentity의 관찰 지점 실패는 installed 이전 runsc 자동 선택이며 current
fixture 누락으로 보지 않는다. 기존 HELOX_RUNSC_BINARY에 current lock SHA512가 맞는
기존 candidate를 명시한다. Security25.729s의 G110은 member.Size CopyN으로 제한하고
vet의 cross-package unkeyed fixture·Staticcheck 미사용 대입도 수정했다. 원본
s04-supply-unit-*와 current-runsc-input이 사실을 소유하며 최종 결합 검증은 진행 중이다.

### Supply owner/materializer/updater — 로컬 결합 확인

Root lock schema5→6의 Semgrep tuple은 기존 generator에서 immutable value로 파생한다.
Generator·developer updater는 기존 runtimeidentity의 동일 typed supply 형식/검증을 쓴다.
다른 language/image/tool/source-profile 값과 기존 support record는 의미상 모두 동일하다.
Latest/EOL은 추정하지 않고 명시 선택1.183.0의 local qualification만 기록했다.

Actual materialize30.049s는 core/source/notice/rules4개481988570B를 선택 SHA·mode에
결합했고 temporary input links를 제거했다. Source/notice는 전부 stream으로 읽으며
metadata 외 대형 body를 memory에 복제하지 않는다. 기존 held copier/native publisher와
source/notice manifest 계약을 공유한다. 공급 receipt는 runtime_qualified=false다.

기존 plan/apply/verify에 Semgrep frozen input을 연결했다. Changed version은 명시 review,
same-engine rules/patch 변경은 명시 proof를 요구하고 upstream tag/commit/epoch 이동과
동일 proof 반복을 거부한다. 원래 same-version drift guard는 다른 component에 유지한다.
Synthetic tests는 중단/reapply·원본 보호·frozen file/document 변조·verify 실패를 확인한다.
실제 full verification/배포 승인을 이 synthetic orchestration 성공으로 대체하지 않는다.

Foundation21.801s는 PASS. 첫 related에서 scripts1.298s/runtime-image0.024s/release49.989s
성공을 보존하고 교정된 updater/check/runtimeidentity만33.172s 재검증했다. Current runsc
입력은 lock SHA512와 일치한다. Security15.162s/vet1.442s/Staticcheck4.819s 및 new
Semgrep race15.674s도 PASS다. Runtime supply는 IMPLEMENTED=YES/WIRED=YES(developer)/
QUALIFIED=YES(local named scope)/ACCEPTANCE_CLOSED=NO다. S04 전체는 미완료이며
다음은 release/package/installed 소비자와 실제 scanner/모든 inspection consumer다.

### Release/installed 소비자 — source 작성 전 대조 (2cf6666)

기존 `release-manifest`·`runtime-image-manifest`·`release-packages`와
`releaseinstall/manifest.go`·`metadata.go`·`installer.go`, releaseactivation의 installed/
bootstrap copy 경로를 재사용한다. Semgrep 네 파일은 모두 IMPLEMENTED=NO/WIRED=NO/
QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다. 기존 native/ZIP DATA/metadata copy·mode·rollback
구현은 YES이며 대체하지 않는다. `IsRuntimeData`의 기존 pipx 전용 크기/identity 소비자를
구분해서 고친다. Native core는 semgrep 실행 역할, rules/source/notices는 semgrep-data의
고정 passive 역할로 전달하며 role 혼용을 거부한다. 기존8–16 asset record는 읽되 새
공급 일부만 있으면 완전한4개/현재 release 집합을 요구한다. Runtime manifest v4에 같은
immutable Semgrep tuple을 추가하고 v2/v3 legacy는 분리한다. 현재 language image5는 그대로다.

완료는 producer/SBOM/checksums·package·설치/업데이트/doctor·rollback/중단 복구가 같은
네 파일을 보존하고, data0444/core0555·크기/해시·구버전/누락/변조/역할 혼용 거부와
원본 보호를 실제/표적 검사로 확인하는 것이다. Workflow 공급/실제 gVisor와 모든
inspection consumer가 미연결이면 전체 S04 qualification은 계속 OPEN이다.

### Release/installed 소비자 — 로컬 결합 검증

기존 producer/package/native selection·DATA copy/recovery 경로에 core와 세 passive 파일을
연결했다. Release는 현재20 assets·bundle24 files, runtime metadata는 v4의 동일 selected
Semgrep tuple을 사용한다. Published8–16 assets/v2–v3 metadata를 읽는 경로는 유지하며,
새 일부 집합·역할 혼용·과대 크기·selected SHA/size·tuple 불일치를 거부한다. SBOM에는
supplied lock version을 쓰고 source 대형 파일 hash와 passive 검사에는 bounded stream을
사용한다. 기존 Pipx 전용 identity/8MiB 제한은 exact Pipx 파일에만 적용한다.

Foundation60.834s PASS 뒤 related64.118s에서 producer58.829s/package·activation/bootstrap/
inspection은 PASS였고 새 install fixture의 같은 bundle 이름과 current schema 기대값 두
실패를 보존했다. Fixture만 교정한 related3.737s, 새 Semgrep race7.589s,
security31.252s/vet10.087s/pinned Staticcheck13.590s는 PASS다. Actual namespace11.266s는
선택된481988570B 네 파일을 두 version에 복사/hash·root single-link0555/0444·doctor·rollback/
변조 거부를 확인했고 tmpfs는 namespace 종료로 제거됐다. Verifier는 주입된 unsigned copy
시험이며 source authenticity/새 hosted 성공은 아니다. Installed Host manifest는 동일하다.
원본은 `goal-implementation-20261101/s04-release-*`와 별도 local receipt를 따른다.

이 단위는 IMPLEMENTED=YES/WIRED=YES(local producer/installed consumers)/QUALIFIED=YES
(named local scope)/ACCEPTANCE_CLOSED=NO다. Release workflow는 기존16개 입력만 공급하고
새 core/source/notices의 hosted 원본 공급이 아직 없으므로 새 producer20개 요구를 만족하지
못한다. 이를 current release qualification으로 숨기지 않는다. Current-only admission gate와
공식 provenance·실제 whole self-quarantine/승인/activation 및 중단 복구의 새4개 actual은
아직 MISSING이다. 다음은 이 hosted 공급의 exact 입력/소유자와 최종 gate 연결, 실제
scanner/gVisor/모든 inspection consumer다. 같은 native SDK 전체 재빌드는 반복하지 않는다.

추가 producer delta9.437s는 네 파일 변조 전 metadata 무작성·package 전체 내용/재현성·
source512MiB 초과·v4 tuple 누락/잘못된 owner를 PASS했다. 단 operator가 실행 중 문서/
inventory를 갱신해 raw의 source_unchanged=false다. 독립 scope 대조에서 변경/추가 항목은
문서4개뿐이며 모든 Go 입력 hash는 동일했다. 전체 입력 불변 성공으로 쓰지 않고 해당
Go 표적 범위만 인정하며 같은 full/native/CI 검사를 다시 돌리지 않는다.

### Scanner runtime — source 작성 전 대조 (4deca33)

| 단위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 재사용 / MISSING |
| --- | --- | --- | --- | --- | --- |
| 고정 installed 도구 reader·tar transport | YES | YES(other tools) | YES(prior named scope) | NO(S04) | `native_installed_tool.go:readInstalledNativeTool`·`native_tool_transport.go` 재사용. Semgrep core0555/rules0444의 exact pin/guest role만 없음 |
| readonly scanner supply/target lifecycle | NO | NO | NO | NO | `pipx_data_volume.go`의 sealed readonly 단계·`pypi_closure_volume.go:createInputVolume`의 identity/attachment/UNKNOWN refusal 재사용. 실행 core와 nonexec target 구분·새 current source reader 필요 |
| 격리 scanner 실행·유한 출력 | NO | NO | NO | NO | 기존 CommandRunner/hosttool·gVisor start/stop/cleanup 재사용. 고정 arguments/네트워크 없음/출력·취소/identity 연결 필요. Untraced runtime 등록이나 Host artifact 실행 없음 |
| JSON 해석·모든 artifact 소비자 | NO | NO | NO | NO | 원래 npm `InspectArchive`, PyPI `InspectWheel`/execution inventory, Go `CheckedFiles`, Cargo `ArchiveFiles`, GitHub format inventory를 재사용. Parsing/error/skip/모든 target·rule coverage·Evidence/Policy/전이 graph 미연결 |

첫 최소 수정은 기존 typed installed-file/transport 경로의 두 exact role(core/rules)이다.
새 source/notices archive를 runtime으로 복사하지 않고 기존 native tool reader를 바꾸거나
대체하지 않는다. Core/rules는 같은 immutable tuple에 결합하고 rules는 DATA0444이며
기존 mutable native runtime volume을 새 scanner에 주지 않는다. 이어 readonly lifecycle과
실행 소비자를 같은 S04 범위에서 완성한다. 이 선행 공급 연결만으로 실제 scanner나 S04
구현/qualification 완료를 표시하지 않는다. 현재 release hosted 공급/whole activation gap도
그대로 남는다. Target/JSON 계약은 실제 고정 scanner 출력과 대조한 뒤 필요한 경계만 추가한다.

Readonly supply의 writer→anchor→writer 제거→seal·attachment/UNKNOWN/cleanup 구현은
`pipx_data_volume.go` 한 곳을 그대로 확장해 공유한다. 기존 Pipx type은 alias로 유지하고
기존 호출·기본 flag·DATA noexec 계약을 보존한다. Semgrep의 두 고정 파일을 위한 공급
identity/용량/허용 mount만 추가하며 동일 상태 머신을 복제하지 않는다. 실제 target data와
실행 admission/관찰 profile은 이 공급 seal과 별도이고 아직 미구현이다.

### Scanner core/rules 공급 — 실제 readonly lifetime 로컬 확인

기존 typed installed-file/tar transport에 고정 core0555/rules0444를 연결했다. Identity는
같은 generated tuple이며 caller hash/size/mode는 Docker 호출 전에 거부한다. 기존 UV/
PNPM/Bun/Pipx 역할은 유지하고 scanner에는 mutable native volume을 주지 않는다.
`readonlyToolVolume`은 원래 Pipx 상태 머신을 공유하며 기존 type alias/기본 DATA noexec와
호출을 보존한다. Semgrep 공급만 별도 root/257MiB 고정 한도·core/rules digest를 사용하고
source/notices나 Artifact target data를 이 executable supply에 넣지 않는다.

Foundation48.107s·첫 표적8.563/readonly8.302s는 PASS였다. Actual V1/15.704s는 공급 bytes/
seal·경쟁 writer 거부 뒤 `closeWritableOutputVolume`의 새 destination 누락으로 실패했다.
또 createInputVolume의 새 역할은 CREATE/후속 identity query 실패에서 기존 protected
candidate 보존 분기를 따라야 한다. 같은 허용 목록에 연결하고, 정상 idle 제거와 UNKNOWN
생성·삭제 거부를 함께 표적8.899s로 확인했다. Test 거래 identity도 basename001 대신 기존
SandboxSessionID를 사용한다. V1의 정확한 owner/pin/options/연결 부재를 확인해 남은 볼륨
하나만 제거했다. 원본 실패를 성공으로 덮지 않는다.

Actual V2/20.116s는 selected core/rules의 protected reader·Docker tar readback·readonly
seal·writer 제거 선행조건·경쟁 writer 거부와 최종 own container/volume0을 PASS했다.
설치된 manifest는 동일하고 only fixed runc sleep keeper만 실행했다. Scanner/Artifact를
실행한 시험이나 gVisor source-analysis qualification은 아니다. Full sandbox Race42.526s,
security25.538s/vet2.042s/pinned Staticcheck13.043s는 PASS다. 첫 foundation 뒤 교정은
해당 destination/UNKNOWN 경계와 test identity만이며 새 full/native/CPP/CI는 없다.
완료된 test binary3개68143541B도 exact receipt/hash를 대조해 제거했다. Active selected
core/source/notices/rules 공급은 다음 실행 소비자를 위해 유지한다.

이 공급 단위는 IMPLEMENTED=YES/WIRED=YES(existing supply/readonly lifecycle)/QUALIFIED=YES
(named local scope)/ACCEPTANCE_CLOSED=NO(S04)다. 다음은 실제 traced gVisor scanner와
readonly target source·유한 실행/출력/parse/coverage, 기존 inspection/Evidence/Policy와
전이 artifact 연결이다. Hosted source 공급/current-only admission·whole self-quarantine/
activation/recovery는 별도 MISSING으로 유지한다. `s04-scanner-*`·first-failure-cleanup과
completed-test-binaries-cleanup raw가 source/입력·실패·정리의 실제 범위를 소유한다.

### Traced scanner 실행/출력 — source 작성 전 대조 (9ae6ad3)

기존 `Sandbox.Execute`는 Artifact의 설치/실행 요청이며 source parser 실행을 넣는 별도
mode/config가 없다. 그 의미를 바꾸지 않고 현재 source-analysis 전용의 좁은 Ports 계약에
기존 `AcquiredArtifact`, `SandboxResult`, `CapturedProcessOutput`을 재사용한다. 새 실행 상태/
완료 영수증 체계는 만들지 않는다. Typed normalized source files만 전달하고 Host 경로,
artifact command/config/plugin/rule 선택 입력은 받지 않는다. 아직 네 상태 모두 NO다.

`SharedObserver.observerExpectedTopology/newObservationTraceLedger`와 C++ helper의 closed
profile/classification 목록에 source-analysis가 없음을 확인했다. 기존 profile의 mount나
실행 허용 범위를 넓히지 않고 고정 core path/argv·readonly supply/target 전용 profile을
연결해야 한다. `boundaryHelper`·direct exec의 one-shot admission/실제 kernel terminal·
resource lease·trace 수집·취소/cleanup과 bounded capture는 재사용한다. 새 untraced runtime
등록, Host scanner 실행, stdout의 완료 authority는 허용하지 않는다. C++/Go 변경을 실제
분석 명령과 묶은 다음 한 번 qualification하며 기존 language version owner는 불변이다.

Inspection은 source source-name/language/body와 exact Artifact digest를 결합하며 기존
archive/member 검사를 통과한 파일만 넘긴다. Semgrep 결과는 고정 version/rule digest와
전 target inventory·rule ID/version/severity·유한 위치를 대조한다. Errors/skips/누락/중복/
nonzero exit/timeout/truncation/일치하지 않는 target은 completed zero findings가 아니다.
`nosemgrep` 표시도 무시한 finding을 지우는 권한이 아니다. Rule pack은 installed fixed
bytes를 원본 hash로 검증해 읽고 raw source snippet·metavariable 값은 Evidence에 저장하지
않는다. Policy는 기존 Finding/Evidence만 소비하고 의심 rule은 MANUAL_REVIEW다.

기존 npm archiveCapture는 selectedPath 한 개를 재생하는 surface이므로 이를 파일마다
호출해 O(n²)로 만들지 않는다. 같은 원래 inventory pass에서 source bytes를 수집하며,
PyPI는 이미 검증한 wheel/member reader, Go CheckedFiles, Cargo ArchiveFiles를 재사용한다.
기존 static 결과·graph budget charge를 두 번 실행하지 않는 연결을 우선한다. 먼저 실제
scanner/출력 계약을 닫고 각 소비자를 연결한 뒤 전체 전이 graph·정상/의심/미완료와
기존 네 PyTorch profile/큰 공개 graph는 해당 최종 묶음에서 qualification한다.

Source input의 새 한도는 한 파일64MiB·source 합계512MiB·target10000개·generated target
metadata8MiB 한 곳에서 관리한다. 기존 각 archive/profile budget을 늘리지 않고 그 검사를
통과한 source만 별도로 제한한다. O(n)으로 opaque 순번 이름·언어·원래 상대 path·body SHA를
결합하고 매체 전달 때 같은 bytes를 다시 hash한다. JSX/TSX 확장자는 보존한다. 표적 첫
10.664s는 새 fixture의 빈 variant가 Domain 생성에서 실패한 준비 오류이며 source 거부
회귀가 실행된 결과가 아니다. Product invariant 대신 fixture를 교정한다.

Source target TAR readback은 기존 `pipxNativeInput.scanTar`의 ordinary-file/0444/owner/hash/
전체 집합·padding 검증을 공유한다. 새 archive parser를 복제하지 않고 원래 callback의
입력 기대값 작성 부분만 유지해 같은 scanner에 전달한다. Pipx의 p.valid/원래 error와
범위는 보존하며, 현재 두 소비자만 쓰는 private streaming transport다.

새 실행 consumer는 기존 `pipxNativeSession`의 raw lease/trace/keeper lifecycle을
`observedNativeSession`으로 공유한다. 기존 type alias와 메서드 본문은 유지하며 별도의
keeper/종료 상태 머신이나 wrapper를 복제하지 않는다. Source input의 같은 readonly
owner를 닫는 슬롯만 추가한다. Opaque 기존 name/label·Python image·unknown-CREATE
기록·authenticated 제거 순서와 원래 Pipx/UV run 호출은 유지한다. Source 실행은 DATA
keeper와 별도의 traced profile이며 기존 Pipx 설치/선택 메서드를 호출하지 않는다.


### Traced scanner 입력/실행 — 부분 연결과 최초 실제 실패

`SourceAnalyzer`는 기존 protected tool reader·readonly writer/anchor/seal, Python image,
`observedNativeSession`의 lease/keeper/trace/cleanup, 기존 ELF one-shot admission와 bounded
capture를 연결한다. Node image probe가 아닌 실제 사용하는 Python probe를 사용한다.
고정 argv17과 controller-generated targets만 실행 입력이며 별도 설치/Artifact 실행은 없다.
Source DATA는 별도 noexec/readonly volume이다. 새 상태 머신/완료 token은 만들지 않는다.

- 입력/기존 Pipx·Semgrep·새 offline args/topology 표적8.583s PASS.
  `s04-scanner-execution-target-v2.json`이 입력/범위를 소유한다.
- canonical foundation65.047s PASS: format/runtime lock·지원 freshness/module/build/test
  validity/architecture/docs. `s04-scanner-execution-foundation-v1.json` 참조.
- observer-only CPP V1/48.889s의138latch가 PASS했다. Actual V1/6.820s는 tool 이전의
  resource peer 거부다. 시험 실행 inode와 등록 client inode가 달랐으며 기존
  `PolicyPeerAuthorizer.AuthorizePeer`와 통합 harness를 대조해 시험만 교정했다.
- Actual V2/20.113s는 setpriv의 pinned OCI `/etc/ld.so.cache`557056에서 STREAM_FAULT다.
  기존 exact ELF demotion read predicate에 source profile만 연결했고 role/provenance/
  pending/path/write 거부를 추가한 CPP V2/54.342s·138latch가 PASS했다.
- Actual V3/20.240s는 실제 core handoff 뒤 system CPU quota32768에서 STREAM_FAULT다.
  고정 core의 CPU quota/period 두 읽기를 기존 native predicate에 source 전용으로
  연결하는 후보를 검증 중이다. GitHub의 library/proc surface는 source에 전용하지 않는다.

원본 `s04-source-analysis-native-v1/v2/v3.json`과 같은 log 및
`s04-source-analysis-observer-batch-v1/v2.json`은 최초 실패/서로 다른 도달점을 보존한다.
각 actual은 helper wait0·원래 Host 파일/inode/Docker/endpoints/process 상태 동일을 확인했고
시험 사본236590503/214342829/214342829B를 정리했다. 새로운 source profile의 실제 scanner
완료·strict JSON 해석·모든 inspection/transitive consumer/Evidence/Policy는 아직 MISSING이다.
기존 Go client/test/helper build는 Go 입력 동일성에 한해 재사용하며 observer delta만 새로
검증한다. 실제 실패를 partial success나 검사 완료0건으로 바꾸지 않는다. CI/영구 설치/
언어 version owner/권한·기존 예산은 변경하지 않았다.

CPP V3/47.378s·138latch는 source만의 exact quota/period32768 및 role/image/flags/mount
거부를 확인했다. Actual V4/18.607s는 그 지점을 지나 core의 OCI UTC read559104에서
멈췄다. Locator07a7ea4562a20073의 bounded known-path 대조는
`/usr/share/zoneinfo/Etc/UTC`와 일치한다. 이 path·flag·OCI-root mount만 후보에 연결하고
다른 zone/alias/write/mount를 거부하도록 검증 중이다. Core ELF에 dynamic section은
없으므로 GitHub의 일반 native library 허용 목록을 추가하지 않았다. Native V4는
Host 동일/helperwait0/사본214342829B 정리이며 scanner 완료가 아니다. Raw
`s04-source-analysis-native-v4.json`과 `s04-source-analysis-observer-batch-v3.json` 참조.

UTC CPP V4/48.079s는 새 source positive 한 latch만 FAIL(기존137PASS)했다. 기존 앞단
flag mask가559104의 O_NONBLOCK을 먼저 거부한 후보 누락이며 actual V5를 그 상태로
실행하지 않았다. Source의 exact559104만 앞단에 연결한 CPP V5/48.314s·138latch PASS 뒤
actual V5/19.344s는 다음 `uname` child exec까지 도달했다. 그 실행은 unexpected이고
뒤 libc cache read에서 STREAM_FAULT이며 전체 완료가 아니다. Exact selected source
`libs/commons/Platform.ml`의 `uname`/`uname -m` 호출과 kernel executable locator를
대조했다. `SourceAnalysisUnameParent`는 실행 승인과 loader 읽기가 함께 소비하는 한
private lineage 검사다. Accepted parser의 exact kernel clone/direct parent/start/image·
ARTIFACT 유지·no CONTROL/network, pinned OCI uname/path/argv와 cache/libc 읽기만 연결한다.
다른 args/alias/role/parent/PID/image/shadow/profile·loader write를 거부하는 실제 clone/exec
parser 회귀를 포함한 CPP V6/48.332s·138latch가 PASS다. 새로운 active marker나 general
child-exec 권한은 없다. 같은 Go source의 full sandbox Race40.198s도 PASS다.
Actual V5는 Host 동일/helperwait0/사본214342965B 정리이며 actual V6는 아직 진행 중이다.
Raw `s04-source-analysis-observer-batch-v4/v5/v6.json`, `s04-source-analysis-native-v5.json`,
`s04-scanner-execution-sandbox-race-v1.json`이 scope를 소유한다. Foundation/Race와 Go
native build를 observer-only delta마다 반복하지 않는다.

Actual V6/20.720s는 CPP V6 후에도 같은 uname unexpected/cache read로 실패했다.
따라서 위 lineage 후보는 **부분 검증/권장 제외**이며 실제 query 해결로 기록하지 않는다.
Host 동일/helperwait0/사본214343093B 정리다. Next는 SDK 사본만의 bounded numeric
query/parent/argv-shape 진단이며 제품 조건을 추측으로 넓히지 않는다. 저장소 C++에는
진단을 넣지 않고 source·credential·argv 원문을 기록하지 않는다. 원본
`s04-source-analysis-native-v6.json`은 최초 V5와 같은 실패를 보존한다.

SDK-only numeric diagnostic CPP/47.527s 후 actual V7/20.632s는 parent_ok/clone_ok/
parent_id_match=1, exact execfn/argv0=uname, argc2/arg1_s=1을 확인했다. 기존 후보는
`uname -s`를 누락했다. 단순 Platform.ml의 `uname`/`-m` 대조를 실제 모든 startup query의
전체 census로 확대하지 않는다. 부모/role/clone 조건을 풀지 않고 그 exact `-s`만 추가한
제품 후보를 다시 검증한다. SDK-only overlay는 새 canonical copy에서 제거하며 root C++에
진단을 넣지 않았다. Raw `s04-source-analysis-native-v7-uname-diagnostic.jsonl`은 숫자/
boolean 한 행만 보존하고 argv·source 원문을 포함하지 않는다. Host 동일/helperwait0/
사본214347293B와 임시 diagnostic 파일 정리를 확인했다. Actual 통과는 아직 아니다.

Canonical CPP V7/47.162s·138latch는 numeric overlay를 제거한 제품 source와
uname/-m/-s·exact lineage/loader 거부를 확인했다. Actual V8/21.501s는 그 query를
expected로 수집(14 expected/unexpected0)한 뒤 core의 OCI read557056에서 멈췄다.
Locator660afad21eb74b5c는 bounded fixed-path `/etc/ssl/certs/ca-certificates.crt`와
일치한다. 고정 OCI CA bundle의 그 읽기만 source 전용에 추가해 검증하며 다른 certificate,
config/alias/flag/mount를 허용하지 않는다. Network none과 generic Artifact/CONTROL
경계는 유지한다. V8 Host 동일/helperwait0/사본214347189B 정리다. 다음 source scanner
완료/JSON/consumer는 여전히 MISSING이며 원본은 `s04-source-analysis-native-v8.json`,
`s04-source-analysis-observer-batch-v7.json`을 따른다.


### Traced scanner — named local 실행 완료

Canonical CPP V8/49.676s·138latch 뒤 actual V9/23.406s(test20.40s)가 PASS했다.
여섯 언어의 정상6/JS 의심1을 같은 fixed core/rules와 readonly source volume으로 읽고
scanned7/findings1/errors0/skipped0·SandboxCompleted를 확인했다. 기존 kernel target/terminal,
observer stream completion과 resource create/register/close 각1·accounted CPU573885usec를
확인했고 caller 이후 cleanup이 성공해야 출력이 반환된다. Helper wait0/own Docker resources0/
Host 파일·inode·endpoints·process 동일이며 시험 사본214343077B를 정리했다.
이것은 실제 parser runtime의 named local 검증이며 Artifact 코드를 설치/실행한 것이 아니다.

Go candidate는 observer-only 변경 동안 불변이었다. Foundation65.047s/full sandbox
Race40.198s/security21.974s/vet2.250s/pinned Staticcheck6.740s와 마지막 Go 표적8.583s를
각 scope에 재사용한다. SDK-only numeric diagnostic는 canonical observer copy로 제거됐고
임시 진단 파일도 삭제했다. CI/push/Host 영구 설치/runtime 등록/언어 version owner는
변경하지 않았다. Raw `s04-source-analysis-native-v9.json`, CPP V8 및
`s04-scanner-execution-{foundation,sandbox-race,security,vet,static}-v1.json` 참조.

S04 전체 네 상태는 여전히 완료가 아니다. 다음은 scanner stdout의 strict JSON/target·rule/
version binding과 partial errors/skips/limit/overflow의 required coverage, 여섯 생태계와
transitive graph·Evidence/Policy의 실제 consumer다. Runtime failure/cancellation·변조/출력
주입과 큰 graph/네 PyTorch profile·grouped hosted도 아직 전부 검증한 것으로 쓰지 않는다.
기존 release workflow16/current20 공급·current-only admission/공식 self-quarantine/activation/
recovery/provenance는 별도 MISSING을 유지한다.


### Scanner 출력 해석 — 구현 전 책임 대조 (4cbc9ec)

기존 advisory의 strict JSON walker는 private·advisory response error/bounds를 소유한다.
이를 content에서 직접 호출하려고 advisory를 import하거나 범용 JSON package로 전체
재배치하지 않는다. Source parser는 실제 소비할 document를 한 번 token으로 구성하면서
중복/case-fold key·UTF8/depth/size/trailing을 거부하고, 고정 Semgrep output의 의미를 같은
owner에서 해석한다. 기존 Domain `CheckExecution`/Finding/Evidence와 composite를 재사용한다.
Raw metavars/snippet/message는 보존하지 않으며 rule pack의 version/severity/disposition은
고정 공급 해시와 결합한다. Runtime의 opaque path 생성 규칙을 Inspection에 복제하지 않고
`SourceAnalysisOutput.Targets`의 controller-owned opaque→original mapping을 반환한다.
그 map은 stdout과 분리되고 cleanup 실패 때 다른 output과 함께 폐기된다.

완료된 이전 Go trial binary3/92571760B는 actual V9/Host 원복 이후 exact build receipt,
SHA/owner/inode/nlink·미실행 상태로 확인하고 삭제했다. Active source/core/rules/SDK/bundle과
global caches는 유지했다. `s04-scanner-execution-binaries-cleanup-v1.json` 참조.


### Strict source 결과 — named local 해석 연결

`content.InspectSources`는 기존 SourceAnalysis port를 한 번 호출하고, pinned rule bytes/
version·전체 rule-engine census·controller-owned target map/scanned inventory·유한 byte 위치를
대조한다. Duplicate/Unicode case-fold key·trailing/UTF8/depth/1MiB·외부 path/rule/언어/metadata/
출력 overflow·실행 미완료를 거부한다. 현재 해석 범위는 rule catalogue에 있는 여섯 언어의
UTF8 same-file AST다. 다른 언어/encoding을 empty clean으로 받지 않는다.

기존 Domain invariant를 유지해 required source coverage와 optional completed signal
interpretation을 분리한다. Parse/skip/timeout/missing target/retained512 초과는 required
INCOMPLETE이며 유효한 bound hit는 별도 completed signal check에 남긴다. Inline nosemgrep
표시도 hit를 지우지 않는다. Body hash는 file마다 한 번 계산해 모든 hit에서 재사용하고,
byte 위치는 file마다 한 forward pass로 확인한다. 원래 ordered inventory와 actual
scanned/unscanned subset digest·counts·고정 same-file scope를 기록한다. Source/filename/
metavars/dataflow trace 원문은 Evidence에 저장하지 않는다.

- First Race V2/1.162s는 새 zero-match case에서 completed signal check의 Evidence 누락을
  검출했다. Domain의 completed 검사 증거 요구를 완화하지 않고 source signal summary를
  추가했다. 최신 package Race V6/1.940s는 정상/0건·부분/error/skip/timeout·512 bound·위조/
  변조·no-target·첫 runtime cause와 여섯 언어를 한정 확인한다.
- Actual V1/23.059s는 runtime 완료 후 HCL alias 검증에서 거부됐다. 고정 rule pack의
  terraform과 target protocol hcl 대응을 catalogue 한 곳에서만 정규화했다. Selected
  source의 Language.ml/semgrep-hcl Main.ml과 실제 schema를 대조했다.
- Actual V2/40.872s의 **전체 parent는 FAIL**이다. Clean child18.06s만 six languages6source/
  findings0/errors0/skip0·coverage COMPLETED와 resource1/CPU517873usec를 PASS했다. Partial
  child는 optional dataflow_trace를 닫힌 extra schema가 누락해 거부했다.
- 고정 `Core_json_output.ml`의 optional match_dataflow_trace를 확인해 그 explanatory
  object만 추가하고 원문을 버린다. Changed partial child만 실행한 actual V3/24.096s
  (child19.21s)는 scanned9/hit1/errors2/skip0·required INCOMPLETE/optional signal COMPLETED,
  ignored=true 보존·resource1/CPU624221usec를 PASS했다. Successful clean child는 재실행하지
  않았고 기존 observer CPP138 V8와 gVisor 본체도 재사용했다.
- New package의 foundation V1/61.260s는 당시 format/import/architecture/build/docs scope다.
  이후 같은 owner의 좁은 수정은 최신 tagged build10.064s, package Race, security29.131s,
  vet2.787s/pinned Staticcheck17.518s로 검증했다. Foundation 전체가 최신 source에서 다시
  PASS했다고 확대하지 않는다. Runtime alias/TAR의 이전 full sandbox Race도 그 scope다.

두 actual의 각 원본은 `s04-content-report-native-v1/v2/v3.json`, checks는
`s04-content-report-{target-v2,target-v6,foundation-v1,security-v1,vet-v1,static-v1}.json`이다.
모두 Host 동일/helperwait0/own resource 제거와 temp copies214436928/214441209/214441743B
정리를 확인했다. 완료 trial binary/helper6개118493232B도 exact build/owner/SHA/inode/nlink·
미실행 상태로 제거했다(`s04-content-report-binaries-cleanup-v1.json`).

다음은 existing npm archiveCapture, PyPI verified member/graph budget, Go CheckedFiles,
Cargo ArchiveFiles 및 GitHub/Terraform structure owners의 같은 pass에 source/native byte
facts를 연결하는 일이다. Native/packed/미지원·source 없음의 applicability/coverage를
분명히 구분하고 `M13_SOURCE_REVIEW`의 실제 Policy 매핑·전이 graph/cache/promotion/CLI와
최종 native/hosted를 닫아야 한다. 현재 함수/실제 시험 소비자는 연결됐지만 production
생태계 consumer·전체 S04 acceptance는 여전히 OPEN이다. 새 CI/Host 영구 설치·언어 version
owner 변경은 없으며 명명/배포 완료로 쓰지 않는다.


### 기존 member pass의 source 연결 — 착수 대조 (`f8a9546`)

| 이번 작은 범위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| npm 기존 archive pass의 검증된 source 반환 | NO | NO | NO | NO | 기존 `archiveCapture.entry/finish`에서 선택 source만 유한 보관; 같은 CRC/원본/경로 검증 성공 뒤 반환; 변조·한도·선택 파일 회귀 |
| 전체 생태계 content 분석 소비 | NO | NO | NO | NO | 나머지 member owners, no-source/unsupported/packed/native applicability, 실제 scanner·Policy/cache/promotion/CLI/전이 graph 연결 |

기존 `InspectArchive`/selected entry 재생 경로와 hash/copy loop는 재구현하지 않는다.
npm 기존20MiB/file·200MiB decoded·manifest 포함10000-entry 한도는 source subset에도
적용되며 backend64MiB/512MiB/10000보다 좁다. 추가 budget/owner를 만들지 않는다.
Source suffix는 parser 선택 힌트이며 안전성·실행 역할 증명이 아니다. 이 범위에서는
source 반환만 구현하고 source 없음·지원하지 않는 파일을 clean으로 판정하지 않는다.
기존 구조 전용 consumers에 content 완료를 암묵적으로 부여하지 않는다.

Go `CheckedFiles`의 정규화된 ZIP member와 현재 `.go` bounded read·CRC·syntax 검사는
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES로 보존한다. Cargo `ArchiveFiles`의
전체 bounded byte map·manifest/CRC 검증도 같은 기존 상태다. 이 두 owner의 source 반환은
별도 NO/NO/NO/NO이며 MISSING은 같은 입력 읽기의 parser 대상 subset 반환·실패 시 폐기·
기존 report 동일성·선택 순서 회귀다. Go testdata의 syntax 적용 제외는 유지하지만
그 bytes를 content 검사에서 삭제하지 않는다. 추가 해제/graph 예산 소비를 만들지 않는다.

첫 foundation36.578s는 기존 build/format 단계를 지난 뒤 npm→content의 미등록 의존
연결을 검출했다. 실제 사용하는 그 edge만 등록하고 content 자체도 Domain/Port/runtime
identity 외 product 및 external dependency를 거부하도록 기존 architecture 검사에 명시한다.
새 sandbox/policy/bootstrap 연결을 허용하는 변경이 아니다.


다음은 source-return API만의 named local 결과다.

- npm `InspectArchiveSources`는 기존 capture/hash/copy에 선택 body 보관만 추가하고,
  전체 finish 성공 뒤에만 inventory/source를 반환한다. Go `InspectSources`는 기존
  `.go` read를 재사용하며 추가 언어도 같은 checked ZIP member에서 CRC를 확인한다.
  Testdata의 기존 syntax 제외는 유지하되 내용 검사용 bytes를 버리지 않는다. Cargo는
  `ArchiveFiles`의 검증된 byte map을 선택하고 선택 key만 안정된 순서로 정렬한다.
- Targeted Race15.389s/architecture1.918s/foundation29.860s/security13.965s/
  vet0.808s/pinned Staticcheck2.664s PASS다. 같은 source hash와 원본 receipt는 public
  review의 `member_sources`에 결합했다. 실패한 foundation36.578s도 보존한다.
- Unit test의 임시 intake는 t.TempDir에서 회수하고 별도 native binary/입력 사본은
  생성하지 않았다. 기존 qualified runtime/CPP/전체 CI를 재실행하지 않았다.
- API 구현과 지정 회귀만 확인했으며 production scanner/Policy 연결은 미실시다.
  PyPI/GitHub/Terraform, native/packed/unsupported/no-source applicability는 미완료다.
  전체 S04는 계속 NO/NO/NO/NO이며 이름·배포 준비 완료가 아니다.


### PyPI source member 연결 — baseline `715e457`

기존 wheel `InspectWheelForSource`→`parseWheelMetadata`→`validateRecord`→
`streamRecordFile`의 SHA/CRC 및 StaticInspector의 graph uncompressed/file charge는
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES인 기존 owner로 재사용한다.
Source byte 반환은 NO/NO/NO/NO이며 MISSING은 같은 RECORD hash stream의 유한 source
보관·전체 검증 뒤 반환·derived wheel과 graph accounting 유지·실패 시 폐기다.

Sdist의 source 반환도 NO/NO/NO/NO다. 기존 `InspectSdist`는 bounded PAX/metadata와
compressed digest를 검사하지만 tar EOF 뒤 gzip을 drain하지 않는다. 새 반례
`s04-pypi-sdist-trailer-red`3.672s에서 CRC 변조·두 번째 gzip member·compressed tail·
nonzero decoded tail 네 가지가 모두 잘못 수용됐다. Source 반환 전 같은 owner의
trailer validation gap을 닫는다. 새 압축 해제기나 기존 byte/file 예산 확대는 하지 않는다.
이 발견을 이전 모든 설치의 감염이나 source 내용 분석 완료로 확대하지 않는다.


PyPI source 반환의 named local 수정·검증:

- Wheel/derived-wheel은 같은 RECORD stream의 digest와 source buffer를 함께 쓰고, 기존
  typed wheel과 graph uncompressed/file charge를 유지한다. 전체 검증이 실패하거나
  source file64MiB/total512MiB/10000 bound를 넘으면 전체 선택을 폐기한다. 그 한도는
  실제 수집자와 기존 backend가 같은 Port 상수를 소비하며 값/런타임 동작은 불변이다.
- Sdist는 기존 PAX pass에서 선택 source를 보관하고 tar EOF 뒤 같은 bounded decoder를
  drain해 CRC·zero padding·compressed exhaustion을 확인한다. 앞 RED 네 반례는 최종
  Race에서 모두 거부된다. Sdist static profile만 v2로 바꾸어 과거 불완전 결과를 현재
  requirement에 사용할 수 없게 했고 기존 wheel profile은 유지했다.
- Targeted Race27.189s PASS 후 초기 foundation64.003s가 build/import/architecture/docs를
  확인했다. 이후 sdist-only profile/derived-source test 추가를 포함한 최종 PyPI 두 owner의
  전체 Race14.398s/security34.303s/vet13.395s/pinned Staticcheck30.339s가 PASS다.
  Foundation 전체를 최종 source에서 재실행한 것으로 쓰지 않는다. Scanner 실제 실행/
  CPP/runtime 값은 이번 source 수집 수정과 별개인 이전 scoped 결과를 재사용한다.
- 정확한 source/raw는 public review의 `pypi_member_sources`가 소유한다. Unit 임시 입력은
  t.TempDir에서 회수하며 새 native binary·Host 설치·CI/push는 없다. 현재 API 반환만
  구현/검증했으므로 production analyzer/Policy와 전체 S04는 계속 미완료다. 다음은
  GitHub/Terraform과 native/packed/unsupported/no-source 사실, 모든 소비자의 연결이다.


### GitHub Release·Terraform member 다음 baseline — `bd42c02`

GitHub `StaticInspector`의 bytes-first format/header/path/type/개수/크기 검사는 기존
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES의 한정 범위를 보존한다. 하지만
현재 ZIP pass는 member를 열지 않고 tar.gz는 tar EOF 뒤 gzip을 drain하지 않는다.
CRC/전체 payload 검증·source 반환은 NO/NO/NO/NO이며 먼저 직접 반례로 확인한다.
기존 `openAsset`/format 검사 pass를 수정하며 새 extract/Host 실행 경로를 만들지 않는다.

Terraform `InspectPackage`는 이미 모든 ZIP member의 CRC/size와 flat layout·ELF bounds,
표준 H1/ZH 및 실행 파일 digest를 검증하고 전체 bounded `Files` map을 반환한다. 이는
기존 YES/YES/YES/YES 범위다. 여기서 source/native content facts·적용/미지원 범위의
반환/Inspection/Evidence 연결은 NO/NO/NO/NO다. Map을 재사용하고 다시 ZIP을 풀거나
동적 providerhelp·서명·기존 승인/graph 예산을 대체하지 않는다.


GitHub Release source-return의 named local 결과:

- 직접 RED0.840s는 ZIP CRC·gzip CRC·두 번째 gzip member·compressed tail 네 가지가
  기존 header-only pass에서 잘못 수용됨을 보였다. 같은 pass에서 전체 regular member를
  기존64MiB/file·200MiB/10000-entry 한도로 읽고 CRC와 bounded gzip padding/exhaustion을
  확인한다. Source suffix는 parser 선택 힌트이며 그 subset만 보관한다.
- Source API는 기존 held asset FD의 압축 SHA256을 acquired subject와 대조한다. 기존
  openAsset에서 os.Root 범위의 nonblocking open과 inode/size 재검증을 사용해 symlink
  ancestor 밖을 읽거나 다른 파일을 source로 반환하지 않는다. Static profile v2는 이전
  header-only 결과가 현재 requirement를 만족하지 못하게 한다. 별도 추출/실행은 없다.
- GitHub inspector 전체 Race3.250s, 설치/교체/rollback/과거 approval의 표적 promotion
  Race22.379s, 현재 import 경계2.320s, canonical security16.779s/vet0.446s/
  pinned Staticcheck1.369s가 PASS다. 같은 불변 runtime/CPP/전체 foundation/CI는 반복하지
  않았다. Unit 임시 Intake는 t.TempDir로 회수하고 새 native binary/Host 설치는 없다.
- Source-return만의 성공이며 production analyzer/Policy·native/packed/unsupported/
  no-source applicability 및 Terraform 연결은 계속 미완료다. 정확한 source/raw는
  public review의 `github_member_sources`가 소유한다. 전체 S04는 OPEN이다.


### Native capability·Terraform 첫 연결 — 착수 지도 (`f56bb86`)

| 작은 범위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| ELF 내용 facts/packed limitation | NO | NO | NO | NO | 검증된 member의 digest·유한 ELF64LE amd64 layout/권한·UPX marker/section 사실·Evidence/미완료·정상/변조 회귀 |
| Terraform 현재 검사·retained 소비 | NO | NO | NO | NO | 기존 validated Files map의 executable을 내용 검사에 전달, 사전 CheckPlan·report 합침, 과거5-record 승인 거부·현재 profile 결합 |

기존 Terraform `InspectPackage`/`inspectProviderELF`는 구조 검증을 유지하며 대체하지 않는다.
현재 반환형은 내용 분석 facts를 갖지 않는다. Inspection은 이미 메모리에 있는 검증된
실행 파일 bytes에서 Go 표준 ELF reader로 유한 metadata만 읽는다. 압축 해제·서명 검증·
graph charge·동적 실행을 반복하지 않는다. 디스어셈블러/디컴파일러를 만들지 않는다.
Source AST와 다른 layout/byte-marker 범위이며 lifecycle section 자체는 악성이 아니다.
허용하지 않는 metadata 크기/형식이나 packed 내용은 완료 0건으로 처리하지 않는다.

현재 cache가 Evidence ID 다섯 개와 optional advisory 하나만 소비하므로 같은 연결에서
native coverage/signal summary의 정확한 ID·현재 profile도 요구한다. Profile을 복제하지
않고 content의 선언 helper만 재사용한다. Cache가 analyzer/Policy를 실행하는 연결은
만들지 않는다. 나머지 생태계·전체 source applicability·Policy 최종 mapping·CLI/native/
hosted qualification은 별도 MISSING으로 유지한다.


Native/Terraform의 named local 결과:

- `content.InspectELF`는 이미 검증된 member digest를 대조하고, 유한 ELF64LE amd64
  program128/section4096/section-name1MiB 범위에서 권한/layout·lifecycle·UPX 표식을
  기록한다. 단순 marker나 init/fini 자체를 악성으로 확정하지 않는다. 위험 layout은
  `M13_BINARY_REVIEW`이며 marker+대응 section으로 인식한 UPX packed 내용은 required
  INCOMPLETE와 별도의 completed signal을 남긴다. 다른 형식은 UNSUPPORTED/NOT_EXECUTED,
  malformed/과다 metadata·digest 불일치는 완료 0건이 아니다. 디컴파일/모든 packer 탐지는
  범위 밖이며 그보다 강한 안전성·탐지율로 표시하지 않는다.
- Terraform은 기존 validated Files map의 실행 파일만 전달하며 static→native→기존 서명
  prerequisite/동적 probe를 유지한다. Native 두 check를 사전 선언한다. Cache는 기존
  required5 + native coverage/signal2를 요구하고 현재 선언 helper의 profile과 정확한
  kind를 대조한다. Optional advisory는 필수 증거를 대체하지 않는다. Cache의 content
  의존은 읽기 전용 선언에 한정하며 분석기/Policy 실행 연결은 없다.
- Initial target2.328s/architecture1.863s/foundation41.171s 뒤, compressed-debug 회귀
  RED0.562s가 후보의 Section.Size(확장 크기)/FileSize(파일 크기) 혼동을 검출했다.
  실제 표준 ELF API의 FileSize로 범위를 확인하며 debug payload를 inflate하지 않는다.
  최종 관련 owner Race38.177s/security27.394s/vet16.337s/Staticcheck22.905s PASS,
  bounded fuzz23.142s(20초 예산/168595exec/no panic) PASS다. 초기 foundation은 이후
  수정까지 재실행한 결과로 확대하지 않는다.
- Old positive approval RED1.805s는 이전 synthetic5 증거가 새 요구를 충족하지 못한
  fixture 갱신 지점이다. 원래 보호를 낮추지 않고 현재 fixture에 실제 parser 결과와
  선언을 연결했다. Cache targeted16.691s는 누락/과거5·정상7/optional·실제 Store로
  재작성·재해시한 wrong profile/kind 거부를 확인했다. 최초 cache-red9.296s는 원래
  negative-only 검사여서 PASS였으며 제품 RED 증거로 쓰지 않는다.
- 실제 Inspector 진입점의 controlled Data/port 모델은 runtime 미준비 때도 native
  coverage/signal과 RWX Finding이 남고 required dynamic은 미완료임을 확인했다.
  V1/1.095s는 test TempDir 권한이 private700 계약을 만족하지 않아 실행 전에 실패했다.
  Fixture만 chmod700으로 고친 V2/1.918s는 두 case PASS다. 공개 서명/격리 실행 성공은
  아니다. 이후 단순 fixture 권한 수정은 production code를 바꾸지 않았다.
- Unit 임시 intake/승인 사본은 t.TempDir로 회수했고 native ELF를 Host에서 실행하거나
  별도 native trial binary·Host 설치·CI/push를 만들지 않았다. 다음은 다중 native/나머지
  생태계의 byte facts와 source/no-source/unsupported/packed inventory, AST 생산 연결,
  Policy/캐시·전이 graph/CLI 및 grouped native·hosted다. 전체 S04는 계속 OPEN이다.


### 현재 내용 신호의 Policy 분류 — baseline `c00dfc8`

Native가 현재 Terraform 경로에 신호를 반환하기 시작했으므로 다른 생태계 확장 전에 그
현재 call path의 분류 gap을 닫는다. 순서를 조정해도 전체 S04의 누락/필수 범위는 줄이지
않는다. 기존 M3/M6 판정·필수 검사·strongest block과 M4/M5 set 전파는 기존
YES/YES/YES/YES 범위로 재사용한다. `M13_SOURCE_REVIEW`/`M13_BINARY_REVIEW`의
명시적 MANUAL_REVIEW 분류는 NO/NO/NO/NO다. 현재 unknown default가 BLOCK이다.
MISSING은 설계 §6.1의 불확정 pattern 분류, 두 entry Policy의 version 갱신, shared
생태계/부분 검사·미선언·무결성/강한 finding 우선·set 전파 회귀다. ALLOW 조건,
unknown finding의 기본 차단, 동적/서명 필수 검사를 바꾸거나 정책 engine을 재작성하지 않는다.


Policy 명시 분류의 named local 결과:

- Shared M3의 실제 source IDs(여섯 생태계 및 별도 PyTorch profiles)와 M6에서 두 content
  code/완료·required 불완전·미선언을 대조한 RED0.839s가 기존 unknown BLOCK 분류를
  확인했다. 기존 switch에 두 review code만 등록하고 M3 version4/M6 version3으로
  갱신했다. Unknown/default BLOCK·무결성·strongest finding·ALLOW 조건은 유지한다.
- 전체 Policy Race3.145s, version이 주입되는 GitHub 설치/교체/복원 표적 Race20.665s,
  canonical security14.625s/vet0.704s/Staticcheck1.246s PASS다. 강한 finding 두 순서,
  무결성 우선, M4/M5 set 전파와 NewVerifiedSet의 review 승격 거부를 확인했다.
- 이 작은 분류 범위만 YES/YES/YES/YES다. Source/native 입력 수집·다중 파일·streaming/
  applicability와 전체 production/CLI/native/hosted는 여전히 MISSING이다. 다른
  생태계 확장 전에 이미 작동하기 시작한 Terraform call path의 정확한 분류를 먼저
  연결했으며 전체 요구나 필수 검사·런타임 예산·언어 버전 갱신 owner를 줄이지 않았다.


### 확장자 없는 source 선택 — 착수 대조

현재 SourceLanguage/수집자는 suffix 힌트만 사용한다. 따라서 명시적인 Node/Python
shebang을 가진 확장자 없는 CLI 파일도 source 반환 대상에서 누락된다. 해당 선택 범위는
NO/NO/NO/NO이며 MISSING은 유한 prefix의 parser 힌트·기존 hash stream 재사용·원본
byte/offset 보존·정상/unknown 대조다. 기존 suffix 처리/Source runtime·한도/설치 역할을
대체하지 않는다. Shebang은 source parser 선택 정보이며 실행 승인·모든 entry surface의
coverage 증명이 아니다. 이 작은 입력 gap부터 닫고 전체 inventory/applicability를 이어간다.


유한 prefix 선택의 named local 결과:

- 기존 source-return owner 다섯 개에 최대512B의 명시 Node/Python shebang 힌트를
  연결했다. Suffix 문법(JSX/TSX 등)을 우선하며 Python 버전 숫자를 별도 고정하지 않는다.
  Npm/GitHub는 읽은 prefix를 같은 hash/payload에 포함하고 Go는 non-source remainder를
  저장하지 않고 CRC까지 소비한다. Cargo는 기존 검증 byte map을 그대로 사용한다.
- PyPI는 기존 sourceMember를 writer로 사용해 여러 조각의 prefix도 정확히 모은다.
  Source로 선정된 경우에만 기존64MiB/512MiB/10000 한도로 전체 body를 보관한다.
  Opaque DATA는 최대512B 판단 후 추가 body를 저장하지 않는다. 기존 전체 검증 실패 뒤
  폐기와 graph charge를 유지한다. 준비/검사 미완료를 source 없음으로 판정하지 않는다.
- Related Race7.629s/security18.235s/vet1.028s/Staticcheck3.673s PASS다. Corpus/새 native/
  CPP/전체 foundation/CI는 반복하지 않았다. Unit 임시 입력은 회수하며 새 Host 설치나
  영구 설정 변경은 없다. Source selection만의 범위이며 metadata가 고른 비-shebang 파일,
  native 다중/streaming·전체 census/applicability·실제 AST 소비자는 여전히 미완료다.


### 다중 native member 수집 — baseline `2e74bb0`

| 작은 범위 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 다중 binary 결과의 유한 집계 | NO | NO | NO | NO | 기존 InspectELF 재사용, member별 Evidence identity, 일부 미지원/오류 뒤 확정 신호 보존, 중복/초과/취소 거부 |
| Cargo의 동일 archive pass native 반환 | NO | NO | NO | NO | 기존 ArchiveFiles map 재사용, source 반환과 native 결과 결합, archive 실패 시 미반환 |

현재 `InspectELF`의 두 check/Evidence ID는 단일 member용이다. 이를 member별 composite로
단순 연결하면 중복 check가 된다. 기존 layout 분석을 대체하지 않고, content에서 같은
두 capability를 member별 bounded fact로 집계한다. Body는 기존 검증된 map에서 참조하고
추가 추출/실행/graph charge를 만들지 않는다. Cargo의 기존 Inspect/InspectSources 계약은
보존하고 명시적인 content 진입점만 추가한다. 같은 파일을 binary와 source로 중복 선정하지
않도록 byte magic을 먼저 확인한다. Binary member가 0이라는 값만으로 N/A/완료를 만들지
않는다. 전체 적용 inventory/streaming 및 production AST/cache/CLI 연결은 후속 gap이다.

다중 native의 named local 결과:

- `InspectBinaries`는 기존 단일 ELF decoder를 재사용하고 두 aggregate check 아래
  member별 SHA/경로 hash·capability/status/limitation을 각각 보존한다. 중복 member·유한
  count/bytes 초과·취소를 거부하며 unsupported/손상/digest 불일치/packed는 전체 완료가
  아니다. 다른 member의 완료된 위험 신호와 근거는 남긴다. Native 선정은 ELF/PE/Mach-O/
  WASM/ar magic과 확장자 힌트이며 지원 decoder는 기존 ELF64LE amd64뿐이다.
- Cargo `InspectContent`는 같은 validated map을 재사용한다. Rust 이름으로 위장한 binary를
  source로 중복 전달하지 않고 두 native member를 구분하며 CRC/second member/취소 때
  내용 결과를 반환하지 않는다. Mode를 소유하지 않는 map에서 권한을 추측하지 않고0으로
  기록한다. Binary0은 적용 제외 근거가 없으므로 INCOMPLETE다. 기존 Inspect/InspectSources
  계약은 유지한다. 실제 AST 실행·source 완전성/생산 설치 경로는 아직 연결하지 않았다.
- Target0.841s 후 최종 관련 Race3.301s/security19.353s/vet0.601s/Staticcheck1.133s PASS다.
  Target은 후속 member 상태/Cargo fixture 추가 전 결과이며 최종 Race가 해당 변경을 포함한다.
  Unit intake는 t.TempDir에서 정리하고 Host artifact 실행/새 binary 공급/CI/push는 없다.
  집계와 Cargo DATA 진입점의 IMPLEMENTED/WIRED/QUALIFIED는 YES(명시된 로컬 범위),
  ACCEPTANCE_CLOSED는 전체 inventory/streaming/생산 소비자가 남아 NO다.


### PyPI native streaming — baseline `a27189b`

다중 집계는 구현돼 있고 Cargo DATA 진입점에 연결됐다. PyPI의 기존 MemberCapture writer는
source만 보존하므로 native 수집/연결/검증/완료는 NO/NO/NO/NO다. MISSING은 같은 RECORD/
PAX pass에서 native 판별, 단일 파일 유한 buffer의 분석 직후 해제, whole archive 검증 전
결과 미공개와 실패 근거 보존이다. `InspectELF`/다중 집계를 대체하지 않고 기존 집계 내부를
증분 호출 가능하게 분리해 재사용한다. Source collector의 기존 경계/typed wheel·sdist와
한 번의 graph charge를 유지하고 native로 위장한 source suffix는 content 진입점에서
byte magic 우선으로 선정한다. 검사/Host 실행/SDK 재빌드/새 CI를 만들지 않는다.

PyPI native의 named local 결과:

- 기존 다중 집계 내부를 `BinaryInspection`의 증분 Inspect/Report로 공유한다. 기존
  InspectBinaries는 같은 owner를 사용하며 단일 InspectELF/고정 capability는 유지한다.
  Body를 해제/덮어쓴 뒤에도 사실과 신호가 남고 반복 readback은 중복 근거를 만들지 않는다.
  후속 취소는 앞선 신호를 보존하면서 전체 coverage를 완료로 만들지 않는다.
- `InspectWheelContent`/`InspectSdistContent`는 기존 RECORD/PAX callback과 typed 반환을
  재사용한다. Native는 최대 한 파일 buffer로 분석한 뒤 즉시 비우고 source와 중복 보존하지
  않는다. 파일/count/전체 선택 한도는 기존64MiB/10000/512MiB이며 archive/graph의 더 좁은
  한도는 그대로다. Oversize를 버퍼링하지 않고 불완전/오류 및 이전 위험 사실을 남긴다.
- 정상 wheel의 두 member/위장 .py, byte-split·버퍼 해제·초과, sdist native, CRC/second
  member/취소·후반 metadata 실패의 미공개와 graph byte/file의 정확히 한 번 charge를
  확인했다. Target1.834s는 후속 fixture 전이며 최종3 owner Race5.294s/security18.972s/
  vet0.908s/Staticcheck2.349s PASS다. Formatter0.896s의 source_unchanged=false는
  지정된 두 Go 파일의 정규 format 변경이고 이후 검증 입력은 모두 불변이다.
- Incremental owner와 PyPI DATA 진입점은 해당 범위 IMPLEMENTED/WIRED/QUALIFIED=YES다.
  ACCEPTANCE_CLOSED=NO이며 전체 census/적용 제외 근거·실제 AST/production 소비자,
  나머지 npm/Go/GitHub native 연결과 grouped native/hosted는 MISSING이다. Unit 임시는
  t.TempDir로 회수하고 실제 artifact/Host 실행이나 새 CI/push는 없다.

Release 공급 후속 확인: 현재 lock은 자체 build의 core/source/notices 정확한 hash/size를
소유하지만 hosted에서 획득할 원본 주소는 없다. 로컬 frozen /tmp를 hosted 입력으로 가정하거나
floating upstream wheel로 대체하지 않는다. 네 asset workflow 연결은 원격 frozen 공급을
준비한 뒤 기존 materializer에서 검증해야 하므로 계속 MISSING이며 현재 검증을 배포/공식
provenance 완료로 표시하지 않는다. 다른 로컬 구현 작업의 중단 사유는 아니다.


### npm/Go native streaming — 착수 지도

현재 두 owner의 구조/무결성/source 반환은 구현돼 있지만 native member 연결은 각각
NO/NO/NO/NO다. MISSING은 기존 tar entry/checked ZIP pass의 prefix 선정·같은 body/digest
재사용·파일별 분석 후 해제와 전체 검증 전 결과 미공개다. npm archiveCapture와 Go inspect에
기존 BinaryInspection만 전달하며 별도 추출기·실행기·source/native 저장소를 만들지 않는다.
기존 public static/source 계약과 Go parser/testdata 의미, 압축/파일/graph 한도를 유지한다.

npm/Go native의 named local 결과:

- npm `InspectArchiveContent`는 같은 prefix/entry body/hash를 증분 native owner에 넘기고
  전체 gzip/압축 SHA/inventory 완료 뒤에만 반환한다. Member body를 추가 사본으로 보존하지
  않고 native와 parser source를 중복 선정하지 않는다. 기존 selectedBody/manifest 의미와
  20MiB/file·200MiB·10000 entries 한도는 유지한다.
- Go `InspectContent`는 같은 checked ZIP read/CRC/body를 재사용한다. Go parser와
  testdata의 기존 적용 규칙은 유지하며 native는 source 배열에 중복 보존하지 않는다.
  Native 부재는 적용 제외가 아니고 필수 INCOMPLETE로 남긴다. 두 owner의 기존
  Inspect/InspectSources 결과 계약을 보존했다.
- 두 member/위장 source·분리된 SHA 근거, 기존 source inventory 보존, late CRC/압축
  SHA/경로·취소/Go syntax·identity 실패의 내용 미공개와 ZIP helper CRC를 확인했다.
  Related Race18.354s/security14.737s/vet0.590s/Staticcheck1.814s PASS다.
  Unit 임시는 t.TempDir로 정리됐고 별도 native 실행/추출/공급·CI/push는 없다.
  이 DATA 진입점 범위는 IMPLEMENTED/WIRED/QUALIFIED=YES지만 ACCEPTANCE_CLOSED=NO다.
  GitHub native와 전체 census/AST·생산 graph/cache/promotion/CLI·hosted는 MISSING이다.


### GitHub native streaming — 착수 지도

기존 static v2의 held intake/전체 SHA·ZIP CRC/TAR payload/trailer/source 반환은 유지한다.
Native 선정/분석은 NO/NO/NO/NO이며 MISSING은 같은 member body의 BinaryInspection 전달,
ZIP/TAR mode facts·압축 subject 결합과 raw ELF의 유한 body 분석이다. 기존 Inspector/
readArchiveMember에 같은 owner를 전달하고 새로운 archive reader/추출기/실행기를 만들지
않는다. Raw ELF가 기존 내용 분석 파일 한도64MiB를 넘으면 완료를 주장하거나 한도를 늘리지
않는다. Whole static failure/cancel 때 provisional native 결과를 공개하지 않는다.

GitHub native의 named local 결과:

- `InspectContent`는 기존 held asset SHA와 ZIP/TAR member read를 유지하고 파일별
  native facts를 증분 수집한다. Source 배열에 binary를 중복 보존하지 않으며 raw ELF도
  동일 held FD의 유한 body와 acquired digest로 분석한다. Static/legacy source API의
  지원 형식/결과 계약은 유지한다. Member별 byte 수집 뒤 body를 추가로 보존하지 않는다.
- Raw/선정 member의 선언 크기가 분석 한도를 넘으면 버퍼를 할당하거나 무결성 변조로
  오인하지 않는다. BinaryMember의 선언 Size를 기존 증분 한도 검사에 전달하고 selected/
  unanalysed 상태와 오류를 남긴다. PyPI의 같은 oversize 경로도 이 필드를 재사용한다.
- ZIP/TAR 각각 두 member(위장 .py 포함), raw ELF·기존 source/census 보존, 마지막 CRC
  실패 때 provisional 위험 사실 미반환, digest/run symlink/취소 거부를 확인했다.
  최종 content/GitHub/PyPI Race7.315s/security17.285s/vet0.676s/Staticcheck2.647s PASS다.
  Format0.901s는 지정 두 Go 파일의 정규 변경이며 이후 검사 입력은 불변이다. Unit intakes는
  t.TempDir로 정리하고 Host artifact 실행/새 SDK·CI/push는 없다.
- 이 DATA 진입점의 IMPLEMENTED/WIRED/QUALIFIED=YES(명시된 범위),
  ACCEPTANCE_CLOSED=NO다. 모든 archive의 applicability census/metadata-selected source와
  실제 AST/전이 graph·cache/promotion/CLI 및 grouped native/hosted는 계속 MISSING이다.


### Cargo metadata-selected source — baseline `c24e006`

전체 applicability 증명 전에 normalized Cargo.toml의 명시 path/build도 선정해야 한다.
현재 확장자/shebang만의 수집은 이 범위 NO/NO/NO/NO다. MISSING은 manifest가 지정한
확장자 없는 Rust target/build script의 수집·기존 경로 검증·빠진/탈출 path 거부다.
[공식 target path](https://doc.rust-lang.org/cargo/reference/cargo-targets.html#the-path-field)와
[build 필드](https://doc.rust-lang.org/cargo/reference/manifest.html#the-build-field)를 확인했다.
기존 artifact/cargo의 TOML parser·validateManifestPath와 검증된 ArchiveFiles map을 재사용한다.
Source metadata 해석을 inspection/content에 새 TOML 의존으로 넣지 않고 기존 owner에서
한정된 선언만 돌려준다. Legacy static/source API와 프로젝트 source policy는 변경하지 않는다.

Cargo 선언 source의 named local 결과:

- `artifact/cargo.DeclaredSourcePaths`는 기존 TOML parser·validateManifestPath와 유한
  manifest/target bounds를 재사용한다. 명시 lib/bin/example/test/bench path와 build script를
  원래 검증된 map 안에서만 선정하며 omitted path와 명시 empty path를 구분한다. Bool build
  값은 명시 path를 만들지 않고 잘못된 타입·탈출/부재 경로는 오류다. Cargo/Host 실행은 없다.
- 현재 `InspectContent`가 이 선언을 source 선택에 합쳐 확장자/shebang 없는 target을 Rust
  DATA로 반환한다. Native byte 선정은 우선하며 parser source로 중복 보존하지 않는다.
  기존 Inspect/InspectSources와 프로젝트 source policy는 그대로다.
- Old consumer + 새 fixture RED0.851s는 여섯 선언 source가 빠짐을 보였다. 새 consumer의
  두 owner Race5.609s/security23.322s/vet1.001s/Staticcheck1.459s는 PASS다. 이후 test-only로
  committed itoa1.0.17의 새 content 진입점 readback을 추가한 표적0.614s도 PASS다.
  앞 Race를 후속 test source의 재실행으로 쓰지 않는다. Format0.916s는 해당 Go1 파일의
  정규 변경이며 이후 검사 입력은 불변이다. Unit intakes는 t.TempDir로 회수했다.
- 이 선언 경로의 IMPLEMENTED/WIRED/QUALIFIED=YES는 위 로컬 DATA 범위다. 전체
  ACCEPTANCE_CLOSED=NO이며 다른 생태계 metadata/census·실제 AST/생산 cache/CLI/native/
  hosted는 계속 MISSING이다. 새 CI/push·runtime pin/Host 변경은 없다.


### Cargo 전체 content census / source 분석 — 착수 지도

파일 단위 source/native 입력은 구현돼 있지만 binary0/source0의 적용 근거와 실제 AST 연결은
NO/NO/NO/NO다. 먼저 content에 immutable한 파일 사실 목록을 두고 Cargo의 기존 검증 map
전체와 선언 source를 결합한다. 같은 subject·모든 file/digest/분류·선택 source 집합이 맞을
때만 zero-target 근거를 사용할 수 있다. 불명/미지원 source는 완료0으로 바꾸지 않는다.
기존 InspectELF/SourceAnalysis/검사 선언·Composite report를 재사용하며, 새 scanner/reader/
workflow engine을 만들지 않는다. Legacy source/binary helper의 proof 없는 빈 입력은 계속
INCOMPLETE다. 생산 bootstrap/cache·CLI 활성화는 같은 근거와 검사 목록이 완성된 뒤 연결한다.

Cargo census/source 진입점의 named local 결과:

- `content.Inventory`는 전체 regular file의 subject/path/size/digest/선언 parser·native/
  unsupported/DATA 분류를 immutable하게 보존하며 body 사본은 갖지 않는다. 원래 archive
  owner가 전부 검증한 사실만 공급하며 이 constructor가 새 reader/서명 authority가 되지
  않는다. Source 선택은 수·중복·언어·바이트 SHA와 정확히 일치해야 한다. 고정 parser 밖
  source/실행 힌트·case variant는 부재로 바꾸지 않는다. 이는 선언/format-hint census이며
  모든 임의 runtime decoding/생성 코드를 발견한다는 증명이 아니다.
- 기존 source/binary helper는 proof 없는 빈 입력을 계속 INCOMPLETE로 둔다. 정확한
  전체 census가 zero를 증명할 때만 source/binary 대상 부재를 완료된 적용 판정으로 보존한다.
  Source 요약에는 no_applicable_source와 runtime_complete=false/report_valid=false를
  함께 적어 scanner 실행 성공으로 오인하지 않는다. Unknown/unsupported source는 이
  부재 판정을 사용할 수 없다. Native는 count/개별 digest가 census와 달라도 완료가 아니다.
- Cargo는 같은 validated map과 선언된 Rust source를 census에 결합하고 native/새 required
  content-inventory 결과를 함께 반환한다. `InspectAnalyzed`는 같은 source DATA와 census를
  기존 fixed analyzer/strict report 해석에 전달한다. Controlled unavailable scanner가 정확한
  7source와 subject로1회 호출되고 source required는 INCOMPLETE이며 앞선 facts는 남는다.
  이는 actual scanner/native 성공이 아니고 bootstrap/실제 cache·promotion/CLI는 미활성이다.
- 이전 flat InspectBinaries는 증분 owner 전환 뒤 production caller가 사라져 제거했다.
  같은 decoder/증분 owner를 테스트/소비자가 직접 재사용하며 사본 framework를 두지 않는다.
- Target1.001s 후 format0.885s(지정7 Go의 정규 변경), foundation76.088s와 관련6 owner
  Race25.816s/security24.704s/vet1.002s/Staticcheck1.412s PASS다. 이후 test-only로 native
  digest 바꿔치기/신호 보존·unsupported source0 거부를 추가한 content 전체 표적0.716s도
  PASS다. 앞 Race/foundation을 후속 test의 재실행으로 쓰지 않는다. 임시 intake는 t.TempDir로
  회수했고 별도 artifact/Host 실행·SDK rebuild·CI/push는 없다.
- Census/Cargo DATA·호출 연결의 IMPLEMENTED/WIRED/QUALIFIED=YES는 위 로컬 범위다.
  ACCEPTANCE_CLOSED=NO이며 다른 archive census/metadata, 실제 positive AST·전체 전이 graph/
  cache/promotion/CLI·native/hosted와 release 공급 원본은 MISSING이다.


### Cargo production/retained 연결 — 착수 지도

현재 content census/InspectAnalyzed는 구현돼 있지만 default bootstrap·사전 expected check·현재
승인 캐시의 필수 source/native/census 근거 검증은 NO/NO/NO/NO다. 기존 StaticInspector에
명시 analyzer 구성을 전달하고 기존 ExpectedChecks에 동일 다섯 content check를 합친다.
기존 Linux bootstrap의 executor/observer/resource adapter로 SourceAnalyzer를 생성하며 UI/
타 platform의 무관한 준비 경로를 확장하지 않는다. Cache는 기존 Record/ReadReference와
frozen plan을 유지하고 같은 현재 declaration의 필수 summary5/현재 profile·kind를 stage 전과
current readback에 확인한다. Historical original 확인(requireProfiles=false)은 현재 승인으로
승격하지 않으며 별도로 유지한다. 기존 fixture authority는 명시 synthetic이며 실제 scanner
성공으로 표시하지 않는다. 버전 pin/한도·observer·Host 설치와 CI는 변경하지 않는다.


### 공통 build fixture의 기존 환경 근거 누락 — 조사/최소 교정

Current Cargo production 묶음 Race231.696s에서 content/Cargo/bootstrap은 PASS이고 promotion은
Go/Cargo build fixture·Unix socket EPERM으로 FAIL이다. 처음 두 normal case를195b0d0의
cache/fixture로 overlay해 재현한8.973s도 같은 environment observation incomplete다.
따라서 새 Cargo cache 보강이 도입한 실패로 쓰지 않는다. 기존 shared goOutputBackend가
환경 상태0을 반환하지만 현재 projectbuild Inspector는 두 profile/VFS coverage를 요구한다.
Production invariant는 유지하고 이 synthetic backend에 두 상태를 공급하며 늘어난 workspace
Evidence2를 기존 정상 assertion에 반영한다. Socket 거부는 별도 실행 환경 한계로 분리한다.

공통 build fixture의 한정 교정 결과:

- Shared synthetic backend에 minimal/development 상태와 VFS coverage를 추가하고 정상
  operation의 Evidence3→5 assertion만 교정했다. Production/required 환경 판정은 변경하지
  않았다. 모든 기존 Go/Cargo build output·원복·위험 Mode 표적15.317s는 PASS다.
- Socket이 허용되는 정상 권한에서 promotion 전체 Race237.832s(Go test199.000s)가
  PASS다. 첫 full231.696s의 실패와 socket EPERM을 보존하며 새 Cargo 내용 검사와
  무관한 기존 fixture 누락의 대조 근거는 baseline-red8.973s다. 다른 세 package는 첫
  Race에서 이미 PASS했으므로 같은 입력으로 재실행하지 않았다.
- 이 두 test 파일의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=YES는 모의
  계약 동기화 범위다. 실제 격리 build/전체 S01/S08·S04 qualification으로 확대하지 않는다.
  별도 test artifact는 생성하지 않고 unit intake는 t.TempDir로 회수했다.

현재 실제 최초 실패는 공개 Cargo의 단일 Rust AST에서 rule-engine-census다. Native-v1
25.238s는 FAIL/검사 후 Policy·cache 미도달이며 noSKIP/helperwait0/Host 동일·own copy
240578517B 제거를 확인했다. 여섯 언어 동시 시험을 단일 언어 완료로 전용하지 않고
고정 도구의 actual engine 목록과 입력 언어를 대조한다. 전체 S04는 계속 OPEN이다.


### Actual 단일 언어 engine census — 원인 확정 / 수정 지도

Cargo native-v2/23.762s의 bounded projection은 Rust4source에서 실제 engine row가 고정
haa-s04-rust-download-shell/OSS 하나임을 보였다. 기존 decoder는 입력 언어와 무관하게
rule pack 전체 row 수를 요구했고, JS 단일 언어 unit fixture도 모든 언어 row를 제조했다.
따라서 실제 정상 입력을 incomplete로 거부했다. 입력의 검증된 언어 집합과 고정 catalogue
교집합을 expected로 만들고 정확한 membership/중복/engine·누락/비적용 추가를 검사한다.
새 scanner/규칙/pin/skip 예외를 만들지 않으며 source completion profile을 v2로 구분한다.
이전 여섯 언어 집합은 교집합=전체라는 동등 범위를 유지한다. Native-v1/v2는 모두 Policy/
cache 미도달 FAIL이며 Host 동일/helperwait0·자기 copy 정리를 확인했다.

Cargo production/단일 언어 교정의 named local 결과:

- Bootstrap은 기존 executor/observer/resource adapter의 고정 SourceAnalyzer를 구성하고
  같은 Cargo Inspector에 주입한다. 설정된 Inspector는 실행 전 static+native2+census+
  source2를 선언하며 기존 advisory/ProjectInspectService와 graph 전체 경로를 재사용한다.
  Source 없는 structural reader는 기존 계약을 유지한다. 사용자 필수 검사를 옵션으로
  내리거나 UI의 무관한 초기화에 새 runtime 준비를 추가하지 않았다.
- Source/native/census의 필수 summary5와 현재 Check/Profile/Kind를 기존 cache stage 전과
  current readback에서 확인한다. Summary ID는 producer/readonly declaration 한 곳의
  constants를 공유한다. 기존 frozen plan/Record/ReadReference를 유지하고 analyzer/Policy를
  cache에서 실행하지 않는다. Historical original(false)은 새 승인이 아니며 current(true)와
  구분한다. Real Store로 재기록·재해시한 wrong profile/kind, source 누락/중복·legacy stage의
  old cache RED9.456s 뒤 최종 current fixture/거부 검사가 PASS다. Fixture는 synthetic
  controller/Policy authority이고 실제 scanner 검증으로 쓰지 않는다.
- Actual 단일 Rust에서 확인한 engine census 차이를 고정 catalogue와 입력 언어의 정확한
  교집합으로 교정했다. 누락/중복/다른 engine과 비적용 rule 추가·바꿔치기는 계속 거부한다.
  Source profile만 v2로 구분해 이전 completion과 혼용하지 않는다. Rule/tool pin, Policy
  분류 버전, observer/권한/예산은 불변이다. 여섯 언어 각각의 정상6과 실패30 case는 old
  decoder RED0.897s→content/Cargo Race3.926s PASS이며 기존 all-six fixture도 보존했다.
- Initial 묶음의 content/Cargo/bootstrap Race는 PASS였고 promotion의 기존 환경 fixture/
  socket 실패는 별도 c8fe972와 앞 절이 소유한다. Current production의 security35.710s/
  vet21.522s/Staticcheck31.641s·tagged compile22.085s는 PASS다. 최종 engine 교정의
  security18.305s/vet0.959s/Staticcheck1.576s·current approval11.532s와 실제 import/port
  검사0.615s도 PASS다. 첫 architecture regex는 no-tests였으므로 검증으로 쓰지 않는다.
  Promotion 전체237.832s는 engine profile v2 전이며 이후 current Cargo approval 표적만
  확인했다. 같은 불변 다른 consumer/full/CPP를 재실행하지 않았다.
- Actual native-v1/25.238s·v2/23.762s는 engine census 실패로 Policy/cache에 미도달했다.
  V2의 bounded log는 Rust4 source와 실제 고정 Rust rule1/OSS를 보였다. 수정한 동일 공개
  itoa1.0.17의 V3/23.623s(Go test20.66s)는 acquisition/integrity·전체 census·실제 traced
  AST·M3/M4 ALLOW·Evidence7·vendor14/current cache readback을 PASS했다. noSKIP/
  helperwait0/Host 동일이며 각 own copy240578517/240582870/240591830B를 정리했다.
  성공한 여섯 언어 native를 단일 언어 성공으로 전용하지 않았고 현재도 whole CLI/hosted나
  악성 탐지율/모든 Cargo graph 성공은 아니다. SDK/CPP는 기존 qualification을 재사용했다.
- 이 Cargo production 연결의 IMPLEMENTED/WIRED/QUALIFIED=YES는 위 한정 범위다.
  ACCEPTANCE_CLOSED=NO다. 다음은 실제 CLI와 다른 archive의 metadata/census·AST,
  모든 cache/전이 graph/current-only/release 공급 및 grouped qualification이다. PyPI의 큰
  native member는 현재64MiB buffer capability로 전체 정상 지원을 주장하지 않으며 기존
  Artifact/profile 예산을 낮추거나 늘리지 않고 별도 bounded 접근을 마저 완성해야 한다.


### Cargo CLI current acceptance — baseline `22d34ef`

Production callback는 연결됐지만 기존 fetch/add native assertion은 이전 required3/Evidence2를
기대한다. Current CLI의 source/native/census·advisory/integrity/static 집합과 cold/warm·원본
보존·tampered cache 거부·후속 observed build는 NO/NO/NO/NO다. 기존 두 gate의 실제 계약
assertion만 current 필수 ID 집합과 summary 수로 갱신하고 기존 script/8분 context·10분 test
상한을 유지한다. Qualification은 이미 검증한 runtime/observer와 임시 staged tool을 재사용하며
새 CPP/pin/budget·영구 Host 설치·별도 CI를 만들지 않는다.

Cargo CLI actual-v1/93.410s는 cold31.784s/warm28.754s의 현재 필수6검사·ALLOW와
cache 변조 거부/원본 보존까지 PASS했으나 후속 build의 관찰 결과 부재로 FAIL이다.
Host 동일/helperwait0/자기 copy240586362B 제거를 확인했다. 준비 operator가 private
current에 scanner 두 파일만 공급하고 build가 무조건 요구하는 고정 responder를 빠뜨린
것을 source 대조로 확인했다. 이는 준비 누락이며 실제 실패 phase의 직접 진단은 아직 없다.
기존 responder owner를 현재 source로 빌드해 같은 private current에 공급하고 기존
--verbose를 build 시험에만 적용해 최초 실패를 보존한다. Product/권한/예산/CPP는 바꾸지 않는다.

Actual-v2/89.251s도 fetch cold29.929s/warm28.688s·변조 거부 뒤 build에서 FAIL이다.
추가 responder를 canonical stripped build 대신 일반 build로 만들어5971563B가 되었고,
기존 max4128768B를 초과한다. Native reader는 초과 공급물을 격리 실행 전에 거부한다.
이 준비 오류를 제품/관찰기 오류로 단정하지 않는다. 기존 release/CI의 CGO0·trimpath·
ldflags=-s -w build를 적용하고, 큰 시험 전 기존 CheckResponderSource를 test preflight에
추가한다. --verbose는 display-only이며 필수 검사·증거 권한은 불변이다. V2도 Host 동일/
helperwait0/자기copy246538115B 제거를 확인했고 최초 원본은 보존한다.

### 다음 npm census 연결 — 책임 지도 `22d34ef`

기존 archiveCapture.entry/finish의 단일 해시·완전 gzip 검증과 source/native DATA 수집은
IMPLEMENTED=YES다. 전체 content.Inventory/zero-native 근거 결합은
IMPLEMENTED=NO / WIRED=NO / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO다. 기존
entry가 regular member의 prefix/mode/digest를 소유하는 시점에 MemberFact를 만들고
finish가 완전 archive 검증 뒤 immutable Inventory를 구성한다. Native Report에 같은
census를 전달하고 기존 census Evidence를 합친다. InspectArchiveContent 시그니처와
legacy 두 API는 보존한다. 새 wrapper/재압축해제/graph 재계상/body 사본은 필요 없다.
직접 소비자는 archive_sources_test의 세 호출뿐이다. 정상 no-native 적용 제외·unknown
실행/source hint의 census 보존·binary disguise 우선·late CRC/digest/path/cancel의 사실
비노출을 확인한다. 기존20MiB/file·200MiB/total·10000 entry 한도는 불변이다.
아직 metadata-declared extensionless/main/bin/exports, configured 실제 AST·production/
project/cache 연결은 MISSING이며 이 단계에서 활성화/전체 npm 완료를 선언하지 않는다.
묶음 source가 고정되면 canonical foundation/security와 영향 Race/vet/고정 Staticcheck를
사용하고 작은 helper CI나 이미 불변인 CPP/native를 재실행하지 않는다.

Cargo CLI actual-v3/102.165s(Go99.12s)는 PASS다. Canonical CGO0/trimpath/
stripped4018336B responder의 사전 준비 검사 뒤 cold30.155s/warm28.997s의 필수6검사/
ALLOW·fetch 미컴파일·manifest/lock 보존, 변조 cache 거부·원본 approval 보존,
후속 observed build38.532s/승인 ELF publication을 확인했다. Source/Host 동일·noSKIP/
helperwait0/자기copy244586456B 제거다. 같은 CPP138/고정 scanner 공급을 재사용했고
제품 수정/권한/한도 완화 없이 준비 누락·잘못된 build 설정을 교정한 범위다.
두 native 실패 원본은 보존한다. Add gate의 Evidence2→8은 compile만 확인했으며
actual single/transitive add 재검증은 아직 MISSING이다. Fetch CLI의 네 상태 YES는
이 exact 공개1crate 흐름에 한정하며 전 생태계/graph/hosted/전체 S04 완료가 아니다.

Npm census는 기존 byte/hash pass의 regular member facts로 동일 subject에 결합한다.
Native magic은 source suffix보다 우선하며 unknown interpreter·uppercase source hint는
검사 대상 부재로 사라지지 않는다. 완전 gzip/trailer·digest와 기존 ArchiveInventory
검증 뒤에만 immutable census/필수 Evidence를 발행한다. Native 없는 경우도 실제 전체
목록으로만 적용 제외를 확인한다. 시그니처/legacy source API·graph 예산은 유지했다.
동일 세 표적의 old consumer RED1.008s 뒤 전체 npm Race19.114s·canonical foundation
61.845s/security15.682s/vet0.513s/고정 Staticcheck1.642s는 PASS/source 불변이다.
새 mixed fixture는 regular6/source1/native1/unsupported2·subject digest를 확인했고,
기존 late CRC/digest/path/cancel 거부도 그대로 통과했다. Test intake는 t.TempDir로 회수했으며
별도 artifact/Host 실행/CPP rebuild·CI는 없다. Census의 IMPLEMENTED/WIRED/QUALIFIED=YES는
이 local DATA 범위고 ACCEPTANCE_CLOSED=NO다. 다음 metadata 역할·실제 AST/configuration/
production/project/cache 연결과 최종 전 생태계 qualification은 계속 MISSING이다.

### npm 명시 main/bin source 역할 — 착수 지도 `3df1dc5`

현재 suffix/shebang 수집은 package.json의 확장자 없는 명시 main/bin을 놓친다.
이 역할의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=NO다. 기존 artifact/npm의
이미 npm/Bun 소비자가 공유하는 bunPackageBins와 validInventoryPath를 그대로 재사용한다.
새 bin parser/실행 resolver나 이름 정리 리팩터링은 만들지 않는다. ArchiveInventory의
검증된 manifest/regular entries로 명시 main/bin target 집합을 만들고, 기존 capture가
이미 보유한 source/native는 재사용한다. 새로 필요한 대상이 있을 때만 동일 acquired
SHA·전체 trailer를 다시 확인하는 기존 capture로 그 대상 bytes만 추가 수집한다.
이 추가 읽기에는 graph 재계상·압축 해제/Host 파일 생성·기존 source body 재복사가 없다.
Manifest가 archive 뒤에 있어도 같은 결과를 내며 source/native 중복·unknown interpreter의
거짓 완료·late source 변경/불완전 읽기는 거부한다. 이미 capture한 source/native/census
최종 결합 전까지 모든 사실은 private다. Lookup과 대상 합치기는 map/단일 순회로 한다.
Main의 JSON DATA와 native 역할을 보존한다. 다른 export/import pattern·nested metadata/
module/browser 역할과 configured AST/production 연결은 별도 MISSING으로 남긴다.
[공식 npm package.json](https://docs.npmjs.com/cli/v11/configuring-npm/package-json/)과
[Node package entry](https://nodejs.org/api/packages.html)는 역할 의미의 근거이며, 직접
경로 수집을 모든 Node/bundler resolution 구현으로 표시하지 않는다. Root main/bin과
manifest first/last·중복 alias·JSON DATA·unknown shebang·없거나 unsafe target을 표적으로
먼저 확인하고 의미 있는 source 묶음의 canonical checks를 모아서 수행한다.

Npm root main/bin의 직접 역할은 같은 old source fixture RED1.132s 뒤 로컬 교정했다.
기존 npm/Bun 공용 bunPackageBins를 그대로 재사용하고 새 parser/이름 변경은 없다.
검증된 manifest에서 regular target을 O(n) map으로 합친 뒤 미수집 target이 있을 때만
기존 전체 authenticated reader로 그 bytes를 보완한다. 기존 source body·graph budget은
중복 계상하지 않으며 최종 census 생성도 이 수집 뒤 한 곳으로 이동해 두 번 만들지 않는다.
정상 manifest first/last·alias dedup·명시 main/bin·JSON DATA/unknown shebang/native 역할·
unsafe/missing/duplicate metadata의 사실 비노출을 확인했다. Canonical foundation64.850s/
security29.178s·artifact+inspection 전체 Race62.535s/vet2.519s/고정 Staticcheck5.817s는
PASS/source 불변이다. 추가된 거부 시험은 처음 RED 이후이며 그 RED의 입력으로 쓰지 않는다.
시험 intake는 t.TempDir로 회수했고 Host/native/CPP/CI·pin/권한/예산은 변경하지 않았다.
명시 root main/bin의 IMPLEMENTED/WIRED/QUALIFIED=YES는 위 DATA 범위이고 전체 metadata/
source inspection ACCEPTANCE_CLOSED=NO다. Exports/imports pattern·nested package metadata/
module/browser·directory bin과 configured 실제 analyzer/current cache/project/CLI는
계속 MISSING이다. 다음에는 이 역할 범위를 먼저 명확히 연결하고 production을 활성화한다.

### npm 나머지 metadata 역할 — 다음 구현 지도 `80e5f1e`

Root main/bin의 DATA 수집은 재사용한다. 나머지 export/import 조건·패턴, nested
package.json, module/browser·directory bin은 NO/NO/NO/NO다. 기존 capture에서 모든
package.json body를 같은 buffer로 보유하고 ArchiveInventory의 measured path/size/SHA와
결합해 기존 strictControlJSON(깊이/토큰/문서 한도)으로 읽는다. Artifact owner의 기존
DeclaredSourcePaths 한 곳을 확장하고 기존 추가 reader 한 번으로 합친 미수집 대상만
수집한다. 별도 Node resolver/Host lookup/실행 권한·graph charge·census 복제는 없다.
조건/배열의 모든 local 후보를 보수적으로 수집하고 imports의 외부 package 이름은
로컬 파일로 위장하지 않는다. URL 형태의 local export/import는 standard URL parser로
경로를 구분한다. JSON DATA/unknown interpreter/native 우선과 whole archive escape
거부를 유지한다. 패턴은 표준 RE2의 고정 literal+wildcard 합친 식으로 한 번 구성하고
전체 regular 목록을 한 번 순회한다. 반복 *의 동일 capture를 Node 실행 계약으로
확정하지 않고 안전한 source 후보 superset 수집으로만 표시한다. 식의 수/총 길이는
기존 entry/metadata 문서 한도 안에 제한하며 초과는 metadata 미완료다. Directory bin은
이미 검증한 경로 prefix로만 선택한다. 새로운 matcher framework·solver는 만들지 않는다.
중첩/지연 metadata·조건/배열·패턴/URL·외부 이름·JSON·directory bin·경로 탈출/변조
거부를 싼 표적부터 확인하고 actual analyzer/production 의미 묶음 전 불변 full/CI는
반복하지 않는다. Source 후보 수집은 실제 Node resolution/실행 안전성 증명이 아니다.

완료된 Cargo 시험 실행 파일7개의 전체 proc105 exe/fd/maps 조회 참조0/오류0 뒤
348974674B를 제거했다. 원본 SHA·삭제목록은 s04-cargo-completed-binary-cleanup-v1.json에
보존하며 current responder/policy·scanner 공급/SDK/cache/원본은 유지했다.

나머지 metadata DATA collector를 기존 경로에 연결했다. 새 선언 parser/Node resolver가
아니라 같은 ArchiveInventory.DeclaredSourcePaths가 measured nested body까지 확인한다.
조건/배열·URL query/decoded path·local/import 외부 이름·module/browser·directory bin과
slash를 포함한 wildcard source 후보를 한 번 합친다. 패턴은 표준 regex의 보수적 후보
집합이며 실제 Node의 조건 선택/반복 capture resolution 성공을 주장하지 않는다.
전체 메타 문서의 expression 수/길이는 기존 entry/ProjectControlLimit에 제한하고
초과 시 사실을 발행하지 않는다. Same buffer/추가 reader/census owner를 재사용한다.
Old conditional/nested 소비자 RED1.157s 뒤 positive target2.180s와 body 누락/추가/동일
길이 바꿔치기·nested escape/encoded escape/duplicate/malformed·전체 pattern count/
bytes 초과를 포함한 최종 표적1.329s는 PASS다. 첫 RED는 후속 거부 fixture의 입력이
아니다. Vet0.715s·고정 Staticcheck 결과는 원장에 연결한다. 이 단계는 IMPLEMENTED=YES/
WIRED=YES(로컬 DATA)/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다. 실제 source analyzer/
production/current cache와 묶음 profile·native/hosted가 남았다. 지금의 cheap 표적을
전체 qualification으로 바꾸지 않고 같은 입력 full/native/CPP/CI를 반복하지 않는다.

### npm configured source analysis — 착수 지도 `79a2ce3`

DATA/metadata 수집은 구현됐지만 default Inspector/사전 선언/프로젝트 Archive 경로의
실제 analyzer 소비는 NO/NO/NO/NO다. 기존 Inspector에 현재 Cargo와 동일한 선택적
ports.SourceAnalysis 구성을 추가하고 Inspect와 InspectArchive가 같은 content 경로를
사용하게 한다. 기존 source DATA API는 structural helper로 보존한다. 같은 최종 census와
source bytes를 content.InspectSources에 전달하고 static/native/census/source의 여섯
체크를 사전 선언한다. native 분석이 부분 실패해도 검증된 source/facts를 결합하며
최초 오류는 보존한다. Configured collector가 없는 legacy default는 기존 계약을 유지한다.
새 parser/추가 IO wrapper/Policy 해석은 만들지 않는다. 이번 단계의 직접 소비자는
설정된 Inspector와 기존 npm/PNPM/Bun project의 InspectArchive/ExpectedChecks 경로다.
실제 bootstrap 주입·현재 cache의 필수 근거/플랫폼 미준비 표시는 후속 연결이며, 이
port 연결의 controlled unavailable scanner를 실제 native 성공으로 표시하지 않는다.
두 진입점의 exact frozen6·한 번의 measured source 전달·실패 시 이전 근거 보존·legacy
보존을 싼 표적으로 확인하고 production/cache까지 모은 뒤 넓은 검증을 진행한다.

Configured Inspector/InspectArchive의 한정 연결과 표적1.341s는 PASS다. 실행 전 frozen6의
정확한 ID/required 여부와 실제6의 profile 결합, measured JS/Python 전달1회, controlled
scanner 실패의 원인/앞선 native+census 근거 보존을 확인했다. Source0은 전체 census에
근거할 때만 no-applicable이고 scanner 호출0·runtime 완료 주장이 없다. Unknown shell의
source0은 미완료로 유지한다. 이 모의 port는 실제 scanner가 아니며 production bootstrap/
현재 cache·넓은 묶음 qualification이 남아 IMPLEMENTED/WIRED=YES(local configured)/
QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다. Native/CPP/CI·영구 Host 설치는 하지 않았다.

### npm 실제 구성·현재 승인 근거 — 착수 지도 `830b1aa`

Configured port의 실제 default 주입·current cache 기준은 NO/NO/NO/NO다. 실제 생성
지점은 bootstrap.go/npm_client.go/pnpm_client.go/bun_client.go/npm_execution.go 다섯
곳이다. 같은 source analyzer/resource 구성과 platform 미준비 의미를 한 기존 bootstrap
파일 안에서 공용 생성 책임으로 묶고 실제 다섯 소비자만 연결한다. Linux는 기존 보호된
executor/observer/resource adapter를 사용한다. 다른 OS는 source 실행 미준비를 검사
실패/미완료로 유지하며 UI/도움말 초기화에 runtime 준비를 추가하지 않는다.
Npm/PNPM/Bun fresh Stage/Open은 이미 pnpm_cache_approval.go의
npmProjectCacheApprovalEvidence/projectCacheEvidenceFacts를 공유한다. 이곳에서 현재
content summary5/profile/kind를 요구한다. Cargo의 verifyCargoContentEvidence 본문을
기존 project_cache_evidence.go로 옮겨 archive 중립 이름으로 그대로 재사용해 validator
복제·새 Policy/검사 호출을 막는다. Cargo의 historical requireProfiles=false와 npm 계열
pnpm_retained.go의 ownership-only 원본 확인은 유지한다. 새 Stage/Open에서 이 원본
확인을 현재 approval로 전용하지 않는다. Existing frozen plan/identity/digest/Record·
ReadReference와 graph/권한/예산은 불변이다.
모의 정상 cache fixtures는 현재 check/evidence 계약만 반영하고 실제 scanner authority로
표시하지 않는다. Old Stage의 현재 근거 누락 수용을 표적으로 재현하고 누락·중복·kind/
profile 변조를 실제 재기록/재해시한 거부와 현재 CLI 정상·실패를 확인한다. Metadata/
source/구성/cache 묶음이 고정된 뒤 canonical foundation/security·영향 Race/vet/
Staticcheck 및 필요한 native 소비자를 모아 수행한다. 작은 helper CI는 만들지 않는다.

Npm production 묶음의 canonical foundation83.575s/security29.304s·artifact/inspection/
bootstrap 전체 Race72.054s·promotion 전체 Race204.707s(정상 socket)·vet5.533s/
고정 Staticcheck14.506s는 PASS/source 불변이다. Tagged native build5.799s 이후에는
gated CLI assertion 두 파일만 추가했고 앞의 full 결과를 그 새 시험 source에 전용하지 않는다.
Actual npm-v1/49.152s(부모45.96s)는 FAIL이다. 실제 current required 검사는 모두 완료/
M3·M4 ALLOW였지만 fresh cache에서 promotion이 거부됐다. Exact raw projection의
binary-content-signals는 optional/SUPPORTED/NOT_EXECUTED/
M13_BINARY_SIGNALS_UNAVAILABLE이며 native coverage와 census는 required/COMPLETED다.
기존 cache가 모든 optional도 COMPLETED를 요구한 것이 직접 거부 원인이다. Synthetic
fixture는 모든 content status를 완료로 만들어 이 실제 no-native 상태를 대표하지 않았다.
Current summary5/profile/kind 확인을 유지한 상태에서 이 exact optional 상태만 허용하고,
required native/census/source coverage를 명시적으로 요구한다. 다른 optional 미완료/
wrong limitation/capability는 계속 거부한다. Producer를 억지 완료로 바꾸지 않는다.
새 CLI assertion도 같은 optional 적용 제외 의미로 교정한다. Unit RED와 native-v1 원본은
보존하며 최초 실패 후 inspect 두 번째 parent는 failfast로 미도달이다. Helperwait0/Host·
source 동일·자기copy244627151B 제거를 확인했고 intake finalization 메시지는 후속
정리 상태와 구분한다. 넓은 불변 consumer/CPP/CI를 반복하지 않고 이 거부 경계를 표적으로
먼저 교정한 뒤 현재 CLI를 재검증한다. Whole S04는 계속 OPEN이다.

Optional no-native의 동일 fixture RED-v2/8.894s 뒤 영향 cache/Cargo metadata Race
19.037s가 PASS다. 최초 red1.033s는 잘못 쓴 enum/accessor의 compile 실패이므로 semantic
RED로 쓰지 않는다. Exact optional native absence만 수용하고 required native/census/source
플래그를 낮춘 입력과 wrong code/unsupported/다른 optional 미완료를 거부했다. 최종 좁은
security19.946s/vet12.631s/Staticcheck17.485s도 PASS/source 불변이다.
Actual npm-v2/120.525s는 install47.224s·ci39.884s의 부모87.11s를 PASS했다. 실제 source
분석/현재 근거/cache/Policy·반입과 lock/동일 payload inode 보존을 확인했다. 이어 inspect
30.32s는 operation COMPLETED/M3 ALLOW였지만 기존 gate가 schema v1을 기대해 FAIL했다.
현재 presenter의 v2는 이번 production 변경 전부터 계약이며, 새 analyzer가 schema를
변경한 원인으로 기록하지 않는다. Test expectation 하나만 v2로 교정하고 이미 통과한
install/ci를 반복하지 않고 inspect parent만 후속 검증한다. V2도 Host/source 동일/
helperwait0·자기copy244629616B 제거다. V2 whole FAIL 원본과 성공 부모를 구분해 보존한다.

Npm inspect-only native-v3/34.557s(부모31.49s)는 PASS다. Current content 여섯 ID·
필수 완료/optional no-native 의미, 실제 AST·두 환경 dynamic·advisory·M3 ALLOW·
보존 Evidence를 확인했다. V2 install/ci87.11s를 반복하지 않았다. Source/Host 동일·
noSKIP/helperwait0·자기copy244629579B 제거다. 현재 production 코드는 optional fix 이후
불변이고 V3는 old schema 기대 한 줄만 교정한 gate다.
Source 생성 책임은 기존 bootstrap 파일 한 곳/다섯 actual entry에서 공유하고, fresh
cache의 current content validator는 Cargo/세 npm client가 기존 한 구현을 재사용한다.
Historical ownership-only 원본 확인은 새 승인에 전용하지 않는다. 새 framework/port/
package·별도 updater·버전/권한/한도 변경과 UI 초기화 준비는 없다. 모의 cache fixture
세 소비자는 기존 한 synthetic facts helper를 재사용하며 실제 scanner authority로 쓰지
않는다.
이 production 연결의 IMPLEMENTED/WIRED/QUALIFIED=YES는 Npm 공개 is-number7의
inspect/install/ci·한정 local checks 범위다. ACCEPTANCE_CLOSED=NO다. PNPM/Bun의
실제 새 content 경로, transitive/실행/거부·metadata-specific actual, 다른 OS·hosted와
다른 생태계/큰 native·release 공급/current-only/전체 S04·설계 completion은 MISSING이다.
기존 성공과 새 실패·시험 교정의 원본을 나눠 보존하며 실제 root/local action 계약을
registry Artifact 내용 검사 완료로 바꾸거나 Host script 실행 권한으로 승격하지 않는다.


### 2026-11-04 PNPM/Bun 기존 cache gate의 현재 내용 검사 연결

Baseline c0eea9e: production의 두 client는 이미 공용 newNPMStaticInspector를
사용한다(IMPLEMENTED=YES/WIRED=YES). 기존 실제 peer4/empty cache gate 두 개는
legacy NewInspector(intake)를 직접 구성해 새 current content acceptance를 검증하지
못한다(QUALIFIED=NO/ACCEPTANCE_CLOSED=NO). MISSING은 이 두 caller를 기존 공용
factory로 바꾸고 고정 native tool·현재 scanner/규칙/observer에서 실제 Stage/Open,
readback·변조/취소 거부·정리를 확인하는 것이다. 제품 재구현·새 검사기/수명 owner·
버전/4분 parent 한도 변경은 없다. 전체 CLI/다른 생태계/hosted는 별도로 남는다.

Actual cache-v1은 Bun peer에서 project Policy 거부로154.59s(전체157.632s) FAIL했다.
PNPM/empty는 failfast 미도달이다. Source/Host 동일·helperwait0·자기사본409762431B
제거를 확인했다. 최초 gate는 typed ProjectPolicyFailure의 entry/check/Evidence를
출력하지 않아 구체 원인은 미확정이다. 실패 때 기존 Store의 신뢰된 facts를 로그에
보존하도록 시험 진단만 보완한다. Parser/Policy 완화나 같은 가설 반복으로 해결하지 않는다.

Bun peer diagnostic-v2/156.254s(부모153.24s)는 React18.3.1 source17/scanned17/
parse_errors0/skipped0/fixpoint_timeouts3를 직접 보존했다. 나머지 세 subject는 현재
검사·M3 ALLOW다. React의 required source 미완료에도 기존 npm/PNPM 준비 gate가
첫 Execution만 보아 dynamic을 실행하는 연결 gap도 확인했다. 두 실제 consumer의
공용 준비 판정은 모든 required check의 SUPPORTED/COMPLETED를 확인해야 한다.
Optional no-native를 required로 바꾸지 않는다. 이 좁은 준비 판정은 구현·연결 gap이고
전체 분석기 교체가 아니다.

고정 upstream Core_error/Limits_semgrep에서 fixpoint는 함수별 taint0.2s 내부 제한임을
확인했다. 단순 -timeout 상향은 이 원인을 해결하지 않는다. 기존 rule prefilter를 켠
326개 DATA1.287s는 기대29 match를 유지했지만 malformed Python 오류를 누락했다.
따라서 flag만 바꾸는 방법은 반증됐고 채택하지 않는다. 다음 최소 수정은 기존 고정
source-build patch의 target handler에서 같은 lazy AST를 한 번 파싱하고 기존 skipped/
inserted/tolerated 오류를 독립 합치는 것이다. 그 뒤 고정 rules의 upstream prefilter를
써도 전체 source 파싱을 보존할 수 있다. 새 AST/규칙/지원 언어·시간/권한 상향 없이
동일 source/notice·고정 tool 공급 owner와 generator를 사용한다. 실제 도입은 malformed/
기존326 DATA·React·현재 native/observer·공급 drift가 검증된 뒤 판정한다.

현재 준비 판정의 직접 RED1.013s는 required 미완료/미지원이 첫 COMPLETED에 가려짐을
재현했고, 공용 작은 predicate를 두 실제 소비자가 재사용한 Race2.509s는 PASS다.
기존 static-all/typed layout·optional no-native 의미와 실패한 handoff 초기화는 유지한다.
후보 target handler의 같은 lazy AST 재사용/오류 합침은 동일 compiler의 incremental
35.506s에서171649056B/SHA b095ce48을 만들었다. 기존326+공식 React17=343 DATA
7.193s는 기대29match·오류2·skip0을 보존했고, 별도6언어의 source/sink 없는 정상/
잘못된 문법12개1.873s도 정확한 오류6을 확인했다. CPP의 exact 새 argv/이전 argv
거부·기존138 latch는52.688s PASS다. 모두 한정된 후보 증거이며 actual 새 gVisor
scanner/Bun 정상·PNPM/전체 CLI·hosted와 final source qualification은 남는다.

기존 current updater의 명시 same-engine plan/apply24.447s가 source checkout을
보존한 external candidate에서 정확히 lock/generated/기존 patch3개만 만들었다.
그 결과를 현재 작업 후보에 반영했다. Upstream1.183/tag/commit·SDK/compiler·다른
언어 pin과 updater 경로는 불변이다. 이전 compiled builder-v5에는 semgrep-proof가
없어 첫 plan은 인수 parsing에서 실패했으며 제품 updater 코드는 바꾸지 않았다.
해당 core/source/notice/rules는 아직 canonical clean 재현 빌드와 실제 runtime 검증
대기다. Source profilev3는 full-strict-ast+rule-prefilter 의미와 선택 tool/rules digest를
결합한다. New lock이 runtime qualification/배포 권한을 뜻하지 않는다.


### 2026-11-04 PyPI 전체 census와 archive 소유 한도 — baseline

현재 pypi/sources.go는 기존 MemberCapture의 같은 RECORD/CRC·sdist digest pass에서
source DATA와 작은 native 사실을 이미 수집한다(IMPLEMENTED/WIRED=YES, 범위 한정).
전체 regular member census/0-source·0-native 근거는 NO이며 static 기본 조합/AST/현재
promotion·CLI qualification도 NO다. Wheel의 RECORD/self·서명 sidecar는 capture에서
제외되고, 공용 Inventory가 source parser의10000/512MiB 한도를 전체 archive 한도로
혼용해 기존 PyTorch 정책20k/큰 native를 보존하지 못한다. MISSING: 기존 collector를
모든 regular member의 hash/역할까지 확장하고 기존 archive 검증 완료 후만 census를
공개하며, 명시 archive-owner bounds를 쓰고 source parser 한도는 따로 유지한다.
기존 public DATA 메서드/typed wheel·sdist 결과/graph charge는 재사용한다.
이번 부분은 constructor/production 활성화가 아니며 .pth 등 Python 실행 역할·큰 native
streaming/기본 source port·승격/실제 CLI는 다음 필수 gap으로 남긴다. 전역 budget 상향·
새 archive parser·임의 압축 해제나 body 전체 복사로 해결하지 않는다.

PyPI 같은 수정 경로의 추가 accounting gap: static.go가 graph uncompressed/files를
정규화 RECORD 목록에서 합산해 self RECORD의 실제 bytes와 제외된 서명 sidecar/ZIP
directory entry를 세지 않는다. Archive parser는 전체 ZIP 크기/목록을 이미 한 번
검증한다. 이 기존 측정값을 typed WheelInspection에 전달하고 전체 regular CRC를
검증해 graph 한도에 적용해야 한다. 새 추출/통계 parser·한도 상향 없이 작은 실제 ZIP
잔여 예산 회귀로 확인한다. Census 추가로 native/source 안전성을 자동 부여하지 않는다.

Canonical clean 재현은1585.681s에 성공해 incremental 후보와 exact b095ce48/171649056B가
일치했다. 기존 source1028·compiler/SDK/worker·원본 Host manifest를 보존했고 cleanup은
성공이다. Updated tuple의 실제 materialize10.149s도 root0/single-link/555·444 네 공급물을
확인했다. 이 build/materialize 자체의 runtime_qualified=false를 수정하지 않는다.

Actual cache-v3/96.402s는 source-all 뒤 동적 준비에서 FAIL했다. 작업자가 responder
빌드의 CGO_ENABLED=0을 빠뜨려4063495B/INTERP/CGO1을 만든 준비 오류였다. Product
validResponderImage를 완화하지 않는다. CGO0/strip canonical 설정과 실제 buildinfo/ELF를
확인하고 기존 CheckResponderSource를 두 gate의 public selection 이전에 호출한다.
V4는 마지막 test-only 수정 전 helper receipt 때문에 엄격한 source preflight가 Host
실행 전에 거부했다. 최종 frozen 입력으로 cached helper build0.592s/0.339s만 다시
기록했고 현재 client build6.354s는 재사용했다. 원래 실패를 성공으로 바꾸지 않는다.

Actual cache-v5/304.285s는 Bun158.55s(peer151.19/empty5.89)와 PNPM142.48s
(peer135.40/empty5.61)를 모두 PASS했다. Current source-profilev3/전체 censusv2와
필수 coverage·M3/M4 ALLOW·fresh Stage/Open·sealed records/native peer provenance·
private 준비/readback/close·변조/취소 거부를 확인했다. Host/source 동일/noSKIP/
helperwait0·자기사본409771057B 제거다. 이 source/runtime의 peer4와 empty0 cache
범위는 IMPLEMENTED/WIRED/QUALIFIED=YES, ACCEPTANCE_CLOSED=NO다. Full CLI/
실행·metadata-specific actual/다른 OS·hosted는 별도다.

PyPI census의 표적8.880s와 content/pypi/artifact-pypi/Cargo 전체 Race21.730s는
PASS다. 이후 같은 owner의 graph accounting 직접 RED1.288s가 self RECORD bytes
누락을 확인했다. 기존 archive parser의 전체 ZIP bytes/entry 측정값 두 개를 typed
WheelInspection에 전달해 graph budget에 사용하고, self/서명 CRC도 항상 검증한다.
정규화 RECORD Files·서명 미제공 허용과 graph 한도는 유지한다. 달라진 wheel 검증은
static profilev2로 구분하며 sdist profile은 유지한다. 해당 정상/예산 경계/세 CRC 거부·
기존 source/native 단일 charge 표적 Race11.331s가 PASS다. 새 extraction/함수 계층·
시간/공간 상향은 없고 source ctor/default/cache/promotion·큰 native는 아직 MISSING이다.

Completed own prototype/test copies5개는110process의 exe/FD/maps 참조0·queryerror0
확인 뒤629393528B를 제거했다. New supply/canonical originals/SDK·현재 native v5
입력과 원본 로그는 보존했다. No CI/push/설치/배포/이름 변경은 없다.

현재 의미 있는 source 묶음의 최종 frozen local checks는 foundation83.741s, security
34.789s, 영향8개 package 전체 Race316.140s, vet11.557s, 고정 Staticcheck25.478s가
모두 PASS/source unchanged다. Race는 Artifact PyPI/content/npm/PyPI/sandbox/bootstrap/
promotion/updater 기존 소비자를 포함한다. Source-byte 수집·typed archive metrics·현재
프로필/기존 updater·읽기 전용 공급 경계를 재검토했으며 새 public framework/package·
버전 owner나 native 권한 상향은 없다. 기존 successful CPP/native/canonical build를
반복하지 않는다. 이 checkpoint 뒤에도 CLI/실행과 PyPI 실제 source/승격·큰 native 및
나머지 보강 설계의 MISSING을 계속한다. 전체 기능 완료나 배포 준비 판정이 아니다.


### 2026-11-04 PNPM/Bun 현재 CLI 검증 baseline

Checkpoint ec1990d의 current content cache는 peer4/empty0를 실제로 검증했다. 제품의
CLI composition/기존 install·frozen·add·update·adopt는 IMPLEMENTED/WIRED=YES지만
현재 source-profilev3의 CLI QUALIFIED/ACCEPTANCE_CLOSED=NO다. 기존 CLI gate는
check ID 없이 required 상태만 보므로 새 검사 누락 여부를 확인하지 못한다. 같은
두 gate에서 현재 여섯 content ID/required 선언을 확인하고 명령 소요 시간을 기록한다.
코드/새 검사기·새 test framework나 production budget은 변경하지 않는다. 먼저 기존
4분 parent에서 single의5개 실제 operation·원본 inode/controls·거부를 한정 검증한다.
Peer cache의151.19/135.40s를 full CLI batch 시간으로 추정해 상향하거나 재검증 결과로
전용하지 않는다. Peer/empty CLI와 execution/negative flow는 계속 필수 MISSING이다.

CLI-single-v1은 Bun의 install50.010/frozen46.895/add48.926/update46.852s가 완료됐으나
마지막 adopt에서 전체5개 명령이 공유한4분 context가 만료되어243.782s(부모240.67)
FAIL했다. Raw는 OPERATION_TIMED_OUT·후속 intake-accounting 메시지를 구분한다.
Host/source 동일/helperwait0·자기copy409537761B 제거이며 PNPM은 failfast 미도달이다.
이제 같은 gate의 독립 CLI 호출마다 기존4분 deadline을 적용하는 작은 local run closure를
세 실제 호출 지점이 재사용한다. Production timeout/권한/검사 요구와 검증 case는 그대로다.
Go test의 전체 harness 제한은 별도이며 현재 single 두 parent10분 내로 확인한다.

CLI-single-v2/463.224s는 Bun238.89s와 PNPM221.09s 모두 PASS다. 각각5개의
fresh operation/current content6·required flags·Policy/records/반입, frozen lock/
원본 inode·사용자 파일 보존, implicit enrollment/unproven adoption 거부를 확인했다.
Host/source 동일/noSKIP/helperwait0·자기사본409541732B 제거다. 개별 명령은
Bun46.138–49.208s, PNPM43.177–44.206s이며 실제 단일 명령의4분 제한은 유지했다.
전체 single 흐름의 합계와 command timeout을 구분한다. 이 범위의 구현/연결/한정
qualification은 YES이며 전체 CLI acceptance는 peer/empty·실행/hosted가 남아 NO다.
이미 성공한 single을 반복하지 않고 같은 compiled source의 peer/empty만 이어간다.
두 client의 peer 두 번을 한10분 harness에 모으면 현재 cache 실측135–151s/회와
CLI 추가 비용상 한계에 걸릴 수 있으므로 client별 기존10분 harness에서 순차 검증한다.
Production/개별 command 한도·case 수와 필수검사는 그대로다.

Bun peer/empty-v3는324.905s에 PASS했다. Peer 두 명령152.345/152.863s와 empty
두 명령7.484/7.378s가 current content6·fresh Policy/records·frozen lock/원본 파일
보존·무단 인계 거부를 확인했다. Host/source 동일/noSKIP/helperwait0이며 자기사본
409541710B를 제거했다. 같은 빌드의 PNPM peer/empty만 이어서 검증한다.

### PyPI 설치 역할과 source census — 다음 누락의 수정 지도 (`ec1990d`)

기존 `StaticInspector.InspectWheelContent`/`sourceCapture`는 전체 regular member를
인증하지만 source 선택은 suffix/shebang만 본다. `WheelInspection.Surface`의 기존
`InstalledFiles`/`SiteHookLines`가 아는 active `.pth` import와 interpreter를 알 수 없는
installed script가 현재 내용 역할에서 빠진다. 부품 IMPLEMENTED/WIRED=YES, 이 역할
연결의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=NO다. 전체 S04는 계속 OPEN이다.

기존 typed installed-role 결과로 누락 target을 고르고, 필요할 때만 기존 authenticated
wheel reader를 한 번 더 사용한다. 기존 source body/graph charge를 중복하지 않고
추가 body도 최초 census의 크기/해시와 대조한다. Active import가 있는 `.pth`는 원문
전체를 Python source DATA로 보존하고 path-only/nested `.pth`는 자동 실행 source로
승격하지 않는다. Path/import 혼합 원문의 parsing 한계는 숨기지 않고 current analyzer의
미완료 규칙을 유지한다. Interpreter 불명 installed script/unresolved 역할은 unsupported로
남기며 native와 알려진 interpreter 역할을 덮지 않는다. Source/target 합산 한도는 그대로다.

수정은 기존 `inspection/pypi/sources.go`와 해당 회귀에 한정한다. Active/mapped/path-only/
nested hook·알려진/불명 script·원문 body/해시·단일 graph charge·재읽기 변조/취소 거부를
먼저 표적 검증하고 의미 있는 content 묶음에서 canonical 검사를 모은다. Default analyzer/
graph/cache/promotion/실제 PyPI와 mixed `.pth` 호환 qualification은 후속 MISSING이다.
진행 중 PNPM native가 소비하는 Root/private source는 바꾸지 않고 별도 임시 사본에서
구현하며 해당 native 종료 후 기준 bytes를 대조해 정확한 수정만 반영한다. 새 CI는 없다.

PNPM peer/empty-v4도319.550s에 PASS했다. Peer142.262/156.630s·empty8.652/7.372s,
current content6/Policy/원본 보존·거부·Host/source 동일/noSKIP/helperwait0와 사본
409541712B 제거를 확인했다. Single-v2와 합친 각 client의 전체 CLI 실측은5분을
넘고10분 안이다. 두 기존 shell 명령과 그 canonical validator의 suite deadline만
5→10분으로 맞췄으며 개별 command4분과 제품 예산/전체 case는 유지한다.

Wheel 역할 표적 RED9.515s는 source2/expected4 누락을 재현했다. 기존 reader/capture
수정의 GREEN1.229s·표적 Race15.163s는 active/mapped hook 원문·해시/known script/
unknown script/native 구분·단일 graph charge와 archive/member 변조·부재·취소·기존
합산 bytes/target 한도 거부를 확인했다. 첫 편집 명령의 치환 assertion이 source 저장 전에
실패해 green-v1 이름의 실행도 old-source RED0.765s였다. 원본을 지우거나 성공으로
해석하지 않는다. 실제 최초 GREEN은 v2다. Mixed `.pth` parsing 호환은 미검증이다.

최종8개 source 경로는 private 사본에서 canonical format 후 foundation52.120s,
security36.219s, PyPI/bootstrap/checker 전체 Race169.490s, vet13.706s, 고정 Staticcheck
21.495s 모두 PASS/source unchanged다. Checker의 실제 누락/중복/명령 변조 거부도 기존
전체 회귀에 포함된다. 종료한 native 이후 Root/private source의 원래 bytes를 확인해
동일8경로만 반영했고 reviewed-file-map-v1이 해시를 연결한다. Existing CPP/canonical
source build/cache/single native 성공은 반복하지 않았다. 아직 PyPI analyzer 구성과
graph/cache/실제 승격, npm계열 execution/metadata/hosted·전체 S04가 남아 있다.

### PyPI configured 내용 검사와 기존 graph — 착수 지도 (`319331c`)

기존 StaticInspector의 typed content/census 부품은 로컬 연결됐지만 analyzer ctor와
실제 Inspect/InspectWheel·ExpectedChecks에는 연결되지 않았다. Composite의 단일/graph/
prerequisite 준비는 첫 Execution만 보며, graph static 오류 시 이미 확보한 reports를
nil로 버린다. Sdist diagnostic의 primitive 재생성도 새 composite checks를 잃을 수 있다.
따라서 이번 좁은 연결은 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=NO다.

기존 StaticInspector ctor의 optional analyzer와 content.report에 같은 SourceAnalysis/
SourceChecks/BinaryChecks/InventoryChecks를 연결한다. Unconfigured typed reader와
builder의 recipe reader는 유지한다. Sdist diagnostic은 primitive static 결과일 때 붙인
다음 기존 composite로 합쳐 child checks를 보존한다. Composite/prerequisite의 기존
준비 predicate를 전체 required supported/completed로 바꾸고, 최초 static/graph 오류에도
확보된 report와 원래 cause를 반환한다. Core 타입/새 analyzer·resolver·권한은 만들지 않는다.

수정은 `inspection/pypi/{static,sources,composite,prerequisite}.go`와 같은 package의
회귀에 한정한다. 기존 parser 포트의 입력/실패·선언 일치, static finding에서 analyzer
미호출, source 미완료에서 dynamic 미호출, 원래 cause/partial checks·sdist diagnostic/
metadata-only source 부재를 표적으로 확인한다. Factory/default/cache/promotion/큰 native/
실제 PyPI·mixed pth는 다음 acceptance로 남긴다. 새 CI/Host 변경은 없다.

구성 연결 표적1.493s와 확장 Race3.146s는 analyzer Port 입력/여섯 선언·실제 child
일치, 오류 시 typed wheel/static·graph facts와 첫 cause 보존, malformed archive에서
parser 미호출, partial parse에서 dynamic 미호출, sdist diagnostic+native Finding과
child6 유지 등을 확인했다. Prerequisite parser 실패도 현재 requested graph의 검사
결과를 보존하면서 errors.Is로 원래 cause를 전달한다. 새 Core API/검사기는 없다.

Frozen source의 foundation22.854s/security16.640s/vet0.582s/pinned Static1.885s는
PASS다. 전체 PyPI/bootstrap/application Race는 각2.559/1.969/1.758s PASS이며,
같은 batch216.366s의 promotion은 기존 socket fixture가 sandbox setsockopt 권한에서
미도달했다. 같은 입력의 socket 사례만 필요한 권한으로 Race3.458s/1.073s에 PASS했다.
성공한 나머지 전체 사례를 반복하거나 최초 batch를 전부 성공으로 고쳐 쓰지 않는다.
Configured core의 한정 구현/연결은 YES지만 default/native/cache/whole acceptance는 NO다.

동일 Go 입력/CGO0의 current tagged client6.716s·policy0.538s·stripped responder0.375s를
준비하고 기존138 observer/strict parser 공급으로 실제 npm execution CLI-v3를 확인했다.
첫 명령37.09s/전체40.261s FAIL이고 일반 오류 뒤 intake-accounting 후속 메시지만
남아 primary 실행 report는 미확보다. Refusal은 failfast 미도달이다. Host/source 동일/
helperwait0·자기사본409566317B 정리를 확인했다. PyPI opt-in 연결이 원인이라고 단정하지
않는다. 다음은 기존 gate가 이미 받은 bounded machine report를 실패 로그에 남겨 첫
실패를 확인하는 최소 진단이다. 성공한 install/cache/CLI나 전체 CPP를 재실행하지 않는다.

CLI-v4는 제품 변경 없이 bounded 실패 report 진단만 추가한 동일 입력에서41.172s
FAIL했다. 현재 v2 machine report는 exact TypeScript digest10e108c9의 integrity/static/
binary/census 완료, source-content INCOMPLETE/source signals NOT_EXECUTED를 보존하며
quarantine/preparation 미완료·program NOT_STARTED다. Runtime 실패와 output decode 오류는
아직 미구분이다. Host/source 동일/helperwait0·자기사본409569029B 제거를 확인했다.
기존 Cargo parser diagnostic을 실제 두 caller가 공유하도록 이름을 정리하고, runtime
상태/limitation/오류와 언어·rule ID 수만 남기는 narrow Port 진단에 같은 public TypeScript
hash의 source-only gate를 연결한다. Source body/메타변수는 로그에 남기지 않는다.
기존 core/규칙/CPP/제품 예산을 바꾸거나 CLI 전체·성공한 소비자를 반복하지 않는다.

진단 v5는 작성한 시험이 Resolver에 없는 Source() 메서드를 가정해 compile0.891s에
실패했고 native는 실행하지 않았다. 실제 source ID 생성 방식으로 교정했다. V6는 별도
전체 npm metadata discovery가 기존2MiB 한도에서0.13s/operator3.198s 미도달했다.
그 한도를 늘리지 않고 기존 workflow의 exact TypeScript URL/integrity fixture를 두
실제 caller가 재사용하도록 했다. 원래 CLI resolve/전체 metadata 지원 완료와 구분한다.

Source-only v7은32.97s/operator36.060s에 같은 archive hash/113 targets에서 FAIL이다.
Port의 실제 runtime은 S04_SOURCE_EXECUTION_INCOMPLETE이고 최초 CPU authorization
exhausted/cancel/typed termination 미완료 뒤 resource accounting 오류를 보존했다.
현재 SourceAnalyzer는 PublicPyPI의30초 누적 CPU 정책을 재사용한다. Decoder나 기존
React taint0.2s fixpoint 문제가 이번 원인이라고 단정하지 않는다. Host/source 동일/
helperwait0·자기사본409581855B 정리다. 다음은 같은 고정 engine/rules/strict flags와
512MiB memory,113개의 source DATA만 network-none/readonly runc에서 비용 측정한다.
DATA의 유한120초 CPU 상한은 제품 승인/지원 예산 변경이나 gVisor qualification이 아니다.
Python 공용 예산/필수 검사·입력/출력 완전성을 약화하지 않고 측정 뒤 수정 범위를 정한다.

종료한 superseded client binary4개는110process의 exe/FD/maps 참조0·query errors0과
원래 compile receipt/현재 hash·single-link를 대조해228361201B 제거했다. Current v7,
공용 helpers/고정 supply/SDK/source와 모든 raw를 보존했다. 새 CI는 없다.

동일113 source의 runc DATA 측정은14.327s에 종료했다. Core wall12.541s, user/system
CPU10.478/2.019s이며 source113개·matches0·fixpoint0·skip0다. 다만 lib/_tsc.js6213092B와
lib/typescript.js9112572B는 기존 per-target256MiB 한도로 Out of memory, 공식
lib.es2015.symbol.wellknown.d.ts의 `[K in keyof readonly any[]]`는 PartialParsing이다.
원래 source19,098,939B의 parser limits/grammar 문제와 traced 누적 CPU 소진을 구분한다.
컨테이너 OOMKilled=false·원본 input/기존 container 집합 동일·자기 container 제거다.
이 DATA는 정상 검사 PASS가 아니며 세 오류를 N/A로 바꾸지 않는다. 현재 원래 코드의
TypeScript grammar index_type_query는 primary_type만 받고 readonly_type은 상위 type에
있음을 확인했다. 기존 parser/tool 공급 owner에서 한정 grammar·비용 보완을 검토하고,
필요한 변경을 모아 공급 재현/실제 검증을 진행한다. 아직 제품 예산/도구는 불변이다.

메모리 DATA512MiB/1024MiB는 같은 두 Out-of-memory와 grammar 오류를 유지했고 각각
CPU16.617/26.182s였다. 2048MiB도 유한120초 DATA deadline에서 미완료여서 상향만으로
해결됐다고 볼 수 없다. 제품 예산은 계속 불변이다. 큰 두 JS에는 현재 rule sink의
fetch/axios/eval/execSync가 없는데, 기존 patch는 rule prefilter보다 먼저 generic AST와
이름 해석을 강제로 만든다. 기존 lazy parser가 실제 사용됐는지 기록하고, rule matching
후 미사용 target만 같은 기존 parser의 syntax AST/오류를 확인하는 한정 후보를 검토한다.
Parse_target의 기존 run/converter를 재사용하며 별도 언어별 parser/규칙/skip 경로를 만들지
않는다. 모든 skipped/inserted/tolerated 오류는 유지하고 applicable rule의 AST/이름 해석은
그대로다. 후보는 먼저 /tmp SDK에서 비교하며 원래 source는 빌드 뒤 복원한다.

Grammar 원형은 기존 fixed SDK/generator와 upstream registry의 CLI0.26.3/ABI14에서
원래 TS parser.c/CST/Parse와 exact byte 일치했다(13.246s). 같은 readonly_type 재사용
후보는 한정 생성이 가능했지만 generated C table의 약284k line diff를 그대로 repository에
넣지 않는다. 기존 Semgrep wrapper의 작은 override도 같은 candidate grammar JSON과
byte 일치함을 확인했다. 필요하면 기존 source input manifest에 고정 CLI를 포함하고 기존
worker에서 TS/TSX를 생성하는 경로를 재사용한다. 별도 updater/Host 설치·Node dependency는
추가하지 않으며 전 후보 qualification 전까지 selected owner/도구/제품은 바꾸지 않는다.

후보의 실제 비교는 해결로 판정하지 않는다. 첫 syntax 후보0cf38573은 full parser를
per-rule deadline 안으로 옮겨 lib.dom의 Timeout2개를 더 만들었고 DATA v2의 오류는5개다.
따라서 그 순서는 반증됐다. 같은 prefilter 뒤 parsing을 원래 deadline 밖에 두고,
미사용 JS/TS에만 기존 native syntax tree/error extractor를 쓰는 후보54558010의 빌드는
79.958s에 완료·원본6파일 복원·compiler/container 보존을 확인했다. 하지만 DATA v3의
113source/13.314s/CPU13.216s도 기존 두 Out-of-memory와 readonly PartialParsing을 유지한다.
큰 두 JS는 실제 applicable search2개(typescript.js에는 taint group1개도 있음)여서
generic AST 생성을 피하지 못했다. 이 최적화는 현재 실패의 해결책으로 채택하지 않는다.
SKIP/fixpoint0·matches0을 정상 검사 통과로 확대하지 않는다.

다음 좁은 진단은 원래 selected parser의 Parse_target 한 파일에만 단계별 CPU/heap 수치를
기록하고 원문/이름/값을 출력하지 않는 임시 빌드다. 파싱/이름 해석/타이핑/상수 분석의
실제 최초 비용을 구분한 뒤 수정 범위를 정한다. 원래 의미·규칙·필수 검사/제품 한도를
유지하고 종료 시 source를 복원한다. 또 다른 무근거 메모리 상향·전체 native/CI는 하지 않는다.

기존 parser 단계 진단34926dbf는78.265s에 빌드·원본 한 파일 복원했다. 첫 DATA는
semgrep.parsing 로그 source가 기본 비활성화라 단계 측정을 얻지 못했다. 기존 로그
source만 활성화한 후속 DATA-v2는 같은 `_tsc.js` 한 파일을43.316s/CPU43.296s,
최대RSS1,421,740KiB에 완료했다. 단계별 CPU는 parse17.096s·naming10.329s·typing/
implicit0.250s·basic constant propagation8.204s·dataflow6.182s다. 오류/skip/fixpoint0이며
기존 의미를 유지했다. 따라서 전체 파싱 비용 자체가 PublicPyPI30초/512MiB를 넘는다는
직접 증거다. 어느 분석을 생략하거나 이름 해석만 버그라고 단정하지 않는다. 새 tool
후보/전역 Python 한도 변경 대신 SourceAnalyzer의 현재 별도 profile과 공통 관찰 원장,
고정 command/CPP 검증 owner 안에서 실측에 맞는 유한 source 전용 예산을 검토한다.
아직 예산을 제품에 적용하지 않았으며 전체113source의 비용/정확성 확인이 선행이다.

TSX도 원형 parser/CST/Parse의 exact byte 일치를15.613s에 확인했다. 최초 생성의
출력 UID 누락은0.481s 미도달 실패로 보존했고 임시 출력 소유자만 맞췄다. 같은 wrapper
변경으로 TS/TSX 후보를 생성했으며, 첫 core compile은 기존 수작업 merged CST/출력기
결합 누락을 검출했다. 해당 두 owner도 같은 readonly tuple로 연결하는 후보를 빌드 중이다.
큰 generated 파일을 제품에 넣거나 기존 버전 갱신 owner를 우회하지 않는다. 임시
진단/실패한 syntax 최적화는 이 grammar 후보에 포함하지 않는다.

자원 보완의 기존 연결 지도: `SourceAnalyzer.Analyze`는 현재 Python policy 변환과
privileged resources.Create의 `pypi`를 함께 사용한다. Docker argv는
`sourceAnalysisCreateArguments → pythonResolverCreateArguments`의512MiB/CPU60초를
재사용하고, helper `hosttool.observationPolicy`도 같은30초 누적 권한을 부여한다.
따라서 controller 값만 바꾸면 helper가 먼저 차단하며 해결되지 않는다. 실측 후 source
전용 유한 CPU/메모리 값을 기존 SourceAnalysis 계약 한 곳에서 소유하고 기존 원장/
helper selector/argv·CPP exact 검증을 함께 연결하는 범위만 검토한다. PublicPyPI/PyTorch/
동적 실행 정책과 검사 완료 판정·부분 오류 거부·한도 초과 차단은 그대로 유지한다.
다음 검증은 grammar 후보의 동일113source DATA, source 전용 경계/원래 profile 불변의
표적 회귀, 변경된 공급·CPP 묶음과 실제 source/native 순서다. 같은 성공 입력의 unrelated
full/CI 반복은 없다. 새 resource service·updater·generic configuration 계층은 필요 없다.

Grammar-only3b83e890의113source DATA-v2는129.891s/CPU128.531s/RSS2,146,176KiB였다.
기존 OOM2와 readonly 구문 오류는 사라졌지만 typescript.js의 credential-body rule
Timeout1/내부 fixpoint3이 남아 정상 검사 통과가 아니다. 기존0.2초를2초로 바꾼 임시
후보fa0af022도 같은 파일 하나96.631s에 Timeout1/fixpoint1을 유지해 해결로 채택하지 않는다.
이 후보의 Limits 수정과 임시 단계 진단은 제품 공급에 넣지 않는다.

관련 owner를 좁혀 보니 `OSS_match_tainting_mode.check_rule`은 기존
`Match_taint_spec.taint_config_of_rule`로 모든 source/sink AST 일치를 확보한 뒤에도
matched sink0인 rule의 모든 함수·class·top-level CFG를 분석한다. CE의 결과는 ToSink
효과이므로 matched sink 부재가 확인되면 전파는 불필요하다. 그 raw_sinks 목록만 조건으로
사용하고 원래 패턴 검사/parse errors/보고 profiling을 보존하는 최소 후보를 비교한다.
새 parser·문자열 기반 sink 추정·taint engine 재작성·기존 시간 제한 확대는 하지 않는다.
실제 sink가 있으면 원래 분석을 그대로 수행한다. 표적 actual과 기존 양성/악성·malformed
corpus 대조 전까지 성능/전체 S04 해결을 주장하지 않는다.

배포 입력은 더 단순한 기존 source offer를 우선한다. 원래 upstream도 generated C/CST를
source로 공급하므로 재현을 확인한 TS/TSX generated8개를 같은 외부 SOURCES manifest의
passive 입력으로 묶고 worker에서 투영하면 된다. 작은 wrapper/merged CST/converter
수정은 기존 quarantine.patch가 소유한다. 새 CLI binary를 제품/SDK에 추가하거나 새로운
grammar 생성 subsystem을 만들 필요가 없다. 고정 CLI/generator의 exact 원형 재현과
생성 출처는 기존 SOURCE-ORIGINS와 원본 증거에 연결한다. 큰 generated C는 Git에 넣지 않는다.

검증/비교가 끝난 임시 core 사본3개는 build receipt/hash·118process 참조0/query errors0을
대조해514,952,928B 제거했다. Selected core/current grammar 후보/SDK와 모든 raw는 유지한다.

### TypeScript source 최초 실패 — 최소 후보·별도 자원 경계 (`5026030` 후속)

최소 sink 후보f2da5078은 typescript.js의 실제 AST source0/sink0을 확인하고71.308s에
errors/fixpoint/skip0으로 완료했다. 최종33ae4500은 임시 진단을 제거하고 빈 결과의
기존 matches_hook도 유지하며 내부0.2초 제한을 바꾸지 않는다. 기존343source의
29matches/parse errors2·누락/추가0(7.222s), 여섯 언어와 readonly TS/TSX의16source/
expected malformed8(0.570s), 원래113source의 errors0/fixpoint0/skip0·CPU115.631s/
wall115.736s/RSS2,100,748KiB를 확인했다. 이는 Runc DATA이며 아직 gVisor/전체 S04 완료가 아니다.

Source 전용 경계는 기존 Ports 계약의 한 상수 집합으로 소유한다. 전체 artifact의 누적
CPU600초(기존 reserve 차감), wall15분, runtime4GiB·per-target heap3072MiB·rule15초와
launch4개다. Controller·기존 privileged helper·Docker argv·check profile-v4를 같은 값에
연결했다. Source input/trace/output/권한·부분 오류 거부는 유지한다. PublicPyPI와 원래
동적 profile은 불변이며 source 전용 이름 외 caller-selected 한도는 받지 않는다.
같은 profile/원래 Python 권한 보존 RED10.066s 후 경계·원래 PyTorch·argv 회귀13.009s는 PASS다.

기존 CPP source 명령의 두 값만15/3072로 맞추고 old2/256도 거부한다. 다른 role·mount·
program guard를 유지한 full138/53.005s가 PASS다. Source offer는1036개/328,022,528B이며
원래1028개 중 patch/worker/출처/설명4개만 갱신하고 passive generated8개를 추가했다.
NOTICES/rules/기존 dependency·SDK·compiler는 불변이다. 기존 same-version plan/apply는
25.138s에 정확히4개 candidate 출력(lock/generated/patch/worker)을 만들었고 원래 checkout은
변경하지 않았다. 검토 후 그 출력4개만 반영했으며 Semgrep 외 모든 runtime owner가 동일하다.

Canonical foundation98.043s는 format/drift/integrity/build/architecture/docs를 PASS했다.
Affected Race60.019s에서 hosttool4.608s/content2.061s/updater35.115s는 PASS이나 sandbox의
Unix socket bind/보호 runtime 조회는 도구 제한 환경에서 미도달했다. 같은 source의 실패한
45개 parent만 필요한 권한으로 Race7.872s(실제 sandbox6.565s)에 PASS했다. 최초 batch를
전부 성공으로 덮거나 통과한 전체 tests를 반복하지 않는다. 현재 native client19.527s/
policy3.418s/CGO0 stripped responder0.510s를 고정했고 기존 공급 검증기12.812s로 root-owned
별도 supply를 확인했다(runtime_qualified=false). 같은113source의 actual gVisor 검증은
진행 중이며 full CLI/refusal/cache와 fresh canonical build, PyPI·다른 소비자는 계속 OPEN이다.

Actual source-only v8는622.94s/operator626.667s에 누적 CPU600초 소진으로 FAIL했다.
기존 integrity/static/binary/census는 완료지만 source는 INCOMPLETE이며 required 거부를
유지했다. Runtime kill/cancel·termination/accounting 후속 오류도 첫 원인과 함께 보존한다.
Host/source 동일·helperwait0·자기 사본409,580,383B 제거를 확인했다. 따라서 현재 source
전용 예산이나 Runc 성공을 actual 호환 해결로 권장하지 않는다. Source/core/CPP는 고정한
채 동일한 전체 native를 반복하거나 CPU를 또 상향하지 않는다. Fresh canonical build와
전체 CLI/CI는 보류하고 같은 큰 파일의 gVisor 자체/관찰 결합 비용을 분리하는 bounded
DATA를 우선한다. Product의 필수 관찰·격리·권한은 변경하지 않는다.

독립 OCI 진단은 parser 이전의 Rootfs mount/시스템 호출 thread 준비에서 실패해 비용
비교 근거가 아니다. 임시 자원은 회수했으며 그 경로의 추가 조정은 중단했다. 기존 실제
경로를 그대로 재사용한 임시 Go overlay는 reader ID만 기록하고 test context를180초로
한정했다. Actual DATA-v11은10/35/70/120초의 read-only stack/ps4개를 확보하고185.004s에
예정된 취소·Host/source 동일·helperwait0·409,580,811B 정리를 확인했다. 앞선 수집기의
`/var/run` alias 거부도 원본으로 남겼고 realpath만 맞췄다. 이는 기능 qualification이 아니다.

Samples의 실행자는 verifier가 아니라 semgrep 본체(4threads/CPU94–96%)였고 kernel
frames는 Futex/Pause/Ppoll이며 Getrusage/ClockGettime은 없었다. `ps.time`은 SDK 구현상
SysTime만 표시하므로20–50ms를 전체 CPU로 해석하지 않는다. 기존 execution owner의
verify도 실행 전후 각 한 번이며 반복 검증을 지연 원인이라고 쓰지 않는다.

동일33ae4500/_tsc.js의 runc에 실제와 같은 단일 물리 CPU를 고정해도43.530s에 완료해
affinity 차이만으로 원인을 설명할 수 없다. 같은 고정 조건에서 엔진이 이미 제공하는
`-use_parmap` 한 옵션은34.469s/errors·fixpoint·skip0, 구문16/expected malformed8은
0.591s에 PASS다. 기존 검사/strict AST를 유지한 Process 실행의 실제 호환을 다음 후보로
비교한다. Source command/profile·기존 CPP exact guard만 함께 맞추며 새 scheduler/engine/
인터페이스·언어별 버전 owner나 CPU 상향은 만들지 않는다. Actual 성공 전에는 Eio가
확정 원인이라고 단정하거나 Process 경로를 해결법으로 권장하지 않는다.

같은343 corpus의 Process 비교는6.651s/29matches/expected errors2/누락·추가0으로
PASS했다. Fixed argv/profile-v5와 old Eio/timeout/heap 거부를 연결한 Go 표적13.555s,
기존 full CPP138/51.094s, canonical foundation71.036s/security29.271s도 PASS다.
현재 세 binary receipt를 갱신한 actual-v12는 원래 TypeScript5.9.3/113source를
175.69s(operator178.980s)에 필수 content6 완료로 통과했다. Optional no-native limitation은
원래대로 보존하며 source 오류·findings는0이다. Host/source 동일/helperwait0/noSKIP과
자기 사본409,822,080B 삭제를 확인했다. 이는 같은 engine/core/입력/한도에서 기존
Process 옵션의 actual 호환을 확인한 결과다. Eio 내부의 정확한 지연 원인은 미확정이며
모든 엔진/생태계 호환이나 탐지율로 일반화하지 않는다. 원본은
`s04-parmap-coverage-v4`, `s04-source-analysis-observer-parmap-v11`,
`s04-parmap-command-target-v1`, `s04-parmap-final-*`,
`s04-npm-parmap-content-native-v12`가 소유한다. Fresh canonical 재현 빌드를 시작했으며
full CLI/refusal/cache와 나머지 S04는 계속 OPEN이다.

### PyPI 기본 연결과 현재 승인 — 다음 acceptance gap (`5026030` 후속)

Configured `StaticInspector`/`CompositeInspector`의 내용 분석은 IMPLEMENTED/WIRED=YES,
default 생성 세 곳의 연결과 current 승인 acceptance는 각각 NO/NO/NO/NO다.
`bootstrap.go`의 pip/UV/pipx 설치, `uv_project_client.go`의 프로젝트,
`python_uv_run_cli.go`의 UV/pipx 실행이 아직 analyzer 없이 같은 constructor를 호출한다.
기존 `python_client.go`의 공통 조립에서 target pin과 source analyzer를 한 번 소유하고
세 caller만 연결한다. Linux는 기존 protected executor/observer/resource adapter를 쓰며
다른 플랫폼의 unavailable은 기존 npm처럼 명시한다. 별도 scanner/graph owner를 만들지 않는다.

`PythonProjectCache.approval`은 현재 declared facts를 읽지만 content의 current profile/kind를
요구하지 않는다. 이미 Cargo/npm이 쓰는 `projectCacheEvidenceFacts`와
`verifyArchiveContentEvidence`를 재사용해 Stage/readback 모두 같은 기준으로 확인한다.
기존 Python 승격/실행 소비자의 기대 check 선언과 Evidence 전달도 실제 사용 위치에서
대조한다. Legacy·wrong profile/kind·missing coverage 거부와 기존 정상 selection을 먼저
표적으로 검증한 뒤 실제 pip/UV 설치·업데이트/실행/프로젝트를 묶는다. Mixed pth/큰 native,
다른 생태계·release 및 grouped hosted는 별도 MISSING이다. Native 빌드 중의 Go 작업은
동결된 외부1036개 source/build 입력과 독립이며 lock/patch/worker/observer는 바꾸지 않는다.

기존 UV sync fixture의 self-consistent legacy/missing-source/wrong-profile/wrong-kind 네
승인 수용을 RED9.728s에 재현했다. 공통 facts/current content 검증기를 Python approval에
연결한 표적13.531s와 Python/UV·관련 pypi/bootstrap/promotion Race63.408s는 PASS다.
같은 approval을 Stage/readback에서 재사용하며 정상 nonempty/empty와 거부·원복을 보존한다.
세 production 생성 지점은 같은 Python 조립 helper에서 기존 target pin과 analyzer를
받는다. Source/runtime version owner는 불변이다. 실제 Python/전체 보강 완료는 아니다.

### npm 실행의 목록 readback — 중복 분석 gap (`5026030` 후속)

`Inspector.InspectArchive`의 current 내용 분석 연결은 IMPLEMENTED/WIRED=YES이며
project inspection의 최초 검역에 필요하다. 그러나 bootstrap의 실행 entry 선택·Node 배치·
PNPM/Bun layout과 original-layout 증명 여섯 caller도 같은 메서드를 사용한다. 이들은
목록/원본 해시만 소비하면서 같은 analyzer를 다시 실행한다. Current source 검역을
약화하지 않고 기존 `captureArchive`의 bounded 전체 구조/압축·member 무결성 측정을
`ReadArchiveInventory`로 재사용한다. 최초 project/standalone 검역은 `InspectArchive`/
`Inspect`의 내용 분석을 유지하고 위 여섯 DATA reader만 분리한다. 새 cache/graph/검사기나
원본 검증 생략은 필요 없다. 이 중복 해소의 네 상태는 모두 NO이며 read-only no-analyzer/
변조 거부·기존 layout 회귀와 actual 실행을 확인한 범위만 갱신한다.

중복 연결의 표적4.118s와 관련 npm/bootstrap Race8.396s는 PASS다. 전체 source 분석을
유지한 최초 검역과 무결성만 재측정하는 여섯 소비자를 구분했다. CLI 시험의 기존4분을
각 요청에 적용하고 요청 종료 직후 cancel한다. 따라서 여러 요청이 앞 요청의 남은
deadline을 나눠 쓰지 않으며 제품의 자원·실행 한도는 바꾸지 않는다. Local suite/hosted
누적 deadline은 실제 whole CLI 비용을 확인한 뒤 기존 workflow guard와 함께 맞춘다.

이 묶음의 canonical foundation53.776s/security26.477s는 PASS다. 뒤 tagged vet/Staticcheck는
requirements 시험의 제거된 runtime 변수 잔류를 검출했다. 같은 pin owner의 단일 사용으로
고친 태그 build-validity5.350s와 관련8package tagged vet2.055s/고정 Staticcheck3.399s는
PASS다. Compile 실패에 동반된 U1000을 실제 불용 코드로 단정하거나 helper를 제거하지 않는다.
Foundation/security의 실행 입력은 그 test-only 한 줄 교정 전이며 재현 원본을 유지한다.
현재 source/kernel/CPP/도구 공급물과 product behavior는 그 교정으로 변하지 않는다.
원본 `s04-python-cache-current-red-v1`/`green-v1`, `s04-python-current-race-v1`,
`s04-npm-readback-target-v1`/`race-v1`, `s04-python-npm-current-*`,
`s04-python-npm-tag-build-green-v2`가 정확한 source와 실행 범위를 소유한다.
Actual Python·npm whole CLI/hosted와 전체 S04 acceptance는 계속 OPEN이다.

### Go 내용 census — 기존 DATA의 다음 gap (`4857a61`)

기존 `gomodule.StaticInspector.InspectSources`/`InspectContent`의 source/native DATA는
IMPLEMENTED=YES/WIRED=YES이며, 전체 source/native/unsupported/DATA census와 analyzer
구성·default·현재 승인 연결은 각각 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=NO다.
같은 checked ZIP reader의 original member 순서를 사용해 O(n)으로 모든 regular member의
해시·역할을 측정한다. 비선택 DATA는 stream hash만 하고 source/native body 사본을 늘리지
않는다. 기존 Go syntax/testdata 규칙과 CRC·파일/전체200MiB 한도를 유지한다. Census는
전체 검증 뒤에만 반환하며 기존 content.Inventory/InspectSources를 그대로 재사용한다.
우선 이 누락만 로컬 검증하고 default/cache/실제 Go는 다음 연결로 남긴다. 빌드 중인
외부 scanner source/lock/worker와 독립이며 새 runtime owner·엔진·graph를 만들지 않는다.

Census 표적0.846s는 plain DATA/unsupported script·native disguise·구문/testdata·CRC와
원본 무결성 실패 뒤 provisional 사실 미반환을 확인했다. Configured ctor/사전 check6/
기존 공통 source analyzer를 연결하고 default Go get/download/upgrade가 그 analyzer를
주입한다. Go cache의 current Stage/readback은 기존 공통 내용 검증기를 사용하며 original의
historical 증거 경계는 유지한다. 새 self-consistent legacy 시험을 이전 go_cache.go에만
적용한 overlay RED8.972s는 Stage/current readback 둘 다 거부 누락을 검출했다. Current
source의 Go inspection/promotion/bootstrap 표적29.977s는 PASS다. 구현/연결은 YES,
QUALIFIED/ACCEPTANCE_CLOSED는 실제 Go/hosted/전체 대조 전까지 NO다. Synthetic current
cache facts는 기존 Cargo fixture를 재사용하며 actual scanner 결과로 표시하지 않는다.
Go source checksum/runtime pin·공통 version updater는 바꾸지 않았다.

고정1036-member source offer의 fresh canonical 빌드는1,437.755s에 정확한33ae4500/
171,651,104B를 재현했다. Host 설치 manifest는 동일하고 cleanup errors는 없으며
`runtime_qualified=false`는 유지한다. 앞 actual-v12와 exact 같은 bytes의 재현이지 새로운
제품 설치/whole CLI/전체 생태계 qualification이 아니다. 원본
`s04-taint-sink-canonical-build-v3`가 source/worker/SDK·해시·정리 결과를 소유한다.
기각된 DATA 진단 core 세 개는119process의 FD/maps/exe/mount 참조0과 hash/inode를
확인해514,954,016B 삭제했다. 첫 cleanup-v4는 좀비8개의 mountinfo EINVAL을 UNKNOWN으로
보존해 삭제하지 않았다. V5는 같은 세대 Z/FD0/maps empty/exe 부재를 모두 확인한 뒤
정리했고 query errors0이다. 선택된 core/이전 qualified baseline/SDK/source offer/raw는
유지한다. 최초 조회 실패를 성공으로 덮지 않는다. 원본 s04-superseded-core-cleanup-v4/v5 참조.

동일 current Go 묶음의 Race84.088s(inspection1.181/promotion61.065/bootstrap1.064),
canonical foundation49.426s/security18.915s, tagged vet10.666s/고정 Staticcheck16.388s는
PASS/source unchanged다. Current tagged CLI8.256s와 helper의 cached fresh receipts를
고정했다. Fresh core/CPP138을 반복하지 않고 같은 source supply로 npm whole execution/
post-Policy refusal actual-v14를 시작했다. 진행 중 Go/lock/observer 입력은 변경하지 않는다.
Raw `s04-go-census-target-v1`, `s04-go-cache-current-red-v1`, `s04-go-current-target-v2`,
`s04-go-current-*`가 이전 overlay와 current source·검증 범위를 구분한다. Go/Python의
실제 검역·설치/캐시와 hosted/whole S04는 여전히 MISSING이다.

Actual whole npm CLI-v14는 첫 요청172.69s(operator175.973s)에 FAIL했다. Exact TypeScript의
source/native/census/static/advisory는 완료했으나 npm-dynamic-lifecycle이
INCOMPLETE/M3_DYNAMIC_OBSERVER_FAILED여서 MANUAL_REVIEW와 실행 전 거부를 유지했다.
나머지 CLI/refusal은 미도달이다. Host/source 동일/helperwait0·자기 사본409,839,061B 삭제와
복원을 확인했다. 기존 CLI의 bounded 결과에는 정규화된 check code만 있고 backend의
operational cause는 노출하지 않으므로 attribution count14만으로 원인을 단정하지 않는다.
현재 source/core/CPP/budget 수정이 lifecycle 실패를 유발했다는 증거도 없다. 원본
`s04-npm-execution-cli-native-v14`를 유지하고 같은 exact artifact의 dynamic-only 경로를
임시 test overlay로 호출해 원래 backend 원인을 확보한다. 재검역/전체 CLI 반복이나 제품
검사·격리·trace 필수 조건 완화는 하지 않는다. 이는 diagnostic이며 whole qualification이 아니다.

Direct dynamic-only v15는16.00s(operator18.937s)에 aggregate EVENT_LIMIT를 재현했다.
정규화 원장9675/10000에 새 counted batch1014가 들어와 거부됐고, bytes는9459/2MiB다.
Source/native resolve/CLI 없이 같은 backend에서 재현했으므로 이를 source 수정의 인과로
쓰지 않는다. Host/source/overlay 동일·helperwait0·자기 사본409,845,094B 정리를 확인했다.
현재 `SharedObserver.receive`는 record당 원장 charge를 한 번 호출한다. TraceDiagnostic의
physical records22와 counted workspace access993·원장events는 서로 다른 단위다.

두 fresh writer와 두 readonly keeper가 한10k 정규화 원장을 공유한다. 기존 Go build에는
같은 physical collector10k와 별도의 aggregate40k가 이미 있다. 제품 예산을 바로 바꾸지
않고 임시 Go overlay-v16에서 npm aggregate만40k로 한정해 정상 총량을 측정한다.
Per-connection collector/helper10k·bytes2MiB·CPU/memory/wall·charging·필수 관찰은 유지한다.
결과를 얻기 전 최종 예산/해결로 채택하지 않는다. 원본
`s04-npm-dynamic-diag-native-v15`, measurement overlay/build/native-v16이 범위를 소유한다.

Measurement-v16는16.90s(operator19.919s)에 두 환경을 완료했다. 동일 artifact의 실제
정규화 총량17,293events/10,478B이며 usage 오류0이다. Host/source/overlay 동일·helperwait0·
자기 사본409,844,129B 삭제를 확인했다. 이는 유한 측정 ceiling의 진단 성공이지 제품 채택이나
whole CLI qualification이 아니다. 원래 receive의 단일 charge를 바꾸거나 이벤트를 버리지 않는다.

정식 수정 지도: 기존 네 세션 공유 원장·resource watch는 IMPLEMENTED/WIRED=YES지만
정상 TypeScript의 aggregate capacity는 QUALIFIED/ACCEPTANCE_CLOSED=NO다. 기존
Ports Sandbox 계약의 한 고정 상수40k를 정규화 원장과 environment resource owner가 함께
사용한다. 네 physical collector는 각각 원래10k이며 한 transaction의 권한을 새 세션마다
갱신하지 않는다. Bytes2MiB/CPU30초/memory/wall/launch4·다른 profile·CPP/syscall 권한은
유지한다. Common npm dynamic descriptor에 이 aggregate bound를 포함해 npm/PNPM/Bun/
project-root의 현재 증거와 이전10k 증거를 구분한다. 새 설정·엔진·별도 ledger를 만들지 않는다.
Boundary RED8.997s는 원래 두 owner10k 및 두 번째10k batch의 거부를 직접 확인했다.
같은 원장의 네 batch 수용/다음1개 거부·거부 시 counter 불변, 다른 profile/physical 불변과
current declaration/profile 거부를 먼저 확인한 뒤 직접 dynamic와 전체 CLI를 차례로 검증한다.

Product aggregate의 boundary GREEN11.542s는 resource/trace owner40k 일치, 네 counted
batch 수용·추가1개 거부와 실패 시 counter 불변, 다른 profile/physical 한도 불변을 확인했다.
기존 npm profile 사전 선언/결과와 이전 descriptor 거부도 PASS다. Descriptor는 실제
공통 상수를 포함하므로 PNPM/Bun/프로젝트 root의 같은 환경 의미에도 hash가 연결된다.

다음 actual의 알려진 fixture 전제도 원본과 대조했다. V14의 실제 JSON과 현재
`newExecutionReport`는 checks가 있으면 execution-result/v2인데 old CLI 시험은 v1만
기대했다. 제품 presenter를 바꾸지 않고 이 두 실행/거부 시험의 기대만 v2로 맞춘다.
정상 JSON에는 selected node마다 current source/native/census required profile과 완료를
확인한다. Manifest/SBOM schema와 실제 명령·case/4분 제한은 그대로다. 이 검증 추가를
관찰 EVENT_LIMIT의 원인이나 whole 실행 성공으로 혼동하지 않는다.

Product aggregate 묶음의 canonical foundation80.199s/security42.007s/tagged vet21.268s/
고정 Staticcheck33.828s는 PASS/source unchanged다. 관련 Race27.355s의 npm1.060s와
bootstrap1.418s는 PASS이며 sandbox는 기존 TestConsumedAdmissionPreservesOriginalTargetFailureAndSession의
Unix socket bind 도구 제한 한 건만 실패했다. 같은 source의 그 parent만 필요한 권한으로
Race2.341s에 PASS했다. 최초 batch를 모두 성공으로 덮거나 나머지를 반복하지 않는다.
원본 s04-npm-aggregate-red-v1/green-v1/related-race-v1/socket-race-v2와 같은 묶음의
foundation/security/vet/static-v1이 정확한 입력을 소유한다.

Current product client-v17은7.208s에 test-only diagnostic append overlay로 빌드했고
normal current product에는 측정 override가 없다. 기존 CPP138/core33ae/canonical proof를
재사용해 exact dynamic 먼저, 이어서 whole npm CLI/refusal을 한 namespace lifecycle에서
검증했다. Direct17.59s·정상 whole CLI 네 경우764.42s·post-Policy refusal 두 경우356.58s가
모두 PASS다(operator1142.690s). NoSKIP/helperwait0/source·overlay·Host 동일과 own 사본
409,850,236B 삭제를 확인했다. 실제 JSON의 current source/native/census assertions와
refusal의 no execution/no manifest를 통과했다. 이 npm 범위의 QUALIFIED=YES이며
Python/Go/Bun/PNPM과 whole S04/hosted는 이 실행 밖이다. 원본 product-native-v17 참조.

### Python 전체 archive 합산의 남은 두 소비자 (`c8ffe4e`, npm source 동결 중)

이전 whole archive metadata/global static graph charge는 IMPLEMENTED/WIRED=YES이고
그 범위의 검증을 유지한다. 그러나 `graphStaticState.footprint`의 runtime closure와
`inspectSelectedWheelArchives`의 사전 selected-wheel graph는 아직 normalized RECORD
Files를 합산한다. 두 연결의 WIRED/QUALIFIED/ACCEPTANCE_CLOSED는 NO다. 새 archive
검사기를 만들 필요 없이 기존 authenticated ArchiveEntries/ArchiveUncompressedBytes를
각 같은 소비자에서 O(1)로 사용하면 된다. 기존 한도·source profile·version owner는 유지한다.

Root/private Go를 바꾸지 않은 test overlay-v1의 실제 validated wheel은 파일5개/전체518B인데
runtime 합산은154B만 보고했다(RED1.512s). 두 번째 overlay-v2는 각각 기존 per-wheel 한도
안인 작은 ZIP 두 개(각5001 empty directory와4 regular member)의 합계10,010 entry를
whole graph10,000 한도 밖인데도 수용함을 재현했다. 두 RED4.487s는 callback이 두 번째
wheel까지 도달했고 첫 정상 single-wheel은 통과했다는 것도 확인했다. Artifact code는 실행하지 않았다.

임시 후보는 두 기존 합산 loop만 이미 검증된 archive totals로 대체한다. 실제 wheel의
원래 hash/RECORD/CRC/내용 읽기는 그대로고 normalized Files의 설치 역할 의미는 바꾸지
않는다. 그 후보+두 회귀의 artifact/pypi·inspection/pypi 전체 Race17.810s(7.656/2.613s)는
PASS다. Npm-v17 종료/복원 후 원래 네 파일의 build receipt hash와 후보 hash를 대조하고
Root/private source에 이 두 제품 변경·두 회귀만 반영했다. 이 두 합산 owner의
IMPLEMENTED/WIRED=YES, QUALIFIED=YES(named local scope), ACCEPTANCE_CLOSED=NO
(whole S04)이며 실제 Python/다른 소비자·grouped 검증을 이어간다. 원본 s04-python-footprint-existing-red-v1,
s04-python-whole-footprint-red-v2/candidate-race-v2 및 별도 overlay inputs가 범위를 소유한다.


### 측정된 npm suite deadline과 다음 Python actual 준비

실제 네 npm CLI 요청764.42s/두 refusal356.58s는 기존 두 suite5분보다 길다. 각 요청의
4분 제한·필수 case·ordinary/CUDA scope·Required를 유지하고 기존 workflow의 두 suite만
18분/10분으로 맞췄다. 새 job/profile은 없다. 기존 exact command validator에 normal
command를 연결하고 omission/comment/ignored/too-short 반례를 추가했다. Old guard의
normal 세 반례 수용 RED2.896s → GREEN2.891s, current parsed workflow2.607s 및 해당
checker Race24.452s는 PASS다. 원본 ci-deadline-guard-*/python-current-workflow-v1/
python-current-checker-race-v1 참조. 새 CI는 실행하지 않았다.

다음 기존 TestLinuxUVInstallCLIIntegration은 current presenter의 operation-result/v2를
기대하고, fresh/retained/root-upgrade/noop/new-root 각 결과의 current required content
profiles까지 확인한다. 제품 renderer/권한/한도는 바꾸지 않는다. Fresh tagged client와
policy/CGO0 stripped responder를 빌드했고 실제 namespace 검증/정리는 아직 미완료다.

Current Python 묶음의 canonical foundation76.803s/security21.330s/docs0.349s,
tagged affected vet2.806s/pinned Staticcheck9.950s는 모두 PASS/source unchanged다.
앞선 candidate full PyPI Race17.810s와 current checker Race24.452s의 정확한 범위를
유지한다. Source/runtime pin·observer CPP 입력은 바꾸지 않았다. Fresh client18.573s/
policy0.517s/responder0.381s receipt로 기존 namespace actual UV install 검증을 시작했다.
이 actual이 끝나기 전 Go 입력 변경·추가 Docker 시험은 하지 않는다. 원본
s04-python-current-{foundation,security,docs,vet,static,build,policy,responder,native}-v1 참조.


Python actual-v1의 첫 fresh UV 설치는56.13s/operator59.146s에 정당한 advisory review로
멈췄다. Idna3.10 exact946d195a의 required integrity/static/source/native/census/dynamic/
advisory는 전부 COMPLETED지만 GHSA-65pc-fj4g-8rjx/PYSEC-2026-215 AFFECTED 때문에
M13_ADVISORY_REVIEW → M5_DISTRIBUTION_REVIEW_REQUIRED/MANUAL_REVIEW이며 반입은
NOT_PERFORMED다. Cleanup/Host/source 동일·helperwait0·own409851050B 삭제를 확인했다.
이 결과를 scanner 오류나 Python 전체 성공으로 쓰지 않는다. 정렬된 출력은 advisory_details가
먼저이므로 schema_version이 첫 필드라고 가정하지 않고 원래 stdout JSON을 보존했다.

동일 actual 묶음의 public fixture를 OSV/PyPI로 한번에 대조했다. Idna3.15/3.16/3.19/3.20,
packaging24.2/25.0, httpcore1.0.9는 조회상 advisory0이고 requests2.32.5는 현재
GHSA-gc5v-m9x4-r6x2/PYSEC-2026-2275 대상이다. 공식 fixed2.33.0을 확인하고 public
requests2.34.2의 개별 조회도0임을 보존했다. 이것은 안전성 증명/검역 대체가 아니다.
정상 UV 시험만 idna3.19→3.20으로, requirements actual의 idna/requests만3.20/2.34.2로
조정한다. 제품 Policy/임계값/OSV 조회·cache는 유지한다. 기존 native original fixture에
명시 wheel version을 전달하고 파일명·guest install·원본 readback이 같은 값을 사용하게
맞췄다. 다른 기존 두 소비자는3.10을 그대로 명시한다. 새 helper/class는 없다.

첫 local test regex가 선언이 필요한 native integration 부모까지 선택해9개 준비 거부를
일으켰다. 이는 actual runtime 실행/제품 회귀가 아니며 원본 positive-fixtures-v2에 남긴다.
기존 synthetic contract 한 parent만 정확히 선택하고 tagged whole compile로 정정한다.
Requirements actual은 각 member의 current required content profiles도 확인하도록 보강했다.
원본 python-current-native-v1/first-failure 및 public-fixture-advisory-preflight-v1 참조.


UV actual-v2는 다섯 install 단계 전부 COMPLETED/ALLOW/반입 완료와 current content
profiles를 확인했다. 이후 native enrollment는 inspection 전
UPDATE_ORIGINAL_STATE_UNSUPPORTED로 멈췄다(test279.25s/operator282.250s).
Host/source 동일·helperwait0·own409851306B 삭제를 확인했다. Passive 공식 wheel 대조에서
3.19/3.16은 idna console entry를 선언하지만3.15는 선언하지 않는다. 기존 native DATA
producer는 site-packages만 tar 복사하므로3.19의 bin script를 원본으로 재현하지 않는다.
이 exact 제품 거부의 내부 cause는 아직 직접 노출하지 않았고 metadata/producer 범위의
불일치를 확인한 단계다. 제품 판정은 완화하지 않는다. 다음 후보는 기존 no-script
원본 fixture에 맞는 OSV 대조 완료 idna3.15→3.20을 사용한다. 원본
python-current-native-v2/idna-native-fixture-entry-points-v1을 보존하며 full actual은 미완료다.


UV actual-v3도 다섯 install/update/retained/new-root 단계를 모두 통과했다. 이후 no-script
native3.15 원본을 실제 UV로 만들었지만 첫 adoption 중 test300.95s/operator303.887s에
전체 test context5분 경계와 함께 registered runsc-trace source-built version mismatch가
나왔다. Host/source 동일·helperwait0·own409851306B 삭제를 확인했다. Existing
hosttool.validateDaemon은 version probe의 실행 오류와 invalid output을 같은 문자열로
반환한다. 이 결과만으로 보호된 runtime bytes가 다르다고 해석하지 않는다. 정확한 내부
probe error는 원래 로그에 없으므로 과거 인과는 미확정으로 보존한다.

다섯 CLI와 두 native operation을 합친 시험의 aggregate context만10분, 기존 workflow
suite만12분으로 맞췄다. 제품 검사·실행 한도는 바꾸지 않았다. 기존 exact command 목록과
짧은 suite 반례를 같은 변경에서 갱신한 guard3.019s/parsed workflow2.385s는 PASS다.
Current normal build-v4는3.207s에 통과했으며 full parent-v4는 실행하지 않았다. 불변인
다섯 case를 또 실행하지 않고 test-only append overlay-v1/v2로 기존 uvNativeEnrollmentCLI의
미도달 adoption/replay·unrecorded-cache refusal만 선택한다. 별도 Public idna3.15는
공식 metadata/digest/size와 고정 exact SHA를 대조한 DATA이며 자체 사본 정리 대상이다.
첫 selector-v1은4.58s/operator7.488s에 기존 helper가 기대하는 빈 Evidence directory가
없어 멈췄다. 이는 제품 오류가 아니고 source/overlay/Host 동일·helperwait0·사본409931457B
삭제까지 확인했다. V2는 그 사전조건만 보완했고 현재 실행 중이다. Normal product override는
없다. Current policy/responder-v4를 재사용하며 최초 실패/미실행 parent를 성공으로 덮지 않는다.
원본 python-current-native-v3/uv-ci-deadline-guard-v4/uv-current-workflow-v4 및
python-native-adoption-v1/v2가 정확한 범위를 소유한다.


Remaining native selector-v2는162.56s/operator165.529s에 PASS했다. Actual no-script3.15
원본 생성·unrecorded cache의 사전 거부·adopt=true current update(controls5)·이후 managed
replay(controls4), user config/files 보존을 기존 helper에서 확인했다. Source/overlay/Host
동일·noSKIP/helperwait0·정리 오류0·own409930361B 삭제다. V3의 다섯 실제 설치 성공과
이 남은 범위를 결합하되, 시간 제한으로 실패한 원래 full parent-v3 또는 미실행 v4를 PASS로
바꾸지 않는다. 같은 CPP/core의 재검증은 하지 않았다. 원본 python-native-adoption-v2 참조.

현재 제품/fixture/연결/필수 content assertions·UV deadline/guard 변경은2b02eeb에 보존했다.
바로 이어 normal current build/helper-v4의 불변 Go 소스로 pip·UV requirements full actual을
시작했다. 이 시험은 기존 native resolver·전체 content/dynamic/Policy·반입과 CLI binding의
component composition을 사용하고 test-only synthetic normalized block/after-materialization
failure/source-drift seam을 명시한다. 별도 default Root 전체/advisory·모든 S04 완료로
확대하지 않는다.20분 간격 local native monitor79320은 자동 상태/종료 receipt를 기록하며
CI는 실행하지 않았다. 원본 python-requirements-native-v1/timer.jsonl/operator.log 참조.

완료된 superseded Python 시험 client 네 사본은 기존 cleanup-v5의 같은 identity/FD/maps/
mount·동일 세대 terminal 판정으로 참조0/query오류0을 확인한 뒤229221469B 삭제했다.
현재 normal-v4/helper/최신 focused-v2/production Root CLI와 모든 원본은 보존했다.
이는 이 네 시험 사본의 정리이며 제품 C01/C02 전체 완료가 아니다.
원본 s04-completed-python-clients-cleanup-v1.json 참조.


Requirements native-v1의20분 타이머는1200.012s/2026-11-05T04:29:41.924646Z에 RUNNING을
직접 기록했다. 이후 timer1377.391s/native1352.748s/test1349.84s에 실패 종료했고,
Host/source 동일·helperwait0·정리 오류0·own409851335B 삭제를 확인했다. Pip의 정상6종/
문법·충돌·해시 거부는 통과했으나 transitive-block에서 기존 합성 주입기가 frozen plan에
없는 synthetic-req-block check(프로필 없음)를 추가해 idna의 POLICY_INPUT_INVALID가
발생했다. 다른 후속 미처리 항목의 같은 오류를 별도 원인으로 합치지 않는다. 실제 실패는
정당한 domain 계획 검증이며 제품 Policy를 완화하지 않는다. UV와 마지막 pip 실패 경우는
미도달이다. 원본 python-requirements-native-v1/-timer.jsonl/-operator.log 참조.

같은 기존 unit에 실제 profiled check 두 개의 frozen plan 검증을 추가하니 RED5.030s에
동일 membership 오류가 재현됐다. 기존 test-only requirementsBlockReport는 실제 check와
프로필/원래 증거를 보존하고, 명시 synthetic Evidence/Finding만 그 실제 check에 붙인다.
새 check/helper/type/declaration/runtime 권한은 추가하지 않는다. 고정 plan·원본 불변성과
기존 finding 회귀의 Race16.932s는 PASS다. 실제 전체 차단은 아직 native 미완료다.
원본 requirements-block-plan-red/green-v2 참조.

다음 local selector는 통과한 pip 정상/문법/해시 cases를 반복하지 않는다. Pip은 현재 승인된
최소 두 root(packaging25.0/idna3.20)를 한 번 만들고 미완료 transitive block/실제 materialization
실패 원복/source drift/final-empty만 확인한다. UV는 기존16종을 전부 수행한다. 제품이나
검사 override는 없고 test-only selector만 별도 hash로 묶는다. Trusted native logger의 정확한
21종 case/중복/parent/noSKIP도 검증한다. Full parent의32종 전체 성공으로 표시하지 않으며
기존 pip 통과 범위와 새 미완료 범위를 각각 유지한다.20분 local monitor는 JSON report 전문을
반복하지 않고 정해진 case 진행 줄만 기록한다. 새 CI는 없다.
