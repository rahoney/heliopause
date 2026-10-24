package releaseprovenance

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func actualPNPMRelease(t *testing.T) ([]byte, PNPMReleaseIdentity) {
	t.Helper()
	read := func(name string) []byte {
		body, err := os.ReadFile(filepath.Join("testdata", "pnpm-12.15.0", name))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	var identity PNPMReleaseIdentity
	if err := json.Unmarshal(read("identity.json"), &identity); err != nil {
		t.Fatal(err)
	}
	return read("bundle.json"), identity
}

func TestPNPMOfficialReleaseProvenance(t *testing.T) {
	proof, identity := actualPNPMRelease(t)
	receipt, err := VerifyPNPMRelease(context.Background(), proof, identity)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Identity() != identity || receipt.BundleSHA256() != uvDigest(proof) || receipt.Invocation() != pnpmRepository+"/actions/runs/39583314875/attempts/1" {
		t.Fatalf("unexpected provenance receipt: %+v", receipt)
	}
	_, haa, subjects := actualSignedRelease(t)
	if _, err := Verify(context.Background(), proof, haa, subjects); err == nil {
		t.Fatal("pnpm proof gained HAA publisher authority")
	}
	uv, uvIdentity := actualUVImage(t)
	if _, err := VerifyUVImage(context.Background(), proof, uv[1], uv[2], uv[3], uvIdentity); err == nil {
		t.Fatal("pnpm proof gained UV publisher authority")
	}
}

func TestPNPMRejectsUnauthenticatedRelease(t *testing.T) {
	proof, identity := actualPNPMRelease(t)
	change := func(fn func(*PNPMReleaseIdentity)) PNPMReleaseIdentity {
		result := identity
		fn(&result)
		return result
	}
	mutate := func(fn func(map[string]any)) []byte {
		var value map[string]any
		if err := json.Unmarshal(proof, &value); err != nil {
			t.Fatal(err)
		}
		fn(value)
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	haa, _, _ := actualSignedRelease(t)
	uv, _ := actualUVImage(t)
	cases := []struct {
		name     string
		proof    []byte
		identity PNPMReleaseIdentity
	}{
		{"wrong-haa-publisher", haa, identity},
		{"wrong-uv-publisher", uv[0], identity},
		{"source", proof, change(func(v *PNPMReleaseIdentity) { v.SourceCommit = strings.Repeat("a", 40) })},
		{"run", proof, change(func(v *PNPMReleaseIdentity) { v.WorkflowRun = "39583314876" })},
		{"version", proof, change(func(v *PNPMReleaseIdentity) { v.Version = "12.15.1" })},
		{"prerelease", proof, change(func(v *PNPMReleaseIdentity) { v.Version += "-rc1" })},
		{"build", proof, change(func(v *PNPMReleaseIdentity) { v.Version += "+build" })},
		{"prefixed-version", proof, change(func(v *PNPMReleaseIdentity) { v.Version = "v" + v.Version })},
		{"archive", proof, change(func(v *PNPMReleaseIdentity) { v.ArchiveSHA256 = strings.Repeat("b", 64) })},
		{"missing", nil, identity},
		{"oversize", make([]byte, maxBundle+1), identity},
		{"trailing-json", append(append([]byte{}, proof...), []byte("\n{}")...), identity},
		{"no-tlog", mutate(func(v map[string]any) { delete(v["verificationMaterial"].(map[string]any), "tlogEntries") }), identity},
		{"tampered-statement", mutate(func(v map[string]any) {
			e := v["dsseEnvelope"].(map[string]any)
			payload, err := base64.StdEncoding.DecodeString(e["payload"].(string))
			if err != nil {
				t.Fatal(err)
			}
			e["payload"] = base64.StdEncoding.EncodeToString(append(payload, ' '))
		}), identity},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := VerifyPNPMRelease(context.Background(), tt.proof, tt.identity); err == nil {
				t.Fatal("invalid release accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := VerifyPNPMRelease(ctx, proof, identity); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}

func TestPNPMPredicateExactBuildBinding(t *testing.T) {
	proof, identity := actualPNPMRelease(t)
	var envelope struct{ DSSEEnvelope struct{ Payload string } }
	if err := json.Unmarshal(proof, &envelope); err != nil {
		t.Fatal(err)
	}
	body, err := base64.StdEncoding.DecodeString(envelope.DSSEEnvelope.Payload)
	if err != nil {
		t.Fatal(err)
	}
	var statement struct{ Predicate json.RawMessage }
	if err := json.Unmarshal(body, &statement); err != nil {
		t.Fatal(err)
	}
	signer := pnpmRepository + "/" + pnpmWorkflow + "@refs/tags/v" + identity.Version
	invocation := pnpmRepository + "/actions/runs/" + identity.WorkflowRun + "/attempts/1"
	if !validPNPMPredicate(statement.Predicate, identity, invocation, signer) {
		t.Fatal("official build rejected")
	}
	for _, value := range [][2]string{
		{"self-hosted", "github-hosted"}, {"50565430", "50565431"}, {"21320719", "21320720"},
		{"push", "pull_request"}, {identity.SourceCommit, strings.Repeat("a", 40)},
		{invocation, invocation + "0"}, {"release.yml", "untrusted.yml"},
		{"https://actions.github.io/buildtypes/workflow/v1", "https://example.invalid/build"},
	} {
		mutated := strings.ReplaceAll(string(statement.Predicate), value[0], value[1])
		if mutated == string(statement.Predicate) || validPNPMPredicate([]byte(mutated), identity, invocation, signer) {
			t.Fatalf("build binding not enforced for %q", value[0])
		}
	}
}
