# M12-003 Cargo 로컬 qualification

[결과](./result.json)는 public crates.io acquisition·독립 checksum 검증, guarded add,
retained approval/cache, offline observed build와 bounded output publication의 로컬
acceptance 완료를 기록한다. [실행 후보](./executed-split-source-portable.json)는
708개 source/mode와 정확한 test/helper/observer를 연결한다. Product/CPP/runtime는
[707→708 delta](./split-cache-source-delta.json)에서 동일하며 새 CPU/cu126은 실제 실행했다.

- Cargo 정상·보안·retained build13개와 Go/Cargo add·Go build14개는 실제 PASS다.
  [독립 Run/output/Evidence bindings](./actual-cargo-run-bindings.json)와
  [소비자별 binaries](./consumer-client-bindings.json)가 원본·후보를 연결한다.
- 필수 sdist·wheel/npm/GitHub/Go/lifecycle/Python root-profile 직접 소비자도 PASS다.
  [원본 inventory](./portable-log-inventory.json)는 로그의 전체 바이트·SHA256을 보존한다.
- 같은708 후보 [CPU523.65s](./split-storage-cpu-full-result.json)와
  [cu1262070.42s](./reclaimed-native-cu126-full-result.json)는 기존15m/40m 안에서 PASS다.
  16,117개 파일 게시, 검사 전용 NumPy의29개 승격집합 제외, viewer NOT_ATTESTED의
  결과·승격 Evidence 참조, helper parentwait0/bounded Docker absence/original12 복원을 확인했다.
- [Canonical checks](./canonical-checks.json)는 quick/platform/corpus/vulnerability와
  fresh complete-candidate workflow/security/docs·Darwin TEST compile을 구분한다.
  [Runtime custody](./runtime-custody.json)는 실제 six-member bundle을 연결하며
  [CI aggregate 검토](./ci-duration-review.json)는 개별 test·제품 한도를 유지한다.
- [최소 first-fault chain](./minimal-first-fault-chain.json)과
  [compiler causal pairs](./compiler-causal-pairs.json)는 같은 fixture의 before/after 범위다.
  이번 full PASS를 과거 지연의 정확 원인 확정이나 split-cache 해결로 확대하지 않는다.

앞선 회계 deadline, WSL 중단, native/loop/split40m timeout과24GiB capacity 실패는
각 raw/result로 보존한다. 실패 project 자료의 [verified custody 이동](./retained-native-storage-recovery.json)은
project rollback이나 당시 승격 성공이 아니다. 새 실행은 물리 공간 회수와 ephemeral
환경 복원·동일 source/runtime 재검증 뒤 fresh native cache/target/private/journal을 사용했다.
제품의 durable sync·Policy·관찰·timeout을 변경하지 않았다.

Native macOS와 새 remote Required는 NOT_RUN이다. Darwin TEST compile이나 과거
M12-001 remote green을 전용하지 않는다. CUDA release-support 결정은 M12-005에 남는다.
최종 문서 delta를 전체 docs/security/whitespace로 확인한 뒤 로컬 checkpoint를 남긴다.
NO PUSH / NO REMOTE CI / NO MERGE. 다음 work item은 Queue의 M12-004다.
