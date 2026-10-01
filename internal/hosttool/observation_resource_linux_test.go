//go:build linux

package hosttool

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type fixedObservationKeeper struct{}

func (fixedObservationKeeper) Start(_ context.Context, parent string) (string, string, uint64, error) {
	if !validObservationParent(parent) {
		return "", "", 0, errors.New("invalid parent")
	}
	path, scope := observationKeeperPaths(parent)
	return path, scope, 0, nil
}
func (fixedObservationKeeper) Verify(context.Context, string, string) error          { return nil }
func (fixedObservationKeeper) Close(context.Context, string, string) (uint64, error) { return 0, nil }

func TestObservationResourceHelperDerivesParentAndPolicy(t *testing.T) {
	id, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	engine := &observationResourceEngine{executor: &Executor{}, keeper: fixedObservationKeeper{}}
	lease, err := engine.apply(context.Background(), 1000, observationResourceRequest{
		Operation: "create", Transaction: id.String(), Profile: "pytorch:cu130",
	})
	if err != nil {
		t.Fatal(err)
	}
	if lease.CgroupParent == "" || lease.CPU < 0 {
		t.Fatalf("lease = %#v", lease)
	}
	if got := engine.sessions[id.String()].ceiling; got != 300_000_000 {
		t.Fatalf("derived CPU ceiling = %d", got)
	}
	if _, err := engine.apply(context.Background(), 1000, observationResourceRequest{
		Operation: "create", Transaction: id.String(), Profile: "pytorch:cu130",
	}); err == nil {
		t.Fatal("duplicate observation parent accepted")
	}
	other, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.apply(context.Background(), 1000, observationResourceRequest{
		Operation: "create", Transaction: other.String(), Profile: "pytorch:cu128",
	}); err == nil {
		t.Fatal("unsupported profile accepted")
	}
}

func TestObservationResourceHelperRejectsCallerChosenPathAndContainer(t *testing.T) {
	id, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	engine := &observationResourceEngine{executor: &Executor{}}
	if _, err := engine.apply(context.Background(), 1000, observationResourceRequest{
		Operation: "create", Transaction: id.String(), Profile: "pypi", Container: "/sys/fs/cgroup/foreign",
	}); err == nil {
		t.Fatal("caller-chosen cgroup path accepted")
	}
	if validDockerContainerID("abcdef0123456789") || validDockerContainerID("/proc/1") {
		t.Fatal("caller-chosen or abbreviated container identity accepted")
	}
}
