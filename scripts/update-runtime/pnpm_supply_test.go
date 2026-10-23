package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

func TestPNPMNoopDoesNotAcquireSupply(t *testing.T) {
	root, state := fixture(t)
	version, err := selectedVersion(root, "pnpm")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	p, err := makePlanResolving(root, state, "pnpm", version, "", func(component, version string) (metadata, error) {
		called = true
		return metadata{}, errors.New("must not acquire during no-op")
	})
	if err != nil || called || len(p.Changes) != 0 || p.Metadata.PNPM != nil {
		t.Fatalf("no-op altered or acquired supply: %+v %v", p, err)
	}
	if err := applyPlan(p, state, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(p, state, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPNPMSupplyFlagsAreScopedAndExplicit(t *testing.T) {
	for _, name := range []string{"same-version-proof", "foreign-component", "implicit-network"} {
		t.Run(name, func(t *testing.T) {
			root, state := fixture(t)
			version, err := selectedVersion(root, "pnpm")
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"plan", "--root", root, "--state", state, "--component", "pnpm", "--version", version, "--pnpm-proof", filepath.Join(state, "unavailable")}
			switch name {
			case "foreign-component":
				args[6] = "node"
			case "implicit-network":
				args = []string{"plan", "--root", root, "--state", state, "--component", "pnpm", "--version", "999.0.0"}
			}
			var output bytes.Buffer
			if err := run(args, &output); err == nil || output.Len() != 0 {
				t.Fatalf("unsupported supply produced a plan: %v %s", err, output.String())
			}
		})
	}
}
