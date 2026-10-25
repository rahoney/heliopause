package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// The original project fixture still owns its project protocol. This decorator
// adds the separately attested native volume; it supplies no execution proof.
type bunArtifactRuntimeRunner struct {
	*pnpmInstanceRunner
	native                        dockerVolumeInspection
	nativeNames                   []string
	nativeRemovals, introductions int
	nativeFault                   string
	cause                         error
}

func (r *bunArtifactRuntimeRunner) Output(ctx context.Context, binary string, args ...string) ([]byte, error) {
	if binary == "docker" && len(args) > 2 && args[0] == "volume" && strings.HasPrefix(args[len(args)-1], "haa-bun-runtime-") {
		r.calls++
		switch args[1] {
		case "create":
			r.native = dockerVolumeInspection{Name: args[len(args)-1], Driver: "local", Scope: "local", Mountpoint: "/owned/native", CreatedAt: "native-fixed", Options: map[string]string{}, Labels: map[string]string{}}
			for i := 2; i+1 < len(args); i++ {
				if args[i] == "--opt" || args[i] == "--label" {
					key, value, _ := strings.Cut(args[i+1], "=")
					if args[i] == "--opt" {
						r.native.Options[key] = value
					} else {
						r.native.Labels[key] = value
					}
					i++
				}
			}
			r.nativeNames = append(r.nativeNames, r.native.Name)
			return []byte(r.native.Name), nil
		case "inspect":
			return r.encoded(r.native), nil
		case "rm":
			r.nativeRemovals++
			if r.nativeFault == "cleanup" {
				return nil, r.cause
			}
			return nil, nil
		}
	}
	if binary == "docker" && len(args) > 0 && args[0] == "create" {
		copy := slices.Clone(args)
		mount := "type=volume,source=" + r.native.Name + ",target=/haa-runtime,volume-nocopy"
		index := slices.Index(copy, mount)
		if index < 1 || copy[index-1] != "--mount" {
			r.t.Fatal("native runtime mount absent")
		}
		copy[index-1], copy[index] = "--tmpfs", boundaryHelperMount
		return r.pnpmInstanceRunner.Output(ctx, binary, copy...)
	}
	if binary == "docker" && len(args) > 2 && args[0] == "inspect" && args[2] == containerMountInspectTemplate {
		body, err := r.pnpmInstanceRunner.Output(ctx, binary, args...)
		if err != nil {
			return nil, err
		}
		var view dockerContainerMountInspection
		if err := json.Unmarshal(body, &view); err != nil {
			r.t.Fatal(err)
		}
		actual := dockerMountInspection{Type: "volume", Name: r.native.Name, Source: r.native.Mountpoint, Destination: "/haa-runtime", Driver: "local", RW: true}
		if r.nativeFault == "native-mount" {
			actual.Source = "/foreign"
		}
		configured := dockerHostMountInspection{Type: "volume", Source: r.native.Name, Target: "/haa-runtime"}
		configured.VolumeOptions.NoCopy = true
		view.Mounts = append(view.Mounts, actual)
		view.HostMounts = append(view.HostMounts, configured)
		return r.encoded(view), nil
	}
	if binary == "docker" && len(args) > 0 && args[0] == "ps" && slices.Contains(args, "volume="+r.native.Name) && r.nativeFault == "foreign-runtime" && r.active {
		r.calls++
		return []byte(strings.Repeat("e", 64)), nil
	}
	if binary == "docker" && len(args) > 0 && args[0] == "exec" && slices.Contains(args, bunRuntimeTargetLifecycleCommand) {
		copy := slices.Clone(args)
		copy[slices.Index(copy, bunRuntimeTargetLifecycleCommand)] = bunTargetLifecycleCommand
		return r.pnpmInstanceRunner.Output(ctx, binary, copy...)
	}
	return r.pnpmInstanceRunner.Output(ctx, binary, args...)
}

type bunArtifactRuntimeTool struct {
	runner *bunArtifactRuntimeRunner
	kind   nativeRuntimeToolKind
}

func (p bunArtifactRuntimeTool) nativeKind() nativeRuntimeToolKind { return p.kind }
func (p bunArtifactRuntimeTool) introduce(_ context.Context, _ CommandRunner, id string) error {
	if id != p.runner.id {
		p.runner.t.Fatal("native tool introduced to foreign session")
	}
	p.runner.introductions++
	p.runner.events = append(p.runner.events, "bun-tool")
	if p.runner.nativeFault == "native-copy" {
		return p.runner.cause
	}
	return nil
}

