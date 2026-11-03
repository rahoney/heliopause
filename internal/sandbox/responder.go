package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const (
	responderToolName       = runtimeidentity.SandboxResponderAssetName
	responderGuestRoot      = "/haa-responder"
	responderGuestPath      = responderGuestRoot + "/" + responderToolName
	responderSupplyCapacity = int64(4 << 20)
	responderSupplyLabel    = "io.heliopause.responder-supply"
)

var errResponderSupply = errors.New("fixed guest responder supply is unavailable or changed")

// A single bounded, code-only supply is shared readonly by this inspection's
// writers. Unlike mutable tmpfs workspaces it must retain bytes before start.
// No artifact or responder ever receives a writable attachment.
type responderSupply struct {
	name, owner, digest, createdAt, mountpoint, imageID string
}

func validResponderImage(body []byte) bool {
	if len(body) == 0 || int64(len(body)) > responderSupplyCapacity-(64<<10) {
		return false
	}
	info, err := buildinfo.Read(bytes.NewReader(body))
	if err != nil || info.Path != "github.com/rahoney/heliopause/cmd/haa-sandbox-responder" || info.GoVersion != "go"+runtimeidentity.GoVersion {
		return false
	}
	settings := make(map[string]string, len(info.Settings))
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	return settings["CGO_ENABLED"] == "0" && settings["GOOS"] == "linux" && settings["GOARCH"] == runtime.GOARCH
}

func prepareResponderSupply(ctx context.Context, runner CommandRunner, installation string) (supply *responderSupply, resultErr error) {
	if ctx == nil || runner == nil {
		return nil, errResponderSupply
	}
	body, err := readInstalledNativeTool(ctx, installation, nativeRuntimeResponder)
	if err != nil || !validResponderImage(body) {
		return nil, errors.Join(errResponderSupply, err)
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		return nil, errors.Join(errResponderSupply, err)
	}
	digest := sha256.Sum256(body)
	supply = &responderSupply{owner: session.String(), digest: hex.EncodeToString(digest[:])}
	supply.name = "haa-responder-" + supply.owner
	value := supply
	image, err := runner.Output(ctx, "docker", "image", "inspect", nodeImageReference, "--format", "{{.Id}}")
	value.imageID = strings.TrimSpace(string(image))
	if err != nil || len(value.imageID) != 71 || !strings.HasPrefix(value.imageID, "sha256:") || !exactObservationContainerID(value.imageID[7:]) {
		return nil, errResponderSupply
	}
	listed, err := runner.Output(ctx, "docker", "volume", "ls", "--quiet", "--filter", "name=^"+value.name+"$")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return nil, errResponderSupply
	}
	// This volume contains at most one validated <=4MiB binary. No process ever
	// runs with a writable attachment. Mutable artifacts stay in quota'd tmpfs.
	created, createErr := runner.Output(ctx, "docker", "volume", "create", "--driver", "local",
		"--label", responderSupplyLabel+"="+value.owner, "--label", closureManifestLabel+"="+value.digest, value.name)
	if createErr != nil || strings.TrimSpace(string(created)) != value.name || value.verify(ctx, runner) != nil {
		return nil, errors.Join(errResponderSupply, createErr)
	}
	stager := "haa-responder-stage-" + value.owner
	var id string
	defer func() {
		cleanup, cancel := resolverCleanupContext()
		defer cancel()
		if id != "" {
			if err := value.verifyStager(cleanup, runner, id, stager); err != nil {
				resultErr = errors.Join(resultErr, err)
				supply = nil
				return
			}
			if _, err := runner.Output(cleanup, "docker", "rm", id); err != nil {
				resultErr = errors.Join(resultErr, err)
				supply = nil
				return
			}
		}
		if resultErr != nil {
			resultErr = errors.Join(resultErr, value.close(cleanup, runner))
			supply = nil
		}
	}()
	created, err = runner.Output(ctx, "docker", "create", "--pull", "never", "--name", stager,
		"--label", responderSupplyLabel+"="+value.owner, "--network", "none", "--read-only", "--user", "0:0",
		"--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--mount",
		"type=volume,source="+value.name+",target="+responderGuestRoot+",volume-nocopy", nodeImageReference)
	candidate := strings.TrimSpace(string(created))
	if !exactObservationContainerID(candidate) {
		return nil, errors.Join(errResponderSupply, err)
	}
	id = candidate
	if err != nil {
		return nil, errors.Join(errResponderSupply, err)
	}
	if err = value.verifyStager(ctx, runner, id, stager); err != nil {
		return nil, err
	}
	if err = introduceNativeTool(ctx, runner, id, nativeRuntimeResponder, body); err != nil {
		return nil, errors.Join(errResponderSupply, err)
	}
	return value, nil
}

