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

func TestEvidenceProfilesRemainExplicitAcrossStorage(t *testing.T) {
	for _, current := range []bool{false, true} {
		store, err := NewStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		run, item := storeFixture(t)
		profile, _ := domain.CheckProfileDigest("record-fixture-v1", "pinned-runtime")
		var checks []domain.CheckExecution
		if current {
			check, err := domain.NewCheckExecution(item.CheckID(), domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
			if err != nil {
				t.Fatal(err)
			}
			checks = append(checks, check)
		}
		refs, err := store.Record(context.Background(), run, []domain.Evidence{item}, checks...)
		if err != nil {
			t.Fatal(err)
		}
		recorded, _, err := store.ReadReference(context.Background(), run, refs[0], item.Identity(), item.Digest())
		if err != nil {
			t.Fatal(err)
		}
		if current && recorded.ProfileDigest() != profile || !current && recorded.ProfileDigest().String() != "" {
			t.Fatal("stored legacy/current profile changed")
		}
	}
}

func TestEvidenceRecordingRefusesForeignOrDuplicateActualCheck(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		store, err := NewStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		run, item := storeFixture(t)
		profile, _ := domain.CheckProfileDigest("record-fixture-v1", "pinned-runtime")
		id := item.CheckID()
		if !duplicate {
			id, _ = domain.NewCheckID("foreign-check")
		}
		check, _ := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
		checks := []domain.CheckExecution{check}
		if duplicate {
			checks = append(checks, check)
		}
		if _, err := store.Record(context.Background(), run, []domain.Evidence{item}, checks...); err == nil {
			t.Fatal("foreign/duplicate check recorded")
		}
		if _, err := os.Stat(filepath.Join(store.root, run.String())); !os.IsNotExist(err) {
			t.Fatal("invalid batch created a run")
		}
	}
}

func TestEvidenceRecordingCannotRelabelAnExistingProfile(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run, item := storeFixture(t)
	profile, _ := domain.CheckProfileDigest("record-fixture-v1", "")
	foreign, _ := domain.CheckProfileDigest("foreign-profile-v1", "")
	bound, err := domain.NewEvidence(item.ID(), item.CheckID(), item.Identity(), item.Digest(), item.Kind(), item.Summary(), foreign)
	if err != nil {
		t.Fatal(err)
	}
	check, _ := domain.NewCheckExecution(item.CheckID(), domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
	if _, err := store.Record(context.Background(), run, []domain.Evidence{bound}, check); err == nil {
		t.Fatal("foreign profile was relabelled as current")
	}
}
