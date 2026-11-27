//go:build linux

package hosttool

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/networkpolicy"
)

var errD3Unconfirmed = errors.New("previous protected execution could not be confirmed stopped and recovered")

type d3Recovery struct {
	mu           sync.Mutex
	journal      *d3Journal
	runner       policyRunner
	peer         d3Peer
	generation   uint32
	fault        bool
	observerDead bool
}

type d3Peer struct {
	uid   uint32
	pid   int32
	start string
}

type d3PolicyRecord struct {
	Session   string        `json:"session"`
	Network   string        `json:"network"`
	Subnet    netip.Prefix  `json:"subnet"`
	Endpoints []netip.Addr  `json:"endpoints"`
	Backend   policyBackend `json:"backend"`
	Removing  bool          `json:"removing,omitempty"`
}

func (r *d3Recovery) acquire(peer d3Peer) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fault || r.journal.record != nil || peer.pid <= 1 || peer.start == "" || r.generation == ^uint32(0) {
		return "", errD3Unconfirmed
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	owner := hex.EncodeToString(nonce[:])
	if err := r.write(&d3Record{Owner: owner}); err != nil {
		return "", err
	}
	r.generation++
	r.peer = peer
	r.observerDead = false
	return owner, nil
}

func (r *d3Recovery) authorize(peer d3Peer) (uint32, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fault || r.journal.record == nil || r.peer != peer || peer.pid <= 1 {
		return 0, errD3Unconfirmed
	}
	return r.generation, nil
}

func (r *d3Recovery) change(generation uint32, update func(*d3Record) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fault || r.journal.record == nil || generation == 0 || r.generation != generation || r.peer.pid == 0 {
		return errD3Unconfirmed
	}
	next := *r.journal.record
	next.Containers = slices.Clone(next.Containers)
	next.Keepers = slices.Clone(next.Keepers)
	next.Policies = slices.Clone(next.Policies)
	if err := update(&next); err != nil {
		return err
	}
	return r.write(&next)
}

func (r *d3Recovery) write(record *d3Record) error {
	if err := r.journal.write(record); err != nil {
		// A failed sync can leave the disk and this process with different
		// generations. Do not admit another request in this process.
		r.fault = true
		return errors.Join(errD3Unconfirmed, err)
	}
	return nil
}

// closeAdmission invalidates already authenticated requests as well as new
// requests. An issued start has a durable full container ID before this point.
func (r *d3Recovery) closeAdmission(peer d3Peer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.peer == peer {
		r.peer = d3Peer{}
	}
}

func (r *d3Recovery) recover(ctx context.Context, restarted bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	snapshot := r.journal.record
	if snapshot == nil {
		if r.fault {
			return errD3Unconfirmed
		}
		return nil
	}
	if r.peer.pid != 0 || !validProcessOwner(snapshot.Owner) {
		return errD3Unconfirmed
	}
	// Independently stop known executions even when a different create
	// request is still uncertain. Never clear the latter on an empty list.
	recordedErr := stopRecordedD3Containers(snapshot)
	observerErr := stopD3Observer(ctx, snapshot.Observer)
	err := errors.Join(recordedErr, observerErr, r.stopContainers(ctx, snapshot))
	if snapshot.ObserverPending || err != nil {
		return errors.Join(errD3Unconfirmed, err)
	}
	// The previous helper is stopped before a restart can replay a recorded
	// removal. A creation whose result was lost remains unconfirmed.
	var removals uint32
	for _, policy := range snapshot.Policies {
		if policy.Removing {
			removals++
		}
	}
	if snapshot.Mutations != removals || removals != 0 && !restarted {
		return errD3Unconfirmed
	}
	if err := r.recoverOtherResources(ctx, snapshot); err != nil {
		return errors.Join(errD3Unconfirmed, err)
	}
	if r.fault || r.journal.record.Mutations != 0 {
		return errD3Unconfirmed
	}
	return r.write(nil)
}

const d3OwnerLabel = "io.heliopause.process-owner"

func validProcessOwner(owner string) bool {
	decoded, err := hex.DecodeString(owner)
	return err == nil && len(decoded) == 16 && strings.ToLower(owner) == owner
}

