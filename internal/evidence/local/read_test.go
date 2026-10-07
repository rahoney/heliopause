package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestReadReferenceRejectsMissingCorruptOrForeignEvidence(t *testing.T) {
	for _, name := range []string{"normal", "missing", "content", "subject", "reference", "symlink", "run-symlink", "directory", "oversize", "duplicate-json"} {
		t.Run(name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			store, _ := NewStore(root)
			run, item := storeFixture(t)
			refs, err := store.Record(context.Background(), run, []domain.Evidence{item})
			if err != nil {
				t.Fatal(err)
			}
			ref := refs[0]
			file := filepath.Join(store.root, run.String(), item.ID().String()+".json")
			identity, digest := item.Identity(), item.Digest()
			switch name {
			case "missing":
				err = os.Remove(file)
			case "content":
				err = os.WriteFile(file, []byte("{}"), 0o600)
			case "subject":
				identity, err = domain.NewResolvedArtifactIdentity(identity.Source(), "foreign", identity.Version(), identity.Variant())
			case "reference":
				ref, err = domain.NewEvidenceReference(item.ID(), "fixture:evidence")
			case "symlink":
				err = os.Remove(file)
				if err == nil {
					err = os.Symlink(filepath.Join(t.TempDir(), "outside"), file)
				}
			case "run-symlink":
				outside := filepath.Join(t.TempDir(), "run")
				err = os.Rename(filepath.Dir(file), outside)
				if err == nil {
					err = os.Symlink(outside, filepath.Dir(file))
				}
			case "directory":
				err = os.Remove(file)
				if err == nil {
					err = os.Mkdir(file, 0o700)
				}
			case "oversize":
				err = os.WriteFile(file, make([]byte, (1<<20)+1), 0o600)
			case "duplicate-json":
				var body []byte
				body, err = os.ReadFile(file)
				if err == nil {
					err = os.WriteFile(file, append(body, []byte("{}")...), 0o600)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			got, hash, err := store.ReadReference(context.Background(), run, ref, identity, digest)
			if name == "normal" {
				if err != nil || got != item || hash.String() == "" {
					t.Fatalf("read: %v", err)
				}
			} else if err == nil || got.ID().String() != "" || hash.String() != "" {
				t.Fatal("invalid Evidence became trusted")
			}
		})
	}
}
