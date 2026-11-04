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