func (r *d3Recovery) list(ctx context.Context, owner string, arguments ...string) ([]string, error) {
	filter := "label=" + d3OwnerLabel + "=" + owner
	args := append(slices.Clone(arguments), "--filter", filter, "--format", "{{.ID}}")
	body, err := r.runner.Output(ctx, "docker", args...)
	if err != nil || len(body) > d3JournalLimit {
		return nil, errD3Unconfirmed
	}
	items := strings.Fields(string(body))
	for _, id := range items {
		if !validDockerContainerID(id) {
			return nil, errD3Unconfirmed
		}
	}
	return items, nil
}

func stopRecordedD3Containers(record *d3Record) error {
	var result error
	for _, c := range record.Containers {
		if !validD3Scope(c.Scope, c.ID) {
			result = errors.Join(result, errD3Unconfirmed)
			continue
		}
		result = errors.Join(result, killObservationScope(c.Scope))
	}
	return result
}

func (r *d3Recovery) stopContainers(ctx context.Context, record *d3Record) error {
	ids, result := r.list(ctx, record.Owner, "ps", "-a", "--no-trunc")
	if len(record.Containers) != 0 {
		body, err := r.runner.Output(ctx, "docker", "ps", "-a", "--no-trunc", "--format", "{{.ID}}")
		if err != nil || len(body) > d3JournalLimit {
			return errors.Join(result, errD3Unconfirmed)
		}
		actual := make(map[string]bool)
		for _, id := range strings.Fields(string(body)) {
			if !validDockerContainerID(id) {
				return errors.Join(result, errD3Unconfirmed)
			}
			actual[id] = true
		}
		for _, c := range record.Containers {
			if actual[c.ID] {
				ids = append(ids, c.ID)
			}
		}
	}
	slices.Sort(ids)
	for _, id := range slices.Compact(ids) {
		if !validDockerContainerID(id) {
			result = errors.Join(result, errD3Unconfirmed)
			continue
		}
		scope, exists, err := r.inspectContainer(ctx, record.Owner, id)
		if err != nil {
			result = errors.Join(result, err)
			continue
		}
		if exists {
			// A failed Docker client is not proof of absence. Recheck the
			// full ID and its independently verified cgroup after removal.
			if scope != "" {
				result = errors.Join(result, killObservationScope(scope))
			}
			_, _ = r.runner.Output(ctx, "docker", "rm", "--force", id)
		}
		_, exists, err = r.inspectContainer(ctx, record.Owner, id)
		if err != nil || exists || scope != "" && verifyDrainedScope(scope) != nil {
			result = errors.Join(result, errD3Unconfirmed)
		}
	}
	for _, c := range record.Containers {
		if !validD3Scope(c.Scope, c.ID) || verifyDrainedScope(c.Scope) != nil {
			result = errors.Join(result, errD3Unconfirmed)
		}
	}
	remaining, err := r.list(ctx, record.Owner, "ps", "-a", "--no-trunc")
	if err != nil || len(remaining) != 0 {
		result = errors.Join(result, errD3Unconfirmed)
	}
	return result
}

func (r *d3Recovery) inspectContainer(ctx context.Context, owner, id string) (string, bool, error) {
	body, err := r.runner.Output(ctx, "docker", "ps", "-a", "--no-trunc", "--filter", "id="+id, "--format", "{{.ID}}")
	if err != nil || len(body) > 1024 {
		return "", false, errD3Unconfirmed
	}
	listed := strings.TrimSpace(string(body))
	if listed == "" {
		return "", false, nil
	}
	if listed != id {
		return "", false, errD3Unconfirmed
	}
	body, err = r.runner.Output(ctx, "docker", "inspect", "--format", "{{json .}}", id)
	var item struct {
		ID         string
		Config     struct{ Labels map[string]string }
		State      struct{ Pid int }
		HostConfig struct{ CgroupParent string }
	}
	if err != nil || len(body) > 64<<10 || json.Unmarshal(body, &item) != nil || item.ID != id || item.Config.Labels[d3OwnerLabel] != owner {
		return "", false, errD3Unconfirmed
	}
	parent := item.HostConfig.CgroupParent
	if parent == "" {
		parent = "system.slice"
	}
	if parent != "system.slice" && !validObservationParent(parent) {
		return "", false, errD3Unconfirmed
	}
	scope := filepath.Join("/sys/fs/cgroup", parent, "docker-"+id+".scope")
	if item.State.Pid > 0 {
		pidPath := filepath.Join("/proc", strconv.Itoa(item.State.Pid), "cgroup")
		cgroup, err := os.ReadFile(pidPath)
		if err != nil || len(cgroup) > 1024 || strings.TrimSpace(string(cgroup)) != "0::"+strings.TrimPrefix(scope, "/sys/fs/cgroup") {
			return "", false, errD3Unconfirmed
		}
	}
	return scope, true, nil
}

