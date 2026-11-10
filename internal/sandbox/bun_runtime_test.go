package sandbox

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestBunQueryRuntimeRetainsOfflineNodeBoundary(t *testing.T) {
	f := &uvVolumeFake{}
	volume, err := createNativeRuntimeVolume(context.Background(), f, nativeRuntimeBun)
	if err != nil {
		t.Fatal(err)
	}
	mount, err := volume.mountArgument()
	if err != nil {
		t.Fatal(err)
	}
	args, err := bunQueryCreateArguments(volume)
	if err != nil || !slices.Equal(args, nodeResolverCreateArguments("none", nil, []string{"--mount", mount})) {
		t.Fatal("Bun changed the existing offline Node boundary", err)
	}
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimeUV, nativeRuntimePNPM, nativeRuntimeToolKind(99)} {
		other := volume
		other.kind = kind
		if _, err := bunQueryCreateArguments(other); err == nil {
			t.Fatal("another tool volume became Bun authority")
		}
	}
	topology, ok := observerExpectedTopology(bunResolverProfile)
	if !ok || len(topology) != 3 || !topology[0].ReadOnly || !topology[1].NoExec || topology[2].FSType != "9p" || topology[2].Class != "helper" {
		t.Fatal("Bun topology is not its private fixed query runtime")
	}
	for _, profile := range []string{"bun", "bun-resolver-extra", "bunx"} {
		if validObserverProfile(profile) {
			t.Fatal("unsupported Bun profile admitted", profile)
		}
	}
	if isPythonObserverProfile(bunResolverProfile) {
		t.Fatal("Bun acquired Python authority")
	}
	if _, err := newObservationTraceLedger(bunResolverProfile); err == nil {
		t.Fatal("Bun acquired Python ledger authority")
	}
	if !strings.Contains(f.volume.Options["o"], "size=134217728,") {
		t.Fatal("Bun runtime cannot hold its pinned bytes")
	}
}
