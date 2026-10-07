package sandbox

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Literal Linux/amd64 syscall fixtures require no host compiler or execution.
// They test the Terraform ARTIFACT boundary, not public signature acceptance.
func terraformSecurityELF(kind string) []byte {
	const base = uint64(0x400000)
	var code, data []byte
	imm := func(op byte, v uint32) { code = append(code, op); code = binary.LittleEndian.AppendUint32(code, v) }
	address := func(op byte, offset int) {
		code = append(code, 0x48, op)
		code = binary.LittleEndian.AppendUint64(code, base+120+uint64(offset))
	}
	syscall := func() { code = append(code, 0x0f, 0x05) }
	switch kind {
	case "filesystem":
		imm(0xb8, 257)
		imm(0xbf, 0xffffff9c)
		address(0xbe, 48)
		imm(0xba, 0)
		code = append(code, 0x45, 0x31, 0xd2)
		syscall()
		code = append(code, 0x89, 0xc7)
		imm(0xb8, 3)
		syscall()
		data = []byte("/etc/passwd\x00")
	case "network":
		imm(0xb8, 41)
		imm(0xbf, 2)
		imm(0xbe, 1)
		imm(0xba, 0)
		syscall()
		code = append(code, 0x89, 0xc7)
		imm(0xb8, 42)
		address(0xbe, 55)
		imm(0xba, 16)
		syscall()
		data = []byte{2, 0, 0, 9, 127, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0}
	case "unexpected-process":
		imm(0xb8, 59)
		address(0xbf, 38)
		address(0xbe, 48)
		code = append(code, 0x31, 0xd2)
		syscall()
		data = append([]byte("/bin/true\x00"), make([]byte, 16)...)
		binary.LittleEndian.PutUint64(data[10:], base+120+38)
	}
	imm(0xb8, 60)
	code = append(code, 0x31, 0xff)
	syscall()
	// Data offsets above are independently checked against fixed code lengths.
	expected := map[string]int{"normal": 9, "filesystem": 48, "network": 55, "unexpected-process": 38}
	if len(code) != expected[kind] {
		panic("trusted fixture instruction layout changed")
	}
	b := make([]byte, 120)
	copy(b, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(b[16:], 2)
	binary.LittleEndian.PutUint16(b[18:], 62)
	binary.LittleEndian.PutUint32(b[20:], 1)
	binary.LittleEndian.PutUint64(b[24:], base+120)
	binary.LittleEndian.PutUint64(b[32:], 64)
	binary.LittleEndian.PutUint16(b[52:], 64)
	binary.LittleEndian.PutUint16(b[54:], 56)
	binary.LittleEndian.PutUint16(b[56:], 1)
	binary.LittleEndian.PutUint32(b[64:], 1)
	binary.LittleEndian.PutUint32(b[68:], 5)
	binary.LittleEndian.PutUint64(b[80:], base)
	binary.LittleEndian.PutUint64(b[88:], base)
	binary.LittleEndian.PutUint64(b[96:], uint64(120+len(code)+len(data)))
	binary.LittleEndian.PutUint64(b[104:], uint64(120+len(code)+len(data)))
	binary.LittleEndian.PutUint64(b[112:], 4096)
	return append(append(b, code...), data...)
}

func terraformSecurityIntake(t *testing.T, root, kind string) domain.AcquiredArtifact {
	t.Helper()
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	header := &zip.FileHeader{Name: "terraform-provider-fixture_v1.0.0_x5", Method: zip.Store}
	header.SetMode(0o755)
	f, e := w.CreateHeader(header)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.Write(terraformSecurityELF(kind)); e != nil {
		t.Fatal(e)
	}
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
	var raw bytes.Buffer
	raw.WriteString("HAA-TERRAFORM-PROVIDER-1\n")
	for _, part := range [][]byte{[]byte("synthetic discovery"), []byte("synthetic versions"), []byte("synthetic package"), []byte("synthetic checksums"), []byte("synthetic signature"), archive.Bytes()} {
		if e := binary.Write(&raw, binary.BigEndian, uint64(len(part))); e != nil {
			t.Fatal(e)
		}
		raw.Write(part)
	}
	run, e := domain.NewRunID()
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(root, run.String())
	if e := os.Mkdir(dir, 0o700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "provider.bundle"), raw.Bytes(), 0o600); e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(raw.Bytes())
	digest, e := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if e != nil {
		t.Fatal(e)
	}
	identity, e := domain.NewResolvedArtifactIdentity(artifactterraform.Source(), "fixture_fixture", "1.0.0", "linux/amd64")
	if e != nil {
		t.Fatal(e)
	}
	artifact, e := domain.NewAcquiredArtifact(identity, digest, "intake:"+run.String()+":linux/amd64", uint64(raw.Len()))
	if e != nil {
		t.Fatal(e)
	}
	return artifact
}

func TestLinuxTerraformProviderSecurityIntegration(t *testing.T) {
	if os.Getenv("HELOX_TERRAFORM_PROVIDER_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	for _, kind := range []string{"normal", "network", "filesystem", "unexpected-process"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			if e := os.Chmod(root, 0o700); e != nil {
				t.Fatal(e)
			}
			artifact := terraformSecurityIntake(t, root, kind)
			supervisor := integrationObserverSupervisor(t)
			defer supervisor.Close()
			runner := integrationRunner{t: t}
			elf, e := NewGitHubELFBackend(runner, root, supervisor.Observer(), integrationCapabilityProbe(runner))
			if e != nil {
				t.Fatal(e)
			}
			backend := &TerraformProviderBackend{elf: elf}
			request, e := domain.NewSandboxRequest(artifact)
			if e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			result, e := backend.Execute(ctx, request)
			if e != nil {
				t.Fatal(e)
			}
			code, _ := result.LimitationCode()
			observed := false
			for _, o := range result.Observations() {
				if kind == "network" && o.Category() == domain.ObservationNetwork {
					observed = true
				}
				if kind == "unexpected-process" && o.Category() == domain.ObservationProcess && o.Subject() == "process-exec-unexpected" {
					observed = true
				}
			}
			switch kind {
			case "normal":
				if result.Status() != domain.SandboxCompleted {
					t.Fatalf("minimal normal ELF failed %s/%s reason=%s", result.Status(), code, integrationObserverFaultReason(supervisor))
				}
			case "network":
				if !observed {
					t.Fatalf("network syscall observation missing %s/%s reason=%s", result.Status(), code, integrationObserverFaultReason(supervisor))
				}
			case "filesystem":
				if result.Status() != domain.SandboxIncomplete || integrationObserverFaultReason(supervisor) != "STREAM_FAULT" {
					t.Fatalf("filesystem access accepted %s/%s reason=%s", result.Status(), code, integrationObserverFaultReason(supervisor))
				}
			case "unexpected-process":
				if !observed {
					t.Fatalf("unexpected exec fact lost %s/%s reason=%s", result.Status(), code, integrationObserverFaultReason(supervisor))
				}
			}
			summary, e := result.ObservationSummary()
			if e != nil {
				t.Fatal(e)
			}
			t.Logf("actual_provider_boundary=%s %s", kind, summary)
		})
	}
}
