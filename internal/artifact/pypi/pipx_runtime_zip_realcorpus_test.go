//go:build realcorpus

package pypi

import (
	"bytes"
	"os"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestPipxRuntimeZIPActualSelectedSnapshot(t *testing.T) {
	filename := os.Getenv("HELOX_PIPX_RUNTIME_ZIP")
	if filename == "" {
		t.Skip("explicit selected pipx ZIP is required")
	}
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	value, err := ParsePipxRuntimeZIP(body)
	if err != nil || !value.Valid() {
		t.Fatal("selected ZIP rejected", err)
	}
	before := bytes.Clone(body)
	body[0] ^= 1
	copyBody := value.Bytes()
	copyBody[0] ^= 1
	facts := value.Files()
	if len(facts) != 657 {
		t.Fatal("unexpected original member census", len(facts))
	}
	facts[0].Path = "caller mutation"
	if !bytes.Equal(value.Bytes(), before) || value.Files()[0].Path == facts[0].Path || pipxSupplyDigest(value.Bytes()) != runtimeidentity.PipxToolSHA256 {
		t.Fatal("snapshot is mutable")
	}
	for _, changed := range [][]byte{body, append(bytes.Clone(before), 0), before[:len(before)-1]} {
		if invalid, err := ParsePipxRuntimeZIP(changed); err == nil || invalid.Valid() {
			t.Fatal("changed acquisition accepted")
		}
	}
	t.Logf("selected bytes=%d members=%d immutable snapshot and altered/truncated/appended refusals passed", len(before), len(value.Files()))
}

func TestPipxRuntimeZIPActualObservationPlan(t *testing.T) {
	filename := os.Getenv("HELOX_PIPX_RUNTIME_ZIP")
	if filename == "" {
		t.Skip("explicit selected pipx ZIP is required")
	}
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	value, err := ParsePipxRuntimeZIP(body)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := value.ObservationPlan()
	if err != nil || !plan.Valid() {
		t.Fatal("selected plan rejected", err)
	}
	required, commands := 0, 0
	for _, unit := range plan.Units() {
		if unit.Coverage == RequiredObservation {
			required++
		} else if unit.Coverage == PostInstallCommandObservation {
			commands++
		} else {
			t.Fatal("unknown coverage")
		}
		t.Logf("unit %s coverage=%q", unit.Candidate, unit.Coverage)
	}
	if len(plan.Distributions()) != 11 || required != 15 || commands != 5 {
		t.Fatal("selected plan census changed", plan.Distributions(), plan.Units())
	}
	units := plan.Units()
	units[0].Candidate = "caller mutation"
	distributions := plan.Distributions()
	distributions[0].Version = "caller mutation"
	if plan.Units()[0].Candidate == units[0].Candidate || plan.Distributions()[0].Version == distributions[0].Version || !bytes.Equal(plan.ZIP().Bytes(), body) {
		t.Fatal("plan is mutable")
	}
	t.Logf("original distributions=%d required=%d post-install=%d; no Python execution", len(plan.Distributions()), required, commands)
}
