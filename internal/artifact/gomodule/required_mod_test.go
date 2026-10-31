package gomodule

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/sumdb/dirhash"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestProjectSnapshotIgnoresUnloadedPrunedLowerReferences(t *testing.T) {
	for _, count := range []int{1, MaxRequiredModControls + 4} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			target, _ := domain.NewInstallTarget("/fixture/pruned")
			install, _ := domain.NewInstallContext(target)
			records := []DownloadRecord{
				{Path: "example.com/a", Version: "v1.0.0", GoMod: "/a.mod", Zip: "/a.zip", Sum: testH1('a'), GoModSum: testH1('b')},
				{Path: "example.com/b", Version: "v1.1.0", GoMod: "/b.mod", Zip: "/b.zip", Sum: testH1('c'), GoModSum: testH1('d')},
			}
			mod := []byte("module example.com/root\ngo 1.26.0\nrequire (\nexample.com/a v1.0.0\nexample.com/b v1.1.0\n)\n")
			graph := "example.com/root example.com/a@v1.0.0\nexample.com/root example.com/b@v1.1.0\nexample.com/root go@1.26.0\ngo@1.26.0 toolchain@go1.26.0\n"
			for i := range count {
				graph += fmt.Sprintf("example.com/a@v1.0.0 example.com/b@v1.0.%d\n", i)
			}
			sums := ""
			for _, record := range records {
				sums += record.Path + " " + record.Version + " " + record.Sum + "\n" + record.Path + " " + record.Version + "/go.mod " + record.GoModSum + "\n"
			}
			snapshot, err := BuildProjectSnapshot(install, records, []byte(graph), mod, []byte(sums))
			if err != nil {
				t.Fatalf("unexpanded lower requirements must not require nonexistent control sums: %v", err)
			}
			for _, artifact := range snapshot.Dependencies() {
				if strings.Contains(artifact.DeclaredIntegrity(), ";required-mods=") {
					t.Fatal("pruned requirements became acquired control subjects")
				}
			}
			ref, _ := ParseReference("example.com/a@v1.0.0")
			requested, err := BuildLockedGraph(ref, records, []byte(graph), mod, []byte(sums))
			if err != nil {
				t.Fatal(err)
			}
			for _, node := range requested.Nodes() {
				matched := false
				for _, artifact := range snapshot.Dependencies() {
					if node.Artifact() == artifact {
						matched = true
					}
				}
				if !matched {
					t.Fatal("requested and complete pruning context differ")
				}
			}
			if count > MaxRequiredModControls {
				for i := range count {
					sums += fmt.Sprintf("example.com/b v1.0.%d/go.mod %s\n", i, testH1('e'))
				}
				if _, err := BuildProjectSnapshot(install, records, []byte(graph), mod, []byte(sums)); err == nil {
					t.Fatal("included controls exceeded the unchanged context bound")
				}
			}
		})
	}
}

func TestProjectSnapshotRetainsRequiredLowerModuleControls(t *testing.T) {
	target, _ := domain.NewInstallTarget("/fixture/project")
	install, _ := domain.NewInstallContext(target)
	records := []DownloadRecord{
		{Path: "example.com/a", Version: "v1.0.0", GoMod: "/a.mod", Zip: "/a.zip", Sum: testH1('a'), GoModSum: testH1('b')},
		{Path: "example.com/b", Version: "v1.1.0", GoMod: "/b.mod", Zip: "/b.zip", Sum: testH1('c'), GoModSum: testH1('d')},
	}
	mod := []byte("module example.com/root\ngo 1.26.0\nrequire (\nexample.com/a v1.0.0\nexample.com/b v1.1.0\n)\n")
	graph := []byte("example.com/root example.com/a@v1.0.0\nexample.com/root example.com/b@v1.1.0\nexample.com/root go@1.26.0\nexample.com/a@v1.0.0 example.com/b@v1.0.0\ngo@1.26.0 toolchain@go1.26.0\n")
	sum := ""
	for _, r := range records {
		sum += r.Path + " " + r.Version + " " + r.Sum + "\n" + r.Path + " " + r.Version + "/go.mod " + r.GoModSum + "\n"
	}
	sum += "example.com/b v1.0.0/go.mod " + testH1('e') + "\n"
	snapshot, err := BuildProjectSnapshot(install, records, graph, mod, []byte(sum))
	if err != nil {
		t.Fatal(err)
	}
	for _, dependency := range snapshot.Dependencies() {
		if dependency.Identity().Name() == "example.com/b" {
			want := "h1=" + testH1('c') + ";go.mod=" + testH1('d') + ";required-mods=v1.0.0:" + testH1('e')
			if dependency.DeclaredIntegrity() != want {
				t.Fatalf("required lower-version go.mod identity was lost: %q", dependency.DeclaredIntegrity())
			}
		}
	}
	ref, _ := ParseReference("example.com/a@v1.0.0")
	requested, err := BuildLockedGraph(ref, records, graph, mod, []byte(sum))
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range requested.Nodes() {
		matched := false
		for _, complete := range snapshot.Dependencies() {
			if node.Artifact() == complete {
				matched = true
			}
		}
		if !matched {
			t.Fatal("requested graph lost complete-project metadata binding")
		}
	}
	oldLine := "example.com/b v1.0.0/go.mod " + testH1('e') + "\n"
	selectedPair := records[0].Path + " " + records[0].Version + "/go.mod " + records[0].GoModSum + "\n"
	for _, bad := range []string{strings.ReplaceAll(sum, selectedPair, ""), sum + oldLine, strings.ReplaceAll(sum, oldLine, "example.com/b v1.0.0/go.mod invalid\n")} {
		if _, err := BuildProjectSnapshot(install, records, graph, mod, []byte(bad)); err == nil {
			t.Fatal("missing selected checksum or ambiguous/malformed lower checksum accepted")
		}
	}
	withUnused := sum + "example.com/a v0.9.0/go.mod " + testH1('e') + "\n"
	complete, err := BuildProjectSnapshot(install, records, graph, mod, []byte(withUnused))
	if err != nil {
		t.Fatal(err)
	}
	for _, dependency := range complete.Dependencies() {
		if dependency.Identity().Name() == "example.com/a" && strings.Contains(dependency.DeclaredIntegrity(), ";required-mods=") {
			t.Fatal("unused old go.sum line became acquisition context")
		}
	}
}

