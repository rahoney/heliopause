package promotion

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
)

func TestNPMProjectSourceFreezesLocalBytesAndRejectsLaterChanges(t *testing.T) {
	for _, fault := range []string{"unchanged", "bytes", "added", "mode", "replacement", "symlink"} {
		t.Run(fault, func(t *testing.T) {
			ctx := context.Background()
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(root, "project")
			if err := os.Mkdir(target, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, "package.json"), []byte(`{"scripts":{"postinstall":"node install.js"}}`), 0o600); err != nil {
				t.Fatal(err)
			}
			code := filepath.Join(target, "install.js")
			if err := os.WriteFile(code, []byte("process.exit(0)\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, ".env"), []byte("fixture-only dummy\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(filepath.Join(root, "evidence"))
			if err != nil {
				t.Fatal(err)
			}
			operation, err := domain.NewOperationID()
			if err != nil {
				t.Fatal(err)
			}
			owner, err := NewNPMProjectPromotion(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"), filepath.Join(root, "state"), store, operation, func(domain.ProjectDependencySnapshot) (*NPMProjectVerifiedCache, error) {
				return nil, errors.New("source-only fixture has no staged approval")
			})
			if err != nil {
				t.Fatal(err)
			}
			destination, err := domain.NewInstallTarget(target)
			if err != nil {
				t.Fatal(err)
			}
			install, err := domain.NewNPMProjectInstallContext(destination)
			if err != nil {
				t.Fatal(err)
			}
			guard, err := owner.Begin(ctx, install)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := guard.Close(); err != nil {
					t.Error(err)
				}
			}()
			subject, err := guard.FreezeNPMProjectSource(ctx)
			if err != nil || subject.Identity().Source().String() != "project-local" || subject.Identity().Variant() != "npm-source" {
				t.Fatal("root source masqueraded as a registry artifact", err)
			}
			parts := strings.Split(subject.ContentHandle(), ":")
			if len(parts) != 3 {
				t.Fatal("root source transport handle invalid")
			}
			archive, err := os.Open(filepath.Join(root, "intake", parts[1], "npm-source.tar"))
			if err != nil {
				t.Fatal(err)
			}
			reader := tar.NewReader(archive)
			names := map[string]int64{}
			for {
				header, err := reader.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					archive.Close()
					t.Fatal(err)
				}
				names[header.Name] = header.Mode
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			if len(names) != 2 || names["install.js"]&0o111 != 0o111 || names["package.json"] != 0o400 {
				t.Fatal("source copied hidden namespace or lost executable/manifest binding", names)
			}
			switch fault {
			case "bytes":
				err = os.WriteFile(code, []byte("process.exit(1)\n"), 0o755)
			case "added":
				err = os.WriteFile(filepath.Join(target, "new.js"), []byte("added"), 0o600)
			case "mode":
				err = os.Chmod(code, 0o600)
			case "replacement":
				replacement := filepath.Join(root, "replacement.js")
				if err = os.WriteFile(replacement, []byte("process.exit(0)\n"), 0o755); err == nil {
					err = os.Rename(replacement, code)
				}
			case "symlink":
				if err = os.Remove(code); err == nil {
					err = os.Symlink(filepath.Join(target, "package.json"), code)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			err = guard.VerifyUnchanged(ctx)
			if (err == nil) != (fault == "unchanged") {
				t.Fatal("source identity change was accepted or unchanged source rejected", fault, err)
			}
		})
	}
}
