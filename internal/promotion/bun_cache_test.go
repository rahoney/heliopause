package promotion

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	evidencelocal "github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

type bunCacheLayoutFixture struct {
	layout artifactnpm.BunInspectionLayout
	err    error
}

func (f bunCacheLayoutFixture) PreparedLayoutFor(domain.ProjectDependencySnapshot, map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.BunInspectionLayout, error) {
	return f.layout, f.err
}

type bunCacheFixture struct {
	cache        *BunVerifiedCache
	set          domain.ProjectVerifiedSet
	archive, run string
	layout       artifactnpm.BunInspectionLayout
}

// Explicit structural source/SRI fixture, not public registry authentication.
// Static bytes and local Evidence storage below are real; actual native/source
// qualification is a separate bootstrap integration consumer.
func newBunCacheFixture(t *testing.T, empty bool) bunCacheFixture {
	t.Helper()
	return newBunCacheFixtureWithAliases(t, empty, false)
}

func newBunCacheFixtureWithAliases(t *testing.T, empty, aliases bool) bunCacheFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	intake, evidenceRoot, cacheRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache")
	manifest := []byte(`{"name":"haa-bun-single","scripts":{"install":"exit 99"},"dependencies":{"is-number":"6.0.0"}}`)
	lock, err := os.ReadFile("../artifact/npm/testdata/bun-lock/single.bun.lock")
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		manifest = []byte(`{}`)
		lock = nil
	}
	lockControl, err := domain.NewProjectControlFile("bun.lock", lock, !empty)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := artifactnpm.ParseBunLock(manifest, lockControl, runtimeidentity.BunVersion)
	if err != nil {
		t.Fatal(err)
	}
	client, err := artifactnpm.NewResolver("https://registry.npmjs.org/", &http.Client{Transport: pnpmCacheTransport(func(request *http.Request) (*http.Response, error) {
		if empty || request.URL.Path != "/is-number/6.0.0" {
			t.Fatal("unexpected fixture source request")
		}
		body, err := json.Marshal(map[string]any{"name": "is-number", "version": "6.0.0", "dist": map[string]string{"tarball": "https://registry.npmjs.org/is-number/-/is-number-6.0.0.tgz", "integrity": parsed.Packages()[0].Integrity}})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(body))}, nil
	})}, intake)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := client.BindBunLock(context.Background(), parsed)
	if err != nil {
		t.Fatal(err)
	}
	manifestControl, err := domain.NewProjectControlFile("package.json", manifest, true)
	if err != nil {
		t.Fatal(err)
	}
	controls := []domain.ProjectControlFile{manifestControl, lockControl}
	target, err := domain.NewInstallTarget(filepath.Join(root, "project"))
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := selection.InspectionContext(install, controls, "controlled-bun-runtime")
	if err != nil {
		t.Fatal(err)
	}
	store, err := evidencelocal.NewStore(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	fixture := bunCacheFixture{}
	inspections := []domain.DependencyInspection{}
	if !empty {
		packageJSON := []byte(`{"name":"is-number","version":"6.0.0"}`)
		var compressed bytes.Buffer
		gzipWriter := gzip.NewWriter(&compressed)
		writer := tar.NewWriter(gzipWriter)
		if err := writer.WriteHeader(&tar.Header{Name: "package/package.json", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(packageJSON))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(packageJSON); err != nil {
			t.Fatal(err)
		}
		if aliases {
			for _, name := range []string{"LICENSE", "COPY"} {
				body := []byte("controlled shared license\n")
				if err := writer.WriteHeader(&tar.Header{Name: "package/" + name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
					t.Fatal(err)
				}
				if _, err := writer.Write(body); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gzipWriter.Close(); err != nil {
			t.Fatal(err)
		}
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		fixture.run = run.String()
		subjectDigest, err := domain.NewSHA256Digest(pnpmCacheHash(compressed.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		subject, err := domain.NewAcquiredArtifactWithIntegrity(selection.Artifacts()[0].Identity(), subjectDigest, "intake:"+run.String()+":tarball", uint64(compressed.Len()), parsed.Packages()[0].Integrity, parsed.Packages()[0].Integrity)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(intake, run.String()), 0o700); err != nil {
			t.Fatal(err)
		}
		fixture.archive = filepath.Join(intake, run.String(), "tarball.tgz")
		if err := os.WriteFile(fixture.archive, compressed.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		node, err := domain.ProjectDependencyNodeID(subject.Identity())
		if err != nil {
			t.Fatal(err)
		}
		input, err := frozen.BindAcquired(frozen.Snapshot(), map[domain.DependencyNodeID]domain.AcquiredArtifact{node: subject})
		if err != nil {
			t.Fatal(err)
		}
		static, err := inspectionnpm.NewInspector(intake)
		if err != nil {
			t.Fatal(err)
		}
		inventory, report, err := static.InspectArchive(context.Background(), subject)
		if err != nil || len(report.Findings()) != 0 {
			t.Fatal(err)
		}
		bodies := map[string][]byte{"node_modules/is-number/package.json": packageJSON}
		if aliases {
			for _, name := range []string{"COPY", "LICENSE"} {
				bodies["node_modules/is-number/"+name] = []byte("controlled shared license\n")
			}
		}
		var native bytes.Buffer
		tw := tar.NewWriter(&native)
		for _, name := range []string{"node_modules", "node_modules/is-number"} {
			if err := tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o755, Uid: 1000, Gid: 1000}); err != nil {
				t.Fatal(err)
			}
		}
		for name, body := range bodies {
			if err := tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write(body); err != nil {
				t.Fatal(err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		nativeView, err := artifactnpm.ParseBunLayoutArchive(context.Background(), native.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		fixture.layout, err = input.BindLayout(context.Background(), []artifactnpm.ArchiveInventory{inventory}, nativeView)
		if err != nil {
			t.Fatal(err)
		}
		session, err := domain.NewSandboxSessionID()
		if err != nil {
			t.Fatal(err)
		}
		result, err := domain.NewSandboxResult(session, domain.SandboxCompleted, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		summary, err := result.ObservationSummary()
		if err != nil {
			t.Fatal(err)
		}
		key := pnpmCacheHash([]byte(parsed.Packages()[0].Key))
		factCheck, err := domain.NewCheckID("bun-instance-facts-" + key[:24])
		if err != nil {
			t.Fatal(err)
		}
		factExecution, err := domain.NewCheckExecution(factCheck, domain.CheckInspection, false, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		if err != nil {
			t.Fatal(err)
		}
		binding, _ := json.Marshal(pnpmInstanceBinding{key, session.String(), fixture.layout.Digest().String(), pnpmCacheHash([]byte(frozen.RuntimeIdentity()))})
		facts := []domain.Evidence{}
		for _, item := range []struct{ kind, summary string }{{"bun-instance-binding", string(binding)}, {"bun-instance-observations", summary}} {
			id, err := domain.NewEvidenceID(item.kind)
			if err != nil {
				t.Fatal(err)
			}
			fact, err := domain.NewEvidence(id, factCheck, subject.Identity(), subject.Digest(), item.kind, item.summary)
			if err != nil {
				t.Fatal(err)
			}
			facts = append(facts, fact)
		}
		factReport, err := domain.NewInspectionReport(factExecution, nil, facts)
		if err != nil {
			t.Fatal(err)
		}
		coverageID, _ := domain.NewCheckID("bun-dynamic-lifecycle")
		coverageCheck, _ := domain.NewCheckExecution(coverageID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		coverageEvidenceID, _ := domain.NewEvidenceID("bun-instance-coverage")
		coverageFact, err := domain.NewEvidence(coverageEvidenceID, coverageID, subject.Identity(), subject.Digest(), "bun-instance-coverage", "expected=1 attempted=1 completed=1")
		if err != nil {
			t.Fatal(err)
		}
		coverageReport, err := domain.NewInspectionReport(coverageCheck, nil, []domain.Evidence{coverageFact})
		if err != nil {
			t.Fatal(err)
		}
		combined, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{report, factReport, coverageReport})
		if err != nil {
			t.Fatal(err)
		}
		verified, err := (verificationnpm.IntegrityVerifier{}).Verify(context.Background(), subject)
		if err != nil {
			t.Fatal(err)
		}
		evidence := append(verified.Evidence(), combined.Evidence()...)
		refs, err := store.Record(context.Background(), run, evidence)
		if err != nil {
			t.Fatal(err)
		}
		policyInput, err := domain.NewPolicyInput(run, subject, verified, combined, refs)
		if err != nil {
			t.Fatal(err)
		}
		decision, err := (policy.M3{}).Evaluate(policyInput)
		if err != nil {
			t.Fatal(err)
		}
		inspection, err := domain.NewDependencyInspection(node, run, subject, append([]domain.CheckExecution{verified.Execution()}, combined.Executions()...), refs, decision)
		if err != nil {
			t.Fatal(err)
		}
		inspection, err = inspection.WithHostInstallActionEvidence(evidence)
		if err != nil {
			t.Fatal(err)
		}
		inspections = append(inspections, inspection)
	}
	set, err := domain.NewInspectedProjectSet(frozen.Snapshot(), inspections)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := (policy.M4{}).EvaluateProjectSet(set)
	if err != nil {
		t.Fatal(err)
	}
	fixture.set, err = domain.NewProjectVerifiedSet(set, decision)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	fixture.cache, err = NewBunVerifiedCache(intake, evidenceRoot, cacheRoot, frozen, store, bunCacheLayoutFixture{layout: fixture.layout}, evidence.Generator{}, operation)
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestBunApprovedDataCacheStagesWholeReceiptAndRejectsDrift(t *testing.T) {
	for _, mutation := range []string{"normal", "archive", "control", "receipt", "extra-file", "extra-root", "mode", "symlink", "hardlink", "evidence", "missing-layout", "source", "manifest", "sbom", "provenance", "missing-record", "foreign-record", "operation"} {
		t.Run(mutation, func(t *testing.T) {
			f := newBunCacheFixture(t, false)
			ctx := context.Background()
			if mutation == "missing-layout" {
				f.cache.layout = bunCacheLayoutFixture{err: errors.New("controlled handoff failure")}
			}
			if mutation == "source" {
				if err := os.WriteFile(f.archive, []byte("changed source"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			staged, err := f.cache.StageProject(ctx, f.set)
			if mutation == "missing-layout" || mutation == "source" {
				if err == nil || staged.Valid() {
					t.Fatal("bad preflight staged")
				}
				if _, err := os.Stat(f.cache.cacheRoot); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("bad preflight wrote cache", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := f.cache.OpenProjectCache(ctx, staged)
			if err != nil {
				t.Fatal(err)
			}
			if mutation == "normal" {
				body, err := os.ReadFile(filepath.Join(data, pnpmCacheArchiveName))
				if err != nil {
					t.Fatal(err)
				}
				view, err := artifactnpm.ParsePNPMLayoutArchive(ctx, body)
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, entry := range view.Entries() {
					if entry.Name == "node_modules/is-number/package.json" && entry.Kind == tar.TypeReg {
						found = true
					}
				}
				if !found {
					t.Fatal("bound Bun source file lost in data tar")
				}
				return
			}
			file := filepath.Join(data, pnpmCacheArchiveName)
			switch mutation {
			case "archive", "control", "manifest", "sbom", "provenance":
				if mutation == "control" {
					file = filepath.Join(data, "package.json")
				}
				if name := map[string]string{"manifest": "manifest.json", "sbom": "sbom.cdx.json", "provenance": "bun-provenance.json"}[mutation]; name != "" {
					file = filepath.Join(filepath.Dir(data), "records", name)
				}
				if err := os.Chmod(file, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte("changed"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(file, 0o444); err != nil {
					t.Fatal(err)
				}
			case "receipt":
				file = filepath.Join(filepath.Dir(data), "receipt.json")
				if err := os.Chmod(file, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(`{}`), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(file, 0o444); err != nil {
					t.Fatal(err)
				}
			case "extra-file":
				if err := os.WriteFile(filepath.Join(data, "foreign"), []byte("foreign"), 0o444); err != nil {
					t.Fatal(err)
				}
			case "extra-root":
				if err := os.WriteFile(filepath.Join(filepath.Dir(data), "foreign"), []byte("foreign"), 0o444); err != nil {
					t.Fatal(err)
				}
			case "mode":
				if err := os.Chmod(file, 0o644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(f.archive, file); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(file, filepath.Join(filepath.Dir(f.cache.cacheRoot), "external-link")); err != nil {
					t.Fatal(err)
				}
			case "evidence":
				if err := os.RemoveAll(filepath.Join(f.cache.evidenceRoot, f.run)); err != nil {
					t.Fatal(err)
				}
			case "missing-record":
				if err := os.Remove(filepath.Join(filepath.Dir(data), "records", "manifest.json")); err != nil {
					t.Fatal(err)
				}
			case "foreign-record":
				if err := os.WriteFile(filepath.Join(filepath.Dir(data), "records", "foreign"), []byte("foreign"), 0o444); err != nil {
					t.Fatal(err)
				}
			case "operation":
				f.cache.operation, err = domain.NewOperationID()
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := f.cache.OpenProjectCache(ctx, staged); err == nil {
				t.Fatal("changed cache admitted", mutation)
			}
		})
	}
}

func TestBunApprovedEmptyDataCacheNeverRequestsLayout(t *testing.T) {
	f := newBunCacheFixture(t, true)
	f.cache.layout = bunCacheLayoutFixture{err: errors.New("empty layout must not be requested")}
	staged, err := f.cache.StageProject(context.Background(), f.set)
	if err != nil {
		t.Fatal(err)
	}
	directory, err := f.cache.OpenProjectCache(context.Background(), staged)
	if err != nil {
		t.Fatal(err)
	}
	approval, _, err := f.cache.approval(context.Background(), f.set)
	if err != nil || len(approval.Controls) != 2 || approval.Controls[1].Path != "bun.lock" || approval.Controls[1].Kind != "absent" {
		t.Fatal("absent Bun lock lost in approval receipt", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != "package.json" {
		t.Fatal("empty cache manufactured package data", err)
	}
}

func TestBunStagingLateFailurePreservesCauseAndRemovesPartialCache(t *testing.T) {
	f := newBunCacheFixture(t, false)
	cause := errors.New("controlled staging failure")
	f.cache.beforeTreeSync = func(string) error { return cause }
	staged, err := f.cache.StageProject(context.Background(), f.set)
	if !errors.Is(err, cause) || staged.Valid() {
		t.Fatal("late staging failure lost cause or returned a value", err)
	}
	entries, err := os.ReadDir(f.cache.cacheRoot)
	if err != nil || len(entries) != 0 {
		t.Fatal("partial data cache retained", err)
	}
}

func TestBunProjectRecordFailurePreventsAllCacheWrites(t *testing.T) {
	f := newBunCacheFixture(t, false)
	cause := errors.New("controlled Manifest generation failure")
	f.cache.recordBuilder = bunFailedRecordBuilder{cause}
	staged, err := f.cache.StageProject(context.Background(), f.set)
	if !errors.Is(err, cause) || staged.Valid() {
		t.Fatal("record failure lost cause or staged", err)
	}
	if _, err := os.Stat(f.cache.cacheRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("record failure wrote cache", err)
	}
}

func TestBunProjectRecordsBindActualOperationAndCompleteNativeContext(t *testing.T) {
	for _, empty := range []bool{false, true} {
		f := newBunCacheFixture(t, empty)
		staged, err := f.cache.StageProject(context.Background(), f.set)
		if err != nil {
			t.Fatal(err)
		}
		data, err := f.cache.OpenProjectCache(context.Background(), staged)
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			ID        string            `json:"manifest_id"`
			Operation string            `json:"operation_id"`
			Entries   []json.RawMessage `json:"entries"`
		}
		body, err := os.ReadFile(filepath.Join(filepath.Dir(data), "records", "manifest.json"))
		if err != nil || json.Unmarshal(body, &manifest) != nil || manifest.ID == "" || manifest.Operation != f.cache.operation.String() || len(manifest.Entries) != len(f.set.Inspected().Inspections()) {
			t.Fatal("Manifest lost actual project binding", err)
		}
		var provenance bunProjectProvenance
		body, err = os.ReadFile(filepath.Join(filepath.Dir(data), "records", "bun-provenance.json"))
		if err != nil || json.Unmarshal(body, &provenance) != nil {
			t.Fatal("provenance unavailable", err)
		}
		approval, _, err := f.cache.approval(context.Background(), f.set)
		if err != nil {
			t.Fatal(err)
		}
		want, err := json.Marshal(bunProjectProvenance{"helox.bun-project-provenance/v1", f.cache.operation.String(), manifest.ID, approval.Graph, approval.Runtime, approval.Layout, f.cache.frozen.Lock().Roots(), f.cache.frozen.Lock().Packages()})
		if err != nil || !bytes.Equal(want, body) {
			t.Fatal("complete native context changed", err)
		}
		body, err = os.ReadFile(filepath.Join(filepath.Dir(data), "records", "sbom.cdx.json"))
		if err != nil || !json.Valid(body) || !bytes.Contains(body, []byte(manifest.ID)) {
			t.Fatal("SBOM lost Manifest binding", err)
		}
	}
}

type bunChangedEvidenceReader struct {
	base     ProjectEvidenceReader
	mutation string
}

func (r bunChangedEvidenceReader) ReadReference(ctx context.Context, run domain.RunID, reference domain.EvidenceReference, identity domain.ResolvedArtifactIdentity, digest domain.ContentDigest) (domain.Evidence, domain.ContentDigest, error) {
	fact, recordDigest, err := r.base.ReadReference(ctx, run, reference, identity, digest)
	if err != nil {
		return fact, recordDigest, err
	}
	kind, summary := fact.Kind(), fact.Summary()
	switch {
	case kind == "bun-instance-binding":
		var binding pnpmInstanceBinding
		if err := json.Unmarshal([]byte(summary), &binding); err != nil {
			return fact, recordDigest, err
		}
		if r.mutation == "runtime" {
			binding.Runtime = strings.Repeat("f", 64)
		}
		if r.mutation == "layout" {
			binding.Layout = strings.Repeat("f", 64)
		}
		if r.mutation == "session" {
			binding.Session = "sbx_invalid"
		}
		body, err := json.Marshal(binding)
		if err != nil {
			return fact, recordDigest, err
		}
		summary = string(body)
	case kind == "bun-instance-coverage" && r.mutation == "coverage":
		summary = "expected=2 attempted=1 completed=1"
	case kind == "bun-instance-coverage" && r.mutation == "missing-coverage":
		kind = "controlled-other-fact"
	case kind == "bun-instance-observations" && r.mutation == "partial":
		summary = `{"session":"sbx_invalid","status":"INCOMPLETE"}`
	}
	changed, err := domain.NewEvidence(fact.ID(), fact.CheckID(), fact.Identity(), fact.Digest(), kind, summary)
	return changed, recordDigest, err
}

func TestBunStagingRechecksRecordedCoverageBeforeAnyCacheWrites(t *testing.T) {
	// This deliberately adverse reader is a contract fixture. It does not claim
	// a real local Store can return modified bytes with an unchanged record hash.
	for _, mutation := range []string{"runtime", "layout", "session", "coverage", "missing-coverage", "partial", "missing-record"} {
		t.Run(mutation, func(t *testing.T) {
			f := newBunCacheFixture(t, false)
			if mutation == "missing-record" {
				if err := os.RemoveAll(filepath.Join(f.cache.evidenceRoot, f.run)); err != nil {
					t.Fatal(err)
				}
			} else {
				f.cache.evidence = bunChangedEvidenceReader{f.cache.evidence, mutation}
			}
			staged, err := f.cache.StageProject(context.Background(), f.set)
			if err == nil || staged.Valid() {
				t.Fatal("invalid recorded approval staged", mutation)
			}
			if _, err := os.Lstat(f.cache.cacheRoot); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid recorded approval wrote cache", err)
			}
		})
	}
}

type bunFailedRecordBuilder struct{ cause error }

func (b bunFailedRecordBuilder) BuildProjectWithControlPresence(context.Context, domain.OperationID, domain.ProjectVerifiedSet, []domain.LockedDependency, []domain.DependencyEdge, []domain.ProjectControlFile, string) (domain.ContentDigest, []byte, []byte, error) {
	return domain.ContentDigest{}, nil, nil, b.cause
}
