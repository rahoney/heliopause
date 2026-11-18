package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type environmentModelFile struct {
	body []byte
	mode int64
	kind byte
	link string
}
type environmentModelContainer struct {
	volume            string
	mounts            map[string]string
	mountReadOnly     map[string]bool
	readOnly, started bool
	trace             *cleanupGatedTraceReader
	done              chan struct{}
}

// Model only the external Docker/resource transports. It never runs package
// code and is not native isolation/observer qualification.
type environmentBackendModel struct {
	mu                                                                    sync.Mutex
	volumes                                                               map[string]dockerVolumeInspection
	files                                                                 map[string]map[string]environmentModelFile
	containers                                                            map[string]*environmentModelContainer
	next, creates, registers, commands, introductions, snapshots, removed int
	parent                                                                string
	owner                                                                 string
	usage, events                                                         uint64
	budget                                                                *observationTraceLedger
	failure                                                               string
	commandCause, closeCause                                              error
	payload                                                               []byte
	traceRecords                                                          []TraceRecord
	traceError                                                            error
	order                                                                 []string
	traces                                                                []*cleanupGatedTraceReader
	cancel                                                                context.CancelFunc
	terminations                                                          int
}

func newEnvironmentBackendState() *environmentBackendModel {
	return &environmentBackendModel{volumes: make(map[string]dockerVolumeInspection), files: make(map[string]map[string]environmentModelFile), containers: make(map[string]*environmentModelContainer), parent: "haaobs-model.slice", payload: []byte("fixed model input")}
}

func newEnvironmentBackendModel(t *testing.T) (*Backend, *environmentBackendModel, domain.SandboxRequest) {
	t.Helper()
	f := newEnvironmentBackendState()
	sum := sha256.Sum256(f.payload)
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.0.0", "tarball")
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", uint64(len(f.payload)))
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewBackend(f, f, f, availableProbe)
	if err != nil {
		t.Fatal(err)
	}
	backend.resources = f
	backend.prepareResponder = f.prepareResponder
	return backend, f, request
}

func (f *environmentBackendModel) prepareResponder(_ context.Context, _ CommandRunner, installation string) (*responderSupply, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if installation != runtimeidentity.NativeInstallRoot || f.owner == "" || f.failure == "responder-supply" {
		return nil, errResponderSupply
	}
	supply := &responderSupply{name: "haa-responder-" + f.owner, owner: f.owner, digest: strings.Repeat("b", 64), createdAt: "model-created"}
	supply.mountpoint = "/model/" + supply.name
	f.volumes[supply.name] = dockerVolumeInspection{Name: supply.name, Driver: "local", Scope: "local", Mountpoint: supply.mountpoint, CreatedAt: supply.createdAt, Labels: map[string]string{responderSupplyLabel: supply.owner, closureManifestLabel: supply.digest}}
	return supply, nil
}

func (f *environmentBackendModel) StartContainerInputOutput(ctx context.Context, input io.Reader, output io.Writer, id string) error {
	f.mu.Lock()
	c := f.containers[id]
	if c == nil || !c.mountReadOnly[responderGuestRoot] {
		f.mu.Unlock()
		return errResponderSupply
	}
	c.started = true
	model := responderControllerModel{terminate: c.done}
	failure, cancel := f.failure, f.cancel
	f.mu.Unlock()
	switch failure {
	case "responder-start":
		_, err := io.WriteString(output, "INVALID\n")
		return err
	case "responder-stop":
		if _, err := io.WriteString(output, "READY\n"); err != nil {
			return err
		}
		if _, err := io.ReadAll(input); err != nil {
			return err
		}
		_, err := io.WriteString(output, "INVALID\n")
		return err
	case "responder-cancel":
		cancel()
		<-ctx.Done()
		return ctx.Err()
	}
	return model.StartContainerInputOutput(ctx, input, output, id)
}

