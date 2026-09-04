# M8 Entry Decision — Production Trust Hardening Contract

M0~M7은 당시 정의된 MVP qualification과 재현 가능한 evidence를 완료했다.
후속 external security review에서 production Host command, gVisor observer
lifecycle, privileged firewall operation과 실제 observation capability에 대한
trust gap이 확인되었다. M8은 과거 M7 evidence를 삭제하거나 실패로 바꾸지
않고, production-ready release 전에 이 gap을 fail-closed 방식으로 닫는다.

M8이 완료되기 전의 Heliopause는 qualified MVP security engine이지
production-ready distribution이 아니다.

## 1. Trusted Host tool boundary

Sandbox와 Promotion이 실행하는 Host tool은 logical name과 ambient `PATH`로
선택하지 않는다. production composition root는 tool별 trusted identity를
구성하고, Sandbox/Promotion adapter는 검증된 executor만 주입받는다.

- Docker, runsc, observer helper와 firewall backend executable은 absolute path,
  symlink resolution, regular-file type, owner, mode와 writable parent chain을
  검증한다.
- runsc와 HAA가 배포하는 helper는 release lock의 digest/source identity를
  만족해야 한다. Docker와 Host firewall tool은 지원 version/capability와
  trusted installation identity를 검증한다.
- 검증과 실행 사이에는 executable identity를 다시 확인한다. untrusted user가
  바꿀 수 있는 file이나 parent directory는 production tool이 될 수 없다.
- child environment는 command별 최소 allowlist로 새로 만든다. `DOCKER_HOST`,
  `DOCKER_CONTEXT`, Docker credential/config, proxy, user `HOME`, package-manager
  config와 기타 ambient Host environment를 임의 상속하지 않는다.
- Docker는 **검증된 local Docker daemon endpoint**만 사용한다. endpoint가
  local transport인지, owner/permission과 daemon identity/capability가 현재
  runtime contract를 만족하는지 확인한다. arbitrary `DOCKER_HOST`, remote TCP,
  SSH context 또는 ambient Docker context는 거부한다. `/var/run/docker.sock`
  하나를 영구 invariant로 고정하지 않으며, rootless를 포함한 다른 local
  endpoint는 별도 검증된 configuration으로만 지원할 수 있다.
- lookup, identity, environment 또는 endpoint 검증 실패는 capability
  unavailable/incomplete이며 다른 PATH tool이나 remote daemon으로 fallback하지
  않는다.

`TrustedHostTool`은 Infrastructure adapter 구현 세부다. Core, Application,
Policy에는 executable path, Docker endpoint, process environment나 Host tool
version type이 노출되지 않는다.

### M8-002 production composition

`internal/hosttool`이 이 경계의 concrete owner다. Linux composition root는 CLI
operation마다 executor 하나를 만들고 Sandbox, resolver와 npm/PyPI Promotion에
같은 instance를 주입한 뒤 operation 종료 시 isolated Docker client state를
정리한다. 기존 adapter의 인자 기반 test seam은 유지하지만 executor 없는 Linux
convenience factory는 Host command를 찾지 않고 fail-closed한다.

기본 설치는 root-owned `/usr/bin/docker`와 `unix:///run/docker.sock`을 사용한다.
다른 정상 local 구성이 필요한 Host는 root-owned `0755`
`/etc/heliopause` 아래 root-owned, non-symlink, group/world-non-writable
`host-tools.json`에 다음 bounded schema를 설정할 수 있다. ordinary `helox`는
설정 identity와 내용을 읽어 검증할 수 있어야 하며, 같은 directory의 민감한
`pod-init.json`은 별도 `0600 root` mode를 유지한다.

```json
{
  "docker_path": "/usr/bin/docker",
  "docker_endpoint": "unix:///run/user/1000/docker.sock",
  "observer_helper_path": "/usr/libexec/heliopause/haa_gvisor_observer"
}
```

- unknown field, trailing JSON, 16 KiB 초과, non-absolute executable 또는 non-Unix
  endpoint는 거부한다.
- Docker executable과 모든 parent는 root-owned real path이며 group/world write가
  없어야 한다. `PATH` candidate나 symlink chain으로 fallback하지 않는다.
- rootful endpoint는 root-owned, rootless endpoint는 current effective uid-owned일
  수 있다. endpoint는 실제 Unix socket이어야 하고 parent chain은 real directory,
  trusted owner, group/world-non-writable여야 한다. socket mode는 Docker access
  group을 허용할 수 있지만 replacement를 허용하는 `/tmp` 같은 shared writable
  parent는 거부한다.
- Docker command마다 executable과 socket identity를 다시 확인하고, explicit
  `--host`와 fresh owner-only `--config`를 앞에 붙인다. child environment는 그
  isolated `HOME`/`DOCKER_CONFIG`와 `LANG=C`, `LC_ALL=C`만 가진다.
