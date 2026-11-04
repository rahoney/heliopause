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
