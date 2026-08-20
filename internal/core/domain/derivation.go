package domain

import (
	"errors"
	"sort"
)

// DerivationBinding retains the generic, reproducible recipe that produced a
// controller-derived Artifact. Adapter-specific build APIs must not cross this
// boundary; the binding is deliberately expressed only in exact digests and
// bounded identifiers.
type DerivationBinding struct {
	sourceDigest ContentDigest
	inputDigests []ContentDigest
	executor     string
	configDigest ContentDigest
}

func NewDerivationBinding(sourceDigest ContentDigest, inputDigests []ContentDigest, executor string, configDigest ContentDigest) (DerivationBinding, error) {
	if sourceDigest.value == "" || configDigest.value == "" || executor == "" {
		return DerivationBinding{}, errors.New("derivation binding requires source, inputs, executor, and configuration")
	}
	if err := validateNormalizedIdentifier(executor, 64, "derivation executor"); err != nil || len(inputDigests) == 0 {
		return DerivationBinding{}, errors.New("derivation binding is invalid")
	}
	inputs := append([]ContentDigest(nil), inputDigests...)
	seen := map[ContentDigest]bool{}
	for _, digest := range inputs {
		if digest.value == "" || seen[digest] {
			return DerivationBinding{}, errors.New("derivation inputs are invalid")
		}
		seen[digest] = true
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].value < inputs[j].value })
	return DerivationBinding{sourceDigest: sourceDigest, inputDigests: inputs, executor: executor, configDigest: configDigest}, nil
}

func (b DerivationBinding) SourceDigest() ContentDigest { return b.sourceDigest }
func (b DerivationBinding) InputDigests() []ContentDigest {
	return append([]ContentDigest(nil), b.inputDigests...)
}
func (b DerivationBinding) Executor() string            { return b.executor }
func (b DerivationBinding) ConfigDigest() ContentDigest { return b.configDigest }

// DerivedDependency is a controller-produced Artifact which is linked to a
// previously locked source node. It is not a replacement for that source.
type DerivedDependency struct {
	source   DependencyNodeID
	node     LockedDependency
	artifact AcquiredArtifact
	binding  DerivationBinding
	check    CheckExecution
	evidence Evidence
}

func NewDerivedDependency(source DependencyNodeID, node LockedDependency, artifact AcquiredArtifact, binding DerivationBinding, check CheckExecution, evidence Evidence) (DerivedDependency, error) {
	if source.value == "" || node.node.value == "" || node.artifact.identity.variant != "derived-wheel" || artifact.identity != node.artifact.identity || artifact.declaredIntegrity != node.artifact.declaredIntegrity || binding.sourceDigest.value == "" || !check.required || check.kind != CheckInspection || check.capability != CapabilitySupported || check.status != ExecutionCompleted || evidence.checkID != check.id || evidence.identity != artifact.identity || evidence.digest != artifact.digest {
		return DerivedDependency{}, errors.New("derived dependency binding is invalid")
	}
	return DerivedDependency{source: source, node: node, artifact: artifact, binding: binding, check: check, evidence: evidence}, nil
}
func (d DerivedDependency) Source() DependencyNodeID   { return d.source }
func (d DerivedDependency) Node() LockedDependency     { return d.node }
func (d DerivedDependency) Artifact() AcquiredArtifact { return d.artifact }
func (d DerivedDependency) Binding() DerivationBinding { return d.binding }
func (d DerivedDependency) Check() CheckExecution      { return d.check }
func (d DerivedDependency) Evidence() Evidence         { return d.evidence }

// ExtendLockedDependencyGraph appends controller-derived nodes and source-to-
// derived edges without changing resolver-owned nodes or edges.
func ExtendLockedDependencyGraph(graph LockedDependencyGraph, derived []DerivedDependency) (LockedDependencyGraph, error) {
	nodes, edges := graph.Nodes(), graph.Edges()
	bindings := graph.Derivations()
	known := map[DependencyNodeID]bool{}
	for _, node := range nodes {
		known[node.Node()] = true
	}
	for _, item := range derived {
		if !known[item.source] || known[item.node.Node()] {
			return LockedDependencyGraph{}, errors.New("derived dependency graph binding is invalid")
		}
		nodes = append(nodes, item.node)
		edge, err := NewDependencyEdge(item.source, item.node.Node())
		if err != nil {
			return LockedDependencyGraph{}, err
		}
		edges = append(edges, edge)
		bindings = append(bindings, DerivedGraphBinding{source: item.source, derived: item.node.Node(), binding: item.binding})
		known[item.node.Node()] = true
	}
	return newLockedDependencyGraph(nodes, edges, bindings)
}
