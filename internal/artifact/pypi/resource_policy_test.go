package pypi

import (
	"context"
	"testing"
	"time"
)

func TestNamedPyTorchProfilesHaveBoundedRootResourcePolicies(t *testing.T) {
	defaultPolicy := PublicPyPIProfile().ResourcePolicy()
	cpu, ok := PyTorchProfile("cpu")
	if !ok {
		t.Fatal("CPU profile is missing")
	}
	cu126, ok := PyTorchProfile("cu126")
	if !ok {
		t.Fatal("cu126 profile is missing")
	}
	if defaultPolicy.MaxArtifactCompressed() != 64<<20 || defaultPolicy.WheelLimits().MaxUncompressed != 256<<20 {
		t.Fatalf("default PyPI resource policy = %#v", defaultPolicy)
	}
	if cpu.ResourcePolicy().MaxArtifactCompressed() != 256<<20 || cpu.ResourcePolicy().MaxGraphCompressed() != 512<<20 || cpu.ResourcePolicy().Duration() != 15*time.Minute {
		t.Fatalf("CPU resource policy = %#v", cpu.ResourcePolicy())
	}
	if cu126.ResourcePolicy().MaxArtifactCompressed() != 1<<30 || cu126.ResourcePolicy().MaxGraphCompressed() != (9<<30)/2 || cu126.ResourcePolicy().Duration() != 40*time.Minute {
		t.Fatalf("cu126 resource policy = %#v", cu126.ResourcePolicy())
	}
}

func TestRootResourceSessionKeepsDefaultAndPyTorchBudgetsSeparate(t *testing.T) {
	cpu, _ := PyTorchProfile("cpu")
	ctx, err := ContextWithResourcePolicy(context.Background(), cpu)
	if err != nil {
		t.Fatal(err)
	}
	policy, session := resourcePolicyFromContext(ctx)
	if policy.MaxArtifactCompressed() != 256<<20 || session.beginArtifact(128<<20) != nil || session.charge(128<<20) != nil {
		t.Fatal("CPU profile rejected an in-budget artifact")
	}
	if session.beginArtifact(400<<20) == nil {
		t.Fatal("CPU profile accepted an over-budget artifact")
	}
	defaultPolicy, defaultSession := resourcePolicyFromContext(context.Background())
	if defaultPolicy.MaxArtifactCompressed() != 64<<20 || defaultSession.beginArtifact(128<<20) == nil {
		t.Fatal("default PyPI inherited a PyTorch budget")
	}
}

func TestRootSourceProfileNameFromContext(t *testing.T) {
	if got, want := RootSourceProfileNameFromContext(context.TODO()), "pypi"; got != want {
		t.Fatalf("TODO context profile = %q, want %q", got, want)
	}
	if got, want := RootSourceProfileNameFromContext(context.Background()), "pypi"; got != want {
		t.Fatalf("background context profile = %q, want %q", got, want)
	}
	defaultCtx, err := ContextWithResourcePolicy(context.Background(), PublicPyPIProfile())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := RootSourceProfileNameFromContext(defaultCtx), "pypi"; got != want {
		t.Fatalf("explicit default profile = %q, want %q", got, want)
	}
	cpu, ok := PyTorchProfile("cpu")
	if !ok {
		t.Fatal("missing cpu profile")
	}
	cpuCtx, err := ContextWithResourcePolicy(context.Background(), cpu)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := RootSourceProfileNameFromContext(cpuCtx), "pytorch:cpu"; got != want {
		t.Fatalf("cpu profile = %q, want %q", got, want)
	}
	cu126, ok := PyTorchProfile("cu126")
	if !ok {
		t.Fatal("missing cu126 profile")
	}
	cu126Ctx, err := ContextWithResourcePolicy(context.Background(), cu126)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := RootSourceProfileNameFromContext(cu126Ctx), "pytorch:cu126"; got != want {
		t.Fatalf("cu126 profile = %q, want %q", got, want)
	}
}
