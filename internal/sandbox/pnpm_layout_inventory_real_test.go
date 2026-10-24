//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// Shape observation only. No output of this fixture grants layout validation,
// lifecycle completion, policy approval or Host installation authority.
func TestPNPMActualLayoutInventoryFixture(t *testing.T) {
	if os.Getenv("HELOX_PNPM_LAYOUT_INVENTORY_FIXTURE") != "1" {
		t.Fatal("declare HELOX_PNPM_LAYOUT_INVENTORY_FIXTURE=1 for native layout shape observation")
	}
	for _, fixture := range pnpmOfflineFixtures() {
		if !t.Run(fixture.name, func(t *testing.T) {
			captured := false
			qualifyPNPMOfflineFixture(t, fixture, func(ctx context.Context, runner CommandRunner, container string, controls []domain.ProjectControlFile) {
				output := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
				if err := runner.(streamingDockerOutput).RunOutput(ctx, &output, "docker", "cp", container+":"+pnpmProjectDirectory+"/node_modules", "-"); err != nil || output.failed {
					t.Fatal("read bounded native layout shape", err)
				}
				type entry struct {
					Name, Link, Digest string
					Type               byte
					Mode, Size         int64
					UID, GID           int
					Generated          string `json:",omitempty"`
				}
				var entries []entry
				parsed, err := artifactnpm.ParsePNPMLayoutArchive(ctx, output.body)
				if err != nil {
					t.Fatal("native layout envelope rejected", err)
				}
				for _, header := range parsed.Entries() {
					item := entry{Name: header.Name, Link: header.Link, Type: header.Kind, Mode: header.Mode, Size: header.Size, UID: header.UID, GID: header.GID, Digest: header.Digest.String()}
					if header.Kind == tar.TypeReg {
						if header.Name == "node_modules/.modules.yaml" || header.Name == "node_modules/.pnpm/lock.yaml" || strings.Contains(header.Name, "/.bin/") {
							if header.Size > 64<<10 {
								t.Fatal("native generated shape exceeds bounded capture")
							}
							body, err := parsed.ReadFile(header.Name)
							if err != nil {
								t.Fatal("native generated shape file unavailable", err)
							}
							item.Generated = string(body)
						}
					}
					entries = append(entries, item)
				}
				if err := ctx.Err(); err != nil {
					t.Fatal(err)
				}
				var manifest, lockfile []byte
				for _, control := range controls {
					if control.Name() == "package.json" {
						manifest = control.Body()
					} else if control.Name() == "pnpm-lock.yaml" {
						lockfile = control.Body()
					}
				}
				lock, err := artifactnpm.ParsePNPMLock(manifest, lockfile, runtimeidentity.PNPMVersion)
				if err != nil {
					t.Fatal("original native controls no longer parse", err)
				}
				packages := make(map[string]artifactnpm.PNPMLockedPackage)
				for _, pkg := range lock.Packages() {
					packages[pkg.Key] = pkg
				}
				files := make(map[string]byte, len(entries))
				for _, entry := range entries {
					files[entry.Name] = entry.Type
				}
				directories := map[string]bool{}
				for _, instance := range lock.Snapshots() {
					directory, err := artifactnpm.PNPMInstanceDirectory(instance.Key)
					if err != nil || directories[directory] {
						t.Fatal("snapshot directory unavailable or colliding", err)
					}
					directories[directory] = true
					name := "node_modules/.pnpm/" + directory + "/node_modules/" + packages[instance.Package].Name + "/package.json"
					if files[name] != tar.TypeReg && files[name] != tar.TypeLink {
						t.Fatal("native layout lacks the encoded snapshot manifest", name)
					}
				}
				t.Logf("pnpm_layout_codec instances=%d source=original_controls path_mapping=VERIFIED full_layout=NOT_VERIFIED", len(directories))
				body, err := json.Marshal(entries)
				if err != nil || len(body) > 1<<20 || len(entries) == 0 {
					t.Fatal("native shape record unavailable or over bound", err)
				}
				t.Logf("pnpm_layout_shape case=%s archive_sha256=%x entries=%d data=%s", fixture.name, sha256.Sum256(output.body), len(entries), body)
				captured = true
			})
			if !captured {
				t.Fatal("native prerequisite did not reach shape capture")
			}
		}) {
			t.FailNow()
		}
	}
}
