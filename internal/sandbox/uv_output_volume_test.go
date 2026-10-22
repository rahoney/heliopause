package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type uvOutputCloseRunner struct {
	volume             []byte
	listed             []byte
	listErr, removeErr error
	removed            bool
}

func (r *uvOutputCloseRunner) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	if len(args) > 1 && args[0] == "volume" && args[1] == "inspect" {
		return r.volume, nil
	}
	if len(args) > 0 && args[0] == "ps" {
		return r.listed, r.listErr
	}
	if len(args) == 3 && args[0] == "volume" && args[1] == "rm" {
		r.removed = true
		return nil, r.removeErr
	}
	return nil, errors.New("unexpected command")
}

func TestUVOutputVolumeCleanupRejectsUnknownOrAttachedIdentity(t *testing.T) {
	volume := closureVolume{name: "haa-closure-output", transaction: "tx", manifestID: strings.Repeat("f", 64), createdAt: "fixed", mountpoint: "/owned/output", capacity: 128 << 20}
	actual := dockerVolumeInspection{
		Name: volume.name, Driver: "local", Scope: "local", Mountpoint: volume.mountpoint, CreatedAt: volume.createdAt,
		Options: map[string]string{"type": "tmpfs", "device": "tmpfs", "o": volume.tmpfsOptions()},
		Labels:  map[string]string{closureVolumeLabel: volume.transaction, closureManifestLabel: volume.manifestID},
	}
	body, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []string{"complete", "query failure", "attached", "identity changed", "remove failure"} {
		t.Run(outcome, func(t *testing.T) {
			runner := &uvOutputCloseRunner{volume: body}
			cause := errors.New("cleanup command failed")
			switch outcome {
			case "query failure":
				runner.listErr = cause
			case "attached":
				runner.listed = []byte(strings.Repeat("a", 64) + "\n")
			case "identity changed":
				runner.volume = []byte(`{}`)
			case "remove failure":
				runner.removeErr = cause
			}
			candidate := volume
			err := closeUVOutputVolume(context.Background(), runner, &candidate)
			if outcome == "complete" {
				if err != nil || !runner.removed {
					t.Fatal("verified idle volume not removed", err)
				}
			} else if err == nil {
				t.Fatal("uncertain cleanup accepted")
			}
			if outcome != "complete" && outcome != "remove failure" && runner.removed {
				t.Fatal("unsafe volume removal attempted")
			}
			if outcome == "remove failure" && !errors.Is(err, cause) {
				t.Fatal("remove failure cause lost")
			}
		})
	}
}
