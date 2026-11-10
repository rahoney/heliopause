package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func uvRunRootsFixture(t *testing.T) (*pipxNativeInput, artifactpypi.UVRunRequirements) {
	t.Helper()
	primary, _ := artifactpypi.ParseUVRunRequirement("probe==1.0")
	child, _ := artifactpypi.ParseUVRunRequirement("child==1.0")
	request, err := artifactpypi.NewUVRunRequirements(primary, []artifactpypi.UVRunRequirement{child})
	if err != nil {
		t.Fatal(err)
	}
	wire := "lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=3.14.7'\n"
	var inputs []artifactpypi.UVWheelInput
	var pages []artifactpypi.SimpleProject
	for _, name := range []string{"probe", "child"} {
		additional := map[string][]byte{}
		if name == "probe" {
			additional["probe-1.0.dist-info/entry_points.txt"] = []byte("[console_scripts]\nprobe = probe:main\n")
		}
		body := linuxDynamicProjectWheel(t, name, "def main(): return 0\n", additional)
		sum := sha256.Sum256(body)
		digest := hex.EncodeToString(sum[:])
		filename := name + "-1.0-py3-none-any.whl"
		url := "https://files.pythonhosted.org/packages/" + filename
		wire += fmt.Sprintf("[[packages]]\nname=%q\nversion='1.0'\nwheels=[{url=%q,size=%d,hashes={sha256=%q}}]\n", name, url, len(body), digest)
		metadata, err := json.Marshal(map[string]any{"info": map[string]any{"name": name, "version": "1.0"}, "urls": []any{map[string]any{"filename": filename, "url": url, "size": len(body), "digests": map[string]string{"sha256": digest}, "requires_python": "", "yanked": false, "packagetype": "bdist_wheel"}}})
		if err != nil {
			t.Fatal(err)
		}
		page, err := artifactpypi.ParseUVReleaseMetadata(name, "1.0", metadata)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
		inputs = append(inputs, artifactpypi.UVWheelInput{URL: url, Reader: bytes.NewReader(body), Size: int64(len(body))})
	}
	lock, err := artifactpypi.ParseUVLock(primary.Reference(), []byte(wire), PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := freezeUVRunSelectionInput(context.Background(), lock, inputs, pages, request)
	if err != nil {
		t.Fatal(err)
	}
	return frozen, request
}

func TestUVRunRootFreezePreservesAllOriginalsAndSeparateGraphs(t *testing.T) {
	frozen, request := uvRunRootsFixture(t)
	if !frozen.valid() || !frozen.selection.Valid() || len(frozen.files) != 2 || len(frozen.selection.Edges()) != 0 || len(frozen.graph.Nodes()) != 0 || pipxNativeRequirementMatches(request.Primary().Base(), frozen) {
		t.Fatal("joint source became an ordinary single-root or pipx input")
	}
	entry, infos, err := uvRunConsoleOriginal(frozen, "probe")
	if err != nil || !entry.Valid() || len(infos) != 2 {
		t.Fatal("whole source or primary console binding missing", err)
	}
	_, facts, _ := uvRunBindingFixture(t)
	if selected, err := uvRunRootSelectionFromCensus(context.Background(), request, frozen, "probe", facts); err == nil || selected.Resolution().Valid() {
		t.Fatal("primary-only native census became complete root selection")
	}
	other, _ := artifactpypi.ParseUVRunRequirement("child")
	frozen.requests, _ = artifactpypi.NewUVRunRequirements(request.Primary(), []artifactpypi.UVRunRequirement{other})
	transport := &pipxInputTransport{}
	if err := frozen.introduce(context.Background(), transport, strings.Repeat("a", 64)); err == nil || transport.calls != 0 {
		t.Fatal("changed request entered native transport")
	}
}

func TestUVRunRootPreparationBindsRequirementsAndFixedInertTerminal(t *testing.T) {
	_, request := uvRunRootsFixture(t)
	args, err := uvRunSelectionPreparationArguments(request, "probe")
	base, baseErr := uvRunPreparationArguments(request.Primary().Base(), "probe")
	if err != nil || baseErr != nil || len(args) != len(base)+2 || !slices.Equal(args[:len(base)-1], base[:len(base)-1]) || !slices.Equal(args[len(base)-1:], []string{"--with", "child==1.0", "/dev/null"}) {
		t.Fatal("joint native preparation differs from the fixed offline ABI", err)
	}
	for _, arg := range args {
		if arg == "probe" {
			t.Fatal("selected program entered native preparation")
		}
	}
	if args, err := uvRunSelectionPreparationArguments(artifactpypi.UVRunRequirements{}, "probe"); err == nil || len(args) != 0 {
		t.Fatal("zero joint request accepted")
	}
}
