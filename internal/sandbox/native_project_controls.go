package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func copyNativeProjectControls(ctx context.Context, runner CommandRunner, container string, kind nativeRuntimeToolKind, controls []domain.ProjectControlFile) error {
	spec, known := nativeProjectIdentity(kind)
	if !known {
		return errPNPMControls
	}
	input, ok := runner.(inputCommandRunner)
	if ctx == nil || !ok || !exactObservationContainerID(container) || len(controls) < 1 || len(controls) > 2 {
		return errPNPMControls
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(errPNPMControls, err)
	}
	values := map[string][]byte{}
	seen := map[string]bool{}
	for _, control := range controls {
		name, value := control.Name(), control.Body()
		if name != "package.json" && name != spec.lock || seen[name] || len(value) > pnpmControlLimit {
			return errPNPMControls
		}
		seen[name] = true
		if control.Present() {
			if len(value) == 0 {
				return errPNPMControls
			}
			values[name] = value
		}
	}
	if len(values["package.json"]) == 0 {
		return errPNPMControls
	}
	var body bytes.Buffer
	archive := tar.NewWriter(&body)
	if err := archive.WriteHeader(&tar.Header{Name: "project/", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
		return errors.Join(errPNPMControls, err)
	}
	for _, name := range []string{"package.json", spec.lock} {
		value, exists := values[name]
		if !exists {
			continue
		}
		if err := archive.WriteHeader(&tar.Header{Name: "project/" + name, Typeflag: tar.TypeReg, Mode: 0600, Uid: 1000, Gid: 1000, Size: int64(len(value))}); err != nil {
			return errors.Join(errPNPMControls, err)
		}
		if _, err := archive.Write(value); err != nil {
			return errors.Join(errPNPMControls, err)
		}
	}
	if kind == nativeRuntimeBun {
		for _, name := range []string{"cache/", "tmp/"} {
			if err := archive.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
				return errors.Join(errPNPMControls, err)
			}
		}
	}
	if err := archive.Close(); err != nil {
		return errors.Join(errPNPMControls, err)
	}
	if err := input.RunInput(ctx, &body, "docker", "cp", "--archive", "-", container+":"+spec.root); err != nil || ctx.Err() != nil {
		return errors.Join(errPNPMControls, err, ctx.Err())
	}
	return nil
}
