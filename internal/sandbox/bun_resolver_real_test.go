//go:build linux && realcorpus

package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestBunPublicNativeResolverIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_NATIVE_RESOLVER_INTEGRATION") != "1" {
		t.Fatal("declare ordinary-user Bun native resolver qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	r, err := NewLinuxBunResolverWithExecutorAndPolicy(executor, supervisor.Observer(), newIntegrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	source, err := artifactnpm.NewPublicResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget("/tmp/haa-bun-native-qualification")
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	controls := func(name string) []domain.ProjectControlFile {
		manifest, err := os.ReadFile("../artifact/npm/testdata/bun-lock/" + name + ".package.json")
		if err != nil {
			t.Fatal(err)
		}
		file, err := domain.NewProjectControlFile("package.json", manifest, true)
		if err != nil {
			t.Fatal(err)
		}
		return []domain.ProjectControlFile{file}
	}
	run := func(name string, intent artifactnpm.BunSelectionIntent, selector string, inputControls []domain.ProjectControlFile, wantArchives int, wantVersion string) (BunNativeSelection, bool) {
		var result BunNativeSelection
		ok := t.Run(name, func(t *testing.T) {
			input, err := artifactnpm.NewBunSelectionInput(intent, selector, inputControls, runtimeidentity.BunVersion)
			if err != nil {
				t.Fatal(err)
			}
			result, err = r.SelectProject(ctx, input)
			if err != nil {
				t.Fatal("native selection/observation/cleanup", err)
			}
			snapshot, err := result.BindProjectSnapshot(ctx, source, install)
			if err != nil || !snapshot.Valid() || len(snapshot.Dependencies()) != wantArchives || snapshot.DependencyFree() != (wantArchives == 0) {
				t.Fatal("native source/snapshot binding", err)
			}
			if wantVersion != "" && (len(snapshot.Dependencies()) != 1 || snapshot.Dependencies()[0].Identity().Version() != wantVersion) {
				t.Fatal("actual selected version differs", wantVersion)
			}
			versions := []string{}
			for _, artifact := range snapshot.Dependencies() {
				versions = append(versions, artifact.Identity().Name()+"@"+artifact.Identity().Version())
			}
			t.Logf("bun_public_native=%s unique_archives=%d instances=%d roots=%d selected_versions=%s source_bound=true snapshot_bound=true native_output_unapproved=true archive_quarantine/Policy/Host_install=NOT_EXERCISED", name, wantArchives, len(result.Selection().Packages()), len(result.Selection().Roots()), strings.Join(versions, ","))
		})
		return result, ok
	}
	var single BunNativeSelection
	for _, test := range []struct {
		name  string
		count int
	}{{"empty", 0}, {"single", 1}, {"transitive", 2}, {"peer", 4}} {
		selected, ok := run(test.name, artifactnpm.BunInstall, "", controls(test.name), test.count, "")
		if !ok {
			return
		}
		if test.name == "single" {
			single = selected
		}
	}
	if _, ok := run("frozen-install", artifactnpm.BunInstall, "", single.Controls(), 1, "6.0.0"); !ok {
		return
	}
	if _, ok := run("add", artifactnpm.BunAdd, "is-number@6.0.0", controls("empty"), 1, "6.0.0"); !ok {
		return
	}
	// Explicitly synthetic old constraint, retaining the actual native6.0.0
	// row/SRI. The preceding frozen invocation proves native readability;
	// this is not an unmanaged-project ownership/adoption qualification.
	old := single.Controls()
	m := bytes.Replace(old[0].Body(), []byte(`"is-number":"6.0.0"`), []byte(`"is-number":">=6.0.0 <8.0.0"`), 1)
	l := bytes.Replace(old[1].Body(), []byte(`"is-number": "6.0.0"`), []byte(`"is-number": ">=6.0.0 <8.0.0"`), 1)
	if bytes.Equal(m, old[0].Body()) || bytes.Equal(l, old[1].Body()) {
		t.Fatal("synthetic old constraint not constructed")
	}
	manifest, err := domain.NewProjectControlFile("package.json", m, true)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := domain.NewProjectControlFile("bun.lock", l, true)
	if err != nil {
		t.Fatal(err)
	}
	old = []domain.ProjectControlFile{manifest, lock}
	if _, ok := run("synthetic-old-frozen", artifactnpm.BunInstall, "", old, 1, "6.0.0"); !ok {
		return
	}
	if _, ok := run("selected-update", artifactnpm.BunUpdate, "is-number", old, 1, "7.0.0"); !ok {
		return
	}
	run("full-update", artifactnpm.BunUpdate, "", old, 1, "7.0.0")
}

func TestBunActualNativeProjectSelectionBoundary(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_NATIVE_RESOLVER_INTEGRATION") != "1" {
		t.Fatal("declare ordinary-user Bun native boundary qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, mode := range []string{"complete", "command and cleanup", "cleanup acknowledgement", "incomplete observation", "forbidden observation"} {
		ok := t.Run(mode, func(t *testing.T) {
			cause := errors.New("injected Bun command/cleanup failure")
			injections := 0
			runner := &pnpmNativeFaultRunner{CommandRunner: admissionAwareRunner(executor), mode: mode, nativeTool: bunToolGuestPath, cause: cause}
			r, err := NewBunResolver(runner, &pnpmCountingEndpoints{}, pnpmNativeFaultObserver{supervisor.Observer(), mode, &injections}, integrationCapabilityProbe(executor), &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			selected, err := r.SelectProject(ctx, bunResolverInput(t, artifactnpm.BunInstall, "", `{}`))
			if mode == "complete" {
				if err != nil || !selected.Selection().DependencyFree() || len(selected.Controls()) != 2 {
					t.Fatal("native empty boundary", err)
				}
			} else {
				if err == nil || len(selected.Controls()) != 0 || selected.Selection().DependencyFree() {
					t.Fatal("failed scope retained output", err)
				}
				if (mode == "command and cleanup" || mode == "cleanup acknowledgement") && !errors.Is(err, cause) {
					t.Fatal("original failure lost", err)
				}
			}
			wantCommand, wantCleanup, wantObservation := 0, 0, 0
			switch mode {
			case "command and cleanup":
				wantCommand, wantCleanup = 1, 1
			case "cleanup acknowledgement":
				wantCleanup = 1
			case "incomplete observation", "forbidden observation":
				wantObservation = 1
			}
			if runner.commandFaults != wantCommand || runner.cleanupFaults != wantCleanup || injections != wantObservation {
				t.Fatal("intended failure not reached", runner.commandFaults, runner.cleanupFaults, injections)
			}
			t.Logf("bun_native_boundary=%s output_controls=%d rejected=%t public_egress=NOT_TESTED", mode, len(selected.Controls()), err != nil)
		})
		if !ok && mode == "complete" {
			return
		}
	}
}