func modelArgument(args []string, name string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}
func (f *environmentBackendModel) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(args) == 0 {
		return nil, errors.New("empty model command")
	}
	switch args[0] {
	case "volume":
		name := args[len(args)-1]
		switch args[1] {
		case "ls":
			return nil, nil
		case "create":
			options, labels := map[string]string{}, map[string]string{}
			for i := 0; i+1 < len(args); i++ {
				if args[i] == "--opt" || args[i] == "--label" {
					key, value, _ := strings.Cut(args[i+1], "=")
					if args[i] == "--opt" {
						options[key] = value
					} else {
						labels[key] = value
					}
				}
			}
			f.volumes[name] = dockerVolumeInspection{Name: name, Driver: "local", Scope: "local", Mountpoint: "/model/" + name, CreatedAt: "2026-11-02T00:00:00Z", Options: options, Labels: labels}
			f.files[name] = make(map[string]environmentModelFile)
			return []byte(name), nil
		case "inspect":
			v, ok := f.volumes[name]
			if !ok {
				return nil, errors.New("unknown model volume")
			}
			return json.Marshal(v)
		case "rm":
			delete(f.volumes, name)
			delete(f.files, name)
			f.removed++
			return nil, nil
		}
	case "create":
		f.next++
		id := fmt.Sprintf("%064x", f.next)
		c := &environmentModelContainer{mounts: make(map[string]string), mountReadOnly: make(map[string]bool)}
		for index, arg := range args {
			if arg != "--mount" || index+1 >= len(args) {
				continue
			}
			fields := map[string]string{}
			for _, part := range strings.Split(args[index+1], ",") {
				key, value, _ := strings.Cut(part, "=")
				fields[key] = value
			}
			c.mounts[fields["target"]] = fields["source"]
			_, c.mountReadOnly[fields["target"]] = fields["readonly"]
			if fields["target"] == "/tmp" {
				c.volume = fields["source"]
				c.readOnly = strings.Contains(args[index+1], ",readonly")
			}
		}
		f.containers[id] = c
		return []byte(id), nil
	case "inspect":
		id := args[len(args)-1]
		c := f.containers[id]
		if c == nil {
			return nil, errors.New("unknown model container")
		}
		if modelArgument(args, "--format") == environmentResolverInspectTemplate {
			dns := []string{"127.0.0.1"}
			if f.failure == "resolver" {
				dns = []string{"8.8.8.8"}
			}
			return json.Marshal(map[string]any{"ID": id, "DNS": dns, "Search": []string{"."}, "Options": []string{"ndots:0", "timeout:1", "attempts:1"}})
		}
		if modelArgument(args, "--format") == containerMountInspectTemplate {
			view := dockerContainerMountInspection{ID: id}
			for target, name := range c.mounts {
				volume := f.volumes[name]
				readOnly := c.mountReadOnly[target]
				configured := dockerHostMountInspection{Type: "volume", Source: name, Target: target, ReadOnly: readOnly}
				configured.VolumeOptions.NoCopy = true
				view.Mounts = append(view.Mounts, dockerMountInspection{Type: "volume", Name: name, Source: volume.Mountpoint, Destination: target, Driver: "local", RW: !readOnly})
				view.HostMounts = append(view.HostMounts, configured)
			}
			return json.Marshal(view)
		}
		return []byte(id + "|true|2"), nil
	case "exec":
		return nil, nil
	case "ps":
		filter := modelArgument(args, "--filter")
		var ids []string
		for id, c := range f.containers {
			matches := filter == "id="+id
			for _, volume := range c.mounts {
				matches = matches || filter == "volume="+volume
			}
			if matches {
				ids = append(ids, id)
			}
		}
		return []byte(strings.Join(ids, "\n")), nil
	case "rm":
		f.remove(args[len(args)-1])
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected model command %q", args[0])
}

