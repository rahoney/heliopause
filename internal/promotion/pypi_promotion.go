package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

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
	if err := artifactpypi.CheckTemporaryDisk(p.stagingRoot, artifactpypi.ResourcePolicyFromContext(ctx)); err != nil {
		return domain.PromotedInstall{}, err
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
	resourcePolicy := artifactpypi.ResourcePolicyFromContext(ctx)
	if err := p.runner.Run(ctx, temporary, pypiPromotionArguments(temporary, resourcePolicy)); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := relocateStagedSchemeRoots(temporary); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := validatePyPIOutput(filepath.Join(temporary, "site"), expected, requirements, resourcePolicy.WheelLimits().MaxMetadata); err != nil {
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

func (p *PyPIPromotion) promoteActiveVenv(ctx context.Context, stagedRoot string, bundle domain.VerifiedBundle, installContext domain.InstallContext) (promoted domain.PromotedInstall, resultErr error) {
	plan, err := discoverPythonVenv(installContext.Target().String())
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	transaction, err := beginPyPIVenvTransaction(plan)
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, transaction.close()) }()
	temporary, err := os.MkdirTemp(plan.root, ".haa-pypi-work-")
	if err != nil {
		return domain.PromotedInstall{}, errors.New("create private PyPI virtual environment workspace")
	}
	defer func() {
		if !transaction.recovery {
			resultErr = errors.Join(resultErr, os.RemoveAll(temporary))
		}
	}()
	if err := os.Chmod(temporary, 0o700); err != nil {
		return domain.PromotedInstall{}, err
	}
	requirements, expected, err := preparePyPIProject(temporary, stagedRoot, bundle)
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	resourcePolicy := artifactpypi.ResourcePolicyFromContext(ctx)
	if err := p.runner.Run(ctx, temporary, pypiPromotionArguments(temporary, resourcePolicy)); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := relocateStagedSchemeRoots(temporary); err != nil {
		return domain.PromotedInstall{}, err
	}
	output := filepath.Join(temporary, "site")
	desired, err := validatedPyPIDestinations(output, expected, requirements, resourcePolicy.WheelLimits().MaxMetadata)
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := transaction.commit(desired); err != nil {
		return domain.PromotedInstall{}, err
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), installContext.Target())
}

func pypiPromotionArguments(project string, resourcePolicy artifactpypi.ResourcePolicy) []string {
	identity := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
	mount := "type=bind,src=" + project + ",dst=/workspace"
	return []string{"run", "--rm", "--pull", "never", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "128", "--memory", strconv.FormatInt(resourcePolicy.RuntimeMemory(), 10), "--cpus", "1", "--user", identity, "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=" + strconv.FormatInt(resourcePolicy.PromotionTmpfs(), 10), "--mount", mount, "--workdir", "/workspace", "--env", "HOME=/tmp", "--env", "PIP_CACHE_DIR=/tmp/pip-cache", "--env", "PIP_CONFIG_FILE=/dev/null", "--entrypoint", "python", sandbox.PinnedPythonRuntime().ImageReference, "-I", "-m", "pip", "install", "--no-cache-dir", "--no-index", "--find-links", "/workspace/wheels", "--require-hashes", "--only-binary", ":all:", "--no-deps", "--no-compile", "--disable-pip-version-check", "--target", "/workspace/site", "--requirement", "/workspace/requirements.txt"}
}

