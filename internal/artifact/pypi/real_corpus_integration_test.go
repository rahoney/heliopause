//go:build realcorpus

package pypi

import (
	"encoding/json"
	"os"
	"testing"
)

func TestModelC_RealCorpusQualification(t *testing.T) {
	corpusDir := os.Getenv("HELOX_CORPUS_ROOT")
	if corpusDir == "" {
		t.Fatal("HELOX_CORPUS_ROOT is required; run scripts/prepare-wheel-corpus.py first")
	}
	var manifest []corpusEntry
	data, err := os.ReadFile("testdata/real-corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest) == 0 {
		t.Fatal("empty required corpus")
	}
	for _, tc := range manifest {
		t.Run(tc.FileOnDisk, func(t *testing.T) {
			insp, err := inspectCorpusWheel(corpusDir, tc)
			if err != nil {
				stage, _ := WheelValidationStageOf(err)
				t.Fatalf("%s: %v (stage=%s)", tc.Canonical, err, stage)
			}
			if insp.Project != tc.Project || insp.Version != tc.Version {
				t.Fatalf("identity mismatch: %s %s", insp.Project, insp.Version)
			}
			wheelFilename := tc.Canonical
			if insp.NoImportSurface != tc.NoImport {
				t.Fatalf("NoImportSurface = %v, want %v", insp.NoImportSurface, tc.NoImport)
			}

			var policy ResourcePolicy
			switch tc.Profile {
			case "pytorch:cpu":
				policy = pyTorchCPUResourcePolicy()
			case "pytorch:cu126":
				policy = pyTorchCU126ResourcePolicy()
			case "pytorch:cu130":
				policy = pyTorchCU130ResourcePolicy()
			case "pytorch:cu132":
				policy = pyTorchCU132ResourcePolicy()
			default:
				policy = defaultResourcePolicy()
			}

			plan, err := BuildObservationPlan(insp, policy)
			if err != nil {
				t.Fatalf("BuildObservationPlan(%q) failed: %v", wheelFilename, err)
			}
			if len(tc.WantImports) > 0 {
				for _, want := range tc.WantImports {
					found := false
					for _, got := range plan.ImportCandidates {
						if got == want {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected candidate import %q in %v", want, plan.ImportCandidates)
					}
				}
			}
			if err := ValidateTypedObservationPlan(plan, policy); err != nil || !plan.Admissible() {
				t.Fatalf("authenticated corpus observation plan is nonqualifying: %v, entry points=%v, scripts=%v", err, plan.EntryPointCoverage, plan.ScriptCoverage)
			}
			t.Logf("%s profile=%s direct=%d units=%d", tc.Canonical, tc.Profile, plan.TotalImportCount, len(plan.Units))
			if tc.ExpectedCandidateCount > 0 && plan.TotalImportCount != tc.ExpectedCandidateCount {
				t.Fatalf("%s total imports = %d, want %d", tc.FileOnDisk, plan.TotalImportCount, tc.ExpectedCandidateCount)
			}
		})
	}
	t.Logf("REAL_CORPUS_CLASSIFICATION: REPRESENTATIVE_REAL_CORPUS (%d authenticated packages verified)", len(manifest))
}
