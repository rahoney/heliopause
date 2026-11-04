package content

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

func inventoryMember(name, language string, body []byte) MemberFact {
	sum := sha256.Sum256(body)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	return MemberFact{Path: name, Language: language, Size: uint64(len(body)), Digest: digest}
}

func TestInventoryBindsCompleteSourceSelectionAndRejectsUnboundFacts(t *testing.T) {
	input, output, _ := sourceFixture(t)
	file := input.Files[0]
	member := inventoryMember(file.Path, file.Language, file.Body)
	inventory, err := NewInventory(input.Artifact, []MemberFact{member})
	if err != nil || inventory.validateSources(input.Artifact, input.Files) != nil {
		t.Fatal("measured source was not bound", err)
	}
	for _, files := range [][]ports.SourceAnalysisFile{
		nil, {file, file}, {{Path: file.Path, Language: file.Language, Body: []byte("changed")}},
		{{Path: "other.js", Language: file.Language, Body: file.Body}},
		{{Path: file.Path, Language: "python", Body: file.Body}},
	} {
		if inventory.validateSources(input.Artifact, files) == nil {
			t.Fatal("missing/duplicate/changed source DATA matched complete inventory")
		}
	}
	foreign, _ := domain.NewAcquiredArtifact(input.Artifact.Identity(), input.Artifact.Digest(), "intake:foreign:tarball", input.Artifact.SizeBytes())
	if inventory.Matches(foreign) || inventory.validateSources(foreign, input.Files) == nil {
		t.Fatal("inventory was transplanted to another intake")
	}
	unsupported := inventoryMember("helper.sh", "", []byte("#!/bin/sh\n"))
	unsupported.UnsupportedSource = true
	inventory, err = NewInventory(input.Artifact, []MemberFact{member, unsupported})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &sourceRuntime{output: output}
	report, err := InspectSources(context.Background(), runtime, input.Artifact, input.Files, inventory)
	if err != nil || runtime.calls != 1 || len(report.Findings()) != 1 || report.Executions()[0].Status() != domain.ExecutionIncomplete || report.Executions()[1].Status() != domain.ExecutionCompleted {
		t.Fatal("unsupported source became clean or erased independently valid signals", err)
	}
}

func TestZeroTargetsRequireCompleteBoundInventoryAndDoNotClaimScannerExecution(t *testing.T) {
	input, _, _ := sourceFixture(t)
	data := inventoryMember("metadata.json", "", []byte("{}"))
	inventory, err := NewInventory(input.Artifact, []MemberFact{data})
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectSources(context.Background(), nil, input.Artifact, nil, inventory)
	if err != nil || report.Executions()[0].Status() != domain.ExecutionCompleted || !strings.Contains(report.Evidence()[0].Summary(), `"no_applicable_source":true`) || !strings.Contains(report.Evidence()[0].Summary(), `"runtime_complete":false`) || !strings.Contains(report.Evidence()[0].Summary(), `"report_valid":false`) {
		t.Fatal("zero-source proof claimed a scanner run or was not bound", err)
	}
	native, _ := NewBinaryInspection(input.Artifact)
	report, err = native.Report(context.Background())
	if err != nil || report.Executions()[0].Status() != domain.ExecutionIncomplete {
		t.Fatal("empty native selection alone became complete", err)
	}
	report, err = native.Report(context.Background(), inventory)
	if err != nil || report.Executions()[0].Status() != domain.ExecutionCompleted {
		t.Fatal("zero native members did not use exact census proof", err)
	}
	data.Binary = true
	containsNative, err := NewInventory(input.Artifact, []MemberFact{data})
	if err != nil {
		t.Fatal(err)
	}
	report, err = native.Report(context.Background(), containsNative)
	if err == nil || report.Executions()[0].Status() == domain.ExecutionCompleted {
		t.Fatal("omitted native member was treated as not applicable")
	}
	if report, err := InspectSources(context.Background(), nil, input.Artifact, nil, Inventory{}); err == nil || len(report.Executions()) != 0 {
		t.Fatal("zero-value census bypassed source coverage")
	}
	data.Binary, data.UnsupportedSource = false, true
	unknownSource, err := NewInventory(input.Artifact, []MemberFact{data})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &sourceRuntime{}
	report, err = InspectSources(context.Background(), runtime, input.Artifact, nil, unknownSource)
	if err == nil || runtime.calls != 0 || report.Executions()[0].Status() != domain.ExecutionIncomplete || strings.Contains(report.Evidence()[0].Summary(), `"no_applicable_source":true`) {
		t.Fatal("unsupported source became absent source", err)
	}
}

func TestBinaryInventoryRejectsDigestSubstitutionAndKeepsSignals(t *testing.T) {
	input, _, _ := sourceFixture(t)
	body := binaryFixture()
	body[68] = 7
	member := inventoryMember("native.so", "", body)
	member.Binary = true
	native, _ := NewBinaryInspection(input.Artifact)
	if err := native.Inspect(context.Background(), BinaryMember{Path: member.Path, Body: body, Digest: member.Digest}); err != nil {
		t.Fatal(err)
	}
	inventory, err := NewInventory(input.Artifact, []MemberFact{member})
	if err != nil {
		t.Fatal(err)
	}
	report, err := native.Report(context.Background(), inventory)
	if err != nil || report.Executions()[0].Status() != domain.ExecutionCompleted || len(report.Findings()) != 1 {
		t.Fatal("bound native coverage was not retained", err)
	}
	member.Digest, _ = domain.NewSHA256Digest(strings.Repeat("b", 64))
	changed, _ := NewInventory(input.Artifact, []MemberFact{member})
	report, err = native.Report(context.Background(), changed)
	if err == nil || report.Executions()[0].Status() != domain.ExecutionIncomplete || len(report.Findings()) != 1 {
		t.Fatal("rebound inventory digest passed or hid validated risk signals", err)
	}
}

func TestInventoryRejectsMalformedOrExcessiveMemberCensus(t *testing.T) {
	input, _, _ := sourceFixture(t)
	base := inventoryMember("data", "", []byte("data"))
	for _, fault := range []string{"empty", "duplicate", "escape", "digest", "language", "overlap", "bytes", "count"} {
		t.Run(fault, func(t *testing.T) {
			member := base
			members := []MemberFact{member}
			switch fault {
			case "empty":
				members = nil
			case "duplicate":
				members = append(members, member)
			case "escape":
				members[0].Path = "../outside"
			case "digest":
				members[0].Digest = domain.ContentDigest{}
			case "language":
				members[0].Language = "unknown-parser"
			case "overlap":
				members[0].Binary, members[0].Language = true, "javascript"
			case "bytes":
				members[0].Size = ports.SourceAnalysisTotalLimit + 1
			case "count":
				members = make([]MemberFact, ports.SourceAnalysisTargetLimit+1)
			}
			if inventory, err := NewInventory(input.Artifact, members); err == nil || inventory.Matches(input.Artifact) {
				t.Fatal("invalid member census produced an applicability proof")
			}
		})
	}
}
