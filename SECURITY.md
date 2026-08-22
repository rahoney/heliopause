# Security Policy

## Supported version

MVP 기간에는 `main`의 최신 커밋만 보안 수정 대상이다. 별도 release tag나
장기 지원 branch는 아직 제공하지 않는다.

## Reporting a vulnerability

취약점, credential 노출, sandbox escape, observer attribution 오류, digest
binding 우회, network-policy 우회 또는 비-`ALLOW` Promotion 가능성을 public
Issue에 게시하지 않는다. GitHub의
[private vulnerability report](https://github.com/rahoney/heliopause/security/advisories/new)를
사용해 다음 정보를 최소 범위로 제공한다.

- 영향을 받는 commit과 Host/runtime identity
- 재현 단계와 예상·실제 Policy/operation 결과
- Artifact identity/digest 및 민감하지 않은 최소 synthetic fixture
- 잠재 영향과 알려진 완화 방법

실제 credential, 개인 Artifact, Host 파일, raw observer payload 또는 공격
대상 식별 정보는 보내지 않는다. 필요한 경우 먼저 redacted reproduction으로
경계를 확인한다.

보고가 접수되면 공개 전 재현·영향 범위·완화책을 비공개로 조율한다. 수정은
관련 regression test, security/quality gate와 trust-boundary 재검토를 통과한
뒤 공개한다. scanner 실행 실패, 관찰 불완전 또는 cleanup 불확실성은 취약점
없음으로 처리하지 않는다.
