package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// UVSyncPreparation belongs to one request. It retains the original target and
// source through a single complete selection; selection grants no approval.
type UVSyncPreparation struct {
	promoter          *PyPIPromotion
	resolver          ports.ProjectLockUpdateResolver
	source            string
	format            artifactpypi.UVSyncSourceFormat
	allowEmpty, adopt bool
	trustedRoots      []string
	native            *pypiNativeEnrollment
	started           bool
	active            *uvSyncSession
}

func NewUVSyncPreparation(promoter *PyPIPromotion, resolver ports.ProjectLockUpdateResolver, source string, format artifactpypi.UVSyncSourceFormat, allowEmpty, adopt bool, trustedRoots []string) (*UVSyncPreparation, error) {
	if promoter == nil || resolver == nil || len(trustedRoots) == 0 || len(trustedRoots) > 32 || (format != artifactpypi.UVSyncRequirements && format != artifactpypi.UVSyncPylock) || format == artifactpypi.UVSyncPylock && allowEmpty {
		return nil, errors.New("UV sync requires its source intent, trusted storage and native resolver")
	}
	staging := false
	for _, root := range trustedRoots {
		staging = staging || root == promoter.stagingRoot
	}
	if !staging {
		return nil, errors.New("UV sync must protect its actual staging root")
	}
	return &UVSyncPreparation{promoter: promoter, resolver: resolver, source: source, format: format, allowEmpty: allowEmpty, adopt: adopt, trustedRoots: append([]string(nil), trustedRoots...)}, nil
}

func (p *UVSyncPreparation) WithNativeEnrollment(resolver ports.DependencyResolver, intake ports.Artifact, intakeRoot string) (*UVSyncPreparation, error) {
	if p == nil || p.started || resolver == nil || intake == nil {
		return nil, errors.New("UV sync original enrollment is unavailable")
	}
	found := false
	for _, root := range p.trustedRoots {
		found = found || root == intakeRoot
	}
	if !found {
		return nil, errors.New("UV sync original intake is outside its frozen storage boundaries")
	}
	copy := *p
	copy.native = &pypiNativeEnrollment{resolver: resolver, intake: intake, intakeRoot: intakeRoot, runner: p.promoter.runner, client: nativeUVClient}
	return &copy, nil
}

type uvSyncSession struct {
	owner                        *UVSyncPreparation
	install                      domain.InstallContext
	transaction                  *pypiVenvTransaction
	original                     pypiInstalledSnapshot
	input                        artifactpypi.UVSyncInput
	baseControls, controls       []domain.ProjectControlFile
	proof                        domain.ProjectControlFile
	selected                     domain.ProjectDependencyUpdate
	attempted, committed, closed bool
	closeErr                     error
}

func (p *UVSyncPreparation) Begin(ctx context.Context, install domain.InstallContext) (result *uvSyncSession, resultErr error) {
	if p == nil || p.started || ctx == nil || !install.Valid() || install.Mode() != domain.InstallPythonVenv || p.promoter.goos != "linux" || p.promoter.goarch != "amd64" || strings.Contains(install.Target().String(), ",") {
		return nil, errors.New("UV sync requires one supported Linux amd64 venv transaction")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.started = true
	plan, err := discoverUVPythonVenv(install.Target().String())
	if err != nil {
		return nil, err
	}
	tx, err := beginPyPIVenvTransactionWithDiscovery(plan, discoverUVPythonVenv)
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
	}
	s := &uvSyncSession{owner: p, install: install, transaction: tx}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, s.Close())
			result = nil
		}
	}()
	s.input, err = freezeUVSyncSource(ctx, tx, p.source, p.format, p.allowEmpty, p.trustedRoots)
	if err != nil {
		return nil, err
	}
	s.original, err = capturePyPIInstalledSnapshot(tx)
	if err != nil {
		return nil, err
	}
	metadata := s.original.metadataFiles()
	managed, err := managedPyPIInventoryFromMetadata(tx, metadata)
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
	}
	if len(metadata) > len(managed) {
		if !p.adopt {
			return nil, ports.NewInstallUpdateFailure("UPDATE_ADOPTION_REQUIRED", errors.New("complete sync requires ownership of every original distribution"))
		}
		if p.native == nil {
			return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", errors.New("UV sync original enrollment collaborators are unavailable"))
		}
		inventory, err := capturePyPINativeInventoryFromMetadata(ctx, tx, metadata)
		if err != nil {
			return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
		}
		selection, err := p.native.selectAllOwnership(ctx, install, inventory)
		if err != nil {
			return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
		}
		if len(selection.destinations) == 0 || !selection.control.Present() {
			return nil, errors.New("complete original ownership proof is missing")
		}
		for _, directory := range selection.absentDirectories {
			if err := tx.freezeParents(directory); err != nil {
				return nil, err
			}
			if _, err := os.Lstat(directory); !errors.Is(err, os.ErrNotExist) {
				return nil, errors.New("UV sync original cache namespace appeared")
			}
		}
		if err := s.VerifyUnchanged(ctx); err != nil {
			return nil, err
		}
		for _, destination := range selection.destinations {
			if _, exists := tx.current.Files[destination.key()]; exists {
				return nil, errors.New("UV sync original proof overlaps another owner")
			}
			tx.current.Files[destination.key()] = destination
		}
		s.proof = selection.control
	}
	s.baseControls, err = sandbox.NewUVSyncOriginalControls(s.input)
	if err != nil {
		return nil, err
	}
	s.controls = append([]domain.ProjectControlFile(nil), s.baseControls...)
	if s.proof.Present() {
		s.controls = append(s.controls, s.proof)
	}
	if err := s.VerifyUnchanged(ctx); err != nil {
		return nil, err
	}
	p.active = s
	return s, nil
}

