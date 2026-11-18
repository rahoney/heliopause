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

// Unit directories are opaque controller identities, never artifact paths.
func validEnvironmentUnitSubpath(value string) bool {
	if len(value) != 69 || !strings.HasPrefix(value, "unit-") {
		return false
	}
	for _, c := range value[5:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (p inspectionEnvironmentProfile) payload(roots ...string) ([]byte, error) {
	if p.identity == "" || len(p.fixtures) == 0 || len(roots) > int(domain.MaximumObservationSummaryCount) {
		return nil, errors.New("inspection environment payload is unavailable")
	}
	rooted := len(roots) != 0
	if !rooted {
		roots = []string{""}
	}
	var body bytes.Buffer
	archive := tar.NewWriter(&body)
	seen := make(map[string]bool, len(roots))
	for _, root := range roots {
		prefix := ""
		if rooted {
			if !validEnvironmentUnitSubpath(root) || seen[root] {
				return nil, errEnvironmentState
			}
			seen[root] = true
			prefix = root + "/"
			if err := archive.WriteHeader(&tar.Header{Name: root, Mode: 0700, Typeflag: tar.TypeDir, Uid: 1000, Gid: 1000}); err != nil {
				return nil, err
			}
		}
		for _, file := range p.fixtures {
			if err := archive.WriteHeader(&tar.Header{Name: prefix + file.name, Mode: file.mode, Typeflag: file.kind, Size: int64(len(file.content)), Uid: 1000, Gid: 1000}); err != nil {
				return nil, err
			}
			if _, err := archive.Write([]byte(file.content)); err != nil {
				return nil, err
			}
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func (p inspectionEnvironmentProfile) verifyPlacement(state environmentState, roots ...string) error {
	if p.identity == "" || state.digest == "" || len(p.fixtures) == 0 || len(roots) > int(domain.MaximumObservationSummaryCount) {
		return errEnvironmentState
	}
	rooted := len(roots) != 0
	if !rooted {
		roots = []string{""}
	}
	seen := make(map[string]bool, len(roots))
	for _, root := range roots {
		prefix := ""
		if rooted {
			if !validEnvironmentUnitSubpath(root) || seen[root] {
				return errEnvironmentState
			}
			seen[root] = true
			prefix = root + "/"
			dir, ok := state.files[root]
			if !ok || dir.kind != tar.TypeDir || dir.mode != 0700 || dir.uid != 1000 || dir.gid != 1000 {
				return errEnvironmentState
			}
		}
		for _, expected := range p.fixtures {
			actual, present := state.files[prefix+expected.name]
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
	base := createArguments(session)
	if readOnly {
		for index := range base {
			if base[index] == "heliopause-"+session.String() {
				base[index] += "-anchor"
			}
		}
	}
	for index, arg := range base {
		if arg == nodeImageReference {
			tail := append([]string{"--label", closureVolumeLabel + "=" + volume.transaction, "--label", closureManifestLabel + "=" + profile.identity}, base[index:]...)
			base = append(base[:index], tail...)
			break
		}
	}
	return applyEnvironmentWorkspace(base, nodeImageReference, profile, volume, readOnly, "")
}

// Preserve each consumer's pinned image, command, closure and labels while
// applying the same fixed /tmp, synthetic environment and guest restriction.
func applyEnvironmentWorkspace(base []string, image string, profile inspectionEnvironmentProfile, volume closureVolume, readOnly bool, subpath string) ([]string, error) {
	buildImage := image == PinnedGoRuntime().ImageReference || image == PinnedCargoRuntime().ImageReference
	if (image != nodeImageReference && image != pythonImageReference && !buildImage) || profile.identity == "" || volume.manifestID != profile.identity || volume.destination() != environmentWorkspaceRoot || volume.projectBuildExecutable {
		return nil, errEnvironmentState
	}
	mount, err := volume.mountArgument(readOnly, subpath)
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, len(base)+len(profile.environment)+4)
	isolated := isolatedContainerEnvironmentArguments()
	workspaces, environment, images := 0, 0, 0
	for index := 0; index < len(base); index++ {
		switch {
		case base[index] == "--tmpfs" && index+1 < len(base) && strings.HasPrefix(base[index+1], "/tmp:"):
			args = append(args, "--mount", mount)
			index++
			workspaces++
		case base[index] == "--env" && index+1 < len(base):
			if environment < len(isolated)/2 {
				if base[index+1] != isolated[2*environment+1] {
					return nil, errEnvironmentState
				}
			} else if buildImage {
				// Keep the current builder's fixed compiler/offline settings.
				// Only the generic prefix is replaced by the synthetic profile.
				args = append(args, base[index], base[index+1])
			} else {
				return nil, errEnvironmentState
			}
			index++
			environment++
		case base[index] == image:
			args = append(args, "--annotation", environmentWorkspacePolicy)
			args = append(args, profile.environment...)
			args = append(args, base[index])
			images++
		default:
			args = append(args, base[index])
		}
	}
	if workspaces != 1 || environment < len(isolated)/2 || images != 1 {
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

// Instance writers retain their existing attested project/runtime layout. A
// DATA keeper uses the ordinary fixed Node helper and reads both workspaces.
func npmInstanceEnvironmentExpectedTopology(kind nativeRuntimeToolKind, nativeBun, readOnly bool) ([]observerMountExpectation, bool) {
	spec, known := nativeProjectIdentity(kind)
	if !known || nativeBun && (kind != nativeRuntimeBun || readOnly) {
		return nil, false
	}
	var topology []observerMountExpectation
	var ok bool
	switch kind {
	case nativeRuntimePNPM:
		topology, ok = pnpmTargetLifecycleExpectedTopology()
	case nativeRuntimeBun:
		if nativeBun {
			topology, ok = bunRuntimeTargetLifecycleExpectedTopology()
		} else {
			topology, ok = bunTargetLifecycleExpectedTopology()
		}
	case nativeRuntimeNPMProject:
		topology, ok = nodeMaterializationExpectedTopology()
	}
	if !ok {
		return nil, false
	}
	workspace, project := 0, 0
	for index, mount := range topology {
		switch mount.Mountpoint {
		case environmentWorkspaceRoot:
			topology[index] = observerMountExpectation{environmentWorkspaceRoot, "workspace", "/", "9p", readOnly, true, true, false}
			workspace++
		case spec.root:
			topology[index].ReadOnly = readOnly
			project++
		}
	}
	return topology, workspace == 1 && project == 1
}

func (o *SharedObserver) startNPMInstanceEnvironmentProfile(ctx context.Context, container string, kind nativeRuntimeToolKind, nativeBun, readOnly bool, budget *observationTraceLedger) (TraceReader, error) {
	topology, ok := npmInstanceEnvironmentExpectedTopology(kind, nativeBun, readOnly)
	if !ok || budget == nil {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return o.startProfileWithBudget(ctx, container, "npm-lifecycle", topology, budget)
}
