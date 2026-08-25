package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// PyPIPromotion installs only staged exact wheels in the pinned, offline
// Python image.  It is deliberately separate from the generic Promotion port
// implementation so pip/Docker details cannot reach Application or Domain.
type PyPIPromotion struct {
	stagingRoot  string
	runner       DockerRunner
	goos, goarch string
}

func NewPyPIPromotion(stagingRoot string) (*PyPIPromotion, error) {
	return newPyPIPromotion(stagingRoot, unavailableDockerRunner{}, runtime.GOOS, runtime.GOARCH)
}

// NewPyPIPromotionWithRunner composes production Promotion with a validated
// Docker capability instead of resolving a Host binary itself.
func NewPyPIPromotionWithRunner(stagingRoot string, runner DockerRunner) (*PyPIPromotion, error) {
	return newPyPIPromotion(stagingRoot, runner, runtime.GOOS, runtime.GOARCH)
}

func newPyPIPromotion(stagingRoot string, runner DockerRunner, goos, goarch string) (*PyPIPromotion, error) {
	if !filepath.IsAbs(stagingRoot) || runner == nil {
		return nil, errors.New("PyPI Promotion requires absolute staging root and runtime runner")
	}
	return &PyPIPromotion{filepath.Clean(stagingRoot), runner, goos, goarch}, nil
}

func (p *PyPIPromotion) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, installContext domain.InstallContext) (promoted domain.PromotedInstall, resultErr error) {
	if ctx == nil || ctx.Err() != nil || p == nil || p.goos != "linux" || p.goarch != "amd64" {
		return domain.PromotedInstall{}, errors.New("automatic PyPI Promotion requires Linux amd64")
	}
	if !bundle.Valid() || staged.ManifestID() != bundle.ManifestID() || staged.Handle() != "staging:"+bundle.ManifestID().String() || verifyDocuments(bundle) != nil {
		return domain.PromotedInstall{}, errors.New("PyPI staged bundle binding is invalid")
	}
	stagedRoot := filepath.Join(p.stagingRoot, bundle.ManifestID().String())
	if filepath.Dir(stagedRoot) != p.stagingRoot || rejectSymlinkPath(stagedRoot) != nil || verifyStagedRecords(stagedRoot, bundle) != nil {
		return domain.PromotedInstall{}, errors.New("PyPI staged records are unavailable or untrusted")
	}
	if installContext.Mode() == domain.InstallPythonVenv {
		return p.promoteActiveVenv(ctx, stagedRoot, bundle, installContext)
	}
	if installContext.Mode() != domain.InstallNewTarget {
		return domain.PromotedInstall{}, errors.New("PyPI install context is unsupported")
	}
	target := installContext.Target().String()
	parent := filepath.Dir(target)
	if target == "" || trustedExistingDirectory(parent) != nil {
		return domain.PromotedInstall{}, errors.New("PyPI install target parent is untrusted")
	}
	parentIdentity, err := os.Stat(parent)
	if err != nil {
		return domain.PromotedInstall{}, errors.New("capture PyPI target parent identity")
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return domain.PromotedInstall{}, errors.New("PyPI install target already exists or cannot be verified")
	}
	temporary, err := os.MkdirTemp(parent, "."+filepath.Base(target)+".haa-")
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			if err := os.RemoveAll(temporary); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("remove incomplete PyPI target: %w", err))
			}
		}
	}()
	if err := os.Chmod(temporary, 0o700); err != nil {
		return domain.PromotedInstall{}, err
	}
	requirements, expected, err := preparePyPIProject(temporary, stagedRoot, bundle)
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := p.runner.Run(ctx, temporary, pypiPromotionArguments(temporary)); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := validatePyPIOutput(filepath.Join(temporary, "site"), expected, requirements); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := syncTree(temporary); err != nil {
		return domain.PromotedInstall{}, err
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return domain.PromotedInstall{}, errors.New("PyPI install target appeared before publish")
	}
	currentParent, err := os.Stat(parent)
	if err != nil || !os.SameFile(parentIdentity, currentParent) || trustedExistingDirectory(parent) != nil {
		return domain.PromotedInstall{}, errors.New("PyPI install target parent changed before publish")
	}
	if err := renameNoReplace(temporary, target); err != nil {
		return domain.PromotedInstall{}, fmt.Errorf("atomically publish PyPI target: %w", err)
	}
	cleanup = false
	if err := syncDirectory(parent); err != nil {
		return domain.PromotedInstall{}, err
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), installContext.Target())
}

