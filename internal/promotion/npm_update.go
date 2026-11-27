package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type NPMUpdatePreparation struct {
	promoter *NPMPromotion
	resolver ports.DependencyUpdateResolver
}

func NewNPMUpdatePreparation(promoter *NPMPromotion, resolver ports.DependencyUpdateResolver) (*NPMUpdatePreparation, error) {
	if promoter == nil || resolver == nil {
		return nil, errors.New("npm update requires existing Promotion and isolated update resolver")
	}
	return &NPMUpdatePreparation{promoter, resolver}, nil
}

type npmUpdateSession struct {
	owner                      *NPMUpdatePreparation
	install                    domain.InstallContext
	reference                  domain.ArtifactReference
	guard                      npmProjectGuard
	plan                       npmProjectPlan
	controls                   []domain.ProjectControlFile
	resolution                 domain.DependencyResolution
	resolved, promoted, closed bool
	closeErr                   error
}

func (p *NPMUpdatePreparation) BeginInstallUpdate(ctx context.Context, source domain.SourceID, selector string, install domain.InstallContext, adopt bool) (result ports.InstallUpdateSession, resultErr error) {
	if p == nil || ctx == nil || source.String() != "npm" || !install.Valid() || install.Mode() != domain.InstallNPMProject || p.promoter.goos != "linux" || p.promoter.goarch != "amd64" {
		return nil, errors.New("npm update requires a supported Linux amd64 project")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if selector != "" {
		ref, err := artifactnpm.ParseReference(selector)
		if err != nil {
			return nil, errors.New("npm update selector must be one package name")
		}
		name, err := artifactnpm.RequestedPackageName(ref)
		if err != nil || name != selector {
			return nil, errors.New("npm update preserves project ranges; select a package name without a version")
		}
	}
	root := install.Target().String()
	if strings.Contains(root, ",") {
		return nil, errors.New("npm update path is unsupported by the Docker mount boundary")
	}
	if trustedExistingDirectory(root) != nil {
		return nil, errors.New("npm update root is untrusted")
	}
	guard, err := acquireNPMProjectGuard(root)
	if err != nil {
		return nil, err
	}
	s := &npmUpdateSession{owner: p, install: install, guard: guard}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, s.Close())
			result = nil
		}
	}()
	if err := rejectInterruptedNPMTransaction(root); err != nil {
		return nil, err
	}
	s.plan, err = freezeNPMProject(root)
	if err != nil {
		return nil, err
	}
	s.plan.updateGuard = &s.guard
	s.plan.members = map[string]*npmTransactionMember{}
	for _, name := range []string{"node_modules", ".heliopause"} {
		path := filepath.Join(root, name)
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			s.plan.members[name] = nil
			continue
		}
		member, err := snapshotNPMTransactionMember(path)
		if err != nil {
			return nil, err
		}
		s.plan.members[name] = member
	}
	if err := s.plan.verifyManagedOrEmpty(); err != nil {
		// Explicit adoption may replace only a bounded native project without a
		// prior HAA namespace. It never bypasses corrupt/ambiguous owned state.
		if s.plan.members[".heliopause"] != nil {
			return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
		}
		if !adopt {
			return nil, ports.NewInstallUpdateFailure("UPDATE_ADOPTION_REQUIRED", err)
		}
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		body, err := readBoundedPromotionFile(filepath.Join(root, name), artifactnpm.ProjectControlLimit)
		if err != nil {
			return nil, err
		}
		control, err := domain.NewProjectControlFile(name, body, true)
		if err != nil {
			return nil, err
		}
		s.controls = append(s.controls, control)
	}
	s.reference, err = artifactnpm.ProjectRootReference(s.controls[0].Body())
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_UNSUPPORTED_PROJECT", err)
	}
	name, err := artifactnpm.RequestedPackageName(s.reference)
	if err != nil || selector != "" && selector != name {
		return nil, errors.New("npm update selector does not name the supported direct root")
	}
	originalGraph, err := artifactnpm.ParseProjectControls(s.reference, s.controls[0].Body(), s.controls[1].Body())
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_UNSUPPORTED_PROJECT", err)
	}
	if err := validateNPMOriginalPayload(root, originalGraph); err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
	}
	if err := validateNPMOriginalMetadata(s.plan.members[".heliopause"]); err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
	}
	if err := s.verify(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *npmUpdateSession) Reference() domain.ArtifactReference { return s.reference }

