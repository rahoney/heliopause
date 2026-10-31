package sandbox

import (
	"os"
	"slices"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

func TestPNPMTargetLifecycleUsesOnePositionalInstanceAndMatchingObserverCommand(t *testing.T) {
	container := strings.Repeat("a", 64)
	for _, target := range []string{
		pnpmProjectDirectory + "/node_modules/.pnpm/is-odd@3.0.1/node_modules/is-odd",
		pnpmProjectDirectory + "/node_modules/.pnpm/use-sync-external-store@1.2.0_react@18.3.1/node_modules/use-sync-external-store",
		pnpmProjectDirectory + "/node_modules/.pnpm/@scope+package@1.2.3/node_modules/@scope/package",
	} {
		args, err := pnpmTargetLifecycleExecArguments(container, target)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"/bin/sh", "-ceu", pnpmTargetLifecycleCommand, "helox-pnpm-target", target}
		if !slices.Equal(args[len(args)-len(want):], want) || strings.Contains(pnpmTargetLifecycleCommand, target) {
			t.Fatal("target entered shell source or extra selectors appeared")
		}
		count := 0
		for _, value := range args {
			if value == target {
				count++
			}
		}
		if count != 1 {
			t.Fatal("target is not one positional argument")
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
		if tokens[k].typ == cppTokIdent && tokens[k].val == "constexpr" &&
			tokens[k+1].typ == cppTokIdent && tokens[k+1].val == "char" &&
			tokens[k+2].typ == cppTokIdent && tokens[k+2].val == "kPNPMTargetLifecycleCommand" &&
			tokens[k+3].val == "[" && tokens[k+4].val == "]" && tokens[k+5].val == "=" &&
			tokens[k+6].typ == cppTokRaw && tokens[k+6].delim == "HAA" && tokens[k+7].val == ";" {
			commands = append(commands, tokens[k+6].body)
		}
	}
	if len(commands) != 1 || commands[0] != pnpmTargetLifecycleCommand {
		t.Fatal("Go producer and exact native admission command differ")
	}
}

func TestPNPMTargetLifecycleRejectsOtherPathsAndAmbiguousSelectors(t *testing.T) {
	base := pnpmProjectDirectory + "/node_modules/.pnpm/pkg@1.0.0/node_modules/"
	for _, target := range []string{"", "/tmp/package", base + "../foreign", base + "pkg/extra", base + "@scope", base + "@/pkg", base + "pkg ", base + "pkg;id", base + "$(id)", base + "pkg\\extra", base + "pkg\x00", base + "pkg//extra", base + ".", base + "..", base + strings.Repeat("a", 512), strings.Replace(base+"pkg", "pkg@1.0.0", "folder", 1), strings.Replace(base+"pkg", ".pnpm/", ".pnpm/../", 1)} {
		if args, err := pnpmTargetLifecycleExecArguments(strings.Repeat("a", 64), target); err == nil || len(args) != 0 {
			t.Fatalf("invalid target admitted: %q", target)
		}
	}
	if _, err := pnpmTargetLifecycleExecArguments("container-name", base+"pkg"); err == nil {
		t.Fatal("ambiguous container admitted")
	}
}

func TestPNPMTargetLifecycleTopologyPreservesOrdinaryNPMDefaults(t *testing.T) {
	ordinary, ok := observerExpectedTopology("npm-lifecycle")
	if !ok || len(ordinary) != 3 {
		t.Fatal("ordinary npm topology changed")
	}
	closure, ok := pnpmTargetLifecycleExpectedTopology()
	if !ok || len(closure) != len(ordinary)+1 || !slices.Equal(closure[:len(ordinary)], ordinary) {
		t.Fatal("target topology changed existing npm mounts")
	}
	want := observerMountExpectation{pnpmProjectRoot, "workspace", "/tmp", "9p", false, false, false, false}
	if closure[len(ordinary)] != want {
		t.Fatal("target volume flags differ")
	}
	closure[0].ReadOnly = false
	again, _ := observerExpectedTopology("npm-lifecycle")
	if !again[0].ReadOnly {
		t.Fatal("caller mutated ordinary profile")
	}
}

func TestPNPMTargetLifecycleAcceptsOnlyReviewedLongCodecShape(t *testing.T) {
	name := strings.Repeat("a", 130)
	slot, err := artifactnpm.PNPMInstanceDirectory(name + "@1.0.0")
	expected := strings.Repeat("a", 87) + "_69b8023e95a408229c8fe59943dd2497"
	if err != nil || slot != expected || strings.Contains(slot, "@") {
		t.Fatal("independent codec vector differs", slot, err)
	}
	prefix := pnpmProjectDirectory + "/node_modules/.pnpm/"
	if _, err := pnpmTargetLifecycleExecArguments(strings.Repeat("a", 64), prefix+slot+"/node_modules/"+name); err != nil {
		t.Fatal("bound long name rejected", err)
	}
	for _, bad := range []string{slot[:119], slot + "f", slot[:87] + "-" + slot[88:], slot[:88] + strings.ToUpper(slot[88:]), slot[:119] + "g"} {
		if validPNPMTargetLifecyclePath(prefix + bad + "/node_modules/" + name) {
			t.Fatal("malformed long hash locator admitted", bad)
		}
	}
}
