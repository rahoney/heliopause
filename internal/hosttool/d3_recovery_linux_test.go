//go:build linux

package hosttool

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/networkpolicy"
	"golang.org/x/sys/unix"
)

func newD3Fixture(t *testing.T, runner policyRunner) *d3Recovery {
	t.Helper()
	journal, err := openD3Journal(filepath.Join(t.TempDir(), "root-state"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = journal.lock.Close() })
	return &d3Recovery{journal: journal, runner: runner}
}

func TestD3RetiresOnlyRecordedObserverInode(t *testing.T) {
	directory := t.TempDir()
	name := ".heliopause-supervisor.lock"
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		t.Fatal(err)
	}
	endpoint := d3Endpoint{Name: name, Dev: st.Dev, Ino: st.Ino}
	foreign := endpoint
	foreign.Ino++
	if err := removeD3ObserverEndpoint(path, foreign, st.Uid); !errors.Is(err, errD3Unconfirmed) {
		t.Fatalf("foreign inode recovery: %v", err)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("foreign inode changed: %v", err)
	}
	if err := removeD3ObserverEndpoint(path, endpoint, st.Uid); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(directory); err != nil || len(entries) != 0 {
		t.Fatalf("endpoint or retirement directory remained: %v, %v", entries, err)
	}
}

func TestD3JournalKeepsUncertainStateAcrossRestart(t *testing.T) {
	ctx := context.Background()
	runner := &d3TestRunner{}
	r := newD3Fixture(t, runner)
	peer := d3Peer{uid: 1000, pid: 30, start: "12"}
	owner, err := r.acquire(peer)
	if err != nil {
		t.Fatal(err)
	}
	generation, err := r.authorize(peer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.authorize(d3Peer{uid: peer.uid, pid: peer.pid, start: "13"}); err == nil {
		t.Fatal("reused PID was accepted")
	}
	if err := r.dockerRequest(ctx, generation, observationResourceRequest{Operation: "guard-create", Owner: owner}); err != nil {
		t.Fatal(err)
	}
	directory := r.journal.directory
	if err := r.journal.lock.Close(); err != nil {
		t.Fatal(err)
	}
	journal, err := openD3Journal(directory, false)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.lock.Close()
	restarted := &d3Recovery{journal: journal, runner: runner}
	if err := restarted.recover(ctx, true); !errors.Is(err, errD3Unconfirmed) {
		t.Fatalf("late create returned %v", err)
	}
	if _, err := restarted.acquire(peer); err == nil || restarted.journal.record == nil || restarted.journal.record.Mutations != 1 {
		t.Fatal("empty Docker inventory erased an issued create")
	}
}

func TestD3FailedCreateAcknowledgementRemainsUncertain(t *testing.T) {
	ctx := context.Background()
	peer := d3Peer{uid: 1000, pid: 31, start: "12"}
	runner := &d3TestRunner{id: strings.Repeat("a", 64)}
	r := newD3Fixture(t, runner)
	owner, err := r.acquire(peer)
	if err != nil {
		t.Fatal(err)
	}
	runner.owner = owner
	generation, err := r.authorize(peer)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.dockerRequest(ctx, generation, observationResourceRequest{Operation: "guard-create", Owner: owner}); err != nil {
		t.Fatal(err)
	}
	directory := r.journal.directory
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	r.journal.directory = blocked
	if err := r.dockerRequest(ctx, generation, observationResourceRequest{Operation: "guard-created", Owner: owner}); !errors.Is(err, errD3Unconfirmed) {
		t.Fatalf("write failure was not reported: %v", err)
	}
	if _, err := r.authorize(peer); !errors.Is(err, errD3Unconfirmed) {
		t.Fatalf("admission after write failure: %v", err)
	}
	if err := r.journal.lock.Close(); err != nil {
		t.Fatal(err)
	}
	journal, err := openD3Journal(directory, false)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.lock.Close()
	restarted := &d3Recovery{journal: journal, runner: runner}
	if err := restarted.recover(ctx, true); !errors.Is(err, errD3Unconfirmed) || runner.removes != 0 {
		t.Fatalf("unobserved create was reconciled: %v", err)
	}
	// Docker reports the owned object only after the first empty snapshot.
	runner.present = true
	if err := restarted.recover(ctx, true); !errors.Is(err, errD3Unconfirmed) || runner.present || runner.removes != 1 {
		t.Fatalf("late create recovery=%v present=%v removes=%d", err, runner.present, runner.removes)
	}
	if j := journal.record; j == nil || j.Owner != owner || j.Mutations != 1 {
		t.Fatal("late removal erased uncertainty about in-flight creation")
	}
	contender := d3Peer{uid: peer.uid, pid: 32, start: "13"}
	if _, err := restarted.acquire(contender); !errors.Is(err, errD3Unconfirmed) {
		t.Fatalf("admission after ambiguous creation: %v", err)
	}
}

func TestD3StopsRecordedContainerBeforeDiscardingState(t *testing.T) {
	ctx := context.Background()
	id := strings.Repeat("a", 64)
	runner := &d3TestRunner{id: id, present: true}
	r := newD3Fixture(t, runner)
	peer := d3Peer{uid: 1000, pid: 11, start: "12"}
	owner, err := r.acquire(peer)
	if err != nil {
		t.Fatal(err)
	}
	runner.owner = owner
	generation, err := r.authorize(peer)
	if err != nil {
		t.Fatal(err)
	}
	request := observationResourceRequest{Operation: "guard-start", Owner: owner, Container: id}
	if err := r.dockerRequest(ctx, generation, request); err != nil {
		t.Fatal(err)
	}
	r.closeAdmission(peer)
	if err := r.dockerRequest(ctx, generation, request); err == nil {
		t.Fatal("previously authorized request admitted during recovery")
	}
	runner.withholdRemoval = true
	if err := r.recover(ctx, false); err == nil || r.journal.record == nil {
		t.Fatal("unverified remove was accepted")
	}
	runner.withholdRemoval = false
	if err := r.recover(ctx, false); err != nil || r.journal.record != nil || runner.present || runner.removes != 2 {
		t.Fatalf("recovery = %v, removes = %d", err, runner.removes)
	}
	if _, err := r.acquire(peer); err != nil {
		t.Fatal(err)
	}
}

func TestD3MissingJournalIsNotFirstStart(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "d3")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if j, err := openD3Journal(directory, true); j != nil || err == nil {
		t.Fatal("missing previous journal was treated as initial enrollment")
	}
}

