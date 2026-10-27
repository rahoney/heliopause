//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pipxDiagnosticWheel struct {
	filename, directory string
	body                []byte
}
type pipxDiagnosticInput struct {
	lock    artifactpypi.UVLock
	inputs  []artifactpypi.UVWheelInput
	pages   []artifactpypi.SimpleProject
	wheels  []pipxDiagnosticWheel
	version string
}

// The actual official archive fixture has synthetic lock provenance. It is
// static input binding, never evidence that UV resolved these versions live.
func pipxDiagnosticFrozenInput(t *testing.T) pipxDiagnosticInput {
	t.Helper()
	directory := os.Getenv("HELOX_PIPX_QUERY_SUPPLY_ROOT")
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		t.Fatal("explicit absolute frozen supply required")
	}
	var owner struct {
		Pipx struct {
			Inputs []struct{ Path, SHA256 string } `json:"inputs"`
		} `json:"pipx_tool"`
	}
	body, err := os.ReadFile("../../scripts/runtimes.lock.json")
	if err != nil || json.Unmarshal(body, &owner) != nil || len(owner.Pipx.Inputs) != 42 {
		t.Fatal("canonical supply owner unavailable", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	originals := map[string][]byte{}
	for _, input := range owner.Pipx.Inputs {
		body, err := root.ReadFile(input.Path)
		sum := sha256.Sum256(body)
		if err != nil || hex.EncodeToString(sum[:]) != input.SHA256 {
			t.Fatal("original fixed input changed", input.Path, err)
		}
		if strings.HasPrefix(input.Path, "userpath/") || strings.HasPrefix(input.Path, "click/") {
			originals[input.Path] = body
		}
	}
	result := pipxDiagnosticInput{}
	lockBody := "lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=" + runtimeidentity.PythonVersion + "'\n"
	for _, input := range owner.Pipx.Inputs {
		pieces := strings.Split(input.Path, "/")
		if len(pieces) != 2 || pieces[0] != "userpath" && pieces[0] != "click" || !strings.HasSuffix(pieces[1], ".whl") {
			continue
		}
		project, version, _, _, _, err := artifactpypi.ParseWheelFilename(pieces[1])
		if err != nil {
			t.Fatal(err)
		}
		page, err := artifactpypi.ParseUVReleaseMetadata(project, version, originals[project+"/metadata.json"])
		if err != nil {
			t.Fatal(err)
		}
		archive := originals[input.Path]
		selected := ""
		for _, file := range page.Files() {
			if file.Filename() == pieces[1] && file.SHA256() == input.SHA256 && file.Size() == uint64(len(archive)) && !file.Yanked() {
				if selected != "" {
					t.Fatal("ambiguous original archive")
				}
				selected = file.URL()
			}
		}
		if selected == "" {
			t.Fatal("original archive absent from independent source")
		}
		result.inputs = append(result.inputs, artifactpypi.UVWheelInput{URL: selected, Reader: bytes.NewReader(archive), Size: int64(len(archive))})
		result.pages = append(result.pages, page)
		result.wheels = append(result.wheels, pipxDiagnosticWheel{filename: pieces[1], directory: strings.ReplaceAll(project, "-", "_") + "-" + version + ".dist-info", body: archive})
		if project == "userpath" {
			result.version = version
		}
		lockBody += fmt.Sprintf("[[packages]]\nname=%q\nversion=%q\nwheels=[{url=%q,size=%d,hashes={sha256=%q}}]\n", project, version, selected, len(archive), input.SHA256)
	}
	reference, err := artifactpypi.ParseReference("userpath@" + result.version)
	if err != nil || len(result.wheels) != 2 {
		t.Fatal("frozen original pair unavailable", err)
	}
	result.lock, err = artifactpypi.ParseUVLock(reference, []byte(lockBody), runtimeidentity.PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := result.lock.BindPipxQueryWheels(result.inputs, result.pages, pipxDiagnosticTarget(), artifactpypi.DefaultWheelLimits())
	if err != nil || len(graph.Nodes()) != 2 {
		t.Fatal("pre-query whole original gate refused", err)
	}
	return result
}

func pipxDiagnosticTarget() artifactpypi.WheelTarget {
	return artifactpypi.WheelTarget{Python: runtimeidentity.PythonInterpreterTag, ABI: runtimeidentity.PythonABITag, Platform: runtimeidentity.PythonPlatformTag}
}

// Runc checks the actual fixed loader/argv/offline native behavior only. It
// supplies neither gVisor attribution nor inspection/Policy/Host approval.
func TestPipxActualOfflineNativeProgram(t *testing.T) {
	if os.Getenv("HELOX_PIPX_NATIVE_PROGRAM") != "1" {
		t.Skip("explicit runc native program diagnostic required")
	}
	if os.Geteuid() != 1000 {
		t.Fatal("requires declared ordinary UID1000 diagnostic caller")
	}
	input := pipxDiagnosticFrozenInput(t) // Before any native interpreter/Docker.
	frozen, err := freezePipxNativeInput(context.Background(), input.lock, input.inputs, input.pages)
	if err != nil {
		t.Fatal(err)
	}
	zipBody, err := os.ReadFile(os.Getenv("HELOX_PIPX_RUNTIME_ZIP"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifactpypi.ParsePipxRuntimeZIP(zipBody); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	runner, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	type keeper struct {
		name, owner, id string
		closed          bool
	}
	var keepers []*keeper
	var data *pipxDataVolume
	var wheelData *pipxWheelVolume
	var uv nativeRuntimeVolume
	var output closureVolume
	remove := func(cleanup context.Context, k *keeper) error {
		if k.closed {
			return nil
		}
		if err := removeOwnedUVResolverContainer(cleanup, runner, k.name, k.owner, k.id); err != nil {
			return err
		}
		k.closed = true
		return nil
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		for i := len(keepers) - 1; i >= 0; i-- {
			if err := remove(cleanup, keepers[i]); err != nil {
				t.Error("owned keeper cleanup", err)
			}
		}
		if output.name != "" {
			if err := closeUVOutputVolume(cleanup, runner, &output); err != nil {
				t.Error("output cleanup", err)
			}
		}
		if uv.name != "" {
			if err := uv.close(cleanup, runner); err != nil {
				t.Error("UV cleanup", err)
			}
		}
		if wheelData != nil {
			if err := wheelData.close(cleanup, runner); err != nil {
				t.Error("wheel DATA cleanup", err)
			}
		}
		if data != nil {
			if err := data.close(cleanup, runner); err != nil {
				t.Error("DATA cleanup", err)
			}
		}
		if err := runner.Close(); err != nil {
			t.Error(err)
		}
	}()
	tool, err := prepareUVTool(ctx, runner)
	if err != nil {
		t.Fatal(err)
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	data, err = createPipxDataVolume(ctx, runner, session.String())
	if err != nil {
		t.Fatal(err)
	}
	wheelData, err = createPipxWheelVolume(ctx, runner, session.String(), frozen)
	if err != nil {
		t.Fatal(err)
	}
	create := func(role, user string, mounts []string, native bool) *keeper {
		t.Helper()
		id, err := domain.NewSandboxSessionID()
		if err != nil {
			t.Fatal(err)
		}
		k := &keeper{owner: id.String(), name: "haa-uv-resolver-" + id.String()}
		keepers = append(keepers, k)
		args := []string{"create", "--pull", "never", "--name", k.name, "--label", uvResolverOwnerLabel + "=" + k.owner, "--runtime", "runc", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1", "--ulimit", "cpu=30:30", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m,uid=1000,gid=1000,mode=0700"}
		args = append(args, mounts...)
		if native {
			// Empty Config.User preserves both root runtime and UID1000 data
			// in Docker stdin archives. PID1 immediately drops real authority.
			if user != "" {
				t.Fatal("native diagnostic must retain the image's empty Config.User")
			}
			args = append(args, "--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP")
		} else {
			args = append(args, "--user", user)
		}
		if native {
			args = append(args, pipxNativeEnvironmentArguments()...)
		} else {
			args = append(args, isolatedContainerEnvironmentArguments()...)
		}
		if native {
			args = append(args, "--entrypoint", "/bin/sh", pythonImageReference, "-ceu", boundaryContainerCommand())
		} else {
			args = append(args, "--entrypoint", "/bin/sleep", pythonImageReference, "300")
		}
		body, err := runner.Output(ctx, "docker", args...)
		if err != nil || !exactObservationContainerID(strings.TrimSpace(string(body))) {
			t.Fatal("keeper CREATE identity unavailable", err)
		}
		k.id = strings.TrimSpace(string(body))
		if err := verifyOwnedUVResolverContainer(ctx, runner, k.name, k.owner, k.id); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Output(ctx, "docker", "start", k.id); err != nil {
			t.Fatal(err)
		}
		t.Logf("owned_runc_keeper role=%s id=%s", role, k.id)
		return k
	}
	rw, err := data.preparationMount()
	if err != nil {
		t.Fatal(err)
	}
	wheelRW, err := wheelData.preparationMount()
	if err != nil {
		t.Fatal(err)
	}
	writer := create("writer", "0:0", []string{"--mount", rw, "--mount", wheelRW}, false)
	if err := data.seed(ctx, runner, &pipxPreparedTool{body: zipBody}, writer.id); err != nil {
		t.Fatal(err)
	}
	if err := wheelData.seed(ctx, runner, writer.id); err != nil {
		t.Fatal(err)
	}
	ro, err := data.readerMount()
	if err != nil {
		t.Fatal(err)
	}
	wheelRO, err := wheelData.readerMount()
	if err != nil {
		t.Fatal(err)
	}
	anchor := create("anchor", "1000:1000", []string{"--mount", ro, "--mount", wheelRO}, false)
	if err := data.bindAnchor(ctx, runner, anchor.id); err != nil {
		t.Fatal(err)
	}
	if err := wheelData.bindAnchor(ctx, runner, anchor.id); err != nil {
		t.Fatal(err)
	}
	if err := remove(ctx, writer); err != nil {
		t.Fatal(err)
	}
	if err := data.seal(ctx, runner); err != nil {
		t.Fatal(err)
	}
	if err := wheelData.seal(ctx, runner); err != nil {
		t.Fatal(err)
	}
	uv, err = createUVRuntimeVolume(ctx, runner)
	if err != nil {
		t.Fatal(err)
	}
	output, err = createClosureVolume(ctx, runner, session.String()+"-native-output", runtimeidentity.PipxToolSHA256, artifactpypi.PublicPyPIProfile().ResourcePolicy().PromotionTmpfs())
	if err != nil {
		t.Fatal(err)
	}
	uvMount, err := uv.mountArgument()
	if err != nil {
		t.Fatal(err)
	}
	outputMount, err := output.mountArgument(false)
	if err != nil {
		t.Fatal(err)
	}
	reader := create("native-reader", "", []string{"--mount", ro, "--mount", wheelRO, "--mount", uvMount, "--mount", outputMount}, true)
	// This runc-only framing value is controller-created and confers neither
	// observer admission nor completion authority. Use the canonical helper
	// solely to remove the Docker exec capability set before every target.
	nonceSum := sha256.Sum256([]byte(session.String()))
	nonce := hex.EncodeToString(nonceSum[:])
	runcExec := func(command ...string) []string {
		return boundaryExecArguments(reader.id, boundaryLaunchMode, append([]string{nonce}, command...)...)
	}
	ready := false
	for attempt := 0; attempt < 20; attempt++ {
		if _, err := runner.Output(ctx, "docker", runcExec("/bin/sh", "-ceu", boundaryReadinessScript)...); err == nil {
			ready = true
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	if !ready {
		t.Fatal("canonical diagnostic boundary readiness unavailable")
	}
	t.Log("native PID1 and exec readiness: UID1000 empty groups zero caps NNP")
	if err := data.verifyReader(ctx, runner, reader.id); err != nil {
		t.Fatal(err)
	}
	if err := wheelData.verifyReader(ctx, runner, reader.id); err != nil {
		t.Fatal(err)
	}
	if err := uv.verifyTarget(ctx, runner, reader.id); err != nil {
		t.Fatal(err)
	}
	if err := output.verifyAttachments(ctx, runner, map[string]bool{reader.id: false}); err != nil {
		t.Fatal(err)
	}
	if err := tool.introduce(ctx, runner, reader.id); err != nil {
		t.Fatal(err)
	}
	// Poison only the private guest HOME. The same selected UV must reject
	// this file without no-config, before pipx's native venv creation succeeds
	// with the fixed environment. No caller/Host configuration is mounted.
	prepareConfig := "from pathlib import Path; p = Path('/tmp/.config/uv/uv.toml'); p.parent.mkdir(parents=True, exist_ok=True); p.write_bytes(b'[invalid TOML configuration\\n')"
	if _, err := runner.Output(ctx, "docker", runcExec("/usr/local/bin/python", "-I", "-S", "-B", "-c", prepareConfig)...); err != nil {
		t.Fatal("prepare fixed guest config data", err)
	}
	negative, configErr := runner.RunBounded(ctx, "docker", runcExec("/usr/bin/env", "-u", "UV_NO_CONFIG", uvToolGuestPath, "venv", "--python", "/usr/local/bin/python", "--no-python-downloads", "/tmp/haa-config-negative")...)
	if configErr == nil {
		t.Fatal("malformed guest UV config was not observed by the negative control")
	}
	configDiagnostic := strings.ToLower(string(negative) + " " + configErr.Error())
	if !strings.Contains(configDiagnostic, "uv.toml") || !strings.Contains(configDiagnostic, "parse") {
		t.Fatal("negative failed outside the intended UV config parser", configErr)
	}
	t.Log("negative native UV venv rejected poisoned guest uv.toml without UV_NO_CONFIG")
	// A foreign RW attachment invalidates the complete reader census. Restoring
	// the exact owned inventory must recover before native execution.
	foreign := create("foreign-wheel-writer", "0:0", []string{"--mount", wheelRW}, false)
	if err := wheelData.verifyReader(ctx, runner, reader.id); err == nil {
		t.Fatal("foreign RW wheel attachment accepted")
	}
	if err := remove(ctx, foreign); err != nil {
		t.Fatal(err)
	}
	if err := wheelData.verifyReader(ctx, runner, reader.id); err != nil {
		t.Fatal(err)
	}
	// Explicit root touch distinguishes filesystem EROFS from UID permission
	// denial. It executes no wheel/Python code and grants no native authority.
	touch, touchErr := runner.RunBounded(ctx, "docker", "exec", "--user", "0:0", anchor.id, "/usr/bin/touch", pipxNativeWheels+"/forbidden")
	if touchErr == nil || !strings.Contains(strings.ToLower(string(touch)), "read-only file system") {
		t.Fatal("wheel RO root write did not fail EROFS", touchErr)
	}
	if err := wheelData.verifyReader(ctx, runner, reader.id); err != nil {
		t.Fatal(err)
	}
	t.Log("wheel original RO root write EROFS; foreign RW rejected then owned removal recovered")
	execute := func(operation pipxNativeOperation, request artifactpypi.UpgradeRequirement) []byte {
		t.Helper()
		args, err := pipxNativeArguments(operation, request)
		if err != nil {
			t.Fatal(err)
		}
		args = runcExec(args...)
		body, err := runner.Output(ctx, "docker", args...)
		if err != nil {
			t.Fatalf("native operation %d: %s: %v", operation, body, err)
		}
		return body
	}
	version := execute(pipxNativeVersion, artifactpypi.UpgradeRequirement{})
	if strings.TrimSpace(string(version)) != runtimeidentity.PipxVersion {
		t.Fatal("native pipx version differs", string(version))
	}
	install, err := artifactpypi.ParseUpgradeRequirement("userpath==" + input.version)
	if err != nil {
		t.Fatal(err)
	}
	execute(pipxNativeInstall, install)
	check := func(phase string) {
		t.Helper()
		environment := pipxNativeHome + "/venvs/userpath"
		body := pipxDiagnosticFile(t, ctx, runner, reader.id, environment+"/pipx_metadata.json")
		receipt, err := artifactpypi.ParsePipxToolReceipt(body)
		if err != nil {
			t.Fatal("actual native receipt", err)
		}
		if receipt.Version() != input.version || receipt.Requirement().Project() != "userpath" {
			t.Fatal("native root/version differs")
		}
		links := pipxDiagnosticLinks(t, ctx, runner, reader.id)
		complete, err := readPipxNativeOutput(ctx, runner, reader.id, "userpath", frozen)
		if err != nil || !bytes.Equal(complete.receipt.Body(), receipt.Body()) || !reflect.DeepEqual(complete.links, links) || len(complete.controls) != 2 || len(complete.graph.Nodes()) != 2 {
			t.Fatal("complete native output differs from named original controls", err)
		}
		if err := receipt.ValidateExposedCommands(environment, pipxNativeBin, "/usr/local/bin/python", links); err != nil {
			t.Fatal(err)
		}
		graph, err := receipt.BindWheels(input.lock, input.inputs, input.pages, pipxDiagnosticTarget(), artifactpypi.DefaultWheelLimits())
		if err != nil || len(graph.Nodes()) != 2 {
			t.Fatal("native receipt differs from authentic original graph", err)
		}
		var metadata []artifactpypi.InstalledMetadataFile
		var wheels []artifactpypi.UVInstalledWheelControls
		for _, w := range input.wheels {
			base := environment + "/lib/python3.14/site-packages/" + w.directory
			m := pipxDiagnosticFile(t, ctx, runner, reader.id, base+"/METADATA")
			wheel := pipxDiagnosticFile(t, ctx, runner, reader.id, base+"/WHEEL")
			found := false
			for _, captured := range complete.controls {
				if captured.DistInfo == w.directory {
					found = bytes.Equal(captured.Metadata, m) && bytes.Equal(captured.Wheel, wheel)
				}
			}
			if !found {
				t.Fatal("complete native control bytes differ from separate capture")
			}
			metadata = append(metadata, artifactpypi.InstalledMetadataFile{Directory: w.directory, Body: m})
			wheels = append(wheels, artifactpypi.UVInstalledWheelControls{DistInfo: w.directory, Metadata: m, Wheel: wheel})
		}
		original, err := artifactpypi.NewPipxToolUpgradeInput(receipt, metadata, runtimeidentity.PythonVersion)
		if err != nil || original.Request().Requirement() != "userpath" {
			t.Fatal("native original upgrade input", err)
		}
		if _, err := original.ReplayWheelControls(wheels, pipxDiagnosticTarget()); err != nil {
			t.Fatal(err)
		}
		if _, err := receipt.WithPipxLayout("/explicit/held/userpath", "/explicit/commands", "/explicit/runtime/python"); err != nil {
			t.Fatal(err)
		}
		pipxDiagnosticCompleteCensus(t, ctx, runner, reader.id, phase)
		t.Logf("native_%s version=%s original_graph=%d exposed=%d request=%s; no entrypoint execution/Policy", phase, receipt.Version(), len(graph.Nodes()), len(links), receipt.Requirement().Requirement())
	}
	check("install")
	upgrade, err := artifactpypi.ParseUpgradeRequirement("userpath")
	if err != nil {
		t.Fatal(err)
	}
	execute(pipxNativeUpgrade, upgrade)
	check("noop_upgrade")
	if err := data.verifyReader(ctx, runner, reader.id); err != nil {
		t.Fatal(err)
	}
	if err := wheelData.verifyReader(ctx, runner, reader.id); err != nil {
		t.Fatal(err)
	}
	if err := uv.verifyTarget(ctx, runner, reader.id); err != nil {
		t.Fatal(err)
	}
	if err := output.verifyAttachments(ctx, runner, map[string]bool{reader.id: false}); err != nil {
		t.Fatal(err)
	}
	t.Log("fixed_native_program_offline_diagnostic_completed; runc only, installed current/gVisor/Policy/publication remain")
}

func pipxDiagnosticTar(t *testing.T, ctx context.Context, runner *hosttool.Executor, id, source string) (*tar.Reader, *io.LimitedReader) {
	t.Helper()
	captured := nativeToolArchiveOutput{limit: 4<<20 + 64<<10}
	if err := runner.RunOutput(ctx, &captured, "docker", "cp", id+":"+source, "-"); err != nil {
		t.Fatal(err)
	}
	reader := &io.LimitedReader{R: bytes.NewReader(captured.body), N: captured.limit}
	return tar.NewReader(reader), reader
}

func pipxDiagnosticFile(t *testing.T, ctx context.Context, runner *hosttool.Executor, id, source string) []byte {
	t.Helper()
	archive, reader := pipxDiagnosticTar(t, ctx, runner, id, source)
	header, err := archive.Next()
	if err != nil || header.Name != path.Base(source) || header.Typeflag != tar.TypeReg && header.Typeflag != 0 || header.Uid != 1000 || header.Gid != 1000 || header.Mode&0022 != 0 || header.Size <= 0 || header.Size > 4<<20 {
		t.Fatal("native control tar identity", source, err)
	}
	body, err := io.ReadAll(io.LimitReader(archive, header.Size))
	if err != nil || int64(len(body)) != header.Size {
		t.Fatal("native control truncated", err)
	}
	if _, err := archive.Next(); err != io.EOF {
		t.Fatal("native control has extra member", err)
	}
	if err := drainUVTarPadding(reader); err != nil {
		t.Fatal(err)
	}
	return body
}

func pipxDiagnosticLinks(t *testing.T, ctx context.Context, runner *hosttool.Executor, id string) map[string]string {
	t.Helper()
	archive, reader := pipxDiagnosticTar(t, ctx, runner, id, pipxNativeBin+"/.")
	links := map[string]string{}
	for {
		h, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(strings.TrimPrefix(h.Name, "./"), "/")
		if h.Typeflag == tar.TypeDir && (name == "" || name == ".") {
			continue
		}
		if h.Typeflag != tar.TypeSymlink || h.Uid != 1000 || h.Gid != 1000 || h.Mode != 0777 || h.Size != 0 || name != path.Base(name) || links[pipxNativeBin+"/"+name] != "" {
			t.Fatal("native exposed link inventory invalid", h.Name)
		}
		links[pipxNativeBin+"/"+name] = h.Linkname
	}
	if err := drainUVTarPadding(reader); err != nil {
		t.Fatal(err)
	}
	return links
}

// This bounded diagnostic records data facts only. It is neither a product
// allowlist nor an inspection completion/admission signal.
func pipxDiagnosticCompleteCensus(t *testing.T, ctx context.Context, runner CommandRunner, container, phase string) {
	t.Helper()
	type fact struct {
		Name         string `json:"name"`
		Kind         byte   `json:"kind"`
		UID, GID     int
		Mode, Size   int64
		Link, SHA256 string
	}
	var facts []fact
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	_, err := readUVNativeTar(ctx, runner, container, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
		limit := policy.MaxGraphUncompressed() + policy.MaxGraphFiles()*8192 + (32 << 20)
		bounded := &io.LimitedReader{R: input, N: limit}
		archive := tar.NewReader(bounded)
		var size int64
		seen := map[string]bool{}
		for {
			h, err := archive.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return uvNativeOutput{}, err
			}
			name := strings.TrimSuffix(strings.TrimPrefix(h.Name, "./"), "/")
			if name != "" && name != "." && !validClosureDestination(name) || seen[name] || h.Size < 0 || h.Size > policy.MaxGraphUncompressed()-size || int64(len(facts)) >= policy.MaxGraphFiles()*8+512 {
				return uvNativeOutput{}, fmt.Errorf("native diagnostic census invalid or over bounds")
			}
			size += h.Size
			seen[name] = true
			row := fact{Name: name, Kind: h.Typeflag, UID: h.Uid, GID: h.Gid, Mode: h.Mode, Size: h.Size, Link: h.Linkname}
			if h.Typeflag == tar.TypeReg || h.Typeflag == 0 {
				hash := sha256.New()
				if n, err := io.CopyN(hash, archive, h.Size); err != nil || n != h.Size {
					return uvNativeOutput{}, fmt.Errorf("native diagnostic census truncated: %w", err)
				}
				row.SHA256 = hex.EncodeToString(hash.Sum(nil))
			}
			facts = append(facts, row)
		}
		if err := drainUVTarPadding(bounded); err != nil {
			return uvNativeOutput{}, err
		}
		if bounded.N == 0 {
			return uvNativeOutput{}, fmt.Errorf("native diagnostic census tar bound exhausted")
		}
		return uvNativeOutput{}, nil
	})
	if err != nil {
		t.Fatal("complete native DATA census", err)
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Name < facts[j].Name })
	body, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("native_complete_DATA_census phase=%s facts=%s", phase, body)
}
