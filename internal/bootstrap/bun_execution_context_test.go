package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// This fixture checks composition only, not archive authentication or approval.
func bunProgramContextEntry(t *testing.T, subject domain.AcquiredArtifact, file, body string) artifactnpm.BunExecutionEntry {
	t.Helper()
	manifest, err := json.Marshal(map[string]string{"name": subject.Identity().Name(), "version": subject.Identity().Version(), "bin": file})
	if err != nil {
		t.Fatal(err)
	}
	digest := func(body []byte) domain.ContentDigest {
		sum := sha256.Sum256(body)
		d, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	inventory, err := artifactnpm.NewArchiveInventory(subject, []artifactnpm.ArchiveEntry{
		{Path: "package.json", Size: int64(len(manifest)), Mode: 0o644, Digest: digest(manifest)},
		{Path: file, Size: int64(len(body)), Mode: 0o755, Digest: digest([]byte(body))},
	}, manifest)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifactnpm.SelectExecutionEntry(inventory, "")
	if err != nil {
		t.Fatal(err)
	}
	proof, err := entry.BindBunTarget([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return proof
}

func TestBunProgramContextFollowsMeasuredDeclarationAndExplicitForce(t *testing.T) {
	operation, resolution, set, original := nodeContextFixture(t)
	for _, test := range []struct {
		name, file, body     string
		force, accepted, bun bool
	}{
		{"Node default", "cli.js", "#!/usr/bin/env node\n", false, true, false},
		{"Bun default", "cli.ts", "#!/usr/bin/env bun\nconst x: number = 1;\n", false, true, true},
		{"forced Node entry", "cli.js", "#!/usr/bin/env node\n", true, true, true},
		{"forced TS", "cli.ts", "const x: number = 1;\n", true, true, true},
		{"no guessed TS runtime", "cli.ts", "const x: number = 1;\n", false, false, false},
		{"Node declaration in TS", "cli.ts", "#!/usr/bin/env node\n", false, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			proof := bunProgramContextEntry(t, original.Entry().Artifact(), test.file, test.body)
			arguments := []string{"--bun", "literal \n", ""}
			current, err := newBunProgramExecutionContext(operation, resolution, set, proof, arguments, test.force)
			if !test.accepted {
				if err == nil || current.Valid() {
					t.Fatal("runtime guessed without a declaration or explicit force")
				}
				return
			}
			want := runtimeidentity.NodeImageReference
			if test.bun {
				want = sandbox.BunExecutionRuntimeIdentity()
			}
			if err != nil || !current.Valid() || current.RuntimeIdentity() != want || current.EntryArtifact() != original.Entry().Artifact() || current.EntryPath() != test.file || current.EntryDigest() != proof.Entry().Target().Digest || current.OperationID() != operation || current.Arguments()[0] != "--bun" {
				t.Fatal("runtime selection changed original facts or treated program argv as force", err)
			}
		})
	}
	if result, err := newBunProgramExecutionContext(operation, resolution, set, artifactnpm.BunExecutionEntry{}, nil, true); err == nil || result.Valid() {
		t.Fatal("zero Bun proof produced an execution context")
	}
}

func TestBunProgramContextUsesSameApprovedOriginalWithDistinctRuntime(t *testing.T) {
	operation, resolution, set, proof := nodeContextFixture(t)
	node, err := newNodeExecutionContext(operation, resolution, set, proof, nil)
	if err != nil {
		t.Fatal(err)
	}
	bunProof, err := proof.Entry().BindBunTarget(proof.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	bun, err := newBunProgramExecutionContext(operation, resolution, set, bunProof, nil, true)
	if err != nil || !bun.Valid() || bun.RuntimeIdentity() == node.RuntimeIdentity() || bun.EntryArtifact() != node.EntryArtifact() || bun.EntryPath() != node.EntryPath() || bun.EntryDigest() != node.EntryDigest() || bun.EntrySize() != node.EntrySize() || bun.SourceFingerprint() != node.SourceFingerprint() || bun.OperationID() != node.OperationID() {
		t.Fatal("Bun context replaced source facts or reused the Node runtime identity", err)
	}
}
