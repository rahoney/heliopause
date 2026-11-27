package sandbox

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/netip"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type npmUpdateRecordingRunner struct {
	recordingRunner
	inputs [][]byte
}

func TestNPMUpdateResolverCancelledSelectionCreatesNoResources(t *testing.T) {
	manifest := []byte(`{"dependencies":{"primary":"1.0.0"}}`)
	ref, _ := artifactnpm.ProjectRootReference(manifest)
	target, _ := domain.NewInstallTarget("/tmp/npm-cancelled-update")
	install, _ := domain.NewNPMProjectInstallContext(target)
	first, _ := domain.NewProjectControlFile("package.json", manifest, true)
	second, _ := domain.NewProjectControlFile("package-lock.json", []byte(resolverLockJSON()), true)
	runner := &npmUpdateRecordingRunner{}
	resolver, err := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, &sequencedObserver{}, &recordingResolverPolicyService{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resolution, err := resolver.ResolveDependencyUpdate(ctx, ref, install, []domain.ProjectControlFile{first, second})
	if !errors.Is(err, context.Canceled) || len(runner.calls) != 0 || len(resolution.Graph().Nodes()) != 0 {
		t.Fatal("cancelled selection started resources or lost cancellation", err, runner.calls)
	}
}

func (r *npmUpdateRecordingRunner) RunInput(ctx context.Context, input io.Reader, binary string, args ...string) error {
	body, err := io.ReadAll(input)
	if err != nil {
		return err
	}
	r.inputs = append(r.inputs, append([]byte(nil), body...))
	return r.recordingRunner.RunInput(ctx, bytes.NewReader(body), binary, args...)
}

func TestNPMUpdateResolverCopiesFrozenControlsAndKeepsRangeAcrossNewVersion(t *testing.T) {
	manifest := []byte(`{"name":"app","dependencies":{"primary":"^1.0.0"},"scripts":{"postinstall":"must-not-execute"}}`)
	before := strings.Replace(resolverLockJSON(), `"primary":"1.0.0"`, `"primary":"^1.0.0"`, 1)
	after := strings.ReplaceAll(strings.Replace(before, `"version":"1.0.0"`, `"version":"1.2.0"`, 1), "primary-1.0.0.tgz", "primary-1.2.0.tgz")
	runner := &npmUpdateRecordingRunner{recordingRunner: recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), []byte("172.30.0.0/24"), []byte("0123456789ab"), nil, nil, []byte(resolverNPMVersion), nil, []byte(after)}}}
	resolver, err := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, &sequencedObserver{}, &recordingResolverPolicyService{})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := artifactnpm.ProjectRootReference(manifest)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/npm-update-source")
	install, _ := domain.NewNPMProjectInstallContext(target)
	controls := []domain.ProjectControlFile{}
	for _, item := range []struct {
		name string
		body []byte
	}{{"package.json", manifest}, {"package-lock.json", []byte(before)}} {
		control, err := domain.NewProjectControlFile(item.name, item.body, true)
		if err != nil {
			t.Fatal(err)
		}
		controls = append(controls, control)
	}
	resolution, err := resolver.ResolveDependencyUpdate(context.Background(), ref, install, controls)
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.inputs) != 2 || !bytes.Equal(runner.inputs[0], manifest) || string(runner.inputs[1]) != before || resolution.Graph().Nodes()[0].Artifact().Identity().Version() != "1.2.0" {
		t.Fatal("original selection not preserved", runner.inputs, resolution)
	}
	selected := map[string][]byte{}
	for _, control := range resolution.ProjectControls() {
		selected[control.Name()] = control.Body()
	}
	if !bytes.Equal(selected["package.json"], manifest) || string(selected["package-lock.json"]) != after {
		t.Fatal("selected public controls lost")
	}
	foundUpdate, removed := false, false
	for _, call := range runner.calls {
		joined := strings.Join(call.arguments, " ")
		if strings.Contains(joined, "npm update ") {
			foundUpdate = true
			for _, flag := range []string{"--package-lock-only", "--ignore-scripts", "--no-audit", "--no-fund", "--save=false", "--registry=https://registry.npmjs.org/"} {
				if !strings.Contains(joined, flag) {
					t.Fatal("update weakened fixed selection flags", joined)
				}
			}
		}
		if strings.Contains(joined, "npm install ") {
			t.Fatal("update aliased to install", joined)
		}
		if joined == "rm --force 0123456789ab" {
			removed = true
		}
	}
	if !foundUpdate || !removed {
		t.Fatal("update/cleanup missing", runner.calls)
	}
}

func TestNPMUpdateResolverRejectsUnsupportedOriginalBeforeNetwork(t *testing.T) {
	runner := &npmUpdateRecordingRunner{}
	resolver, err := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, &sequencedObserver{}, &recordingResolverPolicyService{})
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := artifactnpm.ParseReference("primary")
	target, _ := domain.NewInstallTarget("/tmp/npm-update-source")
	install, _ := domain.NewNPMProjectInstallContext(target)
	bad, _ := domain.NewProjectControlFile("package.json", []byte(`{"dependencies":{"primary":"file:/outside"}}`), true)
	lock, _ := domain.NewProjectControlFile("package-lock.json", []byte(resolverLockJSON()), true)
	if _, err := resolver.ResolveDependencyUpdate(context.Background(), ref, install, []domain.ProjectControlFile{bad, lock}); err == nil || len(runner.calls) != 0 || len(runner.inputs) != 0 {
		t.Fatal("unsupported original reached network/runtime", err)
	}
}
