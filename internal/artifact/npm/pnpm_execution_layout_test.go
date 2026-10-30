package npm

import (
	"archive/tar"
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPNPMExecutionArchiveCopiesHardlinksWithoutChangingRelativeTopology(t *testing.T) {
	f := fullInspectionLayoutFixture(t, false)
	first, linked := "", false
	for i := range f.entries {
		entry := &f.entries[i]
		if !strings.HasSuffix(entry.header.Name, "/LICENSE") {
			continue
		}
		if first == "" {
			first = entry.header.Name
			continue
		}
		entry.header.Typeflag, entry.header.Linkname, entry.header.Size, entry.body = tar.TypeLink, first, 0, ""
		linked = true
		break
	}
	if !linked {
		t.Fatal("fixture lacks separate original copies")
	}
	prepared, err := f.input.BindLayout(context.Background(), f.inventories, f.parse(t), "/tmp/haa-store")
	if err != nil {
		t.Fatal(err)
	}
	archive, err := copyPNPMExecutionArchive(context.Background(), prepared)
	if err != nil {
		t.Fatal(err)
	}
	links := 0
	for _, entry := range archive.Entries() {
		if entry.Kind == tar.TypeLink {
			t.Fatal("execution shares a mutable hardlink")
		}
		if entry.Kind == tar.TypeSymlink {
			links++
		}
		if strings.HasSuffix(entry.Name, "/LICENSE") {
			body, err := archive.ReadFile(entry.Name)
			if err != nil || entry.Size != int64(len(body)) || string(body) != "same license bytes\n" {
				t.Fatal("copied alias lost measured original bytes", err)
			}
		}
	}
	if links == 0 {
		t.Fatal("native relative dependency topology disappeared")
	}
	data, _, _ := executionLayoutFixture(t, "node_modules/child")
	execution, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	layout := NodeExecutionLayout{binding: executionLayoutFacts(execution), layout: archive, entry: first, digest: prepared.Digest(), nativePNPM: true}
	body, err := RenderNodeExecutionInputs(context.Background(), execution, layout)
	if err != nil {
		t.Fatal("bound PNPM transport rejected its relative links", err)
	}
	if len(body) == 0 {
		t.Fatal("empty execution input")
	}
	layout.nativePNPM = false
	if _, err := RenderNodeExecutionInputs(context.Background(), execution, layout); err == nil {
		t.Fatal("PNPM transport broadened generic npm link acceptance")
	}
}