func (s *responderSupply) verify(ctx context.Context, runner CommandRunner) error {
	if s == nil || ctx == nil || runner == nil || s.name != "haa-responder-"+s.owner || s.owner == "" || len(s.digest) != 64 {
		return errResponderSupply
	}
	body, err := runner.Output(ctx, "docker", "volume", "inspect", "--format", "{{json .}}", s.name)
	var actual dockerVolumeInspection
	if err != nil || len(body) == 0 || len(body) > 16<<10 || json.Unmarshal(body, &actual) != nil ||
		actual.Name != s.name || actual.Driver != "local" || actual.Scope != "local" || len(actual.Options) != 0 ||
		len(actual.Labels) != 2 || actual.Labels[responderSupplyLabel] != s.owner || actual.Labels[closureManifestLabel] != s.digest ||
		actual.CreatedAt == "" || actual.Mountpoint == "" || s.createdAt != "" && (s.createdAt != actual.CreatedAt || s.mountpoint != actual.Mountpoint) {
		return errResponderSupply
	}
	s.createdAt, s.mountpoint = actual.CreatedAt, actual.Mountpoint
	return nil
}

func (s *responderSupply) verifyStager(ctx context.Context, runner CommandRunner, id, name string) error {
	if s.verify(ctx, runner) != nil || !exactObservationContainerID(id) {
		return errResponderSupply
	}
	body, err := runner.Output(ctx, "docker", "inspect", "--format", "{{json .}}", id)
	var value struct {
		ID, Name, Image string
		Config          struct {
			Image, User string
			Labels      map[string]string
		}
		State struct {
			Status, StartedAt string
			Running           bool
			PID               int `json:"Pid"`
		}
		Mounts     []dockerMountInspection
		HostConfig struct {
			NetworkMode string
			Binds       []string
		}
	}
	if err != nil || len(body) == 0 || len(body) > 64<<10 || json.Unmarshal(body, &value) != nil ||
		value.ID != id || value.Name != "/"+name || value.Image != s.imageID || value.Config.Image != nodeImageReference || value.Config.User != "0:0" ||
		value.Config.Labels[responderSupplyLabel] != s.owner || value.State.Status != "created" || value.State.Running || value.State.PID != 0 ||
		value.State.StartedAt != "0001-01-01T00:00:00Z" || value.HostConfig.NetworkMode != "none" || len(value.HostConfig.Binds) != 0 || len(value.Mounts) != 1 {
		return errResponderSupply
	}
	m := value.Mounts[0]
	if m.Type != "volume" || m.Name != s.name || m.Source != s.mountpoint || m.Destination != responderGuestRoot || m.Driver != "local" || !m.RW {
		return errResponderSupply
	}
	return nil
}

func (s *responderSupply) mountArgument() (string, error) {
	if s == nil || s.name != "haa-responder-"+s.owner || s.createdAt == "" || s.mountpoint == "" {
		return "", errResponderSupply
	}
	return "type=volume,source=" + s.name + ",target=" + responderGuestRoot + ",volume-nocopy,readonly", nil
}

func (s *responderSupply) close(ctx context.Context, runner CommandRunner) error {
	if err := s.verify(ctx, runner); err != nil {
		return err
	}
	listed, err := runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "volume="+s.name, "--format", "{{.ID}}")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return errors.Join(errResponderSupply, err)
	}
	if _, err := runner.Output(ctx, "docker", "volume", "rm", s.name); err != nil {
		return fmt.Errorf("remove fixed responder supply: %w", err)
	}
	return nil
}

// Init attachment has a separate controller pipe; it never occupies a direct
// exec admission while the Artifact runs in this same container.
type inspectionResponder struct {
	mu             sync.Mutex
	position       int
	stopRequested  bool
	input          *io.PipeWriter
	cancel         context.CancelFunc
	ready, stopped chan struct{}
	done           chan struct{}
	result         error
	fault          error
}

func (r *inspectionResponder) Write(body []byte) (int, error) {
	const protocol = "READY\nSTOPPED\n"
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fault != nil {
		return 0, r.fault
	}
	for index, value := range body {
		if r.position >= len(protocol) || value != protocol[r.position] || r.position >= 6 && !r.stopRequested {
			r.fault = errResponderSupply
			if r.cancel != nil {
				r.cancel()
			}
			return index, r.fault
		}
		r.position++
		if r.position == 6 {
			close(r.ready)
		}
		if r.position == len(protocol) {
			close(r.stopped)
		}
	}
	return len(body), nil
}