- daemon version을 확인한 뒤 daemon이 `runsc-trace`에 등록한 canonical absolute
  path를 읽는다. 그 exact executable이 runtime lock의 architecture별 SHA-512와
  release를 모두 만족해야 하며 별도 PATH `runsc`는 사용하지 않는다.
- controlled Linux 설치는 runsc를 root-owned, non-writable
  `/usr/libexec/heliopause/runsc`에 설치하고 그 exact path를 Docker runtime에
  등록한다. 일반 사용자가 쓸 수 있는 `/usr/local/bin`이 writable한 Host에서는
  그 위치를 trusted runtime identity로 승인하지 않는다.
- firewall executable은 fixed absolute candidate 중 신뢰 검사를 통과한 것만
  등록한다. 실제 privilege separation과 typed firewall operation은 M8-005가
  소유한다.
- observer helper 기본 경로는 root-owned, non-writable
  `/usr/libexec/heliopause/haa_gvisor_observer`다. 다른 controlled installation은
  같은 bounded root-owned configuration의 `observer_helper_path`만 사용할 수
  있으며, absolute identity·regular file·owner·mode·parent chain을 helper 시작
  직전에 다시 검증한다. helper는 pinned gVisor source commit을 build identity로
  명시하고 supervisor start 직전에 그 identity acknowledgement를 확인한다.

## 2. Observer supervisor and exclusive endpoint ownership

production Linux dynamic path는 process-scoped observer supervisor 하나가 다음
lifecycle을 소유한다.

```text
exclusive runtime ownership
  → normalized HAA receiver 준비
  → pinned observer helper 시작
  → remote endpoint readiness와 helper identity 확인
  → npm/PyPI/GitHub Sandbox 실행
  → 모든 stream complete·Sandbox cleanup
  → helper 종료 확인
  → receiver와 endpoint 정리
```

- npm, PyPI resolver, wheel runner, sdist builder와 GitHub ELF backend는 같은
  supervisor/observer instance를 공유한다. 같은 output endpoint를 factory마다
  unlink/rebind하지 않는다.
- fixed gVisor remote endpoint는 한 supervisor만 소유한다. MVP에서는 endpoint
  exclusive ownership을 process lock과 socket identity로 보장하고, 이미 다른
  supervisor가 소유하면 임의 연결·재사용·삭제하지 않고 fail-closed한다.
- helper readiness는 단순 sleep이 아니라 process 생존, trusted endpoint
  type/owner/mode와 readiness pipe의 explicit acknowledgement로 확인한다.
- supervisor는 protected runtime directory에서 atomic exclusive lock을 먼저
  만들고 output receiver와 remote socket 모두가 absent인 것을 확인한다. 다른
  owner의 socket/lock은 unlink하거나 재사용하지 않는다. cleanup은 생성 당시의
  inode identity가 같은 socket/lock만 제거하고, identity·stop·unlink 불확실성은
  lock을 남긴 fail-closed failure다.
- helper start/readiness/exit, receiver failure, endpoint cleanup 불확실성은 관련
  required observation을 incomplete로 만든다.
- 여러 `helox` process의 multiplexing은 M8 범위가 아니다. 향후 Host-scoped
  observer supervisor/service가 여러 process와 container/session mapping을
  중재하는 별도 확장으로 둔다.

target container에는 remote/output endpoint, helper process, Evidence Store나
supervisor control channel을 mount·전달하지 않는다.

## 3. Connection attribution and truthful observation capability

observer helper는 connection에서 처음 확인된 valid `container_id`를 identity로
latch한다. 이후 모든 event는 같은 ID여야 하며 mismatch, empty/invalid ID,
drop count, malformed payload 또는 unexpected stream end는 latched session의
incomplete fault로 전달하고 connection을 폐기한다.

M8은 synthetic test가 만들 수 있는 event와 production seccheck transport가
실제로 생성하는 event를 구분한다.

- production helper가 만들지 않는 `honeytoken-access`,
  `process-unexpected`, `filesystem-violation`을 지원 capability나 clean result로
  간주하지 않는다.
- 기존 M3 contract가 required로 선언한 detection은 bounded upstream field,
  Host-independent normalization과 실제 gVisor integration evidence가 있어야
  활성화한다.
- path, argv, process 또는 filesystem detail을 추가할 때는 size/count,
  normalization, sensitive-data suppression과 no-raw-Evidence 규칙을 먼저
  확정한다.
- required detection signal을 생성·해석할 수 없으면 check는
  unsupported/incomplete이며 Policy는 `ALLOW`하지 않는다.