func validD3Scope(scope string, id string) bool {
	if !validDockerContainerID(id) || filepath.Clean(scope) != scope || filepath.Base(scope) != "docker-"+id+".scope" {
		return false
	}
	parent := filepath.Dir(scope)
	return filepath.Dir(parent) == "/sys/fs/cgroup" &&
		(filepath.Base(parent) == "system.slice" || validObservationParent(filepath.Base(parent)))
}

func (r *d3Recovery) recoverOtherResources(ctx context.Context, record *d3Record) error {
	if len(record.Policies) != 0 {
		engine, err := NewPolicyEngine(r.runner)
		if err != nil {
			return err
		}
		for _, saved := range record.Policies {
			policy, err := saved.policy()
			if err != nil {
				return errD3Unconfirmed
			}
			effect := policyRecord{policy: policy, backend: saved.Backend, chain: policyChain(policy)}
			absent, err := r.policyAbsent(ctx, effect)
			if err != nil {
				return err
			}
			if !absent {
				if engine.verifyNetwork(ctx, policy) != nil {
					return errD3Unconfirmed
				}
				if effect.backend == policyIPTables {
					err = r.removeRecordedIPTables(ctx, effect)
				} else if engine.verify(ctx, effect) != nil {
					err = errD3Unconfirmed
				} else {
					err = engine.remove(ctx, effect)
				}
				if err != nil {
					return err
				}
				if absent, err := r.policyAbsent(ctx, effect); err != nil || !absent {
					return errD3Unconfirmed
				}
			}
			next := *r.journal.record
			if saved.Removing {
				next.Mutations--
			}
			next.Policies = slices.DeleteFunc(slices.Clone(next.Policies), func(v d3PolicyRecord) bool {
				return v.Session == saved.Session && v.Network == saved.Network
			})
			if err := r.write(&next); err != nil {
				return err
			}
		}
	}
	for _, parent := range record.Keepers {
		executor, ok := r.runner.(*Executor)
		if !ok || !validObservationParent(parent) {
			return errD3Unconfirmed
		}
		if err := (systemdObservationKeeper{executor: executor}).recover(ctx, parent); err != nil {
			return err
		}
	}
	networks, err := r.list(ctx, record.Owner, "network", "ls", "--no-trunc")
	if err != nil {
		return err
	}
	for _, id := range networks {
		body, err := r.runner.Output(ctx, "docker", "network", "inspect", "--format", "{{json .}}", id)
		var network struct {
			ID     string
			Labels map[string]string
		}
		if err != nil || len(body) > 64<<10 || json.Unmarshal(body, &network) != nil || network.ID != id || network.Labels[d3OwnerLabel] != record.Owner {
			return errD3Unconfirmed
		}
		if _, err := r.runner.Output(ctx, "docker", "network", "rm", id); err != nil {
			return err
		}
	}
	networks, err = r.list(ctx, record.Owner, "network", "ls", "--no-trunc")
	if err != nil || len(networks) != 0 {
		return errD3Unconfirmed
	}
	return r.recoverVolumes(ctx, record.Owner)
}

func (r *d3Recovery) recoverVolumes(ctx context.Context, owner string) error {
	args := []string{"volume", "ls", "--filter", "label=" + d3OwnerLabel + "=" + owner, "--format", "{{.Name}}"}
	body, err := r.runner.Output(ctx, "docker", args...)
	if err != nil || len(body) > d3JournalLimit {
		return errD3Unconfirmed
	}
	for _, name := range strings.Fields(string(body)) {
		if name == "" || len(name) > 128 || strings.ContainsFunc(name, func(c rune) bool {
			return !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-", c)
		}) {
			return errD3Unconfirmed
		}
		body, err := r.runner.Output(ctx, "docker", "volume", "inspect", "--format", "{{json .}}", name)
		var volume struct {
			Name   string
			Labels map[string]string
		}
		if err != nil || len(body) > 8192 || json.Unmarshal(body, &volume) != nil || volume.Name != name || volume.Labels[d3OwnerLabel] != owner {
			return errD3Unconfirmed
		}
		if _, err := r.runner.Output(ctx, "docker", "volume", "rm", "--", name); err != nil {
			return err
		}
	}
	body, err = r.runner.Output(ctx, "docker", args...)
	if err != nil || strings.TrimSpace(string(body)) != "" {
		return errD3Unconfirmed
	}
	return nil
}

