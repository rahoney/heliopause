package cargo

import (
	"strings"
	"testing"
)

const checksumA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const checksumB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestReferenceAndDownloadURLArePinned(t *testing.T) {
	reference, err := ParseReference("serde@1.0.200")
	if err != nil || reference.Source() != Source() {
		t.Fatalf("reference = %#v, error = %v", reference, err)
	}
	urlValue, err := DownloadURL("serde", "1.0.200")
	if err != nil || urlValue != "https://static.crates.io/crates/serde/serde-1.0.200.crate" {
		t.Fatalf("URL = %q, error = %v", urlValue, err)
	}
	for _, value := range []string{"serde", "serde@latest", "serde@1.0", "serde@1.0.200@evil"} {
		if _, err := ParseReference(value); err == nil {
			t.Fatalf("accepted invalid crate reference %q", value)
		}
	}
}

func TestCargoMetadataRejectsGitAndPreservesChecksumGraph(t *testing.T) {
	body := `{"packages":[{"id":"registry+https://github.com/rust-lang/crates.io-index#serde@1.0.200","name":"serde","version":"1.0.200","source":"registry+https://github.com/rust-lang/crates.io-index","checksum":"` + checksumA + `"},{"id":"registry+https://github.com/rust-lang/crates.io-index#serde_derive@1.0.100","name":"serde_derive","version":"1.0.100","source":"registry+https://github.com/rust-lang/crates.io-index","checksum":"` + checksumB + `"}],"resolve":{"nodes":[{"id":"registry+https://github.com/rust-lang/crates.io-index#serde@1.0.200","deps":[{"pkg":"registry+https://github.com/rust-lang/crates.io-index#serde_derive@1.0.100"}]},{"id":"registry+https://github.com/rust-lang/crates.io-index#serde_derive@1.0.100","deps":[]}]}}`
	records, edges, err := ParseMetadata([]byte(body))
	if err != nil || len(records) != 2 {
		t.Fatalf("records = %#v, edges = %s, error = %v", records, edges, err)
	}
	reference, _ := ParseReference("serde@1.0.200")
	graph, err := BuildLockedGraph(reference, records, edges)
	if err != nil || len(graph.Nodes()) != 2 || len(graph.Edges()) != 1 {
		t.Fatalf("graph = %#v, error = %v", graph, err)
	}
	for _, node := range graph.Nodes() {
		if node.Artifact().Identity().Source() != Source() || !strings.HasPrefix(node.Artifact().DeclaredIntegrity(), "sha256=") {
			t.Fatalf("node lost checksum/source: %#v", node)
		}
	}
	unsafe := strings.Replace(body, `"source":"registry+https://github.com/rust-lang/crates.io-index"`, `"source":"git+https://evil.example/repo"`, 1)
	if _, _, err := ParseMetadata([]byte(unsafe)); err == nil {
		t.Fatal("accepted git dependency")
	}
}
