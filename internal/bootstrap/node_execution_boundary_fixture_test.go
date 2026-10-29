//go:build linux && realcorpus

package bootstrap

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

const nodeBoundaryPackage = "haa-node-runtime-fixture"
const nodeBoundaryURL = "https://registry.npmjs.org/haa-node-runtime-fixture/-/haa-node-runtime-fixture-1.0.0.tgz"

const nodeBoundaryScript = "#!/usr/bin/env node\n'use strict';\n" +
	"const fs = require('node:fs');\n" +
	"switch (process.argv[2]) {\n" +
	"case 'argv': fs.writeSync(1, JSON.stringify(process.argv.slice(3))); break;\n" +
	"case 'term': fs.writeSync(1, 'signal\\n'); process.kill(process.pid, 'SIGTERM'); break;\n" +
	"case 'kill': fs.writeSync(1, 'signal\\n'); process.kill(process.pid, 'SIGKILL'); break;\n" +
	"case 'cancel': case 'deadline': fs.writeSync(1, 'ready\\n'); setInterval(() => {}, 1000); break;\n" +
	"case 'stdout-overflow': for (;;) fs.writeSync(1, Buffer.alloc(65536, 97));\n" +
	"case 'stderr-overflow': for (;;) fs.writeSync(2, Buffer.alloc(65536, 98));\n" +
	"case 'outside-read': fs.writeSync(1, 'fs-ready\\n'); try { fs.readFileSync('/home/haa-node-boundary-probe'); } catch {} break;\n" +
	"case 'network': fs.writeSync(1, 'network-ready\\n'); { const s = require('node:net').connect(9, '127.0.0.1'); s.on('error', () => {}); } break;\n" +
	"case 'child': fs.writeSync(1, 'child-ready\\n'); { const c = require('node:child_process').spawn(process.execPath, ['-e', 'setInterval(() => {}, 1000)'], {stdio: ['ignore', 'inherit', 'inherit']}); c.on('error', () => {}); c.unref(); } break;\n" +
	"case 'comm-spoof': process.title = 'haa-boundary'; fs.writeSync(1, 'spoof-ready\\n'); try { fs.readFileSync('/proc/self/status'); } catch {} break;\n" +
	"case 'readonly': { try { fs.writeFileSync(__filename, 'unexpected-write'); process.exitCode = 37; } catch (e) { if (!['EROFS', 'EACCES', 'EPERM'].includes(e.code)) throw e; fs.writeSync(1, 'readonly-denied\\n'); } } break;\n" +
	"default: process.exit(37);\n}\n"

// Only the existing explicitly injected test resolver receives these bytes.
// This is controlled fixture selection, not public registry authentication.
type nodeBoundaryTransport struct{ metadata, archive []byte }

func (f nodeBoundaryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := request.Context().Err(); err != nil {
		return nil, err
	}
	if request.Method != http.MethodGet || request.URL.Scheme != "https" || request.URL.Host != "registry.npmjs.org" || request.URL.RawQuery != "" {
		return nil, errors.New("unexpected controlled Node fixture request")
	}
	var body []byte
	contentType := "application/json"
	switch request.URL.String() {
	case "https://registry.npmjs.org/" + nodeBoundaryPackage:
		body = f.metadata
	case nodeBoundaryURL:
		body, contentType = f.archive, "application/octet-stream"
	default:
		return nil, errors.New("controlled Node fixture has no requested resource")
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Request: request}, nil
}

