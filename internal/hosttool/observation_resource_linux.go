//go:build linux

package hosttool

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"golang.org/x/sys/unix"
)

const observationResourceSocket = "/run/heliopause-network-policy/observation.sock"
const maxObservationResourceMessage = 2048

func observationResourceTimeout(operation string) time.Duration {
	switch operation {
	case "create", "close":
		return 20 * time.Second
	case "terminate":
		return 7 * time.Second
	default:
		return 5 * time.Second
	}
}

// ObservationResourceLease contains only controller-operational values. The
// helper chooses both cgroup identity and CPU affinity from protected inputs.
type ObservationResourceLease struct {
	CgroupParent string `json:"cgroup_parent"`
	CPU          int    `json:"cpu"`
	UsageUsec    uint64 `json:"usage_usec"`
}

type observationResourceRequest struct {
	Operation   string `json:"operation"`
	Transaction string `json:"transaction"`
	Profile     string `json:"profile,omitempty"`
	Container   string `json:"container,omitempty"`
	Role        string `json:"role,omitempty"`
}

type observationResourceResponse struct {
	OK    bool                     `json:"ok"`
	Lease ObservationResourceLease `json:"lease"`
}

type observationResourceSession struct {
	peer              uint32
	parent            string
	cpu               int
	ceiling           uint64
	usage             uint64
	path              string
	keeperScope       string
	containers        map[string]string
	scopes            map[string]string
	drainedScopes     map[string]struct{}
	drainedContainers map[string]string
	registrations     int
	maxLaunches       int
	failed            bool
	closed            bool
}

type observationResourceEngine struct {
	mu       sync.Mutex
	sessions map[string]*observationResourceSession
	executor *Executor
	keeper   observationKeeper
}

