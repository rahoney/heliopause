package npm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// NPMProjectRootLifecycleScripts validates the local manifest without using its optional
// package name/version as registry identity. Only native install lifecycle keys
// select execution; other script bodies are not executed by this operation.
func NPMProjectRootLifecycleScripts(manifest []byte) ([]string, error) {
	if len(manifest) == 0 || len(manifest) > ProjectControlLimit || strictControlJSON(manifest) != nil {
		return nil, errors.New("npm local lifecycle manifest is invalid")
	}
	var fields struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(manifest, &fields) != nil {
		return nil, errors.New("npm local lifecycle scripts must be strings")
	}
	var phases []string
	for _, key := range []string{"preinstall", "install", "postinstall", "prepublish", "preprepare", "prepare", "postprepare", "dependencies"} {
		if fields.Scripts[key] != "" {
			phases = append(phases, key)
		}
	}
	return phases, nil
}

// NPMProjectRootArtifact exposes an already frozen local subject through the
// existing Inspect workflow. Local byte integrity never claims npm provenance.
type NPMProjectRootArtifact struct {
	source   domain.AcquiredArtifact
	resolved domain.ResolvedArtifact
}

func NewNPMProjectRootArtifact(source domain.AcquiredArtifact) (*NPMProjectRootArtifact, error) {
	integrity, declared := source.DeclaredIntegrity()
	parts := strings.Split(source.ContentHandle(), ":")
	if source.Identity().Source().String() != "project-local" || source.Identity().Variant() != "npm-source" || source.SizeBytes() == 0 || source.SizeBytes() > uint64(NPMProjectInspectionWorkspaceLimit) || !declared || integrity != "sha256:"+source.Digest().String() || len(parts) != 3 || parts[0] != "intake" || parts[2] != "npm-source" {
		return nil, errors.New("npm root inspection requires its frozen local source")
	}
	if _, err := domain.ParseRunID(parts[1]); err != nil {
		return nil, err
	}
	resolved, err := domain.NewResolvedArtifact(source.Identity(), "project-local:"+source.Digest().String(), integrity)
	if err != nil {
		return nil, err
	}
	return &NPMProjectRootArtifact{source, resolved}, nil
}

func (a *NPMProjectRootArtifact) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	if a == nil || ctx == nil || reference.Source() != a.source.Identity().Source() || reference.Locator() != a.source.Identity().Name() {
		return domain.ResolvedArtifact{}, errors.New("npm root reference differs from the frozen local subject")
	}
	if err := ctx.Err(); err != nil {
		return domain.ResolvedArtifact{}, err
	}
	return a.resolved, nil
}

func (a *NPMProjectRootArtifact) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if a == nil || ctx == nil || run.String() == "" || resolved != a.resolved {
		return domain.AcquiredArtifact{}, errors.New("npm root acquisition differs from the frozen local subject")
	}
	if err := ctx.Err(); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	return a.source, nil
}

// The root subject belongs to this exact current project, separately from its
// registry dependencies. This checks data identity, not safety or approval.
func (i NPMProjectInspectionInput) MatchesRootSource(source domain.AcquiredArtifact) bool {
	if !i.Valid() || source.Identity().Source().String() != "project-local" || source.Identity().Variant() != "npm-source" || source.Identity().Version() != "snapshot" {
		return false
	}
	project := sha256.Sum256([]byte(i.context.snapshot.Context().Target().String()))
	return source.Identity().Name() == "project-"+hex.EncodeToString(project[:])
}

// NPMProjectRootInspectionInput keeps the current complete dependency layout
// and the independently frozen local source together, without a fake primary.
type NPMProjectRootInspectionInput struct {
	dependencies NPMProjectInspectionInput
	layout       NPMProjectInspectionLayout
	source       domain.AcquiredArtifact
}

func NewNPMProjectRootInspectionInput(dependencies NPMProjectInspectionInput, layout NPMProjectInspectionLayout, source domain.AcquiredArtifact) (NPMProjectRootInspectionInput, error) {
	input := NPMProjectRootInspectionInput{dependencies, layout, source}
	if !input.Valid() {
		return NPMProjectRootInspectionInput{}, errors.New("npm root inspection lacks the exact local source and complete dependency layout")
	}
	return input, nil
}

func (i NPMProjectRootInspectionInput) Valid() bool {
	if !i.dependencies.MatchesRootSource(i.source) {
		return false
	}
	if i.dependencies.Context().Snapshot().DependencyFree() {
		return i.layout.Digest().String() == "" && len(i.layout.Entries()) == 0
	}
	return i.layout.MatchesInput(i.dependencies)
}
func (i NPMProjectRootInspectionInput) Source() domain.AcquiredArtifact { return i.source }
func (i NPMProjectRootInspectionInput) Dependencies() NPMProjectInspectionInput {
	return i.dependencies
}
func (i NPMProjectRootInspectionInput) DependencyLayout() NPMProjectInspectionLayout { return i.layout }
