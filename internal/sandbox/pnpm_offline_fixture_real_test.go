//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// Primitive qualification only: native store population from verified archives
// and an unchanged registry-shaped lock. Dynamic completion remains separate.
func TestPNPMActualOfflineStoreFixture(t *testing.T) {
	if os.Getenv("HELOX_PNPM_PUBLIC_METADATA_FIXTURE") != "1" {
		t.Fatal("declare HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 for the authenticated metadata preparation stage")
	}
	for _, fixture := range pnpmOfflineFixtures() {
		if !t.Run(fixture.name, func(t *testing.T) { qualifyPNPMOfflineFixture(t, fixture) }) {
			t.FailNow()
		}
	}
}

type pnpmOfflineIdentity struct{ directory, name, version string }
type pnpmOfflineLink struct{ name, target string }
type pnpmOfflineFixture struct {
	name, manifest, lockFixture string
	specs                       []string
	roots, snapshots            map[string]any
	installed                   []pnpmOfflineIdentity
	links                       []pnpmOfflineLink
}

// Explicit qualification inputs, never a general lock/layout codec.
func pnpmOfflineFixtures() []pnpmOfflineFixture {
	return []pnpmOfflineFixture{
		{
			name: "transitive", manifest: `{"dependencies":{"is-odd":"3.0.1"}}`,
			specs:     []string{"is-odd@3.0.1", "is-number@6.0.0"},
			roots:     map[string]any{"is-odd": map[string]string{"specifier": "3.0.1", "version": "3.0.1"}},
			snapshots: map[string]any{"is-number@6.0.0": map[string]any{}, "is-odd@3.0.1": map[string]any{"dependencies": map[string]string{"is-number": "6.0.0"}}},
			installed: []pnpmOfflineIdentity{{"is-odd@3.0.1", "is-odd", "3.0.1"}, {"is-number@6.0.0", "is-number", "6.0.0"}},
			links:     []pnpmOfflineLink{{".pnpm/is-odd@3.0.1/node_modules/is-number", "../../is-number@6.0.0/node_modules/is-number"}},
		},
		{
			name: "peer", manifest: `{"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0"}}`,
			specs:       []string{"react@18.3.1", "use-sync-external-store@1.2.0", "loose-envify@1.4.0", "js-tokens@4.0.0"},
			lockFixture: "pnpm-v9-peer.synthetic.yaml",
			installed:   []pnpmOfflineIdentity{{"react@18.3.1", "react", "18.3.1"}, {"use-sync-external-store@1.2.0_react@18.3.1", "use-sync-external-store", "1.2.0"}, {"loose-envify@1.4.0", "loose-envify", "1.4.0"}, {"js-tokens@4.0.0", "js-tokens", "4.0.0"}},
			links: []pnpmOfflineLink{
				{".pnpm/use-sync-external-store@1.2.0_react@18.3.1/node_modules/react", "../../react@18.3.1/node_modules/react"},
				{".pnpm/react@18.3.1/node_modules/loose-envify", "../../loose-envify@1.4.0/node_modules/loose-envify"},
				{".pnpm/loose-envify@1.4.0/node_modules/js-tokens", "../../js-tokens@4.0.0/node_modules/js-tokens"},
				{"react", ".pnpm/react@18.3.1/node_modules/react"},
				{"use-sync-external-store", ".pnpm/use-sync-external-store@1.2.0_react@18.3.1/node_modules/use-sync-external-store"},
			},
		},
	}
}

