# M12-003 Cargo parser와 공개 source acquisition — 부분 검증

M12-002 완료 `bdd3a29`에서 기존 Cargo owner를 재사용한다. 실제 고정 Cargo metadata/lock
fixture의 old FAIL → new PASS와 source acquisition 정상·부정 경계를 확인했다.
[결과와 범위](./result.json), [baseline audit](./audit.json), [정확 실행 후보](./qualified-source.json)를 따른다.

실제 빈 intake의 공개 `itoa@1.0.17`은 공식 HTTPS sparse index와 crate bytes를 독립 대조하고
정적 검사·기존 Evidence/Policy까지 ALLOW다. 이 결과는 isolated project resolver, verified
cache, add transaction 또는 observed build의 PASS가 아니다. Stock OCI metadata grammar
capture의 source replacement는 진단 fixture에만 사용했으며 제품 입력에서 허용하지 않는다.

기존 primary graph helper와 generic Application/Evidence/Policy 경계를 유지한다. 원래
원격 CPU/CUDA PASS를 현재 dependency/runtime 후보의 qualification으로 전용하지 않는다.
M12-003 IN_PROGRESS, M12-004 NOT_STARTED이며 push·remote CI·merge는 수행하지 않았다.
