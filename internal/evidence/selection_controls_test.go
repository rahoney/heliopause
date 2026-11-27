package evidence_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
)

func TestManifestIdentityBindsOpaqueSelectionControls(t *testing.T) {
	set := verifiedFixture(t)
	op, _ := domain.NewOperationID()
	target, _ := domain.NewInstallTarget("/tmp/haa-selection-target")
	install, _ := domain.NewInstallContext(target)
	digest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	ctx := evidence.ManifestContext{OperationID: op, InstallContext: install, ResolverRuntime: "fixture-runtime", LockfileDigest: digest}
	old, err := evidence.BuildVerifiedBundle(ctx, set)
	if err != nil {
		t.Fatal(err)
	}
	control, _ := domain.NewProjectControlFile("package.json", []byte(`{"dependencies":{"first":"^1"}}`), true)
	resolution, _ := domain.NewDependencyResolution(set.Inspected().Graph(), "fixture-runtime", digest)
	resolution, err = resolution.WithProjectControls([]domain.ProjectControlFile{control})
	if err != nil {
		t.Fatal(err)
	}
	ctx.ProjectControls = resolution.ProjectControls()
	bound, err := evidence.BuildVerifiedBundle(ctx, set)
	if err != nil {
		t.Fatal(err)
	}
	if old.ManifestID() == bound.ManifestID() || bytes.Contains(old.ManifestDocument(), []byte("project_controls")) {
		t.Fatal("selection did not change identity or legacy bytes changed")
	}
	var doc struct {
		Controls []struct {
			Content []byte
			SHA256  string
		} `json:"project_controls"`
	}
	if json.Unmarshal(bound.ManifestDocument(), &doc) != nil || len(doc.Controls) != 1 || !bytes.Equal(doc.Controls[0].Content, control.Body()) || doc.Controls[0].SHA256 != control.Digest().String() {
		t.Fatal("exact selection bytes/digest missing")
	}
	copy := resolution.ProjectControls()
	copy[0] = domain.ProjectControlFile{}
	body := control.Body()
	body[0] = 'x'
	if resolution.ProjectControls()[0].Name() != "package.json" || resolution.ProjectControls()[0].Body()[0] != '{' {
		t.Fatal("selection controls mutable through caller")
	}
	if _, err := resolution.WithProjectControls([]domain.ProjectControlFile{control, control}); err == nil {
		t.Fatal("duplicate controls accepted")
	}
	if _, err := resolution.WithProjectControls([]domain.ProjectControlFile{{}}); err == nil {
		t.Fatal("invalid controls accepted")
	}
}
