# Planning

Heliopause 구현 milestone, 의존 순서와 현재 실행 작업을 기록한다.

## Documents

| File | Step | Status | Read when |
| --- | --- | --- | --- |
| [01-milestones.md](./01-milestones.md) | Step 12 | Complete | 구현 단계·의존 관계·완료 조건을 결정하거나 검토할 때 |
| [02-current-work-queue.md](./02-current-work-queue.md) | Step 13 | Complete | 다음에 실행할 작업·우선순위·blocked 상태를 관리할 때 |
| [03-m0-qualification.md](./03-m0-qualification.md) | M0 | Complete | M0 completion evidence, active gate, 공급망 검토와 제한을 확인할 때 |
| [04-m1-qualification.md](./04-m1-qualification.md) | M1 | Complete | M1 Domain workflow completion evidence, active gate와 M2 handoff를 확인할 때 |
| [05-m2-qualification.md](./05-m2-qualification.md) | M2 | Complete | M2 npm static inspect completion evidence와 M3 handoff를 확인할 때 |
| [06-m3-qualification.md](./06-m3-qualification.md) | M3 | Complete | M3 Linux dynamic inspect completion evidence와 M4 handoff를 확인할 때 |
| [07-m4-qualification.md](./07-m4-qualification.md) | M4 | Complete | M4 npm install/Promotion completion evidence와 M5 handoff를 확인할 때 |
| [08-m5-qualification.md](./08-m5-qualification.md) | M5 | Complete | M5 qualification, npm regression and M6 handoff evidence |
| [09-m6-qualification.md](./09-m6-qualification.md) | M6 | Complete | GitHub Releases qualification and M7 handoff evidence |
| [10-m7-mvp-qualification-contract.md](./10-m7-mvp-qualification-contract.md) | M7 | Complete | MVP qualification evidence matrix and completion record |
| [11-m8-production-trust-hardening-contract.md](./11-m8-production-trust-hardening-contract.md) | M8 | Complete | production Host tool, observer, privilege와 observation trust remediation |
| [12-m9-product-install-ux-contract.md](./12-m9-product-install-ux-contract.md) | M9 | Complete | transactional npm/pip/GitHub install UX와 hostile boundary |
| [13-m10-verified-distribution-bootstrap-contract.md](./13-m10-verified-distribution-bootstrap-contract.md) | M10 | Complete | release identity, artifact manifest와 verified bootstrap chain |
| [14-m11-dynamic-detection-depth-contract.md](./14-m11-dynamic-detection-depth-contract.md) | M11 | In progress | bounded process/filesystem/network detection과 raw payload non-retention |
| [16-m12-01-ecosystem-expansion-contract.md](./16-m12-01-ecosystem-expansion-contract.md) | M12 | In progress | PyTorch·Go Modules·Cargo·Terraform Provider 생태계 확장 |
| [17-m12-02-fix-list.md](./17-m12-02-fix-list.md) | M12-02 | Reserved | final red-team/fix gate |
| [18-m13-production-release-operations-contract.md](./18-m13-production-release-operations-contract.md) | M13 | Blocked | protected release 운영 설정과 최종 public deployment |

## Rule

- 이 README는 navigation 전용이다.
- milestone의 canonical source는 `01-milestones.md`다.
- 현재 작업 상태와 실행 순서는 Step 13 Work Queue가 소유한다.
- 상세 Architecture·Domain·Security 결정을 planning 문서에 복제하지 않는다.