func (p *PyPIPromotion) promoteActiveVenv(ctx context.Context, stagedRoot string, bundle domain.VerifiedBundle, installContext domain.InstallContext) (domain.PromotedInstall, error) {
	plan, err := discoverPythonVenv(installContext.Target().String())
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	temporary, err := os.MkdirTemp(plan.root, ".haa-pypi-work-")
	if err != nil {
		return domain.PromotedInstall{}, errors.New("create private PyPI virtual environment workspace")
	}
	defer os.RemoveAll(temporary)
	if err := os.Chmod(temporary, 0o700); err != nil {
		return domain.PromotedInstall{}, err
	}
	requirements, expected, err := preparePyPIProject(temporary, stagedRoot, bundle)
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := p.runner.Run(ctx, temporary, pypiPromotionArguments(temporary)); err != nil {
		return domain.PromotedInstall{}, err
	}
	output := filepath.Join(temporary, "site")
	if err := validatePyPIOutput(output, expected, requirements); err != nil {
		return domain.PromotedInstall{}, err
	}
	desired, err := plan.outputState(output)
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := plan.commit(output, desired); err != nil {
		return domain.PromotedInstall{}, err
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), installContext.Target())
}

func pypiPromotionArguments(project string) []string {
	identity := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
	mount := "type=bind,src=" + project + ",dst=/workspace"
	return []string{"run", "--rm", "--pull", "never", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "128", "--memory", "512m", "--cpus", "1", "--user", identity, "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m", "--mount", mount, "--workdir", "/workspace", "--env", "HOME=/tmp", "--env", "PIP_CACHE_DIR=/tmp/pip-cache", "--env", "PIP_CONFIG_FILE=/dev/null", "--entrypoint", "python", sandbox.PinnedPythonRuntime().ImageReference, "-I", "-m", "pip", "install", "--no-index", "--find-links", "/workspace/wheels", "--require-hashes", "--only-binary", ":all:", "--no-deps", "--no-compile", "--disable-pip-version-check", "--target", "/workspace/site", "--requirement", "/workspace/requirements.txt"}
}

type pypiExpected struct{ name, version, digest string }

func preparePyPIProject(project, stagedRoot string, bundle domain.VerifiedBundle) ([]byte, map[string]pypiExpected, error) {
	wheels := filepath.Join(project, "wheels")
	if err := os.Mkdir(wheels, 0o700); err != nil {
		return nil, nil, err
	}
	if err := copyExactRecord(filepath.Join(stagedRoot, manifestFilename), filepath.Join(project, manifestFilename), bundle.ManifestDocument()); err != nil {
		return nil, nil, err
	}
	if err := copyExactRecord(filepath.Join(stagedRoot, sbomFilename), filepath.Join(project, sbomFilename), bundle.SBOMDocument()); err != nil {
		return nil, nil, err
	}
	expected := map[string]pypiExpected{}
	lines := make([]string, 0)
	for _, node := range bundle.Set().Inspected().Graph().Nodes() {
		resolved := node.Artifact()
		if resolved.Identity().Variant() == "sdist" {
			continue
		}
		if resolved.Identity().Source().String() != "pypi" || (resolved.Identity().Variant() != "wheel" && resolved.Identity().Variant() != "derived-wheel") {
			return nil, nil, errors.New("PyPI Promotion requires exact wheels only")
		}
		filename := filepath.Base(node.RecordPath())
		projectName, version, _, _, _, err := artifactpypi.ParseWheelFilename(filename)
		if err != nil || projectName != resolved.Identity().Name() || version != resolved.Identity().Version() || expected[projectName].name != "" {
			return nil, nil, errors.New("PyPI Promotion wheel filename is invalid or ambiguous")
		}
		inspection, ok := findInspection(bundle, node.Node())
		if !ok || inspection.Artifact().Identity() != resolved.Identity() {
			return nil, nil, errors.New("PyPI Promotion inspection binding is incomplete")
		}
		digest := inspection.Artifact().Digest().String()
		source := filepath.Join(stagedRoot, "artifacts", digest+".whl")
		if err := copyDigestFile(source, filepath.Join(wheels, filename), digest); err != nil {
			return nil, nil, err
		}
		expected[projectName] = pypiExpected{projectName, version, digest}
		lines = append(lines, projectName+"=="+version+" --hash=sha256:"+digest)
	}
	if len(lines) == 0 {
		return nil, nil, errors.New("PyPI Promotion wheel set is empty")
	}
	sort.Strings(lines)
	requirements := []byte(strings.Join(lines, "\n") + "\n")
	if err := writePromotionFile(filepath.Join(project, "requirements.txt"), requirements); err != nil {
		return nil, nil, err
	}
	return requirements, expected, nil
}

