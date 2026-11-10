//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// Static layout binding only. No output grants lifecycle completion, policy
// approval or Host installation authority.
func TestPNPMActualLayoutInventoryFixture(t *testing.T) {
	if os.Getenv("HELOX_PNPM_LAYOUT_INVENTORY_FIXTURE") != "1" {
		t.Fatal("declare HELOX_PNPM_LAYOUT_INVENTORY_FIXTURE=1 for native layout shape observation")
	}
	for _, fixture := range pnpmOfflineFixtures() {
		if !t.Run(fixture.name, func(t *testing.T) {
			captured := false
			qualifyPNPMOfflineFixture(t, fixture, func(ctx context.Context, runner CommandRunner, container string, controls []domain.ProjectControlFile, input artifactnpm.PNPMInspectionInput, inventories []artifactnpm.ArchiveInventory) {
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
				binCount := 0
				parsed, err := artifactnpm.ParsePNPMLayoutArchive(ctx, output.body)
				if err != nil {
					t.Fatal("native layout envelope rejected", err)
				}
				bound, err := input.BindLayout(ctx, inventories, parsed.RestrictDirectoryWrites(), pnpmProjectRoot+"/store")
				if err != nil {
					t.Fatal("native layout differs from complete acquired inputs", err)
				}
				if _, err := bound.ReadFile("node_modules/.pnpm-workspace-state-v1.json"); err == nil {
					t.Fatal("derived native cache exposed in bound payload")
				}
				for _, entry := range bound.Entries() {
					if entry.Name == "node_modules/.pnpm-workspace-state-v1.json" {
						t.Fatal("derived native cache entry in bound payload")
					}
				}
				instanceCount := 0
				for _, subject := range input.Artifacts() {
					instances, err := input.InstancesFor(subject)
					if err != nil {
						t.Fatal(err)
					}
					for _, instance := range instances {
						if _, err := bound.Target(subject, instance.Key); err != nil {
							t.Fatal("bound layout lost an exact acquired instance", err)
						}
						instanceCount++
					}
				}
				t.Logf("pnpm_layout_binding archives=%d instances=%d digest=%s full_static_layout=VERIFIED dynamic=NOT_VERIFIED", len(inventories), instanceCount, bound.Digest().String())
				checkPNPMLayoutMutations(t, ctx, input, inventories, parsed)
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
							if strings.Contains(header.Name, "/.bin/") {
								target, expected := fixture.bins[path.Base(header.Name)]
								if !expected {
									t.Fatal("fixture has an unexpected bin command")
								}
								if _, err := artifactnpm.ParsePNPMNodeBinShim(body, header.Name, "node_modules/.pnpm/"+target); err != nil {
									t.Fatal("native bin code differs from the reviewed full template", err)
								}
								binCount++
							}
						}
					}
					entries = append(entries, item)
				}
				if len(fixture.bins) != 0 && binCount != 3 {
					t.Fatal("fixture did not reach the three independent bin locations", binCount)
				}
				t.Logf("pnpm_layout_bin_code shims=%d full_code_format=VERIFIED archive_bin_binding=VERIFIED", binCount)
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
				t.Logf("pnpm_layout_codec instances=%d source=original_controls path_mapping=VERIFIED full_static_layout=VERIFIED", len(directories))
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

// Each negative reuses the single captured native run. Only parser/binder data
// changes; no mutated artifact is introduced or executed in a Sandbox.
func checkPNPMLayoutMutations(t *testing.T, ctx context.Context, input artifactnpm.PNPMInspectionInput, inventories []artifactnpm.ArchiveInventory, original artifactnpm.PNPMLayoutArchive) {
	t.Helper()
	count := 0
	for _, change := range []string{"payload", "missing", "extra", "edge", "bin", "metadata", "virtual-lock", "cache-json"} {
		var output bytes.Buffer
		writer := tar.NewWriter(&output)
		mutated := change == "extra"
		for _, entry := range original.Entries() {
			header := tar.Header{Name: entry.Name, Typeflag: entry.Kind, Mode: entry.Mode, Uid: entry.UID, Gid: entry.GID, Size: entry.Size, Linkname: entry.Link}
			var body []byte
			if entry.Kind == tar.TypeReg {
				var err error
				body, err = original.ReadFile(entry.Name)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !mutated {
				switch {
				case (change == "payload" || change == "missing") && strings.HasSuffix(entry.Name, "/package.json"):
					mutated = true
					if change == "missing" {
						continue
					}
					body = append(body, '\n')
				case change == "edge" && entry.Kind == tar.TypeSymlink:
					header.Linkname += "-foreign"
					mutated = true
				case change == "bin" && strings.Contains(entry.Name, "/.bin/") && entry.Kind == tar.TypeReg:
					body = append(body, []byte("\nprintf injected\n")...)
					mutated = true
				case change == "metadata" && entry.Name == "node_modules/.modules.yaml":
					body = bytes.Replace(body, []byte("isolated"), []byte("hoisted"), 1)
					mutated = true
				case change == "virtual-lock" && entry.Name == "node_modules/.pnpm/lock.yaml":
					body = bytes.Replace(body, []byte("sha512-"), []byte("sha512-A"), 1)
					mutated = true
				case change == "cache-json" && entry.Name == "node_modules/.pnpm-workspace-state-v1.json":
					body = []byte("{invalid-json}")
					mutated = true
				}
			}
			if entry.Kind == tar.TypeReg {
				header.Size = int64(len(body))
			}
			if err := writer.WriteHeader(&header); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write(body); err != nil {
				t.Fatal(err)
			}
		}
		if change == "extra" {
			if err := writer.WriteHeader(&tar.Header{Name: "node_modules/foreign", Typeflag: tar.TypeReg, Mode: 0o644, Uid: 1000, Gid: 1000, Size: 1}); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write([]byte{'!'}); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if !mutated {
			if change == "bin" {
				continue
			}
			t.Fatal("negative fixture failed to change its target", change)
		}
		parsed, err := artifactnpm.ParsePNPMLayoutArchive(ctx, output.Bytes())
		if err != nil {
			t.Fatal("negative failed before semantic binding", change, err)
		}
		if bound, err := input.BindLayout(ctx, inventories, parsed.RestrictDirectoryWrites(), pnpmProjectRoot+"/store"); err == nil || bound.Digest().String() != "" {
			t.Fatal("mutated native data admitted", change)
		}
		count++
	}
	t.Logf("pnpm_layout_binding_negatives cases=%d no_new_native_run=true no_artifact_execution=true", count)
}
