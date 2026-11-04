package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func semgrepSupplyFixture(t *testing.T, seed ...semgrepMaterializationOwner) (string, string, semgrepMaterializationOwner) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repository := filepath.Join(base, "repository")
	input := filepath.Join(base, "input")
	collection := filepath.Join(base, "collection")
	for _, p := range []string{repository, input, collection} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	pin := semgrepMaterializationOwner{Version: "1.2.3", Architecture: "amd64", RuntimeImage: "python_image", SourceCommit: strings.Repeat("a", 40), SourceTagObject: strings.Repeat("b", 40), SourceDateEpoch: 1793297858}
	if len(seed) == 1 {
		pin.Version, pin.SourceCommit, pin.SourceTagObject, pin.SourceDateEpoch = seed[0].Version, seed[0].SourceCommit, seed[0].SourceTagObject, seed[0].SourceDateEpoch
	} else if len(seed) != 0 {
		t.Fatal("one supply identity seed is supported")
	}
	source := semgrepSourcePackageInput{Schema: "helox.semgrep-source-inputs/v1", SourceCommit: pin.SourceCommit, SourceDateEpoch: pin.SourceDateEpoch}
	asset := func(path string, body []byte) semgrepAsset {
		return semgrepAsset{Path: path, SHA256: hash(body), Size: int64(len(body))}
	}
	bodies := map[string][]byte{
		"patches/quarantine.patch": []byte("patch fixture\n"),
		"build/rebuild-worker.sh":  []byte("#!/bin/sh\nexit 0\n"),
		"build/sdk-prepare.sh":     []byte(semgrepSDKPrepare),
		"rules/heliopause.json":    []byte(`{"rules":[]}`),
		"notices/catalogue.json":   []byte(`{"components":[]}`),
	}
	notice := []byte("fixture notice\n")
	bodies["notices/"+hash(notice)+".txt"] = notice
	origins := map[string]any{"schema": "helox.semgrep-source-origins/v1", "source": map[string]any{"version": pin.Version, "tag": "v" + pin.Version, "tag_object": pin.SourceTagObject, "commit": pin.SourceCommit}}
	bodies["SOURCE-ORIGINS.json"], err = json.Marshal(origins)
	if err != nil {
		t.Fatal(err)
	}
	pin.Rules = asset("tools/semgrep/rules.json", bodies["rules/heliopause.json"])
	pin.Patch = asset("tools/semgrep/quarantine.patch", bodies["patches/quarantine.patch"])
	pin.Worker = asset("scripts/update-runtime/semgrep_build_worker.sh", bodies["build/rebuild-worker.sh"])
	for name, body := range bodies {
		p := filepath.Join(collection, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, body, 0600); err != nil {
			t.Fatal(err)
		}
		source.Files = append(source.Files, semgrepSourcePackageFile{name, hash(body), int64(len(body)), 0444})
	}
	sort.Slice(source.Files, func(i, j int) bool { return source.Files[i].Path < source.Files[j].Path })
	pin.SourceManifestSHA256 = hash(writeSemgrepSourceFixture(t, collection, source))
	for _, selected := range []struct {
		target  *semgrepAsset
		name    string
		notices bool
	}{{&pin.Sources, "semgrep-SOURCES.tar", false}, {&pin.Notices, "semgrep-NOTICES.tar", true}} {
		args := []string{"--input", collection, "--output", filepath.Join(input, selected.name)}
		if selected.notices {
			args = append(args, "--notices")
		}
		var out bytes.Buffer
		if err := packageSemgrepSource(args, &out); err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(filepath.Join(input, selected.name))
		if err != nil {
			t.Fatal(err)
		}
		*selected.target = asset(selected.name, body)
	}
	// Minimal ELF shape is DATA only; never executed by this supply test.
	native := make([]byte, 120)
	copy(native, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(native[16:], 2)
	binary.LittleEndian.PutUint16(native[18:], 62)
	binary.LittleEndian.PutUint32(native[20:], 1)
	binary.LittleEndian.PutUint64(native[32:], 64)
	binary.LittleEndian.PutUint16(native[52:], 64)
	binary.LittleEndian.PutUint16(native[54:], 56)
	binary.LittleEndian.PutUint16(native[56:], 1)
	binary.LittleEndian.PutUint32(native[64:], 0x6474e551)
	binary.LittleEndian.PutUint32(native[68:], 6)
	pin.Tool = asset("semgrep-core-linux-amd64", native)
	if err := os.WriteFile(filepath.Join(input, pin.Tool.Path), native, 0600); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		asset semgrepAsset
		body  []byte
	}{{pin.Rules, bodies["rules/heliopause.json"]}, {pin.Patch, bodies["patches/quarantine.patch"]}, {pin.Worker, bodies["build/rebuild-worker.sh"]}} {
		p := filepath.Join(repository, row.asset.Path)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, row.body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeSemgrepSupplyOwner(t, repository, pin)
	return repository, input, pin
}

