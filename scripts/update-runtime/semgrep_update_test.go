package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSemgrepFrozenPlanAppliesOnlyOwnedChanges(t *testing.T) {
	root, state := fixture(t)
	repository, input, pin := semgrepSupplyFixture(t)
	owner, err := os.ReadFile(filepath.Join(repository, runtimeFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(input, "owner.json"), owner, 0600); err != nil {
		t.Fatal(err)
	}
	var reviews struct {
		Schema  int              `json:"schema_version"`
		Records []map[string]any `json:"records"`
	}
	if err := loadJSON(filepath.Join(root, supportFile), &reviews); err != nil {
		t.Fatal(err)
	}
	var review []map[string]any
	for _, r := range reviews.Records {
		if r["component"] == "Semgrep CE" {
			r["pinned_version"] = pin.Version
			r["checked_at"] = time.Now().UTC().Format("2006-01-02")
			r["latest_stable_at_check"] = pin.Version
			r["reason"] = "Synthetic supply-plan fixture only; no actual upstream/support/runtime qualification"
			review = append(review, r)
		}
	}
	reviewPath := filepath.Join(filepath.Dir(state), "review.json")
	body, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reviewPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	p, err := makePlanResolving(root, state, "semgrep", pin.Version, reviewPath, func(component, version string) (metadata, error) {
		if component != "semgrep" {
			t.Fatal("unrelated resolver called")
		}
		return freezeSemgrepSupply(version, input, state)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changes) != 6 || len(p.RequiredQualification) == 0 || p.Metadata.Semgrep == nil {
		t.Fatal("missing owned changes or qualification boundary", len(p.Changes))
	}
	for _, c := range p.Changes {
		switch c.Path {
		case runtimeFile, supportFile, generatedFile, pin.Rules.Path, pin.Patch.Path, pin.Worker.Path:
		default:
			t.Fatal("unrelated change", c.Path)
		}
		if c.Path == runtimeFile {
			var before, after map[string]any
			if json.Unmarshal(c.Before, &before) != nil || json.Unmarshal(c.After, &after) != nil {
				t.Fatal("invalid runtime patch")
			}
			delete(before, "semgrep_tool")
			delete(after, "semgrep_tool")
			if !reflect.DeepEqual(before, after) {
				t.Fatal("other runtime owner changed")
			}
		}
	}
	fault := errors.New("synthetic interruption")
	if err := applyPlan(p, state, io.Discard, func(int) error { return fault }); !errors.Is(err, fault) {
		t.Fatal("interruption not preserved", err)
	}
	if err := applyPlan(p, state, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	if err := checkCandidate(p, filepath.Join(state, "candidate"), true); err != nil {
		t.Fatal(err)
	}
	if err := compareSnapshot(p); err != nil {
		t.Fatal("original source changed", err)
	}
	source := filepath.Join(state, "semgrep-input", pin.Sources.Path)
	if err := os.Chmod(source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("replaced"), 0600); err != nil {
		t.Fatal(err)
	}
	var report bytes.Buffer
	if err := verifyPlan(p, state, "", &report); err == nil {
		t.Fatal("verify accepted changed frozen source")
	}
	if report.Len() != 0 {
		t.Fatal("failed verify emitted completion")
	}
	if err := compareSnapshot(p); err != nil {
		t.Fatal("source changed after failure", err)
	}
}

func TestSemgrepExplicitSameEngineRefreshPreservesUpstreamIdentity(t *testing.T) {
	for _, scenario := range []string{"refresh", "upstream move", "unchanged"} {
		t.Run(scenario, func(t *testing.T) {
			root, state := fixture(t)
			current, err := readSemgrepMaterializationOwner(root)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "upstream move" {
				current.SourceTagObject = "0000000000000000000000000000000000000000"
			}
			repository, input, _ := semgrepSupplyFixture(t, current)
			if scenario == "unchanged" {
				proposed, err := readSemgrepMaterializationOwner(root)
				if err != nil {
					t.Fatal(err)
				}
				writeSemgrepSupplyOwner(t, repository, proposed)
			}
			owner, err := os.ReadFile(filepath.Join(repository, runtimeFile))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(input, "owner.json"), owner, 0600); err != nil {
				t.Fatal(err)
			}
			var report bytes.Buffer
			err = run([]string{"plan", "--root", root, "--state", state, "--component", "semgrep", "--version", current.Version, "--semgrep-proof", input}, &report)
			if scenario != "refresh" {
				if err == nil {
					t.Fatal("ambiguous/redundant same-engine refresh accepted")
				}
				if _, err := os.Lstat(filepath.Join(state, "semgrep-input")); !os.IsNotExist(err) {
					t.Fatal("rejected proof copied large inputs", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var p plan
			if err := loadJSON(filepath.Join(state, "plan.json"), &p); err != nil {
				t.Fatal(err)
			}
			if p.Current != p.Version || len(p.RequiredQualification) == 0 || len(p.Changes) == 0 {
				t.Fatal("same-engine refresh lost qualification boundary")
			}
			for _, c := range p.Changes {
				if c.Path == supportFile {
					t.Fatal("same engine rewrote version support review")
				}
			}
			originalTag := p.Metadata.Semgrep.Owner.SourceTagObject
			p.Metadata.Semgrep.Owner.SourceTagObject = strings.Repeat("0", 40)
			if _, err := renderChanges(p, state); err == nil || !strings.Contains(err.Error(), "upstream") {
				t.Fatal("frozen plan bypassed upstream identity guard", err)
			}
			p.Metadata.Semgrep.Owner.SourceTagObject = originalTag
			if err := applyPlan(p, state, io.Discard, nil); err != nil {
				t.Fatal(err)
			}
			if err := compareSnapshot(p); err != nil {
				t.Fatal(err)
			}
		})
	}
}
