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

func actualUVImage(t *testing.T) ([][]byte, UVImageIdentity) {
	t.Helper()
	var documents [][]byte
	for _, name := range []string{"bundle.json", "index.json", "manifest.json", "config.json", "identity.json"} {
		body, err := os.ReadFile(filepath.Join("testdata", "uv-0.13.4", name))
		if err != nil {
			t.Fatal(err)
		}
		documents = append(documents, body)
	}
	var identity UVImageIdentity
	if err := json.Unmarshal(documents[4], &identity); err != nil {
		t.Fatal(err)
	}
	return documents[:4], identity
}

func TestActualUVImageProvenanceBindsSignedIndexToAMD64(t *testing.T) {
	d, identity := actualUVImage(t)
	receipt, err := VerifyUVImage(context.Background(), d[0], d[1], d[2], d[3], identity)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Identity() != identity || receipt.ConfigSHA256() != uvDigest(d[3]) || receipt.BundleSHA256() != uvDigest(d[0]) || len(receipt.Layers()) != 2 || receipt.Layers()[0].Size != 19395988 || receipt.Invocation() != uvRepository+"/actions/runs/39441647119/attempts/1" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	layers := receipt.Layers()
	layers[0].Size = 1
	if receipt.Layers()[0].Size == 1 {
		t.Fatal("authenticated receipt is mutable")
	}
	// A valid upstream proof must never gain HAA publisher authority.
	_, haa, subjects := actualSignedRelease(t)
	if _, err := Verify(context.Background(), d[0], haa, subjects); err == nil {
		t.Fatal("upstream proof accepted as HAA release")
	}
}

func TestUVImageRejectsUnverifiedProofAndIdentity(t *testing.T) {
	d, identity := actualUVImage(t)
	change := func(fn func(*UVImageIdentity)) UVImageIdentity { result := identity; fn(&result); return result }
	mutate := func(fn func(map[string]any)) []byte {
		var value map[string]any
		if err := json.Unmarshal(d[0], &value); err != nil {
			t.Fatal(err)
		}
		fn(value)
		result, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	haaProof, _, _ := actualSignedRelease(t)
	cases := []struct {
		name     string
		proof    []byte
		identity UVImageIdentity
	}{
		{"wrong-publisher", haaProof, identity},
		{"wrong-source", d[0], change(func(v *UVImageIdentity) { v.SourceCommit = strings.Repeat("a", 40) })},
		{"wrong-run", d[0], change(func(v *UVImageIdentity) { v.WorkflowRun = "39441647120" })},
		{"wrong-version", d[0], change(func(v *UVImageIdentity) { v.Version = "0.13.5" })},
		{"prerelease", d[0], change(func(v *UVImageIdentity) { v.Version = "0.13.4-rc1" })},
		{"unsigned-index", d[0], change(func(v *UVImageIdentity) { v.IndexSHA256 = strings.Repeat("b", 64) })},
		{"different-manifest", d[0], change(func(v *UVImageIdentity) { v.ManifestSHA256 = strings.Repeat("b", 64) })},
		{"missing-proof", nil, identity},
		{"oversize", make([]byte, maxBundle+1), identity},
		{"trailing-json", append(append([]byte{}, d[0]...), []byte("\n{}")...), identity},
		{"no-tlog", mutate(func(v map[string]any) { delete(v["verificationMaterial"].(map[string]any), "tlogEntries") }), identity},
		{"tampered-statement", mutate(func(v map[string]any) {
			e := v["dsseEnvelope"].(map[string]any)
			p, _ := base64.StdEncoding.DecodeString(e["payload"].(string))
			e["payload"] = base64.StdEncoding.EncodeToString(append(p, ' '))
		}), identity},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r, err := VerifyUVImage(context.Background(), tt.proof, d[1], d[2], d[3], tt.identity)
			if err == nil || r.BundleSHA256() != "" || len(r.Layers()) != 0 {
				t.Fatalf("unverified receipt: %+v %v", r, err)
			}
		})
	}
	for i, name := range []string{"index", "manifest", "config"} {
		t.Run("changed-"+name, func(t *testing.T) {
			copy := append([][]byte(nil), d...)
			copy[i+1] = append(append([]byte{}, d[i+1]...), ' ')
			if _, err := VerifyUVImage(context.Background(), copy[0], copy[1], copy[2], copy[3], identity); err == nil {
				t.Fatal("changed document accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := VerifyUVImage(ctx, d[0], d[1], d[2], d[3], identity); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestUVImageRejectsAmbiguousPlatformAndConfigChain(t *testing.T) {
	d, identity := actualUVImage(t)
	for _, name := range []string{"duplicate-platform", "wrong-platform", "manifest-size", "trailing-index", "config-digest", "layer-bound", "layer-type", "version-label", "revision-label"} {
		t.Run(name, func(t *testing.T) {
			var index, image, config map[string]any
			for i, v := range []*map[string]any{&index, &image, &config} {
				if err := json.Unmarshal(d[i+1], v); err != nil {
					t.Fatal(err)
				}
			}
			switch name {
			case "duplicate-platform":
				index["manifests"] = append(index["manifests"].([]any), index["manifests"].([]any)[0])
			case "wrong-platform":
				index["manifests"].([]any)[0].(map[string]any)["platform"].(map[string]any)["architecture"] = "arm64"
			case "manifest-size":
				index["manifests"].([]any)[0].(map[string]any)["size"] = float64(1)
			case "config-digest":
				image["config"].(map[string]any)["digest"] = "sha256:" + strings.Repeat("a", 64)
			case "layer-bound":
				image["layers"].([]any)[0].(map[string]any)["size"] = float64(65 << 20)
			case "layer-type":
				image["layers"].([]any)[0].(map[string]any)["mediaType"] = "application/octet-stream"
			case "version-label":
				config["config"].(map[string]any)["Labels"].(map[string]any)["org.opencontainers.image.version"] = "0.13.5"
			case "revision-label":
				config["config"].(map[string]any)["Labels"].(map[string]any)["org.opencontainers.image.revision"] = strings.Repeat("a", 40)
			}
			configBody, _ := json.Marshal(config)
			if name == "version-label" || name == "revision-label" {
				image["config"].(map[string]any)["digest"] = "sha256:" + uvDigest(configBody)
				image["config"].(map[string]any)["size"] = len(configBody)
			} else {
				configBody = d[3]
			}
			imageBody := d[2]
			if name == "config-digest" || name == "layer-bound" || name == "layer-type" || name == "version-label" || name == "revision-label" {
				imageBody, _ = json.Marshal(image)
				index["manifests"].([]any)[0].(map[string]any)["digest"] = "sha256:" + uvDigest(imageBody)
				index["manifests"].([]any)[0].(map[string]any)["size"] = len(imageBody)
			}
			indexBody, _ := json.Marshal(index)
			if name == "trailing-index" {
				indexBody = append(indexBody, []byte("\n{}")...)
			}
			expected := identity
			expected.IndexSHA256 = uvDigest(indexBody)
			expected.ManifestSHA256 = uvDigest(imageBody)
			// This tests structural binding independently; altered index bytes have
			// no signature and must not be passed off as an authenticated receipt.
			if _, err := bindUVImage(indexBody, imageBody, configBody, expected); err == nil {
				t.Fatal("invalid chain accepted")
			}
		})
	}
}