// CPU reads remain available while Docker is stopping a runtime. Holding the
// engine lock across docker rm would prevent the trusted controller from
// sampling during its reserved termination interval.
func (e *observationResourceEngine) read(ctx context.Context, peer uint32, request observationResourceRequest) (ObservationResourceLease, error) {
	if ctx == nil || request.Container != "" || request.Role != "" || request.Profile != "" {
		return ObservationResourceLease{}, errors.New("invalid observation accounting request")
	}
	parent, err := observationParent(request.Transaction)
	if err != nil {
		return ObservationResourceLease{}, err
	}
	e.mu.Lock()
	session := e.sessions[request.Transaction]
	if session == nil || session.closed || session.failed || session.peer != peer || session.parent != parent || session.path == "" {
		e.mu.Unlock()
		return ObservationResourceLease{}, errors.New("observation accounting transaction unavailable")
	}
	snapshot := *session
	snapshot.scopes = make(map[string]string, len(session.scopes))
	for id, scope := range session.scopes {
		snapshot.scopes[id] = scope
	}
	snapshot.drainedScopes = make(map[string]struct{}, len(session.drainedScopes))
	for scope := range session.drainedScopes {
		snapshot.drainedScopes[scope] = struct{}{}
	}
	e.mu.Unlock()
	if e.keeper == nil || e.keeper.Verify(ctx, snapshot.parent, snapshot.keeperScope) != nil {
		e.fail(request.Transaction)
		return ObservationResourceLease{}, errors.New("observation accounting keeper unavailable")
	}
	if err := verifyObservationMembers(&snapshot); err != nil {
		e.fail(request.Transaction)
		return ObservationResourceLease{}, err
	}
	usage, err := readObservationCPU(snapshot.path)
	if err != nil {
		e.fail(request.Transaction)
		return ObservationResourceLease{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	current := e.sessions[request.Transaction]
	if current != session || session.closed || session.failed || usage < session.usage || usage > session.ceiling {
		if current == session {
			session.failed = true
		}
		return ObservationResourceLease{}, errors.New("observation CPU counter changed or exceeded authorization")
	}
	session.usage = usage
	return ObservationResourceLease{CgroupParent: session.parent, CPU: session.cpu, UsageUsec: usage}, nil
}

func (e *observationResourceEngine) fail(transaction string) {
	e.mu.Lock()
	if session := e.sessions[transaction]; session != nil {
		session.failed = true
	}
	e.mu.Unlock()
}

func (e *observationResourceEngine) terminate(ctx context.Context, peer uint32, request observationResourceRequest) (ObservationResourceLease, error) {
	if ctx == nil || request.Role != "" || request.Profile != "" || !validDockerContainerID(request.Container) {
		return ObservationResourceLease{}, errors.New("invalid observation termination request")
	}
	parent, err := observationParent(request.Transaction)
	if err != nil {
		return ObservationResourceLease{}, err
	}
	e.mu.Lock()
	session := e.sessions[request.Transaction]
	if session == nil || session.closed || session.peer != peer || session.parent != parent {
		e.mu.Unlock()
		return ObservationResourceLease{}, errors.New("observation runtime is not registered")
	}
	scope := session.scopes[request.Container]
	alreadyDrained := false
	if scope == "" {
		scope = session.drainedContainers[request.Container]
		alreadyDrained = scope != ""
	}
	if scope == "" {
		e.mu.Unlock()
		return ObservationResourceLease{}, errors.New("observation runtime is not registered")
	}
	e.mu.Unlock()
	// Removal is idempotent only when the independently queried daemon and old
	// cgroup scope both prove exact absence. No client-supplied PID is used.
	if !alreadyDrained {
		if err := killObservationScope(scope); err != nil {
			e.fail(request.Transaction)
			return ObservationResourceLease{}, err
		}
	}
	_ = e.executor.RunDiscard(ctx, "docker", "rm", "--force", request.Container)
	if err := e.awaitRuntimeDrained(ctx, request.Container, scope); err != nil {
		e.fail(request.Transaction)
		return ObservationResourceLease{}, errors.New("observation runtime drainage is unproven")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sessions[request.Transaction] != session ||
		(!alreadyDrained && session.scopes[request.Container] != scope) ||
		(alreadyDrained && session.drainedContainers[request.Container] != scope) {
		session.failed = true
		return ObservationResourceLease{}, errors.New("observation runtime identity changed during termination")
	}
	if !alreadyDrained {
		session.drainedScopes[scope] = struct{}{}
		session.drainedContainers[request.Container] = scope
		delete(session.containers, request.Container)
		delete(session.scopes, request.Container)
	}
	if e.keeper == nil || e.keeper.Verify(ctx, session.parent, session.keeperScope) != nil || verifyObservationMembers(session) != nil {
		session.failed = true
		return ObservationResourceLease{}, errors.New("observation transaction cgroup membership changed")
	}
	usage, err := readObservationCPU(session.path)
	if err != nil || usage < session.usage || usage > session.ceiling {
		session.failed = true
		return ObservationResourceLease{}, errors.New("observation CPU accounting after termination is unavailable")
	}
	session.usage = usage
	return ObservationResourceLease{CgroupParent: session.parent, CPU: session.cpu, UsageUsec: usage}, nil
}

func (e *observationResourceEngine) awaitRuntimeDrained(ctx context.Context, container, scope string) error {
	if e == nil || e.executor == nil || ctx == nil || !validDockerContainerID(container) {
		return errors.New("observation runtime drainage identity is invalid")
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		remaining, err := e.executor.Output(ctx, "docker", "ps", "-a", "--no-trunc", "--filter", "id="+container, "--format", "{{.ID}}")
		if err == nil && len(remaining) <= 1024 && strings.TrimSpace(string(remaining)) == "" && verifyDrainedScope(scope) == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("observation runtime drainage deadline expired")
		case <-ticker.C:
		}
	}
}

func observationParent(transaction string) (string, error) {
	if _, err := domain.ParseSandboxSessionID(transaction); err != nil {
		return "", errors.New("invalid observation transaction identity")
	}
	digest := sha256.Sum256([]byte(transaction))
	return "haaobs" + hex.EncodeToString(digest[:12]) + ".slice", nil
}

func observationPolicy(name string) (artifactpypi.ResourcePolicy, error) {
	if name == "pypi" {
		return artifactpypi.PublicPyPIProfile().ResourcePolicy(), nil
	}
	if !strings.HasPrefix(name, "pytorch:") {
		return artifactpypi.ResourcePolicy{}, errors.New("unsupported observation resource policy")
	}
	profile, ok := artifactpypi.PyTorchProfile(strings.TrimPrefix(name, "pytorch:"))
	if !ok || profile.Name() != name {
		return artifactpypi.ResourcePolicy{}, errors.New("unsupported observation resource policy")
	}
	return profile.ResourcePolicy(), nil
}

func firstAvailableCPU() (int, error) {
	var set unix.CPUSet
	if err := unix.SchedGetaffinity(0, &set); err != nil {
		return 0, err
	}
	for cpu := 0; cpu < 1024; cpu++ {
		if set.IsSet(cpu) {
			return cpu, nil
		}
	}
	return 0, errors.New("no available observation CPU")
}

func (e *observationResourceEngine) apply(ctx context.Context, peer uint32, request observationResourceRequest) (lease ObservationResourceLease, resultErr error) {
	if e == nil || e.executor == nil || ctx == nil {
		return ObservationResourceLease{}, errors.New("observation resource helper is unavailable")
	}
	parent, err := observationParent(request.Transaction)
	if err != nil {
		return ObservationResourceLease{}, err
	}
	if request.Operation == "read" {
		return e.read(ctx, peer, request)
	}
	if request.Operation == "terminate" {
		return e.terminate(ctx, peer, request)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	session := e.sessions[request.Transaction]
	if session != nil {
		defer func() {
			if resultErr != nil {
				session.failed = true
			}
		}()
	}
	if request.Operation == "create" {
		if session != nil || request.Container != "" || request.Role != "" {
			return ObservationResourceLease{}, errors.New("observation resource transaction already exists")
		}
		policy, err := observationPolicy(request.Profile)
		if err != nil {
			return ObservationResourceLease{}, err
		}
		cpu, err := firstAvailableCPU()
		if err != nil {
			return ObservationResourceLease{}, err
		}
		if e.sessions == nil {
			e.sessions = make(map[string]*observationResourceSession)
		}
		if len(e.sessions) >= 8 {
			return ObservationResourceLease{}, errors.New("observation resource service is at capacity")
		}
		if e.keeper == nil {
			return ObservationResourceLease{}, errors.New("observation transaction keeper unavailable")
		}
		path, keeperScope, baseline, err := e.keeper.Start(ctx, parent)
		if err != nil {
			return ObservationResourceLease{}, err
		}
		session = &observationResourceSession{peer: peer, parent: parent, cpu: cpu, path: path, keeperScope: keeperScope, usage: baseline,
			ceiling:     uint64(policy.RuntimeCPUSecs()) * 1_000_000,
			maxLaunches: 2*policy.MaxObservationImportsPerArtifact() + 4,
			containers:  make(map[string]string), scopes: make(map[string]string),
			drainedScopes: make(map[string]struct{}), drainedContainers: make(map[string]string)}
		e.sessions[request.Transaction] = session
		return ObservationResourceLease{CgroupParent: parent, CPU: cpu, UsageUsec: baseline}, nil
	}
	if session == nil || session.closed || session.peer != peer || session.parent != parent || request.Profile != "" {
		return ObservationResourceLease{}, errors.New("observation resource transaction identity does not match")
	}
	if session.failed && request.Operation != "terminate" && request.Operation != "read" && request.Operation != "close" {
		return ObservationResourceLease{}, errors.New("observation resource transaction has failed")
	}
	switch request.Operation {
	case "register":
		if request.Role != "preparation" && request.Role != "anchor" && request.Role != "observation" {
			return ObservationResourceLease{}, errors.New("invalid observation runtime role")
		}
		if _, err := domain.ParseSandboxSessionID(request.Transaction); err != nil || !validDockerContainerID(request.Container) ||
			session.containers[request.Container] != "" || session.drainedContainers[request.Container] != "" ||
			session.registrations >= session.maxLaunches {
			return ObservationResourceLease{}, errors.New("invalid or duplicate observation runtime")
		}
		path, scope, err := e.verifyDockerMember(ctx, session, request.Container, request.Transaction, request.Role)
		if err != nil {
			return ObservationResourceLease{}, err
		}
		if session.path != path {
			return ObservationResourceLease{}, errors.New("observation cgroup parent changed")
		}
		session.containers[request.Container] = request.Role
		session.scopes[request.Container] = scope
		session.registrations++
	case "close":
		if request.Container != "" || request.Role != "" || len(session.containers) != 0 {
			return ObservationResourceLease{}, errors.New("observation transaction still has live runtimes")
		}
		// Cleanup must remain possible after CPU exhaustion. Otherwise the
		// failed transaction would strand its privileged keeper and slice.
		if e.keeper == nil || e.keeper.Verify(ctx, session.parent, session.keeperScope) != nil || verifyObservationMembers(session) != nil {
			return ObservationResourceLease{}, errors.New("observation transaction cleanup membership unproven")
		}
		usage, err := e.keeper.Close(ctx, session.parent, session.keeperScope)
		if err != nil {
			return ObservationResourceLease{}, errors.New("observation keeper cleanup failed")
		}
		failed := session.failed || usage < session.usage || usage > session.ceiling
		session.closed = true
		delete(e.sessions, request.Transaction)
		if failed {
			return ObservationResourceLease{}, errors.New("failed observation resource transaction cannot qualify")
		}
		return ObservationResourceLease{CgroupParent: session.parent, CPU: session.cpu, UsageUsec: usage}, nil
	default:
		return ObservationResourceLease{}, errors.New("unsupported observation resource operation")
	}
	if e.keeper == nil || e.keeper.Verify(ctx, session.parent, session.keeperScope) != nil {
		return ObservationResourceLease{}, errors.New("observation transaction keeper changed")
	}
	if err := verifyObservationMembers(session); err != nil {
		return ObservationResourceLease{}, err
	}
	usage, err := readObservationCPU(session.path)
	if err != nil || usage < session.usage || usage > session.ceiling {
		return ObservationResourceLease{}, errors.New("observation CPU accounting is missing, decreasing or exhausted")
	}
	session.usage = usage
	return ObservationResourceLease{CgroupParent: session.parent, CPU: session.cpu, UsageUsec: usage}, nil
}

func validDockerContainerID(id string) bool {
	if len(id) != 64 {
		return false
	}
	for _, character := range id {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func (e *observationResourceEngine) verifyDockerMember(ctx context.Context, session *observationResourceSession, container, transaction, role string) (string, string, error) {
	const format = `{{json .Id}}|{{json .HostConfig.CgroupParent}}|{{json .HostConfig.CpusetCpus}}|{{json .State.Pid}}|{{json .HostConfig.Runtime}}|{{json .Config.Labels}}`
	body, err := e.executor.Output(ctx, "docker", "inspect", "--format", format, container)
	if err != nil || len(body) > 4096 {
		return "", "", errors.New("observation runtime inspection unavailable")
	}
	parts := strings.Split(strings.TrimSpace(string(body)), "|")
	if len(parts) != 6 {
		return "", "", errors.New("observation runtime inspection malformed")
	}
	var id, parent, cpuSet, runtimeName string
	var pid int
	var labels map[string]string
	if json.Unmarshal([]byte(parts[0]), &id) != nil || json.Unmarshal([]byte(parts[1]), &parent) != nil ||
		json.Unmarshal([]byte(parts[2]), &cpuSet) != nil || json.Unmarshal([]byte(parts[3]), &pid) != nil ||
		json.Unmarshal([]byte(parts[4]), &runtimeName) != nil || json.Unmarshal([]byte(parts[5]), &labels) != nil ||
		id != container || parent != session.parent || cpuSet != strconv.Itoa(session.cpu) || pid <= 1 || runtimeName != "runsc-trace" ||
		labels["io.heliopause.observation-transaction"] != transaction ||
		labels["io.heliopause.observation-phase"] != role || len(labels["io.heliopause.closure-manifest-sha256"]) != 64 {
		return "", "", errors.New("observation runtime cgroup admission mismatch")
	}
	if decoded, err := hex.DecodeString(labels["io.heliopause.closure-manifest-sha256"]); err != nil || len(decoded) != sha256.Size {
		return "", "", errors.New("observation runtime closure label is invalid")
	}
	proc, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cgroup"))
	if err != nil || len(proc) > 4096 {
		return "", "", errors.New("observation runtime cgroup membership unavailable")
	}
	var relative string
	for _, line := range strings.Split(strings.TrimSpace(string(proc)), "\n") {
		if strings.HasPrefix(line, "0::/") {
			relative = strings.TrimPrefix(line, "0::/")
		}
	}
	components := strings.Split(relative, "/")
	for index, component := range components {
		if component == session.parent {
			path := filepath.Join(append([]string{"/sys/fs/cgroup"}, components[:index+1]...)...)
			info, statErr := os.Lstat(path)
			if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return "", "", errors.New("observation parent cgroup is unavailable")
			}
			if index+1 >= len(components) || components[index+1] != "docker-"+container+".scope" || index+2 != len(components) {
				return "", "", errors.New("observation runtime scope is not bound to its container")
			}
			return path, filepath.Join(path, components[index+1]), nil
		}
	}
	return "", "", errors.New("observation runtime is outside transaction cgroup")
}

func verifyDrainedScope(scope string) error {
	if scope == "" || !strings.HasPrefix(scope, "/sys/fs/cgroup/") {
		return errors.New("observation runtime scope identity unavailable")
	}
	info, err := os.Lstat(scope)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("observation runtime scope changed")
	}
	body, err := os.ReadFile(filepath.Join(scope, "cgroup.events"))
	if err != nil || len(body) > 1024 || !strings.Contains("\n"+string(body), "\npopulated 0\n") {
		return errors.New("observation runtime descendants have not drained")
	}
	return nil
}

// Kill the exact Docker scope before waiting for Docker teardown. This bounds
// artifact CPU even if the daemon takes time to remove a stopped container.
// The scope was obtained from Docker's verified full container identity, not
// from a caller path or PID.
func killObservationScope(scope string) error {
	if scope == "" || !strings.HasPrefix(scope, "/sys/fs/cgroup/") || !strings.HasSuffix(scope, ".scope") {
		return errors.New("observation runtime kill scope unavailable")
	}
	info, err := os.Lstat(scope)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("observation runtime kill scope changed")
	}
	if verifyDrainedScope(scope) == nil {
		return nil
	}
	file, err := os.OpenFile(filepath.Join(scope, "cgroup.kill"), os.O_WRONLY, 0)
	if err != nil {
		return errors.New("observation runtime cgroup kill unavailable")
	}
	_, writeErr := file.WriteString("1")
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return errors.New("observation runtime cgroup kill failed")
	}
	return nil
}

func verifyObservationMembers(session *observationResourceSession) error {
	if session == nil || session.path == "" || session.keeperScope == "" {
		return errors.New("observation parent cgroup is not established")
	}
	entries, err := os.ReadDir(session.path)
	if err != nil || len(entries) > 128 {
		return errors.New("observation parent cgroup is unreadable")
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(session.path, entry.Name())
		known := path == session.keeperScope
		for _, scope := range session.scopes {
			if scope == path {
				known = true
				break
			}
		}
		if !known {
			if _, drained := session.drainedScopes[path]; drained {
				if verifyDrainedScope(path) != nil {
					return errors.New("drained observation scope became populated")
				}
				known = true
			}
		}
		if !known {
			return errors.New("unexpected transaction cgroup member")
		}
	}
	body, err := os.ReadFile(filepath.Join(session.path, "cgroup.procs"))
	if err != nil || len(body) > 1024 || strings.TrimSpace(string(body)) != "" {
		return errors.New("unexpected process in observation parent cgroup")
	}
	return nil
}

func readObservationCPU(parent string) (uint64, error) {
	if parent == "" || !strings.HasPrefix(parent, "/sys/fs/cgroup/") {
		return 0, errors.New("observation cgroup parent is unavailable")
	}
	body, err := os.ReadFile(filepath.Join(parent, "cpu.stat"))
	if err != nil || len(body) == 0 || len(body) > 4096 {
		return 0, errors.New("observation CPU counter is unreadable")
	}
	found := false
	var value uint64
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "usage_usec" {
			if found {
				return 0, errors.New("duplicate observation CPU counter")
			}
			value, err = strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, errors.New("invalid observation CPU counter")
			}
			found = true
		}
	}
	if !found {
		return 0, errors.New("missing observation CPU counter")
	}
	return value, nil
}

