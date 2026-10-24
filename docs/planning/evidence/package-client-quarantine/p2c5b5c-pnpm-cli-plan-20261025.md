# P2c5-B5c — PNPM 사용자 명령 연결 착수 감사

Baseline e9ca85e4d2af2acd1485f98ff2557762cf3bb4a8/tree1833b8925425f28979fe55f3c978b63ecddb8882.
CI407/run40128032402는 이 앞 source의 원격 qualification이며 첫 확인은2026-10-25
06:02:54 KST다. 대기 중 같은 M13-004/P2의 독립 CLI gap을 진행하고 새 push로 취소하지 않는다.

Read: Queue, package-client 설계§1/2/4·확장 계획§3/4/5, B5b3 audit/local·B5b4 plan/local,
User Journey01/02·terminal presentation, Artifact/Domain 계약과 관련 artifact/evidence/ci lessons.

## 독립 baseline audit

| 책임 / 실제 위치 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| PNPM add/install/update complete native selection→Application→target/state | YES | YES(actual bootstrap) | YES(named local, final CI script actual197.220s) | NO(P2) | 실제 사용자 CLI 진입 |
| typed client descriptor/preflight/help — cli/package_client.go, PreparePresentation | YES | YES(pip/uv-pip) | YES(existing) | NO(PNPM) | PNPM 세 명령/인수·대상·client/source 라우팅 |
| guarded complete-set report — cargo_update.go writer와 UV sync caller | YES | YES(existing) | YES(existing) | NO(PNPM) | npm package 표시·INSTALL/UPDATE와 실제 records |
| 성공 후 opaque records — UVSyncProjectPromotion.PublishedRecords | YES | YES(UV) | YES(existing) | NO(PNPM) | PNPM committed approval/Manifest/handle의 같은 narrow getter |
| request-local PNPM factory — bootstrap/pnpm_project.go | YES | YES(actual tests) | YES(named local) | NO(CLI) | CLI 요청마다 actual adapters/operation·bindings |
| native/unmanaged payload adoption | NO | NO | NO | NO | 독립 old archive/source proof; 이 CLI 단계로 지원 완료 표시하지 않음 |

이미 구현한 resolver/cache/transaction을 대체하지 않는다. `npm` source와 `pnpm` client를
구분하며 pnpm add를 npm install로 바꾸거나 complete set에 가짜 primary를 붙이지 않는다.

## 현재 수정 지도

1. 기존 descriptor에 PNPM client와 `add <spec>`/`install`/`update [name]` 세 명령을 등록한다.
   단일 existing project의 cwd/명시 --target을 사용하고 invalid/empty/duplicate/unknown flags,
   local/direct/VCS·잘못된 selector·unsupported platform은 runtime composition 전에 거부한다.
   Help는 source/target/cache/runtime을 준비하지 않는다. Preflight의 bounded control read는
   구문 검사이며 실제 Begin의 original guard를 대체하지 않는다.
2. CLI의 typed request/use-case binding과 bootstrap의 한 요청 factory를 연결한다. Run은
   Python client factory를 Python route에서만 소비하고 PNPM은 기존 npm source의 전용 factory를
   선택한다. Existing npm/pip/UV/Go/Cargo/Terraform 라우트와 지연 help/doctor를 유지한다.
3. Promotion은 transaction 성공 뒤에만 실제 approval의 Manifest·staged handle을 getter로
   제공한다. 기존 UV getter와 같은 opaque records 표현을 사용하며 target의 user-writable
   marker를 다시 읽어 report authority로 쓰지 않는다. Bootstrap private wrapper는 기존 Apply와
   records getter를 함께 보관하고 CLI는 성공한 exact request만 completion으로 출력한다.
4. 기존 complete-set report의 entry/check/Evidence/policy 표현을 재사용한다. Install/add는
   INSTALL, update는 UPDATE로 표시하며 npm 결과를 crate/Cargo build 안내로 출력하지 않는다.
   기존 update schema/기존 소비자 동작은 보존하고 필요한 private rendering 책임만 추출한다.
   원래 오류·inspection facts·cleanup 실패와 confirmed publication 의미를 확인한다.
