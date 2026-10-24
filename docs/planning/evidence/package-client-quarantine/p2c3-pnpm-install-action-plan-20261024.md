# P2c3 — archive 설치 동작 사실의 보존과 complete-project 정책

2026-10-24, baseline `940d8e47c2a4639f3cbc158aeb92c12de43a50cb`.
M13-004/P2의 순차 하위 작업이다. [P2c 지도](./p2c-pnpm-selection-plan-20261024.md),
[M4 계약 §3·§7](../../../domain-model/13-m4-npm-install-promotion-contract.md),
[Inspection/Policy 계약](../../../domain-model/08-contract-inspection-sandbox-policy.md)을 따른다.
기존 구현을 교체하거나 Host lifecycle 실행 권한을 추가하지 않는다.

## 1. Source 작성 전 baseline audit

| 책임 / 실제 symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| npm archive 정적 검사 — `inspection/npm/static.go` Inspect/validateManifest | YES | YES | YES | YES | 새 machine-readable 설치 동작 사실만 없음; 검사기를 다시 작성하지 않음 |
| 기존 graph HostInstallAction — `artifact/npm/lockfile.go`, `policy/m4.go` EvaluateSet | YES | YES | YES | YES | 기존 lock의 hasInstallScript 경로 유지 |
| 정규화 Evidence 저장 — `domain/evidence.go`, `evidence/local/store.go` | YES | YES | YES | YES | Kind/subject/digest 저장 형식 재사용; 새 Store/schema 불필요 |
| archive 사실 → DependencyInspection → rootless M4 | NO | NO | NO | NO | 참조로 축약하기 전 typed 의미·exact subject·check·저장 참조 결합, rootless npm의 미확인/필요 사실 거부 |
| PNPM complete snapshot/layout/transaction/CLI | NO | NO | NO | NO | 본 단위 뒤 별도 정확한 지도와 실제 검증 필요 |

`DependencyInspection`은 현재 check와 EvidenceReference만 보존한다. `ProjectInspectService`는
rootless 프로젝트를 재사용하지만 `EvaluateProjectSet`에는 legacy graph의 HostInstallAction
분기가 없다. PNPM lock의 필드 부재를 false로 바꾸거나 Summary 문자열을 파싱하면 안 된다.
현재 public native8 결과는 archive acquisition/Policy를 실행하지 않았으므로 이 gap의 통과 증거가 아니다.

## 2. 정확한 수정 위치와 방법

1. `internal/core/domain/evidence.go`의 기존 Kind에 두 안정된 의미 상수를 선언한다:
   `host-install-action-required`, `host-install-action-not-required`. `DependencyInspection`에
   optional Evidence 값 하나와 copy-returning 결합 method를 추가한다. 사실 부재는 UNKNOWN이다.
   두 kind만 인식하며 exact acquired identity/digest, 이미 저장된 EvidenceID, required
   SUPPORTED/COMPLETED INSPECTION check를 모두 요구한다. 중복/상충/다른 subject/orphan은 오류다.
   Summary는 판단에 쓰지 않으며 기존 constructor/caller는 유지한다.
2. `internal/inspection/npm/static.go`의 기존 archive 순회와 validateManifest를 재사용한다.
   정상 archive에만 같은 check 아래 `npm-host-install-action` Evidence를 하나 더 낸다.
   기존 여섯 lifecycle key 중 하나라도 선언되거나 root `package/binding.gyp`가 있으면
   REQUIRED로 보수적으로 표시한다. 빈 script나 gypfile:false로 M4의 자동 반입 범위를
   넓히지 않는다. Nested binding.gyp/일반 scripts는 그 자체로 REQUIRED가 아니다.
   기존 Finding·check·summary/크기·경로·archive 거부를 유지하고 위반 archive에는 부정 사실을 내지 않는다.
3. `internal/application/install_inspect.go`의 acquired/single 두 경로와
   `dependency_batch_inspect.go`의 공용 batch 경로 모두 저장한 Evidence와 record를 결합한다.
   Record/fact validation은 run finalization 전에 수행한다. Batch의 static-all/closure와
   최초 오류 보존은 유지한다. Application은 ecosystem 이름/Kind 문자열/외부 schema를 해석하지 않는다.
4. `internal/policy/m4.go` EvaluateProjectSet의 기존 BLOCK 우선 뒤 npm complete-project에
   같은 M4 Host lifecycle 제한을 적용한다. `npm-static-archive`에 결합된 REQUIRED이면
   `M4_HOST_LIFECYCLE_UNSUPPORTED`, 사실이 없거나 해당 static check의 사실이 아니면
   `M4_HOST_LIFECYCLE_UNCONFIRMED` MANUAL_REVIEW다. 기존 entry review/필수 검사 완료도
   계속 필요하다. 다른 source/legacy EvaluateSet 및 M3 inspect-only 판정은 변경하지 않는다.
   Fact는 Finding이 아니므로 단순 선언을 M3의 악성 결과로 바꾸지 않는다.
5. `internal/cli/human_report.go`의 기존 필수 검사 미완료 설명에 새 UNCONFIRMED code를
   연결한다. 기존 다국어 문구를 재사용하고 JSON의 안정된 영어 code는 유지한다.

