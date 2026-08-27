//go:build !linux

package sandbox

import "testing"

func newIntegrationResolverPolicyService(t *testing.T) ResolverPolicyService {
	t.Helper()
	return &recordingResolverPolicyService{}
}
