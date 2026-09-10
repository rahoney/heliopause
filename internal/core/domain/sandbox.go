package domain

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	observationSummarySchema                = "m11-004"
	MaximumObservationSummaryUniqueSubjects = 32
	MaximumObservationSummaryCount          = uint64(10000)
)

// SandboxSessionID identifies one ephemeral dynamic-inspection session.
type SandboxSessionID struct{ value string }

// NewSandboxSessionID creates a cryptographically random Sandbox Session identifier.
func NewSandboxSessionID() (SandboxSessionID, error) {
	value, err := generateID("sbx_", rand.Reader)
	if err != nil {
		return SandboxSessionID{}, fmt.Errorf("generate Sandbox Session ID: %w", err)
	}
	return SandboxSessionID{value: value}, nil
}

// ParseSandboxSessionID validates a serialized Sandbox Session identifier.
func ParseSandboxSessionID(value string) (SandboxSessionID, error) {
	if err := validateID(value, "sbx_"); err != nil {
		return SandboxSessionID{}, err
	}
	return SandboxSessionID{value: value}, nil
}

func (id SandboxSessionID) String() string { return id.value }

// SandboxStatus records completion independently from an operational backend error.
type SandboxStatus string

const (
	SandboxCompleted  SandboxStatus = "COMPLETED"
	SandboxIncomplete SandboxStatus = "INCOMPLETE"
)

// ObservationCategory is a bounded runtime fact category, not a security verdict.
type ObservationCategory string

const (
	ObservationProcess    ObservationCategory = "PROCESS"
	ObservationFilesystem ObservationCategory = "FILESYSTEM"
	ObservationNetwork    ObservationCategory = "NETWORK"
	ObservationHoneytoken ObservationCategory = "HONEYTOKEN"
	ObservationResource   ObservationCategory = "RESOURCE"
)

// SandboxObservation is a sanitized raw runtime fact for later Inspection interpretation.
type SandboxObservation struct {
	category ObservationCategory
	subject  string
	count    uint64
}

// NewSandboxObservation constructs a bounded fact without raw runtime output or Host paths.
func NewSandboxObservation(category ObservationCategory, subject string) (SandboxObservation, error) {
	return NewCountedSandboxObservation(category, subject, 1)
}

// NewCountedSandboxObservation constructs one bounded normalized subject with
// its retained saturating event count.
func NewCountedSandboxObservation(category ObservationCategory, subject string, count uint64) (SandboxObservation, error) {
	switch category {
	case ObservationProcess, ObservationFilesystem, ObservationNetwork, ObservationHoneytoken, ObservationResource:
	default:
		return SandboxObservation{}, fmt.Errorf("invalid Sandbox Observation category %q", category)
	}
	if err := validateNormalizedIdentifier(subject, 64, "Sandbox Observation subject"); err != nil {
		return SandboxObservation{}, err
	}
	if count == 0 || count > MaximumObservationSummaryCount {
		return SandboxObservation{}, errors.New("sandbox observation count is invalid")
	}
	return SandboxObservation{category: category, subject: subject, count: count}, nil
}

func (o SandboxObservation) Category() ObservationCategory { return o.category }
func (o SandboxObservation) Subject() string               { return o.subject }
func (o SandboxObservation) Count() uint64                 { return o.count }

// SandboxRequest gives a backend only the exact acquired content subject to execute.
type SandboxRequest struct{ artifact AcquiredArtifact }

// NewSandboxRequest constructs a dynamic-inspection request for exact controlled content.
func NewSandboxRequest(artifact AcquiredArtifact) (SandboxRequest, error) {
	if artifact.Identity().Source().String() == "" || artifact.Digest().String() == "" || artifact.ContentHandle() == "" {
		return SandboxRequest{}, errors.New("sandbox request requires an acquired Artifact")
	}
	return SandboxRequest{artifact: artifact}, nil
}

func (r SandboxRequest) Artifact() AcquiredArtifact { return r.artifact }

// SandboxResult captures one disposed Session's bounded raw observations.
type SandboxResult struct {
	sessionID  SandboxSessionID
	status     SandboxStatus
	limitation string
	observed   []SandboxObservation
}

