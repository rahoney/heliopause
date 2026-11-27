package pypi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUVProjectRecordsUseActualArchivesAndCompleteEdges(t *testing.T) {
	for _, dependency := range []string{"", "child>=1.0"} {
		t.Run(dependency, func(t *testing.T) {
			base, inputs, pages := uvTestSelection(t, dependency)
			lock, err := ParseUVProjectLock(uvProjectBody(base.Packages()), "3.14.7")
			if err != nil {
				t.Fatal(err)
			}
			target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
			snapshot, err := lock.BindProject(uvProjectContext(t, "/tmp/uv-records"), uvProjectControls(t, lock), inputs, pages, target, DefaultWheelLimits())
			if err != nil {
				t.Fatal(err)
			}
			nodes, edges, err := lock.ProjectRecords(snapshot.Dependencies(), inputs, target, DefaultWheelLimits())
			if err != nil || len(nodes) != 2 || len(edges) != map[bool]int{true: 0, false: 1}[dependency == ""] {
				t.Fatal("actual complete records differ", err)
			}
			for _, node := range nodes {
				if node.Role() != domain.DependencyTransitive || !strings.HasSuffix(node.RecordPath(), ".whl") || node.HostInstallAction() {
					t.Fatal("invented primary or missing actual filename")
				}
			}
			broken := append([]UVWheelInput(nil), inputs...)
			broken[0].Reader = bytes.NewReader(bytes.Repeat([]byte("x"), int(broken[0].Size)))
			if _, _, err := lock.ProjectRecords(snapshot.Dependencies(), broken, target, DefaultWheelLimits()); err == nil {
				t.Fatal("changed approved archive generated records")
			}
			artifacts := snapshot.Dependencies()
			a := artifacts[0]
			artifacts[0], _ = domain.NewResolvedArtifact(a.Identity(), a.AcquisitionLocator(), "sha256:"+strings.Repeat("c", 64))
			if _, _, err := lock.ProjectRecords(artifacts, inputs, target, DefaultWheelLimits()); err == nil {
				t.Fatal("changed locked integrity generated records")
			}
		})
	}
	lock, err := ParseUVProjectLock(uvProjectBody(nil), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	nodes, edges, err := lock.ProjectRecords(nil, nil, WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	if err != nil || len(nodes) != 0 || len(edges) != 0 {
		t.Fatal("explicit empty invented a component", err)
	}
}