type bunArtifactRuntimeObserver struct{ *pnpmInstanceObserver }

func (o *bunArtifactRuntimeObserver) StartBunRuntimeTargetLifecycleProfile(ctx context.Context, id string) (TraceReader, error) {
	return o.StartPNPMTargetLifecycleProfile(ctx, id)
}

func newBunArtifactRuntimeTestBackend(t *testing.T, r *bunArtifactRuntimeRunner) *Backend {
	t.Helper()
	b, err := NewBackend(r, &recordingIntroducer{}, &bunArtifactRuntimeObserver{&pnpmInstanceObserver{}}, availableProbe)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBunArtifactRuntimeOwnsFreshNativeAndProjectVolumes(t *testing.T) {
	layout, subject, instance := bunInstanceFixture(t)
	r := &bunArtifactRuntimeRunner{pnpmInstanceRunner: &pnpmInstanceRunner{kind: nativeRuntimeBun, t: t}}
	b := newBunArtifactRuntimeTestBackend(t, r)
	prepare := func(context.Context) (nativeProjectTool, error) {
		return bunArtifactRuntimeTool{r, nativeRuntimeBun}, nil
	}
	var sessions []string
	for range 2 {
		result, err := b.executeBunRuntimeInstance(context.Background(), layout, subject, instance, prepare)
		if err != nil || result.Status() != domain.SandboxCompleted {
			t.Fatal("runtime supply failed", err, result.Status())
		}
		sessions = append(sessions, result.SessionID().String())
	}
	if sessions[0] == sessions[1] || len(r.nativeNames) != 2 || r.nativeNames[0] == r.nativeNames[1] || r.nativeRemovals != 2 || r.volumeRemovals != 2 || r.removals != 2 || r.introductions != 2 || r.executions != 2 {
		t.Fatal("runtime/session reused or not finalized", r)
	}
	if slices.Index(r.events, "bun-tool") < 0 || slices.Index(r.events, "bun-tool") > slices.Index(r.events, "copy") {
		t.Fatal("bound artifact preceded verified native supply")
	}
}

func TestBunArtifactRuntimeRefusesUnprovenSupplyAndPreservesCause(t *testing.T) {
	layout, subject, instance := bunInstanceFixture(t)
	for _, mode := range []string{"prepare", "wrong-kind", "native-mount", "foreign-runtime", "native-copy", "cleanup"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("exact native runtime failure")
			r := &bunArtifactRuntimeRunner{pnpmInstanceRunner: &pnpmInstanceRunner{kind: nativeRuntimeBun, t: t}, nativeFault: mode, cause: cause}
			b := newBunArtifactRuntimeTestBackend(t, r)
			prepare := func(context.Context) (nativeProjectTool, error) {
				if mode == "prepare" {
					return nil, cause
				}
				kind := nativeRuntimeBun
				if mode == "wrong-kind" {
					kind = nativeRuntimePNPM
				}
				return bunArtifactRuntimeTool{r, kind}, nil
			}
			result, err := b.executeBunRuntimeInstance(context.Background(), layout, subject, instance, prepare)
			if err == nil || result.Status() != domain.SandboxIncomplete {
				t.Fatal("unproven runtime admitted", err)
			}
			if mode != "cleanup" && r.executions != 0 {
				t.Fatal("failed supply executed artifact")
			}
			if mode == "prepare" || mode == "wrong-kind" {
				if r.calls != 0 {
					t.Fatal("preflight started Docker")
				}
			} else if r.nativeRemovals != 1 || r.removals != 1 || r.volumeRemovals != 1 {
				t.Fatal("runtime resources not finalized")
			}
			if (mode == "prepare" || mode == "native-copy" || mode == "cleanup") && !errors.Is(err, cause) {
				t.Fatal("native failure cause lost", err)
			}
		})
	}
}

