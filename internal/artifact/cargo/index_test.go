package cargo

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCargoIndexExactIdentityAndSource(t *testing.T) {
	for name, want := range map[string]string{"a": "1/a", "Ab": "2/ab", "Foo": "3/f/foo", "My_Crate": "my/_c/my_crate"} {
		got, err := IndexURL(name)
		if err != nil || got != "https://index.crates.io/"+want {
			t.Fatalf("index layout %q: %q %v", name, got, err)
		}
	}
	body, err := os.ReadFile("testdata/itoa-index.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	checksum, err := ParseIndexChecksum(body, "itoa", "1.0.17")
	if err != nil || checksum != "92ecc6618181def0457392ccd0ee51198e065e016d1d527a7ac1b6dc7c1f09d2" {
		t.Fatalf("actual independent index: %q %v", checksum, err)
	}
	if _, err := ParseIndexChecksum(body, "ItOA", "1.0.17"); err == nil {
		t.Fatal("index path folding changed package identity")
	}
	if _, err := ParseIndexChecksum(body, "itoa", "99.0.0"); err == nil {
		t.Fatal("absent exact version accepted")
	}
	for _, config := range []string{
		`{"dl":"https://evil.invalid/crates"}`,
		`{"dl":"https://static.crates.io/crates","auth-required":true}`,
		`{"dl":"https://static.crates.io/crates","api":"https://evil.invalid"}`,
		`{"dl":"https://static.crates.io/crates","dl":"https://evil.invalid"}`,
		`{"dl":"https://static.crates.io/crates/{crate}/{version}"}`,
		`{"dl":"https://static.crates.io/crates"}{}`,
	} {
		if ValidateRegistryConfig([]byte(config)) == nil {
			t.Fatal("substituted or ambiguous config accepted")
		}
	}
	if err := ValidateRegistryConfig([]byte(`{"dl":"https://static.crates.io/crates","api":"https://crates.io"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestCargoIndexRejectsAmbiguousOrExcessiveDeclarations(t *testing.T) {
	valid := `{"name":"fixture","vers":"1.2.3","cksum":"` + strings.Repeat("a", 64) + `"}`
	for name, body := range map[string][]byte{
		"duplicate version":   []byte(valid + "\n" + valid),
		"build alias":         []byte(valid + "\n" + strings.Replace(valid, "1.2.3", "1.2.3+other", 1)),
		"duplicate field":     []byte(strings.Replace(valid, `"vers":"1.2.3"`, `"vers":"1.2.3","vers":"1.2.3"`, 1)),
		"unknown schema":      []byte(strings.Replace(valid, `"name":`, `"v":3,"name":`, 1)),
		"uppercase digest":    []byte(strings.Replace(valid, strings.Repeat("a", 64), strings.Repeat("A", 64), 1)),
		"malformed digest":    []byte(strings.Replace(valid, strings.Repeat("a", 64), "invalid", 1)),
		"trailing JSON":       []byte(valid + "{}"),
		"empty internal line": []byte(valid + "\n\n" + strings.Replace(valid, "1.2.3", "1.2.4", 1)),
		"byte cap":            bytes.Repeat([]byte(" "), MaxIndexBytes+1),
		"line cap":            bytes.Repeat([]byte("a\n"), 16385),
		"record cap":          bytes.Repeat([]byte(" "), (128<<10)+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseIndexChecksum(body, "fixture", "1.2.3"); err == nil {
				t.Fatal("ambiguous or excessive registry record accepted")
			}
		})
	}
}