// NewSandboxResult constructs an immutable result from one ephemeral Sandbox Session.
func NewSandboxResult(sessionID SandboxSessionID, status SandboxStatus, limitationCode string, observations []SandboxObservation) (SandboxResult, error) {
	if sessionID.value == "" {
		return SandboxResult{}, errors.New("sandbox result requires a Session ID")
	}
	switch status {
	case SandboxCompleted:
		if limitationCode != "" {
			return SandboxResult{}, errors.New("completed Sandbox result cannot have a limitation")
		}
	case SandboxIncomplete:
		if !failureCodePattern.MatchString(limitationCode) {
			return SandboxResult{}, errors.New("incomplete Sandbox result requires an uppercase limitation code")
		}
	default:
		return SandboxResult{}, fmt.Errorf("invalid sandbox status %q", status)
	}
	owned := make([]SandboxObservation, 0, len(observations))
	indices := make(map[string]int)
	for index, observation := range observations {
		if observation.subject == "" {
			return SandboxResult{}, fmt.Errorf("sandbox observation %d is invalid", index)
		}
		if observation.count == 0 || observation.count > MaximumObservationSummaryCount {
			return SandboxResult{}, fmt.Errorf("sandbox observation %d has an invalid count", index)
		}
		key := string(observation.category) + ":" + observation.subject
		if existing, ok := indices[key]; ok {
			count := owned[existing].count
			if MaximumObservationSummaryCount-count < observation.count {
				count = MaximumObservationSummaryCount
			} else {
				count += observation.count
			}
			owned[existing].count = count
			continue
		}
		indices[key] = len(owned)
		owned = append(owned, observation)
	}
	return SandboxResult{sessionID: sessionID, status: status, limitation: limitationCode, observed: owned}, nil
}

func (r SandboxResult) SessionID() SandboxSessionID    { return r.sessionID }
func (r SandboxResult) Status() SandboxStatus          { return r.status }
func (r SandboxResult) LimitationCode() (string, bool) { return r.limitation, r.limitation != "" }
func (r SandboxResult) Observations() []SandboxObservation {
	return append([]SandboxObservation(nil), r.observed...)
}

// ObservationSummary returns the bounded, normalized representation retained by
// Evidence. It never includes raw runtime payloads, paths, arguments or IDs.
func (r SandboxResult) ObservationSummary() (string, error) {
	if r.sessionID.value == "" {
		return "", errors.New("observation summary requires a Sandbox Session")
	}
	if r.status != SandboxCompleted && r.status != SandboxIncomplete {
		return "", errors.New("observation summary status is invalid")
	}
	counts := make(map[string]uint64)
	var total uint64
	for _, observation := range r.observed {
		key := string(observation.category) + ":" + observation.subject
		if _, exists := counts[key]; !exists {
			if len(counts) >= MaximumObservationSummaryUniqueSubjects {
				return "", errors.New("observation summary unique-subject bound exceeded")
			}
			counts[key] = 0
		}
		if counts[key] < MaximumObservationSummaryCount {
			remaining := MaximumObservationSummaryCount - counts[key]
			if observation.count > remaining {
				counts[key] = MaximumObservationSummaryCount
			} else {
				counts[key] += observation.count
			}
		}
		if total < MaximumObservationSummaryCount {
			remaining := MaximumObservationSummaryCount - total
			if observation.count > remaining {
				total = MaximumObservationSummaryCount
			} else {
				total += observation.count
			}
		}
	}
	value := struct {
		Schema         string            `json:"schema"`
		Session        string            `json:"session"`
		Status         SandboxStatus     `json:"status"`
		Limitation     string            `json:"limitation,omitempty"`
		UniqueSubjects int               `json:"unique_subjects"`
		Total          uint64            `json:"total"`
		Counts         map[string]uint64 `json:"counts"`
	}{
		Schema:         observationSummarySchema,
		Session:        r.sessionID.String(),
		Status:         r.status,
		Limitation:     r.limitation,
		UniqueSubjects: len(counts),
		Total:          total,
		Counts:         counts,
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode observation summary: %w", err)
	}
	if len(encoded) > maxEvidenceSummaryLength {
		return "", errors.New("observation summary byte bound exceeded")
	}
	return string(encoded), nil
}
