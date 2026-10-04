package gomodule

import (
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestDependencyFreeProjectRequiresExplicitCompleteGraph(t *testing.T) {
	target, _ := domain.NewInstallTarget("/fixture/project")
	ctx, _ := domain.NewInstallContext(target)
	mod := []byte("module example.com/app\ngo 1.26.0\n")
	graph := []byte("example.com/app go@1.26.0\ngo@1.26.0 toolchain@go1.26.0\n")
	for _, body := range [][]byte{nil, []byte(" \n")} {
		records, err := ParseProjectDownloadJSON(body)
		if err != nil || len(records) != 0 {
			t.Fatal(err)
		}
		snapshot, err := BuildProjectSnapshot(ctx, records, graph, mod, nil)
		if err != nil || !snapshot.DependencyFree() || !snapshot.Valid() || len(snapshot.Dependencies()) != 0 {
			t.Fatalf("explicit empty project: %v", err)
		}
		if _, err := ParseDownloadJSON(body); err == nil {
			t.Fatal("primary module grammar accepted a missing result")
		}
		if _, err := domain.NewProjectDependencySnapshot(ctx, Source(), snapshot.ControlDigests(), nil, snapshot.GraphDigest()); err == nil {
			t.Fatal("implicit missing dependency list became a complete empty project")
		}
	}
	for name, change := range map[string]struct{ mod, graph, sum string }{
		"missing-go-edge":    {string(mod), "", ""},
		"wrong-main":         {string(mod), "example.com/foreign go@1.26.0\n", ""},
		"public-edge":        {string(mod), "example.com/app example.com/missing@v1.0.0\n", ""},
		"hidden-requirement": {string(mod) + "require example.com/missing v1.0.0\n", string(graph), ""},
		"excluded-module":    {string(mod) + "exclude example.com/missing v1.0.0\n", string(graph), ""},
		"tool-module":        {string(mod) + "tool example.com/missing/tool\n", string(graph), ""},
		"retained-sums":      {string(mod), string(graph), "example.com/missing v1.0.0 " + testH1('a') + "\n"},
	} {
		t.Run(name, func(t *testing.T) {
			if snapshot, err := BuildProjectSnapshot(ctx, nil, []byte(change.graph), []byte(change.mod), []byte(change.sum)); err == nil || snapshot.Valid() {
				t.Fatal("unbound empty graph accepted")
			}
		})
	}
}