func TestBunArtifactRuntimePreflightAvoidsToolAndRuntime(t *testing.T) {
	layout, subject, key := bunInstanceFixture(t)
	for _, mode := range []string{"zero", "subject", "instance", "cancel", "observer", "factory"} {
		t.Run(mode, func(t *testing.T) {
			r := &bunArtifactRuntimeRunner{pnpmInstanceRunner: &pnpmInstanceRunner{kind: nativeRuntimeBun, t: t}}
			b := newBunArtifactRuntimeTestBackend(t, r)
			candidate, item, instance := layout, subject, key
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			prepare := func(context.Context) (nativeProjectTool, error) { calls++; return nil, errors.New("must not prepare") }
			switch mode {
			case "zero":
				candidate = artifactnpm.BunInspectionLayout{}
			case "subject":
				item = domain.AcquiredArtifact{}
			case "instance":
				instance += "foreign"
			case "cancel":
				cancel()
			case "observer":
				b.observer = &emptyObserver{}
			case "factory":
				prepare = nil
			}
			if _, err := b.executeBunRuntimeInstance(ctx, candidate, item, instance, prepare); err == nil || calls != 0 || r.calls != 0 {
				t.Fatal("preflight performed runtime work", err)
			}
		})
	}
}

func TestBunArtifactRuntimeMountAndTopologyAreSeparate(t *testing.T) {
	native := nativeRuntimeVolume{name: "haa-bun-runtime-test", owner: "test", kind: nativeRuntimeBun, createdAt: "fixed", mountpoint: "/owned"}
	old := []string{"create", "--tmpfs", boundaryHelperMount, "image"}
	args, err := bunArtifactRuntimeCreateArguments(old, native)
	if err != nil || !slices.Equal(old, []string{"create", "--tmpfs", boundaryHelperMount, "image"}) || args[1] != "--mount" {
		t.Fatal("fixed mount changed original input", err)
	}
	for _, candidate := range [][]string{{"create"}, {"--tmpfs", boundaryHelperMount, "--tmpfs", boundaryHelperMount}} {
		if _, err := bunArtifactRuntimeCreateArguments(candidate, native); err == nil {
			t.Fatal("ambiguous helper mount admitted")
		}
	}
	native.kind = nativeRuntimePNPM
	if _, err := bunArtifactRuntimeCreateArguments(old, native); err == nil {
		t.Fatal("foreign native runtime admitted")
	}
	base, ok := bunTargetLifecycleExpectedTopology()
	runtime, known := bunRuntimeTargetLifecycleExpectedTopology()
	if !ok || !known || len(base) != len(runtime) {
		t.Fatal("topology unavailable")
	}
	changes := 0
	for i := range base {
		if base[i] != runtime[i] {
			changes++
			if base[i].Mountpoint != "/haa-runtime" || base[i].FSType != "tmpfs" || runtime[i].FSType != "9p" || runtime[i].NoSUID {
				t.Fatal("unrelated topology changed")
			}
		}
	}
	after, _ := bunTargetLifecycleExpectedTopology()
	if changes != 1 || !slices.Equal(after, base) {
		t.Fatal("legacy topology mutated")
	}
}

func TestBunArtifactRuntimeCommandMatchesExactObserverBoundary(t *testing.T) {
	target := bunProjectDirectory + "/node_modules/@scope/pkg"
	args, err := bunRuntimeTargetLifecycleExecArguments(strings.Repeat("a", 64), target)
	if err != nil || !slices.Contains(args, bunRuntimeTargetLifecycleCommand) || slices.Contains(args, bunTargetLifecycleCommand) || args[len(args)-1] != target {
		t.Fatal("native runtime command selection differs", err)
	}
	if _, err := bunRuntimeTargetLifecycleExecArguments("unknown", target); err == nil {
		t.Fatal("ambiguous native container")
	}
	body, err := os.ReadFile("../../tools/gvisor-observer/observer.cc")
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := tokenizeCPPSource(body)
	if err != nil {
		t.Fatal(err)
	}
	var commands []string
	for i := 0; i+7 < len(tokens); i++ {
		if tokens[i].val == "constexpr" && tokens[i+1].val == "char" && tokens[i+2].val == "kBunRuntimeTargetLifecycleCommand" && tokens[i+3].val == "[" && tokens[i+4].val == "]" && tokens[i+5].val == "=" && tokens[i+6].typ == cppTokRaw && tokens[i+6].delim == "HAA" && tokens[i+7].val == ";" {
			commands = append(commands, tokens[i+6].body)
		}
	}
	if len(commands) != 1 || commands[0] != bunRuntimeTargetLifecycleCommand {
		t.Fatal("Go and observer native command differ")
	}
}
