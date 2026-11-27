package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

type uvReaderRunner struct {
	*recordingRunner
	metadata, archive                   []byte
	commandErr, cleanupErr, metadataErr error
	cancel                              context.CancelFunc
	streams                             int
	filePath                            string
	metadataURL                         string
	archiveURL                          string
	bounded                             bool
}

func (r *uvReaderRunner) Output(ctx context.Context, binary string, args ...string) ([]byte, error) {
	for _, arg := range args {
		if arg == uvDataFetchScript {
			r.metadataURL = args[len(args)-4]
			r.bounded = false
			_, r.bounded = ctx.Deadline()
			return r.metadata, r.metadataErr
		}
	}
	if len(args) > 0 && args[0] == "rm" && r.cleanupErr != nil {
		return nil, r.cleanupErr
	}
	return r.recordingRunner.Output(ctx, binary, args...)
}

func (r *uvReaderRunner) RunOutput(ctx context.Context, writer io.Writer, _ string, args ...string) error {
	r.streams++
	r.archiveURL = args[len(args)-4]
	if output, ok := writer.(*uvArchiveOutput); ok {
		r.filePath = output.file.Name()
	}
	_, err := writer.Write(r.archive)
	if r.cancel != nil {
		r.cancel()
	}
	return errors.Join(err, r.commandErr, ctx.Err())
}

