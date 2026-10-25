package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

func bunBoundPayloadFixture(t *testing.T) artifactnpm.BunInspectionLayout {
	t.Helper()
	_, input, inventories := bunPreparationInputFixture(t, BunResolverRuntimeIdentity())
	name := input.Artifacts()[0].Identity().Name()
	manifest := inventories[0].Manifest()
	files := []pnpmPayloadTestFile{
		{header: tar.Header{Name: "node_modules", Typeflag: tar.TypeDir, Mode: 0o755, Uid: 1000, Gid: 1000}},
		{header: tar.Header{Name: "node_modules/" + name, Typeflag: tar.TypeDir, Mode: 0o755, Uid: 1000, Gid: 1000}},
		{header: tar.Header{Name: "node_modules/" + name + "/package.json", Typeflag: tar.TypeReg, Mode: 0o644, Uid: 1000, Gid: 1000, Size: int64(len(manifest))}, body: manifest},
	}
	archive, err := artifactnpm.ParseBunLayoutArchive(context.Background(), pnpmPayloadTestTar(t, files))
	if err != nil {
		t.Fatal(err)
	}
	layout, err := input.BindLayout(context.Background(), inventories, archive)
	if err != nil || !layout.MatchesInput(input) {
		t.Fatal("complete Bun payload fixture", err)
	}
	return layout
}

func TestBunInspectionPayloadKeepsOnlyBoundExecutionData(t *testing.T) {
	layout := bunBoundPayloadFixture(t)
	payload, err := prepareBunInspectionPayload(context.Background(), layout)
	if err != nil || payload.kind != nativeRuntimeBun || payload.layout != layout.Digest() {
		t.Fatal(err)
	}
	repeated, err := prepareBunInspectionPayload(context.Background(), layout)
	if err != nil || !bytes.Equal(payload.body, repeated.body) {
		t.Fatal("unstable renderer", err)
	}
	parsed, err := artifactnpm.ParsePNPMInspectionPayloadArchive(context.Background(), payload.body)
	if err != nil || comparePNPMInspectionPayload(context.Background(), payload, parsed) != nil {
		t.Fatal("complete readback", err)
	}
	manifest, err := parsed.ReadFile("project/package.json")
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(manifest, &fields) != nil || len(fields) != 2 || fields["private"] == nil || fields["dependencies"] == nil {
		t.Fatal("original root settings entered execution", string(manifest))
	}
	if !bytes.Equal(manifest, layout.InspectionManifest()) || len(parsed.Entries()) != len(layout.Entries())+2 {
		t.Fatal("incomplete execution data")
	}
}

func TestBunInspectionPayloadRejectsWholeReadbackChanges(t *testing.T) {
	payload, err := prepareBunInspectionPayload(context.Background(), bunBoundPayloadFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"bytes", "missing", "extra", "mode", "owner", "alias"} {
		t.Run(fault, func(t *testing.T) {
			files := pnpmPayloadFiles(t, payload.body)
			for index := range files {
				if files[index].header.Name != "project/package.json" {
					continue
				}
				switch fault {
				case "bytes":
					files[index].body = append(files[index].body, '\n')
					files[index].header.Size++
				case "missing":
					files = append(files[:index], files[index+1:]...)
				case "mode":
					files[index].header.Mode = 0o644
				case "owner":
					files[index].header.Uid = 0
				}
				break
			}
			if fault == "extra" {
				files = append(files, pnpmPayloadTestFile{header: tar.Header{Name: "project/bun.lock", Typeflag: tar.TypeReg, Mode: 0o600, Uid: 1000, Gid: 1000, Size: 2}, body: []byte("{}")})
			}
			if fault == "alias" {
				files = append(files, pnpmPayloadTestFile{header: tar.Header{Name: "project/unknown", Typeflag: tar.TypeSymlink, Mode: 0o777, Uid: 1000, Gid: 1000, Linkname: "package.json"}})
			}
			actual, err := artifactnpm.ParsePNPMInspectionPayloadArchive(context.Background(), pnpmPayloadTestTar(t, files))
			if err == nil && comparePNPMInspectionPayload(context.Background(), payload, actual) == nil {
				t.Fatal("changed whole payload admitted")
			}
		})
	}
}

func TestBunInspectionPayloadTransportPreservesKindAndFirstCause(t *testing.T) {
	layout := bunBoundPayloadFixture(t)
	payload, err := prepareBunInspectionPayload(context.Background(), layout)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 64)
	runner := &pnpmControlTransport{body: payload.body}
	if err := copyBunInspectionPayload(context.Background(), runner, id, payload); err != nil || runner.calls != 2 {
		t.Fatal("complete transfer", err)
	}
	runner = &pnpmControlTransport{}
	if err := copyPNPMInspectionPayload(context.Background(), runner, id, payload); err == nil || runner.calls != 0 {
		t.Fatal("Bun payload crossed fixed root")
	}
	foreign, err := preparePNPMInspectionPayload(context.Background(), pnpmBoundPayloadFixture(t, true))
	if err != nil {
		t.Fatal(err)
	}
	if err := copyBunInspectionPayload(context.Background(), runner, id, foreign); err == nil || runner.calls != 0 {
		t.Fatal("PNPM payload crossed fixed root")
	}
	cause := errors.New("fixture Bun payload transport failure")
	runner = &pnpmControlTransport{err: cause}
	if err := copyBunInspectionPayload(context.Background(), runner, id, payload); !errors.Is(err, cause) || runner.calls != 1 {
		t.Fatal("first transport cause or dependent readback", err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	runner = &pnpmControlTransport{}
	if err := copyBunInspectionPayload(ctx, runner, id, payload); !errors.Is(err, cause) || runner.calls != 0 {
		t.Fatal("canceled transport", err)
	}
	if _, err := prepareBunInspectionPayload(ctx, layout); !errors.Is(err, cause) {
		t.Fatal("canceled renderer", err)
	}
	if _, err := prepareBunInspectionPayload(context.Background(), artifactnpm.BunInspectionLayout{}); err == nil {
		t.Fatal("zero layout admitted")
	}
	payload.body = append(append([]byte(nil), payload.body...), 'x')
	if err := copyBunInspectionPayload(context.Background(), runner, id, payload); err == nil || runner.calls != 0 {
		t.Fatal("changed serialized payload admitted")
	}
}
