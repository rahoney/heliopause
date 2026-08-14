package domain

import "errors"

const maximumVerifiedDocumentBytes = 16 << 20

// VerifiedSet binds one complete inspected graph to its set-level ALLOW.
// It is the only set eligible for Manifest generation and Staging.
type VerifiedSet struct {
	inspected InspectedDependencySet
	decision  PolicyDecision
}

func NewVerifiedSet(inspected InspectedDependencySet, decision PolicyDecision) (VerifiedSet, error) {
	if !inspected.Valid() || decision.Decision() != DecisionAllow {
		return VerifiedSet{}, errors.New("verified set requires a complete inspected set and set-level ALLOW")
	}
	for _, inspection := range inspected.Inspections() {
		if inspection.PolicyDecision().Decision() != DecisionAllow {
			return VerifiedSet{}, errors.New("verified set contains an entry without ALLOW")
		}
	}
	return VerifiedSet{inspected: inspected, decision: decision}, nil
}

func (s VerifiedSet) Valid() bool {
	return s.inspected.Valid() && s.decision.Decision() == DecisionAllow
}
func (s VerifiedSet) Inspected() InspectedDependencySet { return s.inspected }
func (s VerifiedSet) Decision() PolicyDecision          { return s.decision }

// VerifiedBundle keeps the canonical Manifest and matching SBOM together.
// Documents are copied at construction and access to prevent mutation after
// the controller has computed the Manifest identity.
type VerifiedBundle struct {
	manifestID       ContentDigest
	set              VerifiedSet
	manifestDocument []byte
	sbomDocument     []byte
}

func NewVerifiedBundle(manifestID ContentDigest, set VerifiedSet, manifestDocument, sbomDocument []byte) (VerifiedBundle, error) {
	if manifestID.String() == "" || !set.Valid() {
		return VerifiedBundle{}, errors.New("verified bundle requires a Manifest identity and verified set")
	}
	if len(manifestDocument) == 0 || len(manifestDocument) > maximumVerifiedDocumentBytes || len(sbomDocument) == 0 || len(sbomDocument) > maximumVerifiedDocumentBytes {
		return VerifiedBundle{}, errors.New("verified bundle documents exceed bounds")
	}
	return VerifiedBundle{
		manifestID:       manifestID,
		set:              set,
		manifestDocument: append([]byte(nil), manifestDocument...),
		sbomDocument:     append([]byte(nil), sbomDocument...),
	}, nil
}

func (b VerifiedBundle) Valid() bool               { return b.manifestID.String() != "" && b.set.Valid() }
func (b VerifiedBundle) ManifestID() ContentDigest { return b.manifestID }
func (b VerifiedBundle) Set() VerifiedSet          { return b.set }
func (b VerifiedBundle) ManifestDocument() []byte  { return append([]byte(nil), b.manifestDocument...) }
func (b VerifiedBundle) SBOMDocument() []byte      { return append([]byte(nil), b.sbomDocument...) }

// StagedSet is an opaque trusted reference to one fully finalized staging
// directory. It does not grant the caller filesystem access.
type StagedSet struct {
	manifestID ContentDigest
	handle     string
}

func NewStagedSet(manifestID ContentDigest, handle string) (StagedSet, error) {
	if manifestID.String() == "" {
		return StagedSet{}, errors.New("staged set requires a Manifest identity")
	}
	if err := validateBoundedText(handle, maxContentHandleLength, "staging handle"); err != nil {
		return StagedSet{}, err
	}
	return StagedSet{manifestID: manifestID, handle: handle}, nil
}

func (s StagedSet) ManifestID() ContentDigest { return s.manifestID }
func (s StagedSet) Handle() string            { return s.handle }