func (s *npmUpdateSession) verify(ctx context.Context) error {
	if s.closed || ctx == nil {
		return errors.New("npm update guard is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.plan.verifyUnchanged(); err != nil {
		return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", err)
	}
	return ctx.Err()
}

func (s *npmUpdateSession) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	if s.resolved || reference != s.reference || install != s.install {
		return domain.DependencyResolution{}, errors.New("npm update session selection changed or was repeated")
	}
	if err := s.verify(ctx); err != nil {
		return domain.DependencyResolution{}, err
	}
	resolution, err := s.owner.resolver.ResolveDependencyUpdate(ctx, reference, install, s.controls)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	selected := map[string][]byte{}
	for _, control := range resolution.ProjectControls() {
		if !control.Present() || selected[control.Name()] != nil {
			return domain.DependencyResolution{}, errors.New("npm resolver returned invalid selected controls")
		}
		selected[control.Name()] = control.Body()
	}
	if len(selected) != 2 || !bytes.Equal(selected["package.json"], s.controls[0].Body()) {
		return domain.DependencyResolution{}, errors.New("npm update changed original project constraints")
	}
	graph, err := artifactnpm.ParseProjectControls(reference, selected["package.json"], selected["package-lock.json"])
	lockSum := sha256.Sum256(selected["package-lock.json"])
	if err != nil || !artifactnpm.SameSelectionGraph(graph, resolution.Graph()) || hex.EncodeToString(lockSum[:]) != resolution.LockfileDigest().String() {
		return domain.DependencyResolution{}, errors.New("npm update selected controls do not bind the exact graph")
	}
	if err := s.verify(ctx); err != nil {
		return domain.DependencyResolution{}, err
	}
	s.resolution, s.resolved = resolution, true
	return resolution, nil
}

func (s *npmUpdateSession) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (promoted domain.PromotedInstall, resultErr error) {
	if !s.resolved || s.promoted || install != s.install || !bundle.Valid() || staged.ManifestID() != bundle.ManifestID() || staged.Handle() != "staging:"+bundle.ManifestID().String() || !artifactnpm.SameSelectionGraph(s.resolution.Graph(), bundle.Set().Inspected().Graph()) {
		return promoted, errors.New("npm update lacks the exact inspected staged selection")
	}
	if err := s.verify(ctx); err != nil {
		return promoted, err
	}
	if err := verifyDocuments(bundle); err != nil {
		return promoted, err
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
		return promoted, errors.New("npm update Manifest belongs to another selection")
	}
	expected := map[string][]byte{}
	for _, control := range s.resolution.ProjectControls() {
		expected[control.Name()] = control.Body()
	}
	for _, control := range metadata.Controls {
		if !bytes.Equal(expected[control.Name], control.Content) || expected[control.Name] == nil {
			return promoted, errors.New("npm update Manifest control differs from selected bytes")
		}
		delete(expected, control.Name)
	}
	if len(expected) != 0 {
		return promoted, errors.New("npm update Manifest controls are incomplete")
	}
	p := s.owner.promoter
	stagedRoot := filepath.Join(p.stagingRoot, bundle.ManifestID().String())
	if filepath.Dir(stagedRoot) != p.stagingRoot || rejectSymlinkPath(stagedRoot) != nil {
		return promoted, errors.New("npm update staging boundary is untrusted")
	}
	if err := verifyStagedRecords(stagedRoot, bundle); err != nil {
		return promoted, err
	}
	workspace, err := s.plan.privateWorkspace()
	if err != nil {
		return promoted, err
	}
	defer func() { resultErr = errors.Join(resultErr, os.RemoveAll(workspace)) }()
	selected := filepath.Join(workspace, "selected")
	if err := os.Mkdir(selected, 0o700); err != nil {
		return promoted, err
	}
	manifest, lock, err := preparePromotionProject(selected, stagedRoot, bundle)
	if err != nil {
		return promoted, err
	}
	if err := p.runner.Run(ctx, selected, promotionArguments(selected)); err != nil {
		return promoted, err
	}
	if err := validatePromotionOutput(selected, bundle, manifest, lock); err != nil {
		return promoted, err
	}
	if err := applyNPMSelectionControls(selected, bundle); err != nil {
		return promoted, err
	}
	if err := s.verify(ctx); err != nil {
		return promoted, err
	}
	transaction, err := beginNPMProjectTransaction(s.plan, selected)
	if err != nil {
		return promoted, err
	}
	if err := transaction.commit(); err != nil {
		return promoted, err
	}
	s.promoted = true
	return domain.NewPromotedInstall(bundle.ManifestID(), install.Target())
}

