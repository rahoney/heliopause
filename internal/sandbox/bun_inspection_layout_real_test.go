//go:build realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

// Frozen actual runc output is explicit named-local input. This does not run a
// native runtime or qualify production gVisor/lifecycle/Policy behavior.
func TestBunFrozenNativeLayoutBinding(t *testing.T) {
	corpus := os.Getenv("HELOX_BUN_LAYOUT_CORPUS")
	if os.Getenv("HELOX_BUN_PUBLIC_SOURCE") != "1" || corpus == "" {
		t.Fatal("declare public source and HELOX_BUN_LAYOUT_CORPUS")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, name := range []string{"transitive", "peer"} {
		if !t.Run(name, func(t *testing.T) {
			input, inventories := bunPublicInspectionInputs(t, ctx, name)
			file, err := os.Open(filepath.Join(corpus, "p3b4b-bun-"+name+"-layout-runc-v2-project-after-frozen.tar"))
			if err != nil {
				t.Fatal(err)
			}
			body, readErr := io.ReadAll(io.LimitReader(file, (8<<20)+1))
			closeErr := file.Close()
			if readErr != nil || closeErr != nil || len(body) > 8<<20 {
				t.Fatal("bounded actual capture unavailable", readErr, closeErr)
			}
			captured, err := artifactnpm.ParsePNPMInspectionPayloadArchive(ctx, body)
			if err != nil {
				t.Fatal("complete native project census", err)
			}
			controls := map[string][]byte{}
			for _, control := range input.Context().Controls() {
				if !control.Present() {
					t.Fatal("nonempty fixture missing control")
				}
				controls["project/"+control.Name()] = control.Body()
			}
			for name, expected := range controls {
				actual, err := captured.ReadFile(name)
				if err != nil || !bytes.Equal(actual, expected) {
					t.Fatal("native controls changed", err)
				}
			}
			render := func(wrongBin bool) []byte {
				var output bytes.Buffer
				writer := tar.NewWriter(&output)
				seenControls := 0
				for _, entry := range captured.Entries() {
					if entry.Name == "project" {
						if entry.Kind != tar.TypeDir {
							t.Fatal("project root kind")
						}
						continue
					}
					if _, present := controls[entry.Name]; present {
						if entry.Kind != tar.TypeReg {
							t.Fatal("control kind")
						}
						seenControls++
						continue
					}
					if entry.Name != "project/node_modules" && !strings.HasPrefix(entry.Name, "project/node_modules/") {
						t.Fatal("unknown native project entry", entry.Name)
					}
					header := tar.Header{Name: strings.TrimPrefix(entry.Name, "project/"), Typeflag: entry.Kind, Mode: entry.Mode, Uid: entry.UID, Gid: entry.GID, Size: entry.Size, Linkname: entry.Link}
					if wrongBin && header.Name == "node_modules/.bin/loose-envify" {
						header.Linkname = "../react/package.json"
					}
					if entry.Kind != tar.TypeReg && entry.Kind != tar.TypeDir && entry.Kind != tar.TypeSymlink {
						t.Fatal("unqualified native alias")
					}
					if err := writer.WriteHeader(&header); err != nil {
						t.Fatal(err)
					}
					if entry.Kind == tar.TypeReg {
						content, err := captured.ReadFile(entry.Name)
						if err != nil {
							t.Fatal(err)
						}
						if _, err := writer.Write(content); err != nil {
							t.Fatal(err)
						}
					}
				}
				if seenControls != 2 {
					t.Fatal("incomplete controls census")
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				if output.Len() > 8<<20 {
					t.Fatal("render exceeds capture bound")
				}
				return output.Bytes()
			}
			layout, err := artifactnpm.ParseBunLayoutArchive(ctx, render(false))
			if err != nil {
				t.Fatal(err)
			}
			bound, err := input.BindLayout(ctx, inventories, layout)
			if err != nil || !bound.MatchesInput(input) {
				t.Fatal("actual full native layout binding", err)
			}
			instances := 0
			for _, archive := range input.Artifacts() {
				placements, err := input.InstancesFor(archive)
				if err != nil {
					t.Fatal(err)
				}
				for _, placement := range placements {
					if _, err := bound.Target(archive, placement.Key); err != nil {
						t.Fatal(err)
					}
					instances++
				}
			}
			if _, err := input.BindLayout(ctx, inventories[1:], layout); err == nil {
				t.Fatal("partial actual inventory admitted")
			}
			if name == "peer" {
				wrong, err := artifactnpm.ParseBunLayoutArchive(ctx, render(true))
				if err != nil {
					t.Fatal("substitution failed before binder", err)
				}
				if got, err := input.BindLayout(ctx, inventories, wrong); err == nil || got.Digest().String() != "" {
					t.Fatal("actual bin substitution admitted")
				}
			}
			t.Logf("bun_native_layout=%s archives=%d instances=%d entries=%d exact_controls=true; gVisor/dynamic/Policy/Host_install=NOT_EXERCISED", name, len(input.Artifacts()), instances, len(bound.Entries()))
		}) {
			return
		}
	}
}
