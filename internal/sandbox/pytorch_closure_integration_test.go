package sandbox

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// This opt-in resolver evidence target acquires metadata inside the normal
// authenticated resolver. It does not qualify or execute the selected wheels.
func TestLinuxPyTorchFrozenClosuresIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTORCH_FREEZE_INTEGRATION") != "1" {
		t.Skip("requires authenticated Linux resolver")
	}
	output := os.Getenv("HELOX_RESOLUTION_OUTPUT")
	if output == "" {
		t.Fatal("HELOX_RESOLUTION_OUTPUT is required")
	}
	if err := os.MkdirAll(output, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(name, func(t *testing.T) {
			profile, ok := artifactpypi.PyTorchProfile(name)
			if !ok {
				t.Fatal("unregistered profile")
			}
			supervisor := integrationObserverSupervisor(t)
			defer supervisor.Close()
			runner := integrationRunner{t: t}
			resolver, err := NewPyTorchResolver(runner, systemNamedEndpointResolver{}, supervisor.Observer(), integrationPythonCapabilityProbe(runner), integrationResolverPolicyService(t), profile)
			if err != nil {
				t.Fatal(err)
			}
			ref, err := artifactpypi.ParseReferenceForSource("torch@2.14.0+"+name, profile.Source())
			if err != nil {
				t.Fatal(err)
			}
			target, _ := domain.NewInstallTarget(filepath.Join(t.TempDir(), "target"))
			install, _ := domain.NewInstallContext(target)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			ctx, err = artifactpypi.ContextWithResourcePolicy(ctx, profile)
			if err != nil {
				t.Fatal(err)
			}
			resolution, err := resolver.ResolveDependencies(ctx, ref, install)
			if err != nil {
				t.Fatal(err)
			}
			type node struct{ Source, Project, Version, URL, Integrity string }
			var nodes []node
			for _, item := range resolution.Graph().Nodes() {
				a := item.Artifact()
				i := a.Identity()
				nodes = append(nodes, node{i.Source().String(), i.Name(), i.Version(), a.AcquisitionLocator(), a.DeclaredIntegrity()})
			}
			data, err := json.MarshalIndent(nodes, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(output, name+".json"), append(data, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Logf("FROZEN_METADATA_ONLY profile=%s nodes=%d", name, len(nodes))
		})
	}
}
