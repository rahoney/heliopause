package domain

import (
	"strings"
	"testing"
)

func TestProjectUpdateBindsSelectedControlsAndCompleteGraph(t *testing.T) {
	snapshot, _ := projectApprovalFixture(t)
	body := []byte("selected opaque control\n")
	selected, err := NewProjectControlFile("go.mod", body, true)
	if err != nil {
		t.Fatal(err)
	}
	original, err := NewProjectControlFile("go.mod", []byte("original\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	control, _ := NewProjectControlDigest("go.mod", selected.Digest())
	snapshot.controls = []ProjectControlDigest{control}
	node, _ := NewDependencyNodeID("requested")
	primary, _ := NewLockedDependency(node, DependencyPrimary, snapshot.dependencies[0])
	graph, err := NewLockedDependencyGraph([]LockedDependency{primary}, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := NewDependencyResolution(graph, "fixture", snapshot.GraphDigest())
	if err != nil {
		t.Fatal(err)
	}
	update, err := NewProjectDependencyUpdate([]ProjectControlFile{original}, []ProjectControlFile{selected}, snapshot, resolution)
	if err != nil || !update.Valid() {
		t.Fatalf("update: %v", err)
	}
	body[0] = 'X'
	copyBody := update.SelectedControls()[0].Body()
	copyBody[0] = 'Y'
	copyFiles := update.SelectedControls()
	copyFiles[0] = ProjectControlFile{}
	if string(update.SelectedControls()[0].Body()) != "selected opaque control\n" {
		t.Fatal("selected control content was mutable")
	}
	for _, name := range []string{"selected-bytes", "duplicate", "missing", "graph", "primary-content", "absent-selected"} {
		t.Run(name, func(t *testing.T) {
			before, after := []ProjectControlFile{original}, []ProjectControlFile{selected}
			changed := resolution
			switch name {
			case "selected-bytes":
				after[0], _ = NewProjectControlFile("go.mod", []byte("other"), true)
			case "duplicate":
				before = append(before, before[0])
				after = append(after, after[0])
			case "missing":
				before = nil
			case "graph":
				changed.lockfileDigest, _ = NewSHA256Digest(strings.Repeat("b", 64))
			case "primary-content":
				changed.graph.nodes = append([]LockedDependency(nil), resolution.graph.nodes...)
				changed.graph.nodes[0].artifact.declaredIntegrity = "other content"
			case "absent-selected":
				after[0], _ = NewProjectControlFile("go.mod", nil, false)
			}
			if _, err := NewProjectDependencyUpdate(before, after, snapshot, changed); err == nil {
				t.Fatal("unbound update accepted")
			}
		})
	}
}

func TestProjectControlPresenceAndBounds(t *testing.T) {
	absent, err := NewProjectControlFile("go.sum", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := NewProjectControlFile("go.sum", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if absent.Present() || !empty.Present() || absent.Digest() != empty.Digest() {
		t.Fatal("empty content and file presence were conflated")
	}
	for _, name := range []string{"", ".", "..", "../go.mod", "a/b", "a\\b", "a\x00", "a\n", "a b"} {
		if _, err := NewProjectControlFile(name, nil, true); err == nil {
			t.Fatalf("unsafe name accepted: %q", name)
		}
	}
	if _, err := NewProjectControlFile("go.sum", []byte("hidden"), false); err == nil {
		t.Fatal("absent file carried content")
	}
	if _, err := NewProjectControlFile("go.mod", make([]byte, (4<<20)+1), true); err == nil {
		t.Fatal("unbounded control accepted")
	}
}
