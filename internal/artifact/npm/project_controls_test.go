package npm

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestPublicProjectControlsKeepRangesAndRejectUnsafeSelection(t *testing.T) {
	manifest := []byte(`{"name":"app","dependencies":{"primary":"^1.0.0"}}`)
	lock := []byte(fmt.Sprintf(`{"lockfileVersion":3,"packages":{"":{"dependencies":{"primary":"^1.0.0"}},"node_modules/primary":{"version":"1.2.0","resolved":"https://registry.npmjs.org/primary/-/primary-1.2.0.tgz","integrity":"sha512-%s"}}}`, base64.StdEncoding.EncodeToString(make([]byte, 64))))
	ref, err := ProjectRootReference(manifest)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ParseProjectControls(ref, manifest, lock)
	if err != nil || len(graph.Nodes()) != 1 || graph.Nodes()[0].Artifact().Identity().Version() != "1.2.0" {
		t.Fatal(graph, err)
	}
	if !SameSelectionGraph(graph, graph) {
		t.Fatal("same complete graph rejected")
	}
	for _, bad := range []string{
		`{"dependencies":{"primary":"file:.heliopause/artifacts/a.tgz"}}`,
		`{"dependencies":{"primary":"https://example.test/a.tgz"}}`,
		`{"dependencies":{"primary":"npm:other@1"}}`,
		`{"dependencies":{"primary":"^1"},"dependencies":{"primary":"^2"}}`,
		`{"dependencies":{"primary":"^1.0.0","other":"*"}}`,
		`{"dependencies":{"primary":"^1"},"devDependencies":{"other":"*"}}`,
		`{"dependencies":{"primary":"^1"},"workspaces":["packages/*"]}`,
		`{"dependencies":{"primary":"^1"}} {}`,
		strings.Repeat(" ", ProjectControlLimit+1),
	} {
		if _, err := ProjectRootReference([]byte(bad)); err == nil {
			t.Fatalf("unsupported original controls accepted: %.100s", bad)
		}
	}
	if _, err := ParseProjectControls(ref, []byte(strings.Replace(string(manifest), "^1.0.0", "^2.0.0", 1)), lock); err == nil {
		t.Fatal("changed root constraints accepted")
	}
	if _, err := ParseProjectControls(ref, manifest, []byte(strings.Replace(string(lock), `"lockfileVersion":3`, `"lockfileVersion":2,"lockfileVersion":3`, 1))); err == nil {
		t.Fatal("duplicate lock key accepted")
	}
}
