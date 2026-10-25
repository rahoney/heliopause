package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type pnpmCacheEntry struct {
	Source, Package, Version, Digest, Integrity, Run string
	Evidence                                         []goCacheEvidence
	Policy                                           string
	PolicyVersion                                    uint64
}
type pnpmCacheDocument struct {
	Schema                          int
	Graph, Project, Runtime, Layout string
	Operation, Manifest             string
	Policy                          string
	PolicyVersion                   uint64
	Controls                        []cacheFileRecord
	Entries                         []pnpmCacheEntry
	Files                           []cacheFileRecord
	Records                         []cacheFileRecord
	DependencyFree                  bool
}
type pnpmInstanceBinding struct{ Instance, Session, Layout, Runtime string }

func (c *PNPMVerifiedCache) approval(ctx context.Context, set domain.ProjectVerifiedSet) (pnpmCacheDocument, artifactnpm.PNPMInspectionInput, error) {
	invalid := errors.New("pnpm staging approval or recorded instance coverage is invalid")
	if c == nil || ctx == nil || !set.Valid() || !c.frozen.MatchesSnapshot(set.Inspected().Snapshot()) {
		return pnpmCacheDocument{}, artifactnpm.PNPMInspectionInput{}, invalid
	}
	if err := ctx.Err(); err != nil {
		return pnpmCacheDocument{}, artifactnpm.PNPMInspectionInput{}, errors.Join(err, context.Cause(ctx))
	}
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for _, inspection := range set.Inspected().Inspections() {
		subject := inspection.Artifact()
		if subject.ContentHandle() != "intake:"+inspection.RunID().String()+":tarball" {
			return pnpmCacheDocument{}, artifactnpm.PNPMInspectionInput{}, invalid
		}
		acquired[inspection.Node()] = subject
	}
	input, err := c.frozen.BindAcquired(set.Inspected().Snapshot(), acquired)
	if err != nil {
		return pnpmCacheDocument{}, input, err
	}
	doc, err := npmProjectCacheApprovalEvidence(ctx, set, c.operation, c.frozen.RuntimeIdentity(), c.frozen.Controls(), c.evidence, func(subject domain.AcquiredArtifact) ([]string, error) {
		instances, err := input.InstancesFor(subject)
		if err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(instances))
		for _, instance := range instances {
			keys = append(keys, instance.Key)
		}
		return keys, nil
	}, npmProjectCachePNPM)
	return doc, input, err
}

