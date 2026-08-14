package npm

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestParsePackageLockV3BuildsExactConnectedGraph(t *testing.T) {
	t.Parallel()

	graph, err := ParsePackageLockV3(lockReference(t, "primary@1.0.0"), []byte(packageLockJSON(t, false)))
	if err != nil {
		t.Fatal(err)
	}
	nodes := graph.Nodes()
	if len(nodes) != 2 || len(graph.Edges()) != 1 || graph.Primary().String() == "" {
		t.Fatalf("graph = %#v", graph)
	}
	names := map[string]domain.DependencyRole{}
	for _, node := range nodes {
		names[node.Artifact().Identity().Name()] = node.Role()
	}
	if names["primary"] != domain.DependencyPrimary || names["child"] != domain.DependencyTransitive {
		t.Fatalf("nodes = %#v", nodes)
	}
	for _, node := range nodes {
		if node.Artifact().DeclaredIntegrity() == "" || node.Artifact().AcquisitionLocator() == "" {
			t.Fatalf("node is not exact: %#v", node)
		}
	}
}

func TestParsePackageLockV3RejectsUnsupportedOrIncompleteSemantics(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		body string
	}{
		{"legacy version", strings.Replace(packageLockJSON(t, false), "\"lockfileVersion\":3", "\"lockfileVersion\":2", 1)},
		{"wrong root dependency", strings.Replace(packageLockJSON(t, false), "\"primary\":\"1.0.0\"", "\"other\":\"1.0.0\"", 1)},
		{"non registry tarball", strings.Replace(packageLockJSON(t, false), "https://registry.npmjs.org/primary/-/primary-1.0.0.tgz", "https://example.test/primary.tgz", 1)},
		{"missing resolved child", strings.Replace(packageLockJSON(t, false), "\"child\":\"1.0.0\"", "\"missing\":\"1.0.0\"", 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParsePackageLockV3(lockReference(t, "primary"), []byte(test.body)); err == nil {
				t.Fatal("ParsePackageLockV3() error = nil")
			}
		})
	}
}

func TestParsePackageLockV3PreservesHostInstallActionForPolicy(t *testing.T) {
	t.Parallel()

	graph, err := ParsePackageLockV3(lockReference(t, "primary"), []byte(packageLockJSON(t, true)))
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range graph.Nodes() {
		if node.Artifact().Identity().Name() == "primary" && node.HostInstallAction() {
			return
		}
	}
	t.Fatal("primary lifecycle install action was not preserved")
}

func lockReference(t *testing.T, locator string) domain.ArtifactReference {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	reference, err := domain.NewArtifactReference(source, locator)
	if err != nil {
		t.Fatal(err)
	}
	return reference
}

func packageLockJSON(t *testing.T, lifecycle bool) string {
	t.Helper()
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(make([]byte, 64))
	body := "{\"lockfileVersion\":3,\"packages\":{\"\":{\"name\":\"resolver\",\"version\":\"1.0.0\",\"dependencies\":{\"primary\":\"1.0.0\"}},\"node_modules/primary\":{\"version\":\"1.0.0\",\"resolved\":\"https://registry.npmjs.org/primary/-/primary-1.0.0.tgz\",\"integrity\":\"%s\",\"dependencies\":{\"child\":\"1.0.0\"},\"hasInstallScript\":%t},\"node_modules/child\":{\"version\":\"1.0.0\",\"resolved\":\"https://registry.npmjs.org/child/-/child-1.0.0.tgz\",\"integrity\":\"%s\"}}}"
	return fmt.Sprintf(body, integrity, lifecycle, integrity)
}
