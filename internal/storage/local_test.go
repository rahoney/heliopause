package storage

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	evidencelocal "github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
)

func openTestStorage(ctx context.Context, path string, policy domain.StoragePolicy) (*Local, error) {
	storage, err := NewLocal(ctx, path, policy)
	if err != nil {
		return nil, err
	}
	if err := storage.open(ctx); err != nil {
		return nil, errors.Join(err, storage.Close())
	}
	return storage, nil
}

func storageTestRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "haa")
}

func storageFixturePolicy(t *testing.T, runs uint32) domain.StoragePolicy {
	t.Helper()
	spec := domain.DefaultStoragePolicy().Spec()
	spec.TotalBytes = 64 << 10
	spec.CacheBytes = 16 << 10
	spec.EvidenceBytes = 32 << 10
	spec.MinimumFreeBytes = 1
	spec.EvidenceRuns = runs
	p, err := domain.NewStoragePolicy(spec)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func writeFixtureEvidence(t *testing.T, s *Local) (domain.RunID, []domain.EvidenceReference, error) {
	t.Helper()
	op, _ := domain.NewOperationID()
	run, _ := domain.NewRunID()
	fixture, _ := fakeworkflow.New(fakeworkflow.Safe)
	reference, _ := fixture.Reference()
	resolved, err := fixture.Resolve(context.Background(), reference)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := fixture.Acquire(context.Background(), run, resolved)
	if err != nil {
		t.Fatal(err)
	}
	verification, err := fixture.Verify(context.Background(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := fixture.Inspect(context.Background(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	store, err := evidencelocal.NewStore(filepath.Join(s.path, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := ports.WithStorageOperation(ports.WithStorage(context.Background(), s), op)
	checks := append([]domain.CheckExecution{verification.Execution()}, inspection.Executions()...)
	refs, err := store.Record(ctx, run, append(verification.Evidence(), inspection.Evidence()...), checks...)
	return run, refs, err
}

func TestEvidenceOwnerReservesAndPublishesDurableAccounting(t *testing.T) {
	path := storageTestRoot(t)
	policy := storageFixturePolicy(t, 2)
	s, err := openTestStorage(context.Background(), path, policy)
	if err != nil {
		t.Fatal(err)
	}
	run, refs, err := writeFixtureEvidence(t, s)
	if err != nil || len(refs) != 2 {
		t.Fatal("current Evidence owner could not publish", err)
	}
	record := s.entries[resourceKey("evidence", run.String())]
	if record.Phase != "RETAINED" || record.LogicalBytes == 0 || record.Content == "" || s.charged != record.LogicalBytes || s.pending != 0 {
		t.Fatal("published accounting lost", record, s.charged, s.pending)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = openTestStorage(context.Background(), path, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.entries[resourceKey("evidence", run.String())] != record || s.charged != record.LogicalBytes || s.runs != 1 {
		t.Fatal("restart lost durable ownership/accounting")
	}
	if _, _, err := writeFixtureEvidence(t, s); err != nil {
		t.Fatal(err)
	}
	if _, _, err := writeFixtureEvidence(t, s); err == nil {
		t.Fatal("full unexpired retention silently grew")
	}
	if s.runs != 2 {
		t.Fatal("failed admission changed existing records")
	}
}

func reserveFixture(t *testing.T, s *Local) (ports.StorageLease, ports.StorageRequest) {
	t.Helper()
	op, _ := domain.NewOperationID()
	run, _ := domain.NewRunID()
	temporary, err := os.MkdirTemp(filepath.Join(s.path, "evidence"), ".owned-")
	if err != nil {
		t.Fatal(err)
	}
	request := ports.StorageRequest{Operation: op, Run: run, Kind: "evidence", Temporary: filepath.Base(temporary), Target: run.String(), Bytes: 1024, Files: 2}
	lease, err := s.Reserve(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return lease, request
}

func TestStorageAbortRequiresConfirmedAbsenceAndDoesNotEraseOnClose(t *testing.T) {
	path := storageTestRoot(t)
	policy := storageFixturePolicy(t, 2)
	s, err := openTestStorage(context.Background(), path, policy)
	if err != nil {
		t.Fatal(err)
	}
	lease, request := reserveFixture(t, s)
	if err := lease.Abort(context.Background()); err == nil {
		t.Fatal("live resource reservation released")
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = openTestStorage(context.Background(), path, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.runs != 1 || s.charged != request.Bytes || s.pending != request.Bytes {
		t.Fatal("restart/PID disappearance lost outstanding reservation")
	}
	// An explicit owner cleanup, rather than a PID guess, authorizes release.
	if err := s.evidence.Remove(request.Temporary); err != nil {
		t.Fatal(err)
	}
	recovered := &resourceLease{store: s, record: s.entries[resourceKey("evidence", request.Run.String())]}
	if err := recovered.Abort(context.Background()); err != nil {
		t.Fatal(err)
	}
	offset := s.offset
	if err := recovered.Abort(context.Background()); err != nil || s.offset != offset {
		t.Fatal("confirmed absent reservation release is not idempotent", err)
	}
	if s.runs != 0 || s.charged != 0 || s.pending != 0 {
		t.Fatal("released accounting remained charged")
	}
	// The rejected duplicate must not append an invalid journal event.
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = openTestStorage(context.Background(), path, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
}

func TestStorageRefusesChangedPublicationAndPreservesForeignFiles(t *testing.T) {
	s, err := openTestStorage(context.Background(), storageTestRoot(t), storageFixturePolicy(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	lease, request := reserveFixture(t, s)
	defer lease.Close()
	if err := s.evidence.Rename(request.Temporary, ".original"); err != nil {
		t.Fatal(err)
	}
	if err := s.evidence.Mkdir(request.Target, 0700); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(s.path, "evidence", request.Target, "foreign.json")
	if err := os.WriteFile(foreign, []byte("foreign bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := lease.Commit(context.Background()); err == nil {
		t.Fatal("replacement directory published under old ownership")
	}
	if err := lease.Abort(context.Background()); err == nil {
		t.Fatal("unknown foreign target treated as absent")
	}
	body, err := os.ReadFile(foreign)
	if err != nil || string(body) != "foreign bytes" {
		t.Fatal("foreign payload changed")
	}
}

func TestStorageParallelCommandsCannotOverbookRetention(t *testing.T) {
	path := storageTestRoot(t)
	policy := storageFixturePolicy(t, 1)
	a, err := openTestStorage(context.Background(), path, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := openTestStorage(context.Background(), path, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	start := make(chan struct{})
	outcomes := make(chan error, 2)
	var wait sync.WaitGroup
	for _, store := range []*Local{a, b} {
		wait.Add(1)
		go func(s *Local) { defer wait.Done(); <-start; _, _, err := writeFixtureEvidence(t, s); outcomes <- err }(store)
	}
	close(start)
	wait.Wait()
	close(outcomes)
	passed := 0
	for err := range outcomes {
		if err == nil {
			passed++
		}
	}
	if passed != 1 {
		t.Fatal("retention reservation was overbooked or made no progress", passed)
	}
	if err := a.withLock(context.Background(), func() error {
		if a.runs != 1 {
			return errors.New("invalid retained run count")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestStorageJournalAndFilesystemUnknownRemainFailClosed(t *testing.T) {
	for _, mode := range []string{"hardlink", "incomplete", "filesystem-reserve"} {
		t.Run(mode, func(t *testing.T) {
			path := storageTestRoot(t)
			policy := storageFixturePolicy(t, 2)
			if mode == "filesystem-reserve" {
				spec := policy.Spec()
				spec.MinimumFreeBytes = 1 << 62
				var err error
				policy, err = domain.NewStoragePolicy(spec)
				if err != nil {
					t.Fatal(err)
				}
			}
			s, err := openTestStorage(context.Background(), path, policy)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "filesystem-reserve" {
				defer s.Close()
				if _, _, err := writeFixtureEvidence(t, s); err == nil || !strings.Contains(err.Error(), "filesystem") {
					t.Fatal("unavailable filesystem reserve ignored", err)
				}
				return
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			journal := filepath.Join(path, ".storage", "resources.jsonl")
			if mode == "hardlink" {
				if err := os.Link(journal, filepath.Join(path, "foreign-link")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(journal, []byte(`{"action":"reserve"`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if reopened, err := openTestStorage(context.Background(), path, policy); err == nil {
				_ = reopened.Close()
				t.Fatal("uncertain journal accepted or silently repaired")
			}
		})
	}
}

func TestStorageAcceptsOwnedLegacyParentButKeepsChildrenPrivate(t *testing.T) {
	path := storageTestRoot(t)
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	s, err := openTestStorage(context.Background(), path, storageFixturePolicy(t, 2))
	if err != nil {
		t.Fatal("existing owned 0755 HAA parent rejected", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(path, "evidence"), 0755); err != nil {
		t.Fatal(err)
	}
	if reopened, err := openTestStorage(context.Background(), path, storageFixturePolicy(t, 2)); err == nil {
		_ = reopened.Close()
		t.Fatal("private Evidence boundary was weakened")
	}
}

func TestInspectApplicationBindsItsActualOperationToStorage(t *testing.T) {
	path := storageTestRoot(t)
	storage, err := NewLocal(context.Background(), path, storageFixturePolicy(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	evidence, err := evidencelocal.NewStore(filepath.Join(path, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, _ := fakeworkflow.New(fakeworkflow.Safe)
	service, err := application.NewInspectService(fixture, fixture, fixture, evidence, policy.M1{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := fixture.Reference()
	request, _ := application.NewInspectRequest(reference)
	result, err := service.Inspect(ports.WithStorage(context.Background(), storage), request)
	if err != nil {
		t.Fatal(err)
	}
	run, ok := result.RunID()
	if !ok {
		t.Fatal("completed Run missing")
	}
	record := storage.entries[resourceKey("evidence", run.String())]
	if !storage.inspected[run] {
		t.Fatal("application did not record completed inspection for last-use finalization")
	}
	if record.Operation != result.OperationID().String() || record.Run != run.String() || record.Phase != "RETAINED" || len(result.Evidence()) != 2 {
		t.Fatal("storage bookkeeping lost actual operation/Run binding")
	}
}

// A storage registration boundary gives a deterministic path-replacement race.
// The existing writer must keep its original directory descriptor throughout
// publication and failure cleanup, rather than following the reused pathname.
type swappedEvidenceStorage struct {
	base     *Local
	sentinel string
	replace  bool
}

func (s *swappedEvidenceStorage) Policy() domain.StoragePolicy { return s.base.Policy() }
func (s *swappedEvidenceStorage) Reserve(ctx context.Context, request ports.StorageRequest) (ports.StorageLease, error) {
	lease, err := s.base.Reserve(ctx, request)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(s.base.path, "evidence")
	if s.replace {
		if err := os.Rename(root, filepath.Join(s.base.path, "evidence-before")); err != nil {
			_ = lease.Close()
			return nil, err
		}
	}
	foreign := filepath.Join(root, request.Temporary)
	if err := os.MkdirAll(foreign, 0700); err != nil {
		_ = lease.Close()
		return nil, err
	}
	s.sentinel = filepath.Join(foreign, "keep.txt")
	if err := os.WriteFile(s.sentinel, []byte("foreign bytes"), 0600); err != nil {
		_ = lease.Close()
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(foreign, ".fixture-integrity-evidence.tmp"), []byte("foreign conflict"), 0600); err != nil {
		_ = lease.Close()
		return nil, err
	}
	return lease, nil
}
func TestEvidenceWriterDoesNotDeleteForeignTreeAfterRootReplacement(t *testing.T) {
	for _, replace := range []bool{true, false} {
		t.Run(map[bool]string{true: "root-replacement", false: "unknown-member"}[replace], func(t *testing.T) {
			path := storageTestRoot(t)
			storage, err := openTestStorage(context.Background(), path, storageFixturePolicy(t, 2))
			if err != nil {
				t.Fatal(err)
			}
			defer storage.Close()
			evidence, err := evidencelocal.NewStore(filepath.Join(path, "evidence"))
			if err != nil {
				t.Fatal(err)
			}
			fixture, _ := fakeworkflow.New(fakeworkflow.Safe)
			service, err := application.NewInspectService(fixture, fixture, fixture, evidence, policy.M1{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			reference, _ := fixture.Reference()
			request, _ := application.NewInspectRequest(reference)
			swapped := &swappedEvidenceStorage{base: storage, replace: replace}
			result, err := service.Inspect(ports.WithStorage(context.Background(), swapped), request)
			if err == nil || result.Status() != domain.OperationFailed {
				t.Fatal("changed storage root became completed inspection")
			}
			if swapped.sentinel == "" {
				t.Fatal("replacement boundary was not reached")
			}
			body, readErr := os.ReadFile(swapped.sentinel)
			if readErr != nil || string(body) != "foreign bytes" {
				t.Fatal("failure cleanup deleted or changed a replacement tree", readErr)
			}
		})
	}
}

func TestStorageCompositionIsInertAndClosedCannotOpen(t *testing.T) {
	path := storageTestRoot(t)
	storage, err := NewLocal(context.Background(), path, storageFixturePolicy(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("composition created storage before a request")
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	if err := storage.open(context.Background()); err == nil {
		t.Fatal("closed storage reopened")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("closed storage mutated filesystem")
	}
}

func TestNPMIntakeRegistersBeforeDownloadAndKeepsSuccessfulPayload(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "download-failure"}[failure], func(t *testing.T) {
			path := storageTestRoot(t)
			spec := domain.DefaultStoragePolicy().Spec()
			spec.MinimumFreeBytes = 1
			policy, err := domain.NewStoragePolicy(spec)
			if err != nil {
				t.Fatal(err)
			}
			storage, err := NewLocal(context.Background(), path, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer storage.Close()
			run, _ := domain.NewRunID()
			operation, _ := domain.NewOperationID()
			body := []byte("small controlled artifact")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				storage.mu.Lock()
				registered := false
				for _, resource := range storage.entries {
					if resource.Run == run.String() && resource.Kind == "intake" && resource.Phase == "IN_USE" {
						registered = true
					}
				}
				storage.mu.Unlock()
				if !registered {
					t.Error("artifact download started before durable intake registration")
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				if failure {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				_, _ = w.Write(body)
			}))
			defer server.Close()
			client, err := artifactnpm.NewResolver(server.URL, server.Client(), filepath.Join(path, "intake"))
			if err != nil {
				t.Fatal(err)
			}
			source, _ := domain.NewSourceID("npm")
			identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.0.0", "tarball")
			sum := sha512.Sum512(body)
			resolved, err := domain.NewResolvedArtifact(identity, server.URL+"/tiny/-/tiny-1.0.0.tgz", "sha512-"+base64.StdEncoding.EncodeToString(sum[:]))
			if err != nil {
				t.Fatal(err)
			}
			ctx := ports.WithStorageOperation(ports.WithStorage(context.Background(), storage), operation)
			artifact, err := client.Acquire(ctx, run, resolved)
			if failure {
				if err == nil {
					t.Fatal("download failure became an acquired Artifact")
				}
				for _, resource := range storage.entries {
					if resource.Run == run.String() && resource.Kind == "intake" {
						t.Fatal("confirmed failed intake remained charged")
					}
				}
				if _, err := os.Stat(filepath.Join(path, "intake", run.String())); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed intake was not removed")
				}
				return
			}
			if err != nil || artifact.ContentHandle() == "" {
				t.Fatal(err)
			}
			registered := false
			for _, resource := range storage.entries {
				if resource.Run == run.String() && resource.Kind == "intake" {
					if resource.Phase != "RETAINED" || resource.LogicalBytes != uint64(len(body)) || resource.Content == "" {
						t.Fatal("intake bytes were not sealed/accounted before returning")
					}
					registered = true
				}
			}
			if !registered {
				t.Fatal("successful intake is absent from lifetime accounting")
			}
			// Intake and Evidence intentionally share a security Run, not one resource key.
			check, _ := domain.NewCheckID("fixture-intake-proof")
			factID, _ := domain.NewEvidenceID("fixture-intake-proof")
			fact, err := domain.NewEvidence(factID, check, artifact.Identity(), artifact.Digest(), "fixture", "Controlled acquired bytes.")
			if err != nil {
				t.Fatal(err)
			}
			evidence, err := evidencelocal.NewStore(filepath.Join(path, "evidence"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := evidence.Record(ctx, run, []domain.Evidence{fact}); err != nil {
				t.Fatal("same-Run intake/Evidence collision or wrong subquota", err)
			}
			if len(storage.entries) != 2 {
				t.Fatal("resource classes overwrote each other")
			}
			if _, err := os.Stat(filepath.Join(path, "intake", run.String(), "tarball.tgz")); err != nil {
				t.Fatal("intake was deleted before inspection/staging", err)
			}
			payload := filepath.Join(path, "intake", run.String(), "tarball.tgz")
			if err := storage.Finish(ctx, true); err == nil {
				t.Fatal("unobserved intake was finalized")
			}
			if _, err := os.Stat(payload); err != nil {
				t.Fatal("unobserved payload deleted", err)
			}
			if err := storage.NoteInspection(ctx, run, true); err != nil {
				t.Fatal(err)
			}
			if err := storage.Finish(ctx, false); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(payload); err != nil {
				t.Fatal("failed outer command lost its protected intake", err)
			}
			if err := storage.Finish(ctx, true); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(payload); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("completed last-use intake remained")
			}
			if len(storage.entries) != 1 || storage.evidenceRuns != 1 {
				t.Fatal("intake finalization removed Evidence or retained payload accounting")
			}
			if err := storage.Finish(ctx, true); err != nil {
				t.Fatal("repeated finalization is not idempotent", err)
			}

		})
	}
}

func TestIntakeFinalizationProtectsChangedContentAndOtherOwners(t *testing.T) {
	for _, mode := range []string{"content", "extra-member", "hardlink", "replacement", "restarted-owner"} {
		t.Run(mode, func(t *testing.T) {
			path := storageTestRoot(t)
			storage, err := openTestStorage(context.Background(), path, storageFixturePolicy(t, 4))
			if err != nil {
				t.Fatal(err)
			}
			defer storage.Close()
			run, _ := domain.NewRunID()
			operation, _ := domain.NewOperationID()
			if err := storage.intake.Mkdir(run.String(), 0700); err != nil {
				t.Fatal(err)
			}
			ctx := ports.WithStorageOperation(context.Background(), operation)
			lease, err := storage.Reserve(ctx, ports.StorageRequest{Operation: operation, Run: run, Kind: "intake", Temporary: run.String(), Target: run.String(), Bytes: 128, Files: 1})
			if err != nil {
				t.Fatal(err)
			}
			payload := filepath.Join(path, "intake", run.String(), "payload")
			if err := os.WriteFile(payload, []byte("sealed bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := lease.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if err := lease.Close(); err != nil {
				t.Fatal(err)
			}
			if err := storage.NoteInspection(ctx, run, true); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "content":
				if err := os.WriteFile(payload, []byte("changed data"), 0600); err != nil {
					t.Fatal(err)
				}
			case "extra-member":
				if err := os.WriteFile(filepath.Join(filepath.Dir(payload), "keep"), []byte("foreign"), 0600); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(payload, filepath.Join(path, "foreign-link")); err != nil {
					t.Fatal(err)
				}
			case "replacement":
				if err := os.Rename(filepath.Dir(payload), filepath.Join(path, "held-original")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Dir(payload), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(payload, []byte("foreign bytes"), 0600); err != nil {
					t.Fatal(err)
				}
			case "restarted-owner":
				if err := storage.Close(); err != nil {
					t.Fatal(err)
				}
				storage, err = openTestStorage(context.Background(), path, storageFixturePolicy(t, 4))
				if err != nil {
					t.Fatal(err)
				}
				defer storage.Close()
			}
			err = storage.Finish(ctx, true)
			if mode != "restarted-owner" && err == nil {
				t.Fatal("uncertain or altered intake was reclaimed")
			}
			if _, err := os.Stat(payload); err != nil {
				t.Fatal("protected payload was deleted", err)
			}
			if len(storage.entries) != 1 || storage.charged == 0 {
				t.Fatal("protected intake lost accounting")
			}
		})
	}
}

type finalizationReadbackContext struct {
	context.Context
	journal, payload string
	checks           int
	mutated          bool
}

func (c *finalizationReadbackContext) Err() error {
	body, err := os.ReadFile(c.journal)
	if err == nil && bytes.Contains(body, []byte(`"action":"deleting"`)) {
		c.checks++
		if c.checks == 3 {
			if err := os.WriteFile(c.payload, []byte("changed data"), 0600); err != nil {
				return err
			}
			c.mutated = true
		}
	}
	return c.Context.Err()
}
func TestIntakeFinalizationRechecksMemberAfterWholeInventory(t *testing.T) {
	path := storageTestRoot(t)
	storage, err := openTestStorage(context.Background(), path, storageFixturePolicy(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	run, _ := domain.NewRunID()
	operation, _ := domain.NewOperationID()
	if err := storage.intake.Mkdir(run.String(), 0700); err != nil {
		t.Fatal(err)
	}
	ctx := ports.WithStorageOperation(context.Background(), operation)
	lease, err := storage.Reserve(ctx, ports.StorageRequest{Operation: operation, Run: run, Kind: "intake", Temporary: run.String(), Target: run.String(), Bytes: 128, Files: 1})
	if err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(path, "intake", run.String(), "payload")
	if err := os.WriteFile(payload, []byte("sealed bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := lease.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := storage.NoteInspection(ctx, run, true); err != nil {
		t.Fatal(err)
	}
	callback := &finalizationReadbackContext{Context: ctx, journal: filepath.Join(path, ".storage", "resources.jsonl"), payload: payload}
	err = storage.Finish(callback, true)
	if !callback.mutated {
		t.Fatal("post-inventory boundary was not reached")
	}
	if err == nil {
		t.Fatal("post-inventory same-inode mutation was deleted")
	}
	body, readErr := os.ReadFile(payload)
	if readErr != nil || string(body) != "changed data" {
		t.Fatal("changed data was removed", readErr)
	}
}
