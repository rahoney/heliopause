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
	if len(facts) != 190 {
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
