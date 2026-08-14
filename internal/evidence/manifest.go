// Package evidence creates deterministic trusted records from normalized
// Domain values. It does not read Artifact bytes or perform Promotion.
package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	verifiedManifestSchema = "helox.verified-manifest/v1"
	cycloneDXSchema        = "http://cyclonedx.org/schema/bom-1.7.schema.json"
	maximumContextText     = 2048
)

// ManifestContext binds the verified set to its original operation, target,
// resolver runtime, and exact lockfile content.
type ManifestContext struct {
	OperationID     domain.OperationID
	InstallContext  domain.InstallContext
	ResolverRuntime string
	LockfileDigest  domain.ContentDigest
}

// Generator adapts deterministic record generation to the Application-owned
// Manifest Port.
type Generator struct{}

func (Generator) Build(ctx context.Context, operationID domain.OperationID, installContext domain.InstallContext, resolution domain.DependencyResolution, set domain.VerifiedSet) (domain.VerifiedBundle, error) {
	if ctx == nil {
		return domain.VerifiedBundle{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.VerifiedBundle{}, err
	}
	return BuildVerifiedBundle(ManifestContext{OperationID: operationID, InstallContext: installContext, ResolverRuntime: resolution.RuntimeIdentity(), LockfileDigest: resolution.LockfileDigest()}, set)
}

// BuildVerifiedBundle emits compact deterministic UTF-8 JSON for both the HAA
// Manifest and CycloneDX 1.7 SBOM, then seals them into one immutable bundle.
func BuildVerifiedBundle(context ManifestContext, set domain.VerifiedSet) (domain.VerifiedBundle, error) {
	if context.OperationID.String() == "" || context.InstallContext.Target().String() == "" || !context.InstallContext.RequiresNewTarget() || context.LockfileDigest.String() == "" || !validContextText(context.ResolverRuntime) || !set.Valid() {
		return domain.VerifiedBundle{}, errors.New("manifest generation requires verified set and complete context")
	}
	entries, edges, err := manifestEntries(set)
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	decision := set.Decision()
	reasons := decision.Reasons()
	sort.Strings(reasons)
	payload := manifestPayload{
		Schema:      verifiedManifestSchema,
		OperationID: context.OperationID.String(),
		Target:      manifestTarget{Path: context.InstallContext.Target().String(), RequiresNew: true},
		Resolver:    manifestResolver{RuntimeIdentity: context.ResolverRuntime, LockfileSHA256: context.LockfileDigest.String()},
		Primary:     set.Inspected().Graph().Primary().String(),
		Policy:      manifestPolicy{Decision: string(decision.Decision()), ID: decision.PolicyID(), Version: decision.Version(), Reasons: reasons},
		Entries:     entries,
		Edges:       edges,
	}
	unsigned, err := canonicalManifestPayload(payload)
	if err != nil {
		return domain.VerifiedBundle{}, errors.New("serialize canonical Manifest payload")
	}
	sum := sha256.Sum256(unsigned)
	manifestID, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	document, err := json.Marshal(manifestDocument{ManifestID: manifestID.String(), manifestPayload: payload})
	if err != nil || !json.Valid(document) {
		return domain.VerifiedBundle{}, errors.New("serialize canonical Manifest")
	}
	sbom, err := buildSBOM(manifestID, set)
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	return domain.NewVerifiedBundle(manifestID, set, document, sbom)
}

func canonicalManifestPayload(payload manifestPayload) ([]byte, error) {
	document, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	return json.Marshal(object)
}

func validContextText(value string) bool {
	return value != "" && value == strings.TrimSpace(value) && len(value) <= maximumContextText && utf8.ValidString(value) && strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) < 0
}

type manifestDocument struct {
	ManifestID string `json:"manifest_id"`
	manifestPayload
}

type manifestPayload struct {
	Schema      string           `json:"schema"`
	OperationID string           `json:"operation_id"`
	Target      manifestTarget   `json:"target"`
	Resolver    manifestResolver `json:"resolver"`
	Primary     string           `json:"primary"`
	Policy      manifestPolicy   `json:"policy"`
	Entries     []manifestEntry  `json:"entries"`
	Edges       []manifestEdge   `json:"edges"`
}

type manifestTarget struct {
	Path        string `json:"path"`
	RequiresNew bool   `json:"requires_new"`
}

type manifestResolver struct {
	RuntimeIdentity string `json:"runtime_identity"`
	LockfileSHA256  string `json:"lockfile_sha256"`
}

type manifestPolicy struct {
	Decision string   `json:"decision"`
	ID       string   `json:"id"`
	Version  uint64   `json:"version"`
	Reasons  []string `json:"reasons"`
}

type manifestEntry struct {
	Node              string         `json:"node"`
	RecordPath        string         `json:"record_path"`
	Role              string         `json:"role"`
	Source            string         `json:"source"`
	Name              string         `json:"name"`
	Version           string         `json:"version"`
	Variant           string         `json:"variant"`
	SHA256            string         `json:"sha256"`
	DeclaredIntegrity string         `json:"declared_integrity"`
	RunID             string         `json:"run_id"`
	Evidence          []string       `json:"evidence"`
	Policy            manifestPolicy `json:"policy"`
}

type manifestEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func manifestEntries(set domain.VerifiedSet) ([]manifestEntry, []manifestEdge, error) {
	inspections := make(map[domain.DependencyNodeID]domain.DependencyInspection)
	for _, inspection := range set.Inspected().Inspections() {
		inspections[inspection.Node()] = inspection
	}
	entries := make([]manifestEntry, 0, len(inspections))
	for _, dependency := range set.Inspected().Graph().Nodes() {
		inspection, ok := inspections[dependency.Node()]
		if !ok || inspection.Artifact().Identity() != dependency.Artifact().Identity() {
			return nil, nil, errors.New("manifest entry does not match verified set")
		}
		declared, ok := inspection.Artifact().DeclaredIntegrity()
		if !ok {
			return nil, nil, errors.New("manifest entry lacks declared integrity")
		}
		evidence := make([]string, 0, len(inspection.Evidence()))
		for _, reference := range inspection.Evidence() {
			evidence = append(evidence, reference.Handle())
		}
		sort.Strings(evidence)
		entryDecision := inspection.PolicyDecision()
		entryReasons := entryDecision.Reasons()
		sort.Strings(entryReasons)
		identity := inspection.Artifact().Identity()
		entries = append(entries, manifestEntry{
			Node: dependency.Node().String(), RecordPath: dependency.RecordPath(), Role: string(dependency.Role()),
			Source: identity.Source().String(), Name: identity.Name(), Version: identity.Version(), Variant: identity.Variant(),
			SHA256: inspection.Artifact().Digest().String(), DeclaredIntegrity: declared, RunID: inspection.RunID().String(), Evidence: evidence,
			Policy: manifestPolicy{Decision: string(entryDecision.Decision()), ID: entryDecision.PolicyID(), Version: entryDecision.Version(), Reasons: entryReasons},
		})
	}
	edges := make([]manifestEdge, 0, len(set.Inspected().Graph().Edges()))
	for _, edge := range set.Inspected().Graph().Edges() {
		edges = append(edges, manifestEdge{From: edge.From().String(), To: edge.To().String()})
	}
	return entries, edges, nil
}

type cycloneDXBOM struct {
	Schema       string                `json:"$schema"`
	BOMFormat    string                `json:"bomFormat"`
	SpecVersion  string                `json:"specVersion"`
	Version      int                   `json:"version"`
	Metadata     cycloneDXMetadata     `json:"metadata"`
	Components   []cycloneDXComponent  `json:"components"`
	Dependencies []cycloneDXDependency `json:"dependencies"`
}

type cycloneDXMetadata struct {
	Lifecycles []cycloneDXLifecycle `json:"lifecycles"`
	Properties []cycloneDXProperty  `json:"properties"`
}

type cycloneDXLifecycle struct {
	Phase string `json:"phase"`
}
type cycloneDXProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type cycloneDXHash struct {
	Algorithm string `json:"alg"`
	Content   string `json:"content"`
}
type cycloneDXComponent struct {
	Type       string              `json:"type"`
	BOMRef     string              `json:"bom-ref"`
	Name       string              `json:"name"`
	Version    string              `json:"version"`
	Hashes     []cycloneDXHash     `json:"hashes"`
	Properties []cycloneDXProperty `json:"properties"`
}
type cycloneDXDependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

func buildSBOM(manifestID domain.ContentDigest, set domain.VerifiedSet) ([]byte, error) {
	inspections := make(map[domain.DependencyNodeID]domain.DependencyInspection)
	for _, inspection := range set.Inspected().Inspections() {
		inspections[inspection.Node()] = inspection
	}
	components := make([]cycloneDXComponent, 0, len(inspections))
	dependencies := make(map[string][]string, len(inspections))
	for _, dependency := range set.Inspected().Graph().Nodes() {
		inspection, ok := inspections[dependency.Node()]
		if !ok {
			return nil, errors.New("SBOM component does not match verified set")
		}
		properties := []cycloneDXProperty{{Name: "heliopause:inspection-run", Value: inspection.RunID().String()}}
		for _, evidence := range inspection.Evidence() {
			properties = append(properties, cycloneDXProperty{Name: "heliopause:evidence-reference", Value: evidence.Handle()})
		}
		sort.Slice(properties, func(i, j int) bool {
			if properties[i].Name == properties[j].Name {
				return properties[i].Value < properties[j].Value
			}
			return properties[i].Name < properties[j].Name
		})
		identity := inspection.Artifact().Identity()
		components = append(components, cycloneDXComponent{
			Type: "library", BOMRef: dependency.Node().String(), Name: identity.Name(), Version: identity.Version(),
			Hashes: []cycloneDXHash{{Algorithm: "SHA-256", Content: inspection.Artifact().Digest().String()}}, Properties: properties,
		})
		dependencies[dependency.Node().String()] = nil
	}
	for _, edge := range set.Inspected().Graph().Edges() {
		from := edge.From().String()
		dependencies[from] = append(dependencies[from], edge.To().String())
	}
	relations := make([]cycloneDXDependency, 0, len(dependencies))
	for _, component := range components {
		children := dependencies[component.BOMRef]
		sort.Strings(children)
		if children == nil {
			children = []string{}
		}
		relations = append(relations, cycloneDXDependency{Ref: component.BOMRef, DependsOn: children})
	}
	bom := cycloneDXBOM{
		Schema: cycloneDXSchema, BOMFormat: "CycloneDX", SpecVersion: "1.7", Version: 1,
		Metadata:   cycloneDXMetadata{Lifecycles: []cycloneDXLifecycle{{Phase: "pre-build"}}, Properties: []cycloneDXProperty{{Name: "heliopause:manifest-id", Value: manifestID.String()}}},
		Components: components, Dependencies: relations,
	}
	if len(bom.Components) == 0 || len(bom.Components) != len(bom.Dependencies) {
		return nil, errors.New("CycloneDX SBOM graph is incomplete")
	}
	document, err := json.Marshal(bom)
	if err != nil || !json.Valid(document) {
		return nil, errors.New("serialize CycloneDX 1.7 SBOM")
	}
	return document, nil
}
