//go:build linux

package githubrelease

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapHeldReaderSurvivesSourceCloseAndCopiesFacts(t *testing.T) {
	fixture := newReleaseSetFixture(t, nil)
	if _, err := fixture.set.ResolveDependencies(context.Background(), fixture.set.Reference(), fixture.install); err != nil {
		t.Fatal(err)
	}
	manifest, proofs, err := fixture.set.BootstrapMaterials()
	if err != nil || manifest.Version() != "v0.1.0" || len(proofs) == 0 {
		t.Fatal(err)
	}
	original := append([]byte(nil), proofs[0]...)
	proofs[0][0] ^= 1
	_, fresh, err := fixture.set.BootstrapMaterials()
	if err != nil || !bytes.Equal(fresh[0], original) {
		t.Fatal("source facts were mutated", err)
	}
	reader, size, err := fixture.set.OpenBootstrapMember(context.Background(), "helox-release-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.set.Close(); err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	if err != nil || uint64(len(body)) != size || len(body) == 0 {
		t.Fatal("held descriptor did not survive source cleanup", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal("cached reader close changed", err)
	}
	if _, _, err := fixture.set.OpenBootstrapMember(context.Background(), "helox-release-manifest.json"); err == nil {
		t.Fatal("closed source reopened")
	}
	if _, _, err := fixture.set.BootstrapMaterials(); err == nil {
		t.Fatal("closed facts returned")
	}
}

func TestBootstrapReaderRejectsUnknownAndCancelledMembers(t *testing.T) {
	fixture := newReleaseSetFixture(t, nil)
	if _, err := fixture.set.ResolveDependencies(context.Background(), fixture.set.Reference(), fixture.install); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../artifact", "helox-linux-arm64", "unknown"} {
		if _, _, err := fixture.set.OpenBootstrapMember(context.Background(), name); err == nil {
			t.Fatal("non-selected member opened", name)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := fixture.set.OpenBootstrapMember(ctx, "helox-release-manifest.json"); err == nil {
		t.Fatal("cancelled member opened")
	}
}

func TestBootstrapReaderUsesHeldInodeAfterPathReplacement(t *testing.T) {
	fixture := newReleaseSetFixture(t, nil)
	if _, err := fixture.set.ResolveDependencies(context.Background(), fixture.set.Reference(), fixture.install); err != nil {
		t.Fatal(err)
	}
	entry := fixture.set.entries["helox-release-manifest.json"]
	parts := strings.Split(entry.Acquired.ContentHandle(), ":")
	path := filepath.Join(fixture.client.intakeRoot, parts[1], "asset")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	saved := path + "-saved"
	if err := os.Rename(path, saved); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(path); err != nil {
			t.Error(err)
		}
		if err := os.Rename(saved, path); err != nil {
			t.Error(err)
		}
	}()
	if err := os.WriteFile(path, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	reader, _, err := fixture.set.OpenBootstrapMember(context.Background(), "helox-release-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(body, original) {
		t.Fatal("reader reopened mutable pathname", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
}
