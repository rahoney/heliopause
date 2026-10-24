package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDefaultAndHelpNeverPrepareAnInstallation(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"help"}} {
		var stdout, stderr bytes.Buffer
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if code := run(ctx, args, &stdout, &stderr); code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "Heliopause Artifact Airlock") {
			t.Fatal("inert helper entry failed", args, code, stderr.String())
		}
	}
	var version, diagnostic bytes.Buffer
	if code := run(context.Background(), []string{"--version"}, &version, &diagnostic); code != 0 || version.String() != "haa-release-installer protocol=HAAUPD1\n" || diagnostic.Len() != 0 {
		t.Fatal("inert version entry failed")
	}
	for _, args := range [][]string{{"install", "/tmp/user-path"}, {"serve", "--target", "/etc/other"}, {"recover", "--uid", "0"}, {"register", "--uid", "1000"}, {"register", "--target", "/tmp/other"}, {"bootstrap-files", "--target", "/tmp/other"}, {"bootstrap-files", "--uid", "1000"}} {
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), args, &stdout, &stderr); code != 1 || stdout.Len() != 0 {
			t.Fatal("caller-selected privileged operation accepted", args)
		}
	}
}

func TestCanceledProtectedOperationFailsWithoutRawDiagnostics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, operation := range []string{"serve", "recover", "register", "bootstrap-files"} {
		var stdout, stderr bytes.Buffer
		if code := run(ctx, []string{operation}, &stdout, &stderr); code != 1 || stdout.Len() != 0 || stderr.String() != "haa-release-installer: SELF_UPDATE_PROTECTED_OPERATION_FAILED\n" {
			t.Fatal("protected failure leaked or succeeded", operation, code, stderr.String())
		}
	}
}
