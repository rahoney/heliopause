package releaseprovenance

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func actualSignedRelease(t *testing.T) ([]byte, Identity, []Subject) {
	t.Helper()
	// Committed portable evidence from the actual protected v0.1.0 release
	// build. No local secret, private fixture or network lookup is required.
	directory := filepath.Join("..", "..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0")
	body, err := os.ReadFile(filepath.Join(directory, "actual-signed-bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(directory, "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(manifest)
	return body, Identity{"v0.1.0", "ef5ef0e0e0437a9083133674ef8b9e6d693b83a9", "39036210335"}, []Subject{{"helox-release-manifest.json", hex.EncodeToString(sum[:])}}
}

func TestActualReleaseAttestationVerifiesWithEmbeddedOfflineRoot(t *testing.T) {
	body, identity, subjects := actualSignedRelease(t)
	receipt, err := Verify(context.Background(), body, identity, subjects)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Identity() != identity || receipt.Invocation() != repository+"/actions/runs/39036210335/attempts/1" || len(receipt.Subjects()) != 1 || receipt.Subjects()[0] != subjects[0] {
		t.Fatalf("unexpected provenance receipt: %+v", receipt)
	}
	sum := sha256.Sum256(body)
	if receipt.BundleSHA256() != hex.EncodeToString(sum[:]) {
		t.Fatal("receipt is not bound to exact attestation bytes")
	}
	subjects[0].Name = "changed-request"
	copy := receipt.Subjects()
	copy[0].Name = "changed-result"
	if receipt.Subjects()[0].Name != "helox-release-manifest.json" {
		t.Fatal("caller mutated authenticated subjects")
	}
	rootSum := sha256.Sum256(publicRoot)
	if hex.EncodeToString(rootSum[:]) != "3c2cc7f357dc064ec527fdcd78da6e9245c21a381e1abaa0f2b62b186bcac1a1" {
		t.Fatal("embedded root differs from reviewed public-good record")
	}
}

func TestActualReleaseAttestationRejectsSubstitutionAndMalformedProof(t *testing.T) {
	body, identity, subjects := actualSignedRelease(t)
	mutateBundle := func(change func(map[string]any)) []byte {
		t.Helper()
		var value map[string]any
		if err := json.Unmarshal(body, &value); err != nil {
			t.Fatal(err)
		}
		change(value)
		result, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	cases := []struct {
		name     string
		identity Identity
		subjects []Subject
		body     []byte
	}{
		{"wrong-source", Identity{identity.Version, strings.Repeat("a", 40), identity.WorkflowRun}, subjects, body},
		{"wrong-tag", Identity{"v0.1.1", identity.SourceCommit, identity.WorkflowRun}, subjects, body},
		{"wrong-run", Identity{identity.Version, identity.SourceCommit, "39036210336"}, subjects, body},
		{"wrong-digest", identity, []Subject{{subjects[0].Name, strings.Repeat("a", 64)}}, body},
		{"name-is-part-of-subject", identity, []Subject{{"wrong-manifest.json", subjects[0].SHA256}}, body},
		{"missing-subject", identity, nil, body},
		{"duplicate-requested-subject", identity, append(append([]Subject{}, subjects...), subjects...), body},
		{"trailing-json", identity, subjects, append(append([]byte{}, body...), []byte("\n{}")...)},
		{"oversize", identity, subjects, make([]byte, maxBundle+1)},
		{"no-proof", identity, subjects, []byte(`{}`)},
		{"no-tlog", identity, subjects, mutateBundle(func(v map[string]any) { delete(v["verificationMaterial"].(map[string]any), "tlogEntries") })},
		{"no-certificate", identity, subjects, mutateBundle(func(v map[string]any) { delete(v["verificationMaterial"].(map[string]any), "certificate") })},
		{"changed-payload", identity, subjects, mutateBundle(func(v map[string]any) {
			envelope := v["dsseEnvelope"].(map[string]any)
			decoded, err := base64.StdEncoding.DecodeString(envelope["payload"].(string))
			if err != nil {
				t.Fatal(err)
			}
			decoded = append(decoded, ' ')
			envelope["payload"] = base64.StdEncoding.EncodeToString(decoded)
		})},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			receipt, err := Verify(context.Background(), tt.body, tt.identity, tt.subjects)
			if err == nil || receipt.BundleSHA256() != "" || len(receipt.Subjects()) != 0 {
				t.Fatalf("unverified proof returned a receipt: %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Verify(ctx, body, identity, subjects); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation cause was lost: %v", err)
	}
}

func TestVerifiedPredicateMustMatchCertificateAndRelease(t *testing.T) {
	body, identity, _ := actualSignedRelease(t)
	var outer struct{ DSSEEnvelope struct{ Payload string } }
	if err := json.Unmarshal(body, &outer); err != nil {
		t.Fatal(err)
	}
	encoded, err := base64.StdEncoding.DecodeString(outer.DSSEEnvelope.Payload)
	if err != nil {
		t.Fatal(err)
	}
	var statement struct{ Predicate json.RawMessage }
	if err := json.Unmarshal(encoded, &statement); err != nil {
		t.Fatal(err)
	}
	invocation := repository + "/actions/runs/39036210335/attempts/1"
	signer := repository + "/" + workflow + "@refs/tags/v0.1.0"
	if !validPredicate(statement.Predicate, identity, invocation, signer) {
		t.Fatal("actual predicate rejected")
	}
	for _, tt := range []struct{ name, old, new string }{
		{"builder", signer, "https://example.invalid/builder"},
		{"source", identity.SourceCommit, strings.Repeat("a", 40)},
		{"ref", "refs/tags/v0.1.0", "refs/heads/main"},
		{"runner", "github-hosted", "self-hosted"},
		{"trigger", "push", "workflow_dispatch"},
		{"repo-id", "1331904030", "1331904031"},
		{"owner-id", "168336203", "168336204"},
		{"invocation", invocation, repository + "/actions/runs/39036210335/attempts/2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			changed := strings.ReplaceAll(string(statement.Predicate), tt.old, tt.new)
			if changed == string(statement.Predicate) {
				t.Fatal("negative fixture did not change")
			}
			if validPredicate([]byte(changed), identity, invocation, signer) {
				t.Fatal("substituted predicate accepted")
			}
		})
	}
}
