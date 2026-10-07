# M12-002 — Go source 검사 범위와 resolver 예산 보완

Baseline `56c703f7722a9f3e5ae74a4b6d353f35cc9de629`에서 사용자가 승인한 신규
Go resolver 연결별 200,000 charged records를 적용했다. 기존 observer source와의
차이는 해당 constant와 profile branch뿐이며 기존 charging·process/image/role·
filesystem/network predicates, CPU/CUDA/npm/PyPI/GitHub 한도는 동일하다.
Collector 10,000 records/2 MiB와 Go CPU/memory/time 한도도 유지한다.

원인은 `StaticInspector.Inspect`가 모든 `.go` member에 source 문법을 강제한 것이다.
같은 subject의 x/tools0.49/0.34 invalid 11/16개는 모두 testdata assets다. 고정
Go1.26.8 공식 package search와 [literal compiler fixture](./go-package-rules.json)를
대조했다. `./...`은 testdata를 제외하지만 직접 지정/import한 source는 compiler가
거부한다. [최소 수정 전 FAIL](./static-before.log)의 정상/부정 fixture는 수정 후
통과했다. 정확한 상대 directory 역할만 구분하며 module 이름·비슷한 이름·case
차이는 면제하지 않는다. 전체 archive 인증/용량/경로/중복/link/CRC reads는 유지한다.
Explicit selection의 실제 observed offline build qualification은 아직 남는다.

생산 observer의 실제 remote-sink below/overflow Go200k·npm10k boundary와 전체
C++ latch가 통과했다. Actual CLI pflag Get+retained Download 및 Cobra7/tools8
modules의 전체 source→SumDB→inspection→Policy→cache/control publish→reuse는
통과했다. 기존 npm lifecycle/environment·GitHub ELF·필수 sdist13.13s·wheel10.77s도
같은 observer binary로 실행했다. 이전 M12-001 full PASS를 새 후보 결과로 전용하지 않는다.

실제 gRPC41 modules는 syntax/Policy 차단을 지나 **별도 캐시 파일 수 제한으로
FAIL**했다. 전체 preflight는 199,788,477 bytes/11,572 files이며 첫 초과 시점은
148,190,059 bytes/10,795 files다. 512 MiB 용량보다 10,000 file bound가 원인이다.
판정 불변의 외부 scalar diagnostics로 전체를 확인했으며 원래 controls·승인 부재·
exact guard cleanup·helper wait0·원래 설치 복원을 확인했다. 생산 오류에도 고정
numeric counts/limits만 보존한다. 캐시 파일 한도 변경은 별도 사용자 답변 pending이다.
이 FAIL을 완료나 성공으로 바꾸지 않는다.

Canonical quick/security/representative corpus와 Darwin cross-compile을 확인했다.
Corpus는 canonical preparation으로 고정 hashes를 재검사한 입력을 사용했다. Darwin
결과는 compilation만이며 native 실행 PASS가 아니다. 최종 docs는 receipt와 함께 검사한다.

[Result receipt](./result.json)에 candidate source/binary·원본 log/hash·범위를 연결한다.
M12-002는 IN_PROGRESS다. Observed offline build/output publish·최종 동일 후보의
전체 local integration/CPU 영향 검증은 별도 MISSING이며 M12-003/004는 NOT_STARTED다.
NO PUSH / NO REMOTE CI / NO MERGE.