func (f *environmentBackendModel) remove(id string) {
	if c := f.containers[id]; c != nil {
		if c.trace != nil {
			select {
			case <-c.trace.cleanup:
			default:
				close(c.done)
			}
		}
		delete(f.containers, id)
	}
}
func (f *environmentBackendModel) RunDiscard(ctx context.Context, _ string, args ...string) error {
	f.mu.Lock()
	if args[0] == "rm" {
		f.remove(args[len(args)-1])
		f.mu.Unlock()
		return nil
	}
	if args[0] == "start" {
		f.containers[args[1]].started = true
		f.mu.Unlock()
		return nil
	}
	if args[0] == "exec" && args[len(args)-1] == npmLifecycleCommand {
		f.commands++
		f.events += 2
		if f.failure == "timeout" {
			f.mu.Unlock()
			<-ctx.Done()
			return ctx.Err()
		}
		if f.commandCause != nil {
			cause := f.commandCause
			f.mu.Unlock()
			return cause
		}
		c := f.containers[args[3]]
		if c == nil {
			f.mu.Unlock()
			return errors.New("model exec identity missing")
		}
		f.files[c.volume]["installed.txt"] = environmentModelFile{body: []byte("done"), mode: 0600, kind: tar.TypeReg}
		if f.failure == "input-mutated" {
			f.files[c.volume]["artifact.tgz"] = environmentModelFile{body: []byte("changed"), mode: 0600, kind: tar.TypeReg}
		}
		f.mu.Unlock()
		return nil
	}
	f.mu.Unlock()
	return errors.New("unexpected model discard")
}
func (f *environmentBackendModel) RunInput(_ context.Context, input io.Reader, _ string, args ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, destination, _ := strings.Cut(args[len(args)-1], ":")
	c := f.containers[id]

	if c == nil {
		return errors.New("unknown model writer")
	}
	volume := c.mounts[destination]
	if volume == "" {
		return errors.New("unknown model destination")
	}
	archive := tar.NewReader(input)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		body, err := io.ReadAll(archive)
		if err != nil {
			return err
		}
		if f.failure == "placement" && header.Name == ".haa-honeytoken" {
			continue
		}
		name := strings.TrimSuffix(header.Name, "/")
		file := environmentModelFile{body: body, mode: header.Mode, kind: header.Typeflag, link: header.Linkname}
		if header.Typeflag == tar.TypeLink {
			file = f.files[volume][header.Linkname]
			if file.kind != tar.TypeReg {
				return errors.New("unresolved model hardlink")
			}
		}
		f.files[volume][name] = file
	}
}
func (f *environmentBackendModel) RunOutput(_ context.Context, output io.Writer, _ string, args ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, _, _ := strings.Cut(args[1], ":")
	c := f.containers[id]
	if c != nil && args[1] == id+":/etc/resolv.conf" {
		const content = "nameserver 127.0.0.1\noptions ndots:0 timeout:1 attempts:1\n"
		archive := tar.NewWriter(output)
		if err := archive.WriteHeader(&tar.Header{Name: "resolv.conf", Mode: 0644, Typeflag: tar.TypeReg, Size: int64(len(content))}); err != nil {
			return err
		}
		if _, err := io.WriteString(archive, content); err != nil {
			return err
		}
		return archive.Close()
	}
	if c == nil || !c.readOnly {
		return errors.New("snapshot reader must be readonly")
	}
	if f.snapshots%2 == 1 {
		for _, other := range f.containers {
			if other.volume == c.volume && !other.readOnly {
				return errors.New("final snapshot preceded writer termination")
			}
		}
	}
	f.snapshots++
	archive := tar.NewWriter(output)
	if err := archive.WriteHeader(&tar.Header{Name: "./", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
		return err
	}
	files := make(map[string]environmentModelFile)
	for name, file := range f.files[c.volume] {
		files[name] = file
	}
	for destination, volume := range c.mounts {
		if destination == "/tmp" || !strings.HasPrefix(destination, "/tmp/") {
			continue
		}
		prefix := strings.TrimPrefix(destination, "/tmp/")
		files[prefix] = environmentModelFile{kind: tar.TypeDir, mode: 0700}
		for name, file := range f.files[volume] {
			files[prefix+"/"+name] = file
		}
	}
	for name, file := range files {
		if err := archive.WriteHeader(&tar.Header{Name: name, Typeflag: file.kind, Mode: file.mode, Size: int64(len(file.body)), Linkname: file.link, Uid: 1000, Gid: 1000}); err != nil {
			return err
		}
		if _, err := archive.Write(file.body); err != nil {
			return err
		}
	}
	return archive.Close()
}
func (f *environmentBackendModel) Introduce(_ context.Context, id string, _ domain.AcquiredArtifact) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.introductions++
	f.order = append(f.order, "artifact-introduced")
	f.files[f.containers[id].volume]["artifact.tgz"] = environmentModelFile{body: bytes.Clone(f.payload), mode: 0600, kind: tar.TypeReg}
	return nil
}
func (f *environmentBackendModel) Start(context.Context, string) (TraceReader, error) {
	return nil, errors.New("legacy observer cannot authorize environment experiments")
}
func (f *environmentBackendModel) StartEnvironmentProfile(_ context.Context, id string, _ bool, budget *observationTraceLedger, _ ...*responderSupply) (TraceReader, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failure == "observer" {
		return nil, errors.New("model observer failed")
	}
	if f.budget != nil && f.budget != budget {
		return nil, errors.New("trace budget replenished")
	}
	f.budget = budget
	done := make(chan struct{})
	reader := &cleanupGatedTraceReader{cleanup: done, records: append([]TraceRecord(nil), f.traceRecords...), err: f.traceError}
	f.containers[id].trace = reader
	f.containers[id].done = done
	f.traces = append(f.traces, reader)
	return reader, nil
}
func (f *environmentBackendModel) AwaitMountAnchors(context.Context, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.order = append(f.order, "mount-anchors-ready")
	if f.failure == "topology" {
		return errors.New("model topology mismatch")
	}
	return nil
}
func (f *environmentBackendModel) observationUsage(*observationTraceLedger) (uint64, uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.events, f.events * 64, nil
}
func (f *environmentBackendModel) lease() ObservationResourceLease {
	return ObservationResourceLease{CgroupParent: f.parent, CPU: 0, UsageUsec: f.usage}
}
func (f *environmentBackendModel) Create(_ context.Context, owner, profile string) (ObservationResourceLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates++
	if f.creates != 1 || profile != "npm-lifecycle" {
		return ObservationResourceLease{}, errors.New("resource authorization replenished")
	}
	f.owner = owner
	return f.lease(), nil
}
func (f *environmentBackendModel) Register(_ context.Context, owner, id, phase string) (ObservationResourceLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registers++
	if f.failure == "registration" {
		return ObservationResourceLease{}, errors.New("model registration failed")
	}
	if owner != f.owner || f.registers > 4 || f.containers[id] == nil || (phase != "anchor" && phase != "observation") {
		return ObservationResourceLease{}, errors.New("wrong model resource admission")
	}
	return f.lease(), nil
}
func (f *environmentBackendModel) Read(context.Context, string) (ObservationResourceLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.usage += 100
	return f.lease(), nil
}
func (f *environmentBackendModel) Terminate(_ context.Context, _ string, id string) (ObservationResourceLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.terminations++
	f.remove(id)
	f.usage += 100
	return f.lease(), nil
}
func (f *environmentBackendModel) Close(context.Context, string) (ObservationResourceLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.containers) != 0 {
		return f.lease(), errors.New("live model runtimes remain")
	}
	return f.lease(), f.closeCause
}

