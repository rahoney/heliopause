# M3 Entry Decision — Linux Dynamic Inspect Contract

M3는 M2의 exact npm Artifact에 Linux 전용 dynamic lifecycle inspection을 추가한다. Sandbox는 raw runtime observation과 execution status만 제공하며, `internal/inspection`이 이를 Evidence/Finding으로 정규화하고 `internal/policy`만 최종 Decision을 만든다.

## 1. Runtime identity와 지원 경계

M3 production backend는 Docker Engine의 OCI runtime integration 위에서 **gVisor `runsc` release-20260810.0**을 사용한다. exact annotated upstream commit은 `5ceb9a5fd5750d6c73dd166441f28306039300d0`이다. gVisor는 일반 container가 아닌 userspace application kernel을 제공하며 Docker와 통합되는 OCI runtime이다. Docker Engine 29.6.0은 이 결정을 확인한 시점의 최신 stable line이다. M3-002에서 binary archive SHA-512와 Docker runtime registration을 lock으로 구현한다.

- supported host: Linux x86_64 또는 arm64, kernel `>= 4.14.77`, Docker Engine `>= 29.6.0`, installed `runsc` release-20260810.0.
- unsupported host: macOS, Windows, Docker/runc-only host, rootless/cgroup capability 또는 gVisor trace capability가 없는 Linux host. 이 경우 required dynamic inspection은 `UNAVAILABLE / M3_DYNAMIC_CAPABILITY_UNAVAILABLE`이며 자동 `ALLOW`가 없다.
- M3 CI는 `ubuntu-24.04` pinned runner에서 explicit runtime probe가 성공할 때만 Linux dynamic integration job을 실행한다. probe 실패는 skipped success가 아니라 failure다.
- workload image는 `node:22.23.1-slim@sha256:6c74791e557ce11fc957704f6d4fe134a7bc8d6f5ca4403205b2966bd488f6b3` index digest로 고정한다. runtime image pull이나 package registry 상태는 unit/contract test success input이 아니다.

