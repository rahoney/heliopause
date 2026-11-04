//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestSemgrepWireRetainsAllPayloadsAndRejectsTruncatedOrChangedBytes(t *testing.T) {
	legacy, files := wireFixture(t)
	var value map[string]any
	if err := json.Unmarshal(legacy.body, &value); err != nil {
		t.Fatal(err)
	}
	assets := value["assets"].([]any)
	roles := map[string]string{runtimeidentity.PipxRuntimeAssetName: "pipx-runtime", "pipx-NOTICE.txt": "release-metadata", runtimeidentity.SandboxResponderAssetName: "sandbox-responder"}
	pin := runtimeidentity.SelectedSemgrepSupply()
	for _, asset := range runtimeidentity.SemgrepReleaseAssets(pin) {
		role := "semgrep-data"
		if asset.Path == pin.Tool.Path {
			role = "semgrep"
		}
		roles[asset.Path] = role
	}
	for name, role := range roles {
		data := []byte("unsigned-copy-only-" + name)
		files[name] = data
		sum := sha256.Sum256(data)
		assets = append(assets, map[string]any{"name": name, "platform": "linux/amd64", "role": role, "sha256": hex.EncodeToString(sum[:]), "size": len(data)})
	}
	value["assets"] = assets
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	release, err := newWireRelease(body, legacy.proofs)
	if err != nil || len(release.files) != 17 {
		t.Fatal("incomplete wire set", err)
	}
	for _, mode := range []string{"complete", "truncated", "changed", "extra"} {
		t.Run(mode, func(t *testing.T) {
			var stream bytes.Buffer
			if err := writeWirePreamble(context.Background(), &stream, release); err != nil {
				t.Fatal(err)
			}
			err := writeWirePayloads(context.Background(), &stream, release, func(name string) (io.ReadCloser, error) {
				data := bytes.Clone(files[name])
				if name == pin.Sources.Path {
					switch mode {
					case "truncated":
						data = data[:len(data)-1]
					case "changed":
						data[0] ^= 1
					case "extra":
						data = append(data, '!')
					}
				}
				return io.NopCloser(bytes.NewReader(data)), nil
			})
			if mode != "complete" {
				if err == nil {
					t.Fatal("changed supply stream accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := readWirePreamble(context.Background(), &stream)
			if err != nil || len(decoded.files) != 17 {
				t.Fatal("wire declaration lost payloads", err)
			}
			for _, file := range decoded.files {
				data := make([]byte, file.Size)
				if _, err := io.ReadFull(&stream, data); err != nil || !bytes.Equal(data, files[file.Name]) {
					t.Fatal("wire payload changed", file.Name, err)
				}
			}
			if stream.String() != wireStagedMarker {
				t.Fatal("wire boundary changed")
			}
		})
	}
}
