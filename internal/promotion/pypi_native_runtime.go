package promotion

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/sandbox"
)

type pypiNativeRuntime struct{ magic, cacheTag string }

// This tool reads only immutable runtime metadata. No artifact, target Python,
// package, bytecode or native RECORD is mounted in this scope.
const nativePythonRuntimeMetadataScript = `import importlib.util,json,sys
with open('/workspace/runtime.json','x') as output:
 json.dump({'version':'.'.join(map(str,sys.version_info[:3])),'magic':importlib.util.MAGIC_NUMBER.hex(),'cache_tag':sys.implementation.cache_tag},output)`

func (e *pypiNativeEnrollment) runtimeMetadata(ctx context.Context) (result pypiNativeRuntime, resultErr error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := ensureTrustedRoot(e.intakeRoot); err != nil {
		return result, err
	}
	project, err := os.MkdirTemp(e.intakeRoot, ".haa-native-runtime-")
	if err != nil {
		return result, err
	}
	identity, err := os.Lstat(project)
	if err != nil {
		return result, err
	}
	defer func() {
		now, err := os.Lstat(project)
		if err != nil || !now.IsDir() || !os.SameFile(identity, now) || now.Mode() != identity.Mode() || trustedExistingDirectory(project) != nil {
			resultErr = errors.Join(resultErr, errors.New("native runtime metadata workspace changed before cleanup"))
			return
		}
		resultErr = errors.Join(resultErr, os.RemoveAll(project))
	}()
	if strings.Contains(project, ",") {
		return result, errors.New("native runtime metadata path is unsupported by the Docker mount boundary")
	}
	image := sandbox.PinnedPythonRuntime()
	arguments := []string{"run", "--rm", "--pull", "never", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "32", "--memory", "128m", "--cpus", "1", "--user", strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid()), "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=16777216", "--mount", "type=bind,src=" + project + ",dst=/workspace", "--workdir", "/workspace", "--env", "HOME=/tmp", "--entrypoint", "python", image.ImageReference, "-I", "-c", nativePythonRuntimeMetadataScript}
	if err := e.runner.Run(ctx, project, arguments); err != nil {
		return result, errors.Join(err, ctx.Err())
	}
	marker := filepath.Join(project, "runtime.json")
	if _, err := snapshotBoundedPromotionFile(marker, 256); err != nil {
		return result, err
	}
	body, err := readBoundedPromotionFile(marker, 256)
	if err != nil {
		return result, err
	}
	var wire struct {
		Version  string `json:"version"`
		Magic    string `json:"magic"`
		CacheTag string `json:"cache_tag"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&wire) != nil || decoder.Decode(new(any)) != io.EOF || wire.Version != image.PythonVersion || wire.CacheTag != "cpython-314" {
		return result, errors.New("native runtime metadata does not match pinned Python")
	}
	magic, err := hex.DecodeString(wire.Magic)
	if err != nil || len(magic) != 4 || wire.Magic != hex.EncodeToString(magic) {
		return result, errors.New("native runtime metadata has invalid bytecode magic")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return pypiNativeRuntime{wire.Magic, wire.CacheTag}, nil
}