- M8은 현재 주장하는 detection의 correctness를 복구한다. 더 넓고 깊은 행위
  분석은 M11에서 별도로 확장한다.

M8-004 production-emittable matrix의 active Finding은
`network-attempt` → `M3_NETWORK_ATTEMPT` 하나다. `process-exec`,
`process-clone`, `filesystem-open`은 raw payload 없이 bounded observation으로만
보존하며 current Policy Finding을 만들지 않는다. `honeytoken-access`,
`process-unexpected`, `filesystem-violation`은 generic synthetic fixture에서만
표현될 수 있고 pinned helper가 생성하지 않으므로 production capability가 아닌
`UNSUPPORTED`이며 M11 후보로 남긴다.

## 4. Least-privilege resolver network policy

일반 `helox` process 전체를 root로 실행하지 않는다. Host firewall mutation은
별도 root-owned privileged helper/service가 담당하고, 가능한 경우 권한을
`CAP_NET_ADMIN`과 필요한 filesystem/socket 접근으로 제한한다.

helper API는 HAA가 정의한 좁은 typed operation만 허용한다. package name,
hostname, shell fragment, executable path 또는 raw iptables/nft arguments를 받지
않는다. 논리적 interface는 다음 범위보다 넓어지지 않는다.

```text
CreateResolverPolicy(session, verifiedNetwork, approvedIPSet)
VerifyResolverPolicy(session)
RemoveResolverPolicy(session)
```

- Linux MVP의 concrete boundary는 root-owned `haa_network_policy_helper` system
  service와 protected local Unix stream socket이다. service configuration은
  root-owned·non-writable이며 ordinary client가 읽을 수 있는 non-secret identity
  contract다. authorized client UID, client executable identity,
  Docker endpoint와 socket group을 명시한다. socket의 filesystem permission은
  접속 후보를 좁힐 뿐이고, service는 Linux `SO_PEERCRED`와 `/proc/<pid>/exe`의
  protected identity를 각각 확인한다. 둘 중 하나라도 확인할 수 없거나 불일치하면
  request를 실행하지 않는다.
- client는 fixed local socket만 사용하고 endpoint identity·owner·mode·parent
  chain을 연결 전과 후에 확인한다. ambient proxy, remote endpoint, stdin,
  environment 또는 shell은 protocol input이 아니다. request와 response는 bounded
  typed JSON envelope 하나이며 unknown field, trailing input, oversized body와
  duplicate request를 거부한다.
- resolver client가 Docker로 만든 network는 HAA-owned label과 exact session
  label을 가져야 하고, service가 trusted local Docker endpoint에서 network ID,
  name, IPv4 subnet, no-attachment initial state와 label을 독립적으로 확인한다.
  request가 전달한 subnet/IP set만으로 policy를 만들지 않는다.
- service는 session과 peer UID를 함께 journal에 묶는다. create는 one live
  session만 만들고 apply 뒤 verify가 성공해야 acknowledgement를 반환한다.
  verify/remove는 같은 peer와 live session만 대상으로 한다. crash, EOF,
  malformed request/response, duplicate/stale session, identity change 또는
  cleanup acknowledgement 불확실성은 caller의 resolver result를 incomplete로
  만들며 ALLOW로 진행하지 않는다.
- service 내부만 fixed trusted firewall tool identity로 `iptables` 또는
  `nftables`를 호출한다. `helox` product process는 firewall executable을
  보유하거나 실행하지 않는다. service setup만 privileged이며 supported workflow는
  ordinary user의 installed `helox` binary다.

- peer credential과 설치된 client/helper identity를 검증한다.
- session identifier, HAA-owned Docker network/subnet, public approved IP set,
  TCP 443 default-deny rule만 허용한다.
- arbitrary table/chain/interface/port/command selection과 general firewall API를
  제공하지 않는다.
- create/apply/verify/remove는 typed result와 bounded journal을 남긴다. helper
  crash, duplicate session, stale rule, mapping disagreement 또는 cleanup
  uncertainty는 fail-closed다.
- controlled Linux integration은 service 설치만 privileged setup으로 수행하고,
  실제 `helox` workflow는 ordinary user identity로 검증한다.
- controlled Linux installation은 root-owned system service unit으로 helper를
  실행한다. protected configuration은 socket path, exact client executable
  identity와 authorized UID/GID만 담으며, package name·hostname·raw firewall
  argument를 받는 runtime API가 아니다.

## 5. Runtime identity single source of truth

runtime/observer lock data가 exact identity의 canonical owner다. Go constants,
CI shell과 workflow validator에 값을 사람이 반복 입력하지 않는다.

- product code와 CI는 canonical lock을 읽거나 deterministic generated output을
  소비한다.
- generated output은 source marker와 regeneration command를 가지며 drift check가
  required gate에 포함된다.