func findInspection(bundle domain.VerifiedBundle, node domain.DependencyNodeID) (domain.DependencyInspection, bool) {
	for _, item := range bundle.Set().Inspected().Inspections() {
		if item.Node() == node {
			return item, true
		}
	}
	return domain.DependencyInspection{}, false
}

func validatePyPIOutput(site string, expected map[string]pypiExpected, requirements []byte) error {
	if len(requirements) == 0 || len(expected) == 0 || rejectSymlinkPath(site) != nil {
		return errors.New("PyPI Promotion output is unavailable")
	}
	installed := map[string]bool{}
	err := filepath.WalkDir(site, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("PyPI Promotion output contains symbolic link")
		}
		if entry.IsDir() {
			if strings.HasSuffix(entry.Name(), ".dist-info") {
				name, version, ok := installedDistInfo(entry.Name())
				if !ok || expected[name].version != version || installed[name] {
					return errors.New("PyPI installed distribution set is invalid")
				}
				if err := validateInstalledRecord(site, path, expected[name]); err != nil {
					return err
				}
				installed[name] = true
			}
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil || !info.Mode().IsRegular() {
			return errors.New("PyPI Promotion output contains special file")
		}
		return nil
	})
	if err != nil || len(installed) != len(expected) {
		return errors.New("PyPI Promotion output does not match exact distribution set")
	}
	return nil
}

func installedDistInfo(value string) (string, string, bool) {
	value = strings.TrimSuffix(value, ".dist-info")
	at := strings.LastIndex(value, "-")
	if at <= 0 {
		return "", "", false
	}
	name, err := artifactpypi.NormalizeProjectName(value[:at])
	if err != nil {
		return "", "", false
	}
	version, err := artifactpypi.NormalizeVersion(value[at+1:])
	return name, version, err == nil
}
func validateInstalledRecord(site, directory string, expected pypiExpected) error {
	metadata, err := os.ReadFile(filepath.Join(directory, "METADATA"))
	if err != nil || len(metadata) == 0 {
		return errors.New("PyPI installed metadata is unavailable")
	}
	metadataText := string(metadata)
	if !strings.Contains(metadataText, "Name: "+expected.name+"\n") || !strings.Contains(metadataText, "Version: "+expected.version+"\n") {
		return errors.New("PyPI installed metadata does not match expected distribution")
	}
	record, err := os.Open(filepath.Join(directory, "RECORD"))
	if err != nil {
		return errors.New("PyPI installed RECORD is unavailable")
	}
	defer record.Close()
	r := csv.NewReader(record)
	r.FieldsPerRecord = 3
	count := 0
	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || row[0] == "" || strings.HasPrefix(row[0], "/") || strings.Contains(row[0], "..") {
			return errors.New("PyPI installed RECORD is invalid")
		}
		if row[1] != "" {
			if !strings.HasPrefix(row[1], "sha256=") {
				return errors.New("PyPI installed RECORD hash is invalid")
			}
			digest, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(row[1], "sha256="))
			if err != nil || len(digest) != sha256.Size {
				return errors.New("PyPI installed RECORD hash is invalid")
			}
			path := filepath.Join(site, filepath.FromSlash(row[0]))
			if relative, err := filepath.Rel(site, path); err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return errors.New("PyPI installed RECORD escapes target")
			}
			body, readErr := os.ReadFile(path)
			sum := sha256.Sum256(body)
			if readErr != nil || !infoRegular(path) || !strings.EqualFold(base64.RawURLEncoding.EncodeToString(sum[:]), strings.TrimPrefix(row[1], "sha256=")) {
				return errors.New("PyPI installed RECORD content hash is invalid")
			}
		}
		count++
	}
	if count == 0 {
		return errors.New("PyPI installed RECORD is empty")
	}
	return nil
}

func infoRegular(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}
