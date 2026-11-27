package npm

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func bunInputFixture(t *testing.T, intent BunSelectionIntent, selector, manifest string) BunSelectionInput {
	t.Helper()
	input, err := NewBunSelectionInput(intent, selector, []domain.ProjectControlFile{bunTestControl(t, "package.json", []byte(manifest), true)}, "1.4.4")
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestBunSelectionInputRejectsUnsupportedIntentAndAuthority(t *testing.T) {
	for _, test := range []struct {
		intent             BunSelectionIntent
		selector, manifest string
	}{
		{BunSelectionIntent("run"), "probe", `{}`}, {BunInstall, "probe", `{}`},
		{BunAdd, "--eval", `{}`}, {BunAdd, "file:../escape", `{}`}, {BunAdd, "https://evil.test/x", `{}`},
		{BunAdd, "probe@latest", `{}`}, {BunAdd, "probe@", `{}`}, {BunAdd, "probe\nother", `{}`},
		{BunUpdate, "missing", `{}`}, {BunUpdate, "probe@1.0.0", `{"dependencies":{"probe":"1.0.0"}}`},
		{BunInstall, "", `{"devEngines":{"runtime":{"name":"bun"}}}`},
		{BunInstall, "", `{"packageManager":"pnpm@12.15.0"}`},
	} {
		input, err := NewBunSelectionInput(test.intent, test.selector, []domain.ProjectControlFile{bunTestControl(t, "package.json", []byte(test.manifest), true)}, "1.4.4")
		if err == nil || len(input.Controls()) != 0 || input.ClientVersion() != "" {
			t.Fatal("unsupported request acquired typed input", test, err)
		}
	}
	for _, controls := range [][]domain.ProjectControlFile{
		{}, {bunTestControl(t, "package.json", nil, false)},
		{bunTestControl(t, "package.json", []byte(`{}`), true), bunTestControl(t, "pnpm-lock.yaml", []byte("foreign"), true)},
		{bunTestControl(t, "package.json", []byte(`{}`), true), bunTestControl(t, "package.json", []byte(`{}`), true)},
		{bunTestControl(t, "package.json", []byte(`{}`), true), bunTestControl(t, "bun.lock", nil, true)},
	} {
		if input, err := NewBunSelectionInput(BunInstall, "", controls, "1.4.4"); err == nil || len(input.Controls()) != 0 {
			t.Fatal("unknown/extra/empty controls accepted")
		}
	}
	input := bunInputFixture(t, BunAdd, "@scope/probe@^1.0.0", `{}`)
	copy := input.Controls()
	copy[0] = domain.ProjectControlFile{}
	if len(input.Controls()) != 2 || input.Controls()[0].Name() != "package.json" || input.Controls()[1].Present() {
		t.Fatal("input controls alias or absence differs")
	}
}

func TestBunSelectionOutputKeepsIntentAndNondependencyMetadata(t *testing.T) {
	manifest, body, present := bunActualControls(t, "single")
	control := bunTestControl(t, "bun.lock", body, present)
	install, err := NewBunSelectionInput(BunInstall, "", []domain.ProjectControlFile{bunTestControl(t, "package.json", manifest, true), control}, "1.4.4")
	if err != nil {
		t.Fatal(err)
	}
	if selected, err := install.ParseOutput(manifest, control); err != nil || len(selected.Packages()) != 1 {
		t.Fatal("unchanged frozen output rejected", err)
	}
	changed := bunTestControl(t, "bun.lock", append(bytes.Clone(body), '\n'), true)
	if selected, err := install.ParseOutput(manifest, changed); err == nil || selected.valid {
		t.Fatal("frozen lock bytes changed")
	}
	before := strings.Replace(string(manifest), `"dependencies":{"is-number":"6.0.0"}`, `"dependencies":{}`, 1)
	add := bunInputFixture(t, BunAdd, "is-number@6.0.0", before)
	if selected, err := add.ParseOutput(manifest, control); err != nil || len(selected.Roots()) != 1 {
		t.Fatal("exact add rejected", err)
	}
	wrong := bunInputFixture(t, BunAdd, "is-number@7.0.0", before)
	if selected, err := wrong.ParseOutput(manifest, control); err == nil || selected.valid {
		t.Fatal("exact add substituted version")
	}
	changedManifest := bytes.Replace(manifest, []byte(`exit 99`), []byte(`echo replaced`), 1)
	if selected, err := add.ParseOutput(changedManifest, control); err == nil || selected.valid {
		t.Fatal("native output replaced unrelated scripts")
	}
	if selected, err := (BunSelectionInput{}).ParseOutput(manifest, control); err == nil || selected.valid {
		t.Fatal("zero input interpreted output")
	}
	for _, intent := range []BunSelectionIntent{BunInstall, BunUpdate} {
		input := bunInputFixture(t, intent, "", string(manifest))
		if selected, err := input.ParseOutput(manifest, control); err != nil || len(selected.Roots()) != 1 {
			t.Fatal("install/update no-op rejected", intent, err)
		}
	}
}

func TestBunSelectionIntentKeepsUnselectedRootsAndRejectsDrops(t *testing.T) {
	before, _ := bunSyntheticLock(t, map[string]string{"a": "1.0.0", "b": "1.0.0"}, nil)
	selected, err := NewBunSelectionInput(BunUpdate, "a", []domain.ProjectControlFile{bunTestControl(t, "package.json", before, true)}, "1.4.4")
	if err != nil {
		t.Fatal(err)
	}
	full, err := NewBunSelectionInput(BunUpdate, "", selected.Controls(), "1.4.4")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name               string
		roots              map[string]string
		selectedOK, fullOK bool
	}{
		{"target", map[string]string{"a": "2.0.0", "b": "1.0.0"}, true, true},
		{"other", map[string]string{"a": "2.0.0", "b": "2.0.0"}, false, true},
		{"drop", map[string]string{"a": "2.0.0"}, false, false},
		{"extra", map[string]string{"a": "2.0.0", "b": "1.0.0", "c": "1.0.0"}, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows := map[string][]any{}
			for name, version := range test.roots {
				rows[name] = bunSyntheticRow(name, version, map[string]any{})
			}
			manifest, body := bunSyntheticLock(t, test.roots, rows)
			control := bunTestControl(t, "bun.lock", body, true)
			for _, check := range []struct {
				input BunSelectionInput
				want  bool
			}{{selected, test.selectedOK}, {full, test.fullOK}} {
				result, err := check.input.ParseOutput(manifest, control)
				if (err == nil) != check.want || !check.want && result.valid {
					t.Fatal("selection changed roots beyond its intent", err)
				}
			}
		})
	}
}
