//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// Command qualification only. Public archive layout is deliberately changed
// into a synthetic script probe; it never receives Policy/staging authority.
func TestPNPMActualTargetLifecycleFixture(t *testing.T) {
	if os.Getenv("HELOX_PNPM_TARGET_LIFECYCLE_FIXTURE") != "1" {
		t.Fatal("declare HELOX_PNPM_TARGET_LIFECYCLE_FIXTURE=1 for synthetic lifecycle qualification")
	}
	legacy := runLinuxGVisorLifecycleIntegration(t, `{"name":"tiny","version":"1.2.3","scripts":{"preinstall":"printf preinstall >> /tmp/legacy-events","install":"printf install >> /tmp/legacy-events","postinstall":"printf postinstall >> /tmp/legacy-events"}}`)
	var legacyArtifactExecs uint64
	for _, observation := range legacy.Observations() {
		if observation.Category() == domain.ObservationProcess && observation.Subject() == "process-exec-unexpected" {
			legacyArtifactExecs += observation.Count()
		}
	}
	if legacyArtifactExecs != 3 {
		t.Fatalf("existing npm script attribution differs: count=%d", legacyArtifactExecs)
	}
	t.Logf("legacy_npm_script_attribution artifact_execs=%d required_review_semantics=UNCHANGED", legacyArtifactExecs)
	for _, fixture := range pnpmOfflineFixtures() {
		if !t.Run(fixture.name, func(t *testing.T) {
			var nativeLayout []byte
			var originalControls []domain.ProjectControlFile
			qualifyPNPMOfflineFixture(t, fixture, func(ctx context.Context, runner CommandRunner, container string, controls []domain.ProjectControlFile, _ artifactnpm.PNPMInspectionInput, _ []artifactnpm.ArchiveInventory) {
				output := nativeToolArchiveOutput{limit: 16 << 20}
				if err := runner.(streamingDockerOutput).RunOutput(ctx, &output, "docker", "cp", container+":"+pnpmProjectDirectory+"/node_modules", "-"); err != nil || output.failed {
					t.Fatal("read bounded native fixture layout", err)
				}
				nativeLayout = append([]byte(nil), output.body...)
				originalControls = append([]domain.ProjectControlFile(nil), controls...)
			})
			if t.Failed() || len(nativeLayout) == 0 {
				t.Fatal("native prerequisite did not complete and drain")
			}
			target := fixture.installed[0]
			if fixture.name == "peer" {
				target = fixture.installed[1]
			}
			targetPath := pnpmProjectDirectory + "/node_modules/.pnpm/" + target.directory + "/node_modules/" + target.name
			for _, item := range []struct {
				name          string
				failure       bool
				events        string
				artifactExecs uint64
			}{
				{"positive", false, "preinstall\ninstall\npostinstall\n", 3},
				{"script-failure", true, "preinstall\n", 2},
			} {
				if !t.Run(item.name, func(t *testing.T) {
					input := pnpmSyntheticTargetLifecycleArchive(t, nativeLayout, originalControls, fixture.installed, target, item.failure)
					t.Logf("synthetic_lifecycle_scope original_layout_sha256=%x changed_input_sha256=%x deliberately_modified=true Policy=NOT_RUN Host_install=NOT_RUN", sha256.Sum256(nativeLayout), sha256.Sum256(input))
					withActualPNPMTargetLifecycle(t, input, pnpmTargetLifecycleExpectation{targetPath, item.failure, item.events, item.artifactExecs})
				}) {
					t.FailNow()
				}
			}
		}) {
			t.FailNow()
		}
	}
}

type pnpmTargetLifecycleExpectation struct {
	target        string
	failure       bool
	events        string
	artifactExecs uint64
}

