package sandbox

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/netip"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type npmWholeSelectionRunner struct {
	npmUpdateRecordingRunner
	manifest, lock []byte
	readError      error
}

func (r *npmWholeSelectionRunner) RunOutput(ctx context.Context, output io.Writer, binary string, arguments ...string) error {
	_, bounded := ctx.Deadline()
	r.calls = append(r.calls, commandCall{binary: binary, arguments: append([]string(nil), arguments...), bounded: bounded})
	if r.readError != nil {
		return r.readError
	}
	var body []byte
	switch arguments[len(arguments)-1] {
	case resolverProjectDir + "/package.json":
		body = r.manifest
	case resolverProjectDir + "/package-lock.json":
		body = r.lock
	default:
		return errors.New("unexpected bounded control path")
	}
	_, err := output.Write(body)
	return err
}

func npmWholeNativeFixture(t *testing.T, intent artifactnpm.NPMProjectIntent, withLock bool) (artifactnpm.NPMProjectInput, *npmWholeSelectionRunner, *NPMResolver) {
	t.Helper()
	manifest := []byte(`{"dependencies":{"primary":"1.0.0"}}`)
	lock := []byte(resolverLockJSON())
	control, _ := domain.NewProjectControlFile("package.json", manifest, true)
	var original []byte
	if withLock {
		original = lock
	}
	locked, _ := domain.NewProjectControlFile("package-lock.json", original, withLock)
	input, err := artifactnpm.NewNPMProjectInput(intent, []domain.ProjectControlFile{control, locked}, resolverNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	runner := &npmWholeSelectionRunner{manifest: manifest, lock: lock}
	runner.responses = [][]byte{[]byte("0123456789abcdef"), []byte("172.30.0.0/24"), []byte("0123456789ab"), nil, nil, []byte(resolverNPMVersion), nil}
	resolver, err := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, &sequencedObserver{}, &recordingResolverPolicyService{})
	if err != nil {
		t.Fatal(err)
	}
	return input, runner, resolver
}

func TestNPMWholeNativeSelectionPreservesIntentAndBoundedControlOutput(t *testing.T) {
	for _, test := range []struct {
		intent artifactnpm.NPMProjectIntent
		locked bool
		argv   string
	}{{artifactnpm.NPMProjectInstall, false, "npm install --package-lock-only"}, {artifactnpm.NPMProjectInstall, true, "npm install --package-lock-only"}, {artifactnpm.NPMProjectCI, true, "npm ci --dry-run"}} {
		t.Run(string(test.intent)+"/"+test.argv, func(t *testing.T) {
			input, runner, resolver := npmWholeNativeFixture(t, test.intent, test.locked)
			selected, err := resolver.SelectProject(context.Background(), input)
			if err != nil || len(selected.Controls()) != 2 || len(selected.Selection().Packages()) != 1 {
				t.Fatal("unapproved complete selection unavailable", err)
			}
			wantInputs := 1
			if test.locked {
				wantInputs = 2
			}
			if len(runner.inputs) != wantInputs || !bytes.Equal(runner.inputs[0], input.Controls()[0].Body()) {
				t.Fatal("frozen controls were not copied exactly")
			}
			commands, boundedReads, removals := 0, 0, 0
			for _, call := range runner.calls {
				joined := strings.Join(call.arguments, " ")
				if strings.Contains(joined, "HOME=/tmp npm_config_cache=/tmp/cache ") {
					commands++
					for _, required := range []string{test.argv, "--ignore-scripts", "--no-audit --no-fund", "--registry=https://registry.npmjs.org/", "--userconfig=/tmp/haa-user.npmrc --globalconfig=/tmp/haa-global.npmrc"} {
						if !strings.Contains(joined, required) {
							t.Fatal("native intent or isolation flag changed", joined)
						}
					}
				}
				if strings.HasSuffix(joined, "cat "+resolverProjectDir+"/package.json") || strings.HasSuffix(joined, "cat "+resolverProjectDir+"/package-lock.json") {
					boundedReads++
					if !call.bounded {
						t.Fatal("control read has no deadline")
					}
				}
				if len(call.arguments) == 3 && call.arguments[0] == "rm" && call.arguments[1] == "--force" {
					removals++
				}
			}
			if commands != 1 || boundedReads != 2 || removals != 1 {
				t.Fatal("selection/lifecycle counts differ", commands, boundedReads, removals)
			}
		})
	}
}

func TestNPMWholeNativeSelectionRejectsDriftFailuresAndCleanupWithoutPartialResult(t *testing.T) {
	for _, fault := range []string{"manifest", "ci-lock-bytes", "malformed-lock", "oversized-control", "read-failure", "native-failure", "cleanup-failure"} {
		t.Run(fault, func(t *testing.T) {
			input, runner, resolver := npmWholeNativeFixture(t, artifactnpm.NPMProjectCI, true)
			sentinel := errors.New("sentinel native failure")
			switch fault {
			case "manifest":
				runner.manifest = append(runner.manifest, '\n')
			case "ci-lock-bytes":
				runner.lock = append(runner.lock, '\n')
			case "malformed-lock":
				runner.lock = []byte("{")
			case "oversized-control":
				runner.manifest = bytes.Repeat([]byte(" "), artifactnpm.ProjectControlLimit+1)
			case "read-failure":
				runner.readError = sentinel
			case "native-failure":
				runner.errors = make([]error, 7)
				runner.errors[6] = sentinel
			case "cleanup-failure":
				runner.errors = make([]error, 10)
				runner.errors[9] = sentinel
			}
			selected, err := resolver.SelectProject(context.Background(), input)
			if err == nil || len(selected.Controls()) != 0 || len(selected.Selection().Packages()) != 0 {
				t.Fatal("failed selection returned authority or lost failure", err)
			}
			if (fault == "read-failure" || fault == "native-failure") && !errors.Is(err, sentinel) {
				t.Fatal("primary failure was lost", err)
			}
		})
	}
}

func TestNPMWholeNativePreflightCreatesNoResources(t *testing.T) {
	input, runner, resolver := npmWholeNativeFixture(t, artifactnpm.NPMProjectInstall, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := resolver.SelectProject(ctx, input); !errors.Is(err, context.Canceled) || len(runner.calls) != 0 || len(runner.inputs) != 0 {
		t.Fatal("cancelled request created resources", err)
	}
	if _, err := resolver.SelectProject(context.Background(), artifactnpm.NPMProjectInput{}); err == nil || len(runner.calls) != 0 {
		t.Fatal("zero input created resources")
	}
	// The admission wrapper advertises RunOutput, but the underlying executor
	// must actually supply that capability before any DNS/container operation.
	plain := &recordingRunner{}
	withoutOutput, err := NewNPMResolverWithObserver(plain, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, &sequencedObserver{}, &recordingResolverPolicyService{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := withoutOutput.SelectProject(context.Background(), input); err == nil || len(plain.calls) != 0 {
		t.Fatal("missing bounded executor reached resource creation")
	}
}
