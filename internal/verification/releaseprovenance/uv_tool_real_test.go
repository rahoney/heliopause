//go:build realcorpus

package releaseprovenance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUVOfficialImageToolClosure(t *testing.T) {
	root := os.Getenv("HELOX_UV_LAYER_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("HELOX_UV_LAYER_ROOT must declare the acquired official layer directory")
	}
	d, identity := actualUVImage(t)
	receipt, err := VerifyUVImage(context.Background(), d[0], d[1], d[2], d[3], identity)
	if err != nil {
		t.Fatal(err)
	}
	var inputs []UVToolLayer
	for _, layer := range receipt.Layers() {
		file, err := os.Open(filepath.Join(root, layer.SHA256+".tar.gz"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = file.Close() })
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() != layer.Size {
			t.Fatal("official layer size/type mismatch")
		}
		inputs = append(inputs, UVToolLayer{file, info.Size()})
	}
	tool, err := receipt.VerifyToolLayers(context.Background(), d[3], inputs)
	if err != nil {
		t.Fatal(err)
	}
	if tool.Image() != identity || tool.Size() != 42544544 || tool.SHA256() != "de77a01a86452cc465a734257b5a2073d41115d1b8bbfe84ac636af53851b499" {
		t.Fatalf("official tool binding mismatch: %+v", tool)
	}
}