func qualifyPNPMOfflineFixture(t *testing.T, fixture pnpmOfflineFixture) {
	t.Helper()
	withActualPNPMNativeRuntimePolicy(t, true, func(ctx context.Context, runner CommandRunner, container, network string) {
		configured, err := runner.Output(ctx, "docker", "inspect", "--format", "{{json .Config.WorkingDir}}", container)
		var workdir string
		if err != nil || json.Unmarshal(configured, &workdir) != nil || (workdir != "" && workdir != "/") {
			t.Fatal("fixed native container working directory differs", err)
		}
		t.Log("pnpm_offline_store_context process_cwd=/ store_command_dir=/")
		intake := t.TempDir()
		source, err := artifactnpm.NewPublicResolver(intake)
		if err != nil {
			t.Fatal(err)
		}
		static, err := inspectionnpm.NewInspector(intake)
		if err != nil {
			t.Fatal(err)
		}
		var recipeBody []byte
		var sourceBound artifactnpm.PNPMSelection
		resolvedInputs := map[string]domain.ResolvedArtifact{}
		if fixture.lockFixture != "" {
			// Reuse the exact versioned parser fixture and its public-binding API.
			_, file, _, ok := runtime.Caller(0)
			if !ok {
				t.Fatal("qualification source location unavailable")
			}
			recipeBody, err = os.ReadFile(filepath.Join(filepath.Dir(file), "..", "artifact", "npm", "testdata", fixture.lockFixture))
			if err != nil {
				t.Fatal("versioned peer fixture unavailable", err)
			}
			parsed, err := artifactnpm.ParsePNPMLock([]byte(fixture.manifest), recipeBody, runtimeidentity.PNPMVersion)
			if err != nil {
				t.Fatal("versioned peer fixture invalid", err)
			}
			sourceBound, err = source.BindPNPMLock(ctx, parsed)
			if err != nil || !sourceBound.SourceBound() {
				t.Fatal("exact peer metadata binding failed", err)
			}
			for _, resolved := range sourceBound.Artifacts() {
				resolvedInputs[resolved.Identity().Name()+"@"+resolved.Identity().Version()] = resolved
			}
		}
		packages := map[string]any{}
		var fileSpecs []string
		var input bytes.Buffer
		archive := tar.NewWriter(&input)
		if err := archive.WriteHeader(&tar.Header{Name: "artifacts", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
		if err := archive.WriteHeader(&tar.Header{Name: "tmp", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
		for _, spec := range fixture.specs {
			ref, err := artifactnpm.ParseReference(spec)
			if err != nil {
				t.Fatal(err)
			}
			resolved, found := resolvedInputs[spec]
			if !found {
				if fixture.lockFixture != "" {
					t.Fatal("exact fixture input was not source bound", spec)
				}
				resolved, err = source.Resolve(ctx, ref)
				if err != nil {
					t.Fatal(err)
				}
			}
			run, err := domain.NewRunID()
			if err != nil {
				t.Fatal(err)
			}
			acquired, err := source.Acquire(ctx, run, resolved)
			if err != nil {
				t.Fatal(err)
			}
			verified, err := (verificationnpm.IntegrityVerifier{}).Verify(ctx, acquired)
			if err != nil || verified.Outcome() != domain.VerificationVerified {
				t.Fatalf("fixture integrity failed: %v", err)
			}
			report, err := static.Inspect(ctx, acquired)
			if err != nil || len(report.Findings()) != 0 || report.Execution().Status() != domain.ExecutionCompleted {
				t.Fatalf("fixture static admission failed: %v", err)
			}
			if acquired.SizeBytes() <= 0 || acquired.SizeBytes() > 4<<20 {
				t.Fatal("fixture archive exceeds its bound")
			}
			body, err := os.ReadFile(filepath.Join(intake, run.String(), "tarball.tgz"))
			if err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(body)
			if uint64(len(body)) != acquired.SizeBytes() || hex.EncodeToString(hash[:]) != acquired.Digest().String() {
				t.Fatal("verified fixture bytes changed")
			}
			name := "artifacts/" + acquired.Digest().String() + ".tgz"
			if err := archive.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0600, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
				t.Fatal(err)
			}
			if _, err := archive.Write(body); err != nil {
				t.Fatal(err)
			}
			integrity, present := acquired.DeclaredIntegrity()
			if !present {
				t.Fatal("fixture lost declared integrity")
			}
			packages[spec] = map[string]any{"resolution": map[string]string{"integrity": integrity}}
			fileSpecs = append(fileSpecs, "file:"+pnpmProjectRoot+"/"+name)
			t.Logf("pnpm_offline_input identity=%s bytes=%d sha256=%s", spec, len(body), acquired.Digest().String())
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		// Explicit fixture data preserves registry source kind and full peer keys.
		recipe, err := json.Marshal(map[string]any{
			"lockfileVersion": "9.0",
			"settings":        map[string]bool{"autoInstallPeers": true, "excludeLinksFromLockfile": false},
			"importers":       map[string]any{".": map[string]any{"dependencies": fixture.roots}},
			"packages":        packages,
			"snapshots":       fixture.snapshots,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(recipeBody) != 0 {
			recipe = recipeBody
		}
		manifest := pnpmTestControl(t, "package.json", fixture.manifest, true)
		lock, err := domain.NewProjectControlFile("pnpm-lock.yaml", recipe, true)
		if err != nil {
			t.Fatal(err)
		}
		controls := []domain.ProjectControlFile{manifest, lock}
		parsed, err := artifactnpm.ParsePNPMLock(manifest.Body(), lock.Body(), runtimeidentity.PNPMVersion)
		if err != nil {
			t.Fatal("public fixture graph invalid before metadata preparation", err)
		}
		if !sourceBound.SourceBound() {
			sourceBound, err = source.BindPNPMLock(ctx, parsed)
		}
		if err != nil || len(sourceBound.Artifacts()) != len(fixture.specs) {
			t.Fatal("public fixture source binding failed before metadata preparation", err)
		}
		if err := copyPNPMProjectControls(ctx, runner, container, controls); err != nil {
			t.Fatal(err)
		}
		if err := runner.(inputCommandRunner).RunInput(ctx, bytes.NewReader(input.Bytes()), "docker", "cp", "--archive", "-", container+":"+pnpmProjectRoot); err != nil {
			t.Fatal("fixture archive introduction failed", err)
		}
		for _, spec := range fileSpecs {
			file := strings.TrimPrefix(spec, "file:")
			body := readPNPMFixtureFile(t, ctx, runner.(streamingDockerOutput), container, file)
			hash := sha256.Sum256(body)
			if filepath.Base(file) != hex.EncodeToString(hash[:])+".tgz" {
				t.Fatal("introduced native archive differs from its verified digest")
			}
		}
		t.Logf("pnpm_offline_input_readback files=%d verified_digest=true", len(fixture.specs))
		// Store-add does not install a project. Its file resolver's directory
		// must match the real process cwd used to open relative file locators.
		storeArgs := []string{pnpmToolGuestPath, "--dir", "/", "store", "add", "--offline", "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--registry=https://registry.npmjs.org/", "--store-dir", pnpmProjectRoot + "/store", "--state-dir", pnpmProjectRoot + "/state", "--reporter=append-only"}
		runPNPMFixtureCommand(t, ctx, runner, container, "store-add", append(storeArgs, fileSpecs...))
		fetchArgs := []string{pnpmToolGuestPath, "--dir", pnpmProjectDirectory, "fetch", "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--registry=https://registry.npmjs.org/", "--store-dir", pnpmProjectRoot + "/store", "--state-dir", pnpmProjectRoot + "/state", "--reporter=append-only", "--node-linker=isolated"}
		runPNPMFixtureCommand(t, ctx, runner, container, "metadata-fetch", fetchArgs)
		if network == "none" {
			t.Fatal("metadata network was not the owned authenticated policy")
		}
		if _, err := runner.Output(ctx, "docker", "network", "disconnect", network, container); err != nil {
			t.Fatal("metadata network disconnect failed", err)
		}
		endpoints, err := runner.Output(ctx, "docker", "inspect", "--format", "{{json .NetworkSettings.Networks}}", container)
		var remaining map[string]json.RawMessage
		if err != nil || json.Unmarshal(endpoints, &remaining) != nil || len(remaining) != 0 {
			t.Fatal("offline fixture network absence is unproven", err)
		}
		t.Log("pnpm_offline_network verified_endpoints=0 metadata_stage=COMPLETED artifact_execution=NOT_STARTED")
		args := []string{pnpmToolGuestPath, "--dir", pnpmProjectDirectory, "install", "--offline", "--frozen-lockfile", "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--registry=https://registry.npmjs.org/", "--store-dir", pnpmProjectRoot + "/store", "--state-dir", pnpmProjectRoot + "/state", "--reporter=silent", "--node-linker=isolated"}
		runPNPMFixtureCommand(t, ctx, runner, container, "frozen-install", args)
		for _, before := range controls {
			after, err := readPNPMProjectControl(ctx, runner.(streamingDockerOutput), container, before.Name())
			if err != nil || !bytes.Equal(before.Body(), after.Body()) {
				t.Fatalf("frozen private control changed: %s %v", before.Name(), err)
			}
		}
		for _, spec := range fixture.installed {
			file := pnpmProjectDirectory + "/node_modules/.pnpm/" + spec.directory + "/node_modules/" + spec.name + "/package.json"
			body := readPNPMFixtureFile(t, ctx, runner.(streamingDockerOutput), container, file)
			var identity struct{ Name, Version string }
			if json.Unmarshal(body, &identity) != nil || identity.Name != spec.name || identity.Version != spec.version {
				t.Fatal("native offline installed identity differs")
			}
		}
		for _, expected := range fixture.links {
			assertPNPMFixtureLink(t, ctx, runner.(streamingDockerOutput), container, expected)
		}
		t.Logf("pnpm_offline_store_fixture case=%s packages=%d instances=%d links=%d controls_unchanged=true native_layout=isolated scripts=NOT_EXECUTED dynamic=NOT_EXECUTED Host_install=NOT_EXECUTED", fixture.name, len(fixture.specs), len(fixture.installed), len(fixture.links))
	})
}

func assertPNPMFixtureLink(t *testing.T, ctx context.Context, runner streamingDockerOutput, container string, expected pnpmOfflineLink) {
	t.Helper()
	output := nativeToolArchiveOutput{limit: 64 << 10}
	if err := runner.RunOutput(ctx, &output, "docker", "cp", container+":"+pnpmProjectDirectory+"/node_modules/"+expected.name, "-"); err != nil || output.failed {
		t.Fatal("read native dependency link", err)
	}
	reader := tar.NewReader(bytes.NewReader(output.body))
	link, err := reader.Next()
	if err != nil || link.Typeflag != tar.TypeSymlink || link.Name != filepath.Base(expected.name) || link.Linkname != expected.target || link.Size != 0 {
		t.Fatal("native dependency link differs from exact fixture", expected.name, err)
	}
	if _, err := reader.Next(); err != io.EOF {
		t.Fatal("native dependency link output has extra entries", err)
	}
}

func runPNPMFixtureCommand(t *testing.T, ctx context.Context, runner CommandRunner, container, stage string, args []string) {
	t.Helper()
	if output, err := runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, args...)...); err != nil {
		var exited *exec.ExitError
		if errors.As(err, &exited) {
			stderr := exited.Stderr
			if len(stderr) > 4096 {
				stderr = stderr[:4096]
			}
			t.Logf("pnpm_offline_native_stderr_qualification_only stage=%s text=%q", stage, stderr)
		}
		if len(output) > 4096 {
			output = output[:4096]
		}
		t.Logf("pnpm_offline_native_stdout_qualification_only stage=%s text=%q", stage, output)
		t.Fatalf("native offline fixture failed stage=%s: %v", stage, err)
	}
}

func readPNPMFixtureFile(t *testing.T, ctx context.Context, runner streamingDockerOutput, container, file string) []byte {
	t.Helper()
	output := nativeToolArchiveOutput{limit: (1 << 20) + (64 << 10)}
	if err := runner.RunOutput(ctx, &output, "docker", "cp", container+":"+file, "-"); err != nil || output.failed {
		t.Fatal("read bounded native fixture output", err)
	}
	reader := tar.NewReader(bytes.NewReader(output.body))
	header, err := reader.Next()
	if err != nil {
		t.Fatal("native fixture output header is unavailable", err)
	}
	if header.Name != filepath.Base(file) || header.Typeflag != tar.TypeReg || header.Linkname != "" || header.Size <= 0 || header.Size > 1<<20 || header.Uid != 1000 || header.Gid != 1000 {
		t.Fatalf("native fixture output metadata invalid: name_matches=%t type=%d uid=%d gid=%d bytes=%d", header.Name == filepath.Base(file), header.Typeflag, header.Uid, header.Gid, header.Size)
	}
	body, err := io.ReadAll(reader)
	if err != nil || int64(len(body)) != header.Size {
		t.Fatal("native fixture output is incomplete", err)
	}
	if _, err := reader.Next(); err != io.EOF {
		t.Fatal("native fixture output contains extra entries", err)
	}
	return body
}
