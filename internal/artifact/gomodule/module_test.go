package gomodule

import (
	"encoding/base64"
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
	graph, err := BuildLockedGraph(reference, records, []byte("example.com/root@v1.0.0 example.com/dep@v1.1.0\n"))
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
	snapshot, err := BuildProjectSnapshot(installContext, records, []byte("example.com/app@v1.0.0 example.com/mod@v1.2.3\n"), []byte("module example.com/app\n"), []byte("example.com/mod v1.2.3 h1:fixture\n"))
	if err != nil || !snapshot.Valid() || len(snapshot.Dependencies()) != 1 || len(snapshot.ControlDigests()) != 2 {
		t.Fatalf("snapshot = %#v, error = %v", snapshot, err)
	}
	if _, err := BuildProjectSnapshot(installContext, records, []byte("example.com/app@v1.0.0 example.com/other@v1.2.3\n"), []byte("module example.com/app\n"), []byte("sum\n")); err == nil {
		t.Fatal("accepted project graph that omits downloaded module")
	}
}
