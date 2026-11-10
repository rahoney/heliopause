//go:build linux && realcorpus

package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestBunActualNativeLockFixtures(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_NATIVE_LOCK_FIXTURE") != "1" {
		t.Fatal("declare ordinary-user fixed Bun native lock qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	scope, err := newBunNativeProjectScope(executor, systemEndpointResolver{}, supervisor.Observer(), func(ctx context.Context) (Capability, error) { return probe(ctx, "linux", executor) }, newIntegrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, manifest string
		wantLock       bool
	}{
		{"empty", `{"name":"haa-bun-empty","version":"0.0.0","private":true,"scripts":{"install":"exit 99"},"dependencies":{}}`, false},
		{"single", `{"name":"haa-bun-single","version":"0.0.0","private":true,"scripts":{"install":"exit 99"},"dependencies":{"is-number":"6.0.0"}}`, true},
		{"transitive", `{"name":"haa-bun-transitive","version":"0.0.0","private":true,"scripts":{"install":"exit 99"},"dependencies":{"is-odd":"3.0.1"}}`, true},
		{"peer", `{"name":"haa-bun-peer","version":"0.0.0","private":true,"scripts":{"install":"exit 99"},"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0"}}`, true},
	}
	for _, test := range cases {
		if !t.Run(test.name, func(t *testing.T) {
			input, err := domain.NewProjectControlFile("package.json", []byte(test.manifest), true)
			if err != nil {
				t.Fatal(err)
			}
			var selected []domain.ProjectControlFile
			err = scope.run(ctx, false, func(ctx context.Context, container, _ string, _ *closureVolume) error {
				if err := copyBunProjectControls(ctx, scope.runner, container, []domain.ProjectControlFile{input}); err != nil {
					return err
				}
				argv := []string{bunToolGuestPath, "--cwd=" + bunProjectDirectory, "install", "--lockfile-only", "--ignore-scripts", "--no-env-file", "--registry=https://registry.npmjs.org/", "--cache-dir=" + bunProjectRoot + "/cache", "--backend=copyfile", "--linker=hoisted", "--save-text-lockfile", "--no-progress", "--no-summary"}
				if _, err := scope.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, argv...)...); err != nil {
					return err
				}
				controls, err := readBunProjectControls(ctx, scope.runner.(streamingDockerOutput), container)
				if err != nil {
					return err
				}
				selected = controls
				return nil
			})
			if err != nil {
				t.Fatal("Bun native lock/complete observation/cleanup failed", err)
			}
			if len(selected) != 2 || !bytes.Equal(selected[0].Body(), input.Body()) || selected[1].Present() != test.wantLock {
				t.Fatal("native manifest/explicit lock presence differs")
			}
			body := selected[1].Body()
			if len(body) > 16<<10 {
				t.Fatal("unexpected qualification fixture size")
			}
			t.Logf("bun_native_lock_fixture=%s manifest_sha256=%x lock_present=%t lock_bytes=%d lock=%q", test.name, sha256.Sum256(input.Body()), selected[1].Present(), len(body), body)
		}) {
			return
		}
	}
}
