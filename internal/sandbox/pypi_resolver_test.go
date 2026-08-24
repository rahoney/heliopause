package sandbox

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const resolverTestSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestPyPIResolverUsesGVisorDefaultDenyLifecycleAndCrossChecks(t *testing.T) {
	t.Parallel()

	report := pypiResolverReportJSON()
	runner := &recordingRunner{responses: [][]byte{
		[]byte("0123456789abcdef"), []byte("172.30.0.0/24"),
		[]byte("0123456789abcdef"), nil, []byte("3.14.7\n"), []byte("pip 26.2.1 from /usr/local/lib/python3.14/site-packages/pip\n"), []byte("Compatible tags:\n  cp314-cp314-manylinux_2_36_x86_64\n"), nil,
		[]byte(report), []byte(pypiResolverSimpleJSON("child", "child-2.0-py3-none-any.whl", "")), []byte(pypiResolverSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.14")),
	}}
	observer := &recordingObserver{reader: &traceReader{records: []TraceRecord{{Kind: "network-attempt", Bytes: 1}}}}
	resolver, err := NewPyPIResolver(runner, pypiStaticEndpoints{}, observer, availablePythonProbe, &recordingResolverPolicyService{})
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactpypi.ParseReference("Primary@1.0")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/target")
	installContext, _ := domain.NewInstallContext(target)
	resolution, err := resolver.ResolveDependencies(context.Background(), reference, installContext)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Graph().Nodes()) != 2 || len(resolution.Graph().Edges()) != 1 || resolution.RuntimeIdentity() == "" || resolution.LockfileDigest().String() == "" {
		t.Fatalf("resolution = %#v", resolution)
	}
	if observer.containerID != "0123456789abcdef" {
		t.Fatalf("observer container = %q", observer.containerID)
	}
	if len(runner.calls) != 13 {
		t.Fatalf("command calls = %d: %#v", len(runner.calls), runner.calls)
	}
	assertPyPIResolverCreate(t, runner.calls[2].arguments)
	if got := runner.calls[7]; got.binary != "docker" || strings.Contains(strings.Join(got.arguments, " "), "--only-binary") || !strings.Contains(strings.Join(got.arguments, " "), "primary==1.0") {
		t.Fatalf("pip resolution command = %#v", got)
	}
	if got := runner.calls[len(runner.calls)-1]; got.binary != "docker" || got.arguments[0] != "network" || got.arguments[1] != "rm" {
		t.Fatalf("network policy cleanup missing: %#v", got)
	}
	for _, call := range runner.calls[len(runner.calls)-5:] {
		if !call.bounded {
			t.Fatalf("cleanup call is unbounded: %#v", call)
		}
	}
}

