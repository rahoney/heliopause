package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	artifactgithub "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const githubOwnershipMarker = ".heliopause-github.json"
const githubAssetLimit int64 = 512 << 20 // Existing M6 acquisition bound.
const githubApprovalLimit int64 = 1 << 20

type githubApprovalCheck struct {
	ID         string                 `json:"id"`
	Kind       domain.CheckKind       `json:"kind"`
	Required   bool                   `json:"required"`
	Capability domain.Capability      `json:"capability"`
	Status     domain.ExecutionStatus `json:"status"`
}

// The target copy is only a binding. Independent controller state and actual
// recorded Evidence must agree before it grants replacement authority.
type githubAssetApproval struct {
	Schema             int                   `json:"schema"`
	Target             string                `json:"target"`
	Reference          string                `json:"reference"`
	Locator            string                `json:"locator"`
	Name               string                `json:"name"`
	Version            string                `json:"version"`
	Asset              string                `json:"asset"`
	SHA256             string                `json:"sha256"`
	Size               uint64                `json:"size"`
	Mode               uint32                `json:"mode"`
	Manifest           string                `json:"manifest"`
	Run                string                `json:"run"`
	Checks             []githubApprovalCheck `json:"checks"`
	Evidence           []goCacheEvidence     `json:"evidence"`
	EntryPolicy        string                `json:"entry_policy"`
	EntryPolicyVersion uint64                `json:"entry_policy_version"`
	SetPolicy          string                `json:"set_policy"`
	SetPolicyVersion   uint64                `json:"set_policy_version"`
}

func (p *GitHubReleasePromotion) approval(ctx context.Context, bundle domain.VerifiedBundle, install domain.InstallContext) (githubAssetApproval, []byte, error) {
	var empty githubAssetApproval
	if !bundle.Valid() || verifyDocuments(bundle) != nil || len(bundle.Set().Inspected().Inspections()) != 1 || len(bundle.Set().Inspected().Graph().Nodes()) != 1 || len(bundle.Set().Inspected().Graph().Edges()) != 0 || len(bundle.Set().Inspected().Graph().Derivations()) != 0 {
		return empty, nil, errors.New("GitHub ownership requires one exact verified standalone asset")
	}
	inspection := bundle.Set().Inspected().Inspections()[0]
	artifact := inspection.Artifact()
	node := bundle.Set().Inspected().Graph().Nodes()[0]
	selector, size, err := artifactgithub.SelectionForResolved(node.Artifact())
	if err != nil || node.Node() != inspection.Node() || node.Artifact().Identity() != artifact.Identity() || node.RecordPath() != artifact.Identity().Variant() || selector.Asset() == githubOwnershipMarker || size != artifact.SizeBytes() || node.Artifact().DeclaredIntegrity() != "sha256:"+artifact.Digest().String() {
		return empty, nil, errors.New("GitHub ownership subject differs from frozen resolution")
	}
	entry, set := inspection.PolicyDecision(), bundle.Set().Decision()
	if entry.Decision() != domain.DecisionAllow || set.Decision() != domain.DecisionAllow {
		return empty, nil, errors.New("GitHub ownership requires complete ALLOW")
	}
	d := githubAssetApproval{Schema: 1, Target: install.Target().String(), Reference: selector.Locator(), Locator: node.Artifact().AcquisitionLocator(), Name: artifact.Identity().Name(), Version: artifact.Identity().Version(), Asset: artifact.Identity().Variant(), SHA256: artifact.Digest().String(), Size: size, Mode: 0o400, Manifest: bundle.ManifestID().String(), Run: inspection.RunID().String(), EntryPolicy: entry.PolicyID(), EntryPolicyVersion: entry.Version(), SetPolicy: set.PolicyID(), SetPolicyVersion: set.Version()}
	for _, c := range inspection.Checks() {
		if limitation, _ := c.LimitationCode(); limitation != "" {
			return empty, nil, errors.New("GitHub ownership check is incomplete")
		}
		d.Checks = append(d.Checks, githubApprovalCheck{c.ID().String(), c.Kind(), c.Required(), c.Capability(), c.Status()})
		if c.ID().String() == "github-release-elf-dynamic" && c.Status() == domain.ExecutionCompleted {
			d.Mode = 0o500
		}
	}
	for _, ref := range inspection.Evidence() {
		_, hash, e := p.evidence.ReadReference(ctx, inspection.RunID(), ref, artifact.Identity(), artifact.Digest())
		if e != nil {
			return empty, nil, e
		}
		d.Evidence = append(d.Evidence, goCacheEvidence{ref.ID().String(), hash.String()})
	}
	if err := p.verifyApproval(ctx, d); err != nil {
		return empty, nil, err
	}
	if err := verifyGitHubManifestContext(bundle, install, d, node); err != nil {
		return empty, nil, err
	}
	body, err := json.Marshal(d)
	if err != nil || len(body) > int(githubApprovalLimit) {
		return empty, nil, errors.New("GitHub approval exceeds bounds")
	}
	return d, body, nil
}

