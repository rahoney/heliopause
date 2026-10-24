package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const uvReleaseMetadataLimit = 4 << 20

// bindUVNativeSelection independently reads registry metadata and whole archive
// bytes inside the existing Python resolver boundary. Python never resolves or
// installs the UV selection, and no artifact is imported on either side.
func (r *PyPIResolver) bindUVNativeSelection(ctx context.Context, reference domain.ArtifactReference, native uvNativeOutput) (resolution domain.DependencyResolution, resultErr error) {
	if r == nil || ctx == nil || r.profile.Source() != artifactpypi.PublicPyPIProfile().Source() || reference.Source() != r.profile.Source() {
		return domain.DependencyResolution{}, errors.New("uv independent reader requires public PyPI")
	}
	if err := ctx.Err(); err != nil {
		return domain.DependencyResolution{}, err
	}
	runtime := PinnedPythonRuntime()
	lock, err := artifactpypi.ParseUVLock(reference, native.lock, runtime.PythonVersion)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	selected, err := lock.SelectInstalledWheelControls(native.controls, target)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	err = r.withUVArchiveInputs(ctx, selected, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		graph, err := lock.BindWheels(inputs, pages, target, policy.WheelLimits())
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		sum := sha256.Sum256(native.lock)
		digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		if err != nil {
			return err
		}
		resolution, err = domain.NewDependencyResolution(graph, UVResolverRuntimeIdentity(), digest)
		return err
	})
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return resolution, nil
}

// Archive bytes and metadata stay inside one bounded workspace/session until
// the actual consumer finishes its independent binding. No solver is run here.
func (r *PyPIResolver) withUVArchiveInputs(ctx context.Context, selected []artifactpypi.UVLockedWheel, operation func([]artifactpypi.UVWheelInput, []artifactpypi.SimpleProject) error) (resultErr error) {
	if r == nil || r.runner == nil || ctx == nil || operation == nil || r.profile.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return errors.New("uv independent archive reader is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(selected) == 0 {
		if err := operation([]artifactpypi.UVWheelInput{}, []artifactpypi.SimpleProject{}); err != nil {
			return err
		}
		return ctx.Err()
	}
	runtime := PinnedPythonRuntime()
	underlying := r.runner
	if admitted, ok := underlying.(*admissionAwareCommandRunner); ok {
		underlying = admitted.runner
	}
	if _, ok := underlying.(streamingDockerOutput); !ok {
		return errors.New("uv archive stream transport is unavailable")
	}
	stream, ok := r.runner.(streamingDockerOutput)
	if !ok {
		return errors.New("uv admitted archive stream transport is unavailable")
	}
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	var total int64
	for _, wheel := range selected {
		if wheel.Size <= 0 || wheel.Size > policy.MaxGraphCompressed()-total {
			return errors.New("uv selected archive budget exceeds bounds")
		}
		total += wheel.Size
	}
	workspace, err := os.MkdirTemp("", "haa-uv-reader-")
	if err != nil {
		return errors.New("uv private archive workspace unavailable")
	}
	var archives []*os.File
	defer func() {
		var cleanupErr error
		for _, file := range archives {
			cleanupErr = errors.Join(cleanupErr, file.Close())
		}
		cleanupErr = errors.Join(cleanupErr, os.RemoveAll(workspace))
		if cleanupErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("uv archive workspace cleanup failed: %w", cleanupErr))
		}
	}()
	if err := artifactpypi.CheckTemporaryDisk(workspace, policy); err != nil {
		return err
	}
	return r.withPythonResolverSourceSession(ctx, r.profile.Source(), func(ctx context.Context, container string, actual PythonRuntime) error {
		if actual != runtime {
			return errors.New("uv independent reader runtime differs from native target")
		}
		ctx, cancel := context.WithTimeout(ctx, pypiResolverTimeout)
		defer cancel()
		var pages []artifactpypi.SimpleProject
		var inputs []artifactpypi.UVWheelInput
		for _, wheel := range selected {
			project, version, _, _, _, err := artifactpypi.ParseWheelFilename(wheel.Filename)
			if err != nil {
				return err
			}
			metadataURL, err := artifactpypi.UVReleaseMetadataURL(project, version)
			if err != nil {
				return err
			}
			body, err := r.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, "python", "-I", "-c", uvDataFetchScript,
				metadataURL, "json", strconv.Itoa(uvReleaseMetadataLimit), "-1")...)
			if err != nil {
				return uvReaderFailure{"read uv independent release metadata failed", err}
			}
			page, err := artifactpypi.ParseUVReleaseMetadata(project, version, body)
			if err != nil {
				return err
			}
			declared := false
			for _, file := range page.Files() {
				if file.URL() == wheel.URL && file.Filename() == wheel.Filename && file.SHA256() == wheel.SHA256 && file.Size() == uint64(wheel.Size) && !file.Yanked() {
					declared = true
				}
			}
			if !declared {
				return errors.New("uv selected archive differs from independent release metadata")
			}
			pages = append(pages, page)
			file, err := os.CreateTemp(workspace, "archive-")
			if err != nil {
				return errors.New("uv private archive file unavailable")
			}
			archives = append(archives, file)
			output := &uvArchiveOutput{file: file, remaining: wheel.Size}
			commandErr := stream.RunOutput(ctx, output, "docker", boundaryExecArguments(container, boundaryLaunchMode, "python", "-I", "-c", uvDataFetchScript,
				wheel.URL, "archive", strconv.FormatInt(wheel.Size, 10), strconv.FormatInt(wheel.Size, 10))...)
			if err := errors.Join(output.failure, commandErr, ctx.Err()); err != nil {
				return uvReaderFailure{"read uv selected archive failed", err}
			}
			if output.remaining != 0 || output.failed {
				return errors.New("uv selected archive stream size differs from lock")
			}
			inputs = append(inputs, artifactpypi.UVWheelInput{URL: wheel.URL, Reader: file, Size: wheel.Size})
		}
		if err := operation(inputs, pages); err != nil {
			return err
		}
		return ctx.Err()
	})
}

