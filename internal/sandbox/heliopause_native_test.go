package sandbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestHeliopausePNPMProbeRejectsUnboundMembersBeforeExecution(t *testing.T) {
	for _, change := range []string{"none", "source", "name", "digest", "size", "other-native"} {
		t.Run(change, func(t *testing.T) {
			source, name, variant := "github-release", "rahoney-heliopause", runtimeidentity.PNPMNativeAssetName
			hash, size := runtimeidentity.PNPMToolSHA256, uint64(runtimeidentity.PNPMToolSize)
			switch change {
			case "source":
				source = "npm"
			case "name":
				name = "foreign-project"
			case "digest":
				hash = strings.Repeat("a", 64)
			case "size":
				size++
			case "other-native":
				variant = "helox-linux-amd64"
			}
			id, err := domain.NewSourceID(source)
			if err != nil {
				t.Fatal(err)
			}
			identity, err := domain.NewResolvedArtifactIdentity(id, name, "v1.0.0-probe", variant)
			if err != nil {
				t.Fatal(err)
			}
			digest, err := domain.NewSHA256Digest(hash)
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:github-release", size)
			if err != nil {
				t.Fatal(err)
			}
			request, err := domain.NewSandboxRequest(artifact)
			if err != nil {
				t.Fatal(err)
			}
			probes := 0
			runner := &recordingRunner{}
			elf, err := NewGitHubELFBackend(runner, t.TempDir(), &recordingObserver{}, func(context.Context) (Capability, error) {
				probes++
				return Capability{LimitationCode: "M3_RUNTIME_UNAVAILABLE"}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			backend := &HeliopauseNativeBackend{elf: elf}
			result, err := backend.Execute(context.Background(), request)
			selected := change == "none" || change == "other-native"
			if selected {
				code, _ := result.LimitationCode()
				if err != nil || probes != 1 || result.Status() != domain.SandboxIncomplete || code != "M3_RUNTIME_UNAVAILABLE" {
					t.Fatalf("capability failure lost: probes=%d result=%v error=%v", probes, result, err)
				}
			} else if err == nil || probes != 0 {
				t.Fatal("unbound PNPM member reached the sandbox")
			}
			if len(runner.calls) != 0 {
				t.Fatal("rejected or unavailable probe executed a command")
			}
		})
	}
}

func TestHeliopausePNPMProbeDiscardsOutputAndPreservesFailure(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		cancel bool
	}{
		{name: "zero"},
		{name: "help-exit-two", err: errors.New("exit status 2")},
		{name: "observer-ambiguous", err: errObserverAuthorityAmbiguous},
		{name: "cancelled", cancel: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &recordingRunner{errors: []error{test.err}}
			backend := &HeliopauseNativeBackend{elf: &GitHubELFBackend{runner: runner}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if test.cancel {
				cancel()
			}
			subject, err := backend.probePNPM(ctx, "0123456789abcdef")
			want := test.err
			if test.cancel {
				want = context.Canceled
			}
			if subject != "" || !errors.Is(err, want) {
				t.Fatalf("probe failure changed: %s %v", subject, err)
			}
			if test.cancel {
				if len(runner.calls) != 0 {
					t.Fatal("cancelled probe executed a command")
				}
				return
			}
			if len(runner.calls) != 1 || !sameStrings(runner.calls[0].arguments, boundaryExecArguments("0123456789abcdef", boundaryELFHandoffMode, "/work/artifact", "--version")) {
				t.Fatal("probe is not the fixed isolated ARTIFACT version command")
			}
		})
	}
}