func (p *GitHubReleasePromotion) verifyApproval(ctx context.Context, d githubAssetApproval) error {
	if p == nil || p.evidence == nil || ctx == nil {
		return errors.New("GitHub approval reader unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if d.Schema != 1 || d.Target == "" || d.Asset == githubOwnershipMarker || d.Size == 0 || d.Size > uint64(githubAssetLimit) || (d.Mode != 0o400 && d.Mode != 0o500) || d.EntryPolicy != "m6-github-release" || d.EntryPolicyVersion != 1 || d.SetPolicy != "m6-github-release" || d.SetPolicyVersion != 1 || len(d.Checks) < 2 || len(d.Checks) > 16 || len(d.Evidence) < 2 || len(d.Evidence) > 32 {
		return errors.New("GitHub independent approval coverage is invalid")
	}
	if _, err := domain.NewInstallTarget(d.Target); err != nil {
		return err
	}
	if _, err := domain.NewSHA256Digest(d.Manifest); err != nil {
		return err
	}
	source, _ := domain.NewSourceID(artifactgithub.SourceName)
	identity, err := domain.NewResolvedArtifactIdentity(source, d.Name, d.Version, d.Asset)
	if err != nil {
		return err
	}
	digest, err := domain.NewSHA256Digest(d.SHA256)
	if err != nil {
		return err
	}
	resolved, err := domain.NewResolvedArtifact(identity, d.Locator, "sha256:"+d.SHA256)
	if err != nil {
		return err
	}
	selector, size, err := artifactgithub.SelectionForResolved(resolved)
	if err != nil || selector.Locator() != d.Reference || size != d.Size {
		return errors.New("GitHub old selector differs from independent approval")
	}
	run, err := domain.ParseRunID(d.Run)
	if err != nil {
		return err
	}
	checks := map[string]bool{}
	required := map[string]bool{}
	for _, c := range d.Checks {
		id, err := domain.NewCheckID(c.ID)
		if err != nil {
			return err
		}
		if _, err := domain.NewCheckExecution(id, c.Kind, c.Required, c.Capability, c.Status, ""); err != nil {
			return err
		}
		if checks[c.ID] || c.Status != domain.ExecutionCompleted || c.Capability != domain.CapabilitySupported {
			return errors.New("GitHub old checks are duplicated or incomplete")
		}
		checks[c.ID] = true
		required[c.ID] = c.Required
	}
	proof := map[string]domain.Evidence{}
	for _, r := range d.Evidence {
		id, err := domain.NewEvidenceID(r.ID)
		if err != nil {
			return err
		}
		if _, exists := proof[r.ID]; exists {
			return errors.New("GitHub old Evidence is duplicated")
		}
		ref, err := domain.NewEvidenceReference(id, "evidence:"+run.String()+":"+id.String())
		if err != nil {
			return err
		}
		e, hash, err := p.evidence.ReadReference(ctx, run, ref, identity, digest)
		if err != nil || hash.String() != r.SHA256 || !checks[e.CheckID().String()] {
			return errors.New("GitHub old Evidence is missing, changed or unbound")
		}
		proof[r.ID] = e
	}
	for _, id := range []string{"github-release-declared-sha256", "github-release-static"} {
		if !required[id] || proof[id+"-result"].CheckID().String() != id {
			return errors.New("GitHub approval omits required source/static proof")
		}
	}
	kind := proof["github-release-static-result"].Kind()
	switch kind {
	case "github-release-elf-static":
		if d.Mode != 0o500 || !required["github-release-elf-dynamic"] || proof["github-release-elf-dynamic-result"].CheckID().String() != "github-release-elf-dynamic" {
			return errors.New("GitHub ELF approval omits dynamic proof")
		}
	case "github-release-zip-static":
		if d.Mode != 0o400 || !strings.HasSuffix(d.Asset, ".zip") {
			return errors.New("GitHub ZIP approval differs from asset")
		}
	case "github-release-targz-static":
		if d.Mode != 0o400 || !strings.HasSuffix(d.Asset, ".tar.gz") {
			return errors.New("GitHub archive approval differs from asset")
		}
	default:
		return errors.New("GitHub approval has unsupported static proof")
	}
	return nil
}

func decodeGitHubApproval(body []byte) (githubAssetApproval, error) {
	var d githubAssetApproval
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if len(body) == 0 || len(body) > int(githubApprovalLimit) || decoder.Decode(&d) != nil {
		return d, errors.New("GitHub approval is invalid")
	}
	canonical, err := json.Marshal(d)
	if err != nil || !bytes.Equal(canonical, body) {
		return d, errors.New("GitHub approval is not canonical")
	}
	return d, nil
}

func verifyGitHubManifestContext(bundle domain.VerifiedBundle, install domain.InstallContext, d githubAssetApproval, node domain.LockedDependency) error {
	var m struct {
		Target struct {
			Path        string `json:"path"`
			RequiresNew bool   `json:"requires_new"`
		} `json:"target"`
		Primary string `json:"primary"`
		Policy  struct {
			Decision string `json:"decision"`
			ID       string `json:"id"`
			Version  uint64 `json:"version"`
		} `json:"policy"`
		Entries []struct {
			Node       string   `json:"node"`
			RecordPath string   `json:"record_path"`
			Source     string   `json:"source"`
			Name       string   `json:"name"`
			Version    string   `json:"version"`
			Variant    string   `json:"variant"`
			SHA256     string   `json:"sha256"`
			Declared   string   `json:"declared_integrity"`
			Run        string   `json:"run_id"`
			Evidence   []string `json:"evidence"`
		} `json:"entries"`
	}
	if json.Unmarshal(bundle.ManifestDocument(), &m) != nil || m.Target.Path != install.Target().String() || m.Target.RequiresNew != install.RequiresNewTarget() || m.Primary != node.Node().String() || len(m.Entries) != 1 || m.Policy.Decision != "ALLOW" || m.Policy.ID != d.SetPolicy || m.Policy.Version != d.SetPolicyVersion {
		return errors.New("GitHub Manifest differs from original target/context")
	}
	e := m.Entries[0]
	refs := []string{}
	for _, r := range d.Evidence {
		refs = append(refs, "evidence:"+d.Run+":"+r.ID)
	}
	slices.Sort(refs)
	if e.Node != node.Node().String() || e.RecordPath != d.Asset || e.Source != "github-release" || e.Name != d.Name || e.Version != d.Version || e.Variant != d.Asset || e.SHA256 != d.SHA256 || e.Declared != "sha256:"+d.SHA256 || e.Run != d.Run || !slices.Equal(refs, e.Evidence) {
		return errors.New("GitHub Manifest entry differs from approved bytes/Evidence")
	}
	return nil
}
