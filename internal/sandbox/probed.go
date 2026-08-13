package sandbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// ProbedSandbox makes the M3 runtime capability result explicit until a Linux
// runtime backend is supplied by the composition root.
type ProbedSandbox struct {
	probe        CapabilityProbe
	newSessionID func() (domain.SandboxSessionID, error)
}

// NewProbedSandbox constructs a fail-closed Sandbox backed by a capability probe.
func NewProbedSandbox(probe CapabilityProbe) (*ProbedSandbox, error) {
	if probe == nil {
		return nil, errors.New("sandbox capability probe is required")
	}
	return &ProbedSandbox{probe: probe, newSessionID: domain.NewSandboxSessionID}, nil
}

// Execute never bypasses a failed or unavailable runtime capability.
func (s *ProbedSandbox) Execute(ctx context.Context, _ domain.SandboxRequest) (domain.SandboxResult, error) {
	if s == nil || s.probe == nil || s.newSessionID == nil {
		return domain.SandboxResult{}, errors.New("probed sandbox is not configured")
	}
	if ctx == nil {
		return domain.SandboxResult{}, errors.New("context is required")
	}
	sessionID, err := s.newSessionID()
	if err != nil {
		return domain.SandboxResult{}, fmt.Errorf("create Sandbox Session ID: %w", err)
	}
	capability, err := s.probe(ctx)
	if err != nil {
		return incomplete(sessionID, "M3_DYNAMIC_CAPABILITY_ERROR")
	}
	if !capability.Available {
		return incomplete(sessionID, capability.LimitationCode)
	}
	return incomplete(sessionID, "M3_DYNAMIC_RUNTIME_UNWIRED")
}
