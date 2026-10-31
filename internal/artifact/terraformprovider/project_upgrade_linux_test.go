package terraformprovider

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestTerraformUpgradeFreezesAllIndependentProviders(t *testing.T) {
	client, transport, _ := clientFixture(t)
	base := append([]byte(nil), transport.bodies[registryEndpoint+"/v1/providers/hashicorp/random/3.7.2/download/linux/amd64"]...)
	oldDigest := sha256Hex([]byte("synthetic acquisition bytes; not a verified provider archive"))
	for _, kind := range []string{"random", "time"} {
		filename := "terraform-provider-" + kind + "_v3.7.2_x5"
		archive := packageFixture(t, []string{filename}, []os.FileMode{0o755}, [][]byte{providerELFFixture()}).parts[5]
		body := bytes.ReplaceAll(base, []byte("random"), []byte(kind))
		body = bytes.ReplaceAll(body, []byte(oldDigest), []byte(sha256Hex(archive)))
		transport.bodies[registryEndpoint+"/v1/providers/hashicorp/"+kind+"/versions"] = upgradeVersions(t, "3.7.2", "3.7.1")
		transport.bodies[registryEndpoint+"/v1/providers/hashicorp/"+kind+"/3.7.2/download/linux/amd64"] = body
		prefix := "https://releases.hashicorp.com/terraform-provider-" + kind + "/3.7.2/terraform-provider-" + kind + "_3.7.2"
		transport.bodies[prefix+"_SHA256SUMS"] = []byte("synthetic checksum metadata; not signed source qualification")
		transport.bodies[prefix+"_SHA256SUMS.72D7468F.sig"] = []byte("synthetic signature")
		transport.bodies[prefix+"_linux_amd64.zip"] = archive
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	configuration := []byte(`terraform {
 required_providers {
  random = { source = "hashicorp/random", version = "~>3.7.0" }
  time = { source = "hashicorp/time", version = ">=3.7.0, <3.8.0" }
 }
}
`)
	if err := os.WriteFile(filepath.Join(root, "main.tf"), configuration, 0o600); err != nil {
		t.Fatal(err)
	}
	oldHash := "zh:" + strings.Repeat("1", 64)
	old, err := EncodeLock([]LockedProvider{{Address: "registry.terraform.io/hashicorp/random", Version: "3.7.1", Hashes: []string{oldHash}}, {Address: "registry.terraform.io/hashicorp/time", Version: "3.7.1", Hashes: []string{oldHash}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, LockControl), old, 0o600); err != nil {
		t.Fatal(err)
	}
	capture, err := CaptureProject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(root)
	install, _ := domain.NewInstallContext(target)
	update, err := client.ResolveProjectLockUpdate(context.Background(), install, capture.Controls, "")
	if err != nil || !update.Valid() || update.HasRequestedResolution() || len(update.Snapshot().Dependencies()) != 2 {
		t.Fatal("complete independent provider selection failed", err)
	}
	for _, control := range update.SelectedControls() {
		if control.Name() == ConfigurationControl {
			for _, original := range capture.Controls {
				if original.Name() == ConfigurationControl && original.Digest() != control.Digest() {
					t.Fatal("configuration changed")
				}
			}
		} else {
			locks, err := ParseLock(control.Body())
			if err != nil || len(locks) != 2 {
				t.Fatal("complete lock lost", err)
			}
			for _, locked := range locks {
				if locked.Version != "3.7.2" || len(locked.Hashes) != 2 {
					t.Fatal("did not upgrade every provider", locked)
				}
				for _, h := range locked.Hashes {
					if h == oldHash {
						t.Fatal("old-version hash retained for changed package")
					}
				}
			}
		}
	}
	for _, kind := range []string{"random", "time"} {
		if transport.calls[registryEndpoint+"/v1/providers/hashicorp/"+kind+"/versions"] != 1 {
			t.Fatal("versions were reselected")
		}
	}
	calls := map[string]int{}
	for k, v := range transport.calls {
		calls[k] = v
		transport.bodies[k] = []byte("changed after selection")
	}
	for _, artifact := range update.Snapshot().Dependencies() {
		run, _ := domain.NewRunID()
		acquired, err := client.Acquire(context.Background(), run, artifact)
		if err != nil {
			t.Fatal(err)
		}
		bundle, err := ReadIntake(client.intakeRoot, acquired)
		if err != nil {
			t.Fatal(err)
		}
		registry, archive, err := ParseIntegrity(artifact.DeclaredIntegrity())
		if err != nil || registry != bundle.RegistryDigest() || archive != bundle.ArchiveDigest() {
			t.Fatal("selected subject changed", err)
		}
	}
	for k, v := range transport.calls {
		if calls[k] != v {
			t.Fatal("inspection acquisition repeated live selection/download", k)
		}
	}
	got, _ := os.ReadFile(filepath.Join(root, LockControl))
	config, _ := os.ReadFile(filepath.Join(root, "main.tf"))
	if !bytes.Equal(got, old) || !bytes.Equal(config, configuration) {
		t.Fatal("selection modified the original project")
	}
}
