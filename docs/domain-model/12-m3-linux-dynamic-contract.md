# M3 Entry Decision — Linux Dynamic Inspect Contract

M3는 M2의 exact npm Artifact에 Linux 전용 dynamic lifecycle inspection을 추가한다. Sandbox는 raw runtime observation과 execution status만 제공하며, `internal/inspection`이 이를 Evidence/Finding으로 정규화하고 `internal/policy`만 최종 Decision을 만든다.

## 1. Runtime identity와 지원 경계

Production backend는 Docker Engine의 OCI runtime integration 위에서 canonical `scripts/runtimes.lock.json`의 gVisor source와 exact HAA patch/build-input·installed-byte identity를 사용한다. 현재 qualification은 `release-20260907.0` / `7c6199801fd233d6d55309af4645d4746a077de7`, Docker29.8.1이다. gVisor는 userspace application kernel을 제공하며 Docker와 통합되는 OCI runtime이다. M3 당시 `release-20260810.0` / `5ceb9a5fd5750d6c73dd166441f28306039300d0`과 Docker29.6.x 및 stock upstream SHA-512 결과는 역사적 qualification 범위다. Distributed patched artifact의 architecture별 final digest와 배포 계약은 M10/M13이 소유한다.

- 현재 qualified Host: Linux amd64와 canonical lock의 Docker/runsc/observer identity·필수 cgroup/trace capability. 다른 architecture의 registration이나 source build만으로 동적 지원을 주장하지 않는다. Public installer/Host activation은 M13의 별도 acceptance다.
- unsupported host: macOS, Windows, Docker/runc-only host, rootless/cgroup capability 또는 gVisor trace capability가 없는 Linux host. 이 경우 required dynamic inspection은 `UNAVAILABLE / M3_DYNAMIC_CAPABILITY_UNAVAILABLE`이며 자동 `ALLOW`가 없다.
- M3 CI는 `ubuntu-24.04` pinned runner에서 explicit runtime probe가 성공할 때만 Linux dynamic integration job을 실행한다. probe 실패는 skipped success가 아니라 failure다.
- workload image는 canonical runtime lock의 `node_image.reference`와 bundled npm identity로 고정한다. 현재 Node24.21.0/npm11.19.0이다. Runtime image pull이나 package registry 상태는 unit/contract test success input이 아니다.

공식 근거: [gVisor installation](https://gvisor.dev/docs/user_guide/install/), [gVisor Docker quick start](https://gvisor.dev/docs/user_guide/quick_start/docker/), [gVisor security model](https://github.com/google/gvisor), [locked gVisor trace/seccheck](https://github.com/google/gvisor/blob/7c6199801fd233d6d55309af4645d4746a077de7/pkg/sentry/seccheck/README.md), [Docker Engine 29 release notes](https://docs.docker.com/engine/release-notes/29/).

## 2. Sandbox Session contract

M3 `Sandbox Port`는 following lifecycle의 one-shot Session만 제공한다.

```text
Create → Prepare → Introduce controlled tarball → Execute npm lifecycle
      → collect raw observations → Terminate → Dispose
```

- every attempt receives a new session ID, private runtime root and fresh writable filesystem. successful, timed out, limited or failed session is never reused.
- no Host bind mount is allowed. After observer attribution, the container starts in a fixed waiting state so the image-provided `/tmp` tmpfs mount target is active; the trusted controller then streams the exact tarball once through `docker exec -i` stdin to `/tmp/artifact.tgz`, after which the fixed lifecycle command proceeds. No Host path enters container argv or environment. The image root remains read-only and `/tmp` is bounded, `noexec`, `nosuid`, and `nodev` tmpfs.
- M5 Python closure observation may mount only its controller-created, transaction-bound Docker `local` tmpfs volume at `/haa-site`. The preparation runtime may attach it read-write; the artifact-free anchor and every observation runtime must attach the same attested volume read-only with `volume-nocopy`. Docker volume identity and attachment inventory, exact container/transaction identity, and the gVisor guest mount topology must all agree. This exception does not permit any Host bind mount, arbitrary Docker volume, alternate destination, writable observation attachment, or unaccounted writer.
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
`/work/artifact`. Artifact-controlled network operations and unmodeled executable
transitions remain actionable after handoff. The bounded Python library-query
operation below is modeled without changing ARTIFACT role or network attribution. No pathname, process class,
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

### 고정 Python runtime의 bounded library query (2026-10-02)

사용자 승인에 따른 좁은 정책 정교화다. ARTIFACT의 일반 실행 전이는 계속
`M3_UNEXPECTED_PROCESS` 대상이지만, 다음 조건을 모두 만족한 `ldconfig -p`는
`process-exec-expected`로 관찰한다. Expected operation은 CONTROL 역할이 아니다.

- 기존 host attestation이 고정 Python image identity를 확인하고, guest topology가
  같은 session의 read-only OCI root와 mount namespace를 seal해야 한다.
- successful Sentry image-load의 kernel-resolved binary는 `/usr/sbin/ldconfig`
  또는 고정 image의 `/sbin/ldconfig` 별칭이어야 한다. Binary와 `/etc/ld.so.cache`
  아래의 다른 mount는 지원하지 않는다. Pathname/comm/argv0 주장만으로 인정하지 않는다.
- execfn/argv0는 위 별칭, argv는 정확히 두 항목(실행 파일, `-p`), env는 정확히
  `LC_ALL=C`와 `LANG=C`다. 추가 cache/root/config/loader/locale/search-path 입력은 거부한다.
- 고정 image의 ldconfig는 ELF interpreter/DT_NEEDED가 없는 static PIE다.
  조회 중 실제 `/etc/ld.so.cache`의 read-only OCI-root open만 runtime read로 처리한다.
  다른 접근, write, exec, network에는 기존 규칙을 적용한다.
- ARTIFACT 역할, consumed root eligibility, 모든 events/counts, unit ledger,
  cumulative CPU/memory/observer limits와 cleanup 요구는 그대로다. 조회 상태는
  exec마다 초기화하며 child에게 상속하지 않는다. 출력은 package 데이터이며
  완료나 신뢰 증거가 아니다. 다른 필수 checks와 Policy 판정이 계속 필요하다.

이는 pinned CPython `ctypes.util.find_library`의 첫 ldconfig 조회만 지원한다.
`gcc`, `objdump`, `ld` fallback은 지원에 포함되지 않는다. Image/mount/관찰 증거가
없거나 일치하지 않으면 비적격이다. 일반 PyPI의 같은 Python runtime에도 동일하게
적용하며 npm/GitHub ELF의 동작 허용 범위는 변경하지 않는다.

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
