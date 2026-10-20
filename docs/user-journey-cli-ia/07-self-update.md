# Heliopause 자체 업데이트 — 설치된 신뢰 체인의 연장

상태: **설계 승인 / 구현 진행**. 2026-10-20 사용자는 기존 설치가 있으면 검역을
거친 업데이트를 우선하고, 별도 설치 전용 권한 경계와 Linux/WSL2 amd64 우선 구현을
승인했다. 새 명령은 아직 제품 완료 기능이 아니다. 소스별 현황·수정 위치·검사와
재검토 결과는 [실행 계획](../planning/evidence/m13-004-convenience/self-update-20261020.md)이 소유한다.

## 사용자 경로

- 최초 설치: 보호된 공식 릴리스의 provenance·digest·platform/runtime identity를
  검증하는 bootstrap이다. 설치된 Heliopause가 없는 상태에서 자체 검역을 했다고
  표현하지 않는다. [M10](../planning/13-m10-verified-distribution-bootstrap-contract.md)을 따른다.
- 기존 설치: **기존의 신뢰된 Heliopause**가 새 릴리스를 획득·검증·검역하고,
  Policy가 허용한 정확한 세트만 교체한다. 새 릴리스의 installer나 install hook이
  자신의 검사 통과를 선언할 수 없다.
- 우선 제공할 명령 제안은 `helox self-update --version vX.Y.Z`다. 명시한 exact
  version 하나를 처리한다. 최신판 탐색은 mutable 별칭을 설치 authority로 사용하지
  않으며, 이번 첫 구현의 필수 범위에 넣지 않는다.
- [일반 아티팩트 업데이트](./06-update-quarantine.md)는 기존 계약을 유지한다.
  프로젝트 안의 동일 이름 패키지를 본체 갱신으로 가로채지 않는다.
- npm·PyPI/pipx·Homebrew 채널은 owner가 이름을 확정한 뒤 같은 native 경로를
  호출할 수 있다. Heliopause 밖의 일반 package-manager 명령에 검역을 강제한다고
  보장하지 않는다. 미검역 본체 갱신을 검역형 업데이트의 대체 성공 경로로 안내하지
  않는다. Cargo/Go/Terraform 지원은 본체의 해당 배포 채널 존재를 뜻하지 않는다.

## 검역과 바이트 동일성

1. 현재 실행 파일·설치 종류·기존 release/manifest·보호된 runtime identity와
   권한을 읽기 확인한다. 외부/개발용 수동 설치나 불명확한 소유권은 자동 덮어쓰지
   않고 migration 필요를 보고한다. 사용자 config나 receipt만으로 root 설치 권한을
   만들지 않는다. 동시 갱신·활성 검사와 충돌하면 교체를 시작하지 않는다.
2. 공식 repository `rahoney/heliopause`의 exact version/tag/source commit과
   보호된 build workflow identity에 묶인 릴리스 세트를 고정한다. API 응답만으로
   보호된 tag/provenance를 입증했다고 처리하지 않는다. 임의 URL·다른 repository·
   ambient credential·proxy·PATH verifier를 허용하지 않는다.
3. bounded HTTPS 획득 후 manifest와 각 선택 파일의 size/digest, SBOM,
   runtime lock/image manifest와 attestation을 검증한다. verifier와 trust root는
   이전 신뢰 체인에서 온다. 새 bundle이 제공한 verifier/trust root를 사용하지 않는다.
   provenance 성공은 content safety나 Policy ALLOW가 아니다.
4. 획득한 native 실행 파일을 기존 Verification/Inspection/Evidence/Policy 경로로
   검사한다. artifact identity·획득 핸들·digest를 묶어 고정하고, 검사 뒤 live
   resolve·추가 download로 바꾸지 않는다. 새 Heliopause·observer·network helper도
   root 권한 없이 격리한다. 실행 probe가 해당 bounded 실험의 관찰이라는 한계를
   보고하며, 정상 기능·악성 코드 부재의 증명으로 쓰지 않는다.
