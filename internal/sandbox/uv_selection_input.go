package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"path"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type uvInstalledSelection struct {
	input    artifactpypi.UpgradeInput
	intent   artifactpypi.UVSelectionIntent
	proposal artifactpypi.UVSelection
}

// This implements the existing update resolver port. The target is never
// mounted or executed; only bounded, frozen METADATA enters native selection.
func (r *UVResolver) ResolveDependencyUpdate(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext, controls []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	if r == nil || r.reader == nil || ctx == nil || reference.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return domain.DependencyResolution{}, errors.New("UV installed-aware selection requires public input")
	}
	if err := ctx.Err(); err != nil {
		return domain.DependencyResolution{}, err
	}
	input, intent, err := artifactpypi.ParseUVSelectionRequest(controls, PinnedPythonRuntime().PythonVersion)
	if err != nil || reference != input.Requirement().Reference() {
		return domain.DependencyResolution{}, errors.New("UV original input differs from request")
	}
	selected := &uvInstalledSelection{input: input, intent: intent}
	native, err := r.resolveNativeSelectionWithInput(ctx, reference, input.Requirement().Requirement(), selected)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	resolution, err := r.reader.bindUVNativeSelection(ctx, reference, native)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	if err := selected.proposal.ValidateGraph(resolution.Graph()); err != nil {
		return domain.DependencyResolution{}, err
	}
	original, err := input.Control()
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	intentControl, err := artifactpypi.NewUVSelectionIntentControl(intent)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	delta, err := domain.NewProjectControlFile(artifactpypi.UVSelectionDeltaControlName, selected.proposal.Delta(), true)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	lock, err := domain.NewProjectControlFile(artifactpypi.UVSelectionLockControlName, native.lock, true)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return resolution.WithProjectControls([]domain.ProjectControlFile{original, intentControl, delta, lock})
}

func uvOriginalSelectionFiles(input artifactpypi.UpgradeInput) map[string][]byte {
	files := map[string][]byte{}
	for _, file := range input.MetadataFiles() {
		files["original/"+file.Directory+"/METADATA"] = file.Body
	}
	return files
}

// Only controller-created paths below the existing private output mount are
// transferred. --archive preserves the fixed ordinary-user ownership.
func introduceUVSelectionFiles(ctx context.Context, runner inputCommandRunner, container string, files map[string][]byte, original bool) error {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) {
		return errors.New("UV selection input transport unavailable")
	}
	var output bytes.Buffer
	archive := tar.NewWriter(&output)
	directories := map[string]bool{}
	if original {
		directories["original"] = true
	}
	var names []string
	var total int
	for name, body := range files {
		if !validClosureDestination(name) || name != "constraints.txt" && !(strings.HasPrefix(name, "original/") && strings.Count(name, "/") == 2 && strings.HasSuffix(name, ".dist-info/METADATA")) || len(body) == 0 || len(body) > uvNativeLockLimit-total {
			return errors.New("UV selection input path or bytes exceed bounds")
		}
		total += len(body)
		names = append(names, name)
		if name != "constraints.txt" {
			directories[path.Dir(name)] = true
			directories["original"] = true
		}
	}
	if len(names) > artifactpypi.UpgradeInputDistributionLimit {
		return errors.New("UV selection metadata count exceeds bounds")
	}
	var dirs []string
	for name := range directories {
		dirs = append(dirs, name)
	}
	sort.Strings(dirs)
	sort.Strings(names)
	for _, name := range dirs {
		mode := int64(0500)
		if name == "original" {
			mode = 0700
		}
		if err := archive.WriteHeader(&tar.Header{Name: name + "/", Typeflag: tar.TypeDir, Mode: mode, Uid: 1000, Gid: 1000}); err != nil {
			return err
		}
	}
	for _, name := range names {
		body := files[name]
		if err := archive.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0400, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
			return err
		}
		if _, err := archive.Write(body); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	return runner.RunInput(ctx, bytes.NewReader(output.Bytes()), "docker", "cp", "--archive", "-", container+":"+pythonSitePath)
}

func uvBoundedSelectionOutput(ctx context.Context, runner CommandRunner, container string, arguments []string) ([]byte, error) {
	stream, ok := runner.(interface {
		RunOutput(context.Context, io.Writer, string, ...string) error
	})
	if !ok {
		return nil, errors.New("UV bounded output transport unavailable")
	}
	writer := &goResolverBoundedOutput{limit: uvNativeLockLimit}
	if err := stream.RunOutput(ctx, writer, "docker", boundaryExecArguments(container, boundaryLaunchMode, arguments...)...); err != nil || writer.exceeded {
		return nil, uvReaderFailure{"UV native dry-run failed or exceeded output bounds", err}
	}
	return append([]byte(nil), writer.Bytes()...), nil
}
