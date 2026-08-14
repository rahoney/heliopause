package domain

import (
	"errors"
	"path"
	"sort"
)

const (
	maxInstallTargetLength = 2048
	maxDependencyNodes     = 4096
	maxDependencyEdges     = 16384
)

// InstallTarget is a lexical, canonical absolute target selected for a new
// installation. It intentionally does not inspect the Host filesystem.
type InstallTarget struct{ value string }

// NewInstallTarget validates the target syntax required by the M4 no-overwrite
// contract. The trusted Promotion boundary later verifies containment, symlink
// safety, and that the target does not already exist.
func NewInstallTarget(value string) (InstallTarget, error) {
	if err := validateBoundedText(value, maxInstallTargetLength, "install target"); err != nil {
		return InstallTarget{}, err
	}
	if !path.IsAbs(value) || value == "/" || path.Clean(value) != value {
		return InstallTarget{}, errors.New("install target must be a canonical non-root absolute path")
	}
	return InstallTarget{value: value}, nil
}

func (t InstallTarget) String() string { return t.value }

// InstallContext preserves the only M4 MVP installation choice. It has no
// package-manager option passthrough and always requires a new target.
type InstallContext struct{ target InstallTarget }

func NewInstallContext(target InstallTarget) (InstallContext, error) {
	if target.value == "" {
		return InstallContext{}, errors.New("install target is required")
	}
	return InstallContext{target: target}, nil
}

func (c InstallContext) Target() InstallTarget   { return c.target }
func (c InstallContext) RequiresNewTarget() bool { return true }

// DependencyNodeID is an adapter-generated opaque identifier for one exact
// graph node. It is not a package-manager lockfile path.
type DependencyNodeID struct{ value string }

func NewDependencyNodeID(value string) (DependencyNodeID, error) {
	if err := validateNormalizedIdentifier(value, maxCoordinateLength, "dependency node ID"); err != nil {
		return DependencyNodeID{}, err
	}
	return DependencyNodeID{value: value}, nil
}

func (id DependencyNodeID) String() string { return id.value }

type DependencyRole string

const (
	DependencyPrimary    DependencyRole = "PRIMARY"
	DependencyTransitive DependencyRole = "DEPENDENCY"
)

// LockedDependency is a parser-normalized exact acquisition candidate.
// ResolvedArtifact retains the source-neutral identity, locator and declared
// integrity needed by the existing Artifact Port.
type LockedDependency struct {
	node              DependencyNodeID
	role              DependencyRole
	artifact          ResolvedArtifact
	recordPath        string
	hostInstallAction bool
}

func NewLockedDependency(node DependencyNodeID, role DependencyRole, artifact ResolvedArtifact) (LockedDependency, error) {
	return NewLockedDependencyWithRecordPath(node, role, artifact, node.String(), false)
}

// NewLockedDependencyWithHostInstallAction preserves parser-normalized Host
// lifecycle/native-install metadata for set-level Policy evaluation only.
func NewLockedDependencyWithHostInstallAction(node DependencyNodeID, role DependencyRole, artifact ResolvedArtifact, hostInstallAction bool) (LockedDependency, error) {
	return NewLockedDependencyWithRecordPath(node, role, artifact, node.String(), hostInstallAction)
}

// NewLockedDependencyWithRecordPath preserves a bounded resolver record path
// for Manifest traceability without exposing its ecosystem-specific semantics.
func NewLockedDependencyWithRecordPath(node DependencyNodeID, role DependencyRole, artifact ResolvedArtifact, recordPath string, hostInstallAction bool) (LockedDependency, error) {
	if node.value == "" || artifact.identity.source.value == "" || artifact.declaredIntegrity == "" {
		return LockedDependency{}, errors.New("locked dependency requires node, exact artifact, and declared integrity")
	}
	if role != DependencyPrimary && role != DependencyTransitive {
		return LockedDependency{}, errors.New("locked dependency role is invalid")
	}
	if err := validateBoundedText(recordPath, maxInstallTargetLength, "dependency record path"); err != nil {
		return LockedDependency{}, err
	}
	return LockedDependency{node: node, role: role, artifact: artifact, recordPath: recordPath, hostInstallAction: hostInstallAction}, nil
}

func (d LockedDependency) Node() DependencyNodeID     { return d.node }
func (d LockedDependency) Role() DependencyRole       { return d.role }
func (d LockedDependency) Artifact() ResolvedArtifact { return d.artifact }
func (d LockedDependency) RecordPath() string         { return d.recordPath }
func (d LockedDependency) HostInstallAction() bool    { return d.hostInstallAction }

// DependencyEdge declares a graph relation using opaque node IDs.
type DependencyEdge struct{ from, to DependencyNodeID }

func NewDependencyEdge(from, to DependencyNodeID) (DependencyEdge, error) {
	if from.value == "" || to.value == "" || from == to {
		return DependencyEdge{}, errors.New("dependency edge must join distinct nodes")
	}
	return DependencyEdge{from: from, to: to}, nil
}

func (e DependencyEdge) From() DependencyNodeID { return e.from }
func (e DependencyEdge) To() DependencyNodeID   { return e.to }