5. 모든 required check와 Evidence가 완료되고 기존 Policy의 ALLOW를 얻은 세트만
   staging한다. BLOCK/MANUAL_REVIEW/UNAVAILABLE/미지원/취소·관찰 또는 cleanup
   불확실성에는 활성화를 하지 않는다. 검사 코드나 helper exit0 자체가 허가는 아니다.
6. staging과 활성화 직전에 동일성·전체 membership·mode·size/digest를 다시
   확인한다. root installer가 나중에 경로를 다시 열어 사용자 파일 교체를 수용하는
   TOCTOU를 막는다. root가 보호한 사본과 검사 세트가 정확히 일치해야 한다.

## 설치 및 권한 경계 — 승인된 방향

현재 user-owned versioned installer만으로는 Linux의 보호된 실제 CLI/runtime을
교체할 수 없다. 반대로 기존 network-policy helper에 파일 교체·임의 process 실행을
추가하면 resolver firewall 전용 경계를 훼손한다.

승인된 방향은 **설치 세트의 활성화·복원만 소유하는 별도의 좁은 privileged installer**다.
이는 새 범용 daemon이나 두 번째 제품 CLI가 아니다. 기존 network helper와 observer의
권한·프로토콜은 유지한다. 다만 현재 [Directory 규칙](../engineering/01-directory-structure.md)의
기존 resolver firewall mutation 역할과 분리된 설치 역할을 추가했다.

제안하는 설치 역할의 허용 범위:

- 보호된 기존 controller 실행 파일의 identity와 kernel peer credentials로 caller를
  확인한다. UID·process name·경로 문자열·사용자 receipt·새 package hook이 허가를
  대신하지 않는다. 인증된 요청도 아래 고정된 operation 밖으로 확대하지 않는다.
- exact 공식 release 세트·provenance·검사 결과와 묶인 활성화 transaction 하나만
  받는다. 단순 `ALLOW=true` JSON 파일을 root 권한의 근거로 받지 않는다. caller가
  제공한 사본은 root-owned 비공개 stage에 고정·재해시한 뒤에만 사용한다.
- argv/env/shell/목적지/UID/service 이름/임의 파일 쓰기를 요청으로 받지 않는다.
  root 작업은 검증된 HAA version directory, 고정 active pointer와 기존 HAA service
  전환·복원에 한정한다. 네트워크 획득과 일반 검사는 ordinary controller가 수행한다.
- 실행 중 검사·observer/network session이 확인된 종료 상태에 도달한 뒤 전환한다.
  query 실패는 UNKNOWN이며, 이 상태에서 dependent rebind/restore를 시도하지 않는다.
- 새 root-installed CLI와 helper 인증 identity를 함께 전환한다. 이미 실행 중인 이전
  controller의 완료 응답·다음 실행 인증까지 고려하며 전환 중 임의 새 caller를 받지
  않는다. 단일 잠금과 durable journal이 transaction owner를 명확히 한다.
- version directory와 prior active state를 보존한다. 고정된 후보 doctor/호환성 검사를
  통과해야 완료로 게시한다. 실패·재부팅·전원 중단 후 journal을 읽어 확정된 소유 상태로
  복구하며, 복구 실패는 별도 오류로 보존한다. 최초 실패를 복구 성공으로 덮지 않는다.
- 무인으로 임의 privilege를 얻지 않는다. OS의 관리자 인증이 필요한 설치라면 그
  경계에서 명시하고, `sudo helox` 전체 실행이나 setuid 본체를 요구하지 않는다.

현재 work item에 실제로 필요한 설치 역할만 구현한다. 사용자 디렉터리의 CLI 교체만으로
이 경계가 구현됐다고 보고하지 않으며, 미래 작업의 placeholder는 생성하지 않는다.

## Runtime·platform 범위

