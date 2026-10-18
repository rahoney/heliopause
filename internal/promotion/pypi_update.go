package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// PyPIUpdatePreparation retains the existing venv transaction from before
// selection through inspection and exact offline publication.
type PyPIUpdatePreparation struct {
	promoter *PyPIPromotion
	resolver ports.DependencyUpdateResolver
	native   *pypiNativeEnrollment
}

func NewPyPIUpdatePreparation(promoter *PyPIPromotion, resolver ports.DependencyUpdateResolver) (*PyPIUpdatePreparation, error) {
	if promoter == nil || resolver == nil {
		return nil, errors.New("PyPI upgrade requires existing Promotion and isolated selection")
	}
	return &PyPIUpdatePreparation{promoter: promoter, resolver: resolver}, nil
}

type pypiUpdateSession struct {
	owner                      *PyPIUpdatePreparation
	install                    domain.InstallContext
	input                      artifactpypi.UpgradeInput
	transaction                *pypiVenvTransaction
	directories                []string
	metadata                   map[string]pypiFileSnapshot
	managedVersions            map[string]string
	emptyMetadata              map[string]bool
	nativeInventory            pypiNativeInventory
	adopt                      bool
	resolution                 domain.DependencyResolution
	resolved, promoted, closed bool
	closeErr                   error
}

func (p *PyPIUpdatePreparation) BeginInstallUpdate(ctx context.Context, source domain.SourceID, selector string, install domain.InstallContext, adopt bool) (result ports.InstallUpdateSession, resultErr error) {
	profile, supported := artifactpypi.ProfileForSource(source)
	if p == nil || ctx == nil || !install.Valid() || install.Mode() != domain.InstallPythonVenv || !supported || p.promoter.goos != "linux" || p.promoter.goarch != "amd64" {
		return nil, errors.New("PyPI upgrade requires a supported existing Linux amd64 virtual environment")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if artifactpypi.IsPyTorchSource(source) && artifactpypi.RootSourceProfileNameFromContext(ctx) != profile.Name() {
		return nil, errors.New("named upgrade requires its root resource context")
	}
	requirement, err := artifactpypi.ParseUpgradeRequirementForSource(selector, source)
	if err != nil {
		return nil, err
	}
	if strings.Contains(install.Target().String(), ",") {
		return nil, errors.New("PyPI upgrade path is unsupported by the Docker mount boundary")
	}
	if p.native != nil && !separateRoots([]string{install.Target().String(), p.native.intakeRoot, p.promoter.stagingRoot}) {
		return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", errors.New("virtual environment overlaps trusted intake or staging"))
	}
	plan, err := discoverPythonVenv(install.Target().String())
	if err != nil {
		return nil, err
	}
	transaction, err := beginPyPIVenvTransaction(plan)
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
	}
	s := &pypiUpdateSession{owner: p, install: install, transaction: transaction, metadata: map[string]pypiFileSnapshot{}, emptyMetadata: map[string]bool{}}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, s.Close())
			result = nil
		}
	}()
	if transaction.metadata == nil && (!adopt || p.native == nil) {
		code := "UPDATE_ADOPTION_REQUIRED"
		if adopt {
			code = "UPDATE_ORIGINAL_STATE_UNSUPPORTED"
		}
		return nil, ports.NewInstallUpdateFailure(code, errors.New("native venv ownership requires independent installed-footprint verification"))
	}
	s.directories, err = pypiUpgradeMetadataDirectories(plan.site)
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
	}
	files := []artifactpypi.InstalledMetadataFile{}
	total := 0
	for _, name := range s.directories {
		directory := filepath.Join(plan.site, name)
		if err := transaction.freezeParents(directory); err != nil {
			return nil, err
		}
		empty, err := pypiUpgradeEmptyDirectory(directory)
		if err != nil {
			return nil, err
		}
		if empty {
			// The existing transaction removes obsolete owned files while
			// retaining directories. An empty directory is no distribution;
			// retain its identity and emptiness guard, never infer ownership.
			s.emptyMetadata[directory] = true
			continue
		}
		if err := rejectPyPIUpgradeDirectSource(directory); err != nil {
			return nil, err
		}
		path := filepath.Join(directory, "METADATA")
		snapshot, err := snapshotBoundedPromotionFile(path, 1<<20)
		if err != nil {
			return nil, err
		}
		body, err := readBoundedPromotionFile(path, 1<<20)
		if err != nil {
			return nil, err
		}
		total += len(body)
		if total > 4<<20 {
			return nil, errors.New("PyPI installed metadata exceeds upgrade input bound")
		}
		sum := sha256.Sum256(body)
		if snapshot.Digest != hex.EncodeToString(sum[:]) {
			return nil, errors.New("PyPI installed metadata changed while reading")
		}
		s.metadata[path] = snapshot
		files = append(files, artifactpypi.InstalledMetadataFile{Directory: name, Body: body})
	}
	s.input, err = artifactpypi.NewUpgradeInput(requirement, files, sandbox.PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
	}
	s.managedVersions, err = managedPyPIInventory(transaction, s.input)
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
	}
	if adopt && p.native != nil {
		s.nativeInventory, err = capturePyPINativeInventory(ctx, transaction, s.input)
		if err != nil {
			return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
		}
		s.adopt = true
	}
	if err := s.verify(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// This bounds allocation as well as parsing. Legacy/editable metadata is not
// silently omitted from the installed dependency constraints.
func pypiUpgradeMetadataDirectories(site string) ([]string, error) {
	file, err := os.Open(site)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	entries, err := file.ReadDir(100001)
	if err != nil && err != io.EOF || len(entries) > 100000 {
		return nil, errors.New("PyPI environment inventory exceeds supported bounds")
	}
	names := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".egg-info") || strings.HasSuffix(name, ".egg-link") {
			return nil, errors.New("legacy or editable installed metadata is unsupported")
		}
		if !strings.HasSuffix(name, ".dist-info") {
			continue
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil, errors.New("installed metadata directory is untrusted")
		}
		names = append(names, name)
		if len(names) > 1024 {
			return nil, errors.New("PyPI installed distribution count exceeds supported bounds")
		}
	}
	sort.Strings(names)
	return names, nil
}

