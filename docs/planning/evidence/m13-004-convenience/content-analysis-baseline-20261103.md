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
