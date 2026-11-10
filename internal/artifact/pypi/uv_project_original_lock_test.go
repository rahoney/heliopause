package pypi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestUVOriginalLockRetainsPublicOldGraphButCannotSelectEditedProject(t *testing.T) {
	old, body, _ := uvNativeProjectFixture(t)
	manifest := []byte(fmt.Sprintf("[project]\nname=%q\nversion=%q\nrequires-python='=="+runtimeidentity.PythonVersion+"'\ndependencies=[]\n", old.project, old.version))
	current, err := ParseUVProjectInput(manifest, body, true, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateUVProjectOriginalLock(current, body); err != nil {
		t.Fatal("safe historical public graph rejected", err)
	}
	if _, err := ParseUVNativeProjectLock(current, body); err == nil {
		t.Fatal("stale original became current selected authority")
	}
	if err := ValidateUVProjectOriginalLock(old, body); err != nil {
		t.Fatal("unchanged original rejected", err)
	}
	for _, change := range []struct{ name, from, to string }{
		{"schema", "version=1", "version=99"},
		{"source", "https://pypi.org/simple", "https://example.org/simple"},
		{"virtual escape", "source={virtual='.'}", "source={virtual='../escape'}"},
		{"root identity", "name='haa-example'", "name='other'"},
		{"Python", "requires-python='==" + runtimeidentity.PythonVersion + "'", "requires-python='==3.13.0'"},
		{"old constraint", "specifier='<2,>=1.0'", "specifier='>=9'"},
		{"missing edge", "dependencies=[{name='child'}]", "dependencies=[{name='missing'}]"},
	} {
		t.Run(change.name, func(t *testing.T) {
			if !strings.Contains(string(body), change.from) {
				t.Fatal("fixture did not exercise mutation")
			}
			bad := []byte(strings.Replace(string(body), change.from, change.to, 1))
			if err := ValidateUVProjectOriginalLock(current, bad); err == nil {
				t.Fatal("unsafe historical input accepted")
			}
		})
	}
	if ValidateUVProjectOriginalLock(UVProjectInput{}, body) == nil || ValidateUVProjectOriginalLock(current, nil) == nil {
		t.Fatal("missing original input accepted")
	}
}