// ServeObservationResources runs within the existing authenticated helper
// process. Its socket is independent of resolver-network protocol state.
func serveObservationResources(ctx context.Context, config policyServiceConfig, executor *Executor) error {
	if os.Geteuid() != 0 || executor == nil || filepath.Dir(observationResourceSocket) != filepath.Dir(config.SocketPath) {
		return errors.New("observation resource helper identity is unavailable")
	}
	if _, err := os.Lstat(observationResourceSocket); !errors.Is(err, os.ErrNotExist) {
		return errors.New("observation resource socket already exists")
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: observationResourceSocket, Net: "unix"})
	if err != nil {
		return err
	}
	defer listener.Close()
	socketID, err := os.Lstat(observationResourceSocket)
	if err != nil || socketID.Mode()&os.ModeSocket == 0 {
		return errors.New("observation resource socket identity unavailable")
	}
	if os.Chmod(observationResourceSocket, 0o660) != nil || os.Chown(observationResourceSocket, 0, int(config.ClientGID)) != nil {
		return errors.New("observation resource socket protection failed")
	}
	defer func() {
		current, statErr := os.Lstat(observationResourceSocket)
		if statErr == nil && os.SameFile(socketID, current) {
			_ = os.Remove(observationResourceSocket)
		}
	}()
	authorizer, err := NewPolicyPeerAuthorizer(config.ClientPath, config.ClientUID)
	if err != nil {
		return err
	}
	engine := &observationResourceEngine{executor: executor, keeper: systemdObservationKeeper{executor: executor}, sessions: make(map[string]*observationResourceSession)}
	go func() { <-ctx.Done(); _ = listener.Close() }()
	for {
		connection, err := listener.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return errors.New("accept observation resource client")
		}
		go func() {
			defer connection.Close()
			_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
			peer, err := authorizer.AuthorizePeer(connection)
			if err != nil {
				return
			}
			body, err := bufio.NewReaderSize(connection, maxObservationResourceMessage+1).ReadBytes('\n')
			if err != nil || len(body) < 2 || len(body) > maxObservationResourceMessage {
				return
			}
			decoder := json.NewDecoder(bytes.NewReader(body[:len(body)-1]))
			decoder.DisallowUnknownFields()
			var request observationResourceRequest
			if decoder.Decode(&request) != nil || decoder.Decode(&struct{}{}) != io.EOF {
				return
			}
			requestLimit := observationResourceTimeout(request.Operation)
			_ = connection.SetDeadline(time.Now().Add(requestLimit))
			requestContext, cancel := context.WithTimeout(ctx, requestLimit)
			lease, err := engine.apply(requestContext, peer, request)
			cancel()
			if err == nil {
				_ = json.NewEncoder(connection).Encode(observationResourceResponse{OK: true, Lease: lease})
			}
		}()
	}
}