func (s *uvSyncSession) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), s.controls...)
}
func (s *uvSyncSession) VerifyUnchanged(ctx context.Context) error {
	if s == nil || s.closed || s.committed {
		return errors.New("UV sync original guard is unavailable")
	}
	return s.original.verify(ctx, s.transaction)
}
func (s *uvSyncSession) Close() error {
	if s != nil && !s.closed {
		s.closed = true
		s.closeErr = s.transaction.close()
	}
	if s == nil {
		return nil
	}
	return s.closeErr
}

func (p *UVSyncPreparation) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if p == nil || p.active == nil {
		return domain.ProjectDependencyUpdate{}, errors.New("UV sync selection requires its held original guard")
	}
	s := p.active
	if s.attempted || selector != "" || install != s.install || !sameUVSyncControls(original, s.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("UV sync selection changed or was repeated")
	}
	if err := s.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	s.attempted = true
	base, err := p.resolver.ResolveProjectLockUpdate(ctx, install, append([]domain.ProjectControlFile(nil), s.baseControls...), "")
	if err = errors.Join(err, s.VerifyUnchanged(ctx)); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	if err := validateUVSyncSelection(base, install, s.input, s.baseControls); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	selected := base
	if s.proof.Present() {
		controls := append(base.SelectedControls(), s.proof)
		snapshot, err := bindUVSyncProof(base.Snapshot(), controls)
		if err != nil {
			return domain.ProjectDependencyUpdate{}, err
		}
		selected, err = domain.NewProjectDependencyUpdateFromSnapshot(s.controls, controls, snapshot)
		if err != nil {
			return domain.ProjectDependencyUpdate{}, err
		}
	}
	s.selected = selected
	return selected, nil
}

// Staging consumes the one actual selection, never reconstructed source input.
func (p *UVSyncPreparation) SelectedUpdate(ctx context.Context) (domain.ProjectDependencyUpdate, error) {
	if p == nil || p.active == nil || !p.active.selected.Valid() {
		return domain.ProjectDependencyUpdate{}, errors.New("UV sync has no frozen complete selection")
	}
	if err := p.active.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	return p.active.selected, nil
}

func sameUVSyncControls(left, right []domain.ProjectControlFile) bool {
	if len(left) != len(right) {
		return false
	}
	a, b := append([]domain.ProjectControlFile(nil), left...), append([]domain.ProjectControlFile(nil), right...)
	sort.Slice(a, func(i, j int) bool { return a[i].Name() < a[j].Name() })
	sort.Slice(b, func(i, j int) bool { return b[i].Name() < b[j].Name() })
	for i := range a {
		if a[i].Name() != b[i].Name() || a[i].Present() != b[i].Present() || a[i].Digest() != b[i].Digest() {
			return false
		}
	}
	return true
}

func validateUVSyncSelection(update domain.ProjectDependencyUpdate, install domain.InstallContext, input artifactpypi.UVSyncInput, original []domain.ProjectControlFile) error {
	if !update.Valid() || update.HasRequestedResolution() || update.Snapshot().Context() != install || update.Snapshot().Source() != artifactpypi.PublicPyPIProfile().Source() || !sameUVSyncControls(update.OriginalControls(), original) {
		return errors.New("UV sync resolver changed complete original context")
	}
	selected := update.SelectedControls()
	if len(selected) != len(original) {
		return errors.New("UV sync resolver changed its control protocol")
	}
	var lock artifactpypi.UVProjectLock
	for _, control := range selected {
		if control.Name() == artifactpypi.UVSelectionLockControlName {
			var err error
			lock, err = artifactpypi.ParseUVProjectLock(control.Body(), input.Python())
			if err != nil {
				return err
			}
		} else {
			found := false
			for _, before := range original {
				found = found || control.Name() == before.Name() && control.Present() == before.Present() && control.Digest() == before.Digest()
			}
			if !found {
				return errors.New("UV sync resolver changed immutable source, intent or runtime")
			}
		}
	}
	if err := input.ValidateSelection(lock); err != nil {
		return err
	}
	_, err := lock.SelectResolvedWheels(update.Snapshot().Dependencies())
	return err
}

func bindUVSyncProof(base domain.ProjectDependencySnapshot, controls []domain.ProjectControlFile) (domain.ProjectDependencySnapshot, error) {
	digests := make([]domain.ProjectControlDigest, 0, len(controls))
	for _, control := range controls {
		digest, err := domain.NewProjectControlDigest(control.Name(), control.Digest())
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		digests = append(digests, digest)
	}
	sort.Slice(digests, func(i, j int) bool { return digests[i].Name() < digests[j].Name() })
	type binding struct{ Name, Digest string }
	parts := make([]binding, 0, len(digests))
	for _, digest := range digests {
		parts = append(parts, binding{digest.Name(), digest.Digest().String()})
	}
	body, err := json.Marshal(struct {
		Schema, Base string
		Controls     []binding
	}{"uv-sync-original-proof/v1", base.GraphDigest().String(), parts})
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	sum := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	if base.DependencyFree() {
		return domain.NewDependencyFreeProjectSnapshot(base.Context(), base.Source(), digests, digest)
	}
	return domain.NewProjectDependencySnapshot(base.Context(), base.Source(), digests, base.Dependencies(), digest)
}

func sameUVSyncUpdate(left, right domain.ProjectDependencyUpdate) bool {
	return left.Valid() && right.Valid() && reflect.DeepEqual(left, right)
}
