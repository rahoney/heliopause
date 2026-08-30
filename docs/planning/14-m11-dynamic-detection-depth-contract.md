# M11 Entry Decision — Dynamic Detection Depth

M11은 M8에서 truthfulness와 fail-closed completeness를 확보한 observer boundary
위에서만 bounded behavior detection을 추가한다. raw gVisor protobuf, arbitrary
argv/path/content, Host log와 target-controlled data는 Core·Application·Policy·CLI와
일반 Evidence로 넘어가지 않는다.

## 1. Exact upstream schema boundary

M11의 sole upstream schema owner는 `scripts/runtimes.lock.json`의 gVisor
`release-20260810.0` / commit `5ceb9a5fd5750d6c73dd166441f28306039300d0`이다.
helper, runsc와 test source checkout은 이 identity를 함께 사용한다.

M3/M11 dynamic inspection image의 base/derived digest, 고정 helper content와
build provenance/signature는 [Trusted Tooling and Evidence](../threat-model/04-trusted-tooling-and-evidence.md)의
D-012/D-015 owner contract로 lock·검증한다. runtime helper 설치, writable helper
mount와 persistent UID 0 bootstrap은 허용하지 않는다.

- gVisor seccheck의 remote sink protobuf framing/protocol은 그대로 사용한다.
- `runsc trace metadata`의 human-readable output은 upstream이 안정성을 보장하지
  않으므로 schema lock의 canonical source가 아니다.
- M11-002는 pinned source를 Bazel로 build한 helper contract test에서 point,
  message type, field accessor와 pod-init JSON을 함께 검증한다. 존재하지 않는
  point/field 또는 protobuf parse mismatch는 capability unavailable이며 ALLOW하지
  않는다.
- `ignore_missing`은 사용하지 않는다. 누락을 숨기면 required behavior가 clean으로
  보일 수 있기 때문이다.
- remote sink setup failure, handshake/version mismatch, dropped count, malformed
  event, identity mismatch, stream end/fault 또는 summary mismatch는 required
  observation incomplete다.

공식 gVisor seccheck 문서는 points마다 optional/context field가 있으며, remote
sink가 protobuf를 UDS로 보낸다고 설명한다. init-time `--pod-init-config` session은
초기 event 누락을 막는 공식 방식이다. M11은 이 M3/M8 transport 결정을 바꾸지
않는다.

## 2. M11 field profile

M11-002가 exact pinned source에서 compile-time으로 확인할 requested profile은
다음이다. logical field 명칭은 이 문서가 소유하며 C++ protobuf type/accessor는
upstream exact source test가 소유한다.

| Trace point | Required context | Required point data | Retained outside helper |
| --- | --- | --- | --- |
| `container/start` | `container_id`, `group_id`, `thread_group_start_time`, `parent_thread_group_id`, `is_exec_session` | none | exact OCI-root identity and session attribution only |
| `sentry/clone` | `container_id`, `group_id`, `thread_group_start_time`, `parent_thread_group_id`, `is_exec_session` | created group ID/start and creator context | bounded process provenance/role only |
| `sentry/execve` | `container_id`, `group_id`, `process_name` | exec identity after `LoadTaskImage` succeeds | authoritative image-load boundary; bounded context validation only |
| `syscall/execve` and `syscall/execveat` enter/exit | `container_id`, `group_id`, `process_name` | pathname, argv, bounded attempt/exit telemetry | telemetry only; EXIT is not final success evidence |
| `syscall/openat/enter` | `container_id`, `group_id`, `process_name` | pathname, flags, mode, sysno | workspace/honeytoken/outside class only |
| `syscall/socket` enter/exit; close/dup/fcntl and clone/exec lifecycle | `container_id`, `group_id`, `process_name` | bounded socket FD-family lifecycle state | helper-only bounded correlation; unknown state is incomplete |
| `syscall/connect/enter` | `container_id`, `group_id`, `process_name` | socket family | bounded communication-attempt class only |
| raw `sendto`/`sendmsg`/`sendmmsg` enter where applicable | `container_id`, `group_id`, `process_name` | socket FD only | helper-only FD-family lookup; no payload/destination retention |

`fd_path`, `cwd`, `credentials`, `env`, read/write data, raw socket address와 raw
process identifier are intentionally excluded. `pathname`과 `argv`는 helper가
transiently classifying할 때만 읽을 수 있으며 output envelope, Domain object,
Evidence, policy reason, test golden, stdout/stderr에 보존·로그·전달하지 않는다.

## 3. Normalized detection contract

helper는 raw data를 HAA boundary에 보내지 않고 per-session bounded kinds와 aggregate
summary만 전송한다. M11-002가 추가할 normalized vocabulary는 다음으로 고정한다.

