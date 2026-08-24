package sandbox

import (
	"context"
	"encoding/base64"
	"errors"
	"net/netip"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestResolverNetworkPolicyIPTablesCreatesVerifiesAndCleansUp(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{
		[]byte("iptables"), []byte("network-id"), []byte("172.30.0.0/24"),
	}}
	policy, err := NewResolverNetworkPolicy(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	if got := runner.calls[0]; got.binary != "docker" || !sameStrings(got.arguments, []string{"info", "--format", "{{.FirewallBackend.Driver}}"}) {
		t.Fatalf("firewall backend query = %#v", got)
	}
	name, err := policy.Prepare(context.Background(), []netip.Addr{netip.MustParseAddr("1.1.1.1")})
	if err != nil || name == "" {
		t.Fatalf("Prepare() = %q, %v", name, err)
	}
	if len(runner.calls) != 9 {
		t.Fatalf("prepare calls = %#v", runner.calls)
	}
	if got := runner.calls[3]; got.binary != "iptables" || got.arguments[0] != "-N" {
		t.Fatalf("chain create = %#v", got)
	}
	if got := runner.calls[4]; got.binary != "iptables" || got.arguments[0] != "-I" || got.arguments[1] != "DOCKER-USER" {
		t.Fatalf("chain attach = %#v", got)
	}
	if got := runner.calls[6]; got.binary != "iptables" || got.arguments[0] != "-A" || got.arguments[len(got.arguments)-1] != "DROP" {
		t.Fatalf("default deny = %#v", got)
	}
	if err := policy.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := runner.calls[len(runner.calls)-1]; got.binary != "docker" || !sameStrings(got.arguments, []string{"network", "rm", name}) {
		t.Fatalf("network cleanup = %#v", got)
	}
}

func TestResolverNetworkPolicyDelegatesFirewallLifecycleToTypedService(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{
		[]byte("0123456789abcdef"), []byte("172.30.0.0/24"),
	}}
	service := &recordingResolverPolicyService{}
	policy, err := NewResolverNetworkPolicyWithService(runner, service)
	if err != nil {
		t.Fatal(err)
	}
	policy.newID = func() (domain.SandboxSessionID, error) {
		return domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	}
	name, err := policy.Prepare(context.Background(), []netip.Addr{netip.MustParseAddr("1.1.1.1")})
	if err != nil {
		t.Fatal(err)
	}
	if name != "haa-resolver-sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa" || service.create != 1 || service.verify != 1 {
		t.Fatalf("typed lifecycle name=%q calls=%#v", name, service)
	}
	for _, call := range runner.calls {
		if call.binary == "iptables" || call.binary == "nft" {
			t.Fatalf("ordinary resolver executed firewall tool: %#v", call)
		}
	}
	if err := policy.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if service.remove != 1 {
		t.Fatalf("typed cleanup calls=%#v", service)
	}
}

type recordingResolverPolicyService struct{ create, verify, remove int }

func (s *recordingResolverPolicyService) NetworkLabels(session domain.SandboxSessionID) map[string]string {
	return map[string]string{"io.heliopause.resolver-policy": "m8", "io.heliopause.resolver-session": session.String()}
}
func (s *recordingResolverPolicyService) Create(context.Context, domain.SandboxSessionID, string, netip.Prefix, []netip.Addr) error {
	s.create++
	return nil
}
func (s *recordingResolverPolicyService) Verify(context.Context, domain.SandboxSessionID, string, netip.Prefix, []netip.Addr) error {
	s.verify++
	return nil
}
func (s *recordingResolverPolicyService) Remove(context.Context, domain.SandboxSessionID, string, netip.Prefix, []netip.Addr) error {
	s.remove++
	return nil
}