5. `--adopt`를 노출한다면 현 구현의 explicit changed-control enrollment만 정확히 설명한다.
   독립 승인된 기존 payload/cache/Evidence 검증을 계속 요구하며 unknown/native payload는
   source proof 후속 acceptance까지 거부한다. Native adoption 전체를 이미 지원한다고 쓰지 않는다.
   Help/examples·영어/한국어와 기존 나머지 언어 fallback은 실제 wired 기능만 표시한다.

구체 preflight reader·records getter/reset과 report의 공통 책임·실패 표시를 기존 source와 더
대조한 뒤 두 번째 검토를 완료하고 source를 작성한다. Canonical profiles·해당 CLI/bootstrap/
promotion/application 회귀/Race와 실제 public CLI add/install/update/peer/empty/no-op·원본 보존/
거부·정리를 확인한다. CI407의 앞 source 성공을 이 후보 qualification으로 전용하지 않는다.
Whole P2와 native adoption/최종 누적 qualification은 계속 미완료다.

## 두 번째 검토와 정확한 연결 확정

- `cli/pnpm_project.go`는 typed PNPMProjectRequest/Outcome/Applier, 세 validator와 binding을
  소유한다. Existing descriptor가 common persistent --target을 이미 제공하므로 중복 flag를
  만들지 않는다. Canonical no-symlink existing target의 bounded regular single-link controls를
  preflight에서 읽어 기존 NewPNPMSelectionInput으로 검사하며 실제 workflow는 새로 guard한다.
  --adopt는 현재 구현된 managed controls 등록만 설명한다. New-target/global/workspace나
  임의 native flag를 통과시키는 빈 interface/config는 만들지 않는다.
- `bootstrap/bootstrap.go`의 Python factory 선택을 실제 Python route 안으로 좁힌다.
  PNPM client는 commandRuntime 전에 Linux amd64를 확인하고 기존 npm source 블록 앞의
  전용 binder를 사용한다. 기존 typed executor/observer/policy owner를 그대로 전달한다.
- `promotion.PNPMProjectPromotion.PublishedRecords`는 성공한 guard의 Manifest/opaque cache
  handle만 반환한다. Begin에서 이전 report pointer를 비우고 Commit의 transaction 성공 후에만
  고정한다. Manifest digest 문법은 mutation 전에 확인한다. Before-commit/failed-next-request와
  ordinary 성공을 default tests에서 검증한다. Root marker를 report authority로 읽지 않는다.
- Bootstrap private factory는 workflow와 promotion owner를 함께 가진 작은 wrapper를 반환한다.
  기존 actual 소비자의 Apply는 그대로 위임하고 CLI caller는 successful Apply 뒤 getter를
  소비한다. Application/Core ports와 complete set은 바꾸지 않는다.
- 기존 `writeProjectUpdateReportWithRecords`의 본문을 private common mutation renderer로
  좁게 추출한다. 기존 update wrapper는 기존 schema/UPDATE 결과를 보존한다. 새 install/add는
  같은 entry/check/Evidence/policy 표현의 project-install/v1·INSTALL을 사용한다. npm branch의
  package count·project completion과 fresh inspection 문구만 추가하며 기존 Cargo footer로
  흐르지 않게 한다. 실패·JSON/한글/다른 언어 fallback·terminal escape를 함께 확인한다.

즉 기존 node layout/engine/selection을 교체하지 않고 current gap의 사용자 진입점/보고만
연결한다. Metadata가 실제로 게시됐다는 사실과 renderer 성공은 별개이며 getter·writer 오류가
workflow 실패나 검사 사실을 지우지 않는지 확인한다. 위 map의 source부터 구현한다.

## 첫 canonical 거부와 경계 보존 교정

첫 format1.522s 뒤 Quick41.596s는 production/test build 통과 후 architecture gate에서 CLI의
직접 runtimeidentity import를 거부했다. Default/actual은 미실행이며 first-source와 로그를
보존했다. npm adapter도 runtimeidentity를 import하지 않으므로 import 예외나 version literal로
우회하지 않는다. Bootstrap이 static command 생성 직후 고정 generated PNPMVersion 값을
`SetPNPMClientVersion`으로 주입한다. 이 inert annotation은 runtime 준비/사용자 설정이 아니며
preflight에서만 기존 typed parser를 호출한다. Actual resolver는 자기 generated pin으로 다시
검사한다. CLI에 package dependency를 추가하지 않고 bound identity 부재·잘못된 입력/지연
help를 검사한다. 기존 architecture allow map은 그대로 유지한다.

