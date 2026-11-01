package sandbox

import (
	"archive/tar"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNPMProjectSourceReaderBindsTransportAndExecutableMode(t *testing.T) {
	for _, fault := range []string{"normal", "nonexecutable", "writable", "hidden", "traversal", "symlink", "missing manifest", "wrong source", "changed transport", "cancelled"} {
		t.Run(fault, func(t *testing.T) {
			intake := t.TempDir()
			members := []goBuildTarFixtureMember{{"install.js", "process.exit(0)\n", tar.TypeReg, 0o511}, {"package.json", `{"scripts":{"postinstall":"node install.js"}}`, tar.TypeReg, 0o400}}
			switch fault {
			case "nonexecutable":
				members[0].mode = 0o400
			case "writable":
				members[0].mode = 0o711
			case "hidden":
				members[0].name = ".env"
			case "traversal":
				members[0].name = "../install.js"
			case "symlink":
				members[0].kind = tar.TypeSymlink
				members[0].body = ""
			case "missing manifest":
				members = members[:1]
			}
			original := goBuildInputFixture(t, intake, "project", members)
			parts := strings.Split(original.ContentHandle(), ":")
			old := filepath.Join(intake, parts[1], "go-source.tar")
			current := filepath.Join(intake, parts[1], "npm-source.tar")
			if err := os.Rename(old, current); err != nil {
				t.Fatal(err)
			}
			identity, err := domain.NewResolvedArtifactIdentity(original.Identity().Source(), original.Identity().Name(), original.Identity().Version(), "npm-source")
			if err != nil {
				t.Fatal(err)
			}
			integrity, _ := original.DeclaredIntegrity()
			source, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, original.Digest(), "intake:"+parts[1]+":npm-source", original.SizeBytes(), integrity)
			if err != nil {
				t.Fatal(err)
			}
			if fault == "wrong source" {
				source = original
			}
			if fault == "changed transport" {
				body, err := os.ReadFile(current)
				if err != nil {
					t.Fatal(err)
				}
				body[0] ^= 1
				if err := os.Chmod(current, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(current, body, 0o400); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(current, 0o400); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "cancelled" {
				cancel()
			}
			reader, err := NewNPMProjectSourceReader(intake)
			if err != nil {
				t.Fatal(err)
			}
			verified := reader.VerifyBuildSource(ctx, source)
			layout, err := reader.ReadSource(ctx, source)
			allowed := fault == "normal" || fault == "nonexecutable"
			if !allowed {
				if verified == nil || err == nil || len(layout.Entries()) != 0 {
					t.Fatal("invalid local source transport admitted", fault, verified, err)
				}
				return
			}
			if verified != nil || err != nil {
				t.Fatal("valid npm local source rejected", verified, err)
			}
			body, err := layout.ReadFile("project/install.js")
			if err != nil || string(body) != "process.exit(0)\n" {
				t.Fatal("normalized source bytes changed", err)
			}
			for _, entry := range layout.Entries() {
				if entry.UID != 1000 || entry.GID != 1000 {
					t.Fatal("untrusted source owner survived normalization")
				}
				if entry.Name == "project/install.js" && entry.Mode != members[0].mode {
					t.Fatal("source executable bits changed")
				}
			}
			// The new source capability must not widen the old Go adapter's identity.
			goReader, err := NewGoBuildSourceReader(intake)
			if err != nil {
				t.Fatal(err)
			}
			if err := goReader.VerifyBuildSource(ctx, source); err == nil {
				t.Fatal("npm source acquired Go source capability")
			}
		})
	}
}