func TestResolverNetworkPolicyRejectsUnknownBackendAndFailedCleanup(t *testing.T) {
	unknown := &recordingRunner{responses: [][]byte{[]byte("unknown")}}
	if _, err := NewResolverNetworkPolicy(context.Background(), unknown); err == nil {
		t.Fatal("NewResolverNetworkPolicy() accepted an unknown backend")
	}

	runner := &recordingRunner{
		responses: [][]byte{[]byte("iptables"), []byte("network-id"), []byte("172.30.0.0/24")},
		errors:    []error{nil, nil, nil, nil, nil, nil, nil, nil, errors.New("detach failed")},
	}
	policy, err := NewResolverNetworkPolicy(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := policy.Prepare(context.Background(), []netip.Addr{netip.MustParseAddr("1.1.1.1")}); err == nil {
		t.Fatal("Prepare() accepted a policy verification failure")
	}
}

func TestResolverNetworkPolicyDefersBackendProbeUntilBridgeExists(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{[]byte("<no value>")}}
	policy, err := NewResolverNetworkPolicy(context.Background(), runner)
	if err != nil || policy.backend != "" {
		t.Fatalf("NewResolverNetworkPolicy() = %#v, %v", policy, err)
	}
	if len(runner.calls) != 1 || runner.calls[0].binary != "docker" {
		t.Fatalf("backend probe calls = %#v", runner.calls)
	}
}

func TestResolverNetworkPolicyUsesNFTablesOnlyWhenIPTablesHookIsUnavailable(t *testing.T) {
	runner := &recordingRunner{errors: []error{errors.New("DOCKER-USER unavailable"), nil}}
	backend, err := probeDockerFirewallBackend(context.Background(), runner)
	if err != nil || backend != firewallBackendNFTables {
		t.Fatalf("probeDockerFirewallBackend() = %q, %v", backend, err)
	}
}

func TestResolverNetworkPolicyNFTablesUsesSeparateTable(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{
		[]byte("nftables"), []byte("network-id"), []byte("172.30.0.0/24"),
	}}
	policy, err := NewResolverNetworkPolicy(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	name, err := policy.Prepare(context.Background(), []netip.Addr{netip.MustParseAddr("1.1.1.1")})
	if err != nil {
		t.Fatal(err)
	}
	if got := runner.calls[3]; got.binary != "nft" || !sameStrings(got.arguments[:4], []string{"add", "table", "ip", got.arguments[3]}) {
		t.Fatalf("nft table create = %#v", got)
	}
	if got := runner.calls[4]; got.binary != "nft" || got.arguments[0] != "add" || got.arguments[1] != "chain" {
		t.Fatalf("nft chain create = %#v", got)
	}
	if got := runner.calls[6]; got.binary != "nft" || got.arguments[len(got.arguments)-1] != "drop" {
		t.Fatalf("nft default deny = %#v", got)
	}
	if err := policy.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := runner.calls[len(runner.calls)-2]; got.binary != "nft" || !sameStrings(got.arguments[:3], []string{"delete", "table", "ip"}) {
		t.Fatalf("nft table cleanup = %#v", got)
	}
	if got := runner.calls[len(runner.calls)-1]; got.binary != "docker" || !sameStrings(got.arguments, []string{"network", "rm", name}) {
		t.Fatalf("network cleanup = %#v", got)
	}
}

