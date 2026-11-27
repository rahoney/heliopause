package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type goTidyNoExecution struct{}

func (goTidyNoExecution) RunGo(context.Context, string, []string, ...string) ([]byte, error) {
	return nil, errors.New("not requested")
}

func TestGoTidyGuardSourceAndControlsRollback(t *testing.T) {
	for _, phase := range []string{"normal", "before-backup", "before-approval"} {
		t.Run(phase, func(t *testing.T) {
			root := canonicalGoTestRoot(t)
			project := filepath.Join(root, "project")
			if err := os.Mkdir(project, 0o700); err != nil {
				t.Fatal(err)
			}
			mod := []byte("module example.com/app\ngo 1.26.0\n")
			if err := os.WriteFile(filepath.Join(project, "go.mod"), mod, 0o640); err != nil {
				t.Fatal(err)
			}
			code := filepath.Join(project, "app.go")
			if err := os.WriteFile(code, []byte("package app\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			originalInfo, err := os.Stat(filepath.Join(project, "go.mod"))
			if err != nil {
				t.Fatal(err)
			}
			cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
			if err != nil {
				t.Fatal(err)
			}
			promoter, err := NewGoProjectPromotion(cache, filepath.Join(root, "state"))
			if err != nil {
				t.Fatal(err)
			}
			resolver, err := sandbox.NewGoModuleResolver(goTidyNoExecution{})
			if err != nil {
				t.Fatal(err)
			}
			preparation, err := NewGoTidyPreparation(promoter, resolver, false)
			if err != nil {
				t.Fatal(err)
			}
			target, _ := domain.NewInstallTarget(project)
			install, _ := domain.NewInstallContext(target)
			ctx := context.Background()
			guard, err := preparation.Begin(ctx, install)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := guard.Close(); err != nil {
					t.Error(err)
				}
			}()
			sum, _ := domain.NewProjectControlFile("go.sum", nil, true)
			modControl, _ := domain.NewProjectControlFile("go.mod", mod, true)
			snapshot, err := artifactgo.BuildProjectSnapshot(install, nil, []byte("example.com/app go@1.26.0\ngo@1.26.0 toolchain@go1.26.0\n"), mod, nil)
			if err != nil {
				t.Fatal(err)
			}
			update, err := domain.NewProjectDependencyUpdateFromSnapshot(guard.Controls(), []domain.ProjectControlFile{modControl, sum}, snapshot)
			if err != nil {
				t.Fatal(err)
			}
			inspected, err := domain.NewInspectedProjectSet(snapshot, nil)
			if err != nil {
				t.Fatal(err)
			}
			decision, err := (policy.M4{}).EvaluateProjectSet(inspected)
			if err != nil {
				t.Fatal(err)
			}
			verified, err := domain.NewProjectVerifiedSet(inspected, decision)
			if err != nil {
				t.Fatal(err)
			}
			staged, err := cache.StageProject(ctx, verified)
			if err != nil {
				t.Fatal(err)
			}
			promoter.updateCheckpoint = func(at string) error {
				if at == phase {
					return os.WriteFile(code, []byte("package changed\n"), 0o600)
				}
				return nil
			}
			err = guard.Commit(ctx, update, staged)
			if (err != nil) != (phase != "normal") {
				t.Fatalf("source publication boundary: %v", err)
			}
			body, readErr := os.ReadFile(filepath.Join(project, "go.mod"))
			info, statErr := os.Stat(filepath.Join(project, "go.mod"))
			if readErr != nil || statErr != nil || !bytes.Equal(body, mod) || !os.SameFile(originalInfo, info) || info.Mode() != originalInfo.Mode() {
				t.Fatal("unchanged module identity/mode not preserved")
			}
			_, sumErr := os.Lstat(filepath.Join(project, "go.sum"))
			if phase != "normal" && !errors.Is(sumErr, os.ErrNotExist) {
				t.Fatal("failed tidy left new sum control")
			}
		})
	}
}
