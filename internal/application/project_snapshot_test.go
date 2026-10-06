package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

var errSnapshotInspectionReached = errors.New("synthetic inspection boundary reached")

type snapshotOrderFixture struct {
	snapshot                domain.ProjectDependencySnapshot
	controls                []domain.ProjectControlFile
	begun, closed, verified bool
}

func (f *snapshotOrderFixture) Begin(context.Context, domain.InstallContext) (ports.ProjectMutationGuard, error) {
	f.begun = true
	return f, nil
}
func (f *snapshotOrderFixture) ResolveProjectDependencies(context.Context, domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	return f.snapshot, nil
}
func (f *snapshotOrderFixture) Controls() []domain.ProjectControlFile { return f.controls }
func (f *snapshotOrderFixture) VerifyUnchanged(context.Context) error { f.verified = true; return nil }
func (f *snapshotOrderFixture) Commit(context.Context, domain.ProjectDependencyUpdate, domain.StagedProjectSet) error {
	return errors.New("unexpected publication")
}
func (f *snapshotOrderFixture) CommitSnapshot(context.Context, domain.ProjectDependencySnapshot, domain.StagedProjectSet) error {
	return errors.New("unexpected publication")
}
func (f *snapshotOrderFixture) Close() error { f.closed = true; return nil }
func (f *snapshotOrderFixture) InspectProject(context.Context, domain.ProjectDependencySnapshot) (domain.ProjectVerifiedSet, error) {
	return domain.ProjectVerifiedSet{}, errSnapshotInspectionReached
}
func (f *snapshotOrderFixture) StageProject(context.Context, domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	return domain.StagedProjectSet{}, errors.New("unexpected staging")
}

func TestCargoBuildSnapshotControlsMatchByIdentity(t *testing.T) {
	for _, sourceName := range []string{"go-proxy", "crates-io"} {
		for _, scenario := range []string{"ordered", "reversed", "duplicate", "substituted", "cancelled"} {
			t.Run(sourceName+"/"+scenario, func(t *testing.T) {
				source, _ := domain.NewSourceID(sourceName)
				target, _ := domain.NewInstallTarget("/fixture/project")
				install, _ := domain.NewInstallContext(target)
				names := []string{"go.mod", "go.sum"}
				if sourceName == "crates-io" {
					names = []string{"Cargo.lock", "Cargo.toml"}
				}
				f := &snapshotOrderFixture{}
				var digests []domain.ProjectControlDigest
				for _, name := range names {
					file, _ := domain.NewProjectControlFile(name, []byte("frozen "+name), true)
					f.controls = append(f.controls, file)
					digest, _ := domain.NewProjectControlDigest(name, file.Digest())
					digests = append(digests, digest)
				}
				graph, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
				var err error
				f.snapshot, err = domain.NewDependencyFreeProjectSnapshot(install, source, digests, graph)
				if err != nil {
					t.Fatal(err)
				}
				switch scenario {
				case "reversed":
					f.controls[0], f.controls[1] = f.controls[1], f.controls[0]
				case "duplicate":
					f.controls[1] = f.controls[0]
				case "substituted":
					f.controls[0], _ = domain.NewProjectControlFile(names[0], []byte("changed"), true)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if scenario == "cancelled" {
					cancel()
				}
				result, err := resolveProjectSnapshot(ctx, install, f, f, f, f, sourceName, "fixture")
				if result.Valid() {
					t.Fatal("failed inspection returned approved snapshot")
				}
				if scenario == "ordered" || scenario == "reversed" {
					if !errors.Is(err, errSnapshotInspectionReached) || !f.verified || !f.closed {
						t.Fatalf("matching frozen controls were not inspected: %v", err)
					}
				} else {
					if err == nil || errors.Is(err, errSnapshotInspectionReached) || f.verified {
						t.Fatal("invalid controls reached inspection")
					}
					if scenario == "cancelled" && f.begun {
						t.Fatal("cancelled request entered guard")
					}
				}
			})
		}
	}
}
