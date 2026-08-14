package promotion

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameNoReplacePreservesRacedDestination(t *testing.T) {
	t.Parallel()
	root := realPromotionRoot(t)
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "source-owner"), []byte("heliopause"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := renameNoReplace(source, destination); err == nil {
		t.Fatal("renameNoReplace() replaced an existing destination")
	}
	if _, err := os.Stat(filepath.Join(source, "source-owner")); err != nil {
		t.Fatalf("failed rename removed source: %v", err)
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 0 {
		t.Fatalf("existing destination changed: %v, %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(destination, "source-owner")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source content appeared in destination: %v", err)
	}
}
