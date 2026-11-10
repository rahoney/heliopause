package bootstrap

import (
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/inspection/advisory"
	"github.com/rahoney/heliopause/internal/networkpolicy"
)

// The composition root supplies the same public-address rule used by the
// resolver boundary, without granting its privileged service to Inspection.
func newAdvisoryInspection(inner ports.Inspection) (*advisory.Inspector, error) {
	client, err := advisory.NewOSVHTTPClient(networkpolicy.IsPublicIPv4)
	if err != nil {
		return nil, err
	}
	inspector, err := advisory.NewInspector(inner, client)
	if err != nil {
		client.CloseIdleConnections()
		return nil, err
	}
	return inspector, nil
}
