//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRegistrationRejectsAmbiguousAndUnscopedState(t *testing.T) {
	valid := registration{registrationSchema, "NATIVE", "v1.2.3", strings.Repeat("a", 64), 1000, 1000}
	body, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseRegistration(body)
	if err != nil || parsed != valid {
		t.Fatal("valid native registration rejected", err)
	}
	for name, input := range map[string][]byte{
		"empty":         nil,
		"duplicate":     bytes.Replace(body, []byte(`"client_uid":`), []byte(`"client_uid":2000,"client_uid":`), 1),
		"case-alias":    bytes.Replace(body, []byte(`"client_uid":`), []byte(`"CLIENT_UID":`), 1),
		"null-group":    bytes.Replace(body, []byte(`"client_gid":1000`), []byte(`"client_gid":null`), 1),
		"root-user":     bytes.Replace(body, []byte(`"client_uid":1000`), []byte(`"client_uid":0`), 1),
		"huge-user":     bytes.Replace(body, []byte(`"client_uid":1000`), []byte(`"client_uid":4294967295`), 1),
		"foreign-mode":  bytes.Replace(body, []byte(`"NATIVE"`), []byte(`"DEB"`), 1),
		"unknown":       bytes.Replace(body, []byte(`"schema":`), []byte(`"target":"/tmp/other","schema":`), 1),
		"wrong-version": bytes.Replace(body, []byte(`"v1.2.3"`), []byte(`"v1"`), 1),
		"path-version":  bytes.Replace(body, []byte(`"v1.2.3"`), []byte(`"../v1.2.3"`), 1),
		"trailing":      append(append([]byte(nil), body...), []byte(" {}")...),
		"excessive":     bytes.Repeat([]byte(" "), 16<<10+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseRegistration(input); !errors.Is(err, ErrUntrusted) {
				t.Fatal("untrusted state accepted", err)
			}
		})
	}
}

func TestRuntimeLeaseUsesActualSharedExclusiveLocksAndIdempotentHandoff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime.lock")
	first, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if err := waitFileLock(context.Background(), first, syscall.LOCK_SH); err != nil {
		t.Fatal(err)
	}
	second, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if err := waitFileLock(context.Background(), second, syscall.LOCK_SH); err != nil {
		t.Fatal(err)
	}
	writer, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if err := waitFileLock(ctx, writer, syscall.LOCK_EX); !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrTransition) {
		t.Fatal("exclusive lock ignored active readers or cancellation", err)
	}
	for _, file := range []*os.File{first, second} {
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
			t.Fatal(err)
		}
	}
	if err := waitFileLock(context.Background(), writer, syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	lease := &RuntimeLease{file: writer}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal("handoff is not idempotent", err)
	}
	if _, err := lease.Installed(); err == nil {
		t.Fatal("closed lease returned stale installation facts")
	}
	if err := waitFileLock(context.Background(), first, syscall.LOCK_SH); err != nil {
		t.Fatal("exclusive handoff retained the lock", err)
	}
	if err := syscall.Flock(int(first.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
}

func TestUnprivilegedObjectsDoNotBecomeProtectedState(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ordinary-user negative ownership test")
	}
	root := t.TempDir()
	path := filepath.Join(root, "state.json")
	if err := os.WriteFile(path, []byte(`{"schema":"untrusted"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if file, _, err := openProtectedFile(path); err == nil {
		_ = file.Close()
		t.Fatal("user-owned state became protected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readProtected(ctx, path, 1024); !errors.Is(err, context.Canceled) {
		t.Fatal("read cancellation cause lost")
	}
	if _, err := ReadInstalledRelease(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("installed-state cancellation cause lost")
	}
	var lease *RuntimeLease
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := lease.Installed(); !errors.Is(err, ErrUnmanaged) {
		t.Fatal("nil lease claimed installation")
	}
}