공식 근거: [gVisor installation](https://gvisor.dev/docs/user_guide/install/), [gVisor Docker quick start](https://gvisor.dev/docs/user_guide/quick_start/docker/), [gVisor security model](https://github.com/google/gvisor), [gVisor trace/seccheck](https://github.com/google/gvisor/blob/master/pkg/sentry/seccheck/README.md), [Docker Engine 29.6 release](https://docs.docker.com/engine/release-notes/29/), [Node 22 slim image digest](https://hub.docker.com/layers/library/node/22-slim/).

## 2. Sandbox Session contract

M3 `Sandbox Port`는 following lifecycle의 one-shot Session만 제공한다.

```text
Create → Prepare → Introduce controlled tarball → Execute npm lifecycle
      → collect raw observations → Terminate → Dispose
```

- every attempt receives a new session ID, private runtime root and fresh writable filesystem. successful, timed out, limited or failed session is never reused.
- no Host bind mount is allowed. After observer attribution, the container starts in a fixed waiting state so the image-provided `/tmp` tmpfs mount target is active; the trusted controller then streams the exact tarball once through `docker exec -i` stdin to `/tmp/artifact.tgz`, after which the fixed lifecycle command proceeds. No Host path enters container argv or environment. The image root remains read-only and `/tmp` is bounded, `noexec`, `nosuid`, and `nodev` tmpfs.
- no Host environment, home directory, Docker socket, process namespace, PID socket, secret, project or internal service is exposed.
- container creation keeps `--cap-drop ALL`, `--network none`, `no-new-privileges`, a read-only root filesystem and the existing pids/memory/CPU/tmpfs limits. A narrow HAA-owned pre-readiness bootstrap may hold only `CAP_SETUID`, `CAP_SETGID` and `CAP_SETPCAP` while atomically installing its immutable boundary helper; before readiness, PID1 irreversibly drops to uid/gid 1000 with empty supplementary groups, all five capability sets zero and `NoNewPrivs=1`. Each later Docker exec enters only the fixed root-owned boundary helper with that same narrow bootstrap set; the helper irreversibly demotes before its requested target's command bytes run. Every requested dynamic target therefore reaches the same non-root, capability-free state. The Sandbox never receives an arbitrary Host command API. D-012/D-015 own the helper identity, provenance and minimum-privilege requirements.
- separate trusted observer socket receives gVisor trace records. Artifact processes cannot write Evidence Store, observer records or Policy state.

### Seccheck remote observer transport

M3의 canonical observation transport는 gVisor seccheck `remote` sink뿐이다. HAA trusted observer가 보호된 shared Unix-domain `SOCK_SEQPACKET` socket을 생성·listen하고, Docker에 설치한 `runsc-trace` runtime의 고정 `--pod-init-config` trace session이 그 endpoint에 접속한다. 이는 gVisor가 문서화한 Docker runtime 설치 경로이며, trace session은 Sandbox start 전에 구성하고 remote sink setup 오류를 무시하지 않는다.

- gVisor가 정의한 handshake, header, protobuf payload와 protocol version을 그대로 사용한다. HAA는 자체 message framing을 만들거나 raw payload를 Evidence/CLI에 기록하지 않는다.
- observer는 Sentry input을 untrusted로 취급하고 protocol/version·길이·drop count·event bound를 검증한다. 필요한 trace point에는 `container_id` context field를 활성화하고, connection·container ID와 HAA가 생성한 Docker container/Sandbox Session mapping이 정확히 하나로 일치할 때만 Observation을 귀속한다. 연결 실패, handshake/protocol 오류, stream 종료·손실, remote dropped event, mapping 불일치·중복·미확정 또는 필수 trace session 구성 실패는 `INCOMPLETE`다.
- 하나의 UDS를 여러 Sandbox가 공유해도 observer runtime state, Observation, Evidence와 결과는 container/Sandbox Session별로 완전히 분리한다. observer socket은 Artifact container에 mount·전달·노출하지 않으며, Evidence Store·controller socket과 별도 trusted runtime directory에 둔다.
- run별 동적 endpoint와 direct `runsc` OCI bundle은 MVP 범위에 넣지 않는다.
- `--strace`, stdout/stderr scraping, Host 일반 파일 로그는 canonical observation transport 또는 fallback이 아니다.

공식 근거: [gVisor seccheck](https://github.com/google/gvisor/blob/release-20260810.0/pkg/sentry/seccheck/README.md), [remote sink protocol](https://github.com/google/gvisor/blob/release-20260810.0/pkg/sentry/seccheck/sinks/remote/README.md).

### M12-001 authoritative filesystem transport extension

M12-001은 M3의 profile별 filesystem policy를 바꾸지 않는다. 다만 relative
`openat`을 pathname text, `cwd`, `fd_path` 또는 descriptor 재구성으로 추측하지
않기 위해, dynamic runtime capability에 다음 patched-gVisor observation contract를
추가한다.

- supported runtime identity는 exact upstream gVisor commit뿐 아니라 그 commit에
  적용된 exact HAA-owned observation patch identity/digest를 함께 lock해야 한다.
  unpatched, wrong-patch, partially patched 또는 required point/schema가 없는
  `runsc`는 required dynamic inspection을 `INCOMPLETE`로 만든다.
- existing `syscall/openat/enter`는 attempt telemetry로 남는다. patched runtime은
  모든 relevant open invocation마다 정확히 한 번 `OPEN_RESULT`를 방출한다.
  failure는 errno만 가지며 nonexistent target의 filesystem class를 추측하지 않는다.
  success는 actual VFS resolution과 FileDescription creation/FD installation 뒤,
  held `FileDescription.VirtualDentry()`, actual Mount와 namespace-root-relative
  reachable pathname에서 얻은 final-object identity를 가진다.
- patched runtime은 Artifact execution 전에 정확히 한 번의 bounded atomic
  `MOUNT_TOPOLOGY_SNAPSHOT`을 방출한다. snapshot은 mount namespace ID, mount ID,
  parent mount ID, namespace-root-relative mountpoint, filesystem type, read-only
  state 및 필요한 `noexec`/`nosuid`/`nodev` flags를 포함한다. mount count는 64,
  mountpoint는 512 bytes, filesystem type은 32 bytes, encoded snapshot은 64 KiB를
  넘지 않는다. existing project-wide bound가 더 엄격하면 그것을 사용한다.
- snapshot은 explicit complete marker, exactly one root와 valid acyclic parent
  graph를 가져야 한다. duplicate, truncated, overflowed 또는 malformed snapshot,
  required result의 missing/malformed event, remote drop, stream fault 또는
  post-ready topology mutation notification은 `INCOMPLETE`다.
- remote sink framing 자체는 계속 upstream protocol을 사용한다. HAA는 separate
  framing이나 target-container `/proc` parsing을 authoritative topology source로
  사용하지 않는다.

## 3. Fixed limits and execution plan

| Boundary | M3 value |
| --- | --- |
| lifecycle command | `npm install --ignore-scripts=false --no-audit --no-fund --offline` from the controlled tarball only |
| wall timeout | 45 s |
| graceful termination | 3 s, then forced session termination and disposal |
| CPU | 1 CPU, 30 CPU-s |
| memory | 512 MiB |
| PID count | 64 |
| writable tmpfs | 256 MiB |
| stdout/stderr raw capture | 256 KiB each, never normal Evidence/output |
| trace event capture | 10,000 events / 2 MiB normalized input; overflow is `INCOMPLETE` |
| network | `none`; a later controlled fake DNS/HTTP network requires a distinct entry decision |

M3 runs npm lifecycle installation only. post-install application invocation, external dependency resolution, real registry/network access, Promotion and Host installation are out of scope.

## 4. Raw observation and normalized interpretation

The backend records bounded raw facts, not findings. M8-004 기준 production gVisor
remote helper가 실제 emit하는 범위는 session lifecycle, process exec/clone, file
open, network capability와 communication attempt다. Trace is enabled at Session
initialization so pre-observer events cannot be missed. Raw path, argv,
environment, file contents, output and trace payload never enter human result
or normal Evidence.

The inspector's production-emittable capability matrix is below. Synthetic
Domain/Policy fixtures may still exercise generic finding rules, but they do not
prove that the pinned helper provides those signals and must not change this
matrix or make an unsupported capability appear clean.

| Observation | Normalized Evidence / Finding |
| --- | --- |
| ordinary AF_INET/AF_INET6 socket creation | bounded raw capability observation only; it is not by itself `M3_NETWORK_ATTEMPT` |
| AF_PACKET socket creation | conservative `M3_NETWORK_ATTEMPT` MANUAL_REVIEW because raw-packet/network capability is security-relevant |
| `connect`, `sendto`, `sendmsg`, `sendmmsg` communication attempt | `M3_NETWORK_ATTEMPT` MANUAL_REVIEW |
| exact pinned HAA one-shot direct control root network operation | bounded `TRUSTED_CONTROL_NETWORK` raw observation only when exact control-root attribution is confirmed; it is not an artifact network Finding |
| network FD-family/event attribution unknown, malformed, dropped or unclassifiable | required dynamic check `INCOMPLETE` / no ALLOW |
| exec ENTER/EXIT attempt; file open; process clone | bounded raw telemetry only |
| valid Sentry unexpected executable-image transition boundary | `M3_UNEXPECTED_PROCESS` MANUAL_REVIEW |
| read/open synthetic honeytoken | `UNSUPPORTED` in production helper; M11 candidate |
| write outside the bounded `/tmp` workspace or excessive file operation | `UNSUPPORTED` in production helper; M11 candidate |
| timeout/resource/event-limit, helper drop/malformed/mismatched stream or observer failure | required dynamic check `INCOMPLETE` / no ALLOW |
| clean completed observation | dynamic check `COMPLETED`; Policy may allow only after all M3 required checks are complete |

`syscall/execve` and `syscall/execveat` ENTER/EXIT are execution-attempt
telemetry only; they do not alone prove an executable-image transition. In
pinned gVisor, syscall EXIT is not final exec-success evidence because the exec
continuation may run after that event. The `sentry/execve` checkpoint is emitted
only after `LoadTaskImage` succeeds and is the authoritative executable-image
transition boundary for M3 observation. A valid bounded Sentry observation may
produce `process-exec-expected` or `process-exec-unexpected` without a retained
matching ENTER; this does not claim final process-image commit or userspace
execution. Failed pathname lookup without Sentry, including explicit syscall
failure, remains bounded attempt telemetry. Malformed or invalid Sentry
observation and observer stream-integrity failure are `INCOMPLETE`.

Process attribution separates execution role from root provenance. The bounded
role state is `UNKNOWN → CONTROL → ARTIFACT`; `ARTIFACT` is irreversible.
Provenance is separately `OCI_ROOT`, `DIRECT_EXEC_ROOT` or `CLONE_CHILD`, and
root eligibility is tracked and consumed separately.

The OCI root is established from exact `container/start` ContextData
(`container_id`, thread-group ID and thread-group start time) and begins in
`CONTROL`. Every HAA Docker exec begins through the verified HAA direct-exec
launcher, which immediately execs its target. Complete `sentry/clone`
provenance is required before accepting the first valid Sentry transition for
an untracked direct-exec group as its launch boundary. Clone-created groups
are permanently ineligible to become direct roots, including `CLONE_PARENT`
children whose reported parent group ID may be zero. `is_exec_session` or a
zero parent group ID alone never creates control trust, and the target
pathname is not a trust anchor.

A valid direct root consumes root eligibility exactly once. Same-group re-exec,
any child group, same-path execution, lifecycle descendant or unknown
attribution does not regain or inherit root eligibility. A CONTROL child may
inherit only bounded control-role context; an ARTIFACT child remains
ARTIFACT.

The verified HAA handoff executable is recognized only as a trust-removal
marker. It performs `CONTROL → ARTIFACT` (or leaves `ARTIFACT` unchanged) and
has no reverse transition. Python package import passes this handoff before
`importlib.import_module`; npm lifecycle execution uses it as its
`script-shell` trampoline; and GitHub ELF execution passes through it before
`/work/artifact`. Artifact-controlled network operations and executable
transitions remain actionable after handoff. No pathname, process class,
child ancestry or handoff marker can create or restore CONTROL trust.
After a direct root has consumed launch eligibility and established its valid
CONTROL lifecycle, that exact group may still execute the verified handoff:
handoff removes trust and does not consume, grant or recreate root eligibility.
A first valid direct-root Sentry may also be that handoff; it records bounded
root provenance while immediately consuming eligibility into ARTIFACT without
ever activating CONTROL-target trust.

The same non-inheritance rule applies to `TRUSTED_CONTROL_NETWORK`: it is
available only while the exact pinned/verified one-shot DirectExecRoot launch
target is currently active in CONTROL. It is false for the OCI root, boundary
marker, capability-demotion transition, clone/lifecycle child, descendant,
handoff, ARTIFACT role, re-exec, same-path execution and unknown/missing
correlation; it cannot be restored. Artifact-controlled communication attempts remain
`M3_NETWORK_ATTEMPT`. D-012/D-015 in
[Trusted Tooling and Evidence](../threat-model/04-trusted-tooling-and-evidence.md)
own the trusted-tool compromise scope; this M3 contract owns only observation
interpretation.

## 5. M3 Policy v3 direction

M3 Policy identity is `m3-npm-dynamic-inspect`, version 1.

1. integrity mismatch or blocking static/dynamic Finding → `BLOCK`.
2. required static, integrity or dynamic check unsupported/unavailable/failed/incomplete → `MANUAL_REVIEW / M3_REQUIRED_CHECK_INCOMPLETE`.
3. completed clean static and dynamic checks → `ALLOW / M3_REQUIRED_CHECKS_COMPLETED`.
4. suspicious dynamic behavior with no blocking finding → `MANUAL_REVIEW` with its normalized reason.

This does not change M2 results retroactively. M3 Policy is only wired when the M3 backend capability is available.
`M3_NETWORK_ATTEMPT` and `M3_UNEXPECTED_PROCESS` remain `MANUAL_REVIEW` reasons;
this contract changes only the raw-observation-to-Finding and attribution
conditions that may produce them.

## 6. Fixtures, retention and exclusions

- fixture packages are generated tarballs only: clean lifecycle, honeytoken access, network attempt, unexpected process, timeout/resource limit, and fresh retry after abnormal termination.
- fake credential/honeytoken is a non-secret fixed sentinel inside the Sandbox only; it is never committed as a realistic credential.
- raw observation is retained only until normalized Evidence writing finishes, then securely discarded with the session; bounded summaries and store-computed record digest follow the existing Evidence Store policy.
- gVisor runtime installation, image pull, runtime root and cleanup are M3-002 implementation concerns. No M3 package, CI job or runtime config is created by this entry decision.
