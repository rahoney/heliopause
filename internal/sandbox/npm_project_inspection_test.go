package sandbox

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func TestNPMProjectLifecycleBindsExactCommandAndWorkspace(t *testing.T) {
	container := strings.Repeat("a", 64)
	for _, tail := range []string{"pkg", "@scope/pkg", "pkg/node_modules/child", "@scope/pkg/node_modules/@other/child"} {
		target := nodeExecutionProject + "/node_modules/" + tail
		args, err := npmProjectTargetLifecycleExecArguments(container, target)
		expected := []string{"/bin/sh", "-ceu", npmProjectTargetLifecycleCommand, "helox-npm-project-target", target}
		if err != nil || len(args) < len(expected) || !slices.Equal(args[len(args)-len(expected):], expected) || strings.Contains(npmProjectTargetLifecycleCommand, target) {
			t.Fatal("npm target is not one independently admitted positional instance", err)
		}
		if validBunTargetLifecyclePath(target) || validPNPMTargetLifecyclePath(target) {
			t.Fatal("npm target acquired another client's workspace")
		}
	}
	for _, target := range []string{nodeExecutionProject + "/node_modules/pkg/child", nodeExecutionProject + "/node_modules/../pkg", bunProjectDirectory + "/node_modules/pkg", pnpmProjectDirectory + "/node_modules/.pnpm/pkg@1/node_modules/pkg", "/tmp/package", nodeExecutionProject + "/node_modules/$(id)"} {
		if args, err := npmProjectTargetLifecycleExecArguments(container, target); err == nil || len(args) != 0 {
			t.Fatalf("foreign or ambiguous npm target admitted: %q", target)
		}
	}
	if _, err := npmProjectTargetLifecycleExecArguments("container-name", nodeExecutionProject+"/node_modules/pkg"); err == nil {
		t.Fatal("ambiguous container admitted")
	}
	source, err := os.ReadFile("../../tools/gvisor-observer/observer.cc")
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := tokenizeCPPSource(source)
	if err != nil {
		t.Fatal(err)
	}
	var commands []string
	for k := 0; k+7 < len(tokens); k++ {
		if tokens[k].val == "constexpr" && tokens[k+1].val == "char" && tokens[k+2].val == "kNPMProjectTargetLifecycleCommand" && tokens[k+3].val == "[" && tokens[k+4].val == "]" && tokens[k+5].val == "=" && tokens[k+6].typ == cppTokRaw && tokens[k+6].delim == "HAA" && tokens[k+7].val == ";" {
			commands = append(commands, tokens[k+6].body)
		}
	}
	if len(commands) != 1 || commands[0] != npmProjectTargetLifecycleCommand {
		t.Fatal("npm command differs from exact observer contract")
	}
	spec, ok := nativeProjectIdentity(nativeRuntimeNPMProject)
	if !ok || spec.root != nodeExecutionRoot || spec.directory != nodeExecutionProject || spec.profile != "npm-lifecycle" {
		t.Fatal("npm runtime acquired a foreign workspace or profile")
	}
	original, ok := observerExpectedTopology("npm-lifecycle")
	topology, current := nodeMaterializationExpectedTopology()
	if !ok || !current || len(topology) != len(original)+1 || !slices.Equal(topology[:len(original)], original) || topology[len(original)].Mountpoint != nodeExecutionRoot {
		t.Fatal("npm did not preserve ordinary lifecycle topology")
	}
}
