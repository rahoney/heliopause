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

type projectManifestPayload struct {
	Schema          string                   `json:"schema"`
	OperationID     string                   `json:"operation_id"`
	Target          manifestTarget           `json:"target"`
	Resolver        manifestResolver         `json:"resolver"`
	DependencyFree  bool                     `json:"dependency_free"`
	Policy          manifestPolicy           `json:"policy"`
	Entries         []manifestEntry          `json:"entries"`
	Edges           []manifestEdge           `json:"edges"`
	ProjectControls []projectManifestControl `json:"project_controls"`
}

// BuildProject emits records for an actual complete approved project, including
// an explicitly empty set. It does not weaken the requested-primary builder.
func (Generator) BuildProject(ctx context.Context, operation domain.OperationID, set domain.ProjectVerifiedSet, nodes []domain.LockedDependency, edges []domain.DependencyEdge, controls []domain.ProjectControlFile, runtime string) (domain.ContentDigest, []byte, []byte, error) {
	return buildProjectRecords(ctx, operation, set, nodes, edges, controls, runtime, false)
}

// BuildProjectWithControlPresence binds explicit absent controls without manufacturing files.
// The ordinary builder retains its all-present contract.
func (Generator) BuildProjectWithControlPresence(ctx context.Context, operation domain.OperationID, set domain.ProjectVerifiedSet, nodes []domain.LockedDependency, edges []domain.DependencyEdge, controls []domain.ProjectControlFile, runtime string) (domain.ContentDigest, []byte, []byte, error) {
	return buildProjectRecords(ctx, operation, set, nodes, edges, controls, runtime, true)
}

type projectManifestControl struct {
	Name    string `json:"name"`
	SHA256  string `json:"sha256"`
	Content []byte `json:"content"`
	Absent  bool   `json:"absent,omitempty"`
}

func buildProjectRecords(ctx context.Context, operation domain.OperationID, set domain.ProjectVerifiedSet, nodes []domain.LockedDependency, edges []domain.DependencyEdge, controls []domain.ProjectControlFile, runtime string, controlPresence bool) (domain.ContentDigest, []byte, []byte, error) {
	if ctx == nil || operation.String() == "" || !set.Valid() || !validContextText(runtime) {
		return domain.ContentDigest{}, nil, nil, errors.New("project records require actual operation, approval and runtime")
	}
	if err := ctx.Err(); err != nil {
		return domain.ContentDigest{}, nil, nil, err
	}
	snapshot := set.Inspected().Snapshot()
	if len(nodes) != len(snapshot.Dependencies()) || len(controls) != len(snapshot.ControlDigests()) || len(controls) > 32 {
		return domain.ContentDigest{}, nil, nil, errors.New("project record coverage differs from approval")
	}
	ordered := append([]domain.LockedDependency(nil), nodes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Node().String() < ordered[j].Node().String() })
	resolved := map[domain.ResolvedArtifactIdentity]domain.ResolvedArtifact{}
	for _, a := range snapshot.Dependencies() {
		resolved[a.Identity()] = a
	}
	known := map[domain.DependencyNodeID]bool{}
	paths := map[string]bool{}
	for _, node := range ordered {
		a := node.Artifact()
		id, err := domain.ProjectDependencyNodeID(a.Identity())
		if err != nil || id != node.Node() || known[id] || paths[node.RecordPath()] || node.RecordPath() == "" || node.Role() != domain.DependencyTransitive || node.HostInstallAction() || resolved[a.Identity()] != a {
			return domain.ContentDigest{}, nil, nil, errors.New("project record invented a primary or changed an approved subject")
		}
		known[id] = true
		paths[node.RecordPath()] = true
	}
	inspections := map[domain.DependencyNodeID]domain.DependencyInspection{}
	for _, inspection := range set.Inspected().Inspections() {
		inspections[inspection.Node()] = inspection
	}
	reasons := set.Decision().Reasons()
	sort.Strings(reasons)
	payload := projectManifestPayload{Schema: "helox.verified-project-manifest/v1", OperationID: operation.String(), Target: manifestTarget{Path: snapshot.Context().Target().String(), RequiresNew: snapshot.Context().RequiresNewTarget()}, Resolver: manifestResolver{runtime, snapshot.GraphDigest().String()}, DependencyFree: snapshot.DependencyFree(), Policy: manifestPolicy{string(set.Decision().Decision()), set.Decision().PolicyID(), set.Decision().Version(), reasons}, Entries: []manifestEntry{}, Edges: []manifestEdge{}, ProjectControls: []projectManifestControl{}}
	for _, node := range ordered {
		i, ok := inspections[node.Node()]
		if !ok {
			return domain.ContentDigest{}, nil, nil, errors.New("project record inspection is missing")
		}
		entry, err := manifestEntryFor(node, i)
		if err != nil {
			return domain.ContentDigest{}, nil, nil, err
		}
		payload.Entries = append(payload.Entries, entry)
	}
	seenEdges := map[domain.DependencyEdge]bool{}
	for _, edge := range edges {
		if !known[edge.From()] || !known[edge.To()] || edge.From() == edge.To() || seenEdges[edge] {
			return domain.ContentDigest{}, nil, nil, errors.New("project records contain an invalid dependency edge")
		}
		seenEdges[edge] = true
		payload.Edges = append(payload.Edges, manifestEdge{edge.From().String(), edge.To().String()})
	}
	sort.Slice(payload.Edges, func(i, j int) bool {
		a, b := payload.Edges[i], payload.Edges[j]
		return a.From < b.From || a.From == b.From && a.To < b.To
	})
	byName := map[string]domain.ProjectControlFile{}
	for _, c := range controls {
		if !controlPresence && !c.Present() || c.Name() == "" || byName[c.Name()].Name() != "" {
			return domain.ContentDigest{}, nil, nil, errors.New("project record controls are missing or repeated")
		}
		byName[c.Name()] = c
	}
	for _, digest := range snapshot.ControlDigests() {
		c := byName[digest.Name()]
		if c.Digest() != digest.Digest() {
			return domain.ContentDigest{}, nil, nil, errors.New("project record control differs from approval")
		}
		payload.ProjectControls = append(payload.ProjectControls, projectManifestControl{c.Name(), c.Digest().String(), c.Body(), !c.Present()})
	}
	unsigned, err := canonicalManifestObject(payload)
	if err != nil {
		return domain.ContentDigest{}, nil, nil, err
	}
	sum := sha256.Sum256(unsigned)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.ContentDigest{}, nil, nil, err
	}
	doc, err := json.Marshal(struct {
		ManifestID string `json:"manifest_id"`
		projectManifestPayload
	}{digest.String(), payload})
	if err != nil {
		return domain.ContentDigest{}, nil, nil, err
	}
	sbom, err := buildSelectionSBOM(digest, ordered, set.Inspected().Inspections(), edges, snapshot.DependencyFree())
	if err != nil {
		return domain.ContentDigest{}, nil, nil, err
	}
	if len(doc) > 32<<20 || len(sbom) > 32<<20 {
		return domain.ContentDigest{}, nil, nil, errors.New("project records exceed bounded content")
	}
	if err := ctx.Err(); err != nil {
		return domain.ContentDigest{}, nil, nil, err
	}
	return digest, doc, sbom, nil
}
