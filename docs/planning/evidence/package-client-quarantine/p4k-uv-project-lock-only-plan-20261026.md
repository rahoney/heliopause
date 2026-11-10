# P4-K UV project lock-only 착수 감사

M13-004의 다음 missing acceptance다. J qualification 중 기준 c1a7659와 현재 J source를
읽어 사전 감사를 수행했다. Exact K baseline은 4c99452ba92e786d3b01091ac3081f7bcb5b69a3/treec7afae1cb0a8b000673635e2fcfb51af70aba6d7/2541 files다.
J와 readiness 교정은 commit했고 K source 수정 전에 이 감사와 map을 확인했다. Read: Queue/plan20§5/client 설계09§1·4·P4 상태,
Domain007 invariants, Contract003 invariants, Architecture008의 staging/identity/promotion,
lessons INDEX/ci L06/artifact L15/sandbox L08/evidence L16 및 현재 실제 source.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Native lock --upgrade/six controls/독립 공개 wheel binding | YES | YES(internal resolver) | YES(P4-E named native) | YES(named selection) | distinct Host control-only context |
| Complete current inspection/Evidence/Policy/approved cache | YES | YES(venv lanes) | YES(P1/P4-F..J named) | YES(named installed lanes) | explicit project-controls context support |
| Anchored project guard/bounded original snapshots/known backup cleanup | YES | YES(existing adapters) | YES(existing transaction consumers) | YES(named scopes) | UV single lock-file publication adapter |
| Public uv lock --upgrade / venv-free lock-only publication | NO | NO | NO | NO | mode/guard/approved readback/atomic lock publication/CLI/actual/failure |
| Persistent tools/temporary runners/final profile matrix | NO | NO | NO | NO | subsequent same M13-004 acceptance |

## Exact map·두 번째 검토

K1: Source-neutral Domain에 project control-only mutation mode를 명시하고 exact target을
project root로 결합한다. 기존 PythonVenv/new target 의미는 그대로다. Native project
resolver/update validator는 이 mode에서 lock-upgrade intent만 허용하고 generic complete
wheel binder/inspection/cache에는 명시한 새 mode만 추가한다. UV pip/installed preparation은
기존 venv mode를 계속 요구한다. Empty/fixed selection·context substitution 부정 회귀를
함께 검증한다. 이 단계만으로 public lock-only 지원을 주장하지 않는다.

K2: 새 UV control-only adapter는 기존 project transaction과 **같은 lock 이름**을 사용해
상호 배제한다. Existing anchored guard/bounded single-link control reader/snapshots와
known-member backup cleanup을 재사용한다. Go/Cargo adapter는 해당 생태계 grammar·상태
파일을 강제하고 venv adapter는 실제 venv discovery/ownership을 요구하므로 잘못된
mode나 가짜 venv plan으로 호출하지 않는다. 이 기존 구현들을 대체하지 않는다.

Original pyproject/native lock의 identity/bytes/mode/presence를 고정하고 native selection을
한 번만 실행한다. 선택 전체를 새 검사·Policy·cache/readback에 연결한 뒤 **uv.lock만**
원자 반영한다. Manifest는 immutable input이며 .venv/installed ownership은 읽거나 변경하지
않는다. 기존 유효한 uv.lock을 요구하는 lock-upgrade 입력 계약을 유지한다. User data를 보존하고
rollback 불확실성/foreign backup은 원본과
transaction lock을 남겨 다음 mutation을 막는다. Proof/old ALLOW는 승인 authority가 아니다.

K3: `helox uv lock --upgrade`를 명시 descriptor/production binding에 연결한다. `--project`와
cwd를 사용하고 target/adopt/임의 native 인수와 upgrade 생략을 runtime 준비 전에 거부한다.
보고는 lock update 상태와 installation NOT_REQUESTED를 분리하고 실제 Manifest/SBOM을
연결한다. 후속 sync는 다시 whole graph를 검사한다. Venv 부재/기존 임의 venv 보존,
기존 lock의 upgrade/no-op/empty·현재 records/failure/rollback/collision과 absent lock의 사전 거부를 검증한다.

