package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const closureVolumeLabel = "io.heliopause.observation-transaction"
const closureManifestLabel = "io.heliopause.closure-manifest-sha256"
const closureTmpfsOptions = ",nosuid,nodev,uid=1000,gid=1000,mode=0700"

type closureVolume struct {
	name                    string
	transaction             string
	manifestID              string
	createdAt               string
	mountpoint              string
	capacity                int64
	goBuild                 bool
	goBuildOutput           bool
	projectBuildDestination string
	projectBuildExecutable  bool
}

type dockerVolumeInspection struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Scope      string            `json:"Scope"`
	Mountpoint string            `json:"Mountpoint"`
	CreatedAt  string            `json:"CreatedAt"`
	Options    map[string]string `json:"Options"`
	Labels     map[string]string `json:"Labels"`
}

func closureVolumeName(transaction string) (string, error) {
	if transaction == "" || len(transaction) > 128 || strings.ContainsAny(transaction, "/\\\x00\n\r") {
		return "", errors.New("invalid Python observation transaction identity")
	}
	digest := sha256.Sum256([]byte(transaction))
	return "haa-closure-" + hex.EncodeToString(digest[:16]), nil
}

func createClosureVolume(ctx context.Context, runner CommandRunner, transaction, manifestID string, capacity int64) (closureVolume, error) {
	return createInputVolume(ctx, runner, transaction, manifestID, capacity, false)
}

// Both consumers use the same exact identity and attachment checks. The Go
// input mount is fixed backend configuration; artifact paths cannot select it.
func createGoBuildInputVolume(ctx context.Context, runner CommandRunner, transaction, manifestID string) (closureVolume, error) {
	return createInputVolume(ctx, runner, transaction, manifestID, goBuildInputCapacity, true)
}

func createGoBuildOutputVolume(ctx context.Context, runner CommandRunner, transaction, manifestID string) (closureVolume, error) {
	volume, err := createInputVolume(ctx, runner, transaction+"-output", manifestID, goBuildOutputCapacity, true)
	volume.goBuildOutput = true
	return volume, err
}

func createInputVolume(ctx context.Context, runner CommandRunner, transaction, manifestID string, capacity int64, goBuild bool, project ...projectBuildVolumeConfig) (closureVolume, error) {
	if ctx == nil || runner == nil || capacity <= 0 || len(manifestID) != 64 {
		return closureVolume{}, errors.New("python closure volume configuration is invalid")
	}
	if len(project) > 1 || (len(project) == 1 && (goBuild || (project[0].destination != cargoBuildGuestInput && project[0].destination != cargoBuildGuestTarget) || project[0].executable != (project[0].destination == cargoBuildGuestTarget))) {
		return closureVolume{}, errors.New("project volume mount contract is invalid")
	}
	name, err := closureVolumeName(transaction)
	if err != nil {
		return closureVolume{}, err
	}
	// Docker volume create is idempotent for an existing name. A failed inspect
	// is ambiguous, so use a successful exact inventory to prove absence first.
	listed, listErr := runner.Output(ctx, "docker", "volume", "ls", "--quiet", "--filter", "name=^"+name+"$")
	if listErr != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return closureVolume{}, errors.New("python closure volume identity already exists")
	}
	volume := closureVolume{name: name, transaction: transaction, manifestID: manifestID, capacity: capacity, goBuild: goBuild}
	if len(project) == 1 {
		volume.projectBuildDestination = project[0].destination
		volume.projectBuildExecutable = project[0].executable
	}
	options := volume.tmpfsOptions()
	created, err := runner.Output(ctx, "docker", "volume", "create", "--driver", "local",
		"--opt", "type=tmpfs", "--opt", "device=tmpfs", "--opt", "o="+options,
		"--label", closureVolumeLabel+"="+transaction,
		"--label", closureManifestLabel+"="+manifestID, name)
	if err != nil || strings.TrimSpace(string(created)) != name {
		return closureVolume{}, errors.New("python closure volume creation failed")
	}
	if err := volume.verify(ctx, runner); err != nil {
		// Never adopt a volume whose post-create identity is uncertain. The caller
		// must still fail closed if this best-effort cleanup cannot complete.
		_ = discardCommand(ctx, runner, "docker", "volume", "rm", name)
		return closureVolume{}, err
	}
	return volume, nil
}

func (v closureVolume) tmpfsOptions() string {
	options := "size=" + strconv.FormatInt(v.capacity, 10) + closureTmpfsOptions
	if v.goBuild || (v.projectBuildDestination != "" && !v.projectBuildExecutable) {
		options += ",noexec"
	}
	return options
}

func (v closureVolume) destination() string {
	if v.projectBuildDestination != "" {
		return v.projectBuildDestination
	}
	if v.goBuildOutput {
		return goBuildGuestOutput
	}
	if v.goBuild {
		return goBuildGuestInput
	}
	return pythonSitePath
}

