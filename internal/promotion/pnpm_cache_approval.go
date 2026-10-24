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
	Policy                          string
	PolicyVersion                   uint64
	Controls                        []cacheFileRecord
	Entries                         []pnpmCacheEntry
	Files                           []cacheFileRecord
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
	doc := pnpmCacheDocument{Schema: 1, Graph: set.Inspected().Snapshot().GraphDigest().String(), Project: pnpmCacheHash([]byte(set.Inspected().Snapshot().Context().Target().String())), Runtime: c.frozen.RuntimeIdentity(), Policy: set.Decision().PolicyID(), PolicyVersion: set.Decision().Version(), DependencyFree: set.Inspected().Snapshot().DependencyFree()}
	for _, control := range c.frozen.Controls() {
		doc.Controls = append(doc.Controls, cacheFileRecord{Path: control.Name(), Kind: "file", Size: int64(len(control.Body())), SHA256: control.Digest().String()})
	}
	sessions := map[string]bool{}
	for _, inspection := range set.Inspected().Inspections() {
		subject := inspection.Artifact()
		action, known := inspection.HostInstallActionEvidence()
		if !known || action.Kind() != domain.EvidenceHostInstallActionNotRequired {
			return pnpmCacheDocument{}, input, invalid
		}
		records, facts, err := projectCacheEvidenceFacts(ctx, c.evidence, inspection)
		if err != nil {
			return pnpmCacheDocument{}, input, err
		}
		instances, err := input.InstancesFor(subject)
		if err != nil {
			return pnpmCacheDocument{}, input, err
		}
		expected := map[string]bool{}
		for _, instance := range instances {
			expected[pnpmCacheHash([]byte(instance.Key))] = false
		}
		checks := map[string]domain.CheckExecution{}
		for _, check := range inspection.Checks() {
			if check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted {
				return pnpmCacheDocument{}, input, invalid
			}
			if _, duplicate := checks[check.ID().String()]; duplicate {
				return pnpmCacheDocument{}, input, invalid
			}
			checks[check.ID().String()] = check
		}
		if !checks["npm-static-archive"].Required() || !checks["pnpm-dynamic-lifecycle"].Required() {
			return pnpmCacheDocument{}, input, invalid
		}
		observations := map[string]domain.Evidence{}
		coverage := 0
		for _, fact := range facts {
			if fact.Kind() == "pnpm-instance-observations" {
				if _, duplicate := observations[fact.CheckID().String()]; duplicate {
					return pnpmCacheDocument{}, input, invalid
				}
				observations[fact.CheckID().String()] = fact
			}
			if fact.Kind() == "pnpm-instance-coverage" {
				coverage++
				if fact.CheckID().String() != "pnpm-dynamic-lifecycle" || fact.Summary() != fmt.Sprintf("expected=%d attempted=%d completed=%d", len(instances), len(instances), len(instances)) {
					return pnpmCacheDocument{}, input, invalid
				}
			}
		}
		bindings := 0
		for _, fact := range facts {
			if fact.Kind() != "pnpm-instance-binding" {
				continue
			}
			var binding pnpmInstanceBinding
			if json.Unmarshal([]byte(fact.Summary()), &binding) != nil {
				return pnpmCacheDocument{}, input, invalid
			}
			canonical, err := json.Marshal(binding)
			seen, present := expected[binding.Instance]
			if err != nil || string(canonical) != fact.Summary() || !present || seen || binding.Runtime != pnpmCacheHash([]byte(c.frozen.RuntimeIdentity())) || len(binding.Layout) != 64 || sessions[binding.Session] {
				return pnpmCacheDocument{}, input, invalid
			}
			if _, err := domain.ParseSandboxSessionID(binding.Session); err != nil {
				return pnpmCacheDocument{}, input, invalid
			}
			if _, err := domain.NewSHA256Digest(binding.Layout); err != nil {
				return pnpmCacheDocument{}, input, invalid
			}
			if doc.Layout != "" && doc.Layout != binding.Layout {
				return pnpmCacheDocument{}, input, invalid
			}
			doc.Layout = binding.Layout
			checkID := "pnpm-instance-facts-" + binding.Instance[:24]
			if fact.CheckID().String() != checkID || checks[checkID].ID().String() != checkID || checks[checkID].Required() {
				return pnpmCacheDocument{}, input, invalid
			}
			var observed struct {
				Session    string               `json:"session"`
				Status     domain.SandboxStatus `json:"status"`
				Limitation string               `json:"limitation"`
			}
			observation, present := observations[checkID]
			if !present || json.Unmarshal([]byte(observation.Summary()), &observed) != nil || observed.Session != binding.Session || observed.Status != domain.SandboxCompleted || observed.Limitation != "" {
				return pnpmCacheDocument{}, input, invalid
			}
			expected[binding.Instance] = true
			sessions[binding.Session] = true
			bindings++
		}
		if coverage != 1 || bindings != len(expected) || len(observations) != bindings {
			return pnpmCacheDocument{}, input, invalid
		}
		integrity, _ := subject.DeclaredIntegrity()
		decision := inspection.PolicyDecision()
		doc.Entries = append(doc.Entries, pnpmCacheEntry{subject.Identity().Source().String(), subject.Identity().Name(), subject.Identity().Version(), subject.Digest().String(), integrity, inspection.RunID().String(), records, decision.PolicyID(), decision.Version()})
	}
	if !doc.DependencyFree && doc.Layout == "" {
		return pnpmCacheDocument{}, input, invalid
	}
	return doc, input, nil
}