기존 Evidence record는 Kind/check/identity/digest를 이미 직렬화한다. 따라서 새 fact도 기존
store hash와 manifest의 EvidenceReference로 추적되며 별도 JSON schema/AI/설치 도구는 없다.
검사 수는 그대로이고 정상 npm Evidence 참조만 하나 늘어난다. 기존 array consumer의 가정도 확인한다.

## 3. 검증과 결과 해석

- 먼저 rootless npm의 모든 entry ALLOW/검사 완료만으로 UNKNOWN을 승인하는 현재 경계를
  직접 RED로 고정한다. 기존 graph의 HostInstallAction MANUAL_REVIEW 회귀는 계속 통과해야 한다.
- Domain: 정상 true/false·absence, 불변 copy, 다른 identity/digest/check/미저장ID,
  duplicate/conflict, unsupported/incomplete/optional/verification check 거부. Summary 변화 무관.
- Static: 여섯 lifecycle key, 빈 값, root binding.gyp, nested 일반 파일, ordinary scripts,
  archive violation/manifest identity 불일치에는 false fact 없음. 실제 composite의 두 필수 검사 유지.
- Application: acquired/single/batch 세 경로에서 사실 보존, 잘못된 사실은 성공 finalization 전 거부.
  Policy: true/unknown/false·BLOCK 우선·기존 required incomplete·다른 source 및 legacy graph 유지.
- Canonical format → quick/security/docs, 관련 Domain/Application/inspection/npm/Policy/Evidence
  전체 Race. 실제 npm archive/기존 소비자는 영향에 맞춰 확인하고 PNPM layout 완료로 확대하지 않는다.
- 현재 실행 중 CI402/head940은 source를 고정한다. 첫 확인은14:57:13 KST이며 이 작은 단위마다
  재push해 취소하지 않는다. 새 remote checkpoint는 다음 실제 연결 경계에서 묶는다.
- 전용 시험 bulk는 마지막 소비·참조/query errors0 확인 뒤 삭제한다. 기존 shared 개발 cache는 유지한다.

## 4. 두 번째 검토 — source 작성 전

두 번 대조한 결과 legacy graph의 bool을 rootless에 임의 복사하거나 PNPM lock으로 새로
추측하는 방법은 제외했다. 기존 Evidence Kind가 필요한 기계 의미와 subject를 이미 표현하므로
새 report/schema를 만들지 않는다. Single/acquired뿐 아니라 batch도 전달 위치에 포함했다.
Fact 검증을 run 완료 이전에 배치하고 malformed archive에는 부정 사실을 만들지 않는다.
M3 inspect-only ALLOW를 M4 Host 실행 권한으로 바꾸지 않으며 BLOCK 우선과 기존 fail-closed
조건을 보존한다. Root project scripts는 PNPM 입력 preflight의 별도 거부이고 여기서 승인하지 않는다.

공식 pinned PNPM source `91774e72a04d52ce61db06339696a433d383c764`의
`pnpm/crates/executor/src/lifecycle.rs`는 dependency 단계를 preinstall/install/postinstall로
구분하고 implicit node-gyp를 설명한다. 이 단위는 M4의 더 보수적인 기존 lifecycle 제한을
유지한다. 실제 peer snapshot/virtual-store/transaction은 여전히 MISSING이며 본 사실 전달
단위를 통과해도 전체 P2 acceptance를 닫지 않는다.

## 5. 구현 후 로컬 검증

첫 fixture는 project가 요구하는 verification check가 빠져 constructor에서 멈췄으므로 제품
RED로 쓰지 않는다. 이를 준비한 직접 before0.788s는 UNKNOWN archive fact에 M4가 ALLOW를
반환해 실패했고 같은 실행의 기존 graph HostInstallAction 수동 검토는 PASS였다.
원본 `p2c3-install-action-{before,valid-before}`를 각각 보존했다.

구현의 첫 canonical Quick142.145s/security7.002s/docs.217s와 관련 여섯 package 전체
Race5.283s는 PASS다. 실제 public `is-number@7.0.0`3,730B는 NOT_REQUIRED,
`node-sass@9.0.0`390,705B는 REQUIRED로 확인했다. 기존 Acquire/SRI verification/static/
실제 Evidence Store와 exact DependencyInspection 결합까지0.936s/PASS다. 스크립트나
dynamic/promotion은 실행하지 않았으며 두 archive와 record의 전용 t.TempDir는 시험이 정리했다.
정상 검사 수는 유지하고 static의 새 사실로 Evidence 참조가 하나 늘어난다.

Pinned upstream lifecycle exact capture21,395B는 Git blob977a71dfb126e56dc9b2e68419923b4d5816d400과
일치한다. [공식 source](https://github.com/pnpm/pnpm/blob/91774e72a04d52ce61db06339696a433d383c764/pnpm/crates/executor/src/lifecycle.rs)를
근거로 보수적인 기존 M4 범위와 native dependency 단계의 차이를 보존했다.
새 real fixture와 reason 연결을 포함한 최종 Quick33.846s도 PASS다.
[로컬 영수증](./p2c3-pnpm-install-action-local-20261024.json)이 최종 inventory·checks와 남은
범위를 연결한다. Source 검토에서 세 production constructor가 모두 공용 결합 helper를
소비하고 Store/manifest가 새 reference를 보존함도 다시 확인했다.
