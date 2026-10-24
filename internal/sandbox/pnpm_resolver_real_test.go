//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/hosttool"
)

// This actual guest boundary test deliberately has an inert policy service and
// an empty project. It proves lifecycle/output rejection, not public egress.
type pnpmNativeFaultRunner struct {
	CommandRunner
	mode                         string
	cause                        error
	commandFaults, cleanupFaults int
}

func (r *pnpmNativeFaultRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if r.mode == "command and cleanup" {
		for _, arg := range args {
			if arg == pnpmToolGuestPath {
				r.commandFaults++
				return nil, r.cause
			}
		}
	}
	body, err := r.CommandRunner.Output(ctx, name, args...)
	if len(args) > 0 && args[0] == "rm" && (r.mode == "cleanup acknowledgement" || r.mode == "command and cleanup") {
		r.cleanupFaults++
		err = errors.Join(err, r.cause)
	}
	return body, err
}
func (r *pnpmNativeFaultRunner) RunInput(ctx context.Context, body io.Reader, name string, args ...string) error {
	return r.CommandRunner.(inputCommandRunner).RunInput(ctx, body, name, args...)
}
func (r *pnpmNativeFaultRunner) RunOutput(ctx context.Context, body io.Writer, name string, args ...string) error {
	return r.CommandRunner.(streamingDockerOutput).RunOutput(ctx, body, name, args...)
}

type pnpmNativeFaultObserver struct {
	TraceObserver
	mode       string
	injections *int
}

func (o pnpmNativeFaultObserver) StartProfile(ctx context.Context, id, profile string) (TraceReader, error) {
	reader, err := startTrace(ctx, o.TraceObserver, id, profile)
	if err != nil {
		return nil, err
	}
	return &pnpmNativeFaultTrace{TraceReader: reader, mode: o.mode, injections: o.injections}, nil
}
func (o pnpmNativeFaultObserver) AwaitMountAnchors(ctx context.Context, id string) error {
	return awaitMountAnchors(ctx, o.TraceObserver, id)
}

type pnpmNativeFaultTrace struct {
	TraceReader
	mode       string
	injected   bool
	injections *int
}

func (r *pnpmNativeFaultTrace) Next(ctx context.Context) (TraceRecord, error) {
	record, err := r.TraceReader.Next(ctx)
	if errors.Is(err, io.EOF) && !r.injected {
		r.injected = true
		switch r.mode {
		case "incomplete observation":
			*r.injections++
			return TraceRecord{}, errors.New("injected observer incompleteness")
		case "forbidden observation":
			*r.injections++
			return TraceRecord{Kind: "network-attempt", Bytes: 64}, nil
		}
	}
	return record, err
}

func TestPNPMActualNativeProjectSelectionBoundary(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_PNPM_NATIVE_RUNTIME") != "1" {
		t.Fatal("requires ordinary-user declared isolated PNPM qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, mode := range []string{"complete", "command and cleanup", "cleanup acknowledgement", "incomplete observation", "forbidden observation"} {
		passed := t.Run(mode, func(t *testing.T) {
			cause := errors.New("injected command/cleanup acknowledgement failure")
			injections := 0
			runner := &pnpmNativeFaultRunner{CommandRunner: admissionAwareRunner(executor), mode: mode, cause: cause}
			resolver, err := NewPNPMResolver(runner, &pnpmCountingEndpoints{}, pnpmNativeFaultObserver{supervisor.Observer(), mode, &injections}, integrationCapabilityProbe(executor), &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			input := pnpmSelectionInputFixture(t, artifactnpm.PNPMInstall, "", `{}`)
			selected, err := resolver.SelectProject(ctx, input)
			if mode == "complete" {
				if err != nil || !selected.Selection().DependencyFree() || len(selected.Controls()) != 2 {
					t.Fatal("empty native selection failed", err)
				}
			} else {
				if err == nil || len(selected.Controls()) != 0 || selected.Selection().DependencyFree() {
					t.Fatal("failed lifecycle retained usable native output", err)
				}
				if (mode == "command and cleanup" || mode == "cleanup acknowledgement") && !errors.Is(err, cause) {
					t.Fatal("primary/cleanup cause lost", err)
				}
			}
			wantCommand, wantCleanup, wantObservation := 0, 0, 0
			switch mode {
			case "command and cleanup":
				wantCommand, wantCleanup = 1, 1
			case "cleanup acknowledgement":
				wantCleanup = 1
			case "incomplete observation", "forbidden observation":
				wantObservation = 1
			}
			if runner.commandFaults != wantCommand || runner.cleanupFaults != wantCleanup || injections != wantObservation {
				t.Fatal("intended fault was not reached", runner.commandFaults, runner.cleanupFaults, injections)
			}
			t.Logf("native_project_boundary=%s output_controls=%d rejected=%t public_egress=NOT_TESTED", mode, len(selected.Controls()), err != nil)
		})
		if !passed && mode == "complete" {
			t.FailNow()
		}
	}
}
