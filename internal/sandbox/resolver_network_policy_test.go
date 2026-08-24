package sandbox

import (
	"context"
	"encoding/base64"
	"errors"
	"net/netip"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestResolverNetworkPolicyUsesOnlyTypedService(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), []byte("172.30.0.0/24")}}
	service := &recordingResolverPolicyService{}
	policy, err := NewResolverNetworkPolicy(runner, service)
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
	if err := policy.Close(context.Background()); err != nil || service.remove != 1 {
		t.Fatalf("typed cleanup error=%v calls=%#v", err, service)
	}
}

func TestResolverNetworkPolicyFailsClosedOnServiceOrCleanupFailure(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), []byte("172.30.0.0/24")}}
	service := &recordingResolverPolicyService{createErr: errors.New("denied")}
	policy, _ := NewResolverNetworkPolicy(runner, service)
	if _, err := policy.Prepare(context.Background(), []netip.Addr{netip.MustParseAddr("1.1.1.1")}); err == nil {
		t.Fatal("Prepare accepted failed typed service")
	}
}

func TestValidateResolverEndpointsRejectsUnsafeOrAmbiguousInputs(t *testing.T) {
	for _, endpoints := range [][]netip.Addr{nil, {netip.MustParseAddr("127.0.0.1")}, {netip.MustParseAddr("10.0.0.1")}, {netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1.1.1.1")}, {netip.MustParseAddr("2606:4700:4700::1111")}} {
		if err := validateResolverEndpoints(endpoints); err == nil {
			t.Fatalf("validateResolverEndpoints(%v) error = nil", endpoints)
		}
	}
}

func TestNPMResolverReturnsOnlyParsedGraphAfterTypedPolicyLifecycle(t *testing.T) {
	lock := resolverLockJSON()
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), []byte("172.30.0.0/24"), []byte("0123456789ab"), nil, []byte(resolverNPMVersion), nil, []byte(lock)}}
	observer := &recordingObserver{reader: &traceReader{records: []TraceRecord{{Kind: "network-attempt", Bytes: 1}}}}
	service := &recordingResolverPolicyService{}
	resolver, err := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, observer, service)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := artifactnpm.ParseReference("primary@1.0.0")
	target, _ := domain.NewInstallTarget("/tmp/target")
	installContext, _ := domain.NewInstallContext(target)
	resolution, err := resolver.ResolveDependencies(context.Background(), reference, installContext)
	if err != nil || len(resolution.Graph().Nodes()) != 1 || resolution.RuntimeIdentity() != resolverRuntimeIdentity || resolution.LockfileDigest().String() == "" {
		t.Fatalf("ResolveDependencies() = %#v, %v", resolution, err)
	}
	if service.create != 1 || service.verify != 1 || service.remove != 1 || observer.containerID != "0123456789ab" {
		t.Fatalf("typed service=%#v observer=%q", service, observer.containerID)
	}
	for _, call := range runner.calls {
		if call.binary == "iptables" || call.binary == "nft" {
			t.Fatalf("resolver executed firewall tool: %#v", call)
		}
	}
}

func TestNPMNetworkArgumentsPinsOnlyValidatedAddressesInStableOrder(t *testing.T) {
	arguments, err := npmNetworkArguments([]netip.Addr{netip.MustParseAddr("1.1.1.2"), netip.MustParseAddr("1.1.1.1")})
	if err != nil || !sameStrings(arguments, []string{"--add-host", "registry.npmjs.org:1.1.1.1", "--add-host", "registry.npmjs.org:1.1.1.2"}) {
		t.Fatalf("npm network arguments = %#v, %v", arguments, err)
	}
}

type recordingResolverPolicyService struct {
	create, verify, remove int
	createErr              error
}

func (s *recordingResolverPolicyService) NetworkLabels(session domain.SandboxSessionID) map[string]string {
	return map[string]string{"io.heliopause.resolver-policy": "m8", "io.heliopause.resolver-session": session.String()}
}
func (s *recordingResolverPolicyService) Create(context.Context, domain.SandboxSessionID, string, netip.Prefix, []netip.Addr) error {
	s.create++
	return s.createErr
}
func (s *recordingResolverPolicyService) Verify(context.Context, domain.SandboxSessionID, string, netip.Prefix, []netip.Addr) error {
	s.verify++
	return nil
}
func (s *recordingResolverPolicyService) Remove(context.Context, domain.SandboxSessionID, string, netip.Prefix, []netip.Addr) error {
	s.remove++
	return nil
}

type staticEndpoints struct{ addresses []netip.Addr }

func (s staticEndpoints) Resolve(context.Context, []string) ([]netip.Addr, error) {
	return append([]netip.Addr(nil), s.addresses...), nil
}

func resolverLockJSON() string {
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(make([]byte, 64))
	return "{\"lockfileVersion\":3,\"packages\":{\"\":{\"dependencies\":{\"primary\":\"1.0.0\"}},\"node_modules/primary\":{\"version\":\"1.0.0\",\"resolved\":\"https://registry.npmjs.org/primary/-/primary-1.0.0.tgz\",\"integrity\":\"" + integrity + "\"}}}"
}
