package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/ports"
)

func TestInstalledMetadataSnapshotWithoutRequestedRoot(t *testing.T) {
	for _, count := range []int{0, 65} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			root, plan := makePypiVenvFixture(t)
			for i := 0; i < count; i++ {
				directory := filepath.Join(plan.site, fmt.Sprintf("original_%03d-1.0.dist-info", i))
				if err := os.Mkdir(directory, 0o700); err != nil {
					t.Fatal(err)
				}
				body := fmt.Sprintf("Metadata-Version: 2.4\nName: original-%03d\nVersion: 1.0\n\n", i)
				if err := os.WriteFile(filepath.Join(directory, "METADATA"), []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			transaction, err := beginPyPIVenvTransaction(plan)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := transaction.close(); err != nil {
					t.Error(err)
				}
			})
			original, err := capturePyPIInstalledSnapshot(transaction)
			if err != nil || len(original.metadataFiles()) != count {
				t.Fatalf("rootless original metadata count=%d: %v", len(original.metadataFiles()), err)
			}
			if err := original.verify(context.Background(), transaction); err != nil {
				t.Fatal(err)
			}
			managed, err := managedPyPIInventoryFromMetadata(transaction, original.metadataFiles())
			if err != nil || len(managed) != 0 || len(transaction.current.Files) != 0 {
				t.Fatal("metadata snapshot invented original ownership", err)
			}
			if _, err := os.Lstat(filepath.Join(root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("metadata capture wrote ownership", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := original.verify(ctx, transaction); !errors.Is(err, context.Canceled) {
				t.Fatal("original verification lost cancellation", err)
			}
			if count == 0 {
				if err := os.Mkdir(filepath.Join(plan.site, "added-1.0.dist-info"), 0o700); err != nil {
					t.Fatal(err)
				}
			} else {
				files := original.metadataFiles()
				first := string(files[0].Body)
				files[0].Body[0] = 'x'
				if string(original.metadataFiles()[0].Body) != first {
					t.Fatal("caller altered frozen original metadata")
				}
				path := filepath.Join(plan.site, files[0].Directory, "METADATA")
				if err := os.WriteFile(path, []byte(first+"changed\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := original.verify(context.Background(), transaction); ports.InstallUpdateFailureCode(err) != "UPDATE_SOURCE_CHANGED" {
				t.Fatal("changed original metadata was not rejected", err)
			}
		})
	}
}

func TestRootlessInventoryRequiresMetadataIdentityCoverage(t *testing.T) {
	metadata := func(directory, project, version string) artifactpypi.InstalledMetadataFile {
		return artifactpypi.InstalledMetadataFile{Directory: directory, Body: []byte("Metadata-Version: 2.4\nName: " + project + "\nVersion: " + version + "\n\n")}
	}
	original := metadata("original_name-1.0.dist-info", "original-name", "1.0")
	for name, files := range map[string][]artifactpypi.InstalledMetadataFile{
		"different name":        {metadata(original.Directory, "other", "1.0")},
		"different version":     {metadata(original.Directory, "original-name", "2.0")},
		"same project versions": {original, metadata("original_name-2.0.dist-info", "original-name", "2.0")},
		"repeated metadata":     {original, original},
		"ambiguous headers":     {{Directory: original.Directory, Body: []byte("Name: original-name\nName: other\nVersion: 1.0\n\n")}},
	} {
		t.Run(name, func(t *testing.T) {
			_, plan := makePypiVenvFixture(t)
			transaction, err := beginPyPIVenvTransaction(plan)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := transaction.close(); err != nil {
					t.Error(err)
				}
			})
			if _, err := managedPyPIInventoryFromMetadata(transaction, files); err == nil {
				t.Fatal("invalid rootless metadata was accepted")
			}
			if len(transaction.current.Files) != 0 {
				t.Fatal("invalid metadata changed original ownership")
			}
		})
	}
}
