package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNodeExecutionInputsRenderOnlyCurrentOriginalTree(t *testing.T) {
	t.Parallel()
	data, inventories, entries := executionLayoutFixture(t, "")
	layout, err := bindExecutionFixture(t, data, inventories, entries)
	if err != nil {
		t.Fatal(err)
	}
	execution, _ := domain.NewExecutionContext(data)
	body, err := RenderNodeExecutionInputs(context.Background(), execution, layout)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]PNPMLayoutEntry{}
	for _, entry := range layout.Entries() {
		expected["project/"+entry.Name] = entry
	}
	reader := tar.NewReader(bytes.NewReader(body))
	seen := 0
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Name == "project" {
			if header.Typeflag != tar.TypeDir || header.Mode != 0o700 || header.Uid != 1000 || header.Gid != 1000 {
				t.Fatal("private project root changed")
			}
			continue
		}
		entry, ok := expected[header.Name]
		if !ok || header.Typeflag != entry.Kind || header.Mode != entry.Mode || header.Size != entry.Size || header.Uid != entry.UID || header.Gid != entry.GID || header.Linkname != "" {
			t.Fatal("extra or changed entry", header.Name)
		}
		if entry.Kind == tar.TypeReg {
			actual, err := io.ReadAll(reader)
			original, originalErr := layout.ReadFile(entry.Name)
			if err != nil || originalErr != nil || !bytes.Equal(actual, original) {
				t.Fatal("original file changed", entry.Name)
			}
		}
		delete(expected, header.Name)
		seen++
	}
	if len(expected) != 0 || seen != len(layout.Entries()) {
		t.Fatal("incomplete original tree")
	}
	data.Arguments = []string{"--private", "value with spaces"}
	otherArgs, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	again, err := RenderNodeExecutionInputs(context.Background(), otherArgs, layout)
	if err != nil || !bytes.Equal(again, body) {
		t.Fatal("argv entered DATA payload")
	}
}

func TestNodeExecutionInputsReadbackRejectsAnyChangedCensus(t *testing.T) {
	t.Parallel()
	data, inventories, entries := executionLayoutFixture(t, "")
	layout, err := bindExecutionFixture(t, data, inventories, entries)
	if err != nil {
		t.Fatal(err)
	}
	execution, _ := domain.NewExecutionContext(data)
	native, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, entries))
	if err != nil || CompareNodeExecutionInputs(context.Background(), execution, layout, native) != nil {
		t.Fatal("exact original census rejected", err)
	}
	for _, field := range []string{"bytes", "mode", "UID", "GID", "missing", "extra"} {
		changed := slices.Clone(entries)
		index := slices.IndexFunc(changed, func(e layoutTarTestEntry) bool { return e.header.Typeflag == tar.TypeReg })
		switch field {
		case "bytes":
			changed[index].body += "changed"
			changed[index].header.Size = int64(len(changed[index].body))
		case "mode":
			changed[index].header.Mode ^= 0o100
		case "UID":
			changed[index].header.Uid = 0
		case "GID":
			changed[index].header.Gid = 0
		case "missing":
			changed = append(changed[:index], changed[index+1:]...)
		case "extra":
			changed = append(changed, layoutTestEntry("node_modules/foreign", tar.TypeReg, "foreign", ""))
		}
		actual, parseErr := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, changed))
		if parseErr == nil && CompareNodeExecutionInputs(context.Background(), execution, layout, actual) == nil {
			t.Fatal("changed readback accepted", field)
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("input validation canceled")
	cancel(cause)
	if _, err := RenderNodeExecutionInputs(ctx, execution, layout); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal("renderer lost cancellation", err)
	}
	if err := CompareNodeExecutionInputs(ctx, execution, layout, native); !errors.Is(err, cause) {
		t.Fatal("readback lost cancellation", err)
	}
	data.RuntimeIdentity = "different-runtime"
	foreign, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RenderNodeExecutionInputs(context.Background(), foreign, layout); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("foreign execution binding accepted or leaked", err)
	}
}
