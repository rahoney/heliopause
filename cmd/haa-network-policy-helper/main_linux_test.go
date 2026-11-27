//go:build linux

package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestInertDefaultHelpAndVersionDoNotStartService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, args := range [][]string{nil, {"--help"}, {"help"}, {"--version"}} {
		var stdout, stderr bytes.Buffer
		if code := run(ctx, args, &stdout, &stderr); code != 0 || stdout.Len() == 0 || stderr.Len() != 0 {
			t.Fatal("inert entry prepared service or failed", args, code, stderr.String())
		}
	}
	for _, args := range [][]string{{"install"}, {"serve", "--target", "/tmp/other"}, {"serve", "--uid", "0"}, {"--identity", "other"}} {
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), args, &stdout, &stderr); code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "unsupported operation") {
			t.Fatal("arbitrary privileged request admitted", args, code)
		}
	}
	var stdout, stderr bytes.Buffer
	if code := run(ctx, []string{"serve"}, &stdout, &stderr); code != 1 || stdout.Len() != 0 {
		t.Fatal("canceled service prepared host state")
	}
}