func npmProjectCacheApprovalEvidence(ctx context.Context, set domain.ProjectVerifiedSet, operation domain.OperationID, runtimeIdentity string, controls []domain.ProjectControlFile, reader ProjectEvidenceReader, instanceKeys func(domain.AcquiredArtifact) ([]string, error), client npmProjectCacheClient) (pnpmCacheDocument, error) {
	spec, known := npmProjectCacheIdentity(client)
	if !known || reader == nil || instanceKeys == nil {
		return pnpmCacheDocument{}, errors.New("unknown project cache approval client")
	}
	invalid := errors.New(spec.prefix + " staging approval or recorded instance coverage is invalid")
	doc := pnpmCacheDocument{Schema: 2, Operation: operation.String(), Graph: set.Inspected().Snapshot().GraphDigest().String(), Project: pnpmCacheHash([]byte(set.Inspected().Snapshot().Context().Target().String())), Runtime: runtimeIdentity, Policy: set.Decision().PolicyID(), PolicyVersion: set.Decision().Version(), DependencyFree: set.Inspected().Snapshot().DependencyFree()}
	for _, control := range controls {
		controlKind := "file"
		if client == npmProjectCacheBun && !control.Present() {
			controlKind = "absent"
		}
		doc.Controls = append(doc.Controls, cacheFileRecord{Path: control.Name(), Kind: controlKind, Size: int64(len(control.Body())), SHA256: control.Digest().String()})
	}
	sessions := map[string]bool{}
	for _, inspection := range set.Inspected().Inspections() {
		subject := inspection.Artifact()
		action, known := inspection.HostInstallActionEvidence()
		if !known || action.Kind() != domain.EvidenceHostInstallActionNotRequired {
			return pnpmCacheDocument{}, invalid
		}
		records, facts, err := projectCacheEvidenceFacts(ctx, reader, inspection)
		if err != nil {
			return pnpmCacheDocument{}, err
		}
		instances, err := instanceKeys(subject)
		if err != nil {
			return pnpmCacheDocument{}, err
		}
		expected := map[string]bool{}
		for _, instance := range instances {
			expected[pnpmCacheHash([]byte(instance))] = false
		}
		checks := map[string]domain.CheckExecution{}
		for _, check := range inspection.Checks() {
			if check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted {
				return pnpmCacheDocument{}, invalid
			}
			if _, duplicate := checks[check.ID().String()]; duplicate {
				return pnpmCacheDocument{}, invalid
			}
			checks[check.ID().String()] = check
		}
		if !checks["npm-static-archive"].Required() || !checks[spec.prefix+"-dynamic-lifecycle"].Required() {
			return pnpmCacheDocument{}, invalid
		}
		observations := map[string]domain.Evidence{}
		coverage := 0
		for _, fact := range facts {
			if fact.Kind() == spec.prefix+"-instance-observations" {
				if _, duplicate := observations[fact.CheckID().String()]; duplicate {
					return pnpmCacheDocument{}, invalid
				}
				observations[fact.CheckID().String()] = fact
			}
			if fact.Kind() == spec.prefix+"-instance-coverage" {
				coverage++
				if fact.CheckID().String() != spec.prefix+"-dynamic-lifecycle" || fact.Summary() != fmt.Sprintf("expected=%d attempted=%d completed=%d", len(instances), len(instances), len(instances)) {
					return pnpmCacheDocument{}, invalid
				}
			}
		}
		bindings := 0
		for _, fact := range facts {
			if fact.Kind() != spec.prefix+"-instance-binding" {
				continue
			}
			var binding pnpmInstanceBinding
			if json.Unmarshal([]byte(fact.Summary()), &binding) != nil {
				return pnpmCacheDocument{}, invalid
			}
			canonical, err := json.Marshal(binding)
			seen, present := expected[binding.Instance]
			if err != nil || string(canonical) != fact.Summary() || !present || seen || binding.Runtime != pnpmCacheHash([]byte(runtimeIdentity)) || len(binding.Layout) != 64 || sessions[binding.Session] {
				return pnpmCacheDocument{}, invalid
			}
			if _, err := domain.ParseSandboxSessionID(binding.Session); err != nil {
				return pnpmCacheDocument{}, invalid
			}
			if _, err := domain.NewSHA256Digest(binding.Layout); err != nil {
				return pnpmCacheDocument{}, invalid
			}
			if doc.Layout != "" && doc.Layout != binding.Layout {
				return pnpmCacheDocument{}, invalid
			}
			doc.Layout = binding.Layout
			checkID := spec.prefix + "-instance-facts-" + binding.Instance[:24]
			if fact.CheckID().String() != checkID || checks[checkID].ID().String() != checkID || checks[checkID].Required() {
				return pnpmCacheDocument{}, invalid
			}
			var observed struct {
				Session    string               `json:"session"`
				Status     domain.SandboxStatus `json:"status"`
				Limitation string               `json:"limitation"`
			}
			observation, present := observations[checkID]
			if !present || json.Unmarshal([]byte(observation.Summary()), &observed) != nil || observed.Session != binding.Session || observed.Status != domain.SandboxCompleted || observed.Limitation != "" {
				return pnpmCacheDocument{}, invalid
			}
			expected[binding.Instance] = true
			sessions[binding.Session] = true
			bindings++
		}
		if coverage != 1 || bindings != len(expected) || len(observations) != bindings {
			return pnpmCacheDocument{}, invalid
		}
		integrity, _ := subject.DeclaredIntegrity()
		decision := inspection.PolicyDecision()
		doc.Entries = append(doc.Entries, pnpmCacheEntry{subject.Identity().Source().String(), subject.Identity().Name(), subject.Identity().Version(), subject.Digest().String(), integrity, inspection.RunID().String(), records, decision.PolicyID(), decision.Version()})
	}
	if !doc.DependencyFree && doc.Layout == "" {
		return pnpmCacheDocument{}, invalid
	}
	return doc, nil
}
