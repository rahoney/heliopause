package gomodule

import (
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestProjectSnapshotRequiresExactControlChecksumPair(t *testing.T) {
	target, err := domain.NewInstallTarget("/fixture/project")
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := domain.NewInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	records := []DownloadRecord{{Path: "example.com/module", Version: "v1.0.0", GoMod: "/private/module.mod", Zip: "/private/module.zip", Sum: testH1('a'), GoModSum: testH1('b')}}
	mod := []byte("module example.com/project\nrequire example.com/module v1.0.0\n")
	graph := []byte("example.com/project example.com/module@v1.0.0\n")
	zLine := "example.com/module v1.0.0 " + testH1('a') + "\n"
	mLine := "example.com/module v1.0.0/go.mod " + testH1('b') + "\n"
	for name, sum := range map[string]string{
		"archive-mismatch": "example.com/module v1.0.0 " + testH1('c') + "\n" + mLine,
		"mod-mismatch":     zLine + "example.com/module v1.0.0/go.mod " + testH1('c') + "\n",
		"missing-archive":  mLine,
		"missing-mod":      zLine,
		"malformed":        "fixture-sum\n",
		"duplicate":        zLine + mLine + zLine,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := BuildProjectSnapshot(ctx, records, graph, mod, []byte(sum)); err == nil {
				t.Fatal("project checksum disagreement was accepted")
			}
		})
	}
	if _, err := BuildProjectSnapshot(ctx, records, graph, mod, []byte(zLine+mLine)); err != nil {
		t.Fatal(err)
	}
}