func relocateStagedSchemeRoots(transactionRoot string) error {
	// Validate ancestors before inspecting children: Lstat alone follows links
	// in intermediate components. The caller owns this private workspace.
	if !filepath.IsAbs(transactionRoot) || filepath.Clean(transactionRoot) != transactionRoot || trustedExistingDirectory(transactionRoot) != nil {
		return errors.New("PyPI private staging root is invalid")
	}
	rootInfo, err := os.Lstat(transactionRoot)
	if err != nil {
		return err
	}
	site := filepath.Join(transactionRoot, "site")
	if !pathWithin(transactionRoot, site) || trustedExistingDirectory(site) != nil {
		return errors.New("PyPI installed site root is invalid")
	}
	siteInfo, err := os.Lstat(site)
	if err != nil {
		return err
	}
	type move struct {
		src, dst string
		info     os.FileInfo
	}
	var moves []move
	// Preflight BOTH schemes before the first mutation, including when only
	// the second scheme is hostile or has a destination collision.
	for _, scheme := range []string{"bin", "share"} {
		src := filepath.Join(site, scheme)
		dst := filepath.Join(transactionRoot, scheme)
		if !pathWithin(site, src) || !pathWithin(transactionRoot, dst) {
			return errors.New("PyPI scheme root escapes private staging")
		}
		if _, err := os.Lstat(dst); !errors.Is(err, os.ErrNotExist) {
			return errors.New("PyPI scheme root already exists")
		}
		info, err := os.Lstat(src)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("PyPI installed scheme root is invalid")
		}
		moves = append(moves, move{src, dst, info})
	}
	for _, planned := range moves {
		for _, identity := range []struct {
			path string
			info os.FileInfo
		}{{transactionRoot, rootInfo}, {site, siteInfo}, {planned.src, planned.info}} {
			if trustedExistingDirectory(identity.path) != nil {
				return errors.New("PyPI staging directory became unsafe")
			}
			now, err := os.Lstat(identity.path)
			if err != nil || !os.SameFile(identity.info, now) {
				return errors.New("PyPI staging directory identity changed")
			}
		}
		if err := renameNoReplace(planned.src, planned.dst); err != nil {
			return fmt.Errorf("relocate PyPI scheme: %w", err)
		}
	}
	return nil
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
		if _, supported := artifactpypi.ProfileForSource(resolved.Identity().Source()); !supported || (resolved.Identity().Variant() != "wheel" && resolved.Identity().Variant() != "derived-wheel") {
			return nil, nil, errors.New("PyPI Promotion requires exact wheels only")
		}
		filename := filepath.Base(node.RecordPath())
		projectName, version, _, _, _, err := artifactpypi.ParseWheelFilenameForSource(filename, resolved.Identity().Source())
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

func validatePyPIOutput(site string, expected map[string]pypiExpected, requirements []byte, maxMetadata ...int64) error {
	_, err := validatedPyPIDestinations(site, expected, requirements, maxMetadata...)
	return err
}

