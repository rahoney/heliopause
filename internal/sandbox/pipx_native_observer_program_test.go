package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// The Go controller remains the owner of its closed native command. The
// external verifier must agree on every code/flag byte, without a second
// hand-maintained Python version pin in the CPP program template.
func TestPipxNativeObserverControllerGrammarStaysBoundToOwner(t *testing.T) {
	source, err := os.ReadFile("../../tools/gvisor-observer/observer.cc")
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		t.Helper()
		expression := regexp.MustCompile(`(?m)^constexpr char ` + regexp.QuoteMeta(name) + `\[\] = ("(?:[^"\\]|\\.)*");$`)
		matches := expression.FindAllSubmatch(source, -1)
		if len(matches) != 1 {
			t.Fatal("CPP fixed grammar absent or duplicated", name)
		}
		var value string
		if err := json.Unmarshal(matches[0][1], &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	if read("kProfilePipxNativeResolver") != pipxNativeResolverProfile {
		t.Fatal("native profile identity drift")
	}
	if read("kPipxNativeProgramPrefix")+runtimeidentity.PythonVersion+read("kPipxNativeProgramSuffix") != pipxNativeProgram {
		t.Fatal("CPP root program differs from canonical controller")
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement("fixture==1.0")
	if err != nil {
		t.Fatal(err)
	}
	args, err := pipxNativeArguments(pipxNativeInstall, requirement)
	if err != nil {
		t.Fatal(err)
	}
	if read("kPipxNativePipArguments") != args[len(args)-1] {
		t.Fatal("CPP fixed backend flags drift")
	}
	query := read("kPipxNativeMetadataQuery")
	sum := sha256.Sum256([]byte(query))
	// Exact selected pipx1.18.4 venv_inspect.py AST/dedent input, independently
	// captured in the C3c3b3b source facts. This hash is no execution authority.
	if len(query) != 1304 || hex.EncodeToString(sum[:]) != "f306acdd8d55713a6732097b055dfb729846cf389c6e5de785a4663b50b10ee2" {
		t.Fatal("native metadata query differs from reviewed original source")
	}
}
