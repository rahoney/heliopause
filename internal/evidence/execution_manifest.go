package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type executionManifestPayload struct {
	Schema      string                  `json:"schema"`
	OperationID string                  `json:"operation_id"`
	Resolver    manifestResolver        `json:"resolver"`
	Primary     string                  `json:"primary"`
	Roots       []string                `json:"roots,omitempty"`
	Policy      manifestPolicy          `json:"policy"`
	Entries     []manifestEntry         `json:"entries"`
	Edges       []manifestEdge          `json:"edges"`
	Execution   executionManifestMember `json:"execution"`
}

type executionManifestMember struct {
	Command           string `json:"command"`
	Path              string `json:"path"`
	SHA256            string `json:"sha256"`
	Size              uint64 `json:"size"`
	RuntimeIdentity   string `json:"runtime_identity"`
	SourceFingerprint string `json:"source_fingerprint"`
	PolicyFingerprint string `json:"policy_fingerprint"`
}

// BuildExecution records the current approved set without an installation
// target. Arguments and program output never enter these storage records.
// The bundle is not an execution request or an admission authorization.
func (Generator) BuildExecution(ctx context.Context, execution domain.ExecutionContext) (domain.VerifiedBundle, error) {
	if ctx == nil || !execution.Valid() || !validContextText(execution.RuntimeIdentity()) || !validContextText(execution.Resolution().RuntimeIdentity()) {
		return domain.VerifiedBundle{}, errors.New("execution records require a valid context and approved set")
	}
	if err := ctx.Err(); err != nil {
		return domain.VerifiedBundle{}, err
	}
	set := execution.Set()
	entries, edges, _, err := manifestEntries(set)
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	decision := set.Decision()
	reasons := decision.Reasons()
	sort.Strings(reasons)
	payload := executionManifestPayload{
		Schema: "helox.verified-execution-manifest/v1", OperationID: execution.OperationID().String(),
		Resolver: manifestResolver{execution.Resolution().RuntimeIdentity(), execution.SourceFingerprint().String()},
		Primary:  set.Inspected().Graph().Primary().String(),
		Policy:   manifestPolicy{string(decision.Decision()), decision.PolicyID(), decision.Version(), reasons},
		Entries:  entries, Edges: edges,
		Execution: executionManifestMember{
			Command: execution.Command(), Path: execution.EntryPath(), SHA256: execution.EntryDigest().String(),
			Size: execution.EntrySize(), RuntimeIdentity: execution.RuntimeIdentity(),
			SourceFingerprint: execution.SourceFingerprint().String(), PolicyFingerprint: execution.PolicyFingerprint().String(),
		},
	}
	id, document, sbom, err := buildExecutionDocuments(payload, set.Inspected().Graph().Nodes(), set.Inspected().Inspections(), set.Inspected().Graph().Edges())
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	return domain.NewVerifiedBundle(id, set, document, sbom)
}

// BuildExecutionSelection records all explicitly selected roots without an
// install target, synthetic dependency edges or private program arguments.
func (Generator) BuildExecutionSelection(ctx context.Context, execution domain.ExecutionSelectionContext) (domain.ExecutionSelectionBundle, error) {
	if ctx == nil || !execution.Valid() || !validContextText(execution.RuntimeIdentity()) || !validContextText(execution.Resolution().RuntimeIdentity()) {
		return domain.ExecutionSelectionBundle{}, errors.New("execution records require a complete approved root selection")
	}
	if err := ctx.Err(); err != nil {
		return domain.ExecutionSelectionBundle{}, err
	}
	set := execution.Set()
	selection := set.Inspected().Selection()
	byNode := map[domain.DependencyNodeID]domain.DependencyInspection{}
	for _, inspection := range set.Inspected().Inspections() {
		byNode[inspection.Node()] = inspection
	}
	entries := make([]manifestEntry, 0, len(byNode))
	for _, node := range selection.Nodes() {
		entry, err := manifestEntryFor(node, byNode[node.Node()])
		if err != nil {
			return domain.ExecutionSelectionBundle{}, err
		}
		entries = append(entries, entry)
	}
	edges := make([]manifestEdge, 0, len(selection.Edges()))
	for _, edge := range selection.Edges() {
		edges = append(edges, manifestEdge{edge.From().String(), edge.To().String()})
	}
	roots := make([]string, 0, len(selection.RootGraphs()))
	for _, root := range selection.RootGraphs() {
		roots = append(roots, root.Primary().String())
	}
	decision := set.Decision()
	reasons := decision.Reasons()
	sort.Strings(reasons)
	payload := executionManifestPayload{
		Schema: "helox.verified-execution-selection-manifest/v1", OperationID: execution.OperationID().String(),
		Resolver: manifestResolver{execution.Resolution().RuntimeIdentity(), execution.SourceFingerprint().String()},
		Primary:  selection.Primary().String(), Roots: roots,
		Policy:  manifestPolicy{string(decision.Decision()), decision.PolicyID(), decision.Version(), reasons},
		Entries: entries, Edges: edges,
		Execution: executionManifestMember{Command: execution.Command(), Path: execution.EntryPath(), SHA256: execution.EntryDigest().String(), Size: execution.EntrySize(), RuntimeIdentity: execution.RuntimeIdentity(), SourceFingerprint: execution.SourceFingerprint().String(), PolicyFingerprint: execution.PolicyFingerprint().String()},
	}
	id, document, sbom, err := buildExecutionDocuments(payload, selection.Nodes(), set.Inspected().Inspections(), selection.Edges())
	if err != nil {
		return domain.ExecutionSelectionBundle{}, err
	}
	return domain.NewExecutionSelectionBundle(id, set, document, sbom)
}

func buildExecutionDocuments(payload executionManifestPayload, nodes []domain.LockedDependency, inspections []domain.DependencyInspection, edges []domain.DependencyEdge) (domain.ContentDigest, []byte, []byte, error) {
	unsigned, err := canonicalManifestObject(payload)
	if err != nil {
		return domain.ContentDigest{}, nil, nil, err
	}
	sum := sha256.Sum256(unsigned)
	id, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.ContentDigest{}, nil, nil, err
	}
	document, err := json.Marshal(struct {
		ManifestID string `json:"manifest_id"`
		executionManifestPayload
	}{id.String(), payload})
	if err != nil {
		return domain.ContentDigest{}, nil, nil, err
	}
	sbom, err := buildSelectionSBOM(id, nodes, inspections, edges, false)
	return id, document, sbom, err
}
