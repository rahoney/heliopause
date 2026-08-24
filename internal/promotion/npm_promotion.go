package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

var promotionNodeImage = runtimeidentity.NodeImageReference

// DockerRunner is a prevalidated, minimal-environment Docker execution
// capability supplied by production bootstrap.
type DockerRunner interface {
	Run(context.Context, string, []string) error
}

type unavailableDockerRunner struct{}

func (unavailableDockerRunner) Run(context.Context, string, []string) error {
	return errors.New("trusted Docker executor is unavailable")
}

// NPMPromotion installs a staged npm graph in a disposable pinned Docker
// runtime, validates its output, and atomically publishes a new target.
type NPMPromotion struct {
	stagingRoot string
	runner      DockerRunner
	goos        string
	goarch      string
}

func NewNPMPromotion(stagingRoot string) (*NPMPromotion, error) {
	return newNPMPromotion(stagingRoot, unavailableDockerRunner{}, runtime.GOOS, runtime.GOARCH)
}

// NewNPMPromotionWithRunner composes production Promotion with a validated
// Docker capability instead of resolving a Host binary itself.
func NewNPMPromotionWithRunner(stagingRoot string, runner DockerRunner) (*NPMPromotion, error) {
	return newNPMPromotion(stagingRoot, runner, runtime.GOOS, runtime.GOARCH)
}

func newNPMPromotion(stagingRoot string, runner DockerRunner, goos, goarch string) (*NPMPromotion, error) {
	root := filepath.Clean(stagingRoot)
	if !filepath.IsAbs(root) || runner == nil {
		return nil, errors.New("npm Promotion requires absolute staging root and runtime runner")
	}
	return &NPMPromotion{stagingRoot: root, runner: runner, goos: goos, goarch: goarch}, nil
}

func (p *NPMPromotion) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, installContext domain.InstallContext) (promoted domain.PromotedInstall, resultErr error) {
	if ctx == nil {
		return domain.PromotedInstall{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.PromotedInstall{}, err
	}
	if p == nil || p.goos != "linux" || p.goarch != "amd64" {
		return domain.PromotedInstall{}, errors.New("automatic npm Promotion requires Linux amd64")
	}
	if staged.ManifestID() != bundle.ManifestID() || staged.Handle() != "staging:"+bundle.ManifestID().String() || !bundle.Valid() {
		return domain.PromotedInstall{}, errors.New("staged set and verified bundle binding is invalid")
	}
	if err := verifyDocuments(bundle); err != nil {
		return domain.PromotedInstall{}, err
	}
	stagedRoot := filepath.Join(p.stagingRoot, bundle.ManifestID().String())
	if filepath.Dir(stagedRoot) != p.stagingRoot || rejectSymlinkPath(stagedRoot) != nil {
		return domain.PromotedInstall{}, errors.New("staged set path is unavailable or untrusted")
	}
	if err := verifyStagedRecords(stagedRoot, bundle); err != nil {
		return domain.PromotedInstall{}, err
	}
	if installContext.Mode() == domain.InstallNPMProject {
		guard, err := acquireNPMProjectGuard(installContext.Target().String())
		if err != nil {
			return domain.PromotedInstall{}, err
		}
		defer func() { resultErr = errors.Join(resultErr, guard.release()) }()
		plan, err := freezeNPMProject(installContext.Target().String())
		if err != nil {
			return domain.PromotedInstall{}, err
		}
		if err := plan.verifyUnchanged(); err != nil {
			return domain.PromotedInstall{}, err
		}
		return domain.PromotedInstall{}, errors.New("npm project transaction commit is not yet available")
	}
	if installContext.Mode() != domain.InstallNewTarget {
		return domain.PromotedInstall{}, errors.New("npm install context is unsupported")
	}
	target := installContext.Target().String()
	if target == "" || strings.Contains(target, ",") {
		return domain.PromotedInstall{}, errors.New("install target is unsupported by the Docker mount boundary")
	}
	parent := filepath.Dir(target)
	if err := trustedExistingDirectory(parent); err != nil {
		return domain.PromotedInstall{}, err
	}
	parentIdentity, err := os.Stat(parent)
	if err != nil {
		return domain.PromotedInstall{}, errors.New("capture install target parent identity")
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return domain.PromotedInstall{}, errors.New("install target already exists or cannot be verified")
	}
	temporary, err := os.MkdirTemp(parent, "."+filepath.Base(target)+".haa-")
	if err != nil {
		return domain.PromotedInstall{}, fmt.Errorf("create temporary install target: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			if err := os.RemoveAll(temporary); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("remove incomplete install target: %w", err))
			}
		}
	}()
	manifest, lock, err := preparePromotionProject(temporary, stagedRoot, bundle)
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	arguments := promotionArguments(temporary)
	if err := p.runner.Run(ctx, temporary, arguments); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := validatePromotionOutput(temporary, bundle, manifest, lock); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := syncTree(temporary); err != nil {
		return domain.PromotedInstall{}, err
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return domain.PromotedInstall{}, errors.New("install target appeared before publish")
	}
	if err := trustedExistingDirectory(parent); err != nil {
		return domain.PromotedInstall{}, err
	}
	currentParent, err := os.Stat(parent)
	if err != nil || !os.SameFile(parentIdentity, currentParent) {
		return domain.PromotedInstall{}, errors.New("install target parent changed before publish")
	}
	if err := renameNoReplace(temporary, target); err != nil {
		return domain.PromotedInstall{}, fmt.Errorf("atomically publish install target: %w", err)
	}
	cleanup = false
	if err := syncDirectory(parent); err != nil {
		cleanupErr := os.RemoveAll(target)
		if cleanupErr != nil {
			return domain.PromotedInstall{}, errors.Join(err, fmt.Errorf("remove uncertain install target: %w", cleanupErr))
		}
		return domain.PromotedInstall{}, err
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), installContext.Target())
}

