package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

// This is an explicit caller pin, never an inferred response to artifact
// output. Only one non-root leaf input is supported by this first contract.
type inspectionPrerequisiteRequest struct {
	TargetSHA256 string `json:"target_sha256"`
	Source       string `json:"source"`
	Project      string `json:"project"`
	Version      string `json:"version"`
	Filename     string `json:"filename"`
	SHA256       string `json:"sha256"`
	Reason       string `json:"reason"`
}

func parseInspectionPrerequisiteRequest(raw string) (inspectionPrerequisiteRequest, error) {
	var request inspectionPrerequisiteRequest
	if len(raw) == 0 || len(raw) > 4096 {
		return request, errors.New("inspection prerequisite configuration exceeds bounds")
	}
	// JSON's usual last-key-wins behavior is unsuitable for exact caller pins.
	keys := json.NewDecoder(strings.NewReader(raw))
	opening, err := keys.Token()
	if err != nil || opening != json.Delim('{') {
		return request, errors.New("inspection prerequisite must be one object")
	}
	seen := make(map[string]bool)
	for keys.More() {
		key, err := keys.Token()
		name, ok := key.(string)
		if err != nil || !ok || seen[name] {
			return request, errors.New("inspection prerequisite has ambiguous keys")
		}
		seen[name] = true
		var value string
		if keys.Decode(&value) != nil {
			return request, errors.New("inspection prerequisite values must be strings")
		}
	}
	if _, err := keys.Token(); err != nil {
		return request, errors.New("inspection prerequisite object is incomplete")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || decoder.Decode(new(any)) != io.EOF {
		return request, errors.New("inspection prerequisite configuration is invalid")
	}
	if request.Source != "pypi" || request.Reason != "BROADER_MODULE_PROBES" || request.Project == "" || request.Version == "" || request.Filename == "" {
		return request, errors.New("inspection prerequisite source or scope is unsupported")
	}
	if _, err := domain.NewSHA256Digest(request.SHA256); err != nil {
		return request, err
	}
	if _, err := domain.NewSHA256Digest(request.TargetSHA256); err != nil {
		return request, err
	}
	if path.Base(request.Filename) != request.Filename || !strings.HasSuffix(request.Filename, ".whl") {
		return request, errors.New("inspection prerequisite filename is invalid")
	}
	return request, nil
}

func validateInspectionPrerequisiteTarget(request inspectionPrerequisiteRequest, graph domain.LockedDependencyGraph) error {
	found := false
	for _, node := range graph.Nodes() {
		if node.Artifact().Identity().Name() == request.Project {
			return errors.New("inspection prerequisite cannot replace a selected dependency")
		}
		if node.Artifact().DeclaredIntegrity() == "sha256:"+request.TargetSHA256 {
			if node.Node() == graph.Primary() {
				return errors.New("requested artifact must be inspected in its original environment")
			}
			found = true
		}
	}
	if !found {
		return errors.New("inspection prerequisite target is not selected")
	}
	return nil
}

// Reject stale caller pins after metadata resolution, before the main intake.
// The loader still validates the same binding before acquiring auxiliary bytes.
type prerequisiteCheckingResolver struct {
	resolver ports.DependencyResolver
	read     func() (string, error)
}

func (r prerequisiteCheckingResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	return r.resolve(ctx, reference, install, nil, false)
}

func (r prerequisiteCheckingResolver) ResolveDependencyUpdate(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext, controls []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	return r.resolve(ctx, reference, install, controls, true)
}

func (r prerequisiteCheckingResolver) resolve(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext, controls []domain.ProjectControlFile, update bool) (domain.DependencyResolution, error) {
	raw, err := r.read()
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	var request inspectionPrerequisiteRequest
	if raw != "" {
		request, err = parseInspectionPrerequisiteRequest(raw)
		if err != nil {
			return domain.DependencyResolution{}, err
		}
	}
	var resolution domain.DependencyResolution
	if update {
		resolver, ok := r.resolver.(ports.DependencyUpdateResolver)
		if !ok {
			return domain.DependencyResolution{}, errors.New("python resolver does not support guarded upgrades")
		}
		resolution, err = resolver.ResolveDependencyUpdate(ctx, reference, install, controls)
	} else {
		resolution, err = r.resolver.ResolveDependencies(ctx, reference, install)
	}
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	if raw != "" {
		if err := validateInspectionPrerequisiteTarget(request, resolution.Graph()); err != nil {
			return domain.DependencyResolution{}, err
		}
	}
	return resolution, nil
}

func inspectionPrerequisiteLoader(read func() (string, error), resolver ports.DependencyResolver, intake ports.Artifact) inspectionpypi.InspectionPrerequisiteLoader {
	return func(ctx context.Context, graph domain.LockedDependencyGraph) ([]inspectionpypi.InspectionPrerequisite, error) {
		raw, err := read()
		if err != nil || raw == "" {
			return nil, err
		}
		request, err := parseInspectionPrerequisiteRequest(raw)
		if err != nil {
			return nil, err
		}
		if err := validateInspectionPrerequisiteTarget(request, graph); err != nil {
			return nil, err
		}
		// Existing isolated official-PyPI selection cross-checks the caller pin;
		// neither wheel output nor a private cache determines the expected hash.
		reference, err := artifactpypi.ParseReferenceForSource(request.Project+"@"+request.Version, artifactpypi.PublicPyPIProfile().Source())
		if err != nil {
			return nil, err
		}
		target, _ := domain.NewInstallTarget("/tmp/heliopause-inspection-input")
		install, _ := domain.NewInstallContext(target)
		resolution, err := resolver.ResolveDependencies(ctx, reference, install)
		if err != nil {
			return nil, err
		}
		nodes := resolution.Graph().Nodes()
		if len(nodes) != 1 || len(resolution.Graph().Edges()) != 0 {
			return nil, errors.New("inspection prerequisite transitive expansion is unsupported")
		}
		resolved := nodes[0].Artifact()
		locator, err := url.Parse(resolved.AcquisitionLocator())
		id := resolved.Identity()
		if err != nil || id.Source() != reference.Source() || id.Name() != request.Project || id.Version() != request.Version || id.Variant() != "wheel" || resolved.DeclaredIntegrity() != "sha256:"+request.SHA256 || path.Base(locator.Path) != request.Filename {
			return nil, errors.New("official resolution does not match inspection prerequisite pin")
		}
		run, err := domain.NewRunID()
		if err != nil {
			return nil, err
		}
		acquired, err := intake.Acquire(ctx, run, resolved)
		if err != nil {
			return nil, err
		}
		verified, err := (verificationpypi.IntegrityVerifier{}).Verify(ctx, acquired)
		if err != nil || verified.Outcome() != domain.VerificationVerified || acquired.Digest().String() != request.SHA256 {
			return nil, errors.New("inspection prerequisite integrity is nonqualifying")
		}
		return []inspectionpypi.InspectionPrerequisite{{TargetDigest: request.TargetSHA256, Artifact: acquired, Reason: request.Reason}}, nil
	}
}