func nodeBoundarySource(t *testing.T, ctx context.Context, intake string) (*artifactnpm.Resolver, domain.DependencyResolution) {
	t.Helper()
	manifest, err := json.Marshal(map[string]any{"name": nodeBoundaryPackage, "version": "1.0.0", "bin": map[string]string{"haa-runtime-probe": "cli.js"}})
	if err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	writer := tar.NewWriter(compressed)
	for _, file := range []struct {
		name string
		mode int64
		body []byte
	}{{"package.json", 0o644, manifest}, {"cli.js", 0o755, []byte(nodeBoundaryScript)}} {
		if err := writer.WriteHeader(&tar.Header{Name: "package/" + file.name, Mode: file.mode, Typeflag: tar.TypeReg, Size: int64(len(file.body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(file.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha512.Sum512(archive.Bytes())
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
	metadata, err := json.Marshal(map[string]any{"name": nodeBoundaryPackage, "versions": map[string]any{"1.0.0": map[string]any{"name": nodeBoundaryPackage, "version": "1.0.0", "dist": map[string]string{"tarball": nodeBoundaryURL, "integrity": integrity}}}})
	if err != nil {
		t.Fatal(err)
	}
	source, err := artifactnpm.NewResolver("https://registry.npmjs.org/", &http.Client{Transport: nodeBoundaryTransport{metadata, archive.Bytes()}}, intake)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactnpm.ParseReference(nodeBoundaryPackage + "@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := source.Resolve(ctx, reference)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := json.Marshal(map[string]any{"lockfileVersion": 3, "packages": map[string]any{
		"":                                    map[string]any{"name": "haa-controlled-fixture", "version": "1.0.0", "dependencies": map[string]string{nodeBoundaryPackage: "1.0.0"}},
		"node_modules/" + nodeBoundaryPackage: map[string]any{"version": "1.0.0", "resolved": nodeBoundaryURL, "integrity": integrity},
	}})
	if err != nil {
		t.Fatal(err)
	}
	graph, err := artifactnpm.ParsePackageLockV3(reference, lock)
	if err != nil || len(graph.Nodes()) != 1 || graph.Nodes()[0].Artifact() != resolved {
		t.Fatal("controlled metadata and canonical lock differ", err)
	}
	digest := sha256.Sum256(lock)
	fingerprint, err := domain.NewSHA256Digest(hex.EncodeToString(digest[:]))
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := domain.NewDependencyResolution(graph, "controlled-node-boundary-fixture", fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	return source, resolution
}

func nodeBoundaryExecution(t *testing.T, ctx context.Context, root string, backend *sandbox.Backend, source *artifactnpm.Resolver, resolution domain.DependencyResolution, arguments []string) (domain.ExecutionContext, artifactnpm.NodeExecutionLayout) {
	t.Helper()
	static, err := inspectionnpm.NewInspector(root + "/intake")
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := inspectionnpm.NewDynamicInspector(backend)
	if err != nil {
		t.Fatal(err)
	}
	composite, err := inspectionnpm.NewCompositeInspector(static, dynamic)
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.NewStore(root + "/evidence")
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewExecutionInspectService(source, verificationnpm.IntegrityVerifier{}, composite, store, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := service.InspectResolved(ctx, resolution)
	if err != nil || !inspection.Valid() || inspection.Set().Decision().Decision() != domain.DecisionAllow {
		t.Fatal("controlled fixture lacks actual current whole Policy approval", err)
	}
	entries := inspection.Set().Inspected().Inspections()
	if len(entries) != 1 || entries[0].PolicyDecision().PolicyID() != "m3-npm-dynamic-inspect" || len(entries[0].Evidence()) < 3 || len(entries[0].Checks()) != 3 {
		t.Fatal("current fixture inspection facts missing")
	}
	for _, check := range entries[0].Checks() {
		if !check.Required() || check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted {
			t.Fatal("controlled fixture required check incomplete")
		}
	}
	measured, err := (nodeExecutionEntryBinder{static}).BindExecutionEntry(ctx, inspection, "haa-runtime-probe")
	if err != nil {
		t.Fatal(err)
	}
	execution, err := domain.NewExecutionContext(domain.ExecutionContextData{
		OperationID: inspection.OperationID(), Resolution: resolution, Set: inspection.Set(),
		EntryArtifact: measured.EntryArtifact(), Command: measured.Command(), EntryPath: measured.EntryPath(),
		EntryDigest: measured.EntryDigest(), EntrySize: measured.EntrySize(), RuntimeIdentity: measured.RuntimeIdentity(), Arguments: arguments,
	})
	if err != nil {
		t.Fatal(err)
	}
	layout, err := materializeNodeExecution(ctx, backend, static, execution)
	if err != nil || !layout.MatchesExecution(execution) {
		t.Fatal("controlled current whole layout failed", err)
	}
	return execution, layout
}
