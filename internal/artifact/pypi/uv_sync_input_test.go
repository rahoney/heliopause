package pypi

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUVSyncInputSeparatesSkipAndExplicitEmpty(t *testing.T) {
	empty, err := ParseUVProjectLock(uvProjectBody(nil), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	var prior []domain.ProjectControlFile
	for _, item := range []struct {
		format      UVSyncSourceFormat
		body        []byte
		allow, skip bool
	}{
		{UVSyncRequirements, nil, false, true},
		{UVSyncRequirements, nil, true, false},
		{UVSyncRequirements, []byte(" \r\n# 설명\n"), false, true},
		{UVSyncPylock, empty.body, false, false},
	} {
		input, err := ParseUVSyncInput(item.format, item.body, item.allow, "3.14.7")
		if err != nil || !input.Valid() || input.ShouldSkip() != item.skip || input.SourceFormat() != item.format || input.AllowEmptyRequirements() != item.allow {
			t.Fatal("empty source intent changed", item, err)
		}
		if err := input.ValidateSelection(empty); (err != nil) != item.skip {
			t.Fatal("skip became positive empty selection", err)
		}
		controls, err := input.Controls()
		if err != nil || len(controls) != 2 || !controls[0].Present() {
			t.Fatal("present empty source was lost", err)
		}
		parsed, err := ParseUVSyncControls([]domain.ProjectControlFile{controls[1], controls[0]}, "3.14.7")
		if err != nil || parsed.ShouldSkip() != input.ShouldSkip() || !bytes.Equal(parsed.Body(), input.Body()) {
			t.Fatal("frozen source roundtrip changed intent", err)
		}
		if prior == nil {
			prior = controls
		} else if item.format == UVSyncRequirements && item.allow && prior[1].Digest() == controls[1].Digest() {
			t.Fatal("explicit empty permission reused skip digest")
		}
	}
	if (UVSyncInput{}).Valid() || (UVSyncInput{}).ShouldSkip() {
		t.Fatal("zero source acquired skip authority")
	}
	if _, err := (UVSyncInput{}).Controls(); err == nil {
		t.Fatal("zero source became frozen input")
	}
	if err := (UVSyncInput{}).ValidateSelection(empty); err == nil {
		t.Fatal("zero source became positive empty")
	}
}

func TestUVSyncInputRequiresOriginalCompletePinnedSelection(t *testing.T) {
	single, inputs, pages := uvTestSelection(t, "child>=1.0")
	lock, err := ParseUVProjectLock(uvProjectBody(single.Packages()), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(" # コメント\r\nParent == 1.0.0\nchild==1.0  # retained pin\n")
	input, err := ParseUVSyncInput(UVSyncRequirements, body, false, "3.14.7")
	if err != nil || input.ShouldSkip() || input.ValidateSelection(lock) != nil {
		t.Fatal("exact PEP440 pins changed", err)
	}
	body[0] = 'x'
	exported := input.Body()
	exported[0] = 'y'
	if input.Body()[0] != ' ' {
		t.Fatal("source bytes were mutable")
	}
	for _, source := range []string{"parent==1.0\n", "parent==2.0\nchild==1.0\n", "parent==1.0\nother==1.0\n", "parent==1.0\nchild==1.0\nextra==1.0\n"} {
		candidate, err := ParseUVSyncInput(UVSyncRequirements, []byte(source), false, "3.14.7")
		if err != nil {
			t.Fatal(err)
		}
		if candidate.ValidateSelection(lock) == nil {
			t.Fatal("native selection changed source pins", source)
		}
	}
	wrong, _ := ParseUVProjectLock(uvProjectBody(single.Packages()), "3.14.8")
	if input.ValidateSelection(wrong) == nil || input.ValidateSelection(UVProjectLock{}) == nil {
		t.Fatal("target/zero selection accepted")
	}
	// Source matching is not dependency closure approval. The common binder
	// must independently reject the authenticated parent without its child.
	parentOnly := single.Packages()[1:]
	partial, err := ParseUVProjectLock(uvProjectBody(parentOnly), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	root, err := ParseUVSyncInput(UVSyncRequirements, []byte("parent==1.0\n"), false, "3.14.7")
	if err != nil || root.ValidateSelection(partial) != nil {
		t.Fatal("root-only native meaning changed", err)
	}
	controls, _ := root.Controls()
	native, _ := partial.Control()
	controls = append(controls, native)
	if _, err := partial.BindProject(uvProjectContext(t, "/tmp/uv-sync-source"), controls, inputs[:1], pages[:1], WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits()); err == nil {
		t.Fatal("incomplete original source bypassed full dependency closure")
	}
}

func TestUVSyncPylockMustRemainTheSameFrozenSource(t *testing.T) {
	single, _, _ := uvTestSelection(t, "child>=1.0")
	body := uvProjectBody(single.Packages())
	input, err := ParseUVSyncInput(UVSyncPylock, body, false, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseUVProjectLock(body, "3.14.7")
	if err != nil || input.ValidateSelection(lock) != nil {
		t.Fatal(err)
	}
	changed, err := ParseUVProjectLock(append(append([]byte(nil), body...), []byte("\n# changed\n")...), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	if input.ValidateSelection(changed) == nil {
		t.Fatal("native pylock source changed after capture")
	}
	if _, err := ParseUVSyncInput(UVSyncPylock, nil, true, "3.14.7"); err == nil {
		t.Fatal("missing pylock became explicit empty")
	}
}

func TestUVSyncInputRejectsUnsupportedGrammarAndBudgets(t *testing.T) {
	for _, source := range []string{
		"parent", "parent>=1.0", "parent==1.*", "parent===1.0", "parent==1.0rc1", "parent==1.0.dev1", "parent==1.0+local",
		"parent[extra]==1.0", "parent==1.0; sys_platform=='linux'", "parent @ https://example.org/a.whl", "file:///tmp/a.whl", "./project", "-e ./project", "-r requirements.txt", "-c constraints.txt", "--index-url=https://example.org/", "parent==1.0 --hash=sha256:abc", "parent==1.0\\\n", "parent==1.0#comment", "parent==1.0\nParent==1.0", "parent_name==1.0\nparent-name==1.0", "parent==1.0\x00", "# bad\xff",
	} {
		t.Run(source, func(t *testing.T) {
			if _, err := ParseUVSyncInput(UVSyncRequirements, []byte(source), false, "3.14.7"); err == nil {
				t.Fatal("unsupported source accepted")
			}
		})
	}
	var maximum strings.Builder
	for i := 0; i < PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts(); i++ {
		fmt.Fprintf(&maximum, "probe-%d==1.0\n", i)
	}
	if _, err := ParseUVSyncInput(UVSyncRequirements, []byte(maximum.String()), false, "3.14.7"); err != nil {
		t.Fatal("canonical source budget rejected", err)
	}
	if _, err := ParseUVSyncInput(UVSyncRequirements, []byte(maximum.String()+"extra==1.0\n"), false, "3.14.7"); err == nil {
		t.Fatal("source project budget exceeded")
	}
	if _, err := ParseUVSyncInput(UVSyncRequirements, bytes.Repeat([]byte("#"), maxReportBytes+1), false, "3.14.7"); err == nil {
		t.Fatal("source byte budget exceeded")
	}
	for _, python := range []string{"", "3.14", "3.14.7.1", "3.14.7rc1", "03.14.7"} {
		if _, err := ParseUVSyncInput(UVSyncRequirements, nil, true, python); err == nil {
			t.Fatal("unsupported Python target", python)
		}
	}
	if _, err := ParseUVSyncInput("other", nil, true, "3.14.7"); err == nil {
		t.Fatal("unknown source format accepted")
	}
}

func TestUVSyncControlsRejectAmbiguousIntentAndMissingSource(t *testing.T) {
	input, err := ParseUVSyncInput(UVSyncRequirements, []byte("parent==1.0\n"), false, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	controls, _ := input.Controls()
	absent, _ := domain.NewProjectControlFile(UVSyncSourceControlName, nil, false)
	unknown, _ := domain.NewProjectControlFile("unknown", nil, true)
	for _, bad := range [][]domain.ProjectControlFile{nil, controls[:1], {controls[0], controls[0]}, {absent, controls[1]}, {unknown, controls[1]}} {
		if _, err := ParseUVSyncControls(bad, "3.14.7"); err == nil {
			t.Fatal("ambiguous frozen source accepted")
		}
	}
	for _, body := range []string{
		`{"version":2,"format":"requirements","allow_empty_requirements":false}`,
		`{"version":1,"format":"requirements","allow_empty_requirements":false,"unknown":1}`,
		`{"version":1,"format":"requirements","allow_empty_requirements":false,"allow_empty_requirements":true}`,
		`{"version":1,"format":"requirements"}`, `null`,
	} {
		intent, _ := domain.NewProjectControlFile(UVSyncIntentControlName, []byte(body), true)
		if _, err := ParseUVSyncControls([]domain.ProjectControlFile{controls[0], intent}, "3.14.7"); err == nil {
			t.Fatal("noncanonical intent accepted", body)
		}
	}
}