func TestD3ObserverLossRevokesNewDockerRequests(t *testing.T) {
	r := newD3Fixture(t, &d3TestRunner{})
	peer := d3Peer{uid: 1000, pid: 30, start: "12"}
	owner, err := r.acquire(peer)
	if err != nil {
		t.Fatal(err)
	}
	generation, err := r.authorize(peer)
	if err != nil {
		t.Fatal(err)
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	fd, err := unix.Dup(int(read.Fd()))
	if err != nil {
		_ = write.Close()
		t.Fatal(err)
	}
	finished := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() {
		defer close(finished)
		r.watchObserver(ctx, generation, fd)
	}()
	if err := write.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-ctx.Done():
		t.Fatal("observer loss did not revoke admission")
	}
	request := observationResourceRequest{Operation: "guard-create", Owner: owner}
	if err := r.dockerRequest(ctx, generation, request); !errors.Is(err, errD3Unconfirmed) {
		t.Fatalf("new create after observer loss: %v", err)
	}
}

func TestD3ResumesOwnedFirewallRemovalAfterJumpIsGone(t *testing.T) {
	base, err := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := networkpolicy.NewResolverPolicy(base, strings.Repeat("a", 64), netip.MustParsePrefix("172.30.2.0/24"),
		[]netip.Addr{netip.MustParseAddr("1.1.1.1")})
	if err != nil {
		t.Fatal(err)
	}
	chain := policyChain(policy)
	known := "-N " + chain + "\n-A " + chain + " -j DROP\n"
	for _, scenario := range []struct {
		name, listing string
		want          []string
	}{
		{name: "owned remainder", listing: known, want: []string{"-F " + chain, "-X " + chain}},
		{name: "unrelated reference", listing: known + "-A OTHER -j " + chain + "\n"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			runner := &d3FirewallFixture{listing: scenario.listing}
			recovery := d3Recovery{runner: runner}
			err := recovery.removeRecordedIPTables(context.Background(), policyRecord{policy: policy, backend: policyIPTables, chain: chain})
			if (err == nil) != (len(scenario.want) != 0) || !reflect.DeepEqual(runner.mutations, scenario.want) {
				t.Fatalf("recovery %v performed mutations %v", err, runner.mutations)
			}
		})
	}
}

