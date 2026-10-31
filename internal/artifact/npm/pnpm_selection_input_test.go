package npm

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func pnpmInputControl(t *testing.T, name, body string, present bool) domain.ProjectControlFile {
	t.Helper()
	c, err := domain.NewProjectControlFile(name, []byte(body), present)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPNPMSelectionInputRejectsAuthorityAndUnknownIntentBeforeExecution(t *testing.T) {
	for _, test := range []struct {
		intent             PNPMSelectionIntent
		selector, manifest string
	}{
		{PNPMAdd, "file:../outside", `{}`}, {PNPMAdd, "--registry=evil", `{}`}, {PNPMAdd, "a@https://evil/a", `{}`},
		{PNPMAdd, "a@", `{}`}, {PNPMAdd, "", `{}`}, {PNPMInstall, "a", `{}`}, {PNPMUpdate, "foreign", pnpmSingleManifest},
		{PNPMUpdate, "is-number@7.0.0", pnpmSingleManifest}, {"exec", "a", `{}`},
		{PNPMInstall, "", `{"devEngines":{"runtime":{"name":"bun","version":"*","onFail":"download"}}}`},
		{PNPMInstall, "", `{"engines":{"runtime":{"name":"node","version":"*","onFail":"download"}}}`},
		{PNPMInstall, "", `{"packageManager":"pnpm@0.0.1"}`}, {PNPMInstall, "", `{"dependencies":{"a":"workspace:*"}}`},
	} {
		c := pnpmInputControl(t, "package.json", test.manifest, true)
		if _, err := NewPNPMSelectionInput(test.intent, test.selector, []domain.ProjectControlFile{c}, runtimeidentity.PNPMVersion); err == nil {
			t.Fatal("untrusted selection input accepted", test)
		}
	}
	c := pnpmInputControl(t, "package.json", `{}`, true)
	for _, controls := range [][]domain.ProjectControlFile{nil, {c, c}, {c, pnpmInputControl(t, ".npmrc", "registry=evil", true)}, {pnpmInputControl(t, "package.json", "", false)}, {c, pnpmInputControl(t, "pnpm-lock.yaml", pnpmEmptyLock+"---\n{}", true)}} {
		if _, err := NewPNPMSelectionInput(PNPMInstall, "", controls, runtimeidentity.PNPMVersion); err == nil {
			t.Fatal("invalid controls accepted")
		}
	}
}

func TestPNPMSelectionOutputBindsIntentAndPreservesOtherManifestFields(t *testing.T) {
	for _, test := range []struct {
		intent                  PNPMSelectionIntent
		selector, before, after string
		lock                    []byte
		valid                   bool
	}{
		{PNPMInstall, "", `{}`, `{}`, []byte(pnpmEmptyLock), true},
		{PNPMInstall, "", pnpmSingleManifest, "{\n\"dependencies\": {\"is-number\": \"6.0.0\"}}", pnpmFixture(t, "single"), true},
		{PNPMAdd, "is-number@6.0.0", `{}`, pnpmSingleManifest, pnpmFixture(t, "single"), true},
		{PNPMAdd, "is-number@5.0.0", `{}`, pnpmSingleManifest, pnpmFixture(t, "single"), false},
		{PNPMAdd, "other@1.0.0", `{}`, pnpmSingleManifest, pnpmFixture(t, "single"), false},
		{PNPMInstall, "", `{}`, pnpmSingleManifest, pnpmFixture(t, "single"), false},
		{PNPMUpdate, "is-number", pnpmSingleManifest, pnpmSingleManifest, pnpmFixture(t, "single"), true},
		{PNPMUpdate, "", pnpmSingleManifest, `{}`, []byte(pnpmEmptyLock), false},
		{PNPMAdd, "is-number@6.0.0", `{}`, `{"scripts":{"install":"malicious"},"dependencies":{"is-number":"6.0.0"}}`, pnpmFixture(t, "single"), false},
		{PNPMUpdate, "is-number", pnpmSingleManifest, `{"dependencies":{"is-number":"6.0.0"},"packageManager":"pnpm@` + runtimeidentity.PNPMVersion + `"}`, pnpmFixture(t, "single"), false},
	} {
		c := pnpmInputControl(t, "package.json", test.before, true)
		input, err := NewPNPMSelectionInput(test.intent, test.selector, []domain.ProjectControlFile{c}, runtimeidentity.PNPMVersion)
		if err != nil {
			t.Fatal(err)
		}
		selected, err := input.ParseOutput([]byte(test.after), test.lock)
		if (err == nil) != test.valid || err != nil && (selected.DependencyFree() || len(selected.Roots()) != 0) {
			t.Fatalf("output decision differs intent=%s selector=%s err=%v", test.intent, test.selector, err)
		}
	}
	manifest := `{"description":{"a":1,"b":[true,false]},"dependencies":{"is-number":"6.0.0"}}`
	input, err := NewPNPMSelectionInput(PNPMInstall, "", []domain.ProjectControlFile{pnpmInputControl(t, "package.json", manifest, true)}, runtimeidentity.PNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	control := input.Controls()
	control[0] = pnpmInputControl(t, "package.json", `{}`, true)
	if !bytes.Equal(input.Controls()[0].Body(), []byte(manifest)) {
		t.Fatal("input getter exposed mutable controls")
	}
	changed := strings.Replace(manifest, `"a":1,"b":[true,false]`, `"b":[true,false],"a":1`, 1)
	if _, err := input.ParseOutput([]byte(changed), pnpmFixture(t, "single")); err != nil {
		t.Fatal("formatting/order changed metadata", err)
	}
	if _, err := (PNPMSelectionInput{}).ParseOutput([]byte(`{}`), []byte(pnpmEmptyLock)); err == nil {
		t.Fatal("zero input produced selected state")
	}
}
