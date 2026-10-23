package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) (string, string) {
	t.Helper()
	source, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	root := filepath.Join(base, "repository")
	state := filepath.Join(base, "state")
	if err := os.MkdirAll(state, 0o700); err != nil {
		t.Fatal(err)
	}
	paths := []string{runtimeFile, supportFile, generatedFile, "tools/pnpm/release-proof.json", "README.md", "scripts/tools.lock.json", "scripts/generate-runtime-lock.go", "tools/gvisor-observer/BUILD", "go.mod", "go.sum", ".github/workflows/heliopause-ci.yml", ".github/workflows/heliopause-security.yml", ".github/workflows/heliopause-release-build.yml", ".github/workflows/heliopause-release-publish.yml"}
	for _, path := range paths {
		body, err := os.ReadFile(filepath.Join(source, path))
		if err != nil {
			t.Fatal(err)
		}
		writeTest(t, filepath.Join(root, path), body)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	state, err = filepath.EvalSymlinks(state)
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "init", "-q")
	gitTest(t, root, "add", "--all")
	commitTest(t, root)
	return root, state
}
func writeTest(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}
func gitTest(t *testing.T, root string, args ...string) {
	t.Helper()
	if _, err := command(root, nil, "git", args...); err != nil {
		t.Fatal(err)
	}
}
func commitTest(t *testing.T, root string) {
	t.Helper()
	gitTest(t, root, "-c", "user.name=Runtime fixture", "-c", "user.email=runtime-fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "격리 fixture 기준선")
}
func noopPlan(t *testing.T, root, state, component string) plan {
	t.Helper()
	selected, err := selectedVersion(root, component)
	if err != nil {
		t.Fatal(err)
	}
	if component == "terraform" || component == "github" {
		selected = ""
	}
	p, err := makePlan(root, state, component, selected, "")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// These are deliberately synthetic future releases. They prove local coordination,
// not upstream availability, OCI execution, support decisions or qualification.
func fixtureMetadata(t *testing.T, component, version string) metadata {
	t.Helper()
	name, tag, err := imageName(component, version)
	if err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"schemaVersion":2,"config":{"digest":"sha256:` + strings.Repeat("a", 64) + `"},"layers":[{}]}`)
	digest := "sha256:" + hash(manifest)
	index := []byte(`{"schemaVersion":2,"manifests":[{"digest":"` + digest + `","platform":{"os":"linux","architecture":"amd64"}}]}`)
	base := "https://registry-1.docker.io/v2/library/" + name + "/manifests/"
	docs := []document{{base + tag, index}, {base + digest, manifest}}
	switch component {
	case "node":
		docs = append(docs, document{releaseURL(component, version), []byte(`[{"version":"v` + version + `","npm":"99.1.2"}]`)})
	case "python":
		docs = append(docs, document{releaseURL(component, version), []byte("_PIP_VERSION = \"99.1.2\"\n")})
	case "go":
		files := []map[string]any{}
		for _, platform := range []string{"linux/amd64", "darwin/amd64", "darwin/arm64"} {
			parts := strings.Split(platform, "/")
			files = append(files, map[string]any{"filename": "go" + version + "." + parts[0] + "-" + parts[1] + ".tar.gz", "os": parts[0], "arch": parts[1], "kind": "archive", "size": 123, "sha256": strings.Repeat("b", 64)})
		}
		body, err := json.Marshal([]map[string]any{{"version": "go" + version, "stable": true, "files": files}})
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, document{releaseURL(component, version), body})
	}
	m, err := metadataFromDocuments(component, version, docs)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func futurePlan(t *testing.T, root, state, component string) plan {
	t.Helper()
	p := noopPlan(t, root, state, component)
	p.Version = map[string]string{"go": "1.99.1", "node": "99.1.1", "python": "3.99.1", "rust": "1.999.1"}[component]
	p.Metadata = fixtureMetadata(t, component, p.Version)
	p.RequiredQualification = qualification(component)
	owners := map[string]string{}
	switch component {
	case "go":
		owners["Go"] = p.Version
	case "node":
		owners["Node"] = p.Version
		owners["npm"] = p.Metadata.Bundled
	case "python":
		owners["Python"] = p.Version
		owners["pip"] = p.Metadata.Bundled
	case "rust":
		owners["Rust/Cargo"] = p.Version
	}
	var support struct {
		Schema  int              `json:"schema_version"`
		Records []map[string]any `json:"records"`
	}
	if err := loadJSON(filepath.Join(root, supportFile), &support); err != nil {
		t.Fatal(err)
	}
	for _, record := range support.Records {
		component, _ := record["component"].(string)
		if version, ok := owners[component]; ok {
			record["pinned_version"] = version
			record["checked_at"] = time.Now().UTC().Format("2006-01-02")
			record["latest_stable_at_check"] = version
			record["reason"] = "Synthetic fixture only; no upstream support assertion"
			p.Review = append(p.Review, record)
		}
	}
	var err error
	p.Changes, err = renderChanges(p, state)
	if err != nil {
		t.Fatal(err)
	}
	p.ID = planID(p)
	return p
}

func TestSameVersionPlanAndApply(t *testing.T) {
	for _, component := range []string{"go", "node", "python", "rust", "uv"} {
		t.Run(component, func(t *testing.T) {
			root, state := fixture(t)
			p := noopPlan(t, root, state, component)
			secondState := filepath.Join(filepath.Dir(state), "second-plan")
			if err := os.Mkdir(secondState, 0o700); err != nil {
				t.Fatal(err)
			}
			secondPlan := noopPlan(t, root, secondState, component)
			if secondPlan.ID != p.ID {
				t.Fatal("same baseline and version produced different frozen plans")
			}
			if len(p.Changes) != 0 || len(p.Metadata.Documents) != 0 || len(p.RequiredQualification) != 0 {
				t.Fatal("same-version must not acquire, mutate or claim new qualification")
			}
			changes, err := renderChanges(p, state)
			if err != nil || !reflect.DeepEqual(p.Changes, changes) {
				t.Fatal("non-deterministic generation", err)
			}
			for i := 0; i < 2; i++ {
				if err := applyPlan(p, state, io.Discard, nil); err != nil {
					t.Fatal(err)
				}
			}
			if err := compareSnapshot(p); err != nil {
				t.Fatal(err)
			}
			status, err := command(filepath.Join(state, "candidate"), nil, "git", "status", "--porcelain")
			if err != nil || len(status) != 0 {
				t.Fatal("same-version diff", string(status), err)
			}
			if _, err := os.Stat(filepath.Join(state, "sdk")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("same-version created SDK")
			}
		})
	}
}
func TestCoordinatedComponentFixtures(t *testing.T) {
	for _, component := range []string{"go", "node", "python", "rust"} {
		t.Run(component, func(t *testing.T) {
			root, state := fixture(t)
			p := futurePlan(t, root, state, component)
			again, err := renderChanges(p, state)
			if err != nil || !reflect.DeepEqual(again, p.Changes) {
				t.Fatal("frozen patch changed", err)
			}
			paths := []string{}
			for _, c := range p.Changes {
				paths = append(paths, c.Path)
			}
			want := []string{generatedFile, runtimeFile, supportFile}
			if component == "go" {
				want = []string{".github/workflows/heliopause-ci.yml", ".github/workflows/heliopause-release-build.yml", ".github/workflows/heliopause-release-publish.yml", ".github/workflows/heliopause-security.yml", "README.md", "go.mod", generatedFile, runtimeFile, "scripts/tools.lock.json", supportFile}
			}
			if !reflect.DeepEqual(paths, want) {
				t.Fatalf("unexpected consumer inventory: %v", paths)
			}
			if err := applyPlan(p, state, io.Discard, nil); err != nil {
				t.Fatal(err)
			}
			candidate := filepath.Join(state, "candidate")
			selected, err := selectedVersion(candidate, component)
			if err != nil || selected != p.Version {
				t.Fatal("target owner mismatch", err)
			}
			if err := checkCandidate(p, candidate, true); err != nil {
				t.Fatal(err)
			}
			if err := compareSnapshot(p); err != nil {
				t.Fatal(err)
			}
			generated, err := os.ReadFile(filepath.Join(candidate, generatedFile))
			if err != nil {
				t.Fatal(err)
			}
			if component == "python" && !bytes.Contains(generated, []byte("cp399")) {
				t.Fatal("ABI not coordinated")
			}
			if component == "go" {
				if err := checkGo(filepath.Join(runtimeGoRoot(t), "bin/go"), candidate); err == nil {
					t.Fatal("old SDK accepted for new Go owner")
				}
			}
			p.Changes = p.Changes[1:]
			p.ID = planID(p)
			if err := applyPlan(p, state, io.Discard, nil); err == nil {
				t.Fatal("partial patch accepted")
			}
		})
	}
}
func runtimeGoRoot(t *testing.T) string {
	t.Helper()
	body, err := command(".", nil, "go", "env", "GOROOT")
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(body))
}

func TestInterruptedApplyRecovery(t *testing.T) {
	root, state := fixture(t)
	p := futurePlan(t, root, state, "python")
	injected := errors.New("injected write interruption")
	if err := applyPlan(p, state, io.Discard, func(int) error { return injected }); !errors.Is(err, injected) {
		t.Fatal(err)
	}
	var before journal
	if err := loadJSON(filepath.Join(state, "journal.json"), &before); err != nil {
		t.Fatal(err)
	}
	if before.FirstFailure != injected.Error() || before.Phase != "INTERRUPTED" {
		t.Fatal("lost first cause")
	}
	if err := applyPlan(p, state, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	var after journal
	if err := loadJSON(filepath.Join(state, "journal.json"), &after); err != nil {
		t.Fatal(err)
	}
	if after.FirstFailure != before.FirstFailure || after.Phase != "APPLIED" {
		t.Fatal("recovery lost failure history")
	}
}
func TestExternalCandidateChangePreserved(t *testing.T) {
	root, state := fixture(t)
	p := futurePlan(t, root, state, "node")
	if err := applyPlan(p, state, io.Discard, func(int) error { return errors.New("interrupted") }); err == nil {
		t.Fatal("fault not injected")
	}
	path := filepath.Join(state, "candidate", p.Changes[0].Path)
	external := []byte("external edit retained\n")
	writeTest(t, path, external)
	if err := applyPlan(p, state, io.Discard, nil); err == nil {
		t.Fatal("external edit overwritten")
	}
	body, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(body, external) {
		t.Fatal("external edit lost", err)
	}
	if err := compareSnapshot(p); err != nil {
		t.Fatal(err)
	}
}
func TestStalePlanAndUnknownInputs(t *testing.T) {
	for _, scenario := range []string{"dirty", "head", "mode", "untracked", "unknown-workflow", "unknown-component", "tampered-patch"} {
		t.Run(scenario, func(t *testing.T) {
			root, state := fixture(t)
			p := noopPlan(t, root, state, "go")
			switch scenario {
			case "dirty":
				writeTest(t, filepath.Join(root, "go.mod"), []byte("changed"))
			case "head":
				commitTest(t, root)
			case "mode":
				if err := os.Chmod(filepath.Join(root, "go.mod"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "untracked":
				writeTest(t, filepath.Join(root, "scripts/new-consumer.go"), []byte("package main\n"))
			case "unknown-workflow":
				writeTest(t, filepath.Join(root, ".github/workflows/unknown.yml"), []byte("name: unknown\n"))
				gitTest(t, root, "add", "--all")
				commitTest(t, root)
			case "unknown-component":
				if _, err := selectedVersion(root, "unknown"); err == nil {
					t.Fatal("unknown accepted")
				}
				return
			case "tampered-patch":
				p.Changes = []change{{"go.mod", []byte("x"), []byte("y")}}
				p.ID = planID(p)
			}
			if err := applyPlan(p, state, io.Discard, nil); err == nil {
				t.Fatal("invalid baseline/patch accepted")
			}
			if _, err := os.Lstat(filepath.Join(state, "candidate")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid input created candidate")
			}
		})
	}
}
func TestInspectionSubjectsDoNotRewriteArtifacts(t *testing.T) {
	for _, component := range []string{"terraform", "github"} {
		t.Run(component, func(t *testing.T) {
			root, state := fixture(t)
			p := noopPlan(t, root, state, component)
			if len(p.Changes) != 0 || len(p.RequiredQualification) != 1 {
				t.Fatal("subject version rewritten")
			}
			if err := applyPlan(p, state, io.Discard, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMetadataNegatives(t *testing.T) {
	for _, scenario := range []string{"digest", "architecture", "ambiguous", "url", "bundled", "sdk-hash", "unknown"} {
		t.Run(scenario, func(t *testing.T) {
			component := "node"
			if scenario == "sdk-hash" {
				component = "go"
			}
			m := fixtureMetadata(t, component, "1.99.1")
			switch scenario {
			case "digest":
				m.Documents[1].Body = append(m.Documents[1].Body, ' ')
			case "architecture":
				m.Documents[0].Body = bytes.ReplaceAll(m.Documents[0].Body, []byte("amd64"), []byte("arm64"))
			case "ambiguous":
				m.Documents[0].Body = bytes.Replace(m.Documents[0].Body, []byte(`}]}`), []byte(`},{"digest":"sha256:`+strings.Repeat("a", 64)+`","platform":{"os":"linux","architecture":"amd64"}}]}`), 1)
			case "url":
				m.Documents[2].URL = "https://example.invalid/unreviewed"
			case "bundled":
				m.Bundled = "1.0.0"
			case "sdk-hash":
				m.Archives[0].SHA256 = strings.Repeat("c", 64)
			case "unknown":
				component = "unknown"
			}
			if err := validateMetadata(component, "1.99.1", m); err == nil {
				t.Fatal("invalid official metadata accepted")
			}
		})
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("injected acquisition failure")
}
func TestAcquisitionFailureAndStateLock(t *testing.T) {
	root, state := fixture(t)
	head, inputs, err := snapshot(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := getOfficial(&http.Client{Transport: failingTransport{}}, "https://nodejs.org/dist/index.json", ""); err == nil {
		t.Fatal("acquisition failure lost")
	}
	_, failure := makePlanResolving(root, state, "go", "1.99.1", "", func(string, string) (metadata, error) {
		return metadata{}, errors.New("injected acquisition failure")
	})
	if failure == nil || !strings.Contains(failure.Error(), "injected acquisition failure") {
		t.Fatal("plan lost acquisition cause", failure)
	}
	if _, err := os.Lstat(filepath.Join(state, "generator")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed acquisition advanced to executable preparation")
	}
	actualHead, actual, err := snapshot(root, true)
	if err != nil || head != actualHead || !reflect.DeepEqual(inputs, actual) {
		t.Fatal("acquisition mutated source", err)
	}
	unlock, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := lockState(state); err == nil {
		second()
		t.Fatal("concurrent state operation accepted")
	}
	unlock()
	third, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	third()
}
func TestStructuredEditorsRejectUnknownConsumers(t *testing.T) {
	if _, err := editJSON([]byte(`{"x":1,"x":2}`), map[string]any{"/x": 3}); err == nil {
		t.Fatal("duplicate JSON accepted")
	}
	if _, err := editJSON([]byte(`{"x":1}`), map[string]any{"/y": 3}); err == nil {
		t.Fatal("missing JSON field accepted")
	}
	root, _ := fixture(t)
	body, err := os.ReadFile(filepath.Join(root, ".github/workflows/heliopause-security.yml"))
	if err != nil {
		t.Fatal(err)
	}
	selected, err := selectedVersion(root, "go")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{bytes.ReplaceAll(body, []byte("check-latest: false"), []byte("check-latest: true")), bytes.ReplaceAll(body, []byte("'"+selected+"'"), []byte("'1.0.0'"))} {
		if _, err := editWorkflow(bad, selected, "1.99.1"); err == nil {
			t.Fatal("unknown workflow selector accepted")
		}
	}
}
func TestSDKExtractionAndIntegrity(t *testing.T) {
	for _, scenario := range []string{"valid", "escape", "symlink", "duplicate"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			var buffer bytes.Buffer
			gz := gzip.NewWriter(&buffer)
			tw := tar.NewWriter(gz)
			header := &tar.Header{Name: "go/bin/go", Mode: 0o755, Typeflag: tar.TypeReg, Size: 4}
			switch scenario {
			case "escape":
				header.Name = "go/../../escape"
			case "symlink":
				header.Typeflag = tar.TypeSymlink
				header.Linkname = "/tmp/escape"
				header.Size = 0
			}
			if err := tw.WriteHeader(header); err != nil {
				t.Fatal(err)
			}
			if header.Size != 0 {
				if _, err := tw.Write([]byte("test")); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "duplicate" {
				if err := tw.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
				if _, err := tw.Write([]byte("test")); err != nil {
					t.Fatal(err)
				}
			}
			if err := tw.Close(); err != nil {
				t.Fatal(err)
			}
			if err := gz.Close(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "archive.tar.gz")
			writeTest(t, path, buffer.Bytes())
			a := archive{Size: int64(buffer.Len()), SHA256: hash(buffer.Bytes())}
			if err := verifyArchive(path, a); err != nil {
				t.Fatal(err)
			}
			a.SHA256 = strings.Repeat("f", 64)
			if err := verifyArchive(path, a); err == nil {
				t.Fatal("wrong archive hash accepted")
			}
			err := extractSDK(path, filepath.Join(root, "sdk"))
			if (err == nil) != (scenario == "valid") {
				t.Fatal("extraction outcome", err)
			}
		})
	}
}

func TestVerificationKeepsQualityExecutablesInState(t *testing.T) {
	state, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	active := t.TempDir()
	t.Setenv("HELOX_TOOL_CACHE", active)
	t.Setenv("GOENV", "/ignored/caller/go-env")
	t.Setenv("GOWORK", "/ignored/caller/go.work")
	t.Setenv("GOFLAGS", "-modfile=/ignored/caller/go.mod")
	env, err := verificationEnvironment("/selected-sdk/bin/go", state)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, entry := range env {
		k, v, ok := strings.Cut(entry, "=")
		if ok {
			if _, duplicate := values[k]; duplicate {
				t.Fatal("duplicate environment key:", k)
			}
			values[k] = v
		}
	}
	for key, want := range map[string]string{"HELOX_TOOL_CACHE": filepath.Join(state, "quality-tools"), "GOENV": "off", "GOWORK": "off", "GOFLAGS": "", "GOTOOLCHAIN": "local", "GOROOT": "/selected-sdk"} {
		if values[key] != want {
			t.Fatal("candidate verification retained caller configuration:", key)
		}
	}
	entries, err := os.ReadDir(active)
	if err != nil || len(entries) != 0 {
		t.Fatal("active tool cache was changed", err)
	}
}

func TestVerificationRejectsQualityBinAlias(t *testing.T) {
	for _, component := range []string{"quality-tools", "quality-tools/bin"} {
		t.Run(component, func(t *testing.T) {
			state, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(state, component)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), path); err != nil {
				t.Fatal(err)
			}
			if _, err := verificationEnvironment("/selected-sdk/bin/go", state); err == nil {
				t.Fatal("candidate executable cache escaped state")
			}
		})
	}
}
