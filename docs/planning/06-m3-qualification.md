# M3 Qualification — Linux Dynamic Inspect

이 문서는 M3의 Docker/gVisor dynamic inspection과 seccheck observer 경계를 자격 검토하는 completion record다. M4 설치·Promotion을 시작하지 않는다.

## Qualification target

- Branch: `milestone/m2-npm-static-inspect`
- Qualified head: `c4ab899`; GitHub Actions run `31694083568`
- Runtime: gVisor `release-20260810.0` (`5ceb9a5fd5750d6c73dd166441f28306039300d0`), Docker Engine `29.6.2`, containerd `2.3.3`, fixed `node:22.23.1-slim` digest
- Observer helper: exact gVisor source checkout inside Bazel build boundary; product Go module 및 minimum Go `1.25.12` gate에 dependency가 없음
- Linux CI helper toolchain: locked Bazel `8.3.1`; helper가 요구하는 gVisor toolchain은 product toolchain과 별도

## Exit criteria audit

| Criterion | Evidence | Result |
| --- | --- | --- |
| Linux-only fail closed | macOS/unsupported/runtime probe failure가 `M3_DYNAMIC_CAPABILITY_*` incomplete 결과로 정규화되어 `ALLOW`가 될 수 없음 | PASS |
| Docker gVisor lifecycle | per-session `docker create` with `runsc-trace`, read-only/non-root/network-none/resource limits, trusted stdin artifact stream to bounded tmpfs, start·collection·remove lifecycle | PASS |
| canonical transport | fixed `--pod-init-config` remote sink, protected shared `SOCK_SEQPACKET`, upstream handshake/header/protobuf only | PASS |
| per-session attribution | required `container_id` context + exact Docker container mapping; duplicate/unconfirmed/malformed/drop/stream end failure is incomplete | PASS |
| trusted boundary | Artifact receives neither host path nor observer/Evidence/controller socket; normalized bounded records only cross helper→product IPC | PASS |
| normalized Policy boundary | raw protocol payload, path, argv, environment and stdout/stderr do not enter Finding/Evidence/CLI; incomplete·violation·network·unexpected-process rules are deterministic | PASS |
| quality and integration | canonical quick/race/docs/platform; Bazel helper build; Linux Docker/runsc lifecycle CI | PASS |

## Reproducible checks

- `go run ./scripts/check quick`
- `go run ./scripts/check docs`
- `go run ./scripts/check platform`
- `go test -race -timeout=5m ./...`
- GitHub Actions run `31694083568`: `Docs`, `Quick`, `Minimum Go`, `macOS`, `gVisor Observer Helper`, `gVisor Linux Integration`, `Required` success

## Limitations and handoff

- macOS and unsupported Linux hosts return required dynamic inspection incomplete; they never bypass M3 Policy.
- The trusted helper is a separately installed Linux infrastructure component. Its fixed `runsc-trace` runtime configuration and protected endpoint must exist before a Sandbox starts; unavailable helper/endpoint/trace transport is incomplete.
- Docker direct runtime integration remains the MVP path. Direct `runsc` OCI bundle ownership is intentionally out of scope.
- M4 opens exactly one entry decision: npm install and trusted Promotion. No M4 source, installer, staging or CI placeholder is introduced by this qualification.