| Category | Normalized subject | Meaning |
| --- | --- | --- |
| PROCESS | execution-attempt telemetry | syscall exec enter; bounded only and never directly a Finding |
| PROCESS | `process-exec-expected` | exact trusted launch/control transition at the valid Sentry image-load boundary |
| PROCESS | `process-exec-unexpected` | unexpected executable-image transition at the valid Sentry image-load boundary; pathname alone does not grant trust |
| FILESYSTEM | `filesystem-workspace-access` | approved transient workspace 안의 access |
| FILESYSTEM | `filesystem-outside-workspace` | approved workspace 밖 access |
| HONEYTOKEN | `honeytoken-access` | HAA-controlled honeytoken class access |
| NETWORK | socket capability telemetry | ordinary AF_INET/AF_INET6 socket creation; bounded raw observation only |
| NETWORK | `trusted-control-network` | exact pinned/verified one-shot HAA direct control root의 bounded raw network operation; artifact Finding이 아님 |
| NETWORK | `network-attempt` | `connect`, `sendto`, `sendmsg`, `sendmmsg` communication attempt, and conservative AF_PACKET socket creation; destination is retained하지 않음 |

expected process allowlist와 workspace/honeytoken class map은 artifact가 제공하는
input이 아니라 trusted backend가 exact sandbox command, image, mount와 controlled
honeytoken placement로 생성한다. unknown path, unsupported syscall semantics,
ambiguous normalization 또는 class map mismatch는 `process-exec-unexpected` 또는
required observation incomplete로 처리한다. generic synthetic fixture만으로 위
production kinds를 주장할 수 없다.

ordinary INET socket capability does not itself become `network-attempt`.
AF_PACKET socket creation remains a conservative actionable exception. Unknown,
malformed, dropped, unclassifiable FD-family state, or missing required network
correlation is fail-closed `INCOMPLETE` rather than a clean observation.

exec ENTER/EXIT is not successful execution. In pinned gVisor, syscall EXIT is
not final exec-success evidence: the exec continuation may execute after the
syscall EXIT event. `sentry/execve` is emitted after `LoadTaskImage` succeeds
and is the authoritative executable-image transition boundary for a process
Finding. A valid bounded Sentry observation may produce
`process-exec-expected` or `process-exec-unexpected` without a retained matching
ENTER, without claiming final process-image commit or userspace execution.
Failed pathname lookup without Sentry, including explicit syscall failure,
remains bounded telemetry. Malformed or invalid Sentry observation and observer
stream-integrity failure are `INCOMPLETE`.

Role and provenance are separate. `UNKNOWN → CONTROL → ARTIFACT` is the
monotonic role state; `OCI_ROOT`, `DIRECT_EXEC_ROOT` and `CLONE_CHILD` are
provenance categories, and root eligibility is consumed separately. The exact
OCI root is seeded from `container/start`. Each HAA Docker exec uses the
verified direct-exec launcher, and its first valid Sentry transition is
eligible to establish one direct root only when complete clone provenance
shows that the group was not created by a clone. `OriginExec` and
`parent_thread_group_id == 0` alone are insufficient. Clone-created groups,
including `CLONE_PARENT` children, are permanently root-ineligible; a
same-group re-exec cannot reacquire root eligibility.

The verified HAA handoff executable is a one-way trust-removal marker:
`CONTROL → ARTIFACT`, or `ARTIFACT → ARTIFACT`. It never grants trust and
cannot restore CONTROL. Python import, npm lifecycle script-shell execution,
and GitHub ELF execution cross this marker before artifact-controlled code
runs. Artifact descendants and re-execs remain Artifact-controlled regardless
of pathname.

### M11-003 trusted profile attribution

raw `pathname`/`argv`를 Go process나 일반 HAA envelope로 보내지 않는다. 대신
helper binary 안에는 `npm-lifecycle`, `pypi-wheel`, `github-elf`별로 audit되는
고정 process/runtime/workspace/honeytoken profile table만 둔다. 각 Linux backend는
Docker가 만든 container ID에 대해 그 backend가 선택한 **profile selector만** trusted
helper control socket으로 등록한 뒤 Sandbox를 start한다. selector, container ID,
profile 등록 순서는 HAA control-plane이 만들며 Artifact input, OCI metadata 또는
container environment에서 유도하지 않는다.

- control socket은 remote seccheck sink와 별도인 protected Host-only UDS이며 target
  container에 mount·environment·network로 전달하지 않는다. 이 control IPC는
  gVisor remote protobuf framing의 대체가 아니며 event transport는 여전히 upstream
  protocol만 사용한다.
