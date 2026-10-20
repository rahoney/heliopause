package hosttool

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// This child exercises the production launcher body, readiness pipe and real
// signals. It does not replace the public executable identity checks.
func TestMain(m *testing.M) {
	if len(os.Args) == 5 && strings.HasPrefix(filepath.Base(os.Args[1]), ".observer-lifecycle-helper-connection") && os.Args[4] == "--ready-fd=3" {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM, syscall.SIGUSR1)
		if err := os.WriteFile(os.Args[2]+".started", []byte("started"), 0o600); err != nil {
			os.Exit(91)
		}
		if !strings.HasSuffix(os.Args[1], "-no-ready") {
			ready := os.NewFile(3, "readiness")
			if _, err := ready.Write([]byte("R")); err != nil {
				os.Exit(92)
			}
			_ = ready.Close()
		}
		if <-stop == syscall.SIGUSR1 {
			os.Exit(23)
		}
		if err := os.WriteFile(os.Args[2]+".stopped", []byte("graceful"), 0o600); err != nil {
			os.Exit(93)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func observerChildPaths(t *testing.T) (string, string, string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	return executable, filepath.Join(root, ".observer-lifecycle-helper-connection"), filepath.Join(root, "output")
}

func stopObserverChild(t *testing.T, process ObserverProcess) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := process.Stop(ctx); err != nil {
		t.Errorf("owned helper Stop: %v", err)
	}
}

func TestObserverReadyChildSurvivesOperationCancellationUntilOwnedStop(t *testing.T) {
	executable, remote, output := observerChildPaths(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	process, err := startObserverProcess(ctx, executable, remote, output)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopObserverChild(t, process) })
	cancel()
	select {
	case <-process.Done():
		t.Fatalf("operation cancellation killed the owned receiver: %v", process.ExitError())
	case <-time.After(250 * time.Millisecond):
	}
	stopObserverChild(t, process)
	if body, err := os.ReadFile(output + ".stopped"); err != nil || string(body) != "graceful" {
		t.Fatalf("receiver did not perform graceful cleanup: %q, %v", body, err)
	}
}

func TestObserverReadinessCancellationStopsOwnedChildGracefully(t *testing.T) {
	executable, remote, output := observerChildPaths(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := startObserverProcess(ctx, executable, remote+"-no-ready", output)
		done <- err
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(output + ".started"); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("child did not start")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled readiness was accepted")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancelled readiness did not reap child")
	}
	if body, err := os.ReadFile(output + ".stopped"); err != nil || string(body) != "graceful" {
		t.Fatalf("failed readiness did not gracefully clean child: %q, %v", body, err)
	}
}

func TestObserverUnexpectedChildDeathRemainsFailure(t *testing.T) {
	executable, remote, output := observerChildPaths(t)
	process, err := startObserverProcess(context.Background(), executable, remote, output)
	if err != nil {
		t.Fatal(err)
	}
	actual := process.(*observerProcess)
	if err := actual.command.Process.Signal(syscall.SIGUSR1); err != nil {
		stopObserverChild(t, process)
		t.Fatal(err)
	}
	select {
	case <-process.Done():
	case <-time.After(3 * time.Second):
		stopObserverChild(t, process)
		t.Fatal("child did not exit")
	}
	if process.ExitError() == nil || process.Stop(context.Background()) == nil {
		t.Fatal("unexpected child death was treated as clean owner shutdown")
	}
}

func TestObserverCancelledStartDoesNotLaunchChild(t *testing.T) {
	executable, remote, output := observerChildPaths(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if process, err := startObserverProcess(ctx, executable, remote, output); err == nil || process != nil {
		t.Fatalf("cancelled start: process=%v err=%v", process, err)
	}
	if _, err := os.Stat(output + ".started"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled context launched child: %v", err)
	}
}
