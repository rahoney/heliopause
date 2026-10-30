package npm

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"reflect"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errNodeExecutionLayout = errors.New("node execution layout differs from its current complete source binding")

type executionLayoutBinding struct {
	Operation                domain.OperationID
	Resolution               domain.DependencyResolution
	Set                      domain.VerifiedSet
	Artifact                 domain.AcquiredArtifact
	Command, Member, Runtime string
	Digest                   domain.ContentDigest
	Size                     uint64
}

func executionLayoutFacts(c domain.ExecutionContext) executionLayoutBinding {
	return executionLayoutBinding{c.OperationID(), c.Resolution(), c.Set(), c.EntryArtifact(), c.Command(), c.EntryPath(), c.RuntimeIdentity(), c.EntryDigest(), c.EntrySize()}
}

// NodeExecutionLayout binds a parsed tree to current whole-set source facts.
// It grants no runtime admission, immutable mount or program completion proof.
type NodeExecutionLayout struct {
	binding    executionLayoutBinding
	layout     PNPMLayoutArchive
	entry      string
	digest     domain.ContentDigest
	nativePNPM bool
}

func (l NodeExecutionLayout) Valid() bool                          { return l.digest.String() != "" }
func (l NodeExecutionLayout) Digest() domain.ContentDigest         { return l.digest }
func (l NodeExecutionLayout) EntryPath() string                    { return l.entry }
func (l NodeExecutionLayout) Entries() []PNPMLayoutEntry           { return l.layout.Entries() }
func (l NodeExecutionLayout) ReadFile(name string) ([]byte, error) { return l.layout.ReadFile(name) }
func (l NodeExecutionLayout) MatchesExecution(c domain.ExecutionContext) bool {
	return l.Valid() && c.Valid() && reflect.DeepEqual(l.binding, executionLayoutFacts(c))
}
func (NodeExecutionLayout) String() string   { return "private node execution layout" }
func (NodeExecutionLayout) GoString() string { return "private node execution layout" }
func (NodeExecutionLayout) MarshalJSON() ([]byte, error) {
	return nil, errors.New("node execution layout is private memory data")
}