첫 검역형 업데이트의 실행 qualification은 현재 동적 backend가 지원하는
**Linux/amd64 + healthy trusted runtime**을 대상으로 한다. macOS/arm64에서
required dynamic 검사를 실행할 수 없으면 기존 설치를 보존하고 미지원 사유를
표시한다. provenance-only bootstrap 지원과 검역형 본체 업데이트 지원은 구분한다.
이 신규 경로의 초기 범위 승인은 기존 artifact/bootstrap 지원을 변경하지 않는다.

## Provenance 구현 결정

`github.com/sigstore/sigstore-go v1.3.0`을 사용해 offline Sigstore bundle을 검증한다.
user PATH의 `gh`나 bundle이 제공한 verifier를 실행하지 않는다. production trust root는
M13-003에서 검증·보존한 public-good root를 본체에 고정하며, 릴리스가 제공한 root로
대체하지 않는다. public-good CT/SCT·transparency log·observer timestamp를 각각
요구한다. private GitHub root나 unsafe artifact/identity 옵션을 허용하지 않는다.

서명 검증 뒤에도 정확한 artifact SHA-256 subject, OIDC issuer, signer workflow/tag,
signer/source commit, repository/owner ID, hosted runner, push trigger와 invocation을
확인한다. SLSA predicate의 workflow/source/run과 manifest도 같은 값으로 연결한다.
Predicate의 사용자 제어 값만으로 certificate identity 검증을 대신하지 않는다.
회전은 기존 신뢰로 검증되는 release에서 root를 명시 검토·갱신하며, 새 root만으로
현재 업데이트를 통과시키지 않는다. root·서명 또는 compatibility 불확실성은 중단한다.

CLI·observer·network helper는 선택한 릴리스 세트로 결합한다. runsc와 companion
manifest, host-specific pod-init/Docker 설정은 각각 기존 protected owner를 유지한다.
runtime lock이 같은 경우에도 실제 installed identity와 호환성을 확인해야 한다.
lock이 바뀌었거나 구성요소를 입증할 수 없으면, 완전한 검증·배포 경로가 구현될 때까지
교체를 거부한다. 누락 구성요소를 latest로 보충하거나 host config를 release 내용으로
덮지 않는다. 현재 release asset set에 없는 runsc를 이미 제공한다고 가정하지 않는다.

`.deb`/`.rpm` 소유 설치는 package database와 소유권을 유지해야 한다. native 관리
설치와 기존 OS package 설치를 같은 경로에서 무조건 덮지 않는다. package-manager
coexistence/migration의 검증 전에는 해당 설치 종류의 자동 갱신을 거부한다.
현재 개발용 `/usr/libexec/heliopause` 설치의 영구 전환도 별도의 migration이다.

## 결과·언어·보존

기존 presentation/progress owner를 사용해 획득·출처 검증·검역·staging·권한 요청·
활성화·doctor·복원의 실제 단계를 표시한다. JSON stdout과 사람이 읽는 report는
구분하고 같은 version/source/digest/Policy/Evidence/activation 상태를 참조한다.
기존 일곱 언어와 English fallback, 영어 technical code를 재사용한다. AI는 필요 없다.
취소나 실패도 기존 버전 보존 여부와 다음 행동을 명시하며 100% 가짜 progress나
`doctor` 일부 성공으로 전체 업데이트 성공을 출력하지 않는다.

검사 원본 근거·최종 결과·작은 recovery journal은 보존한다. transaction 종료 후
진행 중 작업이 사용하지 않는 download/stage/test source/cache/operator는 삭제한다.
rollback용 현재·직전 정상 release는 의미 없는 테스트 잔재와 구분한다. 활성·복구
가능성을 확인하지 못한 version을 공간 확보 목적으로 삭제하지 않는다.

## 최초 설치 인계

현재 실제 `bootstrap prepare`와 Root 파일 준비/등록의 경계·정확한 OS 설정·남은
qualification은 [Native 최초 설치 인계](./08-native-bootstrap.md)를 따른다.
