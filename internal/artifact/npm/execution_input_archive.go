package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errNodeExecutionInputs = errors.New("node execution input differs from its current complete layout")

// RenderNodeExecutionInputs emits only the already-bound original tree as
// relative DATA. It grants no immutable mount or runtime execution authority.
func RenderNodeExecutionInputs(ctx context.Context, execution domain.ExecutionContext, layout NodeExecutionLayout) ([]byte, error) {
	if ctx == nil || !layout.MatchesExecution(execution) {
		return nil, errNodeExecutionInputs
	}
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	if err := writer.WriteHeader(&tar.Header{Name: "project", Typeflag: tar.TypeDir, Mode: 0o700, Uid: 1000, Gid: 1000}); err != nil {
		return nil, errors.Join(errNodeExecutionInputs, err)
	}
	for _, entry := range layout.Entries() {
		if ctx.Err() != nil {
			return nil, errors.Join(errNodeExecutionInputs, ctx.Err(), context.Cause(ctx))
		}
		if entry.Kind != tar.TypeDir && entry.Kind != tar.TypeReg || entry.Link != "" {
			return nil, errNodeExecutionInputs
		}
		if err := writer.WriteHeader(&tar.Header{Name: "project/" + entry.Name, Typeflag: entry.Kind, Mode: entry.Mode, Size: entry.Size, Uid: entry.UID, Gid: entry.GID}); err != nil {
			return nil, errors.Join(errNodeExecutionInputs, err)
		}
		if entry.Kind == tar.TypeReg {
			body, err := layout.ReadFile(entry.Name)
			if err != nil || int64(len(body)) != entry.Size || len(body) > PNPMInspectionWorkspaceLimit-output.Len() {
				return nil, errors.Join(errNodeExecutionInputs, err)
			}
			if _, err := writer.Write(body); err != nil {
				return nil, errors.Join(errNodeExecutionInputs, err)
			}
		}
		if output.Len() > PNPMInspectionWorkspaceLimit {
			return nil, errNodeExecutionInputs
		}
	}
	if err := writer.Close(); err != nil || output.Len() > PNPMInspectionWorkspaceLimit {
		return nil, errors.Join(errNodeExecutionInputs, err)
	}
	return output.Bytes(), nil
}

// CompareNodeExecutionInputs reuses the bounded native parser's measured
// file hashes and full metadata. Transport ordering is not source identity.
func CompareNodeExecutionInputs(ctx context.Context, execution domain.ExecutionContext, layout NodeExecutionLayout, native PNPMLayoutArchive) error {
	if ctx == nil || !layout.MatchesExecution(execution) || len(native.entries) != len(layout.layout.entries) {
		return errNodeExecutionInputs
	}
	expected := make(map[string]PNPMLayoutEntry, len(layout.layout.entries))
	for _, entry := range layout.layout.entries {
		expected[entry.Name] = entry
	}
	for _, entry := range native.entries {
		if ctx.Err() != nil {
			return errors.Join(errNodeExecutionInputs, ctx.Err(), context.Cause(ctx))
		}
		if original, present := expected[entry.Name]; !present || original != entry {
			return errNodeExecutionInputs
		}
		delete(expected, entry.Name)
	}
	if len(expected) != 0 {
		return errNodeExecutionInputs
	}
	return nil
}
