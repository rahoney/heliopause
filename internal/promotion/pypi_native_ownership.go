package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type pypiNativeEnrollment struct {
	resolver   ports.DependencyResolver
	intake     ports.Artifact
	intakeRoot string
	runner     DockerRunner
	client     pypiNativeClient
}

// WithNativeEnrollment adds only the concrete old-footprint collaborators to an
// immutable copy. New selection still uses the original guarded update resolver.
func (p *PyPIUpdatePreparation) WithNativeEnrollment(resolver ports.DependencyResolver, intake ports.Artifact, intakeRoot string) (*PyPIUpdatePreparation, error) {
	if p == nil || p.promoter == nil || p.resolver == nil || resolver == nil || intake == nil || !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || !separateRoots([]string{intakeRoot, p.promoter.stagingRoot}) {
		return nil, errors.New("native PyPI enrollment requires separate trusted intake and existing source capabilities")
	}
	copy := *p
	copy.native = &pypiNativeEnrollment{resolver: resolver, intake: intake, intakeRoot: intakeRoot, runner: p.promoter.runner}
	return &copy, nil
}

type pypiNativeProofFile struct {
	Scheme   string `json:"scheme"`
	Relative string `json:"relative"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Role     string `json:"ownership_basis"`
}

type pypiNativeProof struct {
	Source   string                `json:"source"`
	Project  string                `json:"project"`
	Version  string                `json:"version"`
	Filename string                `json:"filename"`
	SHA256   string                `json:"wheel_sha256"`
	Files    []pypiNativeProofFile `json:"files"`
}

type pypiNativeEnrollmentSelection struct {
	destinations      []pypiDestination
	control           domain.ProjectControlFile
	absentDirectories []string
}

func (e *pypiNativeEnrollment) selectOwnership(ctx context.Context, graph domain.LockedDependencyGraph, install domain.InstallContext, inventory pypiNativeInventory) (selection pypiNativeEnrollmentSelection, resultErr error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	defer func() {
		if err := ctx.Err(); err != nil {
			selection = pypiNativeEnrollmentSelection{}
			resultErr = errors.Join(resultErr, err)
		}
	}()
	proofs := []pypiNativeProof{}
	var runtime *pypiNativeRuntime
	loadRuntime := func() (pypiNativeRuntime, error) {
		if runtime == nil {
			value, err := e.runtimeMetadata(ctx)
			if err != nil {
				return pypiNativeRuntime{}, err
			}
			runtime = &value
		}
		return *runtime, nil
	}
	for _, node := range graph.Nodes() {
		project := inventory.projects[node.Artifact().Identity().Name()]
		if project == nil {
			continue
		}
		if err := ctx.Err(); err != nil {
			return selection, err
		}
		if node.Artifact().Identity().Source() != artifactpypi.PublicPyPIProfile().Source() {
			return selection, errors.New("native source ownership cannot be redirected")
		}
		wheel, filename, digest, err := e.oldWheel(ctx, install, project)
		if err != nil {
			return selection, err
		}
		destinations, files, err := proveNativePythonWheelFootprint(e.client, project, wheel, digest, loadRuntime)
		if err != nil {
			return selection, err
		}
		if e.client == nativeUVClient {
			paths, err := nativeUVCacheAbsences(project, wheel)
			if err != nil {
				return selection, err
			}
			selection.absentDirectories = append(selection.absentDirectories, paths...)
		}
		selection.destinations = append(selection.destinations, destinations...)
		proofs = append(proofs, pypiNativeProof{Source: "pypi", Project: project.project, Version: project.version, Filename: filename, SHA256: digest, Files: files})
	}
	if len(proofs) == 0 {
		return selection, nil
	}
	sort.Slice(proofs, func(i, j int) bool { return proofs[i].Project < proofs[j].Project })
	body, err := json.Marshal(struct {
		Version int               `json:"version"`
		Proofs  []pypiNativeProof `json:"proofs"`
	}{1, proofs})
	if err != nil || len(body) > 4<<20 {
		return pypiNativeEnrollmentSelection{}, errors.New("native ownership proof exceeds its control bound")
	}
	selection.control, err = domain.NewProjectControlFile("pypi-update-native-ownership.json", body, true)
	return selection, err
}

func (e *pypiNativeEnrollment) oldWheel(ctx context.Context, install domain.InstallContext, project *pypiNativeDistribution) (inspection artifactpypi.WheelInspection, filename, digest string, resultErr error) {
	reference, err := artifactpypi.ParseReference(project.project + "@" + project.version)
	if err != nil {
		return inspection, "", "", errors.New("native original version is unsupported by public PyPI")
	}
	resolution, err := e.resolver.ResolveDependencies(ctx, reference, install)
	if err != nil {
		return inspection, "", "", err
	}
	runtime := sandbox.PinnedPythonRuntime()
	if resolution.RuntimeIdentity() != nativeLookupRuntimeIdentity(e.client) || nativeLookupRuntimeIdentity(e.client) == "" {
		return inspection, "", "", errors.New("native original lookup has a different runtime")
	}
	var resolved domain.ResolvedArtifact
	for _, node := range resolution.Graph().Nodes() {
		if node.Node() == resolution.Graph().Primary() {
			resolved = node.Artifact()
			filename = node.RecordPath()
		}
	}
	id := resolved.Identity()
	if id.Source() != reference.Source() || id.Name() != project.project || id.Version() != project.version || id.Variant() != "wheel" || filepath.Base(filename) != filename {
		return inspection, "", "", errors.New("native original lookup did not select the exact public wheel")
	}
	digest = strings.TrimPrefix(resolved.DeclaredIntegrity(), "sha256:")
	if _, err := domain.NewSHA256Digest(digest); err != nil || resolved.DeclaredIntegrity() != "sha256:"+digest {
		return inspection, "", "", errors.New("native original wheel lacks canonical integrity")
	}
	if err := ensureTrustedRoot(e.intakeRoot); err != nil {
		return inspection, "", "", err
	}
	run, err := domain.NewRunID()
	if err != nil {
		return inspection, "", "", err
	}
	directory := filepath.Join(e.intakeRoot, run.String())
	if _, err := os.Lstat(directory); !errors.Is(err, os.ErrNotExist) {
		return inspection, "", "", errors.New("native original intake scope already exists")
	}
	acquired, err := e.intake.Acquire(ctx, run, resolved)
	if err != nil {
		return inspection, "", "", err
	}
	identity, err := os.Lstat(directory)
	if err != nil || !identity.IsDir() || trustedExistingDirectory(directory) != nil {
		return inspection, "", "", errors.New("native original intake directory is untrusted")
	}
	defer func() {
		now, err := os.Lstat(directory)
		if err != nil || !now.IsDir() || !os.SameFile(identity, now) || trustedExistingDirectory(directory) != nil {
			resultErr = errors.Join(resultErr, errors.New("native original intake identity changed before cleanup"))
			return
		}
		resultErr = errors.Join(resultErr, os.RemoveAll(directory))
	}()
	declared, present := acquired.DeclaredIntegrity()
	limits := artifactpypi.DefaultWheelLimits()
	if acquired.Identity() != id || acquired.ContentHandle() != "intake:"+run.String()+":wheel" || !present || declared != resolved.DeclaredIntegrity() || acquired.Digest().String() != digest || acquired.SizeBytes() == 0 || acquired.SizeBytes() > uint64(limits.MaxCompressed) {
		return inspection, "", "", errors.New("native original acquisition differs from the resolved wheel")
	}
	namePath := filepath.Join(directory, "filename")
	if _, err := snapshotBoundedPromotionFile(namePath, 255); err != nil {
		return inspection, "", "", err
	}
	name, err := readBoundedPromotionFile(namePath, 255)
	if err != nil || string(name) != filename {
		return inspection, "", "", errors.New("native original intake filename changed")
	}
	filePath := filepath.Join(directory, "wheel.whl")
	before, err := snapshotBoundedPromotionFile(filePath, limits.MaxCompressed)
	if err != nil || before.Digest != digest || before.Size != int64(acquired.SizeBytes()) {
		return inspection, "", "", errors.New("native original intake bytes differ from authenticated content")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return inspection, "", "", err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	inspection, err = artifactpypi.InspectWheel(file, before.Size, filename, digest, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}, limits)
	if err != nil {
		return inspection, "", "", err
	}
	if !before.matches(filePath) || inspection.Project != project.project || inspection.Version != project.version {
		return inspection, "", "", errors.New("native original footprint or intake identity changed")
	}
	return inspection, filename, digest, ctx.Err()
}

func proveNativePyPIWheelFootprint(project *pypiNativeDistribution, wheel artifactpypi.WheelInspection, digest string, loadRuntime func() (pypiNativeRuntime, error)) ([]pypiDestination, []pypiNativeProofFile, error) {
	return proveNativePythonWheelFootprint(nativePipClient, project, wheel, digest, loadRuntime)
}

func proveNativePythonWheelFootprint(client pypiNativeClient, project *pypiNativeDistribution, wheel artifactpypi.WheelInspection, digest string, loadRuntime func() (pypiNativeRuntime, error)) ([]pypiDestination, []pypiNativeProofFile, error) {
	if nativeClientName(client) == "" {
		return nil, nil, errors.New("native installer format is unsupported")
	}
	if client == nativeUVClient {
		for name, body := range map[string]string{"INSTALLER": "uv", "REQUESTED": ""} {
			claim, ok := project.files["site/"+project.directory+"/"+name]
			// UV only emits REQUESTED for directly requested distributions.
			// It is optional for dependencies and is never ownership authority.
			if !ok && name == "REQUESTED" {
				continue
			}
			sum := sha256.Sum256([]byte(body))
			if !ok || claim.unhashed || claim.original.Size != int64(len(body)) || claim.original.Digest != hex.EncodeToString(sum[:]) {
				return nil, nil, errors.New("native UV metadata differs from supported installer output")
			}
		}
	}

	if wheel.Project != project.project || wheel.Version != project.version || wheel.ObservedSHA256 != digest || wheel.DeclaredSHA256 != digest {
		return nil, nil, errors.New("native footprint is not bound to the authentic original wheel")
	}
	for _, entry := range wheel.Surface.EntryPointDetails {
		if entry.Group == "console_scripts" || entry.Group == "gui_scripts" {
			return nil, nil, errors.New("native generated entry-point ownership is unsupported")
		}
	}
	matched := map[string]string{}
	pythonSources := map[string]pypiNativeFile{}
	selfKey := "site/" + project.directory + "/RECORD"
	for _, file := range wheel.Surface.InstalledFiles {
		if file.Scheme != artifactpypi.SchemeSite && file.Scheme != artifactpypi.SchemeDistInfo {
			return nil, nil, errors.New("native cross-scheme wheel ownership is unsupported")
		}
		key := "site/" + file.Destination
		claim, ok := project.files[key]
		if !ok || matched[key] != "" {
			return nil, nil, errors.New("native original wheel footprint is incomplete or ambiguous")
		}
		if key == selfKey {
			if !claim.unhashed {
				return nil, nil, errors.New("native RECORD self ownership is invalid")
			}
			matched[key] = "validated-installed-record"
			continue
		}
		if claim.unhashed || file.SHA256 == "" || claim.original.Digest != file.SHA256 || claim.original.Size != file.Size {
			return nil, nil, errors.New("native original file differs from canonical wheel bytes")
		}
		matched[key] = "authenticated-wheel-file"
		if file.Scheme == artifactpypi.SchemeSite && strings.HasSuffix(file.Destination, ".py") && (file.Role == artifactpypi.RuntimeRolePythonModule || file.Role == artifactpypi.RuntimeRolePythonPackage || file.Role == artifactpypi.RuntimeRoleExecutableSubpackage) {
			pythonSources[file.Destination] = claim
		}
	}
	if matched[selfKey] == "" {
		return nil, nil, errors.New("native footprint lacks its installed RECORD")
	}
	for key, claim := range project.files {
		if matched[key] != "" {
			continue
		}
		switch key {
		case "site/" + project.directory + "/INSTALLER":
			sum := sha256.Sum256([]byte(nativeInstallerMarker(client)))
			if claim.unhashed || claim.original.Size != int64(len(nativeInstallerMarker(client))) || claim.original.Digest != hex.EncodeToString(sum[:]) {
				return nil, nil, errors.New("native pip installer marker differs from pinned output")
			}
			matched[key] = "pinned-" + nativeClientName(client) + "-installer-marker"
		case "site/" + project.directory + "/REQUESTED":
			sum := sha256.Sum256(nil)
			if claim.unhashed || claim.original.Size != 0 || claim.original.Digest != hex.EncodeToString(sum[:]) {
				return nil, nil, errors.New("native pip requested marker differs from pinned output")
			}
			matched[key] = "pinned-" + nativeClientName(client) + "-requested-marker"
		default:
			if client == nativeUVClient {
				return nil, nil, errors.New("native UV extra cache, build, script or source files are unsupported")
			}
			if claim.destination.Scheme != "site" || !strings.HasSuffix(claim.destination.Relative, ".pyc") {
				return nil, nil, errors.New("native RECORD claims a file outside the authentic installed footprint")
			}
			runtime, err := loadRuntime()
			if err != nil {
				return nil, nil, err
			}
			found := false
			for sourcePath, source := range pythonSources {
				cache := path.Join(path.Dir(sourcePath), "__pycache__", strings.TrimSuffix(path.Base(sourcePath), ".py")+"."+runtime.cacheTag+".pyc")
				if claim.destination.Relative != cache {
					continue
				}
				if err := validateNativeTimestampCache(claim, source, runtime); err != nil {
					return nil, nil, err
				}
				found = true
				break
			}
			if !found {
				return nil, nil, errors.New("native bytecode claim has no authentic source/cache role")
			}
			matched[key] = "pinned-timestamp-cache-removal-only"
		}
	}
	keys := make([]string, 0, len(project.files))
	for key := range project.files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	destinations := make([]pypiDestination, 0, len(keys))
	files := make([]pypiNativeProofFile, 0, len(keys))
	for _, key := range keys {
		claim := project.files[key]
		destination := claim.destination
		destination.ArtifactDigest = digest
		destinations = append(destinations, destination)
		files = append(files, pypiNativeProofFile{destination.Scheme, destination.Relative, destination.Digest, destination.Size, matched[key]})
	}
	return destinations, files, nil
}

// Cache bytes are never loaded or approved. The verified original source and
// pinned runtime relationship authorize removal only at this exact cache role.
func validateNativeTimestampCache(cache, source pypiNativeFile, runtime pypiNativeRuntime) error {
	body, err := readBoundedPromotionFile(cache.destination.Final, nativePyPIBytecodeLimit)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	if len(body) < 16 || int64(len(body)) != cache.original.Size || hex.EncodeToString(sum[:]) != cache.original.Digest || hex.EncodeToString(body[:4]) != runtime.magic || binary.LittleEndian.Uint32(body[4:8]) != 0 || binary.LittleEndian.Uint32(body[8:12]) != uint32(source.original.info.ModTime().Unix()) || binary.LittleEndian.Uint32(body[12:16]) != uint32(source.original.Size) {
		return errors.New("native bytecode does not have the pinned original timestamp/cache relationship")
	}
	return nil
}