func TestValidateResolverEndpointsRejectsUnsafeOrAmbiguousInputs(t *testing.T) {
	for _, endpoints := range [][]netip.Addr{
		nil,
		{netip.MustParseAddr("127.0.0.1")},
		{netip.MustParseAddr("10.0.0.1")},
		{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1.1.1.1")},
		{netip.MustParseAddr("2606:4700:4700::1111")},
	} {
		if err := validateResolverEndpoints(endpoints); err == nil {
			t.Fatalf("validateResolverEndpoints(%v) error = nil", endpoints)
		}
	}
}

func TestNPMResolverReturnsOnlyParsedGraphAfterPolicyProtectedLifecycle(t *testing.T) {
	lock := resolverLockJSON()
	runner := &recordingRunner{responses: [][]byte{
		[]byte("iptables"), []byte("network-id"), []byte("172.30.0.0/24"),
		nil, nil, nil, nil, nil, nil,
		[]byte("0123456789ab"), nil, []byte(resolverNPMVersion), nil, []byte(lock),
	}}
	observer := &recordingObserver{reader: &traceReader{records: []TraceRecord{{Kind: "network-attempt", Bytes: 1}}}}
	resolver, err := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, observer)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactnpm.ParseReference("primary@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/target")
	installContext, _ := domain.NewInstallContext(target)
	resolution, err := resolver.ResolveDependencies(context.Background(), reference, installContext)
	if err != nil || len(resolution.Graph().Nodes()) != 1 || resolution.RuntimeIdentity() != resolverRuntimeIdentity || resolution.LockfileDigest().String() == "" {
		t.Fatalf("ResolveDependencies() = %#v, %v", resolution, err)
	}
	if len(runner.inputCalls) != 1 || runner.inputCalls[0].binary != "docker" {
		t.Fatalf("manifest input = %#v", runner.inputCalls)
	}
	if observer.containerID != "0123456789ab" {
		t.Fatalf("resolver observer container = %q", observer.containerID)
	}
	var create commandCall
	for _, call := range runner.calls {
		if call.binary == "docker" && len(call.arguments) > 1 && call.arguments[0] == "create" {
			create = call
			break
		}
	}
	if !containsSubsequence(create.arguments, []string{"--add-host", "registry.npmjs.org:1.1.1.1"}) {
		t.Fatalf("resolver container does not pin preflight registry address: %#v", create)
	}
	if got := string(runner.input); !strings.Contains(got, "\"primary\":\"1.0.0\"") {
		t.Fatalf("manifest = %q", got)
	}
	if got := runner.calls[len(runner.calls)-1]; got.binary != "docker" || got.arguments[0] != "network" {
		t.Fatalf("policy cleanup missing: %#v", got)
	}
	for _, call := range runner.calls[len(runner.calls)-5:] {
		if !call.bounded {
			t.Fatalf("cleanup call is unbounded: %#v", call)
		}
	}
}

func TestNPMNetworkArgumentsPinsOnlyValidatedAddressesInStableOrder(t *testing.T) {
	arguments, err := npmNetworkArguments([]netip.Addr{netip.MustParseAddr("1.1.1.2"), netip.MustParseAddr("1.1.1.1")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--add-host", "registry.npmjs.org:1.1.1.1", "--add-host", "registry.npmjs.org:1.1.1.2"}
	if !sameStrings(arguments, want) {
		t.Fatalf("npm network arguments = %#v, want %#v", arguments, want)
	}
	if _, err := npmNetworkArguments([]netip.Addr{netip.MustParseAddr("127.0.0.1")}); err == nil {
		t.Fatal("npm network arguments accepted unsafe address")
	}
}

func containsSubsequence(values, want []string) bool {
	if len(want) == 0 {
		return true
	}
	for offset := 0; offset+len(want) <= len(values); offset++ {
		if sameStrings(values[offset:offset+len(want)], want) {
			return true
		}
	}
	return false
}

type staticEndpoints struct{ addresses []netip.Addr }

func (s staticEndpoints) Resolve(context.Context, []string) ([]netip.Addr, error) {
	return append([]netip.Addr(nil), s.addresses...), nil
}

func resolverLockJSON() string {
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(make([]byte, 64))
	return "{\"lockfileVersion\":3,\"packages\":{\"\":{\"dependencies\":{\"primary\":\"1.0.0\"}},\"node_modules/primary\":{\"version\":\"1.0.0\",\"resolved\":\"https://registry.npmjs.org/primary/-/primary-1.0.0.tgz\",\"integrity\":\"" + integrity + "\"}}}"
}
