//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestClientHandoffWaitsForPreparedAndDoesNotActivateAfterFailure(t *testing.T) {
	for _, kind := range []string{"success", "cleanup-failed", "cancel-before-activate", "bad-prepared", "lost-commit", "changed-payload"} {
		t.Run(kind, func(t *testing.T) {
			release, files := wireFixture(t)
			client, server := net.Pipe()
			defer client.Close()
			defer server.Close()
			_ = client.SetDeadline(time.Now().Add(5 * time.Second))
			_ = server.SetDeadline(time.Now().Add(5 * time.Second))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			type result struct {
				activated bool
				err       error
			}
			done := make(chan result, 1)
			cleanup := errors.New("controlled cleanup error")
			calls := 0
			go func() {
				activated, err := exchangeUpdate(ctx, client, release, func(name string) (io.ReadCloser, error) {
					body := files[name]
					if kind == "changed-payload" {
						body = append(append([]byte(nil), body...), '!')
					}
					return io.NopCloser(bytes.NewReader(body)), nil
				}, func() error {
					calls++
					if kind == "cleanup-failed" {
						return cleanup
					}
					if kind == "cancel-before-activate" {
						cancel()
					}
					return nil
				})
				_ = client.Close()
				done <- result{activated, err}
			}()
			decoded, err := readWirePreamble(context.Background(), server)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.manifest.SHA256() != release.manifest.SHA256() {
				t.Fatal("manifest changed")
			}
			if kind == "changed-payload" {
				_, _ = io.Copy(io.Discard, server)
			} else {
				for _, file := range decoded.files {
					body := make([]byte, file.Size)
					if _, err := io.ReadFull(server, body); err != nil || !bytes.Equal(body, files[file.Name]) {
						t.Fatal("different selected bytes", err)
					}
				}
				if err := readControl(context.Background(), server, wireStagedMarker); err != nil {
					t.Fatal(err)
				}
				prepared := "PREPARED1\n"
				if kind == "bad-prepared" {
					prepared = "REJECTED1\n"
				}
				if err := writeWireBytes(context.Background(), server, []byte(prepared)); err != nil {
					t.Fatal(err)
				}
				if kind == "success" || kind == "lost-commit" {
					if err := readControl(context.Background(), server, "ACTIVATE1\n"); err != nil {
						t.Fatal(err)
					}
					if kind == "success" {
						if err := writeWireBytes(context.Background(), server, []byte("COMMITTED1\n")); err != nil {
							t.Fatal(err)
						}
					} else {
						_ = server.Close()
					}
				} else {
					body, err := io.ReadAll(server)
					if err != nil || len(body) != 0 {
						t.Fatal("activation after rejected handoff", string(body), err)
					}
				}
			}
			r := <-done
			wantActivation := kind == "success" || kind == "lost-commit"
			if r.activated != wantActivation || (r.err == nil) != (kind == "success") {
				t.Fatal(r)
			}
			wantCalls := 1
			if kind == "bad-prepared" || kind == "changed-payload" {
				wantCalls = 0
			}
			if calls != wantCalls {
				t.Fatal("premature or missing runtime cleanup", calls)
			}
			if kind == "cleanup-failed" && !errors.Is(r.err, cleanup) {
				t.Fatal("lost primary cleanup failure", r.err)
			}
			if kind == "cancel-before-activate" && !errors.Is(r.err, context.Canceled) {
				t.Fatal("lost cancellation", r.err)
			}
		})
	}
}

func TestClientWaitRetainsActualHelperFailureAndReapsOnce(t *testing.T) {
	want := errors.New("controlled child failure")
	for _, result := range []error{nil, want} {
		done := make(chan error, 1)
		process := &installerProcess{done: done}
		if process.exited() {
			t.Fatal("unobserved child reported stopped")
		}
		done <- result
		for range 2 {
			err := process.wait()
			if (err == nil) != (result == nil) || result != nil && (!errors.Is(err, want) || !errors.Is(err, ErrInstaller)) {
				t.Fatal(err)
			}
		}
		if !process.exited() {
			t.Fatal("lost actual wait")
		}
	}
}

func TestClientRejectsUnmanagedAndCanceledBeforePrivilegedLaunch(t *testing.T) {
	release, _ := wireFixture(t)
	open := func(string) (io.ReadCloser, error) { t.Fatal("unmanaged content opened"); return nil, nil }
	drain := func() error { t.Fatal("unmanaged runtime handed off"); return nil }
	if err := Activate(context.Background(), nil, release.body, release.proofs, open, drain); !errors.Is(err, ErrUnmanaged) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Activate(ctx, nil, release.body, release.proofs, open, drain); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
