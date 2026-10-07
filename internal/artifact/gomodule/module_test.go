package gomodule

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func testH1(byteValue byte) string {
	return "h1:" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string([]byte{byteValue}), 32)))
}

func TestReferenceAndProxyURLAreCanonical(t *testing.T) {
	reference, err := ParseReference("example.com/mod@v1.2.3")
	if err != nil || reference.Source() != Source() {
		t.Fatalf("reference = %#v, error = %v", reference, err)
	}
	urlValue, err := ProxyURL("example.com/mod", "v1.2.3", ".zip")
	if err != nil || urlValue != "https://proxy.golang.org/example.com/mod/@v/v1.2.3.zip" {
		t.Fatalf("proxy URL = %q, error = %v", urlValue, err)
	}
	for _, value := range []string{"example.com/mod", "example.com/mod@latest", "https://evil.example/mod@v1.2.3", "example.com/../mod@v1.2.3"} {
		if _, err := ParseReference(value); err == nil {
			t.Fatalf("accepted invalid Go module reference %q", value)
		}
	}
}

func TestResolverEnvironmentRejectsAmbientOverrides(t *testing.T) {
	if err := ValidateResolverEnvironment(ResolverEnvironment()); err != nil {
		t.Fatalf("canonical environment rejected: %v", err)
	}
	unsafe := append([]string(nil), ResolverEnvironment()...)
	unsafe[0] = "GOPROXY=https://evil.example"
	if err := ValidateResolverEnvironment(unsafe); err == nil {
		t.Fatal("accepted an overridden GOPROXY")
	}
	missing := ResolverEnvironment()[:len(ResolverEnvironment())-1]
	if err := ValidateResolverEnvironment(missing); err == nil {
		t.Fatal("accepted an incomplete resolver environment")
	}
	for _, extra := range []string{"UNKNOWN=", "GOFLAGS=-toolexec=/host/tool", "GOWORK=/host/go.work", "GOENV=/host/go.env"} {
		if err := ValidateResolverEnvironment(append(ResolverEnvironment(), extra)); err == nil {
			t.Fatalf("accepted unexpected environment entry %q", extra)
		}
	}
}

func TestResolverEnvironmentRequiresOperationPrivateCache(t *testing.T) {
	environment, err := ResolverEnvironmentForCache("/private/tmp/haa-go-cache")
	if err != nil || ValidateResolverEnvironmentForCache(environment, "/private/tmp/haa-go-cache") != nil {
		t.Fatalf("private cache environment = %#v, error = %v", environment, err)
	}
	if err := ValidateResolverEnvironmentForCache(environment, "/private/tmp/other-cache"); err == nil {
		t.Fatal("accepted substituted Go module cache")
	}
}

func TestBuildEnvironmentDisablesNetworkAndModuleMutation(t *testing.T) {
	environment, err := BuildEnvironmentForCache("/private/tmp/haa-go-cache")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(environment, "\n")
	for _, required := range []string{"GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=readonly", "GOMODCACHE=/private/tmp/haa-go-cache"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("build environment is missing %q: %#v", required, environment)
		}
	}
	if err := ValidateBuildEnvironmentForCache(environment, "/private/tmp/haa-go-cache"); err != nil {
		t.Fatal(err)
	}
	unsafe := append([]string(nil), environment...)
	unsafe[0] = "GOPROXY=https://proxy.golang.org"
	if err := ValidateBuildEnvironmentForCache(unsafe, "/private/tmp/haa-go-cache"); err == nil {
		t.Fatal("accepted a network-enabled Go build environment")
	}
}

func TestDownloadRecordsRejectDirectVCSAndInvalidChecksum(t *testing.T) {
	valid := `{"Path":"example.com/mod","Version":"v1.2.3","Info":"/tmp/mod.info","GoMod":"/tmp/mod.mod","Zip":"/tmp/mod.zip","Sum":"` + testH1('a') + `","GoModSum":"` + testH1('b') + `","Origin":null}`
	records, err := ParseDownloadJSON([]byte(valid + "\n"))
	if err != nil || len(records) != 1 {
		t.Fatalf("valid download records = %#v, error = %v", records, err)
	}
	for _, body := range []string{
		strings.Replace(valid, `"Origin":null`, `"Origin":{"VCS":"git","URL":"https://evil.example/repo"}`, 1),
		strings.Replace(valid, testH1('a'), "h1:bad", 1),
	} {
		if _, err := ParseDownloadJSON([]byte(body)); err == nil {
			t.Fatal("accepted unsafe Go module download record")
		}
	}
}

