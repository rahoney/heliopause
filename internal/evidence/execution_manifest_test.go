package evidence_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
)

func executionRecordsFixture(t *testing.T) domain.ExecutionContextData {
	t.Helper()
	set := verifiedFixture(t)
	operation, _ := domain.NewOperationID()
	lock, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	member, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	resolution, err := domain.NewDependencyResolution(set.Inspected().Graph(), "original-resolver", lock)
	if err != nil {
		t.Fatal(err)
	}
	return domain.ExecutionContextData{OperationID: operation, Resolution: resolution, Set: set,
		EntryArtifact: set.Inspected().Inspections()[0].Artifact(), Command: "first", EntryPath: "bin/cli.js",
		EntryDigest: member, EntrySize: 7, RuntimeIdentity: "required-runtime", Arguments: []string{"private-argument-fixture", "", "a b"}}
}

func executionRecords(t *testing.T, data domain.ExecutionContextData) domain.VerifiedBundle {
	t.Helper()
	execution, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := (evidence.Generator{}).BuildExecution(context.Background(), execution)
	if err != nil || !bundle.Valid() {
		t.Fatal("execution records rejected", err)
	}
	return bundle
}

func TestExecutionRecordsBindWholeSetAndMemberWithoutTargetOrArguments(t *testing.T) {
	data := executionRecordsFixture(t)
	first := executionRecords(t, data)
	second := executionRecords(t, data)
	if first.ManifestID() != second.ManifestID() || !bytes.Equal(first.ManifestDocument(), second.ManifestDocument()) || !bytes.Equal(first.SBOMDocument(), second.SBOMDocument()) {
		t.Fatal("execution records are not deterministic")
	}
	var manifest struct {
		Schema, ManifestID string
		Primary            string
		Entries            []struct {
			Node     string
			Evidence []string
		}
		Edges     []json.RawMessage
		Execution struct {
			Command, Path, SHA256, RuntimeIdentity, SourceFingerprint, PolicyFingerprint string
			Size                                                                         uint64
		}
	}
	// Decode snake_case fields explicitly while retaining all record members for
	// canonical checksum and forbidden-field checks below.
	var document map[string]json.RawMessage
	if err := json.Unmarshal(first.ManifestDocument(), &document); err != nil {
		t.Fatal(err)
	}
	if _, present := document["target"]; present {
		t.Fatal("execution invented an installation target")
	}
	if err := json.Unmarshal(first.ManifestDocument(), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != "helox.verified-execution-manifest/v1" || manifest.Primary != "first" || len(manifest.Entries) != 2 || len(manifest.Edges) != 1 || len(manifest.Entries[0].Evidence) == 0 || len(manifest.Entries[1].Evidence) == 0 {
		t.Fatal("whole graph or current evidence coverage missing")
	}
	var entry map[string]json.RawMessage
	if err := json.Unmarshal(document["execution"], &entry); err != nil {
		t.Fatal(err)
	}
	for key, expected := range map[string]string{"command": data.Command, "path": data.EntryPath, "sha256": data.EntryDigest.String(), "runtime_identity": data.RuntimeIdentity, "source_fingerprint": data.Resolution.LockfileDigest().String()} {
		var actual string
		if json.Unmarshal(entry[key], &actual) != nil || actual != expected {
			t.Fatal("entry binding lost", key)
		}
	}
	var size uint64
	if json.Unmarshal(entry["size"], &size) != nil || size != data.EntrySize {
		t.Fatal("entry byte size missing")
	}
	var policy string
	if json.Unmarshal(entry["policy_fingerprint"], &policy) != nil || len(policy) != 64 {
		t.Fatal("actual policy fingerprint missing")
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(first.ManifestDocument()))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		t.Fatal(err)
	}
	if object["manifest_id"] != first.ManifestID().String() {
		t.Fatal("manifest identity missing")
	}
	delete(object, "manifest_id")
	unsigned, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(unsigned)
	if hex.EncodeToString(sum[:]) != first.ManifestID().String() {
		t.Fatal("canonical manifest checksum differs from document")
	}
	var sbom struct {
		Components   []json.RawMessage `json:"components"`
		Dependencies []json.RawMessage `json:"dependencies"`
	}
	if json.Unmarshal(first.SBOMDocument(), &sbom) != nil || len(sbom.Components) != 2 || len(sbom.Dependencies) != 2 || !bytes.Contains(first.SBOMDocument(), []byte(first.ManifestID().String())) {
		t.Fatal("existing whole-set SBOM or manifest binding lost")
	}
	if bytes.Contains(first.ManifestDocument(), []byte("private-argument-fixture")) || bytes.Contains(first.SBOMDocument(), []byte("private-argument-fixture")) {
		t.Fatal("private argv persisted")
	}
	data.Arguments = []string{"different-private-argument"}
	other := executionRecords(t, data)
	if first.ManifestID() != other.ManifestID() || !bytes.Equal(first.ManifestDocument(), other.ManifestDocument()) || !bytes.Equal(first.SBOMDocument(), other.SBOMDocument()) {
		t.Fatal("program argv or its hash became storage identity")
	}
}

func TestExecutionRecordIdentityTracksSourcePolicyAndExactEntry(t *testing.T) {
	data := executionRecordsFixture(t)
	first := executionRecords(t, data)
	for _, change := range []string{"source", "policy", "runtime", "command", "member", "size", "digest"} {
		t.Run(change, func(t *testing.T) {
			modified := data
			switch change {
			case "source":
				lock, _ := domain.NewSHA256Digest(strings.Repeat("e", 64))
				modified.Resolution, _ = domain.NewDependencyResolution(data.Resolution.Graph(), data.Resolution.RuntimeIdentity(), lock)
			case "policy":
				decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, data.Set.Decision().PolicyID(), 2, []string{"ALLOW"})
				modified.Set, _ = domain.NewVerifiedSet(data.Set.Inspected(), decision)
			case "runtime":
				modified.RuntimeIdentity = "different-runtime"
			case "command":
				modified.Command = "another"
			case "member":
				modified.EntryPath = "other.js"
			case "size":
				modified.EntrySize++
			case "digest":
				modified.EntryDigest, _ = domain.NewSHA256Digest(strings.Repeat("f", 64))
			}
			if executionRecords(t, modified).ManifestID() == first.ManifestID() {
				t.Fatal("changed binding retained manifest identity")
			}
		})
	}
}

func TestExecutionRecordsRejectMissingAndCanceledContext(t *testing.T) {
	execution, err := domain.NewExecutionContext(executionRecordsFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	generator := evidence.Generator{}
	for _, input := range []struct {
		name      string
		ctx       context.Context
		execution domain.ExecutionContext
	}{
		{"nil context", nil, execution},
		{"zero execution", context.Background(), domain.ExecutionContext{}},
	} {
		if bundle, err := generator.BuildExecution(input.ctx, input.execution); err == nil || bundle.Valid() {
			t.Fatal("missing input accepted", input.name)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if bundle, err := generator.BuildExecution(ctx, execution); !errors.Is(err, context.Canceled) || bundle.Valid() {
		t.Fatal("cancellation was not preserved", err)
	}
}
