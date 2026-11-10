# P1 — uv pip baseline / 실행 중

기준 ef1e03e452f49219681a154b774ca2eed38dfaf9. M13-004만 IN_PROGRESS이며
현재 substep은 P1a 실제 uv lock 파싱·인증된 wheel 선택 경계다. P1 전체는 아직 미완료다.

## 독립 판정

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| 기존 PyPI install/update·wheel 검사·Policy/transaction | YES | YES | YES | YES | 기존 bounded scope는 재사용; 재작성 없음 |
| uv 0.13.4 공급 metadata/실제 출력 조사 | YES | NO | NO | NO | product pin/서명·runtime admission·maintenance 연결 |
| uv native lock parser와 선택 검증 | NO | NO | NO | NO | 실제 pylock fixture, malformed/대체/중복/누락 거부, wheel digest/metadata binding |
| uv install/upgrade/sync CLI·resolver·promotion | NO | NO | NO | NO | descriptor·runtime·resolver·complete set·rollback·actual qualification |

Read: [확장 설계](../../../user-journey-cli-ia/09-package-client-quarantine.md),
[전체 계획](../../20-package-client-quarantine-plan.md),
[runtime 유지보수](../../../engineering/05-runtime-upgrade-maintenance.md),
[Artifact Port](../../../domain-model/07-contract-artifact-port.md),
[M8](../../11-m8-production-trust-hardening-contract.md).
기존 `pypi/simple.go`, `wheel.go`, `sandbox/pypi_resolver.go`, pypi_update_resolver.go와
promotion 소유권을 대조했다. 기존 pip 보고서 parser를 uv 보고서라고 재사용하지 않는다.

## 실제 입력 확인과 수정 지도

공식 PyPI uv0.13.4 wheel metadata/digest를 확인했고 Host에서는 실행하지 않았다.
읽기 전용·무권한 container의 native help는 install에 --report가 없고 compile은
requirements.txt/pylock.toml 출력임을 확인했다. 실제 requests2.32.5 resolve는5개
package와 charset-normalizer의 wheel3개를 기록했다. Pylock1.0은 dependency edges가
없다. 이 조사만으로 production gVisor/network/검역 qualification을 주장하지 않는다.
원본은 local `docs/github-logs/package-client-quarantine-20261021/uv-*.json`이다.

1. `internal/artifact/pypi/uv_lock.go`: pylock1.0을 기존 pinned go-toml로 strict parsing.
   source·name/version·root·Python constraint·URL/hash/size·wheel identity를 검증하고
   모든 대체 wheel 후보를 보존한다. Lock만으로 complete graph/ALLOW를 만들지 않는다.
2. 같은 owner의 후속 selection binding은 실제 selected wheel의 전체 bytes를 기존
   InspectWheel/RECORD·digest 검사로 확인하고 lock·Simple metadata와 연결한다.
   인증된 Requires-Dist를 기존 marker/graph parser로 연결하여 누락 dependency를 거부한다.
3. `scripts/check/architecture.go`: 이미 고정·사용 중인 go-toml/v2 하나를 pypi parser에도
   명시 허용한다. Domain/Core import, shell/network/runtime 권한은 넓히지 않는다.
4. native uv lock fixture를 testdata에 고정하고 alternate wheel·substitution·unknown schema·
   direct/editable/VCS·중복·누락·bytes bounds를 테스트한다. 기존 wheel/marker/graph 검사를 보존한다.
5. P1b는 runtime lock/generator/probe·배포물·update-runtime과 typed network를 연결하고,
   P1c는 실제 uv install/upgrade/sync를 기존 guarded workflow에 연결한다. Unwired 상태를
   사용자 help의 지원으로 표시하지 않는다. 각 경계를 검증하고 다음 substep을 계속한다.

## 두 번째 검토

보고서 형식 이름 교체, 첫 wheel 임의 선택, lock만으로 dependency complete 단정,
UV 로그를 완료 authority로 사용하는 설계는 제외했다. 작은 typed parser는 현재 P1의
실제 입력 owner이며 미래 plugin/interface scaffold가 아니다. 새 dependency version은
없고 기존 TOML parser의 좁은 새 소비자만 추가한다. 입력 크기·entry/variant 수와 metadata
검증을 먼저 집행한다. 현재 Python/관찰기·npm/PyTorch runtime pins는 변경하지 않는다.

### 실제 materialization 확인 뒤 보완

Native uv no-index/no-build 설치는 INSTALLER만 남겼고 direct_url.json/uv_cache.json을
원본 URL authority로 사용할 수 없었다. 표준 WHEEL tag snapshot으로 정확히 하나의
lock 대안을 식별하고 전체 archive 검사 뒤 metadata를 연결한다. 실제 charset-normalizer
전체 Simple은2461 files라 기존1024 bound를 넘는다. UV의 독립 metadata는 전체
exact-release JSON172 files를 사용하며 한도를 넓히지 않는다. Canonical 자료 경계는
[UV P1 설계](../../../user-journey-cli-ia/09-package-client-quarantine.md#6-uv-p1의-실제-selection--metadata-경계)가 소유한다.
첫 parser 후보의 Quick175.11s/security6.65s/docs0.19s는 PASS지만 후속 선택·release
코드는 포함하지 않았다. 최종 후보를 다시 검사하고 native5 archive·4edge의 명시
realcorpus 시험과 별도로 기록한다. Filtered Simple fixture만의 첫 준비를 최종
exact-release 소비자 성공으로 바꾸지 않는다.

## P1a 로컬 종료 / 후속 P1b

최종 public-budget 후보의 canonical format/quick/security/docs/fuzz가 PASS이며 실제
uv 선택5개 archive의 SHA/RECORD·독립 exact-release metadata·4edge도 PASS다.
[로컬 영수증](./p1a-uv-local-qualification-20261021.json)은 code fingerprints와 실제 입력·
측정 결과를 소유한다. P1a IMPLEMENTED=YES, production WIRED=NO, QUALIFIED=NO,
ACCEPTANCE_CLOSED=NO다. 새 source 원격 CI와 전체 P1 runtime/CLI acceptance는 남는다.

전체 P1은 계속 IN_PROGRESS다. 다음 P1b는 official distroless uv image에서 tool만
기존 Python runtime에 공급하는 경로의 lock/검증/유지보수와 observer admission이다.
공식 uv+Python Trixie image는 Python3.14.8이고 기존 target은3.14.7이라 무조건 대체하지 않는다.
Bookworm image tag404와 실제 distroless metadata·public attestation200/1개 관측은 local
raw uv-image-*가 보존한다. Signature verification·pin adoption은 아직 아니다.