func TestNPMEnvironmentFlowUsesTwoFreshWorkspacesAndOneBudget(t *testing.T) {
	backend, model, request := newEnvironmentBackendModel(t)
	result, err := backend.Execute(context.Background(), request)
	if err != nil || result.Status() != domain.SandboxCompleted {
		code, _ := result.LimitationCode()
		t.Fatalf("result=%s/%s err=%v", result.Status(), code, err)
	}
	if model.creates != 1 || model.registers != 4 || model.commands != 2 || model.introductions != 2 || model.snapshots != 4 || model.removed != 3 || len(model.containers) != 0 || len(model.volumes) != 0 {
		t.Fatalf("resources were reused, replenished or not drained: creates=%d registrations=%d commands=%d introductions=%d snapshots=%d removed=%d", model.creates, model.registers, model.commands, model.introductions, model.snapshots, model.removed)
	}
	states := result.EnvironmentStates()
	if len(states) != 2 || states[0].FixtureDigest == states[1].FixtureDigest {
		t.Fatal("fresh profile identities missing")
	}
	for _, state := range states {
		if state.Created != 1 || state.Deleted != 0 || state.Modified != 0 || state.Before == state.After {
			t.Fatalf("final snapshot facts not retained: %+v", state)
		}
	}
	counts := map[string]uint64{}
	for _, fact := range result.Observations() {
		counts[fact.Subject()] += fact.Count()
	}
	for _, profile := range []inspectionEnvironment{minimalEnvironment, developmentEnvironment} {
		if counts["environment-"+string(profile)+"-completed"] != 1 {
			t.Fatal("required environment not completed", counts)
		}
	}
}

