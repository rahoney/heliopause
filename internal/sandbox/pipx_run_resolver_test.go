package sandbox

import (
	"context"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"strings"
	"testing"
)

func TestPipxRunPreparationSafety(t *testing.T) {
	requirement, err := artifactpypi.ParseUpgradeRequirement("probe@1.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range []string{"", "pipx.run", "console_scripts", "gui_scripts", "other.group", "raw-script", "selected-pipx.run"} {
		t.Run(group, func(t *testing.T) {
			entries := "[console_scripts]\nprobe=probe:main\n"
			extra := map[string][]byte{}
			if group == "selected-pipx.run" {
				entries += "[pipx.run]\nprobe=probe:main\n"
			} else if group == "raw-script" {
				extra["probe-1.0.data/scripts/"+pipxRunPreparationCommand] = []byte("#!/usr/local/bin/python\nraise RuntimeError('must not execute')\n")
			} else if group != "" {
				if group != "console_scripts" {
					entries += "[" + group + "]\n"
				}
				entries += pipxRunPreparationCommand + "=probe:main\n"
			}
			extra["probe-1.0.dist-info/entry_points.txt"] = []byte(entries)
			frozen, _, _, _, _ := pipxFrozenInputFixture(t, extra)
			args, err := pipxRunPreparationArguments(frozen, requirement, "probe")
			allowed := group == "" || group == "other.group"
			if allowed != (err == nil) || !allowed && len(args) != 0 {
				t.Fatal("inert query role guard differs", group, err)
			}
			if allowed && (len(args) < 7 || args[5] != pipxNativeProgram || args[6] != "run" || args[len(args)-1] != pipxRunPreparationCommand || strings.Contains(strings.Join(args[6:], " "), "/dev/null")) {
				t.Fatal("query did not retain the fixed native frontend/terminal")
			}
		})
	}
}

func TestPipxRunBareRequestReplaysOnlyItsOwnedResolvedVersion(t *testing.T) {
	frozen, _, _, _, _ := pipxFrozenInputFixture(t, map[string][]byte{"probe-1.0.dist-info/entry_points.txt": []byte("[console_scripts]\nprobe=probe:main\n")})
	requirement, err := artifactpypi.ParseUpgradeRequirement("probe")
	if err != nil {
		t.Fatal(err)
	}
	args, err := pipxRunPreparationArguments(frozen, requirement, "probe")
	if err != nil || len(args) != 17 || args[14] != "probe==1.0" {
		t.Fatal("bare request escaped the same frozen version", args, err)
	}
}

func TestPipxRunSelectionBindsItsNativeIdentityRequestAndWholeDATA(t *testing.T) {
	frozen, facts, _ := pythonRunBindingFixture(t, true)
	request, _ := artifactpypi.ParseUpgradeRequirement("probe@1.0")
	selected, err := pipxRunSelectionFromCensus(context.Background(), request, frozen, "probe", facts)
	if err != nil || !selected.Entry().Valid() || selected.Resolution().RuntimeIdentity() != "pipx-run-isolated:"+PipxResolverRuntimeIdentity() || len(selected.Resolution().Graph().Nodes()) != 1 || len(selected.Resolution().ProjectControls()) != 0 {
		t.Fatal("native pipx selection differs or invents installed controls", err)
	}
	digest := selected.Resolution().LockfileDigest()
	reordered := cloneUVRunFacts(facts)
	for i, j := 0, len(reordered)-1; i < j; i, j = i+1, j-1 {
		reordered[i], reordered[j] = reordered[j], reordered[i]
	}
	same, err := pipxRunSelectionFromCensus(context.Background(), request, frozen, "probe", reordered)
	if err != nil || same.Resolution().LockfileDigest() != digest {
		t.Fatal("enumeration order changed native identity", err)
	}
	bare, _ := artifactpypi.ParseUpgradeRequirement("probe")
	different, err := pipxRunSelectionFromCensus(context.Background(), bare, frozen, "probe", facts)
	if err != nil || different.Resolution().LockfileDigest() == digest {
		t.Fatal("request was omitted from fingerprint", err)
	}
	changed := cloneUVRunFacts(facts)
	for i := range changed {
		if strings.HasSuffix(changed[i].Name, "/RECORD") {
			changed[i].SHA256 = strings.Repeat("a", 64)
		}
	}
	data, err := pipxRunSelectionFromCensus(context.Background(), request, frozen, "probe", changed)
	if err != nil || data.Resolution().LockfileDigest() == digest {
		t.Fatal("unapproved native DATA omitted from fingerprint", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	refused, err := pipxRunSelectionFromCensus(ctx, request, frozen, "probe", facts)
	if err == nil || refused.Entry().Valid() || len(refused.Resolution().Graph().Nodes()) != 0 {
		t.Fatal("canceled selection returned DATA")
	}
	refused, err = pipxRunSelectionFromCensus(context.Background(), request, frozen, "probe", nil)
	if err == nil || refused.Entry().Valid() || len(refused.Resolution().Graph().Nodes()) != 0 {
		t.Fatal("missing native DATA returned selection")
	}
}
