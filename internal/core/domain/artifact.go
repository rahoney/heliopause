package domain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxSourceIDLength      = 64
	maxLocatorLength       = 1024
	maxCoordinateLength    = 256
	maxContentHandleLength = 512
)

var normalizedIdentifier = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9._-]*[a-z0-9])?$`)

// SourceID identifies a normalized Artifact source. Its zero value is invalid.
type SourceID struct{ value string }

// NewSourceID validates an ecosystem-neutral, lowercase source identifier.
func NewSourceID(value string) (SourceID, error) {
	if err := validateNormalizedIdentifier(value, maxSourceIDLength, "source ID"); err != nil {
		return SourceID{}, err
	}
	return SourceID{value: value}, nil
}

func (id SourceID) String() string { return id.value }

// ArtifactReference preserves the bounded source locator requested by a caller.
type ArtifactReference struct {
	source  SourceID
	locator string
}

// NewArtifactReference constructs a validated Artifact Reference.
func NewArtifactReference(source SourceID, locator string) (ArtifactReference, error) {
	if source.value == "" {
		return ArtifactReference{}, errors.New("source ID is required")
	}
	if err := validateBoundedText(locator, maxLocatorLength, "artifact locator"); err != nil {
		return ArtifactReference{}, err
	}
	return ArtifactReference{source: source, locator: locator}, nil
}

func (r ArtifactReference) Source() SourceID { return r.source }
func (r ArtifactReference) Locator() string  { return r.locator }

// ResolvedArtifactIdentity is an exact logical Artifact identity.
type ResolvedArtifactIdentity struct {
	source  SourceID
	name    string
	version string
	variant string
}

// NewResolvedArtifactIdentity constructs an exact identity with no wildcard fields.
func NewResolvedArtifactIdentity(source SourceID, name, version, variant string) (ResolvedArtifactIdentity, error) {
	if source.value == "" {
		return ResolvedArtifactIdentity{}, errors.New("source ID is required")
	}
	for label, value := range map[string]string{"artifact name": name, "artifact version": version, "artifact variant": variant} {
		if err := validateBoundedText(value, maxCoordinateLength, label); err != nil {
			return ResolvedArtifactIdentity{}, err
		}
	}
	return ResolvedArtifactIdentity{source: source, name: name, version: version, variant: variant}, nil
}

func (i ResolvedArtifactIdentity) Source() SourceID { return i.source }
func (i ResolvedArtifactIdentity) Name() string     { return i.name }
func (i ResolvedArtifactIdentity) Version() string  { return i.version }
func (i ResolvedArtifactIdentity) Variant() string  { return i.variant }

// ContentDigest is the observed SHA-256 digest of acquired bytes.
type ContentDigest struct{ value string }

// NewSHA256Digest validates a lowercase, canonical SHA-256 hexadecimal digest.
func NewSHA256Digest(value string) (ContentDigest, error) {
	if len(value) != 64 || value != strings.ToLower(value) {
		return ContentDigest{}, errors.New("SHA-256 digest must be 64 lowercase hexadecimal characters")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return ContentDigest{}, errors.New("SHA-256 digest must be 64 lowercase hexadecimal characters")
	}
	return ContentDigest{value: value}, nil
}

func (d ContentDigest) Algorithm() string { return "sha256" }
func (d ContentDigest) String() string    { return d.value }

// AcquiredArtifact binds an exact identity to observed content in controlled intake.
type AcquiredArtifact struct {
	identity          ResolvedArtifactIdentity
	digest            ContentDigest
	handle            string
	size              uint64
	declaredIntegrity string
	observedIntegrity string
}

// NewAcquiredArtifact constructs a validated acquired content subject.
func NewAcquiredArtifact(identity ResolvedArtifactIdentity, digest ContentDigest, handle string, sizeBytes uint64) (AcquiredArtifact, error) {
	return NewAcquiredArtifactWithIntegrity(identity, digest, handle, sizeBytes, "", "")
}

// NewAcquiredArtifactWithDeclaredIntegrity constructs an acquired subject and bounded source integrity input.
func NewAcquiredArtifactWithDeclaredIntegrity(identity ResolvedArtifactIdentity, digest ContentDigest, handle string, sizeBytes uint64, declaredIntegrity string) (AcquiredArtifact, error) {
	return NewAcquiredArtifactWithIntegrity(identity, digest, handle, sizeBytes, declaredIntegrity, "")
}

// NewAcquiredArtifactWithIntegrity binds declared and observed source-integrity values without replacing content identity.
func NewAcquiredArtifactWithIntegrity(identity ResolvedArtifactIdentity, digest ContentDigest, handle string, sizeBytes uint64, declaredIntegrity, observedIntegrity string) (AcquiredArtifact, error) {
	if identity.source.value == "" {
		return AcquiredArtifact{}, errors.New("resolved artifact identity is required")
	}
	if digest.value == "" {
		return AcquiredArtifact{}, errors.New("content digest is required")
	}
	if err := validateBoundedText(handle, maxContentHandleLength, "content handle"); err != nil {
		return AcquiredArtifact{}, err
	}
	if declaredIntegrity != "" {
		if err := validateBoundedText(declaredIntegrity, maxDeclaredIntegrityLength, "declared integrity"); err != nil {
			return AcquiredArtifact{}, err
		}
	}
	if observedIntegrity != "" {
		if err := validateBoundedText(observedIntegrity, maxDeclaredIntegrityLength, "observed integrity"); err != nil {
			return AcquiredArtifact{}, err
		}
	}
	return AcquiredArtifact{identity: identity, digest: digest, handle: handle, size: sizeBytes, declaredIntegrity: declaredIntegrity, observedIntegrity: observedIntegrity}, nil
}

func (a AcquiredArtifact) Identity() ResolvedArtifactIdentity { return a.identity }
func (a AcquiredArtifact) Digest() ContentDigest              { return a.digest }
func (a AcquiredArtifact) ContentHandle() string              { return a.handle }
func (a AcquiredArtifact) SizeBytes() uint64                  { return a.size }
func (a AcquiredArtifact) DeclaredIntegrity() (string, bool) {
	return a.declaredIntegrity, a.declaredIntegrity != ""
}
func (a AcquiredArtifact) ObservedIntegrity() (string, bool) {
	return a.observedIntegrity, a.observedIntegrity != ""
}

func validateNormalizedIdentifier(value string, maximum int, label string) error {
	if len(value) == 0 || len(value) > maximum || !utf8.ValidString(value) || !normalizedIdentifier.MatchString(value) {
		return fmt.Errorf("%s must be a normalized lowercase identifier of at most %d bytes", label, maximum)
	}
	return nil
}

func validateBoundedText(value string, maximum int, label string) error {
	if value == "" || value != strings.TrimSpace(value) || len(value) > maximum || !utf8.ValidString(value) || strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return fmt.Errorf("%s must be non-empty bounded text of at most %d bytes", label, maximum)
	}
	return nil
}