func promotionArguments(project string) []string {
	identity := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
	mount := "type=bind,src=" + project + ",dst=/workspace"
	return []string{"run", "--rm", "--pull", "never", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "128", "--memory", "512m", "--cpus", "1", "--user", identity, "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m", "--mount", mount, "--workdir", "/workspace", "--env", "HOME=/tmp", "--env", "npm_config_cache=/tmp/npm-cache", "--env", "npm_config_userconfig=/tmp/haa-user.npmrc", "--env", "npm_config_globalconfig=/tmp/haa-global.npmrc", "--entrypoint", "npm", promotionNodeImage, "ci", "--offline", "--ignore-scripts", "--no-audit", "--no-fund", "--bin-links=false"}
}

type promotionPackage struct {
	Name         string            `json:"name,omitempty"`
	Version      string            `json:"version,omitempty"`
	Resolved     string            `json:"resolved,omitempty"`
	Integrity    string            `json:"integrity,omitempty"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

type promotionLock struct {
	Name            string                      `json:"name"`
	Version         string                      `json:"version"`
	LockfileVersion int                         `json:"lockfileVersion"`
	Requires        bool                        `json:"requires"`
	Packages        map[string]promotionPackage `json:"packages"`
}

func preparePromotionProject(project, stagedRoot string, bundle domain.VerifiedBundle) ([]byte, []byte, error) {
	haa := filepath.Join(project, ".heliopause")
	artifacts := filepath.Join(haa, "artifacts")
	if err := os.MkdirAll(artifacts, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create target-local artifact directory: %w", err)
	}
	if err := copyExactRecord(filepath.Join(stagedRoot, manifestFilename), filepath.Join(haa, manifestFilename), bundle.ManifestDocument()); err != nil {
		return nil, nil, err
	}
	if err := copyExactRecord(filepath.Join(stagedRoot, sbomFilename), filepath.Join(haa, sbomFilename), bundle.SBOMDocument()); err != nil {
		return nil, nil, err
	}
	written := map[string]bool{}
	for _, inspection := range bundle.Set().Inspected().Inspections() {
		digest := inspection.Artifact().Digest().String()
		if written[digest] {
			continue
		}
		if err := copyDigestFile(filepath.Join(stagedRoot, "artifacts", digest+".tgz"), filepath.Join(artifacts, digest+".tgz"), digest); err != nil {
			return nil, nil, err
		}
		written[digest] = true
	}
	manifest, lock, err := generatedNPMProject(bundle)
	if err != nil {
		return nil, nil, err
	}
	if err := writePromotionFile(filepath.Join(project, "package.json"), manifest); err != nil {
		return nil, nil, err
	}
	if err := writePromotionFile(filepath.Join(project, "package-lock.json"), lock); err != nil {
		return nil, nil, err
	}
	return manifest, lock, nil
}

func generatedNPMProject(bundle domain.VerifiedBundle) ([]byte, []byte, error) {
	graph := bundle.Set().Inspected().Graph()
	nodes := graph.Nodes()
	byID := make(map[domain.DependencyNodeID]domain.LockedDependency, len(nodes))
	inspections := make(map[domain.DependencyNodeID]domain.DependencyInspection, len(nodes))
	for _, node := range nodes {
		if node.Artifact().Identity().Source().String() != "npm" {
			return nil, nil, errors.New("npm Promotion received a non-npm dependency")
		}
		byID[node.Node()] = node
	}
	for _, inspection := range bundle.Set().Inspected().Inspections() {
		inspections[inspection.Node()] = inspection
	}
	children := make(map[domain.DependencyNodeID][]domain.DependencyNodeID)
	for _, edge := range graph.Edges() {
		children[edge.From()] = append(children[edge.From()], edge.To())
	}
	packages := make(map[string]promotionPackage, len(nodes)+1)
	primary := byID[graph.Primary()]
	primaryInspection := inspections[graph.Primary()]
	primarySpec := localArtifactSpec(primaryInspection.Artifact().Digest().String())
	packages[""] = promotionPackage{Name: "heliopause-promoted-set", Version: "1.0.0", Dependencies: map[string]string{primary.Artifact().Identity().Name(): primarySpec}}
	for _, node := range nodes {
		inspection, ok := inspections[node.Node()]
		if !ok {
			return nil, nil, errors.New("npm Promotion graph inspection is incomplete")
		}
		dependencies := map[string]string{}
		for _, childID := range children[node.Node()] {
			child := byID[childID]
			if _, duplicate := dependencies[child.Artifact().Identity().Name()]; duplicate {
				return nil, nil, errors.New("npm Promotion graph contains ambiguous child names")
			}
			dependencies[child.Artifact().Identity().Name()] = child.Artifact().Identity().Version()
		}
		if len(dependencies) == 0 {
			dependencies = nil
		}
		declared, ok := inspection.Artifact().DeclaredIntegrity()
		if !ok {
			return nil, nil, errors.New("npm Promotion artifact lacks declared integrity")
		}
		packages[node.RecordPath()] = promotionPackage{Name: node.Artifact().Identity().Name(), Version: node.Artifact().Identity().Version(), Resolved: localArtifactSpec(inspection.Artifact().Digest().String()), Integrity: declared, Dependencies: dependencies}
	}
	project := struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		Private      bool              `json:"private"`
		Dependencies map[string]string `json:"dependencies"`
	}{Name: "heliopause-promoted-set", Version: "1.0.0", Private: true, Dependencies: packages[""].Dependencies}
	manifest, err := json.Marshal(project)
	if err != nil {
		return nil, nil, err
	}
	lock, err := json.Marshal(promotionLock{Name: project.Name, Version: project.Version, LockfileVersion: 3, Requires: true, Packages: packages})
	if err != nil {
		return nil, nil, err
	}
	return manifest, lock, nil
}

func localArtifactSpec(digest string) string { return "file:.heliopause/artifacts/" + digest + ".tgz" }

func verifyStagedRecords(root string, bundle domain.VerifiedBundle) error {
	for name, expected := range map[string][]byte{manifestFilename: bundle.ManifestDocument(), sbomFilename: bundle.SBOMDocument()} {
		path := filepath.Join(root, name)
		if err := rejectSymlinkPath(path); err != nil {
			return errors.New("staged records contain an untrusted path")
		}
		actual, err := os.ReadFile(path)
		if err != nil || !equalBytes(actual, expected) {
			return errors.New("staged records changed before Promotion")
		}
	}
	return nil
}

func copyExactRecord(source, destination string, expected []byte) error {
	actual, err := os.ReadFile(source)
	if err != nil || !equalBytes(actual, expected) {
		return errors.New("staged record changed before Promotion")
	}
	return writePromotionFile(destination, actual)
}

func copyDigestFile(source, destination, expected string) error {
	if err := rejectSymlinkPath(source); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return errors.New("open staged artifact for Promotion")
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("staged artifact is not a regular file")
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("create target-local artifact")
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(output, hash), input)
	if copyErr == nil {
		copyErr = output.Sync()
	}
	if closeErr := output.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil || hex.EncodeToString(hash.Sum(nil)) != expected {
		return errors.New("staged artifact digest changed before Promotion")
	}
	return nil
}

func writePromotionFile(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create Promotion input: %w", err)
	}
	if _, err = file.Write(contents); err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func validatePromotionOutput(project string, bundle domain.VerifiedBundle, expectedManifest, expectedLock []byte) error {
	for name, expected := range map[string][]byte{"package.json": expectedManifest, "package-lock.json": expectedLock} {
		actual, err := os.ReadFile(filepath.Join(project, name))
		if err != nil || !equalBytes(actual, expected) {
			return errors.New("npm Promotion mutated its frozen project inputs")
		}
	}
	for name, expected := range map[string][]byte{manifestFilename: bundle.ManifestDocument(), sbomFilename: bundle.SBOMDocument()} {
		actual, err := os.ReadFile(filepath.Join(project, ".heliopause", name))
		if err != nil || !equalBytes(actual, expected) {
			return errors.New("npm Promotion mutated verified records")
		}
	}
	expected := make(map[string]bool)
	inspections := make(map[domain.DependencyNodeID]domain.DependencyInspection)
	for _, inspection := range bundle.Set().Inspected().Inspections() {
		inspections[inspection.Node()] = inspection
	}
	for _, node := range bundle.Set().Inspected().Graph().Nodes() {
		expected[filepath.FromSlash(node.RecordPath())] = true
		path := filepath.Join(project, filepath.FromSlash(node.RecordPath()))
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("npm Promotion output is missing an exact locked package")
		}
		body, err := os.ReadFile(filepath.Join(path, "package.json"))
		if err != nil || len(body) > 2<<20 {
			return errors.New("npm Promotion output package identity is unavailable")
		}
		var identity struct{ Name, Version string }
		if json.Unmarshal(body, &identity) != nil || identity.Name != node.Artifact().Identity().Name() || identity.Version != node.Artifact().Identity().Version() {
			return errors.New("npm Promotion output package identity differs from Manifest")
		}
		inspection := inspections[node.Node()]
		digest := inspection.Artifact().Digest().String()
		if err := verifyDigestFile(filepath.Join(project, ".heliopause", "artifacts", digest+".tgz"), digest); err != nil {
			return err
		}
	}
	actual, err := installedPackagePaths(filepath.Join(project, "node_modules"), project)
	if err != nil {
		return err
	}
	if len(actual) != len(expected) {
		return errors.New("npm Promotion output package set differs from Manifest")
	}
	for path := range actual {
		if !expected[path] {
			return errors.New("npm Promotion output contains an unlisted package")
		}
	}
	return nil
}

func verifyDigestFile(path, expected string) error {
	if err := rejectSymlinkPath(path); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return errors.New("open target-local artifact for verification")
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || hex.EncodeToString(hash.Sum(nil)) != expected {
		return errors.New("target-local artifact changed during Promotion")
	}
	return nil
}

func installedPackagePaths(nodeModules, project string) (map[string]bool, error) {
	result := map[string]bool{}
	err := filepath.WalkDir(nodeModules, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("npm Promotion output contains a symbolic link")
		}
		if !entry.IsDir() || path == nodeModules {
			return nil
		}
		if _, err := os.Lstat(filepath.Join(path, "package.json")); err == nil {
			relative, err := filepath.Rel(project, path)
			if err != nil || strings.HasPrefix(relative, "..") {
				return errors.New("npm Promotion package escapes target")
			}
			result[relative] = true
		}
		return nil
	})
	return result, err
}

func trustedExistingDirectory(path string) error {
	if err := rejectSymlinkPath(path); err != nil {
		return errors.New("install target parent is unavailable or contains a symbolic link")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() {
		return errors.New("install target parent is not a trusted directory")
	}
	return nil
}

func syncTree(root string) error {
	directories := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("promotion output contains a symbolic link")
		}
		if entry.IsDir() {
			directories = append(directories, path)
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("promotion output contains a non-regular file")
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		err = file.Sync()
		closeErr := file.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
	if err != nil {
		return err
	}
	sort.Slice(directories, func(i, j int) bool { return len(directories[i]) > len(directories[j]) })
	for _, directory := range directories {
		if err := syncDirectory(directory); err != nil {
			return err
		}
	}
	return nil
}

func equalBytes(first, second []byte) bool {
	if len(first) != len(second) {
		return false
	}
	var difference byte
	for index := range first {
		difference |= first[index] ^ second[index]
	}
	return difference == 0
}