func uvReaderFixture(t *testing.T) (uvNativeOutput, *uvReaderRunner) {
	t.Helper()
	wheel := linuxDynamicProjectWheel(t, "probe", "raise RuntimeError('archive must never execute')\n", nil)
	sum := sha256.Sum256(wheel)
	digest := hex.EncodeToString(sum[:])
	const filename = "probe-1.0-py3-none-any.whl"
	const url = "https://files.pythonhosted.org/packages/" + filename
	lock := fmt.Sprintf("lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=%s'\n[[packages]]\nname='probe'\nversion='1.0'\n[[packages.wheels]]\nurl='%s'\nsize=%d\nhashes={sha256='%s'}\n", PinnedPythonRuntime().PythonVersion, url, len(wheel), digest)
	metadata, err := json.Marshal(map[string]any{
		"info": map[string]any{"name": "probe", "version": "1.0"},
		"urls": []any{map[string]any{"filename": filename, "url": url, "size": len(wheel), "digests": map[string]string{"sha256": digest}, "requires_python": "", "yanked": false, "packagetype": "bdist_wheel"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime := PinnedPythonRuntime()
	runner := &uvReaderRunner{recordingRunner: &recordingRunner{responses: [][]byte{
		[]byte("0123456789abcdef"), []byte("172.30.0.0/24"), []byte("0123456789abcdef"), nil,
		[]byte(runtime.PythonVersion + "\n"), []byte("pip " + runtime.PipVersion + " from pip\n"),
		[]byte(runtime.InterpreterTag + "-" + runtime.ABITag + "-" + runtime.PlatformTag + "\n"), nil,
	}}, metadata: metadata, archive: wheel}
	return uvNativeOutput{lock: []byte(lock), controls: []artifactpypi.UVInstalledWheelControls{{DistInfo: "probe-1.0.dist-info", Metadata: []byte("Name: probe\nVersion: 1.0\n"), Wheel: []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")}}}, runner
}

func TestUVIndependentReaderBindsWholeArchiveAndClearsFailures(t *testing.T) {
	for _, outcome := range []string{"complete", "metadata mismatch", "metadata file hash", "metadata requires python", "metadata size", "metadata command failure", "archive checksum", "short archive", "extra archive", "command failure", "canceled", "cleanup failure", "trace incomplete"} {
		t.Run(outcome, func(t *testing.T) {
			native, runner := uvReaderFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cause := errors.New("untrusted-child-output-must-not-escape")
			trace := &traceReader{}
			switch outcome {
			case "metadata mismatch":
				runner.metadata = bytes.Replace(runner.metadata, []byte(`"version":"1.0"`), []byte(`"version":"9.0"`), 1)
			case "metadata file hash":
				sum := sha256.Sum256(runner.archive)
				runner.metadata = bytes.Replace(runner.metadata, []byte(hex.EncodeToString(sum[:])), []byte(strings.Repeat("0", 64)), 1)
			case "metadata requires python":
				runner.metadata = bytes.Replace(runner.metadata, []byte(`"requires_python":""`), []byte(`"requires_python":">=3.8"`), 1)
			case "metadata size":
				runner.metadata = bytes.Repeat([]byte("x"), uvReleaseMetadataLimit+1)
			case "metadata command failure":
				runner.metadataErr = cause
				runner.cleanupErr = errors.New("second cleanup failure")
			case "archive checksum":
				runner.archive = append([]byte(nil), runner.archive...)
				runner.archive[0] ^= 1
			case "short archive":
				runner.archive = runner.archive[:len(runner.archive)-1]
			case "extra archive":
				runner.archive = append(runner.archive, byte('x'))
			case "command failure":
				runner.commandErr = cause
				runner.cleanupErr = errors.New("second cleanup failure")
			case "canceled":
				runner.cancel = cancel
			case "cleanup failure":
				runner.cleanupErr = errors.New("cleanup failed")
			case "trace incomplete":
				trace.err = errors.New("stream failed")
			}
			resolver, err := NewPyPIResolver(runner, pypiStaticEndpoints{}, &recordingObserver{reader: trace}, availablePythonProbe, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			reference, _ := artifactpypi.ParseReference("probe@1.0")
			result, err := resolver.bindUVNativeSelection(ctx, reference, native)
			if outcome == "complete" {
				if err != nil || len(result.Graph().Nodes()) != 1 || !strings.HasPrefix(result.RuntimeIdentity(), "uv:") || strings.Contains(result.RuntimeIdentity(), ";pip:") {
					t.Fatal("independent native binding failed", err)
				}
			} else if err == nil || len(result.Graph().Nodes()) != 0 {
				t.Fatal("failed reader retained graph", err)
			}
			if (outcome == "command failure" || outcome == "metadata command failure") && (!errors.Is(err, cause) || strings.Contains(err.Error(), cause.Error())) {
				t.Fatal("primary cause lost or child output rendered", err)
			}
			if outcome == "canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation cause lost", err)
			}
			if (outcome == "metadata mismatch" || outcome == "metadata file hash" || outcome == "metadata size" || outcome == "metadata command failure") && runner.streams != 0 {
				t.Fatal("archive read before independent metadata matched")
			}
			if !runner.bounded || runner.metadataURL != "https://pypi.org/pypi/probe/1.0/json" {
				t.Fatal("reader endpoint/deadline changed", runner.metadataURL)
			}
			if runner.streams != 0 && runner.archiveURL != "https://files.pythonhosted.org/packages/probe-1.0-py3-none-any.whl" {
				t.Fatal("archive endpoint changed")
			}
			if runner.filePath != "" {
				if _, err := os.Stat(runner.filePath); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("completed archive was retained", err)
				}
			}
		})
	}
}

func TestUVIndependentReaderRejectsSelectionBeforeNetworkMutation(t *testing.T) {
	for _, outcome := range []string{"invalid lock", "missing controls", "stream unavailable", "canceled"} {
		t.Run(outcome, func(t *testing.T) {
			native, runner := uvReaderFixture(t)
			var commands CommandRunner = runner
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch outcome {
			case "invalid lock":
				native.lock = []byte("invalid")
			case "missing controls":
				native.controls = nil
			case "stream unavailable":
				commands = admissionAwareRunner(runner.recordingRunner)
			case "canceled":
				cancel()
			}
			resolver, err := NewPyPIResolver(commands, pypiStaticEndpoints{}, &recordingObserver{reader: &traceReader{}}, availablePythonProbe, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			reference, _ := artifactpypi.ParseReference("probe@1.0")
			result, err := resolver.bindUVNativeSelection(ctx, reference, native)
			if err == nil || len(result.Graph().Nodes()) != 0 || len(runner.calls) != 0 {
				t.Fatal("invalid reader request mutated network or returned graph")
			}
		})
	}
}

func TestUVArchiveWriterKeepsStickyBoundAcrossCopyFastPaths(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "archive-")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	writer := &uvArchiveOutput{file: file, remaining: 3}
	if _, err := io.Copy(writer, strings.NewReader("four")); err == nil || !writer.failed {
		t.Fatal("copy bypassed archive bound")
	}
	if _, err := writer.Write([]byte("a")); err == nil || !errors.Is(err, writer.failure) {
		t.Fatal("bound failure was not sticky")
	}
	if info, err := file.Stat(); err != nil || info.Size() != 0 {
		t.Fatal("overflow bytes were stored", err)
	}
}