// BindNodeExecutionLayout reuses the bounded layout parser's measured bytes.
// Every package member must match its exact independently inspected archive.
func BindNodeExecutionLayout(ctx context.Context, execution domain.ExecutionContext, inventories []ArchiveInventory, layout PNPMLayoutArchive) (NodeExecutionLayout, error) {
	if ctx == nil || !execution.Valid() || len(layout.entries) == 0 {
		return NodeExecutionLayout{}, errNodeExecutionLayout
	}
	if ctx.Err() != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, ctx.Err(), context.Cause(ctx))
	}
	byNode := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	var subjects []domain.AcquiredArtifact
	for _, inspection := range execution.Set().Inspected().Inspections() {
		byNode[inspection.Node()] = inspection.Artifact()
		subjects = append(subjects, inspection.Artifact())
	}
	bySubject, err := indexNPMProjectInventories(subjects, inventories, errNodeExecutionLayout)
	if err != nil {
		return NodeExecutionLayout{}, err
	}
	graph := execution.Resolution().Graph()
	paths := map[domain.DependencyNodeID]string{}
	lookup := map[string]parsedLockPackage{}
	wanted := map[string]PNPMLayoutEntry{}
	add := func(name string, fact PNPMLayoutEntry) error {
		if !validInventoryPath(name) {
			return errNodeExecutionLayout
		}
		if previous, present := wanted[name]; present && previous != fact {
			return errNodeExecutionLayout
		}
		wanted[name] = fact
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if previous, present := wanted[parent]; present && previous.Kind != tar.TypeDir {
				return errNodeExecutionLayout
			}
			wanted[parent] = PNPMLayoutEntry{Kind: tar.TypeDir}
		}
		if len(wanted) > pnpmLayoutEntryLimit {
			return errNodeExecutionLayout
		}
		return nil
	}
	for _, node := range graph.Nodes() {
		if ctx.Err() != nil {
			return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, ctx.Err(), context.Cause(ctx))
		}
		root := node.RecordPath()
		name, valid := packageNameFromLockPath(root)
		if !valid || name != node.Artifact().Identity().Name() || node.Artifact().Identity().Source().String() != "npm" {
			return NodeExecutionLayout{}, errNodeExecutionLayout
		}
		if _, duplicate := lookup[root]; duplicate {
			return NodeExecutionLayout{}, errNodeExecutionLayout
		}
		paths[node.Node()], lookup[root] = root, parsedLockPackage{}
		inventory, present := bySubject[byNode[node.Node()]]
		if !present {
			return NodeExecutionLayout{}, errNodeExecutionLayout
		}
		if _, err := pnpmInventoryBins(inventory); err != nil {
			return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
		}
		if err := add(root, PNPMLayoutEntry{Kind: tar.TypeDir}); err != nil {
			return NodeExecutionLayout{}, err
		}
		for _, member := range inventory.entries {
			fact := PNPMLayoutEntry{Kind: tar.TypeReg, Mode: member.Mode &^ 0o022, Size: member.Size, Digest: member.Digest}
			if member.Directory {
				fact = PNPMLayoutEntry{Kind: tar.TypeDir}
			}
			if err := add(path.Join(root, member.Path), fact); err != nil {
				return NodeExecutionLayout{}, err
			}
		}
	}
	for _, edge := range graph.Edges() {
		target := paths[edge.To()]
		name, valid := packageNameFromLockPath(target)
		actual, found := resolveLockedDependencyPath(paths[edge.From()], name, lookup)
		if !valid || !found || actual != target {
			return NodeExecutionLayout{}, errNodeExecutionLayout
		}
	}
	if len(layout.entries) != len(wanted) {
		return NodeExecutionLayout{}, errNodeExecutionLayout
	}
	for _, actual := range layout.entries {
		if ctx.Err() != nil {
			return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, ctx.Err(), context.Cause(ctx))
		}
		fact, present := wanted[actual.Name]
		if !present || actual.Kind != fact.Kind || !pnpmLayoutEntryModeOwner(actual, pnpmLayoutRequirement{kind: fact.Kind, executable: fact.Mode&0o111 != 0}, 1000, 1000) {
			return NodeExecutionLayout{}, errNodeExecutionLayout
		}
		if fact.Kind == tar.TypeReg && (actual.Mode != fact.Mode || actual.Size != fact.Size || actual.Digest != fact.Digest) {
			return NodeExecutionLayout{}, errNodeExecutionLayout
		}
	}
	primary := bySubject[execution.EntryArtifact()]
	selected, err := SelectExecutionEntry(primary, execution.Command())
	if err != nil || selected.Target().Path != execution.EntryPath() || selected.Target().Digest != execution.EntryDigest() || uint64(selected.Target().Size) != execution.EntrySize() {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	root := paths[graph.Primary()]
	if root != path.Join("node_modules", execution.EntryArtifact().Identity().Name()) {
		return NodeExecutionLayout{}, errNodeExecutionLayout
	}
	entry := path.Join(root, execution.EntryPath())
	body, err := layout.ReadFile(entry)
	if err != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	if _, err := selected.BindNodeTarget(body); err != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	return bindNodeExecutionLayoutFacts(ctx, execution, layout, entry, false)
}

func bindNodeExecutionLayoutFacts(ctx context.Context, execution domain.ExecutionContext, layout PNPMLayoutArchive, entry string, nativePNPM bool) (NodeExecutionLayout, error) {
	census := sha256.Sum256(layout.body)
	schema := "helox.node-execution-layout/v1"
	if nativePNPM {
		schema = "helox.pnpm-node-execution-layout/v1"
	}
	metadata, err := json.Marshal(struct {
		Schema, Census, Source, Policy, Runtime, Command, Member, MemberSHA string
		Size                                                                uint64
	}{schema, hex.EncodeToString(census[:]), execution.SourceFingerprint().String(), execution.PolicyFingerprint().String(), execution.RuntimeIdentity(), execution.Command(), entry, execution.EntryDigest().String(), execution.EntrySize()})
	if err != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	sum := sha256.Sum256(metadata)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	if ctx.Err() != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, ctx.Err(), context.Cause(ctx))
	}
	return NodeExecutionLayout{binding: executionLayoutFacts(execution), layout: layout, entry: entry, digest: digest, nativePNPM: nativePNPM}, nil
}
