//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// This cache exists only in the fresh owned native output volume.
const uvRunCensusCache = "/haa-site/uv-run-cache"

// Diagnostic DATA does not grant source identity, admission or Policy approval.
type uvRunCensusFact struct {
	Name         string
	Kind         byte
	UID, GID     int
	Mode, Size   int64
	Link, SHA256 string
	Body         []byte `json:",omitempty"`
}

func scanUVRunCensus(input io.Reader) ([]uvRunCensusFact, error) {
	bounded := &io.LimitedReader{R: input, N: 4<<20 + 64<<10}
	archive := tar.NewReader(bounded)
	var facts []uvRunCensusFact
	seen := map[string]bool{}
	var expanded, controlBytes int64
	for {
		h, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("native cache census tar: %w", err)
		}
		name := strings.TrimSuffix(strings.TrimPrefix(h.Name, "./"), "/")
		if name != "" && name != "." && !validClosureDestination(name) || seen[name] || h.Size < 0 || h.Size > (4<<20)-expanded || len(facts) >= 4096 {
			return nil, errors.New("native cache census path/duplicate/count/size bound")
		}
		seen[name] = true
		expanded += h.Size
		row := uvRunCensusFact{Name: name, Kind: h.Typeflag, UID: h.Uid, GID: h.Gid, Mode: h.Mode, Size: h.Size, Link: h.Linkname}
		if h.Typeflag == tar.TypeReg || h.Typeflag == 0 {
			digest := sha256.New()
			capture := strings.HasSuffix(name, "/METADATA") || strings.HasSuffix(name, "/WHEEL") || strings.HasSuffix(name, "/pyvenv.cfg") || strings.HasSuffix(name, "/bin/cowsay")
			if capture {
				if h.Size > 64<<10 || h.Size > (256<<10)-controlBytes {
					return nil, errors.New("native census control bytes exceed diagnostic bound")
				}
				row.Body, err = io.ReadAll(io.TeeReader(io.LimitReader(archive, h.Size), digest))
				if err != nil || int64(len(row.Body)) != h.Size {
					return nil, errors.New("native census control truncated")
				}
				controlBytes += h.Size
			} else if n, err := io.CopyN(digest, archive, h.Size); err != nil || n != h.Size {
				return nil, errors.New("native census regular file truncated")
			}
			row.SHA256 = hex.EncodeToString(digest.Sum(nil))
		}
		facts = append(facts, row)
	}
	if err := drainUVTarPadding(bounded); err != nil {
		return nil, err
	}
	if bounded.N == 0 || len(facts) == 0 {
		return nil, errors.New("native census absent or transport bound exhausted")
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Name < facts[j].Name })
	return facts, nil
}

func uvRunNativeCensus(t *testing.T, ctx context.Context, runner CommandRunner, id string) []uvRunCensusFact {
	t.Helper()
	var facts []uvRunCensusFact
	_, err := readUVNativeTar(ctx, runner, id, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
		var err error
		facts, err = scanUVRunCensus(input)
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