func TestD3RestartReconcilesOnlyRecordedPolicyRemoval(t *testing.T) {
	ctx := context.Background()
	session, err := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := networkpolicy.NewResolverPolicy(session, strings.Repeat("a", 64), netip.MustParsePrefix("172.30.2.0/24"),
		[]netip.Addr{netip.MustParseAddr("1.1.1.1")})
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"recorded removal", "removal and unknown create"} {
		t.Run(scenario, func(t *testing.T) {
			chain := policyChain(policy)
			runner := &d3PolicyRestartFixture{policy: policy, firewall: d3FirewallFixture{listing: "-N " + chain + "\n-A " + chain + " -j DROP\n"}}
			r := newD3Fixture(t, runner)
			peer := d3Peer{uid: 1000, pid: 123, start: "15"}
			owner, err := r.acquire(peer)
			if err != nil {
				t.Fatal(err)
			}
			generation, err := r.authorize(peer)
			if err != nil {
				t.Fatal(err)
			}
			row := policyRecord{uid: generation, policy: policy, backend: policyIPTables, chain: chain}
			if err := r.beginPolicyMutation(row, true); err != nil {
				t.Fatal(err)
			}
			if err := r.endPolicyMutation(row, true); err != nil {
				t.Fatal(err)
			}
			if err := r.beginPolicyMutation(row, false); err != nil {
				t.Fatal(err)
			}
			if scenario == "removal and unknown create" {
				if err := r.dockerRequest(ctx, generation, observationResourceRequest{Operation: "guard-create", Owner: owner}); err != nil {
					t.Fatal(err)
				}
			}
			r.closeAdmission(peer)
			if err := r.recover(ctx, false); !errors.Is(err, errD3Unconfirmed) || len(runner.firewall.mutations) != 0 {
				t.Fatalf("live helper could still be removing the policy: %v", err)
			}
			if err := r.journal.lock.Close(); err != nil {
				t.Fatal(err)
			}
			journal, err := openD3Journal(r.journal.directory, false)
			if err != nil {
				t.Fatal(err)
			}
			defer journal.lock.Close()
			restarted := &d3Recovery{journal: journal, runner: runner}
			err = restarted.recover(ctx, true)
			if scenario == "removal and unknown create" {
				if !errors.Is(err, errD3Unconfirmed) || journal.record == nil || journal.record.Mutations != 2 || len(runner.firewall.mutations) != 0 {
					t.Fatalf("lost create was confused with policy removal: %v", err)
				}
				return
			}
			want := []string{"-F " + chain, "-X " + chain}
			if err != nil || journal.record != nil || !reflect.DeepEqual(runner.firewall.mutations, want) {
				t.Fatalf("replay error=%v mutations=%v", err, runner.firewall.mutations)
			}
		})
	}
}

type d3PolicyRestartFixture struct {
	policy   networkpolicy.ResolverPolicy
	firewall d3FirewallFixture
}

func (f *d3PolicyRestartFixture) Output(ctx context.Context, program string, args ...string) ([]byte, error) {
	switch program {
	case "iptables":
		body, err := f.firewall.Output(ctx, program, args...)
		if args[0] == "-X" && err == nil {
			f.firewall.listing = "-P INPUT ACCEPT\n"
		}
		return body, err
	case "docker":
		if args[0] == "network" && args[1] == "inspect" {
			return json.Marshal(map[string]any{
				"Id": f.policy.NetworkID(), "Name": networkpolicy.NetworkName(f.policy.Session()),
				"Labels": networkpolicy.NetworkLabels(f.policy.Session()),
				"IPAM":   map[string]any{"Config": []map[string]string{{"Subnet": f.policy.Subnet().String()}}},
			})
		}
		return nil, nil
	default:
		return nil, errors.New("unexpected replay command")
	}
}

type d3FirewallFixture struct {
	listing   string
	mutations []string
}

func (f *d3FirewallFixture) Output(_ context.Context, program string, args ...string) ([]byte, error) {
	if program != "iptables" {
		return nil, errors.New("unexpected program")
	}
	if len(args) == 1 && args[0] == "-S" {
		return []byte(f.listing), nil
	}
	f.mutations = append(f.mutations, strings.Join(args, " "))
	return nil, nil
}

type d3TestRunner struct {
	id, owner       string
	present         bool
	withholdRemoval bool
	removes         int
}

func (r *d3TestRunner) Output(_ context.Context, tool string, args ...string) ([]byte, error) {
	if tool != "docker" || len(args) == 0 {
		return nil, errors.New("unexpected tool")
	}
	switch args[0] {
	case "network", "volume":
		return nil, nil
	case "ps":
		if r.present {
			return []byte(r.id + "\n"), nil
		}
		return nil, nil
	case "inspect":
		return json.Marshal(map[string]any{
			"Id": r.id, "Config": map[string]any{"Labels": map[string]string{d3OwnerLabel: r.owner}},
		})
	case "rm":
		if len(args) != 3 || args[2] != r.id {
			return nil, errors.New("unowned removal")
		}
		r.removes++
		if !r.withholdRemoval {
			r.present = false
		}
		return nil, nil
	}
	return nil, errors.New("unexpected Docker request")
}
