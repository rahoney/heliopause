package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type uvProjectCompositionNative struct{ calls int }

func (n *uvProjectCompositionNative) ResolveProjectLockUpdate(context.Context, domain.InstallContext, []domain.ProjectControlFile, string) (domain.ProjectDependencyUpdate, error) {
	n.calls++
	return domain.ProjectDependencyUpdate{}, errors.New("unexpected native selection")
}
func (n *uvProjectCompositionNative) Run(context.Context, string, []string) error {
	n.calls++
	return errors.New("unexpected materializer execution")
}

func TestUVProjectCompositionRejectsMissingCollaboratorsAndCanceledRequests(t *testing.T) {
	for _, name := range []string{"intake", "inspector", "evidence", "promoter", "resolver", "canceled", "missing target"} {
		t.Run(name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			intakeRoot, evidenceRoot, cacheRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache")
			intake, err := artifactpypi.NewPublicIntake(intakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			runtime := sandbox.PinnedPythonRuntime()
			inspector, err := inspectionpypi.NewStaticInspector(intakeRoot, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(evidenceRoot)
			if err != nil {
				t.Fatal(err)
			}
			native := &uvProjectCompositionNative{}
			promoter, err := promotion.NewPyPIPromotionWithRunner(cacheRoot, native)
			if err != nil {
				t.Fatal(err)
			}
			b := uvProjectBindings{intakeRoot, evidenceRoot, cacheRoot, intake, inspector, store, promoter, native, nil}
			switch name {
			case "intake":
				b.intake = nil
			case "inspector":
				b.inspector = nil
			case "evidence":
				b.evidence = nil
			case "promoter":
				b.promoter = nil
			case "resolver":
				b.resolver = nil
			}
			service, err := newUVProjectService(b, root, artifactpypi.UVProjectSync, "", false)
			if name != "canceled" && name != "missing target" {
				if err == nil || service != nil {
					t.Fatal("missing collaborator accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			target, err := domain.NewInstallTarget(filepath.Join(root, ".venv"))
			if err != nil {
				t.Fatal(err)
			}
			install, err := domain.NewPythonVenvInstallContext(target)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if name == "canceled" {
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			if set, err := service.Apply(ctx, install); err == nil || set.Valid() || name == "canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal("invalid request completed", err)
			}
			if native.calls != 0 {
				t.Fatal("invalid request reached native selection/materializer")
			}
			if _, err := os.Lstat(target.String()); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid request created target", err)
			}
			if _, _, _, ok := service.promoter.PublishedRecords(); ok {
				t.Fatal("invalid request exposed completed records")
			}
		})
	}
}
