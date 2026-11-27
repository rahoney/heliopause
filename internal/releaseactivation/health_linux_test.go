//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == ProtectedHealthArgument {
		err := RunProtectedHealth(context.Background(), func(context.Context, bool) error {
			return os.WriteFile(filepath.Join(os.Getenv("HOME"), "unexpected-health-callback"), []byte("borrowed"), 0600)
		})
		if err != nil {
			os.Exit(37)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestHealthFramesAndPrivateBorrowingRequireAuthority(t *testing.T) {
	message := healthMessage{journalFixture().Old, journalFixture().ManifestSHA256, "CANDIDATE"}
	body, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := parseHealthMessage(body)
	if err != nil || decoded.Old != message.Old || decoded.Mode != "CANDIDATE" {
		t.Fatal("health frame changed", err)
	}
	for _, bad := range [][]byte{
		bytes.Replace(body, []byte(`"mode":"CANDIDATE"`), []byte(`"mode":"COMMAND"`), 1),
		bytes.Replace(body, []byte(`"mode":"CANDIDATE"`), []byte(`"mode":"CANDIDATE","mode":"ROLLBACK"`), 1),
		bytes.Replace(body, []byte(`"old":`), []byte(`"argv":"other","old":`), 1),
		append(append([]byte(nil), body...), []byte(`{}`)...),
	} {
		if _, err := parseHealthMessage(bad); err == nil {
			t.Fatal("foreign health operation accepted")
		}
	}
	for _, stage := range []string{"../outside", ".prepared-../../outside", ".prepared-", ".prepared-a\x00b", ".prepared-a\\b"} {
		if _, err := healthWorkspacePath(stage); err == nil {
			t.Fatal("foreign health path accepted")
		}
	}
	if _, err := healthWorkspacePath(".prepared-fixture"); err != nil {
		t.Fatal(err)
	}
	if err := readControl(context.Background(), bytes.NewBufferString("ACTIVATE1\n"), "RELEASE1\n"); err == nil {
		t.Fatal("wrong control command accepted")
	}
	if err := readControl(context.Background(), bytes.NewBufferString("HEAL"), "HEALTHY1\n"); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal("truncated cause lost")
	}
	ctx := context.WithValue(context.Background(), borrowedRuntimeKey{}, borrowedRuntime{})
	if _, err := AcquireRuntimeLease(ctx); err == nil {
		t.Fatal("unissued identity used borrowed access")
	}
	if _, err := AcquireUpdateOwner(context.Background(), &RuntimeLease{borrowed: true}); !errors.Is(err, ErrTransition) {
		t.Fatal("borrowed health started recursive update")
	}
	for _, body := range []string{"Uid:\t1000\t1000\t1000\t1000\nGid:\t1000\t1000\t1000\t1000\n"} {
		if !ordinaryKernelCredentials([]byte(body), 1000, 1000) {
			t.Fatal("actual ordinary credential form rejected")
		}
	}
	for _, body := range []string{"Uid:\t0\t0\t0\t0\nGid:\t0\t0\t0\t0\n", "Uid:\t1000\t0\t1000\t1000\nGid:\t1000\t1000\t1000\t1000\n", "Uid:\t1000\t1000\t1000\t1000\n"} {
		if ordinaryKernelCredentials([]byte(body), 1000, 1000) {
			t.Fatal("mixed or unknown child credential admitted")
		}
	}
}