func TestPyPIResolverFailsClosedForObserverOrEndpointFailure(t *testing.T) {
	t.Parallel()

	reference, _ := artifactpypi.ParseReference("primary@1.0")
	target, _ := domain.NewInstallTarget("/tmp/target")
	installContext, _ := domain.NewInstallContext(target)
	tests := []struct {
		name      string
		endpoints NamedEndpointResolver
		observer  TraceObserver
		responses [][]byte
	}{
		{
			name:      "untrusted endpoint",
			endpoints: pypiUnsafeEndpoints{},
			observer:  &emptyObserver{},
		},
		{
			name:      "observer stream incomplete",
			endpoints: pypiStaticEndpoints{},
			observer:  &recordingObserver{reader: &traceReader{err: errors.New("observer ended")}},
			responses: [][]byte{[]byte("0123456789abcdef"), []byte("172.30.0.0/24"), []byte("0123456789abcdef"), nil, []byte("3.14.7"), []byte("pip 26.2.1 from pip"), []byte("cp314-cp314-manylinux_2_36_x86_64"), nil, []byte(pypiResolverReportJSON()), []byte(pypiResolverSimpleJSON("child", "child-2.0-py3-none-any.whl", "")), []byte(pypiResolverSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.14"))},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			runner := &recordingRunner{responses: test.responses}
			resolver, err := NewPyPIResolver(runner, test.endpoints, test.observer, availablePythonProbe, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			if resolution, err := resolver.ResolveDependencies(context.Background(), reference, installContext); err == nil || len(resolution.Graph().Nodes()) != 0 {
				t.Fatalf("ResolveDependencies() = %#v, %v", resolution, err)
			}
		})
	}
}

func TestPyPINetworkArgumentsPinOnlyExpectedIPv4Addresses(t *testing.T) {
	t.Parallel()

	addresses, arguments, err := pypiNetworkArguments(map[string][]netip.Addr{
		pypiIndexEndpoint:        {netip.MustParseAddr("1.1.1.1")},
		pypiDistributionEndpoint: {netip.MustParseAddr("2.2.2.2")},
	})
	if err != nil || len(addresses) != 2 || len(arguments) != 4 {
		t.Fatalf("pypiNetworkArguments() = %v, %v, %v", addresses, arguments, err)
	}
	if _, _, err := pypiNetworkArguments(map[string][]netip.Addr{
		pypiIndexEndpoint:        {netip.MustParseAddr("127.0.0.1")},
		pypiDistributionEndpoint: {netip.MustParseAddr("2.2.2.2")},
	}); err == nil {
		t.Fatal("pypiNetworkArguments accepted loopback endpoint")
	}
}

func assertPyPIResolverCreate(t *testing.T, arguments []string) {
	t.Helper()
	joined := strings.Join(arguments, " ")
	for _, required := range []string{"--pull never", "--runtime " + gVisorRuntimeName, "--network haa-resolver-", "--read-only", "--cap-drop ALL", "no-new-privileges", "--add-host pypi.org:", "--add-host files.pythonhosted.org:", pythonImageReference} {
		if !strings.Contains(joined, required) {
			t.Errorf("create command missing %q: %q", required, joined)
		}
	}
	for _, forbidden := range []string{"--mount", "--volume", "-v ", "--env", "--privileged", "--network host", "/var/run/docker.sock"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("create command contains forbidden %q: %q", forbidden, joined)
		}
	}
}

func availablePythonProbe(context.Context) (PythonCapability, error) {
	return PythonCapability{Available: true, Runtime: PinnedPythonRuntime()}, nil
}

type pypiStaticEndpoints struct{}

func (pypiStaticEndpoints) Resolve(context.Context, []string) (map[string][]netip.Addr, error) {
	return map[string][]netip.Addr{
		pypiIndexEndpoint:        {netip.MustParseAddr("1.1.1.1")},
		pypiDistributionEndpoint: {netip.MustParseAddr("2.2.2.2")},
	}, nil
}

type pypiUnsafeEndpoints struct{}

func (pypiUnsafeEndpoints) Resolve(context.Context, []string) (map[string][]netip.Addr, error) {
	return map[string][]netip.Addr{
		pypiIndexEndpoint:        {netip.MustParseAddr("127.0.0.1")},
		pypiDistributionEndpoint: {netip.MustParseAddr("2.2.2.2")},
	}, nil
}

func pypiResolverSimpleJSON(project, filename, requiresPython string) string {
	return `{"meta":{"api-version":"1.4"},"name":"` + project + `","files":[{"filename":"` + filename + `","url":"https://files.pythonhosted.org/packages/` + filename + `","hashes":{"sha256":"` + resolverTestSHA256 + `"},"requires-python":"` + requiresPython + `","yanked":false,"size":123}]}`
}

func pypiResolverReportJSON() string {
	return `{"version":"1","pip_version":"26.2.1","environment":{"implementation_name":"cpython","implementation_version":"3.14.7","python_full_version":"3.14.7","platform_machine":"x86_64","sys_platform":"linux"},"install":[{"download_info":{"url":"https://files.pythonhosted.org/packages/primary-1.0-py3-none-any.whl","archive_info":{"hash":"sha256=` + resolverTestSHA256 + `","hashes":{"sha256":"` + resolverTestSHA256 + `"}}},"is_direct":false,"requested":true,"metadata":{"name":"primary","version":"1.0","requires_python":">=3.14","requires_dist":["child>=2"]}},{"download_info":{"url":"https://files.pythonhosted.org/packages/child-2.0-py3-none-any.whl","archive_info":{"hash":"sha256=` + resolverTestSHA256 + `","hashes":{"sha256":"` + resolverTestSHA256 + `"}}},"is_direct":false,"requested":false,"metadata":{"name":"child","version":"2.0","requires_python":"","requires_dist":[]}}]}`
}
