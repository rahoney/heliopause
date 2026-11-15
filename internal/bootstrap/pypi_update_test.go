package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type upgradeSelectionPorts struct {
	prerequisiteSelectionPorts
	updates    int
	controls   []domain.ProjectControlFile
	lockUpdate func(context.Context, domain.InstallContext, []domain.ProjectControlFile, string) (domain.ProjectDependencyUpdate, error)
}

func (p *upgradeSelectionPorts) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, controls []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	return p.lockUpdate(ctx, install, controls, selector)
}

func TestPythonRequirementsRoutingRetainsNativeInputAndErrors(t *testing.T) {
	for _, mode := range []string{"public", "cancelled", "missing-public", "unsupported-public", "torch-only"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			target, _ := domain.NewInstallTarget("/tmp/haa-requirements-routing")
			install, _ := domain.NewPythonVenvInstallContext(target)
			control, _ := domain.NewProjectControlFile("requirements-input.json", []byte("frozen controls"), true)
			cause := errors.New("native selection cause")
			calls := 0
			base := &upgradeSelectionPorts{lockUpdate: func(got context.Context, selected domain.InstallContext, controls []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
				calls++
				if got != ctx || selected != install || selector != "native selector" || len(controls) != 1 || !bytes.Equal(controls[0].Body(), control.Body()) {
					t.Fatal("router changed the native request")
				}
				if got.Err() != nil {
					return domain.ProjectDependencyUpdate{}, got.Err()
				}
				return domain.ProjectDependencyUpdate{}, cause
			}}
			routes := map[string]ports.DependencyResolver{"pypi": base}
			switch mode {
			case "missing-public":
				delete(routes, "pypi")
			case "unsupported-public":
				routes["pypi"] = &base.prerequisiteSelectionPorts
			case "torch-only":
				delete(routes, "pypi")
				routes["pytorch-cu126"] = base
			}
			update, err := (pythonSourceResolver{routes: routes}).ResolveProjectLockUpdate(ctx, install, []domain.ProjectControlFile{control}, "native selector")
			if err == nil || update.Valid() {
				t.Fatal("router manufactured a completed selection")
			}
			if mode == "public" && (!errors.Is(err, cause) || calls != 1) || mode == "cancelled" && (!errors.Is(err, context.Canceled) || calls != 1) || mode != "public" && mode != "cancelled" && calls != 0 {
				t.Fatal("public route/error changed or another source was used", mode, calls, err)
			}
		})
	}
}

type namedUpdateResourceFixture struct {
	t       *testing.T
	profile artifactpypi.SourceProfile
	calls   int
}

func (i *namedUpdateResourceFixture) Install(ctx context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	i.t.Helper()
	i.calls++
	deadline, ok := ctx.Deadline()
	remaining := time.Until(deadline)
	wanted := i.profile.ResourcePolicy().Duration()
	if !ok || remaining > wanted || remaining < wanted-time.Second || artifactpypi.RootSourceProfileNameFromContext(ctx) != i.profile.Name() || request.Reference().Source() != i.profile.Source() {
		i.t.Fatal("named update lost resource policy or deadline", remaining, ctx)
	}
	return application.InstallOutcome{}, ctx.Err()
}

func TestPyTorchUpgradeResourceWrapperCoversUpdateRequestAndCancellation(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		profile, _ := artifactpypi.PyTorchProfile(name)
		for _, cancelled := range []bool{false, true} {
			target, _ := domain.NewInstallTarget("/tmp/haa-named-update-resource")
			install, _ := domain.NewPythonVenvInstallContext(target)
			request, err := application.NewInstallUpdateRequest(profile.Source(), "torch>=2,<4", install, false)
			if err != nil {
				t.Fatal(err)
			}
			fixture := &namedUpdateResourceFixture{t: t, profile: profile}
			wrapper := pyTorchResourceInstaller{installer: fixture, profile: profile}
			ctx, cancel := context.WithCancel(context.Background())
			if cancelled {
				cancel()
			}
			_, err = wrapper.Install(ctx, request)
			cancel()
			if fixture.calls != 1 || cancelled != errors.Is(err, context.Canceled) {
				t.Fatal("update wrapper lost cancellation", err)
			}
		}
	}
}

func (p *upgradeSelectionPorts) ResolveDependencyUpdate(_ context.Context, _ domain.ArtifactReference, _ domain.InstallContext, controls []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	p.updates++
	p.controls = controls
	hash, _ := domain.NewSHA256Digest(strings.Repeat("f", 64))
	return domain.NewDependencyResolution(p.graph, "fixture-runtime", hash)
}

func TestPythonUpgradeRoutingPreservesPrerequisiteValidationAndFrozenInput(t *testing.T) {
	selected := qualificationResolutionFixture(t, "valid")
	pin, _, err := qualificationPrerequisite(selected, "cu126", runtimeidentity.PyTorchCU126Version, cudaBindingsNumPyPlan)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"valid", "empty", "malformed", "read-failure", "stale", "unsupported-source", "install-only"} {
		t.Run(mode, func(t *testing.T) {
			base := &upgradeSelectionPorts{prerequisiteSelectionPorts: prerequisiteSelectionPorts{graph: selected.Graph()}}
			source, _ := domain.NewSourceID("pypi")
			reference, _ := domain.NewArtifactReference(source, "primary")
			request := pin
			if mode == "stale" {
				request.TargetSHA256 = strings.Repeat("c", 64)
			}
			body, _ := json.Marshal(request)
			raw := string(body)
			if mode == "empty" {
				raw = ""
			}
			if mode == "malformed" {
				raw = "{}"
			}
			routes := map[string]ports.DependencyResolver{"pypi": base}
			if mode == "unsupported-source" {
				delete(routes, "pypi")
			}
			if mode == "install-only" {
				routes["pypi"] = &base.prerequisiteSelectionPorts
			}
			resolver := prerequisiteCheckingResolver{resolver: pythonSourceResolver{routes: routes}, read: func() (string, error) {
				if mode == "read-failure" {
					return "", errors.New("fixture read failure")
				}
				return raw, nil
			}}
			control, _ := domain.NewProjectControlFile("pypi-update-input.json", []byte("frozen-input"), true)
			resolution, err := resolver.ResolveDependencyUpdate(context.Background(), reference, domain.InstallContext{}, []domain.ProjectControlFile{control})
			accepted := mode == "valid" || mode == "empty"
			if accepted != (err == nil) || !accepted && len(resolution.Graph().Nodes()) != 0 {
				t.Fatal("prerequisite/routing acceptance changed", err)
			}
			if base.resolves != 0 || base.acquires != 0 {
				t.Fatal("update fell through ordinary install or acquired bytes")
			}
			called := accepted || mode == "stale"
			if called != (base.updates == 1) {
				t.Fatal("invalid input reached resolver or selection was skipped", base.updates)
			}
			if called && (len(base.controls) != 1 || !bytes.Equal(base.controls[0].Body(), control.Body())) {
				t.Fatal("frozen input changed")
			}
		})
	}
}
