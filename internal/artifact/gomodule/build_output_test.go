package gomodule

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func buildOutputInputsFixture(t *testing.T) domain.ProjectBuildInputs {
	t.Helper()
	target, _ := domain.NewInstallTarget("/fixture/project")
	install, _ := domain.NewInstallContext(target)
	snapshot, err := BuildProjectSnapshot(install, nil, []byte("example.com/app go@1.26.0\ngo@1.26.0 toolchain@go1.26.0\n"), []byte("module example.com/app\ngo 1.26.0\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	local, _ := domain.NewSourceID("project-local")
	run, _ := domain.NewRunID()
	artifacts := make([]domain.AcquiredArtifact, 2)
	for index, variant := range []string{"go-source", "go-cache"} {
		version := "snapshot"
		if index == 1 {
			version = snapshot.GraphDigest().String()
		}
		identity, _ := domain.NewResolvedArtifactIdentity(local, "fixture", version, variant)
		digest, _ := domain.NewSHA256Digest(strings.Repeat(fmt.Sprint(index+1), 64))
		artifacts[index], err = domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":"+variant, 1024, "sha256:"+digest.String())
		if err != nil {
			t.Fatal(err)
		}
	}
	inputs, err := domain.NewProjectBuildInputs(snapshot, artifacts[0], artifacts[1], run, "./...")
	if err != nil {
		t.Fatal(err)
	}
	return inputs
}

func outputTar(t *testing.T, headers ...tar.Header) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := tar.NewWriter(&body)
	for _, header := range headers {
		if err := writer.WriteHeader(&header); err != nil {
			t.Fatal(err)
		}
		// Oversized advertised content must be rejected before reading its body.
		if header.Size > MaxBuildOutputFileBytes {
			return body.Bytes()
		}
		if _, err := io.CopyN(writer, strings.NewReader(strings.Repeat("x", int(header.Size))), header.Size); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func TestBuildOutputRejectsUntrustedArchiveBoundaries(t *testing.T) {
	good := tar.Header{Name: "./program", Typeflag: tar.TypeReg, Mode: 0o755, Uid: 1000, Gid: 1000, Size: 7}
	for _, name := range []string{"normal", "empty", "absolute", "parent", "nested", "reserved", "symlink", "hardlink", "fifo", "duplicate", "case-collision", "file-count", "oversize", "setuid", "writable", "uid", "pax", "trailing-data", "truncated", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Chmod(root, 0o700); err != nil {
				t.Fatal(err)
			}
			inputs := buildOutputInputsFixture(t)
			headers := []tar.Header{good}
			switch name {
			case "empty":
				headers = nil
			case "absolute":
				headers[0].Name = "/program"
			case "parent":
				headers[0].Name = "../program"
			case "nested":
				headers[0].Name = "sub/program"
			case "reserved":
				headers[0].Name = ".haa-build.json"
			case "symlink":
				headers[0].Typeflag = tar.TypeSymlink
				headers[0].Linkname = "/outside"
				headers[0].Size = 0
			case "hardlink":
				headers[0].Typeflag = tar.TypeLink
				headers[0].Linkname = "program"
				headers[0].Size = 0
			case "fifo":
				headers[0].Typeflag = tar.TypeFifo
				headers[0].Size = 0
			case "duplicate":
				headers = append(headers, good)
			case "case-collision":
				other := good
				other.Name = "Program"
				headers = append(headers, other)
			case "file-count":
				headers = nil
				for index := range MaxBuildOutputFiles + 1 {
					h := good
					h.Name = fmt.Sprintf("program-%03d", index)
					headers = append(headers, h)
				}
			case "oversize":
				headers[0].Size = MaxBuildOutputFileBytes + 1
			case "setuid":
				headers[0].Mode |= 0o4000
			case "writable":
				headers[0].Mode |= 0o002
			case "uid":
				headers[0].Uid = 0
			case "pax":
				headers[0].Format = tar.FormatPAX
				headers[0].PAXRecords = map[string]string{"custom": "untrusted"}
			}
			body := outputTar(t, headers...)
			if name == "trailing-data" {
				body = append(body, []byte("hidden")...)
			}
			if name == "truncated" {
				body = body[:512+3]
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if name == "cancelled" {
				cancel()
			}
			output, err := CaptureBuildOutput(ctx, root, inputs, bytes.NewReader(body))
			if name == "normal" || name == "empty" {
				if err != nil {
					t.Fatal(err)
				}
				artifact := output.Artifact()
				if err := output.Close(true); err != nil {
					t.Fatal(err)
				}
				files, err := ReadBuildOutput(ctx, root, artifact, nil)
				if err != nil || len(files) != len(headers) {
					t.Fatalf("exact data read: %v", err)
				}
			} else {
				if err == nil || output != nil {
					t.Fatal("untrusted output accepted")
				}
				entries, err := os.ReadDir(root)
				if err != nil || len(entries) != 0 {
					t.Fatal("failed output left partial intake")
				}
			}
		})
	}
}

func TestBuildOutputRechecksConsumedBytesAndAnchoredIdentity(t *testing.T) {
	for _, name := range []string{"byte-mutation", "symlink", "hardlink", "directory-replacement", "during-copy", "wrong-subject"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Chmod(root, 0o700); err != nil {
				t.Fatal(err)
			}
			inputs := buildOutputInputsFixture(t)
			output, err := CaptureBuildOutput(context.Background(), root, inputs, bytes.NewReader(outputTar(t, tar.Header{Name: "program", Typeflag: tar.TypeReg, Mode: 0o755, Uid: 1000, Gid: 1000, Size: 7})))
			if err != nil {
				t.Fatal(err)
			}
			artifact := output.Artifact()
			if err := output.Close(true); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, inputs.RunID().String(), "go-output.tar")
			mutate := func() {
				if err := os.Chmod(path, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("tampered"), 0o400); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, 0o400); err != nil {
					t.Fatal(err)
				}
			}
			switch name {
			case "byte-mutation":
				mutate()
			case "symlink":
				if err := os.Rename(path, path+"-real"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("go-output.tar-real", path); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, path+"-link"); err != nil {
					t.Fatal(err)
				}
			case "wrong-subject":
				identity, _ := domain.NewResolvedArtifactIdentity(artifact.Identity().Source(), artifact.Identity().Name(), inputs.RunID().String(), "go-source")
				artifact, _ = domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, artifact.Digest(), artifact.ContentHandle(), artifact.SizeBytes(), "sha256:"+artifact.Digest().String())
			}
			var consumer func(string, int64, io.Reader) error
			if name == "during-copy" || name == "directory-replacement" {
				consumer = func(_ string, _ int64, r io.Reader) error {
					_, err := io.Copy(io.Discard, r)
					if err != nil {
						return err
					}
					if name == "during-copy" {
						mutate()
					} else {
						parent := filepath.Dir(path)
						if err := os.Rename(parent, parent+"-real"); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(parent, 0o700); err != nil {
							t.Fatal(err)
						}
					}
					return nil
				}
			}
			files, err := ReadBuildOutput(context.Background(), root, artifact, consumer)
			if err == nil || files != nil {
				t.Fatal("changed consumed output returned success")
			}
			if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), root) {
				t.Fatal("raw Host path or absence authority exposed")
			}
		})
	}
}