- workflow는 lock의 gVisor commit, runsc digest, Bazel identity, Docker version과
  Node/Python image digest를 사용한다.
- runtime probe는 Docker가 실제 등록한 runsc path/identity를 검증하며, PATH의
  다른 runsc version 출력만으로 daemon runtime을 신뢰하지 않는다.

### M12-001 patched observation capability and mount readiness extension

M12-001 filesystem attribution은 기존 M8 lifecycle ownership을 확장하지만 M8의
historical completion evidence를 대체하지 않는다. runtime/observer lock은 upstream
gVisor commit과 함께 exact HAA-owned filesystem-observation patch identity/digest를
소유한다. probe는 Docker가 실제 등록한 `runsc`가 그 exact patched identity이고
`OPEN_RESULT` 및 `MOUNT_TOPOLOGY_SNAPSHOT` schema/capability를 제공함을 확인해야
한다. 어느 하나라도 확인할 수 없으면 dynamic result는 clean이 아니라
`INCOMPLETE`다.

backend는 trusted profile selector로부터 expected logical mount topology를
등록한다. 이 정보는 expected mountpoint/class, required filesystem type,
read-only/security flags와 parent relation만 포함하며, gVisor-internal mount ID를
추측하지 않는다. helper는 gVisor startup topology snapshot으로부터 actual
namespace topology를 받아 exact reconciliation을 수행한다.

```text
profile and expected topology registered
→ observer/session registration
→ container start and actual topology snapshot
→ helper reconciliation and sealed anchors
→ MOUNT_ANCHORS_READY
→ ordinary boundary readiness
→ Artifact introduction and Artifact-bearing execution
```

- `MOUNT_ANCHORS_READY`는 observer process readiness와 다른 session-specific
  acknowledgement다. backend는 두 readiness가 모두 성공하기 전 Artifact를
  introduce하거나 Artifact-bearing execution을 허용하지 않는다.
- Artifact pathname, role, provenance, process identity, first observed mount ID
  또는 first filesystem event는 mount anchor를 등록하거나 변경할 수 없다.
- reconciliation 뒤 anchor table은 session 동안 sealed다. mount, bind mount,
  remount, unmount, move mount, pivot/root namespace replacement, unexpected nested
  mount 또는 다른 topology mutation은 table을 갱신하지 않고 `INCOMPLETE`다.
- capability drop, 특히 Artifact의 `CAP_SYS_ADMIN` 부재는 defense-in-depth이며
  topology completeness의 대체 증거가 아니다.

## 6. M8 work breakdown

| Order | ID | Scope |
| --- | --- | --- |
| 1 | M8-001 | production trust hardening entry·canonical contract |
| 2 | M8-002 | Trusted Host tool identity·executor·local Docker endpoint |
| 3 | M8-003 | observer supervisor·exclusive endpoint·process-scoped composition |
| 4 | M8-004 | connection identity latch·production observation completeness |
| 5 | M8-005 | least-privilege typed resolver network-policy helper |
| 6 | M8-006 | runtime lock single source of truth·CI integration |
| 7 | M8-007 | hostile-boundary regression·Linux security requalification |

## 7. M8 exit criteria

- hostile PATH, writable/symlink tool path, ambient Docker/proxy environment,
  remote daemon와 runtime identity mismatch가 trusted execution으로 이어지지
  않는다.
- one production supervisor가 helper/observer lifecycle을 소유하고 concurrent
  endpoint ownership, helper death와 cleanup uncertainty를 fail-closed 처리한다.
- connection/container/session attribution과 actual production observation
  capability가 integration evidence로 일치하며 unavailable signal이 clean
  result나 `ALLOW`가 되지 않는다.
- ordinary user workflow가 whole-process sudo 없이 narrow privileged helper를
  사용하고 arbitrary firewall operation을 수행할 수 없다.
- canonical runtime lock과 code/CI/runtime consumption 사이 drift가 required
  gate에서 차단된다.
- 전체 quick/docs/security/vulnerability/race와 pinned Linux gVisor lifecycle,
  npm/PyPI resolver, dynamic inspection, Promotion integration이 통과한다.
- M7 qualification evidence는 보존되고 M8 completion 이후에만
  production-ready release blocker를 해제할 수 있다.

## 8. Deferred milestones

- M9 — Product Install UX: transactional default npm project/active Python
  environment Promotion, canonical `pip` command와 optional advanced `--target`.
- M10 — Verified Distribution & Bootstrap: release signing identity, native
  binary/helper release, required GHCR runtime image, installer/doctor와 provenance
  verification.
- M11 — Dynamic Detection Depth: M8의 truthful minimum 위에서 bounded richer
  process/filesystem/network behavior analysis를 확장한다.
