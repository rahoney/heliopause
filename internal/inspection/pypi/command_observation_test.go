package pypi

import (
	"context"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
	"strings"
	"testing"
)

type commandRunner struct {
	result sandbox.PythonObservationResult
	plan   artifactpypi.ObservationPlan
}

func (r *commandRunner) InspectWheel(context.Context, domain.AcquiredArtifact, []string) (domain.SandboxResult, error) {
	panic("typed plan required")
}
func (r *commandRunner) InspectWheelWithPlan(_ context.Context, _ domain.AcquiredArtifact, p artifactpypi.ObservationPlan, _ []domain.AcquiredArtifact) (sandbox.PythonObservationResult, error) {
	r.plan = p
	return r.result, nil
}

func TestCommandObservationPreservesPolicyAndEvidenceScope(t *testing.T) {
	for _, profile := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(profile, func(t *testing.T) {
			selected, _ := artifactpypi.PyTorchProfile(profile)
			ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), selected)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				name                      string
				zero, incomplete, missing bool
				category                  domain.ObservationCategory
				subject                   string
				allow                     bool
			}{
				{name: "zero", zero: true, allow: true},
				{name: "nonzero", allow: true},
				{name: "network before nonzero", category: domain.ObservationNetwork, subject: "network-attempt"},
				{name: "exec before nonzero", category: domain.ObservationProcess, subject: "process-exec-unexpected"},
				{name: "filesystem before nonzero", category: domain.ObservationFilesystem, subject: "filesystem-outside-workspace"},
				{name: "incomplete trusted evidence", incomplete: true},
				{name: "missing command outcome", missing: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					root := t.TempDir()
					a := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", map[string]string{"example/cli.py": "raise ModuleNotFoundError('pandas')\n", "example-1.0.dist-info/entry_points.txt": "[console_scripts]\nview = example.cli:main\n"}, "")
					static, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
					wheel, _, err := static.InspectWheel(ctx, a)
					if err != nil {
						t.Fatal(err)
					}
					var observations []domain.SandboxObservation
					if tc.subject != "" {
						observations = append(observations, observation(t, tc.category, tc.subject))
					}
					result := completedResultWithObservations(t, observations)
					if tc.incomplete {
						result, err = domain.NewSandboxResult(result.SessionID(), domain.SandboxIncomplete, "M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE", nil)
						if err != nil {
							t.Fatal(err)
						}
					}
					r := &commandRunner{result: sandbox.PythonObservationResult{SandboxResult: result}}
					if !tc.missing {
						r.result.Commands = []sandbox.PythonCommandObservation{{Module: "example.cli", UnitID: "DIRECT_IMPORT:" + strings.Repeat("a", 64), OwnerSHA256: a.Digest().String(), ZeroExit: tc.zero}}
					}
					inspector, _ := NewDynamicInspector(r)
					report, err := inspector.InspectWheel(ctx, a, wheel)
					if err != nil {
						t.Fatal(err)
					}
					if len(r.plan.Units) != 2 || r.plan.Units[0].Coverage != artifactpypi.RequiredObservation || r.plan.Units[1].Coverage != artifactpypi.PostInstallCommandObservation {
						t.Fatal(r.plan.Units)
					}
					if tc.subject != "" && len(report.Findings()) != 1 {
						t.Fatal("actionable finding lost", report.Findings())
					}
					if !tc.incomplete && !tc.missing {
						disposition := "NOT_ATTESTED"
						if tc.zero && tc.subject == "" {
							disposition = "ATTESTED"
						}
						ev := report.Evidence()[len(report.Evidence())-1]
						if !strings.Contains(ev.Summary(), `"disposition":"`+disposition+`"`) || !strings.Contains(ev.Summary(), `"later_execution_enforced":false`) || !strings.Contains(ev.Summary(), `"functionality_attested":false`) {
							t.Fatal(ev.Summary())
						}
					}
					verified, err := (verificationpypi.IntegrityVerifier{}).Verify(ctx, a)
					if err != nil {
						t.Fatal(err)
					}
					var refs []domain.EvidenceReference
					for _, ev := range append(verified.Evidence(), report.Evidence()...) {
						ref, err := domain.NewEvidenceReference(ev.ID(), "evidence:"+ev.ID().String())
						if err != nil {
							t.Fatal(err)
						}
						refs = append(refs, ref)
					}
					run, _ := domain.NewRunID()
					input, err := domain.NewPolicyInput(run, a, verified, report, refs)
					if err != nil {
						t.Fatal(err)
					}
					decision, err := (policy.M3{}).Evaluate(input)
					if err != nil {
						t.Fatal(err)
					}
					if (decision.Decision() == domain.DecisionAllow) != tc.allow {
						t.Fatalf("decision=%v allow=%v", decision.Decision(), tc.allow)
					}
				})
			}
		})
	}
}
