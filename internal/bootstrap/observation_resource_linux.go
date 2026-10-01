//go:build linux

package bootstrap

import (
	"context"

	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// The composition root converts the authenticated Host resource response into
// the sandbox port. The sandbox never depends on the privileged helper package.
type observationResourceAdapter struct {
	client *hosttool.ObservationResourceClient
}

func newObservationResourceAdapter() sandbox.ObservationResourceClient {
	client, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		return nil
	}
	return observationResourceAdapter{client: client}
}

func observationLease(lease hosttool.ObservationResourceLease) sandbox.ObservationResourceLease {
	return sandbox.ObservationResourceLease{CgroupParent: lease.CgroupParent, CPU: lease.CPU, UsageUsec: lease.UsageUsec}
}

func (a observationResourceAdapter) Create(ctx context.Context, transaction, profile string) (sandbox.ObservationResourceLease, error) {
	lease, err := a.client.Create(ctx, transaction, profile)
	return observationLease(lease), err
}

func (a observationResourceAdapter) Register(ctx context.Context, transaction, container, role string) (sandbox.ObservationResourceLease, error) {
	lease, err := a.client.Register(ctx, transaction, container, role)
	return observationLease(lease), err
}

func (a observationResourceAdapter) Read(ctx context.Context, transaction string) (sandbox.ObservationResourceLease, error) {
	lease, err := a.client.Read(ctx, transaction)
	return observationLease(lease), err
}

func (a observationResourceAdapter) Terminate(ctx context.Context, transaction, container string) (sandbox.ObservationResourceLease, error) {
	lease, err := a.client.Terminate(ctx, transaction, container)
	return observationLease(lease), err
}

func (a observationResourceAdapter) Close(ctx context.Context, transaction string) (sandbox.ObservationResourceLease, error) {
	lease, err := a.client.Close(ctx, transaction)
	return observationLease(lease), err
}
