package promotion

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// The view contains controller-created data only. Its Python symlink resolves
// inside the pinned guest image, never through a Host interpreter or target.
type toolPythonView struct {
	path    string
	root    *os.Root
	info    os.FileInfo
	entries map[string]os.FileInfo
	config  pypiFileSnapshot
}

const toolPythonViewName = ".haa-tool-python-"

func prepareToolPythonView(parent string) (_ *toolPythonView, resultErr error) {
	if !filepath.IsAbs(parent) || filepath.Clean(parent) != parent || trustedExistingDirectory(parent) != nil {
		return nil, errors.New("tool materialization workspace is unsafe")
	}
	path, err := os.MkdirTemp(parent, toolPythonViewName)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	v := &toolPythonView{path: path, root: root, entries: map[string]os.FileInfo{}}
	accepted := false
	defer func() {
		if !accepted {
			resultErr = errors.Join(resultErr, root.Close())
		}
	}()
	v.info, err = root.Stat(".")
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"bin", "lib", "lib/python3.14", "lib/python3.14/site-packages"} {
		if err := root.Mkdir(name, 0700); err != nil {
			return nil, err
		}
	}
	file, err := root.OpenFile("pyvenv.cfg", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	body := "home = /usr/local/bin\ninclude-system-site-packages = false\nversion = " + sandbox.PinnedPythonRuntime().PythonVersion + "\n"
	_, writeErr := file.WriteString(body)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return nil, err
	}
	if err := root.Symlink("/usr/local/bin/python", "bin/python"); err != nil {
		return nil, err
	}
	for _, name := range []string{"bin", "lib", "lib/python3.14", "lib/python3.14/site-packages", "pyvenv.cfg", "bin/python"} {
		info, err := root.Lstat(name)
		if err != nil {
			return nil, err
		}
		v.entries[name] = info
	}
	v.config, err = snapshotBoundedPromotionFile(filepath.Join(path, "pyvenv.cfg"), 4096)
	if err != nil {
		return nil, err
	}
	accepted = true
	return v, nil
}

