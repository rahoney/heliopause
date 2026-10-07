package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func cargoBuildPreparedFixture(t *testing.T) (string, domain.ProjectBuildInputs) {
	t.Helper()
	target, _ := domain.NewInstallTarget(t.TempDir())
	install, _ := domain.NewInstallContext(target)
	manifest := "[package]\nname='fixture'\nversion='0.1.0'\nedition='2021'\n"
	lock := "version=4\n[[package]]\nname='fixture'\nversion='0.1.0'\n"
	controls := []domain.ProjectControlDigest{}
	for name, body := range map[string]string{"Cargo.toml": manifest, "Cargo.lock": lock} {
		hash := sha256.Sum256([]byte(body))
		digest, _ := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
		control, _ := domain.NewProjectControlDigest(name, digest)
		controls = append(controls, control)
	}
	graph, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	snapshot, err := domain.NewDependencyFreeProjectSnapshot(install, artifactcargo.Source(), controls, graph)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	keyHash := sha256.Sum256([]byte(target.String()))
	key := hex.EncodeToString(keyHash[:])
	local, _ := domain.NewSourceID("project-local")
	convert := func(prefix, name, version string, members []goBuildTarFixtureMember) domain.AcquiredArtifact {
		original := goBuildInputFixture(t, root, prefix, members)
		parts := strings.Split(original.ContentHandle(), ":")
		variant := "cargo-" + strings.TrimPrefix(original.Identity().Variant(), "go-")
		if err := os.Rename(filepath.Join(root, parts[1], original.Identity().Variant()+".tar"), filepath.Join(root, parts[1], variant+".tar")); err != nil {
			t.Fatal(err)
		}
		identity, _ := domain.NewResolvedArtifactIdentity(local, name, version, variant)
		result, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, original.Digest(), "intake:"+parts[1]+":"+variant, original.SizeBytes(), "sha256:"+original.Digest().String())
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	source := convert("project", "project-"+key, "snapshot", []goBuildTarFixtureMember{
		{"Cargo.toml", manifest, tar.TypeReg, 0o400}, {"Cargo.lock", lock, tar.TypeReg, 0o400},
		{"src", "", tar.TypeDir, 0o700}, {"src/main.rs", "fn main() {}\n", tar.TypeReg, 0o400},
		{"testdata", "", tar.TypeDir, 0o700}, {"testdata/Cargo.toml", "intentionally malformed data", tar.TypeReg, 0o400},
	})
	cache := convert("cache", "cache-"+key, graph.String(), nil)
	run, _ := domain.NewRunID()
	inputs, err := domain.NewProjectBuildInputs(snapshot, source, cache, run, "default")
	if err != nil {
		t.Fatal(err)
	}
	return root, inputs
}

