package domain

import "errors"

const (
	maxAcquisitionLocatorLength = 2048
	maxDeclaredIntegrityLength  = 1024
)

// ResolvedArtifact fixes the exact identity and source-supplied acquisition inputs for one Run.
// Its locator and declared integrity are inputs to later verification, never a safety decision.
type ResolvedArtifact struct {
	identity           ResolvedArtifactIdentity
	acquisitionLocator string
	declaredIntegrity  string
}

// NewResolvedArtifact constructs a bounded source-neutral acquisition target.
func NewResolvedArtifact(identity ResolvedArtifactIdentity, acquisitionLocator, declaredIntegrity string) (ResolvedArtifact, error) {
	if identity.source.value == "" {
		return ResolvedArtifact{}, errors.New("resolved artifact identity is required")
	}
	if err := validateBoundedText(acquisitionLocator, maxAcquisitionLocatorLength, "acquisition locator"); err != nil {
		return ResolvedArtifact{}, err
	}
	if declaredIntegrity != "" {
		if err := validateBoundedText(declaredIntegrity, maxDeclaredIntegrityLength, "declared integrity"); err != nil {
			return ResolvedArtifact{}, err
		}
	}
	return ResolvedArtifact{identity: identity, acquisitionLocator: acquisitionLocator, declaredIntegrity: declaredIntegrity}, nil
}

func (a ResolvedArtifact) Identity() ResolvedArtifactIdentity { return a.identity }
func (a ResolvedArtifact) AcquisitionLocator() string         { return a.acquisitionLocator }
func (a ResolvedArtifact) DeclaredIntegrity() string          { return a.declaredIntegrity }