// LockedDependencyGraph is a bounded, connected exact graph emitted by a
// trusted resolver after it has parsed ecosystem-specific lock data.
type LockedDependencyGraph struct {
	primary DependencyNodeID
	nodes   []LockedDependency
	edges   []DependencyEdge
}

func NewLockedDependencyGraph(nodes []LockedDependency, edges []DependencyEdge) (LockedDependencyGraph, error) {
	if len(nodes) == 0 || len(nodes) > maxDependencyNodes || len(edges) > maxDependencyEdges {
		return LockedDependencyGraph{}, errors.New("locked dependency graph exceeds bounds")
	}
	byNode := make(map[DependencyNodeID]LockedDependency, len(nodes))
	var primary DependencyNodeID
	for _, node := range nodes {
		if node.node.value == "" || node.artifact.identity.source.value == "" || node.artifact.declaredIntegrity == "" {
			return LockedDependencyGraph{}, errors.New("locked dependency graph contains an invalid node")
		}
		if _, exists := byNode[node.node]; exists {
			return LockedDependencyGraph{}, errors.New("locked dependency graph contains duplicate node IDs")
		}
		if node.role == DependencyPrimary {
			if primary.value != "" {
				return LockedDependencyGraph{}, errors.New("locked dependency graph requires exactly one primary node")
			}
			primary = node.node
		} else if node.role != DependencyTransitive {
			return LockedDependencyGraph{}, errors.New("locked dependency graph contains an invalid role")
		}
		byNode[node.node] = node
	}
	if primary.value == "" {
		return LockedDependencyGraph{}, errors.New("locked dependency graph requires one primary node")
	}
	adjacent := make(map[DependencyNodeID][]DependencyNodeID, len(nodes))
	inbound := make(map[DependencyNodeID]bool, len(nodes))
	seenEdges := make(map[DependencyEdge]bool, len(edges))
	for _, edge := range edges {
		if _, ok := byNode[edge.from]; !ok {
			return LockedDependencyGraph{}, errors.New("dependency edge source is unknown")
		}
		if _, ok := byNode[edge.to]; !ok || edge.from == edge.to {
			return LockedDependencyGraph{}, errors.New("dependency edge target is invalid")
		}
		if seenEdges[edge] {
			return LockedDependencyGraph{}, errors.New("locked dependency graph contains duplicate edges")
		}
		seenEdges[edge] = true
		adjacent[edge.from] = append(adjacent[edge.from], edge.to)
		inbound[edge.to] = true
	}
	if inbound[primary] {
		return LockedDependencyGraph{}, errors.New("primary dependency cannot have an inbound edge")
	}
	visited := map[DependencyNodeID]bool{primary: true}
	queue := []DependencyNodeID{primary}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacent[current] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	if len(visited) != len(nodes) {
		return LockedDependencyGraph{}, errors.New("locked dependency graph must be connected to its primary node")
	}
	copyNodes := append([]LockedDependency(nil), nodes...)
	sort.Slice(copyNodes, func(i, j int) bool { return copyNodes[i].node.value < copyNodes[j].node.value })
	copyEdges := append([]DependencyEdge(nil), edges...)
	sort.Slice(copyEdges, func(i, j int) bool {
		if copyEdges[i].from.value == copyEdges[j].from.value {
			return copyEdges[i].to.value < copyEdges[j].to.value
		}
		return copyEdges[i].from.value < copyEdges[j].from.value
	})
	return LockedDependencyGraph{primary: primary, nodes: copyNodes, edges: copyEdges}, nil
}

func (g LockedDependencyGraph) Primary() DependencyNodeID { return g.primary }
func (g LockedDependencyGraph) Nodes() []LockedDependency {
	return append([]LockedDependency(nil), g.nodes...)
}
func (g LockedDependencyGraph) Edges() []DependencyEdge {
	return append([]DependencyEdge(nil), g.edges...)
}

// DependencyResolution binds an exact graph to the resolver runtime and raw
// lockfile content identity that produced it. Raw lockfile bytes stay inside
// the resolver adapter.
type DependencyResolution struct {
	graph           LockedDependencyGraph
	runtimeIdentity string
	lockfileDigest  ContentDigest
}

func NewDependencyResolution(graph LockedDependencyGraph, runtimeIdentity string, lockfileDigest ContentDigest) (DependencyResolution, error) {
	if len(graph.nodes) == 0 || lockfileDigest.String() == "" {
		return DependencyResolution{}, errors.New("dependency resolution requires graph and lockfile digest")
	}
	if err := validateBoundedText(runtimeIdentity, maxInstallTargetLength, "resolver runtime identity"); err != nil {
		return DependencyResolution{}, err
	}
	return DependencyResolution{graph: graph, runtimeIdentity: runtimeIdentity, lockfileDigest: lockfileDigest}, nil
}

func (r DependencyResolution) Graph() LockedDependencyGraph { return r.graph }
func (r DependencyResolution) RuntimeIdentity() string      { return r.runtimeIdentity }
func (r DependencyResolution) LockfileDigest() ContentDigest {
	return r.lockfileDigest
}