두 번째 검토: source/role/runtime/observer/한도·Policy는 바꾸지 않는다. Native lock 선택과
Host 파일 반영/설치 결과를 구분하고, controller-selected lock bytes만 반영하며 cache에서
새 resolve나 artifact 실행을 하지 않는다. 기존 installed UV와 다른 client consumer의
실제 영향 범위를 보존한다. Helper별 full CI는 없고 최종 영향 profile 경계는 plan20§5다.

K1 최초 fixture 교정: strict lock-upgrade는 원본 lock을 요구한다. 새 시험이 absent lock으로
입력을 구성해 parser에서 거부됐고 source mode 동작에 도달하지 않았다. 고정 empty-project
원본 lock fixture를 제공하며 native grammar를 완화하지 않는다. 첫 canonical/Race 원본은
p4k-control-mode-v1-*로 보존한다.

K1은 [typed component receipt](./p4k1-uv-project-control-mode-local-20261026.json)의 canonical
profiles/전체 affected Race를 통과했다. 새 mode의 native public application/CLI actual은
미실행이며 전체 K 완료로 표시하지 않는다. K2 original guard/transaction/workflow가 다음이다.

K2 exact baseline은 a0135e8cd798b99debdc848a007e1ddff3225dc8이다. 기존 anchored guard/reader/snapshot/known backup
helpers를 재사용하고 새 control-only transaction·promotion과 실제 Application/Bootstrap
factory를 연결한다. Backup 디렉터리도 held os.Root의 controller RunID 이름으로 생성해
mutable absolute 경로를 다시 따라가지 않는다. 네 mutation boundary의 ENOSPC 원복과
foreign published/backup·manifest drift/reentry·사전 취소·다른 mode·missing/symlink/hardlink
거부를 확인한다. 설치 materializer나 기존 venv owner를 사용하지 않는다.

K2 focused11.848s/Race PASS는 lock-only 실제 파일 반영과 네 경계 원복·unknown 원본
보존/reentry 거부를 확인했다. Canonical Quick168.957s/security11.164s/docs0.258s도
PASS다. Full affected Race와 actual native/project-current Policy는 별도로 진행한다.
Native actual은 기존 installed 다섯 단계를 유지한 같은 test lifecycle 뒤 distinct
lock-only 세 요청(venv 부재/current no-op/edited-empty)을 이어서 검증한다.

K2는 [backend receipt](./p4k2-uv-project-lock-publication-local-20261026.json)의 full
affected Race146.556s와 actual173.014s까지 PASS다. Installed5/control-only3의 원래
관찰·현재 Policy·Host 복원을 확인했다. 새 실제 fixture만 추가한 default/production source
불변 경계는 raw native-fixture-only-reuse-boundary가 소유한다. Public K3 명령과 no-install
결과가 다음이며 전체 P4/원격 qualification은 계속 열려 있다.

K3 exact baseline은 e9565ad7577b7eb81319f0b13a144c523bfd4120이다. Existing descriptor/preflight/current record gating와
K2 factory를 재사용한다(IMPLEMENTED/WIRED/QUALIFIED=YES, named existing scope).
Public lock 명령·no-install report는 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED=NO,
MISSING은 이 연결과 실제 CLI/거부 경로 검증이다. 기존 source preflight를 공용 함수로
추출하고 current result facts 계산을 별도 renderer와 공유한다. 원래 판단 body의 mapped
동일성 및 기존 add/sync/다른 project report 소비자를 검증한다. Reader의 descriptor close는
source preflight 완료 시점이며 실제 held original guard는 K2가 다시 capture/유지한다.

K3는 [public CLI receipt](./p4k3-uv-project-lock-cli-local-20261026.json)의 canonical/full
CLI Race와 actual252.100s를 통과했다. Existing add/sync7와 lock-only4/current fresh approval·
명시 no-install 결과·원본 보존/helperwait0/Host 동일을 확인했다. Named UV project lanes는
locally qualified이며 persistent tools/whole P4와 최종 영향 profile은 별도다.
