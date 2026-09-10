# Threat Model

Heliopause가 다루는 위협, 보호 자산과 신뢰 경계를 정의한다.

## Documents

| File | Decisions | Read when |
| --- | --- | --- |
| [01-threat-boundary-and-trust.md](./01-threat-boundary-and-trust.md) | D-001~005, D-009 | 위협·보호 자산·신뢰 모델·방어 한계 작업 |
| [02-isolation-and-inspection.md](./02-isolation-and-inspection.md) | D-006~008, D-014 | 격리 실행·Host 보호·자원 제한 작업 |
| [03-fail-closed-and-promotion.md](./03-fail-closed-and-promotion.md) | D-010~011 | 검사 실패·Staging·반입 작업 |
| [04-trusted-tooling-and-evidence.md](./04-trusted-tooling-and-evidence.md) | D-012~013, D-015 | 검사 도구·Host command·privileged helper 신뢰와 Evidence 무결성 작업 |

## Rule

- 이 README는 navigation 전용이다.
- 실제 결정의 canonical source는 각 상세 문서다.
- 작업에 필요한 상세 문서만 읽는다.
