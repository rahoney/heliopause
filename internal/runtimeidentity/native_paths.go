package runtimeidentity

// NativeInstallRoot is the existing protected native release location. Runtime
// data consumers and the installer share its path without sharing activation.
const NativeInstallRoot = "/usr/lib/heliopause"

const PNPMNativeAssetName = "pnpm-linux-" + PNPMArchitecture

const BunNativeAssetName = "bun-linux-" + BunArchitecture

const PipxRuntimeAssetName = "pipx-runtime-linux-" + PipxArchitecture + ".zip"
