package content

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

func TestBinaryMembersPreserveSignalsAcrossPartialCoverage(t *testing.T) {
	input, _, _ := sourceFixture(t)
	risky := binaryFixture()
	binary.LittleEndian.PutUint32(risky[68:], 7)
	member := func(name string, body []byte) BinaryMember {
		sum := sha256.Sum256(body)
		digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		return BinaryMember{Path: name, Body: body, Mode: 0o755, Digest: digest}
	}
	for _, scenario := range []string{"complete", "unsupported-first", "unsupported-last", "damaged-last", "digest-last", "packed-last", "duplicate", "empty", "count-limit"} {
		t.Run(scenario, func(t *testing.T) {
			members := []BinaryMember{member("normal.so", binaryFixture()), member("risky.so", risky)}
			wantComplete, wantError, wantSignals, wantFindings := scenario == "complete", scenario != "complete" && scenario != "empty" && scenario != "packed-last", true, 1
			switch scenario {
			case "unsupported-first":
				members = append([]BinaryMember{member("private.exe", []byte("MZ"))}, members...)
			case "unsupported-last":
				members = append(members, member("private.exe", []byte("MZ")))
			case "damaged-last":
				members = append(members, member("broken.so", []byte("\x7fELF")))
			case "digest-last":
				last := member("changed.so", binaryFixture())
				last.Body[960] ^= 1
				members = append(members, last)
			case "packed-last":
				body := binaryFixture()
				copy(body[880:], []byte("\x00UPX0\x00UPX1\x00"))
				copy(body[960:], "UPX!")
				members = append(members, member("packed.so", body))
				wantFindings = 2
			case "duplicate":
				members = append(members, members[0])
			case "empty":
				members, wantSignals, wantFindings = nil, false, 0
			case "count-limit":
				members = make([]BinaryMember, ports.SourceAnalysisTargetLimit+1)
				wantSignals, wantFindings = false, 0
			}
			report, err := InspectBinaries(context.Background(), input.Artifact, members)
			checks := report.Executions()
			if (err != nil) != wantError || len(checks) != 2 || (checks[0].Status() == domain.ExecutionCompleted) != wantComplete || (checks[1].Status() == domain.ExecutionCompleted) != wantSignals || len(report.Findings()) != wantFindings {
				t.Fatal("aggregate coverage lost or overstated member facts", err, checks, len(report.Findings()))
			}
			seen := map[domain.EvidenceID]bool{}
			unsupported := false
			for _, fact := range report.Evidence() {
				if seen[fact.ID()] || fact.Digest() != input.Artifact.Digest() || strings.Contains(fact.Summary(), "private.exe") {
					t.Fatal("member evidence identity or sanitization was lost")
				}
				seen[fact.ID()] = true
				unsupported = unsupported || strings.Contains(fact.Summary(), "M13_BINARY_FORMAT_UNSUPPORTED")
			}
			if strings.HasPrefix(scenario, "unsupported-") && !unsupported {
				t.Fatal("aggregate hid member unsupported capability")
			}
			for _, finding := range report.Findings() {
				for _, id := range finding.EvidenceIDs() {
					if !seen[id] {
						t.Fatal("aggregate finding references discarded evidence")
					}
				}
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if report, err := InspectBinaries(ctx, input.Artifact, nil); !errors.Is(err, context.Canceled) || len(report.Executions()) != 0 {
		t.Fatal("cancelled analysis invented a result", err)
	}
}

func TestBinarySelectionUsesBytesAndUnsupportedFormatHints(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want bool
	}{
		{"fake.rs", "\x7fELF", true}, {"no-extension", "MZ", true},
		{"engine", "\x00asm", true}, {"native.SO.1", "", true},
		{"module.node", "", true}, {"lib.a", "!<arch>\n", true},
		{"native", "\xcf\xfa\xed\xfe", true}, {"fat", "\xca\xfe\xba\xbf", true},
		{"real.rs", "fn main() {}", false}, {"manual.txt", "Contains MZ and ELF examples", false},
	} {
		if IsBinary(test.name, []byte(test.body)) != test.want {
			t.Fatal("binary DATA was misclassified", test.name)
		}
	}
}

func TestIncrementalBinaryInspectionRetainsFactsWithoutRetainingBodies(t *testing.T) {
	input, _, _ := sourceFixture(t)
	inspection, err := NewBinaryInspection(input.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	body := binaryFixture()
	binary.LittleEndian.PutUint32(body[68:], 7)
	sum := sha256.Sum256(body)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err := inspection.Inspect(context.Background(), BinaryMember{Path: "native.so", Body: body, Digest: digest}); err != nil {
		t.Fatal(err)
	}
	clear(body) // The archive owner may release/reuse its buffer immediately.
	first, err := inspection.Report(context.Background())
	if err != nil || first.Executions()[0].Status() != domain.ExecutionCompleted || len(first.Findings()) != 1 {
		t.Fatal("incremental facts depended on retained member bytes", err)
	}
	second, err := inspection.Report(context.Background())
	if err != nil || len(second.Evidence()) != len(first.Evidence()) {
		t.Fatal("report readback accumulated duplicate summaries", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := inspection.Inspect(ctx, BinaryMember{Path: "later.so"}); !errors.Is(err, context.Canceled) {
		t.Fatal("incremental cancellation was lost", err)
	}
	report, err := inspection.Report(context.Background())
	if !errors.Is(err, context.Canceled) || report.Executions()[0].Status() == domain.ExecutionCompleted || len(report.Findings()) != 1 {
		t.Fatal("cancelled member became complete or hid earlier signals", err)
	}
}
