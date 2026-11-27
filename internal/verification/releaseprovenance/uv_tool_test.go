package releaseprovenance

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"testing"
)

func syntheticUVExecutable() []byte {
	body := make([]byte, 120)
	copy(body, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(body[16:], 2)
	binary.LittleEndian.PutUint16(body[18:], 62)
	binary.LittleEndian.PutUint32(body[20:], 1)
	binary.LittleEndian.PutUint64(body[32:], 64)
	binary.LittleEndian.PutUint16(body[52:], 64)
	binary.LittleEndian.PutUint16(body[54:], 56)
	binary.LittleEndian.PutUint16(body[56:], 1)
	binary.LittleEndian.PutUint32(body[64:], 1)
	binary.LittleEndian.PutUint32(body[68:], 5)
	binary.LittleEndian.PutUint64(body[96:], uint64(len(body)))
	binary.LittleEndian.PutUint64(body[104:], uint64(len(body)))
	binary.LittleEndian.PutUint64(body[112:], 4096)
	return body
}

func syntheticUVLayers(t testing.TB, edit func(*tar.Header)) (UVImageReceipt, []byte, []UVToolLayer) {
	t.Helper()
	var expanded bytes.Buffer
	writer := tar.NewWriter(&expanded)
	for _, name := range []string{"uv", "uvx"} {
		body := syntheticUVExecutable()
		header := &tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0755, Uid: 0, Gid: 0, Size: int64(len(body))}
		if edit != nil {
			edit(header)
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeReg && header.Size > 0 {
			if _, err := writer.Write(body[:header.Size]); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	z := gzip.NewWriter(&compressed)
	if _, err := z.Write(expanded.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	config, err := json.Marshal(map[string]any{"rootfs": map[string]any{"diff_ids": []string{"sha256:" + uvDigest(expanded.Bytes())}}})
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic structural coverage only. External callers cannot construct
	// this receipt; actual publisher/native closure is tested by realcorpus.
	receipt := UVImageReceipt{identity: UVImageIdentity{Version: "0.13.4"}, bundleSHA256: "synthetic-test-only", configSHA256: uvDigest(config), layers: []UVImageLayer{{uvDigest(compressed.Bytes()), int64(compressed.Len())}}}
	return receipt, config, []UVToolLayer{{bytes.NewReader(compressed.Bytes()), int64(compressed.Len())}}
}

func FuzzUVToolLayersNoPanic(f *testing.F) {
	_, _, inputs := syntheticUVLayers(f, nil)
	z, err := gzip.NewReader(inputs[0].Reader)
	if err != nil {
		f.Fatal(err)
	}
	seed, err := io.ReadAll(z)
	if err != nil {
		f.Fatal(err)
	}
	if err := z.Close(); err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, expanded []byte) {
		if len(expanded) > 1<<20 {
			return
		}
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err := writer.Write(expanded); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		config, err := json.Marshal(map[string]any{"rootfs": map[string]any{"diff_ids": []string{"sha256:" + uvDigest(expanded)}}})
		if err != nil {
			t.Fatal(err)
		}
		r := UVImageReceipt{identity: UVImageIdentity{Version: "0.13.4"}, bundleSHA256: "structural-fuzz-only", configSHA256: uvDigest(config), layers: []UVImageLayer{{uvDigest(compressed.Bytes()), int64(compressed.Len())}}}
		_, _ = r.VerifyToolLayers(context.Background(), config, []UVToolLayer{{bytes.NewReader(compressed.Bytes()), int64(compressed.Len())}})
	})
}

func TestUVToolStructuralBindingAndRefusals(t *testing.T) {
	receipt, config, layers := syntheticUVLayers(t, nil)
	verified, err := receipt.VerifyToolLayers(context.Background(), config, layers)
	if err != nil || verified.Size() != 120 || verified.SHA256() != uvDigest(syntheticUVExecutable()) || verified.Image() != receipt.Identity() {
		t.Fatalf("tool binding failed: %+v %v", verified, err)
	}
	for _, name := range []string{"zero-receipt", "missing-layer", "changed-config", "changed-layer", "size", "diffID", "duplicate-layer"} {
		t.Run(name, func(t *testing.T) {
			r, c, inputs := syntheticUVLayers(t, nil)
			switch name {
			case "zero-receipt":
				r = UVImageReceipt{}
			case "missing-layer":
				inputs = nil
			case "changed-config":
				c = append(c, ' ')
			case "changed-layer":
				inputs[0].Reader = bytes.NewReader([]byte("changed"))
			case "size":
				inputs[0].Size++
			case "diffID":
				c = []byte(`{"rootfs":{"diff_ids":["sha256:wrong"]}}`)
				r.configSHA256 = uvDigest(c)
			case "duplicate-layer":
				inputs = append(inputs, inputs[0])
			}
			if result, err := r.VerifyToolLayers(context.Background(), c, inputs); err == nil || result.SHA256() != "" {
				t.Fatalf("unverified native receipt: %+v %v", result, err)
			}
		})
	}
	for _, name := range []string{"symlink", "owner", "writable", "traversal", "whiteout", "duplicate-file", "missing-uv", "invalid-elf"} {
		t.Run(name, func(t *testing.T) {
			r, c, inputs := syntheticUVLayers(t, func(h *tar.Header) {
				switch name {
				case "symlink":
					h.Typeflag = tar.TypeSymlink
					h.Linkname = "/etc/passwd"
					h.Size = 0
				case "owner":
					h.Uid = 1000
				case "writable":
					h.Mode = 0777
				case "traversal":
					h.Name = "../" + h.Name
				case "whiteout":
					h.Name = ".wh." + h.Name
				case "duplicate-file":
					h.Name = "uv"
				case "missing-uv":
					h.Name = "other"
				case "invalid-elf":
					h.Size = 1
				}
			})
			if _, err := r.VerifyToolLayers(context.Background(), c, inputs); err == nil {
				t.Fatal("unsupported native layout accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := receipt.VerifyToolLayers(ctx, config, layers); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
}

func TestUVToolRejectsLaterLayerReplacementAndTotalBudget(t *testing.T) {
	r, c, layers := syntheticUVLayers(t, nil)
	body, err := io.ReadAll(layers[0].Reader)
	if err != nil {
		t.Fatal(err)
	}
	var value struct {
		RootFS struct {
			DiffIDs []string `json:"diff_ids"`
		} `json:"rootfs"`
	}
	if err := json.Unmarshal(c, &value); err != nil {
		t.Fatal(err)
	}
	value.RootFS.DiffIDs = append(value.RootFS.DiffIDs, value.RootFS.DiffIDs[0])
	c, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	r.configSHA256 = uvDigest(c)
	r.layers = append(r.layers, r.layers[0])
	layers = []UVToolLayer{{bytes.NewReader(body), int64(len(body))}, {bytes.NewReader(body), int64(len(body))}}
	if _, err := r.VerifyToolLayers(context.Background(), c, layers); err == nil {
		t.Fatal("later layer replaced tool")
	}
	r.layers[0].Size = maxUVToolSupplyBytes + 1
	layers[0].Size = r.layers[0].Size
	if _, err := r.VerifyToolLayers(context.Background(), c, layers); err == nil {
		t.Fatal("oversize layer accepted")
	}
}