func writeSemgrepSupplyOwner(t *testing.T, repository string, pin semgrepMaterializationOwner) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"schema_version": 6, "semgrep_tool": pin})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(repository, runtimeFile)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, body, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSemgrepSupplyPublishesOnlyBoundPassiveSet(t *testing.T) {
	repository, input, pin := semgrepSupplyFixture(t)
	output := filepath.Join(filepath.Dir(repository), "output")
	var report bytes.Buffer
	if err := materializeSemgrep([]string{"--root", repository, "--input", input, "--output", output}, &report); err != nil {
		t.Fatal(err)
	}
	for _, selected := range []struct {
		name, digest string
		mode         os.FileMode
	}{{pin.Tool.Path, pin.Tool.SHA256, 0555}, {pin.Sources.Path, pin.Sources.SHA256, 0444}, {pin.Notices.Path, pin.Notices.SHA256, 0444}, {"semgrep-rules.json", pin.Rules.SHA256, 0444}} {
		p := filepath.Join(output, selected.name)
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Lstat(p)
		if err != nil || info.Mode().Perm() != selected.mode || hash(body) != selected.digest {
			t.Fatal("supply output identity differs", selected.name, err)
		}
	}
	var receipt struct {
		RuntimeQualified bool   `json:"runtime_qualified"`
		Schema           string `json:"schema"`
	}
	if json.Unmarshal(report.Bytes(), &receipt) != nil || receipt.RuntimeQualified || receipt.Schema != "helox.semgrep-supply/v1" {
		t.Fatal("passive supply became runtime admission")
	}
	if err := materializeSemgrep([]string{"--root", repository, "--input", input, "--output", output}, &report); err == nil {
		t.Fatal("existing output was overwritten")
	}
}

func TestSemgrepSupplyRejectsChangedInputsBeforeOutput(t *testing.T) {
	for _, kind := range []string{"native", "source", "notices", "rules", "patch", "worker", "missing notice", "role mix", "manifest", "schema"} {
		t.Run(kind, func(t *testing.T) {
			repository, input, pin := semgrepSupplyFixture(t)
			output := filepath.Join(filepath.Dir(repository), "output")
			var target string
			switch kind {
			case "native":
				target = filepath.Join(input, pin.Tool.Path)
			case "source":
				target = filepath.Join(input, pin.Sources.Path)
			case "notices":
				target = filepath.Join(input, pin.Notices.Path)
			case "rules":
				target = filepath.Join(repository, pin.Rules.Path)
			case "patch":
				target = filepath.Join(repository, pin.Patch.Path)
			case "worker":
				target = filepath.Join(repository, pin.Worker.Path)
			case "missing notice":
				if err := os.Remove(filepath.Join(input, pin.Notices.Path)); err != nil {
					t.Fatal(err)
				}
			case "role mix":
				pin.Tool.Path = pin.Sources.Path
				writeSemgrepSupplyOwner(t, repository, pin)
			case "manifest":
				pin.SourceManifestSHA256 = strings.Repeat("0", 64)
				writeSemgrepSupplyOwner(t, repository, pin)
			case "schema":
				p := filepath.Join(repository, runtimeFile)
				body, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				body = bytes.Replace(body, []byte(`"schema_version":6`), []byte(`"schema_version":5`), 1)
				if err := os.WriteFile(p, body, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if target != "" {
				if err := os.Chmod(target, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var report bytes.Buffer
			if err := materializeSemgrep([]string{"--root", repository, "--input", input, "--output", output}, &report); err == nil {
				t.Fatal("changed supply was accepted")
			}
			if _, err := os.Lstat(output); !os.IsNotExist(err) {
				t.Fatal("invalid supply wrote output", err)
			}
			if report.Len() != 0 {
				t.Fatal("invalid supply emitted a success receipt")
			}
		})
	}
}

func TestSemgrepFrozenSupplyBindsPlanDocumentsAndFiles(t *testing.T) {
	repository, input, pin := semgrepSupplyFixture(t)
	owner, err := os.ReadFile(filepath.Join(repository, runtimeFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(input, "owner.json"), owner, 0600); err != nil {
		t.Fatal(err)
	}
	state := t.TempDir()
	m, err := freezeSemgrepSupply(pin.Version, input, state)
	if err != nil {
		t.Fatal(err)
	}
	if m.Semgrep == nil || len(m.Semgrep.Documents) != 3 {
		t.Fatal("frozen recipe/rules were lost")
	}
	if err := verifyFrozenSemgrepSupply(pin.Version, m, state); err != nil {
		t.Fatal(err)
	}
	original := m.Semgrep.Documents[pin.Rules.Path]
	m.Semgrep.Documents[pin.Rules.Path] = []byte("injected")
	if err := verifyFrozenSemgrepSupply(pin.Version, m, state); err == nil {
		t.Fatal("plan document injection accepted")
	}
	m.Semgrep.Documents[pin.Rules.Path] = original
	p := filepath.Join(state, "semgrep-input", pin.Sources.Path)
	if err := os.Chmod(p, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyFrozenSemgrepSupply(pin.Version, m, state); err == nil {
		t.Fatal("frozen source substitution accepted")
	}
}
