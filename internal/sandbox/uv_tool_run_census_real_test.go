//go:build linux && realcorpus

package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func uvRunNativeCensus(t *testing.T, ctx context.Context, runner CommandRunner, id string) []uvRunCensusFact {
	t.Helper()
	var facts []uvRunCensusFact
	_, err := readUVNativeTar(ctx, runner, id, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
		var err error
		facts, err = scanUVRunCensus(input, "cowsay")
		return uvNativeOutput{}, err
	})
	if err != nil {
		t.Fatal("native output-volume DATA census", err)
	}
	body, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("uv_run_complete_DATA_census facts=%s", body)
	return facts
}

func TestUVActualPublicToolRunNativeCensus(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_RUN_NATIVE_CENSUS_INTEGRATION") != "1" || os.Getenv("HELOX_UV_NATIVE_RUNTIME") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user native UV complete DATA census")
	}
	frozen := publicUVToolRunInput(t)
	withActualUVNativeRuntimeInput(t, frozen, func(ctx context.Context, runner CommandRunner, id string) {
		args := []string{uvToolGuestPath, "tool", "run", "--isolated", "--from", "cowsay==6.1", "--offline", "--no-index", "--find-links", pipxNativeWheels, "--no-build", "--no-config", "--cache-dir", uvRunCensusCache, "--no-python-downloads", "--no-managed-python", "--python", "/usr/local/bin/python", "--link-mode", "copy", "--no-sources", "/dev/null"}
		output, runErr := runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, args...)...)
		var exited *exec.ExitError
		if !errors.As(runErr, &exited) || exited.ExitCode() == 0 || len(output) != 0 || !strings.Contains(string(exited.Stderr), "/dev/null") || !strings.Contains(string(exited.Stderr), "Permission denied") {
			t.Fatal("native census preparation did not reach fixed EACCES", runErr)
		}
		facts := uvRunNativeCensus(t, ctx, runner, id)
		entry, err := bindUVRunConsoleData(frozen, "cowsay", facts)
		if err != nil || !entry.Valid() {
			t.Fatal("native complete source/console binding", err)
		}
		t.Logf("uv_native_source_console phase=bound original=WHOLE source=INDEPENDENT command=%s metadata_sha256=%s native_record=UNAPPROVED_DATA Policy=NOT_RUN program=NOT_RUN", entry.Command(), entry.Metadata().SHA256)
		environments := 0
		for _, f := range facts {
			if strings.HasPrefix(f.Name, "uv-run-cache/archive-v0/") && strings.HasSuffix(f.Name, "/pyvenv.cfg") {
				environments++
			}
		}
		if environments != 1 {
			t.Fatal("expected one fresh native environment census", environments)
		}
		t.Logf("uv_public_native_census phase=collected source=ACTUAL_PUBLIC_INDEPENDENT whole_original_startup=BOUND readonly_source=SEALED environments=%d facts=%d selected_program=NOT_RUN Evidence=NOT_RUN Policy=NOT_RUN selection_equivalence=NOT_QUALIFIED native_terminal=EACCES", environments, len(facts))
	})
}