func (v *closureVolume) verify(ctx context.Context, runner CommandRunner) error {
	if v == nil || ctx == nil || runner == nil || v.name == "" || v.capacity <= 0 || len(v.manifestID) != 64 {
		return errors.New("python closure volume identity is unavailable")
	}
	body, err := runner.Output(ctx, "docker", "volume", "inspect", "--format", "{{json .}}", v.name)
	if err != nil || len(body) == 0 || len(body) > 16<<10 {
		return errors.New("python closure volume inspection failed")
	}
	var actual dockerVolumeInspection
	if json.Unmarshal(body, &actual) != nil || actual.Name != v.name || actual.Driver != "local" ||
		actual.Scope != "local" || actual.Mountpoint == "" || actual.CreatedAt == "" ||
		actual.Options["type"] != "tmpfs" || actual.Options["device"] != "tmpfs" ||
		actual.Options["o"] != v.tmpfsOptions() ||
		actual.Labels[closureVolumeLabel] != v.transaction ||
		actual.Labels[closureManifestLabel] != v.manifestID || len(actual.Options) != 3 {
		return errors.New("python closure volume topology is untrusted")
	}
	if v.createdAt != "" && (v.createdAt != actual.CreatedAt || v.mountpoint != actual.Mountpoint) {
		return errors.New("python closure volume was replaced")
	}
	v.createdAt, v.mountpoint = actual.CreatedAt, actual.Mountpoint
	return nil
}

func (v closureVolume) mountArgument(readOnly bool) (string, error) {
	if v.name == "" || v.createdAt == "" || v.mountpoint == "" {
		return "", errors.New("python closure volume is not attested")
	}
	argument := "type=volume,source=" + v.name + ",target=" + v.destination() + ",volume-nocopy"
	if readOnly {
		argument += ",readonly"
	}
	return argument, nil
}

type dockerMountInspection struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Driver      string `json:"Driver"`
	RW          bool   `json:"RW"`
}

type dockerHostMountInspection struct {
	Type          string `json:"Type"`
	Source        string `json:"Source"`
	Target        string `json:"Target"`
	ReadOnly      bool   `json:"ReadOnly"`
	VolumeOptions struct {
		NoCopy bool `json:"NoCopy"`
	} `json:"VolumeOptions"`
}

type dockerContainerMountInspection struct {
	ID         string                      `json:"id"`
	Mounts     []dockerMountInspection     `json:"mounts"`
	HostMounts []dockerHostMountInspection `json:"host_mounts"`
	Binds      []string                    `json:"binds"`
}

const containerMountInspectTemplate = `{"id":{{json .Id}},"mounts":{{json .Mounts}},"host_mounts":{{json .HostConfig.Mounts}},"binds":{{json .HostConfig.Binds}}}`

func (v *closureVolume) verifyContainerMount(ctx context.Context, runner CommandRunner, containerID string, readOnly bool) error {
	if v == nil || !containerIDPattern.MatchString(containerID) || v.verify(ctx, runner) != nil {
		return errors.New("python closure volume identity changed")
	}
	body, err := runner.Output(ctx, "docker", "inspect", "--format", containerMountInspectTemplate, containerID)
	if err != nil || len(body) == 0 || len(body) > 16<<10 {
		return errors.New("python closure container mount inspection failed")
	}
	var actual dockerContainerMountInspection
	if json.Unmarshal(body, &actual) != nil || actual.ID != containerID || len(actual.Binds) != 0 {
		return errors.New("python closure container mount identity is invalid")
	}
	for _, mount := range actual.Mounts {
		if mount.Type == "bind" {
			return errors.New("python closure container has an unauthorized bind mount")
		}
	}
	for _, mount := range actual.HostMounts {
		if mount.Type == "bind" {
			return errors.New("python closure container requests an unauthorized bind mount")
		}
	}
	actualMatches, configuredMatches := 0, 0
	for _, mount := range actual.Mounts {
		if mount.Destination != v.destination() {
			continue
		}
		actualMatches++
		if mount.Type != "volume" || mount.Name != v.name || mount.Source != v.mountpoint ||
			mount.Driver != "local" || mount.RW == readOnly {
			return errors.New("python closure container mount is substituted or writable")
		}
	}
	for _, mount := range actual.HostMounts {
		if mount.Target != v.destination() {
			continue
		}
		configuredMatches++
		if mount.Type != "volume" || mount.Source != v.name || mount.ReadOnly != readOnly ||
			!mount.VolumeOptions.NoCopy {
			return errors.New("python closure container mount configuration is invalid")
		}
	}
	if actualMatches != 1 || configuredMatches != 1 {
		return fmt.Errorf("python closure container mount count mismatch: %d/%d", actualMatches, configuredMatches)
	}
	return nil
}

// verifyAttachments rejects a competing writer or a volume attachment that
// the controller did not create for this exact transaction. Docker's volume
// filter is used only as inventory; every returned full ID is checked again.
func (v *closureVolume) verifyAttachments(ctx context.Context, runner CommandRunner, authorized map[string]bool) error {
	if v == nil || v.verify(ctx, runner) != nil || len(authorized) == 0 {
		return errors.New("python closure volume attachment inventory is unavailable")
	}
	listed, err := runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "volume="+v.name, "--format", "{{.ID}}")
	if err != nil || len(listed) > 16<<10 {
		return errors.New("python closure volume attachment inventory failed")
	}
	seen := make(map[string]bool, len(authorized))
	for _, id := range strings.Fields(string(listed)) {
		readOnly, expected := authorized[id]
		if !exactObservationContainerID(id) || !expected || seen[id] || v.verifyContainerMount(ctx, runner, id, readOnly) != nil {
			return errors.New("python closure volume has an unauthorized attachment")
		}
		seen[id] = true
	}
	if len(seen) != len(authorized) {
		return errors.New("python closure volume is missing an authorized attachment")
	}
	return nil
}
