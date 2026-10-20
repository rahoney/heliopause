//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func wireFixture(t *testing.T) (wireRelease, map[string][]byte) {
	t.Helper()
	base := filepath.Join("..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0")
	read := func(name string) []byte {
		body, err := os.ReadFile(filepath.Join(base, name))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	var value map[string]any
	if err := json.Unmarshal(read("helox-release-manifest.json"), &value); err != nil {
		t.Fatal(err)
	}
	assets := value["assets"].([]any)
	assets = append(assets, map[string]any{"name": "haa-release-installer-linux-amd64", "platform": "linux/amd64", "role": "release-installer"})
	files := map[string][]byte{"helox-release-sbom.cdx.json": read("helox-release-sbom.cdx.json"), "helox-runtime-images.json": read("helox-runtime-images.json")}
	for _, raw := range assets {
		item := raw.(map[string]any)
		name := item["name"].(string)
		if item["platform"] == "linux/amd64" && !strings.HasSuffix(name, ".deb") && !strings.HasSuffix(name, ".rpm") {
			body := []byte("wire-only-" + name)
			sum := sha256.Sum256(body)
			files[name] = body
			item["sha256"] = hex.EncodeToString(sum[:])
			item["size"] = len(body)
		}
	}
	value["assets"] = assets
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	release, err := newWireRelease(body, [][]byte{read("actual-signed-bundle.json")})
	if err != nil {
		t.Fatal(err)
	}
	return release, files
}

func TestWireFreezesExactMembersWithoutGrantingApproval(t *testing.T) {
	release, files := wireFixture(t)
	var stream bytes.Buffer
	if err := writeWirePreamble(context.Background(), &stream, release); err != nil {
		t.Fatal(err)
	}
	if err := writeWirePayloads(context.Background(), &stream, release, func(name string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(files[name])), nil }); err != nil {
		t.Fatal(err)
	}
	decoded, err := readWirePreamble(context.Background(), &stream)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.manifest.SHA256() != release.manifest.SHA256() || len(decoded.files) != 6 {
		t.Fatal("wire metadata or role set changed")
	}
	for _, file := range decoded.files {
		body := make([]byte, file.Size)
		if _, err := io.ReadFull(&stream, body); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(body, files[file.Name]) {
			t.Fatal("canonical role order differs")
		}
	}
	if stream.String() != wireStagedMarker {
		t.Fatal("payload terminator differs")
	}
	current := InstalledRelease{record: registration{Schema: registrationSchema, Version: release.manifest.Version()}, manifest: release.manifest, proof: []byte("test-current")}
	if _, err := decoded.verify(context.Background(), current); !errors.Is(err, ErrUntrusted) {
		t.Fatal("real SDK approved the unsigned synthetic native set", err)
	}
	decoded.body[0] = '!'
	decoded.proofs[0][0] = '!'
	if release.body[0] != '{' || release.proofs[0][0] != '{' {
		t.Fatal("decoded wire aliases sender facts")
	}
}

func TestWireRejectsBadLengthsTruncationAndForeignFields(t *testing.T) {
	release, _ := wireFixture(t)
	var buffer bytes.Buffer
	if err := writeWirePreamble(context.Background(), &buffer, release); err != nil {
		t.Fatal(err)
	}
	valid := buffer.Bytes()
	badMagic := append([]byte(nil), valid...)
	badMagic[0] = '!'
	badLength := append([]byte(nil), valid...)
	binary.BigEndian.PutUint32(badLength[len(updateWireMagic):], 1<<20+1)
	zeroLength := append([]byte(nil), valid...)
	binary.BigEndian.PutUint32(zeroLength[len(updateWireMagic):], 0)
	countOffset := len(updateWireMagic) + 4 + len(release.body)
	badCount := append([]byte(nil), valid...)
	binary.BigEndian.PutUint32(badCount[countOffset:], 17)
	for name, body := range map[string][]byte{"magic": badMagic, "length": badLength, "zero-length": zeroLength, "count": badCount, "short-magic": valid[:3], "short-frame": valid[:len(valid)-1]} {
		t.Run(name, func(t *testing.T) {
			if _, err := readWirePreamble(context.Background(), bytes.NewReader(body)); err == nil {
				t.Fatal("malformed wire accepted")
			}
		})
	}
	foreign := bytes.Replace(release.body, []byte(`"schema":`), []byte(`"target":"/etc/other","schema":`), 1)
	if _, err := newWireRelease(foreign, release.proofs); err == nil {
		t.Fatal("arbitrary target became a wire field")
	}
	for _, proofs := range [][][]byte{nil, {nil}, {make([]byte, 1<<20+1)}, make([][]byte, 17)} {
		if _, err := newWireRelease(release.body, proofs); err == nil {
			t.Fatal("invalid proof budget accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readWirePreamble(ctx, bytes.NewReader(valid)); !errors.Is(err, context.Canceled) {
		t.Fatal("read cancellation lost")
	}
	if err := writeWirePreamble(ctx, io.Discard, release); !errors.Is(err, context.Canceled) {
		t.Fatal("write cancellation lost")
	}
}

func TestWirePayloadRefusesChangedSizeDigestAndReaderFailure(t *testing.T) {
	release, files := wireFixture(t)
	for _, kind := range []string{"truncated", "extra", "digest", "nil-reader", "open-error"} {
		t.Run(kind, func(t *testing.T) {
			failure := errors.New("sealed member unavailable")
			err := writeWirePayloads(context.Background(), io.Discard, release, func(name string) (io.ReadCloser, error) {
				if kind == "nil-reader" {
					return nil, nil
				}
				if kind == "open-error" {
					return nil, failure
				}
				body := append([]byte(nil), files[name]...)
				switch kind {
				case "truncated":
					body = body[:len(body)-1]
				case "extra":
					body = append(body, '!')
				case "digest":
					body[0] ^= 1
				}
				return io.NopCloser(bytes.NewReader(body)), nil
			})
			if err == nil {
				t.Fatal("changed sealed payload accepted")
			}
			if kind == "open-error" && !errors.Is(err, failure) {
				t.Fatal("primary open cause lost")
			}
		})
	}
}
