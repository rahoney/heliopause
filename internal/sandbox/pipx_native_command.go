package sandbox

import (
	"errors"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pipxNativeOperation uint8

const (
	pipxNativeVersion pipxNativeOperation = iota + 1
	pipxNativeInstall
	pipxNativeUpgrade
)

const pipxNativeHome = pythonSitePath + "/pipx"
const pipxNativeBin = pythonSitePath + "/tool-bin"
const pipxNativeWheels = "/haa-wheels"

// This creates closed argv, never execution permission. Its caller must hold
// the installed DATA lease, admit the runtime/query and validate the complete
// frozen wheel input before any installed interpreter can run.
func pipxNativeArguments(operation pipxNativeOperation, requirement artifactpypi.UpgradeRequirement) ([]string, error) {
	args := []string{"/usr/local/bin/python", "-I", "-S", "-B", "-c", pipxNativeProgram}
	if operation == pipxNativeVersion {
		if requirement.Requirement() != "" || requirement.Project() != "" {
			return nil, errors.New("pipx version accepts no requirement")
		}
		return append(args, "--version"), nil
	}
	canonical, err := artifactpypi.ParseUpgradeRequirement(requirement.Requirement())
	if err != nil || canonical.Requirement() != requirement.Requirement() || requirement.Reference().Source() != artifactpypi.PublicPyPIProfile().Source() {
		return nil, errors.New("pipx native input requires one canonical public requirement")
	}
	switch operation {
	case pipxNativeInstall:
		args = append(args, "install", canonical.Requirement(), "--backend", "uv", "--python", "/usr/local/bin/python")
	case pipxNativeUpgrade:
		if canonical.Requirement() != canonical.Project() {
			return nil, errors.New("pipx upgrade accepts only a bare project")
		}
		args = append(args, "upgrade", canonical.Project(), "--backend", "uv")
	default:
		return nil, errors.New("pipx native operation is unsupported")
	}
	// The pinned backend supplies its venv --python itself. Passing UV's common
	// resolver --python here would add a second, conflicting interpreter target.
	flags := []string{"--no-build", "--no-sources", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--link-mode=copy", "--index-url=" + artifactpypi.PublicPyPIProfile().IndexURL()}
	return append(args, "--pip-args="+strings.Join(flags, " ")), nil
}

// Public acquisition is separate and already closed. The native pipx/UV phase
// sees only fixed guest paths and frozen wheels; no Host environment is copied.
func pipxNativeEnvironmentArguments() []string {
	args := isolatedContainerEnvironmentArguments()
	for i := 1; i < len(args); i += 2 {
		if strings.HasPrefix(args[i], "PATH=") {
			args[i] = "PATH=/haa-runtime:" + strings.TrimPrefix(args[i], "PATH=")
		}
	}
	for _, entry := range []string{
		"PIPX_HOME=" + pipxNativeHome,
		"PIPX_BIN_DIR=" + pipxNativeBin,
		"PIPX_MAN_DIR=" + pythonSitePath + "/tool-man",
		"PIPX_COMPLETION_DIR=" + pythonSitePath + "/tool-completions",
		"PIPX_DEFAULT_PYTHON=/usr/local/bin/python",
		"PIPX_DEFAULT_BACKEND=uv",
		"PYTHONDONTWRITEBYTECODE=1",
		"UV_OFFLINE=true",
		"UV_NO_INDEX=true",
		"UV_FIND_LINKS=" + pipxNativeWheels,
		"UV_NO_CONFIG=true",
		"UV_PYTHON_DOWNLOADS=never",
		"UV_NO_CACHE=true",
	} {
		args = append(args, "--env", entry)
	}
	return args
}

// Only the original immutable ZIP is appended after the pinned stdlib. The
// program has no completion token and no authority to approve its own result.
const pipxNativeProgram = `import importlib.machinery, sys
version = '` + runtimeidentity.PythonVersion + `'
minor = version.split('.')[:2]
stdlib = '/usr/local/lib/python' + '.'.join(minor)
expected = ['/usr/local/lib/python' + ''.join(minor) + '.zip', stdlib, stdlib + '/lib-dynload']
if sys.version.split()[0] != version or not (sys.flags.isolated and sys.flags.no_site and sys.flags.dont_write_bytecode) or 'site' in sys.modules or sys.path != expected:
    raise RuntimeError('pipx native interpreter differs from pinned isolated startup')
root = '/haa-pipx/pipx-runtime.zip'
spec = importlib.machinery.PathFinder.find_spec('pipx', [root])
if spec is None or spec.origin != root + '/pipx/__init__.py' or list(spec.submodule_search_locations or []) != [root + '/pipx']:
    raise RuntimeError('pipx native package is outside original ZIP')
entry = importlib.machinery.PathFinder.find_spec('pipx.__main__', spec.submodule_search_locations)
if entry is None or entry.origin != root + '/pipx/__main__.py':
    raise RuntimeError('pipx native entry is outside original ZIP')
sys.path.append(root)
sys.argv = ['pipx', *sys.argv[1:]]
from pipx.__main__ import cli
raise SystemExit(cli())
`
