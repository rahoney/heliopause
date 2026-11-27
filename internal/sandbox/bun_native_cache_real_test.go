//go:build realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

// Each expected native file-cache member is independently bound. Global native
// lookup indices are discarded data, never approval or completion evidence.
func TestBunFrozenNativeCacheBinding(t *testing.T) {
	corpus := os.Getenv("HELOX_BUN_CACHE_CORPUS")
	if os.Getenv("HELOX_BUN_PUBLIC_SOURCE") != "1" || corpus == "" {
		t.Fatal("declare public source and HELOX_BUN_CACHE_CORPUS")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, name := range []string{"transitive", "peer"} {
		if !t.Run(name, func(t *testing.T) {
			input, inventories := bunPublicInspectionInputs(t, ctx, name)
			if err := input.CheckNativeCacheInputs(inventories); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(filepath.Join(corpus, "p3b4c-bun-file-seed-"+name+"-runc-v4-cache-after-seed.tar"))
			if err != nil {
				t.Fatal(err)
			}
			body, readErr := io.ReadAll(io.LimitReader(file, (8<<20)+1))
			closeErr := file.Close()
			if readErr != nil || closeErr != nil || len(body) > 8<<20 {
				t.Fatal("bounded frozen cache capture unavailable", readErr, closeErr)
			}
			var bound []artifactnpm.BunNativeCacheEntry
			for _, inventory := range inventories {
				root, err := input.NativeCacheFolder(inventory.Artifact())
				if err != nil {
					t.Fatal(err)
				}
				reader := tar.NewReader(bytes.NewReader(body))
				var output bytes.Buffer
				writer := tar.NewWriter(&output)
				count := 0
				for {
					header, err := reader.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
					count++
					if count > 100000 {
						t.Fatal("capture entry bound")
					}
					prefix := "cache/" + root
					if header.Name != prefix && !strings.HasPrefix(header.Name, prefix+"/") {
						continue
					}
					copy := *header
					copy.Name = strings.TrimPrefix(copy.Name, "cache/")
					copy.PAXRecords = maps.Clone(header.PAXRecords)
					if _, present := copy.PAXRecords["path"]; present {
						copy.PAXRecords["path"] = copy.Name
					}
					if err := writer.WriteHeader(&copy); err != nil {
						t.Fatal(err)
					}
					if header.Typeflag == tar.TypeReg {
						if header.Size < 0 || header.Size > artifactnpm.ArchiveFileLimit {
							t.Fatal("cache file bound")
						}
						content, err := io.ReadAll(io.LimitReader(reader, header.Size+1))
						if err != nil || int64(len(content)) != header.Size {
							t.Fatal("capture content incomplete", err)
						}
						if _, err := writer.Write(content); err != nil {
							t.Fatal(err)
						}
					}
					if output.Len() > 8<<20 {
						t.Fatal("cache member envelope bound")
					}
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				entry, err := input.BindNativeCache(ctx, inventory, output.Bytes())
				if err != nil {
					t.Fatal("full measured cache member", err)
				}
				bound = append(bound, entry)
			}
			registry, err := input.RenderRegistryCache(ctx, bound)
			if err != nil || len(registry) == 0 {
				t.Fatal("complete registry envelope", err)
			}
			if partial, err := input.RenderRegistryCache(ctx, bound[1:]); err == nil || len(partial) != 0 {
				t.Fatal("partial cache accepted")
			}
			t.Logf("bun_native_cache=%s archives=%d full_source_files_and_SRI_marker=true registry_bytes=%d; gVisor/native_execution/Policy/Host_install=NOT_EXERCISED", name, len(bound), len(registry))
		}) {
			return
		}
	}
}
