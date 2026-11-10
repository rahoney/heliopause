package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

// Account for the complete native project output before selecting its tree.
// Original controls never become the Artifact session's inspection manifest.
func bindBunInspectionProject(ctx context.Context, input artifactnpm.BunInspectionInput, inventories []artifactnpm.ArchiveInventory, body []byte) (artifactnpm.BunInspectionLayout, error) {
	if ctx == nil || !input.Valid() {
		return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
	}
	project, err := artifactnpm.ParsePNPMInspectionPayloadArchive(ctx, body)
	if err != nil {
		return artifactnpm.BunInspectionLayout{}, errors.Join(errBunInspectionPreparation, err)
	}
	controls := map[string][]byte{}
	for _, control := range input.Context().Controls() {
		if !control.Present() || control.Name() != "package.json" && control.Name() != "bun.lock" {
			return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
		}
		controls["project/"+control.Name()] = control.Body()
	}
	if len(controls) != 2 {
		return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
	}
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	seen := map[string]bool{}
	for _, entry := range project.Entries() {
		if err := ctx.Err(); err != nil {
			return artifactnpm.BunInspectionLayout{}, errors.Join(errBunInspectionPreparation, err, context.Cause(ctx))
		}
		if entry.Name == "project" {
			if entry.Kind != tar.TypeDir || entry.Mode&0o022 != 0 || entry.Mode&0o700 != 0o700 {
				return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
			}
			continue
		}
		if expected, control := controls[entry.Name]; control {
			if entry.Kind != tar.TypeReg || entry.Mode != 0o600 && entry.Mode != 0o644 {
				return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
			}
			actual, err := project.ReadFile(entry.Name)
			if err != nil || !bytes.Equal(actual, expected) {
				return artifactnpm.BunInspectionLayout{}, errors.Join(errBunInspectionPreparation, err)
			}
			seen[entry.Name] = true
			continue
		}
		if entry.Name != "project/node_modules" && !strings.HasPrefix(entry.Name, "project/node_modules/") {
			return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
		}
		if entry.Kind != tar.TypeReg && entry.Kind != tar.TypeDir && entry.Kind != tar.TypeSymlink {
			return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
		}
		header := tar.Header{Name: strings.TrimPrefix(entry.Name, "project/"), Typeflag: entry.Kind, Mode: entry.Mode, Uid: entry.UID, Gid: entry.GID, Size: entry.Size, Linkname: entry.Link}
		if err := writer.WriteHeader(&header); err != nil {
			return artifactnpm.BunInspectionLayout{}, err
		}
		if entry.Kind == tar.TypeReg {
			content, err := project.ReadFile(entry.Name)
			if err != nil {
				return artifactnpm.BunInspectionLayout{}, err
			}
			if len(content) > artifactnpm.PNPMLayoutArchiveLimit || output.Len() > artifactnpm.PNPMLayoutArchiveLimit-len(content) {
				return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
			}
			if _, err := writer.Write(content); err != nil {
				return artifactnpm.BunInspectionLayout{}, err
			}
		}
		if output.Len() > artifactnpm.PNPMLayoutArchiveLimit {
			return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
		}
	}
	if len(seen) != 2 {
		return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
	}
	if err := writer.Close(); err != nil {
		return artifactnpm.BunInspectionLayout{}, err
	}
	if output.Len() > artifactnpm.PNPMLayoutArchiveLimit {
		return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
	}
	layout, err := artifactnpm.ParseBunLayoutArchive(ctx, output.Bytes())
	if err != nil {
		return artifactnpm.BunInspectionLayout{}, err
	}
	return input.BindLayout(ctx, inventories, layout.RestrictDirectoryWrites())
}