func pnpmSyntheticTargetLifecycleArchive(t *testing.T, original []byte, controls []domain.ProjectControlFile, installed []pnpmOfflineIdentity, selected pnpmOfflineIdentity, scriptFailure bool) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	write := func(header *tar.Header, body []byte) {
		t.Helper()
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	write(&tar.Header{Name: "project/", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}, nil)
	poison := map[string]string{"preinstall": "exit 73", "install": "exit 73", "postinstall": "exit 73", "prepare": "exit 73", "prepublish": "exit 73", "preprepare": "exit 73"}
	for _, control := range controls {
		body := control.Body()
		if control.Name() == "package.json" {
			var manifest map[string]any
			if err := json.Unmarshal(body, &manifest); err != nil {
				t.Fatal(err)
			}
			manifest["scripts"] = poison
			var err error
			body, err = json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
		}
		write(&tar.Header{Name: "project/" + control.Name(), Typeflag: tar.TypeReg, Mode: 0600, Uid: 1000, Gid: 1000, Size: int64(len(body))}, body)
	}
	expectedManifests := map[string]pnpmOfflineIdentity{}
	for _, item := range installed {
		expectedManifests["node_modules/.pnpm/"+item.directory+"/node_modules/"+item.name+"/package.json"] = item
	}
	reader := tar.NewReader(bytes.NewReader(original))
	seen := map[string]bool{}
	targets, dependencies := 0, 0
	for entries := 0; ; entries++ {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil || entries >= 1000 {
			t.Fatal("synthetic fixture layout is malformed or oversized", err)
		}
		clean := path.Clean(header.Name)
		if seen[clean] || clean != "node_modules" && !strings.HasPrefix(clean, "node_modules/") || path.IsAbs(header.Name) || strings.Contains(header.Name, "\\") || (header.Name != clean && header.Name != clean+"/") || len(clean) > 512 || header.Uid != 1000 || header.Gid != 1000 || header.Size < 0 || header.Size > 1<<20 {
			t.Fatal("synthetic fixture layout metadata invalid", header.Name)
		}
		seen[clean] = true
		var body []byte
		switch header.Typeflag {
		case tar.TypeDir:
			if header.Size != 0 {
				t.Fatal("directory has payload")
			}
		case tar.TypeReg:
			body, err = io.ReadAll(reader)
			if err != nil || int64(len(body)) != header.Size {
				t.Fatal("synthetic fixture file incomplete", err)
			}
			if expected, found := expectedManifests[clean]; found {
				var manifest map[string]any
				if err := json.Unmarshal(body, &manifest); err != nil {
					t.Fatal(err)
				}
				if manifest["name"] != expected.name || manifest["version"] != expected.version {
					t.Fatal("synthetic source identity changed")
				}
				if expected == selected {
					targets++
					scripts := map[string]string{"preinstall": `printf 'preinstall\n' >> /tmp/haa-pnpm-input/lifecycle-events`, "install": `printf 'install\n' >> /tmp/haa-pnpm-input/lifecycle-events`, "postinstall": `printf 'postinstall\n' >> /tmp/haa-pnpm-input/lifecycle-events`, "prepare": "exit 75"}
					if scriptFailure {
						scripts["install"] = "exit 37"
					}
					manifest["scripts"] = scripts
				} else {
					dependencies++
					manifest["scripts"] = poison
				}
				body, err = json.Marshal(manifest)
				if err != nil {
					t.Fatal(err)
				}
				header.Size = int64(len(body))
			}
		case tar.TypeSymlink:
			resolved := path.Join(path.Dir(clean), header.Linkname)
			if header.Linkname == "" || header.Size != 0 || path.IsAbs(header.Linkname) || !strings.HasPrefix(resolved, "node_modules/") || strings.Contains(header.Linkname, "\\") {
				t.Fatal("synthetic fixture link escapes", header.Name)
			}
		case tar.TypeLink:
			if header.Linkname == clean || !seen[header.Linkname] || header.Size != 0 {
				t.Fatal("synthetic fixture hardlink is not a prior member", header.Name)
			}
			header.Linkname = "project/" + header.Linkname
		default:
			t.Fatal("synthetic fixture has unsupported member type", header.Typeflag)
		}
		header.Name = "project/" + clean
		write(header, body)
	}
	if targets != 1 || dependencies != len(installed)-1 {
		t.Fatal("synthetic target/dependency subjects incomplete")
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func withActualPNPMTargetLifecycle(t *testing.T, input []byte, expect pnpmTargetLifecycleExpectation) {
	t.Helper()
	wantFailure, wantEvents, wantArtifactExecs := expect.failure, expect.events, expect.artifactExecs
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
	runner := admissionAwareRunner(executor)
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	project, err := createPNPMProjectVolume(ctx, runner, session.String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, done := resolverCleanupContext()
		defer done()
		if err := closePNPMProjectVolume(cleanup, runner, &project); err != nil {
			t.Error(err)
		}
	}()
	mount, err := project.mountArgument(false)
	if err != nil {
		t.Fatal(err)
	}
	args := createArguments(session)
	image := slices.Index(args, runtimeidentity.NodeImageReference)
	if image < 0 {
		t.Fatal("pinned NPM image absent")
	}
	fixed := append([]string{}, args[:image]...)
	fixed = append(fixed, "--mount", mount)
	args = append(fixed, args[image:]...)
	created, err := runner.Output(ctx, "docker", args...)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		t.Fatal("target lifecycle create failed", err)
	}
	t.Logf("owned_pnpm_target_container id=%s project_volume=%s", id, project.name)
	var trace TraceReader
	defer func() {
		cleanup, done := resolverCleanupContext()
		defer done()
		if _, err := runner.Output(cleanup, "docker", "rm", "--force", id); err != nil {
			t.Error("target container cleanup unproven", err)
		}
		if trace != nil {
			_, limitation, diagnostic := collectTraceDiagnostic(cleanup, trace)
			t.Logf("pnpm_target_trace limitation=%s diagnostic=%+v", limitation, diagnostic)
			if limitation != "" {
				t.Error("target lifecycle trace incomplete", limitation)
			}
			// The three known script shells must remain ARTIFACT observations,
			// just as in the original npm install baseline above.
			if diagnostic.KindCounts["process-exec-unexpected"] != wantArtifactExecs {
				t.Error("synthetic script shells lost their untrusted attribution")
			}
			for _, kind := range []string{"filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
				if diagnostic.KindCounts[kind] != 0 {
					t.Errorf("target lifecycle contains %s=%d", kind, diagnostic.KindCounts[kind])
				}
			}
		}
	}()
	observer, ok := supervisor.Observer().(interface {
		StartPNPMTargetLifecycleProfile(context.Context, string) (TraceReader, error)
	})
	if !ok {
		t.Fatal("exact target lifecycle observer unavailable")
	}
	trace, err = observer.StartPNPMTargetLifecycleProfile(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Output(ctx, "docker", "start", id); err != nil {
		t.Fatal(err)
	}
	if err := awaitBoundaryHelper(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	if err := awaitMountAnchors(ctx, supervisor.Observer(), id); err != nil {
		t.Fatal(err)
	}
	if err := runner.(inputCommandRunner).RunInput(ctx, bytes.NewReader(input), "docker", "cp", "--archive", "-", id+":"+pnpmProjectRoot); err != nil {
		t.Fatal("synthetic layout introduction failed", err)
	}
	mode, err := runner.Output(ctx, "docker", "inspect", "--format", "{{.HostConfig.NetworkMode}}", id)
	if err != nil || strings.TrimSpace(string(mode)) != "none" {
		t.Fatal("target lifecycle network is not none", err)
	}
	endpoints, err := runner.Output(ctx, "docker", "inspect", "--format", "{{json .NetworkSettings.Networks}}", id)
	var networks map[string]struct{ NetworkID, EndpointID, IPAddress, GlobalIPv6Address string }
	if err != nil || json.Unmarshal(endpoints, &networks) != nil {
		t.Fatal("target network endpoint query failed", err)
	}
	noneBody, err := runner.Output(ctx, "docker", "network", "inspect", "none", "--format", "{{json .}}")
	var none struct{ Name, Id, Driver, Scope string }
	if err != nil || json.Unmarshal(noneBody, &none) != nil || none.Name != "none" || none.Driver != "null" || none.Scope != "local" || !exactObservationContainerID(none.Id) {
		t.Fatal("canonical null network identity unavailable", err)
	}
	for name, view := range networks {
		t.Logf("synthetic_target_network name=%s logical_endpoint_present=%t address_present=%t null_network_identity_matches=%t", name, view.EndpointID != "", view.IPAddress != "" || view.GlobalIPv6Address != "", view.NetworkID == none.Id)
		if name != "none" || view.NetworkID != none.Id || view.IPAddress != "" || view.GlobalIPv6Address != "" {
			t.Fatal("target has a non-null network or real address")
		}
	}
	target := expect.target
	command, err := pnpmTargetLifecycleExecArguments(id, target)
	if err != nil {
		t.Fatal(err)
	}
	run, stop := context.WithTimeout(ctx, sandboxWallTimeout)
	defer stop()
	output, runErr := runner.Output(run, "docker", command...)
	if wantFailure {
		var exited *exec.ExitError
		if !errors.As(runErr, &exited) || exited.ExitCode() != 37 || run.Err() != nil {
			t.Fatal("target script first exit was not preserved", runErr)
		}
		t.Logf("synthetic_target_negative command_exit=%d original_script_exit=37 timeout=false operation=FAILED_EXPECTED", exited.ExitCode())
	} else if runErr != nil {
		var exited *exec.ExitError
		if errors.As(runErr, &exited) {
			t.Logf("synthetic_target_stderr_qualification_only=%q", boundedPNPMTargetDiagnostic(exited.Stderr))
		}
		t.Logf("synthetic_target_stdout_qualification_only=%q", boundedPNPMTargetDiagnostic(output))
		t.Fatal("synthetic target lifecycle failed", runErr)
	}
	events := readPNPMFixtureFile(t, ctx, runner.(streamingDockerOutput), id, pnpmProjectRoot+"/lifecycle-events")
	if string(events) != wantEvents {
		t.Fatal("target events are missing, duplicated or out of order", string(events))
	}
	t.Logf("synthetic_target_lifecycle script_failure=%t artifact_execs=%d exact_event_prefix=true root_scripts=0 dependency_scripts=0 target_prepare=0 network_mode=none Policy=NOT_RUN Host_install=NOT_RUN", wantFailure, wantArtifactExecs)
}

func boundedPNPMTargetDiagnostic(body []byte) []byte {
	if len(body) > 4096 {
		return body[:4096]
	}
	return body
}
