package sandbox

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func TestBunTargetLifecycleSelectsOneHoistedInstance(t *testing.T) {
	for _, tail := range []string{"pkg", "@scope/pkg", "pkg/node_modules/child", "@scope/pkg/node_modules/@other/child"} {
		target := bunProjectDirectory + "/node_modules/" + tail
		args, err := bunTargetLifecycleExecArguments(strings.Repeat("a", 64), target)
		expected := []string{"/bin/sh", "-ceu", bunTargetLifecycleCommand, "helox-bun-target", target}
		if err != nil || !slices.Equal(args[len(args)-len(expected):], expected) || strings.Contains(bunTargetLifecycleCommand, target) {
			t.Fatal("target is not one positional selector", err)
		}
		count := 0
		for _, arg := range args {
			if arg == target {
				count++
			}
		}
		if count != 1 {
			t.Fatal("ambiguous target")
		}
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
		if tokens[k].typ == cppTokIdent && tokens[k].val == "constexpr" && tokens[k+1].typ == cppTokIdent && tokens[k+1].val == "char" && tokens[k+2].typ == cppTokIdent && tokens[k+2].val == "kBunTargetLifecycleCommand" && tokens[k+3].val == "[" && tokens[k+4].val == "]" && tokens[k+5].val == "=" && tokens[k+6].typ == cppTokRaw && tokens[k+6].delim == "HAA" && tokens[k+7].val == ";" {
			commands = append(commands, tokens[k+6].body)
		}
	}
	if len(commands) != 1 || commands[0] != bunTargetLifecycleCommand {
		t.Fatal("Go command and exact observer command differ")
	}
}

func TestBunTargetLifecycleRejectsOtherRootsAndAmbiguousPaths(t *testing.T) {
	prefix := bunProjectDirectory + "/node_modules/"
	for _, tail := range []string{"", "@scope", "@/pkg", "../pkg", "pkg/child", "pkg/node_modules", "pkg/node_modules/", "pkg//node_modules/child", "pkg;id", "$(id)", "pkg\\child", "pkg\x00", "pkg/node_modules/../child", strings.Repeat("a", 512)} {
		if args, err := bunTargetLifecycleExecArguments(strings.Repeat("a", 64), prefix+tail); err == nil || len(args) != 0 {
			t.Fatalf("invalid selector admitted: %q", tail)
		}
	}
	for _, target := range []string{"", pnpmProjectDirectory + "/node_modules/.pnpm/pkg@1/node_modules/pkg", "/tmp/package"} {
		if args, err := bunTargetLifecycleExecArguments(strings.Repeat("a", 64), target); err == nil || len(args) != 0 {
			t.Fatal("foreign target admitted")
		}
	}
	if _, err := bunTargetLifecycleExecArguments("container-name", prefix+"pkg"); err == nil {
		t.Fatal("ambiguous container")
	}
}

func TestBunTargetLifecycleTopologyPreservesExistingConsumers(t *testing.T) {
	ordinary, ok := observerExpectedTopology("npm-lifecycle")
	if !ok {
		t.Fatal("default topology unavailable")
	}
	pnpm, pnpmOK := pnpmTargetLifecycleExpectedTopology()
	bun, bunOK := bunTargetLifecycleExpectedTopology()
	if !pnpmOK || !bunOK || len(bun) != len(ordinary)+1 || !slices.Equal(bun[:len(ordinary)], ordinary) || bun[len(ordinary)].Mountpoint != bunProjectRoot || pnpm[len(ordinary)].Mountpoint != pnpmProjectRoot {
		t.Fatal("separate attested project topology differs")
	}
	after, _ := observerExpectedTopology("npm-lifecycle")
	if !slices.Equal(after, ordinary) {
		t.Fatal("ordinary caller mutated")
	}
}