func TestNPMEnvironmentPreparationAndFinalStateFailuresCannotComplete(t *testing.T) {
	for _, test := range []struct {
		failure, code string
		commands      int
	}{
		{"placement", "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", 0},
		{"responder-supply", "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE", 0},
		{"resolver", "M3_DYNAMIC_SETUP_FAILED", 0},
		{"input-mutated", "M3_DYNAMIC_ENVIRONMENT_INCOMPLETE", 1},
		{"registration", "M3_DYNAMIC_SETUP_FAILED", 0},
	} {
		t.Run(test.failure, func(t *testing.T) {
			backend, model, request := newEnvironmentBackendModel(t)
			model.failure = test.failure
			model.traceRecords = []TraceRecord{{Kind: "process-exec-expected", Bytes: 1}}
			result, err := backend.Execute(context.Background(), request)
			code, _ := result.LimitationCode()
			if err == nil || result.Status() != domain.SandboxIncomplete || code != test.code || model.commands != test.commands {
				t.Fatalf("result=%s/%s commands=%d err=%v", result.Status(), code, model.commands, err)
			}
			if len(model.containers) != 0 || len(model.volumes) != 0 {
				t.Fatal("known owned resources retained")
			}
			for _, reader := range model.traces {
				if len(reader.records) != 0 {
					t.Fatal("started observer was not drained")
				}
			}
		})
	}
}

func TestNPMResponderFailureAndCancellationCannotCompleteOrLeak(t *testing.T) {
	for _, failure := range []string{"responder-start", "responder-stop", "responder-cancel"} {
		t.Run(failure, func(t *testing.T) {
			backend, model, request := newEnvironmentBackendModel(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			model.failure, model.cancel = failure, cancel
			model.traceRecords = []TraceRecord{{Kind: "network-attempt", Bytes: 1}}
			result, err := backend.Execute(ctx, request)
			if err == nil || result.Status() != domain.SandboxIncomplete {
				t.Fatal("failed responder became completed", result.Status(), err)
			}
			if failure == "responder-cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("caller cancellation lost", err)
			}
			wantCommands := 1
			if failure == "responder-stop" {
				wantCommands = 2
			}
			if model.commands != wantCommands || model.terminations < 2 || model.creates != 1 || len(model.containers) != 0 || len(model.volumes) != 0 {
				t.Fatal("failure replenished authorization, ran extra work or leaked owned resources", model.commands, model.terminations)
			}
			for _, trace := range model.traces {
				if len(trace.records) != 0 {
					t.Fatal("failed responder trace not drained")
				}
			}
			found := false
			for _, fact := range result.Observations() {
				found = found || fact.Category() == domain.ObservationNetwork
			}
			if !found {
				t.Fatal("observed network fact lost during failure cleanup")
			}
		})
	}
}

func TestNPMEnvironmentRejectsReusedSessionAndAbsentResourcePort(t *testing.T) {
	for _, absent := range []bool{false, true} {
		t.Run(fmt.Sprintf("absent-resource=%t", absent), func(t *testing.T) {
			backend, model, request := newEnvironmentBackendModel(t)
			if absent {
				backend.resources = nil
			} else {
				session, _ := domain.NewSandboxSessionID()
				backend.newSessionID = func() (domain.SandboxSessionID, error) { return session, nil }
			}
			result, err := backend.Execute(context.Background(), request)
			code, _ := result.LimitationCode()
			want := "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED"
			if absent {
				want = "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE"
			}
			if err != nil || result.Status() != domain.SandboxIncomplete || code != want || model.commands != 0 || len(model.containers) != 0 || len(model.volumes) != 0 {
				t.Fatalf("result=%s/%s commands=%d err=%v", result.Status(), code, model.commands, err)
			}
		})
	}
}