func (saved d3PolicyRecord) policy() (networkpolicy.ResolverPolicy, error) {
	session, err := domain.ParseSandboxSessionID(saved.Session)
	if err != nil || saved.Backend != policyIPTables && saved.Backend != policyNFTables {
		return networkpolicy.ResolverPolicy{}, errD3Unconfirmed
	}
	return networkpolicy.NewResolverPolicy(session, saved.Network, saved.Subnet, saved.Endpoints)
}

func (r *d3Recovery) beginPolicyMutation(policy policyRecord, create bool) error {
	return r.change(policy.uid, func(row *d3Record) error {
		p := policy.policy
		index := slices.IndexFunc(row.Policies, func(v d3PolicyRecord) bool {
			return v.Session == p.Session().String() && v.Network == p.NetworkID()
		})
		if (index >= 0) == create || row.Mutations >= 64 || create && r.observerDead {
			return errD3Unconfirmed
		}
		if create {
			row.Policies = append(row.Policies, d3PolicyRecord{
				Session: p.Session().String(), Network: p.NetworkID(),
				Subnet: p.Subnet(), Endpoints: p.Endpoints(), Backend: policy.backend,
			})
		} else {
			if row.Policies[index].Removing {
				return errD3Unconfirmed
			}
			row.Policies[index].Removing = true
		}
		row.Mutations++
		return nil
	})
}

func (r *d3Recovery) endPolicyMutation(policy policyRecord, create bool) error {
	return r.change(policy.uid, func(row *d3Record) error {
		if row.Mutations == 0 {
			return errD3Unconfirmed
		}
		row.Mutations--
		if !create {
			row.Policies = slices.DeleteFunc(row.Policies, func(v d3PolicyRecord) bool {
				return v.Session == policy.policy.Session().String() && v.Network == policy.policy.NetworkID()
			})
		}
		return nil
	})
}

func (r *d3Recovery) policyAbsent(ctx context.Context, policy policyRecord) (bool, error) {
	if policy.backend == policyIPTables {
		body, err := r.runner.Output(ctx, "iptables", "-S")
		if err != nil || len(body) > d3JournalLimit || len(body) == 0 {
			return false, errD3Unconfirmed
		}
		return !slices.Contains(strings.Fields(string(body)), policy.chain), nil
	}
	body, err := r.runner.Output(ctx, "nft", "list", "tables")
	if err != nil || len(body) > d3JournalLimit {
		return false, errD3Unconfirmed
	}
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != "table" {
			return false, errD3Unconfirmed
		}
		if fields[1] == "ip" && fields[2] == strings.ToLower(policy.chain) {
			return false, nil
		}
	}
	return true, nil
}

func (r *d3Recovery) removeRecordedIPTables(ctx context.Context, policy policyRecord) error {
	body, err := r.runner.Output(ctx, "iptables", "-S")
	if err != nil || len(body) > d3JournalLimit || len(body) == 0 {
		return errD3Unconfirmed
	}
	chain := policy.chain
	jump := "-A DOCKER-USER -s " + policy.policy.Subnet().String() + " -j " + chain
	owned := map[string]bool{"-A " + chain + " -j DROP": true}
	for _, endpoint := range policy.policy.Endpoints() {
		owned["-A "+chain+" -d "+endpoint.String()+"/32 -p tcp -m tcp --dport 443 -j ACCEPT"] = true
	}
	foundChain, foundJump := false, false
	for _, row := range strings.Split(string(body), "\n") {
		if !slices.Contains(strings.Fields(row), chain) {
			continue
		}
		switch {
		case row == "-N "+chain && !foundChain:
			foundChain = true
		case row == jump && !foundJump:
			foundJump = true
		case owned[row]:
			delete(owned, row)
		default:
			return errD3Unconfirmed
		}
	}
	if !foundChain {
		return errD3Unconfirmed
	}
	if foundJump {
		if _, err := r.runner.Output(ctx, "iptables", "-D", "DOCKER-USER", "-s", policy.policy.Subnet().String(), "-j", chain); err != nil {
			return err
		}
	}
	if _, err := r.runner.Output(ctx, "iptables", "-F", chain); err != nil {
		return err
	}
	_, err = r.runner.Output(ctx, "iptables", "-X", chain)
	return err
}