// validatedPyPIDestinations is the sole source of transaction destinations.
// Both new-target and existing-venv promotion consume this RECORD validation.
func validatedPyPIDestinations(site string, expected map[string]pypiExpected, requirements []byte, maxMetadata ...int64) ([]pypiDestination, error) {
	limit := artifactpypi.DefaultWheelLimits().MaxMetadata
	if len(maxMetadata) > 0 && maxMetadata[0] > 0 {
		limit = maxMetadata[0]
	}
	if len(requirements) == 0 || len(expected) == 0 || rejectSymlinkPath(site) != nil {
		return nil, errors.New("PyPI Promotion output is unavailable")
	}
	transactionRoot := filepath.Dir(filepath.Clean(site))
	if transactionRoot == site || !filepath.IsAbs(site) {
		return nil, errors.New("PyPI Promotion output root is invalid")
	}
	installed := map[string]bool{}
	recorded := map[string]bool{}
	destinations := []pypiDestination{}
	err := filepath.WalkDir(site, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("PyPI Promotion output contains symbolic link")
		}
		if entry.IsDir() {
			if strings.HasSuffix(entry.Name(), ".dist-info") && filepath.Clean(filepath.Dir(path)) == filepath.Clean(site) {
				name, version, ok := installedDistInfo(entry.Name())
				if !ok || expected[name].version != version || installed[name] {
					return errors.New("PyPI installed distribution set is invalid")
				}
				if err := validateInstalledRecord(site, path, expected[name], recorded, &destinations, limit); err != nil {
					return err
				}
				installed[name] = true
				return filepath.SkipDir
			}
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil || !info.Mode().IsRegular() {
			return errors.New("PyPI Promotion output contains special file")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(installed) != len(expected) {
		return nil, errors.New("PyPI Promotion output does not match exact distribution set")
	}
	if err := validateRecordedOutputRoots(site, transactionRoot, recorded); err != nil {
		return nil, err
	}
	sort.Slice(destinations, func(i, j int) bool { return destinations[i].key() < destinations[j].key() })
	return destinations, nil
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
	version, err := artifactpypi.NormalizeInstalledVersion(value[at+1:])
	return name, version, err == nil
}

func validateInstalledRecord(site, directory string, expected pypiExpected, recorded map[string]bool, destinations *[]pypiDestination, limit int64) error {
	metadata, err := readBoundedPromotionFile(filepath.Join(directory, "METADATA"), limit)
	if err != nil || len(metadata) == 0 {
		return errors.New("PyPI installed metadata is unavailable")
	}
	name, version, err := parseInstalledMetadata(metadata)
	if err != nil || name != expected.name || version != expected.version {
		return errors.New("PyPI installed metadata does not match expected distribution")
	}
	recordBody, err := readBoundedPromotionFile(filepath.Join(directory, "RECORD"), limit)
	if err != nil {
		return errors.New("PyPI installed RECORD is unavailable")
	}
	r := csv.NewReader(bytes.NewReader(recordBody))
	r.FieldsPerRecord = 3
	r.ReuseRecord = false
	count := 0
	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || row[0] == "" {
			return errors.New("PyPI installed RECORD is invalid")
		}
		destination, err := resolveInstalledRecordPath(site, directory, row[0])
		entryPath := destination.Source
		if err != nil || recorded[entryPath] {
			return errors.New("PyPI installed RECORD is invalid")
		}
		recorded[entryPath] = true
		self := entryPath == filepath.Join(directory, "RECORD")
		if row[1] == "" {
			if !self || row[2] != "" {
				return errors.New("PyPI installed RECORD hash is invalid")
			}
		} else {
			if !strings.HasPrefix(row[1], "sha256=") {
				return errors.New("PyPI installed RECORD hash is invalid")
			}
			digest, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(row[1], "sha256="))
			if err != nil || len(digest) != sha256.Size {
				return errors.New("PyPI installed RECORD hash is invalid")
			}
			body, readErr := os.ReadFile(entryPath)
			sum := sha256.Sum256(body)
			if readErr != nil || !infoRegular(entryPath) || !bytes.Equal(sum[:], digest) {
				return errors.New("PyPI installed RECORD content hash is invalid")
			}
		}
		if row[2] != "" {
			size, sizeErr := strconv.ParseInt(row[2], 10, 64)
			info, infoErr := os.Lstat(entryPath)
			if sizeErr != nil || size < 0 || infoErr != nil || !info.Mode().IsRegular() || info.Size() != size {
				return errors.New("PyPI installed RECORD size is invalid")
			}
		}
		destination.Distribution = expected.name
		destination.Version = expected.version
		destination.ArtifactDigest = expected.digest
		fingerprint, err := snapshotPyPIFile(entryPath)
		if err != nil {
			return errors.New("PyPI installed output identity is unsafe")
		}
		if row[1] != "" {
			declared, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(row[1], "sha256="))
			if err != nil || hex.EncodeToString(declared) != fingerprint.Digest {
				return errors.New("PyPI RECORD output changed before destination binding")
			}
		}
		if row[2] != "" {
			size, err := strconv.ParseInt(row[2], 10, 64)
			if err != nil || size != fingerprint.Size {
				return errors.New("PyPI RECORD size changed before destination binding")
			}
		}
		destination.Digest, destination.Size = fingerprint.Digest, fingerprint.Size
		*destinations = append(*destinations, destination)
		count++
	}
	if count == 0 {
		return errors.New("PyPI installed RECORD is empty")
	}
	return nil
}

