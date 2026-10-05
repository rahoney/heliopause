package cargo

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func cargoActualMetadataFixture(t *testing.T) ([]byte, []byte) {
	t.Helper()
	body, err := os.ReadFile("testdata/metadata-v1-rust-1.99.0.json")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("testdata/metadata-v1-rust-1.99.0.lock")
	if err != nil {
		t.Fatal(err)
	}
	return body, lock
}

func TestLockedMetadataActualOutputAndStableGraph(t *testing.T) {
	body, lock := cargoActualMetadataFixture(t)
	records, _, state, err := ParseLockedMetadata(body, lock, "/fixture")
	if err != nil || len(records) != 1 || records[0].Name != "itoa" || records[0].Checksum != "92ecc6618181def0457392ccd0ee51198e065e016d1d527a7ac1b6dc7c1f09d2" {
		t.Fatalf("actual locked metadata: count=%d err=%v", len(records), err)
	}
	if bytes.Contains(state, []byte("/fixture")) {
		t.Fatal("canonical graph retained a private absolute path")
	}
	relocated := []byte(strings.ReplaceAll(string(body), "/fixture", "/private/second-project"))
	_, _, relocatedState, err := ParseLockedMetadata(relocated, lock, "/private/second-project")
	if err != nil || !bytes.Equal(state, relocatedState) {
		t.Fatalf("private workspace relocation changed graph: %v", err)
	}
	var document projectMetadata
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	document.Resolve.Nodes[1].Features = []string{"no-panic"}
	changed, _ := json.Marshal(document)
	_, _, featureState, err := ParseLockedMetadata(changed, lock, "/fixture")
	if err != nil || bytes.Equal(state, featureState) {
		t.Fatalf("selected features not bound: %v", err)
	}
	target := "cfg(unix)"
	document.Resolve.Nodes[0].Deps[0].Kinds[0].Target = &target
	changed, _ = json.Marshal(document)
	_, _, targetState, err := ParseLockedMetadata(changed, lock, "/fixture")
	if err != nil || bytes.Equal(featureState, targetState) {
		t.Fatalf("dependency target not bound: %v", err)
	}
}

func TestLockedMetadataRejectsUncertainGraphAndLock(t *testing.T) {
	body, lock := cargoActualMetadataFixture(t)
	for _, tc := range []struct {
		name string
		edit func(*projectMetadata)
	}{
		{"unknown schema", func(d *projectMetadata) { d.Version = 2 }},
		{"wrong workspace", func(d *projectMetadata) { d.WorkspaceRoot = "/other" }},
		{"missing graph", func(d *projectMetadata) { d.Resolve.Nodes = nil }},
		{"disconnected selected package", func(d *projectMetadata) { d.Resolve.Nodes[0].Deps = nil }},
		{"duplicate package", func(d *projectMetadata) { d.Packages[1].ID = d.Packages[0].ID }},
		{"duplicate node", func(d *projectMetadata) { d.Resolve.Nodes[1].ID = d.Resolve.Nodes[0].ID }},
		{"unknown dependency", func(d *projectMetadata) { d.Resolve.Nodes[0].Deps[0].Pkg = "unknown" }},
		{"local escape", func(d *projectMetadata) { d.Packages[0].ManifestPath = "/outside/Cargo.toml" }},
		{"invalid member", func(d *projectMetadata) { d.WorkspaceMembers[0] = d.Packages[1].ID }},
		{"registry swap", func(d *projectMetadata) {
			value := "registry+https://evil.example/index"
			d.Packages[1].Source = &value
		}},
		{"git source", func(d *projectMetadata) { value := "git+https://evil.example/repo"; d.Packages[1].Source = &value }},
		{"checksum disagreement", func(d *projectMetadata) { value := strings.Repeat("a", 64); d.Packages[1].Checksum = &value }},
		{"duplicate feature", func(d *projectMetadata) { d.Resolve.Nodes[1].Features = []string{"x", "x"} }},
		{"unknown dependency kind", func(d *projectMetadata) { value := "arbitrary"; d.Resolve.Nodes[0].Deps[0].Kinds[0].Kind = &value }},
		{"duplicate edge", func(d *projectMetadata) {
			d.Resolve.Nodes[0].Deps = append(d.Resolve.Nodes[0].Deps, d.Resolve.Nodes[0].Deps[0])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var document projectMetadata
			if err := json.Unmarshal(body, &document); err != nil {
				t.Fatal(err)
			}
			tc.edit(&document)
			changed, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			if records, edges, state, err := ParseLockedMetadata(changed, lock, "/fixture"); err == nil || records != nil || edges != nil || state != nil {
				t.Fatal("accepted uncertain graph or returned partial success")
			}
		})
	}
	for name, changed := range map[string][]byte{
		"unknown lock version":       []byte(strings.Replace(string(lock), "version = 4", "version = 2", 1)),
		"missing checksum":           []byte(strings.Replace(string(lock), `checksum = "92ecc6618181def0457392ccd0ee51198e065e016d1d527a7ac1b6dc7c1f09d2"`, "", 1)),
		"duplicate key":              append(append([]byte(nil), lock...), []byte("checksum=\"duplicate\"\n")...),
		"duplicate package":          append(append([]byte(nil), lock...), []byte("[[package]]\nname=\"haa_schema_fixture\"\nversion=\"0.1.0\"\n")...),
		"unknown lock edge":          []byte(strings.Replace(string(lock), `["itoa"]`, `["missing"]`, 1)),
		"missing selected lock edge": []byte(strings.Replace(string(lock), `["itoa"]`, `[]`, 1)),
		"lock registry substitution": []byte(strings.Replace(string(lock), registrySourceURL, "registry+https://evil.example/index", 1)),
		"control bound":              bytes.Repeat([]byte(" "), MaxProjectControlBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := ParseLockedMetadata(body, changed, "/fixture"); err == nil {
				t.Fatal("accepted invalid frozen lock")
			}
		})
	}
	for name, changed := range map[string][]byte{
		"trailing object":       append(append([]byte(nil), body...), []byte("{}")...),
		"duplicate JSON key":    []byte(strings.Replace(string(body), `"source":null`, `"source":null,"source":null`, 1)),
		"deep ignored metadata": []byte(`{"extra":` + strings.Repeat("[", 40) + "0" + strings.Repeat("]", 40) + `}`),
		"JSON byte bound":       bytes.Repeat([]byte(" "), maxMetadataBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := ParseLockedMetadata(changed, lock, "/fixture"); err == nil {
				t.Fatal("accepted ambiguous or excessive JSON")
			}
		})
	}
}