func (v *toolPythonView) verify() error {
	if v == nil || v.root == nil {
		return errors.New("tool interpreter view is unavailable")
	}
	info, err := os.Lstat(v.path)
	if err != nil || !os.SameFile(v.info, info) || info.Mode() != v.info.Mode() {
		return errors.New("tool interpreter view identity changed")
	}
	seen := map[string]bool{}
	err = fs.WalkDir(v.root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		before, ok := v.entries[name]
		if !ok {
			return errors.New("tool interpreter view contains foreign data")
		}
		now, err := v.root.Lstat(name)
		if err != nil || !os.SameFile(before, now) || now.Mode() != before.Mode() || now.Size() != before.Size() && !now.IsDir() {
			return errors.New("tool interpreter view member changed")
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != len(v.entries) || !v.config.matches(filepath.Join(v.path, "pyvenv.cfg")) {
		return errors.New("tool interpreter view config or membership changed")
	}
	target, err := v.root.Readlink("bin/python")
	if err != nil || target != "/usr/local/bin/python" {
		return errors.New("tool interpreter alias changed")
	}
	return nil
}

func (v *toolPythonView) close() (resultErr error) {
	if v == nil || v.root == nil {
		return errors.New("tool interpreter view is unavailable")
	}
	defer func() { resultErr = errors.Join(resultErr, v.root.Close()) }()
	if err := v.verify(); err != nil {
		return err
	}
	names := make([]string, 0, len(v.entries))
	for name := range v.entries {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for _, name := range names {
		now, err := v.root.Lstat(name)
		before := v.entries[name]
		if err != nil || !os.SameFile(before, now) || now.Mode() != before.Mode() {
			return errors.New("tool interpreter cleanup member changed")
		}
		if name == "pyvenv.cfg" && !v.config.matches(filepath.Join(v.path, name)) {
			return errors.New("tool interpreter cleanup config changed")
		}
		if name == "bin/python" {
			target, err := v.root.Readlink(name)
			if err != nil || target != "/usr/local/bin/python" {
				return errors.New("tool interpreter cleanup alias changed")
			}
		}
		if err := v.root.Remove(name); err != nil {
			return err
		}
	}
	info, err := os.Lstat(v.path)
	if err != nil || !os.SameFile(v.info, info) {
		return errors.New("tool interpreter cleanup root changed")
	}
	return os.Remove(v.path)
}

func toolPythonPromotionArguments(project, view, target string, policy artifactpypi.ResourcePolicy) ([]string, error) {
	for _, value := range []string{project, view, target} {
		if !filepath.IsAbs(value) || filepath.Clean(value) != value || value == "/" || strings.ContainsAny(value, ",:\\\x00\r\n") {
			return nil, errors.New("tool interpreter mount path is unsupported")
		}
	}
	if filepath.Dir(view) != filepath.Dir(project) || !strings.HasPrefix(filepath.Base(view), toolPythonViewName) || len(filepath.Base(view)) <= len(toolPythonViewName) || !separateRoots([]string{project, view}) {
		return nil, errors.New("tool interpreter view is outside private materialization")
	}
	for _, runtime := range []string{"/workspace", "/usr/local/bin", "/usr/local/lib", "/usr/bin", "/usr/lib", "/usr/lib64", "/bin", "/lib", "/lib64", "/etc", "/haa-runtime", "/proc", "/sys", "/dev", "/run"} {
		if !separateRoots([]string{target, runtime}) {
			return nil, errors.New("tool interpreter mount would shadow fixed runtime or input")
		}
	}
	if target == "/tmp" {
		return nil, errors.New("tool interpreter mount would shadow private temporary storage")
	}
	base := pypiPromotionArguments(project, policy)
	image := sandbox.PinnedPythonRuntime().ImageReference
	index := -1
	for i, value := range base {
		if value == image {
			if index != -1 {
				return nil, errors.New("tool materialization image is ambiguous")
			}
			index = i
		}
	}
	if index < 1 || len(base)-index < 4 || base[index+1] != "-I" || base[index+2] != "-m" || base[index+3] != "pip" {
		return nil, errors.New("tool materialization base recipe differs")
	}
	result := append([]string(nil), base[:index]...)
	result = append(result, "--env", "PYTHONDONTWRITEBYTECODE=1", "--mount", "type=bind,src="+view+",dst="+target)
	result = append(result, image, "-I", "-B", "-m", "pip", "--python", filepath.Join(target, "bin", "python"))
	return append(result, base[index+4:]...), nil
}

// The Host venv is represented by an empty controlled view in the guest.
// Fixed pip renders wrappers and their RECORD hashes for the final interpreter;
// the existing whole output validator remains the only destination producer.
func (p *PyPIPromotion) materializeToolPythonProject(ctx context.Context, temporary string, requirements []byte, expected map[string]pypiExpected, install domain.InstallContext) (result []pypiDestination, resultErr error) {
	if ctx == nil || p == nil || p.runner == nil || p.goos != "linux" || p.goarch != "amd64" || !install.Valid() || install.Mode() != domain.InstallPythonVenv {
		return nil, errors.New("tool materialization requires an existing Linux amd64 venv context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	policy := artifactpypi.ResourcePolicyFromContext(ctx)
	_, err := toolPythonPromotionArguments(temporary, filepath.Join(filepath.Dir(temporary), toolPythonViewName+"preflight"), install.Target().String(), policy)
	if err != nil {
		return nil, err
	}
	view, err := prepareToolPythonView(filepath.Dir(temporary))
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, view.close())
		if resultErr != nil {
			result = nil
		}
	}()
	arguments, err := toolPythonPromotionArguments(temporary, view.path, install.Target().String(), policy)
	if err != nil {
		return nil, err
	}
	result, resultErr = p.materializePythonProjectWithArguments(ctx, temporary, requirements, expected, arguments)
	return result, resultErr
}
