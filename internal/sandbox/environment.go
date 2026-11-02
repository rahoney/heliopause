package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type inspectionEnvironment = domain.SandboxEnvironmentProfile

const (
	minimalEnvironment         = domain.SandboxEnvironmentMinimal
	developmentEnvironment     = domain.SandboxEnvironmentDevelopment
	environmentWorkspaceRoot   = "/tmp"
	environmentWorkspacePolicy = "dev.heliopause.workspace-policy=tmp-noexec-v1"
)

type environmentFixture struct {
	name, content string
	mode          int64
	kind          byte
	canary        bool
}

// One controller-owned synthetic profile for all ecosystem backends. None of
// these values come from the Host or an artifact. Configuration fixtures are
// distinct from the existing exact honeytoken path.
type inspectionEnvironmentProfile struct {
	kind        inspectionEnvironment
	identity    string
	environment []string
	fixtures    []environmentFixture
}

func newInspectionEnvironmentProfile(kind inspectionEnvironment, session domain.SandboxSessionID) (inspectionEnvironmentProfile, error) {
	if session.String() == "" || (kind != minimalEnvironment && kind != developmentEnvironment) {
		return inspectionEnvironmentProfile{}, errors.New("inspection environment profile is invalid")
	}
	marker := sha256.Sum256([]byte(string(kind) + "\x00" + session.String()))
	invalid := "haa-invalid-" + hex.EncodeToString(marker[:16])
	profile := inspectionEnvironmentProfile{kind: kind, environment: isolatedContainerEnvironmentArguments(), fixtures: []environmentFixture{
		{name: ".haa-honeytoken", content: invalid + "\n", mode: 0400, kind: tar.TypeReg, canary: true},
	}}
	if kind == developmentEnvironment {
		// The fixed lifecycle already uses HOME=/tmp. Put ordinary config
		// cues there so they are visible without changing its trusted argv.
		for _, name := range []string{"GITHUB_TOKEN", "NPM_TOKEN", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"} {
			profile.environment = append(profile.environment, "--env", name+"="+invalid)
		}
		profile.fixtures = append(profile.fixtures,
			environmentFixture{name: ".aws", mode: 0700, kind: tar.TypeDir},
			environmentFixture{name: ".ssh", mode: 0700, kind: tar.TypeDir},
			environmentFixture{name: "project", mode: 0700, kind: tar.TypeDir},
			environmentFixture{name: "project/.env", content: "APP_ENV=development\nAPI_TOKEN=" + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: ".aws/credentials", content: "[default]\naws_access_key_id=" + invalid + "\naws_secret_access_key=" + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: ".npmrc", content: "//registry.npmjs.org/:_authToken=" + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: ".pypirc", content: "[pypi]\nusername=__token__\npassword=" + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: ".ssh/id_ed25519", content: "HAA INVALID SYNTHETIC KEY " + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: "project/README.md", content: "Synthetic inspection workspace.\n", mode: 0600, kind: tar.TypeReg},
		)
	}
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%s\x00%s\n", kind, session.String())
	for _, entry := range profile.environment {
		_, _ = fmt.Fprintf(hash, "%d:%s\n", len(entry), entry)
	}
	for _, file := range profile.fixtures {
		_, _ = fmt.Fprintf(hash, "%s\x00%o:%d:%t:%d:%s\n", file.name, file.mode, file.kind, file.canary, len(file.content), file.content)
	}
	profile.identity = hex.EncodeToString(hash.Sum(nil))
	return profile, nil
}

