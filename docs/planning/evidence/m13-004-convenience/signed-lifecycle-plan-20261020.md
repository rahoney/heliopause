# M13-004 — 실제 서명된 native 세트의 단계별 시험

2026-10-20 사용자 승인: 시험 태그와 공식 서명 빌드를 진행한다. Source는 CI355의
Required+9 SUCCESS와 전체 merge tree 동일성이 확인된 `a22aa12202009ba21c7e379001d27f9ae2e58a60`에
고정한다. 이 계획의 문서 commit을 signed source로 묵시 변경하지 않는다. 기존
[초기 설치 인계](../../../user-journey-cli-ia/08-native-bootstrap.md)와 production verifier/
installer/격리 시험을 재사용하며 새 runtime·권한 프로토콜·Policy를 만들지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| native4+metadata3·기존 SDK/검역/활성화/복구 | YES | YES | YES, local components/ordinary CI | PARTIAL | 실제 새 signed inputs와 booted lifecycle |
| actual native9 producer | YES | YES, tag workflow | YES, local producer | PARTIAL | 현재 소스의 official tagged build/attestation |
| first files/register/ordinary health | YES | YES | YES, local mocked-positive/real-negative | PARTIAL | 실제 새 서명·clean OS·실제 service |
| signed base→target/systemd/crash/reboot | PARTIAL | PARTIAL | NO | NO | 아래 순서의 실제 증거 |

## 순서와 범위

1. 현재 실제 source/CI·태그 부재·creation/immutable rules를 읽어 저장한다.
   `v0.1.1-rc.1`을 정확한 승인 source에 생성하고 기존 tag-push release build/attestation을
   실행한다. 기존 v0.1.0/main/develop·보호 규칙·public Release는 변경하지 않는다.
2. 성공 workflow의 source/ref/event/run과 exact13-file artifact를 고정한다. 공식 고정
   gh2.102.0을 이전 공식 checksum과 재대조해 independent verification에 사용한다.
   manifest/native4/metadata3를 포함한 actual bundle signature, exact tag/source/signer/
   workflow/run/subject·size·digest·SBOM/runtime lock/image manifest를 확인한다.
3. 새 production SDK가 actual selected7을 승인하고 wrong source/tag/subject/tamper를
   거부하는지 확인한다. SDK positive를 위해 mock·trust root 교체·새 privileged 입력을
   추가하지 않는다. 실제 독립 검증할 Root helper는 실제 사용할 Root 보호 사본이어야 한다.
4. 먼저 base의 파일 준비/등록·ordinary doctor/nominal을 가능한 격리 환경에서 확인한다.
   실제 systemd가 없으면 file/kernel/SDK 증거로만 기록하고 service/reboot 성공으로
   확대하지 않는다. 현재 개발용 설치의 영구 migration은 이 승인에 포함하지 않는다.
5. 첫 결과가 확인된 뒤 `v0.1.1-rc.2`의 필요 입력/source를 다시 고정하고 official
   서명 build를 진행한다. 같은 source의 두 version이라면 payload가 같을 수 있으므로
   version/path/metadata 전환과 실제 code-change regression의 범위를 구분한다.
6. 기존 production controller의 검역/Policy/Evidence→Root 동일 세트/전환·건강 검사와
   failure/복원/중단·재부팅을 확인한다. Public Release가 없는 tag artifact만으로 actual
   public Release API acquisition을 통과했다고 하지 않는다. 실제 booted environment와
   public end-to-end 입력이 없으면 exact gap을 남기며 main merge/public publish는 실행하지 않는다.

## 태그 보존·정리와 두 번째 대조

현재 creation rule24969939는 v* 생성에 owner를 허용하고 immutable rule24969940은
v* update/deletion/non-fast-forward를 bypass 없이 막는다. 승인 전의 '나중 삭제 가능'은
즉시 git delete가 가능하다는 뜻이 아니다. 관리자라도 해당 시험 태그를 정확히 한정한
보호 규칙 조정이 필요하다. 이번에는 규칙을 변경하지 않고 시험 중 tag identity를 유지한다.
검증 이력·attestation/source/manifest와 실제 소비가 끝난 뒤의 태그 정리에는 이 조건을
명시한다. 운영 tag 보호를 일반적으로 끄거나 자동 rollback/force update하지 않는다.

완료된 download/archive/binary/test root/cache/operator는 마지막 사용과 실행 참조 확인 뒤
삭제한다. 작은 원본 검증 증거·서명 proof·현재/직전 정상 설치와 recovery journal은 보존한다.
서명 proof 보존과 payload 정리를 구분하며 credential·temporary signed URL은 저장하지 않는다.

두 번째 대조: tag build는 exact official source의 서명 파일을 만들기 위한 것이고 공개
Release 발행/기존 main-history publication gate의 우회가 아니다. SDK가 기대하는 tag/source/
workflow/push identity는 유지한다. 두 tag를 동시에 만들거나 SDK mock positive를 actual
signed lifecycle로 전용하지 않는다. M13-004만 IN_PROGRESS로 유지한다.
