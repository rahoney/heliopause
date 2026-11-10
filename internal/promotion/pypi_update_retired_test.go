package promotion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPyPIRetiredMetadataDirectoryBudgetDepthAndNonemptyRefusal(t *testing.T) {
	root := realPromotionRoot(t)
	child := filepath.Join(root, "licenses", "third_party")
	if err := os.MkdirAll(child, 0o700); err != nil {
		t.Fatal(err)
	}
	if tree, empty, err := snapshotPyPIRetiredMetadata(root, 3); err != nil || !empty || len(tree) != 3 {
		t.Fatal("exact directory budget rejected", tree, empty, err)
	}
	if _, _, err := snapshotPyPIRetiredMetadata(root, 2); err == nil {
		t.Fatal("directory budget bypassed")
	}
	if _, _, err := snapshotPyPIRetiredMetadata(root, 0); err == nil {
		t.Fatal("exhausted cumulative directory budget accepted")
	}
	file := filepath.Join(child, "user-data")
	if err := os.WriteFile(file, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, empty, err := snapshotPyPIRetiredMetadata(root, 3); err != nil || empty {
		t.Fatal("user file treated as retired empty state", err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realPromotionRoot(t), file); err != nil {
		t.Fatal(err)
	}
	if _, empty, err := snapshotPyPIRetiredMetadata(root, 3); err != nil || empty {
		t.Fatal("symlink treated as retired empty state", err)
	}
	deep := realPromotionRoot(t)
	current := deep
	for i := 0; i < pypiRetiredMetadataDepthLimit; i++ {
		current = filepath.Join(current, "d")
		if err := os.Mkdir(current, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if _, empty, err := snapshotPyPIRetiredMetadata(deep, 1000); err != nil || !empty {
		t.Fatal("exact depth bound rejected", err)
	}
	if err := os.Mkdir(filepath.Join(current, "too-deep"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := snapshotPyPIRetiredMetadata(deep, 1000); err == nil {
		t.Fatal("depth budget bypassed")
	}
}