교정 Quick139.581s는 architecture/vet/Staticcheck를 통과하고 default에서 기존 launcher examples의
선언된 existing project/pin fixture 누락과 새 human test의 uppercase enum 기대를 거부했다.
실제 renderer는 기존 계약대로 Installation/설치 등으로 번역하며 JSON만 enum을 유지한다.
기존 UV source fixture와 같은 방식으로 inert project와 generated static pin·표시 target의
테스트 사본을 제공하고 human assertion을 실제 사용자 label에 맞춘다. 제품 인수 검증을
생략하거나 enum을 human에 강제하지 않는다. Default/actual 전체 성공으로 쓰지 않으며
composed-source/quick이 최초 범위와 원본을 소유한다.

## 현재 결과와 같은 CI owner 연결 지도

교정 canonical Quick41.895s/security10.387s/docs0.231s와 다섯 package 전체 Race134.175s가 PASS다.
실제 사용자 CLI180.442s/test178.02s는 single/peer/empty의9개 새 operation으로 install/repeat/
add/update와 명시 관리 설정 등록, 실제 Manifest/opaque cache records·SBOM/원본 inode·0600와
user file 보존을 검증했다. Implicit control enrollment/unproven native adoption은 거부했다.
기존 npm inspect CLI 실제9.234s도 PASS다. 두 owning lifecycle은 helperwait0/Host 구성 전후 동일이다.

CLI actual을 기존 CI project profile에 별도 고정 명령으로 추가한다. 기존 cache+transaction의
5분 invocation을 합쳐 한도를 올리지 않고 CLI도 독립5분 invocation으로 둔다. Profile의 같은
native 공급·protected bootstrap client/helper·EXIT cleanup을 재사용한다. Checker는 새 env와
정확한 function·호출 순서를 기존 project scope에 포함하고 missing/comment/filter/env/중복/
실패 무시를 거부한다. 실제 qualified CLI shell line과 canonical CI line의 동일성을 대조한다.
이는 새 CI 호출 연결 검증이며 기존 cache/transaction 실제 검사를 변경 없이 다시 돌리지 않는다.
새 원격 전체 순서/Required는 CI407 뒤의 최종 후보에서 확인한다. 앞 runtime source와 app 코드의
Race/actual 결과를 새 checker source의 검사로 바꾸어 쓰지 않는다.

## 최종 후보의 검증 범위

최종 제품 source는 앞 five-package Race134.175s/CLI actual180.442s와 같다. 이후에는 CI caller/
checker와 canonical 지원 문서만 추가했다. 이 최종 경계의 format1.218s/workflow1.879s/
Quick47.946s/security8.659s/docs0.231s·checker 전체 Race30.325s가 PASS다. Darwin arm64의
bootstrap/CLI test compile6.736s도 PASS이며 native macOS 실행으로 확대하지 않는다.

현재 CLI와 source/client 구분·새 install schema와 기존 update schema·지원 인계 한계는
canonical 설계§9에 한 번 기록했다. [로컬 영수증](./p2c5b5c-pnpm-cli-local-20261025.json)이
source/검사/원본·실패와 정확한 범위를 소유한다. 세 완료 actual/compile 디렉터리501,366,905B는
작은 원본·전체 inventory 및 참조/query errors0·inode/mode/owner/hash 확인 뒤 삭제했다.
활성 source/SDK/quality/observer·다음 locked archive와 CI407 timer는 보존했다.

Native/unmanaged old payload의 독립 source proof가 다음 acceptance다. New-target/global/
workspace/execution은 현재 지원 profile 밖이다. CI407은 앞 e9ca85e만 검사하며 이 로컬 CLI
성공을 그 원격 성공으로 바꾸어 쓰지 않는다. 작은 helper 재push로 진행 중 CI를 취소하지 않는다.
