package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BinaryChecks declares the bounded ELF capability independently of source AST
// analysis. Cache consumers reuse this declaration; they never run the analyzer.
func BinaryChecks() ([]domain.CheckRequirement, error) {
	profile, _ := domain.CheckProfileDigest("binary-content-elf64-v1", "layout+upx;programs=128;sections=4096;section-names=1048576")
	var result []domain.CheckRequirement
	for _, name := range []string{"binary-content", "binary-content-signals"} {
		id, _ := domain.NewCheckID(name)
		check, err := domain.NewCheckRequirement(id, domain.CheckInspection, name == "binary-content", profile)
		if err != nil {
			return nil, err
		}
		result = append(result, check)
	}
	return result, nil
}

type binaryFacts struct {
	CompressedSections                                                         int
	Programs, Sections, WritableExecutablePrograms, WritableExecutableSections int
	ExecutableStack, UPXMarker, UPXSections, Packed                            bool
	InitArrays, FiniArrays                                                     int
}

var errBinaryFormatUnsupported = errors.New("binary format is outside ELF64 little-endian amd64 capability")

// inspectBinaryLayout never loads symbols, relocations or artifact code. Bound
// the ELF reader's tables and section-name allocation before constructing it.
func inspectBinaryLayout(body []byte) (binaryFacts, error) {
	var facts binaryFacts
	if len(body) < 4 || !bytes.Equal(body[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return facts, errBinaryFormatUnsupported
	}
	if len(body) < 64 {
		return facts, errors.New("binary ELF header is incomplete")
	}
	if !bytes.Equal(body[4:7], []byte{2, 1, 1}) || binary.LittleEndian.Uint16(body[18:20]) != uint16(elf.EM_X86_64) {
		return facts, errBinaryFormatUnsupported
	}
	order := binary.LittleEndian
	phoff, shoff := order.Uint64(body[32:40]), order.Uint64(body[40:48])
	phsize, phnum := order.Uint16(body[54:56]), order.Uint16(body[56:58])
	shsize, shnum, stringsIndex := order.Uint16(body[58:60]), order.Uint16(body[60:62]), order.Uint16(body[62:64])
	if order.Uint16(body[52:54]) != 64 || phnum == 0 || phnum > 128 || phsize != 56 || phoff > uint64(len(body)) || uint64(phnum)*56 > uint64(len(body))-phoff || shnum > 4096 || shnum == 0 && shoff != 0 || shnum != 0 && shsize != 64 || shoff > uint64(len(body)) || uint64(shnum)*64 > uint64(len(body))-shoff || shnum != 0 && stringsIndex >= shnum {
		return facts, errors.New("binary layout tables are unsupported or exceed bounds")
	}
	if stringsIndex != 0 {
		if stringsIndex >= shnum {
			return facts, errors.New("binary section-name table is invalid")
		}
		header := body[shoff+uint64(stringsIndex)*64 : shoff+uint64(stringsIndex+1)*64]
		offset, size, flags := order.Uint64(header[24:32]), order.Uint64(header[32:40]), order.Uint64(header[8:16])
		if size > 1<<20 || flags&uint64(elf.SHF_COMPRESSED) != 0 || offset > uint64(len(body)) || size > uint64(len(body))-offset {
			return facts, errors.New("binary section-name data is unsupported or exceeds bounds")
		}
	}
	file, err := elf.NewFile(bytes.NewReader(body))
	if err != nil {
		return facts, errors.New("binary layout cannot be decoded")
	}
	defer file.Close()
	if file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN {
		return facts, errors.New("binary layout is not an executable or shared object")
	}
	facts.Programs, facts.Sections = len(file.Progs), len(file.Sections)
	for _, program := range file.Progs {
		if program.Off > uint64(len(body)) || program.Filesz > uint64(len(body))-program.Off || program.Filesz > program.Memsz {
			return binaryFacts{}, errors.New("binary program extent is invalid")
		}
		if program.Type == elf.PT_LOAD && program.Flags&(elf.PF_W|elf.PF_X) == elf.PF_W|elf.PF_X {
			facts.WritableExecutablePrograms++
		}
		if program.Type == elf.PT_GNU_STACK && program.Flags&elf.PF_X != 0 {
			facts.ExecutableStack = true
		}
	}
	upx0, upx1 := false, false
	for _, section := range file.Sections {
		// Size is the expanded size for compressed debug sections. FileSize
		// is the checked on-disk extent; this capability never inflates them.
		if section.Type != elf.SHT_NOBITS && (section.Offset > uint64(len(body)) || section.FileSize > uint64(len(body))-section.Offset) {
			return binaryFacts{}, errors.New("binary section extent is invalid")
		}
		if section.Flags&elf.SHF_COMPRESSED != 0 {
			facts.CompressedSections++
		}
		if section.Flags&(elf.SHF_WRITE|elf.SHF_EXECINSTR) == elf.SHF_WRITE|elf.SHF_EXECINSTR {
			facts.WritableExecutableSections++
		}
		if section.Type == elf.SHT_INIT_ARRAY {
			facts.InitArrays++
		}
		if section.Type == elf.SHT_FINI_ARRAY {
			facts.FiniArrays++
		}
		upx0 = upx0 || section.Name == "UPX0"
		upx1 = upx1 || section.Name == "UPX1"
	}
	// A literal marker alone is not a malware or packing verdict: unpackers and
	// documentation may contain it. Require the corresponding layout as well.
	facts.UPXMarker = bytes.Contains(body, []byte("UPX!"))
	facts.UPXSections = upx0 && upx1
	facts.Packed = facts.UPXMarker && facts.UPXSections
	return facts, nil
}

// InspectELF consumes an already archive-validated member, never a Host path.
// It attests a limited layout/byte-marker capability, not decompilation or the
// absence of malicious behavior. The member's expected digest must match.
func InspectELF(ctx context.Context, artifact domain.AcquiredArtifact, name string, body []byte, mode uint32, expected domain.ContentDigest) (domain.InspectionReport, error) {
	if ctx == nil || artifact.Digest().String() == "" || name == "" || expected.String() == "" {
		return domain.InspectionReport{}, errors.New("binary inspection subject is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	sum, pathSum := sha256.Sum256(body), sha256.Sum256([]byte(name))
	measured := hex.EncodeToString(sum[:])
	var facts binaryFacts
	var decodeErr error
	if measured != expected.String() {
		decodeErr = errors.New("binary member digest differs from verified inventory")
	} else {
		facts, decodeErr = inspectBinaryLayout(body)
	}
	valid, complete := decodeErr == nil, decodeErr == nil && !facts.Packed
	checks, err := BinaryChecks()
	if err != nil {
		return domain.InspectionReport{}, err
	}
	status, limitation := domain.ExecutionCompleted, ""
	capability := domain.CapabilitySupported
	if !complete {
		status, limitation = domain.ExecutionIncomplete, "M13_BINARY_ANALYSIS_INCOMPLETE"
		if facts.Packed {
			limitation = "M13_BINARY_PACKED_CONTENT"
		}
	}
	if errors.Is(decodeErr, errBinaryFormatUnsupported) {
		capability, status, limitation = domain.CapabilityUnsupported, domain.ExecutionNotExecuted, "M13_BINARY_FORMAT_UNSUPPORTED"
	}
	coverage, _ := domain.NewCheckExecution(checks[0].ID(), domain.CheckInspection, true, capability, status, limitation, checks[0].ProfileDigest())
	bodySummary, err := json.Marshal(struct {
		Schema   string      `json:"schema"`
		Scope    string      `json:"scope"`
		Path     string      `json:"path_sha256"`
		Content  string      `json:"member_sha256"`
		Bytes    int         `json:"bytes"`
		Mode     uint32      `json:"archive_mode"`
		Valid    bool        `json:"layout_valid"`
		Complete bool        `json:"coverage_complete"`
		Facts    binaryFacts `json:"facts"`
	}{"binary-content-elf64-v1", "ELF64LE-amd64-layout+UPX-marker", hex.EncodeToString(pathSum[:]), measured, len(body), mode, valid, complete, facts})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	coverageID, _ := domain.NewEvidenceID("binary-content-coverage")
	evidence, err := domain.NewEvidence(coverageID, coverage.ID(), artifact.Identity(), artifact.Digest(), "binary-content-coverage", string(bodySummary), coverage.ProfileDigest())
	if err != nil {
		return domain.InspectionReport{}, err
	}
	coverageReport, err := domain.NewInspectionReport(coverage, nil, []domain.Evidence{evidence})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	signalStatus, signalLimit := domain.ExecutionCompleted, ""
	if !valid {
		signalStatus, signalLimit = domain.ExecutionNotExecuted, "M13_BINARY_SIGNALS_UNAVAILABLE"
	}
	signals, _ := domain.NewCheckExecution(checks[1].ID(), domain.CheckInspection, false, domain.CapabilitySupported, signalStatus, signalLimit, checks[1].ProfileDigest())
	signalID, _ := domain.NewEvidenceID("binary-content-signals-summary")
	signalFact, err := domain.NewEvidence(signalID, signals.ID(), artifact.Identity(), artifact.Digest(), "binary-content-signals", string(bodySummary), signals.ProfileDigest())
	if err != nil {
		return domain.InspectionReport{}, err
	}
	var findings []domain.Finding
	if valid && (facts.WritableExecutablePrograms != 0 || facts.WritableExecutableSections != 0 || facts.ExecutableStack || facts.Packed) {
		finding, err := domain.NewFinding("M13_BINARY_REVIEW", []domain.EvidenceID{signalID})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		findings = append(findings, finding)
	}
	signalReport, err := domain.NewInspectionReport(signals, findings, []domain.Evidence{signalFact})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	report, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{coverageReport, signalReport})
	if cancelled := ctx.Err(); cancelled != nil {
		return domain.InspectionReport{}, errors.Join(decodeErr, err, cancelled)
	}
	return report, errors.Join(decodeErr, err)
}