func (p inspectionEnvironmentProfile) payload() ([]byte, error) {
	if p.identity == "" || len(p.fixtures) == 0 {
		return nil, errors.New("inspection environment payload is unavailable")
	}
	var body bytes.Buffer
	archive := tar.NewWriter(&body)
	for _, file := range p.fixtures {
		if err := archive.WriteHeader(&tar.Header{Name: file.name, Mode: file.mode, Typeflag: file.kind, Size: int64(len(file.content)), Uid: 1000, Gid: 1000}); err != nil {
			return nil, err
		}
		if _, err := archive.Write([]byte(file.content)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func (p inspectionEnvironmentProfile) verifyPlacement(state environmentState) error {
	if p.identity == "" || state.digest == "" || len(p.fixtures) == 0 {
		return errEnvironmentState
	}
	for _, expected := range p.fixtures {
		actual, present := state.files[expected.name]
		if !present || actual.kind != expected.kind || actual.mode != expected.mode || actual.uid != 1000 || actual.gid != 1000 || actual.size != int64(len(expected.content)) {
			return errEnvironmentState
		}
		if expected.kind == tar.TypeReg {
			digest := sha256.Sum256([]byte(expected.content))
			if actual.digest != hex.EncodeToString(digest[:]) {
				return errEnvironmentState
			}
		}
	}
	return nil
}

// Reuse the existing finite named-volume owner. Keeping the destination fixed
// preserves the existing runtime paths; a profile cannot select a Host mount.
func createEnvironmentWorkspace(ctx context.Context, runner CommandRunner, session domain.SandboxSessionID, profile inspectionEnvironmentProfile, capacity int64) (closureVolume, error) {
	if session.String() == "" || profile.identity == "" {
		return closureVolume{}, errEnvironmentState
	}
	return createInputVolume(ctx, runner, session.String(), profile.identity, capacity, false, projectBuildVolumeConfig{environmentWorkspaceRoot, false})
}

func environmentContainerArguments(session domain.SandboxSessionID, profile inspectionEnvironmentProfile, volume closureVolume, readOnly bool) ([]string, error) {
	if session.String() == "" || profile.identity == "" || volume.manifestID != profile.identity ||
		volume.destination() != environmentWorkspaceRoot || volume.projectBuildExecutable {
		return nil, errEnvironmentState
	}
	if _, err := domain.ParseSandboxSessionID(volume.transaction); err != nil {
		return nil, errEnvironmentState
	}
	mount, err := volume.mountArgument(readOnly)
	if err != nil {
		return nil, err
	}
	base := createArguments(session)
	if readOnly {
		for index := range base {
			if base[index] == "heliopause-"+session.String() {
				base[index] += "-anchor"
			}
		}
	}
	args := make([]string, 0, len(base)+len(profile.environment)+4)
	workspaces, environment, images := 0, 0, 0
	for index := 0; index < len(base); index++ {
		switch {
		case base[index] == "--tmpfs" && index+1 < len(base) && strings.HasPrefix(base[index+1], "/tmp:"):
			args = append(args, "--mount", mount)
			index++
			workspaces++
		case base[index] == "--env" && index+1 < len(base):
			index++
			environment++
		case base[index] == nodeImageReference:
			args = append(args, "--label", closureVolumeLabel+"="+volume.transaction,
				"--label", closureManifestLabel+"="+profile.identity, "--annotation", environmentWorkspacePolicy)
			args = append(args, profile.environment...)
			args = append(args, base[index])
			images++
		default:
			args = append(args, base[index])
		}
	}
	if workspaces != 1 || environment != len(isolatedContainerEnvironmentArguments())/2 || images != 1 {
		return nil, errEnvironmentState
	}
	return args, nil
}

// Only this typed entry changes the existing npm profile's workspace topology.
// Guest noexec/nosuid must be present independently of the backing tmpfs.
// A runtime that only provides ordinary 9p fails before any artifact launch.
func (o *SharedObserver) StartEnvironmentProfile(ctx context.Context, container string, readOnly bool, budget *observationTraceLedger) (TraceReader, error) {
	topology, ok := environmentExpectedTopology(readOnly)
	if !ok || budget == nil {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return o.startProfileWithBudget(ctx, container, "npm-lifecycle", topology, budget)
}

func environmentExpectedTopology(readOnly bool) ([]observerMountExpectation, bool) {
	topology, ok := observerExpectedTopology("npm-lifecycle")
	if !ok {
		return nil, false
	}
	for index, mount := range topology {
		if mount.Mountpoint == environmentWorkspaceRoot {
			topology[index] = observerMountExpectation{environmentWorkspaceRoot, "workspace", "/", "9p", readOnly, true, true, false}
			return topology, true
		}
	}
	return nil, false
}
