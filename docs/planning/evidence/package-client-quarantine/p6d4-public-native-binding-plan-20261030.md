# P6-D4A — public native UV 준비 전 whole original 결합

Baseline47e0c23/tree8905c3db/3077tracked·runtime1370이다. 단일 active item M13-004다.
D3 named cached query는 CPP132/UV120·clean native4.56s·기존 Python2/68.28s로 local 완료다.
CI434는 source d81을 고정했고 poll03은7SUCCESS/macOS build2mFAIL/Linux 진행, next15:41:50UTC다.

Read: PROJECT/README/Queue·client09§3·plan20§5, lessons INDEX/관련 artifact L01/L02·sandbox
L03/L08/L12/L18·evidence L16·ci L05/L06, D3 local/raw와 fixed UV run/environment/cache 원본,
UVResolver/native session/archive reader·tool controls·Python preparation/runtime/startup gate,
BindPipxQueryWheels·freezePipxNativeInput·pipxWheelVolume의 seed/anchor/seal/readback·native
session dataKeeper/owned cleanup과 현재 withActualUVNativeRuntime를 대조했다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| UV native proposal와 independent public release/archive source | YES | YES | YES (기존 ordinary) | YES (named) | 재사용 |
| Whole byte snapshot/source/graph/startup gate | YES | YES (현재 Pipx) | YES (기존) | YES (named) | UV 준비 앞 consumer 연결 |
| Root0/0444/RO source volume·writer 제거/anchor/seal/readback | YES | YES (현재 Pipx) | YES (기존) | YES (named) | UV fixture의 exact readonly mount 연결 |
| Cached UV query kernel role/readonly ABI | YES | YES | YES (D3 local) | YES (named) | 재구현하지 않음 |
| Public frozen original로 native tool-run 준비 | NO | NO | NO | NO | 현재 test-only end-to-end DATA consumer |
| Native complete selection/census equivalence·runtime adapter/CLI | NO | NO | NO | NO | D4B 이후 별도 |

## Exact map와 두 번째 검토

기존 production parser/freezer/readonly-volume/lifecycle를 대체하지 않는다. 현재 native fixture
withActualUVNativeRuntime의 기존 body를 한 extended helper에 그대로 두고 원래 signature는 nil
input wrapper로 보존한다. Optional frozen input은 현재 consumer가 즉시 사용한다. 새 native
public tagged test는 NewLinuxUVResolverWithExecutorAndPolicy→resolveNativeSelection의 actual
proposal/lock→independent withUVArchiveInputs→unchanged freezePipxNativeInput의 owned bytes/
whole startup gate를 사용한다. 이 private type 이름은 pipx runtime/승인을 부여하지 않으며
새 native receipt를 꾸며내거나 uv-tool-install identity를 uvx로 바꾸지 않는다.

UV 준비 fixture에만 기존 createPipxWheelVolume(65MiB current cap)을 사용한다. 기존 dataKeeper의
root0/no-network runc writer로 seed/readback, ordinary readonly anchor, writer 제거/seal 후
native UV container에 /haa-wheels readonly mount를 추가한다. Native 전후 verifyReader 및
sole attachment/readback를 요구한다. Keeper는 DATA 보관 전용이며 image artifact를 실행하지
않는다. 기존 nil-input native3와 D1 probe의 args/순서/권한/한도는 유지한다. 새 source2는
uv_runtime_real_test.go의 좁은 fixture extension과 linux&&realcorpus tagged public test1이다.
Go production/Core/CLI·CPP72a1/132/UV120·wire·pins·지원/CPU/trace/CI 한도는 변경하지 않는다.

Actual native는 isolated/offline/no-index/find-links readonly source/no-build/no-config/no-cache/
fixed Python/copy/no-sources와 고정 /dev/null만 사용한다. 원본 Artifact program을 UV CONTROL로
실행하지 않는다. Native EACCES는 준비 terminal DATA이며 public Policy/프로그램 성공이 아니다.
Current whole/source/startup 결합·authenticated public source는 ACTUAL로 구분하고 Evidence/
Policy·selected user program·native command equivalence/aliases/CLI는 NOT_RUN/OPEN으로 둔다.
이 단위는 명명된 public pre-query binding이며 whole native golden 완료를 주장하지 않는다.

## 검증·정리

Canonical format/Quick·sandbox 전체 Race/tagged vet/pinned Staticcheck·tagged sandbox test build,
docs/security를 사용한다. CPP와 production Go inputs는 exact unchanged proof로 기존 결과를
재사용한다. Existing freeze/readonly negatives와 old nil-input native3/D1 probe·새 public native
binding actual을 확인한다. Host/source/42/ZIP/Docker·완료 trace/Outside0/helperwait0/전체 owned
정리와 frozen wheel digest/readback를 보존한다. 완료 duplicate ELF만 root 참조0/query errors0
뒤 삭제하고 현재 재사용 cache는 유지한다. CI434는 실제20분 GET을 계속하며 실행 중 push하지 않는다.


Topology 추가 대조: SharedObserver.startProfile은 기존 closed profile에 명시적인 expected
mounts를 받으며 현재 Node/Python 입력도 이를 사용한다. Test-only optional mirror 경우에만
기존 UV expected topology의 fresh slice에 /haa-wheels workspace/root-parent/9p/readonly 항목
하나를 더해 등록하고 실제 seal/anchors와 대조한다. 기본 profile/table/CPP/wire는 유지한다.
원래 nil-input 경로는 기존 startTrace 그대로다. Mirror seeding은 pipxNativeSession.dataKeeper/
createKeeper/removeKeeper와 pipxWheelVolume을 그대로 사용하며 pipx ZIP은 만들지 않는다.

## D4A named local 완료

Canonical6와 actual 기본 파일 선택/bytes 동일성으로 Quick·전체 sandbox Race 재사용을 확인했다. 기존 native4/22.899s operator와 새 public1/30.04s(parent),32.618s(operator)가 PASS다. 새 whole source/startup/RO mirror는 native query 전에 결합했고 native trace complete16events/1842B/workspace158·Outside/Network/Honeytoken/UnexpectedExec0과 고정 EACCES를 확인했다. Public 획득/조회 단계의 허용된 관찰 네트워크와 offline native 준비 trace는 구분한다. Source/Host5/42/ZIP/Docker 불변·helperwait0/모든 정리가 PASS다. 원본 보존 후 root118proc/ref0/errors0으로 완료 사본4개62042442B를 정리해 누적231ELF+cache6/약10.57GiB 논리 용량이다. Native complete census/equivalence·현재 승인 실행 adapter/aliases/CLI/hosted/final은 아직 OPEN이며 D4B의 기존 owner·고정 UV source 감사를 계속한다.
