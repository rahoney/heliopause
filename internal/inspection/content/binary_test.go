package content

import (
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func binaryFixture() []byte {
	body := make([]byte, 1024)
	copy(body, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	order := binary.LittleEndian
	order.PutUint16(body[16:], uint16(elf.ET_EXEC))
	order.PutUint16(body[18:], uint16(elf.EM_X86_64))
	order.PutUint32(body[20:], 1)
	order.PutUint64(body[32:], 64)
	order.PutUint64(body[40:], 256)
	order.PutUint16(body[52:], 64)
	order.PutUint16(body[54:], 56)
	order.PutUint16(body[56:], 1)
	order.PutUint16(body[58:], 64)
	order.PutUint16(body[60:], 4)
	order.PutUint16(body[62:], 3)
	order.PutUint32(body[64:], uint32(elf.PT_LOAD))
	order.PutUint32(body[68:], uint32(elf.PF_R|elf.PF_X))
	order.PutUint64(body[96:], uint64(len(body)))
	order.PutUint64(body[104:], uint64(len(body)))
	for index, name := range []uint32{1, 6} {
		header := body[320+index*64:]
		order.PutUint32(header, name)
		order.PutUint32(header[4:], uint32(elf.SHT_PROGBITS))
		order.PutUint64(header[24:], 960)
		order.PutUint64(header[32:], 4)
	}
	order.PutUint32(body[452:], uint32(elf.SHT_STRTAB))
	order.PutUint64(body[472:], 880)
	order.PutUint64(body[480:], 11)
	copy(body[880:], []byte("\x00seg0\x00seg1\x00"))
	return body
}

func TestBinaryFactsKeepLayoutSignalsSeparateFromPackedCoverage(t *testing.T) {
	input, _, _ := sourceFixture(t)
	for _, scenario := range []string{"normal", "marker-only", "writable-executable-program", "executable-stack", "writable-executable-section", "lifecycle-only", "compressed-debug", "packed"} {
		t.Run(scenario, func(t *testing.T) {
			body := binaryFixture()
			order := binary.LittleEndian
			wantFinding, wantComplete := false, true
			switch scenario {
			case "marker-only":
				copy(body[960:], "UPX!")
			case "writable-executable-program":
				order.PutUint32(body[68:], uint32(elf.PF_R|elf.PF_W|elf.PF_X))
				wantFinding = true
			case "executable-stack":
				order.PutUint16(body[56:], 2)
				order.PutUint32(body[120:], uint32(elf.PT_GNU_STACK))
				order.PutUint32(body[124:], uint32(elf.PF_R|elf.PF_W|elf.PF_X))
				wantFinding = true
			case "writable-executable-section":
				order.PutUint64(body[328:], uint64(elf.SHF_ALLOC|elf.SHF_WRITE|elf.SHF_EXECINSTR))
				wantFinding = true
			case "lifecycle-only":
				order.PutUint32(body[324:], uint32(elf.SHT_INIT_ARRAY))
				order.PutUint32(body[388:], uint32(elf.SHT_FINI_ARRAY))
			case "compressed-debug":
				// Layout analysis reads only this compression header, never the
				// expanded debug payload. Its file extent is 32, not 64MiB.
				order.PutUint64(body[328:], uint64(elf.SHF_COMPRESSED))
				order.PutUint64(body[352:], 32)
				order.PutUint32(body[960:], uint32(elf.COMPRESS_ZLIB))
				order.PutUint64(body[968:], 64<<20)
				order.PutUint64(body[976:], 1)
			case "packed":
				copy(body[880:], []byte("\x00UPX0\x00UPX1\x00"))
				copy(body[960:], "UPX!")
				wantFinding, wantComplete = true, false
			}
			sum := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			report, err := InspectELF(context.Background(), input.Artifact, "private/native.so", body, 0o755, digest)
			if err != nil || len(report.Executions()) != 2 || (report.Executions()[0].Status() == domain.ExecutionCompleted) != wantComplete || report.Executions()[1].Status() != domain.ExecutionCompleted || (len(report.Findings()) != 0) != wantFinding {
				t.Fatal("layout fact, review signal or packed coverage was misrepresented", err, report.Executions())
			}
			if wantFinding && report.Findings()[0].Code() != "M13_BINARY_REVIEW" {
				t.Fatal("binary facts claimed an unrelated malware verdict")
			}
			for _, fact := range report.Evidence() {
				if strings.Contains(fact.Summary(), "private/native.so") || strings.Contains(fact.Summary(), "UPX!") {
					t.Fatal("raw member name or content leaked to Evidence")
				}
			}
		})
	}
}

func FuzzBinaryLayoutBounds(f *testing.F) {
	f.Add(binaryFixture())
	f.Add([]byte("\x7fELF"))
	f.Add([]byte("not an ELF"))
	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > 1<<20 {
			return
		}
		facts, err := inspectBinaryLayout(body)
		if err == nil && (facts.Programs < 1 || facts.Programs > 128 || facts.Sections > 4096 || facts.Packed && (!facts.UPXMarker || !facts.UPXSections)) {
			t.Fatal("accepted layout escaped its declared bounds")
		}
	})
}

func TestBinaryFactsRejectUnboundUnsupportedAndExcessiveMetadata(t *testing.T) {
	input, _, _ := sourceFixture(t)
	for _, fault := range []string{"digest", "non-elf", "truncated", "program-offset", "section-count", "extended-sections", "name-size", "compressed-name-table"} {
		t.Run(fault, func(t *testing.T) {
			body := binaryFixture()
			order := binary.LittleEndian
			switch fault {
			case "non-elf":
				body[0] = 'M'
			case "truncated":
				body = body[:32]
			case "program-offset":
				order.PutUint64(body[32:], ^uint64(0))
			case "section-count":
				order.PutUint16(body[60:], 4097)
			case "extended-sections":
				order.PutUint16(body[60:], 0)
			case "name-size":
				order.PutUint64(body[480:], 1<<20+1)
			case "compressed-name-table":
				order.PutUint64(body[456:], uint64(elf.SHF_COMPRESSED))
			}
			sum := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			if fault == "digest" {
				digest, _ = domain.NewSHA256Digest(strings.Repeat("a", 64))
			}
			report, err := InspectELF(context.Background(), input.Artifact, "native.so", body, 0o755, digest)
			if err == nil || len(report.Executions()) != 2 || report.Executions()[0].Status() == domain.ExecutionCompleted || report.Executions()[1].Status() == domain.ExecutionCompleted || len(report.Findings()) != 0 {
				t.Fatal("unbound or unsupported binary became a completed clean result", err)
			}
			if fault == "non-elf" && report.Executions()[0].Capability() != domain.CapabilityUnsupported {
				t.Fatal("unsupported format capability was hidden")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sum := sha256.Sum256(binaryFixture())
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if report, err := InspectELF(ctx, input.Artifact, "native.so", binaryFixture(), 0o755, digest); !errors.Is(err, context.Canceled) || len(report.Executions()) != 0 {
		t.Fatal("cancelled binary analysis invented completed checks", err)
	}
}
