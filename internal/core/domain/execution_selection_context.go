package domain

import (
	"errors"
	"reflect"
	"slices"
)

// ExecutionSelectionContextData binds the complete approved root selection to
// its measured primary console. Arguments remain private request data.
type ExecutionSelectionContextData struct {
	OperationID     OperationID
	Resolution      ExecutionSelectionResolution
	Set             VerifiedExecutionSelection
	EntryArtifact   AcquiredArtifact
	Command         string
	EntryPath       string
	EntryDigest     ContentDigest
	EntrySize       uint64
	RuntimeIdentity string
	Arguments       []string
}

// ExecutionSelectionContext preserves the separate roots through execution.
// It supplies neither an install target nor runtime admission authority.
type ExecutionSelectionContext struct {
	data   ExecutionSelectionContextData
	policy ContentDigest
}

func NewExecutionSelectionContext(data ExecutionSelectionContextData) (ExecutionSelectionContext, error) {
	invalid := errors.New("execution context requires matching complete approved roots and a measured primary console")
	if data.OperationID.value == "" || !data.Resolution.Valid() || !data.Set.Valid() ||
		!reflect.DeepEqual(data.Resolution.Selection(), data.Set.Inspected().Selection()) ||
		validateBoundedText(data.RuntimeIdentity, maxInstallTargetLength, "execution runtime identity") != nil ||
		ValidateExecutionCommand(data.Command) != nil || data.EntryDigest.value == "" || !validExecutionMemberPath(data.EntryPath) {
		return ExecutionSelectionContext{}, invalid
	}
	matched := false
	for _, inspection := range data.Set.Inspected().Inspections() {
		if inspection.Node() == data.Resolution.Selection().Primary() && inspection.Artifact() == data.EntryArtifact {
			matched = true
		}
	}
	if !matched {
		return ExecutionSelectionContext{}, invalid
	}
	if err := ValidateExecutionArguments(data.Arguments); err != nil {
		return ExecutionSelectionContext{}, err
	}
	policy, err := currentExecutionPolicyFingerprint(data.Set.Decision(), data.Set.Inspected().Inspections())
	if err != nil {
		return ExecutionSelectionContext{}, err
	}
	data.Arguments = slices.Clone(data.Arguments)
	return ExecutionSelectionContext{data, policy}, nil
}

func (c ExecutionSelectionContext) Valid() bool              { return c.data.Set.Valid() && c.policy.value != "" }
func (c ExecutionSelectionContext) OperationID() OperationID { return c.data.OperationID }
func (c ExecutionSelectionContext) Resolution() ExecutionSelectionResolution {
	return c.data.Resolution
}
func (c ExecutionSelectionContext) Set() VerifiedExecutionSelection { return c.data.Set }
func (c ExecutionSelectionContext) EntryArtifact() AcquiredArtifact { return c.data.EntryArtifact }
func (c ExecutionSelectionContext) Command() string                 { return c.data.Command }
func (c ExecutionSelectionContext) EntryPath() string               { return c.data.EntryPath }
func (c ExecutionSelectionContext) EntryDigest() ContentDigest      { return c.data.EntryDigest }
func (c ExecutionSelectionContext) EntrySize() uint64               { return c.data.EntrySize }
func (c ExecutionSelectionContext) RuntimeIdentity() string         { return c.data.RuntimeIdentity }
func (c ExecutionSelectionContext) Arguments() []string             { return slices.Clone(c.data.Arguments) }
func (c ExecutionSelectionContext) SourceFingerprint() ContentDigest {
	return c.data.Resolution.LockfileDigest()
}
func (c ExecutionSelectionContext) PolicyFingerprint() ContentDigest { return c.policy }

// ExecutionSelectionBundle keeps whole-selection storage records together.
// It is never an installation bundle or authority to run a program.
type ExecutionSelectionBundle struct {
	id             ContentDigest
	set            VerifiedExecutionSelection
	manifest, sbom []byte
}

func NewExecutionSelectionBundle(id ContentDigest, set VerifiedExecutionSelection, manifest, sbom []byte) (ExecutionSelectionBundle, error) {
	if id.value == "" || !set.Valid() || len(manifest) == 0 || len(manifest) > maximumVerifiedDocumentBytes || len(sbom) == 0 || len(sbom) > maximumVerifiedDocumentBytes {
		return ExecutionSelectionBundle{}, errors.New("execution selection bundle requires whole approval and bounded records")
	}
	return ExecutionSelectionBundle{id, set, slices.Clone(manifest), slices.Clone(sbom)}, nil
}
func (b ExecutionSelectionBundle) Valid() bool                     { return b.id.value != "" && b.set.Valid() }
func (b ExecutionSelectionBundle) ManifestID() ContentDigest       { return b.id }
func (b ExecutionSelectionBundle) Set() VerifiedExecutionSelection { return b.set }
func (b ExecutionSelectionBundle) ManifestDocument() []byte        { return slices.Clone(b.manifest) }
func (b ExecutionSelectionBundle) SBOMDocument() []byte            { return slices.Clone(b.sbom) }