func rejectPyPIUpgradeDirectSource(directory string) error {
	if _, err := os.Lstat(filepath.Join(directory, "direct_url.json")); !errors.Is(err, os.ErrNotExist) {
		return errors.New("installed direct-source or editable selection is unsupported")
	}
	return nil
}

func pypiUpgradeEmptyDirectory(directory string) (bool, error) {
	if trustedExistingDirectory(directory) != nil {
		return false, errors.New("untrusted installed metadata directory")
	}
	file, err := os.Open(directory)
	if err != nil {
		return false, err
	}
	entries, readErr := file.ReadDir(1)
	closeErr := file.Close()
	if readErr != nil && readErr != io.EOF || closeErr != nil {
		return false, errors.Join(readErr, closeErr)
	}
	return len(entries) == 0, nil
}

func (s *pypiUpdateSession) Reference() domain.ArtifactReference {
	return s.input.Requirement().Reference()
}

func (s *pypiUpdateSession) verify(ctx context.Context) error {
	if s.closed || s.promoted || ctx == nil {
		return errors.New("PyPI upgrade original-state guard is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.transaction.checkBefore(); err != nil {
		return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", err)
	}
	names, err := pypiUpgradeMetadataDirectories(s.transaction.plan.site)
	if err != nil || !reflect.DeepEqual(names, s.directories) {
		return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", errors.New("installed distribution inventory changed"))
	}
	for directory := range s.emptyMetadata {
		empty, err := pypiUpgradeEmptyDirectory(directory)
		if err != nil || !empty {
			return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", errors.New("retired metadata directory gained contents"))
		}
	}
	for path, before := range s.metadata {
		now, err := snapshotBoundedPromotionFile(path, 1<<20)
		if err != nil || !os.SameFile(before.info, now.info) || before.Digest != now.Digest || before.Size != now.Size || before.Mode != now.Mode || rejectPyPIUpgradeDirectSource(filepath.Dir(path)) != nil {
			return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", errors.New("installed dependency metadata changed"))
		}
	}
	return ctx.Err()
}

func (s *pypiUpdateSession) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	if s.resolved || reference != s.Reference() || install != s.install {
		return domain.DependencyResolution{}, errors.New("PyPI upgrade selection changed or was repeated")
	}
	if err := s.verify(ctx); err != nil {
		return domain.DependencyResolution{}, err
	}
	control, err := s.input.Control()
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	resolution, err := s.owner.resolver.ResolveDependencyUpdate(ctx, reference, install, []domain.ProjectControlFile{control})
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	runtime := sandbox.PinnedPythonRuntime()
	var graph domain.LockedDependencyGraph
	if artifactpypi.IsPyTorchSource(reference.Source()) {
		selection, err := s.input.ParseSourcePinnedUpgrade(resolution.ProjectControls(), runtime.PipVersion)
		if err != nil {
			return domain.DependencyResolution{}, err
		}
		graph = selection.Graph()
		profile, _ := artifactpypi.ProfileForSource(reference.Source())
		expectedRuntime := "python:" + runtime.PythonVersion + ";pip:" + runtime.PipVersion + ";target:" + runtime.InterpreterTag + "/" + runtime.ABITag + "/" + runtime.PlatformTag + ";source:" + profile.Name()
		if !reflect.DeepEqual(graph, resolution.Graph()) || resolution.RuntimeIdentity() != expectedRuntime || resolution.LockfileDigest() != selection.Digest() {
			return domain.DependencyResolution{}, errors.New("named selected graph/runtime/digest does not bind complete controls")
		}
	} else {
		selected := map[string][]byte{}
		for _, item := range resolution.ProjectControls() {
			if !item.Present() || selected[item.Name()] != nil {
				return domain.DependencyResolution{}, errors.New("PyPI selected controls are invalid")
			}
			selected[item.Name()] = item.Body()
		}
		if len(selected) != 3 || !bytes.Equal(selected[artifactpypi.UpgradeInputControlName], control.Body()) {
			return domain.DependencyResolution{}, errors.New("PyPI resolver changed original selection input")
		}
		proposal, err := s.input.ParseUpgradeDelta(selected["pypi-update-delta.json"], runtime.PipVersion)
		if err != nil {
			return domain.DependencyResolution{}, err
		}
		report, err := artifactpypi.ParseInstallationReport(proposal.Reference(), selected["pypi-update-report.json"], runtime.PipVersion, runtime.PythonVersion)
		if err != nil {
			return domain.DependencyResolution{}, err
		}
		if err := proposal.ValidateFinal(report); err != nil {
			return domain.DependencyResolution{}, err
		}
		graph, err = artifactpypi.BuildLockedGraph(reference, report.Candidates())
		if err != nil {
			return domain.DependencyResolution{}, err
		}
		sum := sha256.Sum256(selected["pypi-update-report.json"])
		expectedRuntime := "python:" + runtime.PythonVersion + ";pip:" + runtime.PipVersion + ";target:" + runtime.InterpreterTag + "/" + runtime.ABITag + "/" + runtime.PlatformTag + ";source:" + artifactpypi.PublicPyPIProfile().Name()
		if !reflect.DeepEqual(graph, resolution.Graph()) || resolution.RuntimeIdentity() != expectedRuntime || resolution.LockfileDigest().String() != hex.EncodeToString(sum[:]) {
			return domain.DependencyResolution{}, errors.New("PyPI selected graph/runtime/digest does not bind complete controls")
		}
	}
	// Existing native files are retained as constraints, never promoted to HAA
	// ownership merely because an untrusted RECORD names their paths.
	installed := map[string]string{}
	for _, project := range s.input.Installed() {
		installed[project.Project()] = project.Version()
	}
	unmanaged := false
	for _, node := range graph.Nodes() {
		name := node.Artifact().Identity().Name()
		if version := installed[name]; version != "" && s.managedVersions[name] != version {
			if !s.adopt {
				return domain.DependencyResolution{}, ports.NewInstallUpdateFailure("UPDATE_ADOPTION_REQUIRED", errors.New("selected native distribution requires explicit independent enrollment"))
			}
			if s.owner.native == nil || s.nativeInventory.projects[name] == nil {
				return domain.DependencyResolution{}, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", errors.New("selected native distribution has no independently verified ownership"))
			}
			unmanaged = true
		}
	}
	if unmanaged {
		selection, err := s.owner.native.selectOwnership(ctx, graph, install, s.nativeInventory)
		if err != nil {
			return domain.DependencyResolution{}, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
		}
		if len(selection.destinations) == 0 || !selection.control.Present() {
			return domain.DependencyResolution{}, errors.New("native selection has no original ownership proof")
		}
		resolution, err = resolution.WithProjectControls(append(resolution.ProjectControls(), selection.control))
		if err != nil {
			return domain.DependencyResolution{}, err
		}
		if err := s.verify(ctx); err != nil {
			return domain.DependencyResolution{}, err
		}
		for _, destination := range selection.destinations {
			if _, present := s.transaction.current.Files[destination.key()]; present {
				return domain.DependencyResolution{}, errors.New("native ownership proof overlaps an existing owner")
			}
		}
		// These are independently proven old files, not raw RECORD claims.
		// The on-disk ledger is still absent/unchanged until full new-set ALLOW.
		for _, destination := range selection.destinations {
			s.transaction.current.Files[destination.key()] = destination
		}
	}
	if err := s.verify(ctx); err != nil {
		return domain.DependencyResolution{}, err
	}
	s.resolved, s.resolution = true, resolution
	return resolution, nil
}