func startResponderInit(ctx context.Context, runner CommandRunner, id string) (*inspectionResponder, error) {
	if ctx == nil {
		return nil, errResponderSupply
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errResponderSupply, err)
	}
	executor, ok := runner.(interface {
		StartContainerInputOutput(context.Context, io.Reader, io.Writer, string) error
	})
	if !ok || !exactObservationContainerID(id) {
		return nil, errResponderSupply
	}
	run, cancel := context.WithCancel(ctx)
	input, control := io.Pipe()
	response := &inspectionResponder{input: control, cancel: cancel, ready: make(chan struct{}), stopped: make(chan struct{}), done: make(chan struct{})}
	go func() {
		err := executor.StartContainerInputOutput(run, input, response, id)
		_ = input.CloseWithError(err)
		response.mu.Lock()
		response.result = errors.Join(response.fault, err)
		response.mu.Unlock()
		close(response.done)
	}()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-response.ready:
		response.mu.Lock()
		fault := response.fault
		response.mu.Unlock()
		if fault != nil {
			cancel()
			_ = control.Close()
			return nil, fault
		}
		return response, nil
	case <-response.done:
		cancel()
		_ = control.Close()
		response.mu.Lock()
		err := response.result
		response.mu.Unlock()
		return nil, errors.Join(errResponderSupply, err)
	case <-ctx.Done():
		cancel()
		_ = control.Close()
		return nil, errors.Join(errResponderSupply, ctx.Err())
	case <-timer.C:
		cancel()
		_ = control.Close()
		return nil, errResponderSupply
	}
}

// Stop only the finite protocols. Resource termination and observer drain must
// still follow before waitAfterTermination can close the controller channel.
func (r *inspectionResponder) stop(ctx context.Context) error {
	r.mu.Lock()
	if r.stopRequested || r.fault != nil {
		r.mu.Unlock()
		return errResponderSupply
	}
	r.stopRequested = true
	r.mu.Unlock()
	written := make(chan error, 1)
	go func() { _, err := io.WriteString(r.input, "STOP\n"); written <- errors.Join(err, r.input.Close()) }()
	select {
	case err := <-written:
		if err != nil {
			return errors.Join(errResponderSupply, err)
		}
	case <-ctx.Done():
		_ = r.input.CloseWithError(ctx.Err())
		return ctx.Err()
	}
	select {
	case <-r.stopped:
		return nil
	case <-r.done:
		r.mu.Lock()
		err := r.result
		r.mu.Unlock()
		return errors.Join(errResponderSupply, err)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// The caller supplies the independently confirmed resource termination result.
// A forced init exit is expected only after our STOP acknowledgement and that
// confirmation. Other exits/cancellation retain their original failure.
func (r *inspectionResponder) waitAfterTermination(ctx context.Context, terminated bool) error {
	defer r.cancel()
	defer r.input.Close()
	select {
	case <-r.done:
		r.mu.Lock()
		err := r.result
		stopped := r.stopRequested && r.position == len("READY\nSTOPPED\n") && r.fault == nil
		r.mu.Unlock()
		if !terminated || !stopped {
			return errors.Join(errResponderSupply, err)
		}
		if err == nil {
			return nil
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 137 {
			return nil
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *responderSupply) expectedTopology(base []observerMountExpectation) ([]observerMountExpectation, error) {
	if _, err := s.mountArgument(); err != nil {
		return nil, err
	}
	result := append([]observerMountExpectation(nil), base...)
	for _, mount := range result {
		if mount.Mountpoint == responderGuestRoot || strings.HasPrefix(mount.Mountpoint, responderGuestRoot+"/") {
			return nil, errResponderSupply
		}
	}
	return append(result, observerMountExpectation{responderGuestRoot, "helper", "/", "9p", true, false, false, false}), nil
}

func (s *responderSupply) verifyReader(ctx context.Context, runner CommandRunner, id string) error {
	if s.verify(ctx, runner) != nil || !exactObservationContainerID(id) {
		return errResponderSupply
	}
	body, err := runner.Output(ctx, "docker", "inspect", "--format", containerMountInspectTemplate, id)
	var actual dockerContainerMountInspection
	if err != nil || len(body) == 0 || len(body) > 16<<10 || json.Unmarshal(body, &actual) != nil || actual.ID != id || len(actual.Binds) != 0 {
		return errResponderSupply
	}
	mounted, configured := 0, 0
	for _, mount := range actual.Mounts {
		if mount.Type == "bind" || strings.HasPrefix(mount.Destination, responderGuestRoot+"/") {
			return errResponderSupply
		}
		if mount.Destination == responderGuestRoot {
			mounted++
			if mount.Type != "volume" || mount.Name != s.name || mount.Source != s.mountpoint || mount.Driver != "local" || mount.RW {
				return errResponderSupply
			}
		} else if mount.Type == "volume" && mount.Name == s.name {
			return errResponderSupply
		}
	}
	for _, mount := range actual.HostMounts {
		if mount.Type == "bind" || strings.HasPrefix(mount.Target, responderGuestRoot+"/") {
			return errResponderSupply
		}
		if mount.Target == responderGuestRoot {
			configured++
			if mount.Type != "volume" || mount.Source != s.name || !mount.ReadOnly || !mount.VolumeOptions.NoCopy || mount.VolumeOptions.Subpath != "" {
				return errResponderSupply
			}
		} else if mount.Type == "volume" && mount.Source == s.name {
			return errResponderSupply
		}
	}
	if mounted != 1 || configured != 1 {
		return errResponderSupply
	}
	return nil
}