func (s *npmUpdateSession) Close() error {
	if !s.closed {
		s.closed = true
		s.closeErr = s.guard.release()
	}
	return s.closeErr
}

// Adoption replaces only the supported locked package namespace. Unlisted
// top-level files/packages and foreign HAA metadata never become owned by intent
// alone. Package contents remain untrusted and are never executed here.
func validateNPMOriginalPayload(project string, graph domain.LockedDependencyGraph) error {
	root := filepath.Join(project, "node_modules")
	if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	expected := map[string]domain.LockedDependency{}
	for _, node := range graph.Nodes() {
		expected[node.RecordPath()] = node
	}
	actual, err := installedPackagePaths(root, project)
	if err != nil {
		return err
	}
	for path := range actual {
		node, known := expected[path]
		if !known {
			return errors.New("npm original state contains an unlisted package")
		}
		body, err := readBoundedPromotionFile(filepath.Join(project, path, "package.json"), 2<<20)
		var identity struct{ Name, Version string }
		if err != nil || json.Unmarshal(body, &identity) != nil || identity.Name != node.Artifact().Identity().Name() || identity.Version != node.Artifact().Identity().Version() {
			return errors.New("npm original package identity differs from the lock")
		}
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(project, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == "node_modules" {
			return nil
		}
		if entry.IsDir() {
			for packagePath := range actual {
				if relative == packagePath || strings.HasPrefix(relative, packagePath+"/") || strings.HasPrefix(packagePath, relative+"/") {
					return nil
				}
			}
			return errors.New("npm original state contains a foreign directory")
		}
		if relative == "node_modules/.package-lock.json" {
			return nil
		}
		for packagePath := range actual {
			if strings.HasPrefix(relative, packagePath+"/") {
				return nil
			}
		}
		return errors.New("npm original state contains a foreign file outside locked packages")
	})
}

func validateNPMOriginalMetadata(member *npmTransactionMember) error {
	if member == nil {
		return nil
	}
	for _, name := range []string{"manifest.json", "sbom.cdx.json", "npm-transaction.json"} {
		if _, ok := member.files[name]; !ok {
			return errors.New("npm managed metadata namespace is incomplete")
		}
	}
	for name := range member.directories {
		if name != "." && name != "artifacts" {
			return errors.New("npm metadata contains a foreign directory")
		}
	}
	for name, file := range member.files {
		if name == "manifest.json" || name == "sbom.cdx.json" || name == "npm-transaction.json" {
			continue
		}
		base := strings.TrimPrefix(filepath.ToSlash(name), "artifacts/")
		digest := strings.TrimSuffix(base, ".tgz")
		if base == filepath.ToSlash(name) || digest == base || len(digest) != 64 || file.Digest != digest {
			return errors.New("npm metadata contains a foreign or changed artifact file")
		}
	}
	return nil
}
