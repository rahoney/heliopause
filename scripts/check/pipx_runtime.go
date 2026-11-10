package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func checkPipxProofFiles(root string, lock map[string]any) error {
	pin, ok := lock["pipx_tool"].(map[string]any)
	if !ok || len(pin) != 6 || pin["architecture"] != "amd64" || pin["runtime_image"] != "python_image" {
		return errors.New("pipx runtime owner fields differ")
	}
	version, ok := pin["pipx_version"].(string)
	if !ok || !exactRuntimeVersion.MatchString(version) {
		return errors.New("pipx version is invalid")
	}
	inputs, ok := pin["inputs"].([]any)
	if !ok || len(inputs) != 42 {
		return errors.New("pipx input set differs")
	}
	namePattern := regexp.MustCompile(`^(argcomplete|click|filelock|iniconfig|packaging|pipx|platformdirs|pluggy|pygments|pytest|userpath)/(identity\.json|metadata\.json|provenance\.json|[a-z]+-[0-9]+(?:\.[0-9]+)+-py3-none-any\.whl)$`)
	digestPattern := regexp.MustCompile(`^[a-f0-9]{64}$`)
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, raw := range inputs {
		f, ok := raw.(map[string]any)
		if !ok || len(f) != 2 {
			return errors.New("pipx input record differs")
		}
		name, nok := f["path"].(string)
		digest, dok := f["sha256"].(string)
		if !nok || !dok || len(name) > 256 || !namePattern.MatchString(name) || !digestPattern.MatchString(digest) || seen[name] {
			return errors.New("pipx input path/hash is invalid")
		}
		seen[name] = true
		project, _, _ := strings.Cut(name, "/")
		counts[project]++
		if strings.HasSuffix(name, ".whl") {
			if !strings.HasPrefix(strings.TrimPrefix(name, project+"/"), project+"-") {
				return errors.New("pipx wheel project differs")
			}
			continue
		}
		canonical := "tools/pipx/supply/" + name
		owned := map[string]any{"path": canonical, "sha256": digest}
		if err := checkPinnedRuntimeMetadata(root, owned, "path", "sha256", canonical); err != nil {
			return err
		}
	}
	for _, project := range []string{"argcomplete", "click", "filelock", "iniconfig", "packaging", "pipx", "platformdirs", "pluggy", "pygments", "pytest", "userpath"} {
		want := 4
		if project == "userpath" || project == "pygments" {
			want = 3
		}
		if counts[project] != want || !seen[project+"/identity.json"] || !seen[project+"/metadata.json"] || (project != "userpath" && project != "pygments") && !seen[project+"/provenance.json"] || (project == "userpath" || project == "pygments") && seen[project+"/provenance.json"] {
			return errors.New("pipx project inputs are incomplete")
		}
	}
	if !seen["pipx/pipx-"+version+"-py3-none-any.whl"] {
		return errors.New("pipx main wheel differs from selected version")
	}
	for _, entry := range []struct {
		key, path string
		limit     float64
	}{{"tool", "pipx-runtime.zip", 8 << 20}, {"notices", "tools/pipx/NOTICE", 64 << 10}} {
		f, ok := pin[entry.key].(map[string]any)
		if !ok || len(f) != 3 || f["path"] != entry.path {
			return errors.New("pipx asset identity fields differ")
		}
		size, sok := f["size"].(float64)
		digest, dok := f["sha256"].(string)
		if !sok || !dok || size <= 0 || size > entry.limit || size != float64(int64(size)) || !digestPattern.MatchString(digest) {
			return errors.New("pipx asset identity is invalid")
		}
		if entry.key == "notices" {
			if err := checkPinnedRuntimeMetadata(root, lock, "pipx_tool.notices.path", "pipx_tool.notices.sha256", entry.path); err != nil {
				return err
			}
			info, err := os.Lstat(filepath.Join(root, entry.path))
			if err != nil || float64(info.Size()) != size {
				return errors.New("pipx notices size differs")
			}
		}
	}
	return nil
}
