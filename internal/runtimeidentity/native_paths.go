package runtimeidentity

// NativeInstallRoot is the existing protected native release location. Runtime
// data consumers and the installer share its path without sharing activation.
const NativeInstallRoot = "/usr/lib/heliopause"

const PNPMNativeAssetName = "pnpm-linux-" + PNPMArchitecture

const BunNativeAssetName = "bun-linux-" + BunArchitecture

const PipxRuntimeAssetName = "pipx-runtime-linux-" + PipxArchitecture + ".zip"

const SandboxResponderAssetName = "haa-sandbox-responder"

const SandboxResponderMaxBytes = (4 << 20) - (64 << 10)

// Behavior changes require a new inspection revision. Compiler updates follow
// the existing Go owner; exact supplied bytes also bind each fixture digest.
const SandboxResponderInspectionIdentity = "loopback-dns-http-v1|go" + GoVersion