func requiredModFixture(t *testing.T) (Bundle, domain.ResolvedArtifactIdentity, string) {
	t.Helper()
	current := []byte("module example.com/module\ngo 1.26.0\n")
	old := []byte("module example.com/module\ngo 1.25\n")
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	for name, body := range map[string][]byte{"go.mod": current, "module.go": []byte("package module\n")} {
		f, err := w.Create("example.com/module@v1.0.0/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	id, _ := domain.NewResolvedArtifactIdentity(Source(), "example.com/module", "v1.0.0", "module")
	b := Bundle{mod: current, zip: archive.Bytes(), required: []RequiredModControl{{"v0.9.0", old}}}
	z, m, err := HashBundle(b, id)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(old)), nil })
	if err != nil {
		t.Fatal(err)
	}
	return b, id, "h1=" + z + ";go.mod=" + m + ";required-mods=v0.9.0:" + sum
}

func TestRequiredModEnvelopeAcquisitionAndCacheBinding(t *testing.T) {
	bundle, id, integrity := requiredModFixture(t)
	root := t.TempDir()
	client, _ := NewPublicClient(filepath.Join(root, "intake"))
	var requests []string
	client.http.Transport = proxyTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "proxy.golang.org" || r.URL.Scheme != "https" || r.Header.Get("Authorization") != "" {
			t.Fatal("noncanonical metadata source")
		}
		requests = append(requests, r.URL.Path)
		var body []byte
		switch r.URL.Path {
		case "/example.com/module/@v/v1.0.0.mod":
			body = bundle.Mod()
		case "/example.com/module/@v/v0.9.0.mod":
			body = bundle.RequiredMods()[0].Body
		case "/example.com/module/@v/v1.0.0.zip":
			body = bundle.Zip()
		default:
			t.Fatalf("unfrozen metadata request: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, ContentLength: int64(len(body)), Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	})
	locator, _ := ProxyURL(id.Name(), id.Version(), ".zip")
	resolved, _ := domain.NewResolvedArtifact(id, locator, integrity)
	run, _ := domain.NewRunID()
	acquired, err := client.Acquire(context.Background(), run, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 3 {
		t.Fatal(requests)
	}
	read, err := ReadIntake(client.intakeRoot, acquired)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.RequiredMods()) != 1 || !bytes.Equal(read.Mod(), bundle.Mod()) {
		t.Fatal("required envelope lost context")
	}
	copy := read.RequiredMods()
	copy[0].Body[0] = '!'
	copy[0].Version = "v0.8.0"
	if read.RequiredMods()[0].Body[0] == '!' || read.RequiredMods()[0].Version != "v0.9.0" {
		t.Fatal("mutable metadata exposed")
	}
	if _, err := HashRequiredMods(read, id, integrity); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "cache")
	if err := os.Mkdir(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeModule(cache, read, id, integrity); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(cache, "cache/download/example.com/module/@v/v0.9.0.mod")
	if body, err := os.ReadFile(old); err != nil || !bytes.Equal(body, bundle.RequiredMods()[0].Body) {
		t.Fatal("required metadata not materialized", err)
	}
	for _, name := range []string{"cache/download/example.com/module/@v/v0.9.0.zip", "example.com/module@v0.9.0"} {
		if _, err := os.Stat(filepath.Join(cache, name)); !os.IsNotExist(err) {
			t.Fatal("unselected old payload installed")
		}
	}
	if err := os.WriteFile(old, []byte("poison"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckMaterializedModule(cache, read, id, integrity); err == nil {
		t.Fatal("required metadata corruption accepted")
	}
}

func TestRequiredModContextRejectsWrongIdentityCoverageAndBounds(t *testing.T) {
	bundle, id, integrity := requiredModFixture(t)
	for _, kind := range []string{"missing", "extra", "body", "same-version", "higher-version", "duplicate", "order", "current-proof-only"} {
		t.Run(kind, func(t *testing.T) {
			b := Bundle{mod: bundle.Mod(), zip: bundle.Zip(), required: bundle.RequiredMods()}
			decl := integrity
			switch kind {
			case "missing":
				b.required = nil
			case "extra":
				b.required = append(b.required, RequiredModControl{"v0.9.1", b.Mod()})
			case "body":
				b.required[0].Body[0] = '!'
			case "same-version":
				decl = strings.ReplaceAll(decl, "v0.9.0:", "v1.0.0:")
				b.required[0].Version = "v1.0.0"
			case "higher-version":
				decl = strings.ReplaceAll(decl, "v0.9.0:", "v1.1.0:")
				b.required[0].Version = "v1.1.0"
			case "duplicate":
				decl += ",v0.9.0:" + testH1('a')
			case "order":
				decl += ",v0.8.0:" + testH1('a')
			case "current-proof-only":
				decl = strings.Split(decl, ";required-mods=")[0]
			}
			if _, err := HashRequiredMods(b, id, decl); err == nil {
				t.Fatal("invalid required context accepted")
			}
		})
	}
	for _, value := range []string{integrity + ",", integrity + ";unknown=value", strings.Split(integrity, ";required-mods=")[0] + ";required-mods=", integrity + strings.Repeat("x", 1024)} {
		if _, _, err := ParseIntegrity(value); err == nil {
			t.Fatal("noncanonical declaration accepted")
		}
	}
	if _, err := encodeBundleWithRequiredMods(bundle.Mod(), bundle.Zip(), []RequiredModControl{{"v0.9.0", make([]byte, MaxModBytes)}}); err == nil {
		t.Fatal("combined control bound widened")
	}
	refs := make([]RequiredModControl, MaxRequiredModControls+1)
	if _, err := encodeBundleWithRequiredMods(bundle.Mod(), bundle.Zip(), refs); err == nil {
		t.Fatal("control count widened")
	}
}

func TestRequiredModEnvelopeRejectsNoncanonicalFraming(t *testing.T) {
	bundle, id, integrity := requiredModFixture(t)
	valid, err := encodeBundleWithRequiredMods(bundle.Mod(), bundle.Zip(), bundle.RequiredMods())
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"valid", "unknown-header", "duplicate-version", "unknown-json", "trailing-json", "empty-controls"} {
		t.Run(kind, func(t *testing.T) {
			body := bytes.Clone(valid)
			switch kind {
			case "unknown-header":
				body[13] = '9'
			case "duplicate-version":
				body, _ = encodeBundleWithRequiredMods(bundle.Mod(), bundle.Zip(), append(bundle.RequiredMods(), bundle.RequiredMods()...))
			case "unknown-json", "trailing-json", "empty-controls":
				control := []byte(`{"mod":"bW9kdWxlIGV4YW1wbGUuY29tL21vZHVsZQo=","required_mods":[]}`)
				if kind == "unknown-json" {
					control = []byte(`{"mod":"YQ==","required_mods":[],"other":true}`)
				}
				if kind == "trailing-json" {
					control = append(control, []byte(" {}")...)
				}
				body = append([]byte(bundleRequiredHeader), make([]byte, 8)...)
				binary.BigEndian.PutUint64(body[len(bundleRequiredHeader):], uint64(len(control)))
				body = append(append(body, control...), bundle.Zip()...)
			}
			root := t.TempDir()
			run, _ := domain.NewRunID()
			if err := os.Mkdir(filepath.Join(root, run.String()), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, run.String(), "module.bundle"), body, 0o600); err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
			a, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(id, digest, "intake:"+run.String()+":module", uint64(len(body)), integrity)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ReadIntake(root, a)
			if (err == nil) != (kind == "valid") {
				t.Fatalf("frame validation=%v", err)
			}
		})
	}
}