// The cause remains inspectable without rendering untrusted subprocess text.
type uvReaderFailure struct {
	step  string
	cause error
}

func (e uvReaderFailure) Error() string { return e.step }
func (e uvReaderFailure) Unwrap() error { return e.cause }

// Only Write is exposed: io.Copy fast paths cannot bypass the archive bound.
type uvArchiveOutput struct {
	file      *os.File
	remaining int64
	failed    bool
	failure   error
}

func (w *uvArchiveOutput) Write(body []byte) (int, error) {
	if w == nil || w.file == nil || w.failed || int64(len(body)) > w.remaining {
		err := errors.New("uv archive stream exceeds exact size bound")
		if w != nil {
			w.failed = true
			if w.failure == nil {
				w.failure = err
			}
			err = w.failure
		}
		return 0, err
	}
	n, err := w.file.Write(body)
	w.remaining -= int64(n)
	if err == nil && n != len(body) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.failed = true
		w.failure = err
	}
	return n, err
}

// URLs and sizes are derived from the parsed native lock or exact-release URL
// builder. Redirects, proxies and interpreter/user configuration are not inputs.
const uvDataFetchScript = "import sys,urllib.request\n" +
	"class NoRedirect(urllib.request.HTTPRedirectHandler):\n def redirect_request(self, req, fp, code, msg, headers, newurl): return None\n" +
	"url,kind=sys.argv[1:3]\nlimit,expected=map(int,sys.argv[3:5])\n" +
	"if limit < 1 or kind not in ('json','archive') or (kind == 'archive' and expected != limit): raise SystemExit(1)\n" +
	"request=urllib.request.Request(url,headers={'Accept':'application/json' if kind == 'json' else 'application/octet-stream'})\n" +
	"response=urllib.request.build_opener(urllib.request.ProxyHandler({}),NoRedirect).open(request,timeout=15)\n" +
	"if response.status != 200 or response.geturl() != url: raise SystemExit(1)\n" +
	"if kind == 'json' and response.headers.get_content_type().lower() != 'application/json': raise SystemExit(1)\n" +
	"length=response.headers.get('Content-Length')\n" +
	"if length is not None and (not length.isdigit() or int(length) < 1 or int(length) > limit or (expected >= 0 and int(length) != expected)): raise SystemExit(1)\n" +
	"total=0\nwhile True:\n chunk=response.read(min(65536,limit-total+1))\n if not chunk: break\n total+=len(chunk)\n if total > limit: raise SystemExit(1)\n sys.stdout.buffer.write(chunk)\n" +
	"if total < 1 or (expected >= 0 and total != expected): raise SystemExit(1)\n"