func (s *pypiUpdateSession) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	if !s.resolved || s.promoted || install != s.install || !bundle.Valid() || !pypiUpgradeGraphExtends(s.resolution.Graph(), bundle.Set().Inspected()) {
		return domain.PromotedInstall{}, errors.New("PyPI upgrade lacks the exact inspected selection")
	}
	if err := s.verify(ctx); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := verifyDocuments(bundle); err != nil {
		return domain.PromotedInstall{}, err
	}
	var metadata struct {
		Resolver struct {
			Runtime string `json:"runtime_identity"`
			Lock    string `json:"lockfile_sha256"`
		} `json:"resolver"`
		Target struct {
			Path string `json:"path"`
		} `json:"target"`
		Controls []struct {
			Name    string `json:"name"`
			Content []byte `json:"content"`
		} `json:"project_controls"`
	}
	if json.Unmarshal(bundle.ManifestDocument(), &metadata) != nil || metadata.Target.Path != install.Target().String() || metadata.Resolver.Lock != s.resolution.LockfileDigest().String() || metadata.Resolver.Runtime != s.resolution.RuntimeIdentity() || len(metadata.Controls) != len(s.resolution.ProjectControls()) {
		return domain.PromotedInstall{}, errors.New("PyPI upgrade Manifest belongs to another selection")
	}
	expected := map[string][]byte{}
	for _, control := range s.resolution.ProjectControls() {
		expected[control.Name()] = control.Body()
	}
	for _, control := range metadata.Controls {
		if expected[control.Name] == nil || !bytes.Equal(expected[control.Name], control.Content) {
			return domain.PromotedInstall{}, errors.New("PyPI upgrade Manifest control differs from selection")
		}
		delete(expected, control.Name)
	}
	if len(expected) != 0 {
		return domain.PromotedInstall{}, errors.New("PyPI upgrade Manifest controls are incomplete")
	}
	promoted, err := s.owner.promoter.promote(ctx, staged, bundle, install, s.transaction, s.verify)
	if err == nil {
		s.promoted = true
	}
	return promoted, err
}

