package domain

import (
	"errors"
	"fmt"
	"strings"
)

const maxEvidenceSummaryLength = 1024

// Host install action facts describe a possible installation requirement;
// they do not authorize execution or replace an inspection's Findings.
const (
	EvidenceHostInstallActionRequired    = "host-install-action-required"
	EvidenceHostInstallActionNotRequired = "host-install-action-not-required"
)

// EvidenceID identifies one normalized Evidence fact. Its zero value is invalid.
type EvidenceID struct{ value string }

// NewEvidenceID validates a normalized Evidence identifier.
func NewEvidenceID(value string) (EvidenceID, error) {
	if err := validateNormalizedIdentifier(value, 64, "evidence ID"); err != nil {
		return EvidenceID{}, err
	}
	return EvidenceID{value: value}, nil
}

func (id EvidenceID) String() string { return id.value }

// Evidence is a bounded normalized fact tied to an exact content subject.
type Evidence struct {
	id       EvidenceID
	checkID  CheckID
	identity ResolvedArtifactIdentity
	digest   ContentDigest
	kind     string
	summary  string
	profile  ContentDigest
}

// NewEvidence constructs sanitized normalized Evidence without raw provider output.
func NewEvidence(id EvidenceID, checkID CheckID, identity ResolvedArtifactIdentity, digest ContentDigest, kind, summary string, profiles ...ContentDigest) (Evidence, error) {
	if id.value == "" || checkID.value == "" || identity.source.value == "" || digest.value == "" {
		return Evidence{}, errors.New("evidence identity, check, artifact identity, and digest are required")
	}
	if err := validateNormalizedIdentifier(kind, 64, "evidence kind"); err != nil {
		return Evidence{}, err
	}
	if err := validateBoundedText(summary, maxEvidenceSummaryLength, "evidence summary"); err != nil {
		return Evidence{}, err
	}
	if containsSensitiveEvidence(summary) {
		return Evidence{}, errors.New("evidence summary contains a credential or identifying Host path pattern")
	}
	var profile ContentDigest
	if len(profiles) > 1 || len(profiles) == 1 && profiles[0].value == "" {
		return Evidence{}, errors.New("evidence profile is invalid")
	}
	if len(profiles) == 1 {
		profile = profiles[0]
	}
	return Evidence{id: id, checkID: checkID, identity: identity, digest: digest, kind: kind, summary: summary, profile: profile}, nil
}

func (e Evidence) ProfileDigest() ContentDigest { return e.profile }

func (e Evidence) ID() EvidenceID                     { return e.id }
func (e Evidence) CheckID() CheckID                   { return e.checkID }
func (e Evidence) Identity() ResolvedArtifactIdentity { return e.identity }
func (e Evidence) Digest() ContentDigest              { return e.digest }
func (e Evidence) Kind() string                       { return e.kind }
func (e Evidence) Summary() string                    { return e.summary }

// Finding is a normalized security interpretation linked to supporting Evidence.
type Finding struct {
	code       string
	evidenceID []EvidenceID
}

// NewFinding constructs a Finding with one or more supporting Evidence identifiers.
func NewFinding(code string, evidenceIDs []EvidenceID) (Finding, error) {
	if !failureCodePattern.MatchString(code) {
		return Finding{}, errors.New("finding code must be 1 to 64 uppercase identifier characters")
	}
	if len(evidenceIDs) == 0 {
		return Finding{}, errors.New("finding requires supporting Evidence")
	}
	owned := make([]EvidenceID, len(evidenceIDs))
	for index, id := range evidenceIDs {
		if id.value == "" {
			return Finding{}, fmt.Errorf("finding evidence ID %d is invalid", index)
		}
		owned[index] = id
	}
	return Finding{code: code, evidenceID: owned}, nil
}

func (f Finding) Code() string              { return f.code }
func (f Finding) EvidenceIDs() []EvidenceID { return append([]EvidenceID(nil), f.evidenceID...) }

// EvidenceReference is a trusted store reference, not raw Evidence content.
type EvidenceReference struct {
	id     EvidenceID
	handle string
}

// NewEvidenceReference constructs a bounded trusted Evidence handle.
func NewEvidenceReference(id EvidenceID, handle string) (EvidenceReference, error) {
	if id.value == "" {
		return EvidenceReference{}, errors.New("evidence ID is required")
	}
	if err := validateBoundedText(handle, 512, "evidence handle"); err != nil {
		return EvidenceReference{}, err
	}
	if containsSensitiveEvidence(handle) {
		return EvidenceReference{}, errors.New("evidence handle contains an identifying Host path pattern")
	}
	return EvidenceReference{id: id, handle: handle}, nil
}

func (r EvidenceReference) ID() EvidenceID { return r.id }
func (r EvidenceReference) Handle() string { return r.handle }

func containsSensitiveEvidence(summary string) bool {
	lower := strings.ToLower(summary)
	for _, pattern := range []string{
		"/users/", "/home/", `c:\users\`,
		"authorization:", "password=", "token=", "api_key=", "apikey=",
		"-----begin private " + "key-----", "-----begin openssh private " + "key-----",
	} {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// BindEvidenceProfiles projects actual check metadata into fresh normalized
// facts. Existing non-empty bindings must agree; they cannot be relabelled.
func BindEvidenceProfiles(checks []CheckExecution, evidence []Evidence) ([]Evidence, error) {
	result := append([]Evidence(nil), evidence...)
	if len(checks) == 0 {
		return result, nil
	}
	byCheck := make(map[CheckID]CheckExecution, len(checks))
	for _, check := range checks {
		if _, duplicate := byCheck[check.id]; duplicate {
			return nil, errors.New("evidence checks contain duplicate identifiers")
		}
		byCheck[check.id] = check
	}
	for index, item := range result {
		check, ok := byCheck[item.checkID]
		if !ok {
			return nil, errors.New("evidence refers to an undeclared actual check")
		}
		if item.profile.value != "" && item.profile != check.profile {
			return nil, errors.New("evidence is already bound to another check profile")
		}
		if check.profile.value != "" {
			item.profile = check.profile
			result[index] = item
		}
	}
	return result, nil
}
