//go:build linux && realcorpus

package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

const pipxRunProbeCommand = "helox-pipx-inert-query"

// Diagnostic-only argv. The original reader and actual session remain the
// existing production owners; this neither selects a program nor grants Policy.
func pipxRunProbeArguments(frozen *pipxNativeInput, requirement artifactpypi.UpgradeRequirement, command string) ([]string, error) {
	if validateUVRunRequest(requirement, command) != nil || !pipxNativeRequirementMatches(requirement, frozen) || requirement.Project() == pipxRunProbeCommand {
		return nil, errPipxNativeInput
	}
	_, inspections, err := uvRunConsoleOriginal(frozen, command)
	if err != nil {
		return nil, err
	}
	for _, info := range inspections {
		for _, ep := range info.Surface.EntryPointDetails {
			if ep.Name == pipxRunProbeCommand && (ep.Group == "pipx.run" || ep.Group == "console_scripts" || ep.Group == "gui_scripts") {
				return nil, errors.New("pipx inert query collides with an original entry point")
			}
		}
		for _, script := range info.Surface.Scripts {
			if path.Base(script) == pipxRunProbeCommand {
				return nil, errors.New("pipx inert query collides with an original script")
			}
		}
	}
	installed, err := pipxNativeArguments(pipxNativeInstall, requirement)
	if err != nil {
		return nil, err
	}
	// Keep the existing isolated original-ZIP interpreter and fixed offline UV
	// flags. Different app/spec names take the native Venv preparation branch.
	args := append([]string(nil), installed[:6]...)
	return append(args, "run", "--backend", "uv", "--python", "/usr/local/bin/python", "--no-cache", "--no-path-check", "--spec", requirement.Requirement(), installed[len(installed)-1], pipxRunProbeCommand), nil
}

func TestPipxRunPreparationSafety(t *testing.T) {
	requirement, err := artifactpypi.ParseUpgradeRequirement("probe@1.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range []string{"", "pipx.run", "console_scripts", "gui_scripts", "other.group", "raw-script"} {
		t.Run(group, func(t *testing.T) {
			entries := "[console_scripts]\nprobe=probe:main\n"
			extra := map[string][]byte{}
			if group == "raw-script" {
				extra["probe-1.0.data/scripts/"+pipxRunProbeCommand] = []byte("#!/usr/local/bin/python\nraise RuntimeError('must not execute')\n")
			} else if group != "" {
				if group != "console_scripts" {
					entries += "[" + group + "]\n"
				}
				entries += pipxRunProbeCommand + "=probe:main\n"
			}
			extra["probe-1.0.dist-info/entry_points.txt"] = []byte(entries)
			frozen, _, _, _, _ := pipxFrozenInputFixture(t, extra)
			args, err := pipxRunProbeArguments(frozen, requirement, "probe")
			allowed := group == "" || group == "other.group"
			if allowed != (err == nil) || !allowed && len(args) != 0 {
				t.Fatal("inert query role guard differs", group, err)
			}
			if allowed && (len(args) < 7 || args[5] != pipxNativeProgram || args[6] != "run" || args[len(args)-1] != pipxRunProbeCommand || strings.Contains(strings.Join(args[6:], " "), "/dev/null")) {
				t.Fatal("query did not retain the fixed native frontend/terminal")
			}
		})
	}
}

