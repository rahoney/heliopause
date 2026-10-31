package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type nativeMetadataRunner struct {
	t             *testing.T
	mode, project string
	calls         int
	cancel        func()
	cause         error
}

func (r *nativeMetadataRunner) Run(_ context.Context, project string, args []string) error {
	r.calls++
	r.project = project
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--network none --read-only --cap-drop ALL") || !strings.Contains(joined, "--entrypoint python "+sandbox.PinnedPythonRuntime().ImageReference+" -I -c "+nativePythonRuntimeMetadataScript) || strings.Count(joined, "type=bind,src=") != 1 || strings.Contains(joined, "site-packages") {
		r.t.Fatal("runtime query lost its empty immutable tool scope", joined)
	}
	if r.mode == "run-failure" {
		return r.cause
	}
	marker := filepath.Join(project, "runtime.json")
	body := "{\"version\":\"" + runtimeidentity.PythonVersion + "\",\"magic\":\"2b0e0d0a\",\"cache_tag\":\"cpython-314\"}"
	switch r.mode {
	case "version":
		body = strings.ReplaceAll(body, runtimeidentity.PythonVersion, "3.13.0")
	case "cache-tag":
		body = strings.ReplaceAll(body, "cpython-314", "cpython-313")
	case "magic":
		body = strings.ReplaceAll(body, "2b0e0d0a", "zz")
	case "oversize":
		body = strings.Repeat("x", 257)
	case "symlink":
		return os.Symlink(filepath.Join(filepath.Dir(project), "user-file"), marker)
	case "replaced-workspace":
		if err := os.Rename(project, project+"-original"); err != nil {
			return err
		}
		if err := os.Mkdir(project, 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(project, "user-file"), []byte("preserve"), 0o600)
	}
	if err := os.WriteFile(marker, []byte(body), 0o600); err != nil {
		return err
	}
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

func TestNativePyPIRuntimeMetadataIsBoundedDataAndProtectsForeignCleanup(t *testing.T) {
	for _, mode := range []string{"valid", "version", "cache-tag", "magic", "oversize", "symlink", "run-failure", "early-cancel", "late-cancel", "replaced-workspace"} {
		t.Run(mode, func(t *testing.T) {
			root := realPromotionRoot(t)
			runner := &nativeMetadataRunner{t: t, mode: mode, cause: errors.New("controlled immutable tool failure")}
			enrollment := &pypiNativeEnrollment{intakeRoot: root, runner: runner}
			userPath := filepath.Join(root, "user-file")
			if err := os.WriteFile(userPath, []byte("preserve"), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "early-cancel" {
				cancel()
			}
			if mode == "late-cancel" {
				runner.cancel = cancel
			}
			runtime, err := enrollment.runtimeMetadata(ctx)
			if (mode == "valid") != (err == nil) {
				t.Fatal("runtime metadata acceptance", mode, runtime, err)
			}
			if mode == "valid" && (runtime.magic != "2b0e0d0a" || runtime.cacheTag != "cpython-314") {
				t.Fatal(runtime)
			}
			if mode == "early-cancel" && runner.calls != 0 {
				t.Fatal("cancelled query started trusted execution")
			}
			if (mode == "early-cancel" || mode == "late-cancel") && !errors.Is(err, context.Canceled) {
				t.Fatal("query lost cancellation", err)
			}
			if mode == "run-failure" && !errors.Is(err, runner.cause) {
				t.Fatal("cleanup replaced the first runtime failure", err)
			}
			if mode == "replaced-workspace" {
				if !strings.Contains(err.Error(), "workspace changed") {
					t.Fatal("foreign cleanup uncertainty hidden", err)
				}
				if body, err := os.ReadFile(filepath.Join(runner.project, "user-file")); err != nil || string(body) != "preserve" {
					t.Fatal("foreign replacement workspace removed", err)
				}
			} else if runner.project != "" {
				if _, err := os.Lstat(runner.project); !os.IsNotExist(err) {
					t.Fatal("runtime query workspace leaked", err)
				}
			}
			if body, err := os.ReadFile(userPath); err != nil || string(body) != "preserve" {
				t.Fatal("metadata query touched unrelated files", err)
			}
		})
	}
}
