package terraformprovider

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

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type nativeSandboxFunc func(context.Context, domain.SandboxRequest) (domain.SandboxResult, error)

func (f nativeSandboxFunc) Execute(c context.Context, r domain.SandboxRequest) (domain.SandboxResult, error) {
	return f(c, r)
}

type nativeVerificationFunc func(context.Context, domain.AcquiredArtifact) (domain.VerificationReport, error)

func (f nativeVerificationFunc) Verify(c context.Context, a domain.AcquiredArtifact) (domain.VerificationReport, error) {
	return f(c, a)
}

// Exercise real intake/package/native/report owners with controlled source
// authority and an unavailable runtime. This is not public signature or native
// execution qualification, and the overall required inspection stays incomplete.
func TestInspectorWiresNativeFactsAndPreservesThemWithoutRuntime(t *testing.T) {
	for _, writable := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "writable-executable"}[writable], func(t *testing.T) {
			body := make([]byte, 128)
			copy(body, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
			order := binary.LittleEndian
			order.PutUint16(body[16:], 2)
			order.PutUint16(body[18:], 62)
			order.PutUint32(body[20:], 1)
			order.PutUint64(body[32:], 64)
			order.PutUint16(body[52:], 64)
			order.PutUint16(body[54:], 56)
			order.PutUint16(body[56:], 1)
			order.PutUint16(body[58:], 64)
			order.PutUint32(body[64:], 1)
			order.PutUint32(body[68:], 5)
			if writable {
				order.PutUint32(body[68:], 7)
			}
			order.PutUint64(body[96:], uint64(len(body)))
			order.PutUint64(body[104:], uint64(len(body)))
			var archive bytes.Buffer
			writer := zip.NewWriter(&archive)
			header := &zip.FileHeader{Name: "terraform-provider-random_v3.7.2_x5", Method: zip.Store}
			header.SetMode(0o755)
			member, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := member.Write(body); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			var envelope bytes.Buffer
			envelope.WriteString("HAA-TERRAFORM-PROVIDER-1\n")
			for _, part := range [][]byte{[]byte("controlled discovery"), []byte("controlled versions"), []byte("controlled package"), []byte("controlled checksums"), []byte("controlled signature"), archive.Bytes()} {
				if err := binary.Write(&envelope, binary.BigEndian, uint64(len(part))); err != nil {
					t.Fatal(err)
				}
				envelope.Write(part)
			}
			root := t.TempDir()
			if err := os.Chmod(root, 0o700); err != nil {
				t.Fatal(err)
			}
			run, _ := domain.NewRunID()
			directory := filepath.Join(root, run.String())
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "provider.bundle"), envelope.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(envelope.Bytes())
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			identity, _ := domain.NewResolvedArtifactIdentity(artifactterraform.Source(), "hashicorp_random", "3.7.2", "linux/amd64")
			artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+run.String()+":linux/amd64", uint64(envelope.Len()))
			if err != nil {
				t.Fatal(err)
			}
			verified, executed := 0, 0
			verification := nativeVerificationFunc(func(_ context.Context, a domain.AcquiredArtifact) (domain.VerificationReport, error) {
				verified++
				id, _ := domain.NewCheckID("controlled-source")
				check, _ := domain.NewCheckExecution(id, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
				eid, _ := domain.NewEvidenceID("controlled-source-result")
				fact, err := domain.NewEvidence(eid, id, a.Identity(), a.Digest(), "fixture", "Controlled source authority for native wiring test.")
				if err != nil {
					return domain.VerificationReport{}, err
				}
				return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{fact})
			})
			sandbox := nativeSandboxFunc(func(context.Context, domain.SandboxRequest) (domain.SandboxResult, error) {
				executed++
				session, _ := domain.NewSandboxSessionID()
				return domain.NewSandboxResult(session, domain.SandboxIncomplete, "M3_RUNTIME_UNAVAILABLE", nil)
			})
			inspector, err := NewInspector(root, sandbox, verification)
			if err != nil {
				t.Fatal(err)
			}
			report, err := inspector.Inspect(context.Background(), artifact)
			if err != nil || verified != 1 || executed != 1 || (len(report.Findings()) != 0) != writable {
				t.Fatal("native result failed to traverse the actual inspector", err, verified, executed)
			}
			planned, err := inspector.ExpectedChecks(context.Background(), artifact, domain.CheckInspection)
			if err != nil || len(planned) != 5 {
				t.Fatal("native capability was omitted from the declared plan", err)
			}
			nativeChecks, dynamicIncomplete := 0, false
			for _, check := range report.Executions() {
				if check.ID().String() == "binary-content" || check.ID().String() == "binary-content-signals" {
					nativeChecks++
					if check.Status() != domain.ExecutionCompleted {
						t.Fatal("runtime absence erased completed native facts")
					}
				}
				if check.ID().String() == "terraform-provider-dynamic" {
					dynamicIncomplete = check.Status() != domain.ExecutionCompleted
				}
			}
			if nativeChecks != 2 || !dynamicIncomplete {
				t.Fatal("native facts hid incomplete required runtime coverage")
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 || entries[0].Name() != "provider.bundle" {
				t.Fatal("native inspection created extracted artifacts", err)
			}
		})
	}
}
