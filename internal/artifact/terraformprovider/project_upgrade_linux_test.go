package terraformprovider

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/storage"
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

func TestTerraformDefaultInitFirstSelectionAndExactRetainedLock(t *testing.T) {
	client, transport, _ := clientFixture(t)
	base := append([]byte(nil), transport.bodies[registryEndpoint+"/v1/providers/hashicorp/random/3.7.2/download/linux/amd64"]...)
	oldDigest := sha256Hex([]byte("synthetic acquisition bytes; not a verified provider archive"))
	for _, kind := range []string{"random", "time"} {
		archive := packageFixture(t, []string{"terraform-provider-" + kind + "_v3.7.2_x5"}, []os.FileMode{0755}, [][]byte{providerELFFixture()}).parts[5]
		body := bytes.ReplaceAll(base, []byte("random"), []byte(kind))
		body = bytes.ReplaceAll(body, []byte(oldDigest), []byte(sha256Hex(archive)))
		transport.bodies[registryEndpoint+"/v1/providers/hashicorp/"+kind+"/versions"] = upgradeVersions(t, "3.7.2")
		transport.bodies[registryEndpoint+"/v1/providers/hashicorp/"+kind+"/3.7.2/download/linux/amd64"] = body
		prefix := "https://releases.hashicorp.com/terraform-provider-" + kind + "/3.7.2/terraform-provider-" + kind + "_3.7.2"
		transport.bodies[prefix+"_SHA256SUMS"] = []byte("synthetic checksum metadata; not signature qualification")
		transport.bodies[prefix+"_SHA256SUMS.72D7468F.sig"] = []byte("synthetic signature")
		transport.bodies[prefix+"_linux_amd64.zip"] = archive
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	config := []byte("terraform {\n required_providers {\n random = {source=\"hashicorp/random\", version=\">=3.7.0, <3.8.0\"}\n time = {source=\"hashicorp/time\", version=\">=3.7.0, <3.8.0\"}\n }\n}\n")
	if err = os.WriteFile(filepath.Join(root, "main.tf"), config, 0600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(root)
	install, _ := domain.NewInstallContext(target)
	captured, err := CaptureProject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.ResolveProjectInit(context.Background(), install, captured.Controls)
	if err != nil || !first.Valid() || first.HasRequestedResolution() || len(first.Snapshot().Dependencies()) != 2 {
		t.Fatal("initial independent provider selection", err)
	}
	var lock []byte
	for _, control := range first.SelectedControls() {
		if control.Name() == LockControl {
			lock = append([]byte("# preserve exact user formatting\n"), control.Body()...)
		}
	}
	if len(lock) == 0 {
		t.Fatal("missing first lock")
	}
	if err = os.WriteFile(filepath.Join(root, LockControl), lock, 0640); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"random", "time"} {
		transport.bodies[registryEndpoint+"/v1/providers/hashicorp/"+kind+"/versions"] = upgradeVersions(t, "3.7.3", "3.7.2")
	}
	captured, err = CaptureProject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := client.ResolveProjectInit(context.Background(), install, captured.Controls)
	if err != nil || !retained.Valid() || retained.HasRequestedResolution() {
		t.Fatal("retained default init upgraded locked providers", err)
	}
	for _, artifact := range retained.Snapshot().Dependencies() {
		if artifact.Identity().Version() != "3.7.2" {
			t.Fatal("default init selected newer provider")
		}
	}
	for _, control := range retained.SelectedControls() {
		if control.Name() == LockControl && !bytes.Equal(control.Body(), lock) {
			t.Fatal("default init normalized retained lock")
		}
	}
	before := 0
	for _, n := range transport.calls {
		before += n
	}
	incompatible := bytes.ReplaceAll(config, []byte(">=3.7.0, <3.8.0"), []byte(">=4.0.0"))
	if err = os.WriteFile(filepath.Join(root, "main.tf"), incompatible, 0600); err != nil {
		t.Fatal(err)
	}
	captured, err = CaptureProject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.ResolveProjectInit(context.Background(), install, captured.Controls); err == nil {
		t.Fatal("incompatible retained lock accepted")
	}
	after := 0
	for _, n := range transport.calls {
		after += n
	}
	if before != after {
		t.Fatal("incompatible lock contacted registry")
	}
	if current, err := os.ReadFile(filepath.Join(root, LockControl)); err != nil || !bytes.Equal(current, lock) {
		t.Fatal("selection mutated original lock", err)
	}
}

func TestTerraformDefaultInitRejectsIncompleteLockBeforePublicIO(t *testing.T) {
	partial, err := EncodeLock([]LockedProvider{{Address: "registry.terraform.io/hashicorp/random", Version: "3.7.2", Hashes: []string{"zh:" + strings.Repeat("1", 64)}}})
	if err != nil {
		t.Fatal(err)
	}
	for name, lock := range map[string][]byte{"empty": {}, "partial": partial} {
		t.Run(name, func(t *testing.T) {
			client, transport, _ := clientFixture(t)
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			config := []byte("terraform {\n required_providers {\n random={source=\"hashicorp/random\",version=\"3.7.2\"}\n time={source=\"hashicorp/time\",version=\"3.7.2\"}\n }\n}\n")
			if err = os.WriteFile(filepath.Join(root, "main.tf"), config, 0600); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(root, LockControl), lock, 0600); err != nil {
				t.Fatal(err)
			}
			captured, err := CaptureProject(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			target, _ := domain.NewInstallTarget(root)
			install, _ := domain.NewInstallContext(target)
			if _, err = client.ResolveProjectInit(context.Background(), install, captured.Controls); err == nil {
				t.Fatal("incomplete retained lock was accepted")
			}
			if len(transport.calls) != 0 {
				t.Fatal("incomplete retained lock contacted public endpoints", transport.calls)
			}
		})
	}
}

func TestProjectSelectionPreparationUsesBoundedMemoryBeforeReservedAcquisition(t *testing.T) {
	for _, mode := range []string{"explicit-init", "default-init", "upgrade"} {
		for _, fault := range []string{"none", "cancelled", "malformed-archive", "memory-limit"} {
			t.Run(mode+"/"+fault, func(t *testing.T) {
				client, transport, ref := clientFixture(t)
				archive := packageFixture(t, []string{"terraform-provider-random_v3.7.2_x5"}, []os.FileMode{0755}, [][]byte{providerELFFixture()}).parts[5]
				key := registryEndpoint + "/v1/providers/hashicorp/random/3.7.2/download/linux/amd64"
				transport.bodies[key] = bytes.ReplaceAll(transport.bodies[key], []byte(sha256Hex([]byte("synthetic acquisition bytes; not a verified provider archive"))), []byte(sha256Hex(archive)))
				archiveURL := "https://releases.hashicorp.com/terraform-provider-random/3.7.2/terraform-provider-random_3.7.2_linux_amd64.zip"
				transport.bodies[archiveURL] = archive
				if fault == "malformed-archive" {
					transport.bodies[archiveURL] = []byte("not a provider archive")
				}
				if fault == "memory-limit" {
					client.preparedBytes = providerExpandedLimit
				}
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				root := filepath.Join(base, "haa")
				owner, err := storage.NewLocal(context.Background(), root, domain.DefaultStoragePolicy())
				if err != nil {
					t.Fatal(err)
				}
				defer owner.Close()
				client.intakeRoot = filepath.Join(root, "intake")
				project := filepath.Join(base, "project")
				if err := os.Mkdir(project, 0700); err != nil {
					t.Fatal(err)
				}
				config := []byte("terraform {\n required_providers {\n random={source=\"hashicorp/random\",version=\"3.7.2\"}\n }\n}\n")
				control := filepath.Join(project, "main.tf")
				if err := os.WriteFile(control, config, 0640); err != nil {
					t.Fatal(err)
				}
				original, err := os.Stat(control)
				if err != nil {
					t.Fatal(err)
				}
				foreign := filepath.Join(base, "foreign.note")
				if err := os.WriteFile(foreign, []byte("preserve unrelated user data"), 0600); err != nil {
					t.Fatal(err)
				}
				capture, err := CaptureProject(context.Background(), project)
				if err != nil {
					t.Fatal(err)
				}
				target, err := domain.NewInstallTarget(project)
				if err != nil {
					t.Fatal(err)
				}
				install, err := domain.NewInstallContext(target)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(ports.WithStorage(context.Background(), owner))
				defer cancel()
				if fault == "cancelled" {
					cancel()
				}
				var update domain.ProjectDependencyUpdate
				switch mode {
				case "explicit-init":
					update, err = client.ResolveProjectDependencyUpdate(ctx, ref, install, capture.Controls)
				case "default-init":
					update, err = client.ResolveProjectInit(ctx, install, capture.Controls)
				case "upgrade":
					update, err = client.ResolveProjectLockUpdate(ctx, install, capture.Controls, "")
				}
				if fault != "none" {
					if err == nil || update.Valid() {
						t.Fatal("failed preparation returned a valid selection", fault, err)
					}
					if fault == "cancelled" && !errors.Is(err, context.Canceled) {
						t.Fatal("cancellation cause lost", err)
					}
				} else if err != nil || !update.Valid() {
					t.Fatal("bounded selection before operation failed", err)
				}
				if _, e := os.Stat(root); !os.IsNotExist(e) {
					t.Fatal("preparation published intake or opened the storage journal", e)
				}
				current, e := os.Stat(control)
				body, re := os.ReadFile(control)
				if e != nil || re != nil || !os.SameFile(original, current) || original.Mode() != current.Mode() || !bytes.Equal(body, config) {
					t.Fatal("selection changed original user control", e, re)
				}
				if body, e := os.ReadFile(foreign); e != nil || string(body) != "preserve unrelated user data" {
					t.Fatal("selection removed foreign data", e)
				}
				if fault != "none" {
					return
				}
				// Preparation is not an approval: real acquisition still requires admission.
				selected := update.Snapshot().Dependencies()[0]
				calls := 0
				for _, n := range transport.calls {
					calls += n
				}
				run, err := domain.NewRunID()
				if err != nil {
					t.Fatal(err)
				}
				if _, err = client.Acquire(ctx, run, selected); err == nil || err.Error() != "invalid storage reservation" {
					t.Fatal("actual acquisition bypassed missing operation", err)
				}
				operation, err := domain.NewOperationID()
				if err != nil {
					t.Fatal(err)
				}
				ctx = ports.WithStorageOperation(ctx, operation)
				run, err = domain.NewRunID()
				if err != nil {
					t.Fatal(err)
				}
				acquired, err := client.Acquire(ctx, run, selected)
				if err != nil {
					t.Fatal("reserved acquisition failed", err)
				}
				bundle, err := ReadIntake(client.intakeRoot, acquired)
				if err != nil || bundle.ArchiveDigest() != sha256Hex(archive) {
					t.Fatal("frozen package identity changed", err)
				}
				afterCalls := 0
				for _, n := range transport.calls {
					afterCalls += n
				}
				if calls != afterCalls {
					t.Fatal("inspection reselected or downloaded package", calls, afterCalls)
				}
				// Real inspection is deliberately not run by this unit. Storage must retain it.
				if err := owner.Finish(ctx, true); err == nil {
					t.Fatal("uninspected intake was treated as complete")
				}
				if _, err := ReadIntake(client.intakeRoot, acquired); err != nil {
					t.Fatal("incomplete inspection intake was deleted", err)
				}
				journal, err := os.ReadFile(filepath.Join(root, ".storage/resources.jsonl"))
				if err != nil || !bytes.Contains(journal, []byte(operation.String())) || !bytes.Contains(journal, []byte(`"phase":"RETAINED"`)) {
					t.Fatal("operation accounting was lost", err)
				}
			})
		}
	}
}