// Keep every resolver-owned node/edge and allow only the existing authenticated
// sdist-to-derived-wheel extension. Build inputs retain their existing binding;
// they are not incorrectly required to be resolver graph nodes.
func pypiUpgradeGraphExtends(original domain.LockedDependencyGraph, inspected domain.InspectedDependencySet) bool {
	graph := inspected.Graph()
	if original.Primary() != graph.Primary() || len(original.Derivations()) != 0 {
		return false
	}
	known := map[domain.DependencyNodeID]domain.LockedDependency{}
	for _, node := range original.Nodes() {
		known[node.Node()] = node
	}
	extra := map[domain.DependencyNodeID]domain.LockedDependency{}
	for _, node := range graph.Nodes() {
		if before, ok := known[node.Node()]; ok {
			if before != node {
				return false
			}
			delete(known, node.Node())
		} else {
			extra[node.Node()] = node
		}
	}
	if len(known) != 0 || len(extra) != len(graph.Derivations()) {
		return false
	}
	edges := map[domain.DependencyEdge]bool{}
	for _, edge := range original.Edges() {
		edges[edge] = true
	}
	seenSource := map[domain.DependencyNodeID]bool{}
	for _, binding := range graph.Derivations() {
		derived, ok := extra[binding.Derived()]
		if !ok || seenSource[binding.Source()] {
			return false
		}
		source, found := findInspectionFromSet(inspected, binding.Source())
		id := derived.Artifact().Identity()
		if !found || source.Artifact().Identity().Variant() != "sdist" || id.Variant() != "derived-wheel" || id.Source() != source.Artifact().Identity().Source() || id.Name() != source.Artifact().Identity().Name() || id.Version() != source.Artifact().Identity().Version() || binding.Binding().SourceDigest() != source.Artifact().Digest() {
			return false
		}
		edge, err := domain.NewDependencyEdge(binding.Source(), binding.Derived())
		if err != nil {
			return false
		}
		edges[edge] = true
		seenSource[binding.Source()] = true
		delete(extra, binding.Derived())
	}
	if len(extra) != 0 || len(edges) != len(graph.Edges()) {
		return false
	}
	for _, edge := range graph.Edges() {
		if !edges[edge] {
			return false
		}
	}
	return true
}

func findInspectionFromSet(set domain.InspectedDependencySet, node domain.DependencyNodeID) (domain.DependencyInspection, bool) {
	for _, item := range set.Inspections() {
		if item.Node() == node {
			return item, true
		}
	}
	return domain.DependencyInspection{}, false
}

func (s *pypiUpdateSession) Close() error {
	if !s.closed {
		s.closed = true
		s.closeErr = s.transaction.close()
	}
	return s.closeErr
}
