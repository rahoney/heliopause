package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeLockMatchesSandboxRuntimeIdentity(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile(filepath.Join("..", "..", "scripts", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		NodeImage struct {
			Reference  string `json:"reference"`
			NPMVersion string `json:"npm_version"`
		} `json:"node_image"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.NodeImage.Reference != nodeImageReference || lock.NodeImage.NPMVersion != resolverNPMVersion {
		t.Fatalf("runtime lock=%#v image=%q npm=%q", lock.NodeImage, nodeImageReference, resolverNPMVersion)
	}
}

func TestProbe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		operatingSystem string
		executor        fakeExecutor
		available       bool
		limitation      string
	}{
		{name: "non Linux", operatingSystem: "darwin", limitation: "M3_LINUX_ONLY"},
		{name: "missing runtime", operatingSystem: "linux", executor: fakeExecutor{lookupError: errors.New("missing")}, limitation: "M3_RUNTIME_UNAVAILABLE"},
		{name: "old Docker", operatingSystem: "linux", executor: fakeExecutor{outputs: map[string]string{"docker version --format {{.Server.Version}}": "29.5.3"}}, limitation: "M3_RUNTIME_VERSION_UNSUPPORTED"},
		{name: "wrong gVisor", operatingSystem: "linux", executor: fakeExecutor{outputs: map[string]string{"docker version --format {{.Server.Version}}": "29.6.0", "runsc --version": "release-20260727.0", "docker info --format {{json (index .Runtimes \"runsc-trace\")}}": "{\"path\":\"/usr/libexec/heliopause/runsc\"}"}}, limitation: "M3_RUNTIME_VERSION_UNSUPPORTED"},
		{name: "unpatched gVisor lacking observation points", operatingSystem: "linux", executor: fakeExecutor{outputs: map[string]string{"docker version --format {{.Server.Version}}": "29.6.0", "runsc --version": gVisorRelease, "runsc trace metadata": "Name: sentry/clone\nName: sentry/execve\n", "docker info --format {{json (index .Runtimes \"runsc-trace\")}}": "{\"path\":\"/usr/libexec/heliopause/runsc\"}"}}, limitation: "M3_RUNTIME_VERSION_UNSUPPORTED"},
		{name: "unregistered runtime", operatingSystem: "linux", executor: fakeExecutor{outputs: map[string]string{"docker version --format {{.Server.Version}}": "29.6.0", "runsc --version": gVisorRelease, "runsc trace metadata": "Name: syscall/open_result\nName: sentry/mount_topology_snapshot\nName: sentry/mount_topology_mutation\n"}}, limitation: "M3_RUNTIME_UNAVAILABLE"},
		{name: "missing image", operatingSystem: "linux", executor: fakeExecutor{outputs: map[string]string{"docker version --format {{.Server.Version}}": "29.6.0", "runsc --version": gVisorRelease, "runsc trace metadata": "Name: syscall/open_result\nName: sentry/mount_topology_snapshot\nName: sentry/mount_topology_mutation\n", "docker info --format {{json (index .Runtimes \"runsc-trace\")}}": "{\"path\":\"/usr/libexec/heliopause/runsc\"}"}}, limitation: "M3_IMAGE_UNAVAILABLE"},
		{name: "available", operatingSystem: "linux", executor: fakeExecutor{outputs: map[string]string{"docker version --format {{.Server.Version}}": "29.6.0", "runsc --version": gVisorRelease, "runsc trace metadata": "Name: syscall/open_result\nName: sentry/mount_topology_snapshot\nName: sentry/mount_topology_mutation\n", "docker info --format {{json (index .Runtimes \"runsc-trace\")}}": "{\"path\":\"/usr/libexec/heliopause/runsc\"}", "docker image inspect " + nodeImageReference + " --format {{.Id}}": "sha256:example"}}, available: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			capability, err := probe(context.Background(), test.operatingSystem, test.executor)
			if err != nil || capability.Available != test.available || capability.LimitationCode != test.limitation {
				t.Fatalf("probe() = %#v, %v", capability, err)
			}
		})
	}
}

func TestProbeUsesDockerRegisteredRunscRatherThanPATH(t *testing.T) {
	patched := "Name: syscall/open_result\nName: sentry/mount_topology_snapshot\nName: sentry/mount_topology_mutation\n"
	base := map[string]string{
		"docker version --format {{.Server.Version}}":                     "29.6.0",
		"docker info --format {{json (index .Runtimes \"runsc-trace\")}}": "{\"path\":\"/usr/libexec/heliopause/runsc\"}",
	}
	for _, test := range []struct {
		name    string
		outputs map[string]string
		want    string
	}{
		{"path patched registered stock", map[string]string{"runsc --version": gVisorRelease, "runsc trace metadata": patched, registeredRunscPath + " --version": gVisorRelease, registeredRunscPath + " trace metadata": "Name: sentry/clone"}, "unsupported"},
		{"path stock registered patched", map[string]string{"runsc --version": "release-old", "runsc trace metadata": "", registeredRunscPath + " --version": gVisorRelease, registeredRunscPath + " trace metadata": patched}, ""},
		{"missing registered path", map[string]string{}, "unavailable"},
		{"registered patched", map[string]string{registeredRunscPath + " --version": gVisorRelease, registeredRunscPath + " trace metadata": patched}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputs := map[string]string{}
			for key, value := range base {
				outputs[key] = value
			}
			for key, value := range test.outputs {
				outputs[key] = value
			}
			if test.name == "missing registered path" {
				outputs["docker info --format {{json (index .Runtimes \"runsc-trace\")}}"] = "{}"
			}
			got, err := probeGVisorRuntime(context.Background(), "linux", fakeExecutor{outputs: outputs}, "linux", "unavailable", "unsupported")
			if err != nil || got != test.want {
				t.Fatalf("probe=%q err=%v", got, err)
			}
		})
	}
}

type fakeExecutor struct {
	lookupError error
	outputs     map[string]string
}

func (f fakeExecutor) LookPath(string) (string, error) {
	if f.lookupError != nil {
		return "", f.lookupError
	}
	return "/test/bin", nil
}
func (f fakeExecutor) Output(_ context.Context, binary string, arguments ...string) ([]byte, error) {
	value, found := f.outputs[binary+" "+join(arguments)]
	if !found && binary == registeredRunscPath {
		value, found = f.outputs["runsc "+join(arguments)]
	}
	if !found {
		return nil, errors.New("unavailable")
	}
	return []byte(value), nil
}
func join(values []string) string {
	result := ""
	for index, value := range values {
		if index != 0 {
			result += " "
		}
		result += value
	}
	return result
}
