# P1 공통 선행조건 — Go1.26.10 보안 패치 baseline

기준 source76e8112c9674cfd454a509c2a7a43425149fc6cc. M13-004만 IN_PROGRESS다.
P1b 공급 검증 단위는 커밋으로 보존했고 production UV runtime/CLI는 미완료다.
CI360 sourceb710b56의 Vulnerability가 GO-2026-6809로 실패해 기존 유지보수 절차로
공통 보안 선행조건을 먼저 충족한다. 임의 compiler 교체·vulnerability suppression은 없다.

## 독립 audit

| 책임 / 근거 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 update-runtime plan/apply/verify·capture-gvisor·SDK bootstrap | YES | YES | YES | YES | 재작성 없음, 현재 보안 버전 후보에 실제 사용 |
| 고정 Go1.26.9 compiler/runtime/tools/bundle | YES | YES | YES | YES | 과거 범위의 완료; 새 advisory에 대해서는 취약 |
| Go1.26.10 coordinated patch | NO | NO | NO | NO | 공식 metadata/SDK 검증·frozen plan·candidate·전체 bundle/observer·새 qualification |
| UV publisher metadata verifier | YES | YES | NO | NO | 새 source 원격 CI; 별도 runtime 공급·CLI |

Read: [유지보수](../../../engineering/05-runtime-upgrade-maintenance.md),
[기존 계획](../../19-runtime-upgrade-maintenance-plan.md),
[trusted tooling](../../../threat-model/04-trusted-tooling-and-evidence.md).
공식 [advisory](https://vuln.go.dev/ID/GO-2026-6809.json)는 2026-10-21 공개됐고
[Go release history](https://go.dev/doc/devel/release#go1.26.10)는 10-19 patch를 확인한다.
공식 Go archive metadata에서 Linux amd64 66,936,659B/SHA256
ccfe7ae822d2c2f41f4fe7c956199967a4dac22d0ee0d2b2bfdb440aab3b20c5를 확인했다.
이 발견만으로 SDK 실행·패치 채택 완료는 아니다.

## 기존 owner를 이용한 수정·검증 순서

1. 검증된 현재 SDK로 update-runtime coordinator를 빌드한다. Clean source와 disjoint
   task state에서 component go/version1.26.10의 실제 공식 metadata 및 support review를
   frozen plan으로 기록한다. 적용은 기존 apply의 별도 candidate worktree만 수정한다.
2. 기존 renderChanges가 go.mod, Go image/runtime lock·generated identities, quality tools의
   setupGo, 알려진 CI/release workflows, support review 및 observer BUILD 소비자를
   구조적으로 변경한다. 사용자 파일/과거 fixture/hash/증거는 일괄 치환하지 않는다.
3. Target SDK는 공식 archive hash/size 확인 뒤 task 경로에 준비한다. Host default SDK,
   /usr/local/bin/helox, 서비스·runtime 등록은 이 source 작업에서 변경하지 않는다.
4. Go owner 변경은 gVisor bundle compiler에도 영향을 주므로 기존 --capture 빌드로
   exact source/patch/SDK의 전체6 member와 module-lock을 확보하고 frozen plan에 연결한다.
   기존 member hashes를 새 binary에 붙이지 않는다. Observer도 같은 준비 경로로 빌드한다.
5. Canonical verify의 bootstrap/quick/security/workflow/release-gate/docs/freshness와
   별도 vulnerability를 새 SDK에서 수행한다. Quality tooling actual compiler/readback,
   reproducible bundle/observer 및 영향받는 실제 installed/npm/PyPI/GitHub/Go/Cargo/
   Terraform·CPU/cu126/cu130/cu132 qualification과 원격 Required를 독립적으로 기록한다.
   실행하지 않은 범위를 이전1.26.9 green으로 완료 처리하지 않는다.
6. 다음 runtime pin을 가진 최초 설치와 same-runtime self-update 경계는 유지한다.
   ABI/지원 불일치나 unknown 입력은 거부하고 mismatch를 허용하는 예외를 추가하지 않는다.

## 사전 두 번째 검토

수정 owner는 이미 존재하며 새 updater/새 compiler lock이 필요하지 않다. Compiler patch와
module version은 별개이고 현재 module pins를 이유 없이 갱신하지 않는다. Go1.27 major로
자동 전환하지 않는다. Metadata/tag와 actual SDK/bundle 동일성은 별도 검증한다. Plan/apply
성공은 qualification/Host activation 완료가 아니다. 재빌드 자료는 마지막 소비 후 지우고
작은 identity/명령/실패/결과만 보존한다. 본 audit는 승인 관문을 신설하지 않는다.


## 첫 실제 verify 실패와 좁은 보완 지도

공식 SDK 검증과 bootstrap은 PASS였으나 Quick production build가 VCS status128로
FAIL이다. 동일 candidate에서 git status는 성공하고 target Go build -x는 cd /tmp 후
실패했다. 선택한1.26.9/1.26.10의 cmd/go/internal/vcs/vcs.go는 .git directory만 root로
보므로 Git이 인정한 linked worktree의 .git file을 지나친다. 상위의 unrelated /tmp/.git는
읽기 전용이며 조작하지 않는다. 이 관측을 compiler patch 회귀라고 단정하지 않는다.

기존 apply의 isolated worktree·journal·full snapshot 보호는 IMPLEMENTED/WIRED YES로
유지한다. 보완할 gap은 verify의 build 입력 packaging뿐이다. scripts/update-runtime/apply.go
verifyPlan에서 기존 clean source의 exact HEAD를 attempt 내부 standalone shared clone으로
투영하고 frozen after bytes와 full modes를 적용한다. checkCandidate로 원래 worktree와
투영의 전체 tracked inventory·HEAD·bytes·modes·index를 검증 전후 대조한다.
Canonical profiles와 VCS stamping은 그대로 실행하며 -buildvcs=false, GIT_DIR 전역 override,
shared /tmp/.git 변경은 사용하지 않는다. Configuration receipt에 실제 checked source를 남긴다.

tests: 실제 standalone .git directory·HEAD/전체 snapshot/after와 mismatch 거부,
외부 후보 편집 보존을 기존 updater 회귀와 대조한다. 최종 Go candidate는 이 source 보완도
포함해야 하며 이전 frozen plan을 몰래 바꾸지 않고 새 source에서 다시 생성한다.
첫 실패 원본 go12610-initial-verify 및 go12610-vcs-*와 verification-865014279는 보존한다.


Standalone source 보완의 canonical format/Quick349.90s/security8.19s/docs0.23s는 기존
Go1.26.9에서 PASS다. [영수증](./verification-source-local-20261021.json)이 exact source를
소유한다. 최초 Go1.26.10 전체 bundle은710.343s/strict verifier PASS이며 module-lock은
8402c7be…로 기존과 같다. 새 member bytes/hash는 실제 capture에 기록됐고 기존 hash를
재사용하지 않았다. 아직 새 observer·재현 build·최종 frozen plan/검역·원격 CI 성공이 아니다.


## 최종 runtime 후보에서 드러난 합성 fixture 결합

보완 sourcecdb9e93에서 Go1.26.10과 실제 six-member hashes를 함께 고정한 plan
97f330a3…의 bootstrap은 PASS, Quick는 default tests에서 FAIL이다. VCS production build는
통과했다. 정확한 실패는 githubrelease의 source/staging-only, releasebootstrap의 file-mechanics
fixture runtime hash 비교, releaseinstall의 historical SBOM Go-version/current image 검증이다.
이 세 소비자는 과거 v0.1.0 자료를 읽어 이미 unsigned synthetic assets로 바꾸거나 현재
compiled runtime과 같다고 가정한다. 현재 same-runtime 거부가 동작한 것이며 prod gate를
완화하거나 historical signed bytes를 새 버전에 맞춰 편집하지 않는다.

보완 범위: internal/testutil/releasefixture의 단일 unsigned material binder가 generated
runtimeidentity에서 현재 compiler/image/runtime tuple을 읽어 합성 manifest/SBOM/image metadata를
생성한다. 실제 네 package의 다섯 fixture 소비자가 재사용하고 기존 HAA signature positive/negative는 원본을
유지한다. 내부 test helper import 하나만 architecture에 선언한다. 과거 signed proof가
재결합한 synthetic bytes를 승인하지 못함과 changed runtime 거부를 별도로 유지한다.
먼저 현재 SDK의 canonical 검사를 통과시키고 해당 source를 기준으로 새 frozen plan을 만든다.
첫 실패 verification-3192815458/go12610-final-verify 원본을 덮지 않는다. 기존 updater/production
검증 owner는 재작성하지 않으며 test data 책임 gap만 고친다.


합성 fixture 보완 최종 source의 기존 SDK Quick27.61s/security4.62s/docs0.20s는 PASS다.
[영수증](./runtime-fixture-local-20261021.json)이 source hash와 보존한 경계를 기록한다.
독립 두 번째 gVisor build590.291s는 전체6 bytes/hash·manifest가 첫 build와 일치했고
observer103 latch/298.313s도 PASS다. Cdb 기반 Go1.26.10 후보의 별도 vulnerability20.138s도
PASS다. 이 component/scan 결과를 새 fixture source의 전체 Go patch 종료로 확대하지 않는다.


## Fixture 보완 source의 최종 로컬 결과

Source5994732의 fresh plan e64de5ec…는 실제 Go1.26.10 tuple과 six-member hashes를
함께 적용했고 canonical bootstrap/Quick/security/workflow/release-gate/docs/freshness를
292.982s에 모두 PASS했다. 같은 source의 vulnerability14.15s와 fuzz77.97s도 PASS다.
[최종 로컬 영수증](./go12610-local-qualification-20261021.json)이 exact source/pin hashes,
독립 재현·103 latch와 실제 namespace12 positive를 소유한다. 최초 journal namespace는
보호 versions directory가 없는 operator 준비에서 실패했고 private versions만 준비한
동일 binary 후속 성공을 구분한다. Production gate를 수정하지 않았다.

검증한9파일을 원래 branch source에만 가져와 전체 tracked bytes/실행 modes를 checked
standalone source와 대조했다. Host SDK·설치·서비스는 바꾸지 않았다. 새 source의 원격
Required/installed CPU·독립 cu126/cu130/cu132와 actual activation은 별도 미완료다.
M13-004/P1을 COMPLETE로 표시하지 않는다. 다음은 원격 qualification을 확인하면서
고정 UV runtime 공급·CLI 연결을 진행하는 것이다.


원본 checkout의 최종 docs 시도는 기존 untracked github-logs/c3-approval Markdown의
상대 링크 두 개 때문에 거부됐다. 해당 원본 자료를 편집하거나 검사에서 숨기지 않았다.
전달 검사는 Git의 전체 tracked tree와 명시14 변경을 담은 별도 standalone checkout에서
수행하고 최종 index tree를 대조한다. 패치 밖 qualify-wsl2-cli.sh의 로컬0744/검증0755는
owner 실행·Git executable mode가 같으며 기존 로컬 group/other 권한을 변경하지 않는다.