func TestDownloadRecordsAcceptGoCommandJSONStream(t *testing.T) {
	// cmd/go emits consecutive MarshalIndent objects, not one JSON object per
	// line. Checksums here are parser fixtures, not authentication evidence.
	first := DownloadRecord{Path: "example.com/first", Version: "v1.0.0", GoMod: "/cache/first.mod", Zip: "/cache/first.zip", Sum: testH1('a'), GoModSum: testH1('b')}
	second := first
	second.Path = "example.com/second"
	one, err := json.MarshalIndent(first, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	two, err := json.MarshalIndent(second, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	body := append(append(one, '\n'), two...)
	records, err := ParseDownloadJSON(body)
	if err != nil || len(records) != 2 {
		t.Fatalf("indented Go command stream = %#v, error = %v", records, err)
	}
	for name, invalid := range map[string][]byte{
		"duplicate":     append(append([]byte{}, one...), one...),
		"truncated":     append(append([]byte{}, one...), two[:len(two)-1]...),
		"trailing text": append(append([]byte{}, body...), []byte("invalid")...),
		"null record":   append(append([]byte{}, body...), []byte("null")...),
		"empty":         []byte(" \n\t"),
		"oversize":      []byte(strings.Repeat(" ", maxDownloadOutput+1)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseDownloadJSON(invalid); err == nil {
				t.Fatal("accepted incomplete or ambiguous Go command stream")
			}
		})
	}
}

func TestProxyURLPreservesGoModuleCaseIdentity(t *testing.T) {
	upper, err := ProxyURL("example.com/Azure/SDK", "v1.2.3-RC.1", ".zip")
	if err != nil || upper != "https://proxy.golang.org/example.com/!azure/!s!d!k/@v/v1.2.3-!r!c.1.zip" {
		t.Fatalf("case-preserving Go proxy URL = %q, error = %v", upper, err)
	}
	lower, err := ProxyURL("example.com/azure/sdk", "v1.2.3-RC.1", ".zip")
	if err != nil || lower == upper {
		t.Fatalf("distinct module paths collapsed: %q, error = %v", lower, err)
	}
}

func TestBuildLockedGraphPreservesSourceAndEdges(t *testing.T) {
	recordsBody := strings.Join([]string{
		`{"Path":"example.com/root","Version":"v1.0.0","GoMod":"/root.mod","Zip":"/root.zip","Sum":"` + testH1('a') + `","GoModSum":"` + testH1('b') + `","Origin":null}`,
		`{"Path":"example.com/dep","Version":"v1.1.0","GoMod":"/dep.mod","Zip":"/dep.zip","Sum":"` + testH1('c') + `","GoModSum":"` + testH1('d') + `","Origin":null}`,
	}, "\n")
	records, err := ParseDownloadJSON([]byte(recordsBody))
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := ParseReference("example.com/root@v1.0.0")
	graph, err := BuildLockedGraph(reference, records, []byte("example.com/app example.com/root@v1.0.0\nexample.com/root@v1.0.0 example.com/dep@v1.1.0\n"), []byte("module example.com/app\nrequire example.com/root v1.0.0\n"))
	if err != nil || len(graph.Nodes()) != 2 || len(graph.Edges()) != 1 {
		t.Fatalf("graph = %#v, error = %v", graph, err)
	}
	for _, node := range graph.Nodes() {
		if node.Artifact().Identity().Source() != Source() || node.Artifact().AcquisitionLocator() == "" {
			t.Fatalf("node lost Go source identity: %#v", node)
		}
	}
}

func TestBuildProjectSnapshotBindsAllRecordsAndControlFiles(t *testing.T) {
	records, err := ParseDownloadJSON([]byte(`{"Path":"example.com/mod","Version":"v1.2.3","GoMod":"/mod.mod","Zip":"/mod.zip","Sum":"` + testH1('a') + `","GoModSum":"` + testH1('b') + `","Origin":null}` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/workspace/project")
	installContext, _ := domain.NewInstallContext(target)
	snapshot, err := BuildProjectSnapshot(installContext, records, []byte("example.com/app example.com/mod@v1.2.3\n"), []byte("module example.com/app\nrequire example.com/mod v1.2.3\n"), []byte("example.com/mod v1.2.3 "+testH1('a')+"\nexample.com/mod v1.2.3/go.mod "+testH1('b')+"\n"))
	if err != nil || !snapshot.Valid() || len(snapshot.Dependencies()) != 1 || len(snapshot.ControlDigests()) != 2 {
		t.Fatalf("snapshot = %#v, error = %v", snapshot, err)
	}
	if _, err := BuildProjectSnapshot(installContext, records, []byte("example.com/app example.com/other@v1.2.3\n"), []byte("module example.com/app\nrequire example.com/mod v1.2.3\n"), []byte("sum\n")); err == nil {
		t.Fatal("accepted project graph that omits downloaded module")
	}
}

func TestProjectSnapshotAcceptsActualGoGraphShape(t *testing.T) {
	records := []DownloadRecord{{Path: "github.com/spf13/pflag", Version: "v1.0.9", GoMod: "/cache/pflag.mod", Zip: "/cache/pflag.zip", Sum: testH1('a'), GoModSum: testH1('b')}}
	mod := []byte("module example.com/haa-m12-fixture\n\ngo 1.26.8\n\nrequire github.com/spf13/pflag v1.0.9\n")
	graph := []byte("example.com/haa-m12-fixture github.com/spf13/pflag@v1.0.9\nexample.com/haa-m12-fixture go@1.26.8\ngo@1.26.8 toolchain@go1.26.8\n")
	target, _ := domain.NewInstallTarget("/workspace/project")
	install, _ := domain.NewInstallContext(target)
	if _, err := BuildProjectSnapshot(install, records, graph, mod, []byte("github.com/spf13/pflag v1.0.9 "+testH1('a')+"\ngithub.com/spf13/pflag v1.0.9/go.mod "+testH1('b')+"\n")); err != nil {
		t.Fatalf("actual Go 1.26.8 graph shape rejected: %v", err)
	}
}

func TestProjectGraphBindsMainAndSelectedMinimumVersions(t *testing.T) {
	records := []DownloadRecord{
		{Path: "example.com/root", Version: "v1.0.0", Sum: testH1('a'), GoModSum: testH1('b')},
		{Path: "example.com/dep", Version: "v1.2.0", Sum: testH1('c'), GoModSum: testH1('d')},
	}
	mod := []byte("module local-app\ngo 1.26.8\nrequire (\nexample.com/root v1.0.0\nexample.com/dep v1.2.0\n)\n")
	body := "local-app example.com/root@v1.0.0\nlocal-app example.com/dep@v1.2.0\nlocal-app go@1.26.8\ngo@1.26.8 toolchain@go1.26.8\nexample.com/root@v1.0.0 example.com/dep@v1.1.0\nexample.com/dep@v1.1.0 go@1.25.0\n"
	edges, err := normalizeProjectGraph([]byte(body), records, mod)
	if err != nil || len(edges) != 1 || edges[0].to != "example.com/dep@v1.2.0" {
		t.Fatalf("selected requirement edges = %#v, error = %v", edges, err)
	}
	for name, invalid := range map[string]string{
		"unrecorded main":           strings.ReplaceAll(body, "local-app", "other-app"),
		"invented main version":     strings.ReplaceAll(body, "local-app ", "local-app@v1.0.0 "),
		"unknown source":            body + "evil.example/mod@v1.0.0 example.com/dep@v1.0.0\n",
		"unknown target":            body + "example.com/root@v1.0.0 evil.example/mod@v1.0.0\n",
		"higher required version":   strings.Replace(body, "example.com/dep@v1.1.0", "example.com/dep@v1.3.0", 1),
		"root differs from control": strings.Replace(body, "local-app example.com/dep@v1.2.0", "local-app example.com/dep@v1.1.0", 1),
		"missing root edge":         strings.Replace(body, "local-app example.com/dep@v1.2.0\n", "", 1),
		"missing Go version":        strings.Replace(body, "local-app go@1.26.8\n", "", 1),
		"unbound Go version":        strings.Replace(body, "local-app go@1.26.8", "local-app go@1.27.0", 1),
		"toolchain mismatch":        strings.Replace(body, "toolchain@go1.26.8", "toolchain@go1.25.0", 1),
		"duplicate edge":            body + "local-app example.com/root@v1.0.0\n",
		"invalid line":              body + "invalid\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeProjectGraph([]byte(invalid), records, mod); err == nil {
				t.Fatal("accepted substituted or incomplete requirement graph")
			}
		})
	}
	ambiguous := append(append([]DownloadRecord{}, records...), DownloadRecord{Path: "example.com/dep", Version: "v1.1.0", Sum: testH1('e'), GoModSum: testH1('f')})
	if _, err := normalizeProjectGraph([]byte(body), ambiguous, mod); err == nil {
		t.Fatal("accepted multiple selected versions of the same module")
	}
	if _, err := normalizeProjectGraph([]byte(strings.Repeat(" ", maxDownloadOutput+1)), records, mod); err == nil {
		t.Fatal("accepted overbound graph")
	}
}

func TestBuildLockedGraphRejectsMalformedReferenceWithoutPanic(t *testing.T) {
	reference, _ := domain.NewArtifactReference(Source(), "missing-version")
	if _, err := BuildLockedGraph(reference, []DownloadRecord{{Path: "example.com/root"}}, []byte("graph"), []byte("mod")); err == nil {
		t.Fatal("accepted malformed reference")
	}
}
