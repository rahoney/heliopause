//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// Pure synthetic command-selection qualification, not public source approval.
func TestBunActualTargetLifecycleFixture(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_TARGET_LIFECYCLE_FIXTURE") != "1" {
		t.Fatal("declare ordinary-user synthetic Bun target qualification")
	}
	for _, target := range []string{"target", "@scope/pkg/node_modules/@other/child"} {
		if !t.Run(target, func(t *testing.T) {
			for _, failure := range []bool{false, true} {
				label := "complete"
				if failure {
					label = "script-failure"
				}
				if !t.Run(label, func(t *testing.T) {
					input := bunSyntheticTargetArchive(t, target, failure)
					events, execs := "preinstall\ninstall\npostinstall\n", uint64(3)
					if failure {
						events = "preinstall\n"
						execs = 2
					}
					withActualNPMTargetLifecycle(t, input, pnpmTargetLifecycleExpectation{target: bunProjectDirectory + "/node_modules/" + target, failure: failure, events: events, artifactExecs: execs}, nativeRuntimeBun)
				}) {
					return
				}
			}
		}) {
			return
		}
	}
}

func bunSyntheticTargetArchive(t *testing.T, target string, failure bool) []byte {
	t.Helper()
	event := func(value string) string {
		return "printf '" + value + "\\n' >> " + bunProjectRoot + "/lifecycle-events"
	}
	forbidden := map[string]string{"preinstall": event("non-target"), "install": event("non-target"), "postinstall": event("non-target"), "prepare": event("non-target")}
	scripts := map[string]string{"preinstall": event("preinstall"), "install": event("install"), "postinstall": event("postinstall"), "prepare": event("prepare")}
	if failure {
		scripts["install"] = "exit 37"
	}
	bodies := map[string][]byte{}
	emit := func(path string, value any) {
		b, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		bodies[path] = b
	}
	rootName, targetName := "target", "target"
	if strings.HasPrefix(target, "@scope/") {
		rootName = "@scope/pkg"
		targetName = "@other/child"
	}
	emit("project/package.json", map[string]any{"name": "haa-synthetic-bun-root", "private": true, "dependencies": map[string]string{rootName: "1.0.0", "helper": "1.0.0"}, "scripts": forbidden})
	emit("project/node_modules/helper/package.json", map[string]any{"name": "helper", "version": "1.0.0", "scripts": forbidden})
	if rootName != targetName {
		emit("project/node_modules/"+rootName+"/package.json", map[string]any{"name": rootName, "version": "1.0.0", "dependencies": map[string]string{targetName: "1.0.0"}, "scripts": forbidden})
	}
	emit("project/node_modules/"+target+"/package.json", map[string]any{"name": targetName, "version": "1.0.0", "dependencies": map[string]string{"helper": "1.0.0"}, "scripts": scripts})
	directories := map[string]bool{"project": true}
	for name := range bodies {
		parts := strings.Split(name, "/")
		for end := 1; end < len(parts); end++ {
			directories[strings.Join(parts[:end], "/")] = true
		}
	}
	var names []string
	for name := range directories {
		names = append(names, name)
	}
	slices.Sort(names)
	var files []pnpmPayloadTestFile
	for _, name := range names {
		files = append(files, pnpmPayloadTestFile{header: tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o755, Uid: 1000, Gid: 1000}})
	}
	names = nil
	for name := range bodies {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		body := bodies[name]
		files = append(files, pnpmPayloadTestFile{header: tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Uid: 1000, Gid: 1000, Size: int64(len(body))}, body: body})
	}
	return pnpmPayloadTestTar(t, files)
}
