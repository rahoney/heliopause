package projectbuild

import (
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type recordedBuildObservation struct {
	observation domain.ProjectBuildObservation
}

func (b recordedBuildObservation) Build(context.Context, domain.ProjectBuildInputs) (domain.ProjectBuildObservation, error) {
	return b.observation, nil
}

func TestBuildEnvironmentCoverageAndEvidence(t *testing.T) {
	for _, kind := range []string{"go", "cargo"} {
		t.Run(kind, func(t *testing.T) {
			digest := func(c string) domain.ContentDigest {
				value, err := domain.NewSHA256Digest(strings.Repeat(c, 64))
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			target, _ := domain.NewInstallTarget(t.TempDir())
			install, _ := domain.NewInstallContext(target)
			registry, _ := domain.NewSourceID(kind)
			control, _ := domain.NewProjectControlDigest("fixture.control", digest("a"))
			snapshot, err := domain.NewDependencyFreeProjectSnapshot(install, registry, []domain.ProjectControlDigest{control}, digest("b"))
			if err != nil {
				t.Fatal(err)
			}
			run, _ := domain.NewRunID()
			local, _ := domain.NewSourceID("project-local")
			acquired := func(variant, version string, hash domain.ContentDigest) domain.AcquiredArtifact {
				identity, err := domain.NewResolvedArtifactIdentity(local, "fixture", version, kind+"-"+variant)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, hash, "intake:"+run.String()+":"+kind+"-"+variant, 1, "sha256:"+hash.String())
				if err != nil {
					t.Fatal(err)
				}
				return artifact
			}
			source := acquired("source", "source", digest("c"))
			cache := acquired("cache", snapshot.GraphDigest().String(), digest("d"))
			inputs, err := domain.NewProjectBuildInputs(snapshot, source, cache, run, "default")
			if err != nil {
				t.Fatal(err)
			}
			output := acquired("output", run.String(), digest("e"))
			binding, err := domain.NewDerivationBinding(source.Digest(), []domain.ContentDigest{cache.Digest(), snapshot.GraphDigest()}, kind+"-build-linux-amd64", digest("f"))
			if err != nil {
				t.Fatal(err)
			}
			for _, test := range []struct {
				name    string
				count   int
				covered bool
			}{
				{"missing", 0, false}, {"one-environment", 1, true}, {"missing-vfs-coverage", 2, false}, {"complete", 2, true},
			} {
				t.Run(test.name, func(t *testing.T) {
					var states []domain.SandboxEnvironmentState
					for _, profile := range []domain.SandboxEnvironmentProfile{domain.SandboxEnvironmentMinimal, domain.SandboxEnvironmentDevelopment}[:test.count] {
						states = append(states, domain.SandboxEnvironmentState{Profile: profile, FixtureDigest: digest("a"), Before: digest("b"), After: digest("b"), VFSOperationsCovered: test.covered})
					}
					session, _ := domain.NewSandboxSessionID()
					result, err := domain.NewSandboxResult(session, domain.SandboxCompleted, "", nil, states...)
					if err != nil {
						t.Fatal(err)
					}
					observation, err := domain.NewProjectBuildObservation(inputs, result, output, binding)
					if err != nil {
						t.Fatal(err)
					}
					inspector, err := NewInspector(recordedBuildObservation{observation}, kind)
					if err != nil {
						t.Fatal(err)
					}
					report, err := inspector.InspectBuild(context.Background(), inputs)
					if test.name != "complete" {
						if err == nil {
							t.Fatal("missing required environment coverage accepted")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					checks := report.Inspection().Executions()
					facts, err := domain.BindEvidenceProfiles(checks, report.Inspection().Evidence())
					if err != nil || len(checks) != 1 || !checks[0].Required() || checks[0].Status() != domain.ExecutionCompleted || len(facts) != 4 {
						t.Fatal("current build check/Evidence incomplete", err)
					}
					seen := 0
					for _, fact := range facts {
						if fact.Kind() == "workspace-state" {
							seen++
							if fact.CheckID() != checks[0].ID() || fact.ProfileDigest() != checks[0].ProfileDigest() || fact.Identity() != source.Identity() || fact.Digest() != source.Digest() {
								t.Fatal("workspace Evidence subject/check/profile substituted")
							}
						}
					}
					if seen != 2 {
						t.Fatal("two environment Evidence values missing")
					}
				})
			}
		})
	}
}