type ObservationResourceClient struct {
	endpoint string
	group    uint32
}

func NewSystemObservationResourceClient() (*ObservationResourceClient, error) {
	config, err := loadPolicyServiceConfig()
	if err != nil || filepath.Dir(config.SocketPath) != filepath.Dir(observationResourceSocket) ||
		verifyPolicySocket(observationResourceSocket, config.ClientGID) != nil {
		return nil, errors.New("observation resource service unavailable")
	}
	return &ObservationResourceClient{endpoint: observationResourceSocket, group: config.ClientGID}, nil
}

func (c *ObservationResourceClient) call(ctx context.Context, request observationResourceRequest) (ObservationResourceLease, error) {
	if c == nil || ctx == nil || verifyPolicySocket(c.endpoint, c.group) != nil {
		return ObservationResourceLease{}, errors.New("observation resource service identity changed")
	}
	connection, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", c.endpoint)
	if err != nil {
		return ObservationResourceLease{}, errors.New("observation resource service connection failed")
	}
	defer connection.Close()
	deadline := time.Now().Add(observationResourceTimeout(request.Operation))
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	_ = connection.SetDeadline(deadline)
	if json.NewEncoder(connection).Encode(request) != nil {
		return ObservationResourceLease{}, errors.New("observation resource request failed")
	}
	decoder := json.NewDecoder(io.LimitReader(connection, maxObservationResourceMessage+1))
	decoder.DisallowUnknownFields()
	var response observationResourceResponse
	if decoder.Decode(&response) != nil || decoder.Decode(&struct{}{}) != io.EOF || !response.OK ||
		verifyPolicySocket(c.endpoint, c.group) != nil {
		return ObservationResourceLease{}, errors.New("observation resource response is untrusted")
	}
	return response.Lease, nil
}

func (c *ObservationResourceClient) Create(ctx context.Context, transaction, profile string) (ObservationResourceLease, error) {
	return c.call(ctx, observationResourceRequest{Operation: "create", Transaction: transaction, Profile: profile})
}
func (c *ObservationResourceClient) Register(ctx context.Context, transaction, container, role string) (ObservationResourceLease, error) {
	return c.call(ctx, observationResourceRequest{Operation: "register", Transaction: transaction, Container: container, Role: role})
}
func (c *ObservationResourceClient) Read(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	return c.call(ctx, observationResourceRequest{Operation: "read", Transaction: transaction})
}
func (c *ObservationResourceClient) Terminate(ctx context.Context, transaction, container string) (ObservationResourceLease, error) {
	return c.call(ctx, observationResourceRequest{Operation: "terminate", Transaction: transaction, Container: container})
}
func (c *ObservationResourceClient) Close(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	return c.call(ctx, observationResourceRequest{Operation: "close", Transaction: transaction})
}
