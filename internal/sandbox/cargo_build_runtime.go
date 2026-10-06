package sandbox

import (
	"errors"
	"time"
)

const (
	cargoBuildTimeout      = 3 * time.Minute
	cargoBuildProfile      = "cargo-build"
	cargoBuildGuestInput   = "/tmp/haa-cargo-input"
	cargoBuildGuestProject = cargoBuildGuestInput + "/project"
	cargoBuildGuestVendor  = cargoBuildGuestInput + "/cache"
	cargoBuildGuestTarget  = "/tmp/haa-cargo-target"
	cargoBuildGuestHome    = cargoBuildGuestInput + "/cargo-home"
)

type projectBuildVolumeConfig struct {
	destination string
	executable  bool
}

func cargoBuildCreateArguments(input, target closureVolume, readOnly bool) ([]string, error) {
	if input.projectBuildDestination != cargoBuildGuestInput || input.projectBuildExecutable || target.projectBuildDestination != cargoBuildGuestTarget || !target.projectBuildExecutable || target.transaction != input.transaction+"-target" || input.name == target.name {
		return nil, errors.New("cargo build volume binding is invalid")
	}
	inputMount, err := input.mountArgument(readOnly)
	if err != nil {
		return nil, err
	}
	targetMount, err := target.mountArgument(false)
	if err != nil {
		return nil, err
	}
	return cargoBuildCreateArgumentsForMounts(inputMount, targetMount), nil
}

// Actual execution and the deterministic recipe consume the same fixed argv.
// Only the already-attested volume references differ per operation.
func cargoBuildCreateArgumentsForMounts(inputMount, targetMount string) []string {
	args := []string{"create", "--pull", "never", "--runtime", gVisorRuntimeName, "--network", "none", "--read-only", "--cap-drop", "ALL", "--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=512m,uid=1000,gid=1000,mode=0700", "--tmpfs", boundaryHelperMount, "--workdir", "/", "--mount", inputMount, "--mount", targetMount}
	args = append(args, isolatedContainerEnvironmentArguments()...)
	runtime := PinnedCargoRuntime()
	for _, value := range []string{"HOME=/tmp", "CARGO_HOME=" + cargoBuildGuestHome, "RUSTUP_HOME=/usr/local/rustup", "RUSTC=" + runtime.RustcBinary(), "RUSTDOC=/usr/local/rustup/toolchains/" + runtime.RustVersion + "-" + runtime.Target + "/bin/rustdoc", "CARGO_BUILD_JOBS=1", "CARGO_NET_OFFLINE=true", "CARGO_INCREMENTAL=0", "CARGO_TARGET_DIR=" + cargoBuildGuestTarget, "TMPDIR=/tmp", "XDG_CONFIG_HOME=/tmp/.config"} {
		args = append(args, "--env", value)
	}
	return append(args, runtime.ImageReference, "/bin/sh", "-ceu", boundaryContainerCommand())
}

// Controller source replacement consumes only independently approved vendor
// bytes. User project replacements and ambient credentials remain forbidden.
const cargoBuildSourceConfiguration = "[source.crates-io]\nreplace-with = 'haa-verified'\n[source.haa-verified]\ndirectory = '/tmp/haa-cargo-input/cache'\n"

func cargoBuildMetadataArguments() []string {
	return []string{"metadata", "--frozen", "--offline", "--locked", "--format-version", "1", "--manifest-path", cargoBuildGuestProject + "/Cargo.toml"}
}
func cargoBuildCompilerArguments() []string {
	return []string{"build", "--frozen", "--offline", "--locked", "--manifest-path", cargoBuildGuestProject + "/Cargo.toml", "--target", PinnedCargoRuntime().Target, "--target-dir", cargoBuildGuestTarget}
}
