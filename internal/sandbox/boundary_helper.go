package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
)

const (
	boundaryHelperPath        = "/haa-runtime/haa-boundary"
	boundaryHelperMount       = "/haa-runtime:rw,exec,nosuid,nodev,size=1m,uid=0,gid=0,mode=0755"
	boundaryLaunchMode        = "--launch"
	boundaryPythonHandoffMode = "--handoff-python"
	boundaryELFHandoffMode    = "--handoff-elf"
)

// boundaryHelper is installed by the trusted controller into a root-owned,
// container-private executable tmpfs. It has two deliberately asymmetric
// operations: launch establishes the Sentry boundary for a direct exec, while
// handoff only removes trust before artifact execution. The helper never
// interprets artifact input.
const boundaryHelper = "#!/bin/sh\nset -eu\ncase \"${1-}\" in\n  --launch) shift; exec \"$@\" ;;\n  --handoff-python|--handoff-elf) shift; exec \"$@\" ;;\n  -c) exec /bin/sh \"$@\" ;;\n  *) exit 125 ;;\nesac\n"

type boundaryInputRunner interface {
	RunInput(context.Context, io.Reader, string, ...string) error
}

func installBoundaryHelper(ctx context.Context, runner interface{}, containerID string) error {
	input, ok := runner.(boundaryInputRunner)
	if !ok || !containerIDPattern.MatchString(containerID) {
		return errors.New("sandbox boundary helper runner is unavailable")
	}
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	if err := tw.WriteHeader(&tar.Header{Name: "haa-boundary", Mode: 0555, Uid: 0, Gid: 0, Size: int64(len(boundaryHelper))}); err != nil {
		return errors.New("sandbox boundary helper archive failed")
	}
	if _, err := tw.Write([]byte(boundaryHelper)); err != nil || tw.Close() != nil {
		return errors.New("sandbox boundary helper archive failed")
	}
	return input.RunInput(ctx, &archive, "docker", "cp", "-", containerID+":/haa-runtime")
}
