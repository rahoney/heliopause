//go:build realcorpus

package githubrelease

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestPipxRuntimeInputActualAcquisitionBinding(t *testing.T) {
	filename := os.Getenv("HELOX_PIPX_RUNTIME_ZIP")
	if filename == "" {
		t.Skip("explicit selected pipx ZIP required")
	}
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"valid", "growth-during-read", "cancel-during-read"} {
		t.Run(mode, func(t *testing.T) {
			reader, artifact, held := pipxInputFixture(t, body)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var read int64
			changed := false
			ctx = ports.WithProgress(ctx, func(event ports.Progress) {
				read = event.Current
				if changed {
					return
				}
				changed = true
				switch mode {
				case "growth-during-read":
					file, err := os.OpenFile(held, os.O_WRONLY|os.O_APPEND, 0)
					if err != nil {
						t.Fatal(err)
					}
					_, writeErr := file.Write(make([]byte, 1024))
					closeErr := file.Close()
					if writeErr != nil || closeErr != nil {
						t.Fatal(writeErr, closeErr)
					}
				case "cancel-during-read":
					cancel()
				}
			})
			input, err := reader.readPipxRuntimeInput(ctx, artifact)
			if read > runtimeidentity.PipxToolSize+1 {
				t.Fatal("read exceeded selected bound", read)
			}
			if mode == "growth-during-read" {
				if err == nil || input.plan.Valid() || read != runtimeidentity.PipxToolSize+1 {
					t.Fatal("growing input accepted or bound not exercised", read, err)
				}
				return
			}
			if mode == "cancel-during-read" {
				if !errors.Is(err, context.Canceled) || input.plan.Valid() {
					t.Fatal("in-flight cancellation lost", err)
				}
				return
			}
			if err != nil || !input.plan.Valid() || input.artifact.Identity() != artifact.Identity() || input.artifact.Digest() != artifact.Digest() || input.artifact.ContentHandle() != artifact.ContentHandle() || len(input.plan.Units()) != 12 {
				t.Fatal("original HAA acquisition binding lost", err)
			}
			if err := os.WriteFile(held, []byte("changed after acquisition"), 0600); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(input.plan.ZIP().Bytes(), body) {
				t.Fatal("intake mutation changed frozen plan")
			}
			t.Logf("original synthetic HAA version=%s selected bytes=%d units=%d; no inspection completion", input.artifact.Identity().Version(), read, len(input.plan.Units()))
		})
	}
}