func TestCargoBuildPreparationBindsExactControlsAndReadOnlyConfiguration(t *testing.T) {
	root, inputs := cargoBuildPreparedFixture(t)
	reader, err := NewCargoBuildSourceReader(root)
	if err != nil || reader.VerifyBuildSource(context.Background(), inputs.Source()) != nil {
		t.Fatal("exact source not verified", err)
	}
	prepared, err := prepareCargoBuildInputs(context.Background(), root, inputs)
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.close()
	if !prepared.cargoConfiguration || len(prepared.manifest.members) != 8 || prepared.manifest.members["cargo-home/config.toml"].size != int64(len(cargoBuildSourceConfiguration)) {
		t.Fatal("controller configuration absent from inventory")
	}
	for _, failure := range []string{"", "config-content", "config-mode", "config-link", "config-owner", "config-missing"} {
		t.Run(failure, func(t *testing.T) {
			var body bytes.Buffer
			writer := tar.NewWriter(&body)
			for _, name := range []string{"project", "cache"} {
				if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o500, Uid: 1000, Gid: 1000}); err != nil {
					t.Fatal(err)
				}
			}
			if err := prepared.source.scan(context.Background(), &prepared.manifest, writer); err != nil {
				t.Fatal(err)
			}
			if err := prepared.cache.scan(context.Background(), &prepared.manifest, writer); err != nil {
				t.Fatal(err)
			}
			if failure == "" {
				if err := writeCargoBuildConfiguration(writer); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := writer.WriteHeader(&tar.Header{Name: "cargo-home", Typeflag: tar.TypeDir, Mode: 0o500, Uid: 1000, Gid: 1000}); err != nil {
					t.Fatal(err)
				}
				data := cargoBuildSourceConfiguration
				header := &tar.Header{Name: "cargo-home/config.toml", Typeflag: tar.TypeReg, Mode: 0o400, Uid: 1000, Gid: 1000, Size: int64(len(data))}
				switch failure {
				case "config-content":
					data = strings.Replace(data, "haa-verified", "bad-verified", 1)
				case "config-mode":
					header.Mode = 0o600
				case "config-link":
					header.Typeflag = tar.TypeLink
					header.Linkname = "project/Cargo.toml"
					header.Size = 0
					data = ""
				case "config-owner":
					header.Uid = 0
				}
				if failure != "config-missing" {
					if err := writer.WriteHeader(header); err != nil {
						t.Fatal(err)
					}
					if _, err := writer.Write([]byte(data)); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := scanGoBuildMaterialization(context.Background(), &body, prepared.manifest); (err == nil) != (failure == "") {
				t.Fatalf("configuration materialization %s: %v", failure, err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := prepareCargoBuildInputs(ctx, root, inputs); err == nil {
		t.Fatal("cancelled input accepted")
	}
	if err := reader.VerifyBuildSource(ctx, inputs.Source()); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled reader proceeded")
	}
	changed, err := domain.NewProjectBuildInputs(inputs.Snapshot(), inputs.Source(), inputs.Cache(), inputs.RunID(), "./...")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepareCargoBuildInputs(context.Background(), root, changed); err == nil {
		t.Fatal("unapproved selector accepted")
	}
}

func TestCargoBuildRuntimeKeepsSeparateOwnedMountsAndOfflineBounds(t *testing.T) {
	input := closureVolume{name: "haa-input", transaction: "tx", createdAt: "created", mountpoint: "/volume/input", projectBuildDestination: cargoBuildGuestInput}
	target := closureVolume{name: "haa-target", transaction: "tx-target", createdAt: "created", mountpoint: "/volume/target", projectBuildDestination: cargoBuildGuestTarget, projectBuildExecutable: true}
	for _, readOnly := range []bool{false, true} {
		args, err := cargoBuildCreateArguments(input, target, readOnly)
		if err != nil {
			t.Fatal(err)
		}
		inputMount, _ := input.mountArgument(readOnly)
		targetMount, _ := target.mountArgument(false)
		var mounts []string
		for i, arg := range args {
			if arg == "--mount" {
				mounts = append(mounts, args[i+1])
			}
		}
		if !reflect.DeepEqual(mounts, []string{inputMount, targetMount}) {
			t.Fatal("mount authority changed")
		}
		joined := strings.Join(args, "\n")
		for _, required := range []string{"--network\nnone", "--read-only", "--memory\n512m", "--pids-limit\n64", "--cpus\n1", "--workdir\n/\n", "CARGO_HOME=" + cargoBuildGuestInput + "/cargo-home", "CARGO_NET_OFFLINE=true", PinnedCargoRuntime().ImageReference} {
			if !strings.Contains(joined, required) {
				t.Fatal("runtime omitted", required)
			}
		}
		topology, ok := cargoBuildExpectedTopology(readOnly)
		if !ok {
			t.Fatal("missing topology")
		}
		found := 0
		for _, mount := range topology {
			if mount.Mountpoint == cargoBuildGuestInput {
				found++
				if mount.ReadOnly != readOnly {
					t.Fatal("input writable at build")
				}
			}
			if mount.Mountpoint == cargoBuildGuestTarget {
				found++
				if mount.ReadOnly {
					t.Fatal("target readonly")
				}
			}
		}
		if found != 2 {
			t.Fatal("input/target topology missing")
		}
	}
	for _, failure := range []string{"input-role", "target-role", "alias", "transaction", "identity"} {
		i, o := input, target
		switch failure {
		case "input-role":
			i.projectBuildExecutable = true
		case "target-role":
			o.projectBuildExecutable = false
		case "alias":
			o.name = i.name
		case "transaction":
			o.transaction = "foreign"
		case "identity":
			o.createdAt = ""
		}
		if _, err := cargoBuildCreateArguments(i, o, true); err == nil {
			t.Fatal("substituted mount accepted", failure)
		}
	}
	budget := traceBudgetForProfile(cargoBuildProfile)
	if budget.events != 10000 || budget.bytes != 2<<20 || !validObserverProfile(cargoBuildProfile) {
		t.Fatal("Cargo budget/registration expanded")
	}
}

func TestCargoBuildFirstCommandFailureKeepsTrustedBoundedStatus(t *testing.T) {
	op := &cargoBuildOperation{builder: &ObservedCargoBuilder{runner: &goBuildFailureOutput{failure: context.DeadlineExceeded}}}
	runtime := &goBuildRuntime{id: strings.Repeat("a", 64)}
	if _, err := op.artifactOutput(context.Background(), runtime, "build"); err == nil {
		t.Fatal("command failure became success")
	}
	op.builder.runner = &goBuildFailureOutput{failure: errors.New("artifact-controlled-secret")}
	if _, err := op.artifactOutput(context.Background(), runtime, "build"); err == nil {
		t.Fatal("second command failure became success")
	}
	err := &isolatedCargoBuildFailure{cause: errors.New("private-host-path"), phase: "BUILD", commandReason: op.commandReason}
	if !strings.Contains(err.Error(), "command_status=DEADLINE_EXCEEDED") || strings.Contains(err.Error(), "private-host") || strings.Contains(err.Error(), "secret") {
		t.Fatal("first trusted status lost or raw text admitted")
	}
}
