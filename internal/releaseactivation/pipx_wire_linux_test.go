//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestPipxDataWirePreservesExactTwelvePayloadsAndRejectsTampering(t *testing.T) {
	legacy, files := wireFixture(t)
	var metadata map[string]any
	if err := json.Unmarshal(legacy.body, &metadata); err != nil {
		t.Fatal(err)
	}
	notices, err := os.ReadFile(filepath.Join("..", "..", "tools", "pipx", "NOTICE"))
	if err != nil {
		t.Fatal(err)
	}
	files[runtimeidentity.PipxRuntimeAssetName] = []byte("unsigned wire DATA only; never imported")
	files["pipx-NOTICE.txt"] = notices
	assets := metadata["assets"].([]any)
	for name, role := range map[string]string{runtimeidentity.PipxRuntimeAssetName: "pipx-runtime", "pipx-NOTICE.txt": "release-metadata"} {
		body := files[name]
		sum := sha256.Sum256(body)
		assets = append(assets, map[string]any{"name": name, "platform": "linux/amd64", "role": role, "size": len(body), "sha256": hex.EncodeToString(sum[:])})
	}
	metadata["assets"] = assets
	body, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	release, err := newWireRelease(body, legacy.proofs)
	if err != nil {
		t.Fatal(err)
	}
	if len(release.files) != 12 {
		t.Fatal("DATA pair omitted from wire")
	}
	native, err := release.manifest.NativeUpdateFiles()
	if err != nil || len(native) != 6 {
		t.Fatal("DATA changed native role set", err)
	}
	var stream bytes.Buffer
	if err := writeWirePreamble(context.Background(), &stream, release); err != nil {
		t.Fatal(err)
	}
	open := func(name string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(files[name])), nil }
	if err := writeWirePayloads(context.Background(), &stream, release, open); err != nil {
		t.Fatal(err)
	}
	decoded, err := readWirePreamble(context.Background(), &stream)
	if err != nil || len(decoded.files) != 12 {
		t.Fatal("wire did not retain complete set", err)
	}
	for _, file := range decoded.files {
		body := make([]byte, file.Size)
		if _, err := io.ReadFull(&stream, body); err != nil || !bytes.Equal(body, files[file.Name]) {
			t.Fatal("payload order/bytes changed", err)
		}
	}
	if stream.String() != wireStagedMarker {
		t.Fatal("payload boundary changed")
	}
	current := InstalledRelease{record: registration{Schema: registrationSchema, Version: legacy.manifest.Version()}, manifest: legacy.manifest, proof: []byte("test-current")}
	if _, err := decoded.verify(context.Background(), current); !errors.Is(err, ErrUntrusted) {
		t.Fatal("unsigned DATA set gained authenticity", err)
	}
	for _, mode := range []string{"missing", "extra", "changed"} {
		err := writeWirePayloads(context.Background(), io.Discard, release, func(name string) (io.ReadCloser, error) {
			data := append([]byte(nil), files[name]...)
			if name == runtimeidentity.PipxRuntimeAssetName {
				switch mode {
				case "missing":
					data = nil
				case "extra":
					data = append(data, '!')
				case "changed":
					data[0] ^= 1
				}
			}
			return io.NopCloser(bytes.NewReader(data)), nil
		})
		if err == nil {
			t.Fatal("changed DATA payload accepted", mode)
		}
	}
}
