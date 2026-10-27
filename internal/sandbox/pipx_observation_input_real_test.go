//go:build linux && realcorpus

package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestPipxObservationActualAcquiredInputAndFrozenProgram(t *testing.T) {
	filename := os.Getenv("HELOX_PIPX_RUNTIME_ZIP")
	if filename == "" {
		t.Skip("explicit selected immutable ZIP required")
	}
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	value, err := artifactpypi.ParsePipxRuntimeZIP(body)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := value.ObservationPlan()
	if err != nil {
		t.Fatal(err)
	}
	a := pipxObservationArtifact(t, "")
	input, err := preparePipxObservationInput(context.Background(), a, plan)
	if err != nil || input.artifact != a || input.tool == nil || len(input.units) != 20 {
		t.Fatal("acquired identity/plan binding failed", err)
	}
	original := bytes.Clone(body)
	body[0] ^= 1
	if !bytes.Equal(input.tool.body, original) {
		t.Fatal("caller changed frozen DATA bytes")
	}
	transport := &pipxDataRoundTrip{container: strings.Repeat("b", 64)}
	if err := input.tool.introduce(context.Background(), transport, transport.container); err != nil || transport.calls != 2 {
		t.Fatal("acquired DATA did not reuse existing transport", err)
	}
	for _, mode := range []string{"source", "name", "variant", "digest", "size"} {
		t.Run(mode, func(t *testing.T) {
			bad, err := preparePipxObservationInput(context.Background(), pipxObservationArtifact(t, mode), plan)
			if err == nil || bad.tool != nil || bad.units != nil {
				t.Fatal("substituted subject reached acquired DATA")
			}
		})
	}
	type fact struct {
		ID, Candidate, OwnerDigest, ProgramSHA256 string
		Coverage                                  artifactpypi.ObservationCoverage
	}
	var facts []fact
	required, commands := 0, 0
	for _, unit := range input.units {
		sum := sha256.Sum256([]byte(unit.program))
		facts = append(facts, fact{unit.id, unit.candidate, unit.ownerDigest, hex.EncodeToString(sum[:]), unit.coverage})
		if unit.coverage == artifactpypi.RequiredObservation {
			required++
		} else if unit.coverage == artifactpypi.PostInstallCommandObservation {
			commands++
		} else {
			t.Fatal("coverage changed")
		}
	}
	if required != 15 || commands != 5 {
		t.Fatal("required/post-install coverage changed", required, commands)
	}
	encoded, err := json.Marshal(struct {
		Program string
		Units   []fact
	}{pipxDirectObservation, facts})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("pipx-frozen-input=%s", encoded)
	t.Logf("original synthetic HAA version=%s units=%d required=%d post-install=%d; no completion evidence", a.Identity().Version(), len(facts), required, commands)
}