// resolveInstalledRecordPath implements the bounded HAA install scheme.  It
// does not treat parent traversal as inherently trusted: normal entries are
// rooted at site-packages, while the only permitted cross-scheme entries are
// scripts and data under the transaction's private bin and share roots.
func resolveInstalledRecordPath(site, distInfo, value string) (pypiDestination, error) {
	if value == "" || strings.ContainsRune(value, '\x00') || strings.Contains(value, "\\") ||
		strings.HasPrefix(value, "/") || path.IsAbs(value) || (len(value) >= 2 && value[1] == ':') {
		return pypiDestination{}, errors.New("PyPI installed RECORD path is invalid")
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." {
			return pypiDestination{}, errors.New("PyPI installed RECORD path is invalid")
		}
	}
	parents := 0
	for parents < len(parts) && parts[parents] == ".." {
		parents++
	}
	for _, part := range parts[parents:] {
		if part == ".." {
			return pypiDestination{}, errors.New("PyPI installed RECORD path is invalid")
		}
	}
	var candidate, root string
	scheme := "site"
	switch parents {
	case 0:
		candidate, root = filepath.Join(site, filepath.FromSlash(value)), site
	case 1:
		if len(parts) == 1 {
			return pypiDestination{}, errors.New("PyPI installed RECORD path is invalid")
		}
		candidate, root = filepath.Join(distInfo, filepath.FromSlash(strings.Join(parts, "/"))), site
	case 2:
		if len(parts) < 3 || (parts[2] != "bin" && parts[2] != "share") {
			return pypiDestination{}, errors.New("PyPI installed RECORD path is invalid")
		}
		candidate = filepath.Join(distInfo, filepath.FromSlash(strings.Join(parts, "/")))
		root = filepath.Join(filepath.Dir(site), parts[2])
		if parts[2] == "bin" {
			scheme = "scripts"
		} else {
			scheme = "data"
		}
	default:
		return pypiDestination{}, errors.New("PyPI installed RECORD path is invalid")
	}
	candidate = filepath.Clean(candidate)
	root = filepath.Clean(root)
	if !pathWithin(root, candidate) {
		return pypiDestination{}, errors.New("PyPI installed RECORD escapes transaction")
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return pypiDestination{}, err
	}
	return pypiDestination{Scheme: scheme, Relative: filepath.ToSlash(relative), Source: candidate}, nil
}

func pathWithin(root, value string) bool {
	relative, err := filepath.Rel(root, value)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func validateRecordedOutputRoots(site, transactionRoot string, recorded map[string]bool) error {
	entries, err := os.ReadDir(transactionRoot)
	if err != nil {
		return errors.New("private output root unavailable")
	}
	for _, entry := range entries {
		switch entry.Name() {
		case filepath.Base(site), "bin", "share", "wheels", manifestFilename, sbomFilename, "requirements.txt":
			// These are the exact scheme roots and preparePyPIProject inputs.
		default:
			return errors.New("private output outside the controlled installation scheme")
		}
	}
	for _, root := range []string{site, filepath.Join(transactionRoot, "bin"), filepath.Join(transactionRoot, "share")} {
		if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil || rejectSymlinkPath(root) != nil {
			return errors.New("PyPI Promotion output root is unsafe")
		}
		if err := filepath.WalkDir(root, func(value string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || !recorded[filepath.Clean(value)] {
				return errors.New("PyPI Promotion output is unrecorded or unsafe")
			}
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() {
				return errors.New("PyPI Promotion output contains special file")
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func infoRegular(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func parseInstalledMetadata(body []byte) (string, string, error) {
	if len(body) == 0 || !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return "", "", errors.New("installed metadata is invalid")
	}
	var name, version string
	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		at := strings.IndexByte(line, ':')
		if at <= 0 {
			return "", "", errors.New("installed metadata header is invalid")
		}
		key := strings.ToLower(strings.TrimSpace(line[:at]))
		val := strings.TrimSpace(line[at+1:])
		switch key {
		case "name":
			if name != "" {
				return "", "", errors.New("installed metadata Name is duplicated")
			}
			normalized, err := artifactpypi.NormalizeProjectName(val)
			if err != nil {
				return "", "", err
			}
			name = normalized
		case "version":
			if version != "" {
				return "", "", errors.New("installed metadata Version is duplicated")
			}
			normalized, err := artifactpypi.NormalizeInstalledVersion(val)
			if err != nil {
				return "", "", err
			}
			version = normalized
		}
	}
	if name == "" || version == "" {
		return "", "", errors.New("installed metadata missing Name or Version")
	}
	return name, version, nil
}

func readBoundedPromotionFile(filename string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, errors.New("promotion file limit is invalid")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(body)) > limit {
		return nil, errors.New("promotion file exceeds metadata bound")
	}
	return body, nil
}
