//go:build realcorpus

package pypi

import (
	"os"
	"path/filepath"
	"testing"
)

// This explicit corpus checks authenticated native uv selection only. It is
// neither a default-test input nor a substitute for production runtime tests.
func TestUVAuthenticatedFrozenSelection(t *testing.T) {
	corpus := os.Getenv("HELOX_UV_WHEEL_ROOT")
	if !filepath.IsAbs(corpus) {
		t.Fatal("HELOX_UV_WHEEL_ROOT must explicitly select an absolute frozen corpus")
	}
	body, err := os.ReadFile("testdata/uv-0.13.4-requests.pylock.toml")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := os.ReadFile("testdata/uv-0.13.4-requests-installed.json")
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := ParseReference("requests@2.32.5")
	lock, err := ParseUVLock(reference, body, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	wheels, err := lock.SelectInstalledWheels(snapshot, target)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(corpus)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	var inputs []UVWheelInput
	var pages []SimpleProject
	for _, wheel := range wheels {
		file, err := root.Open(wheel.Filename)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		stat, err := file.Stat()
		if err != nil || !stat.Mode().IsRegular() {
			t.Fatal("wheel input must be regular", err)
		}
		project, version, _, _, _, err := ParseWheelFilename(wheel.Filename)
		if err != nil {
			t.Fatal(err)
		}
		metadata, err := root.ReadFile(project + ".release.json")
		if err != nil {
			t.Fatal(err)
		}
		page, err := ParseUVReleaseMetadata(project, version, metadata)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
		inputs = append(inputs, UVWheelInput{URL: wheel.URL, Reader: file, Size: stat.Size()})
	}
	full, err := root.ReadFile("charset-normalizer.full-simple.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseSimpleProject("charset-normalizer", full); err == nil {
		t.Fatal("existing full Simple bound unexpectedly weakened")
	}
	graph, err := lock.BindWheels(inputs, pages, target, DefaultWheelLimits())
	if err != nil || len(graph.Nodes()) != 5 || len(graph.Edges()) != 4 {
		t.Fatal("native selected authenticated wheel graph mismatch", err)
	}
	t.Log("five native uv-selected archives, independent exact-release registry records, complete SHA/RECORD/metadata and four dependency edges validated")
}