func TestLinuxPipxFrozenRunPreparationProbe(t *testing.T) {
	if os.Getenv("HELOX_PIPX_RUN_PREPARATION_PROBE") != "1" {
		t.Skip("requires explicit fixed original inputs and observed native DATA probe")
	}
	if os.Geteuid() == 0 {
		t.Fatal("ordinary caller required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	input := pipxDiagnosticFrozenInput(t)
	frozen, err := freezePipxNativeInput(ctx, input.lock, input.inputs, input.pages)
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement("userpath==" + input.version)
	if err != nil {
		t.Fatal(err)
	}
	command, err := pipxRunProbeArguments(frozen, requirement, "userpath")
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(os.Getenv("HELOX_PIPX_RUNTIME_ZIP"))
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := nativeRuntimeToolIdentity(nativeRuntimePipx)
	digest := sha256.Sum256(body)
	if !ok || int64(len(body)) != spec.size || hex.EncodeToString(digest[:]) != spec.sha256 {
		t.Fatal("native probe ZIP differs from the fixed supply")
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	client, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	resources := &pipxActualResources{ObservationResourceClient: integrationObservationResources{client: client}}
	runner := admissionAwareRunner(&pipxNativeSessionActualRunner{CommandRunner: executor})
	runErr := func() (resultErr error) {
		now := time.Now()
		policy, err := observationPolicyForPython(artifactpypi.PublicPyPIProfile().ResourcePolicy(), pipxNativeResolverProfile, 0, now)
		if err != nil {
			return err
		}
		ledger, err := newObservationTransaction(policy, nil, now)
		if err != nil {
			return err
		}
		run, stop := context.WithDeadline(ctx, policy.wallDeadline)
		defer stop()
		uv, err := prepareUVTool(run, runner)
		if err != nil {
			return err
		}
		identity, err := domain.NewSandboxSessionID()
		if err != nil {
			return err
		}
		s := &pipxNativeSession{runner: runner, observer: supervisor.Observer(), resources: resources, transaction: identity.String(), ledger: ledger}
		var nativeErr error
		defer func() {
			resultErr = errors.Join(resultErr, s.cleanup(), ctx.Err(), context.Cause(ctx))
			if resultErr != nil {
				resultErr = errors.Join(nativeErr, resultErr)
			}
		}()
		s.lease, err = resources.Create(run, s.transaction, "pypi")
		if err != nil {
			return err
		}
		s.resourceCreated = true
		if s.lease.CgroupParent == "" || s.lease.CPU < 0 {
			return errors.New("invalid native probe resource lease")
		}
		if err = s.prepare(run, &pipxPreparedTool{body: body}, frozen); err != nil {
			return err
		}
		if err = s.start(run, stop); err != nil {
			return err
		}
		if !s.registered || s.watch == nil {
			return errors.New("native probe lacks resource admission")
		}
		if err = uv.introduce(run, runner, s.reader.id); err != nil {
			return err
		}
		if err = s.runtime.verifyTarget(run, runner, s.reader.id); err != nil {
			return err
		}
		var output []byte
		output, nativeErr = runner.(boundedCommandRunner).RunBounded(run, "docker", boundaryExecArguments(s.reader.id, boundaryPythonHandoffMode, command...)...)
		t.Logf("pipx_run_preparation_output=%q", output[max(0, len(output)-4096):])
		var terminal *exec.ExitError
		expected := "'" + pipxRunProbeCommand + "' executable script not found in package '" + requirement.Project() + "'."
		if !errors.As(nativeErr, &terminal) || !terminal.Exited() || terminal.ExitCode() <= 0 || !strings.Contains(string(output), expected) {
			return errors.Join(errors.New("native pipx run did not reach the fixed missing-app terminal"), nativeErr)
		}
		_, err = readUVNativeTar(run, runner, s.reader.id, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
			facts, err := scanUVRunCensus(input, "userpath")
			if err != nil {
				return uvNativeOutput{}, err
			}
			body, err := json.Marshal(facts)
			if err != nil {
				return uvNativeOutput{}, err
			}
			t.Logf("pipx_run_preparation_census=%s", body)
			return uvNativeOutput{}, nil
		})
		if err != nil {
			return err
		}
		if err = s.data.verifyReader(run, runner, s.reader.id); err != nil {
			return err
		}
		return s.wheels.verifyReader(run, runner, s.reader.id)
	}()
	resources.mu.Lock()
	creates, registers, closes, cpu := resources.creates, resources.registers, resources.closes, resources.cpu
	resources.mu.Unlock()
	t.Logf("pipx_run_probe original_graph=%d synthetic_lock=true creates=%d registers=%d closes=%d cpu_usec=%d observer_reason=%s selection=NOT_RETURNED Policy=NOT_RUN error=%v", len(frozen.graph.Nodes()), creates, registers, closes, cpu, integrationObserverFaultReason(supervisor), runErr)
	if runErr != nil || creates != 1 || registers != 1 || closes != 1 || cpu == 0 || cpu >= 13_000_000 || integrationObserverFaultReason(supervisor) != "" {
		t.Fatal("native pipx run DATA preparation incomplete", runErr)
	}
}
