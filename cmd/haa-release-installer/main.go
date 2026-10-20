// haa-release-installer is the separate, narrow protected installation role.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "serve" || os.Args[1] == "recover" || os.Args[1] == "register" || os.Args[1] == "bootstrap-files") {
		// Host tools and SDK trust are fixed by the installation. Never inherit
		// caller HOME/TMPDIR/proxy/tool configuration in the privileged role.
		os.Clearenv()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || len(args) == 1 && (args[0] == "--help" || args[0] == "help") {
		if _, err := fmt.Fprintln(stdout, "Heliopause Artifact Airlock release installer\nusage: haa-release-installer serve | recover | register | bootstrap-files"); err != nil {
			return 1
		}
		return 0
	}
	if len(args) == 1 && args[0] == "--version" {
		if _, err := fmt.Fprintln(stdout, "haa-release-installer protocol=HAAUPD1"); err != nil {
			return 1
		}
		return 0
	}
	if len(args) != 1 || args[0] != "serve" && args[0] != "recover" && args[0] != "register" && args[0] != "bootstrap-files" {
		_, _ = fmt.Fprintln(stderr, "haa-release-installer: unsupported operation")
		return 1
	}
	if err := runOperation(ctx, args[0]); err != nil {
		// The public CLI owns presentation. Never serialize raw OS errors or
		// candidate data from this privileged role.
		_, _ = fmt.Fprintln(stderr, "haa-release-installer: SELF_UPDATE_PROTECTED_OPERATION_FAILED")
		return 1
	}
	if args[0] == "bootstrap-files" {
		if _, err := fmt.Fprintln(stdout, "native-files=PREPARED registration=PENDING health=UNVERIFIED"); err != nil {
			return 1
		}
	}
	return 0
}
