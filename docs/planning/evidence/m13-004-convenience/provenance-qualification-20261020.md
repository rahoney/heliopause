# M13-004 — production provenance primitive의 로컬 검증

범위: 고정 Sigstore verifier 구현 단위. **전체 자체 업데이트의 완료가 아니다.**
기준 parent는 `770c30226d24d4fca10d5ad365e21ba06b8a27d7`이며 이 문서와 함께
커밋되는 후보의 실제 Go/manifest/lock 입력을 검사했다.

| 판정 | 상태 | 범위 |
| --- | --- | --- |
| IMPLEMENTED | YES | [Verify](../../../../internal/verification/releaseprovenance/verifier.go)의 offline 서명·certificate identity·subject·SLSA 검증 |
| WIRED | NO | 본체 release 획득/Application/활성화 caller는 다음 구현 단위 |
| QUALIFIED | YES | 아래 로컬 primitive·기존 소비자 회귀·cross compile 범위 |
| ACCEPTANCE_CLOSED | NO | M13-004 전체 A1..A9와 실제 자체 업데이트 경로는 미완료 |

기존 일반 GitHub digest verifier·InspectService·사용자 installer는 재작성하지 않았다.
별도 설치 역할과 초기 Linux/WSL2 amd64/same-runtime 범위의 사용자 승인을 Directory와
현재 Queue에 반영했다. 새 root executable·service·활성화 명령은 아직 없다.

## 실제 검사

고정 Go1.26.9, parent를 실제 fetch한 task-private Git 사본에서 실행했다.
새 input graph의 source inventory와 후보 blob/mode는 로컬
`docs/github-logs/self-update-20261020/provenance-final-source-inventory.json` 및
`provenance-final-checks.json`에 연결했다. 검사 전후 source가 같음을 대조했다.

| 검사 | 실제 결과 |
| --- | --- |
| canonical format | PASS; 명시적인 format profile만 source를 정리 |
| actual signed fixture focused | PASS7.53s; top-level3개/하위21개와 취소 원인 보존 |
| canonical Quick | PASS231.31s; format/lock/module integrity/build/test validity/Architecture/workflow/vet/Staticcheck/default |
| canonical security | PASS6.65s; gosec/Gitleaks/detection contract |
| canonical vulnerability | exit0/12.22s; 현재 reachable import/DB 범위 |
| canonical docs | PASS0.23s; 이 결과 문서 추가 후 문서/보안 후보는 별도로 다시 검사 |
| 관련 Race | PASS58.47s; 새 verifier·Go intake/verification·hosttool·CLI |
| 네트워크 차단 실제 실행 | PASS; 새 network namespace, UID1000, 그룹·capability 제거, 빈 환경에서 실제 서명 positive와13개 부정 회귀 |
| Darwin amd64/arm64 | CLI build와 각 verifier/Host/CLI test binary compile, 총8회 exit0; native 실행은 아님 |

실제 protected v0.1.0 signed bundle과 manifest는 기존 committed
[M13-003 evidence](../m13-003-release-build/qualification-20261017.md)를 그대로 쓴다.
fake verifier exit0·생성한 test signer·private key·사용자 token을 사용하지 않았다.
signature/proof와 source/tag/run/name/digest 교체를 거부하고, signed predicate의
source/workflow/runner/repository IDs/invocation도 certificate/manifest에 연결한다.
이는 provenance 확인이며 content safety나 Policy ALLOW를 뜻하지 않는다.

## 실패와 교정의 범위

- 첫 archive-only 사본은 ambient `/tmp/.git` 때문에 Go VCS status128로 build 전에
  실패했다. private 사본에만 정확한 parent Git metadata를 연결했다. compiler flag와
  global Git 설정을 완화하지 않았고 원본 실패는 보존했다.
- 다음 Staticcheck는 불필요한 nil-context 테스트 호출 SA1012를 거부했다. 그 호출만
  제거했다. analyzer suppression을 추가하지 않았으며 취소의 실제 cause 보존은 별도
  회귀로 확인했다.
- 첫 import graph의 Quick/security는 통과했지만 vulnerability는 x/net/gRPC의
  reachable 취약점5개를 보고했다. [dependency review](../../../engineering/02-coding-security-rules.md)에
  공식 수정 pin과 필수 전이 선택을 먼저 기록하고 최종 graph를 검사했다. 이전
  source의 DB 결과나 exploit 가능성을 이번 결과만으로 단정하지 않는다.
- 첫 standalone offline harness는 package 대신 repository root에서 실행해 fixture를
  찾지 못했다. crypto 미도달이며, 정확한 package cwd로 교정한 실제 실행과 보안
  수정 후 최종 바이너리 실행은 모두 통과했다.

## 다음 구현 및 보존 경계

다음은 exact release discovery/획득을 한 번 고정하고 기존 InspectService에 동일
바이트를 공급하는 경로다. 공개 attestation API의 익명 읽기 실제200을 확인했다.
현재 응답은 inline bundle 대신 repository-bound Azure URL이며 body는 raw Snappy다.
기존 고정 compression module로 bounded decode한 결과가 M13-003 signed bundle과
의미적으로 같음을 확인했다. 이는 **획득 audit**이며 아직 제품 download 구현은 없다.
SAS URL/query는 repository 증거에 저장하지 않았고 완료한 audit body/operator는 삭제했다.

검증이 끝난 Linux test binary·cross-compile 출력은 신원/결과 캡처 후 삭제한다.
작업 사본/빌드 cache는 바로 이어지는 같은 work item 구현에 실제 재사용하는 동안만
유지한다. 공유 pinned SDK/modules/tools와 원본 증거는 보존한다. 최종 사용이 끝나면
task-private source/cache/operator도 삭제한다. 설치 파일·helper·runtime·사용자 프로젝트의
영구 변경, registry publish·tag·merge는 이번 검증에서0이다.