- helper는 connection의 첫 `container/start` 전에 exact one profile registration이
  있어야 한다. missing, malformed, duplicate, unknown selector, container/profile
  mismatch 또는 control channel failure는 stream fault/incomplete로 처리한다.
- profile table의 raw path/command 값은 helper build source에만 존재한다. helper는
  comparison 후 normalized kind만 output하고, profile 값·matched pathname·argv를
  Go Domain, Evidence, CLI, log 또는 test golden에 내보내지 않는다.
- profile 내부에서는 Honeytoken match를 먼저 처리하고, pinned runtime read-only
  roots는 actionable workspace violation으로 오인하지 않는다. profile이 어떤
  pathname class도 확정할 수 없으면 normal observation을 생략하지 않고 incomplete로
  처리한다.

`trusted-control-network` is equally narrow: it is active only for the exact
pinned/verified one-shot HAA direct-control target after its first accepted
launch target transition. OCI root, the launch marker, lifecycle children,
descendants, clone/control children, same-group re-exec, same-path execution
and unknown or missing attribution never receive or regain it. The active state
is cleared before a later successful executable transition and at handoff;
their `connect`/send communication attempts remain `network-attempt`. The
trusted-tool compromise scope belongs to
D-012/D-015 in [Trusted Tooling and Evidence](../threat-model/04-trusted-tooling-and-evidence.md);
this document does not suppress trusted-tool behavior broadly.

The C++ normalized producer and Go fixed decoder are one finite helper
envelope: every emitted relation, class, reason and parent category must be a
canonical value accepted by the decoder. Unknown or mismatched values are
stream-integrity failure, not a reason to widen the decoder indiscriminately;
decoder rejection must preserve a bounded fault reason rather than silently
becoming a blank diagnostic.

`process-exec-expected`와 `filesystem-workspace-access`는 Evidence summary의 count로
집계할 수 있지만, actionable Finding은 `process-exec-unexpected`,
`filesystem-outside-workspace`, `honeytoken-access`, `network-attempt`에만 만든다.
M11-003은 각 source adapter의 actual gVisor integration evidence가 확인된 뒤
Finding을 활성화했으며, M11-005가 cross-ecosystem qualification을 마무리한다.
`network-attempt`와 `process-exec-unexpected`의 existing M3/M5 policy mapping은
변경하지 않는다. 이 contract는 raw observation, successful-transition proof 및
trusted attribution의 조건만 정의한다.

## 4. Privacy, bounds and Evidence

- helper input message, argv element, pathname 및 output datagram은 per-event,
  per-session count/byte bound를 가진다. bound 초과, truncation 또는 aggregate
  overflow는 incomplete다.
- HAA는 session별 normalized unique subject와 saturating count만 retain한다. raw
  path, argv, environment, credential, file content, network destination, Host path,
  PID/UID/GID는 retain하지 않는다.
- summary에는 schema revision, session/container attribution, normalized count,
  dropped/fault/completeness 상태만 포함한다. raw trace artifact나 replay input은
  만들지 않는다.
- retention은 existing Evidence lifecycle을 따르며 M11-004가 summary schema,
  maximum unique subjects/count와 deletion regression을 canonical test로 고정한다.

## 5. Work breakdown and acceptance

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M11-001 | field profile·normalization·privacy/fail-closed entry decision | COMPLETE |
| 2 | M11-002 | pinned helper/parser와 bounded observation normalization | COMPLETE |
| 3 | M11-003 | production Finding/policy wiring과 honeytoken/workspace fixtures | COMPLETE |
| 4 | M11-004 | Evidence summary/retention·hostile regression | COMPLETE |
| 5 | M11-005 | actual gVisor integration·cross-ecosystem qualification·M12 handoff | COMPLETE |

M11 is complete only when every active detection has an exact pinned-source schema
test, production gVisor integration fixture and fail-closed regression. Raw payload
non-retention, stream completeness and M8/M9 boundaries must be independently
checked before M12 production activation.

Status: COMPLETE
Evidence: pinned M3/M8 observer transport; gVisor seccheck remote-sink and pod-init-config documentation; M11-001 field/normalization/privacy contract; M11-002 exact-source Bazel helper contract; M11-003 helper latch and Linux lifecycle/Required CI run 32836463839; M11-004 bounded summary, retention deletion and hostile regression with Required CI run 32919553761; M11-005 npm/PyPI/GitHub cross-ecosystem summary/Finding regression and actual gVisor qualification with Required CI run 32920407746
Blocker: none for M11.
Next: M12-001 — release environment·tag policy·Ruleset activation
